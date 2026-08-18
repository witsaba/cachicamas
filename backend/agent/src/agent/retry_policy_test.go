// AG-15 — strict-TDD tests for the retry policy and the failover seam
// (R-RTY-001..012, S-RTY-001..015 + bites S-RTY-010, S-RTY-011,
// S-RTY-012).
//
// Every scenario is in package agent_test (NFR-RTY-001): the
// external-test posture proves every behavioral claim from outside
// the package. The one exception is the predicate's own directly-
// callable table-driven unit test (S-RTY-001), which NFR-RTY-001
// itself permits to exercise the predicate "through whatever surface
// sdd-design fixed" — design AD-2 fixes `retryDecision` and
// `retryVerdict` as unexported, so that one test lives in a separate,
// package-internal file (retry_decision_internal_test.go), not here.
// Every OTHER behavioral claim this spec makes is independently
// provable through this file's tests, driving only the exported
// surface (Turn, Harness).
//
// # Phase 0 (D1) — the pre-stream bracket gap
//
// D1 closes the gap Turn's three pre-stream failure paths left open
// since AG-07: buildLoopRequest (loop.go:304-308), the pre-request
// hook (loop.go:317-328), and provider.Stream (loop.go:332-338) each
// left the turn bracket open on failure, so CheckStream rejected any
// second turn_start on the same lane (stream_check.go:141-143) — the
// exact shape a harness retry over a pre-output failure would
// produce. Without D1, no AG-15.1 retry scenario can pass
// (agent-loop-skeleton's R-LSK-001 delta, S-LSK-021).
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// eventKindsOf renders events' kinds in order, for a failed
// assertion's diagnostic message.
func eventKindsOf(events []agent.Event) []agent.EventKind {
	kinds := make([]agent.EventKind, len(events))
	for i, e := range events {
		kinds[i] = e.Kind()
	}
	return kinds
}

// assertPreStreamAbortSequence asserts the AG-15 D1 event sequence
// for one of Turn's three pre-stream failure paths (S-LSK-021):
// run_start (nil-continuation path only), turn_start,
// turn_end(Aborted) carrying a non-nil *Failure, and (nil-
// continuation path only) run_end(Failed) carrying a non-nil
// *Failure — in that order, and nothing else on the stream.
func assertPreStreamAbortSequence(t *testing.T, got []agent.Event, nilContinuation bool) {
	t.Helper()

	wantKinds := []agent.EventKind{agent.EventKindTurnStart, agent.EventKindTurnEnd}
	if nilContinuation {
		wantKinds = []agent.EventKind{
			agent.EventKindRunStart,
			agent.EventKindTurnStart,
			agent.EventKindTurnEnd,
			agent.EventKindRunEnd,
		}
	}

	if len(got) != len(wantKinds) {
		t.Fatalf("event sequence = %v, want kinds %v (nilContinuation=%v)", eventKindsOf(got), wantKinds, nilContinuation)
	}
	for i, wantKind := range wantKinds {
		if got[i].Kind() != wantKind {
			t.Fatalf("event[%d].Kind() = %v, want %v (full sequence: %v)", i, got[i].Kind(), wantKind, eventKindsOf(got))
		}
	}

	turnEndIdx := 1
	if nilContinuation {
		turnEndIdx = 2
	}
	turnEnd, ok := got[turnEndIdx].TurnEnd()
	if !ok {
		t.Fatalf("event[%d] does not carry a TurnEnd payload", turnEndIdx)
	}
	if turnEnd.Outcome() != agent.TurnOutcomeAborted {
		t.Errorf("turn_end outcome = %v, want %v", turnEnd.Outcome(), agent.TurnOutcomeAborted)
	}
	if failure, hasFailure := turnEnd.Failure(); !hasFailure || failure == nil {
		t.Error("turn_end carries no *Failure, want a non-nil one (AG-15 D1, R-LSK-001)")
	}

	if !nilContinuation {
		return
	}

	runEnd, ok := got[3].RunEnd()
	if !ok {
		t.Fatal("event[3] does not carry a RunEnd payload")
	}
	if runEnd.Outcome() != agent.RunOutcomeFailed {
		t.Errorf("run_end outcome = %v, want %v", runEnd.Outcome(), agent.RunOutcomeFailed)
	}
	if failure, hasFailure := runEnd.Failure(); !hasFailure || failure == nil {
		t.Error("run_end carries no *Failure, want a non-nil one (AG-15 D1, R-LSK-001)")
	}
}

// fullTurnContinuation builds a fully-configured TurnContinuation —
// the harness_test.go:38-45 fixture shape, reused here so a
// continuation-path pre-stream failure can be driven directly
// through Turn without a Harness.
func fullTurnContinuation(run agent.RunID) *agent.TurnContinuation {
	return &agent.TurnContinuation{
		Run:       run,
		Stamper:   &agent.LaneStamper{},
		Scheduler: &agent.Scheduler{},
		History:   agent.NewHistory(),
	}
}

// TestTurn_PreStreamBuildErrorEmitsTurnEnd — AG-15 D1, first of three
// pre-stream paths (loop.go:304-308). An empty system string makes
// buildLoopRequest fail before any I/O (ai.NewSystemText rejects it,
// system_instruction.go:37) — the "None" cell of design.md's
// existing-test enumeration table: no test today drives this path.
func TestTurn_PreStreamBuildErrorEmitsTurnEnd(t *testing.T) {
	t.Parallel()

	t.Run("nil continuation", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider() // never consulted: build fails first
		sink := make(chan *agent.Event, 16)

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"", // empty system: buildLoopRequest fails (R-LSK-001)
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the request-build error")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, true)
	})

	t.Run("continuation", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider()
		sink := make(chan *agent.Event, 16)
		cont := fullTurnContinuation("run-rty-d1-build")

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{Continuation: cont},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the request-build error")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, false)
	})
}

// TestTurn_PreStreamHookErrorEmitsTurnEnd — AG-15 D1, second of three
// pre-stream paths (loop.go:317-328). Reuses hookBoomAlwaysErrors
// (loop_hook_test.go:107), the S-PRH-003 fixture already proving the
// hook aborts before I/O; this test additionally proves the bracket
// now closes.
func TestTurn_PreStreamHookErrorEmitsTurnEnd(t *testing.T) {
	t.Parallel()

	t.Run("nil continuation", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sink := make(chan *agent.Event, 16)

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for D1 hook error",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{PreRequestHook: hookBoomAlwaysErrors()},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the hook's typed pre-stream failure")
		}
		if len(provider.Requests()) != 0 {
			t.Errorf("provider captured %d request(s), want 0 (hook failure aborts BEFORE provider.Stream)", len(provider.Requests()))
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, true)
	})

	t.Run("continuation", func(t *testing.T) {
		t.Parallel()

		provider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
		sink := make(chan *agent.Event, 16)
		cont := fullTurnContinuation("run-rty-d1-hook")

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for D1 hook error continuation",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{PreRequestHook: hookBoomAlwaysErrors(), Continuation: cont},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the hook's typed pre-stream failure")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, false)
	})
}

// TestTurn_PreStreamProviderErrorEmitsTurnEnd — AG-15 D1, third of
// three pre-stream paths (loop.go:332-338). Reuses errorProvider
// (loop_test.go:1408-1421), the S-LSK-011-adjacent fixture
// TestTurn_ProviderPreStreamFailureSurfacesOnReturn already drives
// (loop_test.go:1436-1461, which stays green byte-unchanged per
// design's enumeration table); this test additionally proves the
// bracket now closes.
func TestTurn_PreStreamProviderErrorEmitsTurnEnd(t *testing.T) {
	t.Parallel()

	t.Run("nil continuation", func(t *testing.T) {
		t.Parallel()

		provider := errorProvider{}
		sink := make(chan *agent.Event, 16)

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for D1 provider error",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the provider's typed pre-stream failure")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, true)
	})

	t.Run("continuation", func(t *testing.T) {
		t.Parallel()

		provider := errorProvider{}
		sink := make(chan *agent.Event, 16)
		cont := fullTurnContinuation("run-rty-d1-provider")

		_, _, err := agent.Turn(
			contextBackground(),
			provider,
			"system prompt for D1 provider error continuation",
			[]ai.Message{firstMessage(t)},
			agent.TurnOptions{Continuation: cont},
			sink,
		)
		if err == nil {
			t.Fatal("Turn returned err = nil, want the provider's typed pre-stream failure")
		}

		got := drainSink(t, sink)
		assertPreStreamAbortSequence(t, got, false)
	})
}
