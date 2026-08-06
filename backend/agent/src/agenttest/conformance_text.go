// AI-23.2 — text and lifecycle conformance cases: ordering, contiguity,
// byte-exact reconstruction across a split multi-byte rune, and the
// legality of an empty completion (R-CNF-005, R-CNF-006).
//
// Both cases delegate their mechanics entirely to AI-22's kit —
// DrainAndRecord and RequireValidStream, which is itself ai.CheckStream
// (ordering) plus CheckContiguity (sequencing) — and to ai's own event
// constructors. Neither ordering nor contiguity is re-derived here
// (NFR-CNF-D).

package agenttest

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

func init() {
	registerConformanceCase("text/order_contiguity_byte_exact_reconstruction", CapStreamingText, textOrderingCase)
	registerConformanceCase("text/empty_completion_is_legal", CapStreamingText, textEmptyCompletionCase)
}

// textOrderingCase proves R-CNF-005: a text interaction produces
// block-start/delta/block-end in order (S-CNF-012), sequence numbers run
// 1..N contiguously (S-CNF-012), and a multi-byte rune deliberately split
// across two adjacent deltas reconstructs byte-exactly (S-CNF-013).
func textOrderingCase(t *testing.T, f Factory) {
	t.Helper()

	responseStart, err := ai.NewResponseStart("resp-order-contiguity", "model-order-contiguity")
	requireConstructed(t, err, "ai.NewResponseStart")
	start, err := ai.NewTextBlockStart(1)
	requireConstructed(t, err, "ai.NewTextBlockStart")
	// "café shop" with the 2-byte UTF-8 encoding of 'é' (0xC3 0xA9) split
	// across delta1's tail and delta2's head — S-CNF-013's own construction.
	delta1, err := ai.NewTextDelta(1, "caf\xc3")
	requireConstructed(t, err, "ai.NewTextDelta")
	delta2, err := ai.NewTextDelta(1, "\xa9 shop")
	requireConstructed(t, err, "ai.NewTextDelta")
	end, err := ai.NewTextBlockEnd(1)
	requireConstructed(t, err, "ai.NewTextBlockEnd")
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	requireConstructed(t, err, "ai.NewCompletion")

	script := Script{Steps: []Step{Emit(responseStart), Emit(start), Emit(delta1), Emit(delta2), Emit(end), Emit(completion)}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("agenttest: text/order_contiguity_byte_exact_reconstruction: Stream returned %v, want no failure (R-CNF-005)", err)
	}

	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	RequireValidStream(t, rec) // ai.CheckStream ordering + CheckContiguity sequencing, both delegated (R-STK-005, R-STK-006)

	events := rec.Events()
	wantKinds := []ai.EventKind{ai.EventKindResponseStart, ai.EventKindTextBlockStart, ai.EventKindTextDelta, ai.EventKindTextDelta, ai.EventKindTextBlockEnd, ai.EventKindCompletion}
	requireDrainedKinds(t, events, wantKinds) // exactly six: the shipped four lacked both the prefix and the terminal (S-CLA-001, R-CNF-019)
	if err := checkLifecyclePrefix(events, "resp-order-contiguity", "model-order-contiguity"); err != nil {
		t.Error(err) // S-CLA-002: identity equality, not mere presence
	}

	d1, ok := events[2].TextDelta()
	if !ok {
		t.Fatal("event 2 carries no TextDelta payload")
	}
	d2, ok := events[3].TextDelta()
	if !ok {
		t.Fatal("event 3 carries no TextDelta payload")
	}
	const wantText = "caf\xc3\xa9 shop"
	if reconstructed := d1.Delta() + d2.Delta(); reconstructed != wantText {
		t.Errorf("reconstructed text = %q, want %q byte-exact across the split rune (S-CNF-013)", reconstructed, wantText)
	}
}

// textEmptyCompletionCase proves R-CNF-006: a normally-finished interaction
// producing no text content is conformant — it terminates normally,
// carries its terminal, and is neither reported as a failure, an
// incomplete stream, nor a contiguity violation (S-CNF-015, S-CNF-016). No
// minimum event count is enforced anywhere in this case.
func textEmptyCompletionCase(t *testing.T, f Factory) {
	t.Helper()

	responseStart, err := ai.NewResponseStart("resp-empty-completion", "model-empty-completion")
	requireConstructed(t, err, "ai.NewResponseStart")
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	requireConstructed(t, err, "ai.NewCompletion")

	script := Script{Steps: []Step{Emit(responseStart), Emit(completion)}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("agenttest: text/empty_completion_is_legal: Stream returned %v, want no failure (R-CNF-006)", err)
	}

	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	RequireValidStream(t, rec) // no contiguity violation is reported for the absent text block (S-CNF-016)

	events := rec.Events()
	requireDrainedKinds(t, events, []ai.EventKind{ai.EventKindResponseStart, ai.EventKindCompletion}) // S-CLA-003: exactly two — no minimum text-content count enforced (R-CNF-006)
	if err := checkLifecyclePrefix(events, "resp-empty-completion", "model-empty-completion"); err != nil {
		t.Error(err) // S-CLA-004: identity equality
	}
	if _, ok := events[1].Completion(); !ok {
		t.Fatal("event 1 is not a Completion, want the terminal present and this case to pass (S-CNF-015)")
	}
}
