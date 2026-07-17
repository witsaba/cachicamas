package ai

import "errors"

// Role is the provider-neutral role tag on a Message.
//
// AI-04 commits a 4-role canonical set that matches AI-02's REQUIRED
// capabilities. Any other role string is unknown to v1; see IsValid.
//
// Wire-format note: Role serializes as a JSON string (e.g., "assistant").
// Marshaling/unmarshaling is owned by AI-11 (event envelope).
type Role string

const (
	// RoleSystem carries the system instruction / developer message.
	// Required by AI-09 (model request shape).
	RoleSystem Role = "system"

	// RoleUser carries a user turn.
	RoleUser Role = "user"

	// RoleAssistant carries an assistant turn. An assistant message may
	// carry text deltas (AI-13) and/or tool calls (AI-15).
	RoleAssistant Role = "assistant"

	// RoleTool carries a tool result message that echoes back to the
	// model after a tool call. Correlation with the originating call is
	// owned by AI-08.
	RoleTool Role = "tool"
)

// ErrInvalidRole is returned when a Role is not part of the v1 canonical
// set. AI-17 will promote this into the broader validation taxonomy.
var ErrInvalidRole = errors.New("ai: invalid role")

// IsValid reports whether r is one of the v1 canonical roles.
// The zero value Role("") is NOT valid.
func (r Role) IsValid() bool {
	switch r {
	case RoleSystem, RoleUser, RoleAssistant, RoleTool:
		return true
	default:
		return false
	}
}

// String returns the wire-format name of the role.
func (r Role) String() string {
	return string(r)
}
