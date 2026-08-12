// AG-04.2 — the turn lifecycle family (R-AEV-005, VL2-EVT-03, VL2-EVT-11).
//
// A turn nests strictly inside the run bracket and never overlaps another
// turn for one harness (design AD-1's scope engine enforces both). Turn-
// end carries the turn's typed outcome, distinguishing "model finished"
// from "turn aborted".
//
// # Failure coupling on abort — a resolved design gap
//
// Design AD-2 pins run-end's failure rule explicitly ("f required iff
// RunOutcomeFailed, forbidden otherwise") but states no equivalent "iff"
// rule for turn-end. This file resolves it, mirroring run-end's rule
// exactly for internal consistency: a [Failure] is required when the
// outcome is [TurnOutcomeAborted] and forbidden otherwise. Recorded as a
// design-gap resolution in apply-progress.md, not a silent choice — a
// later milestone that needs an unmodelled abort (interruption with no
// captured failure) amends this rule in its own change, per the
// forward-fix preference.
package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TurnStart is the payload opening a turn nested inside the run bracket
// (VL2-EVT-03). It carries no data of its own: a turn's identity is an
// envelope field, read through [Event.Turn].
type TurnStart struct{}

func (TurnStart) kind() EventKind { return EventKindTurnStart }

// validate reports TurnStart's own broken rule, or nil. TurnStart carries
// no payload-intrinsic data; its envelope identity (run, turn) is
// validated by the constructor that receives it.
func (TurnStart) validate(ai.Path) *ai.Violation { return nil }

// NewTurnStart constructs a turn-start event. Both run and turn are
// required. On failure the zero [Event] is returned.
func NewTurnStart(run RunID, turn TurnID) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	return Event{payload: TurnStart{}, run: run, turn: turn, hasTurn: true}, nil
}

// TurnStart returns the event's turn-start payload, and whether the event
// carries one. Called on an event of another kind, or on an event that
// was never constructed, it returns the zero [TurnStart] and false, and
// never panics.
func (e Event) TurnStart() (TurnStart, bool) {
	payload, ok := e.payload.(TurnStart)
	return payload, ok
}

// String renders the turn-start payload for a diagnostic reader, naming
// the type and nothing it carries.
func (TurnStart) String() string { return "turn_start" }

// GoString renders the turn-start payload for the %#v verb.
func (s TurnStart) GoString() string { return s.String() }

// TurnOutcome is the turn's typed terminal outcome (VL2-EVT-11), carried
// by [TurnEnd]. The zero value is not a member: [NewTurnEnd] rejects it,
// so there is no route to a turn-end event with no outcome (R-AEV-005,
// S-AEV-044).
type TurnOutcome uint8

const (
	_ TurnOutcome = iota // the zero value names no outcome

	// TurnOutcomeFinished is the outcome for a turn the model finished
	// normally.
	TurnOutcomeFinished

	// TurnOutcomeAborted is the outcome for a turn that was aborted. A
	// turn-end carrying this outcome MUST also carry a [Failure]; every
	// other outcome MUST NOT (this file's own resolved design gap, above).
	TurnOutcomeAborted

	// turnOutcomeLimit is the first value outside the vocabulary. It must
	// stay last.
	turnOutcomeLimit
)

// String renders the outcome, or "unset"/"turnoutcome(N)" for a
// non-member.
func (o TurnOutcome) String() string {
	switch o {
	case TurnOutcomeFinished:
		return "finished"
	case TurnOutcomeAborted:
		return "aborted"
	case 0:
		return "unset"
	default:
		return "turnoutcome(" + strconv.FormatUint(uint64(o), 10) + ")"
	}
}

// TurnEnd is the payload closing a turn (VL2-EVT-03), carrying the turn's
// typed outcome and, iff the outcome is [TurnOutcomeAborted], the
// [Failure] that caused it.
type TurnEnd struct {
	outcome TurnOutcome
	failure *Failure
}

func (TurnEnd) kind() EventKind { return EventKindTurnEnd }

// validate reports TurnEnd's own first broken rule, or nil: the outcome is
// a member of the closed vocabulary, and the failure is present iff the
// outcome is [TurnOutcomeAborted].
func (e TurnEnd) validate(at ai.Path) *ai.Violation {
	if e.outcome == 0 || e.outcome >= turnOutcomeLimit {
		return ai.Invalid(ai.ErrNotInVocabulary, under(at, ai.At("outcome"))...)
	}
	if e.outcome == TurnOutcomeAborted && e.failure == nil {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("failure"))...)
	}
	if e.outcome != TurnOutcomeAborted && e.failure != nil {
		return ai.Invalid(ai.ErrMisplaced, under(at, ai.At("failure"))...)
	}
	return nil
}

// NewTurnEnd constructs the event closing a turn (R-AEV-005). run and turn
// are required. f is required when outcome is [TurnOutcomeAborted] and
// forbidden otherwise. On failure the zero [Event] is returned.
func NewTurnEnd(run RunID, turn TurnID, outcome TurnOutcome, f *Failure) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := TurnEnd{outcome: outcome, failure: f}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// TurnEnd returns the event's turn-end payload, and whether the event
// carries one. Called on an event of another kind, or on an event that
// was never constructed, it returns the zero [TurnEnd] and false, and
// never panics.
func (e Event) TurnEnd() (TurnEnd, bool) {
	payload, ok := e.payload.(TurnEnd)
	return payload, ok
}

// Outcome returns the turn's typed terminal outcome (R-AEV-005).
func (e TurnEnd) Outcome() TurnOutcome { return e.outcome }

// Failure returns the failure that caused the turn to abort, and whether
// one is carried. Only a [TurnOutcomeAborted] turn-end carries one.
func (e TurnEnd) Failure() (*Failure, bool) { return e.failure, e.failure != nil }

// String renders the turn-end payload for a diagnostic reader, naming the
// type and its outcome — never a failure's own content.
func (e TurnEnd) String() string { return "turn_end(" + e.outcome.String() + ")" }

// GoString renders the turn-end payload for the %#v verb.
func (e TurnEnd) GoString() string { return e.String() }
