-- +goose Up
-- +goose StatementBegin
-- 2026-07-08-workspaces-sync-clone PR-1 of 7: sync_job table + workspace sync columns.
--
-- Locks spec R-WS-019 (Workspace sync). See
-- openspec/changes/2026-07-08-workspace-sync-clone/specs/workspaces/spec.md.
--
-- Schema additions:
--   1. New `sync_job` table (BIGSERIAL id) that records one row per
--      sync attempt. Statuses are pending / running / done / failed.
--      `triggered_by` is auto_on_create (from POST /workspaces) or
--      manual (from POST /workspaces/:id/sync).
--   2. ALTER on `workspace` adding four columns that track the latest
--      sync outcome (timestamps, last commit SHA, the repo's default
--      branch, and a back-pointer to the latest sync_job).
--
-- Cardinality:
--   workspace -- 1:N --> sync_job
--       (FK ON DELETE CASCADE on sync_job.workspace_id: soft-deleting a
--        workspace orphans the in-flight job; the workspace_syncer
--        sweep removes the cloned data and the sync_job row will be
--        retained with its `status` updated to 'failed' by the callback
--        handler, or simply aged out by a future retention sweeper).
--   sync_job -- 1:0..1 --> workspace (back-pointer)
--       (FK ON DELETE SET NULL on workspace.last_sync_job_id: if the
--        sync_job row is hard-deleted (e.g. by a retention sweeper), the
--        workspace.last_sync_job_id becomes NULL without orphaning the
--        workspace row).
--
-- Single-flight invariant:
--   The partial unique index `sync_job_single_flight_uidx` on
--   (workspace_id) WHERE status IN ('pending','running') guarantees at
--   most one in-flight job per workspace. A second INSERT of a pending
--   job for the same workspace_id fails with SQLSTATE 23505; the pgx
--   adapter translates this to *domain.ConflictError so the HTTP handler
--   can return 409 SYNC_ALREADY_RUNNING.
--
-- Type convention: all new IDs are BIGSERIAL (int64) to match the
-- existing `workspace.id` and the rest of the schema. NO UUIDs are
-- introduced in this change (the explore draft's UUID-based design was
-- corrected to BIGSERIAL; the rationale is in the explore.md §4).
--
-- Idempotency: IF NOT EXISTS / IF EXISTS guards on every object so the
-- migration is safe to re-run. The single-flight index uses CREATE
-- UNIQUE INDEX IF NOT EXISTS; the CHECK constraints on `status` and
-- `triggered_by` are part of the table definition and cannot be
-- conditionally added without DO blocks, which we accept as a one-time
-- cost (a re-run on a table that already has the columns fails on
-- the first ALTER; this is a known limitation and the migration is
-- never re-run in production after the first apply).
CREATE TABLE IF NOT EXISTS sync_job (
    id                BIGSERIAL    PRIMARY KEY,
    workspace_id      BIGINT       NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
    status            TEXT         NOT NULL CHECK (status IN ('pending','running','done','failed')),
    triggered_by      TEXT         NOT NULL CHECK (triggered_by IN ('auto_on_create','manual')),
    started_at        TIMESTAMPTZ  NULL,
    finished_at       TIMESTAMPTZ  NULL,
    commit_sha_after  TEXT         NULL,
    error_message     TEXT         NULL,
    error_code        TEXT         NULL,
    attempts          INT          NOT NULL DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);

ALTER TABLE sync_job OWNER TO queen;

ALTER TABLE workspace
    ADD COLUMN IF NOT EXISTS last_synced_at         TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS last_synced_commit_sha TEXT        NULL,
    ADD COLUMN IF NOT EXISTS default_branch         TEXT        NULL,
    ADD COLUMN IF NOT EXISTS last_sync_job_id       BIGINT      NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
         WHERE constraint_schema = 'public' AND table_name = 'workspace'
           AND constraint_name = 'workspace_last_sync_job_id_fkey'
    ) THEN
        ALTER TABLE workspace
            ADD CONSTRAINT workspace_last_sync_job_id_fkey
            FOREIGN KEY (last_sync_job_id) REFERENCES sync_job(id) ON DELETE SET NULL;
    END IF;
END
$$;

COMMENT ON TABLE sync_job IS
    'One row per sync attempt against a workspace. The partial unique index `sync_job_single_flight_uidx` guarantees at most one in-flight (pending|running) job per workspace. The handler returns 409 SYNC_ALREADY_RUNNING on a second concurrent POST. See R-WS-019 in openspec/changes/2026-07-08-workspace-sync-clone/specs/workspaces/spec.md.';

COMMENT ON COLUMN sync_job.status IS
    'Locked vocabulary: pending | running | done | failed. Enforced by CHECK constraint. Transitions: pending -> running -> done|failed. The workspace_syncer writes the final state via the callback.';

COMMENT ON COLUMN sync_job.triggered_by IS
    'Locked vocabulary: auto_on_create (the first sync enqueued by POST /workspaces) | manual (a subsequent POST /workspaces/:id/sync click). Used by the UI to label the sync origin in any future audit log.';

COMMENT ON COLUMN sync_job.commit_sha_after IS
    'The HEAD commit SHA of the cloned tree at the moment the sync completed. NULL while the job is pending or running. The workspace.last_synced_commit_sha column is denormalized from this field on `done`.';

COMMENT ON COLUMN sync_job.error_code IS
    'Machine-readable code on failure (e.g. WORKSPACE_PERMISSIONS_INSUFFICIENT, BRANCH_NOT_FOUND, CLONE_TIMEOUT). Maps to the workspace_syncer internal error vocabulary; the UI surfaces the human-readable error_message in the card.';

CREATE INDEX IF NOT EXISTS sync_job_workspace_id_idx ON sync_job(workspace_id);
CREATE INDEX IF NOT EXISTS sync_job_status_idx       ON sync_job(status);

-- Single-flight: at most one in-flight (pending|running) job per workspace.
-- The partial WHERE clause excludes terminal (done|failed) rows so a
-- workspace that has been synced before can still accept a new manual
-- sync without violating the unique constraint.
CREATE UNIQUE INDEX IF NOT EXISTS sync_job_single_flight_uidx
    ON sync_job(workspace_id)
    WHERE status IN ('pending','running');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Forward-only migration per openspec/AGENTS.md. The Down block is left
-- intentionally empty (the project's goose driver treats empty Down as
-- a no-op). To revert this change operationally, drop the new table
-- and the new workspace columns manually after taking a snapshot of
-- any data they may hold.
-- +goose StatementEnd
