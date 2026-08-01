// AI-14 — the test-only witness bridge (design.md D6).
//
// AI-14 registers zero production event kinds (R-AEE-006): response
// start/completion, content blocks and the terminal error all belong to
// AI-15 … AI-19, none of which has landed. Everything this milestone's own
// spec requires of a *registered* kind — a derivable kind, a sealed payload,
// external readability with no type switch over an unexported type, an
// exhaustive registry, a payload-independent descriptor — therefore has to be
// proved against a kind this file registers for tests alone.
//
// # Why this is package ai, and not ai_test
//
// [WitnessPayload] must implement the unexported eventPayload and
// blockPayload interfaces, and [NewWitnessEvent] must build an [Event] by
// composite literal — both illegal from another package, by the sealing
// R-AEE-003 exists to prove. A _test.go file in package ai is the one place
// both are legal and the file still never reaches a non-test build
// (S-AEE-013, S-AEE-017): a package's own _test.go files are excluded from
// every ordinary `go build`.
//
// # Fixed kind vs. dynamic registration
//
// [KindTestWitness] is one fixed kind, registered once below, for tests that
// only need *a* registered kind — the AI-14.1 exhaustiveness guard
// (event_registry_test.go) and the AI-14.2 sequence tests. AI-14.4's checker
// tests (cardinality, terminal, block ordering) each need a *different*
// descriptor, so [RegisterTestKind] grows the registry dynamically instead:
// append, with a cleanup that truncates it back. Because eventRegistry is one
// shared package-level slice, a test holding a registration MUST NOT use
// t.Parallel — a concurrent truncation from another test would corrupt it.
package ai

// KindTestWitness is the one event kind this milestone's tests may construct
// through the exported surface.
//
// It borrows the value the first real production kind would take
// (eventKindEnd) without moving that bound, so [EventKinds] stays empty for a
// non-test build (S-AEE-013, S-AEE-017) even though this file registers it.
const KindTestWitness EventKind = eventKindEnd

// WitnessPayload is a test-only payload proving eventPayload's three
// properties — a derivable kind, external readability, and a validation path
// — and blockPayload's block-index readability, without registering a real
// production kind (R-AEE-004).
type WitnessPayload struct {
	k     EventKind
	block BlockIndex
}

// kind reports this payload's registered kind.
func (w WitnessPayload) kind() EventKind { return w.k }

// blockIndex reports the block this payload's event belongs to, satisfying
// the unexported blockPayload interface the checker asserts to.
func (w WitnessPayload) blockIndex() BlockIndex { return w.block }

// BlockIndex reports the block this payload's event belongs to. It is the
// exported counterpart of blockIndex, reachable from ai_test — R-AEE-005's
// proof that a consumer reads a typed payload through an exported accessor,
// not a type switch over an unexported type.
func (w WitnessPayload) BlockIndex() BlockIndex { return w.block }

// validate reports the payload's first broken rule, or nil. The witness
// carries no rule of its own at this milestone's envelope-skeleton stage; a
// rule is added once a test needs one to fail (see sequence_test.go's
// R-AEE-010 coverage). The position parameter is unused for that reason —
// named _ rather than removed, since eventPayload's signature requires it.
func (w WitnessPayload) validate(_ Path) *Violation { return nil }

// init registers KindTestWitness once, at package-test-load time, before any
// test can observe an incomplete registry. It is the only writer of this
// specific entry; RegisterTestKind's dynamic entries always land after it.
func init() {
	if EventKind(len(eventRegistry)) != KindTestWitness {
		panic("export_test.go: eventRegistry is not aligned with KindTestWitness at init")
	}
	eventRegistry = append(eventRegistry, eventRegistration{
		name:       "test_witness",
		descriptor: EventDescriptor{Role: BlockRoleNone, Cardinality: CardinalityAny, Terminal: false},
	})
}

// NewWitnessEvent constructs a valid event of kind [KindTestWitness] carrying
// block — R-AEE-004's leg 1 (a constructor).
func NewWitnessEvent(block BlockIndex) Event {
	return Event{payload: WitnessPayload{k: KindTestWitness, block: block}}
}

// WitnessPayload reports e's payload as a [WitnessPayload], and whether e
// carries one — R-AEE-004's leg 2 (an accessor), and R-AEE-005's proof that
// an external package reads a typed payload through an exported accessor
// with no type switch over an unexported type.
func (e Event) WitnessPayload() (WitnessPayload, bool) {
	w, ok := e.payload.(WitnessPayload)
	return w, ok
}

// RegisterTestKind appends a new test-only kind with descriptor d, returning
// the kind and a cleanup that truncates the registry back to its state before
// the call.
//
// It is what gives AI-14.4's checker tests arbitrary descriptors — cardinality
// at-most-one, terminal, any block role — without registering a production
// kind. Callers MUST call the returned cleanup (typically via t.Cleanup)
// before another test registers a kind of its own, and MUST NOT run in
// parallel with one that does: see the file comment.
func RegisterTestKind(name string, d EventDescriptor) (EventKind, func()) {
	k := EventKind(len(eventRegistry))
	eventRegistry = append(eventRegistry, eventRegistration{name: name, descriptor: d})
	return k, func() { eventRegistry = eventRegistry[:k] }
}

// NewTestEvent constructs an event of a dynamically registered test kind k,
// carrying block. k is ordinarily obtained from [RegisterTestKind]; any other
// value builds an event [CheckEmit] rejects as unregistered, which is itself
// a legitimate test input for the rejection path.
func NewTestEvent(k EventKind, block BlockIndex) Event {
	return Event{payload: WitnessPayload{k: k, block: block}}
}

// AllTestEventKinds returns the production vocabulary ([EventKinds], empty at
// this milestone) plus every currently registered test kind: [KindTestWitness]
// and whatever a test has registered through [RegisterTestKind] and not yet
// cleaned up.
//
// design.md D6 names this symbol "TestEventKinds"; go vet's tests analyzer
// (run by both `go test` and `make lint`) requires any top-level identifier
// starting with "Test" in a _test.go file to have the exact signature
// func(t *testing.T) — a Go-tooling constraint the design predates. Renamed
// here to the nearest non-colliding name; the semantics D6 describes are
// unchanged.
func AllTestEventKinds() []EventKind {
	out := EventKinds()
	for k := eventKindEnd; int(k) < len(eventRegistry); k++ {
		out = append(out, k)
	}
	return out
}

// DescriptorOf reports k's registered [EventDescriptor], and whether k is
// registered at all — R-AEE-014's own bridge, additive to design.md D6's
// named surface: AI-14.1..AI-14.3 never needed to inspect a descriptor's
// fields directly, only its effects through CheckEmit; AI-14.4's checker
// tests do.
func DescriptorOf(k EventKind) (EventDescriptor, bool) {
	entry, ok := eventRegistryEntry(k)
	if !ok {
		return EventDescriptor{}, false
	}
	return entry.descriptor, true
}
