// AI-21.3 — a scripted terminal error: any AI-19 failure category, in both
// partial-output states, and terminal exclusivity.
//
// fake_error_test.go proves R-AFP-008/009 from outside package agenttest,
// reusing fake_text_test.go's shared helpers (drainFake, boundedFakeTimeout,
// mustTextStartEvent) and provider_test.go's mustSimpleRequest — same
// agenttest_test package.
package agenttest_test

import (
	"context"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// mustMidStreamFailure builds a *ai.Failure of category, carrying
// outputPreceded — this file's own local builder, mirroring
// ai/provider_test.go's mustMidStreamFailure (an internal test helper of
// another package, not reusable from here).
func mustMidStreamFailure(t *testing.T, category ai.FailureCategory, outputPreceded bool) *ai.Failure {
	t.Helper()

	failure, err := ai.MidStreamFailure(ai.FailureReport{Category: category}, outputPreceded)
	if err != nil {
		t.Fatalf("ai.MidStreamFailure(%v, %v) returned %v, want no failure", category, outputPreceded, err)
	}
	return failure
}

// AI-21.3 item 1 (R-AFP-008, S-AFP-018/019) — every AI-19 failure category
// is scriptable as a terminal error, both with and without prior output,
// and the drained terminal event's partial-output discriminator matches
// the state the script actually produced.
func TestProvider_ScriptedTerminalError_EveryCategory_BothPartialOutputStates(t *testing.T) {
	t.Parallel()

	for _, category := range ai.FailureCategories() {
		t.Run(category.String()+"_with_prior_output", func(t *testing.T) {
			t.Parallel()

			terminal, err := ai.ErrorEvent(mustMidStreamFailure(t, category, true))
			if err != nil {
				t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
			}
			script := agenttest.Script{Steps: []agenttest.Step{
				agenttest.Emit(mustPriorOutputEvent(t)),
				agenttest.Emit(terminal),
			}}
			provider := agenttest.NewProvider(script)
			ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
			if err != nil {
				t.Fatalf("Stream returned %v, want no failure", err)
			}
			got := drainFake(t, ch)
			if len(got) != 2 {
				t.Fatalf("drained %d event(s), want 2 (one prior event + the terminal)", len(got))
			}
			payload, ok := got[1].ErrorPayload()
			if !ok {
				t.Fatal("last event carries no ErrorPayload")
			}
			if payload.Category() != category {
				t.Errorf("category = %v, want %v", payload.Category(), category)
			}
			if !payload.PartialOutput() {
				t.Error("PartialOutput() = false, want true (a text event preceded the terminal)")
			}
		})

		t.Run(category.String()+"_with_no_prior_output", func(t *testing.T) {
			t.Parallel()

			terminal, err := ai.ErrorEvent(mustMidStreamFailure(t, category, false))
			if err != nil {
				t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
			}
			script := agenttest.Script{Steps: []agenttest.Step{agenttest.Emit(terminal)}}
			provider := agenttest.NewProvider(script)
			ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
			if err != nil {
				t.Fatalf("Stream returned %v, want no failure", err)
			}
			got := drainFake(t, ch)
			if len(got) != 1 {
				t.Fatalf("drained %d event(s), want 1 (the terminal, still delivered mid-stream rather than pre-stream)", len(got))
			}
			payload, ok := got[0].ErrorPayload()
			if !ok {
				t.Fatal("event carries no ErrorPayload")
			}
			if payload.Category() != category {
				t.Errorf("category = %v, want %v", payload.Category(), category)
			}
			if payload.PartialOutput() {
				t.Error("PartialOutput() = true, want false (nothing preceded the terminal)")
			}
			if payload.Delivery() != ai.DeliveryMidStream {
				t.Errorf("Delivery() = %v, want %v (still mid-stream, not pre-stream)", payload.Delivery(), ai.DeliveryMidStream)
			}
		})
	}
}

// AI-21.3 item 2 (R-AFP-009, S-AFP-020) — after a scripted terminal error
// the stream closes and nothing follows: the next receive reports a
// closed stream.
func TestProvider_AfterTerminalError_NextReceiveReportsClosedStream(t *testing.T) {
	t.Parallel()

	terminal, err := ai.ErrorEvent(mustMidStreamFailure(t, ai.FailureCategoryUnavailable, false))
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	script := agenttest.Script{Steps: []agenttest.Step{agenttest.Emit(terminal)}}
	provider := agenttest.NewProvider(script)
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}

	first, ok := <-ch
	if !ok {
		t.Fatal("first receive reported closed, want the terminal event")
	}
	if _, isErr := first.ErrorPayload(); !isErr {
		t.Fatal("first event carries no ErrorPayload, want the terminal error event")
	}

	select {
	case _, ok := <-ch:
		if ok {
			t.Error("second receive returned a value, want the stream closed with nothing following the terminal")
		}
	case <-time.After(boundedFakeTimeout):
		t.Fatal("second receive did not report closed within the bounded timeout")
	}
}

// AI-21.3 item 2's second scenario (R-AFP-009, S-AFP-021) — a script that
// places a text-block-start event after a terminal error fails loudly at
// the fake and names the misplaced event, rather than emitting it.
func TestProvider_ScriptWithEventAfterTerminalError_FailsLoudlyAndNamesTheMisplacedEvent(t *testing.T) {
	t.Parallel()

	terminal, err := ai.ErrorEvent(mustMidStreamFailure(t, ai.FailureCategoryUnavailable, false))
	if err != nil {
		t.Fatalf("ai.ErrorEvent returned %v, want no failure", err)
	}
	afterTerminal := mustTextStartEvent(t)
	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(terminal),
		agenttest.Emit(afterTerminal),
	}}
	provider := agenttest.NewProvider(script)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("Stream did not panic for a script with an event after a terminal error, want a loud failure naming the misplaced event (S-AFP-021)")
		}
		msg, ok := r.(string)
		if !ok || msg == "" {
			t.Fatalf("panicked with %#v, want a non-empty message naming the misplaced event", r)
		}
	}()

	_, _ = provider.Stream(context.Background(), mustSimpleRequest(t))
	t.Fatal("unreachable: Stream should have panicked before this point")
}
