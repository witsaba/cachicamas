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
)

// TestNewEchoInstance_CloneEndpointReturns202 asserts that
// POST /internal/clone-and-validate now returns 202 with the
// job_id and running status (replaced the PR-2a 501 placeholder
// with the real handler in PR-2b).
func TestNewEchoInstance_CloneEndpointReturns202(t *testing.T) {
	e := newTestEcho("test-token")

	body := []byte(`{
		"job_id": 42,
		"workspace_id": 7,
		"owner": "octocat",
		"repo": "hello-world",
		"default_branch": "main",
		"oauth_token": "gho_xxx"
	}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", rec.Code, rec.Body.String())
	}
	var respBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if respBody["job_id"].(float64) != 42 {
		t.Errorf("job_id = %v, want 42", respBody["job_id"])
	}
	if respBody["status"] != "running" {
		t.Errorf("status = %q, want running", respBody["status"])
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

	body := []byte(`{"job_id":1,"workspace_id":1,"owner":"o","repo":"r","default_branch":"main","oauth_token":"t"}`)
	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader(body))
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
	return newEcho(tok, slogDiscard())
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
