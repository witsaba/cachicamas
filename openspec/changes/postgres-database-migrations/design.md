# Design — postgres-database-migrations

> **Change**: `postgres-database-migrations`
> **Status**: designed
> **Date**: 2026-06-21
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Base branch**: `feat/add-postgres-database-migrations` (= `main` @ `7939de0`)
> **Persistence**: hybrid (Engram `sdd/postgres-database-migrations/design` + filesystem)
> **Depends on**: `sdd/postgres-database-migrations/proposal` (id 1586), `sdd/postgres-database-migrations/spec` (id 1587)
> **ADRs**: `adr/pgx-v5-stdlib-adapter`, `adr/goose-v3-embed-fs` (saved to Engram in this phase)

---

## 1. Overview

The `database_administrator` service embeds SQL migrations at build time via `//go:embed`, applies them on container start using `pressly/goose v3.27.1` through a `jackc/pgx/v5` driver, and the database timezone is pinned to `UTC` in `infra/postgres/init/01-init.sql` (one-shot, superuser context). All migration code lives behind a `domain.Runner` port and an `application.migrationService` use case, called from `cmd/server/main.go` before Echo binds.

### Boot sequence (ASCII for portability)

```
docker compose up
   |
   +--> postgres container
   |      initdb -> /docker-entrypoint-initdb.d/01-init.sql
   |         +-- CREATE EXTENSION ...
   |         +-- CREATE ROLE queen (NOSUPERUSER, CREATEROLE, CREATEDB, REPLICATION)
   |         +-- ALTER ROLE queen WITH CREATEROLE CREATEDB REPLICATION
   |         +-- CREATE SCHEMA catalog/identity/observability
   |         +-- ALTER DATABASE current_database() OWNER TO queen        <-- NEW (this change)
   |         +-- ALTER DATABASE current_database() SET timezone='UTC'   <-- NEW (this change)
   |         +-- CREATE TABLE public.schema_migrations (...)
   |
   +--> database_administrator container
          cmd/server/main.go
             |
             +-- otel.Init() + slog setup
             |
             +-- migrationService.Up(ctx)         <-- NEW (this change)
             |      |
             |      +-- driver.Open(DATABASE_URL or POSTGRES_* env)
             |      |     <-- jackc/pgx/v5 stdlib adapter
             |      |
             |      +-- goose.NewProvider(DialectPostgres, db, fs.Sub(MigrationsFS, "sql"),
             |      |                       WithSessionLocker(lock.NewPostgresSessionLocker()),
             |      |                       WithTableName("schema_migrations"))
             |      |
             |      +-- provider.Up(ctx)
             |             +-- pg_try_advisory_lock(42)
             |             +-- SELECT MAX(version_id) FROM schema_migrations -> 0
             |             +-- INSERT INTO schema_migrations VALUES (20260621120000, ...)
             |             +-- exec `20260621120000_hello_world.sql` (body: SELECT 1;)
             |             +-- pg_advisory_unlock(42)
             |
             +-- echo.New() + register routes
             |
             +-- echo.Start(":" + SERVICE_PORT)
```

## 2. Component map

| File | Package | Imports | Purpose |
|---|---|---|---|
| `domain/migration.go` | domain | (none) | Port: `Runner` interface (no goose import) |
| `application/migration_service.go` | application | domain, slog, otel | Use case: `Up(ctx) error`, `Status(ctx) ([]Version, error)` |
| `migration/embed.go` | migration | embed | `//go:embed sql/*.sql` |
| `migration/runner.go` | migration | domain (port only), goose, otel, slog | Programmatic goose.Up wrapper |
| `migration/postgres/driver.go` | migration/postgres | stdlib pgx, slog | `*sql.DB` factory from env |
| `migration/sql/*.sql` | n/a (data) | n/a | Migration files (embedded) |
| `cmd/server/main.go` | main | application, migration, otel, echo | Pre-Echo migration hook |

**Hexagonal rule**: only `migration/runner.go` imports `goose`. Only `migration/postgres/driver.go` imports `pgx`. The `domain` package imports nothing from this slice. The `application` package imports `domain` only.

## 3. Hexagonal slice

### Port (domain/migration.go)

```go
package domain

import "context"

type Version struct {
    ID          int64
    Description string
    AppliedAt   time.Time
}

type Runner interface {
    Up(ctx context.Context) (applied []Version, err error)
    Status(ctx context.Context) ([]Version, error)
}
```

### Use case (application/migration_service.go)

```go
package application

type MigrationService struct {
    runner domain.Runner
    logger *slog.Logger
    tracer trace.Tracer
}

func NewMigrationService(r domain.Runner, l *slog.Logger, t trace.Tracer) *MigrationService

func (s *MigrationService) Up(ctx context.Context) error
    // wraps runner.Up in OTel span "migration.up", logs result, returns error

func (s *MigrationService) Status(ctx context.Context) ([]domain.Version, error)
```

### Adapter (migration/runner.go)

```go
package migration

import (
    "embed"
    "github.com/pressly/goose/v3"
    "github.com/pressly/goose/v3/lock"
    "io/fs"
    "database/sql"
)

//go:embed sql/*.sql
var migrationsFS embed.FS

type GooseRunner struct {
    db        *sql.DB
    tableName string
    logger    *slog.Logger
}

func NewGooseRunner(db *sql.DB, tableName string, l *slog.Logger) *GooseRunner

func (r *GooseRunner) Up(ctx context.Context) ([]domain.Version, error)
    // goose.NewProvider(goose.DialectPostgres, r.db, fs.Sub(MigrationsFS, "sql"),
    //     goose.WithSessionLocker(lock.NewPostgresSessionLocker()),
    //     goose.WithTableName(r.tableName))
    // provider.Up(ctx)

func (r *GooseRunner) Status(ctx context.Context) ([]domain.Version, error)
    // goose.GetMigrations ...
```

## 4. Configuration (Before / After)

| Env var | Default | Read in | Before | After |
|---|---|---|---|---|
| `POSTGRES_HOST` | `postgres` | `migration/postgres/driver.go` | (existed) | unchanged |
| `POSTGRES_PORT` | `5432` | `migration/postgres/driver.go` | (existed) | unchanged |
| `POSTGRES_DB` | `cachicamas_pg` | `migration/postgres/driver.go` | (existed) | unchanged |
| `POSTGRES_USER` | `queen` | `migration/postgres/driver.go` | (existed) | unchanged |
| `POSTGRES_PASSWORD` | (required) | `migration/postgres/driver.go` | (existed) | unchanged |
| `MIGRATION_TIMEOUT` | `30s` | `application/migration_service.go` | n/a | **NEW**: max time for the entire Up call |
| `MIGRATION_TABLE` | `schema_migrations` | `migration/runner.go` | n/a | **NEW**: overrides goose's `goose_db_version` |
| `MIGRATION_DIR` | `sql` | `migration/runner.go` | n/a | **NEW**: subdir inside embed.FS |
| `INTEGRATION` | (unset) | `*_test.go` | n/a | **NEW**: when set to `1`, integration tests run |

**Precedence (locked)**: `DATABASE_URL` takes precedence over `POSTGRES_*` if both are set. If neither is set, the service fails to start (fail-fast).

## 5. Q3 — Runner timing: every container start (LOCKED)

The runner fires from `cmd/server/main.go` **before** Echo binds the listener, with a bounded context (`MIGRATION_TIMEOUT`, default 30s). The bookkeeping table makes this a no-op on every restart after the first. A separate init container is rejected for v1.

**Code path in main.go** (pseudocode):
```go
func main() {
    ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer cancel()

    otel.Init(ctx) // existing
    logger := slog.New(...) // existing

    // NEW: pre-Echo migration hook
    migrationService := application.NewMigrationService(runner, logger, otel.Tracer("database_administrator"))
    if err := migrationService.Up(ctx); err != nil {
        logger.Error("migration.up failed; exiting", "error", err)
        os.Exit(1)
    }

    // existing: Echo bind + serve
    e := echo.New()
    e.GET("/health", healthHandler)
    e.Start(":" + os.Getenv("SERVICE_PORT"))
}
```

## 6. Q4 — Failure behavior: crash on migration error (LOCKED)

A failed migration returns the error from `goose.Up`; `main.go` calls `slog.Error` and `os.Exit(1)`. The container orchestrator restarts the container. This is fail-fast.

**Why not serve 500s**: silent corruption is worse than a loud crash. The DB schema is the contract that the application code depends on. If the contract is broken, the application is wrong by definition.

## 7. Q5 — Down migrations: write for schema-only, comment for data moves (LOCKED)

goose supports both up and down. **v3 idiom**: ONE file per migration with both `-- +goose Up` and `-- +goose Down` blocks (separated by `-- +goose StatementEnd` + `-- +goose StatementBegin`). The legacy v2 paired `XXX.sql` + `XXX.down.sql` is rejected by v3 with `found duplicate migration version` — see §9.1. Default convention:
- **Schema-only changes** (CREATE TABLE, ALTER TABLE ADD COLUMN): always include a `-- +goose Down` block that reverses the change.
- **Data moves** (UPDATE that backfills, DELETE that purges): ship a `-- +goose Down` block only if the move is logically reversible; otherwise leave a `-- TODO: data migration, down is destructive` comment above the Down block.

The hello-world migration ships a no-op `SELECT 1;` Down block (it costs nothing, keeps `goose down` working locally for tests).

## 8. Q6 — Layout: one global `src/migration/sql/`, schema-qualified names (LOCKED)

All migration files live in `src/migration/sql/`. Filenames are timestamp-prefixed: `YYYYMMDDHHMMSS_<context>_<description>.sql`. Examples:
- `20260621120000_hello_world.sql`
- `20260622_catalog_create_products.sql`
- `20260701_identity_add_users_email_index.sql`

`goose.Up` reads the directory as a flat list; nested subdirectories are not traversed (verified against goose v3.27.1 source).

## 9. Q7 — Bookkeeping table shape: goose v3 schema (LOCKED)

`infra/postgres/init/01-init.sql` provisions `public.schema_migrations` with **goose v3.27.1's expected columns**:

```sql
CREATE TABLE IF NOT EXISTS public.schema_migrations (
    id         BIGSERIAL    PRIMARY KEY,
    version_id BIGINT       NOT NULL,
    is_applied BOOLEAN      NOT NULL,
    tstamp     TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS schema_migrations_version_id_idx
    ON public.schema_migrations(version_id);
```

**Why BIGINT, not TEXT**: goose v3's `INSERT` is `INSERT INTO schema_migrations (version_id, is_applied) VALUES ($1, true)` — the column is typed `bigint` in the v3 source. The pre-existing `(version TEXT PK, applied_at TIMESTAMPTZ, description TEXT)` shape was provisioned by `01-init.sql` before this change, was based on the goose v2 column layout, and is **incompatible** with v3: the first `INSERT` would fail with `column "version_id" of relation "schema_migrations" does not exist`. We rewrite the CREATE TABLE block to match v3.

**Trade-off documented**: no implicit cast, but `01-init.sql` is now the source of truth for the v3 schema. Any future goose upgrade that changes column names must be reflected here in the same change. Mitigation: pin goose to `v3.27.1` exactly; upgrade via a deliberate PR with both `go.sum` refresh and `01-init.sql` schema update.

## 9.1 Goose v3 single-file migration idiom (LOCKED)

Each migration is ONE `.sql` file with both `-- +goose Up` and `-- +goose Down` blocks separated by `-- +goose StatementEnd` + `-- +goose StatementBegin` directives. The legacy goose v2 paired `XXX.sql` + `XXX.down.sql` files are **rejected** by v3 with `found duplicate migration version` because both files share the numeric timestamp prefix. Future migrations must follow the single-file shape.

## 10. Q8 — OTel attributes (LOCKED)

Every `Up` call emits one span with these attributes:

| Attribute | Type | Value | When |
|---|---|---|---|
| `db.system` | string | `"postgresql"` | always |
| `migration.dir` | string | `MIGRATION_DIR` value (default `"sql"`) | always |
| `migration.table` | string | `MIGRATION_TABLE` value (default `"schema_migrations"`) | always |
| `migration.applied_count` | int | number of migrations applied | on success |
| `migration.duration_ms` | int | wall-clock time of the Up call | always |
| `migration.error` | string | error message | on failure |
| `migration.error.kind` | string | `pg_advisory_lock_timeout`, `pg_query_error`, `pg_connect_error`, `embed_error` | on failure (categorized) |

Span name: `migration.up`. Span kind: `internal`. Span status: `Ok` on success, `Error` on failure.

Log line shape:
```
INFO  migration.up applied version=20260621120000 applied_count=1 duration_ms=42
ERROR migration.up failed error="..." kind=pg_query_error duration_ms=15
```

## 11. Q9 — pgx/v5 ADR (full ADR below in §13)

## 12. Q10 — Goose ADR (full ADR below in §14)

## 13. ADR-001: jackc/pgx/v5 as the Postgres driver

### Status
Accepted (2026-06-21)

### Context
We need a Postgres driver for the new migration runner. The runner uses goose, which expects a `*sql.DB`. The two options are:
- `github.com/jackc/pgx/v5` (modern, MIT, very active)
- `github.com/lib/pq` (deprecated since 2024)

### Decision
Use `github.com/jackc/pgx/v5` with its `stdlib` adapter (`github.com/jackc/pgx/v5/stdlib`). The stdlib adapter registers pgx as a `database/sql` driver, so `sql.Open("pgx", dsn)` returns a `*sql.DB` that goose can use without forking goose.

### Alternatives considered
- **`lib/pq`**: deprecated since 2024. Active forks exist (`lib/pq` continues to receive security patches) but the upstream is in maintenance mode. Rejected.
- **Native `pgx` API (`github.com/jackc/pgx/v5/pgxpool`)**: would require forking goose or maintaining a parallel connection pool. Rejected.
- **`pgx/v4`**: prior major version. v5 is the current line. Rejected.

### Consequences
- **Positive**: modern pgx v5 features available via the stdlib adapter (e.g., `Batched`, `CopyFrom`); MIT license; very active maintenance; recommended by the pgx README for `database/sql` users.
- **Negative**: the stdlib adapter has a small performance overhead vs the native pgx API (one extra function call per query). For a migration runner, this is irrelevant.
- **Risk**: pgx v5 could release a breaking change. Mitigated by pinning the exact patch in `go.mod` and bumping via a deliberate PR.

### References
- github.com/jackc/pgx README — "database/sql" section
- github.com/jackc/pgx/v5/stdlib godoc
- github.com/lib/pq — last commit 2024-XX, deprecation notice

## 14. ADR-002: pressly/goose v3 as the migration library

### Status
Accepted (2026-06-21)

### Context
We need a Go migration library. The runner must (a) embed SQL via `embed.FS`, (b) be concurrency-safe (non-blocking advisory lock), (c) work as a pure library (no CLI-only features), (d) run as a non-superuser (`queen`).

### Decision
Use `github.com/pressly/goose v3.27.1` (pinned exactly). License: MIT.

### Alternatives considered
- **`golang-migrate` v4.19.1**: `pg_advisory_lock` is session-level and **blocks indefinitely with no timeout** — disqualifying for container orchestrators.
- **`ariga/atlas` v1.2.0**: designed for declarative diff-and-apply; `atlasexec` shells out to a CLI binary; EULA complexity.
- **`gorm.io/gorm` AutoMigrate**: not a migration tool — best-effort sync to struct tags, no audit trail, no `embed.FS`, no advisory lock, no down.
- **Hand-rolled sqlx + custom runner**: reinvents locking and bookkeeping in ~500 LOC. Violates DRY.
- **`riverqueue/river` + `rivermigrate`**: ties migrations to a job-queue worker pool we don't run.

### Consequences
- **Positive**: MIT; `embed.FS` via `goose.SetBaseFS`; programmatic library; opt-in `pg_try_advisory_lock` (non-blocking, retries, no timeout death); pure library — no framework imports; hexagonal-clean.
- **Negative**: goose's recovery model (`goose fix`) does not use a `dirty` column, so a half-applied migration requires manual intervention. Documented in `migration/README.md`.
- **Risk**: goose could release a breaking change. Mitigated by pinning exactly and bumping via a deliberate PR.

### References
- github.com/pressly/goose — README
- github.com/pressly/goose lock/postgres.go — `SELECT pg_try_advisory_lock($1)`
- github.com/pressly/goose/releases — v3.27.1 (2026-04-24)

## 15. Failure mode sequence diagrams

### 15.1 DB unreachable at boot

```
main() -> migrationService.Up(ctx)
   +-- driver.Open(DATABASE_URL)
   |      +-- pgx connect attempt
   |             +-- connect timeout (5s via pgx.Config.ConnConfig.ConnectTimeout)
   |             +-- ERROR error: "connection refused"
   +-- return error
   +-- main() -> os.Exit(1)
```

### 15.2 Migration transaction rolls back

```
goose.Up(db, "sql")
   +-- pg_try_advisory_lock(42) -> OK
   +-- SELECT MAX(version_id) -> 0
   +-- BEGIN
   +-- exec "20260621120000_hello_world.sql" -> SELECT 1; -> OK
   +-- INSERT INTO schema_migrations ...
   +-- COMMIT -> OK
   +-- pg_advisory_unlock(42)
   +-- return []Version{...}
```

If a future migration (not the hello world) raises an exception:
```
goose.Up(db, "sql")
   +-- pg_try_advisory_lock(42) -> OK
   +-- BEGIN
   +-- exec "20260622_foo.sql" -> RAISE EXCEPTION 'boom'
   +-- ROLLBACK (automatic on error)
   +-- pg_advisory_unlock(42)
   +-- return error
```
The next start sees `MAX(version_id) = 20260621120000` (the previous successful one) and tries to re-apply 20260622_foo.sql. If the operator fixes the SQL, the next start succeeds.

### 15.3 Two replicas start in parallel

```
replica A: pg_try_advisory_lock(42) -> OK (lock acquired)
replica B: pg_try_advisory_lock(42) -> false (lock not acquired)
replica B: log "waiting for migration lock" every 1s
replica A: ... apply migrations ... pg_advisory_unlock(42)
replica B: pg_try_advisory_lock(42) -> OK (lock acquired)
replica B: SELECT MAX(version_id) -> 20260621120000 (no new migrations)
replica B: pg_advisory_unlock(42)
replica B: continue to Echo bind
```

## 16. Risks (additions to proposal §Risks)

| Risk | Likelihood | Mitigation |
|---|---|---|
| `embed.FS` doesn't include newly-added migration files. | Medium | `make migrate/lint` opens the embed.FS and asserts every filename matches `^\d{14}_[a-z0-9_]+\.sql$`. |
| `goose.WithSessionLocker` causes the second replica to spin-log "waiting for lock" if the first replica crashes mid-migration. | Low | pgx releases the lock when the connection drops (session-scoped). The second replica acquires the lock on its next 1s retry. |
| The runner's bounded context (`MIGRATION_TIMEOUT=30s`) is too short for a slow migration. | Low (future) | Documented in `migration/README.md`: large migrations must use online ops (`CREATE INDEX CONCURRENTLY`, `ADD COLUMN ... DEFAULT NULL` then backfill). |
| The stdlib pgx adapter's `sql.Open` does not surface a connection error until the first query. | Low | We test connectivity in `driver.Open` itself (a `db.Ping()` call). Fail-fast at Open, not at the first query. |
| `goose fix` recovery requires manual `UPDATE` on the bookkeeping table. | Low | Documented in `migration/README.md`. |

## 17. Rollback

See proposal §Rollback. Refinements:
- Reverting the design itself (this doc) is informational only.
- Reverting the design decisions requires a new SDD cycle (this change is the locked state).

## 18. Review checklist

- [ ] reviewer can confirm the boot sequence diagram shows both the postgres initdb and database_administrator start paths
- [ ] reviewer can confirm the hexagonal rule is enforced: domain imports nothing from migration/, application imports domain only, migration/runner imports goose, migration/postgres imports pgx
- [ ] reviewer can confirm Q3 (runner timing) is locked to "every container start"
- [ ] reviewer can confirm Q4 (failure behavior) is locked to "crash on migration error"
- [ ] reviewer can confirm Q5 (down migrations) is locked to "schema-only yes, data-moves no"
- [ ] reviewer can confirm Q6 (layout) is locked to "one global dir, schema-qualified names"
- [ ] reviewer can confirm Q7 (bookkeeping column) is locked to "keep TEXT, no BIGINT in this change"
- [ ] reviewer can confirm Q8 (OTel attributes) lists the exact attribute names + types
- [ ] reviewer can confirm Q9 (pgx/v5 ADR) is saved to Engram under `adr/pgx-v5-stdlib-adapter`
- [ ] reviewer can confirm Q10 (goose ADR) is saved to Engram under `adr/goose-v3-embed-fs`
- [ ] reviewer can confirm the three failure-mode sequence diagrams cover DB-unreachable, tx-rollback, and parallel-replica cases
- [ ] reviewer can confirm no file under `backend/database_administrator/src/` was modified by this design
- [ ] reviewer can confirm the scoped infra exception is not widened by the design (only `01-init.sql` is touched, by apply)

## 19. Incompleteness log entry

- [2026-06-21] [openspec/changes/postgres-database-migrations/] `tasks.md` missing — design phase produces design.md; tasks land in sdd-tasks. **Proposed fix**: track in `sdd-tasks`.
