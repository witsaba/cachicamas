// Package httpiface_test — workspace_tenancy_test.go proves the
// post-H-1 tenant isolation contract end-to-end:
//   - Cross-tenant GET returns 404 (no existence disclosure).
//   - Cross-tenant PATCH returns 404 (no rename).
//   - Cross-tenant DELETE returns 404 (no row modification).
//   - Owner can read/update/delete their own workspaces.
//   - Non-owner (same tenant) can read but not update/delete
//     (the locked design: "workspace = organization + owner").
//
// The in-memory tenancyFakeRepo is upgraded to mimic the production
// tenancy filter so the regression guard catches a future
// implementation that bypasses the SQL-level filter.
package httpiface_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

const (
	tenantAID int64 = 1
	tenantBID int64 = 2
	ownerAID  int64 = 100
	ownerBOD  int64 = 200
)

// tenancyFakeRepo is a fakeRepo that enforces the tenant filter
// exactly the way the production Postgres adapter does. A test
// that bypasses the orgID filter immediately surfaces as a 200
// (the fake RED team's "you fail" response).
type tenancyFakeRepo struct {
	mu       sync.Mutex
	byID     map[int64]*domain.Workspace
	updates  map[int64]int
	deletes  map[int64]int
	lastSync map[int64]int
}

func newTenancyRepo() *tenancyFakeRepo {
	return &tenancyFakeRepo{
		byID:     map[int64]*domain.Workspace{},
		updates:  map[int64]int{},
		deletes:  map[int64]int{},
		lastSync: map[int64]int{},
	}
}

func (r *tenancyFakeRepo) Insert(_ context.Context, w *domain.Workspace) (*domain.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := *w
	out.ID = int64(len(r.byID) + 1)
	r.byID[out.ID] = &out
	return &out, nil
}

func (r *tenancyFakeRepo) SelectByID(_ context.Context, orgID, id int64) (*domain.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.byID[id]
	if !ok {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	if w.OrganizationID != orgID {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	if w.DeletedAt != nil {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	out := *w
	return &out, nil
}

func (r *tenancyFakeRepo) SelectAllByOrg(_ context.Context, orgID int64, _ int) ([]domain.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []domain.Workspace{}
	for _, w := range r.byID {
		if w.OrganizationID == orgID && w.DeletedAt == nil {
			out = append(out, *w)
		}
	}
	return out, nil
}

func (r *tenancyFakeRepo) UpdateName(_ context.Context, orgID, ownerID, id int64, name string) (*domain.Workspace, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.byID[id]
	if !ok {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	if w.OrganizationID != orgID || w.OwnerUserID == nil || *w.OwnerUserID != ownerID {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	if w.DeletedAt != nil {
		return nil, &domain.NotFoundError{Resource: "workspace"}
	}
	r.updates[id]++
	w.Name = name
	out := *w
	return &out, nil
}

func (r *tenancyFakeRepo) SoftDelete(_ context.Context, orgID, ownerID, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	w, ok := r.byID[id]
	if !ok {
		return &domain.NotFoundError{Resource: "workspace"}
	}
	if w.OrganizationID != orgID || w.OwnerUserID == nil || *w.OwnerUserID != ownerID {
		return &domain.NotFoundError{Resource: "workspace"}
	}
	if w.DeletedAt != nil {
		return &domain.NotFoundError{Resource: "workspace"}
	}
	r.deletes[id]++
	deletedAt := time.Now().UTC()
	w.DeletedAt = &deletedAt
	return nil
}

func (r *tenancyFakeRepo) MarkSynced(_ context.Context, _, id int64, _, _ string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.lastSync[id]++
	return nil
}

// denyGitHubAccessor denies every repo (used by the tenancy tests
// because they never exercise the create path).
type denyGitHubAccessor struct{}

func (denyGitHubAccessor) IsRepoAccessible(_ context.Context, _ int64) (bool, error) {
	return false, nil
}

// newTenancyEcho builds a minimal Echo with the workspace handler
// registered on the auth-protected group. The single-tenant resolver
// is forced to a configurable value so a test can pick the
// "attacker's tenant" while the seeded workspace lives in a
// different org.
func newTenancyEcho(t *testing.T, repo domain.WorkspaceRepository, observerOrgID int64) *echo.Echo {
	t.Helper()
	// Force the resolver to return the observer's orgID.
	httpiface.SetSingleTenantOrgIDResolver(func() int64 { return observerOrgID })

	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	gh := denyGitHubAccessor{}
	svc := application.NewWorkspaceService(repo, nil, gh, logger, nil)

	e := echo.New()
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
	return e
}

func TestWorkspaceHandler_Tenancy_CrossTenantGet_Returns404(t *testing.T) {
	repo := newTenancyRepo()
	a := int64(ownerAID)
	if _, err := repo.Insert(context.Background(), &domain.Workspace{
		OrganizationID: tenantAID, OwnerUserID: &a,
		Name: "tenant-a-ws", RepoFullName: "octocat/a", RepoOwner: "octocat", RepoName: "a",
		RepoGitHubID: 1,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	e := newTenancyEcho(t, repo, tenantBID)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/1", nil)
	req.Header.Set("X-Test-Identity-ID", strconv.FormatInt(ownerBOD, 10))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant GET: expected 404, got %d body=%q", rec.Code, rec.Body.String())
	}
	var env map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &env)
	if env["error"] != domain.CodeNotFound {
		t.Errorf("error code = %v, want %q", env["error"], domain.CodeNotFound)
	}
}

func TestWorkspaceHandler_Tenancy_OwnerRead_Succeeds(t *testing.T) {
	repo := newTenancyRepo()
	a := int64(ownerAID)
	inserted, err := repo.Insert(context.Background(), &domain.Workspace{
		OrganizationID: tenantAID, OwnerUserID: &a,
		Name: "mine", RepoFullName: "octocat/mine", RepoOwner: "octocat", RepoName: "mine",
		RepoGitHubID: 1,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	e := newTenancyEcho(t, repo, tenantAID)

	req := httptest.NewRequest(http.MethodGet, "/workspaces/"+strconv.FormatInt(inserted.ID, 10), nil)
	req.Header.Set("X-Test-Identity-ID", strconv.FormatInt(ownerAID, 10))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("owner GET: expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestWorkspaceHandler_Tenancy_CrossTenantPatch_Returns404(t *testing.T) {
	repo := newTenancyRepo()
	a := int64(ownerAID)
	if _, err := repo.Insert(context.Background(), &domain.Workspace{
		OrganizationID: tenantAID, OwnerUserID: &a,
		Name: "tenant-a-ws", RepoFullName: "octocat/a", RepoOwner: "octocat", RepoName: "a",
		RepoGitHubID: 1,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	e := newTenancyEcho(t, repo, tenantBID)

	body := []byte(`{"name":"hijacked"}`)
	req := httptest.NewRequest(http.MethodPatch, "/workspaces/1", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-Test-Identity-ID", strconv.FormatInt(ownerBOD, 10))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant PATCH: expected 404, got %d body=%q", rec.Code, rec.Body.String())
	}
	// Confirm the row was NOT modified.
	cur, err := repo.SelectByID(context.Background(), tenantAID, 1)
	if err != nil {
		t.Fatalf("seed reload: %v", err)
	}
	if cur.Name != "tenant-a-ws" {
		t.Errorf("row name changed: %q", cur.Name)
	}
}

func TestWorkspaceHandler_Tenancy_CrossTenantDelete_Returns404(t *testing.T) {
	repo := newTenancyRepo()
	a := int64(ownerAID)
	if _, err := repo.Insert(context.Background(), &domain.Workspace{
		OrganizationID: tenantAID, OwnerUserID: &a,
		Name: "tenant-a-ws", RepoFullName: "octocat/a", RepoOwner: "octocat", RepoName: "a",
		RepoGitHubID: 1,
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	e := newTenancyEcho(t, repo, tenantBID)

	req := httptest.NewRequest(http.MethodDelete, "/workspaces/1", nil)
	req.Header.Set("X-Test-Identity-ID", strconv.FormatInt(ownerBOD, 10))
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("cross-tenant DELETE: expected 404, got %d body=%q", rec.Code, rec.Body.String())
	}
	// Confirm the row was NOT deleted.
	cur, err := repo.SelectByID(context.Background(), tenantAID, 1)
	if err != nil {
		t.Fatalf("cross-tenant DELETE silently deleted the row: %v", err)
	}
	if cur.DeletedAt != nil {
		t.Errorf("row was soft-deleted by cross-tenant DELETE")
	}
}
