// AI-18 — the streamed tool-call event family: call-start, call-delta,
// call-end.
//
// AI-09's [ToolCall] describes a tool call that has already finished
// arriving. This file describes one arriving: three separately registered
// event kinds on AI-14's envelope, following event_descriptor.go's six-step
// procedure once per kind, in the same layout response_start.go and
// completion.go already established — an opaque value type with unexported
// fields, a package-owned constructor as the only door in, and a typed
// accessor on the event.
//
// # Construction is validated eagerly, same as ToolCall and the AI-15 events
//
// AI-14's own design.md sketches a two-phase, never-erroring constructor.
// AI-15's landed code (response_start.go, completion.go) does not take that
// shape: NewResponseStart and NewCompletion validate at construction and
// return (Event, error), exactly like tool_call.go's NewToolCall — and this
// spec's own scenario text is written the same way ("when construction
// runs, then it fails" — S-ATC-005, S-ATC-006, S-ATC-007, S-ATC-011). The
// three constructors below follow the landed precedent, not the sketch: they
// validate via [Violation]-returning validate(at Path) and return
// (Event, error). CheckEmit still re-runs the same validate() a second time
// at the true stream emission boundary, after a [Stamper] assigns a
// sequence — the "two entry points, one rule set" posture content_part.go's
// partPayload documents, restated for the event envelope.
//
// # Block index is a shared, generic concern — not a per-payload one
//
// All three kinds are block-scoped ([BlockRoleStart], [BlockRoleDelta],
// [BlockRoleEnd]), so each implements the unexported blockPayload interface.
// [CheckEmit]'s own rule 3 already rejects a block index of 0 generically,
// for any block-scoped kind, before it ever reaches a payload's validate().
// Rule 1 below (blockIndex >= 1) is this milestone's own construction-time
// enforcement of R-ATC-004 — the rule CheckEmit's rule 3 restates a second
// time, at the later boundary, for a hand-built event that skipped this
// constructor.
//
// # Fragment-only deltas, byte-exact ends, no re-marshalling
//
// A fragment and a call's complete argument bytes are both stored as
// unexported strings — comparable, immutable, and the reason every byte
// accessor below returns a fresh []byte copy on each call (converting a Go
// string to []byte always allocates), mirroring tool_call.go's
// Arguments() exactly (R-ATC-006). Neither NewToolCallDelta nor
// NewToolCallEnd parses, validates or rewrites what it is given: no call to
// a JSON well-formedness check, no {} canonicalization of empty arguments —
// both belong to AI-30's reassembly through NewToolCall, once, at the single
// point this package validates a tool call's arguments (R-ATC-007, "one rule
// set, two entry points, no third").
//
// # Delta optionality
//
// A call-start immediately followed by its call-end, with zero deltas
// between them, is legal and complete (R-ATC-009). No consumer contract in
// this file requires at least one delta, and nothing here can observe how
// many a caller happened to send: [ToolCallDelta] exposes no counter, and
// [ToolCallEnd] carries no signal distinguishing a whole delivery from a
// fragmented one (R-ATC-010).
//
// # The call ordinal is not here
//
// Restating R-ATM-006 rather than re-defining it: none of the three structs
// below carries a field naming a call's position among a response's calls.
// The ordinal (V-STR-21) is derived by filtering a recorded stream's
// call-start events in order — test-local code, the stream-side
// continuation of ToolCalls() — never an accumulator this package ships
// (R-ATC-012).

package ai

// ToolCallStart is the event announcing a streamed tool call: the call's
// identity and the tool's name, both readable from an external package
// before any argument byte has arrived (R-ATC-002).
type ToolCallStart struct {
	block    BlockIndex
	id, name string
}

// kind reports this payload's registered kind.
func (ToolCallStart) kind() EventKind { return EventKindToolCallStart }

// blockIndex reports the block this payload's event belongs to, satisfying
// the unexported blockPayload interface CheckEmit and CheckStream read
// through (R-AEE-015) — never a concrete payload type.
func (s ToolCallStart) blockIndex() BlockIndex { return s.block }

// validate reports the payload's first broken rule at the given position, or
// nil. The rules are checked in the order written, per V-FAIL-04: the block
// index is at least 1, else [ErrOutOfRange] at "block_index"
// (R-ATC-004 = R-ATE-003 adopted); then the identity is not empty, else
// [ErrEmpty] at "id"; then the tool name is not empty, else [ErrEmpty] at
// "name" (R-ATC-002, D3).
func (s ToolCallStart) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if s.block == 0 {
				return Invalid(ErrOutOfRange, under(at, At("block_index"))...)
			}
			return nil
		},
		func() *Violation {
			if s.id == "" {
				return Invalid(ErrEmpty, under(at, At("id"))...)
			}
			return nil
		},
		func() *Violation {
			if s.name == "" {
				return Invalid(ErrEmpty, under(at, At("name"))...)
			}
			return nil
		},
	)
}

// NewToolCallStart constructs a call-start event (R-ATC-002).
//
// It is the only door in. The rules are [ToolCallStart.validate]'s, run from
// here — this package's producer boundary — and again at [CheckEmit], the
// emission boundary every event must pass before it may be treated as part
// of a stream (R-AEE-010). On failure the zero [Event] is returned, so a
// caller that ignores the error cannot mistake the result for a constructed
// event. The returned event is unstamped (sequence 0); stamping is a
// [Stamper]'s job, per-stream, at the producer boundary.
func NewToolCallStart(block BlockIndex, id, name string) (Event, error) {
	payload := ToolCallStart{block: block, id: id, name: name}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// ToolCallStart returns the event's call-start payload, and whether the
// event carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [ToolCallStart] and false, and never
// panics.
func (e Event) ToolCallStart() (ToolCallStart, bool) {
	payload, ok := e.payload.(ToolCallStart)
	if !ok {
		return ToolCallStart{}, false
	}
	return payload, true
}

// BlockIndex returns the shared, stream-wide index of the block this call
// occupies (R-ATC-003).
func (s ToolCallStart) BlockIndex() BlockIndex { return s.block }

// ID returns the call's identity, byte-exact.
func (s ToolCallStart) ID() string { return s.id }

// Name returns the tool's name, byte-exact and readable before any argument
// byte has arrived.
func (s ToolCallStart) Name() string { return s.name }

// String renders the call-start for a diagnostic reader, naming the type and
// nothing it carries — [ToolCall.String]'s posture, restated (V-FAIL-13).
// Not the identity, not the name, and not their length: a length is
// caller-derived and a prefix of a secret is still a secret.
func (s ToolCallStart) String() string { return "toolcallstart" }

// GoString renders the call-start for the %#v verb.
//
// Without it, %#v falls back to reflection and prints the unexported
// fields, which would make the redaction posture a property of which verb
// someone reached for rather than a property of the type.
func (s ToolCallStart) GoString() string { return s.String() }

// ToolCallDelta carries one new fragment of an open tool call's argument
// bytes — never an accumulated transcript (V-STR-16, R-ATC-005).
type ToolCallDelta struct {
	block    BlockIndex
	fragment string
}

// kind reports this payload's registered kind.
func (ToolCallDelta) kind() EventKind { return EventKindToolCallDelta }

// blockIndex reports the block this payload's event belongs to (R-AEE-015).
func (d ToolCallDelta) blockIndex() BlockIndex { return d.block }

// validate reports the payload's first broken rule at the given position, or
// nil. The only rule a delta carries is the block index (R-ATC-004): the
// fragment is raw bytes and is never itself validated — zero-length,
// whitespace-only and invalid-UTF-8 fragments are all accepted unaltered
// (R-ATC-005).
func (d ToolCallDelta) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if d.block == 0 {
				return Invalid(ErrOutOfRange, under(at, At("block_index"))...)
			}
			return nil
		},
	)
}

// NewToolCallDelta constructs a call-delta event (R-ATC-005).
//
// fragment is stored as supplied — never trimmed, validated or normalized.
// It is the only door in; the rules are [ToolCallDelta.validate]'s, run from
// here and again at [CheckEmit] (R-AEE-010). On failure the zero [Event] is
// returned. The returned event is unstamped (sequence 0).
func NewToolCallDelta(block BlockIndex, fragment []byte) (Event, error) {
	payload := ToolCallDelta{block: block, fragment: string(fragment)}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// ToolCallDelta returns the event's call-delta payload, and whether the
// event carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [ToolCallDelta] and false, and never
// panics.
func (e Event) ToolCallDelta() (ToolCallDelta, bool) {
	payload, ok := e.payload.(ToolCallDelta)
	if !ok {
		return ToolCallDelta{}, false
	}
	return payload, true
}

// BlockIndex returns the shared, stream-wide index of the block this delta
// belongs to (R-ATC-003).
func (d ToolCallDelta) BlockIndex() BlockIndex { return d.block }

// Fragment returns the new argument bytes this delta carries — never a
// snapshot of what came before.
//
// The result is a fresh slice on every call (converting the stored string
// back to bytes always allocates), so a caller that mutates what it received
// cannot affect the event, and two reads are never able to observe each
// other (R-ATC-006's byte-exactness discipline, restated for a fragment).
func (d ToolCallDelta) Fragment() []byte { return []byte(d.fragment) }

// String renders the call-delta for a diagnostic reader, naming the type and
// nothing it carries (V-FAIL-13). Not the fragment, and not its length: a
// length is caller-derived and a prefix of a secret is still a secret.
func (d ToolCallDelta) String() string { return "toolcalldelta" }

// GoString renders the call-delta for the %#v verb.
//
// Without it, %#v falls back to reflection and prints the unexported
// fragment field, which would make the redaction posture a property of
// which verb someone reached for rather than a property of the type.
func (d ToolCallDelta) GoString() string { return d.String() }

// ToolCallEnd closes a tool-call block, carrying its complete argument
// bytes — byte-for-byte equal to the ordered concatenation of that call's
// delta fragments (R-ATC-006).
type ToolCallEnd struct {
	block     BlockIndex
	arguments string
}

// kind reports this payload's registered kind.
func (ToolCallEnd) kind() EventKind { return EventKindToolCallEnd }

// blockIndex reports the block this payload's event belongs to (R-AEE-015).
func (c ToolCallEnd) blockIndex() BlockIndex { return c.block }

// validate reports the payload's first broken rule at the given position, or
// nil. The only rule a call-end carries is the block index (R-ATC-004): its
// argument bytes are never parsed, validated for JSON well-formedness, or
// canonicalized when empty — that is AI-30's, once, at reassembly through
// NewToolCall (R-ATC-007).
func (c ToolCallEnd) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if c.block == 0 {
				return Invalid(ErrOutOfRange, under(at, At("block_index"))...)
			}
			return nil
		},
	)
}

// NewToolCallEnd constructs a call-end event (R-ATC-006, R-ATC-007).
//
// arguments is stored exactly as supplied: no re-marshalling, no key
// reordering, no whitespace normalization, no well-formedness check, and —
// unlike NewToolCall — no substitution of "{}" for zero-length arguments.
// It is the only door in; the rules are [ToolCallEnd.validate]'s, run from
// here and again at [CheckEmit] (R-AEE-010). On failure the zero [Event] is
// returned. The returned event is unstamped (sequence 0).
func NewToolCallEnd(block BlockIndex, arguments []byte) (Event, error) {
	payload := ToolCallEnd{block: block, arguments: string(arguments)}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// ToolCallEnd returns the event's call-end payload, and whether the event
// carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [ToolCallEnd] and false, and never
// panics.
func (e Event) ToolCallEnd() (ToolCallEnd, bool) {
	payload, ok := e.payload.(ToolCallEnd)
	if !ok {
		return ToolCallEnd{}, false
	}
	return payload, true
}

// BlockIndex returns the shared, stream-wide index of the block this call
// occupies (R-ATC-003).
func (c ToolCallEnd) BlockIndex() BlockIndex { return c.block }

// Arguments returns the call's complete argument bytes, byte-for-byte equal
// to the ordered concatenation of the call's delta fragments (R-ATC-006).
//
// The result is a fresh slice on every call, so a caller that mutates what
// it received cannot affect the event, and two reads are never able to
// observe each other.
func (c ToolCallEnd) Arguments() []byte { return []byte(c.arguments) }

// String renders the call-end for a diagnostic reader, naming the type and
// nothing it carries (V-FAIL-13). Not the argument bytes, and not their
// length: a length is caller-derived and a prefix of a secret is still a
// secret.
func (c ToolCallEnd) String() string { return "toolcallend" }

// GoString renders the call-end for the %#v verb.
//
// Without it, %#v falls back to reflection and prints the unexported
// arguments field, which would make the redaction posture a property of
// which verb someone reached for rather than a property of the type.
func (c ToolCallEnd) GoString() string { return c.String() }
