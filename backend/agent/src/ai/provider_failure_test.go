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
	"io/fs"
	"path/filepath"
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

		event := mustErrorEvent(t, mustMidStreamFailure(t, ai.FailureCategoryRateLimit, false))
		if got := event.Kind(); got != ai.EventKindError {
			t.Errorf("event.Kind() = %v, want ai.EventKindError", got)
		}
	})

	t.Run("a pre-stream-delivery payload is rejected with ErrMisplaced at payload", func(t *testing.T) {
		t.Parallel()

		// A *Failure whose Delivery() is DeliveryPreStream was returned
		// directly from Stream — it never crossed a carrier. Wrapping it
		// as the stream's terminal event would produce an event whose own
		// payload reports the wrong delivery path, so the contradictory
		// combination is unconstructible here, not merely undocumented —
		// the same posture R-AIP-010's fourth cell already takes.
		pre := mustPreStreamFailure(t, ai.FailureCategoryTimeout)
		event, err := ai.ErrorEvent(pre)
		if err == nil {
			t.Fatal("ai.ErrorEvent(pre-stream failure) = nil error, want a rejection — " +
				"the wrapped event's Delivery() would contradict its own carrier")
		}
		if !errors.Is(err, ai.ErrMisplaced) {
			t.Errorf("errors.Is(err, ai.ErrMisplaced) = false, want true; err = %v", err)
		}
		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "payload"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
		if event != (ai.Event{}) {
			t.Error("ai.ErrorEvent(pre-stream failure) returned a non-zero Event alongside its rejection")
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
		failed := mustErrorEvent(t, mustMidStreamFailure(t, ai.FailureCategoryTimeout, false))
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

		errorEvent := mustErrorEvent(t, mustMidStreamFailure(t, ai.FailureCategoryTimeout, false))
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

	// No data accessor may let a category be read off a completion, or a
	// finish reason off a failure — proven by an exhaustive method-set
	// comparison, not a sample, so an accessor added later that happens to
	// collide fails this test by name rather than by luck.
	//
	// diagnosticRenderers exempts exactly the one name that both terminal
	// payloads now share (AI-41.2, R-AIP-016): GoString renders a fixed
	// diagnostic string on each — "completion", or "provider failure:
	// <category>" — never the other type's data, so sharing it does not
	// violate this scenario's own text ("... in a way that lets a consumer
	// read a category off a completion or a finish reason off a failure").
	//
	// The exemption is deliberately one name, not the three renderers of Go's
	// formatting protocol. Completion exports no Error, so exempting it would
	// be unreachable; exempting String would pre-emptively disable a real
	// signal, as a String() on *Failure reporting a finish reason would then
	// pass this guard silently. Every other exported name, present or added
	// later, is still checked with zero exemption — the data-accessor
	// assertions below are unchanged.
	t.Run("Failure and Completion export no data accessor of the same name (S-AIP-008)", func(t *testing.T) {
		t.Parallel()

		diagnosticRenderers := map[string]bool{"GoString": true}

		completionMethods := exportedMethodNames(reflect.TypeOf(ai.Completion{}))
		failureMethods := exportedMethodNames(reflect.TypeOf(&ai.Failure{}))

		for name := range completionMethods {
			if diagnosticRenderers[name] {
				continue
			}
			if failureMethods[name] {
				t.Errorf("both ai.Completion and ai.Failure export a non-diagnostic accessor named %q, want terminal exclusivity with no shared accessor name", name)
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

		errEvent := mustErrorEvent(t, mustMidStreamFailure(t, ai.FailureCategoryTimeout, false))
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
		// parameter — the fourth cell would then be reachable. The
		// explicit type is deliberate, not redundant: it pins the exact
		// signature as a regression guard, so it stays even though
		// staticcheck's QF1011 would otherwise infer it away.
		var _ func(ai.FailureReport) (*ai.Failure, error) = ai.PreStreamFailure //nolint:staticcheck // explicit type is the point: pins the exact signature (S-AIP-032)
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

// R-AIP-013 — the same concrete type is returned pre-stream and carried as
// the terminal event's payload mid-stream. No second failure type, no
// second vocabulary, no converter. validation.go's AI-04 rule-class
// registry is untouched by this milestone — a review-only, git-verifiable
// fact (`git diff` shows no edit to validation.go), per S-ARP-026's "not
// repeated here" precedent, not re-proven by a test.
func TestFailure_SameConcreteTypeOnBothPaths_NoSecondTypeOrVocabulary(t *testing.T) {
	t.Parallel()

	t.Run("PreStreamFailure and MidStreamFailure return the identical dynamic type (S-AIP-041)", func(t *testing.T) {
		t.Parallel()

		pre := mustPreStreamFailure(t, ai.FailureCategoryTimeout)
		mid := mustMidStreamFailure(t, ai.FailureCategoryTimeout, false)
		if reflect.TypeOf(pre) != reflect.TypeOf(mid) {
			t.Fatalf("reflect.TypeOf(pre) = %v, reflect.TypeOf(mid) = %v, want identical", reflect.TypeOf(pre), reflect.TypeOf(mid))
		}
	})

	t.Run("the value ErrorEvent carries IS the value passed to it — no converter or copy in between (S-AIP-041/043)", func(t *testing.T) {
		t.Parallel()

		f := mustMidStreamFailure(t, ai.FailureCategoryTimeout, false)
		event := mustErrorEvent(t, f)
		carried, ok := event.ErrorPayload()
		if !ok {
			t.Fatal("event.ErrorPayload() reported no payload on an event of its own kind")
		}
		if carried != f {
			t.Error("event.ErrorPayload() returned a different *Failure than the one passed to ErrorEvent, " +
				"want the identical pointer — proof that no converter or copy runs in between")
		}
	})

	t.Run("provider_failure.go declares exactly five types — no second failure type or vocabulary (S-AIP-042/044)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "provider_failure.go", nil, 0)
		if err != nil {
			t.Fatalf("parsing provider_failure.go: %v", err)
		}
		var declaredTypes []string
		for _, decl := range file.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				if typed, ok := spec.(*ast.TypeSpec); ok {
					declaredTypes = append(declaredTypes, typed.Name.Name)
				}
			}
		}
		want := []string{"FailureCategory", "DeliveryPath", "RetryDelay", "FailureReport", "Failure"}
		if !reflect.DeepEqual(declaredTypes, want) {
			t.Errorf("provider_failure.go declares types %v, want exactly %v — R-AIP-013 forbids a second "+
				"failure type or a second vocabulary", declaredTypes, want)
		}
	})

	t.Run("no converter/mapper identifier is exported (S-AIP-043)", func(t *testing.T) {
		t.Parallel()

		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "provider_failure.go", nil, 0)
		if err != nil {
			t.Fatalf("parsing provider_failure.go: %v", err)
		}
		forbidden := []string{"Convert", "ToFailure", "FromFailure", "Adapt"}
		for _, name := range exportedTopLevelNames(file) {
			for _, bad := range forbidden {
				if strings.Contains(name, bad) {
					t.Errorf("provider_failure.go exports %q — R-AIP-013 forbids a converter between a "+
						"pre-stream and a mid-stream representation; there is only one representation", name)
				}
			}
		}
	})
}

// theCategorySentinels pairs every charter category with its own sentinel
// (design.md's exact list), named by hand — theFailureCategoryVocabulary's
// own precedent: a table the package supplied would agree with itself no
// matter what was added to it. It also proves R-AIP-014's "no umbrella
// sentinel": each row's sentinel matches only its own category, so there is
// no single value every category would match.
var theCategorySentinels = []struct {
	category ai.FailureCategory
	sentinel error
}{
	{ai.FailureCategoryAuthentication, ai.ErrAuthentication},
	{ai.FailureCategoryAuthorization, ai.ErrAuthorization},
	{ai.FailureCategoryRateLimit, ai.ErrRateLimited},
	{ai.FailureCategoryUnavailable, ai.ErrUnavailable},
	{ai.FailureCategoryTimeout, ai.ErrTimeout},
	{ai.FailureCategoryCancellation, ai.ErrCancelled},
	{ai.FailureCategoryMalformedResponse, ai.ErrMalformedResponse},
	{ai.FailureCategoryUnsupportedCapability, ai.ErrUnsupportedCapability},
	{ai.FailureCategoryUnknown, ai.ErrUnknownFailure},
}

// R-AIP-014 — per-category sentinels for errors.Is without pre-unwrapping;
// errors.As reaches the full failure; both survive at least one wrap;
// Unwrap() keeps the cause's own chain intact. No umbrella sentinel: each
// category matches its own sentinel and no other.
func TestFailure_PerCategorySentinels_SurviveAWrapForBothIsAndAs(t *testing.T) {
	t.Parallel()

	t.Run("every category matches its own sentinel, and no other — no umbrella sentinel (S-AIP-045)", func(t *testing.T) {
		t.Parallel()

		if len(theCategorySentinels) != 9 {
			t.Fatalf("test fixture error: theCategorySentinels has %d rows, want 9", len(theCategorySentinels))
		}
		for _, row := range theCategorySentinels {
			f := mustPreStreamFailure(t, row.category)
			if !errors.Is(f, row.sentinel) {
				t.Errorf("category %v: errors.Is(f, its own sentinel) = false, want true", row.category)
			}
			for _, other := range theCategorySentinels {
				if other.category == row.category {
					continue
				}
				if errors.Is(f, other.sentinel) {
					t.Errorf("category %v: errors.Is(f, %v's sentinel) = true, want false — no umbrella sentinel", row.category, other.category)
				}
			}
		}
	})

	t.Run("errors.Is matches the failure's own category sentinel through one wrap, without pre-unwrapping (S-AIP-045)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailure(t, ai.FailureCategoryRateLimit)
		wrapped := fmt.Errorf("adapter: %w", f)
		if !errors.Is(wrapped, ai.ErrRateLimited) {
			t.Error("errors.Is(wrapped, ai.ErrRateLimited) = false, want true")
		}
		if errors.Is(wrapped, ai.ErrTimeout) {
			t.Error("errors.Is(wrapped, ai.ErrTimeout) = true, want false — the failure's category is rate-limit, not timeout")
		}
	})

	t.Run("errors.As reaches the full *Failure through one wrap (S-AIP-046)", func(t *testing.T) {
		t.Parallel()

		f := mustPreStreamFailure(t, ai.FailureCategoryUnavailable)
		wrapped := fmt.Errorf("adapter: %w", f)
		var got *ai.Failure
		if !errors.As(wrapped, &got) {
			t.Fatal("errors.As(wrapped, &got) = false, want true")
		}
		if got != f {
			t.Error("errors.As reached a different *Failure than the one wrapped")
		}
	})

	t.Run("the cause's own identity survives through Unwrap(), through the failure's own wrap and a second, adapter-added wrap (S-AIP-047)", func(t *testing.T) {
		t.Parallel()

		sentinelCause := errors.New("upstream io timeout")
		f := mustPreStreamFailureReport(t, ai.FailureReport{Category: ai.FailureCategoryTimeout, Cause: sentinelCause})
		if !errors.Is(f, sentinelCause) {
			t.Error("errors.Is(f, sentinelCause) = false, want true — the cause's own identity must survive through Unwrap()")
		}
		wrapped := fmt.Errorf("adapter: %w", f)
		if !errors.Is(wrapped, sentinelCause) {
			t.Error("errors.Is(wrapped, sentinelCause) = false, want true — through both the adapter's wrap and the failure's own Unwrap()")
		}
		// The category sentinel is still reachable through the same wrap —
		// the cause's chain and the category sentinel coexist, neither
		// shadowing the other.
		if !errors.Is(wrapped, ai.ErrTimeout) {
			t.Error("errors.Is(wrapped, ai.ErrTimeout) = false, want true — the category sentinel must remain reachable alongside the cause's own chain")
		}
	})
}

// R-AIP-015 — either delivery path alone classifies every failure; the
// accessor sets on both paths are identical (they are the same concrete
// type, S-AIP-041, so this is the behavioral consequence of that structural
// fact, proven per category rather than asserted once).
func TestFailure_BothDeliveryPaths_ClassifyEveryCategoryWithIdenticalAccessors(t *testing.T) {
	t.Parallel()

	for _, row := range theCategorySentinels {
		t.Run(row.category.String(), func(t *testing.T) {
			t.Parallel()

			// "a pre-stream-only consumer" — never sees an Event at all.
			preStream := mustPreStreamFailure(t, row.category)

			// "a terminal-event-only consumer" — only ever reaches a
			// *Failure through event.ErrorPayload(), after MidStreamFailure
			// + ErrorEvent.
			mid, err := ai.MidStreamFailure(ai.FailureReport{Category: row.category}, false)
			if err != nil {
				t.Fatalf("ai.MidStreamFailure(category=%v) returned %v, want no failure", row.category, err)
			}
			event := mustErrorEvent(t, mid)
			terminalEventOnly, ok := event.ErrorPayload()
			if !ok {
				t.Fatal("event.ErrorPayload() reported no payload on an event of its own kind")
			}

			// Both consumers classify the same category through the exact
			// same accessor calls (S-AIP-048/049).
			if preStream.Category() != terminalEventOnly.Category() {
				t.Errorf("Category(): pre-stream = %v, terminal-event = %v, want equal", preStream.Category(), terminalEventOnly.Category())
			}
			if !errors.Is(preStream, row.sentinel) {
				t.Errorf("a pre-stream-only consumer cannot classify category %v via errors.Is", row.category)
			}
			if !errors.Is(terminalEventOnly, row.sentinel) {
				t.Errorf("a terminal-event-only consumer cannot classify category %v via errors.Is", row.category)
			}

			// Identical accessor sets on both paths (S-AIP-050): every
			// accessor this milestone exports is callable on both values
			// with no compile error and no panic — which is what
			// "identical set" means for two values of one concrete type.
			for _, f := range []*ai.Failure{preStream, terminalEventOnly} {
				_ = f.Category()
				_ = f.Retryable()
				_, _ = f.RetryAfter()
				_ = f.PartialOutput()
				_ = f.Delivery()
				_ = f.RawLabel()
				_, _ = f.StatusClass()
				_ = f.RequestID()
				_ = f.Error()
				_ = f.Unwrap()
			}
		})
	}
}

// NFR-AIP-B, S-AIP-052 — totality: no exported entry point of this
// milestone panics for a zero, nil or out-of-range input. Mirrors
// completion_test.go's TestResponseEvents_ExtremeInputs_NeverPanic and
// event_test.go's TestEvent_ExtremeInputs_NeverPanics shape: each case
// recovers its own panic, so one panicking case does not hide the rest.
func TestFailure_ExtremeInputs_NeverPanic(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		act  func()
	}{
		{"the nil *Failure, read every exported way", func() {
			var f *ai.Failure
			_ = f.Error()
			_ = f.Unwrap()
			_ = f.Is(ai.ErrTimeout)
			_ = f.Category()
			_ = f.Retryable()
			_, _ = f.RetryAfter()
			_ = f.PartialOutput()
			_ = f.Delivery()
			_ = f.RawLabel()
			_, _ = f.StatusClass()
			_ = f.RequestID()
		}},
		{"the zero FailureCategory, read every way", func() {
			var c ai.FailureCategory
			_ = c.String()
			_ = c.Validate()
		}},
		{"an out-of-range FailureCategory", func() {
			c := ai.FailureCategory(255)
			_ = c.String()
			_ = c.Validate()
		}},
		{"PreStreamFailure with the entirely zero-value FailureReport", func() {
			_, _ = ai.PreStreamFailure(ai.FailureReport{})
		}},
		{"MidStreamFailure with an out-of-range category and a nil cause", func() {
			_, _ = ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategory(200), Cause: nil}, true)
		}},
		{"a report with an over-long raw label and request ID", func() {
			over := strings.Repeat("z", 1000)
			_, _ = ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnknown, RawLabel: over, RequestID: over})
		}},
		{"a report with an out-of-range negative and positive status class", func() {
			_, _ = ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, StatusClass: -999})
			_, _ = ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable, StatusClass: 999})
		}},
		{"ErrorEvent(nil) — the nil payload case", func() {
			_, _ = ai.ErrorEvent(nil)
		}},
		{"a wrong-kind accessor on the zero Event", func() {
			_, _ = ai.Event{}.ErrorPayload()
		}},
		{"CheckEmit on the zero (unconstructed) Event", func() {
			_ = ai.CheckEmit(ai.Event{})
		}},
		{"FailureCategories called repeatedly stays internally consistent", func() {
			_ = ai.FailureCategories()
			_ = ai.FailureCategories()
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recovered := recover(); recovered != nil {
					t.Errorf("panicked: %v", recovered)
				}
			}()
			tc.act()
		})
	}
}

// NFR-AIP-C, S-AIP-053 — every rejecting scenario in this milestone resolves
// through AI-04's one failure value (*ai.Violation, errors.As-reachable)
// and a landed sentinel, positioned to name the offending field. Each row
// is proven at its own point of origin already (S-AIP-005,
// TestNewFailure_InvalidCategory…, TestNewFailure_OutOfRangeStatusClass…);
// this closing table confirms the rejection set is exactly these four
// paths — not more, not fewer — gathered in one place rather than scattered.
func TestProviderFailure_EveryRejectionPath_RoutesThroughAI04(t *testing.T) {
	t.Parallel()

	rejections := []struct {
		name     string
		act      func() error
		sentinel error
		position string
	}{
		{
			name:     "FailureCategory.Validate on the zero value",
			act:      func() error { var c ai.FailureCategory; return c.Validate(ai.At("category")) },
			sentinel: ai.ErrNotInVocabulary,
			position: "category",
		},
		{
			name:     "PreStreamFailure with an invalid (zero-value) category",
			act:      func() error { _, err := ai.PreStreamFailure(ai.FailureReport{}); return err },
			sentinel: ai.ErrNotInVocabulary,
			position: "category",
		},
		{
			name: "PreStreamFailure with an out-of-range status class",
			act: func() error {
				_, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout, StatusClass: 9})
				return err
			},
			sentinel: ai.ErrOutOfRange,
			position: "status_class",
		},
		{
			name:     "ErrorEvent(nil)",
			act:      func() error { _, err := ai.ErrorEvent(nil); return err },
			sentinel: ai.ErrEmpty,
			position: "payload",
		},
	}

	for _, row := range rejections {
		t.Run(row.name, func(t *testing.T) {
			t.Parallel()

			err := row.act()
			if err == nil {
				t.Fatal("act() returned nil, want a rejection")
			}
			if !errors.Is(err, row.sentinel) {
				t.Errorf("errors.Is(err, the expected sentinel) = false, want true; err = %v", err)
			}
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, &violation) = false, want true — every rejection must route "+
					"through AI-04's one failure value; err = %v", err)
			}
			if got := violation.Path().String(); got != row.position {
				t.Errorf("violation position = %q, want %q", got, row.position)
			}
		})
	}
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

// canaryCause is a test-local, value-kind error used only to plant a canary
// in [FailureReport.Cause] for TestFailure_GoString_RedactsLikeError.
//
// It is deliberately value-kind (a defined string type), not a pointer
// wrapping errors.New(...): fmt's reflective %#v walk prints an unexported
// pointer field's target as a bare hex address (CanInterface is false for a
// field reached through an unexported struct field, so the pointer's own
// Error() method is never consulted), which would make a canary planted only
// there pass today, before GoString exists — a vacuous RED (design.md D4).
// A value-kind cause has no such escape: its contents are reflected
// verbatim, which is exactly what makes today's %#v rendering unsafe.
type canaryCause string

// Error implements the error interface for canaryCause.
func (c canaryCause) Error() string { return string(c) }

// R-AIP-016 — redaction is a property of the failure payload, not of the
// caller's formatting verb: %#v must render exactly Error()'s text, the same
// as %v/%s/%+v, and must never reflect over the cause or any other
// unexported field. Proven with three planted canaries — content_part_test.go
// TestPart_String_CarriesNoPayload's adversarial-verb-loop shape, restated
// for *Failure and %#v specifically (D-4).
func TestFailure_GoString_RedactsLikeError(t *testing.T) {
	t.Parallel()

	const (
		canaryRawLabel  = "raw-label-CANARY-77f3"
		canaryRequestID = "request-id-CANARY-88e1"
		canaryCauseText = "cause-CANARY-99a2"
	)

	f := mustPreStreamFailureReport(t, ai.FailureReport{
		Category:  ai.FailureCategoryUnknown,
		RawLabel:  canaryRawLabel,
		RequestID: canaryRequestID,
		Cause:     canaryCause(canaryCauseText),
	})

	t.Run("no verb reproduces a planted canary (S-AIP-056)", func(t *testing.T) {
		t.Parallel()

		for _, verb := range []string{"%v", "%s", "%+v", "%#v", "%q"} {
			rendered := fmt.Sprintf(verb, f)
			for _, canary := range []string{canaryRawLabel, canaryRequestID, canaryCauseText} {
				if strings.Contains(rendered, canary) {
					t.Errorf("fmt.Sprintf(%q, f) = %q, contains the planted canary %q, want it excluded", verb, rendered, canary)
				}
			}
		}
	})

	t.Run("%#v is byte-for-byte identical to Error() (S-AIP-057)", func(t *testing.T) {
		t.Parallel()

		want := f.Error()
		if got := fmt.Sprintf("%#v", f); got != want {
			t.Errorf("fmt.Sprintf(\"%%#v\", f) = %q, want %q (Error()'s own text, unchanged)", got, want)
		}
	})
}

// NFR-AIP-B — totality: %#v on a nil *Failure must never panic, and must
// agree with (*ai.Failure)(nil).Error() rather than a hard-coded string
// literal, so the assertion cannot silently drift from noProviderFailure's
// actual text (D-5: GoString delegates to Error(), which already has the one
// nil check).
func TestFailure_GoString_NilReceiver_TotalByDelegation(t *testing.T) {
	t.Parallel()

	want := (*ai.Failure)(nil).Error()

	for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
		t.Run(verb, func(t *testing.T) {
			t.Parallel()

			// No recover guard: fmt installs its own panic catcher around a
			// GoString/Error method call (handleMethods' catchPanic), so a
			// panicking method never escapes Sprintf — it renders as a
			// "%!verb(PANIC=...)" marker instead. The equality assertion
			// below is therefore what enforces nil totality: a panic would
			// surface as that marker and fail the comparison.
			if got := fmt.Sprintf(verb, (*ai.Failure)(nil)); got != want {
				t.Errorf("fmt.Sprintf(%q, (*ai.Failure)(nil)) = %q, want %q", verb, got, want)
			}
		})
	}
}

// valueShapedCause is a test-local, value-kind error used to plant a
// canary in FailureReport.Cause for the copied-value hazard proofs below
// (AI-36, WU-6, D-1, R-AIP-017). It is deliberately value-kind with a
// VALUE receiver Error() — stored AS A VALUE in the cause chain — so a
// copied Failure's reflective rendering reaches it directly, unlike a
// pointer-shaped cause (contrast: TestFailure_CopiedValue_PointerShapedCause_DoesNotSurfaceCanary).
type valueShapedCause struct{ Detail string }

// Error implements the error interface for valueShapedCause.
func (c valueShapedCause) Error() string { return c.Detail }

// TestFailure_CopiedValue_ValueShapedCause_RendersCanary is the mechanism
// pin (S-AIP-058): it demonstrates a REAL, KNOWN leak — not a defect to
// fix, per R-AIP-017's own chosen discharge (D-1: prove the shape
// unreachable and guard it, never widen GoString's coverage to copies,
// which would re-open NFR-AIP-B for a shape no constructor produces).
// Expected GREEN: this records the scope R-AIP-017 states; it is not a
// RED-first behavior change.
//
// Acceptance condition (explicit, not a comment): the cause MUST be
// value-shaped, not pointer-shaped. Under Go-syntax rendering at nesting
// depth greater than zero, a pointer-shaped cause renders as a machine
// address rather than its text, so a canary planted only in a
// pointer-shaped cause would not surface even if the mechanism were
// unsafe — the assertion would pass vacuously. See the contrast pin
// immediately below for the proof that this placement is the one that
// bites.
func TestFailure_CopiedValue_ValueShapedCause_RendersCanary(t *testing.T) {
	t.Parallel()

	canary := "AI36-D1-VALUESHAPED-CANARY-" + "e4b7c1"

	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnavailable,
		Cause:    valueShapedCause{Detail: canary},
	}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}

	// The AI-41-deferred shape: a copied value form of the payload, not
	// *ai.Failure. GoString/Error/Unwrap/Is are all pointer-receiver, so
	// this value is neither a fmt.GoStringer nor an error — its rendering
	// falls back to raw reflection over every field, cause included.
	copied := *failure

	for _, verb := range []string{"%#v", "%v", "%+v"} {
		rendered := fmt.Sprintf(verb, copied)
		if !strings.Contains(rendered, canary) {
			t.Fatalf("fmt.Sprintf(%q, copied) = %q, want it to contain the canary %q — a copied Failure value is outside GoString's pointer-receiver coverage (R-AIP-017 item 1); if this ever stops leaking, a value-receiver method was added and NFR-AIP-B's tradeoff must be re-argued in the open (design.md AD-3)", verb, rendered, canary)
		}
	}
}

// TestFailure_CopiedValue_PointerShapedCause_DoesNotSurfaceCanary is the
// depth-hazard contrast pin (S-AIP-059): identical construction with a
// POINTER-shaped cause (errors.New, an *errorString) does NOT surface the
// canary under the same verbs — documenting, in-test, why the
// value-shaped plant above is mandatory: under Go-syntax rendering at
// nesting depth greater than zero, fmt renders a pointer-typed field as a
// machine address, never its text, so a canary planted only in a
// pointer-shaped cause would pass "absent" vacuously even if the
// underlying mechanism were unsafe.
func TestFailure_CopiedValue_PointerShapedCause_DoesNotSurfaceCanary(t *testing.T) {
	t.Parallel()

	canary := "AI36-D1-POINTERSHAPED-CANARY-" + "9f2a6d"

	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnavailable,
		Cause:    errors.New("upstream: " + canary),
	}, false)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure returned %v, want no failure", err)
	}

	copied := *failure

	for _, verb := range []string{"%#v", "%v", "%+v"} {
		rendered := fmt.Sprintf(verb, copied)
		if strings.Contains(rendered, canary) {
			t.Errorf("fmt.Sprintf(%q, copied) = %q, unexpectedly contains the pointer-shaped canary — want it absent, establishing that the pointer-shaped placement is vacuous and the value-shaped placement above is the one that can bite (S-AIP-059, design.md AD-3)", verb, rendered)
		}
	}
}

// isFailureByValueType reports whether t (a parsed field/parameter/result
// type expression) names the failure payload BY VALUE: the bare
// identifier Failure when parsing a file that is itself package ai, or
// any qualified <pkg>.Failure selector from any other package (AI-36,
// WU-6, R-AIP-017 item 2). A pointer form is never matched here — callers
// pass field.Type directly, and *ast.StarExpr falls through to the
// default case, which is exactly the point: the guard flags only the
// by-value shape.
func isFailureByValueType(t ast.Expr, inAIPackage bool) bool {
	switch e := t.(type) {
	case *ast.Ident:
		return inAIPackage && e.Name == "Failure"
	case *ast.SelectorExpr:
		return e.Sel.Name == "Failure"
	default:
		return false
	}
}

// embeddedFieldName derives the field name Go assigns to an embedded
// (anonymous) struct field from its type expression — the tail
// identifier, dereferencing one leading pointer if present.
func embeddedFieldName(t ast.Expr) string {
	switch e := t.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.StarExpr:
		return embeddedFieldName(e.X)
	default:
		return ""
	}
}

// findingsInFile inspects one parsed file's declarations for an exported
// function/method signature or an exported struct field that carries the
// failure payload by value (R-AIP-017 item 2), returning one finding per
// offending declaration naming its position and shape — never any
// payload content, since a Go identifier is not a secret.
func findingsInFile(fset *token.FileSet, file *ast.File) []string {
	var findings []string
	inAI := file.Name.Name == "ai"

	for _, decl := range file.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			if !d.Name.IsExported() {
				continue
			}
			if d.Type.Params != nil {
				for _, field := range d.Type.Params.List {
					if isFailureByValueType(field.Type, inAI) {
						findings = append(findings, fmt.Sprintf("%s: func %s: a parameter carries the failure payload by value", fset.Position(d.Pos()), d.Name.Name))
					}
				}
			}
			if d.Type.Results != nil {
				for _, field := range d.Type.Results.List {
					if isFailureByValueType(field.Type, inAI) {
						findings = append(findings, fmt.Sprintf("%s: func %s: a return value carries the failure payload by value", fset.Position(d.Pos()), d.Name.Name))
					}
				}
			}
		case *ast.GenDecl:
			if d.Tok != token.TYPE {
				continue
			}
			for _, spec := range d.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok || !ts.Name.IsExported() {
					continue
				}
				st, ok := ts.Type.(*ast.StructType)
				if !ok || st.Fields == nil {
					continue
				}
				for _, field := range st.Fields.List {
					if len(field.Names) == 0 {
						name := embeddedFieldName(field.Type)
						if name != "" && ast.IsExported(name) && isFailureByValueType(field.Type, inAI) {
							findings = append(findings, fmt.Sprintf("%s: struct %s: embedded field %s carries the failure payload by value", fset.Position(field.Pos()), ts.Name.Name, name))
						}
						continue
					}
					for _, name := range field.Names {
						if name.IsExported() && isFailureByValueType(field.Type, inAI) {
							findings = append(findings, fmt.Sprintf("%s: struct %s: field %s carries the failure payload by value", fset.Position(field.Pos()), ts.Name.Name, name.Name))
						}
					}
				}
			}
		}
	}
	return findings
}

// scanModuleForByValueFailure walks every non-test .go file under root
// (this module's whole source tree) and reports every finding
// findingsInFile produces (R-AIP-017 item 2's structural guard,
// S-AIP-060). testdata directories are skipped, matching the Go
// toolchain's own convention: nothing under testdata/ is part of the
// module's published (importable, buildable) surface.
func scanModuleForByValueFailure(root string) ([]string, error) {
	var findings []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == "testdata" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		fset := token.NewFileSet()
		file, perr := parser.ParseFile(fset, path, nil, 0)
		if perr != nil {
			return perr
		}
		findings = append(findings, findingsInFile(fset, file)...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return findings, nil
}

// TestNoPublishedSurfaceReturnsFailureByValue is the AST escape guard
// (S-AIP-060, R-AIP-017 item 2): a go/parser walk over every non-test .go
// file in this module flags any exported function/method signature or
// exported struct field that carries the failure payload by value. The
// current tree is clean.
//
// ".." is this module's whole source tree (src/, both src/ai and
// src/agenttest) from this test's own working directory (src/ai) — go
// test always sets the working directory to the package under test, the
// same convention this file's other go/parser-based guards already rely
// on (e.g. parser.ParseFile(fset, "provider_failure.go", nil, 0) above).
func TestNoPublishedSurfaceReturnsFailureByValue(t *testing.T) {
	t.Parallel()

	findings, err := scanModuleForByValueFailure("..")
	if err != nil {
		t.Fatalf("scanModuleForByValueFailure: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("found %d published shape(s) carrying the failure payload by value, want 0 (R-AIP-017 item 2): %v", len(findings), findings)
	}
}

// TestNoPublishedSurfaceReturnsFailureByValue_PositiveControl proves the
// guard is falsifiable (S-AIP-060's own control): an in-memory synthetic
// source declaring a published function that returns the failure payload
// by value MUST be flagged.
func TestNoPublishedSurfaceReturnsFailureByValue_PositiveControl(t *testing.T) {
	t.Parallel()

	const syntheticLeakSource = `package synthetic

import "github.com/cachicamas/backend/agent/src/ai"

func Leak() ai.Failure { return ai.Failure{} }
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "synthetic_leak.go", syntheticLeakSource, 0)
	if err != nil {
		t.Fatalf("parser.ParseFile: %v", err)
	}

	findings := findingsInFile(fset, file)
	if len(findings) == 0 {
		t.Fatal("findingsInFile did not flag func Leak() ai.Failure, want the guard to catch it — an escape guard that can never fail is not a proof (S-AIP-060)")
	}
}

// TestFailure_PublishedRenderingSetUnchanged_AbsentPayloadTotality is the
// regression pin (S-AIP-061): R-AIP-017 changes no rendering behavior —
// the payload's published rendering set (String is not exported on
// *Failure; Error and GoString are) is identical to what R-AIP-016
// landed, and every absent-payload (nil *Failure) rendering under all
// four verbs still yields the contract's defined rendering and never
// panics (NFR-AIP-B, S-AIP-052, S-AIP-057).
func TestFailure_PublishedRenderingSetUnchanged_AbsentPayloadTotality(t *testing.T) {
	t.Parallel()

	failureMethods := exportedMethodNames(reflect.TypeOf(&ai.Failure{}))
	wantRenderingMethods := map[string]bool{"Error": true, "GoString": true}
	for name := range wantRenderingMethods {
		if !failureMethods[name] {
			t.Errorf("*ai.Failure no longer exports %s, want the rendering set R-AIP-016 landed unchanged", name)
		}
	}
	// String is deliberately NOT part of the published rendering set —
	// R-AIP-016 never added one, and R-AIP-017 item 6 forbids adding any
	// new rendering behavior to the payload's published set.
	if failureMethods["String"] {
		t.Error("*ai.Failure now exports String, want the rendering set unchanged from R-AIP-016 (R-AIP-017 item 6)")
	}

	want := (*ai.Failure)(nil).Error()
	for _, verb := range []string{"%v", "%s", "%+v", "%#v"} {
		if got := fmt.Sprintf(verb, (*ai.Failure)(nil)); got != want {
			t.Errorf("fmt.Sprintf(%q, (*ai.Failure)(nil)) = %q, want %q (NFR-AIP-B, S-AIP-052, S-AIP-057)", verb, got, want)
		}
	}
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
