// Package prompts_test — integration tests for the PromptRevisionRepo.
// Companion file to prompt_repo_test.go. Both files share the same
// helpers (openTestDB, ensurePromptMigrations, cleanPromptTables,
// makePromptInput, seedPrompt) defined in prompt_repo_test.go.
package prompts_test

import (
	"context"
	"errors"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/prompts"
)

// ---------------------------------------------------------------------------
// PromptRevisionRepo.Insert
// ---------------------------------------------------------------------------

func TestPromptRevisionRepo_Insert_HappyPath(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-insert", "desc", "body")

	rev := &domain.PromptRevision{
		PromptID:       prompt.ID,
		RevisionNumber: 1,
		Description:    "desc",
		Body:           "body",
	}
	if err := repo.Insert(context.Background(), db, rev); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if rev.ID == 0 {
		t.Errorf("ID not set: %+v", rev)
	}
	if rev.CreatedAt.IsZero() {
		t.Errorf("CreatedAt not set: %+v", rev)
	}
}

func TestPromptRevisionRepo_Insert_DuplicateRevisionNumber_ReturnsConflictError(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-dup", "desc", "body")
	ctx := context.Background()

	if err := repo.Insert(ctx, db, &domain.PromptRevision{
		PromptID:       prompt.ID,
		RevisionNumber: 1,
		Description:    "d", Body: "b",
	}); err != nil {
		t.Fatalf("first Insert: %v", err)
	}
	err := repo.Insert(ctx, db, &domain.PromptRevision{
		PromptID:       prompt.ID,
		RevisionNumber: 1,
		Description:    "d", Body: "b",
	})
	if err == nil {
		t.Fatalf("expected conflict error on duplicate revision_number")
	}
	var cerr *domain.ConflictError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *domain.ConflictError, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// PromptRevisionRepo.SelectLatestForPrompt
// ---------------------------------------------------------------------------

func TestPromptRevisionRepo_SelectLatestForPrompt_ReturnsLatest(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-latest", "desc", "body")
	ctx := context.Background()

	for n := 1; n <= 3; n++ {
		_ = repo.Insert(ctx, db, &domain.PromptRevision{
			PromptID:       prompt.ID,
			RevisionNumber: n,
			Description:    "d",
			Body:           "body v" + string(rune('0'+n)),
		})
	}

	got, err := repo.SelectLatestForPrompt(ctx, db, prompt.ID)
	if err != nil {
		t.Fatalf("SelectLatestForPrompt: %v", err)
	}
	if got.RevisionNumber != 3 {
		t.Errorf("Latest = %d, want 3", got.RevisionNumber)
	}
	if got.Body != "body v3" {
		t.Errorf("Latest body = %q, want %q", got.Body, "body v3")
	}
}

func TestPromptRevisionRepo_SelectLatestForPrompt_NoRevisions_ReturnsNotFound(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-empty", "desc", "body")
	_, err := repo.SelectLatestForPrompt(context.Background(), db, prompt.ID)
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *domain.NotFoundError, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// PromptRevisionRepo.SelectByPromptAndNumber
// ---------------------------------------------------------------------------

func TestPromptRevisionRepo_SelectByPromptAndNumber_HappyPath(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-by-number", "desc", "body")
	ctx := context.Background()
	for n := 1; n <= 5; n++ {
		_ = repo.Insert(ctx, db, &domain.PromptRevision{
			PromptID: prompt.ID, RevisionNumber: n,
			Description: "d", Body: "body " + string(rune('0'+n)),
		})
	}

	got, err := repo.SelectByPromptAndNumber(ctx, db, prompt.ID, 3)
	if err != nil {
		t.Fatalf("SelectByPromptAndNumber: %v", err)
	}
	if got.RevisionNumber != 3 {
		t.Errorf("revision_number = %d, want 3", got.RevisionNumber)
	}
	if got.Body != "body 3" {
		t.Errorf("body = %q, want %q", got.Body, "body 3")
	}
}

func TestPromptRevisionRepo_SelectByPromptAndNumber_NotFound(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-missing", "desc", "body")
	_, err := repo.SelectByPromptAndNumber(context.Background(), db, prompt.ID, 99)
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *domain.NotFoundError, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// PromptRevisionRepo.SelectListByPrompt
// ---------------------------------------------------------------------------

func TestPromptRevisionRepo_SelectListByPrompt_OrderDesc(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-list", "desc", "body")
	ctx := context.Background()
	for n := 1; n <= 4; n++ {
		_ = repo.Insert(ctx, db, &domain.PromptRevision{
			PromptID: prompt.ID, RevisionNumber: n,
			Description: "d", Body: "body",
		})
	}

	got, err := repo.SelectListByPrompt(ctx, db, prompt.ID)
	if err != nil {
		t.Fatalf("SelectListByPrompt: %v", err)
	}
	if len(got) != 4 {
		t.Fatalf("len = %d, want 4", len(got))
	}
	// Newest first: 4, 3, 2, 1.
	for i, want := range []int{4, 3, 2, 1} {
		if got[i].RevisionNumber != want {
			t.Errorf("got[%d].revision_number = %d, want %d", i, got[i].RevisionNumber, want)
		}
	}
}

func TestPromptRevisionRepo_SelectListByPrompt_EmptyReturnsEmptySlice(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	repo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-empty-list", "desc", "body")
	got, err := repo.SelectListByPrompt(context.Background(), db, prompt.ID)
	if err != nil {
		t.Fatalf("SelectListByPrompt: %v", err)
	}
	if got == nil {
		t.Fatalf("returned nil; want empty slice")
	}
	if len(got) != 0 {
		t.Errorf("len = %d, want 0", len(got))
	}
}

// ---------------------------------------------------------------------------
// Cascade delete: when a prompt is hard-deleted, its revisions go too.
// ---------------------------------------------------------------------------

func TestPromptRevisionRepo_CascadeDeleteRemovesRevisions(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer db.Close()
	ensurePromptMigrations(t, db)
	cleanPromptTables(t, db)

	revRepo := prompts.NewPromptRevisionRepo(db)
	prompt := seedPrompt(t, db, "rev-cascade", "desc", "body")
	for n := 1; n <= 3; n++ {
		_ = revRepo.Insert(context.Background(), db, &domain.PromptRevision{
			PromptID: prompt.ID, RevisionNumber: n,
			Description: "d", Body: "b",
		})
	}

	// Hard-delete the prompt (cascade should remove revisions).
	if _, err := db.ExecContext(context.Background(), "DELETE FROM prompt WHERE id = $1", prompt.ID); err != nil {
		t.Fatalf("hard delete: %v", err)
	}

	got, err := revRepo.SelectListByPrompt(context.Background(), db, prompt.ID)
	if err != nil {
		t.Fatalf("SelectListByPrompt after cascade: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("after cascade: len = %d, want 0", len(got))
	}
}