// Package postgres_test contains the integration test suite for
// the organization repository adapter. The tests are gated on
// INTEGRATION=1 because they need a live Postgres (the same
// compose-provisioned instance used by the migration runner
// tests). The unit-level cases that do not need a live DB are
// kept separate.
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// organization_repo.go existed. Running
// `INTEGRATION=1 go test ./src/infrastructure/postgres/...` with
// no OrgRepo type must fail with
// "undefined: OrgRepo" — that failure IS the RED step.
package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"database/sql"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// integrationDB returns a *sql.DB pointed at the compose-postgres
// instance if INTEGRATION=1, or skips the test if not. Mirrors the
// pattern in src/migration/runner_test.go.
func integrationDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	host := envOr(t, "POSTGRES_HOST", "localhost")
	port := envOr(t, "POSTGRES_PORT", "5432")
	dbname := envOr(t, "POSTGRES_DB", "cachicamas_pg")
	user := envOr(t, "POSTGRES_USER", "cachicamas")
	pass := envOr(t, "POSTGRES_PASSWORD", "changeme-local-only")

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

func envOr(t *testing.T, key, fallback string) string {
	t.Helper()
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ensureMigrations runs a minimal Up so the organization table
// exists. The integration DB starts from a known-empty schema; we
// can either rely on the parent worktree's migration runner having
// run, or we can issue the DDL here. For test isolation we use
// the project's migration runner when possible and fall back to a
// direct DDL statement that mirrors the locked migration.
func ensureMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Mirror the locked DDL from migration/sql/20260622120000_orgs_and_projects.sql.
	// We use CREATE TABLE IF NOT EXISTS so the helper is idempotent.
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS organization (
		    id              BIGSERIAL    PRIMARY KEY,
		    shortname       TEXT,
		    full_name       TEXT         NOT NULL,
		    identification  TEXT         NOT NULL UNIQUE,
		    is_active       BOOLEAN      NOT NULL DEFAULT TRUE,
		    email           TEXT,
		    phone           TEXT,
		    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatalf("ensure organization table: %v", err)
	}
}

// truncateOrgs wipes the organization table for test isolation.
// Uses TRUNCATE ... RESTART IDENTITY CASCADE so identity is
// reset and any FK chain is torn down cleanly.
func truncateOrgs(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "TRUNCATE organization RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate organization: %v", err)
	}
}

// stringPtr is a tiny helper to take the address of a string literal
// in test fixtures.
func stringPtr(s string) *string { return &s }

// ---------------------------------------------------------------------------
// Category A — Integration tests (gated on INTEGRATION=1).
// ---------------------------------------------------------------------------

// TestRepo_Insert_AndSelectByID_RoundTrip covers spec B-1 (DB side):
// a freshly inserted org must be readable via SelectByID with
// parity on every field that the application layer relies on.
func TestRepo_Insert_AndSelectByID_RoundTrip(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	truncateOrgs(t, db)

	repo := postgres.NewOrgRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	in := &domain.Organization{
		FullName:       "Acme Industrial S.A.",
		Identification: "acme-industrial-roundtrip",
		ShortName:      stringPtr("Acme"),
		Email:          stringPtr("hello@acme.example"),
		Phone:          stringPtr("+14155552671"),
	}

	got, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if got.ID <= 0 {
		t.Errorf("Insert returned ID = %d, want > 0", got.ID)
	}
	if !got.IsActive {
		t.Errorf("Insert returned IsActive = false, want true (DB default)")
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("Insert returned CreatedAt zero, want non-zero (DB default now())")
	}
	if got.UpdatedAt.IsZero() {
		t.Errorf("Insert returned UpdatedAt zero, want non-zero (DB default now())")
	}
	if got.FullName != in.FullName {
		t.Errorf("FullName = %q, want %q", got.FullName, in.FullName)
	}
	if got.Identification != in.Identification {
		t.Errorf("Identification = %q, want %q", got.Identification, in.Identification)
	}
	if got.ShortName == nil || *got.ShortName != *in.ShortName {
		t.Errorf("ShortName = %v, want %q", got.ShortName, *in.ShortName)
	}
	if got.Email == nil || *got.Email != *in.Email {
		t.Errorf("Email = %v, want %q", got.Email, *in.Email)
	}
	if got.Phone == nil || *got.Phone != *in.Phone {
		t.Errorf("Phone = %v, want %q", got.Phone, *in.Phone)
	}

	// Round-trip via SelectByID.
	got2, err := repo.SelectByID(ctx, got.ID)
	if err != nil {
		t.Fatalf("SelectByID: %v", err)
	}
	if got2.ID != got.ID {
		t.Errorf("SelectByID ID = %d, want %d", got2.ID, got.ID)
	}
	if got2.FullName != got.FullName {
		t.Errorf("SelectByID FullName = %q, want %q", got2.FullName, got.FullName)
	}
	if got2.Identification != got.Identification {
		t.Errorf("SelectByID Identification = %q, want %q", got2.Identification, got.Identification)
	}
}

// TestRepo_Insert_DuplicateIdentification_ReturnsConflictError covers
// spec B-4: a second Insert with the same identification must
// return *domain.ConflictError (handler maps to 409). The
// underlying pgx unique-violation must NOT leak through.
func TestRepo_Insert_DuplicateIdentification_ReturnsConflictError(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	truncateOrgs(t, db)

	repo := postgres.NewOrgRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	first := &domain.Organization{
		FullName:       "Acme One",
		Identification: "acme-dup",
	}
	if _, err := repo.Insert(ctx, first); err != nil {
		t.Fatalf("first Insert: %v", err)
	}

	second := &domain.Organization{
		FullName:       "Acme Two",
		Identification: "acme-dup",
	}
	_, err := repo.Insert(ctx, second)
	if err == nil {
		t.Fatalf("second Insert: expected error, got nil")
	}
	var cerr *domain.ConflictError
	if !errors.As(err, &cerr) {
		t.Fatalf("second Insert returned %T, want *domain.ConflictError", err)
	}
	if cerr.Code() != "conflict" {
		t.Errorf("Code() = %q, want %q", cerr.Code(), "conflict")
	}
}

// TestRepo_SelectAll_EmptyAndOrdered covers spec B-5 + B-5b:
// SelectAll returns an empty slice (NOT nil panic) when zero rows
// exist, and returns rows in (created_at ASC, id ASC) order
// regardless of insertion order.
func TestRepo_SelectAll_EmptyAndOrdered(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	truncateOrgs(t, db)

	repo := postgres.NewOrgRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	t.Run("empty", func(t *testing.T) {
		got, err := repo.SelectAll(ctx)
		if err != nil {
			t.Fatalf("SelectAll (empty): %v", err)
		}
		if len(got) != 0 {
			t.Errorf("SelectAll (empty) = %d rows, want 0", len(got))
		}
	})

	t.Run("ordered", func(t *testing.T) {
		// Insert two rows with a small sleep so created_at differs.
		if _, err := repo.Insert(ctx, &domain.Organization{
			FullName: "First", Identification: "ord-first",
		}); err != nil {
			t.Fatalf("insert first: %v", err)
		}
		// Postgres TIMESTAMPTZ has microsecond resolution; a 10ms gap
		// is more than enough to guarantee distinct created_at values
		// for the ordering assertion.
		time.Sleep(10 * time.Millisecond)
		if _, err := repo.Insert(ctx, &domain.Organization{
			FullName: "Second", Identification: "ord-second",
		}); err != nil {
			t.Fatalf("insert second: %v", err)
		}

		got, err := repo.SelectAll(ctx)
		if err != nil {
			t.Fatalf("SelectAll (ordered): %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("SelectAll (ordered) = %d rows, want 2", len(got))
		}
		if got[0].Identification != "ord-first" {
			t.Errorf("got[0].Identification = %q, want %q (created_at ASC ordering)",
				got[0].Identification, "ord-first")
		}
		if got[1].Identification != "ord-second" {
			t.Errorf("got[1].Identification = %q, want %q (created_at ASC ordering)",
				got[1].Identification, "ord-second")
		}
	})
}

// TestRepo_SelectByID_NotFound_ReturnsNotFoundError covers spec B-6b:
// a SelectByID with no matching row returns *domain.NotFoundError,
// and the handler maps that to HTTP 404.
func TestRepo_SelectByID_NotFound_ReturnsNotFoundError(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	truncateOrgs(t, db)

	repo := postgres.NewOrgRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := repo.SelectByID(ctx, 999_999_999)
	if err == nil {
		t.Fatalf("SelectByID (missing): expected error, got nil (org=%+v)", got)
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("SelectByID (missing) returned %T, want *domain.NotFoundError", err)
	}
	if nerr.Code() != "not_found" {
		t.Errorf("Code() = %q, want %q", nerr.Code(), "not_found")
	}
	if nerr.Resource != "organization" {
		t.Errorf("Resource = %q, want %q", nerr.Resource, "organization")
	}
}

// ---------------------------------------------------------------------------
// Tests for HasOrganization (R-OW-005 / S-OW-040..044)
//
// The HasOrganization repo method backs the /setup-state endpoint.
// Implementation uses SELECT EXISTS (SELECT 1 FROM organization) so
// the query short-circuits at the first row and never materializes
// the result set (S-OW-044).
// ---------------------------------------------------------------------------

// TestRepo_HasOrganization_EmptyTable verifies that an empty
// organizations table yields (false, nil) (S-OW-040).
func TestRepo_HasOrganization_EmptyTable(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	truncateOrgs(t, db)

	repo := postgres.NewOrgRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := repo.HasOrganization(ctx)
	if err != nil {
		t.Fatalf("HasOrganization (empty): %v", err)
	}
	if got {
		t.Errorf("HasOrganization (empty) = true, want false")
	}
}

// TestRepo_HasOrganization_OneRow verifies that a single row yields
// (true, nil) (S-OW-041).
func TestRepo_HasOrganization_OneRow(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	truncateOrgs(t, db)

	if _, err := db.ExecContext(
		context.Background(),
		`INSERT INTO organization (shortname, full_name, identification, is_active) VALUES ($1, $2, $3, TRUE)`,
		"only", "Only Org", "ord-only-has",
	); err != nil {
		t.Fatalf("insert only org: %v", err)
	}

	repo := postgres.NewOrgRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := repo.HasOrganization(ctx)
	if err != nil {
		t.Fatalf("HasOrganization (one row): %v", err)
	}
	if !got {
		t.Errorf("HasOrganization (one row) = false, want true")
	}
}

// TestRepo_HasOrganization_ManyRows verifies that any number of rows
// greater than zero collapses to (true, nil) (S-OW-042). This
// protects against an implementation that returns the count and
// then compares > 0 with the wrong operator.
func TestRepo_HasOrganization_ManyRows(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	truncateOrgs(t, db)

	for i, id := range []string{"ord-multi-a", "ord-multi-b", "ord-multi-c"} {
		if _, err := db.ExecContext(
			context.Background(),
			`INSERT INTO organization (shortname, full_name, identification, is_active) VALUES ($1, $2, $3, TRUE)`,
			nil, fmt.Sprintf("Org %d", i), id,
		); err != nil {
			t.Fatalf("insert org %d: %v", i, err)
		}
	}

	repo := postgres.NewOrgRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	got, err := repo.HasOrganization(ctx)
	if err != nil {
		t.Fatalf("HasOrganization (many rows): %v", err)
	}
	if !got {
		t.Errorf("HasOrganization (many rows) = false, want true")
	}
}
