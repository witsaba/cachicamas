// Package httpiface_test — integration tests for the 7 prompt HTTP
// endpoints. Tests are gated on INTEGRATION=1 because they need a
// live Postgres. Mirrors the pattern in workspace_handler_test.go
// (which uses real services + integration DB).
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
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres/prompts"
	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

// ---------------------------------------------------------------------------
// Setup helpers
// ---------------------------------------------------------------------------

func setupHandler(t *testing.T) (*echo.Echo, *httpiface.PromptHandler, *sql.DB) {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	host := osOr("POSTGRES_HOST", "localhost")
	port := osOr("POSTGRES_PORT", "5432")
	user := osOr("QUEEN_USER", "queen")
	pass := osOr("QUEEN_PASSWORD", "changeme-queen")
	dbname := osOr("POSTGRES_DB", "cachicamas_pg")
	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + dbname + " sslmode=disable"
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := dbConn.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	ensureHandlerTables(t, dbConn)
	cleanHandlerTables(t, dbConn)

	promptRepo := prompts.NewPromptRepo(dbConn)
	revRepo := prompts.NewPromptRevisionRepo(dbConn)
	svc := application.NewPromptService(promptRepo, revRepo, dbConn, nil)

	// Capture logs in a buffer to assert no PII leakage (S-PR-X3).
	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	h := httpiface.NewPromptHandler(svc, logger)
	e := echo.New()
	h.RegisterPromptRoutes(e.Group(""))
	return e, h, dbConn
}

func osOr(k, f string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return f
}

func ensureHandlerTables(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS prompt (
		    id           BIGSERIAL    PRIMARY KEY,
		    description  TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 280),
		    slug         TEXT         NOT NULL CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,98}[a-z0-9]$'),
		    body         TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
		    deleted_at   TIMESTAMPTZ  NULL,
		    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
		)`,
		`CREATE TABLE IF NOT EXISTS prompt_revision (
		    id              BIGSERIAL    PRIMARY KEY,
		    prompt_id       BIGINT       NOT NULL REFERENCES prompt(id) ON DELETE CASCADE,
		    revision_number INT          NOT NULL CHECK (revision_number > 0),
		    description     TEXT         NOT NULL CHECK (length(description) BETWEEN 1 AND 280),
		    body            TEXT         NOT NULL CHECK (length(body) BETWEEN 1 AND 524288),
		    change_note     TEXT         NULL,
		    created_by      TEXT         NULL,
		    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
		    CONSTRAINT prompt_revision_unique UNIQUE (prompt_id, revision_number)
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS prompt_slug_active_uidx ON prompt(slug) WHERE deleted_at IS NULL`,
		`CREATE INDEX IF NOT EXISTS prompt_updated_at_idx ON prompt(updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS prompt_revision_prompt_id_idx ON prompt_revision(prompt_id, revision_number DESC)`,
	}
	for _, s := range stmts {
		if _, err := db.ExecContext(ctx, s); err != nil {
			t.Fatalf("ensureHandlerTables: %v", err)
		}
	}
}

func cleanHandlerTables(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := db.ExecContext(ctx, "DELETE FROM prompt_revision"); err != nil {
		t.Fatalf("clean prompt_revision: %v", err)
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM prompt"); err != nil {
		t.Fatalf("clean prompt: %v", err)
	}
}

// httpDo executes an HTTP request against the Echo app and returns
// the response recorder.
func httpDo(t *testing.T, e *echo.Echo, method, path string, body []byte) *httptest.ResponseRecorder {
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

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestPromptHandler_Create_HappyPath_201(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	body := []byte(`{"slug":"welcome-email","description":"Welcome email body","body":"# Welcome"}`)
	rec := httpDo(t, e, "POST", "/prompts", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["slug"] != "welcome-email" {
		t.Errorf("slug = %v, want welcome-email", got["slug"])
	}
}

func TestPromptHandler_Create_DuplicateSlug_409(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	body := []byte(`{"slug":"dup","description":"d","body":"b"}`)
	_ = httpDo(t, e, "POST", "/prompts", body)
	rec := httpDo(t, e, "POST", "/prompts", body)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rec.Code)
	}
}

func TestPromptHandler_Create_InvalidSlug_400(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	body := []byte(`{"slug":"BAD","description":"d","body":"b"}`)
	rec := httpDo(t, e, "POST", "/prompts", body)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Get / List
// ---------------------------------------------------------------------------

func TestPromptHandler_GetBySlug_200(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	_ = httpDo(t, e, "POST", "/prompts", []byte(`{"slug":"get","description":"d","body":"b"}`))
	rec := httpDo(t, e, "GET", "/prompts/get", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}

func TestPromptHandler_GetBySlug_404(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	rec := httpDo(t, e, "GET", "/prompts/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPromptHandler_List_ExcludesDeleted(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	_ = httpDo(t, e, "POST", "/prompts", []byte(`{"slug":"a","description":"d","body":"b"}`))
	_ = httpDo(t, e, "POST", "/prompts", []byte(`{"slug":"b","description":"d","body":"b"}`))
	_ = httpDo(t, e, "DELETE", "/prompts/b", nil)

	rec := httpDo(t, e, "GET", "/prompts", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	for _, p := range got {
		if p["slug"] == "b" {
			t.Errorf("List returned deleted slug b")
		}
	}
}

// ---------------------------------------------------------------------------
// Update / SoftDelete
// ---------------------------------------------------------------------------

func TestPromptHandler_Update_HappyPath_200(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	_ = httpDo(t, e, "POST", "/prompts", []byte(`{"slug":"upd","description":"d1","body":"b1"}`))
	body := []byte(`{"body":"new body"}`)
	rec := httpDo(t, e, "PATCH", "/prompts/upd", body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
}

func TestPromptHandler_Update_Deleted_410(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	_ = httpDo(t, e, "POST", "/prompts", []byte(`{"slug":"del","description":"d","body":"b"}`))
	_ = httpDo(t, e, "DELETE", "/prompts/del", nil)
	rec := httpDo(t, e, "PATCH", "/prompts/del", []byte(`{"body":"x"}`))
	if rec.Code != http.StatusGone {
		t.Fatalf("status = %d, want 410", rec.Code)
	}
}

func TestPromptHandler_Delete_204(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	_ = httpDo(t, e, "POST", "/prompts", []byte(`{"slug":"d","description":"d","body":"b"}`))
	rec := httpDo(t, e, "DELETE", "/prompts/d", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Revisions
// ---------------------------------------------------------------------------

func TestPromptHandler_ListRevisions_NewestFirst(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	_ = httpDo(t, e, "POST", "/prompts", []byte(`{"slug":"rev","description":"d","body":"b1"}`))
	_ = httpDo(t, e, "PATCH", "/prompts/rev", []byte(`{"body":"b2"}`))
	_ = httpDo(t, e, "PATCH", "/prompts/rev", []byte(`{"body":"b3"}`))
	rec := httpDo(t, e, "GET", "/prompts/rev/revisions", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var got []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
	if got[0]["revision_number"].(float64) != 3 {
		t.Errorf("got[0].revision_number = %v, want 3", got[0]["revision_number"])
	}
}

func TestPromptHandler_Restore_HappyPath_200(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	_ = httpDo(t, e, "POST", "/prompts", []byte(`{"slug":"restore","description":"d","body":"v1"}`))
	_ = httpDo(t, e, "PATCH", "/prompts/restore", []byte(`{"body":"v2"}`))
	rec := httpDo(t, e, "POST", "/prompts/restore/revisions/1/restore", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}
	var got map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if got["body"] != "v1" {
		t.Errorf("body = %v, want v1", got["body"])
	}
}

// ---------------------------------------------------------------------------
// Error envelope (S-PR-X4)
// ---------------------------------------------------------------------------

func TestPromptHandler_ErrorEnvelopeShape(t *testing.T) {
	e, _, db := setupHandler(t)
	defer func() { _ = db.Close() }()
	rec := httpDo(t, e, "GET", "/prompts/missing", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	var env map[string]map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	errObj, ok := env["error"]
	if !ok {
		t.Fatalf("envelope missing 'error' key: %v", env)
	}
	if _, ok := errObj["code"]; !ok {
		t.Errorf("envelope missing 'error.code'")
	}
	if _, ok := errObj["message"]; !ok {
		t.Errorf("envelope missing 'error.message'")
	}
}

// ---------------------------------------------------------------------------
// Log redaction (S-PR-X3)
// ---------------------------------------------------------------------------

// TestPromptHandler_NoPIIInLogs re-creates the handler with a capturing
// logger, sends a POST whose body contains a unique sentinel string,
// then asserts the sentinel is absent from the captured log lines.
func TestPromptHandler_NoPIIInLogs(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	host := osOr("POSTGRES_HOST", "localhost")
	port := osOr("POSTGRES_PORT", "5432")
	user := osOr("QUEEN_USER", "queen")
	pass := osOr("QUEEN_PASSWORD", "changeme-queen")
	dbname := osOr("POSTGRES_DB", "cachicamas_pg")
	dsn := "host=" + host + " port=" + port + " user=" + user + " password=" + pass + " dbname=" + dbname + " sslmode=disable"
	dbConn, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	ensureHandlerTables(t, dbConn)
	cleanHandlerTables(t, dbConn)

	var logBuf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	svc := application.NewPromptService(prompts.NewPromptRepo(dbConn), prompts.NewPromptRevisionRepo(dbConn), dbConn, logger)
	h := httpiface.NewPromptHandler(svc, logger)
	e := echo.New()
	h.RegisterPromptRoutes(e.Group(""))

	defer func() { _ = dbConn.Close() }()
	sentinel := "SECRET_SENTINEL_TOKEN_DO_NOT_LOG"
	body := []byte(`{"slug":"log-test","description":"d","body":"` + sentinel + `"}`)
	rec := httpDo(t, e, "POST", "/prompts", body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body = %s", rec.Code, rec.Body.String())
	}

	if strings.Contains(logBuf.String(), sentinel) {
		t.Errorf("log buffer leaked sentinel %q; captured log:\n%s", sentinel, logBuf.String())
	}
}