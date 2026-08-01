// AI-07 — reasoning content and its opaque round-trip token.
//
// This file implements V-REQ-09 reasoning content, V-REQ-10 reasoning state and
// V-REQ-11 round-trip token. It is one file for one kind, the layout AI-06
// established: the payload type, its rules, its constructors and its accessor
// live together, so adding a kind is one new file plus three appends in
// content_part.go.
//
// # The token is correctness, not metadata
//
// doc 0001 § 6 makes it seam 11 and § 3.2 states the cost of its absence in the
// strongest terms that document uses about a field: at least one provider signs
// reasoning blocks cryptographically, and a signature not returned byte-exact
// fails multi-turn extended thinking with tool use. That failure appears on the
// *second* turn of a tool-using conversation, which is why the storage had to
// exist before the adapter that produces it rather than after the first bug
// report. So nothing here parses, verifies, decrypts, normalizes, trims,
// re-encodes or measures a token for meaning. The only rule it meets is a
// documented sanity bound, and [MaxReasoningTokenLen] records why that is not a
// provider limit.
//
// # Nothing new, on purpose
//
// doc 0001 § 3.1 records that the retired Layer 1 shipped two strategies for one
// contract, and that reasoning was the kind that broke the pattern: it
// implemented the content interface directly while three payload-carrying kinds
// kept a wrapper, and the two "had to be reconciled". This file therefore
// invents nothing. AI-06's decision.md § 9 is a table written so that this
// milestone can cite a row instead of re-deriving it, and every row is cited:
// one part type, the kind derived from the payload, one accessor shape, one
// rule set with two entry points, and a registration the AI-06.4 guard checks.
//
// # Two doors, and the state nobody sets
//
// [Reasoning] stores no state field. [Reasoning.State] is computed, for
// decision.md § 5's reason one level down: a stored state is a second source of
// truth, and a payload whose state says "redacted" while carrying plaintext
// would need a rule class to report a contradiction between two fields — which
// AI-04's five do not name. Deriving the state removes the rule instead of
// appending a sentinel for it. The one bit that is not derivable from the bytes
// is redaction, so it has a door of its own.

package ai

import (
	"strconv"
	"strings"
)

// MaxReasoningTokenLen is the documented byte bound on a round-trip token. A
// token longer than this is a caller-contract failure reported as
// [ErrOutOfRange].
//
// It is the **only** rule this package applies to a token, and it is a sanity
// bound rather than a provider limit — the identical position [MaxTextLen]
// takes, for the identical reason: Layer 1 holds no model catalog (V-OUT-14),
// so it cannot know what any provider will accept, and a bound that pretended
// to would be wrong for every provider the day after it was written. What the
// bound is for is making an unbounded value decidable from the request alone.
//
// It counts bytes, because bytes are what the token is: it is not text, it has
// no encoding this package knows, and counting anything else would mean
// decoding it — which is precisely what V-REQ-11 forbids.
//
// It is exported so a caller can check before constructing rather than by
// catching a failure.
const MaxReasoningTokenLen = 1 << 20

// ReasoningState is the closed vocabulary distinguishing the shapes reasoning
// content can take (V-REQ-10).
//
// It exists so that "this provider emitted no reasoning text" is a recorded
// state rather than an empty string a consumer has to interpret — doc 0001
// § 3.3 row 2, where one provider returns signed blocks, another an opaque
// token count, and a third thought signatures.
//
// It is derived rather than stored, so no reasoning payload can report a state
// its content contradicts. The zero value is not a member: it is the state of a
// value nobody built, and no constructed payload can hold it.
//
// This is the closed-vocabulary pattern role.go documents as "the pattern every
// later closed vocabulary in this package reuses", followed to the letter: the
// members start at iota + 1, and [ReasoningStates] walks the constant space
// rather than the name table.
type ReasoningState uint8

// The reasoning state vocabulary. Append-only: a new member takes the next
// value and its entry in reasoningStateNames, in the same commit.
const (
	// ReasoningStateText is reasoning the model emitted as text, with or
	// without a round-trip token beside it.
	ReasoningStateText ReasoningState = iota + 1

	// ReasoningStateRedacted is reasoning whose plaintext the provider withheld,
	// shipping an opaque payload in its place. At least one provider ships
	// encrypted redacted blocks that must replay verbatim, which is why this is
	// a state of its own rather than "reasoning that happens to have no text".
	ReasoningStateRedacted

	// ReasoningStateTokenOnly is reasoning with no text at all: what came back
	// is the token. It is both V-REQ-10's signature-only shape and its "a
	// provider that emitted no reasoning text at all" — doc 0002's AI-07.4
	// item 2 names them as one shape, because they are one shape under two
	// vendors' words for it.
	ReasoningStateTokenOnly

	// reasoningStateFirst and reasoningStateEnd bound the declared constant
	// space. They are not members and are never exported: reasoningStateEnd is
	// one past the last state, and moving it is what makes a newly declared
	// member visible to ReasoningStates.
	reasoningStateFirst = ReasoningStateText
	reasoningStateEnd   = ReasoningStateTokenOnly + 1
)

// reasoningStateNames is the table, and the single source of a state's
// rendering and its membership.
//
// A slice indexed by the constant rather than a map, for AI-04's reason:
// nothing in this package may let an unordered iteration decide anything.
var reasoningStateNames = []string{
	ReasoningStateText:      "text",
	ReasoningStateRedacted:  "redacted",
	ReasoningStateTokenOnly: "token-only",
}

// ReasoningStates returns the reasoning state vocabulary in declaration order.
//
// The result is a fresh slice on every call: a package-level variable would be
// a closed vocabulary any consumer could rewrite. It enumerates the declared
// constants rather than the table, so a member declared without an entry stays
// visible to the assertion that exists to catch it.
func ReasoningStates() []ReasoningState {
	out := make([]ReasoningState, 0, int(reasoningStateEnd-reasoningStateFirst))
	for s := reasoningStateFirst; s < reasoningStateEnd; s++ {
		out = append(out, s)
	}
	return out
}

// String renders the state.
//
// A member renders as its stable, lowercase, registered name. The zero value
// renders as "unset" and anything else as "reasoningstate(N)" — deliberately
// shapes no parser accepts, because this package has no parser for a state and
// a diagnostic rendering must never round-trip into a value.
func (s ReasoningState) String() string {
	if name := reasoningStateName(s); name != "" {
		return name
	}
	if s == 0 {
		return "unset"
	}
	return "reasoningstate(" + strconv.FormatUint(uint64(s), 10) + ")"
}

// reasoningStateName returns a state's registered name, or "" when the state
// has no table entry — which is exactly what makes it not a member.
func reasoningStateName(s ReasoningState) string {
	if int(s) >= len(reasoningStateNames) {
		return ""
	}
	return reasoningStateNames[s]
}

// Reasoning carries a model's intermediate reasoning (V-REQ-09).
//
// It is an opaque value type with unexported fields, built only by a
// constructor of this package — decision.md § 6.3's rule for a payload that
// carries structure rather than a bare string.
// # Why the token is held as a string
//
// It is bytes, and a []byte field would be the obvious shape. It is the wrong
// one for two reasons that only appear later, and both were found by a test
// rather than by inspection:
//
//   - Aliasing. A stored slice shares its backing array with the caller's
//     buffer, so the token changes with nobody writing to the part; and because
//     Reasoning is handed *out* by value, the same hazard exists on the read
//     path, where two consumers of one message can overwrite each other. A
//     defensive clone in the constructor fixes exactly half of that. A string
//     is immutable, so the type closes both halves instead of two remembered
//     clones. The conversions at the boundary each copy.
//   - Comparability. Part documents that it is "a value, and safe to copy…
//     Equality with == is defined and compares payloads". A payload containing
//     a slice makes == panic at runtime for that kind, which would have removed
//     a landed property of Part because a second kind arrived.
//
// A Go string is a byte container and not a text type — text_content.go already
// relies on that, storing bytes that are not well-formed UTF-8 — so nothing
// here treats the token as text.
type Reasoning struct {
	text     string
	token    string
	hasToken bool
	redacted bool
}

// kind reports this payload's registered kind.
func (Reasoning) kind() PartKind { return PartKindReasoning }

// validate reports the payload's first broken rule at the given position, or
// nil. The rules are checked in the order written, per V-FAIL-04:
//
//  1. reasoning in the [ReasoningStateText] state holds at least one
//     non-whitespace character, else [ErrEmpty] — a part carrying only
//     separators carries nothing a reader reads;
//  2. the reasoning text is at most [MaxTextLen] bytes, else [ErrOutOfRange] —
//     reasoning text is model-visible text and takes the package's existing
//     bound rather than a second name for one number;
//  3. reasoning in any other state carries a token, else [ErrEmpty] — a part
//     with neither text nor token carries nothing at all. A token that is
//     present and zero bytes long satisfies this rule: an empty token the
//     provider sent is a fact to record, not one to correct;
//  4. the token, when present, is at most [MaxReasoningTokenLen] bytes, else
//     [ErrOutOfRange]. This is the only rule the token itself meets.
//
// Text before token because text is the model-visible half, and within each,
// emptiness before bound: "you gave me nothing" and "you gave me too much" are
// different facts with different fixes, and a caller fixing one at a time must
// make progress in a predictable direction. That is text_content.go's ordering,
// restated here so a reader does not have to fetch it.
func (r Reasoning) validate(at Path) *Violation {
	return firstViolation(
		func() *Violation {
			if r.State() == ReasoningStateText && strings.TrimSpace(r.text) == "" {
				return Invalid(ErrEmpty, under(at, At("text"))...)
			}
			return nil
		},
		func() *Violation {
			if len(r.text) > MaxTextLen {
				return Invalid(ErrOutOfRange, under(at, At("text"))...)
			}
			return nil
		},
		func() *Violation {
			if r.State() != ReasoningStateText && !r.hasToken {
				return Invalid(ErrEmpty, under(at, At("token"))...)
			}
			return nil
		},
		func() *Violation {
			if r.hasToken && len(r.token) > MaxReasoningTokenLen {
				return Invalid(ErrOutOfRange, under(at, At("token"))...)
			}
			return nil
		},
	)
}

// State reports which shape this reasoning takes (V-REQ-10).
//
// It is derived from the payload on every call and is never stored, so the
// state and the content it describes cannot disagree — decision.md § 5's
// argument for [Part.Kind], one level down. Redaction is the one bit that is
// not derivable from the bytes, because a withheld plaintext and a signature
// are both opaque bytes with no text beside them, so it is set by
// [NewRedactedReasoning] and by nothing else.
func (r Reasoning) State() ReasoningState {
	switch {
	case r.redacted:
		return ReasoningStateRedacted
	case r.text != "":
		return ReasoningStateText
	default:
		return ReasoningStateTokenOnly
	}
}

// Token returns the opaque round-trip token and whether one is present at all
// (V-REQ-11).
//
// The second result is the distinction the whole milestone turns on: **absent
// is not empty**. A provider that returned an empty token and a provider that
// returned no token are two facts, and an adapter rebuilding the request must
// reproduce the one that happened. Presence is stored beside the bytes rather
// than inferred from their length, so nothing that touches the bytes — a copy,
// a clone, a conversion — can collapse the two.
//
// The bytes come back exactly as they were supplied: nothing on the
// construction path or the read path parses, verifies, decrypts, normalizes,
// trims or re-encodes them. Called on reasoning that carries no token, it
// returns nil and false.
func (r Reasoning) Token() ([]byte, bool) { return []byte(r.token), r.hasToken }

// Text returns the model's reasoning text, byte for byte as it was supplied.
//
// It is empty for every state other than [ReasoningStateText], and the state is
// what records which of those it is: a second boolean here would be a second
// way to ask one question.
func (r Reasoning) Text() string { return r.text }

// NewRedactedReasoning constructs reasoning whose plaintext the provider
// withheld, carrying the opaque payload it shipped instead (V-REQ-10).
//
// It is a second door because redaction cannot be derived from the bytes: a
// redacted block and a signature are both opaque bytes with no text beside
// them, and doc 0002's AI-07.4 item 1 requires the two to be distinguishable.
// At least one provider ships encrypted redacted blocks that must replay
// verbatim, so the payload is stored and returned untouched, exactly like any
// other round-trip token.
//
// A nil token means no token at all, which for this state is a caller-contract
// failure: a redacted part with nothing to replay carries nothing.
func NewRedactedReasoning(token []byte) (Part, error) {
	return newReasoning(Reasoning{redacted: true}, token)
}

// NewReasoning constructs reasoning content (V-REQ-09).
//
// The state is not a parameter: it is derived from what this call is given.
// Text yields [ReasoningStateText]; no text yields [ReasoningStateTokenOnly],
// which is how "this provider emitted no reasoning text" becomes a recorded
// state rather than an empty string a consumer has to interpret. The one state
// that cannot be derived has its own door, [NewRedactedReasoning].
//
// A nil token means the provider attached none. Any non-nil slice, including
// one of length zero, is a token that is present — the distinction V-REQ-11
// needs kept, because a provider that returned an empty token and a provider
// that returned no token are two facts and the return trip must reproduce the
// one that happened.
//
// On failure the zero [Part] is returned, so a caller that ignored the error
// cannot mistake the result for a constructed part.
func NewReasoning(text string, token []byte) (Part, error) {
	return newReasoning(Reasoning{text: text}, token)
}

// newReasoning attaches a token to a partly-built payload, runs the payload's
// own rules, and returns the part.
//
// It is the single place both doors pass through, so the rules are one rule set
// with two entry points and not two implementations that have to be kept in
// agreement — decision.md § 7.2.
func newReasoning(payload Reasoning, token []byte) (Part, error) {
	payload.token, payload.hasToken = string(token), token != nil
	if violation := payload.validate(nil); violation != nil {
		return Part{}, violation
	}
	return Part{payload: payload}, nil
}

// Reasoning returns the part's reasoning payload and whether the part carries
// one at all.
//
// This is the accessor shape decision.md § 6.1 named for this milestone. Called
// on a part of another kind, or on a part that was never constructed, it
// returns the zero payload and false, and never panics.
func (p Part) Reasoning() (Reasoning, bool) {
	payload, ok := p.payload.(Reasoning)
	return payload, ok
}
