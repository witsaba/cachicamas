// AG-19.1 — Phase 2 (U2): a nested run's events on the parent's
// stream (R-DEL-002, R-DEL-004, R-DEL-005, charter AG-19.1 scenario
// 1).
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-DEL-010 — charter AG-19.1 scenario 1, the interleaving half.
// Given a test-only tool hosting a child harness running a scripted
// conversation, when the parent run completes, then the parent's
// recorded stream carries, in order, exactly one subagent_started,
// the admissible child events of R-DEL-002, and exactly one
// subagent_ended, all inside the hosting turn's bracket;
// CheckStream accepts the parent's stream unmodified; and the
// parent's sequence is contiguous and 1-based across the whole run
// with no gap, repeat or restart.
func TestNestedRun_ParentStreamCarriesSubagentBracketAndAdmissibleChildEvents(t *testing.T) {
	t.Parallel()

	childProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	child := &agent.Harness{Provider: childProvider, System: "system prompt for the hosted child"}

	toolName := "nested_run_tool"
	tool := &delegatingTool{toolName: toolName, effect: agent.EffectClassRead, child: child, prompt: firstMessage(t)}
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	turnOneScript := scriptToolCallResponse(t, "call-del-010", toolName, []byte(`{}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	parentProvider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: parentProvider, System: "system prompt for del-010 parent", Turn: agent.TurnOptions{Tools: reg}}
	sink := make(chan *agent.Event, 512)
	_, _, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("parent Run returned err = %v, want nil", err)
	}
	parentEvents := drainSink(t, sink)
	childEvents := tool.ChildEvents()

	validateStreamsSeparately(t, parentEvents, childEvents)

	if got := countKind(parentEvents, agent.EventKindSubagentStarted); got != 1 {
		t.Errorf("subagent_started count on parent stream = %d, want exactly 1", got)
	}
	if got := countKind(parentEvents, agent.EventKindSubagentEnded); got != 1 {
		t.Errorf("subagent_ended count on parent stream = %d, want exactly 1", got)
	}

	startedIdx := indexOfFirst(parentEvents, agent.EventKindSubagentStarted)
	endedIdx := indexOfFirst(parentEvents, agent.EventKindSubagentEnded)
	if startedIdx < 0 || endedIdx < 0 || startedIdx >= endedIdx {
		t.Fatalf("subagent_started index = %d, subagent_ended index = %d, want started strictly before ended, both present", startedIdx, endedIdx)
	}

	// Every mirrored kind between the bracket markers must be one of
	// R-DEL-002's admitted kinds. The child's own message_start_text
	// (turn 1) proves at least one ordinary message kind crossed.
	sawMessageStartText := false
	for i := startedIdx + 1; i < endedIdx; i++ {
		if parentEvents[i].Kind() == agent.EventKindMessageStartText {
			sawMessageStartText = true
		}
		switch parentEvents[i].Kind() {
		case agent.EventKindRunStart, agent.EventKindRunEnd, agent.EventKindTurnStart, agent.EventKindTurnEnd,
			agent.EventKindPermissionResolutionRemembered, agent.EventKindCostTurn:
			t.Errorf("event %d between the subagent bracket carries refused kind %v — it must never have been mirrored", i, parentEvents[i].Kind())
		}
	}
	if !sawMessageStartText {
		t.Error("no message_start_text observed between subagent_started and subagent_ended — the child's own text turn did not mirror")
	}

	// Both markers must fall inside the hosting turn's own bracket:
	// the parent's turn_start/turn_end surrounding the tool call must
	// each be found, with the bracket strictly containing both
	// markers.
	turnStartIdx := indexOfFirst(parentEvents, agent.EventKindTurnStart)
	turnEndIdx := indexOfFirst(parentEvents, agent.EventKindTurnEnd)
	if turnStartIdx < 0 || turnEndIdx < 0 || turnStartIdx >= startedIdx || endedIdx >= turnEndIdx {
		t.Errorf("subagent bracket [%d,%d] is not strictly inside the hosting turn's bracket [%d,%d]", startedIdx, endedIdx, turnStartIdx, turnEndIdx)
	}

	if len(tool.PublishErrors()) == 0 {
		t.Error("PublishErrors() is empty, want at least one refusal — the child's own run_start/run_end/turn_start/turn_end must each have been refused at the seam")
	}
}
