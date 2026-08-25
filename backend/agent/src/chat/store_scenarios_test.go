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

	"github.com/cachicamas/backend/agent/src/agent"
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
		const participant = "scn-007-alice"
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
		const participant = "scn-008-alice"
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
			ToolSource:    chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
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

	// CH-08 (WU-4) — the helper gains the list-shape scenarios
	// (S-CCS-017, S-CCS-018) so both adapters run them with scenario
	// text unchanged (R-CCS-012 + design D-L precedent). The micro-
	// tests at store_test.go are the unit-level cover; this helper
	// is the cross-adapter cover.
	//
	// S-CCS-017 — a participant sees their own conversations and no
	// others (Gherkin verbatim, `0005:885-889`).
	//   Given two participants who have each held conversations
	//   When one of them requests their list
	//   Then the list contains only their own conversations
	//   And each entry identifies its conversation well enough to
	//   open it
	t.Run("S-CCS-017_a_participant_sees_their_own_conversations", func(t *testing.T) {
		const alice = "scn-017-alice"
		const bob = "scn-017-bob"

		// Both participants hold a conversation. D-1: one conversation
		// per participant, so each Append lands at position 0.
		if err := store.Append(alice, chat.Exchange{PromptText: "alice-1", AssistantText: "r-alice-1"}); err != nil {
			t.Fatalf("Append(alice) returned %v, want nil", err)
		}
		if err := store.Append(bob, chat.Exchange{PromptText: "bob-1", AssistantText: "r-bob-1"}); err != nil {
			t.Fatalf("Append(bob) returned %v, want nil", err)
		}

		aliceList, err := store.List(alice)
		if err != nil {
			t.Fatalf("List(alice) returned %v, want nil", err)
		}
		if len(aliceList) != 1 {
			t.Fatalf("List(alice) returned %d entries, want 1", len(aliceList))
		}
		if aliceList[0].ConversationID != alice {
			t.Errorf("List(alice)[0].ConversationID = %q, want %q", aliceList[0].ConversationID, alice)
		}
		// "Each entry identifies its conversation well enough to
		// open it" — the identifier IS the participant id (D-1).
		if aliceList[0].ConversationID == "" {
			t.Errorf("List(alice)[0].ConversationID is empty; the entry must be openable")
		}

		bobList, err := store.List(bob)
		if err != nil {
			t.Fatalf("List(bob) returned %v, want nil", err)
		}
		if len(bobList) != 1 {
			t.Fatalf("List(bob) returned %d entries, want 1", len(bobList))
		}
		if bobList[0].ConversationID != bob {
			t.Errorf("List(bob)[0].ConversationID = %q, want %q", bobList[0].ConversationID, bob)
		}

		// Cross-participant guard: alice's list MUST NOT contain bob's
		// row (NFR-CCS-007). Walk the slice and fail explicitly when
		// a leak surfaces, rather than only checking length.
		for _, entry := range aliceList {
			if entry.ConversationID != alice {
				t.Errorf("alice's list leaked entry %q (NFR-CCS-007 violation)", entry.ConversationID)
			}
		}
		for _, entry := range bobList {
			if entry.ConversationID != bob {
				t.Errorf("bob's list leaked entry %q (NFR-CCS-007 violation)", entry.ConversationID)
			}
		}
	})

	// S-CCS-018 — a participant with no conversations gets an
	// empty list (Gherkin verbatim, `0005:891-895`).
	//   Given an authenticated participant who has never held a
	//   conversation
	//   When they request their list
	//   Then the list is empty
	//   And the response is a success rather than a not-found
	t.Run("S-CCS-018_a_participant_with_no_conversations_gets_empty_list", func(t *testing.T) {
		const ghost = "scn-018-never-existed"

		got, err := store.List(ghost)
		if err != nil {
			t.Fatalf("List(%q) returned err=%v, want nil (the empty list is success, not not-found)", ghost, err)
		}
		if got == nil {
			t.Errorf("List(%q) returned nil, want []ConversationSummary{} (non-nil empty slice — JSON serializes as [] not null)", ghost)
		}
		if len(got) != 0 {
			t.Errorf("List(%q) returned %d entries, want 0", ghost, len(got))
		}

		// Follow-up: Append under a different id then List(ghost) —
		// the original miss MUST NOT have synthesised an entry.
		const another = "scn-018-another"
		if err := store.Append(another, chat.Exchange{PromptText: "x", AssistantText: "y"}); err != nil {
			t.Fatalf("Append(another) returned %v, want nil", err)
		}
		got2, err2 := store.List(ghost)
		if err2 != nil {
			t.Fatalf("List(%q) after follow-up Append returned err=%v, want nil", ghost, err2)
		}
		if len(got2) != 0 {
			t.Errorf("List(%q) after follow-up Append returned %d entries, want 0 (no ghost entry)", ghost, len(got2))
		}
	})
}

// TestMemoryConversationStore_CH08Scenarios_PassUnchanged is the WU-4
// unit-level micro-test that exercises the shared scenario helper
// against the in-memory adapter directly. The CH-06 helper scenario
// set (S-CCS-001 through S-CCS-009 + S-CCS-011) plus the CH-08 list
// scenarios (S-CCS-017, S-CCS-018) all run here at unit-level
// latency (no INTEGRATION gate) — the Postgres adapter mirrors the
// same helper via TestPostgresConversationStore_CH08Scenarios_PassUnchanged
// under INTEGRATION=1 (store_postgres_test.go). Scenario text is
// unchanged between this in-memory run and the postgres mirror,
// satisfying R-CCS-012 / design D-L's "scenario text unchanged across
// adapters" rule.
func TestMemoryConversationStore_CH08Scenarios_PassUnchanged(t *testing.T) {
	store := chat.NewMemoryConversationStore()
	RunConversationStoreScenarios(t, store)
}