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

// cachicamas-ai-conformance-tool-amendment (AI-30 slice 2) — the matcher's
// own permanent negatives (R-CNF-019, R-CNF-020). checkRelativeKindOrder is
// the one shared helper the two amended tool-call cases use instead of
// requireDrainedKinds: an anchored walk, not a naive subsequence scan, so
// an unexpected kind — even one sandwiched between a consumed ToolCallStart
// and its expected ToolCallEnd — still fails unless it is specifically an
// EventKindToolCallDelta (S-CTA-006's matcher-level half). These tests are
// written against a function that does not exist yet in this package
// (compile-RED) until conformance_lifecycle.go lands it.

// TestRelativeKindOrderGuard_TolersExtraDeltasInsideOpenWindow proves the
// positive companion every amended case depends on: extra ToolCallDelta
// occurrences between a consumed ToolCallStart and its expected
// ToolCallEnd do not fail the match — the occurrence count is unconstrained
// (R-ATC-010, S-ATC-027).
func TestRelativeKindOrderGuard_TolersExtraDeltasInsideOpenWindow(t *testing.T) {
	rs, err := ai.NewResponseStart("resp-1", "model-1")
	requireConstructed(t, err, "ai.NewResponseStart")
	tcs, err := ai.NewToolCallStart(1, "call-1", "search")
	requireConstructed(t, err, "ai.NewToolCallStart")
	delta1, err := ai.NewToolCallDelta(1, []byte(`{"q":`))
	requireConstructed(t, err, "ai.NewToolCallDelta")
	delta2, err := ai.NewToolCallDelta(1, []byte(`"weather"}`))
	requireConstructed(t, err, "ai.NewToolCallDelta")
	tce, err := ai.NewToolCallEnd(1, nil)
	requireConstructed(t, err, "ai.NewToolCallEnd")
	comp, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	requireConstructed(t, err, "ai.NewCompletion")

	want := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindToolCallStart, ai.EventKindToolCallEnd, ai.EventKindCompletion}
	events := []ai.Event{rs, tcs, delta1, delta2, tce, comp}

	if err := checkRelativeKindOrder(events, want); err != nil {
		t.Errorf("checkRelativeKindOrder(two extra deltas inside the window) = %v, want nil — delta occurrences are unconstrained", err)
	}
}

// TestRelativeKindOrderGuard_ZeroExtraEvents_Passes is the same helper's
// simplest positive case: want and events agree exactly, with no deltas at
// all — the zero-delta shape every amended case must also accept.
func TestRelativeKindOrderGuard_ZeroExtraEvents_Passes(t *testing.T) {
	rs, err := ai.NewResponseStart("resp-1", "model-1")
	requireConstructed(t, err, "ai.NewResponseStart")
	tcs, err := ai.NewToolCallStart(1, "call-1", "search")
	requireConstructed(t, err, "ai.NewToolCallStart")
	tce, err := ai.NewToolCallEnd(1, []byte(`{}`))
	requireConstructed(t, err, "ai.NewToolCallEnd")
	comp, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	requireConstructed(t, err, "ai.NewCompletion")

	want := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindToolCallStart, ai.EventKindToolCallEnd, ai.EventKindCompletion}
	events := []ai.Event{rs, tcs, tce, comp}

	if err := checkRelativeKindOrder(events, want); err != nil {
		t.Errorf("checkRelativeKindOrder(exact match, zero deltas) = %v, want nil", err)
	}
}

// TestRelativeKindOrderGuard_DroppedEnd_FailsNamingMissingKind proves
// S-CTA-009: given a subject that drops the scripted tool-call end
// entirely, the helper still fails, naming the missing kind — this is a
// permanent artifact of the suite, not a staged mutation demonstrated once
// and deleted (obs #2471 shape 9).
func TestRelativeKindOrderGuard_DroppedEnd_FailsNamingMissingKind(t *testing.T) {
	rs, err := ai.NewResponseStart("resp-1", "model-1")
	requireConstructed(t, err, "ai.NewResponseStart")
	tcs, err := ai.NewToolCallStart(1, "call-1", "search")
	requireConstructed(t, err, "ai.NewToolCallStart")
	comp, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	requireConstructed(t, err, "ai.NewCompletion")

	want := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindToolCallStart, ai.EventKindToolCallEnd, ai.EventKindCompletion}
	events := []ai.Event{rs, tcs, comp} // ToolCallEnd dropped entirely

	err = checkRelativeKindOrder(events, want)
	if err == nil {
		t.Fatal("checkRelativeKindOrder(dropped end) = nil, want a violation naming the missing kind (S-CTA-009)")
	}
	if !contains(err.Error(), ai.EventKindToolCallEnd.String()) {
		t.Errorf("error %q does not name %v, the dropped kind", err, ai.EventKindToolCallEnd)
	}
}

// TestRelativeKindOrderGuard_UnexpectedKind_FailsNamingIndexGotAndWant
// proves the matcher's core negative: an event that is neither the next
// wanted kind nor a tolerated delta inside an open window fails, naming
// the index, the kind actually found and the kind wanted there.
func TestRelativeKindOrderGuard_UnexpectedKind_FailsNamingIndexGotAndWant(t *testing.T) {
	rs, err := ai.NewResponseStart("resp-1", "model-1")
	requireConstructed(t, err, "ai.NewResponseStart")
	textStart, err := ai.NewTextBlockStart(1)
	requireConstructed(t, err, "ai.NewTextBlockStart")
	comp, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	requireConstructed(t, err, "ai.NewCompletion")

	want := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindCompletion}
	events := []ai.Event{rs, textStart, comp} // TextBlockStart is neither wanted nor a tolerated delta

	err = checkRelativeKindOrder(events, want)
	if err == nil {
		t.Fatal("checkRelativeKindOrder(unexpected kind) = nil, want a violation")
	}
	if !contains(err.Error(), ai.EventKindTextBlockStart.String()) {
		t.Errorf("error %q does not name %v, the kind actually found", err, ai.EventKindTextBlockStart)
	}
	if !contains(err.Error(), ai.EventKindCompletion.String()) {
		t.Errorf("error %q does not name %v, the kind wanted at that position", err, ai.EventKindCompletion)
	}
}

// TestRelativeKindOrderGuard_EventsExhausted_FailsNamingFirstMissingKind
// proves the matcher's other negative branch: the drained window ending
// before every wanted kind was consumed fails, naming the first kind never
// seen.
func TestRelativeKindOrderGuard_EventsExhausted_FailsNamingFirstMissingKind(t *testing.T) {
	rs, err := ai.NewResponseStart("resp-1", "model-1")
	requireConstructed(t, err, "ai.NewResponseStart")

	want := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindToolCallStart, ai.EventKindCompletion}
	events := []ai.Event{rs} // the drained window ends right after the prefix

	err = checkRelativeKindOrder(events, want)
	if err == nil {
		t.Fatal("checkRelativeKindOrder(exhausted events) = nil, want a violation naming the first missing kind")
	}
	if !contains(err.Error(), ai.EventKindToolCallStart.String()) {
		t.Errorf("error %q does not name %v, the first kind never seen", err, ai.EventKindToolCallStart)
	}
}

// TestRelativeKindOrderGuard_TextDeltaInsideToolCallWindow_Fails proves
// S-CTA-006's matcher-level half: the window's delta tolerance is scoped to
// EventKindToolCallDelta specifically — a text delta occurring after the
// tool-call start is not tolerated, even though it occurs positionally
// "inside" the open window.
func TestRelativeKindOrderGuard_TextDeltaInsideToolCallWindow_Fails(t *testing.T) {
	rs, err := ai.NewResponseStart("resp-1", "model-1")
	requireConstructed(t, err, "ai.NewResponseStart")
	tcs, err := ai.NewToolCallStart(1, "call-1", "search")
	requireConstructed(t, err, "ai.NewToolCallStart")
	stray, err := ai.NewTextDelta(2, "stray")
	requireConstructed(t, err, "ai.NewTextDelta")
	tce, err := ai.NewToolCallEnd(1, []byte(`{}`))
	requireConstructed(t, err, "ai.NewToolCallEnd")
	comp, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	requireConstructed(t, err, "ai.NewCompletion")

	want := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindToolCallStart, ai.EventKindToolCallEnd, ai.EventKindCompletion}
	events := []ai.Event{rs, tcs, stray, tce, comp}

	err = checkRelativeKindOrder(events, want)
	if err == nil {
		t.Fatal("checkRelativeKindOrder(text delta inside the tool-call window) = nil, want a violation (S-CTA-006)")
	}
	if !contains(err.Error(), ai.EventKindTextDelta.String()) {
		t.Errorf("error %q does not name %v, the kind that broke the window's tolerance", err, ai.EventKindTextDelta)
	}
}
