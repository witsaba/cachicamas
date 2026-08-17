// AG-14 — Phase 5 and Phase 7: shutdown's typed-abort composition proof
// during a permission suspension (R-CAN-003 composition, S-APP-017's
// shutdown half) and shutdown's own wind-down plus terminal refusal
// (R-CAN-005, S-CAN-005).
package agent_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AG-14 — Phase 5: shutdown during a permission suspension aborts it
// typed, naming the shutdown signal (R-CAN-003 composition, S-APP-017's
// shutdown half). Same parked-call shape as
// TestHarness_Interrupt_DuringSuspensionAbortsTyped
// (cancellation_interrupt_test.go), the opposite signal fired — the
// assertions swap relative to that test so the typed value's identity,
// not merely its category, is proven to track the firing signal
// (S-APP-017's own point, cross-referenced S-CAN-003). By the time this
// test is written, R-CAN-003's underlying mechanism
// (typedCancellationFailureFromError, Phase 5.2) is already GREEN for
// the interrupt half, so this is a composition proof rather than a
// fresh RED — recorded, not silently assumed, with its real first-run
// output in apply-progress.md.
func TestHarness_Shutdown_DuringSuspensionAbortsTypedNamesShutdownSignal(t *testing.T) {
	t.Parallel()

	deferring := &scriptedPermissionPolicy{
		resolve: func(_ context.Context, _ ai.ToolCall) agent.PermissionVerdict {
			return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
		},
	}
	// Never released: the call must never reach Run — the gate aborts
	// it before executeCall ever calls tool.Run.
	tool := BlockingScriptedTool("read_can_003_shutdown", agent.EffectClassRead, make(chan struct{}))
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_can_003_shutdown": tool})

	provider := agenttest.NewProvider(scriptToolCallResponse(t, "call-can-003-shutdown", "read_can_003_shutdown", []byte(`{}`)))
	hist := agent.NewHistory()
	sched := &agent.Scheduler{}
	h := agent.Harness{
		Provider:  provider,
		System:    "system prompt for can-003-shutdown",
		Turn:      agent.TurnOptions{Tools: reg, PermissionPolicy: deferring},
		History:   hist,
		Scheduler: sched,
	}

	sink := make(chan *agent.Event, 256)
	type runOutcome struct {
		msg    ai.Message
		finish ai.FinishReason
		err    error
	}
	resultCh := make(chan runOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- runOutcome{msg, finish, err}
	}()

	events, decision := readUntilDecisionRequired(t, sink)
	if decision.CallID() != "call-can-003-shutdown" {
		t.Fatalf("permission_decision_required.CallID() = %q, want %q", decision.CallID(), "call-can-003-shutdown")
	}
	h.Shutdown()

	events = append(events, drainSink(t, sink)...)
	got := <-resultCh

	if got.err == nil {
		t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the shutdown sentinel")
	}
	if !errors.Is(got.err, agent.ErrShutdown) {
		t.Errorf("Run error = %v, want errors.Is(err, agent.ErrShutdown)", got.err)
	}
	if errors.Is(got.err, agent.ErrInterrupted) {
		t.Error("Run error unexpectedly satisfies errors.Is against agent.ErrInterrupted too — the two signals must stay distinguishable")
	}

	var failureEv agent.ToolEndExecutionFailure
	var found bool
	for _, ev := range events {
		if tef, ok := ev.ToolEndExecutionFailure(); ok && tef.CallID() == "call-can-003-shutdown" {
			failureEv, found = tef, true
		}
	}
	if !found {
		t.Fatal("no tool_end_execution_failure observed for call-can-003-shutdown")
	}
	failure, hasFailure := failureEv.Failure()
	if !hasFailure {
		t.Fatal("tool_end_execution_failure carries no *Failure")
	}
	if failure.Category() != ai.FailureCategoryCancellation {
		t.Errorf("failure.Category() = %v, want ai.FailureCategoryCancellation", failure.Category())
	}
	if !errors.Is(failure.Unwrap(), agent.ErrShutdown) {
		t.Error("failure does not unwrap to errors.Is-match agent.ErrShutdown")
	}
	if errors.Is(failure.Unwrap(), agent.ErrInterrupted) {
		t.Error("failure unexpectedly unwraps to errors.Is-match agent.ErrInterrupted too — the abort must name the firing signal, not a category constant")
	}

	if err := sched.WakeParked("call-can-003-shutdown"); !errors.Is(err, agent.ErrStrayDecision) {
		t.Errorf("WakeParked(%q) after wind-down = %v, want errors.Is to match agent.ErrStrayDecision", "call-can-003-shutdown", err)
	}

	if err := hist.CloseTurn(); err != nil {
		t.Errorf("history.CloseTurn() after wind-down = %v, want nil (no open call remains)", err)
	}

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(emitted stream) = %v, want no violation (unmodified acceptance)", report.Violation())
	}
}

// AG-14 — Phase 7: shutdown winds down identically to an interrupt and
// then terminally refuses new prompts (R-CAN-005, R-CAN-007 consumer,
// S-CAN-005, charter AG-14.2). Given a run in flight with the same
// shape S-CAN-001's Arm A uses, when the shutdown signal fires, then
// the wind-down produces the same synthesized-origin closures and
// nil-failure run-close shape as an interrupt, the run-end outcome is
// the shutdown member — distinct from both interrupted and failed —
// the returned error satisfies errors.Is against the shutdown sentinel
// and fails it against the interrupt sentinel; and when a second Run
// is then invoked on the same harness value, then it returns the typed
// post-shutdown refusal, that error satisfies errors.Is against the
// shutdown sentinel, the consumer observes no event at all, and
// CheckStream still accepts the first run's stream unmodified.
func TestHarness_Shutdown_WindsDownAndRefusesNewPrompts(t *testing.T) {
	t.Parallel()

	seedPart, err := ai.NewToolCall("call-can-005-seed", "read_can_005_seed", []byte(`{}`))
	if err != nil {
		t.Fatalf("ai.NewToolCall: %v", err)
	}
	seedMsg, err := ai.NewMessage(ai.RoleAssistant, seedPart)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	seeded, err := agent.NewSeededHistory([]ai.Message{seedMsg})
	if err != nil {
		t.Fatalf("agent.NewSeededHistory: %v", err)
	}

	gate := agenttest.NewGate()
	provider := agenttest.NewProvider(heldTurnScript(t, gate))
	h := agent.Harness{Provider: provider, System: "system prompt for can-005", History: seeded}

	sink := make(chan *agent.Event, 256)
	type runOutcome struct {
		msg    ai.Message
		finish ai.FinishReason
		err    error
	}
	resultCh := make(chan runOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- runOutcome{msg, finish, err}
	}()

	<-gate.Reached()
	h.Shutdown()

	events := drainSink(t, sink)
	got := <-resultCh

	if got.err == nil {
		t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the shutdown sentinel")
	}
	if !errors.Is(got.err, agent.ErrShutdown) {
		t.Errorf("first Run error = %v, want errors.Is(err, agent.ErrShutdown)", got.err)
	}
	if errors.Is(got.err, agent.ErrInterrupted) {
		t.Error("first Run error unexpectedly satisfies errors.Is against agent.ErrInterrupted too")
	}

	if len(events) == 0 {
		t.Fatal("zero events observed")
	}
	last := events[len(events)-1]
	runEnd, ok := last.RunEnd()
	if !ok {
		t.Fatalf("last event kind = %v, want run_end", last.Kind())
	}
	if runEnd.Outcome() != agent.RunOutcomeShutdown {
		t.Errorf("run_end outcome = %v, want RunOutcomeShutdown", runEnd.Outcome())
	}
	if runEnd.Outcome() == agent.RunOutcomeInterrupted {
		t.Error("run_end outcome = RunOutcomeInterrupted, want distinct from an interrupt's own outcome")
	}
	if runEnd.Outcome() == agent.RunOutcomeFailed {
		t.Error("run_end outcome = RunOutcomeFailed, want never Failed for a shut-down run")
	}
	if _, hasFailure := runEnd.Failure(); hasFailure {
		t.Error("run_end carries a *Failure, want none for a shut-down run")
	}

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(first run's emitted stream) = %v, want no violation (unmodified acceptance)", report.Violation())
	}

	// AG-14 remediation (verify-report.md MAJOR-3): this test shares
	// S-CAN-001 Arm A's exact heldTurnScript+gate shape, so it exercises
	// the SAME turn-close path (loop.go:442-449) with the shutdown
	// sentinel instead of the interrupt one — the turn-level outcome and
	// failure category must be identical, only the run-level outcome
	// differs (RunOutcomeShutdown here vs RunOutcomeInterrupted there).
	var turnEnd agent.TurnEnd
	var turnEndFound bool
	for _, ev := range events {
		if payload, ok := ev.TurnEnd(); ok {
			turnEnd, turnEndFound = payload, true
		}
	}
	if !turnEndFound {
		t.Fatal("no turn_end event observed, want the aborted turn's own turn_end")
	}
	if turnEnd.Outcome() != agent.TurnOutcomeAborted {
		t.Errorf("turn_end outcome = %v, want agent.TurnOutcomeAborted", turnEnd.Outcome())
	}
	turnFailure, hasTurnFailure := turnEnd.Failure()
	if !hasTurnFailure {
		t.Fatal("turn_end carries no *Failure, want one for an aborted turn")
	}
	if turnFailure.Category() != ai.FailureCategoryCancellation {
		t.Errorf("turn_end failure.Category() = %v, want ai.FailureCategoryCancellation", turnFailure.Category())
	}

	entries := seeded.Entries()
	if len(entries) != 3 {
		t.Fatalf("history has %d entries, want 3 (seeded call, run's own prompt, synthesized result)", len(entries))
	}
	if entries[2].Origin() != agent.EntryOriginSynthesized {
		t.Errorf("entries[2].Origin() = %v, want EntryOriginSynthesized — the same wind-down closure shape an interrupt produces", entries[2].Origin())
	}

	// Second Run on the same harness value: the typed post-shutdown
	// refusal, no event at all.
	secondSink := make(chan *agent.Event, 8)
	secondMsg, secondFinish, secondErr := h.Run(contextBackground(), firstMessage(t), secondSink)
	if !errors.Is(secondErr, agent.ErrPromptAfterShutdown) {
		t.Errorf("second Run error = %v, want errors.Is to match agent.ErrPromptAfterShutdown", secondErr)
	}
	if !errors.Is(secondErr, agent.ErrShutdown) {
		t.Errorf("second Run error = %v, want errors.Is to match agent.ErrShutdown (the refusal wraps it)", secondErr)
	}
	if !secondMsg.ID().IsZero() {
		t.Errorf("second Run message ID = %v, want the zero ai.MessageID", secondMsg.ID())
	}
	if secondFinish != 0 {
		t.Errorf("second Run finish = %v, want the zero ai.FinishReason", secondFinish)
	}
	secondEvents := drainSink(t, secondSink)
	if len(secondEvents) != 0 {
		t.Errorf("second Run emitted %d event(s), want zero", len(secondEvents))
	}
}
