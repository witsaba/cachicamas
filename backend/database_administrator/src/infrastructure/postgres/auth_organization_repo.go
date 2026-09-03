// Package postgres — auth_organization_repo.go is the pgx-backed
// adapter that satisfies auth.OrganizationRepository. The pgx import
// stays restricted to this package per the hexagonal rule (design
// §2.2): the application layer depends only on domain; the handler
// depends only on application + domain.
//
// The file is named with the `auth_` prefix to coexist with the
// legacy `organization_repo.go` (which serves the public.organization
// table). The two adapters target different tables (auth.organizations
// vs public.organization) and different domain ports; future refactors
// may consolidate them once the legacy public.organization slice is
// retired.
//
// The adapter translates:
//
//   - "no rows in result set" errors (from stdlib database/sql)
//     into (nil, nil) so the bootstrap service can branch on
//     existence without importing sql.ErrNoRows.
//   - Unique-violation pgx errors (SQLSTATE 23505) into
//     *auth.ConflictError so the handler maps to HTTP 409 without
//     importing pgx itself.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// AuthOrganizationRepo is the pgx-backed adapter that satisfies
// auth.OrganizationRepository for the auth.organizations table. It
// uses the stdlib database/sql interface so the rest of the system
// (handler, service) does not need to know about pgx directly.
type AuthOrganizationRepo struct {
	db *sql.DB
}

// NewAuthOrganizationRepo constructs an AuthOrganizationRepo. The
// caller passes in an already-opened *sql.DB (typically produced by
// migration/postgres.Open in the composition root). The constructor
// is cheap and side-effect-free.
func NewAuthOrganizationRepo(db *sql.DB) *AuthOrganizationRepo {
	return &AuthOrganizationRepo{db: db}
}

// Compile-time check that AuthOrganizationRepo satisfies
// auth.OrganizationRepository. If the public surface drifts the
// build breaks here, not in a downstream consumer.
var _ auth.OrganizationRepository = (*AuthOrganizationRepo)(nil)

// authOrgColumnList is the canonical column order shared by every
// SELECT against auth.organizations. Centralised so a future schema
// migration that adds a column fails exactly one scan helper + one
// test, not every read site. Prefixed to avoid collision with the
// legacy organization adapter's `orgColumnList` constant in this
// same package.
const authOrgColumnList = "id, owner_id, name, slug, created_at, updated_at"

// authPgUniqueViolation mirrors the existing pgUniqueViolation
// constant defined by the legacy organization adapter. Duplicated
// here (rather than shared) so the auth slice can evolve its error
// vocabulary without entangling with the legacy public.organization
// surface. Both resolve to Postgres SQLSTATE 23505.
const authPgUniqueViolation = "23505"

// authRowScanner is the small interface satisfied by both *sql.Row
// (from QueryRowContext) and *sql.Rows (from QueryContext). The
// scan helper accepts either so Create (single row via RETURNING)
// shares the same scan logic with future ListAll-style callers.
// Prefixed to avoid collision with the legacy `rowScanner` type
// defined in the same package by the public.organization adapter.
type authRowScanner interface {
	Scan(dest ...any) error
}

// scanAuthOrganizationRow reads one row from a SELECT (or RETURNING)
// into a fresh *auth.Organization. The column order MUST match
// authOrgColumnList.
func scanAuthOrganizationRow(row authRowScanner) (*auth.Organization, error) {
	var o auth.Organization
	if err := row.Scan(
		&o.ID,
		&o.OwnerID,
		&o.Name,
		&o.Slug,
		&o.CreatedAt,
		&o.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &o, nil
}

// FindByOwnerID resolves an auth.organizations row by owner_id (the
// 1:1 FK to auth.users per MVP tenancy, design D2 / R-DB-003).
// Returns (nil, nil) when no live row matches; only DB-level
// failures yield a non-nil error.
func (r *AuthOrganizationRepo) FindByOwnerID(ctx context.Context, q auth.Querier, ownerID int64) (*auth.Organization, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+authOrgColumnList+` FROM auth.organizations
         WHERE owner_id = $1 AND deleted_at IS NULL
         LIMIT 1`,
		ownerID,
	)
	o, err := scanAuthOrganizationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres.AuthOrganizationRepo.FindByOwnerID: %w", err)
	}
	return o, nil
}

// FindByID resolves an auth.organizations row by primary key.
// Soft-deleted rows (deleted_at IS NOT NULL) are invisible.
func (r *AuthOrganizationRepo) FindByID(ctx context.Context, q auth.Querier, id int64) (*auth.Organization, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+authOrgColumnList+` FROM auth.organizations
         WHERE id = $1 AND deleted_at IS NULL
         LIMIT 1`,
		id,
	)
	o, err := scanAuthOrganizationRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres.AuthOrganizationRepo.FindByID: %w", err)
	}
	return o, nil
}

// Create persists a new organization. The slug is generated by the SQL
// itself as 'pyme-' || owner_id::text (per design §1.2 step 4),
// which keeps the slug deterministic and avoids collisions in the
// MVP. The adapter accepts the caller-supplied name (already
// lowercased via DerivePymeNameFromEmail) and the owner_id; the
// adapter does NOT need the slug.
//
// Implementation note: $1 (owner_id) is referenced exactly once via
// a CTE so the pgx prepared-statement planner does not raise
// "inconsistent types deduced for parameter $1" (SQLSTATE 42P08).
// The previous form (`VALUES ($1, $2, 'pyme-' || $1::text)`) bound
// $1 in two different type contexts (bigint from the column +
// text from the cast) and broke under pgx/database/sql prepared
// statements. Surfaced by the strict-TDD integration test
// TestAuthOrganizationRepo_Create (PR-2 backend handlers).
//
// A unique-violation on slug (Postgres SQLSTATE 23505, raised by the
// partial unique index auth_organizations_slug_live_key) is
// translated into *auth.ConflictError so the handler does not need
// to import pgx. This is a near-impossible case in the MVP (slugs
// are derived from user_id) but the surface is locked.
func (r *AuthOrganizationRepo) Create(ctx context.Context, q auth.Querier, o *auth.Organization) (*auth.Organization, error) {
	if o == nil {
		return nil, errors.New("postgres.AuthOrganizationRepo.Create: nil organization")
	}
	if err := o.Validate(); err != nil {
		return nil, fmt.Errorf("postgres.AuthOrganizationRepo.Create: validate: %w", err)
	}
	row := q.QueryRowContext(ctx,
		`WITH params AS (SELECT $1::bigint AS owner_id, $2::text AS name)
         INSERT INTO auth.organizations (owner_id, name, slug)
         SELECT owner_id, name, 'pyme-' || owner_id::text FROM params
         RETURNING `+authOrgColumnList,
		o.OwnerID,
		o.Name,
	)
	created, err := scanAuthOrganizationRow(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == authPgUniqueViolation {
			return nil, &auth.ConflictError{Cause: err, Resource: "auth.organizations"}
		}
		return nil, fmt.Errorf("postgres.AuthOrganizationRepo.Create: %w", err)
	}
	return created, nil
}