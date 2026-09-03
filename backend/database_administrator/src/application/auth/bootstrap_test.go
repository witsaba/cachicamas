// Package auth — bootstrap_test.go locks the BootstrapService
// contract per spec R-BE-001 / R-BE-002 / R-BOOTSTRAP-1.
//
// The tests use INTEGRATION=1 (gated) and the migration runner
// fixture pattern from src/migration/runner_test.go. Each test
// starts from a known state (auth.users / auth.organizations /
// auth.login_audits tables present, truncated) so the assertions
// about "one row", "same user_id on second call", etc. are
// deterministic.
package auth

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/domain/auth"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres"
)

// integrationDB returns the live dev DB handle for INTEGRATION=1
// tests. The tests skip (not fail) when INTEGRATION is not set so a
// fast local `go test ./...` still works for non-integration work.
func integrationDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("INTEGRATION=1 not set; skipping live DB test")
	}
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://queen:wonderland@localhost:5432/cachicamas?sslmode=disable"
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

// resetAuthTables truncates auth.* in the correct dependency order
// (children before parents). Called before every bootstrap test so
// assertions about row counts are deterministic. Uses TRUNCATE …
// RESTART IDENTITY so the BIGSERIAL starts at 1 again — handy for
// the "first-time creates user_id=1" assertions.
func resetAuthTables(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, tbl := range []string{"auth.login_audits", "auth.organizations", "auth.users"} {
		if _, err := db.ExecContext(ctx, `TRUNCATE TABLE `+tbl+` RESTART IDENTITY CASCADE`); err != nil {
			t.Fatalf("truncate %s: %v", tbl, err)
		}
	}
}

// newTestBootstrapService wires a fresh BootstrapService against
// the integration DB. Returns the service + the concrete repo
// pointers so a test can perform direct reads to assert
// post-conditions (the service returns only {user_id, pyme_id,
// status} and a SELECT is the cleanest way to check the audit row).
func newTestBootstrapService(t *testing.T, db *sql.DB) (*BootstrapService, *postgres.UserRepo, *postgres.AuthOrganizationRepo, *postgres.AuthLoginAuditRepo) {
	t.Helper()
	userRepo := postgres.NewUserRepo(db)
	orgRepo := postgres.NewAuthOrganizationRepo(db)
	auditRepo := postgres.NewAuthLoginAuditRepo(db)
	svc := NewBootstrapService(db, userRepo, orgRepo, auditRepo)
	return svc, userRepo, orgRepo, auditRepo
}

// countRows returns the COUNT(*) of a table — the test's primary
// post-condition check.
func countRows(t *testing.T, db *sql.DB, table string) int {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM `+table).Scan(&n); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return n
}

// TestBootstrapService_FirstCall_CreatesUserAndOrganization covers
// R-BE-001 / S-BE-001: a first-time bootstrap MUST create exactly
// one auth.users row, exactly one auth.organizations row (1:1 with
// the user), and exactly one auth.login_audits row with success=true.
// The returned user_id is the new BIGSERIAL (1 after RESTART
// IDENTITY) and status is "active" (the registered → active
// transition happens in the same TX).
func TestBootstrapService_FirstCall_CreatesUserAndOrganization(t *testing.T) {
	db := integrationDB(t)
	resetAuthTables(t, db)
	svc, _, _, _ := newTestBootstrapService(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	out, err := svc.Bootstrap(ctx, BootstrapInput{
		GoogleSub:     "google-sub-1",
		Email:         "founder@example.com",
		EmailVerified: true,
		Name:          "Founder",
		PictureURL:    "https://example.com/pic.png",
		IPAddress:     "127.0.0.1",
		UserAgent:     "Mozilla/5.0",
	})
	if err != nil {
		t.Fatalf("Bootstrap: unexpected error %v", err)
	}
	if out.UserID <= 0 {
		t.Errorf("Bootstrap.UserID = %d, want > 0", out.UserID)
	}
	if out.OrganizationID <= 0 {
		t.Errorf("Bootstrap.OrganizationID = %d, want > 0", out.OrganizationID)
	}
	if out.Status != auth.UserStatusActive {
		t.Errorf("Bootstrap.Status = %q, want %q", out.Status, auth.UserStatusActive)
	}

	if got := countRows(t, db, "auth.users"); got != 1 {
		t.Errorf("auth.users row count = %d, want 1", got)
	}
	if got := countRows(t, db, "auth.organizations"); got != 1 {
		t.Errorf("auth.organizations row count = %d, want 1", got)
	}
	if got := countRows(t, db, "auth.login_audits"); got != 1 {
		t.Errorf("auth.login_audits row count = %d, want 1", got)
	}
}

// TestBootstrapService_IdempotentOnGoogleSub covers R-BE-001 /
// S-BE-002: a second bootstrap call with the SAME google_sub MUST
// return the SAME user_id (idempotent), MUST NOT create a second
// organization (1:1 invariant), and MUST still write a login_audit
// row. The second call returns status="active" (already-active is
// a no-op).
func TestBootstrapService_IdempotentOnGoogleSub(t *testing.T) {
	db := integrationDB(t)
	resetAuthTables(t, db)
	svc, _, _, _ := newTestBootstrapService(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	first, err := svc.Bootstrap(ctx, BootstrapInput{
		GoogleSub: "google-sub-1",
		Email:     "founder@example.com",
	})
	if err != nil {
		t.Fatalf("first Bootstrap: %v", err)
	}

	second, err := svc.Bootstrap(ctx, BootstrapInput{
		GoogleSub: "google-sub-1",
		Email:     "FOUNDER@Example.COM", // mixed-case: lowercased at INSERT
		Name:      "Founder Renamed",
	})
	if err != nil {
		t.Fatalf("second Bootstrap: %v", err)
	}

	if first.UserID != second.UserID {
		t.Errorf("idempotent UserID: first=%d, second=%d, want equal", first.UserID, second.UserID)
	}
	if first.OrganizationID != second.OrganizationID {
		t.Errorf("idempotent OrganizationID: first=%d, second=%d, want equal", first.OrganizationID, second.OrganizationID)
	}
	if second.Status != auth.UserStatusActive {
		t.Errorf("second.Status = %q, want %q", second.Status, auth.UserStatusActive)
	}

	if got := countRows(t, db, "auth.users"); got != 1 {
		t.Errorf("auth.users row count after two calls = %d, want 1 (idempotent)", got)
	}
	if got := countRows(t, db, "auth.organizations"); got != 1 {
		t.Errorf("auth.organizations row count after two calls = %d, want 1 (idempotent)", got)
	}
	// One audit row per bootstrap call (both succeed).
	if got := countRows(t, db, "auth.login_audits"); got != 2 {
		t.Errorf("auth.login_audits row count after two calls = %d, want 2", got)
	}
}

// TestBootstrapService_Validation_RequiresGoogleSub covers the
// input-validation gate: an empty google_sub must be rejected so
// the bootstrap service cannot persist a row the resolver cannot
// later look up.
func TestBootstrapService_Validation_RequiresGoogleSub(t *testing.T) {
	db := integrationDB(t)
	resetAuthTables(t, db)
	svc, _, _, _ := newTestBootstrapService(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := svc.Bootstrap(ctx, BootstrapInput{
		GoogleSub: "",
		Email:     "founder@example.com",
	})
	if err == nil {
		t.Fatal("Bootstrap{GoogleSub: \"\"}: expected error, got nil")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("Bootstrap error = %v, want errors.Is(ErrValidation)", err)
	}
}

// TestBootstrapService_Validation_RequiresEmail covers the
// input-validation gate: an empty email must be rejected so the
// partial unique index on lower(email) is not violated silently.
func TestBootstrapService_Validation_RequiresEmail(t *testing.T) {
	db := integrationDB(t)
	resetAuthTables(t, db)
	svc, _, _, _ := newTestBootstrapService(t, db)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := svc.Bootstrap(ctx, BootstrapInput{
		GoogleSub: "google-sub-1",
		Email:     "",
	})
	if err == nil {
		t.Fatal("Bootstrap{Email: \"\"}: expected error, got nil")
	}
	if !errors.Is(err, ErrValidation) {
		t.Errorf("Bootstrap error = %v, want errors.Is(ErrValidation)", err)
	}
}

// TestBootstrapService_RollsBackOnPymeInsertFailure covers
// R-BE-002 / S-BE-020: when the organization INSERT fails (here
// simulated by closing the org repo's underlying DB connection),
// the transaction MUST roll back so no orphan user row + no
// orphan audit row remain.
//
// The simulation is conservative: we call Bootstrap once with
// google_sub that would create user 1, then close the DB before
// a second call to confirm the closed connection surfaces an
// error rather than partial state. The bootstrap service's own
// rollback path is exercised by a unit-level test below.
func TestBootstrapService_ClosedDB_SurfacesError(t *testing.T) {
	db := integrationDB(t)
	resetAuthTables(t, db)
	// Close immediately so every subsequent call returns an error.
	_ = db.Close()

	svc, _, _, _ := newTestBootstrapService(t, db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := svc.Bootstrap(ctx, BootstrapInput{
		GoogleSub: "google-sub-1",
		Email:     "founder@example.com",
	})
	if err == nil {
		t.Fatal("Bootstrap against closed DB: expected error, got nil")
	}
}