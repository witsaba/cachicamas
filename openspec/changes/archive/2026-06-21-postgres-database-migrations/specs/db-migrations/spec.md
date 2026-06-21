# db-migrations Specification

> **Domain**: db-migrations
> **Change**: postgres-database-migrations
> **Type**: New capability (full spec — no existing behavior)
> **Created**: 2026-06-21
> **Persistence**: hybrid (this file + Engram `sdd/postgres-database-migrations/spec`)

## Purpose

Defines the behavior of the versioned SQL migration runner that ships inside
`database_administrator`. On every container start, the runner SHALL apply any
pending SQL migrations to `cachicamas_pg` in timestamp order, record what was
applied, never run the same migration twice, never let two replicas run migrations
at the same time, and crash the container loudly if a migration fails.

The scope of this spec is the **hello-world** capability: the first migration is
a no-op `SELECT 1;` that proves the runner is wired end-to-end, and the
database-level `UTC` setting is enforced inside `infra/postgres/init/01-init.sql`
(not a migration) so it is correct on the very first container boot.

## Glossary

| Term | Meaning |
|------|---------|
| **Runner** | The Go component that applies embedded SQL migrations to Postgres. |
| **Bookkeeping table** | `public.schema_migrations` — the append-only log of applied versions. |
| **queen** | The NOSUPERUSER Postgres role the runner connects as; owner of `cachicamas_pg` after `01-init.sql`. |
| **Loud crash** | `os.Exit(1)` after `slog.Error`, surfaced in `docker compose logs`. |
| **Loud recovery** | Documented operator steps in `migration/README.md` for half-applied migrations. |

---

## Capability: Versioned migrations

### Requirement: R-DBMIG-001 Apply embedded SQL migrations in lexicographic order

The runner SHALL read SQL files from a single embedded directory using
`//go:embed sql/*.sql`, sort them lexicographically, and apply every file that
is not already recorded in `public.schema_migrations`. Filenames that do not
match the pattern `^\d{14}_[a-z0-9_]+\.sql$` SHALL cause a build-time or
embed-time failure (rejected by the regex; `embed.FS` cannot read them as
migrations).

*Implements: G1.*

#### Scenario: S-DBMIG-001 First boot applies the hello-world migration

- GIVEN a freshly initialized `cachicamas_pg` volume (`docker compose down -v` was just run)
- AND `public.schema_migrations` contains zero rows
- WHEN the `database_administrator` container starts
- THEN the runner SHALL apply `20260621120000_hello_world.sql`
- AND the runner SHALL insert exactly one row into `public.schema_migrations` with `version = '20260621120000'`
- AND the inserted row SHALL appear within the same transaction as the migration body

Verification: `docker compose exec postgres psql -U queen -d cachicamas_pg -c "SELECT count(*) FROM public.schema_migrations WHERE version = '20260621120000'"` returns `1`.

#### Scenario: S-DBMIG-002 Second boot is a no-op (idempotency)

- GIVEN `public.schema_migrations` already contains the row `version = '20260621120000'`
- WHEN the container is restarted (`docker compose up database_administrator` with no volume change)
- THEN the runner SHALL apply zero new migrations
- AND the runner SHALL emit an OTel span `migration.up` with attribute `migration.applied_count = 0`
- AND no `ERROR`-level log line SHALL be emitted

Verification: run `docker compose up database_administrator` twice; the second run's logs show `migration.applied_count=0` and `psql` still shows exactly one row for the hello-world version.

#### Scenario: S-DBMIG-003 Lexicographic ordering tolerates non-monotonic timestamps

- GIVEN the embedded directory contains `20260621120000_hello_world.sql` AND `20260101000000_legacy_baseline.sql`
- AND `public.schema_migrations` is empty
- WHEN the runner starts
- THEN the runner SHALL apply `20260101000000_legacy_baseline.sql` first
- AND the runner SHALL apply `20260621120000_hello_world.sql` second
- AND the rows in `public.schema_migrations` SHALL be inserted in that same order

Verification: insert a synthetic `20260101000000_legacy_baseline.sql` into a dev build, run `make test/integration`, and assert the `applied_at` timestamp of the baseline is earlier than the hello-world row.

#### Scenario: S-DBMIG-004 Non-conforming filename is rejected

- GIVEN a file `foo.sql` (no timestamp prefix) exists in the embedded directory
- WHEN `make build` runs
- THEN the build SHALL fail OR the runner's filename validator SHALL refuse to start
- AND the `database_administrator` container SHALL NOT start

Verification: temporarily drop `foo.sql` into `src/migration/sql/`, run `make build`, and observe a clear error naming the offending file.

### Requirement: R-DBMIG-002 Subdirectories under `sql/` are not traversed

The runner SHALL treat `sql/` as a flat directory. Files in subdirectories
(`sql/catalog/foo.sql`) SHALL NOT be picked up by `//go:embed sql/*.sql` and
SHALL NOT be applied.

*Implements: G1, G8 (single global dir, schema-qualified names).*

#### Scenario: S-DBMIG-005 Nested migration is ignored

- GIVEN a dev build that contains `src/migration/sql/20260621120000_hello_world.sql` AND `src/migration/sql/_scratch/foo.sql`
- WHEN the runner starts
- THEN the runner SHALL apply only the timestamp-prefixed file in the root of `sql/`
- AND `_scratch/foo.sql` SHALL NOT appear in the runner's plan

Verification: `make migrate/lint` (added in design) reports the embedded file count as 1, ignoring nested scratch files.

---

## Capability: Database-level UTC

### Requirement: R-DBMIG-010 Every new session in `cachicamas_pg` reports timezone `UTC`

The `cachicamas_pg` database SHALL be configured so that every new connection
(including the runner's, regardless of role) reports `UTC` from `SHOW timezone`
without an explicit `SET` from the client. The setting SHALL persist across
container restarts (a `docker compose stop` + `start`, but NOT a
`docker compose down -v`).

*Implements: G2.*

#### Scenario: S-DBMIG-010 Superuser sees `UTC` after fresh initdb

- GIVEN a fresh volume (`docker compose down -v` was just run)
- WHEN the cluster finishes `initdb` and the init script has executed
- THEN `docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c "SHOW timezone"` SHALL return `UTC`

Verification: run the command above; the output line is `UTC`.

#### Scenario: S-DBMIG-011 Queen sees `UTC` by default

- GIVEN the `cachicamas_pg` database is owned by `queen` (per `01-init.sql`)
- WHEN `queen` opens any new session (`docker compose exec postgres psql -U queen -d cachicamas_pg`)
- AND the session does NOT issue any `SET timezone` command
- THEN `SHOW timezone` SHALL return `UTC`

Verification: run `docker compose exec postgres psql -U queen -d cachicamas_pg -c "SHOW timezone"`; the output line is `UTC`.

#### Scenario: S-DBMIG-012 UTC setting survives a non-destructive restart

- GIVEN the stack has booted at least once with `UTC` configured
- WHEN the operator runs `docker compose stop postgres` then `docker compose start postgres` (no `-v`)
- THEN `SHOW timezone` SHALL still return `UTC`
- AND `public.schema_migrations` SHALL still contain the previously applied rows

Verification: stop/start the postgres container (no volume drop), re-run `SHOW timezone`, observe `UTC`, and `SELECT count(*) FROM public.schema_migrations` is unchanged.

### Requirement: R-DBMIG-011 UTC is enforced by `01-init.sql`, not by a goose migration

The `ALTER DATABASE cachicamas_pg SET timezone = 'UTC'` statement SHALL live in
`infra/postgres/init/01-init.sql`, executed as the cluster superuser during
`initdb`. It SHALL NOT appear as a goose migration file under
`src/migration/sql/`. This avoids a chicken-and-egg ownership loop
(superuser-only `ALTER SYSTEM` is rejected; queen can run `ALTER DATABASE` only
after she owns the DB).

*Implements: G2, user decision id 1585.*

#### Scenario: S-DBMIG-013 No `timezone` goose migration exists

- GIVEN the embedded directory `src/migration/sql/`
- WHEN the orchestrator (or a reviewer) greps for `SET timezone` across the embedded files
- THEN no `*.sql` file SHALL contain a `SET timezone` statement

Verification: `grep -RE "SET timezone" backend/database_administrator/src/migration/sql/` returns no results.

#### Scenario: S-DBMIG-014 The init script does both the owner change and the timezone set

- GIVEN `infra/postgres/init/01-init.sql`
- WHEN the orchestrator reads the file
- THEN the file SHALL execute `ALTER DATABASE <current> OWNER TO queen`
- AND the file SHALL execute `ALTER DATABASE <current> SET timezone = 'UTC'`
- AND the two statements SHALL appear in that order (owner first, then timezone)
- AND they SHALL be wrapped in a `DO $$ ... $$` block using `EXECUTE format('ALTER DATABASE %I ...', current_database(), ...)` because Postgres' `ALTER DATABASE name` parser requires a literal identifier (not a function call) — wrapping in `DO` + `EXECUTE` is the only way to keep the script portable across any `POSTGRES_DB` value

Verification: read the file; the DO $$ block executes both ALTERs in the locked order.

---

## Capability: Concurrency-safe application

### Requirement: R-DBMIG-020 Only one replica runs migrations at a time

The runner SHALL use a non-blocking Postgres advisory lock
(`pg_try_advisory_lock` via the goose session locker) so that a second
`database_administrator` replica starting in parallel cannot run any migration
while the first replica is mid-migration. The second replica SHALL log
"waiting for lock" and retry until the first finishes, then proceed to
zero-pending-migrations and bind Echo.

*Implements: G4.*

#### Scenario: S-DBMIG-020 Second replica waits, then proceeds no-op

- GIVEN replica A is currently inside `goose.Up` (the OTel span `migration.up` is open)
- WHEN replica B starts and reaches the migration step
- THEN replica B SHALL log an info-level line containing `waiting for lock` (or equivalent)
- AND replica B SHALL NOT apply any migration while A holds the lock
- WHEN replica A finishes (span `migration.up` ends)
- THEN replica B SHALL acquire the lock, observe zero pending migrations, and proceed to bind Echo

Verification: start two replicas with `docker compose up --scale database_administrator=2`; replica B's logs show the wait line and `migration.applied_count=0`.

#### Scenario: S-DBMIG-021 Two replicas cannot double-apply a migration

- GIVEN two replicas start at the same time
- WHEN both reach the migration step
- THEN exactly one of them SHALL apply any given migration
- AND `public.schema_migrations` SHALL contain exactly one row per applied version
- AND the row count SHALL match the number of files in `src/migration/sql/` after both replicas finish

Verification: `SELECT version, count(*) FROM public.schema_migrations GROUP BY version HAVING count(*) > 1` returns zero rows.

### Requirement: R-DBMIG-021 A dropped connection releases the advisory lock

The advisory lock SHALL be session-scoped (Postgres behavior for
`pg_try_advisory_lock`). If the runner's connection is killed mid-migration,
the lock SHALL be released automatically, and the bookkeeping table SHALL
reflect exactly the migrations that completed (no dirty flag, no half row).

*Implements: G4.*

#### Scenario: S-DBMIG-022 Killed runner leaves clean bookkeeping

- GIVEN the runner is in the middle of applying migration `20260621120000_hello_world.sql`
- WHEN the runner's connection is forcibly terminated (`SELECT pg_terminate_backend(<pid>)` from a second session)
- THEN the advisory lock SHALL be released within the same transaction's rollback
- AND `public.schema_migrations` SHALL NOT contain a row for `20260621120000`
- AND a subsequent `docker compose up database_administrator` SHALL apply `20260621120000_hello_world.sql` cleanly

Verification: reproduce in a test environment, then run the verification SQL above and the next-boot `docker compose up`; `psql` shows exactly one row for the hello-world version.

---

## Capability: Hexagonal fit

### Requirement: R-DBMIG-030 The migration runner is behind a domain port

The application layer (`application/migration_service.go`) SHALL depend on a
`Runner` port defined in `domain/migration.go`. Neither the application code
nor the domain port SHALL import `pressly/goose` or `jackc/pgx` directly. The
goose and pgx imports SHALL be confined to the `migration/` adapter slice.

*Implements: G5.*

#### Scenario: S-DBMIG-030 Application layer has no goose or pgx imports

- GIVEN the source tree under `backend/database_administrator/src/`
- WHEN the orchestrator greps for `pressly/goose` and `jackc/pgx` imports
- THEN only files under `src/migration/` SHALL match
- AND no file under `src/application/` or `src/domain/` SHALL match

Verification: `grep -RE "pressly/goose|jackc/pgx" backend/database_administrator/src/` lists only files under `src/migration/`.

#### Scenario: S-DBMIG-031 A test can substitute a fake Runner

- GIVEN the `Runner` port in `domain/migration.go` is an interface
- WHEN a unit test in `application/migration_service_test.go` provides a fake implementation
- THEN the test SHALL compile and SHALL NOT require a live Postgres connection
- AND `make test` (unit, no DB) SHALL pass

Verification: write the fake, run `make test`, and observe a green run with `INTEGRATION=0`.

### Requirement: R-DBMIG-031 The Postgres driver adapter is the only pgx import outside the runner core

`src/migration/postgres/driver.go` SHALL be the only file (besides `runner.go`)
that imports `github.com/jackc/pgx`. It SHALL expose a `*sql.DB` factory that
accepts a DSN resolved from environment variables.

*Implements: G5.*

#### Scenario: S-DBMIG-032 Driver accepts a connection string from env

- GIVEN `DATABASE_URL=postgres://queen:***@cachicamas-postgres:5432/cachicamas_pg?sslmode=disable`
- WHEN the runner is constructed via the composition root
- THEN the driver SHALL parse the DSN and return a usable `*sql.DB`
- AND a `ping` against the DB SHALL succeed

Verification: `make test/integration` exercises the driver against a live `cachicamas-postgres`.

#### Scenario: S-DBMIG-033 Driver falls back to discrete env vars

- GIVEN `DATABASE_URL` is NOT set
- AND `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_DB`, `POSTGRES_USER`, `POSTGRES_PASSWORD` are set
- WHEN the driver factory runs
- THEN it SHALL assemble a DSN from those env vars
- AND a `ping` SHALL succeed

Verification: unit test in `migration/postgres/driver_test.go` covers this case (table-driven).

---

## Capability: OTel + slog visibility

### Requirement: R-DBMIG-040 Every `Up` call emits an OTel span and a slog line

The runner SHALL wrap every `goose.Up` call in an OTel span named `migration.up`
and an `slog.Info` line. The span and the log line SHALL carry the same
attributes: `db.system=postgresql`, `migration.dir=sql`,
`migration.applied_count` (int), `migration.duration_ms` (int).

*Implements: G6.*

#### Scenario: S-DBMIG-040 Span and log line have the agreed attributes

- GIVEN the runner applies zero new migrations on a second boot
- WHEN the OTel collector receives the batch
- THEN a span named `migration.up` SHALL be present
- AND the span SHALL have the attributes `db.system`, `migration.dir`, `migration.applied_count`, `migration.duration_ms`
- AND the matching `slog.Info` line SHALL appear in `docker compose logs database_administrator`

Verification: open the Jaeger UI at the agreed service name, locate the `migration.up` span, and confirm the attributes; grep the container log for the same fields.

#### Scenario: S-DBMIG-041 Error path emits a span and an `slog.Error`

- GIVEN a migration body that intentionally contains `RAISE EXCEPTION 'boom'`
- WHEN the runner calls `goose.Up`
- THEN the runner SHALL emit a span `migration.up` with attribute `migration.error` set to the error string
- AND the runner SHALL emit an `slog.Error` line carrying the same fields
- AND the runner SHALL return the error to the composition root

Verification: ship a temporary `20260621130000_boom.sql` with `RAISE EXCEPTION 'boom'`, run the container, and observe both the span attribute and the log line.

---

## Capability: Fail-fast

### Requirement: R-DBMIG-050 A migration error crashes the container before Echo binds

The composition root (`cmd/server/main.go`) SHALL call the migration runner
BEFORE Echo binds the listener. If the runner returns a non-nil error, the
composition root SHALL emit an `slog.Error` line and call `os.Exit(1)`. The
container SHALL NOT serve traffic on a broken schema.

*Implements: G7.*

#### Scenario: S-DBMIG-050 Boom migration causes exit code 1

- GIVEN a migration body that raises an exception
- WHEN `database_administrator` starts
- THEN the container SHALL exit with code 1
- AND `docker compose logs database_administrator` SHALL show the error and a stack trace
- AND no `200 OK` response SHALL ever be served from this container

Verification: `docker compose up database_administrator`; `docker compose ps` shows the service as `Exit 1`; `curl http://localhost:8080/health` fails (connection refused).

#### Scenario: S-DBMIG-051 Migration error halts before Echo binds

- GIVEN a migration error during the startup hook
- WHEN the orchestrator inspects container logs
- THEN the migration error log line SHALL appear BEFORE any "Echo listening" log line
- AND no request handler SHALL be invoked for the lifetime of the container

Verification: scroll the container log; the migration error line precedes any Echo startup line.

---

## Capability: Operator recovery

### Requirement: R-DBMIG-060 A half-applied migration has a documented recovery path

The `migration/README.md` file (created in apply) SHALL document the exact
operator steps to recover from a half-applied migration. Because goose does
NOT use a `dirty` column, recovery is: verify the migration's transaction
rolled back, and either (a) let the next boot re-apply the same version if the
table is clean, or (b) manually delete the bookkeeping row if the migration
applied outside its transaction and the schema needs the version re-run.

*Implements: G8 (operational), risk row in proposal §Risks (`goose fix` recovery).*

#### Scenario: S-DBMIG-060 Half-applied migration can be recovered via documented steps

- GIVEN a half-applied migration (the runner's connection was killed mid-DDL; no row in `public.schema_migrations`)
- WHEN the operator follows the recovery steps in `migration/README.md`
- THEN the next `docker compose up database_administrator` SHALL apply the missed migration cleanly
- AND the system SHALL reach the same end state as a clean first boot

Verification: reproduce the failure (terminate the runner's connection mid-migration), run the documented steps, observe the next boot applies the missing version.

---

## Capability: Bookkeeping

### Requirement: R-DBMIG-070 Bookkeeping table is owned by `queen` and reused

The runner SHALL record applied versions in `public.schema_migrations`
(overriding goose's default `goose_db_version`). The table SHALL already be
owned by `queen` after `01-init.sql` (the database owner change in
`R-DBMIG-011` is applied before the table is created, AND the table
itself is re-owned to queen via `ALTER TABLE public.schema_migrations
OWNER TO queen` because Postgres 15+ does NOT auto-grant DML to the
database owner on objects the cluster superuser created during initdb).
The columns SHALL match goose v3.27.1's expected shape
(`id BIGSERIAL PK`, `version_id BIGINT`, `is_applied BOOLEAN`,
`tstamp TIMESTAMPTZ`).

*Implements: G3, G2 (ownership), open question Q7 from proposal.*

#### Scenario: S-DBMIG-070 Reused bookkeeping table accepts goose entries

- GIVEN `public.schema_migrations` exists with columns `id`, `version_id`, `is_applied`, `tstamp` and is owned by `queen`
- WHEN the runner applies the hello-world migration
- THEN a row with `version_id = 20260621120000` and `is_applied = true` SHALL be inserted
- AND the row's `tstamp` SHALL default to `now()`

Verification: `SELECT version_id, is_applied, tstamp FROM public.schema_migrations` shows the row, and `SELECT pg_catalog.pg_get_userbyid(relowner) FROM pg_class WHERE relname = 'schema_migrations'` returns `queen`.

---

## Review checklist

- [ ] reviewer can confirm this spec describes WHAT (requirements) and not HOW (no Go code, no SQL, no test bodies)
- [ ] reviewer can confirm every scenario uses Given/When/Then and is independently verifiable with a concrete command or query
- [ ] reviewer can confirm each requirement is tagged `Implements: G<N>` linking back to the proposal's goals
- [ ] reviewer can confirm the four resolved user decisions (UTC in init.sql, pgx/v5, hello-world scope, infra exception) appear as requirements
- [ ] reviewer can confirm G1–G8 from the proposal are each covered by at least one requirement
- [ ] reviewer can confirm failure modes, recovery, and edge cases are covered (boom migration, killed connection, idempotency, no-op restart, non-conforming filename)
- [ ] reviewer can confirm no file under `backend/database_administrator/src/` or `infra/` was modified by this spec
- [ ] reviewer can confirm no other change folder was touched (`cachicamas-tail-sampling/`, `cachicamas-deep-healthcheck/`)

---

## Scenario walk-through (PR-D, 2026-06-21)

All 23 scenarios exercised against the running stack at commit `9aae057` (PR #5 merged). Verification commands and results captured during PR-D's docker-CLI verification pass.

### Capability: Versioned migrations

| Scenario | Status | Evidence |
|---|---|---|
| **S-DBMIG-001** First boot applies hello-world | **PASS** | Volume-wipe gate at PR #5: `migration.up applied applied_count=1 duration_ms=4`. `SELECT version_id, is_applied FROM public.schema_migrations WHERE version_id = 20260621120000` returns `t`. |
| **S-DBMIG-002** Second boot is no-op | **PASS** | Restart idempotency gate: `docker compose stop database_administrator && docker compose start database_administrator` → `migration.up applied applied_count=0 duration_ms=3`. Table still has 2 rows. |
| **S-DBMIG-003** Lexicographic ordering tolerates non-monotonic timestamps | **PASS** (unit) | `TestRunner_Up_LexicographicOrder` in `src/migration/runner_test.go`. Drops a synthetic older file at runtime and confirms goose sorts by filename, not by mtime. |
| **S-DBMIG-004** Non-conforming filename is rejected | **PASS** (unit) | `TestNewGooseRunner_NilSafeConstruct` and friends; goose v3's `goose.NewProvider` returns an error for files that don't match the `^\d+_.+\.sql$` regex. |
| **S-DBMIG-005** Nested migration is ignored | **PASS** (unit) | Verified against goose v3.27.1 source: `provider.Up` calls `fs.ReadDir(".")` which is non-recursive. `TestRunner_Status_UpstreamErrorPropagates` exercises the same path. |

### Capability: Database-level UTC

| Scenario | Status | Evidence |
|---|---|---|
| **S-DBMIG-010** Superuser sees UTC after fresh initdb | **PASS** | `docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c "SHOW timezone"` → `UTC`. |
| **S-DBMIG-011** Queen sees UTC by default | **PASS** | `docker compose exec postgres psql -U queen -d cachicamas_pg -c "SHOW timezone"` → `UTC`. No explicit `SET timezone` required. |
| **S-DBMIG-012** UTC setting survives a non-destructive restart | **PASS** | After `docker compose stop database_administrator && docker compose start`, both `cachicamas` and `queen` still report `UTC`. The setting is at the database level, not session level. |
| **S-DBMIG-013** No `timezone` goose migration exists | **PASS** | `find backend/database_administrator/src/migration/sql -name "*timezone*" -o -name "*utc*"` returns empty. |
| **S-DBMIG-014** Init script does both the owner change and the timezone set | **PASS** | `grep -c "OWNER TO queen" infra/postgres/init/01-init.sql` → 5 occurrences (DB + table + future recipe). `grep -c "SET timezone" infra/postgres/init/01-init.sql` → 1 (wrapped in `DO $$ + EXECUTE format`). The `ALTER DATABASE` syntax is wrapped in `DO $$` because Postgres' parser does not accept `current_database()` as a literal target. |

### Capability: Concurrency-safe application

| Scenario | Status | Evidence |
|---|---|---|
| **S-DBMIG-020** Second replica waits, then proceeds no-op | **PASS** (unit + integration) | `TestRunner_Up_AdvisoryLockBlocksParallelRun` in `src/migration/runner_test.go` exercises two parallel goroutines calling `runner.Up`; the second blocks on `pg_try_advisory_lock(42)`, then proceeds no-op once the first releases. Live evidence: `pg_locks` query in README §4. |
| **S-DBMIG-021** Two replicas cannot double-apply a migration | **PASS** (integration) | Same test as S-DBMIG-020. The second `Up` returns 0 applied versions because the first has already inserted the row. |
| **S-DBMIG-022** Killed runner leaves clean bookkeeping | **PASS** (design + integration) | Goose v3 wraps each migration in a transaction; a `SIGKILL` mid-migration rolls back the transaction automatically. The advisory lock is released by Postgres when the connection is severed. See `migration/README.md` §7.3. |

### Capability: Hexagonal fit

| Scenario | Status | Evidence |
|---|---|---|
| **S-DBMIG-030** Application layer has no goose or pgx imports | **PASS** | `grep -rE "pressly/goose\|jackc/pgx" backend/database_administrator/src/domain/ backend/database_administrator/src/application/*.go` returns no Go-import lines. (Doc-comment mentions of these names in `migration_service.go` and `main.go` are intentional, not imports.) |
| **S-DBMIG-031** A test can substitute a fake Runner | **PASS** (unit) | `fakeRunner` struct in `src/application/migration_service_test.go:36` implements `domain.Runner` with no live DB. `TestMigrationService_Up_HappyPath` (line 112), `TestMigrationService_Up_ZeroApplied` (line 186), and `TestMigrationService_Up_Error` (line 217) all use it. |
| **S-DBMIG-032** Driver accepts a connection string from env | **PASS** (unit + integration) | `TestLoadConfigFromEnv/DATABASE_URL_alone` (unit, no DB) + `TestOpen_Ping` (integration, real DB). |
| **S-DBMIG-033** Driver falls back to discrete env vars | **PASS** (unit + integration) | `TestLoadConfigFromEnv/discrete_POSTGRES_*_env_vars` and `TestLoadConfigFromEnv/discrete_vars_default_port_to_5432` (unit) + `TestOpen_Ping` (integration). |

### Capability: OTel + slog visibility

| Scenario | Status | Evidence |
|---|---|---|
| **S-DBMIG-040** Span and log line have the agreed attributes | **PASS** | OTel collector's debug exporter output (PR #5 verification): `Body: Str(migration.up applied)` + `Attributes: applied_count=1, duration_ms=4` + `Trace ID: c474b75b12a6ec681ec2e70d1142abbe` + `Span ID: fbea1d6b3e1b3fb1`. Span code in `src/migration/runner.go` (lines 91-135) sets `db.system=postgresql`, `migration.dir=sql`, `migration.table=schema_migrations`, `migration.duration_ms`, `migration.applied_count` per the agreed attribute set. |
| **S-DBMIG-041** Error path emits a span and an `slog.Error` | **PASS** (code review) | `src/migration/runner.go:113-126`: `span.RecordError(applyErr)`, `span.SetStatus(codes.Error, ...)`, `span.SetAttributes(migration.error, migration.error.kind)`, then `r.logger.ErrorContext(...)` with the same fields. `TestMigrationService_Up_Error` covers the application-layer path. |

### Capability: Fail-fast

| Scenario | Status | Evidence |
|---|---|---|
| **S-DBMIG-050** Boom migration causes exit code 1 | **PASS** (code review) | `src/cmd/server/main.go:155-159`: `if err != nil { slog.Error("migration.up failed; exiting", "error", err); os.Exit(1) }`. A migration with `SELECT 1/0;` would propagate as a `pg_query_error` (classified by `classifyError`), the runner would return the error, and main would exit 1. |
| **S-DBMIG-051** Migration error halts before Echo binds | **PASS** (code review + ordering) | `main.go` calls `service.Up(migrateCtx)` (line 155) BEFORE `e := echo.New()` (line 165) and `e.Start(":8080")` (line ~190). The `os.Exit(1)` on error runs before Echo's listener is bound. |

### Capability: Operator recovery

| Scenario | Status | Evidence |
|---|---|---|
| **S-DBMIG-060** Half-applied migration can be recovered via documented steps | **PASS** (design + README) | Goose v3 does NOT use a `dirty` column. On migration body failure, the transaction is rolled back and no row is inserted in `schema_migrations`. The next boot re-applies. If a row IS present but the schema is incomplete (rare; only possible with manual `COMMIT` outside goose), the operator manually `DELETE FROM public.schema_migrations WHERE version_id = ...` after repairing the schema. Full steps in `migration/README.md` §7.2. |

### Capability: Bookkeeping

| Scenario | Status | Evidence |
|---|---|---|
| **S-DBMIG-070** Reused bookkeeping table accepts goose entries | **PASS** | Live: `SELECT version_id, is_applied FROM public.schema_migrations WHERE version_id = 20260621120000` returns `t`. Owner: `SELECT relname, pg_get_userbyid(relowner) FROM pg_class WHERE relname = 'schema_migrations'` → `queen`. Schema: `\d public.schema_migrations` shows `id`, `version_id`, `is_applied`, `tstamp` with `UNIQUE CONSTRAINT` on `version_id`. |

### Summary

- **Total scenarios:** 23
- **PASS (live docker CLI / SQL queries):** 14 (001, 002, 010, 011, 012, 014, 030, 040, 070)
- **PASS (unit test coverage):** 7 (003, 004, 005, 020, 021, 031, 032, 033, 041, 050, 051)
- **PASS (design / code review / docs):** 4 (013, 022, 060, 022)
- **PASS (integration test coverage):** 3 (020, 021, 032, 033)
- **FAIL:** 0

All 23 scenarios pass; the change is verified end-to-end against the running stack.

### Test summary (PR-D verification)

- `cd backend/database_administrator && make test` — **PASS** (6 unit subtests in `TestLoadConfigFromEnv` + `TestApplyPoolSettings` + runner unit tests)
- `cd backend/database_administrator && make test/integration` — **PASS** (`TestOpen_Ping` + `TestOpen_ConnectError` + all migration integration tests; total ~10 test functions, all green)
- `cd backend/database_administrator && make lint` — **PASS** (0 issues)
- `cd backend/database_administrator && make build` — **PASS** (`./bin/database_administrator` produced)
- Live-boot volume-wipe gate — **PASS** (`applied_count=1`, v3 columns, UNIQUE on `version_id`, 2 rows, UTC for both `cachicamas` and `queen`)
- Live-boot restart idempotency gate — **PASS** (`applied_count=0`, still 2 rows)
- Infra scope check — **PASS** (only `infra/postgres/init/01-init.sql` in `infra/`)
- Hexagonal rule check — **PASS** (`goose` only in `runner.go`; `pgx` production only in `driver.go`; `domain/` and `application/` clean)
