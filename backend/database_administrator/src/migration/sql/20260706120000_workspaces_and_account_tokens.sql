-- +goose Up
-- +goose StatementBegin
-- cachicamas-workspaces migration 1 of 1: 5 nullable OAuth token columns
-- on identity.account so the workspaces feature (PR1c-i onward) can read
-- the user's GitHub access_token from the DB and call /user/repos and
-- (future) clone repos.
--
-- Locks spec R-WS-010 (OAuth scope + access token persistence),
-- R-WS-096 (access_token NEVER returned in HTTP responses).
-- See openspec/changes/2026-07-06-workspaces/spec.md.
--
-- Why additive ALTER only (no backfill):
--   The columns are nullable TEXT/TIMESTAMPTZ. Pre-PR1a rows (signed in
--   before this migration lands) keep NULL tokens; the workspaces
--   surface detects NULL and shows a "Reconnect GitHub" banner (locked
--   R-WS-017). No data migration is needed.
--
-- Ordering: this migration lands BEFORE the workspaces feature (PR1b-i)
-- so the auth middleware can populate the columns from the first
-- post-PR1a sign-in onward.
ALTER TABLE identity.account
    ADD COLUMN IF NOT EXISTS access_token  TEXT,
    ADD COLUMN IF NOT EXISTS refresh_token TEXT,
    ADD COLUMN IF NOT EXISTS expires_at    TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS token_type    TEXT,
    ADD COLUMN IF NOT EXISTS scope         TEXT;

COMMENT ON COLUMN identity.account.access_token  IS
    'OAuth access token granted at sign-in (encrypted-at-rest deferred — see R-WS-014 tech-debt). NULL for pre-PR1a rows.';
COMMENT ON COLUMN identity.account.refresh_token IS
    'OAuth refresh token. NULL when the provider does not grant offline access.';
COMMENT ON COLUMN identity.account.expires_at    IS
    'Unix timestamp at which the access_token expires. NULL = unknown / no expiry.';
COMMENT ON COLUMN identity.account.token_type    IS
    'OAuth token type — typically "bearer". NULL when not provided.';
COMMENT ON COLUMN identity.account.scope         IS
    'Space-separated OAuth scope string the user granted. NULL when not provided. cachicamas requests scope=repo (PR1a onwards).';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse order: drop the 5 additive columns. Pre-PR1a code does not
-- reference them, so the rollback is non-destructive as long as no
-- workspace data was written with this migration applied (workspaces
-- tables land in PR1b-i, which depends on this migration).
ALTER TABLE identity.account
    DROP COLUMN IF EXISTS scope,
    DROP COLUMN IF EXISTS token_type,
    DROP COLUMN IF EXISTS expires_at,
    DROP COLUMN IF EXISTS refresh_token,
    DROP COLUMN IF EXISTS access_token;
-- +goose StatementEnd