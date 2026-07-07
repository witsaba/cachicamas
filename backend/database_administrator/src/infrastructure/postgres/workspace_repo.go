// Package postgres contains the Postgres adapters for the
// workspace hexagonal slice (PR1b-ii.a). This file is the ONLY
// file in the repo that imports jackc/pgx (and pgconn) for
// workspace data access, mirroring the existing
// src/migration/postgres/driver.go rule. Per design §6, the
// application layer depends only on domain; the handler depends
// only on application + domain. The pgx import surface stays
// restricted to this package.
//
// The adapter implements domain.WorkspaceRepository. It
// translates:
//   - unique-violation pgx errors (SQLSTATE 23505) into
//     *domain.ConflictError so the handler can map to HTTP 409
//     without importing pgx itself.
//   - "no rows in result set" pgx errors into
//     *domain.NotFoundError so the handler can map to HTTP 404.
//
// 2026-07-08-workspaces-simplify changelog:
//   - Dropped AddLinkedRepo / RemoveLinkedRepo / SelectLinkedRepos
//     (the workspace_repository table no longer exists).
//   - Dropped scanLinkedRepository and the workspace_repository
//     constants.
//   - SoftDelete no longer cascades into workspace_repository (the
//     table no longer exists).
//   - Renamed primary_repo_* columns to repo_* in the workspace
//     table (the SQL columns match the renames in
//     domain.Workspace.RepoGitHubID etc.).
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// WorkspaceRepo is the pgx-backed adapter that satisfies
// domain.WorkspaceRepository. It uses the stdlib database/sql
// interface so the rest of the system (handler, service) does not
// need to know about pgx directly.
type WorkspaceRepo struct {
	db *sql.DB
}

// NewWorkspaceRepo constructs a WorkspaceRepo. The caller passes
// in an already-opened *sql.DB (typically produced by
// migration/postgres.Open in the composition root). The
// constructor is cheap and side-effect-free; the first
// repo.Insert / repo.SelectByID / repo.SelectAllByOrg is what
// actually touches the database.
func NewWorkspaceRepo(db *sql.DB) *WorkspaceRepo {
	return &WorkspaceRepo{db: db}
}

// Compile-time check that WorkspaceRepo satisfies the
// domain.WorkspaceRepository port. Mirrors the pattern in
// migration/runner.go: if the public surface drifts the build
// breaks here, not in a downstream consumer.
var _ domain.WorkspaceRepository = (*WorkspaceRepo)(nil)

// ---------------------------------------------------------------------------
// Locked query constants. Magic strings become compile-time
// constants so a future contributor who renames a column catches
// it in the IDE before the test suite runs.
// ---------------------------------------------------------------------------

const (
	workspaceTableName        = "workspace"
	workspaceColumnList       = "id, organization_id, owner_user_id, name, repo_github_id, repo_full_name, repo_owner, repo_name, created_at, updated_at, deleted_at"
	workspaceInsertColumnList = "organization_id, owner_user_id, name, repo_github_id, repo_full_name, repo_owner, repo_name"
	workspaceInsertValuesCount = 7
	workspaceLiveUniqueIndex  = "workspace_org_name_live_key"
)

// ---------------------------------------------------------------------------
// Workspace CRUD
// ---------------------------------------------------------------------------

// Insert persists a new workspace. The DB sets id, created_at,
// updated_at, and deleted_at (NULL by default) from column
// defaults; the adapter uses RETURNING to read the populated row
// back into the domain.Workspace pointer.
//
// A unique-violation on (organization_id, name) WHERE deleted_at
// IS NULL (the partial unique index workspace_org_name_live_key)
// is translated to *domain.ConflictError so the handler does not
// need to import pgx.
//
// 2026-07-08-workspaces-simplify: the workspace row's
// primary_repo_* columns are now repo_*. The $N placeholders
// below must match the new column order.
func (r *WorkspaceRepo) Insert(ctx context.Context, w *domain.Workspace) (*domain.Workspace, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO `+workspaceTableName+` (`+workspaceInsertColumnList+`)
             VALUES ($1, $2, $3, $4, $5, $6, $7)
             RETURNING `+workspaceColumnList,
		w.OrganizationID,
		nullableInt64(w.OwnerUserID),
		w.Name,
		w.RepoGitHubID,
		w.RepoFullName,
		w.RepoOwner,
		w.RepoName,
	)
	out, err := scanWorkspace(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, &domain.ConflictError{Cause: err}
		}
		return nil, fmt.Errorf("postgres.WorkspaceRepo.Insert: %w", err)
	}
	return out, nil
}

// SelectByID returns the live (deleted_at IS NULL) row with the
// given id, or *domain.NotFoundError when no live row matches.
// The partial unique index + the explicit `WHERE deleted_at IS
// NULL` clause together ensure soft-deleted rows are invisible.
func (r *WorkspaceRepo) SelectByID(ctx context.Context, id int64) (*domain.Workspace, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+workspaceColumnList+` FROM `+workspaceTableName+`
             WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	w, err := scanWorkspace(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: workspaceTableName}
		}
		return nil, fmt.Errorf("postgres.WorkspaceRepo.SelectByID: %w", err)
	}
	return w, nil
}

// SelectAllByOrg returns every live (deleted_at IS NULL) workspace
// in the given organization, ordered by (created_at DESC, id
// DESC) for stable pagination. The deterministic order ensures
// the snapshot tests for the list endpoint are stable across
// calls. The limit caps the slice at N — the caller picks N; the
// repo MUST NOT silently truncate beyond the cap. Returns an
// empty slice (NOT nil) when zero rows exist.
func (r *WorkspaceRepo) SelectAllByOrg(ctx context.Context, orgID int64, limit int) ([]domain.Workspace, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+workspaceColumnList+` FROM `+workspaceTableName+`
             WHERE organization_id = $1 AND deleted_at IS NULL
             ORDER BY created_at DESC, id DESC
             LIMIT $2`,
		orgID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.WorkspaceRepo.SelectAllByOrg: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []domain.Workspace{}
	for rows.Next() {
		w, err := scanWorkspace(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres.WorkspaceRepo.SelectAllByOrg: scan: %w", err)
		}
		out = append(out, *w)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.WorkspaceRepo.SelectAllByOrg: rows: %w", err)
	}
	return out, nil
}

// UpdateName renames a live (deleted_at IS NULL) workspace and
// bumps updated_at. The unique violation on (organization_id,
// name) WHERE deleted_at IS NULL is translated to
// *domain.ConflictError. Returns *NotFoundError if no live row
// matches (the partial index hides soft-deleted rows; renaming a
// soft-deleted row is impossible without un-deleting it first,
// which is out of scope for v1).
func (r *WorkspaceRepo) UpdateName(ctx context.Context, id int64, name string) (*domain.Workspace, error) {
	row := r.db.QueryRowContext(ctx,
		`UPDATE `+workspaceTableName+`
             SET name = $1, updated_at = now()
             WHERE id = $2 AND deleted_at IS NULL
             RETURNING `+workspaceColumnList,
		name, id,
	)
	w, err := scanWorkspace(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: workspaceTableName}
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, &domain.ConflictError{Cause: err}
		}
		return nil, fmt.Errorf("postgres.WorkspaceRepo.UpdateName: %w", err)
	}
	return w, nil
}

// SoftDelete sets `deleted_at = now()` on the workspace row.
// Returns *NotFoundError if no live row matches.
//
// After SoftDelete, the partial unique index
// workspace_org_name_live_key allows a new workspace to be
// inserted with the same name in the same organization (per
// R-WS-005 / S-WS-043).
//
// 2026-07-08-workspaces-simplify: no longer cascades into
// workspace_repository (the table no longer exists). Single SQL
// statement; no transaction needed.
func (r *WorkspaceRepo) SoftDelete(ctx context.Context, id int64) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE `+workspaceTableName+`
             SET deleted_at = now()
             WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("postgres.WorkspaceRepo.SoftDelete: update: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres.WorkspaceRepo.SoftDelete: rows-affected: %w", err)
	}
	if rowsAffected == 0 {
		return &domain.NotFoundError{Resource: workspaceTableName}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Scan helpers (mirror organization_repo.go)
// ---------------------------------------------------------------------------

// scanWorkspace reads one row from a SELECT (or RETURNING) into
// a fresh *domain.Workspace. The column order MUST match
// workspaceColumnList (the SELECT above).
//
// 2026-07-08-workspaces-simplify: the primary_repo_* columns are
// now repo_*. The Scan targets (w.RepoGitHubID etc.) must match
// the SELECT order.
func scanWorkspace(row rowScanner) (*domain.Workspace, error) {
	var (
		w           domain.Workspace
		ownerUserID sql.NullInt64
		deletedAt   sql.NullTime
	)
	if err := row.Scan(
		&w.ID,
		&w.OrganizationID,
		&ownerUserID,
		&w.Name,
		&w.RepoGitHubID,
		&w.RepoFullName,
		&w.RepoOwner,
		&w.RepoName,
		&w.CreatedAt,
		&w.UpdatedAt,
		&deletedAt,
	); err != nil {
		return nil, err
	}
	if ownerUserID.Valid {
		v := ownerUserID.Int64
		w.OwnerUserID = &v
	}
	if deletedAt.Valid {
		t := deletedAt.Time
		w.DeletedAt = &t
	}
	return &w, nil
}

// nullableInt64 returns the value of a *int64 as an
// `any`-compatible type. When s is nil it returns untyped nil so
// the database/sql driver writes SQL NULL instead of 0. Mirrors
// the `nullableString` helper in organization_repo.go.
func nullableInt64(s *int64) any {
	if s == nil {
		return nil
	}
	return *s
}