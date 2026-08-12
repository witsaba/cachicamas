// AG-04.1/AG-04.2 — the run lifecycle family (R-AEV-004, VL2-EVT-02,
// VL2-EVT-10).
//
// The run bracket has exactly one opener and exactly one closer per run:
// [NewRunStart] (or its delegated form, [NewDelegatedRunStart] — the only
// door that sets an event's parent identity at AG-04, R-AEV-003) and
// [NewRunEnd], which carries the run's typed [RunOutcome]. Both payloads
// are markers with no payload-intrinsic data of their own: everything they
// need to validate is either an envelope identity field (validated by the
// constructor that receives it) or, for run-end, the outcome and its
// optional [Failure].
//
// # Sequencing note
//
// [RunStart] and its two constructors land at AG-04.1, ahead of
// run_events.go's design-time node attribution to AG-04.2, because
// AG-04.1's own envelope scenarios (R-AEV-001 kind derivation, R-AEV-003
// parent-before-delegation) need at least one constructible event kind to
// test against the public surface. [RunEnd] and [RunOutcome] land at
// AG-04.2 alongside the validator's bracket engine, which is the first
// consumer of a complete run. Recorded in apply-progress.md as the same
// shape of reconciliation tasks.md already documents for stream_check.go's
// own two-commit split.
package agent

import "github.com/cachicamas/backend/agent/src/ai"

// RunStart is the payload opening the run bracket (VL2-EVT-02). It carries
// no data of its own: a run's identity and its optional parent identity
// are envelope fields, read through [Event.Run] and [Event.Parent].
type RunStart struct{}

func (RunStart) kind() EventKind { return EventKindRunStart }

// validate reports RunStart's own broken rule, or nil. RunStart carries no
// payload-intrinsic data, so there is nothing to check here: its envelope
// identity (run, absent-or-present parent) is validated by the constructor
// that receives it, per design AD-1.
func (RunStart) validate(ai.Path) *ai.Violation { return nil }

// NewRunStart constructs a top-level run-start event (R-AEV-001,
// R-AEV-003): run is required, and the event carries no parent.
//
// On failure the zero [Event] is returned, so a caller that ignores the
// error cannot mistake the result for a constructed event. The returned
// event is unstamped (sequence 0); stamping is a [LaneStamper]'s job, per
// lane, at the producer boundary.
func NewRunStart(run RunID) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	return Event{payload: RunStart{}, run: run}, nil
}

// NewDelegatedRunStart constructs a run-start event belonging to a run
// delegated from parent (R-AEV-003, VL2-EVT-13). It is the only door in
// the public surface that sets an event's parent identity at AG-04 — no
// delegation or subagent mechanism exists yet; the field exists now
// because explicit nesting cannot be retrofitted (S-AEV-022).
//
// Both run and parent are required. On failure the zero [Event] is
// returned.
func NewDelegatedRunStart(run, parent RunID) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if parent == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("parent"))
	}
	return Event{payload: RunStart{}, run: run, parent: parent, hasParent: true}, nil
}

// RunStart returns the event's run-start payload, and whether the event
// carries one. Called on an event of another kind, or on an event that was
// never constructed, it returns the zero [RunStart] and false, and never
// panics.
func (e Event) RunStart() (RunStart, bool) {
	payload, ok := e.payload.(RunStart)
	return payload, ok
}

// String renders the run-start payload for a diagnostic reader, naming the
// type and nothing it carries.
func (RunStart) String() string { return "run_start" }

// GoString renders the run-start payload for the %#v verb.
func (s RunStart) GoString() string { return s.String() }
