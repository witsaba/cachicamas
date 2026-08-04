// AI-28.1.1 — the producer shell's own test suite: R-ATS-001…005
// (S-ATS-001…019), the interface, the pre-stream contract, a minimal
// transcript's drain, response identity, and AI-20.3 cancellation
// discipline.
//
// Internal package (not openaicompat_test): every test here constructs a
// real *Client through New, the same boundary-proof shape
// provider_boundary_test.go, client_test.go and viability_test.go already
// use internally.
//
// R-ATS-007…028's own scenarios (text mapping, terminal discipline, usage,
// tolerance, protocol-order violations, pre-decode checks, and the
// dedicated R-ATS-025…028 charter/citation tables) are NOT this file's
// scope — they land in later slices (AI-28.1.2 … AI-28.6) per
// tasks.md's own slice boundaries. This file's fixtures therefore never
// carry a choice-0 content string: no text delta exists to map yet.

package openaicompat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// goroutineTolerance bounds the slack this file's leak checks allow
// between a "before" and "after" runtime.NumGoroutine() sample — background
// goroutines owned by the test harness or GC can fluctuate by a small
// amount independent of anything Stream does.
const goroutineTolerance = 3

// drainTimeout bounds how long this file's drain helpers wait for the
// carrier to produce an event or close, before failing the test rather
// than hanging it.
const drainTimeout = 5 * time.Second

// mustClient builds a real *Client against endpoint, failing the test on
// any construction error — every scenario in this file drives a real
// *Client over real HTTP, per R-ATS-003's and R-ATS-011's own httptest.Server
// posture.
func mustClient(t *testing.T, endpoint string) *Client {
	t.Helper()
	c, err := New(Config{
		Endpoint:   endpoint,
		Credential: NewCredential("stream-test-token"),
	})
	if err != nil {
		t.Fatalf("New() error = %v, want nil", err)
	}
	return c
}

// mustTextPart builds one text content part, failing the test on error.
func mustTextPart(t *testing.T, text string) ai.Part {
	t.Helper()
	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText(%q) error = %v, want nil", text, err)
	}
	return part
}

// validRequest builds a minimal, valid ai.Request — one user message, one
// text part — for the scenarios that need req.IsZero() == false.
func validRequest(t *testing.T) ai.Request {
	t.Helper()
	msg, err := ai.NewMessage(ai.RoleUser, mustTextPart(t, "hello"))
	if err != nil {
		t.Fatalf("ai.NewMessage() error = %v, want nil", err)
	}
	req, err := ai.NewRequest("requested-model", []ai.Message{msg})
	if err != nil {
		t.Fatalf("ai.NewRequest() error = %v, want nil", err)
	}
	return req
}

// sseServer returns a local test server that writes transcript verbatim as
// the whole response body, with an SSE content type, then returns — the
// simplest replay shape R-ATS-003's httptest.Server posture needs for a
// fixture that is not testing cancellation specifically.
func sseServer(t *testing.T, transcript string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, transcript)
	}))
}

// slowSSEServer returns a local test server whose handler writes first,
// flushes it onto the wire, then blocks — on the returned release channel,
// or on the request's own context being done, whichever comes first — so a
// test can drive Stream to read exactly one flushed write and then cancel
// while the producer's next Body.Read is genuinely in flight (R-ATS-005,
// S-ATS-016).
func slowSSEServer(t *testing.T, first string) (server *httptest.Server, release chan struct{}) {
	t.Helper()
	release = make(chan struct{})
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, first)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	return server, release
}

// drainAll reads every event off ch until it closes, failing the test if
// that takes longer than drainTimeout — never a bare `for range ch` that
// could hang the suite on a producer defect.
func drainAll(t *testing.T, ch <-chan ai.Event) []ai.Event {
	t.Helper()
	var events []ai.Event
	deadline := time.After(drainTimeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, ev)
		case <-deadline:
			t.Fatalf("drainAll: timed out after %s with %d event(s) so far", drainTimeout, len(events))
			return events
		}
	}
}

// drainUntilClosed reads and discards events until ch closes, without
// asserting on their content — the cancellation scenarios' own "either a
// terminal event follows, or the channel simply closes with none" premise
// (S-ATS-017): both are acceptable, only a hang is not.
func drainUntilClosed(t *testing.T, ch <-chan ai.Event, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatalf("drainUntilClosed: channel did not close within %s", timeout)
			return
		}
	}
}

// assertClosedExactlyOnce reads twice more from an already-drained ch,
// requiring both reads to report the closed-channel form immediately
// (S-ATS-011, S-ATS-018) — never a panic, never a further event.
func assertClosedExactlyOnce(t *testing.T, ch <-chan ai.Event) {
	t.Helper()
	for i := 0; i < 2; i++ {
		select {
		case _, ok := <-ch:
			if ok {
				t.Fatalf("receive %d after drain returned ok=true, want a closed channel", i+1)
			}
		case <-time.After(time.Second):
			t.Fatalf("receive %d after drain timed out, want an immediate closed-channel read", i+1)
		}
	}
}

// requireCheckStreamClean asserts events satisfies AI-14.4's own
// ai.CheckStream ordering invariants (design.md D8, verify-report C1/S3):
// every text block this package opened must be closed by the time the
// stream's own terminal event — a normal completion or a mid-stream error
// — appears. This is the one shared invocation site every failure-path
// test in this package calls, not merely the tests naming an
// unterminated-block probe by name — the whole-suite invariant S3 asked
// for, not just the three named probe shapes.
func requireCheckStreamClean(t *testing.T, events []ai.Event) {
	t.Helper()
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil — every open text block must close before the stream's terminal event (design.md D8)", report.Violation())
	}
}

// requireBlockClosedBeforeError asserts events end in a mid-stream
// ErrorPayload terminal, immediately preceded by the TextBlockEnd that
// closed the block the failure interrupted (design.md D8). This is the
// stronger, shape-specific companion to requireCheckStreamClean, reserved
// for the probe shapes verify-report C1 named explicitly — a block that
// was genuinely open at the moment of failure: truncation mid-block,
// invalid JSON mid-block, and an out-of-enum finish_reason mid-block.
func requireBlockClosedBeforeError(t *testing.T, events []ai.Event) {
	t.Helper()
	if len(events) < 2 {
		t.Fatalf("drained %d event(s), want at least a text_block_end and the terminal error event (design.md D8)", len(events))
	}
	last := events[len(events)-1]
	if _, ok := last.ErrorPayload(); !ok {
		t.Fatalf("last event kind = %v, want the terminal error event (design.md D8)", last.Kind())
	}
	preceding := events[len(events)-2]
	if _, ok := preceding.TextBlockEnd(); !ok {
		t.Fatalf("event preceding the terminal error = %v, want text_block_end (design.md D8)", preceding.Kind())
	}
}

// waitForGoroutineBaseline polls runtime.NumGoroutine() until it returns to
// within goroutineTolerance of before, or fails the test once timeout
// elapses — S-ATS-005's and S-ATS-016's own leak checks, both needing a
// bounded wait rather than one immediate sample (goroutine teardown is not
// instantaneous).
func waitForGoroutineBaseline(t *testing.T, before int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		after := runtime.NumGoroutine()
		if after <= before+goroutineTolerance {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("goroutine count = %d before, %d after — want within tolerance %d after %s", before, after, goroutineTolerance, timeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// ---------------------------------------------------------------------
// R-ATS-001 — one streaming entry point, satisfying ai.ModelProvider.
// ---------------------------------------------------------------------

// TestStream_SatisfiesModelProviderAtRuntime covers S-ATS-001.
func TestStream_SatisfiesModelProviderAtRuntime(t *testing.T) {
	t.Parallel()

	_, ok := any(&Client{}).(ai.ModelProvider)
	if !ok {
		t.Fatal("*Client does not satisfy ai.ModelProvider at run time (S-ATS-001)")
	}
}

// TestStream_SignatureMatchesModelProvider covers S-ATS-002: the reflected
// method set carries a Stream method whose signature is exactly
// func(context.Context, ai.Request) (<-chan ai.Event, error).
func TestStream_SignatureMatchesModelProvider(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(&Client{})
	method, ok := typ.MethodByName("Stream")
	if !ok {
		t.Fatal("*Client has no Stream method (S-ATS-002)")
	}

	// method.Func's type carries the receiver as its own first "in"
	// parameter (reflect's method-value shape), so the declared signature
	// starts at index 1.
	got := method.Func.Type()
	if got.NumIn() != 3 {
		t.Fatalf("Stream has %d parameter(s) including the receiver, want 3 (S-ATS-002)", got.NumIn())
	}
	wantCtx := reflect.TypeOf((*context.Context)(nil)).Elem()
	if got.In(1) != wantCtx {
		t.Errorf("Stream's first parameter = %v, want context.Context (S-ATS-002)", got.In(1))
	}
	wantReq := reflect.TypeOf(ai.Request{})
	if got.In(2) != wantReq {
		t.Errorf("Stream's second parameter = %v, want ai.Request (S-ATS-002)", got.In(2))
	}
	if got.NumOut() != 2 {
		t.Fatalf("Stream returns %d value(s), want 2 (S-ATS-002)", got.NumOut())
	}
	wantChan := reflect.TypeOf((<-chan ai.Event)(nil))
	if got.Out(0) != wantChan {
		t.Errorf("Stream's first return = %v, want <-chan ai.Event (S-ATS-002)", got.Out(0))
	}
	wantErr := reflect.TypeOf((*error)(nil)).Elem()
	if got.Out(1) != wantErr {
		t.Errorf("Stream's second return = %v, want error (S-ATS-002)", got.Out(1))
	}
}

// S-ATS-003 is not a new test in this file: it is agenttest's own,
// pre-existing TestModelProviderInterface_SignatureGuard
// (src/agenttest), run unchanged by the full-suite evidence in tasks.md —
// this milestone touches neither src/ai/provider.go nor src/agenttest
// (design.md's File Changes table), so the guard's own continued pass is
// the scenario's proof, not a duplicate assertion here.

// ---------------------------------------------------------------------
// R-ATS-002 — pre-stream contract: validate once, before I/O, before ctx.
// ---------------------------------------------------------------------

// TestStream_ZeroValueRequest_FailsBeforeAnyIO covers S-ATS-004 and
// S-ATS-005: a zero-value request fails directly, with a nil channel, zero
// inbound requests, and no goroutine leak.
func TestStream_ZeroValueRequest_FailsBeforeAnyIO(t *testing.T) {
	t.Parallel()

	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := mustClient(t, server.URL)

	before := runtime.NumGoroutine()
	ch, err := c.Stream(context.Background(), ai.Request{})
	after := runtime.NumGoroutine()

	if err == nil {
		t.Fatal("Stream() error = nil, want non-nil for a zero-value request (S-ATS-004)")
	}
	if ch != nil {
		t.Error("Stream() returned a non-nil channel for a zero-value request, want nil (S-ATS-004)")
	}
	if hits != 0 {
		t.Errorf("server recorded %d inbound request(s), want 0 (S-ATS-004)", hits)
	}
	if after > before+goroutineTolerance {
		t.Errorf("goroutine count = %d before, %d after — want within tolerance %d (S-ATS-005)", before, after, goroutineTolerance)
	}
}

// TestStream_ValidRequestAlreadyCancelledContext_ReportsCancellationPreStream
// covers S-ATS-006: once req is valid, an already-cancelled ctx short-
// circuits as a pre-stream cancellation failure, with zero inbound
// requests.
func TestStream_ValidRequestAlreadyCancelledContext_ReportsCancellationPreStream(t *testing.T) {
	t.Parallel()

	var hits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := mustClient(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ch, err := c.Stream(ctx, validRequest(t))
	if ch != nil {
		t.Error("Stream() returned a non-nil channel for an already-cancelled context, want nil (S-ATS-006)")
	}
	var failure *ai.Failure
	if !errors.As(err, &failure) {
		t.Fatalf("Stream() error = %v (%T), want a *ai.Failure (S-ATS-006)", err, err)
	}
	if failure.Category() != ai.FailureCategoryCancellation {
		t.Errorf("Category() = %v, want FailureCategoryCancellation (S-ATS-006)", failure.Category())
	}
	if failure.Delivery() != ai.DeliveryPreStream {
		t.Errorf("Delivery() = %v, want DeliveryPreStream (S-ATS-006)", failure.Delivery())
	}
	if hits != 0 {
		t.Errorf("server recorded %d inbound request(s), want 0 (S-ATS-006)", hits)
	}
}

// TestStream_InvalidRequestAndCancelledContext_ReportsValidationFailureFirst
// covers S-ATS-007: when both an invalid request and an already-cancelled
// context hold, the reported failure is the validation failure, never the
// cancellation one — validation is ordered strictly first. The endpoint is
// deliberately unreachable so an implementation that got the order wrong
// (consulted ctx or attempted I/O first) would fail loudly and
// distinguishably, rather than passing by accident.
func TestStream_InvalidRequestAndCancelledContext_ReportsValidationFailureFirst(t *testing.T) {
	t.Parallel()

	c := mustClient(t, "http://127.0.0.1:1")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.Stream(ctx, ai.Request{})
	if err == nil {
		t.Fatal("Stream() error = nil, want the validation failure (S-ATS-007)")
	}
	var failure *ai.Failure
	if errors.As(err, &failure) {
		t.Fatalf("Stream() reported a *ai.Failure (category=%v) — want the validation failure, not the cancellation one (S-ATS-007)", failure.Category())
	}
	if !errors.Is(err, ai.ErrEmpty) {
		t.Errorf("errors.Is(err, ai.ErrEmpty) = false, want true — validation must be ordered strictly first (S-ATS-007)")
	}
}

// ---------------------------------------------------------------------
// R-ATS-003 — a minimal transcript drains end to end, sequenced from 1,
// closed exactly once.
// ---------------------------------------------------------------------

// TestStream_MinimalTranscript_DrainsResponseStartThenCompletion covers
// S-ATS-008, S-ATS-009, S-ATS-010 and S-ATS-011.
func TestStream_MinimalTranscript_DrainsResponseStartThenCompletion(t *testing.T) {
	t.Parallel()

	server := sseServer(t, "data: {\"id\":\"chatcmpl-abc\",\"model\":\"gizmo-1\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ATS-008)", err)
	}

	events := drainAll(t, ch)

	// S-ATS-008
	if len(events) != 2 {
		t.Fatalf("drained %d event(s), want exactly 2 (S-ATS-008): %+v", len(events), events)
	}
	if events[0].Kind() != ai.EventKindResponseStart {
		t.Errorf("events[0].Kind() = %v, want EventKindResponseStart (S-ATS-008)", events[0].Kind())
	}
	if events[1].Kind() != ai.EventKindCompletion {
		t.Errorf("events[1].Kind() = %v, want EventKindCompletion (S-ATS-008)", events[1].Kind())
	}

	// S-ATS-009
	report := ai.CheckStream(events)
	if report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil (S-ATS-009)", report.Violation())
	}
	if !report.Terminated() {
		t.Error("CheckStream Terminated() = false, want true (S-ATS-009)")
	}

	// S-ATS-010
	for i, ev := range events {
		want := ai.Sequence(i + 1) //nolint:gosec // i is bounded by len(events), always small and non-negative
		if ev.Sequence() != want {
			t.Errorf("events[%d].Sequence() = %d, want %d (S-ATS-010)", i, ev.Sequence(), want)
		}
	}

	// S-ATS-011
	assertClosedExactlyOnce(t, ch)
}

// ---------------------------------------------------------------------
// R-ATS-004 — the vendor's response identity and served model land in the
// start event.
// ---------------------------------------------------------------------

// TestStream_ResponseIdentity_ByteExactAndNeverComparedToRequestedModel
// covers S-ATS-012 and S-ATS-013.
func TestStream_ResponseIdentity_ByteExactAndNeverComparedToRequestedModel(t *testing.T) {
	t.Parallel()

	server := sseServer(t, "data: {\"id\":\"chatcmpl-Xq7\",\"model\":\"gizmo-4o-2026-05-13\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
	defer server.Close()

	c := mustClient(t, server.URL)
	// validRequest's own requested model ("requested-model") deliberately
	// differs from the transcript's served model, above.
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want at least a response-start event")
	}
	start, ok := events[0].ResponseStart()
	if !ok {
		t.Fatal("events[0] carries no ResponseStart payload")
	}

	// S-ATS-012
	if start.ResponseID() != "chatcmpl-Xq7" {
		t.Errorf("ResponseID() = %q, want %q (S-ATS-012)", start.ResponseID(), "chatcmpl-Xq7")
	}
	if start.ServedModel() != "gizmo-4o-2026-05-13" {
		t.Errorf("ServedModel() = %q, want %q (S-ATS-012)", start.ServedModel(), "gizmo-4o-2026-05-13")
	}

	// S-ATS-013: no failure despite served model differing from requested
	// — no comparison is performed.
	for _, ev := range events {
		if ev.Kind() == ai.EventKindError {
			t.Fatal("unexpected error event — served model must never be compared to the requested model (S-ATS-013)")
		}
	}
}

// TestStream_ResponseIdentity_PreservesWhitespaceAndCase covers S-ATS-014:
// the response id reproduces its transcript bytes unchanged, including
// surrounding whitespace and mixed case.
func TestStream_ResponseIdentity_PreservesWhitespaceAndCase(t *testing.T) {
	t.Parallel()

	const id = "  ChatCmpl-MiXeD-Id  "
	transcript := fmt.Sprintf("data: {\"id\":%q,\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n", id)
	server := sseServer(t, transcript)
	defer server.Close()

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want at least a response-start event")
	}
	start, ok := events[0].ResponseStart()
	if !ok {
		t.Fatal("events[0] carries no ResponseStart payload")
	}
	if start.ResponseID() != id {
		t.Errorf("ResponseID() = %q, want %q, unchanged (S-ATS-014)", start.ResponseID(), id)
	}
}

// TestStream_FirstChunkMalformedIdentity_TerminatesWithMalformedResponseFailure
// covers S-ATS-015, triangulated across the three shapes ai.NewResponseStart
// rejects: an empty id, an empty model, and an absent id key entirely.
func TestStream_FirstChunkMalformedIdentity_TerminatesWithMalformedResponseFailure(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		chunk string
	}{
		{"empty id", `{"id":"","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`},
		{"empty model", `{"id":"x","model":"","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`},
		{"absent id key", `{"model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			transcript := "data: " + tc.chunk + "\n\ndata: [DONE]\n\n"
			server := sseServer(t, transcript)
			defer server.Close()

			c := mustClient(t, server.URL)
			ch, err := c.Stream(context.Background(), validRequest(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil — the failure is mid-stream, not returned directly (S-ATS-015)", err)
			}
			events := drainAll(t, ch)
			if len(events) != 1 {
				t.Fatalf("drained %d event(s), want exactly 1 — no response-start event with a minted or empty identity (S-ATS-015): %+v", len(events), events)
			}
			failure, ok := events[0].ErrorPayload()
			if !ok {
				t.Fatal("events[0] carries no error payload, want a malformed-response terminal (S-ATS-015)")
			}
			if failure.Category() != ai.FailureCategoryMalformedResponse {
				t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATS-015)", failure.Category())
			}
			if !errors.Is(failure, ai.ErrMalformedResponse) {
				t.Error("errors.Is(failure, ai.ErrMalformedResponse) = false, want true (S-ATS-015)")
			}
			requireCheckStreamClean(t, events)
		})
	}
}

// ---------------------------------------------------------------------
// R-ATS-005 — cancellation honors AI-20.3: every send selects, one closing
// site, no leak.
// ---------------------------------------------------------------------

// TestStream_CancellationMidStream_GoroutineExitsAndCarrierClosesCleanly
// covers S-ATS-016 and S-ATS-017.
func TestStream_CancellationMidStream_GoroutineExitsAndCarrierClosesCleanly(t *testing.T) {
	t.Parallel()

	before := runtime.NumGoroutine()

	server, release := slowSSEServer(t, "data: {\"id\":\"chatcmpl-x\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
	defer close(release)
	defer server.Close()

	c := mustClient(t, server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := c.Stream(ctx, validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	// Drain the one event the slow server manages to deliver before it
	// blocks, so the producer proceeds to its next Body.Read — the point
	// this test cancels against (R-ATS-005: cancellation aborts Body.Read).
	<-ch

	cancel()

	// S-ATS-017: either a terminal event follows, or the carrier simply
	// closes with none — both are the cancellation path; only a hang is
	// not acceptable.
	drainUntilClosed(t, ch, drainTimeout)

	// S-ATS-016
	waitForGoroutineBaseline(t, before, drainTimeout)
}

// TestStream_CarrierClosesExactlyOnce_AcrossOutcomes covers S-ATS-018: a
// normal completion, a terminal failure, and a cancellation each close
// their own carrier exactly once, and the whole table runs -race clean.
func TestStream_CarrierClosesExactlyOnce_AcrossOutcomes(t *testing.T) {
	t.Parallel()

	t.Run("normal completion", func(t *testing.T) {
		t.Parallel()

		server := sseServer(t, "data: {\"id\":\"chatcmpl-a\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		defer server.Close()

		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		drainAll(t, ch)
		assertClosedExactlyOnce(t, ch)
	})

	t.Run("terminal failure", func(t *testing.T) {
		t.Parallel()

		server := sseServer(t, "data: {\"id\":\"\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\ndata: [DONE]\n\n")
		defer server.Close()

		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		events := drainAll(t, ch)
		assertClosedExactlyOnce(t, ch)
		requireCheckStreamClean(t, events)
	})

	t.Run("cancellation", func(t *testing.T) {
		t.Parallel()

		server, release := slowSSEServer(t, "data: {\"id\":\"chatcmpl-b\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
		defer close(release)
		defer server.Close()

		c := mustClient(t, server.URL)
		ctx, cancel := context.WithCancel(context.Background())
		ch, err := c.Stream(ctx, validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil", err)
		}
		<-ch
		cancel()
		drainUntilClosed(t, ch, drainTimeout)
		assertClosedExactlyOnce(t, ch)
	})
}

// S-ATS-019 is [inspection], not [test]: it is discharged by a reviewer
// reading the shipped producer source (grep -c '^func\|close(' style
// check), never by a test in this file — see tasks.md's own evidence log
// for the recorded count.
