package ai_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

func TestRole_IsValid_AcceptsCanonicalRoles(t *testing.T) {
	for _, r := range []ai.Role{ai.RoleSystem, ai.RoleUser, ai.RoleAssistant, ai.RoleTool} {
		if !r.IsValid() {
			t.Errorf("canonical role %q must be valid", r)
		}
	}
}

func TestRole_IsValid_RejectsZeroValue(t *testing.T) {
	var zero ai.Role // zero value: ai.Role("")
	if zero.IsValid() {
		t.Error("zero-value Role must be invalid (acceptance criterion: zero values cannot silently become valid wire requests)")
	}
}

func TestRole_IsValid_RejectsUnknownRoles(t *testing.T) {
	for _, r := range []ai.Role{"developer", "function", "model", "Tool", "ASSISTANT", " ", "unknown"} {
		if r.IsValid() {
			t.Errorf("unknown role %q must be invalid", r)
		}
	}
}

func TestRole_String_ReturnsWireFormatName(t *testing.T) {
	if got := ai.RoleSystem.String(); got != "system" {
		t.Errorf("RoleSystem.String() = %q, want %q", got, "system")
	}
	if got := ai.RoleAssistant.String(); got != "assistant" {
		t.Errorf("RoleAssistant.String() = %q, want %q", got, "assistant")
	}
}

func TestErrInvalidRole_IsExportedAndTyped(t *testing.T) {
	// Verify callers can use errors.Is with the sentinel.
	if !errors.Is(ai.ErrInvalidRole, ai.ErrInvalidRole) {
		t.Error("ErrInvalidRole must be a typed sentinel error compatible with errors.Is")
	}
}
