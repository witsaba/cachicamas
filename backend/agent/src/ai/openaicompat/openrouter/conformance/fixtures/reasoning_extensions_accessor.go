// AI-38 — OpenRouter reasoning-extension accessor.
//
// OpenrouterReasoningExtensionStream returns the recorded reasoning-
// extension canonical byte sequence (reasoning_details + reasoning
// renamed fields) — see fixtures/reasoning_extensions.go.

package fixtures

// OpenrouterReasoningExtensionStream returns the recorded
// OpenRouter-shaped reasoning-extension stream
// (ResponseStart-with-reasoning_details + TextDelta-with-reasoning
// + Completion + [DONE]) — see reasoning_extensions.go.
func OpenrouterReasoningExtensionStream() string {
	return openrouterReasoningExtensionStream
}
