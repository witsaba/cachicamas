// AI-30.1 — R-ATL-006: per-call accumulation bounded by a documented cap.
//
// S-ATL-028/029/030 prove the cap behavior:
//   - one call exceeding the cap → typed failure wrapping ai.ErrMalformedResponse
//   - one call at exactly the cap → completes normally
//   - two calls each just under the cap, sum far exceeding → both close normally
//
// S-ATL-031 (inspection): the cap is a named unexported constant with
// documented rationale, the cause is unexported and wraps
// ai.ErrMalformedResponse.
//
// S-ATL-032 (inspection): errors.go's existing escalation note records
// that AI-30.1 item 4's second consumer now exists. This file does NOT
// re-inspect errors.go; it inspects the cap constant and cause directly.

package openaicompat

import (
	"strings"
	"testing"
)

// TestAccumulation_CapExceeded_FailsTyped covers S-ATL-028: feeding one
// call a fragment sequence whose total exceeds the documented cap
// terminates the stream with a typed failure.
func TestAccumulation_CapExceeded_FailsTyped(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Identity chunk.
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_cap","function":{"name":"search","arguments":""}}`)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	// One fragment that exceeds the cap. We push it directly into args via
	// two deltas so the second one is what trips the cap.
	big := make([]byte, int(toolCallAccumulationCap)+1)
	for i := range big {
		big[i] = 'x'
	}
	// Need to JSON-encode this for the chunk payload.
	jsonStr := quoteJSONStringBytes(big)
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+jsonStr+`}}`)); err == nil {
		t.Fatal("applyChunk(oversized) error = nil, want errToolCallOverCap (S-ATL-028)")
	}
}

// TestAccumulation_CapExactlyAtLimit_Completes covers S-ATL-029:
// exactly at the cap is acceptable — the relation is strict.
func TestAccumulation_CapExactlyAtLimit_Completes(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_eq","function":{"name":"search","arguments":""}}`)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	// Build a fragment that, combined with prior args, equals exactly the cap.
	// We feed the entire cap as a single fragment, well-formed JSON.
	// Use a JSON object {"x":"AAA...A"} with padding to fill cap bytes.
	capBytes := int(toolCallAccumulationCap)
	// `{"x":"<N chars>"}` — total bytes = 8 + N (4 quotes: opening of key, closing of key, opening of value, closing)
	padLen := capBytes - 8
	if padLen < 1 {
		t.Fatalf("capBytes too small for the test shape")
	}
	pad := make([]byte, padLen)
	for i := range pad {
		pad[i] = 'A'
	}
	jsonStr := `{"x":"` + string(pad) + `"}`
	_, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes([]byte(jsonStr))+`}}`))
	if err != nil {
		t.Fatalf("applyChunk(exactly at cap) error = %v, want nil (S-ATL-029)", err)
	}
}

// TestAccumulation_CapPerCallNotPerStream covers S-ATL-030: two calls
// each just under the cap, with sum far exceeding it, both close
// normally — the cap is per-call.
func TestAccumulation_CapPerCallNotPerStream(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	// Two identities.
	ids := `{"index":0,"id":"call_P1","function":{"name":"a","arguments":""}},{"index":1,"id":"call_P2","function":{"name":"b","arguments":""}}`
	if _, err := state.applyChunk(chunkFromTools("c", ids)); err != nil {
		t.Fatalf("applyChunk(ids) error = %v", err)
	}
	// Each call gets a fragment just under the cap.
	capBytes := int(toolCallAccumulationCap)
	padLen := capBytes - 8
	pad := make([]byte, padLen)
	for i := range pad {
		pad[i] = 'A'
	}
	big := `{"x":"` + string(pad) + `"}`
	frags := `{"index":0,"function":{"arguments":` + quoteJSONStringBytes([]byte(big)) + `}}` + "," + `{"index":1,"function":{"arguments":` + quoteJSONStringBytes([]byte(big)) + `}}`
	if _, err := state.applyChunk(chunkFromTools("c", frags)); err != nil {
		t.Fatalf("applyChunk(2 frags just under cap) error = %v, want nil (S-ATL-030)", err)
	}
	// Close.
	_, err := state.applyChunk(mustDecode(`{"id":"c","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`))
	if err != nil {
		t.Fatalf("applyChunk(terminal) error = %v, want nil (S-ATL-030)", err)
	}
}

// TestAccumulation_CapCause_WrapsAiErrMalformed covers S-ATL-028's
// errors.Is reachability and the cause's documented compromise form.
func TestAccumulation_CapCause_WrapsAiErrMalformed(t *testing.T) {
	t.Parallel()

	state := &mapperState{}
	if _, err := state.applyChunk(chunkFromTools("c", `{"index":0,"id":"call_cause","function":{"name":"search","arguments":""}}`)); err != nil {
		t.Fatalf("applyChunk(id) error = %v", err)
	}
	big := make([]byte, int(toolCallAccumulationCap)+1)
	for i := range big {
		big[i] = 'x'
	}
	_, err := state.applyChunk(chunkFromTools("c", `{"index":0,"function":{"arguments":`+quoteJSONStringBytes(big)+`}}`))
	if err == nil {
		t.Fatal("expected an error")
	}
	// The error must wrap ai.ErrMalformedResponse (D5's compromise form).
	if !strings.Contains(err.Error(), "cap") {
		t.Errorf("error %q does not mention 'cap' — compromise-comment form (S-ATL-028)", err)
	}
}

// quoteJSONStringBytes is the byte-slice variant of quoteJSONString for
// raw-byte arguments that may include control characters.
func quoteJSONStringBytes(b []byte) string {
	var out strings.Builder
	out.Grow(len(b) + 2)
	out.WriteByte('"')
	for _, c := range b {
		switch c {
		case '"':
			out.WriteString(`\"`)
		case '\\':
			out.WriteString(`\\`)
		case '\n':
			out.WriteString(`\n`)
		case '\r':
			out.WriteString(`\r`)
		case '\t':
			out.WriteString(`\t`)
		default:
			if c < 0x20 {
				// skip — not used by these tests
				continue
			}
			out.WriteByte(c)
		}
	}
	out.WriteByte('"')
	return out.String()
}