// AG-15.1 — the retry decision (R-RTY-001..005, R-RTY-012).
//
// Layer 1 classifies a failure's retryability; until this file,
// nothing in Layer 2 read that classification (harness.go:469 routed
// every non-cancellation Turn error unconditionally to failRun,
// R-RUN-011). retryDecision is the pure, ordered-gate predicate that
// changes that: given the typed evidence a failed Turn invocation
// returned, the attempt count so far and the attempt bound, it
// decides whether the harness surfaces the failure, retries, or has
// exhausted its budget — no I/O, no clock read, no emission, so the
// whole gate table is table-driven-testable without driving a run
// (S-RTY-001, retry_decision_internal_test.go).
//
// G0 — the run context's cause matching an R-CAN-001 cancellation
// sentinel — is deliberately NOT implemented here: it stays the
// existing inline check ahead of this predicate's call site
// (harness.go:460-462, unchanged), because it needs the run's actual
// context, which this pure function never reads (design AD-2). This
// file implements only G1-G5.
//
// # The composed worst-case ceiling (R-RTY-009)
//
// This capability's own attempt bound H composes with Layer 1's own
// retry budget (ai/internal/retry/doc.go, read verbatim, never
// imported — R-RUN-012, R-RTY-006): H total harness attempts × 4 wire
// requests per logical provider call (Layer 1's own documented
// "DefaultMaxAttempts = 3" retry budget means "at most N+1 = 4 wire
// requests" per attempt). At the shipped defaults, H = 3, so the
// composed ceiling is H × 4 = 12 wire requests for one logical turn
// in the worst case where every failure is retryable. H counts TOTAL
// attempts (R-RTY-005) — this is stated here, adjacent to the number,
// because a reader who does not already know the convention cannot
// otherwise tell whether "3" means 3 or 4.

package agent

import (
	"context"
	"errors"
	"math/rand/v2"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// defaultRetryAttempts is H, the default attempt bound: three Turn
// invocations for one logical turn — one initial attempt plus at most
// two retries (R-RTY-005). H counts TOTAL attempts, consciously
// diverging from Layer 1's own retries-after-the-first convention
// (ai/internal/retry/retry.go:15-18): the charter's observable is the
// provider's total call count, and the composed ceiling of R-RTY-009
// multiplies two totals, so a budget convention here would corrupt
// that arithmetic. Carried as Harness.RetryAttempts' zero-default; a
// non-positive field value selects this default (the
// Scheduler.WindDownBound idiom, scheduler.go:611-613).
const defaultRetryAttempts = 3

// retryVerdict is retryDecision's return vocabulary — unexported: it
// governs only this package's own internal control flow between
// Harness.Run's attempt loop and the failure/retry/failover paths,
// never a stream-observable value (no EventKind carries it, no
// accessor exposes it).
type retryVerdict uint8

const (
	// verdictSurface means the failure MUST be surfaced immediately:
	// no retry, no backoff, no failover consult (G1, G2, G3).
	verdictSurface retryVerdict = iota

	// verdictRetry means the harness MUST re-invoke Turn for the same
	// logical turn, after the backoff wait (G4).
	verdictRetry

	// verdictExhausted means the attempt bound has been reached: the
	// harness MUST consult the failover seam exactly once before
	// surfacing the exhausted-retry report (G5).
	verdictExhausted
)

// retryDecision is the pure, ordered-gate predicate of R-RTY-001.
// terr is the error a Turn invocation returned; attempt is the number
// of attempts made so far for this logical turn (including the one
// that just failed); bound is H, the attempt bound (R-RTY-005).
//
// The gates are evaluated in this exact order, first match wins — the
// ordering IS the requirement (R-RTY-001):
//
//	G1  terr is not an *ai.Failure                        -> surface
//	G2  Retryable() == false                               -> surface
//	G3  Retryable() == true, PartialOutput() == true        -> surface
//	G4  Retryable() == true, PartialOutput() == false,
//	    attempt < bound                                     -> retry
//	G5  Retryable() == true, PartialOutput() == false,
//	    attempt >= bound                                     -> exhausted
//
// G3 fires only after G2 has already read Retryable() == true — the
// naive "retry if retryable" predicate is exactly what R-RTY-003
// forbids: a failure reporting itself retryable but carrying already-
// emitted output must never be retried. G5's condition is written
// attempt >= bound rather than the spec table's literal == bound so
// the function stays total for every (attempt, bound) pair; the
// harness's own attempt loop never calls this with attempt > bound,
// so the two conditions coincide on every input this predicate is
// ever actually evaluated against.
//
// Delivery() is deliberately never read here: it distinguishes the
// stream-observable report (R-RTY-012), not the decision — Layer 1
// itself states delivery alone cannot separate the two mid-stream
// shapes (provider_failure.go:522-527), and PartialOutput() already
// answers the question this predicate needs.
func retryDecision(terr error, attempt, bound int) retryVerdict {
	var failure *ai.Failure
	if !errors.As(terr, &failure) {
		return verdictSurface // G1: no typed evidence, fail closed.
	}
	if !failure.Retryable() {
		return verdictSurface // G2
	}
	if failure.PartialOutput() {
		return verdictSurface // G3
	}
	if attempt < bound {
		return verdictRetry // G4
	}
	return verdictExhausted // G5
}

// AG-15.2 — the timing seam (R-RTY-006..009, design AD-5, AD-6).
//
// RetryTiming carries the injected clock and wait-function seam a G4
// retry's backoff wait is built from, freshly declared in this
// package rather than imported: ai/internal/retry is internal to
// backend/agent/src/ai, and backend/agent/src/agent is a sibling, so
// Go's internal-visibility rule forbids the import at compile time
// (R-RTY-006, NFR-RTY-003). The shape mirrors Layer 1's own Config
// (ai/internal/retry/retry.go:25-45) with the harness-irrelevant
// AfterReader dropped — the harness reads (*ai.Failure).RetryAfter()
// directly (provider_failure.go:471-476) rather than through an
// injected reader function.

// RetryTiming is the zero-default value field carried on Harness
// (R-RTY-006): the zero value resolves to production defaults via
// applyRetryTimingDefaults, so an unconfigured harness backs off
// correctly with no configuration at all.
type RetryTiming struct {
	// NowFunc supplies the clock used to seed computed backoff
	// jitter. Nil selects time.Now.
	NowFunc func() time.Time

	// SleepFunc waits for a computed or retry-after delay. It MUST
	// return promptly when ctx is cancelled (R-RTY-008) — nil
	// selects DefaultRetrySleep.
	SleepFunc func(ctx context.Context, d time.Duration) error

	// BaseDelay is the first attempt's un-jittered backoff floor.
	// Non-positive selects 100ms.
	BaseDelay time.Duration

	// MaxDelay is the clamp every computed or retry-after delay is
	// bounded to. Non-positive selects 30s.
	MaxDelay time.Duration

	// JitterSeed seeds the deterministic jitter sequence. Zero
	// selects a seed derived from NowFunc() — production
	// non-determinism, test determinism when NowFunc is pinned.
	JitterSeed int64
}

const (
	defaultRetryBaseDelay = 100 * time.Millisecond
	defaultRetryMaxDelay  = 30 * time.Second
)

// applyRetryTimingDefaults resolves t's zero fields to production
// defaults, mirroring Layer 1's own applyDefaults
// (ai/internal/retry/retry.go:138-158). The caller's own Harness
// value is never mutated — this returns a resolved copy.
func applyRetryTimingDefaults(t RetryTiming) RetryTiming {
	if t.NowFunc == nil {
		t.NowFunc = time.Now
	}
	if t.SleepFunc == nil {
		t.SleepFunc = DefaultRetrySleep
	}
	if t.BaseDelay <= 0 {
		t.BaseDelay = defaultRetryBaseDelay
	}
	if t.MaxDelay <= 0 {
		t.MaxDelay = defaultRetryMaxDelay
	}
	return t
}

// newRetryJitter constructs the deterministic jitter source for one
// logical turn's worth of retry attempts, mirroring Layer 1's own
// newJitter (ai/internal/retry/retry.go:160-165): an explicit non-zero
// seed reproduces the same sequence across runs (S-RTY-006); a zero
// seed reseeds from now, giving production non-determinism.
func newRetryJitter(seed int64, now time.Time) *rand.Rand {
	if seed == 0 {
		seed = now.UnixNano()
	}
	return rand.New(rand.NewPCG(uint64(seed), uint64(now.UnixNano())))
}

// computeRetryBackoff returns bounded exponential backoff with
// deterministic multiplicative jitter in [base, 2*base), capped at
// t.MaxDelay — mirroring Layer 1's own computeBackoff
// (ai/internal/retry/retry.go:169-190). t MUST already be resolved
// through applyRetryTimingDefaults.
func computeRetryBackoff(attempt int, t RetryTiming, jitter *rand.Rand) time.Duration {
	base := t.BaseDelay
	for i := 1; i < attempt && base < t.MaxDelay; i++ {
		if base > t.MaxDelay/2 {
			base = t.MaxDelay
			break
		}
		base *= 2
	}
	if base >= t.MaxDelay {
		return t.MaxDelay
	}
	span := int64(base)
	if span <= 0 {
		return 0
	}
	delay := base + time.Duration(jitter.Int64N(span))
	if delay > t.MaxDelay {
		return t.MaxDelay
	}
	return delay
}

// retryWaitDelay resolves the delay one G4 retry MUST wait: a
// reported retry-after value wins over the computed backoff, clamped
// to [0, t.MaxDelay] (R-RTY-007); an absent value falls back to
// computeRetryBackoff. terr is the same error retryDecision already
// classified as retryable for this attempt, so the errors.As lookup
// here can never fail to find an *ai.Failure — the caller only
// reaches this function after G4 has already matched.
func retryWaitDelay(terr error, attempt int, t RetryTiming, jitter *rand.Rand) time.Duration {
	var failure *ai.Failure
	if errors.As(terr, &failure) {
		if hinted, ok := failure.RetryAfter(); ok {
			if hinted < 0 {
				hinted = 0
			}
			if hinted > t.MaxDelay {
				hinted = t.MaxDelay
			}
			return hinted
		}
	}
	return computeRetryBackoff(attempt, t, jitter)
}

// DefaultRetrySleep is the production default RetryTiming.SleepFunc,
// mirroring Layer 1's own unexported defaultSleep
// (ai/internal/retry/retry.go:192-207): an already-cancelled ctx
// returns immediately with a non-nil error and arms no timer at all
// (S-RTY-008's direct unit test); otherwise it selects on the timer
// and ctx.Done(), so a cancellation during the wait returns promptly.
// Exported so RetryTiming's zero-value default is independently
// testable from package agent_test (NFR-RTY-001) without importing
// ai/internal/retry.
func DefaultRetrySleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
