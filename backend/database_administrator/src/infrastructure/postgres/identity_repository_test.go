// Package postgres_test contains the integration test suite for the
// identity repository adapter. The tests are gated on INTEGRATION=1
// because they need a live Postgres (the same compose-provisioned
// instance used by the migration runner tests and the organization
// repo tests). The handler-level unit tests in
// src/interfaces/http/auth_middleware_test.go (PR-3) cover the
// non-DB paths with a fake.
//
// Strict TDD discipline (per openspec/AGENTS.md and
// sdd-init/cachicamas): this file was written BEFORE
// identity_repository.go existed. Running
// `INTEGRATION=1 go test ./src/infrastructure/postgres/... -run TestIdentityRepository`
// with no PostgresIdentityRepo type must fail with "undefined:
// PostgresIdentityRepo" — that failure IS the RED step.
package postgres_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"database/sql"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/infrastructure/postgres"
)

// ---------------------------------------------------------------------------
// Helpers (re-use the shape used by organization_repo_test.go)
// ---------------------------------------------------------------------------

func identityIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("INTEGRATION") != "1" {
		t.Skip("integration; set INTEGRATION=1 to run")
	}
	host := envOrIdentity(t, "POSTGRES_HOST", "localhost")
	port := envOrIdentity(t, "POSTGRES_PORT", "5432")
	dbname := envOrIdentity(t, "POSTGRES_DB", "cachicamas_pg")
	user := envOrIdentity(t, "POSTGRES_USER", "cachicamas")
	pass := envOrIdentity(t, "POSTGRES_PASSWORD", "changeme-local-only")
	dsn := "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + dbname + "?sslmode=disable"
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("ping compose postgres: %v (is `make test/integration` running?)", err)
	}
	return db
}

// envOrIdentity returns the env value or a default. Local alias of
// the helper in organization_repo_test.go (which is package-private
// in the same package, so we can reuse it via identityIntegrationDB
// — but for readability we keep this short helper here too).
func envOrIdentity(t *testing.T, key, def string) string {
	t.Helper()
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return def
}

// seedIdentity inserts a fresh identity.user row + identity.account
// row directly via SQL and returns the user id. The function is
// used by every sub-test that needs a known row to look up. It
// returns a cleanup func that DELETEs the seeded rows so tests do
// not leave data behind.
//
// Implementation note: we INSERT directly (instead of calling the
// repository) because the repository only exposes LookupByEmail;
// the slice does not yet own the create / update use case (those
// live on the frontend signIn callback in PR-2).
func seedIdentity(t *testing.T, db *sql.DB, email, name, provider, providerAccountID string) (userID int64, cleanup func()) {
	t.Helper()
	ctx := context.Background()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := tx.QueryRowContext(ctx,
		`INSERT INTO identity.user (email, name) VALUES ($1, $2) RETURNING id`,
		email, name,
	).Scan(&userID); err != nil {
		t.Fatalf("seed identity.user: %v", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO identity.account (user_id, provider, provider_account_id) VALUES ($1, $2, $3)`,
		userID, provider, providerAccountID,
	); err != nil {
		t.Fatalf("seed identity.account: %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("seed commit: %v", err)
	}
	cleanup = func() {
		_, _ = db.ExecContext(context.Background(),
			`DELETE FROM identity.user WHERE id = $1`, userID,
		)
	}
	return userID, cleanup
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestIdentityRepository_LookupByEmail_Hit is the happy path:
// seed a user + account, look up by email, assert all fields are
// returned correctly. Per spec S-IS (identity-schema) and the
// backend-auth-middleware spec R-BAM-040 / S-BAM-040.
func TestIdentityRepository_LookupByEmail_Hit(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	const email = "lookup-hit@example.com"
	const name = "Lookup Hit"
	const provider = "github"
	const pid = "12345"
	userID, cleanup := seedIdentity(t, db, email, name, provider, pid)
	defer cleanup()

	got, err := repo.LookupByEmail(context.Background(), email)
	if err != nil {
		t.Fatalf("LookupByEmail(%q): unexpected error: %v", email, err)
	}
	if got.ID != userID {
		t.Errorf("ID:\n got  = %d\n want = %d", got.ID, userID)
	}
	if got.Email != email {
		t.Errorf("Email:\n got  = %q\n want = %q", got.Email, email)
	}
	if got.Name != name {
		t.Errorf("Name:\n got  = %q\n want = %q", got.Name, name)
	}
	if got.Provider != provider {
		t.Errorf("Provider:\n got  = %q\n want = %q", got.Provider, provider)
	}
	if got.ProviderAccountID != pid {
		t.Errorf("ProviderAccountID:\n got  = %q\n want = %q", got.ProviderAccountID, pid)
	}
}

// TestIdentityRepository_LookupByEmail_Miss asserts the no-row
// path: looking up an email that is not in identity.user must
// return *domain.IdentityNotFoundError (NOT sql.ErrNoRows), so the
// HTTP handler can errors.As it and map to a 404 envelope.
func TestIdentityRepository_LookupByEmail_Miss(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	_, err := repo.LookupByEmail(context.Background(), "ghost-miss@example.com")
	if err == nil {
		t.Fatalf("expected not-found error; got nil")
	}
	var target *domain.IdentityNotFoundError
	if !errors.As(err, &target) {
		t.Fatalf("expected error chain to carry *domain.IdentityNotFoundError; got %v", err)
	}
	if target.Email != "ghost-miss@example.com" {
		t.Errorf("expected error.Email to be passed through; got %q", target.Email)
	}
}

// TestIdentityRepository_LookupByEmail_CaseInsensitive asserts the
// CITEXT behavior: the email column is case-insensitive (because
// identity_schema R-IS-001 uses CITEXT), so looking up with a
// different case MUST return the same row.
func TestIdentityRepository_LookupByEmail_CaseInsensitive(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	const email = "Lookup-Mixed@example.com"
	userID, cleanup := seedIdentity(t, db, email, "Case Mix", "github", "99999")
	defer cleanup()

	got, err := repo.LookupByEmail(context.Background(), "lookup-mixed@example.com")
	if err != nil {
		t.Fatalf("LookupByEmail lower-case: unexpected error: %v", err)
	}
	if got.ID != userID {
		t.Errorf("case-insensitive lookup returned the wrong row:\n got  = %d\n want = %d", got.ID, userID)
	}
}

// ---------------------------------------------------------------------------
// Upsert (cachicamas-identity-signin-callback slice)
// ---------------------------------------------------------------------------
//
// The TRIANGULATE step: the handler-level tests cover HMAC + schema
// validation against a fake service. These tests exercise the real
// Postgres UPSERT path against the live compose instance. They are
// gated on INTEGRATION=1 like the LookupByEmail tests.
//
// Strict TDD discipline: this block was written BEFORE the Upsert
// method on IdentityRepo existed. Running these tests with no Upsert
// method must fail with "undefined: (*IdentityRepo).Upsert" — that
// failure IS the RED step.

// TestIdentityRepository_Upsert_NewUser covers the first-time GitHub
// sign-in: the email is new, no identity.user row exists, the repo
// INSERTs one and returns the new identity with provider + provider
// account id populated.
func TestIdentityRepository_Upsert_NewUser(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	email := "upsert-new@example.com"
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM identity.user WHERE email = $1`, email)
	})

	got, err := repo.Upsert(context.Background(), domain.IdentityEvent{
		Email:             email,
		Name:              "Upsert New",
		ImageURL:          "https://example.com/new.png",
		Provider:          "github",
		ProviderAccountID: "upsert-new-id-1",
	})
	if err != nil {
		t.Fatalf("Upsert: unexpected error: %v", err)
	}
	if got.Email != email {
		t.Errorf("Email:\n got  = %q\n want = %q", got.Email, email)
	}
	if got.Name != "Upsert New" {
		t.Errorf("Name:\n got  = %q\n want = %q", got.Name, "Upsert New")
	}
	if got.ImageURL != "https://example.com/new.png" {
		t.Errorf("ImageURL:\n got  = %q\n want = %q", got.ImageURL, "https://example.com/new.png")
	}
	if got.Provider != "github" {
		t.Errorf("Provider:\n got  = %q\n want = %q", got.Provider, "github")
	}
	if got.ProviderAccountID != "upsert-new-id-1" {
		t.Errorf("ProviderAccountID:\n got  = %q\n want = %q", got.ProviderAccountID, "upsert-new-id-1")
	}
	if got.ID == 0 {
		t.Errorf("expected non-zero ID; got 0")
	}

	// Verify the row was actually written.
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM identity.user WHERE email = $1`, email,
	).Scan(&n); err != nil {
		t.Fatalf("count identity.user: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 identity.user row; got %d", n)
	}
	if err := db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM identity.account WHERE provider = $1 AND provider_account_id = $2`,
		"github", "upsert-new-id-1",
	).Scan(&n); err != nil {
		t.Fatalf("count identity.account: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 identity.account row; got %d", n)
	}
}

// TestIdentityRepository_Upsert_ExistingUser_ReusesAccount covers the
// returning-GitHub-signin case: an identity.user row exists AND an
// identity.account row exists for the same (provider, account_id).
// The Upsert must be idempotent — second call returns the same user
// id and the account row count stays at 1 (ON CONFLICT DO NOTHING).
func TestIdentityRepository_Upsert_ExistingUser_ReusesAccount(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	email := "upsert-existing@example.com"
	const accountID = "upsert-existing-id-1"
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM identity.user WHERE email = $1`, email)
	})

	ev := domain.IdentityEvent{
		Email:             email,
		Name:              "Upsert Existing",
		ImageURL:          "https://example.com/existing.png",
		Provider:          "github",
		ProviderAccountID: accountID,
	}

	// First call inserts.
	got1, err := repo.Upsert(context.Background(), ev)
	if err != nil {
		t.Fatalf("Upsert first call: unexpected error: %v", err)
	}
	// Second call is a no-op for identity.account (ON CONFLICT), but
	// identity.user is also a no-op because the email already exists.
	got2, err := repo.Upsert(context.Background(), ev)
	if err != nil {
		t.Fatalf("Upsert second call: unexpected error: %v", err)
	}
	if got1.ID != got2.ID {
		t.Errorf("Upsert is not idempotent on identity.user:\n first  = %d\n second = %d", got1.ID, got2.ID)
	}

	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM identity.user WHERE email = $1`, email,
	).Scan(&n); err != nil {
		t.Fatalf("count identity.user: %v", err)
	}
	if n != 1 {
		t.Errorf("expected exactly 1 identity.user row after idempotent re-call; got %d", n)
	}
	if err := db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM identity.account WHERE provider = $1 AND provider_account_id = $2`,
		"github", accountID,
	).Scan(&n); err != nil {
		t.Fatalf("count identity.account: %v", err)
	}
	if n != 1 {
		t.Errorf("expected exactly 1 identity.account row after idempotent re-call; got %d", n)
	}
}

// TestIdentityRepository_Upsert_SameEmailDifferentAccount covers the
// auto-link-on-email-match slice: the email already has an
// identity.user row, but a NEW (provider, account_id) arrives (the
// user signed in with a different GitHub account but kept the email).
// The repo MUST reuse the existing identity.user and INSERT a new
// identity.account row (NOT update the existing account row).
func TestIdentityRepository_Upsert_SameEmailDifferentAccount(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	email := "link-test@example.com"
	const firstAccount = "link-first-id"
	const secondAccount = "link-second-id"
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM identity.user WHERE email = $1`, email)
	})

	// First sign-in: providerAccountId=firstAccount.
	got1, err := repo.Upsert(context.Background(), domain.IdentityEvent{
		Email:             email,
		Name:              "Link Tester",
		ImageURL:          "https://example.com/link.png",
		Provider:          "github",
		ProviderAccountID: firstAccount,
	})
	if err != nil {
		t.Fatalf("Upsert first: unexpected error: %v", err)
	}
	firstUserID := got1.ID

	// Second sign-in: same email, different providerAccountId.
	got2, err := repo.Upsert(context.Background(), domain.IdentityEvent{
		Email:             email,
		Name:              "Link Tester",
		ImageURL:          "https://example.com/link.png",
		Provider:          "github",
		ProviderAccountID: secondAccount,
	})
	if err != nil {
		t.Fatalf("Upsert second: unexpected error: %v", err)
	}

	if got1.ID != got2.ID {
		t.Errorf("auto-link did not reuse identity.user:\n first  = %d\n second = %d", got1.ID, got2.ID)
	}
	if got1.ID != firstUserID {
		t.Errorf("expected user ID to stay at %d; got %d", firstUserID, got1.ID)
	}

	// Two distinct identity.account rows for the same user.
	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM identity.account WHERE user_id = $1 AND provider = 'github'`,
		firstUserID,
	).Scan(&n); err != nil {
		t.Fatalf("count identity.account: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 identity.account rows for the linked user; got %d", n)
	}
}

// TestIdentityRepository_Upsert_PersistsOAuthTokenFields is the
// PR1a (2026-07-06-workspaces) RED step: when the IdentityEvent
// carries the 5 OAuth token fields (AccessToken, RefreshToken,
// ExpiresAt, TokenType, Scope), the repo MUST persist all 5 on the
// identity.account row. Pre-PR1a the struct had no such fields;
// this test pins the new persistence path end-to-end.
//
// The test reads the row back via a direct SELECT (not via the
// repo) so the test does not depend on a not-yet-extended
// LookupByProviderAccountID surface — it is purely about the
// write side, which is what the workspaces feature (PR1c-i)
// depends on for retrieving the access_token from the DB.
func TestIdentityRepository_Upsert_PersistsOAuthTokenFields(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	email := "tokens-new@example.com"
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM identity.user WHERE email = $1`, email)
	})

	at := "gho_pr1a_test"
	rt := "ghr_pr1a_test"
	exp := time.Unix(1234567890, 0).UTC()
	tt := "bearer"
	sc := "repo"

	got, err := repo.Upsert(context.Background(), domain.IdentityEvent{
		Email:             email,
		Name:              "Token Persister",
		ImageURL:          "https://example.com/t.png",
		Provider:          "github",
		ProviderAccountID: "tokens-new-id-1",
		AccessToken:       &at,
		RefreshToken:      &rt,
		ExpiresAt:         &exp,
		TokenType:         &tt,
		Scope:             &sc,
	})
	if err != nil {
		t.Fatalf("Upsert: unexpected error: %v", err)
	}
	if got.ID == 0 {
		t.Fatalf("expected non-zero identity.user id; got 0")
	}

	// Read the row back via direct SQL and assert all 5 token columns
	// match the values we wrote.
	row := db.QueryRowContext(context.Background(),
		`SELECT access_token, refresh_token, expires_at, token_type, scope
               FROM identity.account
              WHERE provider = $1 AND provider_account_id = $2`,
		"github", "tokens-new-id-1",
	)
	var (
		gotAT, gotRT, gotTT, gotSC sql.NullString
		gotEX                      sql.NullTime
	)
	if err := row.Scan(&gotAT, &gotRT, &gotEX, &gotTT, &gotSC); err != nil {
		t.Fatalf("scan identity.account: %v", err)
	}
	if !gotAT.Valid || gotAT.String != at {
		t.Errorf("access_token:\n got  = %v\n want = %q", gotAT, at)
	}
	if !gotRT.Valid || gotRT.String != rt {
		t.Errorf("refresh_token:\n got  = %v\n want = %q", gotRT, rt)
	}
	if !gotEX.Valid {
		t.Errorf("expires_at: expected non-NULL, got NULL")
	} else if !gotEX.Time.Equal(exp) {
		t.Errorf("expires_at:\n got  = %v\n want = %v (UTC)", gotEX.Time, exp)
	}
	if !gotTT.Valid || gotTT.String != tt {
		t.Errorf("token_type:\n got  = %v\n want = %q", gotTT, tt)
	}
	if !gotSC.Valid || gotSC.String != sc {
		t.Errorf("scope:\n got  = %v\n want = %q", gotSC, sc)
	}
}

// TestIdentityRepository_Upsert_AllTokenFieldsNil covers the
// TRIANGULATE case where the IdentityEvent omits all 5 token
// fields. Pre-PR1a code paths (and any future provider that does
// not grant offline access) must still work: every column must
// accept NULL without erroring.
func TestIdentityRepository_Upsert_AllTokenFieldsNil(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	email := "tokens-nil@example.com"
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM identity.user WHERE email = $1`, email)
	})

	got, err := repo.Upsert(context.Background(), domain.IdentityEvent{
		Email:             email,
		Name:              "Nil Token User",
		ImageURL:          "",
		Provider:          "github",
		ProviderAccountID: "tokens-nil-id-1",
		// AccessToken / RefreshToken / ExpiresAt / TokenType / Scope
		// intentionally left nil — must persist as SQL NULL.
	})
	if err != nil {
		t.Fatalf("Upsert with nil tokens: %v", err)
	}
	if got.ID == 0 {
		t.Fatalf("expected non-zero identity.user id; got 0")
	}

	row := db.QueryRowContext(context.Background(),
		`SELECT access_token, refresh_token, expires_at, token_type, scope
               FROM identity.account
              WHERE user_id = $1 AND provider = 'github'`, got.ID,
	)
	var (
		gotAT, gotRT, gotTT, gotSC sql.NullString
		gotEX                      sql.NullTime
	)
	if err := row.Scan(&gotAT, &gotRT, &gotEX, &gotTT, &gotSC); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if gotAT.Valid {
		t.Errorf("access_token: expected NULL, got %q", gotAT.String)
	}
	if gotRT.Valid {
		t.Errorf("refresh_token: expected NULL, got %q", gotRT.String)
	}
	if gotEX.Valid {
		t.Errorf("expires_at: expected NULL, got %v", gotEX.Time)
	}
	if gotTT.Valid {
		t.Errorf("token_type: expected NULL, got %q", gotTT.String)
	}
	if gotSC.Valid {
		t.Errorf("scope: expected NULL, got %q", gotSC.String)
	}
}

// TestIdentityRepository_Upsert_OnlyAccessToken covers another
// TRIANGULATE case: only one of the 5 token fields is populated.
// The repo MUST persist just that one and leave the rest NULL —
// important because GitHub sometimes omits refresh_token when
// access_type != "offline".
func TestIdentityRepository_Upsert_OnlyAccessToken(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	email := "tokens-only-at@example.com"
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM identity.user WHERE email = $1`, email)
	})

	at := "gho_only_at"
	got, err := repo.Upsert(context.Background(), domain.IdentityEvent{
		Email:             email,
		Name:              "AT Only",
		ImageURL:          "",
		Provider:          "github",
		ProviderAccountID: "tokens-only-at-id-1",
		AccessToken:       &at,
		// RefreshToken / ExpiresAt / TokenType / Scope left nil.
	})
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	row := db.QueryRowContext(context.Background(),
		`SELECT access_token, refresh_token, expires_at, token_type, scope
               FROM identity.account
              WHERE user_id = $1`, got.ID,
	)
	var (
		gotAT, gotRT, gotTT, gotSC sql.NullString
		gotEX                      sql.NullTime
	)
	if err := row.Scan(&gotAT, &gotRT, &gotEX, &gotTT, &gotSC); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if !gotAT.Valid || gotAT.String != at {
		t.Errorf("access_token: expected %q, got %v", at, gotAT)
	}
	if gotRT.Valid || gotEX.Valid || gotTT.Valid || gotSC.Valid {
		t.Errorf("expected the other 4 columns NULL; got RT=%v EX=%v TT=%v SC=%v",
			gotRT, gotEX, gotTT, gotSC)
	}
}

// TestIdentityRepository_Upsert_CaseInsensitiveEmail covers the
// CITEXT column on identity.user: an Upsert with a different-case
// email MUST reuse the existing row (case-insensitive match).
func TestIdentityRepository_Upsert_CaseInsensitiveEmail(t *testing.T) {
	db := identityIntegrationDB(t)
	repo := postgres.NewIdentityRepo(db)

	const seededEmail = "Case-Test@Example.com"
	const accountID = "case-test-id"
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `DELETE FROM identity.user WHERE email = $1`, seededEmail)
	})

	// Seed with mixed case via the repo.
	got1, err := repo.Upsert(context.Background(), domain.IdentityEvent{
		Email:             seededEmail,
		Name:              "Case Test",
		ImageURL:          "",
		Provider:          "github",
		ProviderAccountID: accountID,
	})
	if err != nil {
		t.Fatalf("Upsert seeded: unexpected error: %v", err)
	}

	// Upsert again with all-lowercase email.
	got2, err := repo.Upsert(context.Background(), domain.IdentityEvent{
		Email:             "case-test@example.com",
		Name:              "Case Test",
		ImageURL:          "",
		Provider:          "github",
		ProviderAccountID: accountID,
	})
	if err != nil {
		t.Fatalf("Upsert lowercase: unexpected error: %v", err)
	}
	if got1.ID != got2.ID {
		t.Errorf("case-insensitive UPSERT did not reuse identity.user:\n got  = %d\n want = %d", got2.ID, got1.ID)
	}

	var n int
	if err := db.QueryRowContext(context.Background(),
		`SELECT count(*) FROM identity.user WHERE lower(email) = lower($1)`,
		seededEmail,
	).Scan(&n); err != nil {
		t.Fatalf("count identity.user: %v", err)
	}
	if n != 1 {
		t.Errorf("expected exactly 1 identity.user row after case-insensitive re-call; got %d", n)
	}
}
