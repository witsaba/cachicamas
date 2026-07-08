// Package application — the SyncService use case facade.
//
// PR-3a introduces EnqueueSync, GetLatestSyncJob, and the
// helper used by WorkspaceService.Create to auto-enqueue the
// first sync job. PR-3b wires the HTTP handlers; PR-3b also
// adds the ProcessSyncCallback use case that receives the
// workspace_syncer's POST /internal/sync-callback.
package application

import (
	"context"
	"errors"
	"log/slog"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// SyncService is the use case facade for sync_job operations.
// The single-flight invariant is enforced at the database level
// (the partial unique index); the service surfaces the violation
// as a typed ConflictError so the handler can return 409.
type SyncService struct {
	repo   domain.SyncJobRepository
	logger *slog.Logger
}

// NewSyncService constructs a SyncService.
func NewSyncService(repo domain.SyncJobRepository, logger *slog.Logger) *SyncService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncService{repo: repo, logger: logger}
}

// EnqueueSync inserts a new sync_job row with status='pending'
// and triggered_by set per the caller. The single-flight invariant
// means a second call for the same workspace_id (when the first
// job is still pending or running) returns ConflictError.
//
// The boolean indicates whether the returned job_id is a fresh
// insert (true) or an existing job (false). The handler uses this
// to decide between 201 (created) and 200 (already running).
//
// On a fresh insert: the boolean is true, the returned job is
// the new one, error is nil.
// On a single-flight hit: the boolean is false, the returned
// job is the EXISTING one (so the caller can poll its status),
// error is nil.
// On a different error: error is non-nil.
func (s *SyncService) EnqueueSync(ctx context.Context, workspaceID int64, triggeredBy string) (jobID int64, isFresh bool, existing *domain.SyncJob, err error) {
	job := &domain.SyncJob{
		WorkspaceID: workspaceID,
		Status:      domain.SyncJobStatusPending,
		TriggeredBy: triggeredBy,
		Attempts:    0,
	}
	persisted, err := s.repo.Insert(ctx, job)
	if err == nil {
		return persisted.ID, true, persisted, nil
	}
	var conflict *domain.ConflictError
	if !errors.As(err, &conflict) {
		return 0, false, nil, err
	}
	// Single-flight hit: fetch the existing job so the caller
	// can poll its status.
	existing, getErr := s.repo.GetLatestForWorkspace(ctx, workspaceID)
	if getErr != nil {
		return 0, false, nil, getErr
	}
	if existing == nil {
		// Race: the conflicting row was deleted between Insert
		// (which failed) and GetLatest (which returned nil).
		// The caller should retry.
		return 0, false, nil, errors.New("sync: single-flight race; retry")
	}
	return existing.ID, false, existing, nil
}

// GetLatestSyncJob returns the most recent sync_job for the
// given workspace_id, or (nil, nil) when no job exists. The
// application layer falls back to the "not synced yet" UI state
// when this returns nil.
func (s *SyncService) GetLatestSyncJob(ctx context.Context, workspaceID int64) (*domain.SyncJob, error) {
	return s.repo.GetLatestForWorkspace(ctx, workspaceID)
}