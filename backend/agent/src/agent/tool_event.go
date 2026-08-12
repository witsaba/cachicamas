// AG-05.2 — the tool execution family (R-AMT-005, R-AMT-006, R-AMT-007,
// VL2-EVT-05).
//
// A tool call bracket opens with [NewToolStart] (carrying the call
// identity, the tool name, and the arguments — sufficient to render
// "running tool X with these arguments" live), carries indexed
// [NewToolProgress] events, and closes with one of three typed end
// outcomes — [NewToolEndSuccess], [NewToolEndResultFailure], or
// [NewToolEndExecutionFailure] — distinct by kind, not by convention
// over payload contents (R-AMT-006, S-AMT-050).
//
// # Call ordinal is payload-side (R-AMT-007, AD-3)
//
// The tool call ordinal correlates each event to the call's issuance
// order, independent of completion order. The ordinal is a payload
// field, NOT an envelope field — R-13 traces the ordinal to doc 0002
// AI-30 (Layer 1 payload-side), and the envelope's identity-shape
// remains untouched (sequence.go byte-unchanged, the AG-05 bet). The
// ordinal lives on the tool payload struct alongside idx (for
// progress) and result/failure (for the end variants), so the rule
// engine reads a uniform envelope and a typed ordinal from each
// payload.
//
// # Per-family file split (AD-6)
//
// Payload + ctors + kind constants + EventDescriptor rows are
// co-located, mirroring AG-04's run_events.go and AG-05.1's
// message_text.go / message_reasoning.go. The same package owns the
// EventKind block (event.go) and the eventRegistry table — this file's
// constants are referenced by event.go's iota continuation; a new kind
// declared here is wired through by the same commit.
//
// # W3 latent-trap guard
//
// All five tool kinds register PlacementTurn, BracketRoleNone,
// CardinalityAny, Terminal:false (R-AEV-012, W3 from AG-04). The
// six-step procedure documented in event_descriptor.go restates this
// explicitly.

package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ToolStart is the payload opening a tool call bracket (VL2-EVT-05,
// R-AMT-005). It carries the call identity, the tool name, and the
// argument bytes — sufficient to render "running tool X with these
// arguments" live. The call ordinal is payload-side (R-AMT-007, AD-3).
//
// Arguments are carried as []byte (the unexported shape json.RawMessage
// would alias to). Layer 2's no-I/O boundary (ADR 0005 § D1 row 2,
// import_boundary_test.go) forbids the encoding/json package because
// its transitive closure pulls in os and io/fs; Layer 2 therefore
// transports argument bytes opaquely and leaves their syntax check to
// the adapter that produces them (AI-09.1, V-REQ-17 — the same byte-
// fidelity rule, applied where the bytes are minted, not where they
// are forwarded).
type ToolStart struct {
	callID    string
	ordinal   uint32
	name      string
	arguments []byte
}

func (ToolStart) kind() EventKind { return EventKindToolStart }

// validate reports ToolStart's own broken rule, or nil. The call
// identity and the tool name are required (R-AMT-005); arguments
// MUST be present (non-empty), not necessarily well-formed JSON at
// Layer 2's boundary.
func (t ToolStart) validate(at ai.Path) *ai.Violation {
	if t.callID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("call_id"))...)
	}
	if t.name == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("name"))...)
	}
	if len(t.arguments) == 0 {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("arguments"))...)
	}
	return nil
}

// NewToolStart constructs the event opening a tool call bracket
// (R-AMT-005, R-AMT-007). ordinal is the call's issuance position
// within the run (R-AMT-007). arguments MUST be non-empty; Layer 2
// does not enforce JSON syntax (see [ToolStart] doc). On failure the
// zero [Event] is returned.
func NewToolStart(run RunID, turn TurnID, callID string, ordinal uint32, name string, arguments []byte) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := ToolStart{
		callID:    callID,
		ordinal:   ordinal,
		name:      name,
		arguments: append([]byte(nil), arguments...),
	}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// ToolStart returns the event's tool-start payload, and whether the
// event carries one. Called on an event of another kind, or on an event
// that was never constructed, it returns the zero [ToolStart] and false,
// and never panics.
func (e Event) ToolStart() (ToolStart, bool) {
	payload, ok := e.payload.(ToolStart)
	return payload, ok
}

// CallID returns the tool call's identity.
func (t ToolStart) CallID() string { return t.callID }

// Ordinal returns the tool call's issuance ordinal within the run
// (R-AMT-007, payload-side).
func (t ToolStart) Ordinal() uint32 { return t.ordinal }

// Name returns the tool's name.
func (t ToolStart) Name() string { return t.name }

// Arguments returns the tool call's argument bytes exactly as supplied.
func (t ToolStart) Arguments() []byte { return t.arguments }

// String renders the payload for a diagnostic reader, naming the type,
// the call identity, the ordinal, and the tool name — never the
// arguments' content.
func (t ToolStart) String() string {
	return "tool_start(" + t.callID + " ordinal=" + strconv.FormatUint(uint64(t.ordinal), 10) + " name=" + strconv.Quote(t.name) + ")"
}

// GoString renders the payload for the %#v verb.
func (t ToolStart) GoString() string { return t.String() }

// ToolProgress is the payload carrying one indexed progress update on a
// tool call (VL2-EVT-05, R-AMT-005). The integer index is positional
// within the call's progress stream; payload is the progress's own
// bytes (free-form, the tool's chosen shape).
type ToolProgress struct {
	callID  string
	ordinal uint32
	idx     uint32
	payload []byte
}

func (ToolProgress) kind() EventKind { return EventKindToolProgress }

// validate reports ToolProgress's own broken rule, or nil: the call
// identity is required.
func (t ToolProgress) validate(at ai.Path) *ai.Violation {
	if t.callID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("call_id"))...)
	}
	return nil
}

// NewToolProgress constructs a tool-progress event (R-AMT-005).
// idx is the positional index; payload is the progress's own bytes
// (free-form, the tool's chosen shape). ordinal correlates to the
// call's issuance position (R-AMT-007). On failure the zero [Event] is
// returned.
func NewToolProgress(run RunID, turn TurnID, callID string, ordinal uint32, idx uint32, payload []byte) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	p := ToolProgress{
		callID:  callID,
		ordinal: ordinal,
		idx:     idx,
		payload: append([]byte(nil), payload...),
	}
	if violation := p.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: p, run: run, turn: turn, hasTurn: true}, nil
}

// ToolProgress returns the event's tool-progress payload, and whether
// the event carries one. Called on an event of another kind, or on an
// event that was never constructed, it returns the zero [ToolProgress]
// and false, and never panics.
func (e Event) ToolProgress() (ToolProgress, bool) {
	payload, ok := e.payload.(ToolProgress)
	return payload, ok
}

// CallID returns the tool call's identity the progress belongs to.
func (t ToolProgress) CallID() string { return t.callID }

// Ordinal returns the tool call's issuance ordinal within the run
// (R-AMT-007).
func (t ToolProgress) Ordinal() uint32 { return t.ordinal }

// Index returns the progress's positional index within its call.
func (t ToolProgress) Index() uint32 { return t.idx }

// Payload returns the progress's own bytes.
func (t ToolProgress) Payload() []byte { return t.payload }

// String renders the payload for a diagnostic reader, naming the type,
// the call identity, the ordinal, and the index — never the payload's
// content.
func (t ToolProgress) String() string {
	return "tool_progress(" + t.callID + " ordinal=" + strconv.FormatUint(uint64(t.ordinal), 10) + " idx=" + strconv.FormatUint(uint64(t.idx), 10) + ")"
}

// GoString renders the payload for the %#v verb.
func (t ToolProgress) GoString() string { return t.String() }

// ToolOutcome is the typed end-state a tool call reaches (R-AMT-006,
// VL2-EVT-15). Three typed outcomes — distinct by kind, not by
// convention — share this enum: success, result-reports-failure, and
// execution-itself-failed. The zero value is not a member: each end
// constructor validates its outcome is a member of the closed
// vocabulary.
type ToolOutcome uint8

const (
	// ToolOutcomeSuccess is the outcome for a tool call that ran and
	// returned a successful result.
	ToolOutcomeSuccess ToolOutcome = iota + 1

	// ToolOutcomeResultFailure is the outcome for a tool call that ran
	// and returned a failure result (a value, not an error).
	ToolOutcomeResultFailure

	// ToolOutcomeExecutionFailure is the outcome for a tool call whose
	// execution itself failed — a typed [Failure] is required for this
	// outcome (R-AMT-006, S-AMT-051) and forbidden for the other two.
	ToolOutcomeExecutionFailure

	// toolOutcomeLimit is the first value outside the vocabulary. It
	// must stay last, mirroring runOutcomeLimit / turnOutcomeLimit.
	toolOutcomeLimit = iota
)

// String renders the outcome, or "unset"/"tooloutcome(N)" for a
// non-member, mirroring RunOutcome.String's posture.
func (o ToolOutcome) String() string {
	if o >= toolOutcomeLimit {
		return "tooloutcome(" + strconv.FormatUint(uint64(o), 10) + ")"
	}
	switch o {
	case ToolOutcomeSuccess:
		return "success"
	case ToolOutcomeResultFailure:
		return "result_failure"
	case ToolOutcomeExecutionFailure:
		return "execution_failure"
	case 0:
		return "unset"
	default:
		return "tooloutcome(" + strconv.FormatUint(uint64(o), 10) + ")"
	}
}

// ToolEndSuccess is the payload closing a tool call that succeeded
// (VL2-EVT-05, R-AMT-006). It carries the typed outcome
// [ToolOutcomeSuccess], the call's ordinal (R-AMT-007), and the
// successful result bytes.
type ToolEndSuccess struct {
	callID  string
	ordinal uint32
	result  []byte
}

func (ToolEndSuccess) kind() EventKind { return EventKindToolEndSuccess }

// validate reports ToolEndSuccess's own broken rule, or nil: the call
// identity is required.
func (t ToolEndSuccess) validate(at ai.Path) *ai.Violation {
	if t.callID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("call_id"))...)
	}
	return nil
}

// NewToolEndSuccess constructs the event closing a tool call that
// succeeded (R-AMT-006). result is the successful result bytes (free-
// form, the tool's chosen shape). On failure the zero [Event] is
// returned.
func NewToolEndSuccess(run RunID, turn TurnID, callID string, ordinal uint32, result []byte) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := ToolEndSuccess{
		callID:  callID,
		ordinal: ordinal,
		result:  append([]byte(nil), result...),
	}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// ToolEndSuccess returns the event's tool-end-success payload, and
// whether the event carries one. Called on an event of another kind,
// or on an event that was never constructed, it returns the zero
// [ToolEndSuccess] and false, and never panics.
func (e Event) ToolEndSuccess() (ToolEndSuccess, bool) {
	payload, ok := e.payload.(ToolEndSuccess)
	return payload, ok
}

// CallID returns the tool call's identity.
func (t ToolEndSuccess) CallID() string { return t.callID }

// Ordinal returns the tool call's issuance ordinal within the run
// (R-AMT-007).
func (t ToolEndSuccess) Ordinal() uint32 { return t.ordinal }

// Result returns the successful result bytes.
func (t ToolEndSuccess) Result() []byte { return t.result }

// Outcome returns [ToolOutcomeSuccess].
func (t ToolEndSuccess) Outcome() ToolOutcome { return ToolOutcomeSuccess }

// String renders the payload for a diagnostic reader, naming the type,
// the call identity, and the ordinal — never the result's content.
func (t ToolEndSuccess) String() string {
	return "tool_end_success(" + t.callID + " ordinal=" + strconv.FormatUint(uint64(t.ordinal), 10) + ")"
}

// GoString renders the payload for the %#v verb.
func (t ToolEndSuccess) GoString() string { return t.String() }

// ToolEndResultFailure is the payload closing a tool call that
// returned a failure result (VL2-EVT-05, R-AMT-006). It carries the
// typed outcome [ToolOutcomeResultFailure], the call's ordinal, and the
// failure-result bytes — distinct from a typed [Failure] by kind
// (R-AMT-006, S-AMT-050), not by payload convention.
type ToolEndResultFailure struct {
	callID  string
	ordinal uint32
	result  []byte
}

func (ToolEndResultFailure) kind() EventKind { return EventKindToolEndResultFailure }

// validate reports ToolEndResultFailure's own broken rule, or nil: the
// call identity is required.
func (t ToolEndResultFailure) validate(at ai.Path) *ai.Violation {
	if t.callID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("call_id"))...)
	}
	return nil
}

// NewToolEndResultFailure constructs the event closing a tool call
// whose result reports a failure (R-AMT-006). result carries the
// failure-result bytes (a value the tool returned, not an execution
// error). On failure the zero [Event] is returned.
func NewToolEndResultFailure(run RunID, turn TurnID, callID string, ordinal uint32, result []byte) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := ToolEndResultFailure{
		callID:  callID,
		ordinal: ordinal,
		result:  append([]byte(nil), result...),
	}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// ToolEndResultFailure returns the event's tool-end-result-failure
// payload, and whether the event carries one. Called on an event of
// another kind, or on an event that was never constructed, it returns
// the zero [ToolEndResultFailure] and false, and never panics.
func (e Event) ToolEndResultFailure() (ToolEndResultFailure, bool) {
	payload, ok := e.payload.(ToolEndResultFailure)
	return payload, ok
}

// CallID returns the tool call's identity.
func (t ToolEndResultFailure) CallID() string { return t.callID }

// Ordinal returns the tool call's issuance ordinal within the run
// (R-AMT-007).
func (t ToolEndResultFailure) Ordinal() uint32 { return t.ordinal }

// Result returns the failure-result bytes.
func (t ToolEndResultFailure) Result() []byte { return t.result }

// Outcome returns [ToolOutcomeResultFailure].
func (t ToolEndResultFailure) Outcome() ToolOutcome { return ToolOutcomeResultFailure }

// String renders the payload for a diagnostic reader, naming the type,
// the call identity, and the ordinal — never the result's content.
func (t ToolEndResultFailure) String() string {
	return "tool_end_result_failure(" + t.callID + " ordinal=" + strconv.FormatUint(uint64(t.ordinal), 10) + ")"
}

// GoString renders the payload for the %#v verb.
func (t ToolEndResultFailure) GoString() string { return t.String() }

// ToolEndExecutionFailure is the payload closing a tool call whose
// execution itself failed (VL2-EVT-05, R-AMT-006). It carries the
// typed outcome [ToolOutcomeExecutionFailure], the call's ordinal, and
// a typed [Failure] (R-AEV-008) — no code path assigns meaning to a
// message string.
type ToolEndExecutionFailure struct {
	callID  string
	ordinal uint32
	failure *Failure
}

func (ToolEndExecutionFailure) kind() EventKind { return EventKindToolEndExecutionFailure }

// validate reports ToolEndExecutionFailure's own broken rule, or nil:
// the call identity is required and the [Failure] is required iff the
// outcome is [ToolOutcomeExecutionFailure] — required for this
// outcome, mirroring run-end's iff rule (AD-2). A failure on this
// outcome without a [Failure], or on any other outcome carrying one,
// are both REJECTED.
func (t ToolEndExecutionFailure) validate(at ai.Path) *ai.Violation {
	if t.callID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("call_id"))...)
	}
	if t.failure == nil {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("failure"))...)
	}
	return nil
}

// NewToolEndExecutionFailure constructs the event closing a tool call
// whose execution itself failed (R-AMT-006, R-AMT-007). failure is
// required — the typed-failure wrap (R-AEV-008) is not optional on
// this outcome (S-AMT-051). On failure the zero [Event] is returned.
func NewToolEndExecutionFailure(run RunID, turn TurnID, callID string, ordinal uint32, failure *Failure) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := ToolEndExecutionFailure{
		callID:  callID,
		ordinal: ordinal,
		failure: failure,
	}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// ToolEndExecutionFailure returns the event's tool-end-execution-
// failure payload, and whether the event carries one. Called on an
// event of another kind, or on an event that was never constructed, it
// returns the zero [ToolEndExecutionFailure] and false, and never
// panics.
func (e Event) ToolEndExecutionFailure() (ToolEndExecutionFailure, bool) {
	payload, ok := e.payload.(ToolEndExecutionFailure)
	return payload, ok
}

// CallID returns the tool call's identity.
func (t ToolEndExecutionFailure) CallID() string { return t.callID }

// Ordinal returns the tool call's issuance ordinal within the run
// (R-AMT-007).
func (t ToolEndExecutionFailure) Ordinal() uint32 { return t.ordinal }

// Failure returns the typed [Failure] that caused the call's execution
// to fail (R-AEV-008). Only a [ToolOutcomeExecutionFailure] tool-end
// carries one.
func (t ToolEndExecutionFailure) Failure() (*Failure, bool) { return t.failure, t.failure != nil }

// Outcome returns [ToolOutcomeExecutionFailure].
func (t ToolEndExecutionFailure) Outcome() ToolOutcome { return ToolOutcomeExecutionFailure }

// String renders the payload for a diagnostic reader, naming the type,
// the call identity, the ordinal, and the outcome — never a failure's
// own content.
func (t ToolEndExecutionFailure) String() string {
	return "tool_end_execution_failure(" + t.callID + " ordinal=" + strconv.FormatUint(uint64(t.ordinal), 10) + ")"
}

// GoString renders the payload for the %#v verb.
func (t ToolEndExecutionFailure) GoString() string { return t.String() }
