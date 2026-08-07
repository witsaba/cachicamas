// AI-33.5a — Drain-before-close in run()'s defer chain (R-AIS-033).
//
// # Why external (package openaicompat_test)
//
// AI-33.5's two files (a_i-33_5a_test.go, a_i-33_5b_test.go) live in
// package openaicompat_test — external, mirroring the posture
// backend/agent/src/ai/openaicompat/openrouter/conformance/bridge_test.go
// already uses for its R-OR conformance bridge (R-CNF-001,
// openrouter/conformance/bridge_test.go:43–67). The other AI-33 test
// files (a_i-33_{1,2,3,4}_test.go) are package openaicompat, with
// full access to mustClient / validRequest / serveTranscripts; AI-33.5
// is external on purpose (R-AIS-038's "full-package wrapper" runs in
// its own package, with no privileged access to the producer's
// unexported seams), and it duplicates the helpers it needs to use
// only the exported surface (openaicompat.New / Config /
// NewCredential, agenttest.RequireNoGoroutineLeak / DrainAndRecord /
// DefaultDrainTimeout). backend/agent/go.mod stays at 3 lines, zero
// requires (R-STK-009 / NFR-STK-A's mechanical proof; spec acceptance
// criterion 7).
//
// # Split rationale (tasks.md AI-33.5 § 33.5.1, line 173)
//
// The aggregate is split into 33.5a (this file: drain tests + drain
// production change, ~250 lines) and 33.5b
// (a_i-33_5b_test.go: full-suite leak check, ~280 lines) so each
// stays well below the 400-line review budget. Both files share
// helpers (same package, same _test.go build context), so this file
// owns the helpers and 33.5b reuses them.
//
// # What this file proves (R-AIS-033)
//
// The producer's `defer func() { _ = resp.Body.Close() }()` at
// stream.go:345 closes the body without draining. capture.go:117-122
// already proves the drain-before-close idiom works on the
// bounded-capture path; this file's tests prove the SAME idiom on the
// producer's own defer chain. They stand up an httptest.NewServer
// whose handler writes a transcript ending in [DONE] FOLLOWED BY
// trailing garbage bytes (the handler flushes once after the write
// to force chunked transfer encoding — without chunked encoding,
// Content-Length would mark the body "fully read" by byte count alone
// and the test would pass vacuously). The producer reads the
// transcript, hits [DONE], exits; the body still has unread bytes.
// Without drain, the transport sees "body not fully consumed" and
// closes the underlying keep-alive connection instead of returning it
// to the idle pool; a follow-up request on the same transport opens
// a NEW TCP connection. With drain (io.Copy(io.Discard, resp.Body)
// immediately before resp.Body.Close), the body is fully drained,
// EOF is observed, and the keep-alive slot is reused.
//
// The proof is connection-reuse count via the server's ConnState
// callback — exactly 1 TCP connection across all 50 repeats means
// drain fired; ≥2 means it did not (Go's transport retries a small
// number of times before opening fresh, so the exact count varies by
// run). Both stream kinds per charter line 1989.
//
// # Pins
//
// R-AIS-033 (drain obligation; capture.go:117-122's own idiom mirrors
// here), R-CNF-009 (single closing site — run's defer chain owns
// close(out) at stream.go:344), R-CNF-011 (conformance bounded-close
// invariant stays green), R-ATS-003 (single producer model; the drain
// does NOT introduce a persistent second goroutine — the drain runs
// INSIDE the existing producer's defer chain), R-STK-007 (leak
// amplitude mechanism — 50 repeats), R-STK-008 (serial-only; no
// t.Parallel() anywhere in this file), R-STK-009 (stdlib-only —
// go.mod unchanged).

package openaicompat_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// ai33_5StallBound is the safety margin added to DefaultDrainTimeout
// for AI-33.5 close bounds. Same posture a_i-33_2_test.go:81 already
// uses (ai33StallBound = 1s) for the same empirical reason.
const ai33_5StallBound = 1 * time.Second

// ai33_5ShortTimeout is a small backstop for "synchronous first-event"
// receives in the between-frames scenarios — large enough not to
// flake, small enough that a stuck producer surfaces here, not as a
// 30s hang.
const ai33_5ShortTimeout = 2 * time.Second

// ai33_5StallHandler returns a handler that writes each frame in
// firstFrames verbatim, flushes each onto the wire, then blocks on
// either the request's own context (fired when the client closes the
// connection — the normal AI-33.2 close path) or a long timer (so the
// handler returns eventually even if the client never cancels, keeping
// the test from holding a goroutine past its useful life).
//
// Same shape as a_i-33_2_test.go:96 (ai33StallHandler) — duplicated
// here because that helper is unexported and unreachable from
// package openaicompat_test. Each frame must already be valid SSE
// bytes (a leading "data: " and a trailing "\n\n" each).
func ai33_5StallHandler(firstFrames ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, _ := w.(http.Flusher)
		for _, frame := range firstFrames {
			_, _ = io.WriteString(w, frame)
			if flusher != nil {
				flusher.Flush()
			}
		}
		select {
		case <-r.Context().Done():
		case <-time.After(60 * time.Second):
		}
	}
}

// ai33_5TextTranscript is the minimal well-formed text stream this
// file's tests use: identity chunk + one content delta + terminal
// chunk + [DONE].
const ai33_5TextTranscript = "" +
	"data: {\"id\":\"chatcmpl-ai33-5-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-5-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-5-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
	"data: [DONE]\n\n"

// ai33_5ToolTranscript is the tool-call counterpart (charter line 1989
// requires both stream kinds per subnode): identity + tool-call start +
// arguments delta + terminal + [DONE].
const ai33_5ToolTranscript = "" +
	"data: {\"id\":\"chatcmpl-ai33-5-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-5-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_ai33_5\",\"function\":{\"name\":\"get_weather\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-5-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{}\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-5-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
	"data: [DONE]\n\n"

// ai33_5TrailingGarbage is the fixed suffix appended AFTER the [DONE]
// sentinel for the drain test. It exists purely so the response body
// carries more bytes than the producer actually reads — the empirical
// condition that distinguishes drain from no-drain at the transport
// layer. The exact size is not load-bearing, only its presence is.
const ai33_5TrailingGarbage = "" +
	"GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-" +
	"GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-GARBAGE-"

// withTrailingGarbage returns transcript with ai33_5TrailingGarbage
// appended after the [DONE] sentinel — the drain test's exact body
// shape (R-AIS-033 / S-1).
func withTrailingGarbage(transcript string) []byte {
	return []byte(transcript + ai33_5TrailingGarbage)
}

// serveBody returns an http.HandlerFunc that serves body verbatim as
// the response with the streaming media type, flushing once after
// the write so the server emits chunked transfer encoding rather
// than Content-Length. With chunked encoding, the client knows the
// body is "fully read" only when it sees the terminating empty
// chunk (0\r\n\r\n) — so a producer that exits without draining the
// body leaves the transport's "fully consumed" flag unset, the
// keep-alive slot is discarded, and the next request opens a new TCP
// connection. With drain (io.Copy reading the remaining bytes), the
// client sees the terminator, the keep-alive slot is reused.
//
// This is the property R-AIS-033 / S-1 names: "the response body is
// drained to io.Discard before resp.Body.Close() runs, AND the next
// request against the same transport succeeds without blocking on
// the prior connection's unread bytes." A plain single-Write handler
// (no flush) would set Content-Length, and the client would mark the
// body "fully read" by byte count alone — drain would be a no-op,
// the test would pass vacuously, and the producer's behavior would
// not be observable.
func serveBody(body []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

// countTCPConnections stands up an httptest.Server with the streaming
// handler, installs a ConnState callback BEFORE Start (so the
// callback is registered without racing the server goroutine, which
// reads server.Config.ConnState under the hood), and returns the
// atomic counter the callback increments on every StateNew transition
// (one increment per accepted TCP connection). Returns the counter
// so the test can read it after the scenario.
func countTCPConnections(handler http.HandlerFunc) (*httptest.Server, *atomic.Int32) {
	var conns atomic.Int32
	server := httptest.NewUnstartedServer(handler)
	server.Config.ConnState = func(_ net.Conn, s http.ConnState) {
		if s == http.StateNew {
			conns.Add(1)
		}
	}
	server.Start()
	return server, &conns
}

// mustNewExternalClient builds a real *openaicompat.Client against
// endpoint — duplicates mustClient (stream_test.go:50) because that
// helper is unexported and unreachable from package openaicompat_test.
func mustNewExternalClient(t *testing.T, endpoint string) *openaicompat.Client {
	t.Helper()
	c, err := openaicompat.New(openaicompat.Config{
		Endpoint:   endpoint,
		Credential: openaicompat.NewCredential("ai-33-5-token"),
	})
	if err != nil {
		t.Fatalf("openaicompat.New: %v", err)
	}
	return c
}

// validTextRequest builds a minimal valid text-only ai.Request —
// duplicates validRequest (stream_test.go:74) for the same external-
// package reason.
func validTextRequest(t *testing.T) ai.Request {
	t.Helper()
	part, err := ai.NewText("hello")
	if err != nil {
		t.Fatalf("ai.NewText: %v", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	req, err := ai.NewRequest("requested-model", []ai.Message{msg})
	if err != nil {
		t.Fatalf("ai.NewRequest: %v", err)
	}
	return req
}

// validToolCallRequest builds a minimal valid tool-call ai.Request —
// duplicates a_i-33_1_test.go:200 (validToolCallRequest) for the same
// reason. One tool, one user message.
func validToolCallRequest(t *testing.T) ai.Request {
	t.Helper()
	tool, err := ai.NewTool(
		"get_weather",
		"Get the current weather for a location.",
		[]byte(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`),
	)
	if err != nil {
		t.Fatalf("ai.NewTool: %v", err)
	}
	tools, err := ai.NewToolSet(tool)
	if err != nil {
		t.Fatalf("ai.NewToolSet: %v", err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, mustTextPartExternal(t, "what is the weather?"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	req, err := ai.NewRequest("requested-model", []ai.Message{msg}, ai.WithTools(tools))
	if err != nil {
		t.Fatalf("ai.NewRequest: %v", err)
	}
	return req
}

// mustTextPartExternal duplicates mustTextPart (stream_test.go:63).
func mustTextPartExternal(t *testing.T, text string) ai.Part {
	t.Helper()
	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText(%q): %v", text, err)
	}
	return part
}

// hijackCloseHandler returns a handler that ACCEPTS the request then
// hijacks the connection and closes it without writing — the same
// shape contextBeforeFirstFrameServer uses (stream_failure_test.go:
// 391) — duplicated here because that helper is unexported.
func hijackCloseHandler(t *testing.T) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Skip("ResponseWriter is not a Hijacker — cannot simulate a transport-close-before-headers")
		}
		conn, _, err := hj.Hijack()
		if err != nil {
			t.Fatalf("Hijack: %v", err)
		}
		_ = conn.Close()
	}
}

// TestAI33_5_DrainBeforeClose_OnCompletionPath_Text covers
// R-AIS-033 / S-1: the producer's normal completion path. The server
// writes a well-formed transcript ending in [DONE] PLUS trailing
// garbage bytes after the sentinel — so the body carries more bytes
// than the producer actually reads. The producer hits [DONE], exits,
// and the trailing bytes are unread.
//
// Without drain (current stream.go:345), the body is closed with
// unread bytes; the transport considers the body "not fully consumed"
// and closes the underlying keep-alive connection instead of
// returning it to the idle pool. A follow-up request on the SAME
// client transport therefore opens a NEW TCP connection — conns = 2.
//
// With drain (R-AIS-033, capture.go:117-122's idiom extended into
// stream.go:345), io.Copy(io.Discard, resp.Body) reads the remaining
// trailing bytes before the close fires; the transport sees EOF, the
// keep-alive slot is reused — conns = 1.
//
// This is the load-bearing RED: without the drain change, conns == 2
// per repeat × 50 repeats (so up to 100 across the suite) and the
// test fails; with it, conns == 1 across all repeats.
//
// # Client reuse — why ONE *openaicompat.Client, not 50
//
// Each call to openaicompat.New builds its own *http.Transport, and
// each Transport holds idle-connection goroutines that take time to
// release (they exit on the next read return or when the Transport's
// connection state stabilises — neither is bounded by a defer).
// Constructing a fresh client per repeat accumulates goroutines
// faster than leakTolerance can absorb; reusing a single client
// across the 50 repeats keeps the Transport stable. The
// connection-reuse invariant under test is about a single client's
// Transport re-using its keep-alive slot — so client reuse across
// repeats is exactly the shape the spec R-AIS-033 / S-1 asks for.
func TestAI33_5_DrainBeforeClose_OnCompletionPath_Text(t *testing.T) {
	server, conns := countTCPConnections(serveBody(withTrailingGarbage(ai33_5TextTranscript)))
	defer server.Close()
	c := mustNewExternalClient(t, server.URL)

	agenttest.RequireNoGoroutineLeak(t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// First stream — drains to close. Without drain, the trailing
		// bytes are unread; the transport closes the keep-alive slot.
		ch1, err := c.Stream(ctx, validTextRequest(t))
		if err != nil {
			t.Fatalf("first Stream() error = %v, want nil (R-AIS-033 / S-1)", err)
		}
		rec := agenttest.DrainAndRecord(t, ch1, agenttest.DefaultDrainTimeout)
		if rec.Len() == 0 {
			t.Fatalf("first Stream() drained 0 events, want ≥2 (R-AIS-033 / S-1): the producer never reached its first send")
		}

		// Second stream against the SAME server (same client, same
		// transport). If drain fired, this reuses the keep-alive
		// connection — total TCP connections == 1. If drain did not
		// fire, the transport closed the prior connection and opens a
		// new one — total TCP connections == 2.
		ch2, err := c.Stream(ctx, validTextRequest(t))
		if err != nil {
			t.Fatalf("second Stream() error = %v, want nil (R-AIS-033 / S-1)", err)
		}
		rec2 := agenttest.DrainAndRecord(t, ch2, agenttest.DefaultDrainTimeout)
		if rec2.Len() == 0 {
			t.Fatalf("second Stream() drained 0 events, want ≥2 (R-AIS-033 / S-1): the second connection never produced events")
		}
	})

	// 50 repeats × 2 streams/repeat = up to 100 connection attempts.
	// Drain present: 1 connection, reused 100 times. Drain absent:
	// the transport closes the prior connection after every
	// non-fully-consumed body and opens a new one — observed across
	// the suite as ≥2 distinct TCP connections (Go's transport
	// retries a small number of times before opening fresh, so the
	// exact count varies by run; the load-bearing bound is conns==1,
	// not some small N).
	if got := conns.Load(); got != 1 {
		t.Errorf("TCP connections across all repeats = %d, want 1 — drain-before-close did NOT free the keep-alive slot for reuse (R-AIS-033 / S-1): without drain, the transport closes the connection after every non-fully-consumed body",
			got)
	}
}
// R-AIS-033 / S-1 on the tool-call stream kind (charter line 1989):
// the same connection-reuse proof, with a tool-call transcript. The
// tool-call accumulation path runs end-to-end, then the producer hits
// [DONE] and exits; the trailing bytes remain unread. Without drain,
// conns == 2 (per repeat × 50 = up to 100); with drain, conns == 1.
//
// Same single-client reuse pattern as the text sibling — see that
// test's header for why (Transport idle-connection goroutines from
// per-repeat New() would accumulate past leakTolerance).
func TestAI33_5_DrainBeforeClose_OnCompletionPath_ToolCall(t *testing.T) {
	server, conns := countTCPConnections(serveBody(withTrailingGarbage(ai33_5ToolTranscript)))
	defer server.Close()
	c := mustNewExternalClient(t, server.URL)

	agenttest.RequireNoGoroutineLeak(t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch1, err := c.Stream(ctx, validToolCallRequest(t))
		if err != nil {
			t.Fatalf("first Stream() error = %v, want nil (R-AIS-033 / S-1)", err)
		}
		rec := agenttest.DrainAndRecord(t, ch1, agenttest.DefaultDrainTimeout)
		if rec.Len() == 0 {
			t.Fatalf("first Stream() drained 0 events, want ≥3 — tool-call path emits ResponseStart + TextBlockStart + ToolCallStart before [DONE] (R-AIS-033 / S-1)")
		}

		ch2, err := c.Stream(ctx, validToolCallRequest(t))
		if err != nil {
			t.Fatalf("second Stream() error = %v, want nil (R-AIS-033 / S-1)", err)
		}
		rec2 := agenttest.DrainAndRecord(t, ch2, agenttest.DefaultDrainTimeout)
		if rec2.Len() == 0 {
			t.Fatalf("second Stream() drained 0 events, want ≥3 (R-AIS-033 / S-1)")
		}
	})

	if got := conns.Load(); got != 1 {
		t.Errorf("TCP connections across all repeats = %d, want 1 — drain-before-close did NOT free the keep-alive slot for reuse on the tool-call path (R-AIS-033 / S-1)",
			got)
	}
}
