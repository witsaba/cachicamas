package openaicompat

// defaultEventType is the event type a frame carries when no event-type
// line contributed to it — and also when one contributed an empty value,
// which the dispatch algorithm treats identically (R-ASD-009): the
// grammar tests the event-type accumulation's emptiness, not whether a
// line was ever seen.
//
// Grounded verbatim in WHATWG HTML § 9.2's dispatch algorithm, which
// creates "an event that uses the MessageEvent interface, with the event
// type message" before conditionally overwriting it.
const defaultEventType = "message"

// Frame is one dispatched server-sent event: an event-type name and its
// accumulated data payload, with the dispatch-time trailing line-feed
// separator already removed (R-ASD-001).
//
// Frame carries no last-event-id, retry interval, comment or sequence
// number. The decoder tracks none of them (R-ASD-010) — there is nothing
// to carry.
type Frame struct {
	// Event is the frame's event-type name. It equals defaultEventType
	// ("message") when no event-type line contributed to the frame, or
	// when one contributed an empty value (R-ASD-009).
	Event string

	// Data is the frame's accumulated data payload. It is always a copy:
	// mutating it never observably affects the Decoder that produced it,
	// and a later Feed call never observably affects a Frame a previous
	// call already returned.
	Data []byte
}
