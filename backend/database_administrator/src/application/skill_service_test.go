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

// ---------------------------------------------------------------------------
// Task 3.5 — Update appends next revision (rev 3 → rev 4).
// ---------------------------------------------------------------------------

func TestSkillService_Update_AppendsNextRevision(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "update-rev",
		Description: "d1",
		Body:        validSkillBody("update-rev", "d1"),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, _, err := svc.Update(ctx, "update-rev", domain.UpdateSkillInput{
		Body: stringPtrSkill(validSkillBody("update-rev", "d2")),
	}); err != nil {
		t.Fatalf("Update 1: %v", err)
	}
	sk, rev, err := svc.Update(ctx, "update-rev", domain.UpdateSkillInput{
		Body:        stringPtrSkill(validSkillBody("update-rev", "d3")),
		Description: stringPtrSkill("d3"),
	})
	if err != nil {
		t.Fatalf("Update 2: %v", err)
	}
	if rev.RevisionNumber != 3 {
		t.Errorf("revision = %d, want 3", rev.RevisionNumber)
	}
	if sk.Description != "d3" {
		t.Errorf("skill.description = %q, want %q", sk.Description, "d3")
	}
}

// ---------------------------------------------------------------------------
// Task 3.6 — Update description-only also appends a revision.
// ---------------------------------------------------------------------------

func TestSkillService_Update_DescriptionOnly_AppendsRevision(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "update-desc-only",
		Description: "orig-desc",
		Body:        validSkillBody("update-desc-only", "orig-desc"),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	sk, rev, err := svc.Update(ctx, "update-desc-only", domain.UpdateSkillInput{
		Description: stringPtrSkill("new-desc"),
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rev.RevisionNumber != 2 {
		t.Errorf("revision = %d, want 2 (desc-only must still append)", rev.RevisionNumber)
	}
	if sk.Description != "new-desc" {
		t.Errorf("description = %q, want %q", sk.Description, "new-desc")
	}
	// Body carried forward — should be unchanged from creation.
	if sk.Body != validSkillBody("update-desc-only", "orig-desc") {
		t.Errorf("body not carried forward; got %q", sk.Body)
	}
}

// ---------------------------------------------------------------------------
// Task 3.7 — Update / Restore on a soft-deleted skill return GoneError.
// ---------------------------------------------------------------------------

func TestSkillService_Update_DeletedSkill_ReturnsGoneError(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "update-deleted",
		Description: "d",
		Body:        validSkillBody("update-deleted", "d"),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "update-deleted"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	_, _, err := svc.Update(ctx, "update-deleted", domain.UpdateSkillInput{
		Body: stringPtrSkill(validSkillBody("update-deleted", "d")),
	})
	if _, ok := domain.AsSkillDeleted(err); !ok {
		t.Fatalf("expected *SkillGoneError, got %T (%v)", err, err)
	}
}

func TestSkillService_Restore_OnDeletedSkill_ReturnsGoneError(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "restore-deleted",
		Description: "d",
		Body:        validSkillBody("restore-deleted", "d"),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "restore-deleted"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	_, _, err := svc.Restore(ctx, "restore-deleted", 1)
	if _, ok := domain.AsSkillDeleted(err); !ok {
		t.Fatalf("expected *SkillGoneError, got %T (%v)", err, err)
	}
}

// ---------------------------------------------------------------------------
// Task 3.8 — Restore appends a NEW revision copying historical body +
// description, with change_note = "restored from revision N".
// ---------------------------------------------------------------------------

func TestSkillService_Restore_AppendsNewRevisionWithHistoricalBody(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "restore-body",
		Description: "v1-desc",
		Body:        validSkillBody("restore-body", "v1-desc"),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	for _, desc := range []string{"v2-desc", "v3-desc", "v4-desc", "v5-desc"} {
		if _, _, err := svc.Update(ctx, "restore-body", domain.UpdateSkillInput{
			Body: stringPtrSkill(validSkillBody("restore-body", desc)),
		}); err != nil {
			t.Fatalf("Update %s: %v", desc, err)
		}
	}
	sk, rev, err := svc.Restore(ctx, "restore-body", 2)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if rev.RevisionNumber != 6 {
		t.Errorf("revision = %d, want 6 (new revision appended, history preserved)", rev.RevisionNumber)
	}
	if rev.Description != "v2-desc" {
		t.Errorf("rev.Description = %q, want %q", rev.Description, "v2-desc")
	}
	if rev.ChangeNote == nil {
		t.Fatalf("ChangeNote must be set on a restore revision")
	}
	if *rev.ChangeNote != "restored from revision 2" {
		t.Errorf("ChangeNote = %q, want %q", *rev.ChangeNote, "restored from revision 2")
	}
	if sk.Description != "v2-desc" {
		t.Errorf("skill.Description = %q, want %q", sk.Description, "v2-desc")
	}
	// Revision 2 must still be intact; history is preserved.
	hist, err := svc.ListRevisions(ctx, "restore-body")
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	if len(hist) != 6 {
		t.Fatalf("len(hist) = %d, want 6", len(hist))
	}
	// hist is DESC; index 4 is revision 2.
	if hist[4].RevisionNumber != 2 {
		t.Errorf("hist[4].RevisionNumber = %d, want 2", hist[4].RevisionNumber)
	}
}

// ---------------------------------------------------------------------------
// Task 3.9 — SoftDelete is idempotent; soft-deleted name can be reused.
// ---------------------------------------------------------------------------

func TestSkillService_SoftDelete_Idempotent(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	if _, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "soft-del-idempotent",
		Description: "d",
		Body:        validSkillBody("soft-del-idempotent", "d"),
	}); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "soft-del-idempotent"); err != nil {
		t.Fatalf("first SoftDelete: %v", err)
	}
	// Second call on an already-deleted name: must return nil.
	if err := svc.SoftDelete(ctx, "soft-del-idempotent"); err != nil {
		t.Fatalf("second SoftDelete (idempotent): %v", err)
	}
}

func TestSkillService_SoftDelete_ThenRecreate_AllowsReuse(t *testing.T) {
	skipIfNoIntegrationSkill(t)
	db := openSkillAppTestDB(t)
	defer db.Close()
	ensureSkillAppMigrations(t, db)
	cleanSkillAppTables(t, db)

	svc := newSkillAppService(t, db)
	ctx := context.Background()
	first, _, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "reusable",
		Description: "d1",
		Body:        validSkillBody("reusable", "d1"),
	})
	if err != nil {
		t.Fatalf("first Create: %v", err)
	}
	if err := svc.SoftDelete(ctx, "reusable"); err != nil {
		t.Fatalf("SoftDelete: %v", err)
	}
	second, rev2, err := svc.Create(ctx, domain.CreateSkillInput{
		Name:        "reusable",
		Description: "d2",
		Body:        validSkillBody("reusable", "d2"),
	})
	if err != nil {
		t.Fatalf("recreate: %v", err)
	}
	if second.ID == first.ID {
		t.Errorf("recreated skill has same ID as deleted: %d", second.ID)
	}
	if rev2.RevisionNumber != 1 {
		t.Errorf("recreated revision = %d, want 1", rev2.RevisionNumber)
	}
}

// stringPtrSkill returns &s for use in *string fields (UpdateSkillInput).
func stringPtrSkill(s string) *string { return &s }
var (
	_ = sync.WaitGroup{}
	_ = atomic.AddInt32
	_ = errors.As
)