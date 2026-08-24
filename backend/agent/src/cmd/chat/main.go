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
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/labstack/echo/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter"
	"github.com/cachicamas/backend/agent/src/chat"
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
func run(_ context.Context, getenv func(string) string, otelShutdown func(context.Context) error, otelTP trace.TracerProvider, logger *slog.Logger, startAndWait func(*echo.Echo, string) error) error {
	cfg, err := loadConfig(getenv)
	if err != nil {
		return err
	}

	provider, err := buildProvider(cfg, otelTP)
	if err != nil {
		return fmt.Errorf("build provider: %w", err)
	}

	resolver := NewResolver([]byte(cfg.AuthSecret), cfg.CookieName)
	chatStore := chat.NewMemoryConversationStore() // CH-06 composition-root seam (CH-07 swaps this one line)
	factory := func(_ string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: provider, Store: chatStore})
	}

	e := echo.New()
	registry, err := chat.RegisterRoutes(e, resolver, factory)
	if err != nil {
		return fmt.Errorf("chat.RegisterRoutes: %w", err)
	}
	_ = registry // held for future graceful-shutdown hooks; CH-04 has no per-turn cleanup

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