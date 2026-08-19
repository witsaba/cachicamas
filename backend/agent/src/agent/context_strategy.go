// AG-17.1 — the context strategy seam (R-CTX-001..005). AG-18 amends
// the verdict deliberately (R-CTX-003 as amended) to carry a compaction
// request.
//
// Mints the seam in one file mirroring failover_policy.go symbol for
// symbol (design.md DD1, DD3): one-method interface, exported-field
// prompt struct, an optional-request verdict struct, shipped no-op,
// var _ guard. Consulted exactly once per LOGICAL turn, at
// Harness.Run's turn boundary (harness.go), never per retry attempt.

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
	// attempt (see the R-RTY-002 argument at harness.go:601-612).
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

// ContextVerdict is Resolve's typed return value. Through AG-17 it
// shipped NO field, so the zero value was the only constructible value
// and a verdict requesting compaction was unconstructible by ANY
// implementation. AG-18 (R-CTX-003 as amended) gives it exactly one
// field, on the FailoverVerdict extension path (failover_policy.go:
// 55-63) that posture was built to admit: "a later version adds route
// and re-budget fields non-breakingly; every existing implementation
// returning the zero verdict keeps compiling." The never-compact
// guarantee moves from the type to the ZERO VALUE — a verdict whose
// Compaction field is nil requests nothing, on every path, forever;
// that is the strongest form still available once the field exists,
// and it is what NoOpContextStrategy below keeps proving.
type ContextVerdict struct {
	// Compaction is the strategy's request to compact, or nil to
	// request nothing (R-CTX-003). Layer 2 never supplies a default
	// for any of CompactionRequest's three injected fields.
	Compaction *CompactionRequest
}

// CompactionRequest is what a strategy supplies to request a
// compaction (AG-18, R-CTX-003, R-CMP-002). Every field is the
// strategy's own — Layer 2 authors no default for any of them, and a
// request with an empty Instruction fails typed rather than acquiring
// a Layer 2-authored one (R-CMP-002).
type CompactionRequest struct {
	// Provider is the model provider the compaction call issues its
	// one non-tool completion request to. It MAY differ from the
	// harness's own Provider — compaction is its own model call
	// (R-CMP-002).
	Provider ai.ModelProvider

	// Options carries the compaction call's model/max-tokens surface.
	// It reuses TurnOptions unchanged (no narrower type is
	// introduced); Tools and Continuation MUST be zero/nil — asserted
	// at runtime in compaction.go, not narrowed at the type level —
	// since a compaction call MUST NOT enter the tool-continuation
	// path (R-CMP-002).
	Options TurnOptions

	// Instruction is the summarization instruction, injected by
	// Layer 3 and never authored, prefixed, suffixed or defaulted by
	// Layer 2 on any path, including error and fallback paths
	// (R-CMP-002). Required: an empty Instruction fails typed before
	// any provider request is issued (S-CMP-005).
	Instruction string

	// Cut is the naive boundary the strategy is requesting: compact
	// transcript[0:Cut). It is denominated in the transcript the
	// prompt carried (ContextPrompt.Transcript) — the only coordinate
	// system the strategy can name, since Resolve never sees harness
	// state (R-CTX-002).
	Cut int
}

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
