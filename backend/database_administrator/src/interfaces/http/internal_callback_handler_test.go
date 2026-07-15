// Package httpiface internal_callback_handler_test.go — strict
// TDD coverage for the 2026-07-08-workspace-sync-clone PR-3b
// internal sync-callback handler (HMAC + anti-replay + schema).
package httpiface_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	httpiface "github.com/cachicamas/backend/database_administrator/src/interfaces/http"
)

type fakeProcessor struct {
	mu            sync.Mutex
	jobID         int64
	status        string
	commitSHA     string
	defaultBranch string
	errorCode     string
	errorMessage  string
	job           *domain.SyncJob
	err           error
}

func (f *fakeProcessor) ProcessSyncCallback(_ context.Context, jobID int64, status, commitSHA, defaultBranch, errorCode, errorMessage string) (*domain.SyncJob, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.jobID = jobID
	f.status = status
	f.commitSHA = commitSHA
	f.defaultBranch = defaultBranch
	f.errorCode = errorCode
	f.errorMessage = errorMessage
	return f.job, f.err
}

const testSyncCallbackSecret = "this-is-a-32-byte-test-secret-1234"

// signSyncBody computes the HMAC signature for a request body
// and returns the headers the client must send. The canonical JSON
// is computed by the same algorithm the handler uses (RFC 8785-lite:
// sort keys, no whitespace).
func signSyncBody(secret, ts string, body []byte) (tsHeader, sigHeader string) {
	// Canonicalize.
	var parsed any
	_ = json.Unmarshal(body, &parsed)
	canonical := canonicalizeForTest(parsed)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(canonical)
	sig := mac.Sum(nil)
	return ts, base64.StdEncoding.EncodeToString(sig)
}

func canonicalizeForTest(v any) []byte {
	// Minimal RFC 8785-lite: sort keys, no whitespace, default
	// JSON number encoding via Marshal. The handler's
	// canonicalWriteCallbackValue uses the same rules; the
	// handler test (TestCanonicalCallbackJSON_KnownVector) pins
	// the byte-level contract.
	b, _ := json.Marshal(v) // NOTE: this is not actually canonical; the test below checks the handler's own canonicalize
	_ = b
	// Build canonical manually:
	var buf bytes.Buffer
	canonicalWriteValueForTest(&buf, v)
	return buf.Bytes()
}

func canonicalWriteValueForTest(buf *bytes.Buffer, v any) {
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
		b, _ := json.Marshal(x)
		buf.Write(b)
	case float64:
		// truncate to int if it's a whole number (json.Unmarshal
		// returns float64 for ints too)
		if x == float64(int64(x)) {
			buf.WriteString(strconv.FormatInt(int64(x), 10))
			return
		}
		buf.WriteString(strconv.FormatFloat(x, 'f', -1, 64))
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sortStrings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			kb, _ := json.Marshal(k)
			buf.Write(kb)
			buf.WriteByte(':')
			canonicalWriteValueForTest(buf, x[k])
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			canonicalWriteValueForTest(buf, e)
		}
		buf.WriteByte(']')
	}
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}

func newCallbackHandler(t *testing.T, p *fakeProcessor) *echo.Echo {
	t.Helper()
	e := echo.New()
	logger := slog.Default()
	httpiface.RegisterInternalSyncCallbackRoute(e, p, testSyncCallbackSecret, logger)
	return e
}

func TestSyncCallback_Done_204(t *testing.T) {
	proc := &fakeProcessor{job: &domain.SyncJob{ID: 7, Status: domain.SyncJobStatusDone}}
	e := newCallbackHandler(t, proc)

	body := []byte(`{"job_id":7,"status":"done","commit_sha":"abc123","default_branch":"main"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsH, sigH := signSyncBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if proc.jobID != 7 {
		t.Errorf("processor.jobID = %d, want 7", proc.jobID)
	}
	if proc.commitSHA != "abc123" {
		t.Errorf("processor.commitSHA = %q, want abc123", proc.commitSHA)
	}
}

func TestSyncCallback_Failed_204(t *testing.T) {
	proc := &fakeProcessor{job: &domain.SyncJob{ID: 7, Status: domain.SyncJobStatusFailed}}
	e := newCallbackHandler(t, proc)

	body := []byte(`{"job_id":7,"status":"failed","error_code":"BRANCH_NOT_FOUND","error_message":"main not found"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsH, sigH := signSyncBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204; body=%s", rec.Code, rec.Body.String())
	}
	if proc.errorCode != "BRANCH_NOT_FOUND" {
		t.Errorf("processor.errorCode = %q, want BRANCH_NOT_FOUND", proc.errorCode)
	}
}

func TestSyncCallback_BadSignature_401(t *testing.T) {
	proc := &fakeProcessor{}
	e := newCallbackHandler(t, proc)

	body := []byte(`{"job_id":7,"status":"done","commit_sha":"abc"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	// Sign with WRONG secret.
	_, sigH := signSyncBody("wrong-secret-also-32-bytes-long-foo", ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachicamas-Timestamp", ts)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestSyncCallback_MissingTimestamp_401(t *testing.T) {
	proc := &fakeProcessor{}
	e := newCallbackHandler(t, proc)

	body := []byte(`{"job_id":7,"status":"done","commit_sha":"abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	// No X-Cachicamas-Timestamp
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestSyncCallback_StaleTimestamp_401(t *testing.T) {
	proc := &fakeProcessor{}
	e := newCallbackHandler(t, proc)

	body := []byte(`{"job_id":7,"status":"done","commit_sha":"abc"}`)
	// 10 minutes in the past (well outside the 5-min window)
	ts := strconv.FormatInt(time.Now().Add(-10*time.Minute).UnixMilli(), 10)
	tsH, sigH := signSyncBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestSyncCallback_DoneWithoutCommitSHA_422(t *testing.T) {
	proc := &fakeProcessor{}
	e := newCallbackHandler(t, proc)

	// status=done but missing commit_sha
	body := []byte(`{"job_id":7,"status":"done"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsH, sigH := signSyncBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestSyncCallback_InvalidStatus_422(t *testing.T) {
	proc := &fakeProcessor{}
	e := newCallbackHandler(t, proc)

	body := []byte(`{"job_id":7,"status":"running"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsH, sigH := signSyncBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}

func TestSyncCallback_MalformedJSON_422(t *testing.T) {
	proc := &fakeProcessor{}
	e := newCallbackHandler(t, proc)

	body := []byte(`not json at all`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsH, sigH := signSyncBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rec.Code)
	}
}