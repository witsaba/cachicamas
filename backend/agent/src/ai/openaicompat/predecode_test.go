// AI-28.6 — pre-decode response checks: R-ATS-023 (a non-stream content
// type is refused before decoding, with a bounded body excerpt) and
// R-ATS-024 (failure statuses route to AI-32's failure mapping before any
// decode).
//
// Both requirements are pre-decode HTTP behaviors this milestone consumes
// rather than authors (design.md D1, step 6 — strictly before step 7's
// carrier creation): R-ATS-023's excerpt reuses capture.go's own
// captureBody/captureLimit (AI-32.1's bounded-capture machinery, not
// reinvented here); R-ATS-024 wires mapResponse verbatim. This file proves
// the WIRING inside Stream — that a real HTTP round trip through a real
// *Client actually reaches these checks before any byte is handed to the
// decoder — not AI-32.1's own status-to-category table, which
// failure_map_test.go already covers exhaustively at the unit level
// (statusTableCases is reused directly below for S-ATS-093 rather than
// re-derived by hand).
//
// Internal package (package openaicompat), matching every earlier slice's
// own convention: mustClient, validRequest and drainAll are this file's own
// shared helpers, already defined in stream_test.go.
package openaicompat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// contentTypeServer returns a local test server whose handler writes status
// with the given Content-Type header and body verbatim.
func contentTypeServer(t *testing.T, status int, contentType, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", contentType)
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
}

// noContentTypeServer returns a local test server whose handler writes
// status and body verbatim while suppressing Go's own automatic
// Content-Type sniffing — net/http's documented ResponseWriter.Header idiom
// ("To suppress automatic response headers... set their value to nil")
// applies to Content-Type exactly as it does to Date: assigning the map key
// to nil both stops net/http.ResponseWriter.Write's own DetectContentType
// fallback (which triggers only when the Header does not already contain a
// Content-Type key) and, since the value slice is then empty, causes no
// "Content-Type: ..." line to be written to the wire at all. This is the
// only way to simulate a response that truly carries no Content-Type header
// (S-ATS-090), rather than one Go's server would otherwise synthesize from
// the first bytes written.
func noContentTypeServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header()["Content-Type"] = nil
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
}

// ---------------------------------------------------------------------
// R-ATS-023 — a non-stream content type is refused before decoding, with a
// bounded body excerpt.
// ---------------------------------------------------------------------

// TestPreDecode_NonStreamContentType_HTMLErrorPage_RefusedWithBoundedExcerpt
// covers S-ATS-086: a 200 response whose content type is text/html and
// whose body is an HTML error page is refused before decoding, with a
// typed error carrying a bounded excerpt of the page — and, since the
// refusal is pre-handover (no carrier ever returned), trivially zero text
// events were ever emitted.
func TestPreDecode_NonStreamContentType_HTMLErrorPage_RefusedWithBoundedExcerpt(t *testing.T) {
	t.Parallel()

	const marker = "PRXY-GATEWAY-TIMEOUT-9f3c"
	body := "<html><body><h1>504 Gateway Timeout</h1><p>" + marker + "</p></body></html>"
	server := contentTypeServer(t, http.StatusOK, "text/html", body)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))

	if ch != nil {
		t.Error("Stream() returned a non-nil channel for a non-stream content type, want nil — no text event can exist without a carrier (S-ATS-086)")
	}
	if err == nil {
		t.Fatal("Stream() error = nil, want a typed pre-decode refusal (S-ATS-086)")
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (S-ATS-086)", err, err)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATS-086)", failure.Category())
	}
	if failure.Delivery() != ai.DeliveryPreStream {
		t.Errorf("Delivery() = %v, want DeliveryPreStream — refused before the carrier was handed over (S-ATS-086, R-ATS-025)", failure.Delivery())
	}
	var typed *nonStreamContentType
	if !errors.As(err, &typed) {
		t.Fatalf("errors.As(err, &typed) = false, want R-ATS-023's own typed cause reachable through the failure's cause chain (S-ATS-086)")
	}
	if !strings.Contains(typed.Error(), marker) {
		t.Errorf("cause.Error() = %q, want it to contain the page's excerpt marker %q (S-ATS-086)", typed.Error(), marker)
	}
}

// TestPreDecode_StreamContentTypeToleratesCaseAndParameters_AcceptedAndDecodesNormally
// covers S-ATS-087: a 200 response whose content type is
// "TEXT/EVENT-STREAM; charset=utf-8" is accepted and decodes normally — the
// match tolerates case and parameters.
func TestPreDecode_StreamContentTypeToleratesCaseAndParameters_AcceptedAndDecodesNormally(t *testing.T) {
	t.Parallel()

	chunk := `{"id":"chatcmpl-tol","model":"gizmo-tol","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	transcript := "data: " + chunk + "\n\ndata: [DONE]\n\n"
	server := contentTypeServer(t, http.StatusOK, "TEXT/EVENT-STREAM; charset=utf-8", transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil — a case/parameter-varied streaming content type must still be accepted (S-ATS-087)", err)
	}
	if ch == nil {
		t.Fatal("Stream() returned a nil channel for an accepted content type, want non-nil (S-ATS-087)")
	}

	events := drainAll(t, ch)
	if len(events) != 2 {
		t.Fatalf("drained %d event(s), want exactly 2 — normal decode proceeded (S-ATS-087): %+v", len(events), events)
	}
	if events[0].Kind() != ai.EventKindResponseStart {
		t.Errorf("events[0].Kind() = %v, want EventKindResponseStart (S-ATS-087)", events[0].Kind())
	}
	if events[1].Kind() != ai.EventKindCompletion {
		t.Errorf("events[1].Kind() = %v, want EventKindCompletion (S-ATS-087)", events[1].Kind())
	}
}

// TestPreDecode_NonStreamContentTypeHugeBody_ExcerptBoundedByCaptureLimit
// covers S-ATS-088: a body far larger than the documented excerpt bound
// still yields an excerpt no longer than that bound — capture.go's own
// captureLimit, reused rather than a second, independent bound.
func TestPreDecode_NonStreamContentTypeHugeBody_ExcerptBoundedByCaptureLimit(t *testing.T) {
	t.Parallel()

	huge := strings.Repeat("A", captureLimit*4)
	server := contentTypeServer(t, http.StatusOK, "text/html", huge)
	defer server.Close()

	c := mustClient(t, server.URL)
	_, err := c.Stream(context.Background(), validRequest(t))

	if len(huge) <= captureLimit {
		t.Fatalf("test setup defect: body length %d is not far larger than captureLimit %d", len(huge), captureLimit)
	}
	var typed *nonStreamContentType
	if !errors.As(err, &typed) {
		t.Fatalf("errors.As(err, &typed) = false, want R-ATS-023's own typed cause reachable (S-ATS-088)")
	}
	if len(typed.excerpt) != captureLimit {
		t.Errorf("len(excerpt) = %d, want exactly captureLimit (%d) for a body far larger than the bound (S-ATS-088)", len(typed.excerpt), captureLimit)
	}
}

// TestPreDecode_MissingContentTypeHeader_RefusedBeforeDecodingDespiteValidBody
// covers S-ATS-090: a 200 response carrying no content-type header at all
// is refused before decoding with the same typed error, never decoded
// optimistically. The transcript is deliberately a fully valid,
// completable stream — content chunk, terminal chunk, sentinel — so an
// implementation missing this check would decode it all the way to a
// normal Completion, a result unambiguously different from the required
// pre-stream refusal (non-vacuity: an incomplete fixture that could not
// complete either way would not distinguish the two).
func TestPreDecode_MissingContentTypeHeader_RefusedBeforeDecodingDespiteValidBody(t *testing.T) {
	t.Parallel()

	chunk1 := `{"id":"chatcmpl-nc","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"should never decode"},"finish_reason":null}]}`
	chunk2 := `{"id":"chatcmpl-nc","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	transcript := "data: " + chunk1 + "\n\ndata: " + chunk2 + "\n\ndata: [DONE]\n\n"
	server := noContentTypeServer(t, http.StatusOK, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))

	if ch != nil {
		t.Fatal("Stream() returned a non-nil channel for a response with no Content-Type header, want nil — a valid-looking body must not be decoded optimistically (S-ATS-090)")
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (S-ATS-090)", err, err)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATS-090)", failure.Category())
	}
	var typed *nonStreamContentType
	if !errors.As(err, &typed) {
		t.Fatalf("errors.As(err, &typed) = false, want the same R-ATS-023 typed cause a present-but-mismatched content type produces (S-ATS-090)")
	}
}

// ---------------------------------------------------------------------
// R-ATS-024 — failure statuses route to the failure mapping before any
// decode.
// ---------------------------------------------------------------------

// TestPreDecode_FailureStatus429WithJSONErrorBody_ZeroContentEventsBeforeTerminalFailure
// covers S-ATS-091: a 429 response with a JSON error body drains zero
// content events before its terminal failure event — trivially true of a
// pre-handover refusal, since no carrier (and so no content event of any
// kind) exists at all.
func TestPreDecode_FailureStatus429WithJSONErrorBody_ZeroContentEventsBeforeTerminalFailure(t *testing.T) {
	t.Parallel()

	body := `{"error":{"type":"rate_limit_exceeded","message":"slow down","param":null,"code":null}}`
	server := contentTypeServer(t, http.StatusTooManyRequests, "application/json", body)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))

	if ch != nil {
		t.Error("Stream() returned a non-nil channel for a 429 response, want nil (S-ATS-091)")
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (S-ATS-091)", err, err)
	}
	if failure.Category() != ai.FailureCategoryRateLimit {
		t.Errorf("Category() = %v, want FailureCategoryRateLimit — AI-32's own mapping (S-ATS-091)", failure.Category())
	}
	if failure.Delivery() != ai.DeliveryPreStream {
		t.Errorf("Delivery() = %v, want DeliveryPreStream (S-ATS-091, R-ATS-025)", failure.Delivery())
	}
}

// TestPreDecode_FailureStatus500WithValidSSETranscriptBody_NoTextDeltaEmitted
// covers S-ATS-092: a 500 response whose body is a valid SSE transcript —
// carrying a Content-Type the tolerant match would otherwise accept, so
// only the status check can be what refuses it — never emits a text delta
// from that body: the status decides the outcome before decoding. The
// transcript is a fully completable one (non-vacuity, same reasoning as
// S-ATS-090's fixture): an implementation that failed to check status
// before decoding would decode this all the way to a normal Completion
// carrying a real "should never decode" delta, not merely fail for some
// unrelated reason.
func TestPreDecode_FailureStatus500WithValidSSETranscriptBody_NoTextDeltaEmitted(t *testing.T) {
	t.Parallel()

	chunk1 := `{"id":"chatcmpl-500","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"should never decode"},"finish_reason":null}]}`
	chunk2 := `{"id":"chatcmpl-500","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`
	transcript := "data: " + chunk1 + "\n\ndata: " + chunk2 + "\n\ndata: [DONE]\n\n"
	server := contentTypeServer(t, http.StatusInternalServerError, "text/event-stream", transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))

	if ch != nil {
		t.Fatal("Stream() returned a non-nil channel for a 500 response carrying a valid SSE transcript, want nil — the status must decide the outcome before any byte reaches the decoder (S-ATS-092)")
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (S-ATS-092)", err, err)
	}
	if failure.Category() != ai.FailureCategoryUnavailable {
		t.Errorf("Category() = %v, want FailureCategoryUnavailable (S-ATS-092)", failure.Category())
	}
}

// TestPreDecode_FailureStatusTable_EveryRowTerminalFailureNeverCompletion
// covers S-ATS-093: a table of failure statuses across the 4xx and 5xx
// classes each yields a terminal failure whose category comes from AI-32's
// mapping and none yields a completion. statusTableCases is
// failure_map_test.go's own table (R-AEM-002) — reused directly rather than
// re-derived by hand, so this test proves Stream's wiring generalizes
// across the whole status space AI-32.1 already classified, not a
// second, independently maintained copy of the same mapping.
func TestPreDecode_FailureStatusTable_EveryRowTerminalFailureNeverCompletion(t *testing.T) {
	t.Parallel()

	for _, tc := range statusTableCases {
		t.Run(fmt.Sprintf("%d", tc.status), func(t *testing.T) {
			t.Parallel()

			server := contentTypeServer(t, tc.status, "application/json", "{}")
			defer server.Close()

			c := mustClient(t, server.URL)
			ch, err := c.Stream(context.Background(), validRequest(t))

			if ch != nil {
				t.Fatalf("status %d: Stream() returned a non-nil channel, want nil — every row must be a pre-decode terminal failure, never a completion (S-ATS-093)", tc.status)
			}
			var failure *ai.Failure
			if !errors.As(err, &failure) {
				t.Fatalf("status %d: Stream() error = %v (%T), want a *ai.Failure (S-ATS-093)", tc.status, err, err)
			}
			if failure.Category() != tc.wantCategory {
				t.Errorf("status %d: Category() = %v, want %v — from AI-32's own mapping (S-ATS-093)", tc.status, failure.Category(), tc.wantCategory)
			}
		})
	}
}
