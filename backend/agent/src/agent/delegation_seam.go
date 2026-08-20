// Package agent is Layer 2 of the cachicamas agent stack. This file
// (delegation_seam.go) hosts AG-19 — the delegation seam: the one
// sanctioned door from inside a tool's Run frame back onto the
// hosting run's event lane (R-DEL-001, design AD-1, AD-2, AD-5).
//
// Everything that knows what a subagent *is* lives in package
// agent_test, which production code cannot import (charter
// 0003:1794/1803, enforced by the compiler rather than by
// convention). This file's whole exported surface — DelegationSeam,
// DelegationSeamFrom, and the two sentinel errors below — publishes
// *events*. It names no subagent concept: no depth, no
// configuration, no child lifecycle.
//
// The seam rides the scheduler's existing emission funnel
// (scheduler.go's `emissions` channel → runDispatcher → sink); it
// introduces no new channel, no new loss rule, and no second
// stamping writer (R-AGE-017, exercised unamended by this change's
// agent-event-delivery delta). The concrete type, its context key,
// its constructor and its installer are all unexported: no code
// outside this package can construct or install a seam — only
// executeCall's per-call installation (scheduler.go) mints one
// (R-DEL-001).
package agent

import (
	"context"
	"errors"
	"sync"
)

// ErrDelegationInadmissible is the sentinel a Publish call returns
// when the offered Event's kind may never cross onto a parent's lane
// (R-DEL-002, design AD-2): a programming error a caller must not
// retry, distinguishable from ErrDelegationRevoked by errors.Is.
var ErrDelegationInadmissible = errors.New("agent: delegation seam refused event kind (inadmissible)")

// ErrDelegationRevoked is the sentinel a Publish call returns once
// the hosting tool call has completed or detached (R-DEL-003, design
// AD-1): a timing outcome a caller must tolerate by simply stopping,
// distinguishable from ErrDelegationInadmissible by errors.Is.
var ErrDelegationRevoked = errors.New("agent: delegation seam revoked (hosting call has completed)")

// DelegationSeam is the re-entrancy seam a tool reaches through its
// context: the one sanctioned door from inside tool.Run back onto the
// parent's event lane. It names no subagent concept (R-DEL-001).
type DelegationSeam interface {
	// Parent reports the identity of the run and turn hosting this
	// tool call — the identities a tool cannot otherwise obtain
	// (Tool.Run receives neither), and without which
	// NewSubagentStarted/NewSubagentEnded are unconstructible.
	Parent() (RunID, TurnID)

	// Publish offers one event to the parent's lane. It returns
	// ErrDelegationInadmissible or ErrDelegationRevoked; a nil return
	// means the event reached the dispatcher funnel.
	Publish(ev Event) error
}

// delegationSeamKey is the unexported context key type installed
// seam values are stored under. Because it is unexported, no code
// outside this package can construct a matching key, so no external
// caller can install a seam onto a context the scheduler will honour
// (R-DEL-001).
type delegationSeamKey struct{}

// delegationSeam is the unexported, unforgeable concrete
// implementation (design AD-5). Revocation is a mutex-guarded latch,
// and the channel send happens UNDER THE SAME MUTEX as the revoked
// check — deliberately, per AD-1's ordering proof: this is what
// makes a publish racing revocation either complete strictly before
// close(emissions), or observe revoked and send nothing, with no
// third interleaving.
type delegationSeam struct {
	runID     RunID
	turnID    TurnID
	emissions chan<- emission

	mu      sync.Mutex
	revoked bool
}

var _ DelegationSeam = (*delegationSeam)(nil)

// newDelegationSeam constructs a seam for exactly one hosting call.
// Unexported: only executeCall (scheduler.go) may mint one.
func newDelegationSeam(runID RunID, turnID TurnID, emissions chan<- emission) *delegationSeam {
	return &delegationSeam{runID: runID, turnID: turnID, emissions: emissions}
}

// withDelegationSeam returns a context carrying seam, reachable only
// through DelegationSeamFrom. Unexported: only executeCall installs
// one.
func withDelegationSeam(ctx context.Context, seam *delegationSeam) context.Context {
	return context.WithValue(ctx, delegationSeamKey{}, seam)
}

// DelegationSeamFrom returns the seam installed for this tool call,
// and whether one is installed. The seam's single type assertion
// lives here, on the seam's own type — scheduler.go gains zero type
// assertions from this change (R-TLS-002, S-TLS-020).
func DelegationSeamFrom(ctx context.Context) (DelegationSeam, bool) {
	seam, ok := ctx.Value(delegationSeamKey{}).(DelegationSeam)
	return seam, ok
}

// Parent implements DelegationSeam.
func (s *delegationSeam) Parent() (RunID, TurnID) {
	return s.runID, s.turnID
}

// Publish implements DelegationSeam (R-DEL-002, R-DEL-003, design
// AD-1). Admissibility is checked first and needs no lock — it reads
// only ev and the package-level registry. The revoked check and the
// channel send are then performed under one critical section, so a
// publish either completes its send strictly before a concurrent
// revoke() returns, or observes revoked and sends nothing — never a
// panic, never a silent drop.
func (s *delegationSeam) Publish(ev Event) error {
	if err := admissible(ev); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.revoked {
		return ErrDelegationRevoked
	}
	s.emissions <- emission{ev: ev} // send under the lock — deliberate (AD-1).
	return nil
}

// revoke latches the seam as revoked (R-DEL-003). Unexported: only
// executeCall's deferred call (scheduler.go) invokes it, registered
// after defer recoverCall so LIFO runs revocation before panic
// recovery's own unwinding completes.
func (s *delegationSeam) revoke() {
	s.mu.Lock()
	s.revoked = true
	s.mu.Unlock()
}

// admissible implements R-DEL-002's five-gate total rule (design
// AD-2): an event is admitted iff, in order, (1) its kind has a
// registry entry, (2) the registered bracket role is none, (3) the
// registered cardinality is not at-most-one, (4) the registered
// descriptor is not terminal, and (5) the kind is not cost_turn
// (R-CST-004's guard — this gate alone is requirement-level rather
// than stream-legality-level: the funnel this seam rides terminates
// in the one channel whose per-attempt forwarder folds cost_turn by
// payload type with no run-identity filter, harness.go:633-635).
//
// Gate 1 is what makes the rule total: a zero Event reports kind 0,
// which has no registry row, so it is refused here rather than
// admitted by its zero-valued descriptor fields (bracket-none,
// cardinality-any, non-terminal) the way gates 2-4 alone would.
//
// Derived from the registry itself, never a hand-maintained kind
// list: a kind registered by a later milestone is decided by its own
// descriptor, with no edit to this function.
func admissible(ev Event) error {
	entry, ok := eventRegistryEntry(ev.Kind())
	if !ok {
		return ErrDelegationInadmissible
	}
	d := entry.descriptor
	if d.Bracket != BracketRoleNone {
		return ErrDelegationInadmissible
	}
	if d.Cardinality == CardinalityAtMostOne {
		return ErrDelegationInadmissible
	}
	if d.Terminal {
		return ErrDelegationInadmissible
	}
	if ev.Kind() == EventKindCostTurn {
		return ErrDelegationInadmissible
	}
	return nil
}
