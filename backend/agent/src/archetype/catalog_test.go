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
	"sync"
	"testing"
	"time"

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

// Test_LoadBySlug_DefaultRowSuppliesConfiguration proves the persisted
// __default__ row is the only fallback when an organization has no override.
func Test_LoadBySlug_DefaultRowSuppliesConfiguration(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	resetCatalogFixtures(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		INSERT INTO archetypes (slug, type, display_name, tagline, status, created_by)
		VALUES ('assistant', 'system', 'Assistant', 'Default', 'active', 'seed');
		INSERT INTO system_archetypes (slug, bundle_version, is_critical)
		VALUES ('assistant', 'v1', true);
		INSERT INTO archetype_configurations
			(archetype_slug, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by)
		VALUES ('assistant', '__default__', 'database-owned default prompt', '["current_time"]'::jsonb,
		        '[]'::jsonb, NULL, 7, now(), 'seed')`); err != nil {
		t.Fatalf("seed database default: %v", err)
	}

	view, found, err := archetype.NewCatalogLoader(db).LoadBySlug(context.Background(), "assistant", "org-without-override")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if !found {
		t.Fatal("LoadBySlug returned found=false, want true")
	}
	if view.Override == nil {
		t.Fatal("Override = nil; want persisted default configuration")
	}
	if view.Override.SystemPrompt != "database-owned default prompt" {
		t.Errorf("SystemPrompt = %q, want database-owned default prompt", view.Override.SystemPrompt)
	}
	if view.IsOverride {
		t.Error("IsOverride = true, want false for __default__ row")
	}
}

// Test_LoadBySlug_AbsentParentReturnsNotFound keeps an unknown archetype
// distinct from an archetype whose persisted configuration is missing.
func Test_LoadBySlug_AbsentParentReturnsNotFound(t *testing.T) {
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
	if view != (archetype.ArchetypeView{}) {
		t.Errorf("view = %+v, want zero value", view)
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
	if view != (archetype.ArchetypeView{}) {
		t.Errorf("archived row returned view %+v, want zero value", view)
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

// Test_LoadBySlug_TwoOrgs_BothOverride_SystemRowReturnedButPerOrgWins
// (T-11 PR-1). Two orgs both have per-org override rows on the
// Assistant archetype. Each org's Loader call returns its own override
// (the per-org row shadows the system row) plus the parent + child
// columns populated. Spec: edge case 1 (spec §4).
func Test_LoadBySlug_TwoOrgs_BothOverride_SystemRowReturnedButPerOrgWins(t *testing.T) {
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
		VALUES
			('assistant', 'org-1', 'org-1-prompt', '["a"]'::jsonb, '[]'::jsonb, NULL, 1, now(), 'alice'),
			('assistant', 'org-2', 'org-2-prompt', '["b"]'::jsonb, '[]'::jsonb, NULL, 1, now(), 'bob')`); err != nil {
		t.Fatalf("seed overrides: %v", err)
	}

	loader := archetype.NewCatalogLoader(db)
	for _, tc := range []struct {
		orgID, want string
	}{
		{"org-1", "org-1-prompt"},
		{"org-2", "org-2-prompt"},
	} {
		view, found, err := loader.LoadBySlug(context.Background(), "assistant", tc.orgID)
		if err != nil {
			t.Fatalf("LoadBySlug(%s): %v", tc.orgID, err)
		}
		if !found {
			t.Fatalf("LoadBySlug(%s) found=false; want true", tc.orgID)
		}
		if view.Override == nil {
			t.Fatalf("LoadBySlug(%s) Override=nil", tc.orgID)
		}
		if view.Override.SystemPrompt != tc.want {
			t.Errorf("org=%s prompt=%q; want %q", tc.orgID, view.Override.SystemPrompt, tc.want)
		}
		// Parent + child still present.
		childRaw, ok := view.ChildColumns()
		if !ok {
			t.Errorf("org=%s ChildColumns=(_,false); want (_,true)", tc.orgID)
		}
		if child, isSys := childRaw.(*archetype.SystemArchetype); !isSys || child == nil || child.BundleVersion != "v1" {
			t.Errorf("org=%s child=%+v; want *SystemArchetype{BundleVersion:v1}", tc.orgID, childRaw)
		}
	}
}

// Test_LoadBySlug_ParentArchived_PerOrgOverrideReturned (T-11 PR-1,
// edge case 2). Even when the parent is terminal (status='archived'
// OR archived_at != NULL), an existing per-org override is returned.
// Spec: edge case 2 — fixed in PR-2 to honour the per-org
// override shadow semantics.
//
// The Loader's terminal predicate gates ONLY the parent lookup;
// the per-org override lookup is unconditional. When the parent is
// archived but a per-org row exists, the Loader returns found=true
// with the override surfaced.
func Test_LoadBySlug_ParentArchived_PerOrgOverrideReturned(t *testing.T) {
	dsn := catalogRequiresPostgres(t)
	resetCatalogFixtures(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO archetypes (slug, type, display_name, tagline, status, archived_at, created_by)
    		VALUES ('assistant', 'system', 'Assistant', 'Default', 'archived', '2026-08-01T00:00:00Z', 'seed')`); err != nil {
		t.Fatalf("seed archived parent: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO system_archetypes (slug, bundle_version, is_critical)
    		VALUES ('assistant', 'v1', true)`); err != nil {
		t.Fatalf("seed child: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO archetype_configurations
    		(archetype_slug, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by)
    		VALUES ('assistant', 'org-1', 'org-1-archived-prompt', '["current_time"]'::jsonb,
    		        '[]'::jsonb, NULL, 7, now(), 'alice')`); err != nil {
		t.Fatalf("seed override: %v", err)
	}

	loader := archetype.NewCatalogLoader(db)
	view, found, err := loader.LoadBySlug(context.Background(), "assistant", "org-1")
	if err != nil {
		t.Fatalf("LoadBySlug: %v", err)
	}
	if !found {
		t.Fatal("LoadBySlug returned found=false; want true (per-org override should shadow archived parent)")
	}
	if view.Override == nil {
		t.Fatal("Override=nil; want per-org override returned despite archived parent")
	}
	if view.Override.SystemPrompt != "org-1-archived-prompt" {
		t.Errorf("Override.SystemPrompt = %q; want %q", view.Override.SystemPrompt, "org-1-archived-prompt")
	}
	if view.Override.Version != 7 {
		t.Errorf("Override.Version = %d; want 7", view.Override.Version)
	}
	if view.Status != "archived" {
		t.Errorf("Status = %q; want archived (diagnostic surfaced from parent)", view.Status)
	}
	if view.ArchivedAt == nil {
		t.Error("ArchivedAt = nil; want set from parent")
	}
}

// -----------------------------------------------------------------------------
// Non-INTEGRATION contract tests for the ListByType surface
// (feat/archetype-list-endpoint, slice 1 — RED).
//
// The Postgres-backed implementation is exercised by the
// INTEGRATION-gated cases in catalog_integration_test.go (Test_ListByType_*
// family). The cases below lock the loader interface contract: any
// implementer that satisfies CatalogLoader MUST accept
// (ctx, orgID) and return ([]ArchetypeView, error). They use a
// local fake that records the orgID and returns a pre-configured
// view list, so the assertion targets the interface (assigned
// through archetype.CatalogLoader) rather than the fake's concrete
// type — that is what makes the failure mode in the RED state a
// compile error (CatalogLoader has no method ListByType yet).
// -----------------------------------------------------------------------------

// listByTypeLoader is a hermetic CatalogLoader stand-in used by the
// non-INTEGRATION contract tests in this file. It records the
// orgID passed to ListByType and returns the pre-configured view
// list, so the assertion targets the interface call shape without
// touching Postgres.
type listByTypeLoader struct {
	mu        sync.Mutex
	lastOrgID string
	views     []archetype.ArchetypeView
	err       error
}

func (f *listByTypeLoader) LoadBySlug(_ context.Context, _, _ string) (archetype.ArchetypeView, bool, error) {
	return archetype.ArchetypeView{}, false, nil
}

func (f *listByTypeLoader) WithTx(_ *sql.Tx) archetype.CatalogLoader { return f }

func (f *listByTypeLoader) ListByType(_ context.Context, orgID string) ([]archetype.ArchetypeView, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.lastOrgID = orgID
	return f.views, f.err
}

// Test_ListByType_PassesOrgID locks the interface contract: the
// loader receives the caller's orgID, returns the pre-configured
// view list, and surfaces a non-nil error only when configured.
// Runs in the standard test suite (no INTEGRATION gate) so the
// interface contract is enforced on every commit.
func Test_ListByType_PassesOrgID(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	views := []archetype.ArchetypeView{
		{
			Slug:        "assistant",
			Type:        "system",
			DisplayName: "Assistant",
			Tagline:     "Your default assistant",
			Status:      "active",
			CreatedAt:   now,
			CreatedBy:   "seed",
		},
	}
	fake := &listByTypeLoader{views: views}

	// Assign through the interface so the test exercises the
	// contract, not the concrete fake type. The compile error
	// `loader.ListByType undefined (type archetype.CatalogLoader
	// has no field or method ListByType)` is the RED state in
	// slice 1.
	var loader archetype.CatalogLoader = fake
	got, err := loader.ListByType(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("ListByType: %v", err)
	}
	if fake.lastOrgID != "org-1" {
		t.Errorf("lastOrgID = %q, want %q", fake.lastOrgID, "org-1")
	}
	if len(got) != 1 {
		t.Fatalf("len(views) = %d, want 1", len(got))
	}
	if got[0].Slug != "assistant" || got[0].Type != "system" {
		t.Errorf("views[0] = %+v, want slug=assistant type=system", got[0])
	}
}
