// AG-21 — Phase 3: stalled-consumer pressure (R-CNH-003, R-CNH-004;
// S-CNH-007..010). Charter AG-21.2 (0003:2012-2028).
//
// The stall is structural, never temporal (R-CNH-003): the sink is
// UNBUFFERED and the consumer stops calling receive at all, so the
// producer is genuinely blocked at sendStamped's own unconditional
// send (harness.go:284-287) with no receiver, by construction. No
// sleep, timeout or poll anywhere creates or detects it.
package agent_test

import (
	"errors"
	"testing"

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

	callID := "call-cnh-pressure-001"
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

	callID := "call-cnh-pressure-002"
	toolName := "cnh_pressure_tool_002"
	release := make(chan struct{}) // never closed during the run: the deaf tool ignores ctx entirely.
	t.Cleanup(func() { close(release) })

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
	<-stalled       // proven stalled: tool_start observed, the consumer has genuinely stopped reading.
	h.Interrupt()   // fired WHILE the consumer is still stalled — the signal itself never needs a reader.
	close(resume)   // the consumer resumes draining — required for the run's own pending unbuffered sends, including wind-down's, to ever be received.

	events := <-consumerOut
	got := <-resultCh // the run RETURNS — observed here, never by a wall-clock assertion.

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
