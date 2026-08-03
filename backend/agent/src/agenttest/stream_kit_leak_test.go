// AI-22.4 — proof for opt-in, serial-only goroutine leak detection
// (R-STK-007 … R-STK-010), from outside package agenttest, the same
// agenttest_test package this milestone's sibling proof files use.
//
// NO TEST IN THIS FILE CALLS t.Parallel(), including any subtest — R-STK-008
// is non-negotiable, and RequireNoGoroutineLeak's own tb.Setenv call would
// panic against a *testing.T that has (S-STK-024).
package agenttest_test

import (
	"context"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
)

// AI-22.4 item 1 (R-STK-007, S-STK-020) — a scenario that leaks one
// goroutine per call, wrapped at the helper's repeat count, fails and names
// the observed growth against the repeat count.
func TestRequireNoGoroutineLeak_LeakingScenario_FailsNamingObservedGrowth(t *testing.T) {
	done := make(chan struct{})
	t.Cleanup(func() { close(done) }) // releases every leaked goroutine once this test ends, so it never pollutes a later test's own count

	leaking := func() {
		go func() { <-done }()
	}

	fake := &fakeTB{}
	agenttest.RequireNoGoroutineLeak(fake, leaking)

	if !fake.failed {
		t.Fatal("RequireNoGoroutineLeak did not fail for a scenario leaking one goroutine per call, want a failure")
	}
	msg := fake.lastFatal()
	if !strings.Contains(msg, "50") {
		t.Errorf("failure message %q does not name the repeat count (50)", msg)
	}
}

// AI-22.4 item 1 (R-STK-007, S-STK-021) — a scenario that leaks nothing,
// wrapped at the same repeat count, passes within tolerance.
func TestRequireNoGoroutineLeak_NonLeakingScenario_PassesWithinTolerance(t *testing.T) {
	nonLeaking := func() {
		done := make(chan struct{})
		go func() { close(done) }()
		<-done
	}

	agenttest.RequireNoGoroutineLeak(t, nonLeaking)
}

// AI-22.4 item 1 (S-STK-022, inspection) — an existing stream test that does
// not call the leak helper is unchanged in behaviour: this is a regression
// run only (`make test` at leaf close), not a new test. See tasks.md's own
// recorded confirmation.

// AI-22.4 item 2 (R-STK-008, S-STK-023) — the helper's own doc comment
// states the t.Parallel() incompatibility and the process-wide-count
// reason. This is a doc-content assertion, not a behavioral one: parsing
// go/doc from this external test package would need its own file-resolution
// machinery for little benefit, so the confirmation is a manual read at
// close, recorded in tasks.md, plus this comment documenting the check.

// abandonedCancelScenario builds a scenario for the cancellation-family
// leak proofs (R-STK-010): stream, receive exactly one event, then cancel.
// mustReceive/drainFake/mustTextDeltaScript/mustSimpleRequest are
// fake_text_test.go's shared helpers, reused here rather than
// reimplemented.
func abandonedCancelScenario(t *testing.T, drainAfterCancel bool) func() {
	t.Helper()

	return func() {
		ctx, cancel := context.WithCancel(context.Background())
		provider := agenttest.NewProvider(mustTextDeltaScript(t))
		ch, err := provider.Stream(ctx, mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		mustReceive(t, ch) // proves the producer started before this consumer stops reading
		cancel()
		if drainAfterCancel {
			drainFake(t, ch)
		}
	}
}

// AI-22.4 item 4 (R-STK-010, S-STK-028) — a stream cancelled mid-consumption
// (the consumer keeps draining to close after cancelling), repeated under
// the leak helper, shows no goroutine growth beyond tolerance.
func TestRequireNoGoroutineLeak_CancelledMidConsumption_NoGrowthBeyondTolerance(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, abandonedCancelScenario(t, true))
}

// AI-22.4 item 4 (R-STK-010, S-STK-029) — a consumer that stops reading and
// whose caller then cancels (abandoned-then-cancelled), repeated under the
// leak helper, shows no goroutine growth beyond tolerance.
func TestRequireNoGoroutineLeak_AbandonedThenCancelled_NoGrowthBeyondTolerance(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, abandonedCancelScenario(t, false))
}

// AI-22.4 item 5 (NFR-STK-E, S-STK-044 partial) — a true no-op scenario
// (zero work) passes without panic.
func TestRequireNoGoroutineLeak_NoOpScenario_PassesWithoutPanic(t *testing.T) {
	agenttest.RequireNoGoroutineLeak(t, func() {})
}
