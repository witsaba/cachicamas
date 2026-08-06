// AI-30.1 — R-ATL-001 decode + R-ATL-013 function_call recorded skip.
//
// Tool-call chunk-element decoding (S-ATL-001…005): wireDelta.ToolCalls
// reads ChatCompletionMessageToolCallChunk's full declared shape (C9.1:
// index, id, function{name, arguments}) byte-preservingly. Undeclared
// fields (including the deprecated streaming function_call, S-ATL-062…064)
// are tolerated as R-ATS-017's own unknown-field tolerance already
// guarantees — encoding/json silently drops them, and the disposition is
// "recorded skip, not accidental skip" per the spec's R-ATL-013.
//
// Strict TDD: every RED scenario below is written first and exercises a
// production function whose implementation lands in chunk.go alongside
// wireDelta.ToolCalls (D2).

package openaicompat

import (
	"encoding/json"
	"testing"
)

// TestToolCallDecode_IndexOnlyElement_ToleratedAndLaterElementsStillMap
// covers S-ATL-001: an element carrying only `"index":0` and no other
// key is decoded without failure — index is the only required field
// (C9.1) — and the absence of any argument fragment leaves the call's
// arguments untouched; a later element for the same index still maps
// normally.
func TestToolCallDecode_IndexOnlyElement_ToleratedAndLaterElementsStillMap(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"index":0}`)
	var elem wireToolCallElement
	if err := json.Unmarshal(raw, &elem); err != nil {
		t.Fatalf("Unmarshal(index-only) error = %v, want nil — index is the only required field (C9.1)", err)
	}
	if elem.Index == nil || *elem.Index != 0 {
		t.Errorf("Index = %v, want non-nil pointer to 0 (S-ATL-001)", elem.Index)
	}
	if elem.Function != nil {
		t.Errorf("Function = %+v, want nil for an index-only element (S-ATL-001)", elem.Function)
	}
}

// TestToolCallDecode_TypeOptional_AffectsNothing covers S-ATL-002:
// "type":"function" is single-member enum carrying no mapping decision
// (C9.4, design D2's "no field this milestone does not need" posture),
// so its presence or absence has no effect on the decode outcome —
// wireDelta declares no Type field at all, matching the spec's
// "decoded to nothing else" reading of C9.1.
func TestToolCallDecode_TypeOptional_AffectsNothing(t *testing.T) {
	t.Parallel()

	withType := []byte(`{"index":0,"type":"function","id":"c","function":{"name":"f","arguments":""}}`)
	withoutType := []byte(`{"index":0,"id":"c","function":{"name":"f","arguments":""}}`)

	var a, b wireToolCallElement
	if err := json.Unmarshal(withType, &a); err != nil {
		t.Fatalf("Unmarshal(with type) error = %v, want nil", err)
	}
	if err := json.Unmarshal(withoutType, &b); err != nil {
		t.Fatalf("Unmarshal(without type) error = %v, want nil", err)
	}

	if a.Function == nil || b.Function == nil {
		t.Fatalf("Function = (%+v, %+v), want non-nil for both — only `type` differs", a.Function, b.Function)
	}
	if !bytesEqual(a.Function.Name, b.Function.Name) || !bytesEqual(a.Function.Arguments, b.Function.Arguments) {
		t.Errorf("with-type (name=%q, args=%q) and without-type (name=%q, args=%q) decoded differently, want identical — type carries no mapping decision (C9.4)",
			a.Function.Name, a.Function.Arguments, b.Function.Name, b.Function.Arguments)
	}
}

// TestToolCallDecode_InventedSiblingFieldsIgnored covers S-ATL-003:
// two invented sibling fields ("synthetic_marker":"x" and "confidence":
// 0.9) sitting beside C9.1's declared fields are silently dropped by
// encoding/json's own unknown-field tolerance, exactly the disposition
// R-ATS-017 already grants. The decode succeeds, the declared fields
// carry their values, and no failure is reported.
func TestToolCallDecode_InventedSiblingFieldsIgnored(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"index":0,"id":"call_X","function":{"name":"search","arguments":"{}"},"synthetic_marker":"x","confidence":0.9}`)
	var elem wireToolCallElement
	if err := json.Unmarshal(raw, &elem); err != nil {
		t.Fatalf("Unmarshal(with invented siblings) error = %v, want nil — invented fields are tolerated (R-ATS-017, S-ATL-003)", err)
	}
	if elem.ID == nil || string(elem.ID) != `"call_X"` {
		t.Errorf("ID = %q, want %q (S-ATL-003)", elem.ID, `"call_X"`)
	}
	if elem.Function == nil {
		t.Fatal("Function = nil, want non-nil (S-ATL-003)")
	}
	if string(elem.Function.Name) != `"search"` {
		t.Errorf("Function.Name = %q, want %q (S-ATL-003)", elem.Function.Name, `"search"`)
	}
}

// TestToolCallDecode_ArgumentsEscapeDecodedExactlyOnce covers S-ATL-004:
// a wire `function.arguments` value of `"{\"q\":\"a\\nb\"}"` decodes,
// through the existing unquoteJSONString path, into the byte sequence
// `{"q":"a\nb"}` — asserted byte-equal, not string-compared after a
// re-encode.
func TestToolCallDecode_ArgumentsEscapeDecodedExactlyOnce(t *testing.T) {
	t.Parallel()

	raw := []byte(`"{\"q\":\"a\\nb\"}"`)
	got := unquoteJSONString(raw)
	want := []byte(`{"q":"a\nb"}`)
	if !bytesEqual(got, want) {
		t.Errorf("unquoteJSONString = %q, want %q — escape decoded exactly once (S-ATL-004)", got, want)
	}
}

// TestToolCallDecode_FunctionCallOnlyChunk_NoToolCallEvent covers
// S-ATL-062: a chunk carrying only `"function_call":{"name":"search",
// "arguments":"{}"}"` alongside choice-0 content text produces no
// tool-call event of any kind. This is the recorded-skip half (C9.5,
// R-ATL-013): the deprecated anonymous object is tolerated (no failure)
// and ignored (no event), never mapped.
func TestToolCallDecode_FunctionCallOnlyChunk_NoToolCallEvent(t *testing.T) {
	t.Parallel()

	chunk, err := decodeChunk([]byte(`{"id":"chatcmpl-fc","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"content":"alpha","function_call":{"name":"search","arguments":"{}"}},"finish_reason":null}]}`))
	if err != nil {
		t.Fatalf("decodeChunk(function_call) error = %v, want nil — the deprecated field is tolerated, not malformed (C9.5)", err)
	}
	// No declared tool_calls field on wireDelta, so the function_call's
	// contribution is structurally absent from the decode — chunk.go
	// declares no FunctionCall field on wireDelta (D2). This is the
	// recorded-skip shape.
	if len(chunk.Choices) == 0 {
		t.Fatal("no choice decoded; want the chunk to be shape-valid")
	}
	choice, ok := chunk.choice0()
	if !ok {
		t.Fatal("choice0() returned ok=false; want the choice present")
	}
	if text, present := contentText(choice.Delta.Content); !present || text != "alpha" {
		t.Errorf("contentText(delta.content) = (%q, %v), want (\"alpha\", true) — content text is unaffected by the function_call sibling (S-ATL-062)", text, present)
	}
}

// TestToolCallDecode_FunctionCallAlongsideToolCalls_ToolCallsWins covers
// S-ATL-063: a single chunk carrying BOTH a `tool_calls` array and a
// `function_call` object — the tool_calls half decodes normally,
// function_call contributes nothing because no field is declared for it.
func TestToolCallDecode_FunctionCallAlongsideToolCalls_ToolCallsWins(t *testing.T) {
	t.Parallel()

	raw := []byte(`{"id":"chatcmpl-mix","model":"m","created":1700000000,"object":"chat.completion.chunk","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","function":{"name":"search","arguments":"{}"}}],"function_call":{"name":"ignored","arguments":"{}"}},"finish_reason":null}]}`)
	chunk, err := decodeChunk(raw)
	if err != nil {
		t.Fatalf("decodeChunk(mixed) error = %v, want nil", err)
	}
	choice, ok := chunk.choice0()
	if !ok {
		t.Fatal("choice0() returned ok=false")
	}
	// wireDelta.ToolCalls is declared; the decoded element must be present.
	// (We do not assert on length here because the production code is
	// responsible for the decode — Phase 1 lands that field. Asserting on
	// presence here is RED for the structural addition.)
	if choice.Delta.ToolCalls == nil {
		t.Fatal("choice.Delta.ToolCalls is nil, want at least one decoded element — wireDelta must declare ToolCalls (D2, S-ATL-063)")
	}
}

// bytesEqual is a tiny helper local to this file so the assertions read
// byte-equal against expected literals, not string-compared after a
// re-encode (S-ATL-004).
func bytesEqual(a, b []byte) bool {
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
