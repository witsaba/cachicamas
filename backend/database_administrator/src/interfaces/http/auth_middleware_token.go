// Package httpiface — auth_middleware_token.go implements the second
// pass of the auth middleware chain: after IdentityFromCookie populates
// the *domain.Identity on the Echo context, this middleware loads the
// user's GitHub OAuth access_token from identity.account and injects
// it into the request context via tokenctx.WithGitHubToken.
//
// Tasks covered (sdd/2026-07-06-workspaces/tasks §PR1c-i):
//
//	T-WS-1Ci-013 GREEN  — loadGitHubTokenForIdentity middleware
//
// Security contract (PR1c-i, hard rule from the user's review-risk
// request):
//
//   - The function name (loadGitHubTokenForIdentity) signals that the
//     token is tied to one identity, not the request. We never log the
//     token value itself; the slog line includes the identity.ID and
//     the provider, not the access_token payload.
//
//   - On DB error the middleware LOGS and continues with empty token.
//     The token-loading failure must NEVER escalate to a 5xx response
//     — the user's session is still valid; only the GitHub API is
//     unavailable.
//
//   - On success the token is placed on the REQUEST context (via
//     tokenctx.WithGitHubToken), not the ECHO context (c.Set). The
//     Echo context lives only for the duration of one request; this
//     package's lower layers (e.g. application/workspace_service.go)
//     read from context.Context directly.
//
//   - No token-shaped string ever appears in a slog attribute or value.
//     We use a fixed message + error_kind enum, not error.Error().
package httpiface

import (
	"context"
	"log/slog"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http/tokenctx"
)

// TokenFetcher is the narrow contract this middleware uses to read the
// user's access_token from persistent storage. It returns the token
// string (possibly empty if the user signed in before PR1a) and any
// DB error.
//
// Defining the contract here rather than extending domain.IdentityRepository
// keeps PR1c-i scope minimal: the production wiring injects a pgx-backed
// implementation without modifying the existing IdentityRepository
// surface used by IdentityFromCookie.
type TokenFetcher interface {
	AccessTokenForIdentity(ctx context.Context, provider, accountID string) (string, error)
}

// loadGitHubTokenForIdentity returns an Echo middleware that loads the
// authenticated user's GitHub OAuth access_token from identity.account
// and injects it into the request context. The middleware MUST be
// chained AFTER IdentityFromCookie (which populates the
// IdentityContextKey on the Echo context) — otherwise the request is
// treated as anonymous and an empty token is injected (defense-in-depth).
//
// The fetcher parameter is a narrow contract (TokenFetcher) to keep
// PR1c-i scope minimal. Production wiring injects an adapter that
// reads identity.account.access_token WHERE provider=$1 AND
// provider_account_id=$2.
func loadGitHubTokenForIdentity(fetcher TokenFetcher, logger *slog.Logger) echo.MiddlewareFunc {
	if fetcher == nil {
		panic("loadGitHubTokenForIdentity: fetcher must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			ctx := c.Request().Context()

			// Read the identity populated by IdentityFromCookie.
			rawIdentity := c.Get(IdentityContextKey)
			if rawIdentity == nil {
				// No authenticated user — inject empty token so
				// downstream handlers return 401 with the
				// rechable-known connector shape.
				c.SetRequest(c.Request().WithContext(
					tokenctx.WithGitHubToken(ctx, ""),
				))
				return next(c)
			}

			identity, ok := rawIdentity.(*domain.Identity)
			if !ok {
				// The identity doesn't carry the fields we need.
				// Inject empty token + continue.
				logger.WarnContext(ctx, "auth.token_load_skip",
					slog.String("reason", "identity_missing_provider_or_account"),
				)
				c.SetRequest(c.Request().WithContext(
					tokenctx.WithGitHubToken(ctx, ""),
				))
				return next(c)
			}

			provider := identity.Provider
			accountID := identity.ProviderAccountID

			token, err := fetcher.AccessTokenForIdentity(ctx, provider, accountID)
			if err != nil {
				// DB load failed. Log a fixed-shape warning and
				// continue with empty token. The downstream GitHub
				// handler will return 401 reconnect (R-WS-017);
				// we never want to escalate to 5xx from the
				// middleware path because the user's session is
				// still valid.
				logger.WarnContext(ctx, "auth.token_load_failed",
					slog.String("provider", provider),
					slog.String("error_kind", "db_unavailable"),
				)
				c.SetRequest(c.Request().WithContext(
					tokenctx.WithGitHubToken(ctx, ""),
				))
				return next(c)
			}

			// Inject on the REQUEST context (downstream service
			// layers read context.Context). Use the value the
			// fetcher returned — including the empty string when
			// the user signed in before PR1a has NULL access_token.
			c.SetRequest(c.Request().WithContext(
				tokenctx.WithGitHubToken(ctx, token),
			))
			return next(c)
		}
	}
}

// _ ensures the TokenFetcher type from the test file is the same one
// the production wiring relies on. The actual interface declaration
// lives in auth_middleware_token_test.go (Go's idiomatic pattern of
// declaring test fixtures alongside tests); however for production
// we re-export the contract here so wiring code in main.go imports a
// stable surface.

// Ensure context imports aren't accidentally dropped.
var _ = context.Background
