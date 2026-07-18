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

// TestMaxToolResultCallIDLength_Value pins MaxToolResultCallIDLength to 512.
func TestMaxToolResultCallIDLength_Value(t *testing.T) {
	const want = 512
	if ai.MaxToolResultCallIDLength != want {
		t.Errorf("MaxToolResultCallIDLength = %d, want %d", ai.MaxToolResultCallIDLength, want)
	}
}

// TestMaxToolResultContentLength_Value pins MaxToolResultContentLength to
// 10 * (1<<20).
func TestMaxToolResultContentLength_Value(t *testing.T) {
	const want = 10 * (1 << 20)
	if ai.MaxToolResultContentLength != want {
		t.Errorf("MaxToolResultContentLength = %d, want %d", ai.MaxToolResultContentLength, want)
	}
}

// =============================================================================
// Constructor — callID validation
// =============================================================================

// TestNewToolResult_CallID_Empty verifies ErrEmptyToolResultCallID when
// callID is "".
func TestNewToolResult_CallID_Empty(t *testing.T) {
	content := json.RawMessage(`{}`)
	tr, err := ai.NewToolResult("", content)
	if !errors.Is(err, ai.ErrEmptyToolResultCallID) {
		t.Errorf("NewToolResult(\"\", %T) error = %v, want ErrEmptyToolResultCallID", content, err)
	}
	if !isZeroToolResult(tr) {
		t.Errorf("NewToolResult returned non-zero ToolResult on error path")
	}
}

// TestNewToolResult_CallID_WhitespaceOnly verifies ErrEmptyToolResultCallID
// for whitespace-only callIDs.
func TestNewToolResult_CallID_WhitespaceOnly(t *testing.T) {
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
			content := json.RawMessage(`{}`)
			tr, err := ai.NewToolResult(c.input, content)
			if !errors.Is(err, ai.ErrEmptyToolResultCallID) {
				t.Errorf("NewToolResult(%q, _) error = %v, want ErrEmptyToolResultCallID", c.input, err)
			}
			if !isZeroToolResult(tr) {
				t.Errorf("NewToolResult(%q, _) returned non-zero on error", c.input)
			}
		})
	}
}

// TestNewToolResult_CallID_TooLong verifies the boundary: exactly
// MaxToolResultCallIDLength bytes is accepted; +1 is rejected with
// ErrToolResultCallIDTooLong.
func TestNewToolResult_CallID_TooLong(t *testing.T) {
	t.Run("exactly_max", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxToolResultCallIDLength)
		content := json.RawMessage(`{}`)
		tr, err := ai.NewToolResult(in, content)
		if err != nil {
			t.Fatalf("NewToolResult(%d bytes) returned error %v, want nil", len(in), err)
		}
		if isZeroToolResult(tr) {
			t.Fatalf("NewToolResult(%d bytes) returned zero-value", len(in))
		}
	})
	t.Run("max_plus_one", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxToolResultCallIDLength+1)
		content := json.RawMessage(`{}`)
		tr, err := ai.NewToolResult(in, content)
		if !errors.Is(err, ai.ErrToolResultCallIDTooLong) {
			t.Errorf("NewToolResult(%d bytes) error = %v, want ErrToolResultCallIDTooLong", len(in), err)
		}
		if !isZeroToolResult(tr) {
			t.Errorf("NewToolResult(%d bytes) returned non-zero on error", len(in))
		}
	})
}

// =============================================================================
// Constructor — content validation
// =============================================================================

// TestNewToolResult_Content_Nil verifies ErrMalformedToolResultContent when
// content is nil.
func TestNewToolResult_Content_Nil(t *testing.T) {
	tr, err := ai.NewToolResult("call-42", nil)
	if !errors.Is(err, ai.ErrMalformedToolResultContent) {
		t.Errorf("NewToolResult(\"call-42\", nil) error = %v, want ErrMalformedToolResultContent", err)
	}
	if !isZeroToolResult(tr) {
		t.Errorf("NewToolResult returned non-zero ToolResult on error path")
	}
}

// TestNewToolResult_Content_Empty verifies ErrMalformedToolResultContent when
// content is empty RawMessage.
func TestNewToolResult_Content_Empty(t *testing.T) {
	tr, err := ai.NewToolResult("call-42", json.RawMessage(""))
	if !errors.Is(err, ai.ErrMalformedToolResultContent) {
		t.Errorf("NewToolResult with empty content error = %v, want ErrMalformedToolResultContent", err)
	}
	if !isZeroToolResult(tr) {
		t.Errorf("NewToolResult returned non-zero ToolResult on error path")
	}
}

// TestNewToolResult_Content_Malformed verifies ErrMalformedToolResultContent
// for malformed JSON content.
func TestNewToolResult_Content_Malformed(t *testing.T) {
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
			tr, err := ai.NewToolResult("call-42", json.RawMessage(c.input))
			if !errors.Is(err, ai.ErrMalformedToolResultContent) {
				t.Errorf("NewToolResult with content=%q error = %v, want ErrMalformedToolResultContent", c.input, err)
			}
			if !isZeroToolResult(tr) {
				t.Errorf("NewToolResult returned non-zero ToolResult on error path")
			}
		})
	}
}

// TestNewToolResult_Content_TooLong verifies ErrToolResultContentTooLong when
// content exceeds MaxToolResultContentLength.
func TestNewToolResult_Content_TooLong(t *testing.T) {
	 oversized := strings.Repeat("a", ai.MaxToolResultContentLength+1)
	content := json.RawMessage(`"` + oversized + `"`)
	tr, err := ai.NewToolResult("call-42", content)
	if !errors.Is(err, ai.ErrToolResultContentTooLong) {
		t.Errorf("NewToolResult with oversized content error = %v, want ErrToolResultContentTooLong", err)
	}
	if !isZeroToolResult(tr) {
		t.Errorf("NewToolResult returned non-zero ToolResult on error path")
	}
}

// TestNewToolResult_Content_ExactlyMaxAccepted verifies that content at
// exactly MaxToolResultContentLength is accepted.
func TestNewToolResult_Content_ExactlyMaxAccepted(t *testing.T) {
	 oversized := strings.Repeat("a", ai.MaxToolResultContentLength-2)
	content := json.RawMessage(`"` + oversized + `"`)
	if len(content) != ai.MaxToolResultContentLength {
		t.Fatalf("test payload len = %d, want %d", len(content), ai.MaxToolResultContentLength)
	}
	tr, err := ai.NewToolResult("call-42", content)
	if err != nil {
		t.Fatalf("NewToolResult with exactly MaxToolResultContentLength content error = %v, want nil", err)
	}
	if isZeroToolResult(tr) {
		t.Fatalf("NewToolResult returned zero-value")
	}
}

// TestNewToolResult_Content_EmptyJSONObject verifies that empty JSON object {}
// is accepted as valid content.
func TestNewToolResult_Content_EmptyJSONObject(t *testing.T) {
	tr, err := ai.NewToolResult("call-42", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("NewToolResult with {} content error = %v, want nil", err)
	}
	if isZeroToolResult(tr) {
		t.Fatalf("NewToolResult returned zero-value")
	}
}

// TestNewToolResult_Content_ErrorJSON verifies that {"error":"file not found"}
// is accepted as valid content (error results are data, not status flags).
func TestNewToolResult_Content_ErrorJSON(t *testing.T) {
	tr, err := ai.NewToolResult("call-42", json.RawMessage(`{"error":"file not found"}`))
	if err != nil {
		t.Fatalf("NewToolResult with error content error = %v, want nil", err)
	}
	if isZeroToolResult(tr) {
		t.Fatalf("NewToolResult returned zero-value")
	}
}

// =============================================================================
// Constructor — happy path + accessors
// =============================================================================

// TestNewToolResult_HappyPath verifies the canonical happy path.
func TestNewToolResult_HappyPath(t *testing.T) {
	callID := "call-42"
	content := json.RawMessage(`{"temp":22}`)
	tr, err := ai.NewToolResult(callID, content)
	if err != nil {
		t.Fatalf("NewToolResult happy path error = %v, want nil", err)
	}
	if isZeroToolResult(tr) {
		t.Fatal("NewToolResult happy path returned zero-value")
	}
	if tr.CallID() != callID {
		t.Errorf("tr.CallID() = %q, want %q", tr.CallID(), callID)
	}
	if !bytesEqual(tr.Content(), content) {
		t.Errorf("tr.Content() = %q, want %q", tr.Content(), content)
	}
}

// TestToolResult_Accessors_RoundTrip verifies CallID() and Content() return
// exactly what was passed to NewToolResult.
func TestToolResult_Accessors_RoundTrip(t *testing.T) {
	t.Run("callID", func(t *testing.T) {
		callID := "call-42"
		tr, err := ai.NewToolResult(callID, json.RawMessage(`{}`))
		if err != nil {
			t.Fatalf("setup error: %v", err)
		}
		if got := tr.CallID(); got != callID {
			t.Errorf("tr.CallID() = %q, want %q", got, callID)
		}
	})
	t.Run("content", func(t *testing.T) {
		content := json.RawMessage(`{"result":"ok"}`)
		tr, err := ai.NewToolResult("call-42", content)
		if err != nil {
			t.Fatalf("setup error: %v", err)
		}
		if got := tr.Content(); !bytesEqual(got, content) {
			t.Errorf("tr.Content() = %q, want %q", got, content)
		}
	})
}

// =============================================================================
// Validate
// =============================================================================

// TestToolResult_ZeroValue_FailsValidate verifies that a literal ToolResult{}
// yields Validate() that returns ErrEmptyToolResultCallID.
func TestToolResult_ZeroValue_FailsValidate(t *testing.T) {
	var zero ai.ToolResult
	if err := zero.Validate(); !errors.Is(err, ai.ErrEmptyToolResultCallID) {
		t.Errorf("ToolResult{}.Validate() = %v, want ErrEmptyToolResultCallID", err)
	}
}

// TestToolResult_Validate_AcceptsValid verifies Validate() returns nil for a
// value produced by NewToolResult without error.
func TestToolResult_Validate_AcceptsValid(t *testing.T) {
	tr, err := ai.NewToolResult("call-42", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("setup error: %v", err)
	}
	if err := tr.Validate(); err != nil {
		t.Errorf("tr.Validate() on valid ToolResult = %v, want nil", err)
	}
}

// TestToolResult_ValidateSelfConsistency verifies that Validate() on a zero-value
// returns the same sentinel as NewToolResult("", nil).
func TestToolResult_ValidateSelfConsistency(t *testing.T) {
	_, errConstructor := ai.NewToolResult("", nil)
	var zero ai.ToolResult
	if err := zero.Validate(); !errors.Is(err, errConstructor) {
		t.Errorf("Validate() self-consistency: zero.Validate() = %v, NewToolResult(\"\", nil) = %v", zero.Validate(), errConstructor)
	}
}

// =============================================================================
// Sentinel distinctness
// =============================================================================

// TestToolResult_Sentinels_Distinct verifies the 4 ToolResult sentinels are
// pairwise distinct under errors.Is.
func TestToolResult_Sentinels_Distinct(t *testing.T) {
	sentinels := map[error]string{
		ai.ErrEmptyToolResultCallID:       "ErrEmptyToolResultCallID",
		ai.ErrToolResultCallIDTooLong:     "ErrToolResultCallIDTooLong",
		ai.ErrMalformedToolResultContent:   "ErrMalformedToolResultContent",
		ai.ErrToolResultContentTooLong:    "ErrToolResultContentTooLong",
	}
	if len(sentinels) != 4 {
		t.Errorf("AI-08 spec § E requires exactly 4 ToolResult sentinels, found %d", len(sentinels))
	}

	pairs := [][2]error{
		{ai.ErrEmptyToolResultCallID, ai.ErrToolResultCallIDTooLong},
		{ai.ErrEmptyToolResultCallID, ai.ErrMalformedToolResultContent},
		{ai.ErrEmptyToolResultCallID, ai.ErrToolResultContentTooLong},
		{ai.ErrToolResultCallIDTooLong, ai.ErrMalformedToolResultContent},
		{ai.ErrToolResultCallIDTooLong, ai.ErrToolResultContentTooLong},
		{ai.ErrMalformedToolResultContent, ai.ErrToolResultContentTooLong},
	}
	for _, p := range pairs {
		if errors.Is(p[0], p[1]) {
			t.Errorf("%v must NOT alias %v via errors.Is", p[0], p[1])
		}
	}

	for err, name := range sentinels {
		if msg := err.Error(); !strings.HasPrefix(msg, "ai: ") {
			t.Errorf("%s.Error() = %q, want prefix %q", name, msg, "ai: ")
		}
	}

	for err, name := range sentinels {
		if !errors.Is(err, err) {
			t.Errorf("%s must satisfy errors.Is(err, err)", name)
		}
	}
}

// TestToolResult_Sentinels_ExportedAndTyped verifies each sentinel is non-nil
// and has a non-empty message.
func TestToolResult_Sentinels_ExportedAndTyped(t *testing.T) {
	cases := map[error]string{
		ai.ErrEmptyToolResultCallID:     "ErrEmptyToolResultCallID",
		ai.ErrToolResultCallIDTooLong:    "ErrToolResultCallIDTooLong",
		ai.ErrMalformedToolResultContent: "ErrMalformedToolResultContent",
		ai.ErrToolResultContentTooLong:   "ErrToolResultContentTooLong",
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
// Boundary pin — ToolResult is NOT a ContentPart
// =============================================================================

// TestToolResult_NotContentPart pins the boundary invariant: ai.ToolResult
// MUST NOT have a Kind() method.
func TestToolResult_NotContentPart(t *testing.T) {
	trType := reflect.TypeOf(ai.ToolResult{})
	_, ok := trType.MethodByName("Kind")
	if ok {
		t.Errorf("ai.ToolResult has a Kind() method — ToolResult itself is NOT a ContentPart; only the toolResultPart wrapper is.")
	}
}

// =============================================================================
// Test helpers
// =============================================================================

// isZeroToolResult reports whether tr is the zero value.
func isZeroToolResult(tr ai.ToolResult) bool {
	return tr.CallID() == "" && len(tr.Content()) == 0
}

// mustToolResult wraps NewToolResult for slice-literal test contexts.
func mustToolResult(t *testing.T, callID string, content json.RawMessage) ai.ToolResult {
	t.Helper()
	tr, err := ai.NewToolResult(callID, content)
	if err != nil {
		t.Fatalf("mustToolResult(%q, %T) returned error %v, want nil", callID, content, err)
	}
	return tr
}
