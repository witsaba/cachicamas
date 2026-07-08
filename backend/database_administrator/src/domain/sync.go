// sync.go — the SyncJob aggregate and SyncJobRepository port for
// the 2026-07-08-workspace-sync-clone change. The migration
// `20260708120200_sync_job.sql` (PR-1) creates the underlying
// table; this file defines the Go-side surface.
//
// The SyncJob records one row per sync attempt. The lifecycle is:
// pending -> running -> done|failed. The single-flight invariant
// (at most one in-flight job per workspace_id) is enforced at
// the database level via a partial unique index; the Go code
// surfaces the violation as a *ConflictError.
package domain

import (
	"context"
	"time"
)

// SyncJobStatus is the locked vocabulary for the
// sync_job.status column. Mirrors the CHECK constraint in the
// migration. The Go type is a string (not a custom type) so
// callers can pass it through the wire and the database without
// conversion; the SQL CHECK constraint is the single source of
// truth for validity.
const (
	SyncJobStatusPending = "pending"
	SyncJobStatusRunning = "running"
	SyncJobStatusDone    = "done"
	SyncJobStatusFailed  = "failed"
)

// SyncJobTrigger is the locked vocabulary for the
// sync_job.triggered_by column.
const (
	SyncJobTriggerAutoOnCreate = "auto_on_create"
	SyncJobTriggerManual       = "manual"
)

// SyncJob is the Go-side mirror of one row of the public.sync_job
// table. Nullable columns are represented as pointer types so the
// caller can distinguish "not set" from "set to the zero value".
type SyncJob struct {
	ID             int64
	WorkspaceID    int64
	Status         string
	TriggeredBy    string
	StartedAt      *time.Time
	FinishedAt     *time.Time
	CommitSHAAfter *string
	ErrorMessage   *string
	ErrorCode      *string
	Attempts       int
	CreatedAt      time.Time
}

// SyncJobRepository is the hexagonal port that the application
// layer depends on. Implementations live in
// infrastructure/postgres/sync_job_repo.go (PR-3a).
//
// Contract:
//   - Insert MUST return *ConflictError on a unique-violation
//     (the single-flight partial index). The application layer
//     surfaces this as 409 SYNC_ALREADY_RUNNING.
//   - GetLatestForWorkspace MUST return (nil, nil) when no job
//     exists for the given workspace_id. The application layer
//     falls back to the "not synced yet" UI state.
//   - Update MUST filter on the current status. A callback
//     that arrives twice for the same job MUST be idempotent
//     (the second Update is a no-op, returning nil).
type SyncJobRepository interface {
	Insert(ctx context.Context, job *SyncJob) (*SyncJob, error)
	GetLatestForWorkspace(ctx context.Context, workspaceID int64) (*SyncJob, error)
	Update(ctx context.Context, job *SyncJob) error
}
