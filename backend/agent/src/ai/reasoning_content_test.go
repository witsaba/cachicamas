// Tests for AI-07 — reasoning content and its round-trip token.
//
// The package under test is imported by its module path from the external test
// package ai_test, so every assertion below is written against exactly the
// surface a consumer in another package sees. That matters more here than it
// did for text: doc 0001 § 3.1 records that the retired Layer 1's reasoning
// type was the one that broke the content-part pattern, and the cheapest way to
// notice a second strategy is to be unable to write the first one's loop.

package ai_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-07.1 item 1 — a reasoning part is constructed and read back through the
// AI-06.1 strategy, exactly like text: no second strategy, no special case.
//
// The body below is the loop AI-26 will write, with one identifier changed from
// the text case. If reasoning ever needs a different loop, this test is where
// that shows up.
func TestReasoning_ConstructedAndReadBack_UsesTheContentPartStrategy(t *testing.T) {
	t.Parallel()

	const thought = "The user asked for a sum, so I will add the two numbers."

	part, err := ai.NewReasoning(thought, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}
	if got := part.Kind(); got != ai.PartKindReasoning {
		t.Fatalf("part.Kind() = %v, want %v", got, ai.PartKindReasoning)
	}

	msg, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}

	content := msg.Content()
	if len(content) != 1 {
		t.Fatalf("msg.Content() returned %d elements, want 1", len(content))
	}

	switch got := content[0]; got.Kind() {
	case ai.PartKindReasoning:
		reasoning, ok := got.Reasoning()
		if !ok {
			t.Fatalf("part.Reasoning() reported no reasoning on a part of kind %v", got.Kind())
		}
		if reasoning.Text() != thought {
			t.Errorf("reasoning.Text() = %q, want %q", reasoning.Text(), thought)
		}
	default:
		t.Fatalf("the part read back reports kind %v, want %v", got.Kind(), ai.PartKindReasoning)
	}
}

// AI-07.1 item 2 — reasoning and text are structurally distinct.
//
// V-REQ-09's last clause is the requirement: "distinct from text content at
// every layer — reasoning is never rendered as, merged into, or substituted for
// text." The two are built from the identical string on purpose, so nothing
// below can pass because the payloads happen to differ.
func TestReasoning_AndText_AreStructurallyDistinct(t *testing.T) {
	t.Parallel()

	const shared = "the same characters, in two different kinds of part"

	reasoningPart, err := ai.NewReasoning(shared, nil)
	if err != nil {
		t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
	}
	textPart, err := ai.NewText(shared)
	if err != nil {
		t.Fatalf("ai.NewText returned %v, want no failure", err)
	}

	t.Run("no accessor yields reasoning content as text", func(t *testing.T) {
		t.Parallel()

		text, ok := reasoningPart.Text()
		if ok {
			t.Errorf("reasoningPart.Text() reported text content on a reasoning part")
		}
		if text != "" {
			t.Errorf("reasoningPart.Text() = %q, want the zero value rather than the reasoning", text)
		}
	})

	t.Run("no accessor yields text content as reasoning", func(t *testing.T) {
		t.Parallel()

		reasoning, ok := textPart.Reasoning()
		if ok {
			t.Errorf("textPart.Reasoning() reported reasoning on a text part")
		}
		if reasoning.Text() != "" {
			t.Errorf("textPart.Reasoning() yielded %q, want the zero payload", reasoning.Text())
		}
	})

	t.Run("a part that was never constructed yields neither", func(t *testing.T) {
		t.Parallel()

		var unconstructed ai.Part

		if _, ok := unconstructed.Reasoning(); ok {
			t.Errorf("the zero Part reported reasoning")
		}
		if _, ok := unconstructed.Text(); ok {
			t.Errorf("the zero Part reported text content")
		}
	})

	t.Run("a consumer switching on kind cannot conflate them", func(t *testing.T) {
		t.Parallel()

		if reasoningPart.Kind() == textPart.Kind() {
			t.Fatalf("both parts report kind %v, so a switch reaches one branch for two kinds", textPart.Kind())
		}

		visited := make(map[ai.PartKind]int, 2)
		for _, part := range []ai.Part{reasoningPart, textPart} {
			switch part.Kind() {
			case ai.PartKindReasoning:
				if _, ok := part.Reasoning(); !ok {
					t.Errorf("the reasoning branch was reached by a part carrying no reasoning")
				}
				visited[ai.PartKindReasoning]++
			case ai.PartKindText:
				if _, ok := part.Text(); !ok {
					t.Errorf("the text branch was reached by a part carrying no text")
				}
				visited[ai.PartKindText]++
			default:
				t.Errorf("a part reached no branch, reporting kind %v", part.Kind())
			}
		}

		if visited[ai.PartKindReasoning] != 1 || visited[ai.PartKindText] != 1 {
			t.Errorf("the switch visited %v, want each branch exactly once", visited)
		}
	})
}

// AI-07.1 item 3 — the reasoning state vocabulary is closed, and each state is
// constructible.
//
// V-REQ-10 exists "so that 'no reasoning text' is a recorded state rather than
// an empty string", which is only true if the state that records it can be
// built. Three members: with text, redacted, and token-only — the last being
// both V-REQ-10's signature-only shape and its "a provider that emitted no
// reasoning text at all", which doc 0002's AI-07.4 item 2 names as one shape.
func TestReasoningStates_Vocabulary_IsClosedStableAndConstructible(t *testing.T) {
	t.Parallel()

	token := []byte("an opaque provider signature")

	t.Run("each state is constructible and reports itself", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			name  string
			build func() (ai.Part, error)
			want  ai.ReasoningState
		}{
			{
				name:  "with text",
				build: func() (ai.Part, error) { return ai.NewReasoning("I will add the two numbers.", token) },
				want:  ai.ReasoningStateText,
			},
			{
				name:  "redacted",
				build: func() (ai.Part, error) { return ai.NewRedactedReasoning(token) },
				want:  ai.ReasoningStateRedacted,
			},
			{
				name:  "token-only",
				build: func() (ai.Part, error) { return ai.NewReasoning("", token) },
				want:  ai.ReasoningStateTokenOnly,
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				part, err := tc.build()
				if err != nil {
					t.Fatalf("construction returned %v, want no failure", err)
				}
				reasoning, ok := part.Reasoning()
				if !ok {
					t.Fatalf("part.Reasoning() reported no reasoning on a part of kind %v", part.Kind())
				}
				if got := reasoning.State(); got != tc.want {
					t.Errorf("reasoning.State() = %v, want %v", got, tc.want)
				}
				if !containsReasoningState(ai.ReasoningStates(), reasoning.State()) {
					t.Errorf("reasoning.State() = %v, which is not a member of the vocabulary", reasoning.State())
				}
			})
		}
	})

	t.Run("the vocabulary is closed, stable and not rewritable by a consumer", func(t *testing.T) {
		t.Parallel()

		states := ai.ReasoningStates()
		if len(states) != 3 {
			t.Fatalf("ai.ReasoningStates() enumerates %d members (%v), want 3", len(states), states)
		}

		want := []ai.ReasoningState{ai.ReasoningStateText, ai.ReasoningStateRedacted, ai.ReasoningStateTokenOnly}
		for i, state := range want {
			if states[i] != state {
				t.Errorf("ai.ReasoningStates()[%d] = %v, want %v — the order is the declaration order", i, states[i], state)
			}
		}

		for i := range states {
			states[i] = 0
		}
		if again := ai.ReasoningStates(); again[0] != ai.ReasoningStateText || len(again) != 3 {
			t.Errorf("ai.ReasoningStates() = %v after a consumer rewrote the slice it received, want the vocabulary unchanged", again)
		}
	})

	t.Run("the zero value is not a member and renders as unset", func(t *testing.T) {
		t.Parallel()

		var unset ai.ReasoningState

		if containsReasoningState(ai.ReasoningStates(), unset) {
			t.Errorf("the zero ReasoningState is a member of the vocabulary, want a non-member")
		}
		if got := unset.String(); got != "unset" {
			t.Errorf("ReasoningState(0).String() = %q, want %q", got, "unset")
		}
	})

	t.Run("a member renders as its registered name and a non-member identifiably", func(t *testing.T) {
		t.Parallel()

		names := map[ai.ReasoningState]string{
			ai.ReasoningStateText:      "text",
			ai.ReasoningStateRedacted:  "redacted",
			ai.ReasoningStateTokenOnly: "token-only",
		}
		for state, want := range names {
			if got := state.String(); got != want {
				t.Errorf("%d.String() = %q, want %q", uint8(state), got, want)
			}
		}

		wild := ai.ReasoningState(200)
		if got := wild.String(); got != "reasoningstate(200)" {
			t.Errorf("ReasoningState(200).String() = %q, want %q", got, "reasoningstate(200)")
		}
	})
}

func containsReasoningState(haystack []ai.ReasoningState, want ai.ReasoningState) bool {
	for _, got := range haystack {
		if got == want {
			return true
		}
	}
	return false
}
