// Package migrator — chat-owned forward-only SQL migration runner.
//
// WU-3 RED scaffold: the first failing test is
// TestNewProvider_AcceptsValidForwardOnlyMigration. It dials a real
// Postgres (gated by INTEGRATION=1; without the gate, the test
// skips) and asserts NewProvider returns a non-nil provider with
// no error when pointed at the embedded 0001_init.sql. The test
// fails RED at WU-3 if NewProvider returns nil / errors on a
// valid forward-only migration.
//
// WU-4 RED test: TestNewProvider_RejectsPlantedDefeat writes a
// scratch migration with DROP TABLE to t.TempDir() and asserts
// NewProvider returns an error naming the file. The test fails RED
// if assertForwardOnly doesn't trip. Mirrors the DBA planted-defeat
// pattern at runner_test.go.
package migrator_test

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/cachicamas/backend/agent/src/chat/migrations"
	"github.com/cachicamas/backend/agent/src/chat/migrator"
)

func TestNewProvider_AcceptsValidForwardOnlyMigration(t *testing.T) {
	t.Parallel()

	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the assertion exercises goose.NewProvider against a real DB; the unit-level contract — NewProvider is wired to the embedded fs.FS without error — is covered by TestNewProvider_ReturnsNonNilOnValidFS without INTEGRATION)")
	}

	dsn := os.Getenv("CACHICAMAS_CHAT_STORE_DSN")
	if dsn == "" {
		dsn = "postgres://cachicamas:changeme-local-only@localhost:5432/cachicamas_pg?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	provider, err := migrator.NewProvider(context.Background(), db, migrations.MigrationsFS, "chat_schema_migrations")
	if err != nil {
		t.Fatalf("NewProvider: %v — the runner must accept the embedded 0001_init.sql without error", err)
	}
	if provider == nil {
		t.Fatal("NewProvider returned nil provider without error")
	}
}

// TestNewProvider_ReturnsNonNilOnValidFS is the unit-level contract
// for WU-3 RED: the runner accepts the embedded forward-only
// migration without error and returns a non-nil provider. The
// runtime-provider assertion (Up against a live DB) is gated by
// INTEGRATION=1 above; this test exercises the file-walking
// forward-only check + the goose.NewProvider construction path
// without needing a DB.
//
// Why both tests: RED requires writing the test FIRST, so the WU-3
// skeleton is asserted even before any migration actually runs.
func TestNewProvider_ReturnsNonNilOnValidFS(t *testing.T) {
	t.Parallel()

	// We don't need a real DB to assert NewProvider's construction
	// contract; sql.OpenDB(nil) returns a *sql.DB that never dials.
	db := sql.OpenDB(nil)
	t.Cleanup(func() { _ = db.Close() })

	provider, err := migrator.NewProvider(context.Background(), db, migrations.MigrationsFS, "chat_schema_migrations")
	if err != nil {
		t.Fatalf("NewProvider on valid embedded FS: %v", err)
	}
	if provider == nil {
		t.Fatal("NewProvider returned nil without error")
	}
}

// TestNewProvider_RejectsNilFS asserts the precondition check
// fires before the forward-only walk (defensive — passing nil fs
// is a programmer error and the runner must refuse rather than
// panic on fs.ReadDir(nil)).
func TestNewProvider_RejectsNilFS(t *testing.T) {
	t.Parallel()

	db := sql.OpenDB(nil)
	t.Cleanup(func() { _ = db.Close() })

	_, err := migrator.NewProvider(context.Background(), db, nil, "chat_schema_migrations")
	if err == nil {
		t.Fatal("NewProvider with nil fsys returned nil error, want an explicit refusal")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("error %q must name the nil fsys so the operator sees what failed", err.Error())
	}
}

// TestNewProvider_RejectsPlantedDefeat is WU-4 RED: a scratch
// migration with a destructive DROP TABLE line MUST cause
// NewProvider to return an error at construction time. Plant the
// defeat in t.TempDir() (per the design's CH-07.2 bite proof
// pattern) and assert the failure message names the file.
func TestNewProvider_RejectsPlantedDefeat(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"0001_init.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE chat_conversations (participant_id text PRIMARY KEY);\nDROP TABLE chat_conversations;\n"),
		},
	}

	db := sql.OpenDB(nil)
	t.Cleanup(func() { _ = db.Close() })

	_, err := migrator.NewProvider(context.Background(), db, fsys, "chat_schema_migrations")
	if err == nil {
		t.Fatal("NewProvider accepted a planted-defeat migration with DROP TABLE; want a forward-only check failure")
	}
	// The error must name the file so the operator's diagnosis
	// points at the right place (per design D-D, the bite proof).
	if !strings.Contains(err.Error(), "0001_init.sql") {
		t.Errorf("error %q must name the file (per design D-D bite-proof contract)", err.Error())
	}
	if !strings.Contains(err.Error(), "line ") {
		t.Errorf("error %q must name the offending line number", err.Error())
	}
}

// TestAssertForwardOnly_BlankAndCommentLinesPass is the boundary
// check: blank lines and `--` SQL line comments MUST NOT trip the
// allowlist (the runner's forward-only check is per-line, not
// per-statement; comments and blank lines are noise).
func TestAssertForwardOnly_BlankAndCommentLinesPass(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"0001_init.sql": &fstest.MapFile{
			Data: []byte("-- a header comment\n\nCREATE TABLE chat_conversations (participant_id text PRIMARY KEY);\n\n-- trailing comment\n"),
		},
	}

	db := sql.OpenDB(nil)
	t.Cleanup(func() { _ = db.Close() })

	if _, err := migrator.NewProvider(context.Background(), db, fsys, "chat_schema_migrations"); err != nil {
		t.Fatalf("NewProvider with blank lines + SQL comments: %v — comments and blank lines must not trip the forward-only check", err)
	}
}

// TestNewProvider_RejectsPlantedDefeatFromTempDir mirrors the
// design's `t.TempDir()` planted-defeat pattern: write a real .sql
// file to a temp directory and use os.DirFS to expose it. The
// failure message must name the file (per the Gherkin bite proof
// at 0005:879-883).
func TestNewProvider_RejectsPlantedDefeatFromTempDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	defeat := filepath.Join(dir, "0001_defeat.sql")
	if err := os.WriteFile(defeat, []byte("DROP TABLE chat_conversations;\n"), 0o644); err != nil {
		t.Fatalf("seed defeat file: %v", err)
	}

	db := sql.OpenDB(nil)
	t.Cleanup(func() { _ = db.Close() })

	_, err := migrator.NewProvider(context.Background(), db, os.DirFS(dir), "chat_schema_migrations")
	if err == nil {
		t.Fatal("NewProvider accepted DROP TABLE from t.TempDir(); want a forward-only check failure")
	}
	if !strings.Contains(err.Error(), filepath.Base(defeat)) {
		t.Errorf("error %q must name the file (Gherkin bite proof contract 0005:879-883)", err.Error())
	}
}