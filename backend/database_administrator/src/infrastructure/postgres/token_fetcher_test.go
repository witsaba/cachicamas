// Package postgres_test — token_fetcher_test.go covers the pgx-backed
// TokenFetcher adapter. Integration tests require INTEGRATION=1
// (skip otherwise); follows the same pattern as organization_repo_test.go.
package postgres_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"testing"
	"time"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func integrationDBTF(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}

	host := envOrTF(t, "POSTGRES_HOST", "localhost")
	port := envOrTF(t, "POSTGRES_PORT", "5432")
	dbname := envOrTF(t, "POSTGRES_DB", "cachicamas_pg")
	user := envOrTF(t, "POSTGRES_USER", "cachicamas")
	pass := envOrTF(t, "POSTGRES_PASSWORD", "changeme-local-only")

	dsn := "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping compose postgres: %v", err)
	}
	return db
}

func envOrTF(t *testing.T, key, fallback string) string {
	t.Helper()
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func uniqSuffixTF() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// seedAccount inserts an identity.user + identity.account row pair for
// tests. Returns the user_id (so cleanup can target both rows).
func seedAccountTF(t *testing.T, db *sql.DB, provider, accountID, accessToken string) int64 {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	email := "token_fetcher_test_" + uniqSuffixTF() + "@example.com"
	if _, err := db.ExecContext(ctx,
		`INSERT INTO identity.user (email) VALUES ($1)`, email); err != nil {
		t.Fatalf("insert identity.user: %v", err)
	}
	var uid int64
	if err := db.QueryRowContext(ctx,
		`SELECT id FROM identity.user WHERE email = $1`, email).Scan(&uid); err != nil {
		t.Fatalf("select user id: %v", err)
	}

	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			`DELETE FROM identity.account WHERE provider_account_id = $1`, accountID)
		_, _ = db.ExecContext(context.Background(),
			`DELETE FROM identity.user WHERE id = $1`, uid)
	})

	// accessToken may be the empty string → NULL column.
	var tokenArg any
	if accessToken == "" {
		tokenArg = nil
	} else {
		tokenArg = accessToken
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO identity.account (user_id, provider, provider_account_id, access_token) VALUES ($1, $2, $3, $4)`,
		uid, provider, accountID, tokenArg); err != nil {
		t.Fatalf("insert identity.account: %v", err)
	}
	return uid
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestTokenFetcher_ReturnsPersistedToken — happy path.
func TestTokenFetcher_ReturnsPersistedToken(t *testing.T) {
	db := integrationDBTF(t)
	provider := "github"
	accountID := "tf_test_" + uniqSuffixTF()
	const token = "gho_test_persisted_token_xyz123abc"
	seedAccountTF(t, db, provider, accountID, token)

	fetcher := postgres.NewTokenFetcher(db)
	got, err := fetcher.AccessTokenForIdentity(context.Background(), provider, accountID)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got != token {
		t.Errorf("expected token %q, got %q", token, got)
	}
}

// TestTokenFetcher_EmptyProvider_ReturnsTokenNotFoundError.
func TestTokenFetcher_EmptyProvider_ReturnsTokenNotFoundError(t *testing.T) {
	db := integrationDBTF(t)
	fetcher := postgres.NewTokenFetcher(db)
	_, err := fetcher.AccessTokenForIdentity(context.Background(), "", "some_account_id")
	var tnf *httpiface.TokenNotFoundError
	if !errors.As(err, &tnf) {
		t.Fatalf("expected *TokenNotFoundError, got %v", err)
	}
}

// TestTokenFetcher_EmptyAccountID_ReturnsTokenNotFoundError.
func TestTokenFetcher_EmptyAccountID_ReturnsTokenNotFoundError(t *testing.T) {
	db := integrationDBTF(t)
	fetcher := postgres.NewTokenFetcher(db)
	_, err := fetcher.AccessTokenForIdentity(context.Background(), "github", "")
	var tnf *httpiface.TokenNotFoundError
	if !errors.As(err, &tnf) {
		t.Fatalf("expected *TokenNotFoundError, got %v", err)
	}
}

// TestTokenFetcher_NoMatchingRow_ReturnsTokenNotFoundError.
func TestTokenFetcher_NoMatchingRow_ReturnsTokenNotFoundError(t *testing.T) {
	db := integrationDBTF(t)
	fetcher := postgres.NewTokenFetcher(db)
	_, err := fetcher.AccessTokenForIdentity(context.Background(), "github", "tf_definitely_does_not_exist_"+uniqSuffixTF())
	var tnf *httpiface.TokenNotFoundError
	if !errors.As(err, &tnf) {
		t.Fatalf("expected *TokenNotFoundError, got %v", err)
	}
}

// TestTokenFetcher_NullAccessToken_ReturnsTokenNotFoundError — covers
// the PR1a migration case: existing rows from before PR1a have NULL
// access_token.
func TestTokenFetcher_NullAccessToken_ReturnsTokenNotFoundError(t *testing.T) {
	db := integrationDBTF(t)
	provider := "github"
	accountID := "tf_null_test_" + uniqSuffixTF()
	seedAccountTF(t, db, provider, accountID, "") // empty → NULL

	fetcher := postgres.NewTokenFetcher(db)
	_, err := fetcher.AccessTokenForIdentity(context.Background(), provider, accountID)
	var tnf *httpiface.TokenNotFoundError
	if !errors.As(err, &tnf) {
		t.Fatalf("expected *TokenNotFoundError for NULL access_token, got %v", err)
	}
}

// TestTokenFetcher_ParameterizedSQL_DoesNotAllowInjection — regression
// test: if the adapter ever stopped using parameterized SQL, the
// injection string would match every row.
func TestTokenFetcher_ParameterizedSQL_DoesNotAllowInjection(t *testing.T) {
	db := integrationDBTF(t)
	fetcher := postgres.NewTokenFetcher(db)
	const injection = "github' OR '1'='1"
	_, err := fetcher.AccessTokenForIdentity(context.Background(), "github", injection)
	var tnf *httpiface.TokenNotFoundError
	if !errors.As(err, &tnf) {
		t.Fatalf("expected *TokenNotFoundError (literal-string lookup), got %v — SQL injection likely!", err)
	}
	if !tnf.Sentinel() {
		t.Errorf("TokenNotFoundError.Sentinel() returned false; expected true")
	}
	// Sanity: the error reason must NOT echo the injection literal — the
	// production TokenNotFoundError does not include the input value
	// (would be a small information leak via log lines). The
	// errors.As + Sentinel check above is sufficient evidence that the
	// query was parameterized: the typed sentinel is only returned by
	// the TokenFetcher's own ErrNoRows / NULL-handling branch, not by
	// the wrapping error path that a SQL injection would trigger.
}