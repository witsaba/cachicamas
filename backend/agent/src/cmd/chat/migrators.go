// Package main — migration runner extraction.
//
// runArchetypeMigrations is the chat composition root's wrapper around
// the archetype package's forward-only migration runner. The wrapper
// lives in its own file so that adding migrations 0004..0007 (PR-1 of
// cachicamas-archetype-system-foundation) and 0008.. (future) does
// not push cmd/chat/main.go past its 350-line budget.
//
// The runner applies migrations against the chat binary's Postgres
// pool (CACHICAMAS_CHAT_STORE_DSN) using the archetype schema's own
// migration table (`archetype_schema_migrations`) — separate from
// `chat_schema_migrations` so the two namespaces stay decoupled at
// the migration-runner layer.
//
// Why this lives next to main.go (not in src/archetype/migrations):
// the wrapper depends on the chat composition root's *sql.DB pool;
// moving it to src/archetype/migrations would force that package to
// take a dependency on the chat binary's wiring. Composition-root-
// only is the layered choice per ADR 0005 § D3 and chat/doc.go:93-107.
package main

import (
	"context"
	"database/sql"
	"fmt"

	archetypeMigrations "github.com/cachicamas/backend/agent/src/archetype/migrations"
	"github.com/cachicamas/backend/agent/src/chat/migrator"
)

// runArchetypeMigrations applies every unapplied *.sql file under
// archetype/migrations/ in lexical order against the supplied pool.
// The runner is idempotent (guarded by archetype_schema_migrations
// rows); re-running on a fresh boot is a no-op when the schema is
// already current.
//
// Migrations 0006 and 0007 ship with paired Go wrappers
// (0006_runner.go, 0007_runner.go) that run the SQL directly via
// ExecContext because the chat composition root's allowlist refuses
// DROP CONSTRAINT / DROP COLUMN / ALTER COLUMN / ADD PRIMARY KEY / ADD
// CONSTRAINT FOREIGN KEY / UPDATE. Each wrapper self-records the
// completion in archetype_schema_migrations (when that table exists)
// so subsequent boots skip the body and rely on the goose runner for
// the remaining migrations.
//
// Returns the underlying error from the runner verbatim so the
// composition root can surface it to the operator with the same
// formatting as every other startup error.
func runArchetypeMigrations(ctx context.Context, db *sql.DB) error {
	if err := archetypeMigrations.Run0006IfNeeded(ctx, db); err != nil {
		return fmt.Errorf("apply archetype migration 0006 wrapper: %w", err)
	}
	if err := archetypeMigrations.Run0007IfNeeded(ctx, db); err != nil {
		return fmt.Errorf("apply archetype migration 0007 wrapper: %w", err)
	}
	provider, err := migrator.NewProvider(ctx, db, archetypeMigrations.MigrationsFS, "archetype_schema_migrations")
	if err != nil {
		return fmt.Errorf("build archetype migration provider: %w", err)
	}
	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply archetype migrations: %w", err)
	}
	return nil
}
