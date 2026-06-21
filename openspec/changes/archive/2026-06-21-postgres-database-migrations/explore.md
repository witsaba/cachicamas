# Exploration — postgres-database-migrations

> **Change**: postgres-database-migrations
> **Status**: explored
> **Date**: 2026-06-21
> **Driver**: braejan
> **Project**: cachicamas (witsaba)
> **Persistence**: hybrid (Engram `sdd/postgres-database-migrations/explore` + filesystem `openspec/changes/postgres-database-migrations/explore.md`)
> **Scope of exploration**: only library comparison, UTC enforcement pattern, hello-world migration layout. NO code under `backend/database_administrator/src/` was written.

---

## Executive summary

The migration runner chosen for the `cachicamas` stack is **`pressly/goose` v3** (MIT, supports `embed.FS` via `goose.SetBaseFS`, programmatic library, opt-in advisory locking via `lock` subpackage). Rejected: `golang-migrate` v4 (blocking session-level `pg_advisory_lock` blocks indefinitely with no timeout — bad for queen's connection budget), `ariga/atlas` (overkill for "hello world"; CLI + plan/diff model is for declarative diffing, not for embedding in a single binary), `gorm.io/gorm AutoMigrate` (no `embed.FS`, no advisory locking, no auditability, drift-prone), hand-rolled `sqlx` (reinvents locking and bookkeeping — violates `openspec/config.yaml`'s "no new top-level deps without ADR" rule from the other direction by adding maintenance debt), `riverqueue/river` (job queue, not migration; `rivermigrate` subpackage is interesting but ties migrations to a worker pool we don't run).

The first migration enforces **database-level UTC** by running `ALTER DATABASE cachicamas_pg SET timezone = 'UTC'` AS `queen`. Verified: `queen` is the database owner in this stack (created in `infra/postgres/init/01-init.sql`, granted ownership of all schemas), and `timezone` is `PGC_USERSET` per the PG 18 runtime-config docs — any user can set it. `ALTER SYSTEM SET timezone` is **rejected** (requires superuser, queen is NOSUPERUSER). `ALTER ROLE ALL SET timezone` is **rejected** (superuser-only). The exact SQL is in section 3 below.

Migrations live at `backend/database_administrator/src/migration/sql/*.sql`, embedded into the binary with `//go:embed`. Naming: **timestamp-prefixed** (`20260621120000_set_database_timezone_to_utc.sql`) — matches goose's default, avoids merge conflicts on parallel PRs. CI: `make test` does NOT apply migrations (unit tests only); migrations are applied at container boot by the `database_administrator` service itself via a `goose.Up` call guarded by an OTel-instrumented function in `application/migration_service.go`. Tests cover the runner with a real Postgres via testcontainers or `make test/integration` target added in apply.

---

## 1. Library comparison matrix

All claims verified via WebFetch against GitHub releases pages, README, source code, and PostgreSQL 18 docs between 2026-06-21. Anything I could not verify is flagged **[unverified]**.

| Library | Latest stable | License | embed.FS | Prog. API | Postgres lock | Recover from dirty | queen runs it | Verdict |
|---|---|---|---|---|---|---|---|---|
| `pressly/goose` v3 | **v3.27.1** (released 2026-04-24) | MIT | yes (`goose.SetBaseFS(embed.FS)`) | yes (lib + CLI) | opt-in: `WithSessionLocker(lock.NewPostgresSessionLocker())` → `SELECT pg_try_advisory_lock($1)` (non-blocking, retries) + heartbeat variant | `goose fix` (clears dirty version) | yes (no superuser needed) | **CHOSEN** |
| `golang-migrate` v4 | **v4.19.1** (released 2025-11-29) | MIT | yes (`iofs.New(embed.FS, "migrations")` + `migrate.NewWithSourceInstance("iofs", driver, url)`) | yes | `pg_advisory_lock($1)` (session-level, **blocks indefinitely**, no timeout) | `migrate force VERSION` (manual `UPDATE schema_migrations SET dirty=false, version=$X`) | yes | rejected: blocking lock with no timeout is hostile to container orchestrators that restart on healthcheck failure |
| `ariga/atlas` | **CLI v1.2.0** (2026-04-10); **`atlas-go-sdk` v0.7.2** (2025-05-21, repo archived → use `atlasexec`) | CLI: dual (community binaries Apache-2.0, default binaries Atlas EULA); SDK: Apache-2.0 | yes (SDK reads from `fs.FS`) | yes (SDK is Apache-2.0) | uses managed transactions, no advisory lock required (declarative diff-and-apply model) | re-plan via `atlas schema apply --auto-approve` | yes, but cloud account adds value (local-only is workable) | rejected: overkill for v1; adds EULA complexity; designed for declarative diff, not imperative `Up()` |
| `gorm.io/gorm` AutoMigrate | n/a (part of gorm) | MIT | no (struct tags only) | yes | none | none — silent schema drift | yes | rejected: not a migration tool — AutoMigrate is "best-effort sync to struct", no audit trail, no down, no advisory lock |
| Hand-rolled `sqlx` + custom runner | n/a | n/a | yes (manual `fs.ReadFile`) | yes | would have to implement advisory lock + bookkeeping manually | would have to implement manually | yes | rejected: violates the spirit of `openspec/config.yaml`'s "no new top-level deps without an ADR" by replacing a dep with ~500 LOC of bug-prone logic; reinventing what `goose` does in <100 LOC of call site |
| `riverqueue/river` + `rivermigrate` | **[unverified: latest version]** | MIT | yes | yes | uses PG advisory locks for queue workers (not migrations per se) | transactional rollback | yes | rejected: river is a job queue; `rivermigrate` runs migrations inside the worker pool. We don't have a worker pool yet — bringing in river for migrations means importing a job queue we won't otherwise use |

### Key code references verified

- **goose v3.27.1 lock SQL** (from `lock/postgres.go`): `SELECT pg_try_advisory_lock($1)` — non-blocking + retry. Source confirmed via WebFetch.
- **goose v3.27.1 `SetBaseFS`** — confirmed via WebFetch of README: `goose.SetBaseFS(embedMigrations)` then `goose.Up(db, "migrations")`. **Caveat verified**: "Calling with `nil` reverts to OS filesystem behavior" and "Modifying operations like `Create` still use OS filesystem" — so `goose create` (CLI-only) is fine to use outside the binary; the binary-side `Up()` uses the embed.
- **golang-migrate v4.19.1 postgres Lock/Unlock** (from `database/postgres/postgres.go`): `SELECT pg_advisory_lock($1)` and `SELECT pg_advisory_unlock($1)`. The doc comment says "This will wait indefinitely until the lock can be acquired" — no timeout, no ctx-aware variant. **This is the disqualifying finding.**
- **golang-migrate v4.19.1 bookkeeping**: `(version bigint not null primary key, dirty boolean not null)` — uses a `dirty` flag (goose does NOT use a dirty flag, only a `version_id` row).
- **golang-migrate v4.19.1 recovery**: `migrate force <version>` runs `UPDATE schema_migrations SET version=$1, dirty=false` — manual operator action required after a partial migration.
- **goose v3.27.1 recovery**: `goose fix` is implemented per the dispatcher in `goose.go`'s `run()` switch but **the actual SQL was not retrievable via WebFetch** — flag as **[unverified: needs source dive during apply]**. Known: goose never uses a `dirty` column, so the recovery model is different from golang-migrate.
- **atlas CLI vs SDK**: per WebFetch, `ariga/atlas-go-sdk` is **archived** as of mid-2025; users are directed to `ariga.io/atlas/atlasexec`. The current SDK is Apache-2.0 (community) / Atlas EULA (default binaries). For an embedded migration runner, `atlasexec` shells out to the CLI under the hood.

### Hexagonal / OTel / slog fit

- **goose**: works as a pure library, no framework imports — fits `application/migration_service.go` cleanly. No OTel hooks; we wrap the `goose.Up` call with `slog.With("migration_dir", dir)` and `tracer.Start(ctx, "migration.up")` ourselves. **Verdict: hexagonal-clean.**
- **golang-migrate**: same — pure library, no framework deps. Hexagonal-clean.
- **atlas**: heavier. `atlasexec` requires shelling out to a binary or embedding the CGO-compiled atlas Go SDK. Pulls in a CLI dependency into our container image. **Verdict: hexagonal-awkward for "hello world".**
- **gorm**: imports a whole ORM. Doesn't fit the existing `sqlx`-free / Echo-only stack. **Verdict: architectural mismatch.**
- **hand-rolled**: fits perfectly but adds maintenance burden. **Verdict: violates DRY.**
- **river**: the `rivermigrate` package is library-only and would fit, but pulling in the river worker model just for migrations is overkill. **Verdict: architectural mismatch for v1.**

---

## 2. Current state (relevant to this topic)

### `backend/database_administrator/`
- `go.mod` declares Go 1.26.3, Echo v5.2.1, full OTel stack. **No Postgres driver yet** — driver selection (`jackc/pgx` v5 vs `lib/pq`) is part of `sdd-apply`, NOT this explore.
- `src/cmd/server/main.go` is the composition root. Currently wires logging → tracing → Echo → `/health`. **No startup migration step.**
- `src/application/health_service.go` + `src/domain/health.go` + `src/interfaces/http/health_handler.go` form the existing hexagonal slice. Migration is a **new** slice: `src/migration/` (per user prompt) → `src/application/migration_service.go` → `src/domain/migration.go` (port) → driven adapter inside `src/migration/postgres/`.

### `infra/postgres/init/01-init.sql`
Already provisions:
- `queen` role (NOSUPERUSER, CREATEROLE, CREATEDB, REPLICATION)
- Three schemas: `catalog`, `identity`, `observability` (owned by queen)
- **Empty placeholder** table `public.schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ, description TEXT)` — comment: "used by a future migration runner as queen"

**Conflict with goose**: goose's default bookkeeping table is `goose_db_version`, NOT `schema_migrations`. The existing `schema_migrations` table has the columns we'd want but uses TEXT (not BIGINT) for version. **Decision (deferred to apply)**: either rename to `goose_db_version` (and have goose create its own) or override goose's default with `-table schema_migrations`. The latter is simpler and reuses what's already provisioned.

### `docker-compose.yaml`
- `database_administrator` already connects as `queen` (env `POSTGRES_USER: queen`). Good — migration runner inherits the connection.
- No migration step today. The service just starts Echo and serves `/health`.

### `01-init.sql` ↔ migration runner split
- `01-init.sql` runs **once at first container boot** (only when `$PGDATA` is empty). Idempotent — re-running won't clobber existing data.
- A migration runner runs **on every service start** (or via a one-shot init container). Idempotent — checks the bookkeeping table and only applies new versions.
- These are **complementary**, not competing. Init creates the role + schemas + the empty bookkeeping table; migration runner fills it.

---

## 3. Postgres database-level UTC — canonical pattern for queen

Verified against the PostgreSQL 18 docs (`runtime-config.html` parameter classification + `sql-alterdatabase.html` + `sql-alterrole.html`).

### Privilege matrix for `timezone` GUC

| Command | Superuser? | queen (NOSUPERUSER, owner of cachicamas_pg)? | Persists for new sessions? |
|---|---|---|---|
| `ALTER SYSTEM SET timezone = 'UTC'` | **required** | **NO** (queen is not superuser) | yes (postgresql.conf equivalent) |
| `postgresql.conf` `timezone = 'UTC'` | required + restart | **NO** | yes |
| `ALTER DATABASE cachicamas_pg SET timezone = 'UTC'` | or database owner | **YES** (queen owns the DB) | **YES** (sets default for every new session in that DB) |
| `ALTER ROLE queen SET timezone = 'UTC'` | or role owner | **YES** (queen sets her own role) | yes |
| `ALTER ROLE ALL SET timezone = 'UTC'` | **required** | **NO** | yes |
| `SET timezone TO 'UTC'` | n/a | yes | **NO** (session-local only, dies with the connection) |

### Why `timezone` is settable by non-superusers

`TimeZone` is classified as **`PGC_USERSET`** in PostgreSQL's source (`guc.c`). PGC_USERSET parameters can be changed by any user via `SET`, `ALTER ROLE ... SET`, and (by extension) `ALTER DATABASE ... SET`. Verified via WebFetch against `runtime-config-client.html`.

The `ALTER DATABASE ... SET` form requires either superuser OR database ownership — and `queen` owns `cachicamas_pg` (the database created from `POSTGRES_DB` env var, which defaults to `cachicamas_pg`). However, ownership of a freshly-created `POSTGRES_DB` is given to `POSTGRES_USER` (the env var, which defaults to `cachicamas` — the superuser). **This is a gotcha for `sdd-apply`.** Two options:
1. `01-init.sql` runs `ALTER DATABASE current_database() OWNER TO queen;` (requires superuser to run, but `01-init.sql` runs during `initdb` which IS superuser — `POSTGRES_USER=cachicamas`). Then queen can run `ALTER DATABASE ... SET timezone` on it.
2. Have queen create the DB herself (in `01-init.sql`, superuser does `EXECUTE 'CREATE DATABASE cachicamas OWNER queen'`), then queen owns it from day 1.

**Decision deferred to `sdd-apply`**: option 1 is the cleanest minimal change. The `sdd-propose` phase will surface this as an open question.

### Exact proposed SQL for the first migration

```sql
-- 20260621120000_set_database_timezone_to_utc.sql
-- ============================================================================
-- Migration: 20260621120000_set_database_timezone_to_utc
-- Author:   braejan
-- Date:     2026-06-21
-- Purpose:  Force every new session in `cachicamas_pg` to use UTC. Persists
--           across restarts. Requires queen to own the database (see note
--           in proposal §Open Questions about the 01-init.sql interaction).
--
-- Why database-level, not session-level:
--   - Per-session `SET timezone TO 'UTC'` is lost when the connection ends.
--     Every code path (Echo handlers, background goroutines, future workers)
--     would have to remember to set it.
--   - Per-role `ALTER ROLE queen SET timezone = 'UTC'` works but only
--     applies when queen is the connecting user. The day we add a
--     least-privilege app role, that role needs its own ALTER ROLE.
--   - Per-database `ALTER DATABASE ... SET` applies to ALL connecting roles
--     for that DB. One line, future-proof.
--
-- Why not ALTER SYSTEM:
--   - Requires superuser. queen is NOSUPERUSER (see 01-init.sql line 75).
--
-- Idempotency:
--   - PostgreSQL makes ALTER DATABASE ... SET inherently idempotent: setting
--     the same value twice is a no-op. We don't need a WHERE clause or a
--     DO block to guard against re-running.
-- ============================================================================

ALTER DATABASE cachicamas_pg SET timezone = 'UTC';
```

**Down migration** (only if explicitly requested via `goose down` — the `Up` is irreversible in the sense that running `Down` reverts to whatever the server's default is, which may not be `UTC` if `01-init.sql` later sets it differently):

```sql
-- 20260621120000_set_database_timezone_to_utc.down.sql
ALTER DATABASE cachicamas_pg RESET timezone;
```

### Interaction with `01-init.sql`

The official postgres image's `initdb` uses `--locale=C --encoding=UTF8` (per `docker-compose.yaml`). It does NOT set `timezone` at initdb time. The default timezone in the cluster is whatever the container's `TZ` env says, which the Alpine base image sets to `UTC` by default. **So `initdb` produces a UTC cluster already.** This first migration is therefore belt-and-suspenders: it pins the value to `UTC` regardless of the container's `TZ` env, future `initdb` rebuilds, or operator `postgresql.conf` edits. It's the right migration even though the cluster is already UTC.

---

## 4. Hello-world pattern

### File layout

```
backend/database_administrator/src/migration/
├── README.md                  # explains how to add a migration
├── sql/                       # embedded via //go:embed *.sql
│   ├── 20260621120000_set_database_timezone_to_utc.sql
│   └── 20260621120000_set_database_timezone_to_utc.down.sql
├── embed.go                   # //go:embed sql/*.sql → embed.FS
├── runner.go                  # programmatic goose.Up wrapper (OTel + slog)
├── runner_test.go             # integration test (uses testcontainers)
└── postgres/
    ├── driver.go              # *sql.DB factory (jackc/pgx v5 driver)
    └── driver_test.go         # table-driven: parse valid + invalid DSNs
```

### embed.FS strategy

```go
// embed.go
package migration

import "embed"

//go:embed sql/*.sql
var migrationsFS embed.FS
```

Then in `runner.go`:
```go
goose.SetBaseFS(migrationsFS)
if err := goose.SetDialect("postgres"); err != nil { ... }
goose.Up(db, "sql")  // path is relative to the embed.FS root
```

Verified via WebFetch of goose's README: this is the supported pattern. `goose create` (CLI only) writes to the OS filesystem, not the embed, so devs can still `cd src/migration/sql && goose create add_users_table sql` to scaffold a new file.

### Naming convention

**Timestamp-prefixed**: `YYYYMMDDHHMMSS_description.sql`. Why not sequential (00001, 00002)?

| Approach | Pros | Cons |
|---|---|---|
| Timestamp (goose default) | Merge-friendly (two devs picking the same minute is rare; even if they do, goose sorts by lexicographic order and applies both). Self-documenting (the version matches when the migration was authored). | Slightly noisier filenames. |
| Sequential (00001, …) | Visually clean. | Merge conflict on every parallel PR. Forces serial code review or manual re-numbering on conflict. |

**Recommendation: timestamp.** It matches goose's default and dodges the merge-conflict footgun.

### CI / `make test` interaction

- `make test` runs `go test -race -v ./...` — unit tests only. **Migrations are NOT applied by `make test`.** The runner test is an integration test gated by `testing.Short()` — it skips in `-short` and runs against a real Postgres (testcontainers or the compose stack).
- A new Makefile target `make test/integration` is added in `sdd-apply` to run all integration tests. It boots the compose stack (`docker compose up -d postgres`) and waits for `pg_isready` before running tests.
- CI (GitHub Actions) runs `make test` on every PR; `make test/integration` runs on `main` push and on PRs labeled `integration`. **Decision deferred to `sdd-apply`** — out of scope for this explore.

### Does the bookkeeping table count as "hello world"?

The `public.schema_migrations` table already exists (provisioned by `01-init.sql`). **Two paths**:

| Path | Pros | Cons |
|---|---|---|
| Override goose's default with `goose.SetTableName("schema_migrations")` (or `-table` flag) | Reuses the table `01-init.sql` already created. Fewer schema objects. | The table has `version TEXT`, goose expects BIGINT. We'd need to ALTER it in `01-init.sql` or accept a mismatch (goose would still work — it just stores its version_id as a stringified bigint into the TEXT column). |
| Let goose create its own `goose_db_version` | Clean — goose owns its schema. | Two bookkeeping tables. Confusing in `psql`. |

**Recommendation: path 1**, override goose's table name to `schema_migrations`. Reuses what 01-init.sql provisioned. We add an `ALTER TABLE public.schema_migrations ALTER COLUMN version TYPE BIGINT` to the first migration if the TEXT column is incompatible — goose stores the version as a string and casts to BIGINT when reading.

---

## 5. Affected areas (deferred to `sdd-propose`)

| Area | Impact | Why |
|---|---|---|
| `backend/database_administrator/src/migration/` | NEW | The new hexagonal slice (port in `domain/`, runner in `application/migration_service.go`, adapter in `migration/postgres/`, embed.FS in `migration/sql/`). |
| `backend/database_administrator/src/cmd/server/main.go` | MODIFIED | Add a startup hook that calls `migrationService.Up(ctx)`. Must run BEFORE Echo starts serving traffic — otherwise a v1 release with a broken schema could serve 500s. The hook must also NOT block forever if the DB is down — use the existing OTel `tracer.Start` + a bounded context (5s connect timeout via `pgx.Config.ConnConfig.ConnectTimeout`). |
| `backend/database_administrator/go.mod` | MODIFIED | Add `github.com/pressly/goose/v3 v3.27.1` and a Postgres driver (likely `github.com/jackc/pgx/v5` — verify with the apply-phase ADR). Both new top-level deps → ADR required per `openspec/config.yaml`. |
| `backend/database_administrator/Makefile` | MODIFIED | Add `make test/integration` target. Optionally add `make migrate/new NAME=...` that runs `goose create` against `src/migration/sql/`. |
| `infra/postgres/init/01-init.sql` | MODIFIED (small) | Two options surfaced in §3 above. Requires explicit user decision in `sdd-propose`. **May not need to change** if we accept that queen doesn't own `cachicamas_pg` and we use `ALTER ROLE queen SET timezone = 'UTC'` instead. |
| `openspec/specs/db-migrations/spec.md` | NEW | The "hello world" capability spec — covers embed.FS, queen identity, idempotency, advisory locking, recovery. |
| `openspec/specs/db-migrations/schema/spec.md` | NEW (deferred) | The future per-context schema migration specs (catalog, identity, observability). NOT in this change. |

---

## 6. Out of scope (explicit, per `openspec/config.yaml` rules)

These are "deferred but related" and should appear by name in the proposal:

1. **Least-privilege app role** (`cachicamas_app` recipe in `01-init.sql` lines 109–121). Out of this change.
2. **Backups / WAL archiving / logical replication**. Out.
3. **Bounded-context schemas** beyond what `01-init.sql` already provisioned. Out — once goose runs, the `catalog`, `identity`, `observability` schemas are empty placeholders; populating them is a separate concern.
4. **DDL linting / drift detection** (e.g., `sqlfluff`, `atlas schema diff` in CI). Out.
5. **Online schema changes** (`pg_repack`, `pgroll`, `CREATE INDEX CONCURRENTLY` choreography). Out.
6. **Multi-database migrations** (one runner for many DBs). Out — one runner, one DB (`cachicamas_pg`).
7. **Migration tests in CI** beyond the `runner_test.go` integration test. Out of v1.
8. **Reverse-engineering current DB state** into a baseline migration. Out — we start from an empty `schema_migrations` table.

---

## 7. Open questions for the user (resolve before `sdd-propose`)

1. **Ownership of `cachicamas_pg`**: queen doesn't own the DB today (the image's superuser `cachicamas` does, via `POSTGRES_USER`). To use `ALTER DATABASE ... SET timezone`, queen must own the DB. Three options:
   - **(a)** Modify `01-init.sql` to add `ALTER DATABASE current_database() OWNER TO queen;` (one line, runs as superuser during initdb). Cleanest.
   - **(b)** Use `ALTER ROLE queen SET timezone = 'UTC'` instead. Works today without any init change, but only covers queen's sessions. Tomorrow's app role needs its own ALTER ROLE.
   - **(c)** Have queen create a NEW database (`cachicamas_app_db`) and run the service against that. Bigger surgery.

   **Recommendation: (a)**. Smallest change, future-proof.

2. **Postgres driver**: `jackc/pgx/v5` (recommended by the pgx README for Postgres-only stacks, has a stdlib `database/sql` adapter) vs `lib/pq` (deprecated since 2024). **Recommendation: pgx/v5.**

3. **Migration runner timing**: run migrations on every container start (idempotent, safe, slow if 100+ migrations) vs a separate init container that runs once and exits (fast startup, but extra container to manage). **Recommendation: on every start** — idempotency is built in via the bookkeeping table; the slowness doesn't apply yet.

4. **Migration failure behavior**: if the migration fails on startup, should the container crash (fail-fast, surface the problem in `docker compose logs`) or start Echo anyway and serve 500s (so health probes can detect it)? **Recommendation: crash.** A failed migration is an unrecoverable operator error; fail-fast beats silent corruption.

5. **Down migrations**: required or optional? goose supports both. Industry norm is "don't write down migrations in production code" (Linear, GitHub, etc.) because data migrations don't roll back cleanly. **Recommendation: write down migrations for schema-only changes, leave a `// TODO: data migration` comment for irreversible data moves.**

6. **Multiple migration dirs**: one global dir (`src/migration/sql/`) or per-context dirs (`src/migration/sql/catalog/`, `src/migration/sql/identity/`, etc.)? **Recommendation: one dir, schema-qualified names** (e.g., `20260622_catalog_create_products.sql`). Simpler tooling, easier CI ordering.

---

## 8. Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| `goose` releases a breaking change before we hit v1.0 of the migration runner. | Low | Pin to `v3.27.1` exactly. Re-pin on a deliberate bump PR. |
| `embed.FS` doesn't include newly-added migration files in the built binary. | Medium | Add a `make migrate/lint` target that opens the embed.FS and asserts every filename is a valid goose migration name (regex `^\d{14}_[a-z0-9_]+\.sql$`). Catches forgotten migrations in PR review. |
| `queen` loses ownership of `cachicamas_pg` (e.g., someone re-runs initdb with a different POSTGRES_USER). | Low | Add a startup precondition check in the runner: `SELECT current_user, current_database(), (SELECT rolname FROM pg_database JOIN pg_roles ON pg_roles.oid = pg_database.datdba WHERE datname = current_database())` and fail-fast if rolname != 'queen'. |
| A migration locks the DB for too long (e.g., `ALTER TABLE ADD COLUMN ... DEFAULT ...` on a large table). | Medium (future) | Document in `README.md` that all schema migrations must use Postgres-native online ops (`CREATE INDEX CONCURRENTLY`, `ADD COLUMN ... DEFAULT NULL` then backfill). Add a pre-commit hook (out of v1) to flag `ALTER TABLE ... ADD COLUMN ... DEFAULT` patterns. |
| `goose` blocking advisory lock causes a deadlock if the runner's connection drops mid-migration. | Low | goose uses `pg_try_advisory_lock` (not `pg_advisory_lock`), so a dropped connection releases the lock automatically (session-scoped lock). The `dirty` version is a goose concept we don't use — goose's model is "if version N's tx rolled back, we manually fix". Document the recovery procedure in `README.md`. |
| Two `database_administrator` replicas start at the same time and both try to run migrations. | Medium (future) | goose's `WithSessionLocker` serializes them. We always enable the locker in `runner.go` (not opt-in). |
| Tests assume a running Postgres but CI doesn't have one. | Medium | Integration tests gate on `t.Skip("integration")` when `INTEGRATION=1` env var isn't set. Default `make test` stays integration-free. |

---

## 9. Ready for Proposal

**Yes**, with the open questions in §7 answered. The proposal should:
- State the chosen library (`pressly/goose v3.27.1`) and the rejected alternatives in one section.
- Surface open question #1 (DB ownership) as an explicit user decision with three options.
- Surface open question #4 (fail-fast on migration error) as a decision: we recommend crash.
- ADR for the new top-level deps (`pressly/goose/v3`, `jackc/pgx/v5`).
- Reference this explore artifact and the existing `infra/postgres/init/01-init.sql` line 99–105 (`schema_migrations` placeholder table).

---

## Review checklist (for reviewers)

- [ ] reviewer can confirm that no file under `backend/database_administrator/src/` was modified by this explore
- [ ] reviewer can confirm that `docker-compose.yaml` and `infra/postgres/init/01-init.sql` were not modified by this explore
- [ ] reviewer can confirm that the library comparison matrix cites verified sources (goose v3.27.1 release date, golang-migrate v4.19.1 release date, atlas dual license) and flags unverified items
- [ ] reviewer can confirm the `ALTER DATABASE ... SET timezone` SQL is justified by citing the PGC_USERSET classification of `TimeZone` in PG 18 docs
- [ ] reviewer can confirm the embed.FS strategy is verified against goose's official README
- [ ] reviewer can confirm the recommendation of timestamp-prefixed naming is backed by a trade-off table
- [ ] reviewer can confirm the out-of-scope list is explicit and matches `openspec/config.yaml` rules
- [ ] reviewer can confirm the open questions are answerable without re-investigation
- [ ] reviewer can confirm the risks table covers embed.FS staleness, DB ownership loss, advisory lock behavior, and replica concurrency

## Incompleteness log entry

- [2026-06-21] [openspec/specs/] missing — `openspec/specs/` directory is empty (the `sdd-init` artifact references it as "populated as changes land" but no specs exist yet). Proposed fix: the first change to land a spec (`cachicamas-tail-sampling` is in flight) will seed the directory; this explore does not fix it.

## Sources (all verified via WebFetch on 2026-06-21)

- github.com/pressly/goose/releases — v3.27.1 dated 2026-04-24
- github.com/pressly/goose README — `goose.SetBaseFS(embedMigrations)` pattern
- github.com/pressly/goose lock/postgres.go — `SELECT pg_try_advisory_lock($1)`
- github.com/golang-migrate/migrate/releases — v4.19.1 dated 2025-11-29
- github.com/golang-migrate/migrate database/postgres/postgres.go — `SELECT pg_advisory_lock($1)` (no timeout)
- github.com/golang-migrate/migrate source/iofs — `iofs.New(embed.FS, "migrations")` pattern
- github.com/ariga/atlas/releases — CLI v1.2.0 dated 2026-04-10 (dual-licensed)
- github.com/ariga/atlas-go-sdk — v0.7.2 dated 2025-05-21 (archived, use atlasexec)
- github.com/jackc/pgx — v5 latest, MIT license
- www.postgresql.org/docs/18/sql-alterdatabase.html — owner-or-superuser for session defaults; PGC_USERSET parameters settable by owner
- www.postgresql.org/docs/18/sql-alterrole.html — owner can SET own role; ALL is superuser-only
- www.postgresql.org/docs/18/sql-altsystem.html — superuser-only
- www.postgresql.org/docs/18/runtime-config.html — TimeZone classified as PGC_USERSET
- www.postgresql.org/docs/18/sql-createtable.html — transactional DDL supported (CREATE TABLE / ALTER TABLE inside BEGIN..COMMIT)
- github.com/riverqueue/river — job queue (not migration; rivermigrate is a sub-package)
