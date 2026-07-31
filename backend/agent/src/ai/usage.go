// AI-13.3 and AI-13.4 — the usage record.
//
// This file implements V-MET-09 … V-MET-12: what a response consumed. Layer 1
// reports usage; it never prices it (V-OUT-07, V-OUT-08).
//
// # Absence is not zero, and that is the whole design
//
// V-MET-11: "not reported" and "reported as nought" are different facts, and a
// consumer that cannot tell them apart writes a wrong cost formula and a wrong
// compaction estimate. V-MET-10 makes each count independently present or
// absent, because providers report different subsets, and AI-03's CAP-R-03 draws
// the consequence: requiring the usage record does not require any count to be
// populated, because "requiring a populated count is requiring a fabricated
// one".
//
// So absence is the zero value of [TokenCount], and a zero [Usage] is a valid
// record whose five counts are all absent — exactly what an adapter that
// reports nothing should produce, with no constructor to remember and no way to
// build a record of fabricated zeros by accident.
//
// # The distinction an adapter must keep
//
// Absent means "the provider did not tell me". It does not mean "none". An
// adapter therefore leaves a count absent when its transcript never carried it,
// and does not substitute a zero to make a record look complete.

package ai

import (
	"slices"
	"strconv"
)

// TokenCount is one counted quantity within a usage record (V-MET-10).
//
// Its zero value is **absent**, which is the fact an adapter reports when the
// provider said nothing. A count reported as nought is a different value, built
// with [Tokens], and the two are distinguishable through [TokenCount.Count] and
// through their rendering.
//
// It is a value type rather than a pointer so that a copy of a usage record is a
// copy of its counts: two records can never be made to share one, and no call
// site can dereference an absent count.
type TokenCount struct {
	count   int64
	present bool
}

// Tokens builds a token count a provider reported.
//
// Tokens(0) is a **reported nought** and is not the same value as an absent
// count.
func Tokens(n int64) TokenCount {
	return TokenCount{count: n, present: true}
}

// Count returns the count and whether the provider reported it.
//
// The two-result shape is this package's idiom for a value that may not be
// there — [Step.Index] has it too — and it exists so that the count cannot be
// read without meeting its presence. A consumer that ignores the second result
// gets 0, which is the uninformative value rather than a plausible one.
func (t TokenCount) Count() (int64, bool) {
	return t.count, t.present
}

// String renders the count, or the fact that there is none.
//
// An absent count and a count reported as nought render differently, so the
// distinction survives into a log line, where it is otherwise the first thing
// lost.
func (t TokenCount) String() string {
	if !t.present {
		return absentTokenCount
	}
	return strconv.FormatInt(t.count, 10)
}

// absentTokenCount renders a count no provider reported.
const absentTokenCount = "absent"

// Usage is the record of what a response consumed (V-MET-09).
//
// Every field is a [TokenCount], so the zero Usage is a valid record with all
// five counts absent, and any subset can be populated by naming it:
//
//	usage := ai.Usage{Input: ai.Tokens(1200), Output: ai.Tokens(340)}
type Usage struct {
	Input      TokenCount
	Output     TokenCount
	CacheRead  TokenCount
	CacheWrite TokenCount
	Reasoning  TokenCount
}

// Validate reports the first rule this usage record violates, positioned at the
// given path, or nil when it violates none.
//
// The only rule is that a reported count is not negative — a caller-contract
// failure of class [ErrOutOfRange]. There is deliberately no rule requiring a
// count to be present: CAP-R-03 makes an all-absent record valid rather than
// deficient, because "requiring a populated count is requiring a fabricated
// one".
//
// The fields are checked in the order V-MET-09 lists them, which is also their
// declaration order, and the first failure is the one reported (V-FAIL-04).
func (u Usage) Validate(at ...Step) error {
	return FirstFailure(
		nonNegative(u.Input, at, "input"),
		nonNegative(u.Output, at, "output"),
		nonNegative(u.CacheRead, at, "cache_read"),
		nonNegative(u.CacheWrite, at, "cache_write"),
		nonNegative(u.Reasoning, at, "reasoning"),
	)
}

// nonNegative builds the rule that a reported count is not negative, at the
// named field of the given position.
//
// The position is cloned rather than appended to in place, and the reason is
// narrower than it first looks. It is *not* that the five rules would overwrite
// each other: [FirstFailure] evaluates lazily and stops at the first failure,
// and [Invalid] copies the position it is given, so at most one rule ever builds
// one. The tests pass either way, and that was checked rather than assumed.
//
// It is that `at` is the caller's slice. Appending to a variadic parameter can
// write into the caller's backing array past its length, so a caller that keeps
// and reuses that array would find a step in it that it never put there. Cloning
// keeps this position purely local, which is the same posture AI-04 took when it
// made [Invalid] copy and [Violation.Path] return a copy.
func nonNegative(count TokenCount, at []Step, name string) Rule {
	return func() *Violation {
		if n, present := count.Count(); present && n < 0 {
			return Invalid(ErrOutOfRange, append(slices.Clone(at), At(name))...)
		}
		return nil
	}
}
