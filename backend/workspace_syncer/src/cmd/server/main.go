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
	"strconv"
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

	// serverReadTimeout is the per-request read ceiling (covers
	// the request body too). 30s bounds a normal clone request
	// without holding open slow connections. Spec
	// http-server-hardening REQ-01.
	serverReadTimeout = 30 * time.Second

	// serverWriteTimeout is the per-request write ceiling. 30s
	// caps the response time for the clone + probe pipeline.
	serverWriteTimeout = 30 * time.Second

	// serverIdleTimeout is the keep-alive idle ceiling. 120s
	// reuses connections for fast follow-up requests while
	// preventing long-lived idle sockets from accumulating.
	serverIdleTimeout = 120 * time.Second

	// serverMaxHeaderBytes caps the size of the request header
	// block. 1 MiB is generous for any legitimate clone request
	// (the body is bounded separately by MaxBytesReader) and
	// small enough to deter header-bomb attacks.
	serverMaxHeaderBytes = 1 << 20

	// shutdownTimeout bounds the graceful-shutdown window. After
	// this, the server force-closes any in-flight connections.
	shutdownTimeout = 10 * time.Second
)

// maxConcurrentClonesEnv is the env var name for tuning the
// bounded clone semaphore (spec audit M-3). The default
// (httphandler.DefaultMaxConcurrentClones) is 4; operators can
// raise it on a larger host, lower it on a smaller one. The
// handler rejects overflow with 503 + Retry-After.
const maxConcurrentClonesEnv = "WORKSPACE_SYNCER_MAX_CONCURRENT_CLONES"

// resolveMaxConcurrentClones reads the env var and applies the
// default when unset or invalid. A non-positive or non-numeric
// value falls back to httphandler.DefaultMaxConcurrentClones.
func resolveMaxConcurrentClones() int {
	raw := os.Getenv(maxConcurrentClonesEnv)
	if raw == "" {
		return httphandler.DefaultMaxConcurrentClones
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		return httphandler.DefaultMaxConcurrentClones
	}
	return n
}

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

	// Read the HMAC secret for the sync-callback. This is a
	// separate secret from the bearer token (defense in depth +
	// independent rotation). MUST match the value in the
	// database_administrator's SYNC_CALLBACK_SECRET env.
	callbackSecret := os.Getenv("SYNC_CALLBACK_SECRET")
	if callbackSecret == "" {
		fmt.Fprintln(os.Stderr, "FATAL: SYNC_CALLBACK_SECRET is required and must be non-empty")
		os.Exit(2)
	}

	port := os.Getenv("WORKSPACE_SYNCER_PORT")
	if port == "" {
		port = defaultPort
	}

	// Wire observability. PR-2a uses a no-op tracer and a basic JSON
	// slog handler. PR-2c upgrades the logger to the redaction
	// handler (security-critical). The real OTLP tracer is a
	// follow-up (deps bloat deferred).
	otelTracer := otel.NewTracer() // consumed in PR-2c by the handler
	_ = otelTracer
	logger := otel.NewLogger()
	logger.Info("workspace_syncer boot",
		slog.String("port", port),
		slog.String("otel.tracer", "noop (real OTLP deferred to follow-up)"),
	)

	// Build the Echo instance with the wired CloneHandler.
	e := newEcho(tokenStr, callbackSecret, logger)

	// Run the server with graceful shutdown on SIGINT/SIGTERM.
	// The ctx is the parent for both the server run and the
	// shutdown deadline; cancelling it triggers the http.Server
	// Shutdown path.
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Startup sweep (PR-2c): remove orphan cloned directories
	// whose workspace_id no longer has a live sync_job. Runs
	// in a goroutine so it does not block the server boot.
	// PR-3b wires the liveIDs lookup to call database_administrator's
	// /internal/live-workspaces endpoint; for now we pass an
	// empty map (the sweep is a no-op on first deploy).
	go runStartupSweep(ctx, logger)

	if err := runServer(ctx, e, port, logger); err != nil {
		logger.Error("workspace_syncer exited with error",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
}

// runStartupSweep performs the cleanup sweep on startup. The
// sweep walks the data dir and removes any directory whose
// workspace_id is not in the liveIDs map.
//
// PR-2c: the liveIDs lookup is hardcoded to an empty map (the
// real lookup against database_administrator's
// /internal/live-workspaces endpoint lands in PR-3b). This
// means PR-2c's sweep is effectively a no-op for production
// (it just logs the data dir as "empty"). The sweep code is
// in place; PR-3b only adds the lookup.
//
// The sweep runs in a goroutine so the server boot is not
// blocked. The sweep is bounded by git.DefaultSweepTimeout
// (30s); a slow sweep logs a warning and exits.
func runStartupSweep(ctx context.Context, logger *slog.Logger) {
	dataDir := os.Getenv("WORKSPACE_SYNCER_DATA_DIR")
	if dataDir == "" {
		dataDir = "/data/workspaces"
	}
	liveIDs := map[int64]bool{} // TODO(PR-3b): fetch from database_administrator
	if err := git.Sweep(ctx, dataDir, liveIDs, git.OSFS{}, logger); err != nil {
		logger.WarnContext(ctx, "startup sweep failed (non-fatal; the server continues)",
			slog.String("data_dir", dataDir),
			slog.String("error", err.Error()),
		)
	}
}

// newEcho constructs the Echo instance with the bearer-token
// middleware, the liveness probe, and the CloneHandler route.
// Exported as a function (not inlined in main) so the test in
// main_test.go can construct the same instance without spinning
// up a real HTTP server.
func newEcho(serviceToken, callbackSecret string, logger *slog.Logger) *echo.Echo {
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
	callback := httpclient.NewCallbackClient(dbAdminURL, callbackSecret)
	// PR-2b uses nil for the GitHub accessor (the real GitHub
	// client lands in PR-2c). The use case skips permission
	// validation when the accessor is nil.
	svc := application.NewCloneService(runner, callback, nil, logger)
	// Spec audit M-3: bounded clone semaphore. The env-driven
	// value lets operators tune the cap without a code change.
	cloneHandler := httphandler.NewCloneHandlerWithLimit(svc, logger, resolveMaxConcurrentClones())
	cloneHandler.Register(e)

	return e
}

// newHardenedServer builds the production *http.Server with
// explicit timeouts and a header-size cap. Extracted as a
// standalone function so main_test.go can assert the struct
// literal fields against spec http-server-hardening REQ-01.
//
// Per spec REQ-01 the server MUST declare ReadTimeout,
// WriteTimeout, IdleTimeout, and MaxHeaderBytes. The values
// are sourced from the package constants (30s, 30s, 120s, 1<<20)
// so a future tuning change updates both prod and tests in lockstep.
func newHardenedServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       serverReadTimeout,
		WriteTimeout:      serverWriteTimeout,
		IdleTimeout:       serverIdleTimeout,
		MaxHeaderBytes:    serverMaxHeaderBytes,
	}
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
	srv := newHardenedServer(addr, e)


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

var _ application.CloneService // keep the import alive
