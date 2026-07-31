// AI-06.2 — the first cross-package readability proof for a real Layer 1 type.
//
// import_compile_test.go reserved this by name: "the external readability of a
// real Layer 1 type — a type that can be constructed here, read back, and
// type-asserted — is proven by AI-06.2, on the first content part that exists.
// This file is its placeholder, not its substitute." This is the substitute.
//
// It is the inverse of doc 0001's defect C2. The retired Layer 1 kept its
// content-part wrappers unexported and exposed only their discriminator, so
// "content cannot be read back out of a request from another package, which
// makes request translation structurally impossible" — a hard blocker on the
// first adapter. Everything below is written from a package that is not ai,
// using nothing but the exported surface, and it is the exact loop AI-26 will
// write.
package agenttest_test

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-06.2 item 1 — an external package constructs a text part, places it in a
// message, and reads the exact text back through the public surface.
//
// Byte-equal is the assertion, not "equal after normalization". A model's input
// is bytes and an adapter renders bytes; a contract that returns a cleaned-up
// version of what it was given is a contract that silently changes a prompt.
func TestContentPart_TextConstructedAndReadFromAnExternalPackage_RoundTripsByteEqual(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		text string
	}{
		{"plain text", "hello"},
		{"an embedded newline", "first line\nsecond line"},
		{"leading and trailing whitespace around content", "  padded  "},
		{"high Unicode", "héllo · 世界 · 🜁 · \U0001F9EA"},
		{"content that looks like markup", "<content type=\"text\">{{ .Body }}</content>"},
		{"a long body", strings.Repeat("paragraph.\n", 4096)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			part, err := ai.NewText(tc.text)
			if err != nil {
				t.Fatalf("ai.NewText returned %v, want no failure", err)
			}

			msg, err := ai.NewMessage(ai.RoleUser, part)
			if err != nil {
				t.Fatalf("ai.NewMessage returned %v, want no failure", err)
			}

			// The loop an adapter writes: discover the kind, call the accessor
			// for it. No type switch, no unexported type, no helper inside
			// package ai.
			content := msg.Content()
			if len(content) != 1 {
				t.Fatalf("len(msg.Content()) = %d, want 1", len(content))
			}

			got := content[0]
			if got.Kind() != ai.PartKindText {
				t.Fatalf("part.Kind() = %v, want %v", got.Kind(), ai.PartKindText)
			}

			text, ok := got.Text()
			if !ok {
				t.Fatalf("part.Text() reported no text on a part of kind %v", got.Kind())
			}
			if text != tc.text {
				t.Errorf("part.Text() = %q, want %q — the payload is not byte-equal", text, tc.text)
			}
			if len(text) != len(tc.text) {
				t.Errorf("len(part.Text()) = %d, want %d — the payload was re-encoded", len(text), len(tc.text))
			}
		})
	}
}
