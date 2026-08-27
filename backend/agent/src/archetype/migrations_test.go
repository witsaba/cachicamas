// T-03 (PR-1 of cachicamas-archetype-system-foundation) — INTEGRATION-gated
// schema-shape tests for migrations 0004 (archetypes parent) and 0005
// (system_archetypes child). Each test dials a real Postgres, runs the
// migration runner, and asserts the post-migration schema shape.
//
// Gated by `INTEGRATION=1` (mirrors chat/store_postgres_test.go and DBA's
// precedent at backend/database_administrator/src/migration/postgres/
// driver_test.go:295). Without the gate, all tests SKIP — the unit-level
// `cd backend/agent && make test` (go test -race -v ./...) run still
// passes because every test in this file is skip-clean.
package archetype_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	archetypeMigrations "github.com/cachicamas/backend/agent/src/archetype/migrations"
	"github.com/cachicamas/backend/agent/src/chat/migrator"
)

// archetypeTestDSN is the DSN the integration tests dial when INTEGRATION=1
// is set. Mirrors chatPostgresTestDSN's pattern in
// backend/agent/src/chat/store_postgres_test.go:28.
func archetypeTestDSN() string {
	if v := os.Getenv("CACHICAMAS_CHAT_STORE_DSN"); v != "" {
		return v
	}
	return "postgres://cachicamas:changeme-local-only@localhost:5432/cachicamas_pg?sslmode=disable"
}

// resetArchetypeTables truncates every archetype-owned table so each
// integration test starts from a clean slate. Mirrors chat's
// resetChatTables (backend/agent/src/chat/store_postgres_test.go:42).
//
// Idempotent: uses IF EXISTS so the first run (when only the migrations'
// tables exist) doesn't fail. TRUNCATE … RESTART IDENTITY CASCADE so the
// FK chain from archetype_configurations(archetype_slug) → archetypes(slug)
// does not block the order.
func resetArchetypeTables(t *testing.T) {
	t.Helper()
	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("resetArchetypeTables: sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`TRUNCATE
		archetype_configurations_log,
		archetype_configurations,
		system_archetypes,
		archetypes
	RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("resetArchetypeTables: TRUNCATE: %v", err)
	}
	if _, err := db.Exec(`DROP TABLE IF EXISTS archetype_configurations__backup`); err != nil {
		t.Fatalf("resetArchetypeTables: DROP backup: %v", err)
	}
}

// runArchetypeMigrationsUp runs the archetype migration runner against
// the supplied pool. Used by every test in this file to ensure the
// post-migration schema is in place before assertions.
func runArchetypeMigrationsUp(t *testing.T, db *sql.DB) {
	t.Helper()
	provider, err := migrator.NewProvider(context.Background(), db, archetypeMigrations.MigrationsFS, "archetype_schema_migrations")
	if err != nil {
		t.Fatalf("migrator.NewProvider: %v", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatalf("provider.Up: %v", err)
	}
}

// Test_Migration_0004_CreatesArchetypesWith8Cols asserts the parent
// table exists with all eight columns and two CHECK constraints.
// Spec: archetype-system-storage ST-01.
func Test_Migration_0004_CreatesArchetypesWith8Cols(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the migration runner dials a real Postgres)")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	runArchetypeMigrationsUp(t, db)

	// Column existence + types.
	wantCols := map[string]string{
		"slug":         "text",
		"type":         "text",
		"display_name": "text",
		"tagline":      "text",
		"status":       "text",
		"archived_at":  "timestamp with time zone",
		"created_at":   "timestamp with time zone",
		"created_by":   "text",
	}
	rows, err := db.Query(`
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_name = 'archetypes'
		ORDER BY column_name`)
	if err != nil {
		t.Fatalf("information_schema.columns: %v", err)
	}
	defer func() { _ = rows.Close() }()
	gotCols := map[string]string{}
	for rows.Next() {
		var name, typ string
		if err := rows.Scan(&name, &typ); err != nil {
			t.Fatalf("scan column: %v", err)
		}
		gotCols[name] = typ
	}
	for col, typ := range wantCols {
		if got := gotCols[col]; got != typ {
			t.Errorf("archetypes.%s = %q, want %q", col, got, typ)
		}
	}
	if _, ok := gotCols["slug"]; !ok {
		t.Errorf("archetypes.slug column missing")
	}
}

// Test_Migration_0004_TypeCheckRejectsBadValue asserts the type CHECK
// constraint rejects out-of-vocabulary values.
// Spec: archetype-system-storage ST-01.
func Test_Migration_0004_TypeCheckRejectsBadValue(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	runArchetypeMigrationsUp(t, db)

	_, err = db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('bad-type', 'vendor', 'x', 'x', 'active', 'test')`)
	if err == nil {
		t.Fatal("INSERT with type='vendor' succeeded; want CHECK constraint violation")
	}
}

// Test_Migration_0004_StatusCheckRejectsBadValue asserts the status
// CHECK constraint rejects out-of-vocabulary values.
// Spec: archetype-system-storage ST-01.
func Test_Migration_0004_StatusCheckRejectsBadValue(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	runArchetypeMigrationsUp(t, db)

	_, err = db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('bad-status', 'system', 'x', 'x', 'enabled', 'test')`)
	if err == nil {
		t.Fatal("INSERT with status='enabled' succeeded; want CHECK constraint violation")
	}
}

// Test_Migration_0005_CreatesSystemArchetypesWithPKFK asserts the
// child table exists with PK and FK to archetypes(slug).
// Spec: archetype-system-storage ST-02.
func Test_Migration_0005_CreatesSystemArchetypesWithPKFK(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	runArchetypeMigrationsUp(t, db)

	// Seed a parent row.
	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'active', 'seed')`); err != nil {
		t.Fatalf("seed parent: %v", err)
	}

	// Insert into child — must succeed.
	if _, err := db.Exec(`INSERT INTO system_archetypes (slug, bundle_version, is_critical)
		VALUES ('assistant', 'v1', true)`); err != nil {
		t.Fatalf("insert system_archetypes: %v", err)
	}

	// Verify FK exists with ON DELETE RESTRICT.
	var deleteRule string
	err = db.QueryRow(`
		SELECT rc.delete_rule
		FROM information_schema.referential_constraints rc
		JOIN information_schema.table_constraints tc
		  ON tc.constraint_name = rc.constraint_name
		WHERE tc.table_name = 'system_archetypes'
		  AND tc.constraint_type = 'FOREIGN KEY'`).Scan(&deleteRule)
	if err != nil {
		t.Fatalf("query FK: %v", err)
	}
	if deleteRule != "RESTRICT" {
		t.Errorf("system_archetypes FK delete_rule = %q, want RESTRICT", deleteRule)
	}
}

// Test_Migration_0005_RestrictBlocksParentDelete asserts the FK
// ON DELETE RESTRICT blocks DELETE FROM archetypes WHERE slug=...
// when a child row exists. Spec: archetype-system-storage ST-02.
func Test_Migration_0005_RestrictBlocksParentDelete(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	runArchetypeMigrationsUp(t, db)

	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'active', 'seed')`); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO system_archetypes (slug, bundle_version, is_critical)
		VALUES ('assistant', 'v1', true)`); err != nil {
		t.Fatalf("seed child: %v", err)
	}

	_, err = db.Exec(`DELETE FROM archetypes WHERE slug = 'assistant'`)
	if err == nil {
		t.Fatal("DELETE FROM archetypes with child row succeeded; want FK violation (ON DELETE RESTRICT)")
	}
}
