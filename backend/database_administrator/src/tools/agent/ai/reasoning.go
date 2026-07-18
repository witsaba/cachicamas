package ai

import "errors"

// MaxReasoningSummaryLength is the upper bound on ReasoningRedacted
// payloads, in bytes. The value is 1 KiB (1024 bytes). Rationale: a
// redacted reasoning summary is a short, vendor-supplied recap that
// fits comfortably in a single network round-trip; we reject oversized
// summaries at the constructor to prevent OOM via malformed inputs.
// Per AI-06 spec § A "Requirement: ContentPartFromReasoning enforces
// per-variant payload rules".
const MaxReasoningSummaryLength = 1 << 10

// MaxReasoningStreamedLength is the upper bound on ReasoningStreamed
// payloads, in bytes. The value is 1 MiB (1,048,576 bytes). Rationale:
// a streamed reasoning payload is the full reasoning transcript; we
// bound it at 1 MiB to match MaxTextLength and prevent OOM via
// accidental huge streams. Per AI-06 spec § A req 2.
const MaxReasoningStreamedLength = 1 << 20

// ErrInvalidReasoningState is returned by ContentPartFromReasoning when
// the supplied state is outside the 3-variant canonical set. The state
// check runs FIRST so an unknown state cannot smuggle a payload past
// validation. Per AI-06 spec § A req 1.
var ErrInvalidReasoningState = errors.New("ai: invalid reasoning state")

// ErrEmptyReasoningStream is returned by ContentPartFromReasoning when
// the state is ReasoningStreamed and the payload is the empty string.
// ReasoningStreamed requires non-empty content per AI-06 spec § A req 2.
var ErrEmptyReasoningStream = errors.New("ai: empty reasoning stream")

// ErrWhitespaceReasoningStream is returned by ContentPartFromReasoning
// when the state is ReasoningStreamed and the payload is all-whitespace
// (every rune reports true for unicode.IsSpace). Reuses the same
// whitespace definition as NewText via isAllWhitespace. Per AI-06 spec
// § A req 2.
var ErrWhitespaceReasoningStream = errors.New("ai: whitespace-only reasoning stream")

// ErrReasoningSummaryTooLong is returned by ContentPartFromReasoning
// when the state is ReasoningRedacted and len(payload) > MaxReasoningSummaryLength.
// Per AI-06 spec § A req 2.
var ErrReasoningSummaryTooLong = errors.New("ai: reasoning summary exceeds MaxReasoningSummaryLength")

// ErrReasoningStreamTooLong is returned by ContentPartFromReasoning
// when the state is ReasoningStreamed and len(payload) > MaxReasoningStreamedLength.
// Per AI-06 spec § A req 2.
var ErrReasoningStreamTooLong = errors.New("ai: reasoning stream exceeds MaxReasoningStreamedLength")

// ReasoningState is the variant discriminator for the optional
// reasoning content part. Three canonical values: ReasoningAbsent
// (v1 default per AI-02 § Reasoning policy), ReasoningRedacted, and
// ReasoningStreamed. The type is intentionally a typed string (not int)
// so it serializes cleanly as a JSON string when AI-11 ships the event
// envelope. Per AI-06 spec § A req 1.
type ReasoningState string

const (
	// ReasoningAbsent signals "no reasoning was emitted by the model".
	// v1 Layer 1 adapters MUST only emit ReasoningAbsent per AI-02 §
	// Reasoning policy; AI-21 conformance asserts this and skips
	// Redacted/Streamed with reason citing AI-02 § Reasoning policy.
	ReasoningAbsent ReasoningState = "absent"

	// ReasoningRedacted carries a short vendor-supplied summary in
	// place of the raw reasoning transcript. The summary length is
	// bounded by MaxReasoningSummaryLength. Reserved for v1.1+ per
	// AI-02 § Reasoning policy (v1 emits only Absent).
	ReasoningRedacted ReasoningState = "redacted"

	// ReasoningStreamed carries the full reasoning transcript as it
	// arrives from the model. The text length is bounded by
	// MaxReasoningStreamedLength and MUST be non-empty and non-whitespace.
	// Reserved for v1.1+ per AI-02 § Reasoning policy.
	ReasoningStreamed ReasoningState = "streamed"
)

// IsValid reports whether s is one of the 3 canonical ReasoningState
// constants. The zero value ReasoningState("") is NOT valid — zero
// values cannot silently become valid wire values (AI-04 invariant
// extended to ReasoningState). Per AI-06 spec § A req 1.
func (s ReasoningState) IsValid() bool {
	switch s {
	case ReasoningAbsent, ReasoningRedacted, ReasoningStreamed:
		return true
	default:
		return false
	}
}

// String returns the wire-format name of the ReasoningState (the
// underlying string). Matches the Kind.String and Role.String
// precedents.
func (s ReasoningState) String() string {
	return string(s)
}

// Reasoning is the provider-neutral value type for an optional
// reasoning content part. See AI-00 § content part (Layer 1 sense)
// and AI-06 § Reasoning for the v1.1 design.
//
// Per design-resolution obs #2059 and AI-06 spec rev 2 § A req 3,
// Reasoning IS the ContentPart: there is NO unexported wrapper
// (reasoningPart) layered between Reasoning and the ContentPart
// interface. This guarantees Layer 2 inspectability via
// part.(ai.Reasoning), which the wrapper pattern would break (the
// runtime type would be reasoningPart, not Reasoning).
//
// The fields are unexported so the constructor is the only path that
// produces valid values: a literal Reasoning{} yields ReasoningState(""),
// which IsValid rejects. Reasoning is struct-valued (not a pointer) so
// var r Reasoning is a non-nil interface value and the standard
// part == nil idiom catches interface-nil literals without a typed-nil
// false positive. Message.Validate's interface-nil check works.
//
// The constructor ContentPartFromReasoning validates state first, then
// payload per the variant matrix in AI-06 spec § A req 2.
type Reasoning struct {
	state ReasoningState
	text  string
}

// Compile-time assertion: Reasoning directly implements ContentPart.
// If the signature drifts (e.g., a wrapper is reintroduced), this line
// fails to compile.
var _ ContentPart = Reasoning{}

// Kind returns KindReasoning. The discriminator is fixed by the
// Reasoning value type itself; there is no way to construct a Reasoning
// with a different Kind. Per AI-06 spec § A req 3.
func (r Reasoning) Kind() Kind {
	return KindReasoning
}

// State returns the ReasoningState variant (Absent / Redacted /
// Streamed). The accessor exposes the variant for Layer 2 inspection
// without vendor metadata, model tags, or provider identifiers — only
// the 3 canonical state strings pass through. Per AI-06 spec § A req 3.
func (r Reasoning) State() ReasoningState {
	return r.state
}

// Text returns the payload carried by this Reasoning: "" for Absent,
// the summary for Redacted, the streamed transcript for Streamed. The
// accessor exposes exactly what the caller passed to
// ContentPartFromReasoning (or, for a literal zero-value Reasoning,
// the empty string). No vendor metadata passes through. Per AI-06 spec
// § A req 3.
func (r Reasoning) Text() string {
	return r.text
}

// ContentPartFromReasoning is the sanctioned constructor for an
// optional reasoning ContentPart. It validates state first, then
// payload per the variant matrix:
//
//	ReasoningAbsent   → payload ignored, always valid
//	ReasoningRedacted → len(payload) ≤ MaxReasoningSummaryLength (1 KiB)
//	ReasoningStreamed → non-empty, !all-whitespace, len ≤ MaxReasoningStreamedLength (1 MiB)
//	any other state   → ErrInvalidReasoningState (state check runs first)
//
// On any error the returned ContentPart is nil and the corresponding
// typed sentinel is returned (compatible with errors.Is). The returned
// ContentPart's runtime type is Reasoning (no wrapper), so Layer 2 can
// recover the variant + payload via part.(ai.Reasoning).
//
// Per AI-06 spec § A reqs 2 and 3.
func ContentPartFromReasoning(state ReasoningState, payload string) (ContentPart, error) {
	if !state.IsValid() {
		return nil, ErrInvalidReasoningState
	}
	switch state {
	case ReasoningAbsent:
		// Payload ignored — the variant IS the information.
		return Reasoning{state: state, text: payload}, nil
	case ReasoningRedacted:
		if len(payload) > MaxReasoningSummaryLength {
			return nil, ErrReasoningSummaryTooLong
		}
		return Reasoning{state: state, text: payload}, nil
	case ReasoningStreamed:
		if payload == "" {
			return nil, ErrEmptyReasoningStream
		}
		if isAllWhitespace(payload) {
			return nil, ErrWhitespaceReasoningStream
		}
		if len(payload) > MaxReasoningStreamedLength {
			return nil, ErrReasoningStreamTooLong
		}
		return Reasoning{state: state, text: payload}, nil
	default:
		// Unreachable: IsValid() above rejected anything outside the
		// canonical 3-variant set. Defensive fallback for forward
		// compatibility if a future variant is added without updating
		// this switch.
		return nil, ErrInvalidReasoningState
	}
}
