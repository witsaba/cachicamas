// AI-31.2 — usage field taxonomy: every detail object the wire reports
// reaches the completion event, with absent-vs-zero preserved
// (R-ACP-005, S-ACP-011/012/013/014), AI-13.4 cache semantics recorded
// rather than enforced (R-ACP-006, S-ACP-015/016/017), and the
// multi-frame last-wins merge pinned (R-ACP-007, S-ACP-018/019/020).
//
// # Pre-condition: landed S-ATS-055…062 are byte-identical
//
// R-ACP-009 / S-ACP-024 / S-ACP-025 freeze the AI-28.3 absent-vs-zero
// handling as a byte-identical contract — the slice-2a gate verifies
// it by diffing the existing tests against the branch's AI-28 landing,
// so this file APPENDS new tests only and never edits a prior one. A
// regression in the landed behaviour would surface as a diff in those
// eight assertions, not as a passing new test on broken semantics.

package openaicompat

import (
	"context"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestUsage_AllDetailFields_Present covers S-ACP-011: a usage chunk
// carrying prompt_tokens, completion_tokens, prompt_tokens_details
// (cached_tokens, cache_write_tokens) and completion_tokens_details
// (reasoning_tokens) reports each neutral field present with the
// wire's exact value — the all-present positive case for R-ACP-005.
func TestUsage_AllDetailFields_Present(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":50,\"total_tokens\":150,\"prompt_tokens_details\":{\"cached_tokens\":20,\"cache_write_tokens\":5},\"completion_tokens_details\":{\"reasoning_tokens\":10}}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-011)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 100 {
		t.Errorf("Input.Count() = (%d, %v), want (100, true) (S-ACP-011)", v, ok)
	}
	if v, ok := usage.Output.Count(); !ok || v != 50 {
		t.Errorf("Output.Count() = (%d, %v), want (50, true) (S-ACP-011)", v, ok)
	}
	if v, ok := usage.CacheRead.Count(); !ok || v != 20 {
		t.Errorf("CacheRead.Count() = (%d, %v), want (20, true) (S-ACP-011)", v, ok)
	}
	if v, ok := usage.CacheWrite.Count(); !ok || v != 5 {
		t.Errorf("CacheWrite.Count() = (%d, %v), want (5, true) (S-ACP-011)", v, ok)
	}
	if v, ok := usage.Reasoning.Count(); !ok || v != 10 {
		t.Errorf("Reasoning.Count() = (%d, %v), want (10, true) (S-ACP-011)", v, ok)
	}
}

// TestUsage_DetailFieldsAbsent_NegativeControl covers S-ACP-012: the
// same transcript with the two detail objects removed (no
// prompt_tokens_details, no completion_tokens_details) drains to
// CacheRead/CacheWrite/Reasoning all reporting absent while Input and
// Output are unchanged — the negative-control for S-ACP-011, in the
// S-ATS-057/060 form: the assertions that hold in S-ACP-011 must fail
// here.
func TestUsage_DetailFieldsAbsent_NegativeControl(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":50,\"total_tokens\":150}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-012)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 100 {
		t.Errorf("Input.Count() = (%d, %v), want (100, true) — Input/Output must be unchanged when detail objects are absent (S-ACP-012)", v, ok)
	}
	if v, ok := usage.Output.Count(); !ok || v != 50 {
		t.Errorf("Output.Count() = (%d, %v), want (50, true) (S-ACP-012)", v, ok)
	}
	if _, ok := usage.CacheRead.Count(); ok {
		t.Error("CacheRead.Count() ok = true, want false — absent detail object must report absent (S-ACP-012)")
	}
	if _, ok := usage.CacheWrite.Count(); ok {
		t.Error("CacheWrite.Count() ok = true, want false (S-ACP-012)")
	}
	if _, ok := usage.Reasoning.Count(); ok {
		t.Error("Reasoning.Count() ok = true, want false (S-ACP-012)")
	}
}

// TestUsage_CachedTokensZeroVsAbsent covers S-ACP-013: two transcripts
// identical except one omits cached_tokens and the other reports it as
// 0 produce usage records that are NOT equal, and the second reports
// CacheRead (0, true) while the first reports absent. The absent/zero
// discrimination that S-ATS-057 establishes for Input is extended to
// the new CacheRead field — pointer-nil-on-absence is the landed
// mechanism, used here for the nested detail object's leaf pointer.
func TestUsage_CachedTokensZeroVsAbsent(t *testing.T) {
	t.Parallel()

	omitted := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":50,\"total_tokens\":150,\"prompt_tokens_details\":{\"cache_write_tokens\":5},\"completion_tokens_details\":{\"reasoning_tokens\":10}}}\n\n")
	explicitZero := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":50,\"total_tokens\":150,\"prompt_tokens_details\":{\"cached_tokens\":0,\"cache_write_tokens\":5},\"completion_tokens_details\":{\"reasoning_tokens\":10}}}\n\n")

	drain := func(transcript string) ai.Usage {
		server := sseServer(t, transcript)
		defer server.Close()
		c := mustClient(t, server.URL)
		ch, err := c.Stream(context.Background(), validRequest(t))
		if err != nil {
			t.Fatalf("Stream() error = %v, want nil (S-ACP-013)", err)
		}
		return drainLastCompletion(t, ch).Usage()
	}

	usageOmitted := drain(omitted)
	usageZero := drain(explicitZero)

	if usageOmitted == usageZero {
		t.Errorf("usage records are equal (%+v), want NOT equal — omitted cached_tokens must be distinguishable from explicit 0 (S-ACP-013)", usageOmitted)
	}
	if _, ok := usageOmitted.CacheRead.Count(); ok {
		t.Error("omitted-field usage reports CacheRead present, want absent (S-ACP-013 premise)")
	}
	if v, ok := usageZero.CacheRead.Count(); !ok || v != 0 {
		t.Errorf("explicit-zero usage CacheRead.Count() = (%d, %v), want (0, true) (S-ACP-013 premise)", v, ok)
	}
}

// TestUsage_ReasoningContainedInOutput covers S-ACP-014: a usage chunk
// with completion_tokens: 100 and reasoning_tokens: 40 drains to
// Reasoning = 40, Output = 100, Output ≥ Reasoning holds, and Output
// is NOT adjusted by Reasoning — the U3 containment relation
// (completion_tokens ⊇ reasoning_tokens, per the chat schema's own
// rejected_prediction_tokens sentence) asserted rather than assumed.
// No subtraction happens; the two figures coexist with the containment
// held by the schema, not by arithmetic.
func TestUsage_ReasoningContainedInOutput(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":100,\"total_tokens\":200,\"completion_tokens_details\":{\"reasoning_tokens\":40}}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-014)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	output, ok := usage.Output.Count()
	if !ok {
		t.Fatal("Output.Count() ok = false, want true (S-ACP-014)")
	}
	reasoning, ok := usage.Reasoning.Count()
	if !ok {
		t.Fatal("Reasoning.Count() ok = false, want true (S-ACP-014)")
	}
	if reasoning != 40 {
		t.Errorf("Reasoning = %d, want 40 (S-ACP-014)", reasoning)
	}
	if output != 100 {
		t.Errorf("Output = %d, want 100 — must NOT be adjusted by Reasoning (S-ACP-014)", output)
	}
	if output < reasoning {
		t.Errorf("Output (%d) < Reasoning (%d), want Output ⊇ Reasoning per U3 (S-ACP-014)", output, reasoning)
	}
}

// TestUsage_RawMappingNoSubtraction covers S-ACP-015: a usage chunk
// with prompt_tokens: 1000 and cached_tokens: 800 drains to Input = 1000
// (NOT 200) and CacheRead = 800 — no subtraction performed, and an
// implementation that subtracted would report Input = 200 and fail
// here. This pins the "map raw, do not adjust" decision of R-ACP-006.
func TestUsage_RawMappingNoSubtraction(t *testing.T) {
	t.Parallel()

	transcript := usageTranscript("data: {\"id\":\"chatcmpl-u\",\"model\":\"m\",\"object\":\"chat.completion.chunk\",\"created\":1700000000,\"choices\":[],\"usage\":{\"prompt_tokens\":1000,\"completion_tokens\":50,\"total_tokens\":1050,\"prompt_tokens_details\":{\"cached_tokens\":800}}}\n\n")
	server := sseServer(t, transcript)
	defer server.Close()
	c := mustClient(t, server.URL)
	ch, err := c.Stream(context.Background(), validRequest(t))
	if err != nil {
		t.Fatalf("Stream() error = %v, want nil (S-ACP-015)", err)
	}
	usage := drainLastCompletion(t, ch).Usage()

	if v, ok := usage.Input.Count(); !ok || v != 1000 {
		t.Errorf("Input.Count() = (%d, %v), want (1000, true) — NO subtraction of cached_tokens is permitted (S-ACP-015)", v, ok)
	}
	if v, ok := usage.CacheRead.Count(); !ok || v != 800 {
		t.Errorf("CacheRead.Count() = (%d, %v), want (800, true) (S-ACP-015)", v, ok)
	}
}
