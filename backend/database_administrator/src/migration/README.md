# Migration runner

The `database_administrator` service runs SQL migrations on every container
start using [`pressly/goose` v3.27.1](https://github.com/pressly/goose) over
the [`jackc/pgx/v5`](https://github.com/jackc/pgx) stdlib adapter. SQL files
are embedded into the binary at build time via `//go:embed`; the runner
acquires a Postgres advisory lock so multiple replicas can boot
concurrently without double-applying migrations.

This README is the operator-facing guide. It assumes you have:

- a working `cachicamas` checkout with the `database_administrator` service
  built and runnable via `docker compose up -d --build database_administrator`,
- shell access to the `cachicamas-postgres` container (via
  `docker compose exec postgres psql -U ... -d cachicamas_pg`).

If you are writing a new migration, also read
[`docs/coding/SCHEMA-CHANGES.md`](../../../../../docs/coding/SCHEMA-CHANGES.md)
when it lands (see Phase 8 of the SDD tasks).

---

## 1. Naming

Migrations are single SQL files under `sql/` with a **14-digit timestamp
prefix** and a **lowercase snake_case context** + **description**:

```
YYYYMMDDHHMMSS_<context>_<description>.sql
```

Regex enforced at code review:

```
^\d{14}_[a-z0-9_]+\.sql$
```

Examples:

- `20260621120000_hello_world.sql`
- `20260622_catalog_create_products.sql`
- `20260701_identity_add_users_email_index.sql`

The timestamp is the **migration version** and must be unique across the
project. Pick a future timestamp; if two PRs land in the same minute, the
loser must renumber (the runner sorts lexicographically, so
`20260621120001` > `20260621120000` regardless of the wall clock).

---

## 2. File shape — goose v3.27.1 single-file idiom

**Each migration is ONE `.sql` file with both `-- +goose Up` and
`-- +goose Down` blocks separated by `-- +goose StatementEnd` and
`-- +goose StatementBegin` directives.**

```sql
-- +goose Up
-- +goose StatementBegin
CREATE TABLE catalog.products (
    id         BIGSERIAL    PRIMARY KEY,
    sku        TEXT         NOT NULL UNIQUE,
    name       TEXT         NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE catalog.products;
-- +goose StatementEnd
```

The legacy v2 paired `XXX.sql` + `XXX.down.sql` files are **rejected** by
goose v3.27.1 with `found duplicate migration version` because both files
share the numeric prefix. Do not create the `.down.sql` variant.

### 2.1 Idiom conventions

- **Schema-only change** (CREATE/ALTER/DROP): always include a Down block
  that reverses the change.
- **Data move** (UPDATE/DELETE/INSERT bulk): include a Down block only if
  the move is logically reversible; otherwise leave a
  `-- TODO: data migration, down is destructive` comment above the Down
  block. The Down block can be a no-op (`SELECT 1;`).
- **Online operations** (large backfills, index creation): wrap in
  `StatementBegin`/`StatementEnd` and consider `-- +goose StatementBreak`
  if the migration must run in two transactions (rare).

### 2.2 Generating a new migration

Two options:

**Option A — use the goose CLI** (generates the v3 single-file shape):

```bash
cd backend/database_administrator
go run github.com/pressly/goose/v3/cmd/goose@latest -dir sql create <name> sql
```

This produces `sql/<timestamp>_<name>.sql` with `-- +goose Up` and
`-- +goose Down` blocks ready to fill in.

**Option B — write it by hand** (copy the template above and edit).

Then:

1. Edit the Up and Down blocks with the actual SQL.
2. Commit the file under `backend/database_administrator/src/migration/sql/`.
3. Open a PR. The next `database_administrator` container boot will apply
   the migration.

The runner picks up the new file at build time (because of `//go:embed`)
— no extra wiring, no config change, no rebuild of the compose stack's
infrastructure.

---

## 3. Bookkeeping table

The runner uses `public.schema_migrations` (overriding goose's default
`goose_db_version`). The table is provisioned by
`infra/postgres/init/01-init.sql` with the goose v3.27.1 expected shape:

```
 Column    |           Type           | Nullable |       Default
-----------+--------------------------+----------+-----------------------
 id        | bigint                   | not null | nextval(..._id_seq)
 version_id| bigint                   | not null |  (UNIQUE)
 is_applied| boolean                  | not null |
 tstamp    | timestamp with time zone | not null | now()
```

- **Owner:** `queen` (not the cluster superuser). Without this, the
  runner — which connects as `queen` — fails the first INSERT with
  `ERROR: permission denied for table schema_migrations`.
- **Zero-version row:** `id=1, version_id=0, is_applied=f` is seeded by
  `01-init.sql`. Goose v3 refuses to start with an empty
  `schema_migrations`; pre-seeding the zero row is what makes
  `01-init.sql` authoritative.
- **Index:** `schema_migrations_version_id_key` (UNIQUE) on `version_id`,
  keeps `SELECT MAX(version_id)` fast as history grows.

To inspect the current state from inside the postgres container:

```bash
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "SELECT id, version_id, is_applied, tstamp FROM public.schema_migrations ORDER BY id"
```

---

## 4. Concurrency — advisory lock

The runner uses `lock.NewPostgresSessionLocker()`, which calls
`pg_try_advisory_lock(42)` on a fixed integer key. The lock is
**non-blocking** (the `_try` variant):

- If the lock is free, the runner acquires it and proceeds.
- If another replica holds the lock, the runner **polls** (default
  backoff in goose v3: 500ms initial, 30s max, with jitter) until the
  lock is released.

When the migration finishes (or the runner exits / crashes), the lock
is released automatically. A `pg_advisory_unlock_all(42)` is called as
a defensive cleanup.

To observe the lock from the postgres container while two replicas
race:

```bash
# Replica A
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "SELECT pid, locktype, granted FROM pg_locks WHERE locktype = 'advisory'"

# Replica B
docker compose exec database_administrator ./bin/database_administrator
# Logs will show: "migration.up waiting for lock" then proceed when A finishes.
```

The integration test `TestRunner_AdvisoryLock_PreventsDoubleApply` in
`src/migration/runner_test.go` exercises this path with two parallel
goroutines; the runner guarantees only one applies the migration.

---

## 5. OTel + slog instrumentation

Every `Up` call emits:

1. **One OTel span** named `migration.up` (kind: `Internal`).
2. **One `slog` line** at INFO on success or ERROR on failure.

Agreed attributes on the span AND the log record:

| Attribute | Type | Value | When |
|---|---|---|---|
| `db.system` | string | `"postgresql"` | always |
| `migration.dir` | string | `MIGRATION_DIR` value (default `"sql"`) | always |
| `migration.table` | string | `MIGRATION_TABLE` value (default `"schema_migrations"`) | always |
| `migration.duration_ms` | int | wall-clock duration of the Up call | always |
| `migration.applied_count` | int | number of migrations applied | on success |
| `migration.error` | string | error message | on failure |
| `migration.error.kind` | string | one of `pg_advisory_lock_timeout`, `pg_connect_error`, `pg_query_error`, `embed_error` | on failure |

The OTel trace context is propagated through slog via
`otelslog.NewHandler` (see `src/otel/logging.go`); every log record
carries `trace_id` and `span_id` automatically. Find the trace ID in:

```bash
docker compose logs otel-collector | grep "Trace ID"
```

…then paste the hex string into the Jaeger UI (`http://localhost:16686`).

> **Note on current trace export:** the OTel Go SDK has an upstream bug
> ([opentelemetry-go#5248](https://github.com/open-telemetry/opentelemetry-go/issues/5248))
> that drops ~9/10 spans at v1.44.0. The PR's OTel emission is correct
> (spans are created and have real trace IDs); they just may not reach
> Jaeger. Workaround: use the `trace_id` in the collector's debug log
> output to find the matching span context. See
> `infra/otel/collector-config.yaml` lines 40-45.

---

## 6. Environment variables

| Var | Default | Effect |
|---|---|---|
| `MIGRATION_TIMEOUT` | `30s` | `context.WithTimeout` for the `Up` call. Bounded so a stuck migration doesn't block the container forever. |
| `MIGRATION_TABLE` | `schema_migrations` | Bookkeeping table name. Override only if you know what you're doing. |
| `MIGRATION_DIR` | `sql` | Subdirectory inside the embed.FS that holds the migrations. |
| `DATABASE_URL` | unset | Postgres DSN. Wins over the discrete env vars. |
| `POSTGRES_HOST` | unset | Discrete env: `host[:port]` |
| `POSTGRES_PORT` | `5432` | Discrete env |
| `POSTGRES_DB` | unset | Discrete env: database name |
| `POSTGRES_USER` | unset | Discrete env: role name (the runner connects as `queen`) |
| `POSTGRES_PASSWORD` | unset | Discrete env: role password |

If neither `DATABASE_URL` nor the discrete env vars are set, the driver
fails fast at boot with an error from `LoadConfigFromEnv`.

---

## 7. Failure modes and recovery

### 7.1 Migration errors (e.g. syntax error, constraint violation)

The runner returns the error from `goose.Up`. `main.go` emits `slog.Error`
and calls `os.Exit(1)`. The container orchestrator restarts the
container — the migration is retried on the next boot. If the SQL is
broken, this is an infinite restart loop. Fix the SQL, rebuild, push.

### 7.2 Half-applied migration

Goose v3 **does not** use a `dirty` column (the v2 `is_applied`+`tstamp`
two-row bookkeeping was retired). When a migration's body fails, the
**transaction is rolled back** by goose; the row in
`public.schema_migrations` is NOT inserted. The next boot re-attempts
the migration from scratch.

To verify the transaction rolled back:

```bash
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "SELECT version_id, is_applied, tstamp FROM public.schema_migrations
   WHERE version_id = <the-failed-version-id>"
```

If the row is missing → the transaction rolled back cleanly → the next
boot will re-apply it. If the row IS present but the schema is
incomplete (e.g. a partial `CREATE TABLE` somehow committed), the
operator must manually clean up:

```bash
# 1. Inspect the partial state
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c "\d+ <schema>.<object>"

# 2. Repair the schema (drop partial object, etc.)

# 3. Remove the bookkeeping row so the next boot re-applies
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "DELETE FROM public.schema_migrations WHERE version_id = <the-failed-version-id>"
```

### 7.3 Lost connection mid-migration

The advisory lock (`pg_advisory_unlock_all`) is released automatically
when the connection is severed. The next replica's `Up` call will
acquire the lock and proceed.

### 7.4 Recovery from "stuck" lock

If a previous replica's connection was severed but the lock is somehow
held (e.g. another long-running session has it), inspect:

```bash
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "SELECT pid, locktype, granted, classid FROM pg_locks WHERE locktype = 'advisory'"
```

The lock key is 42 (single-int, not dual-int). Force-release:

```bash
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "SELECT pg_advisory_unlock(42)"
```

### 7.5 Container restart loop (fatal migration error)

If a migration is broken in a way that crashes the runner on every
boot, the container will keep restarting. The docker-compose logs will
show the same error repeatedly. To recover:

1. Revert the broken migration in `git`.
2. Rebuild: `docker compose up -d --build database_administrator`.
3. The new container will boot, see the broken migration file removed,
   and run the rest of the migrations successfully.
4. Re-apply the corrected migration in a new PR.

---

## 8. Operational boundaries

### 8.1 `MIGRATION_TIMEOUT` (default 30s)

The migration context is bounded. A migration that takes longer than
`MIGRATION_TIMEOUT` will be cancelled and the runner will return an
error. **Do not** write migrations that exceed this window without
raising the timeout.

For long-running operations, use **online ops**:

- `CREATE INDEX CONCURRENTLY` (not `CREATE INDEX`)
- `ALTER TABLE ... ADD COLUMN ... DEFAULT NULL` then backfill in a
  second migration, then `ALTER COLUMN ... SET DEFAULT ...`
- `ALTER TABLE ... ALTER COLUMN ... TYPE ... USING ...` with batching
  for big tables

These are safe to run while the application is serving traffic and can
take minutes or hours without holding a long transaction.

### 8.2 Single global directory

All migration files live in `src/migration/sql/`. **No subdirectories**
are traversed (verified against goose v3.27.1 source). Schema-qualified
table names (`catalog.products`, `identity.users`) are how bounded
contexts are separated; the directory layout is intentionally flat.

Rationale (per design §8): keeping all migrations in one directory makes
it trivial to reason about "what's been applied". Schema-qualified names
keep the bounded contexts logically separate without physical file
grouping.

### 8.3 One database, one runner

The runner expects to own the `public.schema_migrations` table. Do not
introduce a second migration system (Atlas, Flyway, Liquibase) in the
same database. The advisory lock is per-DB, not per-tool.

---

## 9. Inspection commands (cheat sheet)

```bash
# Current state of the bookkeeping table
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "SELECT id, version_id, is_applied, tstamp FROM public.schema_migrations ORDER BY id"

# Schema shape
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "\d public.schema_migrations"

# Owner of the table
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "SELECT relname, pg_get_userbyid(relowner) AS owner
   FROM pg_class WHERE relname = 'schema_migrations'"

# Active advisory locks
docker compose exec postgres psql -U cachicamas -d cachicamas_pg -c \
  "SELECT pid, locktype, granted FROM pg_locks WHERE locktype = 'advisory'"

# Timezone for any user/database
docker compose exec postgres psql -U <user> -d cachicamas_pg -c "SHOW timezone"

# Last migration's OTel trace ID (in collector's debug output)
docker compose logs otel-collector | grep "migration.up applied" | grep -oE 'Trace ID: [a-f0-9]+'
```

---

## 10. References

- **ADR-001** — `jackc/pgx/v5` driver choice. Engram id 1588.
- **ADR-002** — `pressly/goose v3` with `embed.FS`. Engram id 1589.
- **Proposal** — `openspec/changes/postgres-database-migrations/proposal.md`.
- **Spec** — `openspec/changes/postgres-database-migrations/specs/db-migrations/spec.md` (12 requirements, 23 scenarios).
- **Design** — `openspec/changes/postgres-database-migrations/design.md`.
- **Tasks** — `openspec/changes/postgres-database-migrations/tasks.md`.
- **Upstream bug** — `opentelemetry-go#5248` blocks end-to-end OTel trace
  export to Jaeger. Tracked in `infra/otel/collector-config.yaml` lines 40-45.
