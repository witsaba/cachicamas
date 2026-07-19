package ai

import (
	"errors"
	"strings"
)

// FinishReason is the provider-neutral termination signal on a model
// completion. AI-10 commits a 6-value canonical set that matches AI-02's
// REQUIRED capabilities (Completion reason row 3 in the capability matrix):
// the model either reported one of the five named outcomes or the upstream
// adapter could not classify the wire signal (Unknown).
//
// FinishReason is intentionally a typed string (not int) so AI-11 can
// serialize it as a JSON string in the event envelope without a custom
// MarshalJSON. See AI-10 design #1.
//
// Wire-format note: marshaling/unmarshaling is owned by AI-11 (event
// envelope). FinishReason exposes no MarshalJSON / UnmarshalJSON /
// MarshalText / UnmarshalText on its own.
//
// Vocabulary: see AI-00 § finish reason.
type FinishReason string

const (
	// FinishReasonStop means the model reached a natural stopping point
	// (end-of-turn, stop sequence, or vendor-equivalent such as Anthropic's
	// end_turn and OpenAI's stop).
	FinishReasonStop FinishReason = "stop"

	// FinishReasonLength means the model exhausted the output token budget
	// before reaching a natural stopping point (vendor-equivalents:
	// max_tokens, length, max_output_tokens).
	FinishReasonLength FinishReason = "length"

	// FinishReasonToolCall means the model emitted at least one tool call
	// instead of a final text response. Layer 2 (agent) routes the call.
	FinishReasonToolCall FinishReason = "tool_call"

	// FinishReasonContentFilter means the model refused to comply or a
	// vendor safety filter blocked the response. This single canonical
	// value covers both Anthropic's refusal and OpenAI's content_filter
	// because they are semantically equivalent for Layer 2 consumption;
	// the per-vendor raw signal is logged internally by the adapter
	// (AI-29) and never crosses the Layer 1 boundary upward.
	FinishReasonContentFilter FinishReason = "content_filter"

	// FinishReasonCancellation means the model itself reported that the
	// generation ended due to cancellation. This is the value-level sense
	// only; transport-level cancellation (ctx.Done(), stream close) is
	// owned by AI-01 § Cancellation contract event/error semantics and
	// MUST NOT be reported as a FinishReason. See AI-10 spec §
	// "Requirement: Cancellation meaning".
	FinishReasonCancellation FinishReason = "cancellation"

	// FinishReasonUnknown is the safe fallback when an adapter encounters
	// a vendor finish signal it cannot map to a canonical outcome (empty
	// raw, unrecognized string, vendor-specific variant). FinishReasonFromProvider
	// returns this value without error and without exposing the raw input
	// upward — the raw signal is logged by the adapter for diagnostics.
	// Per AI-10 design #3 and milestone doc line 223 (no provider wire
	// leakage upward).
	FinishReasonUnknown FinishReason = "unknown"
)

// ErrInvalidFinishReason is reserved for callers that need to detect an
// invalid FinishReason value (e.g., a future inbound wire decoder).
// FinishReasonFromProvider does NOT return this error — unknown input
// collapses to FinishReasonUnknown. Per AI-10 design #3.
var ErrInvalidFinishReason = errors.New("ai: invalid finish reason")

// IsValid reports whether r is one of the 6 canonical FinishReason
// constants. The zero value (FinishReason("")) is NOT valid — zero values
// cannot silently become valid wire signals. Per AI-10 spec § "Requirement:
// Canonical outcomes".
func (r FinishReason) IsValid() bool {
	switch r {
	case FinishReasonStop,
		FinishReasonLength,
		FinishReasonToolCall,
		FinishReasonContentFilter,
		FinishReasonCancellation,
		FinishReasonUnknown:
		return true
	default:
		return false
	}
}

// String returns the normalized name of the FinishReason. Matches the
// Kind.String, Role.String, and ReasoningState.String precedents.
func (r FinishReason) String() string {
	return string(r)
}

// FinishReasonFromProvider maps a raw vendor finish signal to a canonical
// FinishReason. Unknown, empty, and unrecognized input collapse to
// FinishReasonUnknown with no error and no panic. The raw input is
// discarded — there is no RawReason() accessor, because AI-10 design #3
// and milestone doc line 223 forbid upward leakage of vendor wire types.
//
// The mapping is case-insensitive and trimmed; vendor-specific synonyms
// (e.g., Anthropic's "end_turn" and "stop_sequence", OpenAI's "tool_calls")
// collapse to the canonical 6-value set. The mapping table:
//
//	"stop" / "end_turn" / "stop_sequence"        → FinishReasonStop
//	"length" / "max_tokens" / "max_output_tokens" → FinishReasonLength
//	"tool_call" / "tool_use" / "tool_calls"      → FinishReasonToolCall
//	"content_filter" / "refusal" / "safety"      → FinishReasonContentFilter
//	"cancellation" / "cancelled" / "cancel"      → FinishReasonCancellation
//	empty, whitespace, or any other string         → FinishReasonUnknown
//
// The function is pure and deterministic: same input → same output, every
// call. Per AI-10 spec § "Requirement: Deterministic provider normalization".
func FinishReasonFromProvider(raw string) FinishReason {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "stop", "end_turn", "stop_sequence":
		return FinishReasonStop
	case "length", "max_tokens", "max_output_tokens":
		return FinishReasonLength
	case "tool_call", "tool_use", "tool_calls":
		return FinishReasonToolCall
	case "content_filter", "refusal", "safety":
		return FinishReasonContentFilter
	case "cancellation", "cancelled", "cancel":
		return FinishReasonCancellation
	default:
		return FinishReasonUnknown
	}
}
