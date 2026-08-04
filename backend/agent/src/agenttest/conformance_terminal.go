// AI-23.4 — terminal and error conformance cases: exactly one terminal on
// every path, the partial-output discriminator always answerable, and all
// nine failure categories iterated against AI-19's own shipped enumerator
// (R-CNF-009, R-CNF-010).
//
// "Exactly one terminal, nothing follows" is AI-03 §5's own stream-lifecycle
// obligation (V-PRV-05) — not one of the eight capabilities — so its case
// registers under CapNone (S-CNF-021/023 default to required with no record
// entry, mirroring AI-23.1's redaction precedent). The partial-output
// discriminator and the nine-category exhaustiveness loop are both
// CAP-R-05's own content ("says whether content preceded it", "every
// failure is classifiable through one vocabulary"), so those two cases
// register under CapTypedFailures.

package agenttest

import (
	"context"
	"fmt"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

func init() {
	registerConformanceCase("terminal/exactly_one_terminal_every_path", CapNone, terminalExactlyOneCase)
	registerConformanceCase("terminal/partial_output_discriminator_both_states", CapTypedFailures, terminalDiscriminatorCase)
	registerConformanceCase("terminal/all_nine_failure_categories_exhaustive", CapTypedFailures, terminalFailureCategoryExhaustivenessCase)
}

// terminalExactlyOneCase proves R-CNF-009's exactly-one-terminal half
// (S-CNF-021): a normal finish, a pre-stream failure and a mid-stream
// failure each carry exactly one terminal outcome, with nothing following
// it. Ordering is delegated to RequireValidStream for the two paths that
// return a carrier; the pre-stream path has no carrier at all, so "exactly
// one terminal" there is the caller-visible fact that the one failure
// value returned directly is the whole outcome.
func terminalExactlyOneCase(t *testing.T, f Factory) {
	t.Helper()

	t.Run("normal_finish", func(t *testing.T) {
		responseStart, err := ai.NewResponseStart("resp-normal-finish", "model-normal-finish")
		requireConstructed(t, err, "ai.NewResponseStart")
		completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
		requireConstructed(t, err, "ai.NewCompletion")
		script := Script{Steps: []Step{Emit(responseStart), Emit(completion)}}
		subject := f.New(t, script)
		ch, err := subject.Stream(t.Context(), minimalRequest(t))
		if err != nil {
			t.Fatalf("agenttest: terminal/exactly_one_terminal_every_path (normal): Stream returned %v, want no failure", err)
		}
		rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
		RequireValidStream(t, rec) // ai.CheckStream's own terminal-exclusivity rule, delegated (R-AEE-018)
		// S-CLA-008: exactly two, behind the lifecycle prefix.
		requireDrainedKinds(t, rec.Events(), []ai.EventKind{ai.EventKindResponseStart, ai.EventKindCompletion})
	})

	t.Run("pre_stream_failure", func(t *testing.T) {
		script := Script{Steps: []Step{Emit(mustEmptyTextScript(t))}}
		subject := f.New(t, script) // never consumed: cancellation is checked before the script queue
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		ch, err := subject.Stream(ctx, minimalRequest(t))
		if ch != nil {
			t.Error("Stream returned a non-nil carrier for a pre-stream failure, want none (V-FAIL-11)")
		}
		if err == nil {
			t.Fatal("Stream returned a nil error for an already-cancelled context, want the sole failure outcome")
		}
	})

	t.Run("mid_stream_failure", func(t *testing.T) {
		responseStart, err := ai.NewResponseStart("resp-mid-stream-failure", "model-mid-stream-failure")
		requireConstructed(t, err, "ai.NewResponseStart")
		failure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
		requireConstructed(t, err, "ai.MidStreamFailure")
		terminal, err := ai.ErrorEvent(failure)
		requireConstructed(t, err, "ai.ErrorEvent")
		script := Script{Steps: []Step{Emit(responseStart), Emit(terminal)}}
		subject := f.New(t, script)
		ch, err := subject.Stream(t.Context(), minimalRequest(t))
		if err != nil {
			t.Fatalf("agenttest: terminal/exactly_one_terminal_every_path (mid-stream): Stream returned %v, want no failure", err)
		}
		rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
		RequireValidStream(t, rec)
		requireDrainedKinds(t, rec.Events(), []ai.EventKind{ai.EventKindResponseStart, ai.EventKindError}) // S-CLA-009: exactly two — the error terminal IS this path's terminal, no completion added
	})
}

// mustEmptyTextScript builds one minimal, harmless step for a script the
// pre-stream-failure subtest never actually consumes (cancellation is
// checked before the script queue is touched at all) — a Script must still
// carry at least a validly-constructed event, per Emit's own rule.
func mustEmptyTextScript(t *testing.T) ai.Event {
	t.Helper()
	ev, err := ai.NewTextBlockStart(1)
	requireConstructed(t, err, "ai.NewTextBlockStart")
	return ev
}

// terminalDiscriminatorCase proves R-CNF-009's discriminator half
// (S-CNF-022, CAP-R-05): a failure raised after content reports content
// preceded it, and a failure raised before any event reports none.
func terminalDiscriminatorCase(t *testing.T, f Factory) {
	t.Helper()

	t.Run("content_preceded", func(t *testing.T) {
		start, err := ai.NewTextBlockStart(1)
		requireConstructed(t, err, "ai.NewTextBlockStart")
		delta, err := ai.NewTextDelta(1, "partial")
		requireConstructed(t, err, "ai.NewTextDelta")
		end, err := ai.NewTextBlockEnd(1)
		requireConstructed(t, err, "ai.NewTextBlockEnd") // a block must be properly closed before a terminal — an open block left unterminated is itself a CheckStream violation (R-AEE-016), independent of the terminal that follows it
		failure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout}, true)
		requireConstructed(t, err, "ai.MidStreamFailure")
		terminal, err := ai.ErrorEvent(failure)
		requireConstructed(t, err, "ai.ErrorEvent")

		script := Script{Steps: []Step{Emit(start), Emit(delta), Emit(end), Emit(terminal)}}
		subject := f.New(t, script)
		ch, err := subject.Stream(t.Context(), minimalRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
		RequireValidStream(t, rec)
		events := rec.Events()
		payload, ok := events[len(events)-1].ErrorPayload()
		if !ok {
			t.Fatal("last event carries no ErrorPayload")
		}
		if !payload.PartialOutput() {
			t.Error("PartialOutput() = false, want true — two text events preceded the terminal (S-CNF-022)")
		}
	})

	t.Run("none_preceded", func(t *testing.T) {
		failure, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout}, false)
		requireConstructed(t, err, "ai.MidStreamFailure")
		terminal, err := ai.ErrorEvent(failure)
		requireConstructed(t, err, "ai.ErrorEvent")

		script := Script{Steps: []Step{Emit(terminal)}}
		subject := f.New(t, script)
		ch, err := subject.Stream(t.Context(), minimalRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
		RequireValidStream(t, rec)
		events := rec.Events()
		payload, ok := events[0].ErrorPayload()
		if !ok {
			t.Fatal("first event carries no ErrorPayload")
		}
		if payload.PartialOutput() {
			t.Error("PartialOutput() = true, want false — nothing preceded the terminal (S-CNF-022)")
		}
	})
}

// requireFailureCategoryCoverage reports the first category
// ai.FailureCategories() enumerates that exercised does not mark true, or
// nil when every one is covered. Pure — no testing.TB involved — so this
// package's own tests can prove the exhaustiveness check itself would
// catch a future upstream addition (S-CNF-025) by passing an artificially
// incomplete map, without needing (and being unable) to construct a real
// tenth category.
func requireFailureCategoryCoverage(exercised map[ai.FailureCategory]bool) error {
	for _, c := range ai.FailureCategories() {
		if !exercised[c] {
			return fmt.Errorf("agenttest: failure category %v was not exercised on both delivery paths — exhaustiveness check failed (R-CNF-010)", c)
		}
	}
	return nil
}

// terminalFailureCategoryExhaustivenessCase proves R-CNF-010 (CAP-R-05):
// every member ai.FailureCategories() enumerates — never a hand-written
// list — is expressible and classifiable on both delivery paths
// (S-CNF-024). requireFailureCategoryCoverage's mechanical check, not
// reviewer vigilance, is what would catch a future upstream addition.
func terminalFailureCategoryExhaustivenessCase(t *testing.T, f Factory) {
	t.Helper()

	exercised := make(map[ai.FailureCategory]bool)
	for _, category := range ai.FailureCategories() {
		midOK := t.Run(category.String()+"/mid_stream", func(t *testing.T) {
			failure, err := ai.MidStreamFailure(ai.FailureReport{Category: category}, false)
			requireConstructed(t, err, "ai.MidStreamFailure")
			terminal, err := ai.ErrorEvent(failure)
			requireConstructed(t, err, "ai.ErrorEvent")
			script := Script{Steps: []Step{Emit(terminal)}}
			subject := f.New(t, script)
			ch, err := subject.Stream(t.Context(), minimalRequest(t))
			if err != nil {
				t.Fatalf("Stream returned %v, want no failure", err)
			}
			rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
			RequireValidStream(t, rec)
			events := rec.Events()
			payload, ok := events[len(events)-1].ErrorPayload()
			if !ok || payload.Category() != category {
				t.Errorf("category = %v (ok=%v), want %v (S-CNF-024)", payload.Category(), ok, category)
			}
		})

		preOK := t.Run(category.String()+"/pre_stream", func(t *testing.T) {
			failure, err := ai.PreStreamFailure(ai.FailureReport{Category: category})
			requireConstructed(t, err, "ai.PreStreamFailure")
			if failure.Category() != category {
				t.Errorf("category = %v, want %v (S-CNF-024)", failure.Category(), category)
			}
			if failure.Delivery() != ai.DeliveryPreStream {
				t.Errorf("delivery = %v, want %v", failure.Delivery(), ai.DeliveryPreStream)
			}
		})

		if midOK && preOK {
			exercised[category] = true
		}
	}

	if err := requireFailureCategoryCoverage(exercised); err != nil {
		t.Error(err)
	}
}
