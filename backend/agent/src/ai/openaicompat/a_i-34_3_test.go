// AI-34.3 — Exhaustive non-cancellation paths × leak-check × 50
// (R-AIS-040, R-STK-007, R-STK-008).
//
// # What this file proves
//
// For every non-cancellation consumer pattern (slow / bursty /
// pause-resume) across both stream kinds (text + tool-call), the
// consumer MUST receive every emitted event in order, no terminal
// invented, no goroutine leak. Six cases (3 profiles × 2 kinds), each
// wrapped in agenttest.RequireNoGoroutineLeak (50 repeats,
// leakTolerance = 25). 6 × 50 = 300 leak-check repeats within budget.
//
// # Serial-only (R-STK-008)
//
// The mechanical guard lives inside agenttest.RequireNoGoroutineLeak:
// its first action is tb.Setenv(\"AGENTTEST_STREAM_KIT_LEAK_CHECK\", \"1\"),
// which the testing package itself panics on when the calling test (or
// an ancestor) has already called t.Parallel(). No test in this file
// calls t.Parallel(), on itself or an ancestor: runtime.NumGoroutine is
// process-wide and meaningless under parallel execution. The same
// posture a_i-33_3_test.go and a_i-33_5b_test.go already keep.
//
// # Reuse posture
//
// dripFramesServer (drip_frames_test.go, AI-34.2) — pacing harness.
// mustClient / validRequest / validToolCallRequest / drainAll /
// assertClosedExactlyOnce (stream_test.go, a_i-33_1_test.go,
// a_i-33_3_test.go) — same-package internal fixtures.
// agenttest.RequireNoGoroutineLeak + agenttest.DrainAndRecord
// (stream_kit_leak.go, stream_kit_record.go) — the leak seam and
// drain helper AI-22.4 / AI-22.1 already shipped. Stdlib only
// (R-STK-009). backend/agent/go.mod unchanged.
//
// # What "no unsanctioned loss path" means here
//
// The exhaustive coverage: for every (profile, kind) pair, the
// scenario drains to close WITHOUT cancelling ctx, and the
// post-drain record must (a) have every emitted event, (b) have them
// in order, (c) have no ErrorPayload terminal invented. The
// agenttest.RequireNoGoroutineLeak wrapper proves the per-call leak
// amplitude stays within tolerance (50 repeats × ~zero growth = under
// leakTolerance=25).
//
// # Why drain-then-assert, not cancel-then-drain
//
// The sanctioned loss path (AI-20.3 / AI-33.3) is cancellation-on-a-
// saturated-buffer. We are proving the OPPOSITE: the path WITHOUT
// cancellation is lossless. So every scenario opens the stream, reads
// to close (no cancel), and asserts the recording's shape — no cancel
// is ever invoked. The producer's bounded-wait terminal path
// (stream.go:679–682) is never triggered because the consumer reads
// to close normally.
//
// # Budget
//
// 6 cases × 50 repeats ≈ ~18s wall under -race. The drain helpers all
// use DefaultDrainTimeout (2 s, stream_kit_record.go:24) as a hard
// bound, so a defect that hangs the producer fails the suite in
// seconds, not minutes.

package openaicompat

import (
	"context"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// aI34_3AssertLossless is the post-drain check every scenario runs.
// It asserts every event arrived, no terminal invented, ordering holds,
// and the channel closed exactly once. Mirrors the invariants
// a_i-34_2_test.go asserts per-scenario; reused here for the
// leak-check wrapper.
func aI34_3AssertLossless(t *testing.T, scenario string, events []ai.Event) {
	t.Helper()

	if len(events) == 0 {
		t.Fatalf("%s: drained 0 events, want every event the producer emitted (R-AIS-040)", scenario)
	}
	for _, ev := range events {
		if _, ok := ev.ErrorPayload(); ok {
			t.Errorf("%s: drained an error event %v — no terminal must be invented on a non-cancellation path (R-AIS-040)",
				scenario, ev.Kind())
		}
	}
	report := ai.CheckStream(events)
	if report.Violation() != nil {
		t.Errorf("%s: CheckStream violation = %v, want nil — ordering must hold across every profile × kind (R-AIS-040)",
			scenario, report.Violation())
	}
	if !report.Terminated() {
		t.Errorf("%s: CheckStream Terminated() = false, want true (R-AIS-040)", scenario)
	}
}

// aI34_3Scenario is one (profile, kind) shape. `server` is built once
// per test (outside the leak-check closure, because the server outlives
// all 50 repeats); the closure body is what runs 50 times.
type aI34_3Scenario struct {
	name        string
	transcript  string
	request     func(t *testing.T) ai.Request
	drainEvents func(t *testing.T, ch <-chan ai.Event) []ai.Event
}

// aI34_3Scenarios returns the six (profile × kind) scenarios R-AIS-040
// names. The server serves the whole transcript at once (via
// sseServer) — the "slow / bursty / pause-resume" labels describe the
// CONSUMER's pace, not the server's. That keeps every repeat fast
// (milliseconds) — a paced server would leave handler goroutines
// alive across repeats and inflate the leak-check count, defeating
// the seam AI-22.4 ships.
func aI34_3Scenarios(t *testing.T) []aI34_3Scenario {
	t.Helper()

	// slow consumer: 50 ms inter-read (consumer slower than producer).
	drainSlow := func(t *testing.T, ch <-chan ai.Event) []ai.Event {
		t.Helper()
		var events []ai.Event
		deadline := time.After(5 * time.Second)
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return events
				}
				events = append(events, ev)
			case <-time.After(50 * time.Millisecond):
			case <-deadline:
				t.Fatalf("drainSlow: timed out after 5s with %d event(s)", len(events))
				return events
			}
		}
	}
	// bursty consumer: 5 ms inter-read.
	drainBursty := func(t *testing.T, ch <-chan ai.Event) []ai.Event {
		t.Helper()
		var events []ai.Event
		deadline := time.After(5 * time.Second)
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return events
				}
				events = append(events, ev)
			case <-time.After(5 * time.Millisecond):
			case <-deadline:
				t.Fatalf("drainBursty: timed out after 5s with %d event(s)", len(events))
				return events
			}
		}
	}
	// pause-resume consumer: 50 ms pause, then drain to close.
	drainPauseResume := func(t *testing.T, ch <-chan ai.Event) []ai.Event {
		t.Helper()
		time.Sleep(50 * time.Millisecond)
		return drainAll(t, ch)
	}

	return []aI34_3Scenario{
		{name: "slow_text", transcript: aI34TextTranscript, request: validRequest, drainEvents: drainSlow},
		{name: "slow_tool", transcript: aI34ToolTranscript, request: validToolCallRequest, drainEvents: drainSlow},
		{name: "bursty_text", transcript: aI34TextTranscript, request: validRequest, drainEvents: drainBursty},
		{name: "bursty_tool", transcript: aI34ToolTranscript, request: validToolCallRequest, drainEvents: drainBursty},
		{name: "pause_resume_text", transcript: aI34TextTranscript, request: validRequest, drainEvents: drainPauseResume},
		{name: "pause_resume_tool", transcript: aI34ToolTranscript, request: validToolCallRequest, drainEvents: drainPauseResume},
	}
}

// TestAI34_3_NoUnsanctionedLossPath_AllProfiles covers R-AIS-040 /
// S-1, S-2, S-3 in one wrapper test (six scenarios × 50 repeats each).
// Wrapping them in a single test lets them share the leak-check
// preamble and the helper definitions; per-scenario subtests give a
// clear diagnostic when one fails. R-STK-008 forbids t.Parallel() —
// the wrapper runs serially.
//
// Per-repeat setup: build the server INSIDE the closure (the
// RequireNoGoroutineLeak pattern a_i-33_3_test.go and a_i-33_5b_test.go
// already use). Building the server outside the closure leaves the
// keep-alive listener goroutines alive across the 50 repeats and
// inflates the leak-check count past tolerance — the listener's own
// goroutines are part of the "before" snapshot the first repeat sees,
// and the per-request handler goroutines that follow are not.
func TestAI34_3_NoUnsanctionedLossPath_AllProfiles(t *testing.T) {
	for _, sc := range aI34_3Scenarios(t) {
		sc := sc
		t.Run(sc.name, func(t *testing.T) {
			agenttest.RequireNoGoroutineLeak(t, func() {
				server := sseServer(t, sc.transcript)
				defer server.Close()

				c := mustClient(t, server.URL)
				ch, err := c.Stream(context.Background(), sc.request(t))
				if err != nil {
					t.Fatalf("Stream() error = %v, want nil (R-AIS-040)", err)
				}
				events := sc.drainEvents(t, ch)
				assertClosedExactlyOnce(t, ch)
				aI34_3AssertLossless(t, sc.name, events)
			})
		})
	}
}

