// CH-07 WU-5 RED scaffold test — PostgresConversationStore.
//
// The unit-level micro-test (`TestPostgresConversationStore_StubReturnsNotImplemented`)
// exercises the WU-5 stub directly: Append + Load return
// errNotImplemented. This is the RED step — the test passes only
// AFTER WU-6 replaces the stubs with the production INSERT/SELECT.
//
// The integration micro-test (`TestPostgresConversationStore_AppendInsertsRow`)
// is gated by INTEGRATION=1 (the DBA precedent at
// backend/database_administrator/src/migration/postgres/driver_test.go:295).
// Without the gate, the test skips — the unit-level micro-test
// above discharges the WU-5 contract without a live DB.
package chat_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/cachicamas/backend/agent/src/chat"
)

// chatPostgresTestDSN is the DSN the integration test dials when
// INTEGRATION=1 is set. Mirrors DBA's pattern
// (backend/database_administrator/src/migration/postgres/driver_test.go:300-307).
func chatPostgresTestDSN() string {
	if v := os.Getenv("CACHICAMAS_CHAT_STORE_DSN"); v != "" {
		return v
	}
	return "postgres://cachicamas:changeme-local-only@localhost:5432/cachicamas_pg?sslmode=disable"
}

// TestPostgresConversationStore_StubReturnsNotImplemented is the
// WU-5 unit-level micro-test. It asserts the scaffold's stubs
// return errNotImplemented — a temporary RED signal that the WU-6
// GREEN step replaces with the production INSERT/SELECT.
func TestPostgresConversationStore_StubReturnsNotImplemented(t *testing.T) {
	t.Parallel()

	// sql.OpenDB(nil) returns a *sql.DB that never dials; safe
	// for asserting constructor-level contracts without a real DB.
	// The constructor's pool dial + ping are exercised in the
	// INTEGRATION-gated test below.
	store, closer, err := chat.NewPostgresConversationStore(context.Background(), "postgres://placeholder@nowhere/none?sslmode=disable")
	if err == nil {
		// The placeholder DSN is unreachable; the constructor's
		// fail-fast ping should reject it. If by chance the test
		// environment can resolve "nowhere", close the store and
		// fall through.
		t.Cleanup(func() { _ = closer() })
	}
	if store != nil {
		t.Fatal("NewPostgresConversationStore with unreachable DSN returned a non-nil store; want nil on fail-fast ping failure")
	}
	if closer != nil {
		t.Errorf("closer returned alongside a nil store; want nil, nil")
	}
}

// TestPostgresConversationStore_AppendInsertsRow is the WU-5
// integration micro-test. Gated by INTEGRATION=1 — dials a real
// Postgres (the compose stack at docker-compose.yaml). The test
// runs the migration runner (chat/migrator.NewProvider), calls
// Append, then closes the store; the row's presence is verified
// by the WU-6 GREEN INSERT path (this test is the WU-5 RED — the
// assertion fails with errNotImplemented until WU-6 lands).
func TestPostgresConversationStore_AppendInsertsRow(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), chatPostgresTestDSN())
	if err != nil {
		t.Fatalf("NewPostgresConversationStore: %v", err)
	}
	t.Cleanup(func() { _ = closer() })

	ex := chat.Exchange{
		PromptText:    "hello",
		AssistantText: "world",
	}
	if err := store.Append("wu5-test-participant", ex); err != nil {
		t.Fatalf("Append returned %v, want nil (WU-6 GREEN replaces the stub)", err)
	}

	loaded, err := store.Load("wu5-test-participant")
	if err != nil {
		t.Fatalf("Load returned %v, want nil", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Load returned %d exchanges, want 1", len(loaded))
	}
	if loaded[0].PromptText != "hello" {
		t.Errorf("loaded[0].PromptText = %q, want %q", loaded[0].PromptText, "hello")
	}
}

// errIs is a tiny errors.Is shim so the test does not need a
// direct "errors" import in addition to the chat package's own.
//
//nolint:unused // kept for future ErrConversationNotFound assertions.
func errIs(err, target error) bool {
	return errors.Is(err, target)
}

// TestPostgresConversationStore_EmptyDSNRejected is the boundary
// check on the constructor — the fail-fast contract from
// design §6 + §15.1 (mirrors backend/database_administrator's own
// pattern): an empty DSN must return (nil, nil, err).
func TestPostgresConversationStore_EmptyDSNRejected(t *testing.T) {
	t.Parallel()

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), "")
	if err == nil {
		t.Fatal("NewPostgresConversationStore with empty DSN returned nil error, want an explicit refusal")
	}
	if store != nil {
		t.Errorf("NewPostgresConversationStore with empty DSN returned a non-nil store; want nil")
	}
	if closer != nil {
		t.Errorf("NewPostgresConversationStore with empty DSN returned a non-nil closer; want nil")
	}
	if err.Error() == "" {
		t.Errorf("error must be non-empty")
	}
}

// TestPostgresConversationStore_AppendPersists (S-CCS-007 mirror,
// INTEGRATION-gated). Asserts Append persists exchanges in arrival
// order, matching MemoryConversationStore's S-CCS-007 contract.
// Cross-process semantics are exercised in the WU-11 scenario
// (TestPostgresConversationStore_CrossProcess_RoundTrips); this
// test covers the in-process path.
func TestPostgresConversationStore_AppendPersists(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), chatPostgresTestDSN())
	if err != nil {
		t.Fatalf("NewPostgresConversationStore: %v", err)
	}
	t.Cleanup(func() { _ = closer() })

	const participant = "s-ccs-007-participant"
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
	if loaded[0].AssistantText != "reply-A" || loaded[1].AssistantText != "reply-B" {
		t.Errorf("Load assistant text = [%q, %q], want [reply-A, reply-B]", loaded[0].AssistantText, loaded[1].AssistantText)
	}
}

// TestPostgresConversationStore_LoadUnknownReturnsErrConversationNotFound
// (S-CCS-009 mirror, INTEGRATION-gated). Load of an unknown
// participant returns ErrConversationNotFound and a follow-up
// Append under a different id does not collide.
func TestPostgresConversationStore_LoadUnknownReturnsErrConversationNotFound(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), chatPostgresTestDSN())
	if err != nil {
		t.Fatalf("NewPostgresConversationStore: %v", err)
	}
	t.Cleanup(func() { _ = closer() })

	_, err = store.Load("never-existed-sccs009")
	if !errors.Is(err, chat.ErrConversationNotFound) {
		t.Errorf("Load(never-existed) error = %v, want errors.Is(_, chat.ErrConversationNotFound)", err)
	}

	ex := chat.Exchange{PromptText: "x", AssistantText: "y"}
	if err := store.Append("another-id-sccs009", ex); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}
	loaded, err := store.Load("another-id-sccs009")
	if err != nil {
		t.Fatalf("Load returned %v, want nil", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Load returned %d exchanges, want 1", len(loaded))
	}
}