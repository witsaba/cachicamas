// AI-33.4 — Cancel after completion (R-AIS-037 / S-1, S-2, S-3).
//
// The trivially-correct cancellation moment, proven against a real HTTP
// transport. The producer's run() emits the completion event
// (stream.go:393), the unique `defer close(out)` at stream.go:344 fires
// on return (stream.go:394), and the producer goroutine is gone. So a
// cancel() that arrives after the terminal event is a no-op for the
// producer (already exited) and a no-op for the consumer (the channel
// it would receive from is already closed). What the AI-33.4 test must
// prove under -race:
//
//   1. No interleaving panic in the consumer's drain loop.
//   2. Exactly one close of `out` (the unique `defer close(out)` at
//      stream.go:344 is the only closing site — single-source-of-truth
//      for the close; R-CNF-009 cites this).
//   3. No goroutine leak under -race (R-STK-007 mechanism, repeated
//      50 times by RequireNoGoroutineLeak; R-STK-008 keeps this file's
//      own tests serial, hence no t.Parallel() anywhere here).
//
// Both stream kinds, per charter line 1989 ("every node runs over text
// AND tool-call streams"). The tool-call variant must close cleanly
// even with a tool-call block mid-accumulation, otherwise the
// tool-call close chain at stream.go:610–619 would surface as a leaked
// or duplicated ToolCallEnd.
//
// # Note on fixture selection
//
// The spec cites bridgeServeTranscripts (openrouter/conformance/
// bridge_test.go:141) as the verbatim transcript server. That helper
// is unexported and lives in package openrouter_conformance, which is
// not importable from this internal test file (separate Go package
// under the same module). The shape-identical, same-package helper
// serveTranscripts (bridge_test.go:98 in this same package) is used
// instead — it serves transcripts[n] to the (n+1)th inbound request,
// is the seam conformanceBridgeFactory itself already uses
// (bridge_test.go:75), and is the established test-only fixture for
// real-HTTP AI-33 work in this package. The conformance suite remains
// untouched (constraint: do NOT touch the conformance suite).
//
// # Pins
//
// R-CNF-009 (single closing site; the `defer close(out)` invariant
// this file's per-test assertClosedExactlyOnce call re-asserts);
// R-CNF-011 (conformance bounded-close invariant; this file's
// cancellation-after-completion scenario stays compatible because
// the conformance case still sees exactly one event after cancel and
// the conformance suite here is unchanged — see
// constraint `do NOT touch the conformance suite`).
//
// # No new dependencies, no conformance changes
//
// This file reuses stream_test.go's mustClient and validRequest, the
// validToolCallRequest builder a_i-33_1_test.go:200 already exposes
// (same package, internal), bridge_test.go:98's serveTranscripts
// (same package, internal), stream_test.go:168's assertClosedExactlyOnce
// and :190's requireCheckStreamClean (same package, internal), and
// agenttest.RequireNoGoroutineLeak + agenttest.DrainAndRecord
// (stdlib-only, R-STK-009). backend/agent/go.mod is unchanged
// (R-STK-009's mechanical proof; spec acceptance criterion 7).

package openaicompat

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// TestAI33_4_TextStream_CancelAfterCompletion covers R-AIS-037 / S-1:
// a real text transcript ending in [DONE] drains to a clean
// completion; the producer's `defer close(out)` fires
// (stream.go:344); cancel() is called after the drain — by then the
// producer goroutine is gone, so cancel() has nothing to signal and
// no observable effect is observed. The 50-repeat
// RequireNoGoroutineLeak loop wraps the scenario so any future
// refactor that grew a per-call leak surfaces here.
func TestAI33_4_TextStream_CancelAfterCompletion(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		// Minimal well-formed text transcript: identity chunk (role
		// only), one content delta, terminal chunk carrying
		// finish_reason:"stop", and the [DONE] sentinel. Mirrors
		// terminal_test.go:53–55's own witness transcript — the same
		// bytes already proven to land a clean Completion under the
		// unmodified stream.go.
		transcript := "" +
			"data: {\"id\":\"chatcmpl-ai33-4-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-4-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-4-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n"

		server := httptest.NewServer(serveTranscripts([][]byte{[]byte(transcript)}))
		defer server.Close()

		c := mustClient(t, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := c.Stream(ctx, validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (R-AIS-037 / S-1)", err)
		}

		// DrainAndRecord fails the test on deadline (a stuck producer
		// would surface here), and returns the full recorded slice on a
		// clean close. The producer is long since exited by the time
		// this returns — the channel closed via the unique defer at
		// stream.go:344.
		rec := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		events := rec.Events()

		// cancel() now — a no-op for the (already-exited) producer.
		// Any future change that introduces a producer-side watcher
		// racing against this cancel will be caught by -race on this
		// scenario's 50 repeats, satisfying R-AIS-037 / S-1's "no
		// interleaving panic" clause on the next race-detector
		// break.
		cancel()

		// Exactly one terminal event (Completion, never Error). The
		// conformance suite's R-CNF-011 already proves this shape
		// over the fake provider; this test proves the real producer
		// matches it on the after-completion path.
		terminals := 0
		for _, ev := range events {
			kind := ev.Kind()
			if kind == ai.EventKindCompletion || kind == ai.EventKindError {
				terminals++
			}
		}
		if terminals != 1 {
			t.Fatalf("terminal event count = %d, want exactly 1 (R-AIS-037 / S-1): %+v",
				terminals, kindsOf(events))
		}

		last := events[len(events)-1]
		if last.Kind() != ai.EventKindCompletion {
			t.Errorf("last event kind = %v, want EventKindCompletion (R-AIS-037 / S-1)", last.Kind())
		}

		// The unique `defer close(out)` at stream.go:344 fired exactly
		// once. A second `close(out)` would panic on a closed channel
		// inside run's defer chain; -race would catch that on the next
		// repeat. assertClosedExactlyOnce re-asserts the closed-channel
		// observation on every subsequent receive, so a duplicate
		// close (which would never occur) would not register as a pass
		// here.
		assertClosedExactlyOnce(t, ch)
	})
}

// TestAI33_4_ToolCallStream_CancelAfterCompletion covers R-AIS-037 / S-2:
// a real tool-call transcript ending in [DONE] drains to a clean
// completion. The tool-call accumulation path is in play: when [DONE]
// lands the producer's mapper has accumulated the open tool-call
// block, and the completion emit at stream.go:393 is what closes the
// whole thing out cleanly. The contract under test: cancel() after
// drain is a no-op even with a tool-call block mid-accumulation.
//
// The tool-call close-chain (stream.go:610–619) sits inside the
// emitFailure path and is not on the success-path Completion, so this
// scenario does not exercise it directly. requireCheckStreamClean
// still asserts AI-14.4's invariants — every block open on the
// carrier closed before the terminal Completion — which is the
// shape-only proof the success path honors AI-14.4's invariants.
func TestAI33_4_ToolCallStream_CancelAfterCompletion(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		// Minimal well-formed tool-call transcript: identity chunk
		// (role only), one tool start (id+name, no arguments — that's
		// how the wire encodes a tool-call Start per R-ATC-002), one
		// tool delta (arguments fragment), terminal chunk carrying
		// finish_reason:"tool_calls", and the [DONE] sentinel.
		transcript := "" +
			"data: {\"id\":\"chatcmpl-ai33-4-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-4-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_ai33_4\",\"function\":{\"name\":\"get_weather\"}}]},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-4-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{}\"}}]},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-4-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
			"data: [DONE]\n\n"

		server := httptest.NewServer(serveTranscripts([][]byte{[]byte(transcript)}))
		defer server.Close()

		c := mustClient(t, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := c.Stream(ctx, validToolCallRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (R-AIS-037 / S-2)", err)
		}

		rec := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		events := rec.Events()

		cancel()

		terminals := 0
		for _, ev := range events {
			kind := ev.Kind()
			if kind == ai.EventKindCompletion || kind == ai.EventKindError {
				terminals++
			}
		}
		if terminals != 1 {
			t.Fatalf("terminal event count = %d, want exactly 1 (R-AIS-037 / S-2): %+v",
				terminals, kindsOf(events))
		}

		last := events[len(events)-1]
		if last.Kind() != ai.EventKindCompletion {
			t.Errorf("last event kind = %v, want EventKindCompletion (R-AIS-037 / S-2)", last.Kind())
		}

		// Success-path block invariants — ToolCallStart and the
		// matching ToolCallEnd both land before the Completion,
		// otherwise requireCheckStreamClean would record the violation.
		// The cleanup-on-error emitFailure path
		// (stream.go:610–619) is exercised by the AI-33.3 tests; this
		// scenario is the after-completion equivalent and does not
		// itself need the cleanup path.
		requireCheckStreamClean(t, events)

		assertClosedExactlyOnce(t, ch)
	})
}

// TestAI33_4_RaceCancelConcurrentWithFinalReceive covers R-AIS-037 / S-3:
// the cancel() call races the consumer's drain. 50 repeats under
// -race (RequireNoGoroutineLeak × 50), each repeat a fresh server and
// a fresh Stream call, so any interleaving panic between the cancel
// goroutine and the drain loop surfaces across many shuffles, not
// just one. The observable contract — exactly one terminal event,
// the channel closed exactly once, no panic, no leak — holds either
// way, on whichever interleaving the race detector happens to
// schedule this repeat.
//
// Two outcomes are valid for the terminal:
//
//   - Completion: cancel() lost the race against the producer, the
//     [DONE] sentinel landed first, run() returned via stream.go:394,
//     and cancel() afterwards is a no-op (producer is gone).
//   - Error (AI-32.3 bounded-wait at stream.go:437–470 / 503–526):
//     cancel() won the race, the producer's read loop noticed
//     ctx.Err() != nil, and the bounded-wait terminal emitted a
//     typed Failure (Category=Cancellation, Delivery=MidStream)
//     before run() returned.
//
// Both are AI-32.3's documented, sanctioned behaviour; both close
// the channel exactly once via the unique `defer close(out)` at
// stream.go:344 (R-CNF-009). The spec's S-3 wording ("exactly one
// terminal is observed, the channel is closed exactly once, no panic
// occurs") admits both outcomes — neither one is a regression. What
// -race coverage actually rules out is any interleaving panic under
// the goroutine shuffle the race detector happens to schedule.
//
// The race is genuine: cancel() may fire while DrainAndRecord is
// mid-receive (the producer is interrupted), or after DrainAndRecord
// returns (the producer has already exited). Both schedules produce
// the same observable contract. -race coverage is what turns "this
// happens to work" into "this provably does not interleave-panic
// across many iterations."
func TestAI33_4_RaceCancelConcurrentWithFinalReceive(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		transcript := "" +
			"data: {\"id\":\"chatcmpl-ai33-4-race\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-4-race\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"bye\"},\"finish_reason\":null}]}\n\n" +
			"data: {\"id\":\"chatcmpl-ai33-4-race\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
			"data: [DONE]\n\n"

		server := httptest.NewServer(serveTranscripts([][]byte{[]byte(transcript)}))
		defer server.Close()

		c := mustClient(t, server.URL)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel() // safety net; the racing cancel is the goroutine below

		ch, err := c.Stream(ctx, validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (R-AIS-037 / S-3)", err)
		}

		// Race: cancel() fires in its own goroutine, no delay, no
		// coordination. The producer's exit (stream.go:344 defer
		// fires, run() returns) and the consumer's drain are both in
		// flight simultaneously with the cancel call — -race
		// exercises every interleaving.
		go cancel()

		rec := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
		events := rec.Events()

		terminals := 0
		for _, ev := range events {
			kind := ev.Kind()
			if kind == ai.EventKindCompletion || kind == ai.EventKindError {
				terminals++
			}
		}
		if terminals != 1 {
			t.Fatalf("terminal event count = %d, want exactly 1 (R-AIS-037 / S-3): %+v",
				terminals, kindsOf(events))
		}
		// Last event kind is either Completion (cancel lost the race
		// against the producer) or Error (AI-32.3 bounded-wait at
		// stream.go:437–470 / 503–526 — cancel won the race). Both are
		// valid under S-3's spec wording; the assertion above (exactly
		// one terminal) is the load-bearing one, this is a sanity
		// confirms that whatever terminal landed is the LAST one, not
		// some trailing event after it.
		last := events[len(events)-1]
		if last.Kind() != ai.EventKindCompletion && last.Kind() != ai.EventKindError {
			t.Errorf("last event kind = %v, want EventKindCompletion or EventKindError — terminal must be the last event (R-AIS-037 / S-3)", last.Kind())
		}

		assertClosedExactlyOnce(t, ch)
	})
}
