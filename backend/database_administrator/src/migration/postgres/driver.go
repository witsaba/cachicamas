// Package postgres contains the Postgres adapter for the migration runner.
// It is the ONLY file (besides migration/runner.go) that imports
// github.com/jackc/pgx — this preserves the hexagonal rule that
// domain/application code does not depend on a concrete database driver.
//
// The adapter exposes two surfaces:
//
//   - LoadConfigFromEnv: env -> Config, with DATABASE_URL taking
//     precedence over the discrete POSTGRES_* family.
//   - Open: Config -> *sql.DB, with a fail-fast PingContext(ctx) so a
//     misconfigured stack crashes at startup, not at first query
//     (design §6 + §15.1).
//
// Per design §4 the defaults are:
//
//	MaxOpenConns     = 10
//	MaxIdleConns     =  5
//	ConnMaxLifetime  = 30 * time.Minute
//	ConnectTimeout   =  5 * time.Second
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"time"

	// pgx stdlib adapter registers itself as the "pgx" database/sql
	// driver. Importing it for its side effects is the standard pgx
	// usage pattern (see ADR-001 — adr/pgx-v5-stdlib-adapter).
	_ "github.com/jackc/pgx/v5/stdlib"
)

// Config carries the resolved Postgres connection parameters plus the
// pool tunings. The zero value is NOT usable; LoadConfigFromEnv is the
// supported way to construct a Config (it fills in defaults).
type Config struct {
	// DSN is the connection string passed to sql.Open. The pgx stdlib
	// adapter accepts both URL form (postgres://user:pass@host:port/db
	// ?sslmode=...) and keyword form (host=... port=... dbname=...).
	DSN string

	// MaxOpenConns is the upper bound on concurrently open connections
	// to the database. Zero means use the database/sql default
	// (unlimited, not recommended). The default applied by
	// LoadConfigFromEnv is 10.
	MaxOpenConns int

	// MaxIdleConns is the upper bound on idle connections retained in
	// the pool. The default applied by LoadConfigFromEnv is 5.
	MaxIdleConns int

	// ConnMaxLifetime is the maximum age of a connection before it is
	// closed and replaced. The default applied by LoadConfigFromEnv is
	// 30 minutes; this matches the convention of recycling connections
	// on a periodic basis so DNS / load-balancer changes propagate.
	ConnMaxLifetime time.Duration

	// ConnectTimeout is the per-dial timeout for the underlying pgx
	// network call. The default applied by LoadConfigFromEnv is 5s.
	// The migration runner bounds the overall call with a context
	// timeout on top of this.
	ConnectTimeout time.Duration
}

// Default tunings (design §4).
const (
	defaultMaxOpenConns    = 10
	defaultMaxIdleConns    = 5
	defaultConnMaxLifetime = 30 * time.Minute
	defaultConnectTimeout  = 5 * time.Second
	defaultPort            = "5432"
	// defaultSSLMODE is the fallback sslmode used when POSTGRES_SSLMODE
	// is unset on the discrete-vars path. "require" encrypts the
	// connection but does not verify the server certificate — a
	// documented dev-friendly default that matches the project
	// convention. DATABASE_URL-pass-through is unchanged (the URL
	// owns its own sslmode).
	defaultSSLMODE = "require"
)

// LoadConfigFromEnv reads Postgres connection parameters from the
// process environment.
//
// Precedence (locked at design):
//
//  1. If DATABASE_URL is set and non-empty, it is used verbatim as the
//     DSN. The discrete POSTGRES_* variables are ignored.
//  2. Otherwise, POSTGRES_HOST, POSTGRES_PORT (default "5432"),
//     POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD are required. A
//     keyword-form DSN is assembled from them.
//  3. If neither path produces a usable set of parameters, the call
//     returns a non-nil error.
//
// The function never panics. It never connects. The caller passes the
// returned Config to Open.
func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		MaxOpenConns:    defaultMaxOpenConns,
		MaxIdleConns:    defaultMaxIdleConns,
		ConnMaxLifetime: defaultConnMaxLifetime,
		ConnectTimeout:  defaultConnectTimeout,
	}

	if url := os.Getenv("DATABASE_URL"); url != "" {
		cfg.DSN = url
		return cfg, nil
	}

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	db := os.Getenv("POSTGRES_DB")
	user := os.Getenv("POSTGRES_USER")
	pass := os.Getenv("POSTGRES_PASSWORD")

	if host == "" || db == "" || user == "" || pass == "" {
		return Config{}, fmt.Errorf(
			"postgres.LoadConfigFromEnv: need either DATABASE_URL or " +
				"all of POSTGRES_HOST, POSTGRES_DB, POSTGRES_USER, POSTGRES_PASSWORD " +
				"(POSTGRES_PORT defaults to 5432)")
	}
	if port == "" {
		port = defaultPort
	}

	// Build a URL-form DSN. The pgx stdlib parser accepts URL form
	// and keyword form interchangeably; URL form is easier to read
	// and to log. The password is URL-escaped so a special character
	// in POSTGRES_PASSWORD (e.g. '#', '?') does not break the DSN.
	//
	// SSL mode is read from POSTGRES_SSLMODE (security M-2). Default
	// is "require" — the driver encrypts the connection but does not
	// verify the certificate. Set POSTGRES_SSLMODE=disable in dev
	// when the local Postgres doesn't terminate TLS. Operators wanting
	// full chain-of-trust set POSTGRES_SSLMODE=verify-full.
	sslmode := os.Getenv("POSTGRES_SSLMODE")
	if sslmode == "" {
		sslmode = defaultSSLMODE
	}
	cfg.DSN = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		url.QueryEscape(user),
		url.QueryEscape(pass),
		host,
		port,
		db,
		sslmode,
	)
	return cfg, nil
}

// Open returns a *sql.DB ready to use against the configured Postgres
// instance. It registers no driver (the blank import in this package
// does that at init time) and applies the pool settings from cfg before
// returning.
//
// The fail-fast guarantee from design §6 + §15.1 is implemented by the
// PingContext call: if the host is unreachable, the credentials are
// wrong, or the database does not exist, Open returns a wrapped error
// without ever handing a broken *sql.DB to the caller.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, fmt.Errorf("postgres.Open: empty DSN in Config")
	}

	// sql.Open does not actually dial — it only validates the DSN
	// format and returns a *sql.DB that will dial lazily on first
	// query. We force the dial here with PingContext so a misconfigured
	// stack fails before Echo binds the listener.
	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("postgres.Open: sql.Open: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		// Close the broken pool so we don't leak the goroutine and
		// connections sql.Open parked in the background.
		_ = db.Close()
		return nil, fmt.Errorf("postgres.Open: ping: %w", err)
	}

	applyPoolSettings(db, cfg)
	return db, nil
}

// applyPoolSettings copies the pool tunings from Config onto the
// *sql.DB. It is unexported because callers go through Open; tests
// exercise it directly to assert the wiring (see driver_test.go).
func applyPoolSettings(db *sql.DB, cfg Config) {
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
}
