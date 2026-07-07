// Package httpiface — workspace_handler_test.go covers the 5
// workspace HTTP endpoints (post-2026-07-08-workspaces-simplify).
// Uses the same in-memory fakeRepo + fakeGitHubAccessor + noop
// OTel tracer pattern as organization_handler_test.go.
//
// 2026-07-08-workspaces-simplify changelog:
//   - Dropped 4 test functions that exercised the linked-repo
//     endpoints (AddRepo, RemoveRepo, ListRepos + their
//     not-accessible / no-leakage variants).
//   - Dropped the linked-repo bookkeeping on fakeRepo
//     (linkedRepos map, addRepoErr, addRepoResult, the 3 fake
//     methods).
//   - Renamed primary_repository -> repository in the test body
//     shapes + assertions.
//   - Updated TestWorkspaceHandler_Get_WireShapeFlat to assert
//     the new shape (top-level "repository" + no
//     "linked_repositories").
//   - Updated TestWorkspaceHandler_Update_DropsPrimaryRepository
//     to TestWorkspaceHandler_Update_DropsRepository.
package httpiface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http/tokenctx"
)

// ---------------------------------------------------------------------------
// fakes
// ---------------------------------------------------------------------------

type fakeRepo struct {
	mu sync.Mutex

	insertResult *domain.Workspace
	insertErr    error

	byID        map[int64]*domain.Workspace
	listByOrg   map[int64][]domain.Workspace
	deletedIDs  map[int64]bool // soft-deleted
	updateErr   error
	deleteErr   error
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		byID:       map[int64]*domain.Workspace{},
		listByOrg:  map[int64][]domain.Workspace{},
		deletedIDs: map[int64]bool{},
	}
}

func (f *fakeRepo) Insert(_ context.Context, w *domain.Workspace) (*domain.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.insertErr != nil {
		return nil, f.insertErr
	}
	if f.insertResult != nil {
		return f.insertResult, nil
	}
	out := *w
	out.ID = int64(len(f.byID) + 1)
	out.CreatedAt = time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	out.UpdatedAt = out.CreatedAt
	f.byID[out.ID] = &out
	f.listByOrg[out.OrganizationID] = append(f.listByOrg[out.OrganizationID], out)
	return &out, nil
}

func (f *fakeRepo) SelectByID(_ context.Context, id int64) (*domain.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deletedIDs[id] {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	w, ok := f.byID[id]
	if !ok {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	return w, nil
}

func (f *fakeRepo) SelectAllByOrg(_ context.Context, orgID int64, limit int) ([]domain.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	all := f.listByOrg[orgID]
	out := make([]domain.Workspace, 0, len(all))
	for _, w := range all {
		if !f.deletedIDs[w.ID] {
			out = append(out, w)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeRepo) UpdateName(_ context.Context, id int64, name string) (*domain.Workspace, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	if f.deletedIDs[id] {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	w, ok := f.byID[id]
	if !ok {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	w.Name = name
	w.UpdatedAt = time.Now().UTC()
	return w, nil
}

func (f *fakeRepo) SoftDelete(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if f.deletedIDs[id] {
		return &domain.NotFoundError{Resource: "workspace"}
	}
	if _, ok := f.byID[id]; !ok {
		return &domain.NotFoundError{Resource: "workspace"}
	}
	f.deletedIDs[id] = true
	return nil
}

// fakeGitHubAccessor — in-memory map of accessible repo IDs.
type fakeGitHubAccessor struct {
	mu         sync.Mutex
	accessible map[int64]bool
	err        error
}

func (a *fakeGitHubAccessor) IsRepoAccessible(_ context.Context, id int64) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.err != nil {
		return false, a.err
	}
	return a.accessible[id], nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestHandler(repo *fakeRepo, ghAcc *fakeGitHubAccessor, t *testing.T) (httpiface.WorkspaceHandler, *application.WorkspaceService, *echo.Echo) {
	_ = t
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	svc := application.NewWorkspaceService(repo, ghAcc, logger, noopTracer())
	e := echo.New()
	// Test identity-injection middleware: reads X-Test-Identity-ID
	// header and seeds c.Set(IdentityContextKey). This stands in for
	// the production IdentityFromCookie chain that
	// RegisterAuthenticatedWorkspaceRoutes requires. Production
	// wiring is exercised end-to-end in workspaces_auth_chain_test.go.
	identityInjector := func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if idHeader := c.Request().Header.Get("X-Test-Identity-ID"); idHeader != "" {
				uid, _ := strconv.ParseInt(idHeader, 10, 64)
				c.Set(httpiface.IdentityContextKey, &domain.Identity{
					ID:                uid,
					Provider:          "github",
					ProviderAccountID: "test-account-" + idHeader,
				})
			}
			return next(c)
		}
	}
	httpiface.RegisterAuthenticatedWorkspaceRoutes(
		e, svc, nil, []echo.MiddlewareFunc{identityInjector}, logger,
	)
	return httpiface.WorkspaceHandler{}, svc, e
}

// noopTracer returns the noop tracer from the application package's
// default. The application.NewWorkspaceService with a nil tracer
// falls back to noop inside the package, so we just pass nil.
func noopTracer() trace.Tracer { return nil }

// Set the single-tenant resolver to 1 (test default).
func init() {
	httpiface.SetSingleTenantOrgIDResolver(func() int64 { return 1 })
}

// dispatch issues a request through the registered handler.
func dispatch(t *testing.T, e *echo.Echo, method, path string, body any, identity *domain.Identity) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		bodyReader = bytes.NewReader(raw)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if body != nil {
		req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	}
	req = req.WithContext(tokenctx.WithGitHubToken(req.Context(), ""))
	if identity != nil {
		req.Header.Set("X-Test-Identity-ID", strconv.FormatInt(identity.ID, 10))
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// ---------------------------------------------------------------------------
// Tests (post-2026-07-08-workspaces-simplify: 5 endpoints covered).
// ---------------------------------------------------------------------------

// 1. POST /workspaces — empty token pre-flight returns 401 reconnect
// (R-WS-017).
func TestWorkspaceHandler_Create_NoToken_Returns401(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{accessible: map[int64]bool{42: true}}
	_, _, e := newTestHandler(repo, gh, t)

	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	body := map[string]any{
		"name": "my-workspace",
		"repository": map[string]any{
			"github_id": 42,
			"full_name": "octocat/hello",
			"owner":     "octocat",
			"name":      "hello",
		},
	}
	rec := dispatch(t, e, http.MethodPost, "/workspaces", body, identity)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 (github_not_connected), got %d: %s", rec.Code, rec.Body.String())
	}
	if !regexp.MustCompile(`github_not_connected`).MatchString(rec.Body.String()) {
		t.Errorf("expected envelope code=github_not_connected, got: %s", rec.Body.String())
	}
}

// TestWorkspaceHandler_Create_ValidationError — empty body triggers
// the validator; validation fires BEFORE the empty-token
// pre-flight.
func TestWorkspaceHandler_Create_ValidationError(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{accessible: map[int64]bool{42: true}}
	_, _, e := newTestHandler(repo, gh, t)

	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	body := map[string]any{
		"name":       "",
		"repository": map[string]any{},
	}
	rec := dispatch(t, e, http.MethodPost, "/workspaces", body, identity)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestWorkspaceHandler_NoTokenResponse_NoLeakage — token-leak regression:
// no token-shaped string appears in any response body.
func TestWorkspaceHandler_NoTokenResponse_NoLeakage(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{accessible: map[int64]bool{}}
	_, _, e := newTestHandler(repo, gh, t)

	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	body := map[string]any{"name": "ws", "repository": map[string]any{"github_id": 42}}
	rec := dispatch(t, e, http.MethodPost, "/workspaces", body, identity)

	if tokenLeaked(t, rec.Body.Bytes()) {
		t.Errorf("token-shaped string leaked in response body: %s", rec.Body.String())
	}
}

// TestWorkspaceHandler_Delete_Returns204 — happy path.
func TestWorkspaceHandler_Delete_Returns204(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	w := seedWorkspace(t, repo, 1)

	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	rec := dispatch(t, e, http.MethodDelete, "/workspaces/"+strconv.FormatInt(w.ID, 10), nil, identity)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestWorkspaceHandler_Delete_AlreadyDeleted_Returns404.
func TestWorkspaceHandler_Delete_AlreadyDeleted_Returns404(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	w := seedWorkspace(t, repo, 1)

	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	dispatch(t, e, http.MethodDelete, "/workspaces/"+strconv.FormatInt(w.ID, 10), nil, identity)
	rec := dispatch(t, e, http.MethodDelete, "/workspaces/"+strconv.FormatInt(w.ID, 10), nil, identity)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestWorkspaceHandler_Get_NotFound_Returns404.
func TestWorkspaceHandler_Get_NotFound_Returns404(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	rec := dispatch(t, e, http.MethodGet, "/workspaces/9999", nil, identity)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestWorkspaceHandler_Update_DropsRepository — locked design decision:
// PATCH silently ignores the repository field (the repo is the
// workspace's identity and cannot change post-create).
//
// 2026-07-08-workspaces-simplify: renamed from
// DropsPrimaryRepository; the wire field is now "repository".
func TestWorkspaceHandler_Update_DropsRepository(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	w := seedWorkspace(t, repo, 1)

	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	body := map[string]any{
		"name": "new-name",
		"repository": map[string]any{
			"github_id": 999,
			"full_name": "attacker/repo",
		},
	}
	rec := dispatch(t, e, http.MethodPatch, "/workspaces/"+strconv.FormatInt(w.ID, 10), body, identity)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	got, _ := repo.SelectByID(context.Background(), w.ID)
	if got.RepoGitHubID != w.RepoGitHubID {
		t.Errorf("repo changed: was %d, got %d", w.RepoGitHubID, got.RepoGitHubID)
	}
	if got.Name != "new-name" {
		t.Errorf("expected name 'new-name', got %q", got.Name)
	}
}

// TestWorkspaceHandler_List_ReturnsWorkspaces.
func TestWorkspaceHandler_List_ReturnsWorkspaces(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	_ = seedWorkspace(t, repo, 1)

	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	rec := dispatch(t, e, http.MethodGet, "/workspaces", nil, identity)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestWorkspaceHandler_List_NoLeakage.
func TestWorkspaceHandler_List_NoLeakage(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	_ = seedWorkspace(t, repo, 1)
	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	rec := dispatch(t, e, http.MethodGet, "/workspaces", nil, identity)
	if tokenLeaked(t, rec.Body.Bytes()) {
		t.Errorf("token-shaped string leaked in list response: %s", rec.Body.String())
	}
}

// TestWorkspaceHandler_Get_NoLeakage.
func TestWorkspaceHandler_Get_NoLeakage(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	w := seedWorkspace(t, repo, 1)
	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	rec := dispatch(t, e, http.MethodGet, "/workspaces/"+strconv.FormatInt(w.ID, 10), nil, identity)
	if tokenLeaked(t, rec.Body.Bytes()) {
		t.Errorf("token-shaped string leaked in get response: %s", rec.Body.String())
	}
}

// TestWorkspaceHandler_Get_WireShapeFlat — 2026-07-07 / 2026-07-08
// regression for the SSR 500. The frontend's WorkspaceDetail
// interface reads the workspace fields flat at the top level:
//
//   - id
//   - name
//   - repository   (the workspace's only repo, post-simplify)
//   - created_at
//   - updated_at
//
// NOT a wrapper like {workspace: {...}, linked_repositories: [...]}.
// Pre-fix (PR1c-ii era) Get returned the wrapper shape while
// Create/Update returned flat — the frontend decoder treated every
// top-level field as undefined → primaryRepo.full_name crashed SSR
// with TypeError. 2026-07-08-workspaces-simplify removes the
// linked_repositories field entirely.
func TestWorkspaceHandler_Get_WireShapeFlat(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	w := seedWorkspace(t, repo, 1)
	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	rec := dispatch(t, e, http.MethodGet, "/workspaces/"+strconv.FormatInt(w.ID, 10), nil, identity)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// Workspace fields MUST be at the top level.
	for _, key := range []string{"id", "name", "repository", "created_at", "updated_at"} {
		if _, ok := body[key]; !ok {
			t.Errorf("GET /workspaces/:id response missing top-level key %q (body=%s)", key, rec.Body.String())
		}
	}
	// legacy keys MUST NOT be present (post-simplify we no longer ship
	// them at all).
	for _, legacy := range []string{"primary_repository", "linked_repositories"} {
		if _, ok := body[legacy]; ok {
			t.Errorf("GET /workspaces/:id response should not carry legacy key %q (body=%s)", legacy, rec.Body.String())
		}
	}
	// The wrapper MUST NOT be present.
	if _, ok := body["workspace"]; ok {
		t.Errorf("GET /workspaces/:id response still has the legacy wrapper key %q (body=%s)", "workspace", rec.Body.String())
	}
}

// TestWorkspaceHandler_Patch_NoLeakage.
func TestWorkspaceHandler_Patch_NoLeakage(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	w := seedWorkspace(t, repo, 1)
	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	body := map[string]any{"name": "n2"}
	rec := dispatch(t, e, http.MethodPatch, "/workspaces/"+strconv.FormatInt(w.ID, 10), body, identity)
	if tokenLeaked(t, rec.Body.Bytes()) {
		t.Errorf("token-shaped string leaked in patch response: %s", rec.Body.String())
	}
}

// TestWorkspaceHandler_NoIdentity_ReturnsValidationError — defense in
// depth: with no identity in the Echo context, the handler returns
// 400 + validation envelope. In production the auth chain runs
// BEFORE the handler and 401s anonymous requests; this 400 path is
// unreachable from real traffic. End-to-end auth-chain coverage
// lives in workspaces_auth_chain_test.go.
func TestWorkspaceHandler_NoIdentity_ReturnsValidationError(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	rec := dispatch(t, e, http.MethodGet, "/workspaces", nil, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 (no identity, defense in depth), got %d: %s", rec.Code, rec.Body.String())
	}
}

// TestWorkspaceHandler_Get_WireShapeFlat_Summary — companion to
// TestWorkspaceHandler_Get_WireShapeFlat: also assert the LIST
// endpoint returns `workspaces[]` items WITHOUT the legacy keys
// (primary_repository, linked_repos_count). This is the wire shape
// the frontend WorkspaceSummary type reads.
func TestWorkspaceHandler_List_WireShape1to1(t *testing.T) {
	repo := newFakeRepo()
	gh := &fakeGitHubAccessor{}
	_, _, e := newTestHandler(repo, gh, t)
	_ = seedWorkspace(t, repo, 1)
	identity := &domain.Identity{ID: 1, Provider: "github", ProviderAccountID: "u1"}
	rec := dispatch(t, e, http.MethodGet, "/workspaces", nil, identity)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Workspaces []map[string]json.RawMessage `json:"workspaces"`
		Truncated  bool                        `json:"truncated"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(body.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace, got %d", len(body.Workspaces))
	}
	item := body.Workspaces[0]
	for _, key := range []string{"id", "name", "repository", "created_at"} {
		if _, ok := item[key]; !ok {
			t.Errorf("list item missing %q (item=%v)", key, item)
		}
	}
	for _, legacy := range []string{"primary_repository", "linked_repos_count"} {
		if _, ok := item[legacy]; ok {
			t.Errorf("list item should not carry legacy key %q (item=%v)", legacy, item)
		}
	}
}

// ---------------------------------------------------------------------------
// Shared utilities
// ---------------------------------------------------------------------------

func seedWorkspace(t *testing.T, repo *fakeRepo, orgID int64) *domain.Workspace {
	t.Helper()
	uid := int64(1)
	w, err := repo.Insert(context.Background(), &domain.Workspace{
		OrganizationID: orgID,
		OwnerUserID:    &uid,
		Name:           "seeded",
		RepoGitHubID:   7,
		RepoFullName:   "octocat/seeded",
		RepoOwner:      "octocat",
		RepoName:       "seeded",
	})
	if err != nil {
		t.Fatalf("seed workspace: %v", err)
	}
	return w
}

// tokenRegex matches `gho_<16+ alnum>` (GitHub OAuth token format).
var tokenRegex = regexp.MustCompile(`gho_[a-zA-Z0-9]{16,}`)

func tokenLeaked(t *testing.T, body []byte) bool {
	t.Helper()
	return tokenRegex.Match(body)
}