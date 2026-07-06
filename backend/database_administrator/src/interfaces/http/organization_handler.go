// Package httpiface exposes HTTP transport adapters (handlers,
// routes) for the database administrator service. This file
// implements the organization handler: POST /organizations, GET
// /organizations, GET /organizations/:id. Errors are mapped to
// the locked HTTP envelope (spec §3.1) via a type switch on the
// domain AppError interface.
package httpiface

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// OrganizationHandler exposes the organization use cases over
// HTTP. It depends on the application service (NOT on the
// pgx-backed adapter) so the hexagonal boundary holds.
type OrganizationHandler struct {
	service *application.OrganizationService
}

// NewOrganizationHandler wires an OrganizationHandler to its
// application service. Mirrors NewHealthHandler.
func NewOrganizationHandler(service *application.OrganizationService) *OrganizationHandler {
	return &OrganizationHandler{service: service}
}

// RegisterOrganizationRoutes wires the two organization-adjacent
// routes on the given Echo instance. The /setup-state endpoint
// powers the ownboarding gate (R-OW-005); POST /organizations is
// the ownboarding submit endpoint. The old GET /organizations
// (list) and GET /organizations/:id (get-by-id) routes were
// removed in the 2026-07-06 ownboarding change.
func RegisterOrganizationRoutes(e *echo.Echo, svc *application.OrganizationService) {
	h := NewOrganizationHandler(svc)
	e.POST("/organizations", h.Create)
	e.GET("/setup-state", h.SetupState)
}

// Create handles POST /organizations. Accepts JSON or
// form-encoded bodies (locked #3). On success returns 201 with
// the full OrganizationResponse and a Location header.
func (h *OrganizationHandler) Create(c *echo.Context) error {
	in, err := parseCreateInput(c)
	if err != nil {
		return writeError(c, err)
	}

	org, err := h.service.Create(c.Request().Context(), in)
	if err != nil {
		return writeError(c, err)
	}

	c.Response().Header().Set("Location", fmt.Sprintf("/organizations/%d", org.ID))
	return c.JSON(http.StatusCreated, org)
}

// SetupState handles GET /setup-state. Returns 200 with
// {"hasOrganization": <bool>} on success. Errors return the locked
// HTTP envelope via writeError (R-OW-005).
func (h *OrganizationHandler) SetupState(c *echo.Context) error {
	state, err := h.service.GetSetupState(c.Request().Context())
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(http.StatusOK, state)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseCreateInput decodes the request body as JSON or
// form-encoded, depending on the Content-Type header. The dual
// content-type contract is locked #3.
//
// On a malformed body, parseCreateInput returns a
// *domain.ValidationError so the error envelope stays consistent
// across every failure mode the client can hit.
func parseCreateInput(c *echo.Context) (domain.CreateOrganizationInput, error) {
	ct := c.Request().Header.Get(echo.HeaderContentType)

	if isJSON(ct) {
		var body struct {
			FullName       string  `json:"full_name"`
			Identification string  `json:"identification"`
			ShortName      *string `json:"shortname"`
			Email          *string `json:"email"`
			Phone          *string `json:"phone"`
		}
		if err := json.NewDecoder(c.Request().Body).Decode(&body); err != nil {
			return domain.CreateOrganizationInput{}, &domain.ValidationError{
				Fields: map[string]string{"body": "Request body is not valid JSON."},
			}
		}
		return domain.CreateOrganizationInput{
			FullName:       body.FullName,
			Identification: body.Identification,
			ShortName:      body.ShortName,
			Email:          body.Email,
			Phone:          body.Phone,
		}, nil
	}

	if isForm(ct) {
		if err := c.Request().ParseForm(); err != nil {
			return domain.CreateOrganizationInput{}, &domain.ValidationError{
				Fields: map[string]string{"body": "Request body is not valid form data."},
			}
		}
		return domain.CreateOrganizationInput{
			FullName:       c.Request().PostFormValue("full_name"),
			Identification: c.Request().PostFormValue("identification"),
			ShortName:      optionalFormValue(c, "shortname"),
			Email:          optionalFormValue(c, "email"),
			Phone:          optionalFormValue(c, "phone"),
		}, nil
	}

	return domain.CreateOrganizationInput{}, &domain.ValidationError{
		Fields: map[string]string{"body": "Content-Type must be application/json or application/x-www-form-urlencoded."},
	}
}

// optionalFormValue returns a non-nil pointer only when the form
// value is present and non-empty. Treats "" as "not provided" so
// the domain validation rule for email/phone (which accepts
// absent-but-not-empty) matches the JSON branch.
func optionalFormValue(c *echo.Context, key string) *string {
	v := c.Request().PostFormValue(key)
	if v == "" {
		return nil
	}
	return &v
}

// isJSON reports whether the Content-Type is application/json (or
// a variant with parameters, e.g. "application/json; charset=utf-8").
func isJSON(ct string) bool {
	// Echo constant is "application/json"; strip parameters before
	// comparing.
	if i := indexOf(ct, ';'); i >= 0 {
		ct = trimSpace(ct[:i])
	}
	return ct == "application/json"
}

// isForm reports whether the Content-Type is
// application/x-www-form-urlencoded.
func isForm(ct string) bool {
	if i := indexOf(ct, ';'); i >= 0 {
		ct = trimSpace(ct[:i])
	}
	return ct == "application/x-www-form-urlencoded"
}

// writeError centralizes the mapping from a domain AppError to
// the locked HTTP envelope (spec §3.1, §3.3). One place to change
// if the envelope shape ever evolves.
func writeError(c *echo.Context, err error) error {
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
			"message": domain.MsgConflictSlug,
		})
	}

	var nerr *domain.NotFoundError
	if errors.As(err, &nerr) {
		return c.JSON(http.StatusNotFound, map[string]any{
			"error":   domain.CodeNotFound,
			"message": domain.MsgNotFound,
		})
	}

	// Any other error (including *InternalError) is a 500 with the
	// locked generic message. The wrapped Cause is logged so an
	// operator can diagnose without the wire leaking internals.
	var ierr *domain.InternalError
	if errors.As(err, &ierr) && ierr.Cause != nil {
		slog.ErrorContext(c.Request().Context(), "organization internal error",
			slog.String("error", ierr.Cause.Error()),
		)
	} else {
		slog.ErrorContext(c.Request().Context(), "organization unhandled error",
			slog.String("error", err.Error()),
		)
	}
	return c.JSON(http.StatusInternalServerError, map[string]any{
		"error":   domain.CodeServer,
		"message": domain.MsgServerFailure,
	})
}

// indexOf is a tiny helper to avoid pulling strings just for
// one substring call.
func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// trimSpace is a tiny helper that strips leading and trailing
// whitespace (space, tab, CR, LF) without pulling strings.
func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end {
		c := s[start]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		start++
	}
	for end > start {
		c := s[end-1]
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			break
		}
		end--
	}
	return s[start:end]
}
