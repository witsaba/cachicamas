// AI-38 — recorded OpenRouter-shaped SSE tool-call fixture (R-ACR-003,
// design D3).
//
// openrouterToolCallStream is testdata/tool_call.sse, embedded verbatim:
// one ResponseStart + one ToolCallStart + two ToolCallDelta chunks + the
// bridge's synthesized terminal chunk (finish_reason: "tool_calls") + the
// SSE [DONE] sentinel. The bridge synthesizes the terminal chunk because
// the conformance tool-call cases (R-CNF-007, R-CNF-008) carry no
// Completion / [DONE] of their own by design — bridgeRenderScript's own
// terminalChunk-mints-[DONE] path is what produces this byte sequence,
// captured over real HTTP and drift-guarded by recorder_test.go's
// TestRecordTranscript_RegeneratesEveryFixture, not hand-typed.
//
// The wire index of the single tool call is 0; the call id is
// "call_or_tool_1"; the call name is "search"; the arguments come in
// two fragments ("{\"q\":" + "\"a\"}"), reconstructing to
// {"q":"a"}. The choice index is 0 throughout.

package fixtures

import _ "embed"

// openrouterToolCallStream is the recorded tool-call canonical byte
// sequence, embedded from testdata/tool_call.sse — see this file's
// package doc for the shape and how it is regenerated/drift-guarded.
//
//go:embed testdata/tool_call.sse
var openrouterToolCallStream string
