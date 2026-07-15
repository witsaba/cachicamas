// Package postgres_test contains the integration test suite for the
// prompt + prompt_revision repository adapters (PR2 of
// 2026-07-15-prompt-storage-table). Tests are gated on INTEGRATION=1
// because they need a live Postgres (the same compose-provisioned
// instance used by the migration runner).
//
// Strict TDD discipline (per openspec/AGENTS.md): this file was
// written BEFORE prompt_repo.go and prompt_revision_repo.go existed.
// Running `INTEGRATION=1 go test ./src/infrastructure/postgres/...
// -run TestPromptRepo` with no PromptRepo type must fail to compile
// — that failure IS the RED step.
package prompts_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/prompts"
)

// ---------------------------------------------------------------------------
// Connection helper.
// ---------------------------------------------------------------------------

// openTestDB connects to the compose Postgres instance using the
// environment variables the project standardizes on. Mirrors the
// pattern in organization_repo_test.go.
func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	host := getenvOr("POSTGRES_HOST", "localhost")
	port := getenvOr("POSTGRES_PORT", "5432")
	user := getenvOr("POSTGRES_USER", "queen")
	pass := getenvOr("QUEEN_PASSWORD", "changeme-queen")
	db := getenvOr("POSTGRES_DB", "cachicamas_pg")
	sslmode := getenvOr("POSTGRES_SSLMODE", "disable")
	connStr := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + db + " sslmode=" + sslmode
	dbConn, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("open test DB: %v", err)
	}
	if err := dbConn.Ping(); err != nil {
		t.Fatalf("ping test DB: %v", err)
	}
	return dbConn
}

func getenvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// ensurePromptMigrations applies the PR1 migration idempotently.
// The compose DB already has all earlier migrations; this helper only
// ensures the prompt + prompt_revision tables exist. CREATE TABLE IF
// NOT EXISTS makes the helper safe to call repeatedly.
func ensurePromptMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
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
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("ensurePromptMigrations: %v\nSQL: %s", err, s)
		}
	}
}

// cleanPromptTables wipes prompt + prompt_revision in FK-safe order.
func cleanPromptTables(t *testing.T, db *sql.DB) {
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

// makePromptInput returns a valid CreatePromptInput for use in tests.
func makePromptInput(slug, description, body string) *domain.Prompt {
	return &domain.Prompt{
		Description: description,
		Slug:        slug,
		Body:        body,
	}
}

// seedPrompt inserts one prompt row directly (bypassing repo.Insert)
// for tests that need a pre-existing row.
func seedPrompt(t *testing.T, db *sql.DB, slug, description, body string) *domain.Prompt {
	t.Helper()
	p := makePromptInput(slug, description, body)
	if err := prompts.NewPromptRepo(db).Insert(context.Background(), db, p); err != nil {
		t.Fatalf("seed prompt: %v", err)
	}
	return p
}

// ---------------------------------------------------------------------------
// Skip if INTEGRATION != "1".
// ---------------------------------------------------------------------------

func skipIfNoIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
}

// ---------------------------------------------------------------------------
// PromptRepo.Insert
// ---------------------------------------------------------------------------

func TestPromptRepo_Insert_HappyPath(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	p := makePromptInput("welcome-email", "Welcome email body", "# Welcome\n\nHello.")
	if err := repo.Insert(context.Background(), db, p); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if p.ID == 0 {
		t.Fatalf("Insert did not set ID: %+v", p)
	}
	if p.CreatedAt.IsZero() {
		t.Fatalf("Insert did not set CreatedAt: %+v", p)
	}
	if p.DeletedAt != nil {
		t.Fatalf("Insert set DeletedAt (should be nil): %+v", p)
	}
}

func TestPromptRepo_Insert_DuplicateSlug_ReturnsConflictError(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	ctx := context.Background()
	if err := repo.Insert(ctx, db, makePromptInput("dup-slug", "first", "body one")); err != nil {
		t.Fatalf("first Insert: %v", err)
	}
	err := repo.Insert(ctx, db, makePromptInput("dup-slug", "second", "body two"))
	if err == nil {
		t.Fatalf("expected conflict error on duplicate slug, got nil")
	}
	var cerr *domain.ConflictError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *domain.ConflictError, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// PromptRepo.SelectBySlug / SelectByID
// ---------------------------------------------------------------------------

func TestPromptRepo_SelectBySlug_HappyPath(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	seeded := seedPrompt(t, db, "select-by-slug", "Test select by slug", "body content")

	got, err := repo.SelectBySlug(context.Background(), db, "select-by-slug")
	if err != nil {
		t.Fatalf("SelectBySlug: %v", err)
	}
	if got.ID != seeded.ID {
		t.Errorf("ID = %d, want %d", got.ID, seeded.ID)
	}
	if got.Body != "body content" {
		t.Errorf("Body = %q, want %q", got.Body, "body content")
	}
}

func TestPromptRepo_SelectBySlug_NotFound(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	_, err := repo.SelectBySlug(context.Background(), db, "nonexistent")
	if err == nil {
		t.Fatalf("expected NotFound error, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *domain.NotFoundError, got %T (%v)", err, err)
	}
}

func TestPromptRepo_SelectBySlug_DeletedPrompt_ReturnsNotFound(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	seeded := seedPrompt(t, db, "soft-deleted-slug", "desc", "body")
	if err := repo.SoftDelete(context.Background(), db, seeded.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	_, err := repo.SelectBySlug(context.Background(), db, "soft-deleted-slug")
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *domain.NotFoundError for soft-deleted slug, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// PromptRepo.SelectList
// ---------------------------------------------------------------------------

func TestPromptRepo_SelectList_OrderByUpdatedAtDesc(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	ctx := context.Background()
	a := seedPrompt(t, db, "list-a", "desc a", "body a")
	time.Sleep(10 * time.Millisecond)
	b := seedPrompt(t, db, "list-b", "desc b", "body b")
	time.Sleep(10 * time.Millisecond)
	c := seedPrompt(t, db, "list-c", "desc c", "body c")

	got, err := repo.SelectList(ctx, db, 50, 0)
	if err != nil {
		t.Fatalf("SelectList: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("SelectList len = %d, want 3", len(got))
	}
	// Newest first: c, b, a.
	if got[0].ID != c.ID || got[1].ID != b.ID || got[2].ID != a.ID {
		t.Errorf("SelectList order: got [%d, %d, %d], want [%d, %d, %d]",
			got[0].ID, got[1].ID, got[2].ID, c.ID, b.ID, a.ID)
	}
}

func TestPromptRepo_SelectList_ExcludesDeleted(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	seedPrompt(t, db, "active", "d", "b")
	deleted := seedPrompt(t, db, "deleted", "d", "b")
	if err := repo.SoftDelete(context.Background(), db, deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	got, err := repo.SelectList(context.Background(), db, 50, 0)
	if err != nil {
		t.Fatalf("SelectList: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("SelectList len = %d, want 1 (deleted must be excluded)", len(got))
	}
	if got[0].Slug != "active" {
		t.Errorf("SelectList returned %q, want only %q", got[0].Slug, "active")
	}
}

func TestPromptRepo_SelectList_EmptyReturnsEmptySlice(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	got, err := repo.SelectList(context.Background(), db, 50, 0)
	if err != nil {
		t.Fatalf("SelectList: %v", err)
	}
	if got == nil {
		t.Fatalf("SelectList returned nil; want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("SelectList len = %d, want 0", len(got))
	}
}

// ---------------------------------------------------------------------------
// PromptRepo.UpdateBody
// ---------------------------------------------------------------------------

func TestPromptRepo_UpdateBody_UpdatesRow(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	ctx := context.Background()
	seeded := seedPrompt(t, db, "update-body", "original desc", "original body")
	originalUpdatedAt := seeded.UpdatedAt

	time.Sleep(10 * time.Millisecond)
	if err := repo.UpdateBody(ctx, db, seeded.ID, "new body", "new desc"); err != nil {
		t.Fatalf("UpdateBody: %v", err)
	}
	got, err := repo.SelectBySlug(ctx, db, "update-body")
	if err != nil {
		t.Fatalf("SelectBySlug: %v", err)
	}
	if got.Body != "new body" {
		t.Errorf("Body = %q, want %q", got.Body, "new body")
	}
	if got.Description != "new desc" {
		t.Errorf("Description = %q, want %q", got.Description, "new desc")
	}
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("UpdatedAt = %v, want > %v", got.UpdatedAt, originalUpdatedAt)
	}
}

func TestPromptRepo_UpdateBody_DeletedPrompt_ReturnsNotFound(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	ctx := context.Background()
	seeded := seedPrompt(t, db, "update-deleted", "desc", "body")
	if err := repo.SoftDelete(ctx, db, seeded.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	err := repo.UpdateBody(ctx, db, seeded.ID, "new body", "new desc")
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *domain.NotFoundError on UpdateBody for soft-deleted, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// PromptRepo.SoftDelete
// ---------------------------------------------------------------------------

func TestPromptRepo_SoftDelete_SetsDeletedAt(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	ctx := context.Background()
	seeded := seedPrompt(t, db, "soft-delete", "desc", "body")
	if err := repo.SoftDelete(ctx, db, seeded.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	// The repo's SelectByID hides soft-deleted; use a raw query to
	// verify deleted_at was set.
	var deletedAt *time.Time
	if err := db.QueryRowContext(ctx, "SELECT deleted_at FROM prompt WHERE id = $1", seeded.ID).Scan(&deletedAt); err != nil {
		t.Fatalf("raw query: %v", err)
	}
	if deletedAt == nil {
		t.Fatalf("SoftDelete did not set deleted_at")
	}
}

func TestPromptRepo_SoftDelete_IsIdempotent(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	ctx := context.Background()
	seeded := seedPrompt(t, db, "soft-delete-twice", "desc", "body")
	if err := repo.SoftDelete(ctx, db, seeded.ID); err != nil {
		t.Fatalf("first SoftDelete: %v", err)
	}
	if err := repo.SoftDelete(ctx, db, seeded.ID); err != nil {
		t.Fatalf("second SoftDelete (idempotent): %v", err)
	}
}

// ---------------------------------------------------------------------------
// PromptRepo.LockAndLoad
// ---------------------------------------------------------------------------

func TestPromptRepo_LockAndLoad_AcquiresRowLock(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	ctx := context.Background()
	seeded := seedPrompt(t, db, "lock-and-load", "desc", "body")

	// Open a transaction, acquire the FOR UPDATE lock, then verify a
	// concurrent transaction blocks (with statement_timeout) on the
	// same row.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	loaded, err := repo.LockAndLoad(ctx, tx, seeded.ID)
	if err != nil {
		t.Fatalf("LockAndLoad: %v", err)
	}
	if loaded.ID != seeded.ID {
		t.Errorf("ID = %d, want %d", loaded.ID, seeded.ID)
	}

	// Second TX with a tight statement_timeout should fail to update
	// the row while the lock is held.
	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx2: %v", err)
	}
	defer func() { _ = tx2.Rollback() }()
	if _, err := tx2.ExecContext(ctx, "SET LOCAL statement_timeout = '500ms'"); err != nil {
		t.Fatalf("SET LOCAL statement_timeout: %v", err)
	}
	_, err = tx2.ExecContext(ctx, "UPDATE prompt SET body = 'blocked' WHERE id = $1", seeded.ID)
	if err == nil {
		t.Errorf("expected the second TX to block on FOR UPDATE; got nil error")
	} else if !strings.Contains(strings.ToLower(err.Error()), "cancel") &&
		!strings.Contains(strings.ToLower(err.Error()), "timeout") &&
		!strings.Contains(strings.ToLower(err.Error()), "lock") {
		t.Logf("second TX error: %v (acceptable if lock-related)", err)
	}
}

func TestPromptRepo_LockAndLoad_NotFound(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	tx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()
	_, err = repo.LockAndLoad(context.Background(), tx, 99999)
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *domain.NotFoundError, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// PromptRepo.MaxRevisionNumber
// ---------------------------------------------------------------------------

func TestPromptRepo_MaxRevisionNumber_ReturnsZeroOnEmpty(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	seeded := seedPrompt(t, db, "max-rev-zero", "desc", "body")
	got, err := repo.MaxRevisionNumber(context.Background(), db, seeded.ID)
	if err != nil {
		t.Fatalf("MaxRevisionNumber: %v", err)
	}
	if got != 0 {
		t.Errorf("MaxRevisionNumber = %d, want 0", got)
	}
}

func TestPromptRepo_MaxRevisionNumber_ReturnsLatestOnExisting(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRepo(db)
	revRepo := prompts.NewPromptRevisionRepo(db)
	ctx := context.Background()

	seeded := seedPrompt(t, db, "max-rev-existing", "desc", "body")
	for n := 1; n <= 4; n++ {
		err := revRepo.Insert(ctx, db, &domain.PromptRevision{
			PromptID:       seeded.ID,
			RevisionNumber: n,
			Description:    seeded.Description,
			Body:           seeded.Body,
		})
		if err != nil {
			t.Fatalf("Insert revision %d: %v", n, err)
		}
	}

	got, err := repo.MaxRevisionNumber(ctx, db, seeded.ID)
	if err != nil {
		t.Fatalf("MaxRevisionNumber: %v", err)
	}
	if got != 4 {
		t.Errorf("MaxRevisionNumber = %d, want 4", got)
	}
}