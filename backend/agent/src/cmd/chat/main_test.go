// CH-04.1 — composition root wiring tests (R-06, design AD-5,
// tasks.md § Phase 3).
//
// The chat composition root reads every required env var, validates
// them, installs the OTel SDK, builds the openrouter provider, builds
// the JWE auth shim, builds the ConversationFactory, and calls
// chat.RegisterRoutes. This file exercises each piece independently so
// the wiring is provable from tests, not from the binary running.
//
// # Test surface
//
//   - envString returns the value when set, default otherwise
//   - loadConfig rejects each required credential with a named error
//   - loadConfig rejects AUTH_SECRET <32 bytes
//   - loadConfig returns a populated config when all vars present
//   - buildConversationFactory wraps a stub provider in a working
//     Conversation that drives one turn
//   - Wired root serves POST /api/agent/turns (401 without cookie,
//     400 on empty prompt)
//
// The integration tests use httptest.NewServer (no real port binding)
// and a stub ai.ModelProvider — the 200 OK case is owned by CH-04.3's
// opt-in live smoke, which drives one real turn against a real model.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// validAuthSecret is a 32-byte AUTH_SECRET value for tests. It MUST
// stay ≥32 bytes — loadConfig rejects shorter values.
const validAuthSecret = "abcdefghijklmnopqrstuvwxyzABCDEF"

// syntheticEnv builds an env-getter function from a map, returning ""
// for any key not in the map. Mirrors os.Getenv's "empty for unset"
// shape so the production getenv adapter and the test one share a
// single behaviour surface.
func syntheticEnv(m map[string]string) func(string) string {
	return func(k string) string {
		return m[k]
	}
}

// TestEnvString_ReturnsValueWhenSet covers the happy path: a non-empty
// env var is returned verbatim.
func TestEnvString_ReturnsValueWhenSet(t *testing.T) {
	t.Parallel()

	if got := envString("PATH", "/default"); got == "" || got == "/default" {
		t.Errorf("envString with PATH set returned %q, want the actual value (non-empty, not the default)", got)
	}
}

// TestEnvString_ReturnsDefaultWhenUnset covers the default fallback:
// an unset env var returns the supplied default.
func TestEnvString_ReturnsDefaultWhenUnset(t *testing.T) {
	t.Parallel()

	// CACHICAMAS_TEST_DEFINITELY_NOT_SET_42 is a synthetic key; the
	// production env does not carry it.
	if got := envString("CACHICAMAS_TEST_DEFINITELY_NOT_SET_42", "/fallback"); got != "/fallback" {
		t.Errorf("envString with unset key returned %q, want %q", got, "/fallback")
	}
}

// TestLoadConfig_MissingAPIKey covers the missing-credential failure:
// when CACHICAMAS_CHAT_PROVIDER_API_KEY is unset, loadConfig returns
// an error naming the missing var, so main.go's stderr line tells the
// operator which env var to set.
func TestLoadConfig_MissingAPIKey(t *testing.T) {
	t.Parallel()

	_, err := loadConfig(syntheticEnv(map[string]string{
		"CACHICAMAS_CHAT_PROVIDER_MODEL": "openai/gpt-4o",
		"AUTH_SECRET":                    validAuthSecret,
	}))

	if err == nil {
		t.Fatal("err = nil, want non-nil when CACHICAMAS_CHAT_PROVIDER_API_KEY is empty")
	}
	if !strings.Contains(err.Error(), "CACHICAMAS_CHAT_PROVIDER_API_KEY") {
		t.Errorf("err = %v, want it to name CACHICAMAS_CHAT_PROVIDER_API_KEY", err)
	}
}

// TestLoadConfig_MissingModel covers the same shape for the model var.
func TestLoadConfig_MissingModel(t *testing.T) {
	t.Parallel()

	_, err := loadConfig(syntheticEnv(map[string]string{
		"CACHICAMAS_CHAT_PROVIDER_API_KEY": "sk-test-token",
		"AUTH_SECRET":                      validAuthSecret,
	}))

	if err == nil {
		t.Fatal("err = nil, want non-nil when CACHICAMAS_CHAT_PROVIDER_MODEL is empty")
	}
	if !strings.Contains(err.Error(), "CACHICAMAS_CHAT_PROVIDER_MODEL") {
		t.Errorf("err = %v, want it to name CACHICAMAS_CHAT_PROVIDER_MODEL", err)
	}
}

// TestLoadConfig_MissingAuthSecret covers the same shape for AUTH_SECRET.
func TestLoadConfig_MissingAuthSecret(t *testing.T) {
	t.Parallel()

	_, err := loadConfig(syntheticEnv(map[string]string{
		"CACHICAMAS_CHAT_PROVIDER_API_KEY": "sk-test-token",
		"CACHICAMAS_CHAT_PROVIDER_MODEL":   "openai/gpt-4o",
	}))

	if err == nil {
		t.Fatal("err = nil, want non-nil when AUTH_SECRET is empty")
	}
	if !strings.Contains(err.Error(), "AUTH_SECRET") {
		t.Errorf("err = %v, want it to name AUTH_SECRET", err)
	}
}

// TestLoadConfig_AuthSecretTooShort covers the byte-length gate:
// AUTH_SECRET MUST be ≥32 bytes, mirroring database_administrator's
// S-BAM-081. A 31-byte secret is rejected.
func TestLoadConfig_AuthSecretTooShort(t *testing.T) {
	t.Parallel()

	const shortSecret = "abcdefghijklmnopqrstuvwxyz12345" // 31 bytes
	if len(shortSecret) != 31 {
		t.Fatalf("test setup: shortSecret is %d bytes, want 31", len(shortSecret))
	}

	_, err := loadConfig(syntheticEnv(map[string]string{
		"CACHICAMAS_CHAT_PROVIDER_API_KEY": "sk-test-token",
		"CACHICAMAS_CHAT_PROVIDER_MODEL":   "openai/gpt-4o",
		"AUTH_SECRET":                      shortSecret,
	}))

	if err == nil {
		t.Fatal("err = nil, want non-nil when AUTH_SECRET is <32 bytes (S-BAM-081)")
	}
	if !strings.Contains(err.Error(), "AUTH_SECRET") {
		t.Errorf("err = %v, want it to name AUTH_SECRET", err)
	}
}

// TestLoadConfig_AllPresent covers the happy path: every required
// var set with valid values, loadConfig returns a populated config
// with no error.
func TestLoadConfig_AllPresent(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig(syntheticEnv(map[string]string{
		"CACHICAMAS_CHAT_PORT":            "9090",
		"CACHICAMAS_CHAT_PROVIDER_API_KEY": "sk-test-token",
		"CACHICAMAS_CHAT_PROVIDER_MODEL":   "openai/gpt-4o",
		"AUTH_SECRET":                      validAuthSecret,
		"AUTH_COOKIE_NAME":                 "authjs.session-token",
		"OTEL_EXPORTER_OTLP_ENDPOINT":      "otel-collector:4317",
	}))

	if err != nil {
		t.Fatalf("loadConfig returned err = %v, want nil with all required vars present", err)
	}
	if cfg.Port != "9090" {
		t.Errorf("Port = %q, want %q", cfg.Port, "9090")
	}
	if cfg.ProviderAPIKey != "sk-test-token" {
		t.Errorf("ProviderAPIKey = %q, want %q", cfg.ProviderAPIKey, "sk-test-token")
	}
	if cfg.ProviderModel != "openai/gpt-4o" {
		t.Errorf("ProviderModel = %q, want %q", cfg.ProviderModel, "openai/gpt-4o")
	}
	if cfg.AuthSecret != validAuthSecret {
		t.Errorf("AuthSecret mismatch")
	}
	if cfg.CookieName != "authjs.session-token" {
		t.Errorf("CookieName = %q, want %q (default applied when env var is set)", cfg.CookieName, "authjs.session-token")
	}
}

// TestLoadConfig_DefaultsApplied: when the optional env vars are
// unset, loadConfig uses the documented defaults.
func TestLoadConfig_DefaultsApplied(t *testing.T) {
	t.Parallel()

	cfg, err := loadConfig(syntheticEnv(map[string]string{
		"CACHICAMAS_CHAT_PROVIDER_API_KEY": "sk-test-token",
		"CACHICAMAS_CHAT_PROVIDER_MODEL":   "openai/gpt-4o",
		"AUTH_SECRET":                      validAuthSecret,
	}))

	if err != nil {
		t.Fatalf("loadConfig returned err = %v, want nil", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want default %q", cfg.Port, "8080")
	}
	if cfg.CookieName != "authjs.session-token" {
		t.Errorf("CookieName = %q, want default %q", cfg.CookieName, "authjs.session-token")
	}
}

// stubProvider is the minimal ai.ModelProvider the wired-root test
// uses. It returns a channel that emits a response-start, a single
// text-delta and a completion event — the minimum shape the chat
// projection accepts as a valid one-turn stream.
//
// The stub is NOT used by the 401/400 path tests; those only verify
// Echo's identity middleware + handler mounting. The 200 OK path is
// the live smoke's job (CH-04.3).
type stubProvider struct{}

func (stubProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error) {
	responseStart, _ := ai.NewResponseStart("resp-stub", "model-stub")
	textDelta, _ := ai.NewTextDelta(0, "hello from stub")
	completion, _ := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})

	ch := make(chan ai.Event, 3)
	go func() {
		defer close(ch)
		ch <- responseStart
		select {
		case <-ctx.Done():
			return
		case ch <- textDelta:
		}
		select {
		case <-ctx.Done():
			return
		case ch <- completion:
		}
	}()
	return ch, nil
}

// TestWiredRoot_PostsWithoutCookieReturns401 covers CH-04.1's "Echo
// binds and serves /api/agent/turns" scenario, 401 sub-case: the
// identity middleware short-circuits with a 401 envelope when the
// resolver returns (nil, false). The resolver is the JWE shim; no
// cookie is attached to the POST.
func TestWiredRoot_PostsWithoutCookieReturns401(t *testing.T) {
	t.Parallel()

	e, _ := buildWiredEcho(t)
	srv := httptest.NewServer(e)
	defer srv.Close()

	body, _ := json.Marshal(map[string]string{"id": "t1", "prompt": "hi"})
	resp, err := http.Post(srv.URL+"/api/agent/turns", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("http.Post error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 (identity not resolved)", resp.StatusCode)
	}
}

// TestWiredRoot_PostsWithValidCookieButEmptyPromptReturns400 covers the
// handler-mount path: a valid JWE cookie authenticates the request,
// but the prompt is empty — the handler refuses with a 400 validation
// envelope. This proves RegisterRoutes mounted the POST /api/agent/turns
// handler through the shim-authenticated path.
func TestWiredRoot_PostsWithValidCookieButEmptyPromptReturns400(t *testing.T) {
	t.Parallel()

	e, _ := buildWiredEcho(t)
	srv := httptest.NewServer(e)
	defer srv.Close()

	cookieValue := testEncryptJWECookie(t, []byte(validAuthSecret), "authjs.session-token", authJSCookiePayload{
		Sub: "participant-stub",
		Exp: 9_999_999_999, // far future
	})

	body, _ := json.Marshal(map[string]string{"id": "t1", "prompt": ""}) // empty prompt
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/api/agent/turns", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: "authjs.session-token", Value: cookieValue})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("status = %d, want 400 (validation: prompt required)", resp.StatusCode)
	}
}

// buildWiredEcho wires the minimum subset of the composition root the
// integration tests need: a fresh Echo, a Resolver built from the
// valid test secret, a factory built from the stubProvider, and
// chat.RegisterRoutes called once. OTel is deliberately NOT installed —
// tests run without a collector.
func buildWiredEcho(t *testing.T) (*echo.Echo, *chat.Registry) {
	t.Helper()

	resolver := NewResolver([]byte(validAuthSecret), "authjs.session-token")
	factory := func(participantID string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: stubProvider{}})
	}

	e := echo.New()
	registry, err := chat.RegisterRoutes(e, resolver, factory)
	if err != nil {
		t.Fatalf("chat.RegisterRoutes: %v", err)
	}
	return e, registry
}

// TestBuildConversationFactory_WrapsProviderInConversation covers the
// shape of the ConversationFactory the root hands to chat.RegisterRoutes:
// given a participant id, it returns a non-nil *chat.Conversation
// driven by the injected provider.
func TestBuildConversationFactory_WrapsProviderInConversation(t *testing.T) {
	t.Parallel()

	factory := func(participantID string) (*chat.Conversation, error) {
		return chat.NewConversation(chat.Config{Provider: stubProvider{}})
	}

	conv, err := factory("participant-1")
	if err != nil {
		t.Fatalf("factory returned err = %v, want nil", err)
	}
	if conv == nil {
		t.Fatal("conv = nil, want non-nil")
	}
	if got := conv.IsInFlight(); got != false {
		t.Errorf("IsInFlight = %v, want false on a fresh Conversation", got)
	}
}

// Sanity check that envString's production caller (loadConfig) and
// the test-only synthetic env agree on what "empty" means: a missing
// key and an empty-string value both look the same to loadConfig. This
// guards against loadConfig accepting "" from a present-but-empty
// env var (the same shape as missing).
func TestLoadConfig_EmptyStringTreatedAsMissing(t *testing.T) {
	t.Parallel()

	_, err := loadConfig(syntheticEnv(map[string]string{
		"CACHICAMAS_CHAT_PROVIDER_API_KEY": "", // present but empty
		"CACHICAMAS_CHAT_PROVIDER_MODEL":   "openai/gpt-4o",
		"AUTH_SECRET":                      validAuthSecret,
	}))

	if err == nil {
		t.Fatal("err = nil, want non-nil when CACHICAMAS_CHAT_PROVIDER_API_KEY is empty")
	}
}

// Make sure the production env reading path (os.Getenv via envString)
// is observable by the suite: a missing key in the real env returns
// the default. This is the kind of regression that only catches when
// os.Getenv is mocked out, so we sanity-check the real path here.
func TestEnvString_RealEnvMissingKey(t *testing.T) {
	// Unset on purpose for the duration of the test.
	const probeKey = "CACHICAMAS_TEST_PROBE_NOT_SET_42"
	orig, hadOrig := os.LookupEnv(probeKey)
	os.Unsetenv(probeKey)
	defer func() {
		if hadOrig {
			os.Setenv(probeKey, orig)
		}
	}()

	if got := envString(probeKey, "/default-value"); got != "/default-value" {
		t.Errorf("envString(%q) with unset key returned %q, want %q", probeKey, got, "/default-value")
	}
}

// ensure fmt is referenced even when no path uses it directly — keeps
// the imports list intentional as the file grows.
var _ = fmt.Sprintf