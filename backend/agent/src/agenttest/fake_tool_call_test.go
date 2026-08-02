// AI-21.2 — a scripted tool call: start, argument deltas and end.
//
// fake_tool_call_test.go proves R-AFP-005…007 from outside package
// agenttest: a delta-carrying call reconstructs to exact argument bytes, a
// zero-delta call is indistinguishable from one after reconstruction, and
// interleaved calls keep their own ordinals. It reuses fake_text_test.go's
// shared helpers (mustSimpleRequest from provider_test.go, drainFake) —
// same agenttest_test package.
package agenttest_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// reconstructedToolCall is one tool call rebuilt from a drained stream's
// start/delta(s)/end events at one block — text_events.go's own posture,
// restated for tool calls: this package ships no accumulator, so
// reconstruction is test-local (doc 0001 § 4.3 invariant 1).
type reconstructedToolCall struct {
	id, name                string
	arguments               []byte
	reconstructedFromDeltas []byte
	sawStart, sawEnd        bool
}

// reconstructToolCall walks events in order and rebuilds the tool call at
// block, concatenating delta fragments byte-for-byte as they arrive.
func reconstructToolCall(events []ai.Event, block ai.BlockIndex) reconstructedToolCall {
	var out reconstructedToolCall
	var buf bytes.Buffer
	for _, ev := range events {
		switch ev.Kind() {
		case ai.EventKindToolCallStart:
			s, ok := ev.ToolCallStart()
			if !ok || s.BlockIndex() != block {
				continue
			}
			out.id, out.name, out.sawStart = s.ID(), s.Name(), true
		case ai.EventKindToolCallDelta:
			d, ok := ev.ToolCallDelta()
			if !ok || d.BlockIndex() != block {
				continue
			}
			buf.Write(d.Fragment())
		case ai.EventKindToolCallEnd:
			e, ok := ev.ToolCallEnd()
			if !ok || e.BlockIndex() != block {
				continue
			}
			out.arguments, out.sawEnd = e.Arguments(), true
		}
	}
	out.reconstructedFromDeltas = buf.Bytes()
	return out
}

// mustToolCallScript builds a script for one tool call at block: a start,
// one delta per fragment (zero fragments is legal — S-AFP-014/015), and an
// end carrying arguments.
//
// arguments is supplied independently of fragments, exactly as a real
// adapter would (tool_call_event.go: NewToolCallEnd's bytes are never
// derived from the deltas that preceded them) — a zero-delta call still
// carries its full, complete arguments on the end event alone.
func mustToolCallScript(t *testing.T, block ai.BlockIndex, id, name, arguments string, fragments ...string) agenttest.Script {
	t.Helper()

	start, err := ai.NewToolCallStart(block, id, name)
	if err != nil {
		t.Fatalf("ai.NewToolCallStart returned %v, want no failure", err)
	}
	steps := []agenttest.Step{agenttest.Emit(start)}

	for _, fragment := range fragments {
		delta, err := ai.NewToolCallDelta(block, []byte(fragment))
		if err != nil {
			t.Fatalf("ai.NewToolCallDelta returned %v, want no failure", err)
		}
		steps = append(steps, agenttest.Emit(delta))
	}

	end, err := ai.NewToolCallEnd(block, []byte(arguments))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd returned %v, want no failure", err)
	}
	steps = append(steps, agenttest.Emit(end))
	return agenttest.Script{Steps: steps}
}

// AI-21.2 item 1 (R-AFP-005, S-AFP-012/013) — a tool call scripted with
// arguments split across three fragments reconstructs to the exact
// scripted bytes, with identity and name preserved, and the drained
// events are exactly a start, the scripted deltas in order, and an end.
func TestProvider_ScriptedToolCall_DeltaCarrying_ReconstructsToExactArgumentBytes(t *testing.T) {
	t.Parallel()

	const wantArgs = `{"q":"weather"}`
	script := mustToolCallScript(t, 1, "call-1", "search", wantArgs, `{"q":`, `"weat`, `her"}`)
	provider := agenttest.NewProvider(script)
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	got := drainFake(t, ch)
	if len(got) != 5 {
		t.Fatalf("drained %d event(s), want 5 (start, three deltas, end)", len(got))
	}

	wantKinds := []ai.EventKind{
		ai.EventKindToolCallStart, ai.EventKindToolCallDelta, ai.EventKindToolCallDelta,
		ai.EventKindToolCallDelta, ai.EventKindToolCallEnd,
	}
	for i, ev := range got {
		if ev.Kind() != wantKinds[i] {
			t.Errorf("event %d kind = %v, want %v (none synthesised, none dropped)", i, ev.Kind(), wantKinds[i])
		}
	}

	call := reconstructToolCall(got, 1)
	if !call.sawStart || !call.sawEnd {
		t.Fatalf("reconstruction missing start (saw=%v) or end (saw=%v)", call.sawStart, call.sawEnd)
	}
	if !bytes.Equal(call.reconstructedFromDeltas, []byte(wantArgs)) {
		t.Errorf("reconstructed arguments = %q, want %q (byte-for-byte)", call.reconstructedFromDeltas, wantArgs)
	}
	if !bytes.Equal(call.arguments, []byte(wantArgs)) {
		t.Errorf("end event's arguments = %q, want %q", call.arguments, wantArgs)
	}
	if call.id != "call-1" {
		t.Errorf("id = %q, want %q", call.id, "call-1")
	}
	if call.name != "search" {
		t.Errorf("name = %q, want %q", call.name, "search")
	}
}

// AI-21.2 item 2 (R-AFP-006, S-AFP-014/015) — a zero-delta tool call is
// scriptable, drains as a start immediately followed by its end with no
// delta between them, and is indistinguishable after reconstruction from a
// delta-carrying call scripted with identical arguments.
func TestProvider_ScriptedToolCall_ZeroDelta_IndistinguishableFromDeltaCarryingAfterReconstruction(t *testing.T) {
	t.Parallel()

	const args = `{"q":"weather"}`
	zeroProvider := agenttest.NewProvider(mustToolCallScript(t, 1, "call-1", "search", args))
	deltaProvider := agenttest.NewProvider(mustToolCallScript(t, 1, "call-1", "search", args, `{"q":`, `"weather"}`))

	zeroCh, err := zeroProvider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("zero-delta Stream returned %v, want no failure", err)
	}
	zeroEvents := drainFake(t, zeroCh)
	if len(zeroEvents) != 2 {
		t.Fatalf("zero-delta script drained %d event(s), want 2 (start, end, no delta between them)", len(zeroEvents))
	}
	if zeroEvents[0].Kind() != ai.EventKindToolCallStart || zeroEvents[1].Kind() != ai.EventKindToolCallEnd {
		t.Fatalf("zero-delta events = [%v, %v], want [start, end]", zeroEvents[0].Kind(), zeroEvents[1].Kind())
	}

	deltaCh, err := deltaProvider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("delta-carrying Stream returned %v, want no failure", err)
	}
	deltaEvents := drainFake(t, deltaCh)

	zeroCall := reconstructToolCall(zeroEvents, 1)
	deltaCall := reconstructToolCall(deltaEvents, 1)

	if zeroCall.id != deltaCall.id || zeroCall.name != deltaCall.name {
		t.Errorf("zero-delta call = (%q,%q), delta-carrying call = (%q,%q), want equal identity/name",
			zeroCall.id, zeroCall.name, deltaCall.id, deltaCall.name)
	}
	if !bytes.Equal(zeroCall.arguments, []byte(args)) {
		t.Errorf("zero-delta call's arguments = %q, want %q (full arguments still reconstructed with no deltas)", zeroCall.arguments, args)
	}
	if !bytes.Equal(zeroCall.arguments, deltaCall.arguments) {
		t.Errorf("zero-delta arguments = %q, delta-carrying arguments = %q, want equal (indistinguishable after reconstruction)",
			zeroCall.arguments, deltaCall.arguments)
	}
}

// AI-21.2 item 3 (R-AFP-007, S-AFP-016/017) — two tool calls whose starts,
// deltas and ends are scripted interleaved each reconstruct independently
// with no cross-contamination, and every event carries the ordinal (block
// index) of the call it belongs to, unchanged by the interleaving.
func TestProvider_InterleavedToolCalls_KeepDistinctOrdinalsAndReconstructIndependently(t *testing.T) {
	t.Parallel()

	start1, err := ai.NewToolCallStart(1, "call-1", "search")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart(1) returned %v, want no failure", err)
	}
	start2, err := ai.NewToolCallStart(2, "call-2", "weather")
	if err != nil {
		t.Fatalf("ai.NewToolCallStart(2) returned %v, want no failure", err)
	}
	delta1a, err := ai.NewToolCallDelta(1, []byte(`{"q":`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta(1a) returned %v, want no failure", err)
	}
	delta2a, err := ai.NewToolCallDelta(2, []byte(`{"city":`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta(2a) returned %v, want no failure", err)
	}
	delta1b, err := ai.NewToolCallDelta(1, []byte(`"a"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta(1b) returned %v, want no failure", err)
	}
	delta2b, err := ai.NewToolCallDelta(2, []byte(`"b"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallDelta(2b) returned %v, want no failure", err)
	}
	end1, err := ai.NewToolCallEnd(1, []byte(`{"q":"a"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd(1) returned %v, want no failure", err)
	}
	end2, err := ai.NewToolCallEnd(2, []byte(`{"city":"b"}`))
	if err != nil {
		t.Fatalf("ai.NewToolCallEnd(2) returned %v, want no failure", err)
	}

	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start1), agenttest.Emit(start2),
		agenttest.Emit(delta1a), agenttest.Emit(delta2a),
		agenttest.Emit(delta1b), agenttest.Emit(delta2b),
		agenttest.Emit(end1), agenttest.Emit(end2),
	}}

	provider := agenttest.NewProvider(script)
	ch, err := provider.Stream(context.Background(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	got := drainFake(t, ch)
	if len(got) != 8 {
		t.Fatalf("drained %d event(s), want 8", len(got))
	}

	call1 := reconstructToolCall(got, 1)
	call2 := reconstructToolCall(got, 2)

	if call1.id != "call-1" || call1.name != "search" {
		t.Errorf("call 1 = (%q,%q), want (call-1,search)", call1.id, call1.name)
	}
	if call2.id != "call-2" || call2.name != "weather" {
		t.Errorf("call 2 = (%q,%q), want (call-2,weather)", call2.id, call2.name)
	}
	if !bytes.Equal(call1.reconstructedFromDeltas, []byte(`{"q":"a"}`)) {
		t.Errorf("call 1 reconstructed = %q, want %q (no cross-contamination from call 2)", call1.reconstructedFromDeltas, `{"q":"a"}`)
	}
	if !bytes.Equal(call2.reconstructedFromDeltas, []byte(`{"city":"b"}`)) {
		t.Errorf("call 2 reconstructed = %q, want %q (no cross-contamination from call 1)", call2.reconstructedFromDeltas, `{"city":"b"}`)
	}

	wantBlocks := []ai.BlockIndex{1, 2, 1, 2, 1, 2, 1, 2}
	for i, ev := range got {
		var block ai.BlockIndex
		switch ev.Kind() {
		case ai.EventKindToolCallStart:
			s, _ := ev.ToolCallStart()
			block = s.BlockIndex()
		case ai.EventKindToolCallDelta:
			d, _ := ev.ToolCallDelta()
			block = d.BlockIndex()
		case ai.EventKindToolCallEnd:
			e, _ := ev.ToolCallEnd()
			block = e.BlockIndex()
		}
		if block != wantBlocks[i] {
			t.Errorf("event %d block = %v, want %v (ordinal unchanged by interleaving)", i, block, wantBlocks[i])
		}
	}
}
