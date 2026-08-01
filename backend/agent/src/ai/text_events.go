// AI-16 — the text-block event family: start, delta, end.
//
// This file implements V-STR-14 narrowed to text, V-STR-15 block index and
// V-STR-16 delta/fragment: the three events a streamed text block produces.
// It follows response_start.go's and completion.go's layout — an opaque
// payload type with unexported fields, a package-owned constructor as the
// only door in, a typed accessor on the event — extended with AI-14's
// unexported blockIndex() method so CheckStream/CheckEmit can read the block
// index through the shared blockPayload interface, with no type switch over
// a concrete payload type (R-AEE-015).
//
// # One family file, not three
//
// TextBlockStart, TextDelta and TextBlockEnd live together here rather than
// in three files, one per kind: design.md's A5 records this as a deliberate
// divergence — the three kinds are one lifecycle sharing one block-index
// rule and one family GoDoc, and the spec's own scenarios (interleave,
// reconstruct, zero-delta) are cross-kind.
//
// # No accumulator, ever
//
// A delta carries only the bytes new since the previous delta of its block
// (R-ATE-005). This file exports no accumulator, transcript rebuilder or
// reducer of a block's deltas — doc 0001 § 4.3 invariant 1 reserves that for
// the consumer (R-ATE-011); text_events_test.go proves byte-exactness with a
// test-local concatenator instead.

package ai

// TextBlockStart opens a text block at its producer-stamped index (V-STR-14
// narrowed to text).
type TextBlockStart struct{ block BlockIndex }

// kind reports this payload's registered kind.
func (TextBlockStart) kind() EventKind { return EventKindTextBlockStart }

// validate reports the payload's first broken rule at the given position, or
// nil: the block index is at least 1, else [ErrOutOfRange] at "block"
// (R-ATE-003).
func (s TextBlockStart) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if s.block == 0 {
				return Invalid(ErrOutOfRange, under(at, At("block"))...)
			}
			return nil
		},
	)
}

// NewTextBlockStart constructs the event opening a text block at block
// (R-ATE-002, R-ATE-003).
//
// It is the only door in. The rule is [TextBlockStart.validate]'s, run from
// here and again at [CheckEmit] (R-AEE-010). On failure the zero [Event] is
// returned, so a caller that ignored the error cannot mistake the result for
// a constructed event. The returned event is unstamped (sequence 0);
// stamping is a [Stamper]'s job, per-stream, at the producer boundary.
func NewTextBlockStart(block BlockIndex) (Event, error) {
	payload := TextBlockStart{block: block}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// TextBlockStart returns the event's text-block-start payload, and whether
// the event carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [TextBlockStart] and false, and never
// panics.
func (e Event) TextBlockStart() (TextBlockStart, bool) {
	payload, ok := e.payload.(TextBlockStart)
	if !ok {
		return TextBlockStart{}, false
	}
	return payload, true
}

// Block returns the index of the text block this event opens (V-STR-15).
func (s TextBlockStart) Block() BlockIndex { return s.block }

// blockIndex reports the block this payload's event belongs to, satisfying
// AI-14's unexported blockPayload interface.
func (s TextBlockStart) blockIndex() BlockIndex { return s.block }

// String renders the text-block-start event for a diagnostic reader, naming
// the type and nothing it carries — [ResponseStart.String]'s posture,
// restated (V-FAIL-13). Not the block index: a block index is caller-derived
// context, and this package's diagnostic rendering never reproduces it.
func (s TextBlockStart) String() string { return "text_block_start" }

// GoString renders the text-block-start event for the %#v verb.
//
// Without it, %#v falls back to reflection and prints the unexported field,
// which would make the redaction posture a property of which verb someone
// reached for rather than a property of the type.
func (s TextBlockStart) GoString() string { return s.String() }

// TextDelta carries the new bytes of a text block's content since the
// block's previous delta — never the block's accumulated content (V-STR-16,
// R-ATE-005).
type TextDelta struct {
	block BlockIndex
	// delta is the fragment's raw bytes. It is a string of bytes, not a
	// string of well-formed UTF-8: an individual fragment may split a
	// multi-byte rune, and this type never validates, repairs or re-encodes
	// it (R-ATE-007).
	delta string
}

// kind reports this payload's registered kind.
func (TextDelta) kind() EventKind { return EventKindTextDelta }

// validate reports the payload's first broken rule at the given position, or
// nil. The rules are checked in the order written, per V-FAIL-04: the block
// index is at least 1, else [ErrOutOfRange] at "block" (R-ATE-003); then the
// fragment is at most [MaxTextLen] bytes, else [ErrOutOfRange] at "delta"
// (R-ATE-009) — the fragment cap is the only bound; the block's reconstructed
// total is never bounded here (R-ATE-011 forbids accumulating it at Layer 1).
// No emptiness rule applies to the fragment at any position: a zero-length or
// whitespace-only fragment is legal (R-ATE-008).
func (d TextDelta) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if d.block == 0 {
				return Invalid(ErrOutOfRange, under(at, At("block"))...)
			}
			return nil
		},
		func() *Violation {
			if len(d.delta) > MaxTextLen {
				return Invalid(ErrOutOfRange, under(at, At("delta"))...)
			}
			return nil
		},
	)
}

// NewTextDelta constructs one text block's delta event, carrying the
// fragment new since the block's previous delta (R-ATE-002, R-ATE-005).
//
// It is the only door in. The rules are [TextDelta.validate]'s, run from
// here and again at [CheckEmit] (R-AEE-010). On failure the zero [Event] is
// returned, so a caller that ignored the error cannot mistake the result for
// a constructed event. The returned event is unstamped (sequence 0);
// stamping is a [Stamper]'s job, per-stream, at the producer boundary.
//
// delta is stored byte for byte: nothing here validates its encoding, trims
// it, or rejects it for being empty or whitespace-only (R-ATE-007,
// R-ATE-008). It is rejected only for exceeding [MaxTextLen] (R-ATE-009) —
// the same documented ceiling [NewText] applies to a complete text part, so a
// fragment of a legal complete text is never rejected here.
func NewTextDelta(block BlockIndex, delta string) (Event, error) {
	payload := TextDelta{block: block, delta: delta}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// TextDelta returns the event's text-delta payload, and whether the event
// carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [TextDelta] and false, and never panics.
func (e Event) TextDelta() (TextDelta, bool) {
	payload, ok := e.payload.(TextDelta)
	if !ok {
		return TextDelta{}, false
	}
	return payload, true
}

// Block returns the index of the text block this delta belongs to
// (V-STR-15).
func (d TextDelta) Block() BlockIndex { return d.block }

// Delta returns the fragment's raw bytes — the bytes new since this block's
// previous delta, never the block's accumulated content (V-STR-16,
// R-ATE-005).
//
// The result is byte-equal to what [NewTextDelta] was given: nothing on this
// path decodes, validates, trims or re-encodes it. A fragment may be invalid
// UTF-8 on its own when a multi-byte rune splits across a delta boundary
// (R-ATE-007).
func (d TextDelta) Delta() string { return d.delta }

// blockIndex reports the block this payload's event belongs to, satisfying
// AI-14's unexported blockPayload interface.
func (d TextDelta) blockIndex() BlockIndex { return d.block }

// String renders the text-delta event for a diagnostic reader, naming the
// type and nothing it carries — not the fragment and not its length: a
// length is caller-derived and a prefix of a secret is still a secret
// (V-FAIL-13).
func (d TextDelta) String() string { return "text_delta" }

// GoString renders the text-delta event for the %#v verb.
func (d TextDelta) GoString() string { return d.String() }

// TextBlockEnd closes a text block opened by a matching [TextBlockStart]
// (V-STR-14 narrowed to text).
type TextBlockEnd struct{ block BlockIndex }

// kind reports this payload's registered kind.
func (TextBlockEnd) kind() EventKind { return EventKindTextBlockEnd }

// validate reports the payload's first broken rule at the given position, or
// nil: the block index is at least 1, else [ErrOutOfRange] at "block"
// (R-ATE-003).
func (b TextBlockEnd) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if b.block == 0 {
				return Invalid(ErrOutOfRange, under(at, At("block"))...)
			}
			return nil
		},
	)
}

// NewTextBlockEnd constructs the event closing the text block at block
// (R-ATE-002, R-ATE-003). A block closed with zero deltas between its start
// and end is legal (R-ATE-010): this constructor enforces nothing about
// whether any delta was ever seen, because a single event has no way to know
// that.
//
// It is the only door in. The rule is [TextBlockEnd.validate]'s, run from
// here and again at [CheckEmit] (R-AEE-010). On failure the zero [Event] is
// returned, so a caller that ignored the error cannot mistake the result for
// a constructed event. The returned event is unstamped (sequence 0);
// stamping is a [Stamper]'s job, per-stream, at the producer boundary.
func NewTextBlockEnd(block BlockIndex) (Event, error) {
	payload := TextBlockEnd{block: block}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// TextBlockEnd returns the event's text-block-end payload, and whether the
// event carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [TextBlockEnd] and false, and never
// panics.
func (e Event) TextBlockEnd() (TextBlockEnd, bool) {
	payload, ok := e.payload.(TextBlockEnd)
	if !ok {
		return TextBlockEnd{}, false
	}
	return payload, true
}

// Block returns the index of the text block this event closes (V-STR-15).
func (b TextBlockEnd) Block() BlockIndex { return b.block }

// blockIndex reports the block this payload's event belongs to, satisfying
// AI-14's unexported blockPayload interface.
func (b TextBlockEnd) blockIndex() BlockIndex { return b.block }

// String renders the text-block-end event for a diagnostic reader, naming
// the type and nothing it carries.
func (b TextBlockEnd) String() string { return "text_block_end" }

// GoString renders the text-block-end event for the %#v verb.
func (b TextBlockEnd) GoString() string { return b.String() }
