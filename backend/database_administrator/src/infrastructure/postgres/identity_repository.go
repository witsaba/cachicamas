// Package postgres contains the Postgres adapters for the identity
// hexagonal slice (cachicamas-github-login). This file is the ONLY
// file in the repo that imports jackc/pgx for identity data access,
// mirroring the existing organization_repo.go rule. Per design §4,
// the application layer depends only on domain; the handler depends
// only on application + domain. The pgx import surface stays
// restricted to this package.
//
// The adapter implements domain.IdentityRepository. It translates:
//
//   - "no rows in result set" errors (from stdlib database/sql)
//     into *domain.IdentityNotFoundError so the handler can map to
//     HTTP 404 without importing pgx itself.
//
// The implementation reads identity.user + the most-recent
// identity.account row in a single SQL statement (LEFT JOIN LATERAL)
// so the verifier middleware (PR-3) can populate
// `c.Set("identity", *Identity)` without a second roundtrip.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// IdentityRepo is the pgx-backed adapter that satisfies
// domain.IdentityRepository. It uses the stdlib database/sql
// interface so the rest of the system (handler, service) does not
// need to know about pgx directly.
type IdentityRepo struct {
	db *sql.DB
}

// NewIdentityRepo constructs an IdentityRepo. The caller passes in
// an already-opened *sql.DB (typically produced by
// migration/postgres.Open in the composition root, when PR-3
// lands). The constructor is cheap and side-effect-free; the first
// repo.LookupByEmail is what actually touches the database.
func NewIdentityRepo(db *sql.DB) *IdentityRepo {
	return &IdentityRepo{db: db}
}

// Compile-time check that IdentityRepo satisfies the
// domain.IdentityRepository port. Mirrors the pattern in
// organization_repo.go: if the public surface drifts the build
// breaks here, not in a downstream consumer.
var _ domain.IdentityRepository = (*IdentityRepo)(nil)

// identitySelect joins user + the user's most-recent account row.
// We don't yet have multi-account semantics on this slice (each
// user signs in via one provider, but future provider additions
// will return the FIRST account by id ASC). For now the LIMIT 1
// is sufficient.
const identitySelect = `SELECT u.id, u.email, COALESCE(u.name, ''), COALESCE(u.image_url, ''), a.provider, a.provider_account_id
        FROM identity.user u
        LEFT JOIN identity.account a ON a.user_id = u.id
        WHERE lower(u.email) = lower($1)
        ORDER BY u.id ASC, a.id ASC
        LIMIT 1`

// LookupByEmail resolves an identity.user row by email (case-
// insensitive via Postgres CITEXT) and returns the matching
// identity.account snapshot (provider + provider_account_id) for
// the user. The result is a single domain.Identity value ready to
// hand to the HTTP handler.
//
// Errors:
//   - *domain.IdentityNotFoundError when no row matches.
//   - Other errors are wrapped with a postgres. prefix and
//     returned unchanged (the HTTP handler maps them to 5xx).
func (r *IdentityRepo) LookupByEmail(ctx context.Context, email string) (*domain.Identity, error) {
	row := r.db.QueryRowContext(ctx, identitySelect, email)
	out := domain.Identity{}
	err := row.Scan(
		&out.ID,
		&out.Email,
		&out.Name,
		&out.ImageURL,
		&out.Provider,
		&out.ProviderAccountID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.IdentityNotFoundError{Email: email}
		}
		return nil, fmt.Errorf("postgres.IdentityRepo.LookupByEmail: %w", err)
	}
	return &out, nil
}

// identitySelectByProviderAccountID joins identity.account + identity.user
// on the (provider, provider_account_id) UNIQUE constraint
// (account_provider_provider_account_id_key). This is the canonical
// lookup path for the JWE verifier middleware (PR-3 IdentityFromCookie)
// — stable across email changes and forward-compatible for multi-
// provider support.
//
// The ORDER BY a.id ASC + LIMIT 1 handles the (rare) case where the
// same (provider, account_id) appears more than once. In practice the
// UNIQUE constraint prevents duplicates, but a defensive LIMIT keeps
// the API deterministic even under migration edge cases.
const identitySelectByProviderAccountID = `SELECT u.id, u.email, COALESCE(u.name, ''), COALESCE(u.image_url, ''), a.provider, a.provider_account_id
        FROM identity.account a
        JOIN identity.user u ON u.id = a.user_id
        WHERE a.provider = $1 AND a.provider_account_id = $2
        ORDER BY a.id ASC
        LIMIT 1`

// LookupByProviderAccountID resolves an identity.user row by the
// OAuth (provider, provider_account_id) pair. This is the canonical
// lookup path for the JWE verifier middleware (PR-3) — it's
// stable across email changes (users can change their GitHub
// primary email without losing identity) and forward-compatible
// for multi-provider support (same user can be linked to GitHub
// + Google + etc. via different account rows).
//
// Errors:
//   - *domain.IdentityNotFoundError when no row matches.
//   - Other errors are wrapped with a postgres. prefix and
//     returned unchanged.
func (r *IdentityRepo) LookupByProviderAccountID(ctx context.Context, provider, providerAccountID string) (*domain.Identity, error) {
	row := r.db.QueryRowContext(ctx, identitySelectByProviderAccountID, provider, providerAccountID)
	out := domain.Identity{}
	err := row.Scan(
		&out.ID,
		&out.Email,
		&out.Name,
		&out.ImageURL,
		&out.Provider,
		&out.ProviderAccountID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, &domain.IdentityNotFoundError{
				Email: fmt.Sprintf("%s/%s", provider, providerAccountID),
			}
		}
		return nil, fmt.Errorf("postgres.IdentityRepo.LookupByProviderAccountID: %w", err)
	}
	return &out, nil
}
