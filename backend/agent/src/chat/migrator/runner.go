// Package migrator — chat-package-local goose runner.
//
// NewProvider builds a goose.Provider over the chat-owned
// forward-only SQL migrations (chat/migrations/*.sql, embedded via
// //go:embed in embed.go). The provider is the value-object the
// composition root holds; its Up(ctx) call is what actually touches
// the database. Constructing a fresh Provider per call is cheap
// (no dial) and lets multiple goroutines share the runner.
//
// Design references:
//   - design D-D (forward-only allowlist, no lock)
//   - design D-H (chat-package-local, no DBA reuse)
//   - DBA precedent: backend/database_administrator/src/migration/runner.go
package migrator

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/pressly/goose/v3"

	// Blank import: registers "pgx" as a database/sql driver so
	// the chat adapter can call sql.Open("pgx", dsn). The blank
	// import's only effect is the driver's init() side effect; the
	// real consumer is chat/store_postgres.go (added at WU-6).
	// ADR 0010 is the dep admission; this file is the first runner
	// site to import goose, per R-AGP-003 same-commit rule.
	_ "github.com/jackc/pgx/v5/stdlib"
)

// NewProvider builds a goose.Provider over the chat-owned forward-only
// SQL migrations. The provider is constructed against db but does
// NOT connect — provider.Up(ctx) is what actually dials. The
// returned provider is safe to share across goroutines.
//
// CH-07 WU-3 skeleton: this commit (1) registers the runner, (2)
// asserts the forward-only allowlist at construction time, and (3)
// returns a non-nil provider. The allowlist check is the
// safety guarantee — a migration with any line outside the set is
// rejected BEFORE provider.Up is ever called.
//
// tableName defaults to "chat_schema_migrations" when empty. DBA's
// runner uses "schema_migrations"; chat uses its own to keep the
// per-archetype bookkeeping tables visually distinct (per ADR 0009
// § D6 ownership).
func NewProvider(ctx context.Context, db *sql.DB, fsys fs.FS, tableName string) (*goose.Provider, error) {
	_ = ctx // unused at construction time; provider.Up carries the dial
	if tableName == "" {
		tableName = "chat_schema_migrations"
	}
	if fsys == nil {
		return nil, fmt.Errorf("migrator.NewProvider: fsys is nil")
	}
	if err := assertForwardOnly(fsys); err != nil {
		return nil, fmt.Errorf("migrator.NewProvider: forward-only check failed: %w", err)
	}
	return goose.NewProvider(
		goose.DialectPostgres,
		db,
		fsys,
		goose.WithTableName(tableName),
	)
}

// allowedMigrationStmts is the regex set the forward-only check
// matches every line of every migration against. A migration with
// ANY line outside the set is rejected at provider construction
// time — not at apply time. NFR-CCS-006.
//
// Every regex is case-insensitive (?i) except the blank-line and
// comment-line patterns (which are pure whitespace / punctuation
// and have no SQL semantics to normalize).
var allowedMigrationStmts = []*regexp.Regexp{
	regexp.MustCompile(`(?i)^\s*CREATE\s+TABLE\s+`),
	regexp.MustCompile(`(?i)^\s*CREATE\s+INDEX\s+`),
	regexp.MustCompile(`(?i)^\s*CREATE\s+SEQUENCE\s+`),
	regexp.MustCompile(`(?i)^\s*CREATE\s+UNIQUE\s+INDEX\s+`),
	regexp.MustCompile(`(?i)^\s*COMMENT\s+ON\s+`),
	regexp.MustCompile(`(?i)^\s*INSERT\s+INTO\s+`),
	regexp.MustCompile(`^\s*$`),  // blank lines
	regexp.MustCompile(`^\s*--`), // SQL line comments
}

// assertForwardOnly walks every .sql file in fsys; refuses if any
// line outside the allowedMigrationStmts set is NOT inside a
// multi-line CREATE/INSERT/COMMENT statement. The error names the
// file and line number for the operator's diagnosis.
//
// The check is at construction time, not apply time: a destructive
// statement that ALMOST runs before the check trips is worse than
// refusing to boot. Planted-defeat coverage in runner_test.go.
//
// Multi-line awareness: a `CREATE TABLE foo (\n  a text,\n  b text\n);`
// spans multiple lines. The check tracks whether the current line
// is inside a CREATE/INSERT/COMMENT block (started by a header
// line that matched an allowlist pattern and not yet closed by a
// trailing `;`); inner lines pass without per-line inspection.
func assertForwardOnly(fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		b, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return fmt.Errorf("read %s: %w", e.Name(), err)
		}
		var inBlock bool
		for i, raw := range strings.Split(string(b), "\n") {
			line := raw
			trimmed := strings.TrimSpace(line)
			// blank / comment lines always pass
			if trimmed == "" || strings.HasPrefix(trimmed, "--") {
				continue
			}
			if inBlock {
				// inside a multi-line CREATE/INSERT/COMMENT block;
				// pass until we see the closing `;`.
				if strings.Contains(line, ";") {
					inBlock = false
				}
				continue
			}
			if !matchesAny(line, allowedMigrationStmts) {
				return fmt.Errorf("migration %s line %d refuses forward-only check: %q", e.Name(), i+1, line)
			}
			// A header line that ends without `;` opens a multi-line block.
			if !strings.Contains(line, ";") {
				inBlock = true
			}
		}
	}
	return nil
}

// matchesAny returns true if line matches any of the patterns. A
// nil pattern (defensive) is treated as "matches nothing" — the
// allowlist's invariants are non-nil on construction.
func matchesAny(line string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p == nil {
			continue
		}
		if p.MatchString(line) {
			return true
		}
	}
	return false
}