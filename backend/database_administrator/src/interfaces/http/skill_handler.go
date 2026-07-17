// Package httpiface — skill_handler.go implements the HTTP transport
// for the 2026-07-17-skills-foundational change (PR1d of the chained
// PR set). Wires 7 endpoints:
//
//	POST   /skills
//	GET    /skills
//	GET    /skills/:name
//	PATCH  /skills/:name
//	DELETE /skills/:name
//	GET    /skills/:name/revisions
//	POST   /skills/:name/revisions/:n/restore
//
// Errors are mapped to the locked HTTP envelope via writeSkillError.
// The wire envelope shape matches the project's existing handler
// convention: `{"error":{"code":"...","message":"...","fields":{...}}}`.
// The locked vocabulary is documented in design §3.6 and spec §7
// (validation, not_found, conflict, skill_deleted, server).
//
// Anti-drift gates (engram obs #1959 / design ADR-SK-008):
//   - Every successful response includes current_revision (kills the
//     v{undefined} bug from the prompts feature).
//   - Log-redaction (spec SCN-6.3): the request body and the skill
//     body MUST NOT appear in any log line. The handler logs only the
//     op + name + skill_id + body_len + desc_len.
package httpiface

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// SkillHandler exposes the 7 skill use cases over HTTP.
type SkillHandler struct {
	service *application.SkillService
	logger  *slog.Logger
}

// NewSkillHandler wires a SkillHandler to its service.
func NewSkillHandler(service *application.SkillService, logger *slog.Logger) *SkillHandler {
	if service == nil {
		panic("NewSkillHandler: service must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &SkillHandler{service: service, logger: logger}
}

// RegisterSkillRoutes wires all 7 routes onto the given Echo router.
// The caller is responsible for any auth middleware applied to the
// group (R-SK-007: admin-only on internal network, no extra header).
func (h *SkillHandler) RegisterSkillRoutes(e *echo.Echo) {
	e.POST("/skills", h.Create)
	e.GET("/skills", h.List)
	e.GET("/skills/:name", h.GetBySlug)
	e.PATCH("/skills/:name", h.Update)
	e.DELETE("/skills/:name", h.Delete)
	e.GET("/skills/:name/revisions", h.ListRevisions)
	e.POST("/skills/:name/revisions/:n/restore", h.Restore)
}

// ---------------------------------------------------------------------------
// Wire types (locked by spec §8 — frontend types MUST match).
// ---------------------------------------------------------------------------

// createSkillRequest is the POST /skills body shape.
type createSkillRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Body        string `json:"body"`
}

// skillDetailResponse is the canonical single-skill response shape.
// current_revision is the anti-drift gate (ADR-SK-008 — the field
// MUST be present and populated so the frontend never renders
// v{undefined}).
type skillDetailResponse struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Body            string `json:"body"`
	CurrentRevision int    `json:"current_revision"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
}

// skillListResponse is the locked list shape (spec §8).
type skillListResponse struct {
	Skills []skillListItem `json:"skills"`
}

// skillListItem is the trimmed per-row list shape.
type skillListItem struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	CurrentRevision int    `json:"current_revision"`
	UpdatedAt       string `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// Create (POST /skills) — anti-drift gate emits current_revision.
// ---------------------------------------------------------------------------

// Create handles POST /skills. Decodes the JSON body, calls
// SkillService.Create (which runs the validator chain internally),
// then emits 201 with current_revision=1.
func (h *SkillHandler) Create(c *echo.Context) error {
	var req createSkillRequest
	if err := json.NewDecoder(c.Request().Body).Decode(&req); err != nil {
		return writeSkillError(c, h.logger, http.StatusBadRequest, domain.CodeValidation, "request body must be JSON with name, description, body")
	}
	sk, _, err := h.service.Create(c.Request().Context(), domain.CreateSkillInput{
		Name:        req.Name,
		Description: req.Description,
		Body:        req.Body,
	})
	if err != nil {
		return writeSkillErrorFromErr(c, h.logger, "create", req.Name, err)
	}
	h.logger.InfoContext(c.Request().Context(), "skill created",
		slog.String("op", "create"),
		slog.String("name", sk.Name),
		slog.Int64("skill_id", sk.ID),
		slog.Int("body_len", len([]rune(sk.Body))),
		slog.Int("desc_len", len([]rune(sk.Description))),
	)
	return c.JSON(http.StatusCreated, skillDetailResponse{
		Name:            sk.Name,
		Description:     sk.Description,
		Body:            sk.Body,
		CurrentRevision: 1,
		CreatedAt:       sk.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:       sk.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// ---------------------------------------------------------------------------
// Stub methods (route registration compiles; bodies land in
// subsequent tasks per strict TDD).
// ---------------------------------------------------------------------------

// List handles GET /skills — implemented in task 4.5.
func (h *SkillHandler) List(c *echo.Context) error {
	return writeSkillError(c, h.logger, http.StatusInternalServerError, domain.CodeServer, "not implemented")
}

// GetBySlug handles GET /skills/:name — implemented in task 4.6.
func (h *SkillHandler) GetBySlug(c *echo.Context) error {
	return writeSkillError(c, h.logger, http.StatusInternalServerError, domain.CodeServer, "not implemented")
}

// Update handles PATCH /skills/:name — implemented in task 4.7.
func (h *SkillHandler) Update(c *echo.Context) error {
	return writeSkillError(c, h.logger, http.StatusInternalServerError, domain.CodeServer, "not implemented")
}

// Delete handles DELETE /skills/:name — implemented in task 4.8.
func (h *SkillHandler) Delete(c *echo.Context) error {
	return writeSkillError(c, h.logger, http.StatusInternalServerError, domain.CodeServer, "not implemented")
}

// ListRevisions handles GET /skills/:name/revisions — implemented in task 4.9.
func (h *SkillHandler) ListRevisions(c *echo.Context) error {
	return writeSkillError(c, h.logger, http.StatusInternalServerError, domain.CodeServer, "not implemented")
}

// Restore handles POST /skills/:name/revisions/:n/restore — implemented in task 4.9.
func (h *SkillHandler) Restore(c *echo.Context) error {
	return writeSkillError(c, h.logger, http.StatusInternalServerError, domain.CodeServer, "not implemented")
}

// ---------------------------------------------------------------------------
// Wire types (continued).
// ---------------------------------------------------------------------------

// skillRevisionResponse is the canonical single-revision response
// shape (used by GET /skills/:name/revisions).
type skillRevisionResponse struct {
	RevisionNumber int     `json:"revision_number"`
	Description    string  `json:"description"`
	ChangeNote     *string `json:"change_note,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

// skillRevisionsListResponse is the locked list shape (spec §8).
type skillRevisionsListResponse struct {
	Skill     string                  `json:"skill"`
	Revisions []skillRevisionResponse `json:"revisions"`
}

// ---------------------------------------------------------------------------
// Error helpers (locked envelope shape — spec §7 / R-SK-006).
// ---------------------------------------------------------------------------

// writeSkillError emits a locked-envelope error response and logs
// the operation context (NEVER the request body or skill body).
func writeSkillError(c *echo.Context, logger *slog.Logger, status int, code, message string) error {
	if logger != nil {
		logger.InfoContext(c.Request().Context(), "skill request rejected",
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

// writeSkillErrorFromErr maps a domain error to the locked HTTP
// envelope. Returns 410 only for *SkillGoneError; everything else
// uses the existing type-based mapping (validation/conflict/not_found).
func writeSkillErrorFromErr(c *echo.Context, logger *slog.Logger, op, name string, err error) error {
	if err == nil {
		return nil
	}
	// Per S-SK-029 the log line must NOT contain the request body or
	// the skill body. Name is safe; we log it.
	if logger != nil {
		logger.InfoContext(c.Request().Context(), "skill request failed",
			slog.String("op", op),
			slog.String("name", name),
			slog.String("err", err.Error()),
		)
	}

	// Special case: *SkillGoneError → 410 with the skill_deleted code.
	if _, ok := domain.AsSkillDeleted(err); ok {
		return writeSkillError(c, nil, http.StatusGone, domain.CodeSkillDeleted, domain.MsgSkillDeleted)
	}

	var verr *domain.ValidationError
	if errors.As(err, &verr) {
		msg := "One or more fields failed validation."
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
		return writeSkillError(c, nil, http.StatusConflict, domain.CodeConflict, domain.MsgSkillConflict)
	}

	var nerr *domain.NotFoundError
	if errors.As(err, &nerr) {
		msg := domain.MsgSkillNotFound
		if nerr.Resource == "skill_revision" {
			msg = domain.MsgSkillRevisionNotFound
		}
		return writeSkillError(c, nil, http.StatusNotFound, domain.CodeNotFound, msg)
	}

	// Fallback: internal error.
	return writeSkillError(c, nil, http.StatusInternalServerError, domain.CodeServer, "Something went wrong. Please try again.")
}

// keep errors import live even when intermediate helper commits
// haven't yet added errors.As uses.
var _ = errors.As
