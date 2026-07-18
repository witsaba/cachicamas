package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"unicode"
)

// MaxToolNameLength is the upper bound on ToolDeclaration names, in
// bytes. The value is 64 bytes. Rationale: matches OpenAI's tool-name
// length convention and accommodates every Anthropic / Google tool-name
// pattern without inflating the request envelope. Per AI-07 spec § A req
// "Name validation rejects empty, whitespace-only, oversized, and
// control-char names".
const MaxToolNameLength = 64

// MaxToolDescriptionLength is the upper bound on ToolDeclaration
// descriptions, in bytes. The value is 1024 bytes (1 KiB). Rationale:
// vendor tool-description limits hover around 1–2 KiB; 1 KiB is the
// conservative cap that fits every supported vendor. Per AI-07 spec § A
// req "Description validation rejects empty, whitespace-only, and
// oversized descriptions".
const MaxToolDescriptionLength = 1 << 10

// ErrEmptyToolName is returned by NewToolDeclaration when the supplied
// name is empty or whitespace-only (every rune reports true for
// unicode.IsSpace, via the same-package isAllWhitespace helper). Per
// AI-07 spec § A req scenario "Empty / whitespace-only name is rejected".
var ErrEmptyToolName = errors.New("ai: empty tool name")

// ErrToolNameTooLong is returned by NewToolDeclaration when
// len(name) > MaxToolNameLength. Per AI-07 spec § A req scenario
// "Name at MaxToolNameLength is accepted; MaxToolNameLength+1 is rejected".
var ErrToolNameTooLong = errors.New("ai: tool name exceeds MaxToolNameLength")

// ErrInvalidToolName is returned by NewToolDeclaration when any rune of
// the name reports true for unicode.IsControl. Covers NUL, LF, TAB, CR,
// and ANSI escape sequences. Per AI-07 spec § A req scenario "Name
// containing a control character is rejected".
var ErrInvalidToolName = errors.New("ai: tool name contains control character")

// ErrEmptyToolDescription is returned by NewToolDeclaration when the
// supplied description is empty or whitespace-only. Per AI-07 spec § A
// req scenario "Empty description is rejected".
var ErrEmptyToolDescription = errors.New("ai: empty tool description")

// ErrToolDescriptionTooLong is returned by NewToolDeclaration when
// len(description) > MaxToolDescriptionLength. Per AI-07 spec § A req
// scenario "Description at MaxToolDescriptionLength is accepted; +1
// is rejected".
var ErrToolDescriptionTooLong = errors.New("ai: tool description exceeds MaxToolDescriptionLength")

// ErrInvalidToolSchema is returned by NewToolDeclaration when the schema
// is nil, empty, or fails json.Unmarshal into a JSON object. Per
// AI-07 spec § A req scenario "Schema that is not parseable JSON is
// rejected".
var ErrInvalidToolSchema = errors.New("ai: tool schema is not valid JSON")

// ErrMissingSchemaType is returned by NewToolDeclaration when the parsed
// schema's top-level object has no "type" key. Per AI-07 spec § A req
// scenario "Schema without a top-level type is rejected".
var ErrMissingSchemaType = errors.New("ai: tool schema missing top-level type")

// ErrUnsupportedSchemaType is returned by NewToolDeclaration when the
// parsed schema's top-level "type" is not the string "object". Per
// AI-07 spec § A req scenario "Schema with a top-level type other than
// object is rejected".
var ErrUnsupportedSchemaType = errors.New("ai: tool schema top-level type must be object")

// ErrUnsupportedSchemaFeature is returned by NewToolDeclaration when the
// parsed schema's top-level keys include a deny-listed keyword. The
// returned error wraps the sentinel via fmt.Errorf so the offending
// keyword name appears in the message; callers should branch on
// errors.Is(err, ErrUnsupportedSchemaFeature) and inspect err.Error()
// for the keyword.
//
// Per AI-07 spec § A req scenario "Schema with each deny-listed
// top-level keyword is rejected with ErrUnsupportedSchemaFeature".
var ErrUnsupportedSchemaFeature = errors.New("ai: tool schema uses unsupported feature")

// ErrDuplicateToolName is returned by ValidateTools when two
// declarations share the same byte-equal name. The returned error wraps
// the sentinel via fmt.Errorf so the offending name appears in the
// message; callers should branch on errors.Is(err, ErrDuplicateToolName).
//
// Per AI-07 spec § A req scenario "Duplicate name in slice is rejected".
var ErrDuplicateToolName = errors.New("ai: duplicate tool name")

// unsupportedSchemaKeywords is the v1 deny-list of top-level JSON Schema
// keywords that Layer 1 does not interpret. The deny-list is enforced
// at the top level only — nested occurrences are out of scope for
// structural-only validation (see AI-07 proposal R3).
//
// Why these keywords:
//
//   - $ref / $schema / $id: cross-file / metadata references that
//     require schema-link resolution.
//   - oneOf / anyOf / allOf / not: combinators that v1 adapters cannot
//     reliably translate without Layer 1 understanding them.
//   - patternProperties: regex-based property patterns that depend on
//     a regex engine.
//   - dependencies / propertyNames: cross-property constraints.
//   - if / then / else: conditional schemas.
//
// Per AI-07 spec § A req scenario "Schema with each deny-listed
// top-level keyword is rejected".
var unsupportedSchemaKeywords = [...]string{
	"$ref",
	"$schema",
	"$id",
	"oneOf",
	"anyOf",
	"allOf",
	"not",
	"patternProperties",
	"dependencies",
	"propertyNames",
	"if",
	"then",
	"else",
}

// ToolDeclaration is the provider-neutral value type for a tool
// declaration carried at the model boundary. See AI-00 § tool
// declaration (Layer 1 sense) and AI-07 spec § A "Requirement:
// ToolDeclaration is a struct with unexported fields, not a ContentPart".
//
// ToolDeclaration is NOT a ContentPart — it deliberately does not
// implement the ContentPart interface. The reserved KindToolDeclaration
// slot in content.go stays unused in v1. This makes ToolDeclaration a
// request-level concept: a future Request (AI-09) carries []ToolDeclaration
// to surface available tools to the model. Per the wording trap on
// milestone doc line 67, Layer 1's role is the transport representation
// — it must not execute tools, resolve tool names, or own application
// behavior.
//
// The fields are unexported so NewToolDeclaration is the only path that
// produces a validated value. A literal ToolDeclaration{} yields empty
// name + empty description + nil schema, all of which Validate (and
// NewToolDeclaration) reject.
type ToolDeclaration struct {
	name        string
	description string
	schema      json.RawMessage
}

// Name returns the tool name exactly as the caller passed it to
// NewToolDeclaration. The name is NOT trimmed by the constructor —
// what the caller passes is what Name returns. Per AI-07 spec § A req
// scenario "accessors return constructor inputs verbatim".
func (t ToolDeclaration) Name() string {
	return t.name
}

// Description returns the tool description exactly as the caller passed
// it to NewToolDeclaration. Per AI-07 spec § A req scenario "accessors
// return constructor inputs verbatim".
func (t ToolDeclaration) Description() string {
	return t.description
}

// Schema returns the tool's input JSON Schema as json.RawMessage. The
// bytes are preserved verbatim from what the caller passed to
// NewToolDeclaration — encoding/json round-trips through RawMessage
// unchanged. Adapters that re-marshal MUST emit the original bytes,
// not a re-encoded parsed form, to preserve user formatting. Per
// AI-07 spec § A req scenario "accessors return constructor inputs verbatim".
func (t ToolDeclaration) Schema() json.RawMessage {
	return t.schema
}

// Validate re-runs the same structural checks as NewToolDeclaration
// against the receiver. It returns nil for a valid ToolDeclaration
// (one that was produced by NewToolDeclaration without error); for a
// zero-value ToolDeclaration{}, it returns ErrEmptyToolName (the name
// check fires first because the field is empty).
//
// Validate is for re-validation of values that may have been mutated
// across serialization boundaries or for sanity checks after
// reflection. The constructor is the sanctioned path for new values.
// Per AI-07 spec § A req "Zero-value ToolDeclaration fails Validate".
func (t ToolDeclaration) Validate() error {
	if t.name == "" || isAllWhitespace(t.name) {
		return ErrEmptyToolName
	}
	if len(t.name) > MaxToolNameLength {
		return ErrToolNameTooLong
	}
	if hasControlChar(t.name) {
		return ErrInvalidToolName
	}
	if t.description == "" || isAllWhitespace(t.description) {
		return ErrEmptyToolDescription
	}
	if len(t.description) > MaxToolDescriptionLength {
		return ErrToolDescriptionTooLong
	}
	return validateToolSchema(t.schema)
}

// NewToolDeclaration validates and constructs a ToolDeclaration. On any
// validation failure it returns the zero-value ToolDeclaration and the
// corresponding typed sentinel error. Validation order is name →
// description → schema (state-first, matching the AI-06 Reasoning
// precedent). Per AI-07 spec § A req "NewToolDeclaration is the
// sanctioned constructor and returns zero-value on error".
//
// Validation rules:
//
//   - Name: non-empty after isAllWhitespace, len ≤ MaxToolNameLength,
//     no rune reporting true for unicode.IsControl. The constructor
//     does NOT trim the name; what the caller passes is what Name
//     returns.
//   - Description: non-empty after isAllWhitespace, len ≤
//     MaxToolDescriptionLength. Newlines, tabs, and other formatting
//     characters are allowed (unlike names).
//   - Schema: parseable JSON, top-level "type":"object", no deny-listed
//     keyword at the top level.
//
// v1 validation is structural only — semantic shape (e.g., "required"
// field types, $ref resolution) is AI-17's concern. Per AI-07 spec § A
// req "Schema validation rejects malformed, missing-type, wrong-type,
// and deny-listed-keyword schemas".
func NewToolDeclaration(name, description string, schema json.RawMessage) (ToolDeclaration, error) {
	if name == "" || isAllWhitespace(name) {
		return ToolDeclaration{}, ErrEmptyToolName
	}
	if len(name) > MaxToolNameLength {
		return ToolDeclaration{}, ErrToolNameTooLong
	}
	if hasControlChar(name) {
		return ToolDeclaration{}, ErrInvalidToolName
	}

	if description == "" || isAllWhitespace(description) {
		return ToolDeclaration{}, ErrEmptyToolDescription
	}
	if len(description) > MaxToolDescriptionLength {
		return ToolDeclaration{}, ErrToolDescriptionTooLong
	}

	if err := validateToolSchema(schema); err != nil {
		return ToolDeclaration{}, err
	}

	return ToolDeclaration{name: name, description: description, schema: schema}, nil
}

// validateToolSchema enforces the structural schema rules. Returns nil
// for a valid schema; otherwise returns the typed sentinel for the
// first failing check. Order: nil/empty → parse → type-key →
// type-value → deny-list. Per AI-07 spec § A req "Schema validation".
func validateToolSchema(schema json.RawMessage) error {
	if len(schema) == 0 {
		return ErrInvalidToolSchema
	}
	var parsed map[string]any
	if err := json.Unmarshal(schema, &parsed); err != nil {
		return ErrInvalidToolSchema
	}
	typeVal, ok := parsed["type"]
	if !ok {
		return ErrMissingSchemaType
	}
	typeStr, ok := typeVal.(string)
	if !ok || typeStr != "object" {
		return ErrUnsupportedSchemaType
	}
	if keyword, found := containsUnsupportedKeyword(parsed); found {
		return fmt.Errorf("%w: %q", ErrUnsupportedSchemaFeature, keyword)
	}
	return nil
}

// containsUnsupportedKeyword walks the top-level keys of parsed and
// returns the first one that appears in unsupportedSchemaKeywords,
// along with true. If no top-level key is denied, it returns "", false.
// Map iteration order is non-deterministic in Go; tests must therefore
// assert via errors.Is + strings.Contains, NOT exact-string equality
// on err.Error() (per AI-07 OI-3).
func containsUnsupportedKeyword(parsed map[string]any) (string, bool) {
	for key := range parsed {
		for _, denied := range unsupportedSchemaKeywords {
			if key == denied {
				return key, true
			}
		}
	}
	return "", false
}

// hasControlChar reports whether any rune of s reports true for
// unicode.IsControl. Reused only within this file (tool names reject
// control characters; descriptions allow them).
func hasControlChar(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}

// ValidateTools walks decls and returns the first validation error.
// Specifically:
//
//  1. Each declaration is validated via Validate(). The first
//     per-instance failure is returned, short-circuiting remaining
//     checks.
//  2. If every declaration passes Validate(), the slice is checked
//     for duplicate names. The second occurrence of a name returns
//     ErrDuplicateToolName (wrapped with the offending name via
//     fmt.Errorf so the name appears in the message).
//
// Empty or nil slices (nil and []ToolDeclaration{}) return nil. Name
// comparison is byte-equal on Name() — whitespace-padded variants
// are treated as distinct.
//
// ValidateTools is a package-level free function (NOT a method) because
// no Request type exists yet (AI-09's concern). When Request lands in
// AI-09, Request.Validate MUST call this helper instead of
// re-implementing the duplicate check. Per AI-07 spec § A req
// "ValidateTools checks per-instance first then rejects duplicate names".
func ValidateTools(decls []ToolDeclaration) error {
	seen := make(map[string]struct{}, len(decls))
	for _, d := range decls {
		if err := d.Validate(); err != nil {
			return err
		}
		name := d.Name()
		if _, dup := seen[name]; dup {
			return fmt.Errorf("%w: %q", ErrDuplicateToolName, name)
		}
		seen[name] = struct{}{}
	}
	return nil
}
