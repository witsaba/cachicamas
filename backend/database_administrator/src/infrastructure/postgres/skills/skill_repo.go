// Package skills contains the Postgres adapters for the skills
// hexagonal slice (PR1b of cachicamas-skills-foundational). This file
// is the ONLY file in the repo that imports jackc/pgx (and pgconn)
// for skill data access, mirroring the existing prompts adapter at
// infrastructure/postgres/prompts/prompt_repo.go and the workspace
// adapter.
//
// The adapter implements domain.SkillRepository. It translates:
//   - unique-violation pgx errors (SQLSTATE 23505) into
//     *domain.ConflictError so the handler can map to HTTP 409
//     without importing pgx itself.
//   - "no rows in result set" pgx errors into
//     *domain.NotFoundError so the handler can map to HTTP 404.
//
// Methods accept a `domain.sqlExecutor` (which *sql.DB and *sql.Tx
// both satisfy) so the application layer can compose multi-statement
// operations (create + first revision, restore + parent update) in a
// single transaction without leaking pgx types into the domain.
package skills

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// pgUniqueViolation is the SQLSTATE returned by Postgres on a unique
// constraint violation. Defined locally so this sub-package does not
// depend on the parent postgres package's constants.
const pgUniqueViolation = "23505"

// SkillRepo is the pgx-backed adapter that satisfies
// domain.SkillRepository. It uses the stdlib database/sql interface
// so the rest of the system (handler, service) does not need to know
// about pgx directly.
type SkillRepo struct {
	// db is held only for tests / one-shot operations that don't run
	// inside a service-supplied transaction. Production methods always
	// take an explicit sqlExecutor parameter so the service can decide
	// whether to use the pool or a transaction.
	db *sql.DB
}

// NewSkillRepo constructs a SkillRepo. The caller passes in an
// already-opened *sql.DB (typically produced by migration/postgres.Open
// in the composition root). The constructor is cheap and
// side-effect-free.
func NewSkillRepo(db *sql.DB) *SkillRepo {
	return &SkillRepo{db: db}
}

// Compile-time check that SkillRepo satisfies the
// domain.SkillRepository port.
var _ domain.SkillRepository = (*SkillRepo)(nil)

// ---------------------------------------------------------------------------
// Locked query constants. Magic strings become compile-time constants
// so a future contributor who renames a column catches it in the IDE
// before the test suite runs.
// ---------------------------------------------------------------------------

const (
	skillTableName         = "skill"
	skillColumnList        = "id, name, description, body, deleted_at, created_at, updated_at"
	skillInsertColumnList  = "name, description, body"
	skillInsertValuesCount = 3
	skillLiveUniqueIndex   = "skill_slug_active_uidx"
)

// ---------------------------------------------------------------------------
// Skill CRUD
// ---------------------------------------------------------------------------

// Insert persists a new skill row. The DB sets id, created_at,
// updated_at, and deleted_at (NULL by default) from column defaults;
// the adapter uses RETURNING to read the populated row back into the
// domain.Skill pointer.
//
// A unique-violation on the partial UNIQUE index
// `skill_slug_active_uidx` (name WHERE deleted_at IS NULL) is
// translated to *domain.ConflictError so the handler does not need to
// import pgx.
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

// ---------------------------------------------------------------------------
// Method stubs — implemented in subsequent RED/GREEN cycles.
//
// Each stub returns zero-value errors/nils so the compile-time
// interface check (var _ domain.SkillRepository = (*SkillRepo)(nil))
// succeeds for task 2.1 (Insert). The stub bodies are REPLACED in
// later tasks; the signatures stay stable.
// ---------------------------------------------------------------------------

// SelectBySlug returns the LIVE (deleted_at IS NULL) row with the
// given name, or *domain.NotFoundError when no live row matches. The
// partial UNIQUE index + the explicit `WHERE deleted_at IS NULL`
// clause together ensure soft-deleted rows are invisible (spec
// R-SK-002 / SCN-2.1).
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

// SelectBySlugAny returns the row with the given name regardless of
// its deleted_at state. Used by the service to distinguish
// "name never existed" (returns *NotFoundError) from "name exists
// but is soft-deleted" (returns the row with DeletedAt != nil; the
// service then maps to *SkillGoneError → 410 per spec R-SK-005).
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
func (r *SkillRepo) SelectByID(context.Context, domain.SQLExecutor, int64) (*domain.Skill, error) {
	return nil, nil
}
func (r *SkillRepo) SelectByIDWithCurrentRevision(context.Context, domain.SQLExecutor, int64) (*domain.SkillDetail, error) {
	return nil, nil
}
func (r *SkillRepo) List(context.Context, domain.SQLExecutor, int) ([]*domain.Skill, error) {
	return nil, nil
}
func (r *SkillRepo) ListWithCurrentRevision(context.Context, domain.SQLExecutor, int) ([]*domain.SkillListItem, error) {
	return nil, nil
}
func (r *SkillRepo) UpdateBody(context.Context, domain.SQLExecutor, int64, string, string) error {
	return nil
}
func (r *SkillRepo) SoftDelete(context.Context, domain.SQLExecutor, int64) error { return nil }
func (r *SkillRepo) LockAndLoad(context.Context, domain.SQLExecutor, int64) (*domain.Skill, error) {
	return nil, nil
}
func (r *SkillRepo) MaxRevisionNumber(context.Context, domain.SQLExecutor, int64) (int, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Scanner. Mirrors scanPrompt in prompts/prompt_repo.go. Accepts both
// *sql.Row (single-row queries) and *sql.Rows (list queries) via the
// rowScanner interface.
// ---------------------------------------------------------------------------

type skillRowScanner interface {
	Scan(dest ...any) error
}

// scanSkill reads one skill row from a skillRowScanner into a fresh
// *domain.Skill. The NULL handling for DeletedAt distinguishes
// "never deleted" (NULL) from "deleted_at was set to a real time"
// (non-NULL); the service reads `s.DeletedAt != nil` to detect the
// soft-deleted state.
func scanSkill(s skillRowScanner) (*domain.Skill, error) {
	var sk domain.Skill
	var deletedAt sql.NullTime
	if err := s.Scan(
		&sk.ID,
		&sk.Name,
		&sk.Description,
		&sk.Body,
		&deletedAt,
		&sk.CreatedAt,
		&sk.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		sk.DeletedAt = &t
	}
	return &sk, nil
}
