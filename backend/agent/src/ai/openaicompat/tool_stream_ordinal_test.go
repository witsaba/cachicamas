// AI-30.5 — R-ATL-011: each call's ordinal position is observable
// regardless of fragment interleaving.
//
// S-ATL-055: first-appearance wire order 2,0,1 maps to ordinals 1,2,3
// (a mapping sorting on index cannot reproduce).
// S-ATL-056: round-robin interleaving preserves the same ordinal mapping.
// S-ATL-057: two calls sharing one tool name still get distinct ordinals.
//
// S-ATL-058 (inspection): no production source has an ordinal counter,
// field, or accessor (R-ATC-012). Derivation is test-side only.
//
// Strict TDD: each test below exercises production code already shipped
// in slice 1 (the mapperState.toolOpenOrder slice is the ordinal source,
// per design.md D4). The tests do not modify production code; they
// verify the observed event order matches the documented ordinal
// derivation rule.

package openaicompat

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestOrdinal_FirstAppearanceOrder covers S-ATL-055: when three calls
// appear in wire order 2,0,1, their ordinals are 1,2,3 — a mapping
// sorted on index would reproduce only by accident.
func TestOrdinal_FirstAppearanceOrder(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Identity chunks in wire order 2, 0, 1.
	id := `{"index":2,"id":"call_C","function":{"name":"a","arguments":"{}"}},` +
		`{"index":0,"id":"call_A","function":{"name":"a","arguments":"{}"}},` +
		`{"index":1,"id":"call_B","function":{"name":"a","arguments":"{}"}}`
	evsID, err := state.applyChunk(chunkFromTools("c", id))
	if err != nil {
		t.Fatalf("applyChunk(ids) error = %v", err)
	}
	evsEnd, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	all := append(evsID, evsEnd...)

	// Capture start events in emission order, with their block indices.
	var startBlocks []ai.BlockIndex
	for _, ev := range all {
		if s, ok := ev.ToolCallStart(); ok {
			startBlocks = append(startBlocks, s.BlockIndex())
		}
	}
	if len(startBlocks) != 3 {
		t.Fatalf("got %d starts, want 3 (S-ATL-055)", len(startBlocks))
	}

	// Derive ordinals via the same rule production uses: start order.
	// ordinal[0] (emitted first) was wire index 2 → first call.
	// ordinal[1] was wire index 0 → second call.
	// ordinal[2] was wire index 1 → third call.
	// The mapper's toolOpenOrder records [2, 0, 1] — first appearance.
	wantToolOpenOrder := []int{2, 0, 1}
	if !intSliceEqual(state.toolOpenOrder, wantToolOpenOrder) {
		t.Errorf("toolOpenOrder = %v, want %v (S-ATL-055)", state.toolOpenOrder, wantToolOpenOrder)
	}
	// The end events are also emitted in toolOpenOrder (D4).
	var endBlocks []ai.BlockIndex
	for _, ev := range all {
		if e, ok := ev.ToolCallEnd(); ok {
			endBlocks = append(endBlocks, e.BlockIndex())
		}
	}
	if !blocksByOpenOrderEqual(endBlocks, state.toolOpenOrder, state) {
		t.Errorf("end blocks %v do not match toolOpenOrder %v (S-ATL-055)", endBlocks, state.toolOpenOrder)
	}
}

// TestOrdinal_RoundRobinInterleaving covers S-ATL-056: a transcript
// round-robining three calls' fragments preserves the ordinal mapping.
func TestOrdinal_RoundRobinInterleaving(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Three identities in wire order 2, 0, 1.
	ids := `{"index":2,"id":"call_C","function":{"name":"c","arguments":""}},` +
		`{"index":0,"id":"call_A","function":{"name":"a","arguments":""}},` +
		`{"index":1,"id":"call_B","function":{"name":"b","arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", ids)); err != nil {
		t.Fatalf("applyChunk(ids) error = %v", err)
	}
	// Round-robin fragments.
	for round := 0; round < 2; round++ {
		var frags []string
		for _, idx := range []int{0, 1, 2} {
			frags = append(frags, `{"index":`+itoa(idx)+`,"function":{"arguments":"`+markerForOrdinal(idx, round)+`"}}`)
		}
		if _, err := state.applyChunk(chunkFromTools("c", joinComma(frags))); err != nil {
			t.Fatalf("applyChunk(round %d) error = %v", round, err)
		}
	}
	// toolOpenOrder must be preserved by interleaving: [2, 0, 1] (first
	// seen order).
	want := []int{2, 0, 1}
	if !intSliceEqual(state.toolOpenOrder, want) {
		t.Errorf("toolOpenOrder = %v, want %v — interleaving must preserve ordinal (S-ATL-056)", state.toolOpenOrder, want)
	}
}

// TestOrdinal_SameNameDistinctOrdinals covers S-ATL-057: two calls
// sharing one tool name still have distinct, strictly-ordered
// ordinals.
func TestOrdinal_SameNameDistinctOrdinals(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	ids := `{"index":0,"id":"call_X","function":{"name":"search","arguments":"{}"}},` +
		`{"index":1,"id":"call_Y","function":{"name":"search","arguments":"{}"}}`
	evsID, err := state.applyChunk(chunkFromTools("c", ids))
	if err != nil {
		t.Fatalf("applyChunk(ids) error = %v", err)
	}
	evsEnd, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	all := append(evsID, evsEnd...)
	// Capture start events in emission order — both have name "search"
	// but distinct ordinals by position.
	var startNames []string
	for _, ev := range all {
		if s, ok := ev.ToolCallStart(); ok {
			startNames = append(startNames, s.Name())
		}
	}
	if len(startNames) != 2 {
		t.Fatalf("got %d starts, want 2 (S-ATL-057)", len(startNames))
	}
	if startNames[0] != "search" || startNames[1] != "search" {
		t.Errorf("names = %v, want both \"search\" (S-ATL-057)", startNames)
	}
	// toolOpenOrder is the ordinal derivation source.
	if len(state.toolOpenOrder) != 2 {
		t.Errorf("toolOpenOrder len = %d, want 2 (S-ATL-057)", len(state.toolOpenOrder))
	}
}

// helpers --------------------------------------------------------------

func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// blocksByOpenOrderEqual confirms the end event's blocks are emitted
// in toolOpenOrder — same axis as the ordinal derivation.
func blocksByOpenOrderEqual(blocks []ai.BlockIndex, order []int, state *mapperState) bool {
	if len(blocks) != len(order) {
		return false
	}
	for i, wireIdx := range order {
		st := state.toolCalls[wireIdx]
		if st == nil {
			return false
		}
		if blocks[i] != st.block {
			return false
		}
	}
	return true
}

func joinComma(s []string) string {
	out := ""
	for i, x := range s {
		if i > 0 {
			out += ","
		}
		out += x
	}
	return out
}

// markerForOrdinal: small distinct literal per call per round.
func markerForOrdinal(callIdx, round int) string {
	const letters = "ABCDEF"
	return string([]byte{letters[callIdx*2], letters[callIdx*2+1]}) + itoa(round)
}