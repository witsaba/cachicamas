// AG-05.1 — the message reasoning lifecycle family (R-AMT-002, VL2-EVT-04).
//
// Reasoning has its own bracket lifecycle, independent of text
// (R-AMT-002): a reasoning stream does not need text events to bracket
// it, and a text bracket does not carry reasoning fragments. Reasoning
// fragments MUST NOT appear in a text payload (S-AMT-002); text
// fragments MUST NOT appear in a reasoning payload (the converse, by
// the same kind-level segregation). Both segregations are structural:
// no reasoning field on [MessageStartText], no text field on
// [MessageStartReasoning].
//
// # Same shape as message_text.go
//
// Payload + ctors + accessors are co-located, mirroring message_text.go
// and run_events.go. Reasoning differs from text only in payload field
// naming and in the absence of any cross-kind accessor — the
// segregation is the absence of a route, not a runtime check.
//
// # Delta rule (R-AMT-003)
//
// Reasoning deltas carry an index and the new fragment only, exactly
// like text deltas. The same construction surface rule applies: no
// field, no overload, no companion constructor accepts an accumulated
// snapshot. Reasoning inherits the joint closure of envelope invariant
// 1 with AG-04.3 (0003:2203).

package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// MessageStartReasoning is the payload opening a reasoning-message
// bracket (VL2-EVT-04, R-AMT-002). It carries the Layer 1 content
// identity of the message the bracket is about — readable from an
// external package per S-AGV-019.
type MessageStartReasoning struct {
	msgID ai.MessageID
}

func (MessageStartReasoning) kind() EventKind { return EventKindMessageStartReasoning }

// validate reports MessageStartReasoning's own broken rule, or nil:
// the message identity is not the Layer 1 zero value.
func (m MessageStartReasoning) validate(at ai.Path) *ai.Violation {
	if m.msgID.IsZero() {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("message_id"))...)
	}
	return nil
}

// NewMessageStartReasoning constructs the event opening a reasoning-
// message bracket (R-AMT-002). run and turn are required; msgID is
// required and MUST be the Layer 1 minted identity, not the zero value.
// On failure the zero [Event] is returned.
func NewMessageStartReasoning(run RunID, turn TurnID, msgID ai.MessageID) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := MessageStartReasoning{msgID: msgID}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// MessageStartReasoning returns the event's message-start-reasoning
// payload, and whether the event carries one. Called on an event of
// another kind, or on an event that was never constructed, it returns
// the zero [MessageStartReasoning] and false, and never panics.
func (e Event) MessageStartReasoning() (MessageStartReasoning, bool) {
	payload, ok := e.payload.(MessageStartReasoning)
	return payload, ok
}

// MessageID returns the Layer 1 minted identity of the reasoning
// message the bracket is about (R-AMT-001, S-AGV-019).
func (m MessageStartReasoning) MessageID() ai.MessageID { return m.msgID }

// String renders the payload for a diagnostic reader, naming the type
// and the message identity.
func (m MessageStartReasoning) String() string {
	return "message_start_reasoning(" + m.msgID.String() + ")"
}

// GoString renders the payload for the %#v verb.
func (m MessageStartReasoning) GoString() string { return m.String() }

// MessageDeltaReasoning is the payload carrying one new fragment of a
// reasoning-message bracket (VL2-EVT-04, R-AMT-003). It carries an
// index and the new fragment only — the no-snapshot pin applies to
// reasoning deltas identically.
type MessageDeltaReasoning struct {
	msgID    ai.MessageID
	idx      uint32
	fragment string
}

func (MessageDeltaReasoning) kind() EventKind { return EventKindMessageDeltaReasoning }

// validate reports MessageDeltaReasoning's own broken rule, or nil.
// The fragment is required, and the message identity MUST be the Layer
// 1 minted identity, not the zero value.
func (d MessageDeltaReasoning) validate(at ai.Path) *ai.Violation {
	if d.msgID.IsZero() {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("message_id"))...)
	}
	if d.fragment == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("fragment"))...)
	}
	return nil
}

// NewMessageDeltaReasoning constructs a reasoning-message delta event
// (R-AMT-003). idx is the positional index; fragment is the new text
// only, never an accumulated snapshot (R-AEV-007, joint with AG-04.3
// via 0003:2203). On failure the zero [Event] is returned.
func NewMessageDeltaReasoning(run RunID, turn TurnID, msgID ai.MessageID, idx uint32, fragment string) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := MessageDeltaReasoning{msgID: msgID, idx: idx, fragment: fragment}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// MessageDeltaReasoning returns the event's message-delta-reasoning
// payload, and whether the event carries one. Called on an event of
// another kind, or on an event that was never constructed, it returns
// the zero [MessageDeltaReasoning] and false, and never panics.
func (e Event) MessageDeltaReasoning() (MessageDeltaReasoning, bool) {
	payload, ok := e.payload.(MessageDeltaReasoning)
	return payload, ok
}

// MessageID returns the Layer 1 minted identity of the reasoning
// message the delta belongs to.
func (d MessageDeltaReasoning) MessageID() ai.MessageID { return d.msgID }

// Index returns the delta's positional index within its reasoning
// message.
func (d MessageDeltaReasoning) Index() uint32 { return d.idx }

// Fragment returns the delta's new fragment only.
func (d MessageDeltaReasoning) Fragment() string { return d.fragment }

// String renders the payload for a diagnostic reader, naming the type,
// the message identity, the index and the fragment — a redaction
// posture mirroring [MessageDeltaText.String].
func (d MessageDeltaReasoning) String() string {
	return "message_delta_reasoning(" + d.msgID.String() + " idx=" + strconv.FormatUint(uint64(d.idx), 10) + " fragment=" + strconv.Quote(d.fragment) + ")"
}

// GoString renders the payload for the %#v verb.
func (d MessageDeltaReasoning) GoString() string { return d.String() }

// MessageEndReasoning is the payload closing a reasoning-message
// bracket (VL2-EVT-04, R-AMT-002). It carries the Layer 1 content
// identity of the reasoning message the bracket is about.
type MessageEndReasoning struct {
	msgID ai.MessageID
}

func (MessageEndReasoning) kind() EventKind { return EventKindMessageEndReasoning }

// validate reports MessageEndReasoning's own broken rule, or nil: the
// message identity is not the Layer 1 zero value.
func (m MessageEndReasoning) validate(at ai.Path) *ai.Violation {
	if m.msgID.IsZero() {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("message_id"))...)
	}
	return nil
}

// NewMessageEndReasoning constructs the event closing a reasoning-
// message bracket (R-AMT-002). run and turn are required; msgID MUST
// match the identity the matching [NewMessageStartReasoning] carried.
// On failure the zero [Event] is returned.
func NewMessageEndReasoning(run RunID, turn TurnID, msgID ai.MessageID) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := MessageEndReasoning{msgID: msgID}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// MessageEndReasoning returns the event's message-end-reasoning
// payload, and whether the event carries one. Called on an event of
// another kind, or on an event that was never constructed, it returns
// the zero [MessageEndReasoning] and false, and never panics.
func (e Event) MessageEndReasoning() (MessageEndReasoning, bool) {
	payload, ok := e.payload.(MessageEndReasoning)
	return payload, ok
}

// MessageID returns the Layer 1 minted identity of the reasoning
// message the bracket is about.
func (m MessageEndReasoning) MessageID() ai.MessageID { return m.msgID }

// String renders the payload for a diagnostic reader, naming the type
// and the message identity.
func (m MessageEndReasoning) String() string {
	return "message_end_reasoning(" + m.msgID.String() + ")"
}

// GoString renders the payload for the %#v verb.
func (m MessageEndReasoning) GoString() string { return m.String() }
