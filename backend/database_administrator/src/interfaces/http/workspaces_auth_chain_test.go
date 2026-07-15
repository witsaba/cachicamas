// Package httpiface — workspaces_auth_chain_test.go asserts that the
// auth chain (IdentityFromCookie → LoadGitHubTokenMiddleware → handler)
// is mounted on every workspace + GitHub-proxy endpoint.
//
// This is the REGRESSION TEST for the wiring bug caught in PR1c-ii:
// RegisterWorkspaceRoutes + RegisterGitHubRoutes were registered on the
// global Echo instance WITHOUT any auth middleware, while only
// IdentityFromCookie itself was wired onto /whoami. Result:
//
//	curl /workspaces  -> handler hits identityFromContext() == nil ->
//	  400/422 validation envelope with field "auth: Authentication
//	  required." (handler soft-fail)  INSTEAD OF 401 + code=unauthorized
//	  from IdentityFromCookie (middleware short-circuit).
//
// S-REWORK-01..S-REWORK-04 below codify the contract: every route under
// /workspaces/* and /github/* must be 401-protected by middleware, not
// 4xx-from-handler. The test boots a mini composition root with the
// SAME middleware chain main.go installs (and the same
// RegisterWorkspaceRoutes + RegisterGitHubRoutes signatures) and
// inspects responses with httptest.NewRecorder.
//
// Reference: openspec/changes/2026-07-06-workspaces/specs/auth-chain/spec.md
package httpiface

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
	githubinfra "github.com/cachicamas/backend/database_administrator/src/infrastructure/github"
)

// fakeWorkspaceRepo is a minimal in-memory implementation of
// domain.WorkspaceRepository used to feed the workspace handler during
// wiring tests. It panics loudly if a path the test does NOT exercise is
// called, so a regression that bypasses the auth chain AND reaches the
// service layer surfaces as a panic instead of a silent green test.
type fakeWorkspaceRepo struct {
	mu     sync.Mutex
	byOrg  map[int64][]domain.Workspace
	nextID int64
}

func newFakeWorkspaceRepo() *fakeWorkspaceRepo {
	return &fakeWorkspaceRepo{byOrg: map[int64][]domain.Workspace{}}
}

func (r *fakeWorkspaceRepo) Insert(ctx context.Context, w *domain.Workspace) (*domain.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	w.ID = r.nextID
	r.byOrg[w.OrganizationID] = append(r.byOrg[w.OrganizationID], *w)
	return w, nil
}

func (r *fakeWorkspaceRepo) SelectByID(_ context.Context, id int64) (*domain.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, list := range r.byOrg {
		for i := range list {
			if list[i].ID == id && list[i].DeletedAt == nil {
				w := list[i]
				return &w, nil
			}
		}
	}
	return nil, &domain.NotFoundError{Resource: "workspace"}
}

func (r *fakeWorkspaceRepo) SelectAllByOrg(_ context.Context, orgID int64, limit int) ([]domain.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Workspace{}
	for _, w := range r.byOrg[orgID] {
		if w.DeletedAt == nil {
			out = append(out, w)
			if limit > 0 && len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}

func (r *fakeWorkspaceRepo) UpdateName(_ context.Context, id int64, name string) (*domain.Workspace, error) {
	return nil, errors.New("fakeWorkspaceRepo.UpdateName: not implemented in auth-chain test")
}
func (r *fakeWorkspaceRepo) SoftDelete(_ context.Context, id int64) error {
	return errors.New("fakeWorkspaceRepo.SoftDelete: not implemented in auth-chain test")
}

func (r *fakeWorkspaceRepo) MarkSynced(_ context.Context, id int64, commitSHA, defaultBranch string) error {
	return errors.New("fakeWorkspaceRepo.MarkSynced: not implemented in auth-chain test")
}

// fakeGitHubAccessorForChain is a no-op GitHubAccessor for the
// auth-chain test: any call returns "not accessible" so the handler
// fails downstream rather than the test pretending business
// behavior works.
type fakeGitHubAccessorForChain struct{}

func (fakeGitHubAccessorForChain) IsRepoAccessible(_ context.Context, _ int64) (bool, error) {
	return false, errors.New("fakeGitHubAccessorForChain: auth-chain test forces handler-side 4xx")
}

// fakeTokenFetcher is the TokenFetcher implementation the auth chain
// exercises. Returns a fixed token (or any implementation-defined
// string) when given a (provider, account_id). Echoes the calls back
// so the test can assert that the auth chain REACHED it.
type fakeTokenFetcher struct {
	mu       sync.Mutex
	fixed    string
	err      error
	gotCalls []tokenCall
}

type tokenCall struct {
	Provider  string
	AccountID string
}

func (f *fakeTokenFetcher) AccessTokenForIdentity(_ context.Context, provider, accountID string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gotCalls = append(f.gotCalls, tokenCall{Provider: provider, AccountID: accountID})
	if f.err != nil {
		return "", f.err
	}
	return f.fixed, nil
}

// newWiringEcho is the mini composition root. It mirrors main.go's
// chain order verbatim so a regression in either direction is caught:
//
//	IdentityFromCookie
//	→ LoadGitHubTokenMiddleware
//	→ (workspace + github routes)
//
// /health and /identity-callback remain unmounted because they have
// their own wiring (and are unaffected by the workspaces bug).
func newWiringEcho(t *testing.T) (*echo.Echo, *fakeWorkspaceRepo, *fakeTokenFetcher) {
	t.Helper()
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	repo := newFakeWorkspaceRepo()
	ghAcc := fakeGitHubAccessorForChain{}
	svc := application.NewWorkspaceService(repo, nil, ghAcc, logger, nil)
	tFetcher := &fakeTokenFetcher{fixed: "test-oauth-token-fixture"}

	e := echo.New()

	// Auth chain. ORDER IS LOAD-BEARING.
	identityMW := IdentityFromCookie(IdentityMiddlewareConfig{
		AuthSecret:   fixtureAuthSecret,
		CookieName:   fixtureCookieName,
		IdentityRepo: newFakeIdentityRepo(),
		Logger:       logger,
	})
	tokenMW := LoadGitHubTokenMiddleware(tFetcher, logger)

	// The wiring under test: production
	// RegisterAuthenticatedWorkspaceRoutes. Encloses the 8 workspace
	// endpoints + /github/repos behind an Echo group that runs the
	// full auth chain FIRST. main.go uses the same function; tests
	// use it too — so they cannot diverge.
	githubHandler := NewGitHubHandler(
		githubinfra.NewRepoCache(time.Minute),
		githubinfra.NewClient(),
		func(c *echo.Context) (int64, bool) {
			raw := c.Get(IdentityContextKey)
			id, ok := raw.(*domain.Identity)
			if !ok {
				return 0, false
			}
			return id.ID, true
		},
		logger,
	)
	RegisterAuthenticatedWorkspaceRoutes(
		e, svc, githubHandler,
		[]echo.MiddlewareFunc{identityMW, tokenMW},
		logger,
	)

	return e, repo, tFetcher
}

func TestWorkspaces_AuthChain_NoCookie_Returns401FromMiddleware(t *testing.T) {
	// S-REWORK-01: a GET /workspaces WITHOUT a session cookie must be
	// short-circuited by IdentityFromCookie with 401 + code=unauthorized.
	//
	// BEFORE the fix: the auth chain was missing; identityFromContext()
	// in the handler returned nil; the handler emitted a 422 validation
	// envelope {fields: {auth: "Authentication required."}}. The
	// frontend rendered that string verbatim, which is the bug the user
	// reported on /home.
	//
	// AFTER the fix: IdentityFromCookie returns 401 BEFORE the handler
	// runs, with envelope {"code": "unauthorized"} and status 401.
	e, _, _ := newWiringEcho(t)

	for _, tc := range []struct {
		name, method, path string
	}{
		{"GET /workspaces", http.MethodGet, "/workspaces"},
		{"POST /workspaces", http.MethodPost, "/workspaces"},
		{"PATCH /workspaces/123", http.MethodPatch, "/workspaces/123"},
		{"DELETE /workspaces/123", http.MethodDelete, "/workspaces/123"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(""))
			if tc.method == http.MethodPost || tc.method == http.MethodPatch {
				req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
			}
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 from auth middleware, got %d body=%q", rec.Code, rec.Body.String())
			}
			var env map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("response body is not JSON: %v body=%q", err, rec.Body.String())
			}
			if env["code"] != "unauthorized" {
				t.Errorf("expected envelope code=unauthorized, got %v", env["code"])
			}
			// Defense-in-depth: the handler-side "auth: ..." sentinel
			// is what the BUGGY wiring surfaced.
			if env["fields"] != nil || strings.Contains(rec.Body.String(), "Authentication required") {
				t.Errorf("unexpected handler-side auth message: body=%q", rec.Body.String())
			}
		})
	}
}

func TestWorkspaces_AuthChain_NoCookie_DetailAlsoReturns401(t *testing.T) {
	// S-REWORK-02: GET /workspaces/:id (any value, including "not-a-number")
	// must also 401 from middleware, NOT a 400 from the path-param parser.
	// BEFORE the fix: the path-param parser ran first (it lives in the
	// handler); it returned 400 "Invalid id" for non-numeric IDs.
	// AFTER the fix: middleware 401s first.
	e, _, _ := newWiringEcho(t)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/not-a-number", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%q", rec.Code, rec.Body.String())
	}
	var env map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env["code"] != "unauthorized" {
		t.Errorf("expected code=unauthorized, got %v", env["code"])
	}
}

func TestWorkspaces_AuthChain_NoCookie_GitHubProxyReturns401(t *testing.T) {
	// S-REWORK-03: GET /github/repos is also behind the auth chain.
	// Before the fix it 401'd from the handler (which is also wrong,
	// but for a different reason — the handler returns its OWN 4xx when
	// identityFromContext() is nil, then the OAuth client returns 502
	// because the token is empty).
	e, _, _ := newWiringEcho(t)

	req := httptest.NewRequest(http.MethodGet, "/github/repos", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 from middleware, got %d body=%q", rec.Code, rec.Body.String())
	}
	var env map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env["code"] != "unauthorized" {
		t.Errorf("expected envelope code=unauthorized, got %v", env["code"])
	}
}

func TestWorkspaces_AuthChain_ValidCookie_ReachesHandler(t *testing.T) {
	// S-REWORK-04: a valid JWE cookie carries the request through the
	// auth chain to the handler. Asserts:
	//   1. IdentityFromCookie validated the JWE (handler observed identity).
	//   2. LoadGitHubTokenMiddleware ran AND called TokenFetcher with the
	//      (provider, account_id) the cookie established.
	//   3. The workspace handler reached List with no records (200 +
	//      empty body), proving every layer ran end-to-end.
	e, repo, fetcher := newWiringEcho(t)
	repo.byOrg[1] = nil // org 1 exists, no workspaces

	jwe := loadFixtureJWE(t)
	req := httptest.NewRequest(http.MethodGet, "/workspaces", nil)
	req.AddCookie(&http.Cookie{Name: fixtureCookieName, Value: jwe})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 from handler, got %d body=%q", rec.Code, rec.Body.String())
	}
	var body struct {
		Workspaces []any `json:"workspaces"`
		Truncated  bool  `json:"truncated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not JSON: %v body=%q", err, rec.Body.String())
	}
	if len(body.Workspaces) != 0 {
		t.Errorf("expected 0 workspaces for empty org, got %d", len(body.Workspaces))
	}

	// The TokenFetcher was called exactly once with the cookie's
	// (provider, account_id) — proving LoadGitHubTokenMiddleware
	// ran AFTER IdentityFromCookie.
	fetcher.mu.Lock()
	defer fetcher.mu.Unlock()
	if len(fetcher.gotCalls) != 1 {
		t.Fatalf("expected exactly one TokenFetcher call, got %d", len(fetcher.gotCalls))
	}
	if fetcher.gotCalls[0].Provider != "github" {
		t.Errorf("expected provider=github, got %q", fetcher.gotCalls[0].Provider)
	}
	if fetcher.gotCalls[0].AccountID != "12345" {
		t.Errorf("expected account_id=12345 (from fixture), got %q", fetcher.gotCalls[0].AccountID)
	}
}

func TestWorkspaces_AuthChain_PublicEndpointHealthStaysUnauthenticated(t *testing.T) {
	// Triangulation: the auth-chain group MUST NOT contain /health.
	// The /health endpoint is mounted elsewhere (no auth) and must
	// remain reachable for orchestrator probes.
	e, _, _ := newWiringEcho(t)

	// /health is not mounted by RegisterWorkspaceRoutes /
	// RegisterGitHubRoutes so it's a 404 here. That's fine for this
	// test: what matters is that 404 (NOT 401) proves the route is
	// not inside the auth chain.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code == http.StatusUnauthorized {
		t.Fatalf("/health must not be behind the auth chain, got 401")
	}
}
