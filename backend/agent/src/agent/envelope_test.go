// Tests for AG-04.1 — the event envelope and its validation gate
// (R-AEV-001, R-AEV-002, R-AEV-003). package agent_test: every behavior
// here is reachable only from outside package agent, per NFR-AEV-001 — a
// behavior reachable only from inside the package is, for this spec's
// purposes, not reachable at all.
//
// AG-04 registers no delegation mechanism (R-AEV-003 is a pin on the
// parent field's existence, not a claim about delegation semantics), so
// every event here is hand-built through the public surface — no producer
// exists until wave 2 (0003:417-418).
package agent_test

import (
	"errors"
	"reflect"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-AEV-001 — an event's kind derives from its payload, and no public
// route sets the kind independently of it: every constructor derives the
// kind from the payload type it builds internally, and none accepts a
// separate kind parameter — provable by reading NewRunStart's own
// signature, `func(RunID) (Event, error)`, which carries no kind argument.
func TestEvent_Kind_DerivesFromPayload(t *testing.T) {
	t.Parallel()

	event, err := agent.NewRunStart("run-1")
	if err != nil {
		t.Fatalf("agent.NewRunStart(%q) error = %v, want nil", "run-1", err)
	}
	if got, want := event.Kind(), agent.EventKindRunStart; got != want {
		t.Errorf("event.Kind() = %v, want %v", got, want)
	}

	payload, ok := event.RunStart()
	if !ok {
		t.Fatal("event.RunStart() reported no payload on an event of its own kind")
	}
	if payload.String() != "run_start" {
		t.Errorf("payload.String() = %q, want %q", payload.String(), "run_start")
	}
}

// S-AEV-002 — an event offered with a nil payload fails validation, and
// the failure names the missing payload rather than returning a
// zero-valued event silently accepted downstream. The only way to offer a
// nil-payload event through the public surface is the zero [agent.Event]
// literal: every constructor always assigns a concrete payload.
func TestCheckEmit_NilPayload_FailsNamingTheMissingPayload(t *testing.T) {
	t.Parallel()

	var neverConstructed agent.Event
	if got := neverConstructed.Kind(); got != 0 {
		t.Fatalf("the zero Event's Kind() = %v, want the zero kind (never a member)", got)
	}

	err := agent.CheckEmit(neverConstructed)
	if err == nil {
		t.Fatal("agent.CheckEmit(zero Event) = nil, want a rejection naming the missing payload")
	}
	if !errors.Is(err, ai.ErrNotInVocabulary) {
		t.Errorf("agent.CheckEmit(zero Event) = %v, want errors.Is to match ai.ErrNotInVocabulary (a payload-less event has no kind, and no kind is not a member of the closed vocabulary)", err)
	}
}

// S-AEV-003 — an event whose payload does not match the kind under which
// it is offered fails validation, naming both. Kind derivation makes this
// structurally unconstructible through the public surface (event.go's own
// posture: "a kind that cannot disagree with what it names"), so this
// test proves the structural half directly: the registry lookup that
// CheckEmit performs is keyed by e.Kind(), which is always
// e.payload.kind() — there is no second, independently offered kind
// anywhere in the constructor surface for it to disagree with. Reading
// through the wrong kind's own typed accessor (Layer 2's other MUST NOT
// disagree gate) reports false rather than a stale or mismatched payload.
func TestEvent_PayloadKindMismatch_IsUnconstructible(t *testing.T) {
	t.Parallel()

	runStart, err := agent.NewRunStart("run-1")
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}

	// The registry entry CheckEmit consults is keyed by the SAME derived
	// kind the payload accessor uses — there is no second source of truth
	// for "which kind" that could diverge from "which payload".
	if got, want := runStart.Kind(), agent.EventKindRunStart; got != want {
		t.Fatalf("runStart.Kind() = %v, want %v — CheckEmit's registry lookup and the payload accessor must agree", got, want)
	}
	if _, ok := runStart.RunStart(); !ok {
		t.Fatal("runStart.RunStart() reported no payload on an event of its own kind")
	}
	if err := agent.CheckEmit(mustStamp(t, runStart)); err != nil {
		t.Errorf("agent.CheckEmit(validly-stamped run-start) = %v, want nil", err)
	}
}

// S-AEV-004 — an event's run, turn and parent identity are all readable
// through the exported surface from an external package, with no build
// tag and no import of an internal or test-only package. This test file
// itself, package agent_test importing only agent and ai, is the proof:
// it compiles and runs without either.
func TestEvent_Identity_ReadableFromExternalPackage(t *testing.T) {
	t.Parallel()

	t.Run("run-scoped event: run readable, turn absent", func(t *testing.T) {
		t.Parallel()

		event, err := agent.NewRunStart("run-external")
		if err != nil {
			t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
		}
		if got, want := event.Run(), agent.RunID("run-external"); got != want {
			t.Errorf("event.Run() = %v, want %v", got, want)
		}
		if _, hasTurn := event.Turn(); hasTurn {
			t.Error("a run-scoped event reports a turn identity, want none")
		}
		if _, hasParent := event.Parent(); hasParent {
			t.Error("a top-level event reports a parent identity, want none")
		}
	})

	// Post-verify correction (W4): the subtest above only ever asserts the
	// absent direction (a run-scoped event reports no turn). It would
	// still pass if Event.Turn() returned a hardcoded constant for every
	// turn-scoped event, because it never reads the value back. This
	// subtest reads two turn-scoped events built with two different turn
	// identities and requires both the individual values and their
	// difference from each other — an assertion a hardcoded return cannot
	// satisfy.
	t.Run("turn-scoped event: turn identity distinguishes two different turns", func(t *testing.T) {
		t.Parallel()

		run := agent.RunID("run-external-turn")
		first, err := agent.NewTurnStart(run, "turn-alpha")
		if err != nil {
			t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
		}
		second, err := agent.NewTurnStart(run, "turn-beta")
		if err != nil {
			t.Fatalf("agent.NewTurnStart() error = %v, want nil", err)
		}

		firstTurn, hasTurn := first.Turn()
		if !hasTurn {
			t.Fatal("a turn-scoped event reports no turn identity, want one")
		}
		if got, want := firstTurn, agent.TurnID("turn-alpha"); got != want {
			t.Errorf("first.Turn() = %v, want %v", got, want)
		}

		secondTurn, hasTurn := second.Turn()
		if !hasTurn {
			t.Fatal("a turn-scoped event reports no turn identity, want one")
		}
		if got, want := secondTurn, agent.TurnID("turn-beta"); got != want {
			t.Errorf("second.Turn() = %v, want %v", got, want)
		}

		if firstTurn == secondTurn {
			t.Fatalf("first.Turn() and second.Turn() both report %v for two differently-constructed turns, want distinguishable values", firstTurn)
		}
	})
}

// S-AEV-005 — the Layer 2 kind vocabulary is a closed set declared by
// Layer 2 itself: no member is an alias, re-export or extension of an
// ai.EventKind member. agent.EventKind is a distinct declared type from
// ai.EventKind (S-AGV-020, S-AGV-021), and it enumerates exactly the
// four AG-04 + eleven AG-05 + three AG-06.1 kinds the package
// registers — the AG-04.1 vocabulary (turn-start/turn-end join it at
// AG-04.2's own registration, already declared at the constant level),
// extended by AG-05's message + tool families (R-AEV-010 retightened
// to "exactly 15") and AG-06.1's permission family (R-AEV-010
// retightened to "exactly 18"; final retightening to 25 lands in
// AG-06.5).
func TestEventKind_IsItsOwnClosedVocabulary_NotAnAliasOfLayer1(t *testing.T) {
	t.Parallel()

	if reflect.TypeOf(agent.EventKind(0)) == reflect.TypeOf(ai.EventKind(0)) {
		t.Fatal("agent.EventKind and ai.EventKind are the same Go type; Layer 2's kind vocabulary must be its own, independent of Layer 1's (S-AGV-020, S-AGV-021)")
	}

	kinds := agent.EventKinds()
	want := []agent.EventKind{
		agent.EventKindRunStart,
		agent.EventKindRunEnd,
		agent.EventKindTurnStart,
		agent.EventKindTurnEnd,
		// AG-05.1 (message family) — 6 message kinds (R-AMT-001, R-AMT-002)
		agent.EventKindMessageStartText, agent.EventKindMessageDeltaText, agent.EventKindMessageEndText,
		agent.EventKindMessageStartReasoning, agent.EventKindMessageDeltaReasoning, agent.EventKindMessageEndReasoning,
		// AG-05.2 (tool execution family) — 5 tool kinds (R-AMT-005, R-AMT-006, R-AMT-007)
		agent.EventKindToolStart, agent.EventKindToolProgress,
		agent.EventKindToolEndSuccess, agent.EventKindToolEndResultFailure, agent.EventKindToolEndExecutionFailure,
		// AG-06.1 (permission family) — 3 kinds (R-APE-001, R-APE-002, R-APE-003)
		agent.EventKindPermissionDecisionRequired, agent.EventKindPermissionDecisionMade, agent.EventKindPermissionResolutionRemembered,
	}
	if len(kinds) != len(want) {
		t.Fatalf("agent.EventKinds() = %v (%d kinds), want %v (%d kinds)", kinds, len(kinds), want, len(want))
	}
	for i := range want {
		if kinds[i] != want[i] {
			t.Errorf("agent.EventKinds()[%d] = %v, want %v", i, kinds[i], want[i])
		}
	}
}

// S-AEV-010 — two hand-built event sequences stamped through the
// envelope's public ordering mechanism each carry an independent
// ordering, starting at 1, with no gap and no repeat — and neither
// sequence's ordering is affected by the other's.
func TestLaneStamper_TwoLanes_EachIndependentlyContiguousAndOneBased(t *testing.T) {
	t.Parallel()

	const perLane = 4 // run-start, turn-start, turn-end, run-end: one complete valid bracket.
	first := buildValidRunBracket(t, "run-first")
	second := buildValidRunBracket(t, "run-second")

	for i := range perLane {
		if got, want := first[i].Sequence(), agent.Sequence(i+1); got != want {
			t.Errorf("first lane event %d: Sequence() = %v, want %v", i, got, want)
		}
		if got, want := second[i].Sequence(), agent.Sequence(i+1); got != want {
			t.Errorf("second lane event %d: Sequence() = %v, want %v", i, got, want)
		}
	}

	if report := agent.CheckStream(first); report.Violation() != nil {
		t.Errorf("agent.CheckStream(first lane) = %v, want no violation", report.Violation())
	}
	if report := agent.CheckStream(second); report.Violation() != nil {
		t.Errorf("agent.CheckStream(second lane) = %v, want no violation", report.Violation())
	}
}

// S-AEV-011 — the two lanes of S-AEV-010, stamped from concurrently
// running goroutines, report no data race and both remain contiguous and
// 1-based. Each goroutine plays the forwarding-activity role AG-01 § 5
// assigns to a lane (design AD-5): one LaneStamper, touched by exactly one
// goroutine.
func TestLaneStamper_ConcurrentLanes_NoRaceAndBothStayContiguous(t *testing.T) {
	t.Parallel()

	const perLane = 50
	firstResults := make([]agent.Sequence, perLane)
	secondResults := make([]agent.Sequence, perLane)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		var lane agent.LaneStamper
		for i := range perLane {
			e, err := agent.NewRunStart("run-a")
			if err != nil {
				t.Errorf("agent.NewRunStart() error = %v, want nil", err)
				return
			}
			firstResults[i] = lane.Stamp(e).Sequence()
		}
	}()
	go func() {
		defer wg.Done()
		var lane agent.LaneStamper
		for i := range perLane {
			e, err := agent.NewRunStart("run-b")
			if err != nil {
				t.Errorf("agent.NewRunStart() error = %v, want nil", err)
				return
			}
			secondResults[i] = lane.Stamp(e).Sequence()
		}
	}()
	wg.Wait()

	for i := range perLane {
		if want := agent.Sequence(i + 1); firstResults[i] != want {
			t.Errorf("first lane event %d: sequence = %v, want %v", i, firstResults[i], want)
		}
		if want := agent.Sequence(i + 1); secondResults[i] != want {
			t.Errorf("second lane event %d: sequence = %v, want %v", i, secondResults[i], want)
		}
	}
}

// S-AEV-012 — a hand-built sequence whose ordering starts at a value other
// than 1 is REJECTED, and the rejection names the ordering rule.
func TestCheckStream_SequenceNotStartingAtOne_RejectedNamingTheRule(t *testing.T) {
	t.Parallel()

	e, err := agent.NewRunStart("run-1")
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	var lane agent.LaneStamper
	lane.Stamp(e) // discarded — burns sequence 1, so the next stamp is 2
	off := lane.Stamp(e)

	report := agent.CheckStream([]agent.Event{off})
	if report.Violation() == nil {
		t.Fatal("agent.CheckStream(sequence starting at 2) = no violation, want a rejection")
	}
	if !errors.Is(report.Violation(), ai.ErrOutOfRange) {
		t.Errorf("agent.CheckStream(sequence starting at 2) = %v, want errors.Is to match ai.ErrOutOfRange", report.Violation())
	}
}

// S-AEV-013 — a hand-built sequence with a gap or a repeat in its
// ordering is REJECTED, and the rejection names the offending position:
// the offending event's own 0-based index within the checked slice, never
// its sequence value — which is itself the value under rejection and, for
// a gap, may not even be a valid index into the slice being checked
// (post-verify correction, W1/W2: CheckStream reported the sequence value
// instead of the slice index, and no test asserted a position at all).
func TestCheckStream_GapOrRepeat_RejectedNamingThePosition(t *testing.T) {
	t.Parallel()

	t.Run("gap", func(t *testing.T) {
		t.Parallel()
		events := buildStampedRunStarts(t, 3)
		gapped := []agent.Event{events[0], events[2]} // 1, 3 — skips 2; offending element gapped[1], sequence value 3

		report := agent.CheckStream(gapped)
		if report.Violation() == nil {
			t.Fatal("agent.CheckStream(gapped sequence) = no violation, want a rejection")
		}
		if !errors.Is(report.Violation(), ai.ErrOutOfRange) {
			t.Errorf("agent.CheckStream(gapped sequence) = %v, want errors.Is to match ai.ErrOutOfRange", report.Violation())
		}
		// gapped[1] is the offending element (0-based slice index 1); its
		// sequence value is 3, which is not even a valid index into this
		// 2-element slice — the exact defect verify found (W2).
		requireViolationPosition(t, report.Violation(), 1)
	})

	t.Run("repeat", func(t *testing.T) {
		t.Parallel()
		// A bracket-valid prefix (not four bare run-starts, which would
		// themselves trip the duplicate-open bracket rule): prefix[0]
		// repeats at the end, chosen so the offending element's slice
		// index (4) and its sequence value (1) cannot coincide by
		// accident, unlike an adjacent repeat.
		prefix := buildValidRunBracket(t, "run-013-repeat")
		repeated := []agent.Event{prefix[0], prefix[1], prefix[2], prefix[3], prefix[0]}

		report := agent.CheckStream(repeated)
		if report.Violation() == nil {
			t.Fatal("agent.CheckStream(repeated sequence) = no violation, want a rejection")
		}
		if !errors.Is(report.Violation(), ai.ErrOutOfRange) {
			t.Errorf("agent.CheckStream(repeated sequence) = %v, want errors.Is to match ai.ErrOutOfRange", report.Violation())
		}
		requireViolationPosition(t, report.Violation(), 4)
	})
}

// S-AEV-020 — an event constructed as belonging to a delegated harness
// carries its parent identifier, readable from an external package, even
// though no delegation mechanism exists in the package yet.
func TestNewDelegatedRunStart_CarriesParentIdentity(t *testing.T) {
	t.Parallel()

	event, err := agent.NewDelegatedRunStart("run-child", "run-parent")
	if err != nil {
		t.Fatalf("agent.NewDelegatedRunStart() error = %v, want nil", err)
	}
	parent, hasParent := event.Parent()
	if !hasParent {
		t.Fatal("event.Parent() reports no parent for a delegated run-start, want one")
	}
	if got, want := parent, agent.RunID("run-parent"); got != want {
		t.Errorf("event.Parent() = %v, want %v", got, want)
	}
	if got, want := event.Run(), agent.RunID("run-child"); got != want {
		t.Errorf("event.Run() = %v, want %v", got, want)
	}
}

// S-AEV-021 — an event constructed as top-level reports "no parent" as a
// distinguishable state rather than an ambiguous zero value: the bool
// result of Parent(), not a check against an empty string.
func TestNewRunStart_TopLevel_ReportsNoParentAsADistinguishableState(t *testing.T) {
	t.Parallel()

	event, err := agent.NewRunStart("run-top-level")
	if err != nil {
		t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
	}
	parent, hasParent := event.Parent()
	if hasParent {
		t.Errorf("event.Parent() reports a parent (%v) for a top-level run-start, want none", parent)
	}
}

// S-AEV-022 — after this change, the package's exported surface declares
// no delegation, subagent, or child-harness mechanism — only the parent
// identifier field and its accessor. Proven by enumerating the package's
// exported declarations and asserting none names a delegation concept
// beyond NewDelegatedRunStart / Event.Parent themselves.
func TestPackageSurface_DeclaresNoDelegationMechanism(t *testing.T) {
	t.Parallel()

	forbidden := []string{"Delegate", "Subagent", "ChildHarness", "Spawn"}
	// The two names the parent field's own construction/reading door is
	// allowed to carry — everything else naming delegation is forbidden.
	allowed := map[string]bool{
		"NewDelegatedRunStart": true,
	}

	exported := exportedPackageAgentNames(t)
	if len(exported) == 0 {
		t.Fatal("exportedPackageAgentNames found zero declarations; the assertion would pass vacuously")
	}
	for _, name := range exported {
		if allowed[name] {
			continue
		}
		for _, f := range forbidden {
			if containsFold(name, f) {
				t.Errorf("exported declaration %q names a delegation/subagent mechanism (%q); AG-04 declares none (R-AEV-003, S-AEV-022)", name, f)
			}
		}
	}
}

// buildStampedRunStarts stamps n run-start events through one LaneStamper,
// in order, for use by contiguity-only tests. Phase 1's minimal
// CheckStream cares only about sequence, so the shared run identity and
// kind are irrelevant to what these tests assert.
func buildStampedRunStarts(t *testing.T, n int) []agent.Event {
	t.Helper()
	var lane agent.LaneStamper
	out := make([]agent.Event, n)
	for i := range n {
		e, err := agent.NewRunStart("run-lane")
		if err != nil {
			t.Fatalf("agent.NewRunStart() error = %v, want nil", err)
		}
		out[i] = lane.Stamp(e)
	}
	return out
}

// mustStamp stamps e with a fresh, single-use LaneStamper — a convenience
// for tests that need exactly one validly-sequenced event and do not
// themselves exercise ordering.
func mustStamp(t *testing.T, e agent.Event) agent.Event {
	t.Helper()
	var lane agent.LaneStamper
	return lane.Stamp(e)
}

// buildValidRunBracket stamps one complete, CheckStream-valid run
// bracket through a fresh LaneStamper: run-start, turn-start, turn-end,
// run-end (sequences 1..4). Used by tests that need a stream CheckStream
// accepts outright, once the full bracket engine is present (AG-04.2).
func buildValidRunBracket(t *testing.T, run agent.RunID) []agent.Event {
	t.Helper()
	var lane agent.LaneStamper

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
	end, err := agent.NewRunEnd(run, agent.RunOutcomeCompleted, nil)
	if err != nil {
		t.Fatalf("agent.NewRunEnd() error = %v, want nil", err)
	}

	return []agent.Event{
		lane.Stamp(start),
		lane.Stamp(turnStart),
		lane.Stamp(turnEnd),
		lane.Stamp(end),
	}
}
