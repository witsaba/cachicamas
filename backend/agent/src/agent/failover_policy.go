// AG-15.3 — the failover seam (R-RTY-010, R-RTY-011, design AD-7).
//
// The seam is a named injection point, consulted exactly once per
// logical turn at G5 (the attempt bound reached), before the
// exhausted-retry report is surfaced. v1 ships the seam and one
// concrete declining implementation; it does not ship a fallback
// route, a re-budgeting strategy, or a cache-prefix restart — those
// are AGS-D's own deferred placeholder, post-v1 (agent-v1-scope
// spec.md:128,133).

package agent

import (
	"context"

	"github.com/cachicamas/backend/agent/src/ai"
)

// FailoverPolicy is the failover seam's own injection point
// (R-RTY-010). It is carried as a nil-default field on the
// caller-owned Harness (Harness.Failover), never on TurnOptions:
// exhaustion is a run-driver concept — Turn knows nothing of attempts
// — and a seam belongs where its consumer is. A nil FailoverPolicy is
// never called and behaves as a decline (R-RTY-011, the inertness
// pin). Method name mirrors PermissionPolicy.Resolve
// (permission_protocol.go:80-94).
type FailoverPolicy interface {
	// Resolve is consulted exactly once when the retry budget for a
	// logical turn is exhausted (G5), after the attempt bound is
	// reached and before the terminal report is emitted.
	//
	// Contract for a real, post-v1 implementation (documentation
	// only in v1 — the implementation behind the seam stays deferred,
	// AGS-D): it MUST re-count the context budget and MUST restart
	// the cache prefix for the substitute route before accepting.
	// Neither obligation is enforceable by this interface alone; a
	// real implementation is responsible for both.
	Resolve(ctx context.Context, prompt FailoverPrompt) FailoverVerdict
}

// FailoverPrompt is the typed report FailoverPolicy.Resolve receives
// (R-RTY-010): an exported-field report, the same FailureReport
// posture Layer 1 uses (provider_failure.go:270-310).
type FailoverPrompt struct {
	// Attempts is H, the number of Turn invocations made for the
	// logical turn that exhausted its retry budget (R-RTY-005).
	Attempts int

	// Failure is the final attempt's own typed failure — the same
	// *ai.Failure value the exhausted-retry run_end wraps
	// (R-RTY-012), never a reconstructed report.
	Failure *ai.Failure
}

// FailoverVerdict is FailoverPolicy.Resolve's typed return value
// (R-RTY-010). Declining is the zero value: v1 ships no acceptance
// field at all, so an accepting verdict is unconstructible by any
// implementation — the same unconstructible-cell posture Layer 1
// uses for PreStreamFailure (provider_failure.go:604-609), stronger
// than an ignored accept flag. A later version adds route and
// re-budget fields non-breakingly; every existing implementation
// returning the zero verdict keeps compiling as a decliner.
type FailoverVerdict struct{}

// NoOpFailoverPolicy is the package's one shipped, installable
// declining implementation (R-RTY-010, the NoOpPermissionPolicy
// shape) — an installable value, which a nil field is not. Resolve
// always returns the zero FailoverVerdict{}, so installing it changes
// nothing observable versus leaving Harness.Failover nil
// (R-RTY-011, the inertness pin).
type NoOpFailoverPolicy struct{}

// Resolve implements FailoverPolicy: always declines.
func (NoOpFailoverPolicy) Resolve(context.Context, FailoverPrompt) FailoverVerdict {
	return FailoverVerdict{}
}

// Compile-time guard: NoOpFailoverPolicy must satisfy FailoverPolicy.
var _ FailoverPolicy = NoOpFailoverPolicy{}
