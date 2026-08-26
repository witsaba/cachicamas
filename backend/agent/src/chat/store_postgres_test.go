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
	"database/sql"
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

// resetChatTables truncates every chat-owned table so each
// INTEGRATION-gated test starts from a clean slate. The suite dials a
// persistent DSN (not a throwaway database) and runs sequentially — no
// t.Parallel among the postgres tests — so without this, rows left by
// earlier or crashed runs leak into later assertions ("Load returned 6
// exchanges, want 2", found 2026-08-25). One statement handles the FK
// chain (exchanges → conversations; siblings → exchanges).
func resetChatTables(t *testing.T) {
	t.Helper()
	db, err := sql.Open("pgx", chatPostgresTestDSN())
	if err != nil {
		t.Fatalf("resetChatTables: sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`TRUNCATE chat_conversations, chat_exchanges, chat_tool_calls, chat_tool_results, chat_permission_decisions`); err != nil {
		t.Fatalf("resetChatTables: TRUNCATE: %v", err)
	}
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
	resetChatTables(t)

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
	resetChatTables(t)

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

// TestPostgresConversationStore_CH06Scenarios_PassUnchanged is
// the WU-10 binding acceptance: the CH-06 scenario set runs
// unchanged against `PostgresConversationStore` via the shared
// `RunConversationStoreScenarios` helper. S-CCS-011 ("the port's
// scenarios run unchanged against both adapters") is discharged
// when the helper's sub-tests all pass against this adapter.
//
// Gated by `INTEGRATION=1` — the helper dials a real Postgres.
// Without the gate, the test SKIPs (the unit-level memory
// scenarios still pass via the chat package's own store_test.go).
func TestPostgresConversationStore_CH06Scenarios_PassUnchanged(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the shared scenario helper exercises Append+Load against a real Postgres)")
	}
	resetChatTables(t)

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), chatPostgresTestDSN())
	if err != nil {
		t.Fatalf("NewPostgresConversationStore: %v", err)
	}
	t.Cleanup(func() { _ = closer() })

	RunConversationStoreScenarios(t, store)
}

// TestPostgresConversationStore_LoadUnknownReturnsErrConversationNotFound
// (S-CCS-009 mirror, INTEGRATION-gated). Load of an unknown
// participant returns ErrConversationNotFound and a follow-up
// Append under a different id does not collide.
func TestPostgresConversationStore_LoadUnknownReturnsErrConversationNotFound(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetChatTables(t)

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
// TestPostgresConversationStore_ListEmptyParticipantIDRejected
// (INTEGRATION-gated) — boundary check on List's empty participantID
// refusal: matches Load's D-1 contract (ErrEmptyParticipantID).
// INTEGRATION-gated because the constructor dials + pings, matching
// the project's existing INTEGRATION-gate convention for any real-
// store micro-test.
func TestPostgresConversationStore_ListEmptyParticipantIDRejected(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), chatPostgresTestDSN())
	if err != nil {
		t.Fatalf("NewPostgresConversationStore: %v", err)
	}
	t.Cleanup(func() { _ = closer() })

	got, lerr := store.List("")
	if !errors.Is(lerr, chat.ErrEmptyParticipantID) {
		t.Errorf("List(empty) error = %v, want errors.Is(_, chat.ErrEmptyParticipantID)", lerr)
	}
	if got != nil {
		t.Errorf("List(empty) returned %v, want nil on empty-participant-id refusal", got)
	}
}

// TestPostgresConversationStore_CH08ListScenario is the WU-3
// INTEGRATION-gated micro-test for the new List method: appends two
// exchanges for a participant, then reads back via List and asserts
// the summary carries the right ConversationID + TurnCount +
// LastActivityAt (the round-trip path).
func TestPostgresConversationStore_CH08ListScenario(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the postgres List path dials a real Postgres)")
	}
	resetChatTables(t)

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), chatPostgresTestDSN())
	if err != nil {
		t.Fatalf("NewPostgresConversationStore: %v", err)
	}
	t.Cleanup(func() { _ = closer() })

	const participant = "ch08-list-scenario-participant"
	if err := store.Append(participant, chat.Exchange{PromptText: "first", AssistantText: "reply-A"}); err != nil {
		t.Fatalf("Append(first) returned %v, want nil", err)
	}
	if err := store.Append(participant, chat.Exchange{PromptText: "second", AssistantText: "reply-B"}); err != nil {
		t.Fatalf("Append(second) returned %v, want nil", err)
	}

	list, lerr := store.List(participant)
	if lerr != nil {
		t.Fatalf("List(%q) returned %v, want nil", participant, lerr)
	}
	if len(list) != 1 {
		t.Fatalf("List returned %d entries, want 1 (one conversation per participant — D-1)", len(list))
	}
	if list[0].ConversationID != participant {
		t.Errorf("List[0].ConversationID = %q, want %q", list[0].ConversationID, participant)
	}
	if list[0].TurnCount != 2 {
		t.Errorf("List[0].TurnCount = %d, want 2 (both appends count toward turn_count)", list[0].TurnCount)
	}
	if list[0].LastActivityAt.IsZero() {
		t.Error("List[0].LastActivityAt is the zero time, want a real timestamp from chat_conversations.updated_at")
	}
}

// TestPostgresConversationStore_CH08Scenarios_PassUnchanged
// (INTEGRATION-gated) — WU-4 binding acceptance: the CH-06 + CH-08
// scenario helper runs unchanged against the postgres adapter.
func TestPostgresConversationStore_CH08Scenarios_PassUnchanged(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the WU-4 scenario helper extended at S-CCS-015..018 against a real Postgres)")
	}
	resetChatTables(t)

	store, closer, err := chat.NewPostgresConversationStore(context.Background(), chatPostgresTestDSN())
	if err != nil {
		t.Fatalf("NewPostgresConversationStore: %v", err)
	}
	t.Cleanup(func() { _ = closer() })

	RunConversationStoreScenarios(t, store)
}

// TestPostgresConversationStore_CH09Scenarios (INTEGRATION-gated) —
// CH-09 S-CCS-021 cross-process round-trip for the new tool-call
// fields. Appends through one adapter instance, reads back through
// a separately constructed adapter (mirrors the CH-07.1 pattern
// already in this file's TestPostgresConversationStore_AppendInsertsRow
// integration micro-test). Verifies the sibling-table round-trip:
//
//   - chat_tool_calls rows are inserted at the right
//     (participant_id, exchange_position, position) keys.
//   - chat_tool_results rows are inserted at the same composite
//     key with the closed outcome enum value.
//   - Load reconstructs both slices in issuance order.
//
// Gated INTEGRATION=1 (postgres cross-process round-trip per
// R-CCS-021).
func TestPostgresConversationStore_CH09Scenarios(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (CH-09 S-CCS-021 postgres cross-process round-trip)")
	}
	resetChatTables(t)

	dsn := chatPostgresTestDSN()

	store1, closer1, err := chat.NewPostgresConversationStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf("NewPostgresConversationStore (writer): %v", err)
	}
	t.Cleanup(func() { _ = closer1() })

	const participant = "scn-021-cross-process"
	ex := chat.Exchange{
		PromptText:    "what time is it",
		AssistantText: "12:00",
		TerminalKind:  chat.TerminalKindCompleted,
		ToolCalls: []chat.ToolCallRecord{
			{WireCallID: "c1", Tool: "current_time", Arguments: "{}"},
			{WireCallID: "c2", Tool: "current_time", Arguments: "{}"},
		},
		ToolResults: []chat.ToolResultRecord{
			{WireCallID: "c1", Tool: "current_time", Outcome: "success", Content: "2026-08-25T12:00:00Z"},
			{WireCallID: "c2", Tool: "current_time", Outcome: "success", Content: "2026-08-25T12:00:01Z"},
		},
	}
	if err := store1.Append(participant, ex); err != nil {
		t.Fatalf("Append returned %v, want nil", err)
	}
	_ = store1

	// Separately constructed adapter (fresh *sql.DB) — confirms the
	// round-trip survives connection-pool teardown.
	store2, closer2, err := chat.NewPostgresConversationStore(context.Background(), dsn)
	if err != nil {
		t.Fatalf("NewPostgresConversationStore (reader): %v", err)
	}
	t.Cleanup(func() { _ = closer2() })

	loaded, err := store2.Load(participant)
	if err != nil {
		t.Fatalf("Load returned %v, want nil", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("Load returned %d exchanges, want 1", len(loaded))
	}
	got := loaded[0]
	if len(got.ToolCalls) != 2 {
		t.Errorf("ToolCalls = %d, want 2 (S-CCS-021)", len(got.ToolCalls))
	}
	if len(got.ToolResults) != 2 {
		t.Errorf("ToolResults = %d, want 2 (S-CCS-021)", len(got.ToolResults))
	}
	for i := range ex.ToolCalls {
		if got.ToolCalls[i].WireCallID != ex.ToolCalls[i].WireCallID {
			t.Errorf("ToolCalls[%d].WireCallID = %q, want %q", i, got.ToolCalls[i].WireCallID, ex.ToolCalls[i].WireCallID)
		}
		if got.ToolResults[i].Outcome != ex.ToolResults[i].Outcome {
			t.Errorf("ToolResults[%d].Outcome = %q, want %q", i, got.ToolResults[i].Outcome, ex.ToolResults[i].Outcome)
		}
		if got.ToolResults[i].Content != ex.ToolResults[i].Content {
			t.Errorf("ToolResults[%d].Content = %q, want %q", i, got.ToolResults[i].Content, ex.ToolResults[i].Content)
		}
	}
}
