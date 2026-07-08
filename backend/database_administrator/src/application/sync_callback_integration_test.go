// sync_callback_integration_test.go — integration tests for the
// ProcessSyncCallback production path (s.db != nil).
//
// These tests use a real Postgres instance (the same one the dev
// compose stack runs) to catch bugs that the unit-test path
// (`s.db == nil`, which uses fake repositories) cannot catch. The
// canonical example is the WHERE-filter bug discovered in UAT on
// 2026-07-08: the unit-test path bypasses the SQL UPDATE entirely,
// so the WHERE filter "WHERE status = 'running'" was never
// exercised. The production path was matching 0 rows because the
// row was always in 'pending' (the syncer never transitions it
// to 'running' before the callback). This file ensures the WHERE
// filter accepts the row in either 'pending' or 'running' state.
//
// Run with: INTEGRATION=1 go test ./src/application/...
package application

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// integrationTestDB opens a real Postgres connection using the
// same env vars the dev compose stack uses. Skips the test if
// INTEGRATION is not set (mirrors the runner_test.go pattern).
func integrationTestDB(t *testing.T) *sql.DB {
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
		t.Fatalf("ping compose postgres: %v (is docker compose up?)", err)
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

// wipeSyncFixtures deletes any existing rows from sync_job, the
// workspace table, and organization (the foreign-key CASCADE
// requires sync_job to be wiped first, then workspace, then
// organization). Safe to call from any test; runs the wipe
// synchronously and registers no cleanup so the test can keep
// working until its own t.Cleanup.
func wipeSyncFixtures(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for _, stmt := range []string{
		`DELETE FROM sync_job`,
		`DELETE FROM workspace`,
		`DELETE FROM organization`,
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("wipe %q: %v", stmt, err)
		}
	}
}

// insertWorkspace inserts one workspace row and returns the id.
// Tests use the row as the parent of a sync_job.
func insertWorkspace(t *testing.T, db *sql.DB, name string) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var orgID int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO organization (full_name, identification)
		VALUES ($1, $1)
		RETURNING id
	`, name).Scan(&orgID); err != nil {
		t.Fatalf("insert org: %v", err)
	}
	var id int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO workspace (organization_id, name, repo_github_id, repo_full_name, repo_owner, repo_name)
		VALUES ($1, $2, 1, 'octocat/hello', 'octocat', 'hello')
		RETURNING id
	`, orgID, name).Scan(&id); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	return id
}

// insertSyncJob inserts one sync_job row in the given status and
// returns the id. Tests use this to set up a specific pre-state
// (e.g., a 'pending' row as it exists when the syncer's callback
// arrives).
func insertSyncJob(t *testing.T, db *sql.DB, workspaceID int64, status, triggeredBy string) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var id int64
	if err := db.QueryRowContext(ctx, `
		INSERT INTO sync_job (workspace_id, status, triggered_by, attempts)
		VALUES ($1, $2, $3, 0)
		RETURNING id
	`, workspaceID, status, triggeredBy).Scan(&id); err != nil {
		t.Fatalf("insert sync_job: %v", err)
	}
	return id
}

// readSyncJob returns the current state of a sync_job row.
// Tests use this to assert post-conditions.
type syncJobRow struct {
	ID             int64
	WorkspaceID    int64
	Status         string
	CommitSHAAfter sql.NullString
	ErrorCode      sql.NullString
	ErrorMessage   sql.NullString
	FinishedAt     sql.NullTime
	Attempts       int
}

func readSyncJob(t *testing.T, db *sql.DB, id int64) syncJobRow {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var r syncJobRow
	if err := db.QueryRowContext(ctx, `
		SELECT id, workspace_id, status, commit_sha_after, error_code,
		       error_message, finished_at, attempts
		  FROM sync_job WHERE id = $1
	`, id).Scan(&r.ID, &r.WorkspaceID, &r.Status, &r.CommitSHAAfter, &r.ErrorCode, &r.ErrorMessage, &r.FinishedAt, &r.Attempts); err != nil {
		t.Fatalf("read sync_job %d: %v", id, err)
	}
	return r
}

type workspaceRow struct {
	LastSyncedAt       sql.NullTime
	LastSyncedCommitSHA sql.NullString
	DefaultBranch      sql.NullString
	LastSyncJobID      sql.NullInt64
}

func readWorkspace(t *testing.T, db *sql.DB, id int64) workspaceRow {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var r workspaceRow
	if err := db.QueryRowContext(ctx, `
		SELECT last_synced_at, last_synced_commit_sha, default_branch, last_sync_job_id
		  FROM workspace WHERE id = $1
	`, id).Scan(&r.LastSyncedAt, &r.LastSyncedCommitSHA, &r.DefaultBranch, &r.LastSyncJobID); err != nil {
		t.Fatalf("read workspace %d: %v", id, err)
	}
	return r
}

// TestProcessSyncCallback_PendingRowTransitionsToDone is the
// regression test for the WHERE-filter bug discovered in UAT on
// 2026-07-08. The previous WHERE filter was `AND status = 'running'`,
// but the syncer never transitions a row to 'running' before
// the callback arrives (the row is in 'pending' the entire time
// the clone runs). The UPDATE matched 0 rows and the handler
// returned 204 without updating the row, so the sync_job stayed
// in 'pending' forever.
//
// The fix: the WHERE filter must accept the row in either
// 'pending' or 'running' state. The single-flight invariant is
// guaranteed by the partial unique index
// `sync_job_single_flight_uidx` on
// (workspace_id) WHERE status IN ('pending','running'), not by
// the callback's WHERE filter.
//
// This test pins the contract: a row in 'pending' state must
// transition to 'done' on a 'done' callback, with the
// commit_sha_after + finished_at + attempts denormalized onto
// the row and the workspace's last_synced_* columns denormalized
// onto the parent workspace.
func TestProcessSyncCallback_PendingRowTransitionsToDone(t *testing.T) {
	db := integrationTestDB(t)
	wipeSyncFixtures(t, db)
	workspaceID := insertWorkspace(t, db, "pending-test")
	jobID := insertSyncJob(t, db, workspaceID, "pending", "manual")

	svc := NewSyncService(nil, nil, db, nil)
	const wantSHA = "ec8fbc8a0d341de02e8b669cb9d628289245e3bf"
	const wantBranch = "main"
	updated, err := svc.ProcessSyncCallback(context.Background(), jobID, "done", wantSHA, wantBranch, "", "")
	if err != nil {
		t.Fatalf("ProcessSyncCallback: %v", err)
	}
	if updated == nil {
		t.Fatalf("ProcessSyncCallback returned nil updated job")
	}
	if updated.Status != "done" {
		t.Errorf("updated.Status = %q, want done", updated.Status)
	}

	// Reload from the DB to verify the row was actually written
	// (not just the in-memory state of the returned struct).
	got := readSyncJob(t, db, jobID)
	if got.Status != "done" {
		t.Errorf("sync_job.status in DB = %q, want done (REGRESSION: the WHERE filter rejected the row because the previous code required status='running')", got.Status)
	}
	if !got.CommitSHAAfter.Valid || got.CommitSHAAfter.String != wantSHA {
		t.Errorf("commit_sha_after = %v, want %q", got.CommitSHAAfter, wantSHA)
	}
	if !got.FinishedAt.Valid {
		t.Errorf("finished_at is NULL, want a timestamp")
	}
	if got.Attempts != 1 {
		t.Errorf("attempts = %d, want 1 (callback increments)", got.Attempts)
	}

	// Workspace denormalization.
	ws := readWorkspace(t, db, workspaceID)
	if !ws.LastSyncedAt.Valid {
		t.Errorf("workspace.last_synced_at is NULL, want a timestamp (REGRESSION: the denormalization only runs after the sync_job UPDATE; the WHERE bug means it never ran)")
	}
	if !ws.LastSyncedCommitSHA.Valid || ws.LastSyncedCommitSHA.String != wantSHA {
		t.Errorf("workspace.last_synced_commit_sha = %v, want %q", ws.LastSyncedCommitSHA, wantSHA)
	}
	if !ws.DefaultBranch.Valid || ws.DefaultBranch.String != wantBranch {
		t.Errorf("workspace.default_branch = %v, want %q", ws.DefaultBranch, wantBranch)
	}
	if !ws.LastSyncJobID.Valid || ws.LastSyncJobID.Int64 != jobID {
		t.Errorf("workspace.last_sync_job_id = %v, want %d", ws.LastSyncJobID, jobID)
	}
}

// TestProcessSyncCallback_RunningRowTransitionsToDone is the
// companion regression test: the WHERE filter must ALSO accept
// rows in 'running' state (a future state where the syncer
// reports a 'started' callback before the worktree probe).
// Both 'pending' and 'running' are valid pre-states.
func TestProcessSyncCallback_RunningRowTransitionsToDone(t *testing.T) {
	db := integrationTestDB(t)
	wipeSyncFixtures(t, db)
	workspaceID := insertWorkspace(t, db, "running-test")
	jobID := insertSyncJob(t, db, workspaceID, "running", "manual")

	svc := NewSyncService(nil, nil, db, nil)
	const wantSHA = "111222333444555666777888999000aaabbbcccddd"
	updated, err := svc.ProcessSyncCallback(context.Background(), jobID, "done", wantSHA, "develop", "", "")
	if err != nil {
		t.Fatalf("ProcessSyncCallback: %v", err)
	}
	if updated.Status != "done" {
		t.Errorf("updated.Status = %q, want done", updated.Status)
	}

	got := readSyncJob(t, db, jobID)
	if got.Status != "done" {
		t.Errorf("sync_job.status in DB = %q, want done", got.Status)
	}
	if !got.CommitSHAAfter.Valid || got.CommitSHAAfter.String != wantSHA {
		t.Errorf("commit_sha_after = %v, want %q", got.CommitSHAAfter, wantSHA)
	}
}

// TestProcessSyncCallback_FailedCallbackTransitionsToFailed pins
// the failure path. A 'failed' callback with error_code +
// error_message must transition the row to 'failed', set the
// error columns, and leave commit_sha_after NULL.
func TestProcessSyncCallback_FailedCallbackTransitionsToFailed(t *testing.T) {
	db := integrationTestDB(t)
	wipeSyncFixtures(t, db)
	workspaceID := insertWorkspace(t, db, "failed-test")
	jobID := insertSyncJob(t, db, workspaceID, "pending", "manual")

	svc := NewSyncService(nil, nil, db, nil)
	updated, err := svc.ProcessSyncCallback(context.Background(), jobID, "failed", "", "main", "CLONE_FAILED", "remote: Repository not found")
	if err != nil {
		t.Fatalf("ProcessSyncCallback: %v", err)
	}
	if updated.Status != "failed" {
		t.Errorf("updated.Status = %q, want failed", updated.Status)
	}

	got := readSyncJob(t, db, jobID)
	if got.Status != "failed" {
		t.Errorf("sync_job.status = %q, want failed", got.Status)
	}
	if !got.ErrorCode.Valid || got.ErrorCode.String != "CLONE_FAILED" {
		t.Errorf("error_code = %v, want CLONE_FAILED", got.ErrorCode)
	}
	if !got.ErrorMessage.Valid || got.ErrorMessage.String != "remote: Repository not found" {
		t.Errorf("error_message = %v, want the actual message", got.ErrorMessage)
	}
	if got.CommitSHAAfter.Valid {
		t.Errorf("commit_sha_after = %v, want NULL on failed", got.CommitSHAAfter)
	}
	if !got.FinishedAt.Valid {
		t.Errorf("finished_at is NULL, want a timestamp")
	}

	// Workspace denormalization does NOT run on failed.
	ws := readWorkspace(t, db, workspaceID)
	if ws.LastSyncedAt.Valid {
		t.Errorf("workspace.last_synced_at = %v, want NULL on failed (only 'done' denormalizes)", ws.LastSyncedAt)
	}
}

// TestProcessSyncCallback_DoneWithoutCommitSHA_Rejected pins the
// validation: a 'done' callback with no commit_sha is a 422 from
// the handler. This is unrelated to the WHERE-filter bug but
// lives in the same function and should be regression-tested
// alongside the others.
func TestProcessSyncCallback_DoneWithoutCommitSHA_Rejected(t *testing.T) {
	db := integrationTestDB(t)
	wipeSyncFixtures(t, db)
	workspaceID := insertWorkspace(t, db, "no-sha-test")
	jobID := insertSyncJob(t, db, workspaceID, "pending", "manual")

	svc := NewSyncService(nil, nil, db, nil)
	_, err := svc.ProcessSyncCallback(context.Background(), jobID, "done", "", "main", "", "")
	if err == nil {
		t.Fatalf("ProcessSyncCallback: expected ValidationError, got nil")
	}
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Errorf("error type = %T, want *ValidationError", err)
	}

	// Row was NOT updated.
	got := readSyncJob(t, db, jobID)
	if got.Status != "pending" {
		t.Errorf("sync_job.status = %q, want pending (validation rejection must not modify the row)", got.Status)
	}
}
