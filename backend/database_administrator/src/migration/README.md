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

---

## 11. Schema overview — witsaba core domain

The witsaba framework models the lifecycle of a product requirement as
8 tables in `public`, landed by 3 goose migration files in dependency
order. Every table carries `created_at` and `updated_at` (`TIMESTAMPTZ
NOT NULL DEFAULT now()`) and uses `BIGSERIAL` synthetic primary keys,
with one deliberate exception.

### 11.1 The 8 tables, in dependency order

```
organization                 (root, no parents)
   |
   +-- project                (1:N; project.organization_id FKs to organization.id)
         |
         +-- requirement      (1:N; requirement.project_id FKs to project.id)
               |
               +-- requirement_spike   (1:N; multi-spike per requirement)
               |
               +-- milestone  (1:1; STRICT INHERITANCE -- PK = FK = requirement_id)
                     |
                     +-- task           (1:N; task.milestone_id FKs to milestone.requirement_id)
                           |
                           +-- spec      (1:N; spec.task_id FKs to task.id)
                                 |
                                 +-- spec_phase  (1:N, append-only history;
                                                  spec_phase.spec_id FKs to spec.id)
```

### 11.2 Cardinality — strict inheritance is the EXCEPTION

`milestone` is the ONLY table that uses strict inheritance
(`requirement_id BIGINT PRIMARY KEY REFERENCES requirement(id)`,
no synthetic `id`). The PK = FK design lets Postgres enforce the
1:1 invariant for free: a second milestone for the same
`requirement_id` fails on `milestone_pkey`. Everywhere else, the
child has a synthetic `BIGSERIAL PRIMARY KEY` plus a non-PK FK,
which lets the same parent accumulate multiple children.

Why this asymmetry: a requirement can be re-investigated (`spike`)
and a milestone can fan out into many tasks, but a requirement has
exactly one milestone for its lifecycle. The strict-inheritance
invariant captures that 1:1 in the schema instead of in app code.

### 11.3 Append-only by default

The framework's default convention is **append-mostly**: history is
the source of truth, and in-place `UPDATE` is allowed ONLY on
`organization.is_active`. The column comment on
`organization.is_active` is the documented contract:

```sql
COMMENT ON COLUMN organization.is_active IS
    'UPDATE-in-place: this column is the ONLY mutation allowed on organization rows.';
```

Any data-access layer change (`witsaba-requirements-api` and friends)
MUST be reviewed against this contract. Concretely:

- `UPDATE organization SET <col> ...` -- only legal when `<col> = is_active`.
- `UPDATE project SET ...`, `UPDATE requirement SET ...`, etc. --
  illegal in v1; append a new row or create a new child row instead.

**DB-level enforcement** (REVOKE UPDATE on append-only tables,
BEFORE UPDATE triggers) is intentionally deferred to follow-up
changes so the v1 schema stays easy to evolve:

- `witsaba-core-tables-append-only-enforcement` (proposed).
- `witsaba-core-tables-project-milestone` (proposed; the 1:N escape
  hatch for the strict-inheritance `milestone`).

### 11.4 The 8 `spec_phase.phase` values (locked)

The CHECK constraint on `spec_phase.phase` enforces EXACTLY these 8
values (no more, no fewer). Adding a ninth phase requires an
`ALTER TABLE ... DROP CONSTRAINT ... ADD CONSTRAINT ... CHECK (...)`
migration; it is intentionally NOT a free-for-all enum extension.

| Value                   | Meaning                                            |
| ----------------------- | -------------------------------------------------- |
| `tdd_red`               | Writing failing tests FIRST                        |
| `implementation`        | Making the tests pass                              |
| `tdd_green`             | Tests now green; refactor pass begins              |
| `verify`                | Full test suite + lint + build green               |
| `pr`                    | PR opened against `main`                           |
| `technical_ai_review`   | AI peer review (architecture, complexity, tests)   |
| `ai_approved`           | AI reviewer signed off                             |
| `human_approved`        | Human maintainer signed off -- PR is mergeable     |

### 11.5 The 256 KiB PRD cap

`requirement.content` is capped at 256 KiB via the CHECK constraint
`requirement_content_size_cap CHECK (octet_length(content) <= 262144)`.
A future PR that needs more room must:

```sql
ALTER TABLE requirement DROP CONSTRAINT requirement_content_size_cap;
ALTER TABLE requirement ADD CONSTRAINT requirement_content_size_cap
    CHECK (octet_length(content) <= <new_cap>);
```

The column comment on `requirement.content` carries the rationale
and the current limit so contributors see the cap before they hit it.

### 11.6 Agent-first re-entry pattern (canonical SQL)

`spec_phase` is append-only history. An agent's transition algorithm
on a phase event is:

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

Key invariants the agent's transition algorithm relies on:

- The `UNIQUE (spec_id, phase, started_at)` constraint lets the
  agent re-enter an earlier phase (e.g., `tdd_red` after
  `technical_ai_review`) without losing the prior entry, as long as
  the new row's `started_at` differs.
- The `idx_spec_phase_current_state` partial index makes the "what
  phase is this spec in right now?" query a single B-tree seek.
- The `notes` column is the agent's reasoning for each transition.
  Filling it in is the agent-first contract; it makes the audit
  trail legible to humans AND to the agent on subsequent passes.

### 11.7 What is NOT in v1 (follow-up changes)

The following are intentionally deferred to keep the v1 schema easy
to evolve:

- **Partial UNIQUE index** on `(spec_id) WHERE ended_at IS NULL` --
  would prevent two open phases for the same spec at the DB level.
  Deferred to `witsaba-core-tables-append-only-enforcement` (see
  design §6.2 for the rationale on app-layer discipline first).
- **DB-level REVOKE UPDATE** on append-only tables from the `queen`
  role. Same follow-up change.
- **`project_milestone`** table for the "deliverable per release
  channel" cardinality escape hatch. Deferred to
  `witsaba-core-tables-project-milestone`.
