// Package git — the Runner that shells out to the system `git`
// binary. See design.md §4 (Filesystem layout) and the ADR at
// adr/workspace-syncer-git-impl for the os/exec rationale.
//
// The Runner is an interface so the application layer (and the
// tests) can substitute a fake. The RunnerFunc adapter at the
// bottom of the file is the bridge from the interface to a
// real *git.Runner for production wiring.
package git

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// DefaultCloneTimeoutSeconds is the hard ceiling on a single
// `git clone` invocation. The design (NFR-WSY-001) calls for 90s;
// callers may override via Runner.CloneTimeout.
const DefaultCloneTimeoutSeconds = 90

// Runner is the abstract interface that the application layer
// depends on. The production wiring uses realRunner; tests use
// fakeRunner. The interface is intentionally narrow (3 methods)
// so a fake is trivial to write.
type Runner interface {
	// Clone runs `git clone --bare <url> <path>` with the given
	// timeout. Returns the absolute path on success. On failure
	// returns one of the typed domain errors (or wraps an
	// unexpected error in *domain.CloneFailedError).
	Clone(ctx context.Context, workspaceID int64, owner, repo, oauthToken string) (string, error)

	// WorktreeProbe runs `git -C <path> worktree add /tmp/probe-<id> HEAD`
	// and returns the HEAD commit SHA on success. On failure
	// returns *domain.WorktreeProbeFailedError.
	WorktreeProbe(ctx context.Context, path string) (headSHA string, err error)

	// ResolveHead returns the current HEAD commit SHA of the
	// bare mirror at <path>. Used by the application layer to
	// report `commit_sha_after` on the sync_job.
	ResolveHead(ctx context.Context, path string) (sha string, err error)

	// ResolveDefaultBranch returns the upstream's default branch
	// name (e.g. "main", "master") from the bare mirror at
	// <path>. Implemented via
	//   git symbolic-ref --short refs/remotes/origin/HEAD
	// which resolves to e.g. "origin/main"; we strip the
	// "origin/" prefix to return "main". The value is what the
	// clone actually points at, NOT the current HEAD (which
	// could be on a feature branch in a worktree setup).
	//
	// Used by the application layer to fill in the
	// callback's `default_branch` field; the workspace row's
	// `default_branch` is denormalized from the callback on
	// `done`. UAT fix (2026-07-08): the prior code passed an
	// empty default_branch through the dispatch+callback
	// pipeline, leaving workspace.default_branch NULL.
	ResolveDefaultBranch(ctx context.Context, path string) (branch string, err error)
}

// Compile-time check: the production implementation MUST satisfy
// the Runner interface. If a future change drifts the surface, the
// build breaks here, not in a downstream consumer.
var _ Runner = (*realRunner)(nil)

// realRunner is the production Runner. It shells out to the
// system `git` binary via os/exec.
type realRunner struct {
	gitPath      string
	cloneTimeout time.Duration
}

// NewRunner constructs a realRunner. The gitPath defaults to
// "git" (resolved via $PATH). The cloneTimeout defaults to
// DefaultCloneTimeoutSeconds. Both can be overridden for tests.
func NewRunner() Runner {
	return &realRunner{
		gitPath:      "git",
		cloneTimeout: DefaultCloneTimeoutSeconds * time.Second,
	}
}

// NewRunnerWithTimeout is a test-friendly constructor.
func NewRunnerWithTimeout(gitPath string, cloneTimeout time.Duration) Runner {
	return &realRunner{
		gitPath:      gitPath,
		cloneTimeout: cloneTimeout,
	}
}

// cloneURL constructs the https URL for the clone. The token is
// inlined in the URL — `git` accepts https://x-access-token:<pat>@github.com/...
// as the basic-auth credentials. This is GitHub's documented pattern.
//
// The function does NOT validate the inputs — callers MUST pass
// validated workspaceID, owner, repo, oauthToken (see
// domain.ValidateCloneRequest).
func (r *realRunner) cloneURL(owner, repo, oauthToken string) string {
	// Per GitHub's docs: https://x-access-token:<token>@github.com/<owner>/<repo>.git
	// We use the bare repo URL because the .git suffix is also accepted
	// by `git clone --bare` (and the URL with no .git works too).
	return fmt.Sprintf("https://x-access-token:%s@github.com/%s/%s.git", oauthToken, owner, repo)
}

// Clone runs `git clone --bare <url> <path>` with the configured
// timeout. The context's deadline is honored as a hard cancel.
func (r *realRunner) Clone(ctx context.Context, workspaceID int64, owner, repo, oauthToken string) (string, error) {
	path, err := WorkspacePath(workspaceID, owner, repo)
	if err != nil {
		return "", fmt.Errorf("clone path: %w", err)
	}
	url := r.cloneURL(owner, repo, oauthToken)

	// Apply the clone timeout as a hard deadline on the context.
	// The context is a child of the caller's so cancellation
	// propagates up (e.g. workspace deletion cancels an in-flight clone).
	cloneCtx, cancel := context.WithTimeout(ctx, r.cloneTimeout)
	defer cancel()

	// Args as a []string: NEVER use sh -c with interpolated input.
	cmd := exec.CommandContext(cloneCtx, r.gitPath,
		"clone",
		"--bare",          // bare mirror (no working tree)
		"--no-tags",       // tags are not needed for the worktree probe
		"--depth=1",       // shallow clone (full history is not needed)
		"--single-branch", // only the default branch
		"--no-checkout",   // bare repo; no working tree to check out
		url,               // URL (validated owner/repo in path)
		path,              // destination (validated workspaceID + owner/repo)
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Distinguish the timeout (context.DeadlineExceeded) from
		// other failures. The error message is intentionally
		// generic; the typed error is what the application layer
		// uses to map to the cross-service error code.
		if errors.Is(cloneCtx.Err(), context.DeadlineExceeded) {
			return "", &CloneTimeoutErrorAlias{Seconds: int(r.cloneTimeout.Seconds())}
		}
		return "", &cloneFailedError{Cause: err, Stderr: stderr.String()}
	}

	return path, nil
}

// WorktreeProbe runs the locked probe: `git -C <path> worktree add
// /tmp/probe-<id> HEAD`. The probe validates that the bare mirror
// is fully usable (a future `git worktree add` from a working tree
// will succeed). The /tmp/probe-<id> path is removed after the probe
// regardless of success or failure.
func (r *realRunner) WorktreeProbe(ctx context.Context, path string) (string, error) {
	// The probe is fast (no actual checkout, just metadata). We
	// use a 30s ceiling which is generous for any real-world repo.
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	probePath := fmt.Sprintf("/tmp/probe-%d", time.Now().UnixNano())
	cmd := exec.CommandContext(probeCtx, r.gitPath,
		"-C", path, // run from the bare mirror
		"worktree", "add", // create a worktree
		"--detach", // detached HEAD (no branch checkout)
		probePath,  // destination
		"HEAD",     // from the HEAD commit
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// Best-effort cleanup of the probe dir.
		_ = exec.CommandContext(ctx, "rm", "-rf", probePath).Run()
		return "", &worktreeProbeFailedError{Stderr: stderr.String(), Cause: err}
	}

	// Capture the HEAD SHA from the bare mirror (the worktree's
	// HEAD is the same as the bare's HEAD at this point).
	sha, err := r.ResolveHead(probeCtx, path)
	if err != nil {
		_ = exec.CommandContext(ctx, "rm", "-rf", probePath).Run()
		return "", &cloneFailedError{Cause: err, Stderr: "failed to resolve HEAD after worktree probe"}
	}

	// Clean up the probe dir. We do this AFTER capturing the SHA
	// so a failure to clean up does not lose the data the
	// application layer needs.
	_ = exec.CommandContext(ctx, "rm", "-rf", probePath).Run()
	return sha, nil
}

// ResolveHead returns the HEAD commit SHA of the bare mirror at
// <path>. The result is the full 40-char SHA (the database_administrator
// stores it; the card shortens to 7 chars for display).
func (r *realRunner) ResolveHead(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, r.gitPath,
		"-C", path,
		"rev-parse", "HEAD",
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &cloneFailedError{Cause: err, Stderr: stderr.String()}
	}
	sha := bytes.TrimSpace(stdout.Bytes())
	if len(sha) != 40 {
		return "", &cloneFailedError{Cause: fmt.Errorf("unexpected SHA length %d", len(sha)), Stderr: stderr.String()}
	}
	return string(sha), nil
}

// ResolveDefaultBranch returns the upstream's default branch name
// (stripped of the "refs/heads/" prefix). UAT fix (2026-07-08): the
// prior code never resolved the default branch, so the callback
// body omitted `default_branch` and the workspace's
// `default_branch` column stayed NULL even after a successful sync.
//
// Implementation: `git symbolic-ref HEAD` on a bare mirror
// returns the symbolic ref HEAD points at, e.g.
// "refs/heads/main". This is set by `git clone --bare` to match
// the upstream's default branch at clone time. We strip the
// `refs/heads/` prefix to return just the branch name.
//
// Why HEAD (not refs/remotes/origin/HEAD): the latter is a
// git-clone-with-checkout convention and does NOT exist on a
// bare mirror (verified empirically against a real clone of
// `witsaba/cachicamas`). HEAD on a bare mirror is set by
// `git clone --bare` to the upstream's default branch and IS
// the correct source for the default branch name.
//
// Errors:
//   - bare mirror not at <path>: the git command exits non-zero
//     (no HEAD reference). We return the wrapped error and the
//     caller logs + posts a callback with default_branch="" (the
//     workspace row stays NULL — recoverable on the next sync).
func (r *realRunner) ResolveDefaultBranch(ctx context.Context, path string) (string, error) {
	cmd := exec.CommandContext(ctx, r.gitPath, "-C", path, "symbolic-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git symbolic-ref HEAD: %w", err)
	}
	branch := strings.TrimSpace(string(out))
	branch = strings.TrimPrefix(branch, "refs/heads/")
	if branch == "" {
		return "", fmt.Errorf("git symbolic-ref returned empty branch for %q", path)
	}
	return branch, nil
}

// ---------------------------------------------------------------------------
// Typed errors. Mirror domain errors but live in the git package so
// the application layer can errors.As to them without importing
// domain (keeps the dependency direction clean: application
// imports git, not the other way around).
// ---------------------------------------------------------------------------

type cloneTimeoutError struct{ Seconds int }

func (e *cloneTimeoutError) Error() string {
	return fmt.Sprintf("repository clone took longer than %d seconds", e.Seconds)
}

// CloneTimeoutErrorAlias is the public type alias for the
// cloneTimeoutError. The application layer uses errors.As to
// match it; the alias is the only way to reference the unexported
// type from outside the git package.
type CloneTimeoutErrorAlias = cloneTimeoutError

type cloneFailedError struct {
	Cause  error
	Stderr string
}

func (e *cloneFailedError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("clone failed: %v (stderr: %s)", e.Cause, e.Stderr)
	}
	return fmt.Sprintf("clone failed: %v", e.Cause)
}

type worktreeProbeFailedError struct {
	Cause  error
	Stderr string
}

func (e *worktreeProbeFailedError) Error() string {
	if e.Stderr != "" {
		return fmt.Sprintf("git worktree add failed: %v (stderr: %s)", e.Cause, e.Stderr)
	}
	return fmt.Sprintf("git worktree add failed: %v", e.Cause)
}
