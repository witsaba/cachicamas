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
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// askpassScript is the embedded helper invoked by git via
// GIT_ASKPASS. The script echoes the token from the
// GIT_ASKPASS_TOKEN environment variable that the Go parent sets
// ONLY on the child's exec.Cmd environment (so the parent's
// environment is never contaminated; see spec
// git-credential-helper-integration REQ-03).
//
//go:embed askpass.sh
var askpassScript string

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
// NOT inlined in the URL — credentials are supplied via the
// GIT_ASKPASS credential helper. The URL MUST stay free of
// secrets so it never leaks into argv (`/proc/<pid>/cmdline`,
// `ps`, audit logs). See spec git-credential-helper-integration
// REQ-01 (no URL-embedded token) and audit finding C-1.
//
// The function does NOT validate the inputs — callers MUST pass
// validated workspaceID, owner, repo, oauthToken (see
// domain.ValidateCloneRequest).
func (r *realRunner) cloneURL(owner, repo string) string {
	// Per spec REQ-01: bare GitHub URL, no basic-auth credentials.
	return fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
}

// cloneCmd constructs (but does not run) the *exec.Cmd for the
// git clone. Extracted so tests can inspect cmd.Args and cmd.Env
// without spawning a process. Production code calls cloneCmd then
// cmd.Run.
//
// Per spec git-credential-helper-integration REQ-01: the OAuth
// token MUST NOT appear in cmd.Args. It is delivered via
// GIT_ASKPASS_TOKEN in cmd.Env (REQ-03).
func (r *realRunner) cloneCmd(ctx context.Context, url, dest, askpassPath, oauthToken string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, r.gitPath,
		"clone",
		"--bare",          // bare mirror (no working tree)
		"--no-tags",       // tags are not needed for the worktree probe
		"--depth=1",       // shallow clone (full history is not needed)
		"--single-branch", // only the default branch
		"--no-checkout",   // bare repo; no working tree to check out
		url,               // URL (validated owner/repo; NO TOKEN)
		dest,              // destination (validated workspaceID + owner/repo)
	)
	cmd.Env = append(os.Environ(),
		"GIT_ASKPASS="+askpassPath,
		"GIT_ASKPASS_TOKEN="+oauthToken,
		"GIT_TERMINAL_PROMPT=0",
	)
	return cmd
}

// newAskpassScript materializes the embedded askpass script into
// a per-pid temp file with mode 0o700. The file path is the only
// thing passed via cmd.Args (via GIT_ASKPASS); the OAuth token
// itself travels only through GIT_ASKPASS_TOKEN in cmd.Env. A
// cleanup func is returned so the caller can `defer` the removal.
//
// Spec git-credential-helper-integration:
//   - REQ-01: token MUST NOT appear in clone argv (it doesn't;
//     argv contains only the script path, not the token).
//   - REQ-02: helper MUST be embedded via go:embed (it is;
//     askpassScript is the //go:embed-bound string).
//   - REQ-03: token MUST be scoped to the child exec.Cmd
//     environment, never to the parent (caller does NOT mutate
//     os.Setenv; the env entries are appended to the child's
//     env slice).
//   - REQ-04: helper MUST work in the existing image (it does;
//     sh + the per-pid script).
func (r *realRunner) newAskpassScript() (path string, cleanup func(), err error) {
	f, err := os.CreateTemp("", "ws-syncer-askpass-*.sh")
	if err != nil {
		return "", nil, fmt.Errorf("create askpass temp file: %w", err)
	}
	cleanupPath := f.Name()
	if _, err := f.WriteString(askpassScript); err != nil {
		_ = f.Close()
		_ = os.Remove(cleanupPath)
		return "", nil, fmt.Errorf("write askpass script: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(cleanupPath)
		return "", nil, fmt.Errorf("close askpass file: %w", err)
	}
	// Mode 0o700 so only the running user can read the script
	// (defense in depth — the script contains no token literal,
	// but the mode prevents accidental leakage through /tmp
	// listings).
	if err := os.Chmod(cleanupPath, 0o700); err != nil {
		_ = os.Remove(cleanupPath)
		return "", nil, fmt.Errorf("chmod askpass file: %w", err)
	}
	cleanup = func() { _ = os.Remove(cleanupPath) }
	return cleanupPath, cleanup, nil
}

// Clone runs `git clone --bare <url> <path>` with the configured
// timeout. The context's deadline is honored as a hard cancel.
//
// Per spec git-credential-helper-integration REQ-01 the clone URL
// is token-free; credentials are supplied via GIT_ASKPASS +
// GIT_ASKPASS_TOKEN. The token MUST NOT appear in cmd.Args
// (which would expose it via `/proc/<pid>/cmdline`); the only
// thing in argv is the script path.
func (r *realRunner) Clone(ctx context.Context, workspaceID int64, owner, repo, oauthToken string) (string, error) {
	path, err := WorkspacePath(workspaceID, owner, repo)
	if err != nil {
		return "", fmt.Errorf("clone path: %w", err)
	}
	url := r.cloneURL(owner, repo)

	// Apply the clone timeout as a hard deadline on the context.
	// The context is a child of the caller's so cancellation
	// propagates up (e.g. workspace deletion cancels an in-flight clone).
	cloneCtx, cancel := context.WithTimeout(ctx, r.cloneTimeout)
	defer cancel()

	// Materialize the askpass helper into a per-pid temp file.
	// The path travels through GIT_ASKPASS, NOT through argv;
	// the token itself travels through GIT_ASKPASS_TOKEN on
	// cmd.Env (scoped to the child process only).
	askpassPath, askpassCleanup, err := r.newAskpassScript()
	if err != nil {
		return "", err
	}
	defer askpassCleanup()

	// Args as a []string: NEVER use sh -c with interpolated input.
	cmd := r.cloneCmd(cloneCtx, url, path, askpassPath, oauthToken)

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

// newProbeDir creates a fresh temp directory under os.TempDir()
// for the worktree probe. The directory has a randomized suffix
// so two concurrent probes never collide (the prior
// `/tmp/probe-<nano>` scheme was predictable and world-writable;
// spec audit finding L-2). Returns the path and a cleanup func
// that the caller MUST `defer`.
func newProbeDir() (string, func(), error) {
	dir, err := os.MkdirTemp("", "ws-syncer-probe-*")
	if err != nil {
		return "", nil, fmt.Errorf("create probe dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }
	return dir, cleanup, nil
}

// WorktreeProbe runs the locked probe: `git -C <path> worktree add
// <tmp> HEAD`. The probe validates that the bare mirror
// is fully usable (a future `git worktree add` from a working tree
// will succeed). The probe directory is created with
// os.MkdirTemp (random suffix under os.TempDir) and removed
// after the probe regardless of success or failure.
//
// Spec audit finding L-2: the prior `/tmp/probe-<nanos>` path was
// predictable in a world-writable dir; an attacker could
// pre-create the path with malicious contents. os.MkdirTemp
// returns a unique path that no other process can guess.
func (r *realRunner) WorktreeProbe(ctx context.Context, path string) (string, error) {
	// The probe is fast (no actual checkout, just metadata). We
	// use a 30s ceiling which is generous for any real-world repo.
	probeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	probePath, probeCleanup, err := newProbeDir()
	if err != nil {
		return "", err
	}
	defer probeCleanup()

	cmd := exec.CommandContext(probeCtx, r.gitPath,
		"-C", path, // run from the bare mirror
		"worktree", "add", // create a worktree
		"--detach", // detached HEAD (no branch checkout)
		probePath,  // destination (unique per call)
		"HEAD",     // from the HEAD commit
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &worktreeProbeFailedError{Stderr: stderr.String(), Cause: err}
	}

	// Capture the HEAD SHA from the bare mirror (the worktree's
	// HEAD is the same as the bare's HEAD at this point).
	sha, err := r.ResolveHead(probeCtx, path)
	if err != nil {
		return "", &cloneFailedError{Cause: err, Stderr: "failed to resolve HEAD after worktree probe"}
	}

	// Cleanup runs via the deferred probeCleanup so a failure
	// to capture the SHA does not lose the data the application
	// layer needs (cleanup happens AFTER the SHA is returned).
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
