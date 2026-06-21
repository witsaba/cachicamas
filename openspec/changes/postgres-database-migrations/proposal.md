# Proposal — postgres-database-migrations

> **Change**: `postgres-database-migrations`
> **Status**: proposed
> **Date**: 2026-06-21
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Base branch**: `feat/add-postgres-database-migrations` (= `main` @ `7939de0`)
> **Persistence**: hybrid (Engram `sdd/postgres-database-migrations/proposal` + filesystem)
> **Depends on**: `sdd/postgres-database-migrations/explore` (id 1584)
> **User decisions baked in**: id 1585 (Q1, Q2, hello-world scope, infra/ scoped exception)

---

## Summary

Wire a versioned, idempotent SQL migration runner into `database_administrator` so the `cachicamas` platform can evolve its Postgres schema safely. The runner uses `pressly/goose` v3 embedded into the binary via `embed.FS`, talks to Postgres through `jackc/pgx/v5`, and runs on every container start under a non-blocking advisory lock. As the first concrete deliverable we pin the database timezone to `UTC` in `infra/postgres/init/01-init.sql` (one-shot, superuser context) and ship a no-op `SELECT 1;` migration to prove the runner is wired end-to-end.

## Problem / motivation

`cachicamas` is preparing to land bounded-context schemas (`catalog`, `identity`, `observability`) and OTel-backed audit tables. Today, the only way to change the database is by hand: open `psql`, run a snippet, hope someone remembered to add it to a wiki page. The empty `public.schema_migrations` table in `01-init.sql` was already provisioned for "a future migration runner" — that runner does not exist yet, so every schema change is one-off and un-audited.

We also have an unrelated gap: the database timezone is implicit. The container happens to default to UTC because `postgres:18-alpine3.24` sets `TZ=UTC`, but nothing pins it. A future operator changing `TZ`, a custom `postgresql.conf`, or a non-Alpine base image would silently shift every `TIMESTAMPTZ` arithmetic and `now()` call. The motivation to fix this **now** is cheap: we are already in this file.

Doing both at once lets us validate the runner with a real, observable side-effect (a verifiable `pg_timezone_names` or `SHOW timezone` change) rather than a "did anything happen?" smoke test.

## Goals

- **G1 — Versioned migrations**: a `goose.Up` call embedded in the binary, driven by `//go:embed sql/*.sql`, with timestamp-prefixed filenames.
- **G2 — Database-level UTC**: every new session in `cachicamas_pg` uses `UTC`, persisted across restarts, applied during `initdb` so it is correct on the first ever container boot.
- **G3 — Idempotency**: re-running the runner is a no-op; the bookkeeping table records what was applied.
- **G4 — Concurrency-safe**: two `database_administrator` replicas starting in parallel cannot both apply migrations (non-blocking `pg_try_advisory_lock` + heartbeat).
- **G5 — Hexagonal fit**: the runner lives behind a `domain` port and an `application` use case, called from `cmd/server/main.go` before Echo binds.
- **G6 — OTel + slog visibility**: every `goose.Up` call is wrapped in a span (`migration.up`) and an `slog` line with `version`, `applied`, `duration_ms`.
- **G7 — Fail-fast**: a migration error crashes the container. Silent 500s are worse than a loud crash.
- **G8 — Strict TDD**: the runner has a failing test written first, then the production code, then refactor.

**Success criteria**:

- `make test` passes (unit tests, no DB required).
- `make test/integration` (new target) passes against a real Postgres provisioned by docker-compose.
- `goose -dir backend/database_administrator/src/migration/sql postgres "..." status` reports `OK` with the hello-world version applied.
- `psql -c "SHOW timezone"` from inside the `cachicamas-postgres` container returns `UTC` after a fresh `docker compose down -v && up`.
- A second `docker compose up` (no volume wipe) is a no-op; `goose status` still reports `OK`.

## Non-goals

- Bounded-context DDL (catalog tables, identity tables, observability tables). The `catalog`, `identity`, `observability` schemas remain empty placeholders.
- Online schema-change choreography (`CREATE INDEX CONCURRENTLY`, `pg_repack`, `pgroll`).
- Multi-database fan-out (one runner, one DB).
- DDL linting / drift detection in CI (`sqlfluff`, `atlas schema diff`).
- Backups, WAL archiving, logical replication.
- Reverse-engineering the current schema into a baseline migration.
- CI integration-test wiring beyond a `make test/integration` target.

## Approach

### Library choice — `pressly/goose` v3.27.1 (MIT)

| Library | Verdict | Why |
|---|---|---|
| **`pressly/goose` v3.27.1** (2026-04-24) | **CHOSEN** | MIT, `embed.FS` via `goose.SetBaseFS`, programmatic API, opt-in `pg_try_advisory_lock` (non-blocking, retries, no timeout death), pure library — no framework imports, hexagonal-clean. |
| `golang-migrate` v4.19.1 (2025-11-29) | rejected | `pg_advisory_lock` is session-level and **blocks indefinitely with no timeout** — disqualifying for container orchestrators that restart on healthcheck failure. |
| `ariga/atlas` v1.2.0 CLI / archived `atlas-go-sdk` v0.7.2 | rejected | Designed for declarative diff-and-apply (overkill for an imperative `Up()`); `atlasexec` shells out to a CLI binary in the container; dual-licensed (EULA risk for default binaries). |
| `gorm.io/gorm` AutoMigrate | rejected | Not a migration tool — best-effort sync to struct tags, no audit trail, no `embed.FS`, no advisory lock, no down. Architectural mismatch with the `sqlx`-free / Echo-only stack. |
| Hand-rolled `sqlx` + custom runner | rejected | Reinvents locking + bookkeeping in ~500 LOC. Violates DRY. |
| `riverqueue/river` + `rivermigrate` | rejected | Ties migrations to a job-queue worker pool we don't run. |

**Library version claims verified** via the explore artifact (id 1584) against GitHub releases on 2026-06-21. `jackc/pgx` v5 is the **latest** line at the time of writing; an exact patch will be resolved in design (ADR required — see "Open questions deferred to design").

### Driver — `github.com/jackc/pgx/v5` (MIT, stdlib `database/sql` adapter)

User-decision baked in (id 1585). ADR will be written at design time, naming the exact patch and confirming the `stdlib` adapter choice (so the `database/sql` API used by `goose.SetDialect("postgres")` works without forking goose).

### File layout

```
backend/database_administrator/
├── go.mod                              # MODIFIED: +pressly/goose/v3, +jackc/pgx/v5
├── go.sum                              # MODIFIED
├── Makefile                            # MODIFIED: +test/integration, +migrate/new
└── src/
    ├── cmd/server/main.go              # MODIFIED: pre-Echo migration hook
    ├── application/migration_service.go        # NEW: use case (Up, Status)
    ├── domain/migration.go                     # NEW: port (Runner interface)
    ├── migration/                              # NEW: adapter slice
    │   ├── README.md                           # NEW: how to add a migration
    │   ├── embed.go                            # NEW: //go:embed sql/*.sql
    │   ├── runner.go                           # NEW: goose.Up wrapper (OTel + slog)
    │   ├── runner_test.go                      # NEW: TDD-first integration test
    │   ├── postgres/driver.go                  # NEW: *sql.DB factory
    │   ├── postgres/driver_test.go            # NEW: DSN parse cases
    │   └── sql/
    │       ├── 20260621120000_hello_world.sql  # NEW: body = SELECT 1;
    │       └── 20260621120000_hello_world.down.sql  # NEW: SELECT 1; (no-op)
    └── (unchanged: application/health_service.go, domain/health.go,
                interfaces/http/, otel/)
infra/postgres/init/01-init.sql          # MODIFIED (scoped exception — see below)
openspec/changes/postgres-database-migrations/
├── explore.md                          # (already exists)
├── proposal.md                         # (this file)
├── specs/db-migrations/spec.md         # NEW in sdd-spec phase
├── design.md                           # NEW in sdd-design phase
└── tasks.md                            # NEW in sdd-tasks phase
```

### Runner timing — every container start (recommended)

The runner fires from `cmd/server/main.go` **before** Echo binds the listener, with a bounded context (5 s connect timeout via `pgx.Config.ConnConfig.ConnectTimeout`). The bookkeeping table makes this a no-op on every restart after the first; it is not a "first-boot" pattern. A separate init container is rejected for v1 — one less moving part.

### Failure behavior — crash on migration error (recommended)

A failed migration returns the error from `goose.Up`; `main.go` calls `slog.Error` and `os.Exit(1)`. The container orchestrator restarts the container; until the operator fixes the SQL, the service stays down. This is fail-fast: surface the problem in `docker compose logs`, do not silently serve 500s from a stale schema.

### Bookkeeping table — `public.schema_migrations` (override goose default)

`01-init.sql` already provisions `public.schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now(), description TEXT)`. The proposal **reuses** it by calling `goose.SetTableName("schema_migrations")` in `runner.go`. goose stores its `version_id` as a stringified bigint; the `TEXT` column is compatible. **Design-phase open question** (see below): whether to also `ALTER TABLE ... ALTER COLUMN version TYPE BIGINT` in `01-init.sql` for type safety, or accept the implicit string-to-bigint cast. The proposal **does not** make this change unilaterally — it stays a design decision.

> **Note for reviewers**: the `COMMENT ON TABLE public.schema_migrations` in `01-init.sql` (line 105) still reads accurately after this change: it is still the append-only log of applied migrations; the comment does not need to change.

### First migration — no-op `SELECT 1;` (user-decision baked in)

Filename: `20260621120000_hello_world.sql`. Body: `SELECT 1;`. The file header honestly documents that this is a "wiring" migration — the value is proving the runner, not the SQL. Filename uses timestamp prefix (goose default) to avoid merge conflicts on future parallel PRs.

A matching `20260621120000_hello_world.down.sql` is shipped so `goose down` does not error if a developer runs it locally; both up and down are `SELECT 1;`.

### The UTC move — `infra/postgres/init/01-init.sql`

This is the only user-facing schema change in this proposal. The exact lines to add to `01-init.sql` (verified in the explore artifact, section 3):

```sql
-- ---------------------------------------------------------------------------
-- Database ownership + UTC default — single source of truth for DB-level
-- timezone. Runs as the cluster superuser (POSTGRES_USER) during initdb.
--
-- Why in init.sql, not a goose migration:
--   • init.sql runs in the superuser context; a goose migration runs as
--     queen (NOSUPERUSER). ALTER DATABASE ... SET timezone is settable by
--     the database owner, so queen CAN run it — but only AFTER she owns
--     the database. Reassigning ownership in a goose migration would be a
--     chicken-and-egg: the migration cannot run until the DB is owned by
--     queen, but queen cannot own the DB until a superuser grants it.
--   • init.sql is the natural place: one file, runs once, and lives in
--     `infra/` (the source of truth for cluster bootstrap).
--
-- Idempotency: ALTER DATABASE ... OWNER TO is idempotent when the target
-- owner is already the owner (no-op). ALTER DATABASE ... SET is idempotent
-- in the value-equal sense (PG treats setting the same value as a no-op).
-- ---------------------------------------------------------------------------
ALTER DATABASE current_database() OWNER TO queen;
ALTER DATABASE current_database() SET timezone = 'UTC';
```

These lines go **after** the `queen` role is created (line 75) and **before** the `CREATE TABLE public.schema_migrations` block (line 99), so the table is created by the new owner (queen), which matches the migration runner's identity.

**Scoped exception to the SDD hard rule "do not modify `infra/`"**: granted for **this conversation only** (a single Claude Code session) and **only for `infra/postgres/init/01-init.sql`**. The exception is **one-shot**; it does not extend to `docker-compose.yaml`, `infra/otel/`, `infra/jaeger/`, or any other file. Authority: user decision id 1585. Reviewers can see the scope of the exception by reading this section; the proposal does not grant itself broader authority.

## Affected areas

| Area | Impact | Why |
|---|---|---|
| `backend/database_administrator/src/cmd/server/main.go` | MODIFIED | Add pre-Echo migration hook (bounded context, fail-fast on error). |
| `backend/database_administrator/src/application/migration_service.go` | NEW | Use case: `Up(ctx) error`, `Status(ctx) ([]Version, error)`. |
| `backend/database_administrator/src/domain/migration.go` | NEW | Port: `Runner` interface (no goose import). |
| `backend/database_administrator/src/migration/runner.go` | NEW | Programmatic `goose.Up` wrapper with OTel span + slog line. |
| `backend/database_administrator/src/migration/embed.go` | NEW | `//go:embed sql/*.sql` → `embed.FS`. |
| `backend/database_administrator/src/migration/runner_test.go` | NEW | TDD-first: integration test against real Postgres (testcontainers or compose). |
| `backend/database_administrator/src/migration/postgres/driver.go` | NEW | `*sql.DB` factory from `DATABASE_URL` env (pgx stdlib adapter). |
| `backend/database_administrator/src/migration/postgres/driver_test.go` | NEW | Table-driven DSN parse cases. |
| `backend/database_administrator/src/migration/sql/20260621120000_hello_world.sql` | NEW | Body: `SELECT 1;`. |
| `backend/database_administrator/src/migration/sql/20260621120000_hello_world.down.sql` | NEW | Body: `SELECT 1;` (no-op down). |
| `backend/database_administrator/src/migration/README.md` | NEW | How to add a migration, how to recover from a half-applied one, naming convention. |
| `backend/database_administrator/go.mod` | MODIFIED | `+github.com/pressly/goose/v3 v3.27.1`, `+github.com/jackc/pgx/v5`. Both require ADRs (see "Open questions deferred to design"). |
| `backend/database_administrator/Makefile` | MODIFIED | `+test/integration` (boots compose Postgres, waits for `pg_isready`); `+migrate/new NAME=...` (runs `goose create`). |
| `infra/postgres/init/01-init.sql` | MODIFIED | +2 lines (`OWNER TO queen`, `SET timezone='UTC'`). **Scoped exception in effect.** |
| `openspec/specs/db-migrations/spec.md` | NEW (sdd-spec) | Hello-world capability spec (Given/When/Then). |
| `openspec/changes/postgres-database-migrations/design.md` | NEW (sdd-design) | Sequence diagrams for boot, failure modes, recovery. |
| `openspec/changes/postgres-database-migrations/tasks.md` | NEW (sdd-tasks) | Grouped by phase, hierarchical numbering, PR-sized. |

**Not touched**: `docker-compose.yaml`, `infra/otel/`, `infra/jaeger/`, the existing `openspec/changes/cachicamas-tail-sampling/`, `openspec/changes/cachicamas-deep-healthcheck/`.

## Risks and mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| `goose` releases a breaking change before we ship v1.0 of the runner. | Low | Pin to `v3.27.1` exactly. Bump via a deliberate PR + `go.sum` refresh. |
| `embed.FS` doesn't include a newly-added migration file in the built binary. | Medium | Add `make migrate/lint` (design phase) that opens the embed.FS and asserts every filename matches `^\d{14}_[a-z0-9_]+\.sql$`. |
| `queen` loses ownership of `cachicamas_pg` (e.g., a future change to `POSTGRES_USER` causes a re-init). | Low | Startup precondition in `runner.go`: query `pg_database` and fail-fast if `datdba` is not queen's `oid`. |
| A migration locks the DB for too long (`ALTER TABLE ADD COLUMN ... DEFAULT` on a large table). | Medium (future) | Document in `migration/README.md` that all schema migrations use Postgres-native online ops. Pre-commit hook (out of v1) to flag risky patterns. |
| A `database_administrator` replica starts while another is mid-migration. | Medium (future) | `goose.WithSessionLocker(...)` is **always on** in `runner.go` (not opt-in). `pg_try_advisory_lock` is non-blocking; the second replica logs "waiting for lock" and retries. |
| Tests assume a running Postgres but CI doesn't have one. | Medium | Integration tests gate on `t.Skip("integration")` when `INTEGRATION=1` env var isn't set. `make test` stays integration-free. |
| `01-init.sql` has a typo in the two new lines. | Low | **One-shot**: a typo only surfaces on the first-ever container boot (or after `docker compose down -v`). After the volume is populated, the new lines do NOT re-run. Mitigation: PR review + `make lint` (SQL is not linted today — out of scope; design may propose `sqlfluff` later). |
| The `public.schema_migrations.version TEXT` mismatch with goose's default BIGINT causes a runtime cast error. | Low | Verified: goose stores `version_id` as a string. The `TEXT` column accepts it. Design phase decides whether to ALTER to BIGINT in `01-init.sql` for type safety. |
| The scoped infra exception is "abused" to also touch `docker-compose.yaml` or other infra files. | Low | The proposal explicitly enumerates "Not touched" files. Any apply-phase change to other infra files requires a new grant from the user. |
| `goose fix` recovery procedure is unclear. | Low | Document in `migration/README.md`: "if a migration's transaction rolled back, edit the bookkeeping table manually — goose does not use a `dirty` column". |

## Rollback plan

Per `openspec/config.yaml` rule "Include rollback plan for every change":

1. **Revert the goose code**:
   - Remove `backend/database_administrator/src/migration/` (entire directory).
   - Revert the pre-Echo migration hook in `src/cmd/server/main.go`.
   - Run `go mod tidy` to drop `pressly/goose/v3` and `jackc/pgx/v5` from `go.mod` / `go.sum`.
   - Revert the Makefile (`test/integration`, `migrate/new`).
   - Result: the binary boots Echo with no migration step. The `public.schema_migrations` table remains (init.sql still provisions it) and is harmless.

2. **Revert the `01-init.sql` change**: requires **dropping the `cachicamas-postgres-data` named volume** and recreating the stack. init.sql is a one-shot — it does NOT re-run on a populated volume. Procedure:
   ```bash
   docker compose down -v          # drops the volume
   git revert <commit>             # reverts the init.sql change
   docker compose up -d postgres   # init.sql runs again, WITHOUT the new lines
   docker compose up -d database_administrator
   ```
   **Caveat**: dropping the volume also drops all data (catalog, identity, observability). Do not run this rollback in any non-dev environment.

3. **Revert the driver change** (if it ever ships separately): revert `go.mod`, swap the import path in `migration/postgres/driver.go` from `jackc/pgx/v5/stdlib` back to `database/sql`'s default registration.

4. **Belt-and-suspenders**: the `01-init.sql` change adds `OWNER TO queen` and `SET timezone='UTC'` independently. If only the `SET timezone` is wrong, an operator can `ALTER DATABASE cachicamas_pg RESET timezone;` from a superuser session — no rollback needed.

## Open questions deferred to design phase

These are the four open questions from explore §7 that were **not** resolved in user-decision id 1585. The design phase MUST decide them; the proposal surfaces them as recommendations so the user can object before design locks them in.

| # | Question | Recommendation | Authority |
|---|---|---|---|
| Q3 | **Runner timing** — every container start vs one-shot init container? | **Every container start.** Idempotent via the bookkeeping table; one fewer container to manage. The "slow if 100+ migrations" concern is moot today (one migration). | explore §7, recommendation row 3 |
| Q4 | **Failure behavior** — crash vs serve 500s? | **Crash on migration error.** Fail-fast > silent corruption. `main.go` returns the error, `os.Exit(1)`. | explore §7, recommendation row 4 |
| Q5 | **Down migrations** — required or optional? | **Write for schema-only changes; comment for data moves.** Default for hello world: ship a `SELECT 1;` down (it costs nothing, keeps `goose down` working locally). | explore §7, recommendation row 5 |
| Q6 | **Migration dir layout** — one global dir or per-context? | **One global dir** (`src/migration/sql/`), schema-qualified names (e.g., `20260622_catalog_create_products.sql`). Simpler tooling, easier CI ordering. | explore §7, recommendation row 6 |

**Additional design-phase questions raised by this proposal** (not in the explore):

| # | Question | Recommendation |
|---|---|---|
| Q7 | **Bookkeeping column type** — keep `version TEXT` or ALTER to `BIGINT`? | Keep `TEXT` in `01-init.sql` (don't widen the change); the design phase confirms goose's string-cast is safe. If a follow-up wants BIGINT, it goes in a future `01-init.sql` change. |
| Q8 | **OTel span name and attributes** — what does `migration.up` carry? | Span attrs: `db.system=postgresql`, `migration.dir=sql`, `migration.applied_count` (int), `migration.duration_ms` (int), `migration.error` (string, on failure). Documented in design. |
| Q9 | **Postgres driver ADR** — exact `pgx/v5` patch and stdlib adapter justification. | Required by `openspec/config.yaml` rule "No new top-level deps without an ADR". Saved to Engram `adr/pgx-v5-stdlib-adapter`. |
| Q10 | **Goose ADR** — pin rationale, upgrade policy, alternative considered. | Required. Saved to Engram `adr/goose-v3-embed-fs`. |

## Out of scope (explicit, per `openspec/config.yaml`)

These are "deferred but related" — they are not in this change and must be raised explicitly so reviewers do not assume they are implicit:

1. **Least-privilege app role** (`cachicamas_app` recipe in `01-init.sql` lines 109-121). Out.
2. **Backups / WAL archiving / logical replication**. Out.
3. **Bounded-context DDL** beyond the empty schemas `01-init.sql` already provisioned. Out.
4. **DDL linting / drift detection** (`sqlfluff`, `atlas schema diff` in CI). Out.
5. **Online schema changes** (`pg_repack`, `pgroll`, `CREATE INDEX CONCURRENTLY` choreography). Out.
6. **Multi-database migrations** (one runner, one DB). Out.
7. **CI integration tests** beyond a `make test/integration` target. Out of v1.
8. **Reverse-engineering current DB state** into a baseline migration. Out — we start from an empty `schema_migrations` table.
9. **Any change to `docker-compose.yaml`**. Out — including the database service's environment, healthcheck, or volume.
10. **Any change to `infra/otel/`, `infra/jaeger/`, or any other `infra/` file**. Out — only `01-init.sql` is touched, under the scoped exception.

## Review checklist

- [ ] reviewer can confirm this proposal does NOT modify any file under `backend/database_administrator/src/` (proposal is markdown only)
- [ ] reviewer can confirm this proposal does NOT touch `docker-compose.yaml`, `infra/otel/`, `infra/jaeger/`, or any file in `infra/postgres/init/` other than `01-init.sql`
- [ ] reviewer can confirm the scoped infra exception is explicitly authorized by user decision id 1585 and is one-shot
- [ ] reviewer can confirm the four resolved user decisions from id 1585 (UTC → init.sql, pgx/v5, hello-world scope, infra exception) are baked in
- [ ] reviewer can confirm `pressly/goose` v3.27.1 is named with a version verified in explore id 1584 (2026-04-24)
- [ ] reviewer can confirm the rejected-alternatives table matches the explore matrix and cites disqualifying findings (golang-migrate's blocking lock, atlas's EULA, gorm's no-embed, etc.)
- [ ] reviewer can confirm the `public.schema_migrations` reuse decision is documented with the TEXT-vs-BIGINT type concern
- [ ] reviewer can confirm the four open questions deferred to design (Q3, Q4, Q5, Q6) carry explicit recommendations
- [ ] reviewer can confirm the rollback plan covers the one-shot nature of `01-init.sql` (volume drop required)
- [ ] reviewer can confirm the out-of-scope list is explicit and matches `openspec/config.yaml` rules
- [ ] reviewer can confirm the risks table covers embed.FS staleness, DB ownership loss, advisory lock behavior, replica concurrency, and the new UTC-move risks
- [ ] reviewer can confirm ADRs are flagged for both new top-level deps (goose, pgx) per `openspec/config.yaml`

## Incompleteness log entry

- [2026-06-21] [openspec/specs/] missing — directory is empty; the first change to land a spec seeds it (likely `cachicamas-tail-sampling`, in flight). This change proposes to add `openspec/specs/db-migrations/spec.md` in the spec phase. **Proposed fix**: track in `sdd-spec`; do not pre-create.

- [2026-06-21] [openspec/changes/postgres-database-migrations/] `design.md` and `tasks.md` missing — proposal phase produces proposal.md only; design and tasks land in their respective phases. **Proposed fix**: track in `sdd-design` and `sdd-tasks`.

- [2026-06-21] [openspec/changes/postgres-database-migrations/] ADRs for `pressly/goose` and `jackc/pgx` missing — `openspec/config.yaml` requires an ADR for any new top-level dep. **Proposed fix**: write both ADRs in the design phase, save to Engram `adr/goose-v3-embed-fs` and `adr/pgx-v5-stdlib-adapter`, reference from `design.md`.
