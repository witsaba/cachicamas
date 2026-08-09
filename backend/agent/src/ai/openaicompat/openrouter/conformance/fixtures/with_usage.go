// AI-38 — recorded OpenRouter-shaped SSE text-stream-with-usage fixture.
//
// openrouterTextStreamWithUsage is testdata/with_usage.sse, embedded
// verbatim: one ResponseStart + two TextDelta chunks + one terminal chunk
// carrying a usage object (prompt_tokens: 7, completion_tokens: 24,
// reasoning_tokens: 0, cached_tokens: 0) + the SSE [DONE] sentinel.
//
// # Recorder-verified (AI-38 WU10 ACCURACY-1 remediation)
//
// Like text_stream.go and tool_call.go, this fixture is wired into
// recorder_test.go's TestRecordTranscript_RegeneratesEveryFixture
// round-trip via recorderCanonicalTextStreamWithUsageScript: Phase
// 5/WU5 landed design D6's "usage object rendering present-fields-only"
// in bridgeWriteTerminalChunk, so a real HTTP capture of the canonical
// script above is now byte-identical to this committed golden (910
// bytes) — verify-report.md's own probe proved this first; this fixture
// closes apply deviation #3 / WARN-3 by shipping that proof as a
// permanent, drift-guarded test rather than a one-off probe.
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
