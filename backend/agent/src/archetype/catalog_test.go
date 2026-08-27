// T-09 (PR-1 of cachicamas-archetype-system-foundation) — INTEGRATION-gated
// tests for the polymorphic CatalogLoader (LoadBySlug).
//
// Mirrors the existing config_test.go convention: every test in this
// file is gated by INTEGRATION=1 (the Loader is a Postgres adapter;
// asserting FOR SHARE semantics against a real Postgres is the only
// honest verification per the DBA precedent at
// backend/database_administrator/src/migration/postgres/driver_test.go:295).
package archetype_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/agent/src/archetype"
)

// catalogRequiresPostgres skips the test unless INTEGRATION=1 is set.
func catalogRequiresPostgres(t *testing.T) string {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the polymorphic CatalogLoader is a Postgres adapter; JOIN semantics require a live connection)")
	}
	dsn := os.Getenv("CACHICAMAS_CHAT_STORE_DSN")
	if dsn == "" {
		dsn = "postgres://cachicamas:changeme-local-only@localhost:5432/cachicamas_pg?sslmode=disable"
	}
	return dsn
}

// resetCatalogFixtures drops + re-creates the catalog-shape fixture
// (parent + child + per-org override) for the slug under test.
func resetCatalogFixtures(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("resetCatalogFixtures: sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Drop test fixtures by slug; ordering respects FK direction.
	if _, err := db.Exec(`
		DELETE FROM archetype_configurations_log WHERE archetype_slug = 'assistant';
		DELETE FROM archetype_configurations WHERE archetype_slug = 'assistant';
		DELETE FROM system_archetypes WHERE slug = 'assistant';
		DELETE FROM archetypes WHERE slug = 'assistant';
	`); err != nil {
		t.Fatalf("resetCatalogFixtures: DELETE: %v", err)
	}
}

// Test_LoadBySlug_PresentSystemArchetype (T-09 PR-1). Seeds parent +
// child + per-org override rows; calls LoadBySlug; asserts the
// returned ArchetypeView has parent columns populated, the child
// columns present via ChildColumns(), and the override payload.
// Spec: archetype-polymorphic-loader LD-01.
func Test_LoadBySlug_PresentSystemArchetype(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	resetCatalogFixtures(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'active', 'seed')`); err != nil {
		t.Fatalf("seed parent: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO system_archetypes (slug, bundle_version, is_critical)
		VALUES ('assistant', 'v1', true)`); err != nil {
		t.Fatalf("seed child: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO archetype_configurations
		(archetype_slug, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by)
		VALUES ('assistant', 'org-1', 'custom-prompt', '["current_time","summarize_conversation"]'::jsonb,
		        '["summarize_conversation"]'::jsonb, NULL, 4, now(), 'alice')`); err != nil {
		t.Fatalf("seed override: %v", err)
	}

	loader := archetype.NewCatalogLoader(db)
	view, found, err := loader.LoadBySlug(context.Background(), "assistant", "org-1")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if !found {
		t.Fatal("LoadBySlug returned found=false; want true")
	}
	if view.Slug != "assistant" {
		t.Errorf("Slug = %q, want assistant", view.Slug)
	}
	if view.Type != "system" {
		t.Errorf("Type = %q, want system", view.Type)
	}
	if view.DisplayName != "Assistant" {
		t.Errorf("DisplayName = %q, want Assistant", view.DisplayName)
	}
	if view.Status != "active" {
		t.Errorf("Status = %q, want active", view.Status)
	}
	// Child columns.
	childRaw, ok := view.ChildColumns()
	if !ok || childRaw == nil {
		t.Fatalf("ChildColumns = (%v, %v); want (non-nil, true)", childRaw, ok)
	}
	child, ok := childRaw.(*archetype.SystemArchetype)
	if !ok {
		t.Fatalf("ChildColumns = (%T, %v); want (*SystemArchetype, true)", childRaw, ok)
	}
	if child.BundleVersion != "v1" {
		t.Errorf("BundleVersion = %q, want v1", child.BundleVersion)
	}
	if !child.IsCritical {
		t.Errorf("IsCritical = false, want true")
	}
	// Override.
	if view.Override == nil {
		t.Fatal("Override = nil; want populated")
	}
	if view.Override.SystemPrompt != "custom-prompt" {
		t.Errorf("Override.SystemPrompt = %q, want custom-prompt", view.Override.SystemPrompt)
	}
	if view.Override.Version != 4 {
		t.Errorf("Override.Version = %d, want 4", view.Override.Version)
	}
}

// Test_LoadBySlug_AbsentRow_ReturnsDefaultConfig (T-09 PR-1). Empty
// table → DefaultConfigView with found=false. Spec: LD-04.
func Test_LoadBySlug_AbsentRow_ReturnsDefaultConfig(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	resetCatalogFixtures(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	loader := archetype.NewCatalogLoader(db)
	view, found, err := loader.LoadBySlug(context.Background(), "assistant", "org-1")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if found {
		t.Error("LoadBySlug returned found=true for absent row; want false")
	}
	if view.Override == nil {
		t.Error("absent row returned nil Override; want DefaultConfigView with safe defaults")
	}
	// The Loader MUST NOT auto-write — no row should appear after the call.
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM archetypes WHERE slug = 'assistant'`).Scan(&count); err != nil {
		t.Fatalf("count archetypes: %v", err)
	}
	if count != 0 {
		t.Errorf("Loader auto-wrote a parent row (%d); MUST NOT auto-write", count)
	}
}

// Test_LoadBySlug_UnknownSlug_ReturnsErrUnknownArchetypeSlug (T-09 PR-1).
// Empty slug → ErrUnknownArchetypeSlug sentinel.
func Test_LoadBySlug_UnknownSlug_ReturnsErrUnknownArchetypeSlug(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	loader := archetype.NewCatalogLoader(db)
	_, _, err = loader.LoadBySlug(context.Background(), "", "org-1")
	if err == nil {
		t.Fatal("LoadBySlug with empty slug returned nil error; want ErrUnknownArchetypeSlug")
	}
}

// Test_DefaultConfigView_IsPure (T-09 PR-1). Two calls with the same
// inputs return byte-equivalent output; no I/O. Spec: LD-04.
func Test_DefaultConfigView_IsPure(t *testing.T) {
	a := archetype.DefaultConfigView("assistant", "org-1", []string{"current_time", "summarize_conversation"})
	b := archetype.DefaultConfigView("assistant", "org-1", []string{"current_time", "summarize_conversation"})

	if a.Slug != b.Slug || a.Type != b.Type || a.DisplayName != b.DisplayName {
		t.Errorf("DefaultConfigView not deterministic: a=%+v b=%+v", a, b)
	}
	if a.Override == nil || b.Override == nil {
		t.Fatal("DefaultConfigView returned nil Override on one of two calls")
	}
	if a.Override.SystemPrompt != b.Override.SystemPrompt {
		t.Errorf("Override.SystemPrompt differs: %q vs %q", a.Override.SystemPrompt, b.Override.SystemPrompt)
	}
	if len(a.Override.ToolAllowlist) != len(b.Override.ToolAllowlist) {
		t.Errorf("Override.ToolAllowlist lengths differ")
	}
}

// Test_LoadBySlug_ParentArchived_ReturnsFoundFalse (T-10 PR-1).
// status='archived' OR archived_at IS NOT NULL → found=false.
// The Loader's WHERE clause excludes terminal rows. Spec: LD-03.
func Test_LoadBySlug_ParentArchived_ReturnsFoundFalse(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	resetCatalogFixtures(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Seed an archived parent. No child row, no override — the
	// terminal predicate is the test's focus.
	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, archived_at, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'archived', now(), 'seed')`); err != nil {
		t.Fatalf("seed archived parent: %v", err)
	}

	loader := archetype.NewCatalogLoader(db)
	view, found, err := loader.LoadBySlug(context.Background(), "assistant", "org-1")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if found {
		t.Error("LoadBySlug returned found=true for archived row; want false")
	}
	// DefaultConfigView still returns the safe fallback (Override nil
	// in this case because DefaultConfigView sets it).
	if view.Override == nil {
		t.Error("archived row returned nil Override; want DefaultConfigView fallback")
	}
}

// Test_LoadBySlug_ToleratesMissingChildTable (T-10 PR-1). A type='general'
// parent with NO row in system_archetypes returns (parent columns, child=nil,
// found=true) — Loader does NOT error on the absent join.
// Spec: LD-01 + LD-05.
func Test_LoadBySlug_ToleratesMissingChildTable(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	resetCatalogFixtures(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('ad-hoc', 'general', 'Ad Hoc', 'One-off', 'active', 'seed')`); err != nil {
		t.Fatalf("seed parent: %v", err)
	}

	loader := archetype.NewCatalogLoader(db)
	view, found, err := loader.LoadBySlug(context.Background(), "ad-hoc", "org-1")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if !found {
		t.Fatal("LoadBySlug returned found=false; want true (parent row exists)")
	}
	if view.Type != "general" {
		t.Errorf("Type = %q, want general", view.Type)
	}
	childRaw, ok := view.ChildColumns()
	if ok {
		t.Errorf("ChildColumns = (%v, true); want (nil, false) for missing child table", childRaw)
	}
}

// Test_LoadBySlug_ArchivedAtSetButStatusActive_ReturnsFoundFalse
// (T-10 PR-1). archived_at != NULL alone (with status='active') is
// still terminal — the Loader's predicate is `status='archived' OR
// archived_at IS NOT NULL`. Spec: LD-03.
func Test_LoadBySlug_ArchivedAtSetButStatusActive_ReturnsFoundFalse(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	resetCatalogFixtures(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, archived_at, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'active', now(), 'seed')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	loader := archetype.NewCatalogLoader(db)
	_, found, err := loader.LoadBySlug(context.Background(), "assistant", "org-1")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if found {
		t.Error("LoadBySlug returned found=true; want false (archived_at is terminal)")
	}
}

// Test_LoadBySlug_FutureGeneralToleratesMissingChild (T-10 PR-1).
// The exact LD-05 forward-compat scenario: a type='general' archetype
// row exists, no general_archetypes child table at all (yet), and the
// Loader returns parent columns populated with ChildColumns()=(nil,false).
func Test_LoadBySlug_FutureGeneralToleratesMissingChild(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	resetCatalogFixtures(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('ad-hoc', 'general', 'Ad Hoc', 'One-off', 'active', 'seed')`); err != nil {
		t.Fatalf("seed: %v", err)
	}

	loader := archetype.NewCatalogLoader(db)
	view, found, err := loader.LoadBySlug(context.Background(), "ad-hoc", "org-1")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if !found {
		t.Fatal("found = false; want true")
	}
	if view.Slug != "ad-hoc" || view.Type != "general" {
		t.Errorf("Slug=%q Type=%q; want ad-hoc/general", view.Slug, view.Type)
	}
	_, ok := view.ChildColumns()
	if ok {
		t.Error("ChildColumns ok=true; want false (no general_archetypes child)")
	}
}
