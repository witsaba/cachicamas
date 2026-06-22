# Explore — witsaba-core-tables

> **Change**: `witsaba-core-tables`
> **Status**: explored
> **Date**: 2026-06-22
> **Driver**: braejan
> **Project**: cachicamas (witsaba framework)
> **Persistence**: hybrid (Engram `sdd/witsaba-core-tables/explore` + filesystem)

## Executive summary

- The migration system in `backend/database_administrator/` is **production-grade and ready to accept new files**: hexagonal layout, `pressly/goose` v3.27.1 + `jackc/pgx/v5` stdlib, embed.FS picks up new files at build time, advisory lock + idempotency proven by integration suite.
- The **witsaba framework** introduces a **load-bearing convention** — single-PK-as-FK inheritance — that conflicts with classical relational modeling. Most one-to-many relationships the user describes will need explicit cardinality decisions before SQL can be written.
- **Five open questions** must be answered before specs: milestone parent (req vs project), phases representation (column vs history), spike/task/spec cardinality, and `is_active` lifecycle.
- **Lower-footprint principle** drives the column budget: each table is documented with the minimum columns to satisfy the user's verbatim requirements plus a clear "NOT now" list. The `metadata JSONB` column is allowed on `project` because the user explicitly asked for it; nowhere else.
- **Append-only enforcement** is a layered concern: app-layer discipline is the primary line of defense; DB-level enforcement is a follow-up decision (REVOKE UPDATE, BEFORE-UPDATE trigger, or a dedicated `*_history` table for state changes).

## Current state of the migration system

**Stack**: Go 1.26.3, `pressly/goose` v3.27.1, `jackc/pgx/v5` v5.10.0 stdlib adapter, `labstack/echo/v5` v5.2.1, OTel SDK v1.44.0. Postgres 18 (`postgres:18-alpine3.24`). DB user `queen` (NOSUPERUSER, CREATEROLE, CREATEDB) provisioned by `infra/postgres/init/01-init.sql` (2 extra lines: `OWNER TO queen` and `SET timezone='UTC'`).

**Hexagonal layout** (from `cmd/server/main.go:72-158`):
- **Domain port**: `domain/migration.go` — `Runner` interface (`Up`, `Status`).
- **Application use case**: `application/migration_service.go:52` — opens OTel span `migration.up`, classifies errors into `pg_advisory_lock_timeout`/`pg_connect_error`/`pg_query_error`/`embed_error`.
- **Adapter**: `migration/runner.go:55` — `GooseRunner` wraps goose; per-`Up` builds a fresh `goose.Provider` with `pg_try_advisory_lock(42)` (session-scoped) + `fs.Sub(MigrationsFS, "sql")` + `goose.WithTableName(tableName)`.
- **Driver**: `migration/postgres/driver.go:33` — pgx/v5 stdlib blank-imported; `LoadConfigFromEnv` honors `DATABASE_URL` first, then discrete `POSTGRES_*`.
- **Composition root**: `main.go:150-158` — runner is built, then `service.Up(migrateCtx)` runs **before** Echo binds. Failure → `os.Exit(1)`.

**Bookkeeping**: `public.schema_migrations` table, columns `version_id`, `is_applied`, `tstamp`. Goose v3 uses one-row-per-version with no `dirty` column.

**Idempotency**: natural via bookkeeping. Second boot with no new files = zero applied. Locking: `pg_try_advisory_lock(42)`, session-scoped, released on connection close or explicit unlock.

**Tests**:
- Use case: pure unit (`application/migration_service_test.go`), in-memory OTel + JSON slog.
- Runner: mix of unit + integration gated on `INTEGRATION=1` (`migration/runner_test.go:44`), live DB at compose defaults.
- Driver: env-loading + ping tests (`migration/postgres/driver_test.go`).
- Compile-time interface check: `var _ domain.Runner = (*GooseRunner)(nil)` (`runner.go:323`).

## Reusable patterns (with file:line)

1. **Filename convention** — `^\d{14}_[a-z0-9_]+\.sql$` (`migration/README.md:1`). 14-digit UTC timestamp `YYYYMMDDHHMMSS` + `_` + snake_case context/description + `.sql`. Lexicographic sort = chronological order.
2. **Annotation format** — `-- +goose Up` / `-- +goose Down` / `-- +goose StatementBegin` / `-- +goose StatementEnd`. Single file with both blocks. Legacy v2 paired files REJECTED by v3 (`README.md:74-76`).
3. **Embed pattern** — `//go:embed sql/*.sql` (`migration/embed.go:28`). Flat, non-recursive; subdirs under `sql/` are ignored (`README.md:§8.2`).
4. **Boot hook** — runner called in `main.go:153-158` pre-Echo, bounded by `MIGRATION_TIMEOUT` (default 30s), fail-fast on error.
5. **OTel span attributes** — `db.system=postgresql`, `migration.dir=sql` (hard-coded), `migration.table=<tableName>`, `migration.duration_ms`, `migration.applied_count` (success only), `migration.error`+`migration.error.kind` (failure only).
6. **Error classification** — substring match against `err.Error()` in `runner.go:268` + `migration_service.go:144` (duplicated by design).
7. **UTC enforcement** — server-side via `01-init.sql` (`SET timezone='UTC'`). Application code does NOT call `time.Now().UTC()` explicitly.
8. **Idempotent default values** — `created_at` and `updated_at` should use `DEFAULT now()` to satisfy the user's "both equal on insert" requirement without a trigger.

## The 5 open questions

### A1. Milestone parent: requirement or project?

**Trade-off**:
- The user wrote literally: "I wanna a table for the milestone inherit from the requirement." → milestone's parent is `requirement`.
- But semantically a milestone is a **time-boxed deliverable within a project**, not within a single requirement. A project with N requirements usually has milestones that span multiple requirements.
- If milestone is a child of requirement, then "a milestone that covers requirements A and B" is impossible without breaking the inheritance convention.

**Recommended answer**: **milestone is a child of `requirement`**, sticking to the user's literal wording and the locked convention. If the user later wants cross-requirement milestones, that's a NEW table (`project_milestone`) with its own synthetic PK, NOT a violation of this convention.

**Confirm with user**: "Did you mean milestone inherits from **requirement** (one requirement → many milestones) — i.e. each milestone is scoped to a single requirement? Or did you mean **project** (one project → many milestones)?"

### A2. Phases representation: single `phase` column vs `spec_phase` history table

**Trade-off**:
- Single `phase` column on `spec` (CHECK-constrained to the 9 values) → current state only, smaller footprint, but loses history. Phase transitions overwrite the previous value.
- `spec_phase` history child table (PK = `(spec_id, started_at)` or similar) → append-only timeline, full audit trail. Bigger footprint, but append-only is the user's principle.
- The 9 phases (`TDD red`, `Implementation`, `TDD green`, `Verify`, `PR`, `Technical AI Review of PR`, `AI approved`, `Human Approved` — and one more I need to confirm) are sequential, but the user might want to record each transition.

**Recommended answer**: **single `phase` column on `spec` for v1, with a comment that history is a follow-up**. Lower footprint wins for the initial change. The `spec_phase` history table can be added later if the user needs audit. Append-only is enforced at the app layer for v1 (no UPDATE statements in the data access code); DB-level enforcement is a follow-up.

**Confirm with user**: "For v1, do you want a single `phase` column on `spec` (current state, smaller footprint) or a `spec_phase` history table (full timeline, append-only)?"

### A3. `requirement_spike` cardinality

**Trade-off**:
- 1:1 (inheritance pattern enforces this — `requirement_spike.requirement_id` is both PK and FK) — one row per requirement.
- 1:many (one requirement → many spikes over time) — would require a synthetic PK on `requirement_spike`, breaking the inheritance convention.
- User wrote "a table for spikes" (singular noun, but in English this is ambiguous).

**Recommended answer**: **1:1, sticking to the inheritance pattern**. The user's wording suggests one spike per requirement. If multi-spike support is needed later, drop the inheritance convention for that edge (give `requirement_spike` its own synthetic PK and a non-PK FK to `requirement`).

**Confirm with user**: "Should one requirement have exactly one spike, or many spikes over its lifetime?"

### A4. `task` and `spec` cardinality

**Trade-off**:
- The user wrote "every task inherit a spec one" — "spec one" reads 1:1. Each task has exactly one spec.
- But "spec" semantically can mean "the spec document for this task" (1:1) or "all the spec iterations of this task" (1:many, with history).
- A `task` is a unit of work; a `spec` describes what the work must satisfy. Most workflows produce 1 spec per task, then iterate on it.

**Recommended answer**: **1:1 between task and spec**. The `spec` row contains the current spec for the task. If the spec needs versioning, that's a separate `spec_version` child table (append-only, owned by the task) added later.

**Confirm with user**: "Is one task supposed to have exactly one spec, or many specs over time?"

### A5. `is_active` lifecycle

**Trade-off**:
- Hard-delete: simple, but loses history. The user said append-only by default, which conflicts.
- Soft-delete: a `deactivated_at TIMESTAMPTZ NULL` column; row stays queryable but is "off". Adds a column.
- Pure boolean `is_active`: UPDATE-in-place exception. Smallest footprint. The row stays queryable.

**Recommended answer**: **pure `is_active BOOLEAN NOT NULL DEFAULT TRUE`**, with the explicit understanding that this is one of the few columns the app will UPDATE. Deactivation is reversible (re-activate by setting to TRUE). No soft-delete column. If the user needs the "when was it deactivated" timestamp, that's a follow-up column.

**Confirm with user**: "Is `is_active` a simple TRUE/FALSE flag (UPDATE-in-place, smallest footprint), or do you need to record WHEN it was deactivated (adds a `deactivated_at` column)?"

## Lower-footprint column budget per table

> For each table: the user's verbatim requirements, the concrete minimum column list, and an explicit "NOT now" list.

### `organization`

| Required (verbatim) | Recommended minimum | NOT now |
|---|---|---|
| Incremental integer PK | `id BIGSERIAL PRIMARY KEY` | UUID, slug, slug-history |
| Shortname nullable | `shortname TEXT` | UNIQUE constraint, length cap |
| Full name NOT NULL | `full_name TEXT NOT NULL` | — |
| Unique org identification (tax ID / RFC / NIT) | `identification TEXT NOT NULL UNIQUE` | International format validation |
| Active/inactive boolean | `is_active BOOLEAN NOT NULL DEFAULT TRUE` | `deactivated_at` |
| Email (verification) | `email TEXT` (nullable) | Format validation, uniqueness |
| Phone (verification) | `phone TEXT` (nullable) | E.164 normalization |

Also: `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`. **No** `metadata JSONB` — none of the user's requirements mention it for organizations.

### `project`

| Required (verbatim) | Recommended minimum | NOT now |
|---|---|---|
| Incremental integer PK | `id BIGSERIAL PRIMARY KEY` | UUID |
| Key name like `cachicamas` | `key TEXT NOT NULL UNIQUE` | Slug validation, prefix scheme |
| Full project name | `full_name TEXT NOT NULL` | — |
| Start date, end date | `start_date DATE`, `end_date DATE` (nullable) | TIMESTAMPTZ, recurring project support |
| Optional JSONB metadata | `metadata JSONB` (nullable) | GIN index (add when needed) |

Also: `organization_id BIGINT NOT NULL REFERENCES organization(id)`, `created_at`, `updated_at`. **No** `status` column, **no** `archived` flag — the framework can derive those from `start_date`/`end_date` for now.

### `requirement`

| Required (verbatim) | Recommended minimum | NOT now |
|---|---|---|
| Filename | `filename TEXT NOT NULL` | Path validation, slug |
| Most-native Markdown store (max ~256 KB PRD) | `content TEXT NOT NULL CHECK (octet_length(content) <= 262144)` (comment: max PRD size = 256 KiB) | TOAST compression tuning, separate asset table |
| Git repository remote URL | `git_repository_url TEXT` (nullable) | Per-host validation, multiple URLs |
| Analysis result | `analysis_result TEXT` (nullable) | Structured JSON, versioned analysis |
| Technical viability boolean | `is_technically_viable BOOLEAN` (nullable) | Viability score, reviewer field |

Also: `project_id BIGINT NOT NULL PRIMARY KEY REFERENCES project(id)` (single-PK-as-FK inheritance — PK IS the FK). `created_at`, `updated_at`. **No** `priority`, `status`, `assignee`, `tags` — none in user's requirements.

### `requirement_spike` (assumes 1:1 from A3)

If A3 is 1:1:

| Required | Recommended minimum | NOT now |
|---|---|---|
| (TBD; user said "a table for spikes" without details) | `requirement_id BIGINT PRIMARY KEY REFERENCES requirement(id)` + minimal fields: `started_at DATE`, `ended_at DATE` (nullable), `outcome TEXT` (nullable), `findings TEXT` (nullable, markdown) | Synthetic PK, multi-spike history |

If A3 is 1:many, this table gets a synthetic PK and breaks the inheritance convention. See A3.

Also: `created_at`, `updated_at`.

### `milestone` (parent = `requirement` per A1)

| Required (verbatim) | Recommended minimum | NOT now |
|---|---|---|
| Start date, end date | `start_date DATE`, `end_date DATE` (nullable) | TIMESTAMPTZ |
| Milestone title | `title TEXT NOT NULL` | Slug, UNIQUE |
| Milestone markdown description | `description TEXT` (nullable) | Versioned description |

Also: `requirement_id BIGINT NOT NULL PRIMARY KEY REFERENCES requirement(id)` (single-PK-as-FK). `created_at`, `updated_at`. **No** `status` (open/closed), no `completed_at` — derive from `end_date` for v1.

### `task`

User said "every task inherit a spec one" but did NOT specify task columns. The task is a unit of work; minimal fields the framework will need:

| Recommended minimum | NOT now |
|---|---|
| `milestone_id BIGINT NOT NULL PRIMARY KEY REFERENCES milestone(id)` (PK = FK) | Synthetic PK |
| `title TEXT NOT NULL` | Slug, status enum |
| `description TEXT` (nullable, markdown) | Versioned description |
| (no assignee, no priority — those are framework concerns not yet in scope) | Assignee, priority, tags |

Also: `created_at`, `updated_at`.

### `spec`

User said "every task inherit a spec one" and described phases. The spec is the contract for a task:

| Required (verbatim) | Recommended minimum | NOT now |
|---|---|---|
| The current spec content (assumed markdown) | `content TEXT NOT NULL` (no max size for v1; add CHECK later if needed) | Versioning, separate history table |
| Start date, end date | `start_date DATE`, `end_date DATE` (nullable) | TIMESTAMPTZ |
| Phase | `phase TEXT NOT NULL CHECK (phase IN ('tdd_red','implementation','tdd_green','verify','pr','technical_ai_review','ai_approved','human_approved'))` — 8 values, see Risks | Enum type (low portability, lower cohesion) |

Also: `task_id BIGINT NOT NULL PRIMARY KEY REFERENCES task(id)` (PK = FK). `created_at`, `updated_at`. **No** `spec_version` table (per A2, defer).

> **Phase values clarification needed**: the user listed 9 phase names ("TDD red, Implementation, TDD green, Verify, PR, Technical AI Review of pr, AI approved, Human Approved and that is for now"). I'm counting 8. Confirm with user.

## Append-only enforcement strategy

**Three layers, prioritized for lower footprint:**

1. **App-layer discipline (primary, v1)** — the data access code NEVER issues UPDATE on append-only tables. Enforced by code review. The convention is documented in the table's column comment.
2. **DB-level: revoke UPDATE permission (optional, follow-up)** — `REVOKE UPDATE ON <table> FROM queen;` and `GRANT SELECT, INSERT ON <table> TO queen;`. The `queen` user can't UPDATE these tables. Add to `01-init.sql` or a dedicated migration.
3. **DB-level: BEFORE UPDATE trigger (optional, follow-up)** — trigger raises an exception if any column is changed on an append-only table. Most defensive, but adds execution overhead per row.

**Recommended v1**: app-layer discipline only. Add a comment on each append-only table's primary key: `-- APPEND-ONLY: no UPDATE statements allowed in app code`. DB-level enforcement is a follow-up change if the user wants it.

**The `organization.is_active` exception**: this column IS updatable. App-layer discipline allows UPDATE only on `is_active`. A trigger or a partial REVOKE is too coarse for v1.

## Migration file layout recommendation

**Recommended: Option B (one migration per bounded context) — 3 files**

**Rationale**:
- **A (one per table, 7 files)** is too granular. The 7 tables have hard FK dependencies; splitting them into 7 files makes the dependency order less obvious and the rollback story harder to follow.
- **B (3 files)** groups tables by bounded context:
  1. `20260622_orgs_and_projects.sql` — `organization`, `project` (2 tables, 0 circular deps)
  2. `20260622_requirements_and_milestones.sql` — `requirement`, `requirement_spike`, `milestone` (3 tables, all inherit from `project` or `requirement`)
  3. `20260622_tasks_and_specs.sql` — `task`, `spec` (2 tables, inherit from `milestone` and `task` respectively)
  - Aligns with the user's "bounded context" framing. 3 PRs to review if chained, 1 PR to review if single.
- **C (1 file, 7 tables)** is fine for atomicity, but a 200+ line migration file is hard to review. Goose handles it fine; humans do not.

**Dependency order** (enforced by file order + FK references):
1. `organization` (no parents)
2. `project` → `organization`
3. `requirement` → `project`
4. `requirement_spike` → `requirement`
5. `milestone` → `requirement`
6. `task` → `milestone`
7. `spec` → `task`

## Test surface plan

**New integration tests** in `backend/database_administrator/src/migration/runner_test.go` (gated on `INTEGRATION=1`):

1. **All tables apply cleanly** — `TestRunner_Up_AllNewMigrations` (line-anchored to the existing `TestRunner_Up_FirstBoot` at line 128). After apply, verify:
   - `public.schema_migrations` has 4 rows (1 hello + 3 new) in chronological order.
   - Each new table exists in `\d` with the expected columns and constraints.
2. **FK enforcement** — `TestRunner_Up_FKConstraintsApply`. `INSERT INTO project (organization_id, ...) VALUES (99999, ...)` should fail with FK violation.
3. **Single-PK-as-FK pattern** — `TestRunner_Up_ChildTableHasNoExtraPK`. Verify that `requirement.project_id` is the only PK column (no `id` column).
4. **Append-only enforcement (app-layer)** — `TestRunner_Up_AppendOnlyConvention`. Verify the convention is documented via column comments. (DB-level enforcement is a follow-up.)
5. **`created_at = updated_at` on insert** — `TestRunner_Up_TimestampsDefaultEqual`. Insert a row, assert `created_at = updated_at`.
6. **Lexicographic order** — extend the existing `TestRunner_Up_LexicographicOrder` (line 217) to include the 3 new files.

**Unit tests** in `application/migration_service_test.go`: no change needed; the service is migration-agnostic.

**Driver tests** in `migration/postgres/driver_test.go`: no change needed.

**PR size forecast**: 3 new SQL files, ~80-120 lines each (DDL with column comments + CHECK constraints). 6 new integration tests, ~200 lines. **Total: ~600-800 lines of new code in `src/`.** This is borderline for single-PR but acceptable. If the user prefers chained PRs, the file-per-bounded-context layout supports 3 chained PRs naturally.

## Risks

1. **Single-PK-as-FK conflicts with true 1:N semantics.** The user's inheritance convention naturally enforces 1:1. If a project truly needs N requirements, N milestones, N tasks, N specs (and not 1:1 with the parent), the convention needs to be re-evaluated. **Mitigation**: confirm cardinality questions A1, A3, A4 with the user before specs.
2. **`created_at = updated_at` on insert relies on app discipline.** A future contributor might write `UPDATE table SET updated_at = now()` in app code without realizing it overrides the convention. **Mitigation**: column comment + a unit test that asserts default values are equal.
3. **Phase enumeration: 8 vs 9 values.** The user listed 9 phase names but I count 8 distinct. If I'm wrong, the CHECK constraint will reject the 9th. **Mitigation**: confirm with user.
4. **256 KiB PRD cap is a soft business limit, not enforced by the DB.** `octet_length(content) <= 262144` enforces it at the DB level, but the user's "indicate in the columns comments" wording suggests it's a documentation requirement, not a hard limit. **Mitigation**: ask the user whether the limit should be enforced by CHECK or just documented in a comment.
5. **OTel upstream bug at SDK v1.44.0 drops ~9/10 spans.** The migration system itself works fine; only observability is affected. Already known. Not a blocker for this change.

## Next recommended phase

`sdd-propose` — formalize the change into a proposal document, locking the 5 open-question answers and the file layout.

## Skill resolution

- `paths-injected`: go-testing, work-unit-commits, chained-pr, branch-pr, cognitive-doc-design (all loaded from `/Users/braejan/.claude/skills/`)
- `fallback-path`: test-driven-development (literal path `/Users/braejan/.claude/skills/test-driven-development/SKILL.md` does not exist on disk; TDD discipline applied from `openspec/AGENTS.md` and the test-driven-development description). Confirmed as a pre-existing drift; matches the discovery logged in session #1634.

## Review checklist

- [ ] reviewer can confirm 5 open questions each have a recommended answer and a confirmation question
- [ ] reviewer can confirm column budgets match the user's verbatim requirements (no extra columns added)
- [ ] reviewer can confirm migration file layout respects the goose v3 single-file convention and the project README §1 regex
- [ ] reviewer can confirm dependency order is correct (orgs → projects → requirements → spikes/milestones → tasks → specs)
- [ ] reviewer can confirm append-only strategy starts with app-layer discipline and defers DB-level enforcement
- [ ] reviewer can confirm the 8-vs-9 phase mismatch is surfaced for user confirmation
- [ ] reviewer can confirm the 256 KiB PRD cap is either enforced by CHECK or documented in a comment (per user preference)
- [ ] reviewer can confirm the test plan covers FK, single-PK-as-FK, timestamps, and ordering
