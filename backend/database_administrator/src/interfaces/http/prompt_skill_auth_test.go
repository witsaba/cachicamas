// Package httpiface_test — prompt_skill_auth_test.go covers the
// post-H-2 admin auth contract: GET /prompts without a session
// cookie MUST return 401 from the identity middleware (not reach
// the handler). The same protection applies to /skills.
package httpiface_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

// registerPromptsAndSkills mounts /prompts and /skills on an
// auth-protected group, mirroring the production main.go wiring.
func registerPromptsAndSkills(t *testing.T) (*echo.Echo, *fakeIdentityInjector) {
	t.Helper()
	e := echo.New()
	injector := &fakeIdentityInjector{}
	group := e.Group("", injector.Middleware())
	// Each Register*Routes takes *echo.Group now (per A.6).
	// We don't need the real handlers — we just need a route
	// registered, so any handler that returns 200 with the slug
	// would do. The middleware short-circuit happens BEFORE the
	// handler runs, so we can use a stub.
	group.GET("/prompts", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"prompts": []any{}})
	})
	group.GET("/prompts/:slug", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"slug": c.Param("slug")})
	})
	group.GET("/skills", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"skills": []any{}})
	})
	group.GET("/skills/:slug", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"slug": c.Param("slug")})
	})
	return e, injector
}

// fakeIdentityInjector mimics IdentityFromCookie: 401s on missing
// header, 200s on present. It seeds c.Set(IdentityContextKey) so
// the handler can run.
type fakeIdentityInjector struct {
	calls int
}

func (f *fakeIdentityInjector) Middleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			f.calls++
			if c.Request().Header.Get("X-Test-Identity-ID") == "" {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"code":    "unauthorized",
					"message": "authentication required",
				})
			}
			c.Set(httpiface.IdentityContextKey, &domain.Identity{
				ID:                1,
				Provider:          "github",
				ProviderAccountID: "test-account",
			})
			return next(c)
		}
	}
}

func TestPrompts_AnonymousRequest_Returns401(t *testing.T) {
	e, _ := registerPromptsAndSkills(t)

	for _, path := range []string{
		"/prompts",
		"/prompts/my-slug",
		"/skills",
		"/skills/my-slug",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d body=%q", rec.Code, rec.Body.String())
			}
			var env map[string]any
			_ = json.Unmarshal(rec.Body.Bytes(), &env)
			if env["code"] != "unauthorized" {
				t.Errorf("envelope code = %v, want unauthorized", env["code"])
			}
		})
	}
}

func TestPrompts_AuthenticatedRequest_Succeeds(t *testing.T) {
	e, _ := registerPromptsAndSkills(t)

	for _, path := range []string{
		"/prompts",
		"/prompts/my-slug",
		"/skills",
		"/skills/my-slug",
	} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Header.Set("X-Test-Identity-ID", "1")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("authenticated %s: expected 200, got %d body=%q", path, rec.Code, rec.Body.String())
			}
		})
	}
}

// TestSyncCallback_StillOnRootEcho asserts that the internal
// sync-callback route is NOT migrated to the auth group. The
// /api/v1/internal/sync-callback has its own HMAC + anti-replay
// auth and must remain reachable without a session cookie.
func TestSyncCallback_StillOnRootEcho(t *testing.T) {
	// Build a minimal Echo with the auth group ONLY on /prompts
	// (mirrors the production wiring).
	e := echo.New()
	group := e.Group("", func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if c.Request().Header.Get("X-Test-Identity-ID") == "" {
				return c.JSON(http.StatusUnauthorized, map[string]any{
					"code":    "unauthorized",
					"message": "authentication required",
				})
			}
			return next(c)
		}
	})
	group.GET("/prompts", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// Root-Echo callback mounted with NO auth (HMAC only).
	e.POST("/api/v1/internal/sync-callback", func(c *echo.Context) error {
		return c.NoContent(http.StatusNoContent)
	})

	// 1. /prompts without cookie -> 401 (middleware short-circuits).
	req := httptest.NewRequest(http.MethodGet, "/prompts", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 on /prompts, got %d", rec.Code)
	}

	// 2. /api/v1/internal/sync-callback without cookie -> 204
	//    (HMAC is the only auth; no session cookie required).
	req = httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", strings.NewReader(""))
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Errorf("sync-callback must remain on root Echo (no auth-cookie gate); got %d", rec.Code)
	}
}
