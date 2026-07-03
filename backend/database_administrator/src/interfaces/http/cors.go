package httpiface

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// CORSMiddlewareConfig configures which origins are trusted to call
// the API cross-origin.  Activate in dev so the Qwik dev server
// (origin http://localhost:5173 by default) can POST + GET to
// localhost:8080 without the browser blocking on the Same-Origin
// Policy.  Production keeps this disabled — the reverse proxy in
// front of the binary is expected to terminate on the same origin
// as the frontend, so CORS isn't needed.
//
// Activation rules (see main.go):
//   - SERVICE_ENV=development  → enabled by default with
//                                 http://localhost:5173 as the only
//                                 allowed origin (override via
//                                 CORS_ALLOW_ORIGINS).
//   - CORS_ALLOW_ORIGINS set   → enabled with the listed origins,
//                                 regardless of SERVICE_ENV.
//   - Neither                  → disabled.  The API behaves exactly
//                                 as if no middleware was registered.
type CORSMiddlewareConfig struct {
	// AllowOrigins is an exact-match allowlist of `Origin` header
	// values.  Each entry should be a complete origin (scheme+host+port),
	// e.g. "http://localhost:5173" or "https://app.example.com".
	//
	// Empty disables CORS — the middleware becomes a passthrough.
	AllowOrigins []string
}

// CORS returns an Echo middleware that sets the right CORS headers
// for cross-origin requests from the allowlisted origins and
// short-circuits OPTIONS preflight requests with `204 No Content`.
//
// We deliberately do NOT enable `Access-Control-Allow-Credentials`:
// the Qwik form does not need cookies, and credentials + `*` origin
// is a security smell we never want lurking in this codebase.
func CORS(cfg CORSMiddlewareConfig) echo.MiddlewareFunc {
	allowed := make(map[string]struct{}, len(cfg.AllowOrigins))
	for _, o := range cfg.AllowOrigins {
		allowed[o] = struct{}{}
	}
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			req := c.Request()
			origin := req.Header.Get("Origin")
			if origin == "" {
				// Not a cross-origin request — let it through unchanged.
				return next(c)
			}
			if _, ok := allowed[origin]; !ok {
				// Origin not in the allowlist.  We do NOT set
				// `Access-Control-Allow-Origin`, so the browser
				// will block the response on the client side;
				// the server logs nothing (a common probe against
				// public services — no need to feed it).
				return next(c)
			}
			h := c.Response().Header()
			h.Set("Access-Control-Allow-Origin", origin)
			// Echo uses `Vary: Accept-Encoding` by default; we add
			// `Origin` so caches do not serve the wrong CORS header
			// when multiple origins hit the same URL.
			h.Add("Vary", "Origin")
			h.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			h.Set("Access-Control-Allow-Headers", "Content-Type")
			if req.Method == http.MethodOptions {
				// Preflight — answer without invoking downstream handlers.
				return c.NoContent(http.StatusNoContent)
			}
			return next(c)
		}
	}
}
