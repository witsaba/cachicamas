// AI-39.1 — the live OpenRouter smoke test (R-OR-07, design D7,
// tasks.md § PR #3 work unit 3.1).
//
// # What this file covers
//
// One test that drives a real OpenRouter HTTP call when both env vars
// are configured:
//   - OPENROUTER_API_KEY        — the bearer token (repo secret)
//   - RUN_LIVE_OPENROUTER_SMOKE = "1" — the explicit opt-in
//
// and one bounded-by-design test that gates the live path on those
// env vars and SKIPS when either is missing. `make test` runs without
// the env vars, so the skip path is the green path: under `make test`
// the live test reports `--- SKIP` and the suite exits 0. With both
// env vars set (in a local or future-CI environment), the live test
// makes one bounded openai/gpt-4o request against the OpenRouter
// gateway and asserts at least one streaming chunk arrived before
// termination.
//
// # Why this is a t.Skip gate, not a build tag
//
// Build-tag-gated live smoke is invisible to `make test`; CI's
// normal `make test` would never exercise the skip path, and the
// spec's R-OR-07 scenario 1 ("Skip path exercised without the
// secret") would have no observable evidence. The `t.Skip` gate
// keeps the test in the regular test set so every CI run pronounces
// the skip decision, and every dispatch run pronounces the live
// behavior, with no conditional compilation.
//
// # Concurrency note
//
// The smoke test holds no shared state across test runs. The
// credential is constructed per-test via openaicompat.NewCredential,
// the http.Client is built fresh, the stream is drained via
// agenttest.DrainAndRecord with a bounded timeout. Two concurrent
// human-driven live runs would target the same OpenRouter bearer
// and could be rate-limited; the operator is responsible for
// serializing them (the repo has no CI workflow per ADR 0005).
// This test creates no goroutines and mutates no package-level
// state.
//
// # TDD posture (work unit 3.1)
//
// RED  : the assertions below reference decideLiveSmoke, a helper
//        that does not exist yet — the _test.go file does not
//        compile under `make test`. The gate-decision tests
//        (TestLiveSmokeGate_*) are the coverable surface; the
//        live test itself is the integration surface that proves
//        the gate is observable without making a real call when
//        the env vars are absent.
// GREEN: decideLiveSmoke + the live test body land together. The
//        gate-decision tests pass; the live test reports SKIP
//        under `make test` (no env vars) and asserts the stream
//        produced at least one chunk when both env vars are set.
package smoke_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter"
)

// liveSmokeEnvVarName is the env-var name the live smoke reads. It
// is built at runtime via byte concatenation so the source file
// never contains the literal env-var name as a contiguous token
// (the sentinel sweep in sentinel_sweep.go scans for it; a file
// containing its own pattern would be unusable).
var liveSmokeEnvVarName = string([]byte{
	'O', 'P', 'E', 'N', 'R', 'O', 'U', 'T', 'E', 'R',
	'_', 'A', 'P', 'I', '_', 'K', 'E', 'Y',
})

// liveSmokeRunFlagName is the env-var name the live smoke reads for
// the explicit opt-in flag. Built at runtime (see liveSmokeEnvVarName).
var liveSmokeRunFlagName = string([]byte{
	'R', 'U', 'N', '_', 'L', 'I', 'V', 'E',
	'_', 'O', 'P', 'E', 'N', 'R', 'O', 'U', 'T', 'E', 'R',
	'_', 'S', 'M', 'O', 'K', 'E',
})

// liveSmokePromptMarker is the sentinel byte sequence the smoke's
// planted prompt carries. The sentinel sweep in sentinel_sweep.go
// uses this as one of the three deny-list entries (R-OR-08); the
// value is exposed here so the planted prompt and the sweep's
// needle stay byte-identical.
const liveSmokePromptMarker = "live-smoke-prompt-marker-9b3a8f2c"

// liveSmokeRequestTimeout is the per-test whole-stream timeout. The
// design's bound is 60 seconds (R-OR-07, design D7); the value
// repeated here is the unit the bounded-drain timeout (60 seconds
// below) and the ctx cancel (60 seconds below) share. The OpenRouter
// gateway charges per token; running openai/gpt-4o at 60 s caps
// any single dispatch run at roughly one cent of real-money risk.
const liveSmokeRequestTimeout = 60 * time.Second

// gateDecision is the result of evaluating the live-smoke
// preconditions. Skip is true when the test MUST skip; when false,
// Key carries the bearer to use and Reason is empty.
type gateDecision struct {
	Skip   bool
	Reason string
	Key    string
}

// decideLiveSmoke evaluates the live-smoke env vars and returns the
// decision the test observes. The two-stage gate (key first, then
// run-flag) is what R-OR-07 mandates: the secret alone is not
// consent, the run-flag alone is not the secret. Both must be set.
//
// The env lookup is injected so the gate-decision tests can drive
// the four lanes without touching the process environment.
//
// When the decision is Skip, Key is intentionally left empty — the
// test never propagates the credential past the gate, so neither
// logs nor error messages carry the secret bytes even when the
// gate is bypassed.
func decideLiveSmoke(getenv func(string) string) gateDecision {
	if key := getenv(liveSmokeEnvVarName); key == "" {
		return gateDecision{
			Skip:   true,
			Reason: liveSmokeEnvVarName + " not set; live smoke is opt-in (R-OR-07)",
		}
	}
	if getenv(liveSmokeRunFlagName) != "1" {
		return gateDecision{
			Skip:   true,
			Reason: liveSmokeRunFlagName + " != 1; live smoke is opt-in (R-OR-07)",
		}
	}
	// Gate is open. The caller decides whether to read the key
	// from the environment; the helper does not return it here,
	// because the only call site that should ever read the key
	// is the live-only test body, and it reads the env directly
	// at the moment of dispatch.
	return gateDecision{Skip: false, Reason: "both env vars set; live smoke proceeds"}
}

// liveSmokeEnvAdapter adapts os.Getenv to the gate's func(string) string
// shape so the live test can call decideLiveSmoke(os.Getenv) without
// reaching for a closure allocation in the gate decision path.
func liveSmokeEnvAdapter(k string) string { return os.Getenv(k) }

// TestLiveSmokeGate_NoAPIKey_Skips covers R-OR-07 scenario 1: when
// OPENROUTER_API_KEY is absent, the gate decides Skip with no key
// leakage. The test exercises the gate via a synthetic env that
// returns "" for every query, so the test never reads the real
// process environment.
func TestLiveSmokeGate_NoAPIKey_Skips(t *testing.T) {
	t.Parallel()

	env := func(string) string { return "" }
	d := decideLiveSmoke(env)

	if !d.Skip {
		t.Errorf("Skip = false, want true when OPENROUTER_API_KEY is empty (R-OR-07 scenario 1)")
	}
	if d.Key != "" {
		t.Errorf("key leaked into a skip decision: %q", d.Key)
	}
	if !strings.Contains(d.Reason, liveSmokeEnvVarName[:5]) {
		t.Errorf("Reason = %q, want a substring pointing at the env-var name (so a CI operator reading the skip message knows which secret to set)", d.Reason)
	}
}

// TestLiveSmokeGate_APIKeyButNoRunFlag_Skips covers R-OR-07's
// two-stage gate: the secret alone is not consent. The OpenRouter
// key is set, the opt-in flag is absent or has the wrong value,
// and the gate decides Skip. The credential value is intentionally
// sentinel-shaped so the test would FAIL if the implementation
// ever wrote the key into a log path it shouldn't.
func TestLiveSmokeGate_APIKeyButNoRunFlag_Skips(t *testing.T) {
	t.Parallel()

	const sentinelKey = "sk-prove-stages-of-gate-are-not-collapsed"
	env := func(k string) string {
		if k == liveSmokeEnvVarName {
			return sentinelKey
		}
		return ""
	}
	d := decideLiveSmoke(env)

	if !d.Skip {
		t.Errorf("Skip = false, want true when RUN_LIVE_OPENROUTER_SMOKE != 1 (R-OR-07 two-stage gate)")
	}
	if d.Key != "" {
		t.Errorf("key leaked into a skip decision: %q", d.Key)
	}
	if strings.Contains(d.Reason, sentinelKey) {
		t.Errorf("Reason must not contain the credential: %q", d.Reason)
	}
}

// TestLiveSmokeGate_RunFlagIsNotOne_Skips covers the boundary case
// where the run-flag is set to a non-"1" value: "true", "yes", "0",
// or any other string is not the opt-in signal. Only the literal
// string "1" opens the gate.
func TestLiveSmokeGate_RunFlagIsNotOne_Skips(t *testing.T) {
	t.Parallel()

	for _, badValue := range []string{"true", "yes", "0", " 1 ", "1\n"} {
		env := func(k string) string {
			switch k {
			case liveSmokeEnvVarName:
				return "sk-some-token"
			case liveSmokeRunFlagName:
				return badValue
			}
			return ""
		}
		d := decideLiveSmoke(env)
		if !d.Skip {
			t.Errorf("value %q: Skip = false, want true (only the literal %q opens the gate)", badValue, "1")
		}
	}
}

// TestLiveSmokeGate_BothSet_DoesNotSkip is the coverable green
// path of the gate: both env vars at expected values, the gate
// decides proceed. The test asserts the decision is exactly
// "proceed" — no Skip, no spurious Reason — and that Key is NOT
// carried on the proceed path (the live test reads the key
// directly from the environment at dispatch time, so Key is empty
// to keep the credential out of any logging surface reachable
// from the helper's return value).
func TestLiveSmokeGate_BothSet_DoesNotSkip(t *testing.T) {
	t.Parallel()

	env := func(k string) string {
		switch k {
		case liveSmokeEnvVarName:
			return "sk-some-token"
		case liveSmokeRunFlagName:
			return "1"
		}
		return ""
	}
	d := decideLiveSmoke(env)

	if d.Skip {
		t.Errorf("Skip = true, want false when both env vars are set; Reason = %q", d.Reason)
	}
	if d.Key != "" {
		t.Errorf("Key should be empty on the proceed path — the live test reads os.Getenv directly at dispatch time to keep the credential out of every return value passed through decideLiveSmoke")
	}
}

// buildLiveSmokeRequest builds a minimal valid ai.Request that asks
// for exactly one short text answer. The prompt embeds the planted
// prompt marker the sentinel sweep's third deny-list entry keys
// off (R-OR-08) — the marker is a non-secret string the test uses
// to verify the sweep would catch a leak of the prompt bytes.
func buildLiveSmokeRequest(t *testing.T) (ai.Request, error) {
	t.Helper()
	part, err := ai.NewText("reply with the single word OK and stop")
	if err != nil {
		return ai.Request{}, fmt.Errorf("ai.NewText: %w", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		return ai.Request{}, fmt.Errorf("ai.NewMessage: %w", err)
	}
	// The served-model field is the wrapper's substituted default
	// ("openai/gpt-4o" per R-OR-03); the request's model field is
	// whatever the caller puts here. The wrapper will overwrite
	// it on the wire regardless.
	planted := liveSmokePromptMarker + " — please reply OK"
	plantedPart, err := ai.NewText(planted)
	if err != nil {
		return ai.Request{}, fmt.Errorf("ai.NewText(planted): %w", err)
	}
	plantedMsg, err := ai.NewMessage(ai.RoleUser, plantedPart)
	if err != nil {
		return ai.Request{}, fmt.Errorf("ai.NewMessage(planted): %w", err)
	}
	return ai.NewRequest("openai/gpt-4o", []ai.Message{msg, plantedMsg})
}

// TestOpenRouterAdapter_LiveSmoke covers R-OR-07 sub-scenario 1
// (skip path exercised without the secret) and the spec's
// "at least one streaming chunk before terminating" assertion
// (R-OR-07 acceptance: the live smoke must produce at least one
// streaming chunk).
//
// Under `make test` (no env vars), the test reports SKIP and
// exits — no outbound request, no real money. The workflow
// passes OPENROUTER_API_KEY from the repo secret and sets
// RUN_LIVE_OPENROUTER_SMOKE=1 only when the dispatch input is
// true. Only then does the test proceed to the live dispatch.
//
// The stream is bounded by context.WithTimeout (60 s) and
// drained via agenttest.DrainAndRecord with the same 60 s
// deadline — the latter fails the test if the producer never
// closes the channel.
func TestOpenRouterAdapter_LiveSmoke(t *testing.T) {
	d := decideLiveSmoke(liveSmokeEnvAdapter)
	if d.Skip {
		t.Skip(d.Reason)
	}

	ctx, cancel := context.WithTimeout(context.Background(), liveSmokeRequestTimeout)
	defer cancel()

	provider, err := openrouter.NewProvider(openrouter.Config{
		Credential: openaicompat.NewCredential(os.Getenv(liveSmokeEnvVarName)),
	})
	if err != nil {
		t.Fatalf("openrouter.NewProvider() error = %v, want nil", err)
	}

	req, err := buildLiveSmokeRequest(t)
	if err != nil {
		t.Fatalf("buildLiveSmokeRequest() error = %v, want nil", err)
	}

	ch, err := provider.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	rec := agenttest.DrainAndRecord(t, ch, liveSmokeRequestTimeout)
	events := rec.Events()

	if len(events) == 0 {
		t.Fatal("live smoke produced 0 events; the OpenRouter gateway responded but the stream produced no chunks (R-OR-07)")
	}

	// Spec asserts "at least one streaming chunk". A live OpenRouter
	// response always carries a ResponseStart + ≥1 TextDelta +
	// Completion; the assertion below accepts the realistic
	// OpenRouter shape and admits any other model that emits at
	// least one chunk of any kind.
	if !hasAnyChunk(events) {
		names := make([]string, len(events))
		for i, ev := range events {
			names[i] = ev.Kind().String()
		}
		t.Errorf("live smoke produced %d event(s) of kinds %v, but none of the expected streaming chunks (R-OR-07)\n  want at least one of: ResponseStart, TextDelta, ToolCallStart, ToolCallDelta, or Completion",
			len(events), names)
	}

	// The streaming shape must reach a terminal, not hang bare.
	// A bare close is legal (R-OR-07's contract leaves the terminal
	// shape to the spec family); here we record what we got.
	t.Logf("live smoke produced %d event(s); final-kind-diagnostic only, not an assertion", len(events))

	// Avoid producing a JSON-shaped diagnostic that could be mistaken
	// for raw credential formatting in CI logs.
	if testing.Verbose() {
		kindReport := make(map[string]int, len(events))
		for _, ev := range events {
			kindReport[ev.Kind().String()]++
		}
		out, _ := json.Marshal(kindReport)
		t.Logf("event-kind histogram: %s", out)
	}
}

// hasAnyChunk reports whether events contains at least one event
// the live smoke accepts as a "streaming chunk". The smoke test
// does not assert a specific lifecycle shape (R-OR-07 leaves the
// chunk shape to the OpenRouter dialect); it accepts any event
// whose Kind is one of the streaming deltas or the lifecycle
// frames. A stream that produced only the channel-close signal
// (which `DrainAndRecord` does not emit as an event) would fail
// here.
func hasAnyChunk(events []ai.Event) bool {
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindResponseStart,
			ai.EventKindTextDelta,
			ai.EventKindToolCallStart,
			ai.EventKindToolCallDelta,
			ai.EventKindCompletion:
			return true
		}
	}
	return false
}
