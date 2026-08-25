// CH-02.1 — the wire vocabulary: the five shapes the projector emits onto,
// mirroring frontend-chat-layer1's frozen wire contract (chat-types.ts:27-32,
// :36-43). R-CCP-003, R-CCP-006, R-CCP-007.

package chat

import "github.com/cachicamas/backend/agent/src/ai"

// WireEvent is the sealed interface every projected wire event implements.
// Membership is closed to this file's own five structs by the unexported
// marker method (design AD-3), so a switch elsewhere in this package can be
// exhaustive over the vocabulary without an external type ever joining it.
type WireEvent interface {
	isWireEvent()
}

// MessageStart is projected from Layer 2's message_start_text (R-CCP-003).
// Index is the archetype's per-message counter (R-CCP-005) — always 0 in a
// v1 single-message turn, never Layer 2's own per-fragment index.
type MessageStart struct {
	MessageID string
	Index     int
}

func (MessageStart) isWireEvent() {}

// MessageDelta is projected from Layer 2's message_delta_text (R-CCP-003).
// Index carries the same archetype-minted meaning as MessageStart's own
// field (R-CCP-005) — deliberately not Layer 2's own per-fragment idx (D8).
type MessageDelta struct {
	Index int
	Delta string
}

func (MessageDelta) isWireEvent() {}

// MessageEnd is projected from Layer 2's message_end_text (R-CCP-003).
// FinishReason is always "stop" in v1 (D7): MessageEndText carries no
// run-level reason, and "stop" is truthful at message granularity — the
// message ended because its text completed.
type MessageEnd struct {
	Index        int
	FinishReason string
}

func (MessageEnd) isWireEvent() {}

// TurnEnd is projected from Layer 2's run_end on a completed or interrupted
// run (R-CCP-006). FinishReason is present — derived from Harness.Run's own
// returned ai.FinishReason (D5) — for a completed run, and absent (a nil
// pointer) for a cancelled one: absence is the cancellation discriminator
// and MUST NOT be replaced by a minted "unknown".
type TurnEnd struct {
	FinishReason *string
}

func (TurnEnd) isWireEvent() {}

// Error is projected from Layer 2's run_end on a failed run (R-CCP-007).
// Kind is always "server" in v1 — validation/conflict/not_found belong to
// CH-03's HTTP layer, never minted here. Message is one of errortext.go's
// fixed, archetype-owned phrases — never provider-authored text (R-CCP-008,
// D6).
type Error struct {
	Kind    string
	Message string
}

func (Error) isWireEvent() {}

// CH-09 — tool-call wire projection (D-3, D-6).
//
// Four new variants on the closed WireEvent interface:
//
//   - ToolCallStart, ToolResult — emitted at v1 by the chat projector
//     (R-CTS-004, R-CTS-005). ToolCallStart carries the wire call id,
//     tool name, and arguments bytes; ToolResult carries the outcome
//     enum and either content bytes (success / result_failure) or a
//     typed failure category string (execution_failure, no provider
//     text — R-CCP-008 / D6 mirror).
//   - ToolCallDelta, ToolCallEnd — reserved-but-unused at v1 (D-6).
//     They keep the union closed against future progress-bearing
//     tools (a long-running MCP tool would need a 5th chat-side event);
//     no chat projector arm emits them at v1.
//
// All four carry `isWireEvent()` markers (mirroring the five pre-
// existing variants). The new variants trigger a compile error in
// `wireFrameName`'s default branch until T-04 finalizes the four cases
// — that is the strict-TDD RED scaffold pre-empting the GREEN.
type ToolCallStart struct {
	WireCallID string
	Tool       string
	Arguments  string
}

func (ToolCallStart) isWireEvent() {}

// ToolCallDelta is reserved for future progress-bearing tools (D-6 /
// NFR-CTS-002). v1 does not emit this variant; the wireFrameName case
// exists so a future long-running tool can land here without a wire
// shape change.
type ToolCallDelta struct {
	WireCallID string
	Delta      string
}

func (ToolCallDelta) isWireEvent() {}

// ToolCallEnd is reserved for v2 dynamic-source surfaces (D-6 /
// NFR-CTS-002). v1 collapses Layer 2's three ToolEnd* kinds into
// ToolResult; ToolCallEnd marks the future extension slot.
type ToolCallEnd struct {
	WireCallID string
	Outcome    string
}

func (ToolCallEnd) isWireEvent() {}

// ToolResult is projected from any of the three Layer 2 ToolEnd* kinds
// (R-CTS-005, D-6 collapse): Success → "success"; ResultFailure →
// "result_failure"; ExecutionFailure → "execution_failure" with the
// typed category string in FailureCategory and empty Content (no
// provider text leaks, R-CCP-008 / D6 mirror). FailureCategory is
// non-empty ONLY when Outcome == "execution_failure".
type ToolResult struct {
	WireCallID     string
	Tool           string
	Outcome        string
	Content        string
	FailureCategory string
}

func (ToolResult) isWireEvent() {}

// The wire's closed, seven-value finish-reason vocabulary (chat-types.ts:36-43).
const (
	FinishReasonStop          = "stop"
	FinishReasonLength        = "length"
	FinishReasonToolCalls     = "tool_calls"
	FinishReasonRefusal       = "refusal"
	FinishReasonPauseTurn     = "pause_turn"
	FinishReasonContentFilter = "content_filter"
	FinishReasonUnknown       = "unknown"
)

// finishReasonByAI maps ai.FinishReason 1:1 onto the wire's own vocabulary
// (D5, R-CCP-006).
var finishReasonByAI = map[ai.FinishReason]string{
	ai.FinishReasonStop:          FinishReasonStop,
	ai.FinishReasonLength:        FinishReasonLength,
	ai.FinishReasonToolCalls:     FinishReasonToolCalls,
	ai.FinishReasonRefusal:       FinishReasonRefusal,
	ai.FinishReasonPauseTurn:     FinishReasonPauseTurn,
	ai.FinishReasonContentFilter: FinishReasonContentFilter,
	ai.FinishReasonUnknown:       FinishReasonUnknown,
}

// finishReasonToWire maps r onto the wire's finish-reason vocabulary (D5).
// A value outside ai.FinishReason's own vocabulary — never produced by a
// completed Harness.Run — falls back to the wire's own "unknown" rather
// than panicking.
func finishReasonToWire(r ai.FinishReason) string {
	if wire, ok := finishReasonByAI[r]; ok {
		return wire
	}
	return FinishReasonUnknown
}
