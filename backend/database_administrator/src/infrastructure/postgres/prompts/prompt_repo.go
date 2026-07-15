// Package prompts contains the Postgres adapters for the prompts
// hexagonal slice (PR2 of 2026-07-15-prompt-storage-table). This
// file is the ONLY file in the repo that imports jackc/pgx (and
// pgconn) for prompt data access, mirroring the existing
// src/migration/postgres/driver.go rule and workspace_repo.go.
//
// The adapter implements domain.PromptRepository. It translates:
//   - unique-violation pgx errors (SQLSTATE 23505) into
//     *domain.ConflictError so the handler can map to HTTP 409
//     without importing pgx itself.
//   - "no rows in result set" pgx errors into
//     *domain.NotFoundError so the handler can map to HTTP 404.
//
// The methods accept a `domain.sqlExecutor` (which *sql.DB and *sql.Tx
// both satisfy) so the application layer can compose multi-statement
// operations (create + first revision, restore + parent update) in a
// single transaction without leaking pgx types into the domain.
package prompts

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

// PromptRepo is the pgx-backed adapter that satisfies
// domain.PromptRepository. It uses the stdlib database/sql interface
// so the rest of the system (handler, service) does not need to know
// about pgx directly.
type PromptRepo struct {
	// db is held only for tests / one-shot operations that don't run
	// inside a service-supplied transaction. Production methods
	// always take an explicit sqlExecutor parameter so the service
	// can decide whether to use the pool or a transaction.
	db *sql.DB
}

// NewPromptRepo constructs a PromptRepo. The caller passes in an
// already-opened *sql.DB (typically produced by migration/postgres.Open
// in the composition root). The constructor is cheap and
// side-effect-free.
func NewPromptRepo(db *sql.DB) *PromptRepo {
	return &PromptRepo{db: db}
}

// Compile-time check that PromptRepo satisfies the
// domain.PromptRepository port. Mirrors the pattern in workspace_repo.
var _ domain.PromptRepository = (*PromptRepo)(nil)

// ---------------------------------------------------------------------------
// Locked query constants. Magic strings become compile-time constants
// so a future contributor who renames a column catches it in the IDE
// before the test suite runs.
// ---------------------------------------------------------------------------

const (
	promptTableName        = "prompt"
	promptColumnList       = "id, description, slug, body, deleted_at, created_at, updated_at"
	promptInsertColumnList = "description, slug, body"
	promptInsertValuesCount = 3
	promptLiveUniqueIndex  = "prompt_slug_active_uidx"
)

// ---------------------------------------------------------------------------
// Prompt CRUD
// ---------------------------------------------------------------------------

// Insert persists a new prompt row. The DB sets id, created_at,
// updated_at, and deleted_at (NULL by default) from column defaults;
// the adapter uses RETURNING to read the populated row back into the
// domain.Prompt pointer.
//
// A unique-violation on the partial UNIQUE index
// `prompt_slug_active_uidx` (slug WHERE deleted_at IS NULL) is
// translated to *domain.ConflictError so the handler does not need to
// import pgx.
func (r *PromptRepo) Insert(ctx context.Context, db domain.SQLExecutor, p *domain.Prompt) error {
	row := db.QueryRowContext(ctx,
		`INSERT INTO `+promptTableName+` (`+promptInsertColumnList+`)
             VALUES ($1, $2, $3)
             RETURNING `+promptColumnList,
		p.Description, p.Slug, p.Body,
	)
	out, err := scanPrompt(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return &domain.ConflictError{Cause: err}
		}
		return fmt.Errorf("postgres.PromptRepo.Insert: %w", err)
	}
	*p = *out
	return nil
}

// SelectBySlug returns the LIVE (deleted_at IS NULL) row with the
// given slug, or *domain.NotFoundError when no live row matches. The
// partial UNIQUE index + the explicit `WHERE deleted_at IS NULL`
// clause together ensure soft-deleted rows are invisible (spec S-PR-6,
// S-PR-9, S-PR-24).
func (r *PromptRepo) SelectBySlug(ctx context.Context, db domain.SQLExecutor, slug string) (*domain.Prompt, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+promptColumnList+` FROM `+promptTableName+`
             WHERE slug = $1 AND deleted_at IS NULL`,
		slug,
	)
	p, err := scanPrompt(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: promptTableName}
		}
		return nil, fmt.Errorf("postgres.PromptRepo.SelectBySlug: %w", err)
	}
	return p, nil
}

// SelectBySlugAny returns the row with the given slug regardless
// of its deleted_at state. Used by the service to distinguish
// "slug never existed" (returns *NotFoundError) from "slug exists
// but is soft-deleted" (returns the row with DeletedAt != nil; the
// service then maps to *GoneError → 410 per spec S-PR-8 / S-PR-5).
func (r *PromptRepo) SelectBySlugAny(ctx context.Context, db domain.SQLExecutor, slug string) (*domain.Prompt, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+promptColumnList+` FROM `+promptTableName+`
             WHERE slug = $1`,
		slug,
	)
	p, err := scanPrompt(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: promptTableName}
		}
		return nil, fmt.Errorf("postgres.PromptRepo.SelectBySlugAny: %w", err)
	}
	return p, nil
}

// SelectByID returns the LIVE row with the given id, or
// *domain.NotFoundError when no live row matches. The caller is
// typically the service layer after LockAndLoad.
func (r *PromptRepo) SelectByID(ctx context.Context, db domain.SQLExecutor, id int64) (*domain.Prompt, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+promptColumnList+` FROM `+promptTableName+`
             WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	p, err := scanPrompt(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: promptTableName}
		}
		return nil, fmt.Errorf("postgres.PromptRepo.SelectByID: %w", err)
	}
	return p, nil
}

// SelectList returns every LIVE prompt ordered by (updated_at DESC,
// id DESC) for stable pagination. The limit is clamped by the service
// to MaxListLimit before reaching the repo. Returns an empty slice
// (NOT nil) when zero rows exist.
func (r *PromptRepo) SelectList(ctx context.Context, db domain.SQLExecutor, limit, offset int) ([]*domain.Prompt, error) {
	rows, err := db.QueryContext(ctx,
		`SELECT `+promptColumnList+` FROM `+promptTableName+`
             WHERE deleted_at IS NULL
             ORDER BY updated_at DESC, id DESC
             LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.PromptRepo.SelectList: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []*domain.Prompt{}
	for rows.Next() {
		p, err := scanPrompt(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres.PromptRepo.SelectList: scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.PromptRepo.SelectList: rows: %w", err)
	}
	return out, nil
}

// UpdateBody updates body, description, and updated_at on a LIVE
// prompt. Used by the service layer after acquiring the FOR UPDATE
// row lock so the version assignment is monotonic (spec INV-4,
// S-PR-21). The DB clock owns updated_at.
func (r *PromptRepo) UpdateBody(ctx context.Context, db domain.SQLExecutor, id int64, body, description string) error {
	res, err := db.ExecContext(ctx,
		`UPDATE `+promptTableName+`
             SET body = $1, description = $2, updated_at = now()
             WHERE id = $3 AND deleted_at IS NULL`,
		body, description, id,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			// A concurrent soft-delete + recreate with the same
			// slug can collide on the partial unique index even
			// though UPDATE doesn't change slug. Translate so the
			// handler can map to 409.
			return &domain.ConflictError{Cause: err}
		}
		return fmt.Errorf("postgres.PromptRepo.UpdateBody: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return &domain.NotFoundError{Resource: promptTableName}
	}
	return nil
}

// SoftDelete sets deleted_at = now() on the prompt row. Idempotent:
// calling it twice on the same row succeeds both times (the second
// call finds zero rows matching `deleted_at IS NULL`, returns nil).
// The partial unique index frees the slug for reuse after delete
// (spec S-PR-7, EC2).
func (r *PromptRepo) SoftDelete(ctx context.Context, db domain.SQLExecutor, id int64) error {
	_, err := db.ExecContext(ctx,
		`UPDATE `+promptTableName+`
             SET deleted_at = now()
             WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("postgres.PromptRepo.SoftDelete: %w", err)
	}
	return nil
}

// LockAndLoad issues `SELECT … FOR UPDATE` on the prompt row and
// returns the row (live OR soft-deleted — the service checks
// DeletedAt to decide whether to return PROMPT_DELETED). The caller
// MUST be inside a transaction; this is the single concurrency gate
// for Update, Restore, and SoftDelete. Returns
// *domain.NotFoundError when no row matches the id.
//
// The reason LockAndLoad returns soft-deleted rows: the service
// needs to know whether the row exists at all. A row that is
// soft-deleted must produce PROMPT_DELETED (410), not PROMPT_NOT_FOUND
// (404). Hiding it in the repo would force the service to issue a
// second SELECT after a 404, which is racy.
func (r *PromptRepo) LockAndLoad(ctx context.Context, db domain.SQLExecutor, id int64) (*domain.Prompt, error) {
	row := db.QueryRowContext(ctx,
		`SELECT `+promptColumnList+` FROM `+promptTableName+`
             WHERE id = $1
             FOR UPDATE`,
		id,
	)
	p, err := scanPrompt(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: promptTableName}
		}
		return nil, fmt.Errorf("postgres.PromptRepo.LockAndLoad: %w", err)
	}
	return p, nil
}

// MaxRevisionNumber returns COALESCE(MAX(revision_number), 0) for
// the given prompt_id. Used by the service under FOR UPDATE to assign
// the next revision number (spec INV-4). Returns 0 for a prompt with
// no revisions yet (the DB invariant says every prompt has at least
// revision 1, so this only happens before the first Insert).
func (r *PromptRepo) MaxRevisionNumber(ctx context.Context, db domain.SQLExecutor, promptID int64) (int, error) {
	var n sql.NullInt64
	if err := db.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(revision_number), 0) FROM prompt_revision WHERE prompt_id = $1`,
		promptID,
	).Scan(&n); err != nil {
		return 0, fmt.Errorf("postgres.PromptRepo.MaxRevisionNumber: %w", err)
	}
	return int(n.Int64), nil
}

// ---------------------------------------------------------------------------
// Scanner. Mirrors scanWorkspace in workspace_repo.go. Accepts both
// *sql.Row (single-row queries) and *sql.Rows (list queries) via
// the rowScanner interface.
// ---------------------------------------------------------------------------

type promptRowScanner interface {
	Scan(dest ...any) error
}

// scanPrompt reads one prompt row from a promptRowScanner into a fresh
// *domain.Prompt. The NULL handling for DeletedAt distinguishes
// "never deleted" (NULL) from "deleted_at was set to a real time"
// (non-NULL); the service reads `p.DeletedAt != nil` to detect the
// soft-deleted state.
func scanPrompt(s promptRowScanner) (*domain.Prompt, error) {
	var p domain.Prompt
	var deletedAt sql.NullTime
	if err := s.Scan(
		&p.ID,
		&p.Description,
		&p.Slug,
		&p.Body,
		&deletedAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		p.DeletedAt = &t
	}
	return &p, nil
}