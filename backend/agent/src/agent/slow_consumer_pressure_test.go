// AG-21 — Phase 3: stalled-consumer pressure (R-CNH-003, R-CNH-004;
// S-CNH-007..010). Charter AG-21.2 (0003:2012-2028).
//
// The stall is structural, never temporal (R-CNH-003): the sink is
// UNBUFFERED and the consumer stops calling receive at all, so the
// producer is genuinely blocked at sendStamped's own unconditional
// send (harness.go:284-287) with no receiver, by construction. No
// sleep, timeout or poll anywhere creates or detects it.
//
// sdd-verify round-2 (MAJOR-1): S-AGE-031 requires this same
// structural stall combined with one of R-CNH-001's four combined
// states PENDING ON THE SAME RUN. Scenarios 1 and 2 above are each
// standalone. cnhDriveCombinedStalledSteering below is the missing
// conjunction, built rather than narrowed away.
package agent_test

import (
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// cnhStalledConsumer drains an UNBUFFERED sink one event at a time
// until stopAfter reports true for an event just read (inclusive),
// then closes stalled — a happens-before edge proving the consumer
// has genuinely stopped reading, never inferred from elapsed time —
// and blocks on resume before draining the remainder to the sink's
// own close. It returns every event read, in arrival order, on the
// returned channel once the sink has closed.
func cnhStalledConsumer(sink <-chan *agent.Event, stopAfter func(agent.Event) bool, resume <-chan struct{}) (result <-chan []agent.Event, stalled <-chan struct{}) {
	out := make(chan []agent.Event, 1)
	reachedStall := make(chan struct{})
	go func() {
		var events []agent.Event
		for {
			ev, ok := <-sink
			if !ok {
				close(reachedStall)
				out <- events
				return
			}
			events = append(events, *ev)
			if stopAfter(*ev) {
				break
			}
		}
		close(reachedStall)
		<-resume
		for ev := range sink {
			events = append(events, *ev)
		}
		out <- events
	}()
	return out, reachedStall
}

// cnhStopAfterN returns a stopAfter predicate that fires once n events
// have been read, inclusive.
func cnhStopAfterN(n int) func(agent.Event) bool {
	count := 0
	return func(agent.Event) bool {
		count++
		return count >= n
	}
}

// cnhStopAfterToolStart returns a stopAfter predicate that fires on
// the tool_start event for callID — the same condition
// readUntilToolStart (cancellation_winddown_test.go) already reads
// for, generalized to this file's own predicate-driven consumer.
func cnhStopAfterToolStart(callID string) func(agent.Event) bool {
	return func(ev agent.Event) bool {
		start, ok := ev.ToolStart()
		return ok && start.CallID() == callID
	}
}

// cnhAssertCommittedFactsPresent is R-CNH-003's completeness check,
// measured against the run's own known, scripted identity set —
// deliberately NOT against a live reference run, whose production
// code could be equally sabotaged and equally silent, which is
// exactly the shape bite (b) defeats. It asserts exactly one tool-end
// event of wantKind for wantCallID, and exactly one cost_session
// carrying CostLabelFinal — AG-16's own guarantee that the run's
// already-real spend is reported regardless of how the run
// terminates (R-CST-006), which is what bite (b) removes.
func cnhAssertCommittedFactsPresent(t *testing.T, label string, events []agent.Event, wantCallID string, wantKind agent.EventKind) {
	t.Helper()

	toolEndCount := 0
	for _, ev := range events {
		if ev.Kind() != wantKind {
			continue
		}
		switch wantKind {
		case agent.EventKindToolEndSuccess:
			if end, ok := ev.ToolEndSuccess(); ok && end.CallID() == wantCallID {
				toolEndCount++
			}
		case agent.EventKindToolEndExecutionFailure:
			if end, ok := ev.ToolEndExecutionFailure(); ok && end.CallID() == wantCallID {
				toolEndCount++
			}
		}
	}
	if toolEndCount != 1 {
		t.Errorf("%s: %v count for call %q = %d, want exactly 1 (committed-fact event describing the tool result)", label, wantKind, wantCallID, toolEndCount)
	}

	costFinalCount := 0
	for _, ev := range events {
		if cs, ok := ev.CostSession(); ok && cs.Label() == agent.CostLabelFinal {
			costFinalCount++
		}
	}
	if costFinalCount != 1 {
		t.Errorf("%s: cost_session(Final) count = %d, want exactly 1 — the run's already-real spend must be reported regardless of how it terminates (R-CST-006)", label, costFinalCount)
	}
}

// ---------------------------------------------------------------------
// Scenario 1 — never cancelled (R-CNH-003; S-CNH-007).
// ---------------------------------------------------------------------

// TestSlowConsumerPressure_NeverCancelled_LosesNothing — S-CNH-007.
// Charter AG-21.2 scenario 1. Given an unbuffered sink whose consumer
// reads exactly k events and then blocks on a test-owned resume
// channel — so the producer is genuinely blocked at its unconditional
// send, by construction rather than by timing — and a run that is
// never cancelled, when the resume channel is closed and the
// consumer drains to completion, then the run completes, CheckStream
// accepts the whole stream unmodified, and every fact committed to
// the transcript has its describing event present on the stream,
// checked against the scripted event identity set (count, kinds and
// call identities) so that a single missing committed-fact event is a
// divergence; and because nothing was cancelled, zero events are
// absent — an absence excused as "sanctioned" (R-AGE-005) FAILS this
// scenario, since that path is unreachable here.
func TestSlowConsumerPressure_NeverCancelled_LosesNothing(t *testing.T) {
	t.Parallel()
	cnhDrivePressureNeverCancelled(t)
}

// cnhDrivePressureNeverCancelled is TestSlowConsumerPressure_
// NeverCancelled_LosesNothing's own callable body — reused, unchanged,
// as one of the leak sweep's own serial drivers (Phase 5, R-CNH-005).
func cnhDrivePressureNeverCancelled(t *testing.T) {
	t.Helper()

	callID := "call-" + cnhMintNonce()
	toolName := "cnh_pressure_tool_001"
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, callID, toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for cnh-pressure-001", Turn: agent.TurnOptions{Tools: reg}, History: hist}

	sink := make(chan *agent.Event) // UNBUFFERED — the structural stall (R-CNH-003).
	resultCh := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- cnhRunOutcome{msg, finish, err}
	}()

	resume := make(chan struct{})
	consumerOut, stalled := cnhStalledConsumer(sink, cnhStopAfterN(2), resume)
	<-stalled     // proven stalled: two events read, the consumer has genuinely stopped reading — never elapsed time.
	close(resume) // nothing cancels this scenario; the consumer simply resumes draining to completion.

	events := <-consumerOut
	got := <-resultCh

	if got.err != nil {
		t.Fatalf("Run returned err = %v, want nil (never cancelled)", got.err)
	}
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream(events) = %v, want nil (stream unmodified)", report.Violation())
	}
	requireHistoryPaired(t, hist)
	cnhAssertCommittedFactsPresent(t, "pressure/never-cancelled", events, callID, agent.EventKindToolEndSuccess)

	if len(events) == 0 {
		t.Fatal("zero events observed")
	}
	last := events[len(events)-1]
	runEnd, ok := last.RunEnd()
	if !ok {
		t.Fatalf("last event kind = %v, want run_end", last.Kind())
	}
	if runEnd.Outcome() != agent.RunOutcomeCompleted {
		t.Errorf("run_end outcome = %v, want RunOutcomeCompleted", runEnd.Outcome())
	}
}

// ---------------------------------------------------------------------
// Scenario 2 — cancellation unblocks a stalled stream within the
// documented bound (R-CNH-004; S-CNH-009).
// ---------------------------------------------------------------------

// TestSlowConsumerPressure_CancelledUnblocksWithinBound — S-CNH-009.
// Charter AG-21.2 scenario 2. Given a consumer stalled structurally as
// in R-CNH-003 after reading up to the tool-start of a
// cancellation-deaf tool, and a caller-owned scheduler injected with a
// small wind-down bound, when the interrupt signal fires WHILE the
// consumer is still stalled and the consumer then resumes draining,
// then the run returns, observed on its completion channel; the
// detached call is reported typed on the existing execution-failure
// kind, its *Failure reporting the cancellation category and
// errors.As-extracting a detached-call value that names the tool and
// the call identity (R-CAN-006); CheckStream accepts the stream
// unmodified; and no assertion in this scenario reads elapsed time.
func TestSlowConsumerPressure_CancelledUnblocksWithinBound(t *testing.T) {
	t.Parallel()
	cnhDrivePressureCancelledUnblocks(t)
}

// cnhDrivePressureCancelledUnblocks is TestSlowConsumerPressure_
// CancelledUnblocksWithinBound's own callable body — reused as one of
// the leak sweep's own serial drivers (Phase 5, R-CNH-005/R-CNH-006).
// The deaf tool's own release channel is closed explicitly, INLINE,
// after the run has returned — never left to t.Cleanup, which would
// pile up unfired across the sweep's 50 repeats and make every
// iteration's own detached goroutine look leaked at snapshot time
// (S-CNH-013's own "accounted, not leaked" distinction).
func cnhDrivePressureCancelledUnblocks(t *testing.T) {
	t.Helper()

	callID := "call-" + cnhMintNonce()
	toolName := "cnh_pressure_tool_002"
	release := make(chan struct{}) // never closed until AFTER the run returns, below.

	tool := BlockingScriptedTool(toolName, agent.EffectClassRead, release)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	provider := agenttest.NewProvider(scriptToolCallResponse(t, callID, toolName, []byte(`{}`)))
	hist := agent.NewHistory()
	sched := &agent.Scheduler{WindDownBound: smallWindDownBound}
	h := agent.Harness{
		Provider:  provider,
		System:    "system prompt for cnh-pressure-002",
		Turn:      agent.TurnOptions{Tools: reg},
		History:   hist,
		Scheduler: sched,
	}

	sink := make(chan *agent.Event) // UNBUFFERED — the structural stall (R-CNH-003).
	resultCh := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- cnhRunOutcome{msg, finish, err}
	}()

	resume := make(chan struct{})
	consumerOut, stalled := cnhStalledConsumer(sink, cnhStopAfterToolStart(callID), resume)
	<-stalled     // proven stalled: tool_start observed, the consumer has genuinely stopped reading.
	h.Interrupt() // fired WHILE the consumer is still stalled — the signal itself never needs a reader.
	close(resume) // the consumer resumes draining — required for the run's own pending unbuffered sends, including wind-down's, to ever be received.

	events := <-consumerOut
	got := <-resultCh // the run RETURNS — observed here, never by a wall-clock assertion.

	// AFTER the run returns: the detached goroutine is accounted for
	// (proven alive past the bound by the typed report below, now
	// proven to exit), not merely excluded from the leak count
	// (R-CNH-006, mirroring cancellation_winddown_test.go's own
	// TestHarness_WindDown_NoHarnessGoroutineRemains discipline).
	close(release)

	if got.err == nil {
		t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the interrupt sentinel")
	}
	if !errors.Is(got.err, agent.ErrInterrupted) {
		t.Errorf("Run error = %v, want errors.Is(_, agent.ErrInterrupted)", got.err)
	}

	var failureEv agent.ToolEndExecutionFailure
	var found bool
	for _, ev := range events {
		if tef, ok := ev.ToolEndExecutionFailure(); ok && tef.CallID() == callID {
			failureEv, found = tef, true
		}
	}
	if !found {
		t.Fatal("no tool_end_execution_failure observed for the deaf tool's call")
	}
	failure, hasFailure := failureEv.Failure()
	if !hasFailure {
		t.Fatal("tool_end_execution_failure carries no *Failure")
	}
	if failure.Category() != ai.FailureCategoryCancellation {
		t.Errorf("failure.Category() = %v, want ai.FailureCategoryCancellation", failure.Category())
	}
	var detached *agent.DetachedCallError
	if !errors.As(failure.Unwrap(), &detached) {
		t.Fatal("failure does not errors.As-extract *agent.DetachedCallError")
	}
	if detached.Tool != toolName {
		t.Errorf("detached.Tool = %q, want %q", detached.Tool, toolName)
	}
	if detached.CallID != callID {
		t.Errorf("detached.CallID = %q, want %q", detached.CallID, callID)
	}

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream(events) = %v, want nil (unmodified acceptance)", report.Violation())
	}
	requireHistoryPaired(t, hist)
	cnhAssertCommittedFactsPresent(t, "pressure/cancelled", events, callID, agent.EventKindToolEndExecutionFailure)
}

// ---------------------------------------------------------------------
// Combined pressure — a structurally stalled consumer AND one of
// R-CNH-001's four combined states pending on the SAME run
// (S-AGE-031; sdd-verify round-2, MAJOR-1). Scenarios 1 and 2 above are
// each standalone: real stalls, but never combined with a pending
// state. This is the missing conjunction, built rather than narrowed
// away — S-AGE-031's own GIVEN clauses are a conjunction on one run,
// and narrowing the text to match what already passed would certify
// the gap rather than close it.
// ---------------------------------------------------------------------

// cnhDriveCombinedStalledSteering combines R-CNH-003's structural stall
// (an unbuffered sink whose consumer has genuinely stopped reading)
// with R-CNH-001's steering-queued state, on the SAME run — the
// conjunction S-AGE-031 requires and that this file's own two
// standalone scenarios do not exercise.
//
// Steering is chosen over the other three combined states (suspension
// pending, compaction mid-run, child harness active) because its own
// pending-proof — <-gate.Reached() plus a synchronous nil Steer()
// return — is entirely INDEPENDENT of the event sink. run_start
// (harness.go:553) and turn_start (loop.go:309-312) are the only two
// events every path unconditionally emits before any provider call;
// once the consumer has read exactly those two and stopped (proven by
// cnhStalledConsumer's own stalled channel), heldTurnScript's Hold step
// — its first step, with nothing emitted ahead of it — is the very
// next thing the run can do, and reaching it needs no further sink
// read. "Consumer genuinely stalled" and "steering queued" are
// therefore each proven by their own happens-before edge and hold
// simultaneously BY CONSTRUCTION, never by inferring an overlap from
// timing. The other three states' own pending-proof
// (readUntilDecisionRequired, <-compGate.Reached() after a text
// bracket, or a read off the child's own stream) each consume a
// FURTHER event off the very same sink to prove pending — which would
// mean the stall had already been broken by the time "pending" is
// provable. Steering is the only one of the four where both
// conditions are cleanly, simultaneously provable without that
// tension, and it is also the lightest existing arrangement to extend
// (no permission subsystem, no second harness, no nested child).
//
// fireInterrupt selects S-AGE-031's two required arms: false drives
// the never-cancelled arm (mirrors cnhDrivePressureNeverCancelled —
// zero loss, run completes); true drives the signalled arm (mirrors
// cnhDrivePressureCancelledUnblocks — interrupt fires while the
// consumer is still stalled, and the run must still return with the
// matching outcome).
func cnhDriveCombinedStalledSteering(t *testing.T, fireInterrupt bool) {
	t.Helper()

	gate := agenttest.NewGate()
	t.Cleanup(gate.Release)

	turnOneScript := heldTurnScript(t, gate)
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := &agent.Harness{Provider: provider, System: "system prompt for cnh-combined-stalled-steering", History: hist}

	sink := make(chan *agent.Event) // UNBUFFERED -- the structural stall (R-CNH-003), combined with steering-queued (R-CNH-001) on the SAME run (S-AGE-031).
	resultCh := make(chan cnhRunOutcome, 1)
	go func() {
		msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
		resultCh <- cnhRunOutcome{msg, finish, err}
	}()

	resume := make(chan struct{})
	consumerOut, stalled := cnhStalledConsumer(sink, cnhStopAfterN(2), resume)
	<-stalled // proven stalled: run_start + turn_start read (the two events every path unconditionally emits before any provider call), the consumer has genuinely stopped reading -- never elapsed time.

	// The steering-queued state is now provable with no further read
	// off the (stalled) sink: the provider cannot have reached its Hold
	// step without those same two events already having been sent, so
	// gate.Reached() firing here is strictly ordered after the stall,
	// not merely likely to be.
	select {
	case <-gate.Reached():
	case <-time.After(5 * time.Second):
		t.Fatal("cnh combined-stalled-steering: turn one never reached the Hold gate")
	}

	steered, err := ai.NewMessage(ai.RoleUser, mustText(t, "cnh combined-stalled-steering message"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	if err := h.Steer(steered); err != nil {
		t.Fatalf("Steer returned %v, want nil -- the message must be accepted while the gate is still held and the consumer is still stalled", err)
	}

	// Fire this arm's resolution WHILE the consumer is STILL stalled --
	// the exact moment S-AGE-031 requires both conditions to coincide.
	if fireInterrupt {
		h.Interrupt()
	}
	gate.Release() // arrangeSteeringQueued's own safety-net sequencing (combined_state_fixtures_test.go:343/347): released either way, harmless once already cancelled.
	close(resume)  // the consumer resumes draining -- required for the run's own pending unbuffered sends, blocked while it was stalled, to ever be received.

	events := <-consumerOut
	got := <-resultCh

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("combined/stalled-steering: CheckStream(events) = %v, want nil (stream unmodified)", report.Violation())
	}
	requireHistoryPaired(t, hist)

	if len(events) == 0 {
		t.Fatal("combined/stalled-steering: zero events observed")
	}
	last := events[len(events)-1]
	runEnd, ok := last.RunEnd()
	if !ok {
		t.Fatalf("combined/stalled-steering: last event kind = %v, want run_end", last.Kind())
	}

	// Completeness (R-AGE-006, S-AGE-031): the run's real spend is a
	// committed fact on EVERY outcome (R-CST-006) -- present exactly
	// once, checked against the scripted identity set, never a raw
	// total. The same mechanism bite (b) (task 3.4) defeat-tested
	// standalone; here it is proven under the combination for the
	// first time.
	costFinalCount := 0
	for _, ev := range events {
		if cs, ok := ev.CostSession(); ok && cs.Label() == agent.CostLabelFinal {
			costFinalCount++
		}
	}
	if costFinalCount != 1 {
		t.Errorf("combined/stalled-steering: cost_session(Final) count = %d, want exactly 1 (committed-fact completeness, R-CST-006)", costFinalCount)
	}

	if fireInterrupt {
		if !errors.Is(got.err, agent.ErrInterrupted) {
			t.Errorf("combined/stalled-steering/interrupt: Run error = %v, want errors.Is(_, agent.ErrInterrupted)", got.err)
		}
		if runEnd.Outcome() != agent.RunOutcomeInterrupted {
			t.Errorf("combined/stalled-steering/interrupt: run_end outcome = %v, want RunOutcomeInterrupted", runEnd.Outcome())
		}
		return
	}

	if got.err != nil {
		t.Fatalf("combined/stalled-steering/never-cancelled: Run returned err = %v, want nil", got.err)
	}
	if runEnd.Outcome() != agent.RunOutcomeCompleted {
		t.Errorf("combined/stalled-steering/never-cancelled: run_end outcome = %v, want RunOutcomeCompleted", runEnd.Outcome())
	}

	// The queued-and-undelivered steer message itself survives the
	// combined pressure: it lands in the durable transcript between
	// turn one's assistant message and turn two's -- the S-RUN-070
	// shape (harness_steering_test.go:107-113), now proven under a
	// genuinely stalled consumer rather than a 256-deep buffer.
	entries := hist.Entries()
	if len(entries) != 4 {
		t.Fatalf("combined/stalled-steering/never-cancelled: history has %d entries, want 4 (prompt, turn-one assistant, steered, turn-two assistant)", len(entries))
	}
	if entries[2].Message().Role() != ai.RoleUser {
		t.Errorf("combined/stalled-steering/never-cancelled: entries[2].Message().Role() = %v, want RoleUser (the steered message)", entries[2].Message().Role())
	}
}

// TestCombinedPressure_StalledSteering_NeverCancelled_LosesNothing --
// S-AGE-031's never-cancelled arm: R-CNH-003's structural stall
// combined with R-CNH-001's steering-queued state, on the same run.
// See cnhDriveCombinedStalledSteering.
func TestCombinedPressure_StalledSteering_NeverCancelled_LosesNothing(t *testing.T) {
	t.Parallel()
	cnhDriveCombinedStalledSteering(t, false)
}

// TestCombinedPressure_StalledSteering_Interrupted -- S-AGE-031's
// signalled arm: the same combination, interrupted while the consumer
// is still stalled. See cnhDriveCombinedStalledSteering.
func TestCombinedPressure_StalledSteering_Interrupted(t *testing.T) {
	t.Parallel()
	cnhDriveCombinedStalledSteering(t, true)
}
