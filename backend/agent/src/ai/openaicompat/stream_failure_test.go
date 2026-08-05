// AI-32.2 / AI-32.3 — the stream-failure path's RED suite (S-AEM-040…055).
//
// Phase 2a's RED cases (S-AEM-040…044) cover R-AEM-010/011: an in-band
// error frame terminates the stream with a typed mid-stream failure, the
// vendor label survives opaque as RawLabel, no events follow the
// terminal, and the in-band identity is distinguishable from a
// transport failure through errors.Is — never by message-text
// inspection. Phase 2b's cases (S-AEM-046…055) land in subsequent
// batches.
//
// Internal package (package openaicompat): these tests reference
// unexported construction sites (failureFromErrorFrame) and the package's
// own ErrInBandErrorFrame alongside the exported Failure accessors, the
// same package-internal pattern stream_test.go / stream_state_test.go
// already use. credential_scan_test.go's external-package scope is
// deliberately untouched (design.md D8 / S-ATS-089).

package openaicompat

import (
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// sseFrame is the on-wire bytes of one server-sent-event — a single
// `data:`-prefixed JSON object terminated by a blank line — assembled
// here so the transcript helpers stay readable at a glance.
func sseFrame(payload string) string {
	if !strings.HasSuffix(payload, "\n\n") {
		return "data: " + payload + "\n\n"
	}
	return "data: " + payload
}

// streamWithTranscript starts a Client.Stream over sseServer(transcript),
// drains the channel to completion, and returns the events collected.
func streamWithTranscript(t *testing.T, transcript string) []ai.Event {
	t.Helper()
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(t.Context(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() = %v, want nil", err)
	}
	return drainAll(t, ch)
}

// terminalFailureFromEvents returns the last event's error payload and
// whether it carries one. Fails the test if the terminal is not an error
// event — every S-AEM-040..055 case asserts an ErrorPayload() shape.
func terminalFailureFromEvents(t *testing.T, events []ai.Event) (*ai.Failure, bool) {
	t.Helper()
	if len(events) == 0 {
		t.Fatalf("drained zero events, want a terminal error event")
	}
	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event kind = %v, want the terminal error event", last.Kind())
	}
	return failure, ok
}

// TestStreamFailure_InBandFrameAfterTwoContent_TerminatesWithPartialOutput
// covers S-AEM-040: two content frames followed by an in-band error frame
// produce a terminal event whose payload reports PartialOutput()==true.
func TestStreamFailure_InBandFrameAfterTwoContent_TerminatesWithPartialOutput(t *testing.T) {
	transcript := sseFrame(`{"id":"chatcmpl-x","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`) +
		sseFrame(`{"id":"chatcmpl-x","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`) +
		sseFrame(`{"error":{"type":"vendor_stream_fault","message":"midstream broken","param":null,"code":"sf_42"}}`)

	events := streamWithTranscript(t, transcript)

	failure, ok := terminalFailureFromEvents(t, events)
	if !ok {
		return
	}
	if !failure.PartialOutput() {
		t.Errorf("PartialOutput() = false, want true after two content frames preceded the in-band error (S-AEM-040)")
	}
	requireCheckStreamClean(t, events)
}

// TestStreamFailure_InBandFrameTerminalCountsZeroEventsAfter covers
// S-AEM-041: after the terminal error event no further event lands on
// the carrier. drainAll drains the channel to its natural close; the
// resulting slice ending at the terminal event is the assertion that
// nothing followed it.
func TestStreamFailure_InBandFrameTerminalCountsZeroEventsAfter(t *testing.T) {
	transcript := sseFrame(`{"id":"chatcmpl-x","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}]}`) +
		sseFrame(`{"error":{"type":"vendor_stream_fault","message":"stop","param":null,"code":"sf_7"}}`)

	events := streamWithTranscript(t, transcript)

	// The drained slice's last event MUST be the terminal. If anything
	// followed the terminal on the carrier it would land after this
	// index — drainAll's natural-close bound is what makes this
	// assertion honest (a hang would surface as a drainAll timeout,
	// not as a false-positive tail count).
	terminalIdx := -1
	for i, ev := range events {
		if _, ok := ev.ErrorPayload(); ok {
			terminalIdx = i
		}
	}
	if terminalIdx == -1 {
		t.Fatalf("no terminal error event in the drained slice (S-AEM-041)")
	}
	if after := len(events) - terminalIdx - 1; after != 0 {
		t.Errorf("events after the terminal = %d, want zero — the carrier had more to send (S-AEM-041): %v", after, events[terminalIdx+1:])
	}
}

// TestStreamFailure_InBandFrameVendorLabel_SurvivesAsRawLabel covers
// S-AEM-042: the in-band error frame's vendor label (`type`) survives
// as the failure's RawLabel(), byte-exact.
func TestStreamFailure_InBandFrameVendorLabel_SurvivesAsRawLabel(t *testing.T) {
	transcript := sseFrame(`{"error":{"type":"vendor_stream_fault","message":"text","param":null,"code":"code_value"}}`)

	events := streamWithTranscript(t, transcript)

	failure, ok := terminalFailureFromEvents(t, events)
	if !ok {
		return
	}
	if got := failure.RawLabel(); got != "vendor_stream_fault" {
		t.Errorf("RawLabel() = %q, want %q (S-AEM-042)", got, "vendor_stream_fault")
	}
}

// TestStreamFailure_InBandVsTruncatedStream_AreErrorsIsDistinguishable
// covers S-AEM-043: one stream terminated by an in-band error frame and
// one terminated by a stream truncation (an EOF before [DONE], not in
// itself an in-band error) are distinguishable through errors.Is(err,
// ErrInBandErrorFrame). The truncation path goes through the producer's
// own errIncompleteStream cause, which carries ai.ErrMalformedResponse —
// never ErrInBandErrorFrame — so this is the right "transport-style
// termination" pair before phase 2b adds context/net.Error coverage.
func TestStreamFailure_InBandVsTruncatedStream_AreErrorsIsDistinguishable(t *testing.T) {
	// In-band path: a single in-band error frame mid-transcript.
	inBandTranscript := sseFrame(`{"error":{"type":"vendor_stream_fault","message":"text","param":null,"code":"x"}}`)
	inBandEvents := streamWithTranscript(t, inBandTranscript)
	inBandFailure, ok := terminalFailureFromEvents(t, inBandEvents)
	if !ok {
		return
	}
	if !errors.Is(inBandFailure, ErrInBandErrorFrame) {
		t.Errorf("in-band failure: errors.Is(_, ErrInBandErrorFrame) = false, want true (S-AEM-043)")
	}

	// Truncation path: a content frame, then EOF before [DONE].
	truncatedTranscript := sseFrame(`{"id":"chatcmpl-y","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}]}`)
	truncatedEvents := streamWithTranscript(t, truncatedTranscript)
	truncatedFailure, ok := terminalFailureFromEvents(t, truncatedEvents)
	if !ok {
		return
	}
	if errors.Is(truncatedFailure, ErrInBandErrorFrame) {
		t.Errorf("truncated-stream failure: errors.Is(_, ErrInBandErrorFrame) = true, want false — in-band identity MUST NOT bleed into transport-style terminations (S-AEM-043)")
	}
}

// TestStreamFailure_CauseChainReachesErrInBandErrorFrame covers the
// errors.Is leg of S-AEM-043 in isolation: errors.Unwrap on the failure
// traverses the cause chain to ErrInBandErrorFrame and that identity is
// what the matcher consults. This is a triangulation case — the
// preceding test exercises the same property at the channel-event level,
// this one asserts the chain shape directly.
func TestStreamFailure_CauseChainReachesErrInBandErrorFrame(t *testing.T) {
	transcript := sseFrame(`{"error":{"type":"x","message":"y","param":null,"code":"z"}}`)

	events := streamWithTranscript(t, transcript)
	failure, ok := terminalFailureFromEvents(t, events)
	if !ok {
		return
	}

	var found *inBandErrorFrame
	if !errors.As(failure, &found) {
		t.Fatalf("errors.As(failure, &*inBandErrorFrame) = false, want true — the cause chain must carry the in-band frame cause directly")
	}
	if found.Unwrap() != ErrInBandErrorFrame {
		t.Errorf("cause.Unwrap() = %v, want ErrInBandErrorFrame (S-AEM-043)", found.Unwrap())
	}
}
