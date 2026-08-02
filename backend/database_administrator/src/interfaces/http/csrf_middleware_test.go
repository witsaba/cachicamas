// Package httpiface_test — csrf_middleware_test.go covers the
// Origin/Referer validation middleware for state-changing requests.
// The contract:
//   - POST/PATCH/DELETE/PUT with a non-matching Origin → 403
//   - POST/PATCH/DELETE/PUT with a matching Origin → 200
//   - GET/HEAD/OPTIONS skip the validator entirely
//   - When Origin is missing, Referer is used as a fallback
//   - When both are missing, the request is rejected
//   - Empty ORIGIN env preserves development (no enforcement)
package httpiface_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"

	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

// buildCSRFProtectedEcho mounts a single echo handler under the
// CSRF middleware. The handler returns 200 with the headers that
// affected the decision (so the test can assert).
func buildCSRFProtectedEcho(t *testing.T, originEnv string) *echo.Echo {
	t.Helper()
	t.Setenv("ORIGIN", originEnv)
	e := echo.New()
	e.Use(httpiface.CSRFOriginValidate())
	e.POST("/x", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"ok": true})
	})
	e.PATCH("/x", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"ok": true})
	})
	e.DELETE("/x", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"ok": true})
	})
	e.PUT("/x", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"ok": true})
	})
	e.GET("/x", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"ok": true})
	})
	return e
}

func TestCSRF_CrossOriginPOST_Rejected(t *testing.T) {
	e := buildCSRFProtectedEcho(t, "https://app.example.com")

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin POST: expected 403, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestCSRF_SameOriginPOST_Allowed(t *testing.T) {
	e := buildCSRFProtectedEcho(t, "https://app.example.com")

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("same-origin POST: expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestCSRF_RefererFallbackAllowsSameOrigin(t *testing.T) {
	e := buildCSRFProtectedEcho(t, "https://app.example.com")

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("Referer", "https://app.example.com/some/path")
	// No Origin header.
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Referer fallback to same origin: expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestCSRF_RefererFallbackRejectsCrossOrigin(t *testing.T) {
	e := buildCSRFProtectedEcho(t, "https://app.example.com")

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("Referer", "https://evil.example.com/")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("cross-origin Referer fallback: expected 403, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestCSRF_MissingBothHeaders_Rejected(t *testing.T) {
	e := buildCSRFProtectedEcho(t, "https://app.example.com")

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	// No Origin, no Referer.
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("missing both Origin + Referer: expected 403, got %d", rec.Code)
	}
}

func TestCSRF_SafeMethodsSkipMiddleware(t *testing.T) {
	e := buildCSRFProtectedEcho(t, "https://app.example.com")

	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/x", nil)
			// Cross-origin header — must NOT be rejected.
			req.Header.Set("Origin", "https://evil.example.com")
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code == http.StatusForbidden {
				t.Fatalf("safe method %s unexpectedly rejected: %d", method, rec.Code)
			}
		})
	}
}

func TestCSRF_EmptyOriginEnv_AllowsAnyReference(t *testing.T) {
	// Empty ORIGIN env = development mode (no enforcement).
	e := buildCSRFProtectedEcho(t, "")

	req := httptest.NewRequest(http.MethodPost, "/x", nil)
	req.Header.Set("Origin", "https://anything.example.com")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("empty ORIGIN env: expected 200 (development unwinding), got %d", rec.Code)
	}
}
