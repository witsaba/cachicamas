// AI-38 — recorded OpenRouter-shaped SSE reasoning-extension
// fixture.
//
// # OpenRouter's renamed reasoning fields
//
// OpenRouter renamed the openai-compat reasoning field — the
// original `delta.reasoning_content` (covered by the shipped
// openaicompat/reasoning_absence_test.go's
// TestReasoningExtensionField_DroppedNotLeakedNotFailed_FixtureAAndFixtureB
// for the openai-compat wire) — into TWO separate fields on its
// own dialect:
//
//   - delta.reasoning_details: an array of structured entries
//     shaped {"type":"reasoning.text", "text":"..."}.
//   - delta.reasoning:          a free-form string carrying the
//                               reasoning bytes the model produced
//                               in this delta.
//
// Both fields are extension fields from the openaicompat
// decoder's view: the shipped wireChunk / wireDelta structs
// declare neither, so encoding/json's own unknown-field
// tolerance (R-ATS-017) silently drops them — exactly the
// mechanism that protects the conformance bridge's text-stream
// shape from leaking reasoning bytes into a text event
// (R-OR-06 sub-scenario 2, R-ARS-015).
//
// # Fixture shape
//
// openrouterReasoningExtensionStream is one recorded byte slice
// carrying BOTH renamed fields in two distinct positions:
//
//   - delta 1 carries reasoning_details with a recognizable
//     sentinel, alongside a declared content string ("alpha").
//   - delta 2 carries reasoning with the same sentinel, alongside
//     a second content string ("omega").
//
// The pair is the OpenRouter-specific twin of the shipped
// openaicompat/reasoning_absence_test.go fixture (which carries
// only the openai-compat `reasoning_content` field). The same
// drop mechanism covers both; the fixture proves the renamed
// fields are also dropped, not leaked into any text event, and
// the stream still reaches its normal completion (R-OR-06
// sub-scenario 2, R-ARS-015).

package fixtures

// openrouterReasoningExtensionStream is the recorded reasoning-
// extension canonical byte sequence — see this file's package
// doc for the shape.
var openrouterReasoningExtensionStream = "" +
	"data: {\"id\":\"chatcmpl-or-rsn\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\",\"content\":\"alpha\",\"reasoning_details\":[{\"type\":\"reasoning.text\",\"text\":\"OR-RSN-SENTINEL-7f3a\"}]},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-rsn\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"omega\",\"reasoning\":\"OR-RSN-SENTINEL-7f3a\"},\"finish_reason\":null}]}\n\n" +
	"data: {\"id\":\"chatcmpl-or-rsn\",\"model\":\"openai/gpt-4o\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n" +
	"data: [DONE]\n\n"

// OpenrouterReasoningExtensionSentinel is the distinctive byte
// sequence the renamed-field fixtures carry inside the undeclared
// `reasoning_details` and `reasoning` members. It is chosen to be
// a literal-bytes search target — unique, recognizable, and not
// a substring of any other fixture value in this package. The
// "OR-" prefix distinguishes it from the shipped openaicompat
// fixture's `reasoningExtensionSentinel = "RSN-EXT-SENTINEL-7f3a"`
// so a cross-fixture search (looking for the wrong sentinel)
// fails mechanically.
const OpenrouterReasoningExtensionSentinel = "OR-RSN-SENTINEL-7f3a"