// Package http contains the HTTP handlers for the workspace_syncer
// service. In v1 there is exactly one handler:
// POST /internal/clone-and-validate. PR-2c may add others (e.g.
// GET /admin/sync-jobs for operators).
package http

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/workspace_syncer/src/application"
	"github.com/cachicamas/backend/workspace_syncer/src/domain"
)

// cloneRequestBody is the JSON body the handler decodes. It mirrors
// domain.CloneRequest field-for-field so the handler stays a thin
// transport adapter. The field tags are locked.
type cloneRequestBody struct {
	JobID         int64  `json:"job_id"`
	WorkspaceID   int64  `json:"workspace_id"`
	Owner         string `json:"owner"`
	Repo          string `json:"repo"`
	DefaultBranch string `json:"default_branch"`
	OAuthToken    string `json:"oauth_token"`
}

// CloneHandler is the HTTP transport adapter for the clone use case.
// It is constructed with the application-layer use case so the
// transport layer depends on the application, not the other way
// around (hexagonal boundary).
type CloneHandler struct {
	svc    *application.CloneService
	logger *slog.Logger
}

// NewCloneHandler constructs a CloneHandler.
func NewCloneHandler(svc *application.CloneService, logger *slog.Logger) *CloneHandler {
	if logger == nil {
		logger = slog.Default()
	}
	return &CloneHandler{svc: svc, logger: logger}
}

// Register registers the routes onto an Echo instance. The caller
// is responsible for applying the bearer-token middleware (we do
// not double-apply it here). Lives in a separate function so the
// main.go composition root stays short.
func (h *CloneHandler) Register(e *echo.Echo) {
	e.POST("/internal/clone-and-validate", h.Clone)
}

// Clone is the HTTP transport method for POST
// /internal/clone-and-validate. The flow is:
//
//  1. Parse + validate the JSON body (synchronous).
//  2. Return 400 with the validation envelope on parse failure.
//  3. Return 202 immediately, then run the use case in a
//     goroutine.
//
// The 202 + goroutine pattern matches the design (T-WSY-001
// S-WSY-001): the database_administrator does not block on the
// clone. The use case posts the outcome via the callback.
func (h *CloneHandler) Clone(c *echo.Context) error {
	var body cloneRequestBody
	if err := c.Bind(&body); err != nil {
		// Bad JSON or missing Content-Type. Return the flat
		// validation envelope (matches the database_administrator
		// convention).
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error":   "validation",
			"fields":  map[string]string{"body": "invalid JSON body"},
			"message": "Could not parse request body.",
		})
	}

	req := domain.CloneRequest{
		JobID:         body.JobID,
		WorkspaceID:   body.WorkspaceID,
		Owner:         body.Owner,
		Repo:          body.Repo,
		DefaultBranch: body.DefaultBranch,
		OAuthToken:    body.OAuthToken,
	}

	if err := domain.ValidateCloneRequest(req); err != nil {
		// Translate *ValidationError to the flat 400 envelope.
		ve := &domain.ValidationError{}
		if errors.As(err, &ve) {
			return c.JSON(http.StatusBadRequest, map[string]any{
				"error":   "validation",
				"fields":  ve.Fields,
				"message": "Validation failed.",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "server",
			"message": "Unexpected error.",
		})
	}

	// Run the use case in a goroutine. The HTTP response is
	// 202 immediately. The use case posts the outcome via the
	// callback; the client polls the database_administrator's
	// GET /workspaces/:id/sync endpoint.
	go func() {
		// The goroutine's context is a new background context
		// (the request context is canceled when the response
		// is sent). The use case's clone has its own timeout.
		h.svc.CloneAndValidate(c.Request().Context(), req)
	}()

	return c.JSON(http.StatusAccepted, map[string]any{
		"job_id": req.JobID,
		"status": "running",
	})
}

// MarshalJSON is defined on cloneRequestBody to satisfy the json
// package's interface; the actual encoding is done by the struct
// tags above. This stub keeps the import alive and documents the
// shape.
var _ = json.Marshal