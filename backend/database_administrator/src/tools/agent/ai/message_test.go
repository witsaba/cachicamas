package ai_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// mustPart wraps ContentPartFromText for slice-literal contexts where
// the two-value return is awkward. Test inputs are all hardcoded valid
// strings, so the error is guaranteed nil. Use only in slice literals
// where you cannot bind the error; prefer binding the error directly
// in production-style tests.
func mustPart(s string) ai.ContentPart {
	p, _ := ai.ContentPartFromText(s)
	return p
}

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
	// Empty ID is meaningful (means "no stable identity"); Role + Content are validated here.
	m := ai.Message{
		Role:    ai.RoleUser,
		Content: []ai.ContentPart{mustPart("hello")},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Message{RoleUser, Content: [text(hello)]}.Validate() = %v, want nil", err)
	}
}

func TestMessage_Validate_AcceptsCanonicalRoleWithIdentity(t *testing.T) {
	m := ai.Message{
		Role:    ai.RoleAssistant,
		ID:      "msg-42",
		Content: []ai.ContentPart{mustPart("hi")},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Message{RoleAssistant, ID: msg-42, Content: [text(hi)]}.Validate() = %v, want nil", err)
	}
}

func TestMessage_Validate_AcceptsAllCanonicalRoles(t *testing.T) {
	for _, role := range []ai.Role{ai.RoleSystem, ai.RoleUser, ai.RoleAssistant, ai.RoleTool} {
		m := ai.Message{
			Role:    role,
			Content: []ai.ContentPart{mustPart("body")},
		}
		if err := m.Validate(); err != nil {
			t.Errorf("Message{Role: %q, Content: [text(body)]}.Validate() = %v, want nil", role, err)
		}
	}
}

// --- AI-05 Content validation tests ---

// TestMessage_Validate_RejectsNilContent verifies ErrEmptyContent for
// a Message with a valid canonical Role but a nil Content slice.
// Spec scenario covered: "Message with nil Content fails Validate".
func TestMessage_Validate_RejectsNilContent(t *testing.T) {
	m := ai.Message{Role: ai.RoleUser}
	if err := m.Validate(); !errors.Is(err, ai.ErrEmptyContent) {
		t.Errorf("Message{RoleUser, Content: nil}.Validate() = %v, want ErrEmptyContent", err)
	}
}

// TestMessage_Validate_RejectsZeroLengthContent verifies ErrEmptyContent
// for an explicit empty slice (not nil). The spec requires both nil and
// empty to be rejected with the same sentinel. Spec scenario covered:
// "Message with empty Content slice fails Validate".
func TestMessage_Validate_RejectsZeroLengthContent(t *testing.T) {
	m := ai.Message{Role: ai.RoleUser, Content: []ai.ContentPart{}}
	if err := m.Validate(); !errors.Is(err, ai.ErrEmptyContent) {
		t.Errorf("Message{RoleUser, Content: []}.Validate() = %v, want ErrEmptyContent", err)
	}
}

// TestMessage_Validate_RejectsTypedNilPart verifies ErrNilContentPart
// when one of the Content slots is an interface-nil value (the typed-
// nil trap). Spec scenario covered: "Message with a typed-nil
// ContentPart fails Validate".
func TestMessage_Validate_RejectsTypedNilPart(t *testing.T) {
	parts := []ai.ContentPart{mustPart("hello"), nil}
	m := ai.Message{Role: ai.RoleUser, Content: parts}
	if err := m.Validate(); !errors.Is(err, ai.ErrNilContentPart) {
		t.Errorf("Message{Content: [text(hello), nil]}.Validate() = %v, want ErrNilContentPart", err)
	}
}

// TestMessage_Validate_RejectsInterfaceNilPart verifies ErrNilContentPart
// when the only slot is interface-nil (no concrete value). Triangulates
// with TestMessage_Validate_RejectsTypedNilPart: the check must catch
// interface-nil regardless of whether other parts are present.
// Spec scenario covered: "Message with an interface-nil ContentPart
// fails Validate".
func TestMessage_Validate_RejectsInterfaceNilPart(t *testing.T) {
	parts := []ai.ContentPart{nil}
	m := ai.Message{Role: ai.RoleUser, Content: parts}
	if err := m.Validate(); !errors.Is(err, ai.ErrNilContentPart) {
		t.Errorf("Message{Content: [nil]}.Validate() = %v, want ErrNilContentPart", err)
	}
}

// TestMessage_Validate_AcceptsSinglePartTextContent verifies the happy
// path with a single Text part. Spec scenario covered: "Valid message
// with a single text part passes Validate".
func TestMessage_Validate_AcceptsSinglePartTextContent(t *testing.T) {
	m := ai.Message{
		Role:    ai.RoleUser,
		Content: []ai.ContentPart{mustPart("hello")},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Message{Content: [text(hello)]}.Validate() = %v, want nil", err)
	}
}

// TestMessage_Validate_AcceptsMultiPartTextContent verifies the happy
// path with two distinct text parts (no auto-concatention at Layer 1;
// byte-identical parts are distinct; providers handle concatenation).
// Spec scenario covered: "Valid message with a single text part passes
// Validate" (extended to multi-part per design resolution obs #2039 § 5).
func TestMessage_Validate_AcceptsMultiPartTextContent(t *testing.T) {
	m := ai.Message{
		Role: ai.RoleUser,
		Content: []ai.ContentPart{
			mustPart("hello, "),
			mustPart("world!"),
		},
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Message{Content: [text(hello, ), text(world!)]}.Validate() = %v, want nil", err)
	}
}

// TestMessage_Validate_RoleCheckRunsFirst pins the order of checks:
// when Role is invalid, the validate MUST return ErrInvalidRole
// regardless of Content. This is the AI-04 invariant preserved by the
// AI-05 extension. Spec scenario covered: "Message with a nil Role
// fails Validate (AI-04 preserved)".
func TestMessage_Validate_RoleCheckRunsFirst(t *testing.T) {
	// Invalid Role + nil Content: must report ErrInvalidRole, NOT ErrEmptyContent.
	m := ai.Message{Role: ai.Role("unknown")}
	if err := m.Validate(); !errors.Is(err, ai.ErrInvalidRole) {
		t.Errorf("Message{unknown Role, Content: nil}.Validate() = %v, want ErrInvalidRole (role check runs first)", err)
	}
	// Invalid Role + empty Content: must report ErrInvalidRole.
	m = ai.Message{Role: ai.Role("unknown"), Content: []ai.ContentPart{}}
	if err := m.Validate(); !errors.Is(err, ai.ErrInvalidRole) {
		t.Errorf("Message{unknown Role, Content: []}.Validate() = %v, want ErrInvalidRole (role check runs first)", err)
	}
	// Invalid Role + typed-nil part: must report ErrInvalidRole.
	m = ai.Message{
		Role:    ai.Role("unknown"),
		Content: []ai.ContentPart{nil},
	}
	if err := m.Validate(); !errors.Is(err, ai.ErrInvalidRole) {
		t.Errorf("Message{unknown Role, Content: [nil]}.Validate() = %v, want ErrInvalidRole (role check runs first)", err)
	}
}

// TestMessage_Validate_ContentCheckRunsBeforeNilPart pins the order of
// Content checks: when Content is empty, the validate MUST return
// ErrEmptyContent (not loop over a zero-length slice and report no
// error). Triangulates with the role-first invariant.
func TestMessage_Validate_ContentCheckRunsBeforeNilPart(t *testing.T) {
	// Empty Content: must report ErrEmptyContent.
	m := ai.Message{Role: ai.RoleUser, Content: []ai.ContentPart{}}
	if err := m.Validate(); !errors.Is(err, ai.ErrEmptyContent) {
		t.Errorf("Message{Content: []}.Validate() = %v, want ErrEmptyContent", err)
	}
}

// TestErrEmptyContent_IsExportedAndTyped verifies ErrEmptyContent is
// a typed sentinel error usable with errors.Is.
func TestErrEmptyContent_IsExportedAndTyped(t *testing.T) {
	if !errors.Is(ai.ErrEmptyContent, ai.ErrEmptyContent) {
		t.Error("ErrEmptyContent must be a typed sentinel error compatible with errors.Is")
	}
	if ai.ErrEmptyContent == nil {
		t.Error("ErrEmptyContent must not be nil")
	}
	if ai.ErrEmptyContent.Error() == "" {
		t.Error("ErrEmptyContent.Error() must be non-empty")
	}
}

// TestErrEmptyContent_DistinctFromOtherSentinels verifies
// ErrEmptyContent is NOT aliasing ErrInvalidRole or ErrNilContentPart.
func TestErrEmptyContent_DistinctFromOtherSentinels(t *testing.T) {
	if errors.Is(ai.ErrEmptyContent, ai.ErrInvalidRole) {
		t.Error("ErrEmptyContent must NOT alias ErrInvalidRole")
	}
	if errors.Is(ai.ErrEmptyContent, ai.ErrNilContentPart) {
		t.Error("ErrEmptyContent must NOT alias ErrNilContentPart")
	}
}
