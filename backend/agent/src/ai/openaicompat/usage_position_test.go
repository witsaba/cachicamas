// AI-31.3 — never-invent / never-assume-position coverage.
//
// Three coverage tests, all asserting requireCheckStreamClean. Per
// design C1, the stream-check helper is the single shared invocation
// site every drained-stream test in this change calls — a fixture that
// accidentally violates stream protocol fails loudly instead of
// passing on its usage or position assertion alone. The coverage-only
// expectation for this slice (design "Open Questions" + R-ACP-008)
// matches the spec-risk-3 evidence in design.md: usage is captured
// unconditionally ahead of all choice-0 checks, and a usage-bearing
// frame BEFORE choice 0's terminal chunk passes hasRequiredFields,
// is captured, then returns at choice0()'s !hasChoice branch — it
// never reaches the R-ATS-020 row-1/2 window block. buildCompletion
// reads s.usage at the sentinel regardless of when it was set.

package openaicompat

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestUsage_CacheReadPresentCacheWriteAbsent_NoPanic covers S-ACP-021:
// a usage chunk whose prompt_tokens_details carries cached_tokens
// present and cache_write_tokens omitted drains to CacheRead present,
// CacheWrite absent, no zero substituted, and no panic — the
// partial-detail-object shape where one leaf is present and its
// sibling absent. Pointer-on-leaf absent-vs-zero discrimination is
// the mechanism; this test proves it discriminates correctly at
// runtime, not just at compile time.
func TestUsage_CacheReadPresentCacheWriteAbsent_NoPanic(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":50,\"total_tokens\":150,\"prompt_tokens_details\":{\"cached_tokens\":42}}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-021)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.CacheRead.Count(); !ok || v != 42 {
		t.Errorf("CacheRead.Count() = (%d, %v), want (42, true) (S-ACP-021)", v, ok)
	}
	if _, ok := usage.CacheWrite.Count(); ok {
		t.Error("CacheWrite.Count() ok = true, want false — sibling leaf was omitted, must stay absent, not be substituted zero (S-ACP-021)")
	}
}

// TestUsage_OddPositionFrame_BeforeTerminalChunk covers S-ACP-022: a
// transcript whose populated usage frame arrives BEFORE choice 0's
// terminal chunk rather than after it. Position is not assumed; the
// adapter's usage capture (stream_state.go's applyChunk) runs
// unconditionally, ahead of all choice-0 checks, so the early usage
// frame's counts reach the eventual completion.
//
// Fixture-authoring preconditions (design G3, validator-proved for
// every odd-position usage frame): the early frame MUST carry a
// non-empty model (else hasRequiredFields fails → errMissingRequiredField)
// and the SAME id as the established identity (else
// errSecondResponseStart). The terminal chunk's identity-establishing
// role means the early frame's id can match without colliding; this
// transcript uses identical "chatcmpl-u" / "m" / 1700000000 across
// every chunk.
func TestUsage_OddPositionFrame_BeforeTerminalChunk(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":99,\"completion_tokens\":7,\"total_tokens\":106}}\n\n" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-022 — odd-position usage frame must complete cleanly)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 99 {
		t.Errorf("Input.Count() = (%d, %v), want (99, true) — the early usage frame's counts must reach the completion (S-ACP-022)", v, ok)
	}
	if v, ok := usage.Output.Count(); !ok || v != 7 {
		t.Errorf("Output.Count() = (%d, %v), want (7, true) (S-ACP-022)", v, ok)
	}
}

// drainLastCompletionFromEvents is the variant of drainLastCompletion
// that takes already-drained events, used by tests that call
// requireCheckStreamClean on the same events slice and would otherwise
// drain the channel twice (the helper drainLastCompletion already
// drains). Defined here rather than in usage_test.go to keep this
// file's helper self-contained.
func drainLastCompletionFromEvents(t *testing.T, events []ai.Event) ai.Completion {
	t.Helper()
	if len(events) == 0 {
		t.Fatal("drained zero events, want a completion")
	}
	completion, ok := events[len(events)-1].Completion()
	if !ok {
		t.Fatalf("last event carries no Completion payload: %+v", events[len(events)-1])
	}
	return completion
}

// TestUsage_MetadataOnlyFrame_ZeroContentEvents covers S-ACP-023: a
// metadata-only frame (empty choices array, no delta) drains to zero
// content events, no protocol-order violation from ai.CheckStream, and
// the completion carries its usage counts. The explicit
// requireCheckStreamClean assertion IS the scenario — S-ACP-023's
// "the stream check reports no protocol violation" is implemented as
// the helper call, not as prose.
//
// A second content chunk with a finish_reason completes the transcript
// to a normal completion (otherwise the stream would be
// incomplete-stream flagged, separate from the S-ACP-023 claim).
func TestUsage_MetadataOnlyFrame_ZeroContentEvents(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":13,\"completion_tokens\":5,\"total_tokens\":18}}\n\n" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-023)", err)
	}
	events := drainAll(t, ch)

	textCount := 0
	for _, ev := range events {
		if ev.Kind() == ai.EventKindTextDelta {
			textCount++
		}
	}
	if textCount != 0 {
		t.Errorf("text delta count = %d, want 0 — metadata-only frame must derive no content events (S-ACP-023)", textCount)
	}
	requireCheckStreamClean(t, events)
	usage := drainLastCompletionFromEvents(t, events).Usage()
	if v, ok := usage.Input.Count(); !ok || v != 13 {
		t.Errorf("Input.Count() = (%d, %v), want (13, true) — completion must carry the metadata frame's counts (S-ACP-023)", v, ok)
	}
}
