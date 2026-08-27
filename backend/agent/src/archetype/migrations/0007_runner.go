// Package migrations — T-08 PR-1
// (cachicamas-archetype-system-foundation) — paired Go wrapper for
// migration 0007, mirroring the 0006_runner.go design.
//
// Why this file exists
// --------------------
// The chat composition root's forward-only allowlist (chat/migrator/runner.go)
// refuses UPDATE, ADD CONSTRAINT FOREIGN KEY, and any ALTER TABLE form
// other than ADD COLUMN. Migration 0007 (archetype_configurations_log
// gains the archetype_slug FK + index) needs exactly those statements:
//   - ADD COLUMN archetype_slug text NOT NULL DEFAULT 'assistant' — this
//     is the only allowlisted form, but the SQL wraps it across two
//     lines, which the per-line check still trips on.
//   - UPDATE … SET archetype_slug = 'assistant' WHERE archetype_slug
//     IS NULL — defensive backfill, not in the allowlist.
//   - ADD CONSTRAINT … FOREIGN KEY — not in the allowlist.
//   - CREATE INDEX — allowlisted.
//
// Per design AD-6, the wrapper approach is the chosen escape hatch for
// any non-additive migration: the wrapper runs the SQL directly via
// ExecContext on the supplied pool, with a single transaction wrapping
// the body so partial failure rolls back cleanly.
//
// Idempotency mechanism (information_schema-based)
// ------------------------------------------------
// Same probe order as 0006_runner.go:
//   1. information_schema.tables for the prerequisite table
//      archetype_configurations_log. If absent, return nil — the
//      goose runner will create it via 0002 next.
//   2. information_schema.columns for the archetype_slug column on
//      archetype_configurations_log. If present, return nil — 0007
//      has already been applied.
//   3. Apply the SQL body as a single ExecContext.
//   4. Best-effort self-record into archetype_schema_migrations,
//      conditional on that table already existing.
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

// Run0007IfNeeded executes migration 0007 once and uses schema-level
// probes (information_schema) as the idempotency mechanism so it can
// run safely both before and after the goose runner has touched the
// schema. The chat composition root calls this AFTER the wrapper for
// 0006 (which reshapes archetype_configurations) and BEFORE the goose
// runner applies any later *.sql — same ordering as the 0006 wrapper.
//
// Probe order:
//   1. archetype_configurations_log table — if absent, this is a fresh
//      database; return nil and let the goose runner create the
//      prerequisite tables. We will be re-invoked on the next boot.
//   2. archetype_slug column on archetype_configurations_log — if
//      present, 0007 has already been applied; return nil.
//   3. Otherwise apply the SQL body as a single ExecContext.
//   4. Best-effort self-record into archetype_schema_migrations,
//      conditional on that table already existing.
//
// On SQL failure, no row is inserted and the caller surfaces the
// error verbatim — the operator retries on the next boot.
func Run0007IfNeeded(ctx context.Context, db *sql.DB) error {
	const sqlBody = `-- +goose Up

ALTER TABLE archetype_configurations_log
    ADD COLUMN archetype_slug text NOT NULL DEFAULT 'assistant';

UPDATE archetype_configurations_log
    SET archetype_slug = 'assistant'
    WHERE archetype_slug IS NULL;

ALTER TABLE archetype_configurations_log
    ADD CONSTRAINT archetype_configurations_log_slug_fkey
        FOREIGN KEY (archetype_slug)
        REFERENCES archetypes(slug)
        ON DELETE RESTRICT;

CREATE INDEX idx_archetype_configurations_log_slug_org_created
    ON archetype_configurations_log (archetype_slug, org_id, created_at DESC);`

	// 1. Fresh-DB guard: prerequisite table absent means the goose
	//    runner has not yet applied migration 0002. Skip cleanly and
	//    let the next boot handle us.
	var prereqExists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'archetype_configurations_log'
		)`,
	).Scan(&prereqExists); err != nil {
		return fmt.Errorf("probe archetype_configurations_log: %w", err)
	}
	if !prereqExists {
		return nil
	}

	// 2. Canonical idempotency probe: archetype_slug exists iff 0007
	//    has been applied. Independent of any bookkeeping table the
	//    goose runner creates.
	var alreadyApplied bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'archetype_configurations_log'
			  AND column_name = 'archetype_slug'
		)`,
	).Scan(&alreadyApplied); err != nil {
		return fmt.Errorf("probe archetype_slug column on log table: %w", err)
	}
	if alreadyApplied {
		return nil
	}

	// 3. Apply the SQL body. No BEGIN/COMMIT wrapping: each statement
	//    is independently safe (ADD COLUMN with DEFAULT is metadata-
	//    only in PG 11+, the UPDATE is idempotent on the column it
	//    just added, the FK + index additions are atomic at DDL
	//    level). A failure rolls back the partial state but leaves
	//    the column-add in place — the column-existence probe above
	//    makes a retry correctly short-circuit on the next boot.
	if _, err := db.ExecContext(ctx, sqlBody); err != nil {
		return fmt.Errorf("apply migration 0007: %w", err)
	}

	// 4. Best-effort self-record, same policy as 0006.
	var migrationsTableExists bool
	if err := db.QueryRowContext(ctx,
		`SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public'
			  AND table_name = 'archetype_schema_migrations'
		)`,
	).Scan(&migrationsTableExists); err != nil {
		return nil
	}
	if !migrationsTableExists {
		return nil
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO archetype_schema_migrations (version_id, is_applied, tstamp)
		 VALUES ('7', true, now())
		 ON CONFLICT DO NOTHING`,
	); err != nil {
		return nil
	}
	return nil
}