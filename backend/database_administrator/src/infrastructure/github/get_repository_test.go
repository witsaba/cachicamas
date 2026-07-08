// Package github get_repository_test.go — strict TDD coverage for
// the 2026-07-08-workspace-sync-clone PR-3b GetRepository method.
// Uses httptest.Server; the handler serves /repos/{owner}/{name}
// with the recorded status + body.
package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gh "github.com/cachicamas/backend/database_administrator/src/infrastructure/github"
)

const sampleRepoJSON = `{
  "id": 123,
  "name": "hello",
  "full_name": "octocat/hello",
  "private": false,
  "visibility": "public",
  "size": 42,
  "default_branch": "main",
  "pushed_at": "2026-07-08T12:00:00Z",
  "language": "Go",
  "owner": { "login": "octocat" },
  "permissions": { "pull": true, "push": true, "admin": false }
}`

const sampleRepoNoPermsJSON = `{
  "id": 456,
  "name": "no-write",
  "full_name": "org/no-write",
  "private": true,
  "visibility": "private",
  "size": 1024,
  "default_branch": "develop",
  "pushed_at": "2026-06-01T00:00:00Z",
  "language": null,
  "owner": { "login": "org" },
  "permissions": { "pull": true, "push": false, "admin": false }
}`

// newMockRepoServer spins up an httptest.Server that serves
// /repos/{owner}/{name} with the given status + body. Records the
// request count + last path + captured bearer token.
func newMockRepoServer(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int64, *string) {
	t.Helper()
	var calls atomic.Int64
	var capturedToken string
	var capturedPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		capturedToken = r.Header.Get("Authorization")
		capturedPath = r.URL.Path
		if !strings.HasPrefix(r.URL.Path, "/repos/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	_ = capturedPath
	return srv, &calls, &capturedToken
}

func TestClient_GetRepository_HappyPath(t *testing.T) {
	srv, calls, tok := newMockRepoServer(t, http.StatusOK, sampleRepoJSON)
	c := gh.NewClientWithBase(srv.URL)
	c.HTTPClient.Timeout = 2 * time.Second

	meta, err := c.GetRepository(context.Background(), "gho_secret", "octocat", "hello")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if calls.Load() != 1 {
		t.Errorf("calls = %d, want 1", calls.Load())
	}
	if tok == nil || *tok != "Bearer gho_secret" {
		t.Errorf("Authorization = %v, want Bearer gho_secret", tok)
	}
	if meta == nil {
		t.Fatalf("meta = nil")
	}
	if meta.GitHubID != 123 {
		t.Errorf("GitHubID = %d, want 123", meta.GitHubID)
	}
	if meta.DefaultBranch != "main" {
		t.Errorf("DefaultBranch = %q, want main", meta.DefaultBranch)
	}
	if meta.Visibility != "public" {
		t.Errorf("Visibility = %q, want public", meta.Visibility)
	}
	if meta.PrimaryLanguage == nil || *meta.PrimaryLanguage != "Go" {
		t.Errorf("PrimaryLanguage = %v, want Go", meta.PrimaryLanguage)
	}
	if !meta.Permissions.Pull || !meta.Permissions.Push {
		t.Errorf("Permissions = %+v, want pull+push both true", meta.Permissions)
	}
	if meta.SizeKB != 42 {
		t.Errorf("SizeKB = %d, want 42", meta.SizeKB)
	}
}

func TestClient_GetRepository_InsufficientPush(t *testing.T) {
	srv, _, _ := newMockRepoServer(t, http.StatusOK, sampleRepoNoPermsJSON)
	c := gh.NewClientWithBase(srv.URL)
	c.HTTPClient.Timeout = 2 * time.Second

	meta, err := c.GetRepository(context.Background(), "gho_x", "org", "no-write")
	if err != nil {
		t.Fatalf("GetRepository: %v", err)
	}
	if meta.Permissions.Pull != true {
		t.Errorf("Permissions.Pull = false, want true")
	}
	if meta.Permissions.Push != false {
		t.Errorf("Permissions.Push = true, want false (insufficient)")
	}
	if meta.PrimaryLanguage != nil {
		t.Errorf("PrimaryLanguage = %v, want nil (no language field)", meta.PrimaryLanguage)
	}
}

func TestClient_GetRepository_NotFound(t *testing.T) {
	srv, _, _ := newMockRepoServer(t, http.StatusNotFound, `{"message":"Not Found"}`)
	c := gh.NewClientWithBase(srv.URL)
	c.HTTPClient.Timeout = 2 * time.Second

	_, err := c.GetRepository(context.Background(), "gho_x", "ghost", "missing")
	if err == nil {
		t.Fatalf("expected error on 404")
	}
	if _, ok := gh.AsNotFound(err); !ok {
		t.Errorf("expected *NotFoundError, got %T (%v)", err, err)
	}
}

func TestClient_GetRepository_Unauthorized(t *testing.T) {
	srv, _, _ := newMockRepoServer(t, http.StatusUnauthorized, `{"message":"bad token"}`)
	c := gh.NewClientWithBase(srv.URL)
	c.HTTPClient.Timeout = 2 * time.Second

	_, err := c.GetRepository(context.Background(), "gho_bad", "octocat", "hello")
	if err == nil {
		t.Fatalf("expected error on 401")
	}
	if _, ok := gh.AsUnauthorized(err); !ok {
		t.Errorf("expected *UnauthorizedError, got %T (%v)", err, err)
	}
}

func TestClient_GetRepository_RateLimited(t *testing.T) {
	srv, _, _ := newMockRepoServer(t, http.StatusForbidden, `{"message":"rate limit"}`)
	c := gh.NewClientWithBase(srv.URL)
	c.HTTPClient.Timeout = 2 * time.Second

	_, err := c.GetRepository(context.Background(), "gho_x", "octocat", "hello")
	if err == nil {
		t.Fatalf("expected error on 403")
	}
	if _, ok := gh.AsRateLimited(err); !ok {
		t.Errorf("expected *RateLimitedError, got %T (%v)", err, err)
	}
}

func TestClient_GetRepository_EmptyToken(t *testing.T) {
	c := gh.NewClient()
	_, err := c.GetRepository(context.Background(), "", "octocat", "hello")
	if err == nil {
		t.Fatalf("expected error on empty token")
	}
	if _, ok := gh.AsUnauthorized(err); !ok {
		t.Errorf("expected *UnauthorizedError, got %T (%v)", err, err)
	}
}

func TestClient_GetRepository_EmptyOwnerOrName(t *testing.T) {
	c := gh.NewClient()
	if _, err := c.GetRepository(context.Background(), "gho_x", "", "hello"); err == nil {
		t.Errorf("expected error on empty owner")
	}
	if _, err := c.GetRepository(context.Background(), "gho_x", "octocat", ""); err == nil {
		t.Errorf("expected error on empty name")
	}
}

func TestClient_GetRepository_ParseError(t *testing.T) {
	srv, _, _ := newMockRepoServer(t, http.StatusOK, `{not valid json`)
	c := gh.NewClientWithBase(srv.URL)
	c.HTTPClient.Timeout = 2 * time.Second

	_, err := c.GetRepository(context.Background(), "gho_x", "octocat", "hello")
	if err == nil {
		t.Fatalf("expected error on malformed JSON")
	}
	// ParseError is unexported type; we verify via wrapped message.
	if err.Error() == "" {
		t.Errorf("expected non-empty error")
	}
}