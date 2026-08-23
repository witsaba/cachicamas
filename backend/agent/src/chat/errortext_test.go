// CH-02.3 — the archetype-owned failure phrase table (R-CCP-007, R-CCP-008,
// D6). Package chat (internal/white-box): phraseFor is unexported, so a
// direct table-driven unit test over the pure function lives here rather
// than being driven end to end through Conversation.Send.
package chat

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestPhraseFor_EveryCategoryHasANonEmptyPhrase — S-CCP-062. Given the
// archetype's category-to-phrase table, when it is read against every
// ai.FailureCategory member and one non-member fallback value, then every
// input produces a non-empty phrase, so no category can reach the wire
// with an empty message.
func TestPhraseFor_EveryCategoryHasANonEmptyPhrase(t *testing.T) {
	t.Parallel()

	categories := ai.FailureCategories()
	if len(categories) == 0 {
		t.Fatal("ai.FailureCategories() returned zero members — the table below would be vacuous")
	}

	seen := make(map[string]int)
	for _, category := range categories {
		phrase := phraseFor(category)
		if phrase == "" {
			t.Errorf("phraseFor(%s) = %q, want a non-empty phrase", category, phrase)
		}
		seen[phrase]++
	}

	// The fallback value: one past the largest declared member, guaranteed
	// not to be a member of ai.FailureCategory's own closed vocabulary.
	nonMember := ai.FailureCategory(categories[len(categories)-1] + 100)
	if phrase := phraseFor(nonMember); phrase == "" {
		t.Errorf("phraseFor(non-member %v) = %q, want a non-empty fallback phrase", nonMember, phrase)
	}

	// Triangulation: at least two distinct phrases exist (a table
	// collapsing every category onto one string would trivially pass the
	// non-empty check above without actually distinguishing categories).
	if len(seen) < 2 {
		t.Errorf("phraseFor produced only %d distinct phrase(s) across %d categories, want more than 1 (a real per-category table, not one shared string)", len(seen), len(categories))
	}
}
