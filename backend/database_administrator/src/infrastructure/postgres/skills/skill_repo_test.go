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

// _ = strings.Contains is used so an unused-import regression fails
// the build deterministically if a future refactor drops it.
var _ = strings.Contains
