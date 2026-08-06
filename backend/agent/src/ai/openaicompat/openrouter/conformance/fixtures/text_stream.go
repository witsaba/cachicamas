// AI-38 — recorded OpenRouter-shaped SSE text-stream fixture.
//
// openrouterTextStream is a single recorded wire byte slice: one
// ResponseStart + three TextDelta chunks + one terminal chunk
// (finish_reason: "stop") + the SSE [DONE] sentinel. Byte-rendered
// using the same constants the conformance bridge's renderer uses:
//
//   - conformanceBridgeChunkCreated = 1700000000
//   - conformanceBridgeObjectDiscriminator = "chat.completion.chunk"
//
// so this fixture is byte-for-byte equivalent to what bridgeRenderScript
// produces when given a script of ResponseStart("chatcmpl-or-text",
// "openai/gpt-4o") + TextBlockStart(1) + TextDelta(1, "alpha") +
// TextDelta(1, " beta") + TextDelta(1, " gamma") + TextBlockEnd(1) +
// Completion(FinishReasonStop, Usage{}).
//
// Any conformance case that consumes the text transcript can be wired
// against this recorded byte slice instead of re-rendering per test
// (sdd-phase-common §E: generated goldens are excluded from authored
// risk count, included in complete snapshot identity). The fixture is
// one shape — a real OpenRouter chat-completion stream — and a future
// test that re-renders it can diff against this recorded canonical
// byte sequence to catch renderer drift.
//
// The identity string "chatcmpl-or-text" and the served model
// "openai/gpt-4o" are deliberately the same identity every
// conformance case in this sub-package uses; cross-case equality is
// what lets the conformance suite compare cases by kind sequence
// (R-CNF-019) without also diffing identity strings.

package fixtures

// openrouterTextStream is the recorded text-stream canonical byte
// sequence — see this file's package doc for the shape and the
// renderer constants.
var openrouterTextStream = "" +
	"data: {\"id\":\"chatcmpl-or-text\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-text\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"alpha\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-text\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" beta\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-text\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\" gamma\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-text\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
	"data: [DONE]\n\n"