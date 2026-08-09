// AI-38 — recorded OpenRouter-shaped SSE text-stream-with-usage fixture.
//
// openrouterTextStreamWithUsage is testdata/with_usage.sse, embedded
// verbatim: one ResponseStart + two TextDelta chunks + one terminal chunk
// carrying a usage object (prompt_tokens: 7, completion_tokens: 24,
// reasoning_tokens: 0, cached_tokens: 0) + the SSE [DONE] sentinel.
//
// # Not yet recorder-verified (Phase 5 / D6 dependency)
//
// Unlike text_stream.go and tool_call.go, this fixture's storage moved
// to the go:embed mechanism (task 1.3) but is NOT wired into
// recorder_test.go's TestRecordTranscript_RegeneratesEveryFixture
// round-trip: the shipped bridgeWriteTerminalChunk (bridge_test.go)
// does not yet render a usage object at all (design D6, Phase 5/WU5 —
// "usage object rendering present-fields-only"). Recording this
// transcript through
// bridgeRenderScript today would produce a DIFFERENT byte sequence (no
// usage key), so this file stays a committed golden the recorder cannot
// yet reproduce until D6 lands, at which point it is regenerated (not
// necessarily byte-identical to today's all-fields-populated shape — D6
// renders present fields only) and added to the round-trip set.
//
// The presence of a usage object on the terminal chunk is what
// exercises the C8 / R-ACP-005 path: openaicompat's chunk.go
// usageFromWire (R-ATS-015, AI-31.2) maps prompt_tokens ->
// Usage.Input and completion_tokens -> Usage.Output, with
// cached_tokens -> CacheRead and reasoning_tokens -> Reasoning as
// raw mappings.
//
// usage is a *wireUsage in the openaicompat decoder (a pointer so
// absent-key and explicit-JSON-null both decode to nil — D10), and
// every leaf inside wireUsage is also a *int64 for the same
// absent-vs-present-with-explicit-zero distinction (S-ACP-013). The
// fixture supplies every C8 leaf present (including 0) so the
// usage case can prove both present-and-zero and present-and-nonzero
// decode identically into ai.Usage.

package fixtures

import _ "embed"

// openrouterTextStreamWithUsage is the recorded text-with-usage
// canonical byte sequence, embedded from testdata/with_usage.sse — see
// this file's package doc for the shape and the Phase 5 dependency.
//
//go:embed testdata/with_usage.sse
var openrouterTextStreamWithUsage string
