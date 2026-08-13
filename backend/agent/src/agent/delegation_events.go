// AG-06.3 — the delegation family (R-APE-006, VL2-EVT-08).
//
// Delegation events record a subagent's lifecycle inside a parent
// run: [NewSubagentStarted] opens the subagent's stream (parent-linked
// via the envelope's [Event.Parent]) and [NewSubagentEnded] closes it.
// Both payloads carry the subagent identity; the parent identifier is
// the same one AG-04.1 reserved on the envelope — AG-06.3 is the
// FIRST non-[NewDelegatedRunStart] consumer of it (`event.go:362-366`,
// R-AEV-003 direction-2).
//
// # Parent identifier is envelope-side (R-AEV-003)
//
// The parent field lives on [Event] itself, not on the payload — the
// same envelope shape AG-04.1 reserved for [NewDelegatedRunStart]. The
// validators and consumers read it through [Event.Parent], which
// reports (RunID, hasParent); the absence is distinguishable from a
// present-but-zero value.
//
// # Per-family file split (AD-6, AG-05 precedent)
//
// Payload + ctors + kind constants + EventDescriptor rows are
// co-located, mirroring message_text.go and tool_event.go. The same
// package owns the EventKind block (event.go) and the eventRegistry
// rows — this file's constants are referenced by event.go's iota
// continuation; a new kind declared here is wired through by the same
// commit.

package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// SubagentStarted is the payload opening a subagent's stream inside a
// delegated run (VL2-EVT-08, R-APE-006). It carries the subagent
// identity — enough for the session log to reconstruct the
// delegation tree.
type SubagentStarted struct {
	subagentID string
}

func (SubagentStarted) kind() EventKind { return EventKindSubagentStarted }

// validate reports SubagentStarted's own broken rule, or nil: the
// subagent identity is required.
func (s SubagentStarted) validate(at ai.Path) *ai.Violation {
	if s.subagentID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("subagent_id"))...)
	}
	return nil
}

// NewSubagentStarted constructs the event opening a subagent's stream
// inside a delegated run (R-APE-006). run is the subagent's OWN run
// identity; parent is the delegating run's identity — the constructor
// sets the envelope's parent identifier to it (the AG-06.3-first
// non-AG-04.1 consumer of the parent field, `event.go:362-366`,
// R-AEV-003 direction-2). subagentID is required. On failure the zero
// [Event] is returned.
func NewSubagentStarted(run, parent RunID, turn TurnID, subagentID string) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if parent == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("parent"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	if subagentID == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("subagent_id"))
	}
	payload := SubagentStarted{subagentID: subagentID}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{
		payload:   payload,
		run:       run,
		turn:      turn,
		hasTurn:   true,
		parent:    parent,
		hasParent: true,
	}, nil
}

// SubagentStarted returns the event's subagent-started payload, and
// whether the event carries one. Called on an event of another kind,
// or on an event that was never constructed, it returns the zero
// value and false, and never panics.
func (e Event) SubagentStarted() (SubagentStarted, bool) {
	payload, ok := e.payload.(SubagentStarted)
	return payload, ok
}

// SubagentID returns the subagent's identity (the leaf's own identity
// inside the delegation tree).
func (s SubagentStarted) SubagentID() string { return s.subagentID }

// String renders the payload for a diagnostic reader, naming the type
// and the subagent identity.
func (s SubagentStarted) String() string {
	return "subagent_started(" + strconv.Quote(s.subagentID) + ")"
}

// GoString renders the payload for the %#v verb.
func (s SubagentStarted) GoString() string { return s.String() }

// SubagentEnded is the payload closing a subagent's stream inside a
// delegated run (VL2-EVT-08, R-APE-006). It carries the subagent
// identity — the same identity its opener carried.
type SubagentEnded struct {
	subagentID string
}

func (SubagentEnded) kind() EventKind { return EventKindSubagentEnded }

// validate reports SubagentEnded's own broken rule, or nil: the
// subagent identity is required.
func (s SubagentEnded) validate(at ai.Path) *ai.Violation {
	if s.subagentID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("subagent_id"))...)
	}
	return nil
}

// NewSubagentEnded constructs the event closing a subagent's stream
// inside a delegated run (R-APE-006). run is the subagent's OWN run
// identity; parent is the delegating run's identity — the constructor
// sets the envelope's parent identifier to it. subagentID is
// required. On failure the zero [Event] is returned.
func NewSubagentEnded(run, parent RunID, turn TurnID, subagentID string) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if parent == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("parent"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	if subagentID == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("subagent_id"))
	}
	payload := SubagentEnded{subagentID: subagentID}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{
		payload:   payload,
		run:       run,
		turn:      turn,
		hasTurn:   true,
		parent:    parent,
		hasParent: true,
	}, nil
}

// SubagentEnded returns the event's subagent-ended payload, and
// whether the event carries one. Called on an event of another kind,
// or on an event that was never constructed, it returns the zero
// value and false, and never panics.
func (e Event) SubagentEnded() (SubagentEnded, bool) {
	payload, ok := e.payload.(SubagentEnded)
	return payload, ok
}

// SubagentID returns the subagent's identity the event closes.
func (s SubagentEnded) SubagentID() string { return s.subagentID }

// String renders the payload for a diagnostic reader, naming the type
// and the subagent identity.
func (s SubagentEnded) String() string {
	return "subagent_ended(" + strconv.Quote(s.subagentID) + ")"
}

// GoString renders the payload for the %#v verb.
func (s SubagentEnded) GoString() string { return s.String() }
