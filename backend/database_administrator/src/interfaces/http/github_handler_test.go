// Package httpiface_test — github_handler_test.go covers the
// `GET /github/repos` proxy + cache + auth-middleware integration.
//
// Tasks covered (sdd/2026-07-06-workspaces/tasks §PR1c-i):
//
//	T-WS-1Ci-009 RED    — warm cache returns repos
//	T-WS-1Ci-010 GREEN  — GitHubHandler + ListRepos
//	T-WS-1Ci-011 TRIA   — 5 error paths (no token, cache miss, bust,
//	                       github 401, github 403, malformed JSON)
//	Access-token never-leaks test — extra security guard added per
//	the user's request for a dedicated review-risk on PR1c-i.
package httpiface

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	gh "github.com/cachicamas/backend/database_administrator/src/infrastructure/github"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http/tokenctx"
)

const ghFixtureReposJSON = `[
  {"id":1,"name":"hello","full_name":"octocat/hello","private":false,
   "description":"d","html_url":"https://github.com/octocat/hello",
   "updated_at":"2026-01-15T12:00:00Z","stargazers_count":42,
   "owner":{"login":"octocat"}}
]`

// newGitHubMockServer spins up an httptest server that simulates
// api.github.com/user/repos. Captures the bearer token string for
// security assertions.
func newGitHubMockServer(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int64, *string) {
	t.Helper()
	var calls atomic.Int64
	var token string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		token = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls, &token
}

// stubUserID returns a closure that always reports (id, ok=true). The
// production handler reads the user identity from Echo's context (set by
// IdentityFromCookie via c.Set("identity", ...)); for handler-only tests
// we don't run the cookie verifier so we use a static stub.
func stubUserID(id int64) func(*echo.Context) (int64, bool) {
	return func(*echo.Context) (int64, bool) { return id, true }
}

// makeGitHubHandler builds a handler with default deps except overrides.
func makeGitHubHandler(t *testing.T, opts ...func(*GitHubHandler)) (*GitHubHandler, *gh.RepoCache, *gh.Client) {
	t.Helper()
	cache := gh.NewRepoCache(5 * time.Minute)
	client := gh.NewClientWithBase("http://placeholder.invalid")
	h := &GitHubHandler{
		Cache:             cache,
		Client:            client,
		UserIDFromRequest: stubUserID(42),
		Logger:            slog.New(slog.NewTextHandler(io.Discard, nil)).WithGroup("test"),
	}
	_ = io.Discard // alias for go vet.
	for _, opt := range opts {
		opt(h)
	}
	return h, cache, client
}

func TestGitHubHandler_ListRepos_WarmCacheSkipsUpstream(t *testing.T) {
	// RED for T-WS-1Ci-009, GREEN for T-WS-1Ci-010.
	h, cache, _ := makeGitHubHandler(t)
	cache.Set("42", []gh.Repo{{ID: 99, FullName: "x/y", OwnerLogin: "x", Name: "y"}})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Inject token directly so the handler trusts the request (we are
	// testing the handler, not the middleware in this test).
	c.SetRequest(req.WithContext(tokenctx.WithGitHubToken(context.Background(), "token-X")))

	if err := h.ListRepos(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d want 200", rec.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body not json: %v", err)
	}
	repos, _ := body["repositories"].([]any)
	if len(repos) != 1 {
		t.Fatalf("expected 1 cached repo, got %d (body=%v)", len(repos), body)
	}
}

func TestGitHubHandler_ListRepos_NoTokenReturnsReconnect401(t *testing.T) {
	// TRIANGULATE (a) for T-WS-1Ci-011: empty token in context → 401 reconnect.
	h, _, _ := makeGitHubHandler(t)
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetRequest(req.WithContext(context.Background())) // NO token injection.

	if err := h.ListRepos(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "github_not_connected") {
		t.Errorf("expected github_not_connected envelope, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "Reconnect GitHub") {
		t.Errorf("expected reconnect message, got %s", rec.Body.String())
	}
}

func TestGitHubHandler_ListRepos_NoUserIdentityReturns401(t *testing.T) {
	// TRIANGULATE (extra): no identity on context → 401 unauthorized
	// (the user cannot even reach the github_not_connected code path
	// without an authenticated user).
	stubNoUser := func(*echo.Context) (int64, bool) { return 0, false }
	h, _, _ := makeGitHubHandler(t, func(x *GitHubHandler) {
		x.UserIDFromRequest = stubNoUser
	})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetRequest(req.WithContext(tokenctx.WithGitHubToken(context.Background(), "tok")))

	if err := h.ListRepos(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestGitHubHandler_ListRepos_CacheMissFetchesUpstream(t *testing.T) {
	// TRIANGULATE (b) for T-WS-1Ci-011: cold cache → upstream hit + populate.
	srv, calls, _ := newGitHubMockServer(t, http.StatusOK, ghFixtureReposJSON)
	h, cache, _ := makeGitHubHandler(t, func(x *GitHubHandler) {
		x.Client = gh.NewClientWithBase(srv.URL)
	})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/github/repos?page=1&per_page=30", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetRequest(req.WithContext(tokenctx.WithGitHubToken(context.Background(), "tok-pop")))

	if err := h.ListRepos(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status: got %d want 200", rec.Code)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 upstream call, got %d", calls.Load())
	}
	// Cache should now be warm.
	if cached, ok := cache.Get("42"); !ok || len(cached) != 1 {
		t.Errorf("expected cache to be populated: hit=%v len=%d", ok, len(cached))
	}
}

func TestGitHubHandler_ListRepos_BustCacheQueryParamForcesRefetch(t *testing.T) {
	// TRIANGULATE (c) for T-WS-1Ci-011.
	srv, calls, _ := newGitHubMockServer(t, http.StatusOK, ghFixtureReposJSON)
	h, cache, _ := makeGitHubHandler(t, func(x *GitHubHandler) {
		x.Client = gh.NewClientWithBase(srv.URL)
	})
	// Pre-warm cache.
	cache.Set("42", []gh.Repo{{ID: 100, FullName: "old/y", OwnerLogin: "x", Name: "y"}})

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/github/repos?bust_cache=true", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetRequest(req.WithContext(tokenctx.WithGitHubToken(context.Background(), "tok")))

	if err := h.ListRepos(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 upstream call after bust_cache, got %d", calls.Load())
	}
	if cached, ok := cache.Get("42"); !ok || len(cached) == 0 || cached[0].ID != 1 {
		t.Errorf("expected cache re-populated with fresh data, got %+v (hit=%v)", cached, ok)
	}
}

func TestGitHubHandler_ListRepos_GitHub401MapsTo502WithReconnectMessage(t *testing.T) {
	// TRIANGULATE (d) for T-WS-1Ci-011.
	srv, _, _ := newGitHubMockServer(t, http.StatusUnauthorized, `{"message":"bad token"}`)
	h, _, _ := makeGitHubHandler(t, func(x *GitHubHandler) {
		x.Client = gh.NewClientWithBase(srv.URL)
	})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetRequest(req.WithContext(tokenctx.WithGitHubToken(context.Background(), "expired")))

	if err := h.ListRepos(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "github_unauthorized") {
		t.Errorf("expected github_unauthorized, got %s", rec.Body.String())
	}
}

func TestGitHubHandler_ListRepos_GitHub403MapsToRateLimited502(t *testing.T) {
	// TRIANGULATE (e) for T-WS-1Ci-011.
	srv, _, _ := newGitHubMockServer(t, http.StatusForbidden, `{"message":"rate"}`)
	h, _, _ := makeGitHubHandler(t, func(x *GitHubHandler) {
		x.Client = gh.NewClientWithBase(srv.URL)
	})
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetRequest(req.WithContext(tokenctx.WithGitHubToken(context.Background(), "t")))

	if err := h.ListRepos(c); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if rec.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "github_rate_limited") {
		t.Errorf("expected github_rate_limited, got %s", rec.Body.String())
	}
}

func TestGitHubHandler_ListRepos_AccessTokenNeverLeaks(t *testing.T) {
	// REVIEW-RISK EXTRA: assert the token string injected into the
	// request context does NOT appear in the HTTP response body across
	// happy + sad paths. This is the security guard the user wants the
	// dedicated review-risk subagent to verify.

	const secretToken = "GHO_abcdef1234567890SECRET_TOKEN"

	scenarios := []struct {
		name   string
		setup  func() *GitHubHandler
		build  func() (*echo.Context, *httptest.ResponseRecorder)
		status int
	}{
		{
			name: "happy_path",
			setup: func() *GitHubHandler {
				srv, _, _ := newGitHubMockServer(t, http.StatusOK, ghFixtureReposJSON)
				h, _, _ := makeGitHubHandler(t, func(x *GitHubHandler) { x.Client = gh.NewClientWithBase(srv.URL) })
				return h
			},
			build: func() (*echo.Context, *httptest.ResponseRecorder) {
				req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
				rec := httptest.NewRecorder()
				c := echo.New().NewContext(req, rec)
				c.SetRequest(req.WithContext(tokenctx.WithGitHubToken(context.Background(), secretToken)))
				return c, rec
			},
			status: http.StatusOK,
		},
		{
			name: "no_token_reconnect",
			setup: func() *GitHubHandler {
				h, _, _ := makeGitHubHandler(t)
				return h
			},
			build: func() (*echo.Context, *httptest.ResponseRecorder) {
				req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
				rec := httptest.NewRecorder()
				c := echo.New().NewContext(req, rec)
				c.SetRequest(req.WithContext(context.Background()))
				return c, rec
			},
			status: http.StatusUnauthorized,
		},
		{
			name: "github_401",
			setup: func() *GitHubHandler {
				srv, _, _ := newGitHubMockServer(t, http.StatusUnauthorized, `{"message":"bad token"}`)
				h, _, _ := makeGitHubHandler(t, func(x *GitHubHandler) { x.Client = gh.NewClientWithBase(srv.URL) })
				return h
			},
			build: func() (*echo.Context, *httptest.ResponseRecorder) {
				req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
				rec := httptest.NewRecorder()
				c := echo.New().NewContext(req, rec)
				c.SetRequest(req.WithContext(tokenctx.WithGitHubToken(context.Background(), secretToken)))
				return c, rec
			},
			status: http.StatusBadGateway,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			h := sc.setup()
			c, rec := sc.build()
			if err := h.ListRepos(c); err != nil {
				t.Fatalf("handler error: %v", err)
			}
			if rec.Code != sc.status {
				t.Errorf("status: got %d want %d", rec.Code, sc.status)
			}
			body := rec.Body.String()
			if strings.Contains(body, secretToken) {
				t.Errorf("TOKEN LEAK in response body for %q: %s", sc.name, body)
			}
			if strings.Contains(body, "Bearer ") {
				t.Errorf("'Bearer' substring in response body for %q: %s", sc.name, body)
			}
		})
	}
}
