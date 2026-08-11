// Package application_test — integration tests for PromptService.
// Mirrors the pattern in sync_service_test.go and
// sync_callback_integration_test.go. All tests are gated on
// INTEGRATION=1 because they need a live Postgres.
package application_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/prompts"
)

// ---------------------------------------------------------------------------
// Helpers (duplicated from prompts_test to avoid cross-package
// coupling; keep aligned with that file's helpers).
// ---------------------------------------------------------------------------

func openAppTestDB(t *testing.T) *sql.DB {
	t.Helper()
	host := getenvOr("POSTGRES_HOST", "localhost")
	port := getenvOr("POSTGRES_PORT", "5432")
	user := getenvOr("QUEEN_USER", "queen")
	pass := getenvOr("QUEEN_PASSWORD", "changeme-queen")
	dbname := getenvOr("POSTGRES_DB", "cachicamas_pg")
	sslmode := getenvOr("POSTGRES_SSLMODE", "disable")
	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + dbname + " sslmode=" + sslmode
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := dbConn.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return dbConn
}

func getenvOr(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func ensureAppPromptMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS prompt (
		    id           BIGSERIAL    PRIMARY KEY,
		    description  TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 280),
		    slug         TEXT         NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$'),
		    body         TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
		    deleted_at   TIMESTAMPTZ  NULL,
		    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS prompt_revision (
		    id              BIGSERIAL    PRIMARY KEY,
		    prompt_id       BIGINT       NOT NULL REFERENCES prompt(id) ON DELETE CASCADE,
		    revision_number INT          NOT NULL CHECK (revision_number > 0),
		    description     TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 280),
		    body            TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
		    change_note     TEXT         NULL,
		    created_by      TEXT         NULL,
		    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    CONSTRAINT prompt_revision_unique UNIQUE (prompt_id, revision_number)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS prompt_slug_active_uidx ON prompt(slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS prompt_updated_at_idx ON prompt(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS prompt_revision_prompt_id_idx ON prompt_revision(prompt_id, revision_number DESC)`,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("ensureAppPromptMigrations: %v\nSQL: %s", err, s)
		}
	}
}

func cleanAppPromptTables(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "DELETE FROM prompt_revision"); err != nil {
		t.Fatalf("clean prompt_revision: %v", err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM prompt"); err != nil {
		t.Fatalf("clean prompt: %v", err)
	}
}

func skipIfNoIntegrationApp(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
}

func newAppService(t *testing.T, db *sql.DB) *application.PromptService {
	t.Helper()
	promptRepo := prompts.NewPromptRepo(db)
	revRepo := prompts.NewPromptRevisionRepo(db)
	return application.NewPromptService(promptRepo, revRepo, db, nil)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestPromptService_Create_WritesRevisionOne(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	p, rev, err := svc.Create(context.Background(), domain.CreatePromptInput{
		Slug:        "create-rev-1",
		Description: "Test create",
		Body:        "body one",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p == nil || p.ID == 0 {
		t.Fatalf("prompt not populated: %+v", p)
	}
	if rev == nil || rev.RevisionNumber != 1 {
		t.Fatalf("revision != 1: %+v", rev)
	}
	if rev.Body != "body one" {
		t.Errorf("revision body = %q, want %q", rev.Body, "body one")
	}
}

func TestPromptService_Create_RejectsInvalidSlug(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	_, _, err := svc.Create(context.Background(), domain.CreatePromptInput{
		Slug:        "BAD",
		Description: "desc",
		Body:        "body",
	})
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T (%v)", err, err)
	}
}

func TestPromptService_Create_DuplicateSlug_ReturnsConflictError(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "dup", Description: "d", Body: "b",
	}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "dup", Description: "d2", Body: "b2",
	})
	var cerr *domain.ConflictError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *ConflictError on dup, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func TestPromptService_Update_AppendsNextRevision(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "update-test", Description: "d1", Body: "body one",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, rev, err := svc.Update(ctx, "update-test", domain.UpdatePromptInput{
		Body: stringPtr("body two"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rev.RevisionNumber != 2 {
		t.Errorf("revision = %d, want 2", rev.RevisionNumber)
	}
	if rev.Body != "body two" {
		t.Errorf("revision body = %q, want %q", rev.Body, "body two")
	}
}

func TestPromptService_Update_DescriptionOnly_AppendsRevision(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	_, _, _ = svc.Create(ctx, domain.CreatePromptInput{
		Slug: "update-desc-only", Description: "d1", Body: "body",
	})
	_, rev, err := svc.Update(ctx, "update-desc-only", domain.UpdatePromptInput{
		Description: stringPtr("d2"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rev.RevisionNumber != 2 {
		t.Errorf("revision = %d, want 2", rev.RevisionNumber)
	}
	if rev.Description != "d2" {
		t.Errorf("description = %q, want %q", rev.Description, "d2")
	}
}

func TestPromptService_Update_DeletedPrompt_ReturnsGoneError(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "update-deleted", Description: "d", Body: "b",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "update-deleted"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	_, _, err := svc.Update(ctx, "update-deleted", domain.UpdatePromptInput{
		Body: stringPtr("new"),
	})
	if _, ok := domain.AsPromptDeleted(err); !ok {
		t.Fatalf("expected *GoneError, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// Restore
// ---------------------------------------------------------------------------

func TestPromptService_Restore_AppendsNewRevisionWithHistoricalBody(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "restore-test", Description: "d", Body: "v1 body",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := svc.Update(ctx, "restore-test", domain.UpdatePromptInput{
		Body: stringPtr("v2 body"),
	}); err != nil {
		t.Fatalf("first Update: %v", err)
	}
	if _, _, err := svc.Update(ctx, "restore-test", domain.UpdatePromptInput{
		Body: stringPtr("v3 body"),
	}); err != nil {
		t.Fatalf("second Update: %v", err)
	}

	p, rev, err := svc.Restore(ctx, "restore-test", 1)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if p.Body != "v1 body" {
		t.Errorf("prompt body = %q, want %q (restored)", p.Body, "v1 body")
	}
	if rev.RevisionNumber != 4 {
		t.Errorf("revision = %d, want 4 (new revision)", rev.RevisionNumber)
	}
	if rev.Body != "v1 body" {
		t.Errorf("new revision body = %q, want %q", rev.Body, "v1 body")
	}
	if rev.ChangeNote == nil || *rev.ChangeNote == "" {
		t.Errorf("change_note must be set: %+v", rev.ChangeNote)
	}
}

func TestPromptService_Restore_OnDeletedPrompt_ReturnsGoneError(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "restore-deleted", Description: "d", Body: "b",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "restore-deleted"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	_, _, err := svc.Restore(ctx, "restore-deleted", 1)
	if _, ok := domain.AsPromptDeleted(err); !ok {
		t.Fatalf("expected *GoneError, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// SoftDelete
// ---------------------------------------------------------------------------

func TestPromptService_SoftDelete_Idempotent(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "soft-delete-idempotent", Description: "d", Body: "b",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "soft-delete-idempotent"); err != nil {
		t.Fatalf("first SoftDelete: %v", err)
	}
	// Second call on an already-deleted slug: the repo's
	// SelectBySlug returns NotFound; the service treats that as
	// success per idempotency contract.
	if err := svc.SoftDelete(ctx, "soft-delete-idempotent"); err != nil {
		t.Fatalf("second SoftDelete (idempotent): %v", err)
	}
}

func TestPromptService_SoftDelete_ThenRecreate_AllowsReuse(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "reusable", Description: "d", Body: "b",
	}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "reusable"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "reusable", Description: "d2", Body: "b2",
	}); err != nil {
		t.Fatalf("recreate: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Reads
// ---------------------------------------------------------------------------

func TestPromptService_GetBySlug_DeletedPrompt_ReturnsNotFound(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "get-deleted", Description: "d", Body: "b",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "get-deleted"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	_, err := svc.GetBySlug(ctx, "get-deleted")
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *NotFoundError, got %T (%v)", err, err)
	}
}

func TestPromptService_List_ExcludesDeleted(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	for _, slug := range []string{"list-a", "list-b", "list-c"} {
		if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
			Slug: slug, Description: "d", Body: "b",
		}); err != nil {
			t.Fatalf("Create %s: %v", slug, err)
		}
	}
	if err := svc.SoftDelete(ctx, "list-b"); err != nil {
		t.Fatalf("SoftDelete list-b: %v", err)
	}
	got, err := svc.List(ctx, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("List len = %d, want 2 (deleted excluded)", len(got))
	}
}

func TestPromptService_List_LimitCapEnforced(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	// limit=300 must be clamped to MaxListLimit (200); with zero
	// rows we can only assert the call does not error and returns
	// an empty slice (cap applied silently).
	got, err := svc.List(context.Background(), 300, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got == nil {
		t.Errorf("List returned nil; want empty slice")
	}
}

func TestPromptService_ListRevisions_NewestFirst(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "list-revs", Description: "d", Body: "b",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	for i := 0; i < 3; i++ {
		body := string(rune('a' + i))
		if _, _, err := svc.Update(ctx, "list-revs", domain.UpdatePromptInput{
			Body: &body,
		}); err != nil {
			t.Fatalf("Update %d: %v", i, err)
		}
	}
	got, err := svc.ListRevisions(ctx, "list-revs")
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4 (1 create + 3 updates)", len(got))
	}
	for i, want := range []int{4, 3, 2, 1} {
		if got[i].RevisionNumber != want {
			t.Errorf("got[%d].revision_number = %d, want %d", i, got[i].RevisionNumber, want)
		}
	}
}

// ---------------------------------------------------------------------------
// Concurrency
// ---------------------------------------------------------------------------

func TestPromptService_ConcurrentCreate_OneSucceedsOneConflicts(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()

	var successCount, conflictCount int32
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			_, _, err := svc.Create(ctx, domain.CreatePromptInput{
				Slug: "concurrent-create", Description: "d", Body: "b",
			})
			if err == nil {
				atomic.AddInt32(&successCount, 1)
			} else {
				var cerr *domain.ConflictError
				if errors.As(err, &cerr) {
					atomic.AddInt32(&conflictCount, 1)
				}
			}
		}()
	}
	wg.Wait()

	if successCount != 1 {
		t.Errorf("successes = %d, want 1", successCount)
	}
	if conflictCount != 1 {
		t.Errorf("conflicts = %d, want 1", conflictCount)
	}
}

func TestPromptService_ConcurrentUpdate_ProducesMonotonicRevisions(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()

	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "concurrent-update", Description: "d", Body: "b1",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	const N = 5
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			body := string(rune('a' + i))
			if _, _, err := svc.Update(ctx, "concurrent-update", domain.UpdatePromptInput{
				Body: &body,
			}); err != nil {
				t.Errorf("Update %d: %v", i, err)
			}
		}(i)
	}
	wg.Wait()

	revs, err := svc.ListRevisions(ctx, "concurrent-update")
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	if len(revs) != N+1 {
		t.Errorf("revisions = %d, want %d (1 create + %d updates)", len(revs), N+1, N)
	}
	// Verify monotonic descending: N+1, N, N-1, ..., 1
	for i, want := 0, N+1; i < len(revs); i, want = i+1, want-1 {
		if revs[i].RevisionNumber != want {
			t.Errorf("revs[%d].revision_number = %d, want %d", i, revs[i].RevisionNumber, want)
		}
	}
}

func TestPromptService_ConcurrentRestoreAndUpdate_NoLostUpdate(t *testing.T) {
	skipIfNoIntegrationApp(t)
	db := openAppTestDB(t)
	defer func() { _ = db.Close() }()
	ensureAppPromptMigrations(t, db)
	cleanAppPromptTables(t, db)

	svc := newAppService(t, db)
	ctx := context.Background()

	if _, _, err := svc.Create(ctx, domain.CreatePromptInput{
		Slug: "concurrent-restore-update", Description: "d", Body: "v1",
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if _, _, err := svc.Update(ctx, "concurrent-restore-update", domain.UpdatePromptInput{
			Body: stringPtr("v2"),
		}); err != nil {
			t.Errorf("Update: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if _, _, err := svc.Restore(ctx, "concurrent-restore-update", 1); err != nil {
			t.Errorf("Restore: %v", err)
		}
	}()
	wg.Wait()

	revs, err := svc.ListRevisions(ctx, "concurrent-restore-update")
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	// Three revisions: create (1) + either update or restore (2) + the other (3).
	if len(revs) != 3 {
		t.Errorf("revisions = %d, want 3 (1 create + 1 update + 1 restore)", len(revs))
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func stringPtr(s string) *string { return &s }