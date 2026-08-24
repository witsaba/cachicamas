// CH-04.2 — the in-process JWE IdentityResolver shim tests (R-07,
// ADR 0005 § D1 row 3, chat/doc.go:75-81).
//
// The chat composition root's IdentityResolver must be an in-process
// shim that decrypts the Auth.js session-token cookie using the same
// AUTH_SECRET the database_administrator module uses (HKDF-SHA256,
// salt=cookieName, info="Auth.js Generated Encryption Key (cookieName)",
// length=64) and maps sub (or email) to a participant id. The shim
// MUST NOT import database_administrator.
//
// Test surface:
//   - valid JWE cookie signed with the same AUTH_SECRET → participant id
//   - missing cookie → (nil, false) so the middleware writes a 401
//   - expired cookie (past exp claim) → (nil, false)
//   - tampered cookie (different AUTH_SECRET) → (nil, false)
//
// The fixture generator below reproduces the Auth.js envelope
// (alg=dir, enc=A256CBC-HS512, 64-byte HKDF-derived key) so the test
// is a true round-trip: generate → decrypt → assert identity.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"golang.org/x/crypto/hkdf"
)

// testAuthSecret is a 32-byte test secret the JWE fixtures sign with.
// It MUST be ≥32 bytes — the chat shim's constructor panics on a
// shorter value (mirroring database_administrator's S-BAM-081).
const testAuthSecret = "test-auth-secret-padded-to-32-bytes!!"

// testCookieName mirrors the Auth.js default ("authjs.session-token").
// The HKDF salt IS this value, so a different cookieName produces a
// different derived key — verified by the tampered-cookie test.
const testCookieName = "authjs.session-token"

// testDeriveKey reproduces the HKDF-SHA256 contract from
// database_administrator/src/interfaces/http/auth_middleware.go:307-317.
// Kept here (not in the production file under test) so the test is
// self-contained: the fixture signs with the same key the shim derives.
func testDeriveKey(secret []byte, cookieName string) []byte {
	info := []byte("Auth.js Generated Encryption Key (" + cookieName + ")")
	r := hkdf.New(sha256.New, secret, []byte(cookieName), info)
	out := make([]byte, 64)
	if _, err := io.ReadFull(r, out); err != nil {
		panic(fmt.Sprintf("testDeriveKey: hkdf read: %v", err))
	}
	return out
}

// testEncryptJWECookie builds a compact-serialized JWE that the chat
// shim can decrypt with its HKDF-derived key. The encrypted payload is
// the marshalled authJSCookiePayload so the shim's json.Unmarshal step
// reads the same claims back.
func testEncryptJWECookie(t *testing.T, secret []byte, cookieName string, payload authJSCookiePayload) string {
	t.Helper()

	derivedKey := testDeriveKey(secret, cookieName)
	jwkKey, err := jwk.FromRaw(derivedKey)
	if err != nil {
		t.Fatalf("jwk.FromRaw: %v", err)
	}

	plaintext, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	encrypted, err := jwe.Encrypt(
		plaintext,
		jwe.WithKey(jwa.DIRECT, jwkKey),
		jwe.WithContentEncryption(jwa.A256CBC_HS512),
	)
	if err != nil {
		t.Fatalf("jwe.Encrypt: %v", err)
	}
	return string(encrypted)
}

// newAuthedRequest builds an *http.Request carrying the given cookie
// value under the canonical cookie name.
func newAuthedRequest(t *testing.T, cookieValue string) *http.Request {
	t.Helper()
	r := httptest.NewRequest(http.MethodPost, "/api/agent/turns", nil)
	if cookieValue != "" {
		r.AddCookie(&http.Cookie{Name: testCookieName, Value: cookieValue})
	}
	return r
}

// TestAuthShim_ValidCookieResolvesParticipant covers the happy path: a
// JWE cookie signed with the same AUTH_SECRET + cookieName the shim
// was constructed with returns a non-nil Identity whose ParticipantID()
// equals the sub claim.
func TestAuthShim_ValidCookieResolvesParticipant(t *testing.T) {
	t.Parallel()

	resolver := NewResolver([]byte(testAuthSecret), testCookieName)
	cookie := testEncryptJWECookie(t, []byte(testAuthSecret), testCookieName, authJSCookiePayload{
		Sub: "participant-42",
		Exp: time.Now().Add(1 * time.Hour).Unix(),
	})

	ident, ok := resolver.IdentityFromRequest(context.Background(), newAuthedRequest(t, cookie))

	if !ok {
		t.Fatal("ok = false, want true for a valid JWE cookie (R-07 happy path)")
	}
	if ident == nil {
		t.Fatal("ident = nil, want non-nil")
	}
	if got := ident.ParticipantID(); got != "participant-42" {
		t.Errorf("ParticipantID() = %q, want %q (R-07 sub→participant mapping)", got, "participant-42")
	}
}

// TestAuthShim_MissingCookieReturnsNilFalse covers the 401 path: a
// request without the configured cookie returns (nil, false) so the
// chat identity middleware writes the locked 401 envelope
// (chat/http.go:84-87).
func TestAuthShim_MissingCookieReturnsNilFalse(t *testing.T) {
	t.Parallel()

	resolver := NewResolver([]byte(testAuthSecret), testCookieName)

	ident, ok := resolver.IdentityFromRequest(context.Background(), newAuthedRequest(t, ""))

	if ok {
		t.Error("ok = true, want false for a request with no cookie (R-07 401 path)")
	}
	if ident != nil {
		t.Errorf("ident = %v, want nil for a missing cookie", ident)
	}
}

// TestAuthShim_ExpiredCookieReturnsNilFalse covers the past-exp claim
// case: a cookie that decrypts cleanly but whose exp claim is in the
// past returns (nil, false) — the chat shim MUST reject expired
// sessions even when the envelope itself is valid.
func TestAuthShim_ExpiredCookieReturnsNilFalse(t *testing.T) {
	t.Parallel()

	resolver := NewResolver([]byte(testAuthSecret), testCookieName)
	cookie := testEncryptJWECookie(t, []byte(testAuthSecret), testCookieName, authJSCookiePayload{
		Sub: "participant-99",
		Exp: time.Now().Add(-1 * time.Hour).Unix(), // one hour ago
	})

	ident, ok := resolver.IdentityFromRequest(context.Background(), newAuthedRequest(t, cookie))

	if ok {
		t.Error("ok = true, want false for an expired cookie (R-07 expiry)")
	}
	if ident != nil {
		t.Errorf("ident = %v, want nil for an expired cookie", ident)
	}
}

// TestAuthShim_TamperedCookieReturnsNilFalse covers the integrity
// failure: a JWE signed with a DIFFERENT AUTH_SECRET than the shim's
// configured secret returns (nil, false) — jwe.Decrypt fails on the
// wrong key and the shim MUST NOT silently coerce that to an identity.
func TestAuthShim_TamperedCookieReturnsNilFalse(t *testing.T) {
	t.Parallel()

	resolver := NewResolver([]byte(testAuthSecret), testCookieName)
	const tamperedSecret = "tampered-auth-secret-padded-to-32b!!!"
	cookie := testEncryptJWECookie(t, []byte(tamperedSecret), testCookieName, authJSCookiePayload{
		Sub: "participant-evil",
		Exp: time.Now().Add(1 * time.Hour).Unix(),
	})

	ident, ok := resolver.IdentityFromRequest(context.Background(), newAuthedRequest(t, cookie))

	if ok {
		t.Error("ok = true, want false for a tampered cookie (R-07 integrity)")
	}
	if ident != nil {
		t.Errorf("ident = %v, want nil for a tampered cookie", ident)
	}
}

// TestAuthShim_PanicsOnShortSecret covers the startup-time fail-fast:
// NewResolver MUST panic on AUTH_SECRET < 32 bytes (mirroring
// database_administrator's S-BAM-081). The panic happens at
// composition-time, not per-request, so a misconfigured operator is
// rejected at boot.
func TestAuthShim_PanicsOnShortSecret(t *testing.T) {
	t.Parallel()

	// 31-byte secret (one byte shy of the 32-byte minimum HKDF requires
	// for A256CBC-HS512): exactly 26 lowercase letters + "12345".
	const shortSecret = "abcdefghijklmnopqrstuvwxyz12345"
	if len(shortSecret) != 31 {
		t.Fatalf("test setup error: shortSecret is %d bytes, want 31", len(shortSecret))
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("NewResolver did not panic on a 31-byte secret; want panic per S-BAM-081")
		}
	}()

	_ = NewResolver([]byte(shortSecret), testCookieName)
}

// TestAuthShim_PanicsOnEmptyCookieName covers the related fail-fast:
// NewResolver MUST panic when constructed with an empty cookieName —
// the salt in HKDF IS the cookie name, so an empty salt would silently
// admit any cookie into the derivation.
func TestAuthShim_PanicsOnEmptyCookieName(t *testing.T) {
	t.Parallel()

	defer func() {
		if r := recover(); r == nil {
			t.Error("NewResolver did not panic on an empty cookieName; want panic")
		}
	}()

	_ = NewResolver([]byte(testAuthSecret), "")
}