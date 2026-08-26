// Package migrations holds the archetype package's forward-only SQL
// migrations. Embedded at compile time via //go:embed; the runner
// lives in chat/migrator and consumes this FS by parameter.
//
// This package exists so the //go:embed pattern can resolve the SQL
// files as siblings to this file. The pattern `*.sql` is deliberately
// flat — goose does not recurse into subdirectories.
//
// Future migrations (audit log at 0002, schema widens at 0003, ...)
// land in this same directory and are picked up by the runner with no
// further wiring.
package migrations

import (
	"embed"
)

// MigrationsFS is the embedded virtual filesystem holding every
// timestamp-prefixed `*.sql` file in this directory.
//
//go:embed *.sql
var MigrationsFS embed.FS
