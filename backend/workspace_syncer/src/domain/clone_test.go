package domain

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateCloneRequest_Valid(t *testing.T) {
	// Reference request that MUST pass every rule. If a future
	// change tightens the rules, update this request so the test
	// continues to reflect the locked contract.
	req := CloneRequest{
		JobID:         42,
		WorkspaceID:   7,
		Owner:         "octocat",
		Repo:          "hello-world",
		DefaultBranch: "main",
		OAuthToken:    "gho_xxx",
	}
	if err := ValidateCloneRequest(req); err != nil {
		t.Fatalf("expected valid request to pass, got: %v", err)
	}
}

func TestValidateCloneRequest_Invalid(t *testing.T) {
	// Table-driven: each row is a single failure mode. Mirrors
	// R-WSY-001 S-WSY-001 in workspace-syncer/spec.md.
	cases := []struct {
		name      string
		req       CloneRequest
		wantField string
	}{
		{
			name: "zero job_id",
			req: CloneRequest{
				JobID: 0, WorkspaceID: 1, Owner: "o", Repo: "r",
				DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "job_id",
		},
		{
			name: "negative job_id",
			req: CloneRequest{
				JobID: -1, WorkspaceID: 1, Owner: "o", Repo: "r",
				DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "job_id",
		},
		{
			name: "zero workspace_id",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 0, Owner: "o", Repo: "r",
				DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "workspace_id",
		},
		{
			name: "empty owner",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 1, Owner: "", Repo: "r",
				DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "owner",
		},
		{
			name: "owner with shell metachar",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 1, Owner: "octo;rm -rf /", Repo: "r",
				DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "owner",
		},
		{
			name: "owner with space",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 1, Owner: "octo cat", Repo: "r",
				DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "owner",
		},
		{
			name: "owner too long",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 1,
				Owner: strings.Repeat("a", repoSegmentMaxLen+1),
				Repo:  "r", DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "owner",
		},
		{
			name: "empty repo",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 1, Owner: "o", Repo: "",
				DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "repo",
		},
		{
			name: "repo with shell metachar",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 1, Owner: "o", Repo: "$(rm)",
				DefaultBranch: "main", OAuthToken: "t",
			},
			wantField: "repo",
		},
		{
			name: "empty default_branch",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 1, Owner: "o", Repo: "r",
				DefaultBranch: "", OAuthToken: "t",
			},
			wantField: "default_branch",
		},
		{
			name: "empty oauth_token",
			req: CloneRequest{
				JobID: 1, WorkspaceID: 1, Owner: "o", Repo: "r",
				DefaultBranch: "main", OAuthToken: "",
			},
			wantField: "oauth_token",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCloneRequest(tc.req)
			if err == nil {
				t.Fatalf("expected validation error on %q, got nil", tc.name)
			}
			ve, ok := IsValidationError(err)
			if !ok {
				t.Fatalf("expected *ValidationError, got %T: %v", err, err)
			}
			if _, found := ve.Fields[tc.wantField]; !found {
				t.Errorf("expected field %q in error, got fields: %v", tc.wantField, ve.Fields)
			}
		})
	}
}

func TestValidateCloneRequest_MultipleFailures(t *testing.T) {
	// When multiple rules fail, the error carries all of them so the
	// handler can render the full set in one envelope.
	req := CloneRequest{} // every field invalid
	err := ValidateCloneRequest(req)
	if err == nil {
		t.Fatalf("expected validation error on empty request, got nil")
	}
	ve, ok := IsValidationError(err)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T: %v", err, err)
	}
	if len(ve.Fields) < 6 {
		t.Errorf("expected at least 6 failing fields, got %d: %v", len(ve.Fields), ve.Fields)
	}
	for _, expected := range []string{"job_id", "workspace_id", "owner", "repo", "default_branch", "oauth_token"} {
		if _, found := ve.Fields[expected]; !found {
			t.Errorf("expected field %q in error, got: %v", expected, ve.Fields)
		}
	}
}

func TestValidationError_ErrorMessage(t *testing.T) {
	ve := &ValidationError{Fields: map[string]string{"owner": "is required"}}
	if msg := ve.Error(); !strings.Contains(msg, "1 field") {
		t.Errorf("Error() = %q, want it to mention field count", msg)
	}
}

func TestValidRepoSegment(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", false},
		{"octocat", true},
		{"hello-world", true},
		{"hello_world", true},
		{"hello.world", true},
		{"hello world", false}, // space
		{"octo;rm", false},      // shell metachar
		{"$(rm)", false},        // command substitution
		{"a/b", false},          // path separator
		{"a\\b", false},         // windows path separator
		{strings.Repeat("a", repoSegmentMaxLen), true},
		{strings.Repeat("a", repoSegmentMaxLen+1), false},
	}
	for _, tc := range cases {
		if got := validRepoSegment(tc.in); got != tc.want {
			t.Errorf("validRepoSegment(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestErrorsAs_ValidationError(t *testing.T) {
	// The handler relies on errors.As; this guards against the
	// contract breaking accidentally.
	ve := &ValidationError{Fields: map[string]string{"x": "y"}}
	var target *ValidationError
	if !errors.As(ve, &target) {
		t.Fatalf("errors.As(ValidationError) failed; handler would not be able to map the error")
	}
}
