// AI-38 — the OpenRouter reasoning-extension drop test (R-OR-06
// sub-scenario 2, R-ARS-015): an OpenRouter-shaped wire carrying
// the renamed `reasoning_details` array + `reasoning` string
// extension fields MUST drop both fields without leaking any byte
// into a text event, MUST NOT emit any reasoning-typed event, and
// MUST reach its normal completion.
//
// # What this file covers
//
// The shipped openaicompat/reasoning_absence_test.go covers the
// openai-compat `reasoning_content` extension field — its RED-
// found defect trail is recorded in Engram #2583's "openai-compat
// reasoning_absence_test.go precedent". OpenRouter renamed the
// field into TWO separate fields on its own dialect
// (delta.reasoning_details + delta.reasoning); this file covers
// the renamed pair under the same drop / not-leak / not-fail
// invariant the shipped test enforces for the original name.
//
// # What this file is not
//
// This is NOT a conformance-suite-driven run (R-CNF-001); it is a
// direct replay of one transcript against the conformance bridge's
// factory-built subject. The conformance-suite-driven test that
// covers the same scenario across the registry arrives in 2.5
// (run_for_test.go); this file is the single-fixture analogue
// that proves the renamed fields are dropped by the decoder
// before the suite-level run is layered on top.
//
// # TDD posture (work-unit 2.3)
//
// RED  : fixtures.OpenrouterReasoningExtensionStream did not exist
//        (unresolved identifier); the assertions below failed to
//        compile.
// GREEN: fixtures/reasoning_extensions.go +
//        fixtures/reasoning_extensions_accessor.go added;
//        TestOpenRouterAdapter_ReasoningExtensionField_DroppedNot
//        LeakedNotFailed PASSes against the recorded canonical
//        bytes.

//nolint:revive // underscore in package name per task plan § PR #2 2.1: "package openrouter_conformance"
package openrouter_conformance

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures"
)

// reasoningExtensionBridge builds a real *openaicompat.Client wired
// to an httptest.Server that serves bytes verbatim, with the bridge-
// local attribution round-tripper injecting the three OpenRouter-
// shaped attribution headers on every outbound request. It is the
// single-fixture analogue of conformanceBridgeFactory — that one
// is the suite-driven door, this one is the direct-replay door for
// a scenario the suite does not (and should not) enumerate as its
// own case.
func reasoningExtensionBridge(t *testing.T, transcript string) (ai.ModelProvider, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(transcript))
	}))
	t.Cleanup(server.Close)

	transport := bridgeAttributionRoundTripper{
		base:        server.Client().Transport,
		referer:     "https://app.cachicamas.test/openrouter-reasoning",
		xTitle:      "cachicamas-reasoning-drop",
		xCategories: "test,reasoning",
	}
	httpClient := &http.Client{Transport: transport}

	client, err := openaicompat.New(openaicompat.Config{
		Endpoint:   server.URL,
		Credential: openaicompat.NewCredential("reasoning-extension-token"),
		HTTPClient: httpClient,
	})
	if err != nil {
		t.Fatalf("openaicompat.New() error = %v, want nil", err)
	}
	return client, server
}

// reasoningExtensionMinimalRequest builds a minimal valid ai.Request
// the conformance bridge's Stream call drives against — one user
// message, one text part. Mirrors openaicompat's own validRequest
// helper but lives here because that one is unexported.
func reasoningExtensionMinimalRequest(t *testing.T) ai.Request {
	t.Helper()
	part, err := ai.NewText("hello")
	if err != nil {
		t.Fatalf("ai.NewText() error = %v, want nil", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	req, err := ai.NewRequest("requested-model", []ai.Message{msg})
	if err != nil {
		t.Fatalf("ai.NewRequest() error = %v, want nil", err)
	}
	return req
}

// assertNoSentinelLeak returns whether events contains no emitted
// text event that carries any byte of sentinel — the "drop, not
// leak" half of R-ARS-015 (S-ARS-036). The check is byte-substring
// across the union of every text event's delta payload: a single
// byte of the sentinel appearing in any one event fails the
// assertion.
//
// This helper is a faithful copy of the shipped
// openaicompat/reasoning_absence_test.go:83-97 helper — the same
// assertion, copied here because the shipped version is package-
// internal and this test lives in a sibling sub-package.
func assertNoSentinelLeak(events []ai.Event, sentinel string) bool {
	if len(sentinel) == 0 {
		return true
	}
	for _, ev := range events {
		d, ok := ev.TextDelta()
		if !ok {
			continue
		}
		if strings.Contains(d.Delta(), sentinel) {
			return false
		}
	}
	return true
}

// assertNoReasoningTypedEvent returns whether events contains no
// event of a reasoning-typed kind — the "drop, not emit-as-
// reasoning-event" half of R-ARS-015 (S-ARS-037). The kind set
// covers every reasoning-shaped kind the production registry
// enumerates (EventKindReasoningBlockStart, EventKindReasoning
// Delta, EventKindReasoningBlockEnd); a future addition to the
// registry will surface here as a compilation gap, which is the
// desired failure shape.
//
// Same helper, copied from
// openaicompat/reasoning_absence_test.go:106-116 — the package-
// internal version is unreachable from this sub-package.
func assertNoReasoningTypedEvent(events []ai.Event) bool {
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindReasoningBlockStart,
			ai.EventKindReasoningDelta,
			ai.EventKindReasoningBlockEnd:
			return false
		}
	}
	return true
}

// assertTerminalIsCompletionAndNoError returns whether events ends
// in the stream's normal completion terminal, with no error event
// anywhere — the "drop, not fail" half of R-ARS-015 (S-ARS-038).
// Empty events return false: a stream that drains to nothing has
// not reached its terminal.
//
// Same helper, copied from
// openaicompat/reasoning_absence_test.go:122-136 — the package-
// internal version is unreachable from this sub-package.
func assertTerminalIsCompletionAndNoError(events []ai.Event) bool {
	if len(events) == 0 {
		return false
	}
	last := events[len(events)-1]
	if last.Kind() != ai.EventKindCompletion {
		return false
	}
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			return false
		}
	}
	return true
}

// TestOpenRouterAdapter_ReasoningExtensionField_DroppedNotLeakedNotFailed
// covers R-OR-06 sub-scenario 2 + R-ARS-015 for the OpenRouter
// renamed pair (`reasoning_details` + `reasoning`): the
// recorded reasoning-extension fixture carries both renamed
// fields in distinct deltas; the conformance bridge's
// subject-built openaicompat client decodes it; the drop
// invariant (no leak, no reasoning-typed event, normal
// completion) holds for both.
//
// The test mirrors the shipped
// openaicompat/reasoning_absence_test.go
// TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAnd
// FixtureB but at the OpenRouter bridge level (different fixture
// field names; same mechanism).
func TestOpenRouterAdapter_ReasoningExtensionField_DroppedNotLeakedNotFailed(t *testing.T) {
	t.Parallel()

	client, _ := reasoningExtensionBridge(t, fixtures.OpenrouterReasoningExtensionStream())

	ch, err := client.Stream(context.Background(), reasoningExtensionMinimalRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	rec := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
	agenttest.RequireValidStream(t, rec)

	events := rec.Events()

	// R-ARS-015 / S-ARS-036 — no text event carries any byte of the
	// sentinel.
	if !assertNoSentinelLeak(events, fixtures.OpenrouterReasoningExtensionSentinel) {
		t.Errorf("a text event carried a byte of %q — reasoning_details / reasoning extension field leaked into text (R-OR-06 sub-scenario 2, R-ARS-015)",
			fixtures.OpenrouterReasoningExtensionSentinel)
	}

	// R-ARS-015 / S-ARS-037 — no reasoning-typed event is emitted
	// for either renamed field.
	if !assertNoReasoningTypedEvent(events) {
		t.Errorf("a reasoning-typed event was emitted — reasoning_details / reasoning extension field was mapped to a reasoning event (R-OR-06 sub-scenario 2, R-ARS-015)")
	}

	// R-ARS-015 / S-ARS-038 — the stream reaches its normal
	// completion terminal with no failure.
	if !assertTerminalIsCompletionAndNoError(events) {
		t.Errorf("terminal is not the normal completion event, or an error event appeared (R-OR-06 sub-scenario 2, R-ARS-015)")
	}

	// Sanity: the declared content survives intact (S-ARS-040
	// extended to the renamed-field shape). The conformance
	// bridge's subject must observe the two declared-content
	// deltas ("alpha", "omega") in order, with no reasoning byte
	// interleaved.
	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	want := []string{"alpha", "omega"}
	if len(deltas) != len(want) {
		t.Fatalf("emitted %d delta(s) = %q, want %d = %q (R-OR-06 sub-scenario 2 — declared content must arrive intact and in order, the renamed reasoning fields must not interleave bytes)",
			len(deltas), deltas, len(want), want)
	}
	for i := range want {
		if deltas[i] != want[i] {
			t.Errorf("delta[%d] = %q, want %q (R-OR-06 sub-scenario 2 — declared content must survive unmodified alongside the dropped reasoning fields)", i, deltas[i], want[i])
		}
	}
}
