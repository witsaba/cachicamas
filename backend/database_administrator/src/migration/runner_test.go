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
	wipeNewTables(t, db)

	r := newTestRunner(db)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	applied, err := r.Up(ctx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(applied) < 1 {
		t.Fatalf("Up applied %d migrations, want at least 1 (got %+v)", len(applied), applied)
	}
	if applied[0].ID != 20260621120000 {
		t.Errorf("applied[0].ID = %d, want 20260621120000", applied[0].ID)
	}
	if applied[0].Description == "" {
		t.Errorf("applied[0].Description must be non-empty (goose fills this from the filename)")
	}
	// Find the hello-world migration in the applied slice regardless of
	// how many subsequent migrations have been added to the embed.FS
	// (the witsaba-core-tables change adds 3 new files on top of the
	// hello-world; this test must stay valid as more files land).
	foundHello := false
	for _, v := range applied {
		if v.ID == 20260621120000 {
			foundHello = true
			break
		}
	}
	if !foundHello {
		t.Errorf("applied slice %+v must contain version 20260621120000", applied)
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
	wipeNewTables(t, db)

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
	wipeNewTables(t, db)

	// Seed the older version (lexicographically earlier timestamp)
	// directly. This is what the bookkeeping table would look like
	// after a hypothetical first boot applied 20260101000000 first.
	// We mirror goose v3.27.1's actual schema (id, version_id,
	// is_applied, tstamp) so a follow-up Up() can both insert the
	// new row and read the existing one.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE public.schema_migrations (
			id         BIGSERIAL    PRIMARY KEY,
			version_id BIGINT       NOT NULL,
			is_applied BOOLEAN      NOT NULL,
			tstamp     TIMESTAMPTZ  NOT NULL DEFAULT now()
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
	// row was already in the bookkeeping table). Allow additional
	// migrations to be applied if new files have landed in the
	// embed.FS (the witsaba-core-tables change adds 3 new files);
	// this test asserts the hello-world is in the slice and the older
	// row was NOT re-applied.
	helloFound := false
	for _, v := range applied {
		if v.ID == 20260621120000 {
			helloFound = true
		}
		if v.ID == 20260101000000 {
			t.Errorf("Up re-applied older version %d (expected it to be skipped because the bookkeeping row was already present)", v.ID)
		}
	}
	if !helloFound {
		t.Errorf("Up applied %+v, want at least version 20260621120000", applied)
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
	wipeNewTables(t, dbA)

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
// witsaba-core-tables integration tests (gated on INTEGRATION=1).
//
// Six new tests cover the schema invariants locked in
// openspec/changes/witsaba-core-tables/{proposal,spec,design}/. Each
// test:
//   1. resets public.schema_migrations,
//   2. runs runner.Up(ctx) to apply the 3 new migration files,
//   3. asserts against the resulting schema and bookkeeping,
//   4. cleans up with TRUNCATE ... CASCADE on the new tables
//      (per design §3.2 — append-mostly bookkeeping is preserved).
// ---------------------------------------------------------------------------

// truncateNewTables is the cleanup recipe used by every new test
// below. It runs TRUNCATE on the 8 new tables in dependency-reversed
// order (children first); CASCADE handles the FK chain regardless of
// the order. The bookkeeping table is intentionally NOT truncated —
// it is append-only by design (see README §3) and the next test
// starts with its own resetSchemaMigrations call.
//
// Returns a func suitable for t.Cleanup so the call sites stay short.
func truncateNewTables(t *testing.T, db *sql.DB) func() {
	t.Helper()
	return func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx,
			"TRUNCATE TABLE spec_phase, spec, task, milestone, requirement_spike, requirement, project, organization CASCADE")
	}
}

// wipeNewTables drops the 8 new tables if they exist. Used at the
// top of each test to guarantee a clean state independent of any
// prior test run that may have left schema behind. Safe to call
// even when none of the tables exist yet (every DROP uses IF EXISTS).
func wipeNewTables(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stmts := []string{
		"DROP TABLE IF EXISTS spec_phase CASCADE",
		"DROP TABLE IF EXISTS spec CASCADE",
		"DROP TABLE IF EXISTS task CASCADE",
		"DROP TABLE IF EXISTS milestone CASCADE",
		"DROP TABLE IF EXISTS requirement_spike CASCADE",
		"DROP TABLE IF EXISTS requirement CASCADE",
		"DROP TABLE IF EXISTS project CASCADE",
		"DROP TABLE IF EXISTS organization CASCADE",
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("wipe (%s): %v", s, err)
		}
	}
}

// TestRunner_Up_AllNewMigrationsApply covers spec B1: a fresh
// `public.schema_migrations` plus a runner.Up() must produce 4
// bookkeeping rows in lexicographic order AND leave the 8 new tables
// in `public` AND accept each of the 8 locked `spec_phase.phase`
// values while rejecting an unknown phase.
func TestRunner_Up_AllNewMigrationsApply(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)
	wipeNewTables(t, db)
	t.Cleanup(truncateNewTables(t, db))

	r := newTestRunner(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	applied, err := r.Up(ctx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	if len(applied) != 4 {
		t.Fatalf("Up applied %d migrations, want 4 (got %+v)", len(applied), applied)
	}
	wantVersions := []int64{20260621120000, 20260622120000, 20260622120001, 20260622120002}
	for i, want := range wantVersions {
		if applied[i].ID != want {
			t.Errorf("applied[%d].ID = %d, want %d", i, applied[i].ID, want)
		}
	}

	// Every new table must exist in `public`.
	wantTables := []string{
		"organization", "project", "requirement", "requirement_spike",
		"milestone", "task", "spec", "spec_phase",
	}
	for _, name := range wantTables {
		var exists bool
		row := db.QueryRowContext(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables
                           WHERE table_schema = 'public' AND table_name = $1)`, name)
		if err := row.Scan(&exists); err != nil {
			t.Fatalf("scan table existence for %s: %v", name, err)
		}
		if !exists {
			t.Errorf("public.%s missing after Up()", name)
		}
	}

	// The 8 phase values must be accepted by spec_phase.phase_check;
	// one bogus value must be rejected. We seed a parent row per
	// insert so the FK is satisfied.
	phases := []string{
		"tdd_red", "implementation", "tdd_green", "verify",
		"pr", "technical_ai_review", "ai_approved", "human_approved",
	}
	// Seed one organization, project, requirement, milestone, task, spec.
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer setupCancel()
	var orgID, projID, reqID int64
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO organization (full_name, identification)
         VALUES ('Acme', 'RFC-TEST-001') RETURNING id`).Scan(&orgID); err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO project (organization_id, key, full_name)
         VALUES ($1, 'core', 'Core') RETURNING id`, orgID).Scan(&projID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO requirement (project_id, filename, content)
         VALUES ($1, 'r.md', 'body') RETURNING id`, projID).Scan(&reqID); err != nil {
		t.Fatalf("seed requirement: %v", err)
	}
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO milestone (requirement_id, title) VALUES ($1, 'M')`, reqID); err != nil {
		t.Fatalf("seed milestone: %v", err)
	}
	var taskID int64
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO task (milestone_id, title) VALUES ($1, 'T') RETURNING id`, reqID).Scan(&taskID); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	var specID int64
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO spec (task_id, content) VALUES ($1, 'body') RETURNING id`, taskID).Scan(&specID); err != nil {
		t.Fatalf("seed spec: %v", err)
	}

	for _, phase := range phases {
		// Each phase insert uses a slightly later started_at so the
		// natural-key UNIQUE (spec_id, phase, started_at) is satisfied
		// across the loop (only relevant for repeated phases, but
		// each phase is unique here so a uniform now() would also
		// work; using now() per insert keeps timestamps realistic).
		if _, err := db.ExecContext(setupCtx,
			`INSERT INTO spec_phase (spec_id, phase, started_at)
             VALUES ($1, $2, now())`, specID, phase); err != nil {
			t.Errorf("insert spec_phase with phase=%q: %v (must be accepted by spec_phase_phase_check)", phase, err)
		}
	}
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO spec_phase (spec_id, phase) VALUES ($1, 'NOT_A_PHASE')`, specID); err == nil {
		t.Errorf("insert spec_phase with phase='NOT_A_PHASE' must fail spec_phase_phase_check, got nil error")
	}
}

// TestRunner_Up_FKConstraintsApply covers spec P4: an INSERT into
// `project` whose `organization_id` does not reference an existing
// `organization.id` must fail with a foreign-key violation on
// `project_organization_id_fkey`.
func TestRunner_Up_FKConstraintsApply(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)
	wipeNewTables(t, db)
	t.Cleanup(truncateNewTables(t, db))

	r := newTestRunner(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	if _, err := db.ExecContext(ctx,
		`INSERT INTO project (organization_id, key, full_name) VALUES (99999, 'orphan', 'Orphan')`); err == nil {
		t.Fatalf("expected FK violation on project.organization_id=99999, got nil error")
	} else if !msgContainsAny(err.Error(),
		"project_organization_id_fkey", "foreign key") {
		t.Errorf("expected error mentioning project_organization_id_fkey, got: %v", err)
	}
}

// msgContainsAny returns true if needle appears anywhere in haystack.
// Mirrors the unexported runner.go helper; duplicated here so the
// test file stays self-contained.
func msgContainsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if len(n) == 0 {
			return true
		}
		for i := 0; i+len(n) <= len(haystack); i++ {
			if haystack[i:i+len(n)] == n {
				return true
			}
		}
	}
	return false
}

// TestRunner_Up_StrictInheritanceOnMilestone covers spec M1, M2, M3:
// `milestone.requirement_id` MUST be the ONLY PK column (no synthetic
// `id`); a second `INSERT INTO milestone` for the same
// `requirement_id` MUST fail with `milestone_pkey` violation.
func TestRunner_Up_StrictInheritanceOnMilestone(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)
	wipeNewTables(t, db)
	t.Cleanup(truncateNewTables(t, db))

	r := newTestRunner(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Seed org -> project -> requirement, then a first milestone.
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer setupCancel()
	var orgID, projID, reqID int64
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO organization (full_name, identification)
         VALUES ('Acme', 'RFC-TEST-002') RETURNING id`).Scan(&orgID); err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO project (organization_id, key, full_name)
         VALUES ($1, 'core', 'Core') RETURNING id`, orgID).Scan(&projID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO requirement (project_id, filename, content)
         VALUES ($1, 'r.md', 'body') RETURNING id`, projID).Scan(&reqID); err != nil {
		t.Fatalf("seed requirement: %v", err)
	}

	// PRIMARY KEY introspection: must list ONLY requirement_id.
	rows, err := db.QueryContext(setupCtx,
		`SELECT a.attname
           FROM pg_index i
           JOIN pg_attribute a
             ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
          WHERE i.indrelid = 'milestone'::regclass AND i.indisprimary`)
	if err != nil {
		t.Fatalf("query milestone PK: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var pkCols []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			t.Fatalf("scan PK col: %v", err)
		}
		pkCols = append(pkCols, c)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	if len(pkCols) != 1 || pkCols[0] != "requirement_id" {
		t.Errorf("milestone PK columns = %v, want exactly [requirement_id]", pkCols)
	}

	// Confirm no `id` column exists on milestone.
	var hasID bool
	if err := db.QueryRowContext(setupCtx,
		`SELECT EXISTS (
            SELECT 1 FROM information_schema.columns
             WHERE table_schema = 'public' AND table_name = 'milestone' AND column_name = 'id'
         )`).Scan(&hasID); err != nil {
		t.Fatalf("check milestone.id: %v", err)
	}
	if hasID {
		t.Errorf("milestone.id must NOT exist (strict inheritance) but column is present")
	}

	// First milestone succeeds.
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO milestone (requirement_id, title) VALUES ($1, 'M1')`, reqID); err != nil {
		t.Fatalf("first milestone insert: %v", err)
	}

	// Second milestone for same requirement_id must fail on
	// milestone_pkey.
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO milestone (requirement_id, title) VALUES ($1, 'M1 dup')`, reqID); err == nil {
		t.Errorf("expected milestone_pkey violation on duplicate requirement_id, got nil")
	} else if !msgContainsAny(err.Error(), "milestone_pkey", "duplicate key") {
		t.Errorf("expected error mentioning milestone_pkey, got: %v", err)
	}
}

// TestRunner_Up_AppendOnlyConvention covers spec O1/A1: the column
// comment on `organization.is_active` MUST explicitly declare it as
// the only UPDATE-in-place column on `organization` rows. The comment
// is the documented contract for code review (DB-level enforcement is
// deferred per design §6.2).
func TestRunner_Up_AppendOnlyConvention(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)
	wipeNewTables(t, db)
	t.Cleanup(truncateNewTables(t, db))

	r := newTestRunner(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	var comment string
	row := db.QueryRowContext(ctx,
		`SELECT col_description('organization'::regclass, attnum)
           FROM pg_attribute
          WHERE attrelid = 'organization'::regclass AND attname = 'is_active'`)
	if err := row.Scan(&comment); err != nil {
		t.Fatalf("scan is_active comment: %v", err)
	}
	want := "UPDATE-in-place: this column is the ONLY mutation allowed on organization rows."
	if comment != want {
		t.Errorf("organization.is_active comment = %q, want %q", comment, want)
	}
}

// TestRunner_Up_PRDSizeCapEnforced covers spec R1, R2: the
// `requirement_content_size_cap` CHECK MUST accept content of
// exactly 262144 bytes (boundary) AND reject 262145 bytes (over).
func TestRunner_Up_PRDSizeCapEnforced(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)
	wipeNewTables(t, db)
	t.Cleanup(truncateNewTables(t, db))

	r := newTestRunner(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Seed org -> project.
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer setupCancel()
	var orgID, projID int64
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO organization (full_name, identification)
         VALUES ('Acme', 'RFC-TEST-003') RETURNING id`).Scan(&orgID); err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO project (organization_id, key, full_name)
         VALUES ($1, 'core', 'Core') RETURNING id`, orgID).Scan(&projID); err != nil {
		t.Fatalf("seed project: %v", err)
	}

	// Boundary: 262144 bytes MUST succeed.
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO requirement (project_id, filename, content)
         VALUES ($1, 'edge.md', repeat('x', 262144))`, projID); err != nil {
		t.Errorf("262144-byte content (boundary) must succeed, got: %v", err)
	}

	// Over: 262145 bytes MUST fail on requirement_content_size_cap.
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO requirement (project_id, filename, content)
         VALUES ($1, 'big.md', repeat('x', 262145))`, projID); err == nil {
		t.Errorf("262145-byte content must fail requirement_content_size_cap, got nil")
	} else if !msgContainsAny(err.Error(), "requirement_content_size_cap", "check constraint") {
		t.Errorf("expected error mentioning requirement_content_size_cap, got: %v", err)
	}
}

// TestRunner_Up_SpecPhaseReEntry covers spec SP5, SP6: the
// agent-first re-entry pattern MUST work end-to-end against the
// live DB. Close current open phase, INSERT new phase (even an
// earlier one), the natural-key UNIQUE must still hold because
// the two rows have different `started_at`, and the partial index
// `idx_spec_phase_current_state` MUST return exactly one row per
// spec when querying `WHERE ended_at IS NULL`.
func TestRunner_Up_SpecPhaseReEntry(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)
	wipeNewTables(t, db)
	t.Cleanup(truncateNewTables(t, db))

	r := newTestRunner(db)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := r.Up(ctx); err != nil {
		t.Fatalf("Up: %v", err)
	}

	// Seed org -> project -> requirement -> milestone -> task -> spec.
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer setupCancel()
	var orgID, projID, reqID, taskID, specID int64
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO organization (full_name, identification)
         VALUES ('Acme', 'RFC-TEST-004') RETURNING id`).Scan(&orgID); err != nil {
		t.Fatalf("seed organization: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO project (organization_id, key, full_name)
         VALUES ($1, 'core', 'Core') RETURNING id`, orgID).Scan(&projID); err != nil {
		t.Fatalf("seed project: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO requirement (project_id, filename, content)
         VALUES ($1, 'r.md', 'body') RETURNING id`, projID).Scan(&reqID); err != nil {
		t.Fatalf("seed requirement: %v", err)
	}
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO milestone (requirement_id, title) VALUES ($1, 'M')`, reqID); err != nil {
		t.Fatalf("seed milestone: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO task (milestone_id, title) VALUES ($1, 'T') RETURNING id`, reqID).Scan(&taskID); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	if err := db.QueryRowContext(setupCtx,
		`INSERT INTO spec (task_id, content) VALUES ($1, 'body') RETURNING id`, taskID).Scan(&specID); err != nil {
		t.Fatalf("seed spec: %v", err)
	}

	// 1. Open tdd_red.
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO spec_phase (spec_id, phase, notes) VALUES ($1, 'tdd_red', 'initial red')`,
		specID); err != nil {
		t.Fatalf("insert tdd_red #1: %v", err)
	}
	// 2. Close it.
	if _, err := db.ExecContext(setupCtx,
		`UPDATE spec_phase SET ended_at = now() WHERE spec_id = $1 AND ended_at IS NULL`,
		specID); err != nil {
		t.Fatalf("close tdd_red: %v", err)
	}
	// 3. Open technical_ai_review.
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO spec_phase (spec_id, phase, notes) VALUES ($1, 'technical_ai_review', 'reviewing')`,
		specID); err != nil {
		t.Fatalf("insert technical_ai_review: %v", err)
	}
	// 4. Close it.
	if _, err := db.ExecContext(setupCtx,
		`UPDATE spec_phase SET ended_at = now() WHERE spec_id = $1 AND ended_at IS NULL`,
		specID); err != nil {
		t.Fatalf("close technical_ai_review: %v", err)
	}
	// Force a non-equal microsecond difference so the natural-key
	// UNIQUE (spec_id, phase, started_at) does not block the re-entry
	// when we re-insert tdd_red below.
	time.Sleep(10 * time.Millisecond)
	// 5. Re-enter tdd_red (agent first — see proposal §"Agent-first re-entry pattern").
	if _, err := db.ExecContext(setupCtx,
		`INSERT INTO spec_phase (spec_id, phase, notes) VALUES ($1, 'tdd_red', 're-entering after AI review found X')`,
		specID); err != nil {
		t.Fatalf("insert tdd_red #2 (re-entry): %v", err)
	}

	// 3 rows for this spec.
	var total int
	if err := db.QueryRowContext(setupCtx,
		`SELECT count(*) FROM spec_phase WHERE spec_id = $1`, specID).Scan(&total); err != nil {
		t.Fatalf("count spec_phase: %v", err)
	}
	if total != 3 {
		t.Errorf("spec_phase rows for spec_id=%d = %d, want 3", specID, total)
	}

	// Exactly one row is currently open (the second tdd_red).
	var openCount int
	if err := db.QueryRowContext(setupCtx,
		`SELECT count(*) FROM spec_phase WHERE spec_id = $1 AND ended_at IS NULL`,
		specID).Scan(&openCount); err != nil {
		t.Fatalf("count open spec_phase: %v", err)
	}
	if openCount != 1 {
		t.Errorf("open spec_phase rows = %d, want 1", openCount)
	}

	// The two tdd_red rows must have distinct started_at (the
	// natural-key UNIQUE enforced differentiation via a 10ms gap).
	var sameStartedAt int
	if err := db.QueryRowContext(setupCtx,
		`SELECT count(DISTINCT started_at) FROM spec_phase WHERE spec_id = $1 AND phase = 'tdd_red'`,
		specID).Scan(&sameStartedAt); err != nil {
		t.Fatalf("distinct started_at: %v", err)
	}
	if sameStartedAt != 2 {
		t.Errorf("distinct tdd_red started_at values = %d, want 2", sameStartedAt)
	}
}

// TestRunner_Up_LexicographicOrder_AllFourVersions covers spec B1
// and extends TestRunner_Up_LexicographicOrder to seed all 4
// bookkeeping versions and assert the runner applies them in
// lexicographic order regardless of the 1-second timestamp spacing.
// It is kept as a separate test to avoid mutating the existing one.
func TestRunner_Up_LexicographicOrder_AllFourVersions(t *testing.T) {
	db := integrationRunnerDB(t)
	resetSchemaMigrations(t, db)
	wipeNewTables(t, db)
	t.Cleanup(truncateNewTables(t, db))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE public.schema_migrations (
			id         BIGSERIAL    PRIMARY KEY,
			version_id BIGINT       NOT NULL,
			is_applied BOOLEAN      NOT NULL,
			tstamp     TIMESTAMPTZ  NOT NULL DEFAULT now()
		)
	`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}
	// Seed the lexicographically-earliest version as already-applied.
	if _, err := db.ExecContext(ctx,
		`INSERT INTO public.schema_migrations (version_id, is_applied) VALUES (20260101000000, TRUE)`); err != nil {
		t.Fatalf("seed older version row: %v", err)
	}

	r := newTestRunner(db)
	upCtx, upCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer upCancel()
	applied, err := r.Up(upCtx)
	if err != nil {
		t.Fatalf("Up: %v", err)
	}
	// After this Up, all 4 witsaba versions must be in the bookkeeping
	// table in chronological order.
	wantVersions := []int64{20260621120000, 20260622120000, 20260622120001, 20260622120002}
	if len(applied) != len(wantVersions) {
		t.Errorf("Up applied %d migrations, want %d (got %+v)", len(applied), len(wantVersions), applied)
	}
	for i, want := range wantVersions {
		if i < len(applied) && applied[i].ID != want {
			t.Errorf("applied[%d].ID = %d, want %d", i, applied[i].ID, want)
		}
	}
	// The older seeded row (20260101000000) must NOT have been
	// re-applied. Read every row by version_id and assert the set is
	// exactly the 5 expected values.
	rows, err := db.QueryContext(ctx,
		`SELECT version_id FROM public.schema_migrations ORDER BY version_id`)
	if err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	defer func() { _ = rows.Close() }()
	var got []int64
	for rows.Next() {
		var v int64
		if err := rows.Scan(&v); err != nil {
			t.Fatalf("scan version_id: %v", err)
		}
		got = append(got, v)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}
	wantSet := []int64{20260101000000, 20260621120000, 20260622120000, 20260622120001, 20260622120002}
	if len(got) != len(wantSet) {
		t.Errorf("public.schema_migrations has %d rows, want %d (got %v)", len(got), len(wantSet), got)
	} else {
		for i := range wantSet {
			if got[i] != wantSet[i] {
				t.Errorf("schema_migrations[%d] = %d, want %d", i, got[i], wantSet[i])
			}
		}
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
