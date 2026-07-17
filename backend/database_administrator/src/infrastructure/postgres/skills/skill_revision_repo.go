// Package skills contains the Postgres adapter for the skill
// revisions hexagonal slice (PR1b of cachicamas-skills-foundational).
// This is the sibling file to skill_repo.go; both translate pgx
// errors into domain errors per the project's existing convention.
//
// The adapter implements domain.SkillRevisionRepository. The
// revisions table is append-only (the service never issues UPDATE or
// DELETE on skill_revision except via CASCADE when the parent skill
// is hard-deleted, which is not exposed by this repo).
package skills

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// SkillRevisionRepo is the pgx-backed adapter that satisfies
// domain.SkillRevisionRepository. Like SkillRepo, it uses the stdlib
// database/sql interface.
type SkillRevisionRepo struct {
	db *sql.DB
}

// NewSkillRevisionRepo constructs a SkillRevisionRepo.
func NewSkillRevisionRepo(db *sql.DB) *SkillRevisionRepo {
	return &SkillRevisionRepo{db: db}
}

// Compile-time check.
var _ domain.SkillRevisionRepository = (*SkillRevisionRepo)(nil)

// ---------------------------------------------------------------------------
// Locked query constants.
// ---------------------------------------------------------------------------

const (
	skillRevisionTableName  = "skill_revision"
	skillRevisionColumnList = "id, skill_id, revision_number, description, body, change_note, created_at"
)

// ---------------------------------------------------------------------------
// SkillRevision CRUD
// ---------------------------------------------------------------------------

// Insert persists a new skill_revision row. The DB sets id and
// created_at from column defaults; the adapter uses RETURNING to read
// the populated row back into the domain.SkillRevision pointer.
//
// A unique-violation on the (skill_id, revision_number) UNIQUE
// constraint is the smoking-gun for a lost-update race (two
// concurrent transactions computed the same next revision number).
// The service is supposed to prevent this with FOR UPDATE; if it
// still happens, the surface returns *domain.ConflictError so the
// handler can map to 409.
func (r *SkillRevisionRepo) Insert(ctx context.Context, db domain.SQLExecutor, rev *domain.SkillRevision) error {
	row := db.QueryRowContext(ctx,
		`INSERT INTO `+skillRevisionTableName+` (skill_id, revision_number, description, body, change_note)
             VALUES ($1, $2, $3, $4, $5)
             RETURNING `+skillRevisionColumnList,
		rev.SkillID, rev.RevisionNumber, rev.Description, rev.Body, rev.ChangeNote,
	)
	out, err := scanSkillRevision(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return &domain.ConflictError{Cause: err}
		}
		return fmt.Errorf("postgres.SkillRevisionRepo.Insert: %w", err)
	}
	*rev = *out
	return nil
}

// SelectBySkillAndNumber returns the specific revision, or
// *domain.NotFoundError when no such revision exists.
func (r *SkillRevisionRepo) SelectBySkillAndNumber(ctx context.Context, db domain.SQLExecutor, skillID int64, n int) (*domain.SkillRevision, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+skillRevisionColumnList+` FROM `+skillRevisionTableName+`
             WHERE skill_id = $1 AND revision_number = $2`,
		skillID, n,
	)
	rev, err := scanSkillRevision(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: skillRevisionTableName}
		}
		return nil, fmt.Errorf("postgres.SkillRevisionRepo.SelectBySkillAndNumber: %w", err)
	}
	return rev, nil
}

// ListBySkillID returns every revision for the skill, ordered by
// revision_number DESC (newest first). Returns an empty slice (NOT
// nil) when zero rows exist.
func (r *SkillRevisionRepo) ListBySkillID(ctx context.Context, db domain.SQLExecutor, skillID int64) ([]*domain.SkillRevision, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+skillRevisionColumnList+` FROM `+skillRevisionTableName+`
             WHERE skill_id = $1
             ORDER BY revision_number DESC`,
		skillID,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.SkillRevisionRepo.ListBySkillID: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*domain.SkillRevision{}
	for rows.Next() {
		rev, err := scanSkillRevision(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres.SkillRevisionRepo.ListBySkillID: scan: %w", err)
		}
		out = append(out, rev)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.SkillRevisionRepo.ListBySkillID: rows: %w", err)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Scanner.
// ---------------------------------------------------------------------------

func scanSkillRevision(s skillRowScanner) (*domain.SkillRevision, error) {
	var rev domain.SkillRevision
	var changeNote sql.NullString
	if err := s.Scan(
		&rev.ID,
		&rev.SkillID,
		&rev.RevisionNumber,
		&rev.Description,
		&rev.Body,
		&changeNote,
		&rev.CreatedAt,
	); err != nil {
		return nil, err
	}
	if changeNote.Valid {
		cn := changeNote.String
		rev.ChangeNote = &cn
	}
	return &rev, nil
}
