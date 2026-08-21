// AG-22 — Phase 4 (RED, design D-G scenario 3): no-tracer parity
// (R-AGO-009, NFR-AOB-001-style), mirroring AI-37's own
// TestAI37_NoopEquivalence_DrainedSequencesEqual, driven against
// not-yet-instrumented code.
package agent_test

import (
	"slices"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// eventKinds maps events to their Kind(), in order — the shape a parity
// proof compares: two SEPARATELY scripted runs mint distinct RunID/
// TurnID values by construction, so exact per-field equality can never
// hold between them; the KIND sequence is what "identical event
// sequence" (design D-G scenario 3) means here.
func eventKinds(events []agent.Event) []agent.EventKind {
	out := make([]agent.EventKind, len(events))
	for i, ev := range events {
		out[i] = ev.Kind()
	}
	return out
}

// agoParityRun drives one scripted tool-calling run with provider
// injected as Harness.TracerProvider (nil for the untraced arm) and
// returns the drained events. A panic on either arm fails the test by
// name (R-AGO-009: no adapter-side nil check on a tracing value may ever
// be needed, mirroring AI-37's own S-AOB-037).
func agoParityRun(t *testing.T, toolName string, provider trace.TracerProvider) []agent.Event {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Run panicked with TracerProvider=%v: %v (R-AGO-009 forbids this)", provider, r)
		}
	}()

	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	turnOneScript := scriptToolCallResponse(t, "call-parity-001", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	modelProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{
		Provider:       modelProvider,
		System:         "system prompt for no-tracer parity",
		Turn:           agent.TurnOptions{Tools: reg},
		TracerProvider: provider,
	}
	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	return drainSink(t, sink)
}

// TestObservability_NoTracerParity_DrainedSequencesEqual covers
// R-AGO-009: the identical scripted run, driven once with no
// TracerProvider configured and once with a recording tracetest.Provider,
// produces the identical drained event-kind sequence and neither arm
// panics.
func TestObservability_NoTracerParity_DrainedSequencesEqual(t *testing.T) {
	t.Parallel()

	eventsPlain := agoParityRun(t, "parity_tool_a", nil)
	provider := tracetest.NewProvider()
	eventsTraced := agoParityRun(t, "parity_tool_a", provider)

	if len(eventsPlain) == 0 {
		t.Fatal("drained zero events on the untraced arm, want a non-trivial scripted run (the equality below would be vacuous)")
	}
	kindsPlain := eventKinds(eventsPlain)
	kindsTraced := eventKinds(eventsTraced)
	if !slices.Equal(kindsPlain, kindsTraced) {
		t.Errorf("drained event-kind sequences differ:\n  untraced: %v\n  traced:   %v", kindsPlain, kindsTraced)
	}
}

// TestObservability_NoTracerParity_TracedArmRecordsAtLeastOneSpan is
// design D-G scenario 3's own non-vacuity floor: the traced arm's
// provider must have recorded at least one span, or the parity claim
// above (and everywhere else in this milestone) would hold vacuously —
// two empty sequences are trivially "equal". This is the genuine RED
// right now (design's bite #4): the run/turn/tool seams are not
// instrumented until Phases 5-7, so provider.Started() == 0.
func TestObservability_NoTracerParity_TracedArmRecordsAtLeastOneSpan(t *testing.T) {
	t.Parallel()

	provider := tracetest.NewProvider()
	agoParityRun(t, "parity_tool_b", provider)

	if got := provider.Started(); got == 0 {
		t.Fatalf("provider.Started() = 0, want > 0 — no span was recorded on the traced arm yet (the run/turn/tool seams are not instrumented until Phases 5-7)")
	}
}
