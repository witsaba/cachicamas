// AI-15.1 — the response-start event.
//
// This file implements V-STR-19 response-start event: the event announcing
// that a provider has begun responding. It is this package's first
// registered production event kind, and it follows tool_call.go's layout
// exactly: an opaque value type with unexported fields, a package-owned
// constructor as the only door in, and a typed accessor on the event.
//
// # Two required fields, both provider-supplied and byte-exact
//
// The provider response identity (V-STR-24) and the served model (V-STR-25)
// are both plain strings following the ToolCall.ID() / Request.Model()
// precedent — never the minted MessageID pattern (R-ARP-002). Neither is
// absent-capable: a provider that ships no response id, or omits the served
// model, obliges its adapter to fill it before this constructor is called
// (R-ARP-004).
//
// # Served model is never compared to the requested model
//
// V-STR-25 served model is a register noun distinct from V-REQ-21 model
// identity. This file holds no reference to the request-side type and
// performs no comparison between the two (R-ARP-003).

package ai

// ResponseStart is the event announcing that a provider has begun
// responding (V-STR-19): the provider's own response identity and the
// model that actually served it.
type ResponseStart struct{ responseID, servedModel string }

// kind reports this payload's registered kind.
func (ResponseStart) kind() EventKind { return EventKindResponseStart }

// validate reports the payload's first broken rule at the given position, or
// nil. The rules are checked in the order written, per V-FAIL-04
// (R-ARP-004): the response identity is not empty, else [ErrEmpty] at
// "response_id"; then the served model is not empty, else [ErrEmpty] at
// "served_model".
func (s ResponseStart) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if s.responseID == "" {
				return Invalid(ErrEmpty, under(at, At("response_id"))...)
			}
			return nil
		},
		func() *Violation {
			if s.servedModel == "" {
				return Invalid(ErrEmpty, under(at, At("served_model"))...)
			}
			return nil
		},
	)
}

// NewResponseStart constructs a response-start event (V-STR-19).
//
// It is the only door in. The rules are [ResponseStart.validate]'s, run
// from here — this package's producer boundary — and again at [CheckEmit],
// the emission boundary every event must pass before it may be treated as
// part of a stream (R-AEE-010). On failure the zero [Event] is returned, so
// a caller that ignores the error cannot mistake the result for a
// constructed event. The returned event is unstamped (sequence 0);
// stamping is a [Stamper]'s job, per-stream, at the producer boundary.
func NewResponseStart(responseID, servedModel string) (Event, error) {
	payload := ResponseStart{responseID: responseID, servedModel: servedModel}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// ResponseStart returns the event's response-start payload, and whether the
// event carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [ResponseStart] and false, and never
// panics.
func (e Event) ResponseStart() (ResponseStart, bool) {
	payload, ok := e.payload.(ResponseStart)
	if !ok {
		return ResponseStart{}, false
	}
	return payload, true
}

// ResponseID returns the provider's own opaque handle for the response this
// stream carries (V-STR-24).
//
// It is byte-exact: nothing here trims, lower-cases, parses or re-formats
// it. It is provider-supplied, never minted by this package — unlike
// [MessageID], there is no synthesis path here, because R-ARP-004 makes an
// unfilled response identity the adapter's obligation, not this
// constructor's.
func (s ResponseStart) ResponseID() string { return s.responseID }

// ServedModel returns the model that actually produced the response
// (V-STR-25), as reported by the provider.
//
// It is distinct from the requested model (V-REQ-21): this package never
// reads, compares or validates against it (R-ARP-003).
func (s ResponseStart) ServedModel() string { return s.servedModel }

// String renders the response start for a diagnostic reader, naming the
// type and nothing it carries — [ToolCall.String]'s posture, restated for
// the first event payload this package registers (V-FAIL-13). Not the
// response identity, not the served model, and not their length: a length
// is caller-derived and a prefix of a secret is still a secret.
func (s ResponseStart) String() string { return "responsestart" }

// GoString renders the response start for the %#v verb.
//
// Without it, %#v falls back to reflection and prints the unexported
// fields, which would make the redaction posture a property of which verb
// someone reached for rather than a property of the type.
func (s ResponseStart) GoString() string { return s.String() }
