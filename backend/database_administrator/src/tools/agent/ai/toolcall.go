package ai

import (
	"encoding/json"
	"errors"
)

// MaxToolCallArgumentsLength is the upper bound on ToolCall arguments, in
// bytes. The value is 1 MiB (1,048,576 bytes). Rationale: realistic tool
// arguments (city names, search queries, file paths) are orders of magnitude
// smaller. A 1 MiB cap prevents OOM while matching the MaxTextLength
// precedent. Per AI-08 spec § D and proposal § Decision 1.
const MaxToolCallArgumentsLength = 1 << 20

// ErrEmptyToolCallName is returned by NewToolCall when the supplied name
// is empty or whitespace-only (every rune reports true for unicode.IsSpace).
// Per AI-08 spec § E scenario "name is empty or whitespace-only".
var ErrEmptyToolCallName = errors.New("ai: empty tool call name")

// ErrToolCallNameTooLong is returned by NewToolCall when
// len(name) > MaxToolNameLength. Per AI-08 spec § E scenario
// "len(name) > 64".
var ErrToolCallNameTooLong = errors.New("ai: tool call name exceeds MaxToolNameLength")

// ErrInvalidToolCallName is returned by NewToolCall when any rune of
// the name reports true for unicode.IsControl. Per AI-08 spec § E
// scenario "any rune is control character".
var ErrInvalidToolCallName = errors.New("ai: tool call name contains control character")

// ErrMalformedToolCallArguments is returned by NewToolCall when the
// arguments is nil, empty, or fails json.Unmarshal into a JSON value.
// Per AI-08 spec § E scenario "arguments is nil/empty/malformed JSON".
var ErrMalformedToolCallArguments = errors.New("ai: tool call arguments is not valid JSON")

// ErrToolCallArgumentsTooLong is returned by NewToolCall when
// len(arguments) > MaxToolCallArgumentsLength. Per AI-08 spec § E
// scenario "len(arguments) > MaxToolCallArgumentsLength".
var ErrToolCallArgumentsTooLong = errors.New("ai: tool call arguments exceeds MaxToolCallArgumentsLength")

// ToolCall is the provider-neutral value type for a tool-call request
// emitted by the assistant in an assistant-role message. See AI-00 § tool
// call (Layer 1 sense) and AI-08 spec § A.
//
// ToolCall is NOT a ContentPart — only the toolCallPart wrapper implements
// ContentPart. This follows the textPart+Text wrapper pattern (not the
// Reasoning direct-implements pattern) per AI-08 design obs #2099.
//
// The fields are unexported so NewToolCall is the only path that produces
// a validated value. A literal ToolCall{} yields empty name + nil arguments,
// both of which Validate (and NewToolCall) reject.
type ToolCall struct {
	name      string
	arguments json.RawMessage
}

// Name returns the tool name exactly as the caller passed it to
// NewToolCall. The name is NOT trimmed by the constructor — what the
// caller passes is what Name returns. Per AI-08 spec § A req scenario
// "accessors return constructor inputs verbatim".
func (t ToolCall) Name() string {
	return t.name
}

// Arguments returns the tool-call arguments as json.RawMessage. The bytes
// are preserved verbatim from what the caller passed to NewToolCall —
// encoding/json round-trips through RawMessage unchanged. Per AI-08 spec
// § A req scenario "accessors return constructor inputs verbatim".
func (t ToolCall) Arguments() json.RawMessage {
	return t.arguments
}

// Validate re-runs the same structural checks as NewToolCall against the
// receiver. Returns nil for a valid ToolCall (one produced by NewToolCall
// without error); for a zero-value ToolCall{} it returns ErrEmptyToolCallName.
//
// Validate is for re-validation of values that may have been mutated across
// serialization boundaries or for sanity checks after reflection. The
// constructor is the sanctioned path for new values. Per AI-08 spec § A req
// "Validate re-runs structural checks".
func (t ToolCall) Validate() error {
	if t.name == "" || isAllWhitespace(t.name) {
		return ErrEmptyToolCallName
	}
	if len(t.name) > MaxToolNameLength {
		return ErrToolCallNameTooLong
	}
	if hasControlChar(t.name) {
		return ErrInvalidToolCallName
	}
	if err := validateToolCallArguments(t.arguments); err != nil {
		return err
	}
	return nil
}

// validateToolCallArguments enforces the structural rules for ToolCall
// arguments. Returns nil for valid arguments; otherwise the typed sentinel.
// Order: nil/empty → parse → length. Per AI-08 spec § F.
func validateToolCallArguments(args json.RawMessage) error {
	if len(args) == 0 {
		return ErrMalformedToolCallArguments
	}
	var parsed any
	if err := json.Unmarshal(args, &parsed); err != nil {
		return ErrMalformedToolCallArguments
	}
	if len(args) > MaxToolCallArgumentsLength {
		return ErrToolCallArgumentsTooLong
	}
	return nil
}

// NewToolCall validates and constructs a ToolCall. On any validation
// failure it returns the zero-value ToolCall and the corresponding typed
// sentinel error. Validation order is name → arguments (state-first, matching
// the NewToolDeclaration precedent). Per AI-08 spec § A req "NewToolCall
// is the sanctioned constructor and returns zero-value on error".
//
// Validation rules:
//
//   - Name: non-empty after isAllWhitespace, len ≤ MaxToolNameLength,
//     no rune reporting true for unicode.IsControl. The constructor does
//     NOT trim the name; what the caller passes is what Name returns.
//   - Arguments: non-nil, non-empty, parseable as JSON (any value),
//     len ≤ MaxToolCallArgumentsLength.
//
// Per AI-08 spec § F.
func NewToolCall(name string, arguments json.RawMessage) (ToolCall, error) {
	if name == "" || isAllWhitespace(name) {
		return ToolCall{}, ErrEmptyToolCallName
	}
	if len(name) > MaxToolNameLength {
		return ToolCall{}, ErrToolCallNameTooLong
	}
	if hasControlChar(name) {
		return ToolCall{}, ErrInvalidToolCallName
	}
	if err := validateToolCallArguments(arguments); err != nil {
		return ToolCall{}, err
	}
	return ToolCall{name: name, arguments: arguments}, nil
}
