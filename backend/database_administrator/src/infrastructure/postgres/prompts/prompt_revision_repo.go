// Package postgres contains the Postgres adapter for the prompt
// revisions hexagonal slice (PR2 of 2026-07-15-prompt-storage-table).
// This is the sibling file to prompt_repo.go; both translate pgx
// errors into domain errors per the project's existing convention.
//
// The adapter implements domain.PromptRevisionRepository. The
// revisions table is append-only (the service never issues UPDATE or
// DELETE on prompt_revision except via CASCADE when the parent prompt
// is hard-deleted, which is not exposed by this repo).
package prompts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cachicamas/backend/database_administrator/src/domain"

	"github.com/jackc/pgx/v5/pgconn"
)

// PromptRevisionRepo is the pgx-backed adapter that satisfies
// domain.PromptRevisionRepository. Like PromptRepo, it uses the stdlib
// database/sql interface.
type PromptRevisionRepo struct {
	db *sql.DB
}

// NewPromptRevisionRepo constructs a PromptRevisionRepo.
func NewPromptRevisionRepo(db *sql.DB) *PromptRevisionRepo {
	return &PromptRevisionRepo{db: db}
}

// Compile-time check.
var _ domain.PromptRevisionRepository = (*PromptRevisionRepo)(nil)

// ---------------------------------------------------------------------------
// Locked query constants.
// ---------------------------------------------------------------------------

const (
	promptRevisionTableName  = "prompt_revision"
	promptRevisionColumnList = "id, prompt_id, revision_number, description, body, change_note, created_by, created_at"
)

// ---------------------------------------------------------------------------
// PromptRevision CRUD
// ---------------------------------------------------------------------------

// Insert persists a new prompt_revision row. The DB sets id and
// created_at from column defaults; the adapter uses RETURNING to read
// the populated row back into the domain.PromptRevision pointer.
//
// A unique-violation on the (prompt_id, revision_number) UNIQUE
// constraint is the smoking-gun for a lost-update race (two
// concurrent transactions computed the same next revision number).
// The service is supposed to prevent this with FOR UPDATE; if it
// still happens, the surface returns *domain.ConflictError so the
// handler can map to 409.
func (r *PromptRevisionRepo) Insert(ctx context.Context, db domain.SqlExecutor, rev *domain.PromptRevision) error {
	row := db.QueryRowContext(ctx,
		`INSERT INTO `+promptRevisionTableName+` (prompt_id, revision_number, description, body, change_note, created_by)
             VALUES ($1, $2, $3, $4, $5, $6)
             RETURNING `+promptRevisionColumnList,
		rev.PromptID, rev.RevisionNumber, rev.Description, rev.Body, rev.ChangeNote, rev.CreatedBy,
	)
	out, err := scanPromptRevision(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return &domain.ConflictError{Cause: err}
		}
		return fmt.Errorf("postgres.PromptRevisionRepo.Insert: %w", err)
	}
	*rev = *out
	return nil
}

// SelectLatestForPrompt returns the highest revision_number for the
// prompt, or *domain.NotFoundError when none exist. The DB invariant
// says every prompt has at least revision 1, so a "no rows" result
// indicates a DB-level invariant violation.
func (r *PromptRevisionRepo) SelectLatestForPrompt(ctx context.Context, db domain.SqlExecutor, promptID int64) (*domain.PromptRevision, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+promptRevisionColumnList+` FROM `+promptRevisionTableName+`
             WHERE prompt_id = $1
             ORDER BY revision_number DESC
             LIMIT 1`,
		promptID,
	)
	rev, err := scanPromptRevision(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: promptRevisionTableName}
		}
		return nil, fmt.Errorf("postgres.PromptRevisionRepo.SelectLatestForPrompt: %w", err)
	}
	return rev, nil
}

// SelectByPromptAndNumber returns the specific revision, or
// *domain.NotFoundError when no such revision exists.
func (r *PromptRevisionRepo) SelectByPromptAndNumber(ctx context.Context, db domain.SqlExecutor, promptID int64, n int) (*domain.PromptRevision, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+promptRevisionColumnList+` FROM `+promptRevisionTableName+`
             WHERE prompt_id = $1 AND revision_number = $2`,
		promptID, n,
	)
	rev, err := scanPromptRevision(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: promptRevisionTableName}
		}
		return nil, fmt.Errorf("postgres.PromptRevisionRepo.SelectByPromptAndNumber: %w", err)
	}
	return rev, nil
}

// SelectListByPrompt returns every revision for the prompt, ordered
// by revision_number DESC (newest first). Returns an empty slice
// (NOT nil) when zero rows exist.
func (r *PromptRevisionRepo) SelectListByPrompt(ctx context.Context, db domain.SqlExecutor, promptID int64) ([]*domain.PromptRevision, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+promptRevisionColumnList+` FROM `+promptRevisionTableName+`
             WHERE prompt_id = $1
             ORDER BY revision_number DESC`,
		promptID,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.PromptRevisionRepo.SelectListByPrompt: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*domain.PromptRevision{}
	for rows.Next() {
		rev, err := scanPromptRevision(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres.PromptRevisionRepo.SelectListByPrompt: scan: %w", err)
		}
		out = append(out, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.PromptRevisionRepo.SelectListByPrompt: rows: %w", err)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Scanner.
// ---------------------------------------------------------------------------

func scanPromptRevision(s promptRowScanner) (*domain.PromptRevision, error) {
	var rev domain.PromptRevision
	var changeNote, createdBy sql.NullString
	if err := s.Scan(
		&rev.ID,
		&rev.PromptID,
		&rev.RevisionNumber,
		&rev.Description,
		&rev.Body,
		&changeNote,
		&createdBy,
		&rev.CreatedAt,
	); err != nil {
		return nil, err
	}
	if changeNote.Valid {
		s := changeNote.String
		rev.ChangeNote = &s
	}
	if createdBy.Valid {
		s := createdBy.String
		rev.CreatedBy = &s
	}
	return &rev, nil
}