// Package postgres — token_fetcher.go implements the pgx-backed
// TokenFetcher adapter that the auth middleware uses to load the
// authenticated user's GitHub OAuth access_token from identity.account.
//
// PR1c-ii closes the review-risk H-1 finding from PR1c-i: production
// TokenFetcher MUST use parameterized SQL to prevent SQL injection
// (provider and provider_account_id are user-controlled strings sourced
// from the JWE cookie claims).
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

// TokenFetcher is the pgx-backed adapter that satisfies the
// httpiface.TokenFetcher contract.
type TokenFetcher struct {
	db *sql.DB
}

// NewTokenFetcher constructs a TokenFetcher. The caller passes in the
// already-opened *sql.DB.
func NewTokenFetcher(db *sql.DB) *TokenFetcher {
	return &TokenFetcher{db: db}
}

// tokenFetcherSelect uses parameterized placeholders. NO string
// interpolation. This is the single source of truth for the
// access_token read query.
const tokenFetcherSelect = `SELECT access_token
        FROM identity.account
        WHERE provider = $1
          AND provider_account_id = $2
        LIMIT 1`

// AccessTokenForIdentity returns the user's access_token for the
// given (provider, provider_account_id).
//
// Returns:
//   - (token, nil) when the row exists with a non-NULL access_token.
//   - ("", &httpiface.TokenNotFoundError) when no row matches OR the
//     row exists but access_token is NULL (signed in before PR1a).
//     The caller (auth middleware) treats this as "sign in again" and
//     logs "signed in before PR1a".
//   - ("", wrappedErr) on any other DB error. The auth middleware logs
//     + continues with empty token (never escalates to 5xx — see
//     PR1c-i auth_middleware_token.go).
func (t *TokenFetcher) AccessTokenForIdentity(ctx context.Context, provider, providerAccountID string) (string, error) {
	if provider == "" || providerAccountID == "" {
		return "", &httpiface.TokenNotFoundError{Reason: "empty provider or account id"}
	}
	var token sql.NullString
	err := t.db.QueryRowContext(ctx, tokenFetcherSelect, provider, providerAccountID).Scan(&token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", &httpiface.TokenNotFoundError{Reason: "no identity.account row for (provider, account_id)"}
		}
		return "", fmt.Errorf("postgres.TokenFetcher.AccessTokenForIdentity: %w", err)
	}
	if !token.Valid {
		return "", &httpiface.TokenNotFoundError{Reason: "access_token column is NULL (signed in before PR1a)"}
	}
	return token.String, nil
}

// Compile-time check that TokenFetcher satisfies httpiface.TokenFetcher.
var _ httpiface.TokenFetcher = (*TokenFetcher)(nil)