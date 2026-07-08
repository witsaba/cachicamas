// Package git provides the canonical filesystem layout for cloned
// repositories on the workspace_syncer host. The path pattern is
//
//	/data/workspaces/{workspace_id}/{owner}/{repo}.git/
//
// The bare-mirror convention is locked (see design.md §4): the path
// ends in `.git/` and contains a bare git repository (no working
// tree files at the top level). The future worktree feature uses
// `git worktree add` from the bare mirror, which is the canonical
// pattern.
//
// SECURITY: this file is the single source of truth for path
// construction. Both the validation (positive int64, alphanumeric
// owner/repo) and the path format are locked. Any future change
// MUST be reviewed for path-traversal and shell-injection vectors.
package git

import (
	"fmt"
	"regexp"
	"strconv"
)

// repoSegmentPattern is the locked regex for owner and repo
// segments. Matches GitHub's own rules: alphanumerics plus `.`,
// `_`, and `-`. Defense-in-depth against path traversal and shell
// injection — the git URL and the filesystem path are constructed
// from these values.
const repoSegmentPattern = `^[a-zA-Z0-9._-]+$`

// repoSegmentMaxLen is the max length of a single owner or repo
// segment. GitHub's own limit is 100 chars; we keep the same limit
// for defense-in-depth.
const repoSegmentMaxLen = 100

var repoSegmentRegexp = regexp.MustCompile(repoSegmentPattern)

// WorkspacePath returns the canonical bare-mirror path for the
// workspace's cloned repository. The path is
// /data/workspaces/{workspace_id}/{owner}/{repo}.git/.
//
// The function is pure: no I/O, no clock, no logging. The caller is
// responsible for any directory creation.
//
// Returns an error when:
//   - workspaceID is not a positive int64 (BIGSERIAL invariant).
//   - owner or repo is empty, too long, or contains characters
//     outside [a-zA-Z0-9._-].
//
// The error message intentionally does NOT echo the invalid input
// back to the caller — defense-in-depth against log-injection (a
// future caller that logs the error must not render unsanitized
// user input).
func WorkspacePath(workspaceID int64, owner, repo string) (string, error) {
	if workspaceID <= 0 {
		return "", fmt.Errorf("invalid workspace_id: must be positive int64")
	}
	if !validRepoSegment(owner) {
		return "", fmt.Errorf("invalid owner: must match %s", repoSegmentPattern)
	}
	if !validRepoSegment(repo) {
		return "", fmt.Errorf("invalid repo: must match %s", repoSegmentPattern)
	}
	// strconv.FormatInt is safe (no shell metachars). The path is
	// then constructed from the validated components only.
	return "/data/workspaces/" + strconv.FormatInt(workspaceID, 10) + "/" + owner + "/" + repo + ".git/", nil
}

// validRepoSegment returns true if s is a non-empty, length-bounded
// alphanumeric + `._-` string. The special filesystem entries `.`
// and `..` are rejected explicitly (the regex would otherwise
// accept them because `.` is in the character class).
func validRepoSegment(s string) bool {
	if s == "" || len(s) > repoSegmentMaxLen {
		return false
	}
	if s == "." || s == ".." {
		return false
	}
	return repoSegmentRegexp.MatchString(s)
}
