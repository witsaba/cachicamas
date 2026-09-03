// Package postgres contains the Postgres adapters for the auth
// hexagonal slice (cachicamas-google-auth-bootstrap PR-2). This file
// is the ONLY file in the repo that imports jackc/pgx for auth.users
// data access, mirroring the existing identity_repository.go rule.
// Per design §2.2, the application layer depends only on domain; the
// handler depends only on application + domain. The pgx import
// surface stays restricted to this package.
//
// The adapter implements auth.UserRepository. It translates:
//
//   - "no rows in result set" errors (from stdlib database/sql)
//     into (nil, nil) so the bootstrap service can branch on
//     existence without importing sql.ErrNoRows.
//   - Unique-violation pgx errors (SQLSTATE 23505) into the existing
//     *domain.ConflictError so the handler maps to HTTP 409 without
//     importing pgx itself.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// UserRepo is the pgx-backed adapter that satisfies
// auth.UserRepository. It uses the stdlib database/sql interface so
// the rest of the system (handler, service) does not need to know
// about pgx directly. The DB handle is captured at construction time
// and reused across calls; the first repo call is what actually
// touches the database.
type UserRepo struct {
	db *sql.DB
}

// NewUserRepo constructs a UserRepo. The caller passes in an
// already-opened *sql.DB (typically produced by migration/postgres.Open
// in the composition root). The constructor is cheap and
// side-effect-free.
func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

// Compile-time check that UserRepo satisfies auth.UserRepository. If
// the public surface drifts the build breaks here, not in a
// downstream consumer.
var _ auth.UserRepository = (*UserRepo)(nil)

// userColumnList is the canonical column order shared by every SELECT
// against auth.users. Centralised so a future schema migration that
// adds a column fails exactly one scan helper + one test, not every
// read site. updated_at is included so callers can detect drift.
const userColumnList = "id, email, email_verified, google_sub, name, picture_url, status, last_login_at, created_at, updated_at"

// scanUser reads one row from a SELECT (or RETURNING) into a fresh
// *auth.User. The column order MUST match userColumnList.
func scanUser(row interface {
	Scan(dest ...any) error
}) (*auth.User, error) {
	var (
		u            auth.User
		emailVerif   bool
		name         sql.NullString
		pictureURL   sql.NullString
		lastLoginAt  sql.NullTime
		googleSub    sql.NullString
	)
	if err := row.Scan(
		&u.ID,
		&u.Email,
		&emailVerif,
		&googleSub,
		&name,
		&pictureURL,
		&u.Status,
		&lastLoginAt,
		&u.CreatedAt,
		&u.UpdatedAt,
	); err != nil {
		return nil, err
	}
	u.EmailVerified = emailVerif
	if googleSub.Valid {
		u.GoogleSub = googleSub.String
	}
	if name.Valid {
		u.Name = name.String
	}
	if pictureURL.Valid {
		u.PictureURL = pictureURL.String
	}
	if lastLoginAt.Valid {
		t := lastLoginAt.Time
		u.LastLoginAt = &t
	}
	// Validate the status coming back from the DB; if a future
	// migration adds an unknown value we surface the corrupt row
	// instead of silently accepting it.
	if _, err := auth.NewUserStatus(string(u.Status)); err != nil {
		return nil, fmt.Errorf("postgres.UserRepo: scan: invalid status from DB: %w", err)
	}
	return &u, nil
}

// FindByGoogleSub resolves an auth.users row by google_sub (the
// provider-stable secondary key per design D4 / R-BE-002). Returns
// (nil, nil) when no live row matches; only DB-level failures yield
// a non-nil error.
func (r *UserRepo) FindByGoogleSub(ctx context.Context, q auth.Querier, googleSub string) (*auth.User, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+userColumnList+` FROM auth.users
         WHERE google_sub = $1 AND deleted_at IS NULL
         LIMIT 1`,
		googleSub,
	)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres.UserRepo.FindByGoogleSub: %w", err)
	}
	return u, nil
}

// FindByID resolves an auth.users row by primary key. Soft-deleted
// rows (deleted_at IS NOT NULL) are invisible.
func (r *UserRepo) FindByID(ctx context.Context, q auth.Querier, id int64) (*auth.User, error) {
	row := q.QueryRowContext(ctx,
		`SELECT `+userColumnList+` FROM auth.users
         WHERE id = $1 AND deleted_at IS NULL
         LIMIT 1`,
		id,
	)
	u, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("postgres.UserRepo.FindByID: %w", err)
	}
	return u, nil
}

// InsertRegistered persists a freshly-built User with the `registered`
// status. Returns the persisted row (including the DB-generated id +
// timestamps from DEFAULT now()). Idempotent on google_sub via ON
// CONFLICT DO NOTHING + RETURNING + re-read; a duplicate insert
// returns the existing row, not an error (spec R-BOOTSTRAP-1).
//
// Implementation note: the two-step pattern (INSERT ... ON CONFLICT
// DO NOTHING RETURNING; SELECT ... WHERE google_sub = $1) handles the
// case where a parallel bootstrap caller raced us to the INSERT — the
// RETURNING yields no row in that case, the SELECT returns the row
// the winner inserted, and both callers see the same user_id.
func (r *UserRepo) InsertRegistered(ctx context.Context, q auth.Querier, u *auth.User) (*auth.User, error) {
	if u == nil {
		return nil, errors.New("postgres.UserRepo.InsertRegistered: nil user")
	}
	if err := u.Validate(); err != nil {
		return nil, fmt.Errorf("postgres.UserRepo.InsertRegistered: validate: %w", err)
	}
	row := q.QueryRowContext(ctx,
		`INSERT INTO auth.users (email, google_sub, status)
         VALUES ($1, $2, 'registered')
         ON CONFLICT (google_sub) DO NOTHING
         RETURNING `+userColumnList,
		u.Email, u.GoogleSub,
	)
	inserted, err := scanUser(row)
	if err == nil {
		return inserted, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("postgres.UserRepo.InsertRegistered: insert: %w", err)
	}
	// ON CONFLICT path: another caller raced us to the INSERT.
	// Re-read the row by google_sub so the caller gets a fully
	// populated *auth.User (including DB-generated timestamps).
	return r.FindByGoogleSub(ctx, q, u.GoogleSub)
}

// UpdateLoginFields advances the mutable login fields on an existing
// user. The column list explicitly EXCLUDES created_at — this is the
// application-layer half of S-DB-002. A DB trigger
// (migration 2026XXXXXX_user_immutable_created_at.sql) provides the
// second half at the database level for any non-Go caller.
//
// Fields updated: email (re-lower-cased), email_verified, name,
// picture_url, last_login_at (= now()), updated_at (= now()).
func (r *UserRepo) UpdateLoginFields(ctx context.Context, q auth.Querier, id int64, u *auth.User) error {
	if u == nil {
		return errors.New("postgres.UserRepo.UpdateLoginFields: nil user")
	}
	now := time.Now().UTC()
	_, err := q.ExecContext(ctx,
		`UPDATE auth.users
            SET email = $2,
                email_verified = $3,
                name = $4,
                picture_url = $5,
                last_login_at = $6,
                updated_at = $6
          WHERE id = $1 AND deleted_at IS NULL`,
		id,
		u.Email,
		u.EmailVerified,
		nullableStringPtr(u.Name),
		nullableStringPtr(u.PictureURL),
		now,
	)
	if err != nil {
		return fmt.Errorf("postgres.UserRepo.UpdateLoginFields: %w", err)
	}
	return nil
}

// PromoteToActive transitions a user from `registered` to `active`.
// The WHERE clause carries `status = 'registered'` so a re-call on an
// already-active user is a no-op (idempotent), matching spec R-BE-002
// (registered → active transition happens at organization creation
//).
//
// The returned (rowsAffected, error) pair lets the bootstrap service
// distinguish "user was promoted" from "user was already active" so
// it can decide whether to create the organization.
func (r *UserRepo) PromoteToActive(ctx context.Context, q auth.Querier, id int64) error {
	_, err := q.ExecContext(ctx,
		`UPDATE auth.users
            SET status = 'active',
                updated_at = now()
          WHERE id = $1 AND status = 'registered' AND deleted_at IS NULL`,
		id,
	)
	if err != nil {
		return fmt.Errorf("postgres.UserRepo.PromoteToActive: %w", err)
	}
	return nil
}

// nullableStringPtr returns untyped nil for the empty string so the
// database/sql driver writes SQL NULL instead of an empty value.
// Used for the optional name / picture_url columns on auth.users.
func nullableStringPtr(s string) any {
	if s == "" {
		return nil
	}
	return s
}