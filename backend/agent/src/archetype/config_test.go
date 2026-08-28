// Package chat_test — assistant_config_test.go is the TDD contract for
// the chat-owned AssistantConfig loader (REQ-CACS-001/002/003,
// design AD-1/AD-2). The 4 RED scenarios + 2 pure-function assertions
// are transcribed verbatim from design §8 (PR-1) and the locked
// REQ-CACS-* invariants from spec #4077.
//
// All four Loader scenarios are gated by INTEGRATION=1 — the Loader is
// a *sql.DB-backed Postgres adapter; asserting FOR SHARE semantics
// against a real Postgres is the only honest verification. Mirrors
// store_postgres_integration_test.go's gating pattern.
//
// Pure-function assertions (Test_Defaults_Pure,
// Test_Validation_SentinelsExist) run without INTEGRATION — they
// exercise the chat-package in-memory path the Loader composes with.
//
// RED state at T-01: this file references archetype.ArchetypeConfig,
// archetype.ArchetypeConfigLoader, archetype.NewPostgresLoader,
// archetype.DefaultConfig, chat.Err* sentinels — none of which
// exist in the package yet. `go test ./src/chat/...` fails to
// compile the test binary. That is the documented RED signal. GREEN
// (T-02) adds the production code; the same `go test` command then
// compiles and runs the scenarios.
package archetype_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"sync"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/agent/src/archetype"
)

// assistantConfigRequiresPostgres skips the test unless INTEGRATION=1
// is set. Mirrors the DBA precedent (DBA's driver_test.go:295) and
// chat's own store_postgres_integration_test.go. The skip message names
// the env var an operator must set to run the test.
func assistantConfigRequiresPostgres(t *testing.T) string {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the assistant config loader is a Postgres adapter; FOR SHARE semantics require a live connection)")
	}
	dsn := os.Getenv("CACHICAMAS_CHAT_STORE_DSN")
	if dsn == "" {
		dsn = "postgres://cachicamas:changeme-local-only@localhost:5432/cachicamas_pg?sslmode=disable"
	}
	return dsn
}

// resetAssistantConfigTable truncates archetype_configurations and
// then re-inserts the seeded `__default__` row (migration 0003) so the
// suite starts from a clean-but-seeded state. Tests that exercise
// the "no rows at all" path (in-memory fallback) can call
// truncateAssistantConfigTable directly instead of going through
// this helper.
func resetAssistantConfigTable(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("resetAssistantConfigTable: sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`TRUNCATE TABLE archetype_configurations`); err != nil {
		t.Fatalf("resetAssistantConfigTable: TRUNCATE: %v", err)
	}
	// Re-seed the system default row (mirrors migration 0003).
	if _, err := db.Exec(`
		INSERT INTO archetype_configurations
			(archetype_slug, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by)
		VALUES
			('chat', '__default__',
			 'You are the cachicamas chat assistant; answer the participant in plain, well-formatted text.',
			 '["current_time", "summarize_conversation"]'::jsonb,
			 '["summarize_conversation"]'::jsonb,
			 NULL, 1, now(), 'seed')
	`); err != nil {
		t.Fatalf("resetAssistantConfigTable: re-seed default: %v", err)
	}
}

// truncateAssistantConfigTable removes ALL rows including the
// `__default__` seed. Use this for tests that explicitly exercise
// the "no rows at all → in-memory fallback" path (REQ-CACS-003).
func truncateAssistantConfigTable(t *testing.T, dsn string) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("truncateAssistantConfigTable: sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(`TRUNCATE TABLE archetype_configurations`); err != nil {
		t.Fatalf("truncateAssistantConfigTable: TRUNCATE: %v", err)
	}
}

// seedRow inserts one archetype_configurations row for the supplied cfg.
// Used by Test_Loader_PresentRow and the FOR SHARE writer goroutine.
func seedRow(t *testing.T, dsn string, cfg archetype.ArchetypeConfig) {
	t.Helper()
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("seedRow: sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()
	allowlist, _ := json.Marshal(cfg.ToolAllowlist)
	defers, _ := json.Marshal(cfg.DeferToolNames)
	var model any
	if cfg.Model != nil {
		model = *cfg.Model
	}
	if _, err := db.Exec(`
		INSERT INTO archetype_configurations
			(archetype_slug, org_id, system_prompt, tool_allowlist, defer_tool_names, model, version, updated_at, updated_by)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6, $7, now(), $8)
	`, string(cfg.Slug), cfg.OrgID, cfg.SystemPrompt, allowlist, defers, model, cfg.Version, cfg.UpdatedBy); err != nil {
		t.Fatalf("seedRow: INSERT: %v", err)
	}
}

// Test_Loader_PresentRow — REQ-CACS-002 / Scenario "present row returns
// config and found=true". Given a row exists for orgID="org-1", when
// the loader is called, then it returns the row's fields, found=true,
// err=nil.
//
// RED at T-01: archetype.NewPostgresLoader is undefined;
// the test file fails to compile. GREEN (T-02) adds the loader.
func Test_Loader_PresentRow(t *testing.T) {
	dsn := assistantConfigRequiresPostgres(t)
	resetAssistantConfigTable(t, dsn)

	seed := archetype.ArchetypeConfig{
		Slug:           archetype.AssistantSlug,
		OrgID:          "org-1",
		SystemPrompt:   "you are org-1's assistant",
		ToolAllowlist:  []string{"current_time", "summarize_conversation"},
		DeferToolNames: []string{"summarize_conversation"},
		Version:        3,
		UpdatedBy:      "user_alice",
		UpdatedAt:      time.Now().UTC(),
	}
	seedRow(t, dsn, seed)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	loader := archetype.NewPostgresLoader(db)
	got, found, lerr := loader.LoadBySlug(context.Background(), archetype.AssistantSlug, "org-1")
	if lerr != nil {
		t.Fatalf("LoadByOrg returned err=%v, want nil", lerr)
	}
	if !found {
		t.Fatalf("LoadByOrg returned found=false, want true")
	}
	if got.OrgID != seed.OrgID {
		t.Errorf("OrgID = %q, want %q", got.OrgID, seed.OrgID)
	}
	if got.SystemPrompt != seed.SystemPrompt {
		t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, seed.SystemPrompt)
	}
	if !reflect.DeepEqual(got.ToolAllowlist, seed.ToolAllowlist) {
		t.Errorf("ToolAllowlist = %v, want %v", got.ToolAllowlist, seed.ToolAllowlist)
	}
	if !reflect.DeepEqual(got.DeferToolNames, seed.DeferToolNames) {
		t.Errorf("DeferToolNames = %v, want %v", got.DeferToolNames, seed.DeferToolNames)
	}
	if got.Version != seed.Version {
		t.Errorf("Version = %d, want %d", got.Version, seed.Version)
	}
	if got.UpdatedBy != seed.UpdatedBy {
		t.Errorf("UpdatedBy = %q, want %q", got.UpdatedBy, seed.UpdatedBy)
	}
}

// Test_Loader_AbsentRowReturnsDefaults — REQ-CACS-002 / REQ-CACS-003
// Scenario "absent row returns defaults and found=false". Given no
// row for orgID="org-2" AND no `__default__` seed row (i.e. DB up
// but the seed was manually removed — the LAST-RESORT fallback path),
// when LoadBySlug runs, then it returns the in-memory safe
// default, found=false, IsOverride=false, AND no row is inserted.
//
// RED at T-01: archetype.DefaultConfig is undefined. GREEN (T-02)
// adds it.
func Test_Loader_AbsentRowReturnsDefaults(t *testing.T) {
	dsn := assistantConfigRequiresPostgres(t)
	// truncate (without re-seed) — exercises the "no rows at all"
	// path that falls back to in-memory defaults (REQ-CACS-003 last
	// resort).
	truncateAssistantConfigTable(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	loader := archetype.NewPostgresLoader(db)
	got, found, lerr := loader.LoadBySlug(context.Background(), archetype.AssistantSlug, "org-2")
	if lerr != nil {
		t.Fatalf("LoadByOrg returned err=%v, want nil", lerr)
	}
	if found {
		t.Fatalf("LoadByOrg returned found=true, want false")
	}
	if got.IsOverride {
		t.Errorf("IsOverride = true, want false (in-memory fallback)")
	}
	wantDefaults := archetype.DefaultConfig(archetype.AssistantSlug, "org-2", []string{"current_time", "summarize_conversation"})
	if got.OrgID != wantDefaults.OrgID {
		t.Errorf("OrgID = %q, want %q", got.OrgID, wantDefaults.OrgID)
	}
	if got.SystemPrompt != wantDefaults.SystemPrompt {
		t.Errorf("SystemPrompt = %q, want %q", got.SystemPrompt, wantDefaults.SystemPrompt)
	}
	if !reflect.DeepEqual(got.ToolAllowlist, wantDefaults.ToolAllowlist) {
		t.Errorf("ToolAllowlist = %v, want %v", got.ToolAllowlist, wantDefaults.ToolAllowlist)
	}
	if got.Version != wantDefaults.Version {
		t.Errorf("Version = %d, want %d (defaults are version=1 per design AD-2)", got.Version, wantDefaults.Version)
	}
	if got.UpdatedBy != "" {
		t.Errorf("UpdatedBy = %q, want empty (defaults carry no actor)", got.UpdatedBy)
	}
	// REQ-CACS-003: no row was created on the read path. Assert via
	// direct SELECT count(*).
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM archetype_configurations WHERE archetype_slug = $1 AND org_id = $2`, archetype.AssistantSlug, "org-2").Scan(&count); err != nil {
		t.Fatalf("assert row absent: %v", err)
	}
	if count != 0 {
		t.Errorf("archetype_configurations row count for org-2 = %d, want 0 (REQ-CACS-003: loader MUST NOT auto-write)", count)
	}
}

// Test_Loader_DefaultRow_FallbackWhenNoPerOrgRow — migration 0003
// seeds a per-archetype system default row at org_id='__default__'.
// When no per-org row exists, the Loader must return the default
// row's content (with OrgID rewritten to the caller's orgID) and
// IsOverride=false.
func Test_Loader_DefaultRow_FallbackWhenNoPerOrgRow(t *testing.T) {
	t.Parallel()

	dsn := assistantConfigRequiresPostgres(t)
	resetAssistantConfigTable(t, dsn) // re-seeds the default row

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	loader := archetype.NewPostgresLoader(db)
	got, found, lerr := loader.LoadBySlug(context.Background(), archetype.AssistantSlug, "user_alice")
	if lerr != nil {
		t.Fatalf("LoadBySlug returned err=%v, want nil", lerr)
	}
	if !found {
		t.Fatalf("LoadBySlug returned found=false, want true (default row exists)")
	}
	if got.IsOverride {
		t.Errorf("IsOverride = true, want false (default row, not per-org)")
	}
	if got.OrgID != "user_alice" {
		t.Errorf("OrgID = %q, want %q (Loader must rewrite default row's sentinel to caller's orgID)", got.OrgID, "user_alice")
	}
	if got.SystemPrompt != archetype.DefaultChatSystemPrompt {
		t.Errorf("SystemPrompt = %q, want %q (default row should match the seeded prompt)", got.SystemPrompt, archetype.DefaultChatSystemPrompt)
	}
	if got.Version != 1 {
		t.Errorf("Version = %d, want 1 (default row seeded at version=1)", got.Version)
	}
	if got.UpdatedBy != "seed" {
		t.Errorf("UpdatedBy = %q, want %q (default row was inserted by the seed)", got.UpdatedBy, "seed")
	}
}

// Test_Loader_PerOrgRow_ShadowsDefaultRow — when both the per-org
// row and the `__default__` row exist, the per-org row wins.
func Test_Loader_PerOrgRow_ShadowsDefaultRow(t *testing.T) {
	t.Parallel()

	dsn := assistantConfigRequiresPostgres(t)
	resetAssistantConfigTable(t, dsn) // default row exists

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	// Seed a per-org override with different values.
	seedRow(t, dsn, archetype.ArchetypeConfig{
		Slug:           archetype.AssistantSlug,
		OrgID:          "user_alice",
		SystemPrompt:   "you are org-1's assistant (overridden)",
		ToolAllowlist:  []string{"current_time", "summarize_conversation"},
		DeferToolNames: []string{"summarize_conversation"},
		Version:        5,
		UpdatedBy:      "user_alice",
	})

	loader := archetype.NewPostgresLoader(db)
	got, found, lerr := loader.LoadBySlug(context.Background(), archetype.AssistantSlug, "user_alice")
	if lerr != nil {
		t.Fatalf("LoadBySlug returned err=%v, want nil", lerr)
	}
	if !found {
		t.Fatalf("LoadBySlug returned found=false, want true")
	}
	if !got.IsOverride {
		t.Errorf("IsOverride = false, want true (per-org row should shadow the default)")
	}
	if got.SystemPrompt != "you are org-1's assistant (overridden)" {
		t.Errorf("SystemPrompt = %q, want the per-org override, not the default", got.SystemPrompt)
	}
	if got.Version != 5 {
		t.Errorf("Version = %d, want 5 (per-org row's version)", got.Version)
	}
}

// Test_Loader_DoubleAbsentReadNoSideEffect — REQ-CACS-003 Scenario
// "read on absent row does not create a row". Two consecutive
// LoadBySlug calls on an empty table both return found=false
// and the table remains empty.
//
// RED at T-01.
func Test_Loader_DoubleAbsentReadNoSideEffect(t *testing.T) {
	dsn := assistantConfigRequiresPostgres(t)
	truncateAssistantConfigTable(t, dsn)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	loader := archetype.NewPostgresLoader(db)

	_, found1, err1 := loader.LoadBySlug(context.Background(), archetype.AssistantSlug, "org-double")
	if err1 != nil {
		t.Fatalf("first LoadByOrg: %v", err1)
	}
	_, found2, err2 := loader.LoadBySlug(context.Background(), archetype.AssistantSlug, "org-double")
	if err2 != nil {
		t.Fatalf("second LoadByOrg: %v", err2)
	}
	if found1 || found2 {
		t.Errorf("found1=%v found2=%v, want both false", found1, found2)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM archetype_configurations WHERE archetype_slug = $1`, archetype.AssistantSlug).Scan(&count); err != nil {
		t.Fatalf("count rows: %v", err)
	}
	if count != 0 {
		t.Errorf("archetype_configurations total rows = %d, want 0 (REQ-CACS-003: two reads must not write)", count)
	}
}

// Test_Loader_FORSHARESerialisesWithWriter — REQ-CACS-002 + design
// §6 forward-only + AD-2 (FOR SHARE row-level lock). Given a row
// exists and goroutine A holds LoadByOrg inside a tx (SELECT ... FOR
// SHARE), when goroutine B issues UPDATE archetype_configurations from
// a separate connection, then B's update blocks until A's tx
// commits. The serialisation invariant is the only honest verification
// of FOR SHARE; SELECT alone does not exhibit the blocking behaviour.
//
// Implementation note: the test uses two distinct *sql.DB connections
// (each holds its own transaction) — sharing one *sql.DB across
// goroutines deadlocks against the connection pool. A holds the load
// for 250ms inside an active tx; B's UPDATE runs on a separate
// connection; the test asserts the wall-clock ordering (B's update
// is blocked until A commits).
//
// RED at T-01: archetype.NewPostgresLoader.WithTx is
// undefined. GREEN (T-02) adds the WithTx helper that binds the
// loader's FOR SHARE query to a caller-provided tx.
func Test_Loader_FORSHARESerialisesWithWriter(t *testing.T) {
	dsn := assistantConfigRequiresPostgres(t)
	resetAssistantConfigTable(t, dsn)

	seed := archetype.ArchetypeConfig{
		Slug:           archetype.AssistantSlug,
		OrgID:        "org-forshare",
		SystemPrompt: "before",
		ToolAllowlist: []string{"current_time"},
		Version:      1,
		UpdatedBy:    "user_seed",
	}
	seedRow(t, dsn, seed)

	connA, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open A: %v", err)
	}
	defer func() { _ = connA.Close() }()
	loaderA := archetype.NewPostgresLoader(connA)

	connB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open B: %v", err)
	}
	defer func() { _ = connB.Close() }()

	var (
		aCfg            archetype.ArchetypeConfig
		aFound          bool
		writerElapsed   time.Duration
		writerCompleted = make(chan struct{})
	)

	var wg sync.WaitGroup
	wg.Add(1)

	// B's writer goroutine: open a tx, UPDATE, measure elapsed time,
	// commit. The elapsed time is the time-from-Exec-to-Commit; if FOR
	// SHARE held B back, writerElapsed should be ~at least the sleep
	// A held inside its tx.
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tx, berr := connB.BeginTx(ctx, nil)
		if berr != nil {
			t.Errorf("BeginTx B: %v", berr)
			close(writerCompleted)
			return
		}
		writerStart := time.Now()
		if _, berr := tx.ExecContext(ctx, `UPDATE archetype_configurations SET system_prompt = $1 WHERE archetype_slug = $2 AND org_id = $3`, "after", string(archetype.AssistantSlug), "org-forshare"); berr != nil {
			t.Errorf("UPDATE B: %v", berr)
			_ = tx.Rollback()
			close(writerCompleted)
			return
		}
		if berr := tx.Commit(); berr != nil {
			t.Errorf("Commit B: %v", berr)
			close(writerCompleted)
			return
		}
		writerElapsed = time.Since(writerStart)
		close(writerCompleted)
	}()

	// A holds the load inside txA. txA's FOR SHARE lock blocks B's
	// UPDATE; A commits after 250ms, releasing the lock.
	holdCtx, holdCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer holdCancel()
	txA, err := connA.BeginTx(holdCtx, nil)
	if err != nil {
		t.Fatalf("BeginTx A: %v", err)
	}
	txLoader := loaderA.WithTx(txA)
	aCfg, aFound, err = txLoader.LoadBySlug(holdCtx, archetype.AssistantSlug, "org-forshare")
	if err != nil {
		t.Fatalf("LoadByOrg (A, tx-bound): %v", err)
	}
	time.Sleep(250 * time.Millisecond)
	if err := txA.Commit(); err != nil {
		t.Fatalf("Commit A: %v", err)
	}

	select {
	case <-writerCompleted:
	case <-time.After(3 * time.Second):
		t.Fatal("writer goroutine did not complete within 3s after A committed (FOR SHARE may be missing)")
	}
	wg.Wait()

	// Sanity: A's load returned found=true with the seeded value.
	if !aFound {
		t.Errorf("A found=false, want true (the seeded row should be visible inside A's tx)")
	}
	if aCfg.SystemPrompt != "before" {
		t.Errorf("A SystemPrompt = %q, want \"before\"", aCfg.SystemPrompt)
	}

	// Ordering invariant: writerElapsed measures time B was blocked
	// (waiting on A's lock) plus the time B took to UPDATE+commit. If
	// FOR SHARE was applied, B should have been blocked for at least
	// ~150ms (the time A held the lock AFTER issuing LoadByOrg, minus
	// the time B took to issue its UPDATE after the load returned).
	if writerElapsed < 100*time.Millisecond {
		t.Errorf("writerElapsed = %v, want >= 100ms (FOR SHARE did not block B; A committed too quickly or no lock was held)", writerElapsed)
	}
}

// Test_Defaults_Pure — design AD-2 + REQ-CACS-003 contract: the
// in-memory default factory must be a pure function. Same input →
// same output, no I/O. Runs without INTEGRATION.
//
// RED at T-01.
func Test_Defaults_Pure(t *testing.T) {
	t.Parallel()
	const org = "org-pure"
	tools := []string{"current_time", "summarize_conversation"}

	first := archetype.DefaultConfig(archetype.AssistantSlug, org, tools)
	second := archetype.DefaultConfig(archetype.AssistantSlug, org, tools)

	if first.OrgID != second.OrgID {
		t.Errorf("OrgID drifted: first=%q second=%q", first.OrgID, second.OrgID)
	}
	if first.SystemPrompt != second.SystemPrompt {
		t.Errorf("SystemPrompt drifted: first=%q second=%q", first.SystemPrompt, second.SystemPrompt)
	}
	if !reflect.DeepEqual(first.ToolAllowlist, second.ToolAllowlist) {
		t.Errorf("ToolAllowlist drifted: first=%v second=%v", first.ToolAllowlist, second.ToolAllowlist)
	}
	if first.Version != second.Version {
		t.Errorf("Version drifted: first=%d second=%d", first.Version, second.Version)
	}
	if first.Version != 1 {
		t.Errorf("Version = %d, want 1 (defaults are version=1 per design AD-2)", first.Version)
	}
	if first.UpdatedBy != "" {
		t.Errorf("UpdatedBy = %q, want empty (defaults carry no actor)", first.UpdatedBy)
	}
}

// Test_Validation_SentinelsExist — sanity check the validation error
// sentinels (REQ-CACAPI-002 + REQ-CACAPI-003) are exported and
// distinguishable via errors.Is. The body of each sentinel is
// verified separately by PUT tests in PR-3 (T-13); here we only
// prove the symbols exist and form a typed discriminated set.
//
// RED at T-01: archetype.ErrSystemPromptTooLong and the rest are
// undefined.
func Test_Validation_SentinelsExist(t *testing.T) {
	t.Parallel()
	sentinels := []error{
		archetype.ErrSystemPromptTooLong,
		archetype.ErrSystemPromptContainsHTML,
		archetype.ErrUnknownToolName,
		archetype.ErrDeferToolNotInAllowlist,
		archetype.ErrToolAllowlistEmpty,
		archetype.ErrSystemPromptEmpty,
	}
	if len(sentinels) != 6 {
		t.Errorf("expected 6 sentinels, got %d", len(sentinels))
	}
	// Each sentinel must be distinct (no duplicates the validator
	// would silently swallow).
	for i := range sentinels {
		for j := i + 1; j < len(sentinels); j++ {
			if errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinels[%d] matches sentinels[%d] via errors.Is (they must be distinct)", i, j)
			}
		}
	}
}