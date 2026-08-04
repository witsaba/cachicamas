package openaicompat

import (
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestClient_HasNoStreamingEntryPoint covers S-ATS-020 (AI-28, R-ATS-001,
// R-ATS-006): the landed adapter exposes a Stream method. Flipped in place
// from its AI-25 form (S-APC-030), which asserted the opposite — that no
// streaming entry point existed yet, because streaming had not arrived.
// The name and the test identity are unchanged; only the assertion's
// polarity is inverted, never removed (S-ATS-020, S-ATS-022): it now fails
// if Stream is absent and passes only once the method exists.
func TestClient_HasNoStreamingEntryPoint(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(&Client{})
	if _, ok := typ.MethodByName("Stream"); !ok {
		t.Fatal("*Client exposes no Stream method — AI-28 must land it (R-ATS-001, S-ATS-020)")
	}
}

// TestClient_DoesNotSatisfyModelProviderAtRuntime covers S-ATS-021 (AI-28,
// R-ATS-001, R-ATS-006): a run-time type assertion reports the adapter DOES
// implement ai.ModelProvider. Flipped in place from its AI-25 form
// (S-APC-031), which asserted the opposite.
//
// This is deliberately NOT the compile-time idiom
// var _ ai.ModelProvider = (*Client)(nil), which would fail the *build*
// rather than fail as a test — the wrong failure mode for a guard whose
// whole point is proving the adapter now advertises an interface it truly
// implements, checked the identical way its AI-25 predecessor proved the
// opposite (S-ATS-021).
func TestClient_DoesNotSatisfyModelProviderAtRuntime(t *testing.T) {
	t.Parallel()

	_, ok := any(&Client{}).(ai.ModelProvider)
	if !ok {
		t.Fatal("*Client does not satisfy ai.ModelProvider — AI-28 must land Stream (R-ATS-001, S-ATS-021)")
	}
}
