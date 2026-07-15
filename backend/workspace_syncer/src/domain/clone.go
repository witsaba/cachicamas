// Package domain contains the core business types of the workspace_syncer
// service. The package is intentionally free of any framework, database,
// or filesystem imports — it is the locked contract that the
// application and infrastructure layers adapt to.
//
// See design §4 (Cross-service contract) for the request/response
// shape and the locked error codes that the workspace_syncer maps to
// before posting the callback to database_administrator.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"time"
)

// CloneRequest is the handler-facing payload for the
// POST /internal/clone-and-validate endpoint. database_administrator
// POSTs this body when a workspace is created (auto_on_create) or
// when the user clicks Sync on the workspace detail page (manual).
//
// All fields are required. ValidateCloneRequest is the single source
// of truth on what counts as a valid request; the handler delegates
// to it before opening a goroutine for the long-running clone work.
type CloneRequest struct {
	JobID         int64
	WorkspaceID   int64
	Owner         string
	Repo          string
	DefaultBranch string
	OAuthToken    string
}

// CloneResult is the result of a successful CloneAndValidate. The
// application layer posts this back to database_administrator's
// /internal/sync-callback with status="done".
type CloneResult struct {
	JobID           int64
	CommitSHAAfter  string
	StartedAt       time.Time
	FinishedAt      time.Time
}

// CloneFailure is the result of a failed CloneAndValidate. The
// application layer posts this back to database_administrator's
// /internal/sync-callback with status="failed".
type CloneFailure struct {
	JobID        int64
	ErrorCode    string
	ErrorMessage string
	StartedAt    time.Time
	FinishedAt   time.Time
}

// Validation constants — mirror the locked rules in the explore.md
// §6 (Permissions validation) and the design.md §4 (Filesystem
// layout). Kept in one place so a future contract change is a
// single-file edit.
const (
	// repoSegmentPattern is the locked regex for owner and repo
	// segments. Matches GitHub's own rules: alphanumerics plus
	// `.`, `_`, and `-`. Defense-in-depth against shell injection
	// and path traversal — the git URL is constructed from these
	// values, so any non-matching segment is rejected before the
	// shell-out to `git`.
	repoSegmentPattern = `^[a-zA-Z0-9._-]+$`

	// repoSegmentMaxLen is the max length of a single owner or repo
	// segment. GitHub's own limit is 100 chars; we keep the same
	// limit for defense-in-depth.
	repoSegmentMaxLen = 100
)

// repoSegmentRegexp is the compiled version of repoSegmentPattern.
// Compiled once at package init for cheap reuse in ValidateCloneRequest.
var repoSegmentRegexp = regexp.MustCompile(repoSegmentPattern)

// ValidateCloneRequest runs every rule from the spec against the
// request and returns nil on success or a *ValidationError with the
// failing fields populated when at least one rule fails.
//
// Locked rules (workspace-syncer spec R-WSY-001 S-WSY-001):
//   - workspace_id MUST be a positive int64 (BIGSERIAL, non-zero, non-negative)
//   - job_id MUST be a positive int64
//   - owner MUST match ^[a-zA-Z0-9._-]+$ and be non-empty
//   - repo MUST match ^[a-zA-Z0-9._-]+$ and be non-empty
//   - default_branch MUST be non-empty
//   - oauth_token MUST be non-empty
//
// The function is pure: no I/O, no clock, no logging.
func ValidateCloneRequest(req CloneRequest) error {
	fields := map[string]string{}

	if req.JobID <= 0 {
		fields["job_id"] = "job_id must be a positive int64."
	}
	if req.WorkspaceID <= 0 {
		fields["workspace_id"] = "workspace_id must be a positive int64."
	}
	if !validRepoSegment(req.Owner) {
		fields["owner"] = "owner is required and must match ^[a-zA-Z0-9._-]+$."
	}
	if !validRepoSegment(req.Repo) {
		fields["repo"] = "repo is required and must match ^[a-zA-Z0-9._-]+$."
	}
	if req.DefaultBranch == "" {
		fields["default_branch"] = "default_branch is required."
	}
	if req.OAuthToken == "" {
		fields["oauth_token"] = "oauth_token is required."
	}

	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: fields}
}

// validRepoSegment returns true if s is a non-empty, length-bounded
// alphanumeric + `._-` string. Used by ValidateCloneRequest to guard
// the values that flow into the git URL and the filesystem path.
func validRepoSegment(s string) bool {
	if s == "" || len(s) > repoSegmentMaxLen {
		return false
	}
	return repoSegmentRegexp.MatchString(s)
}

// ---------------------------------------------------------------------------
// ValidationError — the single shape used for every 400 response in
// the workspace_syncer. Mirrors the database_administrator envelope
// (`{"error": "validation", "fields": {...}}`) so the callback
// handler can translate it without a per-field switch.
// ---------------------------------------------------------------------------

// ValidationError is returned by ValidateCloneRequest when one or
// more rules fail. The Fields map contains ONE entry per failing
// field; the handler reads Fields[fc] to render the field-level
// error envelope.
type ValidationError struct {
	Fields map[string]string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed: %d field(s)", len(e.Fields))
}

// IsValidationError is a convenience for the handler. The handler
// uses errors.As to map the error to a 400 envelope.
func IsValidationError(err error) (*ValidationError, bool) {
	var v *ValidationError
	if errors.As(err, &v) {
		return v, true
	}
	return nil, false
}
