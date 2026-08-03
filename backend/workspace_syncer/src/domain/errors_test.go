package domain

import "testing"

// TestAppError_CodeLockdown pins the AppError.Code() return values
// for every typed error. A change to a code is a breaking change to
// the database_administrator contract and requires an ADR (see
// design §6 and the workspace-syncer spec R-WSY-001).
func TestAppError_CodeLockdown(t *testing.T) {
	cases := []struct {
		name string
		err  AppError
		want string
	}{
		{"PermissionsInsufficient", &PermissionsInsufficientError{Owner: "o", Repo: "r"}, ErrCodePermissionsInsufficient},
		{"BranchNotFound", &BranchNotFoundError{Owner: "o", Repo: "r", DefaultBranch: "main"}, ErrCodeBranchNotFound},
		{"WorktreeProbeFailed", &WorktreeProbeFailedError{ExitCode: 1}, ErrCodeWorktreeProbeFailed},
		{"CloneTimeout", &CloneTimeoutError{TimeoutSeconds: 90}, ErrCodeCloneTimeout},
		{"TokenExpired", &TokenExpiredError{}, ErrCodeTokenExpired},
		{"CloneFailed", &CloneFailedError{}, ErrCodeCloneFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.err.Code(); got != tc.want {
				t.Errorf("%s.Code() = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestAppError_NotNil guards against a future contributor returning
// a nil AppError from a constructor.
func TestAppError_NotNil(t *testing.T) {
	errors := []AppError{
		&PermissionsInsufficientError{},
		&BranchNotFoundError{},
		&WorktreeProbeFailedError{},
		&CloneTimeoutError{},
		&TokenExpiredError{},
		&CloneFailedError{},
	}
	for i, err := range errors {
		if err == nil {
			t.Errorf("AppError[%d] is nil", i)
		}
		if err.Error() == "" {
			t.Errorf("AppError[%d].Error() is empty; consumers log the message", i)
		}
	}
}

// TestCloneErrorCode_IsValid pins the closed vocab of codes the
// application layer is permitted to send in a callback. Per spec
// audit finding H-1, the callback envelope MUST NOT carry a
// free-form code; a future contributor who adds a code outside
// this set MUST update this test (and the locked vocabulary).
func TestCloneErrorCode_IsValid(t *testing.T) {
	cases := []struct {
		code CloneErrorCode
		want bool
	}{
		{CloneErrCodeFailed, true},
		{CloneErrCodeTimeout, true},
		{CloneErrCodeWorktreeProbeFailed, true},
		{CloneErrCodeGitHubUnreachable, true},
		{CloneErrCodeInvalidRepo, true},
		{CloneErrCodeAuthFailed, true},
		{CloneErrorCode("FREE_FORM"), false},
		{CloneErrorCode(""), false},
		{CloneErrorCode("CLONE_FAILED: detail here"), false},
	}
	for _, tc := range cases {
		if got := tc.code.IsValid(); got != tc.want {
			t.Errorf("CloneErrorCode(%q).IsValid() = %v, want %v", tc.code, got, tc.want)
		}
	}
}

// TestCloneErrorCode_WireStrings pins the wire string for each
// code. The cross-service contract is the literal string;
// renaming a code is a breaking change.
func TestCloneErrorCode_WireStrings(t *testing.T) {
	cases := []struct {
		code CloneErrorCode
		want string
	}{
		{CloneErrCodeFailed, "CLONE_FAILED"},
		{CloneErrCodeTimeout, "CLONE_TIMEOUT"},
		{CloneErrCodeWorktreeProbeFailed, "WORKTREE_PROBE_FAILED"},
		{CloneErrCodeGitHubUnreachable, "GITHUB_UNREACHABLE"},
		{CloneErrCodeInvalidRepo, "INVALID_REPO"},
		{CloneErrCodeAuthFailed, "AUTH_FAILED"},
	}
	for _, tc := range cases {
		if got := tc.code.String(); got != tc.want {
			t.Errorf("CloneErrorCode(%q).String() = %q, want %q", tc.code, got, tc.want)
		}
	}
}

// TestIsLockedErrorMessage_ClosedVocab asserts that arbitrary
// text (including text containing err.Error() content or
// stderr) is REJECTED by the closed-vocab guard.
func TestIsLockedErrorMessage_ClosedVocab(t *testing.T) {
	allowed := []string{
		"",
		"token lacks push permission on this repository",
		"github api unreachable",
		"repository not accessible to this token",
		"token rejected by github",
		"clone failed unexpectedly",
		"clone exceeded timeout",
		"worktree probe failed",
	}
	for _, msg := range allowed {
		if !IsLockedErrorMessage(msg) {
			t.Errorf("IsLockedErrorMessage(%q) = false; want true (this is a locked message)", msg)
		}
	}
	rejected := []string{
		"fatal: could not clone https://x-access-token:ghp_abc@github.com/foo/bar",
		"network down",
		"stderr: fatal: Authentication failed",
		"some user-supplied note",
		"clone failed: exit status 128",
		"permission denied (publickey)",
	}
	for _, msg := range rejected {
		if IsLockedErrorMessage(msg) {
			t.Errorf("IsLockedErrorMessage(%q) = true; want false (free-form text must be rejected)", msg)
		}
	}
}
