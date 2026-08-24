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

	"github.com/cachicamas/backend/agent/src/chat"
	"github.com/cachicamas/backend/agent/src/chat/migrations"
	"github.com/cachicamas/backend/agent/src/chat/migrator"
)

func TestPostgresConversationStore_CrossProcess_RoundTrips(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (S-CCS-010 requires a real Postgres; two adapters over the same DSN)")
	}

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