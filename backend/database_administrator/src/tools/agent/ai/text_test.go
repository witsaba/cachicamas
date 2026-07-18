package ai_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// TestMaxTextLength_Value pins the public MaxTextLength constant to
// 1 MiB (1,048,576 bytes). Per AI-05 spec § B "Requirement: NewText
// validates maximum length" — the limit MUST be 1 MiB.
func TestMaxTextLength_Value(t *testing.T) {
	const want = 1 << 20 // 1 MiB = 1,048,576 bytes
	if ai.MaxTextLength != want {
		t.Errorf("MaxTextLength = %d, want %d", ai.MaxTextLength, want)
	}
}

// TestNewText_AcceptsValidString verifies the happy path. Spec scenario
// covered: "NewText accepts a valid ASCII string".
func TestNewText_AcceptsValidString(t *testing.T) {
	cases := []string{
		"hello",
		"hello, world!",
		"a",
		strings.Repeat("a", 100),
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := ai.NewText(in)
			if err != nil {
				t.Fatalf("NewText(%q) returned error %v, want nil", in, err)
			}
			if got.Value() != in {
				t.Errorf("NewText(%q).Value() = %q, want %q", in, got.Value(), in)
			}
			if got.Kind() != ai.KindText {
				t.Errorf("NewText(%q).Kind() = %q, want %q", in, got.Kind(), ai.KindText)
			}
		})
	}
}

// TestNewText_AcceptsAccentedUnicode verifies Unicode input is accepted.
// Per D-C (design resolution obs #2039), NFC normalization is deferred —
// NewText stores input as-is. The test uses NFC-form input so the
// Value() assertion is meaningful without normalization. Spec scenario
// covered: "NewText accepts accented Unicode (NFC-normalized)".
func TestNewText_AcceptsAccentedUnicode(t *testing.T) {
	cases := []string{
		"héllo", // NFC: single rune é
		"你好，世界", // CJK
		"🌍🌎🌏",   // emoji
		"",      // placeholder — actual case is in TestNewText_RejectsEmpty
	}
	cases = cases[:len(cases)-1] // drop the placeholder
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := ai.NewText(in)
			if err != nil {
				t.Fatalf("NewText(%q) returned error %v, want nil", in, err)
			}
			if got.Value() != in {
				t.Errorf("NewText(%q).Value() = %q, want %q", in, got.Value(), in)
			}
		})
	}
}

// TestNewText_RejectsEmpty verifies ErrEmptyText for the empty string.
// Spec scenario covered: "NewText rejects the empty string".
func TestNewText_RejectsEmpty(t *testing.T) {
	got, err := ai.NewText("")
	if !errors.Is(err, ai.ErrEmptyText) {
		t.Errorf("NewText(\"\") error = %v, want ErrEmptyText", err)
	}
	if got.Value() != "" {
		t.Errorf("NewText(\"\").Value() = %q, want zero value \"\"", got.Value())
	}
}

// TestNewText_RejectsASCIISpaces verifies ErrWhitespaceText for ASCII
// spaces. Spec scenario covered: "NewText rejects ASCII spaces".
func TestNewText_RejectsASCIISpaces(t *testing.T) {
	got, err := ai.NewText("  ")
	if !errors.Is(err, ai.ErrWhitespaceText) {
		t.Errorf("NewText(\"  \") error = %v, want ErrWhitespaceText", err)
	}
	if got.Value() != "" {
		t.Errorf("NewText(\"  \").Value() = %q, want zero value \"\"", got.Value())
	}
}

// TestNewText_RejectsNewlinesAndTabs verifies ErrWhitespaceText for
// "\n\t". Spec scenario covered: "NewText rejects newlines and tabs".
func TestNewText_RejectsNewlinesAndTabs(t *testing.T) {
	got, err := ai.NewText("\n\t")
	if !errors.Is(err, ai.ErrWhitespaceText) {
		t.Errorf("NewText(\"\\n\\t\") error = %v, want ErrWhitespaceText", err)
	}
	if got.Value() != "" {
		t.Errorf("NewText(\"\\n\\t\").Value() = %q, want zero value \"\"", got.Value())
	}
}

// TestNewText_RejectsNonBreakingSpace verifies ErrWhitespaceText for
// NBSP U+00A0. Spec scenario covered: "NewText rejects non-breaking
// space".
func TestNewText_RejectsNonBreakingSpace(t *testing.T) {
	got, err := ai.NewText("\u00A0")
	if !errors.Is(err, ai.ErrWhitespaceText) {
		t.Errorf("NewText(\"\\u00A0\") error = %v, want ErrWhitespaceText", err)
	}
	if got.Value() != "" {
		t.Errorf("NewText(\"\\u00A0\").Value() = %q, want zero value \"\"", got.Value())
	}
}

// TestNewText_RejectsIdeographicSpace verifies ErrWhitespaceText for
// U+3000 (ideographic space). Triangulates with
// TestNewText_RejectsNonBreakingSpace — the spec § D3 whitespace rule
// must cover all unicode.IsSpace runes, not just ASCII whitespace.
func TestNewText_RejectsIdeographicSpace(t *testing.T) {
	got, err := ai.NewText("\u3000")
	if !errors.Is(err, ai.ErrWhitespaceText) {
		t.Errorf("NewText(\"\\u3000\") error = %v, want ErrWhitespaceText", err)
	}
	if got.Value() != "" {
		t.Errorf("NewText(\"\\u3000\").Value() = %q, want zero value \"\"", got.Value())
	}
}

// TestNewText_RejectsAllWhitespaceMix verifies ErrWhitespaceText for
// a mix of whitespace runes (NBSP + tabs + newlines). Triangulates
// the whitespace rule with a richer input.
func TestNewText_RejectsAllWhitespaceMix(t *testing.T) {
	cases := []string{
		" \n\t\r ",
		"\u00A0\u00A0\u00A0",
		"\u3000 \t",
		"\n\u00A0\t\u3000\r",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := ai.NewText(in)
			if !errors.Is(err, ai.ErrWhitespaceText) {
				t.Errorf("NewText(%q) error = %v, want ErrWhitespaceText", in, err)
			}
			if got.Value() != "" {
				t.Errorf("NewText(%q).Value() = %q, want zero value \"\"", in, got.Value())
			}
		})
	}
}

// TestNewText_AcceptsLeadingOrTrailingWhitespace verifies that content
// with surrounding whitespace is accepted (only all-whitespace is
// rejected). This pins the rule precisely.
func TestNewText_AcceptsLeadingOrTrailingWhitespace(t *testing.T) {
	cases := []string{
		" hello",
		"hello ",
		" hello ",
		"\thello\n",
		"\u00A0hello\u00A0",
		"hello world",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			got, err := ai.NewText(in)
			if err != nil {
				t.Errorf("NewText(%q) returned error %v, want nil (content with surrounding whitespace is valid)", in, err)
			}
			if got.Value() != in {
				t.Errorf("NewText(%q).Value() = %q, want %q", in, got.Value(), in)
			}
		})
	}
}

// TestNewText_AcceptsExactly1MiB verifies the boundary at MaxTextLength.
// Spec scenario covered: "NewText accepts exactly 1 MiB".
func TestNewText_AcceptsExactly1MiB(t *testing.T) {
	in := strings.Repeat("a", ai.MaxTextLength)
	got, err := ai.NewText(in)
	if err != nil {
		t.Fatalf("NewText(%d bytes) returned error %v, want nil", len(in), err)
	}
	if got.Value() != in {
		t.Errorf("NewText(%d bytes).Value() length = %d, want %d", len(in), len(got.Value()), len(in))
	}
}

// TestNewText_Rejects1MiBPlus1 verifies ErrTextTooLong over the limit.
// Spec scenario covered: "NewText rejects 1 MiB + 1 byte".
func TestNewText_Rejects1MiBPlus1(t *testing.T) {
	in := strings.Repeat("a", ai.MaxTextLength+1)
	got, err := ai.NewText(in)
	if !errors.Is(err, ai.ErrTextTooLong) {
		t.Errorf("NewText(%d bytes) error = %v, want ErrTextTooLong", len(in), err)
	}
	if got.Value() != "" {
		t.Errorf("NewText(over max).Value() = %q, want zero value \"\"", got.Value())
	}
}

// TestNewText_RejectsWayOverMax verifies ErrTextTooLong for inputs
// way over the limit (sanity check, not a spec scenario).
func TestNewText_RejectsWayOverMax(t *testing.T) {
	in := strings.Repeat("a", ai.MaxTextLength*10)
	_, err := ai.NewText(in)
	if !errors.Is(err, ai.ErrTextTooLong) {
		t.Errorf("NewText(%d bytes) error = %v, want ErrTextTooLong", len(in), err)
	}
}

// TestText_ValueReturnsStoredString verifies the round-trip accessor.
// Spec scenario covered: "Value() returns the original input".
func TestText_ValueReturnsStoredString(t *testing.T) {
	cases := []string{
		"hello",
		"héllo🌍",
		"你好，世界",
		"with\nnewlines\tand\ttabs",
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			txt, err := ai.NewText(in)
			if err != nil {
				t.Fatalf("NewText(%q) returned error %v, want nil", in, err)
			}
			if got := txt.Value(); got != in {
				t.Errorf("Text{%q}.Value() = %q, want %q", in, got, in)
			}
		})
	}
}

// TestText_KindReturnsKindText pins the Kind() discriminator.
func TestText_KindReturnsKindText(t *testing.T) {
	txt, err := ai.NewText("anything")
	if err != nil {
		t.Fatalf("NewText returned error %v, want nil", err)
	}
	if got := txt.Kind(); got != ai.KindText {
		t.Errorf("Text.Kind() = %q, want %q", got, ai.KindText)
	}
}

// TestText_FailedNewTextReturnsZeroValue verifies that a failed NewText
// returns a zero-value Text. Spec scenario covered: "A failed NewText
// returns a zero-value Text".
func TestText_FailedNewTextReturnsZeroValue(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"ascii spaces", "  "},
		{"tabs and newlines", "\n\t"},
		{"nbsp", "\u00A0"},
		{"too long", strings.Repeat("x", ai.MaxTextLength+1)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			txt, err := ai.NewText(tc.in)
			if err == nil {
				t.Fatalf("NewText(%q) returned no error, want validation error", tc.in)
			}
			if txt.Value() != "" {
				t.Errorf("NewText(%q).Value() = %q, want zero value \"\" (failed constructors MUST return zero-value Text)", tc.in, txt.Value())
			}
			if txt.Kind() != ai.KindText {
				t.Errorf("NewText(%q).Kind() = %q, want KindText (zero-value Text still satisfies the interface)", tc.in, txt.Kind())
			}
		})
	}
}

// TestContentPartFromText_PropagatesNewTextErrors verifies that
// ContentPartFromText (sanctioned constructor) propagates the same
// validation errors as NewText and returns nil ContentPart on error.
// Spec scenario covered: "A failed NewText cannot be used as a valid
// ContentPart" — the wrapper must reject empty / whitespace / too-long.
func TestContentPartFromText_PropagatesNewTextErrors(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		wantErr error
	}{
		{"empty string", "", ai.ErrEmptyText},
		{"ascii spaces", "  ", ai.ErrWhitespaceText},
		{"newlines tabs", "\n\t", ai.ErrWhitespaceText},
		{"nbsp", "\u00A0", ai.ErrWhitespaceText},
		{"ideographic space", "\u3000", ai.ErrWhitespaceText},
		{"whitespace mix", " \t\n\u00A0\u3000", ai.ErrWhitespaceText},
		{"too long", strings.Repeat("a", ai.MaxTextLength+1), ai.ErrTextTooLong},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			part, err := ai.ContentPartFromText(tc.in)
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("ContentPartFromText(%q) error = %v, want %v", tc.in, err, tc.wantErr)
			}
			if part != nil {
				t.Errorf("ContentPartFromText(%q) returned non-nil ContentPart on error path", tc.in)
			}
		})
	}
}

// TestErrEmptyText_IsExportedAndTyped verifies ErrEmptyText is a typed
// sentinel error usable with errors.Is.
func TestErrEmptyText_IsExportedAndTyped(t *testing.T) {
	if !errors.Is(ai.ErrEmptyText, ai.ErrEmptyText) {
		t.Error("ErrEmptyText must be a typed sentinel error compatible with errors.Is")
	}
	if ai.ErrEmptyText == nil {
		t.Error("ErrEmptyText must not be nil")
	}
	if ai.ErrEmptyText.Error() == "" {
		t.Error("ErrEmptyText.Error() must be non-empty")
	}
}

// TestErrWhitespaceText_IsExportedAndTyped verifies ErrWhitespaceText.
func TestErrWhitespaceText_IsExportedAndTyped(t *testing.T) {
	if !errors.Is(ai.ErrWhitespaceText, ai.ErrWhitespaceText) {
		t.Error("ErrWhitespaceText must be a typed sentinel error compatible with errors.Is")
	}
	if ai.ErrWhitespaceText == nil {
		t.Error("ErrWhitespaceText must not be nil")
	}
	if ai.ErrWhitespaceText.Error() == "" {
		t.Error("ErrWhitespaceText.Error() must be non-empty")
	}
}

// TestErrTextTooLong_IsExportedAndTyped verifies ErrTextTooLong.
func TestErrTextTooLong_IsExportedAndTyped(t *testing.T) {
	if !errors.Is(ai.ErrTextTooLong, ai.ErrTextTooLong) {
		t.Error("ErrTextTooLong must be a typed sentinel error compatible with errors.Is")
	}
	if ai.ErrTextTooLong == nil {
		t.Error("ErrTextTooLong must not be nil")
	}
	if ai.ErrTextTooLong.Error() == "" {
		t.Error("ErrTextTooLong.Error() must be non-empty")
	}
}

// TestErrSentinelsAreDistinct verifies the three sentinels are distinct
// errors (callers can branch on which constraint failed). Sanity check
// against accidental merging of the three errors into one.
func TestErrSentinelsAreDistinct(t *testing.T) {
	seen := map[error]string{
		ai.ErrEmptyText:      "ErrEmptyText",
		ai.ErrWhitespaceText: "ErrWhitespaceText",
		ai.ErrTextTooLong:    "ErrTextTooLong",
	}
	if len(seen) != 3 {
		t.Errorf("expected 3 distinct sentinel errors, found %d", len(seen))
	}
	// Cross-check: an empty input is NOT a whitespace-only input and
	// NOT a too-long input; the sentinels must NOT be aliases of each
	// other via errors.Is.
	if errors.Is(ai.ErrEmptyText, ai.ErrWhitespaceText) {
		t.Error("ErrEmptyText must NOT alias ErrWhitespaceText")
	}
	if errors.Is(ai.ErrEmptyText, ai.ErrTextTooLong) {
		t.Error("ErrEmptyText must NOT alias ErrTextTooLong")
	}
	if errors.Is(ai.ErrWhitespaceText, ai.ErrTextTooLong) {
		t.Error("ErrWhitespaceText must NOT alias ErrTextTooLong")
	}
}
