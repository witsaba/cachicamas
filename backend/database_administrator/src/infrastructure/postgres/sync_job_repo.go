// Package postgres — the pgx adapter for the SyncJobRepository
// port. Mirrors the pattern in workspace_repo.go.
//
// SQLSTATE handling: the partial unique index
// sync_job_single_flight_uidx returns 23505 on a duplicate
// (workspace_id, status IN ('pending','running')) insert. We
// translate that to *domain.ConflictError so the handler can
// return 409 without importing pgx.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// syncJobRepo is the pgx-backed implementation of
// domain.SyncJobRepository.
type syncJobRepo struct {
	db *sql.DB
}

// NewSyncJobRepo constructs a syncJobRepo. The caller passes an
// already-opened *sql.DB (typically produced by
// migration/postgres.Open in the composition root).
func NewSyncJobRepo(db *sql.DB) domain.SyncJobRepository {
	return &syncJobRepo{db: db}
}

// Insert persists a new SyncJob. The DB sets id and created_at
// from column defaults; RETURNING reads the populated row back.
//
// A unique-violation on the single-flight partial index is
// translated to *domain.ConflictError. Other errors are wrapped.
func (r *syncJobRepo) Insert(ctx context.Context, job *domain.SyncJob) (*domain.SyncJob, error) {
	row := r.db.QueryRowContext(ctx, `
		INSERT INTO sync_job (workspace_id, status, triggered_by, attempts)
		VALUES ($1, $2, $3, $4)
		RETURNING id, workspace_id, status, triggered_by, started_at,
		          finished_at, commit_sha_after, error_message,
		          error_code, attempts, created_at
	`, job.WorkspaceID, job.Status, job.TriggeredBy, job.Attempts)

	out, err := scanSyncJob(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, &domain.ConflictError{Cause: err}
		}
		return nil, fmt.Errorf("postgres.syncJobRepo.Insert: %w", err)
	}
	return out, nil
}

// GetLatestForWorkspace returns the most recent sync_job row for
// the given workspace_id, or (nil, nil) when no job exists. The
// ORDER BY id DESC ensures "most recent" picks the latest insert
// (BIGSERIAL ids are monotonic).
func (r *syncJobRepo) GetLatestForWorkspace(ctx context.Context, workspaceID int64) (*domain.SyncJob, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, workspace_id, status, triggered_by, started_at,
		       finished_at, commit_sha_after, error_message,
		       error_code, attempts, created_at
		  FROM sync_job
		 WHERE workspace_id = $1
		 ORDER BY id DESC
		 LIMIT 1
	`, workspaceID)

	out, err := scanSyncJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres.syncJobRepo.GetLatestForWorkspace: %w", err)
	}
	return out, nil
}

// Update modifies an existing SyncJob row. The WHERE clause
// filters on id AND status='running' so a callback that arrives
// twice for the same job is idempotent (the second Update
// matches no rows; we return nil, not an error).
//
// The method does NOT change the id, workspace_id, or created_at
// fields; those are immutable.
func (r *syncJobRepo) Update(ctx context.Context, job *domain.SyncJob) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE sync_job
		   SET status = $2,
		       started_at = $3,
		       finished_at = $4,
		       commit_sha_after = $5,
		       error_message = $6,
		       error_code = $7,
		       attempts = $8
		 WHERE id = $1
		   AND status = 'running'
	`, job.ID, job.Status, job.StartedAt, job.FinishedAt, job.CommitSHAAfter, job.ErrorMessage, job.ErrorCode, job.Attempts)
	if err != nil {
		return fmt.Errorf("postgres.syncJobRepo.Update: %w", err)
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres.syncJobRepo.Update rows-affected: %w", err)
	}
	// 0 rows is OK: the job was already updated (idempotent
	// callback). >0 rows is the normal case. The caller does
	// not need to know which.
	_ = rows
	return nil
}

// scanSyncJob reads one row from a SELECT (or RETURNING) into a
// fresh *domain.SyncJob. The column order MUST match the
// SELECT/RETURNING above.
func scanSyncJob(row interface {
	Scan(dest ...any) error
}) (*domain.SyncJob, error) {
	var (
		j              domain.SyncJob
		startedAt      sql.NullTime
		finishedAt     sql.NullTime
		commitSHAAfter sql.NullString
		errorMessage   sql.NullString
		errorCode      sql.NullString
	)
	if err := row.Scan(
		&j.ID,
		&j.WorkspaceID,
		&j.Status,
		&j.TriggeredBy,
		&startedAt,
		&finishedAt,
		&commitSHAAfter,
		&errorMessage,
		&errorCode,
		&j.Attempts,
		&j.CreatedAt,
	); err != nil {
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

// Compile-time check that syncJobRepo satisfies the port.
var _ domain.SyncJobRepository = (*syncJobRepo)(nil)