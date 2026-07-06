// Package domain contains the core business types of the database
// administrator service. This file covers the workspace aggregate
// (Workspace, PrimaryRepository, LinkedRepository, WorkspaceRepository
// port) and its validation + error contracts.
//
// Strict TDD discipline (per openspec/AGENTS.md and sdd-init/cachicamas):
// every behavior here is described by a failing test FIRST. The file was
// written BEFORE workspace.go existed; running `go test ./src/domain`
// against this file with no workspace.go must fail with
// "undefined: ValidateCreateWorkspace" and
// "undefined: (*ValidationError).Code" — those failures ARE the RED step.
//
// The unit-level table-driven cases below do not need a live DB; they
// exercise the pure validation function and the error type contracts.
// Integration scenarios (INSERT/UPDATE/DELETE against Postgres) live
// in src/infrastructure/postgres/workspace_repo_test.go (PR1b-ii.a) and
// in src/migration/runner_test.go (the schema invariants covered by
// TestRunner_Up_WorkspacesPR1bI_*).
package domain

import (
	"strings"
	"testing"
)

// TestValidateCreateWorkspace_RejectsInvalidInputs covers spec R-WS-001
// S-WS-003 (name required + length rules) + the locked primary-repo
// requirement: every input below MUST return *ValidationError with the
// locked fields map key. The shape is intentionally small (one test,
// many sub-cases) so the RED → GREEN transition is a single function
// implementation in workspace.go.
func TestValidateCreateWorkspace_RejectsInvalidInputs(t *testing.T) {
	cases := []struct {
		name       string
		in         CreateWorkspaceInput
		wantFields []string // each test case asserts every key in this slice is present in Fields
	}{
		{
			name:       "empty name",
			in:         CreateWorkspaceInput{Name: "", PrimaryRepo: PrimaryRepository{GitHubID: 1, FullName: "o/n", Owner: "o", Name: "n"}},
			wantFields: []string{"name"},
		},
		{
			name:       "name too short (2 chars)",
			in:         CreateWorkspaceInput{Name: "ab", PrimaryRepo: PrimaryRepository{GitHubID: 1, FullName: "o/n", Owner: "o", Name: "n"}},
			wantFields: []string{"name"},
		},
		{
			name:       "name too long (61 chars)",
			in:         CreateWorkspaceInput{Name: strings.Repeat("a", 61), PrimaryRepo: PrimaryRepository{GitHubID: 1, FullName: "o/n", Owner: "o", Name: "n"}},
			wantFields: []string{"name"},
		},
		{
			name:       "missing primary repo (GitHubID == 0)",
			in:         CreateWorkspaceInput{Name: "valid-name", PrimaryRepo: PrimaryRepository{GitHubID: 0, FullName: "o/n", Owner: "o", Name: "n"}},
			wantFields: []string{"primary_repository"},
		},
		{
			name:       "primary repo with empty FullName",
			in:         CreateWorkspaceInput{Name: "valid-name", PrimaryRepo: PrimaryRepository{GitHubID: 1, FullName: "", Owner: "o", Name: "n"}},
			wantFields: []string{"primary_repository"},
		},
		{
			name:       "primary repo with empty Owner",
			in:         CreateWorkspaceInput{Name: "valid-name", PrimaryRepo: PrimaryRepository{GitHubID: 1, FullName: "o/n", Owner: "", Name: "n"}},
			wantFields: []string{"primary_repository"},
		},
		{
			name:       "primary repo with empty Name",
			in:         CreateWorkspaceInput{Name: "valid-name", PrimaryRepo: PrimaryRepository{GitHubID: 1, FullName: "o/n", Owner: "o", Name: ""}},
			wantFields: []string{"primary_repository"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCreateWorkspace(tc.in)
			if err == nil {
				t.Fatalf("ValidateCreateWorkspace(%s): expected *ValidationError, got nil", tc.name)
			}
			verr, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("ValidateCreateWorkspace(%s): error type = %T, want *ValidationError", tc.name, err)
			}
			for _, field := range tc.wantFields {
				if _, present := verr.Fields[field]; !present {
					t.Errorf("ValidateCreateWorkspace(%s): missing fields[%q] in %v", tc.name, field, verr.Fields)
				}
			}
		})
	}
}

// TestValidateCreateWorkspace_AcceptsValidInput covers spec R-WS-001
// S-WS-001 + the TRIANGULATE minimum-viable input: a valid input MUST
// return nil and a minimal valid input (just the required fields) MUST
// also return nil. The locked name range (3..60) is exercised here with
// boundary values.
func TestValidateCreateWorkspace_AcceptsValidInput(t *testing.T) {
	cases := []struct {
		name string
		in   CreateWorkspaceInput
	}{
		{
			name: "minimum viable (3-char name, all repo fields set)",
			in: CreateWorkspaceInput{
				Name: "abc",
				PrimaryRepo: PrimaryRepository{
					GitHubID: 12345,
					FullName: "octocat/hello-world",
					Owner:    "octocat",
					Name:     "hello-world",
				},
			},
		},
		{
			name: "maximum name length (60 chars)",
			in: CreateWorkspaceInput{
				Name: strings.Repeat("a", 60),
				PrimaryRepo: PrimaryRepository{
					GitHubID: 1,
					FullName: "o/n",
					Owner:    "o",
					Name:     "n",
				},
			},
		},
		{
			name: "single-char-per-segment repo with hyphenated name",
			in: CreateWorkspaceInput{
				Name: "my-team-2026",
				PrimaryRepo: PrimaryRepository{
					GitHubID: 999999,
					FullName: "my-org/my-team-2026",
					Owner:    "my-org",
					Name:     "my-team-2026",
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCreateWorkspace(tc.in)
			if err != nil {
				t.Errorf("ValidateCreateWorkspace(%s): expected nil, got %v", tc.name, err)
			}
		})
	}
}

// TestWorkspaceErrorCodes_LockedVocabulary covers spec §3.4 of the
// design + the locked error code vocabulary for the workspace handler.
// Each typed error MUST expose a Code() string that matches the locked
// vocabulary. The handler maps Code() to the HTTP envelope, so a wrong
// Code() silently routes the response to the wrong status code.
//
// GitHubNotConnectedError gets its OWN code (`github_not_connected`)
// distinct from CodeServer — the frontend uses this code to render the
// "Reconnect GitHub" banner (R-WS-017).
func TestWorkspaceErrorCodes_LockedVocabulary(t *testing.T) {
	cases := []struct {
		name string
		err  AppError
		want string
	}{
		{"ValidationError", &ValidationError{Fields: map[string]string{"name": "x"}}, CodeValidation},
		{"ConflictError", &ConflictError{}, CodeConflict},
		{"NotFoundError", &NotFoundError{Resource: "workspace"}, CodeNotFound},
		{"GitHubNotConnectedError", &GitHubNotConnectedError{}, "github_not_connected"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.err.Code()
			if got != tc.want {
				t.Errorf("%s.Code() = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

// TestGitHubNotConnectedError_CodeIsDistinctFromCodeServer is a
// dedicated regression for the locked decision (design T7 + proposal
// §"Decisions locked this round"): the workspaces handler MUST be able
// to differentiate "user has no access_token" from "internal server
// failure" at the HTTP boundary so the frontend can render the
// "Reconnect GitHub" banner instead of a generic alert. A regression
// here would silently send every GitHub-not-connected case through
// the 500 path.
func TestGitHubNotConnectedError_CodeIsDistinctFromCodeServer(t *testing.T) {
	ghnc := &GitHubNotConnectedError{}
	if ghnc.Code() == CodeServer {
		t.Errorf("GitHubNotConnectedError.Code() = %q (== CodeServer); must be distinct so the handler can route to a 401 with the reconnect message", ghnc.Code())
	}
}

// Compile-time check: GitHubNotConnectedError MUST satisfy the AppError
// marker interface so the workspace handler can errors.As it and dispatch
// on Code(). If a future contributor removes the Code() method, the build
// breaks here, not at runtime in a 500 envelope.
var _ AppError = (*GitHubNotConnectedError)(nil)
