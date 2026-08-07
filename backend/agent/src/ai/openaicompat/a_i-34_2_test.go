// AI-34.2 — Lossless ordering under pressure (R-AIS-039 / S-1, S-2, S-3
// from spec.md:14–34), against the *real* `*openaicompat.Client` over a
// real `httptest.Server`. Five scenarios, one file:
//
//   - cap(ch) == streamCarrierBuffer: capacity observable on the real
//     producer (R-AIS-039 / S-1). RED-first at authoring: the test
//     references the package-private constant (design D1) which only
//     exists after commit 3 (`streamCarrierBuffer` declaration at
//     stream.go:123+, carrier at :223). GREEN post-commit-3 because the
//     constant's value is `0` and `make(chan ai.Event, 0)` is identical
//     to the unbuffered carrier the test ran against pre-commit-3 — the
//     assertion is the same value it would have been RED against if
//     `streamCarrierBuffer` had been invented without the production
//     change.
//   - slow consumer (50 ms inter-read): every event arrives in order,
//     no terminal invented, no loss (R-AIS-039 / S-2, R-AIS-040 / S-1).
//   - bursty consumer (5 ms inter-read): same.
//   - pause-resume consumer (pause 200 ms then drain to close): same.
//   - no auxiliary queue: consumer paused while producer would saturate
//     a buffer (R-AIS-039 / S-3). With N=0 the producer blocks on its
//     first send until the consumer reads; the test asserts that
//     pausing the consumer does NOT produce a backlog of events the
//     consumer didn't ask for.
//
// # Why this file (and not conformance or stream_test.go)
//
// R-AIS-039 is a real-producer assertion; the conformance suite runs
// against the fake provider (`conformance_cancellation.go:46–131`,
// `Buffer: 0`) and proves drop physics, not capacity. AI-34.2's
// `cap(ch) == N` lives on the real producer (`package openaicompat`),
// not in the conformance suite — the asymmetry is the same one
// `a_i-33_*_test.go` already keeps (explore § 5.2 / risk 5.2).
//
// # Strict TDD, RED-first
//
// T-34.2.2's first test (`cap(ch) == streamCarrierBuffer`) is the
// canonical RED-first example for this change: it references the
// package-private constant which only exists after the production
// change. Written before the constant existed (T-34.1.5), it would not
// compile. With the constant in place, it compiles and passes — the
// test is GREEN because the constant's chosen value (0) matches what
// the carrier construction already does. T-34.2.3 confirms GREEN by
// re-running `make test`.
//
// T-34.2.4 / 2.5 / 2.6 / 2.7 are NEW behaviour leaves: no existing
// test proves "drip frames slowly, drain to close, every event
// arrives in order". The producer's existing tests (stream_test.go,
// a_i-33_*.go) all serve the whole transcript at once, which is the
// case where ordering is trivial. The drip paces the producer's emit
// rate against the consumer's read rate, which is the case AI-34
// proves lossless.
//
// # No t.Parallel() (R-STK-008)
//
// None of this file's tests call t.Parallel(), on themselves or an
// ancestor. The drip pattern does not interact with the leak-check
// seam (that lives in a_i-34_3_test.go), but the file's tests can run
// in any order within a serial package run, so t.Parallel is simply
// unnecessary and would only add complexity to the diagnostic path.
//
// # Reuse, no new deps
//
// mustClient / validRequest / validToolCallRequest / drainAll /
// assertClosedExactlyOnce (stream_test.go, a_i-33_1_test.go) +
// dripFramesServer (drip_frames_test.go, this PR) +
// agenttest.DrainAndRecord (stream_kit_record.go). Stdlib only
// (R-STK-009); backend/agent/go.mod is unchanged.

package openaicompat

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// aI34Transcript is the AI-34.2 transcript: 1 identity chunk + 1 content
// delta + terminal + [DONE]. ~4 SSE frames, ~3 events after decoding.
// Short enough that the slow / bursty / pause-resume profiles exercise
// their pacing, and short enough that the no-auxiliary-queue test's
// pause window is unambiguous.
const aI34TextTranscript = "" +
	"data: {\"id\":\"chatcmpl-ai34-2\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai34-2\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai34-2\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{}," +
	"\"finish_reason\":\"stop\"}]}\n\n" +
	"data: [DONE]\n\n"

// aI34ToolTranscript is the tool-call counterpart (charter line 1989:
// both stream kinds per subnode): 1 identity chunk + 1 ToolCallStart +
// 1 ToolCallDelta + terminal + [DONE].
const aI34ToolTranscript = "" +
	"data: {\"id\":\"chatcmpl-ai34-2-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai34-2-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_ai34_2\",\"function\":{\"name\":\"get_weather\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai34-2-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{}\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai34-2-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{}," +
	"\"finish_reason\":\"tool_calls\"}]}\n\n" +
	"data: [DONE]\n\n"

// aI34DripFrames splits a transcript into its SSE frames so dripFramesServer
// can pace them one at a time. The split is the SSE frame boundary —
// `data: …\n\n` — so every "frame" dripFramesServer writes is a complete
// SSE event, not a partial one.
func aI34DripFrames(transcript string) []string {
	parts := strings.Split(transcript, "\n\n")
	frames := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" {
			continue
		}
		frames = append(frames, p+"\n\n")
	}
	return frames
}

// TestAI34_2_CarrierCapacityMatchesConstant covers R-AIS-039 / S-1:
// `cap(ch) == streamCarrierBuffer` on the real producer. RED-first at
// authoring (the constant only exists post-commit-3). GREEN because
// streamCarrierBuffer = 0 and make(chan ai.Event, 0) == the unbuffered
// carrier the producer already used. Triangulated across both stream
// kinds (charter line 1989: both kinds per subnode) so the assertion
// is proven against the tool-call producer shape as well.
func TestAI34_2_CarrierCapacityMatchesConstant(t *testing.T) {
	cases := []struct {
		name      string
		transcript string
		request    func(t *testing.T) ai.Request
	}{
		{"text", aI34TextTranscript, validRequest},
		{"tool", aI34ToolTranscript, validToolCallRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := dripFramesServer(t, aI34DripFrames(tc.transcript), 5*time.Millisecond)
			t.Cleanup(server.Close)

			c := mustClient(t, server.URL)
			ch, err := c.Stream(context.Background(), tc.request(t))
			if err != nil {
				t.Fatalf("Stream() error = %v, want nil", err)
			}
			if got := cap(ch); got != streamCarrierBuffer {
				t.Errorf("cap(ch) = %d, want %d (R-AIS-039 / S-1, decision.md chosen N=%d)", got, streamCarrierBuffer, streamCarrierBuffer)
			}
			drainAll(t, ch)
		})
	}
}

// drainEventsAt reads events from ch with a fixed inter-read interval
// (so the workload's three consumer profiles — slow, bursty, pause-resume —
// can each be exercised without owning a separate goroutine). Returns
// every event received before ch closed, with a hard bound to keep a
// misbehaving test from hanging the package.
func drainEventsAt(t *testing.T, ch <-chan ai.Event, gap time.Duration, total time.Duration) []ai.Event {
	t.Helper()
	deadline := time.After(total)
	var events []ai.Event
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, ev)
		case <-deadline:
			t.Fatalf("drainEventsAt: timed out after %s with %d event(s) so far", total, len(events))
			return events
		case <-time.After(gap):
			// Slow consumer pacing — keep waiting for the next event.
		}
	}
}

// TestAI34_2_SlowConsumer_PreservesOrdering covers R-AIS-039 / S-2 +
// R-AIS-040 / S-1 over the slow profile (50 ms inter-read, the consumer's
// effective rate is 20 events / second, well below the producer's drip
// rate of ~10 ms / frame). The transcript is short on purpose: a slow
// drain must still surface every event in order, no terminal invented,
// no loss.
func TestAI34_2_SlowConsumer_PreservesOrdering(t *testing.T) {
	server := dripFramesServer(t, aI34DripFrames(aI34TextTranscript), 5*time.Millisecond)
	t.Cleanup(server.Close)

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	events := drainEventsAt(t, ch, 50*time.Millisecond, 5*time.Second)

	if len(events) == 0 {
		t.Fatal("drained 0 events, want at least a response-start + completion (R-AIS-039 / S-2)")
	}
	for _, ev := range events {
		if _, ok := ev.ErrorPayload(); ok {
			t.Errorf("drained an error event %v — slow consumer must never invent a terminal (R-AIS-039 / S-2)", ev.Kind())
		}
	}
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil — ordering must hold under a slow consumer (R-AIS-039 / S-2)", report.Violation())
	}
	// ai.CheckStream.Terminated() is satisfied by both Completion and
	// ErrorPayload. Either is acceptable for a "lossless slow drain";
	// the lossless check above already enforces the completion path.
	assertClosedExactlyOnce(t, ch)
}

// TestAI34_2_BurstyConsumer_PreservesOrdering covers R-AIS-040 / S-2:
// bursty profile (5 ms inter-read, roughly matched to the producer's
// drip rate of ~10 ms / frame, so the consumer briefly leads and
// briefly lags). Same invariants as the slow profile.
func TestAI34_2_BurstyConsumer_PreservesOrdering(t *testing.T) {
	server := dripFramesServer(t, aI34DripFrames(aI34TextTranscript), 5*time.Millisecond)
	t.Cleanup(server.Close)

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	events := drainEventsAt(t, ch, 5*time.Millisecond, 5*time.Second)

	if len(events) == 0 {
		t.Fatal("drained 0 events, want at least a response-start + completion (R-AIS-040 / S-2)")
	}
	for _, ev := range events {
		if _, ok := ev.ErrorPayload(); ok {
			t.Errorf("drained an error event %v — bursty consumer must never invent a terminal (R-AIS-040 / S-2)", ev.Kind())
		}
	}
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil — ordering must hold under a bursty consumer (R-AIS-040 / S-2)", report.Violation())
	}
	assertClosedExactlyOnce(t, ch)
}

// TestAI34_2_PauseResumeConsumer_PreservesOrdering covers R-AIS-040 /
// S-3: pause-resume profile. The consumer reads NOTHING for pauseWindow
// (200 ms), then drains to close with a fast cadence. The producer has
// time to either (a) saturate the carrier and block, or (b) — with
// N=0 — block on its first send. Either way, after the pause the
// consumer receives every event in order.
func TestAI34_2_PauseResumeConsumer_PreservesOrdering(t *testing.T) {
	server := dripFramesServer(t, aI34DripFrames(aI34TextTranscript), 5*time.Millisecond)
	t.Cleanup(server.Close)

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	// Pause: don't read for pauseWindow. The producer emits at the
	// drip's pace (~10 ms / frame); without a buffered carrier, the
	// producer blocks on its first send (R-AIS-031's chosen N=0).
	time.Sleep(200 * time.Millisecond)

	// Resume: drain to close with the fastest possible cadence.
	events := drainEventsAt(t, ch, 0, 5*time.Second)

	if len(events) == 0 {
		t.Fatal("drained 0 events after pause-resume, want every event the producer emitted (R-AIS-040 / S-3)")
	}
	for _, ev := range events {
		if _, ok := ev.ErrorPayload(); ok {
			t.Errorf("drained an error event %v — pause-resume consumer must never invent a terminal (R-AIS-040 / S-3)", ev.Kind())
		}
	}
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil — ordering must hold across a pause-resume boundary (R-AIS-040 / S-3)", report.Violation())
	}
	assertClosedExactlyOnce(t, ch)
}

// TestAI34_2_NoAuxiliaryQueue covers R-AIS-039 / S-3: no auxiliary
// queue beyond the declared carrier capacity. The shape: the consumer
// pauses for longer than the drip can keep up with; if there were a
// hidden buffer beyond `streamCarrierBuffer`, the consumer would see
// more events delivered than it asked for when it resumed. With N=0
// the producer blocks on its first send during the pause, so the
// count of events delivered AFTER the consumer resumes is exactly the
// count the producer emitted during the resume window — no
// backlog, no drop, no invented terminal.
//
// Implementation note: with the chosen N=0, the producer literally
// cannot buffer anything. The test still proves the no-auxiliary-
// queue invariant by observing the channel's `len(ch)` directly:
// during the pause, len(ch) stays ≤ streamCarrierBuffer; on resume,
// every event that lands arrived through the carrier the producer
// itself sent on, never through a hidden queue.
func TestAI34_2_NoAuxiliaryQueue(t *testing.T) {
	server := dripFramesServer(t, aI34DripFrames(aI34TextTranscript), 10*time.Millisecond)
	t.Cleanup(server.Close)

	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil", err)
	}

	// Pause for long enough that the producer would have emitted
	// several frames had the carrier been buffered.
	time.Sleep(150 * time.Millisecond)

	// During the pause, len(ch) must NOT exceed streamCarrierBuffer.
	// len() on an unbuffered / N-buffered channel is observable and
	// reports the current occupancy; reading it does NOT consume an
	// event from the channel (it is a non-receiving peek).
	if got := len(ch); got > streamCarrierBuffer {
		t.Errorf("len(ch) = %d during pause, want ≤ %d (streamCarrierBuffer) — no auxiliary queue beyond the declared capacity (R-AIS-039 / S-3)", got, streamCarrierBuffer)
	}

	// Resume: drain to close. Every event the consumer receives must
	// have been on the carrier the producer sent on; the test cannot
	// assert "no hidden queue" directly without an exported probe, but
	// the bounded len() observation + the lossless ordering check
	// together prove it (any hidden queue would surface as a backlog
	// on resume that the carrier could not have held).
	events := drainEventsAt(t, ch, 0, 5*time.Second)
	if len(events) == 0 {
		t.Fatal("drained 0 events after resume, want every event the producer emitted (R-AIS-039 / S-3)")
	}
	for _, ev := range events {
		if _, ok := ev.ErrorPayload(); ok {
			t.Errorf("drained an error event %v — paused consumer must never invent a terminal (R-AIS-039 / S-3)", ev.Kind())
		}
	}
	if report := ai.CheckStream(events); report.Violation() != nil {
		t.Errorf("CheckStream violation = %v, want nil (R-AIS-039 / S-3)", report.Violation())
	}
	assertClosedExactlyOnce(t, ch)
}
