// AG-14 — Phase 9 and Phase 10: the wind-down bound (R-CAN-006 Thens
// 1-2, S-CAN-006) and the goroutine-leak proof (R-CAN-006 Then 3, part
// of S-CAN-006), serial-only.
package agent_test

import (
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// smallWindDownBound is the injected wind-down bound this file's tests
// use in place of the 100ms package default (design.md §3.1): small
// enough to keep the leak test's 50 repeats fast. Every deaf tool in
// this file never releases until the test itself closes `release`, so
// the bound always fires — its exact duration only controls how fast
// the test runs, never whether the assertion holds (no flakiness from
// picking a small value).
const smallWindDownBound = 20 * time.Millisecond

// readUntilToolStart reads events off sink, in arrival order, until a
// tool_start event for callID is observed (inclusive), returning every
// event read so far. AG-14 Phase 9 (S-CAN-006): the stream is the
// synchronization point for firing a signal once a call has genuinely
// been dispatched — mirrors readUntilDecisionRequired
// (cancellation_interrupt_test.go) for a call executed directly rather
// than parked on the permission gate. The deadline below is a
// defensive guard against a genuinely stuck test, never the signal
// itself (NFR-CAN-002).
func readUntilToolStart(t *testing.T, sink <-chan *agent.Event, callID string) []agent.Event {
	t.Helper()
	var events []agent.Event
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev, ok := <-sink:
			if !ok {
				t.Fatal("readUntilToolStart: sink closed before a tool_start event was observed")
				return nil
			}
			events = append(events, *ev)
			if start, ok := ev.ToolStart(); ok && start.CallID() == callID {
				return events
			}
		case <-deadline:
			t.Fatal("readUntilToolStart: no tool_start event observed within 2s")
			return nil
		}
	}
}

// AG-14 — Phase 9 (S-CAN-006, charter AG-14.3, Thens 1-2). Given a run
// whose scheduled call uses a cancellation-deaf scripted tool blocking
// on a channel the test never closes during the run, and a
// caller-owned scheduler injected with a small wind-down bound, when
// the interrupt signal fires, then the run RETURNS — observed by a
// read on the run's completion channel, never a wall-clock assertion —
// and the stream carries a tool_end_execution_failure for that call
// whose *Failure errors.As-extracts a DetachedCallError naming that
// tool and that call's identity.
func TestHarness_WindDown_DeafToolCannotHoldRunHostage(t *testing.T) {
	t.Parallel()

	release := make(chan struct{}) // never closed during the run: the deaf tool ignores ctx entirely.
	t.Cleanup(func() { close(release) })

	tool := BlockingScriptedTool("read_can_006", agent.EffectClassRead, release)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_can_006": tool})

	provider := agenttest.NewProvider(scriptToolCallResponse(t, "call-can-006", "read_can_006", []byte(`{}`)))
	hist := agent.NewHistory()
	sched := &agent.Scheduler{WindDownBound: smallWindDownBound}
	h := agent.Harness{
		Provider:  provider,
		System:    "system prompt for can-006",
		Turn:      agent.TurnOptions{Tools: reg},
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

	events := readUntilToolStart(t, sink, "call-can-006")
	h.Interrupt()

	events = append(events, drainSink(t, sink)...)
	got := <-resultCh // the run RETURNS — observed here, never by a wall-clock assertion.

	if got.err == nil {
		t.Fatal("Run returned err = nil, want a non-nil error satisfying errors.Is against the interrupt sentinel")
	}
	if !errors.Is(got.err, agent.ErrInterrupted) {
		t.Errorf("Run error = %v, want errors.Is(err, agent.ErrInterrupted)", got.err)
	}

	var failureEv agent.ToolEndExecutionFailure
	var found bool
	for _, ev := range events {
		if tef, ok := ev.ToolEndExecutionFailure(); ok && tef.CallID() == "call-can-006" {
			failureEv, found = tef, true
		}
	}
	if !found {
		t.Fatal("no tool_end_execution_failure observed for call-can-006")
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
	if detached.Tool != "read_can_006" {
		t.Errorf("detached.Tool = %q, want %q", detached.Tool, "read_can_006")
	}
	if detached.CallID != "call-can-006" {
		t.Errorf("detached.CallID = %q, want %q", detached.CallID, "call-can-006")
	}

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(emitted stream) = %v, want no violation (unmodified acceptance)", report.Violation())
	}
}

// AG-14 — Phase 10 (R-CAN-006 Then 3, part of S-CAN-006). No
// t.Parallel(): mechanically enforced by RequireNoGoroutineLeak's own
// tb.Setenv sentinel (stream_kit_leak.go:80,110) — this test, or an
// ancestor, MUST NOT call t.Parallel(). Repeats the deaf-tool interrupt
// scenario leakRepeats (50) times; each iteration closes its OWN
// `release` channel only AFTER the run has returned, so the detached
// third-party goroutine is ACCOUNTED FOR — proven alive past the bound
// by the typed report the previous test already confirms, proven to
// exit once the third-party code returns — rather than merely excluded
// from the count. A plain exclusion (never releasing, or releasing
// before the run returns) would silently pass the raw goroutine-count
// check while still violating R-CAN-006 — the discipline both
// design.md and tasks.md flag explicitly. A fresh Harness value is
// constructed per iteration so this test stays independent of
// S-CAN-002's own same-harness-reuse proof.
func TestHarness_WindDown_NoHarnessGoroutineRemains(t *testing.T) {
	scenario := func() {
		release := make(chan struct{})
		tool := BlockingScriptedTool("read_can_006_leak", agent.EffectClassRead, release)
		reg := agent.NewMapRegistry(map[string]agent.Tool{"read_can_006_leak": tool})

		provider := agenttest.NewProvider(scriptToolCallResponse(t, "call-can-006-leak", "read_can_006_leak", []byte(`{}`)))
		sched := &agent.Scheduler{WindDownBound: smallWindDownBound}
		h := agent.Harness{
			Provider:  provider,
			System:    "system prompt for can-006-leak",
			Turn:      agent.TurnOptions{Tools: reg},
			Scheduler: sched,
		}

		sink := make(chan *agent.Event, 256)
		resultCh := make(chan error, 1)
		go func() {
			_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
			resultCh <- err
		}()

		readUntilToolStart(t, sink, "call-can-006-leak")
		h.Interrupt()

		drainSink(t, sink)
		runErr := <-resultCh
		if !errors.Is(runErr, agent.ErrInterrupted) {
			t.Errorf("scenario Run error = %v, want errors.Is(err, agent.ErrInterrupted)", runErr)
		}

		// AFTER the run returns: the detached goroutine is accounted
		// for (proven alive past the bound, now proven to exit), not
		// merely excluded from the leak count.
		close(release)
	}

	agenttest.RequireNoGoroutineLeak(t, scenario)
}
