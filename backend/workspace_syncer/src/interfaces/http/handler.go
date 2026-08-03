// Package http contains the HTTP handlers for the workspace_syncer
// service. In v1 there is exactly one handler:
// POST /internal/clone-and-validate. PR-2c may add others (e.g.
// GET /admin/sync-jobs for operators).
package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/workspace_syncer/src/application"
	"github.com/cachicamas/backend/workspace_syncer/src/domain"
)

// asyncJobTimeout is the hard cap on the goroutine that runs the
// clone + worktree probe + callback. The use case's individual
// steps have their own timeouts (clone 90s, probe 30s, callback
// 30s — ~3 min worst case); 5 min gives generous headroom for
// slow repos while preventing a stuck goroutine from leaking.
//
// PR-2b fix: this used to be c.Request().Context(), which is
// canceled when Echo returns the 202 response to the client.
// The clone then aborted immediately with "context canceled"
// and the sync_job row stayed in 'pending' forever. The fix is
// to derive a fresh context from context.Background() so the
// goroutine is decoupled from the request lifecycle.
const asyncJobTimeout = 5 * time.Minute

// maxRequestBodyBytes bounds the size of the clone request body.
// Per spec audit finding M-1 the prior code accepted unbounded
// bodies (a 100 MiB JSON payload would be happily JSON-decoded
// into memory before any validation). 1 MiB is generous for the
// 6-field clone payload (a few hundred bytes in practice) and
// matches the database_administrator's BodyLimit("1M") value.
const maxRequestBodyBytes = 1 << 20

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
//  1. Cap the request body at 1 MiB (http.MaxBytesReader).
//  2. Parse + validate the JSON body (synchronous).
//  3. Return 400 with the validation envelope on parse failure.
//  4. Return 202 immediately, then run the use case in a
//     goroutine.
//
// Per spec audit finding M-1 the request body is bounded by
// http.MaxBytesReader so a 100 MiB POST does not allocate a
// matching buffer in the handler. The resulting decode error
// is mapped to a 400 envelope (a 2 MiB body is rejected before
// JSON parsing begins).
//
// The 202 + goroutine pattern matches the design (T-WSY-001
// S-WSY-001): the database_administrator does not block on the
// clone. The use case posts the outcome via the callback.
func (h *CloneHandler) Clone(c *echo.Context) error {
	// Spec audit M-1: cap the request body at maxRequestBodyBytes
	// BEFORE JSON decode. http.MaxBytesReader replaces r.Body with
	// a reader that returns an error on the next Read past the cap;
	// the JSON decoder surfaces that as ErrBodyTooLarge (or a
	// wrapped form), which we map to a 400 envelope below.
	c.Request().Body = http.MaxBytesReader(c.Response(), c.Request().Body, maxRequestBodyBytes)

	var body cloneRequestBody
	if err := c.Bind(&body); err != nil {
		// Distinguish body-too-large from generic JSON errors.
		// http.MaxBytesReader returns *http.MaxBytesError via the
		// json decoder (the decoder wraps it). If the body
		// exceeds maxRequestBodyBytes, return 400 with a closed-
		// vocab message; otherwise the generic JSON error.
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			return c.JSON(http.StatusBadRequest, map[string]any{
				"error":   "validation",
				"fields":  map[string]string{"body": fmt.Sprintf("body exceeds %d bytes", maxRequestBodyBytes)},
				"message": "Request body too large.",
			})
		}
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
	//
	// PR-2b fix: the goroutine MUST use a fresh context derived
	// from context.Background() (with a 5-min timeout), NOT
	// c.Request().Context(). The request context is canceled
	// when Echo returns the 202 response to the client; the
	// clone then aborts immediately with "context canceled"
	// and the sync_job row stays in 'pending' forever.
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), asyncJobTimeout)
		defer cancel()
		h.svc.CloneAndValidate(ctx, req)
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