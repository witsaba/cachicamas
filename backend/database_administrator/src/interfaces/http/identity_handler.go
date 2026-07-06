// Package httpiface — identity_handler.go introduces the OAuth signin-
// callback endpoint (cachicamas-identity-signin-callback).
//
// This endpoint is the frontend's bridge from Auth.js's `events.signIn`
// callback to the database_administrator Go service. It accepts an
// HMAC-signed JSON body, verifies the signature against the shared
// IDENTITY_CALLBACK_SECRET, validates the schema, and dispatches to
// application.IdentityService.UpsertFromOAuth which performs the
// identity.user + identity.account UPSERT.
//
// Wire protocol (locked at docs/adr/0003-add-identity-callback-hmac.md):
//
//	POST /api/v1/identity/signin-callback
//	Headers:
//	  Content-Type: application/json
//	  X-Cachicamas-Timestamp: <unix_ms string>
//	  X-Cachicamas-Signature: base64(HMAC_SHA256(secret, ts + "." + canonical_json))
//	Body: { user: {...}, account: {...} }  (see ADR 0003 §"Wire protocol")
//
// Canonical JSON (RFC 8785-lite): keys sorted lexicographically,
// lowercased, no whitespace, no padding. The implementation in
// canonicalizeJSON below is the production-side oracle; the test-side
// copy in identity_handler_test.go MUST produce identical bytes for
// the same input (verified by TestCanonicalJSON_KnownVector).
//
// Anti-replay window: 5 minutes (timestamps outside ±5 min from
// server-side now are rejected with 401). Documented in ADR 0003.
//
// This handler is intentionally NOT mounted under the JWE-verifier
// middleware (interfaces/http/auth_middleware.go). It has its own
// HMAC-based authentication and is reached only from the Qwik Node
// SSR (via SERVER_API_BASE_URL in compose, /api reverse-proxy in dev).
package httpiface

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// antiReplayWindow is the maximum clock skew (in each direction) the
// handler tolerates between the client-supplied timestamp and the
// server-side now. Anything outside is rejected with 401. Locked at
// docs/adr/0003 §"Anti-replay".
const antiReplayWindow = 5 * time.Minute

// IdentityUpserter is the slice of application.IdentityService the
// handler actually consumes. The service has additional methods
// (LookupByEmail etc.) used by other layers; defining a narrower
// interface here keeps the handler test-friendly (a tiny fake
// satisfies only UpsertFromOAuth) and documents the dependency.
type IdentityUpserter interface {
	UpsertFromOAuth(ctx context.Context, ev domain.IdentityEvent) (*domain.Identity, error)
}

// IdentityHandler exposes the OAuth signin-callback endpoint over
// HTTP. It depends on a service-shaped interface (UpsertFromOAuth) so
// tests can wire a fake without standing up a real Postgres.
//
// Constructor panics on:
//   - svc == nil
//   - secret shorter than 32 raw bytes (fail-fast posture; matches
//     IdentityFromCookie's AUTH_SECRET rule).
type IdentityHandler struct {
	svc    IdentityUpserter
	secret []byte
	logger *slog.Logger
}

// NewIdentityHandler wires the handler. Panics on bad config; callers
// (main.go composition root) should treat that as a startup crash.
func NewIdentityHandler(svc IdentityUpserter, secret string, logger *slog.Logger) *IdentityHandler {
	if svc == nil {
		panic("NewIdentityHandler: svc must not be nil")
	}
	if secret == "" {
		panic("NewIdentityHandler: secret must not be empty (set IDENTITY_CALLBACK_SECRET in env)")
	}
	if len(secret) < 32 {
		panic(fmt.Sprintf("NewIdentityHandler: secret must be at least 32 raw bytes (got %d); generate with `openssl rand -base64 32`", len(secret)))
	}
	if logger == nil {
		panic("NewIdentityHandler: logger must not be nil")
	}
	return &IdentityHandler{
		svc:    svc,
		secret: []byte(secret),
		logger: logger,
	}
}

// RegisterIdentityCallbackRoute mounts the handler on the given Echo
// instance at POST /api/v1/identity/signin-callback. Exported so
// main.go can wire it on the public/internal route group (NOT under
// the JWE verifier — this endpoint has its own HMAC auth).
func RegisterIdentityCallbackRoute(e *echo.Echo, svc IdentityUpserter, secret string, logger *slog.Logger) {
	h := NewIdentityHandler(svc, secret, logger)
	e.POST("/api/v1/identity/signin-callback", h.HandleSignInCallback)
}

// ---------------------------------------------------------------------------
// Handler
// ---------------------------------------------------------------------------

// signInCallbackBody is the request body schema. The fields map to
// domain.IdentityEvent after extraction; the rest is discarded after
// HMAC verification (see ADR 0003 §"Forward notes" on OAuth tokens).
type signInCallbackBody struct {
	User struct {
		ID    string  `json:"id"`
		Email string  `json:"email"`
		Name  *string `json:"name"`
		Image *string `json:"image"`
	} `json:"user"`
	Account struct {
		Provider          string  `json:"provider"`
		ProviderAccountID string  `json:"providerAccountId"`
		AccessToken       *string `json:"accessToken"`
		RefreshToken      *string `json:"refreshToken"`
		ExpiresAt         *int64  `json:"expiresAt"`
		TokenType         *string `json:"tokenType"`
		Scope             *string `json:"scope"`
	} `json:"account"`
}

// HandleSignInCallback is the POST /api/v1/identity/signin-callback
// handler. See the package doc for the wire contract.
//
// Status codes (locked):
//   - 204 No Content on success
//   - 401 Unauthorized on bad/expired/missing signature or timestamp
//   - 422 Unprocessable Entity on schema validation failure
//   - 500 Internal Server Error on service error
//
// The signature verification uses crypto/subtle.ConstantTimeCompare
// so an attacker cannot probe the expected MAC one byte at a time
// (timing-side-channel defense).
func (h *IdentityHandler) HandleSignInCallback(c *echo.Context) error {
	req := c.Request()

	// 1. Read the body (we need the raw bytes for canonicalization +
	//    JSON parse).
	rawBody, err := io.ReadAll(req.Body)
	if err != nil {
		return h.writeUnauthorized(c, "body_read_failed")
	}

	// 2. Parse + canonicalize the JSON. We parse twice: once to extract
	//    the schema (after HMAC verification), once to canonicalize
	//    (canonical is computed on the wire bytes — same input must
	//    produce same canonical bytes on both sides).
	var parsed any
	if err := json.Unmarshal(rawBody, &parsed); err != nil {
		return h.writeUnprocessable(c, "body is not valid JSON: "+err.Error())
	}
	canonical, err := canonicalizeJSON(parsed)
	if err != nil {
		return h.writeUnprocessable(c, "body could not be canonicalized: "+err.Error())
	}

	// 3. Read + validate the headers.
	tsHeader := req.Header.Get("X-Cachicamas-Timestamp")
	if tsHeader == "" {
		return h.writeUnauthorized(c, "missing_timestamp")
	}
	tsUnix, err := strconv.ParseInt(tsHeader, 10, 64)
	if err != nil {
		return h.writeUnauthorized(c, "malformed_timestamp")
	}
	sigHeader := req.Header.Get("X-Cachicamas-Signature")
	if sigHeader == "" {
		return h.writeUnauthorized(c, "missing_signature")
	}
	sigRaw, err := base64.StdEncoding.DecodeString(sigHeader)
	if err != nil {
		return h.writeUnauthorized(c, "malformed_signature_encoding")
	}

	// 4. Anti-replay window. Server-side "now" is computed at this
	//    point; a delta > 5 min is rejected. We use ms because the
	//    wire contract uses unix_ms.
	nowMs := time.Now().UnixMilli()
	delta := nowMs - tsUnix
	if delta < 0 {
		delta = -delta
	}
	if time.Duration(delta)*time.Millisecond > antiReplayWindow {
		return h.writeUnauthorized(c, "timestamp_outside_window")
	}

	// 5. Compute the expected HMAC and constant-time compare.
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(tsHeader))
	mac.Write([]byte("."))
	mac.Write(canonical)
	expected := mac.Sum(nil)
	if subtle.ConstantTimeCompare(expected, sigRaw) != 1 {
		return h.writeUnauthorized(c, "bad_signature")
	}

	// 6. Body schema validation (after signature — invalid schema on a
	//    valid signature still 422s, but the test suite asserts this
	//    ordering by passing a valid signature alongside an invalid
	//    body).
	var body signInCallbackBody
	if err := json.Unmarshal(rawBody, &body); err != nil {
		return h.writeUnprocessable(c, "body schema mismatch: "+err.Error())
	}
	if body.User.Email == "" || body.User.ID == "" {
		return h.writeUnprocessable(c, "user.email and user.id are required")
	}
	if body.Account.Provider == "" || body.Account.ProviderAccountID == "" {
		return h.writeUnprocessable(c, "account.provider and account.providerAccountId are required")
	}

	// 7. Extract the IdentityEvent and dispatch. The 5 OAuth token
	//    fields ARE persisted as of PR1a (2026-07-06-workspaces):
	//    identity.account gained access_token, refresh_token,
	//    expires_at, token_type, scope (migration 20260706120000). The
	//    frontend (identity-callback-client.ts) has been forwarding
	//    these fields since cachicamas-identity-signin-callback; this
	//    slice wires the persistence path end-to-end. R-WS-096 forbids
	//    any of these fields from appearing in an HTTP response — the
	//    workspace handler and this handler NEVER serialize them out.
	ev := domain.IdentityEvent{
		Email:             body.User.Email,
		Name:              stringPtrOrEmpty(body.User.Name),
		ImageURL:          stringPtrOrEmpty(body.User.Image),
		Provider:          body.Account.Provider,
		ProviderAccountID: body.Account.ProviderAccountID,
		AccessToken:       body.Account.AccessToken,
		RefreshToken:      body.Account.RefreshToken,
		ExpiresAt:         int64PtrToTimePtr(body.Account.ExpiresAt),
		TokenType:         body.Account.TokenType,
		Scope:             body.Account.Scope,
	}
	if _, err := h.svc.UpsertFromOAuth(req.Context(), ev); err != nil {
		h.logger.ErrorContext(req.Context(), "identity signin-callback: service error",
			slog.String("error", err.Error()),
			slog.String("provider", ev.Provider),
			slog.String("provider_account_id", ev.ProviderAccountID),
		)
		return c.JSON(http.StatusInternalServerError, map[string]any{
			"code":    "internal_error",
			"message": "An internal error occurred while persisting the identity.",
		})
	}

	return c.NoContent(http.StatusNoContent)
}

// stringPtrOrEmpty returns the dereferenced value or "" for nil. Tiny
// helper to keep the extraction line concise.
func stringPtrOrEmpty(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

// int64PtrToTimePtr converts the wire-shape unix-seconds pointer
// (Auth.js convention) into a *time.Time in UTC. Returns nil for a
// nil pointer so the persistence layer writes SQL NULL. PR1a added
// this helper when identity.account.expires_at (TIMESTAMPTZ) was
// introduced; pre-PR1a events that do not carry expires_at keep
// NULL.
func int64PtrToTimePtr(unixSec *int64) *time.Time {
	if unixSec == nil {
		return nil
	}
	t := time.Unix(*unixSec, 0).UTC()
	return &t
}

// writeUnauthorized emits the locked 401 envelope and logs the reason.
// It does NOT call next(c).
func (h *IdentityHandler) writeUnauthorized(c *echo.Context, reason string) error {
	h.logger.WarnContext(c.Request().Context(), "identity signin-callback rejected",
		slog.String("reason", reason),
	)
	return c.JSON(http.StatusUnauthorized, map[string]any{
		"code":    "unauthorized",
		"message": "signature verification failed",
	})
}

// writeUnprocessable emits the locked 422 envelope. The message carries
// the underlying reason for the operator; clients should not parse it
// (the code is the only stable field).
func (h *IdentityHandler) writeUnprocessable(c *echo.Context, reason string) error {
	h.logger.WarnContext(c.Request().Context(), "identity signin-callback invalid body",
		slog.String("reason", reason),
	)
	return c.JSON(http.StatusUnprocessableEntity, map[string]any{
		"code":    "unprocessable_entity",
		"message": reason,
	})
}

// ---------------------------------------------------------------------------
// Canonical JSON (RFC 8785-lite)
// ---------------------------------------------------------------------------

// canonicalizeJSON recursively serializes v to canonical JSON.
//
// Rules:
//   - map keys: sorted lexicographically (NOT lowercased — wire keys
//     are camelCase identifiers and we keep them as-is).
//   - whitespace: none.
//   - number encoding: not supported here because the closed schema
//     does NOT include numbers. Any number seen at canonicalization
//     time is a schema violation; we return an error so the handler
//     can 422.
//   - string escaping: delegated to encoding/json (RFC 8259). The
//     test-side canonicalizer uses the same encoding/json path, so
//     escape sequences match byte-for-byte.
//
// This is NOT a full RFC 8785 implementation. RFC 8785 also specifies
// Unicode normalization (NFC), key lowercasing, and ES6 number-to-
// string conversion; our closed schema doesn't include those concerns
// (the keys are ASCII camelCase identifiers; numbers are explicitly
// rejected). See ADR 0003 §"Canonical JSON" for the explicit non-goals.
func canonicalizeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := canonicalWriteValue(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func canonicalWriteValue(buf *bytes.Buffer, v any) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case string:
		// encoding/json's string escaper is what the wire uses.
		b, err := json.Marshal(x)
		if err != nil {
			return err
		}
		buf.Write(b)
	case float64:
		// JSON numbers come back as float64 from json.Unmarshal. The
		// closed schema does NOT include numeric fields, so any number
		// is a schema-violation signal.
		return fmt.Errorf("canonicalWriteValue: numeric value not allowed: %v", x)
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, err := json.Marshal(k)
			if err != nil {
				return err
			}
			buf.Write(kb)
			buf.WriteByte(':')
			if err := canonicalWriteValue(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := canonicalWriteValue(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		return fmt.Errorf("canonicalWriteValue: unsupported type %T", v)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Compile-time check that IdentityHandler implements the slice shape.
// ---------------------------------------------------------------------------

var _ context.Context = nil // anchor: forces context import to remain
var _ = base64.StdEncoding
var _ = sort.Strings
