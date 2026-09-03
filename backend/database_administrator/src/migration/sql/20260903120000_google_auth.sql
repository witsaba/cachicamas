-- +goose Up
-- +goose StatementBegin
-- cachicamas-google-auth-bootstrap migration 1 of 1:
--   auth schema + auth.users + auth.organizations + auth.login_audits
--   + rewrite workspace.owner_user_id FK (identity.user → auth.users)
--   + rewrite organization.owner_user_id FK (identity.user → auth.users)
--   + drop identity.user + identity.account
--
-- Locks spec R-DB-001..006 (Engram #4222):
--   R-DB-001 auth.users   — id BIGSERIAL PK, email VARCHAR(255) NOT NULL
--                          (partial unique index over live rows), google_sub
--                          VARCHAR(255) UNIQUE NULL, status state machine,
--                          created_at / updated_at, soft delete.
--   R-DB-002 Partial unique on (lower(email)) WHERE deleted_at IS NULL.
--   R-DB-003 auth.organizations   — owner_id FK to auth.users(id) ON DELETE RESTRICT,
--                          slug VARCHAR(64) UNIQUE among live rows.
--   R-DB-004 auth.login_audits — user_id FK to auth.users(id) ON DELETE SET
--                          NULL (nullable so failed logins still audit),
--                          success BOOLEAN NOT NULL, failure_reason, login_at.
--   R-DB-005 FK rewrite — workspace.owner_user_id AND organization.owner_user_id
--                          both rewritten to auth.users(id) (AD-3); identity.user
--                          + identity.account then dropped in the SAME TX.
--   R-DB-006 status enum — registered | active | inactive | blocked; validated
--                          at the Go domain layer (DB accepts any value; this
--                          migration does NOT add a CHECK on status so the
--                          state machine stays flexible for future phases).
--
-- See openspec/changes/cachicamas-google-auth-bootstrap/design.md §4 for the
-- full rationale and AD-3 / AD-4 / AD-5 / AD-6 (atomicity, FK ON DELETE
-- preservation, schema provisioning, plural table names).
--
-- The migration is forward-only on Down: it recreates identity.user /
-- identity.account as tombstones (per spec R-DB-005 "for emergency rollback")
-- but data inserted into auth.* after the Up run is LOST on Down.

-- ── Provision the `auth` schema (AD-5) ────────────────────────────────────
-- Created in this migration, NOT in infra/postgres/init/01-init.sql
-- (per design AD-5; AGENTS.md hard rule says do not modify infra/).
CREATE SCHEMA IF NOT EXISTS auth;
ALTER SCHEMA auth OWNER TO queen;

-- ── auth.users (R-DB-001 / R-DB-002) ───────────────────────────────────────
CREATE TABLE auth.users (
    id              BIGSERIAL    PRIMARY KEY,
    email           VARCHAR(255) NOT NULL,
    email_verified  BOOLEAN      NOT NULL DEFAULT FALSE,
    google_sub      VARCHAR(255) UNIQUE,
    name            VARCHAR(255),
    picture_url     TEXT,
    status          VARCHAR(32)  NOT NULL DEFAULT 'registered',
    last_login_at   TIMESTAMPTZ,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);
ALTER TABLE auth.users OWNER TO queen;

-- Partial unique index: allow re-registration after soft-delete (R-DB-002).
CREATE UNIQUE INDEX auth_users_email_live_key
    ON auth.users (lower(email))
    WHERE deleted_at IS NULL;

-- Secondary indexes used by the bootstrap service (R-BE-002):
--   - google_sub lookup is the primary key for the resolver
--     (unique constraint above already provides an index; this is
--      explicit for reviewability).
--   - email lookup supports display-name / admin tooling.
CREATE INDEX auth_users_google_sub_idx ON auth.users (google_sub);
CREATE INDEX auth_users_email_idx      ON auth.users (lower(email));

COMMENT ON TABLE auth.users IS
    'One row per human known to the system. Provisioned by the google_auth migration as part of PR-1 Foundations (cachicamas-google-auth-bootstrap). google_sub is the provider-stable secondary key (R-DB-001).';
COMMENT ON COLUMN auth.users.email IS
    'Lowercased at the application layer (see user_repo). Uniqueness among non-deleted rows is enforced by auth_users_email_live_key (R-DB-002).';
COMMENT ON COLUMN auth.users.status IS
    'State machine: registered | active | inactive | blocked. Validated by the Go domain layer (R-DB-006 / S-DB-050); the DB does not enforce the vocabulary.';
COMMENT ON COLUMN auth.users.created_at IS
    'Registration date = first successful login (R-DB-001 / proposal D-4). Immutable on update (asserted by repo tests; no DB trigger is needed because the column is set ONLY in INSERT paths).';

-- ── auth.organizations (R-DB-003) ────────────────────────────────────────────────
CREATE TABLE auth.organizations (
    id          BIGSERIAL    PRIMARY KEY,
    owner_id    BIGINT       NOT NULL REFERENCES auth.users(id) ON DELETE RESTRICT,
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(64)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ
);
ALTER TABLE auth.organizations OWNER TO queen;

CREATE UNIQUE INDEX auth_organizations_slug_live_key
    ON auth.organizations (slug)
    WHERE deleted_at IS NULL;

CREATE INDEX auth_organizations_owner_id_idx ON auth.organizations (owner_id);

COMMENT ON TABLE auth.organizations IS
    'One row per organization owned by exactly one user (MVP tenancy: 1 Google user = 1 organization, proposal D-2). owner_id FK RESTRICT prevents organization deletion when user exists.';
COMMENT ON COLUMN auth.organizations.owner_id IS
    'FK to auth.users(id) ON DELETE RESTRICT (R-DB-003). 1:1 with auth.users in the MVP; multi-organization is a follow-up.';

-- ── auth.login_audits (R-DB-004) ──────────────────────────────────────────
CREATE TABLE auth.login_audits (
    id                 BIGSERIAL    PRIMARY KEY,
    user_id            BIGINT       REFERENCES auth.users(id) ON DELETE SET NULL,
    email_attempted    VARCHAR(255),
    provider           VARCHAR(32)  NOT NULL DEFAULT 'google',
    provider_subject   VARCHAR(255),
    ip_address         VARCHAR(45),
    user_agent         TEXT,
    country_code       CHAR(2),
    city               VARCHAR(128),
    success            BOOLEAN      NOT NULL,
    failure_reason     VARCHAR(255),
    login_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ
);
ALTER TABLE auth.login_audits OWNER TO queen;

CREATE INDEX auth_login_audits_user_id_idx  ON auth.login_audits (user_id);
CREATE INDEX auth_login_audits_login_at_idx ON auth.login_audits (login_at DESC);

COMMENT ON TABLE auth.login_audits IS
    'One row per login attempt (success or failure). user_id is nullable so failed logins before the user row exists still audit (R-DB-004 / S-DB-030). Append-mostly: rows are inserted and rarely updated.';
COMMENT ON COLUMN auth.login_audits.user_id IS
    'Nullable FK to auth.users(id) ON DELETE SET NULL (R-DB-004). NULL for failed logins before the user row exists.';
COMMENT ON COLUMN auth.login_audits.success IS
    'BOOLEAN NOT NULL. failure_reason is empty on success; the failure taxonomy is locked in spec R-BE-007 (state_mismatch | code_exchange_failed | userinfo_failed | internal_error).';

-- ── Rewrite workspace.owner_user_id FK (T1.4 / R-FK-1) ────────────────────
-- The legacy FK pointed at identity.user(id); per design AD-3 the new FK
-- points at auth.users(id). ON DELETE SET NULL is preserved (AD-4) — the
-- existing semantics are unchanged for workspace rows.
ALTER TABLE workspace DROP CONSTRAINT IF EXISTS workspace_owner_user_id_fkey;
ALTER TABLE workspace
    ADD CONSTRAINT workspace_owner_user_id_fkey
    FOREIGN KEY (owner_user_id) REFERENCES auth.users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS workspace_owner_user_id_idx ON workspace(owner_user_id);

-- ── Rewrite organization.owner_user_id FK (T1.5 / R-FK-2 / AD-3) ──────────
-- Same pattern as workspace: AD-3 explicitly requires BOTH FKs rewritten
-- atomically with the identity.* removal. The legacy FK on organization
-- (added by 20260703120000_github_login.sql) had no explicit ON DELETE
-- (so it defaults to NO ACTION); we keep NO ACTION for the new FK so
-- semantics are preserved (AD-4 spirit: preserve existing semantics).
ALTER TABLE organization DROP CONSTRAINT IF EXISTS organization_owner_user_id_fkey;
ALTER TABLE organization
    ADD CONSTRAINT organization_owner_user_id_fkey
    FOREIGN KEY (owner_user_id) REFERENCES auth.users(id);

CREATE INDEX IF NOT EXISTS idx_organization_owner_user_id ON organization(owner_user_id);

-- ── Drop legacy identity tables (T1.3 / R-DROP-1) ────────────────────────
-- identity.account must go BEFORE identity.user so the FK
-- identity.account.user_id -> identity.user(id) is resolved. Both tables
-- are dropped here, NOT in the Down, because the Up is forward-only per
-- spec R-DB-005; the Down section recreates them as tombstones for
-- emergency rollback only.
DROP TABLE IF EXISTS identity.account;
DROP TABLE IF EXISTS identity.user;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Emergency rollback: recreate identity.user + identity.account as empty
-- tombstones, restore the legacy FKs on workspace + organization, drop
-- auth.* schema. Caveat: any rows inserted into auth.* after the Up run
-- are LOST on Down — the Down is a safety net for "we have to revert
-- immediately" only, not a clean rollback.
--
-- Reverse order: restore identity.* FIRST (so the legacy FKs can be
-- added), then restore the FKs, then drop auth.*.

-- 1. Recreate identity.user + identity.account as tombstones.
CREATE TABLE identity.user (
    id          BIGSERIAL    PRIMARY KEY,
    email       TEXT,
    name        TEXT,
    image_url   TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
ALTER TABLE identity.user OWNER TO queen;

CREATE TABLE identity.account (
    id                  BIGSERIAL    PRIMARY KEY,
    user_id             BIGINT       NOT NULL REFERENCES identity.user(id) ON DELETE CASCADE,
    provider            TEXT         NOT NULL,
    provider_account_id TEXT         NOT NULL
);
ALTER TABLE identity.account OWNER TO queen;

-- 2. Restore the legacy FKs (workspace ON DELETE SET NULL; organization
--    with default NO ACTION — matches the original 20260703120000 +
--    20260706120002 migrations).
ALTER TABLE workspace DROP CONSTRAINT IF EXISTS workspace_owner_user_id_fkey;
ALTER TABLE workspace
    ADD CONSTRAINT workspace_owner_user_id_fkey
    FOREIGN KEY (owner_user_id) REFERENCES identity.user(id) ON DELETE SET NULL;

ALTER TABLE organization DROP CONSTRAINT IF EXISTS organization_owner_user_id_fkey;
ALTER TABLE organization
    ADD CONSTRAINT organization_owner_user_id_fkey
    FOREIGN KEY (owner_user_id) REFERENCES identity.user(id);

-- 3. Drop auth.* schema in reverse dependency order.
DROP TABLE IF EXISTS auth.login_audits;
DROP TABLE IF EXISTS auth.organizations;
DROP TABLE IF EXISTS auth.users;
DROP SCHEMA IF EXISTS auth;
-- +goose StatementEnd