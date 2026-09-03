// Package httpiface — auth_handler.go exposes the google-auth-bootstrap
// HTTP endpoints (POST /internal/auth/bootstrap, GET /internal/me/:user_id)
// plus the X-Internal-Secret middleware.
//
// The handler is a thin wrapper over the application/auth services:
// it decodes the JSON request body, invokes the service, and maps
// the service's domain errors to HTTP envelopes (400 / 401 / 404 /
// 500). The hexagonal rule holds: the handler depends on the
// application layer (services) only; pgx + database/sql imports are
// restricted to the postgres adapter package.
package httpiface

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/labstack/echo/v5"

	appauth "github.com/cachicamas/backend/database_administrator/src/application/auth"
	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
)

// AuthHandler exposes the google-auth-bootstrap endpoints. It
// depends on the BootstrapService + MeService; the middleware
// factory is exposed separately (see XInternalSecretMiddleware
// below).
type AuthHandler struct {
	bootstrap *appauth.BootstrapService
	me        *appauth.MeService
}

// NewAuthHandler wires the application services into an HTTP
// transport. The middleware (XInternalSecretMiddleware) is wired
// separately so it can be reused across multiple route groups.
func NewAuthHandler(bootstrap *appauth.BootstrapService, me *appauth.MeService) *AuthHandler {
	return &AuthHandler{bootstrap: bootstrap, me: me}
}

// RegisterAuthRoutes wires the two endpoints + the
// X-Internal-Secret gate on the given Echo instance. The gate is
// applied per-route (not globally) so unrelated endpoints (e.g.
// /health) remain reachable without the internal secret.
func RegisterAuthRoutes(e *echo.Echo, h *AuthHandler, internalSecret string) {
	g := e.Group("/internal", XInternalSecretMiddleware(internalSecret))
	g.POST("/auth/bootstrap", h.Bootstrap)
	g.GET("/me/:user_id", h.Me)
}

// bootstrapRequest is the wire DTO for POST /internal/auth/bootstrap.
// Field names mirror the locked spec §5 R-BE-001 shape.
type bootstrapRequest struct {
	GoogleSub     string `json:"google_sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	PictureURL    string `json:"picture_url"`
}

// bootstrapResponse mirrors the locked response shape: {user_id,
// pyme_id, status}. pyme_id is the alias used by the wire contract
// even though the schema is auth.organizations (the field name is
// preserved for backward-compatibility with the spec).
type bootstrapResponse struct {
	UserID  int64            `json:"user_id"`
	PymeID  int64            `json:"pyme_id"`
	Status  auth.UserStatus  `json:"status"`
}

// Bootstrap handles POST /internal/auth/bootstrap. Maps the
// service result to {user_id, pyme_id, status} on success; maps
// ErrValidation to 400, other errors to 500.
func (h *AuthHandler) Bootstrap(c *echo.Context) error {
	var req bootstrapRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "invalid request body: " + err.Error(),
		})
	}
	out, err := h.bootstrap.Bootstrap(c.Request().Context(), appauth.BootstrapInput{
		GoogleSub:     req.GoogleSub,
		Email:         req.Email,
		EmailVerified: req.EmailVerified,
		Name:          req.Name,
		PictureURL:    req.PictureURL,
		IPAddress:     c.Request().RemoteAddr,
		UserAgent:     c.Request().UserAgent(),
	})
	if err != nil {
		if errors.Is(err, appauth.ErrValidation) {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		slog.ErrorContext(c.Request().Context(), "bootstrap failed",
			slog.String("google_sub", req.GoogleSub),
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "bootstrap failed",
		})
	}
	return c.JSON(http.StatusOK, bootstrapResponse{
		UserID: out.UserID,
		PymeID: out.OrganizationID,
		Status: out.Status,
	})
}

// Me handles GET /internal/me/:user_id. Returns {user, organization}
// on success; 404 for unknown user_id; 400 for non-positive id.
func (h *AuthHandler) Me(c *echo.Context) error {
	rawID := c.Param("user_id")
	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "user_id must be a positive integer",
		})
	}
	out, err := h.me.Get(c.Request().Context(), id)
	if err != nil {
		var notFound *auth.NotFoundError
		if errors.As(err, &notFound) {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "user not found",
			})
		}
		if errors.Is(err, appauth.ErrValidation) {
			return c.JSON(http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
		}
		slog.ErrorContext(c.Request().Context(), "me failed",
			slog.Int64("user_id", id),
			slog.String("error", err.Error()),
		)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "me lookup failed",
		})
	}
	return c.JSON(http.StatusOK, out)
}

// XInternalSecretMiddleware validates the X-Internal-Secret header
// against the expected value using the application-layer gate
// (auth.CheckInternalSecret, constant-time compare). The
// middleware is per-route, mounted via RegisterAuthRoutes; it is
// NOT installed globally so unrelated endpoints remain reachable.
func XInternalSecretMiddleware(expected string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			got := c.Request().Header.Get("X-Internal-Secret")
			if err := appauth.CheckInternalSecret(got, expected); err != nil {
				// The application-layer gate distinguishes missing
				// from wrong so the operator log captures the
				// difference. The HTTP response is always 401 —
				// the client never needs to know which failed.
				slog.WarnContext(c.Request().Context(), "internal secret rejected",
					slog.String("error", err.Error()),
				)
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "unauthorized",
				})
			}
			return next(c)
		}
	}
}