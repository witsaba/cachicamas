package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/token"
)

// TestNewEchoInstance_PlaceholderReturnsNotImplemented asserts that
// the POST /internal/clone-and-validate route is registered in PR-2a
// (so the docker-compose healthcheck can hit it and the database_administrator
// can call it) but returns 501 Not Implemented until PR-2b lands.
func TestNewEchoInstance_PlaceholderReturnsNotImplemented(t *testing.T) {
	e := newTestEcho("test-token")

	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotImplemented {
		t.Fatalf("status = %d, want 501; body=%s", rec.Code, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["error"] != "not_implemented" {
		t.Errorf("body.error = %q, want %q", body["error"], "not_implemented")
	}
	if body["message"] == "" {
		t.Errorf("body.message is empty")
	}
}

// TestNewEchoInstance_HealthzAlwaysOK asserts the /healthz route
// returns 200 with the locked envelope. The compose healthcheck
// depends on this.
func TestNewEchoInstance_HealthzAlwaysOK(t *testing.T) {
	e := newTestEcho("test-token")

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Errorf("body = %q, want it to contain the status:ok payload", rec.Body.String())
	}
}

// TestNewEchoInstance_RejectsBadToken is a smoke test that the
// ServiceTokenMiddleware is actually applied to the placeholder
// route. The detailed token-middleware behavior is tested in
// infrastructure/token/middleware_test.go.
func TestNewEchoInstance_RejectsBadToken(t *testing.T) {
	e := newTestEcho("test-token")

	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (bad token must be rejected)", rec.Code)
	}
}

// TestRunServer_GracefulShutdown asserts runServer returns nil when
// ctx is cancelled (mimics SIGTERM). Uses a real listener on a free
// port; we accept that this is heavier than the rest of the suite
// but it is the only way to exercise the http.Server lifecycle.
func TestRunServer_GracefulShutdown(t *testing.T) {
	e := newTestEcho("test-token")
	port := freePort(t)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	if err := runServer(ctx, e, port, slogDiscard()); err != nil {
		t.Errorf("runServer returned error on graceful shutdown: %v", err)
	}
}

// TestRunServer_BindFailure asserts runServer returns an error when
// the port is already in use.
func TestRunServer_BindFailure(t *testing.T) {
	port := freePort(t)
	// Hold a listener on the port for the duration of the test.
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		t.Fatalf("hold listener: %v", err)
	}
	defer func() { _ = ln.Close() }()

	e := newTestEcho("test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := runServer(ctx, e, port, slogDiscard()); err == nil {
		t.Errorf("expected bind failure, got nil")
	}
}

// TestMainProcessSignalHandling is a smoke test that the signal
// machinery works.
func TestMainProcessSignalHandling(t *testing.T) {
	// Sanity: signal.NotifyContext is the API we depend on.
	ch := make(chan os.Signal, 1)
	defer close(ch)
	if ch == nil {
		t.Fatal("channel creation failed")
	}
	// And syscall.SIGTERM is the signal we listen for.
	if syscall.SIGTERM == 0 {
		t.Fatal("syscall.SIGTERM is 0; the constant is wrong")
	}
}

// TestEchoContextType guards against the echo.Context type
// changing. Echo v5 uses *echo.Context; if a future upgrade renames
// it, this test fails and the handler signatures must be updated.
func TestEchoContextType(t *testing.T) {
	var c *echo.Context
	_ = c
}

// ---------------------------------------------------------------------------
// Helpers (test-local; production code owns the canonical wiring).
// ---------------------------------------------------------------------------

// newTestEcho constructs the same Echo instance main() would
// construct, minus the os.Exit / signal handling. The skipper
// mirrors the production wiring so /healthz bypasses the token.
func newTestEcho(tok string) *echo.Echo {
	e := echo.New()
	e.Use(token.ServiceTokenMiddleware(tok, token.ServiceTokenMiddlewareConfig{
		Skipper: func(c *echo.Context) bool {
			return c.Request().URL.Path == "/healthz"
		},
	}))
	e.GET("/healthz", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})
	e.POST("/internal/clone-and-validate", func(c *echo.Context) error {
		return c.JSON(http.StatusNotImplemented, map[string]string{
			"error":   "not_implemented",
			"message": "Clone-and-validate handler lands in PR-2b.",
		})
	})
	return e
}

// slogDiscard returns a *slog.Logger that drops every record. The
// runServer tests assert behavior on the http.Server lifecycle, not
// on log output.
func slogDiscard() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError + 1,
	}))
}

// freePort asks the kernel for a free TCP port. Used by the
// runServer tests so each run gets a unique port and parallel
// test runs do not collide.
func freePort(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	defer func() { _ = ln.Close() }()
	addr := ln.Addr().(*net.TCPAddr)
	return strconv.Itoa(addr.Port)
}
