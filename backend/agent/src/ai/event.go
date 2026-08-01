// AI-14.1 — the event envelope: one opaque value type, readable and sealed.
//
// This file implements V-STR-10 event and V-STR-11 event kind. It mirrors
// content_part.go's shape exactly, because the property it proves is the
// identical one C1/C2 (doc 0001 § 3.1) named for a content part: a kind that
// cannot disagree with what it names, and no valid value that skipped
// construction.
//
// # The kind is derived, never stored
//
// [Event] holds no kind field. [Event.Kind] asks the payload, so the
// discriminator and the payload cannot disagree — there is only one of them.
// An event that carries no payload therefore has no kind, which is what makes
// an unconstructed value fail the closed-vocabulary rule rather than needing a
// rule class of its own (R-AEE-002).
//
// # Zero production kinds
//
// This milestone registers none (R-AEE-006): response start/completion
// (AI-15), text/reasoning/tool-call blocks (AI-16 … AI-18) and the terminal
// error (AI-19) are none of them declared here. What this file proves —
// derivation, sealing, external readability, exhaustiveness — is proved
// against a test-only witness payload instead, bridged into package ai_test
// through export_test.go. See event_descriptor.go's file comment for the
// six-step procedure a later milestone follows to add a real one.
package ai

import "strconv"

// EventKind is the closed discriminator naming which payload an event
// carries (V-STR-11).
//
// It is derived from the payload rather than stored beside it, so a kind and
// a payload can never disagree. The zero value is not a member: it is the
// kind of an event that was never constructed, and CheckEmit rejects it
// wherever an event is offered at the producer's emission boundary
// (R-AEE-002).
//
// # Registered kinds
//
// None. AI-14 registers zero production event kinds (R-AEE-006). A later
// milestone appends one following event_descriptor.go's six-step procedure;
// none of the requirements in this capability's spec is edited to do it.
type EventKind uint8

const (
	// eventKindFirst and eventKindEnd bound the declared production constant
	// space, mirroring content_part.go's partKindFirst/partKindEnd. AI-14
	// declares no production EventKind constant, so both equal 1 — the value
	// the first real kind will take, and the value the test-only witness
	// (export_test.go) borrows without moving this bound, so [EventKinds]
	// stays empty for a non-test build (S-AEE-013, S-AEE-017).
	eventKindFirst EventKind = 1
	eventKindEnd   EventKind = eventKindFirst
)

// eventRegistration is one row of the kind registry: a kind's name and its
// payload-independent ordering descriptor (R-AEE-014). Pairing them in one
// table is what makes registering a kind without a descriptor structurally
// impossible — there is nowhere else for either to live.
type eventRegistration struct {
	name       string
	descriptor EventDescriptor
}

// eventRegistry is the kind registration table, indexed by the kind constant
// itself — content_part.go's partKindNames shape, for AI-04's stated reason:
// nothing in this package may let an unordered iteration decide anything.
//
// Index 0 is the permanent placeholder for the zero (non-member) kind, so a
// real registration never lands there. It starts as the only entry: AI-14
// registers zero production kinds, and export_test.go is the only file that
// grows it further, for exactly one test-only witness plus whatever a test
// dynamically registers and later unregisters through RegisterTestKind.
var eventRegistry = []eventRegistration{{}}

// eventRegistration returns a kind's registry entry, and whether it has one.
// A kind past the end of the table, like one before eventKindFirst, has no
// entry — which is exactly what makes it not a member of the vocabulary.
func eventRegistryEntry(k EventKind) (eventRegistration, bool) {
	if int(k) <= 0 || int(k) >= len(eventRegistry) {
		return eventRegistration{}, false
	}
	return eventRegistry[k], true
}

// EventKinds returns the production event-kind vocabulary in declaration
// order.
//
// The result is a fresh slice on every call: a package-level variable would
// be a closed vocabulary any consumer could rewrite. It enumerates the
// declared constant space (eventKindFirst … eventKindEnd), not the registry —
// content_part.go's PartKinds documents why: an enumeration derived from the
// table would list exactly the members that have an entry, making a constant
// declared without one invisible to the assertion that exists to catch it.
//
// It is empty at this milestone (R-AEE-006): eventKindFirst == eventKindEnd.
func EventKinds() []EventKind {
	out := make([]EventKind, 0, int(eventKindEnd-eventKindFirst))
	for k := eventKindFirst; k < eventKindEnd; k++ {
		out = append(out, k)
	}
	return out
}

// String renders the kind.
//
// A member renders as its registered name. The zero value renders as
// "unset", and anything else as "eventkind(N)" — a shape no parser accepts,
// because a diagnostic rendering must never round-trip into a value.
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
// sum type without being an interface — content_part.go's partPayload
// reasoning, restated for the event envelope.
//
// It is unexported and its methods are unexported, so no type outside this
// package can be one.
type eventPayload interface {
	// kind reports the registered kind this payload is.
	kind() EventKind

	// validate reports the payload's first broken rule at the given
	// position, or nil.
	validate(at Path) *Violation
}

// Event is Layer 1's envelope for one item on a stream: a kind, its typed
// payload, and the sequence position it was stamped with (V-STR-10).
//
// Like [Part], it is a value with one unexported field, so reading is methods
// and constructing is a constructor — Go forbids naming an unexported field
// in a composite literal from another package. The zero value is reachable
// and carries no payload, therefore no kind, therefore no membership of the
// closed kind vocabulary; [CheckEmit] rejects it.
type Event struct {
	payload eventPayload
	seq     Sequence
}

// Kind reports which payload this event carries.
//
// It is derived from the payload on every call and is never stored, so the
// answer cannot drift from the payload it describes. An event that was never
// constructed reports the zero kind, which is not a member of the vocabulary
// (R-AEE-001).
func (e Event) Kind() EventKind {
	if e.payload == nil {
		return 0
	}
	return e.payload.kind()
}

// Sequence reports the position this event was stamped with, or the zero
// [Sequence] — the documented "never stamped" sentinel — if it never was
// (R-AEE-010).
func (e Event) Sequence() Sequence { return e.seq }

// CheckEmit is the producer's emission boundary (R-AEE-010, design.md D4):
// the point every event is offered to before it may be treated as part of a
// stream.
//
// The rules are checked in the order written, per V-FAIL-04 (design.md D5):
//
//  1. the event carries a payload registered in the kind table, else
//     [ErrNotInVocabulary] at "event" — a value that skipped construction, or
//     one whose payload type this package never registered, has no kind and
//     is not a member of the closed vocabulary (R-AEE-002);
//  2. the sequence is not the unstamped sentinel 0, else [ErrOutOfRange] at
//     "event", "sequence" — an event offered here must already have passed
//     through a [Stamper] (R-AEE-010).
//
// A later step of this same milestone adds rule 3, a block-scoped kind
// carries a valid block index (R-AEE-014), and rule 4, the payload satisfies
// its own rules.
func CheckEmit(e Event) error {
	return FirstFailure(
		func() *Violation {
			if _, ok := eventRegistryEntry(e.Kind()); !ok {
				return Invalid(ErrNotInVocabulary, At("event"))
			}
			return nil
		},
		func() *Violation {
			if e.Sequence() == 0 {
				return Invalid(ErrOutOfRange, At("event"), At("sequence"))
			}
			return nil
		},
	)
}
