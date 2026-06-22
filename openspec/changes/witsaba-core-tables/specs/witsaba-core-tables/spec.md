# Spec: witsaba-core-tables

> Source of truth: behavioral requirements and scenarios for the
> `witsaba-core-tables` change. Locks the 8-table schema introduced by 3
> goose migration files in `backend/database_administrator/src/migration/sql/`,
> plus the agent-first re-entry pattern for `spec_phase`.

## Conventions

- All scenarios use Given / When / Then.
- All table names are unqualified (live in `public`, the runner's default schema).
- All timestamps are `TIMESTAMPTZ` in UTC (server-enforced via `infra/postgres/init/01-init.sql`).
- Integer identifiers are 64-bit (`BIGSERIAL`).
- Each scenario MUST be implementable as an integration test in
  `backend/database_administrator/src/migration/runner_test.go`, gated on
  `INTEGRATION=1` per the existing `integrationRunnerDB(t)` helper.
- All RFC 2119 keywords (`MUST`, `MUST NOT`, `SHALL`, `SHOULD`) carry their
  standard meanings per `openspec/AGENTS.md`.

## Layout decision

ONE spec file (this one) for the whole change. The 8 tables form a single
coherent domain — the witsaba framework core — joined by a strict FK chain
that must land together or not at all. Splitting into 3 bounded-context
specs would duplicate the boot/migration runner scenarios and obscure the
dependency order that the proposal locks.

---

## B. Boot and migration runner

The runner is the production-grade `pressly/goose` v3.27.1 wrapper from
`backend/database_administrator/src/migration/`. The 3 new migrations land
under `src/migration/sql/` using the existing `^\d{14}_[a-z0-9_]+\.sql$`
filename convention, in dependency order:
`20260622120000_orgs_and_projects.sql`, `20260622120001_requirements_and_milestones.sql`,
`20260622120002_tasks_and_specs.sql`.

### B1. Fresh apply

- **GIVEN** an empty `public.schema_migrations` table
- **WHEN** `service.Up(ctx)` is called for the first time against a clean DB
- **THEN** exactly 4 rows appear in `public.schema_migrations` with
  `version_id` values `20260621120000`, `20260622120000`, `20260622120001`,
  `20260622120002` in that order, all `is_applied = TRUE`.

### B2. Re-run after success is a no-op

- **GIVEN** a DB that has already applied all 4 migrations cleanly
- **WHEN** `service.Up(ctx)` is called a second time
- **THEN** zero migrations are applied (no errors, no new bookkeeping rows).

### B3. Failed migration rolls back its transaction

- **GIVEN** a fresh DB and a migration file whose `-- +goose Up` body raises
  a constraint violation (e.g., `CREATE TABLE duplicate (id INT PRIMARY KEY);`
  twice in one Up block)
- **WHEN** `service.Up(ctx)` runs against that file
- **THEN** the failed `Up` returns an error AND `public.schema_migrations`
  has NO row for that version (the transaction rolled back atomically).

---

## O. `organization` (root)

### O1. `is_active` is the only UPDATE-in-place column

- **GIVEN** the `organization` table is migrated and contains one row with
  `is_active = TRUE`
- **WHEN** an agent queries `col_description('organization'::regclass, <attnum_of_is_active>)`
- **THEN** the column comment explicitly declares that `is_active` is the
  only column allowed to receive an `UPDATE` on `organization` rows.

### O2. `identification` is UNIQUE

- **GIVEN** the `organization` table with one row whose `identification = 'RFC-XYZ-001'`
- **WHEN** a second `INSERT INTO organization (full_name, identification) VALUES ('Acme 2', 'RFC-XYZ-001')`
- **THEN** the insert fails with a unique-constraint violation on
  `organization_identification_key` (or equivalent).

### O3. `full_name` is NOT NULL

- **GIVEN** the `organization` table
- **WHEN** `INSERT INTO organization (full_name, identification) VALUES (NULL, 'NEW-ID')`
- **THEN** the insert fails with a NOT NULL violation on `full_name`.

### O4. `shortname`, `email`, `phone` are nullable

- **GIVEN** the `organization` table
- **WHEN** `INSERT INTO organization (full_name, identification) VALUES ('Acme', 'A1')`
  (omitting `shortname`, `email`, `phone`)
- **THEN** the insert succeeds and the new row has `NULL` in those three columns.

---

## P. `project`

### P1. `key` is UNIQUE

- **GIVEN** the `project` table with one row whose `key = 'cachicamas'`
- **WHEN** a second `INSERT INTO project (organization_id, key, full_name) VALUES (1, 'cachicamas', 'Dup')`
- **THEN** the insert fails with a unique-constraint violation on `key`.

### P2. `metadata` accepts arbitrary JSON

- **GIVEN** a valid `organization_id` and the `project` table
- **WHEN** an `INSERT` supplies `metadata` as an object, an array, a primitive,
  and the JSON literal `'null'` respectively in four separate rows
- **THEN** all four inserts succeed; Postgres accepts any valid JSON value
  for a `JSONB` column.

### P3. `metadata` can be NULL

- **GIVEN** a valid `organization_id` and the `project` table
- **WHEN** `INSERT INTO project (organization_id, key, full_name) VALUES (1, 'no-meta', 'No Metadata')`
  (omitting `metadata`)
- **THEN** the insert succeeds and the new row stores `metadata IS NULL`.

### P4. FK to `organization` is enforced

- **GIVEN** the `project` table with no row whose `organization_id = 99999`
- **WHEN** `INSERT INTO project (organization_id, key, full_name) VALUES (99999, 'orphan', 'Orphan')`
- **THEN** the insert fails with a foreign-key violation referencing
  `organization(id)`.

---

## R. `requirement`

### R1. `content` 256 KiB cap is enforced (over)

- **GIVEN** a valid `project_id` and the `requirement` table
- **WHEN** `INSERT INTO requirement (project_id, filename, content) VALUES (1, 'big.md', repeat('x', 262145))`
- **THEN** the insert fails the `requirement_content_size_cap` CHECK constraint
  with `octet_length(content) <= 262144`.

### R2. `content` 256 KiB cap is allowed (boundary)

- **GIVEN** a valid `project_id` and the `requirement` table
- **WHEN** `INSERT INTO requirement (project_id, filename, content) VALUES (1, 'edge.md', repeat('x', 262144))`
- **THEN** the insert succeeds (`octet_length = 262144` satisfies `<=`).

### R3. `content` of 1 byte is allowed

- **GIVEN** a valid `project_id` and the `requirement` table
- **WHEN** `INSERT INTO requirement (project_id, filename, content) VALUES (1, 'tiny.md', 'x')`
- **THEN** the insert succeeds.

### R4. `filename` is NOT NULL

- **GIVEN** a valid `project_id` and the `requirement` table
- **WHEN** `INSERT INTO requirement (project_id, filename, content) VALUES (1, NULL, 'body')`
- **THEN** the insert fails with a NOT NULL violation on `filename`.

### R5. FK to `project` is enforced

- **GIVEN** the `requirement` table with no project of `id = 99999`
- **WHEN** `INSERT INTO requirement (project_id, filename, content) VALUES (99999, 'orphan.md', 'x')`
- **THEN** the insert fails with a foreign-key violation referencing
  `project(id)`.

### R6. `analysis_result`, `git_repository_url`, `is_technically_viable` are nullable

- **GIVEN** a valid `project_id` and the `requirement` table
- **WHEN** an `INSERT` provides `filename` and `content` but omits
  `analysis_result`, `git_repository_url`, and `is_technically_viable`
- **THEN** the insert succeeds and all three columns are `NULL`.

---

## S. `requirement_spike`

### S1. `ended_at >= started_at` enforced

- **GIVEN** the `requirement_spike` table with a valid `requirement_id`
- **WHEN** `INSERT INTO requirement_spike (requirement_id, started_at, ended_at) VALUES (1, DATE '2026-06-15', DATE '2026-06-10')`
- **THEN** the insert fails the `requirement_spike_dates_valid` CHECK constraint.

### S2. `started_at` and `ended_at` are nullable (open-ended spike)

- **GIVEN** a valid `requirement_id` and the `requirement_spike` table
- **WHEN** `INSERT INTO requirement_spike (requirement_id) VALUES (1)`
  (omitting both dates)
- **THEN** the insert succeeds; the new spike has `started_at IS NULL AND ended_at IS NULL`.

### S3. FK to `requirement` enforced; multiple spikes per requirement allowed

- **GIVEN** the `requirement_spike` table with no requirement of `id = 99999`
- **WHEN** a first insert against `requirement_id = 99999` is attempted
- **THEN** the insert fails with a foreign-key violation.
- **AND WHEN** two distinct inserts succeed against the same valid
  `requirement_id = 1` (each with its own synthetic `id`)
- **THEN** both rows exist and the FK from `requirement_spike.requirement_id`
  to `requirement(id)` is intact (1:N cardinality verified).

---

## M. `milestone` (STRICT INHERITANCE — PK = FK = `requirement_id`)

### M1. `requirement_id` is the ONLY PK column (no synthetic `id`)

- **GIVEN** the `milestone` table
- **WHEN** `SELECT a.attname FROM pg_index i JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey) WHERE i.indrelid = 'milestone'::regclass AND i.indisprimary`
- **THEN** the result set contains exactly one row, `requirement_id`. There
  is NO `id` column on `milestone` (confirmed via `\d milestone`).

### M2. One milestone per requirement (1:1 cardinality)

- **GIVEN** the `requirement` table with `id = 1`
- **WHEN** `INSERT INTO milestone (requirement_id, title) VALUES (1, 'M1')`
- **THEN** the insert succeeds and `SELECT count(*) FROM milestone WHERE requirement_id = 1 = 1`.

### M3. Second milestone for same `requirement_id` fails

- **GIVEN** `milestone` already has one row with `requirement_id = 1`
- **WHEN** `INSERT INTO milestone (requirement_id, title) VALUES (1, 'M1 dup')`
- **THEN** the insert fails with a primary-key violation on `milestone.pkey`.

### M4. `end_date >= start_date` enforced when both non-NULL

- **GIVEN** the `milestone` table
- **WHEN** `INSERT INTO milestone (requirement_id, title, start_date, end_date) VALUES (1, 'M reversed', DATE '2026-06-15', DATE '2026-06-10')`
- **THEN** the insert fails the `milestone_dates_valid` CHECK constraint.

### M5. `start_date` and `end_date` are nullable

- **GIVEN** a valid `requirement_id` and the `milestone` table
- **WHEN** `INSERT INTO milestone (requirement_id, title) VALUES (1, 'open-ended')`
- **THEN** the insert succeeds; `start_date IS NULL AND end_date IS NULL`.

---

## T. `task`

### T1. `title` is NOT NULL

- **GIVEN** a valid `milestone_id` and the `task` table
- **WHEN** `INSERT INTO task (milestone_id, title) VALUES (1, NULL)`
- **THEN** the insert fails with a NOT NULL violation on `title`.

### T2. `description` is nullable (markdown allowed)

- **GIVEN** a valid `milestone_id` and the `task` table
- **WHEN** `INSERT INTO task (milestone_id, title) VALUES (1, 'Just a title')`
  (omitting `description`)
- **THEN** the insert succeeds; the new row has `description IS NULL`.

### T3. FK to `milestone` enforced; multiple tasks per milestone allowed (1:N)

- **GIVEN** the `task` table with no milestone of `id = 99999`
- **WHEN** a first insert against `milestone_id = 99999` is attempted
- **THEN** the insert fails with a foreign-key violation.
- **AND WHEN** two distinct inserts succeed against the same valid
  `milestone_id = 1` (each with its own synthetic `id`)
- **THEN** both rows exist and the FK from `task.milestone_id` to
  `milestone(requirement_id)` is intact.

---

## S. `spec`

### S4. `content` is NOT NULL

- **GIVEN** a valid `task_id` and the `spec` table
- **WHEN** `INSERT INTO spec (task_id, content) VALUES (1, NULL)`
- **THEN** the insert fails with a NOT NULL violation on `content`.

### S5. `start_date`, `end_date` are nullable

- **GIVEN** a valid `task_id` and the `spec` table
- **WHEN** `INSERT INTO spec (task_id, content) VALUES (1, 'spec body')`
  (omitting both dates)
- **THEN** the insert succeeds; `start_date IS NULL AND end_date IS NULL`.

### S6. `end_date >= start_date` enforced when both non-NULL

- **GIVEN** a valid `task_id` and the `spec` table
- **WHEN** `INSERT INTO spec (task_id, content, start_date, end_date) VALUES (1, 'body', DATE '2026-06-15', DATE '2026-06-10')`
- **THEN** the insert fails the `spec_dates_valid` CHECK constraint.

### S7. FK to `task` enforced; multiple specs per task allowed (1:N)

- **GIVEN** the `spec` table with no task of `id = 99999`
- **WHEN** a first insert against `task_id = 99999` is attempted
- **THEN** the insert fails with a foreign-key violation.
- **AND WHEN** two distinct inserts succeed against the same valid
  `task_id = 1` (each with its own synthetic `id`)
- **THEN** both rows exist and the FK from `spec.task_id` to `task(id)` is intact.

---

## SP. `spec_phase` (agent-first re-entry)

The 8 phase values are locked by the proposal (Q1):
`tdd_red`, `implementation`, `tdd_green`, `verify`, `pr`,
`technical_ai_review`, `ai_approved`, `human_approved`.

### SP1. Phase must be one of the 8 values

- **GIVEN** a valid `spec_id` and the `spec_phase` table
- **WHEN** `INSERT INTO spec_phase (spec_id, phase) VALUES (1, 'NOT_A_PHASE')`
- **THEN** the insert fails the `spec_phase_phase_check` CHECK constraint.

### SP2. `(spec_id, phase, started_at)` is UNIQUE

- **GIVEN** the `spec_phase` table with one row for `(spec_id=1, phase='tdd_red', started_at='2026-06-22T10:00:00Z')`
- **WHEN** a second row is inserted with the same triple
- **THEN** the insert fails with a unique-constraint violation on
  `spec_phase_natural_key`.

### SP3. `ended_at >= started_at` enforced

- **GIVEN** the `spec_phase` table
- **WHEN** `INSERT INTO spec_phase (spec_id, phase, started_at, ended_at) VALUES (1, 'tdd_red', TIMESTAMPTZ '2026-06-22 12:00:00Z', TIMESTAMPTZ '2026-06-22 10:00:00Z')`
- **THEN** the insert fails the `spec_phase_dates_valid` CHECK constraint.

### SP4. `notes` is nullable (empty transition allowed)

- **GIVEN** a valid `spec_id` and the `spec_phase` table
- **WHEN** `INSERT INTO spec_phase (spec_id, phase) VALUES (1, 'implementation')`
  (omitting `notes`)
- **THEN** the insert succeeds; the new row has `notes IS NULL`.

### SP5. Agent re-entry pattern: close current, open new

- **GIVEN** the `spec_phase` table contains an OPEN row for
  `(spec_id=1, phase='implementation', started_at=T1, ended_at=NULL)`
- **WHEN** the agent runs
  `UPDATE spec_phase SET ended_at = now() WHERE spec_id = 1 AND ended_at IS NULL`
  followed by
  `INSERT INTO spec_phase (spec_id, phase, notes) VALUES (1, 'tdd_red', 'AI review at <PR> found gap X')`
- **THEN** the UPDATE sets the original row's `ended_at` to a non-null value
  AND the INSERT succeeds with `started_at > ended_at` of the prior row
  AND `SELECT count(*) FROM spec_phase WHERE spec_id = 1 = 2`.

### SP6. Partial index `idx_spec_phase_current_state` returns exactly one row per spec

- **GIVEN** the `spec_phase` table after the re-entry pattern in SP5
- **WHEN** `EXPLAIN SELECT ... FROM spec_phase WHERE spec_id = 1 AND ended_at IS NULL`
- **THEN** the query plan uses `idx_spec_phase_current_state`
  AND `SELECT id, phase, started_at FROM spec_phase WHERE spec_id = 1 AND ended_at IS NULL`
  returns exactly one row.

### SP7. FK to `spec` enforced

- **GIVEN** the `spec_phase` table with no spec of `id = 99999`
- **WHEN** `INSERT INTO spec_phase (spec_id, phase) VALUES (99999, 'tdd_red')`
- **THEN** the insert fails with a foreign-key violation referencing `spec(id)`.

---

## A. Append-only convention

### A1. App-layer discipline on every table except `organization.is_active`

- **GIVEN** the 8-table schema is fully migrated
- **WHEN** a reviewer (human or CI lint) inspects the data access layer
  written in any future change
- **THEN** the reviewer MUST reject any `UPDATE` statement targeting
  `organization` columns OTHER than `is_active` AND any `UPDATE` statement
  targeting ANY column of `project`, `requirement`, `requirement_spike`,
  `milestone`, `task`, `spec`, or `spec_phase`. The column comment on
  `organization.is_active` is the documented contract.

> **Implementation note**: A1 is enforced by code review (per the proposal
> "Append-only enforcement — app-layer only (v1)" section). DB-level
> `REVOKE UPDATE` and BEFORE-UPDATE triggers are follow-up changes, not
> part of this spec.

---

## D. Migration rollback (Down blocks)

### D1. Down blocks reverse the corresponding Up blocks in reverse dependency order

- **GIVEN** the 3 new migration files are applied to a DB
- **WHEN** an operator runs `goose down` against all 3 files
- **THEN** the tables are dropped in this order: `spec_phase`, `spec`,
  `task`, `milestone`, `requirement_spike`, `requirement`, `project`,
  `organization`. Each file's `-- +goose Down` block contains `DROP TABLE`
  statements in reverse of its `-- +goose Up` block's creation order.

### D2. Down then Up cycle is byte-identical

- **GIVEN** the 3 new migrations are applied to a fresh DB and
  `public.schema_migrations` has 4 rows
- **WHEN** an operator runs `goose down` until `schema_migrations` is empty,
  then `goose up` again
- **THEN** the final schema has the same 8 tables, the same column
  definitions, the same CHECK constraints, the same UNIQUE constraints, the
  same FK constraints, and the same indexes as before the Down — and
  `public.schema_migrations` again has 4 rows.

---

## Review checklist

- [ ] reviewer can confirm every behavioral guarantee in `proposal.md`
  (8 tables, 3-file layout, 6 new tests, agent-first re-entry) has at least
  one scenario in this spec
- [ ] reviewer can confirm no scenario invents new behavior beyond the
  locked decisions in `proposal.md` ("Conventions locked" section)
- [ ] reviewer can confirm each scenario is implementable as a Go test in
  `backend/database_administrator/src/migration/runner_test.go`, gated on
  `INTEGRATION=1`
- [ ] reviewer can confirm the spec uses Given / When / Then format
  consistently and no scenario depends on SQL implementation details
- [ ] reviewer can confirm B1–B3 cover the runner's boot/idempotency/
  rollback guarantees end-to-end
- [ ] reviewer can confirm SP5 codifies the canonical agent re-entry
  SQL pattern from `proposal.md §"Agent-first re-entry pattern"`
- [ ] reviewer can confirm the 8 phase values in SP1 match the locked
  list in `proposal.md §Q1` (no more, no less)
- [ ] reviewer can confirm append-only enforcement (A1) is documented as
  a code-review contract, consistent with the proposal's deferred
  DB-level enforcement strategy