// AG-04.2/AG-04.3 — the typed-failure surface (R-AEV-008, design AD-2).
//
// [Failure] is a thin wrap holding an unexported `*ai.Failure`, never a
// parallel category/cause vocabulary (S-AGV-020). It is carried by a
// run-end payload when its [RunOutcome] is [RunOutcomeFailed] and by a
// turn-end payload when its [TurnOutcome] is [TurnOutcomeAborted] — AG-04
// registers no separate error kind; failures ride the typed outcomes.
//
// # Sequencing note
//
// This type lands ahead of its design-time node attribution to AG-04.3,
// for the same structural reason [RunStart] moved to AG-04.1: [RunEnd] and
// [TurnEnd] (AG-04.2) both carry a `*Failure` field per design AD-2's own
// signature (`NewRunEnd(run RunID, outcome RunOutcome, f *Failure)`), so
// the type must exist before AG-04.2's payloads compile. Its own dedicated
// scenarios (S-AEV-070..073, invariant_pin_test.go) still land at AG-04.3
// as tasks.md planned — only the type's existence moved earlier. Recorded
// in apply-progress.md alongside the identical RunStart reconciliation.
package agent

import "github.com/cachicamas/backend/agent/src/ai"

// Failure is Layer 2's typed-failure surface (R-AEV-008): a thin wrap
// holding the `*ai.Failure` it delegates every accessor to, so Layer 2
// never re-declares Layer 1's closed nine-member category vocabulary or
// its per-category sentinels.
type Failure struct{ wrapped *ai.Failure }

// NewFailure wraps f as Layer 2's typed-failure surface. f is required:
// a nil f is rejected, so a caller cannot construct a [Failure] that
// wraps nothing.
func NewFailure(f *ai.Failure) (*Failure, error) {
	if f == nil {
		return nil, ai.Invalid(ai.ErrEmpty, ai.At("failure"))
	}
	return &Failure{wrapped: f}, nil
}

// Category returns the failure's classification, unchanged from Layer 1's
// closed vocabulary (R-AEV-008) — the "mapping" S-AEV-072 requires is
// identity: Layer 2 declares no category of its own. A nil *Failure
// reports the zero [ai.FailureCategory] rather than panicking.
func (f *Failure) Category() ai.FailureCategory {
	if f == nil {
		return 0
	}
	return f.wrapped.Category()
}

// Delivery reports which carrier handed this failure over — pre-stream or
// mid-stream, VL2-EVT-15 — distinguishable without parsing any text. A
// nil *Failure reports the zero [ai.DeliveryPath] rather than panicking.
func (f *Failure) Delivery() ai.DeliveryPath {
	if f == nil {
		return 0
	}
	return f.wrapped.Delivery()
}

// Retryable reports the machine-readable retry signal, delegated
// unchanged from Layer 1. A nil *Failure reports false rather than
// panicking.
func (f *Failure) Retryable() bool {
	if f == nil {
		return false
	}
	return f.wrapped.Retryable()
}

// Unwrap returns the wrapped `*ai.Failure` as an error, keeping the whole
// `errors.Is`/`errors.As` sentinel chain of Layer 1's provider-failure
// taxonomy reachable through the envelope. A nil *Failure unwraps to nil
// rather than panicking.
func (f *Failure) Unwrap() error {
	if f == nil {
		return nil
	}
	return f.wrapped
}
