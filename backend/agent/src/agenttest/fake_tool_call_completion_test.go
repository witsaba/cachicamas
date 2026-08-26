// CH-11 — AG-23 kit-gap closure for tool-call scripted responses. The
// CH-11 chat-archetype acceptance (chat_acceptance_test.go, phases 7 and
// 8) needs to drive a scripted provider that emits one tool call then
// completes with a chosen finishReason — the post-CH-09 kit surface. The
// only reachable helper today is mustToolCallScript in fake_tool_call_test.go
// (package agenttest_test, unreachable from chat_test) and the
// package-private scriptToolCallResponse in agent/harness_test.go:1095
// (hard-coded block 1, hard-coded FinishReasonToolCalls, args as []byte).
//
// ToolCallThenCompletionScript lives in package agenttest_test and is the
// WHITE-BOX smoke for agenttest.ToolCallThenCompletionScript's exported
// counterpart in fake_provider.go. Mirrors mustToolCallScript's posture:
// the helper under test is the production-posture builder; this test
// file pins its signature + behavior byte-for-byte from the chat
// archetype's first real consumer.
package agenttest_test

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// TestToolCallThenCompletionScript_EmitsToolCallThenCompletion — CH-11
// kit-gap cover. Given the helper under test invoked with a chosen
// block index, call id, tool name, arguments, and finishReason, when a
// scripted provider drains the script, then the events land in the
// order: ToolCallStart(block,id,name) → ToolCallDelta(block,args) →
// ToolCallEnd(block,args) → Completion(finishReason). Verifies the
// signature is the one chat_test reaches for (block explicit, args as
// string, finishReason explicit — both width axes the prior kit
// surface lacked).
//
// This is the WHITE-BOX smoke; the chat_acceptance_test.go phases 7 + 8
// are the integration cover that proves the helper is reachable from
// package chat_test.
func TestToolCallThenCompletionScript_EmitsToolCallThenCompletion(t *testing.T) {
	t.Parallel()

	const (
		wantBlock  ai.BlockIndex = 1
		wantID                   = "call-ct11-001"
		wantName                 = "current_time"
		wantArgs                 = `{"now":"2026-08-26T00:00:00Z"}`
		wantFinish               = ai.FinishReasonStop
	)

	script := agenttest.ToolCallThenCompletionScript(t, wantBlock, wantID, wantName, wantArgs, wantFinish)

	provider := agenttest.NewProvider(script)
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	events := drainFake(t, ch)

	if len(events) != 4 {
		t.Fatalf("drained %d event(s), want 4 (start, delta, end, completion)", len(events))
	}

	// 1. ToolCallStart — block + id + name
	start, ok := events[0].ToolCallStart()
	if !ok {
		t.Fatalf("event[0] kind = %v, want ToolCallStart", events[0].Kind())
	}
	if start.BlockIndex() != wantBlock {
		t.Errorf("ToolCallStart.BlockIndex = %v, want %v", start.BlockIndex(), wantBlock)
	}
	if start.ID() != wantID {
		t.Errorf("ToolCallStart.ID = %q, want %q", start.ID(), wantID)
	}
	if start.Name() != wantName {
		t.Errorf("ToolCallStart.Name = %q, want %q", start.Name(), wantName)
	}

	// 2. ToolCallDelta — full args bytes
	delta, ok := events[1].ToolCallDelta()
	if !ok {
		t.Fatalf("event[1] kind = %v, want ToolCallDelta", events[1].Kind())
	}
	if delta.BlockIndex() != wantBlock {
		t.Errorf("ToolCallDelta.BlockIndex = %v, want %v", delta.BlockIndex(), wantBlock)
	}
	if string(delta.Fragment()) != wantArgs {
		t.Errorf("ToolCallDelta.Fragment = %q, want %q (scripted verbatim)", string(delta.Fragment()), wantArgs)
	}

	// 3. ToolCallEnd — full args bytes (independent of deltas, tool_call_event.go convention)
	end, ok := events[2].ToolCallEnd()
	if !ok {
		t.Fatalf("event[2] kind = %v, want ToolCallEnd", events[2].Kind())
	}
	if end.BlockIndex() != wantBlock {
		t.Errorf("ToolCallEnd.BlockIndex = %v, want %v", end.BlockIndex(), wantBlock)
	}
	if string(end.Arguments()) != wantArgs {
		t.Errorf("ToolCallEnd.Arguments = %q, want %q", string(end.Arguments()), wantArgs)
	}

	// 4. Completion — finishReason is the one the helper was passed
	completion, ok := events[3].Completion()
	if !ok {
		t.Fatalf("event[3] kind = %v, want Completion", events[3].Kind())
	}
	if completion.FinishReason() != wantFinish {
		t.Errorf("Completion.FinishReason = %v, want %v", completion.FinishReason(), wantFinish)
	}
}