// Package chat — PostgresConversationStore is the durable adapter
// behind the closed two-method ConversationStore port (R-CCS-010,
// R-CCS-011, R-CCS-012). It implements the same `Append` + `Load`
// methods as MemoryConversationStore; both adapters sit behind the
// single `chat.ConversationStore` interface.
//
// Design references:
//   - D-B: chat-package-local placement (this file; sibling to
//     chat/store.go, the port declaration).
//   - D-I: pgx-via-stdlib + database/sql (no pgxpool in v1).
//   - chat/migrations/0001_init.sql: the two tables (chat_conversations,
//     chat_exchanges) this adapter reads + writes.
//
// CH-07 WU-6 GREEN: Append + Load are production. The unit-level
// micro-tests exercise the WU-5 RED contract (still returns
// errNotImplemented for direct constructor paths), and the
// INTEGRATION=1 micro-test exercises the production INSERT/SELECT
// against chat_exchanges.
package chat

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// errNotImplemented is reserved for future stubs (no longer
// returned by Append/Load after WU-6 GREEN; WU-5 RED consumed it).
//
//nolint:unused // kept for future WU regressions on the stub contract.
var errNotImplemented = errors.New("chat: postgres adapter: not implemented")

// Default pool settings — match the DBA precedent at
// backend/database_administrator/src/migration/postgres/driver.go:69-73
// (MaxOpenConns=10, MaxIdleConns=5, ConnMaxLifetime=30m,
// ConnectTimeout=5s). CH-07 inherits these defaults; CH-04 owns
// env-var tuning (out of scope per design D-I).
const (
	defaultMaxOpenConns    = 10
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnectTimeout  = 5 * time.Second
)

// PostgresConversationStore is the durable chat adapter behind the
// ConversationStore port (R-CCS-011). It holds an open *sql.DB
// (acquired in the constructor, returned via the closer closure)
// and shares the two-method surface with MemoryConversationStore.
type PostgresConversationStore struct {
	db *sql.DB
}

// RawDB exposes the underlying *sql.DB. Test code (notably
// store_postgres_integration_test.go's cross-process scenario)
// uses this to run the chat-owned migrations against the same
// connection the adapter uses, without forcing callers to
// construct the migrator separately.
//
// Production code MUST NOT call RawDB — the migration runner is
// invoked by the composition root exactly once at startup
// (cmd/chat/main.go:198-217), and the adapter's own Append/Load
// are the only supported paths after that.
func (s *PostgresConversationStore) RawDB() *sql.DB {
	return s.db
}

// NewPostgresConversationStore opens the connection pool and
// returns the store + a closer. The closer MUST be invoked by the
// composition root AFTER the OTel pipeline flushes (post-listener
// shutdown, per design D-E), so spans land before the pool tears
// down.
//
// Returns:
//   - the *PostgresConversationStore on success;
//   - a closer func() error that releases the pool on shutdown;
//   - a non-nil error if open / ping fails.
func NewPostgresConversationStore(ctx context.Context, dsn string) (*PostgresConversationStore, func() error, error) {
	if dsn == "" {
		return nil, nil, errors.New("chat.NewPostgresConversationStore: empty dsn")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("chat.NewPostgresConversationStore: sql.Open: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, defaultConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("chat.NewPostgresConversationStore: ping: %w", err)
	}
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)

	store := &PostgresConversationStore{db: db}
	closer := func() error { return db.Close() }
	return store, closer, nil
}

// messageIDsJSON encodes a []string as the JSONB wire form pgx
// expects. An empty/nil slice encodes as the empty array `[]` so
// the column's NOT NULL DEFAULT '[]'::jsonb constraint never
// trips.
func messageIDsJSON(ids []string) []byte {
	if len(ids) == 0 {
		return []byte("[]")
	}
	b, err := json.Marshal(ids)
	if err != nil {
		// json.Marshal on []string never errors; the panic is
		// defensive but unreachable.
		panic(fmt.Sprintf("messageIDsJSON: marshal: %v", err))
	}
	return b
}

// decodeMessageIDs parses the JSONB wire form back into []string.
// pgx returns the column as []byte for JSONB; we json.Unmarshal.
func decodeMessageIDs(raw []byte) ([]string, error) {
	if len(raw) == 0 {
		return []string{}, nil
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, fmt.Errorf("decode %q: %w", string(raw), err)
	}
	return ids, nil
}

// parseTerminalKind maps the wire string back to the typed enum.
// Falls back to TerminalKindCompleted for unknown values so a
// future TerminalKind addition doesn't break round-trip on older
// binary versions.
func parseTerminalKind(s string) TerminalKind {
	switch s {
	case "completed":
		return TerminalKindCompleted
	case "cancelled":
		return TerminalKindCancelled
	case "failed":
		return TerminalKindFailed
	default:
		return TerminalKindCompleted
	}
}

// parseFailureCategory parses the wire string back into the
// FailureCategory from ai. The empty string maps to the zero
// value (the legitimate non-failed-turn value, per chat/store.go
// D-7); an unknown string also maps to the zero value rather than
// failing the round-trip — a future vocabulary addition
// gracefully degrades to zero on older binary versions.
func parseFailureCategory(s string) ai.FailureCategory {
	if s == "" {
		return 0
	}
	for _, c := range ai.FailureCategories() {
		if c.String() == s {
			return c
		}
	}
	return 0
}

// parseFinishReason parses the wire string into a *ai.FinishReason.
// The empty string maps to nil (the wire's FinishReason ABSENCE
// for cancelled/failed turns — chat/store.go D-7, R-CCS-004).
// Unknown values map to FinishReasonUnknown rather than failing
// the round-trip (per ai.NormalizeFinishReason's own posture).
func parseFinishReason(s string) *ai.FinishReason {
	if s == "" {
		return nil
	}
	fr := ai.NormalizeFinishReason(s)
	return &fr
}

// Append records ex under participantID. The store is responsible
// for assigning Exchange.Position (insertion order within the
// participant — R-CCS-001). On duplicate or invalid input, a
// non-nil error may be returned.
//
// CH-07 WU-6 GREEN implementation:
//   1. BEGIN
//   2. INSERT INTO chat_conversations (participant_id) VALUES ($1)
//      ON CONFLICT DO NOTHING (ensure the parent row exists)
//   3. INSERT INTO chat_exchanges (participant_id, position, ...)
//      VALUES ($1, COALESCE((SELECT MAX(position) + 1 FROM
//      chat_exchanges WHERE participant_id = $1), 0), ...) —
//      store-assigned position via PRIMARY KEY semantics
//   4. COMMIT
//   5. On error, rollback; the transaction guarantees the parent
//      row INSERT and the exchange INSERT land together (or not at
//      all).
//
// message_ids is serialised as JSONB (pgx marshals []string ↔
// JSONB transparently). terminal_kind is stored as the TerminalKind
// String() form (completed / cancelled / failed); Load deserialises
// back to TerminalKind by mapping the strings to constants.
func (s *PostgresConversationStore) Append(participantID string, ex Exchange) error {
	if participantID == "" {
		return ErrEmptyParticipantID
	}

	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("chat.PostgresConversationStore.Append: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Ensure the parent row exists. ON CONFLICT DO NOTHING makes
	// the call idempotent — the row stays untouched on second +
	// subsequent Appends for the same participant.
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO chat_conversations (participant_id) VALUES ($1) ON CONFLICT (participant_id) DO NOTHING`,
		participantID,
	); err != nil {
		return fmt.Errorf("chat.PostgresConversationStore.Append: insert chat_conversations: %w", err)
	}

	// 2. Compute the next position via COALESCE — first row gets
	// position 0, subsequent rows get MAX(position) + 1. The
	// PRIMARY KEY (participant_id, position) enforces uniqueness.
	var nextPos int
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(position), -1) + 1 FROM chat_exchanges WHERE participant_id = $1`,
		participantID,
	).Scan(&nextPos); err != nil {
		return fmt.Errorf("chat.PostgresConversationStore.Append: compute position: %w", err)
	}

	// 3. Insert the exchange row. message_ids is JSONB-encoded
	// from the []string field via the driver.
	var finishReason *string
	if ex.FinishReason != nil {
		fr := ex.FinishReason.String()
		finishReason = &fr
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO chat_exchanges (
			participant_id, position, prompt_text, assistant_text, partial,
			terminal_kind, failure_category, finish_reason, message_ids
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9::jsonb)`,
		participantID,
		nextPos,
		ex.PromptText,
		ex.AssistantText,
		ex.Partial,
		ex.TerminalKind.String(),
		ex.FailureCategory.String(),
		finishReason,
		messageIDsJSON(ex.MessageIDs),
	); err != nil {
		return fmt.Errorf("chat.PostgresConversationStore.Append: insert chat_exchanges: %w", err)
	}

	// 4. Bump the parent row's updated_at so an operator querying
	// chat_conversations can find the most-recently-active
	// participant without scanning chat_exchanges.
	if _, err := tx.ExecContext(ctx,
		`UPDATE chat_conversations SET updated_at = now() WHERE participant_id = $1`,
		participantID,
	); err != nil {
		return fmt.Errorf("chat.PostgresConversationStore.Append: update chat_conversations.updated_at: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("chat.PostgresConversationStore.Append: commit: %w", err)
	}
	return nil
}

// List returns the participant-scoped conversation summaries for
// participantID (R-CCS-013, the third additive method on the
// ConversationStore port). At v1 (one conversation per participant
// per D-1) the result slice carries 0 or 1 entry.
//
// Implementation (CH-08 WU-3 GREEN):
//   - SELECT participant_id, updated_at,
//            COALESCE((SELECT MAX(position)+1 FROM chat_exchanges
//                      WHERE participant_id = $1), 0) AS turn_count
//     FROM chat_conversations
//    WHERE participant_id = $1
//    ORDER BY updated_at DESC
//   - The correlated subquery reads chat_exchanges' position counter
//     (R-CCS-001's `position` is contiguous per participant; MAX+1
//     equals COUNT(*) but never over-counts on gaps, future-proof).
//   - A miss (no chat_conversations row) returns a non-nil empty
//     slice — the empty list is success, not not-found (S-CCS-018,
//     S-CRI-004). The chat_conversations row exists iff Append has
//     run for the participant (the Append path INSERTs with
//     ON CONFLICT DO NOTHING).
//   - The defensive copy follows Load's pattern (NFR-CCS-004).
func (s *PostgresConversationStore) List(participantID string) ([]ConversationSummary, error) {
	if participantID == "" {
		return nil, ErrEmptyParticipantID
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT participant_id, updated_at,
		        COALESCE((SELECT MAX(position)+1 FROM chat_exchanges
		                  WHERE participant_id = $1), 0) AS turn_count
		   FROM chat_conversations
		  WHERE participant_id = $1
		  ORDER BY updated_at DESC`,
		participantID,
		participantID,
	)
	if err != nil {
		return nil, fmt.Errorf("chat.PostgresConversationStore.List: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]ConversationSummary, 0)
	for rows.Next() {
		var (
			pid        string
			updatedAt  time.Time
			turnCount  int
		)
		if err := rows.Scan(&pid, &updatedAt, &turnCount); err != nil {
			return nil, fmt.Errorf("chat.PostgresConversationStore.List: scan: %w", err)
		}
		out = append(out, ConversationSummary{
			ConversationID: pid,
			LastActivityAt: updatedAt,
			TurnCount:      turnCount,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("chat.PostgresConversationStore.List: rows: %w", err)
	}
	return out, nil
}

// Load returns every exchange recorded for participantID, in
// insertion order. A miss returns (nil, ErrConversationNotFound).
// The returned slice is a fresh copy on every call (NFR-CCS-004).
//
// CH-07 WU-6 GREEN implementation:
//   - SELECT position, prompt_text, assistant_text, partial,
//     terminal_kind, failure_category, finish_reason, message_ids
//     FROM chat_exchanges WHERE participant_id = $1
//     ORDER BY position ASC
//   - Reconstruct each Exchange from the row (decode message_ids
//     JSONB → []string; map terminal_kind string back to the
//     typed enum; map finish_reason string back to *ai.FinishReason
//     if non-empty)
//   - Defensive copy of the slice — caller-side mutation cannot
//     corrupt the adapter's read state.
func (s *PostgresConversationStore) Load(participantID string) ([]Exchange, error) {
	if participantID == "" {
		return nil, ErrEmptyParticipantID
	}

	ctx := context.Background()
	rows, err := s.db.QueryContext(ctx,
		`SELECT position, prompt_text, assistant_text, partial,
		        terminal_kind, failure_category, finish_reason, message_ids
		   FROM chat_exchanges
		  WHERE participant_id = $1
		  ORDER BY position ASC`,
		participantID,
	)
	if err != nil {
		return nil, fmt.Errorf("chat.PostgresConversationStore.Load: query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]Exchange, 0)
	for rows.Next() {
		var (
			position        int
			promptText      string
			assistantText   string
			partial         bool
			terminalKindStr string
			failureCatStr   string
			finishReason    sql.NullString
			messageIDsRaw   []byte
		)
		if err := rows.Scan(
			&position,
			&promptText,
			&assistantText,
			&partial,
			&terminalKindStr,
			&failureCatStr,
			&finishReason,
			&messageIDsRaw,
		); err != nil {
			return nil, fmt.Errorf("chat.PostgresConversationStore.Load: scan: %w", err)
		}
		ids, err := decodeMessageIDs(messageIDsRaw)
		if err != nil {
			return nil, fmt.Errorf("chat.PostgresConversationStore.Load: decode message_ids: %w", err)
		}
		var fr *ai.FinishReason
		if finishReason.Valid && finishReason.String != "" {
			fr = parseFinishReason(finishReason.String)
		}
		out = append(out, Exchange{
			Position:        position,
			PromptText:      promptText,
			AssistantText:   assistantText,
			Partial:         partial,
			TerminalKind:    parseTerminalKind(terminalKindStr),
			FailureCategory: parseFailureCategory(failureCatStr),
			FinishReason:    fr,
			MessageIDs:      ids,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("chat.PostgresConversationStore.Load: rows: %w", err)
	}

	if len(out) == 0 {
		// Distinguish "no rows" from "no conversation" — the
		// composition root maps ErrConversationNotFound to nil
		// exchanges (CH-06 factory closure). The chat_conversations
		// row may exist without any chat_exchanges (a participant
		// constructed via the registry but with no turns yet); that
		// case also returns ErrConversationNotFound.
		return nil, ErrConversationNotFound
	}

	// Defensive copy — caller-side mutation cannot corrupt the
	// adapter's read state (NFR-CCS-004). make + copy gives an
	// independent backing array; the slice header is itself a
	// value, so returning it without copying would leak the
	// adapter's growable capacity.
	cp := make([]Exchange, len(out))
	copy(cp, out)
	return cp, nil
}