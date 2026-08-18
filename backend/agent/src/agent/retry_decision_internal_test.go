// AG-15 — S-RTY-001, the retry predicate's own directly-callable
// table-driven unit test (R-RTY-001).
//
// This is the ONE test in this change that is package-internal rather
// than package agent_test. NFR-RTY-001 states the default ("every
// behavioral test MUST live in package agent_test") and its own
// carve-out in the same sentence: "The predicate's own table-driven
// test MAY exercise it through whatever surface sdd-design fixed,
// provided every behavioral claim above is also observable
// externally." Design AD-2 fixes retryDecision and retryVerdict as
// unexported — deliberately, mirroring every other pure decision
// helper in this package (wrapHarnessFailure, cancellationTurnFailure,
// typedFailureFromError) — so a directly-callable test of the
// predicate itself cannot be written from agent_test at all; Go's
// export rule leaves no other route. Every requirement this predicate
// serves (R-RTY-002..005, R-RTY-010) is independently and externally
// proven by retry_policy_test.go's Harness.Run-driven scenarios
// (S-RTY-002..004, S-RTY-013), satisfying NFR-RTY-001's proviso.
//
// Both substrate filters carry this file's exact suffix
// (retry_decision_internal_test.go), landed in the same commit that
// introduces it, per R-LSK-004's naming discipline.
package agent

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// errPlainRetryDecision is a plain Go error (not an *ai.Failure) —
// the G1 fixture: no typed evidence exists, so no evidence-driven
// decision is possible.
var errPlainRetryDecision = errors.New("retry decision test: plain error")

// mustRetryDecisionFailure builds an *ai.Failure for the table below.
// preStream selects PreStreamFailure (never claims partial output,
// the R-AIP-010 unconstructible cell) vs MidStreamFailure (accepts an
// explicit outputPreceded flag).
func mustRetryDecisionFailure(t *testing.T, retryable, preStream, outputPreceded bool) *ai.Failure {
	t.Helper()
	report := ai.FailureReport{Category: ai.FailureCategoryUnavailable, Retryable: retryable}
	if preStream {
		f, err := ai.PreStreamFailure(report)
		if err != nil {
			t.Fatalf("ai.PreStreamFailure returned %v, want no failure", err)
		}
		return f
	}
	f, err := ai.MidStreamFailure(report, outputPreceded)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}
	return f
}

// TestRetryDecision — S-RTY-001. Given the predicate as a directly
// callable pure function and a table of cases covering every gate it
// implements (G1-G5; G0 is the pre-existing inline cancellation check
// ahead of this call, unchanged and untested here), when each case is
// evaluated, then each yields the verdict its gate names.
func TestRetryDecision(t *testing.T) {
	t.Parallel()

	const bound = 3

	tests := []struct {
		name    string
		terr    error
		attempt int
		want    retryVerdict
	}{
		{
			name:    "G1 plain Go error surfaces, no type-assertion panic",
			terr:    errPlainRetryDecision,
			attempt: 1,
			want:    verdictSurface,
		},
		{
			name:    "G2 non-retryable, no partial output, surfaces",
			terr:    mustRetryDecisionFailure(t, false, true, false),
			attempt: 1,
			want:    verdictSurface,
		},
		{
			name:    "G2 evaluated before G3: non-retryable AND partial output still surfaces at G2",
			terr:    mustRetryDecisionFailure(t, false, false, true),
			attempt: 1,
			want:    verdictSurface,
		},
		{
			name:    "G3 reached, not G4: retryable AND partial output surfaces, is not retried",
			terr:    mustRetryDecisionFailure(t, true, false, true),
			attempt: 1,
			want:    verdictSurface,
		},
		{
			name:    "G4 retryable, no partial output, attempt < bound retries",
			terr:    mustRetryDecisionFailure(t, true, true, false),
			attempt: 1,
			want:    verdictRetry,
		},
		{
			name:    "G4 retryable, no partial output, attempt still < bound at the last eligible attempt",
			terr:    mustRetryDecisionFailure(t, true, false, false),
			attempt: bound - 1,
			want:    verdictRetry,
		},
		{
			name:    "G5 retryable, no partial output, attempt == bound is exhausted (failover consult)",
			terr:    mustRetryDecisionFailure(t, true, true, false),
			attempt: bound,
			want:    verdictExhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := retryDecision(tt.terr, tt.attempt, bound); got != tt.want {
				t.Errorf("retryDecision(%v, attempt=%d, bound=%d) = %v, want %v", tt.terr, tt.attempt, bound, got, tt.want)
			}
		})
	}
}

// TestRetryDecision_SameEvidenceRetriesThenExhausts — S-RTY-001's
// final clause: "the same evidence evaluated at attempts < H and at
// attempts == H yields the retry verdict and the failover-consult
// verdict respectively". Builds one retryable, no-partial-output
// failure and evaluates it at both attempt counts, proving the bound
// alone (not the evidence) separates G4 from G5.
func TestRetryDecision_SameEvidenceRetriesThenExhausts(t *testing.T) {
	t.Parallel()

	const bound = 3
	failure := mustRetryDecisionFailure(t, true, true, false)

	if got := retryDecision(failure, bound-1, bound); got != verdictRetry {
		t.Errorf("retryDecision(evidence, attempt=%d, bound=%d) = %v, want %v (G4)", bound-1, bound, got, verdictRetry)
	}
	if got := retryDecision(failure, bound, bound); got != verdictExhausted {
		t.Errorf("retryDecision(evidence, attempt=%d, bound=%d) = %v, want %v (G5)", bound, bound, got, verdictExhausted)
	}
}
