// AI-06 — the content part: one opaque value type, readable and sealed.
//
// This file implements V-REQ-04 content part and V-REQ-05 content-part kind. Both
// of V-REQ-04's properties are constitutive rather than incidental — a payload
// readable from outside this package, and no valid value that skipped
// construction — and the retired Layer 1 shipped a design that had one without
// the other, twice, in opposite directions (doc 0001 § 3.1, defects C1 and C2).
//
// # Why a struct and not an interface
//
// For an interface, "readable" and "implementable" are the same word: both are
// methods. Every method exported so a consumer can read a payload is a method a
// consumer can supply. Sealing with an unexported method removes implementation
// and not satisfaction — an embedded interface promotes the method and yields a
// value that panics on the first call, which is precisely what AI-05's seam
// documented and this file closes.
//
// For a struct they are different words. Reading is methods; constructing is
// fields. Go forbids naming an unexported field in a composite literal from
// another package, and forbids a positional literal for a type that has one. So
// [Part] exports behavior without exporting construction, and the tension doc
// 0002 names dissolves: it was a property of interfaces, not of the problem.
//
// # The kind is derived, never stored
//
// [Part] holds no kind field. [Part.Kind] asks the payload, so the discriminator
// and the payload cannot disagree — there is only one of them. A part that
// carries no payload therefore has no kind, which is what makes an unconstructed
// value fail the closed-vocabulary rule rather than needing a rule class of its
// own.
//
// # What a later kind must add
//
// AI-07 reasoning and AI-09 tool call / tool result inherit this file instead of
// re-deriving it. Adding a kind is five steps, and AI-06.4's guard fails when any
// of them is missing:
//
//  1. declare the constant at the end of the PartKind block and move partKindEnd;
//  2. add an unexported payload type with kind() and validate();
//  3. add the exported constructor New<Kind>, whose rules are the payload's;
//  4. add the exported accessor func (p Part) <Kind>() (T, bool);
//  5. add the name to partKindNames and to the documented list on [PartKind].

package ai

import (
	"slices"
	"strconv"
)

// PartKind is the closed discriminator naming which payload a content part
// carries (V-REQ-05).
//
// It is derived from the payload rather than stored beside it, so a kind and a
// payload can never disagree. The zero value is not a member: it is the kind of
// a part that was never constructed, and it is rejected wherever a part is
// accepted.
//
// # Registered kinds
//
//   - text — model-visible natural-language text (V-REQ-08).
//   - tool_call — a model's intent to invoke a tool (V-REQ-16).
//   - tool_result — the answer to a tool call (V-REQ-18).
//
// The list above is not prose. AI-06.4 scans it and fails when it disagrees with
// the registration table, so a kind cannot be added without documenting it and
// cannot be documented without adding it.
//
// Image and audio are named in the content vocabulary and deliberately have no
// producer — V-PRV-07 and doc 0001 § 8.
type PartKind uint8

// The content-part kind vocabulary. Append-only: a new member takes the next
// value, its entry in partKindNames, and its line in the documented list above,
// in the same commit.
const (
	// PartKindText is the kind carrying model-visible natural-language text
	// (V-REQ-08). It is the first subject the part contract is proven against.
	PartKindText PartKind = iota + 1

	// PartKindToolCall is the kind carrying a model's request to invoke a tool
	// (V-REQ-16): its identity, the tool's name, and exact argument bytes. It is
	// an intent to invoke; this package never acts on it.
	PartKindToolCall

	// PartKindToolResult is the kind carrying the answer to a tool call
	// (V-REQ-18): its correlation to the originating call, its content, and
	// whether the tool reported failure. A result that reports failure is
	// ordinary content, not a failure of this package's taxonomy.
	PartKindToolResult

	// partKindFirst and partKindEnd bound the declared constant space. They are
	// not members and are never exported: partKindEnd is one past the last kind,
	// and moving it is what makes a newly declared member visible to PartKinds.
	partKindFirst = PartKindText
	partKindEnd   = PartKindToolResult + 1
)

// partKindNames is the registration table, and the single source of a kind's
// rendering and its validity.
//
// It is a slice indexed by the constant itself rather than a map, for AI-04's
// reason: nothing in this package may let an unordered iteration decide
// anything, and a registry is where that temptation is strongest.
var partKindNames = []string{
	PartKindText:       "text",
	PartKindToolCall:   "tool_call",
	PartKindToolResult: "tool_result",
}

// PartKinds returns the content-part kind vocabulary in declaration order.
//
// The result is a fresh slice on every call: a package-level variable would be a
// closed vocabulary any consumer could rewrite.
//
// It enumerates the declared constants rather than the table — AI-05's role.go
// states the reason and it applies verbatim: an enumeration derived from the
// table lists exactly the members that have an entry, so a member declared
// without one would be invisible to the assertion that exists to catch it.
func PartKinds() []PartKind {
	out := make([]PartKind, 0, int(partKindEnd-partKindFirst))
	for k := partKindFirst; k < partKindEnd; k++ {
		out = append(out, k)
	}
	return out
}

// String renders the kind.
//
// A member renders as its stable, lowercase, registered name. The zero value
// renders as "unset", and anything else as "partkind(N)" — deliberately shapes
// no parser accepts, because this package has no parser for a kind and a
// diagnostic rendering must never round-trip into a value.
func (k PartKind) String() string {
	if name := partKindName(k); name != "" {
		return name
	}
	if k == 0 {
		return "unset"
	}
	return "partkind(" + strconv.FormatUint(uint64(k), 10) + ")"
}

// partKindName returns a kind's registered name, or "" when the kind has no
// table entry — which is exactly what makes it not a member of the vocabulary.
//
// The bounds check is why a kind declared past the end of the table is rejected
// rather than panicking, and therefore why a scratch kind added without a table
// entry fails the guard instead of crashing it.
func partKindName(k PartKind) string {
	if int(k) >= len(partKindNames) {
		return ""
	}
	return partKindNames[k]
}

// partPayload is what a content part carries, and it is the reason [Part] is a
// sum type without being an interface.
//
// It is unexported and its methods are unexported, so no type outside this
// package can be one. Every payload answers two questions: which kind it is, and
// whether it satisfies its own construction rules.
type partPayload interface {
	// kind reports the registered kind this payload is. It is a method rather
	// than a field so that declaring a payload without a kind is a compile
	// error.
	kind() PartKind

	// validate reports the payload's first broken rule at the given position,
	// or nil. It is called by the kind's constructor and by the message
	// boundary — two entry points, one rule set.
	validate(at Path) *Violation
}

// Part is one element of a message's ordered content, carrying exactly one kind
// of payload (V-REQ-04).
//
// # Readable
//
// A consumer in any package discovers a part's kind with [Part.Kind] and obtains
// its payload with the accessor for that kind — [Part.Text] today. The payload
// comes back byte-equal to the value it was constructed from: nothing on the
// construction path or the read path normalizes, trims, re-encodes or escapes
// it. That is V-REQ-06, and without it request translation is structurally
// impossible rather than merely inconvenient.
//
// # Sealed
//
// A part is obtainable only from a constructor of this package. The struct has
// one field and it is unexported, so from another package no composite literal
// can set it, in either the named or the positional form, and reflection cannot
// either. Nothing can implement the contract instead, because there is no
// interface to implement: a type that embeds Part is a different type and cannot
// be offered as message content at all.
//
// What remains reachable from outside is the zero value, and the zero value has
// no payload — therefore no kind, therefore no membership of the closed kind
// vocabulary. Message construction rejects it with [ErrNotInVocabulary]. That is
// V-REQ-07, and it is what makes doc 0001's defect C1 unwritable.
//
// # A value, and safe to copy
//
// Part is a value. Copying one yields an equal part carrying the same payload,
// and payloads hold no mutable state, so two holders of a copy cannot observe
// each other. Equality with == is defined and compares payloads.
type Part struct{ payload partPayload }

// Kind reports which payload this part carries.
//
// It is derived from the payload on every call and is never stored, so the
// answer cannot drift from the payload it describes. A part that was never
// constructed reports the zero kind, which is not a member of the vocabulary.
func (p Part) Kind() PartKind {
	if p.payload == nil {
		return 0
	}
	return p.payload.kind()
}

// validate reports the part's first broken rule at the given position, or nil.
//
// The rules are checked in the order written, per V-FAIL-04:
//
//  1. the part carries a payload, else [ErrNotInVocabulary] — a value that
//     skipped construction has no payload, therefore no kind, therefore no
//     membership of the closed kind vocabulary. That is why this failure needs
//     no rule class of its own: doc 0001's defect C1 is a vocabulary failure,
//     and AI-05 already answered the identical question for the role nobody set;
//  2. the payload's kind is registered, else [ErrNotInVocabulary] — a payload
//     type added without a table entry makes every part carrying it invalid,
//     which is what keeps the registration table load-bearing rather than
//     decorative;
//  3. the payload satisfies its own rules, reported beneath this position.
func (p Part) validate(at Path) *Violation {
	if p.payload == nil {
		return Invalid(ErrNotInVocabulary, at...)
	}
	if partKindName(p.payload.kind()) == "" {
		return Invalid(ErrNotInVocabulary, at...)
	}
	return p.payload.validate(at)
}

// validateContent reports the first broken rule in a content sequence, or nil.
//
// # The prefix, and the caller it is for
//
// The prefix is prepended to every position this function reports. [NewMessage]
// passes none, so a failure renders as "content[0]". AI-10 holds a request,
// whose content sits one level deeper, and will pass AtIndex("messages", i) so
// the same failure renders as "messages[2].content[0]".
//
// That parameter is the whole of AI-06.3 item 3. "The same value offered through
// the request path is rejected there too" is one rule applied at two depths, not
// two implementations that have to be kept in agreement — and AI-10 does not
// exist yet, so the alternative would have been a speculative request type this
// milestone has no charter to build. The reuse point is pinned by
// content_part_internal_test.go, which fails if the position stops composing.
//
// Elements are checked in index order, so the first failing element is the one
// reported: V-FAIL-04 applied to a sequence.
func validateContent(prefix Path, content []Part) *Violation {
	for i, part := range content {
		if violation := part.validate(under(prefix, AtIndex("content", i))); violation != nil {
			return violation
		}
	}
	return nil
}

// String renders the part for a diagnostic reader, naming its kind and never its
// payload.
//
// A content part is the second thing in this package that formats caller data —
// validation.go records that a validation failure is the first — and here the
// default is worse than silence: fmt prints the unexported fields of a struct it
// has no String method for, so "%v" on a part would print the prompt. A consumer
// that wants the payload calls the accessor, which is what an accessor is for.
//
// The rendering names the kind alone. Not the payload, and not its length: a
// length is caller-derived, and a prefix of a secret is still a secret.
func (p Part) String() string {
	if p.payload == nil {
		return "part(unset)"
	}
	return "part(" + p.Kind().String() + ")"
}

// GoString renders the part for the %#v verb.
//
// It exists so the redaction posture covers every fmt verb rather than the three
// a reader thinks of. Without it, %#v falls back to reflection and prints the
// payload — which makes the posture a property of which verb someone reached
// for, and V-FAIL-13 puts it on the type instead.
func (p Part) GoString() string { return p.String() }

// under composes a position: a prefix followed by the steps beneath it.
//
// It concatenates rather than appending, so a caller's prefix is never rewritten
// by a rule that extends it — the aliasing hazard AI-05's copy semantics are
// about, one line long and one dimension smaller.
func under(prefix Path, steps ...Step) []Step {
	return slices.Concat([]Step(prefix), steps)
}

// firstViolation evaluates rules in order and reports the first violation, or
// nil when every rule passes. A nil rule is skipped.
//
// It is [FirstFailure] with a narrower result, and it exists because a rule that
// runs inside another rule needs to report *Violation rather than error. AI-04
// documented why the exported composer returns error: a nil *Violation placed in
// an error interface is not a nil error, and reproducing that trap at every call
// site of every contract was the cost it removed. This sibling does not
// reintroduce it. Its result is only ever compared to nil as a *Violation, and
// the single conversion to error happens once, at the exported boundary that
// returns it.
//
// Keeping the order as data rather than as control flow is the reason it is
// worth three lines: a reviewer reads which rule wins instead of tracing it.
func firstViolation(rules ...Rule) *Violation {
	for _, rule := range rules {
		if rule == nil {
			continue
		}
		if violation := rule(); violation != nil {
			return violation
		}
	}
	return nil
}
