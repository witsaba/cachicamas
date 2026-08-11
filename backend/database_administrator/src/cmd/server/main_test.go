// Package cmd tests for the middleware chain. The main package is
// the composition root; main_test.go is a sibling package that
// exercises the security middleware decisions (BodyLimit + Recover)
// end-to-end without spinning up an Echo listener.
package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// bodyLimitBytes is the production-security cap. The constant lives
// at the package level so the wiring tests can reference it without
// duplicating the magic number.
const bodyLimitBytes int64 = 1 << 20 // 1 MiB

// TestBodyLimit_RejectsOversizeBody verifies that a request body
// exceeding the 1 MiB cap is rejected with 413 Request Entity Too
// Large (security M-1). The middleware must close the connection
// before the handler runs.
func TestBodyLimit_RejectsOversizeBody(t *testing.T) {
	e := echo.New()
	e.Use(middleware.BodyLimit(bodyLimitBytes))
	e.POST("/x", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	oversize := bytes.Repeat([]byte("a"), int(bodyLimitBytes)+1)
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(oversize))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413, got %d body=%q", rec.Code, rec.Body.String())
	}
}

// TestBodyLimit_AcceptsNormalBody verifies that a 100 KiB body is
// not rejected by the 1 MiB cap (happy path).
func TestBodyLimit_AcceptsNormalBody(t *testing.T) {
	e := echo.New()
	e.Use(middleware.BodyLimit(bodyLimitBytes))
	e.POST("/x", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	normal := bytes.Repeat([]byte("a"), 100*1024)
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(normal))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d body=%q", rec.Code, rec.Body.String())
	}
}

// TestRecover_HandlerPanicReturns500 verifies that a panicking
// handler is recovered (security M-4) and the process does NOT
// crash. The middleware supplies the 500 envelope.
func TestRecover_HandlerPanicReturns500(t *testing.T) {
	e := echo.New()
	e.Use(middleware.Recover())
	e.GET("/panic", func(_ *echo.Context) error {
		panic("something exploded (intentional test panic)")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 after panic, got %d body=%q", rec.Code, rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "intentional test panic") {
		t.Errorf("panic details leaked into response body: %q", rec.Body.String())
	}
}

// TestProductionMiddlewareChain_Ordering asserts the production
// ordering is Recover + BodyLimit installed before any route. The
// regression test below builds a chain in the SAME order main.go
// uses and asserts the documented contracts hold end-to-end.
func TestProductionMiddlewareChain_Ordering(t *testing.T) {
	e := echo.New()
	// Production order: recover first, then body limit (so recover
	// catches any panic downstream).
	e.Use(middleware.Recover(), middleware.BodyLimit(bodyLimitBytes))
	e.POST("/x", func(c *echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// Oversize body still 413.
	oversize := bytes.Repeat([]byte("a"), int(bodyLimitBytes)+1)
	req := httptest.NewRequest(http.MethodPost, "/x", bytes.NewReader(oversize))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("expected 413 with BodyLimit + Recover, got %d", rec.Code)
	}

	// Panic handler still 500.
	e.GET("/panic", func(_ *echo.Context) error {
		panic("test panic")
	})
	req = httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 after panic, got %d", rec.Code)
	}
}

// TestResolveSyncerURL covers the WORKSPACE_SYNCER_URL_REQUIRE_TLS
// opt-in env (security M-3). The decision is encoded in
// resolveSyncerURL(): when the env is set to a truthy value, an
// http:// URL is rejected; anything else (https, unset env, env=0)
// is accepted. The test table covers every documented combination.
func TestResolveSyncerURL(t *testing.T) {
	cases := []struct {
		name        string
		envValue    string // empty = unset
		url         string
		wantErr     bool
		wantErrHint string
	}{
		{
			name:     "https url is always accepted",
			envValue: "",
			url:      "https://workspace_syncer:8443",
		},
		{
			name:     "http url accepted when env unset",
			envValue: "",
			url:      "http://workspace_syncer:8080",
		},
		{
			name:     "http url accepted when env=0",
			envValue: "0",
			url:      "http://workspace_syncer:8080",
		},
		{
			name:     "http url accepted when env=false",
			envValue: "false",
			url:      "http://workspace_syncer:8080",
		},
		{
			name:        "http url rejected when env=1",
			envValue:    "1",
			url:         "http://workspace_syncer:8080",
			wantErr:     true,
			wantErrHint: "WORKSPACE_SYNCER_URL_REQUIRE_TLS",
		},
		{
			name:        "http url rejected when env=true",
			envValue:    "true",
			url:         "http://workspace_syncer:8080",
			wantErr:     true,
			wantErrHint: "WORKSPACE_SYNCER_URL_REQUIRE_TLS",
		},
		{
			name:        "http url rejected when env=yes",
			envValue:    "yes",
			url:         "http://workspace_syncer:8080",
			wantErr:     true,
			wantErrHint: "WORKSPACE_SYNCER_URL_REQUIRE_TLS",
		},
		{
			name:     "https url accepted when env=1",
			envValue: "1",
			url:      "https://workspace_syncer:8443",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("WORKSPACE_SYNCER_URL_REQUIRE_TLS", tc.envValue)
			if tc.envValue == "" {
				clearEnv(t, "WORKSPACE_SYNCER_URL_REQUIRE_TLS")
			}

			got, err := resolveSyncerURL(tc.url)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (url=%q)", got)
				}
				if tc.wantErrHint != "" && !strings.Contains(err.Error(), tc.wantErrHint) {
					t.Errorf("error %q must contain hint %q", err.Error(), tc.wantErrHint)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.url {
				t.Errorf("got url %q, want %q", got, tc.url)
			}
		})
	}
}

// clearEnv unsets a single env var for the duration of the test and
// restores it on cleanup. t.Setenv with empty value records "" in
// the post-test snapshot, which is NOT the same as unset for
// truthiness checks (e.g. strconv.ParseBool on "" returns an error).
func clearEnv(t *testing.T, key string) {
	t.Helper()
	original, present := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("unsetenv %s: %v", key, err)
	}
	t.Cleanup(func() {
		if present {
			_ = os.Setenv(key, original)
		} else {
			_ = os.Unsetenv(key)
		}
	})
}
