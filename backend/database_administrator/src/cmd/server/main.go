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
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
	otelglobal "go.opentelemetry.io/otel"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
	githubinfra "github.com/cachicamas/backend/database_administrator/src/infrastructure/github"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres"
	promptspg "github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/prompts"
	skillspg "github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/skills"
	workspacesyncer "github.com/cachicamas/backend/database_administrator/src/infrastructure/workspacesyncer"
	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
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
//     Works regardless of SERVICE_ENV;
//     safest knob for production-like
//     staging where you need a specific
//     allowlist.
//   - SERVICE_ENV=development  → default to http://localhost:5173
//     so `pnpm dev` Just Works.
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

// resolveSyncerURL encodes the WORKSPACE_SYNCER_URL_REQUIRE_TLS opt-in
// policy (security M-3). When the env is set to a truthy value
// ("1", "true", "yes") an http:// URL is rejected; otherwise the URL
// is returned verbatim. The default URL flips to https only when an
// operator actively opts in — the v1 plain-HTTP deployment still works
// until the workspace_syncer TLS change lands in slice B.
//
// Documented values:
//   WORKSPACE_SYNCER_URL_REQUIRE_TLS=0  (or unset) → http:// accepted
//   WORKSPACE_SYNCER_URL_REQUIRE_TLS=1            → http:// rejected
//   WORKSPACE_SYNCER_URL_REQUIRE_TLS=true         → http:// rejected
//   WORKSPACE_SYNCER_URL_REQUIRE_TLS=yes          → http:// rejected
func resolveSyncerURL(rawURL string) (string, error) {
	requireTLS := false
	if v := os.Getenv("WORKSPACE_SYNCER_URL_REQUIRE_TLS"); v != "" {
		parsed, err := parseBoolLoose(v)
		if err != nil {
			// Treat malformed values as "off" but log the discrepancy.
			slog.Warn("WORKSPACE_SYNCER_URL_REQUIRE_TLS: invalid value, treating as false",
				slog.String("value", v),
				slog.String("hint", "use 1, true, yes, 0, false, or no"),
			)
		} else {
			requireTLS = parsed
		}
	}
	if requireTLS && strings.HasPrefix(strings.ToLower(rawURL), "http://") {
		return "", fmt.Errorf(
			"resolveSyncerURL: WORKSPACE_SYNCER_URL_REQUIRE_TLS=1 but URL is http:// (%s); "+
				"either unset the env, flip the URL to https://, or set the env to 0",
			rawURL,
		)
	}
	return rawURL, nil
}

// parseBoolLoose accepts strconv.ParseBool's documented values plus
// "yes" / "no" (the common shell idiom). Anything else returns an
// error so the caller can log + fall back to false.
func parseBoolLoose(v string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "yes":
		return true, nil
	case "no":
		return false, nil
	}
	return strconv.ParseBool(v)
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
	//
	// Security middleware (security M-1 + M-4 + design §3):
	//   - middleware.Recover() — catches any handler panic and returns
	//     a 500 envelope instead of crashing the process. Production
	//     stability without this is a single buggy handler away.
	//   - middleware.BodyLimit(1 << 20) — caps request bodies at 1 MiB
	//     so a malicious / buggy client can't OOM the process by
	//     streaming a 10 GiB body. Mirrors the workspace_syncer's
	//     MaxBytesReader cap (slice B).
	//
	// Recover is installed FIRST so the body-limit handler still
	// benefits from the panic-recovery net.
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit(1 << 20))
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

	// Identity (cachicamas-github-login PR-3): wire the identity
	// repository + the JWE-cookie verifier middleware. The middleware
	// is registered AFTER CORS and BEFORE the protected route group,
	// so a missing / tampered cookie short-circuits with 401 before
	// the handler is invoked. AUTH_SECRET is the same value the
	// frontend uses to encrypt the cookie (per ADR 0002 byte-level
	// envelope contract).
	authSecret := envString("AUTH_SECRET", "")
	if authSecret == "" {
		slog.Error("AUTH_SECRET must be set; exiting (cachicamas-github-login S-BAM-080)")
		os.Exit(1)
	}
	cookieName := envString("AUTH_COOKIE_NAME", "authjs.session-token")
	if httpiface.IsLikelyHTTPS(envString("ORIGIN", "")) {
		cookieName = "__Secure-authjs.session-token"
	}
	identityRepo := postgres.NewIdentityRepo(db)
	identityService := application.NewIdentityService(identityRepo, logger, otelglobal.Tracer(serviceName))
	httpiface.RegisterProtectedWhoAmIRoute(e, identityService, httpiface.IdentityMiddlewareConfig{
		AuthSecret:     authSecret,
		CookieName:     cookieName,
		IdentityRepo:   identityRepo,
		Logger:         logger,
		TracerProvider: otelglobal.GetTracerProvider(),
	})

	// Workspaces (2026-07-06): auth middleware chain MUST be
	// IdentityFromCookie → loadGitHubTokenForIdentity → ... route
	// registrations. The chain is built once and passed to
	// RegisterAuthenticatedWorkspaceRoutes; the function mounts the
	// 8 workspace endpoints + /github/repos inside an Echo sub-group
	// that runs the chain before the handlers. This is the
	// COMPILER-enforced wire contract — the old `e.Use(tokenMW)` +
	// `RegisterWorkspaceRoutes(e,...)` shape silently omitted
	// IdentityFromCookie and let anonymous requests reach the
	// handlers (symptom: 400 "auth: Authentication required." on
	// /home, observed 2026-07-07).
	tokenFetcher := postgres.NewTokenFetcher(db)
	identityMW := httpiface.IdentityFromCookie(httpiface.IdentityMiddlewareConfig{
		AuthSecret:     authSecret,
		CookieName:     cookieName,
		IdentityRepo:   identityRepo,
		Logger:         logger,
		TracerProvider: otelglobal.GetTracerProvider(),
	})
	authChain := []echo.MiddlewareFunc{
		identityMW,
		httpiface.LoadGitHubTokenMiddleware(tokenFetcher, logger),
	}

	// Workspaces HTTP routes: 8 endpoints + the github proxy.
	// Wire the pgx-backed WorkspaceRepo → WorkspaceService →
	// WorkspaceHandler, then the GitHub client + cache →
	// GitHubHandler. The githubClient is wrapped in a WorkspaceGitHubAccessor
	// adapter so WorkspaceService (which doesn't import http) can call it.
	workspaceRepo := postgres.NewWorkspaceRepo(db)
	githubClient := githubinfra.NewClient()

	// 2026-07-08-workspace-sync-clone PR-3a: wire the sync_job repo
	// and the SyncService. WorkspaceService.Create auto-enqueues
	// the first sync job via the SyncService. PR-3b added the
	// workspaceRepo + db params (the callback path needs Tx access).
	syncJobRepo := postgres.NewSyncJobRepo(db)
	syncSvc := application.NewSyncService(syncJobRepo, workspaceRepo, db, logger)

	workspaceService := application.NewWorkspaceService(
		workspaceRepo,
		syncSvc,
		httpiface.NewWorkspaceGitHubAccessor(githubClient),
		logger,
		otelglobal.Tracer(serviceName),
	)

	// GitHub repos proxy (R-WS-009): 5-min in-memory cache keyed by
	// user_id. Cache TTL configurable via GITHUB_REPOS_CACHE_TTL env
	// (default 5m).
	cacheTTL := envDuration("GITHUB_REPOS_CACHE_TTL", 5*time.Minute)
	repoCache := githubinfra.NewRepoCache(cacheTTL)
	githubHandler := httpiface.NewGitHubHandler(
		repoCache,
		githubClient,
		func(c *echo.Context) (int64, bool) {
			raw := c.Get(httpiface.IdentityContextKey)
			id, ok := raw.(*domain.Identity)
			if !ok {
				return 0, false
			}
			return id.ID, true
		},
		logger,
	)

	// Compile-time guarantee that every workspace + GitHub-proxy
	// route runs the full auth chain (see the regression test in
	// src/interfaces/http/workspaces_auth_chain_test.go).
	httpiface.RegisterAuthenticatedWorkspaceRoutes(
		e, workspaceService, githubHandler, authChain, logger,
	)

	// 2026-07-08-workspace-sync-clone PR-3b: wire the workspace_syncer
	// HTTP client + the sync endpoints. The client is constructed
	// with the shared INTERNAL_SERVICE_TOKEN; the URL defaults to
	// http://workspace_syncer:8080 (the docker-compose service
	// name; overridable for dev).
	//
	// Security M-3: WORKSPACE_SYNCER_URL_REQUIRE_TLS=1 enforces that
	// the URL is https:// — when the env is set and an http:// URL is
	// supplied the service refuses to start (fail-fast), so the
	// cleartext path is impossible to deploy by mistake.
	syncerBaseURL, err := resolveSyncerURL(envString("WORKSPACE_SYNCER_URL", "http://workspace_syncer:8080"))
	if err != nil {
		slog.Error("WORKSPACE_SYNCER_URL rejected; exiting", slog.String("error", err.Error()))
		os.Exit(1)
	}
	syncerToken := envString("INTERNAL_SERVICE_TOKEN", "")
	if syncerToken == "" {
		slog.Error("INTERNAL_SERVICE_TOKEN must be set; exiting (workspace_syncer auth)")
		os.Exit(1)
	}
	syncerClient := workspacesyncer.NewClient(syncerBaseURL, syncerToken)
	syncerDispatcher := httpiface.NewWSClientAdapter(syncerClient)

	// PR-3c: pass workspaceRepo (WorkspaceRowLoader) + tokenFetcher
	// (TokenFetcher) to the SyncHandler so it can populate the
	// syncer dispatch with the real owner/repo/default_branch/
	// oauth_token. Both are already wired in main.go (the
	// workspaceRepo from PR-1; the tokenFetcher from PR1c-ii).
	syncHandler := httpiface.NewSyncHandler(syncSvc, workspaceRepo, tokenFetcher, syncerDispatcher, logger)

	// Mount the sync endpoints inside the auth-protected workspace
	// group. We need to attach the routes to the same Echo sub-group
	// that already runs the authChain; RegisterAuthenticatedWorkspaceRoutes
	// does not return it, so we re-create the group with the same
	// middleware and register the sync routes on the group.
	syncGroup := e.Group("", authChain...)
	httpiface.RegisterSyncRoutes(syncGroup, syncHandler)

	// SSE stream for live sync_job updates (UAT fix 2026-07-08:
	// replaces the fragile polling that failed to propagate signal
	// updates through QRL closures). Mounted on the same
	// auth-protected group as the other sync routes so the
	// IdentityFromCookie middleware applies. The prior wiring
	// mounted the SSE on the root Echo (no authChain), so a
	// request with a valid session cookie returned 400
	// "Authentication required" — see
	// sync_stream_handler_test.go:TestSSE_RejectsUnauthenticatedRequests
	// for the regression guard.
	streamHandler := httpiface.NewSyncStreamHandler(syncSvc, logger)
	httpiface.RegisterSyncStreamRoute(syncGroup, streamHandler)

	// Internal sync callback receiver (PR-3b). Mounted on the public
	// group; HMAC + anti-replay window. Secret is independent of
	// IDENTITY_CALLBACK_SECRET (defense in depth: a leak of one does
	// not compromise the other).
	syncCallbackSecret := envString("SYNC_CALLBACK_SECRET", "")
	if syncCallbackSecret == "" {
		slog.Error("SYNC_CALLBACK_SECRET must be set; exiting")
		os.Exit(1)
	}
	if len(syncCallbackSecret) < 32 {
		slog.Error("SYNC_CALLBACK_SECRET must be at least 32 raw bytes; exiting",
			"length", len(syncCallbackSecret),
			"hint", "generate with `openssl rand -base64 32`",
		)
		os.Exit(1)
	}
	httpiface.RegisterInternalSyncCallbackRoute(e, syncSvc, syncCallbackSecret, logger)
	// 2026-07-08-workspace-sync-clone PR-3a: hold a reference to
	// the syncSvc so PR-3b can mount the sync endpoints.
	_ = syncSvc
	// Single-tenant resolver hook (used by the workspace handlers).
	httpiface.SetSingleTenantOrgIDResolver(func() int64 { return 1 })

	// Identity signin-callback (cachicamas-identity-signin-callback):
	// HMAC-signed POST endpoint from the Qwik frontend's events.signIn
	// callback. Mounted on the public/internal route group; NOT under
	// the JWE verifier middleware (PR-3) — this endpoint has its own
	// HMAC + timestamp anti-replay window. See
	// docs/adr/0003-add-identity-callback-hmac.md for the wire contract.
	callbackSecret := envString("IDENTITY_CALLBACK_SECRET", "")
	if callbackSecret == "" {
		slog.Error("IDENTITY_CALLBACK_SECRET must be set; exiting (cachicamas-identity-signin-callback)")
		os.Exit(1)
	}
	if len(callbackSecret) < 32 {
		slog.Error("IDENTITY_CALLBACK_SECRET must be at least 32 raw bytes; exiting",
			"length", len(callbackSecret),
			"hint", "generate with `openssl rand -base64 32`",
		)
		os.Exit(1)
	}
	httpiface.RegisterIdentityCallbackRoute(e, identityService, callbackSecret, logger)

	// 2026-07-08-workspace-sync-clone PR-3b: mount the syncer's
	// internal sync-callback receiver on the same public group as
	// the identity callback. Both endpoints share the HMAC + timestamp
	// anti-replay pattern but use independent secrets.

	// 2026-07-15-prompt-storage-table PR 4 of 4: wire the prompts
	// HTTP surface. Admin-only, no extra header (Q-D locked). The
	// routes mount on the public Echo group; in production they
	// should sit behind the compose internal network only.
	promptRepo := promptspg.NewPromptRepo(db)
	promptRevRepo := promptspg.NewPromptRevisionRepo(db)
	promptService := application.NewPromptService(promptRepo, promptRevRepo, db, logger)
	promptHandler := httpiface.NewPromptHandler(promptService, logger)
	promptHandler.RegisterPromptRoutes(e)

	// 2026-07-17-skills-foundational PR1d of 7: wire the skills
	// HTTP surface. Admin-only, no extra header (mirrors prompts).
	// The 7 routes mount on the public Echo group; the SQL JOIN
	// that emits current_revision (anti-drift gate ADR-SK-008)
	// lives in SkillRepo.ListWithCurrentRevision / SelectByIDWithCurrentRevision.
	skillRepo := skillspg.NewSkillRepo(db)
	skillRevRepo := skillspg.NewSkillRevisionRepo(db)
	skillService := application.NewSkillService(skillRepo, skillRevRepo, db, logger)
	skillHandler := httpiface.NewSkillHandler(skillService, logger)
	skillHandler.RegisterSkillRoutes(e)

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
