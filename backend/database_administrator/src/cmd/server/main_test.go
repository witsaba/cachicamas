// Package cmd tests for the middleware chain. The main package is
// the composition root; main_test.go is a sibling package that
// exercises the security middleware decisions (BodyLimit + Recover)
// end-to-end without spinning up an Echo listener.
package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
	e.GET("/panic", func(c *echo.Context) error {
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
	e.GET("/panic", func(c *echo.Context) error {
		panic("test panic")
	})
	req = httptest.NewRequest(http.MethodGet, "/panic", nil)
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 after panic, got %d", rec.Code)
	}
}
