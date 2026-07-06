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
//   - FK-violation pgx errors on AddLinkedRepo (workspace_id does
//     not exist) into *domain.InternalError so the handler can map
//     to HTTP 500. The wrapping preserves the underlying cause for
//     slog logging.
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

// NewWorkspaceRepo constructs a WorkspaceRepo. The caller passes in
// an already-opened *sql.DB (typically produced by
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
	pgForeignKeyViolation = "23503"

	workspaceTableName              = "workspace"
	workspaceColumnList             = "id, organization_id, owner_user_id, name, primary_repo_github_id, primary_repo_full_name, primary_repo_owner, primary_repo_name, created_at, updated_at, deleted_at"
	workspaceInsertColumnList       = "organization_id, owner_user_id, name, primary_repo_github_id, primary_repo_full_name, primary_repo_owner, primary_repo_name"
	workspaceInsertValuesCount      = 7
	workspaceLiveUniqueIndex        = "workspace_org_name_live_key"
	workspaceRepositoryTableName    = "workspace_repository"
	workspaceRepositoryColumnList   = "id, workspace_id, github_id, github_full_name, github_owner, github_name, added_at"
	workspaceRepositoryInsertCols   = "workspace_id, github_id, github_full_name, github_owner, github_name"
	workspaceRepositoryInsertValues = 5
	workspaceRepoUniqueConstraint   = "workspace_repository_workspace_github_key"
)

// ---------------------------------------------------------------------------
// Workspace CRUD
// ---------------------------------------------------------------------------

// Insert persists a new workspace. The DB sets id, created_at,
// updated_at, and deleted_at (NULL by default) from column defaults;
// the adapter uses RETURNING to read the populated row back into the
// domain.Workspace pointer.
//
// A unique-violation on (organization_id, name) WHERE deleted_at IS
// NULL (the partial unique index workspace_org_name_live_key) is
// translated to *domain.ConflictError so the handler does not need to
// import pgx.
func (r *WorkspaceRepo) Insert(ctx context.Context, w *domain.Workspace) (*domain.Workspace, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO `+workspaceTableName+` (`+workspaceInsertColumnList+`)
         VALUES ($1, $2, $3, $4, $5, $6, $7)
         RETURNING `+workspaceColumnList,
		w.OrganizationID,
		nullableInt64(w.OwnerUserID),
		w.Name,
		w.PrimaryRepoGitHubID,
		w.PrimaryRepoFullName,
		w.PrimaryRepoOwner,
		w.PrimaryRepoName,
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
// given id, or *domain.NotFoundError when no live row matches. The
// partial unique index + the explicit `WHERE deleted_at IS NULL`
// clause together ensure soft-deleted rows are invisible.
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
// in the given organization, ordered by (created_at DESC, id DESC)
// for stable pagination. The deterministic order ensures the
// snapshot tests for the list endpoint are stable across calls.
// The limit caps the slice at N — the caller picks N; the repo
// MUST NOT silently truncate beyond the cap. Returns an empty
// slice (NOT nil) when zero rows exist.
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
// bumps updated_at. The unique violation on
// (organization_id, name) WHERE deleted_at IS NULL is translated
// to *domain.ConflictError. Returns *NotFoundError if no live row
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

// SoftDelete sets `deleted_at = now()` on the workspace row AND
// hard-deletes every row in `workspace_repository WHERE
// workspace_id = id`. Both SQL statements run inside one
// transaction so a crash mid-delete cannot leave linked repos
// dangling. Returns *NotFoundError if no live row matches.
//
// After SoftDelete, the partial unique index
// workspace_org_name_live_key allows a new workspace to be
// inserted with the same name in the same organization (per
// R-WS-005 / S-WS-043).
func (r *WorkspaceRepo) SoftDelete(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres.WorkspaceRepo.SoftDelete: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx,
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

	if _, err := tx.ExecContext(ctx,
		`DELETE FROM `+workspaceRepositoryTableName+` WHERE workspace_id = $1`,
		id,
	); err != nil {
		return fmt.Errorf("postgres.WorkspaceRepo.SoftDelete: cascade-delete: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres.WorkspaceRepo.SoftDelete: commit: %w", err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Linked repository CRUD
// ---------------------------------------------------------------------------

// AddLinkedRepo inserts a new row into workspace_repository. The
// DB sets id and added_at from column defaults; the adapter uses
// RETURNING to read the populated row back.
//
// The unique violation on (workspace_id, github_id) (constraint
// workspace_repository_workspace_github_key) is translated to
// *domain.ConflictError.
//
// FK violations (workspace_id does not exist) are translated to
// *domain.InternalError wrapping the cause — the handler maps to
// HTTP 500. The application layer is expected to verify the
// workspace exists + is live before calling AddLinkedRepo; this
// translation is a defense-in-depth fallback.
func (r *WorkspaceRepo) AddLinkedRepo(ctx context.Context, repo *domain.LinkedRepository) (*domain.LinkedRepository, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO `+workspaceRepositoryTableName+` (`+workspaceRepositoryInsertCols+`)
         VALUES ($1, $2, $3, $4, $5)
         RETURNING `+workspaceRepositoryColumnList,
		repo.WorkspaceID,
		repo.GitHubID,
		repo.FullName,
		repo.Owner,
		repo.Name,
	)
	out, err := scanLinkedRepository(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case pgUniqueViolation:
				return nil, &domain.ConflictError{Cause: err}
			case pgForeignKeyViolation:
				return nil, &domain.InternalError{Cause: err}
			}
		}
		return nil, fmt.Errorf("postgres.WorkspaceRepo.AddLinkedRepo: %w", err)
	}
	return out, nil
}

// RemoveLinkedRepo hard-deletes a single linked repo row. Returns
// *domain.NotFoundError if 0 rows are affected (the (workspaceID,
// repoID) pair does not exist). The function does NOT check the
// workspace's soft-delete state — the caller (application layer)
// has already filtered for live workspaces by the time this
// method is invoked.
func (r *WorkspaceRepo) RemoveLinkedRepo(ctx context.Context, workspaceID, repoID int64) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM `+workspaceRepositoryTableName+`
         WHERE id = $1 AND workspace_id = $2`,
		repoID, workspaceID,
	)
	if err != nil {
		return fmt.Errorf("postgres.WorkspaceRepo.RemoveLinkedRepo: %w", err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("postgres.WorkspaceRepo.RemoveLinkedRepo: rows-affected: %w", err)
	}
	if rowsAffected == 0 {
		return &domain.NotFoundError{Resource: workspaceRepositoryTableName}
	}
	return nil
}

// SelectLinkedRepos returns every linked repo for the given
// workspace, ordered by (added_at ASC, id ASC) — chronological.
// The deterministic order keeps the snapshot tests for the
// detail endpoint stable. Returns an empty slice (NOT nil) when
// zero rows exist.
func (r *WorkspaceRepo) SelectLinkedRepos(ctx context.Context, workspaceID int64) ([]domain.LinkedRepository, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+workspaceRepositoryColumnList+` FROM `+workspaceRepositoryTableName+`
         WHERE workspace_id = $1
         ORDER BY added_at ASC, id ASC`,
		workspaceID,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.WorkspaceRepo.SelectLinkedRepos: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []domain.LinkedRepository{}
	for rows.Next() {
		repo, err := scanLinkedRepository(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres.WorkspaceRepo.SelectLinkedRepos: scan: %w", err)
		}
		out = append(out, *repo)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.WorkspaceRepo.SelectLinkedRepos: rows: %w", err)
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// Scan helpers (mirror organization_repo.go)
// ---------------------------------------------------------------------------

// scanWorkspace reads one row from a SELECT (or RETURNING) into a
// fresh *domain.Workspace. The column order MUST match
// workspaceColumnList.
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
		&w.PrimaryRepoGitHubID,
		&w.PrimaryRepoFullName,
		&w.PrimaryRepoOwner,
		&w.PrimaryRepoName,
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

// scanLinkedRepository reads one row from a SELECT (or RETURNING)
// into a fresh *domain.LinkedRepository. The column order MUST
// match workspaceRepositoryColumnList.
func scanLinkedRepository(row rowScanner) (*domain.LinkedRepository, error) {
	var r domain.LinkedRepository
	if err := row.Scan(
		&r.ID,
		&r.WorkspaceID,
		&r.GitHubID,
		&r.FullName,
		&r.Owner,
		&r.Name,
		&r.AddedAt,
	); err != nil {
		return nil, err
	}
	return &r, nil
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
