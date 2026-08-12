// AG-04.1 — the event envelope: one opaque value type, readable and sealed
// (R-AEV-001, R-AEV-003).
//
// This file mirrors `backend/agent/src/ai/event.go`'s shape: a kind that
// cannot disagree with what it names, and no valid value that skipped
// construction. Two differences from Layer 1, both deliberate:
//
//   - Layer 2's kind vocabulary is its own closed set, independent of
//     `ai.EventKind` — no alias, no re-export, no extension (S-AGV-020,
//     S-AGV-021).
//   - Run identity, turn identity and parent identity are envelope data,
//     not payload data (design AD-1): they live on [Event] itself, so the
//     validator can read them without ever reading a concrete payload
//     type, and so every registered kind — including a future one with no
//     payload-intrinsic data at all — carries them uniformly.
//
// # The kind is derived, never stored
//
// [Event] holds no kind field. [Event.Kind] asks the payload, so the
// discriminator and the payload cannot disagree. An event that carries no
// payload therefore has no kind, which is what makes an unconstructed
// value fail the closed-vocabulary rule at [CheckEmit] rather than needing
// a rule class of its own (R-AEV-001).
//
// # The parent identifier exists before delegation does
//
// [Event.Parent] is readable from AG-04's birth even though no delegation
// mechanism exists yet (R-AEV-003): explicit nesting cannot be retrofitted,
// so the field is part of the envelope now, and delegation semantics are
// AG-19's.

package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// EventKind is the closed discriminator naming which payload an event
// carries (R-AEV-001). It is Layer 2's own vocabulary: no member is an
// alias, re-export or extension of an `ai.EventKind` member (S-AGV-020,
// S-AGV-021).
//
// It is derived from the payload rather than stored beside it, so a kind
// and a payload can never disagree. The zero value is not a member: it is
// the kind of an event that was never constructed, and [CheckEmit] rejects
// it wherever an event is offered at the producer's emission boundary.
//
// # Registered kinds (R-AEV-010 — exactly these four, no more)
//
//   - run_start — opens the run bracket (VL2-EVT-02).
//   - run_end — closes the run bracket, carrying the run's typed outcome
//     (VL2-EVT-02, VL2-EVT-10).
//   - turn_start — opens a turn nested inside the run bracket (VL2-EVT-03).
//   - turn_end — closes a turn, carrying the turn's typed outcome
//     (VL2-EVT-03, VL2-EVT-11).
//
// AG-04 registers no message, tool, permission, cost, delegation or
// compaction kind under any name — those belong to AG-05 and AG-06.
type EventKind uint8

const (
	// EventKindRunStart is the kind opening the run bracket. See
	// run_events.go.
	EventKindRunStart EventKind = iota + 1

	// EventKindRunEnd is the kind closing the run bracket, carrying the
	// run's typed outcome. Terminal. See run_events.go.
	EventKindRunEnd

	// EventKindTurnStart is the kind opening a turn nested inside the run
	// bracket. See turn_events.go.
	EventKindTurnStart

	// EventKindTurnEnd is the kind closing a turn, carrying the turn's
	// typed outcome. See turn_events.go.
	EventKindTurnEnd

	// eventKindFirst and eventKindEnd bound the declared production
	// constant space, mirroring `ai.eventKindFirst`/`eventKindEnd`.
	eventKindFirst = EventKindRunStart
	eventKindEnd   = EventKindTurnEnd + 1
)

// eventRegistration is one row of the kind registry: a kind's name and its
// payload-independent ordering descriptor. Pairing them in one table is
// what makes registering a kind without a descriptor structurally
// impossible.
type eventRegistration struct {
	name       string
	descriptor EventDescriptor
}

// eventRegistry is the kind registration table, indexed by the kind
// constant itself, mirroring `ai.eventRegistry`. Index 0 is the permanent
// placeholder for the zero (non-member) kind.
var eventRegistry = []eventRegistration{
	EventKindRunStart: {
		name:       "run_start",
		descriptor: EventDescriptor{Bracket: BracketRoleOpensRun},
	},
	EventKindRunEnd: {
		name:       "run_end",
		descriptor: EventDescriptor{Bracket: BracketRoleClosesRun, Terminal: true},
	},
	EventKindTurnStart: {
		name:       "turn_start",
		descriptor: EventDescriptor{Bracket: BracketRoleOpensTurn},
	},
	EventKindTurnEnd: {
		name:       "turn_end",
		descriptor: EventDescriptor{Bracket: BracketRoleClosesTurn},
	},
}

// eventRegistryEntry returns a kind's registry entry, and whether it has
// one. A kind past the end of the table, like one before eventKindFirst,
// has no entry — which is exactly what makes it not a member of the
// vocabulary.
func eventRegistryEntry(k EventKind) (eventRegistration, bool) {
	if int(k) <= 0 || int(k) >= len(eventRegistry) {
		return eventRegistration{}, false
	}
	return eventRegistry[k], true
}

// EventKinds returns the registered event-kind vocabulary in declaration
// order (R-AEV-010).
//
// The result is a fresh slice on every call: a package-level variable would
// be a closed vocabulary any consumer could rewrite. It enumerates the
// declared constant space (eventKindFirst … eventKindEnd), not the
// registry — mirroring `ai.EventKinds`'s own stated reason: an enumeration
// derived from the table would list exactly the members that have an
// entry, making a constant declared without one invisible to the
// assertion that exists to catch it.
func EventKinds() []EventKind {
	out := make([]EventKind, 0, int(eventKindEnd-eventKindFirst))
	for k := eventKindFirst; k < eventKindEnd; k++ {
		out = append(out, k)
	}
	return out
}

// String renders the kind. A member renders as its registered name. The
// zero value renders as "unset", and anything else as "eventkind(N)" — a
// shape no parser accepts, because a diagnostic rendering must never
// round-trip into a value.
func (k EventKind) String() string {
	if entry, ok := eventRegistryEntry(k); ok {
		return entry.name
	}
	if k == 0 {
		return "unset"
	}
	return "eventkind(" + strconv.FormatUint(uint64(k), 10) + ")"
}

// eventPayload is what an event carries, and it is the reason [Event] is a
// sum type without being an interface. It is unexported and its methods
// are unexported, so no type outside this package can be one.
type eventPayload interface {
	// kind reports the registered kind this payload is.
	kind() EventKind

	// validate reports the payload's own first broken rule at the given
	// position, or nil. It never reads envelope identity (run, turn,
	// parent): those are validated by the constructor that receives them
	// directly (design AD-1 — identity is envelope data, not payload
	// data).
	validate(at ai.Path) *ai.Violation
}

// under extends at with steps, cloning first so the original is never
// mutated through append's capacity reuse — a payload's own validate()
// methods use it to build a sub-position, mirroring `ai.content_part.go`'s
// unexported helper of the same name and shape.
func under(at ai.Path, steps ...ai.Step) ai.Path {
	out := make(ai.Path, len(at), len(at)+len(steps))
	copy(out, at)
	return append(out, steps...)
}

// RunID is the opaque identity of one run (VL2-EVT-02). It is also the
// identity a delegated event's parent field carries (VL2-EVT-13) — a
// parent identifier is always a RunID.
type RunID string

// TurnID is the opaque identity of one turn, nested inside a run
// (VL2-EVT-03).
type TurnID string

// Event is Layer 2's envelope for one item on a lane: a kind, its typed
// payload, the sequence position it was stamped with, and its identity —
// which run it belongs to, which turn (if any), and which run it was
// delegated from (if any).
//
// It is a value with unexported fields, so reading is methods and
// constructing is a constructor. The zero value is reachable and carries
// no payload, therefore no kind, therefore no membership of the closed
// kind vocabulary; [CheckEmit] rejects it.
type Event struct {
	payload   eventPayload
	seq       Sequence
	run       RunID
	turn      TurnID
	hasTurn   bool
	parent    RunID
	hasParent bool
}

// Kind reports which payload this event carries. It is derived from the
// payload on every call and is never stored, so the answer cannot drift
// from the payload it describes. An event that was never constructed
// reports the zero kind, which is not a member of the vocabulary
// (R-AEV-001).
func (e Event) Kind() EventKind {
	if e.payload == nil {
		return 0
	}
	return e.payload.kind()
}

// Sequence reports the position this event was stamped with, or the zero
// [Sequence] — the documented "never stamped" sentinel — if it never was.
func (e Event) Sequence() Sequence { return e.seq }

// Run reports the identity of the run this event belongs to.
func (e Event) Run() RunID { return e.run }

// Turn reports the identity of the turn this event belongs to, and whether
// it belongs to one. A run-scoped event (run-start, run-end) reports false
// (R-AEV-001, S-AEV-004).
func (e Event) Turn() (TurnID, bool) { return e.turn, e.hasTurn }

// Parent reports the run identity this event was delegated from, and
// whether it was delegated at all. A top-level event reports "no parent"
// as this distinguishable false, never an ambiguous zero value (R-AEV-003,
// S-AEV-021).
func (e Event) Parent() (RunID, bool) { return e.parent, e.hasParent }

// String renders the event for a diagnostic reader: "event(<kind>
// run=<run> seq=N)", and nothing a payload carries.
func (e Event) String() string {
	return "event(" + e.Kind().String() + " run=" + string(e.run) + " seq=" + strconv.FormatUint(uint64(e.seq), 10) + ")"
}

// GoString renders the event for the %#v verb. Without it, %#v falls back
// to reflection and prints every field — delegating to the already-total
// [Event.String] keeps that from becoming a property of which verb a
// caller reaches for.
func (e Event) GoString() string { return e.String() }

// CheckEmit is the producer's emission boundary (R-AEV-001): the point
// every event is offered to before it may be treated as part of a stream.
//
// The rules are checked in the order written:
//
//  1. the event carries a payload registered in the kind table, else
//     [ai.ErrNotInVocabulary] at "event" — a value that skipped
//     construction, or one whose payload type this package never
//     registered, has no kind and is not a member of the closed
//     vocabulary;
//  2. the sequence is not the unstamped sentinel 0, else
//     [ai.ErrOutOfRange] at "event", "sequence" — an event offered here
//     must already have passed through a [LaneStamper];
//  3. the payload satisfies its own construction rules.
//
// Envelope identity (run, turn, parent) is validated once, by the
// constructor that receives it directly, not re-validated here: every
// public constructor requires its identity arguments, so there is no
// route through the public surface that reaches CheckEmit with an empty
// run identity on a non-zero-kind event.
func CheckEmit(e Event) error {
	return ai.FirstFailure(
		func() *ai.Violation {
			if _, ok := eventRegistryEntry(e.Kind()); !ok {
				return ai.Invalid(ai.ErrNotInVocabulary, ai.At("event"))
			}
			return nil
		},
		func() *ai.Violation {
			if e.Sequence() == 0 {
				return ai.Invalid(ai.ErrOutOfRange, ai.At("event"), ai.At("sequence"))
			}
			return nil
		},
		func() *ai.Violation {
			return e.payload.validate(ai.Path{ai.At("event")})
		},
	)
}
