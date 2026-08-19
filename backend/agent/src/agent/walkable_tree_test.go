// AG-19.1 — Phase 2 (U2): the walkable tree, and the two lanes stay
// independently contiguous (R-DEL-004, R-DEL-005).
package agent_test

import (
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// buildWalkableTreeFixture drives one nested run through delegatingTool
// and returns the parent's recorded stream and the delegatingTool
// value carrying the child's own captured stream — the fixture
// S-DEL-011 and S-DEL-012 share.
func buildWalkableTreeFixture(t *testing.T) (parentEvents []agent.Event, tool *delegatingTool) {
	t.Helper()

	childProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	child := &agent.Harness{Provider: childProvider, System: "system prompt for the walkable-tree child"}

	toolName := "walkable_tree_tool"
	tool = &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-del-011", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	parentProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: parentProvider, System: "system prompt for del-011 parent", Turn: agent.TurnOptions{Tools: reg}}
	sink := make(chan *agent.Event, 512)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("parent Run returned err = %v, want nil", err)
	}
	return drainSink(t, sink), tool
}

// delegationEventConstructorParentParamCounts maps every exported
// New* event constructor this package declares to how many of its
// leading identity parameters are agent.RunID-typed. A "parent
// parameter" is a SECOND RunID parameter after the first (run,
// parent RunID, ...) — exactly NewDelegatedRunStart, NewSubagentStarted
// and NewSubagentEnded declare one; every other constructor must
// declare at most one RunID parameter (its own run identity).
func delegationEventConstructorParentParamCounts() map[string]any {
	return map[string]any{
		"NewRunStart":                       agent.NewRunStart,
		"NewDelegatedRunStart":              agent.NewDelegatedRunStart,
		"NewRunEnd":                         agent.NewRunEnd,
		"NewTurnStart":                      agent.NewTurnStart,
		"NewTurnEnd":                        agent.NewTurnEnd,
		"NewMessageStartText":               agent.NewMessageStartText,
		"NewMessageDeltaText":               agent.NewMessageDeltaText,
		"NewMessageEndText":                 agent.NewMessageEndText,
		"NewMessageStartReasoning":          agent.NewMessageStartReasoning,
		"NewMessageDeltaReasoning":          agent.NewMessageDeltaReasoning,
		"NewMessageEndReasoning":            agent.NewMessageEndReasoning,
		"NewToolStart":                      agent.NewToolStart,
		"NewToolProgress":                   agent.NewToolProgress,
		"NewToolEndSuccess":                 agent.NewToolEndSuccess,
		"NewToolEndResultFailure":           agent.NewToolEndResultFailure,
		"NewToolEndExecutionFailure":        agent.NewToolEndExecutionFailure,
		"NewPermissionDecisionRequired":     agent.NewPermissionDecisionRequired,
		"NewPermissionDecisionMade":         agent.NewPermissionDecisionMade,
		"NewPermissionResolutionRemembered": agent.NewPermissionResolutionRemembered,
		"NewCostTurn":                       agent.NewCostTurn,
		"NewCostSession":                    agent.NewCostSession,
		"NewSubagentStarted":                agent.NewSubagentStarted,
		"NewSubagentEnded":                  agent.NewSubagentEnded,
		"NewCompactionStarted":              agent.NewCompactionStarted,
		"NewCompactionFinished":             agent.NewCompactionFinished,
		"NewCompactionFailed":               agent.NewCompactionFailed,
	}
}

// countLeadingRunIDParams counts how many of fn's parameters, from
// the start, have exactly agent.RunID's reflect.Type — the shape a
// parent-bearing constructor declares as its first two parameters
// (run, parent RunID).
func countLeadingRunIDParams(fn any) int {
	runIDType := reflect.TypeOf(agent.RunID(""))
	typ := reflect.TypeOf(fn)
	n := 0
	for i := 0; i < typ.NumIn(); i++ {
		if typ.In(i) != runIDType {
			break
		}
		n++
	}
	return n
}

// S-DEL-011 — charter AG-19.1 scenario 1, the walking half. Given
// those two streams, when a consumer resolves each mirrored child
// event to a parent using only Run() and the single
// subagent_started's Parent(), then every mirrored event resolves in
// one hop; no mirrored event other than subagent_started/
// subagent_ended reports a parent of its own; and when the package's
// event construction surface is enumerated, then no constructor
// gained a parent parameter in this change.
func TestWalkableTree_EveryMirroredEventResolvesInOneHop(t *testing.T) {
	t.Parallel()

	parentEvents, tool := buildWalkableTreeFixture(t)
	childRun := tool.ChildRun()

	startedIdx := indexOfFirst(parentEvents, agent.EventKindSubagentStarted)
	endedIdx := indexOfFirst(parentEvents, agent.EventKindSubagentEnded)
	if startedIdx < 0 || endedIdx < 0 {
		t.Fatalf("subagent_started idx = %d, subagent_ended idx = %d, want both present", startedIdx, endedIdx)
	}

	resolvedParent, found := resolveChildToParent(parentEvents, childRun)
	if !found {
		t.Fatal("resolveChildToParent found no subagent_started resolving childRun")
	}

	for i, ev := range parentEvents {
		if i == startedIdx || i == endedIdx {
			continue
		}
		if ev.Run() != childRun {
			continue // not a mirrored child event.
		}
		// Every mirrored event resolves in one hop: its Run() matches
		// the one subagent_started's Run(), whose Parent() is the
		// resolved parent.
		if _, hasParent := ev.Parent(); hasParent {
			t.Errorf("mirrored event[%d] kind=%v reports its own Parent(), want false — only subagent_started/subagent_ended may", i, ev.Kind())
		}
		_ = resolvedParent
	}

	// The construction surface: exactly three constructors declare a
	// parent parameter (two leading RunID params); every other
	// constructor declares at most one.
	wantParentBearing := map[string]bool{
		"NewDelegatedRunStart": true,
		"NewSubagentStarted":   true,
		"NewSubagentEnded":     true,
	}
	for name, fn := range delegationEventConstructorParentParamCounts() {
		got := countLeadingRunIDParams(fn)
		if wantParentBearing[name] {
			if got != 2 {
				t.Errorf("%s declares %d leading RunID parameter(s), want 2 (run, parent) — it is one of the three parent-bearing constructors", name, got)
			}
			continue
		}
		if got > 1 {
			t.Errorf("%s declares %d leading RunID parameter(s), want at most 1 — no constructor outside the three named ones may gain a parent parameter", name, got)
		}
	}
}

// S-DEL-012 — the two lanes stay independently contiguous. Given the
// same run, when the child's own captured stream is walked, then its
// sequence is contiguous and 1-based independently of the parent's,
// and the mirrored copies on the parent's lane carry the parent
// lane's stamps rather than the child's — the two lanes are never
// merged and no event carries a sequence from the other lane.
func TestWalkableTree_TwoLanesIndependentlyContiguous(t *testing.T) {
	t.Parallel()

	parentEvents, tool := buildWalkableTreeFixture(t)
	childEvents := tool.ChildEvents()
	childRun := tool.ChildRun()

	validateStreamsSeparately(t, parentEvents, childEvents)

	assertContiguous1Based(t, "child", childEvents)
	assertContiguous1Based(t, "parent", parentEvents)

	// The mirrored copies on the parent's lane carry the PARENT
	// lane's stamps: for each mirrored event (Run() == childRun), its
	// Sequence() must fall inside the parent's own 1..len(parentEvents)
	// contiguous range — never a sequence value copied from the
	// child's own lane.
	for i, ev := range parentEvents {
		if ev.Run() != childRun {
			continue
		}
		if want := agent.Sequence(i + 1); ev.Sequence() != want {
			t.Errorf("mirrored parent event[%d] Sequence() = %v, want %v (the parent lane's own stamp)", i, ev.Sequence(), want)
		}
	}
}

// assertContiguous1Based asserts events' Sequence() values are
// exactly 1, 2, 3, ... with no gap, repeat or restart — the
// per-lane contiguity CheckStream itself also enforces, restated
// here as a direct, standalone assertion so a caller need not
// re-derive it from a CheckStream pass.
func assertContiguous1Based(t *testing.T, label string, events []agent.Event) {
	t.Helper()
	for i, ev := range events {
		if want := agent.Sequence(i + 1); ev.Sequence() != want {
			t.Errorf("%s lane event[%d].Sequence() = %v, want %v (contiguous, 1-based)", label, i, ev.Sequence(), want)
		}
	}
}
