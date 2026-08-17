// AG-14 — Phase 0: the shutdown run outcome is a stream-level
// discriminator (R-CAN-007, R-CAN-008). This file proves Decision 1's
// substrate release at runtime, before any harness wiring exists
// (design.md's forwarded obligation 4): NewRunEnd(run,
// RunOutcomeShutdown, nil) must stream through CheckStream with
// stream_check.go byte-unchanged.
package agent_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AG-14 — S-CAN-007. Given the run-outcome vocabulary after this
// change, when the shutdown member is rendered, then it reads
// "shutdown"; and when a run-close event is constructed with the
// shutdown outcome and no failure, then construction succeeds; and
// when one is constructed with the shutdown outcome and a non-nil
// failure, then it is rejected with the misplaced-failure rule class
// the validator already applies to every non-Failed outcome; and when
// CheckStream is run over a complete, otherwise-valid stream ending in
// a shutdown run-close, then it accepts the stream — proving
// stream_check.go's own bound check admits the new member with no
// edit to that file (R-CAN-007).
func TestRunOutcomeShutdown_VocabularyAndStreamAcceptance(t *testing.T) {
	t.Parallel()

	t.Run("String renders shutdown", func(t *testing.T) {
		t.Parallel()
		if got := agent.RunOutcomeShutdown.String(); got != "shutdown" {
			t.Errorf("agent.RunOutcomeShutdown.String() = %q, want %q", got, "shutdown")
		}
	})

	t.Run("construction with no failure succeeds", func(t *testing.T) {
		t.Parallel()
		e, err := agent.NewRunEnd("run-can-007a", agent.RunOutcomeShutdown, nil)
		if err != nil {
			t.Fatalf("agent.NewRunEnd(Shutdown, nil) error = %v, want nil", err)
		}
		payload, ok := e.RunEnd()
		if !ok {
			t.Fatal("e.RunEnd() reported no payload on an event of its own kind")
		}
		if got := payload.Outcome(); got != agent.RunOutcomeShutdown {
			t.Errorf("payload.Outcome() = %v, want RunOutcomeShutdown", got)
		}
		if _, hasFailure := payload.Failure(); hasFailure {
			t.Error("payload.Failure() reports one for a RunOutcomeShutdown run-end, want none")
		}
	})

	t.Run("construction with a failure is rejected misplaced", func(t *testing.T) {
		t.Parallel()
		failure := mustAgentFailure(t)
		_, err := agent.NewRunEnd("run-can-007b", agent.RunOutcomeShutdown, failure)
		if err == nil {
			t.Fatal("agent.NewRunEnd(Shutdown, non-nil failure) = nil error, want a rejection")
		}
		if !errors.Is(err, ai.ErrMisplaced) {
			t.Errorf("agent.NewRunEnd(Shutdown, failure) = %v, want errors.Is to match ai.ErrMisplaced", err)
		}
	})

	t.Run("CheckStream accepts a complete stream ending in run_end(shutdown)", func(t *testing.T) {
		t.Parallel()

		var lane agent.LaneStamper
		run := agent.RunID("run-can-007c")

		start, err := agent.NewRunStart(run)
		if err != nil {
			t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
		}
		turnStart, err := agent.NewTurnStart(run, "turn-1")
		if err != nil {
			t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
		}
		turnEnd, err := agent.NewTurnEnd(run, "turn-1", agent.TurnOutcomeFinished, nil)
		if err != nil {
			t.Fatalf("agent.NewTurnEnd() error = %v, want nil", err)
		}
		end, err := agent.NewRunEnd(run, agent.RunOutcomeShutdown, nil)
		if err != nil {
			t.Fatalf("agent.NewRunEnd(Shutdown) error = %v, want nil", err)
		}

		events := []agent.Event{
			lane.Stamp(start),
			lane.Stamp(turnStart),
			lane.Stamp(turnEnd),
			lane.Stamp(end),
		}

		if report := agent.CheckStream(events); report.Violation() != nil {
			t.Errorf("agent.CheckStream(shutdown-terminated bracket) = %v, want no violation — stream_check.go's own bound check (RunEnd.validate) must admit the new member unedited", report.Violation())
		}
	})
}

// AG-14 — R-CAN-001's own vocabulary clause. Given the package's
// cancellation vocabulary, when the two named sentinels are compared
// with errors.Is against themselves, then each matches itself and
// fails to match the other — two distinct identities, not one signal
// wearing two names; and when the post-shutdown prompt refusal is
// compared with errors.Is against the shutdown sentinel, then it
// matches — the refusal wraps shutdown rather than standing alone; and
// when a detached-call carrier is built naming a tool and a call
// identity and rendered, then its message names both.
func TestCancellationVocabulary_SentinelsDistinctRefusalWrapsCarrierNames(t *testing.T) {
	t.Parallel()

	t.Run("the two sentinels are errors.Is-matchable and mutually distinct", func(t *testing.T) {
		t.Parallel()
		if !errors.Is(agent.ErrInterrupted, agent.ErrInterrupted) {
			t.Error("errors.Is(ErrInterrupted, ErrInterrupted) = false, want true")
		}
		if !errors.Is(agent.ErrShutdown, agent.ErrShutdown) {
			t.Error("errors.Is(ErrShutdown, ErrShutdown) = false, want true")
		}
		if errors.Is(agent.ErrInterrupted, agent.ErrShutdown) {
			t.Error("errors.Is(ErrInterrupted, ErrShutdown) = true, want false — distinct signals")
		}
		if errors.Is(agent.ErrShutdown, agent.ErrInterrupted) {
			t.Error("errors.Is(ErrShutdown, ErrInterrupted) = true, want false — distinct signals")
		}
	})

	t.Run("the post-shutdown refusal wraps the shutdown sentinel, not interrupt", func(t *testing.T) {
		t.Parallel()
		if !errors.Is(agent.ErrPromptAfterShutdown, agent.ErrShutdown) {
			t.Error("errors.Is(ErrPromptAfterShutdown, ErrShutdown) = false, want true")
		}
		if errors.Is(agent.ErrPromptAfterShutdown, agent.ErrInterrupted) {
			t.Error("errors.Is(ErrPromptAfterShutdown, ErrInterrupted) = true, want false")
		}
	})

	t.Run("DetachedCallError names the tool and the call identity", func(t *testing.T) {
		t.Parallel()
		err := &agent.DetachedCallError{Tool: "read_can_vocab", CallID: "call-can-vocab-1"}
		var target *agent.DetachedCallError
		if !errors.As(error(err), &target) {
			t.Fatal("errors.As(*DetachedCallError, &target) = false, want true")
		}
		if target.Tool != "read_can_vocab" {
			t.Errorf("target.Tool = %q, want %q", target.Tool, "read_can_vocab")
		}
		if target.CallID != "call-can-vocab-1" {
			t.Errorf("target.CallID = %q, want %q", target.CallID, "call-can-vocab-1")
		}
		msg := err.Error()
		if !strings.Contains(msg, "read_can_vocab") {
			t.Errorf("Error() = %q, want it to name the tool %q", msg, "read_can_vocab")
		}
		if !strings.Contains(msg, "call-can-vocab-1") {
			t.Errorf("Error() = %q, want it to name the call identity %q", msg, "call-can-vocab-1")
		}
	})

	t.Run("a second, differently-named carrier renders a different message", func(t *testing.T) {
		t.Parallel()
		a := &agent.DetachedCallError{Tool: "tool_a", CallID: "call_a"}
		b := &agent.DetachedCallError{Tool: "tool_b", CallID: "call_b"}
		if a.Error() == b.Error() {
			t.Errorf("two differently-named carriers rendered the same message %q, want distinct messages", a.Error())
		}
	})
}
