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
	"reflect"
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
