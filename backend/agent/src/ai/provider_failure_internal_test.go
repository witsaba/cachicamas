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

import "testing"

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
