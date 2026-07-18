package ai

import "errors"

// Kind is the discriminator for the ContentPart discriminated union.
// See AI-00 § content part and AI-05 § ContentPart interface.
//
// Kind is intentionally a typed string (not int) so it serializes
// cleanly as a JSON string when AI-11 ships the event envelope
// ("kind": "text", "kind": "reasoning", etc.) without requiring a
// custom MarshalJSON. See AI-05 § JSON deferral.
//
// The v1 union reserves 7 slots. Only KindText is constructed in v1;
// the other 6 are reserved for AI-06 (KindReasoning), AI-07
// (KindToolDeclaration), AI-08 (KindToolCall, KindToolResult), and
// future image/audio content per AI-02 § 10 deferral list.
//
// D-A from design resolution obs #2039: the enum exposes 7 slots
// (KindToolDeclaration was added vs. the 6-slot proposal #2037 to keep
// AI-07 additive).
type Kind string

const (
	// KindText is a UTF-8 text content part. Constructed by
	// ContentPartFromText in v1; see text.go for the value type.
	KindText Kind = "text"

	// KindReasoning is reserved by AI-05 for AI-06 (reasoning content).
	// v1 policy is ABSENT per AI-02 § 4; AI-17 must reject non-text
	// parts in v1 requests at the request validation level.
	KindReasoning Kind = "reasoning"

	// KindImage is reserved by AI-05 for the future image-input
	// capability per AI-02 § 10 (type variant reservation). v1 input
	// policy is TEXT-ONLY; the slot exists so AI-06 / image milestones
	// are additive one-line changes, not breaking refactors.
	KindImage Kind = "image"

	// KindAudio is reserved by AI-05 for the future audio-input
	// capability per AI-02 § 10. Same rationale as KindImage.
	KindAudio Kind = "audio"

	// KindToolDeclaration is reserved by AI-05 for AI-07 (tool
	// declarations — the assistant-role list of available tools
	// surfaced to the model in a request).
	KindToolDeclaration Kind = "tool_declaration"

	// KindToolCall is reserved by AI-05 for AI-08 (tool-call requests
	// emitted by the assistant in an assistant-role message).
	KindToolCall Kind = "tool_call"

	// KindToolResult is reserved by AI-05 for AI-08 (tool-result
	// content parts carried in a tool-role message that echoes the
	// result back to the model).
	KindToolResult Kind = "tool_result"
)

// String returns the wire-format name of the Kind (the underlying string).
// Matches the Role.String precedent set in AI-04.
func (k Kind) String() string {
	return string(k)
}

// IsValid reports whether k is one of the 7 canonical Kind constants.
// Unknown Kind values (including the zero value Kind("")) are rejected
// at validation boundaries; see AI-05 spec § A "Unknown Kind values are
// rejected at validation boundaries".
func (k Kind) IsValid() bool {
	switch k {
	case KindText, KindReasoning, KindImage, KindAudio, KindToolDeclaration, KindToolCall, KindToolResult:
		return true
	default:
		return false
	}
}

// ContentPart is the discriminated union of provider-neutral atomic
// message elements (text, reasoning, image, audio, tool declaration,
// tool call, tool result). See AI-00 § content part (provider-neutral
// union type) and AI-05 § ContentPart interface.
//
// The Kind() method is the stable discriminator that AI-11 (event
// envelope) and AI-24 (request translation) use to emit JSON without
// needing a ContentPart-level MarshalJSON in AI-05. Adding a new
// variant requires (a) one new const in the Kind block above,
// (b) one new unexported wrapper struct, and (c) one new exported
// constructor like ContentPartFromText. The interface itself never
// changes (additive only).
type ContentPart interface {
	Kind() Kind
}

// ErrNilContentPart is returned by Message.Validate when Content
// contains an interface-nil slot (the interface variable holds no
// concrete value). See AI-05 § Validation rules and message.go.
var ErrNilContentPart = errors.New("ai: nil content part in message")

// textPart is the sealed v1 implementation of ContentPart for text
// values. Unexported so callers MUST go through ContentPartFromText;
// this prevents third parties from constructing a ContentPart with
// a mismatched Kind(). See AI-05 § Construction ergonomics.
//
// Commit-2 GREEN note: the underlying field is `v string` here as a
// placeholder. Commit 4 (GREEN text) introduces the ai.Text value type
// and wires ContentPartFromText to call NewText; in that commit the
// field is renamed to `t Text` and the constructor propagates
// validation errors. The discriminator (KindText) and the constructor
// signature are stable across that refactor.
type textPart struct {
	v string
}

// Kind returns KindText. The discriminator is fixed by the sealed
// wrapper; there is no way to construct a textPart with a different Kind.
func (p textPart) Kind() Kind {
	return KindText
}

// ContentPartFromText wraps a string in the ContentPart union. This is
// the only sanctioned v1 constructor for ContentPart; other
// constructors (ContentPartFromReasoning, ContentPartFromToolCall,
// etc.) will be added by AI-06, AI-07, AI-08 following the same
// unexported-wrapper + exported-constructor pattern.
//
// Commit-2 GREEN note: this implementation does not yet validate the
// input. Validation against NewText's empty / whitespace-only / max-
// length rules is wired in commit 4 (GREEN text) when ai.Text exists.
// The signature is stable across that refactor.
func ContentPartFromText(s string) (ContentPart, error) {
	return textPart{v: s}, nil
}
