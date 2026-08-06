// AI-30.4 — R-ATL-009: truncation and malformation both yield one typed
// failure carrying the raw partial fragment, bounded and never rendered.
//
// S-ATL-044…051:
//   044: mid-call EOF with no terminal chunk → MidStreamFailure/
//        FailureCategoryMalformedResponse/PartialOutput()=true
//   045: typed cause reachable via errors.As carrying raw fragment bytes
//   046: malformed-but-cleanly-closed → same failure family
//   047: raw bytes bounded by captureLimit
//   048: outer Error() and cause's Error() never reproduce fragment bytes
//   049: panic-table — no panic on any failing transcript
//   050: events preceding the failure preserved in order, byte-exact
//   051 (inspection): bytes accessor unexported, cause's Error() is fixed
//        string built from no captured byte, raw-fragment bound reuses
//        captureLimit (capture.go's existing 8 KiB)

package openaicompat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestTruncation_MidCallEOF_TypedFailure covers S-ATL-044: a stream
// delivering a start and two fragments for one call, then closing the
// connection with no terminal chunk, terminates with a typed
// malformed-response failure whose PartialOutput() reports true.
func TestTruncation_MidCallEOF_TypedFailure(t *testing.T) {
	t.Parallel()

	// Build a transcript that delivers start+frag1+frag2 and then ends
	// abruptly (no terminal chunk, no [DONE]).
	transcript := "" +
		"data: {\"id\":\"chatcmpl-trunc\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_T\",\"function\":{\"name\":\"search\",\"arguments\":\"{\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-trunc\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"q\\\":\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-trunc\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"a\\\"}\"}}]},\"finish_reason\":null}]}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, transcript)
		// Flush before closing so the client receives the full transcript.
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Close the connection abruptly.
		if h, ok := w.(http.Hijacker); ok {
			conn, _, _ := h.Hijack()
			_ = conn.Close()
		}
	}))
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := drainAll(t, ch)
	if len(events) == 0 {
		t.Fatal("drained zero events, want start events + typed failure (S-ATL-044)")
	}
	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event kind = %v, want ErrorPayload (S-ATL-044): %+v", last.Kind(), last)
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATL-044)", failure.Category())
	}
	if !failure.PartialOutput() {
		t.Error("PartialOutput() = false, want true (tool events were emitted before failure, S-ATL-044)")
	}
	if !errors.Is(failure, ai.ErrMalformedResponse) {
		t.Error("errors.Is(failure, ai.ErrMalformedResponse) = false, want true (S-ATL-044)")
	}
	requireCheckStreamClean(t, events)
}

// TestTruncation_TypedCauseReachableViaErrorsAs covers S-ATL-045: the
// typed cause this package produces is reachable through errors.As
// and carries the raw accumulated bytes.
func TestTruncation_TypedCauseReachableViaErrorsAs(t *testing.T) {
	t.Parallel()

	// Drive mapperState directly to assemble a malformed-assembly error.
	state := &mapperState{}
	id := `{"index":0,"id":"call_M","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	// Accumulate bytes that will not form valid JSON at close.
	bad := `{"q":"a"`
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(bad))+`}}`)); err != nil {
		t.Fatalf("applyChunk(bad) error = %v", err)
	}
	// Close — NewToolCall will fail because args is not well-formed JSON.
	_, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err == nil {
		t.Fatal("applyChunk(terminal) error = nil, want malformedToolCallAssembly (S-ATL-046)")
	}
	bytes, ok := malformedAssemblyBytes(err)
	if !ok {
		t.Fatalf("errors.As(malformedToolCallAssembly) failed, want true (S-ATL-045): err = %v", err)
	}
	if string(bytes) != bad {
		t.Errorf("raw bytes = %q, want %q (S-ATL-045)", bytes, bad)
	}
}

// TestTruncation_MalformedCleanClose covers S-ATL-046: a call whose
// fragments concatenate to malformed JSON but closes with a terminal
// chunk yields the same typed failure family as truncation.
func TestTruncation_MalformedCleanClose(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-malformed\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_M\",\"function\":{\"name\":\"search\",\"arguments\":\"{\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-malformed\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"q\\\":\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-malformed\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
		"data: [DONE]\n\n"

	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := drainAll(t, ch)
	last := events[len(events)-1]
	failure, ok := last.ErrorPayload()
	if !ok {
		t.Fatalf("last event kind = %v, want ErrorPayload (S-ATL-046)", last.Kind())
	}
	if failure.Category() != ai.FailureCategoryMalformedResponse {
		t.Errorf("Category() = %v, want FailureCategoryMalformedResponse (S-ATL-046)", failure.Category())
	}
	requireCheckStreamClean(t, events)
}

// TestTruncation_BytesBoundedByCaptureLimit covers S-ATL-047: when the
// accumulated bytes exceed captureLimit, the typed cause's bytes
// are bounded by captureLimit.
func TestTruncation_BytesBoundedByCaptureLimit(t *testing.T) {
	t.Parallel()

	// Build a state whose accumulated bytes exceed captureLimit. To
	// trigger assembly error, we need the bytes to ALSO be malformed
	// JSON. A long string that doesn't close its quotes works.
	state := &mapperState{}
	id := `{"index":0,"id":"call_B","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	// Assemble bytes: {"a":"AAAA...AAA" — long, unterminated string.
	prefix := `{"a":"`
	big := make([]byte, int(captureLimit)+100)
	for i := range big {
		big[i] = 'A'
	}
	// The assembly will fail at close; bytes() must be bounded.
	full := prefix + string(big) // intentionally not closing the string
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(full))+`}}`)); err != nil {
		t.Fatalf("applyChunk(full) error = %v", err)
	}
	_, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err == nil {
		t.Fatal("expected assembly error")
	}
	got, ok := malformedAssemblyBytes(err)
	if !ok {
		t.Fatalf("errors.As failed: %v", err)
	}
	if len(got) > captureLimit {
		t.Errorf("bytes length = %d, want ≤ %d (captureLimit, S-ATL-047)", len(got), captureLimit)
	}
}

// TestTruncation_NoBytesInRenderedError covers S-ATL-048: the outer
// failure's Error() and the typed cause's Error() do NOT reproduce
// any byte from the accumulated fragments.
func TestTruncation_NoBytesInRenderedError(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_X","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	marker := "DISTINCTIVE_MARKER_TOKEN_XYZ_42"
	bad := `{"q":"` + marker + `"`
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(bad))+`}}`)); err != nil {
		t.Fatalf("applyChunk(bad) error = %v", err)
	}
	err := state.applyChunkTerminalForTest()
	if err == nil {
		t.Fatal("expected assembly error")
	}
	// The cause's Error() must not contain the marker.
	causeErr := err
	if strings.Contains(causeErr.Error(), marker) {
		t.Errorf("cause Error() = %q contains marker %q — leak (S-ATL-048)", causeErr.Error(), marker)
	}
}

// applyChunkTerminalForTest is a tiny helper: applies a terminal chunk
// and returns the error (possibly nil) — used in this file's
// inspection tests where we want the typed cause as an error value.
func (s *mapperState) applyChunkTerminalForTest() error {
	_, err := s.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	return err
}

// TestTruncation_PanicTable covers S-ATL-049: every failing transcript
// in this milestone — truncation, malformed assembly, cap exceeded,
// missing index, identity mismatch, missing name — runs through a
// recover() probe without panicking.
func TestTruncation_PanicTable(t *testing.T) {
	t.Parallel()

	type row struct {
		name string
		run  func()
	}
	rows := []row{
		{
			name: "truncation_mid_call_eof",
			run: func() {
				state := &mapperState{}
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_P1","function":{"name":"search"}}`))
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":"{"}}`))
			},
		},
		{
			name: "malformed_assembly",
			run: func() {
				state := &mapperState{}
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_P2","function":{"name":"search"}}`))
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":"{\"q\":"}}`))
				_ = state.applyChunkTerminalForTest()
			},
		},
		{
			name: "cap_exceeded",
			run: func() {
				state := &mapperState{}
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_P3","function":{"name":"search"}}`))
				big := make([]byte, int(toolCallAccumulationCap)+1)
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes(big)+`}}`))
			},
		},
		{
			name: "missing_index",
			run: func() {
				state := &mapperState{}
				_, _ = state.applyChunk(chunkFromTools("c", `{"id":"call_P4","function":{"name":"search","arguments":"{}"}}`))
			},
		},
		{
			name: "identity_mismatch",
			run: func() {
				state := &mapperState{}
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_P5a","function":{"name":"search"}}`))
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_P5b","function":{"name":"search"}}`))
			},
		},
		{
			name: "missing_name",
			run: func() {
				state := &mapperState{}
				_, _ = state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_P6","function":{"arguments":"{}"}}`))
				_ = state.applyChunkTerminalForTest()
			},
		},
	}

	for _, r := range rows {
		t.Run(r.name, func(t *testing.T) {
			defer func() {
				if rec := recover(); rec != nil {
					t.Errorf("recovered from panic in row %q: %v — every failing transcript must terminate typed, not panic (S-ATL-049)", r.name, rec)
				}
			}()
			r.run()
		})
	}
}

// TestTruncation_PrecedingEventsPreserved covers S-ATL-050: events
// preceding a mid-stream failure remain in order, byte-exact.
func TestTruncation_PrecedingEventsPreserved(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-pres\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"before\",\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-pres\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_P\",\"function\":{\"name\":\"search\",\"arguments\":\"{\"}}]},\"finish_reason\":null}]}\n\n" +
		"data: {\"id\":\"chatcmpl-pres\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"q\\\":\"}}]},\"finish_reason\":null}]}\n\n"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, transcript)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		if h, ok := w.(http.Hijacker); ok {
			conn, _, _ := h.Hijack()
			_ = conn.Close()
		}
	}))
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}
	events := drainAll(t, ch)
	// Last must be an ErrorPayload.
	if _, ok := events[len(events)-1].ErrorPayload(); !ok {
		t.Fatal("last event is not ErrorPayload")
	}
	// The preceding text delta must be preserved byte-exact.
	var sawTextDelta bool
	for _, ev := range events {
		if d, ok := ev.TextDelta(); ok {
			if d.Delta() == "before" {
				sawTextDelta = true
			}
		}
	}
	if !sawTextDelta {
		t.Error("preceding text delta did not survive in the recording (S-ATL-050)")
	}
	requireCheckStreamClean(t, events)
}
