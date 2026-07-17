package ai_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

func TestMessage_Validate_RejectsZeroValueRole(t *testing.T) {
	var m ai.Message // zero value: ai.Message{Role: "", ID: ""}
	if err := m.Validate(); !errors.Is(err, ai.ErrInvalidRole) {
		t.Errorf("zero-value Message.Validate() = %v, want ErrInvalidRole", err)
	}
}

func TestMessage_Validate_RejectsUnknownRole(t *testing.T) {
	m := ai.Message{Role: ai.Role("not-a-real-role"), ID: "msg-1"}
	if err := m.Validate(); !errors.Is(err, ai.ErrInvalidRole) {
		t.Errorf("Message{unknown Role}.Validate() = %v, want ErrInvalidRole", err)
	}
}

func TestMessage_Validate_AcceptsCanonicalRoleWithNoIdentity(t *testing.T) {
	// Empty ID is meaningful (means "no stable identity"); only Role is validated here.
	m := ai.Message{Role: ai.RoleUser}
	if err := m.Validate(); err != nil {
		t.Errorf("Message{RoleUser}.Validate() = %v, want nil", err)
	}
}

func TestMessage_Validate_AcceptsCanonicalRoleWithIdentity(t *testing.T) {
	m := ai.Message{Role: ai.RoleAssistant, ID: "msg-42"}
	if err := m.Validate(); err != nil {
		t.Errorf("Message{RoleAssistant, ID: msg-42}.Validate() = %v, want nil", err)
	}
}

func TestMessage_Validate_AcceptsAllCanonicalRoles(t *testing.T) {
	for _, role := range []ai.Role{ai.RoleSystem, ai.RoleUser, ai.RoleAssistant, ai.RoleTool} {
		m := ai.Message{Role: role}
		if err := m.Validate(); err != nil {
			t.Errorf("Message{Role: %q}.Validate() = %v, want nil", role, err)
		}
	}
}
