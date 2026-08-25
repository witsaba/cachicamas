// Package main is the chat archetype's composition root. CH-04 wires
// every required dependency from the environment, installs the OTel
// SDK (in otel.go), builds the openrouter provider, builds the JWE
// IdentityResolver shim (in auth_shim.go), wires the chat HTTP
// surface, and binds Echo with graceful shutdown on SIGTERM/SIGINT.
//
// CH-04 is the ONLY package permitted to install the OpenTelemetry
// SDK and its exporters (ADR 0005 § D3, adr:242; ADR 0009 § D2
// substitution; chat/doc.go:75-81). It is the ONLY package that
// reads the environment (R-06) — every other chat package receives
// its dependencies from this root.
//
// Nothing imports this package, and nothing can: Go forbids importing
// a package main. That is the whole mechanism behind the "one
// composition root, imported by nothing" half of R-06 — enforced by
// the compiler, so this change ships no test asserting it.
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter"
	"github.com/cachicamas/backend/agent/src/chat"
	"github.com/cachicamas/backend/agent/src/chat/migrations"
	"github.com/cachicamas/backend/agent/src/chat/migrator"

	// pgx stdlib adapter registers "pgx" as a database/sql driver.
	// Blank import: side-effect only (driver's init()), no runtime
	// cost. Mirrors the DBA precedent at
	// backend/database_administrator/src/migration/postgres/driver.go:30-33.
	// ADR 0010-admit pgx + goose in backend/agent is the dep
	// admission; this is the chat composition root's first
	// production import site.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// config is the validated runtime configuration the composition root
// reads from the environment. Every required field is validated in
// loadConfig; a missing or invalid value yields a named error before
// any listener is bound.
type config struct {
	// Port is the TCP port Echo binds to. Default: 8080.
	Port string

	// ProviderAPIKey is the bearer token openrouter.NewProvider uses
	// for every outbound request. Required.
	ProviderAPIKey string

	// ProviderModel is the wire-side model identifier openrouter
	// substitutes on every request body. Required.
	ProviderModel string

	// AuthSecret is the raw AUTH_SECRET the JWE shim derives its HKDF
	// key from. MUST be ≥32 bytes (S-BAM-081). Required.
	AuthSecret string

	// CookieName is the Auth.js session-token cookie name. Default:
	// "authjs.session-token".
	CookieName string

	// OTLPEndpoint is the OTLP/gRPC collector endpoint the OTel SDK
	// exports to. Default: localhost:4317 (read from the standard
	// OTEL_EXPORTER_OTLP_ENDPOINT env var by otlptracegrpc itself).
	OTLPEndpoint string

	// ChatStoreDSN is the Postgres connection string the chat
	// adapter dials through. Required — the chat binary refuses to
	// start without it (mirrors DBA's LoadConfigFromEnv pattern).
	// Read from CACHICAMAS_CHAT_STORE_DSN. CH-07 (closes R-08):
	// the chat archetype owns its own tables, and the only durable
	// surface for conversations is the postgres adapter behind
	// the closed ConversationStore port.
	ChatStoreDSN string
}

// envString returns the value of an env var, or def if unset/empty.
// Centralized so the composition root doesn't sprinkle os.Getenv.
// Mirrors database_administrator/src/cmd/server/main.go:75.
func envString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// loadConfig reads every required env var via getenv, validates the
// byte-length gates, and returns a populated config. Every required
// var's absence yields an error whose message names the missing var so
// the operator sees which key to set in the stderr line main prints.
//
// getenv is injected so tests can drive the validation paths without
// touching the real process environment.
func loadConfig(getenv func(string) string) (config, error) {
	cfg := config{
		Port:           envStringFrom(getenv, "CACHICAMAS_CHAT_PORT", "8080"),
		ProviderAPIKey: getenv("CACHICAMAS_CHAT_PROVIDER_API_KEY"),
		ProviderModel:  getenv("CACHICAMAS_CHAT_PROVIDER_MODEL"),
		AuthSecret:     getenv("AUTH_SECRET"),
		CookieName:     envStringFrom(getenv, "AUTH_COOKIE_NAME", "authjs.session-token"),
		OTLPEndpoint:   getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		ChatStoreDSN:   getenv("CACHICAMAS_CHAT_STORE_DSN"),
	}

	if cfg.ProviderAPIKey == "" {
		return cfg, fmt.Errorf("missing required environment variable: CACHICAMAS_CHAT_PROVIDER_API_KEY (the chat binary refuses to start without a provider credential)")
	}
	if cfg.ProviderModel == "" {
		return cfg, fmt.Errorf("missing required environment variable: CACHICAMAS_CHAT_PROVIDER_MODEL (the chat binary refuses to start without a model selection)")
	}
	if cfg.AuthSecret == "" {
		return cfg, fmt.Errorf("missing required environment variable: AUTH_SECRET (the chat binary refuses to start without the Auth.js JWE encryption key)")
	}
	if len(cfg.AuthSecret) < 32 {
		return cfg, fmt.Errorf("AUTH_SECRET must be at least 32 raw bytes, got %d (S-BAM-081: Auth.js HKDF requires ≥32 bytes)", len(cfg.AuthSecret))
	}
	if cfg.ChatStoreDSN == "" {
		return cfg, fmt.Errorf("missing required environment variable: CACHICAMAS_CHAT_STORE_DSN (the chat binary refuses to start without a postgres DSN; conversations persist in tables the chat archetype owns, per ADR 0009 § D6)")
	}
	return cfg, nil
}

// envStringFrom is the testable twin of envString: it takes the env
// reader as a parameter so loadConfig can drive synthetic envs. The
// production caller (envString) and this helper share the same
// "empty string is treated as unset" semantics so the production path
// and the test path see identical behaviour.
func envStringFrom(getenv func(string) string, key, def string) string {
	if v := getenv(key); v != "" {
		return v
	}
	return def
}

// buildProvider constructs the openrouter provider with the chat
// archetype's attribution headers (HTTPReferer/XTitle/XCategories).
// The OTel TracerProvider is injected so the openaicompat client
// threads spans through the chat binary's OTLP pipeline.
func buildProvider(cfg config, tp trace.TracerProvider) (ai.ModelProvider, error) {
	return openrouter.NewProvider(openrouter.Config{
		Credential:     openaicompat.NewCredential(cfg.ProviderAPIKey),
		Model:          cfg.ProviderModel,
		HTTPReferer:    "cachicamas-chat",
		XTitle:         "Cachicamas Chat",
		XCategories:    "chat",
		TracerProvider: tp,
	})
}

// run is the composition root's actual work, factored out of main() so
// tests can exercise the same wiring path without process exit. It
// returns on graceful shutdown; main() exits 0 if run returns nil and
// 1 otherwise.
//
// Echo v5's e.Start(address) installs its own SIGINT/SIGTERM signal
// handler and performs a graceful shutdown (bounded by
// StartConfig.GracefulTimeout, default 10s) before returning. run()
// waits for that return, then fires the OTel shutdown so any in-flight
// spans are flushed before the TracerProvider tears down.
//
// startAndWait is injected so tests can return immediately without
// binding a real port; main() passes e.Start, which blocks until the
// listener drains. This is the seam that lets
// TestComposeRoot_GracefulShutdown_CallsOTelShutdown exercise the
// post-listener shutdown path (spec scenario 2.2) without owning a
// process.
func run(ctx context.Context, getenv func(string) string, otelShutdown func(context.Context) error, otelTP trace.TracerProvider, logger *slog.Logger, startAndWait func(*echo.Echo, string) error) error {
	cfg, err := loadConfig(getenv)
	if err != nil {
		return err
	}

	provider, err := buildProvider(cfg, otelTP)
	if err != nil {
		return fmt.Errorf("build provider: %w", err)
	}

	resolver := NewResolver([]byte(cfg.AuthSecret), cfg.CookieName)

// CH-07 (closes R-08): the chat-owned Postgres adapter is the
// conversation store. The composition root dials the pool,
// pings, runs the chat-owned forward-only migrations, and
// constructs the adapter behind the same ConversationStore
// port the CH-06 in-memory adapter satisfied. The factory
// closure body below is BYTE-UNCHANGED from CH-06 — that is
// the proof the swap was a swap (R-CCS-010).
	db, err := sql.Open("pgx", cfg.ChatStoreDSN)
	if err != nil {
		return fmt.Errorf("open postgres for chat store: %w", err)
	}
	pingCtx, cancelPing := context.WithTimeout(ctx, 5*time.Second)
	if err := db.PingContext(pingCtx); err != nil {
		cancelPing()
		_ = db.Close()
		return fmt.Errorf("ping postgres for chat store: %w", err)
	}
	cancelPing()

	migrationProvider, err := migrator.NewProvider(ctx, db, migrations.MigrationsFS, "chat_schema_migrations")
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("build chat migration provider: %w", err)
	}
	if _, err := migrationProvider.Up(ctx); err != nil {
		_ = db.Close()
		return fmt.Errorf("apply chat migrations: %w", err)
	}

	chatStore, closeStore, err := chat.NewPostgresConversationStore(ctx, cfg.ChatStoreDSN)
	if err != nil {
		_ = db.Close()
		return fmt.Errorf("build postgres conversation store: %w", err)
	}
	// CH-09.1 (closes R-03, R-09 seam): the chat-owned tool source
	// port wraps Layer 2's agent.Registry. The first tool is
	// current_time (D-2) — proves the seam end-to-end with a
	// fixed-injection clock. The factory closure body below is
	// BYTE-UNCHANGED from CH-08 except for the new ToolSource line
	// in chat.NewConversation's Config literal — R-CTS-001 + D-1 +
	// R-CCS-010 closed-port posture preserved.
	//
	// CH-10.1 (closes R-15 + R-CPM-001/002): the registry gains a
	// second tool — summarize_conversation (D-2 G2, S-CTT-101..104).
	// EffectClassMutate; writes the summary column on
	// chat_conversations via the 4th port method UpdateSummary
	// (R-CCS-017). SummarizeConversationTool is per-conversation
	// (participantID is captured at construction), so the
	// ToolSource is constructed INSIDE the factory closure —
	// CH-09's `toolSource` shared-across-conversations pattern
	// doesn't apply to stateful tools. The factory closure body
	// gains exactly one line: a PermissionPolicy field on the
	// Config literal (R-CPM-001 + S-CPM-003 composition-root-only
	// discipline).
	//
	// CH-10.1 (D-2, G7, R-CPM-002): the chat-owned default
	// permission policy. Defers for tools in deferToolNames; allows
	// everything else synchronously with AllowOnce. Constructed
	// once at composition root (S-CPM-003 discipline); the same
	// value flows into every Conversation.
	permissionPolicy := chat.NewDefaultPermissionPolicy([]string{"summarize_conversation"})

	// closeStore is invoked AFTER otelShutdown so spans flush
	// before the pool tears down (design D-E).
	factory := func(participantID string) (*chat.Conversation, error) {
		exchanges, lerr := chatStore.Load(participantID)
		if lerr != nil {
			if !errors.Is(lerr, chat.ErrConversationNotFound) {
				return nil, lerr
			}
			exchanges = nil
		}
		history, herr := chat.ExchangesToHistory(exchanges)
		if herr != nil {
			return nil, herr
		}
		// CH-10.1: per-conversation ToolSource — the summarizer
		// captures participantID at construction. CurrentTimeTool
		// is stateless, so it shares a single instance across
		// conversations.
		toolSource := chat.FromAgentRegistry(agent.NewMapRegistry(map[string]agent.Tool{
			"current_time":          chat.NewCurrentTimeTool(time.Now),
			"summarize_conversation": chat.NewSummarizeConversationTool(chatStore, participantID),
		}))
		return chat.NewConversation(chat.Config{
			Provider:         provider,
			Store:            chatStore,
			ParticipantID:    participantID,
			InitialHistory:   history,
			ToolSource:       toolSource,
			PermissionPolicy: permissionPolicy,
		})
	}

	e := echo.New()
	registry, err := chat.RegisterRoutes(e, resolver, factory)
	if err != nil {
		return fmt.Errorf("chat.RegisterRoutes: %w", err)
	}
	_ = registry // held for future graceful-shutdown hooks; CH-04 has no per-turn cleanup

	// CH-08 (closes R-14, R-16): the resume read surface — two GET
	// endpoints behind the same identityMiddleware. The store is the
	// same adapter the factory closure consumes; RegisterResumeRoutes
	// only registers the routes (it does not own the store's
	// lifecycle). The factory closure body above is BYTE-UNCHANGED
	// from CH-07 — that is the proof the swap was a swap (R-CCS-010);
	// CH-08 widens the HTTP surface but does not touch the
	// per-participant lifecycle.
	if err := chat.RegisterResumeRoutes(e, resolver, chatStore); err != nil {
		return fmt.Errorf("chat.RegisterResumeRoutes: %w", err)
	}

	logger.Info("chat composition root listening", "address", ":"+cfg.Port)
	startErr := startAndWait(e, ":"+cfg.Port)

	// Echo has drained. Now flush the OTel pipeline so any spans held
	// in the BatchSpanProcessor make it to the collector before exit.
	if otelShutdown != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := otelShutdown(shutdownCtx); err != nil {
			logger.Warn("otel shutdown", "error", err)
		}
	}

	// CH-07: close the postgres pool AFTER OTel shutdown, so spans
	// (which may log pool stats) flush before the pool tears down.
	if err := closeStore(); err != nil {
		logger.Warn("close chat store", "error", err)
	}

	return startErr
}

func main() {
	logger := slog.Default()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	otelShutdown, otelTP, err := installProductionOTelSDK(ctx)
	if err != nil {
		// No logger configured yet — fall back to stderr so the error
		// is visible to the operator before any structured logging.
		fmt.Fprintf(os.Stderr, "chat: OTel install failed: %v\n", err)
		os.Exit(1)
	}

	if err := run(ctx, os.Getenv, otelShutdown, otelTP, logger, startEcho); err != nil {
		fmt.Fprintf(os.Stderr, "chat: %v\n", err)
		os.Exit(1)
	}
}

// installProductionOTelSDK is the production wiring helper: calls
// installOTelSDK and returns both the shutdown closure and the global
// TracerProvider. main() uses both; tests skip OTel entirely (the
// guard the spec requires lives at chat/, not cmd/chat/).
func installProductionOTelSDK(ctx context.Context) (func(context.Context) error, trace.TracerProvider, error) {
	shutdown, err := installOTelSDK(ctx)
	if err != nil {
		return nil, nil, err
	}
	return shutdown, otel.GetTracerProvider(), nil
}

// startEcho is the production startAndWait injected into run(): blocks
// until the Echo listener drains (graceful shutdown on SIGINT/SIGTERM,
// bounded by StartConfig.GracefulTimeout, default 10s). main() uses
// this; tests inject a function that returns immediately to exercise
// the post-listener shutdown path without binding a real port.
func startEcho(e *echo.Echo, address string) error {
	return e.Start(address)
}