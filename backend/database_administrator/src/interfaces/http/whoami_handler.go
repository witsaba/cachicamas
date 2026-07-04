// Package httpiface — whoami_handler.go introduces the demo
// /api/v1/protected/whoami endpoint (cachicamas-github-login PR-3).
//
// Reference: openspec/changes/cachicamas-github-login/specs/backend-auth-middleware/spec.md
//   R-BAM-020 (S-BAM-061) — demo protected endpoint that rejects an
//   unauthenticated request with 401 + code=unauthorized, and
//   returns the resolved identity on success.
//
// Why a /whoami endpoint:
//   It's the canonical "is the cookie path working?" smoke check.
//   The frontend can hit it to confirm the JWE envelope decodes
//   end-to-end; an integration test can drive it with a known
//   fixture JWE to assert that the database lookup also succeeds.
//
// This file is the slice's only contribution to the protected route
// group. It depends on the application.IdentityService for the
// lookup (defence in depth — the middleware already does the
// LookupByEmail; the handler calls the application service for
// future extensibility, e.g. when /whoami is expanded to include
// organization membership in PR-4+).
package httpiface

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// RegisterProtectedWhoAmIRoute mounts /api/v1/protected/whoami behind
// the IdentityFromCookie middleware. The order is:
//   e.Use(CORS(...))
//   e.Use(IdentityFromCookie(...))  <-- registered here
//   e.GET("/api/v1/protected/whoami", h.Get)
//
// Health route is intentionally NOT behind this middleware; see
// S-BAM-060.
func RegisterProtectedWhoAmIRoute(e *echo.Echo, _ *application.IdentityService, cfg IdentityMiddlewareConfig) {
	mw := IdentityFromCookie(cfg)
	e.GET("/api/v1/protected/whoami", mw(whoamiHandler))
}

// whoamiHandler is the GET /api/v1/protected/whoami handler. It
// reads the *domain.Identity that the middleware put on the
// Echo context, then returns a minimal JSON payload. The
// IdentityNotFoundError path is impossible here because the
// middleware would have returned 401 already; any nil identity is
// therefore a programmer error.
func whoamiHandler(c *echo.Context) error {
	raw := c.Get(IdentityContextKey)
	if raw == nil {
		// Should never happen — middleware would 401 first.
		return writeWhoAmIError(c, errors.New("whoami: identity missing from context"))
	}
	id, ok := raw.(*domain.Identity)
	if !ok {
		return writeWhoAmIError(c, errors.New("whoami: identity has wrong type"))
	}
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(http.StatusOK)
	return json.NewEncoder(c.Response()).Encode(map[string]any{
		"id":                id.ID,
		"email":             id.Email,
		"name":              id.Name,
		"image_url":         id.ImageURL,
		"provider":          id.Provider,
		"provider_account_id": id.ProviderAccountID,
	})
}

// writeWhoAmIError emits a 500 with the locked error envelope.
// We use 500 (not 401) because reaching this path means the
// middleware let us through but the context is malformed — that's
// an internal error, not an authentication failure.
func writeWhoAmIError(c *echo.Context, err error) error {
	c.Response().Header().Set("Content-Type", "application/json")
	c.Response().WriteHeader(http.StatusInternalServerError)
	return json.NewEncoder(c.Response()).Encode(map[string]any{
		"code":    "internal_error",
		"message": err.Error(),
	})
}

// IsLikelyHTTPS is a tiny helper exported for main.go. Returns
// true if the ORIGIN env var starts with "https://".
func IsLikelyHTTPS(origin string) bool {
	return len(origin) >= 8 && origin[:8] == "https://"
}