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
