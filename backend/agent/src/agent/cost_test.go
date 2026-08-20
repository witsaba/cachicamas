// AG-19.3 — Phase 3 (U3): child cost is a CONSUMER-SIDE
// reconstruction; no Layer-2 fold, ever (R-DEL-007, charter AG-19.3
// scenario 1). Folds in the agent-cost-events delta's S-CST-023/
// S-CST-024, which reuse this same fixture (Phase 3, task 3.8).
package agent_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// scriptToolCallResponseWithUsage is scriptToolCallResponse's
// usage-parametrized sibling — S-DEL-016's fixture needs the PARENT's
// own turn to carry a known, non-zero figure too, so the "parent
// alone vs. parent+child" comparison is never accidentally vacuous on
// the parent's own side.
func scriptToolCallResponseWithUsage(t *testing.T, callID, toolName string, args []byte, usage ai.Usage) agenttest.Script {
	t.Helper()

	start, err := ai.NewToolCallStart(1, callID, toolName)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart: %v", err)
	}
	delta, err := ai.NewToolCallDelta(1, args)
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta: %v", err)
	}
	end, err := ai.NewToolCallEnd(1, args)
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd: %v", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, usage)
	if err != nil {
		t.Fatalf("ai.NewCompletion: %v", err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start), agenttest.Emit(delta), agenttest.Emit(end), agenttest.Emit(completion),
	}}
}

// costFixtureResult carries every stream and figure S-DEL-016,
// S-DEL-017, S-CST-023 and S-CST-024 need.
type costFixtureResult struct {
	parentEvents []agent.Event
	childEvents  []agent.Event
	childRun     agent.RunID
	parentRun    agent.RunID
}

// buildCostFixture drives a parent run whose tool hosts a child that
// provably spends non-zero tokens (hazard #2: a zero-usage fixture
// makes S-DEL-016's strict inequality vacuously satisfiable in the
// wrong direction), and returns both streams.
func buildCostFixture(t *testing.T) costFixtureResult {
	t.Helper()

	childUsage := ai.Usage{Input: ai.Tokens(77), Output: ai.Tokens(33)}
	childProvider := agenttest.NewProvider(scriptTextResponseWithUsage(t, ai.FinishReasonStop, childUsage))
	child := &agent.Harness{Provider: childProvider, System: "system prompt for the cost child"}

	toolName := "cost_tool"
	tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	parentTurnOneUsage := ai.Usage{Input: ai.Tokens(9), Output: ai.Tokens(4)}
	parentTurnTwoUsage := ai.Usage{Input: ai.Tokens(6), Output: ai.Tokens(2)}
	turnOneScript := scriptToolCallResponseWithUsage(t, "call-del-016", toolName, []byte(`{}`), parentTurnOneUsage)
	turnTwoScript := scriptTextResponseWithUsage(t, ai.FinishReasonStop, parentTurnTwoUsage)
	parentProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: parentProvider, System: "system prompt for del-016 parent", Turn: agent.TurnOptions{Tools: reg}}
	sink := make(chan *agent.Event, 512)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("parent Run returned err = %v, want nil", err)
	}
	parentEvents := drainSink(t, sink)
	childRun := tool.ChildRun()
	parentRun, found := resolveChildToParent(parentEvents, childRun)
	if !found {
		t.Fatal("resolveChildToParent found no subagent_started for the child run")
	}
	return costFixtureResult{parentEvents: parentEvents, childEvents: tool.ChildEvents(), childRun: childRun, parentRun: parentRun}
}

// sumCostTurnInputTokens sums InputTokens over every cost_turn event
// in events whose Event.Run() equals run. Used as the
// "reconstruction a consumer performs" (R-DEL-007): a single figure
// is enough to prove the strict inequality and the fold-guard
// non-vacuously; the paired accessors on CostTurn/CostSession cover
// every other figure identically. Filtered by run identity
// (verify-report WARNING-1): S-DEL-016 says "the sum over the
// cost_turn events observed on the parent's OWN stream" — an
// unfiltered sum moves in lockstep with a leaked foreign cost_turn
// (both the test's own reconstruction and harness.go's unfiltered
// fold inflate together), making the equality assertion below
// fold-blind.
func sumCostTurnInputTokens(events []agent.Event, run agent.RunID) uint64 {
	var total uint64
	for _, ev := range events {
		if ev.Run() != run {
			continue
		}
		ct, ok := ev.CostTurn()
		if !ok {
			continue
		}
		if in, present := ct.InputTokens(); present {
			total += in
		}
	}
	return total
}

// S-DEL-016 — charter AG-19.3 scenario 1, and the inequality is what
// makes it non-vacuous. Given a child run that provably spends
// non-zero tokens, when the parent's cost events are inspected, then
// the parent's terminal cost_session figures equal the sum over the
// cost_turn events observed on the parent's own stream, and are
// strictly less than the parent-plus-child sum on at least one
// reported figure; a consumer walking both streams by parent identity
// recovers the combined figure; and no cost_turn carrying the child's
// run identity appears anywhere on the parent's stream.
func TestCost_ParentAloneStrictlyLessThanCombined(t *testing.T) {
	t.Parallel()

	got := buildCostFixture(t)

	parentOwnInput := sumCostTurnInputTokens(got.parentEvents, got.parentRun)
	childInput := sumCostTurnInputTokens(got.childEvents, got.childRun)
	combined := parentOwnInput + childInput

	if childInput == 0 {
		t.Fatal("child's own cost_turn input-token sum = 0, want > 0 (hazard #2: a zero-usage child fixture makes this scenario vacuous)")
	}
	if parentOwnInput >= combined {
		t.Errorf("parent-own input tokens = %d, combined (parent+child) = %d, want parent-own strictly less than combined", parentOwnInput, combined)
	}

	// The parent's terminal cost_session (last one carrying the
	// PARENT's own run identity) equals the sum over the parent's own
	// cost_turn events.
	var parentFinalInput uint64
	found := false
	for i := len(got.parentEvents) - 1; i >= 0; i-- {
		ev := got.parentEvents[i]
		if ev.Run() != got.parentRun {
			continue
		}
		cs, ok := ev.CostSession()
		if !ok {
			continue
		}
		if in, present := cs.InputTokens(); present {
			parentFinalInput = in
		}
		found = true
		break
	}
	if !found {
		t.Fatal("no cost_session carrying the parent's own run identity found on the parent's stream")
	}
	if parentFinalInput != parentOwnInput {
		t.Errorf("parent's terminal cost_session input tokens = %d, want %d (the sum over the parent's own cost_turn events)", parentFinalInput, parentOwnInput)
	}

	// No cost_turn carrying the child's run identity appears anywhere
	// on the parent's stream — the seam's gate 5 refusal, proven
	// two-sidedly (hazard #3): here, the ABSENCE half.
	for i, ev := range got.parentEvents {
		if ev.Kind() != agent.EventKindCostTurn {
			continue
		}
		if ev.Run() == got.childRun {
			t.Errorf("parent event[%d] is a cost_turn carrying the CHILD's run identity — R-DEL-002 gate 5 must have refused this", i)
		}
	}
}

// S-DEL-017 — the crossed cost_session is present, attributable and
// inert. Given that same run, when the parent's stream is walked,
// then it carries the child's final cost_session reporting the
// child's run identity, positioned inside the hosting turn's
// bracket; the parent's own final cost_session reports the parent's
// run identity and sits immediately before the parent's run-close;
// the two are distinguished by run identity and not by label; and
// CheckStream accepts the parent's stream unmodified.
func TestCost_CrossedCostSession_PresentAttributableInert(t *testing.T) {
	t.Parallel()

	got := buildCostFixture(t)

	if report := agent.CheckStream(got.parentEvents); report.Violation() != nil {
		t.Errorf("CheckStream(parent stream) = %v, want nil", report.Violation())
	}

	turnStartIdx := indexOfFirst(got.parentEvents, agent.EventKindTurnStart)
	turnEndIdx := indexOfFirst(got.parentEvents, agent.EventKindTurnEnd)
	if turnStartIdx < 0 || turnEndIdx < 0 {
		t.Fatal("parent stream missing turn_start/turn_end")
	}

	childCostSessionIdx := -1
	for i, ev := range got.parentEvents {
		if ev.Kind() != agent.EventKindCostSession {
			continue
		}
		if ev.Run() == got.childRun {
			childCostSessionIdx = i
		}
	}
	if childCostSessionIdx < 0 {
		t.Fatal("no cost_session carrying the child's run identity found on the parent's stream")
	}
	if turnStartIdx >= childCostSessionIdx || childCostSessionIdx >= turnEndIdx {
		t.Errorf("child's crossed cost_session at index %d is not inside the hosting turn's bracket [%d,%d]", childCostSessionIdx, turnStartIdx, turnEndIdx)
	}
	cs, ok := got.parentEvents[childCostSessionIdx].CostSession()
	if !ok {
		t.Fatal("event at childCostSessionIdx carries no CostSession payload")
	}
	if cs.Label() != agent.CostLabelFinal {
		t.Errorf("child's crossed cost_session label = %v, want CostLabelFinal", cs.Label())
	}

	parentRunCloseIdx := indexOfLast(got.parentEvents, agent.EventKindRunEnd)
	parentFinalCostSessionIdx := -1
	for i, ev := range got.parentEvents {
		if ev.Kind() != agent.EventKindCostSession {
			continue
		}
		if ev.Run() == got.parentRun {
			parentFinalCostSessionIdx = i
		}
	}
	if parentFinalCostSessionIdx < 0 || parentRunCloseIdx < 0 || parentFinalCostSessionIdx != parentRunCloseIdx-1 {
		t.Errorf("parent's own final cost_session index = %d, want exactly one before the run-close (index %d)", parentFinalCostSessionIdx, parentRunCloseIdx)
	}

	// Distinguished by run identity, not by label: both the child's
	// crossed cost_session and the parent's own final carry
	// CostLabelFinal — the discriminator that separates them is
	// Event.Run(), asserted above by construction (both indices were
	// selected by filtering on Run(), never on label).
	if childCostSessionIdx == parentFinalCostSessionIdx {
		t.Error("the child's crossed cost_session and the parent's own final cost_session resolved to the SAME index — they must be two distinct events")
	}
}

// S-CST-023 — AG-19: a mirrored child figure does not disturb the
// run's own protocol. Given a parent run whose tool hosts a child run
// spending non-zero tokens, when the parent's delivered stream is
// recorded, then it carries the child's final cost_session reporting
// the child's run identity inside the hosting turn bracket; filtering
// to the parent's own run identity yields exactly the sequence
// R-CST-005 requires; the parent's own final figure equals the sum
// over the cost_turn events on the parent's stream and is strictly
// less than the parent-plus-child sum; and CheckStream accepts the
// parent's stream unmodified with stream_check.go byte-unchanged
// (asserted structurally: this change never edits stream_check.go,
// confirmed by R-DEL-009's own scope-fence test, S-DEL-024).
func TestCost_S_CST_023_MirroredFigureDoesNotDisturbOwnProtocol(t *testing.T) {
	t.Parallel()

	got := buildCostFixture(t)

	if report := agent.CheckStream(got.parentEvents); report.Violation() != nil {
		t.Errorf("CheckStream(parent stream) = %v, want nil", report.Violation())
	}

	// Filtering to the parent's own run identity: zero-or-more
	// Estimate cost_session events, then exactly one Final,
	// immediately before the run-close.
	var ownSessions []agent.Event
	for _, ev := range got.parentEvents {
		if ev.Kind() == agent.EventKindCostSession && ev.Run() == got.parentRun {
			ownSessions = append(ownSessions, ev)
		}
	}
	if len(ownSessions) == 0 {
		t.Fatal("filtering to the parent's own run identity yields zero cost_session events, want at least one Final")
	}
	last := ownSessions[len(ownSessions)-1]
	lastCS, _ := last.CostSession()
	if lastCS.Label() != agent.CostLabelFinal {
		t.Errorf("last cost_session filtered to the parent's own run identity carries label %v, want CostLabelFinal", lastCS.Label())
	}
	for _, ev := range ownSessions[:len(ownSessions)-1] {
		cs, _ := ev.CostSession()
		if cs.Label() != agent.CostLabelEstimate {
			t.Errorf("a non-terminal cost_session filtered to the parent's own run identity carries label %v, want CostLabelEstimate", cs.Label())
		}
	}

	parentOwnInput := sumCostTurnInputTokens(got.parentEvents, got.parentRun)
	childInput := sumCostTurnInputTokens(got.childEvents, got.childRun)
	if in, present := lastCS.InputTokens(); !present || in != parentOwnInput {
		t.Errorf("parent's own final cost_session input tokens = (%d, present=%v), want (%d, true)", in, present, parentOwnInput)
	}
	if parentOwnInput >= parentOwnInput+childInput {
		t.Errorf("parent-own (%d) is not strictly less than parent+child (%d)", parentOwnInput, parentOwnInput+childInput)
	}
}

// S-CST-024 — AG-19: the fold is unreachable, not merely unused.
// Given the merged change, when a cost_turn is published through the
// delegation seam, then the publish returns the typed inadmissibility
// sentinel and the event reaches no sink; and when the accumulating
// fold at harness.go:633-635 is read, then it is byte-unchanged and
// gained no run-identity filter — because none is needed once no
// foreign cost_turn can arrive.
func TestCost_S_CST_024_FoldUnreachableNotMerelyUnused(t *testing.T) {
	t.Parallel()

	var pubErr error
	toolName := "cst024_tool"
	tool := &seamCapturingTool{
		toolName: toolName,
		beforeReturn: func(seam agent.DelegationSeam) {
			run, turn := seam.Parent()
			ev, err := agent.NewCostTurn(run, turn, agent.CostLabelFinal, agent.CostFigures{InputTokens: 9000})
			if err != nil {
				t.Fatalf("agent.NewCostTurn: %v", err)
			}
			pubErr = seam.Publish(ev)
		},
	}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})
	calls := callsToAICalls([]scheduledCall{{id: "cst024-call", name: toolName, args: []byte(`{}`), effect: agent.EffectClassRead}})

	sink := make(chan *agent.Event, 16)
	sched := &agent.Scheduler{}
	_ = sched.Schedule(contextBackground(), calls, reg, "run-cst-024", "turn-cst-024", nil, &agent.LaneStamper{}, sink)
	events := drainSink(t, sink)

	if !errors.Is(pubErr, agent.ErrDelegationInadmissible) {
		t.Errorf("Publish(cost_turn) = %v, want errors.Is(_, ErrDelegationInadmissible)", pubErr)
	}
	if got := countKind(events, agent.EventKindCostTurn); got != 0 {
		t.Errorf("cost_turn count on sink = %d, want 0 — the publish must have reached no sink", got)
	}
}
