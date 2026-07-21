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
// supplied delta contains a UTF-8 leading byte (0xC2–0xF4) without its
// expected 1–3 continuation bytes (0x80–0xBF). Such a truncated or
// malformed sequence would split a multi-byte rune across a delta
// boundary. Per AI-13 spec § "Requirement: Rune-boundary safety" and
// REQ-AI13-4. Orphan continuation bytes, overlong lead bytes 0xC0–0xC1,
// and invalid bytes 0xF5–0xFF are not mid-rune splits and pass this
// boundary check; full UTF-8 validity remains a separate concern.
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
//  2. The validator walks the complete string. ASCII, orphan continuation,
//     overlong lead, and invalid bytes advance one byte and pass.
//  3. Each UTF-8 leading byte (0xC2–0xF4) must be followed by the expected
//     1–3 continuation bytes (0x80–0xBF); otherwise the constructor returns
//     (TextDeltaPayload{}, ErrInvalidUTF8Boundary).
//  4. A complete walk returns (TextDeltaPayload{delta: delta}, nil).
//
// See validateUTF8Boundary for the byte-level rationale and the
// verification table in the design doc. Per AI-13 spec REQ-AI13-2 and
// REQ-AI13-5.
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

// validateUTF8Boundary walks s and reports whether every UTF-8 leading
// byte has the complete continuation-byte sequence required to keep the
// delta boundary outside a multi-byte rune:
//
//   - Empty strings are valid (keepalive / zero-content frame).
//   - ASCII bytes (0x00–0x7F) are complete single-byte runes and advance
//     the walk by one byte.
//   - Orphan continuation bytes (0x80–0xBF) and overlong lead bytes
//     (0xC0–0xC1) are not mid-rune splits; they advance one byte and pass
//     per REQ-AI13-5 row 10.
//   - Invalid bytes (0xF5–0xFF) likewise advance one byte and pass per
//     REQ-AI13-5 row 12.
//   - A leading byte (0xC2–0xF4) determines a 2-, 3-, or 4-byte rune. The
//     walk rejects the string with ErrInvalidUTF8Boundary when fewer than
//     the required bytes remain or any required continuation byte falls
//     outside 0x80–0xBF; otherwise it advances by the complete rune length.
//
// The check is O(n) in len(s), idempotent, and deliberately narrower than
// full UTF-8 validation: it validates rune completeness at delta boundaries,
// not Unicode scalar validity or canonical encoding.
//
// DO NOT introduce utf8.ValidString here. REQ-AI13-5 row 12 requires
// `"\xFF"` to pass because a lone invalid byte is not a mid-rune split.
// A consumer that requires full validity may apply utf8.ValidString to the
// concatenated stream.
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
