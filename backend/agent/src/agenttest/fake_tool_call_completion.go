// CH-11 — AG-23 kit-gap closure for tool-call scripted responses.
//
// ToolCallThenCompletionScript is the chat-archetype acceptance's
// (chat_acceptance_test.go phases 7 + 8) scripted-provider builder:
// one ToolCallStart + ToolCallDelta + ToolCallEnd at block, then a
// Completion carrying finishReason. The finishReason is explicit
// (FinishReasonToolCalls for the call-issuing half of a tool turn,
// FinishReasonStop for the post-tool "model emits final text"
// follow-up) so the kit does not require two near-duplicate helpers.
//
// Why this lives in package agenttest (not duplicated into chat_test):
// the design's explored kit-gap resolution (Option A) commits to
// promoting tool-call scripted providers to the kit so the next
// archetype's acceptance can reuse them. The hard-coded
// scriptToolCallResponse at agent/harness_test.go:1095 is not
// reachable from chat_test (unexported, package agent_test), and its
// signature (hard-coded block 1, []byte args, FinishReasonToolCalls)
// does not fit the chat archetype's two-script needs. This helper is
// the explicit-block + string-args + explicit-finishReason variant.
//
// Mirrors mustToolCallScript's posture (agenttest_test/fake_tool_call_test.go:70):
// construction-time validation through the ai package's own
// constructors, t.Helper + t.Fatalf at every failure mode, Script{Steps:
// ...} returned so it slots into NewProvider's variadic queue verbatim.
package agenttest

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ToolCallThenCompletionScript builds a Script that emits one
// ToolCallStart + ToolCallDelta + ToolCallEnd triple at block (id,
// name, arguments — the same byte values the chat projector and the
// tool executor receive) followed by a Completion carrying
// finishReason. Used by archetype acceptance tests that need to drive
// "model emits one tool call, turn ends" against the scripted kit
// without standing up a real provider.
//
// finishReason's two documented uses are:
//
//   - ai.FinishReasonToolCalls — the call-issuing half of a two-script
//     tool turn (the harness then re-invokes the provider for the
//     post-tool "model emits final text" turn).
//   - ai.FinishReasonStop — a one-shot scripted turn that happens to
//     include a tool call (rare; mostly future-proofing for the
//     "permission denied — model acknowledges" path).
func ToolCallThenCompletionScript(t *testing.T, block ai.BlockIndex, id, name, arguments string, finishReason ai.FinishReason) Script {
	t.Helper()

	start, err := ai.NewToolCallStart(block, id, name)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart returned %v, want no failure", err)
	}
	delta, err := ai.NewToolCallDelta(block, []byte(arguments))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta returned %v, want no failure", err)
	}
	end, err := ai.NewToolCallEnd(block, []byte(arguments))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(finishReason, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}

	return Script{Steps: []Step{
		Emit(start),
		Emit(delta),
		Emit(end),
		Emit(completion),
	}}
}