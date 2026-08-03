// AI-22.4 — opt-in, serial-only goroutine leak detection (design.md D4).
//
// # Mechanism decision, and the rejected alternative (R-STK-009, `[decision]`)
//
// This file's mechanism is hand-rolled, standard-library-only: repeat the
// caller's scenario a fixed number of times, compare live-goroutine counts
// before and after (runtime.NumGoroutine, no per-goroutine attribution),
// and tolerate roughly half that repeat count of background jitter. A
// third-party leak detector — concretely, go.uber.org/goleak, the option
// this decision actually weighed — was REJECTED for this change, for two
// reasons, both stated so the choice reads as decided rather than
// forgotten:
//
//  1. openspec/AGENTS.md rule 5 ("New top-level dependency ⇒ ADR first.
//     Document rationale, alternatives, rollback.") is not satisfied by this
//     change; adopting a third-party detector needs its own ADR, which this
//     milestone does not write.
//  2. doc.go documents this package as dependency-free — R-STK-009's own
//     citation — and a third-party detector would break that pin before
//     AI-24 (the milestone that first selects a transport dependency).
//
// The rejection is SCOPED to this change and REVERSIBLE: a later milestone
// MAY adopt go.uber.org/goleak by writing that ADR. It is not a permanent
// rule. NFR-STK-A's own dependency-purity guard (go.mod's zero requires,
// both AI-00 import guards) is this decision's mechanical proof.
//
// Also rejected, for the reasons R-STK-007/R-STK-008 already state:
// widening the tolerance band to "support" parallel use (an unbounded band
// asserts nothing), and instrumenting every drain automatically (false
// positives against every existing parallel stream test).
//
// # Coverage is narrowed to two paths, deliberately (R-STK-010)
//
// This file's tests carry leak assertions on the CANCELLATION path and the
// ABANDONED-THEN-CANCELLED path only. The ABANDONED-NEVER-CANCELLED path —
// a consumer that stops reading and never cancels — is deliberately NOT
// asserted here. ai-stream-lifecycle § 5 rules it untestable to
// termination: "no test proves a goroutine never exits, and a bounded
// observation that it has not exited yet is a strictly weaker claim that
// would be mistaken for the stronger one," and states plainly that "the
// abandoned-then-cancelled path is testable and is where the leak
// assertions go; the abandoned-never-cancelled path is the documented
// violation and gets no test pretending otherwise." Doc 0002's charter
// phrase "the abandoned-consumer and cancellation paths" is narrowed
// accordingly, on the record, not silently.

package agenttest

import (
	"runtime"
	"testing"
	"time"
)

// leakRepeats is how many times RequireNoGoroutineLeak runs the caller's
// scenario. A per-iteration leak grows the live-goroutine count by roughly
// this many; background jitter from unrelated goroutines does not — the
// same repeated-call amplification fake_cancellation_test.go's own
// S-AFP-031 check already uses for exactly this reason, restated here as
// the kit's one general-purpose mechanism instead of a third bespoke copy.
const leakRepeats = 50

// leakTolerance is the growth RequireNoGoroutineLeak still accepts as
// jitter (R-STK-008, S-STK-025): half the repeat count. A scenario leaking
// even one goroutine per iteration grows the count by ~leakRepeats, well
// past this band, so the tolerance stays meaningful rather than being
// widened into uselessness.
const leakTolerance = leakRepeats / 2

// leakSettle is a fixed window after the repeated calls, before the "after"
// snapshot, giving every scenario's goroutines a chance to actually start
// (a leaked one) or actually finish (a non-leaked one) — the same
// documented-settling-step posture fake_cancellation_test.go's
// settleAfterFakeCancel already establishes for this exact codebase
// (S-AFP-054-permitted: never used to order events, only to let a
// goroutine's own state become observable before the count is read).
const leakSettle = 50 * time.Millisecond

// leakSerialOnlySentinel is the environment variable name
// RequireNoGoroutineLeak pins through tb.Setenv (R-STK-008's mechanical
// enforcement). Its value is never read back; only the call itself matters.
const leakSerialOnlySentinel = "AGENTTEST_STREAM_KIT_LEAK_CHECK"

// RequireNoGoroutineLeak fails tb when repeating scenario leakRepeats times
// grows the live-goroutine count by more than leakTolerance — an
// amplitude-based check, never a single before/after snapshot, so a
// per-iteration leak (which grows roughly linearly with the repeat count)
// is distinguishable from ordinary background jitter (R-STK-007).
//
// It is opt-in: nothing calls this automatically, no package-level or
// suite-level hook exists, and a test that never calls it is completely
// unaffected by this file's existence (S-STK-022).
//
// # Serial-only, non-negotiably (R-STK-008)
//
// A test using this helper MUST NOT call t.Parallel(), on itself or an
// ancestor. runtime.NumGoroutine's count is PROCESS-WIDE: this package
// cannot distinguish this scenario's goroutines from a sibling parallel
// test's, and Go offers no API to query a caller's own parallelism, so this
// file does not attempt to attribute goroutines to one test among many —
// it scopes AROUND that problem instead of claiming to solve it. The
// contract is enforced mechanically, not only documented: this function
// calls tb.Setenv on a sentinel variable, which the testing package itself
// panics on when the calling test (or an ancestor) has already called
// t.Parallel() — a parallel misuse fails loudly at the call site, not
// silently with an unreliable count.
func RequireNoGoroutineLeak(tb testing.TB, scenario func()) {
	tb.Helper()

	tb.Setenv(leakSerialOnlySentinel, "1")

	before := runtime.NumGoroutine()
	for range leakRepeats {
		scenario()
	}
	time.Sleep(leakSettle)
	after := runtime.NumGoroutine()

	if after > before+leakTolerance {
		tb.Fatalf("agenttest: RequireNoGoroutineLeak: goroutine count grew from %d to %d across %d repeats (tolerance %d), want no per-call leak (R-STK-007)",
			before, after, leakRepeats, leakTolerance)
	}
}
