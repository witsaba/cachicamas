// AI-30.3 — R-ATL-008: reassembled arguments are byte-identical to the
// concatenated fragments, and nothing re-marshals them.
//
// S-ATL-038…043:
//   038: escape sequences (`\\`, `\"`, `\n`, `é`) byte-equal
//   039: extreme numerics byte-equal
//   040: duplicated spacing/non-alphabetical key order preserved
//   041: split-inside-a-JSON-escape reassembles intact
//   042: two independent comparisons (literal vs concatenated deltas)
//   043 (inspection): no json.Marshal/json.Compact/json.Indent/numeric round-trip

package openaicompat

import (
	"testing"
)

// TestByteFidelity_EscapeSequences covers S-ATL-038: a call whose
// fragments concatenate to {"path":"C:\\tmp\\x","quote":"say \"hi\"",
// "nl":"a\nb","u":"café"} produces an end event whose Arguments() is
// byte-equal to that decoded literal.
func TestByteFidelity_EscapeSequences(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_BF","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	// The full decoded literal, fed as a single fragment.
	full := `{"path":"C:\\tmp\\x","quote":"say \"hi\"","nl":"a\nb","u":"café"}`
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(full))+`}}`)); err != nil {
		t.Fatalf("applyChunk(full) error = %v", err)
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
	if string(endArgs) != full {
		t.Errorf("end args byte-equal mismatch (S-ATL-038):\n  got:  %q\n  want: %q", endArgs, full)
	}
}

// TestByteFidelity_ExtremeNumerics covers S-ATL-039: a call whose
// fragments concatenate to a JSON object with extreme numerics
// reproduces every digit and sign byte for byte — no float round-trip.
func TestByteFidelity_ExtremeNumerics(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_X","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	full := `{"big":123456789012345678901234567890,"tiny":1e-320,"neg":-0,"prec":3.141592653589793238462643383279}`
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(full))+`}}`)); err != nil {
		t.Fatalf("applyChunk(full) error = %v", err)
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
	if string(endArgs) != full {
		t.Errorf("end args byte-equal mismatch (S-ATL-039):\n  got:  %q\n  want: %q", endArgs, full)
	}
}

// TestByteFidelity_NonAlphabeticalKeyOrder covers S-ATL-040: a call
// whose fragments concatenate to {"b":2,"a":1} preserves the b-before-a
// order — no key reordering.
func TestByteFidelity_NonAlphabeticalKeyOrder(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_O","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	full := `{ "b" : 2,  "a":1 }`
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(full))+`}}`)); err != nil {
		t.Fatalf("applyChunk(full) error = %v", err)
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
	if string(endArgs) != full {
		t.Errorf("end args byte-equal mismatch (S-ATL-040):\n  got:  %q\n  want: %q", endArgs, full)
	}
}

// TestByteFidelity_SplitInsideEscape covers S-ATL-041: a call's
// argument text is split inside a JSON escape — one fragment ends
// with a single backslash and the next starts with the rest of the
// escape sequence. The concatenation must carry the intact escape.
func TestByteFidelity_SplitInsideEscape(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	id := `{"index":0,"id":"call_S","function":{"name":"search"}}`
	if _, err := state.applyChunk(chunkFromTools("c", id)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	// First fragment decoded: ... ending with a single backslash byte.
	// Wire-level (JSON-string-encoded): the backslash is escaped as `\\`
	// so the raw wire value ends with two `\\` bytes.
	half1 := `{"k":"val\` // 10 bytes; ends with single backslash byte
	half2 := `u00e9"}`    // 7 bytes; continues the escape

	// After unquote + concatenation, the decoded content is:
	//   {"k":"val\u00e9"}
	// (the `\u00e9` is 6 literal ASCII bytes — a JSON escape the
	// adapter must NOT decode into UTF-8; byte-fidelity holds the
	// bytes verbatim).
	want := `{"k":"val\u00e9"}`

	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(half1))+`}}`)); err != nil {
		t.Fatalf("applyChunk(half1) error = %v", err)
	}
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(half2))+`}}`)); err != nil {
		t.Fatalf("applyChunk(half2) error = %v", err)
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
	if string(endArgs) != want {
		t.Errorf("end args byte-equal mismatch (S-ATL-041):\n  got:  %q\n  want: %q", endArgs, want)
	}
}

// TestByteFidelity_TwoIndependentComparisons covers S-ATL-042: two
// independent comparisons of the same call — one against the fixture
// literal, the other against the concatenated delta fragments —
// neither derived from the other.
func TestByteFidelity_TwoIndependentComparisons(t *testing.T) {
	t.Parallel()

	// Feed identity chunk + two fragments + terminal chunk all through
	// drainEventsFromMapper — the deltas are emitted at fragment-apply
	// time (not at terminal), so to compare against what the producer
	// actually emitted we must walk the full event stream across all
	// chunks, not just the terminal chunk's own events.
	idChunk := chunkFromTools("c", `{"index":0,"id":"call_T","function":{"name":"search"}}`)
	frag1 := `{"a":`
	frag2 := `"value"}`
	frag1Chunk := chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(frag1))+`}}`)
	frag2Chunk := chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(frag2))+`}}`)
	termChunk := mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`)
	all := drainEventsFromMapper(t, idChunk, frag1Chunk, frag2Chunk, termChunk)

	var endArgs []byte
	var emittedFrags []byte
	for _, ev := range all {
		if e, ok := ev.ToolCallEnd(); ok {
			endArgs = e.Arguments()
		}
		if d, ok := ev.ToolCallDelta(); ok {
			emittedFrags = append(emittedFrags, d.Fragment()...)
		}
	}

	// First independent comparison: end args byte-equal to fixture literal.
	wantLiteral := `{"a":"value"}`
	if string(endArgs) != wantLiteral {
		t.Errorf("end args = %q, want fixture literal %q (S-ATL-042, comparison 1)", endArgs, wantLiteral)
	}

	// Second independent comparison: end args byte-equal to the
	// fragments the producer actually emitted during accumulation
	// (collected live from the ToolCallDelta events, NOT derived
	// from the source fixture). This is the discriminating
	// comparison: a producer that re-marshalled or canonicalised
	// would still match the source-derived literal in comparison 1
	// but would diverge from the live-emitted fragments here.
	if string(endArgs) != string(emittedFrags) {
		t.Errorf("end args = %q, want emitted fragments %q (S-ATL-042, comparison 2)", endArgs, emittedFrags)
	}
}
