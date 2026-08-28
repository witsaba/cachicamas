// S1-E (T3.1 of cachicamas-agent-catalog-config-reload) — INTEGRATION-gated
// fresh-database bootstrap test.
//
// Test_FreshDB_Bootstrap_CatalogAndConfigServe simulates the composition
// root's boot sequence (src/cmd/chat/migrators.go:50-58) against a fresh,
// throwaway database, running the sequence TWICE to prove two-boot
// convergence, then asserts the /api/archetypes surface serves the seeded
// catalog and config (CRL-S-020, CRL-S-021, CRL-S-022).
//
// Two-boot convergence contract (documented, NOT asserted at boot 1):
//
//	Boot 1 on a fresh DB: Run0006IfNeeded's fresh-DB guard finds
//	archetype_configurations absent (the goose runner has not applied
//	0001 yet) and returns nil; Run0007IfNeeded does the same for
//	archetype_configurations_log. The goose runner then applies
//	0001..0005. The assistant parent seed inside the 0006 SQL body does
//	NOT land on boot 1 — boot-1 state is intentionally NOT asserted.
//
//	Boot 2: the prerequisite tables now exist, the archetype_slug
//	columns are absent, so the 0006 wrapper applies (seeding the
//	assistant parent + child, reshaping the config PK, and backfilling
//	the 0003 sentinel config row to (assistant, __default__)), the
//	0007 wrapper adds the log FK, and the goose runner is a no-op.
//	From boot 2 onward the assistant catalog + config are served.
//
// The HTTP layer is exercised with the production handlers via
// RegisterArchetypeRoutes and a fake authed resolver whose orgID is
// archetype.DefaultRowOrgID — the sentinel org_id of the 0003 seed row,
// backfilled to (assistant, __default__) by the 0006 reshape. That is
// the "out-of-the-box boot state" the fresh-DB scenario locks: a fresh
// install serves the seeded assistant config before any per-org PUT.
//
// Gated by INTEGRATION=1 like every other Postgres adapter test in this
// package (catalog_test.go's precedent); skips cleanly without a DSN.
package archetype_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/agent/src/archetype"
	archetypeMigrations "github.com/cachicamas/backend/agent/src/archetype/migrations"
	"github.com/cachicamas/backend/agent/src/chat/migrator"
)

// swapDSNDatabase returns the DSN pointing at dbName on the same server
// as baseDSN. Supports URL-form DSNs (postgres://user:pass@host:port/db)
// and keyword=value DSNs (replacing or appending the dbname key).
// Returns ok=false only when the DSN shape is unrecognisable, so callers
// can fail loudly instead of dialing the wrong database.
func swapDSNDatabase(baseDSN, dbName string) (dsn string, ok bool) {
	if u, err := url.Parse(baseDSN); err == nil && u.Scheme != "" {
		u.Path = "/" + dbName
		return u.String(), true
	}
	// keyword=value form: "host=... user=... dbname=... sslmode=...".
	for _, field := range strings.Fields(baseDSN) {
		if strings.HasPrefix(field, "dbname=") {
			return strings.Replace(baseDSN, field, "dbname="+dbName, 1), true
		}
	}
	return baseDSN + " dbname=" + dbName, true
}

// freshThrowawayDatabase creates a uniquely named empty database on the
// server behind baseDSN and returns a DSN pointing at it. The database
// is dropped via t.Cleanup (DROP DATABASE … WITH (FORCE) needs PG 13+;
// the compose image is postgres:18-alpine3.24).
func freshThrowawayDatabase(t *testing.T, baseDSN string) string {
	t.Helper()

	adminDSN, _ := swapDSNDatabase(baseDSN, "postgres")
	admin, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatalf("freshThrowawayDatabase: sql.Open admin: %v", err)
	}
	if err := admin.Ping(); err != nil {
		t.Fatalf("freshThrowawayDatabase: ping admin: %v", err)
	}
	name := fmt.Sprintf("cachicamas_catalog_boot_test_%d_%d", os.Getpid(), time.Now().UnixNano())
	if _, err := admin.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, name)); err != nil {
		t.Fatalf("freshThrowawayDatabase: CREATE DATABASE %q: %v", name, err)
	}
	t.Cleanup(func() {
		if _, err := admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, name)); err != nil {
			t.Logf("freshThrowawayDatabase: DROP DATABASE %q: %v (leftover may need manual cleanup)", name, err)
		}
		_ = admin.Close()
	})
	dsn, _ := swapDSNDatabase(baseDSN, name)
	return dsn
}

// Test_FreshDB_Bootstrap_CatalogAndConfigServe — S1-E T3.1 (CRL-S-020,
// CRL-S-021, CRL-S-022). Fresh throwaway DB → two-boot simulation of the
// composition-root migration sequence → the /api/archetypes surface
// serves the seeded assistant catalog + config, and a PUT round-trips.
func Test_FreshDB_Bootstrap_CatalogAndConfigServe(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the two-boot bootstrap convergence contract requires a live Postgres)")
	}
	baseDSN := catalogRequiresPostgres(t)
	dsn := freshThrowawayDatabase(t, baseDSN)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	ctx := context.Background()
	boot := func(attempt int) {
		t.Helper()
		// Composition-root order of src/cmd/chat/migrators.go:50-58:
		// 0006 wrapper → 0007 wrapper → goose runner Up.
		if err := archetypeMigrations.Run0006IfNeeded(ctx, db); err != nil {
			t.Fatalf("boot %d: Run0006IfNeeded: %v", attempt, err)
		}
		if err := archetypeMigrations.Run0007IfNeeded(ctx, db); err != nil {
			t.Fatalf("boot %d: Run0007IfNeeded: %v", attempt, err)
		}
		provider, err := migrator.NewProvider(ctx, db, archetypeMigrations.MigrationsFS, "archetype_schema_migrations")
		if err != nil {
			t.Fatalf("boot %d: migrator.NewProvider: %v", attempt, err)
		}
		if _, err := provider.Up(ctx); err != nil {
			t.Fatalf("boot %d: provider.Up: %v", attempt, err)
		}
	}
	// Two-boot convergence: the 0006 wrapper's fresh-DB guard skips when
	// archetype_configurations is absent, so the assistant seed lands on
	// boot 2 only. Boot-1 state is deliberately NOT asserted — see the
	// file-level contract comment.
	boot(1)
	boot(2)

	// The resolver's orgID is the 0003 sentinel row's org_id: after the
	// 0006 reshape that row is (assistant, __default__), i.e. the
	// out-of-the-box config a fresh install serves before any per-org PUT.
	resolver := &fakeResolver{signIn: true, orgID: archetype.DefaultRowOrgID}
	loader := archetype.NewCatalogLoader(db)
	writer := archetype.NewPostgresWriter(db)

	e := echo.New()
	if err := archetype.RegisterArchetypeRoutes(e, resolver, loader, writer); err != nil {
		t.Fatalf("RegisterArchetypeRoutes: %v", err)
	}

	// CRL-S-020: GET /api/archetypes → 200 array containing slug
	// `assistant` with type=system.
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/archetypes", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/archetypes: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var views []archetype.ArchetypeView
	if err := json.Unmarshal(rec.Body.Bytes(), &views); err != nil {
		t.Fatalf("GET /api/archetypes: decode body: %v", err)
	}
	assistantListed := false
	for _, v := range views {
		if v.Slug == "assistant" {
			assistantListed = true
			if v.Type != "system" {
				t.Errorf("GET /api/archetypes: assistant type = %q, want system", v.Type)
			}
		}
	}
	if !assistantListed {
		t.Fatalf("GET /api/archetypes: assistant missing from directory; views=%+v", views)
	}

	// CRL-S-021: GET /api/archetypes/assistant/config/ → 200 with a
	// non-empty system_prompt (the 0003 seed served through the JOIN).
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant/config/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /api/archetypes/assistant/config/: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	var view archetype.ArchetypeView
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatalf("GET config: decode body: %v", err)
	}
	if view.Override == nil || strings.TrimSpace(view.Override.SystemPrompt) == "" {
		t.Fatalf("GET config: Override=%+v; want non-empty system_prompt served from the boot seed", view.Override)
	}

	// CRL-S-022: PUT a valid body → 200, then GET returns the PUT prompt.
	putBody := validPutBody()
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, newPutRequest(t, putBody))
	if rec.Code != http.StatusOK {
		t.Fatalf("PUT config: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	rec = httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/archetypes/assistant/config/", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("GET config after PUT: status = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &view); err != nil {
		t.Fatalf("GET config after PUT: decode body: %v", err)
	}
	wantPrompt := putBody["system_prompt"].(string)
	if view.Override == nil || view.Override.SystemPrompt != wantPrompt {
		t.Errorf("GET config after PUT: Override=%+v; want system_prompt %q round-tripped", view.Override, wantPrompt)
	}
}
