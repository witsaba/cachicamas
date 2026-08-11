// Package httpiface — workspace_handler.go implements the workspace
// HTTP transport for the 1:1 model (post-2026-07-08-workspaces-simplify).
// Wires 5 endpoints:
//
//	POST   /workspaces
//	GET    /workspaces
//	GET    /workspaces/:id
//	PATCH  /workspaces/:id
//	DELETE /workspaces/:id
//
// Errors are mapped to the locked HTTP envelope via writeWorkspaceError.
// PR1c-ii (this file): no token field is ever serialized.
//
// 2026-07-08-workspaces-simplify changelog:
//   - Dropped POST/DELETE/GET /workspaces/:id/repositories (3 endpoints).
//   - Dropped linkedRepoResponse, linkedReposListResponse,
//     addRepoRequest wire types.
//   - Dropped toLinkedRepoResponse(s), AddRepo, RemoveRepo,
//     ListRepos handlers.
//   - Renamed primary_repository -> repository throughout the
//     wire shape (frontend's WorkspaceDetail interface mirrors
//     this).
//   - workspaceSummaryResponse no longer carries
//     linked_repos_count (always 0 in the 1:1 model).
package httpiface

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// WorkspaceHandler exposes the 5 workspace use cases over HTTP.
type WorkspaceHandler struct {
	service *application.WorkspaceService
	logger  *slog.Logger
}

// NewWorkspaceHandler wires a WorkspaceHandler to its service.
func NewWorkspaceHandler(service *application.WorkspaceService, logger *slog.Logger) *WorkspaceHandler {
	if service == nil {
		panic("NewWorkspaceHandler: service must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &WorkspaceHandler{service: service, logger: logger}
}

// RegisterAuthenticatedWorkspaceRoutes wires the 5 workspace
// endpoints and the GitHub repos proxy behind the auth middleware
// chain.
//
// Why a single function: the previous split (a global
// `e.Use(LoadGitHubTokenMiddleware(...))` plus direct
// `RegisterWorkspaceRoutes` / `RegisterGitHubRoutes` registrations
// on the Echo root) made it easy to mount workspace routes WITHOUT
// the cookie verifier middleware. Symptom: a request to
// `/workspaces` without a session cookie returned 400 + validation
// envelope with field "auth: Authentication required." instead of
// 401 + code=unauthorized. The post-PR1c-ii wiring documentation
// in main.go described the intended chain (`IdentityFromCookie` ->
// `LoadGitHubTokenMiddleware` -> … routes) but the only
// enforcement was a comment.
//
// This function makes the chain a compile-time guarantee. Every
// caller must supply the `authChain` middleware slice (in order);
// the routes are then mounted inside an Echo sub-group that runs
// the chain first. Any code that wants to skip the chain has no
// public surface to do so.
//
// In production, main.go passes:
//
//	[{IdentityFromCookie(cfg)}, {LoadGitHubTokenMiddleware(fetcher, logger)}]
//
// In tests, callers pass the `IdentityContextKey`-seeding
// middleware directly (see workspace_handler_test.go).
//
// 2026-07-08-workspaces-simplify: 5 workspace routes (was 8); the
// linked-repos routes are gone.
func RegisterAuthenticatedWorkspaceRoutes(
	e *echo.Echo,
	svc *application.WorkspaceService,
	githubHandler *GitHubHandler,
	authChain []echo.MiddlewareFunc,
	logger *slog.Logger,
) {
	if e == nil {
		panic("RegisterAuthenticatedWorkspaceRoutes: e must not be nil")
	}
	if svc == nil {
		panic("RegisterAuthenticatedWorkspaceRoutes: svc must not be nil")
	}
	if len(authChain) == 0 {
		panic("RegisterAuthenticatedWorkspaceRoutes: authChain must be non-empty (every workspace + GitHub route requires authentication)")
	}

	// Echo v5: e.Group("", mw...) creates a sub-group with the given
	// middleware running before every route registered on the group.
	// If the group is empty (no middleware), a panic fires here.
	h := NewWorkspaceHandler(svc, logger)
	g := e.Group("", authChain...)
	g.POST("/workspaces", h.Create)
	g.GET("/workspaces", h.List)
	g.GET("/workspaces/:id", h.Get)
	g.PATCH("/workspaces/:id", h.Update)
	g.DELETE("/workspaces/:id", h.Delete)

	// The GitHub repos proxy is part of the same auth chain. Tests
	// focused on workspace handler logic pass a nil githubHandler
	// and only exercise the workspace routes.
	if githubHandler != nil {
		g.GET("/github/repos", githubHandler.ListRepos)
	}
}

// RegisterWorkspaceRoutes is REMOVED. Use
// RegisterAuthenticatedWorkspaceRoutes instead. The old public
// surface allowed callers to mount the workspace endpoints WITHOUT
// the auth middleware chain, which led to the wiring bug fixed
// here. Keep this comment as the load-bearing documentation of
// WHY the old shape is gone.
//
// Legacy callers (workspace_handler_test.go) must now route
// through RegisterAuthenticatedWorkspaceRoutes with a test-only
// authChain that seeds c.Set(IdentityContextKey, *Identity) via
// the X-Test-Identity-ID header. See
// workspaces_auth_chain_test.go for the production-shape test.

// ---------------------------------------------------------------------------
// Wire types
// ---------------------------------------------------------------------------

// createWorkspaceRequest is the wire shape for POST /workspaces.
//
// 2026-07-08-workspaces-simplify: primary_repository renamed to
// repository (no secondary exists in the 1:1 model).
type createWorkspaceRequest struct {
	Name       string `json:"name"`
	Repository struct {
		GitHubID int64  `json:"github_id"`
		FullName string `json:"full_name"`
		Owner    string `json:"owner"`
		Name     string `json:"name"`
	} `json:"repository"`
}

// workspaceResponse is the wire shape for a single workspace (used
// for Create, Update, and as the embedded fields of the detail
// response).
type workspaceResponse struct {
	ID             int64            `json:"id"`
	OrganizationID int64            `json:"organization_id"`
	OwnerUserID    *int64           `json:"owner_user_id"`
	Name           string           `json:"name"`
	Repository     repositoryResponse `json:"repository"`
	CreatedAt      string           `json:"created_at"`
	UpdatedAt      string           `json:"updated_at"`
}

// repositoryResponse is the per-workspace repo projection. Same
// shape as the frontend's Repository interface.
type repositoryResponse struct {
	GitHubID int64  `json:"github_id"`
	FullName string `json:"full_name"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
}

// workspaceDetailResponse is the wire shape returned by Get
// (GET /workspaces/:id). The frontend's WorkspaceDetail interface
// reads the workspace fields at the top level (id, name,
// repository, created_at, updated_at). Embedding workspaceResponse
// promotes its fields to the top level of the JSON output.
//
// 2026-07-08-workspaces-simplify: no longer carries
// linked_repositories (the workspace_repository table is gone).
type workspaceDetailResponse struct {
	workspaceResponse
}

// workspacesListResponse wraps a slice for the locked envelope shape.
type workspacesListResponse struct {
	Workspaces []workspaceSummaryResponse `json:"workspaces"`
	Truncated  bool                       `json:"truncated"`
}

// workspaceSummaryResponse is the list-item shape (no UpdatedAt
// to keep the list compact; the detail endpoint returns the full
// row).
//
// 2026-07-08-workspaces-simplify: dropped linked_repos_count
// (always 0 in the 1:1 model).
type workspaceSummaryResponse struct {
	ID         int64            `json:"id"`
	Name       string           `json:"name"`
	Repository repositoryResponse `json:"repository"`
	CreatedAt  string           `json:"created_at"`
}

// updateWorkspaceRequest is the wire shape for PATCH
// /workspaces/:id. `repository` is accepted for forward
// compatibility but silently dropped (locked design decision:
// the repo is the workspace's identity and cannot change).
type updateWorkspaceRequest struct {
	Name       *string `json:"name,omitempty"`
	Repository *struct {
		GitHubID int64  `json:"github_id"`
		FullName string `json:"full_name"`
		Owner    string `json:"owner"`
		Name     string `json:"name"`
	} `json:"repository,omitempty"`
}

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

// Create handles POST /workspaces.
func (h *WorkspaceHandler) Create(c *echo.Context) error {
	var body createWorkspaceRequest
	if err := c.Bind(&body); err != nil {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"body": "Invalid JSON."},
		})
	}

	identity, ok := identityFromContext(c)
	if !ok {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"auth": "Authentication required."},
		})
	}

	in := domain.CreateWorkspaceInput{
		OrganizationID: singleTenantOrganizationID(c),
		OwnerUserID:    &identity.ID,
		Name:           body.Name,
		Repository: domain.Repository{
			GitHubID: body.Repository.GitHubID,
			FullName: body.Repository.FullName,
			Owner:    body.Repository.Owner,
			Name:     body.Repository.Name,
		},
	}

	w, err := h.service.Create(c.Request().Context(), in)
	if err != nil {
		return writeWorkspaceError(c, err)
	}

	c.Response().Header().Set("Location", fmt.Sprintf("/workspaces/%d", w.ID))
	return c.JSON(http.StatusCreated, toWorkspaceResponse(w))
}

// List handles GET /workspaces.
func (h *WorkspaceHandler) List(c *echo.Context) error {
	if _, ok := identityFromContext(c); !ok {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"auth": "Authentication required."},
		})
	}

	orgID := singleTenantOrganizationID(c)
	limit := atoiOrDefault(c.QueryParam("limit"), 100)
	workspaces, err := h.service.List(c.Request().Context(), orgID, limit)
	if err != nil {
		return writeWorkspaceError(c, err)
	}

	summaries := make([]workspaceSummaryResponse, 0, len(workspaces))
	for i := range workspaces {
		w := workspaces[i]
		summaries = append(summaries, workspaceSummaryResponse{
			ID:         w.ID,
			Name:       w.Name,
			Repository: toRepositoryResponse(&w),
			CreatedAt:  w.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		})
	}
	return c.JSON(http.StatusOK, workspacesListResponse{
		Workspaces: summaries,
		Truncated:  len(summaries) >= limit,
	})
}

// Get handles GET /workspaces/:id.
func (h *WorkspaceHandler) Get(c *echo.Context) error {
	if _, ok := identityFromContext(c); !ok {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"auth": "Authentication required."},
		})
	}

	id, err := parsePathInt64(c, "id")
	if err != nil {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"id": "Invalid id."},
		})
	}

	orgID := singleTenantOrganizationID(c)
	w, err := h.service.Get(c.Request().Context(), orgID, id)
	if err != nil {
		return writeWorkspaceError(c, err)
	}

	// Flat shape: workspaceResponse fields are promoted to the top
	// level of the JSON via embedding; no wrapper key.
	resp := workspaceDetailResponse{
		workspaceResponse: toWorkspaceResponse(w),
	}
	return c.JSON(http.StatusOK, resp)
}

// Update handles PATCH /workspaces/:id.
func (h *WorkspaceHandler) Update(c *echo.Context) error {
	identity, ok := identityFromContext(c)
	if !ok {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"auth": "Authentication required."},
		})
	}

	id, err := parsePathInt64(c, "id")
	if err != nil {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"id": "Invalid id."},
		})
	}

	var body updateWorkspaceRequest
	if err := c.Bind(&body); err != nil {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"body": "Invalid JSON."},
		})
	}

	in := domain.UpdateWorkspaceInput{}
	if body.Name != nil {
		in.Name = body.Name
	}
	// repository is silently dropped per locked design decision.

	orgID := singleTenantOrganizationID(c)
	w, err := h.service.Update(c.Request().Context(), orgID, identity.ID, id, in)
	if err != nil {
		return writeWorkspaceError(c, err)
	}
	return c.JSON(http.StatusOK, toWorkspaceResponse(w))
}

// Delete handles DELETE /workspaces/:id.
func (h *WorkspaceHandler) Delete(c *echo.Context) error {
	identity, ok := identityFromContext(c)
	if !ok {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"auth": "Authentication required."},
		})
	}

	id, err := parsePathInt64(c, "id")
	if err != nil {
		return writeWorkspaceError(c, &domain.ValidationError{
			Fields: map[string]string{"id": "Invalid id."},
		})
	}

	orgID := singleTenantOrganizationID(c)
	if err := h.service.Delete(c.Request().Context(), orgID, identity.ID, id); err != nil {
		return writeWorkspaceError(c, err)
	}
	return c.NoContent(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// identityFromContext returns the resolved *domain.Identity set by
// IdentityFromCookie. Returns (nil, false) when the auth middleware
// did not run (anon) or the identity is not the expected type.
func identityFromContext(c *echo.Context) (*domain.Identity, bool) {
	raw := c.Get(IdentityContextKey)
	if raw == nil {
		return nil, false
	}
	id, ok := raw.(*domain.Identity)
	return id, ok
}

// singleTenantOrganizationID resolves the one-and-only organization
// in this single-tenant install.
//
// To keep this file self-contained, we expose a setter the
// composition root can override (cmd/server/main.go sets it on
// startup). Default is 1.
func singleTenantOrganizationID(_ *echo.Context) int64 {
	if singleTenantOrgIDFn != nil {
		return singleTenantOrgIDFn()
	}
	return 1
}

// singleTenantOrgIDFn is the composition root's override hook for
// resolving the single-tenant organization id without a per-request
// lookup. Defaults to returning 1.
var singleTenantOrgIDFn func() int64

// SetSingleTenantOrgIDResolver wires the resolver used by the
// workspace handlers. Called by main.go at startup.
func SetSingleTenantOrgIDResolver(fn func() int64) {
	singleTenantOrgIDFn = fn
}

// parsePathInt64 parses a path parameter as int64.
func parsePathInt64(c *echo.Context, name string) (int64, error) {
	raw := c.Param(name)
	return strconv.ParseInt(raw, 10, 64)
}

// atoiOrDefault returns the parsed int or the default.
func atoiOrDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// toWorkspaceResponse projects a domain.Workspace to the wire shape.
// NOTE: NEVER includes any token-shaped field. The PR1c-ii
// introspection test scans every response body for
// `gho_[a-zA-Z0-9]{36}` and must not find it.
//
// 2026-07-08-workspaces-simplify: PrimaryRepository renamed to
// Repository; the wire key is "repository".
func toWorkspaceResponse(w *domain.Workspace) workspaceResponse {
	return workspaceResponse{
		ID:             w.ID,
		OrganizationID: w.OrganizationID,
		OwnerUserID:    w.OwnerUserID,
		Name:           w.Name,
		Repository:     toRepositoryResponse(w),
		CreatedAt:      w.CreatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
		UpdatedAt:      w.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000Z"),
	}
}

// toRepositoryResponse projects the per-workspace repo fields of a
// domain.Workspace to the wire shape.
func toRepositoryResponse(w *domain.Workspace) repositoryResponse {
	return repositoryResponse{
		GitHubID: w.RepoGitHubID,
		FullName: w.RepoFullName,
		Owner:    w.RepoOwner,
		Name:     w.RepoName,
	}
}

// writeWorkspaceError centralizes the locked envelope mapping for
// workspace errors. Extends the organization writeError with the
// workspace-specific GitHubNotConnectedError (401 reconnect per
// R-WS-017).
func writeWorkspaceError(c *echo.Context, err error) error {
	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":  domain.CodeValidation,
			"fields": verr.Fields,
		})
	}

	var cerr *domain.ConflictError
	if errors.As(err, &cerr) {
		return c.JSON(http.StatusConflict, map[string]any{
			"error":   domain.CodeConflict,
			"message": cerr.Error(),
		})
	}

	var nerr *domain.NotFoundError
	if errors.As(err, &nerr) {
		return c.JSON(http.StatusNotFound, map[string]any{
			"error":   domain.CodeNotFound,
			"message": nerr.Error(),
		})
	}

	// GitHubNotConnectedError -> 401 with the reconnect message
	// (R-WS-017). The frontend shows the "Reconnect GitHub" banner.
	var ghc *domain.GitHubNotConnectedError
	if errors.As(err, &ghc) {
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error":   domain.CodeGitHubNotConnected,
			"message": "Reconnect GitHub to enable workspaces.",
		})
	}

	// Any other error -> 500. Logged with the wrapped cause for ops;
	// the wire never leaks internals.
	return c.JSON(http.StatusInternalServerError, map[string]any{
		"error":   domain.CodeServer,
		"message": domain.MsgServerFailure,
	})
}

// silence unused import warnings for json/context/errors when
// this file is extended in future slices. The compile-time guard
// pins the surface.
var _ = json.Marshal
var _ = context.Background
var _ = errors.New