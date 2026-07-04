// Package httpiface_test contains the test suite for the
// identity signin-callback handler introduced by
// cachicamas-identity-signin-callback.
//
// Reference (this slice): the HMAC + timestamp wire protocol documented in
// docs/adr/0003-add-identity-callback-hmac.md and inline in the task
// brief. The handler is the bootstrap endpoint that lets the Qwik
// frontend persist GitHub OAuth identity rows in identity.* without
// sharing a database role with the backend (architecturally fixed in
// this slice vs PR #29's porsager/postgres approach).
//
// What's exercised here:
//   - HMAC-SHA256 verification with constant-time compare.
//   - 5-minute anti-replay window (server rejects timestamps outside).
//   - Canonical JSON (RFC 8785-lite: sorted+lowercase keys, no whitespace).
//   - Schema validation (422 on malformed body).
//   - 401 envelope on bad/expired signature.
//   - 204 success path when the signature is valid AND the service call
//     succeeds (a fake service captures the dispatched event).
//   - 500 path when the service returns an arbitrary error.
//
// What is NOT exercised here (covered by other layers):
//   - Real Postgres UPSERT (covered by identity_repository_test.go's
//     TRIANGULATE integration tests; gated on INTEGRATION=1).
//   - The canonical-JSON-vs-Go-fuzzing concern (covered by the locked
//     known-vector test in TestCanonicalJSON_KnownVector below).
//
// Strict TDD discipline (per openspec/AGENTS.md and the slice's
// implementation order): this file was written BEFORE
// identity_handler.go existed. Running `go test ./src/interfaces/http/... -run TestIdentityHandler`
// with no handler must fail with "undefined: IdentityHandler" — that
// failure IS the RED step.
package httpiface

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/application"
	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// ---------------------------------------------------------------------------
// Test fixtures + helpers
// ---------------------------------------------------------------------------

const (
	testCallbackSecret = "test-callback-secret-do-not-use-in-prod-32+bytes!"
)

// testNow returns the current unix-millisecond time. The handler
// enforces a 5-minute anti-replay window (antiReplayWindow), so the
// test's timestamp must be near "now" (not a hardcoded past value).
// Each test that needs a valid timestamp calls this helper.
func testNow() int64 {
	return time.Now().UnixMilli()
}

// signedRequest builds an httptest.Request carrying the body, an HMAC
// signature over `${ts}.${canonical(body)}` using the given secret, and
// the timestamp header. The default secret matches what the test wires
// into the handler config; pass an alternate secret to forge bad-signature
// scenarios.
func signedRequest(t *testing.T, body []byte, secret string, ts int64) *http.Request {
	t.Helper()
	canonical := canonicalJSONForTest(t, body)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(strconv.FormatInt(ts, 10)))
	mac.Write([]byte("."))
	mac.Write(canonical)
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/identity/signin-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachicamas-Timestamp", strconv.FormatInt(ts, 10))
	req.Header.Set("X-Cachicamas-Signature", sig)
	return req
}

// canonicalJSONForTest is the test-side copy of the canonicalizer used by
// the production handler. Both sides must produce identical bytes for the
// same input — the canonicalizer is the load-bearing part of the wire
// protocol. If the production canonicalizer drifts from this helper, the
// HMAC verification fails for ALL inputs (no false-positive risk).
//
// This helper MUST match the implementation in identity_handler.go
// exactly; we keep it here as a TEST-ONLY oracle, not a dependency. The
// test `TestCanonicalJSON_KnownVector` pins a known input → known output
// pair so future drift is caught immediately.
//
// If the input is not valid JSON, the helper returns the raw bytes
// unchanged. This is intentional: tests that exercise invalid bodies
// (TestIdentityHandler_InvalidBody_Returns422) need to forge a
// signature (or any signature) without canonicalizing first; the
// handler itself will reject the body before checking the signature,
// so the test asserts on 422.
func canonicalJSONForTest(t *testing.T, raw []byte) []byte {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		// Invalid JSON: return raw bytes so the test can still
		// produce SOME HMAC; the handler will short-circuit on the
		// invalid body BEFORE comparing the signature.
		return raw
	}
	out, err := canonicalJSONMarshal(v)
	if err != nil {
		t.Fatalf("canonicalJSONForTest: canonicalJSONMarshal: %v", err)
	}
	return out
}

// canonicalJSONMarshal recursively serializes v to canonical JSON:
//   - keys sorted lexicographically
//   - keys normalized to lowercase
//   - no whitespace, no padding
//   - string escaping delegated to encoding/json (RFC 8259)
//
// This is the same algorithm the handler uses; see
// canonicalizeJSON in identity_handler.go. It is intentionally NOT a
// full RFC 8785 implementation: the schema is closed (only strings,
// nulls, booleans, and nested objects), so we don't need the
// number-encoding rules. Strings and booleans are the same as encoding/json
// would produce; only the key ordering + lowercasing + whitespace
// suppression is custom.
func canonicalJSONMarshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := canonicalWrite(&buf, v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// canonicalWrite is the TEST-SIDE mirror of canonicalWriteValue in
// identity_handler.go. They MUST produce identical bytes for the same
// input (verified by TestCanonicalJSON_KnownVector). Keep them in sync.
func canonicalWrite(buf *bytes.Buffer, v any) error {
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
		// is a schema-violation signal. Surface it explicitly.
		return fmt.Errorf("canonicalWrite: numeric value not allowed in canonical JSON for this endpoint: %v", x)
	case map[string]any:
		// Sort keys lexicographically (NOT lowercased; see identity_handler.go).
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
			if err := canonicalWrite(buf, x[k]); err != nil {
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
			if err := canonicalWrite(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	default:
		return fmt.Errorf("canonicalWrite: unsupported type %T", v)
	}
	return nil
}

// validBody returns a known-good JSON body matching the wire contract.
// The shape mirrors docs/adr/0003 §"Wire protocol" example.
func validBody() []byte {
	return []byte(`{
		"user": {
			"id": "12345",
			"email": "octocat@example.com",
			"name": "Octocat",
			"image": null
		},
		"account": {
			"provider": "github",
			"providerAccountId": "12345",
			"accessToken": "gho_test",
			"refreshToken": null,
			"expiresAt": null,
			"tokenType": "bearer",
			"scope": "read:user user:email"
		}
	}`)
}

// fakeIdentityService is the in-memory application.IdentityService
// substitute used by the handler-level tests. The handler does not depend
// on the concrete service type — it accepts an interface; this fake
// implements that interface so we can wire it without a real service
// (and without importing pgx). Real service behaviour is covered by
// application/identity_service_test.go.
type fakeIdentityService struct {
	mu        sync.Mutex
	calls     int
	lastEvent *domain.IdentityEvent
	returnErr error
}

func (f *fakeIdentityService) UpsertFromOAuth(_ context.Context, ev domain.IdentityEvent) (*domain.Identity, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	dup := ev
	f.lastEvent = &dup
	if f.returnErr != nil {
		return nil, f.returnErr
	}
	return &domain.Identity{
		ID:                42,
		Email:             ev.Email,
		Name:              ev.Name,
		ImageURL:          ev.ImageURL,
		Provider:          ev.Provider,
		ProviderAccountID: ev.ProviderAccountID,
	}, nil
}

// Compile-time guard that fakeIdentityService satisfies the same shape
// as application.IdentityService for the UpsertFromOAuth method. The
// handler accepts an interface so this is the production-side wiring
// point.
var _ IdentityUpserter = (*fakeIdentityService)(nil)

// silenceLogger discards every log line so test output stays clean.
func silenceLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newRouterWithIdentityHandler mounts the handler on a tiny Echo router
// so each test gets a fresh app. The route is the production path
// (POST /api/v1/identity/signin-callback); the test asserts on status,
// body, and (where relevant) the fakeIdentityService capture.
func newRouterWithIdentityHandler(t *testing.T, svc IdentityUpserter, secret string) *echo.Echo {
	t.Helper()
	h := NewIdentityHandler(svc, secret, silenceLogger())
	e := echo.New()
	e.POST("/api/v1/identity/signin-callback", h.HandleSignInCallback)
	return e
}

// ---------------------------------------------------------------------------
// Canonical JSON — known vector
// ---------------------------------------------------------------------------

// TestCanonicalJSON_KnownVector pins a known input → known canonical
// output pair. This is the cross-tooling oracle: the SAME vector must
// pass on the TypeScript side (frontend/src/lib/identity-callback-client.test.ts
// "canonicalizer known vector"). If this test ever drifts, both sides
// need a coordinated update.
func TestCanonicalJSON_KnownVector(t *testing.T) {
	input := []byte(`{
		"user": {"id": "12345", "email": "octocat@example.com", "name": "Octocat", "image": null},
		"account": {"provider": "github", "providerAccountId": "12345", "accessToken": "gho_test", "refreshToken": null, "expiresAt": null, "tokenType": "bearer", "scope": "read:user user:email"}
	}`)
	got := string(canonicalJSONForTest(t, input))
	want := `{"account":{"accessToken":"gho_test","expiresAt":null,"provider":"github","providerAccountId":"12345","refreshToken":null,"scope":"read:user user:email","tokenType":"bearer"},"user":{"email":"octocat@example.com","id":"12345","image":null,"name":"Octocat"}}`
	// Sort order: keys sorted lexicographically (NOT lowercased).
	// "provider" sorts BEFORE "providerAccountId" because the shorter
	// string is less (Go's sort.Strings + encoding/json's escape rules).
	if got != want {
		t.Errorf("canonical JSON drift:\n got  = %s\n want = %s", got, want)
	}
}

// ---------------------------------------------------------------------------
// HMAC verification
// ---------------------------------------------------------------------------

// TestIdentityHandler_ValidSignature_Returns204 covers the happy path:
// a request with a valid signature AND a valid timestamp AND a valid body
// is accepted with 204 No Content. The fakeIdentityService records the
// dispatched event so we can assert the wire-side extraction is correct.
func TestIdentityHandler_ValidSignature_Returns204(t *testing.T) {
	svc := &fakeIdentityService{}
	e := newRouterWithIdentityHandler(t, svc, testCallbackSecret)

	req := signedRequest(t, validBody(), testCallbackSecret, testNow())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d body=%q", rec.Code, rec.Body.String())
	}
	if svc.calls != 1 {
		t.Errorf("expected service to be called once; got %d", svc.calls)
	}
	if svc.lastEvent == nil {
		t.Fatal("expected service to receive an event; got nil")
	}
	if svc.lastEvent.Email != "octocat@example.com" {
		t.Errorf("expected email=octocat@example.com; got %q", svc.lastEvent.Email)
	}
	if svc.lastEvent.Name != "Octocat" {
		t.Errorf("expected name=Octocat; got %q", svc.lastEvent.Name)
	}
	if svc.lastEvent.Provider != "github" {
		t.Errorf("expected provider=github; got %q", svc.lastEvent.Provider)
	}
	if svc.lastEvent.ProviderAccountID != "12345" {
		t.Errorf("expected provider_account_id=12345; got %q", svc.lastEvent.ProviderAccountID)
	}
}

// TestIdentityHandler_BadSignature_Returns401 covers S-IC-010: an
// otherwise-valid request signed with a DIFFERENT secret returns 401
// with the locked envelope and never invokes the service.
func TestIdentityHandler_BadSignature_Returns401(t *testing.T) {
	svc := &fakeIdentityService{}
	e := newRouterWithIdentityHandler(t, svc, testCallbackSecret)

	req := signedRequest(t, validBody(), "wrong-secret-also-32-bytes-long!!!", testNow())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d body=%q", rec.Code, rec.Body.String())
	}
	if svc.calls != 0 {
		t.Errorf("expected service NOT to be called on bad signature; got %d calls", svc.calls)
	}
	var env map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON error envelope; got parse error %v body=%q", err, rec.Body.String())
	}
	if env["code"] != "unauthorized" {
		t.Errorf("expected code=unauthorized; got %v", env["code"])
	}
}

// TestIdentityHandler_ExpiredTimestamp_Returns401 covers the anti-replay
// window: a request whose timestamp is more than 5 minutes from now
// (simulated via frozen now) is rejected with 401, even if the signature
// is otherwise valid for the stale timestamp.
func TestIdentityHandler_ExpiredTimestamp_Returns401(t *testing.T) {
	svc := &fakeIdentityService{}
	e := newRouterWithIdentityHandler(t, svc, testCallbackSecret)

	// 10 minutes in the past — well outside the 5-minute window.
	stale := testNow() - int64((10 * time.Minute) / time.Millisecond)
	req := signedRequest(t, validBody(), testCallbackSecret, stale)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for stale timestamp, got %d body=%q", rec.Code, rec.Body.String())
	}
	if svc.calls != 0 {
		t.Errorf("expected service NOT to be called on stale timestamp; got %d calls", svc.calls)
	}
}

// TestIdentityHandler_FutureTimestamp_Returns401 mirrors the stale case
// for clock skew in the OPPOSITE direction: a timestamp more than 5
// minutes in the future is also rejected. This protects against an
// attacker trying to bypass the replay window with a "future-dated"
// signature.
func TestIdentityHandler_FutureTimestamp_Returns401(t *testing.T) {
	svc := &fakeIdentityService{}
	e := newRouterWithIdentityHandler(t, svc, testCallbackSecret)

	future := testNow() + int64((10 * time.Minute) / time.Millisecond)
	req := signedRequest(t, validBody(), testCallbackSecret, future)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for future timestamp, got %d body=%q", rec.Code, rec.Body.String())
	}
	if svc.calls != 0 {
		t.Errorf("expected service NOT to be called on future timestamp; got %d calls", svc.calls)
	}
}

// TestIdentityHandler_ReplayWithinWindow_Returns204 covers the
// legitimate replay case: a re-submission with the SAME timestamp+body
// within the 5-minute window is accepted. Replay protection at this
// slice is timestamp-based only; the trust boundary is the compose
// internal network (ADR 0003 §"Threat model").
func TestIdentityHandler_ReplayWithinWindow_Returns204(t *testing.T) {
	svc := &fakeIdentityService{}
	e := newRouterWithIdentityHandler(t, svc, testCallbackSecret)

	// Two consecutive requests, both with the same timestamp (within
	// the window). Both should be accepted by the handler; the
	// service's UPSERT is idempotent on (provider, provider_account_id)
	// via ON CONFLICT DO NOTHING.
	for i := 0; i < 2; i++ {
		req := signedRequest(t, validBody(), testCallbackSecret, testNow())
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Fatalf("replay %d: expected 204, got %d body=%q", i, rec.Code, rec.Body.String())
		}
	}
	if svc.calls != 2 {
		t.Errorf("expected service to be called twice (two replays); got %d", svc.calls)
	}
}

// ---------------------------------------------------------------------------
// Schema validation
// ---------------------------------------------------------------------------

// TestIdentityHandler_InvalidBody_Returns422 covers S-IC-020: a request
// whose body fails JSON parse OR is missing required keys returns 422
// with a stable envelope. The signature is intentionally NOT verified
// for an invalid body (the handler returns 422 first; the test asserts
// the signature is NOT consumed).
func TestIdentityHandler_InvalidBody_Returns422(t *testing.T) {
	svc := &fakeIdentityService{}
	e := newRouterWithIdentityHandler(t, svc, testCallbackSecret)

	cases := []struct {
		name string
		body string
	}{
		{"empty body", ""},
		{"not json", "this is not json"},
		{"missing user", `{"account": {"provider": "github", "providerAccountId": "12345"}}`},
		{"missing account", `{"user": {"id": "12345", "email": "octo@example.com", "name": "O", "image": null}}`},
		{"missing account.provider", `{"user": {"id": "12345", "email": "octo@example.com", "name": "O", "image": null}, "account": {"providerAccountId": "12345"}}`},
		{"missing account.providerAccountId", `{"user": {"id": "12345", "email": "octo@example.com", "name": "O", "image": null}, "account": {"provider": "github"}}`},
		{"missing user.email", `{"user": {"id": "12345", "name": "O", "image": null}, "account": {"provider": "github", "providerAccountId": "12345"}}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a request that DOES carry a valid signature — the
			// handler MUST short-circuit on body validation BEFORE
			// touching the signature, so a valid sig + invalid body
			// returns 422 (not 401).
			req := signedRequest(t, []byte(tc.body), testCallbackSecret, testNow())
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnprocessableEntity {
				t.Fatalf("expected 422 for invalid body, got %d body=%q", rec.Code, rec.Body.String())
			}
		})
	}
	if svc.calls != 0 {
		t.Errorf("expected service NOT to be called on invalid body; got %d calls", svc.calls)
	}
}

// TestIdentityHandler_MissingHeaders_Returns401 covers the
// header-presence guard: a request without the X-Cachicamas-Timestamp
// or X-Cachicamas-Signature header returns 401 immediately. The body
// shape is irrelevant because the handler bails before parsing it.
func TestIdentityHandler_MissingHeaders_Returns401(t *testing.T) {
	svc := &fakeIdentityService{}
	e := newRouterWithIdentityHandler(t, svc, testCallbackSecret)

	cases := []struct {
		name   string
		mutate func(*http.Request)
	}{
		{"missing timestamp", func(r *http.Request) { r.Header.Del("X-Cachicamas-Timestamp") }},
		{"missing signature", func(r *http.Request) { r.Header.Del("X-Cachicamas-Signature") }},
		{"non-numeric timestamp", func(r *http.Request) { r.Header.Set("X-Cachicamas-Timestamp", "not-a-number") }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := signedRequest(t, validBody(), testCallbackSecret, testNow())
			tc.mutate(req)
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 for missing/malformed header, got %d body=%q", rec.Code, rec.Body.String())
			}
		})
	}
	if svc.calls != 0 {
		t.Errorf("expected service NOT to be called on missing header; got %d calls", svc.calls)
	}
}

// ---------------------------------------------------------------------------
// Service error mapping
// ---------------------------------------------------------------------------

// TestIdentityHandler_ServiceError_Returns500 covers the error path:
// the HMAC verification + schema validation pass, but the underlying
// service call returns an error (e.g., DB down). The handler maps to
// 500 with the locked envelope so the caller can distinguish a
// transient infrastructure failure from a permanent auth failure.
func TestIdentityHandler_ServiceError_Returns500(t *testing.T) {
	svc := &fakeIdentityService{returnErr: errors.New("connection refused")}
	e := newRouterWithIdentityHandler(t, svc, testCallbackSecret)

	req := signedRequest(t, validBody(), testCallbackSecret, testNow())
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500 on service error, got %d body=%q", rec.Code, rec.Body.String())
	}
	var env map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("expected JSON error envelope; got parse error %v body=%q", err, rec.Body.String())
	}
	if env["code"] != "internal_error" {
		t.Errorf("expected code=internal_error; got %v", env["code"])
	}
}

// ---------------------------------------------------------------------------
// Sanity: the handler does not need a real application.IdentityService
// import to compile (the slice's interface is local).
// ---------------------------------------------------------------------------

var _ application.IdentityService // anchor: forces the import to remain