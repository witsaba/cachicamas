package git

import (
	"strconv"
	"strings"
	"testing"
)

func TestWorkspacePath_Happy(t *testing.T) {
	got, err := WorkspacePath(42, "octocat", "hello-world")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/data/workspaces/42/octocat/hello-world.git/"
	if got != want {
		t.Errorf("WorkspacePath(42, octocat, hello-world) = %q, want %q", got, want)
	}
}

func TestWorkspacePath_RejectsNonPositiveID(t *testing.T) {
	cases := []int64{0, -1, -42, -9223372036854775808}
	for _, id := range cases {
		t.Run("id="+itoa(id), func(t *testing.T) {
			if _, err := WorkspacePath(id, "octocat", "hello"); err == nil {
				t.Errorf("WorkspacePath(%d, ...) accepted, want error", id)
			}
		})
	}
}

func TestWorkspacePath_RejectsBadOwner(t *testing.T) {
	cases := []string{
		"",                       // empty
		"octo cat",               // space
		"octo;rm -rf /",          // shell metachar
		"$(rm -rf /)",            // command substitution
		"a/b",                    // path separator
		"a\\b",                   // windows path separator
		strings.Repeat("a", repoSegmentMaxLen+1), // too long
	}
	for _, owner := range cases {
		t.Run("owner="+owner, func(t *testing.T) {
			if _, err := WorkspacePath(1, owner, "r"); err == nil {
				t.Errorf("WorkspacePath accepted bad owner %q, want error", owner)
			}
		})
	}
}

func TestWorkspacePath_RejectsBadRepo(t *testing.T) {
	cases := []string{
		"",                       // empty
		"hello world",             // space
		"hello;rm",               // shell metachar
		strings.Repeat("a", repoSegmentMaxLen+1), // too long
	}
	for _, repo := range cases {
		t.Run("repo="+repo, func(t *testing.T) {
			if _, err := WorkspacePath(1, "o", repo); err == nil {
				t.Errorf("WorkspacePath accepted bad repo %q, want error", repo)
			}
		})
	}
}

func TestWorkspacePath_PathTraversalIsRejected(t *testing.T) {
	// The path is /data/workspaces/{id}/{owner}/{repo}.git/. A
	// malicious owner of "../../../etc" would attempt to escape
	// /data/workspaces. The validRepoSegment check rejects the
	// ".." owner before the path is constructed.
	if _, err := WorkspacePath(1, "..", "passwd"); err == nil {
		t.Errorf("WorkspacePath accepted path-traversal attempt")
	}
	if _, err := WorkspacePath(1, "../../../etc", "passwd"); err == nil {
		t.Errorf("WorkspacePath accepted multi-level path-traversal attempt")
	}
}

func TestWorkspacePath_AcceptsMaxLengthSegment(t *testing.T) {
	// Exactly 100 chars is allowed.
	maxLenOwner := strings.Repeat("a", repoSegmentMaxLen)
	got, err := WorkspacePath(1, maxLenOwner, "r")
	if err != nil {
		t.Fatalf("WorkspacePath accepted 100-char owner, got error: %v", err)
	}
	if !strings.HasPrefix(got, "/data/workspaces/1/"+maxLenOwner+"/") {
		t.Errorf("WorkspacePath did not include the 100-char owner: %q", got)
	}
}

// itoa is a small helper for t.Run names (which need a string).
func itoa(n int64) string {
	return strconv.FormatInt(n, 10)
}
