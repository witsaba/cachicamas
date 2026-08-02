// Package httpiface_test — internal_callback_org_test.go focuses on
// the post-H-1 organization_id requirement in the sync-callback
// payload. The wire contract is:
//   - organization_id is REQUIRED; missing -> 422
//   - organization_id <= 0 -> 422
//   - present and valid -> forwarded to the processor
package httpiface_test

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/domain"
)

// syncTestBody is the test-side mirror of the production callback
// body. The new organization_id field is required.
type syncTestBody struct {
	JobID          int64  `json:"job_id"`
	OrganizationID int64  `json:"organization_id"`
	Status         string `json:"status"`
	CommitSHA      string `json:"commit_sha,omitempty"`
	DefaultBranch  string `json:"default_branch,omitempty"`
	ErrorCode      string `json:"error_code,omitempty"`
	ErrorMessage   string `json:"error_message,omitempty"`
}

func signSyncTestBody(secret, ts string, body []byte) (string, string) {
	var parsed any
	_ = json.Unmarshal(body, &parsed)
	canonical := canonicalizeSyncTestBody(parsed)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts))
	mac.Write([]byte("."))
	mac.Write(canonical)
	return ts, base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

func canonicalizeSyncTestBody(v any) []byte {
	// Mirror the production canonicalizer byte-for-byte (the
	// existing test helper on the same file already does this).
	return canonicalizeForTest(v)
}

func TestSyncCallback_MissingOrganizationID_Returns422(t *testing.T) {
	proc := &fakeProcessor{job: &domain.SyncJob{ID: 7, Status: domain.SyncJobStatusDone}}
	e := newCallbackHandler(t, proc)

	// Body WITHOUT organization_id.
	body := []byte(`{"job_id":7,"status":"done","commit_sha":"abc","default_branch":"main"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsH, sigH := signSyncTestBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for missing organization_id, got %d body=%q", rec.Code, rec.Body.String())
	}
	if proc.jobID != 0 {
		t.Errorf("processor.jobID = %d, want 0 (handler must short-circuit before dispatch)", proc.jobID)
	}
}

func TestSyncCallback_ZeroOrganizationID_Returns422(t *testing.T) {
	proc := &fakeProcessor{job: &domain.SyncJob{ID: 7, Status: domain.SyncJobStatusDone}}
	e := newCallbackHandler(t, proc)

	body := []byte(`{"job_id":7,"organization_id":0,"status":"done","commit_sha":"abc","default_branch":"main"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsH, sigH := signSyncTestBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 for organization_id=0, got %d body=%q", rec.Code, rec.Body.String())
	}
}

func TestSyncCallback_ValidOrganizationID_ForwardsToProcessor(t *testing.T) {
	proc := &fakeProcessor{job: &domain.SyncJob{ID: 7, Status: domain.SyncJobStatusDone}}
	e := newCallbackHandler(t, proc)

	body := []byte(`{"job_id":7,"organization_id":42,"status":"done","commit_sha":"abc","default_branch":"main"}`)
	ts := strconv.FormatInt(time.Now().UnixMilli(), 10)
	tsH, sigH := signSyncTestBody(testSyncCallbackSecret, ts, body)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/sync-callback", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	req.Header.Set("X-Cachicamas-Timestamp", tsH)
	req.Header.Set("X-Cachicamas-Signature", sigH)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204 for valid callback, got %d body=%q", rec.Code, rec.Body.String())
	}
	if proc.orgID != 42 {
		t.Errorf("processor.orgID = %d, want 42", proc.orgID)
	}
	if proc.jobID != 7 {
		t.Errorf("processor.jobID = %d, want 7", proc.jobID)
	}
}
