package ai

// Message is the provider-neutral wire turn sent to the model. See
// AI-00 § message (Layer 1 sense) for the cross-layer disambiguation.
//
// As of AI-04 the Message shell exposes the Role + optional stable ID
// that the content-part layer (AI-05) and the tool-call layer (AI-08)
// will attach to. Content parts and tool calls are deliberately absent
// here; Validate() therefore checks only Role.
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
}

// Validate returns ErrInvalidRole if Role is not a v1 canonical role.
// Content validation is owned by AI-05 and AI-08 (out of scope here).
func (m Message) Validate() error {
	if !m.Role.IsValid() {
		return ErrInvalidRole
	}
	return nil
}
