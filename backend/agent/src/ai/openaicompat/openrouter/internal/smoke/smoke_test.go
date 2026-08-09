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
//
// credential-scan:deliberate-plant — plants "sk-prove-stages-of-gate-are-not-collapsed"
// to prove the live smoke's two-stage gate (key first, then run-flag) does
// not leak the credential into a skip decision (R-OR-07). Declared here per
// AI-36's recursive credential scan (D-3, R-ART-022); enumerated in
// credential_scan_test.go's deliberatePlantAllowlist.
package smoke_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/sweep"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke"
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
func buildLiveSmokeRequest(tb testing.TB) (ai.Request, error) {
	tb.Helper()
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

// TestOpenRouterAdapter_LiveSmoke covers the full ai-live-smoke contract
// (R-LSM-001…R-LSM-008) and its vendor-side anchor R-OR-07/R-OR-08.
//
// Under `make test` (no env vars), the test reports SKIP and exits — no
// outbound request, no real money. A human operator sets both
// OPENROUTER_API_KEY and RUN_LIVE_OPENROUTER_SMOKE=1 locally (see
// README.md in this directory) to exercise the live path; no CI workflow
// sets them (ADR 0005 § Enforcement — the repository has no
// .github/workflows/ directory).
//
// The whole live path — dispatch, drain, and the R-LSM-003 shape
// invariants — runs behind evaluateSweepGate's capture funnel
// (design.md D3): the positive control passes before any request is
// sent, every diagnostic runLiveSmoke produces is captured rather than
// logged directly, the sweep gates the capture before release on every
// path (success or failure), and only a clean sweep's output ever
// reaches t.Log (R-LSM-004).
func TestOpenRouterAdapter_LiveSmoke(t *testing.T) {
	d := decideLiveSmoke(liveSmokeEnvAdapter)
	if d.Skip {
		t.Skip(d.Reason)
	}

	ctx, cancel := context.WithTimeout(context.Background(), liveSmokeRequestTimeout)
	defer cancel()

	key := os.Getenv(liveSmokeEnvVarName)
	deny := smoke.BuildDenyList(liveSmokeEnvVarName, key[:min(4, len(key))], liveSmokePromptMarker)

	release, err := evaluateSweepGate(ctx, deny, runLiveSmoke)
	if release != "" {
		t.Logf("%s", release)
	}
	if err != nil {
		t.Fatal(err)
	}
}

// streamShapeReport is the result of the three independent stream-shape
// invariants R-LSM-003 mandates. Each is its own field, checked
// separately, so that suppressing any one of the three leaves the other
// two still able to fail — never one disjunction standing in for the
// three (S-LSM-007).
type streamShapeReport struct {
	responseStartErr error
	contentErr       error
	terminalErr      error
}

// checkStreamShape evaluates the three R-LSM-003 invariants over events
// and returns them independently:
//
//  1. responseStartErr is non-nil unless at least one
//     [ai.EventKindResponseStart] event is present.
//  2. contentErr is non-nil unless at least one content event
//     ([ai.EventKindTextDelta], [ai.EventKindToolCallStart] or
//     [ai.EventKindToolCallDelta]) is present.
//  3. terminalErr is non-nil unless [ai.CheckStream] reports no violation
//     AND its report is Terminated() — exactly one terminal event, in
//     terminal position. An empty stream fails this check rather than
//     passing vacuously (S-LSM-009).
//
// No check reads, compares, or matches the model's generated text, tool
// arguments, token counts, or any other provider-chosen content — only
// event kinds, counts, order and presence are consulted (S-LSM-010).
func checkStreamShape(events []ai.Event) streamShapeReport {
	return streamShapeReport{
		responseStartErr: checkResponseStartPresent(events),
		contentErr:       checkContentPresent(events),
		terminalErr:      checkExactlyOneTerminal(events),
	}
}

// checkResponseStartPresent is R-LSM-003 invariant 1: at least one
// response-start event is present.
func checkResponseStartPresent(events []ai.Event) error {
	for _, ev := range events {
		if ev.Kind() == ai.EventKindResponseStart {
			return nil
		}
	}
	return fmt.Errorf("stream shape: response-start: no response-start event observed (R-LSM-003)")
}

// checkContentPresent is R-LSM-003 invariant 2: at least one content
// event follows. Only the event's Kind is consulted — never its payload
// (S-LSM-010).
func checkContentPresent(events []ai.Event) error {
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindTextDelta, ai.EventKindToolCallStart, ai.EventKindToolCallDelta:
			return nil
		}
	}
	return fmt.Errorf("stream shape: content-presence: no content event (text delta or tool call) observed (R-LSM-003, S-LSM-008)")
}

// checkExactlyOneTerminal is R-LSM-003 invariant 3: exactly one terminal
// event exists, in terminal position. It delegates ordering and
// at-most-one enforcement to [ai.CheckStream] (never reimplemented) and
// additionally requires Terminated() == true, so an empty or
// never-terminated stream fails here rather than passing vacuously
// (S-LSM-009).
func checkExactlyOneTerminal(events []ai.Event) error {
	report := ai.CheckStream(events)
	if violation := report.Violation(); violation != nil {
		return fmt.Errorf("stream shape: terminal-exclusivity: %w (R-LSM-003, S-LSM-009)", violation)
	}
	if !report.Terminated() {
		return fmt.Errorf("stream shape: terminal-exclusivity: no terminal event observed (R-LSM-003, S-LSM-009)")
	}
	return nil
}

// captureTB is a local testing.TB double (design.md D3) that routes every
// message a TB-taking helper (agenttest.DrainAndRecord) would otherwise
// send straight to the real *testing.T into sink instead, so nothing
// reaches the test log before the sentinel sweep has run over it
// (R-LSM-004).
//
// The embedded testing.TB is deliberately never assigned a real value:
// every method a call site actually reaches (Helper, Logf, Errorf,
// Fatalf) is overridden below, following this module's existing fakeTB
// idiom (src/agenttest/stream_kit_record_test.go). A call that fell
// through to the nil embedded interface would panic — the intended,
// loud failure mode for a method this double was supposed to override
// but did not.
//
// Fatalf deliberately does NOT call the real FailNow/runtime.Goexit path:
// it only records and marks failed, then returns normally. This is safe
// because every TB-taking helper this double is passed to (verified:
// agenttest.DrainAndRecord) places an explicit return statement
// immediately after each Fatalf call, so control flow after a captured
// Fatalf is exactly what the caller already wrote for the "test failed,
// stop now" case — DrainAndRecord's timeout branch already returns
// Recording{events: got} right after tb.Fatalf(...).
type captureTB struct {
	testing.TB
	sink   *bytes.Buffer
	failed bool
}

// Helper is a no-op: captureTB carries no real *testing.T call stack to
// mark, so there is nothing for it to record.
func (c *captureTB) Helper() {}

// Logf records an informational message into the sink without marking
// captureTB failed.
func (c *captureTB) Logf(format string, args ...any) {
	fmt.Fprintf(c.sink, format+"\n", args...)
}

// Errorf records a failure message into the sink and marks captureTB
// failed. Unlike the real testing.T.Errorf, it does not abort the
// calling goroutine.
func (c *captureTB) Errorf(format string, args ...any) {
	fmt.Fprintf(c.sink, format+"\n", args...)
	c.failed = true
}

// Fatalf records a failure message into the sink and marks captureTB
// failed. It deliberately does NOT call runtime.Goexit() (see the type's
// doc comment) — every call site this double is passed to already
// returns immediately after calling Fatalf.
func (c *captureTB) Fatalf(format string, args ...any) {
	fmt.Fprintf(c.sink, format+"\n", args...)
	c.failed = true
}

// The blank assignment below pins runLiveSmoke's exact call shape against
// what evaluateSweepGate's run parameter expects — (ctx, &sink) error —
// as a standalone, dedicated compile-time assertion, never invoked (a
// live dispatch has no place in a unit test). The real call site
// (TestOpenRouterAdapter_LiveSmoke, below) enforces the identical
// constraint by passing runLiveSmoke as an argument; this pin catches a
// signature drift even if that call site is ever refactored away from a
// direct reference.
var _ func(context.Context, *bytes.Buffer) error = runLiveSmoke

// evaluateSweepGate is the D3 capture-funnel sequence (R-LSM-004): the
// positive control MUST pass before any dispatch is trusted — no spend
// on a broken sweep (S-LSM-014) — then run is invoked with a fresh sink,
// then the sweep gates every byte run wrote before any of it is
// released, on every path run may take, success or failure alike
// (S-LSM-011, S-LSM-012).
//
// A sweep match returns ("", err): err names only the offending vector
// and states the output was withheld; the matched bytes are never
// reproduced (S-LSM-013). A clean sweep returns the full captured output
// for release regardless of whether run itself failed — the caller
// learns run's own failure only through the returned error, after the
// diagnostics are already proven safe to log (S-LSM-012, S-LSM-015).
func evaluateSweepGate(ctx context.Context, deny []smoke.DenyEntry, run func(context.Context, *bytes.Buffer) error) (release string, err error) {
	if selfTestErr := sweep.SelfTest(deny); selfTestErr != nil {
		return "", fmt.Errorf("sweep positive control failed, refusing to trust a clean result: %w", selfTestErr)
	}

	var sink bytes.Buffer
	runErr := run(ctx, &sink)
	if runErr != nil {
		fmt.Fprintf(&sink, "live smoke failed: %v\n", runErr)
	}

	if scanErr := smoke.Scan(sink.Bytes(), deny); scanErr != nil {
		return "", fmt.Errorf("%v — captured output withheld", scanErr)
	}

	if runErr != nil {
		return sink.String(), fmt.Errorf("live smoke failed; swept diagnostics released above: %w", runErr)
	}
	return sink.String(), nil
}

// runLiveSmoke performs the one bounded live dispatch (R-LSM-002):
// provider construction, request build, Stream, drain (via a local
// captureTB so nothing reaches a real *testing.T before the sweep,
// R-LSM-004), sequence contiguity, and the three R-LSM-003 stream-shape
// invariants. Every failure path returns an error into sink rather than
// calling a real t.Fatal, so evaluateSweepGate always gets the last word
// on what is safe to release.
func runLiveSmoke(ctx context.Context, sink *bytes.Buffer) error {
	provider, err := openrouter.NewProvider(openrouter.Config{
		Credential: openaicompat.NewCredential(os.Getenv(liveSmokeEnvVarName)),
	})
	if err != nil {
		return fmt.Errorf("openrouter.NewProvider: %w", err)
	}

	tb := &captureTB{sink: sink}
	req, err := buildLiveSmokeRequest(tb)
	if err != nil {
		return fmt.Errorf("buildLiveSmokeRequest: %w", err)
	}

	ch, err := provider.Stream(ctx, req)
	if err != nil {
		return fmt.Errorf("Stream: %w", err)
	}

	rec := agenttest.DrainAndRecord(tb, ch, liveSmokeRequestTimeout)
	if tb.failed {
		return fmt.Errorf("drain failed to close within the bound")
	}
	events := rec.Events()

	if contiguityErr := agenttest.CheckContiguity(events); contiguityErr != nil {
		return fmt.Errorf("stream shape: %w", contiguityErr)
	}

	report := checkStreamShape(events)
	if report.responseStartErr != nil {
		return report.responseStartErr
	}
	if report.contentErr != nil {
		return report.contentErr
	}
	if report.terminalErr != nil {
		return report.terminalErr
	}
	return nil
}

// TestEvaluateSweepGate covers R-LSM-004's capture-sink funnel (design.md
// D3) in isolation from any real dispatch: run is injected so each case
// controls exactly what lands in the sink and whether run itself fails,
// without ever opening a network connection.
func TestEvaluateSweepGate(t *testing.T) {
	t.Parallel()

	t.Run("a planted leak fails naming only the vector, output withheld", func(t *testing.T) {
		t.Parallel()

		const secret = "sk-planted-gate-leak-do-not-reprint"
		deny := []smoke.DenyEntry{{Vector: "secret prefix", Needle: []byte(secret[:4])}}
		run := func(_ context.Context, sink *bytes.Buffer) error {
			fmt.Fprintf(sink, "diagnostic: leaked %s by mistake\n", secret)
			return nil
		}

		release, err := evaluateSweepGate(t.Context(), deny, run)

		if err == nil {
			t.Fatal("err = nil, want non-nil: the planted leak must fail the gate (S-LSM-013)")
		}
		if release != "" {
			t.Errorf("release = %q, want empty — a swept match must never be released (S-LSM-013)", release)
		}
		if strings.Contains(err.Error(), secret) {
			t.Errorf("error reprints the credential: %q (S-LSM-013)", err.Error())
		}
		if !strings.Contains(err.Error(), "secret prefix") {
			t.Errorf("error = %q, want it to name the vector 'secret prefix'", err.Error())
		}
		if !strings.Contains(strings.ToLower(err.Error()), "withheld") {
			t.Errorf("error = %q, want it to state output was withheld (R-LSM-004)", err.Error())
		}
	})

	t.Run("a clean sweep releases the full captured diagnostics", func(t *testing.T) {
		t.Parallel()

		deny := smoke.BuildDenyList("UNRELATED_ENV_VAR", "zzzz", "unrelated-planted-prompt")
		run := func(_ context.Context, sink *bytes.Buffer) error {
			fmt.Fprintf(sink, "ordinary diagnostic, nothing sensitive\n")
			return nil
		}

		release, err := evaluateSweepGate(t.Context(), deny, run)

		if err != nil {
			t.Fatalf("err = %v, want nil on a clean sweep (S-LSM-015)", err)
		}
		if !strings.Contains(release, "ordinary diagnostic, nothing sensitive") {
			t.Errorf("release = %q, want the full captured buffer released (S-LSM-015)", release)
		}
	})

	t.Run("the positive control forced to fail fails the run before any dispatch", func(t *testing.T) {
		t.Parallel()

		// An entry with an empty Needle can never bite its own self-test
		// corpus (agenttest/sweep.SelfTest) — the deliberate way to force
		// the positive control to fail (S-LSM-014).
		deny := []smoke.DenyEntry{{Vector: "broken-vector", Needle: nil}}
		dispatched := false
		run := func(_ context.Context, sink *bytes.Buffer) error {
			dispatched = true
			fmt.Fprintf(sink, "should never run")
			return nil
		}

		release, err := evaluateSweepGate(t.Context(), deny, run)

		if err == nil {
			t.Fatal("err = nil, want non-nil: a broken positive control must fail the run rather than report clean (S-LSM-014)")
		}
		if release != "" {
			t.Errorf("release = %q, want empty — nothing is trustworthy before the positive control passes", release)
		}
		if dispatched {
			t.Error("run was dispatched despite the positive control failing, want no spend on a broken sweep (design.md D3)")
		}
	})

	t.Run("a run failure is still swept and released before the run is reported failed", func(t *testing.T) {
		t.Parallel()

		deny := smoke.BuildDenyList("UNRELATED_ENV_VAR", "zzzz", "unrelated-planted-prompt")
		run := func(_ context.Context, sink *bytes.Buffer) error {
			fmt.Fprintf(sink, "request attempted, no leak here\n")
			return fmt.Errorf("simulated dispatch failure")
		}

		release, err := evaluateSweepGate(t.Context(), deny, run)

		if err == nil {
			t.Fatal("err = nil, want non-nil: a run failure must still fail the gate (S-LSM-012)")
		}
		if !strings.Contains(release, "request attempted, no leak here") {
			t.Errorf("release = %q, want the swept diagnostics released even on a run failure (S-LSM-012, S-LSM-015)", release)
		}
		if !strings.Contains(err.Error(), "simulated dispatch failure") {
			t.Errorf("error = %v, want it to surface the run failure", err)
		}
	})
}

// mustEvent fails t immediately if constructing a synthetic test event
// returned an error, otherwise returns the event. A test-only helper for
// building the []ai.Event fixtures TestStreamShape drives.
func mustEvent(t *testing.T, ev ai.Event, err error) ai.Event {
	t.Helper()
	if err != nil {
		t.Fatalf("constructing test event: %v", err)
	}
	return ev
}

// TestStreamShape covers R-LSM-003: the three stream-shape invariants are
// asserted as three SEPARATE checks over a synthetic []ai.Event, never as
// one disjunction. Each case below isolates one (or, for the empty
// stream, all three) of the three checks so that suppressing any single
// one leaves the other two still able to fail independently (S-LSM-007).
func TestStreamShape(t *testing.T) {
	t.Parallel()

	responseStartEv, responseStartErr := ai.NewResponseStart("resp-shape", "model-shape")
	responseStart := mustEvent(t, responseStartEv, responseStartErr)
	completionEv, completionErr := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	completion := mustEvent(t, completionEv, completionErr)
	toolStartEv, toolStartErr := ai.NewToolCallStart(1, "call-shape-1", "tool-shape")
	toolStart := mustEvent(t, toolStartEv, toolStartErr)
	toolEndEv, toolEndErr := ai.NewToolCallEnd(1, []byte("{}"))
	toolEnd := mustEvent(t, toolEndEv, toolEndErr)

	tests := []struct {
		name                  string
		events                []ai.Event
		wantResponseStartFail bool
		wantContentFail       bool
		wantTerminalFail      bool
	}{
		{
			// S-LSM-007: response-start absent, content and a single
			// well-placed terminal both present — only the
			// response-start check fails.
			name:                  "missing start",
			events:                []ai.Event{toolStart, toolEnd, completion},
			wantResponseStartFail: true,
			wantContentFail:       false,
			wantTerminalFail:      false,
		},
		{
			// S-LSM-008: response-start and a single well-placed
			// terminal present, no content event between them — the
			// content-presence check fails specifically.
			name:                  "no content between start and terminal",
			events:                []ai.Event{responseStart, completion},
			wantResponseStartFail: false,
			wantContentFail:       true,
			wantTerminalFail:      false,
		},
		{
			// S-LSM-009 (second half): an empty stream must fail rather
			// than pass vacuously — all three checks fail.
			name:                  "zero terminals (empty stream)",
			events:                nil,
			wantResponseStartFail: true,
			wantContentFail:       true,
			wantTerminalFail:      true,
		},
		{
			// S-LSM-009 (first half): response-start and content both
			// present, but two terminal events — only the
			// terminal-exclusivity check fails.
			name:                  "two terminals",
			events:                []ai.Event{responseStart, toolStart, toolEnd, completion, completion},
			wantResponseStartFail: false,
			wantContentFail:       false,
			wantTerminalFail:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			report := checkStreamShape(tt.events)

			if gotFail := report.responseStartErr != nil; gotFail != tt.wantResponseStartFail {
				t.Errorf("responseStartErr = %v, want failure=%v (R-LSM-003)", report.responseStartErr, tt.wantResponseStartFail)
			}
			if gotFail := report.contentErr != nil; gotFail != tt.wantContentFail {
				t.Errorf("contentErr = %v, want failure=%v (R-LSM-003, S-LSM-008)", report.contentErr, tt.wantContentFail)
			}
			if gotFail := report.terminalErr != nil; gotFail != tt.wantTerminalFail {
				t.Errorf("terminalErr = %v, want failure=%v (R-LSM-003, S-LSM-009)", report.terminalErr, tt.wantTerminalFail)
			}
		})
	}
}

// TestCaptureTB covers work unit C task 2's captureTB double: it records
// every message into its sink, marks itself failed on Errorf/Fatalf (but
// not on Logf), and — critically — its Fatalf must NOT call the real
// runtime.Goexit()-driven FailNow path, so a caller that keeps going
// after tb.Fatalf(...) (as DrainAndRecord does, design.md D3) observes
// ordinary control flow, not an aborted goroutine.
func TestCaptureTB(t *testing.T) {
	t.Parallel()

	t.Run("Logf records without marking failed", func(t *testing.T) {
		t.Parallel()

		var sink bytes.Buffer
		tb := &captureTB{sink: &sink}
		tb.Helper()
		tb.Logf("informational: %s", "hello")

		if !strings.Contains(sink.String(), "informational: hello") {
			t.Errorf("sink = %q, want it to contain the Logf message", sink.String())
		}
		if tb.failed {
			t.Error("failed = true after Logf, want false — Logf must not mark failure")
		}
	})

	t.Run("Errorf records and marks failed", func(t *testing.T) {
		t.Parallel()

		var sink bytes.Buffer
		tb := &captureTB{sink: &sink}
		tb.Errorf("problem: %s", "boom")

		if !strings.Contains(sink.String(), "problem: boom") {
			t.Errorf("sink = %q, want it to contain the Errorf message", sink.String())
		}
		if !tb.failed {
			t.Error("failed = false after Errorf, want true")
		}
	})

	t.Run("Fatalf records, marks failed, and returns control to the caller", func(t *testing.T) {
		t.Parallel()

		var sink bytes.Buffer
		tb := &captureTB{sink: &sink}
		tb.Fatalf("fatal: %s", "kaboom")
		// Reaching this line at all is part of the proof: a real
		// testing.T.Fatalf calls runtime.Goexit() and this line would
		// never execute (design.md D3 — "verified safe: DrainAndRecord
		// returns after Fatalf").

		if !strings.Contains(sink.String(), "fatal: kaboom") {
			t.Errorf("sink = %q, want it to contain the Fatalf message", sink.String())
		}
		if !tb.failed {
			t.Error("failed = false after Fatalf, want true")
		}
	})

	t.Run("forwards nothing to a real testing.TB", func(t *testing.T) {
		t.Parallel()

		// The embedded testing.TB is left at its zero value (nil): every
		// method captureTB is actually exercised through here (Helper,
		// Logf, Errorf, Fatalf) is overridden below. If any call fell
		// through to the nil embedded interface, this subtest would
		// panic on a nil-interface method call — the negative-space
		// proof that captureTB forwards nothing to a real *testing.T.
		var sink bytes.Buffer
		tb := &captureTB{sink: &sink}
		tb.Helper()
		tb.Logf("a")
		tb.Errorf("b")
		tb.Fatalf("c")
		// No panic reaching here is the assertion.
	})
}
