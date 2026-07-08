// Package token contains the bearer-token middleware that guards
// every internal endpoint on the workspace_syncer. See
// openspec/changes/2026-07-08-workspace-sync-clone/design.md §5
// (Cross-service auth posture) and the ADR at
// adr/workspace-syncer-internal-auth for the v1 static-bearer choice.
package token

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// bearerPrefix is the literal "Bearer " prefix per RFC 7235. The
// comparison against the Authorization header value is
// case-insensitive (we lowercase the prefix and the header prefix
// before comparing).
const bearerPrefix = "Bearer "

// ServiceTokenMiddleware returns an Echo middleware that rejects any
// request whose Authorization header does not exactly match
// "Bearer <expected>" using a constant-time compare.
//
// Behavior:
//   - Missing Authorization header → 401 with
//     {"error": "unauthorized", "message": "..."}.
//   - Authorization header without the "Bearer " prefix → 401.
//   - Authorization header with the wrong token → 401.
//   - Authorization header with the correct token → next handler runs.
//
// The comparison uses subtle.ConstantTimeCompare to defend against
// timing attacks; both length-mismatch and value-mismatch are
// handled in constant time. The handler does NOT distinguish between
// "missing", "malformed", and "wrong token" in the response body —
// the 401 envelope is the same so an attacker cannot probe for
// "is the token shape right" vs "is the token value right".
//
// The expected token is captured by value at construction time (not
// by reference) so a future caller that mutates the source slice
// after NewServiceTokenMiddleware does not change the middleware's
// behavior. Defense-in-depth.
//
// The middleware is the ONLY auth gate on the workspace_syncer; it
// is applied to the entire Echo instance in main.go.
// ServiceTokenMiddlewareConfig is the optional configuration for
// ServiceTokenMiddleware. The zero value is the strictest possible:
// no skipper (every request is checked), and the fail-safe empty-token
// behavior is in effect.
type ServiceTokenMiddlewareConfig struct {
	// Skipper, when set, returns true to skip the token check. Used
	// to expose a public liveness endpoint (e.g. /healthz) without
	// the bearer token. The skipper is called BEFORE the token is
	// inspected, so a skipper that matches a path is cheaper than
	// the constant-time compare.
	Skipper func(c *echo.Context) bool
}

// ServiceTokenMiddleware returns an Echo middleware that rejects any
// request whose Authorization header does not exactly match
// "Bearer <expected>" using a constant-time compare.
//
// Behavior:
//   - If the skipper returns true for a request, the middleware
//     passes through without inspecting the Authorization header.
//   - Missing Authorization header → 401 with
//     {"error": "unauthorized", "message": "..."}.
//   - Authorization header without the "Bearer " prefix → 401.
//   - Authorization header with the wrong token → 401.
//   - Authorization header with the correct token → next handler runs.
//
// The comparison uses subtle.ConstantTimeCompare to defend against
// timing attacks; both length-mismatch and value-mismatch are
// handled in constant time.
//
// The expected token is captured by value at construction time (not
// by reference) so a future caller that mutates the source slice
// after NewServiceTokenMiddleware does not change the middleware's
// behavior. Defense-in-depth.
//
// The middleware is the ONLY auth gate on the workspace_syncer; it
// is applied to the entire Echo instance in main.go. Routes that
// must be public (e.g. /healthz) opt out via the Skipper config.
func ServiceTokenMiddleware(expected string, cfg ...ServiceTokenMiddlewareConfig) echo.MiddlewareFunc {
	var skipper func(c *echo.Context) bool
	if len(cfg) > 0 {
		skipper = cfg[0].Skipper
	}
	// Fail-safe: if the expected token is empty, every request is
	// rejected unconditionally (unless the skipper matches). This
	// guards against a misconfiguration (e.g. main.go forgot to
	// read INTERNAL_SERVICE_TOKEN) — better to refuse all traffic
	// than to accept any. main.go's own fail-fast on empty env var
	// is the first line of defense; this is the second.
	if expected == "" {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c *echo.Context) error {
				if skipper != nil && skipper(c) {
					return next(c)
				}
				return unauthorized(c)
			}
		}
	}
	expectedBytes := []byte(expected)
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if skipper != nil && skipper(c) {
				return next(c)
			}
			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return unauthorized(c)
			}
			// Case-insensitive prefix check: the RFC says case-sensitive
			// but every major client (curl -H, Go net/http) sends the
			// canonical "Bearer " form. We accept both.
			if len(auth) < len(bearerPrefix) ||
				!strings.EqualFold(auth[:len(bearerPrefix)], bearerPrefix) {
				return unauthorized(c)
			}
			token := []byte(auth[len(bearerPrefix):])
			// Constant-time compare. The function returns 1 iff the
			// two byte slices have equal content AND equal length;
			// otherwise 0. Both branches take the same amount of
			// time, defeating timing-based token extraction.
			if subtle.ConstantTimeCompare(token, expectedBytes) != 1 {
				return unauthorized(c)
			}
			return next(c)
		}
	}
}

// unauthorized writes the locked 401 envelope. The body shape is
// the flat `{ "error": "unauthorized", "message": "..." }` envelope,
// matching the database_administrator convention (see
// design.md §6 — Error envelope consolidation).
func unauthorized(c *echo.Context) error {
	return c.JSON(http.StatusUnauthorized, map[string]string{
		"error":   "unauthorized",
		"message": "Invalid or missing service token.",
	})
}
