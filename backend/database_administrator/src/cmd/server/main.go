// Command server is the composition root for the database administrator
// service: it wires structured logging, the OpenTelemetry tracing and
// logging pipelines, the migration runner (which runs BEFORE Echo
// binds, per spec R-DBMIG-050), and the HTTP transport adapter; then
// starts the Echo server.
//
// Migration timing (locked at design §5 + spec R-DBMIG-050): the
// migration runner fires from main.go before Echo binds the listener,
// under a bounded context (MIGRATION_TIMEOUT, default 30s). A failed
// migration returns an error from service.Up; main.go emits an
// slog.Error line and calls os.Exit(1). The container orchestrator
// restarts the container — silent corruption is worse than a loud
// crash (the DB schema is the contract the application depends on).
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	otelglobal "go.opentelemetry.io/otel"

	"github.com/cachicamas/backend/database_administrator/src/application"
	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres"
	"github.com/cachicamas/backend/database_administrator/src/migration"
	migrationpg "github.com/cachicamas/backend/database_administrator/src/migration/postgres"
	"github.com/cachicamas/backend/database_administrator/src/otel"
)

// Build metadata is baked into the binary by the Dockerfile via -ldflags.
// Defaults keep `go run` working without a build step.
var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildTime    = "unknown"
)

const (
	serviceName        = "database_administrator"
	defaultServicePort = "8080"
)

// envDuration reads a duration env var (e.g. "30s", "1m500ms"); an
// empty string or a parse failure returns def. We use this for
// MIGRATION_TIMEOUT so the operator can tune the migration window
// without rebuilding the binary.
func envDuration(key string, def time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

// envString returns the value of an env var, or def if unset/empty.
// Centralized so the composition root doesn't sprinkle os.Getenv.
func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// resolveCORSAllowOrigins picks the CORS allowlist for this run.
// Activation rules (see httpiface/cors.go):
//
//   - CORS_ALLOW_ORIGINS set   → split on `,`, trim, keep non-empty.
//                                 Works regardless of SERVICE_ENV;
//                                 safest knob for production-like
//                                 staging where you need a specific
//                                 allowlist.
//   - SERVICE_ENV=development  → default to http://localhost:5173
//                                 so `pnpm dev` Just Works.
//   - Anything else            → returns nil (CORS disabled).
func resolveCORSAllowOrigins() []string {
	if v := os.Getenv("CORS_ALLOW_ORIGINS"); v != "" {
		var out []string
		for _, o := range strings.Split(v, ",") {
			if s := strings.TrimSpace(o); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	if envString("SERVICE_ENV", "development") == "development" {
		return []string{"http://localhost:5173"}
	}
	return nil
}

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

	// ----------------------------------------------------------------
	// Pre-Echo migration hook (design §5, spec R-DBMIG-050).
	//
	// Order matters: the runner must complete (success or loud crash)
	// BEFORE Echo binds the listener. The DB schema is the contract
	// every request handler depends on; serving traffic on a broken
	// schema is worse than refusing to start at all.
	//
	// The runner is reached through three layers (hexagonal rule from
	// design §3 / tasks.md hard constraints):
	//
	//   migration/postgres.Open  -- the only file importing pgx
	//     -> migration.NewGooseRunner   -- the only file importing goose
	//        -> application.NewMigrationService -- OTel+slog wrapper
	//
	// main.go imports the public surface of all three packages; it
	// does NOT import pressly/goose or jackc/pgx directly. The
	// service.Up call returns ([]domain.Version, error); we log the
	// applied count and exit 1 on any non-nil error.
	// ----------------------------------------------------------------

	dbCfg, err := migrationpg.LoadConfigFromEnv()
	if err != nil {
		slog.Error("migration config load failed; exiting", "error", err)
		os.Exit(1)
	}
	db, err := migrationpg.Open(ctx, dbCfg)
	if err != nil {
		slog.Error("migration driver open failed; exiting", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.Close(); err != nil {
			slog.Error("db close", "error", err)
		}
	}()

	runner := migration.NewGooseRunner(db, envString("MIGRATION_TABLE", "schema_migrations"), logger)
	service := application.NewMigrationService(runner, logger, otelglobal.Tracer(serviceName))

	migrateCtx, cancel := context.WithTimeout(ctx, envDuration("MIGRATION_TIMEOUT", 30*time.Second))
	defer cancel()
	applied, err := service.Up(migrateCtx)
	if err != nil {
		slog.Error("migration.up failed; exiting", "error", err)
		os.Exit(1)
	}
	slog.Info("migration.up ok", "applied_count", len(applied))

	// HTTP server with the OTel middleware installed globally so every
	// route emits a span (and the span is correlated with any slog
	// record that carries a request context).
	e := echo.New()
	e.Use(otel.Middleware(serviceName))
	httpiface.RegisterHealthRoute(e)

	// CORS: cross-origin requests from the Qwik dev server
	// (http://localhost:5173 by default) need an explicit allowlist
	// so the browser doesn't block the POST round-trip.  See
	// httpiface/cors.go for the activation rules.
	corsOrigins := resolveCORSAllowOrigins()
	if len(corsOrigins) > 0 {
		slog.Info("cors enabled",
			"service_env", envString("SERVICE_ENV", "development"),
			"allowed_origins", corsOrigins,
		)
		e.Use(httpiface.CORS(httpiface.CORSMiddlewareConfig{AllowOrigins: corsOrigins}))
	} else {
		slog.Info("cors disabled (same-origin expected)",
			"service_env", envString("SERVICE_ENV", "development"),
		)
	}

	// Organizations API: wire the pgx-backed repository (sharing the
	// same *sql.DB the migration runner just used) into the
	// application service, then mount the HTTP handler. No new
	// dependencies; no new pool.
	orgRepo := postgres.NewOrgRepo(db)
	orgService := application.NewOrganizationService(orgRepo, logger, otelglobal.Tracer(serviceName))
	httpiface.RegisterOrganizationRoutes(e, orgService)

	port := envString("SERVICE_PORT", defaultServicePort)
	addr := ":" + port
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
