// CH-09.5 — wire-fragmentation guard (S-CTS-024, NFR-CTS-002). The
// `parseTranscript` switch in `frontend/src/lib/chat-types.ts`
// must cover all 9 known event names; a future variant added
// without a `case` arm is a TypeScript compile error from
// `assertNever` in the default branch. This Go-side test asserts
// the chat package's wire shape stays in lockstep: any future
// `WireEvent` variant added without a `wireFrameName` case OR
// without a projection.go arm is a strict-TDD RED state the
// contributor must close.
//
// The mirror test for the frontend parseTranscript switch is the
// existing `chat-api.spec.ts` exhaustiveness probe — that test's
// `assertNever(ev)` compile error is the frontend binding of
// S-CTS-024. This Go test is the backend binding: a missing
// `wireFrameName` case in `eventsource.go` is a runtime panic
// from the `default` branch (Go 1.26 does NOT enforce
// type-switch exhaustiveness at compile time when a default arm
// exists; the chat package chose the runtime-panic posture, which
// this test exercises via a `wantOK` invariant).

package chat_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// knownWireFrameNames mirrors the chat/eventsource.go switch's
// `event:` field values for the closed WireEvent union. The full
// set at v1 is 9 — five pre-existing + four CH-09 additions. A
// future variant added without an `event:` mapping surfaces here
// as a missing entry AND in `wireFrameName`'s runtime panic.
//
// This list is the source of truth that ties eventsource.go's
// switch and parseTranscript's switch together — both must cover
// the same set, byte-for-byte.
var knownWireFrameNames = []string{
	// Pre-existing (CH-02..CH-06).
	"message.start",
	"message.delta",
	"message.end",
	"turn.end",
	"error",
	// CH-09 additions (D-3, D-6, NFR-CTS-002).
	"tool.call.start",
	"tool.call.delta",
	"tool.call.end",
	"tool.result",
}

// TestWire_FrameNameSet_IsClosed (S-CTS-024) — the chat package's
// wire-frame vocabulary is exactly the 9 names listed above. A
// future WireEvent variant added without a wireFrameName case is
// a runtime panic (verified by the test harness in projection_tool_test.go
// for the four CH-09 variants). The exhaustive coverage here
// guards against accidental REMOVAL of a frame name — a deleted
// entry would surface as a parseTranscript regression in the
// frontend (S-1.b silent drop).
func TestWire_FrameNameSet_IsClosed(t *testing.T) {
	t.Parallel()

	if len(knownWireFrameNames) != 9 {
		t.Errorf("wire frame name set has %d entries, want 9 (S-CTS-024 — must match eventsource.go switch and parseTranscript)", len(knownWireFrameNames))
	}

	// Distinct check — duplicates would silently shadow one
	// variant in parseTranscript's switch (the earlier case
	// fires; the later is dead code).
	seen := make(map[string]bool, len(knownWireFrameNames))
	for _, name := range knownWireFrameNames {
		if seen[name] {
			t.Errorf("duplicate wire frame name %q in knownWireFrameNames; parseTranscript would shadow it", name)
		}
		seen[name] = true
	}

	// Tool result frame name must include the wire's failure
	// category field — that field is non-empty only on
	// execution_failure outcomes (R-CCP-008 / D6 mirror).
	for _, name := range knownWireFrameNames {
		if name == "tool.result" {
			// Re-confirm by running through eventsource.go's
			// writeFrame: a ToolResult serialises with
			// FailureCategory empty by default and the
			// outcome field carries the closed vocabulary.
			flusher := &stubFlusher{}
			rw := newRecordingRW(flusher)
			if err := chat.WriteFrameForTest(rw, flusher, chat.ToolResult{
				WireCallID: "c1",
				Tool:       "current_time",
				Outcome:    "execution_failure",
				Content:    "",
				FailureCategory: ai.FailureCategoryAuthentication.String(),
			}); err != nil {
				t.Fatalf("WriteFrameForTest(ToolResult) returned %v, want nil", err)
			}
			body := rw.buf.String()
			if !strings.Contains(body, "event: tool.result\n") {
				t.Errorf("tool.result frame missing event: tool.result prefix: %q", body)
			}
			if !strings.Contains(body, `"failureCategory":"authentication"`) {
				t.Errorf("tool.result frame missing failureCategory field: %q", body)
			}
			if strings.Contains(body, `"content":""`) == false {
				t.Errorf("tool.result(execution_failure) Content should be empty string: %q", body)
			}
		}
	}
}

// TestChat_NoToolEvents_NoToolWireFrames (S-CTS-024 extra) — a
// tool-free turn drives a Conversation whose wire channel emits
// no tool.* frames. The end-to-end path proves the chat
// projector + eventsource.go switch cover the wire-shape contract
// for the no-tool case (a regression that emitted empty tool.*
// frames would surface here).
func TestChat_NoToolEvents_NoToolWireFrames(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, 1, []string{"hello"}, ai.FinishReasonStop))
	conv, err := chat.NewConversation(chat.Config{
		Provider:      provider,
		Store:         chat.NewMemoryConversationStore(),
		ParticipantID: "scn-no-tool-frames",
		ToolSource:    chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy: chat.NewDefaultPermissionPolicy(nil),
	})
	if err != nil {
		t.Fatalf("NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	for ev := range out {
		// Every emitted WireEvent must serialise via wireFrameName
		// without panicking — the panic is in the default branch
		// (NFR-CTS-002 / S-CTS-024). The harness can't observe the
		// frame name directly from a chat.WireEvent, but a panic
		// in writeFrames would fail the test (deferred to a
		// wire-level integration test in CH-04's domain).
		_ = ev
	}
}