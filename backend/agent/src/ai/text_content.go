// AI-06.2 — text content, the first subject the part contract is proven against.
//
// This file implements V-REQ-08 text content: "the content-part kind carrying
// model-visible natural-language text". It is one file for one kind, which is
// the layout AI-07 and AI-09 inherit: the payload type, its rules, its
// constructor and its accessor live together, so adding a kind is one new file
// plus three lines in content_part.go.

package ai

import "strings"

// MaxTextLen is the documented byte bound on a text content part. Text longer
// than this is a caller-contract failure reported as [ErrOutOfRange].
//
// It is a sanity bound, not a provider limit. Layer 1 holds no model catalog
// (V-OUT-14), so it cannot know any provider's context window and a bound that
// pretended to would be wrong for every provider the day after it was written.
// What the bound is for is making an unbounded value decidable from the request
// alone — the same reasoning V-REQ-24 uses for the breakpoint cap.
//
// It counts bytes, not runes. The cost on a wire is bytes, and counting runes
// would mean decoding the string, which is a re-encoding on the one path whose
// whole promise is that nothing re-encodes. It is exported so a caller can check
// before constructing rather than by catching a failure.
const MaxTextLen = 1 << 20

// textPayload carries model-visible natural-language text (V-REQ-08).
//
// It is unexported and holds a string, so a text part is one interface word wide
// and carries no mutable state: copying a part copies the payload, and a string
// cannot be rewritten through the copy.
type textPayload struct{ text string }

// kind reports this payload's registered kind.
func (textPayload) kind() PartKind { return PartKindText }

// validate reports the payload's first broken rule at the given position, or
// nil. The rules are checked in the order written, per V-FAIL-04:
//
//  1. the text holds at least one non-whitespace character, else [ErrEmpty] —
//     V-REQ-08 says "model-visible", and a part carrying only separators carries
//     nothing a model reads, so an empty string and a string of spaces are one
//     fact and report one failure;
//  2. the text is at most [MaxTextLen] bytes, else [ErrOutOfRange].
//
// Emptiness is first because "you gave me nothing" and "you gave me too much"
// are different facts with different fixes, and a caller fixing one at a time
// must make progress in a predictable direction.
//
// strings.TrimSpace is used as a test and its result is discarded. Storing the
// trimmed text would pass every rule assertion and silently rewrite a caller's
// prompt — the property AI-06.2 item 4 exists to catch.
func (t textPayload) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if strings.TrimSpace(t.text) == "" {
				return Invalid(ErrEmpty, under(at, At("text"))...)
			}
			return nil
		},
		func() *Violation {
			if len(t.text) > MaxTextLen {
				return Invalid(ErrOutOfRange, under(at, At("text"))...)
			}
			return nil
		},
	)
}

// NewText constructs a text content part (V-REQ-08).
//
// It is the only door in. The rules are the payload's own — emptiness first,
// then the documented bound — so the constructor and the message boundary run
// one rule set from two entry points rather than two implementations that have
// to be kept in agreement.
//
// The rules are checked in the order written, per V-FAIL-04:
//
//  1. the text holds at least one non-whitespace character, else [ErrEmpty] at
//     "text";
//  2. the text is at most [MaxTextLen] bytes, else [ErrOutOfRange] at "text".
//
// On failure the zero [Part] is returned, so a caller that ignored the error
// cannot mistake the result for a constructed part: the zero part carries no
// payload, reports no kind, and is rejected by [NewMessage].
//
// Valid text is stored byte for byte. Nothing here trims, normalizes, escapes,
// re-encodes or validates the encoding: a text part may hold embedded newlines,
// surrounding whitespace, astral-plane runes, or bytes that are not well-formed
// UTF-8, and reads back exactly as it was given.
func NewText(text string) (Part, error) {
	payload := textPayload{text: text}
	if violation := payload.validate(nil); violation != nil {
		return Part{}, violation
	}
	return Part{payload: payload}, nil
}

// Text returns the part's text and whether the part carries text at all.
//
// This is the accessor shape every later content kind inherits: a method on
// [Part], named for the kind it reads, returning the typed payload and whether
// the part is of that kind. A consumer switches on [Part.Kind] and calls the
// matching accessor; there is no type switch, because there is only one type.
//
// The second result is what distinguishes "this part carries an empty text" — a
// state the construction rules make unreachable — from "this part is not a text
// part", which is a question with the answer no rather than a caller-contract
// failure. Called on a part of another kind, or on a part that was never
// constructed, it returns "" and false and never panics.
//
// The string is returned by value and is byte-equal to the one [NewText] was
// given: nothing on this path decodes, normalizes, trims or re-encodes it.
func (p Part) Text() (string, bool) {
	payload, ok := p.payload.(textPayload)
	if !ok {
		return "", false
	}
	return payload.text, true
}
