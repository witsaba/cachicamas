// AI-30.2 — R-ATL-007: empty fragments are no-ops, zero-fragment calls
// canonicalize to `{}`, and a whole call normalizes identically to its
// fragmented twin.
//
// S-ATL-033…037:
//   033: ""-then-content-then-"" no-op
//   034: zero-accumulated-bytes end byte-equal to NewToolCall(id,name,nil)'s
//   035: absent-key vs "" key parity
//   036: whole call vs its 5-fragment twin byte-identical ends
//   037: zero ToolCallDelta events for the zero-byte case

package openaicompat

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestEmptyFragment_EmptyThenContentThenEmpty_NoOp covers S-ATL-033: an
// element carrying "", then content, then "" is treated as if the empty
// fragments were absent.
func TestEmptyFragment_EmptyThenContentThenEmpty_NoOp(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_E","function":{"name":"search","arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	empty1 := `{"index":0,"function":{"arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", empty1)); err != nil {
		t.Fatalf("applyChunk(empty1) error = %v", err)
	}
	content := `{"index":0,"function":{"arguments":"{\"q\":"}}`
	if _, err := state.applyChunk(chunkFromTools("c", content)); err != nil {
		t.Fatalf("applyChunk(content) error = %v", err)
	}
	empty2 := `{"index":0,"function":{"arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", empty2)); err != nil {
		t.Fatalf("applyChunk(empty2) error = %v", err)
	}
	rest := `{"index":0,"function":{"arguments":"\"a\"}"}}`
	if _, err := state.applyChunk(chunkFromTools("c", rest)); err != nil {
		t.Fatalf("applyChunk(rest) error = %v", err)
	}
	evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	var endArgs []byte
	for _, ev := range evs {
		if e, ok := ev.ToolCallEnd(); ok {
			endArgs = e.Arguments()
		}
	}
	if string(endArgs) != `{"q":"a"}` {
		t.Errorf("end args = %q, want %q — empty fragments are no-ops (S-ATL-033)", endArgs, `{"q":"a"}`)
	}
}

// TestEmptyFragment_ZeroAccumulatedBytes_EqualsEmptyToolArguments
// covers S-ATL-034: a call whose only element carries index+id+name (no
// arguments) closes with arguments byte-equal to NewToolCall(id,name,nil)'s.
func TestEmptyFragment_ZeroAccumulatedBytes_EqualsEmptyToolArguments(t *testing.T) {
	t.Parallel()

	// Compute the canonical empty-arguments bytes from ai.NewToolCall
	// itself (R-ATL-007's "different code path" requirement).
	ref, err := ai.NewToolCall("call_Z", "search", nil)
	if err != nil {
		t.Fatalf("ai.NewToolCall(ref): %v", err)
	}
	refTC, _ := ref.ToolCall()
	want := refTC.Arguments()

	state := &mapperState{}
	id := `{"index":0,"id":"call_Z","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	var endArgs []byte
	for _, ev := range evs {
		if e, ok := ev.ToolCallEnd(); ok {
			endArgs = e.Arguments()
		}
	}
	if string(endArgs) != string(want) {
		t.Errorf("end args = %q, want %q — zero-byte canonicalization matches NewToolCall (S-ATL-034)", endArgs, want)
	}
}

// TestEmptyFragment_AbsentKeyVsEmptyKey_Parity covers S-ATL-035: an
// element with "arguments":"" and an element omitting the key are
// indistinguishable.
func TestEmptyFragment_AbsentKeyVsEmptyKey_Parity(t *testing.T) {
	t.Parallel()

	stateEmpty := &mapperState{}
	stateAbsent := &mapperState{}

	// Empty-key: identity chunk explicitly carries arguments:"".
	idEmpty := `{"index":0,"id":"call_E","function":{"name":"search","arguments":""}}`
	idAbsent := `{"index":0,"id":"call_E","function":{"name":"search"}}`

	if _, err := stateEmpty.applyChunk(chunkFromTools("c", idEmpty)); err != nil {
		t.Fatalf("applyChunk(idEmpty) error = %v", err)
	}
	if _, err := stateAbsent.applyChunk(chunkFromTools("c", idAbsent)); err != nil {
		t.Fatalf("applyChunk(idAbsent) error = %v", err)
	}

	term := `{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`
	evsEmpty, err := stateEmpty.applyChunk(mustDecode(term))
	if err != nil {
		t.Fatalf("applyChunk(term empty) error = %v", err)
	}
	evsAbsent, err := stateAbsent.applyChunk(mustDecode(term))
	if err != nil {
		t.Fatalf("applyChunk(term absent) error = %v", err)
	}

	getEnd := func(evs []ai.Event) []byte {
		for _, ev := range evs {
			if e, ok := ev.ToolCallEnd(); ok {
				return e.Arguments()
			}
		}
		return nil
	}
	a, b := getEnd(evsEmpty), getEnd(evsAbsent)
	if string(a) != string(b) {
		t.Errorf("empty-key end = %q, absent-key end = %q, want byte-equal (S-ATL-035)", a, b)
	}
}

// TestEmptyFragment_WholeCallVsFragmentedTwin covers S-ATL-036: a call
// delivered whole and its 5-fragment twin normalize identically at the
// end event.
func TestEmptyFragment_WholeCallVsFragmentedTwin(t *testing.T) {
	t.Parallel()

	wholeSrc := `{"city":"Caracas","units":"c"}`
	twinSrcs := []string{`{"city":`, `"Caracas",`, `"units":`, `"c"}`, ``} // the last fragment is empty

	runOnce := func(firstFrag string, moreFrags []string) []byte {
		state := &mapperState{}
		id := `{"index":0,"id":"call_W","function":{"name":"search"}}`
		if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
			t.Fatalf("applyChunk(id) error = %v", err)
		}
		if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(firstFrag))+`}}`)); err != nil {
			t.Fatalf("applyChunk(first) error = %v", err)
		}
		for _, f := range moreFrags {
			if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(f))+`}}`)); err != nil {
				t.Fatalf("applyChunk(more) error = %v", err)
			}
		}
		evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
		if err != nil {
			t.Fatalf("applyChunk(term) error = %v", err)
		}
		for _, ev := range evs {
			if e, ok := ev.ToolCallEnd(); ok {
				return e.Arguments()
			}
		}
		return nil
	}

	wholeEnd := runOnce(wholeSrc, nil)
	twinEnd := runOnce(twinSrcs[0], twinSrcs[1:])

	if string(wholeEnd) != string(twinEnd) {
		t.Errorf("whole end = %q, twin end = %q, want byte-identical (S-ATL-036)", wholeEnd, twinEnd)
	}
	if string(wholeEnd) != wholeSrc {
		t.Errorf("whole end = %q, want %q", wholeEnd, wholeSrc)
	}
}

// TestEmptyFragment_ZeroDeltaCount covers S-ATL-037: a zero-byte call
// emits zero ToolCallDelta events.
func TestEmptyFragment_ZeroDeltaCount(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_Z","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	var deltaCount int
	for _, ev := range evs {
		if _, ok := ev.ToolCallDelta(); ok {
			deltaCount++
		}
	}
	if deltaCount != 0 {
		t.Errorf("zero-byte call emitted %d deltas, want 0 (S-ATL-037)", deltaCount)
	}
}

// TestEmptyFragment_EmptyArgsAfterIdentity_NoDelta covers the corner
// case: identity chunk carries arguments:"" — no delta should be emitted.
func TestEmptyFragment_EmptyArgsAfterIdentity_NoDelta(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_E","function":{"name":"search","arguments":""}}`
	evs, err := state.applyChunk(chunkFromTools("c", id))
	if err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	var deltaCount int
	for _, ev := range evs {
		if _, ok := ev.ToolCallDelta(); ok {
			deltaCount++
		}
	}
	if deltaCount != 0 {
		t.Errorf("identity chunk with arguments:\"\" emitted %d deltas, want 0 (S-ATL-033)", deltaCount)
	}
}

// ensure imports are used
var _ = strings.Contains
