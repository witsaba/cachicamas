package ai_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// =============================================================================
// Compile-time / boundary pins
// =============================================================================
//
// ToolDeclaration is NOT a ContentPart. Per AI-07 spec § A req 1, the
// absence of a `Kind() Kind` method on ToolDeclaration is the contract —
// any future maintainer who adds one (intentionally or by drift) would
// silently make ToolDeclaration a ContentPart, violating Decision 1 of
// the AI-07 proposal. This file deliberately does NOT contain a
// `var _ ai.ContentPart = ai.ToolDeclaration{}` line. The negative
// assertion is the verified contract.
//
// Similarly, the file does NOT reference MarshalJSON, UnmarshalJSON, Run,
// Execute, Invoke, Callback, Handler, or Func on ToolDeclaration. The
// TestToolDeclaration_NoMarshalOrExecute runtime check below reflects on
// the type to assert that none of those methods exist.

// =============================================================================
// Constant pins
// =============================================================================

// TestMaxToolNameLength_Value pins MaxToolNameLength to 64 bytes. Per
// AI-07 spec § A req — the constant MUST be 64 to match OpenAI tool-name
// conventions (see proposal § Decision 3 § Validation scope).
func TestMaxToolNameLength_Value(t *testing.T) {
	const want = 64
	if ai.MaxToolNameLength != want {
		t.Errorf("MaxToolNameLength = %d, want %d", ai.MaxToolNameLength, want)
	}
}

// TestMaxToolDescriptionLength_Value pins MaxToolDescriptionLength to
// 1024 bytes (1 KiB). Per AI-07 spec § A req — the constant MUST be
// 1 KiB to match the conservative cap across supported vendors.
func TestMaxToolDescriptionLength_Value(t *testing.T) {
	const want = 1 << 10 // 1 KiB = 1024 bytes
	if ai.MaxToolDescriptionLength != want {
		t.Errorf("MaxToolDescriptionLength = %d, want %d", ai.MaxToolDescriptionLength, want)
	}
}

// =============================================================================
// Constructor — name validation
// =============================================================================

// TestNewToolDeclaration_Name_Empty verifies ErrEmptyToolName when the
// name is the empty string. Per AI-07 spec § A req 1 scenario "Empty
// name is rejected".
func TestNewToolDeclaration_Name_Empty(t *testing.T) {
	td, err := ai.NewToolDeclaration("", "valid description", validObjectSchema())
	if !errors.Is(err, ai.ErrEmptyToolName) {
		t.Errorf("NewToolDeclaration(\"\", ...) error = %v, want ErrEmptyToolName", err)
	}
	if (td != ai.ToolDeclaration{}) {
		t.Errorf("NewToolDeclaration(\"\", ...) returned non-zero ToolDeclaration %+v on error", td)
	}
}

// TestNewToolDeclaration_Name_WhitespaceOnly verifies the four canonical
// whitespace-only name variants are rejected with ErrEmptyToolName.
// Per AI-07 spec § A req scenario "Whitespace-only name is rejected".
// Reuses the same-package isAllWhitespace semantics from text.go via the
// public constructor.
func TestNewToolDeclaration_Name_WhitespaceOnly(t *testing.T) {
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
			td, err := ai.NewToolDeclaration(c.input, "valid description", validObjectSchema())
			if !errors.Is(err, ai.ErrEmptyToolName) {
				t.Errorf("NewToolDeclaration(%q) error = %v, want ErrEmptyToolName", c.input, err)
			}
			if (td != ai.ToolDeclaration{}) {
				t.Errorf("NewToolDeclaration(%q) returned non-zero ToolDeclaration on error", c.input)
			}
		})
	}
}

// TestNewToolDeclaration_Name_TooLong verifies the boundary pair:
// MaxToolNameLength bytes is accepted, MaxToolNameLength+1 is rejected
// with ErrToolNameTooLong. Per AI-07 spec § A req scenario "Name at
// MaxToolNameLength is accepted; MaxToolNameLength+1 is rejected".
func TestNewToolDeclaration_Name_TooLong(t *testing.T) {
	t.Run("exactly_max", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxToolNameLength)
		td, err := ai.NewToolDeclaration(in, "valid description", validObjectSchema())
		if err != nil {
			t.Fatalf("NewToolDeclaration(%d bytes) returned error %v, want nil (boundary must be accepted)", len(in), err)
		}
		if (td == ai.ToolDeclaration{}) {
			t.Fatalf("NewToolDeclaration(%d bytes) returned zero-value ToolDeclaration", len(in))
		}
	})
	t.Run("max_plus_one", func(t *testing.T) {
		in := strings.Repeat("a", ai.MaxToolNameLength+1)
		td, err := ai.NewToolDeclaration(in, "valid description", validObjectSchema())
		if !errors.Is(err, ai.ErrToolNameTooLong) {
			t.Errorf("NewToolDeclaration(%d bytes) error = %v, want ErrToolNameTooLong", len(in), err)
		}
		if (td != ai.ToolDeclaration{}) {
			t.Errorf("NewToolDeclaration(%d bytes) returned non-zero ToolDeclaration on error", len(in))
		}
	})
}

// TestNewToolDeclaration_Name_ControlChars verifies ErrInvalidToolName
// for any rune reporting true for unicode.IsControl. Per AI-07 spec § A
// req scenario "Name containing a control character is rejected".
// The 5 cases cover NUL, LF, TAB, CR, and ANSI escape.
func TestNewToolDeclaration_Name_ControlChars(t *testing.T) {
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
			td, err := ai.NewToolDeclaration(c.input, "valid description", validObjectSchema())
			if !errors.Is(err, ai.ErrInvalidToolName) {
				t.Errorf("NewToolDeclaration(%q) error = %v, want ErrInvalidToolName", c.input, err)
			}
			if (td != ai.ToolDeclaration{}) {
				t.Errorf("NewToolDeclaration(%q) returned non-zero ToolDeclaration on error", c.input)
			}
		})
	}
}

// TestNewToolDeclaration_Name_NonASCII verifies non-ASCII names within
// the byte budget are accepted. Per AI-07 spec § A req — CJK, emoji,
// and accented Latin must pass when ≤ MaxToolNameLength bytes and
// contain no control characters.
func TestNewToolDeclaration_Name_NonASCII(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"cjk", "天気予報"},
		{"emoji", "get_weather_☀️"},
		{"accented", "get_café_info"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if len(c.input) > ai.MaxToolNameLength {
				t.Skipf("test input %q is %d bytes, exceeds MaxToolNameLength=%d", c.input, len(c.input), ai.MaxToolNameLength)
			}
			td, err := ai.NewToolDeclaration(c.input, "valid description", validObjectSchema())
			if err != nil {
				t.Errorf("NewToolDeclaration(%q) error = %v, want nil (non-ASCII must be accepted if ≤ MaxToolNameLength bytes and no control chars)", c.input, err)
			}
			if (td == ai.ToolDeclaration{}) {
				t.Errorf("NewToolDeclaration(%q) returned zero-value ToolDeclaration", c.input)
			}
			if td.Name() != c.input {
				t.Errorf("NewToolDeclaration(%q).Name() = %q, want %q", c.input, td.Name(), c.input)
			}
		})
	}
}

// =============================================================================
// Constructor — description validation
// =============================================================================

// TestNewToolDeclaration_Description_Empty verifies ErrEmptyToolDescription
// when the description is the empty string. Per AI-07 spec § A req
// scenario "Empty description is rejected".
func TestNewToolDeclaration_Description_Empty(t *testing.T) {
	td, err := ai.NewToolDeclaration("valid_name", "", validObjectSchema())
	if !errors.Is(err, ai.ErrEmptyToolDescription) {
		t.Errorf("NewToolDeclaration(\"valid_name\", \"\", ...) error = %v, want ErrEmptyToolDescription", err)
	}
	if (td != ai.ToolDeclaration{}) {
		t.Errorf("NewToolDeclaration returned non-zero ToolDeclaration on error")
	}
}

// TestNewToolDeclaration_Description_WhitespaceOnly verifies
// ErrEmptyToolDescription for whitespace-only descriptions. Per AI-07
// spec § A req — uses isAllWhitespace from text.go.
func TestNewToolDeclaration_Description_WhitespaceOnly(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"ascii_spaces", "   "},
		{"newlines_tabs", "\n\t\r"},
		{"nbsp", "\u00A0"},
		{"ideographic_space", "\u3000"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			td, err := ai.NewToolDeclaration("valid_name", c.input, validObjectSchema())
			if !errors.Is(err, ai.ErrEmptyToolDescription) {
				t.Errorf("NewToolDeclaration(%q) error = %v, want ErrEmptyToolDescription", c.input, err)
			}
			if (td != ai.ToolDeclaration{}) {
				t.Errorf("NewToolDeclaration returned non-zero ToolDeclaration on error")
			}
		})
	}
}

// TestNewToolDeclaration_Description_TooLong verifies the boundary pair
// for descriptions: exactly MaxToolDescriptionLength bytes accepted,
// MaxToolDescriptionLength+1 rejected with ErrToolDescriptionTooLong.
// Per AI-07 spec § A req scenario "Description at MaxToolDescriptionLength
// is accepted; +1 is rejected".
func TestNewToolDeclaration_Description_TooLong(t *testing.T) {
	t.Run("exactly_max", func(t *testing.T) {
		in := strings.Repeat("d", ai.MaxToolDescriptionLength)
		td, err := ai.NewToolDeclaration("valid_name", in, validObjectSchema())
		if err != nil {
			t.Fatalf("NewToolDeclaration(%d bytes) error = %v, want nil (boundary must be accepted)", len(in), err)
		}
		if (td == ai.ToolDeclaration{}) {
			t.Fatalf("NewToolDeclaration(%d bytes) returned zero-value ToolDeclaration", len(in))
		}
	})
	t.Run("max_plus_one", func(t *testing.T) {
		in := strings.Repeat("d", ai.MaxToolDescriptionLength+1)
		td, err := ai.NewToolDeclaration("valid_name", in, validObjectSchema())
		if !errors.Is(err, ai.ErrToolDescriptionTooLong) {
			t.Errorf("NewToolDeclaration(%d bytes) error = %v, want ErrToolDescriptionTooLong", len(in), err)
		}
		if (td != ai.ToolDeclaration{}) {
			t.Errorf("NewToolDeclaration returned non-zero ToolDeclaration on error")
		}
	})
}

// =============================================================================
// Constructor — schema validation
// =============================================================================

// TestNewToolDeclaration_Schema_Nil verifies ErrInvalidToolSchema when
// the schema is the nil RawMessage. Per AI-07 spec § A req scenario
// "Schema that is not parseable JSON is rejected".
func TestNewToolDeclaration_Schema_Nil(t *testing.T) {
	td, err := ai.NewToolDeclaration("valid_name", "valid description", nil)
	if !errors.Is(err, ai.ErrInvalidToolSchema) {
		t.Errorf("NewToolDeclaration with nil schema error = %v, want ErrInvalidToolSchema", err)
	}
	if (td != ai.ToolDeclaration{}) {
		t.Errorf("NewToolDeclaration with nil schema returned non-zero ToolDeclaration on error")
	}
}

// TestNewToolDeclaration_Schema_Empty verifies ErrInvalidToolSchema when
// the schema is the empty RawMessage. Per AI-07 spec § A req — empty
// RawMessage is treated as malformed.
func TestNewToolDeclaration_Schema_Empty(t *testing.T) {
	td, err := ai.NewToolDeclaration("valid_name", "valid description", json.RawMessage(""))
	if !errors.Is(err, ai.ErrInvalidToolSchema) {
		t.Errorf("NewToolDeclaration with empty schema error = %v, want ErrInvalidToolSchema", err)
	}
	if (td != ai.ToolDeclaration{}) {
		t.Errorf("NewToolDeclaration with empty schema returned non-zero ToolDeclaration on error")
	}
}

// TestNewToolDeclaration_Schema_Malformed verifies ErrInvalidToolSchema
// for 4 distinct kinds of malformed JSON. Per AI-07 spec § A req scenario
// "Schema that is not parseable JSON is rejected".
func TestNewToolDeclaration_Schema_Malformed(t *testing.T) {
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
			td, err := ai.NewToolDeclaration("valid_name", "valid description", json.RawMessage(c.input))
			if !errors.Is(err, ai.ErrInvalidToolSchema) {
				t.Errorf("NewToolDeclaration with schema=%q error = %v, want ErrInvalidToolSchema", c.input, err)
			}
			if (td != ai.ToolDeclaration{}) {
				t.Errorf("NewToolDeclaration with malformed schema returned non-zero ToolDeclaration on error")
			}
		})
	}
}

// TestNewToolDeclaration_Schema_MissingType verifies ErrMissingSchemaType
// when the parsed top-level object has no "type" key. Per AI-07 spec § A
// req scenario "Schema without a top-level type is rejected".
func TestNewToolDeclaration_Schema_MissingType(t *testing.T) {
	cases := []string{
		`{}`,
		`{"properties":{}}`,
	}
	for _, in := range cases {
		t.Run(in, func(t *testing.T) {
			td, err := ai.NewToolDeclaration("valid_name", "valid description", json.RawMessage(in))
			if !errors.Is(err, ai.ErrMissingSchemaType) {
				t.Errorf("NewToolDeclaration with schema=%q error = %v, want ErrMissingSchemaType", in, err)
			}
			if (td != ai.ToolDeclaration{}) {
				t.Errorf("NewToolDeclaration with missing-type schema returned non-zero ToolDeclaration on error")
			}
		})
	}
}

// TestNewToolDeclaration_Schema_UnsupportedType verifies ErrUnsupportedSchemaType
// for top-level types other than "object". Per AI-07 spec § A req scenario
// "Schema with a top-level type other than object is rejected".
func TestNewToolDeclaration_Schema_UnsupportedType(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"array", `{"type":"array"}`},
		{"string", `{"type":"string"}`},
		{"integer", `{"type":"integer"}`},
		{"boolean", `{"type":"boolean"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			td, err := ai.NewToolDeclaration("valid_name", "valid description", json.RawMessage(c.input))
			if !errors.Is(err, ai.ErrUnsupportedSchemaType) {
				t.Errorf("NewToolDeclaration with schema=%q error = %v, want ErrUnsupportedSchemaType", c.input, err)
			}
			if (td != ai.ToolDeclaration{}) {
				t.Errorf("NewToolDeclaration with unsupported-type schema returned non-zero ToolDeclaration on error")
			}
		})
	}
}

// TestNewToolDeclaration_Schema_SupportedObject verifies the happy path
// for schema validation: top-level type:"object" with a populated
// properties block is accepted. Per AI-07 spec § A req scenario "Schema
// with only top-level type object is accepted".
func TestNewToolDeclaration_Schema_SupportedObject(t *testing.T) {
	schema := json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`)
	td, err := ai.NewToolDeclaration("valid_name", "valid description", schema)
	if err != nil {
		t.Fatalf("NewToolDeclaration with object schema error = %v, want nil", err)
	}
	if (td == ai.ToolDeclaration{}) {
		t.Fatal("NewToolDeclaration with object schema returned zero-value ToolDeclaration")
	}
	if !bytes.Equal(td.Schema(), []byte(schema)) {
		t.Errorf("td.Schema() = %q, want %q (raw bytes preserved)", td.Schema(), schema)
	}
}

// TestNewToolDeclaration_Schema_DenyListKeywords verifies ErrUnsupportedSchemaFeature
// for each of the 12 deny-listed top-level JSON Schema keywords. Per
// AI-07 spec § A req scenario "Schema with each deny-listed top-level
// keyword is rejected". Each keyword is asserted via errors.Is AND
// strings.Contains on the error message (per AI-07 OI-3 — map iteration
// order is non-deterministic).
func TestNewToolDeclaration_Schema_DenyListKeywords(t *testing.T) {
	cases := []struct {
		name    string
		keyword string
	}{
		{"ref", "$ref"},
		{"schema", "$schema"},
		{"id", "$id"},
		{"oneOf", "oneOf"},
		{"anyOf", "anyOf"},
		{"allOf", "allOf"},
		{"not", "not"},
		{"patternProperties", "patternProperties"},
		{"dependencies", "dependencies"},
		{"propertyNames", "propertyNames"},
		{"if", "if"},
		{"then", "then"},
		{"else", "else"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			schema := json.RawMessage(`{"type":"object","` + c.keyword + `":{}}`)
			td, err := ai.NewToolDeclaration("valid_name", "valid description", schema)
			if !errors.Is(err, ai.ErrUnsupportedSchemaFeature) {
				t.Fatalf("NewToolDeclaration with %q in schema error = %v, want ErrUnsupportedSchemaFeature", c.keyword, err)
			}
			if !strings.Contains(err.Error(), c.keyword) {
				t.Errorf("ErrUnsupportedSchemaFeature message = %q, want substring %q", err.Error(), c.keyword)
			}
			if (td != ai.ToolDeclaration{}) {
				t.Errorf("NewToolDeclaration with deny-listed keyword returned non-zero ToolDeclaration on error")
			}
		})
	}
}

// =============================================================================
// Constructor — happy path + accessors
// =============================================================================

// TestNewToolDeclaration_HappyPath verifies the canonical happy path:
// valid name + description + type:"object" schema. Per AI-07 spec § A
// req scenario "Happy path returns populated ToolDeclaration and nil".
func TestNewToolDeclaration_HappyPath(t *testing.T) {
	name := "get_weather"
	desc := "Look up the weather for a city"
	schema := validObjectSchema()
	td, err := ai.NewToolDeclaration(name, desc, schema)
	if err != nil {
		t.Fatalf("NewToolDeclaration happy path error = %v, want nil", err)
	}
	if (td == ai.ToolDeclaration{}) {
		t.Fatal("NewToolDeclaration happy path returned zero-value ToolDeclaration")
	}
	if td.Name() != name {
		t.Errorf("td.Name() = %q, want %q", td.Name(), name)
	}
	if td.Description() != desc {
		t.Errorf("td.Description() = %q, want %q", td.Description(), desc)
	}
	if !bytes.Equal(td.Schema(), []byte(schema)) {
		t.Errorf("td.Schema() = %q, want %q (raw bytes preserved)", td.Schema(), schema)
	}
}

// TestAccessors_RoundTrip verifies the Name/Description/Schema accessors
// return exactly what was passed to NewToolDeclaration. Per AI-07 spec
// § A req scenario "accessors return constructor inputs verbatim".
func TestAccessors_RoundTrip(t *testing.T) {
	t.Run("name", func(t *testing.T) {
		name := "search_files"
		td, err := ai.NewToolDeclaration(name, "desc", validObjectSchema())
		if err != nil {
			t.Fatalf("setup error: %v", err)
		}
		if got := td.Name(); got != name {
			t.Errorf("td.Name() = %q, want %q", got, name)
		}
	})
	t.Run("description", func(t *testing.T) {
		desc := "search for files by name"
		td, err := ai.NewToolDeclaration("valid_name", desc, validObjectSchema())
		if err != nil {
			t.Fatalf("setup error: %v", err)
		}
		if got := td.Description(); got != desc {
			t.Errorf("td.Description() = %q, want %q", got, desc)
		}
	})
	t.Run("schema", func(t *testing.T) {
		schema := json.RawMessage(`{"type":"object","properties":{"q":{"type":"string"}}}`)
		td, err := ai.NewToolDeclaration("valid_name", "valid description", schema)
		if err != nil {
			t.Fatalf("setup error: %v", err)
		}
		if got := td.Schema(); !bytes.Equal(got, []byte(schema)) {
			t.Errorf("td.Schema() = %q, want %q", got, schema)
		}
	})
}

// TestToolDeclaration_ZeroValue_FailsValidate verifies that a literal
// ToolDeclaration{} (constructed outside the sanctioned constructor)
// yields a value that Validate() rejects with ErrEmptyToolName. Per
// AI-07 spec § A req scenario "Zero-value ToolDeclaration fails Validate".
func TestToolDeclaration_ZeroValue_FailsValidate(t *testing.T) {
	var zero ai.ToolDeclaration
	if err := zero.Validate(); !errors.Is(err, ai.ErrEmptyToolName) {
		t.Errorf("ToolDeclaration{}.Validate() = %v, want ErrEmptyToolName", err)
	}
	if zero.Name() != "" {
		t.Errorf("ToolDeclaration{}.Name() = %q, want \"\"", zero.Name())
	}
	if zero.Description() != "" {
		t.Errorf("ToolDeclaration{}.Description() = %q, want \"\"", zero.Description())
	}
	if zero.Schema() != nil {
		t.Errorf("ToolDeclaration{}.Schema() = %q, want nil", zero.Schema())
	}
}

// =============================================================================
// Validate — re-runs structural checks
// =============================================================================

// TestToolDeclaration_Validate_AcceptsValid verifies that Validate() on
// a constructed ToolDeclaration returns nil. Per AI-07 spec § A req —
// Validate re-runs the structural checks; for a valid instance it must
// return nil.
func TestToolDeclaration_Validate_AcceptsValid(t *testing.T) {
	td, err := ai.NewToolDeclaration("valid_name", "valid description", validObjectSchema())
	if err != nil {
		t.Fatalf("setup error: %v", err)
	}
	if err := td.Validate(); err != nil {
		t.Errorf("td.Validate() on valid ToolDeclaration = %v, want nil", err)
	}
}

// =============================================================================
// ValidateTools — collection helper
// =============================================================================

// TestValidateTools_EmptySlice verifies that nil and empty slices both
// return nil from ValidateTools (no declarations → no validation
// failures, no duplicates). Per AI-07 spec § A req scenario "Empty or
// nil slice passes ValidateTools".
func TestValidateTools_EmptySlice(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		if err := ai.ValidateTools(nil); err != nil {
			t.Errorf("ValidateTools(nil) = %v, want nil", err)
		}
	})
	t.Run("empty_slice", func(t *testing.T) {
		if err := ai.ValidateTools([]ai.ToolDeclaration{}); err != nil {
			t.Errorf("ValidateTools([]) = %v, want nil", err)
		}
	})
}

// TestValidateTools_UniqueNames verifies that a slice with distinct
// valid names returns nil from ValidateTools. Per AI-07 spec § A req
// scenario "Distinct-name slice passes ValidateTools".
func TestValidateTools_UniqueNames(t *testing.T) {
	decls := []ai.ToolDeclaration{
		mustToolDeclaration(t, "tool_a", "Tool A", validObjectSchema()),
		mustToolDeclaration(t, "tool_b", "Tool B", validObjectSchema()),
		mustToolDeclaration(t, "tool_c", "Tool C", validObjectSchema()),
	}
	if err := ai.ValidateTools(decls); err != nil {
		t.Errorf("ValidateTools([3 distinct names]) = %v, want nil", err)
	}
}

// TestValidateTools_DuplicateName verifies ErrDuplicateToolName when two
// declarations share the same byte-equal name. Per AI-07 spec § A req
// scenario "Duplicate name in slice is rejected with ErrDuplicateToolName".
// Whitespace-padded variants (`"foo"` vs `"foo "`) MUST be treated as
// distinct — the comparison is byte-equal.
func TestValidateTools_DuplicateName(t *testing.T) {
	t.Run("byte_equal_duplicates", func(t *testing.T) {
		a := mustToolDeclaration(t, "duplicate", "first", validObjectSchema())
		b := mustToolDeclaration(t, "duplicate", "second", validObjectSchema())
		err := ai.ValidateTools([]ai.ToolDeclaration{a, b})
		if !errors.Is(err, ai.ErrDuplicateToolName) {
			t.Errorf("ValidateTools([a, b] with same name) = %v, want ErrDuplicateToolName", err)
		}
		if !strings.Contains(err.Error(), "duplicate") {
			t.Errorf("ErrDuplicateToolName message = %q, want substring %q", err.Error(), "duplicate")
		}
	})
	t.Run("whitespace_padded_distinct", func(t *testing.T) {
		// "foo" vs "foo " differ by one trailing space — byte-equal comparison
		// must treat them as distinct names.
		a := mustToolDeclaration(t, "foo", "first", validObjectSchema())
		b := mustToolDeclaration(t, "foo ", "second", validObjectSchema())
		if err := ai.ValidateTools([]ai.ToolDeclaration{a, b}); err != nil {
			t.Errorf("ValidateTools([foo, foo ]) = %v, want nil (whitespace-padded variants are distinct by byte-equal)", err)
		}
	})
}

// TestValidateTools_FirstFailureWins verifies that per-instance failures
// short-circuit the duplicate-name check. Per AI-07 spec § A req scenario
// "Per-instance failure short-circuits duplicate detection".
func TestValidateTools_FirstFailureWins(t *testing.T) {
	invalid := ai.ToolDeclaration{} // zero value — Validate() returns ErrEmptyToolName
	a := mustToolDeclaration(t, "tool_a", "Tool A", validObjectSchema())
	b := mustToolDeclaration(t, "tool_a", "Tool A duplicate", validObjectSchema())
	err := ai.ValidateTools([]ai.ToolDeclaration{invalid, a, b})
	if !errors.Is(err, ai.ErrEmptyToolName) {
		t.Errorf("ValidateTools([invalid, a, b where a.Name()==b.Name()]) = %v, want ErrEmptyToolName (per-instance first)", err)
	}
}

// =============================================================================
// Sentinel distinctness + package prefix
// =============================================================================

// TestSentinels_Distinct verifies the 10 ToolDeclaration sentinels are
// distinct errors (callers can branch on which constraint failed) via a
// pairwise errors.Is cross-check. Per AI-07 spec § A req scenario "Every
// pair of the 10 sentinels is distinct under errors.Is".
func TestSentinels_Distinct(t *testing.T) {
	sentinels := map[error]string{
		ai.ErrEmptyToolName:          "ErrEmptyToolName",
		ai.ErrToolNameTooLong:        "ErrToolNameTooLong",
		ai.ErrInvalidToolName:        "ErrInvalidToolName",
		ai.ErrEmptyToolDescription:   "ErrEmptyToolDescription",
		ai.ErrToolDescriptionTooLong: "ErrToolDescriptionTooLong",
		ai.ErrInvalidToolSchema:      "ErrInvalidToolSchema",
		ai.ErrMissingSchemaType:      "ErrMissingSchemaType",
		ai.ErrUnsupportedSchemaType:  "ErrUnsupportedSchemaType",
		ai.ErrUnsupportedSchemaFeature: "ErrUnsupportedSchemaFeature",
		ai.ErrDuplicateToolName:      "ErrDuplicateToolName",
	}
	if len(sentinels) != 10 {
		t.Errorf("AI-07 spec § A req requires exactly 10 ToolDeclaration sentinels, found %d", len(sentinels))
	}

	// Pairwise cross-check: distinct sentinels must NOT alias via errors.Is.
	pairs := []struct {
		a, b error
	}{
		{ai.ErrEmptyToolName, ai.ErrToolNameTooLong},
		{ai.ErrEmptyToolName, ai.ErrInvalidToolName},
		{ai.ErrEmptyToolName, ai.ErrEmptyToolDescription},
		{ai.ErrEmptyToolName, ai.ErrToolDescriptionTooLong},
		{ai.ErrEmptyToolName, ai.ErrInvalidToolSchema},
		{ai.ErrEmptyToolName, ai.ErrMissingSchemaType},
		{ai.ErrEmptyToolName, ai.ErrUnsupportedSchemaType},
		{ai.ErrEmptyToolName, ai.ErrUnsupportedSchemaFeature},
		{ai.ErrEmptyToolName, ai.ErrDuplicateToolName},
		{ai.ErrToolNameTooLong, ai.ErrInvalidToolName},
		{ai.ErrToolNameTooLong, ai.ErrEmptyToolDescription},
		{ai.ErrToolNameTooLong, ai.ErrToolDescriptionTooLong},
		{ai.ErrToolNameTooLong, ai.ErrInvalidToolSchema},
		{ai.ErrToolNameTooLong, ai.ErrMissingSchemaType},
		{ai.ErrToolNameTooLong, ai.ErrUnsupportedSchemaType},
		{ai.ErrToolNameTooLong, ai.ErrUnsupportedSchemaFeature},
		{ai.ErrToolNameTooLong, ai.ErrDuplicateToolName},
		{ai.ErrInvalidToolName, ai.ErrEmptyToolDescription},
		{ai.ErrInvalidToolName, ai.ErrToolDescriptionTooLong},
		{ai.ErrInvalidToolName, ai.ErrInvalidToolSchema},
		{ai.ErrInvalidToolName, ai.ErrMissingSchemaType},
		{ai.ErrInvalidToolName, ai.ErrUnsupportedSchemaType},
		{ai.ErrInvalidToolName, ai.ErrUnsupportedSchemaFeature},
		{ai.ErrInvalidToolName, ai.ErrDuplicateToolName},
		{ai.ErrEmptyToolDescription, ai.ErrToolDescriptionTooLong},
		{ai.ErrEmptyToolDescription, ai.ErrInvalidToolSchema},
		{ai.ErrEmptyToolDescription, ai.ErrMissingSchemaType},
		{ai.ErrEmptyToolDescription, ai.ErrUnsupportedSchemaType},
		{ai.ErrEmptyToolDescription, ai.ErrUnsupportedSchemaFeature},
		{ai.ErrEmptyToolDescription, ai.ErrDuplicateToolName},
		{ai.ErrToolDescriptionTooLong, ai.ErrInvalidToolSchema},
		{ai.ErrToolDescriptionTooLong, ai.ErrMissingSchemaType},
		{ai.ErrToolDescriptionTooLong, ai.ErrUnsupportedSchemaType},
		{ai.ErrToolDescriptionTooLong, ai.ErrUnsupportedSchemaFeature},
		{ai.ErrToolDescriptionTooLong, ai.ErrDuplicateToolName},
		{ai.ErrInvalidToolSchema, ai.ErrMissingSchemaType},
		{ai.ErrInvalidToolSchema, ai.ErrUnsupportedSchemaType},
		{ai.ErrInvalidToolSchema, ai.ErrUnsupportedSchemaFeature},
		{ai.ErrInvalidToolSchema, ai.ErrDuplicateToolName},
		{ai.ErrMissingSchemaType, ai.ErrUnsupportedSchemaType},
		{ai.ErrMissingSchemaType, ai.ErrUnsupportedSchemaFeature},
		{ai.ErrMissingSchemaType, ai.ErrDuplicateToolName},
		{ai.ErrUnsupportedSchemaType, ai.ErrUnsupportedSchemaFeature},
		{ai.ErrUnsupportedSchemaType, ai.ErrDuplicateToolName},
		{ai.ErrUnsupportedSchemaFeature, ai.ErrDuplicateToolName},
	}
	for _, p := range pairs {
		if errors.Is(p.a, p.b) {
			t.Errorf("%v must NOT alias %v via errors.Is", p.a, p.b)
		}
	}

	// Pin the package prefix convention.
	for err, name := range sentinels {
		if msg := err.Error(); !strings.HasPrefix(msg, "ai: ") {
			t.Errorf("%s.Error() = %q, want prefix %q (package convention)", name, msg, "ai: ")
		}
	}

	// Self-distinctness: every sentinel must satisfy errors.Is against itself.
	for err, name := range sentinels {
		if !errors.Is(err, err) {
			t.Errorf("%s must satisfy errors.Is(err, err)", name)
		}
	}
}

// TestSentinels_ExportedAndTyped verifies each ToolDeclaration sentinel
// is non-nil, has a non-empty message, and is compatible with errors.Is
// (self-check). Per AI-07 spec § A req — every exported sentinel must
// be usable by callers.
func TestSentinels_ExportedAndTyped(t *testing.T) {
	cases := map[error]string{
		ai.ErrEmptyToolName:          "ErrEmptyToolName",
		ai.ErrToolNameTooLong:        "ErrToolNameTooLong",
		ai.ErrInvalidToolName:        "ErrInvalidToolName",
		ai.ErrEmptyToolDescription:   "ErrEmptyToolDescription",
		ai.ErrToolDescriptionTooLong: "ErrToolDescriptionTooLong",
		ai.ErrInvalidToolSchema:      "ErrInvalidToolSchema",
		ai.ErrMissingSchemaType:      "ErrMissingSchemaType",
		ai.ErrUnsupportedSchemaType:  "ErrUnsupportedSchemaType",
		ai.ErrUnsupportedSchemaFeature: "ErrUnsupportedSchemaFeature",
		ai.ErrDuplicateToolName:      "ErrDuplicateToolName",
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
// Boundary pin — ToolDeclaration is NOT a ContentPart
// =============================================================================

// TestToolDeclaration_NotContentPart pins the boundary invariant via
// reflection: ai.ToolDeclaration MUST NOT have a `Kind() ai.Kind` method.
// If a future maintainer adds one (intentionally or by drift), this
// assertion fails. Per AI-07 proposal § Decision 1 + spec § A req 1
// scenario "ToolDeclaration does not satisfy ContentPart".
//
// The negative compile-time assertion (no `var _ ai.ContentPart = ai.ToolDeclaration{}`
// in this file) is the verified contract. Adding one would compile but
// would break the wording trap (milestone doc line 67).
func TestToolDeclaration_NotContentPart(t *testing.T) {
	tdType := reflect.TypeOf(ai.ToolDeclaration{})
	kindMethod, ok := tdType.MethodByName("Kind")
	if ok {
		// If Kind() exists on ToolDeclaration, fail loudly. A wrapper
		// would not survive this check.
		t.Errorf("ai.ToolDeclaration has a Kind() method — this would make it a ContentPart. "+
			"Per AI-07 proposal § Decision 1 and milestone doc line 67 (wording trap), "+
			"ToolDeclaration is a request-level concept, NOT a ContentPart. "+
			"Got: %s", kindMethod.Type)
	}
}

// TestToolDeclaration_NoMarshalOrExecute pins the absence of execution
// surface on ToolDeclaration. Per AI-07 spec § A req scenarios "ToolDeclaration
// has no MarshalJSON/UnmarshalJSON in v1" and "ToolDeclaration has no
// execution callback". The reflection check catches any future maintainer
// who adds these forbidden methods to tool.go.
func TestToolDeclaration_NoMarshalOrExecute(t *testing.T) {
	tdType := reflect.TypeOf(ai.ToolDeclaration{})
	forbidden := []string{
		"MarshalJSON",
		"UnmarshalJSON",
		"Run",
		"Execute",
		"Invoke",
		"Callback",
		"Handler",
		"Func",
	}
	for _, name := range forbidden {
		if _, ok := tdType.MethodByName(name); ok {
			t.Errorf("ai.ToolDeclaration must NOT have a %s() method (per AI-07 spec § A req)", name)
		}
	}
}

// =============================================================================
// doc.go amendment smoke test
// =============================================================================

// TestDocGo_ToolDeclarationParagraph verifies the AI-07 doc.go paragraph:
//   - ends with a period
//   - is at most 10 lines
//   - mentions the 3 API names + 2 max-length constants
//   - contains "See AI-07" (the policy citation)
//
// Per AI-07 spec § A req scenario "doc.go paragraph ends with a period
// and contains the required substrings".
func TestDocGo_ToolDeclarationParagraph(t *testing.T) {
	// Resolve the doc.go path: this test file is at
	// backend/database_administrator/src/tools/agent/ai/tool_test.go,
	// so doc.go is in the same directory.
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	docPath := filepath.Join(filepath.Dir(thisFile), "doc.go")
	contents, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read %s: %v", docPath, err)
	}
	text := string(contents)

	// Required substrings per AI-07 spec § A req scenario.
	required := []string{
		"ToolDeclaration",
		"NewToolDeclaration",
		"ValidateTools",
		"MaxToolNameLength",
		"MaxToolDescriptionLength",
		"See AI-07",
	}
	for _, sub := range required {
		if !strings.Contains(text, sub) {
			t.Errorf("doc.go must contain substring %q (per AI-07 spec § A req)", sub)
		}
	}

	// Locate the AI-07 paragraph by finding the "As of AI-07" marker, then
	// extract its lines up to the package declaration (or end of file).
	const marker = "As of AI-07"
	start := strings.Index(text, marker)
	if start == -1 {
		t.Fatal("doc.go does not contain the 'As of AI-07' paragraph marker")
	}
	// Find the package declaration after the marker.
	after := text[start:]
	pkgIdx := strings.Index(after, "package ai")
	if pkgIdx == -1 {
		t.Fatal("doc.go AI-07 paragraph is not followed by 'package ai'")
	}
	paragraph := after[:pkgIdx]
	lines := strings.Split(strings.TrimRight(paragraph, "\n"), "\n")

	if len(lines) > 10 {
		t.Errorf("AI-07 doc.go paragraph has %d lines, want ≤ 10 (spec § A req budget)", len(lines))
	}

	// The final non-empty line of the paragraph must end with a period.
	last := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			last = strings.TrimSpace(lines[i])
			break
		}
	}
	if last == "" {
		t.Fatal("AI-07 doc.go paragraph has no non-empty trailing line")
	}
	if !strings.HasSuffix(last, ".") {
		t.Errorf("AI-07 doc.go paragraph's last line %q does not end with a period", last)
	}
}

// =============================================================================
// Test helpers
// =============================================================================

// validObjectSchema returns a canonical type:"object" schema for tests.
// Kept in the test file because it has no production-side utility.
func validObjectSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

// mustToolDeclaration wraps NewToolDeclaration for slice-literal test
// contexts. Test inputs are hardcoded valid combinations, so the error
// is guaranteed nil. Mirrors the mustReasoningPart helper in reasoning_test.go.
func mustToolDeclaration(t *testing.T, name, description string, schema json.RawMessage) ai.ToolDeclaration {
	t.Helper()
	td, err := ai.NewToolDeclaration(name, description, schema)
	if err != nil {
		t.Fatalf("mustToolDeclaration(%q, %q) returned error %v, want nil", name, description, err)
	}
	return td
}
