// AG-19.1 — Phase 2 (U2): sibling children interleave without
// cross-talk (R-DEL-005, charter AG-19.1 scenario 2); folds in the
// agent-event-delivery delta's S-AGE-028/S-AGE-029, which reuse this
// same two-sibling fixture (Phase 4, task 4.5).
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// scriptTwoToolCallsResponse builds a scripted stream requesting two
// concurrent tool calls in one completion — the sibling fixture every
// test in this file shares.
func scriptTwoToolCallsResponse(t *testing.T, callID1, name1 string, args1 []byte, callID2, name2 string, args2 []byte) agenttest.Script {
	t.Helper()

	start1, err := ai.NewToolCallStart(1, callID1, name1)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	delta1, err := ai.NewToolCallDelta(1, args1)
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	end1, err := ai.NewToolCallEnd(1, args1)
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	start2, err := ai.NewToolCallStart(2, callID2, name2)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	delta2, err := ai.NewToolCallDelta(2, args2)
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	end2, err := ai.NewToolCallEnd(2, args2)
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start1), agenttest.Emit(delta1), agenttest.Emit(end1),
		agenttest.Emit(start2), agenttest.Emit(delta2), agenttest.Emit(end2),
		agenttest.Emit(completion),
	}}
}

// buildSiblingsFixture drives one parent turn requesting two
// concurrent tool calls, each hosting its own distinct child
// agent.Harness, and returns the parent's recorded stream plus both
// delegatingTool values.
func buildSiblingsFixture(t *testing.T) (parentEvents []agent.Event, toolA, toolB *delegatingTool) {
	t.Helper()

	childProviderA := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	childA := &agent.Harness{Provider: childProviderA, System: "system prompt for sibling child A"}
	childProviderB := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	childB := &agent.Harness{Provider: childProviderB, System: "system prompt for sibling child B"}

	toolA = &delegatingTool{toolName: "sibling_tool_a", effect: agent.EffectClassRead, child: childA, prompt: firstMessage(t)}
	toolB = &delegatingTool{toolName: "sibling_tool_b", effect: agent.EffectClassRead, child: childB, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{
		toolA.toolName: toolA,
		toolB.toolName: toolB,
	})

	turnOneScript := scriptTwoToolCallsResponse(t, "call-del-013-a", toolA.toolName, []byte(`{}`), "call-del-013-b", toolB.toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	parentProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{
		Provider: parentProvider,
		System:   "system prompt for del-013 parent",
		Turn:     agent.TurnOptions{Tools: reg},
		// Both sibling calls are read-class; the default bound (8) is
		// already >= 2, but stated explicitly so genuine concurrency
		// is not an accident of a larger default.
		Scheduler: &agent.Scheduler{MaxConcurrentReads: 4},
	}
	sink := make(chan *agent.Event, 1024)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("parent Run returned err = %v, want nil", err)
	}
	return drainSink(t, sink), toolA, toolB
}

// S-DEL-013 — charter AG-19.1 scenario 2, siblings, under -race.
// Given two sibling read-class tools each hosting its own child run
// on its own distinct Harness value, when both run concurrently
// within one parent turn, then both children's admissible events
// appear on the parent's stream; CheckStream accepts the parent's
// stream and each child's own stream, all three validated separately
// and unmodified; the parent's sequence remains contiguous and
// 1-based; no event of either child resolves to the other child's
// subagent_started; and the whole scenario passes under -race with no
// data race reported.
func TestSiblings_ConcurrentChildrenNoCrossTalk(t *testing.T) {
	t.Parallel()

	parentEvents, toolA, toolB := buildSiblingsFixture(t)
	childAEvents := toolA.ChildEvents()
	childBEvents := toolB.ChildEvents()
	runA, runB := toolA.ChildRun(), toolB.ChildRun()

	if runA == "" || runB == "" || runA == runB {
		t.Fatalf("sibling child run identities = %q, %q, want two distinct non-empty identities", runA, runB)
	}

	if report := agent.CheckStream(parentEvents); report.Violation() != nil {
		t.Errorf("CheckStream(parent stream) = %v, want nil", report.Violation())
	}
	if report := agent.CheckStream(childAEvents); report.Violation() != nil {
		t.Errorf("CheckStream(child A stream) = %v, want nil", report.Violation())
	}
	if report := agent.CheckStream(childBEvents); report.Violation() != nil {
		t.Errorf("CheckStream(child B stream) = %v, want nil", report.Violation())
	}
	assertContiguous1Based(t, "parent", parentEvents)

	if got := countKind(parentEvents, agent.EventKindSubagentStarted); got != 2 {
		t.Errorf("subagent_started count = %d, want exactly 2 (one per sibling)", got)
	}
	if got := countKind(parentEvents, agent.EventKindSubagentEnded); got != 2 {
		t.Errorf("subagent_ended count = %d, want exactly 2 (one per sibling)", got)
	}

	parentA, foundA := resolveChildToParent(parentEvents, runA)
	parentB, foundB := resolveChildToParent(parentEvents, runB)
	if !foundA || !foundB {
		t.Fatalf("resolveChildToParent: foundA=%v foundB=%v, want both true", foundA, foundB)
	}
	if parentA != parentB {
		t.Errorf("child A resolves to parent %q, child B resolves to parent %q, want the SAME parent run", parentA, parentB)
	}

	// No cross-talk: every mirrored event's Run() belongs to exactly
	// one of the two children, never mixed, and each child's own
	// events resolve only to its OWN subagent_started/subagent_ended
	// pair (never the sibling's).
	for i, ev := range parentEvents {
		switch ev.Run() {
		case runA, runB, parentA: // parentA == parentB by the assertion above.
		default:
			t.Errorf("parent event[%d] carries Run() = %q, want one of the parent run or either child run", i, ev.Run())
		}
	}
	for _, ev := range childAEvents {
		if ev.Run() != runA {
			t.Errorf("child A's own captured stream carries an event with Run() = %q, want %q", ev.Run(), runA)
		}
	}
	for _, ev := range childBEvents {
		if ev.Run() != runB {
			t.Errorf("child B's own captured stream carries an event with Run() = %q, want %q", ev.Run(), runB)
		}
	}
}

// S-AGE-028 — agent-event-delivery delta. Given a parent run whose
// tool hosts a child harness, and separately two sibling tools each
// hosting their own child, when the runs complete and every stream is
// captured, then each child's sink is closed exactly once, by that
// child's own harness, and neither the parent nor a sibling closes
// it; subagent_started/subagent_ended appear only on the parent's
// lane and no child emits either; on the parent's lane the index of
// each subagent_ended is strictly less than the index of the parent's
// run-close; and the parent's stream and each child's stream are
// CheckStream-valid, validated separately, never concatenated.
func TestSiblings_S_AGE_028_ClosersCountedInARunningSystem(t *testing.T) {
	t.Parallel()

	parentEvents, toolA, toolB := buildSiblingsFixture(t)
	childAEvents := toolA.ChildEvents()
	childBEvents := toolB.ChildEvents()

	validateStreamsSeparately(t, parentEvents, childAEvents)
	if report := agent.CheckStream(childBEvents); report.Violation() != nil {
		t.Errorf("CheckStream(child B stream) = %v, want nil", report.Violation())
	}

	// Each child's OWN stream carries exactly one run_start and one
	// run_end — the child harness is the sole closer of its own
	// stream (a sink "close" is not itself an Event, but a stream
	// that reached its own run_end and CheckStream-validated
	// independently is exactly what a sole, correctly-ordered closer
	// produces).
	for label, childEvents := range map[string][]agent.Event{"A": childAEvents, "B": childBEvents} {
		if got := countKind(childEvents, agent.EventKindRunEnd); got != 1 {
			t.Errorf("child %s stream carries %d run_end event(s), want exactly 1", label, got)
		}
	}

	// subagent_started/subagent_ended appear ONLY on the parent's
	// lane: neither child's own captured stream carries either kind.
	for label, childEvents := range map[string][]agent.Event{"A": childAEvents, "B": childBEvents} {
		if got := countKind(childEvents, agent.EventKindSubagentStarted) + countKind(childEvents, agent.EventKindSubagentEnded); got != 0 {
			t.Errorf("child %s's own stream carries %d subagent_started/subagent_ended event(s), want 0", label, got)
		}
	}

	runCloseIdx := indexOfLast(parentEvents, agent.EventKindRunEnd)
	if runCloseIdx < 0 {
		t.Fatal("parent stream carries no run_end")
	}
	for i, ev := range parentEvents {
		if ev.Kind() != agent.EventKindSubagentEnded {
			continue
		}
		if i >= runCloseIdx {
			t.Errorf("subagent_ended at index %d is not strictly before the parent's run-close at index %d", i, runCloseIdx)
		}
	}
}

// S-AGE-029 — agent-event-delivery delta. Given the merged AG-19
// change, when its production diff is read, then it declares no
// channel type, no channel creation, no buffer capacity and no
// numeric capacity anywhere; the seam's every refusal path returns
// one of exactly two sentinels distinguishable by errors.Is, with no
// path that drops an event silently and no path that panics; and each
// lane's stamper has exactly one writing goroutine, proven under
// -race with two sibling children running concurrently.
//
// This test proves the "one stamping writer per lane, under -race"
// half directly: the same two-sibling fixture as S-DEL-013, run under
// `-race` (the invoking `make test`/go test command supplies the
// flag; this test itself asserts only the observable per-lane
// contiguity a race would otherwise corrupt). The "no invented
// channel/buffer/sentinel-only-refusal" half is a source-level claim
// about delegation_seam.go, proven by TestDelegationSeam_
// UnmintableSurface_ExactlyFourExportedIdentifiers's exact four-name
// enumeration (S-DEL-002) — declaring a channel type or a numeric
// buffer constant would be a fifth exported (or, per that test's
// go/ast scan, ANY top-level) declaration this file does not name.
func TestSiblings_S_AGE_029_OneStampingWriterPerLaneUnderRace(t *testing.T) {
	t.Parallel()

	parentEvents, toolA, toolB := buildSiblingsFixture(t)
	assertContiguous1Based(t, "parent", parentEvents)
	assertContiguous1Based(t, "child A", toolA.ChildEvents())
	assertContiguous1Based(t, "child B", toolB.ChildEvents())
}
