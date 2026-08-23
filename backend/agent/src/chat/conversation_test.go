// CH-02.1 — the conversation and its projection onto the wire vocabulary
// (R-CCP-001..005). Every scenario lives in package chat_test, proving the
// archetype's behaviour from outside the package, with no reach into
// unexported surface.
package chat_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/chat"
)

// TestConversation_SystemPrompt — S-CCP-010. Given the merged tree, when the
// archetype's system-prompt constant is read, then its value is
// byte-identical to the literal R-CCP-002 pins (D13).
func TestConversation_SystemPrompt(t *testing.T) {
	t.Parallel()

	const want = "You are the cachicamas chat assistant; answer the participant in plain, well-formatted text."
	if chat.SystemPrompt != want {
		t.Errorf("chat.SystemPrompt = %q, want %q", chat.SystemPrompt, want)
	}
}
