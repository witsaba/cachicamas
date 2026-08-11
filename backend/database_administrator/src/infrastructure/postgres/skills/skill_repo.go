// Package skills contains the Postgres adapters for the skills
// hexagonal slice (PR1b of cachicamas-skills-foundational). Mirrors
// the prompts adapter at infrastructure/postgres/prompts/. pgconn
// errors (SQLSTATE 23505) translate to *domain.ConflictError; "no
// rows" to *domain.NotFoundError, so handlers don't import pgx.
//
// Methods accept a `domain.sqlExecutor` (which *sql.DB and *sql.Tx
// both satisfy) so the application layer can compose multi-statement
// operations in a single transaction without leaking pgx types.
package skills

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

const pgUniqueViolation = "23505"

// SkillRepo is the pgx-backed adapter for domain.SkillRepository.
// db is held only for tests / one-shot ops; production methods take
// an explicit sqlExecutor parameter so the service can decide
// pool-vs-transaction.
type SkillRepo struct {
	db *sql.DB
}

// NewSkillRepo returns a SkillRepo backed by the supplied
// *sql.DB. The repo satisfies domain.SkillRepository (compile-time
// checked on the next line).
func NewSkillRepo(db *sql.DB) *SkillRepo { return &SkillRepo{db: db} }

// Compile-time check.
var _ domain.SkillRepository = (*SkillRepo)(nil)

const (
	skillTableName        = "skill"
	skillColumnList       = "id, name, description, body, deleted_at, created_at, updated_at"
	skillInsertColumnList = "name, description, body"
	skillLiveUniqueIndex  = "skill_slug_active_uidx"
)

// currentRevisionSubquery is the locked SQL fragment for the JOIN
// against skill_revision (anti-drift gate ADR-SK-008 — kills the
// "v{undefined}" prompt bug from obs #1959 #2). Defined once so the
// SQL structure is identical between SelectByIDWithCurrentRevision
// and ListWithCurrentRevision.
const currentRevisionSubquery = `(SELECT skill_id, MAX(revision_number) AS current_revision FROM skill_revision GROUP BY skill_id)`

// Insert persists a new skill row (DB sets id/created_at/updated_at/
// deleted_at from defaults). A unique-violation on
// `skill_slug_active_uidx` translates to *domain.ConflictError.
func (r *SkillRepo) Insert(ctx context.Context, db domain.SQLExecutor, s *domain.Skill) error {
	row := db.QueryRowContext(ctx,
		`INSERT INTO `+skillTableName+` (`+skillInsertColumnList+`)
             VALUES ($1, $2, $3)
             RETURNING `+skillColumnList,
		s.Name, s.Description, s.Body,
	)
	out, err := scanSkill(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return &domain.ConflictError{Cause: err}
		}
		return fmt.Errorf("postgres.SkillRepo.Insert: %w", err)
	}
	*s = *out
	return nil
}

// SelectBySlug returns the LIVE row (deleted_at IS NULL) with the
// given name. Partial UNIQUE + WHERE deleted_at IS NULL ensure
// soft-deleted rows are invisible (spec R-SK-002).
func (r *SkillRepo) SelectBySlug(ctx context.Context, db domain.SQLExecutor, name string) (*domain.Skill, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+skillColumnList+` FROM `+skillTableName+`
             WHERE name = $1 AND deleted_at IS NULL`,
		name,
	)
	s, err := scanSkill(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: skillTableName}
		}
		return nil, fmt.Errorf("postgres.SkillRepo.SelectBySlug: %w", err)
	}
	return s, nil
}

// SelectBySlugAny returns the row regardless of deleted_at state.
// The service inspects DeletedAt to distinguish 404 from 410.
func (r *SkillRepo) SelectBySlugAny(ctx context.Context, db domain.SQLExecutor, name string) (*domain.Skill, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+skillColumnList+` FROM `+skillTableName+`
             WHERE name = $1`,
		name,
	)
	s, err := scanSkill(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: skillTableName}
		}
		return nil, fmt.Errorf("postgres.SkillRepo.SelectBySlugAny: %w", err)
	}
	return s, nil
}

// SelectByID returns the row by PK regardless of deleted_at state.
// Caller is typically the service after LockAndLoad or after a write
// (for the response re-read).
func (r *SkillRepo) SelectByID(ctx context.Context, db domain.SQLExecutor, id int64) (*domain.Skill, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+skillColumnList+` FROM `+skillTableName+`
             WHERE id = $1`,
		id,
	)
	s, err := scanSkill(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: skillTableName}
		}
		return nil, fmt.Errorf("postgres.SkillRepo.SelectByID: %w", err)
	}
	return s, nil
}

// SelectByIDWithCurrentRevision returns Skill + current revision via
// a single LEFT JOIN (no N+1). Skills with no revisions yield
// CurrentRevision=0 (COALESCE).
func (r *SkillRepo) SelectByIDWithCurrentRevision(ctx context.Context, db domain.SQLExecutor, id int64) (*domain.SkillDetail, error) {
	row := db.QueryRowContext(ctx,
		`SELECT s.id, s.name, s.description, s.body, s.deleted_at, s.created_at, s.updated_at,
                COALESCE(r.current_revision, 0) AS current_revision
             FROM `+skillTableName+` s
             LEFT JOIN `+currentRevisionSubquery+` r ON r.skill_id = s.id
             WHERE s.id = $1`,
		id,
	)
	out, err := scanSkillDetail(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: skillTableName}
		}
		return nil, fmt.Errorf("postgres.SkillRepo.SelectByIDWithCurrentRevision: %w", err)
	}
	return out, nil
}

// List returns active skills ordered by (updated_at DESC, id DESC).
// Service clamps limit to MaxListLimit before reaching the repo.
func (r *SkillRepo) List(ctx context.Context, db domain.SQLExecutor, limit int) ([]*domain.Skill, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+skillColumnList+` FROM `+skillTableName+`
             WHERE deleted_at IS NULL
             ORDER BY updated_at DESC, id DESC
             LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.SkillRepo.List: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*domain.Skill{}
	for rows.Next() {
		s, err := scanSkill(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres.SkillRepo.List: scan: %w", err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.SkillRepo.List: rows: %w", err)
	}
	return out, nil
}

// ListWithCurrentRevision returns active skills joined with their
// current revision. ONE SQL statement (anti-drift ADR-SK-008).
func (r *SkillRepo) ListWithCurrentRevision(ctx context.Context, db domain.SQLExecutor, limit int) ([]*domain.SkillListItem, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT s.name, s.description, COALESCE(r.current_revision, 0) AS current_revision, s.updated_at
             FROM `+skillTableName+` s
             LEFT JOIN `+currentRevisionSubquery+` r ON r.skill_id = s.id
             WHERE s.deleted_at IS NULL
             ORDER BY s.updated_at DESC, s.id DESC
             LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.SkillRepo.ListWithCurrentRevision: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*domain.SkillListItem{}
	for rows.Next() {
		var it domain.SkillListItem
		if err := rows.Scan(&it.Name, &it.Description, &it.CurrentRevision, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("postgres.SkillRepo.ListWithCurrentRevision: scan: %w", err)
		}
		out = append(out, &it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.SkillRepo.ListWithCurrentRevision: rows: %w", err)
	}
	return out, nil
}

// UpdateBody updates body, description, updated_at on a LIVE skill
// (under FOR UPDATE). DB clock owns updated_at.
func (r *SkillRepo) UpdateBody(ctx context.Context, db domain.SQLExecutor, id int64, body, description string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE `+skillTableName+`
             SET body = $1, description = $2, updated_at = now()
             WHERE id = $3 AND deleted_at IS NULL`,
		body, description, id,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return &domain.ConflictError{Cause: err}
		}
		return fmt.Errorf("postgres.SkillRepo.UpdateBody: %w", err)
	}
	return nil
}

// SoftDelete sets deleted_at = now(). Idempotent (second call finds
// zero rows matching deleted_at IS NULL, returns nil). Partial
// unique index frees the name for reuse (spec R-SK-002, SCN-2.3).
func (r *SkillRepo) SoftDelete(ctx context.Context, db domain.SQLExecutor, id int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE `+skillTableName+`
             SET deleted_at = now()
             WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("postgres.SkillRepo.SoftDelete: %w", err)
	}
	return nil
}

// LockAndLoad issues SELECT … FOR UPDATE on the row (live OR
// soft-deleted — service checks DeletedAt to choose 404 vs 410).
// Caller MUST be inside a transaction. Single concurrency gate for
// Update, Restore, and SoftDelete.
func (r *SkillRepo) LockAndLoad(ctx context.Context, db domain.SQLExecutor, id int64) (*domain.Skill, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+skillColumnList+` FROM `+skillTableName+`
             WHERE id = $1
             FOR UPDATE`,
		id,
	)
	s, err := scanSkill(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: skillTableName}
		}
		return nil, fmt.Errorf("postgres.SkillRepo.LockAndLoad: %w", err)
	}
	return s, nil
}

// MaxRevisionNumber returns COALESCE(MAX(revision_number), 0) for
// the given skill_id. Service uses this under FOR UPDATE to assign
// the next revision number.
func (r *SkillRepo) MaxRevisionNumber(ctx context.Context, db domain.SQLExecutor, skillID int64) (int, error) {
	var n sql.NullInt64
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(revision_number), 0) FROM skill_revision WHERE skill_id = $1`,
		skillID,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres.SkillRepo.MaxRevisionNumber: %w", err)
	}
	return int(n.Int64), nil
}

// ---------------------------------------------------------------------------
// Scanners.
// ---------------------------------------------------------------------------

type skillRowScanner interface {
	Scan(dest ...any) error
}

// scanSkill reads one skill row (Skill entity). NULL handling for
// DeletedAt distinguishes "never deleted" from "soft-deleted".
func scanSkill(s skillRowScanner) (*domain.Skill, error) {
	var sk domain.Skill
	var deletedAt sql.NullTime
	if err := s.Scan(
		&sk.ID, &sk.Name, &sk.Description, &sk.Body,
		&deletedAt, &sk.CreatedAt, &sk.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		sk.DeletedAt = &t
	}
	return &sk, nil
}

// scanSkillDetail reads a SkillDetail (Skill + CurrentRevision).
// Row shape is scanSkill plus an extra COALESCE column at the end.
func scanSkillDetail(s skillRowScanner) (*domain.SkillDetail, error) {
	var d domain.SkillDetail
	var deletedAt sql.NullTime
	if err := s.Scan(
		&d.ID, &d.Name, &d.Description, &d.Body,
		&deletedAt, &d.CreatedAt, &d.UpdatedAt,
		&d.CurrentRevision,
	); err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		d.DeletedAt = &t
	}
	return &d, nil
}
