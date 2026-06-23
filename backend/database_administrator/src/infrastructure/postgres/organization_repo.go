// Package postgres contains the Postgres adapters for the
// organization hexagonal slice. This file is the ONLY file in the
// repo that imports jackc/pgx (and pgconn) for organization data
// access, mirroring the existing
// src/migration/postgres/driver.go rule. Per design §6, the
// application layer depends only on domain; the handler depends
// only on application + domain. The pgx import surface stays
// restricted to this package.
//
// The adapter implements domain.OrganizationRepository. It
// translates:
//
//   - unique-violation pgx errors (SQLSTATE 23505) into
//     *domain.ConflictError so the handler can map to HTTP 409
//     without importing pgx itself.
//   - "no rows in result set" pgx errors into
//     *domain.NotFoundError so the handler can map to HTTP 404.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// PostgresOrgRepo is the pgx-backed adapter that satisfies
// domain.OrganizationRepository. It uses the stdlib database/sql
// interface so the rest of the system (handler, service) does not
// need to know about pgx directly.
type PostgresOrgRepo struct {
	db *sql.DB
}

// NewPostgresOrgRepo constructs a PostgresOrgRepo. The caller
// passes in an already-opened *sql.DB (typically produced by
// migration/postgres.Open in the composition root). The
// constructor is cheap and side-effect-free; the first
// repo.Insert / repo.SelectByID / repo.SelectAll is what
// actually touches the database.
func NewPostgresOrgRepo(db *sql.DB) *PostgresOrgRepo {
	return &PostgresOrgRepo{db: db}
}

// Compile-time check that PostgresOrgRepo satisfies the
// domain.OrganizationRepository port. Mirrors the pattern in
// migration/runner.go: if the public surface drifts the build
// breaks here, not in a downstream consumer.
var _ domain.OrganizationRepository = (*PostgresOrgRepo)(nil)

// ---------------------------------------------------------------------------
// Locked query constants. Magic strings become compile-time
// constants so a future contributor who renames a column catches
// it in the IDE before the test suite runs.
// ---------------------------------------------------------------------------

const (
	pgUniqueViolation = "23505"

	orgTableName          = "organization"
	orgColumnList         = "id, shortname, full_name, identification, is_active, email, phone, created_at, updated_at"
	orgInsertColumnList   = "shortname, full_name, identification, is_active, email, phone"
	orgInsertValuesCount  = 6
	orgUniqueConstraintID = "organization_identification_key"
)

// Insert persists a new organization. The DB sets id, is_active,
// created_at, and updated_at from column defaults; the adapter
// uses RETURNING to read the populated row back into the
// domain.Organization pointer.
//
// A unique-violation on identification (Postgres SQLSTATE 23505)
// is translated into *domain.ConflictError so the handler does
// not need to import pgx.
func (r *PostgresOrgRepo) Insert(ctx context.Context, o *domain.Organization) (*domain.Organization, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO organization (`+orgInsertColumnList+`)
         VALUES ($1, $2, $3, TRUE, $4, $5)
         RETURNING `+orgColumnList,
		nullableString(o.ShortName),
		o.FullName,
		o.Identification,
		nullableString(o.Email),
		nullableString(o.Phone),
	)
	out, err := scanOrganization(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
			return nil, &domain.ConflictError{Cause: err}
		}
		return nil, fmt.Errorf("postgres.OrganizationRepo.Insert: %w", err)
	}
	return out, nil
}

// SelectAll returns every row in (created_at ASC, id ASC) order.
// The deterministic order is required by spec §3.2 so the
// snapshot tests for the list endpoint are stable. Returns an
// empty slice (NOT nil) when zero rows exist.
func (r *PostgresOrgRepo) SelectAll(ctx context.Context) ([]domain.Organization, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+orgColumnList+` FROM `+orgTableName+` ORDER BY created_at ASC, id ASC`,
	)
	if err != nil {
		return nil, fmt.Errorf("postgres.OrganizationRepo.SelectAll: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := []domain.Organization{}
	for rows.Next() {
		org, err := scanOrganization(rows)
		if err != nil {
			return nil, fmt.Errorf("postgres.OrganizationRepo.SelectAll: scan: %w", err)
		}
		out = append(out, *org)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres.OrganizationRepo.SelectAll: rows: %w", err)
	}
	return out, nil
}

// SelectByID returns the row with the given id, or
// *domain.NotFoundError when no row matches. SQLSTATE-mapped
// errors are returned wrapped in *domain.NotFoundError so the
// handler does not need to import pgx.
func (r *PostgresOrgRepo) SelectByID(ctx context.Context, id int64) (*domain.Organization, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+orgColumnList+` FROM `+orgTableName+` WHERE id = $1`,
		id,
	)
	org, err := scanOrganization(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.NotFoundError{Resource: orgTableName}
		}
		return nil, fmt.Errorf("postgres.OrganizationRepo.SelectByID: %w", err)
	}
	return org, nil
}

// rowScanner is the small interface satisfied by both *sql.Row
// (from QueryRowContext) and *sql.Rows (from QueryContext). The
// scan helper accepts either so Insert (single row via RETURNING)
// and SelectAll/SelectByID (row at a time from rows.Next) share
// the same scan logic.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanOrganization reads one row from a SELECT (or RETURNING)
// into a fresh *domain.Organization. The column order MUST match
// orgColumnList.
func scanOrganization(row rowScanner) (*domain.Organization, error) {
	var (
		o          domain.Organization
		shortname  sql.NullString
		email      sql.NullString
		phone      sql.NullString
	)
	if err := row.Scan(
		&o.ID,
		&shortname,
		&o.FullName,
		&o.Identification,
		&o.IsActive,
		&email,
		&phone,
		&o.CreatedAt,
		&o.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if shortname.Valid {
		s := shortname.String
		o.ShortName = &s
	}
	if email.Valid {
		e := email.String
		o.Email = &e
	}
	if phone.Valid {
		p := phone.String
		o.Phone = &p
	}
	return &o, nil
}

// nullableString returns the value of a *string as an
// `any`-compatible type. When s is nil it returns untyped nil
// so the database/sql driver writes SQL NULL instead of the
// empty string.
func nullableString(s *string) any {
	if s == nil {
		return nil
	}
	return *s
}
