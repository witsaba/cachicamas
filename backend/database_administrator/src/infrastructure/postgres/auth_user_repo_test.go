//go:build integration

// Package postgres — auth_user_repo_test.go locks the UserRepo
// adapter contract at the DB level per spec R-DB-001 / R-BOOTSTRAP-1
// / S-DB-002 / S-DB-010 / S-DB-011 / S-DB-030.
//
// Tests are INTEGRATION=1 gated; skipped (not failed) without a
// live Postgres connection. The dev compose stack provides the DB
// (see openspec/project.md).
package postgres

    import (
    	"context"
    	"database/sql"
    	"io"
    	"log/slog"
    	"os"
    	"testing"

    	_ "github.com/jackc/pgx/v5/stdlib"

    	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
    	"github.com/cachicamas/backend/database_administrator/src/migration"
    )

    // TestMain applies migrations once before any auth_*_test runs and
    // truncates (does NOT drop) auth.* tables after the test binary
    // completes. Running the runner here means the auth.* tables exist
    // regardless of the order packages run in (the runner tests in
    // migration/runner_test.go DROP auth.* in their cleanup, which would
    // otherwise leave auth tests with no schema). Truncate (not drop)
    // keeps the schema in place for downstream test packages (e.g.
    // workspace_repo_test that seeds auth.users(id=1) via FK).
    func TestMain(m *testing.M) {
    	if os.Getenv("INTEGRATION") != "1" {
    		// Fast path: not running integration, no setup needed.
    		os.Exit(m.Run())
    	}
    	dsn := authTestMainDSN()
    	db, err := sql.Open("pgx", dsn)
    	if err != nil {
    		// sql.Open never errors on a valid DSN; surface anything that
    		// does happen as a hard failure so the test binary aborts.
    		panic("auth_test TestMain sql.Open: " + err.Error())
    	}
    	defer db.Close()
    	ctx, cancel := context.WithTimeout(context.Background(), 30_000_000_000)
    	defer cancel()
    	if err := db.PingContext(ctx); err != nil {
    		// DB not reachable: skip every test in the package via a
    		// sentinel env var that authIntegrationDB honours. This
    		// keeps the skip-path single-sourced and avoids fan-out
    		// from TestMain into per-test skip logic.
    		os.Setenv("AUTH_INTEGRATION_UNREACHABLE", "1")
    		os.Exit(m.Run())
    	}
    	runner := migration.NewGooseRunner(db, "schema_migrations", slog.New(slog.NewTextHandler(io.Discard, nil)))
    	// Force a clean re-application of ALL migrations from scratch.
    	// The dev compose Postgres can be left in a state where the
    	// schema_migrations bookkeeping lists everything as applied but
    	// the actual tables were DROPPED by a prior runner_test cleanup
    	// (which calls DROP TABLE ... CASCADE without clearing the
    	// bookkeeping). Partial re-application of just the auth
    	// migrations is not safe: the google_auth DDL rewrites the FK on
    	// workspace.owner_user_id, so workspace (and the other 7 tables
    	// google_auth depends on) must exist when Up() runs. Wiping
    	// schema_migrations + dropping every migration-managed table +
    	// running Up() restores the canonical PR-1 runner_test baseline
    	// (see runner_test.go's wipeNewTables + resetSchemaMigrations).
    	wipeAllMigrationTables(ctx, db)
    	if _, err := runner.Up(ctx); err != nil {
    		panic("auth_test TestMain migration.Up: " + err.Error())
    	}
    	// Cleanup runs after m.Run(); truncate (NOT drop) so downstream
    	// test packages can still use the auth.* schema (workspace_repo
    	// tests seed auth.users(id=1) via FK).
    	defer func() {
    		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5_000_000_000)
    		defer cancel()
    		for _, tbl := range []string{"auth.login_audits", "auth.organizations", "auth.users"} {
    			_, _ = db.ExecContext(cleanupCtx, `TRUNCATE TABLE `+tbl+` RESTART IDENTITY CASCADE`)
    		}
    	}()
    	os.Exit(m.Run())
    }

    // authTestMainDSN is the TestMain-only DSN assembly. Kept separate
    // from authIntegrationDB so TestMain can build the DSN before any
    // t.Helper is available (TestMain does not receive *testing.T).
    func authTestMainDSN() string {
    	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
    		return v
    	}
    	host := envOrDefault("POSTGRES_HOST", "localhost")
    	port := envOrDefault("POSTGRES_PORT", "5432")
    	dbname := envOrDefault("POSTGRES_DB", "cachicamas_pg")
    	user := envOrDefault("POSTGRES_USER", "cachicamas")
    	pass := envOrDefault("POSTGRES_PASSWORD", "changeme-local-only")
    	return "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
    }

    // wipeAllMigrationTables drops every migration-managed table (and
    // the public.schema_migrations bookkeeping) so runner.Up() can
    // reapply all 14 migrations from a clean slate. Mirrors the
    // wipeNewTables + resetSchemaMigrations pair from
    // migration/runner_test.go. Order matters: children before parents
    // so FK chains resolve.
    func wipeAllMigrationTables(ctx context.Context, db *sql.DB) {
    	stmts := []string{
    		// auth.* children first (login_audits FKs users).
    		"DROP TABLE IF EXISTS auth.login_audits CASCADE",
    		"DROP TABLE IF EXISTS auth.organizations CASCADE",
    		"DROP TABLE IF EXISTS auth.users CASCADE",
    		// spec_phase / spec / task / milestone reference each other.
    		"DROP TABLE IF EXISTS spec_phase CASCADE",
    		"DROP TABLE IF EXISTS spec CASCADE",
    		"DROP TABLE IF EXISTS task CASCADE",
    		"DROP TABLE IF EXISTS milestone CASCADE",
    		// requirement_spike references requirement.
    		"DROP TABLE IF EXISTS requirement_spike CASCADE",
    		"DROP TABLE IF EXISTS requirement CASCADE",
    		// project references organization.
    		"DROP TABLE IF EXISTS project CASCADE",
    		// organization may FK to auth.users.
    		"DROP TABLE IF EXISTS organization CASCADE",
    		// sync_job FKs workspace + repo.
    		"DROP TABLE IF EXISTS sync_job CASCADE",
    		// workspace references organization + auth.users.
    		"DROP TABLE IF EXISTS workspace CASCADE",
    		// legacy identity.* in FK-safe order.
    		"DROP TABLE IF EXISTS identity.account CASCADE",
    		"DROP TABLE IF EXISTS identity.user CASCADE",
    		// prompt / prompt_revision / skill / skill_revision from
    		// prompts + skills PR-1 entries; safe to drop unconditionally.
    		"DROP TABLE IF EXISTS prompt_revision CASCADE",
    		"DROP TABLE IF EXISTS prompt CASCADE",
    		"DROP TABLE IF EXISTS skill_revision CASCADE",
    		"DROP TABLE IF EXISTS skill CASCADE",
    		// bookkeeping last.
    		"DROP TABLE IF EXISTS public.schema_migrations",
    	}
    	for _, s := range stmts {
    		if _, err := db.ExecContext(ctx, s); err != nil {
    		panic("auth_test TestMain wipeAllMigrationTables (" + s + "): " + err.Error())
    		}
    	}
    }

// authIntegrationDB returns the live dev DB handle (or skips).
// DSN assembly matches the canonical pattern from migration/runner_test.go:
// POSTGRES_HOST/PORT/DB/USER/PASSWORD env vars, falling back to the
// dev compose stack defaults (cachicamas_pg / cachicamas / changeme-local-only).
// TEST_DATABASE_URL overrides everything when set.
//
// If TestMain already determined the DB is unreachable (it set the
// AUTH_INTEGRATION_UNREACHABLE sentinel during startup), this helper
// skips immediately without retrying the Ping. Single-sourced skip
// path keeps the message consistent across all auth_*_test.go files.
func authIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("INTEGRATION=1 not set; skipping live DB test")
	}
	if os.Getenv("AUTH_INTEGRATION_UNREACHABLE") == "1" {
		t.Skip("dev compose Postgres not reachable (TestMain sentinel); skipping live DB test")
	}
	var dsn string
	if v := os.Getenv("TEST_DATABASE_URL"); v != "" {
		dsn = v
	} else {
		host := envOrDefault("POSTGRES_HOST", "localhost")
		port := envOrDefault("POSTGRES_PORT", "5432")
		dbname := envOrDefault("POSTGRES_DB", "cachicamas_pg")
		user := envOrDefault("POSTGRES_USER", "cachicamas")
		pass := envOrDefault("POSTGRES_PASSWORD", "changeme-local-only")
		dsn = "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Skipf("Postgres not reachable at %q: %v", dsn, err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// envOrDefault returns os.Getenv(key) or fallback when unset.
func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resetAuthTablesForRepoTest truncates auth.* in dependency order.
func resetAuthTablesForRepoTest(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5_000_000_000)
	defer cancel()
	for _, tbl := range []string{"auth.login_audits", "auth.organizations", "auth.users"} {
		if _, err := db.ExecContext(ctx, `TRUNCATE TABLE `+tbl+` RESTART IDENTITY CASCADE`); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
}

// TestUserRepo_FindByGoogleSub covers the lookup path.
func TestUserRepo_FindByGoogleSub(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)
	repo := NewUserRepo(db)

	ctx := context.Background()
	u := auth.NewUser("imm@example.com", "imm-sub-1")
	created, err := repo.InsertRegistered(ctx, db, u)
	if err != nil {
		t.Fatalf("InsertRegistered: %v", err)
	}

	got, err := repo.FindByGoogleSub(ctx, db, "imm-sub-1")
	if err != nil {
		t.Fatalf("FindByGoogleSub: %v", err)
	}
	if got == nil {
		t.Fatal("FindByGoogleSub: nil result")
	}
	if got.ID != created.ID {
		t.Errorf("ID = %d, want %d", got.ID, created.ID)
	}
	if got.Email != "imm@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "imm@example.com")
	}
	if got.Status != auth.UserStatusRegistered {
		t.Errorf("Status = %q, want %q (default)", got.Status, auth.UserStatusRegistered)
	}
}

// TestUserRepo_FindByGoogleSub_MissReturnsNil covers the not-found
// contract.
func TestUserRepo_FindByGoogleSub_MissReturnsNil(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)
	repo := NewUserRepo(db)

	got, err := repo.FindByGoogleSub(context.Background(), db, "nonexistent")
	if err != nil {
		t.Fatalf("FindByGoogleSub: %v", err)
	}
	if got != nil {
		t.Errorf("FindByGoogleSub on miss: got=%v, want nil", got)
	}
}

// TestUserRepo_InsertRegistered_Idempotent covers S-BE-002 /
// R-BOOTSTRAP-1: a second InsertRegistered for the same
// google_sub returns the existing user, not a new row.
func TestUserRepo_InsertRegistered_Idempotent(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)
	repo := NewUserRepo(db)

	ctx := context.Background()
	first, err := repo.InsertRegistered(ctx, db, auth.NewUser("first@example.com", "sub-1"))
	if err != nil {
		t.Fatalf("first InsertRegistered: %v", err)
	}
	second, err := repo.InsertRegistered(ctx, db, auth.NewUser("second@example.com", "sub-1"))
	if err != nil {
		t.Fatalf("second InsertRegistered: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("idempotent insert: first.ID=%d, second.ID=%d, want equal", first.ID, second.ID)
	}
	if second.Email != "first@example.com" {
		t.Errorf("idempotent insert: second.Email=%q, want first insert's email %q", second.Email, "first@example.com")
	}
}

// TestUserRepo_UpdateLoginFields_DoesNotAdvanceCreatedAt covers
// S-DB-002: an UPDATE that touches login fields MUST NOT modify
// created_at.
func TestUserRepo_UpdateLoginFields_DoesNotAdvanceCreatedAt(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)
	repo := NewUserRepo(db)

	ctx := context.Background()
	created, err := repo.InsertRegistered(ctx, db, auth.NewUser("imm@example.com", "imm-sub-1"))
	if err != nil {
		t.Fatalf("InsertRegistered: %v", err)
	}
	originalCreatedAt := created.CreatedAt

	u := auth.NewUser("imm@example.com", "imm-sub-1")
	u.Name = "Renamed"
	if err := repo.UpdateLoginFields(ctx, db, created.ID, u); err != nil {
		t.Fatalf("UpdateLoginFields: %v", err)
	}

	got, err := repo.FindByID(ctx, db, created.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got == nil {
		t.Fatal("FindByID: nil")
	}
	if !got.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("created_at drifted: original=%v, after=%v", originalCreatedAt, got.CreatedAt)
	}
	if got.Name != "Renamed" {
		t.Errorf("Name after update = %q, want %q", got.Name, "Renamed")
	}
}

// TestUserRepo_PromoteToActive_Idempotent covers the
// registered → active transition: re-call on an already-active
// user is a no-op.
func TestUserRepo_PromoteToActive_Idempotent(t *testing.T) {
	db := authIntegrationDB(t)
	resetAuthTablesForRepoTest(t, db)
	repo := NewUserRepo(db)

	ctx := context.Background()
	created, err := repo.InsertRegistered(ctx, db, auth.NewUser("imm@example.com", "imm-sub-1"))
	if err != nil {
		t.Fatalf("InsertRegistered: %v", err)
	}

	if err := repo.PromoteToActive(ctx, db, created.ID); err != nil {
		t.Fatalf("first PromoteToActive: %v", err)
	}
	got, _ := repo.FindByID(ctx, db, created.ID)
	if got == nil || got.Status != auth.UserStatusActive {
		t.Fatalf("after first promote: status=%q, want active", got.Status)
	}

	// Second call: must NOT error (idempotent).
	if err := repo.PromoteToActive(ctx, db, created.ID); err != nil {
		t.Errorf("second PromoteToActive: %v (expected no-op)", err)
	}
}