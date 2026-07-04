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

// identityUpsertUser selects an identity.user row by case-insensitive
// email; if missing, INSERTs a new row and returns the new id. The
// second statement is split out so the SELECT and the conditional
// INSERT can run as a single round-trip (CTE) without the application
// layer orchestrating two queries. We use a WITH ... SELECT / INSERT
// pattern that mirrors what porsager/postgres would have done on the
// frontend side in PR #29 (rejected architecture).
const identityUpsertUser = `
WITH existing AS (
    SELECT id FROM identity.user WHERE lower(email) = lower($1) LIMIT 1
), inserted AS (
    INSERT INTO identity.user (email, name, image_url)
    SELECT $1, $2, $3
    WHERE NOT EXISTS (SELECT 1 FROM existing)
    RETURNING id
)
SELECT id FROM existing
UNION ALL
SELECT id FROM inserted
LIMIT 1`

// identityInsertAccount is the second half of the Upsert path. It
// INSERTs an identity.account row using the resolved user_id, with
// ON CONFLICT (provider, provider_account_id) DO NOTHING so a
// returning user does not error out on the unique constraint. The
// resolved identity.account row is then JOINed to identity.user so the
// returned *domain.Identity has all six fields populated.
const identityInsertAccount = `
INSERT INTO identity.account (user_id, provider, provider_account_id)
VALUES ($1, $2, $3)
ON CONFLICT (provider, provider_account_id) DO NOTHING
RETURNING user_id`

// Upsert persists an OAuth identity event into identity.user +
// identity.account. Behaviour (mirrors PR #29's sign-in-callback.ts
// semantics — auto-link on email match, idempotent on (provider,
// provider_account_id)):
//
//   1. CTE upserts identity.user keyed by case-insensitive email.
//   2. INSERTs identity.account with ON CONFLICT DO NOTHING.
//
// Returns the resolved *domain.Identity (new or reused) with the
// account row's provider + provider_account_id populated.
//
// Errors:
//   - Any DB-level error is wrapped with a `postgres.` prefix and
//     returned unchanged. The handler maps to 5xx.
//
// OAuth tokens (access_token, refresh_token, expires_at, token_type,
// scope) are intentionally NOT persisted: the identity.account schema
// does not store tokens in this slice (cachicamas-identity-signin-
// callback ADR 0003 §"Forward notes"). The handler accepts them in
// the wire body for cross-tooling compatibility but discards them
// after HMAC verification.
func (r *IdentityRepo) Upsert(ctx context.Context, ev domain.IdentityEvent) (*domain.Identity, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("postgres.IdentityRepo.Upsert: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var userID int64
	if err := tx.QueryRowContext(ctx, identityUpsertUser, ev.Email, ev.Name, ev.ImageURL).Scan(&userID); err != nil {
		return nil, fmt.Errorf("postgres.IdentityRepo.Upsert: user upsert: %w", err)
	}
	if _, err := tx.ExecContext(ctx, identityInsertAccount, userID, ev.Provider, ev.ProviderAccountID); err != nil {
		return nil, fmt.Errorf("postgres.IdentityRepo.Upsert: account insert: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("postgres.IdentityRepo.Upsert: commit: %w", err)
	}

	// Re-read so the returned *Identity reflects the persisted row
	// (name, image_url, provider, provider_account_id are all locked
	// fields the verifier middleware depends on).
	out, err := r.LookupByProviderAccountID(ctx, ev.Provider, ev.ProviderAccountID)
	if err != nil {
		// The row was just inserted; a miss here means a concurrent
		// DELETE on identity.account (rare). Wrap and return.
		return nil, fmt.Errorf("postgres.IdentityRepo.Upsert: post-insert lookup: %w", err)
	}
	return out, nil
}
