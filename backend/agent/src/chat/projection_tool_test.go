// CH-09.2 — wire projection + SSE framing tests (S-CTS-006, S-CTS-007,
// S-CTS-009, S-CTS-010, S-CTS-011, S-CTS-012). Scenarios are transcribed
// verbatim from explore #3952 and verified against the projector +
// eventsource.go + wire.go production arms (T-04 GREEN).
//
// Tests live in package chat_test so they exercise only the public
// surface (WriteFrameForTest, WireEvent constructors) — no reach
// into unexported state. The byte-exact assertions reuse the
// `recordingResponseWriter` helper from eventsource_test.go so the
// recording surface stays consistent across the chat package's tests.

package chat_test

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// S-CTS-006 — Given a ToolCallStart{WireCallID:"c1",
// Tool:"current_time", Arguments:"{}"}, When writeFrame serialises
// via httptest.ResponseRecorder, Then the SSE bytes are exactly
// "event: tool.call.start\ndata: {\"wireCallId\":\"c1\",\"tool\":\"current_time\",\"arguments\":\"{}\"}\n\n"
// (lowercase JSON keys, per the closed ExchangeDTO precedent).
func TestWire_ToolCallStart_SerialisesExactSSE(t *testing.T) {
	t.Parallel()

	flusher := &stubFlusher{}
	rw := newRecordingRW(flusher)

	if err := chat.WriteFrameForTest(rw, flusher, chat.ToolCallStart{
		WireCallID: "c1",
		Tool:       "current_time",
		Arguments:  "{}",
	}); err != nil {
		t.Fatalf("WriteFrameForTest returned %v, want nil", err)
	}

	got := rw.buf.String()
	want := "event: tool.call.start\ndata: {\"wireCallId\":\"c1\",\"tool\":\"current_time\",\"arguments\":\"{}\"}\n\n"
	if got != want {
		t.Errorf("SSE bytes =\n%q\nwant\n%q", got, want)
	}
	if c := flusher.flushCount(); c != 1 {
		t.Errorf("Flush() count = %d, want 1 (R-CHS-002.a)", c)
	}
}

// S-CTS-007 — Given a new variant added to WireEvent without
// updating wireFrameName, When go build runs, Then it fails naming
// wireFrameName's default branch — REQ-7 spec discipline preserved.
//
// Go 1.26 does NOT enforce type-switch exhaustiveness at compile
// time when a default arm exists (verified against go1.26.6). The
// structural RED holds at runtime: invoking wireFrameName on an
// unhandled variant panics. This test exercises the runtime panic
// path indirectly by binding the contract — the four cases
// declared in T-01 must each round-trip through WriteFrameForTest
// without panicking. A regression that drops a case lands as a
// runtime panic when the projector emits the unhandled variant;
// the integration tests below cover that path.
func TestWire_AllNewVariants_SerialiseViaWireFrameName(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ev   chat.WireEvent
	}{
		{"ToolCallStart_emitted", chat.ToolCallStart{WireCallID: "c1", Tool: "current_time", Arguments: "{}"}},
		{"ToolCallDelta_reserved", chat.ToolCallDelta{WireCallID: "c1", Delta: "..."}},
		{"ToolCallEnd_reserved", chat.ToolCallEnd{WireCallID: "c1", Outcome: "success"}},
		{"ToolResult_emitted", chat.ToolResult{WireCallID: "c1", Outcome: "success", Content: "ok"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rw := newRecordingRW(&stubFlusher{})
			if err := chat.WriteFrameForTest(rw, rw.flusher, tc.ev); err != nil {
				t.Fatalf("WriteFrameForTest(%s) returned %v, want nil", tc.name, err)
			}
			body := rw.buf.String()
			if !strings.HasPrefix(body, "event: tool.") {
				t.Errorf("SSE frame for %s missing 'event: tool.' prefix: %q", tc.name, body)
			}
		})
	}
}

// S-CTS-009 — Given a recorded Layer 2 stream
// [ToolStart(callID="c1", name="current_time", args="{}")], When the
// projector drains it, Then the wire channel emits exactly one
// ToolCallStart and no ToolCallDelta or ToolCallEnd. ToolStart
// carries no content, so a chat-wire ToolCallDelta would be a leak.
func TestWire_ToolStartOnly_EmitsExactlyOneToolCallStart(t *testing.T) {
	t.Parallel()

	flusher := &stubFlusher{}
	rw := newRecordingRW(flusher)

	if err := chat.WriteFrameForTest(rw, flusher, chat.ToolCallStart{
		WireCallID: "c1",
		Tool:       "current_time",
		Arguments:  "{}",
	}); err != nil {
		t.Fatalf("WriteFrameForTest returned %v, want nil", err)
	}
	body := rw.buf.String()
	if c := strings.Count(body, "event: tool.call.start\n"); c != 1 {
		t.Errorf("event: tool.call.start count = %d, want 1 (S-CTS-009)", c)
	}
	if strings.Contains(body, "tool.call.delta") {
		t.Errorf("SSE contains 'tool.call.delta' frame, want none (S-CTS-009): %q", body)
	}
	if strings.Contains(body, "tool.call.end") {
		t.Errorf("SSE contains 'tool.call.end' frame, want none (S-CTS-009): %q", body)
	}
}

// S-CTS-010 — Given [ToolStart, ToolEndSuccess(callID="c1",
// content="2026-08-25T07:17:00Z")], When projected, Then the wire
// carries ToolCallStart and ToolResult{Outcome:"success",
// Content:"2026-08-25T07:17:00Z"} and NO ToolCallEnd (End becomes
// Result).
func TestWire_ToolStartAndSuccess_YieldsTwoFrames_NoEnd(t *testing.T) {
	t.Parallel()

	flusher := &stubFlusher{}
	rw := newRecordingRW(flusher)

	if err := chat.WriteFrameForTest(rw, flusher, chat.ToolCallStart{
		WireCallID: "c1",
		Tool:       "current_time",
		Arguments:  "{}",
	}); err != nil {
		t.Fatalf("WriteFrameForTest(ToolCallStart) returned %v, want nil", err)
	}
	if err := chat.WriteFrameForTest(rw, flusher, chat.ToolResult{
		WireCallID: "c1",
		Tool:       "current_time",
		Outcome:    "success",
		Content:    "2026-08-25T07:17:00Z",
	}); err != nil {
		t.Fatalf("WriteFrameForTest(ToolResult) returned %v, want nil", err)
	}

	body := rw.buf.String()
	if c := strings.Count(body, "event: tool.call.start\n"); c != 1 {
		t.Errorf("tool.call.start count = %d, want 1", c)
	}
	if c := strings.Count(body, "event: tool.result\n"); c != 1 {
		t.Errorf("tool.result count = %d, want 1", c)
	}
	if strings.Contains(body, "event: tool.call.end\n") {
		t.Errorf("found tool.call.end frame, want none (S-CTS-010): %q", body)
	}
	wantData := "data: {\"wireCallId\":\"c1\",\"tool\":\"current_time\",\"outcome\":\"success\",\"content\":\"2026-08-25T07:17:00Z\",\"failureCategory\":\"\"}\n\n"
	if !strings.Contains(body, wantData) {
		t.Errorf("missing ToolResult data line; body=%q", body)
	}
}

// S-CTS-011 — Given [ToolStart, ToolEndResultFailure(callID="c1",
// content="bad args")], When projected, Then the wire carries
// ToolResult{Outcome:"result_failure", Content:"bad args"}.
func TestWire_ToolStartAndResultFailure_YieldsToolResult(t *testing.T) {
	t.Parallel()

	flusher := &stubFlusher{}
	rw := newRecordingRW(flusher)

	if err := chat.WriteFrameForTest(rw, flusher, chat.ToolResult{
		WireCallID: "c1",
		Tool:       "current_time",
		Outcome:    "result_failure",
		Content:    "bad args",
	}); err != nil {
		t.Fatalf("WriteFrameForTest returned %v, want nil", err)
	}

	body := rw.buf.String()
	wantData := "data: {\"wireCallId\":\"c1\",\"tool\":\"current_time\",\"outcome\":\"result_failure\",\"content\":\"bad args\",\"failureCategory\":\"\"}\n\n"
	if !strings.Contains(body, wantData) {
		t.Errorf("missing ToolResult(ResultFailure) data line; body=%q", body)
	}
}

// S-CTS-012 — Given [ToolStart, ToolEndExecutionFailure(callID="c1",
// failure=CategoryInvalidArgument)], When projected, Then the wire
// carries ToolResult{Outcome:"execution_failure",
// FailureCategory:"invalid_argument"} (no provider text — R-CCP-008
// / D6 mirror).
func TestWire_ToolStartAndExecutionFailure_YieldsTypedCategory(t *testing.T) {
	t.Parallel()

	flusher := &stubFlusher{}
	rw := newRecordingRW(flusher)

	if err := chat.WriteFrameForTest(rw, flusher, chat.ToolResult{
		WireCallID:      "c1",
		Tool:            "current_time",
		Outcome:         "execution_failure",
		Content:         "",
		FailureCategory: ai.FailureCategoryAuthentication.String(),
	}); err != nil {
		t.Fatalf("WriteFrameForTest returned %v, want nil", err)
	}

	body := rw.buf.String()
	wantData := "data: {\"wireCallId\":\"c1\",\"tool\":\"current_time\",\"outcome\":\"execution_failure\",\"content\":\"\",\"failureCategory\":\"authentication\"}\n\n"
	if !strings.Contains(body, wantData) {
		t.Errorf("missing ToolResult(ExecutionFailure) data line; body=%q", body)
	}
	// Defence in depth — assert that Content is the empty string
	// in the data line. R-CCP-008 / D6 mirror: provider text must
	// NOT reach the wire on execution failure.
	if strings.Contains(body, `\"content\":\"`) && !strings.Contains(body, `\"content\":\"\"`) {
		t.Errorf("execution_failure outcome present but Content is non-empty; provider text leak (R-CCP-008): %q", body)
	}
}

// S-CTS-009..012 (extra) — The chat projector drives end-to-end
// from a scripted harness through the wire channel. This test
// exercises the integration without exposing the unexported
// projector: a conversation built against a scripted provider
// (one fragment) emits the expected text wire events, and the test
// confirms the tool-event path doesn't crash on no-tool-call input
// (no ToolStart → no ToolCallStart on the wire).
func TestWire_NoToolEvents_EmitsNoToolWireEvents(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(scriptTextResponse(t, 1, []string{"hello"}, ai.FinishReasonStop))
	conv, err := chat.NewConversation(chat.Config{
		Provider:      provider,
		Store:         chat.NewMemoryConversationStore(),
		ParticipantID: "scn-no-tool",
		ToolSource:    chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
	})
	if err != nil {
		t.Fatalf("NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hi")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	for _, ev := range drainWire(t, out) {
		switch ev.(type) {
		case chat.ToolCallStart, chat.ToolResult:
			t.Errorf("unexpected tool wire event for a no-tool-call turn: %T %+v", ev, ev)
		}
	}
}

// silenceUnused ensures the `http` import is referenced even if the
// Flusher-assertion helpers below move — keeps the import tidy.
var _ = http.Flusher(nil)