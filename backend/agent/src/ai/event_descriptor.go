// AI-14 foundation — the ordering descriptor, shared by every registered
// event kind.
//
// This file implements the payload-independent half of R-AEE-014: the shape
// a descriptor takes, before any kind exists to carry one. Decision D1
// (design.md) fixes what the AI-14.4 checker is allowed to read — a kind, a
// descriptor, a block index and a sequence, never a concrete payload type —
// and this is where that shape is declared, so the checker (stream_check.go)
// never has to import knowledge of AI-15's response events, AI-16 … AI-18's
// content blocks, or AI-19's terminal error to enforce ordering on them.
//
// # Adding a kind: the six-step procedure
//
// AI-06.4's five-step procedure for a content-part kind gains a sixth step
// here, because a registered event kind carries one more thing than a content
// part does: an ordering descriptor.
//
//  1. declare the constant at the end of the EventKind block and move
//     eventKindEnd past it;
//  2. add an unexported payload type with kind() and validate() — and
//     blockIndex() too, when the kind's descriptor Role is not
//     [BlockRoleNone];
//  3. add the exported constructor New<Kind>, whose rules are the payload's;
//  4. add the exported accessor func (e Event) <Kind>() (T, bool);
//  5. add the registry line in eventRegistry, pairing the kind's name with
//     its [EventDescriptor];
//  6. add the name to the documented list on [EventKind].
//
// A kind registered without a descriptor entry fails the AI-14.1 exhaustiveness
// assertion (R-AEE-014's own scenario, event_registry_test.go) exactly as a
// missing witness leg does — the registry is one table, so a kind structurally
// cannot register without one.
//
// # Registering a delta kind: fragment-only, never a snapshot
//
// A kind whose descriptor Role is [BlockRoleDelta] carries a block index and
// only the new fragment of that block's content (V-STR-16, R-AEE-020) — step
// 2's payload holds a fragment, not an accumulated transcript, and step 3's
// constructor takes a fragment, not the block's content so far. This
// capability exports no accumulator, transcript rebuilder or reducer of a
// block's deltas: doc 0001 § 4.3 invariant 1 reserves reconstructing a block
// from its fragments for the consumer, never this package. A registered
// delta kind whose payload instead carries or exposes a running snapshot
// violates this the moment it lands, regardless of how its descriptor reads.

package ai

// BlockRole is a registered kind's position in a block's lifecycle (V-STR-14).
//
// The zero value, [BlockRoleNone], means the kind is not part of a block at
// all — deliberately the least restrictive reading, so a kind registered
// without stating a role does not silently join block ordering it never asked
// to be part of.
type BlockRole uint8

const (
	// BlockRoleNone is the zero value: the kind carries no block index and
	// the AI-14.4 checker's block-ordering rule (R-AEE-016) does not apply to
	// it.
	BlockRoleNone BlockRole = iota

	// BlockRoleStart opens a block at its [BlockIndex]. Every delta and the
	// end for that index must follow it (R-AEE-016).
	BlockRoleStart

	// BlockRoleDelta carries one fragment of a block's content, and only the
	// new fragment — never a snapshot of what came before (V-STR-16,
	// R-AEE-020).
	BlockRoleDelta

	// BlockRoleEnd closes a block at its [BlockIndex]. A block with no
	// matching end is reported as unterminated, distinguishably from an
	// ordering violation (R-AEE-016).
	BlockRoleEnd
)

// Cardinality bounds how many events of a registered kind MAY appear on one
// stream.
//
// The zero value, [CardinalityAny], is the least restrictive reading:
// unbounded repetition, including zero, unless a kind's registration states
// otherwise.
type Cardinality uint8

const (
	// CardinalityAny is the zero value: the kind may appear any number of
	// times on one stream, including not at all.
	CardinalityAny Cardinality = iota

	// CardinalityAtMostOne restricts a kind to at most one event per stream
	// (R-AEE-017). A second event of the kind is reported by the checker,
	// naming the kind and the second event's sequence.
	CardinalityAtMostOne
)

// BlockIndex names one block within a stream (V-STR-15).
//
// It is 1-based; 0 is the unset value, rejected wherever a block-scoped event
// is checked (R-AEE-014) — consistent with AI-16's R-ATE-003, so a stream
// mixing AI-14's own block-scoped kinds with AI-16's text blocks reads one
// indexing convention rather than two.
type BlockIndex uint64

// blockPayload is the shared, unexported interface a block-scoped payload
// implements so its block index is readable without the checker naming a
// concrete payload type (R-AEE-015).
//
// It is deliberately a second, narrower interface rather than a method folded
// into [eventPayload]: a kind whose descriptor is [BlockRoleNone] never needs
// to answer this question, and forcing every payload to implement it would
// give every kind a block index whether or not it has one. The checker
// type-asserts to this interface — never to a concrete payload type — which
// is what keeps R-AEE-015 true as new block-scoped kinds are added.
type blockPayload interface {
	// blockIndex reports which block this payload's event belongs to. It is
	// called only for a kind whose descriptor Role is not [BlockRoleNone].
	blockIndex() BlockIndex
}

// EventDescriptor is the payload-independent ordering metadata a registered
// kind declares (R-AEE-014): its position in a block's lifecycle, how many
// times it may appear on one stream, and whether it ends the stream.
//
// The AI-14.4 checker (stream_check.go) reads only this shape plus a kind, a
// block index and a sequence — never a concrete payload type — so a later
// milestone extends the checker by registering a descriptor, with no change
// to the checker's own source (R-AEE-015). "At most one" and "terminal" are
// reusable, descriptor-driven primitives for exactly this reason: AI-15
// registers response-start as at-most-one and completion as terminal, and
// AI-19 registers its error as terminal, none of it touching AI-14.
type EventDescriptor struct {
	// Role is the kind's position in a block's lifecycle, or [BlockRoleNone]
	// when the kind is not block-scoped.
	Role BlockRole

	// Cardinality bounds how many events of this kind one stream may carry.
	Cardinality Cardinality

	// Terminal reports whether this kind ends the stream (V-STR-18): the
	// checker reports a violation for any event that follows one, and for a
	// second terminal event on the same stream.
	Terminal bool
}
