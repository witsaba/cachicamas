// AI-30.1 — R-ATL-002/003/004: per-call accumulation in mapperState.
//
// S-ATL-006…012 prove the index-correlation rule + D1's shared next-free
// block-index allocator (no collision with the text block, stable per
// call, ≥ 1, never equal to wire index).
//
// S-ATL-013…018 prove the identity/name rule: byte-exact from the start
// event before any argument byte; absent and identical-repeat tolerated;
// differing non-empty values fail.
//
// S-ATL-019…021 prove the per-call buffer isolation: two interleaved calls
// with distinguishable bytes after every interleave point (S-ATL-019);
// three-call round-robin with per-call marker bytes (S-ATL-020); single
// chunk two-element array (S-ATL-021).
//
// Strict TDD: every RED scenario here exercises a production method whose
// implementation lands in stream_state.go alongside mapperState.toolCalls,
// toolOpenOrder and nextBlockIndex (D1/D3).

package openaicompat

import (
	"context"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// chunkFromTools builds a wireChunk whose choice-0 delta carries the
// given tool_calls elements and an empty content (matches every
// fixture in this file; AI-30.1's tool-call-only shape).
func chunkFromTools(id string, toolCallsJSON string) wireChunk {
	// Marshal the supplied JSON literal as a RawMessage.
	raw := []byte(`{"id":"` + id + `","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[` + toolCallsJSON + `]},"finish_reason":null}]}`)
	chunk, err := decodeChunk(raw)
	if err != nil {
		panic("chunkFromTools: " + err.Error())
	}
	return chunk
}

// drainEventsFromMapper feeds every supplied chunk to mapperState and
// returns the concatenated event slice. Used by every test in this file.
func drainEventsFromMapper(t *testing.T, chunks ...wireChunk) []ai.Event {
	t.Helper()
	state := &mapperState{}
	var events []ai.Event
	for _, c := range chunks {
		evs, err := state.applyChunk(c)
		if err != nil {
			t.Fatalf("applyChunk error = %v at chunk with %d tool_calls", err, len(c.Choices[0].Delta.ToolCalls))
		}
		events = append(events, evs...)
	}
	return events
}

// TestAccumulation_InterleavedTwoCalls_DistinctBlocksAndOrderedArgs
// covers S-ATL-006, S-ATL-007, S-ATL-019: two concurrent calls whose
// fragments interleave on wire index reconstruct independently with no
// cross-contamination, each carrying a distinct block index (≥ 1).
func TestAccumulation_InterleavedTwoCalls_DistinctBlocksAndOrderedArgs(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Identity chunks first: each call establishes its own id/name.
	id0 := `{"index":0,"id":"call_A1","function":{"name":"search","arguments":""}}`
	id1 := `{"index":1,"id":"call_A2","function":{"name":"weather","arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", id0)); err != nil {
		t.Fatalf("applyChunk(id0) error = %v", err)
	}
	if _, err := state.applyChunk(chunkFromTools("c", id1)); err != nil {
		t.Fatalf("applyChunk(id1) error = %v", err)
	}
	// Two interleave points with distinguishable bytes after each.
	frag0a := `{"index":0,"function":{"arguments":"{\"q\":\"al"}}`
	frag1a := `{"index":1,"function":{"arguments":"{\"city\":\"be"}}`
	frag0b := `{"index":0,"function":{"arguments":"pha\"}"}}`
	frag1b := `{"index":1,"function":{"arguments":"ta\"}"}}`
	if _, err := state.applyChunk(chunkFromTools("c", frag0a+","+frag1a)); err != nil {
		t.Fatalf("applyChunk(frag round 1) error = %v", err)
	}
	if _, err := state.applyChunk(chunkFromTools("c", frag0b+","+frag1b)); err != nil {
		t.Fatalf("applyChunk(frag round 2) error = %v", err)
	}
	// Terminal chunk closes both.
	term := `{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`
	tc, err := decodeChunk([]byte(term))
	if err != nil {
		t.Fatalf("decodeChunk(terminal) error = %v", err)
	}
	evs, err := state.applyChunk(tc)
	if err != nil {
		t.Fatalf("applyChunk(terminal) error = %v", err)
	}
	all := append(append([]ai.Event(nil), nil...), evs...) // placeholder
	_ = all

	// Find the two ToolCallEnd events; their blocks must differ and their
	// arguments must match the source literals byte-equal.
	var ends []ai.ToolCallEnd
	for _, ev := range evs {
		if end, ok := ev.ToolCallEnd(); ok {
			ends = append(ends, end)
		}
	}
	if len(ends) != 2 {
		t.Fatalf("got %d ToolCallEnd events, want 2 (S-ATL-006, S-ATL-024)", len(ends))
	}
	if ends[0].BlockIndex() == ends[1].BlockIndex() {
		t.Errorf("both calls reported block %d, want distinct blocks (S-ATL-007)", ends[0].BlockIndex())
	}
	if ends[0].BlockIndex() < 1 || ends[1].BlockIndex() < 1 {
		t.Errorf("block indices = (%d, %d), want ≥ 1 — BlockIndex 0 is invalid (R-ATC-004, S-ATL-008)", ends[0].BlockIndex(), ends[1].BlockIndex())
	}
	got0, got1 := ends[0].Arguments(), ends[1].Arguments()
	if string(got0) != `{"q":"alpha"}` || string(got1) != `{"city":"beta"}` {
		t.Errorf("end arguments = (%q, %q), want (%q, %q) — distinguishable bytes after every interleave (S-ATL-019)",
			got0, got1, `{"q":"alpha"}`, `{"city":"beta"}`)
	}
}

// TestAccumulation_WireIndexZero_LegalAndDistinctFromText covers
// S-ATL-008: wire index 0 is legal (C9.1, the schema does not require
// any specific value); the call's block index is ≥ 1 (R-ATC-004).
func TestAccumulation_WireIndexZero_LegalAndDistinctFromText(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	chunks := []wireChunk{
		chunkFromTools("c", `{"index":0,"id":"call_0","function":{"name":"search","arguments":""}}`),
	}
	if _, err := state.applyChunk(chunks[0]); err != nil {
		t.Fatalf("applyChunk error = %v, want nil — wire index 0 is legal (C9.1)", err)
	}
	// Force a close by terminal chunk.
	term := `{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`
	tc, err := decodeChunk([]byte(term))
	if err != nil {
		t.Fatalf("decodeChunk(terminal) error = %v", err)
	}
	evs, err := state.applyChunk(tc)
	if err != nil {
		t.Fatalf("applyChunk(terminal) error = %v", err)
	}
	var ends []ai.ToolCallEnd
	for _, ev := range evs {
		if end, ok := ev.ToolCallEnd(); ok {
			ends = append(ends, end)
		}
	}
	if len(ends) != 1 {
		t.Fatalf("got %d ends, want 1", len(ends))
	}
	if ends[0].BlockIndex() != 1 {
		t.Errorf("block index = %d, want 1 — no text block was minted, so the first tool call takes the text block's slot (D1, S-ATL-008)", ends[0].BlockIndex())
	}
}

// TestAccumulation_TextPresent_ToolBlocksNeverTakeTextIndex covers
// S-ATL-009: when a stream carries both text content and tool calls, no
// tool-call event reports block 1 (the text block's constant). The
// landed wire shape carries text in earlier chunks and tool calls in
// later chunks (mixedTextAndToolCallCase's own shape), so this test
// follows that ordering.
func TestAccumulation_TextPresent_ToolBlocksNeverTakeTextIndex(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Text chunk first, then tool call chunk.
	textChunk := mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hello"},"finish_reason":null}]}`)
	toolChunk := chunkFromTools("c", `{"index":0,"id":"call_T","function":{"name":"search","arguments":"{}"}}`)
	if _, err := state.applyChunk(textChunk); err != nil {
		t.Fatalf("applyChunk(text) error = %v", err)
	}
	evs, err := state.applyChunk(toolChunk)
	if err != nil {
		t.Fatalf("applyChunk(tool) error = %v", err)
	}
	var toolBlock ai.BlockIndex
	for _, ev := range evs {
		if s, ok := ev.ToolCallStart(); ok {
			toolBlock = s.BlockIndex()
		}
	}
	if toolBlock == textBlockIndex {
		t.Errorf("tool block = %d, want ≠ %d (text block constant) — D1 forbids collision (S-ATL-009)", toolBlock, textBlockIndex)
	}
}

// mustDecode is a tiny helper for hand-rolled JSON literal chunks in
// this file's tests; panics on decode error (test-only).
func mustDecode(s string) wireChunk {
	c, err := decodeChunk([]byte(s))
	if err != nil {
		panic("mustDecode: " + err.Error())
	}
	return c
}

// TestAccumulation_IdentityBeforeDelta covers S-ATL-013: an element
// carrying id+name+arguments in ONE chunk produces a start event whose
// id/name byte-equal the source AND whose start precedes any delta for
// the same call.
func TestAccumulation_IdentityBeforeDelta(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Use a complete JSON fragment so the terminal close passes
	// ai.NewToolCall's well-formed-JSON gate.
	evs, err := state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_A1","function":{"name":"search","arguments":"{\"q\":\"a\"}"}}`))
	if err != nil {
		t.Fatalf("applyChunk(id+frag) error = %v", err)
	}
	// evs[0] is the response start (always first chunk); the tool-call
	// events come after it.
	var startIdx, deltaIdx = -1, -1
	for i, ev := range evs {
		switch ev.Kind() {
		case ai.EventKindToolCallStart:
			if startIdx == -1 {
				startIdx = i
				if s, ok := ev.ToolCallStart(); ok {
					if s.ID() != "call_A1" {
						t.Errorf("start.ID() = %q, want %q (S-ATL-013)", s.ID(), "call_A1")
					}
					if s.Name() != "search" {
						t.Errorf("start.Name() = %q, want %q (S-ATL-013)", s.Name(), "search")
					}
				}
			}
		case ai.EventKindToolCallDelta:
			if deltaIdx == -1 {
				deltaIdx = i
			}
		}
	}
	if startIdx == -1 {
		t.Fatal("no ToolCallStart found in events (S-ATL-013)")
	}
	if deltaIdx == -1 {
		t.Fatal("no ToolCallDelta found in events (S-ATL-013)")
	}
	if startIdx > deltaIdx {
		t.Errorf("start at index %d, delta at index %d — start must precede delta (S-ATL-013)", startIdx, deltaIdx)
	}
}

// TestAccumulation_IdentityOmittedOnLaterElement_Tolerated covers
// S-ATL-014: a call whose second and third elements carry no id/name
// still produces exactly one start and both fragments attribute to the
// same call.
func TestAccumulation_IdentityOmittedOnLaterElement_Tolerated(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_omit","function":{"name":"search","arguments":""}}`
	d1 := `{"index":0,"function":{"arguments":"{\"q\":"}}`
	d2 := `{"index":0,"function":{"arguments":"\"a\"}"}}`
	term := `{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`

	evsID, err := state.applyChunk(chunkFromTools("c", id))
	if err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	evsD1, err := state.applyChunk(chunkFromTools("c", d1))
	if err != nil {
		t.Fatalf("applyChunk(d1) error = %v", err)
	}
	evsD2, err := state.applyChunk(chunkFromTools("c", d2))
	if err != nil {
		t.Fatalf("applyChunk(d2) error = %v", err)
	}
	evsEnd, err := state.applyChunk(mustDecode(term))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	all := append(append(append(evsID, evsD1...), evsD2...), evsEnd...)

	var startCount, endCount int
	for _, ev := range all {
		if _, ok := ev.ToolCallStart(); ok {
			startCount++
		}
		if _, ok := ev.ToolCallEnd(); ok {
			endCount++
		}
	}
	if startCount != 1 {
		t.Errorf("got %d starts, want exactly 1 — identity-omitted later elements must not produce a second start (S-ATL-014)", startCount)
	}
	if endCount != 1 {
		t.Errorf("got %d ends, want exactly 1 (S-ATL-014)", endCount)
	}
}

// TestAccumulation_IdentityIdenticalRepeat_Tolerated covers S-ATL-015:
// every element repeating identical id/name alongside its fragment
// produces exactly one start and the end matches the concatenated
// fragment bytes.
func TestAccumulation_IdentityIdenticalRepeat_Tolerated(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_rep","function":{"name":"search","arguments":"{\"q\":"}}`
	repeat := `{"index":0,"id":"call_rep","function":{"name":"search","arguments":"\"a\"}"}}`
	term := `{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`

	evsID, err := state.applyChunk(chunkFromTools("c", id))
	if err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	evsRepeat, err := state.applyChunk(chunkFromTools("c", repeat))
	if err != nil {
		t.Fatalf("applyChunk(repeat) error = %v", err)
	}
	evsEnd, err := state.applyChunk(mustDecode(term))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	all := append(append(evsID, evsRepeat...), evsEnd...)

	var startCount int
	var endArgs []byte
	for _, ev := range all {
		if _, ok := ev.ToolCallStart(); ok {
			startCount++
		}
		if e, ok := ev.ToolCallEnd(); ok {
			endArgs = e.Arguments()
		}
	}
	if startCount != 1 {
		t.Errorf("got %d starts, want 1 — identical repeat is tolerated (S-ATL-015)", startCount)
	}
	if string(endArgs) != `{"q":"a"}` {
		t.Errorf("end args = %q, want %q (S-ATL-015)", endArgs, `{"q":"a"}`)
	}
}

// TestAccumulation_DifferingIdentity_FailsTyped covers S-ATL-016: a
// later element carrying a DIFFERENT non-empty id for an established
// call yields a typed malformed-response failure.
func TestAccumulation_DifferingIdentity_FailsTyped(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	first := `{"index":0,"id":"call_A1","function":{"name":"search","arguments":""}}`
	differ := `{"index":0,"id":"call_B2","function":{"name":"search","arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", first)); err != nil {
		t.Fatalf("applyChunk(first) error = %v", err)
	}
	_, err := state.applyChunk(chunkFromTools("c", differ))
	if err == nil {
		t.Fatal("applyChunk(differing id) error = nil, want errToolCallIdentityMismatch (S-ATL-016)")
	}
	if !strings.Contains(err.Error(), "identity") && !strings.Contains(err.Error(), "id") {
		t.Errorf("error %q does not name the identity mismatch (S-ATL-016)", err)
	}
}

// TestAccumulation_NeverSuppliedNameAtClose_Fails covers S-ATL-017: a
// call whose elements never carry `function.name` yields a typed
// failure at close, not an empty-name start event.
func TestAccumulation_NeverSuppliedNameAtClose_Fails(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	noName := `{"index":0,"id":"call_X","function":{"arguments":"{}"}}`
	term := `{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`

	if _, err := state.applyChunk(chunkFromTools("c", noName)); err != nil {
		t.Fatalf("applyChunk(no name) error = %v", err)
	}
	_, err := state.applyChunk(mustDecode(term))
	if err == nil {
		t.Fatal("applyChunk(terminal) error = nil, want errToolCallMissingIdentity (S-ATL-017)")
	}
}

// TestAccumulation_SingleChunkTwoElementArray covers S-ATL-021: a
// single chunk whose tool_calls array carries two elements with
// different indices and different fragments attributes each fragment
// to its own call — neither call receives both.
func TestAccumulation_SingleChunkTwoElementArray(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Identity for both calls in the same chunk.
	both := `{"index":0,"id":"call_A","function":{"name":"search","arguments":""}},{"index":1,"id":"call_B","function":{"name":"weather","arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", both)); err != nil {
		t.Fatalf("applyChunk(both identity) error = %v", err)
	}
	// Now: single chunk with two fragments for different indices.
	frags := `{"index":0,"function":{"arguments":"{\"q\":\"a\"}"}},{"index":1,"function":{"arguments":"{\"city\":\"b\"}"}}`
	evs, err := state.applyChunk(chunkFromTools("c", frags))
	if err != nil {
		t.Fatalf("applyChunk(frags) error = %v", err)
	}
	// Find the two Delta events; they must reference different blocks.
	var blocks []ai.BlockIndex
	for _, ev := range evs {
		if d, ok := ev.ToolCallDelta(); ok {
			blocks = append(blocks, d.BlockIndex())
		}
	}
	if len(blocks) != 2 {
		t.Fatalf("got %d delta blocks, want 2 — single-chunk two-element array (S-ATL-021)", len(blocks))
	}
	if blocks[0] == blocks[1] {
		t.Errorf("both deltas share block %d, want distinct blocks (S-ATL-021)", blocks[0])
	}
	// Close and verify end arguments.
	evsClose, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	for _, ev := range evsClose {
		if e, ok := ev.ToolCallEnd(); ok {
			got := string(e.Arguments())
			if got != `{"q":"a"}` && got != `{"city":"b"}` {
				t.Errorf("end args = %q, want one of the expected literals — single-chunk per-call attribution (S-ATL-021)", got)
			}
		}
	}
}

// TestAccumulation_ThreeCallRoundRobin covers S-ATL-020: three calls
// round-robined over three fragments each, every fragment carrying a
// call-unique marker byte, reconstruct independently. Markers are valid
// JSON fragments so ai.NewToolCall's well-formed-JSON gate accepts the
// assembled bytes.
func TestAccumulation_ThreeCallRoundRobin(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Three identities.
	ids := `{"index":0,"id":"call_X","function":{"name":"a","arguments":""}},{"index":1,"id":"call_Y","function":{"name":"b","arguments":""}},{"index":2,"id":"call_Z","function":{"name":"c","arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", ids)); err != nil {
		t.Fatalf("applyChunk(ids) error = %v", err)
	}
	// Three rounds of three fragments each.
	for round := 0; round < 3; round++ {
		var frags []string
		for idx := 0; idx < 3; idx++ {
			// Arguments is a JSON string in the chunk; the marker is the
			// JSON-escaped form of the fragment we want post-unquote.
			frags = append(frags, `{"index":`+itoa(idx)+`,"function":{"arguments":`+quoteJSONString(markerFor(idx, round))+`}}`)
		}
		if _, err := state.applyChunk(chunkFromTools("c", strings.Join(frags, ","))); err != nil {
			t.Fatalf("applyChunk(round %d) error = %v", round, err)
		}
	}
	evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	want0, want1, want2 := markerFor(0, 0)+markerFor(0, 1)+markerFor(0, 2), markerFor(1, 0)+markerFor(1, 1)+markerFor(1, 2), markerFor(2, 0)+markerFor(2, 1)+markerFor(2, 2)
	gotByBlock := map[ai.BlockIndex]string{}
	for _, ev := range evs {
		if e, ok := ev.ToolCallEnd(); ok {
			gotByBlock[e.BlockIndex()] = string(e.Arguments())
		}
	}
	wantSet := map[string]bool{want0: true, want1: true, want2: true}
	for block, got := range gotByBlock {
		if !wantSet[got] {
			t.Errorf("block %d end args %q does not match any expected literal (%v) — three-call round-robin (S-ATL-020)", block, got, []string{want0, want1, want2})
		}
	}
}

// quoteJSONString is a tiny helper: JSON-encode s as a JSON string
// literal (with surrounding quotes). Used to wrap argument-fragment
// markers as proper JSON string values in chunk payloads.
func quoteJSONString(s string) string {
	var out strings.Builder
	out.Grow(len(s) + 2)
	out.WriteByte('"')
	for i := 0; i < len(s); i++ {
		b := s[i]
		switch b {
		case '"':
			out.WriteString(`\"`)
		case '\\':
			out.WriteString(`\\`)
		default:
			out.WriteByte(b)
		}
	}
	out.WriteByte('"')
	return out.String()
}

// markerFor returns a per-call, per-round JSON-fragment marker — a
// distinct fragment whose concatenation with the other two rounds for
// the same call forms a valid JSON object literal. Each fragment is
// itself a valid JSON object, so ai.NewToolCall's well-formed-JSON gate
// accepts the assembled bytes.
func markerFor(callIdx, round int) string {
	// Each call's three fragments concatenate to:  {"k0":N,"k1":N+1,"k2":N+2}
	// where N is per-call: call 0 → N=1, call 1 → N=4, call 2 → N=7.
	switch round {
	case 0:
		return `{"k0":` + itoa(callIdx*3+1)
	case 1:
		return `,"k1":` + itoa(callIdx*3+2)
	default: // round 2
		return `,"k2":` + itoa(callIdx*3+3) + `}`
	}
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var out []byte
	for i > 0 {
		out = append([]byte{byte('0' + i%10)}, out...)
		i /= 10
	}
	return string(out)
}

// TestAccumulation_MissingIndex_FailsTyped covers S-ATL-011: a chunk
// whose tool_calls element carries id+arguments but NO `index` key
// yields a typed malformed-response failure.
func TestAccumulation_MissingIndex_FailsTyped(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	noIdx := `{"id":"call_no_idx","function":{"name":"search","arguments":"{}"}}`
	_, err := state.applyChunk(chunkFromTools("c", noIdx))
	if err == nil {
		t.Fatal("applyChunk(missing index) error = nil, want errToolElementMissingIndex (S-ATL-011)")
	}
}

// TestAccumulation_TerminalSeenThenToolElement_Fails covers the
// terminal-window check (D4): a tool_calls element appearing AFTER
// terminalSeen (a prior chunk set finish_reason) yields a typed
// failure (errToolDeltaAfterCloseToolCalls).
func TestAccumulation_TerminalSeenThenToolElement_Fails(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// First, set terminalSeen via a choice-less chunk? No — terminalSeen
	// is only set when a choice carries a finish_reason. Construct: a
	// text-only terminal chunk first, then a tool-call chunk.
	termFirst := mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":"stop"}]}`)
	if _, err := state.applyChunk(termFirst); err != nil {
		t.Fatalf("applyChunk(term first) error = %v", err)
	}
	// Now a tool_call element AFTER terminal chunk.
	_, err := state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_X","function":{"name":"search","arguments":"{}"}}`))
	if err == nil {
		t.Fatal("applyChunk(tool after terminal) error = nil, want errToolDeltaAfterCloseToolCalls (D4)")
	}
}

// TestAccumulation_FiveEventsOneCall_ShareBlock covers S-ATL-010: one
// call delivered as start, three fragments, and a close emits five
// events all reporting the same block index.
func TestAccumulation_FiveEventsOneCall_ShareBlock(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_K","function":{"name":"search","arguments":""}}`
	frag1 := `{"index":0,"function":{"arguments":"{\"a\":"}}`
	frag2 := `{"index":0,"function":{"arguments":"1,"}}`
	frag3 := `{"index":0,"function":{"arguments":"\"b\":2}"}}`
	evsID, err := state.applyChunk(chunkFromTools("c", id))
	if err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	var all []ai.Event
	all = append(all, evsID...)
	for _, f := range []string{frag1, frag2, frag3} {
		evs, err := state.applyChunk(chunkFromTools("c", f))
		if err != nil {
			t.Fatalf("applyChunk(frag) error = %v", err)
		}
		all = append(all, evs...)
	}
	evsEnd, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	all = append(all, evsEnd...)

	// Collect the block index from every Start/Delta/End for the only call.
	var blocks []ai.BlockIndex
	for _, ev := range all {
		switch ev.Kind() {
		case ai.EventKindToolCallStart, ai.EventKindToolCallDelta, ai.EventKindToolCallEnd:
			switch ev.Kind() {
			case ai.EventKindToolCallStart:
				if s, ok := ev.ToolCallStart(); ok {
					blocks = append(blocks, s.BlockIndex())
				}
			case ai.EventKindToolCallDelta:
				if d, ok := ev.ToolCallDelta(); ok {
					blocks = append(blocks, d.BlockIndex())
				}
			case ai.EventKindToolCallEnd:
				if e, ok := ev.ToolCallEnd(); ok {
					blocks = append(blocks, e.BlockIndex())
				}
			}
		}
	}
	if len(blocks) < 5 {
		t.Fatalf("got %d tool-call events, want ≥ 5 (start + 3 deltas + end = 5; S-ATL-010)", len(blocks))
	}
	for i := 1; i < len(blocks); i++ {
		if blocks[i] != blocks[0] {
			t.Errorf("event %d block = %d, want %d — all events for one call share one block (S-ATL-010)", i, blocks[i], blocks[0])
		}
	}
}

// TestAccumulation_CloseOnStopReason covers S-ATL-025: completeness is
// the terminal chunk, not the finish_reason value — a call followed by
// a chunk carrying "finish_reason":"stop" still closes with exactly
// one end event.
func TestAccumulation_CloseOnStopReason(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_S","function":{"name":"search","arguments":"{}"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	// A terminal chunk carrying "stop" instead of "tool_calls".
	evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term stop) error = %v", err)
	}
	var endCount int
	for _, ev := range evs {
		if _, ok := ev.ToolCallEnd(); ok {
			endCount++
		}
	}
	if endCount != 1 {
		t.Errorf("got %d ends on finish_reason=stop, want exactly 1 (S-ATL-025)", endCount)
	}
}

// TestAccumulation_EndByteEqualConcatenated covers S-ATL-022: a call
// receiving four fragments closes with exactly one end whose arguments
// are byte-equal to the concatenation of the four emitted deltas.
func TestAccumulation_EndByteEqualConcatenated(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_4","function":{"name":"search","arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	frags := []string{
		`{"index":0,"function":{"arguments":"{\"a\":"}}`,
		`{"index":0,"function":{"arguments":"1,"}}`,
		`{"index":0,"function":{"arguments":"\"b\":"}}`,
		`{"index":0,"function":{"arguments":"2}"}}`,
	}
	for _, f := range frags {
		if _, err := state.applyChunk(chunkFromTools("c", f)); err != nil {
			t.Fatalf("applyChunk(frag) error = %v", err)
		}
	}
	evs, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	var endBytes []byte
	for _, ev := range evs {
		if e, ok := ev.ToolCallEnd(); ok {
			endBytes = e.Arguments()
		}
	}
	if string(endBytes) != `{"a":1,"b":2}` {
		t.Errorf("end bytes = %q, want %q (S-ATL-022)", endBytes, `{"a":1,"b":2}`)
	}
}

// TestAccumulation_AscendingOrdinalEmission covers S-ATL-024: with two
// open calls followed by a terminal chunk, the emitted order is
// ToolCallEnd(ordinal 1), ToolCallEnd(ordinal 2), and ordinal = start
// order (NOT block index, NOT wire index).
func TestAccumulation_AscendingOrdinalEmission(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Identity in reverse wire-index order so ordinal != wire-index.
	idOrder := `{"index":1,"id":"call_2","function":{"name":"b","arguments":"{}"}},{"index":0,"id":"call_1","function":{"name":"a","arguments":"{}"}}`
	evsID, err := state.applyChunk(chunkFromTools("c", idOrder))
	if err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	evsEnd, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(term) error = %v", err)
	}
	all := append(evsID, evsEnd...)
	// Walk events: find Start events in order, capture their blocks;
	// then End events in order, capture their blocks. The end order
	// must match the start order (S-ATL-024).
	var startBlocks, endBlocks []ai.BlockIndex
	for _, ev := range all {
		switch ev.Kind() {
		case ai.EventKindToolCallStart:
			if s, ok := ev.ToolCallStart(); ok {
				startBlocks = append(startBlocks, s.BlockIndex())
			}
		case ai.EventKindToolCallEnd:
			if e, ok := ev.ToolCallEnd(); ok {
				endBlocks = append(endBlocks, e.BlockIndex())
			}
		}
	}
	if len(startBlocks) != 2 || len(endBlocks) != 2 {
		t.Fatalf("got (starts=%d, ends=%d), want 2 of each (S-ATL-024)", len(startBlocks), len(endBlocks))
	}
	for i := 0; i < 2; i++ {
		if startBlocks[i] != endBlocks[i] {
			t.Errorf("start[%d] block=%d, end[%d] block=%d — ascending-ordinal emission (S-ATL-024)", i, startBlocks[i], i, endBlocks[i])
		}
	}
}

// ensure context import is used
var _ = context.Background
