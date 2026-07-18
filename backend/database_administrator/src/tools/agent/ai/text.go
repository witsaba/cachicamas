package ai

import (
	"errors"
	"unicode"
)

// MaxTextLength is the upper bound on Text values, in bytes. The value
// is 1 MiB (1,048,576 bytes). Rationale: prevent OOM via accidental
// huge inputs while accommodating long-form text content for any
// realistic provider. Per AI-05 spec § B "Requirement: NewText validates
// maximum length".
const MaxTextLength = 1 << 20

// ErrEmptyText is returned by NewText when the input is the empty
// string. The returned Text is the zero value.
var ErrEmptyText = errors.New("ai: empty text")

// ErrWhitespaceText is returned by NewText when every rune of the
// input reports true for unicode.IsSpace. The check covers ASCII
// spaces, \n, \t, \r, NBSP U+00A0, ideographic space U+3000, etc.
// Per AI-05 spec § B "Requirement: NewText validates whitespace-only
// input".
var ErrWhitespaceText = errors.New("ai: whitespace-only text")

// ErrTextTooLong is returned by NewText when len(s) > MaxTextLength.
// The returned Text is the zero value.
var ErrTextTooLong = errors.New("ai: text exceeds MaxTextLength")

// Text is the provider-neutral value type for a text content part.
// See AI-00 § content part (Layer 1 sense) for the cross-layer
// disambiguation and AI-05 § Text for the v1 design.
//
// Text is a thin wrapper around a UTF-8 string. The field is
// unexported so callers MUST go through NewText (the sanctioned
// constructor) for v1 validation. Direct struct literal construction
// is not part of the public contract.
//
// Wire-format note: JSON marshaling is owned by AI-11 (event envelope).
type Text struct {
	v string
}

// NewText validates and constructs a Text wrapping s.
//
// Validation rules (per AI-05 spec § B and design resolution obs #2039
// § 1 D-B):
//
//  1. If s == "" → returns (Text{}, ErrEmptyText).
//  2. If every rune of s reports true for unicode.IsSpace → returns
//     (Text{}, ErrWhitespaceText).
//  3. If len(s) > MaxTextLength → returns (Text{}, ErrTextTooLong).
//  4. Otherwise returns (Text{v: s}, nil).
//
// On error the returned Text is the zero value.
//
// NFC normalization deferred — see AI-31 normalization milestone.
// NewText stores input as-is in v1.
func NewText(s string) (Text, error) {
	if s == "" {
		return Text{}, ErrEmptyText
	}
	if isAllWhitespace(s) {
		return Text{}, ErrWhitespaceText
	}
	if len(s) > MaxTextLength {
		return Text{}, ErrTextTooLong
	}
	return Text{v: s}, nil
}

// isAllWhitespace reports whether every rune of s reports true for
// unicode.IsSpace. An empty string is NOT all-whitespace by this
// definition (NewText catches empty separately via ErrEmptyText), so
// callers must check emptiness first.
func isAllWhitespace(s string) bool {
	for _, r := range s {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

// Value returns the underlying UTF-8 string carried by this Text.
func (t Text) Value() string {
	return t.v
}

// Kind returns KindText. Text is the v1 Text variant of the
// ContentPart discriminated union; the discriminator is fixed.
func (t Text) Kind() Kind {
	return KindText
}
