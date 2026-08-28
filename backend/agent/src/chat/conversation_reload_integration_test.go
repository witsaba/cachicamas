// S1-E (T3.2 of cachicamas-agent-catalog-config-reload) — INTEGRATION-gated
// end-to-end reload test against a real Postgres.
//
// Test_PutConfig_NextTurn_ReloadsPrompt proves the full PUT→reload chain
// (CRL-S-023, CRL-S-024, CRL-S-026):
//
//	archetype.PostgresWriter.WriteConfig (v1, prompt A)
//	  → chat.NewConversation(AssistantConfigLoader=archetype.NewPostgresLoader)
//	    → SystemPromptForTest == A, LoadedAssistantConfigVersion == 1
//	      → WriteConfig (v2, prompt B)
//	        → ReloadAssistantConfig (the Send-boundary call, conversation.go)
//	          → SystemPromptForTest == B, version == 2
//	            → reload with NO write → version stays 2, prompt B
//	              (the version-match no-op; CRL-S-024).
//
// Mid-turn isolation (CRL-S-026) is asserted by the boundary-only design
// of this test: v2 is written only AFTER the first turn completes, and
// the prompt swap is only observed at the NEXT boundary — an in-flight
// turn can never see a mid-turn write because reloads happen at the
// boundary before Harness.Run, not inside a turn (conversation.go:356).
//
// Harness: a fresh throwaway database migrates via the composition-root
// order (Run0006IfNeeded → Run0007IfNeeded → goose Up, run TWICE for
// two-boot convergence — the same contract bootstrap_integration_test.go
// documents). The 0006 seed makes the `assistant` catalog row exist, so
// the test's per-org WriteConfig has a valid FK parent; no manual seed
// INSERT is needed. Skips cleanly without a DSN (NFR-CRL-006).
package chat_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"net/url"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/archetype"
	archetypeMigrations "github.com/cachicamas/backend/agent/src/archetype/migrations"
	"github.com/cachicamas/backend/agent/src/chat"
	"github.com/cachicamas/backend/agent/src/chat/migrator"
)

// swapDSNDatabase returns the DSN pointing at dbName on the same server
// as baseDSN (URL-form and keyword=value form both supported).
func swapDSNDatabase(baseDSN, dbName string) (string, bool) {
	if u, err := url.Parse(baseDSN); err == nil && u.Scheme != "" {
		u.Path = "/" + dbName
		return u.String(), true
	}
	for _, field := range strings.Fields(baseDSN) {
		if strings.HasPrefix(field, "dbname=") {
			return strings.Replace(baseDSN, field, "dbname="+dbName, 1), true
		}
	}
	return baseDSN + " dbname=" + dbName, true
}

// freshReloadDatabase creates a uniquely named empty database on the
// server behind baseDSN and returns a DSN pointing at it, dropped via
// t.Cleanup (DROP DATABASE … WITH (FORCE) needs PG 13+; the compose
// image is postgres:18-alpine3.24).
func freshReloadDatabase(t *testing.T, baseDSN string) string {
	t.Helper()

	adminDSN, _ := swapDSNDatabase(baseDSN, "postgres")
	admin, err := sql.Open("pgx", adminDSN)
	if err != nil {
		t.Fatalf("freshReloadDatabase: sql.Open admin: %v", err)
	}
	if err := admin.Ping(); err != nil {
		t.Fatalf("freshReloadDatabase: ping admin: %v", err)
	}
	name := fmt.Sprintf("cachicamas_chat_reload_test_%d_%d", os.Getpid(), time.Now().UnixNano())
	// pi-lens-ignore: go-sql-injection
	if _, err := admin.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, name)); err != nil {
		t.Fatalf("freshReloadDatabase: CREATE DATABASE %q: %v", name, err)
	}
	t.Cleanup(func() {
		// pi-lens-ignore: go-sql-injection
		if _, err := admin.Exec(fmt.Sprintf(`DROP DATABASE IF EXISTS "%s" WITH (FORCE)`, name)); err != nil {
			t.Logf("freshReloadDatabase: DROP DATABASE %q: %v (leftover may need manual cleanup)", name, err)
		}
		_ = admin.Close()
	})
	dsn, _ := swapDSNDatabase(baseDSN, name)
	return dsn
}

// bootstrapFreshDB runs the composition-root migration sequence
// (src/cmd/chat/migrators.go:49-63) TWICE against the DSN. Boot-1 state
// is deliberately not asserted (see bootstrap_integration_test.go for
// the two-boot convergence contract).
func bootstrapFreshDB(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	boot := func(n int) {
		t.Helper()
		if err := archetypeMigrations.Run0006IfNeeded(ctx, db); err != nil {
			t.Fatalf("boot %d: Run0006IfNeeded: %v", n, err)
		}
		if err := archetypeMigrations.Run0007IfNeeded(ctx, db); err != nil {
			t.Fatalf("boot %d: Run0007IfNeeded: %v", n, err)
		}
		provider, err := migrator.NewProvider(ctx, db, archetypeMigrations.MigrationsFS, "archetype_schema_migrations")
		if err != nil {
			t.Fatalf("boot %d: migrator.NewProvider: %v", n, err)
		}
		if _, err := provider.Up(ctx); err != nil {
			t.Fatalf("boot %d: provider.Up: %v", n, err)
		}
	}
	boot(1)
	boot(2)
}

// writeAssistantConfig marshals update via the production
// PostgresWriter and asserts the write succeeded with the expected
// version (v1 on first write, prevVersion+1 afterwards).
func writeAssistantConfig(t *testing.T, db *sql.DB, orgID string, update archetype.ConfigUpdate, wantVersion int) {
	t.Helper()
	writer := archetype.NewPostgresWriter(db)
	cfg, err := writer.WriteConfig(context.Background(), archetype.AssistantSlug, orgID, update, "chat-reload-test")
	if err != nil {
		t.Fatalf("WriteConfig (want v%d): %v", wantVersion, err)
	}
	if cfg.Version != wantVersion {
		t.Fatalf("WriteConfig version = %d, want %d", cfg.Version, wantVersion)
	}
}

// newReloadConversation builds a Conversation whose assistant config
// comes from the real Postgres loader against the throwaway DB — the
// main.go:203-240 composition shape (same DSN pool as the writer).
func newReloadConversation(t *testing.T, db *sql.DB, participantID string) *chat.Conversation {
	t.Helper()
	conv, err := chat.NewConversation(chat.Config{
		Provider:              scriptedProvider(t),
		Store:                 chat.NewMemoryConversationStore(),
		ParticipantID:         participantID,
		ToolSource:            chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy:      chat.NewDefaultPermissionPolicy(nil),
		AssistantConfigLoader: archetype.NewPostgresLoader(db),
	})
	if err != nil {
		t.Fatalf("chat.NewConversation: %v", err)
	}
	return conv
}

func Test_PutConfig_NextTurn_ReloadsPrompt(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run (the PUT→next-turn reload chain requires a live Postgres)")
	}
	baseDSN := reloadRequiresPostgres(t)
	dsn := freshReloadDatabase(t, baseDSN)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer func() { _ = db.Close() }()

	bootstrapFreshDB(t, db)

	const orgID = "org-reload-test"

	// v1: prompt A.
	writeAssistantConfig(t, db, orgID, archetype.ConfigUpdate{
		SystemPrompt:   "reload prompt A",
		ToolAllowlist:  []string{"current_time"},
		DeferToolNames: []string{"current_time"},
	}, 1)

	// Conversation seeds from the v1 row at construction.
	conv := newReloadConversation(t, db, orgID)
	if got := conv.SystemPromptForTest(); got != "reload prompt A" {
		t.Fatalf("initial SystemPromptForTest = %q, want %q", got, "reload prompt A")
	}
	if got := conv.LoadedAssistantConfigVersion(); got != 1 {
		t.Fatalf("initial LoadedAssistantConfigVersion = %d, want 1", got)
	}

	// v2: prompt B — written only after the first turn boundary, per
	// the CRL-S-026 mid-turn isolation contract.
	writeAssistantConfig(t, db, orgID, archetype.ConfigUpdate{
		SystemPrompt:   "reload prompt B",
		ToolAllowlist:  []string{"current_time"},
		DeferToolNames: []string{"current_time"},
	}, 2)

	// The Send-boundary call (conversation.go:356) picks up v2 (CRL-S-023).
	if err := conv.ReloadAssistantConfig(context.Background()); err != nil {
		t.Fatalf("ReloadAssistantConfig: %v", err)
	}
	if got := conv.SystemPromptForTest(); got != "reload prompt B" {
		t.Fatalf("after v2 reload SystemPromptForTest = %q, want %q", got, "reload prompt B")
	}
	if got := conv.LoadedAssistantConfigVersion(); got != 2 {
		t.Fatalf("after v2 reload LoadedAssistantConfigVersion = %d, want 2", got)
	}

	// Reload with NO write: version-match no-op (CRL-S-024).
	if err := conv.ReloadAssistantConfig(context.Background()); err != nil {
		t.Fatalf("no-op ReloadAssistantConfig: %v", err)
	}
	if got := conv.SystemPromptForTest(); got != "reload prompt B" {
		t.Errorf("no-op reload SystemPromptForTest = %q, want %q", got, "reload prompt B")
	}
	if got := conv.LoadedAssistantConfigVersion(); got != 2 {
		t.Errorf("no-op reload LoadedAssistantConfigVersion = %d, want 2 (match must not re-apply)", got)
	}
}

// reloadRequiresPostgres mirrors catalogRequiresPostgres: skips unless
// INTEGRATION=1, then returns the DSN (env override or the compose
// default). The chat package has no Postgres helper of its own; keeping
// a local copy preserves the "skip cleanly without a DSN" contract
// (NFR-CRL-006) without importing test helpers across packages.
func reloadRequiresPostgres(t *testing.T) string {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	dsn := os.Getenv("CACHICAMAS_CHAT_STORE_DSN")
	if dsn == "" {
		dsn = "postgres://cachicamas:changeme-local-only@localhost:5432/cachicamas_pg?sslmode=disable"
	}
	return dsn
}
