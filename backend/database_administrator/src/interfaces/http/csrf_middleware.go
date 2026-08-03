// Package httpiface — csrf_middleware.go introduces the Origin /
// Referer validation middleware for state-changing requests.
//
// 2026-08-02-security-vulnerability-remediation (cross-cutting): this
// middleware is the server-side enforcement for the CSRF surface
// the spec calls csrf-origin-validation. The frontend also adds
// the X-Requested-With header (slice C, frontend-fixes), but the
// authority is the server-side check here.
//
// Why server-side: the browser sets Referer + Origin automatically
// for cross-origin requests; a malicious page cannot suppress them
// (the browser will omit them but never rewrite them). The proxy
// at /api/v1/* forwards the headers as-is, so the backend sees
// the original value.
//
// Why Origin + Referer fallback: some HTTP clients (and some
// browser configurations) omit Origin on same-origin requests and
// rely on Referer instead. The fallback accepts both shapes.
//
// Why empty ORIGIN env preserves development: dev environments
// (pnpm dev, /etc/hosts aliases) frequently have a different
// origin than production. The whitelist is env-driven so dev can
// skip enforcement without recompiling.
package httpiface

import (
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/labstack/echo/v5"
)

// csrfStateChangingMethods is the set of methods that require
// same-origin validation. Safe methods (GET/HEAD/OPTIONS) skip the
// middleware entirely per the spec.
var csrfStateChangingMethods = map[string]bool{
	http.MethodPost:   true,
	http.MethodPatch:  true,
	http.MethodPut:    true,
	http.MethodDelete: true,
}

// CSRFOriginValidate returns the Echo middleware that validates
// Origin / Referer on state-changing requests. The ORIGIN env var
// is the canonical allowed origin (e.g. "https://app.example.com").
// An empty ORIGIN env disables the middleware (development mode).
//
// The middleware:
//   - skips GET, HEAD, OPTIONS (safe methods per the spec).
//   - reads `Origin` first; falls back to `Referer` when Origin is
//     missing.
//   - rejects the request with 403 when no Origin/Referer is
//     supplied, or when the parsed origin does not match ORIGIN.
//   - never reads the request body (the headers are enough).
//
// Notes:
//   - The http.Client / fetch spec requires the browser to set
//     Origin on POST/PATCH/DELETE/PUT. A missing Origin on a
//     state-changing request is a strong signal that the request
//     is NOT coming from a browser — reject it.
//   - We DO NOT compare the full URL (path / query); only the
//     Origin scheme + host + port. The spec is same-origin, not
//     same-path.
func CSRFOriginValidate() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			method := c.Request().Method
			if !csrfStateChangingMethods[strings.ToUpper(method)] {
				return next(c)
			}

			allowed := strings.TrimSpace(os.Getenv("ORIGIN"))
			if allowed == "" {
				// Development mode: no enforcement.
				return next(c)
			}

			origin := strings.TrimSpace(c.Request().Header.Get("Origin"))
			if origin == "" {
				// Fallback to Referer when Origin is missing.
				// RFC 7231: Referer may be absent for same-origin
				// requests; Origin is the modern signal.
				referer := strings.TrimSpace(c.Request().Header.Get("Referer"))
				if referer == "" {
					return csrfReject(c, "missing_origin_and_referer")
				}
				origin = referer
			}

			if !sameOrigin(origin, allowed) {
				return csrfReject(c, "origin_mismatch")
			}
			return next(c)
		}
	}
}

// sameOrigin returns true when parsedURL's origin (scheme + host +
// port) matches the allowedOrigin string. The comparison is
// string-based on the canonicalized scheme://host[:port] form to
// avoid leaking the exact path / query / fragment.
func sameOrigin(rawURL, allowedOrigin string) bool {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	allowed, err := url.Parse(allowedOrigin)
	if err != nil || allowed.Scheme == "" || allowed.Host == "" {
		return false
	}
	return strings.EqualFold(u.Scheme, allowed.Scheme) && strings.EqualFold(u.Host, allowed.Host)
}

// csrfReject emits the locked 403 envelope and a slog reason line.
// The body is generic; the operator learns the reason via the log.
func csrfReject(c *echo.Context, reason string) error {
	slog.WarnContext(c.Request().Context(), "csrf origin rejected",
		slog.String("reason", reason),
		slog.String("path", c.Request().URL.Path),
		slog.String("method", c.Request().Method),
	)
	return c.JSON(http.StatusForbidden, map[string]any{
		"code":    "csrf_origin_rejected",
		"message": "request origin does not match the configured ORIGIN",
	})
}
