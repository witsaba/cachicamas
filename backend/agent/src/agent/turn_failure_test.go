// AG-11.2 — typed failure reaches the caller: the mid-stream fatal path
// emits turn_end(Aborted, *Failure) + run_end(Failed, *Failure) before
// the sink closes (R-ATT-005), the partial-output discriminator and
// the failure-identity pin (R-ATT-006, jointly with the failure-
// construction pin), the surviving partial assistant content
// (R-ATT-007), and the exactly-one-provider-call guarantee (R-ATT-008,
// D7). package agent_test (NFR-ATT-001).
//
// D1's type fork is proven from both arms: the provider-*ai.Failure arm
// (every test in this file but the last) and the internal-construction-
// error arm, which stays byte-unchanged from pre-AG-11
// (TestTurn_InternalErrorArm_EmitsNothing).
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// scriptTextThenTerminalFailure builds a script that delivers one text
// bracket (content byte-for-byte "partial-before-failure") and then a
// terminal mid-stream ai.ErrorEvent of the given category, with
// outputPreceded recorded as instructed — this file's shared fixture
// for the fatal-path scenarios.
func scriptTextThenTerminalFailure(t *testing.T, category ai.FailureCategory, outputPreceded bool) agenttest.Script {
	t.Helper()

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	delta, err := ai.NewTextDelta(1, "partial-before-failure")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	failure, err := ai.MidStreamFailure(ai.FailureReport{Category: category}, outputPreceded)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}
	terminal, err := ai.ErrorEvent(failure)
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(terminal),
	}}
}

// S-ATT-007 — typed brackets on the fatal path, plus task 5.3's
// failure-identity pin (D1's single-construction intent): turnEnd.Failure()
// and runEnd.Failure() must be the IDENTICAL *agent.Failure pointer, not
// merely equal values.
func TestTurn_FatalPath_EmitsTypedBrackets(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextThenTerminalFailure(t, ai.FailureCategoryUnavailable, true))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for fatal path",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err == nil {
		t.Fatal("agent.Turn returned err = nil, want a non-nil error (mid-stream terminal failure)")
	}

	events := drainSink(t, sink)
	if len(events) < 2 {
		t.Fatalf("observed %d event(s), want at least turn_end + run_end", len(events))
	}
	turnEndEv := events[len(events)-2]
	runEndEv := events[len(events)-1]

	turnEnd, ok := turnEndEv.TurnEnd()
	if !ok {
		t.Fatalf("second-to-last event (kind %v) is not a turn_end payload", turnEndEv.Kind())
	}
	if turnEnd.Outcome() != agent.TurnOutcomeAborted {
		t.Errorf("turn_end.Outcome() = %v, want TurnOutcomeAborted", turnEnd.Outcome())
	}
	turnFailure, hasTurnFailure := turnEnd.Failure()
	if !hasTurnFailure || turnFailure == nil {
		t.Fatal("turn_end carries no Failure, want a non-nil *agent.Failure")
	}

	runEnd, ok := runEndEv.RunEnd()
	if !ok {
		t.Fatalf("last event (kind %v) is not a run_end payload", runEndEv.Kind())
	}
	if runEnd.Outcome() != agent.RunOutcomeFailed {
		t.Errorf("run_end.Outcome() = %v, want RunOutcomeFailed", runEnd.Outcome())
	}
	runFailure, hasRunFailure := runEnd.Failure()
	if !hasRunFailure || runFailure == nil {
		t.Fatal("run_end carries no Failure, want a non-nil *agent.Failure")
	}

	// Task 5.3 — pointer identity, not value equality: one NewFailure
	// call feeds both emissions.
	if turnFailure != runFailure {
		t.Error("turn_end.Failure() and run_end.Failure() are different *agent.Failure values, want the identical pointer")
	}

	// No other turn_end/run_end sneaks in earlier than the last two
	// positions (order: ... turn_end, run_end, [close]).
	for i, ev := range events[:len(events)-2] {
		if ev.Kind() == agent.EventKindTurnEnd || ev.Kind() == agent.EventKindRunEnd {
			t.Errorf("event[%d] is a %v terminal event before the expected position — want turn_end then run_end as the last two events", i, ev.Kind())
		}
	}
}

// S-ATT-009 — partial content survives the failure: the returned msg
// carries the delivered text byte-for-byte, the emitted deltas
// reconstruct to that same message via the AG-05.3 helper, and the
// turn_end's *Failure reports PartialOutput() == true.
func TestTurn_PartialContentSurvives(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextThenTerminalFailure(t, ai.FailureCategoryTimeout, true))
	sink := make(chan *agent.Event, 16)

	msg, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for partial content",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err == nil {
		t.Fatal("agent.Turn returned err = nil, want a non-nil error")
	}
	if len(msg.Content()) == 0 {
		t.Fatal("msg has no content, want the delivered text bracket (not the zero ai.Message{})")
	}
	text, ok := msg.Content()[0].Text()
	if !ok || text != "partial-before-failure" {
		t.Errorf("msg.Content()[0] text = (%q, %v), want (%q, true) — byte-for-byte", text, ok, "partial-before-failure")
	}

	events := drainSink(t, sink)
	recon, ok := reconstructLoopMessage(events)
	if !ok {
		t.Fatal("reconstructLoopMessage(events) returned no reconstruction")
	}
	fromEmit := msgFromParts(recon.Parts)
	if !equalByContent(msg, fromEmit) {
		t.Errorf("returned msg differs from emitted-events reconstruction:\n  returned: %q\n  emitted:  %q", msgContentText(msg), msgContentText(fromEmit))
	}

	var turnFailure *agent.Failure
	for _, ev := range events {
		if payload, ok := ev.TurnEnd(); ok {
			turnFailure, _ = payload.Failure()
		}
	}
	if turnFailure == nil {
		t.Fatal("no turn_end Failure observed")
	}
	if !turnFailure.PartialOutput() {
		t.Error("turnFailure.PartialOutput() = false, want true — content was delivered before the failure")
	}
}

// S-ATT-010 (D7) — exactly one provider call on any failing turn, both
// when content preceded the failure and when the failure is the very
// first event. A second script is queued but must never be consumed —
// if the loop retried, Requests() would report 2.
func TestTurn_ExactlyOneProviderCall(t *testing.T) {
	t.Run("failure after partial content", func(t *testing.T) {
		t.Parallel()
		secondScript := scriptTerminationResponse(t, ai.FinishReasonStop)
		provider := agenttest.NewProvider(scriptTextThenTerminalFailure(t, ai.FailureCategoryUnavailable, true), secondScript)
		sink := make(chan *agent.Event, 16)

		_, _, err := agent.Turn(contextBackground(), provider, "system prompt for exactly-one (partial)", []ai.Message{firstMessage(t)}, agent.TurnOptions{}, sink)
		if err == nil {
			t.Fatal("agent.Turn returned err = nil, want a non-nil error")
		}
		_ = drainSink(t, sink)

		if got := len(provider.Requests()); got != 1 {
			t.Errorf("provider.Requests() has %d entry/entries, want exactly 1 — the loop must never issue a second call on a failing turn", got)
		}
	})

	t.Run("failure before any content", func(t *testing.T) {
		t.Parallel()
		failure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
		if err != nil {
			t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
		}
		terminal, err := ai.ErrorEvent(failure)
		if err != nil {
			t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
		}
		immediateFailScript := agenttest.Script{Steps: []agenttest.Step{agenttest.Emit(terminal)}}
		secondScript := scriptTerminationResponse(t, ai.FinishReasonStop)
		provider := agenttest.NewProvider(immediateFailScript, secondScript)
		sink := make(chan *agent.Event, 16)

		_, _, err = agent.Turn(contextBackground(), provider, "system prompt for exactly-one (immediate)", []ai.Message{firstMessage(t)}, agent.TurnOptions{}, sink)
		if err == nil {
			t.Fatal("agent.Turn returned err = nil, want a non-nil error")
		}
		_ = drainSink(t, sink)

		if got := len(provider.Requests()); got != 1 {
			t.Errorf("provider.Requests() has %d entry/entries, want exactly 1 — the loop must never issue a second call on a failing turn", got)
		}
	})
}

// TestTurn_InternalErrorArm_EmitsNothing pins D1's type fork from its
// OTHER arm: an internal construction error (a plain Go error, never a
// *ai.Failure) MUST stay byte-unchanged from pre-AG-11 — drain, close,
// return (ai.Message{}, 0, err) — with NO turn_end/run_end emitted.
// Forced via a tool-call whose accumulated argument bytes are not
// well-formed JSON: NewToolCall's own validation (ai/tool_call.go)
// rejects it, setting turn.fatal to a *ai.Violation, which does not
// type-assert to *ai.Failure.
func TestTurn_InternalErrorArm_EmitsNothing(t *testing.T) {
	t.Parallel()

	start, err := ai.NewToolCallStart(1, "call-att-internal-error", "probe-tool")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart returned %v, want no failure", err)
	}
	delta, err := ai.NewToolCallDelta(1, []byte("not-well-formed-json"))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta returned %v, want no failure", err)
	}
	end, err := ai.NewToolCallEnd(1, []byte("not-well-formed-json"))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd returned %v, want no failure", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
	}}
	provider := agenttest.NewProvider(script)
	sink := make(chan *agent.Event, 16)

	msg, finish, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for internal-error arm",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err == nil {
		t.Fatal("agent.Turn returned err = nil, want a non-nil error (malformed tool-call arguments, an internal construction failure)")
	}
	if _, ok := err.(*ai.Failure); ok {
		t.Error("agent.Turn returned a *ai.Failure for an internal construction error, want a plain Go error (D1: only the provider-*ai.Failure arm emits typed brackets)")
	}
	if len(msg.Content()) != 0 {
		t.Errorf("msg.Content() has %d part(s), want 0 (the zero ai.Message{}) — the internal-error arm is byte-unchanged from pre-AG-11 (D1)", len(msg.Content()))
	}
	if finish != 0 {
		t.Errorf("finish = %v, want 0 — the internal-error arm is byte-unchanged from pre-AG-11 (D1)", finish)
	}

	events := drainSink(t, sink)
	for _, ev := range events {
		if ev.Kind() == agent.EventKindTurnEnd {
			t.Error("observed a turn_end event on the internal-construction-error arm, want none (D1)")
		}
		if ev.Kind() == agent.EventKindRunEnd {
			t.Error("observed a run_end event on the internal-construction-error arm, want none (D1)")
		}
	}
}

// S-AEV-075 — the loop delivers the typed failure through the
// registered outcomes: turn_end(Aborted) and run_end(Failed) each carry
// a non-nil *Failure whose Category(), Delivery(), Retryable() and
// PartialOutput() are all inspectable as typed values; no assertion
// reads a message string (S-AEV-073 carried forward).
func TestTurn_TypedFailureFullyInspectable(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextThenTerminalFailure(t, ai.FailureCategoryRateLimit, true))
	sink := make(chan *agent.Event, 16)

	_, _, err := agent.Turn(
		contextBackground(),
		provider,
		"system prompt for full inspectability",
		[]ai.Message{firstMessage(t)},
		agent.TurnOptions{},
		sink,
	)
	if err == nil {
		t.Fatal("agent.Turn returned err = nil, want a non-nil error")
	}

	events := drainSink(t, sink)
	seenZeroKind := false
	var turnFailure, runFailure *agent.Failure
	for _, ev := range events {
		if ev.Kind() == 0 {
			seenZeroKind = true
		}
		if payload, ok := ev.TurnEnd(); ok {
			turnFailure, _ = payload.Failure()
		}
		if payload, ok := ev.RunEnd(); ok {
			runFailure, _ = payload.Failure()
		}
	}
	if seenZeroKind {
		t.Error("observed the zero EventKind, want every event to carry a registered kind (no new EventKind for failure, D8)")
	}
	if turnFailure == nil {
		t.Fatal("no turn_end Failure observed")
	}
	if runFailure == nil {
		t.Fatal("no run_end Failure observed")
	}

	for name, f := range map[string]*agent.Failure{"turn_end": turnFailure, "run_end": runFailure} {
		if got := f.Category(); got != ai.FailureCategoryRateLimit {
			t.Errorf("%s failure.Category() = %v, want %v", name, got, ai.FailureCategoryRateLimit)
		}
		if got := f.Delivery(); got != ai.DeliveryMidStream {
			t.Errorf("%s failure.Delivery() = %v, want %v", name, got, ai.DeliveryMidStream)
		}
		_ = f.Retryable() // inspectable as a typed value; the category's own default is not this scenario's concern
		if !f.PartialOutput() {
			t.Errorf("%s failure.PartialOutput() = false, want true", name)
		}
	}
}
