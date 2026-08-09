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

// # Not recorder-eligible (permanent, not a Phase 5 dependency)
//
// This fixture's storage moved to go:embed (task 1.3) but, unlike
// text_stream.go and tool_call.go, it is never wired into
// recorder_test.go's round-trip: reasoning_details and reasoning are
// vendor wire-extension fields with no corresponding ai.Event —
// bridgeRenderScript has no script vocabulary that produces them (that is
// the whole point: the decoder's tolerant unknown-field handling drops
// them, R-ATS-017). This is a hand-authored fixture describing a
// hypothetical vendor response shape, not a captured rendering of an
// agenttest.Script, so it is permanently outside the recorder's
// applicability rather than blocked on a later phase.

package fixtures

import _ "embed"

// openrouterReasoningExtensionStream is the recorded reasoning-
// extension canonical byte sequence, embedded from
// testdata/reasoning_extension.sse — see this file's package doc for
// the shape and why it stays outside the recorder round-trip.
//
//go:embed testdata/reasoning_extension.sse
var openrouterReasoningExtensionStream string

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
