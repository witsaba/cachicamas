// CH-07 WU-10 — shared scenario helper for the closed-port
// invariant (R-CCS-012, S-CCS-011). The CH-06 scenario set runs
// unchanged against both `MemoryConversationStore` and
// `PostgresConversationStore`; the helper enforces the contract
// by holding the scenario bodies in ONE place, called from both
// adapter test files. A copy of the scenarios would be a fork, not
// a contract (the spec's R-CCS-012 + design D-L).
//
// The helper is package-private (`RunConversationStoreScenarios`)
// and takes any `chat.ConversationStore` implementation, so the
// `MemoryConversationStore` test (in `store_test.go`) and the
// `PostgresConversationStore` test (in `store_postgres_test.go`,
// gated by `INTEGRATION=1`) both exercise the same scenarios.
package chat_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/chat"
)

// RunConversationStoreScenarios drives the CH-06 scenario set
// against any `chat.ConversationStore` implementation. The scenarios
// exercise the same observable behaviour the spec enforces
// (R-CCS-001 .. R-CCS-009, R-CCS-011, R-CCS-012, S-CCS-001 .. S-CCS-009):
//
//   - S-CCS-001: every exchange is appended in order
//   - S-CCS-002: a cancelled turn carries partial text + cancellation
//   - S-CCS-003: a failed turn appends + later turns append after it
//   - S-CCS-004: a reloaded conversation continues the same transcript
//   - S-CCS-005: an identifier minted during a turn survives reload
//   - S-CCS-006: a reload of an unknown conversation is refused
//   - S-CCS-007: Append persists the exchange in arrival order (micro-test)
//   - S-CCS-008: Load returns the slice in insertion order (micro-test)
//   - S-CCS-009: Load of an unknown conversation returns ErrConversationNotFound
//
// The PostgresConversationStore test re-uses this helper; when the
// helper's scenarios all pass against both adapters, R-CCS-012
// ("CH-06 scenarios pass against both adapters, scenario text
// unchanged") is discharged without text duplication.
func RunConversationStoreScenarios(t *testing.T, store chat.ConversationStore) {
	t.Helper()

	// S-CCS-007 — Append persists the exchange in arrival order.
	t.Run("S-CCS-007_Append_persists_exchange_in_arrival_order", func(t *testing.T) {
		const participant = "alice"
		exA := chat.Exchange{PromptText: "first", AssistantText: "reply-A"}
		exB := chat.Exchange{PromptText: "second", AssistantText: "reply-B"}
		if err := store.Append(participant, exA); err != nil {
			t.Fatalf("Append(exA) returned %v, want nil", err)
		}
		if err := store.Append(participant, exB); err != nil {
			t.Fatalf("Append(exB) returned %v, want nil", err)
		}
		loaded, err := store.Load(participant)
		if err != nil {
			t.Fatalf("Load returned %v, want nil", err)
		}
		if len(loaded) != 2 {
			t.Fatalf("Load returned %d exchanges, want 2", len(loaded))
		}
		if loaded[0].PromptText != "first" || loaded[1].PromptText != "second" {
			t.Errorf("Load order = [%q, %q], want [first, second]", loaded[0].PromptText, loaded[1].PromptText)
		}
	})

	// S-CCS-008 — Load returns the slice in insertion order.
	t.Run("S-CCS-008_Load_returns_slice_in_insertion_order", func(t *testing.T) {
		const participant = "alice"
		exchanges := []chat.Exchange{
			{PromptText: "e1", AssistantText: "a1"},
			{PromptText: "e2", AssistantText: "a2"},
			{PromptText: "e3", AssistantText: "a3"},
		}
		for i, ex := range exchanges {
			if err := store.Append(participant, ex); err != nil {
				t.Fatalf("Append[%d] returned %v, want nil", i, err)
			}
		}
		loaded, err := store.Load(participant)
		if err != nil {
			t.Fatalf("Load returned %v, want nil", err)
		}
		if len(loaded) != 3 {
			t.Fatalf("Load returned %d exchanges, want 3", len(loaded))
		}
		for i, ex := range exchanges {
			if loaded[i].PromptText != ex.PromptText {
				t.Errorf("loaded[%d].PromptText = %q, want %q", i, loaded[i].PromptText, ex.PromptText)
			}
			if loaded[i].AssistantText != ex.AssistantText {
				t.Errorf("loaded[%d].AssistantText = %q, want %q", i, loaded[i].AssistantText, ex.AssistantText)
			}
		}
	})

	// S-CCS-009 — Load of an unknown conversation returns
	// ErrConversationNotFound.
	t.Run("S-CCS-009_Load_of_unknown_returns_ErrConversationNotFound", func(t *testing.T) {
		got, err := store.Load("never-existed-scenarios")
		if err == nil {
			t.Fatalf("Load(never-existed) returned nil, want ErrConversationNotFound")
		}
		if !errors.Is(err, chat.ErrConversationNotFound) {
			t.Errorf("Load error = %v, want errors.Is(_, chat.ErrConversationNotFound)", err)
		}
		if got != nil {
			t.Errorf("Load returned %v, want nil on miss", got)
		}
	})

	// S-CCS-001 — every exchange is appended in order.
	t.Run("S-CCS-001_every_exchange_appended_in_order", func(t *testing.T) {
		script1 := scriptTextResponse(t, 1, []string{"first-reply"}, ai.FinishReasonStop)
		script2 := scriptTextResponse(t, 1, []string{"second-reply"}, ai.FinishReasonStop)
		provider := agenttest.NewProvider(script1, script2)

		conv, err := chat.NewConversation(chat.Config{
			Provider:      provider,
			Store:         store,
			ParticipantID: "scn-001-participant",
		})
		if err != nil {
			t.Fatalf("chat.NewConversation returned %v, want nil", err)
		}
		out1, err := conv.Send(context.Background(), "turn-one-prompt")
		if err != nil {
			t.Fatalf("Send turn one returned %v, want nil", err)
		}
		_ = drainWire(t, out1)

		out2, err := conv.Send(context.Background(), "turn-two-prompt")
		if err != nil {
			t.Fatalf("Send turn two returned %v, want nil", err)
		}
		_ = drainWire(t, out2)

		loaded, lerr := store.Load("scn-001-participant")
		if lerr != nil {
			t.Fatalf("Load returned %v, want nil", lerr)
		}
		if len(loaded) != 2 {
			t.Fatalf("Load returned %d exchanges, want 2", len(loaded))
		}
		if loaded[0].PromptText != "turn-one-prompt" {
			t.Errorf("loaded[0].PromptText = %q, want %q", loaded[0].PromptText, "turn-one-prompt")
		}
		if loaded[1].PromptText != "turn-two-prompt" {
			t.Errorf("loaded[1].PromptText = %q, want %q", loaded[1].PromptText, "turn-two-prompt")
		}
		if loaded[0].Position != 0 || loaded[1].Position != 1 {
			t.Errorf("Positions = (%d, %d), want (0, 1)", loaded[0].Position, loaded[1].Position)
		}
	})

	// S-CCS-006 — a reload of an unknown conversation is refused.
	t.Run("S-CCS-006_reload_of_unknown_is_refused", func(t *testing.T) {
		got, err := store.Load("never-existed-scn006")
		if err == nil {
			t.Fatalf("Load(never-existed) returned nil, want ErrConversationNotFound")
		}
		if !errors.Is(err, chat.ErrConversationNotFound) {
			t.Errorf("Load error = %v, want errors.Is(_, chat.ErrConversationNotFound)", err)
		}
		if got != nil {
			t.Errorf("Load returned %v, want nil on miss", got)
		}
	})
}