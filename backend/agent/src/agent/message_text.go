// AG-05.1 — the message text lifecycle family (R-AMT-001, R-AMT-003,
// R-AMT-004, VL2-EVT-04).
//
// The text bracket is the "model wrote something the user can read" bracket:
// it opens with [NewMessageStartText], carries deltas, and closes with
// [NewMessageEndText]. Reasoning has its own bracket, in its own file —
// reasoning fragments MUST NOT appear in a text payload, and that
// segregation is asserted by the absence of any "Reasoning" field on
// [MessageStartText] / [MessageDeltaText] / [MessageEndText], not by a
// runtime check inside validate() (R-AMT-001's kind-level segregation,
// S-AMT-002).
//
// # Per-family file split (AD-6)
//
// Payload + ctors + kind constants + EventDescriptor rows are
// co-located, mirroring AG-04's run_events.go precedent. The same package
// owns the EventKind block (event.go) and the eventRegistry rows, so this
// file's constants are referenced by event.go's `iota + N` continuation;
// a new kind declared here is wired through by the same commit.
//
// # Delta rule (R-AMT-003, joint with R-AEV-007)
//
// A delta carries an index and the new fragment only. The construction
// surface for [MessageDeltaText] takes a `fragment string` and an `idx
// uint32` — there is no field and no overload for an accumulated snapshot,
// which is the mechanism the no-snapshot-route bite (S-AMT-021,
// invariant_pin_test.go) proves by enumeration. Envelope invariant 1 is
// closed jointly by AG-04.3 + AG-05.1 (0003:2203).
//
// # Reconstruction (R-AMT-004)
//
// A message delivered whole (`start` + `end`) and an equivalent message
// delivered as deltas (`start` + N `delta` + `end`) reconstruct equally.
// The reconstruction helper lives in reconstruction_test.go (AG-05.3) and
// is bite-tested RED twice (S-AMT-071, S-AMT-072) before the property
// test (S-AMT-070) is GREEN.

package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// MessageStartText is the payload opening a text-message bracket
// (VL2-EVT-04, R-AMT-001). It carries the Layer 1 content identity of
// the message the bracket is about — readable from an external package
// per S-AGV-019, so a downstream consumer can correlate a text bracket
// to the message it brackets.
type MessageStartText struct {
	msgID ai.MessageID
}

func (MessageStartText) kind() EventKind { return EventKindMessageStartText }

// validate reports MessageStartText's own broken rule, or nil. Its
// envelope identity (run, turn) is validated by the constructor that
// receives it; the only payload-intrinsic rule is that the message
// identity is not the Layer 1 zero value.
func (m MessageStartText) validate(at ai.Path) *ai.Violation {
	if m.msgID.IsZero() {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("message_id"))...)
	}
	return nil
}

// NewMessageStartText constructs the event opening a text-message
// bracket (R-AMT-001). run and turn are required; msgID is required and
// MUST be the Layer 1 minted identity, not the zero value. On failure the
// zero [Event] is returned.
func NewMessageStartText(run RunID, turn TurnID, msgID ai.MessageID) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := MessageStartText{msgID: msgID}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// MessageStartText returns the event's message-start-text payload, and
// whether the event carries one. Called on an event of another kind, or
// on an event that was never constructed, it returns the zero
// [MessageStartText] and false, and never panics.
func (e Event) MessageStartText() (MessageStartText, bool) {
	payload, ok := e.payload.(MessageStartText)
	return payload, ok
}

// MessageID returns the Layer 1 minted identity of the message the
// bracket is about (R-AMT-001, S-AGV-019).
func (m MessageStartText) MessageID() ai.MessageID { return m.msgID }

// String renders the payload for a diagnostic reader, naming the type and
// the message identity — never the fragment's content.
func (m MessageStartText) String() string {
	return "message_start_text(" + m.msgID.String() + ")"
}

// GoString renders the payload for the %#v verb.
func (m MessageStartText) GoString() string { return m.String() }

// MessageDeltaText is the payload carrying one new fragment of a
// text-message bracket (VL2-EVT-04, R-AMT-003). It carries an index and
// the new fragment only — no field, no overload, and no companion
// constructor accepts an accumulated snapshot, which is the structural
// pin on the construction surface the no-snapshot-route bite
// (S-AMT-021) enumerates.
type MessageDeltaText struct {
	msgID    ai.MessageID
	idx      uint32
	fragment string
}

func (MessageDeltaText) kind() EventKind { return EventKindMessageDeltaText }

// validate reports MessageDeltaText's own broken rule, or nil. The
// fragment is required (an empty fragment is a no-op carrier, not a
// meaningful one), and the message identity MUST be the Layer 1 minted
// identity, not the zero value.
func (d MessageDeltaText) validate(at ai.Path) *ai.Violation {
	if d.msgID.IsZero() {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("message_id"))...)
	}
	if d.fragment == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("fragment"))...)
	}
	return nil
}

// NewMessageDeltaText constructs a text-message delta event (R-AMT-003).
// idx is the positional index of this delta within its message; fragment
// is the new text only, never an accumulated snapshot of what came
// before (R-AEV-007, joint with R-AEV-007 via 0003:2203). On failure the
// zero [Event] is returned.
func NewMessageDeltaText(run RunID, turn TurnID, msgID ai.MessageID, idx uint32, fragment string) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := MessageDeltaText{msgID: msgID, idx: idx, fragment: fragment}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// MessageDeltaText returns the event's message-delta-text payload, and
// whether the event carries one. Called on an event of another kind, or
// on an event that was never constructed, it returns the zero
// [MessageDeltaText] and false, and never panics.
func (e Event) MessageDeltaText() (MessageDeltaText, bool) {
	payload, ok := e.payload.(MessageDeltaText)
	return payload, ok
}

// MessageID returns the Layer 1 minted identity of the message the
// delta belongs to.
func (d MessageDeltaText) MessageID() ai.MessageID { return d.msgID }

// Index returns the delta's positional index within its message.
func (d MessageDeltaText) Index() uint32 { return d.idx }

// Fragment returns the delta's new fragment only — the no-snapshot pin
// (R-AEV-007, R-AMT-003) is the structural reason this accessor does
// not exist.
func (d MessageDeltaText) Fragment() string { return d.fragment }

// String renders the payload for a diagnostic reader, naming the type,
// the message identity, the index and the fragment — a redaction posture
// mirroring [RunEnd.String].
func (d MessageDeltaText) String() string {
	return "message_delta_text(" + d.msgID.String() + " idx=" + strconv.FormatUint(uint64(d.idx), 10) + " fragment=" + strconv.Quote(d.fragment) + ")"
}

// GoString renders the payload for the %#v verb.
func (d MessageDeltaText) GoString() string { return d.String() }

// MessageEndText is the payload closing a text-message bracket
// (VL2-EVT-04, R-AMT-001). It carries the Layer 1 content identity of
// the message the bracket is about — the same identity its opener
// carried.
type MessageEndText struct {
	msgID ai.MessageID
}

func (MessageEndText) kind() EventKind { return EventKindMessageEndText }

// validate reports MessageEndText's own broken rule, or nil: the
// message identity is not the Layer 1 zero value.
func (m MessageEndText) validate(at ai.Path) *ai.Violation {
	if m.msgID.IsZero() {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("message_id"))...)
	}
	return nil
}

// NewMessageEndText constructs the event closing a text-message bracket
// (R-AMT-001). run and turn are required; msgID MUST match the
// identity the matching [NewMessageStartText] carried. On failure the
// zero [Event] is returned.
func NewMessageEndText(run RunID, turn TurnID, msgID ai.MessageID) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := MessageEndText{msgID: msgID}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// MessageEndText returns the event's message-end-text payload, and
// whether the event carries one. Called on an event of another kind, or
// on an event that was never constructed, it returns the zero
// [MessageEndText] and false, and never panics.
func (e Event) MessageEndText() (MessageEndText, bool) {
	payload, ok := e.payload.(MessageEndText)
	return payload, ok
}

// MessageID returns the Layer 1 minted identity of the message the
// bracket is about.
func (m MessageEndText) MessageID() ai.MessageID { return m.msgID }

// String renders the payload for a diagnostic reader, naming the type
// and the message identity.
func (m MessageEndText) String() string {
	return "message_end_text(" + m.msgID.String() + ")"
}

// GoString renders the payload for the %#v verb.
func (m MessageEndText) GoString() string { return m.String() }
