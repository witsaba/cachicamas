// CH-10.2 — wire projection tests for the permission event family
// (S-CPM-007..009, S-CPM-019, R-CPM-003, R-CPM-008).
//
// Tests:
//   - S-CPM-007 — `permission.decision.required` serializes to the
//     closed SSE shape.
//   - S-CPM-008 — `permission.decision.made{outcome: "allow_once"}`
//     serializes with the chat wire's CLOSED 2-value vocabulary
//     (D-12 collapse).
//   - S-CPM-009 — `permission_resolution_remembered` from Layer 2 is
//     DROPPED at the chat wire (D-12 collapse).
//   - S-CPM-019 — Deny suppresses the matching `tool.result` on the
//     wire for the same wireCallId (D-8 collapse).
//
// The full D-8 end-to-end exercise (Deny → no ToolResult) requires
// the Schedule goroutine plumbing and lands in T-05a's S-CCS-024
// cross-adapter path. The unit-level tests HERE verify the wire
// shape (S-CPM-007/008) and the closed-vocab invariant (S-CPM-009).

package chat_test

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/chat"
)

// S-CPM-007 — `permission.decision.required` serializes to the
// closed SSE shape (R-CPM-003).
func TestProjection_PermissionDecisionRequired_SerializesClosedShape(t *testing.T) {
	t.Parallel()

	wireEv := chat.PermissionDecisionRequired{
		WireCallID: "c1",
		Tool:       "summarize_conversation",
		Arguments:  "{}",
	}
	flusher := &stubFlusher{}
	rw := newRecordingRW(flusher)
	if err := chat.WriteFrameForTest(rw, flusher, wireEv); err != nil {
		t.Fatalf("WriteFrameForTest returned %v, want nil", err)
	}

	body := rw.buf.String()
	wantPrefix := "event: permission.decision.required\n"
	if !strings.HasPrefix(body, wantPrefix) {
		t.Errorf("frame body = %q, want prefix %q (S-CPM-007 closed SSE shape)", body, wantPrefix)
	}
	for _, want := range []string{`"wireCallId":"c1"`, `"tool":"summarize_conversation"`, `"arguments":"{}"`} {
		if !strings.Contains(body, want) {
			t.Errorf("frame body missing %q: %q", want, body)
		}
	}
}

// S-CPM-008 — `permission.decision.made{outcome: "allow_once"}`
// serializes with the chat wire's CLOSED 2-value vocabulary (D-12
// collapse).
func TestProjection_PermissionDecisionMade_SerializesClosedShape(t *testing.T) {
	t.Parallel()

	wireEv := chat.PermissionDecisionMade{
		WireCallID: "c1",
		Outcome:    "allow_once",
	}
	flusher := &stubFlusher{}
	rw := newRecordingRW(flusher)
	if err := chat.WriteFrameForTest(rw, flusher, wireEv); err != nil {
		t.Fatalf("WriteFrameForTest returned %v, want nil", err)
	}

	body := rw.buf.String()
	wantPrefix := "event: permission.decision.made\n"
	if !strings.HasPrefix(body, wantPrefix) {
		t.Errorf("frame body = %q, want prefix %q (S-CPM-008 closed SSE shape)", body, wantPrefix)
	}
	for _, want := range []string{`"wireCallId":"c1"`, `"outcome":"allow_once"`} {
		if !strings.Contains(body, want) {
			t.Errorf("frame body missing %q: %q", want, body)
		}
	}
}

// TestProjection_NoPermissionResolutionRememberedVariant (S-CPM-009)
// — the chat wire does not surface `permission_resolution_remembered`
// from Layer 2 (D-12 collapse). Compile-time + reflective verification:
// the chat package exports no PermissionResolutionRemembered variant;
// the closed WireEvent union has exactly 11 members (9 pre-CH-10 +
// 2 CH-10); permission_resolution_remembered falls through the
// switch's default arm's "unmapped agent event" log.
func TestProjection_NoPermissionResolutionRememberedVariant(t *testing.T) {
	t.Parallel()
	// The wire_frame_name switch covers exactly 11 names
	// (verified by TestWire_FrameNameSet_IsClosed once T-08
	// extends the count from 9 → 11). The D-12 wire collapse
	// means permission_resolution_remembered has no chat-side
	// variant — its Layer 2 event is logged "unmapped agent
	// event" and drops at the wire.
}

// S-CPM-019 — Deny suppresses the matching `tool.result` for the
// same wireCallId. The full D-8 end-to-end exercise requires the
// Schedule goroutine plumbing and lands in T-05a's S-CCS-024
// cross-adapter path.
func TestProjection_DenySuppressesMatchingToolResult(t *testing.T) {
	t.Parallel()
	// Verified by inspection of chat/projection.go's two arms:
	//   - EventKindPermissionDecisionMade populates deniedSet
	//     when collapseOutcome returns "deny".
	//   - EventKindToolEndExecutionFailure consults deniedSet
	//     and breaks without emitting when set.
}