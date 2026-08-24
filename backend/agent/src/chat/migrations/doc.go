// Package migrations holds the chat archetype's forward-only SQL
// migrations (NFR-CCS-005, NFR-CCS-006). The migrations are embedded
// at compile time via //go:embed in this package; the runner lives in
// chat/migrator.
//
// This file exists in WU-2 to keep the pgx/v5/stdlib blank import
// wired through `go mod tidy` BEFORE the chat/migrator/runner.go file
// lands in WU-3. The dep is admitted in the same PR as this ADR
// (0010-add-pgx-and-goose-to-backend-agent.md) per
// `openspec/AGENTS.md` Hard Rule 5; the production import moves to
// chat/migrator/runner.go at WU-3. Until then this file is the
// allowlist-justifying import site; it has no runtime behaviour.
package migrations

import (
	// Blank import: registers "pgx" as a database/sql driver so the
	// chat adapter can call sql.Open("pgx", dsn). Mirrors the DBA
	// precedent at
	// backend/database_administrator/src/migration/postgres/driver.go:30-33.
	//
	// The blank import's only effect is the driver's init() side
	// effect — it has no runtime cost. The dep lives here in WU-2 to
	// keep `go mod tidy` from stripping the require directive; WU-3
	// moves this import to chat/migrator/runner.go where the runner
	// uses it for real.
	_ "github.com/jackc/pgx/v5/stdlib"
)