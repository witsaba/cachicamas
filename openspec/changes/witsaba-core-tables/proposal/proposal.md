# Proposal: witsaba-core-tables

> **Change**: `witsaba-core-tables`
> **Status**: proposed
> **Date**: 2026-06-22
> **Driver**: braejan
> **Project**: cachicamas (witsaba framework)
> **Artifact store**: openspec (filesystem, committable)

## Why

The witsaba framework needs a stable, append-mostly relational core to model
the lifecycle of a product requirement — from a Markdown PRD, through a
spike investigation, to a milestone, a set of tasks, specs, and a per-spec
phase timeline that an AI agent (and a human) can both query and reason
over. The existing migration system (`backend/database_administrator/src/migration/`,
`pressly/goose` v3.27.1 + `jackc/pgx/v5`) is production-grade and ready to
accept new migration files; this change is purely additive schema work.
The schema must be **agent-first** (an AI agent is the primary operator of
the workflow), **append-mostly by default** (history is the source of
truth, in-place mutation is the exception), and **lower-footprint**
(each table gets only the columns the user explicitly asked for, plus
the framework-wide timestamp convention).

## What changes

This change introduces **8 new tables** in `public` (the runner's default
schema) across **3 new goose migration files**, plus **6 new integration
tests** that exercise the schema against the live compose-postgres
instance. The tables form a strict dependency chain rooted at
`organization` and terminating at `spec_phase`. Cardinality is locked per
the user's verbatim answers (see "Conventions locked" below): `milestone`
is strict-inherited from `requirement` (PK = FK, 1:1); every other
parent-child link is 1:N with a synthetic PK on the child. The 256 KiB PRD
cap is enforced by a CHECK constraint AND documented in a column
comment. `spec_phase` is an append-only history child of `spec` with
synthetic PK + UNIQUE `(spec_id, phase, started_at)` so an agent can
re-enter earlier phases (e.g., `tdd_red` after `technical_ai_review` finds
a gap) without losing the audit trail.

## What does NOT change

- **No changes to the runner, the embed pattern, the composition root, the
  driver, the domain port, or any non-SQL code in
  `backend/database_administrator/src/`.** The migration system is
  consumed only as a tool; this change writes files under
  `sql/` and adds test cases to `runner_test.go`.
- **No changes to `docker-compose.yaml`, `infra/`, or the OTel collector
  config.** The new migrations run on the same compose stack.
- **No application-level data access code.** This change ships DDL +
  tests only. A future change (e.g., `witsaba-requirements-api`) will
  add the data access layer; it will read the column comments and
  CHECK constraints written here.
- **No DB-level append-only enforcement** (REVOKE UPDATE, BEFORE UPDATE
  trigger, or history-sidecar tables). App-layer discipline is the v1
  defense, documented via column comments; DB-level enforcement is a
  follow-up change.
- **No new top-level Go dependencies.** This change is pure DDL + Go
  test code (no new imports).
- **No data backfill.** All 8 tables are new; no existing tables are
  altered.

## The schema — 8 tables, full DDL

All tables live in the `public` schema (the runner's default). Every
table has `created_at TIMESTAMPTZ NOT NULL DEFAULT now()` and
`updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`; on insert both equal
`now()` (UTC, server-enforced via `infra/postgres/init/01-init.sql`).
All FK columns are explicitly named (`organization_id`, `project_id`,
etc.), never `parent_id` or generic.

### 1. `organization` (root)

```sql
CREATE TABLE organization (
    id              BIGSERIAL    PRIMARY KEY,
    shortname       TEXT,                                -- nullable
    full_name       TEXT         NOT NULL,
    identification  TEXT         NOT NULL UNIQUE,        -- country tax ID / RFC / NIT
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,  -- UPDATE-in-place exception
    email           TEXT,                                -- verification only
    phone           TEXT,                                -- verification only
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON COLUMN organization.is_active IS
    'UPDATE-in-place: this column is the ONLY mutation allowed on organization rows.';
```

**Cardinality**: root (no parents). **Rationale**: `identification` is
the natural business key (per country: RFC, NIT, EIN, etc.) so it
carries `UNIQUE`. `is_active` is the only UPDATE-in-place column in the
whole schema; the comment makes that exception explicit for code
reviewers.

### 2. `project` (synthetic PK + FK to organization, 1:N)

```sql
CREATE TABLE project (
    id              BIGSERIAL    PRIMARY KEY,
    organization_id BIGINT       NOT NULL REFERENCES organization(id),
    key             TEXT         NOT NULL UNIQUE,        -- e.g., 'cachicamas'
    full_name       TEXT         NOT NULL,
    start_date      DATE,                                -- nullable
    end_date        DATE,                                -- nullable
    metadata        JSONB,                               -- nullable, ONLY place JSONB is allowed
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_project_organization_id ON project(organization_id);
```

**Cardinality**: one organization → many projects. **Rationale**:
`metadata JSONB` is the only place JSONB is allowed in the entire
schema (per the user's explicit request). `key` is the natural
identifier (e.g., `cachicamas`) and is UNIQUE globally, not just
within an organization, because the witsaba CLI uses it as a stable
handle.

### 3. `requirement` (synthetic PK + FK to project, 1:N)

```sql
CREATE TABLE requirement (
    id                     BIGSERIAL    PRIMARY KEY,
    project_id             BIGINT       NOT NULL REFERENCES project(id),
    filename               TEXT         NOT NULL,
    content                TEXT         NOT NULL,
    git_repository_url     TEXT,                          -- nullable
    analysis_result        TEXT,                          -- nullable
    is_technically_viable  BOOLEAN,                       -- nullable
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT requirement_content_size_cap CHECK (octet_length(content) <= 262144)
);
COMMENT ON COLUMN requirement.content IS
    'Max PRD size: 256 KiB (262144 bytes), enforced via CHECK constraint.';

CREATE INDEX idx_requirement_project_id ON requirement(project_id);
```

**Cardinality**: one project → many requirements. **Rationale**:
256 KiB cap is enforced as a CHECK constraint AND documented via a
column comment (belt + suspenders, per locked decision Q2). The
`is_technically_viable` flag is nullable: an unanalyzed requirement
has not yet been judged.

### 4. `requirement_spike` (synthetic PK + FK to requirement, 1:N)

```sql
CREATE TABLE requirement_spike (
    id              BIGSERIAL    PRIMARY KEY,
    requirement_id  BIGINT       NOT NULL REFERENCES requirement(id),
    started_at      DATE,                                -- nullable
    ended_at        DATE,                                -- nullable
    outcome         TEXT,                                -- nullable
    findings        TEXT,                                -- nullable, markdown
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT requirement_spike_dates_valid
        CHECK (ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX idx_requirement_spike_requirement_id ON requirement_spike(requirement_id);
```

**Cardinality**: one requirement → many spikes (per locked decision
A3, breaks the inheritance convention in exchange for multi-spike
support). **Rationale**: a single synthetic PK lets a requirement
accumulate spikes over time as it is re-investigated. The
`requirement_spike_dates_valid` CHECK prevents the agent from
recording nonsensical date order.

### 5. `milestone` (STRICT INHERITANCE — PK = FK = `requirement_id`, 1:1)

```sql
CREATE TABLE milestone (
    requirement_id  BIGINT       PRIMARY KEY REFERENCES requirement(id),  -- PK = FK
    title           TEXT         NOT NULL,
    description     TEXT,                                -- nullable, markdown
    start_date      DATE,                                -- nullable
    end_date        DATE,                                -- nullable,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT milestone_dates_valid
        CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);
```

**Cardinality**: one requirement → exactly one milestone (per locked
decision A1). **Rationale**: the `requirement_id` column is BOTH the
PK and the FK. Postgres enforces the 1:1 invariant by refusing a
second milestone for the same `requirement_id`. There is no
synthetic `id` column; that is the design. The `milestone_dates_valid`
CHECK prevents nonsensical date order.

### 6. `task` (synthetic PK + FK to milestone, 1:N)

```sql
CREATE TABLE task (
    id              BIGSERIAL    PRIMARY KEY,
    milestone_id    BIGINT       NOT NULL REFERENCES milestone(id),
    title           TEXT         NOT NULL,
    description     TEXT,                                -- nullable, markdown
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_task_milestone_id ON task(milestone_id);
```

**Cardinality**: one milestone → many tasks (per locked decision A4,
breaks the inheritance convention so a milestone can hold many
tasks). **Rationale**: tasks are units of work; a single milestone
typically bundles several. A synthetic PK on `task` is required
because the parent is now in a 1:N relationship, not 1:1.

### 7. `spec` (synthetic PK + FK to task, 1:N)

```sql
CREATE TABLE spec (
    id              BIGSERIAL    PRIMARY KEY,
    task_id         BIGINT       NOT NULL REFERENCES task(id),
    content         TEXT         NOT NULL,               -- markdown, the spec body
    start_date      DATE,                                -- nullable
    end_date        DATE,                                -- nullable
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT spec_dates_valid
        CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);

CREATE INDEX idx_spec_task_id ON spec(task_id);
```

**Cardinality**: one task → many specs (per locked decision A4, the
many side of the `task ↔ spec` relationship). **Rationale**: a task
can have many specs over its lifetime (initial spec, revised spec
after an AI review, etc.). Each spec is independently current until
its successor is written; history is captured by the row itself plus
its `spec_phase` children.

### 8. `spec_phase` (synthetic PK + FK to spec, history/append-only)

```sql
CREATE TABLE spec_phase (
    id              BIGSERIAL    PRIMARY KEY,
    spec_id         BIGINT       NOT NULL REFERENCES spec(id),
    phase           TEXT         NOT NULL,
    started_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    ended_at        TIMESTAMPTZ,                         -- nullable, set on phase exit
    notes           TEXT,                                -- agent's reasoning, esp. for re-entries
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT spec_phase_phase_check CHECK (phase IN (
        'tdd_red','implementation','tdd_green','verify',
        'pr','technical_ai_review','ai_approved','human_approved'
    )),
    CONSTRAINT spec_phase_natural_key UNIQUE (spec_id, phase, started_at),
    CONSTRAINT spec_phase_dates_valid
        CHECK (ended_at IS NULL OR ended_at >= started_at)
);

CREATE INDEX idx_spec_phase_spec_id        ON spec_phase(spec_id);
CREATE INDEX idx_spec_phase_current_state  ON spec_phase(spec_id) WHERE ended_at IS NULL;
```

**Cardinality**: one spec → many phase events (append-only history).
**Rationale**: per locked decisions Q3 and Q4, this table is
**agent-first**. The synthetic `id` enables future child tables
(e.g., `spec_phase_artifact` for per-phase AI feedback) to FK with a
single column. The `UNIQUE (spec_id, phase, started_at)` enforces
the natural-key invariant: the same phase cannot be entered twice
at the exact same instant for the same spec. The
`idx_spec_phase_current_state` partial index makes the "what phase is
this spec in right now?" query a single B-tree seek. The
`spec_phase_dates_valid` CHECK prevents the agent from setting
`ended_at` to a time before `started_at`.

## Migration file layout — 3 files, dependency order

Files land under `backend/database_administrator/src/migration/sql/`
using the existing 14-digit timestamp + snake_case convention
(README §1). Timestamps are spaced 1 second apart from the existing
hello-world (20260621120000) for in-PR reviewability of the new
files.

```
20260622120000_orgs_and_projects.sql            -- organization, project
20260622120001_requirements_and_milestones.sql  -- requirement, requirement_spike, milestone
20260622120002_tasks_and_specs.sql              -- task, spec, spec_phase
```

Each file is a SINGLE SQL block with `-- +goose Up` and `-- +goose
Down` (per README §2). The `-- +goose StatementBegin` /
`-- +goose StatementEnd` wrappers wrap each `CREATE TABLE` so each
table can be its own transaction. The Down blocks reverse in
dependency order: DROP `spec_phase`, DROP `spec`, DROP `task`, DROP
`milestone`, DROP `requirement_spike`, DROP `requirement`, DROP
`project`, DROP `organization`.

### 20260622120000_orgs_and_projects.sql (sketch)

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE organization (
    id              BIGSERIAL    PRIMARY KEY,
    shortname       TEXT,
    full_name       TEXT         NOT NULL,
    identification  TEXT         NOT NULL UNIQUE,
    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
    email           TEXT,
    phone           TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
COMMENT ON COLUMN organization.is_active IS
    'UPDATE-in-place: this column is the ONLY mutation allowed on organization rows.';

CREATE TABLE project (
    id              BIGSERIAL    PRIMARY KEY,
    organization_id BIGINT       NOT NULL REFERENCES organization(id),
    key             TEXT         NOT NULL UNIQUE,
    full_name       TEXT         NOT NULL,
    start_date      DATE,
    end_date        DATE,
    metadata        JSONB,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_project_organization_id ON project(organization_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE project;
DROP TABLE organization;
-- +goose StatementEnd
```

### 20260622120001_requirements_and_milestones.sql (sketch)

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE requirement (
    id                     BIGSERIAL    PRIMARY KEY,
    project_id             BIGINT       NOT NULL REFERENCES project(id),
    filename               TEXT         NOT NULL,
    content                TEXT         NOT NULL,
    git_repository_url     TEXT,
    analysis_result        TEXT,
    is_technically_viable  BOOLEAN,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT requirement_content_size_cap CHECK (octet_length(content) <= 262144)
);
COMMENT ON COLUMN requirement.content IS
    'Max PRD size: 256 KiB (262144 bytes), enforced via CHECK constraint.';
CREATE INDEX idx_requirement_project_id ON requirement(project_id);

CREATE TABLE requirement_spike (
    id              BIGSERIAL    PRIMARY KEY,
    requirement_id  BIGINT       NOT NULL REFERENCES requirement(id),
    started_at      DATE,
    ended_at        DATE,
    outcome         TEXT,
    findings        TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT requirement_spike_dates_valid
        CHECK (ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at)
);
CREATE INDEX idx_requirement_spike_requirement_id ON requirement_spike(requirement_id);

CREATE TABLE milestone (
    requirement_id  BIGINT       PRIMARY KEY REFERENCES requirement(id),
    title           TEXT         NOT NULL,
    description     TEXT,
    start_date      DATE,
    end_date        DATE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT milestone_dates_valid
        CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE milestone;
DROP TABLE requirement_spike;
DROP TABLE requirement;
-- +goose StatementEnd
```

### 20260622120002_tasks_and_specs.sql (sketch)

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE task (
    id              BIGSERIAL    PRIMARY KEY,
    milestone_id    BIGINT       NOT NULL REFERENCES milestone(id),
    title           TEXT         NOT NULL,
    description     TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_task_milestone_id ON task(milestone_id);

CREATE TABLE spec (
    id              BIGSERIAL    PRIMARY KEY,
    task_id         BIGINT       NOT NULL REFERENCES task(id),
    content         TEXT         NOT NULL,
    start_date      DATE,
    end_date        DATE,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT spec_dates_valid
        CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)
);
CREATE INDEX idx_spec_task_id ON spec(task_id);

CREATE TABLE spec_phase (
    id              BIGSERIAL    PRIMARY KEY,
    spec_id         BIGINT       NOT NULL REFERENCES spec(id),
    phase           TEXT         NOT NULL,
    started_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    ended_at        TIMESTAMPTZ,
    notes           TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT spec_phase_phase_check CHECK (phase IN (
        'tdd_red','implementation','tdd_green','verify',
        'pr','technical_ai_review','ai_approved','human_approved'
    )),
    CONSTRAINT spec_phase_natural_key UNIQUE (spec_id, phase, started_at),
    CONSTRAINT spec_phase_dates_valid
        CHECK (ended_at IS NULL OR ended_at >= started_at)
);
CREATE INDEX idx_spec_phase_spec_id       ON spec_phase(spec_id);
CREATE INDEX idx_spec_phase_current_state ON spec_phase(spec_id) WHERE ended_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE spec_phase;
DROP TABLE spec;
DROP TABLE task;
-- +goose StatementEnd
```

## Agent-first re-entry pattern (canonical SQL)

The agent's transition algorithm on a phase event. The full version
is recorded in Engram observation #1648 ("Witsaba framework —
conventions and domain tree (r3)"); the minimum canonical SQL is:

```sql
-- 1. Close the current open phase (if any)
UPDATE spec_phase
SET ended_at = now(), updated_at = now()
WHERE spec_id = :spec_id AND ended_at IS NULL;

-- 2. Enter the new phase (any of the 8, even an earlier one)
INSERT INTO spec_phase (spec_id, phase, notes)
VALUES (:spec_id, 'tdd_red',
        'AI review at <PR url> found missing test cases for X; returning to TDD red.');

-- 3. Read the spec's current state (single-row read via the partial index)
SELECT id, phase, started_at, notes
FROM spec_phase
WHERE spec_id = :spec_id AND ended_at IS NULL;

-- 4. Read the spec's full history (audit trail)
SELECT phase, started_at, ended_at, notes
FROM spec_phase
WHERE spec_id = :spec_id
ORDER BY started_at;
```

The `notes` column is the agent's reasoning for each transition. It
is the single most important agent-first feature in the schema: it
makes transitions legible to humans AND to the agent on subsequent
reasoning passes.

## Conventions locked

User's verbatim decisions (from explore Q1–Q4 + A1–A5 follow-ups; full
trail in Engram #1648 and #1649). These are NOT revisit-able in
sdd-spec or sdd-design.

- **A1**: `milestone` is strict-inherited from `requirement`. PK = FK
  = `requirement_id`. One requirement → exactly one milestone.
- **A2**: `spec_phase` is a history child table (append-only). No
  `current_phase` column on `spec`; the current state is the row in
  `spec_phase` with `ended_at IS NULL`.
- **A3**: One requirement → MANY spikes. `requirement_spike` gets
  synthetic PK + non-PK FK to `requirement`.
- **A4**: One milestone → MANY tasks. One task → MANY specs. Both
  `task` and `spec` get synthetic PK + non-PK FK.
- **A5**: `organization.is_active` is a plain `BOOLEAN NOT NULL
  DEFAULT TRUE`. No `deactivated_at` column.
- **Q1**: Phase enumeration = **8 values exactly**: `tdd_red`,
  `implementation`, `tdd_green`, `verify`, `pr`,
  `technical_ai_review`, `ai_approved`, `human_approved`.
- **Q2**: 256 KiB PRD cap enforced via `CHECK (octet_length(content)
  <= 262144)` AND documented via `COMMENT ON COLUMN requirement.content
  IS 'Max PRD size: 256 KiB (enforced).'`.
- **Q3 (agent first)**: the framework MUST be agent first. The
  agent must be able to re-enter earlier phases when a later phase
  fails. `spec_phase` rows record each phase-entry event with a
  `notes` column for the agent's reasoning.
- **Q4**: `spec_phase` PK is `id BIGSERIAL PRIMARY KEY` (synthetic),
  with `UNIQUE (spec_id, phase, started_at)` to enforce the
  natural-key invariant. Child tables can FK cleanly to
  `spec_phase(id)`.

**Cross-cutting conventions** (also locked):

- All tables have `created_at TIMESTAMPTZ NOT NULL DEFAULT now()` and
  `updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()`. Both equal
  `now()` on insert. UTC at the server (`infra/postgres/init/01-init.sql`
  sets `timezone='UTC'`).
- Incremental integer PKs (`BIGSERIAL`), not UUIDs, for v1.
- `metadata JSONB` ONLY on `project`. No other table gets it.
- Append-mostly by default. UPDATE-in-place is allowed ONLY on
  `organization.is_active`. App-layer discipline is the primary
  defense; DB-level REVOKE/trigger is a follow-up.
- All FK columns are explicitly named (`organization_id`, `project_id`,
  `requirement_id`, `milestone_id`, `task_id`, `spec_id`),
  never `parent_id` or generic.
- CHECK constraints on enum-like columns (phase values) and on
  date-pair validity (`ended_at >= started_at` where both
  applicable).

## Test surface — 6 new integration tests in `runner_test.go`

All gated on `INTEGRATION=1` (same gating as the existing
`TestRunner_Up_*` cases). Each test seeds the bookkeeping table
cleanly via the existing `resetSchemaMigrations` helper, runs the
runner to apply all 4 migrations (hello + 3 new), and asserts
schema-level invariants.

1. **`TestRunner_Up_AllNewMigrationsApply`** — all 3 new migrations
   apply cleanly; `public.schema_migrations` has 4 rows
   (hello + 3 new) in chronological order; `\d`-style introspection
   of each new table confirms expected columns exist.
2. **`TestRunner_Up_FKConstraintsApply`** — `INSERT INTO project
   (organization_id, ...) VALUES (99999, ...)` fails with FK
   violation (proves `REFERENCES organization(id)` is enforced).
3. **`TestRunner_Up_StrictInheritanceOnMilestone`** —
   `milestone.requirement_id` is the only PK column (no synthetic
   `id`); the `pg_index` row for `milestone`'s primary key lists
   only `requirement_id`. Inserting two milestones for the same
   `requirement_id` fails.
4. **`TestRunner_Up_AppendOnlyConvention`** — column comments on
   `organization.is_active` and on append-only tables declare the
   convention (asserted via `col_description(...)`).
5. **`TestRunner_Up_PRDSizeCapEnforced`** — `INSERT INTO requirement
   (project_id, filename, content) VALUES (1, 'big.md',
   repeat('x', 262145))` fails the CHECK constraint.
6. **`TestRunner_Up_SpecPhaseReEntry`** — agent pattern: open a
   phase, UPDATE its `ended_at`, INSERT a new `spec_phase` row with
   `phase = 'tdd_red'` after a `technical_ai_review` row. Both
   succeed; the `UNIQUE (spec_id, phase, started_at)` constraint
   still holds (the two rows have different `started_at`); the
   `idx_spec_phase_current_state` partial index returns exactly
   one row for the spec.

Plus the existing `TestRunner_Up_LexicographicOrder` is extended
to assert the 3 new files sort in the correct order.

## PR size forecast

| Component                          | Estimated lines    |
| ---------------------------------- | ------------------ |
| 3 new SQL migration files          | 150-220 total      |
| 6 new integration tests            | 250-300            |
| Schema DDL comments + indexes      | 30-50 (in the SQL) |
| **Total new code in `src/`**       | **~400-500 lines** |

**Recommendation**: single PR. The 8 tables are tightly coupled by
FKs; splitting them across chained PRs risks breaking the dependency
chain (file 2 references project.id; file 3 references
requirement.id, milestone.id, task.id) and forces a "scratch DB +
rebuild" step in every chained-PR branch. The 3-file layout IS
structured so that chained PRs ARE possible (one PR per file, with
each PR's migration referencing only the prior file's tables), but
the PR's reviewer load stays low either way (a 400-line diff is the
default review budget per `sdd-tasks`/`sdd-apply` workflow).

**If the user wants chained PRs**: the layout supports 3 chained PRs
in dependency order (orgs+projects → requirements+milestones →
tasks+specs), each independently mergeable against an empty DB and
each ending with a clean `make test/integration` green. Surface
this as a choice to the user during sdd-tasks / sdd-apply.

## Risks

1. **256 KiB PRD cap is hard** — a future requirement that needs more
   must ALTER TABLE (drop the CHECK, re-add with a higher cap).
   Documented in the column comment so contributors see the limit
   before they hit it.
2. **`spec_phase` re-entry relies on app discipline** — a bug in the
   app's "close current phase before opening next" code creates
   overlapping rows (two open phases for the same spec). The
   `idx_spec_phase_current_state` partial index does NOT enforce
   uniqueness on `(spec_id) WHERE ended_at IS NULL`; that is a
   known gap. Mitigation: add a follow-up migration to add a
   partial UNIQUE index, OR rely on the existing `notes` audit
   trail to detect and resolve overlaps in human/agent review.
3. **`MIGRATION_TIMEOUT` (30s) may be tight for a first-time apply
   of all 3 new files against a populated DB.** Three CREATE TABLE
   blocks with no backfill is fast (<1s typical), but the
   `application/migration_service` adds per-call OTel overhead.
   Measure during the GREEN step; bump `MIGRATION_TIMEOUT` if the
   apply exceeds 25s in a clean test run.
4. **OTel upstream bug at v1.44.0 drops ~9/10 spans.** The migration
   span for these 3 new files will work (spans are created), but may
   not reach Jaeger. The `trace_id` is recoverable from
   `docker compose logs otel-collector` per the existing README §5
   guidance. Not a blocker; documented in README §5.
5. **Strict inheritance on `milestone` makes the cardinality
   fixed at 1:1 forever.** A future requirement that genuinely
   needs multiple milestones (a "deliverable per release channel"
   pattern) cannot add a second `milestone` row for the same
   `requirement_id`. Workaround: a future `project_milestone` table
   (project-scoped) is the escape hatch; the strict inheritance
   here stays intact for `requirement → milestone`.

## Alternatives considered

**Migration file layout — Option B (3 files) chosen over A and C**.

- **Option A (one file per table, 7 files)**: too granular; 7
  small files make the dependency order less obvious and the
  rollback story harder to follow. Rejected.
- **Option C (one file, 8 tables, ~250 lines)**: fine for atomicity
  but a 250-line migration file is hard to review in a PR. Goose
  handles it fine; humans do not. Rejected.

**PK design — synthetic BIGSERIAL chosen over composite natural**.

- Composite natural (e.g., `requirement_id` on `milestone`, and
  `(spec_id, started_at)` on `spec_phase`) is a smaller footprint
  on paper but blocks future child tables. A `spec_phase_artifact`
  table (for per-phase AI feedback) would need to FK to a
  composite key, doubling the FK columns and the index size. The
  8-byte cost per row of a synthetic PK is negligible at the
  framework's expected scale.

**Append-only enforcement — app-layer only (v1)**.

- REVOKE UPDATE on append-only tables (from `queen`) is a clean
  defense but makes the framework harder to evolve (a later
  decision to allow in-place edits on `is_active` requires a
  GRANT). Defer to a follow-up change when the schema stabilizes.
- BEFORE UPDATE triggers add per-row overhead. Defer.

## Success criteria

- [ ] All 3 new migration files apply cleanly via
      `INTEGRATION=1 make test` (which runs `go test -race -v ./...`
      against the compose stack).
- [ ] `public.schema_migrations` has 4 rows after first apply
      (1 hello + 3 new), in chronological order, with `is_applied = TRUE`.
- [ ] All 6 new integration tests pass.
- [ ] `TestRunner_Up_LexicographicOrder` extended test still passes.
- [ ] No existing unit or integration test regresses
      (`TestRunner_Up_FirstBoot`, `TestRunner_Up_SecondBootIsNoOp`,
      `TestRunner_Up_AdvisoryLockBlocksParallelRun`,
      `TestNewGooseRunner_NilSafeConstruct`,
      `TestRunner_Status_UpstreamErrorPropagates`).
- [ ] `make lint` is clean (no new vet/staticcheck/revive issues).
- [ ] `make build` produces a `./bin/database_administrator` binary
      that embeds the 3 new SQL files (verifiable by `strings` or by
      a second-boot Up call).
- [ ] The 3 `-- +goose Down` blocks reverse cleanly: rolling back
      to version 0 leaves no residual tables in `public`.

## Next recommended phase

`sdd-spec` (and `sdd-design` in parallel — they have no dependency
on each other once the proposal is locked). `sdd-tasks` follows
both.

## Skill resolution

- `paths-injected`: go-testing, work-unit-commits, chained-pr,
  branch-pr, cognitive-doc-design (all loaded from
  `/Users/braejan/.claude/skills/`).
- `fallback-path`: test-driven-development (literal SKILL.md does
  not exist on disk; TDD discipline applied from
  `openspec/AGENTS.md`). Drift noted in Engram session #1634.

## Review checklist

- [ ] reviewer can confirm 8 tables exist with the columns and constraints listed in §"The schema"
- [ ] reviewer can confirm `milestone` uses strict inheritance (PK = FK = `requirement_id`, no synthetic `id`)
- [ ] reviewer can confirm 256 KiB cap is enforced via `CHECK (octet_length(content) <= 262144)` AND documented via `COMMENT ON COLUMN requirement.content`
- [ ] reviewer can confirm `spec_phase` PK is `id BIGSERIAL` (synthetic), with `UNIQUE (spec_id, phase, started_at)`
- [ ] reviewer can confirm the 8 phase values in the CHECK constraint match the locked list (no more, no less)
- [ ] reviewer can confirm migration file layout is 3 files in dependency order with timestamps `20260622120000`, `20260622120001`, `20260622120002`
- [ ] reviewer can confirm Down blocks reverse in dependency order (last-created first-dropped)
- [ ] reviewer can confirm each table has `created_at` and `updated_at` with `NOT NULL DEFAULT now()`
- [ ] reviewer can confirm `metadata JSONB` appears only on `project`, nowhere else
- [ ] reviewer can confirm 6 new integration tests are listed and each tests a distinct schema invariant
- [ ] reviewer can confirm PR size forecast is within the 400-line default review budget
- [ ] reviewer can confirm no existing runner.go, embed.go, driver.go, or composition-root code is modified
- [ ] reviewer can confirm no new top-level Go dependencies are introduced
- [ ] reviewer can confirm the agent-first re-entry pattern (UPDATE current row's `ended_at`, INSERT new row) is the canonical SQL
- [ ] reviewer can confirm `notes` is on `spec_phase` (not on every append-only table) per the lower-footprint principle
