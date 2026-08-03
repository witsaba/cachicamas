// Package domain — locked error vocabulary for the workspace_syncer.
//
// The errors below are the typed shapes that the application layer
// returns. The infrastructure layer (the git runner, the GitHub REST
// caller) maps the underlying concrete errors to one of these typed
// shapes so the application layer never sees pgx / exec.ExitError /
// url.Error directly. The handler maps each error type to a locked
// HTTP envelope (see design §6 — Error envelope consolidation).
//
// The ErrorCode values are the locked machine codes that the
// database_administrator callback surfaces on workspace.last_sync_job_id
// (via the sync_job.error_code column) and on the card's inline error
// banner (via the workspace.error_message column). Any future change
// to a code is a breaking change to the database_administrator
// contract and requires an ADR.
package domain

import "fmt"

// ---------------------------------------------------------------------------
// Locked error codes — these are the strings that flow into
// database_administrator's sync_job.error_code column and the card's
// UI. Pinned in openspec/changes/2026-07-08-workspace-sync-clone/specs/workspace-syncer/spec.md
// and the design.md §6.
// ---------------------------------------------------------------------------

const (
	// ErrCodePermissionsInsufficient — the OAuth token does not grant
	// `permissions.push === true` on the workspace's primary repo.
	// UI hint: "Reconnect GitHub to refresh the token's scope."
	ErrCodePermissionsInsufficient = "WORKSPACE_PERMISSIONS_INSUFFICIENT"

	// ErrCodeBranchNotFound — the workspace's default_branch does not
	// exist on the remote (e.g. force-push renamed it). UI hint: "The
	// repo's default branch was renamed. Pick a new primary repo or
	// contact the repo owner."
	ErrCodeBranchNotFound = "BRANCH_NOT_FOUND"

	// ErrCodeWorktreeProbeFailed — the git clone succeeded but the
	// `git worktree add /tmp/probe HEAD` probe exited non-zero. UI
	// hint: "The repo is too large or has an unusual history for the
	// worktree operation. Contact support."
	ErrCodeWorktreeProbeFailed = "WORKTREE_PROBE_FAILED"

	// ErrCodeCloneTimeout — the git clone exceeded the configured
	// 90s timeout. UI hint: "Repository clone took longer than 90
	// seconds. Retry."
	ErrCodeCloneTimeout = "CLONE_TIMEOUT"

	// ErrCodeTokenExpired — the GitHub API rejected the access_token
	// with 401 (token revoked or expired). UI hint: "Reconnect GitHub
	// to refresh the token."
	ErrCodeTokenExpired = "TOKEN_EXPIRED"

	// ErrCodeCloneFailed — catch-all for unexpected git failures
	// (network error, git not installed in the container, etc.). UI
	// hint: "Clone failed unexpectedly. Retry."
	ErrCodeCloneFailed = "CLONE_FAILED"
)

// CloneErrorCode is the closed vocabulary of error codes that the
// application layer attaches to the database_administrator callback
// when the clone pipeline fails. Per spec audit finding H-1 the
// code MUST be a value from this set — never a free-form string,
// never an echo of stderr or err.Error(). The db_admin
// denormalizes the code onto sync_job.error_code; a future code
// is a breaking change to the cross-service contract.
//
// The set deliberately excludes codes that carry context the
// caller might leak (e.g. "INVALID_REPO" is preferred over
// "REPO_NOT_FOUND: <owner>/<repo>" so the owner/repo never reaches
// the wire on a failure).
type CloneErrorCode string

const (
	// CloneErrCodeFailed — catch-all for unexpected git failures.
	CloneErrCodeFailed CloneErrorCode = CloneErrorCode(ErrCodeCloneFailed)
	// CloneErrCodeTimeout — the git clone exceeded the timeout.
	CloneErrCodeTimeout CloneErrorCode = CloneErrorCode(ErrCodeCloneTimeout)
	// CloneErrCodeWorktreeProbeFailed — the worktree probe exited non-zero.
	CloneErrCodeWorktreeProbeFailed CloneErrorCode = CloneErrorCode(ErrCodeWorktreeProbeFailed)
	// CloneErrCodeGitHubUnreachable — the GitHub REST API was not reachable
	// (network error, DNS failure, TLS handshake). Distinct from INVALID_REPO
	// (which means the repo exists but the token can't push to it) and from
	// AUTH_FAILED (which means the token is invalid).
	CloneErrCodeGitHubUnreachable CloneErrorCode = "GITHUB_UNREACHABLE"
	// CloneErrCodeInvalidRepo — the repo is not accessible to the token
	// (404, or push permission denied). Distinct from AUTH_FAILED which is
	// reserved for credential-shape failures.
	CloneErrCodeInvalidRepo CloneErrorCode = "INVALID_REPO"
	// CloneErrCodeAuthFailed — the token is malformed, expired, revoked,
	// or otherwise rejected by GitHub with 401/403.
	CloneErrCodeAuthFailed CloneErrorCode = "AUTH_FAILED"
)

// String returns the wire string for the code. The value MUST
// match one of the locked ErrCode* constants (the enum values
// above are pinned to those strings).
func (c CloneErrorCode) String() string { return string(c) }

// IsValid reports whether the code is a member of the closed
// vocabulary. The clone_service uses this to assert that no
// free-form string leaks into the callback envelope.
func (c CloneErrorCode) IsValid() bool {
	switch c {
	case CloneErrCodeFailed, CloneErrCodeTimeout, CloneErrCodeWorktreeProbeFailed,
		CloneErrCodeGitHubUnreachable, CloneErrCodeInvalidRepo, CloneErrCodeAuthFailed:
		return true
	}
	return false
}

// lockedCloneErrorMessages is the closed vocabulary for the
// callback's `error_message` field. Per spec audit finding H-1
// the message MUST come from this set — NEVER from err.Error(),
// NEVER from stderr, NEVER from user input. An empty string is
// the default and is itself a member of the closed vocab
// ("no detail is the safe default").
//
// The slice is used at runtime to assert closed-vocab; a future
// contributor who adds a free-form message will fail the
// assertion in CloneAndValidate.
var lockedCloneErrorMessages = map[string]struct{}{
	"":                  {},
	"token lacks push permission on this repository": {},
	"github api unreachable":                            {},
	"repository not accessible to this token":           {},
	"token rejected by github":                          {},
	"clone failed unexpectedly":                         {},
	"clone exceeded timeout":                            {},
	"worktree probe failed":                             {},
}

// IsLockedErrorMessage reports whether msg is in the closed
// vocabulary. The clone_service uses this in a test (and at
// runtime in dev builds via the errCodeMessage helper) to
// guarantee no free-form text reaches the callback.
func IsLockedErrorMessage(msg string) bool {
	_, ok := lockedCloneErrorMessages[msg]
	return ok
}

// ---------------------------------------------------------------------------
// AppError interface — the marker for handler errors. The handler
// uses errors.As to map each typed error to the locked HTTP
// envelope. Mirrors the database_administrator convention.
// ---------------------------------------------------------------------------

// AppError is the marker interface for handler-mapped errors.
type AppError interface {
	error
	// Code returns the machine-readable error code (e.g.
	// WORKSPACE_PERMISSIONS_INSUFFICIENT). The value is locked and
	// MUST match one of the ErrCode* constants.
	Code() string
}

// ---------------------------------------------------------------------------
// Typed error shapes — one per failure mode. The git runner and the
// HTTP client (to GitHub) return these so the application layer
// never sees concrete error types.
// ---------------------------------------------------------------------------

// PermissionsInsufficientError signals that the OAuth token does
// not grant push permission on the target repo. The handler maps
// this to a CallbackFailure with code WORKSPACE_PERMISSIONS_INSUFFICIENT.
type PermissionsInsufficientError struct {
	Owner string
	Repo  string
}

func (e *PermissionsInsufficientError) Error() string {
	return fmt.Sprintf("token lacks push permission on %s/%s", e.Owner, e.Repo)
}

func (e *PermissionsInsufficientError) Code() string { return ErrCodePermissionsInsufficient }

// BranchNotFoundError signals that the default_branch is missing
// on the remote. The handler maps this to a CallbackFailure with
// code BRANCH_NOT_FOUND.
type BranchNotFoundError struct {
	Owner         string
	Repo          string
	DefaultBranch string
}

func (e *BranchNotFoundError) Error() string {
	return fmt.Sprintf("default branch %q not found on %s/%s", e.DefaultBranch, e.Owner, e.Repo)
}

func (e *BranchNotFoundError) Code() string { return ErrCodeBranchNotFound }

// WorktreeProbeFailedError signals that the worktree probe exited
// non-zero after a successful clone. The handler maps this to a
// CallbackFailure with code WORKTREE_PROBE_FAILED.
type WorktreeProbeFailedError struct {
	ExitCode int
}

func (e *WorktreeProbeFailedError) Error() string {
	return fmt.Sprintf("git worktree add exited with code %d", e.ExitCode)
}

func (e *WorktreeProbeFailedError) Code() string { return ErrCodeWorktreeProbeFailed }

// CloneTimeoutError signals that the git clone exceeded the
// configured timeout. The handler maps this to a CallbackFailure
// with code CLONE_TIMEOUT.
type CloneTimeoutError struct {
	TimeoutSeconds int
}

func (e *CloneTimeoutError) Error() string {
	return fmt.Sprintf("repository clone took longer than %d seconds", e.TimeoutSeconds)
}

func (e *CloneTimeoutError) Code() string { return ErrCodeCloneTimeout }

// TokenExpiredError signals that GitHub returned 401 for the
// access_token. The handler maps this to a CallbackFailure with
// code TOKEN_EXPIRED.
type TokenExpiredError struct{}

func (e *TokenExpiredError) Error() string {
	return "github token is no longer valid; please reconnect github"
}

func (e *TokenExpiredError) Code() string { return ErrCodeTokenExpired }

// CloneFailedError is the catch-all for unexpected git failures.
// The handler maps this to a CallbackFailure with code CLONE_FAILED.
type CloneFailedError struct {
	Cause error
}

func (e *CloneFailedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("clone failed: %v", e.Cause)
	}
	return "clone failed"
}

func (e *CloneFailedError) Code() string { return ErrCodeCloneFailed }
