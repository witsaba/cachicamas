// AG-22 — Phase 4 (RED, design D-G scenario 3): no-tracer parity
// (R-AGO-009, NFR-AOB-001-style), mirroring AI-37's own
// TestAI37_NoopEquivalence_DrainedSequencesEqual, driven against
// not-yet-instrumented code.
//
// # AG-22 correction (MAJOR-4, sdd-verify round 1)
//
// R-AGO-009/S-AGO-080 both say "element for element and value for
// value" / "with the same values". The original implementation compared
// only Kind() sequences, with its own comment conceding per-field
// equality "can never hold" — true of a NAIVE field-by-field compare,
// because two SEPARATELY scripted Harness.Run calls mint distinct
// RunID/TurnID/MessageID values REGARDLESS of tracing (mintHarnessRunID,
// mintLoopTurnID and mintLoopMessageID are process-global atomic
// counters/opaque generators, never seeded from the tracer). That is not
// a testing limitation to route around by narrowing the claim — it is a
// structural fact about identity minting, orthogonal to what R-AGO-009
// is actually asserting: that ADDING a tracer changes nothing about what
// the run does. agoParityRedactedEvent below projects OUT exactly the
// identifiers this codebase mints fresh per invocation — never anything
// else — so kind AND every remaining value (tool call IDs, arguments,
// results, outcomes, failure categories, text/reasoning fragment
// content, cost figures) are compared element for element, genuinely
// discharging "value for value" rather than restating "kind for kind"
// under a stronger-sounding name.
package agent_test

import (
	"slices"
	"strconv"
	"testing"

	"go.opentelemetry.io/otel/trace"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/agenttest/tracetest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// eventKinds maps events to their Kind(), in order — used ONLY for a
// fast pre-check (equal LENGTH and equal KIND sequence) before the real,
// value-carrying comparison below; kept because a length/kind mismatch
// produces a far more readable failure than the redacted-string
// comparison would on its own.
func eventKinds(events []agent.Event) []agent.EventKind {
	out := make([]agent.EventKind, len(events))
	for i, ev := range events {
		out[i] = ev.Kind()
	}
	return out
}

// agoParityRedactedEvent renders one event's kind plus every value R-AGO-
// 009's equality claim CAN hold across two separately-minted runs of the
// identical script — every field except the identifiers this codebase
// mints fresh per invocation (RunID, TurnID, MessageID), which the
// envelope's own Run()/Turn() and each per-kind payload's own MessageID
// accessor would otherwise smuggle in. Handles every event kind
// agoParityRun's own fixture (a single tool call, then a text-only turn)
// can produce; an unhandled kind falls back to the bare kind name —
// still stronger than the pre-check above catching nothing, and a
// panic-free default for a future caller who scripts a fixture with a
// kind this function does not yet special-case.
func agoParityRedactedEvent(ev agent.Event) string {
	switch ev.Kind() {
	case agent.EventKindRunStart:
		return "run_start"
	case agent.EventKindRunEnd:
		re, _ := ev.RunEnd()
		s := "run_end(outcome=" + re.Outcome().String()
		if f, ok := re.Failure(); ok {
			s += " category=" + f.Category().String()
		}
		return s + ")"
	case agent.EventKindTurnStart:
		return "turn_start"
	case agent.EventKindTurnEnd:
		te, _ := ev.TurnEnd()
		s := "turn_end(outcome=" + te.Outcome().String()
		if f, ok := te.Failure(); ok {
			s += " category=" + f.Category().String()
		}
		return s + ")"
	case agent.EventKindToolStart:
		ts, _ := ev.ToolStart()
		return "tool_start(call=" + ts.CallID() + " ordinal=" + strconv.FormatUint(uint64(ts.Ordinal()), 10) + " name=" + ts.Name() + " args=" + string(ts.Arguments()) + ")"
	case agent.EventKindToolEndSuccess:
		te, _ := ev.ToolEndSuccess()
		return "tool_end_success(call=" + te.CallID() + " ordinal=" + strconv.FormatUint(uint64(te.Ordinal()), 10) + " result=" + string(te.Result()) + ")"
	case agent.EventKindToolEndResultFailure:
		te, _ := ev.ToolEndResultFailure()
		return "tool_end_result_failure(call=" + te.CallID() + " ordinal=" + strconv.FormatUint(uint64(te.Ordinal()), 10) + " result=" + string(te.Result()) + ")"
	case agent.EventKindToolEndExecutionFailure:
		te, _ := ev.ToolEndExecutionFailure()
		s := "tool_end_execution_failure(call=" + te.CallID() + " ordinal=" + strconv.FormatUint(uint64(te.Ordinal()), 10)
		if f, ok := te.Failure(); ok {
			s += " category=" + f.Category().String()
		}
		return s + ")"
	case agent.EventKindMessageStartText:
		return "message_start_text"
	case agent.EventKindMessageDeltaText:
		d, _ := ev.MessageDeltaText()
		return "message_delta_text(idx=" + strconv.FormatUint(uint64(d.Index()), 10) + " fragment=" + strconv.Quote(d.Fragment()) + ")"
	case agent.EventKindMessageEndText:
		return "message_end_text"
	case agent.EventKindMessageStartReasoning:
		return "message_start_reasoning"
	case agent.EventKindMessageDeltaReasoning:
		d, _ := ev.MessageDeltaReasoning()
		return "message_delta_reasoning(idx=" + strconv.FormatUint(uint64(d.Index()), 10) + " fragment=" + strconv.Quote(string(d.Fragment())) + ")"
	case agent.EventKindMessageEndReasoning:
		return "message_end_reasoning"
	case agent.EventKindCostTurn:
		// CostTurn carries no minted identity of its own (label +
		// figures only) — ITS OWN String() (not Event.String(), which
		// embeds the RunID) is already identity-free: label and
		// input-token count (cost_events.go).
		ct, _ := ev.CostTurn()
		return ct.String()
	case agent.EventKindCostSession:
		cs, _ := ev.CostSession()
		return cs.String()
	default:
		return ev.Kind().String()
	}
}

// agoParityRedactedSequence projects events through
// agoParityRedactedEvent, in order.
func agoParityRedactedSequence(events []agent.Event) []string {
	out := make([]string, len(events))
	for i, ev := range events {
		out[i] = agoParityRedactedEvent(ev)
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
// produces the identical drained event sequence — element for element
// AND value for value, modulo the identifiers this codebase mints fresh
// per invocation regardless of tracing (agoParityRedactedEvent's own
// doc comment) — and neither arm panics.
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
		t.Fatalf("drained event-kind sequences differ:\n  untraced: %v\n  traced:   %v", kindsPlain, kindsTraced)
	}

	// R-AGO-009's own "value for value" clause, S-AGO-080: every field
	// EXCEPT the minted identities compared, element for element — not
	// merely the kind sequence again under a different name.
	redactedPlain := agoParityRedactedSequence(eventsPlain)
	redactedTraced := agoParityRedactedSequence(eventsTraced)
	if !slices.Equal(redactedPlain, redactedTraced) {
		t.Errorf("drained event sequences differ once minted identifiers are projected out (R-AGO-009, S-AGO-080):\n  untraced: %v\n  traced:   %v", redactedPlain, redactedTraced)
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
