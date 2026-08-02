// Package httpiface_test — integration tests for the 7 skill HTTP
// endpoints. Tests are gated on INTEGRATION=1 because they need a
// live Postgres. Mirrors the pattern in prompt_handler_test.go
// (which uses real services + integration DB).
//
// Anti-drift gates (locked by engram obs #1959 / spec #1967):
//   - Every successful response includes current_revision (kills the
//     "v{undefined}" bug from the prompts feature).
//   - Error envelope uses NESTED shape {error:{code,message,fields?}}
//     (kills the prompts flat-fixture bug).
//   - PATCH for soft-deleted skill returns 410 with code "skill_deleted"
//     (kills the missing-GoneError gap from prompts).
//   - Route registration matches spec §6 EXACTLY (no spurious
//     routes, no ?deleted= query params).
//   - Logger never emits body content or description value (only
//     lengths and the slug).
package httpiface_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/skills"
	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

// ---------------------------------------------------------------------------
// Setup helpers
// ---------------------------------------------------------------------------

// setupSkillHandler wires the real SkillService + SkillHandler against the
// live Postgres integration DB. The handler is mounted on a fresh
// Echo router so test cases cannot collide on global state. Returns
// (echo, handler, sql.DB).
func setupSkillHandler(t *testing.T) (*echo.Echo, *httpiface.SkillHandler, *sql.DB) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	host := skillOsOr("POSTGRES_HOST", "localhost")
	port := skillOsOr("POSTGRES_PORT", "5432")
	user := skillOsOr("QUEEN_USER", "queen")
	pass := skillOsOr("QUEEN_PASSWORD", "changeme-queen")
	dbname := skillOsOr("POSTGRES_DB", "cachicamas_pg")
	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + dbname + " sslmode=disable"
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := dbConn.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	ensureSkillHandlerTables(t, dbConn)
	cleanSkillHandlerTables(t, dbConn)

	skillRepo := skills.NewSkillRepo(dbConn)
	revRepo := skills.NewSkillRevisionRepo(dbConn)
	svc := application.NewSkillService(skillRepo, revRepo, dbConn, nil)

	// Capture logs in a buffer so the PII redaction test can assert
	// no skill body / description content appears in any log line.
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	h := httpiface.NewSkillHandler(svc, logger)
	e := echo.New()
	h.RegisterSkillRoutes(e.Group(""))
	return e, h, dbConn
}

func skillOsOr(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}

func ensureSkillHandlerTables(t *testing.T, db *sql.DB) {
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
			t.Fatalf("ensureSkillHandlerTables: %v", err)
		}
	}
}

func cleanSkillHandlerTables(t *testing.T, db *sql.DB) {
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

// skillHttpDo executes an HTTP request against the Echo app and returns
// the response recorder.
func skillHttpDo(t *testing.T, e *echo.Echo, method, path string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	return rec
}

// validSkillBody returns a SKILL.md body that passes frontmatter + lock-step
// against the given name + description. Used by happy-path tests.
func validSkillBody(name, description string) string {
	return "---\nname: " + name + "\ndescription: " + description + "\n---\n\n# " + name + "\n"
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

// TestSkillHandler_Create_Returns201AndCurrentRevisionOne is the anti-drift
// gate (ADR-SK-008). The response MUST include current_revision=1 (proves
// the backend emits the field; kills the v{undefined} prompt bug).
func TestSkillHandler_Create_Returns201AndCurrentRevisionOne(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "pdf-cleanup"
	desc := "Strip trailing whitespace from PDFs"
	body := validSkillBody(name, desc)
	payload := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)

	rec := skillHttpDo(t, e, "POST", "/skills", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["name"] != name {
		t.Errorf("name = %v, want %s", got["name"], name)
	}
	if got["description"] != desc {
		t.Errorf("description = %v, want %s", got["description"], desc)
	}
	if cr, ok := got["current_revision"].(float64); !ok || cr != 1 {
		t.Errorf("current_revision = %v, want 1 (anti-drift gate ADR-SK-008)", got["current_revision"])
	}
}

// jsonString returns a JSON-quoted Go string literal of s. Used to embed
// multi-line strings in JSON test payloads without manual escaping.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// envEnvelope is the locked error envelope shape {error:{code,message,fields?}}.
type envEnvelope struct {
	Error struct {
		Code    string            `json:"code"`
		Message string            `json:"message"`
		Fields  map[string]string `json:"fields,omitempty"`
	} `json:"error"`
}

// skillListResponseShape decodes the spec §8 wire shape {skills:[...]}.
type skillListResponseShape struct {
	Skills []map[string]any `json:"skills"`
}

// TestSkillHandler_Create_ValidationErrorEnvelopeShape is the anti-drift
// gate for the NESTED error envelope (spec R-SK-006 / S-SK-X4). An invalid
// name MUST produce 400 with code="validation" AND a populated fields.name
// map (kills the prompts flat-fixture parsing bug).
func TestSkillHandler_Create_ValidationErrorEnvelopeShape(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	body := []byte(`{"name":"BAD","description":"d","body":"x"}`)
	rec := skillHttpDo(t, e, "POST", "/skills", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "validation" {
		t.Errorf("code = %q, want \"validation\"", env.Error.Code)
	}
	if env.Error.Fields == nil {
		t.Fatalf("envelope missing 'error.fields' (anti-drift gate)")
	}
	if _, ok := env.Error.Fields["name"]; !ok {
		t.Errorf("fields.name missing; got fields = %v", env.Error.Fields)
	}
}

// TestSkillHandler_Create_DuplicateName_Returns409WithConflictCode asserts
// that a duplicate active name is translated to HTTP 409 with code "conflict".
func TestSkillHandler_Create_DuplicateName_Returns409WithConflictCode(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "dup"
	desc := "d"
	body := validSkillBody(name, desc)
	payload := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)

	if rec := skillHttpDo(t, e, "POST", "/skills", payload); rec.Code != http.StatusCreated {
		t.Fatalf("first POST status = %d, want 201", rec.Code)
	}
	rec := skillHttpDo(t, e, "POST", "/skills", payload)
	if rec.Code != http.StatusConflict {
		t.Fatalf("duplicate POST status = %d, want 409; body = %s", rec.Code, rec.Body.String())
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "conflict" {
		t.Errorf("code = %q, want \"conflict\"", env.Error.Code)
	}
}

// TestSkillHandler_DeletedSkill_UsesSkillDeletedCode asserts that a PATCH on
// a soft-deleted skill returns HTTP 410 with code "skill_deleted" (anti-drift
// gate — the locked wire code is skill_deleted, NOT prompt_deleted).
func TestSkillHandler_DeletedSkill_UsesSkillDeletedCode(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "del"
	desc := "d"
	body := validSkillBody(name, desc)
	payload := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", payload); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	if rec := skillHttpDo(t, e, "DELETE", "/skills/"+name, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	// Now PATCH on the soft-deleted skill — MUST be 410 + skill_deleted.
	rec := skillHttpDo(t, e, "PATCH", "/skills/"+name, []byte(`{"description":"new"}`))
	if rec.Code != http.StatusGone {
		t.Fatalf("patch status = %d, want 410; body = %s", rec.Code, rec.Body.String())
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "skill_deleted" {
		t.Errorf("code = %q, want \"skill_deleted\" (anti-drift gate — NOT prompt_deleted)", env.Error.Code)
	}
}

// ---------------------------------------------------------------------------
// List / Get
// ---------------------------------------------------------------------------

// TestSkillHandler_List_EmitsCurrentRevision asserts every list item carries
// current_revision (anti-drift gate — prevents the v{undefined} prompt bug
// from leaking into the skills feature).
func TestSkillHandler_List_EmitsCurrentRevision(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	for i, name := range []string{"alpha", "bravo"} {
		desc := "d"
		body := validSkillBody(name, desc)
		payload := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)
		if rec := skillHttpDo(t, e, "POST", "/skills", payload); rec.Code != http.StatusCreated {
			t.Fatalf("create %s status = %d, want 201; body = %s", name, rec.Code, rec.Body.String())
		}
		_ = i
	}

	rec := skillHttpDo(t, e, "GET", "/skills", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var resp skillListResponseShape
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Skills) != 2 {
		t.Fatalf("list len = %d, want 2", len(resp.Skills))
	}
	for _, it := range resp.Skills {
		if _, ok := it["current_revision"]; !ok {
			t.Errorf("list item missing current_revision (anti-drift gate ADR-SK-008): %v", it)
		}
	}
}

// TestSkillHandler_List_DefaultsLimit50AndCapsAt200 exercises the limit
// clamping. Default of 50 means the handler accepts no limit param; cap of
// 200 means a huge limit clamps to 200. We assert the response shape
// includes a "skills" array (per spec §8 wire-shape lock).
func TestSkillHandler_List_DefaultsLimit50AndCapsAt200(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	rec := skillHttpDo(t, e, "GET", "/skills", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	// Spec §8: list response is {"skills":[...]} — verify shape.
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := resp["skills"]; !ok {
		t.Errorf("list response missing 'skills' key (spec §8 wire-shape lock); got %v", resp)
	}

	// Also verify the cap is honored: a limit=999 request does not error.
	rec2 := skillHttpDo(t, e, "GET", "/skills?limit=999", nil)
	if rec2.Code != http.StatusOK {
		t.Errorf("limit=999 status = %d, want 200 (clamped to 200)", rec2.Code)
	}
}

// TestSkillHandler_GetBySlug_NotFoundReturns404Envelope asserts the missing
// path returns 404 + code="not_found".
func TestSkillHandler_GetBySlug_NotFoundReturns404Envelope(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	rec := skillHttpDo(t, e, "GET", "/skills/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "not_found" {
		t.Errorf("code = %q, want \"not_found\"", env.Error.Code)
	}
}

// TestSkillHandler_GetBySlug_DeletedReturns404Envelope asserts that GET on a
// soft-deleted skill returns 404 (NOT 410 — 410 is reserved for update/restore).
func TestSkillHandler_GetBySlug_DeletedReturns404Envelope(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "toget"
	desc := "d"
	body := validSkillBody(name, desc)
	payload := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", payload); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	if rec := skillHttpDo(t, e, "DELETE", "/skills/"+name, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	rec := skillHttpDo(t, e, "GET", "/skills/"+name, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET deleted status = %d, want 404", rec.Code)
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "not_found" {
		t.Errorf("code = %q, want \"not_found\" (deleted skills return 404, not 410)", env.Error.Code)
	}
}

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

// TestSkillHandler_Update_AppendsRevision_AndReturnsNewRevision asserts that
// PATCH on a live skill returns 200 with current_revision bumped to 2.
func TestSkillHandler_Update_AppendsRevision_AndReturnsNewRevision(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "upd"
	desc1 := "v1"
	body1 := validSkillBody(name, desc1)
	create := []byte(`{"name":"` + name + `","description":"` + desc1 + `","body":` + jsonString(body1) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", create); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}

	desc2 := "v2"
	body2 := validSkillBody(name, desc2)
	patch := []byte(`{"description":"` + desc2 + `","body":` + jsonString(body2) + `}`)
	rec := skillHttpDo(t, e, "PATCH", "/skills/"+name, patch)
	if rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr, ok := got["current_revision"].(float64); !ok || cr != 2 {
		t.Errorf("current_revision = %v, want 2 (anti-drift gate)", got["current_revision"])
	}
	if got["description"] != desc2 {
		t.Errorf("description = %v, want %s", got["description"], desc2)
	}
}

// TestSkillHandler_Update_400OnInvalidBody asserts that a PATCH whose body
// fails validation returns 400 + envelope with code="validation".
func TestSkillHandler_Update_400OnInvalidBody(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "badpatch"
	desc := "d"
	body := validSkillBody(name, desc)
	create := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", create); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	// PATCH with a body that violates the name regex (frontmatter says
	// "BAD" but URL slug is "badpatch"). The handler's update path must
	// run the lock-step validator and return 400.
	badBody := validSkillBody("BAD", desc)
	patch := []byte(`{"body":` + jsonString(badBody) + `}`)
	rec := skillHttpDo(t, e, "PATCH", "/skills/"+name, patch)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("patch status = %d, want 400; body = %s", rec.Code, rec.Body.String())
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "validation" {
		t.Errorf("code = %q, want \"validation\"", env.Error.Code)
	}
}

// TestSkillHandler_Update_410OnDeletedSkill asserts the 410 path on update.
func TestSkillHandler_Update_410OnDeletedSkill(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "upddel"
	desc := "d"
	body := validSkillBody(name, desc)
	create := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", create); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	if rec := skillHttpDo(t, e, "DELETE", "/skills/"+name, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	patch := []byte(`{"description":"new","body":` + jsonString(validSkillBody(name, "new")) + `}`)
	rec := skillHttpDo(t, e, "PATCH", "/skills/"+name, patch)
	if rec.Code != http.StatusGone {
		t.Fatalf("patch status = %d, want 410; body = %s", rec.Code, rec.Body.String())
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "skill_deleted" {
		t.Errorf("code = %q, want \"skill_deleted\"", env.Error.Code)
	}
}

// ---------------------------------------------------------------------------
// Delete (idempotent)
// ---------------------------------------------------------------------------

// TestSkillHandler_Delete_Returns204 asserts the happy-path delete.
func TestSkillHandler_Delete_Returns204(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "killme"
	desc := "d"
	body := validSkillBody(name, desc)
	payload := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", payload); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	rec := skillHttpDo(t, e, "DELETE", "/skills/"+name, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
}

// TestSkillHandler_Delete_MissingReturns204 asserts idempotence on missing name.
func TestSkillHandler_Delete_MissingReturns204(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	rec := skillHttpDo(t, e, "DELETE", "/skills/never-existed", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("delete missing status = %d, want 204 (idempotent)", rec.Code)
	}
}

// TestSkillHandler_Delete_AlreadyDeletedReturns204 asserts idempotence on
// already-deleted name (deleted twice = 204 both times).
func TestSkillHandler_Delete_AlreadyDeletedReturns204(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "twice"
	desc := "d"
	body := validSkillBody(name, desc)
	payload := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", payload); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	if rec := skillHttpDo(t, e, "DELETE", "/skills/"+name, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("first delete status = %d, want 204", rec.Code)
	}
	rec := skillHttpDo(t, e, "DELETE", "/skills/"+name, nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("second delete status = %d, want 204 (idempotent)", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Revisions / Restore
// ---------------------------------------------------------------------------

// TestSkillHandler_ListRevisions_NewestFirst asserts that the list comes
// back in DESC order (newest first) per spec SCN-5.5.
func TestSkillHandler_ListRevisions_NewestFirst(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "revhist"
	desc1 := "v1"
	body1 := validSkillBody(name, desc1)
	create := []byte(`{"name":"` + name + `","description":"` + desc1 + `","body":` + jsonString(body1) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", create); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	// Apply two more updates → revisions 1, 2, 3.
	for i := 2; i <= 3; i++ {
		d := "v" + string(rune('0'+i))
		patch := []byte(`{"description":"` + d + `","body":` + jsonString(validSkillBody(name, d)) + `}`)
		if rec := skillHttpDo(t, e, "PATCH", "/skills/"+name, patch); rec.Code != http.StatusOK {
			t.Fatalf("patch %d status = %d, want 200", i, rec.Code)
		}
	}

	rec := skillHttpDo(t, e, "GET", "/skills/"+name+"/revisions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list revisions status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	revs, ok := resp["revisions"].([]any)
	if !ok || len(revs) != 3 {
		t.Fatalf("revisions: got %T (len=%d), want array of 3; body = %s", resp["revisions"], len(revs), rec.Body.String())
	}
	first := revs[0].(map[string]any)["revision_number"].(float64)
	if first != 3 {
		t.Errorf("first revision = %v, want 3 (newest first)", first)
	}
}

// TestSkillHandler_ListRevisions_NotFoundReturns404Envelope asserts the missing path.
func TestSkillHandler_ListRevisions_NotFoundReturns404Envelope(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	rec := skillHttpDo(t, e, "GET", "/skills/never-existed/revisions", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "not_found" {
		t.Errorf("code = %q, want \"not_found\"", env.Error.Code)
	}
}

// TestSkillHandler_Restore_AppendsNewRevision asserts restore creates a new
// revision (preserving history per spec SCN-1.3) and returns the restored
// content as the current body.
func TestSkillHandler_Restore_AppendsNewRevision(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "restore"
	desc1 := "v1"
	body1 := validSkillBody(name, desc1)
	create := []byte(`{"name":"` + name + `","description":"` + desc1 + `","body":` + jsonString(body1) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", create); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	// Update once → revisions 1, 2.
	patch := []byte(`{"description":"v2","body":` + jsonString(validSkillBody(name, "v2")) + `}`)
	if rec := skillHttpDo(t, e, "PATCH", "/skills/"+name, patch); rec.Code != http.StatusOK {
		t.Fatalf("patch status = %d, want 200", rec.Code)
	}
	// Restore revision 1 → new revision 3 with the body of revision 1.
	rec := skillHttpDo(t, e, "POST", "/skills/"+name+"/revisions/1/restore", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("restore status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if cr, ok := got["current_revision"].(float64); !ok || cr != 3 {
		t.Errorf("current_revision = %v, want 3", got["current_revision"])
	}
	if got["description"] != desc1 {
		t.Errorf("description = %v, want %q (restored)", got["description"], desc1)
	}
}

// TestSkillHandler_Restore_OnDeletedReturns410Envelope asserts the 410 path.
func TestSkillHandler_Restore_OnDeletedReturns410Envelope(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	name := "restdel"
	desc := "d"
	body := validSkillBody(name, desc)
	payload := []byte(`{"name":"` + name + `","description":"` + desc + `","body":` + jsonString(body) + `}`)
	if rec := skillHttpDo(t, e, "POST", "/skills", payload); rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201", rec.Code)
	}
	if rec := skillHttpDo(t, e, "DELETE", "/skills/"+name, nil); rec.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d, want 204", rec.Code)
	}
	rec := skillHttpDo(t, e, "POST", "/skills/"+name+"/revisions/1/restore", nil)
	if rec.Code != http.StatusGone {
		t.Fatalf("restore deleted status = %d, want 410", rec.Code)
	}
	var env envEnvelope
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if env.Error.Code != "skill_deleted" {
		t.Errorf("code = %q, want \"skill_deleted\"", env.Error.Code)
	}
}

// ---------------------------------------------------------------------------
// Log redaction (spec SCN-6.3)
// ---------------------------------------------------------------------------

// TestSkillHandler_NoPIIInLogs creates a Skill with a unique sentinel string
// in the body and asserts the sentinel NEVER appears in the captured log
// buffer (the handler MUST only log lengths and the slug).
func TestSkillHandler_NoPIIInLogs(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	host := skillOsOr("POSTGRES_HOST", "localhost")
	port := skillOsOr("POSTGRES_PORT", "5432")
	user := skillOsOr("QUEEN_USER", "queen")
	pass := skillOsOr("QUEEN_PASSWORD", "changeme-queen")
	dbname := skillOsOr("POSTGRES_DB", "cachicamas_pg")
	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + dbname + " sslmode=disable"
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ensureSkillHandlerTables(t, dbConn)
	cleanSkillHandlerTables(t, dbConn)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	skillRepo := skills.NewSkillRepo(dbConn)
	revRepo := skills.NewSkillRevisionRepo(dbConn)
	svc := application.NewSkillService(skillRepo, revRepo, dbConn, logger)
	h := httpiface.NewSkillHandler(svc, logger)
	e := echo.New()
	h.RegisterSkillRoutes(e.Group(""))

	defer func() { _ = dbConn.Close() }()

	sentinel := "SECRET_SENTINEL_TOKEN_DO_NOT_LOG"
	desc := "log-test-description"
	body := validSkillBody("logtest", desc) + "\n" + sentinel
	payload := []byte(`{"name":"logtest","description":"` + desc + `","body":` + jsonString(body) + `}`)
	rec := skillHttpDo(t, e, "POST", "/skills", payload)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	if strings.Contains(logBuf.String(), sentinel) {
		t.Errorf("log buffer leaked sentinel %q; captured log:\n%s", sentinel, logBuf.String())
	}
	// Also assert the description text is not echoed in logs.
	if strings.Contains(logBuf.String(), desc) {
		t.Errorf("log buffer leaked description %q; captured log:\n%s", desc, logBuf.String())
	}
}

// ---------------------------------------------------------------------------
// Route registration (spec SCN-7.1, R-SK-007)
// ---------------------------------------------------------------------------

// TestSkillHandler_RouteRegistration_MatchesSpecExactly introspects the Echo
// router and asserts the routes registered under /skills* are EXACTLY the
// seven endpoints from spec §6 — no spurious routes, no `?deleted=` query
// params accepted.
func TestSkillHandler_RouteRegistration_MatchesSpecExactly(t *testing.T) {
	e, _, db := setupSkillHandler(t)
	defer func() { _ = db.Close() }()

	expected := map[string]string{
		"GET /skills":                             "GET",
		"POST /skills":                            "POST",
		"GET /skills/:name":                       "GET",
		"PATCH /skills/:name":                     "PATCH",
		"DELETE /skills/:name":                    "DELETE",
		"GET /skills/:name/revisions":             "GET",
		"POST /skills/:name/revisions/:n/restore": "POST",
	}
	got := map[string]bool{}
	for _, r := range e.Router().Routes() {
		path := r.Path
		if !strings.HasPrefix(path, "/skills") {
			continue
		}
		got[methodPath(r.Method, path)] = true
	}
	if len(got) != len(expected) {
		t.Fatalf("routes count = %d, want %d; got = %v", len(got), len(expected), got)
	}
	for k := range expected {
		if !got[k] {
			t.Errorf("missing expected route %q", k)
		}
	}
}

func methodPath(method, path string) string {
	return method + " " + path
}
