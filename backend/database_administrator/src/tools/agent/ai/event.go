// Package-level note for the AI-11 event envelope (event.go).
//
// This file implements the event envelope defined by milestone AI-11
// of the cachicamas_ai Layer 1 contract. See ADR 0004 for the
// layer-boundary rule (cachicamas_coding -> cachicamas_agent ->
// cachicamas_ai; this package may import only the Go standard library
// and vendor SDKs) and AI-01 for the streaming contract that
// every ModelProvider producer-owned <-chan Event must satisfy
// (closure semantics, cancellation contract, error delivery).
//
// Event is the value type that crosses the Layer 1 boundary. It is
// NOT a pointer, NOT a generic carrier (`any` / `json.RawMessage`
// are forbidden by the AI-11 vendor-leak guard), and NOT a wire
// format — AI-16+ owns marshaling. The payload is sealed via the
// unexported aiPayload() marker on eventPayload, so external packages
// cannot construct a payload implementation; only this package can.
//
// Ordering invariants (documented fully in doc.go's AI-11 paragraph):
//   - Exactly one EventKindResponseStart per producer's stream
//     (the first event on a fresh producer carries sequence 1).
//     Sequence ordering across producers is not enforced; AI-20/AI-21
//     own cross-producer conformance.
//   - Sequence is 1-based, producer-assigned, contiguous.
//   - At most one EventKindResponseComplete per stream, mutually
//     exclusive with EventKindError.
//   - AI-01 cancellation is best-effort: a cancelled stream may close
//     without any terminal event.

package ai

import (
	"errors"
	"sync/atomic"
)

// EventKind is the provider-neutral discriminator for AI-11 stream events.
// Every Event crossing the Layer 1 boundary carries one of the 12 reserved
// EventKind constants declared below. AI-12 through AI-15 add concrete
// payload implementations; AI-18 may add additional payload kinds. AI-11
// reserves the slot names only — adding a payload does not change this
// file.
//
// EventKind is intentionally a typed string (not int) so AI-16+ can
// serialize it as a JSON string in the event envelope without a custom
// MarshalJSON. The zero value EventKind("") is NOT valid; see IsValid.
//
// Wire-format note: marshaling/unmarshaling is owned by the envelope
// codec layer (AI-16+). EventKind exposes no MarshalJSON / UnmarshalJSON
// of its own.
//
// See AI-00 § event, § provider, and § tool call for the vocabulary,
// and AI-11 § "Requirement: ai-event-envelope-kind-typed-string" for the
// canonical set.
type EventKind string

const (
	// EventKindResponseStart marks the first event of every successful
	// stream. Exactly one response.start is required per stream; the
	// ordering rules in doc.go document this invariant.
	EventKindResponseStart EventKind = "response.start"

	// EventKindResponseComplete marks the successful terminal event.
	// At most one response.complete may appear per stream, and it MUST
	// NOT coexist with EventKindError. AI-01 § Cancellation contract
	// governs when this event is omitted (cancellation best-effort).
	EventKindResponseComplete EventKind = "response.complete"

	// EventKindTextStart marks the opening of a text content stream.
	// AI-13 owns the text payload.
	EventKindTextStart EventKind = "text.start"

	// EventKindTextDelta carries an incremental text chunk. AI-13
	// owns the delta payload shape (string content + index).
	EventKindTextDelta EventKind = "text.delta"

	// EventKindTextEnd marks the closing of a text content stream.
	// AI-13 owns the end payload.
	EventKindTextEnd EventKind = "text.end"

	// Reasoning-event kinds — reserved-but-unsupported in v1.
	// ---------------------------------------------------------------
	// The three EventKindReasoning* constants below are reserved for a
	// future AI-XX milestone that adds a provider emitting ReasoningRedacted
	// or ReasoningStreamed. In v1, AI-06 § Reasoning policy locks adapter
	// emission to ReasoningAbsent (see AI-06 spec #2057 § v1 emits only
	// ReasoningAbsent), so no ReasoningStart/Delta/End payloads exist;
	// AI-14 design documents this as an explicit no-op contract (see
	// AI-14 spec #2204 REQ-AI14-5). AI-21 conformance skips reasoning
	// payload cases with reason citing "see AI-02 § Reasoning policy".
	// The EventKind values and wire strings stay in the canonical
	// AllEventKinds() registry so future provider enablement is additive
	// (one new payload type per reserved slot, no registry change).

	// EventKindReasoningStart marks the opening of a reasoning
	// content stream. Reserved for a future AI-XX; v1 has no payload
	// implementation (AI-14 design, AI-06 § Reasoning policy).
	EventKindReasoningStart EventKind = "reasoning.start"

	// EventKindReasoningDelta carries an incremental reasoning chunk.
	// Reserved for a future AI-XX; v1 has no payload implementation
	// (AI-14 design, AI-06 § Reasoning policy).
	EventKindReasoningDelta EventKind = "reasoning.delta"

	// EventKindReasoningEnd marks the closing of a reasoning content
	// stream. Reserved for a future AI-XX; v1 has no payload
	// implementation (AI-14 design, AI-06 § Reasoning policy).
	EventKindReasoningEnd EventKind = "reasoning.end"

	// EventKindToolCallStart marks the opening of a tool-call stream.
	// AI-15 owns the tool-call payload (name + argument json.RawMessage).
	EventKindToolCallStart EventKind = "tool_call.start"

	// EventKindToolCallDelta carries an incremental tool-call chunk
	// (typically a partial JSON argument fragment). AI-15 owns the
	// delta payload.
	EventKindToolCallDelta EventKind = "tool_call.delta"

	// EventKindToolCallEnd marks the closing of a tool-call stream.
	// AI-15 owns the end payload (final assembled arguments).
	EventKindToolCallEnd EventKind = "tool_call.end"

	// EventKindError marks a terminal failure. AI-18 owns the error
	// payload (vendor-neutral error code + retryable flag + raw
	// diagnostic). EventKindError is mutually exclusive with
	// EventKindResponseComplete.
	EventKindError EventKind = "error"
)

// ErrEventKindUnregistered is returned by Event.Validate when the
// EventKind is the zero value or any value outside the 12-slot canonical
// set declared above. NewEvent does not return this error — it derives
// the kind from the payload and stamps it; Validate is the validation
// boundary.
var ErrEventKindUnregistered = errors.New("ai: event kind is not registered")

// ErrEventPayloadKindMismatch is returned by Event.Validate when
// Event.Kind does not match payload.Kind(). NewEvent guarantees parity
// by stamping Event.Kind from payload.Kind(); only direct struct
// construction (bypassing NewEvent) can produce a mismatch, and
// Validate catches it.
var ErrEventPayloadKindMismatch = errors.New("ai: event payload kind does not match Event.Kind")

// ErrEventPayloadMissing is returned by NewEvent when the payload is
// nil (interface-nil literal). Event.Validate also returns this error
// for directly-constructed Events with a nil Payload.
var ErrEventPayloadMissing = errors.New("ai: event payload is required")

// IsValid reports whether k is one of the 12 canonical EventKind
// constants. The zero value (EventKind("")) is NOT valid — zero values
// cannot silently become valid wire kinds. Per AI-11 spec §
// "Requirement: ai-event-envelope-kind-typed-string".
func (k EventKind) IsValid() bool {
	switch k {
	case EventKindResponseStart,
		EventKindResponseComplete,
		EventKindTextStart,
		EventKindTextDelta,
		EventKindTextEnd,
		EventKindReasoningStart,
		EventKindReasoningDelta,
		EventKindReasoningEnd,
		EventKindToolCallStart,
		EventKindToolCallDelta,
		EventKindToolCallEnd,
		EventKindError:
		return true
	default:
		return false
	}
}

// String returns the wire-format name of the EventKind (the underlying
// string). Matches the Role.String, FinishReason.String, Kind.String
// precedent set by AI-04 / AI-05 / AI-10.
func (k EventKind) String() string {
	return string(k)
}

// eventPayload is the sealed interface every AI-11 event payload must
// implement. Only types in this package can implement it: the marker
// method aiPayload() is unexported, so external packages cannot define
// a value satisfying the interface. This is the seal that makes the
// "Payload/Kind mismatch is unrepresentable" guarantee work — every
// payload implementation returns its own EventKind via Kind(), and
// NewEvent derives Event.Kind from payload.Kind() so callers cannot
// supply a mismatched (Kind, Payload) pair.
//
// AI-12, AI-13, AI-14, AI-15, and AI-18 each add concrete unexported
// payload types implementing this interface (one struct per kind slot,
// with the per-kind fields exposed via accessor methods). AI-11 itself
// does NOT add payload implementations — that work belongs to the
// follow-on milestones.
type eventPayload interface {
	aiPayload()
	Kind() EventKind
}

// Event is the provider-neutral envelope crossing the Layer 1 boundary
// (see ADR 0004). Every ModelProvider producer-owned <-chan Event
// defined by AI-01 carries values of this type. Event is value-typed
// (not pointer-typed) and comparable; producers send values, consumers
// receive copies.
//
// Per AI-11 spec § "Requirement: ai-event-envelope-event-value".
type Event struct {
	// Kind is the discriminator. NewEvent stamps this from
	// payload.Kind(); Validate refuses zero/unregistered values and
	// payload/Kind mismatches.
	Kind EventKind

	// Sequence is the producer-assigned 1-based contiguous ordering
	// number. NewEvent increments the package-private atomic counter
	// and stamps the result here. Consumers can detect dropped events
	// by inspecting gaps in Sequence.
	Sequence uint64

	// Payload is the sealed per-kind value. The concrete payload type
	// is determined by Kind (AI-12..15, AI-18). Payload is the
	// eventPayload interface — only types in this package can implement
	// it (sealed marker aiPayload()).
	Payload eventPayload
}

// sequenceCounter is the package-private atomic counter driving
// Sequence. It starts at 0 (Go zero value of uint64) and is incremented
// atomically by NewEvent. The counter is process-scoped; each NewEvent
// call advances it by exactly one.
//
// Per AI-11 design decision #3: producer-assigned, 1-based, contiguous.
// Per-producer isolation is the responsibility of the caller (one
// producer owns one counter for the lifetime of its stream); this
// package exposes the counter via the package-private symbol so that
// follow-on AI-20 (stream testkit) and AI-21 (conformance suite) can
// iterate it deterministically.
var sequenceCounter atomic.Uint64

// NewEvent constructs an Event with the next Sequence from the
// package-private atomic counter, derives Kind from the payload's
// Kind() method, and stores the payload verbatim. Kind/payload
// mismatch is unrepresentable because Kind is derived, not supplied.
//
// NewEvent returns ErrEventPayloadMissing when p is the interface-nil
// literal. It does NOT validate the resulting Kind — that is the job
// of Event.Validate, which catches zero/unregistered kinds and any
// payload/Kind mismatch introduced by direct struct construction.
//
// Per AI-11 spec § "Scenario: Construct an event with kind + payload".
func NewEvent(p eventPayload) (Event, error) {
	if p == nil {
		return Event{}, ErrEventPayloadMissing
	}
	seq := sequenceCounter.Add(1)
	return Event{
		Kind:     p.Kind(),
		Sequence: seq,
		Payload:  p,
	}, nil
}

// Validate re-runs the validation rules against the Event in canonical
// order:
//
//  1. Payload is non-nil                              → ErrEventPayloadMissing
//  2. Kind is non-zero AND registered                 → ErrEventKindUnregistered
//  3. Payload.Kind() matches Event.Kind               → ErrEventPayloadKindMismatch
//
// Validate is idempotent and pure. The zero-value Event{} fails step
// 1 with ErrEventPayloadMissing (the Payload field is the interface-
// nil literal).
//
// Per AI-11 spec § "Requirement: ai-event-envelope-event-value" and
// "Requirement: ai-event-envelope-sealed-payload".
func (e Event) Validate() error {
	if e.Payload == nil {
		return ErrEventPayloadMissing
	}
	if !e.Kind.IsValid() {
		return ErrEventKindUnregistered
	}
	if e.Kind != e.Payload.Kind() {
		return ErrEventPayloadKindMismatch
	}
	return nil
}

// AllEventKinds returns a defensive copy of the canonical registry in
// registration order. AI-20 (stream testkit) and AI-21 (conformance
// suite) iterate this list to assert exhaustiveness: every reserved
// EventKind has a payload implementation registered in some future
// AI-12..15, AI-18 milestone.
//
// The return value is a copy, so callers may mutate or sort it without
// affecting the canonical registry. Per AI-11 design decision #4.
func AllEventKinds() []EventKind {
	out := make([]EventKind, len(allKinds))
	copy(out, allKinds)
	return out
}

// allKinds is the package-private canonical registry of reserved
// EventKind values. AI-11 ships 12 entries; AI-12..15 and AI-18 do
// not modify this slice (adding payload implementations is
// non-modifying). The slice preserves registration order so AllEventKinds
// is deterministic across runs.
var allKinds = []EventKind{
	EventKindResponseStart,
	EventKindResponseComplete,
	EventKindTextStart,
	EventKindTextDelta,
	EventKindTextEnd,
	EventKindReasoningStart,
	EventKindReasoningDelta,
	EventKindReasoningEnd,
	EventKindToolCallStart,
	EventKindToolCallDelta,
	EventKindToolCallEnd,
	EventKindError,
}
