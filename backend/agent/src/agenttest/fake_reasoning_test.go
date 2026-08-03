// AI-21.7 — scripted reasoning: deltas, a byte-exact round-trip token, the
// redacted/signature-only shapes, and the wall against text events.
//
// fake_reasoning_test.go proves R-AFP-016…018 from outside package
// agenttest, reusing fake_text_test.go's shared helpers — same
// agenttest_test package.
package agenttest_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// mustReasoningBlockStart builds a non-redacted reasoning-block-start event
// at blockIndex and returns both the event (to script) and its payload (to
// pass into NewReasoningDelta/NewReasoningBlockEnd, whose block index and
// redaction bit are inherited from it).
func mustReasoningBlockStart(t *testing.T, blockIndex uint64) (ai.Event, ai.ReasoningBlockStart) {
	t.Helper()

	ev, err := ai.NewReasoningBlockStart(blockIndex)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockStart returned %v, want no failure", err)
	}
	payload, ok := ev.ReasoningBlockStart()
	if !ok {
		t.Fatal("event carries no ReasoningBlockStart payload")
	}
	return ev, payload
}

// AI-21.7 item 1 (R-AFP-016, S-AFP-038/039) — reasoning deltas and a
// terminal round-trip token round-trip byte-exact, including a token
// containing bytes that are not valid text, unchanged — not normalised,
// escaped or truncated.
func TestProvider_ScriptedReasoning_DeltasAndRoundTripToken_ByteExact(t *testing.T) {
	t.Parallel()

	startEvent, startPayload := mustReasoningBlockStart(t, 1)
	invalidUTF8Token := []byte{0xff, 0xfe, 'a', 0x00, 'b'} // bytes that are not valid text

	delta1, err := ai.NewReasoningDelta(startPayload, []byte("first "))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	delta2, err := ai.NewReasoningDelta(startPayload, []byte("second"))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	endEvent, err := ai.NewReasoningBlockEnd(startPayload, invalidUTF8Token)
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}

	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(startEvent), agenttest.Emit(delta1), agenttest.Emit(delta2), agenttest.Emit(endEvent),
	}}
	provider := agenttest.NewProvider(script)
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	got := drainFake(t, ch)
	if len(got) != 4 {
		t.Fatalf("drained %d event(s), want 4 (start, two deltas, end)", len(got))
	}

	d1, ok := got[1].ReasoningDelta()
	if !ok || d1.Fragment() == nil || string(d1.Fragment()) != "first " {
		t.Errorf("first delta = (%q, ok=%v), want (%q, true)", d1.Fragment(), ok, "first ")
	}
	d2, ok := got[2].ReasoningDelta()
	if !ok || string(d2.Fragment()) != "second" {
		t.Errorf("second delta = (%q, ok=%v), want (%q, true)", d2.Fragment(), ok, "second")
	}

	end, ok := got[3].ReasoningBlockEnd()
	if !ok {
		t.Fatal("last event carries no ReasoningBlockEnd payload")
	}
	token, hasToken := end.Token()
	if !hasToken {
		t.Fatal("Token() reported absent, want present")
	}
	if !bytes.Equal(token, invalidUTF8Token) {
		t.Errorf("token = %v, want %v byte-exact (survives unchanged even though it is not valid text)", token, invalidUTF8Token)
	}
}

// AI-21.7 item 2 (R-AFP-017, S-AFP-040…042) — a redacted reasoning block
// (no visible fragment) and a signature-only block are both scriptable and
// drainable. A visible fragment on a redacted block fails loudly and names
// the violation at the ai package's own reasoning-delta constructor — the
// door every scripted event must pass through before Emit can ever see it
// (design.md: "events come from ai constructors, already validated"), so
// the fake never receives such an event to drive at all.
func TestProvider_ScriptedReasoning_RedactedAndSignatureOnlyShapes(t *testing.T) {
	t.Parallel()

	t.Run("redacted block: no reasoning text, terminal token present", func(t *testing.T) {
		t.Parallel()

		startEvent, err := ai.NewRedactedReasoningBlockStart(1)
		if err != nil {
			t.Fatalf("ai.NewRedactedReasoningBlockStart returned %v, want no failure", err)
		}
		startPayload, ok := startEvent.ReasoningBlockStart()
		if !ok {
			t.Fatal("event carries no ReasoningBlockStart payload")
		}
		endEvent, err := ai.NewReasoningBlockEnd(startPayload, []byte("redacted-token"))
		if err != nil {
			t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
		}
		script := agenttest.Script{Steps: []agenttest.Step{agenttest.Emit(startEvent), agenttest.Emit(endEvent)}}
		provider := agenttest.NewProvider(script)

		ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		got := drainFake(t, ch)
		if len(got) != 2 {
			t.Fatalf("drained %d event(s), want 2 (start, end, no reasoning text between them)", len(got))
		}
		start, ok := got[0].ReasoningBlockStart()
		if !ok || !start.Redacted() {
			t.Errorf("start.Redacted() = %v (ok=%v), want true", start.Redacted(), ok)
		}
	})

	t.Run("signature-only block: signature present, no content fragment", func(t *testing.T) {
		t.Parallel()

		startEvent, startPayload := mustReasoningBlockStart(t, 1)
		endEvent, err := ai.NewReasoningBlockEnd(startPayload, []byte("signature-only"))
		if err != nil {
			t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
		}
		script := agenttest.Script{Steps: []agenttest.Step{agenttest.Emit(startEvent), agenttest.Emit(endEvent)}}
		provider := agenttest.NewProvider(script)

		ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
		if err != nil {
			t.Fatalf("Stream returned %v, want no failure", err)
		}
		got := drainFake(t, ch)
		if len(got) != 2 {
			t.Fatalf("drained %d event(s), want 2", len(got))
		}
		end, ok := got[1].ReasoningBlockEnd()
		if !ok {
			t.Fatal("last event carries no ReasoningBlockEnd payload")
		}
		token, hasToken := end.Token()
		if !hasToken || string(token) != "signature-only" {
			t.Errorf("token = (%q, ok=%v), want (%q, true)", token, hasToken, "signature-only")
		}
	})

	t.Run("a visible fragment inside a redacted block fails loudly, naming the violation", func(t *testing.T) {
		t.Parallel()

		startEvent, err := ai.NewRedactedReasoningBlockStart(1)
		if err != nil {
			t.Fatalf("ai.NewRedactedReasoningBlockStart returned %v, want no failure", err)
		}
		startPayload, ok := startEvent.ReasoningBlockStart()
		if !ok {
			t.Fatal("event carries no ReasoningBlockStart payload")
		}

		_, err = ai.NewReasoningDelta(startPayload, []byte("visible fragment"))
		if err == nil {
			t.Fatal("ai.NewReasoningDelta on a redacted block with a non-empty fragment unexpectedly succeeded, want a loud failure naming the violation (S-AFP-042)")
		}
		if !errors.Is(err, ai.ErrMisplaced) {
			t.Errorf("errors.Is(err, ai.ErrMisplaced) = false on %v, want the misplaced-value rule class", err)
		}
	})
}

// AI-21.7 item 3 (R-AFP-018, S-AFP-043/044) — a stream mixing reasoning and
// text deltas keeps them walled apart in both directions: no scripted
// reasoning fragment or token appears in any collected text event, and no
// scripted text fragment appears in any collected reasoning event.
func TestProvider_MixedReasoningAndText_NeverCrossesIntoTheOtherEventKind(t *testing.T) {
	t.Parallel()

	const reasoningMarker = "REASONING_ONLY_FRAGMENT"
	const textMarker = "TEXT_ONLY_FRAGMENT"

	reasoningStartEvent, reasoningPayload := mustReasoningBlockStart(t, 1)
	reasoningDelta, err := ai.NewReasoningDelta(reasoningPayload, []byte(reasoningMarker))
	if err != nil {
		t.Fatalf("ai.NewReasoningDelta returned %v, want no failure", err)
	}
	reasoningEnd, err := ai.NewReasoningBlockEnd(reasoningPayload, []byte("round-trip-token"))
	if err != nil {
		t.Fatalf("ai.NewReasoningBlockEnd returned %v, want no failure", err)
	}

	textStart, err := ai.NewTextBlockStart(2)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	textDelta, err := ai.NewTextDelta(2, textMarker)
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	textEnd, err := ai.NewTextBlockEnd(2)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}

	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(reasoningStartEvent), agenttest.Emit(textStart),
		agenttest.Emit(reasoningDelta), agenttest.Emit(textDelta),
		agenttest.Emit(reasoningEnd), agenttest.Emit(textEnd),
	}}
	provider := agenttest.NewProvider(script)
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	got := drainFake(t, ch)
	if len(got) != 6 {
		t.Fatalf("drained %d event(s), want 6", len(got))
	}

	var textEvents, reasoningEvents []ai.Event
	for _, ev := range got {
		switch ev.Kind() {
		case ai.EventKindTextBlockStart, ai.EventKindTextDelta, ai.EventKindTextBlockEnd:
			textEvents = append(textEvents, ev)
		case ai.EventKindReasoningBlockStart, ai.EventKindReasoningDelta, ai.EventKindReasoningBlockEnd:
			reasoningEvents = append(reasoningEvents, ev)
		}
	}
	if len(textEvents) != 3 || len(reasoningEvents) != 3 {
		t.Fatalf("collected %d text event(s) and %d reasoning event(s), want 3 and 3", len(textEvents), len(reasoningEvents))
	}

	for _, ev := range textEvents {
		if d, ok := ev.TextDelta(); ok && strings.Contains(d.Delta(), reasoningMarker) {
			t.Errorf("a text event's delta contains the reasoning marker %q, want the wall to hold", reasoningMarker)
		}
	}
	for _, ev := range reasoningEvents {
		if d, ok := ev.ReasoningDelta(); ok && strings.Contains(string(d.Fragment()), textMarker) {
			t.Errorf("a reasoning event's fragment contains the text marker %q, want the wall to hold", textMarker)
		}
		if e, ok := ev.ReasoningBlockEnd(); ok {
			if token, _ := e.Token(); strings.Contains(string(token), textMarker) {
				t.Errorf("a reasoning event's token contains the text marker %q, want the wall to hold", textMarker)
			}
		}
	}
}
