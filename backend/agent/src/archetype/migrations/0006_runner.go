// Package migrations — T-05 PR-1
// (cachicamas-archetype-system-foundation) — one-shot SQL runner wrapper
// for migration 0006.
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
//   - Reads the SQL body once at boot.
//   - Probes the schema via information_schema so the probe does NOT
//     depend on the goose runner having created archetype_schema_migrations
//     yet (chicken-and-egg on a fresh database).
//   - Runs the entire body as a single ExecContext on the supplied
//     pool (the SQL already wraps every statement in BEGIN; ... COMMIT;).
//   - On success, attempts to record completion in
//     archetype_schema_migrations — but ONLY if that table already
//     exists. Self-record is best-effort, not load-bearing: column
//     existence is the canonical idempotency signal.
//
// Idempotency mechanism (information_schema-based)
// ------------------------------------------------
// The wrapper runs BEFORE the goose runner, so archetype_schema_migrations
// is not guaranteed to exist when the wrapper executes — it is only
// created once the goose runner applies its very first migration. On a
// fresh database that table is absent on the first call, so any probe
// against archetype_schema_migrations trips SQLSTATE 42P01 and the chat
// binary crash-loops.
//
// Instead, the wrapper probes:
//   1. information_schema.tables for the prerequisite table
//      archetype_configurations. If absent, return nil immediately —
//      the goose runner will apply migrations 0001..0005 next and the
//      wrapper will be re-invoked on the following boot.
//   2. information_schema.columns for the archetype_slug column on
//      archetype_configurations. If present, return nil — 0006 has
//      already been applied (the column does not exist before 0006
//      runs and exists permanently afterwards, so its presence is the
//      canonical idempotency signal).
//   3. information_schema.tables for archetype_schema_migrations when
//      self-recording. If absent, skip the insert — the column-existence
//      check will still keep the wrapper idempotent on subsequent boots.
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

// Run0006IfNeeded executes migration 0006 once and uses schema-level
// probes (information_schema) as the idempotency mechanism so it can
// run safely both before and after the goose runner has touched the
// schema. The chat composition root calls this BEFORE the goose
// runner.
//
// Probe order:
//   1. archetype_configurations table — if absent, this is a fresh
//      database; return nil and let the goose runner create the
//      prerequisite tables. We will be re-invoked on the next boot.
//   2. archetype_slug column on archetype_configurations — if present,
//      0006 has already been applied; return nil.
//   3. Otherwise apply the SQL body as a single ExecContext.
//   4. Best-effort self-record into archetype_schema_migrations,
//      conditional on that table already existing.
//
// On SQL failure, no row is inserted and the caller surfaces the
// error verbatim — the operator retries on the next boot.
func Run0006IfNeeded(ctx context.Context, db *sql.DB) error {
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

	// 1. Fresh-DB guard: prerequisite table absent means the goose
	//    runner has not yet applied migrations 0001..0005. Skip
	//    cleanly and let the next boot handle us.
	var prereqExists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'archetype_configurations'
		)`,
	).Scan(&prereqExists); err != nil {
		return fmt.Errorf("probe archetype_configurations: %w", err)
	}
	if !prereqExists {
		return nil
	}

	// 2. Canonical idempotency probe: archetype_slug exists iff 0006
	//    has been applied. This is independent of any bookkeeping
	//    table the goose runner creates.
	var alreadyApplied bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'archetype_configurations'
			  AND column_name = 'archetype_slug'
		)`,
	).Scan(&alreadyApplied); err != nil {
		return fmt.Errorf("probe archetype_slug column: %w", err)
	}
	if alreadyApplied {
		return nil
	}

	// 3. Apply the SQL body. The body wraps every statement in
	//    BEGIN; ... COMMIT; so a single ExecContext is sufficient
	//    and the transaction boundary is owned by the SQL itself.
	if _, err := db.ExecContext(ctx, sqlBody); err != nil {
		return fmt.Errorf("apply migration 0006: %w", err)
	}

	// 4. Best-effort self-record. Only insert if the migrations
	//    table exists; on a fresh database the goose runner may
	//    not have created it yet (depending on call ordering vs the
	//    wrapper). Column-existence above is the canonical signal,
	//    so a missing row here is harmless: the next boot's column
	//    probe will skip the wrapper regardless.
	var migrationsTableExists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'archetype_schema_migrations'
		)`,
	).Scan(&migrationsTableExists); err != nil {
		// Probe failure here is non-fatal: we have already applied
		// the schema change. Surface as a warning-free no-op so the
		// caller does not block on best-effort bookkeeping.
		return nil
	}
	if !migrationsTableExists {
		return nil
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO archetype_schema_migrations (version_id, is_applied, tstamp)
		 VALUES ('6', true, now())
		 ON CONFLICT DO NOTHING`,
	); err != nil {
		// Same policy: schema change already applied; the row is
		// bookkeeping. A failed insert must not crash the chat
		// binary. The column-existence probe keeps the wrapper
		// idempotent on the next boot regardless.
		return nil
	}
	return nil
}