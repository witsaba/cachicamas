// Package httpiface — auth_middleware_token_test.go covers PR1c-i's
// auth_middleware additions: load the user's access_token from
// identity.account and inject it into the request context via
// tokenctx.WithGitHubToken.
//
// Tasks covered (sdd/2026-07-06-workspaces/tasks §PR1c-i):
//
//	T-WS-1Ci-012 RED    — middleware loads token, injects into context
//	T-WS-1Ci-013 GREEN  — implementation
//	T-WS-1Ci-014 TRIA   — NULL token in DB + DB load failure fallback
//
// These tests are in a SEPARATE file from the existing
// auth_middleware_test.go (cachicamas-github-login PR-3) to keep
// the JWE-cookie verifier coverage separate from the token-loading
// coverage; the two files belong to different SDD changes.
package httpiface

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http/tokenctx"
)

// stubTokenFetcher is a tiny in-memory TokenFetcher for the token-loading
// tests.
type stubTokenFetcher struct {
	accessToken *string
	dbErr       error
}

func (s *stubTokenFetcher) AccessTokenForIdentity(ctx context.Context, provider, accountID string) (string, error) {
	if s.dbErr != nil {
		return "", s.dbErr
	}
	if s.accessToken == nil {
		return "", nil
	}
	return *s.accessToken, nil
}

var _ TokenFetcher = (*stubTokenFetcher)(nil)

// extractTokenHandler is the terminal handler that captures whatever
// tokenctx.GitHubTokenFromContext returns. We use it to assert that
// the middleware actually injected the token.
func extractTokenHandler(t *testing.T) (echo.HandlerFunc, *string) {
	t.Helper()
	var captured string
	h := func(c *echo.Context) error {
		tok, _ := tokenctx.GitHubTokenFromContext(c.Request().Context())
		captured = tok
		return c.String(http.StatusOK, "ok")
	}
	return h, &captured
}

// noopLogger returns a discarded logger so the tests don't pollute output.
func noopLogger() *slog.Logger {
	return slog.Default()
}

// TestAuthMiddleware_LoadsTokenFromIdentityAccount is RED for T-WS-1Ci-012.
func TestAuthMiddleware_LoadsTokenFromIdentityAccount(t *testing.T) {
	tok := "gho_real_test_token_xxx"
	fetcher := &stubTokenFetcher{accessToken: &tok}

	mw := loadGitHubTokenForIdentity(fetcher, noopLogger())
	terminal, captured := extractTokenHandler(t)

	c := echo.New().NewContext(
		httptest.NewRequest(http.MethodGet, "/x", nil),
		httptest.NewRecorder(),
	)
	c.Set(IdentityContextKey, &domain.Identity{ID: 42, Provider: "github", ProviderAccountID: "abc"})

	if err := mw(terminal)(c); err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if *captured != tok {
		t.Errorf("expected token %q in downstream context, got %q", tok, *captured)
	}
}

func TestAuthMiddleware_NilTokenPassesEmptyString(t *testing.T) {
	// TRIANGULATE (a) for T-WS-1Ci-014.
	fetcher := &stubTokenFetcher{accessToken: nil}
	mw := loadGitHubTokenForIdentity(fetcher, noopLogger())
	terminal, captured := extractTokenHandler(t)

	c := echo.New().NewContext(
		httptest.NewRequest(http.MethodGet, "/x", nil),
		httptest.NewRecorder(),
	)
	c.Set(IdentityContextKey, &domain.Identity{ID: 42})

	if err := mw(terminal)(c); err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if *captured != "" {
		t.Errorf("expected empty token for nil DB value, got %q", *captured)
	}
	out, ok := tokenctx.GitHubTokenFromContext(c.Request().Context())
	if !ok {
		t.Errorf("expected ok=true even when token is empty")
	}
	if out != "" {
		t.Errorf("expected empty string, got %q", out)
	}
}

func TestAuthMiddleware_DBErrorLoggedAndProceedsWithEmptyToken(t *testing.T) {
	// TRIANGULATE (b) for T-WS-1Ci-014.
	boom := errors.New("connection refused")
	fetcher := &stubTokenFetcher{dbErr: boom}
	mw := loadGitHubTokenForIdentity(fetcher, noopLogger())
	terminal, captured := extractTokenHandler(t)

	c := echo.New().NewContext(
		httptest.NewRequest(http.MethodGet, "/x", nil),
		httptest.NewRecorder(),
	)
	c.Set(IdentityContextKey, &domain.Identity{ID: 42})

	if err := mw(terminal)(c); err != nil {
		t.Fatalf("middleware error (must continue, never block): %v", err)
	}
	if *captured != "" {
		t.Errorf("expected empty token when DB load fails, got %q", *captured)
	}
}

func TestAuthMiddleware_NoIdentityOnContext_PassesThroughEmptyToken(t *testing.T) {
	// Defense-in-depth.
	fetcher := &stubTokenFetcher{accessToken: nil}
	mw := loadGitHubTokenForIdentity(fetcher, noopLogger())
	terminal, captured := extractTokenHandler(t)

	c := echo.New().NewContext(
		httptest.NewRequest(http.MethodGet, "/x", nil),
		httptest.NewRecorder(),
	)
	// No c.Set(IdentityContextKey) — the request is anonymous.

	if err := mw(terminal)(c); err != nil {
		t.Fatalf("middleware error: %v", err)
	}
	if *captured != "" {
		t.Errorf("expected empty token when no identity, got %q", *captured)
	}
}
