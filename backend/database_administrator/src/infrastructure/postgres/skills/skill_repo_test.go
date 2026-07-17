// Package skills_test contains the integration test suite for the
// skill + skill_revision repository adapters (PR1b of
// cachicamas-skills-foundational). Tests are gated on INTEGRATION=1
// because they need a live Postgres (the compose-provisioned instance
// used by the migration runner). Mirrors the prompts adapter pattern
// at prompts/prompt_repo_test.go (no test fixtures; each test seeds
// its own state).
//
// Strict TDD discipline (per sdd-apply/strict-tdd.md): every test
// here was written BEFORE the corresponding SkillRepo /
// SkillRevisionRepo method existed. The first commit in the PR
// (the RED commit) references types and methods that do not exist
// yet, so `go build ./...` fails — that build failure IS the RED
// step. The next commit (GREEN) adds the production code that
// resolves the build failure and makes the test pass.
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

// ---------------------------------------------------------------------------
// Connection helper.
// ---------------------------------------------------------------------------

// openTestDB connects to the compose Postgres instance using the
// environment variables the project standardizes on. Mirrors
// prompts/prompt_repo_test.go:openTestDB.
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
// The compose DB already has the migration in schema_migrations; this
// helper only ensures the skill + skill_revision tables exist. CREATE
// TABLE IF NOT EXISTS makes the helper safe to call repeatedly.
func ensureSkillMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS skill (
		    id          BIGSERIAL    PRIMARY KEY,
		    name        TEXT         NOT NULL
		                 CHECK (name ~ '^[a-z0-9]+(-[a-z0-9]+)*$' AND length(name) BETWEEN 1 AND 64),
		    description TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 1024),
		    body        TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
		    deleted_at  TIMESTAMPTZ  NULL,
		    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS skill_revision (
		    id              BIGSERIAL    PRIMARY KEY,
		    skill_id        BIGINT       NOT NULL REFERENCES skill(id) ON DELETE CASCADE,
		    revision_number INT          NOT NULL CHECK (revision_number > 0),
		    description     TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 1024),
		    body            TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
		    change_note     TEXT         NULL,
		    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    CONSTRAINT skill_revision_unique UNIQUE (skill_id, revision_number)
		)`,
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

// cleanSkillTables wipes skill + skill_revision in FK-safe order.
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

// makeSkillInput returns a valid *domain.Skill for use in tests.
func makeSkillInput(name, description, body string) *domain.Skill {
	return &domain.Skill{
		Name:        name,
		Description: description,
		Body:        body,
	}
}

// seedSkill inserts one skill row directly (bypassing repo.Insert)
// for tests that need a pre-existing row.
func seedSkill(t *testing.T, db *sql.DB, name, description, body string) *domain.Skill {
	t.Helper()
	s := makeSkillInput(name, description, body)
	if err := skills.NewSkillRepo(db).Insert(context.Background(), db, s); err != nil {
		t.Fatalf("seed skill: %v", err)
	}
	return s
}

// skipIfNoIntegration skips the test if INTEGRATION != "1".
func skipIfNoIntegration(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
}

// ---------------------------------------------------------------------------
// SkillRepo.Insert
// ---------------------------------------------------------------------------

// TestSkillRepo_Insert_PersistsRowAndReturnsID covers spec S-SK-001 +
// design §3.6: SkillRepo.Insert persists a new skill row and returns
// a populated *Skill with non-zero ID, non-zero CreatedAt, non-zero
// UpdatedAt, and nil DeletedAt (per the DB DEFAULT). This is the
// minimal "happy path" RED/GREEN cycle for task 2.1.
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
	if s.ID == 0 {
		t.Fatalf("Insert did not set ID: %+v", s)
	}
	if s.CreatedAt.IsZero() {
		t.Fatalf("Insert did not set CreatedAt: %+v", s)
	}
	if s.UpdatedAt.IsZero() {
		t.Fatalf("Insert did not set UpdatedAt: %+v", s)
	}
	if s.DeletedAt != nil {
		t.Fatalf("Insert set DeletedAt (should be nil): %+v", s)
	}
}

// TestSkillRepo_Insert_TranslatesUniqueViolation covers spec S-SK-037
// + ADR-SK-003: a second INSERT against the partial UNIQUE index
// `skill_slug_active_uidx` must surface as *domain.ConflictError
// (with Code() == "conflict") so the handler can map to HTTP 409
// without importing pgx. This locks the pgconn 23505 → ConflictError
// translation at the infra boundary.
func TestSkillRepo_Insert_TranslatesUniqueViolation(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()
	if err := repo.Insert(ctx, db, makeSkillInput("dup-name", "first", "body one")); err != nil {
		t.Fatalf("first Insert: %v", err)
	}
	err := repo.Insert(ctx, db, makeSkillInput("dup-name", "second", "body two"))
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

// TestSkillRepo_SelectBySlug_ExcludesDeleted covers spec S-SK-038 +
// ADR-SK-003: a soft-deleted skill MUST NOT be returned by
// SelectBySlug (active-only). SelectBySlugAny exists to bridge the
// "still exists but soft-deleted" → *GoneError (410) flow in the
// service layer; SelectBySlug returns *NotFoundError on the same row.
func TestSkillRepo_SelectBySlug_ExcludesDeleted(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()

	seeded := seedSkill(t, db, "live-skill", "active", "body")
	deleted := seedSkill(t, db, "dead-skill", "active", "body")
	if err := repo.SoftDelete(ctx, db, deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	// Live row is returned.
	got, err := repo.SelectBySlug(ctx, db, "live-skill")
	if err != nil {
		t.Fatalf("SelectBySlug(live-skill): %v", err)
	}
	if got.ID != seeded.ID {
		t.Errorf("live ID = %d, want %d", got.ID, seeded.ID)
	}

	// Soft-deleted row is hidden.
	_, err = repo.SelectBySlug(ctx, db, "dead-skill")
	var nerr *domain.NotFoundError
	if !errors.As(err, &nerr) {
		t.Fatalf("SelectBySlug(deleted): expected *domain.NotFoundError, got %T (%v)", err, err)
	}
}

// TestSkillRepo_SelectBySlugAny_IncludesDeleted covers spec S-SK-005 +
// design §3.2 (PATCH flow): the service needs to distinguish
// "slug never existed" (→ 404) from "slug exists but is soft-deleted"
// (→ 410 skill_deleted). SelectBySlugAny returns the row regardless
// of deleted_at state and the service inspects DeletedAt.
func TestSkillRepo_SelectBySlugAny_IncludesDeleted(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)
	ctx := context.Background()

	live := seedSkill(t, db, "alive", "active", "body")
	deleted := seedSkill(t, db, "doomed", "active", "body")
	if err := repo.SoftDelete(ctx, db, deleted.ID); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}

	// Live row.
	got, err := repo.SelectBySlugAny(ctx, db, "alive")
	if err != nil {
		t.Fatalf("SelectBySlugAny(alive): %v", err)
	}
	if got.ID != live.ID {
		t.Errorf("alive ID = %d, want %d", got.ID, live.ID)
	}
	if got.DeletedAt != nil {
		t.Errorf("alive.DeletedAt = %v, want nil", got.DeletedAt)
	}

	// Soft-deleted row is returned with DeletedAt set; the service
	// uses this to emit 410 instead of 404.
	got, err = repo.SelectBySlugAny(ctx, db, "doomed")
	if err != nil {
		t.Fatalf("SelectBySlugAny(doomed): %v", err)
	}
	if got.ID != deleted.ID {
		t.Errorf("doomed ID = %d, want %d", got.ID, deleted.ID)
	}
	if got.DeletedAt == nil {
		t.Errorf("doomed.DeletedAt must be non-nil for soft-deleted row")
	}
}

// _ = strings.Contains is used so an unused-import regression fails
// the build deterministically if a future refactor drops it.
var _ = strings.Contains
