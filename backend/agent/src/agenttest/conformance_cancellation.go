// AI-23.5 — cancellation and closure conformance cases: bounded, leak-free
// closing on cancellation, and the abandoned-then-cancelled saturated path
// dropping cleanly with at most one admitted cancellation terminal
// (R-CNF-011, R-CNF-012, amended AI-38 design D4).
//
// # Cancellation tail admission (AI-38 design D4, LOCKED decision 1)
//
// R-AEM-014 (S-AEM-051…055, AI-32.3) mandates the shipped adapter surface a
// typed terminal failure on ctx cancel/deadline mid-stream — a real
// conflict with this file's earlier bare-close-only reading of R-CNF-011 /
// R-CNF-012. The maintainer locked the resolution suite-side:
// requireCancellationTail below now admits EITHER a bare close OR exactly
// one ai.ErrorEvent whose Category() == ai.FailureCategoryCancellation, in
// final position, optionally preceded by block-closing end events for
// blocks opened before cancellation (stream.go's own valid-stream-closure
// contract). Still rejected, unchanged: any Completion, more than one
// error terminal, a terminal of any other category, or any event after the
// terminal. AI-21's fake still closes bare (tail []), so this amendment is
// additive, not a regression on the existing green path.
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
		responseStart, err := ai.NewResponseStart("resp-bounded-close", "model-bounded-close")
		requireConstructed(t, err, "ai.NewResponseStart")
		start, err := ai.NewTextBlockStart(1)
		requireConstructed(t, err, "ai.NewTextBlockStart")
		delta, err := ai.NewTextDelta(1, "hello")
		requireConstructed(t, err, "ai.NewTextDelta")
		end, err := ai.NewTextBlockEnd(1)
		requireConstructed(t, err, "ai.NewTextBlockEnd")

		script := Script{Steps: []Step{Emit(responseStart), Emit(start), Emit(delta), Emit(end)}} // Buffer 0: unbuffered; no terminal — a cancelled stream closes bare (AI-20.3)
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
		if first.Kind() != ai.EventKindResponseStart {
			t.Fatalf("first event kind = %v, want %v (S-CLA-010: the synchronous-handoff proof now targets the lifecycle prefix)", first.Kind(), ai.EventKindResponseStart)
		}

		cancel()
		rec := DrainAndRecord(t, ch, DefaultDrainTimeout) // bounded: fails this scenario if the stream never closes in time (S-CNF-026)
		requireCancellationTail(t, rec.Events())
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
		requireCancellationTail(t, rec.Events())
	}

	RequireNoGoroutineLeak(t, scenario)
}

// requireCancellationTail asserts events — the tail drained after
// cancellation — never carries an invented Completion, never carries more
// than one error terminal, and — when exactly one error terminal IS
// present — that terminal is of the cancellation category and in final
// position, with nothing drained after it (design D4, LOCKED decision 1).
//
// A drained tail carrying NO error terminal at all — whether truly empty
// (bare close) or a short prefix of ordinary (non-terminal, non-Completion)
// events — is accepted either way, and so is an ordinary prefix preceding
// an admitted terminal. Both shapes trace to the same race, empirically
// confirmed on both subjects this suite drives: a small transcript can
// arrive, decode and queue several events from a single read before the
// consumer's post-cancel drain starts, and each already-queued send races
// the producer noticing ctx.Done() independently — Go's select has no case
// priority, so a scripted or wire-decoded event already in flight can
// legitimately land on either side of that race without either side being
// a defect (AI-21's fake, over an in-process channel, and the real
// *openaicompat.Client, over real HTTP with the whole small fixture
// arriving in one Read, both exhibit it — see
// TestConformanceCancellation_AbandonedThenCancelledCase_
// PassesAgainstFakeFactory). What this helper refuses, unconditionally, is
// a FABRICATED terminal: an invented Completion, a second error terminal,
// a terminal of the wrong category, or any event following an admitted
// one — R-CNF-011/R-CNF-012's actual content, restated without the
// stronger "only a bare close" claim R-AEM-014 (S-AEM-051…055) supersedes.
func requireCancellationTail(tb testing.TB, events []ai.Event) {
	tb.Helper()

	for _, ev := range events {
		if _, ok := ev.Completion(); ok {
			tb.Errorf("agenttest: cancellation tail: a Completion terminal was invented, want none (R-CNF-011/R-CNF-012)")
		}
	}

	errorIndex, errorCount := -1, 0
	for i, ev := range events {
		if _, ok := ev.ErrorPayload(); ok {
			errorIndex, errorCount = i, errorCount+1
		}
	}
	if errorCount == 0 {
		return // bare, or a raced-through prefix with no terminal invented — conformant either way (R-CNF-011/R-CNF-012)
	}
	if errorCount > 1 {
		tb.Errorf("agenttest: cancellation tail: %d error terminal(s) observed, want at most one admitted cancellation terminal (R-CNF-011/R-CNF-012)", errorCount)
		return
	}
	if errorIndex != len(events)-1 {
		tb.Errorf("agenttest: cancellation tail: the error terminal is followed by %d further event(s), want it in final position (R-CNF-011/R-CNF-012)", len(events)-1-errorIndex)
	}
	payload, _ := events[errorIndex].ErrorPayload()
	if payload.Category() != ai.FailureCategoryCancellation {
		tb.Errorf("agenttest: cancellation tail: terminal category = %v, want %v (R-CNF-011/R-CNF-012)", payload.Category(), ai.FailureCategoryCancellation)
	}
}
