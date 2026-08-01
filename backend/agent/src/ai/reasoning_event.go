// AI-17 — the streamed reasoning block event family.
//
// This file implements AI-17.1..AI-17.3: three separately registered event
// kinds — reasoning block start, reasoning delta, reasoning block end — so a
// streamed reasoning block can never reach a consumer as assistant text
// (R-ARE-002), and AI-07's opaque round-trip token survives the event
// boundary byte-identically (R-ARE-010). It follows response_start.go and
// completion.go's layout: an opaque value type per kind with unexported
// fields, a package-owned constructor as the only door in, and a typed
// accessor on the event — plus event_descriptor.go's six-step procedure for
// a block-scoped kind.
//
// # Two doors on the start, one shape carried forward
//
// [NewReasoningBlockStart] and [NewRedactedReasoningBlockStart] are the only
// two ways to open a reasoning block (D1, reasoning_content.go's
// [NewRedactedReasoning] precedent restated on the wire side). The redaction
// bit set there is never re-decided: [NewReasoningDelta] and
// [NewReasoningBlockEnd] both take the resulting [ReasoningBlockStart] value
// rather than a bare block index, so the block's redaction bit is visible at
// construction and a delta or an end can be checked against it without a
// second, exported, cross-event validator (R-ARE-013, S-ARE-036). Each
// carries the bit forward into its own unexported field so the same rule is
// re-provable from [CheckEmit], which only ever calls a payload's own
// validate — never the constructor that built it.
//
// # No accumulator, ever
//
// A reasoning delta carries only the new fragment since the previous delta of
// its block (V-STR-16, R-ARE-005) — never accumulated text, never token
// bytes, never a redacted signal or a [ReasoningState]. This file exports no
// function that reduces a block's deltas to its complete text or its derived
// state: doc 0001 § 4.3 invariant 1 reserves that for the consumer
// (R-ARE-008). This milestone's own tests prove byte-exact reconstruction
// with a test-local concatenator instead.
package ai

// blockIndexRule returns a [Rule] rejecting a zero block index at
// "block_index" (R-ARE-003) — shared by all three payloads' validate
// methods below, extracted so the identical check is not written three times
// with room to drift between them.
func blockIndexRule(at Path, block BlockIndex) Rule {
	return func() *Violation {
		if block == 0 {
			return Invalid(ErrOutOfRange, under(at, At("block_index"))...)
		}
		return nil
	}
}

// ReasoningBlockStart opens a reasoning block (R-ARE-001): the block index
// every delta and the matching end must carry, and whether the block's
// plaintext is withheld (R-ARE-012).
type ReasoningBlockStart struct {
	block    BlockIndex
	redacted bool
}

// kind reports this payload's registered kind.
func (r ReasoningBlockStart) kind() EventKind { return EventKindReasoningBlockStart }

// blockIndex reports the block this payload's event belongs to, satisfying
// the unexported blockPayload interface event.go's CheckEmit asserts to.
func (r ReasoningBlockStart) blockIndex() BlockIndex { return r.block }

// validate reports the payload's first broken rule at the given position, or
// nil: the block index is at least 1, else [ErrOutOfRange] at "block_index"
// (R-ARE-003).
func (r ReasoningBlockStart) validate(at Path) *Violation {
	return firstViolation(blockIndexRule(at, r.block))
}

// NewReasoningBlockStart constructs the start of a reasoning block whose
// plaintext is not withheld (R-ARE-001, R-ARE-003).
//
// It is one of the two doors in (the other is [NewRedactedReasoningBlockStart]
// — D1). The rules are [ReasoningBlockStart.validate]'s, run from here and
// again at [CheckEmit] (R-AEE-010). On failure the zero [Event] is returned,
// so a caller that ignored the error cannot mistake the result for a
// constructed event. The returned event is unstamped (sequence 0); stamping
// is a [Stamper]'s job, per-stream, at the producer boundary.
func NewReasoningBlockStart(blockIndex uint64) (Event, error) {
	payload := ReasoningBlockStart{block: BlockIndex(blockIndex)}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// NewRedactedReasoningBlockStart constructs the start of a reasoning block
// whose plaintext the provider withheld (R-ARE-012, D1) —
// reasoning_content.go's [NewRedactedReasoning] door, restated on the wire
// side. The redaction bit set here is carried forward by every
// [NewReasoningDelta] and [NewReasoningBlockEnd] built from the returned
// event's payload, and is never re-decided by either.
//
// It is the second of the two doors in. On failure the zero [Event] is
// returned. The returned event is unstamped (sequence 0); stamping is a
// [Stamper]'s job, per-stream, at the producer boundary.
func NewRedactedReasoningBlockStart(blockIndex uint64) (Event, error) {
	payload := ReasoningBlockStart{block: BlockIndex(blockIndex), redacted: true}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// ReasoningBlockStart returns the event's reasoning-block-start payload, and
// whether the event carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [ReasoningBlockStart] and false, and
// never panics.
func (e Event) ReasoningBlockStart() (ReasoningBlockStart, bool) {
	payload, ok := e.payload.(ReasoningBlockStart)
	if !ok {
		return ReasoningBlockStart{}, false
	}
	return payload, true
}

// BlockIndex returns the index of the block this event opens (V-STR-15),
// unwrapping AI-14's internal [BlockIndex] type to the plain uint64 shape
// AI-16's accessors already use.
func (r ReasoningBlockStart) BlockIndex() uint64 { return uint64(r.block) }

// Redacted reports whether this block's plaintext was withheld by the
// provider (R-ARE-012). It is the only place this signal is carried: no
// other event of this family exposes it, and no event stores a
// [ReasoningState] — a reconstructed block's state is derived exactly as
// [Reasoning.State] derives it.
func (r ReasoningBlockStart) Redacted() bool { return r.redacted }

// String renders the reasoning-block-start event for a diagnostic reader,
// naming the type and nothing it carries — [ResponseStart.String]'s posture,
// restated (V-FAIL-13). Not the block index and not the redacted signal: a
// diagnostic rendering never reproduces caller-derived context.
func (r ReasoningBlockStart) String() string { return "reasoningblockstart" }

// GoString renders the reasoning-block-start event for the %#v verb.
//
// Without it, %#v falls back to reflection and prints the unexported fields,
// which would make the redaction posture a property of which verb someone
// reached for rather than a property of the type.
func (r ReasoningBlockStart) GoString() string { return r.String() }

// ReasoningDelta carries one fragment of new reasoning text for the block its
// event belongs to (R-ARE-005): only the bytes new since the block's
// previous delta, never the accumulated content, never token bytes, never a
// redacted signal or a [ReasoningState].
type ReasoningDelta struct {
	block    BlockIndex
	fragment string
	// redacted mirrors the block-start payload's own bit, copied at
	// construction (never a second, independently-writable copy a caller
	// could set): it is what lets validate re-check S-ARE-036's
	// construction-time rule from CheckEmit too, which only ever calls a
	// payload's own validate. It has no exported accessor: R-ARE-005
	// forbids exposing a redacted signal on the delta kind, and this field
	// exists only to reproduce the constructor's own rule.
	redacted bool
}

// kind reports this payload's registered kind.
func (r ReasoningDelta) kind() EventKind { return EventKindReasoningDelta }

// blockIndex reports the block this payload's event belongs to, satisfying
// the unexported blockPayload interface event.go's CheckEmit asserts to.
func (r ReasoningDelta) blockIndex() BlockIndex { return r.block }

// validate reports the payload's first broken rule at the given position, or
// nil, in order (V-FAIL-04): the block index is at least 1, else
// [ErrOutOfRange] at "block_index" (R-ARE-003); then the fragment is at most
// [MaxTextLen] bytes, else [ErrOutOfRange] at "fragment" (R-ARE-006). No
// emptiness rule applies to the fragment: a zero-length or whitespace-only
// fragment is legal (R-ARE-007) — Reasoning.validate's complete-part rules do
// not apply to a fragment, which is never a complete part on its own.
func (r ReasoningDelta) validate(at Path) *Violation {
	return firstViolation(
		blockIndexRule(at, r.block),
		func() *Violation {
			if len(r.fragment) > MaxTextLen {
				return Invalid(ErrOutOfRange, under(at, At("fragment"))...)
			}
			return nil
		},
	)
}

// NewReasoningDelta constructs one fragment of new reasoning text for the
// block start opened (R-ARE-005, R-ARE-006, R-ARE-007).
//
// start is the event returned by [NewReasoningBlockStart] or
// [NewRedactedReasoningBlockStart] — its block index is inherited so the
// four events of one block structurally cannot disagree about it (S-ARE-007),
// and its redaction bit travels with it for the rule
// [ReasoningDelta.validate] applies once AI-17.3 lands. The rules are
// [ReasoningDelta.validate]'s, run from here and again at [CheckEmit]
// (R-AEE-010). On failure the zero [Event] is returned. The returned event is
// unstamped (sequence 0); stamping is a [Stamper]'s job, per-stream, at the
// producer boundary.
//
// fragment is stored byte for byte: nothing here validates its encoding,
// trims it, or rejects it for being empty or whitespace-only. An individual
// fragment may not be well-formed UTF-8 on its own when a multi-byte rune
// splits across a delta boundary (R-ARE-006).
func NewReasoningDelta(start ReasoningBlockStart, fragment []byte) (Event, error) {
	payload := ReasoningDelta{block: start.block, fragment: string(fragment), redacted: start.redacted}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// ReasoningDelta returns the event's reasoning-delta payload, and whether the
// event carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [ReasoningDelta] and false, and never
// panics.
func (e Event) ReasoningDelta() (ReasoningDelta, bool) {
	payload, ok := e.payload.(ReasoningDelta)
	if !ok {
		return ReasoningDelta{}, false
	}
	return payload, true
}

// BlockIndex returns the index of the block this delta belongs to
// (V-STR-15).
func (r ReasoningDelta) BlockIndex() uint64 { return uint64(r.block) }

// Fragment returns the fragment's raw bytes — the bytes new since this
// block's previous delta, never the block's accumulated content (V-STR-16,
// R-ARE-005).
//
// The result is byte-equal to what [NewReasoningDelta] was given: nothing on
// this path decodes, validates, trims or re-encodes it. A fragment may be
// invalid UTF-8 on its own when a multi-byte rune splits across a delta
// boundary (R-ARE-006).
func (r ReasoningDelta) Fragment() []byte { return []byte(r.fragment) }

// String renders the reasoning-delta event for a diagnostic reader, naming
// the type and nothing it carries — not the fragment and not its length: a
// length is caller-derived and a prefix of a secret is still a secret
// (V-FAIL-13).
func (r ReasoningDelta) String() string { return "reasoningdelta" }

// GoString renders the reasoning-delta event for the %#v verb.
func (r ReasoningDelta) GoString() string { return r.String() }

// ReasoningBlockEnd closes a reasoning block, carrying its opaque round-trip
// token whole (R-ARE-009).
type ReasoningBlockEnd struct {
	block    BlockIndex
	token    string
	hasToken bool
	// redacted mirrors the block-start payload's own bit, copied at
	// construction — the same reproducibility reason [ReasoningDelta.redacted]
	// documents. No exported accessor: this event's own [ReasoningBlockEnd.Token]
	// is the only token-shaped surface it exposes.
	redacted bool
}

// kind reports this payload's registered kind.
func (r ReasoningBlockEnd) kind() EventKind { return EventKindReasoningBlockEnd }

// blockIndex reports the block this payload's event belongs to, satisfying
// the unexported blockPayload interface event.go's CheckEmit asserts to.
func (r ReasoningBlockEnd) blockIndex() BlockIndex { return r.block }

// validate reports the payload's first broken rule at the given position, or
// nil, in order (V-FAIL-04): the block index is at least 1, else
// [ErrOutOfRange] at "block_index" (R-ARE-003); then the token, when present,
// is at most [MaxReasoningTokenLen] bytes, else [ErrOutOfRange] at "token" —
// [Reasoning.validate]'s only rule on the token, restated verbatim
// (R-ARE-010).
func (r ReasoningBlockEnd) validate(at Path) *Violation {
	return firstViolation(
		blockIndexRule(at, r.block),
		func() *Violation {
			if r.hasToken && len(r.token) > MaxReasoningTokenLen {
				return Invalid(ErrOutOfRange, under(at, At("token"))...)
			}
			return nil
		},
	)
}

// NewReasoningBlockEnd constructs the end of the reasoning block start
// opened, carrying its round-trip token (R-ARE-009..011).
//
// start is the event returned by [NewReasoningBlockStart] or
// [NewRedactedReasoningBlockStart] — see [NewReasoningDelta] for why the
// typed start, not a bare index, is the parameter. The rules are
// [ReasoningBlockEnd.validate]'s, run from here and again at [CheckEmit]
// (R-AEE-010). On failure the zero [Event] is returned. The returned event is
// unstamped (sequence 0); stamping is a [Stamper]'s job, per-stream, at the
// producer boundary.
//
// A nil token means the provider attached none. Any non-nil slice, including
// one of length zero, is a token that is present — [Reasoning.Token]'s
// distinction, kept verbatim on the wire side (R-ARE-011).
func NewReasoningBlockEnd(start ReasoningBlockStart, token []byte) (Event, error) {
	payload := ReasoningBlockEnd{block: start.block, token: string(token), hasToken: token != nil, redacted: start.redacted}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// ReasoningBlockEnd returns the event's reasoning-block-end payload, and
// whether the event carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [ReasoningBlockEnd] and false, and never
// panics.
func (e Event) ReasoningBlockEnd() (ReasoningBlockEnd, bool) {
	payload, ok := e.payload.(ReasoningBlockEnd)
	if !ok {
		return ReasoningBlockEnd{}, false
	}
	return payload, true
}

// BlockIndex returns the index of the block this event closes (V-STR-15).
func (r ReasoningBlockEnd) BlockIndex() uint64 { return uint64(r.block) }

// Token returns the opaque round-trip token and whether one is present at
// all — [Reasoning.Token]'s two-result presence contract, reused verbatim
// (R-ARE-011): absent is not empty. The bytes come back exactly as they were
// supplied: nothing on the construction path or the read path parses,
// verifies, decrypts, normalizes, trims or re-encodes them (R-ARE-010).
// Called on a block-end event with no token, it returns nil and false.
func (r ReasoningBlockEnd) Token() ([]byte, bool) { return []byte(r.token), r.hasToken }

// String renders the reasoning-block-end event for a diagnostic reader,
// naming the type and nothing it carries — not the token and not its length:
// a length is caller-derived and a prefix of a secret is still a secret
// (V-FAIL-13).
func (r ReasoningBlockEnd) String() string { return "reasoningblockend" }

// GoString renders the reasoning-block-end event for the %#v verb.
//
// Without it, %#v falls back to reflection and prints the unexported fields
// — including the token — which would make the redaction posture a property
// of which verb someone reached for rather than a property of the type.
func (r ReasoningBlockEnd) GoString() string { return r.String() }
