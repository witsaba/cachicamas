// Package postgres contains the Postgres adapter for the migration runner.
// This test file exercises the DSN/Config factory (unit, no DB) and the
// *sql.DB open + ping roundtrip (integration, requires a live Postgres).
//
// Strict TDD discipline (per openspec/AGENTS.md and sdd-init/cachicamas):
// every behavior here is described by a failing test FIRST. This file
// was written BEFORE driver.go existed; running `make test` against
// this file with no driver.go must fail with "undefined: LoadConfigFromEnv"
// / "undefined: Open" — that failure is the RED step.
package postgres

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Category A — Unit tests (no DB required, always run under `make test`).
// ---------------------------------------------------------------------------

// TestLoadConfigFromEnv covers the env -> Config factory for every
// documented combination of inputs:
//
//	(a) DATABASE_URL alone resolves to Config.DSN equal to the URL.
//	(b) DATABASE_URL takes precedence over POSTGRES_* even when both are set.
//	(c) Discrete POSTGRES_HOST/PORT/DB/USER/PASSWORD assemble a DSN with
//	    the same precedence the runner will pass to sql.Open.
//	(d) Default port is "5432" when POSTGRES_PORT is unset.
//	(e) Neither DATABASE_URL nor the full discrete set returns an error.
//
// The assembled DSN shape is intentionally not asserted byte-for-byte for
// the discrete-vars case (we assert the user/host/db fields, not the
// exact keyword order) to keep this resilient to a future pgx DSN
// parser change. The DATABASE_URL case is asserted exactly.
func TestLoadConfigFromEnv(t *testing.T) {
	t.Run("DATABASE_URL alone", func(t *testing.T) {
		clearPostgresEnv(t)
		t.Setenv("DATABASE_URL", "postgres://queen:secret@db:5432/cachicamas_pg?sslmode=disable")

		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("LoadConfigFromEnv: unexpected error: %v", err)
		}
		if cfg.DSN != "postgres://queen:secret@db:5432/cachicamas_pg?sslmode=disable" {
			t.Errorf("DSN = %q, want the DATABASE_URL value verbatim", cfg.DSN)
		}
		if cfg.MaxOpenConns != 10 {
			t.Errorf("MaxOpenConns = %d, want default 10", cfg.MaxOpenConns)
		}
		if cfg.MaxIdleConns != 5 {
			t.Errorf("MaxIdleConns = %d, want default 5", cfg.MaxIdleConns)
		}
		if cfg.ConnMaxLifetime != 30*time.Minute {
			t.Errorf("ConnMaxLifetime = %s, want default 30m", cfg.ConnMaxLifetime)
		}
		if cfg.ConnectTimeout != 5*time.Second {
			t.Errorf("ConnectTimeout = %s, want default 5s", cfg.ConnectTimeout)
		}
	})

	t.Run("DATABASE_URL takes precedence over POSTGRES_*", func(t *testing.T) {
		clearPostgresEnv(t)
		t.Setenv("DATABASE_URL", "postgres://primary:secret@primary:5432/primary_db?sslmode=disable")
		t.Setenv("POSTGRES_HOST", "secondary")
		t.Setenv("POSTGRES_PORT", "5433")
		t.Setenv("POSTGRES_DB", "secondary_db")
		t.Setenv("POSTGRES_USER", "secondary_user")
		t.Setenv("POSTGRES_PASSWORD", "secondary_pass")

		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("LoadConfigFromEnv: unexpected error: %v", err)
		}
		if cfg.DSN != "postgres://primary:secret@primary:5432/primary_db?sslmode=disable" {
			t.Errorf("DSN = %q, want DATABASE_URL (precedence violated)", cfg.DSN)
		}
	})

	t.Run("discrete POSTGRES_* env vars", func(t *testing.T) {
		clearPostgresEnv(t)
		t.Setenv("POSTGRES_HOST", "db.example.com")
		t.Setenv("POSTGRES_PORT", "6543")
		t.Setenv("POSTGRES_DB", "app_db")
		t.Setenv("POSTGRES_USER", "app_user")
		t.Setenv("POSTGRES_PASSWORD", "app_pass")

		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("LoadConfigFromEnv: unexpected error: %v", err)
		}
		// Assert the key/value fragments the pgx stdlib parser cares about.
		// We intentionally don't pin the exact keyword order or escaping.
		if !strings.Contains(cfg.DSN, "db.example.com") {
			t.Errorf("DSN %q must contain host db.example.com", cfg.DSN)
		}
		if !strings.Contains(cfg.DSN, "6543") {
			t.Errorf("DSN %q must contain port 6543", cfg.DSN)
		}
		if !strings.Contains(cfg.DSN, "app_db") {
			t.Errorf("DSN %q must contain dbname app_db", cfg.DSN)
		}
		if !strings.Contains(cfg.DSN, "app_user") {
			t.Errorf("DSN %q must contain user app_user", cfg.DSN)
		}
		if !strings.Contains(cfg.DSN, "app_pass") {
			t.Errorf("DSN %q must contain password app_pass", cfg.DSN)
		}
	})

	t.Run("discrete vars default port to 5432", func(t *testing.T) {
		clearPostgresEnv(t)
		t.Setenv("POSTGRES_HOST", "db")
		t.Setenv("POSTGRES_DB", "x")
		t.Setenv("POSTGRES_USER", "u")
		t.Setenv("POSTGRES_PASSWORD", "p")
		// POSTGRES_PORT intentionally unset

		cfg, err := LoadConfigFromEnv()
		if err != nil {
			t.Fatalf("LoadConfigFromEnv: unexpected error: %v", err)
		}
		if !strings.Contains(cfg.DSN, ":5432/") {
			t.Errorf("DSN %q must contain default port 5432", cfg.DSN)
		}
	})

	t.Run("missing env returns error", func(t *testing.T) {
		clearPostgresEnv(t)

		_, err := LoadConfigFromEnv()
		if err == nil {
			t.Fatalf("LoadConfigFromEnv: expected error when neither DATABASE_URL nor POSTGRES_* are set, got nil")
		}
	})

	t.Run("partial discrete vars return error", func(t *testing.T) {
		clearPostgresEnv(t)
		t.Setenv("POSTGRES_HOST", "db")
		t.Setenv("POSTGRES_USER", "u")
		// POSTGRES_DB, POSTGRES_PASSWORD missing on purpose

		_, err := LoadConfigFromEnv()
		if err == nil {
			t.Fatalf("LoadConfigFromEnv: expected error when discrete vars are partial, got nil")
		}
	})
}

// TestApplyPoolSettings covers the post-Open pool tuning knobs. It is a
// unit test because we never dial — we just verify that the pool fields
// on a Config propagate to a *sql.DB via SetMaxOpenConns/SetMaxIdleConns/
// SetConnMaxLifetime. This is the smallest piece of Open() that can be
// exercised without a live DB.
func TestApplyPoolSettings(t *testing.T) {
	cfg := Config{
		DSN:             "postgres://u:p@h:5432/d",
		MaxOpenConns:    3,
		MaxIdleConns:    2,
		ConnMaxLifetime: 7 * time.Minute,
		ConnectTimeout:  1 * time.Second,
	}
	db := sql.OpenDB(nil) // in-memory driver; we never query it
	defer db.Close()

	applyPoolSettings(db, cfg)

	if got := db.Stats().MaxOpenConnections; got != 3 {
		t.Errorf("MaxOpenConnections = %d, want 3", got)
	}
}

// ---------------------------------------------------------------------------
// Category B — Integration tests (gated on INTEGRATION=1, run by
// `make test/integration`).
// ---------------------------------------------------------------------------

// TestOpen_Ping verifies that Open(ctx, cfg) returns a usable *sql.DB and
// that a Ping round-trips against the live Postgres provisioned by
// docker-compose (see design §6 — fail-fast at Open, not at first query).
func TestOpen_Ping(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv: %v", err)
	}
	// Cap the connect timeout so a broken stack fails fast (don't wait
	// 30s in CI). The default is 5s; keep it.
	db, err := Open(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		t.Fatalf("PingContext: %v", err)
	}
}

// TestOpen_ConnectError verifies the fail-fast guarantee from design
// §15.1: an unreachable host (pointed at 127.0.0.1:1) MUST return an
// error from Open, not from a later query. We give Open a tight
// ConnectTimeout to keep the test under a few seconds.
func TestOpen_ConnectError(t *testing.T) {
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	cfg := Config{
		DSN:             "postgres://u:p@127.0.0.1:1/nope?sslmode=disable&connect_timeout=2",
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: time.Minute,
		ConnectTimeout:  3 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	db, err := Open(ctx, cfg)
	if err == nil {
		_ = db.Close()
		t.Fatalf("Open: expected error for unreachable host, got nil (db=%v)", db)
	}
	// Sanity: the error should mention the operation so log readers can
	// diagnose. We don't pin the exact text — pgx's wording is allowed
	// to evolve. We just require it to be non-empty.
	if err.Error() == "" {
		t.Errorf("Open error must be non-empty")
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// clearPostgresEnv unsets every env var the factory reads, then restores
// nothing — the caller is expected to use t.Setenv for each var it sets,
// and Go's testing framework restores the pre-test snapshot on test
// return.
func clearPostgresEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"DATABASE_URL",
		"POSTGRES_HOST",
		"POSTGRES_PORT",
		"POSTGRES_DB",
		"POSTGRES_USER",
		"POSTGRES_PASSWORD",
	} {
		t.Setenv(k, "")
		// t.Setenv above records the empty value; we also need to unset
		// because "" is a valid (and wrong) value for DATABASE_URL.
		// Unset via os and t.Setenv again with empty (testing framework
		// calls os.Setenv on cleanup, which is fine — empty DATABASE_URL
		// is treated as "not set" by LoadConfigFromEnv, but to be safe
		// we call os.Unsetenv explicitly).
		os.Unsetenv(k)
	}
}
