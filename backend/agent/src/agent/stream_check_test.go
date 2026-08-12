// Tests for AG-04.2 — the run and turn lifecycle families and the
// stream-contract validator's two-level scope engine (R-AEV-004,
// R-AEV-005, R-AEV-006). package agent_test: NFR-AEV-001.
package agent_test

import (
	"errors"
	"go/ast"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-AEV-030 — one run-start preceding all other events and one run-end
// following them is ACCEPTED.
func TestCheckStream_SingleValidRunBracket_Accepted(t *testing.T) {
	t.Parallel()

	events := buildValidRunBracket(t, "run-030")
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(valid bracket) = %v, want no violation", report.Violation())
	}
}

// S-AEV-031 — two run-start events is REJECTED, naming the duplicate
// bracket opening.
func TestCheckStream_TwoRunStarts_RejectedNamingDuplicate(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	first, err := agent.NewRunStart("run-031")
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	second, err := agent.NewRunStart("run-031b")
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	events := []agent.Event{lane.Stamp(first), lane.Stamp(second)}

	report := agent.CheckStream(events)
	if report.Violation() == nil {
		t.Fatal("agent.CheckStream(two run-starts) = no violation, want a rejection")
	}
	if !errors.Is(report.Violation(), ai.ErrDuplicate) {
		t.Errorf("agent.CheckStream(two run-starts) = %v, want errors.Is to match ai.ErrDuplicate", report.Violation())
	}
}

// S-AEV-032 — a sequence with no run-end is REJECTED, naming the unclosed
// run bracket.
func TestCheckStream_NoRunEnd_RejectedNamingUnclosedBracket(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	start, err := agent.NewRunStart("run-032")
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	events := []agent.Event{lane.Stamp(start)}

	report := agent.CheckStream(events)
	if report.Violation() == nil {
		t.Fatal("agent.CheckStream(no run-end) = no violation, want a rejection")
	}
	if !errors.Is(report.Violation(), ai.ErrMalformed) {
		t.Errorf("agent.CheckStream(no run-end) = %v, want errors.Is to match ai.ErrMalformed", report.Violation())
	}
}

// S-AEV-033 — any event after run-end is REJECTED, naming the event that
// follows the terminal.
func TestCheckStream_EventAfterRunEnd_RejectedNamingTheFollowingEvent(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	start, err := agent.NewRunStart("run-033")
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	end, err := agent.NewRunEnd("run-033", agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd() error = %v, want nil", err)
	}
	trailing, err := agent.NewRunStart("run-033-trailing")
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	events := []agent.Event{lane.Stamp(start), lane.Stamp(end), lane.Stamp(trailing)}

	report := agent.CheckStream(events)
	if report.Violation() == nil {
		t.Fatal("agent.CheckStream(event after run-end) = no violation, want a rejection")
	}
	if !errors.Is(report.Violation(), ai.ErrMisplaced) {
		t.Errorf("agent.CheckStream(event after run-end) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
	}
}

// S-AEV-034 — a sequence whose first event is not run-start is REJECTED,
// naming the event that preceded the bracket.
func TestCheckStream_FirstEventNotRunStart_RejectedNamingThePreceder(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	end, err := agent.NewRunEnd("run-034", agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd() error = %v, want nil", err)
	}
	events := []agent.Event{lane.Stamp(end)}

	report := agent.CheckStream(events)
	if report.Violation() == nil {
		t.Fatal("agent.CheckStream(first event not run-start) = no violation, want a rejection")
	}
	if !errors.Is(report.Violation(), ai.ErrMisplaced) {
		t.Errorf("agent.CheckStream(first event not run-start) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
	}
}

// S-AEV-035 — run-end events constructed with each of completed,
// interrupted and failed are reachable as typed values distinguishing the
// three, and no public route constructs a run-end without an outcome.
func TestNewRunEnd_TypedOutcome_DistinguishableAndRequired(t *testing.T) {
	t.Parallel()

	t.Run("completed", func(t *testing.T) {
		t.Parallel()
		e, err := agent.NewRunEnd("run-035a", agent.RunOutcomeCompleted, nil)
		if err != nil {
			t.Fatalf("agent.NewRunEnd(Completed) error = %v, want nil", err)
		}
		payload, ok := e.RunEnd()
		if !ok {
			t.Fatal("e.RunEnd() reported no payload on an event of its own kind")
		}
		if got := payload.Outcome(); got != agent.RunOutcomeCompleted {
			t.Errorf("payload.Outcome() = %v, want RunOutcomeCompleted", got)
		}
	})

	t.Run("interrupted", func(t *testing.T) {
		t.Parallel()
		e, err := agent.NewRunEnd("run-035b", agent.RunOutcomeInterrupted, nil)
		if err != nil {
			t.Fatalf("agent.NewRunEnd(Interrupted) error = %v, want nil", err)
		}
		payload, _ := e.RunEnd()
		if got := payload.Outcome(); got != agent.RunOutcomeInterrupted {
			t.Errorf("payload.Outcome() = %v, want RunOutcomeInterrupted", got)
		}
	})

	t.Run("failed", func(t *testing.T) {
		t.Parallel()
		failure := mustAgentFailure(t)
		e, err := agent.NewRunEnd("run-035c", agent.RunOutcomeFailed, failure)
		if err != nil {
			t.Fatalf("agent.NewRunEnd(Failed) error = %v, want nil", err)
		}
		payload, _ := e.RunEnd()
		if got := payload.Outcome(); got != agent.RunOutcomeFailed {
			t.Errorf("payload.Outcome() = %v, want RunOutcomeFailed", got)
		}
		if _, hasFailure := payload.Failure(); !hasFailure {
			t.Error("payload.Failure() reports none for a RunOutcomeFailed run-end, want one")
		}
	})

	t.Run("no route without an outcome: the zero value is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := agent.NewRunEnd("run-035d", 0, nil)
		if err == nil {
			t.Fatal("agent.NewRunEnd(zero outcome) = nil error, want a rejection")
		}
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("agent.NewRunEnd(zero outcome) = %v, want errors.Is to match ai.ErrNotInVocabulary", err)
		}
	})
}

// S-AEV-040 — one or more non-overlapping turn-start/turn-end pairs
// entirely inside the run bracket is ACCEPTED.
func TestCheckStream_NonOverlappingTurnPairs_Accepted(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	run := agent.RunID("run-040")
	start, err := agent.NewRunStart(run)
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	turn1Start, err := agent.NewTurnStart(run, "turn-1")
	if err != nil {
		t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
	}
	turn1End, err := agent.NewTurnEnd(run, "turn-1", agent.TurnOutcomeFinished, nil)
	if err != nil {
		t.Fatalf("agent.NewTurnEnd() error = %v, want nil", err)
	}
	turn2Start, err := agent.NewTurnStart(run, "turn-2")
	if err != nil {
		t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
	}
	turn2End, err := agent.NewTurnEnd(run, "turn-2", agent.TurnOutcomeFinished, nil)
	if err != nil {
		t.Fatalf("agent.NewTurnEnd() error = %v, want nil", err)
	}
	end, err := agent.NewRunEnd(run, agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd() error = %v, want nil", err)
	}

	events := []agent.Event{
		lane.Stamp(start), lane.Stamp(turn1Start), lane.Stamp(turn1End),
		lane.Stamp(turn2Start), lane.Stamp(turn2End), lane.Stamp(end),
	}
	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Errorf("agent.CheckStream(two non-overlapping turn pairs) = %v, want no violation", report.Violation())
	}
}

// S-AEV-041 — a second turn-start before the preceding turn-end is
// REJECTED, naming the overlap.
func TestCheckStream_OverlappingTurnStart_RejectedNamingOverlap(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	run := agent.RunID("run-041")
	start, err := agent.NewRunStart(run)
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	turn1Start, err := agent.NewTurnStart(run, "turn-1")
	if err != nil {
		t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
	}
	turn2Start, err := agent.NewTurnStart(run, "turn-2")
	if err != nil {
		t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
	}
	events := []agent.Event{lane.Stamp(start), lane.Stamp(turn1Start), lane.Stamp(turn2Start)}

	report := agent.CheckStream(events)
	if report.Violation() == nil {
		t.Fatal("agent.CheckStream(overlapping turn-start) = no violation, want a rejection")
	}
	if !errors.Is(report.Violation(), ai.ErrMisplaced) {
		t.Errorf("agent.CheckStream(overlapping turn-start) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
	}
}

// S-AEV-042 — a turn-start or turn-end outside the run bracket (before
// run-start or after run-end) is REJECTED, naming the escaped bracket.
func TestCheckStream_TurnOutsideRunBracket_RejectedNamingEscapedBracket(t *testing.T) {
	t.Parallel()

	t.Run("turn-start before run-start", func(t *testing.T) {
		t.Parallel()
		var lane agent.LaneStamper
		turnStart, err := agent.NewTurnStart("run-042a", "turn-1")
		if err != nil {
			t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
		}
		events := []agent.Event{lane.Stamp(turnStart)}

		report := agent.CheckStream(events)
		if report.Violation() == nil {
			t.Fatal("agent.CheckStream(turn-start before run-start) = no violation, want a rejection")
		}
		if !errors.Is(report.Violation(), ai.ErrMisplaced) {
			t.Errorf("agent.CheckStream(turn-start before run-start) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
		}
	})

	t.Run("turn-start after run-end", func(t *testing.T) {
		t.Parallel()
		var lane agent.LaneStamper
		run := agent.RunID("run-042b")
		start, err := agent.NewRunStart(run)
		if err != nil {
			t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
		}
		end, err := agent.NewRunEnd(run, agent.RunOutcomeCompleted, nil)
		if err != nil {
			t.Fatalf("agent.NewRunEnd() error = %v, want nil", err)
		}
		turnStart, err := agent.NewTurnStart(run, "turn-1")
		if err != nil {
			t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
		}
		events := []agent.Event{lane.Stamp(start), lane.Stamp(end), lane.Stamp(turnStart)}

		report := agent.CheckStream(events)
		if report.Violation() == nil {
			t.Fatal("agent.CheckStream(turn-start after run-end) = no violation, want a rejection")
		}
		if !errors.Is(report.Violation(), ai.ErrMisplaced) {
			t.Errorf("agent.CheckStream(turn-start after run-end) = %v, want errors.Is to match ai.ErrMisplaced", report.Violation())
		}
	})
}

// S-AEV-043 — a turn-start never closed by a turn-end before run-end is
// REJECTED, naming the unclosed turn.
func TestCheckStream_UnclosedTurnBeforeRunEnd_RejectedNamingUnclosedTurn(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	run := agent.RunID("run-043")
	start, err := agent.NewRunStart(run)
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	turnStart, err := agent.NewTurnStart(run, "turn-1")
	if err != nil {
		t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
	}
	end, err := agent.NewRunEnd(run, agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd() error = %v, want nil", err)
	}
	events := []agent.Event{lane.Stamp(start), lane.Stamp(turnStart), lane.Stamp(end)}

	report := agent.CheckStream(events)
	if report.Violation() == nil {
		t.Fatal("agent.CheckStream(unclosed turn before run-end) = no violation, want a rejection")
	}
	if !errors.Is(report.Violation(), ai.ErrMalformed) {
		t.Errorf("agent.CheckStream(unclosed turn before run-end) = %v, want errors.Is to match ai.ErrMalformed", report.Violation())
	}
}

// S-AEV-044 — turn-end events constructed with each outcome are
// distinguishable as typed values, and no public route constructs a
// turn-end without an outcome.
func TestNewTurnEnd_TypedOutcome_DistinguishableAndRequired(t *testing.T) {
	t.Parallel()

	t.Run("finished", func(t *testing.T) {
		t.Parallel()
		e, err := agent.NewTurnEnd("run-044a", "turn-1", agent.TurnOutcomeFinished, nil)
		if err != nil {
			t.Fatalf("agent.NewTurnEnd(Finished) error = %v, want nil", err)
		}
		payload, ok := e.TurnEnd()
		if !ok {
			t.Fatal("e.TurnEnd() reported no payload on an event of its own kind")
		}
		if got := payload.Outcome(); got != agent.TurnOutcomeFinished {
			t.Errorf("payload.Outcome() = %v, want TurnOutcomeFinished", got)
		}
	})

	t.Run("aborted", func(t *testing.T) {
		t.Parallel()
		failure := mustAgentFailure(t)
		e, err := agent.NewTurnEnd("run-044b", "turn-1", agent.TurnOutcomeAborted, failure)
		if err != nil {
			t.Fatalf("agent.NewTurnEnd(Aborted) error = %v, want nil", err)
		}
		payload, _ := e.TurnEnd()
		if got := payload.Outcome(); got != agent.TurnOutcomeAborted {
			t.Errorf("payload.Outcome() = %v, want TurnOutcomeAborted", got)
		}
	})

	t.Run("no route without an outcome: the zero value is rejected", func(t *testing.T) {
		t.Parallel()
		_, err := agent.NewTurnEnd("run-044c", "turn-1", 0, nil)
		if err == nil {
			t.Fatal("agent.NewTurnEnd(zero outcome) = nil error, want a rejection")
		}
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("agent.NewTurnEnd(zero outcome) = %v, want errors.Is to match ai.ErrNotInVocabulary", err)
		}
	})
}

// S-AEV-050 — a test in a package other than agent/agent_test calls
// CheckStream with no build tag and no test-only import. This whole
// package, agent_test, importing only agent and ai, already proves it —
// this test additionally proves the call itself compiles and runs.
func TestCheckStream_CallableFromExternalPackage_NoBuildTagNoTestOnlyImport(t *testing.T) {
	t.Parallel()

	report := agent.CheckStream(buildValidRunBracket(t, "run-050"))
	if report.Violation() != nil {
		t.Errorf("agent.CheckStream() = %v, want no violation", report.Violation())
	}
}

// S-AEV-051 — CheckStream's declaration is exported, lives in a
// non-_test.go file, and takes a finite ordered slice rather than a
// channel — proven by AST inspection of stream_check.go's own source,
// mirroring event_registry_test.go's structural-guard posture.
func TestCheckStream_Declaration_IsExportedNonTestFileFiniteSlice(t *testing.T) {
	t.Parallel()

	sig := funcDeclSignature(t, "stream_check.go", "CheckStream")
	if sig == nil {
		t.Fatal("stream_check.go declares no CheckStream function; the guard would pass vacuously")
	}
	if len(sig.Params.List) != 1 {
		t.Fatalf("CheckStream has %d parameter groups, want exactly 1 (events []Event)", len(sig.Params.List))
	}
	if _, isSlice := sig.Params.List[0].Type.(*ast.ArrayType); !isSlice {
		t.Errorf("CheckStream's parameter type = %T, want a slice type (*ast.ArrayType), never a channel", sig.Params.List[0].Type)
	}
}

// S-AEV-052 — a sequence that violates exactly one rule is reported
// naming that rule and the position of the offending event, and does not
// report unrelated rules. Proven by re-checking S-AEV-034's single
// violating sequence (first event not run-start) and confirming the
// verdict names exactly ai.ErrMisplaced, no other rule class.
func TestCheckStream_SingleRuleViolation_NamesOnlyThatRule(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	end, err := agent.NewRunEnd("run-052", agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd() error = %v, want nil", err)
	}
	report := agent.CheckStream([]agent.Event{lane.Stamp(end)})

	violation := report.Violation()
	if violation == nil {
		t.Fatal("agent.CheckStream() = no violation, want exactly one")
	}
	unrelated := []error{ai.ErrOutOfRange, ai.ErrDuplicate, ai.ErrNotInVocabulary, ai.ErrUnresolvedReference}
	for _, rule := range unrelated {
		if errors.Is(violation, rule) {
			t.Errorf("agent.CheckStream() violation = %v, unexpectedly also matches unrelated rule %v", violation, rule)
		}
	}
	if !errors.Is(violation, ai.ErrMisplaced) {
		t.Errorf("agent.CheckStream() violation = %v, want errors.Is to match ai.ErrMisplaced", violation)
	}
}

// S-AEV-053 — a fully valid hand-built run containing run-start, two
// non-overlapping turn brackets, and run-end is ACCEPTED with an empty
// violation set.
func TestCheckStream_FullyValidRunWithTwoTurnBrackets_AcceptedEmptyViolationSet(t *testing.T) {
	t.Parallel()

	var lane agent.LaneStamper
	run := agent.RunID("run-053")
	start, _ := agent.NewRunStart(run)
	turn1Start, _ := agent.NewTurnStart(run, "turn-1")
	turn1End, _ := agent.NewTurnEnd(run, "turn-1", agent.TurnOutcomeFinished, nil)
	turn2Start, _ := agent.NewTurnStart(run, "turn-2")
	turn2End, _ := agent.NewTurnEnd(run, "turn-2", agent.TurnOutcomeFinished, nil)
	end, _ := agent.NewRunEnd(run, agent.RunOutcomeCompleted, nil)

	events := []agent.Event{
		lane.Stamp(start), lane.Stamp(turn1Start), lane.Stamp(turn1End),
		lane.Stamp(turn2Start), lane.Stamp(turn2End), lane.Stamp(end),
	}
	report := agent.CheckStream(events)
	if report.Violation() != nil {
		t.Errorf("agent.CheckStream(valid run + 2 turn brackets) = %v, want an empty violation set (nil)", report.Violation())
	}
}

// S-AEV-054 — every rule of R-AEV-001–R-AEV-005 that is expressible over a
// finite sequence has a corresponding check in CheckStream, and no check
// enforces a rule this spec does not state. Proven by reading
// stream_check.go's own documented rule list — a documentation-reading
// assertion, mirroring S-AEV-091's own posture, rather than a mechanical
// re-derivation of the rule engine.
func TestCheckStream_RuleCoverage_MatchesTheDocumentedList(t *testing.T) {
	t.Parallel()

	doc := allFileComments(t, "stream_check.go")
	wantPhrases := []string{
		"contiguity",
		"nothing follows the terminal",
		"bracket transition",
		"run bracket is open",
		"PlacementTurn",
		"CardinalityAtMostOne",
	}
	for _, phrase := range wantPhrases {
		if !containsFold(doc, phrase) {
			t.Errorf("stream_check.go's documentation does not mention %q; the rule-coverage enumeration must name every rule the engine checks", phrase)
		}
	}
}

// mustAgentFailure builds a *agent.Failure wrapping a fresh mid-stream
// *ai.Failure, for tests that need a non-nil failure value and are not
// themselves testing the failure surface (that is invariant_pin_test.go's
// job, AG-04.3).
func mustAgentFailure(t *testing.T) *agent.Failure {
	t.Helper()
	inner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure() error = %v, want nil", err)
	}
	f, err := agent.NewFailure(inner)
	if err != nil {
		t.Fatalf("agent.NewFailure() error = %v, want nil", err)
	}
	return f
}
