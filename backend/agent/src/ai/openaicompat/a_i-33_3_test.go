// AI-33.3 — Truly-abandoned consumer + cancellation drops cleanly, with no
// terminal invented (R-AIS-036 / S-1, S-2, S-3).
//
// # R-CNF-012, quoted verbatim (the wording this subnode is pinned to)
//
// From openspec/specs/ai-provider-conformance-suite/spec.md, R-CNF-012:
//
//	"The suite MUST assert AI-20.3's saturated-drop physics: a consumer that
//	stops reading until the buffer saturates, whose caller then cancels, MUST
//	see the stream close bare — no terminal event invented, no undelivered
//	event forced through, no leak. The suite MUST NOT assert the
//	abandoned-never-cancelled path; that narrowing is inherited from
//	ai-stream-lifecycle § 5's untestability ruling and is recorded here rather
//	than left silently absent."
//
// The quote is reproduced here, not paraphrased, so that any future drift
// between this file and the conformance requirement is a diff a reviewer can
// see rather than a divergence that has to be re-derived (proposal § 7's open
// decision, tasks.md risk R2).
//
// The conformance suite already proves those physics over the FAKE provider
// (agenttest/conformance_cancellation.go:94, whose produce() selects on
// `out <- ev` vs `<-ctx.Done()`). What was missing, and what this file adds,
// is the same proof over the REAL producer and a REAL HTTP transport.
//
// # Why the observation must never read before emitFailureSendBound expires
//
// `out` is unbuffered (stream.go:209), so a consumer that never reads leaves
// the producer blocked on its very first send. Cancellation makes emit lose
// the `out <- stamped` vs `<-ctx.Done()` race (stream.go:557–565) and return
// false, at which point AI-32.3's bounded-wait branch (stream.go:437–470,
// mirrored for the read side at :503–526) tries once more to hand a typed
// terminal over, this time selecting against `time.After(emitFailureSendBound)`
// rather than against ctx.
//
// That send is a rendezvous. A test that receives while it is still pending
// would PAIR with it and take delivery of the terminal — which is
// perturbation, not observation, and would report "a terminal was invented"
// when in truth the terminal only landed because the test stopped abandoning.
// (Measured: receiving 100ms after cancel yields an `error` event; receiving
// 5.5s after cancel yields a bare close.) So the truly-abandoned observation
// is: read NOTHING until the bound has provably expired, then let the FIRST
// receive report what happened. `ok == false` on that first receive is the
// bare close itself — the terminal was dropped, never invented.
//
// The slow-but-alive consumer, whose read DOES pair with that bounded send
// and who therefore DOES observe the typed terminal, is AI-32.3's own
// observable and is explicitly out of scope here (spec "Out of scope":
// "Slow-but-alive consumer mid-send ... Out of AI-33.3").
//
// # Leak posture, and a recorded deviation
//
// R-AIS-036 / S-1 attaches "RequireNoGoroutineLeak (50 repeats)" to the
// truly-abandoned scenario. Taken literally that costs 50 × emitFailureSendBound
// = 4m10s PER scenario (8m20s for both stream kinds) against a package that
// otherwise runs in ~8s and a `go test` default timeout of 10m — and AI-33.5 /
// S-6 is specified to reuse this scenario, which would push the package over
// that timeout outright. So the two obligations are proven separately:
//
//   - no terminal invented — TestAI33_3_TextStream_AbandonedThenCancelled and
//     its tool-call sibling, each one full-fidelity truly-abandoned run. Each
//     also proves the producer goroutine terminated, because `close(out)` is
//     run's OUTERMOST defer (stream.go:344): observing the close IS observing
//     that run returned.
//   - no leak across repeats — TestAI33_3_AbandonedThenCancelled_LeakFree,
//     RequireNoGoroutineLeak × 50 (R-STK-007) over the same abandoned-then-
//     cancelled shape the conformance case uses (cancel, THEN drain), which
//     costs milliseconds per repeat and detects an accumulating per-call leak
//     with the same amplitude sensitivity.
//
// # Serial only (R-STK-008)
//
// No test in this file calls t.Parallel(), on itself or an ancestor:
// RequireNoGoroutineLeak's count is process-wide and meaningless under
// parallel execution.
//
// # Reuse, and no new dependencies
//
// serveTranscripts (bridge_test.go:98), mustClient / validRequest /
// assertClosedExactlyOnce (stream_test.go:50, :74, :168) and
// validToolCallRequest (a_i-33_1_test.go:200) are all same-package, internal,
// already-established fixtures. agenttest contributes only
// RequireNoGoroutineLeak and DrainAndRecord, both stdlib-only (R-STK-009).
// backend/agent/go.mod is unchanged. The conformance suite is untouched.

package openaicompat

import (
	"context"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// ai33AbandonWindow is how long the consumer stays truly abandoned before it
// looks at the channel at all. It MUST exceed emitFailureSendBound: the whole
// point is to let AI-32.3's bounded-wait send expire unpaired, so the first
// receive observes the drop rather than causing a delivery.
const ai33AbandonWindow = emitFailureSendBound + 500*time.Millisecond

// ai33CloseObserveBound caps the first receive once the abandonment window has
// elapsed. By then the producer has already returned and closed, so this is a
// liveness backstop, not a wait: it fails loudly if the stream is still open.
const ai33CloseObserveBound = 2 * time.Second

// ai33TextTranscript is a minimal well-formed text stream: identity chunk,
// one content delta, terminal chunk with finish_reason "stop", [DONE].
const ai33TextTranscript = "" +
	"data: {\"id\":\"chatcmpl-ai33-3-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-3-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hi\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-3-text\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
	"data: [DONE]\n\n"

// ai33ToolTranscript is the tool-call counterpart (charter line 1989 requires
// both stream kinds per subnode): identity chunk, tool-call start, arguments
// delta, terminal chunk with finish_reason "tool_calls", [DONE].
const ai33ToolTranscript = "" +
	"data: {\"id\":\"chatcmpl-ai33-3-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-3-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_ai33_3\",\"function\":{\"name\":\"get_weather\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-3-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{}\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-ai33-3-tool\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
	"data: [DONE]\n\n"

// ai33AbandonAndCancel stands up a real httptest server over transcript, opens
// a real stream through a real *Client, and cancels immediately — WITHOUT ever
// receiving. The returned channel is left untouched: the producer is blocked
// on its first send from the moment it has a frame to emit.
func ai33AbandonAndCancel(t *testing.T, transcript string, req ai.Request) <-chan ai.Event {
	t.Helper()

	server := httptest.NewServer(serveTranscripts([][]byte{[]byte(transcript)}))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := mustClient(t, server.URL).Stream(ctx, req)
	if err != nil {
		cancel()
		t.Fatalf("Stream() error = %v, want nil — the abandonment must happen on a live stream (R-AIS-036)", err)
	}

	cancel() // abandoned from the start: no receive has happened, none will
	return ch
}

// ai33RequireBareClose is the truly-abandoned observation. It reads nothing
// until ai33AbandonWindow has elapsed, then requires the FIRST receive to
// report a closed channel: no completion, no error event, nothing at all.
func ai33RequireBareClose(t *testing.T, ch <-chan ai.Event, scenario string) {
	t.Helper()

	time.Sleep(ai33AbandonWindow)

	select {
	case ev, ok := <-ch:
		if ok {
			t.Fatalf("%s: first receive after the abandonment window returned %v, want a closed channel — "+
				"an abandoned-then-cancelled stream must close bare, with no terminal invented and no undelivered event forced through (R-CNF-012, R-AIS-036)",
				scenario, ev.Kind())
		}
	case <-time.After(ai33CloseObserveBound):
		t.Fatalf("%s: channel still open %s after cancellation, want a bare close within emitFailureSendBound (%s) + safety (R-AIS-036, R-STK-029)",
			scenario, ai33AbandonWindow, emitFailureSendBound)
	}
}

// TestAI33_3_TextStream_AbandonedThenCancelled covers R-AIS-036 / S-1: a real
// text transcript over a real HTTP transport, a consumer that never reads, and
// a cancel that lands while the producer is blocked on its first send. The
// stream closes bare within emitFailureSendBound + safety; no ai.Completion
// and no ai.ErrorEvent is observed by any future reader.
//
// The observed close is also the proof the producer goroutine terminated:
// close(out) is run's outermost defer (stream.go:344), so it cannot fire while
// run is still on the stack.
func TestAI33_3_TextStream_AbandonedThenCancelled(t *testing.T) {
	ch := ai33AbandonAndCancel(t, ai33TextTranscript, validRequest(t))

	ai33RequireBareClose(t, ch, "R-AIS-036 / S-1 (text)")

	// The close is stable and singular: every later receive still reports the
	// closed form, never a late event and never a panic (R-CNF-009's single
	// closing site).
	assertClosedExactlyOnce(t, ch)
}

// TestAI33_3_ToolCallStream_AbandonedThenCancelled covers R-AIS-036 / S-2: the
// same observable outcome over a tool-call transcript. The tool-call variant
// matters because the producer's failure path closes still-open tool blocks
// (stream.go:610–619) through the plain, ctx-selecting emit — so an abandoned
// consumer must see those dropped too, not forced through.
func TestAI33_3_ToolCallStream_AbandonedThenCancelled(t *testing.T) {
	ch := ai33AbandonAndCancel(t, ai33ToolTranscript, validToolCallRequest(t))

	ai33RequireBareClose(t, ch, "R-AIS-036 / S-2 (tool-call)")

	assertClosedExactlyOnce(t, ch)
}

// TestAI33_3_AbandonedThenCancelled_LeakFree is R-AIS-036's leak clause
// (R-STK-007's amplitude mechanism, 50 repeats): the abandoned-then-cancelled
// shape must not leak a goroutine per call. The scenario mirrors the
// conformance case's own ordering — Stream, cancel with no prior receive, then
// drain (agenttest/conformance_cancellation.go:113–120) — which lets each
// repeat finish in milliseconds instead of waiting out emitFailureSendBound.
//
// The drain here does pair with AI-32.3's bounded-wait send, so this scenario
// deliberately asserts NOTHING about terminals; that is what the two
// truly-abandoned tests above are for. What it asserts is the leak amplitude.
func TestAI33_3_AbandonedThenCancelled_LeakFree(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {
		server := httptest.NewServer(serveTranscripts([][]byte{[]byte(ai33TextTranscript)}))
		defer server.Close()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ch, err := mustClient(t, server.URL).Stream(ctx, validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (R-AIS-036 leak clause)", err)
		}

		cancel() // abandoned: no receive has happened yet

		agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout)
	})
}

// TestAI33_3_AbandonedNeverCancelledPathNotAsserted covers R-AIS-036 / S-3.
// The abandoned-NEVER-cancelled path is ruled untestable to termination by
// ai-stream-lifecycle § 5 — "no test proves a goroutine never exits, and a
// bounded observation that it has not exited yet is a strictly weaker claim
// that would be mistaken for the stronger one" — and R-CNF-012 records that
// narrowing rather than leaving it silently absent.
//
// This is the mechanical form of that record: the absence is asserted, so a
// future contributor who adds such a test discovers the ruling here instead of
// shipping a weaker claim dressed as a stronger one.
func TestAI33_3_AbandonedNeverCancelledPathNotAsserted(t *testing.T) {
	const selfName = "TestAI33_3_AbandonedNeverCancelledPathNotAsserted"

	files, err := filepath.Glob("a_i-33_*_test.go")
	if err != nil {
		t.Fatalf("globbing the AI-33 test files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no a_i-33_*_test.go files found to scan — this guard must never pass vacuously (R-AIS-036 / S-3)")
	}

	testDecl := regexp.MustCompile(`func (Test\w+)\(`)
	neverCancelled := regexp.MustCompile(`(?i)never_?cancel`)

	declared := 0
	for _, path := range files {
		src, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatalf("reading %s: %v", path, readErr)
		}
		for _, m := range testDecl.FindAllStringSubmatch(string(src), -1) {
			declared++
			name := m[1]
			if name != selfName && neverCancelled.MatchString(name) {
				t.Errorf("%s declares %s: the abandoned-never-cancelled path must not be asserted — "+
					"ai-stream-lifecycle § 5 rules it untestable to termination and R-CNF-012 inherits that narrowing (R-AIS-036 / S-3, R-STK-010)",
					path, name)
			}
		}
	}

	if declared == 0 {
		t.Fatalf("scanned %d AI-33 test file(s) but matched no test declaration — the scan is not exercising the guard (R-AIS-036 / S-3)", len(files))
	}
	t.Logf("scanned %d AI-33 test file(s), %d test declaration(s): abandoned-never-cancelled remains unasserted, on the record (S-CNF-031, R-STK-010)", len(files), declared)
}
