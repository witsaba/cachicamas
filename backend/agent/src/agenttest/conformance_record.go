// AI-23.1 (types stub) / AI-23.6 (Verdict, comparison) — the capability
// record: AI-03 §10's four-value outcome set and totality, implemented as
// this suite emits it (R-CNF-017, R-CNF-018).
//
// AI-23.1 needs CapabilityRecord, Standing and Outcome to exist so
// conformance_suite.go's runner compiles and can initialize every entry to
// OutcomeNotExercised (design.md's stated zero-value posture, D5): this file
// carries exactly that much at AI-23.1. Verdict() and entry-by-entry
// comparison are AI-23.6's own deliverable, appended once every
// case-producing leaf exists to make the record total in practice, not only
// by construction (see this milestone's tasks.md "Implementation-order
// note").

package agenttest

import "fmt"

// Standing is AI-03 §5/§6's classification of a capability — required or
// optional — never supplied by a run (R-CNF-017: "standing comes from AI-03
// alone").
type Standing uint8

const (
	// StandingRequired is every capability outside AI-03 §6's optional list,
	// including one this suite has not classified at all (§11's default).
	StandingRequired Standing = iota + 1

	// StandingOptional is exactly CAP-O-01…03 (AI-03 §6).
	StandingOptional
)

var standingNames = [...]string{
	StandingRequired: "required",
	StandingOptional: "optional",
}

// String renders the standing, or "standing(N)" for a value outside the
// two-member vocabulary — a shape no parser accepts, matching this module's
// package-wide closed-vocabulary idiom.
func (s Standing) String() string {
	if int(s) < len(standingNames) && standingNames[s] != "" {
		return standingNames[s]
	}
	return fmt.Sprintf("standing(%d)", uint8(s))
}

// Outcome is AI-03 §10's closed four-value outcome set. The zero value is
// not a member — design.md D5, the package-wide "zero names nothing" idiom
// restated for this vocabulary — so OutcomeNotExercised is an explicit
// fourth member rather than the type's zero value, which would let a
// record built without going through this suite's own constructor
// silently read as "not exercised" instead of "never built".
type Outcome uint8

const (
	// OutcomeSatisfied — the cases were exercised and the provider met them.
	// Legal for required and optional entries.
	OutcomeSatisfied Outcome = iota + 1

	// OutcomeAbsent — the provider was asked, does not advertise the
	// capability, and its cases were deliberately not exercised. A
	// conclusion, legal for optional entries only.
	OutcomeAbsent

	// OutcomeFailed — the cases were exercised and the provider did not
	// meet them. Legal for required and optional entries; sticky once set
	// (AI-03 §10: "there are no waivers").
	OutcomeFailed

	// OutcomeNotExercised — the cases did not run: a missing transcript, a
	// harness error, an interrupted run. Never a conformance outcome, and
	// never legal for either standing at verdict time (R-CNF-018).
	OutcomeNotExercised
)

var outcomeNames = [...]string{
	OutcomeSatisfied:    "satisfied",
	OutcomeAbsent:       "absent",
	OutcomeFailed:       "failed",
	OutcomeNotExercised: "not_exercised",
}

// String renders the outcome, or "outcome(N)" for a value outside the
// closed set.
func (o Outcome) String() string {
	if int(o) < len(outcomeNames) && outcomeNames[o] != "" {
		return outcomeNames[o]
	}
	return fmt.Sprintf("outcome(%d)", uint8(o))
}

// CapabilityRecordEntry is one capability's row of the record: its
// capability identifier, its standing (from AI-03, never from the run) and
// its outcome (from the run, never from AI-03).
type CapabilityRecordEntry struct {
	Capability Capability
	Standing   Standing
	Outcome    Outcome
}

// String renders the entry for a diagnostic reader. It carries no
// capability-specific detail, model content or credential — R-CNF-017's own
// restriction — because there is none to carry: a capability identifier, a
// standing and an outcome are the entry's whole content.
func (e CapabilityRecordEntry) String() string {
	return fmt.Sprintf("%s(standing=%s outcome=%s)", e.Capability, e.Standing, e.Outcome)
}

// CapabilityRecord is AI-03 §10's capability record as this suite emits it:
// a subject identifier and exactly one entry per capability in AI-03's two
// closed lists — eight entries, always (R-CNF-017). It carries no
// capability-specific detail, no model content, no credential and no raw
// provider text: the entries are its entire content. The array is built
// once by newCapabilityRecord and mutated only through the record's own
// methods, so a capability with no entry is structurally impossible — built
// from the enumerator, never appended ad hoc.
type CapabilityRecord struct {
	subject string
	entries [8]CapabilityRecordEntry
}

// newCapabilityRecord builds a record for subject with exactly one entry
// per capability in Capabilities()' order, standing taken from
// Capability.Optional() alone, and every outcome initialized to
// OutcomeNotExercised (design.md: "entries initialize to OutcomeNotExercised").
// This is the record's only constructor; every entry a caller ever reads
// was placed here, from the enumerator, never appended ad hoc.
func newCapabilityRecord(subject string) CapabilityRecord {
	r := CapabilityRecord{subject: subject}
	for i, c := range Capabilities() {
		r.entries[i] = CapabilityRecordEntry{Capability: c, Standing: standingOf(c), Outcome: OutcomeNotExercised}
	}
	return r
}

// Subject reports which subject produced this record.
func (r CapabilityRecord) Subject() string { return r.subject }

// Entries returns a fresh copy of the record's eight entries, in
// Capabilities()' order (R-CNF-017 totality). Every call allocates a new
// slice, so a caller that mutates what it received cannot alter the record.
func (r CapabilityRecord) Entries() []CapabilityRecordEntry {
	out := make([]CapabilityRecordEntry, len(r.entries))
	copy(out, r.entries[:])
	return out
}

// Entry returns c's entry and whether the record carries one. It reports
// false only for a capability outside AI-03's two closed lists (including
// CapNone and the zero value): every real capability always has an entry,
// by construction.
func (r CapabilityRecord) Entry(c Capability) (CapabilityRecordEntry, bool) {
	if i := r.entryIndex(c); i >= 0 {
		return r.entries[i], true
	}
	return CapabilityRecordEntry{}, false
}

// entryIndex returns c's index into r.entries, or -1 when c has none.
func (r CapabilityRecord) entryIndex(c Capability) int {
	for i, e := range r.entries {
		if e.Capability == c {
			return i
		}
	}
	return -1
}

// setOutcome moves c's entry toward o, honouring the record's merge rule so
// that several cases sharing one capability converge on a single, correct
// final outcome (R-CNF-017's one-entry-per-capability totality, applied
// across possibly many cases per capability):
//
//   - OutcomeFailed is sticky: once an entry fails, nothing overrides it —
//     AI-03 §10's "there are no waivers", restated at the entry level.
//   - OutcomeSatisfied only advances a not-exercised entry (an entry
//     already satisfied stays satisfied; an entry already absent is left
//     alone, since a properly skipped capability's cases never actually
//     run to report a contradicting satisfied).
//   - OutcomeAbsent only advances a not-exercised entry.
//
// A capability with no entry (outside both closed lists) is silently a
// no-op: such a case's own defect is reported by its caller, not by this
// bookkeeping method.
func (r *CapabilityRecord) setOutcome(c Capability, o Outcome) {
	i := r.entryIndex(c)
	if i < 0 {
		return
	}
	if r.entries[i].Outcome == OutcomeFailed {
		return
	}
	switch o {
	case OutcomeFailed:
		r.entries[i].Outcome = OutcomeFailed
	case OutcomeSatisfied:
		if r.entries[i].Outcome != OutcomeAbsent {
			r.entries[i].Outcome = OutcomeSatisfied
		}
	case OutcomeAbsent:
		if r.entries[i].Outcome == OutcomeNotExercised {
			r.entries[i].Outcome = OutcomeAbsent
		}
	case OutcomeNotExercised:
		// Never regresses an entry backward to not-exercised.
	}
}

// recordCaseResult folds one case's pass/fail verdict for capability c into
// the record, through setOutcome's merge rule.
func (r *CapabilityRecord) recordCaseResult(c Capability, passed bool) {
	if passed {
		r.setOutcome(c, OutcomeSatisfied)
	} else {
		r.setOutcome(c, OutcomeFailed)
	}
}

// Verdict is the suite's own three-way reading of AI-03 §10's four-value
// outcome set — pass, fail or inconclusive — computed mechanically from a
// CapabilityRecord (R-CNF-018). The four-value outcome set itself is never
// collapsed to three at the entry level; Verdict only ever summarizes it.
type Verdict uint8

const (
	// VerdictPass — every entry is satisfied, or (for an optional entry
	// only) absent, and no entry is not exercised.
	VerdictPass Verdict = iota + 1

	// VerdictFail — at least one entry is failed. This is checked first
	// and wins unconditionally: a failed required entry can never pass,
	// and no optional result offsets it (AI-03 §10's own rule); a failed
	// optional entry is equally a real signal — advertising binds
	// (R-AMP-019) — so it fails the verdict too, not only a required one.
	VerdictFail

	// VerdictInconclusive — no entry failed, but at least one is not
	// exercised: the run cannot say whether the subject conforms. Neither
	// a pass nor a failure, and never silently read as either.
	VerdictInconclusive
)

var verdictNames = [...]string{
	VerdictPass:         "pass",
	VerdictFail:         "fail",
	VerdictInconclusive: "inconclusive",
}

// String renders the verdict, or "verdict(N)" for a value outside this
// three-member vocabulary.
func (v Verdict) String() string {
	if int(v) < len(verdictNames) && verdictNames[v] != "" {
		return verdictNames[v]
	}
	return fmt.Sprintf("verdict(%d)", uint8(v))
}

// Verdict computes AI-03 §10's verdict rule over r, mechanically, with
// exactly one code path per outcome combination — no case falls through
// to an implicit default (R-CNF-018's own REFACTOR confirmation):
//
//  1. Any entry Failed → VerdictFail, checked first and unconditionally —
//     no later, better-looking entry ever offsets it.
//  2. Otherwise, any entry that is NOT Satisfied and NOT Absent →
//     VerdictInconclusive. This covers OutcomeNotExercised by name, and —
//     defensively — any value outside the closed four-member outcome set,
//     including Outcome's own zero value: a zero-value, never-constructed
//     CapabilityRecord (NFR-CNF-E's own extreme input) MUST NOT silently
//     read as a pass just because none of its entries happen to equal
//     OutcomeFailed.
//  3. Otherwise → VerdictPass.
func (r CapabilityRecord) Verdict() Verdict {
	for _, e := range r.entries {
		if e.Outcome == OutcomeFailed {
			return VerdictFail
		}
	}
	for _, e := range r.entries {
		if e.Outcome != OutcomeSatisfied && e.Outcome != OutcomeAbsent {
			return VerdictInconclusive
		}
	}
	return VerdictPass
}

// CapabilityDifference is one capability whose outcome differs between two
// compared records, naming both sides so the direction of the difference
// is readable without a second lookup.
type CapabilityDifference struct {
	Capability Capability
	Got, Want  Outcome
}

// String renders the difference as "<capability>: got <outcome>, want
// <outcome>".
func (d CapabilityDifference) String() string {
	return fmt.Sprintf("%v: got %v, want %v", d.Capability, d.Got, d.Want)
}

// CompareCapabilityRecords compares got against want entry by entry and
// returns every capability whose outcome differs, in Capabilities()'
// order, naming both sides — AI-24.1's "records are comparable entry by
// entry" requirement (AI-03 §10), read from this suite's side: an
// unexpected absent is a regression, an unexpected satisfied means the
// adapter grew a capability nobody reviewed, and either direction is
// nameable from the result alone. A nil result means the two records agree
// on every entry.
func CompareCapabilityRecords(got, want CapabilityRecord) []CapabilityDifference {
	var diffs []CapabilityDifference
	for _, wantEntry := range want.entries {
		gotEntry, ok := got.Entry(wantEntry.Capability)
		if !ok || gotEntry.Outcome != wantEntry.Outcome {
			diffs = append(diffs, CapabilityDifference{
				Capability: wantEntry.Capability,
				Got:        gotEntry.Outcome,
				Want:       wantEntry.Outcome,
			})
		}
	}
	return diffs
}
