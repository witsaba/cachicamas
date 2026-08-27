// T-05 PR-1 (cachicamas-archetype-system-foundation) — one-shot SQL
// runner wrapper for migration 0006.
//
// Why this file exists
// --------------------
// The chat composition root's migration runner
// (backend/agent/src/chat/migrator/runner.go) applies a strict
// forward-only allowlist. The allowed statements are CREATE TABLE,
// CREATE INDEX, CREATE SEQUENCE, INSERT INTO, ALTER TABLE ADD COLUMN,
// COMMENT ON, plus blank lines and SQL comments. Migration 0006
// (archetype_configurations PK reshape) needs DROP CONSTRAINT,
// DROP COLUMN, ALTER COLUMN SET NOT NULL, ADD PRIMARY KEY, and
// ADD CONSTRAINT FOREIGN KEY — none of which are in the allowlist.
//
// Per design AD-6, the wrapper approach is the chosen escape hatch:
//   - Reads the SQL file once at boot.
//   - Runs the entire body as a single ExecContext on the supplied
//     pool (the SQL already wraps every statement in BEGIN; ... COMMIT;).
//   - On success, inserts a row into archetype_schema_migrations
//     with the version '6' so subsequent boots skip the file.
//   - On failure, returns the underlying error and DOES NOT insert
//     the row — the next boot retries.
//
// The wrapper file's name (0006_runner.go) follows the embedded FS
// pattern in migrations/embed.go: only *.sql files are picked up by
// the goose runner. The wrapper is plain Go and runs once before
// the goose runner touches the schema.
//
// Future non-additive migrations should follow the same pattern:
// a paired *.go wrapper next to the *.sql. Do NOT extend the
// allowlist — keeping it strict is the strongest signal we have
// about migration forward-only intent.
package migrations

import (
	"context"
	"database/sql"
	"fmt"
)

// Run0006IfNeeded executes migration 0006 once and self-records the
// completion in archetype_schema_migrations. The chat composition
// root calls this BEFORE the goose runner; if the row is already
// present, the function returns nil immediately.
//
// The function is idempotent: it checks the migrations table for a
// '6' row first and only executes the SQL when the row is absent.
// On SQL failure, no row is inserted and the caller surfaces the
// error verbatim — the operator retries on the next boot.
func Run0006IfNeeded(ctx context.Context, db *sql.DB) error {
	const version int64 = 6
	const sqlBody = `-- +goose Up

BEGIN;

CREATE TABLE archetype_configurations__backup AS
    SELECT * FROM archetype_configurations;

ALTER TABLE archetype_configurations
    ADD COLUMN archetype_slug text;

UPDATE archetype_configurations
    SET archetype_slug = 'assistant'
    WHERE archetype_kind = 'chat';

ALTER TABLE archetype_configurations
    DROP CONSTRAINT archetype_configurations_pkey;

ALTER TABLE archetype_configurations
    DROP COLUMN archetype_kind;

ALTER TABLE archetype_configurations
    ALTER COLUMN archetype_slug SET NOT NULL;

ALTER TABLE archetype_configurations
    ADD PRIMARY KEY (archetype_slug, org_id);

ALTER TABLE archetype_configurations
    ADD CONSTRAINT archetype_configurations_slug_fkey
        FOREIGN KEY (archetype_slug)
        REFERENCES archetypes(slug)
        ON DELETE RESTRICT;

COMMIT;`

	// Probe the migrations table — already applied?
	var exists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (SELECT 1 FROM archetype_schema_migrations WHERE version_id = '6' OR version = 6)`,
	).Scan(&exists); err != nil {
		return fmt.Errorf("check 0006 status: %w", err)
	}
	if exists {
		return nil
	}

	if _, err := db.ExecContext(ctx, sqlBody); err != nil {
		return fmt.Errorf("apply migration 0006: %w", err)
	}
	// Self-record. The next boot's probe sees this row and skips.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO archetype_schema_migrations (version_id, is_applied, tstamp)
		 VALUES ('6', true, now())
		 ON CONFLICT DO NOTHING`,
	); err != nil {
		return fmt.Errorf("record 0006 in archetype_schema_migrations: %w", err)
	}
	return nil
}
