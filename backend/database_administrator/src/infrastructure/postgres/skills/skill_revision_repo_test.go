// Package skills_test — integration tests for SkillRevisionRepo.
// Companion to skill_repo_test.go. Same helpers (openTestDB,
// ensureSkillMigrations, cleanSkillTables, makeSkillInput, seedSkill)
// live in skill_repo_test.go.
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

// seedSkillRevisionAt inserts a skill_revision row via raw SQL.
// Used by SkillRevisionRepo tests without depending on the repo's Insert.
func seedSkillRevisionAt(t *testing.T, db *sql.DB, skillID int64, n int, description, body string) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := db.ExecContext(ctx,
		`INSERT INTO skill_revision (skill_id, revision_number, description, body)
         VALUES ($1, $2, $3, $4)`, skillID, n, description, body)
	return err
}

// TestSkillRevisionRepo_Insert_PersistsRow covers spec R-SK-001:
// Insert sets id, created_at from the DB clock (append-only invariant).
func TestSkillRevisionRepo_Insert_PersistsRow(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRevisionRepo(db)
	skill := seedSkill(t, db, "rev-insert", "d", "b")
	rev := &domain.SkillRevision{
		SkillID: skill.ID, RevisionNumber: 1, Description: "d", Body: "b",
	}
	if err := repo.Insert(context.Background(), db, rev); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if rev.ID == 0 || rev.CreatedAt.IsZero() {
		t.Errorf("Insert did not populate row: %+v", rev)
	}
}

// TestSkillRevisionRepo_Insert_RejectsDuplicateRevisionNumber covers
// spec S-SK-041 + ADR-SK-004: duplicate (skill_id, revision_number)
// collides on UNIQUE and surfaces as *domain.ConflictError (409).
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
		SkillID: skill.ID, RevisionNumber: 1, Description: "d", Body: "b",
	}); err != nil {
		t.Fatalf("first Insert: %v", err)
	}
	err := repo.Insert(ctx, db, &domain.SkillRevision{
		SkillID: skill.ID, RevisionNumber: 1, Description: "d", Body: "b",
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
// covers spec R-SK-001 + S-SK-013: service restores a historical
// revision, must fetch exact (skill_id, n) row and read body+description.
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
	if got.RevisionNumber != 3 || got.Body != "body 3" || got.Description != "desc" {
		t.Errorf("got = {n:%d, body:%q, desc:%q}, want {n:3, body:body 3, desc:desc}",
			got.RevisionNumber, got.Body, got.Description)
	}
}

// TestSkillRevisionRepo_SelectBySkillAndNumber_MissingReturnsNotFound
// covers spec S-SK-013 negative path: non-existent revision number
// returns *domain.NotFoundError (404).
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

// TestSkillRevisionRepo_ListBySkillID_NewestFirst covers spec SCN-5.5:
// revisions are listed in descending revision_number order (newest first).
func TestSkillRevisionRepo_ListBySkillID_NewestFirst(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRevisionRepo(db)
	skill := seedSkill(t, db, "rev-list", "d", "b")
	for n := 1; n <= 5; n++ {
		if err := seedSkillRevisionAt(t, db, skill.ID, n, "d", "body"); err != nil {
			t.Fatalf("seed rev %d: %v", n, err)
		}
	}

	got, err := repo.ListBySkillID(context.Background(), db, skill.ID)
	if err != nil {
		t.Fatalf("ListBySkillID: %v", err)
	}
	if len(got) != 5 {
		t.Fatalf("len = %d, want 5", len(got))
	}
	// Newest first: 5, 4, 3, 2, 1.
	for i, want := range []int{5, 4, 3, 2, 1} {
		if got[i].RevisionNumber != want {
			t.Errorf("got[%d].revision_number = %d, want %d", i, got[i].RevisionNumber, want)
		}
	}
}
