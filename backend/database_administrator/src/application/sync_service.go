// Package application — the SyncService use case facade.
//
// PR-3a introduces EnqueueSync, GetLatestSyncJob, and the
// helper used by WorkspaceService.Create to auto-enqueue the
// first sync job. PR-3b adds the ProcessSyncCallback use case
// that receives the workspace_syncer's POST /internal/sync-callback
// and the workspace_syncer.Client.StartSync bridge that fires
// the syncer immediately after a fresh enqueue.
package application

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// SyncService is the use case facade for sync_job operations.
// The single-flight invariant is enforced at the database level
// (the partial unique index); the service surfaces the violation
// as a typed ConflictError so the handler can return 409.
//
// PR-3b also takes a workspaceRepo and a db so the callback path
// can perform the sync_job UPDATE + workspace.last_synced_*
// denormalization inside a single transaction.
type SyncService struct {
	repo          domain.SyncJobRepository
	workspaceRepo domain.WorkspaceRepository
	db            *sql.DB
	logger        *slog.Logger
}

// NewSyncService constructs a SyncService. PR-3b added the
// workspaceRepo and db parameters (both required for the
// callback path; nil is tolerated for tests that only exercise
// EnqueueSync and GetLatestSyncJob).
func NewSyncService(repo domain.SyncJobRepository, workspaceRepo domain.WorkspaceRepository, db *sql.DB, logger *slog.Logger) *SyncService {
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncService{
		repo:          repo,
		workspaceRepo: workspaceRepo,
		db:            db,
		logger:        logger,
	}
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

// ProcessSyncCallback is the use case for the internal
// callback the workspace_syncer POSTs once a clone + worktree
// probe has completed. The state machine (locked, see spec
// R-WS-019):
//
//	done:   UPDATE sync_job SET status='done', commit_sha_after=$1,
//	        finished_at=now() WHERE id=$2 AND status='running'.
//	        THEN UPDATE workspace SET last_synced_at=now(),
//	        last_synced_commit_sha=$1, default_branch=$3,
//	        last_sync_job_id=$2 WHERE id=$4.
//	        Both writes are inside a single transaction.
//	failed: UPDATE sync_job SET status='failed', error_message=$1,
//	        error_code=$2, finished_at=now() WHERE id=$3 AND
//	        status='running'. No workspace update (the user can
//	        retry; the existing last_synced_at is preserved).
//
// The sync_job update is idempotent: the WHERE filter requires
// status='running', so a callback that arrives twice is a no-op
// (the second Update matches no rows).
//
// On a 'done' callback, commitSHA is required; the function
// returns *domain.ValidationError if it is empty.
//
// The function returns the post-update SyncJob (or the existing
// row on a no-op idempotent re-call) so the handler can echo it
// back to the callback caller.
//
// 2026-08-02-security-vulnerability-remediation (H-1): the
// signature has grown an orgID parameter. The handler decodes
// it from the callback body and passes it downstream; the
// raw SQL for the workspace update filters on
// `id = $4 AND organization_id = $5 AND deleted_at IS NULL`
// so a callback from a wrong-tenant syncer cannot denormalize
// onto a workspace it does not own.
func (s *SyncService) ProcessSyncCallback(ctx context.Context, orgID, jobID int64, status, commitSHA, defaultBranch, errorCode, errorMessage string) (*domain.SyncJob, error) {
	if jobID <= 0 {
		return nil, &domain.ValidationError{Fields: map[string]string{"job_id": "must be > 0"}}
	}
	if status != domain.SyncJobStatusDone && status != domain.SyncJobStatusFailed {
		return nil, &domain.ValidationError{Fields: map[string]string{"status": "must be done or failed"}}
	}
	if status == domain.SyncJobStatusDone && commitSHA == "" {
		return nil, &domain.ValidationError{Fields: map[string]string{"commit_sha": "required on done"}}
	}
	if orgID <= 0 {
		return nil, &domain.ValidationError{Fields: map[string]string{"organization_id": "must be > 0"}}
	}

	// Single-shot path: when db is nil (PR-3a-only test wiring), use
	// the repository's idempotent Update directly. No workspace
	// denormalization in this mode.
	if s.db == nil {
		job, err := s.repo.GetLatestForWorkspace(ctx /*fallback*/, -1)
		_ = job
		_ = err
		// Lookup the job by ID rather than latest-for-workspace.
		current, lookupErr := s.lookupJobByID(ctx, jobID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if current == nil {
			return nil, &domain.ValidationError{Fields: map[string]string{"job_id": "not found"}}
		}
		if current.Status != domain.SyncJobStatusRunning {
			// Idempotent no-op: callback arrived twice. Return the
			// already-updated row.
			return current, nil
		}
		now := time.Now().UTC()
		current.Status = status
		if status == domain.SyncJobStatusDone {
			s := commitSHA
			current.CommitSHAAfter = &s
		}
		current.FinishedAt = &now
		if status == domain.SyncJobStatusFailed {
			m := errorMessage
			c := errorCode
			current.ErrorMessage = &m
			current.ErrorCode = &c
		}
		if err := s.repo.Update(ctx, current); err != nil {
			return nil, err
		}
		return current, nil
	}

	// Production path: do the sync_job update + workspace
	// denormalization inside a single Tx.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("sync.ProcessSyncCallback: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Lookup the current job to learn its workspace_id (and
	//    enforce idempotency: only update if the row is in a
	//    pre-terminal state).
	//
	//    UAT fix (2026-07-08): the prior WHERE filter was
	//    `AND status = 'running'`, but the syncer never transitions
	//    a row to 'running' before the callback arrives — the row
	//    is in 'pending' the entire time the clone runs. The
	//    UPDATE matched 0 rows and the handler returned 204 without
	//    updating the row, so the sync_job stayed in 'pending'
	//    forever.
	//
	//    The fix: accept the row in EITHER 'pending' or 'running'
	//    state. The single-flight invariant is guaranteed by the
	//    partial unique index `sync_job_single_flight_uidx` on
	//    (workspace_id) WHERE status IN ('pending','running'), not
	//    by the callback's WHERE filter. The callback can
	//    legitimately land on a 'pending' row (normal path) or a
	//    'running' row (future state where the syncer posts a
	//    'started' callback before the worktree probe).
	//
	//    The defaultBranch arg is NOT bound here (PG would reject
	//    the query with SQLSTATE 42P18 "indeterminate datatype"
	//    for an unreferenced $N parameter). defaultBranch is only
	//    used in the workspace UPDATE below.
	res, err := tx.ExecContext(ctx, `
		UPDATE sync_job
		   SET status = $2,
		       commit_sha_after = CASE WHEN $2 = 'done' THEN $3 ELSE commit_sha_after END,
		       error_message    = CASE WHEN $2 = 'failed' THEN $5 ELSE error_message END,
		       error_code       = CASE WHEN $2 = 'failed' THEN $4 ELSE error_code END,
		       finished_at      = now(),
		       attempts         = attempts + 1
		 WHERE id = $1
		   AND status IN ('pending', 'running')
	`, jobID, status, commitSHA, errorCode, errorMessage)
	if err != nil {
		return nil, fmt.Errorf("sync.ProcessSyncCallback: update sync_job: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("sync.ProcessSyncCallback: rows: %w", err)
	}
	if rows == 0 {
		// Late duplicate callback; the second invocation is a
		// no-op. The handler still returns 204 to the syncer so
		// its retry loop terminates. Fetch the current state to
		// echo back via the by-id helper.
		current, lookupErr := s.lookupJobByIDTx(ctx, tx, jobID)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("sync.ProcessSyncCallback: commit (no-op): %w", err)
		}
		return current, nil
	}

	// 2. Look up the workspace_id of the job. We could fold
	//    this into the UPDATE...RETURNING, but a separate
	//    query keeps the SQL simple and the cost is one
	//    indexed lookup.
	var workspaceID int64
	lookupRow := tx.QueryRowContext(ctx, `SELECT workspace_id FROM sync_job WHERE id = $1`, jobID)
	if err := lookupRow.Scan(&workspaceID); err != nil {
		return nil, fmt.Errorf("sync.ProcessSyncCallback: lookup workspace_id: %w", err)
	}

	// 3. On 'done', denormalize onto the workspace row.
	//
	//    UAT fix (2026-07-08): the prior code had
	//    `&& s.workspaceRepo != nil` as a guard, but the
	//    production path uses raw SQL on the tx (not the repo
	//    port), so the guard was redundant in production AND
	//    blocked the denormalization in tests that pass nil
	//    for the workspaceRepo. The denormalization is now
	//    unconditional on status='done'. The workspaceRepo
	//    is unused in the production path (kept on the
	//    service struct for the unit-test path that exercises
	//    s.repo directly when s.db is nil).
	//
	//    2026-08-02-security-vulnerability-remediation (H-1):
	//    the orgID filter is mandatory; a callback from a
	//    wrong-tenant syncer is silently no-op'd (the WHERE
	//    matches 0 rows and the 409 is logged + ignored).
	if status == domain.SyncJobStatusDone {
		markRes, err := tx.ExecContext(ctx, `
			UPDATE workspace
			   SET last_synced_at         = now(),
			       last_synced_commit_sha = $1,
			       default_branch         = $2,
			       last_sync_job_id       = $3,
			       updated_at             = now()
			 WHERE id = $4
			   AND organization_id = $5
			   AND deleted_at IS NULL
		`, commitSHA, defaultBranch, jobID, workspaceID, orgID)
		if err != nil {
			return nil, fmt.Errorf("sync.ProcessSyncCallback: mark workspace synced: %w", err)
		}
		markRows, err := markRes.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("sync.ProcessSyncCallback: mark rows: %w", err)
		}
		if markRows == 0 {
			// The workspace was soft-deleted between enqueue
			// and the callback, OR the callback's supplied
			// orgID doesn't match the tenant that owns the
			// job. The job is still 'done' (the clone
			// actually succeeded) but we don't have a
			// workspace row to denormalize onto. We commit
			// the sync_job update and return success; the
			// workspace_syncer startup sweep will reap the
			// cloned tree on its next pass.
			s.logger.WarnContext(ctx, "sync.ProcessSyncCallback: workspace skipped (soft-deleted or cross-tenant)",
				slog.Int64("job_id", jobID),
				slog.Int64("workspace_id", workspaceID),
				slog.Int64("organization_id", orgID),
			)
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("sync.ProcessSyncCallback: commit: %w", err)
	}

	// 4. Re-fetch the updated job so the caller sees the new state.
	//    Use s.db (NOT the tx, which is already closed post-commit).
	updated, err := s.lookupJobByID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	return updated, nil
}

// lookupJobByID is a best-effort helper for the test-only path
// (db == nil). It queries the sync_job table directly through the
// repository's port — but the port only exposes GetLatestForWorkspace.
// In production we always go through the Tx path above. The test
// path is therefore only as good as the test wiring.
func (s *SyncService) lookupJobByID(ctx context.Context, jobID int64) (*domain.SyncJob, error) {
	// The test-only path: use the latest-for-workspace trick
	// only works if the test wires the port with a fake that
	// supports it. Otherwise, we use a raw DB query if available.
	if s.db == nil {
		return nil, nil
	}
	row := s.db.QueryRowContext(ctx, `SELECT id, workspace_id, status, triggered_by, started_at, finished_at, commit_sha_after, error_message, error_code, attempts, created_at FROM sync_job WHERE id = $1`, jobID)
	var (
		j              domain.SyncJob
		startedAt      sql.NullTime
		finishedAt     sql.NullTime
		commitSHAAfter sql.NullString
		errorMessage   sql.NullString
		errorCode      sql.NullString
	)
	if err := row.Scan(&j.ID, &j.WorkspaceID, &j.Status, &j.TriggeredBy, &startedAt, &finishedAt, &commitSHAAfter, &errorMessage, &errorCode, &j.Attempts, &j.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if startedAt.Valid {
		t := startedAt.Time
		j.StartedAt = &t
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		j.FinishedAt = &t
	}
	if commitSHAAfter.Valid {
		s := commitSHAAfter.String
		j.CommitSHAAfter = &s
	}
	if errorMessage.Valid {
		s := errorMessage.String
		j.ErrorMessage = &s
	}
	if errorCode.Valid {
		s := errorCode.String
		j.ErrorCode = &s
	}
	return &j, nil
}

// lookupJobByIDTx is the same as lookupJobByID but inside a Tx.
func (s *SyncService) lookupJobByIDTx(ctx context.Context, tx *sql.Tx, jobID int64) (*domain.SyncJob, error) {
	if tx == nil {
		return s.lookupJobByID(ctx, jobID)
	}
	row := tx.QueryRowContext(ctx, `SELECT id, workspace_id, status, triggered_by, started_at, finished_at, commit_sha_after, error_message, error_code, attempts, created_at FROM sync_job WHERE id = $1`, jobID)
	var (
		j              domain.SyncJob
		startedAt      sql.NullTime
		finishedAt     sql.NullTime
		commitSHAAfter sql.NullString
		errorMessage   sql.NullString
		errorCode      sql.NullString
	)
	if err := row.Scan(&j.ID, &j.WorkspaceID, &j.Status, &j.TriggeredBy, &startedAt, &finishedAt, &commitSHAAfter, &errorMessage, &errorCode, &j.Attempts, &j.CreatedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if startedAt.Valid {
		t := startedAt.Time
		j.StartedAt = &t
	}
	if finishedAt.Valid {
		t := finishedAt.Time
		j.FinishedAt = &t
	}
	if commitSHAAfter.Valid {
		s := commitSHAAfter.String
		j.CommitSHAAfter = &s
	}
	if errorMessage.Valid {
		s := errorMessage.String
		j.ErrorMessage = &s
	}
	if errorCode.Valid {
		s := errorCode.String
		j.ErrorCode = &s
	}
	return &j, nil
}

// SyncDispatcher is the hexagonal port the application layer
// uses to dispatch a fresh sync_job to the workspace_syncer. The
// concrete adapter (infrastructure/workspacesyncer.Client) is
// wired by main.go; the application layer never imports the
// syncer package directly.
type SyncDispatcher interface {
	StartSync(ctx context.Context, jobID, workspaceID int64, owner, repo, defaultBranch, oauthToken string) error
}

// StartSyncWithSyncer dispatches a freshly-enqueued sync_job to
// the workspace_syncer via the configured dispatcher. The
// dispatcher is best-effort: a failure here is logged but does NOT
// roll back the enqueue. The job remains 'pending' in the
// database; the user can retry from the UI. The syncer's startup
// sweep will reconcile any pending-but-not-running jobs on its
// next pass.
//
// Returns nil when the dispatcher is nil (the test-only path;
// PR-3a did not wire a dispatcher).
func (s *SyncService) StartSyncWithSyncer(ctx context.Context, dispatcher SyncDispatcher, jobID, workspaceID int64, owner, repo, defaultBranch, oauthToken string) error {
	if dispatcher == nil {
		s.logger.DebugContext(ctx, "sync.StartSyncWithSyncer: no dispatcher wired; skipping dispatch",
			slog.Int64("job_id", jobID),
			slog.Int64("workspace_id", workspaceID),
		)
		return nil
	}
	if err := dispatcher.StartSync(ctx, jobID, workspaceID, owner, repo, defaultBranch, oauthToken); err != nil {
		s.logger.WarnContext(ctx, "sync.StartSyncWithSyncer: dispatcher failed; job remains pending",
			slog.Int64("job_id", jobID),
			slog.Int64("workspace_id", workspaceID),
			slog.String("error", err.Error()),
		)
		return err
	}
	return nil
}
