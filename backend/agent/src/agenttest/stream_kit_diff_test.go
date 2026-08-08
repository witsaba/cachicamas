// AI-22.2 — proof for readable, bounded event diffs (R-STK-003, R-STK-004),
// from outside package agenttest, the same agenttest_test package this
// milestone's sibling proof files use.
package agenttest_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// diffWitnessEvents builds one event of every registered ai.EventKind —
// this file's shared fixture for every diff test that needs "one event of
// a given kind" or "one event of every kind" (S-STK-007…012), mirroring
// ai/event_registry_test.go's own witness-construction shape (one
// constructor call per kind, fixed valid arguments).
func diffWitnessEvents(t *testing.T) map[ai.EventKind]ai.Event {
	t.Helper()

	out := make(map[ai.EventKind]ai.Event, len(ai.EventKinds()))

	responseStart, err := ai.NewResponseStart("resp_witness", "model_witness")
	mustNoDiffErr(t, err)
	out[ai.EventKindResponseStart] = responseStart

	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{Input: ai.Tokens(10), Output: ai.Tokens(20)})
	mustNoDiffErr(t, err)
	out[ai.EventKindCompletion] = completion

	reasoningStart, err := ai.NewReasoningBlockStart(1)
	mustNoDiffErr(t, err)
	out[ai.EventKindReasoningBlockStart] = reasoningStart
	reasoningStartPayload, ok := reasoningStart.ReasoningBlockStart()
	if !ok {
		t.Fatal("reasoningStart.ReasoningBlockStart() reported false on an event of its own kind")
	}

	reasoningDelta, err := ai.NewReasoningDelta(reasoningStartPayload, []byte("witness_reasoning_fragment"))
	mustNoDiffErr(t, err)
	out[ai.EventKindReasoningDelta] = reasoningDelta

	reasoningEnd, err := ai.NewReasoningBlockEnd(reasoningStartPayload, []byte("witness_reasoning_token"))
	mustNoDiffErr(t, err)
	out[ai.EventKindReasoningBlockEnd] = reasoningEnd

	textStart, err := ai.NewTextBlockStart(1)
	mustNoDiffErr(t, err)
	out[ai.EventKindTextBlockStart] = textStart

	textDelta, err := ai.NewTextDelta(1, "witness_text_fragment")
	mustNoDiffErr(t, err)
	out[ai.EventKindTextDelta] = textDelta

	textEnd, err := ai.NewTextBlockEnd(1)
	mustNoDiffErr(t, err)
	out[ai.EventKindTextBlockEnd] = textEnd

	toolStart, err := ai.NewToolCallStart(1, "call_witness", "tool_witness")
	mustNoDiffErr(t, err)
	out[ai.EventKindToolCallStart] = toolStart

	toolDelta, err := ai.NewToolCallDelta(1, []byte("witness_tool_fragment"))
	mustNoDiffErr(t, err)
	out[ai.EventKindToolCallDelta] = toolDelta

	toolEnd, err := ai.NewToolCallEnd(1, []byte(`{"witness":true}`))
	mustNoDiffErr(t, err)
	out[ai.EventKindToolCallEnd] = toolEnd

	failure, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryTimeout})
	mustNoDiffErr(t, err)
	errorEvent, err := ai.ErrorEvent(failure)
	mustNoDiffErr(t, err)
	out[ai.EventKindError] = errorEvent

	return out
}

func mustNoDiffErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("witness construction returned %v, want no failure", err)
	}
}

// stampN stamps a fresh, independent per-call ai.Stamper across events in
// order, giving event i sequence i+1 — this file's shared way to build a
// realistic (but not necessarily ai.CheckStream-valid; RequireSameEvents
// does not call it) ordered slice from witness events.
func stampN(events []ai.Event) []ai.Event {
	var s ai.Stamper
	out := make([]ai.Event, len(events))
	for i, ev := range events {
		out[i] = s.Stamp(ev)
	}
	return out
}

// AI-22.2 item 1 (R-STK-003, S-STK-007) — two four-event recordings
// differing only at index 2 produce a failure naming index 2 and both
// sides' kind and sequence, and do not name index 3 as the divergence.
func TestRequireSameEvents_DifferAtIndex2_NamesIndexAndBothSidesKindAndSequence(t *testing.T) {
	t.Parallel()

	w := diffWitnessEvents(t)
	got := stampN([]ai.Event{w[ai.EventKindResponseStart], w[ai.EventKindTextBlockStart], w[ai.EventKindTextDelta], w[ai.EventKindTextBlockEnd]})
	want := stampN([]ai.Event{w[ai.EventKindResponseStart], w[ai.EventKindTextBlockStart], w[ai.EventKindToolCallStart], w[ai.EventKindTextBlockEnd]})

	fake := &fakeTB{}
	agenttest.RequireSameEvents(fake, got, want)

	if !fake.failed {
		t.Fatal("RequireSameEvents did not fail for recordings differing at index 2, want a failure")
	}
	if len(fake.fatal) != 1 {
		t.Fatalf("RequireSameEvents called Fatal/Fatalf %d time(s), want exactly 1", len(fake.fatal))
	}
	msg := fake.lastFatal()
	if !strings.Contains(msg, "index 2") {
		t.Errorf("failure message %q does not name index 2", msg)
	}
	if strings.Contains(msg, "index 3") {
		t.Errorf("failure message %q names index 3, want only the first divergence (index 2)", msg)
	}
	if !strings.Contains(msg, ai.EventKindTextDelta.String()) {
		t.Errorf("failure message %q does not name the got side's kind %v", msg, ai.EventKindTextDelta)
	}
	if !strings.Contains(msg, ai.EventKindToolCallStart.String()) {
		t.Errorf("failure message %q does not name the want side's kind %v", msg, ai.EventKindToolCallStart)
	}
	if !strings.Contains(msg, "seq=3") {
		t.Errorf("failure message %q does not name the sequence (3) shared by both sides at index 2", msg)
	}
}

// AI-22.2 item 1 (R-STK-003, S-STK-008) — two element-wise-equal recordings
// diff cleanly: no failure.
func TestRequireSameEvents_ElementWiseEqual_NoFailure(t *testing.T) {
	t.Parallel()

	w := diffWitnessEvents(t)
	events := []ai.Event{w[ai.EventKindResponseStart], w[ai.EventKindTextBlockStart], w[ai.EventKindTextDelta], w[ai.EventKindTextBlockEnd]}
	got := stampN(events)
	want := stampN(events)

	fake := &fakeTB{}
	agenttest.RequireSameEvents(fake, got, want)

	if fake.failed {
		t.Fatalf("RequireSameEvents failed for element-wise equal recordings, want success: %v", fake.fatal)
	}
}

// AI-22.2 item 1 (R-STK-003, S-STK-009) — a three-event recording and a
// five-event recording sharing their first three events produce a failure
// naming index 3 (where the shorter recording ended) and the two extra
// events' kinds.
func TestRequireSameEvents_LengthMismatch_NamesShorterEndAndExtraKinds(t *testing.T) {
	t.Parallel()

	w := diffWitnessEvents(t)
	shared := []ai.Event{w[ai.EventKindResponseStart], w[ai.EventKindTextBlockStart], w[ai.EventKindTextDelta]}
	got := stampN(shared)
	wantSource := append(append([]ai.Event{}, shared...), w[ai.EventKindTextBlockEnd], w[ai.EventKindCompletion])
	want := stampN(wantSource)

	fake := &fakeTB{}
	agenttest.RequireSameEvents(fake, got, want)

	if !fake.failed {
		t.Fatal("RequireSameEvents did not fail for a length mismatch, want a failure")
	}
	msg := fake.lastFatal()
	if !strings.Contains(msg, "index 3") {
		t.Errorf("failure message %q does not name index 3 (where the shorter recording ended)", msg)
	}
	if !strings.Contains(msg, ai.EventKindTextBlockEnd.String()) {
		t.Errorf("failure message %q does not name the first extra event's kind %v", msg, ai.EventKindTextBlockEnd)
	}
	if !strings.Contains(msg, ai.EventKindCompletion.String()) {
		t.Errorf("failure message %q does not name the second extra event's kind %v", msg, ai.EventKindCompletion)
	}
}

// AI-22.2 item 2 (R-STK-004, S-STK-010) — a payload far exceeding the cap
// renders bounded, carrying the elision marker, never the full payload
// verbatim.
func TestRequireSameEvents_PayloadFarExceedsCap_RenderingBoundedWithElisionMarker(t *testing.T) {
	t.Parallel()

	longFragment := strings.Repeat("A", 200)
	longDelta, err := ai.NewTextDelta(1, longFragment)
	mustNoDiffErr(t, err)
	otherDelta, err := ai.NewTextDelta(1, "short")
	mustNoDiffErr(t, err)

	got := stampN([]ai.Event{longDelta})
	want := stampN([]ai.Event{otherDelta})

	fake := &fakeTB{}
	agenttest.RequireSameEvents(fake, got, want)

	if !fake.failed {
		t.Fatal("RequireSameEvents did not fail for a divergence at index 0, want a failure")
	}
	msg := fake.lastFatal()
	if !strings.Contains(msg, "…") {
		t.Errorf("failure message %q does not carry the elision marker for a payload far exceeding the cap", msg)
	}
	if strings.Contains(msg, longFragment) {
		t.Errorf("failure message %q renders the full 200-byte payload verbatim, want it capped", msg)
	}
	wantPrefix := longFragment[:32]
	if !strings.Contains(msg, wantPrefix) {
		t.Errorf("failure message %q does not carry the payload's first 32 runes %q", msg, wantPrefix)
	}
	if !strings.Contains(msg, "len=200") {
		t.Errorf("failure message %q does not report the payload's true length (200)", msg)
	}
}

// AI-22.2 item 2 (R-STK-004, S-STK-011/012) — every registered kind
// produces a non-empty, kind-naming rendering that is NOT the generic,
// payload-free event(kind seq=N) fallback — proof that a specific,
// registered summary function ran for each of the 12 kinds, not a silent
// fallback that would equally "name the kind" without actually summarising
// anything (the exhaustiveness property S-STK-012 describes).
func TestRequireSameEvents_EveryRegisteredKind_UsesASpecificSummaryNotTheGenericFallback(t *testing.T) {
	t.Parallel()

	w := diffWitnessEvents(t)
	if len(w) != len(ai.EventKinds()) {
		t.Fatalf("diffWitnessEvents() carries %d witness(es), want %d (one per ai.EventKinds())", len(w), len(ai.EventKinds()))
	}

	for _, kind := range ai.EventKinds() {
		t.Run(kind.String(), func(t *testing.T) {
			t.Parallel()

			ev, ok := w[kind]
			if !ok {
				t.Fatalf("diffWitnessEvents() carries no witness for %v", kind)
			}

			// Two independently-stamped copies of the SAME witness event
			// diverge only by sequence — enough to force RequireSameEvents
			// to report a divergence and render both sides' summaries,
			// without changing kind.
			got := stampN([]ai.Event{ev})
			var skip ai.Stamper
			skip.Stamp(ev) // advance a fresh stamper past sequence 1
			want := []ai.Event{skip.Stamp(ev)}

			fake := &fakeTB{}
			agenttest.RequireSameEvents(fake, got, want)

			if !fake.failed {
				t.Fatalf("RequireSameEvents did not fail for two %v events at different sequences, want a failure", kind)
			}
			msg := fake.lastFatal()
			if !strings.Contains(msg, kind.String()) {
				t.Errorf("failure message %q does not name kind %v", msg, kind)
			}
			// A kind with no summaryTable entry falls back to its bare kind
			// name followed directly by " (seq=" (this file's fakeTB
			// appends the sequence itself). A kind WITH a registered,
			// payload-aware summary always renders more than that: the
			// kind name is immediately followed by "(" opening its
			// structural/payload fields, never by a bare space. This is
			// the exhaustiveness signature (S-STK-012): its presence means
			// this kind silently fell back rather than being summarised.
			genericFallback := fmt.Sprintf("%s (seq=", kind.String())
			if strings.Contains(msg, genericFallback) {
				t.Errorf("failure message %q renders %v as the bare, payload-free fallback %q, want a kind-specific, payload-aware summary (exhaustiveness, S-STK-012)", msg, kind, genericFallback)
			}
		})
	}
}

// AI-22.2 item 2 (R-STK-004, S-STK-012 partial) — an event of an
// unregistered kind (the zero Event, Kind()==0, which is never a member of
// ai.EventKinds() and therefore never a key in this file's summary table)
// still renders distinctly rather than blank, falling back to Event's own
// documented "unset" naming instead of silently producing an empty string.
func TestRequireSameEvents_UnregisteredKind_RendersDistinctlyNotBlank(t *testing.T) {
	t.Parallel()

	w := diffWitnessEvents(t)
	got := stampN([]ai.Event{{}}) // zero Event: Kind() == 0
	want := stampN([]ai.Event{w[ai.EventKindCompletion]})

	fake := &fakeTB{}
	agenttest.RequireSameEvents(fake, got, want)

	if !fake.failed {
		t.Fatal("RequireSameEvents did not fail comparing a zero Event against a completion event, want a failure")
	}
	msg := fake.lastFatal()
	if !strings.Contains(msg, "unset") {
		t.Errorf("failure message %q does not distinctly name the unregistered/zero kind as %q (ai.Event's own zero-value name)", msg, "unset")
	}
}

// AI-36 (WU-4) — bounded-summary hygiene: the divergence report's own
// failure output is proven sentinel-free, with a positive control
// (R-STK-014, S-STK-047…049).
//
// overBoundSentinelCredential and overBoundSentinelContent are both
// deliberately longer than summaryRuneCap (32 runes, stream_kit_diff.go)
// so a report that reconstructs either in full — rather than only a
// bounded prefix — is unambiguously a leak, never a coincidental
// substring collision with legitimate short content.
const overBoundSentinelCredential = "AI36-STK-CREDENTIAL-SENTINEL-0123456789-abcdefghij"
const overBoundSentinelContent = "AI36-STK-CONTENT-BODY-SENTINEL-9876543210-zyxwvutsr"

// AI-36 item 2 (R-STK-014, S-STK-047) — two recordings diverging in a
// payload carrying a planted sentinel credential (a TextDelta fragment)
// and, separately, a planted sentinel content body (a ToolCallDelta
// fragment — a different free-form field, proving the property is not
// specific to one kind) never reconstruct either sentinel in full; the
// report still names the first diverging position and kind.
func TestRequireSameEvents_DivergenceReport_SentinelFree(t *testing.T) {
	t.Parallel()

	if len(overBoundSentinelCredential) <= 32 || len(overBoundSentinelContent) <= 32 {
		t.Fatal("test fixture error: both sentinels must exceed the 32-rune bound (summaryRuneCap)")
	}

	t.Run("credential sentinel via a TextDelta fragment", func(t *testing.T) {
		t.Parallel()

		credentialDelta, err := ai.NewTextDelta(1, overBoundSentinelCredential)
		mustNoDiffErr(t, err)
		otherDelta, err := ai.NewTextDelta(1, "short, clean text")
		mustNoDiffErr(t, err)

		got := stampN([]ai.Event{credentialDelta})
		want := stampN([]ai.Event{otherDelta})

		fake := &fakeTB{}
		agenttest.RequireSameEvents(fake, got, want)

		if !fake.failed {
			t.Fatal("RequireSameEvents did not fail for a divergence carrying the planted sentinel, want a failure")
		}
		msg := fake.lastFatal()
		if strings.Contains(msg, overBoundSentinelCredential) {
			t.Errorf("divergence report %q reconstructs the full planted sentinel credential, want only a bounded fragment (S-STK-047)", msg)
		}
		if !strings.Contains(msg, "index 0") {
			t.Errorf("divergence report %q does not name the first diverging position (index 0)", msg)
		}
		if !strings.Contains(msg, ai.EventKindTextDelta.String()) {
			t.Errorf("divergence report %q does not name the diverging kind %v", msg, ai.EventKindTextDelta)
		}
	})

	t.Run("content-body sentinel via a ToolCallDelta fragment", func(t *testing.T) {
		t.Parallel()

		toolDelta, err := ai.NewToolCallDelta(1, []byte(overBoundSentinelContent))
		mustNoDiffErr(t, err)
		otherToolDelta, err := ai.NewToolCallDelta(1, []byte("{}"))
		mustNoDiffErr(t, err)

		got := stampN([]ai.Event{toolDelta})
		want := stampN([]ai.Event{otherToolDelta})

		fake := &fakeTB{}
		agenttest.RequireSameEvents(fake, got, want)

		if !fake.failed {
			t.Fatal("RequireSameEvents did not fail for a divergence carrying the planted content sentinel, want a failure")
		}
		msg := fake.lastFatal()
		if strings.Contains(msg, overBoundSentinelContent) {
			t.Errorf("divergence report %q reconstructs the full planted content-body sentinel, want only a bounded fragment (S-STK-047)", msg)
		}
	})
}

// AI-36 item 3 (R-STK-014, S-STK-048) — the paired positive control: a
// sentinel SHORT ENOUGH to fit within the bound (never truncated) IS
// reproduced in full by the same divergence-report mechanism — proving
// TestRequireSameEvents_DivergenceReport_SentinelFree's absence claim is
// falsifiable, not vacuous. This is the mechanism itself demonstrating a
// real bite (an under-bound fragment renders verbatim, by construction —
// boundedFragment only bounds length, it does not redact content), not a
// hand-built fake report.
func TestRequireSameEvents_DivergenceReport_PositiveControl(t *testing.T) {
	t.Parallel()

	const underBoundSentinel = "short-sentinel-19chars"
	if len([]rune(underBoundSentinel)) > 32 {
		t.Fatal("test fixture error: this sentinel must fit within the 32-rune bound to prove the check can bite")
	}

	delta, err := ai.NewTextDelta(1, underBoundSentinel)
	mustNoDiffErr(t, err)
	other, err := ai.NewTextDelta(1, "different")
	mustNoDiffErr(t, err)

	got := stampN([]ai.Event{delta})
	want := stampN([]ai.Event{other})

	fake := &fakeTB{}
	agenttest.RequireSameEvents(fake, got, want)

	if !fake.failed {
		t.Fatal("RequireSameEvents did not fail for this divergence, want a failure to inspect")
	}
	msg := fake.lastFatal()
	if !strings.Contains(msg, underBoundSentinel) {
		t.Fatalf("divergence report %q does NOT reproduce an under-bound sentinel, want it to — a check whose absence claim can never fail is not a proof (S-STK-048)", msg)
	}
	if !strings.Contains(msg, ai.EventKindTextDelta.String()) {
		t.Errorf("divergence report %q does not name the diverging vector/kind %v", msg, ai.EventKindTextDelta)
	}
}

// AI-36 item 4 (R-STK-014, S-STK-049) — the bound holds for every event
// kind carrying a plantable string-shaped field: no kind may render its
// over-bound payload in full, and the planted sentinel must never be
// reconstructible from the report. Kinds with no string-shaped field at
// all (Completion, ReasoningBlockStart, TextBlockStart, TextBlockEnd)
// have nothing to plant a sentinel into and are outside this table by
// construction, not by omission.
func TestRequireSameEvents_BoundHoldsForEveryEventKind(t *testing.T) {
	t.Parallel()

	const sentinel = "AI36-STK-049-EVERY-KIND-OVERBOUND-SENTINEL-abcdefghij"
	if len([]rune(sentinel)) <= 32 {
		t.Fatal("test fixture error: sentinel must exceed the 32-rune bound (summaryRuneCap)")
	}

	reasoningStart, err := ai.NewReasoningBlockStart(1)
	mustNoDiffErr(t, err)
	reasoningStartPayload, ok := reasoningStart.ReasoningBlockStart()
	if !ok {
		t.Fatal("reasoningStart.ReasoningBlockStart() reported false on an event of its own kind")
	}

	cases := []struct {
		name string
		ev   func(t *testing.T) ai.Event
	}{
		{"ResponseStart/id", func(t *testing.T) ai.Event {
			ev, err := ai.NewResponseStart(sentinel, "witness_model")
			mustNoDiffErr(t, err)
			return ev
		}},
		{"ResponseStart/model", func(t *testing.T) ai.Event {
			ev, err := ai.NewResponseStart("witness_id", sentinel)
			mustNoDiffErr(t, err)
			return ev
		}},
		{"ReasoningDelta", func(t *testing.T) ai.Event {
			ev, err := ai.NewReasoningDelta(reasoningStartPayload, []byte(sentinel))
			mustNoDiffErr(t, err)
			return ev
		}},
		{"ReasoningBlockEnd/token", func(t *testing.T) ai.Event {
			ev, err := ai.NewReasoningBlockEnd(reasoningStartPayload, []byte(sentinel))
			mustNoDiffErr(t, err)
			return ev
		}},
		{"TextDelta", func(t *testing.T) ai.Event {
			ev, err := ai.NewTextDelta(1, sentinel)
			mustNoDiffErr(t, err)
			return ev
		}},
		{"ToolCallStart/id", func(t *testing.T) ai.Event {
			ev, err := ai.NewToolCallStart(1, sentinel, "witness_name")
			mustNoDiffErr(t, err)
			return ev
		}},
		{"ToolCallStart/name", func(t *testing.T) ai.Event {
			ev, err := ai.NewToolCallStart(1, "witness_id", sentinel)
			mustNoDiffErr(t, err)
			return ev
		}},
		{"ToolCallDelta", func(t *testing.T) ai.Event {
			ev, err := ai.NewToolCallDelta(1, []byte(sentinel))
			mustNoDiffErr(t, err)
			return ev
		}},
		{"ToolCallEnd", func(t *testing.T) ai.Event {
			ev, err := ai.NewToolCallEnd(1, []byte(`{"sentinel":"`+sentinel+`"}`))
			mustNoDiffErr(t, err)
			return ev
		}},
		{"Error/RawLabel", func(t *testing.T) ai.Event {
			f, err := ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnknown, RawLabel: sentinel})
			mustNoDiffErr(t, err)
			ev, err := ai.ErrorEvent(f)
			mustNoDiffErr(t, err)
			return ev
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := diffWitnessEvents(t)
			leaking := tc.ev(t)
			clean, ok := w[leaking.Kind()]
			if !ok {
				t.Fatalf("no clean witness event registered for kind %v", leaking.Kind())
			}

			got := stampN([]ai.Event{leaking})
			want := stampN([]ai.Event{clean})

			fake := &fakeTB{}
			agenttest.RequireSameEvents(fake, got, want)
			if !fake.failed {
				t.Fatal("RequireSameEvents did not fail for this divergence, want a failure")
			}
			msg := fake.lastFatal()
			if strings.Contains(msg, sentinel) {
				t.Errorf("divergence report for kind %v reconstructs the full over-bound sentinel, want only a bounded summary (S-STK-049): %q", leaking.Kind(), msg)
			}
		})
	}
}

// AI-22.2 item 3 (NFR-STK-E, S-STK-044 partial) — two empty recordings and
// two one-element recordings never panic RequireSameEvents; a failure, when
// one occurs, is attributable rather than a crash.
func TestRequireSameEvents_ExtremeInputs_NeverPanic(t *testing.T) {
	t.Parallel()

	w := diffWitnessEvents(t)

	t.Run("both empty: no failure", func(t *testing.T) {
		t.Parallel()

		fake := &fakeTB{}
		agenttest.RequireSameEvents(fake, []ai.Event{}, []ai.Event{})
		if fake.failed {
			t.Fatalf("RequireSameEvents failed for two empty recordings, want success: %v", fake.fatal)
		}
	})

	t.Run("got empty, want one element: attributable failure naming index 0", func(t *testing.T) {
		t.Parallel()

		fake := &fakeTB{}
		agenttest.RequireSameEvents(fake, []ai.Event{}, stampN([]ai.Event{w[ai.EventKindCompletion]}))
		if !fake.failed {
			t.Fatal("RequireSameEvents did not fail for an empty got against a one-element want, want a failure")
		}
		if msg := fake.lastFatal(); !strings.Contains(msg, "index 0") {
			t.Errorf("failure message %q does not name index 0", msg)
		}
	})

	t.Run("both one element, equal: no failure", func(t *testing.T) {
		t.Parallel()

		events := []ai.Event{w[ai.EventKindTextBlockStart]}
		fake := &fakeTB{}
		agenttest.RequireSameEvents(fake, stampN(events), stampN(events))
		if fake.failed {
			t.Fatalf("RequireSameEvents failed for two equal one-element recordings, want success: %v", fake.fatal)
		}
	})

	t.Run("both one element, differing: attributable failure naming index 0", func(t *testing.T) {
		t.Parallel()

		fake := &fakeTB{}
		got := stampN([]ai.Event{w[ai.EventKindTextBlockStart]})
		want := stampN([]ai.Event{w[ai.EventKindCompletion]})
		agenttest.RequireSameEvents(fake, got, want)
		if !fake.failed {
			t.Fatal("RequireSameEvents did not fail for two differing one-element recordings, want a failure")
		}
		if msg := fake.lastFatal(); !strings.Contains(msg, "index 0") {
			t.Errorf("failure message %q does not name index 0", msg)
		}
	})
}
