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
// # maxFrameBytes: a single in-progress frame's hard cap
//
// NewDecoder's maxFrameBytes parameter bounds a single in-progress
// frame's accumulated size (R-ASD-018..021). Feed aborts with
// ErrFrameTooLarge the instant that accumulation grows strictly greater
// than the cap — never merely equal to it, so a frame that reaches
// exactly the cap and no further still decodes (S-ASD-063). Frames
// already dispatched earlier in the same Feed call are still returned
// alongside the error, never dropped because an error accompanies them
// (S-ASD-085). The first terminal error poisons the decoder: every later
// Feed or Finish call returns that same error, with zero frames.
type Decoder struct {
	// maxFrameBytes bounds a single in-progress frame's accumulated size
	// (R-ASD-018..021); see the type's doc comment for enforcement.
	maxFrameBytes int

	// buf is the retained tail: bytes fed but not yet resolved into a
	// complete line, carried across Feed calls.
	buf []byte

	// data and eventType accumulate the in-progress frame's fields.
	// Both reset to empty immediately after each dispatch.
	data      []byte
	eventType []byte

	// bomChecked is true once the decoder has resolved whether the
	// stream begins with a byte-order mark (R-ASD-007). It is set at
	// most once, for the whole stream's lifetime — never reset — so a
	// mark occurring anywhere else, including immediately after a
	// stripped leading one, is ordinary content (S-ASD-026).
	bomChecked bool

	// err is the decoder's terminal error, once one occurs (R-ASD-019's
	// cap-exceeded abort is the only source in this slice; AI-27.6 adds
	// truncation as a second). Once set, every later Feed and Finish
	// call returns it immediately, touching no other state (design.md's
	// "Poisoning" decision) — a half-consumed frame boundary is
	// unrecoverable, so there is nothing safe to resume.
	err error
}

// NewDecoder returns a Decoder ready to accept bytes through Feed.
//
// maxFrameBytes bounds a single in-progress frame's accumulated size
// (R-ASD-018..021). A non-positive value selects DefaultMaxFrameBytes —
// pinned by bounded_memory_test.go's own dedicated assertion, since which
// exact value zero selects had never previously been observable. See
// Decoder's doc comment for the cap's enforcement and poisoning behavior.
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
// A line may be terminated by CRLF, a lone LF, or a lone CR (R-ASD-006);
// see nextLine for how a CRLF pair split across two Feed calls stays a
// single terminator instead of injecting a phantom blank line.
//
// A leading byte-order mark, if present, is stripped exactly once for
// the whole stream before any line is scanned (R-ASD-007); see stripBOM.
// A blank line dispatches only when the data accumulation is non-empty
// (R-ASD-008); see dispatch.
//
// If the current in-progress frame's accumulated size grows strictly
// greater than maxFrameBytes, Feed returns ErrFrameTooLarge alongside any
// frames already dispatched earlier in this same call (R-ASD-019,
// S-ASD-085) — never dropping delivered content because an error
// accompanies it. Once returned, that error poisons the decoder: every
// later Feed call returns it immediately with zero frames, touching no
// other state.
func (d *Decoder) Feed(p []byte) ([]Frame, error) {
	if d.err != nil {
		return nil, d.err
	}

	d.buf = append(d.buf, p...)
	d.stripBOM()

	var frames []Frame
	cursor := 0
	for {
		line, next, ok := nextLine(d.buf, cursor)
		if !ok {
			// d.buf[cursor:] is now confirmed a genuine partial line —
			// nextLine searched it in full and found no terminator — so
			// its length is safe to add to the cap accounting here. This
			// must NOT be checked any earlier, at an arbitrary cursor:
			// summing the entire unscanned suffix before nextLine has
			// looked at it would count bytes belonging to a later,
			// not-yet-reached frame in the same input chunk against the
			// current one (S-ASD-085's mixed case would wrongly abort
			// before its complete first frame ever dispatches). See
			// exceedsCap's doc comment for the full accounting.
			if d.exceedsCap(len(d.buf) - cursor) {
				d.err = ErrFrameTooLarge
				return frames, d.err
			}
			break
		}
		cursor = next

		if len(line) == 0 {
			if frame, dispatched := d.dispatch(); dispatched {
				frames = append(frames, frame)
			}
			continue
		}
		d.processFieldLine(line)
		if d.exceedsCap(0) {
			d.err = ErrFrameTooLarge
			return frames, d.err
		}
	}

	d.buf = append(d.buf[:0], d.buf[cursor:]...)
	return frames, nil
}

// nextLine returns the next line in buf starting at start, and the
// cursor offset immediately following its terminator. ok is false when no
// terminator has arrived yet, in which case buf[start:] must be retained
// as an unconsumed tail rather than treated as a (short) line.
//
// Three terminators are recognized per WHATWG HTML § 9.2 (R-ASD-006): a
// lone LF, a lone CR, or a CRLF pair treated as ONE terminator — never as
// two independently-ending lines, which would inject a spurious blank
// line at exactly that split offset (R-ASD-013). A CR found at the very
// end of the bytes currently available is never resolved here: whether
// it is a lone CR or the first half of a CRLF whose LF has not arrived
// yet requires a following byte nextLine does not have. That CR is left
// exactly where it is, in the retained tail buf already carries across
// Feed calls — held structurally, with no separate "pending CR" flag
// that could desynchronize — and is resolved by the next Feed call's
// leading byte, or (AI-27.6, slice 6) by Finish.
func nextLine(buf []byte, start int) (line []byte, next int, ok bool) {
	rest := buf[start:]
	i := bytes.IndexAny(rest, "\r\n")
	if i < 0 {
		return nil, start, false
	}

	if rest[i] == '\n' {
		end := start + i
		return buf[start:end], end + 1, true
	}

	termLen, ok := resolveCR(rest, i)
	if !ok {
		return nil, start, false
	}
	end := start + i
	return buf[start:end], end + termLen, true
}

// resolveCR decides how many bytes the CR found at rest[crPos] terminates
// with: 2 for a CRLF pair, when the CR's very next byte is LF; 1 for a
// lone CR. ok is false when rest has no byte after the CR yet, so the
// question cannot be answered from the bytes fed so far — the caller
// must leave the CR unresolved rather than guess.
//
// This is the general "a boundary is visible but not yet resolvable
// without one more byte" shape nextLine needs for CR/CRLF; slice 2b's BOM
// prefix-wait check (a fixed 3-byte prefix, not a 1-byte lookahead) grows
// the same wait-rather-than-guess idea for a different boundary.
func resolveCR(rest []byte, crPos int) (termLen int, ok bool) {
	if crPos+1 >= len(rest) {
		return 0, false
	}
	if rest[crPos+1] == '\n' {
		return 2, true
	}
	return 1, true
}

// bom is the three-byte UTF-8 byte-order mark. A leading occurrence is
// stripped exactly once for the whole stream (R-ASD-007); an occurrence
// anywhere else — immediately after a stripped leading one, or inside a
// field value — is ordinary content the decoder never touches.
var bom = []byte{0xEF, 0xBB, 0xBF}

// stripBOM resolves, at most once per Decoder, whether the retained
// buffer starts with a byte-order mark, stripping it if so. bomChecked
// guards it so it runs at most once in the decoder's lifetime, however
// many Feed calls that takes.
//
// This grows resolveCR's "boundary visible but not yet resolvable
// without one more byte" shape (see its doc comment) for a fixed 3-byte
// prefix instead of a 1-byte lookahead: while the bytes seen so far are
// a proper prefix of the mark, stripBOM consumes nothing and waits for
// the next Feed call rather than guessing — the mark can arrive split
// across feeds (S-ASD-028). Once decided — either a full 3-byte match
// (strip) or an early mismatch (nothing to strip) — bomChecked is set
// and this never runs again, so a second mark immediately following the
// first is never re-examined and survives as ordinary content
// (S-ASD-026).
func (d *Decoder) stripBOM() {
	if d.bomChecked {
		return
	}
	if len(d.buf) < len(bom) {
		if bytes.HasPrefix(bom, d.buf) {
			return // proper prefix so far — wait for more bytes, never guess
		}
		d.bomChecked = true
		return
	}
	d.buf = bytes.TrimPrefix(d.buf, bom)
	d.bomChecked = true
}

// Finish signals that no further bytes will arrive.
//
// If an earlier Feed call already returned ErrFrameTooLarge, Finish
// returns that same error (design.md's poisoning decision) rather than
// nil: the decoder has nothing further to report once its first terminal
// error has occurred.
//
// Beyond that poisoning check, this slice's implementation is still a
// stub: truncation detection (an incomplete frame still pending when the
// caller signals completion) is AI-27.6's own ErrTruncated obligation.
// Every non-poisoned scenario this slice proves ends its transcript at a
// clean frame boundary, so Finish has nothing else yet to detect — but
// the method exists now because it is part of the public shape every
// later slice builds on (R-ASD-003's "decode completes" scenarios call
// it).
func (d *Decoder) Finish() error {
	if d.err != nil {
		return d.err
	}
	return nil
}

// processFieldLine parses one non-blank line as a field and updates the
// accumulator its name selects.
//
// Any name other than "event" or "data" — including a comment line
// (whose name is the empty string produced by a line starting with ':'),
// an "id" line and a "retry" line — falls through unrecognized and
// disturbs no accumulation. The decoder deliberately tracks no
// last-event-id or retry-interval state at all (R-ASD-010): id/retry
// lines are simply never matched by this switch, so their values are
// parsed off the line but discarded, never contributing to any
// accumulator — pinned by field_grammar_test.go's id/retry equivalence
// cases (S-ASD-035..038) rather than left as an accident of the
// fallthrough. Comment-line and unknown-event-name handling remain
// slice 4's own scope (AI-27.4).
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
// dispatched is false when the data accumulation is empty (R-ASD-008):
// no frame is yielded, but the accumulation and event-type state are
// still reset, so a suppressed frame leaves no residue for the next one
// (R-ASD-009, S-ASD-034). The reset is unconditional — a single deferred
// call covers both the empty-data path and the normal-dispatch path, so
// there is exactly one place a dispatch can forget to reset, not two.
//
// The trailing line feed each data line's append leaves behind is
// stripped once, here, per § 9.2's dispatch step — the same mechanism
// R-ASD-005 (slice 2a) names for multi-line joins, needed already so a
// single data line's payload is byte-exact with no trailing separator
// (S-ASD-001). Data is copied so a later Feed's buffer reuse can never
// observably change a Frame already returned.
func (d *Decoder) dispatch() (frame Frame, dispatched bool) {
	defer d.reset()

	if len(d.data) == 0 {
		return Frame{}, false
	}

	data := d.data
	if n := len(data); n > 0 && data[n-1] == '\n' {
		data = data[:n-1]
	}

	eventType := defaultEventType
	if len(d.eventType) > 0 {
		eventType = string(d.eventType)
	}

	return Frame{
		Event: eventType,
		Data:  append([]byte(nil), data...),
	}, true
}

// reset clears the in-progress frame's accumulators after a dispatch,
// keeping their backing arrays for reuse (R-ASD-009).
func (d *Decoder) reset() {
	d.data = d.data[:0]
	d.eventType = d.eventType[:0]
}

// exceedsCap reports whether the current in-progress frame's accumulated
// size is strictly greater than maxFrameBytes (R-ASD-019): retainedTail —
// the not-yet-resolved bytes beyond the scan cursor, when there are any —
// plus the already-accumulated data and event-type buffers. The relation
// is deliberately >, never >=: a frame whose accumulation reaches exactly
// the cap and no further still decodes (S-ASD-063).
//
// This is the one counter both of Feed's cap-check call sites share
// (task 5.8's consolidation): the retained-tail branch, reached only
// once nextLine has confirmed d.buf[cursor:] is a genuine partial line
// rather than more complete, not-yet-scanned lines (retainedTail > 0);
// and the post-field-line branch, reached after a resolved line has just
// grown data or eventType (retainedTail == 0, since nothing beyond the
// line just resolved has been examined yet).
//
// Passing the whole unscanned suffix at an arbitrary cursor — rather
// than only once nextLine has confirmed it is a genuine partial line —
// would count bytes belonging to a later, not-yet-reached frame in the
// same input chunk against the frame currently in progress: exactly the
// bug that would make S-ASD-085's mixed case (a complete frame followed
// by an over-cap frame, in one Feed call) wrongly abort before the
// complete frame ever has a chance to dispatch.
func (d *Decoder) exceedsCap(retainedTail int) bool {
	return retainedTail+len(d.data)+len(d.eventType) > d.maxFrameBytes
}
