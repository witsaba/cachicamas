package ai_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// =============================================================================
// Constant pins
// =============================================================================

// TestMaxToolCallArgumentsLength_Value pins MaxToolCallArgumentsLength to
// 1<<20 (1 MiB). Per AI-08 spec § D.
func TestMaxToolCallArgumentsLength_Value(t *testing.T) {
	const want = 1 << 20
	if ai.MaxToolCallArgumentsLength != want {
		t.Errorf("MaxToolCallArgumentsLength = %d, want %d", ai.MaxToolCallArgumentsLength, want)
	}
}

// =============================================================================
// Constructor — name validation
// =============================================================================

// TestNewToolCall_Name_Empty verifies ErrEmptyToolCallName when name is "".
func TestNewToolCall_Name_Empty(t *testing.T) {
	args := json.RawMessage(`{}`)
	tc, err := ai.NewToolCall("", args)
	if !errors.Is(err, ai.ErrEmptyToolCallName) {
		t.Errorf("NewToolCall(\"\", %T) error = %v, want ErrEmptyToolCallName", args, err)
	}
	if !isZeroToolCall(tc) {
		t.Errorf("NewToolCall returned non-zero ToolCall on error path")
	}
}

// TestNewToolCall_Name_WhitespaceOnly verifies ErrEmptyToolCallName for
// whitespace-only names.
func TestNewToolCall_Name_WhitespaceOnly(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"ascii_spaces", "   "},
		{"tab_newline", "\n\t"},
		{"nbsp", "\u00A0"},
		{"ideographic_space", "\u3000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := json.RawMessage(`{}`)
			tc, err := ai.NewToolCall(c.input, args)
			if !errors.Is(err, ai.ErrEmptyToolCallName) {
				t.Errorf("NewToolCall(%q, _) error = %v, want ErrEmptyToolCallName", c.input, err)
			}
			if !isZeroToolCall(tc) {
				t.Errorf("NewToolCall(%q, _) returned non-zero on error", c.input)
			}
		})
	}
}

// TestNewToolCall_Name_TooLong verifies the boundary: exactly MaxToolNameLength
// bytes is accepted; MaxToolNameLength+1 is rejected with ErrToolCallNameTooLong.
func TestNewToolCall_Name_TooLong(t *testing.T) {
	t.Run("exactly_max", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxToolNameLength)
		args := json.RawMessage(`{}`)
		tc, err := ai.NewToolCall(in, args)
		if err != nil {
			t.Fatalf("NewToolCall(%d bytes) returned error %v, want nil (boundary must be accepted)", len(in), err)
		}
		if isZeroToolCall(tc) {
			t.Fatalf("NewToolCall(%d bytes) returned zero-value", len(in))
		}
	})
	t.Run("max_plus_one", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxToolNameLength+1)
		args := json.RawMessage(`{}`)
		tc, err := ai.NewToolCall(in, args)
		if !errors.Is(err, ai.ErrToolCallNameTooLong) {
			t.Errorf("NewToolCall(%d bytes) error = %v, want ErrToolCallNameTooLong", len(in), err)
		}
		if !isZeroToolCall(tc) {
			t.Errorf("NewToolCall(%d bytes) returned non-zero on error", len(in))
		}
	})
}

// TestNewToolCall_Name_ControlChars verifies ErrInvalidToolCallName for any
// rune reporting true for unicode.IsControl.
func TestNewToolCall_Name_ControlChars(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"nul", "wea\x00ther"},
		{"lf", "wea\nther"},
		{"tab", "wea\tther"},
		{"cr", "wea\rther"},
		{"ansi_escape", "wea\x1b[31mther"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			args := json.RawMessage(`{}`)
			tc, err := ai.NewToolCall(c.input, args)
			if !errors.Is(err, ai.ErrInvalidToolCallName) {
				t.Errorf("NewToolCall(%q, _) error = %v, want ErrInvalidToolCallName", c.input, err)
			}
			if !isZeroToolCall(tc) {
				t.Errorf("NewToolCall(%q, _) returned non-zero on error", c.input)
			}
		})
	}
}

// =============================================================================
// Constructor — arguments validation
// =============================================================================

// TestNewToolCall_Arguments_Nil verifies ErrMalformedToolCallArguments when
// arguments is nil.
func TestNewToolCall_Arguments_Nil(t *testing.T) {
	tc, err := ai.NewToolCall("get_weather", nil)
	if !errors.Is(err, ai.ErrMalformedToolCallArguments) {
		t.Errorf("NewToolCall(\"get_weather\", nil) error = %v, want ErrMalformedToolCallArguments", err)
	}
	if !isZeroToolCall(tc) {
		t.Errorf("NewToolCall returned non-zero ToolCall on error path")
	}
}

// TestNewToolCall_Arguments_Empty verifies ErrMalformedToolCallArguments when
// arguments is empty RawMessage.
func TestNewToolCall_Arguments_Empty(t *testing.T) {
	tc, err := ai.NewToolCall("get_weather", json.RawMessage(""))
	if !errors.Is(err, ai.ErrMalformedToolCallArguments) {
		t.Errorf("NewToolCall with empty arguments error = %v, want ErrMalformedToolCallArguments", err)
	}
	if !isZeroToolCall(tc) {
		t.Errorf("NewToolCall returned non-zero ToolCall on error path")
	}
}

// TestNewToolCall_Arguments_Malformed verifies ErrMalformedToolCallArguments
// for 4 distinct kinds of malformed JSON.
func TestNewToolCall_Arguments_Malformed(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"plain_text", "not json"},
		{"unterminated_object", "{"},
		{"trailing_comma", `{"a":}`},
		{"unquoted_key", `{a:1}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tc, err := ai.NewToolCall("get_weather", json.RawMessage(c.input))
			if !errors.Is(err, ai.ErrMalformedToolCallArguments) {
				t.Errorf("NewToolCall with args=%q error = %v, want ErrMalformedToolCallArguments", c.input, err)
			}
			if !isZeroToolCall(tc) {
				t.Errorf("NewToolCall returned non-zero ToolCall on error path")
			}
		})
	}
}

// TestNewToolCall_Arguments_TooLong verifies ErrToolCallArgumentsTooLong when
// arguments exceed MaxToolCallArgumentsLength.
func TestNewToolCall_Arguments_TooLong(t *testing.T) {
	// Build exactly MaxToolCallArgumentsLength+1 bytes of valid JSON.
	// Use a simple string of 'a' chars: `"aaaa...` which is valid JSON.
	 oversized := strings.Repeat("a", ai.MaxToolCallArgumentsLength+1)
	args := json.RawMessage(`"` + oversized + `"`)
	tc, err := ai.NewToolCall("get_weather", args)
	if !errors.Is(err, ai.ErrToolCallArgumentsTooLong) {
		t.Errorf("NewToolCall with oversized arguments error = %v, want ErrToolCallArgumentsTooLong", err)
	}
	if !isZeroToolCall(tc) {
		t.Errorf("NewToolCall returned non-zero ToolCall on error path")
	}
}

// TestNewToolCall_Arguments_ExactlyMaxAccepted verifies that arguments at
// exactly MaxToolCallArgumentsLength are accepted.
func TestNewToolCall_Arguments_ExactlyMaxAccepted(t *testing.T) {
	// The raw message bytes must be exactly MaxToolCallArgumentsLength.
	// json.RawMessage("\"...") adds 2 bytes for the surrounding quotes,
	// so the inner string must be 2 bytes shorter to hit the boundary exactly.
	 oversized := strings.Repeat("a", ai.MaxToolCallArgumentsLength-2)
	args := json.RawMessage(`"` + oversized + `"`)
	if len(args) != ai.MaxToolCallArgumentsLength {
		t.Fatalf("test payload len = %d, want %d (accounting for JSON quotes)", len(args), ai.MaxToolCallArgumentsLength)
	}
	tc, err := ai.NewToolCall("get_weather", args)
	if err != nil {
		t.Fatalf("NewToolCall with exactly MaxToolCallArgumentsLength args error = %v, want nil", err)
	}
	if isZeroToolCall(tc) {
		t.Fatalf("NewToolCall returned zero-value")
	}
}

// TestNewToolCall_Arguments_EmptyJSONObject verifies that empty JSON object
// {} is accepted as valid arguments (tool with no parameters).
func TestNewToolCall_Arguments_EmptyJSONObject(t *testing.T) {
	tc, err := ai.NewToolCall("get_weather", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("NewToolCall with {} args error = %v, want nil", err)
	}
	if isZeroToolCall(tc) {
		t.Fatalf("NewToolCall returned zero-value")
	}
}

// =============================================================================
// Constructor — happy path + accessors
// =============================================================================

// TestNewToolCall_HappyPath verifies the canonical happy path.
func TestNewToolCall_HappyPath(t *testing.T) {
	name := "get_weather"
	args := json.RawMessage(`{"city":"Tokyo"}`)
	tc, err := ai.NewToolCall(name, args)
	if err != nil {
		t.Fatalf("NewToolCall happy path error = %v, want nil", err)
	}
	if isZeroToolCall(tc) {
		t.Fatal("NewToolCall happy path returned zero-value")
	}
	if tc.Name() != name {
		t.Errorf("tc.Name() = %q, want %q", tc.Name(), name)
	}
	if !bytesEqual(tc.Arguments(), args) {
		t.Errorf("tc.Arguments() = %q, want %q", tc.Arguments(), args)
	}
}

// TestToolCall_Accessors_RoundTrip verifies Name() and Arguments() return exactly
// what was passed to NewToolCall.
func TestToolCall_Accessors_RoundTrip(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		name := "search_files"
		tc, err := ai.NewToolCall(name, json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("setup error: %v", err)
		}
		if got := tc.Name(); got != name {
			t.Errorf("tc.Name() = %q, want %q", got, name)
		}
	})
	t.Run("arguments", func(t *testing.T) {
		args := json.RawMessage(`{"q":"search term"}`)
		tc, err := ai.NewToolCall("search_files", args)
		if err != nil {
			t.Fatalf("setup error: %v", err)
		}
		if got := tc.Arguments(); !bytesEqual(got, args) {
			t.Errorf("tc.Arguments() = %q, want %q", got, args)
		}
	})
}

// =============================================================================
// Validate
// =============================================================================

// TestToolCall_ZeroValue_FailsValidate verifies that a literal ToolCall{}
// yields Validate() that returns ErrEmptyToolCallName.
func TestToolCall_ZeroValue_FailsValidate(t *testing.T) {
	var zero ai.ToolCall
	if err := zero.Validate(); !errors.Is(err, ai.ErrEmptyToolCallName) {
		t.Errorf("ToolCall{}.Validate() = %v, want ErrEmptyToolCallName", err)
	}
}

// TestToolCall_Validate_AcceptsValid verifies Validate() returns nil for a
// value produced by NewToolCall without error.
func TestToolCall_Validate_AcceptsValid(t *testing.T) {
	tc, err := ai.NewToolCall("get_weather", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("setup error: %v", err)
	}
	if err := tc.Validate(); err != nil {
		t.Errorf("tc.Validate() on valid ToolCall = %v, want nil", err)
	}
}

// TestValidateSelfConsistency verifies that Validate() on a zero-value returns
// the same sentinel as NewToolCall("", nil).
func TestValidateSelfConsistency(t *testing.T) {
	_, errConstructor := ai.NewToolCall("", nil)
	var zero ai.ToolCall
	if err := zero.Validate(); !errors.Is(err, errConstructor) {
		t.Errorf("Validate() self-consistency: zero.Validate() = %v, NewToolCall(\"\", nil) = %v", zero.Validate(), errConstructor)
	}
}

// =============================================================================
// Sentinel distinctness
// =============================================================================

// TestToolCall_Sentinels_Distinct verifies the 5 ToolCall sentinels are pairwise
// distinct under errors.Is.
func TestToolCall_Sentinels_Distinct(t *testing.T) {
	sentinels := map[error]string{
		ai.ErrEmptyToolCallName:         "ErrEmptyToolCallName",
		ai.ErrToolCallNameTooLong:        "ErrToolCallNameTooLong",
		ai.ErrInvalidToolCallName:       "ErrInvalidToolCallName",
		ai.ErrMalformedToolCallArguments: "ErrMalformedToolCallArguments",
		ai.ErrToolCallArgumentsTooLong:   "ErrToolCallArgumentsTooLong",
	}
	if len(sentinels) != 5 {
		t.Errorf("AI-08 spec § E requires exactly 5 ToolCall sentinels, found %d", len(sentinels))
	}

	// Pairwise cross-check.
	pairs := [][2]error{
		{ai.ErrEmptyToolCallName, ai.ErrToolCallNameTooLong},
		{ai.ErrEmptyToolCallName, ai.ErrInvalidToolCallName},
		{ai.ErrEmptyToolCallName, ai.ErrMalformedToolCallArguments},
		{ai.ErrEmptyToolCallName, ai.ErrToolCallArgumentsTooLong},
		{ai.ErrToolCallNameTooLong, ai.ErrInvalidToolCallName},
		{ai.ErrToolCallNameTooLong, ai.ErrMalformedToolCallArguments},
		{ai.ErrToolCallNameTooLong, ai.ErrToolCallArgumentsTooLong},
		{ai.ErrInvalidToolCallName, ai.ErrMalformedToolCallArguments},
		{ai.ErrInvalidToolCallName, ai.ErrToolCallArgumentsTooLong},
		{ai.ErrMalformedToolCallArguments, ai.ErrToolCallArgumentsTooLong},
	}
	for _, p := range pairs {
		if errors.Is(p[0], p[1]) {
			t.Errorf("%v must NOT alias %v via errors.Is", p[0], p[1])
		}
	}

	// Pin the ai: prefix convention.
	for err, name := range sentinels {
		if msg := err.Error(); !strings.HasPrefix(msg, "ai: ") {
			t.Errorf("%s.Error() = %q, want prefix %q", name, msg, "ai: ")
		}
	}

	// Self-distinctness.
	for err, name := range sentinels {
		if !errors.Is(err, err) {
			t.Errorf("%s must satisfy errors.Is(err, err)", name)
		}
	}
}

// TestToolCall_Sentinels_ExportedAndTyped verifies each sentinel is non-nil and has
// a non-empty message.
func TestToolCall_Sentinels_ExportedAndTyped(t *testing.T) {
	cases := map[error]string{
		ai.ErrEmptyToolCallName:         "ErrEmptyToolCallName",
		ai.ErrToolCallNameTooLong:        "ErrToolCallNameTooLong",
		ai.ErrInvalidToolCallName:       "ErrInvalidToolCallName",
		ai.ErrMalformedToolCallArguments: "ErrMalformedToolCallArguments",
		ai.ErrToolCallArgumentsTooLong:   "ErrToolCallArgumentsTooLong",
	}
	for err, name := range cases {
		if err == nil {
			t.Errorf("%s must not be nil", name)
			continue
		}
		if err.Error() == "" {
			t.Errorf("%s.Error() must be non-empty", name)
		}
		if !errors.Is(err, err) {
			t.Errorf("%s must be compatible with errors.Is", name)
		}
	}
}

// =============================================================================
// Boundary pin — ToolCall is NOT a ContentPart
// =============================================================================

// TestToolCall_NotContentPart pins the boundary invariant: ai.ToolCall
// MUST NOT have a Kind() method (it is NOT a ContentPart — only the wrapper
// toolCallPart is).
func TestToolCall_NotContentPart(t *testing.T) {
	tcType := reflect.TypeOf(ai.ToolCall{})
	_, ok := tcType.MethodByName("Kind")
	if ok {
		t.Errorf("ai.ToolCall has a Kind() method — ToolCall itself is NOT a ContentPart; only the toolCallPart wrapper is. " +
			"Per AI-08 design (obs #2099) § Decision: wrapper pattern.")
	}
}

// =============================================================================
// Test helpers
// =============================================================================

// isZeroToolCall reports whether tc is the zero value.
func isZeroToolCall(tc ai.ToolCall) bool {
	return tc.Name() == "" && len(tc.Arguments()) == 0
}

// bytesEqual compares two json.RawMessage values for equality.
func bytesEqual(a, b json.RawMessage) bool {
	return string(a) == string(b)
}

// mustToolCall wraps NewToolCall for slice-literal test contexts. Errors are
// fatal since test inputs are hardcoded valid combinations.
func mustToolCall(t *testing.T, name string, args json.RawMessage) ai.ToolCall {
	t.Helper()
	tc, err := ai.NewToolCall(name, args)
	if err != nil {
		t.Fatalf("mustToolCall(%q, %T) returned error %v, want nil", name, args, err)
	}
	return tc
}
