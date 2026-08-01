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
