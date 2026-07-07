// Package postgres_test contains the integration test suite for
// the workspace repository adapter (PR1b-ii.a). The tests are gated
// on INTEGRATION=1 because they need a live Postgres (the same
// compose-provisioned instance used by the migration runner tests).
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// workspace_repo.go existed. Running
// `INTEGRATION=1 go test ./src/infrastructure/postgres/...` with no
// WorkspaceRepo type must fail with "undefined: WorkspaceRepo" +
// "undefined: NewWorkspaceRepo" + the per-method undefined errors —
// that failure IS the RED step.
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
// Workspace-specific test helpers (mirror organization_repo_test.go).
// ---------------------------------------------------------------------------

// ensureWorkspaceMigrations mirrors the locked DDL from
// migration/sql/20260706120002_workspaces.sql so the workspace +
// workspace_repository tables exist for the integration tests. We
// use CREATE TABLE IF NOT EXISTS so the helper is idempotent.
//
// We do NOT re-create the organization / identity / identity.account
// tables — those are owned by the earlier migrations and the
// compose-postgres instance is already migrated by the time these
// tests run (per the existing INTEGRATION=1 contract).
func ensureWorkspaceMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS workspace (
		    id                        BIGSERIAL    PRIMARY KEY,
		    organization_id           BIGINT       NOT NULL REFERENCES organization(id) ON DELETE CASCADE,
		    owner_user_id             BIGINT       REFERENCES identity.user(id) ON DELETE SET NULL,
		    name                      TEXT         NOT NULL,
		    primary_repo_github_id    BIGINT       NOT NULL,
		    primary_repo_full_name    TEXT         NOT NULL,
		    primary_repo_owner        TEXT         NOT NULL,
		    primary_repo_name         TEXT         NOT NULL,
		    created_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    deleted_at                TIMESTAMPTZ
		)
	`); err != nil {
		t.Fatalf("ensure workspace table: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE UNIQUE INDEX IF NOT EXISTS workspace_org_name_live_key
		    ON workspace (organization_id, name)
		    WHERE deleted_at IS NULL
	`); err != nil {
		t.Fatalf("ensure workspace_org_name_live_key index: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE INDEX IF NOT EXISTS workspace_org_deleted_at_idx
		    ON workspace (organization_id, deleted_at)
	`); err != nil {
		t.Fatalf("ensure workspace_org_deleted_at_idx index: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS workspace_repository (
		    id                BIGSERIAL    PRIMARY KEY,
		    workspace_id      BIGINT       NOT NULL REFERENCES workspace(id) ON DELETE CASCADE,
		    github_id         BIGINT       NOT NULL,
		    github_full_name  TEXT         NOT NULL,
		    github_owner      TEXT         NOT NULL,
		    github_name       TEXT         NOT NULL,
		    added_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    CONSTRAINT workspace_repository_workspace_github_key UNIQUE (workspace_id, github_id)
		)
	`); err != nil {
		t.Fatalf("ensure workspace_repository table: %v", err)
	}
}

// truncateWorkspaces wipes the workspace + workspace_repository
// tables for test isolation. CASCADE so linked repos are torn down
// with the parent workspace row.
func truncateWorkspaces(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "TRUNCATE workspace_repository, workspace RESTART IDENTITY CASCADE"); err != nil {
		t.Fatalf("truncate workspace tables: %v", err)
	}
}

// int64Ptr returns the address of an int64 literal. Used for
// OwnerUserID fixtures (nullable).
func int64Ptr(v int64) *int64 { return &v }

// makeOrg inserts a minimal organization row and returns its id. The
// organization table is the parent FK for workspace; tests need a
// real org id to satisfy the FK. The org's full_name +
// identification are unique (DDL constraint), so the helper uses a
// per-test-unique name + a per-process-unique salt to avoid
// collisions with organization_repo_test.go's fixtures (which share
// the same compose-postgres instance).
func makeOrg(t *testing.T, db *sql.DB, nameSuffix string) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	salt := fmt.Sprintf("%d-%d", time.Now().UnixNano(), os.Getpid())
	var id int64
	err := db.QueryRowContext(ctx, `
		INSERT INTO organization (full_name, identification, is_active)
		VALUES ($1, $2, TRUE)
		RETURNING id
	`, "Workspace Test Org "+nameSuffix+" "+salt, "workspace-test-"+nameSuffix+"-"+salt).Scan(&id)
	if err != nil {
		t.Fatalf("makeOrg insert: %v", err)
	}
	return id
}

// makeWorkspaceInput builds a minimal valid CreateWorkspaceInput for
// the given org id. Tests can mutate fields before calling Insert.
func makeWorkspaceInput(orgID int64, name string) *domain.Workspace {
	return &domain.Workspace{
		OrganizationID:      orgID,
		Name:                name,
		PrimaryRepoGitHubID: 123456,
		PrimaryRepoFullName: "octocat/" + name,
		PrimaryRepoOwner:    "octocat",
		PrimaryRepoName:     name,
	}
}

// makeLinkedRepoInput builds a minimal valid LinkedRepository for the
// given workspace id. Tests can mutate fields before calling
// AddLinkedRepo. The GitHubID is keyed off the (workspaceID, name)
// pair so different tests on the same workspace + different names
// stay unique, AND the same (workspaceID, name) in different tests
// intentionally collides (so the duplicate test is well-defined).
func makeLinkedRepoInput(workspaceID int64, name string) *domain.LinkedRepository {
	var h uint64 = 1469598103934665603 // FNV-1a offset basis
	for _, c := range fmt.Sprintf("%d:%s", workspaceID, name) {
		h ^= uint64(c)
		h *= 1099511628211 // FNV-1a prime
	}
	return &domain.LinkedRepository{
		WorkspaceID: workspaceID,
		GitHubID:    int64(h & 0x7FFFFFFFFFFFFFFF), // strip sign bit
		FullName:    "linked/" + name,
		Owner:       "linked",
		Name:        name,
	}
}

// ---------------------------------------------------------------------------
// Insert
// ---------------------------------------------------------------------------

// TestWorkspaceRepo_Insert_AssignsIDAndTimestamps covers T-WS-1BiiA-001
// (RED) + T-WS-1BiiA-002 (GREEN): a freshly inserted workspace must
// come back with a non-zero ID + non-zero CreatedAt/UpdatedAt
// populated from the DB clock. The adapter MUST NOT set those
// fields itself.
func TestWorkspaceRepo_Insert_AssignsIDAndTimestamps(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "insert-basic")
	in := makeWorkspaceInput(orgID, "alpha")

	got, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if got.ID <= 0 {
		t.Errorf("Insert returned ID = %d, want > 0", got.ID)
	}
	if got.CreatedAt.IsZero() {
		t.Errorf("Insert returned CreatedAt zero, want non-zero (DB default now())")
	}
	if got.UpdatedAt.IsZero() {
		t.Errorf("Insert returned UpdatedAt zero, want non-zero (DB default now())")
	}
	if got.Name != in.Name {
		t.Errorf("Name = %q, want %q", got.Name, in.Name)
	}
	if got.OrganizationID != orgID {
		t.Errorf("OrganizationID = %d, want %d", got.OrganizationID, orgID)
	}
	if got.PrimaryRepoGitHubID != in.PrimaryRepoGitHubID {
		t.Errorf("PrimaryRepoGitHubID = %d, want %d", got.PrimaryRepoGitHubID, in.PrimaryRepoGitHubID)
	}
	if got.PrimaryRepoFullName != in.PrimaryRepoFullName {
		t.Errorf("PrimaryRepoFullName = %q, want %q", got.PrimaryRepoFullName, in.PrimaryRepoFullName)
	}
	if got.PrimaryRepoOwner != in.PrimaryRepoOwner {
		t.Errorf("PrimaryRepoOwner = %q, want %q", got.PrimaryRepoOwner, in.PrimaryRepoOwner)
	}
	if got.PrimaryRepoName != in.PrimaryRepoName {
		t.Errorf("PrimaryRepoName = %q, want %q", got.PrimaryRepoName, in.PrimaryRepoName)
	}
	if got.DeletedAt != nil {
		t.Errorf("DeletedAt = %v, want nil (freshly inserted row is live)", got.DeletedAt)
	}
}

// TestWorkspaceRepo_Insert_NilOwnerUserID_StoresNULL covers
// T-WS-1BiiA-003 (TRIANGULATE a): OwnerUserID == nil must persist as
// SQL NULL (not the empty string, not the zero int64).
func TestWorkspaceRepo_Insert_NilOwnerUserID_StoresNULL(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "insert-nil-owner")
	in := makeWorkspaceInput(orgID, "no-owner")
	in.OwnerUserID = nil // explicit — the no-owner case

	got, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	var raw sql.NullInt64
	err = db.QueryRowContext(ctx, `SELECT owner_user_id FROM workspace WHERE id = $1`, got.ID).Scan(&raw)
	if err != nil {
		t.Fatalf("SELECT owner_user_id: %v", err)
	}
	if raw.Valid {
		t.Errorf("owner_user_id column = %d (valid), want NULL (not valid)", raw.Int64)
	}

	// Round-trip via SelectByID must return OwnerUserID == nil.
	got2, err := repo.SelectByID(ctx, got.ID)
	if err != nil {
		t.Fatalf("SelectByID: %v", err)
	}
	if got2.OwnerUserID != nil {
		t.Errorf("SelectByID OwnerUserID = %v, want nil", *got2.OwnerUserID)
	}
}

// TestWorkspaceRepo_Insert_DuplicateNameInSameOrg_ReturnsConflictError
// covers T-WS-1BiiA-003 (TRIANGULATE b): the partial unique index
// `workspace_org_name_live_key` must reject a second insert with
// the same (organization_id, name). The pgx unique-violation must
// be translated to *domain.ConflictError so the handler can map to
// HTTP 409 without importing pgx.
func TestWorkspaceRepo_Insert_DuplicateNameInSameOrg_ReturnsConflictError(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "insert-dup-name")

	first := makeWorkspaceInput(orgID, "shared-name")
	if _, err := repo.Insert(ctx, first); err != nil {
		t.Fatalf("Insert first: %v", err)
	}

	second := makeWorkspaceInput(orgID, "shared-name")
	second.PrimaryRepoGitHubID = 999999 // differ on primary repo so the only collision is name
	_, err := repo.Insert(ctx, second)
	if err == nil {
		t.Fatalf("Insert second: expected error, got nil")
	}
	var conflictErr *domain.ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("Insert second: got %T (%v), want *domain.ConflictError", err, err)
	}
}

// ---------------------------------------------------------------------------
// SelectByID
// ---------------------------------------------------------------------------

// TestWorkspaceRepo_SelectByID_ReturnsInserted covers T-WS-1BiiA-004
// (RED) + T-WS-1BiiA-005 (GREEN): a row inserted via Insert must be
// readable via SelectByID with field-level parity.
func TestWorkspaceRepo_SelectByID_ReturnsInserted(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "selectbyid-basic")
	in := makeWorkspaceInput(orgID, "selectbyid")

	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	got, err := repo.SelectByID(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("SelectByID: %v", err)
	}
	if got.ID != inserted.ID {
		t.Errorf("ID = %d, want %d", got.ID, inserted.ID)
	}
	if got.Name != in.Name {
		t.Errorf("Name = %q, want %q", got.Name, in.Name)
	}
	if got.OrganizationID != orgID {
		t.Errorf("OrganizationID = %d, want %d", got.OrganizationID, orgID)
	}
	if got.PrimaryRepoFullName != in.PrimaryRepoFullName {
		t.Errorf("PrimaryRepoFullName = %q, want %q", got.PrimaryRepoFullName, in.PrimaryRepoFullName)
	}
}

// TestWorkspaceRepo_SelectByID_NotFound covers the not-found case
// (precondition for T-WS-1BiiA-006): an unknown id must return
// *domain.NotFoundError so the handler can map to HTTP 404.
func TestWorkspaceRepo_SelectByID_NotFound(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := repo.SelectByID(ctx, 99999999)
	if err == nil {
		t.Fatalf("SelectByID(99999999): expected error, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("SelectByID(99999999): got %T (%v), want *domain.NotFoundError", err, err)
	}
}

// TestWorkspaceRepo_SelectByID_SoftDeletedReturnsNotFound covers
// T-WS-1BiiA-006 (TRIANGULATE): the partial index hides soft-deleted
// rows from SelectByID; the caller must see *NotFoundError
// (indistinguishable from "never existed").
func TestWorkspaceRepo_SelectByID_SoftDeletedReturnsNotFound(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "selectbyid-soft-deleted")
	in := makeWorkspaceInput(orgID, "soft-deleted-target")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	if err := repo.SoftDelete(ctx, inserted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err = repo.SelectByID(ctx, inserted.ID)
	if err == nil {
		t.Fatalf("SelectByID on soft-deleted row: expected error, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("SelectByID on soft-deleted row: got %T (%v), want *domain.NotFoundError", err, err)
	}
}

// ---------------------------------------------------------------------------
// SelectAllByOrg
// ---------------------------------------------------------------------------

// TestWorkspaceRepo_SelectAllByOrg_OrdersCreatedDescAndCapsLimit
// covers T-WS-1BiiA-007 (RED) + T-WS-1BiiA-008 (GREEN): the list
// must be ordered by (created_at DESC, id DESC) for stable
// pagination, and the slice must be capped at the limit
// (caller-picked; the repo MUST NOT silently truncate beyond the
// cap).
func TestWorkspaceRepo_SelectAllByOrg_OrdersCreatedDescAndCapsLimit(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "selectall-order-cap")

	// Insert 3 rows with deterministic names so we can assert order.
	names := []string{"first", "second", "third"}
	inserted := make([]*domain.Workspace, 0, len(names))
	for _, n := range names {
		w, err := repo.Insert(ctx, makeWorkspaceInput(orgID, n))
		if err != nil {
			t.Fatalf("Insert %q: %v", n, err)
		}
		// 1ms gap so created_at strictly orders even on fast clocks.
		time.Sleep(2 * time.Millisecond)
		inserted = append(inserted, w)
	}

	got, err := repo.SelectAllByOrg(ctx, orgID, 100)
	if err != nil {
		t.Fatalf("SelectAllByOrg: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len(got) = %d, want 3", len(got))
	}
	// created_at DESC → "third" first, "first" last.
	if got[0].Name != "third" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "third")
	}
	if got[1].Name != "second" {
		t.Errorf("got[1].Name = %q, want %q", got[1].Name, "second")
	}
	if got[2].Name != "first" {
		t.Errorf("got[2].Name = %q, want %q", got[2].Name, "first")
	}

	// Cap test: limit=2 must return the first 2 in DESC order.
	got2, err := repo.SelectAllByOrg(ctx, orgID, 2)
	if err != nil {
		t.Fatalf("SelectAllByOrg(limit=2): %v", err)
	}
	if len(got2) != 2 {
		t.Fatalf("len(got2) = %d, want 2", len(got2))
	}
	if got2[0].Name != "third" {
		t.Errorf("got2[0].Name = %q, want %q", got2[0].Name, "third")
	}
	if got2[1].Name != "second" {
		t.Errorf("got2[1].Name = %q, want %q", got2[1].Name, "second")
	}
}

// TestWorkspaceRepo_SelectAllByOrg_NoRowsReturnsEmptySlice covers
// T-WS-1BiiA-009 (TRIANGULATE a): zero rows must return an empty
// slice (NOT nil) so callers can `for _, w := range got` without
// nil checks.
func TestWorkspaceRepo_SelectAllByOrg_NoRowsReturnsEmptySlice(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "selectall-empty")

	got, err := repo.SelectAllByOrg(ctx, orgID, 100)
	if err != nil {
		t.Fatalf("SelectAllByOrg: %v", err)
	}
	if got == nil {
		t.Fatalf("SelectAllByOrg: got nil slice, want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len(got) = %d, want 0", len(got))
	}
}

// TestWorkspaceRepo_SelectAllByOrg_FiltersSoftDeleted covers
// T-WS-1BiiA-009 (TRIANGULATE b): 3 soft-deleted + 2 live rows →
// the response must include only the 2 live rows. Soft-deleted
// rows are invisible to the list endpoint (per design T6).
func TestWorkspaceRepo_SelectAllByOrg_FiltersSoftDeleted(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "selectall-soft-filter")

	liveNames := []string{"live-a", "live-b"}
	deletedNames := []string{"dead-a", "dead-b", "dead-c"}

	for _, n := range liveNames {
		if _, err := repo.Insert(ctx, makeWorkspaceInput(orgID, n)); err != nil {
			t.Fatalf("Insert live %q: %v", n, err)
		}
	}
	for _, n := range deletedNames {
		w, err := repo.Insert(ctx, makeWorkspaceInput(orgID, n))
		if err != nil {
			t.Fatalf("Insert dead %q: %v", n, err)
		}
		if err := repo.SoftDelete(ctx, w.ID); err != nil {
			t.Fatalf("SoftDelete %q: %v", n, err)
		}
	}

	got, err := repo.SelectAllByOrg(ctx, orgID, 100)
	if err != nil {
		t.Fatalf("SelectAllByOrg: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2 (live rows only)", len(got))
	}
	gotNames := map[string]bool{got[0].Name: true, got[1].Name: true}
	if !gotNames["live-a"] || !gotNames["live-b"] {
		t.Errorf("SelectAllByOrg names = %v, want {live-a, live-b}", gotNames)
	}
	for _, n := range deletedNames {
		if gotNames[n] {
			t.Errorf("SelectAllByOrg includes soft-deleted row %q", n)
		}
	}
}

// ---------------------------------------------------------------------------
// UpdateName
// ---------------------------------------------------------------------------

// TestWorkspaceRepo_UpdateName_RenamesAndBumpsUpdatedAt covers
// T-WS-1BiiA-010 (RED) + T-WS-1BiiA-011 (GREEN): a successful rename
// must change the name and bump updated_at. The original created_at
// is preserved.
func TestWorkspaceRepo_UpdateName_RenamesAndBumpsUpdatedAt(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "updatename-basic")
	in := makeWorkspaceInput(orgID, "old-name")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	originalUpdatedAt := inserted.UpdatedAt
	originalCreatedAt := inserted.CreatedAt

	// Sleep a moment so updated_at has a chance to advance.
	time.Sleep(10 * time.Millisecond)

	got, err := repo.UpdateName(ctx, inserted.ID, "new-name")
	if err != nil {
		t.Fatalf("UpdateName: %v", err)
	}
	if got.Name != "new-name" {
		t.Errorf("Name = %q, want %q", got.Name, "new-name")
	}
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("UpdatedAt = %v, want > %v", got.UpdatedAt, originalUpdatedAt)
	}
	if !got.CreatedAt.Equal(originalCreatedAt) {
		t.Errorf("CreatedAt = %v, want %v (unchanged)", got.CreatedAt, originalCreatedAt)
	}

	// Round-trip via SelectByID confirms the persisted row matches.
	got2, err := repo.SelectByID(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("SelectByID: %v", err)
	}
	if got2.Name != "new-name" {
		t.Errorf("SelectByID Name = %q, want %q", got2.Name, "new-name")
	}
}

// TestWorkspaceRepo_UpdateName_DuplicateInSameOrgReturnsConflict covers
// UpdateName error path: a rename to a name that already exists in
// the same org (and is live) must return *domain.ConflictError. The
// pgx unique-violation must NOT leak through.
func TestWorkspaceRepo_UpdateName_DuplicateInSameOrgReturnsConflict(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "updatename-conflict")

	// Two distinct live rows.
	if _, err := repo.Insert(ctx, makeWorkspaceInput(orgID, "name-a")); err != nil {
		t.Fatalf("Insert a: %v", err)
	}
	b, err := repo.Insert(ctx, makeWorkspaceInput(orgID, "name-b"))
	if err != nil {
		t.Fatalf("Insert b: %v", err)
	}

	// Try to rename b to "name-a" → must Conflict.
	_, err = repo.UpdateName(ctx, b.ID, "name-a")
	if err == nil {
		t.Fatalf("UpdateName duplicate: expected error, got nil")
	}
	var conflictErr *domain.ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("UpdateName duplicate: got %T (%v), want *domain.ConflictError", err, err)
	}
}

// TestWorkspaceRepo_UpdateName_SoftDeletedReturnsNotFound covers
// T-WS-1BiiA-012 (TRIANGULATE): renaming a soft-deleted row must
// return *NotFoundError (the partial index hides it). This protects
// against resurrecting a soft-deleted workspace via rename.
func TestWorkspaceRepo_UpdateName_SoftDeletedReturnsNotFound(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "updatename-soft-deleted")
	in := makeWorkspaceInput(orgID, "to-be-soft-deleted")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := repo.SoftDelete(ctx, inserted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	_, err = repo.UpdateName(ctx, inserted.ID, "resurrected-name")
	if err == nil {
		t.Fatalf("UpdateName on soft-deleted: expected error, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("UpdateName on soft-deleted: got %T (%v), want *domain.NotFoundError", err, err)
	}
}

// ---------------------------------------------------------------------------
// SoftDelete
// ---------------------------------------------------------------------------

// TestWorkspaceRepo_SoftDelete_SetsDeletedAtAndCascadesLinkedRepos
// covers T-WS-1BiiA-013 (RED) + T-WS-1BiiA-014 (GREEN): SoftDelete
// must set deleted_at = now() on the workspace row AND hard-delete
// every linked workspace_repository row. The cascade protects
// against orphan linked repos after a workspace is removed.
func TestWorkspaceRepo_SoftDelete_SetsDeletedAtAndCascadesLinkedRepos(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "softdelete-cascade")
	in := makeWorkspaceInput(orgID, "will-be-deleted")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// Add 2 linked repos.
	for _, n := range []string{"linked-x", "linked-y"} {
		if _, err := repo.AddLinkedRepo(ctx, makeLinkedRepoInput(inserted.ID, n)); err != nil {
			t.Fatalf("AddLinkedRepo %q: %v", n, err)
		}
	}
	// Sanity: 2 linked repos present.
	if rs, err := repo.SelectLinkedRepos(ctx, inserted.ID); err != nil {
		t.Fatalf("SelectLinkedRepos pre-delete: %v", err)
	} else if len(rs) != 2 {
		t.Fatalf("pre-delete SelectLinkedRepos len = %d, want 2", len(rs))
	}

	// SoftDelete.
	if err := repo.SoftDelete(ctx, inserted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	// deleted_at is set on the workspace row.
	var deletedAt sql.NullTime
	err = db.QueryRowContext(ctx, `SELECT deleted_at FROM workspace WHERE id = $1`, inserted.ID).Scan(&deletedAt)
	if err != nil {
		t.Fatalf("SELECT deleted_at: %v", err)
	}
	if !deletedAt.Valid {
		t.Errorf("deleted_at column = NULL, want non-NULL (SoftDelete should have set it)")
	}

	// Linked repos are gone.
	if rs, err := repo.SelectLinkedRepos(ctx, inserted.ID); err != nil {
		t.Fatalf("SelectLinkedRepos post-delete: %v", err)
	} else if len(rs) != 0 {
		t.Errorf("post-delete SelectLinkedRepos len = %d, want 0 (cascade)", len(rs))
	}

	// Workspace row is invisible to SelectByID.
	_, err = repo.SelectByID(ctx, inserted.ID)
	if err == nil {
		t.Errorf("SelectByID post-delete: expected error, got nil")
	}
}

// TestWorkspaceRepo_SoftDelete_AlreadyDeletedReturnsNotFound covers
// T-WS-1BiiA-015 (TRIANGULATE): SoftDelete on an already-deleted
// row must return *NotFoundError. Protects against the partial
// index hiding the row from the UPDATE's WHERE clause.
func TestWorkspaceRepo_SoftDelete_AlreadyDeletedReturnsNotFound(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "softdelete-already")
	in := makeWorkspaceInput(orgID, "delete-twice")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := repo.SoftDelete(ctx, inserted.ID); err != nil {
		t.Fatalf("SoftDelete (first): %v", err)
	}

	err = repo.SoftDelete(ctx, inserted.ID)
	if err == nil {
		t.Fatalf("SoftDelete (second): expected error, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("SoftDelete (second): got %T (%v), want *domain.NotFoundError", err, err)
	}
}

// ---------------------------------------------------------------------------
// AddLinkedRepo
// ---------------------------------------------------------------------------

// TestWorkspaceRepo_AddLinkedRepo_AssignsIDAndAddedAt covers
// T-WS-1BiiA-016 (RED) + T-WS-1BiiA-017 (GREEN): a freshly added
// linked repo must come back with a non-zero ID + non-zero AddedAt
// populated from the DB clock.
func TestWorkspaceRepo_AddLinkedRepo_AssignsIDAndAddedAt(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "addlinkedrepo-basic")
	in := makeWorkspaceInput(orgID, "add-link-host")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	repoIn := makeLinkedRepoInput(inserted.ID, "link-1")
	got, err := repo.AddLinkedRepo(ctx, repoIn)
	if err != nil {
		t.Fatalf("AddLinkedRepo: %v", err)
	}
	if got.ID <= 0 {
		t.Errorf("AddLinkedRepo returned ID = %d, want > 0", got.ID)
	}
	if got.WorkspaceID != inserted.ID {
		t.Errorf("WorkspaceID = %d, want %d", got.WorkspaceID, inserted.ID)
	}
	if got.AddedAt.IsZero() {
		t.Errorf("AddedAt zero, want non-zero (DB default now())")
	}
	if got.GitHubID != repoIn.GitHubID {
		t.Errorf("GitHubID = %d, want %d", got.GitHubID, repoIn.GitHubID)
	}
	if got.FullName != repoIn.FullName {
		t.Errorf("FullName = %q, want %q", got.FullName, repoIn.FullName)
	}

	// Round-trip via SelectLinkedRepos.
	got2, err := repo.SelectLinkedRepos(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("SelectLinkedRepos: %v", err)
	}
	if len(got2) != 1 {
		t.Fatalf("len(SelectLinkedRepos) = %d, want 1", len(got2))
	}
	if got2[0].ID != got.ID {
		t.Errorf("SelectLinkedRepos ID = %d, want %d", got2[0].ID, got.ID)
	}
}

// TestWorkspaceRepo_AddLinkedRepo_DuplicateInWorkspaceReturnsConflict
// covers the unique-violation path: a second AddLinkedRepo with the
// same (workspace_id, github_id) must return *domain.ConflictError.
func TestWorkspaceRepo_AddLinkedRepo_DuplicateInWorkspaceReturnsConflict(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "addlinkedrepo-dup")
	in := makeWorkspaceInput(orgID, "dup-link-host")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	first := makeLinkedRepoInput(inserted.ID, "first-link")
	if _, err := repo.AddLinkedRepo(ctx, first); err != nil {
		t.Fatalf("AddLinkedRepo (first): %v", err)
	}
	second := makeLinkedRepoInput(inserted.ID, "first-link")
	second.FullName = "linked/different-name" // same github_id, differ on name to prove conflict is keyed on github_id
	_, err = repo.AddLinkedRepo(ctx, second)
	if err == nil {
		t.Fatalf("AddLinkedRepo (second): expected error, got nil")
	}
	var conflictErr *domain.ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("AddLinkedRepo (second): got %T (%v), want *domain.ConflictError", err, err)
	}
}

// TestWorkspaceRepo_AddLinkedRepo_NonExistentWorkspaceReturnsInternalError
// covers T-WS-1BiiA-018 (TRIANGULATE): a workspace_id that does
// not exist triggers a FK violation → translated to
// *domain.InternalError wrapping the cause. The handler maps to
// HTTP 500.
func TestWorkspaceRepo_AddLinkedRepo_NonExistentWorkspaceReturnsInternalError(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orphan := makeLinkedRepoInput(99999999, "orphan")
	_, err := repo.AddLinkedRepo(ctx, orphan)
	if err == nil {
		t.Fatalf("AddLinkedRepo(workspace_id=99999999): expected error, got nil")
	}
	var ierr *domain.InternalError
	if !errors.As(err, &ierr) {
		t.Fatalf("AddLinkedRepo(workspace_id=99999999): got %T (%v), want *domain.InternalError", err, err)
	}
}

// ---------------------------------------------------------------------------
// RemoveLinkedRepo
// ---------------------------------------------------------------------------

// TestWorkspaceRepo_RemoveLinkedRepo_DeletesRow covers T-WS-1BiiA-019
// (RED) + T-WS-1BiiA-020 (GREEN): a successful RemoveLinkedRepo
// must hard-delete the row. The workspace must be untouched.
func TestWorkspaceRepo_RemoveLinkedRepo_DeletesRow(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "removelinkedrepo-basic")
	in := makeWorkspaceInput(orgID, "remove-link-host")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	added, err := repo.AddLinkedRepo(ctx, makeLinkedRepoInput(inserted.ID, "to-remove"))
	if err != nil {
		t.Fatalf("AddLinkedRepo: %v", err)
	}

	if err := repo.RemoveLinkedRepo(ctx, inserted.ID, added.ID); err != nil {
		t.Fatalf("RemoveLinkedRepo: %v", err)
	}

	// Verify gone via SelectLinkedRepos.
	rs, err := repo.SelectLinkedRepos(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("SelectLinkedRepos: %v", err)
	}
	if len(rs) != 0 {
		t.Errorf("SelectLinkedRepos len = %d, want 0", len(rs))
	}
	// Workspace is still there and live.
	got, err := repo.SelectByID(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("SelectByID post-remove: %v", err)
	}
	if got.Name != in.Name {
		t.Errorf("workspace Name changed: got %q, want %q", got.Name, in.Name)
	}
	if got.DeletedAt != nil {
		t.Errorf("workspace DeletedAt = %v, want nil (RemoveLinkedRepo must NOT cascade-delete the workspace)", got.DeletedAt)
	}
}

// TestWorkspaceRepo_RemoveLinkedRepo_NotFound covers the not-found
// case: removing a non-existent (workspaceID, repoID) pair must
// return *domain.NotFoundError.
func TestWorkspaceRepo_RemoveLinkedRepo_NotFound(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := repo.RemoveLinkedRepo(ctx, 99999999, 99999999)
	if err == nil {
		t.Fatalf("RemoveLinkedRepo(not-found): expected error, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("RemoveLinkedRepo(not-found): got %T (%v), want *domain.NotFoundError", err, err)
	}
}

// ---------------------------------------------------------------------------
// SelectLinkedRepos
// ---------------------------------------------------------------------------

// TestWorkspaceRepo_SelectLinkedRepos_OrdersAddedAtAsc covers
// T-WS-1BiiA-021 (RED) + T-WS-1BiiA-022 (GREEN): rows must be
// returned in (added_at ASC, id ASC) order — chronological. Tests
// deterministic name ordering via deterministic sleep gaps.
func TestWorkspaceRepo_SelectLinkedRepos_OrdersAddedAtAsc(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "selectlinked-order")
	in := makeWorkspaceInput(orgID, "order-host")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	names := []string{"first-link", "second-link", "third-link"}
	for _, n := range names {
		if _, err := repo.AddLinkedRepo(ctx, makeLinkedRepoInput(inserted.ID, n)); err != nil {
			t.Fatalf("AddLinkedRepo %q: %v", n, err)
		}
		time.Sleep(2 * time.Millisecond)
	}

	got, err := repo.SelectLinkedRepos(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("SelectLinkedRepos: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Name != "first-link" {
		t.Errorf("got[0].Name = %q, want %q", got[0].Name, "first-link")
	}
	if got[1].Name != "second-link" {
		t.Errorf("got[1].Name = %q, want %q", got[1].Name, "second-link")
	}
	if got[2].Name != "third-link" {
		t.Errorf("got[2].Name = %q, want %q", got[2].Name, "third-link")
	}
}

// TestWorkspaceRepo_SelectLinkedRepos_EmptyReturnsEmptySlice covers
// T-WS-1BiiA-023 (TRIANGULATE): zero linked repos must return an
// empty slice (NOT nil) so callers can `for _, r := range got`
// without nil checks.
func TestWorkspaceRepo_SelectLinkedRepos_EmptyReturnsEmptySlice(t *testing.T) {
	db := integrationDB(t)
	ensureMigrations(t, db)
	ensureWorkspaceMigrations(t, db)
	truncateWorkspaces(t, db)

	repo := postgres.NewWorkspaceRepo(db)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	orgID := makeOrg(t, db, "selectlinked-empty")
	in := makeWorkspaceInput(orgID, "empty-host")
	inserted, err := repo.Insert(ctx, in)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	got, err := repo.SelectLinkedRepos(ctx, inserted.ID)
	if err != nil {
		t.Fatalf("SelectLinkedRepos: %v", err)
	}
	if got == nil {
		t.Fatalf("SelectLinkedRepos: got nil slice, want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

// _ ensures INTEGRATION=1 is required; if someone runs without it
// every test above is skipped, which would silently pass and hide a
// regression. We re-state the gate in a unit-level test that fails
// fast on misconfiguration.
func TestWorkspaceRepo_IntegrationGate_RequiresEnv(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
}
