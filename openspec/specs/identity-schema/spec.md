# identity-schema Specification

> **Domain**: identity-schema
> **Change**: cachicamas-github-login
> **Type**: New capability (full spec — no existing user/account tables; the `identity` schema is provisioned but empty)
> **Created**: 2026-07-03
> **Persistence**: hybrid (this file + Engram `sdd/cachicamas-github-login/spec/identity-schema`)

## Purpose

Defines the persistent identity schema for cachicamas GitHub login. Two new
tables (`identity.user`, `identity.account`) MUST be added under the existing
`identity` schema, and a nullable `organization.owner_user_id` foreign key
MUST be added to the existing `organization` table. The schema MUST support
the auto-link-on-email-match account-linking policy chosen in the proposal
(§5 question 3) and MUST NOT introduce a `identity.session` table (the slice
is stateless per the proposal).

The schema MUST be defined via a goose v3 migration under
`backend/database_administrator/src/migration/sql/` (consistent with the
project's migration discipline, see `db-migrations` R-DBMIG-001). It MUST
NOT be added by hand-editing `infra/postgres/init/01-init.sql` (that file is
one-shot for first boot and re-running it does not clobber existing state).

## Glossary

| Term | Meaning |
| ------ | --------- |
| **`identity` schema** | A logical schema provisioned at first boot by `infra/postgres/init/01-init.sql` (per its locked comment "Users, roles, sessions, audit log"). The schema exists, is owned by `queen`, and is empty. |
| **`identity.user`** | One row per human being known to the system. The natural person, not the OAuth account. |
| **`identity.account`** | One row per `(provider, provider_account_id)` pair. A single human can have multiple accounts (e.g., a future Google account on top of their GitHub one). |
| **`organization.owner_user_id`** | The nullable FK from `organization.id` to `identity.user.id`. Lets us record "who created this org" and gate ownership-protected routes later. |
| **Auto-link on email match** | The chosen account-linking policy (proposal §5 Q3): if a returning OAuth account's email matches an existing `identity.user.email`, the existing user is reused and a new `identity.account` row is added. |
| **CITEXT** | The Postgres extension for case-insensitive text. `email` is CITEXT so `braejan@example.com` and `BRAEJAN@example.com` are the same row. |

---

## Capability: identity.user table

### Requirement: R-IS-001 The `identity.user` table exists with the locked column set

The `identity.user` table MUST exist in the `identity` schema and MUST have
exactly the following columns:

| Column | Type | Constraints | Notes |
| --- | --- | --- | --- |
| `id` | `BIGSERIAL` | `PRIMARY KEY` | Same shape as `organization.id` for cross-table consistency. |
| `email` | `CITEXT` | `NOT NULL UNIQUE` | The natural key for auto-link. UNIQUE is the same constraint that drives the S-IS-010 + S-IS-011 scenarios. |
| `name` | `TEXT` | nullable | GitHub display name; can be null if the user hides it. |
| `image_url` | `TEXT` | nullable | GitHub avatar URL. |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` | |
| `updated_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` | |

The table MUST be owned by `queen` (consistent with the project's policy that
all schema objects are owned by queen, per `infra/postgres/init/01-init.sql`).
The migration MUST be reversible (the Down section drops the table in the
opposite order of the FKs, and the test suite MUST exercise both directions
in dev).

#### Scenario: S-IS-010 identity.user table exists

- GIVEN the goose migration `20260703120000_github_login.sql` has been applied
- WHEN the orchestrator connects to Postgres as `queen` and runs `\d identity.user`
- THEN the table SHALL appear with exactly the columns listed in R-IS-001
- AND the owner SHALL be `queen`

Verification: `psql -U queen -d cachicamas_pg -c "\d identity.user"` shows the locked columns; `psql -U queen -d cachicamas_pg -c "SELECT pg_get_userbyid(relowner) FROM pg_class WHERE relname = 'user' AND relnamespace = 'identity'::regnamespace"` returns `queen`.

#### Scenario: S-IS-011 email column rejects duplicates case-insensitively

- GIVEN `identity.user` is empty
- WHEN the orchestrator inserts `(email = 'braejan@example.com')` and then `(email = 'BRAEJAN@example.com')`
- THEN the second insert SHALL fail with `ERROR: duplicate key value violates unique constraint "user_email_key"`

Verification: two `INSERT` statements; observe the second one fail with the locked constraint name.

### Requirement: R-IS-002 The CITEXT extension is available

The `citext` extension MUST be enabled in the `cachicamas_pg` database before
`identity.user.email` is created. The extension is already enabled in
`infra/postgres/init/01-init.sql` (`CREATE EXTENSION IF NOT EXISTS "citext";`),
so the migration MUST NOT re-create it; the migration's Up section MUST
include a comment that documents this dependency.

#### Scenario: S-IS-020 citext extension is enabled

- GIVEN a freshly initialized volume
- WHEN the orchestrator runs `psql -U cachicamas -d cachicamas_pg -c "SELECT extname FROM pg_extension WHERE extname = 'citext'"`
- THEN the query SHALL return one row with `extname = 'citext'`

Verification: same query; the output is a single `citext` row.

---

## Capability: identity.account table

### Requirement: R-IS-010 The `identity.account` table exists and links users to OAuth providers

The `identity.account` table MUST exist in the `identity` schema and MUST
have exactly the following columns:

| Column | Type | Constraints | Notes |
| --- | --- | --- | --- |
| `id` | `BIGSERIAL` | `PRIMARY KEY` | |
| `user_id` | `BIGINT` | `NOT NULL REFERENCES identity.user(id) ON DELETE CASCADE` | The owning user. |
| `provider` | `TEXT` | `NOT NULL` | E.g., `'github'`. A `CHECK (provider IN ('github'))` constraint is OPTIONAL this slice; new providers are follow-up. |
| `provider_account_id` | `TEXT` | `NOT NULL` | The provider-side user ID (numeric string for GitHub, opaque string for future OIDC). |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT now()` | |

A unique constraint on `(provider, provider_account_id)` MUST exist so the
same GitHub account cannot be linked to two `identity.user` rows.

#### Scenario: S-IS-030 identity.account table exists

- GIVEN the migration is applied
- WHEN the orchestrator runs `\d identity.account`
- THEN the table SHALL appear with the locked columns
- AND a `UNIQUE` index on `(provider, provider_account_id)` SHALL exist

Verification: `\d identity.account` lists the index `account_provider_provider_account_id_key`.

#### Scenario: S-IS-031 A second account for the same provider+account_id is rejected

- GIVEN `identity.user` has one row for `braejan@example.com` (id = 42)
- AND `identity.account` has one row for `(provider='github', provider_account_id='12345', user_id=42)`
- WHEN the orchestrator inserts a second row for `(provider='github', provider_account_id='12345', user_id=99)` (a different user)
- THEN the insert SHALL fail with the locked unique-constraint error

Verification: `INSERT` returns the unique-violation error.

#### Scenario: S-IS-032 A user can have multiple accounts from different providers

- GIVEN the same state as S-IS-031
- WHEN the orchestrator inserts a second account row for `(provider='github', provider_account_id='99999', user_id=42)` (a different GitHub account for the same user, e.g., the user changed their GitHub handle and a new ID was assigned)
- THEN the insert SHALL succeed (different `provider_account_id`)
- AND the new row's `user_id` SHALL be `42`

Verification: insert succeeds; `SELECT count(*) FROM identity.account WHERE user_id = 42` returns 2.

#### Scenario: S-IS-033 Deleting a user cascades to their accounts

- GIVEN `identity.account` has a row with `user_id = 42`
- WHEN `DELETE FROM identity.user WHERE id = 42` is executed
- THEN the matching `identity.account` row SHALL be removed automatically

Verification: before the delete, `SELECT count(*) FROM identity.account WHERE user_id = 42` returns ≥1; after, it returns 0.

---

## Capability: organization.owner_user_id

### Requirement: R-IS-020 The `organization.owner_user_id` column is added

The `organization` table MUST gain a new nullable column
`owner_user_id BIGINT REFERENCES identity.user(id)`. The column is nullable
because pre-existing organizations have no known owner. The column MUST be
created in the same migration as the `identity.*` tables (one Up, one Down).

#### Scenario: S-IS-040 organization has the new column

- GIVEN the migration is applied
- WHEN the orchestrator runs `\d organization`
- THEN the column `owner_user_id` SHALL appear as `bigint` (nullable)
- AND a foreign-key constraint to `identity.user(id)` SHALL be listed

Verification: `\d organization` lists `owner_user_id | bigint` and the FK constraint name (e.g., `organization_owner_user_id_fkey`).

#### Scenario: S-IS-041 Existing rows keep owner_user_id = NULL

- GIVEN the database has 3 rows in `organization` from before this slice
- WHEN the migration runs
- THEN the 3 rows SHALL remain with `owner_user_id = NULL`
- AND no row SHALL be backfilled

Verification: `SELECT id, owner_user_id FROM organization` shows the 3 rows, all NULL.

#### Scenario: S-IS-042 Setting owner_user_id to a known user is allowed

- GIVEN `identity.user` has a row with id = 7
- AND `organization` has a row with id = 100
- WHEN `UPDATE organization SET owner_user_id = 7 WHERE id = 100` is executed
- THEN the update SHALL succeed
- AND `SELECT owner_user_id FROM organization WHERE id = 100` SHALL return `7`

Verification: SQL above; the row reflects the new owner.

#### Scenario: S-IS-043 Setting owner_user_id to a non-existent user is rejected

- GIVEN `identity.user` has no row with id = 99999
- WHEN `UPDATE organization SET owner_user_id = 99999 WHERE id = 100` is executed
- THEN the update SHALL fail with the FK-violation error

Verification: SQL above; observe the FK violation.

---

## Capability: Auto-link-on-email-match semantics

### Requirement: R-IS-030 The application layer (not the schema) implements the linking policy

The schema MUST support the auto-link-on-email-match policy but MUST NOT
enforce it at the DB level. The linking happens in the Auth.js `signIn`
callback (frontend, see `frontend-auth` R-FA-011) and in any future
`IdentityService` on the Go side. The schema only requires:

- `identity.user.email` is `CITEXT UNIQUE` (so the lookup is case-insensitive)
- `identity.account` has `UNIQUE(provider, provider_account_id)` (so a single
  OAuth account can be linked to at most one `identity.user`)

The schema MUST NOT use a trigger or partial unique index that would force
the linking policy into the DB. (This is an explicit non-goal so future
linking policies can change without a migration.)

#### Scenario: S-IS-050 Linking is implemented in the application

- GIVEN the change is applied
- WHEN the orchestrator greps for `signIn` callback implementations
- THEN the callback SHALL exist in `frontend/src/routes/plugin@auth.ts`
- AND no SQL `CREATE TRIGGER` or `CREATE FUNCTION` SHALL be present in
      the migration file (so linking is application-side only)

Verification: `grep -RE "CREATE TRIGGER|CREATE FUNCTION" backend/database_administrator/src/migration/sql/20260703120000_github_login.sql` returns no results.

---

## Capability: Migration discipline

### Requirement: R-IS-040 The migration follows the project's existing pattern

The new migration file MUST be named
`20260703120000_github_login.sql` (14-digit YYYYMMDDHHMMSS prefix) and MUST
live under `backend/database_administrator/src/migration/sql/`. The Up
section MUST create the two tables and add the FK column in a single
transaction (one `-- +goose Up` block, multiple `-- +goose StatementBegin`
blocks if needed). The Down section MUST reverse the operations in the
opposite order, dropping the FK column before the tables.

The migration MUST NOT touch the `public.schema_migrations` table directly
(goose writes to it; the operator has already reserved the bookkeeping shape
per `db-migrations` R-DBMIG-070).

#### Scenario: S-IS-060 Migration filename matches the project pattern

- GIVEN the change is applied
- WHEN the orchestrator lists `backend/database_administrator/src/migration/sql/`
- THEN the file `20260703120000_github_login.sql` SHALL exist
- AND no other new migration file SHALL exist

Verification: `ls backend/database_administrator/src/migration/sql/` lists the four pre-existing files plus exactly this one new file.

#### Scenario: S-IS-061 Migration is reversible

- GIVEN the migration is applied
- WHEN the operator runs `goose -dir backend/database_administrator/src/migration/sql postgres <DSN> down`
- THEN the migration SHALL roll back cleanly
- AND `\d identity.user`, `\d identity.account`, `\d organization` SHALL all
      return to their pre-migration shapes (no `owner_user_id` column, no
      `identity.user` table, no `identity.account` table)

Verification: after the down, the same `\d` queries confirm the rollback; running the up again restores the schema.

### Requirement: R-IS-041 The migration is fully owned by the runner (queen)

The migration MUST succeed when run by the project's migration runner
(connecting as `queen`). The migration MUST NOT require any object outside
the `identity` and `public` schemas, MUST NOT require any superuser-only
operation (`ALTER SYSTEM`, `REINDEX DATABASE`, etc.), and MUST be idempotent
under the goose v3 retry-on-startup behavior (no `INSERT ... ON CONFLICT`
trickery required).

#### Scenario: S-IS-070 Migration applies cleanly on a fresh stack

- GIVEN `docker compose down -v` was just run
- WHEN `docker compose up -d database_administrator` is executed
- THEN the runner SHALL apply the new migration
- AND the row in `public.schema_migrations` SHALL have `version_id = 20260703120000`, `is_applied = true`
- AND the migration.up OTel span SHALL appear with `migration.applied_count ≥ 1` and `migration.error = ""`

Verification: `docker compose logs database_administrator | grep "migration.up"` shows the new version; `psql -U queen -d cachicamas_pg -c "SELECT version_id, is_applied FROM public.schema_migrations WHERE version_id = 20260703120000"` returns the row.

---

## Capability: Naming and access policy

### Requirement: R-IS-050 All new objects are owned by `queen` and live under `identity`

Consistent with the existing pattern in `infra/postgres/init/01-init.sql`
(queen owns the schemas), the new tables MUST be owned by `queen`. The
migration MUST include an `ALTER TABLE identity.user OWNER TO queen;` and the
same for `identity.account` (defensive — even if the schema default is
queen, the explicit ALTER guarantees it survives future role changes).

#### Scenario: S-IS-080 ownership is queen

- GIVEN the migration is applied
- WHEN the orchestrator runs `SELECT relname, pg_get_userbyid(relowner) FROM pg_class WHERE relname IN ('user', 'account') AND relnamespace = 'identity'::regnamespace`
- THEN both rows SHALL have `pg_get_userbyid(relowner) = 'queen'`

Verification: query above; both rows show `queen`.

---

## Review checklist

- [ ] reviewer can confirm the spec describes WHAT (capabilities, requirements, scenarios) and not HOW (no Go code, no application-logic detail beyond the linking policy)
- [ ] reviewer can confirm every scenario uses `GIVEN/WHEN/THEN` and is independently verifiable with a `psql` query
- [ ] reviewer can confirm no `identity.session` table is introduced (intentional per proposal)
- [ ] reviewer can confirm the migration is reversible (R-IS-040, S-IS-061)
- [ ] reviewer can confirm `organization.owner_user_id` is nullable (R-IS-020, S-IS-041)
- [ ] reviewer can confirm CITEXT is used for email (R-IS-001) and citext extension is already enabled (R-IS-002)
- [ ] reviewer can confirm linking is application-side, not DB-enforced (R-IS-030, S-IS-050)
- [ ] reviewer can confirm the migration follows the project's filename pattern (R-IS-040, S-IS-060)
- [ ] reviewer can confirm no infra file (`infra/`) was modified (the schema is provisioned; the tables come from the migration)
