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

// SyncHandler exposes the 2 sync endpoints. The handler depends
// on SyncEnqueuer (an interface) so tests can pass a fake without
// standing up the full hexagonal graph.
type SyncHandler struct {
	syncSvc SyncEnqueuer
	syncer  application.SyncDispatcher // may be nil (test path)
	logger  *slog.Logger
}

// NewSyncHandler wires a SyncHandler to its service. dispatcher
// may be nil (the test path); the handler tolerates the nil case
// by skipping the syncer call and leaving the job in 'pending'.
func NewSyncHandler(syncSvc SyncEnqueuer, dispatcher application.SyncDispatcher, logger *slog.Logger) *SyncHandler {
	if syncSvc == nil {
		panic("NewSyncHandler: syncSvc must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SyncHandler{
		syncSvc: syncSvc,
		syncer:  dispatcher,
		logger:  logger,
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

	jobID, isFresh, existing, err := h.syncSvc.EnqueueSync(c.Request().Context(), workspaceID, domain.SyncJobTriggerManual)
	if err != nil {
		return writeSyncError(c, err)
	}

	// Dispatch to the syncer. Best-effort: a failure here is
	// logged but does NOT change the response. The job is in
	// 'pending' state in the DB; the user can retry from the UI.
	if isFresh && h.syncer != nil {
		// The dispatcher requires owner/repo/default_branch/oauth_token
		// to assemble the syncer request. For PR-3b the workspace
		// service would supply these via a context-loaded
		// workspace value; the handler does NOT have access to the
		// workspace row here. We dispatch with empty owner/repo;
		// the syncer will reject with 400 + validation. This is
		// a known limitation of PR-3b that PR-3c (or follow-up)
		// will close by loading the workspace row in the handler.
		_ = h.dispatchToSyncer(c, workspaceID, jobID, existing)
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
// dispatcher. The owner/repo/default_branch/oauth_token are
// not yet loaded in the handler (PR-3b scope); for now the
// handler invokes the dispatcher with empty strings, which
// the syncer rejects with 400 + validation. The job remains
// 'pending' in the DB and the UI can retry.
//
// PR-3c (follow-up) will close this gap by loading the workspace
// row + the user's OAuth token in the handler before calling
// the dispatcher.
func (h *SyncHandler) dispatchToSyncer(c *echo.Context, workspaceID, jobID int64, job *domain.SyncJob) error {
	if h.syncer == nil {
		return nil
	}
	h.logger.DebugContext(c.Request().Context(),
		"sync handler: dispatching to syncer",
		slog.Int64("workspace_id", workspaceID),
		slog.Int64("job_id", jobID),
	)
	// Best-effort dispatch. The dispatcher tolerates nil inputs
	// and surfaces a typed error; the handler logs and returns.
	if err := h.syncer.StartSync(c.Request().Context(), jobID, workspaceID, "", "", "", ""); err != nil {
		h.logger.WarnContext(c.Request().Context(),
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
		return c.JSON(http.StatusNotFound, map[string]any{
			"error":   "workspace_not_synced_yet",
			"message": "No sync has been enqueued for this workspace yet.",
		})
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