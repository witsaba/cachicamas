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
	// Soft-delete via raw SQL — SoftDelete is implemented in task 2.8.
	if _, err := db.ExecContext(ctx, "UPDATE skill SET deleted_at = now() WHERE id = $1", deleted.ID); err != nil {
		t.Fatalf("raw soft-delete: %v", err)
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
	// Soft-delete via raw SQL — SoftDelete is implemented in task 2.8.
	if _, err := db.ExecContext(ctx, "UPDATE skill SET deleted_at = now() WHERE id = $1", deleted.ID); err != nil {
		t.Fatalf("raw soft-delete: %v", err)
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

// seedSkillRevisions inserts n revisions (1..n) directly via raw SQL
// for the given skill. Used by tests that need a revision history
// without depending on SkillRevisionRepo.Insert (which lands in
// task 2.9). Mirrors the seedPrompt helper pattern.
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

// TestSkillRepo_SelectByIDWithCurrentRevision_EmitsCurrentRevision
// covers spec S-SK-042 + ADR-SK-008 (ANITI-DRIFT GATE from
// obs #1959 #2 — backend emits current_revision via SQL JOIN, kills
// the "v{undefined}" bug). A skill with revisions 1..5 must return
// CurrentRevision=5 from SelectByIDWithCurrentRevision; a skill with
// NO revisions returns CurrentRevision=0 (or whatever COALESCE yields).
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

	// Skill with revisions: CurrentRevision = 5.
	got, err := repo.SelectByIDWithCurrentRevision(ctx, db, withRevs.ID)
	if err != nil {
		t.Fatalf("SelectByIDWithCurrentRevision(with-revs): %v", err)
	}
	if got.ID != withRevs.ID {
		t.Errorf("ID = %d, want %d", got.ID, withRevs.ID)
	}
	if got.CurrentRevision != 5 {
		t.Errorf("CurrentRevision = %d, want 5", got.CurrentRevision)
	}

	// Skill without revisions: CurrentRevision = 0 (COALESCE on MAX).
	got, err = repo.SelectByIDWithCurrentRevision(ctx, db, noRevs.ID)
	if err != nil {
		t.Fatalf("SelectByIDWithCurrentRevision(no-revs): %v", err)
	}
	if got.CurrentRevision != 0 {
		t.Errorf("CurrentRevision = %d, want 0 (no revisions yet)", got.CurrentRevision)
	}
}

// TestSkillRepo_ListWithCurrentRevision_EmitsCurrentRevisionOnAllRows
// covers spec S-SK-043 + ADR-SK-008: the list endpoint must emit
// current_revision for EVERY row, not just rows that happen to have
// revisions. The fixture has 3 skills: skillA (5 revs), skillB (1
// rev), skillC (no revs). The list query must return 3 items, each
// with the right CurrentRevision.
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
	// Build a map by name for stable assertions regardless of order.
	byName := map[string]int{}
	for _, it := range items {
		byName[it.Name] = it.CurrentRevision
	}
	if byName["skill-a"] != 5 {
		t.Errorf("skill-a.CurrentRevision = %d, want 5", byName["skill-a"])
	}
	if byName["skill-b"] != 1 {
		t.Errorf("skill-b.CurrentRevision = %d, want 1", byName["skill-b"])
	}
	if byName["skill-c"] != 0 {
		t.Errorf("skill-c.CurrentRevision = %d, want 0 (no revisions)", byName["skill-c"])
	}
}

// TestSkillRepo_N1Query_OneStatementForList covers design §7 R8
// (anti-drift): ListWithCurrentRevision MUST issue exactly ONE SQL
// statement for any list size. The test uses pgx's pgx_stat_statements
// proxy via a counter wrapper: every QueryContext call increments a
// counter, and the assertion is that one List() call = one statement.
//
// To avoid introducing a heavy tracing dependency, we use a
// sentinel-driver trick: open a *sql.DB with a custom query observer
// via the `database/sql` driver's instrumentation. The simplest
// portable approach: capture the call count by wrapping the executor
// passed to the repo. We wrap the db's QueryContext via a thin
// proxy struct that the repo accepts as domain.SQLExecutor.
func TestSkillRepo_N1Query_OneStatementForList(t *testing.T) {
	skipIfNoIntegration(t)
	db := openTestDB(t)
	defer func() { _ = db.Close() }()
	ensureSkillMigrations(t, db)
	cleanSkillTables(t, db)

	repo := skills.NewSkillRepo(db)

	// Seed 5 skills, each with a few revisions.
	for i := 0; i < 5; i++ {
		name := "n1-skill-" + string(rune('a'+i))
		s := seedSkill(t, db, name, "d", "b")
		seedSkillRevisions(t, db, s.ID, i+1) // 1..5 revs
	}

	// Wrap the DB in a counter proxy.
	counter := &countingExecutor{inner: db, queryCount: 0}

	_, err := repo.ListWithCurrentRevision(context.Background(), counter, 50)
	if err != nil {
		t.Fatalf("ListWithCurrentRevision: %v", err)
	}
	if counter.queryCount != 1 {
		t.Errorf("ListWithCurrentRevision issued %d statements, want 1 (N+1 detected)", counter.queryCount)
	}
}

// countingExecutor wraps a domain.SQLExecutor and counts every
// QueryContext and QueryRowContext call. Domain.SQLExecutor is the
// repo's input; we implement it on *sql.DB by re-exposing the same
// surface. We delegate ExecContext (which the repo also uses for
// inserts) but don't count it for the N+1 assertion — N+1 manifests
// as per-row reads, not writes.
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

// TestSkillRepo_MaxRevisionNumber_EmptyTableReturnsZero covers spec
// INV-4 + design §3.2: a skill with no revisions returns 0 from
// MaxRevisionNumber (so the service can assign revision_number=1 to
// the very first write under FOR UPDATE).
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
// a skill with revisions 1..4 must report MaxRevisionNumber=4 (used
// by the service to assign revision_number=5 on the next write).
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

// _ = strings.Contains is used so an unused-import regression fails
// the build deterministically if a future refactor drops it.
var _ = strings.Contains
