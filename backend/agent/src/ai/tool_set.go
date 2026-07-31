// AI-08.2 — the tool set.

package ai

import "slices"

// ToolSet is the ordered, deterministically iterable collection of tool
// declarations offered on one request (V-REQ-14).
//
// # Why a slice, and why that is the whole design
//
// The failure mode V-REQ-14 names is the nastiest kind. A map-backed tool set
// produces no error, no crash and no wrong answer — it produces a cache prefix
// that differs on every call, because V-REQ-25 makes the tool set the first
// region a change invalidates. The only symptom is an input bill roughly ten
// times larger than it should be. A comment saying "keep this ordered" is not a
// countermeasure; being a slice is.
//
// Nothing in this type lets an unordered iteration decide anything, which is
// AI-04's own rule about registries applied to the one collection this package
// exports.
//
// # The zero value
//
// The zero ToolSet is the empty set, and that is correct rather than
// convenient. An empty tool set is legal, so unlike a closed vocabulary — whose
// zero value must be invalid — here the zero value is a member of the legal
// domain.
type ToolSet struct {
	tools []Tool
}

// NewToolSet constructs a tool set from an ordered sequence of declarations.
//
// A name that repeats an earlier one fails with [ErrMalformed] positioned at
// the *second* occurrence by index. The position is an index and not a name
// because V-REQ-14 makes the set deterministically iterable, so an index
// identifies one declaration unambiguously and the caller — which holds the set
// — resolves it in one step, without a caller-supplied value reaching a message
// that will be logged.
//
// [ErrMalformed] is the reading rather than a sixth sentinel: a duplicate is
// cleanly none of AI-04's five classes, and AI-04's decision record is explicit
// that the set grows only in the pull request that needs a class. It is not a
// bound, and the set is not a closed vocabulary; "the value you gave is not
// well-formed for what this field must be" describes a set with a repeated name
// accurately, and the position says which element broke it.
//
// A declaration that never passed [NewTool] is rejected with [ErrEmpty] at its
// index, before the duplicate check. Its name is empty, which no constructed
// declaration's is, so emptiness is a sound detector without a separate flag —
// and without it a value that skipped the constructor would reach the wire
// carrying a name no provider accepts.
//
// The sequence is copied, so a caller may reuse the slice it passed.
//
// The scan is linear over the ordered sequence rather than a map lookup, for
// AI-04's reason: nothing in this package may let an unordered iteration decide
// anything, and a registry is where that temptation is strongest. It costs
// O(n squared) over a collection whose realistic size is single or low double
// digits, and it buys a deterministic answer — the reported duplicate is always
// the lowest index whose name repeats an earlier one.
func NewToolSet(tools ...Tool) (ToolSet, error) {
	for i, tool := range tools {
		if tool.Name() == "" {
			return ToolSet{}, Invalid(ErrEmpty, AtIndex("tools", i))
		}
		for j := range i {
			if tools[j].Name() == tool.Name() {
				return ToolSet{}, Invalid(ErrMalformed, AtIndex("tools", i))
			}
		}
	}
	return ToolSet{tools: slices.Clone(tools)}, nil
}

// Tools returns the set's declarations in the order the caller supplied.
//
// The order is the caller's, not merely a stable one, and the difference
// matters: a stable-but-arbitrary order satisfies self-comparison and still
// breaks the property V-REQ-25 needs, which is that a caller who builds the
// same set twice obtains the same cache prefix.
//
// The result is a fresh slice on every call, for message.go's reason: a
// consumer that rewrites what it received must not be able to rewrite the set,
// and two consumers must not be able to observe each other.
func (s ToolSet) Tools() []Tool { return slices.Clone(s.tools) }

// Len returns the number of declarations the set holds.
func (s ToolSet) Len() int { return len(s.tools) }

// Declares reports whether the set declares a tool with the given name.
//
// The match is exact: not case-folded, not trimmed, not aliased. It is the
// question V-REQ-15's cross-validation asks, and it deliberately does not
// return the declaration — a lookup that handed back the tool would invite a
// consumer to resolve a name to an implementation, which is V-OUT-04's and not
// this layer's.
//
// The scan walks the ordered sequence, so the answer never depends on iteration
// order.
func (s ToolSet) Declares(name string) bool {
	for _, tool := range s.tools {
		if tool.Name() == name {
			return true
		}
	}
	return false
}
