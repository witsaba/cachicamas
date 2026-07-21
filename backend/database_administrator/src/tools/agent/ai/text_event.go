// Package-level note for the AI-13 text start/delta/end payloads (text_event.go).
//
// This file implements three concrete eventPayload implementations that
// satisfy the AI-11 sealed interface (see event.go): TextStartPayload
// (zero-field marker for the "open" of a text span); TextDeltaPayload
// (one unexported `delta string` field, validated for rune-level UTF-8
// boundary safety); TextEndPayload (zero-field marker for the "close"
// of a text span). The unexported aiPayload() marker seals each type
// into the interface — only this package can implement eventPayload.
//
// Ordering invariants (AI-11) are documented but NOT enforced at runtime
// here: AI-20 (stream testkit) and AI-21 (conformance suite) own
// per-producer stream validation. The payload types validate only their
// own field-level invariants. See AI-13 design #1 (payload scope) and
// #5 (ordering scope).
package ai

import (
	"errors"
)

// ErrInvalidUTF8Boundary is returned by NewTextDeltaPayload when the
// supplied delta ends with a UTF-8 leading byte (0xC2–0xF4), which would
// split a multi-byte rune across the delta boundary and produce invalid
// UTF-8 in the reconstructed stream. Per AI-13 spec § "Requirement:
// Rune-boundary safety" and REQ-AI13-4. Continuation bytes (0x80–0xBF)
// are always the END of a multi-byte rune and are NOT rejected; bytes
// in 0xF5–0xFF are also NOT rejected (the per-byte check accepts them;
// utf8.ValidString on the CONCATENATED stream is a separate concern).
var ErrInvalidUTF8Boundary = errors.New("ai: text delta ends mid-rune (UTF-8 boundary violation)")

// TextStartPayload is the AI-13 event payload for text.start events.
// It is a zero-field open marker: Layer 2 pattern-matches on
// ev.Kind == EventKindTextStart to start text accumulation; the
// payload carries no data. The struct is value-typed with zero
// exported fields; NewTextStartPayload is the sanctioned constructor.
// The unexported aiPayload() marker seals the type into the eventPayload
// interface — only this package can satisfy that interface.
//
// Per AI-13 design #1: payload scope is per-event only; ordering
// invariants are NOT validated here. AI-20 / AI-21 own stream-level
// conformance.
type TextStartPayload struct{}

// aiPayload is the unexported marker method that seals TextStartPayload
// into the eventPayload interface. See event.go for the full contract.
func (TextStartPayload) aiPayload() {}

// Kind returns the canonical EventKind for this payload. Every
// TextStartPayload returns EventKindTextStart verbatim, regardless of
// field values (there are none). The seal guarantees that Event.Kind
// (derived from this in NewEvent) cannot disagree with the payload for
// events constructed via the sanctioned entry points.
func (TextStartPayload) Kind() EventKind { return EventKindTextStart }

// Validate is a no-op: TextStartPayload has no fields and therefore no
// invariants to re-check. Idempotent and pure. Mirrors the
// ResponseStartPayload.Validate pattern (response.go lines 121–141)
// but with no checks.
func (TextStartPayload) Validate() error { return nil }

// NewTextStartPayload constructs a TextStartPayload. There are no
// inputs and no failure paths; the constructor exists for API symmetry
// with NewTextDeltaPayload and NewTextEndPayload and to document the
// sanctioned entry point.
func NewTextStartPayload() (TextStartPayload, error) {
	return TextStartPayload{}, nil
}

// NewTextStartEvent is the producer-side event constructor for
// text.start. It validates first (via NewTextStartPayload) then defers
// to NewEvent so the package-private atomic counter stamps the next
// Sequence and the sealed eventPayload interface stores the payload
// verbatim. Returns the zero Event with the first validation error.
// Per AI-13 spec § "Scenario: Construct a text start event".
func NewTextStartEvent() (Event, error) {
	p, err := NewTextStartPayload()
	if err != nil {
		return Event{}, err
	}
	return NewEvent(p)
}

// TextDeltaPayload is the AI-13 event payload for text.delta events.
// It carries exactly one unexported field `delta string` representing
// an ordered chunk of the streamed text. The struct is value-typed
// with unexported fields; NewTextDeltaPayload is the sanctioned
// constructor that validates the rune-level UTF-8 boundary. The
// unexported aiPayload() marker seals the type into the eventPayload
// interface.
//
// Per AI-13 design #1: payload scope is per-event only; ordering
// invariants (start-before-delta-before-end) are NOT validated here.
// AI-20 / AI-21 own stream-level conformance.
//
// Per AI-13 spec REQ-AI13-2: the rune-level UTF-8 boundary is the
// v1 contract. Grapheme-cluster, ZWJ-sequence, and combining-mark
// boundaries are documented limits (Layer 2 normalizes if needed).
type TextDeltaPayload struct {
	delta string
}

// aiPayload is the unexported marker method that seals TextDeltaPayload
// into the eventPayload interface. See event.go for the full contract.
func (TextDeltaPayload) aiPayload() {}

// Kind returns the canonical EventKind for this payload. Every
// TextDeltaPayload returns EventKindTextDelta verbatim, regardless of
// the field value. The seal guarantees that Event.Kind (derived from
// this in NewEvent) cannot disagree with the payload for events
// constructed via the sanctioned entry points.
func (TextDeltaPayload) Kind() EventKind { return EventKindTextDelta }

// Delta returns the verbatim delta string supplied at construction.
// NewTextDeltaPayload rejects deltas that end mid-rune before storing;
// Validate re-runs the same check. Mirrors the Text.Value() precedent
// (text.go line 86) and ToolCall.Arguments() (toolcall.go line 68).
func (p TextDeltaPayload) Delta() string { return p.delta }

// Validate re-runs the construction-time UTF-8 boundary check. The
// check is idempotent and pure: a valid delta stays valid, an invalid
// delta keeps returning ErrInvalidUTF8Boundary. The empty string is
// VALID (keepalive / zero-content frame — see doc.go AI-13 paragraph).
func (p TextDeltaPayload) Validate() error {
	return validateUTF8Boundary(p.delta)
}

// NewTextDeltaPayload constructs a TextDeltaPayload after validating
// the rune-level UTF-8 boundary on the supplied delta:
//
//  1. Empty delta "" → returns (TextDeltaPayload{}, nil) — keepalive /
//     zero-content frame is valid.
//  2. Non-empty delta whose last byte is a UTF-8 leading byte
//     (0xC2–0xF4) → returns (TextDeltaPayload{}, ErrInvalidUTF8Boundary).
//  3. Otherwise → returns (TextDeltaPayload{delta: delta}, nil).
//
// The check is per-byte; see validateUTF8Boundary for the byte-level
// rationale and the verification table in the design doc. Per AI-13
// spec REQ-AI13-2 and REQ-AI13-5.
//
// Returns the zero value TextDeltaPayload{} with the first validation
// error — never a half-constructed payload.
func NewTextDeltaPayload(delta string) (TextDeltaPayload, error) {
	if err := validateUTF8Boundary(delta); err != nil {
		return TextDeltaPayload{}, err
	}
	return TextDeltaPayload{delta: delta}, nil
}

// NewTextDeltaEvent is the producer-side event constructor for
// text.delta. It validates first (via NewTextDeltaPayload) then defers
// to NewEvent so the package-private atomic counter stamps the next
// Sequence and the sealed eventPayload interface stores the payload
// verbatim. Returns the zero Event with the first validation error.
// Per AI-13 spec § "Scenario: Construct a text delta event".
func NewTextDeltaEvent(delta string) (Event, error) {
	p, err := NewTextDeltaPayload(delta)
	if err != nil {
		return Event{}, err
	}
	return NewEvent(p)
}

// TextEndPayload is the AI-13 event payload for text.end events.
// It is a zero-field close marker: Layer 2 pattern-matches on
// ev.Kind == EventKindTextEnd to stop text accumulation; the payload
// carries no data. The struct is value-typed with zero exported fields;
// NewTextEndPayload is the sanctioned constructor. The unexported
// aiPayload() marker seals the type into the eventPayload interface.
type TextEndPayload struct{}

// aiPayload is the unexported marker method that seals TextEndPayload
// into the eventPayload interface. See event.go for the full contract.
func (TextEndPayload) aiPayload() {}

// Kind returns the canonical EventKind for this payload. Every
// TextEndPayload returns EventKindTextEnd verbatim, regardless of
// field values (there are none). The seal guarantees that Event.Kind
// (derived from this in NewEvent) cannot disagree with the payload for
// events constructed via the sanctioned entry points.
func (TextEndPayload) Kind() EventKind { return EventKindTextEnd }

// Validate is a no-op: TextEndPayload has no fields and therefore no
// invariants to re-check. Idempotent and pure.
func (TextEndPayload) Validate() error { return nil }

// NewTextEndPayload constructs a TextEndPayload. There are no inputs
// and no failure paths; the constructor exists for API symmetry with
// NewTextStartPayload and NewTextDeltaPayload.
func NewTextEndPayload() (TextEndPayload, error) {
	return TextEndPayload{}, nil
}

// NewTextEndEvent is the producer-side event constructor for
// text.end. It validates first (via NewTextEndPayload) then defers
// to NewEvent so the package-private atomic counter stamps the next
// Sequence and the sealed eventPayload interface stores the payload
// verbatim. Returns the zero Event with the first validation error.
func NewTextEndEvent() (Event, error) {
	p, err := NewTextEndPayload()
	if err != nil {
		return Event{}, err
	}
	return NewEvent(p)
}

// validateUTF8Boundary reports whether s is a valid UTF-8 streaming
// boundary:
//
//   - Empty strings are valid (keepalive / zero-content frame).
//   - A non-empty string whose last byte is a UTF-8 leading byte
//     (0xC2–0xF4 inclusive) is REJECTED because the leading byte
//     indicates an incomplete multi-byte sequence: splitting a
//     multi-byte rune across the delta boundary would produce invalid
//     UTF-8 in the reconstructed stream.
//   - Continuation bytes (0x80–0xBF) always end a multi-byte rune
//     and PASS.
//   - ASCII bytes (0x00–0x7F) are single-byte runes and PASS.
//   - Bytes 0xF5–0xFF are invalid in UTF-8 but are NOT a boundary
//     violation this guard catches — they PASS at this layer. The
//     consumer MAY apply utf8.ValidString to the CONCATENATED stream
//     if it needs that guarantee (per REQ-AI13-5 row 12).
//
// The check is O(1) per call. The empty short-circuit prevents a
// negative-length access on s[len(s)-1].
//
// DO NOT introduce a utf8.ValidString check here. REQ-AI13-5 row 12
// requires `"\xFF"` to PASS the per-delta check (because
// utf8.ValidString("\xFF") == false but the boundary semantics allow
// lone invalid bytes; the consumer applies utf8.ValidString on the
// CONCATENATED stream — see REQ-AI13-5 row 12). Adding utf8.ValidString
// would over-reject and break row 12.
//
// DO NOT widen the range to `last >= 0xC2` without an upper bound.
// Bytes 0xF5–0xFF are NOT UTF-8 leading bytes and MUST be accepted at
// this layer for the same reason (REQ-AI13-5 row 12).
func validateUTF8Boundary(s string) error {
	if len(s) == 0 {
		return nil
	}
	for i := 0; i < len(s); {
		b := s[i]
		switch {
		case b < 0x80:
			i++
		case b < 0xC2:
			i++
		case b > 0xF4:
			i++
		default:
			var expected int
			switch {
			case b < 0xE0:
				expected = 2
			case b < 0xF0:
				expected = 3
			default:
				expected = 4
			}
			if i+expected > len(s) {
				return ErrInvalidUTF8Boundary
			}
			for j := 1; j < expected; j++ {
				cont := s[i+j]
				if cont < 0x80 || cont > 0xBF {
					return ErrInvalidUTF8Boundary
				}
			}
			i += expected
		}
	}
	return nil
}

// AsTextStart returns the payload of ev as a TextStartPayload when
// ev.Kind matches EventKindTextStart AND the dynamic payload type is
// TextStartPayload. The kind check guards the type assertion: if a
// caller (or a future regression) bypasses NewEvent by direct struct
// construction with a mismatched Kind, the helper returns (zero, false)
// rather than panicking on the assertion. Returns (zero TextStartPayload,
// false) when either check fails.
//
// Per AI-13 design #3: helpers verify Event.Kind parity BEFORE the type
// assertion so a same-kind-different-payload Event cannot leak through.
func AsTextStart(ev Event) (TextStartPayload, bool) {
	if ev.Kind != EventKindTextStart {
		return TextStartPayload{}, false
	}
	p, ok := ev.Payload.(TextStartPayload)
	if !ok {
		return TextStartPayload{}, false
	}
	return p, true
}

// AsTextDelta returns the payload of ev as a TextDeltaPayload when
// ev.Kind matches EventKindTextDelta AND the dynamic payload type is
// TextDeltaPayload. Same parity check as AsTextStart.
func AsTextDelta(ev Event) (TextDeltaPayload, bool) {
	if ev.Kind != EventKindTextDelta {
		return TextDeltaPayload{}, false
	}
	p, ok := ev.Payload.(TextDeltaPayload)
	if !ok {
		return TextDeltaPayload{}, false
	}
	return p, true
}

// AsTextEnd returns the payload of ev as a TextEndPayload when
// ev.Kind matches EventKindTextEnd AND the dynamic payload type is
// TextEndPayload. Same parity check as AsTextStart.
func AsTextEnd(ev Event) (TextEndPayload, bool) {
	if ev.Kind != EventKindTextEnd {
		return TextEndPayload{}, false
	}
	p, ok := ev.Payload.(TextEndPayload)
	if !ok {
		return TextEndPayload{}, false
	}
	return p, true
}
