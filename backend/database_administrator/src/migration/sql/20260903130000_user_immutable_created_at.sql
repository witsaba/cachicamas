-- +goose Up
-- +goose StatementBegin
-- cachicamas-google-auth-bootstrap PR-2 migration 1 of 1:
--   Enforce auth.users.created_at immutability at the database layer.
--
-- Locks spec R-DB-001 / S-DB-002:
--   "created_at is the registration date = first successful login.
--    MUST be immutable on update."
--
-- The application layer already excludes created_at from the
-- repository UPDATE column list (see
-- src/infrastructure/postgres/user_repo.go::UpdateLoginFields).
-- This trigger is the database-layer backstop: any UPDATE that
-- tries to modify created_at — whether from a future bug, a raw
-- psql session, or a third-party migration — is silently
-- reverted before the row is committed.
--
-- Per design §1 the trigger is BEFORE UPDATE so the row
-- modification happens before the constraint check; the result
-- is a successful UPDATE that preserves created_at without
-- raising an exception (which would break legitimate UPDATE
-- paths that happen to touch other columns).
CREATE OR REPLACE FUNCTION auth.prevent_created_at_update()
RETURNS TRIGGER AS $$
BEGIN
    NEW.created_at = OLD.created_at;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_created_at_immutable
    BEFORE UPDATE ON auth.users
    FOR EACH ROW
    EXECUTE FUNCTION auth.prevent_created_at_update();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse: drop the trigger + the trigger function. Existing rows
-- are unchanged; the application-layer exclusion in
-- user_repo.go::UpdateLoginFields remains the second line of
-- defence.
DROP TRIGGER IF EXISTS users_created_at_immutable ON auth.users;
DROP FUNCTION IF EXISTS auth.prevent_created_at_update();
-- +goose StatementEnd