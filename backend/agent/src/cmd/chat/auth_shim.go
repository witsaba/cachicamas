// Package main — auth_shim.go is the in-process JWE IdentityResolver
// the chat composition root uses in place of an HTTP round-trip to
// database_administrator (R-07, ADR 0005 § D1 row 3, chat/doc.go:75-81).
//
// The shim duplicates the HKDF-SHA256 + jwe.Decrypt + claims parse
// sequence from
// database_administrator/src/interfaces/http/auth_middleware.go:121-200
// INLINE — the duplication is bounded to JWE envelope only (no Postgres,
// no domain rules, no cross-module import). The barrier is
// non-negotiable: any future change that reaches across the module
// boundary by import, rather than this in-process adapter, must carry
// its own ADR.
//
// # Why Auth.js envelope shape
//
// Auth.js v0.34.3+ produces this exact envelope (verified against
// @auth/core@0.41.2 source):
//
//   alg=dir, enc=A256CBC-HS512
//   HKDF-SHA256 over AUTH_SECRET with
//   salt=cookieName, info="Auth.js Generated Encryption Key
//   (cookieName)", length=64.
//
// The two modules (database_administrator, chat) MUST agree on every
// byte of this envelope or a cookie minted by the Qwik frontend will
// fail to verify against whichever side serves the next request — the
// chat shim's testDeriveKey in auth_shim_test.go is the byte-identical
// fixture that catches drift.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"golang.org/x/crypto/hkdf"

	"github.com/cachicamas/backend/agent/src/chat"
)

// Resolver decrypts Auth.js JWE session cookies and maps them to a
// chat.Identity. In-process — does NOT import database_administrator.
//
// derivedKey is the HKDF-SHA256 derivation of (AUTH_SECRET, cookieName)
// performed once at construction; HKDF is pure, so re-deriving per
// request is wasteful (mirrors auth_middleware.go:156-162).
type Resolver struct {
	derivedKey []byte
	cookieName string
}

// NewResolver derives the HKDF encryption key from authSecret and
// cookieName, and returns a Resolver ready to serve IdentityFromRequest
// for every incoming request.
//
// Panics (fail-fast at startup, mirroring database_administrator's
// S-BAM-080/081):
//   - if authSecret is shorter than 32 raw bytes (S-BAM-081)
//   - if cookieName is empty (the HKDF salt IS cookieName — empty salt
//     would silently admit any cookie into the derivation)
//
// AUTH_SECRET and cookieName MUST match the values the Auth.js encoder
// uses on the Qwik side and the values database_administrator's own
// middleware reads. The three sites (frontend, database_administrator,
// chat) agree by convention; the chat shim's own test fixture is the
// byte-identical evidence that drift has not silently crept in.
func NewResolver(authSecret []byte, cookieName string) *Resolver {
	if len(authSecret) < 32 {
		panic(fmt.Sprintf("cmd/chat: NewResolver: AUTH_SECRET must be at least 32 raw bytes, got %d (S-BAM-081)", len(authSecret)))
	}
	if cookieName == "" {
		panic("cmd/chat: NewResolver: cookieName must be set (it is the HKDF salt)")
	}
	return &Resolver{
		derivedKey: deriveEncryptionKey(authSecret, cookieName),
		cookieName: cookieName,
	}
}

// deriveEncryptionKey applies the HKDF-SHA256 contract from
// @auth/core/src/jwt.ts (verified for 0.34.3+; the same shape applies
// to 0.41.x). salt is the cookie name; info is the literal
// "Auth.js Generated Encryption Key (salt)"; length is 64 — the value
// A256CBC-HS512 demands.
//
// Why this matches the byte-for-byte contract:
//
//	Auth.js calls `hkdf("sha256", keyMaterial, salt, info, length)`
//	where keyMaterial is the raw AUTH_SECRET bytes (NOT base64-decoded),
//	salt is the cookie name, info is the literal above, length is 64
//	for A256CBC-HS512 (or 32 for A256GCM, but Auth.js' default is the
//	former).
//
// This function is the byte-identical twin of
// database_administrator/src/interfaces/http/auth_middleware.go:307-317.
// If the two ever diverge, every JWE minted on one side fails to
// verify on the other.
func deriveEncryptionKey(authSecret []byte, cookieName string) []byte {
	info := []byte("Auth.js Generated Encryption Key (" + cookieName + ")")
	r := hkdf.New(sha256.New, authSecret, []byte(cookieName), info)
	out := make([]byte, 64)
	if _, err := io.ReadFull(r, out); err != nil {
		// hkdf.New with a fixed-length reader can only error on
		// programmer bugs (length=0). Treat as fatal.
		panic(fmt.Sprintf("deriveEncryptionKey: hkdf read: %v", err))
	}
	return out
}

// IdentityFromRequest reads the configured cookie, decrypts the JWE,
// parses the claims, and returns a chat.Identity whose ParticipantID()
// is the claims.sub (preferred) or claims.email (fallback).
//
// Returns (nil, false) on:
//   - missing cookie (the chat middleware writes 401 with kind=server)
//   - JWE decrypt failure (wrong key, tampered ciphertext)
//   - JSON unmarshal failure (malformed payload)
//   - missing both sub and email claims (no participant identifier)
//   - expired session (claims.exp in the past) — the shim is stricter
//     than database_administrator's middleware, which does not validate
//     exp. The chat surface serves a model round-trip; an expired
//     session reaching a model would be a real bug.
func (r *Resolver) IdentityFromRequest(_ context.Context, req *http.Request) (chat.Identity, bool) {
	cookie, err := req.Cookie(r.cookieName)
	if err != nil {
		return nil, false
	}

	jwkKey, err := jwk.FromRaw(r.derivedKey)
	if err != nil {
		// jwk.FromRaw only fails for nil/empty input; we control the
		// derived key so this is a programmer error. Returning (nil,
		// false) is safer than panicking — a single malformed request
		// should not crash the process.
		return nil, false
	}

	plaintext, err := jwe.Decrypt(
		[]byte(cookie.Value),
		jwe.WithKey(jwa.DIRECT, jwkKey),
	)
	if err != nil {
		return nil, false
	}

	var claims authJSCookiePayload
	if err := json.Unmarshal(plaintext, &claims); err != nil {
		return nil, false
	}

	// Expired sessions are rejected even when the envelope is valid.
	// claims.exp is unix seconds; a zero exp means "no expiry set"
	// (Auth.js always sets one for issued sessions, so zero is treated
	// as "never expires" rather than "always expired").
	if claims.Exp != 0 && claims.Exp < time.Now().Unix() {
		return nil, false
	}

	// Map sub (preferred) → email (fallback) to a participant id.
	// sub is the GitHub user ID when minted without an Auth.js database
	// adapter; email is the email claim. Both are stable for the
	// session's lifetime.
	participantID := claims.Sub
	if participantID == "" {
		participantID = claims.Email
	}
	if participantID == "" {
		return nil, false
	}

	return jweIdentity{participantID: participantID}, true
}

// authJSCookiePayload mirrors the canonical Auth.js JWT claims the shim
// reads. Field names are LOCKED — they match @auth/core's Encode<JWT>()
// shape (sub/email/exp/iat).
//
// The same struct appears in auth_shim_test.go so the test fixture and
// the production parser stay byte-identical; the shim test's
// authJSCookiePayload is structurally a subset of this one (the test
// version drops fields it doesn't exercise).
type authJSCookiePayload struct {
	Sub   string `json:"sub,omitempty"`
	Email string `json:"email,omitempty"`
	Exp   int64  `json:"exp,omitempty"`
	Iat   int64  `json:"iat,omitempty"`
}

// jweIdentity is the chat.Identity implementation the shim hands out.
// Unexported because no code outside this package ever constructs it.
type jweIdentity struct {
	participantID string
}

// ParticipantID returns the configured participant id. Never empty when
// produced by Resolver.IdentityFromRequest's ok=true branch — the empty
// case short-circuits to false there.
func (i jweIdentity) ParticipantID() string { return i.participantID }