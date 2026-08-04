// AI-23.2 amendment — the permanent negative proof for R-CNF-020: the
// lifecycle-prefix check must be able to fail against a synthetic,
// start-less event slice, and must distinguish identity equality from mere
// presence, forever — never a staged mutation demonstrated once and
// deleted (D2, TestSequenceGuard_*_Fails idiom).
//
// White-box (package agenttest, not agenttest_test): checkLifecyclePrefix
// is unexported, matching conformance_suite_test.go's own precedent for
// this directory's runner-internal tests.

package agenttest

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestLifecyclePrefixGuard_StartlessSlice_FailsNamingAbsentResponseStart
// proves S-CLA-015: a synthetic slice whose first (and only) event is not a
// response start is rejected, naming the absent lifecycle event.
func TestLifecyclePrefixGuard_StartlessSlice_FailsNamingAbsentResponseStart(t *testing.T) {
	start, err := ai.NewTextBlockStart(1)
	requireConstructed(t, err, "ai.NewTextBlockStart")

	err = checkLifecyclePrefix([]ai.Event{start}, "resp-1", "model-1")
	if err == nil {
		t.Fatal("checkLifecyclePrefix(start-less slice) = nil, want a violation naming the absent lifecycle event (S-CLA-015)")
	}
	if !contains(err.Error(), ai.EventKindResponseStart.String()) {
		t.Errorf("error %q does not name %v, the absent lifecycle event", err, ai.EventKindResponseStart)
	}
}

// TestLifecyclePrefixGuard_MismatchedIdentity_FailsNamingField proves
// S-CLA-016: equality, not mere presence, is what is asserted — a response
// start present at position 0 whose ResponseID() or ServedModel() differs
// from the wanted values is rejected, naming the mismatched field.
func TestLifecyclePrefixGuard_MismatchedIdentity_FailsNamingField(t *testing.T) {
	t.Run("response id differs", func(t *testing.T) {
		start, err := ai.NewResponseStart("other-id", "other-model")
		requireConstructed(t, err, "ai.NewResponseStart")

		err = checkLifecyclePrefix([]ai.Event{start}, "wanted-id", "other-model")
		if err == nil {
			t.Fatal("checkLifecyclePrefix(mismatched response id) = nil, want a violation naming the field (S-CLA-016)")
		}
		if !contains(err.Error(), "ResponseID") {
			t.Errorf("error %q does not name ResponseID, the mismatched field", err)
		}
	})

	t.Run("served model differs", func(t *testing.T) {
		start, err := ai.NewResponseStart("other-id", "other-model")
		requireConstructed(t, err, "ai.NewResponseStart")

		err = checkLifecyclePrefix([]ai.Event{start}, "other-id", "wanted-model")
		if err == nil {
			t.Fatal("checkLifecyclePrefix(mismatched served model) = nil, want a violation naming the field (S-CLA-016)")
		}
		if !contains(err.Error(), "ServedModel") {
			t.Errorf("error %q does not name ServedModel, the mismatched field", err)
		}
	})
}

// TestLifecyclePrefixGuard_Passes is the positive companion: a response
// start at position 0 carrying exactly the wanted identity is accepted.
func TestLifecyclePrefixGuard_Passes(t *testing.T) {
	start, err := ai.NewResponseStart("resp-1", "model-1")
	requireConstructed(t, err, "ai.NewResponseStart")

	if err := checkLifecyclePrefix([]ai.Event{start}, "resp-1", "model-1"); err != nil {
		t.Errorf("checkLifecyclePrefix(matching identity) = %v, want nil", err)
	}
}
