// Package httpiface exposes HTTP transport adapters (handlers, routes) for the
// database administrator service. It depends on the application layer and on
// the Echo web framework.
package httpiface

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
)

// HealthHandler exposes health check use cases over HTTP.
type HealthHandler struct {
	service *application.HealthService
}

// NewHealthHandler wires a HealthHandler to its application service.
func NewHealthHandler(service *application.HealthService) *HealthHandler {
	return &HealthHandler{service: service}
}

// Check handles GET /health.
//
// DEV-ONLY fail injection: if the query string contains `fail=true` AND
// SERVICE_ENV=development, the handler returns HTTP 500. Used by
// scripts/human-run-tail-sampling-verify.sh (test 3.1) to generate error
// spans against a live stack. Has no effect in any other SERVICE_ENV.
func (h *HealthHandler) Check(c *echo.Context) error {
	if c.QueryParam("fail") == "true" && os.Getenv("SERVICE_ENV") == "development" {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"status": "fail-injected",
		})
	}
	// slog.Info is called with the request context so the otelslog bridge
	// can attach trace_id / span_id to the resulting log record. The
	// record flows through the OTel pipeline and ends up correlated with
	// the GET /health span emitted by the otel.Middleware in main.go.
	slog.InfoContext(c.Request().Context(), "health check served")
	report := h.service.Check()
	return c.JSON(http.StatusOK, report)
}

// RegisterHealthRoute wires the /health route on the given Echo instance.
// Exported so tests can build a minimal app without booting a real server.
func RegisterHealthRoute(e *echo.Echo) {
	h := NewHealthHandler(application.NewHealthService())
	e.GET("/health", h.Check)
}
