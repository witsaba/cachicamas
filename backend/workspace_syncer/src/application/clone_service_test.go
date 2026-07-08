package application

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/cachicamas/backend/workspace_syncer/src/domain"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/httpclient"
)

// fakeRunner is a controllable Runner for tests. The zero value is
// a runner that returns path=42, sha=ok for all calls; the test
// can override individual fields.
type fakeRunner struct {
	mu        sync.Mutex
	clonePath string
	cloneErr  error
	probeSHA  string
	probeErr  error
}

func (f *fakeRunner) Clone(ctx context.Context, workspaceID int64, owner, repo, oauthToken string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.clonePath, f.cloneErr
}

func (f *fakeRunner) WorktreeProbe(ctx context.Context, path string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.probeSHA, f.probeErr
}

func (f *fakeRunner) ResolveHead(ctx context.Context, path string) (string, error) {
	return f.probeSHA, nil
}

// fakeCallback is a controllable CallbackClientPort for tests.
type fakeCallback struct {
	mu      sync.Mutex
	calls   []httpclient.CallbackRequest
	postErr error
}

func (f *fakeCallback) Post(ctx context.Context, req httpclient.CallbackRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	return f.postErr
}

// fakeGitHub is a controllable GitHubAccessorPort for tests.
type fakeGitHub struct {
	accessible bool
	err        error
}

func (f *fakeGitHub) IsRepoAccessible(ctx context.Context, owner, repo string) (bool, error) {
	return f.accessible, f.err
}

func validRequest() domain.CloneRequest {
	return domain.CloneRequest{
		JobID:         42,
		WorkspaceID:   7,
		Owner:         "octocat",
		Repo:          "hello-world",
		DefaultBranch: "main",
		OAuthToken:    "gho_xxx",
	}
}

func TestCloneService_CloneAndValidate_Success(t *testing.T) {
	runner := &fakeRunner{clonePath: "/data/workspaces/7/octocat/hello-world.git/", probeSHA: "abc1234567890abcdef1234567890abcdef12345"}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: true}
	svc := NewCloneService(runner, callback, github, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.Status != "done" {
		t.Errorf("callback status = %q, want done", c.Status)
	}
	if c.CommitSHAAfter != "abc1234567890abcdef1234567890abcdef12345" {
		t.Errorf("callback commit_sha_after = %q", c.CommitSHAAfter)
	}
	if c.ErrorCode != "" {
		t.Errorf("callback error_code = %q, want empty", c.ErrorCode)
	}
}

func TestCloneService_CloneAndValidate_InvalidRequest_NoCallback(t *testing.T) {
	// Validation failures do NOT post a callback (the database_administrator
	// should never have sent an invalid request).
	runner := &fakeRunner{clonePath: "/x", probeSHA: "ok"}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: true}
	svc := NewCloneService(runner, callback, github, nil)

	invalid := validRequest()
	invalid.Owner = "octo;rm -rf /" // shell metachar
	svc.CloneAndValidate(context.Background(), invalid)

	if len(callback.calls) != 0 {
		t.Errorf("callback calls = %d, want 0 (validation failure must not post a callback)", len(callback.calls))
	}
}

func TestCloneService_CloneAndValidate_PermissionDenied(t *testing.T) {
	runner := &fakeRunner{clonePath: "/x", probeSHA: "ok"}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: false}
	svc := NewCloneService(runner, callback, github, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.Status != "failed" {
		t.Errorf("status = %q, want failed", c.Status)
	}
	if c.ErrorCode != "WORKSPACE_PERMISSIONS_INSUFFICIENT" {
		t.Errorf("error_code = %q, want WORKSPACE_PERMISSIONS_INSUFFICIENT", c.ErrorCode)
	}
}

func TestCloneService_CloneAndValidate_CloneError(t *testing.T) {
	runner := &fakeRunner{cloneErr: errors.New("network down")}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: true}
	svc := NewCloneService(runner, callback, github, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.Status != "failed" {
		t.Errorf("status = %q, want failed", c.Status)
	}
	if c.ErrorCode != "CLONE_FAILED" {
		t.Errorf("error_code = %q, want CLONE_FAILED", c.ErrorCode)
	}
}

func TestCloneService_CloneAndValidate_ProbeError(t *testing.T) {
	runner := &fakeRunner{
		clonePath: "/data/workspaces/7/octocat/hello-world.git/",
		probeErr:  errors.New("worktree add failed"),
	}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: true}
	svc := NewCloneService(runner, callback, github, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.Status != "failed" {
		t.Errorf("status = %q, want failed", c.Status)
	}
	if c.ErrorCode != "WORKTREE_PROBE_FAILED" {
		t.Errorf("error_code = %q, want WORKTREE_PROBE_FAILED", c.ErrorCode)
	}
}

func TestCloneService_CloneAndValidate_CallbackFailureIsLogged(t *testing.T) {
	// A failed callback post is logged but does not affect the
	// function's return value (it has none). The runner still
	// completes the work.
	runner := &fakeRunner{clonePath: "/x", probeSHA: "ok"}
	callback := &fakeCallback{postErr: errors.New("callback service down")}
	github := &fakeGitHub{accessible: true}
	svc := NewCloneService(runner, callback, github, nil)

	// No panic, no error. The function is best-effort on the callback.
	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Errorf("callback was called %d times, want 1", len(callback.calls))
	}
}

func TestCloneService_CloneAndValidate_GitHubAccessorError(t *testing.T) {
	runner := &fakeRunner{clonePath: "/x", probeSHA: "ok"}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: false, err: errors.New("network unreachable")}
	svc := NewCloneService(runner, callback, github, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.ErrorCode != "GITHUB_UNREACHABLE" {
		t.Errorf("error_code = %q, want GITHUB_UNREACHABLE", c.ErrorCode)
	}
}

func TestCloneService_WithoutGitHubAccessor(t *testing.T) {
	// When github is nil, the use case skips the permission check
	// and proceeds directly to clone. This is the PR-2b behavior
	// (the real GitHub client lands in PR-2c).
	runner := &fakeRunner{clonePath: "/x", probeSHA: "ok"}
	callback := &fakeCallback{}
	svc := NewCloneService(runner, callback, nil, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.Status != "done" {
		t.Errorf("status = %q, want done (no github accessor → skip permission check)", c.Status)
	}
}