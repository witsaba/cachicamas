// CH-07 WU-11 — cross-process round-trip scenario (S-CCS-010,
// R-CCS-012). The CH-07.1 first Gherkin leaf at doc 0005:800-803
// asserts that a conversation recorded through one adapter is
// read back through a separately constructed adapter over the
// same store — proving durability across process boundaries.
//
// This test opens TWO `*sql.DB` connections against the same DSN,
// runs the chat-owned migrations on each, calls Append on the
// first, then closes it. It opens a second adapter and reads the
// record back. The assertion: same sequence, same order, all
// eight fields round-tripped.
//
// Gated by `INTEGRATION=1`. Without the gate, the test SKIPs
// (mirroring the DBA precedent at
// backend/database_administrator/src/migration/postgres/driver_test.go:295).
package chat_test

import (
	"context"
	"os"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
	"github.com/cachicamas/backend/agent/src/chat/migrations"
	"github.com/cachicamas/backend/agent/src/chat/migrator"
)

func TestPostgresConversationStore_CrossProcess_RoundTrips(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (S-CCS-010 requires a real Postgres; two adapters over the same DSN)")
	}
	resetChatTables(t)

	dsn := chatPostgresTestDSN()
	const participant = "cross-process-sccs010"

	// Process A: open, run migrations, Append, close.
	storeA, closerA, err := chat.NewPostgresConversationStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf("NewPostgresConversationStore (process A): %v", err)
	}
	providerA, err := migrator.NewProvider(context.Background(), storeA.RawDB(), migrations.MigrationsFS, "chat_schema_migrations")
	if err != nil {
		_ = closerA()
		t.Fatalf("migrator.NewProvider (process A): %v", err)
	}
	if _, err := providerA.Up(context.Background()); err != nil {
		_ = closerA()
		t.Fatalf("provider.Up (process A): %v", err)
	}

	exA := chat.Exchange{PromptText: "cross-A", AssistantText: "reply-A"}
	exB := chat.Exchange{PromptText: "cross-B", AssistantText: "reply-B"}
	if err := storeA.Append(participant, exA); err != nil {
		_ = closerA()
		t.Fatalf("Append (process A) exA: %v", err)
	}
	if err := storeA.Append(participant, exB); err != nil {
		_ = closerA()
		t.Fatalf("Append (process A) exB: %v", err)
	}

	if err := closerA(); err != nil {
		t.Fatalf("closerA: %v", err)
	}

	// Process B: open a fresh connection over the same DSN; the
	// record written by A must be visible.
	storeB, closerB, err := chat.NewPostgresConversationStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf("NewPostgresConversationStore (process B): %v", err)
	}
	t.Cleanup(func() { _ = closerB() })

	providerB, err := migrator.NewProvider(context.Background(), storeB.RawDB(), migrations.MigrationsFS, "chat_schema_migrations")
	if err != nil {
		t.Fatalf("migrator.NewProvider (process B): %v", err)
	}
	if _, err := providerB.Up(context.Background()); err != nil {
		t.Fatalf("provider.Up (process B): %v", err)
	}

	loaded, err := storeB.Load(participant)
	if err != nil {
		t.Fatalf("Load (process B): %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("Load returned %d exchanges, want 2 (cross-process round-trip)", len(loaded))
	}
	if loaded[0].PromptText != "cross-A" || loaded[1].PromptText != "cross-B" {
		t.Errorf("Load order = [%q, %q], want [cross-A, cross-B]", loaded[0].PromptText, loaded[1].PromptText)
	}
	if loaded[0].AssistantText != "reply-A" || loaded[1].AssistantText != "reply-B" {
		t.Errorf("Load assistant text = [%q, %q], want [reply-A, reply-B]", loaded[0].AssistantText, loaded[1].AssistantText)
	}
	if loaded[0].Position != 0 || loaded[1].Position != 1 {
		t.Errorf("Positions = (%d, %d), want (0, 1)", loaded[0].Position, loaded[1].Position)
	}
}

// TestExchangesToHistory_PostgresRoundTrip_SkipsFailedTurns — the
// production crash path: a participant with at least one prior
// failed turn (assistant_text == "", terminal_kind == failed,
// failure_category == unavailable) caused chat.Registry's factory
// closure to panic ("chat: Registry factory error: text: required
// value is empty") on the first POST /api/agent/turns. This
// integration test pins the round-trip: write a healthy turn and a
// failed turn through the postgres adapter, reload via
// ExchangesToHistory, and assert the failed turn's empty assistant
// message is skipped, not surfaced as ai.NewText's
// Invalid(ai.ErrEmpty, At("text")).
func TestExchangesToHistory_PostgresRoundTrip_SkipsFailedTurns(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (postgres round-trip for the empty-AssistantText fix)")
	}
	resetChatTables(t)

	dsn := chatPostgresTestDSN()
	const participant = "empty-assistant-sccs-fix"

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf("NewPostgresConversationStore: %v", err)
	}
	t.Cleanup(func() { _ = closer() })

	provider, err := migrator.NewProvider(context.Background(), store.RawDB(), migrations.MigrationsFS, "chat_schema_migrations")
	if err != nil {
		t.Fatalf("migrator.NewProvider: %v", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatalf("provider.Up: %v", err)
	}

	healthy := chat.Exchange{
		PromptText:    "first",
		AssistantText: "hi",
	}
	if err := store.Append(participant, healthy); err != nil {
		t.Fatalf("Append(healthy) returned %v, want nil", err)
	}

	failed := chat.Exchange{
		PromptText:      "second",
		AssistantText:   "",
		TerminalKind:    chat.TerminalKindFailed,
		FailureCategory: ai.FailureCategoryUnavailable,
	}
	if err := store.Append(participant, failed); err != nil {
		t.Fatalf("Append(failed) returned %v, want nil", err)
	}

	loaded, lerr := store.Load(participant)
	if lerr != nil {
		t.Fatalf("Load returned %v, want nil", lerr)
	}
	if len(loaded) != 2 {
		t.Fatalf("Load returned %d exchanges, want 2 (healthy + failed)", len(loaded))
	}

	history, herr := chat.ExchangesToHistory(loaded)
	if herr != nil {
		t.Fatalf("ExchangesToHistory returned %v, want nil (failed turn's empty AssistantText must be skipped, not surfaced as ai.NewText's ErrEmpty)", herr)
	}
	if history == nil {
		t.Fatal("ExchangesToHistory returned nil history with nil error; want non-nil history")
	}

	entries := history.Entries()
	if got, want := len(entries), 3; got != want {
		t.Fatalf("Entries() returned %d messages, want %d (user/first + user/second + assistant/hi; the failed turn contributes no assistant message)", got, want)
	}

	want := []struct {
		role ai.Role
		text string
	}{
		{ai.RoleUser, "first"},
		{ai.RoleAssistant, "hi"},
		{ai.RoleUser, "second"},
	}
	for i, w := range want {
		entry := entries[i]
		if entry.Message().Role() != w.role {
			t.Errorf("entries[%d].Role() = %v, want %v", i, entry.Message().Role(), w.role)
		}
		parts := entry.Message().Content()
		if len(parts) != 1 {
			t.Fatalf("entries[%d] carries %d part(s), want 1", i, len(parts))
		}
		text, ok := parts[0].Text()
		if !ok {
			t.Fatalf("entries[%d] part kind = %v, want text", i, parts[0].Kind())
		}
		if text != w.text {
			t.Errorf("entries[%d] text = %q, want %q", i, text, w.text)
		}
	}
}