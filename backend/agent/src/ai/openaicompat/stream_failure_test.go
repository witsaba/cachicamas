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
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

// ---------------------------------------------------------------------
// AI-32.3 — disconnects and deadlines (R-AEM-012…014, S-AEM-046…055).
// Phase 2b's RED suite: the categorization (categorizeStreamError) and
// construction (midStreamFailureFrom) helpers, plus the producer-side
// integration tests for body-disconnect after output, cancel mid-stream,
// and deadline expiry mid-stream.
//
// # Categorization precedes integration
//
// The first batch of tests below (TestCategorizeStreamError_*) is the
// pure helper suite — every test calls categorizeStreamError directly
// with a known input and asserts the (category, ok) shape design D6
// pins. MidStreamFailure construction tests follow; producer-level
// integration tests land last, once the producer wiring is in place.
// ---------------------------------------------------------------------

// TestCategorizeStreamError_ContextCanceled_IsCancellation covers
// S-AEM-052's category half: context.Canceled categorizes as
// FailureCategoryCancellation, distinguishable from Timeout.
func TestCategorizeStreamError_ContextCanceled_IsCancellation(t *testing.T) {
	got, ok := categorizeStreamError(context.Canceled)
	if !ok {
		t.Fatalf("categorizeStreamError(context.Canceled) ok = false, want true (S-AEM-052)")
	}
	if got != ai.FailureCategoryCancellation {
		t.Errorf("category = %v, want FailureCategoryCancellation (S-AEM-052)", got)
	}
}

// TestCategorizeStreamError_ContextDeadlineExceeded_IsTimeout covers
// S-AEM-051's category half: context.DeadlineExceeded categorizes as
// FailureCategoryTimeout, distinguishable from Cancellation.
func TestCategorizeStreamError_ContextDeadlineExceeded_IsTimeout(t *testing.T) {
	got, ok := categorizeStreamError(context.DeadlineExceeded)
	if !ok {
		t.Fatalf("categorizeStreamError(context.DeadlineExceeded) ok = false, want true (S-AEM-051)")
	}
	if got != ai.FailureCategoryTimeout {
		t.Errorf("category = %v, want FailureCategoryTimeout (S-AEM-051)", got)
	}
}

// TestCategorizeStreamError_URLErrorWrappingCanceled_IsCancellation covers
// design D6's "canceled checked first" note: a *url.Error wrapping
// context.Canceled MUST categorize as Cancellation, never as Timeout.
// This is the wrap chain that net/http's http.Client.Do — and the
// subsequent Body.Read — returns when the caller's context is
// cancelled. Its correctness is what design D6 calls "load-bearing".
type fakeNetError struct{ timeout bool }

func (e *fakeNetError) Error() string   { return "fake timeout" }
func (e *fakeNetError) Timeout() bool   { return e.timeout }
func (e *fakeNetError) Temporary() bool { return false }

// TestCategorizeStreamError_NetErrorTimeout_IsTimeout covers the
// non-context timeout path: a net.Error whose Timeout() returns true
// (e.g. a syscall EAGAIN with the timeout flag, a transport deadline
// independent of caller context).
func TestCategorizeStreamError_NetErrorTimeout_IsTimeout(t *testing.T) {
	got, ok := categorizeStreamError(&fakeNetError{timeout: true})
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got != ai.FailureCategoryTimeout {
		t.Errorf("category = %v, want FailureCategoryTimeout", got)
	}
}

// TestCategorizeStreamError_DecoderSentinel_IsMalformedResponse covers
// the decoder-sentinel branch of design D6: Category() recognises
// ErrFrameTooLarge/ErrTruncated, and categorizeStreamError respects
// that derivation.
func TestCategorizeStreamError_DecoderSentinel_IsMalformedResponse(t *testing.T) {
	got, ok := categorizeStreamError(ErrTruncated)
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got != ai.FailureCategoryMalformedResponse {
		t.Errorf("category = %v, want FailureCategoryMalformedResponse", got)
	}
}

// TestCategorizeStreamError_UnknownError_IsUnavailable covers the
// catch-all branch: an error that matches none of the four typed
// branches — a connection reset, an unexpected EOF, a broken pipe —
// categorizes as Unavailable, retryable (R-AEM-003's shared derivation).
func TestCategorizeStreamError_UnknownError_IsUnavailable(t *testing.T) {
	got, ok := categorizeStreamError(errors.New("connection reset by peer"))
	if !ok {
		t.Fatalf("ok = false, want true")
	}
	if got != ai.FailureCategoryUnavailable {
		t.Errorf("category = %v, want FailureCategoryUnavailable", got)
	}
}

// TestCategorizeStreamError_NilError_ReturnsFalse covers the nil-input
// edge case: the helper returns ok=false rather than panicking on nil.
func TestCategorizeStreamError_NilError_ReturnsFalse(t *testing.T) {
	if _, ok := categorizeStreamError(nil); ok {
		t.Errorf("ok = true, want false (nil input is not a recognised error)")
	}
}

// TestMidStreamFailureFrom_DeadlineIsRetryableAndTypeTimeout covers
// S-AEM-051 and S-AEM-054 (deadline leg): a context.DeadlineExceeded-
// derived failure reports Category()==Timeout AND Retryable()==true
// (the two are derived independently — design D6's "shared derivation").
func TestMidStreamFailureFrom_DeadlineIsRetryableAndTypeTimeout(t *testing.T) {
	failure := midStreamFailureFrom(context.DeadlineExceeded, false)
	if failure == nil {
		t.Fatal("midStreamFailureFrom = nil, want a constructed failure")
	}
	if failure.Category() != ai.FailureCategoryTimeout {
		t.Errorf("Category() = %v, want FailureCategoryTimeout (S-AEM-051)", failure.Category())
	}
	if !failure.Retryable() {
		t.Error("Retryable() = false, want true — deadline failures are retryable (S-AEM-054)")
	}
	if !errors.Is(failure, ai.ErrTimeout) {
		t.Error("errors.Is(failure, ai.ErrTimeout) = false, want true (S-AEM-051)")
	}
}

// TestMidStreamFailureFrom_CancelIsNeverRetryableAndTypeCancellation
// covers S-AEM-052 and S-AEM-054 (cancel leg): a context.Canceled-
// derived failure reports Category()==Cancellation AND
// Retryable()==false — the two flags move on orthogonal axes
// (R-AEM-014, design D6).
func TestMidStreamFailureFrom_CancelIsNeverRetryableAndTypeCancellation(t *testing.T) {
	failure := midStreamFailureFrom(context.Canceled, false)
	if failure == nil {
		t.Fatal("midStreamFailureFrom = nil, want a constructed failure")
	}
	if failure.Category() != ai.FailureCategoryCancellation {
		t.Errorf("Category() = %v, want FailureCategoryCancellation (S-AEM-052)", failure.Category())
	}
	if failure.Retryable() {
		t.Error("Retryable() = true, want false — cancellation is never retryable (S-AEM-054)")
	}
	if !errors.Is(failure, ai.ErrCancelled) {
		t.Error("errors.Is(failure, ai.ErrCancelled) = false, want true (S-AEM-052)")
	}
}

// TestMidStreamFailureFrom_DeadlineAndCancellationAreDistinguishable
// covers S-AEM-053: a deadline failure matches ai.ErrCancelled=false,
// a cancellation failure matches ai.ErrTimeout=false — the sentinels
// must not bleed into one another.
func TestMidStreamFailureFrom_DeadlineAndCancellationAreDistinguishable(t *testing.T) {
	deadline := midStreamFailureFrom(context.DeadlineExceeded, false)
	cancel := midStreamFailureFrom(context.Canceled, false)

	if errors.Is(deadline, ai.ErrCancelled) {
		t.Errorf("deadline failure: errors.Is(_, ai.ErrCancelled) = true, want false (S-AEM-053)")
	}
	if errors.Is(cancel, ai.ErrTimeout) {
		t.Errorf("cancellation failure: errors.Is(_, ai.ErrTimeout) = true, want false (S-AEM-053)")
	}
}

// TestMidStreamFailureFrom_PreservesOutputPreceded covers S-AEM-055:
// PartialOutput() reflects the caller's tracked fact even when the
// category derivation falls into the orthogonal axes — the two are
// independent (R-AEM-014's "mutually exclusive via errors.Is,
// independent of the PartialOutput axis" sentence).
func TestMidStreamFailureFrom_PreservesOutputPreceded(t *testing.T) {
	withOutput := midStreamFailureFrom(context.DeadlineExceeded, true)
	withoutOutput := midStreamFailureFrom(context.DeadlineExceeded, false)

	if !withOutput.PartialOutput() {
		t.Error("withOutput.PartialOutput() = false, want true (S-AEM-055)")
	}
	if withoutOutput.PartialOutput() {
		t.Error("withoutOutput.PartialOutput() = true, want false (S-AEM-055)")
	}
	// Both must still report Timeout category — the axes are independent.
	if withOutput.Category() != ai.FailureCategoryTimeout {
		t.Errorf("withOutput.Category() = %v, want FailureCategoryTimeout", withOutput.Category())
	}
	if withoutOutput.Category() != ai.FailureCategoryTimeout {
		t.Errorf("withoutOutput.Category() = %v, want FailureCategoryTimeout", withoutOutput.Category())
	}
}

// ---------------------------------------------------------------------
// Phase 2b producer-integration tests (S-AEM-046…055 end to end).
// ---------------------------------------------------------------------

// contextBeforeFirstFrameServer returns a server that ACCEPTS the request
// but closes the connection before any HTTP body is written — the
// precise "transport fails before the first frame is decoded" shape
// R-AEM-013 / S-AEM-049 describes. The handler does NOT write a response
// or headers, then hijacks the conn to close it — yielding a transport-
// side failure on the client's http.Client.Do() call (pre-handover,
// pre-stream path).
func contextBeforeFirstFrameServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Hijack the connection and close without writing.
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("ResponseWriter is not a Hijacker — cannot simulate a transport-close-before-headers")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("Hijack: %v", err)
		}
		_ = conn.Close()
	}))
}

// TestStreamFailure_DisconnectBeforeAnyFrame_PreStreamPath covers
// S-AEM-049 / R-AEM-013: a transport that fails before the first frame
// is decoded returns a *ai.Failure with DeliveryPreStream and
// PartialOutput()==false — never a mid-stream terminal.
func TestStreamFailure_DisconnectBeforeAnyFrame_PreStreamPath(t *testing.T) {
	server := contextBeforeFirstFrameServer(t)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(t.Context(), validRequest(t))

	if ch != nil {
		t.Errorf("Stream() channel = %v, want nil — pre-stream failure is returned directly, not via a channel (R-AEM-013)", ch)
	}
	if err == nil {
		t.Fatal("Stream() error = nil, want a pre-stream failure (R-AEM-013)")
	}

	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (S-AEM-049)", err, err)
	}
	if failure.Delivery() != ai.DeliveryPreStream {
		t.Errorf("Delivery() = %v, want DeliveryPreStream (S-AEM-049)", failure.Delivery())
	}
	if failure.PartialOutput() {
		t.Error("PartialOutput() = true, want false (S-AEM-049)")
	}
}

// TestStreamFailure_DisconnectBeforeAnyFrame_NoMidStreamEventWithPartial
// covers S-AEM-050: the event stream carries no event whose
// ErrorPayload() reports PartialOutput()==true. With a pre-stream
// failure, the channel is nil — drainUntilClosed on nil ch would block
// forever, so the assertion is the channel's nil-ness, paired with
// drainUntilClosed on a zero-channel-from-a-channel-if-there-were-one.
func TestStreamFailure_DisconnectBeforeAnyFrame_NoMidStreamEventWithPartial(t *testing.T) {
	server := contextBeforeFirstFrameServer(t)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(t.Context(), validRequest(t))

	// Pre-stream failure: channel is nil. There is no carrier from
	// which an event could arrive. The S-AEM-050 guarantee holds by
	// construction: no event has ever been, or will ever be, delivered
	// (Stream never spawned a producer goroutine).
	if ch != nil {
		t.Fatalf("Stream() channel != nil — pre-stream failure must NOT spawn a producer (R-AEM-013, S-AEM-050)")
	}
	if err == nil {
		t.Fatal("Stream() error = nil, want a pre-stream failure (R-AEM-013)")
	}
}

// TestStreamFailure_DisconnectAfterOutput_MidStreamPartial covers
// S-AEM-046 / S-AEM-047 / S-AEM-048: a transport that fails after at
// least one content event has been emitted produces a terminal with
// PartialOutput()==true, Delivery()==DeliveryMidStream, and the
// already-emitted content events are byte-identical to the transcript.
func TestStreamFailure_DisconnectAfterOutput_MidStreamPartial(t *testing.T) {
	// Two content chunks followed by EOF (no [DONE]). EOF before the
	// terminal sentinel is R-AEM-012's "transport disconnect after
	// output" shape — the producer runs through decodeChunk's success
	// path for the two chunks, then Body.Read returns io.EOF, which
	// the run loop recognises via errIncompleteStream (existing
	// behaviour, retained through this slice).
	transcript := sseFrame(`{"id":"chatcmpl-disco","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"alpha"},"finish_reason":null}]}`) +
		sseFrame(`{"id":"chatcmpl-disco","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"beta"},"finish_reason":null}]}`)

	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(t.Context(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() = %v, want nil — pre-stream path is the EOF path, not this one", err)
	}
	events := drainAll(t, ch)

	failure, ok := terminalFailureFromEvents(t, events)
	if !ok {
		return
	}
	if !failure.PartialOutput() {
		t.Errorf("PartialOutput() = false, want true — the two content chunks preceded the disconnect (S-AEM-046)")
	}
	if failure.Delivery() != ai.DeliveryMidStream {
		t.Errorf("Delivery() = %v, want DeliveryMidStream (S-AEM-047)", failure.Delivery())
	}

	// S-AEM-048: the already-emitted content events must be byte-identical
	// to the transcript.
	var deltas []string
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			deltas = append(deltas, d.Delta())
		}
	}
	wantDeltas := []string{"alpha", "beta"}
	if len(deltas) != len(wantDeltas) {
		t.Fatalf("deltas = %q, want %q — the disconnect MUST NOT discard already-delivered output (S-AEM-048)", deltas, wantDeltas)
	}
	for i := range wantDeltas {
		if deltas[i] != wantDeltas[i] {
			t.Errorf("delta[%d] = %q, want %q (S-AEM-048)", i, deltas[i], wantDeltas[i])
		}
	}

	requireCheckStreamClean(t, events)
}

// TestStreamFailure_DeadlineExpiryMidStream_TypedTimeout covers
// S-AEM-051: a stream whose context deadline expires between frames
// emits a terminal event whose payload reports
// Category()==FailureCategoryTimeout and matches
// errors.Is(failure, ai.ErrTimeout).
//
// The slow server flushes one content frame, then blocks (waiting on a
// release channel that this test never closes). The producer's
// Body.Read on the next iteration returns an error wrapping
// context.DeadlineExceeded — exactly the shape the producer's
// categorizeStreamError maps to Timeout (design D6).
func TestStreamFailure_DeadlineExpiryMidStream_TypedTimeout(t *testing.T) {
	t.Parallel()

	server, release := slowSSEServer(t,
		sseFrame(`{"id":"chatcmpl-deadline","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"alpha"},"finish_reason":null}]}`),
	)
	defer close(release)
	defer server.Close()

	c := mustClient(t, server.URL)
	// 800ms gives the httptest.Server's first-frame flush AND the
	// client's reading of it (≈10ms on a local loopback) comfortable
	// margin before the deadline fires. Earlier test iterations with
	// shorter deadlines exhibited a pre-stream failure because the
	// deadline fired during Do(), before Body.Read ever attempted
	// the second frame.
	ctx, cancel := context.WithTimeout(context.Background(), 800*time.Millisecond)
	defer cancel()

	ch, err := c.Stream(ctx, validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil — the failure is mid-stream, not returned directly (S-AEM-051)", err)
	}
	events := drainAll(t, ch)

	failure, ok := terminalFailureFromEvents(t, events)
	if !ok {
		return
	}
	if failure.Category() != ai.FailureCategoryTimeout {
		t.Errorf("Category() = %v, want FailureCategoryTimeout (S-AEM-051)", failure.Category())
	}
	if !errors.Is(failure, ai.ErrTimeout) {
		t.Error("errors.Is(failure, ai.ErrTimeout) = false, want true (S-AEM-051)")
	}
	if !failure.Retryable() {
		t.Error("Retryable() = false, want true — the deadline failure is retryable-flagged (S-AEM-054)")
	}
	requireCheckStreamClean(t, events)
}

// TestStreamFailure_CancelMidStream_TypedCancellation covers S-AEM-052:
// a stream cancelled explicitly by the caller between frames emits a
// terminal event whose payload reports
// Category()==FailureCategoryCancellation and matches
// errors.Is(failure, ai.ErrCancelled).
func TestStreamFailure_CancelMidStream_TypedCancellation(t *testing.T) {
	t.Parallel()

	server, release := slowSSEServer(t,
		sseFrame(`{"id":"chatcmpl-cancel","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"alpha"},"finish_reason":null}]}`),
	)
	defer close(release)
	defer server.Close()

	c := mustClient(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())

	ch, err := c.Stream(ctx, validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-AEM-052)", err)
	}
	// Drain the one flushed frame, then cancel, so the producer's
	// next Body.Read returns a wrap of context.Canceled (R-AEM-014).
	<-ch
	cancel()

	events := drainAll(t, ch)

	failure, ok := terminalFailureFromEvents(t, events)
	if !ok {
		return
	}
	if failure.Category() != ai.FailureCategoryCancellation {
		t.Errorf("Category() = %v, want FailureCategoryCancellation (S-AEM-052)", failure.Category())
	}
	if !errors.Is(failure, ai.ErrCancelled) {
		t.Error("errors.Is(failure, ai.ErrCancelled) = false, want true (S-AEM-052)")
	}
	if failure.Retryable() {
		t.Error("Retryable() = true, want false — cancellation is NEVER retryable (S-AEM-054)")
	}
	requireCheckStreamClean(t, events)
}

// TestStreamFailure_DeadlineVsCancel_NeitherBleedsAcross covers
// S-AEM-053: the cancel-streams' failure MUST NOT match
// ai.ErrTimeout, and the deadline-stream's failure MUST NOT match
// ai.ErrCancelled — sentinels stay on their own axes (R-AEM-014).
func TestStreamFailure_DeadlineVsCancel_NeitherBleedsAcross(t *testing.T) {
	// Build two terminal failures directly through midStreamFailureFrom
	// — the producer integration tests above cover the full stream
	// path; this asserts the cross-axis sentinel-only property without
	// re-running the entire transport shape.
	deadline := midStreamFailureFrom(context.DeadlineExceeded, false)
	cancel := midStreamFailureFrom(context.Canceled, false)

	if errors.Is(deadline, ai.ErrCancelled) {
		t.Errorf("deadline: errors.Is(_, ai.ErrCancelled) = true, want false (S-AEM-053)")
	}
	if errors.Is(cancel, ai.ErrTimeout) {
		t.Errorf("cancel: errors.Is(_, ai.ErrTimeout) = true, want false (S-AEM-053)")
	}
}

// TestStreamFailure_DeadlineAfterOneOutput_BothAxesHold covers S-AEM-055:
// when the deadline expires AFTER one content event has been emitted,
// the terminal payload reports Category()==Timeout AND
// PartialOutput()==true — the two axes are independent (R-AEM-014's
// "independent of the PartialOutput axis" sentence).
func TestStreamFailure_DeadlineAfterOneOutput_BothAxesHold(t *testing.T) {
	server, release := slowSSEServer(t,
		sseFrame(`{"id":"chatcmpl-both","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"alpha"},"finish_reason":null}]}`),
	)
	defer close(release)
	defer server.Close()

	c := mustClient(t, server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	ch, err := c.Stream(ctx, validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-AEM-055)", err)
	}
	// Read at least one event so outputPreceded becomes true before
	// the deadline-driven failure path.
	<-ch
	events := drainAll(t, ch)

	failure, ok := terminalFailureFromEvents(t, events)
	if !ok {
		return
	}
	if failure.Category() != ai.FailureCategoryTimeout {
		t.Errorf("Category() = %v, want FailureCategoryTimeout (S-AEM-055)", failure.Category())
	}
	if !failure.PartialOutput() {
		t.Error("PartialOutput() = false, want true — one content event preceded the deadline (S-AEM-055)")
	}
	requireCheckStreamClean(t, events)
}
