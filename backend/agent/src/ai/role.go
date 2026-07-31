// AI-05.1 — the closed role vocabulary.
//
// This file implements V-REQ-01 role: "the closed, provider-neutral vocabulary
// value naming who a message is attributed to." Closed is the operative word,
// and here it is a runtime property rather than a documented intention — a
// value outside the vocabulary is a caller-contract failure, reported through
// AI-04's taxonomy, and never an unknown passed through.
//
// # The vocabulary is three members, and the absent fourth is deliberate
//
// user, assistant and tool. Each has a citable case: doc 0001 § 3.3 row 5 for
// the first two ("only one provider enforces strict user/assistant
// alternation"), and doc 0002's AI-10.3 item 3 for the third, which asks AI-10
// to decide "whether a tool-result part may appear in a non-tool role" — a
// question that presupposes a tool role.
//
// There is no system role. V-REQ-19 gives the system instruction its own region
// of the request, segmented from birth and owned by AI-10; doc 0001 § 3.3 row 8
// records that every provider takes it as a top-level shape an adapter renders.
// A system role would give Layer 1 two ways to express one fact. The vocabulary
// is append-only, so the milestone that meets the case adds a constant and its
// table entry together — and the exhaustiveness pin in role_test.go is what
// makes "together" mechanical rather than remembered.
//
// # The pattern, which three later milestones reuse
//
// AI-07 reasoning state, AI-08 tool choice and AI-13 finish reasons each land a
// closed vocabulary. doc 0002 calls this one "the pattern every later closed
// vocabulary in this package reuses", so it is written to be copied, and the
// two rules that carry it belong where they are enforced rather than only in
// the SDD:
//
//  1. The constants start at iota + 1, so a field nobody set holds a value that
//     is rejected exactly like a wild one.
//  2. [Roles] walks the constant space, not the name table. An enumeration
//     derived from the table lists exactly the members that have an entry, so a
//     member declared without one would be invisible to every assertion over
//     the enumeration, and the exhaustiveness pin would be decorative.
package ai

import "strconv"

// Role is the closed, provider-neutral vocabulary value naming who a message is
// attributed to.
//
// It is not a provider's own role string: an adapter maps between the two, in
// both directions, and this package neither guesses nor passes through. The
// zero value is not a member — see the file comment.
type Role uint8

// The role vocabulary. Append-only: a new member takes the next value and its
// entry in roleNames, in the same commit.
const (
	// RoleUser attributes a message to the caller of the model.
	RoleUser Role = iota + 1

	// RoleAssistant attributes a message to the model.
	RoleAssistant

	// RoleTool attributes a message to a tool's answer. Whether a tool-result
	// part may appear in another role is AI-10.3's rule, not this file's.
	RoleTool

	// roleFirst and roleEnd bound the declared constant space. They are not
	// members and are never exported: roleEnd is one past the last role, and
	// moving it is what makes a newly declared member visible to Roles.
	roleFirst = RoleUser
	roleEnd   = RoleTool + 1
)

// roleNames is the table, and it is the single source of a role's rendering,
// its parse mapping and its validity.
//
// It is a slice indexed by the constant itself rather than a map, for AI-04's
// reason: nothing in this package may let an unordered iteration decide
// anything, and a registry is where that temptation is strongest. Indexing by
// the constant also puts the pairing in front of a reader without counting
// positions.
var roleNames = []string{
	RoleUser:      "user",
	RoleAssistant: "assistant",
	RoleTool:      "tool",
}

// Roles returns the role vocabulary in declaration order.
//
// The result is a fresh slice on every call: a package-level variable would be
// a closed vocabulary that any consumer could rewrite. The order is the
// declaration order and is stable, so a caller may index it, range over it, or
// compare two calls for equality.
//
// It enumerates the declared constants rather than the table — see the file
// comment, rule 2. That is what allows the exhaustiveness pin to fail on a
// member that was declared without being tabulated.
func Roles() []Role {
	out := make([]Role, 0, int(roleEnd-roleFirst))
	for r := roleFirst; r < roleEnd; r++ {
		out = append(out, r)
	}
	return out
}

// String renders the role.
//
// A member renders as its stable, lowercase, registered name. Anything else
// renders as "role(N)", deliberately a shape no parser accepts, so a diagnostic
// rendering can never round-trip into a value. A Role is an integer and carries
// no caller data, so the rendering is safe to log.
func (r Role) String() string {
	if name := roleName(r); name != "" {
		return name
	}
	return "role(" + strconv.FormatUint(uint64(r), 10) + ")"
}

// ParseRole maps a rendering back to its role.
//
// It is exact. A case variant, a padded form, an alias and a provider's own
// role string are all rejected: V-REQ-01's last clause puts that mapping in an
// adapter, and a parser that guesses is how a closed vocabulary becomes
// advisory.
//
// The two rules are checked in the order written, per V-FAIL-04: an empty name
// reports [ErrEmpty] and an unrecognised one reports [ErrNotInVocabulary],
// because "you gave me nothing" and "you gave me something I do not know" are
// different facts with different fixes. Neither failure reproduces the offered
// string — the position is a constant this package wrote, which is the whole of
// what AI-04's redaction posture costs at a call site.
func ParseRole(name string) (Role, error) {
	parsed := roleWithName(name)
	if err := FirstFailure(
		func() *Violation {
			if name == "" {
				return Invalid(ErrEmpty, At("role"))
			}
			return nil
		},
		func() *Violation {
			if parsed == 0 {
				return Invalid(ErrNotInVocabulary, At("role"))
			}
			return nil
		},
	); err != nil {
		return 0, err
	}
	return parsed, nil
}

// roleName returns a role's registered rendering, or "" when the role has no
// table entry — which is exactly what makes it not a member of the vocabulary.
//
// The bounds check is why a role declared past the end of the table is rejected
// rather than panicking, and therefore why a scratch member added without a
// table entry fails the pin instead of crashing it.
func roleName(r Role) string {
	if int(r) >= len(roleNames) {
		return ""
	}
	return roleNames[r]
}

// roleWithName returns the role a rendering names, or the zero role when no
// member carries that exact rendering.
//
// The scan walks the ordered table, so the answer does not depend on iteration
// order even if two entries were ever to collide.
func roleWithName(name string) Role {
	for r, registered := range roleNames {
		if registered != "" && registered == name {
			return Role(r)
		}
	}
	return 0
}
