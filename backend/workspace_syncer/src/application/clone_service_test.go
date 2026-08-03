package application

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/workspace_syncer/src/domain"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/git"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/httpclient"
)

// fakeRunner is a controllable Runner for tests. The zero value is
// a runner that returns path=42, sha=ok for all calls; the test
// can override individual fields.
type fakeRunner struct {
	mu               sync.Mutex
	clonePath        string
	cloneErr         error
	probeSHA         string
	probeErr         error
	defaultBranch    string
	defaultBranchErr error
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

func (f *fakeRunner) ResolveDefaultBranch(ctx context.Context, path string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.defaultBranchErr != nil {
		return "", f.defaultBranchErr
	}
	return f.defaultBranch, nil
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
	runner := &fakeRunner{
		clonePath:     "/data/workspaces/7/octocat/hello-world.git/",
		probeSHA:      "abc1234567890abcdef1234567890abcdef12345",
		defaultBranch: "main",
	}
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

// TestCloneService_CloneAndValidate_SuccessCallbackIncludesDefaultBranch
// is the regression test for the UAT bug discovered on 2026-07-08:
// the callback body omitted `default_branch`, leaving the
// workspace's `default_branch` column NULL after a successful sync.
// The fix threads the resolved branch through to the callback
// via CallbackRequest.DefaultBranch (which the db_admin
// denormalizes onto workspace.default_branch on `done`).
func TestCloneService_CloneAndValidate_SuccessCallbackIncludesDefaultBranch(t *testing.T) {
	runner := &fakeRunner{
		clonePath:     "/data/workspaces/7/octocat/hello-world.git/",
		probeSHA:      "abc1234567890abcdef1234567890abcdef12345",
		defaultBranch: "main",
	}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: true}
	svc := NewCloneService(runner, callback, github, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.DefaultBranch != "main" {
		t.Errorf("callback DefaultBranch = %q, want %q (REGRESSION: the UAT bug left workspace.default_branch NULL because the callback body omitted this field)", c.DefaultBranch, "main")
	}
}

// TestCloneService_CloneAndValidate_DefaultBranchResolveError pins
// the failure path: when ResolveDefaultBranch fails, the
// callback body has DefaultBranch="" (the workspace row stays
// NULL — recoverable on the next sync). The use case logs
// the resolve error but does NOT fail the whole clone; the
// callback is still posted with the commit SHA.
func TestCloneService_CloneAndValidate_DefaultBranchResolveError(t *testing.T) {
	runner := &fakeRunner{
		clonePath:        "/data/workspaces/7/octocat/hello-world.git/",
		probeSHA:         "abc1234567890abcdef1234567890abcdef12345",
		defaultBranchErr: errors.New("git symbolic-ref HEAD: exit status 128"),
	}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: true}
	svc := NewCloneService(runner, callback, github, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.Status != "done" {
		t.Errorf("status = %q, want done (the clone succeeded; the default_branch resolve error is non-fatal)", c.Status)
	}
	if c.DefaultBranch != "" {
		t.Errorf("DefaultBranch = %q, want empty (the resolve failed; the callback carries an empty default_branch)", c.DefaultBranch)
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
	if c.ErrorCode != domain.CloneErrCodeInvalidRepo.String() {
		t.Errorf("error_code = %q, want %q (INVALID_REPO closed vocab)", c.ErrorCode, domain.CloneErrCodeInvalidRepo.String())
	}
	if !domain.IsLockedErrorMessage(c.ErrorMessage) {
		t.Errorf("error_message = %q is NOT in the closed vocab (H-1 regression)", c.ErrorMessage)
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
	if c.ErrorCode != domain.CloneErrCodeFailed.String() {
		t.Errorf("error_code = %q, want %q (CLONE_FAILED closed vocab)", c.ErrorCode, domain.CloneErrCodeFailed.String())
	}
	// Per H-1: the message MUST be closed-vocab and MUST NOT
	// echo the err.Error() ("network down" must not appear).
	if !domain.IsLockedErrorMessage(c.ErrorMessage) {
		t.Errorf("error_message = %q is NOT in the closed vocab (H-1 regression)", c.ErrorMessage)
	}
	if strings.Contains(c.ErrorMessage, "network down") {
		t.Errorf("error_message = %q echoes err.Error() (H-1 regression)", c.ErrorMessage)
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
	if c.ErrorCode != domain.CloneErrCodeWorktreeProbeFailed.String() {
		t.Errorf("error_code = %q, want %q (WORKTREE_PROBE_FAILED closed vocab)", c.ErrorCode, domain.CloneErrCodeWorktreeProbeFailed.String())
	}
	if !domain.IsLockedErrorMessage(c.ErrorMessage) {
		t.Errorf("error_message = %q is NOT in the closed vocab (H-1 regression)", c.ErrorMessage)
	}
	if strings.Contains(c.ErrorMessage, "worktree add failed") {
		t.Errorf("error_message = %q echoes err.Error() (H-1 regression)", c.ErrorMessage)
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
	if c.ErrorCode != domain.CloneErrCodeGitHubUnreachable.String() {
		t.Errorf("error_code = %q, want %q (GITHUB_UNREACHABLE closed vocab)", c.ErrorCode, domain.CloneErrCodeGitHubUnreachable.String())
	}
	// Per H-1: the err.Error() detail ("network unreachable")
	// MUST NOT appear in the callback envelope.
	if strings.Contains(c.ErrorMessage, "network unreachable") {
		t.Errorf("error_message = %q echoes err.Error() (H-1 regression)", c.ErrorMessage)
	}
	if !domain.IsLockedErrorMessage(c.ErrorMessage) {
		t.Errorf("error_message = %q is NOT in the closed vocab", c.ErrorMessage)
	}
}

// TestCloneService_CloneAndValidate_CloneTimeoutError is the
// regression test for the CLONE_TIMEOUT path: a timeout error
// from the runner MUST produce the CLONE_TIMEOUT code and a
// closed-vocab message (not the err.Error() which would echo
// the timeout duration).
func TestCloneService_CloneAndValidate_CloneTimeoutError(t *testing.T) {
	timeoutErr := &git.CloneTimeoutErrorAlias{Seconds: 90}
	runner := &fakeRunner{cloneErr: timeoutErr}
	callback := &fakeCallback{}
	github := &fakeGitHub{accessible: true}
	svc := NewCloneService(runner, callback, github, nil)

	svc.CloneAndValidate(context.Background(), validRequest())

	if len(callback.calls) != 1 {
		t.Fatalf("callback calls = %d, want 1", len(callback.calls))
	}
	c := callback.calls[0]
	if c.ErrorCode != domain.CloneErrCodeTimeout.String() {
		t.Errorf("error_code = %q, want %q", c.ErrorCode, domain.CloneErrCodeTimeout.String())
	}
	if !domain.IsLockedErrorMessage(c.ErrorMessage) {
		t.Errorf("error_message = %q is NOT in the closed vocab", c.ErrorMessage)
	}
	// The err.Error() message would contain "90 seconds"; that
	// MUST NOT appear in the callback.
	if strings.Contains(c.ErrorMessage, "90") || strings.Contains(c.ErrorMessage, "second") {
		t.Errorf("error_message = %q echoes timeout details (H-1 regression)", c.ErrorMessage)
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
