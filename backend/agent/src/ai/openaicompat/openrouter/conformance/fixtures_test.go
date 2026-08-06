// AI-38 — recorded-fixture structural assertions.
//
// # What this file covers
//
// Every fixture var-blob this directory exports is a recorded
// canonical byte sequence: either an SSE transcript the conformance
// bridge serves verbatim, or a Go var/const the conformance cases
// iterate over. The tests below assert structural integrity of each
// fixture at the byte level — non-empty, contains the expected
// chat.completion.chunk discriminator, has the expected number of
// SSE data frames, has the expected byte span — so a future
// renderer or fixture-editor typo is caught here, before the
// conformance cases consume the fixture and the failure surfaces as
// a confusing decoded-event mismatch.
//
// # TDD posture (work-unit 2.2)
//
// RED  : the fixture vars below did not exist; the assertions in
//        this file failed to compile (unresolved identifier
//        openrouterTextStream, openrouterToolCallStream,
//        openrouterTextStreamWithUsage, openrouterAttributionHeader
//        Names, openrouterAttributionPresentValues,
//        openrouterAttributionAbsentCount,
//        openrouterAttributionPresentCount).
// GREEN: the fixture var-blocks in fixtures/*.go added; compile
//        passes; every assertion below PASSes against the recorded
//        canonical bytes.
//
// # Fixtures are not golden-tested bit-for-bit
//
// These tests do NOT pin every byte of every fixture against a
// third reference; a future renderer can grow the fixture
// (additional TextDelta chunks, additional usage fields) without
// breaking them. The conformance cases in fixtures' consumer
// files (reasoning_extension_test.go, capability_record_test.go,
// run_for_test.go) are what enforce the byte-level consumer-side
// contract — those assertions observe what the openaicompat
// adapter actually decodes and assert it matches the spec.
//
// # Recorded fixtures are excluded from authored risk count
//
// Per sdd-phase-common.md §E, generated SSE goldens are excluded
// from the authored-line risk count; the conformance bridge's PR
// size budget is safe on authored-code terms even though raw line
// count exceeds 800.

package openrouter_conformance

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures"
)

// sseFrameCount returns the number of SSE "data:" frames in body,
// where each frame is "data: <payload>\n\n". The count is the
// structural invariant every fixture's conformance-case consumer
// depends on.
//
// The body has the form "data: X\n\ndata: Y\n\ndata: [DONE]\n\n",
// so the split on "\n\n" yields ["data: X", "data: Y", "data:
// [DONE]", ""]. We count non-empty entries.
func sseFrameCount(body string) int {
	count := 0
	for _, frame := range strings.Split(body, "\n\n") {
		if frame != "" {
			count++
		}
	}
	return count
}

// TestFixtures_TextStreamShape asserts the recorded text-stream
// fixture carries the expected chunk count and the
// conformance-bridge constants. It is the structural pre-flight
// every CapStreamingText case consumes — if this fails the case
// failures downstream will be confusing.
func TestFixtures_TextStreamShape(t *testing.T) {
	t.Parallel()

	body := fixtures.OpenrouterTextStream()

	if len(body) == 0 {
		t.Fatal("openrouterTextStream is empty, want 6 SSE data frames (R-CNF-005: text interaction has ResponseStart + N TextDelta + Completion + [DONE])")
	}
	if !strings.Contains(body, fixtures.OpenrouterAttributionObjectDiscriminatorValue()) {
		// The fixture uses the bridge's chat.completion.chunk spelling
		// via the conformance bridge's own constants; this guard
		// asserts the constant's value, not a hard-coded literal.
		_ = fixtures.OpenrouterAttributionObjectDiscriminatorValue
		t.Errorf("openrouterTextStream missing %q object discriminator (R-ATS-017 / D3)", fixtures.OpenrouterAttributionObjectDiscriminatorValue())
	}
	if !strings.Contains(body, "\"model\":\"openai/gpt-4o\"") {
		t.Error("openrouterTextStream missing the openai/gpt-4o served model — bridge renders the default model the wrapper's Config substitutes (R-OR-03)")
	}
	// 6 SSE data frames expected: ResponseStart + 3 TextDelta +
	// Completion + [DONE]. The structural test asserts the count,
	// not each frame's payload — payload assertions live in the
	// conformance cases.
	wantFrames := 6
	if got := sseFrameCount(body); got != wantFrames {
		t.Errorf("openrouterTextStream SSE data-frame count = %d, want %d (one ResponseStart + 3 TextDelta + one Completion + [DONE])", got, wantFrames)
	}
	if !strings.Contains(body, "data: [DONE]") {
		t.Error("openrouterTextStream missing the [DONE] sentinel — bridge always terminates a replay with the SSE done marker")
	}
}

// TestFixtures_ToolCallStreamShape asserts the recorded tool-call
// fixture carries the expected chunk count, the tool_calls delta
// shape, and the bridge-synthesized terminal chunk with
// finish_reason: "tool_calls". The structural pre-flight
// CapToolCalls cases consume.
func TestFixtures_ToolCallStreamShape(t *testing.T) {
	t.Parallel()

	body := fixtures.OpenrouterToolCallStream()

	if len(body) == 0 {
		t.Fatal("openrouterToolCallStream is empty, want 6 SSE data frames (R-CNF-007: tool call has ToolCallStart + N ToolCallDelta + bridge-synthesized Completion + [DONE])")
	}
	if !strings.Contains(body, fixtures.OpenrouterAttributionObjectDiscriminatorValue()) {
		t.Errorf("openrouterToolCallStream missing %q object discriminator (R-ATS-017 / D3)", fixtures.OpenrouterAttributionObjectDiscriminatorValue())
	}
	if !strings.Contains(body, "\"tool_calls\":[{\"index\":0,\"id\":\"call_or_tool_1\",\"function\":{\"name\":\"search\"}}]") {
		t.Error("openrouterToolCallStream missing the tool_call start element (R-ATL-001 / C9.1: {index, id, function{name}})")
	}
	if !strings.Contains(body, "\"finish_reason\":\"tool_calls\"") {
		t.Error("openrouterToolCallStream missing the bridge-synthesized terminal chunk — C9.6 records no per-call end signal exists on the wire, so the bridge must mint the finish_reason:tool_calls terminal itself")
	}
	wantFrames := 6
	if got := sseFrameCount(body); got != wantFrames {
		t.Errorf("openrouterToolCallStream SSE data-frame count = %d, want %d (ResponseStart + ToolCallStart + 2 ToolCallDelta + Completion + [DONE])", got, wantFrames)
	}
}

// TestFixtures_WithUsageShape asserts the recorded text-with-usage
// fixture carries the expected chunk count, the C8 usage object on
// the terminal chunk, and every usage leaf present. The
// structural pre-flight CapCompletionMetadata's usage case consumes
// (R-CNF-016, R-ACP-005, AI-31.2).
func TestFixtures_WithUsageShape(t *testing.T) {
	t.Parallel()

	body := fixtures.OpenrouterTextStreamWithUsage()

	if len(body) == 0 {
		t.Fatal("openrouterTextStreamWithUsage is empty, want 5 SSE data frames (ResponseStart + 2 TextDelta + Completion-with-usage + [DONE])")
	}
	if !strings.Contains(body, fixtures.OpenrouterAttributionObjectDiscriminatorValue()) {
		t.Errorf("openrouterTextStreamWithUsage missing %q object discriminator (R-ATS-017 / D3)", fixtures.OpenrouterAttributionObjectDiscriminatorValue())
	}
	if !strings.Contains(body, "\"usage\":{") {
		t.Error("openrouterTextStreamWithUsage missing the C8 usage object on the terminal chunk — usageFromWire (chunk.go) only fires when the wire carries a populated usage object")
	}
	for _, leaf := range []string{
		"\"prompt_tokens\":7",
		"\"completion_tokens\":24",
		"\"cached_tokens\":0",
		"\"cache_write_tokens\":0",
		"\"reasoning_tokens\":0",
	} {
		if !strings.Contains(body, leaf) {
			t.Errorf("openrouterTextStreamWithUsage missing usage leaf %q (R-ATS-015, R-ACP-005…007, AI-31.2's three new raw mappings)", leaf)
		}
	}
	wantFrames := 5
	if got := sseFrameCount(body); got != wantFrames {
		t.Errorf("openrouterTextStreamWithUsage SSE data-frame count = %d, want %d", got, wantFrames)
	}
}

// TestFixtures_AttributionHeaderNamesConsistent asserts the Go
// var-block of attribution header names is non-empty and ordered:
// HTTP-Referer, X-OpenRouter-Title, X-Title (alias), X-OpenRouter
// -Categories. A future change to OpenRouter's header set is forced
// to update the list in two places (this package and the openrouter
// package's transport.go) — the test guards against a silent
// drift.
func TestFixtures_AttributionHeaderNamesConsistent(t *testing.T) {
	t.Parallel()

	names := fixtures.OpenrouterAttributionHeaderNames()
	want := []string{
		"HTTP-Referer",
		"X-OpenRouter-Title",
		"X-Title",
		"X-OpenRouter-Categories",
	}
	if len(names) != len(want) {
		t.Fatalf("OpenrouterAttributionHeaderNames length = %d, want %d (R-OR-02: every attribution header name must be listed, no extras, no omissions)", len(names), len(want))
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("OpenrouterAttributionHeaderNames[%d] = %q, want %q (R-OR-02 ordering invariant)", i, names[i], want[i])
		}
	}
}

// TestFixtures_AttributionHeaderValuesConsistent asserts the
// present-variant map carries every name in
// OpenrouterAttributionHeaderNames with the configured value, and
// that the X-OpenRouter-Title / X-Title alias pair carries the
// same value (R-OR-02 sub-scenario 1).
func TestFixtures_AttributionHeaderValuesConsistent(t *testing.T) {
	t.Parallel()

	values := fixtures.OpenrouterAttributionPresentValues()
	if len(values) != fixtures.OpenrouterAttributionPresentCount() {
		t.Fatalf("OpenrouterAttributionPresentValues length = %d, want %d (R-OR-02: every attribution header name carries a value when non-empty)", len(values), fixtures.OpenrouterAttributionPresentCount())
	}
	for _, name := range fixtures.OpenrouterAttributionHeaderNames() {
		if _, ok := values[name]; !ok {
			t.Errorf("OpenrouterAttributionPresentValues missing header %q — present-variant fixture must cover every header name (R-OR-02)", name)
		}
	}
	title, ok1 := values["X-OpenRouter-Title"]
	alias, ok2 := values["X-Title"]
	if !ok1 || !ok2 {
		t.Fatal("present-variant fixture missing the title/alias pair — R-OR-02 requires both names carry the same value")
	}
	if title != alias {
		t.Errorf("X-OpenRouter-Title = %q, X-Title = %q; aliases must carry the same value (R-OR-02 sub-scenario 1)", title, alias)
	}
}

// TestFixtures_AttributionAbsentCountPinned asserts the
// openrouterAttributionAbsentCount constant equals zero — the
// expected count of attribution-named headers in the outbound
// header when every attribution field is the zero value. The
// constant exists so a future change to the header set is forced
// to update the count, not silently leave a stale assertion in
// place (R-OR-02 sub-scenario 2).
func TestFixtures_AttributionAbsentCountPinned(t *testing.T) {
	t.Parallel()

	if got := fixtures.OpenrouterAttributionAbsentCount(); got != 0 {
		t.Errorf("OpenrouterAttributionAbsentCount = %d, want 0 (R-OR-02 sub-scenario 2: every attribution field zero-valued -> zero attribution-named headers on outbound)", got)
	}
	if got := fixtures.OpenrouterAttributionPresentCount(); got != len(fixtures.OpenrouterAttributionHeaderNames()) {
		t.Errorf("OpenrouterAttributionPresentCount = %d, want %d (every attribution name listed, present when non-empty) (R-OR-02 sub-scenario 1)", got, len(fixtures.OpenrouterAttributionHeaderNames()))
	}
}