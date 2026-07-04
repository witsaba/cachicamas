// Package postgres_test contains the integration test suite for the
// identity repository adapter. The tests are gated on INTEGRATION=1
// because they need a live Postgres (the same compose-provisioned
// instance used by the migration runner tests and the organization
// repo tests). The handler-level unit tests in
// src/interfaces/http/auth_middleware_test.go (PR-3) cover the
// non-DB paths with a fake.
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// identity_repository.go existed. Running
// `INTEGRATION=1 go test ./src/infrastructure/postgres/... -run TestIdentityRepository`
// with no PostgresIdentityRepo type must fail with "undefined:
// PostgresIdentityRepo" — that failure IS the RED step.
package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres"
)

// ---------------------------------------------------------------------------
// Helpers (re-use the shape used by organization_repo_test.go)
// ---------------------------------------------------------------------------

func identityIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	host := envOrIdentity(t, "POSTGRES_HOST", "localhost")
	port := envOrIdentity(t, "POSTGRES_PORT", "5432")
	dbname := envOrIdentity(t, "POSTGRES_DB", "cachicamas_pg")
	user := envOrIdentity(t, "POSTGRES_USER", "cachicamas")
	pass := envOrIdentity(t, "POSTGRES_PASSWORD", "changeme-local-only")
	dsn := "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping compose postgres: %v (is `make test/integration` running?)", err)
	}
	return db
}

// envOrIdentity returns the env value or a default. Local alias of
// the helper in organization_repo_test.go (which is package-private
// in the same package, so we can reuse it via identityIntegrationDB
// — but for readability we keep this short helper here too).
func envOrIdentity(t *testing.T, key, def string) string {
	t.Helper()
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

// seedIdentity inserts a fresh identity.user row + identity.account
// row directly via SQL and returns the user id. The function is
// used by every sub-test that needs a known row to look up. It
// returns a cleanup func that DELETEs the seeded rows so tests do
// not leave data behind.
//
// Implementation note: we INSERT directly (instead of calling the
// repository) because the repository only exposes LookupByEmail;
// the slice does not yet own the create / update use case (those
// live on the frontend signIn callback in PR-2).
func seedIdentity(t *testing.T, db *sql.DB, email, name, provider, providerAccountID string) (userID int64, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := tx.QueryRowContext(ctx,
		`INSERT INTO identity.user (email, name) VALUES ($1, $2) RETURNING id`,
		email, name,
	).Scan(&userID); err != nil {
		t.Fatalf("seed identity.user: %v", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO identity.account (user_id, provider, provider_account_id) VALUES ($1, $2, $3)`,
		userID, provider, providerAccountID,
	); err != nil {
		t.Fatalf("seed identity.account: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("seed commit: %v", err)
	}
	cleanup = func() {
		_, _ = db.ExecContext(context.Background(),
			`DELETE FROM identity.user WHERE id = $1`, userID,
		)
	}
	return userID, cleanup
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestIdentityRepository_LookupByEmail_Hit is the happy path:
// seed a user + account, look up by email, assert all fields are
// returned correctly. Per spec S-IS (identity-schema) and the
// backend-auth-middleware spec R-BAM-040 / S-BAM-040.
func TestIdentityRepository_LookupByEmail_Hit(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	const email = "lookup-hit@example.com"
	const name = "Lookup Hit"
	const provider = "github"
	const pid = "12345"
	userID, cleanup := seedIdentity(t, db, email, name, provider, pid)
	defer cleanup()

	got, err := repo.LookupByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("LookupByEmail(%q): unexpected error: %v", email, err)
	}
	if got.ID != userID {
		t.Errorf("ID:\n got  = %d\n want = %d", got.ID, userID)
	}
	if got.Email != email {
		t.Errorf("Email:\n got  = %q\n want = %q", got.Email, email)
	}
	if got.Name != name {
		t.Errorf("Name:\n got  = %q\n want = %q", got.Name, name)
	}
	if got.Provider != provider {
		t.Errorf("Provider:\n got  = %q\n want = %q", got.Provider, provider)
	}
	if got.ProviderAccountID != pid {
		t.Errorf("ProviderAccountID:\n got  = %q\n want = %q", got.ProviderAccountID, pid)
	}
}

// TestIdentityRepository_LookupByEmail_Miss asserts the no-row
// path: looking up an email that is not in identity.user must
// return *domain.IdentityNotFoundError (NOT sql.ErrNoRows), so the
// HTTP handler can errors.As it and map to a 404 envelope.
func TestIdentityRepository_LookupByEmail_Miss(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	_, err := repo.LookupByEmail(context.Background(), "ghost-miss@example.com")
	if err == nil {
		t.Fatalf("expected not-found error; got nil")
	}
	var target *domain.IdentityNotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("expected error chain to carry *domain.IdentityNotFoundError; got %v", err)
	}
	if target.Email != "ghost-miss@example.com" {
		t.Errorf("expected error.Email to be passed through; got %q", target.Email)
	}
}

// TestIdentityRepository_LookupByEmail_CaseInsensitive asserts the
// CITEXT behavior: the email column is case-insensitive (because
// identity_schema R-IS-001 uses CITEXT), so looking up with a
// different case MUST return the same row.
func TestIdentityRepository_LookupByEmail_CaseInsensitive(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	const email = "Lookup-Mixed@example.com"
	userID, cleanup := seedIdentity(t, db, email, "Case Mix", "github", "99999")
	defer cleanup()

	got, err := repo.LookupByEmail(context.Background(), "lookup-mixed@example.com")
	if err != nil {
		t.Fatalf("LookupByEmail lower-case: unexpected error: %v", err)
	}
	if got.ID != userID {
		t.Errorf("case-insensitive lookup returned the wrong row:\n got  = %d\n want = %d", got.ID, userID)
	}
}
