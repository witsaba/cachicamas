// Command server is the composition root for the database administrator
// service: it wires structured logging, the OpenTelemetry tracing and
// logging pipelines, and the HTTP transport adapter; then starts the
// Echo server.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/labstack/echo/v5"

	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
	"github.com/cachicamas/backend/database_administrator/src/otel"
)

// Build metadata is baked into the binary by the Dockerfile via -ldflags.
// Defaults keep `go run` working without a build step.
var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildTime    = "unknown"
)

const serviceName = "database_administrator"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Logging first so subsequent setup errors are captured in the
	// OTel pipeline (and on stderr as a fallback). SetupLogging returns
	// a *slog.Logger; we install it as the default so libraries that
	// call slog.* directly (e.g. OTel SDK internals) also flow through
	// the same handler.
	logger, shutdownLogging, err := otel.SetupLogging(ctx, serviceName, buildVersion)
	if err != nil {
		// No logger yet — fall back to stderr so the error is visible.
		slog.New(slog.NewJSONHandler(os.Stderr, nil)).
			Error("setup logging failed", "error", err)
		os.Exit(1)
	}
	slog.SetDefault(logger)
	defer func() {
		shutdownCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := shutdownLogging(shutdownCtx); err != nil {
			slog.Error("shutdown logging", "error", err)
		}
	}()

	// Tracing: install a global OTLP/gRPC tracer provider and a W3C
	// trace-context propagator. The endpoint comes from
	// OTEL_EXPORTER_OTLP_ENDPOINT (set in docker-compose.yaml).
	shutdownTracing, err := otel.SetupTracing(ctx, serviceName, buildVersion)
	if err != nil {
		slog.Error("setup tracing failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if err := shutdownTracing(shutdownCtx); err != nil {
			slog.Error("shutdown tracing", "error", err)
		}
	}()

	// HTTP server with the OTel middleware installed globally so every
	// route emits a span (and the span is correlated with any slog
	// record that carries a request context).
	e := echo.New()
	e.Use(otel.Middleware(serviceName))
	httpiface.RegisterHealthRoute(e)

	addr := ":8080"
	slog.Info("database_administrator listening",
		"address", addr,
		"version", buildVersion,
		"commit", buildCommit,
		"built", buildTime,
	)
	if err := e.Start(addr); err != nil {
		slog.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
