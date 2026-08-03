// AI-21.5 — cancellation fidelity: mid-stream cancellation behaves exactly
// as AI-20.3 requires, including the sanctioned drop; pre-stream
// cancellation takes the pre-stream path, after validation.
//
// fake_cancellation_test.go proves R-AFP-012/013 from outside package
// agenttest, mirroring ai/provider_test.go's own cancellation tests —
// same agenttest_test package as this file's siblings.
package agenttest_test

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// requireAgenttestViolation asserts err is an *ai.Violation naming
// wantRule at a position whose final step is wantField — this file's own
// requireViolation, since ai_test's is an internal test helper of another
// package and unreachable from here.
func requireAgenttestViolation(t *testing.T, err error, wantRule error, wantField string) {
	t.Helper()

	if err == nil {
		t.Fatal("err is nil, want a violation")
	}
	if !errors.Is(err, wantRule) {
		t.Errorf("errors.Is(err, %v) = false on %v", wantRule, err)
	}
	var violation *ai.Violation
	if !errors.As(err, &violation) {
		t.Fatalf("errors.As(err, *ai.Violation) = false on %v", err)
	}
	path := violation.Path()
	if len(path) == 0 {
		t.Fatalf("violation.Path() is empty, want a path ending in %q", wantField)
	}
	if got := path[len(path)-1].Name(); got != wantField {
		t.Errorf("violation.Path()'s final step = %q, want %q", got, wantField)
	}
}

// AI-21.5 item 1 (R-AFP-012, S-AFP-027…029) — mid-script cancellation
// closes the stream within bounded time under -race; the saturated
// (unbuffered, unread) path drops the remaining scripted events and closes
// bare, with no terminal event and no send after the close.
func TestProvider_MidScriptCancellation_ClosesWithinBoundedTime_DropsRemainingAndClosesBare(t *testing.T) {
	t.Parallel()

	for i := range 20 { // repetition under -race, mirroring AI-20.3's own pattern
		t.Run(fmt.Sprintf("iteration_%d", i), func(t *testing.T) {
			t.Parallel()

			start := mustTextStartEvent(t)
			end, err := ai.NewTextBlockEnd(1)
			if err != nil {
				t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
			}
			script := agenttest.Script{Steps: []agenttest.Step{agenttest.Emit(start), agenttest.Emit(end)}} // buffer 0
			provider := agenttest.NewProvider(script)
			ctx, cancel := context.WithCancel(context.Background())

			ch, err := provider.Stream(ctx, mustSimpleRequest(t))
			if err != nil {
				t.Fatalf("Stream returned %v, want no failure", err)
			}

			// Synchronous handoff: receiving the first event proves the
			// producer reached (and passed) its first send.
			first := mustReceive(t, ch)
			if first.Kind() != ai.EventKindTextBlockStart {
				t.Fatalf("first event kind = %v, want %v", first.Kind(), ai.EventKindTextBlockStart)
			}

			cancel()
			settleAfterFakeCancel()
			got := drainFake(t, ch)
			if len(got) != 0 {
				t.Errorf("received %d further event(s) after cancelling a saturated stream, want 0 (dropped, closed bare, no terminal forced through)", len(got))
			}
		})
	}
}

// AI-21.5 item 2 (R-AFP-013, S-AFP-030…032) — a valid scripted request
// against an already-cancelled context fails on the pre-stream path with a
// typed failure carrying the cancellation category, no usable carrier and
// no additional goroutine; a zero-value request against an
// already-cancelled context reports the validation failure instead,
// preserving AI-20.2's ordering rather than short-circuiting on
// cancellation.
func TestProvider_PreStreamCancellation_TypedFailureNoCarrier_ValidationOrderedFirst(t *testing.T) {
	t.Parallel()

	t.Run("valid request, already-cancelled context: typed cancellation failure, no carrier, no extra goroutine", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		provider := agenttest.NewProvider(mustTextDeltaScript(t))

		ch, err := provider.Stream(ctx, mustSimpleRequest(t))
		if ch != nil {
			t.Error("Stream returned a non-nil channel for an already-cancelled context, want no carrier (R-AFP-013)")
		}
		if err == nil {
			t.Fatal("Stream returned a nil error for an already-cancelled context, want a typed cancellation failure")
		}
		if !errors.Is(err, ai.ErrCancelled) {
			t.Errorf("errors.Is(err, ai.ErrCancelled) = false on %v", err)
		}
		var failure *ai.Failure
		if !errors.As(err, &failure) {
			t.Fatalf("errors.As(err, *ai.Failure) = false on %v", err)
		}
		if got := failure.Category(); got != ai.FailureCategoryCancellation {
			t.Errorf("failure.Category() = %v, want %v", got, ai.FailureCategoryCancellation)
		}
		if got := failure.Delivery(); got != ai.DeliveryPreStream {
			t.Errorf("failure.Delivery() = %v, want %v (pre-stream, never carried by an event)", got, ai.DeliveryPreStream)
		}

		// S-AFP-031 — a single before/after runtime.NumGoroutine() snapshot
		// is unreliable in this package's own busy, heavily t.Parallel()
		// binary (unrelated tests' goroutines wind down concurrently), so
		// this repeats the failing call many times instead: a per-call
		// leak would grow the count by roughly the repeat count, while
		// background jitter from sibling tests does not — a bound well
		// below the repeat count still catches a real leak.
		const repeats = 50
		before := runtime.NumGoroutine()
		for range repeats {
			if _, err := agenttest.NewProvider(mustTextDeltaScript(t)).Stream(ctx, mustSimpleRequest(t)); err == nil {
				t.Fatal("repeated pre-stream call unexpectedly succeeded")
			}
		}
		settleAfterFakeCancel()
		if after := runtime.NumGoroutine(); after > before+repeats/2 {
			t.Errorf("goroutine count grew from %d to %d across %d pre-stream failures, want no per-call goroutine leak", before, after, repeats)
		}
	})

	t.Run("zero-value request, already-cancelled context: the validation failure wins, not cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		provider := agenttest.NewProvider(mustTextDeltaScript(t))
		var zero ai.Request

		ch, err := provider.Stream(ctx, zero)
		if ch != nil {
			t.Error("Stream returned a non-nil channel for a zero request, want no carrier")
		}
		requireAgenttestViolation(t, err, ai.ErrEmpty, "request")
		if errors.Is(err, ai.ErrCancelled) {
			t.Error("a zero request against an already-cancelled context reported cancellation; validation must win (R-AFP-013)")
		}
	})
}
