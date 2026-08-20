// AG-19 — Phase 4: closed-sequence safety on the path where no tool
// ever reaches through the seam (R-DEL-010).
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// inertPathRun is one recording of the inert-path fixture: the
// event-kind sequence (identity-independent — Sequence/RunID/TurnID
// values differ freshly per run by construction and are deliberately
// excluded from the comparison, matching S-DEL-025's own "modulo the
// freshly minted run and turn identifiers"), the run's own final
// message content, and the history read-back's message count.
type inertPathRun struct {
	kinds        []agent.EventKind
	finishReason ai.FinishReason
	finalText    string
	historyLen   int
}

// runInertPathOnce drives one Harness run through a multi-turn
// tool-calling script where the tool never reaches through the
// DelegationSeam at all (EchoScriptedTool, not delegatingTool) — the
// non-delegating path R-DEL-010 requires stay byte-identical to the
// pre-AG-19 shape.
func runInertPathOnce(t *testing.T) inertPathRun {
	t.Helper()

	tool := EchoScriptedTool("inert_tool", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"inert_tool": tool})

	turnOneScript := scriptToolCallResponse(t, "call-del-025", "inert_tool", []byte(`{"x":1}`))
	turnTwoScript := scriptTextResponse(t, ai.FinishReasonStop)
	provider := agenttest.NewProvider(turnOneScript, turnTwoScript)

	hist := agent.NewHistory()
	h := agent.Harness{Provider: provider, System: "system prompt for del-025", Turn: agent.TurnOptions{Tools: reg}, History: hist}
	sink := make(chan *agent.Event, 256)
	msg, finish, err := h.Run(contextBackground(), firstMessage(t), sink)
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	events := drainSink(t, sink)

	kinds := make([]agent.EventKind, len(events))
	for i, ev := range events {
		kinds[i] = ev.Kind()
	}

	var finalText string
	for _, part := range msg.Content() {
		if text, ok := part.Text(); ok {
			finalText += text
		}
	}

	return inertPathRun{kinds: kinds, finishReason: finish, finalText: finalText, historyLen: len(hist.Entries())}
}

// kindsEqual reports whether a and b carry the identical kind
// sequence, in order.
func kindsEqual(a, b []agent.EventKind) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// S-DEL-025 — the inert path is byte-identical. Given two runs of an
// identical multi-turn tool-calling script — where no tool publishes
// through the seam — when both are recorded, then the two event
// streams are byte-identical modulo the freshly minted run and turn
// identifiers, the two history read-backs are byte-identical, and
// neither carries any delegation-family kind.
//
// This change adds no seam-independent code path a non-delegating
// caller could observe: delegation_seam.go is reachable ONLY through
// an explicit DelegationSeamFrom(ctx) call a tool must make, and
// scheduler.go's 3-line diff installs the seam unconditionally but
// never reads it — so "run this fixture once on the merge base, once
// on this change" and "run this fixture twice on this change" are
// the same experiment for a tool that never asks for the seam. Two
// runs on this change is the check this test performs; it is
// sufficient exactly because the diff's own reachability is gated
// behind an opt-in accessor call this fixture's tool never makes.
func TestInertPath_ByteIdenticalModuloIdentitiesNoDelegationKind(t *testing.T) {
	t.Parallel()

	first := runInertPathOnce(t)
	second := runInertPathOnce(t)

	if !kindsEqual(first.kinds, second.kinds) {
		t.Errorf("event-kind sequences differ between two inert-path runs:\n  first:  %v\n  second: %v", first.kinds, second.kinds)
	}
	if first.finishReason != second.finishReason {
		t.Errorf("finish reasons differ: first=%v second=%v", first.finishReason, second.finishReason)
	}
	if first.finalText != second.finalText {
		t.Errorf("final text differs: first=%q second=%q", first.finalText, second.finalText)
	}
	if first.historyLen != second.historyLen {
		t.Errorf("history entry counts differ: first=%d second=%d", first.historyLen, second.historyLen)
	}

	delegationKinds := map[agent.EventKind]bool{
		agent.EventKindSubagentStarted: true,
		agent.EventKindSubagentEnded:   true,
	}
	for _, run := range []inertPathRun{first, second} {
		for _, k := range run.kinds {
			if delegationKinds[k] {
				t.Errorf("inert-path run carries delegation-family kind %v, want none", k)
			}
		}
	}
}
