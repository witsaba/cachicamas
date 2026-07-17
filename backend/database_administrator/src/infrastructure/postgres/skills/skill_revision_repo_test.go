// Package skills_test — integration tests for the SkillRevisionRepo.
// Companion file to skill_repo_test.go. Mirrors prompts/
// prompt_revision_repo_test.go shape: separate file, separate test
// functions, same helpers (openTestDB, ensureSkillMigrations,
// cleanSkillTables, makeSkillInput, seedSkill) defined in
// skill_repo_test.go.
package skills_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/skills"
)

// ---------------------------------------------------------------------------
// SkillRevisionRepo.Insert
// ---------------------------------------------------------------------------

// TestSkillRevisionRepo_Insert_PersistsRow covers spec R-SK-001 +
// design §3.2: Insert sets id, created_at from the DB clock (the
// append-only invariant). The skill_id/revision_number pair must be
// (already-existing after the parent skill is seeded by seedSkill).
func TestSkillRevisionRepo_Insert_PersistsRow(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRevisionRepo(db)
	skill := seedSkill(t, db, "rev-insert", "d", "b")

	rev := &domain.SkillRevision{
		SkillID:        skill.ID,
		RevisionNumber: 1,
		Description:    "d",
		Body:           "b",
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

// TestSkillRevisionRepo_Insert_RejectsDuplicateRevisionNumber covers
// spec S-SK-041 + ADR-SK-004: a duplicate (skill_id, revision_number)
// collides on the UNIQUE constraint and surfaces as
// *domain.ConflictError so the handler can map to 409. This is the
// smoking gun for a lost-update race that bypassed FOR UPDATE.
func TestSkillRevisionRepo_Insert_RejectsDuplicateRevisionNumber(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRevisionRepo(db)
	skill := seedSkill(t, db, "rev-dup", "d", "b")
	ctx := context.Background()

	if err := repo.Insert(ctx, db, &domain.SkillRevision{
		SkillID:        skill.ID,
		RevisionNumber: 1,
		Description:    "d", Body: "b",
	}); err != nil {
		t.Fatalf("first Insert: %v", err)
	}
	err := repo.Insert(ctx, db, &domain.SkillRevision{
		SkillID:        skill.ID,
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

// TestSkillRevisionRepo_SelectBySkillAndNumber_ReturnsBodyAndDescription
// covers spec R-SK-001 + S-SK-013: the service restores a historical
// revision, so it must be able to fetch the exact (skill_id, n) row
// and read body+description for the new revision.
func TestSkillRevisionRepo_SelectBySkillAndNumber_ReturnsBodyAndDescription(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRevisionRepo(db)
	skill := seedSkill(t, db, "rev-by-number", "d", "b")
	ctx := context.Background()
	for n := 1; n <= 5; n++ {
		if err := seedSkillRevisionAt(t, db, skill.ID, n, "desc", "body "+string(rune('0'+n))); err != nil {
			t.Fatalf("seed rev %d: %v", n, err)
		}
	}

	got, err := repo.SelectBySkillAndNumber(ctx, db, skill.ID, 3)
	if err != nil {
		t.Fatalf("SelectBySkillAndNumber: %v", err)
	}
	if got.RevisionNumber != 3 {
		t.Errorf("revision_number = %d, want 3", got.RevisionNumber)
	}
	if got.Body != "body 3" {
		t.Errorf("body = %q, want %q", got.Body, "body 3")
	}
	if got.Description != "desc" {
		t.Errorf("description = %q, want %q", got.Description, "desc")
	}
}

// TestSkillRevisionRepo_SelectBySkillAndNumber_MissingReturnsNotFound
// covers spec S-SK-013 negative path: a non-existent revision number
// for an existing skill returns *domain.NotFoundError so the handler
// can map to 404.
func TestSkillRevisionRepo_SelectBySkillAndNumber_MissingReturnsNotFound(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRevisionRepo(db)
	skill := seedSkill(t, db, "rev-missing", "d", "b")
	_, err := repo.SelectBySkillAndNumber(context.Background(), db, skill.ID, 99)
	if err == nil {
		t.Fatalf("expected NotFound, got nil")
	}
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("expected *domain.NotFoundError, got %T (%v)", err, err)
	}
}

// seedSkillRevisionAt inserts a skill_revision row directly via SQL
// at the given revision_number. Used by SkillRevisionRepo tests
// without depending on the repo's own Insert.
func seedSkillRevisionAt(t *testing.T, db *sql.DB, skillID int64, n int, description, body string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx,
		`INSERT INTO skill_revision (skill_id, revision_number, description, body)
         VALUES ($1, $2, $3, $4)`, skillID, n, description, body)
	return err
}
