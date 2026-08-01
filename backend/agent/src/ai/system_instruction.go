// AI-10.2 — the system instruction, segmented from birth.

package ai

import (
	"slices"
	"strconv"
	"strings"
)

// Segment is one ordered, individually markable piece of the system
// instruction (V-REQ-19).
type Segment struct{ text string }

// NewSegment constructs one piece of the system instruction.
//
// One rule: the text is present, else [ErrEmpty] at "text". Whitespace-only is
// folded into emptiness, and the reason is V-REQ-19's own — a segment of spaces
// places no text into the instruction while occupying an ordinal that is
// individually markable, so accepting it would let a caller mark a cache
// boundary on nothing. Non-ASCII whitespace counts, so a non-breaking or
// ideographic space is no more a segment than an ASCII one.
//
// The text is not trimmed. Whitespace-only is a rejection criterion and never a
// normalization: a segment whose text is "  Be terse.  " round-trips with its
// padding intact, because an adapter concatenating segments needs the caller's
// own separators and prompt text must survive byte-exact.
//
// There is deliberately no upper bound on a segment's length. A text content
// part has one because a part is model-visible content with a documented
// encoding; a system instruction's size limit is a provider's, decidable only
// by asking, and belongs to AI-19.
func NewSegment(text string) (Segment, error) {
	if strings.TrimSpace(text) == "" {
		return Segment{}, Invalid(ErrEmpty, At("text"))
	}
	return Segment{text: text}, nil
}

// Text returns the segment's text, byte-equal to what it was constructed with.
//
// Byte-equal includes surrounding whitespace: an adapter concatenating segments
// needs the caller's own separators, and a constructor that trimmed would
// silently change the prompt.
func (s Segment) Text() string { return s.text }

// IsZero reports whether the segment was never constructed.
//
// A constructed segment always carries non-empty text, so emptiness is a sound
// detector and no separate flag is needed.
func (s Segment) IsZero() bool { return s.text == "" }

// SystemInstruction is the request's ordered system-instruction segments.
type SystemInstruction struct{ segments []Segment }

// NewSystemInstruction constructs a system instruction from ordered segments.
//
// The rules are checked in the order written, per V-FAIL-04:
//
//  1. There is at least one segment, else [ErrEmpty] at "system". No arguments,
//     an empty sequence and a nil sequence are one fact and report one failure.
//  2. Every segment was constructed, else [ErrEmpty] at "system[i]", checked in
//     index order. A segment that skipped [NewSegment] carries no text, which no
//     constructed segment does, so emptiness is a sound detector without a
//     separate flag.
//
// Rule 1 could have been dropped, letting a zero-segment instruction stand for
// absence. That is the trap it exists to close: it would give the package two
// spellings of absence — the option unapplied, and the option applied with an
// empty value — and every later reader would have to know both. One spelling,
// enforced at construction, which is what makes "one empty segment"
// unrepresentable rather than merely equivalent to absence.
//
// The sequence is copied, so a caller may reuse the slice it passed.
func NewSystemInstruction(segments ...Segment) (SystemInstruction, error) {
	if err := FirstFailure(
		func() *Violation {
			if len(segments) == 0 {
				return Invalid(ErrEmpty, At("system"))
			}
			return nil
		},
		func() *Violation {
			for i, seg := range segments {
				if seg.IsZero() {
					return Invalid(ErrEmpty, AtIndex("system", i))
				}
			}
			return nil
		},
	); err != nil {
		return SystemInstruction{}, err
	}
	return SystemInstruction{segments: slices.Clone(segments)}, nil
}

// IsZero reports whether the instruction was never constructed.
//
// It is how [NewRequest] tells an applied-but-unconstructed instruction from a
// real one, the same question [MessageID.IsZero] answers for a message.
func (i SystemInstruction) IsZero() bool { return len(i.segments) == 0 }

// Segments returns the instruction's ordered segments.
func (i SystemInstruction) Segments() []Segment { return slices.Clone(i.segments) }

// Len returns the number of segments the instruction holds.
func (i SystemInstruction) Len() int { return len(i.segments) }

// WithSystemInstruction attaches a system instruction to a request.
func WithSystemInstruction(system SystemInstruction) RequestOption {
	return func(d *requestDraft) { d.system, d.hasSystem = system, true }
}

// SystemInstruction returns the request's system instruction and whether it
// carries one.
func (r Request) SystemInstruction() (SystemInstruction, bool) {
	return r.system, r.hasSystem
}

// NewSystemText constructs a single-segment system instruction.
//
// It is the ergonomic path for the common case, and it produces a value
// indistinguishable from [NewSystemInstruction] called with one segment —
// same count, same text, equal segments. doc 0002 prices segments at zero on
// exactly that condition, and one provider's system field takes either a string
// or an ordered array of blocks, so the ergonomic path must produce the
// one-block array rather than a parallel shape beside it.
//
// It is written as a call to the two constructors rather than as a direct
// build. Building the value here would pass the same test today and would let
// the two paths diverge on the next rule added to a segment, which is AI-06's
// "one rule set, two callers" failure mode one dimension smaller.
func NewSystemText(text string) (SystemInstruction, error) {
	seg, err := NewSegment(text)
	if err != nil {
		return SystemInstruction{}, err
	}
	return NewSystemInstruction(seg)
}

// String renders the segment, naming that it is one and never its text.
//
// Not the text, and not its length: a length is caller-derived, and a prefix of
// a secret is still a secret. [Part] renders on the same rule, and the system
// instruction is the region most likely to carry a proprietary prompt.
//
// AI-11 may extend this to name a cache marker, because a marker is structural
// rather than payload.
func (s Segment) String() string {
	if s.IsZero() {
		return "segment(unset)"
	}
	return "segment"
}

// GoString renders the segment for the %#v verb.
func (s Segment) GoString() string { return s.String() }

// String renders the instruction, naming its segment count and never their
// text.
//
// A count is a fact about the request's shape rather than about a caller's
// payload, and it is the fact a reader debugging a cache boundary needs.
func (i SystemInstruction) String() string {
	return "system(" + strconv.Itoa(len(i.segments)) + " segments)"
}

// GoString renders the instruction for the %#v verb.
func (i SystemInstruction) GoString() string { return i.String() }
