// AI-23.3 — tool-call conformance cases: fragmented/interleaved calls,
// zero-delta whole calls, observable ordinals, and mixed text+tool content
// ending on the tool-call finish reason (R-CNF-007, R-CNF-008).
//
// reconstructToolCalls is this leaf's one fragment-attribution function,
// shared by every case below — text_events.go's and tool_call_event.go's
// own posture restated: this package ships no accumulator, so
// reconstruction from a recorded stream is the suite's own, test-local
// logic (doc 0001 § 4.3 invariant 1), mirroring AI-21's own
// fake_tool_call_test.go precedent (unreachable from this package).

package agenttest

import (
	"bytes"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

func init() {
	registerConformanceCase("tool_call/fragmented_interleaved_reconstructs_exactly", CapToolCalls, toolCallInterleavedCase)
	registerConformanceCase("tool_call/zero_delta_whole_call_accepted", CapToolCalls, toolCallZeroDeltaCase)
	registerConformanceCase("tool_call/ordinal_distinguishes_same_tool_name", CapToolCalls, toolCallOrdinalCase)
	registerConformanceCase("tool_call/mixed_text_and_tool_ends_on_tool_call_finish_reason", CapToolCalls, mixedTextAndToolCallCase)
}

// reconstructedCall is one tool call rebuilt from a recorded stream's
// start/delta(s)/end events at one block index.
type reconstructedCall struct {
	id, name         string
	arguments        []byte
	fromDeltas       []byte
	ordinal          int // 1-based position among the stream's tool-call-start events, in order (V-STR-21) — a distinct axis from block index
	sawStart, sawEnd bool
}

// reconstructToolCalls walks events once and rebuilds every tool call it
// finds, keyed by block index. A call's ordinal is assigned by the order
// its start event was encountered — never by block index, which is a
// separate, unrelated identity axis (R-ATC-012: the ordinal is derived
// stream-side, never shipped as an accumulator by ai itself).
func reconstructToolCalls(events []ai.Event) map[ai.BlockIndex]*reconstructedCall {
	out := make(map[ai.BlockIndex]*reconstructedCall)
	nextOrdinal := 1
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindToolCallStart:
			s, ok := ev.ToolCallStart()
			if !ok {
				continue
			}
			out[s.BlockIndex()] = &reconstructedCall{id: s.ID(), name: s.Name(), sawStart: true, ordinal: nextOrdinal}
			nextOrdinal++
		case ai.EventKindToolCallDelta:
			d, ok := ev.ToolCallDelta()
			if !ok {
				continue
			}
			if call, exists := out[d.BlockIndex()]; exists {
				call.fromDeltas = append(call.fromDeltas, d.Fragment()...)
			}
		case ai.EventKindToolCallEnd:
			e, ok := ev.ToolCallEnd()
			if !ok {
				continue
			}
			if call, exists := out[e.BlockIndex()]; exists {
				call.arguments = e.Arguments()
				call.sawEnd = true
			}
		}
	}
	return out
}

// toolCallInterleavedCase proves R-CNF-007's fragmented-interleave half
// (S-CNF-017): two concurrent tool calls whose argument fragments
// interleave both reconstruct to their exact source bytes, with no
// fragment misattributed to the other call.
func toolCallInterleavedCase(t *testing.T, f Factory) {
	t.Helper()

	start1, err := ai.NewToolCallStart(1, "call-1", "search")
	requireConstructed(t, err, "ai.NewToolCallStart")
	start2, err := ai.NewToolCallStart(2, "call-2", "weather")
	requireConstructed(t, err, "ai.NewToolCallStart")
	delta1a, err := ai.NewToolCallDelta(1, []byte(`{"q":`))
	requireConstructed(t, err, "ai.NewToolCallDelta")
	delta2a, err := ai.NewToolCallDelta(2, []byte(`{"city":`))
	requireConstructed(t, err, "ai.NewToolCallDelta")
	delta1b, err := ai.NewToolCallDelta(1, []byte(`"a"}`))
	requireConstructed(t, err, "ai.NewToolCallDelta")
	delta2b, err := ai.NewToolCallDelta(2, []byte(`"b"}`))
	requireConstructed(t, err, "ai.NewToolCallDelta")
	end1, err := ai.NewToolCallEnd(1, []byte(`{"q":"a"}`))
	requireConstructed(t, err, "ai.NewToolCallEnd")
	end2, err := ai.NewToolCallEnd(2, []byte(`{"city":"b"}`))
	requireConstructed(t, err, "ai.NewToolCallEnd")

	script := Script{Steps: []Step{
		Emit(start1), Emit(start2),
		Emit(delta1a), Emit(delta2a),
		Emit(delta1b), Emit(delta2b),
		Emit(end1), Emit(end2),
	}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("agenttest: tool_call/fragmented_interleaved_reconstructs_exactly: Stream returned %v, want no failure (R-CNF-007)", err)
	}
	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	RequireValidStream(t, rec)

	calls := reconstructToolCalls(rec.Events())
	call1, ok := calls[1]
	if !ok || !call1.sawStart || !call1.sawEnd {
		t.Fatalf("call at block 1 = %+v (found=%v), want a complete start+end", call1, ok)
	}
	call2, ok := calls[2]
	if !ok || !call2.sawStart || !call2.sawEnd {
		t.Fatalf("call at block 2 = %+v (found=%v), want a complete start+end", call2, ok)
	}

	if !bytes.Equal(call1.fromDeltas, []byte(`{"q":"a"}`)) {
		t.Errorf("call 1 reconstructed from deltas = %q, want %q — no cross-contamination from call 2 (S-CNF-017)", call1.fromDeltas, `{"q":"a"}`)
	}
	if !bytes.Equal(call2.fromDeltas, []byte(`{"city":"b"}`)) {
		t.Errorf("call 2 reconstructed from deltas = %q, want %q — no cross-contamination from call 1 (S-CNF-017)", call2.fromDeltas, `{"city":"b"}`)
	}
	if !bytes.Equal(call1.arguments, []byte(`{"q":"a"}`)) {
		t.Errorf("call 1 end arguments = %q, want %q", call1.arguments, `{"q":"a"}`)
	}
	if !bytes.Equal(call2.arguments, []byte(`{"city":"b"}`)) {
		t.Errorf("call 2 end arguments = %q, want %q", call2.arguments, `{"city":"b"}`)
	}
}

// toolCallZeroDeltaCase proves R-CNF-007's zero-delta half (S-CNF-018): a
// tool call delivered whole, with zero argument deltas, is accepted — not
// rejected for missing incremental delivery.
func toolCallZeroDeltaCase(t *testing.T, f Factory) {
	t.Helper()

	start, err := ai.NewToolCallStart(1, "call-1", "search")
	requireConstructed(t, err, "ai.NewToolCallStart")
	end, err := ai.NewToolCallEnd(1, []byte(`{"q":"weather"}`))
	requireConstructed(t, err, "ai.NewToolCallEnd")

	script := Script{Steps: []Step{Emit(start), Emit(end)}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("agenttest: tool_call/zero_delta_whole_call_accepted: Stream returned %v, want no failure (R-CNF-007)", err)
	}
	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	RequireValidStream(t, rec)

	events := rec.Events()
	if len(events) != 2 {
		t.Fatalf("drained %d event(s), want 2 (start immediately followed by end, no delta) (S-CNF-018)", len(events))
	}
	calls := reconstructToolCalls(events)
	call, ok := calls[1]
	if !ok || !call.sawStart || !call.sawEnd {
		t.Fatalf("call at block 1 = %+v (found=%v), want a complete start+end", call, ok)
	}
	if !bytes.Equal(call.arguments, []byte(`{"q":"weather"}`)) {
		t.Errorf("arguments = %q, want %q — a zero-delta call still carries its full arguments on the end event alone", call.arguments, `{"q":"weather"}`)
	}
}

// toolCallOrdinalCase proves R-CNF-007's ordinal half (S-CNF-019): two
// calls to the same tool name have distinguishable, unambiguously ordered
// ordinals.
func toolCallOrdinalCase(t *testing.T, f Factory) {
	t.Helper()

	start1, err := ai.NewToolCallStart(1, "call-1", "search")
	requireConstructed(t, err, "ai.NewToolCallStart")
	end1, err := ai.NewToolCallEnd(1, []byte(`{}`))
	requireConstructed(t, err, "ai.NewToolCallEnd")
	start2, err := ai.NewToolCallStart(2, "call-2", "search") // same tool name as call 1
	requireConstructed(t, err, "ai.NewToolCallStart")
	end2, err := ai.NewToolCallEnd(2, []byte(`{}`))
	requireConstructed(t, err, "ai.NewToolCallEnd")

	script := Script{Steps: []Step{Emit(start1), Emit(end1), Emit(start2), Emit(end2)}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("agenttest: tool_call/ordinal_distinguishes_same_tool_name: Stream returned %v, want no failure (R-CNF-007)", err)
	}
	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	RequireValidStream(t, rec)

	calls := reconstructToolCalls(rec.Events())
	call1, ok1 := calls[1]
	call2, ok2 := calls[2]
	if !ok1 || !ok2 {
		t.Fatalf("calls = (%v, %v), want both found", ok1, ok2)
	}
	if call1.ordinal == call2.ordinal {
		t.Errorf("both calls to %q report ordinal %d, want distinct ordinals (S-CNF-019)", call1.name, call1.ordinal)
	}
	if call1.ordinal >= call2.ordinal {
		t.Errorf("call 1's ordinal (%d) does not order before call 2's (%d), want an unambiguous order", call1.ordinal, call2.ordinal)
	}
}

// mixedTextAndToolCallCase proves R-CNF-008: an interaction emitting a text
// block and then a tool call terminates normally with the finish reason
// that denotes a tool call, and both kinds of content survive with their
// ordering invariants intact. It reuses reconstructToolCalls for the
// tool-call half rather than re-asserting tool-call shape independently.
func mixedTextAndToolCallCase(t *testing.T, f Factory) {
	t.Helper()

	textStart, err := ai.NewTextBlockStart(1)
	requireConstructed(t, err, "ai.NewTextBlockStart")
	textDelta, err := ai.NewTextDelta(1, "let me check that")
	requireConstructed(t, err, "ai.NewTextDelta")
	textEnd, err := ai.NewTextBlockEnd(1)
	requireConstructed(t, err, "ai.NewTextBlockEnd")
	callStart, err := ai.NewToolCallStart(2, "call-1", "search")
	requireConstructed(t, err, "ai.NewToolCallStart")
	callEnd, err := ai.NewToolCallEnd(2, []byte(`{"q":"weather"}`))
	requireConstructed(t, err, "ai.NewToolCallEnd")
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	requireConstructed(t, err, "ai.NewCompletion")

	script := Script{Steps: []Step{
		Emit(textStart), Emit(textDelta), Emit(textEnd),
		Emit(callStart), Emit(callEnd),
		Emit(completion),
	}}
	subject := f.New(t, script)
	ch, err := subject.Stream(t.Context(), minimalRequest(t))
	if err != nil {
		t.Fatalf("agenttest: tool_call/mixed_text_and_tool_ends_on_tool_call_finish_reason: Stream returned %v, want no failure (R-CNF-008)", err)
	}
	rec := DrainAndRecord(t, ch, DefaultDrainTimeout)
	RequireValidStream(t, rec)

	events := rec.Events()
	if len(events) != 6 {
		t.Fatalf("drained %d event(s), want 6 (text start/delta/end, call start/end, completion) (S-CNF-020)", len(events))
	}

	sawText := false
	for _, ev := range events {
		if ev.Kind() == ai.EventKindTextDelta {
			d, _ := ev.TextDelta()
			if d.Delta() == "let me check that" {
				sawText = true
			}
		}
	}
	if !sawText {
		t.Error("the scripted text delta did not survive in the recording, want text and tool content to coexist (S-CNF-020)")
	}

	calls := reconstructToolCalls(events)
	call, ok := calls[2]
	if !ok || !call.sawStart || !call.sawEnd {
		t.Fatalf("call at block 2 = %+v (found=%v), want a complete start+end", call, ok)
	}
	if !bytes.Equal(call.arguments, []byte(`{"q":"weather"}`)) {
		t.Errorf("call arguments = %q, want %q", call.arguments, `{"q":"weather"}`)
	}

	last := events[len(events)-1]
	comp, ok := last.Completion()
	if !ok {
		t.Fatal("the last event is not a Completion, want the terminal to carry the finish reason")
	}
	if comp.FinishReason() != ai.FinishReasonToolCalls {
		t.Errorf("finish reason = %v, want %v — the closed-vocabulary tool-call value (S-CNF-020)", comp.FinishReason(), ai.FinishReasonToolCalls)
	}
}
