package token

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func newTestEcho(token string) *echo.Echo {
	e := echo.New()
	e.Use(ServiceTokenMiddleware(token))
	e.POST("/protected", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"ok": "true"})
	})
	return e
}

func TestServiceTokenMiddleware_NoHeader(t *testing.T) {
	e := newTestEcho("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/protected", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "unauthorized" {
		t.Errorf("body.error = %q, want %q", body["error"], "unauthorized")
	}
	if body["message"] == "" {
		t.Errorf("body.message is empty")
	}
}

func TestServiceTokenMiddleware_WrongPrefix(t *testing.T) {
	// "Basic abc" instead of "Bearer abc" — should be rejected.
	e := newTestEcho("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/protected", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestServiceTokenMiddleware_CaseInsensitivePrefix(t *testing.T) {
	// "bearer" (lowercase) and "BEARER" (uppercase) MUST both be
	// accepted per the middleware's case-insensitive check.
	cases := []string{"bearer secret-token", "BEARER secret-token", "BeArEr secret-token"}
	for _, auth := range cases {
		t.Run(auth, func(t *testing.T) {
			e := newTestEcho("secret-token")
			req := httptest.NewRequest(http.MethodPost, "/protected", bytes.NewReader([]byte("{}")))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", auth)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200 (auth=%q)", rec.Code, auth)
			}
		})
	}
}

func TestServiceTokenMiddleware_WrongToken(t *testing.T) {
	e := newTestEcho("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/protected", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestServiceTokenMiddleware_PrefixOnly(t *testing.T) {
	// "Bearer " with no token after it — should be rejected because
	// the constant-time compare sees a zero-length token against the
	// expected non-empty token.
	e := newTestEcho("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/protected", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestServiceTokenMiddleware_CorrectToken(t *testing.T) {
	e := newTestEcho("secret-token")
	req := httptest.NewRequest(http.MethodPost, "/protected", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"ok":"true"`) {
		t.Errorf("body = %q, want it to contain the ok payload", rec.Body.String())
	}
}

func TestServiceTokenMiddleware_EmptyToken(t *testing.T) {
	// ServiceTokenMiddleware("") means no client can ever authenticate
	// (the constant-time compare against an empty expected never
	// matches a non-empty token). This is intentional: it is a
	// fail-safe if main.go forgets to wire the env var. main.go's
	// own fail-fast on empty INTERNAL_SERVICE_TOKEN is the
	// first line of defense; this is the second.
	e := newTestEcho("")
	req := httptest.NewRequest(http.MethodPost, "/protected", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (empty expected token must reject every request)", rec.Code)
	}
}
