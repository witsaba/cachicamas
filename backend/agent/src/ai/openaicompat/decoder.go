package openaicompat

import "bytes"

// DefaultMaxFrameBytes is the cap NewDecoder selects when its caller
// passes a non-positive value. 8 MiB comfortably covers a large tool
// result while still bounding runaway growth.
const DefaultMaxFrameBytes = 8 * 1024 * 1024

// Decoder incrementally decodes a byte stream in the WHATWG HTML Living
// Standard § 9.2 server-sent-events wire format into Frame values.
//
// # Framing layer, not semantics
//
// Decoder recognizes only § 9.2's grammar: field lines, comments, line
// terminators and the dispatch algorithm that turns them into frames. It
// draws no conclusion from a frame's content — in particular, it does not
// recognize this dialect's terminal sentinel and yields it as an ordinary
// frame; classifying wire failures and reading a frame's meaning both
// belong to later milestones (R-ASD-025).
//
// # Pure incremental function over bytes
//
// Decoder is driven entirely by its caller pushing bytes through Feed and
// signaling completion through Finish. It never reads from a transport,
// starts a goroutine, or depends on wall-clock time (R-ASD-003) — the
// caller owns every I/O concern and chooses every chunk boundary; decoding
// the same bytes always yields the same frames.
//
// # maxFrameBytes: accepted here, enforced from AI-27.5
//
// NewDecoder's maxFrameBytes parameter is normalized and stored from this
// slice onward, but no code path in this slice's Feed checks it: cap
// enforcement (ErrFrameTooLarge) is AI-27.5's own RED/GREEN pair. A
// reviewer of this slice should not read the parameter's presence as
// proof the cap is already live — it is deliberately inert until then.
type Decoder struct {
	// maxFrameBytes bounds a single in-progress frame's accumulated size.
	// Not yet enforced in this slice; see the type's doc comment.
	maxFrameBytes int

	// buf is the retained tail: bytes fed but not yet resolved into a
	// complete line, carried across Feed calls.
	buf []byte

	// data and eventType accumulate the in-progress frame's fields.
	// Both reset to empty immediately after each dispatch.
	data      []byte
	eventType []byte
}

// NewDecoder returns a Decoder ready to accept bytes through Feed.
//
// maxFrameBytes bounds a single in-progress frame's accumulated size. A
// non-positive value selects DefaultMaxFrameBytes. See Decoder's doc
// comment for why this slice accepts and normalizes the value without yet
// enforcing it.
func NewDecoder(maxFrameBytes int) *Decoder {
	if maxFrameBytes <= 0 {
		maxFrameBytes = DefaultMaxFrameBytes
	}
	return &Decoder{maxFrameBytes: maxFrameBytes}
}

// Feed appends p to the decoder's retained bytes, scans every complete
// line it can now resolve, and returns the frames those lines dispatch,
// in arrival order (R-ASD-002). Bytes that do not yet form a complete
// line are retained for the next Feed call.
//
// This slice's scan recognizes a single line feed as the only line
// terminator; CR and CRLF handling is AI-27.2's growth (slice 2a).
func (d *Decoder) Feed(p []byte) ([]Frame, error) {
	d.buf = append(d.buf, p...)

	var frames []Frame
	cursor := 0
	for {
		line, next, ok := nextLine(d.buf, cursor)
		if !ok {
			break
		}
		cursor = next

		if len(line) == 0 {
			frames = append(frames, d.dispatch())
			continue
		}
		d.processFieldLine(line)
	}

	d.buf = append(d.buf[:0], d.buf[cursor:]...)
	return frames, nil
}

// nextLine returns the next line in buf starting at start, and the
// cursor offset immediately following its terminator. ok is false when no
// terminator has arrived yet, in which case buf[start:] must be retained
// as an unconsumed tail rather than treated as a (short) line.
//
// This is the whole cursor-bookkeeping contract Feed's scan loop depends
// on, extracted so slice 2a can grow it into full three-terminator
// handling (CR, LF, and CRLF-as-one, including a CRLF split across two
// Feed calls) without touching Feed's own loop shape. This slice's
// version recognizes a single line feed only.
func nextLine(buf []byte, start int) (line []byte, next int, ok bool) {
	i := bytes.IndexByte(buf[start:], '\n')
	if i < 0 {
		return nil, start, false
	}
	end := start + i
	return buf[start:end], end + 1, true
}

// Finish signals that no further bytes will arrive.
//
// This slice's implementation is intentionally a stub returning nil
// unconditionally: truncation detection (an incomplete frame still
// pending when the caller signals completion) is AI-27.6's ErrTruncated
// obligation. Every scenario this slice proves ends its transcript at a
// clean frame boundary, so Finish has nothing yet to detect — but the
// method exists now because it is part of the public shape every later
// slice builds on (R-ASD-003's "decode completes" scenarios call it).
func (d *Decoder) Finish() error {
	return nil
}

// processFieldLine parses one non-blank line as a field and updates the
// accumulator its name selects.
//
// Any name other than "event" or "data" — including a comment line
// (whose name is the empty string produced by a line starting with ':')
// and the id/retry fields slice 2b pins as ignored — falls through
// unrecognized and disturbs no accumulation. That is this slice's minimum
// needed to keep S-ASD-001…009 honest about a line it does not
// specifically recognize; slice 2a and 2b each add their own RED/GREEN
// pair for the field grammar's remaining edge cases.
func (d *Decoder) processFieldLine(line []byte) {
	name, value := splitField(line)
	switch string(name) {
	case "event":
		d.eventType = append(d.eventType[:0], value...)
	case "data":
		d.data = append(d.data, value...)
		d.data = append(d.data, '\n')
	}
}

// splitField splits line at its first colon per WHATWG HTML § 9.2: the
// field name is the text before the colon, and the field value is the
// text after it with exactly one leading space removed, if present. A
// colonless line yields the whole line as the name and a nil value.
//
// R-ASD-004's full edge-case coverage (colon-in-value, two-space,
// no-space, colonless-name) is slice 2a's dedicated table; this is the
// mechanism this slice's own fixtures already need to be byte-exact.
func splitField(line []byte) (name, value []byte) {
	i := bytes.IndexByte(line, ':')
	if i < 0 {
		return line, nil
	}
	name = line[:i]
	value = line[i+1:]
	if len(value) > 0 && value[0] == ' ' {
		value = value[1:]
	}
	return name, value
}

// dispatch builds a Frame from the current accumulation and resets it.
//
// The trailing line feed each data line's append leaves behind is
// stripped once, here, per § 9.2's dispatch step — the same mechanism
// R-ASD-005 (slice 2a) names for multi-line joins, needed already so a
// single data line's payload is byte-exact with no trailing separator
// (S-ASD-001). Data is copied so a later Feed's buffer reuse can never
// observably change a Frame already returned.
func (d *Decoder) dispatch() Frame {
	data := d.data
	if n := len(data); n > 0 && data[n-1] == '\n' {
		data = data[:n-1]
	}

	eventType := defaultEventType
	if len(d.eventType) > 0 {
		eventType = string(d.eventType)
	}

	frame := Frame{
		Event: eventType,
		Data:  append([]byte(nil), data...),
	}
	d.reset()
	return frame
}

// reset clears the in-progress frame's accumulators after a dispatch,
// keeping their backing arrays for reuse (R-ASD-009).
func (d *Decoder) reset() {
	d.data = d.data[:0]
	d.eventType = d.eventType[:0]
}
