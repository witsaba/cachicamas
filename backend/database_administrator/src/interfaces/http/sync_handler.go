// Package httpiface — sync_handler.go implements the HTTP
// transport for the 2026-07-08-workspace-sync-clone PR-3b sync
// flow. Wires 2 endpoints on the workspace-authenticated
// sub-group:
//
//	POST /workspaces/:id/sync  -> 202 (fresh) or 200 (single-flight hit)
//	GET  /workspaces/:id/sync  -> 200 (job snapshot) or 404 (no jobs yet)
//
// The callback endpoint (POST /internal/sync-callback) lives in
// internal_callback_handler.go because it has a different auth
// posture (HMAC + docker network, NOT the user JWE cookie).
//
// Wire shape (locked, see spec R-WS-019 S-WS-190..200):
//   - POST /workspaces/:id/sync body is empty (the workspace id
//     comes from the path).
//   - Response is {job_id, status, workspace_id} where status
//     is "pending" on a fresh enqueue or "running"/"done"/"failed"
//     on a single-flight hit (the handler looks up the existing
//     job so the caller can poll its actual state).
//   - GET /workspaces/:id/sync returns the latest sync_job row
//     or 404 with code=workspace_not_synced_yet if no jobs exist.
package httpiface

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
	wsclient "github.com/cachicamas/backend/database_administrator/src/infrastructure/workspacesyncer"
)

// SyncEnqueuer is the slice of *application.SyncService the
// handler actually consumes. Defining it here keeps the handler
// test-friendly and documents the dependency. Production code
// passes *application.SyncService (which satisfies the slice);
// tests pass any implementation including a stub.
type SyncEnqueuer interface {
	EnqueueSync(ctx context.Context, workspaceID int64, triggeredBy string) (int64, bool, *domain.SyncJob, error)
	GetLatestSyncJob(ctx context.Context, workspaceID int64) (*domain.SyncJob, error)
}

// WorkspaceRowLoader is the narrow contract PR-3c uses to load
// the workspace row in the handler. The workspace row carries
// the repo identity (RepoOwner, RepoName) the syncer needs.
// Production: *postgres.WorkspaceRepo. Tests: a fake.
//
// 2026-08-02-security-vulnerability-remediation (H-1): the
// SelectByID signature now takes orgID so the SQL cannot return
// a workspace in a different tenant than the one the handler
// knows the request belongs to.
type WorkspaceRowLoader interface {
	SelectByID(ctx context.Context, orgID, id int64) (*domain.Workspace, error)
}

// SyncHandler exposes the 2 sync endpoints. The handler depends
// on SyncEnqueuer + WorkspaceRowLoader (interfaces) so tests can
// pass fakes without standing up the full hexagonal graph.
type SyncHandler struct {
	syncSvc      SyncEnqueuer
	workspaces   WorkspaceRowLoader
	tokenFetcher TokenFetcher
	syncer       application.SyncDispatcher // may be nil (test path)
	logger       *slog.Logger
}

// NewSyncHandler wires a SyncHandler to its service. dispatcher
// may be nil (the test path); the handler tolerates the nil case
// by skipping the syncer call and leaving the job in 'pending'.
func NewSyncHandler(syncSvc SyncEnqueuer, workspaces WorkspaceRowLoader, tokenFetcher TokenFetcher, dispatcher application.SyncDispatcher, logger *slog.Logger) *SyncHandler {
	if syncSvc == nil {
		panic("NewSyncHandler: syncSvc must not be nil")
	}
	if workspaces == nil {
		panic("NewSyncHandler: workspaces must not be nil")
	}
	if tokenFetcher == nil {
		panic("NewSyncHandler: tokenFetcher must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncHandler{
		syncSvc:      syncSvc,
		workspaces:   workspaces,
		tokenFetcher: tokenFetcher,
		syncer:       dispatcher,
		logger:       logger,
	}
}

// RegisterSyncRoutes wires the 2 sync endpoints on the supplied
// Echo sub-group (typically the auth-protected workspace group
// produced by e.Group("", authChain...)). The caller is
// responsible for applying the auth chain via the group middleware;
// the handler does not double-apply.
func RegisterSyncRoutes(g *echo.Group, h *SyncHandler) {
	g.POST("/workspaces/:id/sync", h.Post)
	g.GET("/workspaces/:id/sync", h.Get)
}

// syncResponse is the wire body for both POST and GET
// /workspaces/:id/sync. The shape is locked (frontend
// WorkspaceSyncCard).
type syncResponse struct {
	JobID          int64      `json:"job_id"`
	WorkspaceID    int64      `json:"workspace_id"`
	Status         string     `json:"status"`
	TriggeredBy    string     `json:"triggered_by"`
	StartedAt      *time.Time `json:"started_at"`
	FinishedAt     *time.Time `json:"finished_at"`
	CommitSHAAfter *string    `json:"commit_sha_after"`
	ErrorMessage   *string    `json:"error_message"`
	ErrorCode      *string    `json:"error_code"`
	Attempts       int        `json:"attempts"`
	CreatedAt      time.Time  `json:"created_at"`
}

func toSyncResponse(j *domain.SyncJob) syncResponse {
	// nil-tolerant: returns a zero-value response with
	// JobID=0, Status="" so the frontend can distinguish
	// "no sync yet" from a missing workspace. Used by both
	// the GET endpoint (200 with job: null) and the SSE
	// stream (one null event then close).
	if j == nil {
		return syncResponse{}
	}
	return syncResponse{
		JobID:          j.ID,
		WorkspaceID:    j.WorkspaceID,
		Status:         j.Status,
		TriggeredBy:    j.TriggeredBy,
		StartedAt:      j.StartedAt,
		FinishedAt:     j.FinishedAt,
		CommitSHAAfter: j.CommitSHAAfter,
		ErrorMessage:   j.ErrorMessage,
		ErrorCode:      j.ErrorCode,
		Attempts:       j.Attempts,
		CreatedAt:      j.CreatedAt.UTC(),
	}
}

// Post handles POST /workspaces/:id/sync.
//
// Status codes (locked):
//   - 202 Accepted on a fresh enqueue (the syncer has been
//     notified; the caller can poll GET /workspaces/:id/sync).
//   - 200 OK on a single-flight hit (the existing in-flight
//     job_id is returned so the caller can resume polling).
//   - 422 Unprocessable Entity on validation failure
//     (workspace_id malformed, etc.).
//   - 500 Internal Server Error on service errors.
func (h *SyncHandler) Post(c *echo.Context) error {
	workspaceID, err := parsePathInt64(c, "id")
	if err != nil {
		return writeSyncError(c, &domain.ValidationError{
			Fields: map[string]string{"id": "Invalid id."},
		})
	}
	if _, ok := identityFromContext(c); !ok {
		return writeSyncError(c, &domain.ValidationError{
			Fields: map[string]string{"auth": "Authentication required."},
		})
	}
	identity, _ := identityFromContext(c)

	jobID, isFresh, existing, err := h.syncSvc.EnqueueSync(c.Request().Context(), workspaceID, domain.SyncJobTriggerManual)
	if err != nil {
		return writeSyncError(c, err)
	}

	// Dispatch to the syncer. Best-effort: a failure here is
	// logged but does NOT change the response. The job is in
	// 'pending' state in the DB; the user can retry from the UI.
	if isFresh && h.syncer != nil {
		// PR-3c: load the workspace row + the user's OAuth token
		// before dispatching. The syncer's ValidateCloneRequest
		// requires owner/repo/default_branch/oauth_token — sending
		// empty strings (the PR-3b behavior) results in a 400
		// and the job stays in 'pending' forever. The handler now
		// loads the real values so the syncer can actually run.
		_ = h.dispatchToSyncer(c, identity, workspaceID, jobID, existing)
	}

	if isFresh {
		return c.JSON(http.StatusAccepted, toSyncResponse(existing))
	}
	// Single-flight hit: the caller is racing another request.
	// Return 200 with the existing job so the caller can resume
	// polling. The handler deliberately does NOT return 409 here
	// because the user-experience path is "your previous click
	// already started a sync; here's its id".
	return c.JSON(http.StatusOK, toSyncResponse(existing))
}

// dispatchToSyncer invokes the syncer via the configured
// dispatcher. Loads the workspace row (for owner/repo) + the
// user's OAuth token (from the auth chain's identity) and
// passes them to the syncer. PR-3b was a placeholder that sent
// empty strings; PR-3c is the wired version.
//
// defaultBranch resolution: the workspace row's DefaultBranch is
// denormalized from the syncer's callback on the first done.
// On the FIRST sync the column is NULL (the syncer hasn't reported
// back yet), so we hardcode "main" — GitHub's default for every
// new repo since Oct 2020. A future slice may call the GitHub
// REST GET /repos/{owner}/{repo} to fetch the actual default,
// but that adds a network call to every dispatch; the workspace
// row denormalization is the source of truth on subsequent syncs.
func (h *SyncHandler) dispatchToSyncer(c *echo.Context, identity *domain.Identity, workspaceID, jobID int64, job *domain.SyncJob) error {
	if h.syncer == nil {
		return nil
	}
	ctx := c.Request().Context()

	// 1. Load the workspace row, scoped to the same tenant the
	// authenticated identity belongs to (security H-1: IDOR).
	orgID := singleTenantOrganizationID(c)
	ws, err := h.workspaces.SelectByID(ctx, orgID, workspaceID)
	if err != nil {
		h.logger.WarnContext(ctx,
			"sync handler: workspace lookup failed; job remains pending",
			slog.Int64("workspace_id", workspaceID),
			slog.Int64("job_id", jobID),
			slog.String("error", err.Error()),
		)
		return err
	}
	if ws == nil {
		h.logger.WarnContext(ctx,
			"sync handler: workspace not found; job remains pending",
			slog.Int64("workspace_id", workspaceID),
			slog.Int64("job_id", jobID),
		)
		return errors.New("workspace not found")
	}

	// 2. Load the user's OAuth token. The identity's
	// ProviderAccountID is the GitHub user id; Provider is "github".
	token, err := h.tokenFetcher.AccessTokenForIdentity(ctx, identity.Provider, identity.ProviderAccountID)
	if err != nil || token == "" {
		h.logger.WarnContext(ctx,
			"sync handler: OAuth token not found (user must reconnect); job remains pending",
			slog.Int64("workspace_id", workspaceID),
			slog.Int64("job_id", jobID),
			slog.String("error", errString(err)),
		)
		return errors.New("OAuth token not found")
	}

	// 3. Resolve default_branch. Use the denormalized value if
	// available; fall back to "main" on the first sync.
	defaultBranch := ""
	if ws.DefaultBranch != nil {
		defaultBranch = *ws.DefaultBranch
	}
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	h.logger.DebugContext(ctx,
		"sync handler: dispatching to syncer",
		slog.Int64("workspace_id", workspaceID),
		slog.Int64("job_id", jobID),
		slog.String("owner", ws.RepoOwner),
		slog.String("repo", ws.RepoName),
		slog.String("default_branch", defaultBranch),
	)

	// 4. Dispatch.
	if err := h.syncer.StartSync(ctx, jobID, workspaceID, ws.RepoOwner, ws.RepoName, defaultBranch, token); err != nil {
		h.logger.WarnContext(ctx,
			"sync handler: dispatcher.StartSync returned error; job remains pending",
			slog.Int64("workspace_id", workspaceID),
			slog.Int64("job_id", jobID),
			slog.String("error", err.Error()),
		)
		_ = job
		return err
	}
	return nil
}

// errString returns err.Error() or "" if err is nil. Tiny helper
// to keep the warn logs single-line.
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

// Get handles GET /workspaces/:id/sync. Returns the most recent
// sync_job for the workspace, or 404 with code=workspace_not_synced_yet
// when no job has ever been enqueued.
//
// Status codes (locked):
//   - 200 OK with the syncResponse on a hit
//   - 404 Not Found with code=workspace_not_synced_yet on a miss
//   - 422 Unprocessable Entity on a malformed workspace id
//   - 500 Internal Server Error on a service error
func (h *SyncHandler) Get(c *echo.Context) error {
	workspaceID, err := parsePathInt64(c, "id")
	if err != nil {
		return writeSyncError(c, &domain.ValidationError{
			Fields: map[string]string{"id": "Invalid id."},
		})
	}
	if _, ok := identityFromContext(c); !ok {
		return writeSyncError(c, &domain.ValidationError{
			Fields: map[string]string{"auth": "Authentication required."},
		})
	}

	job, err := h.syncSvc.GetLatestSyncJob(c.Request().Context(), workspaceID)
	if err != nil {
		return writeSyncError(c, err)
	}
	if job == nil {
		// UAT fix 2026-07-08 (clean-rebuild bug): the previous
		// behavior was 404 with code=workspace_not_synced_yet.
		// That was technically correct (no row) but semantically
		// misleading: the workspace DOES exist; only the sync
		// state is empty. The 404 made the frontend think the
		// workspace was missing (a different code path). The
		// correct shape is 200 with a zero-value syncResponse
		// so the card renders the "Sync now" CTA without
		// surfacing an error.
		return c.JSON(http.StatusOK, toSyncResponse(nil))
	}
	return c.JSON(http.StatusOK, toSyncResponse(job))
}

// writeSyncError centralizes the locked envelope mapping for
// the sync endpoints. Mirrors writeWorkspaceError's style with
// extensions for the sync-specific error types.
func writeSyncError(c *echo.Context, err error) error {
	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":  domain.CodeValidation,
			"fields": verr.Fields,
		})
	}

	var sync *domain.ErrSyncAlreadyRunning
	if errors.As(err, &sync) {
		return c.JSON(http.StatusConflict, map[string]any{
			"error":   domain.CodeSyncAlreadyRunning,
			"job_id":  sync.JobID,
			"message": "A sync is already running for this workspace.",
		})
	}

	var perm *domain.ErrInsufficientPermissions
	if errors.As(err, &perm) {
		return c.JSON(http.StatusUnprocessableEntity, map[string]any{
			"error":   domain.CodeSyncInsufficientPermissions,
			"message": "GitHub token does not have push permission for this repository.",
		})
	}

	var expired *domain.ErrTokenExpired
	if errors.As(err, &expired) {
		return c.JSON(http.StatusUnauthorized, map[string]any{
			"error":   domain.CodeSyncTokenExpired,
			"message": "GitHub access token has expired; reconnect required.",
		})
	}

	var nf *domain.NotFoundError
	if errors.As(err, &nf) {
		return c.JSON(http.StatusNotFound, map[string]any{
			"error":   domain.CodeNotFound,
			"message": nf.Error(),
		})
	}

	var cerr *domain.ConflictError
	if errors.As(err, &cerr) {
		return c.JSON(http.StatusConflict, map[string]any{
			"error":   domain.CodeConflict,
			"message": cerr.Error(),
		})
	}

	return c.JSON(http.StatusInternalServerError, map[string]any{
		"error":   domain.CodeServer,
		"message": domain.MsgServerFailure,
	})
}

// wsclientAdapter wraps the workspace_syncer.Client to satisfy
// application.SyncDispatcher. The adapter is constructed in
// main.go (composition root); the application layer depends on
// the interface only.
type wsclientAdapter struct {
	client *wsclient.Client
}

// NewWSClientAdapter returns an application.SyncDispatcher that
// delegates to the supplied workspace_syncer.Client.
func NewWSClientAdapter(client *wsclient.Client) application.SyncDispatcher {
	return &wsclientAdapter{client: client}
}

// StartSync satisfies application.SyncDispatcher.
func (a *wsclientAdapter) StartSync(ctx context.Context, jobID, workspaceID int64, owner, repo, defaultBranch, oauthToken string) error {
	if a == nil || a.client == nil {
		return errors.New("wsclientAdapter: nil client")
	}
	_, err := a.client.StartSync(ctx, wsclient.StartSyncRequest{
		JobID:         jobID,
		WorkspaceID:   workspaceID,
		Owner:         owner,
		Repo:          repo,
		DefaultBranch: defaultBranch,
		OAuthToken:    oauthToken,
	})
	return err
}

// Compile-time interface satisfaction.
var _ application.SyncDispatcher = (*wsclientAdapter)(nil)
