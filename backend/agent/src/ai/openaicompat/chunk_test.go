// AI-28.1.2 — chunk.go's own test suite: R-ATS-007 (S-ATS-025…027), R-ATS-009
// (S-ATS-033…035) and R-ATS-010's raw-string-strict finish-reason gate
// (S-ATS-036…039). Internal package, matching stream_test.go's own posture:
// every test here calls chunk.go's unexported wire-decode functions
// directly — the "direct through Decoder.Feed+mapper" seam design.md's
// Testing Strategy names, distinct from stream_test.go's own
// httptest.Server-driven seam.
//
// S-ATS-024 and S-ATS-028…032 (multi-chunk sequencing and block minting)
// live in stream_state_test.go: they exercise the mapper's own state across
// several chunks, not one chunk's own trichotomy in isolation.

package openaicompat

import (
	"encoding/json"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ---------------------------------------------------------------------
// R-ATS-007 — content trichotomy: JSON string / null / absent (S-ATS-025…027).
// ---------------------------------------------------------------------

// TestContentText_NullContent_NotPresent covers S-ATS-025's own premise: a
// JSON null content value contributes no text delta.
func TestContentText_NullContent_NotPresent(t *testing.T) {
	t.Parallel()

	text, present := contentText(json.RawMessage(`null`))
	if present {
		t.Errorf("contentText(null) present = true, want false (S-ATS-025)")
	}
	if text != "" {
		t.Errorf("contentText(null) text = %q, want empty (S-ATS-025)", text)
	}
}

// TestContentText_AbsentKey_NotPresent covers S-ATS-026: a delta with no
// "content" key at all — encoding/json leaves the RawMessage field nil —
// contributes no text delta, the same outcome as null but a different input
// shape.
func TestContentText_AbsentKey_NotPresent(t *testing.T) {
	t.Parallel()

	text, present := contentText(nil)
	if present {
		t.Errorf("contentText(nil) present = true, want false (S-ATS-026)")
	}
	if text != "" {
		t.Errorf("contentText(nil) text = %q, want empty (S-ATS-026)", text)
	}
}

// TestContentText_EmptyString_PresentAndEmpty covers S-ATS-027: a
// present-but-empty fragment is distinguishable from an absent one —
// present must be true even though text is "".
func TestContentText_EmptyString_PresentAndEmpty(t *testing.T) {
	t.Parallel()

	text, present := contentText(json.RawMessage(`""`))
	if !present {
		t.Fatal("contentText(\"\") present = false, want true (S-ATS-027)")
	}
	if text != "" {
		t.Errorf("contentText(\"\") text = %q, want empty string (S-ATS-027)", text)
	}
}

// TestContentText_OrdinaryString_ReturnsBytesUnmodified triangulates the
// trichotomy against a normal, non-empty value, so present=true is not
// merely a hardcoded response to the empty-string case above.
func TestContentText_OrdinaryString_ReturnsBytesUnmodified(t *testing.T) {
	t.Parallel()

	text, present := contentText(json.RawMessage(`"mundo"`))
	if !present {
		t.Fatal("contentText(\"mundo\") present = false, want true")
	}
	if text != "mundo" {
		t.Errorf("contentText(\"mundo\") text = %q, want %q", text, "mundo")
	}
}

// ---------------------------------------------------------------------
// R-ATS-009 — byte-exact reconstruction across a split multi-byte rune
// (S-ATS-033…035). Dialect-conventional robustness fixtures (raw bytes) —
// see spec.md's own labelling; not cited wire behavior (no C-claim).
// ---------------------------------------------------------------------

// TestContentText_SplitTwoByteRune_PreservesRawBytesUnmodified covers
// S-ATS-033/034: "café shop, abierto" with é's 2-byte UTF-8 encoding
// (0xC3 0xA9) split across the fixture boundary, with fifteen further bytes
// after the split point — non-vacuity: meaningful content after the split,
// per R-ATS-009's own discipline and the vacuous-pass catalogue's shape 4,
// so a producer that silently dropped or reordered the tail cannot pass.
func TestContentText_SplitTwoByteRune_PreservesRawBytesUnmodified(t *testing.T) {
	t.Parallel()

	first := json.RawMessage("\"caf\xc3\"")
	second := json.RawMessage("\"\xa9 shop, abierto\"")

	firstText, ok := contentText(first)
	if !ok {
		t.Fatal("contentText(first) present = false, want true (S-ATS-033/034)")
	}
	if firstText != "caf\xc3" {
		t.Errorf("first delta = %q, want %q unmodified — no replacement character, no truncation, no re-encoding (S-ATS-034)", firstText, "caf\xc3")
	}

	secondText, ok := contentText(second)
	if !ok {
		t.Fatal("contentText(second) present = false, want true (S-ATS-033)")
	}

	const want = "caf\xc3\xa9 shop, abierto"
	if got := firstText + secondText; got != want {
		t.Errorf("concatenated = %q, want %q byte-exact (S-ATS-033)", got, want)
	}
}

// TestContentText_SplitFourByteEmoji_ThreeDeltasReconstructByteExact covers
// S-ATS-035: a 4-byte emoji (0xF0 0x9F 0x98 0x80, U+1F600) split across
// three deltas as raw bytes, with trailing text after the third delta.
func TestContentText_SplitFourByteEmoji_ThreeDeltasReconstructByteExact(t *testing.T) {
	t.Parallel()

	parts := []json.RawMessage{
		json.RawMessage("\"\xf0\x9f\""),
		json.RawMessage("\"\x98\""),
		json.RawMessage("\"\x80 landed\""),
	}
	const want = "\xf0\x9f\x98\x80 landed"

	var got string
	for i, p := range parts {
		text, ok := contentText(p)
		if !ok {
			t.Fatalf("contentText(parts[%d]) present = false, want true (S-ATS-035)", i)
		}
		got += text
	}
	if got != want {
		t.Errorf("concatenated = %q, want %q byte-exact across 3 deltas (S-ATS-035)", got, want)
	}
	if len(parts) != 3 {
		t.Fatalf("test fixture carries %d part(s), want exactly 3 — the emitted delta count this scenario pins (S-ATS-035)", len(parts))
	}
}

// ---------------------------------------------------------------------
// R-ATS-010 — raw-string-strict finish-reason gate, before normalization
// (S-ATS-036…039, design.md D5/N-6).
// ---------------------------------------------------------------------

// TestRawStrictFinishReason_StopAndLength_MapToDistinctCanonicalReasons
// covers S-ATS-036/037: two of C2's five enum members map to distinguishable
// canonical FinishReason values.
func TestRawStrictFinishReason_StopAndLength_MapToDistinctCanonicalReasons(t *testing.T) {
	t.Parallel()

	stop, ok := rawStrictFinishReason("stop")
	if !ok {
		t.Fatal("rawStrictFinishReason(\"stop\") ok = false, want true (S-ATS-036)")
	}
	if stop != ai.FinishReasonStop {
		t.Errorf("rawStrictFinishReason(\"stop\") = %v, want FinishReasonStop (S-ATS-036)", stop)
	}

	length, ok := rawStrictFinishReason("length")
	if !ok {
		t.Fatal("rawStrictFinishReason(\"length\") ok = false, want true (S-ATS-037)")
	}
	if length != ai.FinishReasonLength {
		t.Errorf("rawStrictFinishReason(\"length\") = %v, want FinishReasonLength (S-ATS-037)", length)
	}
	if length == stop {
		t.Error("length and stop reported the same FinishReason, want distinguishable (S-ATS-037)")
	}
}

// TestRawStrictFinishReason_OutsideEnum_RejectedEvenWhenNormalizeWouldAccept
// covers S-ATS-039 and the gate's own raw-string-strict rule (D5/N-6): a
// value outside C2's enum is rejected, and — the case that actually
// distinguishes this gate from a bare ai.NormalizeFinishReason call — a
// case variant of a legal member is ALSO rejected, even though
// NormalizeFinishReason would happily map it.
func TestRawStrictFinishReason_OutsideEnum_RejectedEvenWhenNormalizeWouldAccept(t *testing.T) {
	t.Parallel()

	if _, ok := rawStrictFinishReason("quota_burned"); ok {
		t.Error("rawStrictFinishReason(\"quota_burned\") ok = true, want false — outside C2's enum (S-ATS-039)")
	}

	if reason := ai.NormalizeFinishReason("STOP"); reason != ai.FinishReasonStop {
		t.Fatalf("test premise broken: ai.NormalizeFinishReason(\"STOP\") = %v, want FinishReasonStop so this case is meaningful", reason)
	}
	if _, ok := rawStrictFinishReason("STOP"); ok {
		t.Error("rawStrictFinishReason(\"STOP\") ok = true, want false — raw-string-strict, no case fold (D5, N-6)")
	}
}

// TestDecodeChunk_ReadsIdentityAndChoice0 proves decodeChunk's own minimal
// contract: id/model/choice-0 fields land in wireChunk unmodified, byte for
// byte — the structural counterpart to R-ATS-004's byte-exactness, now
// through chunk.go's own decode path rather than stream.go's slice-1 one.
func TestDecodeChunk_ReadsIdentityAndChoice0(t *testing.T) {
	t.Parallel()

	data := []byte(`{"id":"chatcmpl-Xq7","model":"gizmo-4o","object":"chat.completion.chunk","created":1700000000,"choices":[{"index":0,"delta":{"content":"hi"},"finish_reason":null}]}`)
	chunk, err := decodeChunk(data)
	if err != nil {
		t.Fatalf("decodeChunk() error = %v, want nil", err)
	}
	if chunk.ID != "chatcmpl-Xq7" {
		t.Errorf("chunk.ID = %q, want %q", chunk.ID, "chatcmpl-Xq7")
	}
	if chunk.Model != "gizmo-4o" {
		t.Errorf("chunk.Model = %q, want %q", chunk.Model, "gizmo-4o")
	}
	choice, ok := chunk.choice0()
	if !ok {
		t.Fatal("chunk.choice0() ok = false, want true")
	}
	if choice.FinishReason != nil {
		t.Errorf("choice.FinishReason = %v, want nil (still null mid-stream)", *choice.FinishReason)
	}
	text, present := contentText(choice.Delta.Content)
	if !present || text != "hi" {
		t.Errorf("contentText(choice.Delta.Content) = (%q, %v), want (\"hi\", true)", text, present)
	}
}

// TestDecodeChunk_EmptyChoicesArray_NoChoice0 covers the usage-chunk shape
// (C4): an empty choices array decodes cleanly and choice0() reports none.
func TestDecodeChunk_EmptyChoicesArray_NoChoice0(t *testing.T) {
	t.Parallel()

	data := []byte(`{"id":"chatcmpl-x","model":"m","object":"chat.completion.chunk","created":1700000000,"choices":[]}`)
	chunk, err := decodeChunk(data)
	if err != nil {
		t.Fatalf("decodeChunk() error = %v, want nil", err)
	}
	if _, ok := chunk.choice0(); ok {
		t.Error("chunk.choice0() ok = true, want false for an empty choices array")
	}
}
