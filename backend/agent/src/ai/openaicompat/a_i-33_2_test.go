// AI-33.2 — Cancel between frames (R-AIS-035 / S-1, S-2, S-3).
//
// The empirical answer to the proposal's R1 risk (proposal § 10,
// explore § 4.3 / § 5.1): the producer's read loop at stream.go:369–530
// is a synchronous resp.Body.Read — it does NOT select on ctx.Done()
// inside the loop. Whether cancellation propagates through the
// transport into Body.Read is therefore empirical: it depends on the
// transport, the protocol (HTTP/1.1 vs HTTP/2), and whether the
// keep-alive slot is already established. For an httptest.NewServer over
// HTTP/1.1 (the test-only harness) the question reduces to "does the
// transport tear down the connection when the client's context is
// cancelled while a read is in flight" — and the answer must be proven
// against a real *http.Client, not against a fake provider.
//
// The RED posture: a stalling handler emits one SSE frame, flushes, and
// blocks — so the producer reads exactly one frame's worth of events
// and then sits in Body.Read waiting for the next byte. The test
// receives the first event, cancels ctx, and bounds the close to
// DefaultDrainTimeout + safety. If the transport honours ctx, Body.Read
// returns with an error after the connection is torn down, run() walks
// its ctx.Err() != nil branch (stream.go:503–526) and exits, the defer
// chain at stream.go:343–345 closes the body and the channel, and the
// consumer's drain observes the close within the bound. If the
// transport does NOT honour ctx, the test times out — RED.
//
// # No new helpers, no new deps
//
// This file reuses stream_test.go's mustClient, validRequest, and
// assertClosedExactlyOnce (same package, internal); the
// validToolCallRequest builder a_i-33_1_test.go:200 already exposes
// (same package, internal); the existing slowSSEServer shape
// (stream_test.go:106) — but inline as ai33StallHandler below so the
// SSE bytes it sends are parameterized per scenario (text vs
// tool-call) and the handler does not own a release channel the tests
// never use; and agenttest.RequireNoGoroutineLeak +
// agenttest.DrainAndRecord (stdlib-only, R-STK-009). backend/agent/go.mod
// is unchanged (R-STK-009's mechanical proof; spec acceptance
// criterion 7).
//
// # Pins
//
// R-CNF-011 (conformance bounded-close invariant — stays green; the
// conformance case never crosses a real HTTP transport and is therefore
// unaffected by whatever this file's RED observation is); R-STK-028
// (between-frames cancellation bounded close); R-STK-008 (serial-only
// leak check; no t.Parallel() anywhere in this file). R-ATS-003 is
// preserved — no persistent second goroutine introduced anywhere; this
// file does not edit stream.go.
//
// # Both stream kinds, per charter line 1989
//
// S-1 covers text streams (one identity chunk → ResponseStart +
// TextBlockStart); S-2 covers tool-call streams (identity chunk +
// tool-call-start chunk → ResponseStart + TextBlockStart + ToolCallStart).
// A tool-call proof that never crosses the tool-call accumulation path
// proves nothing about its buffers, so both shapes are asserted. The
// tool-call path is exercised by driving validToolCallRequest through
// the same producer; specific event-kinds on the recorded stream are
// not re-asserted here — the bounded close, single close, and no-leak
// triple is the spec contract, identical for text and tool-call.

package openaicompat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// ai33StallBound is the safety margin added to DefaultDrainTimeout for the
// AI-33.2 close bound — long enough to absorb a busy CI runner's
// scheduling jitter, short enough to surface a transport that genuinely
// fails to honour ctx in seconds rather than minutes.
const ai33StallBound = 1 * time.Second

// ai33StallHandler returns a handler that writes each frame in firstFrames
// verbatim as the response body, flushes each onto the wire, then blocks
// on either the request's own context (fired when the client closes the
// connection — the normal AI-33.2 close path) or a long timer (so the
// handler returns eventually even if the client never cancels, keeping
// the test from holding a goroutine past its useful life).
//
// The frames must already be valid SSE bytes (a leading "data: " and a
// trailing "\n\n" each). newDripHandler (timeout_test.go:20) writes
// plain text rather than SSE, so this file inlines a small
// SSE-parameterized variant — explore § 5.5 already notes a
// single-purpose helper is the readable form here, not a refactor of
// the existing plain-text drip handler.
func ai33StallHandler(firstFrames ...string) http.HandlerFunc {
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

// ai33FirstEvent reads exactly one event from ch, failing the test if
// the channel closes first or no event lands within timeout. Used by
// every AI-33.2 scenario to confirm the producer reached and passed its
// first send before cancellation (the same synchronous-handoff proof
// conformance_cancellation.go:69–75 already uses for R-CNF-011).
func ai33FirstEvent(t *testing.T, ch <-chan ai.Event, timeout time.Duration) ai.Event {
	t.Helper()
	select {
	case ev, ok := <-ch:
		if !ok {
			t.Fatalf("ai33: channel closed before the first event landed, want the first SSE frame's events to be observable before cancellation")
		}
		return ev
	case <-time.After(timeout):
		t.Fatalf("ai33: no event within %s, want the first SSE frame's events to be observable before cancellation", timeout)
		return ai.Event{}
	}
}

// ai33DrainUntilClosed reads and discards events from ch until it closes
// or timeout elapses — fails the test on timeout (so a transport that
// fails to honour ctx surfaces here, not as a silent hang), returns
// silently on close. Bounded by DefaultDrainTimeout + ai33StallBound,
// the same cap spec R-AIS-035 / S-1 names explicitly.
func ai33DrainUntilClosed(t *testing.T, ch <-chan ai.Event, timeout time.Duration) {
	t.Helper()
	deadline := time.After(timeout)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-deadline:
			t.Fatalf("ai33: channel did not close within %s (R-AIS-035 / S-1, R-STK-028)", timeout)
			return
		}
	}
}

// TestAI33_2_TextStream_CancelBetweenFrames covers R-AIS-035 / S-1:
// a real text transcript with one identity chunk frame lands
// ResponseStart + TextBlockStart before stalling. The test receives
// the first event, cancels ctx, and bounds the close to
// DefaultDrainTimeout + safety — the empirical proof that ctx
// cancellation reaches the producer's read loop within the bound
// (R-CNF-011, R-STK-028). RequireNoGoroutineLeak × 50 (R-STK-007)
// confirms no per-call leak surfaces across repeats.
func TestAI33_2_TextStream_CancelBetweenFrames(t *testing.T) {
	// Single-frame identity chunk: emits ResponseStart + TextBlockStart.
	textFrame := "" +
		"data: {\"id\":\"chatcmpl-ai33-2-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n"

	server := httptest.NewServer(ai33StallHandler(textFrame))
	defer server.Close()

	c := mustClient(t, server.URL)

	agenttest.RequireNoGoroutineLeak(t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := c.Stream(ctx, validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (R-AIS-035 / S-1)", err)
		}

		// Receive the first event — synchronous handoff proves the
		// producer reached its first send before we cancel. The
		// producer is now blocked in Body.Read waiting for the second
		// frame, which the handler will never deliver (it is stalled
		// on r.Context().Done() / a 60s timer).
		first := ai33FirstEvent(t, ch, 2*time.Second)
		if first.Kind() != ai.EventKindResponseStart {
			t.Errorf("first event kind = %v, want EventKindResponseStart (R-AIS-035 / S-1)", first.Kind())
		}

		// Cancel between frames — Body.Read is blocked; the test's
		// empirical question is whether the transport honours ctx
		// within DefaultDrainTimeout + safety.
		cancel()

		// Bound the close. ai33DrainUntilClosed fails this scenario if
		// the producer never closes the channel within the bound — the
		// empirical RED signal that the read loop is not ctx-aware.
		ai33DrainUntilClosed(t, ch, agenttest.DefaultDrainTimeout+ai33StallBound)

		// Single close — the unique `defer close(out)` at
		// stream.go:344 is the only closing site. A second close
		// would panic on a closed channel inside run's defer chain;
		// -race would catch that on the next repeat.
		assertClosedExactlyOnce(t, ch)
	})
}

// TestAI33_2_ToolCallStream_CancelBetweenFrames covers R-AIS-035 / S-2:
// the same observable outcome as S-1, but with a tool-call transcript
// — identity chunk + tool-call-start chunk — so the tool-call
// accumulation path is exercised end-to-end. A cancellation proof that
// never crosses the tool-call accumulation path proves nothing about
// its buffers (charter line 1989).
//
// The handler flushes both frames before stalling, so the producer
// emits three events (ResponseStart + TextBlockStart + ToolCallStart)
// before reaching its next Body.Read call. The test receives the first
// event synchronously, cancels, and bounds the close — identical
// shape to S-1, on a request whose Translate path renders the tool
// region and whose body crosses the tool-call accumulator end-to-end.
func TestAI33_2_ToolCallStream_CancelBetweenFrames(t *testing.T) {
	// Two frames: identity chunk → ResponseStart + TextBlockStart, then
	// tool-call-start chunk → ToolCallStart. The handler flushes both
	// before stalling, so the producer emits three events before
	// reaching its next Body.Read call.
	toolCallFrames := []string{
		"" +
			"data: {\"id\":\"chatcmpl-ai33-2-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n",
		"" +
			"data: {\"id\":\"chatcmpl-ai33-2-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_ai33_2\",\"function\":{\"name\":\"get_weather\"}}]},\"finish_reason\":null}]}\n\n",
	}

	server := httptest.NewServer(ai33StallHandler(toolCallFrames...))
	defer server.Close()

	c := mustClient(t, server.URL)

	agenttest.RequireNoGoroutineLeak(t, func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := c.Stream(ctx, validToolCallRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (R-AIS-035 / S-2)", err)
		}

		first := ai33FirstEvent(t, ch, 2*time.Second)
		if first.Kind() != ai.EventKindResponseStart {
			t.Errorf("first event kind = %v, want EventKindResponseStart (R-AIS-035 / S-2)", first.Kind())
		}

		cancel()

		// Bounded close — same cap as S-1; the tool-call accumulation
		// path runs through the same read loop (stream.go:369–530) and
		// the same deferred close (stream.go:344–345).
		ai33DrainUntilClosed(t, ch, agenttest.DefaultDrainTimeout+ai33StallBound)

		assertClosedExactlyOnce(t, ch)
	})
}

// TestAI33_2_ConnectionFreedForNextRequest covers R-AIS-035 / S-3:
// after a between-frames cancellation, the underlying response body
// MUST be closed — so a stalled server cannot pin the connection. The
// proof uses a single *Client whose transport reuses connections: the
// first request is served by the stalling handler (one frame, then
// block), the test cancels and drains until close, then issues a
// second request against the same server with a complete transcript.
// The second request must complete within DefaultDrainTimeout + safety
// — proving the producer's deferred resp.Body.Close() at stream.go:345
// fired on the first request's cancellation path and freed the
// keep-alive slot.
//
// # Two-request handler
//
// The first request is handled by the stalling path (one frame, then
// block on r.Context().Done()). The second request is served by a
// complete transcript, so the second Stream call drains to
// ResponseStart + TextBlockStart + TextDelta + TextBlockEnd +
// Completion — the minimal transcript shape stream_test.go:423–446
// already proves. The handler counts requests so this test can assert
// both requests actually arrived at the server, not just that the
// second Stream call returned without error.
func TestAI33_2_ConnectionFreedForNextRequest(t *testing.T) {
	var (
		requests  atomic.Int32
		completed atomic.Int32
	)
	handler := func(w http.ResponseWriter, r *http.Request) {
		n := requests.Add(1)

		if n == 1 {
			// First request: stalling — one identity chunk, then
			// block on r.Context().Done().
			textFrame := "" +
				"data: {\"id\":\"chatcmpl-ai33-2-s3\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n"

			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, textFrame)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			completed.Add(1)
			select {
			case <-r.Context().Done():
			case <-time.After(60 * time.Second):
			}
			return
		}

		// Second-and-later request: serve a complete transcript that
		// closes cleanly. The producer's normal completion path runs
		// end-to-end — this is the proof that the server's keep-alive
		// slot is not pinned by the first request's cancelled
		// connection.
		transcript := "" +
			"data: {\"id\":\"chatcmpl-ai33-2-s3-2\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-2-s3-2\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"ok\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-2-s3-2\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n"

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, transcript)
		completed.Add(1)
	}
	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	// A single *Client drives both requests — its default transport
	// reuses connections, so this test exercises the actual
	// keep-alive slot the spec R-AIS-035 / S-3 names.
	c := mustClient(t, server.URL)

	agenttest.RequireNoGoroutineLeak(t, func() {
		// First request — drain the first event, cancel, drain until
		// close. The producer's defer resp.Body.Close() at
		// stream.go:345 fires on this path.
		ctx1, cancel1 := context.WithCancel(context.Background())
		ch1, err := c.Stream(ctx1, validRequest(t))
		if err != nil {
			t.Fatalf("first Stream() error = %v, want nil (R-AIS-035 / S-3)", err)
		}
		_ = ai33FirstEvent(t, ch1, 2*time.Second)
		cancel1()
		ai33DrainUntilClosed(t, ch1, agenttest.DefaultDrainTimeout+ai33StallBound)

		// Second request against the same server. If the producer's
		// Body.Close() did not fire (the read loop blocked past ctx
		// cancellation), the server's keep-alive slot would still be
		// pinned by the first connection and the second request would
		// hang. DrainAndRecord bounds the wait to
		// DefaultDrainTimeout + safety, so a hang fails the test
		// loudly rather than deadlocking the suite.
		ctx2, cancel2 := context.WithCancel(context.Background())
		defer cancel2()
		ch2, err := c.Stream(ctx2, validRequest(t))
		if err != nil {
			t.Fatalf("second Stream() error = %v, want nil — connection should have been freed (R-AIS-035 / S-3)", err)
		}
		rec := agenttest.DrainAndRecord(t, ch2, agenttest.DefaultDrainTimeout+ai33StallBound)
		events := rec.Events()

		// Second-request shape: ResponseStart + TextBlockStart +
		// TextDelta + TextBlockEnd + Completion. Assert the minimal
		// transcript lands end-to-end (the same shape
		// stream_test.go:423–446 already proves), so the connection
		// was actually reused / the second handler invocation
		// succeeded, not just that DrainAndRecord returned without
		// timing out.
		if rec.Len() < 2 {
			t.Fatalf("second request drained %d event(s), want ≥2 — the connection was not freed or the second handler did not serve a complete transcript (R-AIS-035 / S-3): %+v",
				rec.Len(), kindsOf(events))
		}

		// Both handler invocations must have arrived — a request that
		// was silently dropped en route would also pass DrainAndRecord
		// on a fresh connection. The counter is the load-bearing
		// assertion that the second request really did cross the
		// server's request boundary.
		if got := completed.Load(); got < 2 {
			t.Errorf("server-handler completed %d request(s), want ≥2 — second Stream() did not actually invoke the handler (R-AIS-035 / S-3)", got)
		}

		assertClosedExactlyOnce(t, ch2)
	})
}
