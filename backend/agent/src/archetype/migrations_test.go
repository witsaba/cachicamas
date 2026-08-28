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

// Test_Migration_0006_ReKeyChatToAssistantPreservesRows (T-06 PR-1).
// Seeds legacy (archetype_kind='chat', org_id=…) rows plus migration
// 0003's __default__ seed; runs the migrations; asserts every row
// re-keyed to (archetype_slug='assistant', org_id) with identical
// system_prompt / tool_allowlist / defer_tool_names content.
// Spec: archetype-system-storage ST-03.
func Test_Migration_0006_ReKeyChatToAssistantPreservesRows(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Seed the parent + child so the FK in 0006 has a target.
	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'active', 'seed')`); err != nil {
		t.Fatalf("seed parent: %v", err)
	}

	// Seed legacy rows on the OLD PK shape.
	if _, err := db.Exec(`INSERT INTO archetype_configurations
		(archetype_kind, org_id, system_prompt, tool_allowlist, defer_tool_names, version, updated_at, updated_by)
		VALUES
			('chat', 'org-1', 'prompt-A', '["current_time"]'::jsonb, '[]'::jsonb, 1, now(), 'seed'),
			('chat', '__default__', 'prompt-default', '["current_time","summarize_conversation"]'::jsonb, '["summarize_conversation"]'::jsonb, 1, now(), 'seed')`); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}

	// Apply the wrapper directly (bypasses the goose allowlist).
	if err := archetypeMigrations.Run0006IfNeeded(context.Background(), db); err != nil {
		t.Fatalf("Run0006IfNeeded: %v", err)
	}

	// Read back via the new PK shape.
	rows, err := db.Query(`SELECT archetype_slug, org_id, system_prompt FROM archetype_configurations ORDER BY org_id`)
	if err != nil {
		t.Fatalf("query: %v", err)
	}
	defer func() { _ = rows.Close() }()
	type row struct{ slug, orgID, prompt string }
	var got []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.slug, &r.orgID, &r.prompt); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got = append(got, r)
	}
	if len(got) != 2 {
		t.Fatalf("got %d rows after 0006, want 2", len(got))
	}
	for _, r := range got {
		if r.slug != "assistant" {
			t.Errorf("archetype_slug = %q, want assistant (re-key 1:1)", r.slug)
		}
	}
	// prompt content preserved verbatim
	wantPrompts := map[string]string{
		"org-1":       "prompt-A",
		"__default__": "prompt-default",
	}
	for _, r := range got {
		if want := wantPrompts[r.orgID]; r.prompt != want {
			t.Errorf("org=%q prompt = %q, want %q", r.orgID, r.prompt, want)
		}
	}

	// The __backup table exists with the pre-migration row count.
	var backupCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM archetype_configurations__backup`).Scan(&backupCount); err != nil {
		t.Fatalf("query backup: %v", err)
	}
	if backupCount != 2 {
		t.Errorf("backup table row count = %d, want 2", backupCount)
	}
}

// Test_Migration_0006_SeedsAssistantBeforeConfigFK reproduces a database with
// migrations 0001 through 0005 applied: the parent and child tables exist but
// contain no Assistant seed, while a legacy chat configuration still exists.
// Migration 0006 must seed the FK target before re-keying the configuration.
func Test_Migration_0006_SeedsAssistantBeforeConfigFK(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO archetype_configurations
		(archetype_kind, org_id, system_prompt, tool_allowlist, defer_tool_names, version, updated_at, updated_by)
		VALUES ('chat', '__default__', 'legacy default prompt', '["current_time"]'::jsonb, '[]'::jsonb, 7, now(), 'seed')`); err != nil {
		t.Fatalf("seed legacy configuration: %v", err)
	}

	if err := archetypeMigrations.Run0006IfNeeded(context.Background(), db); err != nil {
		t.Fatalf("Run0006IfNeeded: %v", err)
	}

	var parentType string
	if err := db.QueryRow(`SELECT type FROM archetypes WHERE slug = 'assistant'`).Scan(&parentType); err != nil {
		t.Fatalf("query Assistant parent: %v", err)
	}
	if parentType != "system" {
		t.Errorf("Assistant parent type = %q, want system", parentType)
	}

	var bundleVersion string
	var isCritical bool
	if err := db.QueryRow(`SELECT bundle_version, is_critical FROM system_archetypes WHERE slug = 'assistant'`).Scan(&bundleVersion, &isCritical); err != nil {
		t.Fatalf("query Assistant child: %v", err)
	}
	if bundleVersion != "v1" || !isCritical {
		t.Errorf("Assistant child = (%q, %t), want (v1, true)", bundleVersion, isCritical)
	}

	var slug, prompt string
	var version int
	if err := db.QueryRow(`SELECT archetype_slug, system_prompt, version
		FROM archetype_configurations WHERE org_id = '__default__'`).Scan(&slug, &prompt, &version); err != nil {
		t.Fatalf("query reshaped configuration: %v", err)
	}
	if slug != "assistant" || prompt != "legacy default prompt" || version != 7 {
		t.Errorf("reshaped configuration = (%q, %q, %d), want (assistant, legacy default prompt, 7)", slug, prompt, version)
	}

	var backupCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM archetype_configurations__backup`).Scan(&backupCount); err != nil {
		t.Fatalf("query backup: %v", err)
	}
	if backupCount != 1 {
		t.Errorf("backup row count = %d, want 1", backupCount)
	}
}

// Test_Migration_0006_RerunIsNoOp (T-06 PR-1). Calling the wrapper
// twice is a no-op — the second invocation sees the recorded row in
// archetype_schema_migrations and returns nil without re-applying.
// Spec: archetype-system-storage ST-05.
func Test_Migration_0006_RerunIsNoOp(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Seed parent + one legacy row so 0006 can apply.
	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'active', 'seed')`); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO archetype_configurations
		(archetype_kind, org_id, system_prompt, tool_allowlist, defer_tool_names, version, updated_at, updated_by)
		VALUES ('chat', 'org-1', 'p', '["a"]'::jsonb, '[]'::jsonb, 1, now(), 'seed')`); err != nil {
		t.Fatalf("seed legacy: %v", err)
	}

	if err := archetypeMigrations.Run0006IfNeeded(context.Background(), db); err != nil {
		t.Fatalf("first call: %v", err)
	}
	// Second call: must not error (idempotent).
	if err := archetypeMigrations.Run0006IfNeeded(context.Background(), db); err != nil {
		t.Fatalf("second call (idempotent): %v", err)
	}
	// archetype_slug column still NOT NULL on every row.
	var nulls int
	if err := db.QueryRow(`SELECT COUNT(*) FROM archetype_configurations WHERE archetype_slug IS NULL`).Scan(&nulls); err != nil {
		t.Fatalf("count nulls: %v", err)
	}
	if nulls != 0 {
		t.Errorf("found %d rows with NULL archetype_slug after re-run", nulls)
	}
}

// Test_Migration_0007_AddsLogFKAndIndex (T-08 PR-1). Asserts the
// log table has the archetype_slug NOT NULL column, the FK
// constraint, and the new index. Spec: archetype-system-storage ST-04.
func Test_Migration_0007_AddsLogFKAndIndex(t *testing.T) {
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

	// Column NOT NULL.
	var isNullable string
	if err := db.QueryRow(`
		SELECT is_nullable FROM information_schema.columns
		WHERE table_name = 'archetype_configurations_log' AND column_name = 'archetype_slug'`,
	).Scan(&isNullable); err != nil {
		t.Fatalf("query is_nullable: %v", err)
	}
	if isNullable != "NO" {
		t.Errorf("archetype_configurations_log.archetype_slug is_nullable = %q, want NO", isNullable)
	}

	// FK exists.
	var fkCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.table_constraints
		WHERE table_name = 'archetype_configurations_log'
		  AND constraint_type = 'FOREIGN KEY'
		  AND constraint_name = 'archetype_configurations_log_slug_fkey'`,
	).Scan(&fkCount); err != nil {
		t.Fatalf("query FK: %v", err)
	}
	if fkCount != 1 {
		t.Errorf("FK count = %d, want 1", fkCount)
	}

	// Index exists.
	var idxCount int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM pg_indexes
		WHERE schemaname = 'public'
		  AND tablename = 'archetype_configurations_log'
		  AND indexname = 'idx_archetype_configurations_log_slug_org_created'`,
	).Scan(&idxCount); err != nil {
		t.Fatalf("query index: %v", err)
	}
	if idxCount != 1 {
		t.Errorf("index count = %d, want 1", idxCount)
	}
}

// Test_Migration_0007_BackfillsAssistantForPriorRows (T-08 PR-1).
// Asserts every prior log row has archetype_slug='assistant' after
// 0007's DEFAULT + UPDATE backfill. Spec: archetype-system-storage ST-04.
func Test_Migration_0007_BackfillsAssistantForPriorRows(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	resetArchetypeTables(t)

	db, err := sql.Open("pgx", archetypeTestDSN())
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Apply 0001..0003 manually to set up legacy state.
	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'active', 'seed')`); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if err := archetypeMigrations.Run0006IfNeeded(context.Background(), db); err != nil {
		t.Fatalf("Run0006IfNeeded: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO archetype_configurations_log
		(archetype_kind, org_id, actor, after, created_at)
		VALUES
			('chat', 'org-1', 'alice', '{}'::jsonb, now() - interval '5 minutes'),
			('chat', '__default__', 'seed', '{}'::jsonb, now() - interval '1 hour')`); err != nil {
		t.Fatalf("seed log rows: %v", err)
	}

	// Run 0007 via the goose runner.
	provider, err := migrator.NewProvider(context.Background(), db, archetypeMigrations.MigrationsFS, "archetype_schema_migrations")
	if err != nil {
		t.Fatalf("migrator.NewProvider: %v", err)
	}
	if _, err := provider.Up(context.Background()); err != nil {
		t.Fatalf("provider.Up: %v", err)
	}

	// Every prior row has archetype_slug='assistant'.
	var notAssistant int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM archetype_configurations_log
		WHERE archetype_slug <> 'assistant'`,
	).Scan(&notAssistant); err != nil {
		t.Fatalf("count: %v", err)
	}
	if notAssistant != 0 {
		t.Errorf("found %d log rows not backfilled to 'assistant'", notAssistant)
	}
}
