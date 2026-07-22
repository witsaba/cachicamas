// Package-level note for the AI-15 tool-call start/delta/end payloads.
//
// Tool-call payloads carry an opaque callID so Layer 2 can independently
// reconstruct interleaved calls. Delta bytes are opaque partial JSON and are
// not subject to UTF-8 boundary or full-JSON validation. Ordering and
// accumulation remain Layer-2 concerns.
package ai

import "encoding/json"

// MaxToolCallDeltaLength is the maximum byte length of one opaque tool-call
// argument fragment. The final arguments remain subject to
// MaxToolCallArgumentsLength.
const MaxToolCallDeltaLength = 1 << 16

// ToolCallStartPayload opens a tool-call lifecycle for callID and name.
type ToolCallStartPayload struct {
	callID string
	name   string
}

func (ToolCallStartPayload) aiPayload() {}

// Kind returns EventKindToolCallStart.
func (ToolCallStartPayload) Kind() EventKind { return EventKindToolCallStart }

// CallID returns the correlation handle verbatim.
func (p ToolCallStartPayload) CallID() string { return p.callID }

// Name returns the tool name verbatim.
func (p ToolCallStartPayload) Name() string { return p.name }

// Validate re-runs the constructor's structural checks.
func (p ToolCallStartPayload) Validate() error {
	return validateToolCallStart(p.callID, p.name)
}

// NewToolCallStartPayload validates and constructs a start payload.
func NewToolCallStartPayload(callID, name string) (ToolCallStartPayload, error) {
	if err := validateToolCallStart(callID, name); err != nil {
		return ToolCallStartPayload{}, err
	}
	return ToolCallStartPayload{callID: callID, name: name}, nil
}

func validateToolCallStart(callID, name string) error {
	if err := validateToolCallEventCallID(callID); err != nil {
		return err
	}
	return validateToolCallEventName(name)
}

// NewToolCallStartEvent constructs a validated tool_call.start event.
func NewToolCallStartEvent(callID, name string) (Event, error) {
	p, err := NewToolCallStartPayload(callID, name)
	if err != nil {
		return Event{}, err
	}
	return NewEvent(p)
}

// AsToolCallStart extracts a start payload after checking kind parity.
func AsToolCallStart(ev Event) (ToolCallStartPayload, bool) {
	if ev.Kind != EventKindToolCallStart {
		return ToolCallStartPayload{}, false
	}
	p, ok := ev.Payload.(ToolCallStartPayload)
	return p, ok
}

// ToolCallDeltaPayload carries one opaque argument fragment for callID.
type ToolCallDeltaPayload struct {
	callID string
	delta  json.RawMessage
}

func (ToolCallDeltaPayload) aiPayload() {}

// Kind returns EventKindToolCallDelta.
func (ToolCallDeltaPayload) Kind() EventKind { return EventKindToolCallDelta }

// CallID returns the correlation handle verbatim.
func (p ToolCallDeltaPayload) CallID() string { return p.callID }

// Delta returns the opaque fragment bytes verbatim.
func (p ToolCallDeltaPayload) Delta() json.RawMessage { return p.delta }

// Validate re-runs callID and byte-length structural checks. It deliberately
// does not parse JSON or inspect UTF-8 boundaries.
func (p ToolCallDeltaPayload) Validate() error {
	if err := validateToolCallEventCallID(p.callID); err != nil {
		return err
	}
	return validateToolCallDelta(p.delta)
}

// NewToolCallDeltaPayload validates and constructs a delta payload.
func NewToolCallDeltaPayload(callID string, delta json.RawMessage) (ToolCallDeltaPayload, error) {
	if err := validateToolCallEventCallID(callID); err != nil {
		return ToolCallDeltaPayload{}, err
	}
	if err := validateToolCallDelta(delta); err != nil {
		return ToolCallDeltaPayload{}, err
	}
	return ToolCallDeltaPayload{callID: callID, delta: delta}, nil
}

func validateToolCallDelta(delta json.RawMessage) error {
	if len(delta) == 0 || len(delta) > MaxToolCallDeltaLength {
		return ErrMalformedToolCallArguments
	}
	return nil
}

// NewToolCallDeltaEvent constructs a validated tool_call.delta event.
func NewToolCallDeltaEvent(callID string, delta json.RawMessage) (Event, error) {
	p, err := NewToolCallDeltaPayload(callID, delta)
	if err != nil {
		return Event{}, err
	}
	return NewEvent(p)
}

// AsToolCallDelta extracts a delta payload after checking kind parity.
func AsToolCallDelta(ev Event) (ToolCallDeltaPayload, bool) {
	if ev.Kind != EventKindToolCallDelta {
		return ToolCallDeltaPayload{}, false
	}
	p, ok := ev.Payload.(ToolCallDeltaPayload)
	return p, ok
}

// ToolCallEndPayload terminates a call with validated final JSON arguments.
type ToolCallEndPayload struct {
	callID    string
	name      string
	arguments json.RawMessage
}

func (ToolCallEndPayload) aiPayload() {}

// Kind returns EventKindToolCallEnd.
func (ToolCallEndPayload) Kind() EventKind { return EventKindToolCallEnd }

// CallID returns the correlation handle verbatim.
func (p ToolCallEndPayload) CallID() string { return p.callID }

// Name returns the tool name verbatim.
func (p ToolCallEndPayload) Name() string { return p.name }

// Arguments returns the final JSON argument bytes verbatim.
func (p ToolCallEndPayload) Arguments() json.RawMessage { return p.arguments }

// Validate re-runs callID, name, and final-argument checks.
func (p ToolCallEndPayload) Validate() error {
	if err := validateToolCallStart(p.callID, p.name); err != nil {
		return err
	}
	return validateToolCallArguments(p.arguments)
}

// NewToolCallEndPayload validates and constructs an end payload.
func NewToolCallEndPayload(callID, name string, arguments json.RawMessage) (ToolCallEndPayload, error) {
	if err := validateToolCallStart(callID, name); err != nil {
		return ToolCallEndPayload{}, err
	}
	if err := validateToolCallArguments(arguments); err != nil {
		return ToolCallEndPayload{}, err
	}
	return ToolCallEndPayload{callID: callID, name: name, arguments: arguments}, nil
}

// NewToolCallEndEvent constructs a validated tool_call.end event.
func NewToolCallEndEvent(callID, name string, arguments json.RawMessage) (Event, error) {
	p, err := NewToolCallEndPayload(callID, name, arguments)
	if err != nil {
		return Event{}, err
	}
	return NewEvent(p)
}

// AsToolCallEnd extracts an end payload after checking kind parity.
func AsToolCallEnd(ev Event) (ToolCallEndPayload, bool) {
	if ev.Kind != EventKindToolCallEnd {
		return ToolCallEndPayload{}, false
	}
	p, ok := ev.Payload.(ToolCallEndPayload)
	return p, ok
}

func validateToolCallEventCallID(callID string) error {
	if callID == "" || isAllWhitespace(callID) {
		return ErrEmptyToolResultCallID
	}
	if len(callID) > MaxToolResultCallIDLength {
		return ErrToolResultCallIDTooLong
	}
	return nil
}

func validateToolCallEventName(name string) error {
	if name == "" || isAllWhitespace(name) {
		return ErrEmptyToolCallName
	}
	if len(name) > MaxToolNameLength {
		return ErrToolCallNameTooLong
	}
	if hasControlChar(name) {
		return ErrInvalidToolCallName
	}
	return nil
}
