# Design: witsaba-core-tables

> Technical design for the `witsaba-core-tables` change. `sdd-apply` MUST follow this document.
>
> The 8-table schema, the 3-file migration layout, the cardinality decisions, and the user's locked choices (A1-A5 + Q1-Q4) are in `proposal/proposal.md`. This document is the HOW: the exact file paths, the exact DDL structure, the exact test architecture, and the exact error-handling + observability behavior.

## 1. File layout

**No new Go files. No edits to existing Go files.** This change is pure DDL + integration tests.

```
backend/database_administrator/src/migration/sql/
  20260621120000_hello_world.sql                                  (existing; unchanged)
  20260622120000_orgs_and_projects.sql                            (NEW)
  20260622120001_requirements_and_milestones.sql                  (NEW)
  20260622120002_tasks_and_specs.sql                              (NEW)

backend/database_administrator/src/migration/runner_test.go       (EDIT — add 6 tests, extend 1)
```

Rationale for the layout:

- The runner is generic. `runner.go:91-135` calls `provider.Up(ctx)` with the embed.FS rooted at `sql`; any new file picked up at build time is applied automatically. No runner changes needed.
- The tests live in the same package (`package migration`) and the same file (`runner_test.go`) because they exercise the existing `GooseRunner` constructor (`runner_test.go:113-118`) and the existing `INTEGRATION=1` gate (`runner_test.go:42-71`). Reusing the file keeps the test scaffolding (helper `integrationRunnerDB`, helper `resetSchemaMigrations`, helper `discardLogger`) DRY.
- The 6 new tests reuse `resetSchemaMigrations` and `newTestRunner` exactly as the existing 4 integration tests do. No new helpers, no new files.

## 2. Migration file internals

Each file follows the goose v3 single-file idiom documented at `migration/README.md:51-88`:

- One `.sql` file with `-- +goose Up` and `-- +goose Down` blocks.
- The Up block is wrapped in `-- +goose StatementBegin` / `-- +goose StatementEnd` so every `CREATE TABLE` is its own transaction (matches the `20260621120000_hello_world.sql` pattern at `sql/20260621120000_hello_world.sql:1-23`).
- The Down block is wrapped the same way; tables are dropped in reverse dependency order.
- Timestamps follow the regex `^\d{14}_[a-z0-9_]+\.sql$` from `migration/README.md:34`. Timestamps are spaced 1 second apart from the hello-world (20260621120000) for in-PR reviewability per the proposal §"Migration file layout".

### 2.1 `20260622120000_orgs_and_projects.sql`

**Tables (in order):** `organization`, then `project`.

`project` MUST come AFTER `organization` because `project.organization_id` REFERENCES `organization(id)`.

| Block | Item | Source line (from proposal) |
|---|---|---|
| `organization` columns | `id BIGSERIAL PK`, `shortname TEXT NULL`, `full_name TEXT NOT NULL`, `identification TEXT NOT NULL UNIQUE`, `is_active BOOLEAN NOT NULL DEFAULT TRUE`, `email TEXT NULL`, `phone TEXT NULL`, `created_at`, `updated_at` | proposal §1 |
| `COMMENT ON COLUMN organization.is_active` | `'UPDATE-in-place: this column is the ONLY mutation allowed on organization rows.'` | proposal §1 |
| `project` columns | `id BIGSERIAL PK`, `organization_id BIGINT NOT NULL REFERENCES organization(id)`, `key TEXT NOT NULL UNIQUE`, `full_name TEXT NOT NULL`, `start_date DATE NULL`, `end_date DATE NULL`, `metadata JSONB NULL`, `created_at`, `updated_at` | proposal §2 |
| Index | `idx_project_organization_id ON project(organization_id)` | proposal §2 |

CHECK constraints: **none** in this file. The only "guarded" column is `organization.identification` (UNIQUE) and `project.key` (UNIQUE); both are covered by the implicit `UNIQUE` index Postgres creates.

Down block: `DROP TABLE project;` then `DROP TABLE organization;` in dependency order (children before parents). No `CASCADE` — we want the FK violation if the down is somehow run against a partially-populated DB; that is a louder failure than silent cascade.

### 2.2 `20260622120001_requirements_and_milestones.sql`

**Tables (in order):** `requirement`, then `requirement_spike`, then `milestone`.

Why this order:

1. `requirement` first because `requirement_spike.requirement_id` and `milestone.requirement_id` both FK to it.
2. `requirement_spike` before `milestone` because `milestone` strict-inherits from `requirement` and the proposal fixes the order as "requirements, then children" — the running order of the two children is independent (no FK between them), but `requirement_spike` is 1:N and `milestone` is 1:1, so we order 1:N first to make the dependency on the synthetic PK obvious to a reader.

| Block | Item | Source line (from proposal) |
|---|---|---|
| `requirement` columns | `id BIGSERIAL PK`, `project_id BIGINT NOT NULL REFERENCES project(id)`, `filename TEXT NOT NULL`, `content TEXT NOT NULL`, `git_repository_url TEXT NULL`, `analysis_result TEXT NULL`, `is_technically_viable BOOLEAN NULL`, `created_at`, `updated_at` | proposal §3 |
| CHECK | `requirement_content_size_cap CHECK (octet_length(content) <= 262144)` | proposal §3, locked decision Q2 |
| `COMMENT ON COLUMN requirement.content` | `'Max PRD size: 256 KiB (262144 bytes), enforced via CHECK constraint.'` | proposal §3, locked decision Q2 |
| Index | `idx_requirement_project_id ON requirement(project_id)` | proposal §3 |
| `requirement_spike` columns | `id BIGSERIAL PK`, `requirement_id BIGINT NOT NULL REFERENCES requirement(id)`, `started_at DATE NULL`, `ended_at DATE NULL`, `outcome TEXT NULL`, `findings TEXT NULL` (markdown), `created_at`, `updated_at` | proposal §4 |
| CHECK | `requirement_spike_dates_valid CHECK (ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at)` | proposal §4 |
| Index | `idx_requirement_spike_requirement_id ON requirement_spike(requirement_id)` | proposal §4 |
| `milestone` columns | `requirement_id BIGINT PRIMARY KEY REFERENCES requirement(id)` (PK = FK), `title TEXT NOT NULL`, `description TEXT NULL`, `start_date DATE NULL`, `end_date DATE NULL`, `created_at`, `updated_at` | proposal §5 |
| CHECK | `milestone_dates_valid CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)` | proposal §5 |

The proposal's CHECK on `milestone.end_date` / `start_date` (Q-and-A in proposal §5) is **kept** as a defensive guard even though the user did not strictly require it; it is consistent with the same pattern on `requirement_spike` and `spec`. One constraint name across all three date-pair tables keeps the review grep-friendly.

Down block: `DROP TABLE milestone;` then `DROP TABLE requirement_spike;` then `DROP TABLE requirement;` (children-first, strict-inheritance-table first because Postgres requires the FK target to outlive the referencing table when dropping without CASCADE).

### 2.3 `20260622120002_tasks_and_specs.sql`

**Tables (in order):** `task`, then `spec`, then `spec_phase`.

Why this order:

1. `task` first because `spec.task_id` FKs to it.
2. `spec` before `spec_phase` because `spec_phase.spec_id` FKs to it.
3. `spec_phase` last because it is the deepest child and its CHECK constraint (`spec_phase_phase_check`) is the most reviewable when read last.

| Block | Item | Source line (from proposal) |
|---|---|---|
| `task` columns | `id BIGSERIAL PK`, `milestone_id BIGINT NOT NULL REFERENCES milestone(id)`, `title TEXT NOT NULL`, `description TEXT NULL`, `created_at`, `updated_at` | proposal §6 |
| Index | `idx_task_milestone_id ON task(milestone_id)` | proposal §6 |
| `spec` columns | `id BIGSERIAL PK`, `task_id BIGINT NOT NULL REFERENCES task(id)`, `content TEXT NOT NULL`, `start_date DATE NULL`, `end_date DATE NULL`, `created_at`, `updated_at` | proposal §7 |
| CHECK | `spec_dates_valid CHECK (end_date IS NULL OR start_date IS NULL OR end_date >= start_date)` | proposal §7 |
| Index | `idx_spec_task_id ON spec(task_id)` | proposal §7 |
| `spec_phase` columns | `id BIGSERIAL PK`, `spec_id BIGINT NOT NULL REFERENCES spec(id)`, `phase TEXT NOT NULL`, `started_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `ended_at TIMESTAMPTZ NULL`, `notes TEXT NULL` (agent reasoning), `created_at`, `updated_at` | proposal §8, locked decision Q4 |
| CHECK 1 | `spec_phase_phase_check CHECK (phase IN ('tdd_red','implementation','tdd_green','verify','pr','technical_ai_review','ai_approved','human_approved'))` (exactly 8 values, locked decision Q1) | proposal §8 |
| CHECK 2 / natural key | `spec_phase_natural_key UNIQUE (spec_id, phase, started_at)` | proposal §8, locked decision Q4 |
| CHECK 3 | `spec_phase_dates_valid CHECK (ended_at IS NULL OR ended_at >= started_at)` | proposal §8 |
| Index 1 | `idx_spec_phase_spec_id ON spec_phase(spec_id)` | proposal §8 |
| Index 2 (partial) | `idx_spec_phase_current_state ON spec_phase(spec_id) WHERE ended_at IS NULL` | proposal §8 |

The **partial UNIQUE index on `(spec_id) WHERE ended_at IS NULL`** (proposed in the prompt's risk #2 mitigation) is **NOT** added to this file in v1. See §6.2 for the explicit follow-up decision and §6 for the rationale.

Down block: `DROP TABLE spec_phase;` then `DROP TABLE spec;` then `DROP TABLE task;` (children-first).

## 3. Test architecture

**File:** `backend/database_administrator/src/migration/runner_test.go`.
**Package:** `migration` (same as existing).
**Gate:** `INTEGRATION=1` (reusing `integrationRunnerDB` at `runner_test.go:42-71`).

### 3.1 Reused infrastructure

| Helper | Location | Why we keep it |
|---|---|---|
| `integrationRunnerDB(t)` | `runner_test.go:42-71` | Returns a `*sql.DB` pointed at the compose-postgres, or skips the test if `INTEGRATION != "1"`. Defaults match compose: `localhost:5432`, db `cachicamas_pg`, user `cachicamas`, pass `changeme-local-only`. |
| `resetSchemaMigrations(t, db)` | `runner_test.go:87-94` | Wipes `public.schema_migrations` so each test starts from an empty bookkeeping table. Used at the top of each new test, identical to the existing 4 tests. |
| `newTestRunner(db)` | `runner_test.go:113-118` | Constructs a `GooseRunner` with table name `schema_migrations` and a discard logger. |
| `discardLogger()` | `runner_test.go:100-106` | Drops log records; we assert on DB state, not log output. |
| `envOr(t, key, fallback)` | `runner_test.go:74-80` | Reads an env var with a default. |

### 3.2 Test-cleanup strategy (recommended)

**Recommendation: TRUNCATE the new tables in `t.Cleanup()`. Do NOT delete the `schema_migrations` bookkeeping rows.**

The 6 new tests are listed below with their cleanup recipes. Justification:

- The bookkeeping table is **append-only by design** (`migration/README.md:121-148`). Deleting rows from it is what the runner is explicitly designed to recover from, but doing so in a test is wasteful — we want the test to leave the DB in the same shape the runner would produce on a real boot.
- TRUNCATE on the new tables is fast, idempotent, and FK-safe (TRUNCATE ... CASCADE handles the FK chain). It also documents which tables are "owned" by this change.
- Using `BEGIN; ... ROLLBACK` in a transaction is tempting but does NOT work for the new tests — the runner commits each `Up` call (it opens its own transaction per `CREATE TABLE` via `-- +goose StatementBegin/End`); wrapping the test in a transaction would deadlock against the runner's own writes. **Do NOT use savepoints.**
- A test-only `DELETE FROM schema_migrations WHERE version_id LIKE '20260622%'` was considered. Rejected: it deletes state that the runner would normally produce; subsequent runs that depend on the prior state (e.g., the existing `TestRunner_Up_SecondBootIsNoOp` style assertions) would not behave like a real boot. The bookkeeping rows from these tests are harmless and read-only.

The new tests' cleanup blocks follow this template:

```go
t.Cleanup(func() {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    // CASCADE truncates spec_phase -> spec -> task -> milestone ->
    // requirement_spike -> requirement -> project -> organization.
    _, _ = db.ExecContext(ctx,
        "TRUNCATE TABLE spec_phase, spec, task, milestone, requirement_spike, requirement, project, organization CASCADE")
})
```

This template is the same across all 6 new tests, but only the tables that test N actually populates need to be in the TRUNCATE list. The defensive full CASCADE is cheap (TRUNCATE on empty tables is ~1ms) and prevents a future schema addition from leaving stragglers.

### 3.3 The 6 new test functions

Each test is described as **Given / When / Then**. The full assertions live in the test body (this design locks WHAT they assert; the test code itself fills in the assertion details).

**T1 — `TestRunner_Up_AllNewMigrationsApply`** (extension of the existing `TestRunner_Up_FirstBoot`)

- **Given** an empty `public.schema_migrations` table.
- **When** `runner.Up(ctx)` is called.
- **Then** the runner applies 3 migrations (in order: 20260622120000, 20260622120001, 20260622120002); the bookkeeping table has 4 rows total (1 hello + 3 new) in chronological order; each new table exists in `public` with the expected columns and the 8 expected phase values for `spec_phase.phase`.

**T2 — `TestRunner_Up_FKConstraintsApply`**

- **Given** an empty `public.schema_migrations` table, then `runner.Up(ctx)` ran successfully.
- **When** we attempt `INSERT INTO project (organization_id, key, full_name) VALUES (99999, 'orphan', 'orphan')`.
- **Then** Postgres returns a foreign-key violation error (`pq: insert or update on table "project" violates foreign key constraint "project_organization_id_fkey"`).

**T3 — `TestRunner_Up_StrictInheritanceOnMilestone`**

- **Given** the schema is applied and a `requirement` row exists.
- **When** we inspect `pg_index` for `milestone`'s primary key, AND we attempt to insert TWO `milestone` rows for the same `requirement_id`.
- **Then** the primary-key index lists ONLY `requirement_id` (no synthetic `id` column); the second insert fails with a duplicate-key violation on `milestone_pkey`.

**T4 — `TestRunner_Up_AppendOnlyConvention`**

- **Given** the schema is applied.
- **When** we query `col_description('organization'::regclass, attnum)` for the `is_active` column.
- **Then** the comment is `'UPDATE-in-place: this column is the ONLY mutation allowed on organization rows.'` (verbatim — asserts the convention is documented, not enforced). This is the only "convention" assertion in the suite; DB-level enforcement is a follow-up (see §6.2).

**T5 — `TestRunner_Up_PRDSizeCapEnforced`**

- **Given** the schema is applied, a `project` and a `requirement` parent row exist.
- **When** we attempt `INSERT INTO requirement (project_id, filename, content) VALUES (1, 'big.md', repeat('x', 262145))`.
- **Then** Postgres returns a CHECK constraint violation (`new row for relation "requirement" violates check constraint "requirement_content_size_cap"`). Companion assertion: `INSERT ... repeat('x', 262144)` succeeds (boundary check).

**T6 — `TestRunner_Up_SpecPhaseReEntry`** (the agent-first scenario)

- **Given** the schema is applied, a `spec` row exists.
- **When** we run the canonical agent-first pattern: `INSERT spec_phase (spec_id, phase, notes) VALUES (?, 'tdd_red', ...)`; then `UPDATE spec_phase SET ended_at = now() WHERE spec_id = ? AND ended_at IS NULL`; then `INSERT spec_phase (spec_id, phase, notes) VALUES (?, 'technical_ai_review', ...)`; then `UPDATE ... ended_at ...`; then `INSERT spec_phase (spec_id, phase, notes) VALUES (?, 'tdd_red', 're-entering after AI review found X')`.
- **Then** all 4 inserts succeed; the natural-key UNIQUE `(spec_id, phase, started_at)` holds because the two `tdd_red` rows have different `started_at` (separated by `time.Sleep(10ms)` to force non-equal microseconds); the partial index `idx_spec_phase_current_state` returns exactly 1 row for the spec (the second `tdd_red` row, which is still open).

**Plus:** extend the existing `TestRunner_Up_LexicographicOrder` (`runner_test.go:217-271`) to seed all 4 versions and assert the runner applies them in 20260621120000, 20260622120000, 20260622120001, 20260622120002 order. The existing test already handles a "non-monotonic" prefix; the extension adds the 3 new versions in lexicographic order with their timestamps spaced 1s apart.

### 3.4 Test-isolation contract

Each new test:

1. Calls `resetSchemaMigrations(t, db)` (line 87) at the top — same as existing.
2. Calls `runner.Up(ctx)` — applies 3 new migrations.
3. Runs its assertions against the resulting schema.
4. Calls `t.Cleanup(...)` with the TRUNCATE recipe from §3.2.

Running all 6 tests serially leaves the DB in the same state as a single fresh boot: 4 rows in `schema_migrations`, 8 tables in `public`, no test data. Running them in any order is safe because the TRUNCATE is unconditional.

## 4. Observability

### 4.1 What gets emitted when the 3 new migrations apply

When the runner calls `provider.Up(ctx)` against the compose stack with all 3 new SQL files in `sql/`, the runner emits **one** OTel span named `migration.up` (`runner.go:97-105`) and **one** `slog.Info` line (`runner.go:130-133`):

| Attribute | Value | Why |
|---|---|---|
| `db.system` | `"postgresql"` | hard-coded at `runner.go:100` |
| `migration.dir` | `"sql"` | hard-coded at `runner.go:101` |
| `migration.table` | `"schema_migrations"` | from `runner.go:102` (matches the runner's bookkeeping table) |
| `migration.duration_ms` | e.g., `~200` (typical for 3 CREATE-TABLE-only files on a warm DB) | from `runner.go:111` |
| `migration.applied_count` | `3` (when the 3 new files apply; `4` on first boot against an empty DB because the hello-world migration is also there) | from `runner.go:129` |

**Per-table span? No.** The runner emits ONE span per `Up` call, not one span per table or per migration file. The 3 new migrations show up as a single `migration.applied_count = 3` (or `4` on a clean first boot).

### 4.2 New OTel spans?

**No.** The existing `migration.up` span covers all 3 new migrations as a single `Up` call. The user does not need per-migration spans (the bookkeeping table records each version with its timestamp), and adding per-table spans would require changes to `runner.go` — which is out of scope per `proposal.md §"What does NOT change"`.

If a future change wants per-migration spans, it would require either:

- A `migration_service.go` change to wrap each `goose.MigrationResult` in its own span (would need to be wired into the runner's result loop), OR
- A pre-`Up` introspection that enumerates the embed.FS and opens a parent span per file.

Neither is in scope for v1.

### 4.3 OTel upstream bug (SDK v1.44.0)

The upstream bug `opentelemetry-go#5248` (documented at `migration/README.md:216-222`) drops ~9/10 spans at the OTLP exporter. The 3 new migrations inherit this behavior: the span is created (the runner does not depend on the exporter), but it may not reach Jaeger.

**Mitigation: do nothing.** The migration succeeds regardless of span delivery. If an operator needs the trace ID for diagnosis, follow `migration/README.md:209-214` — the `trace_id` is recoverable from `docker compose logs otel-collector | grep "Trace ID"`. This is the existing behavior and is explicitly out of scope.

### 4.4 Log lines

Each `Up` call emits one of:

- Success: `INFO migration.up applied applied_count=4 duration_ms=~200` (first boot) or `applied_count=3 duration_ms=~200` (subsequent boots that pick up only the 3 new files).
- Failure: `ERROR migration.up failed error.kind=pg_query_error error="..." duration_ms=...`.

No new log lines are added. No new log attributes are needed.

## 5. Error handling

### 5.1 SQL syntax error in a migration file

- **What happens:** `provider.Up(ctx)` returns the syntax error wrapped by goose; `runner.Up` returns it (`runner.go:113-126`); the application service classifies it as `pg_query_error` (`runner.go:268-282`); the OTel span gets `Status=Error` and `migration.error.kind=pg_query_error`; the composition root calls `os.Exit(1)` (`README.md:250`).
- **Recovery:** the operator fixes the SQL, rebuilds, and restarts the container. The bookkeeping table does NOT have a row for the broken migration (the transaction rolled back; `README.md:255-260`). The next boot re-applies.
- **Test impact:** none — the new tests do not assert on syntax errors. The existing `TestRunner_Status_UpstreamErrorPropagates` (`runner_test.go:400-424`) is sufficient.

### 5.2 CHECK constraint rejects existing data (would-be 5.2)

- **Scenario:** hypothetical — a future change adds a new CHECK constraint that rejects rows from the 8 new tables. The migration fails because the constraint is checked against existing rows in the same transaction.
- **What happens:** the migration's transaction rolls back (goose v3 semantics, `README.md:255-260`); no bookkeeping row is inserted; the same `pg_query_error` path as §5.1 applies.
- **v1 implication:** this scenario CANNOT happen because the 8 tables are brand new and have no existing data. A future migration that adds a CHECK to an existing table (out of scope) would hit this; it would need to use `ALTER TABLE ... ADD CONSTRAINT ... NOT VALID` followed by `VALIDATE CONSTRAINT` to avoid the table-level lock (`README.md:332-343`).

### 5.3 Agent INSERT violates `UNIQUE (spec_id, phase, started_at)`

- **What happens:** Postgres raises `pq: duplicate key value violates unique constraint "spec_phase_natural_key"`. The agent's query (whatever driver it uses — pgx, psql, or the future witsaba-requirements-api) bubbles up the error.
- **Why this is a "soft error":** it does NOT crash the runner or the container. The migration is complete; the agent's runtime surfaces the constraint violation to whatever called `INSERT INTO spec_phase ...`. The agent's recovery: change the `started_at` by a microsecond (or re-`UPDATE ended_at` on the existing row instead of `INSERT`-ing a new one). The 6th test (`T6` in §3.3) covers the canonical re-entry pattern.
- **Test impact:** T6 asserts that the natural-key UNIQUE holds across multiple rows with different `started_at`; the failure path is exercised implicitly by the Go test framework's `t.Fatal` semantics if the constraint blocks a valid INSERT (it should not, because we `time.Sleep(10ms)` between rows).

### 5.4 Agent UPDATE violates `ended_at >= started_at`

- **What happens:** identical to §5.3 — Postgres raises a CHECK violation; the agent's runtime surfaces it; the runner is unaffected.
- **Test impact:** not asserted directly. The CHECK is exercised on INSERT (Postgres evaluates CHECKs on every write). A follow-up test could `INSERT spec_phase (spec_id, phase, started_at) VALUES (?, 'tdd_red', '2026-01-01'); UPDATE spec_phase SET ended_at = '2025-12-31' WHERE id = ?` and assert the violation; deferred to a future spec_phase-specific test change.

### 5.5 Container loses connection mid-migration

- **What happens:** the `pgx` connection's TCP socket closes; Postgres releases the session-scoped advisory lock automatically on connection close (`README.md:286-290`); the runner's `provider.Up` returns an error; the composition root exits 1; the orchestrator restarts the container.
- **On the next boot:** the bookkeeping table does NOT have a row for the partially-applied file (the transaction rolled back); the runner re-applies from scratch.
- **Edge case — half-applied WITH a bookkeeping row:** if somehow the row was committed but the schema was not (e.g., `CREATE TABLE` outside a transaction in some future migration), the operator follows `README.md:262-285` to manually repair. **For v1, every `CREATE TABLE` is inside `-- +goose StatementBegin/End`, so this edge case is not reachable.**

## 6. Risk mitigations

The 5 risks from `proposal.md §"Risks"` map to concrete mitigations in v1. Two are addressed in this change; two are deferred; one is a no-op.

### 6.1 R1: 256 KiB PRD cap is hard — IN-SCOPE, documented in v1

- **Mitigation:** the column comment on `requirement.content` (`proposal §3`) reads `'Max PRD size: 256 KiB (262144 bytes), enforced via CHECK constraint.'` — both a constraint name and a human-readable reason. A contributor who hits the cap sees the comment first, then the constraint.
- **Future escape hatch:** `ALTER TABLE requirement DROP CONSTRAINT requirement_content_size_cap; ALTER TABLE requirement ADD CONSTRAINT requirement_content_size_cap CHECK (octet_length(content) <= <new_cap>);`. Documented here; no migration is committed for the future change (it would be a no-op against the existing constraint).
- **Test:** T5 (`TestRunner_Up_PRDSizeCapEnforced`) asserts the boundary at 262144 bytes (passes) and at 262145 bytes (fails with the constraint violation).

### 6.2 R2: `spec_phase` re-entry relies on app discipline — DEFERRED to follow-up

- **Decision: NOT IN SCOPE for v1.** The partial UNIQUE index `CREATE UNIQUE INDEX idx_spec_phase_one_open_per_spec ON spec_phase(spec_id) WHERE ended_at IS NULL;` is **NOT** added to `20260622120002_tasks_and_specs.sql`.
- **Rationale:**
  1. The locked convention (proposal §"Conventions locked", A2) describes `spec_phase` as an append-only history table whose invariant is enforced at the application layer for v1.
  2. The proposal explicitly lists DB-level append-only enforcement as a follow-up (`proposal §"What does NOT change"`, bullet 4).
  3. The agent-first re-entry SQL pattern in `proposal.md` ("Agent-first re-entry pattern (canonical SQL)") already includes the closing UPDATE — the canonical agent behavior IS to close the current row before opening the next one.
  4. The `idx_spec_phase_current_state` partial index from proposal §8 is what makes the "what phase is this spec in right now?" query a single B-tree seek; adding a UNIQUE on top of it would prevent legitimate re-entries in the future (e.g., parallel agents on the same spec — currently not supported, but the schema should not block it).
- **Compensating controls in v1:**
  - The `notes` column on `spec_phase` (locked decision Q3) is the agent's reasoning for each transition. A bug that leaves two open phases is detectable via the partial index query (`WHERE ended_at IS NULL`) returning >1 rows.
  - The `runner_test.go:400-424` pattern (existing `TestRunner_Status_UpstreamErrorPropagates`) is available as a template for a follow-up test that asserts at most 1 open phase per spec.
- **Follow-up change:** `witsaba-core-tables-append-only-enforcement` (proposed, not started). Will add the partial UNIQUE index, a DB-level REVOKE on UPDATE for `spec_phase`, and a unit test that asserts the agent re-entry pattern fails when the previous row is NOT closed.

### 6.3 R3: `MIGRATION_TIMEOUT` (30s) may be tight — IN-SCOPE, documented procedure

- **Expected duration:** the 3 new migrations are CREATE-TABLE-only (no backfill, no data migration, no `CREATE INDEX CONCURRENTLY`). On a warm compose stack, 3 CREATE TABLE blocks with `-- +goose StatementBegin/End` wrappers take well under 1 second. The 30s default is comfortable.
- **First-apply procedure:**
  1. Boot the container once with the default `MIGRATION_TIMEOUT=30s`.
  2. Read the `migration.duration_ms` attribute on the `migration.up` span (or the `duration_ms` slog line).
  3. If duration is under 25s, leave the timeout alone.
  4. If duration is over 25s, set `MIGRATION_TIMEOUT=60s` for the first apply, then revert to 30s after the bookkeeping row is present.
- **Test impact:** the existing tests run with `context.WithTimeout(30 * time.Second)` (`runner_test.go:134`). The new tests MUST use the same 30s window. If T1 (`TestRunner_Up_AllNewMigrationsApply`) consistently exceeds 25s on the local compose stack, the runner is doing something wrong (probably OTel overhead per `migration_service.go:144`); investigate before bumping the timeout.

### 6.4 R4: OTel upstream bug — NO ACTION

- See §4.3. The migration succeeds; the spans may not reach Jaeger; the `trace_id` is recoverable from the collector logs. No code change.

### 6.5 R5: strict inheritance on `milestone` is locked at 1:1 — DOCUMENTED escape hatch

- **Decision: NO migration escape hatch in v1.** The escape hatch is a future table called `project_milestone` (mentioned in `explore.md §A1` and `proposal.md §"Risks" 5`).
- **Why not ship `project_milestone` now:** it is not in the user's locked requirements (the proposal locks cardinality for v1; see A1). The 8-table schema is the v1 deliverable.
- **Test:** T3 (`TestRunner_Up_StrictInheritanceOnMilestone`) asserts the 1:1 invariant by attempting a duplicate insert.
- **Follow-up change:** `witsaba-core-tables-project-milestone` (proposed, not started). Will add a `project_milestone` table with a synthetic PK, FK to `project(id)`, and the same date-pair CHECK constraint.

## 7. Implementation order

The change is small enough for a single PR (per `proposal.md §"PR size forecast"`), but the work breaks into 4 work-unit commits that map to the 4 work units in the `work-unit-commits` skill. Each commit passes `make test` independently.

### Commit 1 — `feat(db): add orgs and projects tables with FK integration test`

- **Files:** `src/migration/sql/20260622120000_orgs_and_projects.sql` (new).
- **Tests:** `TestRunner_Up_AllNewMigrationsApply` and `TestRunner_Up_FKConstraintsApply` (in `runner_test.go`).
- **Verification:** `make test/integration` runs both new tests green; `make test` (no INTEGRATION) skips both; no other tests regress.
- **Rollback:** delete the SQL file; delete the two tests; runner reverts to the hello-world-only state.

### Commit 2 — `feat(db): add requirements, spikes, and strict-inherited milestone`

- **Files:** `src/migration/sql/20260622120001_requirements_and_milestones.sql` (new).
- **Tests:** `TestRunner_Up_PRDSizeCapEnforced` and `TestRunner_Up_StrictInheritanceOnMilestone` (in `runner_test.go`).
- **Verification:** `make test/integration` runs all 4 new tests green.
- **Rollback:** delete the SQL file and the 2 tests; the first migration file stays applied.

### Commit 3 — `feat(db): add tasks, specs, and append-only spec_phase history`

- **Files:** `src/migration/sql/20260622120002_tasks_and_specs.sql` (new).
- **Tests:** `TestRunner_Up_AppendOnlyConvention` and `TestRunner_Up_SpecPhaseReEntry` (in `runner_test.go`); extend `TestRunner_Up_LexicographicOrder` to seed all 4 versions.
- **Verification:** `make test/integration` runs all 6 new tests green plus the extended lexicographic-order test.
- **Rollback:** delete the SQL file, the 2 new tests, and the lexicographic-order extension.

### Commit 4 — `chore(db): document witsaba-core-tables migration in migration README`

- **Files:** `src/migration/README.md` — add a §11 "Schema overview" subsection that lists the 8 tables with their canonical use, the cardinality decisions (1:1 milestone, 1:N elsewhere), and the append-only conventions. Pure docs; no behavior change.
- **Verification:** `make test` and `make lint` are still green.
- **Why a separate commit:** keeps the diff for the schema work itself clean; the README update is a follow-up doc.

The 4 commits land in a single PR (per `delivery_strategy: single-pr`). The PR description cites this design document and the proposal.

## 8. Definition of done

The change is DONE when **every** item below is checked off:

- [ ] `src/migration/sql/20260622120000_orgs_and_projects.sql` exists with `organization` + `project` + the `COMMENT ON COLUMN organization.is_active` + the `idx_project_organization_id` index.
- [ ] `src/migration/sql/20260622120001_requirements_and_milestones.sql` exists with `requirement` (incl. `requirement_content_size_cap` CHECK + `COMMENT ON COLUMN requirement.content`) + `requirement_spike` + `milestone` (PK = FK = `requirement_id`).
- [ ] `src/migration/sql/20260622120002_tasks_and_specs.sql` exists with `task` + `spec` + `spec_phase` (incl. `spec_phase_phase_check` with 8 values, `spec_phase_natural_key`, `spec_phase_dates_valid`, the two indexes including the partial one).
- [ ] All 3 `-- +goose Down` blocks reverse in dependency order; running `goose down` to version 0 leaves `public` empty (asserted manually post-merge by an operator).
- [ ] 6 new tests added to `runner_test.go`: `TestRunner_Up_AllNewMigrationsApply`, `TestRunner_Up_FKConstraintsApply`, `TestRunner_Up_StrictInheritanceOnMilestone`, `TestRunner_Up_AppendOnlyConvention`, `TestRunner_Up_PRDSizeCapEnforced`, `TestRunner_Up_SpecPhaseReEntry`.
- [ ] Existing `TestRunner_Up_LexicographicOrder` extended to cover all 4 versions.
- [ ] All 6 new tests + the extended test pass under `INTEGRATION=1 make test`.
- [ ] `make test` passes (race-enabled, full suite, INTEGRATION unset → integration tests skip).
- [ ] `make lint` passes (golangci-lint v2.9.0; no new vet/staticcheck/revive issues).
- [ ] No source code under `src/` (Go) is modified: `runner.go`, `embed.go`, `postgres/driver.go`, `application/migration_service.go`, `domain/migration.go`, `otel/*`, `cmd/server/main.go` are all untouched.
- [ ] No new top-level Go dependencies introduced (no changes to `go.mod` / `go.sum`).
- [ ] All 4 work-unit commits follow Conventional Commits; no `Co-Authored-By` trailer; no AI attribution.
- [ ] PR opened against `main` (single PR per cached `delivery_strategy: single-pr`).
- [ ] The partial UNIQUE index on `(spec_id) WHERE ended_at IS NULL` is explicitly NOT in scope (decision recorded in §6.2).
- [ ] The Incompleteness Log (`wiki/Incompleteness-Log.md`, if it exists) is updated with the open follow-ups: append-only enforcement (§6.2) and `project_milestone` (§6.5).

## Alternatives considered

- **One migration file per table (8 files).** Rejected in `proposal §"Alternatives considered"`. 8 tiny files make the dependency chain harder to follow and bloat the PR for no review gain.
- **One migration file for all 8 tables.** Rejected in `proposal §"Alternatives considered"`. A 250+ line SQL file is hard to review; goose handles it fine, humans do not.
- **Composite natural PK on `spec_phase` (e.g., `(spec_id, started_at)`).** Rejected in `proposal §"Alternatives considered"`. A future child table (e.g., `spec_phase_artifact` for AI feedback per phase) would need a composite FK; the 8-byte cost of a synthetic PK is negligible.
- **Partial UNIQUE index in v1.** Rejected (see §6.2). Defers to a follow-up change to keep v1 aligned with the user's locked decision (proposal A2: app-layer discipline).
- **Per-table OTel spans.** Rejected (see §4.2). The existing `migration.up` span covers the entire `Up` call; per-table spans would require runner changes that are out of scope.
- **BEGIN/ROLLBACK transaction per test.** Rejected (see §3.2). The runner opens its own transactions per `-- +goose StatementBegin/End`; a wrapping test transaction would deadlock.
- **`DELETE FROM schema_migrations WHERE version_id LIKE '20260622%'` in cleanup.** Rejected (see §3.2). The bookkeeping table is append-only by design; tests should not delete from it. TRUNCATE on the new tables is sufficient.

## Review checklist

- [ ] reviewer can confirm no new Go files are added (DDL + tests only — pure SQL + additions to `runner_test.go`)
- [ ] reviewer can confirm the 3 SQL files are in the right order: `20260622120000_orgs_and_projects.sql`, `20260622120001_requirements_and_milestones.sql`, `20260622120002_tasks_and_specs.sql`
- [ ] reviewer can confirm `organization` is created before `project`, `requirement` before `requirement_spike` and `milestone`, and `task` before `spec` before `spec_phase`
- [ ] reviewer can confirm `milestone` uses strict inheritance (PK = FK = `requirement_id`, no synthetic `id`)
- [ ] reviewer can confirm `requirement_content_size_cap` CHECK uses `octet_length(content) <= 262144` AND the column comment is on `requirement.content`
- [ ] reviewer can confirm `spec_phase` has the 3 constraints (phase_check with 8 values, natural_key UNIQUE, dates_valid) and the 2 indexes (spec_id, partial current_state)
- [ ] reviewer can confirm the Down blocks reverse in dependency order (children first, strict-inheritance first within a level)
- [ ] reviewer can confirm the 6 new test names match §3.3 and that each test's Given/When/Then is documented
- [ ] reviewer can confirm the test cleanup strategy is TRUNCATE-the-new-tables-CASCADE (NOT delete from schema_migrations, NOT BEGIN/ROLLBACK)
- [ ] reviewer can confirm all 6 new tests reuse the existing `INTEGRATION=1` gate at `runner_test.go:42-71` (no new gating mechanism)
- [ ] reviewer can confirm the partial UNIQUE index on `(spec_id) WHERE ended_at IS NULL` is **NOT** added to `20260622120002_tasks_and_specs.sql` (deferred to follow-up)
- [ ] reviewer can confirm no new OTel spans are added (single `migration.up` span covers the 3 new migrations)
- [ ] reviewer can confirm the implementation order is 4 work-unit commits, each independently testable
- [ ] reviewer can confirm `make test`, `make test/integration`, and `make lint` are all listed in the Definition of Done
