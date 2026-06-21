// Package migration wires the embedded SQL migrations to the goose
// runner. embed.FS is the only public surface this package exports to
// the rest of the binary: the runner (runner.go) calls
// goose.SetBaseFS(MigrationsFS) so goose reads from the embedded
// virtual filesystem instead of from disk.
//
// Per design §3 + §14 (ADR-002), the embed pattern is the
// intentional alternative to runtime-on-disk migrations: the binary
// is self-contained, the migration set is reproducible across
// environments, and there is no path for a deployment to forget a
// migration file.
package migration

import "embed"

// MigrationsFS is the embedded virtual filesystem holding every
// timestamp-prefixed `*.sql` file under sql/. The pattern `sql/*.sql`
// is deliberately flat — goose does not recurse into subdirectories
// (verified against goose v3.27.1). Nested scratch files (e.g. a
// dev-only `sql/_scratch/foo.sql`) must NOT be added; they would
// silently violate the lexicographic-ordering guarantee from
// spec S-DBMIG-003.
//
// Exported (capital M) because runner.go consumes it across files in
// the same package; in a stricter hexagonal cut it would be lower-case
// with a constructor returning an interface. KISS for v1.
//
//go:embed sql/*.sql
var MigrationsFS embed.FS
