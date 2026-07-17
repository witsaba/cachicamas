// Package application_test — integration tests for SkillService
// (PR1c of cachicamas-skills-foundational). Mirrors
// prompt_service_test.go's shape (helper duplication intentional to
// avoid cross-package coupling). Tests are gated on INTEGRATION=1
// because they need a live Postgres.
package application_test

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/skills"
)

// ---------------------------------------------------------------------------
// Helpers (mirrored from prompt_service_test.go; kept in lock-step
// to avoid surprising divergence).
// ---------------------------------------------------------------------------

func openSkillAppTestDB(t *testing.T) *sql.DB {
	t.Helper()
	host := getenvOrSkill("POSTGRES_HOST", "localhost")
	port := getenvOrSkill("POSTGRES_PORT", "5432")
	user := getenvOrSkill("POSTGRES_USER", "queen")
	pass := getenvOrSkill("QUEEN_PASSWORD", "changeme-queen")
	dbname := getenvOrSkill("POSTGRES_DB", "cachicamas_pg")
	sslmode := getenvOrSkill("POSTGRES_SSLMODE", "disable")
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

func getenvOrSkill(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

// ensureSkillAppMigrations applies the PR1a skill + skill_revision
// schema idempotently. Mirrors the migration shipped by PR1a; safe
// to call repeatedly because every statement uses IF NOT EXISTS.
func ensureSkillAppMigrations(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS skill (
		    id BIGSERIAL PRIMARY KEY,
		    name TEXT NOT NULL
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
			t.Fatalf("ensureSkillAppMigrations: %v\nSQL: %s", err, s)
		}
	}
}

// cleanSkillAppTables wipes both tables; runs BEFORE each test for
// isolation.
func cleanSkillAppTables(t *testing.T, db *sql.DB) {
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

func skipIfNoIntegrationSkill(t *testing.T) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
}

func newSkillAppService(t *testing.T, db *sql.DB) *application.SkillService {
	t.Helper()
	repo := skills.NewSkillRepo(db)
	revRepo := skills.NewSkillRevisionRepo(db)
	return application.NewSkillService(repo, revRepo, db, nil)
}

// validSkillBody returns a SKILL.md body that passes all service-layer
// validators (frontmatter name + description match the slug + the
// provided description). Helps keep test bodies short.
func validSkillBody(name, description string) string {
	return "---\nname: " + name + "\ndescription: " + description + "\n---\n\n# " + name + "\n"
}

// ---------------------------------------------------------------------------
// Task 3.1 — Create writes revision_number = 1.
// ---------------------------------------------------------------------------

func TestSkillService_Create_WritesRevisionOne(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	s, rev, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "create-rev-1",
		Description: "First revision",
		Body:        validSkillBody("create-rev-1", "First revision"),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if s == nil || s.ID == 0 {
		t.Fatalf("skill not populated: %+v", s)
	}
	if rev == nil || rev.RevisionNumber != 1 {
		t.Fatalf("revision != 1: %+v", rev)
	}
	if rev.Body == "" || rev.Description == "" {
		t.Errorf("revision snapshot fields empty: %+v", rev)
	}
	if rev.Description != "First revision" {
		t.Errorf("rev.Description = %q, want %q", rev.Description, "First revision")
	}
}

// ---------------------------------------------------------------------------
// Task 3.2 — Create rejects invalid + reserved names.
// ---------------------------------------------------------------------------

func TestSkillService_Create_RejectsInvalidName(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	cases := []struct {
		name string
		bad  string
	}{
		{"uppercase", "BadName"},
		{"leading-hyphen", "-leading"},
		{"trailing-hyphen", "trailing-"},
		{"consecutive-hyphens", "foo--bar"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := svc.Create(context.Background(), domain.CreateSkillInput{
				Name:        tc.bad,
				Description: "desc",
				Body:        validSkillBody(tc.bad, "desc"),
			})
			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected *ValidationError for name=%q, got %T (%v)", tc.bad, err, err)
			}
		})
	}
}

func TestSkillService_Create_RejectsReservedName(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	cases := []string{
		"anthropic-toolkit",
		"claude-helper",
		"my-anthropic-skill",
		"claudeCode",
	}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := svc.Create(context.Background(), domain.CreateSkillInput{
				Name:        name,
				Description: "desc",
				Body:        validSkillBody(name, "desc"),
			})
			var verr *domain.ValidationError
			if !errors.As(err, &verr) {
				t.Fatalf("expected *ValidationError for reserved name=%q, got %T (%v)", name, err, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 3.3 — Create rejects long description, oversize body,
// missing frontmatter, and frontmatter-vs-request lock-step mismatches.
// ---------------------------------------------------------------------------

func TestSkillService_Create_RejectsLongDescription(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	long := strings.Repeat("x", 1025)
	_, _, err := svc.Create(context.Background(), domain.CreateSkillInput{
		Name:        "long-desc",
		Description: long,
		Body:        validSkillBody("long-desc", "ok"),
	})
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T (%v)", err, err)
	}
}

func TestSkillService_Create_RejectsOversizeBody(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	huge := strings.Repeat("x", 524289)
	_, _, err := svc.Create(context.Background(), domain.CreateSkillInput{
		Name:        "oversize-body",
		Description: "ok",
		Body:        huge,
	})
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError, got %T (%v)", err, err)
	}
}

func TestSkillService_Create_RejectsMissingFrontmatter(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	// No leading "---\n" — body is bare markdown.
	_, _, err := svc.Create(context.Background(), domain.CreateSkillInput{
		Name:        "no-frontmatter",
		Description: "ok",
		Body:        "# No frontmatter here\n",
	})
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError for missing frontmatter, got %T (%v)", err, err)
	}
}

func TestSkillService_Create_RejectsNameLockStep(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	// Frontmatter name "other-name" differs from slug "mismatched".
	body := "---\nname: other-name\ndescription: ok\n---\nbody\n"
	_, _, err := svc.Create(context.Background(), domain.CreateSkillInput{
		Name:        "mismatched",
		Description: "ok",
		Body:        body,
	})
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError for name lock-step, got %T (%v)", err, err)
	}
}

func TestSkillService_Create_RejectsDescriptionLockStep(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	// Request description differs from frontmatter description.
	body := "---\nname: desc-locked\ndescription: frontmatter-desc\n---\nbody\n"
	_, _, err := svc.Create(context.Background(), domain.CreateSkillInput{
		Name:        "desc-locked",
		Description: "request-desc",
		Body:        body,
	})
	var verr *domain.ValidationError
	if !errors.As(err, &verr) {
		t.Fatalf("expected *ValidationError for description lock-step, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// Task 3.4 — Duplicate active name → *ConflictError (409).
// ---------------------------------------------------------------------------

func TestSkillService_Create_DuplicateName_ReturnsConflictError(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "dup-name",
		Description: "first",
		Body:        validSkillBody("dup-name", "first"),
	}); err != nil {
		t.Fatalf("first Create: %v", err)
	}
	_, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "dup-name",
		Description: "second",
		Body:        validSkillBody("dup-name", "second"),
	})
	var cerr *domain.ConflictError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *ConflictError on dup, got %T (%v)", err, err)
	}
}

// _ = sync / atomic are used by later concurrency tests; referencing
// them here avoids unused-import errors as the file grows.
var (
	_ = sync.WaitGroup{}
	_ = atomic.AddInt32
	_ = errors.As
)