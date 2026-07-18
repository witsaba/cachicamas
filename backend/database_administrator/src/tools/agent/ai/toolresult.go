package ai

import (
	"encoding/json"
	"errors"
)

// MaxToolResultCallIDLength is the upper bound on ToolResult call-ID
// correlation strings, in bytes. The value is 512 bytes. Rationale:
// sufficient for any realistic correlation-ID scheme (UUIDs, ULIDs,
// counter-based IDs). Per AI-08 spec § D.
const MaxToolResultCallIDLength = 512

// MaxToolResultContentLength is the upper bound on ToolResult content
// payloads, in bytes. The value is 10 MiB (10,485,760 bytes). Rationale:
// tool results can legitimately be larger than typical text content
// (file-read, database-query tools returning bulk data). A 10 MiB cap
// (10× text limit) provides realistic headroom without creating OOM
// vectors. Per AI-08 spec § D and proposal § Decision 2.
const MaxToolResultContentLength = 10 * (1 << 20)

// ErrEmptyToolResultCallID is returned by NewToolResult when the supplied
// callID is empty or whitespace-only. Per AI-08 spec § E scenario
// "callID is empty or whitespace-only".
var ErrEmptyToolResultCallID = errors.New("ai: empty tool result call ID")

// ErrToolResultCallIDTooLong is returned by NewToolResult when
// len(callID) > MaxToolResultCallIDLength. Per AI-08 spec § E scenario
// "len(callID) > 512".
var ErrToolResultCallIDTooLong = errors.New("ai: tool result call ID exceeds MaxToolResultCallIDLength")

// ErrMalformedToolResultContent is returned by NewToolResult when the
// content is nil, empty, or fails json.Unmarshal into a JSON value.
// Per AI-08 spec § E scenario "content is nil/empty/malformed JSON".
var ErrMalformedToolResultContent = errors.New("ai: tool result content is not valid JSON")

// ErrToolResultContentTooLong is returned by NewToolResult when
// len(content) > MaxToolResultContentLength. Per AI-08 spec § E
// scenario "len(content) > MaxToolResultContentLength".
var ErrToolResultContentTooLong = errors.New("ai: tool result content exceeds MaxToolResultContentLength")

// ToolResult is the provider-neutral value type for a tool-result content
// part carried in a tool-role message that echoes the result back to the
// model. Correlation with the originating ToolCall is via the containing
// Message.ID field (message.go line 24). See AI-00 § tool result (Layer 1
// sense) and AI-08 spec § B.
//
// ToolResult is NOT a ContentPart — only the toolResultPart wrapper
// implements ContentPart. Per AI-08 design obs #2099.
//
// The fields are unexported so NewToolResult is the only path that produces
// a validated value. A literal ToolResult{} yields empty callID + nil
// content, both of which Validate (and NewToolResult) reject.
type ToolResult struct {
	callID  string
	content json.RawMessage
}

// CallID returns the correlation ID exactly as the caller passed it to
// NewToolResult. Per AI-08 spec § B req scenario "accessors return
// constructor inputs verbatim".
func (t ToolResult) CallID() string {
	return t.callID
}

// Content returns the tool-result content as json.RawMessage. The bytes
// are preserved verbatim from what the caller passed to NewToolResult —
// encoding/json round-trips through RawMessage unchanged. Per AI-08 spec
// § B req scenario "accessors return constructor inputs verbatim".
func (t ToolResult) Content() json.RawMessage {
	return t.content
}

// Validate re-runs the same structural checks as NewToolResult against
// the receiver. Returns nil for a valid ToolResult (one produced by
// NewToolResult without error); for a zero-value ToolResult{} it returns
// ErrEmptyToolResultCallID.
//
// Validate is for re-validation of values that may have been mutated
// across serialization boundaries. The constructor is the sanctioned path
// for new values. Per AI-08 spec § B req "Validate re-runs structural checks".
func (t ToolResult) Validate() error {
	if t.callID == "" || isAllWhitespace(t.callID) {
		return ErrEmptyToolResultCallID
	}
	if len(t.callID) > MaxToolResultCallIDLength {
		return ErrToolResultCallIDTooLong
	}
	if err := validateToolResultContent(t.content); err != nil {
		return err
	}
	return nil
}

// validateToolResultContent enforces the structural rules for ToolResult
// content. Returns nil for valid content; otherwise the typed sentinel.
// Order: nil/empty → parse → length. Per AI-08 spec § F.
func validateToolResultContent(content json.RawMessage) error {
	if len(content) == 0 {
		return ErrMalformedToolResultContent
	}
	var parsed any
	if err := json.Unmarshal(content, &parsed); err != nil {
		return ErrMalformedToolResultContent
	}
	if len(content) > MaxToolResultContentLength {
		return ErrToolResultContentTooLong
	}
	return nil
}

// NewToolResult validates and constructs a ToolResult. On any validation
// failure it returns the zero-value ToolResult and the corresponding
// typed sentinel error. Validation order is callID → content (state-first,
// matching the NewToolDeclaration and NewToolCall precedent). Per AI-08
// spec § B req "NewToolResult is the sanctioned constructor and returns
// zero-value on error".
//
// Validation rules:
//
//   - callID: non-empty after isAllWhitespace, len ≤ MaxToolResultCallIDLength.
//     The constructor does NOT trim the callID.
//   - content: non-nil, non-empty, parseable as JSON (any value),
//     len ≤ MaxToolResultContentLength.
//
// Per AI-08 spec § F.
func NewToolResult(callID string, content json.RawMessage) (ToolResult, error) {
	if callID == "" || isAllWhitespace(callID) {
		return ToolResult{}, ErrEmptyToolResultCallID
	}
	if len(callID) > MaxToolResultCallIDLength {
		return ToolResult{}, ErrToolResultCallIDTooLong
	}
	if err := validateToolResultContent(content); err != nil {
		return ToolResult{}, err
	}
	return ToolResult{callID: callID, content: content}, nil
}
