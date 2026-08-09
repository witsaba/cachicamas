// AI-38 — recorded OpenRouter-shaped SSE text-stream fixture (R-ACR-003,
// design D3).
//
// openrouterTextStream is testdata/text_stream.sse, embedded verbatim: one
// ResponseStart + three TextDelta chunks + one terminal chunk
// (finish_reason: "stop") + the SSE [DONE] sentinel. It is produced and
// drift-guarded by recorder_test.go's TestRecordTranscript_
// RegeneratesEveryFixture — a real HTTP capture of bridgeRenderScript's
// output for the script ResponseStart("chatcmpl-or-text", "openai/gpt-4o")
// + TextBlockStart(1) + TextDelta(1, "alpha") + TextDelta(1, " beta") +
// TextDelta(1, " gamma") + TextBlockEnd(1) + Completion(FinishReasonStop,
// Usage{}), not a hand-typed literal: a hand edit to the committed file
// fails that test, naming this file.
//
// Any conformance case that consumes the text transcript can be wired
// against this recorded byte slice instead of re-rendering per test
// (sdd-phase-common §E: generated goldens are excluded from authored
// risk count, included in complete snapshot identity).
//
// The identity string "chatcmpl-or-text" and the served model
// "openai/gpt-4o" are deliberately the same identity every
// conformance case in this sub-package uses; cross-case equality is
// what lets the conformance suite compare cases by kind sequence
// (R-CNF-019) without also diffing identity strings.

package fixtures

import _ "embed"

// openrouterTextStream is the recorded text-stream canonical byte
// sequence, embedded from testdata/text_stream.sse — see this file's
// package doc for the shape and how it is regenerated/drift-guarded.
//
//go:embed testdata/text_stream.sse
var openrouterTextStream string
