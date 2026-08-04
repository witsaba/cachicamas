package openaicompat

import (
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestClient_HasStreamingEntryPoint covers S-ATS-020 (AI-28, R-ATS-001,
// R-ATS-006): the landed adapter exposes a Stream method. Originally
// flipped in place from its AI-25 form (S-APC-030), which asserted the
// opposite — that no streaming entry point existed yet, because streaming
// had not arrived — keeping the OLD name under design D12's own "keep
// their names' successors with inverted polarity" guidance.
//
// # Renamed (verify-report S2 corrective, superseding D12's naming choice here)
//
// The retained name read as a standing negative ("Has NO streaming entry
// point") long after the assertion itself flipped to a positive one,
// actively misleading a reader grepping the suite. Renamed to match the
// assertion's own post-flip meaning; the test identity, the assertion, and
// S-ATS-020's own polarity — fails if Stream is absent, passes only once
// the method exists — are all unchanged (S-ATS-020, S-ATS-022).
func TestClient_HasStreamingEntryPoint(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(&Client{})
	if _, ok := typ.MethodByName("Stream"); !ok {
		t.Fatal("*Client exposes no Stream method — AI-28 must land it (R-ATS-001, S-ATS-020)")
	}
}

// TestClient_SatisfiesModelProviderAtRuntime covers S-ATS-021 (AI-28,
// R-ATS-001, R-ATS-006): a run-time type assertion reports the adapter DOES
// implement ai.ModelProvider. Originally flipped in place from its AI-25
// form (S-APC-031), which asserted the opposite, under design D12's own
// name-retention guidance (see TestClient_HasStreamingEntryPoint's own doc
// comment for the verify-report S2 corrective that renamed both guards).
//
// This is deliberately NOT the compile-time idiom
// var _ ai.ModelProvider = (*Client)(nil), which would fail the *build*
// rather than fail as a test — the wrong failure mode for a guard whose
// whole point is proving the adapter now advertises an interface it truly
// implements, checked the identical way its AI-25 predecessor proved the
// opposite (S-ATS-021).
func TestClient_SatisfiesModelProviderAtRuntime(t *testing.T) {
	t.Parallel()

	_, ok := any(&Client{}).(ai.ModelProvider)
	if !ok {
		t.Fatal("*Client does not satisfy ai.ModelProvider — AI-28 must land Stream (R-ATS-001, S-ATS-021)")
	}
}
