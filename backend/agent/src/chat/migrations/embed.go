// Package migrations holds the chat archetype's forward-only SQL
// migrations (NFR-CCS-005, NFR-CCS-006). The migrations are embedded
// at compile time via //go:embed; the runner lives in chat/migrator.
//
// This package exists so the //go:embed pattern can resolve the SQL
// files as siblings to this file (Go's //go:embed forbids ".." in
// patterns, so the embed source must be co-located with the
// embed directive — same pattern as DBA's
// backend/database_administrator/src/migration/embed.go).
package migrations

import (
	"embed"

	// Blank import: registers "pgx" as a database/sql driver so
	// the chat adapter can call sql.Open("pgx", dsn). Mirrors the
	// DBA precedent at
	// backend/database_administrator/src/migration/postgres/driver.go:30-33.
	//
	// The blank import's only effect is the driver's init() side
	// effect — it has no runtime cost. The dep lives here in the
	// migrations package because the package is the canonical
	// "Postgres-backed persistence" home for the chat archetype,
	// and the runner's own package imports chat/migrations for
	// MigrationsFS, so the blank import propagates transitively.
	// ADR 0010 is the dep admission; this blank import is the
	// single first production import site, per R-AGP-003 same-commit
	// rule.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// MigrationsFS is the embedded virtual filesystem holding every
// timestamp-prefixed `*.sql` file in this directory. The pattern
// `*.sql` is deliberately flat — goose does not recurse into
// subdirectories (verified against goose v3.27.1).
//
// Exported (capital M) because chat/migrator/runner.go consumes it
// across files in a different package; mirroring DBA's
// `migration.MigrationsFS` precedent. In a stricter hexagonal cut
// it would be lower-case with a constructor returning an interface.
// KISS for v1.
//
//go:embed *.sql
var MigrationsFS embed.FS