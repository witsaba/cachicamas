// AI-23.5 — cancellation and closure conformance cases: bounded, leak-free
// closing on cancellation, and the abandoned-then-cancelled saturated path
// dropping cleanly with no invented terminal (R-CNF-011, R-CNF-012).
//
// # Serial-only, non-negotiably (R-STK-008)
//
// Neither case in this file, nor any helper it calls, calls t.Parallel() on
// itself or an ancestor: both are wrapped in AI-22's RequireNoGoroutineLeak,
// whose own leak count is process-wide and therefore meaningless under
// parallel execution. RequireNoGoroutineLeak enforces this mechanically
// (tb.Setenv panics if a parallel ancestor already called t.Parallel()),
// and this file's own absence of any t.Parallel() call is confirmed by
// inspection (this milestone's tasks.md AI-23.5 item 1.3).
//
// # Scope, restated from ai-stream-lifecycle § 5 (S-CNF-031)
//
// This file asserts the cancellation path (R-CNF-011) and the
// abandoned-THEN-cancelled saturated path (R-CNF-012) only. The
// abandoned-NEVER-cancelled path — a consumer that stops reading and never
// cancels — is deliberately NOT asserted here: ai-stream-lifecycle § 5
// rules it untestable to termination ("no test proves a goroutine never
// exits, and a bounded observation that it has not exited yet is a
// strictly weaker claim that would be mistaken for the stronger one").

package agenttest

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

func init() {
	registerConformanceCase("cancellation/bounded_close_leak_free", CapCancellation, cancellationBoundedCloseCase)
	registerConformanceCase("cancellation/abandoned_then_cancelled_drops_bare", CapCancellation, cancellationAbandonedThenCancelledCase)
}

// cancellationBoundedCloseCase proves R-CNF-011: cancelling mid-consumption
// closes the stream before AI-22's bounded drain deadline (S-CNF-026,
// delegated to DrainAndRecord rather than a hand-rolled deadline
// mechanism — the same pattern AI-22.1's own drain helper already uses,
// this leaf's own REFACTOR confirmation), and repeating the scenario
// leaks no goroutine beyond RequireNoGoroutineLeak's tolerance
// (S-CNF-027).
func cancellationBoundedCloseCase(t *testing.T, f Factory) {
	t.Helper()

	scenario := func() {
		start, err := ai.NewTextBlockStart(1)
		requireConstructed(t, err, "ai.NewTextBlockStart")
		delta, err := ai.NewTextDelta(1, "hello")
		requireConstructed(t, err, "ai.NewTextDelta")
		end, err := ai.NewTextBlockEnd(1)
		requireConstructed(t, err, "ai.NewTextBlockEnd")

		script := Script{Steps: []Step{Emit(start), Emit(delta), Emit(end)}} // Buffer 0: unbuffered
		subject := f.New(t, script)
		ctx, cancel := context.WithCancel(t.Context())
		ch, err := subject.Stream(ctx, minimalRequest(t))
		if err != nil {
			t.Fatalf("agenttest: cancellation/bounded_close_leak_free: Stream returned %v, want no failure", err)
		}

		// Synchronous handoff: receiving the first event proves the
		// producer reached (and passed) its first send before cancellation.
		first, ok := <-ch
		if !ok {
			t.Fatal("first receive reported the stream already closed, want the scripted first event")
		}
		if first.Kind() != ai.EventKindTextBlockStart {
			t.Fatalf("first event kind = %v, want %v", first.Kind(), ai.EventKindTextBlockStart)
		}

		cancel()
		rec := DrainAndRecord(t, ch, DefaultDrainTimeout) // bounded: fails this scenario if the stream never closes in time (S-CNF-026)
		if got := rec.Len(); got != 0 {
			t.Errorf("received %d further event(s) after cancelling an unbuffered stream, want 0 (dropped, closed bare, R-CNF-011)", got)
		}
	}

	RequireNoGoroutineLeak(t, scenario)
}

// cancellationAbandonedThenCancelledCase proves R-CNF-012: a consumer that
// never reads at all until the caller cancels sees the stream close bare —
// no terminal event invented, whatever WAS delivered left intact
// (S-CNF-029) — and repeating the scenario leaks no goroutine (S-CNF-030).
// An unbuffered channel with zero reads is saturated from the first
// scripted event onward by construction, so no wall-clock wait is needed
// to reach that state deterministically (NFR-CNF-E).
func cancellationAbandonedThenCancelledCase(t *testing.T, f Factory) {
	t.Helper()

	scenario := func() {
		start, err := ai.NewTextBlockStart(1)
		requireConstructed(t, err, "ai.NewTextBlockStart")
		steps := []Step{Emit(start)}
		for range 8 {
			d, err := ai.NewTextDelta(1, "x")
			requireConstructed(t, err, "ai.NewTextDelta")
			steps = append(steps, Emit(d))
		}
		end, err := ai.NewTextBlockEnd(1)
		requireConstructed(t, err, "ai.NewTextBlockEnd")
		steps = append(steps, Emit(end))

		script := Script{Steps: steps} // Buffer 0
		subject := f.New(t, script)
		ctx, cancel := context.WithCancel(t.Context())
		ch, err := subject.Stream(ctx, minimalRequest(t))
		if err != nil {
			t.Fatalf("agenttest: cancellation/abandoned_then_cancelled_drops_bare: Stream returned %v, want no failure", err)
		}

		cancel() // abandoned from the start: no read happens before this

		rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
		for _, ev := range rec.Events() {
			if _, ok := ev.ErrorPayload(); ok {
				t.Error("an error terminal was invented for the abandoned-then-cancelled path, want none (R-CNF-012)")
			}
			if _, ok := ev.Completion(); ok {
				t.Error("a completion terminal was invented, want none (R-CNF-012)")
			}
		}
	}

	RequireNoGoroutineLeak(t, scenario)
}
