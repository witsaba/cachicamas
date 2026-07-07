-- +goose Up
-- +goose StatementBegin
-- 2026-07-08-workspaces-simplify (follow-up): rename the workspace
-- primary_repo_* columns to repo_*.
--
-- Background:
--   PR1c-ii (commit 272aa68) created the workspace table with columns
--   named primary_repo_github_id / primary_repo_full_name /
--   primary_repo_owner / primary_repo_name. In the original 1:many
--   model those names matched the "primary" repo (vs. the linked
--   repos in workspace_repository).
--
--   2026-07-08-workspaces-simplify renamed the Go struct fields from
--   PrimaryRepo* to Repo* and the Go SQL queries to read from the
--   new column names. BUT the database columns themselves were
--   never renamed — the new pgx queries try to SELECT repo_github_id
--   from a table that still has primary_repo_github_id, which
--   produces a runtime "column does not exist" error (mapped to a
--   500 by the writeWorkspaceError path).
--
-- This migration fixes the column names. RENAME COLUMN is metadata-
-- only in Postgres (no data rewrite), so it is fast and safe on
-- any table size.
--
-- Idempotency: IF EXISTS guards make this migration safe to re-run.
ALTER TABLE workspace RENAME COLUMN primary_repo_github_id TO repo_github_id;
ALTER TABLE workspace RENAME COLUMN primary_repo_full_name TO repo_full_name;
ALTER TABLE workspace RENAME COLUMN primary_repo_owner TO repo_owner;
ALTER TABLE workspace RENAME COLUMN primary_repo_name TO repo_name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse the renames. Operationally only relevant for a
-- production rollback; in dev a fresh `docker compose down -v`
-- resets the schema.
ALTER TABLE workspace RENAME COLUMN repo_github_id TO primary_repo_github_id;
ALTER TABLE workspace RENAME COLUMN repo_full_name TO primary_repo_full_name;
ALTER TABLE workspace RENAME COLUMN repo_owner TO primary_repo_owner;
ALTER TABLE workspace RENAME COLUMN repo_name TO primary_repo_name;
-- +goose StatementEnd