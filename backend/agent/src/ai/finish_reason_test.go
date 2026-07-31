// Tests for AI-13.1 and AI-13.2 — the finish-reason vocabulary.
//
// The package under test is imported by its module path from the external test
// package ai_test, so every assertion is written against exactly the surface a
// Layer 2 consumer sees. Nothing here can reach an unexported value, which is
// what makes the exhaustiveness pin meaningful: it discovers the vocabulary the
// way a consumer would, rather than reading a list the package handed it.
package ai_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// theVocabulary is the seven values of V-MET-02 … V-MET-08, named by hand.
//
// Written out rather than obtained from the package, and that is the point: the
// pin at the bottom of this file compares this hand-written list against the set
// the package actually treats as its vocabulary. A list the package supplied
// would agree with itself no matter what was added to it.
var theVocabulary = []ai.FinishReason{
	ai.FinishReasonStop,
	ai.FinishReasonLength,
	ai.FinishReasonToolCalls,
	ai.FinishReasonContentFilter,
	ai.FinishReasonRefusal,
	ai.FinishReasonPauseTurn,
	ai.FinishReasonUnknown,
}

// AI-13.1 — the vocabulary is closed and each value is constructible.
//
// "Closed" is proven in two directions. Every value the register names is
// constructible from another package and validates; and the zero value — the one
// a caller gets by declaring a variable and forgetting to set it — does not,
// because "nobody set this" and "the provider said something I do not
// recognise" are two facts (design.md § 3.1).
func TestFinishReason_TheVocabulary_IsClosedAndEachValueIsConstructible(t *testing.T) {
	t.Parallel()

	t.Run("the zero value names no finish reason", func(t *testing.T) {
		t.Parallel()

		var unset ai.FinishReason

		err := unset.Validate(ai.At("finish_reason"))
		if err == nil {
			t.Fatalf("FinishReason(0).Validate() = <nil>, want a violation — the zero value must name no finish reason")
		}
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("errors.Is(err, ErrNotInVocabulary) = false, want true; err = %v", err)
		}

		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &*ai.Violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "finish_reason"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("every value of the vocabulary is constructible and valid", func(t *testing.T) {
		t.Parallel()

		for _, reason := range theVocabulary {
			if err := reason.Validate(ai.At("finish_reason")); err != nil {
				t.Errorf("FinishReason(%d).Validate() = %v, want <nil>", reason, err)
			}
			if reason.String() == "" {
				t.Errorf("FinishReason(%d).String() = %q, want a non-empty stable string form", reason, "")
			}
		}
	})

	t.Run("the seven values and their string forms are pairwise distinct", func(t *testing.T) {
		t.Parallel()

		seenValue := make(map[ai.FinishReason]bool, len(theVocabulary))
		seenName := make(map[string]bool, len(theVocabulary))
		for _, reason := range theVocabulary {
			if seenValue[reason] {
				t.Errorf("FinishReason(%d) appears twice in the vocabulary", reason)
			}
			seenValue[reason] = true

			name := reason.String()
			if seenName[name] {
				t.Errorf("string form %q is shared by two values of the vocabulary", name)
			}
			seenName[name] = true
		}
		if len(seenValue) != 7 {
			t.Errorf("the vocabulary holds %d distinct values, want 7", len(seenValue))
		}
	})
}

// AI-13.1 — refusal and the content filter are two events, not one.
//
// The bare inequality of two constants is nearly vacuous, so the assertion that
// carries weight is the one about provider strings: nothing a provider says
// about its own filter may land on the model's decision to decline, and nothing
// a model says about declining may land on the filter. Collapsing them produces
// a harness that retries refusals forever and one that reports a provider policy
// intervention as the model's opinion. The line itself is documented on the two
// declarations (design.md § 4.4).
func TestFinishReason_RefusalAndContentFilter_AreDistinctValues(t *testing.T) {
	t.Parallel()

	t.Run("the two values are distinct", func(t *testing.T) {
		t.Parallel()

		if ai.FinishReasonRefusal == ai.FinishReasonContentFilter {
			t.Errorf("FinishReasonRefusal == FinishReasonContentFilter, want two values")
		}
		if ai.FinishReasonRefusal.String() == ai.FinishReasonContentFilter.String() {
			t.Errorf("both values render as %q, want two string forms", ai.FinishReasonRefusal.String())
		}
	})

	t.Run("the refusal family", func(t *testing.T) {
		t.Parallel()

		for _, providerStopValue := range []string{"refusal", "refused"} {
			if got := ai.NormalizeFinishReason(providerStopValue); got != ai.FinishReasonRefusal {
				t.Errorf("NormalizeFinishReason(%q) = %v, want refusal", providerStopValue, got)
			}
		}
	})

	t.Run("the content-filter family", func(t *testing.T) {
		t.Parallel()

		for _, providerStopValue := range []string{"content_filter", "safety", "recitation", "prohibited_content"} {
			if got := ai.NormalizeFinishReason(providerStopValue); got != ai.FinishReasonContentFilter {
				t.Errorf("NormalizeFinishReason(%q) = %v, want content_filter", providerStopValue, got)
			}
		}
	})
}

// AI-13.1 — a provider stop value normalizes into the vocabulary after trimming
// and lowering.
//
// The casing cases are not padding. One candidate vendor reports upper snake
// case, another lower snake case, and a table that matched only one of them
// would send every stop condition of the other vendor to unknown — the collapse
// this milestone exists to prevent, arriving through the back door.
func TestNormalizeFinishReason_ProviderStrings_MapIntoTheVocabulary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		providerStopValue string
		want              ai.FinishReason
	}{
		// The natural-stop family.
		{"stop", ai.FinishReasonStop},
		{"end_turn", ai.FinishReasonStop},
		{"stop_sequence", ai.FinishReasonStop},
		{"complete", ai.FinishReasonStop},

		// The length family.
		{"length", ai.FinishReasonLength},
		{"max_tokens", ai.FinishReasonLength},
		{"max_output_tokens", ai.FinishReasonLength},

		// The tool-call family.
		{"tool_calls", ai.FinishReasonToolCalls},
		{"tool_use", ai.FinishReasonToolCalls},
		{"function_call", ai.FinishReasonToolCalls},

		// The content-filter and refusal families, already landed by item 2 and
		// re-asserted here so that widening the table cannot narrow them.
		{"content_filter", ai.FinishReasonContentFilter},
		{"refusal", ai.FinishReasonRefusal},

		// The pause family.
		{"pause_turn", ai.FinishReasonPauseTurn},
		{"pause", ai.FinishReasonPauseTurn},

		// The unknown family. A provider that says "other" has told us
		// something: that its own vocabulary has a hole in it too.
		{"unknown", ai.FinishReasonUnknown},
		{"other", ai.FinishReasonUnknown},

		// Trimming and lowering, on values from three different families.
		{"  stop  ", ai.FinishReasonStop},
		{"STOP", ai.FinishReasonStop},
		{"MAX_TOKENS", ai.FinishReasonLength},
		{"\tSAFETY\n", ai.FinishReasonContentFilter},
		{" Tool_Use ", ai.FinishReasonToolCalls},
	}

	for _, tc := range cases {
		if got := ai.NormalizeFinishReason(tc.providerStopValue); got != tc.want {
			t.Errorf("NormalizeFinishReason(%q) = %v, want %v", tc.providerStopValue, got, tc.want)
		}
	}

	t.Run("every string form of the vocabulary round-trips", func(t *testing.T) {
		t.Parallel()

		for _, reason := range theVocabulary {
			if got := ai.NormalizeFinishReason(reason.String()); got != reason {
				t.Errorf("NormalizeFinishReason(%q) = %v, want %v", reason.String(), got, reason)
			}
		}
	})
}

// AI-13.1 — an unrecognized provider stop value maps to unknown, without error.
//
// This is the normalizer-crash bug class, pinned from the first day. Vendors add
// stop values without notice, so "the string I was given is not in my table" is
// the normal operating condition of this function over a long enough period, not
// an exceptional one. The cases below are the shapes that break a normalizer
// written by someone who assumed otherwise: a value invented after this table, a
// missing field arriving as the empty string, whitespace, an enormous string,
// control bytes, and near-misses of real spellings.
func TestNormalizeFinishReason_UnrecognizedString_MapsToUnknownWithoutError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name              string
		providerStopValue string
	}{
		{"a vendor value added after this table", "model_context_window_exceeded"},
		{"the empty string", ""},
		{"whitespace only", "   \t\n  "},
		{"an enormous value", strings.Repeat("x", 4096)},
		{"control bytes", "\x00\x01\x02"},
		{"a near-miss of a real spelling", "end-turn"},
		{"a real spelling with a suffix", "stop_reason"},
		{"the placeholder for a value outside the vocabulary", "invalid"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recovered := recover(); recovered != nil {
					t.Fatalf("NormalizeFinishReason(%.32q) panicked: %v", tc.providerStopValue, recovered)
				}
			}()

			if got := ai.NormalizeFinishReason(tc.providerStopValue); got != ai.FinishReasonUnknown {
				t.Errorf("NormalizeFinishReason(%.32q) = %v, want unknown", tc.providerStopValue, got)
			}
		})
	}
}

// AI-13.1 *(pin)* — a value added to the vocabulary without a string form and a
// normalization entry fails here.
//
// Exempt from red-first by doc 0002's leaf anatomy, and fully mechanical. It is
// recorded biting against a deliberate scratch constant in tasks.md.
//
// The mechanism is the language rather than a linter, because the module carries
// zero requires and may not add one until AI-24. The underlying integer is small
// enough to walk exhaustively, so the test discovers the vocabulary the way a
// consumer would — by asking the package which values it accepts — and compares
// that against the list this file names by hand. A list obtained from the
// package would agree with itself no matter what was appended to it.
//
// One omission therefore fails three assertions: the discovered set grows past
// the named set, the new value renders as the placeholder, and its string form
// does not round-trip.
func TestFinishReason_AddingAValue_FailsWithoutATableAndAStringForm(t *testing.T) {
	t.Parallel()

	// The placeholder is obtained from a value known to be outside the
	// vocabulary rather than written out, so this test does not have to be
	// updated if the spelling of the placeholder ever changes.
	placeholder := ai.FinishReason(0).String()

	named := make(map[ai.FinishReason]bool, len(theVocabulary))
	for _, reason := range theVocabulary {
		named[reason] = true
	}

	var discovered []ai.FinishReason
	for candidate := 0; candidate <= 255; candidate++ {
		reason := ai.FinishReason(candidate)
		if reason.Validate() != nil {
			continue
		}
		discovered = append(discovered, reason)

		if !named[reason] {
			t.Errorf("the package validates FinishReason(%d), which this test does not name — "+
				"add it to theVocabulary and give it an obligation", candidate)
		}
		if got := reason.String(); got == placeholder {
			t.Errorf("FinishReason(%d) validates but renders as the placeholder %q — "+
				"add it to finishReasonNames", candidate, got)
		}
		if got := ai.NormalizeFinishReason(reason.String()); got != reason {
			t.Errorf("FinishReason(%d): NormalizeFinishReason(%q) = %v, want the value back — "+
				"add it to finishReasonBySpelling", candidate, reason.String(), got)
		}
	}

	if len(discovered) != len(theVocabulary) {
		t.Errorf("the package validates %d values, want exactly the %d named in this test",
			len(discovered), len(theVocabulary))
	}
	for _, reason := range theVocabulary {
		if reason.Validate() != nil {
			t.Errorf("this test names FinishReason(%d), which the package does not validate", reason)
		}
	}
}
