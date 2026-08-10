// AI-31.2 — impossible-arithmetic probe and multi-frame merge pin.
//
// TestUsage_ImpossibleArithmeticUnderExclusivity (S-ACP-017, the
// strongest raw-mapping proof an authored fixture can carry).
//
// This test wires a usage chunk whose cache-bearing values are
// arithmetically impossible under any exclusive reading of Input:
// prompt_tokens = 500 with cached_tokens = 800, so no subtraction can
// produce a non-negative Input and no consistency arithmetic can hold.
// The probe asserts the adapter reports Input = 500 and CacheRead = 800
// raw, with no error, no clamp, no reconciliation, no negative value.
//
// If the adapter enforced ANY consistency arithmetic anywhere on the
// path (subtract cached_tokens from prompt_tokens, clamp a negative,
// reconcile cache fields, etc.), it would have to fail, clamp, or
// adjust HERE, and this test would fail. The test therefore proves the
// stronger claim that S-ACP-015 cannot: not merely that subtraction did
// not happen on one plausible input, but that no consistency arithmetic
// is enforced anywhere. A path that fails this test by returning an
// error or a clamped value is a path that implemented an inference the
// spec deliberately refuses to record as fact.
//
// TestUsage_MultiplePopulatedFrames_LastWinsNoFold (S-ACP-019, the
// dialect-conventional pin for R-ACP-007's multi-frame case).
//
// R-ACP-012's spec table names this test function as the inline-literal
// pinning fixture for the multi-frame usage dialect-conventional label.
// Three populated usage frames with prompt_tokens 10, 20, 30 drain to
// Input = 30, NOT 60 — the landed D10 last-populated-wins,
// wholesale-overwrite merge (design.md D10, stream_state.go) holds.
// This is the ONE dialect-conventional claim AI-31 introduces (per
// R-ACP-012's rev-2 table); the test function named here is the pin.

package openaicompat

import (
	"context"
	"strconv"
	"testing"
)

// TestUsage_ImpossibleArithmeticUnderExclusivity covers S-ACP-017:
// the arithmetically-impossible-under-exclusivity probe. prompt_tokens
// is 500 and cached_tokens is 800 — no exclusive reading of Input can
// be derived (the subtraction 500 - 800 = -300 is negative) and every
// consistency arithmetic on the path would have to fail, clamp or
// adjust here. The adapter reports Input = 500 and CacheRead = 800
// raw, no error or rejection occurs, and no count is clamped,
// reconciled or made negative.
func TestUsage_ImpossibleArithmeticUnderExclusivity(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":500,\"completion_tokens\":50,\"total_tokens\":550,\"prompt_tokens_details\":{\"cached_tokens\":800}}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil — the impossible-arithmetic fixture must still complete (S-ACP-017)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 500 {
		t.Errorf("Input.Count() = (%d, %v), want (500, true) — the raw prompt_tokens value must be preserved unchanged (S-ACP-017)", v, ok)
	}
	if v, ok := usage.CacheRead.Count(); !ok || v != 800 {
		t.Errorf("CacheRead.Count() = (%d, %v), want (800, true) — the raw cached_tokens value must be preserved unchanged (S-ACP-017)", v, ok)
	}
}

// TestUsage_SingleFrame_U6Shape covers S-ACP-018: a transcript with
// content chunks carrying usage:null followed by exactly one populated
// usage chunk with empty choices drains to a completion whose usage is
// that chunk's values exactly, and the preceding nulls neither reset
// nor contribute — U6's single-frame shape, behaving exactly as
// today. This is the baseline test that the multi-frame case
// (S-ACP-019) extends; it must continue to pass once that case lands.
func TestUsage_SingleFrame_U6Shape(t *testing.T) {
	t.Parallel()

	transcript := "" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"hi\"},\"finish_reason\":null}],\"usage\":null}\n\n" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":null}\n\n" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":42,\"completion_tokens\":17,\"total_tokens\":59}}\n\n" +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-018)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 42 {
		t.Errorf("Input.Count() = (%d, %v), want (42, true) — single populated frame must drive Input (S-ACP-018)", v, ok)
	}
	if v, ok := usage.Output.Count(); !ok || v != 17 {
		t.Errorf("Output.Count() = (%d, %v), want (17, true) (S-ACP-018)", v, ok)
	}
}

// TestUsage_MultiplePopulatedFrames_LastWinsNoFold covers S-ACP-019:
// three populated usage frames with prompt_tokens 10, 20, 30 drain to
// Input = 30 — NOT 60 (which would be a fold/sum). The landed D10
// last-populated-wins, wholesale-overwrite rule (stream_state.go's
// `s.usage = usageFromWire(*chunk.Usage)`) holds; this is the
// pinning fixture for R-ACP-012's single dialect-conventional label
// (multi-frame usage, see spec/table).
//
// All three usage frames carry an empty choices array (U6's
// usage-chunk shape) and identical id/model/created so they pass
// hasRequiredFields and the second-response-start check.
func TestUsage_MultiplePopulatedFrames_LastWinsNoFold(t *testing.T) {
	t.Parallel()

	frame := func(n int64) string {
		return "data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":" +
			strconv.FormatInt(n, 10) +
			",\"completion_tokens\":1,\"total_tokens\":" + strconv.FormatInt(n+1, 10) + "}}\n\n"
	}
	transcript := "" +
		"data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":null}\n\n" +
		frame(10) + frame(20) + frame(30) +
		"data: [DONE]\n\n"
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-019)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 30 {
		t.Errorf("Input.Count() = (%d, %v), want (30, true) — last populated frame must win wholesale, NOT fold to 60 (S-ACP-019)", v, ok)
	}
}
