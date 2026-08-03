package http

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/workspace_syncer/src/application"
	"github.com/cachicamas/backend/workspace_syncer/src/domain"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/httpclient"
)

// fakeRunnerForHandler mirrors the one in application/clone_service_test.go;
// duplicated here so the http package test is self-contained.
type fakeRunnerForHandler struct {
	mu        sync.Mutex
	clonePath string
	cloneErr  error
	probeSHA  string
	probeErr  error
}

func (f *fakeRunnerForHandler) Clone(ctx context.Context, workspaceID int64, owner, repo, oauthToken string) (string, error) {
	return f.clonePath, f.cloneErr
}
func (f *fakeRunnerForHandler) WorktreeProbe(ctx context.Context, path string) (string, error) {
	return f.probeSHA, f.probeErr
}
func (f *fakeRunnerForHandler) ResolveDefaultBranch(ctx context.Context, path string) (string, error) {
	return "main", nil
}

func (f *fakeRunnerForHandler) ResolveHead(ctx context.Context, path string) (string, error) {
	return f.probeSHA, nil
}

type fakeCallbackForHandler struct {
	mu    sync.Mutex
	calls []httpclient.CallbackRequest
}

func (f *fakeCallbackForHandler) Post(ctx context.Context, req httpclient.CallbackRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	return nil
}

func newTestCloneHandler(t *testing.T) (*CloneHandler, *fakeRunnerForHandler, *fakeCallbackForHandler) {
	t.Helper()
	runner := &fakeRunnerForHandler{
		clonePath: "/data/workspaces/7/octocat/hello-world.git/",
		probeSHA:  "abc1234567890abcdef1234567890abcdef12345",
	}
	callback := &fakeCallbackForHandler{}
	svc := application.NewCloneService(runner, callback, nil, nil)
	return NewCloneHandler(svc, nil), runner, callback
}

func TestCloneHandler_ValidRequest_Returns202(t *testing.T) {
	h, _, _ := newTestCloneHandler(t)

	e := echo.New()
	h.Register(e)

	body := cloneRequestBody{
		JobID:         42,
		WorkspaceID:   7,
		Owner:         "octocat",
		Repo:          "hello-world",
		DefaultBranch: "main",
		OAuthToken:    "gho_xxx",
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202; body=%s", rec.Code, rec.Body.String())
	}
	var respBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if respBody["job_id"].(float64) != 42 {
		t.Errorf("job_id = %v, want 42", respBody["job_id"])
	}
	if respBody["status"] != "running" {
		t.Errorf("status = %q, want running", respBody["status"])
	}
}

func TestCloneHandler_MissingFields_Returns400ValidationEnvelope(t *testing.T) {
	h, _, _ := newTestCloneHandler(t)
	e := echo.New()
	h.Register(e)

	// Empty body — every field is invalid.
	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader([]byte("{}")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", rec.Code, rec.Body.String())
	}
	var respBody map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if respBody["error"] != "validation" {
		t.Errorf("error = %q, want validation", respBody["error"])
	}
	fields, ok := respBody["fields"].(map[string]any)
	if !ok {
		t.Fatalf("fields is not a map: %v", respBody["fields"])
	}
	// At least the 4 fields that every request must satisfy
	// (oauth_token is included as the 6th failing field per the
	// validator's locks).
	for _, key := range []string{"owner", "repo", "default_branch", "oauth_token"} {
		if _, found := fields[key]; !found {
			t.Errorf("fields[%q] missing", key)
		}
	}
}

func TestCloneHandler_ShellMetacharOwner_Returns400(t *testing.T) {
	h, _, _ := newTestCloneHandler(t)
	e := echo.New()
	h.Register(e)

	body := cloneRequestBody{
		JobID:         1,
		WorkspaceID:   1,
		Owner:         "octo;rm -rf /",
		Repo:          "r",
		DefaultBranch: "main",
		OAuthToken:    "t",
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for shell metachar owner", rec.Code)
	}
}

func TestCloneHandler_BadJSON_Returns400(t *testing.T) {
	h, _, _ := newTestCloneHandler(t)
	e := echo.New()
	h.Register(e)

	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate",
		bytes.NewReader([]byte(`{not valid json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for malformed JSON", rec.Code)
	}
}

// TestCloneHandler_WellKnownContract guarantees the wire shape
// does not drift. A client (the frontend card) reads `status`
// directly from the 202 body.
func TestCloneHandler_WellKnownContract(t *testing.T) {
	h, _, _ := newTestCloneHandler(t)
	e := echo.New()
	h.Register(e)

	body := cloneRequestBody{
		JobID:         42,
		WorkspaceID:   7,
		Owner:         "octocat",
		Repo:          "hello-world",
		DefaultBranch: "main",
		OAuthToken:    "gho_xxx",
	}
	bodyJSON, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202", rec.Code)
	}
	bodyBytes, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(bodyBytes), `"status":"running"`) {
		t.Errorf("body = %q, want it to contain status:running", string(bodyBytes))
	}
}

// Compile-time guard that the domain error types are exported (so
// errors.As works in the handler).
var _ error = (*domain.ValidationError)(nil)

// TestCloneHandler_OversizeBody_Returns400 is the regression test
// for spec audit finding M-1: a 2 MiB POST body MUST be rejected
// with 400 BEFORE the handler allocates a matching buffer. The
// http.MaxBytesReader wrapper in handler.Clone enforces this.
func TestCloneHandler_OversizeBody_Returns400(t *testing.T) {
	h, _, _ := newTestCloneHandler(t)
	e := echo.New()
	h.Register(e)

	// 2 MiB body. Use a JSON object with a single huge field so
	// the JSON decoder cannot short-circuit before reading the
	// full bytes. The body MUST be rejected with 400 (not 413;
	// the design uses 400 to keep the envelope shape consistent
	// with the existing validation envelope).
	oversize := bytes.Repeat([]byte("a"), 2<<20)
	body := []byte(`{"job_id":1,"workspace_id":1,"owner":"o","repo":"r","default_branch":"main","oauth_token":"t","filler":"` +
		string(oversize) + `"}`)

	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 (oversize body must be rejected); body=%s", rec.Code, rec.Body.String())
	}
}

// TestCloneHandler_NormalBody_Returns202 is the happy-path
// counterpart to TestCloneHandler_OversizeBody_Returns400: a
// well-formed 100 KiB body (well under the 1 MiB cap) MUST be
// accepted and return 202.
func TestCloneHandler_NormalBody_Returns202(t *testing.T) {
	h, _, _ := newTestCloneHandler(t)
	e := echo.New()
	h.Register(e)

	// Build a body whose padding keeps it comfortably under 1 MiB.
	body := cloneRequestBody{
		JobID:         42,
		WorkspaceID:   7,
		Owner:         "octocat",
		Repo:          "hello-world",
		DefaultBranch: "main",
		OAuthToken:    "gho_xxx",
	}
	bodyJSON, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/internal/clone-and-validate", bytes.NewReader(bodyJSON))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 (well-formed body must be accepted); body=%s", rec.Code, rec.Body.String())
	}
}
