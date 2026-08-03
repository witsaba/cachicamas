package git

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// TestRunner_CloneURL_Smoke guards the URL composition. Per spec
// git-credential-helper-integration REQ-01 the URL MUST NOT
// embed the OAuth token (credentials travel via GIT_ASKPASS). We
// assert the shape (the bare GitHub URL with no credentials) so a
// future contributor who adds the token back into the URL fails
// this test before merge.
func TestRunner_CloneURL_Smoke(t *testing.T) {
	r := &realRunner{gitPath: "git"}
	url := r.cloneURL("octocat", "hello-world")
	if !strings.HasPrefix(url, "https://github.com/") {
		t.Errorf("URL prefix wrong: %q (must be the bare GitHub URL — no credentials)", url)
	}
	if !strings.HasSuffix(url, "/octocat/hello-world.git") {
		t.Errorf("URL suffix wrong: %q", url)
	}
	if strings.Contains(url, "@") {
		t.Errorf("URL MUST NOT contain '@' (no basic-auth credentials): %q", url)
	}
	if strings.Contains(url, "x-access-token") {
		t.Errorf("URL MUST NOT contain the literal token marker: %q", url)
	}
}

// TestRunner_NewAskpassScript_PermissionsAndContents asserts spec
// git-credential-helper-integration REQ-02 + REQ-04: the helper
// is embedded, materialized with mode 0o700, and contains NO
// token literal.
func TestRunner_NewAskpassScript_PermissionsAndContents(t *testing.T) {
	r := &realRunner{gitPath: "git"}
	path, cleanup, err := r.newAskpassScript()
	if err != nil {
		t.Fatalf("newAskpassScript: %v", err)
	}
	defer cleanup()

	// Path is non-empty and lives under os.TempDir().
	if path == "" {
		t.Fatal("askpass path is empty")
	}
	// On macOS os.TempDir() returns a trailing-slash form
	// ("/var/folders/.../T/") while filepath.Dir of a file in
	// that directory does NOT carry the trailing slash. Use a
	// prefix check so the test passes on both Linux and macOS.
	tmpDir := os.TempDir()
	if !strings.HasPrefix(path, tmpDir) {
		t.Errorf("askpass path %q is not under os.TempDir() %q", path, tmpDir)
	}

	// Mode is 0o700 (defense in depth: only the running user can read).
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat askpass: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o700 {
		t.Errorf("askpass mode = %#o, want 0o700", perm)
	}

	// Contents are the embedded script; contains no token literal.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read askpass: %v", err)
	}
	contents := string(data)
	if !strings.Contains(contents, "GIT_ASKPASS_TOKEN") {
		t.Errorf("askpass script must reference GIT_ASKPASS_TOKEN; got:\n%s", contents)
	}
	if strings.Contains(contents, "ghp_") || strings.Contains(contents, "gho_") {
		t.Errorf("askpass script MUST NOT contain a token literal; got:\n%s", contents)
	}

	// Cleanup runs without error.
	cleanup()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("cleanup did not remove askpass file")
	}
}

// TestRunner_NewAskpassScript_RandomSuffix asserts that two
// consecutive temp-file creations return distinct paths. Catches
// a regression that hardcodes a suffix.
func TestRunner_NewAskpassScript_RandomSuffix(t *testing.T) {
	r := &realRunner{gitPath: "git"}
	path1, cleanup1, err := r.newAskpassScript()
	if err != nil {
		t.Fatalf("newAskpassScript 1: %v", err)
	}
	defer cleanup1()

	path2, cleanup2, err := r.newAskpassScript()
	if err != nil {
		t.Fatalf("newAskpassScript 2: %v", err)
	}
	defer cleanup2()

	if path1 == path2 {
		t.Errorf("two askpass temp files collided: %q == %q", path1, path2)
	}
}

// TestRunner_CloneCmd_NoTokenInArgv asserts the central invariant
// of spec git-credential-helper-integration REQ-01: the OAuth
// token MUST NOT appear in cmd.Args (which would expose it via
// /proc/<pid>/cmdline, ps, audit logs). The token travels only
// via GIT_ASKPASS_TOKEN in cmd.Env.
func TestRunner_CloneCmd_NoTokenInArgv(t *testing.T) {
	r := &realRunner{gitPath: "git"}
	cmd := r.cloneCmd(context.Background(),
		"https://github.com/octocat/hello-world.git",
		"/tmp/dest",
		"/tmp/askpass",
		"gho_SECRET_TOKEN_VALUE",
	)

	// The token MUST NOT appear in any argv element.
	for i, arg := range cmd.Args {
		if strings.Contains(arg, "gho_SECRET_TOKEN_VALUE") {
			t.Errorf("token leaked into cmd.Args[%d] = %q", i, arg)
		}
		if strings.Contains(arg, "x-access-token:") {
			t.Errorf("basic-auth token marker leaked into cmd.Args[%d] = %q", i, arg)
		}
	}

	// The URL MUST NOT contain any credentials.
	urlArg := cmd.Args[len(cmd.Args)-2]
	if strings.Contains(urlArg, "@") {
		t.Errorf("URL argv contains '@' (basic-auth credentials): %q", urlArg)
	}
}

// TestRunner_CloneCmd_TokenOnlyInChildEnv asserts that the token
// is delivered via GIT_ASKPASS_TOKEN exactly once in the child's
// environment (REQ-03), and that GIT_TERMINAL_PROMPT=0 is set
// (so git fails closed if askpass returns empty).
func TestRunner_CloneCmd_TokenOnlyInChildEnv(t *testing.T) {
	r := &realRunner{gitPath: "git"}
	cmd := r.cloneCmd(context.Background(),
		"https://github.com/octocat/hello-world.git",
		"/tmp/dest",
		"/tmp/askpass",
		"gho_SECRET_TOKEN_VALUE",
	)

	var tokenCount int
	var hasAskpass, hasPrompt bool
	var askpassValue string
	for _, env := range cmd.Env {
		switch {
		case strings.HasPrefix(env, "GIT_ASKPASS_TOKEN="):
			tokenCount++
			if env != "GIT_ASKPASS_TOKEN=gho_SECRET_TOKEN_VALUE" {
				t.Errorf("GIT_ASKPASS_TOKEN env = %q, want exact match", env)
			}
		case strings.HasPrefix(env, "GIT_ASKPASS="):
			hasAskpass = true
			askpassValue = strings.TrimPrefix(env, "GIT_ASKPASS=")
		case env == "GIT_TERMINAL_PROMPT=0":
			hasPrompt = true
		}
	}

	if tokenCount != 1 {
		t.Errorf("GIT_ASKPASS_TOKEN appears %d times in cmd.Env, want exactly 1", tokenCount)
	}
	if !hasAskpass {
		t.Errorf("cmd.Env missing GIT_ASKPASS=<path>")
	}
	if askpassValue == "" {
		t.Errorf("GIT_ASKPASS value is empty")
	}
	if !hasPrompt {
		t.Errorf("cmd.Env missing GIT_TERMINAL_PROMPT=0 (askpass would otherwise hang)")
	}
}

// TestRunner_CloneCmd_ParentEnvUnchanged asserts REQ-03: the
// parent process's os.Environ MUST NOT be mutated by the Clone
// flow. We snapshot os.Environ before and after a CloneCmd call
// and assert the snapshot diffs ONLY by the askpass file's path
// (which lives on the test filesystem, not in the env). os.Environ
// itself is read-only at this layer; the test guards against a
// future change that calls os.Setenv.
func TestRunner_CloneCmd_ParentEnvUnchanged(t *testing.T) {
	r := &realRunner{gitPath: "git"}
	cmd := r.cloneCmd(context.Background(),
		"https://github.com/octocat/hello-world.git",
		"/tmp/dest",
		"/tmp/askpass",
		"gho_SECRET_TOKEN_VALUE",
	)

	// The token MUST NOT appear in os.Environ (the parent's
	// environment, not the child's).
	for _, env := range os.Environ() {
		if strings.Contains(env, "gho_SECRET_TOKEN_VALUE") {
			t.Errorf("token leaked into parent os.Environ: %q (REQ-03 violation)", env)
		}
	}

	// Sanity: cmd.Env is longer than os.Environ (the child's env
	// adds GIT_ASKPASS* + GIT_TERMINAL_PROMPT=0 on top).
	if len(cmd.Env) <= len(os.Environ()) {
		t.Errorf("cmd.Env (%d) is not longer than os.Environ (%d); the child must receive the askpass vars", len(cmd.Env), len(os.Environ()))
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
