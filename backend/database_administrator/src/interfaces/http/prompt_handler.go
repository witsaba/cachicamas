// Package httpiface — prompt_handler.go implements the HTTP transport
// for the 2026-07-15-prompt-storage-table change (PR 4 of 4). Wires
// 7 endpoints:
//
//	POST   /prompts
//	GET    /prompts
//	GET    /prompts/:slug
//	PATCH  /prompts/:slug
//	DELETE /prompts/:slug
//	GET    /prompts/:slug/revisions
//	POST   /prompts/:slug/revisions/:n/restore
//
// Errors are mapped to the locked HTTP envelope via writePromptError.
// The wire envelope shape matches the project's existing handler
// convention (see workspace_handler.go): `{"error":{"code":"...","message":"..."}}`.
// The error CODE is generic (`validation`, `conflict`, `not_found`);
// the only feature-specific code is `prompt_deleted` for HTTP 410
// (no existing type covers 410).
//
// Log-redaction (spec S-PR-X3): the request body and the prompt body
// MUST NOT appear in any log line. The handler logs only the slug,
// the prompt ID, and the operation. The TestPromptHandler_NoPIITokenInLogs
// test in prompt_handler_test.go asserts this contract.
package httpiface

import (
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

// PromptHandler exposes the 7 prompt use cases over HTTP.
type PromptHandler struct {
	service *application.PromptService
	logger  *slog.Logger
}

// NewPromptHandler wires a PromptHandler to its service.
func NewPromptHandler(service *application.PromptService, logger *slog.Logger) *PromptHandler {
	if service == nil {
		panic("NewPromptHandler: service must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &PromptHandler{service: service, logger: logger}
}

// ---------------------------------------------------------------------------
// Wire types
// ---------------------------------------------------------------------------

type createPromptRequest struct {
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

type updatePromptRequest struct {
	Description *string `json:"description,omitempty"`
	Body        *string `json:"body,omitempty"`
}

type promptResponse struct {
	ID          int64  `json:"id"`
	Description string `json:"description"`
	Slug        string `json:"slug"`
	Body        string `json:"body"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type promptRevisionResponse struct {
	ID             int64   `json:"id"`
	PromptID       int64   `json:"prompt_id"`
	RevisionNumber int     `json:"revision_number"`
	Description    string  `json:"description"`
	Body           string  `json:"body"`
	ChangeNote     *string `json:"change_note,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Handler methods
// ---------------------------------------------------------------------------

// Create handles POST /prompts.
func (h *PromptHandler) Create(c *echo.Context) error {
	var req createPromptRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writePromptError(c, h.logger, http.StatusBadRequest, domain.CodeValidation, "request body must be JSON with slug, description, body")
	}
	p, rev, err := h.service.Create(c.Request().Context(), domain.CreatePromptInput{
		Slug:        req.Slug,
		Description: req.Description,
		Body:        req.Body,
	})
	if err != nil {
		return writePromptErrorFromErr(c, h.logger, "create", req.Slug, err)
	}
	h.logger.InfoContext(c.Request().Context(), "prompt created",
		slog.String("slug", p.Slug),
		slog.Int64("prompt_id", p.ID),
		slog.Int("revision", rev.RevisionNumber),
	)
	return c.JSON(http.StatusCreated, promptResponse{
		ID:          p.ID,
		Description: p.Description,
		Slug:        p.Slug,
		Body:        p.Body,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// List handles GET /prompts?limit=50&offset=0.
func (h *PromptHandler) List(c *echo.Context) error {
	limit := parseIntDefault(c.QueryParam("limit"), 0)
	offset := parseIntDefault(c.QueryParam("offset"), 0)
	prompts, err := h.service.List(c.Request().Context(), limit, offset)
	if err != nil {
		return writePromptErrorFromErr(c, h.logger, "list", "", err)
	}
	out := make([]promptResponse, 0, len(prompts))
	for _, p := range prompts {
		out = append(out, promptResponse{
			ID:          p.ID,
			Description: p.Description,
			Slug:        p.Slug,
			Body:        p.Body,
			CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return c.JSON(http.StatusOK, out)
}

// GetBySlug handles GET /prompts/:slug.
func (h *PromptHandler) GetBySlug(c *echo.Context) error {
	slug := c.Param("slug")
	p, err := h.service.GetBySlug(c.Request().Context(), slug)
	if err != nil {
		return writePromptErrorFromErr(c, h.logger, "get-by-slug", slug, err)
	}
	return c.JSON(http.StatusOK, promptResponse{
		ID:          p.ID,
		Description: p.Description,
		Slug:        p.Slug,
		Body:        p.Body,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Update handles PATCH /prompts/:slug.
func (h *PromptHandler) Update(c *echo.Context) error {
	slug := c.Param("slug")
	var req updatePromptRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writePromptError(c, h.logger, http.StatusBadRequest, domain.CodeValidation, "request body must be JSON")
	}
	p, rev, err := h.service.Update(c.Request().Context(), slug, domain.UpdatePromptInput{
		Description: req.Description,
		Body:        req.Body,
	})
	if err != nil {
		return writePromptErrorFromErr(c, h.logger, "update", slug, err)
	}
	h.logger.InfoContext(c.Request().Context(), "prompt updated",
		slog.String("slug", p.Slug),
		slog.Int64("prompt_id", p.ID),
		slog.Int("revision", rev.RevisionNumber),
	)
	return c.JSON(http.StatusOK, promptResponse{
		ID:          p.ID,
		Description: p.Description,
		Slug:        p.Slug,
		Body:        p.Body,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// Delete handles DELETE /prompts/:slug.
func (h *PromptHandler) Delete(c *echo.Context) error {
	slug := c.Param("slug")
	if err := h.service.SoftDelete(c.Request().Context(), slug); err != nil {
		return writePromptErrorFromErr(c, h.logger, "delete", slug, err)
	}
	h.logger.InfoContext(c.Request().Context(), "prompt soft-deleted", slog.String("slug", slug))
	return c.NoContent(http.StatusNoContent)
}

// ListRevisions handles GET /prompts/:slug/revisions.
func (h *PromptHandler) ListRevisions(c *echo.Context) error {
	slug := c.Param("slug")
	revs, err := h.service.ListRevisions(c.Request().Context(), slug)
	if err != nil {
		return writePromptErrorFromErr(c, h.logger, "list-revisions", slug, err)
	}
	out := make([]promptRevisionResponse, 0, len(revs))
	for _, r := range revs {
		out = append(out, promptRevisionResponse{
			ID:             r.ID,
			PromptID:       r.PromptID,
			RevisionNumber: r.RevisionNumber,
			Description:    r.Description,
			Body:           r.Body,
			ChangeNote:     r.ChangeNote,
			CreatedAt:      r.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return c.JSON(http.StatusOK, out)
}

// Restore handles POST /prompts/:slug/revisions/:n/restore.
func (h *PromptHandler) Restore(c *echo.Context) error {
	slug := c.Param("slug")
	n, err := strconv.Atoi(c.Param("n"))
	if err != nil || n < 1 {
		return writePromptError(c, h.logger, http.StatusBadRequest, domain.CodeValidation, "revision number must be a positive integer")
	}
	p, rev, err := h.service.Restore(c.Request().Context(), slug, n)
	if err != nil {
		return writePromptErrorFromErr(c, h.logger, "restore", slug, err)
	}
	h.logger.InfoContext(c.Request().Context(), "prompt restored",
		slog.String("slug", p.Slug),
		slog.Int64("prompt_id", p.ID),
		slog.Int("from_revision", n),
		slog.Int("new_revision", rev.RevisionNumber),
	)
	return c.JSON(http.StatusOK, promptResponse{
		ID:          p.ID,
		Description: p.Description,
		Slug:        p.Slug,
		Body:        p.Body,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ---------------------------------------------------------------------------
// Error helpers
// ---------------------------------------------------------------------------

// writePromptError emits a locked-envelope error response and logs
// the operation context (NEVER the request body or prompt body).
func writePromptError(c *echo.Context, logger *slog.Logger, status int, code, message string) error {
	if logger != nil {
		logger.InfoContext(c.Request().Context(), "prompt request rejected",
			slog.Int("status", status),
			slog.String("code", code),
		)
	}
	return c.JSON(status, map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	})
}

// writePromptErrorFromErr maps a domain error to the locked HTTP
// envelope. Returns 410 only for *GoneError; everything else uses
// the existing type-based mapping (validation/conflict/not_found).
func writePromptErrorFromErr(c *echo.Context, logger *slog.Logger, op, slug string, err error) error {
	if err == nil {
		return nil
	}
	// Per S-PR-X3 the log line must NOT contain the request body or
	// the prompt body. Slug is safe; we log it.
	if logger != nil {
		logger.InfoContext(c.Request().Context(), "prompt request failed",
			slog.String("op", op),
			slog.String("slug", slug),
			slog.String("err", err.Error()),
		)
	}

	// Special case: *GoneError -> 410 with the prompt_deleted code.
	if _, ok := domain.AsPromptDeleted(err); ok {
		return writePromptError(c, nil, http.StatusGone, domain.CodePromptDeleted, domain.MsgPromptDeleted)
	}

	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		// Use the first field's message as the top-level message; the
		// full Fields map is in the data field for richer clients.
		msg := domain.MsgPromptValidationFailed
		for _, m := range verr.Fields {
			msg = m
			break
		}
		return c.JSON(http.StatusBadRequest, map[string]any{
			"error": map[string]any{
				"code":    domain.CodeValidation,
				"message": msg,
				"fields":  verr.Fields,
			},
		})
	}

	var cerr *domain.ConflictError
	if errors.As(err, &cerr) {
		return writePromptError(c, nil, http.StatusConflict, domain.CodeConflict, domain.MsgPromptSlugTaken)
	}

	var nerr *domain.NotFoundError
	if errors.As(err, &nerr) {
		// Distinguish "revision not found" from "prompt not found"
		// by the resource name. The repo sets Resource to
		// promptTableName or promptRevisionTableName.
		msg := domain.MsgPromptNotFound
		code := domain.CodeNotFound
		if nerr.Resource == "prompt_revision" {
			msg = domain.MsgPromptRevisionNotFound
		}
		return writePromptError(c, nil, http.StatusNotFound, code, msg)
	}

	// Fallback: internal error.
	return writePromptError(c, nil, http.StatusInternalServerError, domain.CodeServer, "Something went wrong. Please try again.")
}

// RegisterPromptRoutes wires all 7 routes onto the given Echo group.
// The caller is responsible for any auth middleware applied to the
// group (Q-D locked: admin-only, no extra header required for v1).
func (h *PromptHandler) RegisterPromptRoutes(e *echo.Echo) {
	e.POST("/prompts", h.Create)
	e.GET("/prompts", h.List)
	e.GET("/prompts/:slug", h.GetBySlug)
	e.PATCH("/prompts/:slug", h.Update)
	e.DELETE("/prompts/:slug", h.Delete)
	e.GET("/prompts/:slug/revisions", h.ListRevisions)
	e.POST("/prompts/:slug/revisions/:n/restore", h.Restore)
}

// ---------------------------------------------------------------------------
// Small helpers
// ---------------------------------------------------------------------------

func parseIntDefault(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

// ensure imports are used (errors, fmt) — keep them explicit so the
// file compiles even when the helpers above are simplified later.
var (
	_ = errors.As
	_ = fmt.Sprintf
)
