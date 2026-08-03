// AI-22.5 — proof for the carrier view (R-STK-011 … R-STK-013), from
// outside package agenttest, the same agenttest_test package this
// milestone's sibling proof files use.
package agenttest_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-22.5 item 1 (R-STK-011, S-STK-031) — a stream and a view over it,
// iterated to completion, ends with the view having performed no close: a
// second receive on the underlying channel reports closed, proving the
// producer's own close (not the view — which cannot even compile a close on
// its receive-only channel) is what ended the stream.
func TestIter_IterateToCompletion_ProducersOwnCloseIsWhatEndsIt(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(mustTextDeltaScript(t))
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	view := agenttest.NewIter(ch)
	var got []ai.Event
	for ev := range view.Events(context.Background()) {
		got = append(got, ev)
	}
	if len(got) != 4 {
		t.Fatalf("view yielded %d event(s), want 4 (start, two deltas, end)", len(got))
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("receiving again after the view's iteration finished reported a value, want the closed zero value")
		}
	case <-time.After(boundedFakeTimeout):
		t.Fatal("timed out waiting for the channel to report closed")
	}
}

// AI-22.5 item 1 (R-STK-011, S-STK-032) — a view abandoned before the
// stream ends, then cancelled by the caller, leaves the stream terminating
// exactly as it does with no view interposed (drainFake on the raw channel
// afterward reproduces AI-21's own R-AFP-012 bare-close proof shape
// unchanged).
func TestIter_AbandonedThenCancelled_StreamTerminatesExactlyAsWithoutAView(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	provider := agenttest.NewProvider(mustTextDeltaScript(t)) // unbuffered: no reader ever again after abandonment
	ch, err := provider.Stream(ctx, mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	view := agenttest.NewIter(ch)
	count := 0
	for range view.Events(context.Background()) {
		count++
		break // abandon the view after the first event
	}
	if count != 1 {
		t.Fatalf("view yielded %d event(s) before abandonment, want 1", count)
	}

	cancel()
	settleAfterFakeCancel()
	got := drainFake(t, ch)
	if len(got) != 0 {
		t.Errorf("received %d further event(s) after abandoning the view and cancelling, want 0 (identical to R-AFP-012's own bare-close behavior)", len(got))
	}
}

// AI-22.5 item 2 (R-STK-012, S-STK-033) — a stream ending in a terminal
// error: iterating to the end, then inspecting Err(), reports the terminal
// failure with its category intact.
func TestIter_StreamEndsInTerminalError_ErrReportsItAfterTheLoopWithCategoryIntact(t *testing.T) {
	t.Parallel()

	failure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, true)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}
	terminal, err := ai.ErrorEvent(failure)
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(mustPriorOutputEvent(t)),
		agenttest.Emit(terminal),
	}}
	provider := agenttest.NewProvider(script)
	ch, streamErr := provider.Stream(context.Background(), mustSimpleRequest(t))
	if streamErr != nil {
		t.Fatalf("Stream returned %v, want no failure", streamErr)
	}

	view := agenttest.NewIter(ch)
	var got []ai.Event
	for ev := range view.Events(context.Background()) {
		got = append(got, ev)
	}
	if len(got) != 2 {
		t.Fatalf("view yielded %d event(s), want 2 (one prior event + the terminal error event)", len(got))
	}

	viewErr := view.Err()
	if viewErr == nil {
		t.Fatal("view.Err() = nil after a stream ending in a terminal error, want the terminal failure")
	}
	var reportedFailure *ai.Failure
	if !errors.As(viewErr, &reportedFailure) {
		t.Fatalf("view.Err() = %v, want an *ai.Failure", viewErr)
	}
	if got := reportedFailure.Category(); got != ai.FailureCategoryUnavailable {
		t.Errorf("view.Err()'s category = %v, want %v (category intact)", got, ai.FailureCategoryUnavailable)
	}
}

// AI-22.5 item 2 (R-STK-012, S-STK-034) — a stream that completes normally:
// iterating to the end, then inspecting Err(), reports none.
func TestIter_StreamCompletesNormally_ErrReportsNoneAfterTheLoop(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(mustTextDeltaScript(t))
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	view := agenttest.NewIter(ch)
	count := 0
	for range view.Events(context.Background()) {
		count++
	}
	if count != 4 {
		t.Fatalf("view yielded %d event(s), want 4 (start, two deltas, end)", count)
	}

	if viewErr := view.Err(); viewErr != nil {
		t.Errorf("view.Err() = %v after a normally completing stream, want nil", viewErr)
	}
}

// AI-22.5 item 2 (R-STK-012, S-STK-035) — a mid-iteration cancellation ends
// iteration before a bounded wait deadline, rather than blocking: the view
// respects context cancellation.
func TestIter_MidIterationCancellation_EndsBeforeBoundedDeadlineRatherThanBlocking(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewProvider(mustTextDeltaScript(t)) // unbuffered: no reader ever again after the first event
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	viewCtx, cancel := context.WithCancel(context.Background())
	view := agenttest.NewIter(ch)

	done := make(chan struct{})
	var count int
	go func() {
		defer close(done)
		for range view.Events(viewCtx) {
			count++
			cancel() // cancel the VIEW's own ctx after the first event, not the stream's
		}
	}()

	select {
	case <-done:
	case <-time.After(boundedFakeTimeout):
		t.Fatal("view.Events(viewCtx) did not end within the bounded deadline after cancellation, want it to respect ctx and stop promptly")
	}
	if count == 0 {
		t.Fatal("view yielded 0 events before cancellation, want at least 1 (fixture did not exercise mid-iteration cancellation)")
	}

	if viewErr := view.Err(); viewErr == nil {
		t.Error("view.Err() = nil after a mid-iteration cancellation, want ctx.Canceled")
	} else if !errors.Is(viewErr, context.Canceled) {
		t.Errorf("view.Err() = %v, want errors.Is(err, context.Canceled)", viewErr)
	}
}

// AI-22.5 item 4 (NFR-STK-E, S-STK-044 partial) — NewIter against a nil
// channel, and Err() called before any iteration, never panic and report
// an attributable result (an empty sequence, and nil respectively).
func TestIter_ExtremeInputs_NeverPanic(t *testing.T) {
	t.Parallel()

	t.Run("nil channel: Events yields nothing", func(t *testing.T) {
		t.Parallel()

		view := agenttest.NewIter(nil)
		count := 0
		for range view.Events(context.Background()) {
			count++
		}
		if count != 0 {
			t.Errorf("view over a nil channel yielded %d event(s), want 0", count)
		}
	})

	t.Run("Err called before any iteration reports nil", func(t *testing.T) {
		t.Parallel()

		view := agenttest.NewIter(make(chan ai.Event))
		if err := view.Err(); err != nil {
			t.Errorf("view.Err() before any iteration = %v, want nil", err)
		}
	})
}
