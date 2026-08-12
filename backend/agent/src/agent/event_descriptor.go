// AG-04.1 — the ordering descriptor, shared by every registered event kind
// (design AD-1).
//
// This file declares the payload-independent shape the validator
// (stream_check.go) is allowed to read: a kind's position in the run/turn
// bracket structure, where it may appear, how many times it may appear, and
// whether it ends the stream. The validator never reads a concrete payload
// type and never special-cases a kind by name — a later milestone that
// registers a kind with a descriptor extends every rule the validator
// checks with no change to the validator's own source, mirroring Layer 1's
// event_descriptor.go (`backend/agent/src/ai/event_descriptor.go:120-130`).
//
// # Adding a kind: the six-step procedure
//
// AG-05 and AG-06 register their own kinds by following these six steps,
// with no edit to the validator's rule engine or to the every-kind-
// constructible guard's own logic:
//
//  1. declare the constant at the end of the EventKind block and move the
//     end bound past it;
//  2. add an unexported payload type implementing kind() and validate();
//  3. add the exported constructor New<Kind>, whose rules are the
//     payload's and the envelope identity's;
//  4. add the exported accessor func (e Event) <Kind>() (T, bool);
//  5. add the registry row in event.go pairing the kind's name with its
//     [EventDescriptor] — one table, so a kind structurally cannot register
//     without a descriptor;
//  6. add the name to [EventKind]'s documented list AND the witness entry
//     in event_registry_test.go — the guard's bidirectional cross-check
//     fails the same pull request that skips either half.
package agent

// BracketRole is a registered kind's position in the run/turn bracket
// structure the validator enforces (design AD-1).
//
// The zero value, [BracketRoleNone], means the kind does not itself open or
// close either bracket — deliberately the least restrictive reading, so a
// kind registered without stating a role does not silently join bracket
// discipline it never asked to be part of. Every kind AG-05 and AG-06
// register uses this zero value; only AG-04's own four kinds use the other
// four.
type BracketRole uint8

const (
	// BracketRoleNone is the zero value: the kind neither opens nor closes
	// a bracket. It is still subject to [Placement] and [Cardinality].
	BracketRoleNone BracketRole = iota

	// BracketRoleOpensRun opens the run bracket. At most one such event may
	// appear, and it must be the first event of a valid stream.
	BracketRoleOpensRun

	// BracketRoleClosesRun closes the run bracket. Nothing may follow it,
	// and it may not close a run that carries a still-open turn.
	BracketRoleClosesRun

	// BracketRoleOpensTurn opens a turn nested inside an open run. At most
	// one turn may be open at a time.
	BracketRoleOpensTurn

	// BracketRoleClosesTurn closes the turn opened by the matching
	// [BracketRoleOpensTurn] event.
	BracketRoleClosesTurn
)

// Placement bounds where a registered kind's events may appear relative to
// the two-level run/turn scope (design AD-1).
//
// The zero value, [PlacementRun], is the least restrictive membership AG-04
// itself needs: every kind AG-04 registers is either a bracket kind or, if
// it were not, would still only require an open run. [PlacementTurn] is
// AG-05's seam — a future kind that must appear only while a turn is open —
// unused by any kind this milestone registers.
type Placement uint8

const (
	// PlacementRun is the zero value: the kind may appear anywhere inside
	// an open run, whether or not a turn is open.
	PlacementRun Placement = iota

	// PlacementTurn restricts the kind to appear only while a turn is
	// open. AG-05's seam; no kind AG-04 registers uses it.
	PlacementTurn
)

// Cardinality bounds how many events of a registered kind MAY appear on one
// stream — mirrors `ai.Cardinality`
// (`backend/agent/src/ai/event_descriptor.go:77-94`).
//
// The zero value, [CardinalityAny], is the least restrictive reading:
// unbounded repetition, including zero, unless a kind's registration states
// otherwise. AG-06's seam; no kind AG-04 registers uses
// [CardinalityAtMostOne].
type Cardinality uint8

const (
	// CardinalityAny is the zero value: the kind may appear any number of
	// times on one stream, including not at all.
	CardinalityAny Cardinality = iota

	// CardinalityAtMostOne restricts a kind to at most one event per
	// stream. AG-06's seam; unused by any kind AG-04 registers.
	CardinalityAtMostOne
)

// EventDescriptor is the payload-independent ordering metadata a registered
// kind declares (design AD-1). The validator (stream_check.go) reads only
// this shape plus a kind and a sequence — never a concrete payload type —
// so a later milestone extends the validator by registering a descriptor,
// with no change to the validator's own source.
type EventDescriptor struct {
	// Bracket is the kind's position in the run/turn bracket structure, or
	// [BracketRoleNone] when the kind does not itself open or close either
	// bracket.
	Bracket BracketRole

	// Placement bounds where the kind may appear relative to the open
	// run/turn scope.
	Placement Placement

	// Cardinality bounds how many events of this kind one stream may
	// carry.
	Cardinality Cardinality

	// Terminal reports whether this kind ends the stream: the validator
	// reports a violation for any event that follows one.
	Terminal bool
}
