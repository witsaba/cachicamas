// Package httpiface_test contains the test suite for the HTTP transport layer.
package httpiface

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

// TestHealthEndpoint_ReturnsOK verifies the GET /health endpoint:
//   - responds with HTTP 200
//   - responds with content-type application/json
//   - responds with body {"status":"ok"}
func TestHealthEndpoint_ReturnsOK(t *testing.T) {
	// Arrange
	e := echo.New()
	RegisterHealthRoute(e)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	// Act
	e.ServeHTTP(rec, req)

	// Assert: status code
	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	// Assert: content type
	if ct := rec.Header().Get(echo.HeaderContentType); ct != echo.MIMEApplicationJSON && ct != "application/json" {
		t.Fatalf("expected content-type application/json, got %q", ct)
	}

	// Assert: body shape
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("response body is not valid JSON: %v (body=%q)", err, rec.Body.String())
	}
	if got, want := body["status"], "ok"; got != want {
		t.Fatalf("expected body status=%q, got %q (full body=%v)", want, got, body)
	}
}
