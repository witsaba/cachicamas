// AI-38 — recorded OpenRouter-shaped SSE tool-call fixture.
//
// openrouterToolCallStream is a single recorded wire byte slice: one
// ToolCallStart + two ToolCallDelta chunks + the bridge's
// synthesized terminal chunk (finish_reason: "tool_calls") + the SSE
// [DONE] sentinel. The bridge synthesizes the terminal chunk because
// the conformance tool-call cases (R-CNF-007, R-CNF-008) carry no
// Completion / [DONE] of their own by design — bridge_renderScript's
// own terminalChunk-mints-[DONE] path is what makes those cases
// replayable, and this fixture records the resulting byte sequence as
// a canonical comparison target.
//
// The wire index of the single tool call is 0; the call id is
// "call_or_tool_1"; the call name is "search"; the arguments come in
// two fragments ("{\"q\":" + "\"a\"}"), reconstructing to
// {"q":"a"}. The choice index is 0 throughout.

package fixtures

// openrouterToolCallStream is the recorded tool-call canonical byte
// sequence — see this file's package doc for the shape.
var openrouterToolCallStream = "" +
	"data: {\"id\":\"chatcmpl-or-tool\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-tool\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"id\":\"call_or_tool_1\",\"function\":{\"name\":\"search\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-tool\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"{\\\"q\\\":\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-tool\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"tool_calls\":[{\"index\":0,\"function\":{\"arguments\":\"\\\"a\\\"}\"}}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-tool\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"tool_calls\"}]}\n\n" +
	"data: [DONE]\n\n"