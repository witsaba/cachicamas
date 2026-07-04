-- +goose Up
-- +goose StatementBegin
-- cachicamas-github-login migration 1 of 1: identity.user + identity.account
-- + organization.owner_user_id.
--
-- Locks spec R-IS-001 (identity.user), R-IS-010 (identity.account),
-- R-IS-020 (organization.owner_user_id), R-IS-050 (queen ownership).
-- See openspec/changes/cachicamas-github-login/specs/identity-schema/spec.md.
--
-- Cardinality:
--   identity.user -- 1:N --> identity.account
--                    (synthetic PK + non-PK FK)
--   organization -- N:1 --> identity.user
--                    (optional FK; nullable so pre-slice rows keep NULL)
--
-- The `identity` schema is already provisioned at first boot by
-- infra/postgres/init/01-init.sql ("Users, roles, sessions, audit log").
-- The `citext` extension is also already installed there; this migration
-- does NOT re-install it (R-IS-002).
--
-- Note on the `UNIQUE` constraint: an explicit name
-- `account_provider_provider_account_id_key` is used so reviewer-side
-- grep + the verify walkthrough (S-IS-030, S-IS-031) can pin it.
CREATE TABLE identity.user (
    id          BIGSERIAL    PRIMARY KEY,
    email       CITEXT       NOT NULL UNIQUE,
    name        TEXT,
    image_url   TEXT,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

COMMENT ON TABLE identity.user IS
    'One row per human known to the system. The natural person, not the OAuth account. Email is case-insensitive (CITEXT) for auto-link on GitHub email match.';

ALTER TABLE identity.user OWNER TO queen;

CREATE TABLE identity.account (
    id                  BIGSERIAL    PRIMARY KEY,
    user_id             BIGINT       NOT NULL REFERENCES identity.user(id) ON DELETE CASCADE,
    provider            TEXT         NOT NULL,
    provider_account_id TEXT         NOT NULL,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT account_provider_provider_account_id_key UNIQUE (provider, provider_account_id)
);

COMMENT ON TABLE identity.account IS
    'One row per (provider, provider_account_id) pair. A single human can have multiple rows when they sign in via different GitHub accounts (e.g., the user changed their GitHub handle). ON DELETE CASCADE keeps accounts cleaned up with their owning user.';

ALTER TABLE identity.account OWNER TO queen;

ALTER TABLE organization
    ADD COLUMN owner_user_id BIGINT REFERENCES identity.user(id);

COMMENT ON COLUMN organization.owner_user_id IS
    'Optional FK to identity.user.id. Set by the create-org flow when the user is authenticated; NULL means no known owner (legacy row or dev/test data).';

CREATE INDEX idx_organization_owner_user_id ON organization(owner_user_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse order: drop the FK column + its index, then the two tables
-- in the opposite order of creation. The FK from identity.account to
-- identity.user is removed automatically when identity.account is dropped.
DROP INDEX IF EXISTS idx_organization_owner_user_id;
ALTER TABLE organization DROP COLUMN IF EXISTS owner_user_id;
DROP TABLE IF EXISTS identity.account;
DROP TABLE IF EXISTS identity.user;
-- +goose StatementEnd
