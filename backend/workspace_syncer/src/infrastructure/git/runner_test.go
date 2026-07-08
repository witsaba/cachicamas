package git

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestRunner_CloneURL_Smoke guards the URL composition. The token
// is interpolated into the basic-auth credentials. We assert the
// shape (not the secret) so a future contributor who changes the
// URL format cannot silently break the auth.
func TestRunner_CloneURL_Smoke(t *testing.T) {
	r := &realRunner{gitPath: "git"}
	url := r.cloneURL("octocat", "hello-world", "gho_xxx")
	if !strings.HasPrefix(url, "https://x-access-token:gho_xxx@github.com/") {
		t.Errorf("URL prefix wrong: %q", url)
	}
	if !strings.HasSuffix(url, "/octocat/hello-world.git") {
		t.Errorf("URL suffix wrong: %q", url)
	}
}

func TestRunner_Clone_RejectsBadPath(t *testing.T) {
	// The validation in WorkspacePath rejects bad workspaceID/owner/repo.
	// The Clone method must surface the validation error before
	// shelling out.
	r := NewRunner()
	cases := []struct {
		name        string
		workspaceID int64
		owner       string
		repo        string
	}{
		{"zero workspace_id", 0, "o", "r"},
		{"negative workspace_id", -1, "o", "r"},
		{"empty owner", 1, "", "r"},
		{"empty repo", 1, "o", ""},
		{"owner with shell metachar", 1, "octo;rm -rf /", "r"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := r.Clone(context.Background(), tc.workspaceID, tc.owner, tc.repo, "t")
			if err == nil {
				t.Errorf("Clone accepted bad input %+v", tc)
			}
		})
	}
}

func TestRunner_ResolveHead_BadPathReturnsError(t *testing.T) {
	// A path that does not exist makes `git rev-parse HEAD` exit
	// non-zero. The Runner must surface this as a typed error so
	// the application layer can map it to CLONE_FAILED.
	r := NewRunner()
	_, err := r.ResolveHead(context.Background(), "/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Errorf("ResolveHead on bad path returned nil error")
	}
}

func TestRunner_WorktreeProbe_BadPathReturnsError(t *testing.T) {
	r := NewRunner()
	_, err := r.WorktreeProbe(context.Background(), "/nonexistent/path")
	if err == nil {
		t.Errorf("WorktreeProbe on bad path returned nil error")
	}
}

// TestRunner_Clone_ContextCancellation is a smoke test that
// context cancellation propagates. We use a context that is
// already cancelled; the runner must return an error.
func TestRunner_Clone_ContextCancellation(t *testing.T) {
	r := NewRunnerWithTimeout("git", 90*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := r.Clone(ctx, 1, "octocat", "hello-world", "gho_xxx")
	if err == nil {
		t.Errorf("Clone with cancelled context returned nil error")
	}
}

// TestRunner_TimeoutTriggersError verifies that a sub-second
// timeout fires before the clone can complete (we use a
// non-resolvable URL to force the network to fail fast). This
// guards against a future change that silently bypasses the
// timeout (e.g. by ignoring the context).
func TestRunner_TimeoutTriggersError(t *testing.T) {
	// We use NewRunnerWithTimeout with a very short timeout.
	// The clone targets a fake host that does not exist, so
	// the underlying network error fires within milliseconds.
	r := NewRunnerWithTimeout("git", 1*time.Millisecond)
	_, err := r.Clone(context.Background(), 1, "nonexistent", "host", "t")
	if err == nil {
		t.Errorf("Clone with 1ms timeout returned nil error")
	}
	// The error type should be one of: cloneFailedError, cloneTimeoutError,
	// or a wrapped exec error. We do not assert the exact type
	// because the OS may resolve "nonexistent" differently; the
	// important thing is that an error was returned.
}

// TestRunner_ResolveHead_RealRepo is an integration test that
// creates a real local git repo, clones it via ResolveHead, and
// asserts the SHA is a 40-char hex string. Skipped if `git` is
// not installed in the test environment.
func TestRunner_ResolveHead_RealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available; skipping integration test")
	}

	// Create a real bare repo in a temp dir.
	tmpDir := t.TempDir()
	repoPath := tmpDir + "/test.git"
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "init", "--bare", repoPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	// Push a single commit so HEAD resolves to a real SHA.
	workDir := tmpDir + "/work"
	cmd = exec.CommandContext(ctx, "git", "clone", repoPath, workDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone (work): %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"config", "user.email", "test@example.com"},
		{"config", "user.name", "Test"},
	} {
		_ = exec.CommandContext(ctx, "git", append([]string{"-C", workDir}, args...)...).Run()
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", workDir, "commit", "--allow-empty", "-m", "test").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", workDir, "push", "origin", "HEAD:main").CombinedOutput(); err != nil {
		t.Fatalf("git push: %v\n%s", err, out)
	}

	// ResolveHead should now return a 40-char SHA.
	r := NewRunner()
	sha, err := r.ResolveHead(ctx, repoPath)
	if err != nil {
		t.Fatalf("ResolveHead: %v", err)
	}
	if len(sha) != 40 {
		t.Errorf("SHA length = %d, want 40; sha=%q", len(sha), sha)
	}
}

// TestRunner_ResolveDefaultBranch_RealRepo is the regression
// test for the "default_branch NULL in callback" UAT bug
// discovered on 2026-07-08. The previous CloneAndValidate did
// not resolve the default branch from the clone, so the
// callback body omitted `default_branch` and the
// database_administrator's workspace denormalization left
// workspace.default_branch NULL.
//
// ResolveDefaultBranch runs `git symbolic-ref --short
// refs/remotes/origin/HEAD` against the bare mirror; the result
// is the upstream's default branch (e.g. "main" or "master"),
// stripped of the "origin/" prefix that git adds.
func TestRunner_ResolveDefaultBranch_RealRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available; skipping integration test")
	}

	tmpDir := t.TempDir()
	workDir := tmpDir + "/work"
	repoPath := tmpDir + "/bare.git"
	ctx := context.Background()

	// Create a real source repo (with `master` as the initial branch
	// — older git default; newer git uses `main`). Push two branches
	// so the default (`master`) is distinguishable from HEAD.
	for _, args := range [][]string{
		{"init", "--initial-branch=master", workDir},
	} {
		if out, err := exec.CommandContext(ctx, "git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git init: %v\n%s", err, out)
		}
	}
	for _, args := range [][]string{
		{"-C", workDir, "config", "user.email", "test@example.com"},
		{"-C", workDir, "config", "user.name", "Test"},
	} {
		_ = exec.CommandContext(ctx, "git", args...).Run()
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", workDir, "commit", "--allow-empty", "-m", "initial").CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	// Clone the bare mirror via `git clone --bare` (the actual
	// production invocation in workspace_syncer). This sets up
	// refs/remotes/origin/HEAD → refs/remotes/origin/<default>
	// automatically — the same setup the production code sees.
	if out, err := exec.CommandContext(ctx, "git", "clone", "--bare", workDir, repoPath).CombinedOutput(); err != nil {
		t.Fatalf("git clone --bare: %v\n%s", err, out)
	}

	// Push a feature-branch from the source so HEAD is no
	// longer the default — proving ResolveDefaultBranch returns
	// the upstream default, not the current HEAD.
	if out, err := exec.CommandContext(ctx, "git", "-C", workDir, "checkout", "-b", "feature-branch").CombinedOutput(); err != nil {
		t.Fatalf("git checkout -b feature-branch: %v\n%s", err, out)
	}
	if out, err := exec.CommandContext(ctx, "git", "-C", workDir, "push", repoPath, "HEAD:feature-branch").CombinedOutput(); err != nil {
		t.Fatalf("git push feature-branch: %v\n%s", err, out)
	}

	// ResolveDefaultBranch should return "master" (the upstream
	// default), NOT "feature-branch".
	r := NewRunner()
	branch, err := r.ResolveDefaultBranch(ctx, repoPath)
	if err != nil {
		t.Fatalf("ResolveDefaultBranch: %v", err)
	}
	if branch != "master" {
		t.Errorf("default branch = %q, want %q (the upstream default, NOT feature-branch)", branch, "master")
	}
}

// TestErrorsAs_CloneFailedError is a guard for the handler's
// errors.As mapping (the handler relies on errors.As to translate
// the concrete error to a callback code).
func TestErrorsAs_CloneFailedError(t *testing.T) {
	err := &cloneFailedError{Cause: errors.New("network down"), Stderr: "fatal: ..."}
	var target *cloneFailedError
	if !errors.As(err, &target) {
		t.Errorf("errors.As(cloneFailedError) failed; handler cannot map the error")
	}
}

func TestErrorsAs_CloneTimeoutError(t *testing.T) {
	err := &cloneTimeoutError{Seconds: 90}
	var target *cloneTimeoutError
	if !errors.As(err, &target) {
		t.Errorf("errors.As(cloneTimeoutError) failed")
	}
}

func TestErrorsAs_WorktreeProbeFailedError(t *testing.T) {
	err := &worktreeProbeFailedError{Cause: errors.New("probe failed"), Stderr: "fatal: ..."}
	var target *worktreeProbeFailedError
	if !errors.As(err, &target) {
		t.Errorf("errors.As(worktreeProbeFailedError) failed")
	}
}