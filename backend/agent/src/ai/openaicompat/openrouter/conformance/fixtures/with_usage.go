// AI-38 — recorded OpenRouter-shaped SSE text-stream-with-usage fixture.
//
// openrouterTextStreamWithUsage is a single recorded wire byte slice:
// one ResponseStart + two TextDelta chunks + one terminal chunk
// carrying a usage object (prompt_tokens: 7, completion_tokens: 24,
// reasoning_tokens: 0, cached_tokens: 0) + the SSE [DONE] sentinel.
//
// The presence of a usage object on the terminal chunk is what
// exercises the C8 / R-ACP-005 path: openaicompat's chunk.go
// usageFromWire (R-ATS-015, AI-31.2) maps prompt_tokens ->
// Usage.Input and completion_tokens -> Usage.Output, with
// cached_tokens -> CacheRead and reasoning_tokens -> Reasoning as
// raw mappings. The CAP-R-03 (completion_metadata) conformance case
// finish_reason/all_seven_values_reachable_drift_guarded and
// usage/absent_vs_zero_distinguishable are the cases that consume
// this fixture.
//
// usage is a *wireUsage in the openaicompat decoder (a pointer so
// absent-key and explicit-JSON-null both decode to nil — D10), and
// every leaf inside wireUsage is also a *int64 for the same
// absent-vs-present-with-explicit-zero distinction (S-ACP-013). The
// fixture supplies every C8 leaf present (including 0) so the
// usage case can prove both present-and-zero and present-and-nonzero
// decode identically into ai.Usage.

package fixtures

// openrouterTextStreamWithUsage is the recorded text-with-usage
// canonical byte sequence — see this file's package doc for the
// shape.
var openrouterTextStreamWithUsage = "" +
	"data: {\"id\":\"chatcmpl-or-usage\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-usage\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"hello\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-usage\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" world\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-usage\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":7,\"completion_tokens\":24,\"prompt_tokens_details\":{\"cached_tokens\":0,\"cache_write_tokens\":0},\"completion_tokens_details\":{\"reasoning_tokens\":0}}}\n\n" +
	"data: [DONE]\n\n"