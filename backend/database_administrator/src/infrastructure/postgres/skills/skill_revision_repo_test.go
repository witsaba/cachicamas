// Package skills_test — integration tests for the SkillRevisionRepo.
// Companion file to skill_repo_test.go. Mirrors prompts/
// prompt_revision_repo_test.go shape: separate file, separate test
// functions, same helpers (openTestDB, ensureSkillMigrations,
// cleanSkillTables, makeSkillInput, seedSkill) defined in
// skill_repo_test.go.
package skills_test

import (
	"context"
	"errors"
	"testing"

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
