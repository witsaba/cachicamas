// Package agent is Layer 2 of the cachicamas agent stack. This file
// (permission_protocol.go) hosts AG-10 — the ask–suspend–resume seam
// Layer 2 owns the protocol half of (doc 0001 § 4.1, § 5.1; doc 0003
// AG-10, R-APP-001..012; spec `agent-permission-protocol/spec.md`).
//
// The seam ties AG-06.1's permission event family to AG-09's hand-rolled
// scheduler: each scheduled tool call consults an injected
// `PermissionPolicy` before `ToolStart`, sync verdicts proceed,
// `Defer` parks the call on a per-call `chan struct{}` keyed by
// `callID` while siblings continue. The parked-set lifetime is one
// `Schedule` call (R-LSK-002 carry); the loop owns the upward-path
// wake.
//
// # Why an interface (D1, design's intentional mismatch with AG-08)
//
// `PermissionPolicy` is an interface — not the function-value
// discipline AG-08 used for `PreRequestHook` — because the policy
// carries internal state a function value cannot hold: remembered
// rules (AG-10.4), a mode flag (Layer 3's concern), an event log a
// harness can replay. The mismatch is intentional, not drift. Layer 3
// (doc 0004 CO-03) supplies the implementation; Layer 2 owns the
// protocol that drives it.
//
// # Substrate preservation (NFR-TLS-003 7th carry)
//
// This file introduces no new top-level Go dependencies, no new
// `EventKind` constant, no new `PermissionOutcome` member — AG-06.1's
// event family stays byte-clean producers of the surface AG-10 emits.
// The substrate filter widens at apply time to exclude this file
// and `permission_protocol_test.go`.
package agent

import (
	"context"
	"errors"
	"sync"

	"github.com/cachicamas/backend/agent/src/ai"
)

// Compile-time guard: ai.ToolCall must be reachable from this package —
// the PermissionPolicy.Resolve signature takes a ToolCall, mirroring the
// scheduler's call-slice signature (scheduler.go:90). The same import
// boundary (tool.go:204) carries the guard for the existing surface; the
// alias `_ = ai.ToolCall{}` is the visible form.
var _ = ai.ToolCall{}

// ErrStrayDecision is the typed sentinel the upward-path wake hand-off
// returns when an event identifies a `callID` that has no parked
// channel registered (R-APP-003, S-PPB-003 bite). It is a typed error,
// not a silent drop and not a panic — the loop or scheduler that wakes
// a parked call surfaces it as a typed protocol rejection.
//
// Distinct from any other typed failure Layer 2 emits (the typed
// *Failure on a Deny verdict, the typed *Failure on a tool's
// `Run` error, the typed *Failure on a registry miss): ErrStrayDecision
// is a protocol-layer rejection, not a tool-execution-layer rejection.
var ErrStrayDecision = errors.New("agent: stray permission decision for non-parked callID")

// PermissionPolicy is the ask–suspend–resume seam AG-10 introduces
// (doc 0001 § 5.1 "PermissionPolicy" port; Layer 3-implemented).
//
// Two methods:
//
//   - Resolve returns one of five typed verdicts for one tool call.
//     AllowOnce / AllowAlways / Deny / ModifyInput are synchronous —
//     the scheduler emits no `decision_required` event; the call
//     proceeds (or is replaced by a typed failure for Deny /
//     substituted-arguments for ModifyInput). Defer parks the call
//     until the upward-path wake closes the per-call `chan` keyed
//     by `callID`.
//   - Remember is invoked after AllowAlways reaches the policy. The
//     policy reports whether it committed the rule to its persistent
//     store. A true return causes the scheduler to emit
//     `permission_resolution_remembered`; a false return suppresses
//     the emission (R-APP-007, `CardinalityAtMostOne` preservation).
//
// Layer 3 implements (doc 0004 CO-03). Layer 2 MUST NOT define rule
// sets or mode flags — those are out of scope (AG-10.0 non-reqs).
type PermissionPolicy interface {
	// Resolve asks the policy for a verdict on one call. The
	// returned verdict's Outcome discriminates the four typed
	// outcomes + Defer. ModifiedArgs is populated iff Outcome ==
	// PermissionOutcomeModifyInput; Failure is populated iff
	// Outcome == PermissionOutcomeDeny.
	Resolve(ctx context.Context, call ai.ToolCall) PermissionVerdict

	// Remember is invoked after a successful AllowAlways decision
	// reaches the policy. Returns true iff the policy committed
	// the rule to its persistent store; the scheduler emits
	// permission_resolution_remembered only on true (preserves
	// CardinalityAtMostOne, S-APE-082 carry).
	Remember(ctx context.Context, toolName string, outcome PermissionOutcome) bool
}

// PermissionVerdict is the typed outcome `PermissionPolicy.Resolve`
// returns (AG-10.1). Wraps the four typed outcomes from
// AG-06.1's closed vocabulary plus a Defer value that is NOT a
// member of `PermissionOutcome` (Defer is the protocol's own
// state — not a decision the policy reaches, just a signal that
// the policy wants a human).
type PermissionVerdict struct {
	// Outcome is one of the five typed values Resolve reaches:
	// the four `PermissionOutcome` members + the protocol's own
	// Defer. Defer is encoded as the zero `PermissionOutcome`
	// value (out-of-vocabulary for AG-06.1, so it cannot collide
	// with a decision outcome).
	Outcome PermissionOutcome

	// ModifiedArgs is the substituted arguments for a ModifyInput
	// outcome; populated iff Outcome == PermissionOutcomeModifyInput.
	// The scheduler passes these bytes to `Tool.Run` and emits a
	// `ToolStart` carrying the modified bytes (AG-10.2 transparency).
	ModifiedArgs []byte

	// Failure is the typed denial payload for a Deny outcome;
	// populated iff Outcome == PermissionOutcomeDeny.
	// The scheduler populates the call's ordinal `Result` slot
	// with `Result{Outcome: ExecutionFailure, Failure: ...}` so
	// the model sees the denial as a typed outcome (R-APP-005).
	Failure *Failure
}

// PermissionDefer is the sentinel PermissionOutcome value the policy
// returns to request a human decision. It is the zero value of the
// typed enum; the AG-06.1 vocabulary's validator rejects zero as
// "not a member" for `decision_made` events, so Defer cannot be
// confused with a typed outcome (R-APP-002).
//
// Defined as a constant for documentation rather than as a member;
// the protocol reads `Outcome == 0` directly. Mirrors AG-06.1's
// "zero is not a member" posture (`permission_events.go:55-58`).
const PermissionDefer PermissionOutcome = 0

// parkedSet is the per-Schedule coordination primitive for calls
// that the policy wants to defer. Keyed by `callID`; the value is a
// per-call `chan struct{}` the call goroutine blocks on — the close
// itself is the signal.
//
// The set lives one Schedule call (R-LSK-002 carry). D-B: park
// release has exactly two production paths — Scheduler.WakeParked
// (closes the specific callID's channel) and the gate's own
// `case <-ctx.Done()` branch (AG-10.3 cancellation, R-APP-009). A
// parked goroutine that observes neither never calls wg.Done(), so
// `Schedule` cannot return while it is still parked — by
// construction there is nothing for a defensive close-everything
// sweep at Schedule exit to ever unblock (an earlier AG-10.3 draft
// carried one; deleted, D-B, once this was shown).
//
// # Concurrency
//
// The set is shared across all call goroutines spawned by one
// `Schedule` call (R-TLS-009's three sub-paths + AG-10's parked
// set). Accesses are guarded by a `sync.Mutex` (D6b carry): every
// map mutation (park / wake) acquires the lock, every read
// (parkedCount / wake lookup) acquires the lock.
type parkedSet struct {
	mu       sync.Mutex
	channels map[string]chan struct{}
}

// newParkedSet constructs an empty parkedSet. The caller owns the
// set's lifetime (one Schedule call).
func newParkedSet() *parkedSet {
	return &parkedSet{channels: make(map[string]chan struct{})}
}

// park returns a fresh per-call channel registered under callID.
// The returned channel is the one the call goroutine blocks on; the
// upward-path wake (Scheduler.WakeParked) closes it.
func (p *parkedSet) park(callID string) chan struct{} {
	ch := make(chan struct{})
	p.mu.Lock()
	p.channels[callID] = ch
	p.mu.Unlock()
	return ch
}

// wake closes the channel registered under callID and returns
// whether one was registered. A wake to an unknown callID is the
// typed rejection AG-10.1 commits to (R-APP-003, S-PPB-003 bite) —
// and, per the lookup-then-delete ordering under one lock
// acquisition, touches no other entry in the set (S-APP-004).
func (p *parkedSet) wake(callID string) bool {
	p.mu.Lock()
	ch, ok := p.channels[callID]
	if !ok {
		p.mu.Unlock()
		return false
	}
	delete(p.channels, callID)
	p.mu.Unlock()
	close(ch)
	return true
}

// parkedCount reports how many calls are currently parked. Test
// inspection surface (R-TLS-008 source-guard bite).
func (p *parkedSet) parkedCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.channels)
}
