// AI-15.2 — the completion event.
//
// This file implements V-STR-20 completion event: the terminal event of a
// stream that finished normally, carrying AI-13's finish reason and usage
// unchanged. It follows response_start.go's layout: an opaque value type
// with unexported fields, a package-owned constructor as the only door in,
// and a typed accessor on the event.
//
// # Embedding, never re-inventing
//
// FinishReason and Usage are AI-13's existing types, held by value and
// embedded as-is (R-ARP-007). This file declares no new finish-reason
// member, no new usage field, and no parallel type: construction delegates
// to FinishReason.Validate and Usage.Validate, in that order (V-FAIL-04),
// so a completion can only be invalid the way AI-13 already says a finish
// reason or a usage record is invalid.
//
// # Absence survives by construction
//
// Usage's absence-versus-zero property (AI-13.3) needs nothing extra here:
// TokenCount's presence bit is a value carried by copy, so a Usage held
// unchanged carries its absent counts unchanged (R-ARP-008).
//
// # Terminal by registration, not by special case
//
// Completion registers with AI-14's descriptor-driven terminal property
// (R-ARP-009): the checker enforces "at most one, and nothing after it" for
// every terminal-registered kind, with no branch naming this one.

package ai

// Completion is the terminal event of a stream that finished normally
// (V-STR-20): AI-13's finish reason and usage, unchanged.
type Completion struct {
	reason FinishReason
	usage  Usage
}

// kind reports this payload's registered kind.
func (Completion) kind() EventKind { return EventKindCompletion }

// validate reports the payload's first broken rule at the given position, or
// nil. The rules are checked in the order written, per V-FAIL-04
// (R-ARP-007): the finish reason is a member of AI-13's closed vocabulary,
// else its own [ErrNotInVocabulary] at "finish_reason"; then the usage
// record carries no negative count, else its own [ErrOutOfRange] naming the
// offending count under "usage". A fully absent usage record passes
// (R-ARP-008).
func (c Completion) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation { return violationOf(c.reason.Validate(under(at, At("finish_reason"))...)) },
		func() *Violation { return violationOf(c.usage.Validate(under(at, At("usage"))...)) },
	)
}

// NewCompletion constructs a completion event (V-STR-20).
//
// It is the only door in. The rules are [Completion.validate]'s — which are
// AI-13's own [FinishReason.Validate] and [Usage.Validate], run in that
// order — from here and again at [CheckEmit] (R-AEE-010). On failure the
// zero [Event] is returned. The returned event is unstamped (sequence 0);
// stamping is a [Stamper]'s job, per-stream, at the producer boundary.
func NewCompletion(reason FinishReason, usage Usage) (Event, error) {
	payload := Completion{reason: reason, usage: usage}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload}, nil
}

// Completion returns the event's completion payload, and whether the event
// carries one.
//
// Called on an event of another kind, or on an event that was never
// constructed, it returns the zero [Completion] and false, and never
// panics.
func (e Event) Completion() (Completion, bool) {
	payload, ok := e.payload.(Completion)
	if !ok {
		return Completion{}, false
	}
	return payload, true
}

// FinishReason returns the reason generation stopped (V-MET-01), AI-13's
// existing type, unchanged.
func (c Completion) FinishReason() FinishReason { return c.reason }

// Usage returns what the response consumed (V-MET-09), AI-13's existing
// type, unchanged. An absent count reported by the provider stays absent
// after crossing this event boundary (R-ARP-008).
func (c Completion) Usage() Usage { return c.usage }

// String renders the completion for a diagnostic reader, naming the type
// and nothing it carries — [ResponseStart.String]'s posture, restated
// (V-FAIL-13). Not the finish reason, not the usage counts, and not their
// length: a length is caller-derived and a prefix of a secret is still a
// secret.
func (c Completion) String() string { return "completion" }

// GoString renders the completion for the %#v verb.
//
// Without it, %#v falls back to reflection and prints the unexported
// fields, which would make the redaction posture a property of which verb
// someone reached for rather than a property of the type.
func (c Completion) GoString() string { return c.String() }
