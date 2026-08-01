// AI-20.2, AI-20.3 — the pre-stream and mid-stream contracts, proven by
// scriptProvider: a milestone-local, deliberately crude ai.ModelProvider
// that exercises the real signature and nothing more.
//
// scriptProvider is unexported and stays in this file (R-AMP-013): AI-21's
// scripted fake and AI-22's stream test kit are blocked BY this change and
// must not be pre-empted by an exported, reusable helper here.
package ai_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// scriptProvider is the one producer AI-20.2/AI-20.3 prove the pre-stream
// and mid-stream contracts with.
//
// terminal, when non-nil, is expected to hold the *ai.Failure a test built
// through ai.MidStreamFailure — the only value this file turns into a
// terminal error event, via ai.ErrorEvent. buffer sizes the returned
// channel exactly as [ai.ModelProvider]'s documented buffering rule
// describes: a capacity, not a promise.
type scriptProvider struct {
	events   []ai.Event
	terminal error
	buffer   int
}

// Stream implements [ai.ModelProvider]. See provider_test.go's package
// comment for why this producer is deliberately this crude.
func (p scriptProvider) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error) {
	// R-AMP-006: validation runs exactly once, before any I/O, and before
	// the already-cancelled-context case below — the zero-value Request is
	// rejected on this same path, not treated as empty-but-valid.
	if req.IsZero() {
		return nil, ai.Invalid(ai.ErrEmpty, ai.At("request"))
	}

	// NFR-AMP-D: a nil context must not panic. It is treated as an
	// uncancelled background context, matching every other Go API that
	// accepts context.Context.
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		failure, ferr := ai.PreStreamFailure(ai.FailureReport{
			Category: ai.FailureCategoryCancellation,
			Cause:    err,
		})
		if ferr != nil {
			// FailureCategoryCancellation is always a member of the
			// vocabulary, so this can never actually fail — handled
			// rather than ignored because errcheck requires it.
			return nil, ferr
		}
		return nil, failure
	}

	out := make(chan ai.Event, p.buffer)
	go func() {
		// R-AMP-009: one goroutine sends; this is the one closing site,
		// reached on every exit path below, after the last send attempt.
		defer close(out)

		for _, ev := range p.events {
			// R-AMP-010/011: every send selects on cancellation. With no
			// receiver and a full buffer, this is also R-AMP-012's
			// sanctioned loss path: cancellation wins, the event (and
			// everything scripted after it) is dropped, and the function
			// returns straight into the deferred close — bare, no
			// terminal event forced through.
			select {
			case out <- ev:
			case <-ctx.Done():
				return
			}
		}

		if p.terminal == nil {
			return
		}
		failure, ok := p.terminal.(*ai.Failure)
		if !ok {
			// terminal is documented to hold a *ai.Failure; anything else
			// is a fixture mistake in a test, not a stream to fail on.
			return
		}
		terminalEvent, err := ai.ErrorEvent(failure)
		if err != nil {
			// ai.ErrorEvent only rejects a nil *ai.Failure, and ok above
			// already excludes that.
			return
		}
		select {
		case out <- terminalEvent:
		case <-ctx.Done():
		}
	}()
	return out, nil
}

// AI-20.2 item 1 (R-AMP-005, S-AMP-013…015/017) — a zero (never
// constructed) request fails pre-stream with AI-04's typed failure, no
// carrier and no producer goroutine.
func TestScriptProvider_PreStream_ZeroRequest_FailsWithATypedFailureNoCarrier(t *testing.T) {
	t.Parallel()

	p := scriptProvider{}
	var zero ai.Request

	ch, err := p.Stream(context.Background(), zero)
	if ch != nil {
		t.Error("Stream returned a non-nil channel for a zero request, want no carrier (R-AMP-005)")
	}
	requireViolation(t, err, ai.ErrEmpty, "request")
}

// AI-20.2 item 2 (R-AMP-006, S-AMP-016/018) — validation runs before the
// already-cancelled-context case: a zero request against an
// already-cancelled context still reports the validation failure, not
// cancellation.
func TestScriptProvider_PreStream_ValidationRunsBeforeTheAlreadyCancelledContextCase(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled at call time

	p := scriptProvider{}
	var zero ai.Request

	ch, err := p.Stream(ctx, zero)
	if ch != nil {
		t.Error("Stream returned a non-nil channel for a zero request, want no carrier")
	}
	requireViolation(t, err, ai.ErrEmpty, "request")
	if errors.Is(err, ai.ErrCancelled) {
		t.Error("a zero request against an already-cancelled context reported cancellation; validation must win (R-AMP-006)")
	}
}

// AI-20.2 item 3 (R-AMP-007, S-AMP-019…021) — a valid request against an
// already-cancelled context fails pre-stream with AI-19's cancellation
// category, no carrier and no producer goroutine.
func TestScriptProvider_PreStream_ValidRequestAlreadyCancelledContext_FailsWithCancellationCategory(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req, err := ai.NewRequest("model", []ai.Message{userTextMessage(t, "hi")})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	p := scriptProvider{}
	ch, streamErr := p.Stream(ctx, req)
	if ch != nil {
		t.Error("Stream returned a non-nil channel for an already-cancelled context, want no carrier (R-AMP-007)")
	}
	if streamErr == nil {
		t.Fatal("Stream returned a nil error for an already-cancelled context, want AI-19's cancellation category")
	}
	if !errors.Is(streamErr, ai.ErrCancelled) {
		t.Errorf("errors.Is(err, ai.ErrCancelled) = false on %v", streamErr)
	}
	var failure *ai.Failure
	if !errors.As(streamErr, &failure) {
		t.Fatalf("errors.As(err, *ai.Failure) = false on %v", streamErr)
	}
	if got := failure.Category(); got != ai.FailureCategoryCancellation {
		t.Errorf("failure.Category() = %v, want %v", got, ai.FailureCategoryCancellation)
	}
	if got := failure.Delivery(); got != ai.DeliveryPreStream {
		t.Errorf("failure.Delivery() = %v, want %v (pre-stream, never carried by an event)", got, ai.DeliveryPreStream)
	}
}

// NFR-AMP-D — a nil context does not panic; Stream treats it as an
// uncancelled background context and proceeds normally.
func TestScriptProvider_PreStream_NilContext_TreatedAsBackgroundNotAPanic(t *testing.T) {
	t.Parallel()

	req, err := ai.NewRequest("model", []ai.Message{userTextMessage(t, "hi")})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}

	p := scriptProvider{}
	ch, err := p.Stream(nil, req) //nolint:staticcheck // deliberately proving NFR-AMP-D's nil-context totality
	if err != nil {
		t.Fatalf("Stream(nil, ...) returned %v, want no failure (NFR-AMP-D)", err)
	}
	if ch == nil {
		t.Fatal("Stream(nil, ...) returned a nil channel with a nil error")
	}
	requireClosedWithin(t, ch, boundedTimeout)
}

// AI-20.2 item 4 (R-AMP-008, S-AMP-022/023) — nothing observable survives a
// failed pre-stream call: the provider value is still usable for a later,
// valid call.
func TestScriptProvider_PreStream_AFailedCallLeavesTheProviderValueUsableForALaterValidCall(t *testing.T) {
	t.Parallel()

	p := scriptProvider{}
	var zero ai.Request
	if _, err := p.Stream(context.Background(), zero); err == nil {
		t.Fatal("first call with a zero request unexpectedly succeeded")
	}

	req, err := ai.NewRequest("model", []ai.Message{userTextMessage(t, "hi")})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	ch, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("second call returned %v, want no failure (R-AMP-008: no partial state survives a failed call)", err)
	}
	if ch == nil {
		t.Fatal("second call returned a nil channel with a nil error")
	}
	requireClosedWithin(t, ch, boundedTimeout)
}

// --- AI-20.3 — the mid-stream contract (R-AMP-009…013) -------------------

// mustCompletionEvent builds a normal terminal completion event (V-STR-20),
// used where a test needs *an* event and nothing about its payload matters.
func mustCompletionEvent(t *testing.T) ai.Event {
	t.Helper()

	ev, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}
	return ev
}

// requireClosedWithin drains ch until it closes or timeout passes,
// returning every event received in order.
//
// It fails the test — rather than hanging it — when ch has not closed by
// the deadline, which is what makes a producer that violates the bounded-
// close rule (R-AMP-011) a fast, legible test failure instead of a stuck
// test binary.
func requireClosedWithin(t *testing.T, ch <-chan ai.Event, timeout time.Duration) []ai.Event {
	t.Helper()

	var got []ai.Event
	deadline := time.After(timeout)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return got
			}
			got = append(got, ev)
		case <-deadline:
			t.Fatalf("channel did not close within %s (%d event(s) received so far) — want a bounded close (R-AMP-011)", timeout, len(got))
			return nil
		}
	}
}

// boundedTimeout is generous enough to never flake on a loaded CI runner
// under -race, and short enough that a genuinely stuck producer fails this
// test suite in seconds, not minutes.
const boundedTimeout = 2 * time.Second

// settleAfterCancel gives a producer's goroutine a window to observe an
// already-cancelled context with NO receiver of this file's own in flight.
//
// Without it, a confirmation read placed immediately after cancel() would
// itself become a second ready case in the producer's own select at
// exactly the moment it evaluates one — and Go resolves two simultaneously
// ready select cases pseudo-randomly, so even a correct implementation
// could occasionally "win" a send a moment before honoring cancellation.
// Sleeping first, with no receiver anywhere on the channel during the
// sleep, removes that race entirely: out<-ev has no candidate receiver for
// the whole window, so <-ctx.Done() is the only case ever ready, and the
// producer's select resolves deterministically before this file reads
// anything.
func settleAfterCancel() { time.Sleep(50 * time.Millisecond) }

// AI-20.3 item 1 (R-AMP-009, S-AMP-024…026) — exactly one closing site,
// reached on every exit path: normal completion, the terminal error event,
// and cancellation.
func TestScriptProvider_MidStream_OneClosingSite_AcrossCompletionErrorAndCancellation(t *testing.T) {
	t.Parallel()

	req := mustSimpleRequestForMidStream(t)

	t.Run("completes normally: every scripted event arrives in order, then the channel closes", func(t *testing.T) {
		t.Parallel()

		ev1, ev2 := mustCompletionEvent(t), mustCompletionEvent(t)
		p := scriptProvider{events: []ai.Event{ev1, ev2}}

		ch, err := p.Stream(context.Background(), req)
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		got := requireClosedWithin(t, ch, boundedTimeout)
		if len(got) != 2 {
			t.Fatalf("received %d event(s), want 2", len(got))
		}
	})

	t.Run("ends with the terminal error event, then the channel closes", func(t *testing.T) {
		t.Parallel()

		ev1 := mustCompletionEvent(t)
		failure := mustMidStreamFailure(t, ai.FailureCategoryUnavailable, false)
		p := scriptProvider{events: []ai.Event{ev1}, terminal: failure}

		ch, err := p.Stream(context.Background(), req)
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		got := requireClosedWithin(t, ch, boundedTimeout)
		if len(got) != 2 {
			t.Fatalf("received %d event(s), want 2 (one scripted event + one terminal error event)", len(got))
		}
		last := got[len(got)-1]
		payload, ok := last.ErrorPayload()
		if !ok {
			t.Fatalf("last event's ErrorPayload() reported false, want the terminal error event carrying %v", failure)
		}
		if payload.Category() != failure.Category() {
			t.Errorf("terminal event's failure category = %v, want %v", payload.Category(), failure.Category())
		}
	})

	t.Run("cancelling after one event closes the channel and delivers nothing further", func(t *testing.T) {
		t.Parallel()

		ev1, ev2 := mustCompletionEvent(t), mustCompletionEvent(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		p := scriptProvider{events: []ai.Event{ev1, ev2}} // buffer 0: unbuffered handoff

		ch, err := p.Stream(ctx, req)
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}

		select {
		case _, ok := <-ch:
			if !ok {
				t.Fatal("channel closed before delivering the first event")
			}
		case <-time.After(boundedTimeout):
			t.Fatal("timed out waiting for the first event")
		}

		cancel()
		settleAfterCancel()
		got := requireClosedWithin(t, ch, boundedTimeout)
		if len(got) != 0 {
			t.Errorf("received %d further event(s) after cancelling, want 0", len(got))
		}
	})
}

// AI-20.3 item 2 (R-AMP-010, S-AMP-027/028) — every send selects on
// cancellation, the terminal send included: with no reader ever present,
// cancellation is the only way the producer's goroutine can make progress,
// so its blocked send must be abandoned rather than completed.
func TestScriptProvider_MidStream_EverySendSelectsOnCancellation_IncludingTheTerminal(t *testing.T) {
	t.Parallel()

	req := mustSimpleRequestForMidStream(t)

	t.Run("an ordinary event's send is abandoned on cancellation when nobody ever reads", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		p := scriptProvider{events: []ai.Event{mustCompletionEvent(t)}} // buffer 0, no reader ever

		ch, err := p.Stream(ctx, req)
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		cancel() // the only way the blocked send can ever resolve
		settleAfterCancel()

		got := requireClosedWithin(t, ch, boundedTimeout)
		if len(got) != 0 {
			t.Errorf("received %d event(s) with no reader ever present, want 0 (the send must select on cancellation)", len(got))
		}
	})

	t.Run("the terminal error event's send is abandoned on cancellation when nobody ever reads", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		p := scriptProvider{terminal: mustMidStreamFailure(t, ai.FailureCategoryUnavailable, false)} // no scripted events, buffer 0, no reader ever

		ch, err := p.Stream(ctx, req)
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		cancel()
		settleAfterCancel()

		got := requireClosedWithin(t, ch, boundedTimeout)
		if len(got) != 0 {
			t.Errorf("received %d event(s) with no reader ever present, want 0 (the terminal send must select on cancellation too)", len(got))
		}
	})
}

// AI-20.3 item 4 (R-AMP-011, S-AMP-029…031) — cancellation closes the
// stream within bounded time, with no send after close: repeated,
// concurrent Stream calls under -race never panic ("close of closed
// channel" or "send on closed channel"), which is exactly what a second
// closer or a send racing the close would produce.
func TestScriptProvider_MidStream_CancellationClosesWithinBoundedTime_NoSendAfterClose(t *testing.T) {
	t.Parallel()

	req := mustSimpleRequestForMidStream(t)

	for i := range 20 { // repetition under -race, not a magic count
		i := i
		t.Run(mapIterationName(i), func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(context.Background())
			p := scriptProvider{events: []ai.Event{mustCompletionEvent(t), mustCompletionEvent(t), mustCompletionEvent(t)}}

			ch, err := p.Stream(ctx, req)
			if err != nil {
				t.Fatalf("Stream returned %v, want no failure", err)
			}
			cancel()
			requireClosedWithin(t, ch, boundedTimeout)
		})
	}
}

// mapIterationName avoids importing strconv purely for a subtest label.
func mapIterationName(i int) string {
	names := [...]string{
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
		"10", "11", "12", "13", "14", "15", "16", "17", "18", "19",
	}
	return "iteration_" + names[i]
}

// AI-20.3 item 3, first half (R-AMP-012, S-AMP-032…035) — the one
// sanctioned loss path: cancellation racing a saturated (here, unbuffered
// and unread) stream drops the late event and closes bare, with no
// terminal event forced through.
func TestScriptProvider_MidStream_SanctionedLossPath_CancellationAfterOneDelivery_DropsTheRestAndClosesBare(t *testing.T) {
	t.Parallel()

	req := mustSimpleRequestForMidStream(t)
	ev1, ev2 := mustCompletionEvent(t), mustCompletionEvent(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p := scriptProvider{events: []ai.Event{ev1, ev2}} // buffer 0

	ch, err := p.Stream(ctx, req)
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	// Synchronous handoff: receiving ev1 proves the producer reached (and
	// passed) its first send.
	select {
	case _, ok := <-ch:
		if !ok {
			t.Fatal("channel closed before delivering the first event")
		}
	case <-time.After(boundedTimeout):
		t.Fatal("timed out waiting for the first event")
	}

	// Cancel before reading further: with no reader for ev2, the producer's
	// only way forward is the cancellation branch.
	cancel()
	settleAfterCancel()
	got := requireClosedWithin(t, ch, boundedTimeout)
	if len(got) != 0 {
		t.Errorf("received %d further event(s) after the sanctioned loss point, want 0 (ev2 must be dropped, not forced through)", len(got))
	}
}

// AI-20.3 item 3, second half (R-AMP-012's converse) — a stream that is
// never cancelled never drops: a saturated (unbuffered) channel makes the
// producer WAIT for a reader, and every scripted event still arrives, in
// order.
func TestScriptProvider_MidStream_NeverCancelled_BackpressureWaitsAndDropsNothing(t *testing.T) {
	t.Parallel()

	req := mustSimpleRequestForMidStream(t)
	events := []ai.Event{mustCompletionEvent(t), mustCompletionEvent(t), mustCompletionEvent(t)}
	p := scriptProvider{events: events} // buffer 0: every send must wait for this test's reader

	ch, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	got := requireClosedWithin(t, ch, boundedTimeout)
	if len(got) != len(events) {
		t.Fatalf("received %d event(s) with no cancellation, want all %d (a full buffer must wait, never drop)", len(got), len(events))
	}
}

// AI-20.3 — buffer sizes the returned channel's capacity exactly, so a
// caller inspecting cap() sees the documented bound rather than a producer
// implementation detail.
func TestScriptProvider_MidStream_BufferSizesTheChannelCapacity(t *testing.T) {
	t.Parallel()

	req := mustSimpleRequestForMidStream(t)
	p := scriptProvider{buffer: 3}

	ch, err := p.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	if got := cap(ch); got != 3 {
		t.Errorf("cap(ch) = %d, want 3", got)
	}
	requireClosedWithin(t, ch, boundedTimeout)
}

// mustSimpleRequestForMidStream avoids re-deriving a valid request in every
// subtest above; it is deliberately separate from any Phase 2 helper so
// this section reads standalone.
func mustSimpleRequestForMidStream(t *testing.T) ai.Request {
	t.Helper()

	req, err := ai.NewRequest("model", []ai.Message{userTextMessage(t, "hi")})
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return req
}
