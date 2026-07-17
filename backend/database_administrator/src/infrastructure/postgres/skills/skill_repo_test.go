// Package skills_test — integration tests for the skill + skill_revision
// repository adapters (PR1b of cachicamas-skills-foundational). Gated
// on INTEGRATION=1 (compose Postgres). Mirrors prompts/prompt_repo_test.go.
// Strict TDD: every test was written before its production code; the
// RED step is a build failure.
package skills_test

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
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/skills"
)

// openTestDB connects to the compose Postgres instance using the
// project-standard env vars. Mirrors prompts/prompt_repo_test.go.
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

// ensureSkillMigrations applies the PR1a skill migration idempotently.
// CREATE TABLE IF NOT EXISTS makes it safe to call repeatedly.
func ensureSkillMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS skill (
		    id BIGSERIAL PRIMARY KEY, name TEXT NOT NULL
		         CHECK (name ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(name) BETWEEN 1 AND 64),
		    description TEXT NOT NULL CHECK (length(description) BETWEEN 1 AND 1024),
		    body TEXT NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
		    deleted_at TIMESTAMPTZ NULL,
		    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		    updated_at TIMESTAMPTZ NOT NULL DEFAULT now())`,
		`CREATE TABLE IF NOT EXISTS skill_revision (
		    id BIGSERIAL PRIMARY KEY,
		    skill_id BIGINT NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
		    revision_number INT NOT NULL CHECK (revision_number > 0),
		    description TEXT NOT NULL CHECK (length(description) BETWEEN 1 AND 1024),
		    body TEXT NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
		    change_note TEXT NULL,
		    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
		    CONSTRAINT skill_revision_unique UNIQUE (skill_id, revision_number))`,
		`CREATE UNIQUE INDEX IF NOT EXISTS skill_slug_active_uidx ON skill(name) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS skill_updated_at_idx ON skill(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS skill_revision_skill_id_idx ON skill_revision(skill_id, revision_number DESC)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("ensureSkillMigrations: %v\nSQL: %s", err, s)
		}
	}
}

func cleanSkillTables(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "DELETE FROM skill_revision"); err != nil {
		t.Fatalf("clean skill_revision: %v", err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM skill"); err != nil {
		t.Fatalf("clean skill: %v", err)
	}
}

func makeSkillInput(name, description, body string) *domain.Skill {
	return &domain.Skill{Name: name, Description: description, Body: body}
}

func seedSkill(t *testing.T, db *sql.DB, name, description, body string) *domain.Skill {
	t.Helper()
	s := makeSkillInput(name, description, body)
	if err := skills.NewSkillRepo(db).Insert(context.Background(), db, s); err != nil {
		t.Fatalf("seed skill: %v", err)
	}
	return s
}

func skipIfNoIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
}

// seedSkillRevisions inserts n revisions (1..n) via raw SQL — used by
// tests that need a revision history without depending on
// SkillRevisionRepo.Insert (which lands in task 2.9).
func seedSkillRevisions(t *testing.T, db *sql.DB, skillID int64, n int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for i := 1; i <= n; i++ {
		if _, err := db.ExecContext(ctx,
			`INSERT INTO skill_revision (skill_id, revision_number, description, body)
             VALUES ($1, $2, 'd', 'b')`, skillID, i); err != nil {
			t.Fatalf("seed revision %d: %v", i, err)
		}
	}
}

// rawSoftDelete sets deleted_at = now() via raw SQL. Used by tests
// that need a soft-deleted row without depending on SkillRepo.SoftDelete.
func rawSoftDelete(t *testing.T, db *sql.DB, id int64) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "UPDATE skill SET deleted_at = now() WHERE id = $1", id); err != nil {
		t.Fatalf("raw soft-delete: %v", err)
	}
}

// ---------------------------------------------------------------------------
// SkillRepo.Insert (tasks 2.1, 2.2)
// ---------------------------------------------------------------------------

// TestSkillRepo_Insert_PersistsRowAndReturnsID covers spec S-SK-001:
// Insert persists a new skill row and returns a populated *Skill with
// non-zero ID, non-zero timestamps, nil DeletedAt (DB DEFAULT).
func TestSkillRepo_Insert_PersistsRowAndReturnsID(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	s := makeSkillInput("pdf-cleanup", "Cleans PDFs", "---\nname: pdf-cleanup\ndescription: Cleans PDFs\n---\n# Body")
	if err := repo.Insert(context.Background(), db, s); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if s.ID == 0 || s.CreatedAt.IsZero() || s.UpdatedAt.IsZero() || s.DeletedAt != nil {
		t.Fatalf("Insert did not populate row: %+v", s)
	}
}

// TestSkillRepo_Insert_TranslatesUniqueViolation covers spec S-SK-037 +
// ADR-SK-003: a second INSERT against the partial UNIQUE index
// `skill_slug_active_uidx` must surface as *domain.ConflictError
// (Code() == "conflict") so the handler can map to 409.
func TestSkillRepo_Insert_TranslatesUniqueViolation(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()
	if err := repo.Insert(ctx, db, makeSkillInput("dup-name", "first", "b1")); err != nil {
		t.Fatalf("first Insert: %v", err)
	}
	err := repo.Insert(ctx, db, makeSkillInput("dup-name", "second", "b2"))
	if err == nil {
		t.Fatalf("expected conflict error on duplicate name, got nil")
	}
	var cerr *domain.ConflictError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *domain.ConflictError, got %T (%v)", err, err)
	}
	if cerr.Code() != domain.CodeConflict {
		t.Errorf("ConflictError.Code() = %q, want %q", cerr.Code(), domain.CodeConflict)
	}
}

// ---------------------------------------------------------------------------
// SkillRepo.SelectBySlug / SelectBySlugAny (task 2.3)
// ---------------------------------------------------------------------------

// TestSkillRepo_SelectBySlug_ExcludesDeleted covers spec S-SK-038 +
// ADR-SK-003: soft-deleted skills MUST NOT be returned by
// SelectBySlug (active-only); SelectBySlugAny exists for the 410 path.
func TestSkillRepo_SelectBySlug_ExcludesDeleted(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()

	live := seedSkill(t, db, "live-skill", "d", "b")
	dead := seedSkill(t, db, "dead-skill", "d", "b")
	rawSoftDelete(t, db, dead.ID)

	got, err := repo.SelectBySlug(ctx, db, "live-skill")
	if err != nil {
		t.Fatalf("SelectBySlug(live): %v", err)
	}
	if got.ID != live.ID {
		t.Errorf("live ID = %d, want %d", got.ID, live.ID)
	}
	_, err = repo.SelectBySlug(ctx, db, "dead-skill")
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("SelectBySlug(deleted): expected *domain.NotFoundError, got %T (%v)", err, err)
	}
}

// TestSkillRepo_SelectBySlugAny_IncludesDeleted covers spec R-SK-005 +
// design §3.2 (PATCH flow): SelectBySlugAny returns the row regardless
// of deleted_at; service inspects DeletedAt to choose 404 vs 410.
func TestSkillRepo_SelectBySlugAny_IncludesDeleted(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()

	live := seedSkill(t, db, "alive", "d", "b")
	dead := seedSkill(t, db, "doomed", "d", "b")
	rawSoftDelete(t, db, dead.ID)

	got, err := repo.SelectBySlugAny(ctx, db, "alive")
	if err != nil {
		t.Fatalf("SelectBySlugAny(alive): %v", err)
	}
	if got.ID != live.ID || got.DeletedAt != nil {
		t.Errorf("alive: ID=%d (want %d), DeletedAt=%v (want nil)", got.ID, live.ID, got.DeletedAt)
	}
	got, err = repo.SelectBySlugAny(ctx, db, "doomed")
	if err != nil {
		t.Fatalf("SelectBySlugAny(doomed): %v", err)
	}
	if got.ID != dead.ID || got.DeletedAt == nil {
		t.Errorf("doomed: ID=%d (want %d), DeletedAt must be non-nil", got.ID, dead.ID)
	}
}

// ---------------------------------------------------------------------------
// SkillRepo.SelectByIDWithCurrentRevision + ListWithCurrentRevision
// (task 2.4 — ANTI-DRIFT GATE ADR-SK-008)
// ---------------------------------------------------------------------------

// TestSkillRepo_SelectByIDWithCurrentRevision_EmitsCurrentRevision
// covers spec S-SK-042 + ADR-SK-008: backend emits current_revision via
// SQL JOIN, kills the "v{undefined}" prompt bug. Skills with no
// revisions yield CurrentRevision=0 (COALESCE).
func TestSkillRepo_SelectByIDWithCurrentRevision_EmitsCurrentRevision(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()

	withRevs := seedSkill(t, db, "with-revs", "d", "b")
	seedSkillRevisions(t, db, withRevs.ID, 5)
	noRevs := seedSkill(t, db, "no-revs", "d", "b")

	got, err := repo.SelectByIDWithCurrentRevision(ctx, db, withRevs.ID)
	if err != nil {
		t.Fatalf("SelectByIDWithCurrentRevision(with-revs): %v", err)
	}
	if got.ID != withRevs.ID || got.CurrentRevision != 5 {
		t.Errorf("with-revs: ID=%d (want %d), CurrentRevision=%d (want 5)", got.ID, withRevs.ID, got.CurrentRevision)
	}
	got, err = repo.SelectByIDWithCurrentRevision(ctx, db, noRevs.ID)
	if err != nil {
		t.Fatalf("SelectByIDWithCurrentRevision(no-revs): %v", err)
	}
	if got.CurrentRevision != 0 {
		t.Errorf("no-revs.CurrentRevision = %d, want 0", got.CurrentRevision)
	}
}

// TestSkillRepo_ListWithCurrentRevision_EmitsCurrentRevisionOnAllRows
// covers spec S-SK-043 + ADR-SK-008: list endpoint emits
// current_revision for EVERY row, not just ones with revisions.
func TestSkillRepo_ListWithCurrentRevision_EmitsCurrentRevisionOnAllRows(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()

	skillA := seedSkill(t, db, "skill-a", "d", "b")
	seedSkillRevisions(t, db, skillA.ID, 5)
	skillB := seedSkill(t, db, "skill-b", "d", "b")
	seedSkillRevisions(t, db, skillB.ID, 1)
	seedSkill(t, db, "skill-c", "d", "b") // no revisions

	items, err := repo.ListWithCurrentRevision(ctx, db, 50)
	if err != nil {
		t.Fatalf("ListWithCurrentRevision: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len = %d, want 3", len(items))
	}
	byName := map[string]int{}
	for _, it := range items {
		byName[it.Name] = it.CurrentRevision
	}
	if byName["skill-a"] != 5 || byName["skill-b"] != 1 || byName["skill-c"] != 0 {
		t.Errorf("byName = %+v, want {skill-a:5, skill-b:1, skill-c:0}", byName)
	}
}

// ---------------------------------------------------------------------------
// N+1 guard (task 2.5)
// ---------------------------------------------------------------------------

// TestSkillRepo_N1Query_OneStatementForList covers design §7 R8: list
// MUST be exactly ONE SQL statement for any list size. countingExecutor
// wraps the SQLExecutor and counts QueryContext/QueryRowContext calls.
func TestSkillRepo_N1Query_OneStatementForList(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	for i := 0; i < 5; i++ {
		s := seedSkill(t, db, "n1-skill-"+string(rune('a'+i)), "d", "b")
		seedSkillRevisions(t, db, s.ID, i+1)
	}

	counter := &countingExecutor{inner: db}
	_, err := repo.ListWithCurrentRevision(context.Background(), counter, 50)
	if err != nil {
		t.Fatalf("ListWithCurrentRevision: %v", err)
	}
	if counter.queryCount != 1 {
		t.Errorf("ListWithCurrentRevision issued %d statements, want 1 (N+1 detected)", counter.queryCount)
	}
}

type countingExecutor struct {
	inner      domain.SQLExecutor
	queryCount int
}

func (c *countingExecutor) ExecContext(ctx context.Context, q string, args ...any) (sql.Result, error) {
	return c.inner.ExecContext(ctx, q, args...)
}
func (c *countingExecutor) QueryContext(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	c.queryCount++
	return c.inner.QueryContext(ctx, q, args...)
}
func (c *countingExecutor) QueryRowContext(ctx context.Context, q string, args ...any) *sql.Row {
	c.queryCount++
	return c.inner.QueryRowContext(ctx, q, args...)
}

// ---------------------------------------------------------------------------
// SkillRepo.MaxRevisionNumber (task 2.6)
// ---------------------------------------------------------------------------

// TestSkillRepo_MaxRevisionNumber_EmptyTableReturnsZero covers spec
// INV-4: a skill with no revisions returns 0 (so the service can
// assign revision_number=1 to the very first write under FOR UPDATE).
func TestSkillRepo_MaxRevisionNumber_EmptyTableReturnsZero(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	seeded := seedSkill(t, db, "max-rev-zero", "d", "b")
	got, err := repo.MaxRevisionNumber(context.Background(), db, seeded.ID)
	if err != nil {
		t.Fatalf("MaxRevisionNumber: %v", err)
	}
	if got != 0 {
		t.Errorf("MaxRevisionNumber = %d, want 0", got)
	}
}

// TestSkillRepo_MaxRevisionNumber_ReturnsMax covers the happy path:
// a skill with revisions 1..4 must report MaxRevisionNumber=4.
func TestSkillRepo_MaxRevisionNumber_ReturnsMax(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	seeded := seedSkill(t, db, "max-rev-existing", "d", "b")
	seedSkillRevisions(t, db, seeded.ID, 4)
	got, err := repo.MaxRevisionNumber(context.Background(), db, seeded.ID)
	if err != nil {
		t.Fatalf("MaxRevisionNumber: %v", err)
	}
	if got != 4 {
		t.Errorf("MaxRevisionNumber = %d, want 4", got)
	}
}

// ---------------------------------------------------------------------------
// SkillRepo.LockAndLoad + UpdateBody (task 2.7)
// ---------------------------------------------------------------------------

// TestSkillRepo_LockAndLoad_HoldsRowLock covers spec S-SK-039 +
// design §3.5: SELECT … FOR UPDATE holds a row lock that blocks a
// concurrent UPDATE (with statement_timeout=500ms). The lock is the
// concurrency gate for Update, Restore, and SoftDelete.
func TestSkillRepo_LockAndLoad_HoldsRowLock(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	seeded := seedSkill(t, db, "lock-and-load", "d", "b")
	ctx := context.Background()

	tx1, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx1: %v", err)
	}
	defer func() { _ = tx1.Rollback() }()

	loaded, err := repo.LockAndLoad(ctx, tx1, seeded.ID)
	if err != nil || loaded.ID != seeded.ID {
		t.Fatalf("LockAndLoad: err=%v, id=%d", err, loaded.ID)
	}

	tx2, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx2: %v", err)
	}
	defer func() { _ = tx2.Rollback() }()
	if _, err := tx2.ExecContext(ctx, "SET LOCAL statement_timeout = '500ms'"); err != nil {
		t.Fatalf("SET LOCAL statement_timeout: %v", err)
	}
	_, err = tx2.ExecContext(ctx, "UPDATE skill SET body = 'blocked' WHERE id = $1", seeded.ID)
	if err == nil {
		t.Errorf("expected second TX to block on FOR UPDATE; got nil error")
	} else if !strings.Contains(strings.ToLower(err.Error()), "cancel") &&
		!strings.Contains(strings.ToLower(err.Error()), "timeout") &&
		!strings.Contains(strings.ToLower(err.Error()), "lock") {
		t.Logf("second TX error: %v (acceptable if lock-related)", err)
	}
}

// TestSkillRepo_UpdateBody_PersistsFieldsAndUpdatesTimestamp covers
// spec R-SK-001 + design §3.5: UpdateBody writes new description +
// body and bumps updated_at from the DB clock.
func TestSkillRepo_UpdateBody_PersistsFieldsAndUpdatesTimestamp(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()
	seeded := seedSkill(t, db, "update-body", "original desc", "original body")
	originalUpdatedAt := seeded.UpdatedAt

	time.Sleep(10 * time.Millisecond)
	if err := repo.UpdateBody(ctx, db, seeded.ID, "new body", "new desc"); err != nil {
		t.Fatalf("UpdateBody: %v", err)
	}
	got, err := repo.SelectByID(ctx, db, seeded.ID)
	if err != nil {
		t.Fatalf("SelectByID: %v", err)
	}
	if got.Body != "new body" || got.Description != "new desc" {
		t.Errorf("Body=%q (want new body), Description=%q (want new desc)", got.Body, got.Description)
	}
	if !got.UpdatedAt.After(originalUpdatedAt) {
		t.Errorf("UpdatedAt = %v, want > %v", got.UpdatedAt, originalUpdatedAt)
	}
}

// ---------------------------------------------------------------------------
// SkillRepo.SoftDelete (task 2.8)
// ---------------------------------------------------------------------------

// TestSkillRepo_SoftDelete_SetsDeletedAt covers spec SCN-2.1:
// SoftDelete writes deleted_at = now() to the row.
func TestSkillRepo_SoftDelete_SetsDeletedAt(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()
	seeded := seedSkill(t, db, "soft-delete", "d", "b")
	if err := repo.SoftDelete(ctx, db, seeded.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	var deletedAt *time.Time
	if err := db.QueryRowContext(ctx, "SELECT deleted_at FROM skill WHERE id = $1", seeded.ID).Scan(&deletedAt); err != nil {
		t.Fatalf("raw query: %v", err)
	}
	if deletedAt == nil {
		t.Fatalf("SoftDelete did not set deleted_at")
	}
}

// TestSkillRepo_SoftDelete_Idempotent covers spec SCN-2.3: calling
// SoftDelete twice on the same row succeeds both times.
func TestSkillRepo_SoftDelete_Idempotent(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()
	seeded := seedSkill(t, db, "soft-delete-twice", "d", "b")
	if err := repo.SoftDelete(ctx, db, seeded.ID); err != nil {
		t.Fatalf("first SoftDelete: %v", err)
	}
	if err := repo.SoftDelete(ctx, db, seeded.ID); err != nil {
		t.Fatalf("second SoftDelete (idempotent): %v", err)
	}
}
