// AI-08.3 — the tool-choice vocabulary, and the closed-vocabulary pattern
// extended for a member that carries a payload.
//
// This file implements V-REQ-15 tool choice: "the request-level instruction
// constraining whether and which tool the model may call, validated against the
// declared tool set." It is the second consumer of the pattern role.go landed,
// and the first that had to widen it.
//
// # What the pattern was, and what this file changes
//
// role.go's pattern is built for payload-free members: an integer defined type
// starting at iota + 1, a name table indexed by the constant, an enumeration
// that walks the *constant space* rather than the table, and an exhaustiveness
// pin over that enumeration. Three of tool choice's four members fit it
// unchanged. The fourth — call this specific tool — carries a name.
//
// The extension, in one sentence: the vocabulary stays a payload-free integer;
// the payload lives in a separate value type carrying a member plus its
// payload; and the table row widens by one column recording whether a member
// takes a payload.
//
// That widening is not an improvisation. AI-05's design.md § 3.2 authorised it
// in advance — "Rule 2's table may carry more than a name … and that is a
// widening of the row, not a change of shape" — and the two rules it calls load
// bearing are untouched here. iota + 1 still makes a mode nobody set fail
// exactly like a wild one. [ToolChoiceModes] still walks the constant space, so
// a member declared without a table entry appears in the enumeration and fails
// the pin rather than being invisible to it.
//
// # Why not put the name on the vocabulary type
//
// Because the vocabulary would stop being an integer, and iota + 1, the indexed
// table and the pin's enumeration would all stop working at once. Making
// "specific" a family of values — one per declared tool — was the other
// candidate and is worse still: the vocabulary's size would depend on the
// request, which is not a closed vocabulary at all.
//
// # What a later vocabulary should take from this
//
// AI-13's finish reasons land the third closed vocabulary. The general move is:
// when a member needs something a bare integer cannot carry, widen the table
// row and put the payload in a separate value type — never widen the vocabulary
// type itself.

package ai

import "strconv"

// ToolChoiceMode is the closed vocabulary a tool choice's member is drawn from.
//
// It is not a provider's own tool-choice string: an adapter maps between the
// two, in both directions, and this package neither guesses nor passes through.
// The zero value is not a member.
type ToolChoiceMode uint8

// The tool-choice vocabulary. Append-only: a new member takes the next value,
// its entry in toolChoiceModes, and — if it carries a payload — a constructor,
// all in the same commit.
const (
	// ToolChoiceAuto lets the model decide whether to call a tool. It is the
	// posture every provider defaults to when tools are offered.
	ToolChoiceAuto ToolChoiceMode = iota + 1

	// ToolChoiceNone forbids the model from calling a tool. It is the only
	// member that is coherent against an empty tool set — see
	// [ToolChoice.ValidateAgainst].
	ToolChoiceNone

	// ToolChoiceRequired obliges the model to call some tool, without saying
	// which.
	ToolChoiceRequired

	// ToolChoiceSpecific obliges the model to call one named tool. It is the
	// vocabulary's only payload-carrying member; construct it with
	// [NewNamedToolChoice].
	ToolChoiceSpecific

	// toolChoiceFirst and toolChoiceEnd bound the declared constant space.
	// They are not members and are never exported: toolChoiceEnd is one past
	// the last member, and moving it is what makes a newly declared member
	// visible to ToolChoiceModes.
	toolChoiceFirst = ToolChoiceAuto
	toolChoiceEnd   = ToolChoiceSpecific + 1
)

// toolChoiceEntry is one row of the vocabulary table: a member's rendering, and
// its arity.
//
// needsName is the extension this milestone makes to AI-05's pattern. It is the
// single source of which constructor a member takes, so [NewToolChoice] and the
// exhaustiveness pin agree by construction rather than by two hard-coded lists
// that could drift.
type toolChoiceEntry struct {
	name      string
	needsName bool
}

// toolChoiceModes is the table, and it is the single source of a mode's
// rendering, its parse mapping, its validity and its arity.
//
// A slice indexed by the constant rather than a map, for AI-04's reason:
// nothing in this package may let an unordered iteration decide anything.
var toolChoiceModes = []toolChoiceEntry{
	ToolChoiceAuto:     {name: "auto"},
	ToolChoiceNone:     {name: "none"},
	ToolChoiceRequired: {name: "required"},
	ToolChoiceSpecific: {name: "specific", needsName: true},
}

// ToolChoiceModes returns the tool-choice vocabulary in declaration order.
//
// The result is a fresh slice on every call: a package-level variable would be
// a closed vocabulary any consumer could rewrite. It enumerates the declared
// constants rather than the table, which is what allows the exhaustiveness pin
// to fail on a member declared without being tabulated.
func ToolChoiceModes() []ToolChoiceMode {
	out := make([]ToolChoiceMode, 0, int(toolChoiceEnd-toolChoiceFirst))
	for m := toolChoiceFirst; m < toolChoiceEnd; m++ {
		out = append(out, m)
	}
	return out
}

// String renders the mode.
//
// A member renders as its stable, lowercase, registered name. Anything else
// renders as "toolChoice(N)", deliberately a shape no parser accepts, so a
// diagnostic rendering can never round-trip into a value. A ToolChoiceMode is
// an integer and carries no caller data, so the rendering is safe to log.
func (m ToolChoiceMode) String() string {
	if entry := toolChoiceEntryOf(m); entry.name != "" {
		return entry.name
	}
	return "toolChoice(" + strconv.FormatUint(uint64(m), 10) + ")"
}

// ParseToolChoiceMode maps a rendering back to its mode.
//
// It is exact: a case variant, a padded form, an alias and a provider's own
// tool-choice string are all rejected. The two rules are checked in the order
// written, per V-FAIL-04 — an empty name reports [ErrEmpty] and an unrecognised
// one reports [ErrNotInVocabulary], because "you gave me nothing" and "you gave
// me something I do not know" are different facts with different fixes.
func ParseToolChoiceMode(name string) (ToolChoiceMode, error) {
	parsed := toolChoiceModeWithName(name)
	if err := FirstFailure(
		func() *Violation {
			if name == "" {
				return Invalid(ErrEmpty, At("toolChoice"))
			}
			return nil
		},
		func() *Violation {
			if parsed == 0 {
				return Invalid(ErrNotInVocabulary, At("toolChoice"))
			}
			return nil
		},
	); err != nil {
		return 0, err
	}
	return parsed, nil
}

// ToolChoice is the request-level instruction constraining whether and which
// tool the model may call (V-REQ-15).
//
// It is a constraint expressed to the model. It is not a decision about whether
// a resulting call is allowed to run — that is V-OUT-05 permission, and it
// belongs to the layers above.
//
// The zero value is not a valid choice: its mode is not a vocabulary member, so
// [ToolChoice.ValidateAgainst] rejects it. Whether a *request* may omit a tool
// choice entirely is a different question and is AI-10.3's; this type models a
// value, not an absence.
type ToolChoice struct {
	mode ToolChoiceMode
	name string
}

// NewToolChoice constructs a tool choice for a member that carries no payload.
//
// The rules are checked in the order written, per V-FAIL-04:
//
//  1. The mode is a member of the vocabulary, else [ErrNotInVocabulary] at
//     "toolChoice".
//  2. The mode does not require a name, else [ErrEmpty] at "name".
//
// Rule 2 is why [ToolChoiceSpecific] fails here rather than quietly producing a
// nameless "specific" choice: the member requires a name and none was supplied,
// which is exactly what the empty rule class means. Use [NewNamedToolChoice].
func NewToolChoice(mode ToolChoiceMode) (ToolChoice, error) {
	if err := FirstFailure(
		func() *Violation {
			if toolChoiceEntryOf(mode).name == "" {
				return Invalid(ErrNotInVocabulary, At("toolChoice"))
			}
			return nil
		},
		func() *Violation {
			if toolChoiceEntryOf(mode).needsName {
				return Invalid(ErrEmpty, At("name"))
			}
			return nil
		},
	); err != nil {
		return ToolChoice{}, err
	}
	return ToolChoice{mode: mode}, nil
}

// NewNamedToolChoice constructs a tool choice naming one specific tool.
//
// The name is checked against the *same* rule a declaration's name is checked
// against, and reports the same classes at the same position. One name rule in
// the package, not two that could drift — and a choice naming a syntactically
// impossible tool can never resolve against any set, so failing it here gives
// the caller the precise fix instead of an unresolved reference later.
//
// The name is not checked against any tool set here. That is
// [ToolChoice.ValidateAgainst], because the set is not known at construction.
func NewNamedToolChoice(name string) (ToolChoice, error) {
	if err := FirstFailure(toolNameRules(name)...); err != nil {
		return ToolChoice{}, err
	}
	return ToolChoice{mode: ToolChoiceSpecific, name: name}, nil
}

// Mode returns the choice's vocabulary member.
func (c ToolChoice) Mode() ToolChoiceMode { return c.mode }

// Name returns the tool the choice names, and whether it names one.
//
// The second result is what distinguishes "names no tool" from "names a tool
// whose name is empty", so no caller has to know that an empty string means
// absence.
func (c ToolChoice) Name() (string, bool) {
	if !toolChoiceEntryOf(c.mode).needsName {
		return "", false
	}
	return c.name, true
}

// ValidateAgainst validates the choice against a declared tool set (V-REQ-15's
// "validated against the declared tool set").
//
// The rules are checked in the order written, per V-FAIL-04, and the order is
// part of the contract:
//
//  1. The mode is a member of the vocabulary, else [ErrNotInVocabulary] at
//     "toolChoice". This is what rejects the zero ToolChoice.
//  2. The mode is anything other than [ToolChoiceNone] and the set is empty,
//     else [ErrEmpty] at "tools".
//  3. The mode is [ToolChoiceSpecific] and the set declares no tool of that
//     name, else [ErrUnresolvedReference] at "toolChoice.name".
//
// Rule 2 precedes rule 3 by choice, and it is the only ordering decision here
// with two defensible answers. Against an empty set a specific choice violates
// both. Reporting rule 3 would say "the tool you named is not declared", which
// is true and unhelpful — no tool is declared. Reporting rule 2 says "you have
// declared no tools at all", which is the more fundamental fact and names the
// thing the caller must fix.
//
// Rule 2 is this contract's most valuable line: a non-none choice against an
// empty tool set is the combination every provider rejects, and catching it
// here costs a comparison instead of a network round trip.
//
// [ToolChoiceNone] against an empty set is legal. Expressing "do not call a
// tool" when there are no tools is coherent, not contradictory.
//
// This is not the only place these rules run. AI-10.3 re-runs them at the
// request boundary, which is why they live on a method a request can call
// rather than inlined into a constructor.
func (c ToolChoice) ValidateAgainst(tools ToolSet) error {
	return FirstFailure(
		func() *Violation {
			if toolChoiceEntryOf(c.mode).name == "" {
				return Invalid(ErrNotInVocabulary, At("toolChoice"))
			}
			return nil
		},
		func() *Violation {
			if c.mode != ToolChoiceNone && tools.Len() == 0 {
				return Invalid(ErrEmpty, At("tools"))
			}
			return nil
		},
		func() *Violation {
			if c.mode == ToolChoiceSpecific && !tools.Declares(c.name) {
				return Invalid(ErrUnresolvedReference, At("toolChoice"), At("name"))
			}
			return nil
		},
	)
}

// toolChoiceEntryOf returns a mode's table row, or the zero row when the mode
// has no entry — which is exactly what makes it not a member.
//
// The bounds check is why a mode declared past the end of the table is rejected
// rather than panicking, and therefore why a scratch member added without a
// table entry fails the pin instead of crashing it.
func toolChoiceEntryOf(m ToolChoiceMode) toolChoiceEntry {
	if int(m) >= len(toolChoiceModes) {
		return toolChoiceEntry{}
	}
	return toolChoiceModes[m]
}

// toolChoiceModeWithName returns the mode a rendering names, or the zero mode
// when no member carries that exact rendering.
//
// The scan walks the ordered table, so the answer does not depend on iteration
// order even if two entries were ever to collide.
func toolChoiceModeWithName(name string) ToolChoiceMode {
	for m, entry := range toolChoiceModes {
		if entry.name != "" && entry.name == name {
			return ToolChoiceMode(m)
		}
	}
	return 0
}
