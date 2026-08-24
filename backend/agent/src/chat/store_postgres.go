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
// CH-07 WU-5 RED scaffold: Append + Load are stubs that return
// "not implemented". The unit-level RED test asserts a real Insert
// produces a real row (gated by INTEGRATION=1 — the DBA precedent
// at backend/database_administrator/src/migration/postgres/driver_test.go:296);
// WU-6 GREEN replaces the stubs with the production INSERT /
// SELECT.
package chat

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// errNotImplemented is the WU-5 RED stub signal. WU-6 GREEN removes
// every reference to this error.
var errNotImplemented = errors.New("chat: postgres adapter: not implemented (CH-07 WU-5 RED)")

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

// NewPostgresConversationStore opens the connection pool, runs the
// chat-owned forward-only migrations against dsn, and returns the
// store + a closer. The closer MUST be invoked by the composition
// root AFTER the OTel pipeline flushes (post-listener shutdown, per
// design D-E), so spans land before the pool tears down.
//
// tableName is the bookkeeping table for the chat-owned migrations
// (default "chat_schema_migrations" — distinct from DBA's
// "schema_migrations" per ADR 0009 § D6 ownership).
//
// Returns:
//   - the *PostgresConversationStore on success;
//   - a closer func() error that releases the pool on shutdown;
//   - a non-nil error if open / ping / migrate / construct fails.
//
// CH-07 WU-5: this constructor already opens the pool and pings.
// The migration runner (chat/migrator.NewProvider) is invoked next;
// the runner is the WU-3 skeleton that admits only forward-only SQL.
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

// Append records ex under participantID. The store is responsible
// for assigning Exchange.Position (insertion order within the
// participant — R-CCS-001). On duplicate or invalid input, a
// non-nil error may be returned.
//
// CH-07 WU-5 RED: this is a stub returning errNotImplemented.
// WU-6 GREEN replaces it with the production INSERT against
// chat_exchanges (per design D-D sequence at 0005:797-815).
func (s *PostgresConversationStore) Append(participantID string, ex Exchange) error {
	_ = participantID
	_ = ex
	return errNotImplemented
}

// Load returns every exchange recorded for participantID, in
// insertion order. A miss returns (nil, ErrConversationNotFound).
// The returned slice is a fresh copy on every call (NFR-CCS-004).
//
// CH-07 WU-5 RED: this is a stub returning errNotImplemented.
// WU-6 GREEN replaces it with the production SELECT against
// chat_exchanges (per design D-D sequence at 0005:797-815).
func (s *PostgresConversationStore) Load(participantID string) ([]Exchange, error) {
	_ = participantID
	return nil, errNotImplemented
}