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
// SelectByID returns the row with the given id regardless of its
// deleted_at state, or *domain.NotFoundError when no row matches.
// The caller is typically the service layer after LockAndLoad or
// after a write (for the re-read that produces the response).
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
// currentRevisionSubquery is the locked SQL fragment for the JOIN
// against skill_revision. Defined as a constant so the SQL structure
// is identical between SelectByIDWithCurrentRevision and
// ListWithCurrentRevision (the design's anti-drift gate from
// ADR-SK-008 + obs #1959 #2 — kills the "v{undefined}" prompt bug).
const currentRevisionSubquery = `(SELECT skill_id, MAX(revision_number) AS current_revision FROM skill_revision GROUP BY skill_id)`

// SelectByIDWithCurrentRevision returns the Skill + its current
// revision number via a single LEFT JOIN (no N+1, no per-row query).
// A skill with no revisions yields CurrentRevision=0 (COALESCE).
// *domain.NotFoundError when the id doesn't match any row (live or
// soft-deleted).
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

// List returns active (deleted_at IS NULL) skills ordered by
// (updated_at DESC, id DESC) for stable pagination. The limit is
// clamped by the service to MaxListLimit before reaching the repo.
// Returns an empty slice (NOT nil) when zero rows exist.
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

// ListWithCurrentRevision returns active (deleted_at IS NULL) skills
// joined with their current revision number. ONE SQL statement for
// the entire list (anti-drift gate ADR-SK-008). Skills with no
// revisions emit current_revision=0 (COALESCE).
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
// UpdateBody updates body, description, and updated_at on a LIVE
// skill. Used by the service layer after acquiring the FOR UPDATE
// row lock so the version assignment is monotonic (spec INV-4 +
// S-SK-021/022). The DB clock owns updated_at.
//
// A unique-violation on the partial unique index (e.g. concurrent
// soft-delete + recreate with the same name) translates to
// *domain.ConflictError so the handler can map to 409.
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

// SoftDelete sets deleted_at = now() on the skill row. Idempotent:
// calling it twice on the same row succeeds both times (the second
// call finds zero rows matching `deleted_at IS NULL`, returns nil).
// The partial unique index frees the name for reuse after delete
// (spec R-SK-002, SCN-2.3).
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

// LockAndLoad issues `SELECT … FOR UPDATE` on the skill row and
// returns the row (live OR soft-deleted — the service checks
// DeletedAt to decide whether to return *SkillGoneError). The caller
// MUST be inside a transaction; this is the single concurrency gate
// for Update, Restore, and SoftDelete. Returns
// *domain.NotFoundError when no row matches the id.
//
// Returns soft-deleted rows: the service needs to know whether the
// row exists at all. A row that is soft-deleted must produce 410
// (skill_deleted), not 404 (not_found). Hiding it in the repo would
// force the service to issue a second SELECT after a 404, which is
// racy.
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
// the given skill_id. Used by the service under FOR UPDATE to assign
// the next revision number (spec INV-4). Returns 0 for a skill with
// no revisions yet.
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

// scanSkillDetail reads one row from a SELECT … LEFT JOIN with
// current_revision. The row shape is the same as scanSkill plus an
// extra COALESCE column at the end. We scan into a SkillDetail
// (which embeds Skill) and populate CurrentRevision separately.
func scanSkillDetail(s skillRowScanner) (*domain.SkillDetail, error) {
	var d domain.SkillDetail
	var deletedAt sql.NullTime
	if err := s.Scan(
		&d.ID,
		&d.Name,
		&d.Description,
		&d.Body,
		&deletedAt,
		&d.CreatedAt,
		&d.UpdatedAt,
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
