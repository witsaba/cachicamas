// AG-13.3 — pause resumption: the partial is replayed verbatim, and the
// pause stays visible (R-RUN-009). The mechanism is entirely
// compositional: Phase 1's continuation-mode Turn already appends the
// turn's own partial message (finishContinuationTurn, R-HIS-010) and
// AG-11's finalize already emits turn_end(TurnOutcomePaused)
// unconditionally; Phase 2's Run dispatch already iterates on
// ai.FinishReasonPauseTurn exactly like ai.FinishReasonToolCalls
// (R-RUN-002). This file's tests pin that composition from AG-13.3's own
// charter angle — jointly closes S-ATT-013.
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AG-13.3 — S-RUN-080. Charter AG-13.3, first Then. Given a provider
// scripted with turn one delivering partial text and reasoning carrying
// a non-empty round-trip token and completing with
// ai.FinishReasonPauseTurn, and turn two delivering final text with a
// terminal finish reason, when the run is driven, then the transcript
// entry for the paused turn is deep-equal to the message Turn returned
// including the round-trip token bytes, the request recorded for turn
// two contains that entry byte-verbatim, and the run ends with the
// completed run outcome.
func TestHarness_PauseFinish_ResumesVerbatimToRealTerminal(t *testing.T) {
	t.Parallel()

	reasoningText := "considering the pause-080 request"
	token := []byte{0x00, 0x01, 0xFF, 0xFE, 0x02} // non-empty, arbitrary binary round-trip token.

	turnOneScript := scriptReasoningTextResponse(t, ai.FinishReasonPauseTurn, reasoningText, token)
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for run-080", History: hist}

	sink := make(chan *agent.Event, 256)
	_, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	if finish != ai.FinishReasonStop {
		t.Errorf("finish = %v, want %v (turn two's real terminal, past the pause)", finish, ai.FinishReasonStop)
	}
	events := drainSink(t, sink)

	// The expected partial: reasoning (carrying the round-trip token)
	// then text — the exact reconstruction order loop.go's own
	// reconstructMessage builds (reasoning before text before, on the
	// continuation path, tool calls).
	reasoningPart, err := ai.NewReasoning(reasoningText, token)
	if err != nil {
		t.Fatalf("ai.NewReasoning: %v", err)
	}
	textPart, err := ai.NewText("hello, world")
	if err != nil {
		t.Fatalf("ai.NewText: %v", err)
	}
	wantPartial, err := ai.NewMessage(ai.RoleAssistant, reasoningPart, textPart)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}

	entries := hist.Entries()
	if len(entries) != 3 {
		t.Fatalf("history has %d entries, want 3 (prompt, turn-one partial, turn-two final)", len(entries))
	}
	pausedEntry := entries[1].Message()
	if !pausedEntry.Equal(wantPartial) {
		t.Error("the paused turn's transcript entry is not byte-identical to the partial content received, including the reasoning round-trip token — it must never be discarded, re-synthesized, re-serialized, truncated or merged")
	}

	// Turn two's recorded request replays that entry byte-verbatim —
	// proof by evidence, not by transcript order alone.
	requests := provider.Requests()
	if len(requests) != 2 {
		t.Fatalf("provider recorded %d request(s), want 2", len(requests))
	}
	var foundVerbatim bool
	for _, m := range requests[1].Messages() {
		if m.Equal(pausedEntry) {
			foundVerbatim = true
		}
	}
	if !foundVerbatim {
		t.Error("turn two's recorded request does not replay the paused entry byte-verbatim")
	}

	last := events[len(events)-1]
	runEnd, ok := last.RunEnd()
	if !ok {
		t.Fatalf("last event kind = %v, want run_end", last.Kind())
	}
	if runEnd.Outcome() != agent.RunOutcomeCompleted {
		t.Errorf("run_end outcome = %v, want RunOutcomeCompleted — the run must continue past the pause to a real terminal", runEnd.Outcome())
	}
}

// AG-13.3 — S-RUN-081. Charter AG-13.3, second Then. Given the same run
// shape, when the event stream is read, then turn one's turn-close event
// carries the paused turn outcome, that outcome value is different from
// both the finished and the aborted outcome values, and no assertion in
// the scenario reads an ai.FinishReason to establish it — the harness
// forwards the turn-close event unrewritten, never absorbing or
// rewriting the outcome. Jointly closes S-ATT-013.
func TestHarness_PauseFinish_TurnEndCarriesPausedOutcomeVisibleAndForwarded(t *testing.T) {
	t.Parallel()

	turnOneScript := scriptReasoningTextResponse(t, ai.FinishReasonPauseTurn, "reasoning for run-081", []byte{0xAB, 0xCD})
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	h := agent.Harness{Provider: provider, System: "system prompt for run-081"}

	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)

	var turnEndOutcomes []agent.TurnOutcome
	for _, ev := range events {
		if te, ok := ev.TurnEnd(); ok {
			turnEndOutcomes = append(turnEndOutcomes, te.Outcome())
		}
	}
	if len(turnEndOutcomes) != 2 {
		t.Fatalf("observed %d turn_end event(s), want 2 (one per turn)", len(turnEndOutcomes))
	}
	if turnEndOutcomes[0] != agent.TurnOutcomePaused {
		t.Errorf("turn one's turn_end outcome = %v, want TurnOutcomePaused — the harness must forward it unrewritten, never absorb it", turnEndOutcomes[0])
	}
	if turnEndOutcomes[1] != agent.TurnOutcomeFinished {
		t.Errorf("turn two's turn_end outcome = %v, want TurnOutcomeFinished", turnEndOutcomes[1])
	}
	if turnEndOutcomes[0] == agent.TurnOutcomeAborted {
		t.Error("paused outcome equals TurnOutcomeAborted, want a distinct value")
	}
	if turnEndOutcomes[0] == turnEndOutcomes[1] {
		t.Error("paused and finished outcome values are equal, want distinct values")
	}
}
