// Package main is the composition root for the workspace_syncer
// service. The binary is built with `make build` into ./bin/workspace_syncer
// and run with `make run`.
//
// PR-2a scope: scaffolding only. The POST /internal/clone-and-validate
// route exists but returns 501 Not Implemented. PR-2b replaces the
// 501 with the real handler. PR-2c adds the OTel span, the
// redaction handler, and the sweep on startup.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/workspace_syncer/src/application"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/git"
	httphandler "github.com/cachicamas/backend/workspace_syncer/src/interfaces/http"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/httpclient"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/token"
	"github.com/cachicamas/backend/workspace_syncer/src/otel"
)

const (
	// defaultPort is the port the service binds to when
	// WORKSPACE_SYNCER_PORT is unset. The compose file uses this
	// default (see docker-compose.yaml workspace_syncer service).
	defaultPort = "8081"

	// readHeaderTimeout bounds the time the HTTP server waits for
	// request headers. Without it, slow-loris attacks can hold
	// connections open indefinitely. 5s is the Echo community
	// default; we keep it.
	readHeaderTimeout = 5 * time.Second

	// shutdownTimeout bounds the graceful-shutdown window. After
	// this, the server force-closes any in-flight connections.
	shutdownTimeout = 10 * time.Second
)

func main() {
	// Read the service-to-service bearer token. The token is required:
	// the service fails to boot with a clear error if the env var is
	// empty. This is the first line of defense (the
	// ServiceTokenMiddleware has its own fail-safe; see
	// token/middleware.go).
	tokenStr := os.Getenv("INTERNAL_SERVICE_TOKEN")
	if tokenStr == "" {
		fmt.Fprintln(os.Stderr, "FATAL: INTERNAL_SERVICE_TOKEN is required and must be non-empty")
		os.Exit(2)
	}

	port := os.Getenv("WORKSPACE_SYNCER_PORT")
	if port == "" {
		port = defaultPort
	}

	// Wire observability. PR-2a uses a no-op tracer and a basic JSON
	// slog handler. PR-2c upgrades both.
	otelTracer := otel.NewTracer() // consumed in PR-2c by the handler
	_ = otelTracer
	logger := otel.NewLogger()
	logger.Info("workspace_syncer boot",
		slog.String("port", port),
		slog.String("otel.tracer", "noop (PR-2c upgrades to OTLP)"),
	)

	// Build the Echo instance with the wired CloneHandler.
	e := newEcho(tokenStr, logger)

	// Run the server with graceful shutdown on SIGINT/SIGTERM.
	// The ctx is the parent for both the server run and the
	// shutdown deadline; cancelling it triggers the http.Server
	// Shutdown path.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := runServer(ctx, e, port, logger); err != nil {
		logger.Error("workspace_syncer exited with error",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}

// newEcho constructs the Echo instance with the bearer-token
// middleware, the liveness probe, and the CloneHandler route.
// Exported as a function (not inlined in main) so the test in
// main_test.go can construct the same instance without spinning
// up a real HTTP server.
func newEcho(serviceToken string, logger *slog.Logger) *echo.Echo {
	e := echo.New()

	// Bearer-token middleware is applied to ALL routes via the skipper:
	//   - /healthz is the only public route (compose healthcheck);
	//     the skipper lets it through without the token.
	//   - Every other route is internal-only and requires the token.
	e.Use(token.ServiceTokenMiddleware(serviceToken, token.ServiceTokenMiddlewareConfig{
		Skipper: func(c *echo.Context) bool {
			return c.Request().URL.Path == "/healthz"
		},
	}))

	// Liveness probe. Always 200 if the process is up. The compose
	// file's healthcheck uses this. The route is intentionally
	// public (no token required) so the compose healthcheck can
	// call it without leaking the service token.
	e.GET("/healthz", func(c *echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Wire the clone + callback pipeline. The real production
	// wiring reads DATABASEEADMINISTRATORBASE_URL from the
	// compose env; for now we read it from DATABASE_ADMINISTRATOR_URL.
	dbAdminURL := os.Getenv("DATABASE_ADMINISTRATOR_URL")
	if dbAdminURL == "" {
		dbAdminURL = "http://database_administrator:8080"
	}
	runner := git.NewRunner()
	callback := httpclient.NewCallbackClient(dbAdminURL, serviceToken)
	// PR-2b uses nil for the GitHub accessor (the real GitHub
	// client lands in PR-2c). The use case skips permission
	// validation when the accessor is nil.
	svc := application.NewCloneService(runner, callback, nil, logger)
	cloneHandler := httphandler.NewCloneHandler(svc, logger)
	cloneHandler.Register(e)

	return e
}

// runServer starts the HTTP server on the given port and blocks
// until ctx is cancelled (signal) or the server fails to start. On
// cancellation, the server is given shutdownTimeout to drain
// in-flight requests before being force-closed.
//
// Exported as a function (not inlined in main) so the test in
// main_test.go can construct the Echo instance directly without
// spinning up a real HTTP server.
func runServer(ctx context.Context, e *echo.Echo, port string, logger *slog.Logger) error {
	addr := ":" + port
	srv := &http.Server{
		Addr:              addr,
		Handler:           e,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// Start the server in a goroutine so we can wait for the
	// context to be cancelled (signal) in the main goroutine.
	listenErrCh := make(chan error, 1)
	go func() {
		logger.Info("workspace_syncer listening", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			listenErrCh <- fmt.Errorf("http listen: %w", err)
			return
		}
		listenErrCh <- nil
	}()

	select {
	case <-ctx.Done():
		logger.Info("workspace_syncer shutting down on signal")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	case err := <-listenErrCh:
		return err
	}
}

// Compile-time check that the net package is used (defends against
// future cleanup removing the net import while leaving a stale
// reference in a comment).
var _ = net.IPv4len
