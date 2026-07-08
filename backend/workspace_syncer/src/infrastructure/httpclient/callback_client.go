// Package httpclient contains the HTTP client that the
// workspace_syncer uses to post outcomes back to
// database_administrator's /api/v1/internal/sync-callback
// endpoint. See design.md §5 (Cross-service contract — callback
// shape).
//
// SECURITY: the bearer token is captured by value at construction
// time. The client is the ONLY caller of this package's Post
// method; the rest of the service never sees the token.
package httpclient

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// DefaultTimeoutSeconds bounds the HTTP round-trip to
// database_administrator. The database_administrator is a
// control-plane service: its responses should be sub-second. 30s
// is a generous ceiling that catches stuck connections without
// holding up the workspace_syncer's goroutine.
const DefaultTimeoutSeconds = 30

// antiReplayToleranceMs is the maximum clock skew (in milliseconds)
// the syncer allows between its timestamp and the db_admin's
// server clock. The db_admin enforces a 5-min window in BOTH
// directions; we use the same value on the sending side so the
// signature is always inside the window.
const antiReplayToleranceMs = 5 * 60 * 1000

// CallbackRequest is the body posted to
// database_administrator's POST /api/v1/internal/sync-callback
// endpoint. See design.md §5 — Cross-service contract.
type CallbackRequest struct {
	JobID          int64  `json:"job_id"`
	WorkspaceID    int64  `json:"workspace_id"`
	Status         string `json:"status"` // "done" | "failed"
	CommitSHAAfter string `json:"commit_sha_after,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
	StartedAt      string `json:"started_at,omitempty"`
	FinishedAt     string `json:"finished_at,omitempty"`
}

// CallbackClient posts CallbackRequest bodies to database_administrator.
// It is the only outbound HTTP client in the workspace_syncer.
//
// PR-2c fix: the callback endpoint uses HMAC-SHA256 signature
// auth (mirrors the identity signin-callback endpoint), not
// bearer auth. The bearer scheme is reserved for the SYNCHRONOUS
// endpoints (clone-and-validate, health). The signature is
// HMAC-SHA256(secret, ts + "." + canonical_json), base64-encoded;
// the timestamp is unix_ms. The db_admin verifies with constant-
// time compare and rejects timestamps outside a 5-min window.
type CallbackClient struct {
	baseURL string
	secret  string
	client  *http.Client
}

// NewCallbackClient constructs a CallbackClient. The baseURL is
// the database_administrator's root (e.g.
// "http://database_administrator:8080"); the client appends
// "/api/v1/internal/sync-callback" itself. The secret is the
// shared SYNC_CALLBACK_SECRET (HMAC-SHA256 key); it MUST match
// the value in db_admin's env (otherwise the db_admin returns
// 401 with reason=bad_signature).
func NewCallbackClient(baseURL, secret string) *CallbackClient {
	return &CallbackClient{
		baseURL: baseURL,
		secret:  secret,
		client: &http.Client{
			Timeout: DefaultTimeoutSeconds * time.Second,
		},
	}
}

// Post posts the callback body to database_administrator. Returns
// nil on a 2xx response; returns a non-nil error on a non-2xx
// response or a network error. The error wraps the HTTP status code
// so the application layer can log it.
//
// The body is encoded as canonical JSON (sorted keys, no
// whitespace) before the signature is computed; the same canonical
// form is what we send on the wire. The receiver re-canonicalizes
// the parsed body and compares HMACs in constant time.
func (c *CallbackClient) Post(ctx context.Context, req CallbackRequest) error {
	canonical, err := canonicalizeJSON(req)
	if err != nil {
		return fmt.Errorf("callback: canonicalize body: %w", err)
	}

	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	sig := computeSignature(c.secret, ts, canonical)

	url := c.baseURL + "/api/v1/internal/sync-callback"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(canonical))
	if err != nil {
		return fmt.Errorf("callback: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Cachicamas-Timestamp", ts)
	httpReq.Header.Set("X-Cachicamas-Signature", sig)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("callback: post: %w", err)
	}
	defer func() {
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("callback: non-2xx response: status=%d", resp.StatusCode)
	}
	return nil
}

// computeSignature returns base64(HMAC-SHA256(secret, ts + "." + body)).
// Mirrors the db_admin's signature computation byte-for-byte.
func computeSignature(secret, ts string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(body)
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// canonicalizeJSON marshals v to canonical JSON (RFC 8785-lite):
//   - object keys sorted lexicographically
//   - no whitespace
//   - strings escaped per encoding/json (RFC 8259)
//   - integers encoded without decimal point
//
// Accepts both maps and arbitrary structs (any type that encoding/
// json can marshal). For structs, the function first marshals to
// JSON, then unmarshals into a map[string]any, then re-emits in
// canonical form. The round-trip is required because the
// canonicalizer works on the generic map type, not typed values.
//
// The db_admin's canonicalWriteCallbackValue follows the same
// rules; the two implementations must produce identical bytes for
// the same input or the HMACs diverge.
func canonicalizeJSON(v any) ([]byte, error) {
	// Round-trip: marshal the typed value to JSON, then unmarshal
	// into a generic map. This is the standard way to feed a
	// typed struct into a generic canonicalizer.
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var generic any
	if err := json.Unmarshal(raw, &generic); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := canonicalWrite(&buf, generic); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

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
		b, err := json.Marshal(x)
		if err != nil {
			return err
		}
		buf.Write(b)
	case float64:
		// json.Unmarshal decodes JSON numbers as float64. Integers
		// (the only number in the closed schema) round-trip without
		// a decimal point; non-integers are rejected (the schema
		// has no floats).
		if x == float64(int64(x)) {
			buf.WriteString(strconv.FormatInt(int64(x), 10))
			return nil
		}
		return fmt.Errorf("canonicalWrite: non-integer numeric value not allowed: %v", x)
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
