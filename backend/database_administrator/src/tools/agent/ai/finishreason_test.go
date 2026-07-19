package ai_test

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// TestFinishReason_CanonicalValuesPinned pins the 6 canonical FinishReason
// constants to their expected normalized strings. Any drift here is a wire-
// format break. Per AI-10 spec § "Requirement: Canonical outcomes".
func TestFinishReason_CanonicalValuesPinned(t *testing.T) {
	cases := []struct {
		got  ai.FinishReason
		want string
	}{
		{ai.FinishReasonStop, "stop"},
		{ai.FinishReasonLength, "length"},
		{ai.FinishReasonToolCall, "tool_call"},
		{ai.FinishReasonContentFilter, "content_filter"},
		{ai.FinishReasonCancellation, "cancellation"},
		{ai.FinishReasonUnknown, "unknown"},
	}
	for _, tc := range cases {
		if string(tc.got) != tc.want {
			t.Errorf("FinishReason constant string drift: got %q, want %q", tc.got, tc.want)
		}
		if tc.got.String() != tc.want {
			t.Errorf("FinishReason.String() = %q, want %q", tc.got.String(), tc.want)
		}
	}
}

// TestFinishReason_IsValid_Matrix covers both canonical acceptance and
// zero/unknown rejection. Per AI-10 spec § "Requirement: Canonical outcomes".
func TestFinishReason_IsValid_Matrix(t *testing.T) {
	t.Run("canonical accepted", func(t *testing.T) {
		for _, r := range []ai.FinishReason{
			ai.FinishReasonStop,
			ai.FinishReasonLength,
			ai.FinishReasonToolCall,
			ai.FinishReasonContentFilter,
			ai.FinishReasonCancellation,
			ai.FinishReasonUnknown,
		} {
			if !r.IsValid() {
				t.Errorf("canonical FinishReason %q must be valid", r)
			}
		}
	})
	t.Run("zero and unknown rejected", func(t *testing.T) {
		for _, r := range []ai.FinishReason{
			ai.FinishReason(""), // zero value
			ai.FinishReason("STOP"),
			ai.FinishReason("tool-call"),
			ai.FinishReason("stop "),
			ai.FinishReason("content-filter"),
			ai.FinishReason("refusal"),
			ai.FinishReason("garbage"),
			ai.FinishReason(" "),
		} {
			if r.IsValid() {
				t.Errorf("non-canonical FinishReason %q must be invalid", r)
			}
		}
	})
}

// TestFinishReasonFromProvider_SupportedMapping exercises the known neutral
// semantics. Each supported input MUST map to its canonical outcome. Per
// AI-10 spec § "Requirement: Deterministic provider normalization".
func TestFinishReasonFromProvider_SupportedMapping(t *testing.T) {
	cases := []struct {
		raw  string
		want ai.FinishReason
	}{
		// Stop family — all neutral variants map to Stop.
		{"stop", ai.FinishReasonStop},
		{"end_turn", ai.FinishReasonStop},
		{"STOP", ai.FinishReasonStop}, // case-insensitive: design mapping normalizes
		{"stop_sequence", ai.FinishReasonStop},

		// Length family — all neutral variants map to Length.
		{"length", ai.FinishReasonLength},
		{"max_tokens", ai.FinishReasonLength},
		{"max_output_tokens", ai.FinishReasonLength},

		// Tool call family.
		{"tool_call", ai.FinishReasonToolCall},
		{"tool_use", ai.FinishReasonToolCall},
		{"tool_calls", ai.FinishReasonToolCall},

		// Content filter / refusal family.
		{"content_filter", ai.FinishReasonContentFilter},
		{"refusal", ai.FinishReasonContentFilter},
		{"safety", ai.FinishReasonContentFilter},

		// Cancellation family.
		{"cancellation", ai.FinishReasonCancellation},
		{"cancelled", ai.FinishReasonCancellation},
		{"cancel", ai.FinishReasonCancellation},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got := ai.FinishReasonFromProvider(tc.raw)
			if got != tc.want {
				t.Errorf("FinishReasonFromProvider(%q) = %q, want %q", tc.raw, got, tc.want)
			}
			if !got.IsValid() {
				t.Errorf("FinishReasonFromProvider(%q) returned invalid FinishReason %q", tc.raw, got)
			}
		})
	}
}

// TestFinishReasonFromProvider_UnknownMappingDeterministic verifies that
// empty and unrecognized input ALWAYS return Unknown, with no error, no
// panic, no raw leakage upward. Determinism requires the same unknown
// input to produce the same result every call. Per AI-10 spec §
// "Requirement: Deterministic provider normalization".
//
// Trimmed-and-canonical variants (e.g., "STOP ", " stop") correctly map
// to Stop per the design's case-insensitive + trimmed normalization; they
// are NOT in this list.
func TestFinishReasonFromProvider_UnknownMappingDeterministic(t *testing.T) {
	cases := []ai.FinishReason{
		ai.FinishReasonFromProvider(""),
		ai.FinishReasonFromProvider("garbage"),
		ai.FinishReasonFromProvider("????"),       // punctuation
		ai.FinishReasonFromProvider("COMPLETED"),  // vendor-specific
		ai.FinishReasonFromProvider("\t\n"),       // whitespace
		ai.FinishReasonFromProvider("recitation"), // OpenAI-recitation-style finish
		ai.FinishReasonFromProvider("done"),       // generic verb
	}
	for i, got := range cases {
		if got != ai.FinishReasonUnknown {
			t.Errorf("unknown mapping #%d = %q, want %q", i, got, ai.FinishReasonUnknown)
		}
		if !got.IsValid() {
			t.Errorf("unknown mapping #%d returned invalid FinishReason %q", i, got)
		}
	}

	// Determinism: re-running the same unknown input 100 times returns
	// the same value (catches any non-deterministic state in FromProvider).
	for i := 0; i < 100; i++ {
		if got := ai.FinishReasonFromProvider("unrecognized"); got != ai.FinishReasonUnknown {
			t.Fatalf("FinishReasonFromProvider non-deterministic at iter %d: %q", i, got)
		}
	}
}

// TestFinishReasonFromProvider_NoError verifies the unknown-mapping path
// is a (FinishReason) signature — no error return, no panic. Per AI-10
// spec § "Requirement: Deterministic provider normalization" (MUST return
// Unknown without error or panic).
func TestFinishReasonFromProvider_NoError(t *testing.T) {
	// Use a recover guard to assert no panic on adversarial input.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("FinishReasonFromProvider panicked on unknown input: %v", r)
		}
	}()
	for _, raw := range []string{"", " ", "\n", "???", "COMPLETED", "tool-use"} {
		_ = ai.FinishReasonFromProvider(raw)
	}
}

// TestErrInvalidFinishReason_ExportedAndTyped verifies the sentinel error
// is exported, typed (compatible with errors.Is), and uses the ai: prefix.
// Per AI-10 design "1 sentinel".
func TestErrInvalidFinishReason_ExportedAndTyped(t *testing.T) {
	if ai.ErrInvalidFinishReason == nil {
		t.Fatal("ErrInvalidFinishReason must be non-nil")
	}
	if !errors.Is(ai.ErrInvalidFinishReason, ai.ErrInvalidFinishReason) {
		t.Error("ErrInvalidFinishReason must satisfy errors.Is(err, err)")
	}
	if !strings.HasPrefix(ai.ErrInvalidFinishReason.Error(), "ai: ") {
		t.Errorf("ErrInvalidFinishReason message lacks ai: prefix: %q", ai.ErrInvalidFinishReason.Error())
	}
}

// TestFinishReason_NoMarshalOrClone verifies, via reflection, that the
// FinishReason type exposes no wire-format or cloning methods. AI-10 keeps
// FinishReason as a pure value type; AI-11 owns marshaling. Per AI-10
// design #11 and spec § "Requirement: Layer and scope boundary".
func TestFinishReason_NoMarshalOrClone(t *testing.T) {
	assertNoMethods(t, reflect.TypeOf(ai.FinishReason("")), "MarshalJSON", "UnmarshalJSON", "MarshalText", "UnmarshalText", "Clone")
}

// TestFinishReason_NoSetters verifies, via reflection, that FinishReason
// exposes no With* setter methods. Per AI-10 design (mirror AI-09).
func TestFinishReason_NoSetters(t *testing.T) {
	typ := reflect.TypeOf(ai.FinishReason(""))
	for i := 0; i < typ.NumMethod(); i++ {
		if strings.HasPrefix(typ.Method(i).Name, "With") {
			t.Errorf("FinishReason exposes forbidden setter %s", typ.Method(i).Name)
		}
	}
}

// TestFinishReason_NotContentPart verifies, via reflection, that
// FinishReason does NOT implement ContentPart (no Kind() method). Per
// AI-10 spec § "Requirement: Layer and scope boundary".
func TestFinishReason_NotContentPart(t *testing.T) {
	if _, ok := reflect.TypeOf(ai.FinishReason("")).MethodByName("Kind"); ok {
		t.Error("FinishReason must not implement ContentPart (no Kind() method)")
	}
}

// TestCancellation_ValueVersusLifecycle disambiguates FinishReasonCancellation
// (model-reported cancel) from transport/ctx lifecycle. Per AI-10 spec §
// "Requirement: Cancellation meaning".
func TestCancellation_ValueVersusLifecycle(t *testing.T) {
	// Value-level: a FinishReason reporting cancellation is exactly that — a
	// canonical finish reason. It does NOT mean ctx.Done().
	if !ai.FinishReasonCancellation.IsValid() {
		t.Error("FinishReasonCancellation must be a valid canonical FinishReason")
	}
	if ai.FinishReasonCancellation.String() != "cancellation" {
		t.Errorf("FinishReasonCancellation.String() = %q, want %q", ai.FinishReasonCancellation.String(), "cancellation")
	}
	// The FinishReason constants must be pairwise distinct (6 values).
	all := []ai.FinishReason{
		ai.FinishReasonStop,
		ai.FinishReasonLength,
		ai.FinishReasonToolCall,
		ai.FinishReasonContentFilter,
		ai.FinishReasonCancellation,
		ai.FinishReasonUnknown,
	}
	for i := range all {
		for j := range all {
			if i != j && all[i] == all[j] {
				t.Errorf("FinishReason constants alias: %q == %q", all[i], all[j])
			}
		}
	}
}

// TestDocGo_AI10Paragraph asserts that doc.go contains the AI-10 paragraph
// referencing FinishReason + Usage after the `package ai` clause. Per AI-10
// spec/design § "Vocabulary anchors" and the AI-07 paragraph-count guard.
func TestDocGo_AI10Paragraph(t *testing.T) {
	data, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("read doc.go: %v", err)
	}
	src := string(data)
	packageIndex := strings.Index(src, "package ai")
	paragraphIndex := strings.Index(src, "AI-10 paragraph")
	if packageIndex < 0 {
		t.Fatal("doc.go missing `package ai` clause")
	}
	if paragraphIndex < 0 {
		t.Fatal("doc.go missing AI-10 paragraph")
	}
	if paragraphIndex < packageIndex {
		t.Error("AI-10 paragraph must appear AFTER `package ai` clause")
	}
	after := src[paragraphIndex:]
	if !strings.Contains(after, "FinishReason") {
		t.Error("AI-10 paragraph must reference FinishReason")
	}
	if !strings.Contains(after, "Usage") {
		t.Error("AI-10 paragraph must reference Usage")
	}
}
