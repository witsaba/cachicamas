-- +goose Up
-- +goose StatementBegin
-- 2026-07-06-workspaces migration 1 of 1: workspace + workspace_repository.
--
-- Locks spec R-WS-001 (workspace), R-WS-005 (soft delete + partial unique),
-- R-WS-006 (linked repo connection). See openspec/changes/2026-07-06-workspaces/spec.md.
--
-- Cardinality:
--   organization -- 1:N --> workspace
--       (FK ON DELETE CASCADE: deleting the install's only org also drops
--        its workspaces — acceptable; the single-tenant model means
--        org deletion is not a normal operation).
--   identity.user -- 1:N --> workspace
--       (FK ON DELETE SET NULL: a user can disappear (account deletion)
--        while the workspace itself is preserved; ownership is reset to
--        unattributed but the row + linked repos stay).
--   workspace -- 1:N --> workspace_repository
--       (FK ON DELETE CASCADE: PR1b-ii.a's SoftDelete uses the same FK to
--        hard-delete the linked repos when the workspace is soft-deleted;
--        a hard delete of workspace via cascade takes the linked repos
--        with it).
--
-- Partial unique index `workspace_org_name_live_key` on
-- (organization_id, name) WHERE deleted_at IS NULL locks spec R-WS-005
-- S-WS-043: a soft-deleted workspace frees up its name for reuse by a
-- fresh live workspace. The index keeps uniqueness enforcement cheap
-- (Postgres planner only walks live rows).
--
-- Synthetic PKs on both tables for join convenience. The natural unique
-- identifier of a workspace_repository row is (workspace_id, github_id);
-- that pair has its own UNIQUE constraint
-- `workspace_repository_workspace_github_key` so a workspace cannot link
-- the same GitHub repo twice.
CREATE TABLE workspace (
    id                       BIGSERIAL    PRIMARY KEY,
    organization_id          BIGINT       NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
    owner_user_id            BIGINT       REFERENCES identity.user(id) ON DELETE SET NULL,
    name                     TEXT         NOT NULL,
    primary_repo_github_id   BIGINT       NOT NULL,
    primary_repo_full_name   TEXT         NOT NULL,
    primary_repo_owner       TEXT         NOT NULL,
    primary_repo_name        TEXT         NOT NULL,
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ
);

ALTER TABLE workspace OWNER TO queen;

COMMENT ON TABLE workspace IS
    'A logical container scoped to one organization. Maps 1:1 to a primary GitHub repository (the "github project" it represents) and can additionally connect N more repositories. Single-tenant today (one organization per install); multi-tenant-ready via organization_id FK.';

COMMENT ON COLUMN workspace.primary_repo_github_id IS
    'GitHub integer id of the primary repo. Locked at create time — PATCH /workspaces/:id silently ignores primary_repo changes (design T9).';

COMMENT ON COLUMN workspace.deleted_at IS
    'Soft-delete tombstone. Partial unique index workspace_org_name_live_key excludes rows where deleted_at IS NOT NULL, so a name can be reused after soft delete (R-WS-005 S-WS-043).';

CREATE UNIQUE INDEX workspace_org_name_live_key
    ON workspace (organization_id, name)
    WHERE deleted_at IS NULL;

CREATE INDEX workspace_org_deleted_at_idx
    ON workspace (organization_id, deleted_at);

CREATE TABLE workspace_repository (
    id                BIGSERIAL    PRIMARY KEY,
    workspace_id      BIGINT       NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    github_id         BIGINT       NOT NULL,
    github_full_name  TEXT         NOT NULL,
    github_owner      TEXT         NOT NULL,
    github_name       TEXT         NOT NULL,
    added_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT workspace_repository_workspace_github_key UNIQUE (workspace_id, github_id)
);

ALTER TABLE workspace_repository OWNER TO queen;

COMMENT ON TABLE workspace_repository IS
    'A GitHub repository linked to a workspace. The pair (workspace_id, github_id) is unique — a workspace cannot link the same GitHub repo twice. The primary repo is NOT stored here (it lives in workspace.primary_repo_*) to avoid duplicate-key confusion.';

CREATE INDEX workspace_repository_workspace_idx
    ON workspace_repository (workspace_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Reverse order: drop the child table first (FK from workspace_repository
-- to workspace must be removed before workspace itself is dropped). CASCADE
-- on the FK handles the index + constraint cleanup.
DROP TABLE IF EXISTS workspace_repository;
DROP TABLE IF EXISTS workspace;
-- +goose StatementEnd