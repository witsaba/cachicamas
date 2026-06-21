// Package migration contains the goose-backed runner and the SQL
// files it applies. This test file exercises the runner against the
// live Postgres provisioned by docker-compose (see design §6, §15).
//
// Strict TDD discipline (per openspec/AGENTS.md and sdd-init/cachicamas):
// every behavior here is described by a failing test FIRST. This file
// was written BEFORE runner.go existed; running `go test ./...` against
// this file with no runner.go must fail with "undefined: NewGooseRunner"
// — that failure IS the RED step. The integration gate
// (`INTEGRATION=1`) is set by `make test/integration`.
//
// The unit-level table-driven cases that do not need a live DB are
// kept separate from the integration cases; running `make test`
// (INTEGRATION unset) exercises only the unit cases plus a SKIP on
// the integration cases. This keeps the PR-A contract intact:
// existing unit tests stay green, new behavior is gated.
package migration

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// integrationRunnerDB returns a *sql.DB pointed at the compose-postgres
// instance if INTEGRATION=1, or skips the test if not. This is the
// same gating pattern PR-A's driver_test.go uses — repeated here so
// runner_test.go is self-contained.
func integrationRunnerDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	// Resolve env vars the same way the driver does; we open with the
	// pgx stdlib adapter (blank import above) so the runner does not
	// depend on driver.go in this test file — the runner's own package
	// can construct its own *sql.DB when needed.
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
		t.Fatalf("ping compose postgres at %s:%s/%s: %v (is `make test/integration` running?)", host, port, dbname, err)
	}
	return db
}

// envOr reads a test env var with a default. Used by integrationRunnerDB.
func envOr(t *testing.T, key, fallback string) string {
	t.Helper()
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// resetSchemaMigrations wipes public.schema_migrations so each test
// starts from a known-empty bookkeeping table. This is the test-only
// equivalent of `docker compose down -v`; it does NOT touch the volume.
// Use sparingly — most tests want the row produced by the hello-world
// migration to assert against.
func resetSchemaMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "DROP TABLE IF EXISTS public.schema_migrations"); err != nil {
		t.Fatalf("drop schema_migrations: %v", err)
	}
}

// discardLogger returns a slog.Logger that drops every record.
// Integration tests assert behavior on the database, not on log
// output; pulling a real otelslog logger here would require a live
// collector.
func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(discardWriter{}, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }

// newTestRunner constructs a GooseRunner with sensible defaults for
// the integration tests. Lives in this test file so production code
// (runner.go) owns the public constructor; we mirror the call shape
// here so the RED->GREEN transition is a single-line impl in
// runner.go.
func newTestRunner(db *sql.DB) *GooseRunner {
	// The RED state of this test file asserts that this constructor
	// does not exist yet. Once runner.go lands it MUST match this
	// signature: NewGooseRunner(db *sql.DB, tableName string, logger *slog.Logger) *GooseRunner
	return NewGooseRunner(db, "schema_migrations", discardLogger())
}

// ---------------------------------------------------------------------------
// Category A — Integration tests (gated on INTEGRATION=1).
// ---------------------------------------------------------------------------

// TestRunner_Up_FirstBoot covers spec S-DBMIG-001: a freshly wiped
// schema_migrations table must end up with exactly one row whose
// version matches the hello-world migration, and the runner must
// return that row in the slice of applied versions.
func TestRunner_Up_FirstBoot(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)

	r := newTestRunner(db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	applied, err := r.Up(ctx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(applied) != 1 {
		t.Fatalf("Up applied %d migrations, want 1 (got %+v)", len(applied), applied)
	}
	if applied[0].ID != 20260621120000 {
		t.Errorf("applied[0].ID = %d, want 20260621120000", applied[0].ID)
	}
	if applied[0].Description == "" {
		t.Errorf("applied[0].Description must be non-empty (goose fills this from the filename)")
	}

	// Bookkeeping row must exist.
	var count int
	row := db.QueryRowContext(ctx, "SELECT count(*) FROM public.schema_migrations WHERE version_id = 20260621120000")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("public.schema_migrations has %d rows for 20260621120000, want 1", count)
	}
}

// TestRunner_Up_SecondBootIsNoOp covers spec S-DBMIG-002: a second
// Up call against a database that already has the hello-world row
// must apply zero new migrations and return nil error. The slice
// shape returned by the runner for a no-op is deliberately
// permissive (it may be nil OR an empty slice — goose v3 returns an
// empty slice).
func TestRunner_Up_SecondBootIsNoOp(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)

	r := newTestRunner(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("first Up: %v", err)
	}
	applied, err := r.Up(ctx)
	if err != nil {
		t.Fatalf("second Up: %v", err)
	}
	if len(applied) != 0 {
		t.Errorf("second Up applied %d migrations, want 0 (got %+v)", len(applied), applied)
	}

	// Bookkeeping row must still be exactly one.
	var count int
	row := db.QueryRowContext(ctx, "SELECT count(*) FROM public.schema_migrations WHERE version_id = 20260621120000")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("public.schema_migrations has %d rows for 20260621120000, want 1 (idempotency violated)", count)
	}
}

// TestRunner_Up_LexicographicOrder covers spec S-DBMIG-003: when the
// embedded directory contains a non-monotonic pair
// (`20260101000000_legacy_baseline.sql` BEFORE
// `20260621120000_hello_world.sql` lexicographically), the runner
// applies the older one first.
//
// To stay within the no-behavior-change principle of the
// hello-world migration, this test seeds the bookkeeping table
// directly with the older version's row (simulating a prior boot
// that applied it) and asserts that a fresh Up call applies only
// the hello-world row, in the correct order — proving the runner
// tolerates non-monotonic timestamps without re-applying the older
// row.
//
// The "older" timestamp (`20260101000000`) is lexicographically
// smaller than `20260621120000` — a synthetic dev build would put
// a file with that prefix in the embed; we simulate it via the
// bookkeeping table instead so the test does not depend on
// regenerating the embed.FS.
func TestRunner_Up_LexicographicOrder(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)

	// Seed the older version (lexicographically earlier timestamp)
	// directly. This is what the bookkeeping table would look like
	// after a hypothetical first boot applied 20260101000000 first.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE public.schema_migrations (
			version_id BIGINT PRIMARY KEY,
			is_applied BOOLEAN NOT NULL DEFAULT TRUE,
			tstamp TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		"INSERT INTO public.schema_migrations (version_id, is_applied) VALUES (20260101000000, TRUE)"); err != nil {
		t.Fatalf("seed older version row: %v", err)
	}

	r := newTestRunner(db)
	upCtx, upCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer upCancel()
	applied, err := r.Up(upCtx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	// Only the hello-world migration should be applied now (the older
	// row was already in the bookkeeping table).
	if len(applied) != 1 || applied[0].ID != 20260621120000 {
		t.Errorf("Up applied %+v, want exactly one migration with ID 20260621120000", applied)
	}

	// The older row's tstamp must still be earlier than the
	// hello-world row's tstamp.
	var olderTS, helloTS time.Time
	row := db.QueryRowContext(ctx, "SELECT tstamp FROM public.schema_migrations WHERE version_id = 20260101000000")
	if err := row.Scan(&olderTS); err != nil {
		t.Fatalf("scan older tstamp: %v", err)
	}
	row = db.QueryRowContext(ctx, "SELECT tstamp FROM public.schema_migrations WHERE version_id = 20260621120000")
	if err := row.Scan(&helloTS); err != nil {
		t.Fatalf("scan hello tstamp: %v", err)
	}
	if !olderTS.Before(helloTS) {
		t.Errorf("older tstamp %v must be BEFORE hello tstamp %v (lexicographic ordering violated)", olderTS, helloTS)
	}
}

// TestRunner_Up_AdvisoryLockBlocksParallelRun covers spec
// S-DBMIG-020 + S-DBMIG-021: two replicas starting at the same
// time must not both apply the hello-world migration. The runner
// uses pg_try_advisory_lock via goose.WithSessionLocker
// (lock.NewPostgresSessionLocker), so the second call blocks
// until the first releases, then proceeds to a no-op.
//
// We simulate this with TWO *sql.DB connections on the SAME
// Postgres instance — that's what two replicas would do. The
// advisory lock is session-scoped, so two sessions are required.
// Each runner uses its own *sql.DB; each gets its own session.
func TestRunner_Up_AdvisoryLockBlocksParallelRun(t *testing.T) {
	dbA := integrationRunnerDB(t)
	resetSchemaMigrations(t, dbA)

	// Open a SECOND connection (session B) for the second runner.
	host := envOr(t, "POSTGRES_HOST", "localhost")
	port := envOr(t, "POSTGRES_PORT", "5432")
	dbname := envOr(t, "POSTGRES_DB", "cachicamas_pg")
	user := envOr(t, "POSTGRES_USER", "cachicamas")
	pass := envOr(t, "POSTGRES_PASSWORD", "changeme-local-only")
	dsn := "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"

	dbB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("dbB sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = dbB.Close() })

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := dbB.PingContext(pingCtx); err != nil {
		t.Fatalf("dbB ping: %v", err)
	}

	runnerA := newTestRunner(dbA)
	runnerB := newTestRunner(dbB)

	// Hold the advisory lock from session A in a goroutine, then have
	// session B try to acquire via the runner. goose's session locker
	// polls pg_try_advisory_lock; once A releases, B acquires and
	// proceeds to a no-op (hello-world is already applied).
	var wg sync.WaitGroup
	var (
		appliedA, appliedB []domain.Version
		errA, errB         error
	)

	wg.Add(2)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		appliedA, errA = runnerA.Up(ctx)
	}()
	go func() {
		defer wg.Done()
		// Tiny delay so A enters goose.Up first and acquires the lock.
		time.Sleep(50 * time.Millisecond)
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		appliedB, errB = runnerB.Up(ctx)
	}()
	wg.Wait()

	if errA != nil {
		t.Fatalf("runnerA.Up: %v", errA)
	}
	if errB != nil {
		t.Fatalf("runnerB.Up: %v", errB)
	}

	// Exactly one of the two runners must report a non-zero applied
	// count; the other reports zero. (Both could report zero only if
	// the bookkeeping table was already seeded before A ran, which
	// resetSchemaMigrations prevents.)
	nonZero := 0
	if len(appliedA) > 0 {
		nonZero++
	}
	if len(appliedB) > 0 {
		nonZero++
	}
	if nonZero != 1 {
		t.Errorf("expected exactly one runner to apply a migration (A=%d, B=%d); got A=%+v B=%+v", len(appliedA), len(appliedB), appliedA, appliedB)
	}

	// Bookkeeping table must contain EXACTLY one row for the
	// hello-world version (no double-apply).
	var count int
	checkCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	row := dbA.QueryRowContext(checkCtx, "SELECT count(*) FROM public.schema_migrations WHERE version_id = 20260621120000")
	if err := row.Scan(&count); err != nil {
		t.Fatalf("scan schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("public.schema_migrations has %d rows for 20260621120000, want 1 (advisory lock failed)", count)
	}
}

// ---------------------------------------------------------------------------
// Category B — Unit tests (no DB required, run under `make test`).
// ---------------------------------------------------------------------------

// TestNewGooseRunner_NilSafeConstruct verifies the constructor does
// not panic on a nil db — the runner should defer its own
// nil-checks to Up/Status. We do not dial; the constructor must
// be cheap and side-effect-free.
func TestNewGooseRunner_NilSafeConstruct(t *testing.T) {
	// Intentionally pass a non-nil but never-used *sql.DB so the
	// constructor signature is satisfied without dialing.
	r := NewGooseRunner(nil, "schema_migrations", discardLogger())
	if r == nil {
		t.Fatalf("NewGooseRunner returned nil")
	}
	if r.tableName != "schema_migrations" {
		t.Errorf("tableName = %q, want %q", r.tableName, "schema_migrations")
	}
}

// TestRunner_Status_UpstreamErrorPropagates verifies that Status
// returns the error from the underlying connection — we cannot
// pass a usable *sql.DB here without dialing, so we close it
// immediately and assert the error comes back. This is the cheapest
// unit-level proof that Status actually queries the DB rather than
// returning a cached value.
func TestRunner_Status_UpstreamErrorPropagates(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://u:p@127.0.0.1:1/nope?sslmode=disable&connect_timeout=1")
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	// Close immediately so any query fails.
	_ = db.Close()

	r := NewGooseRunner(db, "schema_migrations", discardLogger())
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = r.Status(ctx)
	if err == nil {
		t.Fatalf("Status on closed pool: expected error, got nil")
	}
	// Sanity: the error should be non-empty so log readers can
	// diagnose. We don't pin the exact text — goose's wording can
	// evolve.
	var dummy interface{ Error() string }
	_ = dummy
	if errors.Unwrap(err) == nil && err.Error() == "" {
		t.Errorf("Status error must carry a message")
	}
}

// Compile-time check that GooseRunner satisfies domain.Runner once
// runner.go lands. If the runner omits Up/Status or returns the
// wrong types, this file will fail to compile — that failure is the
// RED signal for the domain port.
var _ domain.Runner = (*GooseRunner)(nil)
