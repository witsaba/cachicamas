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
// The composed worst-case ceiling this attempt bound multiplies
// against Layer 1's own retry budget (R-RTY-009) is documented once
// AG-15.2 lands the timing seam (retry_backoff_test.go); it is not
// yet stated here.
package agent

import (
	"errors"

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
