// AG-17.1 — the context strategy seam (R-CTX-001..005).
//
// Mints the seam in one file mirroring failover_policy.go symbol for
// symbol (design.md DD1, DD3): one-method interface, exported-field
// prompt struct, empty verdict struct, shipped no-op, var _ guard.
// Consulted exactly once per LOGICAL turn, at Harness.Run's turn
// boundary (harness.go), never per retry attempt.

package agent

import (
	"context"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ContextStrategy is the turn-boundary context seam (R-CTX-001).
// Method name mirrors PermissionPolicy.Resolve and FailoverPolicy.Resolve
// (failover_policy.go:38) — the house one-method seam convention.
type ContextStrategy interface {
	// Resolve is consulted exactly once per LOGICAL turn, at AG-13's
	// turn boundary in Harness.Run's outer loop — never per retry
	// attempt (see the R-RTY-002 argument at harness.go:550-561).
	Resolve(ctx context.Context, prompt ContextPrompt) ContextVerdict
}

// ContextPrompt is the typed report Resolve receives — the same
// exported-field posture as FailoverPrompt (failover_policy.go:44).
type ContextPrompt struct {
	// Transcript is a fresh clone of the slice the coming logical
	// turn's attempts will send (request.go:372-375's own argument:
	// a consumer that rewrites what it received must not be able to
	// rewrite the harness's slice).
	Transcript []ai.Message

	// Budget is Layer 3's stated budget, possibly absent (R-CTX-005).
	Budget ContextBudget

	// Accounting is the transcript's measured size with type-level
	// provenance (R-CTX-007, R-CTX-008). Even TokenSourceReported is
	// exact only for the PRE-hook request (R-CTX-011): it is the
	// provider's own figure FOR THE PRE-HOOK REQUEST, never a claim
	// about the bytes actually sent.
	Accounting TokenAccounting
}

// ContextVerdict is Resolve's typed return value. v1 ships NO field:
// the zero value is the only constructible value, so a verdict that
// requests compaction is unconstructible by ANY implementation — the
// FailoverVerdict posture (failover_policy.go:55-63). AG-18 adds
// compaction fields non-breakingly; every implementation returning
// the zero verdict keeps compiling.
type ContextVerdict struct{}

// NoOpContextStrategy is the one shipped, installable never-compact
// default. Installing it changes nothing observable versus leaving
// Harness.ContextStrategy nil (R-CTX-004, the inertness pin).
type NoOpContextStrategy struct{}

// Resolve implements ContextStrategy: always the zero verdict.
func (NoOpContextStrategy) Resolve(context.Context, ContextPrompt) ContextVerdict {
	return ContextVerdict{}
}

// Compile-time guard: NoOpContextStrategy must satisfy ContextStrategy.
var _ ContextStrategy = NoOpContextStrategy{}

// ContextBudget carries Layer 3's stated token budget (R-CTX-005). Its
// zero value is ABSENT — "Layer 3 stated no budget" — never "a budget
// of zero tokens"; ContextBudgetOf(0) is a stated zero and a different
// value (the ai.TokenCount discipline, ai/usage.go:34-47).
type ContextBudget struct {
	limit   int64
	present bool
}

// ContextBudgetOf builds a stated budget. It is total: n < 0 yields
// the absent zero value — a negative limit is not a budget, and
// minting it present would hand AG-18 a bound under which every
// transcript overflows, the exact defect the presence bit prevents.
func ContextBudgetOf(n int64) ContextBudget {
	if n < 0 {
		return ContextBudget{}
	}
	return ContextBudget{limit: n, present: true}
}

// Limit returns the limit and whether Layer 3 stated one — the
// two-result idiom, so the limit is unreadable without its presence
// (ai/usage.go:57-62).
func (b ContextBudget) Limit() (int64, bool) { return b.limit, b.present }
