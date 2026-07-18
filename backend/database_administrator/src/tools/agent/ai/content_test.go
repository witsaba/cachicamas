package ai_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// TestKind_AllSlotsPresent verifies that the Kind enum exposes exactly
// the 7 canonical constants reserved by AI-05 (D-A per design resolution
// obs #2039) and that each has a stable, distinct wire-format string.
// Spec scenarios covered: "All 7 constants exist and are distinct",
// "Kind has a stable wire-format string".
func TestKind_AllSlotsPresent(t *testing.T) {
	want := map[ai.Kind]string{
		ai.KindText:            "text",
		ai.KindReasoning:       "reasoning",
		ai.KindImage:           "image",
		ai.KindAudio:           "audio",
		ai.KindToolDeclaration: "tool_declaration",
		ai.KindToolCall:        "tool_call",
		ai.KindToolResult:      "tool_result",
	}
	if got := len(want); got != 7 {
		t.Fatalf("AI-05 D-A requires exactly 7 Kind slots, found %d", got)
	}
	seen := make(map[ai.Kind]string, len(want))
	for k, wire := range want {
		// Per AI-05 § JSON deferral: Kind is type Kind string so the
		// underlying string IS the wire-format. Any constant that breaks
		// this contract fails loudly here.
		if string(k) != wire {
			t.Errorf("Kind constant %q has wire-format %q, want %q", k, string(k), wire)
		}
		if prev, dup := seen[k]; dup {
			t.Errorf("duplicate Kind constant %q (wire %q) shadows %q", k, wire, prev)
		}
		seen[k] = wire
	}
}

// TestKind_String_StableIdentifiers verifies that Kind.String() returns
// the same wire-format string as the underlying typed-string value.
// Spec scenario covered: "Kind has a stable wire-format string".
func TestKind_String_StableIdentifiers(t *testing.T) {
	cases := []struct {
		k    ai.Kind
		want string
	}{
		{ai.KindText, "text"},
		{ai.KindReasoning, "reasoning"},
		{ai.KindImage, "image"},
		{ai.KindAudio, "audio"},
		{ai.KindToolDeclaration, "tool_declaration"},
		{ai.KindToolCall, "tool_call"},
		{ai.KindToolResult, "tool_result"},
	}
	for _, tc := range cases {
		t.Run(string(tc.k), func(t *testing.T) {
			if got := tc.k.String(); got != tc.want {
				t.Errorf("Kind(%q).String() = %q, want %q", tc.k, got, tc.want)
			}
		})
	}
}

// TestKind_IsValid_AcceptsCanonical verifies that Kind.IsValid() returns
// true for every one of the 7 canonical constants and false for the
// zero value. Spec scenarios covered: "Unknown Kind values are rejected
// at validation boundaries" (happy half).
func TestKind_IsValid_AcceptsCanonical(t *testing.T) {
	canonical := []ai.Kind{
		ai.KindText,
		ai.KindReasoning,
		ai.KindImage,
		ai.KindAudio,
		ai.KindToolDeclaration,
		ai.KindToolCall,
		ai.KindToolResult,
	}
	for _, k := range canonical {
		t.Run(string(k), func(t *testing.T) {
			if !k.IsValid() {
				t.Errorf("canonical Kind %q must be valid (IsValid returned false)", k)
			}
		})
	}
	// Zero value: Kind("") must be invalid. Zero values cannot silently
	// become valid wire values (AI-04 invariant extended to Kind).
	if (ai.Kind("")).IsValid() {
		t.Error("zero-value Kind(\"\") must be invalid")
	}
}

// TestKind_IsValid_RejectsUnknown verifies that Kind.IsValid() returns
// false for Kind values outside the 7-slot canonical set. Spec scenario
// covered: "Unknown Kind is rejected at the validation boundary".
func TestKind_IsValid_RejectsUnknown(t *testing.T) {
	unknown := []ai.Kind{
		"video",
		"file",
		"text ",
		" TEXT",
		"tool_declarations", // plural, ambiguous with reserved slot
		"tool-call",         // hyphenated form, not the underscore wire format
		"toolResult",        // camelCase, not the snake_case wire format
	}
	for _, k := range unknown {
		t.Run(string(k), func(t *testing.T) {
			if k.IsValid() {
				t.Errorf("unknown Kind %q must be invalid", k)
			}
		})
	}
}

// TestContentPartFromText_HappyPath verifies the sanctioned constructor
// returns a ContentPart whose Kind() reports KindText. Spec scenario
// covered: "A Text content part reports KindText".
func TestContentPartFromText_HappyPath(t *testing.T) {
	part, err := ai.ContentPartFromText("hello, world!")
	if err != nil {
		t.Fatalf("ContentPartFromText(hello) returned error %v, want nil", err)
	}
	if part == nil {
		t.Fatal("ContentPartFromText returned nil ContentPart")
	}
	if got := part.Kind(); got != ai.KindText {
		t.Errorf("ContentPartFromText(hello).Kind() = %q, want %q", got, ai.KindText)
	}
}

// TestContentPartFromText_HappyPath_PreservesUnicode verifies the
// sanctioned constructor accepts multibyte UTF-8 (no silent truncation).
// Triangulates with TestContentPartFromText_HappyPath.
func TestContentPartFromText_HappyPath_PreservesUnicode(t *testing.T) {
	cases := []string{
		"héllo, wörld 🌍",
		"你好，世界",
		"\u3000-leading ideographic space + content", // NB: leading U+3000 makes this NON-whitespace-only — content follows
		"line1\nline2",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			part, err := ai.ContentPartFromText(in)
			if err != nil {
				t.Fatalf("ContentPartFromText(%q) returned error %v, want nil", in, err)
			}
			if part == nil {
				t.Fatalf("ContentPartFromText(%q) returned nil ContentPart", in)
			}
			if got := part.Kind(); got != ai.KindText {
				t.Errorf("ContentPartFromText(%q).Kind() = %q, want %q", in, got, ai.KindText)
			}
		})
	}
}

// TestContentPart_InterfaceSatisfaction is a compile-time check that
// ContentPartFromText returns a value implementing the ContentPart
// interface. If the interface or wrapper drifts, this test fails to
// compile.
func TestContentPart_InterfaceSatisfaction(t *testing.T) {
	part, _ := ai.ContentPartFromText("compile-time check")
	var _ ai.ContentPart = part
}

// TestErrNilContentPart_IsExportedAndTyped verifies ErrNilContentPart
// is a typed sentinel error usable with errors.Is.
func TestErrNilContentPart_IsExportedAndTyped(t *testing.T) {
	if !errors.Is(ai.ErrNilContentPart, ai.ErrNilContentPart) {
		t.Error("ErrNilContentPart must be a typed sentinel error compatible with errors.Is")
	}
	if ai.ErrNilContentPart == nil {
		t.Error("ErrNilContentPart must not be nil")
	}
}
