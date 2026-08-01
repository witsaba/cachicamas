// Internal tests for AI-19 — the provider error taxonomy and the terminal
// error event.
//
// This file is package ai, not ai_test, for the same reason
// validation_registry_internal_test.go and content_part_internal_test.go are:
// some of what this milestone must pin is a property of the closed
// vocabulary's own unexported tables (failureCategoryLimit,
// failureCategoryNames, failureCategorySentinels, deliveryPathLimit), never
// reachable from outside the package. Everything an adapter can observe is
// proven externally, in provider_failure_test.go.
package ai

import (
	"errors"
	"testing"
)

// TestFailureCategory_ZeroValue_IsNotAMember is Phase 1's groundwork (task
// 1.1) for R-AIP-005: before provider_failure.go declares FailureCategory and
// its nine members, this file does not compile — the RED state task 1.1
// requires. The full vocabulary surface (String, Validate, FailureCategories)
// is Phase 3's; this only pins that the zero value is never one of the nine
// named members, and that the bound constant tracks the member count.
//
// Triangulation: skipped — purely structural (a closed constant set), and the
// check is already exhaustive over every named member rather than a single
// sample.
func TestFailureCategory_ZeroValue_IsNotAMember(t *testing.T) {
	var zero FailureCategory
	members := []FailureCategory{
		FailureCategoryAuthentication,
		FailureCategoryAuthorization,
		FailureCategoryRateLimit,
		FailureCategoryUnavailable,
		FailureCategoryTimeout,
		FailureCategoryCancellation,
		FailureCategoryMalformedResponse,
		FailureCategoryUnsupportedCapability,
		FailureCategoryUnknown,
	}
	for _, m := range members {
		if zero == m {
			t.Fatalf("the zero FailureCategory equals named member %v, want the zero value excluded from the vocabulary", m)
		}
	}
	if got, want := int(failureCategoryLimit), len(members)+1; got != want {
		t.Errorf("failureCategoryLimit = %d, want %d (%d charter members + the blank zero)", got, want, len(members))
	}
}

// TestDeliveryPath_ZeroValue_IsNotAMember is Phase 1's groundwork (task 1.1)
// for R-AIP-010: the zero DeliveryPath must not equal either registered
// delivery path, so a zero Failure reports an invalid delivery rather than
// forging a legal pre-stream one (design.md D8).
func TestDeliveryPath_ZeroValue_IsNotAMember(t *testing.T) {
	var zero DeliveryPath
	if zero == DeliveryPreStream {
		t.Error("the zero DeliveryPath equals DeliveryPreStream, want them distinct")
	}
	if zero == DeliveryMidStream {
		t.Error("the zero DeliveryPath equals DeliveryMidStream, want them distinct")
	}
	if got, want := int(deliveryPathLimit), 3; got != want {
		t.Errorf("deliveryPathLimit = %d, want %d (blank zero + 2 members)", got, want)
	}
}

// TestFailureCategory_ZeroAndOutOfRange_RejectWithErrNotInVocabulary is
// Phase 3's internal coverage for S-AIP-013/014: every value the package
// does not name as a member — not only the zero value — rejects through
// ErrNotInVocabulary and never panics. Internal so the sweep can walk the
// exact bound (failureCategoryLimit) rather than a hardcoded guess —
// finish_reason_test.go's TestFinishReason_AddingAValue_FailsWithoutATableAndAStringForm
// precedent, restated with access to the unexported bound instead of a
// brute-force 0..255 scan.
func TestFailureCategory_ZeroAndOutOfRange_RejectWithErrNotInVocabulary(t *testing.T) {
	named := make(map[FailureCategory]bool, int(failureCategoryLimit)-1)
	for c := FailureCategory(1); c < failureCategoryLimit; c++ {
		named[c] = true
	}
	if len(named) != 9 {
		t.Fatalf("test fixture error: failureCategoryLimit names %d members, want 9", len(named))
	}

	for candidate := 0; candidate <= 255; candidate++ {
		c := FailureCategory(candidate)
		err := c.Validate()
		if named[c] {
			if err != nil {
				t.Errorf("FailureCategory(%d).Validate() = %v, want <nil> — it is a named member", candidate, err)
			}
			continue
		}
		if err == nil {
			t.Errorf("FailureCategory(%d).Validate() = <nil>, want ErrNotInVocabulary — it names no member", candidate)
			continue
		}
		if !errors.Is(err, ErrNotInVocabulary) {
			t.Errorf("FailureCategory(%d).Validate(): errors.Is(err, ErrNotInVocabulary) = false, want true; err = %v", candidate, err)
		}
	}
}

// TestFailureCategoryNames_Exhaustiveness is task 3.5's REFACTOR pin: the
// name array's length tracks failureCategoryLimit exactly, and every named
// member has a non-empty string form. finish_reason.go carries no equivalent
// standalone pin (its own AddingAValue test, external, folds the same
// property in); this one is internal because it reads the unexported array
// directly rather than re-deriving the claim through String()'s fallback.
//
// The sentinel half of task 3.5 ("every category has a sentinel") is
// deferred to Phase 6 (task 6.4, TestFailureCategorySentinels_Exhaustiveness):
// the sentinels (ErrAuthentication … ErrUnknownFailure) do not exist until
// that phase, for the same Go-compile-order reason Phase 2/3's validate()
// wiring was staged (see apply-progress.md deviation 3).
func TestFailureCategoryNames_Exhaustiveness(t *testing.T) {
	if got, want := len(failureCategoryNames), int(failureCategoryLimit); got != want {
		t.Fatalf("len(failureCategoryNames) = %d, want %d (failureCategoryLimit) — an appended category "+
			"must move the array bound with it", got, want)
	}
	for c := FailureCategory(1); c < failureCategoryLimit; c++ {
		if failureCategoryNames[c] == "" {
			t.Errorf("failureCategoryNames[%d] is empty for named member %v, want a non-empty stable string form", c, c)
		}
	}
}
