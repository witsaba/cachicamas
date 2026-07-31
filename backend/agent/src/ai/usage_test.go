// Tests for AI-13.3 and AI-13.4 — the usage record.
//
// External test package, so every assertion is written against exactly the
// surface an adapter and a Layer 2 consumer see. That matters more here than
// anywhere else in this milestone: the property under test is that an
// unreported count cannot be mistaken for a reported nought, and a test with
// access to unexported state could prove that about a representation while a
// consumer still could not observe it.
package ai_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// usageField names one token count and gives the test a way to set and read it
// without reflection, so that every property below is asserted once per field
// rather than once on whichever field happened to be written first.
//
// V-MET-10 makes each count independently present or absent. A property proven
// on Input alone would be a property of Input.
type usageField struct {
	name string
	set  func(*ai.Usage, ai.TokenCount)
	get  func(ai.Usage) ai.TokenCount
}

// usageFields is the five counts of V-MET-09, in the order the register lists
// them, which is also the struct's field order and the order Validate reports.
var usageFields = []usageField{
	{"input", func(u *ai.Usage, c ai.TokenCount) { u.Input = c }, func(u ai.Usage) ai.TokenCount { return u.Input }},
	{"output", func(u *ai.Usage, c ai.TokenCount) { u.Output = c }, func(u ai.Usage) ai.TokenCount { return u.Output }},
	{"cache_read", func(u *ai.Usage, c ai.TokenCount) { u.CacheRead = c }, func(u ai.Usage) ai.TokenCount { return u.CacheRead }},
	{"cache_write", func(u *ai.Usage, c ai.TokenCount) { u.CacheWrite = c }, func(u ai.Usage) ai.TokenCount { return u.CacheWrite }},
	{"reasoning", func(u *ai.Usage, c ai.TokenCount) { u.Reasoning = c }, func(u ai.Usage) ai.TokenCount { return u.Reasoning }},
}

// AI-13.3 — an absent token count is distinguishable from a zero one.
//
// V-MET-11: "not reported" and "reported as nought" are different facts, and a
// consumer that cannot tell them apart writes a wrong cost formula and a wrong
// compaction estimate. The assertion runs once per count because the register
// makes each of them independently present or absent.
func TestUsage_AnAbsentCount_IsDistinguishableFromZero(t *testing.T) {
	t.Parallel()

	for _, field := range usageFields {
		t.Run(field.name, func(t *testing.T) {
			t.Parallel()

			var absent ai.Usage
			count, present := field.get(absent).Count()
			if present {
				t.Errorf("absent %s: Count() = (%d, true), want (0, false) — an unreported count must not report as present", field.name, count)
			}
			if count != 0 {
				t.Errorf("absent %s: Count() = (%d, _), want 0 as the uninformative value", field.name, count)
			}

			var reportedZero ai.Usage
			field.set(&reportedZero, ai.Tokens(0))
			count, present = field.get(reportedZero).Count()
			if !present {
				t.Errorf("%s reported as nought: Count() = (_, false), want (0, true)", field.name)
			}
			if count != 0 {
				t.Errorf("%s reported as nought: Count() = (%d, _), want 0", field.name, count)
			}

			if got, want := field.get(absent).String(), field.get(reportedZero).String(); got == want {
				t.Errorf("absent %s and %s reported as nought both render as %q, want two renderings", field.name, field.name, got)
			}
		})
	}

	t.Run("a present count carries its value", func(t *testing.T) {
		t.Parallel()

		count, present := ai.Tokens(1234).Count()
		if !present || count != 1234 {
			t.Errorf("Tokens(1234).Count() = (%d, %t), want (1234, true)", count, present)
		}
	})
}

// AI-13.3 — a usage record is constructible with any subset of counts present.
//
// AI-03's CAP-R-03 clause 2: "an adapter that reports only input and output
// produces a valid usage record, not a deficient one … requiring a populated
// count is requiring a fabricated one". So the interesting assertion is the
// negative one — that validation has no opinion about presence at all — and the
// only rule it does enforce is that a reported count is not negative.
func TestUsage_AnySubsetOfCounts_ProducesAValidRecord(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name  string
		usage ai.Usage
	}{
		{"a provider that reports nothing", ai.Usage{}},
		{"a provider that reports only input and output", ai.Usage{Input: ai.Tokens(1200), Output: ai.Tokens(340)}},
		{"a provider that reports only a reported nought", ai.Usage{CacheRead: ai.Tokens(0)}},
		{"a provider that reports everything", ai.Usage{
			Input:      ai.Tokens(100),
			Output:     ai.Tokens(500),
			CacheRead:  ai.Tokens(900),
			CacheWrite: ai.Tokens(0),
			Reasoning:  ai.Tokens(380),
		}},
	}

	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.usage.Validate(ai.At("usage")); err != nil {
				t.Errorf("Validate() = %v, want <nil> — any subset of counts is a valid record", err)
			}
		})
	}

	t.Run("an all-absent record reports absence rather than zero", func(t *testing.T) {
		t.Parallel()

		var reported ai.Usage
		for _, field := range usageFields {
			if _, present := field.get(reported).Count(); present {
				t.Errorf("%s of an all-absent record reports as present, want absent", field.name)
			}
		}
	})

	t.Run("a negative count is rejected out of range at its own field", func(t *testing.T) {
		t.Parallel()

		usage := ai.Usage{Input: ai.Tokens(10), CacheRead: ai.Tokens(-1)}

		err := usage.Validate(ai.At("completion"), ai.At("usage"))
		if err == nil {
			t.Fatalf("Validate() = <nil>, want a violation for a negative count")
		}
		if !errors.Is(err, ai.ErrOutOfRange) {
			t.Errorf("errors.Is(err, ErrOutOfRange) = false, want true; err = %v", err)
		}

		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &*ai.Violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "completion.usage.cache_read"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("several negative counts report the first in the documented order", func(t *testing.T) {
		t.Parallel()

		usage := ai.Usage{
			Output:    ai.Tokens(-3),
			CacheRead: ai.Tokens(-1),
			Reasoning: ai.Tokens(-7),
		}

		var violation *ai.Violation
		if err := usage.Validate(ai.At("usage")); !errors.As(err, &violation) {
			t.Fatalf("Validate() = %v, want a violation", err)
		}
		if got, want := violation.Path().String(), "usage.output"; got != want {
			t.Errorf("violation position = %q, want %q — output precedes cache_read and reasoning", got, want)
		}
	})
}

// AI-13.3 — a usage record is readable from another package, field by field,
// with absence surfaced rather than defaulted.
//
// Written without the usageFields helper on purpose: this is the shape a real
// consumer writes, and the point of the item is that the shape works from
// outside the package. Retired defect C2 was a contract that could not be read
// from another package at all, which made translation structurally impossible;
// this test is the equivalent proof for the usage record, one milestone before
// AI-06.2 makes it for content parts.
func TestUsage_FromAnExternalPackage_IsReadableFieldByField(t *testing.T) {
	t.Parallel()

	// A provider that reported four of the five counts, one of them as nought.
	usage := ai.Usage{
		Input:     ai.Tokens(1200),
		Output:    ai.Tokens(340),
		CacheRead: ai.Tokens(0),
		Reasoning: ai.Tokens(96),
	}

	reads := []struct {
		name        string
		count       ai.TokenCount
		wantValue   int64
		wantPresent bool
	}{
		{"input", usage.Input, 1200, true},
		{"output", usage.Output, 340, true},
		{"cache_read reported as nought", usage.CacheRead, 0, true},
		{"cache_write never reported", usage.CacheWrite, 0, false},
		{"reasoning", usage.Reasoning, 96, true},
	}

	for _, read := range reads {
		value, present := read.count.Count()
		if value != read.wantValue || present != read.wantPresent {
			t.Errorf("%s: Count() = (%d, %t), want (%d, %t)",
				read.name, value, present, read.wantValue, read.wantPresent)
		}
	}

	t.Run("a copy of the record is a copy of its counts", func(t *testing.T) {
		t.Parallel()

		copied := usage
		copied.Input = ai.Tokens(7)

		if value, _ := usage.Input.Count(); value != 1200 {
			t.Errorf("writing to a copy changed the original: input = %d, want 1200", value)
		}
	})
}
