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

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

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

// RunOutcome is the run's typed terminal outcome (VL2-EVT-10), carried by
// [RunEnd]. The zero value is not a member: [NewRunEnd] rejects it, so
// there is no route to a run-end event with no outcome (R-AEV-004,
// S-AEV-035).
type RunOutcome uint8

const (
	_ RunOutcome = iota // the zero value names no outcome

	// RunOutcomeCompleted is the outcome for a run that finished normally.
	RunOutcomeCompleted

	// RunOutcomeInterrupted is the outcome for a run that was interrupted
	// before it finished.
	RunOutcomeInterrupted

	// RunOutcomeFailed is the outcome for a run that ended in failure. A
	// run-end carrying this outcome MUST also carry a [Failure]
	// (R-AEV-004); every other outcome MUST NOT.
	RunOutcomeFailed

	// runOutcomeLimit is the first value outside the vocabulary. It must
	// stay last.
	runOutcomeLimit
)

// String renders the outcome, or "unset"/"runoutcome(N)" for a
// non-member, mirroring `ai.FailureCategory.String`'s posture: a
// diagnostic rendering must never round-trip into a value.
func (o RunOutcome) String() string {
	switch o {
	case RunOutcomeCompleted:
		return "completed"
	case RunOutcomeInterrupted:
		return "interrupted"
	case RunOutcomeFailed:
		return "failed"
	case 0:
		return "unset"
	default:
		return "runoutcome(" + strconv.FormatUint(uint64(o), 10) + ")"
	}
}

// RunEnd is the payload closing the run bracket (VL2-EVT-02), carrying the
// run's typed outcome and, iff the outcome is [RunOutcomeFailed], the
// [Failure] that caused it.
type RunEnd struct {
	outcome RunOutcome
	failure *Failure
}

func (RunEnd) kind() EventKind { return EventKindRunEnd }

// validate reports RunEnd's own first broken rule, or nil: the outcome is
// a member of the closed vocabulary, and the failure is present iff the
// outcome is [RunOutcomeFailed] — required for that outcome, forbidden for
// every other one.
func (r RunEnd) validate(at ai.Path) *ai.Violation {
	if r.outcome == 0 || r.outcome >= runOutcomeLimit {
		return ai.Invalid(ai.ErrNotInVocabulary, under(at, ai.At("outcome"))...)
	}
	if r.outcome == RunOutcomeFailed && r.failure == nil {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("failure"))...)
	}
	if r.outcome != RunOutcomeFailed && r.failure != nil {
		return ai.Invalid(ai.ErrMisplaced, under(at, ai.At("failure"))...)
	}
	return nil
}

// NewRunEnd constructs the event closing the run bracket (R-AEV-004).
//
// f is required when outcome is [RunOutcomeFailed] and forbidden
// otherwise — the fourth combination (a non-failed outcome carrying a
// failure) is unconstructible here, not merely undocumented, mirroring
// `ai.PreStreamFailure`'s own posture for its own impossible cell. On
// failure the zero [Event] is returned.
func NewRunEnd(run RunID, outcome RunOutcome, f *Failure) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	payload := RunEnd{outcome: outcome, failure: f}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run}, nil
}

// RunEnd returns the event's run-end payload, and whether the event
// carries one. Called on an event of another kind, or on an event that
// was never constructed, it returns the zero [RunEnd] and false, and
// never panics.
func (e Event) RunEnd() (RunEnd, bool) {
	payload, ok := e.payload.(RunEnd)
	return payload, ok
}

// Outcome returns the run's typed terminal outcome (R-AEV-004).
func (r RunEnd) Outcome() RunOutcome { return r.outcome }

// Failure returns the failure that caused the run to end, and whether one
// is carried. Only a [RunOutcomeFailed] run-end carries one.
func (r RunEnd) Failure() (*Failure, bool) { return r.failure, r.failure != nil }

// String renders the run-end payload for a diagnostic reader, naming the
// type and its outcome — never a failure's own content.
func (r RunEnd) String() string { return "run_end(" + r.outcome.String() + ")" }

// GoString renders the run-end payload for the %#v verb. Without it, %#v
// falls back to reflection and would print the wrapped failure — the same
// redaction posture `ai.Failure.GoString` exists for.
func (r RunEnd) GoString() string { return r.String() }
