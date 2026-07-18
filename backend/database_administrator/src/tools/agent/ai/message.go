package ai

import "errors"

// Message is the provider-neutral wire turn sent to the model. See
// AI-00 § message (Layer 1 sense) for the cross-layer disambiguation.
//
// As of AI-05 the Message shell exposes the Role + optional stable ID
// (AI-04) plus a Content slice of ContentPart values (AI-05). The
// Content slice is the union defined in content.go; v1 only populates
// KindText parts, but the slice + reserved Kind slots mean AI-06
// (reasoning), AI-07 (tool declarations), AI-08 (tool calls + tool
// results) are additive changes — no Message struct break.
//
// Wire-format note: Message JSON marshaling/unmarshaling is owned by
// AI-11 (event envelope).
type Message struct {
	// Role is REQUIRED and validated. The zero value is invalid.
	Role Role

	// ID is the optional stable identity of this message. Empty string
	// means "no stable identity" (the zero value is meaningful, not an
	// error). AI-08 uses ID to correlate tool calls with tool results.
	ID string

	// Content is the ordered sequence of ContentPart values that make up
	// this message's payload. v1 only populates KindText; other Kind
	// slots are reserved per AI-02 § 10 deferral list and AI-05 §
	// Multi-part semantics. The zero value (nil slice) is invalid;
	// Validate() returns ErrEmptyContent if Content is empty.
	//
	// See AI-00 § message (Layer 1 sense): "one role and a sequence of
	// content parts".
	Content []ContentPart
}

// ErrEmptyContent is returned by Message.Validate when Content is nil
// or empty. AI-17 will promote this to a richer validation taxonomy;
// the v1 sentinel is exported so callers can use errors.Is.
var ErrEmptyContent = errors.New("ai: empty message content")

// Validate returns:
//   - ErrInvalidRole if Role is not a v1 canonical role.
//   - ErrEmptyContent if Content is nil or empty.
//   - ErrNilContentPart if Content contains a typed-nil interface.
//
// Structural invariants only: AI-05 does not validate per-part content
// (e.g., "is this Text value meaningful?"). Per-part semantic
// validation is AI-17's concern at the request level. See AI-05 §
// Validation rules.
//
// Order of checks matters: Role is checked first so callers can
// distinguish "no role" (programmer error) from "no content"
// (validation error). The Content checks are ordered so the cheaper
// length check runs before the per-element interface-nil loop.
func (m Message) Validate() error {
	if !m.Role.IsValid() {
		return ErrInvalidRole
	}
	if len(m.Content) == 0 {
		return ErrEmptyContent
	}
	for _, part := range m.Content {
		// part == nil is the interface-nil check (a nil interface slot).
		// The current textPart wrapper is a struct (not a pointer), so
		// a typed-nil textPart does not exist; future AI-06 / AI-07 /
		// AI-08 wrappers must either stay struct-valued (no typed-nil
		// possibility) or be checked explicitly here. The interface-nil
		// check catches var p ContentPart; p == nil regardless of
		// concrete wrapper type.
		if part == nil {
			return ErrNilContentPart
		}
	}
	return nil
}
