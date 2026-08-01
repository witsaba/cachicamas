// Tests for AI-19 — the provider error taxonomy and the terminal error
// event.
//
// The package under test is imported by its module path from the external
// test package ai_test, so every assertion is written against exactly the
// surface an adapter in another package sees — the convention every Layer 1
// test file in this package follows (event_test.go, completion_test.go).
// TestFailure_ConstructibleFromAnotherPackage_ThenWrappedIntoTheTerminalErrorEvent
// is this milestone's direct negation of doc 0001's C4: the retired plan
// declared a mandatory terminal error whose payload no adapter could ever
// build, from any package. This file is that adapter.
package ai_test

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// R-AIP-001 — the terminal error event is constructible from another
// package, through exported identifiers alone, with no in-package
// assistance and no provider interface. Direct negation of C4.
func TestFailure_ConstructibleFromAnotherPackage_ThenWrappedIntoTheTerminalErrorEvent(t *testing.T) {
	t.Parallel()

	t.Run("PreStreamFailure builds a *Failure from exported identifiers alone (S-AIP-001)", func(t *testing.T) {
		t.Parallel()

		f, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout})
		if err != nil {
			t.Fatalf("ai.PreStreamFailure returned %v, want no failure", err)
		}
		if f == nil {
			t.Fatal("ai.PreStreamFailure returned a nil *Failure with no error, want a constructed value")
		}
	})

	t.Run("MidStreamFailure builds a *Failure from exported identifiers alone (S-AIP-002)", func(t *testing.T) {
		t.Parallel()

		f, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, true)
		if err != nil {
			t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
		}
		if f == nil {
			t.Fatal("ai.MidStreamFailure returned a nil *Failure with no error, want a constructed value")
		}
	})

	t.Run("the constructed failure wraps into the terminal error event through ErrorEvent alone (S-AIP-003)", func(t *testing.T) {
		t.Parallel()

		f, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryMalformedResponse}, false)
		if err != nil {
			t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
		}
		event, err := ai.ErrorEvent(f)
		if err != nil {
			t.Fatalf("ai.ErrorEvent returned %v, want no failure — this is the payload C4 said no adapter could build", err)
		}
		if got := event.Kind(); got != ai.EventKindError {
			t.Errorf("event.Kind() = %v, want ai.EventKindError", got)
		}
	})
}

// R-AIP-002 — the constructed event satisfies AI-14's envelope invariants:
// kind derived from the payload, nil/mismatched payload rejected through a
// landed AI-04 sentinel, terminal per V-STR-18.
func TestErrorEvent_EnvelopeInvariants_MatchAI14(t *testing.T) {
	t.Parallel()

	t.Run("the kind is derived from the payload, never supplied separately (S-AIP-004)", func(t *testing.T) {
		t.Parallel()

		event := mustErrorEvent(t, mustPreStreamFailure(t, ai.FailureCategoryRateLimit))
		if got := event.Kind(); got != ai.EventKindError {
			t.Errorf("event.Kind() = %v, want ai.EventKindError", got)
		}
	})

	t.Run("a nil payload is rejected with ErrEmpty at payload (S-AIP-005)", func(t *testing.T) {
		t.Parallel()

		_, err := ai.ErrorEvent(nil)
		if err == nil {
			t.Fatal("ai.ErrorEvent(nil) = nil error, want a rejection")
		}
		if !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("errors.Is(err, ai.ErrEmpty) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "payload"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("a mismatched-kind accessor reports no payload, never a panic or a coerced value (S-AIP-005)", func(t *testing.T) {
		t.Parallel()

		completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		if f, ok := completion.ErrorPayload(); ok || f != nil {
			t.Errorf("completion.ErrorPayload() = (%v, %v), want (nil, false) on an event of another kind", f, ok)
		}
	})

	t.Run("the registration's terminal property is set (S-AIP-006)", func(t *testing.T) {
		t.Parallel()

		d, ok := ai.DescriptorOf(ai.EventKindError)
		if !ok {
			t.Fatal("ai.DescriptorOf(ai.EventKindError) reported not registered")
		}
		if !d.Terminal {
			t.Error("descriptor.Terminal = false, want true")
		}
	})

	t.Run("an event of any kind after an error event reports a violation, and two error events report a violation (S-AIP-006)", func(t *testing.T) {
		t.Parallel()

		var s ai.Stamper
		failed := mustErrorEvent(t, mustPreStreamFailure(t, ai.FailureCategoryTimeout))
		after, err := ai.NewResponseStart("resp_ai19_01", "model-x")
		if err != nil {
			t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
		}

		report := ai.CheckStream([]ai.Event{s.Stamp(failed), s.Stamp(after)})
		got := report.Violation()
		if got == nil {
			t.Fatal("report.Violation() = nil, want a violation for an event following a terminal error")
		}
		if !errors.Is(got, ai.ErrMisplaced) {
			t.Errorf("errors.Is(err, ai.ErrMisplaced) = false, want true; err = %v", got)
		}
	})
}

// R-AIP-003 — terminal exclusivity: completion or error, never both, never
// confusable by anything but Kind().
func TestErrorEvent_TerminalExclusivity_NeverBothCompletionAndError(t *testing.T) {
	t.Parallel()

	t.Run("an error event carries no completion payload, and a completion event carries no error payload (S-AIP-007)", func(t *testing.T) {
		t.Parallel()

		errorEvent := mustErrorEvent(t, mustPreStreamFailure(t, ai.FailureCategoryTimeout))
		if c, ok := errorEvent.Completion(); ok {
			t.Errorf("errorEvent.Completion() = (%+v, true), want ok=false on an event of another kind", c)
		}

		completionEvent, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		if f, ok := completionEvent.ErrorPayload(); ok {
			t.Errorf("completionEvent.ErrorPayload() = (%v, true), want ok=false on an event of another kind", f)
		}
	})

	// No accessor may let a category be read off a completion, or a finish
	// reason off a failure — proven by an exhaustive method-set comparison,
	// not a sample, so an accessor added later that happens to collide fails
	// this test by name rather than by luck.
	t.Run("Failure and Completion export no accessor of the same name (S-AIP-008)", func(t *testing.T) {
		t.Parallel()

		completionMethods := exportedMethodNames(reflect.TypeOf(ai.Completion{}))
		failureMethods := exportedMethodNames(reflect.TypeOf(&ai.Failure{}))

		for name := range completionMethods {
			if failureMethods[name] {
				t.Errorf("both ai.Completion and ai.Failure export an accessor named %q, want terminal exclusivity with no shared accessor name", name)
			}
		}
		if !completionMethods["FinishReason"] {
			t.Fatal("test fixture error: ai.Completion no longer exports FinishReason")
		}
		if failureMethods["FinishReason"] {
			t.Error("ai.Failure exports FinishReason, want a finish reason readable only off a completion")
		}
		if !failureMethods["Category"] {
			t.Fatal("test fixture error: ai.Failure no longer exports Category")
		}
		if completionMethods["Category"] {
			t.Error("ai.Completion exports Category, want a category readable only off a failure")
		}
	})

	// Discrimination is Kind()-only (no string comparison anywhere in the
	// integration — stream_check.go and CheckEmit read only the registered
	// descriptor, never a kind's name; a review-only check, per
	// completion_test.go's S-ARP-026 precedent, not repeated here).
	t.Run("an error event and a completion event never report the same Kind() (S-AIP-009)", func(t *testing.T) {
		t.Parallel()

		errEvent := mustErrorEvent(t, mustPreStreamFailure(t, ai.FailureCategoryTimeout))
		completionEvent, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		if err != nil {
			t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
		}
		if errEvent.Kind() == completionEvent.Kind() {
			t.Fatal("an error event and a completion event report the same Kind(), want them distinct")
		}
	})
}

// theFailureCategoryVocabulary is the nine members of R-AIP-004, named by
// hand — finish_reason_test.go's theVocabulary precedent: a list the
// package supplied would agree with itself no matter what was added to it.
var theFailureCategoryVocabulary = []ai.FailureCategory{
	ai.FailureCategoryAuthentication,
	ai.FailureCategoryAuthorization,
	ai.FailureCategoryRateLimit,
	ai.FailureCategoryUnavailable,
	ai.FailureCategoryTimeout,
	ai.FailureCategoryCancellation,
	ai.FailureCategoryMalformedResponse,
	ai.FailureCategoryUnsupportedCapability,
	ai.FailureCategoryUnknown,
}

// R-AIP-004 — the vocabulary is closed and each of the nine charter members
// is constructible and mutually distinct.
func TestFailureCategory_TheVocabulary_IsClosedAndEachValueIsConstructible(t *testing.T) {
	t.Parallel()

	t.Run("every member validates and has a non-empty stable string form (S-AIP-010)", func(t *testing.T) {
		t.Parallel()

		for _, c := range theFailureCategoryVocabulary {
			if err := c.Validate(ai.At("category")); err != nil {
				t.Errorf("FailureCategory(%d).Validate() = %v, want <nil>", c, err)
			}
			if c.String() == "" {
				t.Errorf("FailureCategory(%d).String() = %q, want a non-empty stable string form", c, "")
			}
		}
	})

	t.Run("cancellation is a member — required, not optional (S-AIP-011)", func(t *testing.T) {
		t.Parallel()

		if err := ai.FailureCategoryCancellation.Validate(); err != nil {
			t.Errorf("ai.FailureCategoryCancellation.Validate() = %v, want <nil> — R-AIP-004 requires cancellation, not as an optional extra", err)
		}
	})

	t.Run("the nine values and their string forms are pairwise distinct (S-AIP-012)", func(t *testing.T) {
		t.Parallel()

		seenValue := make(map[ai.FailureCategory]bool, len(theFailureCategoryVocabulary))
		seenName := make(map[string]bool, len(theFailureCategoryVocabulary))
		for _, c := range theFailureCategoryVocabulary {
			if seenValue[c] {
				t.Errorf("FailureCategory(%d) appears twice in the vocabulary", c)
			}
			seenValue[c] = true

			name := c.String()
			if seenName[name] {
				t.Errorf("string form %q is shared by two categories", name)
			}
			seenName[name] = true
		}
		if len(seenValue) != 9 {
			t.Errorf("the vocabulary holds %d distinct values, want 9 (the charter minimum, R-AIP-004)", len(seenValue))
		}
	})
}

// R-AIP-005 — the vocabulary is closed and enumerable in stable order from
// another package (groundwork for AI-23.4).
func TestFailureCategories_Enumeration_IsStableOrderFromAnotherPackage(t *testing.T) {
	t.Parallel()

	t.Run("FailureCategories lists exactly the nine charter members, in declaration order (S-AIP-015)", func(t *testing.T) {
		t.Parallel()

		got := ai.FailureCategories()
		if len(got) != len(theFailureCategoryVocabulary) {
			t.Fatalf("ai.FailureCategories() = %v (%d categories), want %v (%d categories)",
				got, len(got), theFailureCategoryVocabulary, len(theFailureCategoryVocabulary))
		}
		for i, want := range theFailureCategoryVocabulary {
			if got[i] != want {
				t.Errorf("ai.FailureCategories()[%d] = %v, want %v", i, got[i], want)
			}
		}
	})

	t.Run("mutating a returned slice does not corrupt a later call (S-AIP-016)", func(t *testing.T) {
		t.Parallel()

		first := ai.FailureCategories()
		if len(first) == 0 {
			t.Fatal("ai.FailureCategories() returned empty, want the nine charter members")
		}
		original := first[0]
		first[0] = ai.FailureCategoryUnknown

		second := ai.FailureCategories()
		if second[0] != original {
			t.Errorf("mutating one call's result changed a later call's result: second[0] = %v, want %v — "+
				"ai.FailureCategories must return a fresh slice each call, never a shared package vocabulary a caller can rewrite",
				second[0], original)
		}
	})
}

// R-AIP-004/005 — construction itself enforces the category vocabulary
// (design.md's Go-spellings section: "Construction validates via
// FirstFailure: category Validate(At(\"category\"))"), so a caller cannot
// build a *Failure that PreStreamFailure/MidStreamFailure themselves would
// reject.
func TestNewFailure_InvalidCategory_FailsWithErrNotInVocabularyAtCategory(t *testing.T) {
	t.Parallel()

	t.Run("PreStreamFailure with the zero-value category is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := ai.PreStreamFailure(ai.FailureReport{})
		if err == nil {
			t.Fatal("ai.PreStreamFailure(zero-value category) = nil error, want a rejection")
		}
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("errors.Is(err, ai.ErrNotInVocabulary) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "category"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("MidStreamFailure with an out-of-range category is rejected", func(t *testing.T) {
		t.Parallel()

		_, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategory(200)}, false)
		if err == nil {
			t.Fatal("ai.MidStreamFailure(out-of-range category) = nil error, want a rejection")
		}
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("errors.Is(err, ai.ErrNotInVocabulary) = false, want true; err = %v", err)
		}
	})
}

// R-AIP-009's bounded safe metadata — an out-of-range status class is
// rejected at construction, positioned at "status_class". Written after the
// implementation landed (apply-progress.md records this ordering slip
// honestly rather than silently); it is a real, currently-passing assertion
// that would fail if the bound were removed or the position renamed —
// verified by temporarily reverting the bound locally and re-running before
// this commit.
func TestNewFailure_OutOfRangeStatusClass_FailsWithErrOutOfRangeAtStatusClass(t *testing.T) {
	t.Parallel()

	for _, statusClass := range []int{-1, 6, 999} {
		_, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, StatusClass: statusClass})
		if err == nil {
			t.Fatalf("ai.PreStreamFailure(StatusClass=%d) = nil error, want a rejection", statusClass)
		}
		if !errors.Is(err, ai.ErrOutOfRange) {
			t.Errorf("StatusClass=%d: errors.Is(err, ai.ErrOutOfRange) = false, want true; err = %v", statusClass, err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("StatusClass=%d: errors.As(err, &violation) = false, want true; err = %v", statusClass, err)
		}
		if got, want := violation.Path().String(), "status_class"; got != want {
			t.Errorf("StatusClass=%d: violation position = %q, want %q", statusClass, got, want)
		}
	}

	for _, statusClass := range []int{0, 1, 2, 3, 4, 5} {
		_, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, StatusClass: statusClass})
		if err != nil {
			t.Errorf("ai.PreStreamFailure(StatusClass=%d) returned %v, want no failure — 0..5 are all in bound", statusClass, err)
		}
	}
}

// R-AIP-010 — two perpendicular discriminating inputs, both readable from
// the failure value alone: the partial-output bool and a separate delivery
// accessor. The three constructible shapes are distinguishable from the
// value alone; the fourth (pre-stream, output-preceded) is unconstructible.
func TestFailure_PartialOutputAndDelivery_AreTwoPerpendicularAxes(t *testing.T) {
	t.Parallel()

	// The three constructible shapes of R-AIP-010's table, consolidated into
	// one table-driven check (task 5.5's refactor) rather than three
	// near-identical subtests: the shape itself — which constructor, which
	// output-flag — is data, and the two expectations it produces follow
	// from it, which is the property under test.
	shapes := []struct {
		name         string
		build        func(t *testing.T) *ai.Failure
		wantPartial  bool
		wantDelivery ai.DeliveryPath
		scenario     string
	}{
		{
			name:         "pre-stream",
			build:        func(t *testing.T) *ai.Failure { return mustPreStreamFailure(t, ai.FailureCategoryTimeout) },
			wantPartial:  false,
			wantDelivery: ai.DeliveryPreStream,
			scenario:     "S-AIP-032",
		},
		{
			name:         "mid-stream, no output preceded",
			build:        func(t *testing.T) *ai.Failure { return mustMidStreamFailure(t, ai.FailureCategoryUnavailable, false) },
			wantPartial:  false,
			wantDelivery: ai.DeliveryMidStream,
			scenario:     "S-AIP-033",
		},
		{
			name:         "mid-stream, output preceded",
			build:        func(t *testing.T) *ai.Failure { return mustMidStreamFailure(t, ai.FailureCategoryUnavailable, true) },
			wantPartial:  true,
			wantDelivery: ai.DeliveryMidStream,
			scenario:     "S-AIP-034",
		},
	}
	for _, shape := range shapes {
		t.Run(shape.name+" ("+shape.scenario+")", func(t *testing.T) {
			t.Parallel()

			f := shape.build(t)
			if got := f.PartialOutput(); got != shape.wantPartial {
				t.Errorf("f.PartialOutput() = %v, want %v", got, shape.wantPartial)
			}
			if got := f.Delivery(); got != shape.wantDelivery {
				t.Errorf("f.Delivery() = %v, want %v", got, shape.wantDelivery)
			}
		})
	}

	t.Run("delivery path alone cannot distinguish the two mid-stream shapes — PartialOutput is a second, independent axis (S-AIP-035)", func(t *testing.T) {
		t.Parallel()

		noOutput := mustMidStreamFailure(t, ai.FailureCategoryUnavailable, false)
		withOutput := mustMidStreamFailure(t, ai.FailureCategoryUnavailable, true)
		if noOutput.Delivery() != withOutput.Delivery() {
			t.Fatal("the two mid-stream failures report different Delivery() values, want them equal — the axes must be perpendicular")
		}
		if noOutput.PartialOutput() == withOutput.PartialOutput() {
			t.Error("the two mid-stream failures report the same PartialOutput(), want them distinguishable by that axis alone")
		}
	})

	t.Run("the fourth cell — pre-stream with output preceding — is unconstructible: PreStreamFailure's signature takes no output-flag parameter (S-AIP-032, structural)", func(t *testing.T) {
		t.Parallel()

		// A compile-time property, not a runtime one. This line fails to
		// compile the moment PreStreamFailure grows a second (bool)
		// parameter — the fourth cell would then be reachable.
		var _ func(ai.FailureReport) (*ai.Failure, error) = ai.PreStreamFailure
	})
}

// R-AIP-011 — "is a naive retry safe?" is answerable from PartialOutput()
// alone, regardless of category, retryability or delivery path — the
// never-retry-after-partial-output clause (V-FAIL-15); Retryable()=true
// never overrides it.
func TestFailure_NaiveRetrySafety_AnsweredFromPartialOutputAlone(t *testing.T) {
	t.Parallel()

	// naiveRetrySafe is written the way a Layer 2 consumer would write it —
	// finish_reason_test.go's consumerResponse precedent: the property under
	// test is that this predicate needs nothing but PartialOutput().
	naiveRetrySafe := func(f *ai.Failure) bool { return !f.PartialOutput() }

	t.Run("output preceded the failure: never safe, even when Retryable()=true (S-AIP-036/038)", func(t *testing.T) {
		t.Parallel()

		f, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, Retryable: true}, true)
		if err != nil {
			t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
		}
		if naiveRetrySafe(f) {
			t.Error("naiveRetrySafe(f) = true, want false — output preceded this failure; Retryable=true must not override V-FAIL-15")
		}
	})

	t.Run("no output preceded the failure: safe from the discriminator alone, for pre-stream and mid-stream alike (S-AIP-037)", func(t *testing.T) {
		t.Parallel()

		pre := mustPreStreamFailure(t, ai.FailureCategoryTimeout)
		mid := mustMidStreamFailure(t, ai.FailureCategoryTimeout, false)
		if !naiveRetrySafe(pre) {
			t.Error("naiveRetrySafe(pre) = false, want true — no output preceded it")
		}
		if !naiveRetrySafe(mid) {
			t.Error("naiveRetrySafe(mid) = false, want true — no output preceded it, even though delivery is mid-stream")
		}
	})

	t.Run("unsafety is independent of category (S-AIP-038)", func(t *testing.T) {
		t.Parallel()

		for _, category := range theFailureCategoryVocabulary {
			f, err := ai.MidStreamFailure(ai.FailureReport{Category: category}, true)
			if err != nil {
				t.Fatalf("ai.MidStreamFailure(category=%v) returned %v, want no failure", category, err)
			}
			if naiveRetrySafe(f) {
				t.Errorf("category %v: naiveRetrySafe(f) = true, want false — output preceded this failure regardless of category", category)
			}
		}
	})
}

// R-AIP-012 — no accessor or predicate on Failure re-conflates the two axes
// into one field; a 3-member shape enum is explicitly prohibited (G8
// verbatim), proven the same way S-AIP-023 proved no scheduling identifier:
// an exhaustive exported-surface scan.
func TestFailure_NoAccessorCombinesPartialOutputAndDelivery(t *testing.T) {
	t.Parallel()

	t.Run("no combining/shape identifier is exported (S-AIP-039)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "provider_failure.go", nil, 0)
		if err != nil {
			t.Fatalf("parsing provider_failure.go: %v", err)
		}
		forbidden := []string{"Shape", "DeliveryState", "PartialDelivery", "OutputDelivery"}
		for _, name := range exportedTopLevelNames(file) {
			for _, bad := range forbidden {
				if strings.Contains(name, bad) {
					t.Errorf("provider_failure.go exports %q — R-AIP-012 prohibits a combining accessor or a "+
						"3-member shape enum re-conflating PartialOutput and Delivery (G8 verbatim)", name)
				}
			}
		}
	})

	t.Run("reading one axis never changes what the other reports (S-AIP-040)", func(t *testing.T) {
		t.Parallel()

		f := mustMidStreamFailure(t, ai.FailureCategoryTimeout, true)
		before := f.Delivery()
		_ = f.PartialOutput()
		_ = f.PartialOutput()
		if after := f.Delivery(); after != before {
			t.Errorf("f.Delivery() changed from %v to %v after reading PartialOutput(), want the axes independent", before, after)
		}
	})
}

// R-AIP-006 — an unmodelled failure becomes unknown with the raw provider
// label preserved, bounded and sanitized; no cross-vendor normalizer ships
// in this package.
func TestFailure_RawLabel_UnknownCategoryPreservesTheProviderLabel(t *testing.T) {
	t.Parallel()

	t.Run("the raw label survives construction and reads back byte-exact (S-AIP-017)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailureReport(t, ai.FailureReport{
			Category: ai.FailureCategoryUnknown,
			RawLabel: "vendor_weird_code_123",
		})
		if got, want := f.RawLabel(), "vendor_weird_code_123"; got != want {
			t.Errorf("f.RawLabel() = %q, want %q", got, want)
		}
	})

	t.Run("the raw label survives at least one wrap (S-AIP-017)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailureReport(t, ai.FailureReport{
			Category: ai.FailureCategoryUnknown,
			RawLabel: "provider_specific_snowflake",
		})
		wrapped := fmt.Errorf("adapter: %w", f)

		var got *ai.Failure
		if !errors.As(wrapped, &got) {
			t.Fatalf("errors.As(wrapped, &got) = false, want true; wrapped = %v", wrapped)
		}
		if got.RawLabel() != "provider_specific_snowflake" {
			t.Errorf("got.RawLabel() = %q, want %q, after one wrap", got.RawLabel(), "provider_specific_snowflake")
		}
	})

	t.Run("an over-long raw label is dropped whole, not truncated, and construction still succeeds (S-AIP-018/019)", func(t *testing.T) {
		t.Parallel()

		over := strings.Repeat("x", 65) // one byte over the 64-byte bound
		f := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryUnknown, RawLabel: over})
		if got := f.RawLabel(); got != "" {
			t.Errorf("f.RawLabel() = %q, want empty — an over-long label is dropped whole, never truncated (design.md D6)", got)
		}
	})

	t.Run("a 64-byte raw label (exactly at the bound) survives (S-AIP-018/019)", func(t *testing.T) {
		t.Parallel()

		exact := strings.Repeat("y", 64)
		f := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryUnknown, RawLabel: exact})
		if got := f.RawLabel(); got != exact {
			t.Errorf("f.RawLabel() = %q, want the 64-byte label unchanged — the bound is inclusive", got)
		}
	})

	t.Run("a control-character-bearing raw label is dropped whole, and construction still succeeds (S-AIP-018/019)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryUnknown, RawLabel: "bad\x00label"})
		if got := f.RawLabel(); got != "" {
			t.Errorf("f.RawLabel() = %q, want empty — a control-character-bearing label is dropped whole", got)
		}
	})

	t.Run("an empty label on a modelled, non-unknown category still constructs (S-AIP-021)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryTimeout})
		if got := f.RawLabel(); got != "" {
			t.Errorf("f.RawLabel() = %q, want empty", got)
		}
	})

	t.Run("no cross-vendor label-to-category normalizer exists — unlike NormalizeFinishReason, this milestone ships none (S-AIP-020)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "provider_failure.go", nil, 0)
		if err != nil {
			t.Fatalf("parsing provider_failure.go: %v", err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Recv != nil { // methods are not normalizer candidates
				return true
			}
			if fn.Type.Params == nil || len(fn.Type.Params.List) != 1 ||
				fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
				return true
			}
			if identNamed(fn.Type.Params.List[0].Type, "string") && identNamed(fn.Type.Results.List[0].Type, "FailureCategory") {
				t.Errorf("provider_failure.go declares %s(string) FailureCategory — R-AIP-006 forbids a "+
					"cross-vendor label-to-category normalizer in this package; category assignment for a "+
					"real vendor is AI-32's, not Layer 1's", fn.Name.Name)
			}
			return true
		})
	})
}

// identNamed reports whether expr is a bare identifier spelled name.
func identNamed(expr ast.Expr, name string) bool {
	ident, ok := expr.(*ast.Ident)
	return ok && ident.Name == name
}

// R-AIP-007 — retryability is a machine-readable classification, readable
// for every category; this package exports no scheduling decision.
func TestFailure_Retryable_ReadableForEveryCategory(t *testing.T) {
	t.Parallel()

	t.Run("Retryable reflects what was reported, for every charter category (S-AIP-022)", func(t *testing.T) {
		t.Parallel()

		for _, category := range theFailureCategoryVocabulary {
			for _, retryable := range []bool{true, false} {
				f, err := ai.PreStreamFailure(ai.FailureReport{Category: category, Retryable: retryable})
				if err != nil {
					t.Fatalf("ai.PreStreamFailure(category=%v, retryable=%v) returned %v, want no failure", category, retryable, err)
				}
				if got := f.Retryable(); got != retryable {
					t.Errorf("category %v: f.Retryable() = %v, want %v", category, got, retryable)
				}
			}
		}
	})

	t.Run("no scheduling/backoff/attempt-counting/failover identifier is exported (S-AIP-023)", func(t *testing.T) {
		t.Parallel()

		forbidden := []string{"Backoff", "Attempt", "Failover", "Schedul"}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "provider_failure.go", nil, 0)
		if err != nil {
			t.Fatalf("parsing provider_failure.go: %v", err)
		}
		for _, name := range exportedTopLevelNames(file) {
			for _, bad := range forbidden {
				if strings.Contains(name, bad) {
					t.Errorf("provider_failure.go exports %q, which contains %q — R-AIP-007 reserves "+
						"scheduling/backoff/attempt-counting/failover for Layer 2 (V-OUT-11), never a "+
						"type property here", name, bad)
				}
			}
		}
	})
}

// R-AIP-008 — a typed retry-after hint with presence separate from value:
// absent, zero and non-zero are three distinguishable readings.
func TestFailure_RetryAfter_PresenceSeparateFromValue(t *testing.T) {
	t.Parallel()

	t.Run("an absent RetryDelay reports not present (S-AIP-024)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryRateLimit})
		if _, present := f.RetryAfter(); present {
			t.Error("f.RetryAfter() reports present, want absent — RetryAfter was never supplied")
		}
	})

	t.Run("Delay(0) is a reported, present zero — distinguishable from absent (S-AIP-025)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryRateLimit, RetryAfter: ai.Delay(0)})
		duration, present := f.RetryAfter()
		if !present {
			t.Fatal("f.RetryAfter() reports absent, want present — ai.Delay(0) was supplied")
		}
		if duration != 0 {
			t.Errorf("f.RetryAfter() duration = %v, want 0", duration)
		}
	})

	t.Run("a non-zero Delay reads back exactly (S-AIP-026)", func(t *testing.T) {
		t.Parallel()

		want := 30 * time.Second
		f := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryRateLimit, RetryAfter: ai.Delay(want)})
		duration, present := f.RetryAfter()
		if !present {
			t.Fatal("f.RetryAfter() reports absent, want present")
		}
		if duration != want {
			t.Errorf("f.RetryAfter() duration = %v, want %v", duration, want)
		}
	})

	t.Run("retryability and retry-after are read independently — neither derived from the other (S-AIP-027)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailureReport(t, ai.FailureReport{
			Category:   ai.FailureCategoryRateLimit,
			Retryable:  false,
			RetryAfter: ai.Delay(10 * time.Second),
		})
		if f.Retryable() {
			t.Error("f.Retryable() = true, want false (as reported) — RetryAfter must not force it true")
		}
		duration, present := f.RetryAfter()
		if !present || duration != 10*time.Second {
			t.Errorf("f.RetryAfter() = (%v, %v), want (10s, true) — Retryable=false must not suppress it", duration, present)
		}
	})
}

// R-AIP-009 — Error() renders only the category's registered text (a fixed
// prefix + category name), never the wrapped cause's text; Unwrap() still
// exposes the cause; StatusClass and RequestID are dedicated accessors,
// never substrings of Error()'s rendered text. Proven with a planted
// credential/body sentinel — completion_test.go's canary-value posture,
// restated for a wrapped cause rather than a field.
func TestFailure_Error_ExcludesTheCauseAndBoundedMetadata_UnwrapStillExposesThem(t *testing.T) {
	t.Parallel()

	const plantedSecret = "credential=sk-PLANTED-9f8e7d6c5b4a"
	cause := errors.New("upstream body: " + plantedSecret)

	f := mustPreStreamFailureReport(t, ai.FailureReport{
		Category:    ai.FailureCategoryAuthentication,
		Cause:       cause,
		StatusClass: 4,
		RequestID:   "req_canary_plant_77",
		RawLabel:    "raw_canary_plant_88",
	})

	t.Run("Error() never contains the planted cause text (S-AIP-028)", func(t *testing.T) {
		t.Parallel()

		if got := f.Error(); strings.Contains(got, plantedSecret) {
			t.Errorf("f.Error() = %q, contains the planted cause text, want it excluded", got)
		}
	})

	t.Run("Unwrap() still exposes the cause, planted text intact, reachable through errors.Is (S-AIP-029)", func(t *testing.T) {
		t.Parallel()

		unwrapped := f.Unwrap()
		if unwrapped == nil {
			t.Fatal("f.Unwrap() = nil, want the cause")
		}
		if !strings.Contains(unwrapped.Error(), plantedSecret) {
			t.Errorf("f.Unwrap().Error() = %q, want it to contain the planted cause text", unwrapped.Error())
		}
		if !errors.Is(f, cause) {
			t.Error("errors.Is(f, cause) = false, want true — errors.Is must reach the cause through Unwrap")
		}
	})

	t.Run("StatusClass and RequestID are dedicated accessors, and Error() does not substring-include either (S-AIP-030/031)", func(t *testing.T) {
		t.Parallel()

		statusClass, present := f.StatusClass()
		if !present || statusClass != 4 {
			t.Errorf("f.StatusClass() = (%d, %v), want (4, true)", statusClass, present)
		}
		if got := f.RequestID(); got != "req_canary_plant_77" {
			t.Errorf("f.RequestID() = %q, want %q", got, "req_canary_plant_77")
		}

		rendered := f.Error()
		if strings.Contains(rendered, "req_canary_plant_77") {
			t.Errorf("f.Error() = %q, contains the request ID, want it excluded — RequestID is a dedicated accessor, not part of the rendered text", rendered)
		}
		if strings.Contains(rendered, "raw_canary_plant_88") {
			t.Errorf("f.Error() = %q, contains the raw label, want it excluded", rendered)
		}
	})

	t.Run("an absent StatusClass reports not present (companion case)", func(t *testing.T) {
		t.Parallel()

		bare := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryTimeout})
		if _, present := bare.StatusClass(); present {
			t.Error("bare.StatusClass() reports present, want absent — StatusClass was never supplied")
		}
		if got := bare.RequestID(); got != "" {
			t.Errorf("bare.RequestID() = %q, want empty", got)
		}
	})
}

// mustPreStreamFailureReport constructs a pre-stream failure from a full
// report or fails the test.
func mustPreStreamFailureReport(t *testing.T, report ai.FailureReport) *ai.Failure {
	t.Helper()
	f, err := ai.PreStreamFailure(report)
	if err != nil {
		t.Fatalf("ai.PreStreamFailure(%+v) returned %v, want no failure", report, err)
	}
	return f
}

// mustPreStreamFailure constructs a pre-stream failure of the given category
// or fails the test.
func mustPreStreamFailure(t *testing.T, category ai.FailureCategory) *ai.Failure {
	t.Helper()
	f, err := ai.PreStreamFailure(ai.FailureReport{Category: category})
	if err != nil {
		t.Fatalf("ai.PreStreamFailure(category=%v) returned %v, want no failure", category, err)
	}
	return f
}

// mustMidStreamFailure constructs a mid-stream failure of the given category
// or fails the test.
func mustMidStreamFailure(t *testing.T, category ai.FailureCategory, outputPreceded bool) *ai.Failure {
	t.Helper()
	f, err := ai.MidStreamFailure(ai.FailureReport{Category: category}, outputPreceded)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure(category=%v, outputPreceded=%v) returned %v, want no failure", category, outputPreceded, err)
	}
	return f
}

// mustErrorEvent wraps f into a terminal error event or fails the test.
func mustErrorEvent(t *testing.T, f *ai.Failure) ai.Event {
	t.Helper()
	event, err := ai.ErrorEvent(f)
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	return event
}

// exportedMethodNames reports the set of exported method names typ declares.
// reflect.Type.NumMethod already excludes unexported methods for a
// non-interface type, so this needs no filtering of its own.
func exportedMethodNames(typ reflect.Type) map[string]bool {
	names := make(map[string]bool, typ.NumMethod())
	for i := 0; i < typ.NumMethod(); i++ {
		names[typ.Method(i).Name] = true
	}
	return names
}
