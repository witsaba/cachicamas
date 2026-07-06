// Package domain contains the core business types of the database
// administrator service. This file defines the workspace aggregate:
// Workspace, PrimaryRepository, LinkedRepository, WorkspaceRepository
// (the hexagonal port), CreateWorkspaceInput, validation, locked
// error vocabulary, and typed errors.
//
// Per design §4, the handler maps each error's Code() to the locked
// HTTP envelope; the application layer propagates errors as-is (do not
// wrap further — the handler uses errors.As).
//
// See src/domain/workspace_test.go for the locked validation rules + error
// code contract (TDD discipline: the test file landed before this one).
package domain

import (
	"context"
	"time"
)

// Workspace is the domain entity for one row of the `workspace`
// table (DDL: migration/sql/20260706120002_workspaces.sql). The struct
// mirrors the DDL column-for-column; the `db` tags tell the pgx adapter
// which column each field reads from, the `json` tags control wire
// serialization.
//
// `OwnerUserID` is a *int64 with NO `omitempty` because the locked
// contract is that a soft-NULL (user deleted → SET NULL) is a distinct
// state from "owner unknown at create time". See workspace_test.go for
// the error-code contract that depends on this distinction.
//
// `DeletedAt` is a *time.Time that lets callers tell live rows from
// soft-deleted rows. The partial unique index
// `workspace_org_name_live_key` keeps live-row name uniqueness cheap;
// the application layer is responsible for filtering
// `WHERE deleted_at IS NULL` on every list/get call.
type Workspace struct {
	ID                  int64      `db:"id"                    json:"id"`
	OrganizationID      int64      `db:"organization_id"       json:"organization_id"`
	OwnerUserID         *int64     `db:"owner_user_id"         json:"owner_user_id"`
	Name                string     `db:"name"                  json:"name"`
	PrimaryRepoGitHubID int64      `db:"primary_repo_github_id" json:"primary_repo_github_id"`
	PrimaryRepoFullName string     `db:"primary_repo_full_name" json:"primary_repo_full_name"`
	PrimaryRepoOwner    string     `db:"primary_repo_owner"    json:"primary_repo_owner"`
	PrimaryRepoName     string     `db:"primary_repo_name"     json:"primary_repo_name"`
	CreatedAt           time.Time  `db:"created_at"            json:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"            json:"updated_at"`
	DeletedAt           *time.Time `db:"deleted_at"           json:"deleted_at"`
}

// PrimaryRepository captures the four GitHub fields that identify the
// workspace's primary repo. Stored as 4 columns on the workspace row
// (rather than a JSONB blob) so a future composite index on
// (primary_repo_github_id) stays cheap and the FK-less natural key is
// available to client queries via simple SELECT/WHERE.
//
// `Owner` and `Name` are denormalized out of `FullName` (which equals
// "owner/name") for the same reason: simpler WHERE clauses, simpler
// UPDATEs, no string-splitting in the Go layer.
type PrimaryRepository struct {
	GitHubID int64  `json:"github_id"`
	FullName string `json:"full_name"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
}

// LinkedRepository captures one row of `workspace_repository`. The
// same denormalization decision applies (4 GitHub fields, no JSONB).
// The primary repo is NOT stored here — it lives on `workspace` so a
// workspace can have at most one primary and N linked, with no
// duplicate-key confusion.
type LinkedRepository struct {
	ID          int64     `db:"id"               json:"id"`
	WorkspaceID int64     `db:"workspace_id"     json:"workspace_id"`
	GitHubID    int64     `db:"github_id"        json:"github_id"`
	FullName    string    `db:"github_full_name" json:"github_full_name"`
	Owner       string    `db:"github_owner"     json:"github_owner"`
	Name        string    `db:"github_name"      json:"github_name"`
	AddedAt     time.Time `db:"added_at"         json:"added_at"`
}

// CreateWorkspaceInput is the handler-facing payload for the POST
// /workspaces use case. The handler decodes JSON or form-encoded bodies
// into this shape, then hands it to
// application.WorkspaceService.Create. The validation rules below are
// the single source of truth on the server side; the frontend Zod
// schema (PR2-iii) mirrors them byte-for-byte.
//
// `OrganizationID` and `OwnerUserID` are filled in by the handler from
// the session — the wire body MUST NOT carry them. The handler is the
// trust boundary that ensures every workspace is scoped to the
// install's single organization and attributed to the signed-in user.
type CreateWorkspaceInput struct {
	OrganizationID int64
	OwnerUserID    *int64
	Name           string
	PrimaryRepo    PrimaryRepository
}

// UpdateWorkspaceInput is the handler-facing payload for PATCH
// /workspaces/:id. Per design T9, primary_repository is NOT
// updateable — the primary repo is the workspace's identity. The
// service silently drops the field even if the caller sends it.
type UpdateWorkspaceInput struct {
	Name *string // nil = do not change
}

// AddRepositoryInput is the handler-facing payload for POST
// /workspaces/:id/repositories. The GitHub id + denormalized fields
// travel together so the repo picker UI can round-trip a single
// selection without a second lookup.
type AddRepositoryInput struct {
	GitHubID int64
	FullName string
	Owner    string
	Name     string
}

// WorkspaceRepository is the hexagonal port the application layer uses
// to read and persist Workspace + LinkedRepository aggregates.
// Implementations live under src/infrastructure/postgres/ (PR1b-ii.a).
// The interface is deliberately small so a fake is trivial to write
// in tests (see application/workspace_service_test.go).
//
// Contract:
//   - Insert(ctx, w) MUST set w.ID, w.CreatedAt, w.UpdatedAt from the
//     database's own clock (Postgres DEFAULT now()) and return the
//     populated w. The application layer never sets timestamps itself.
//   - SelectByID(ctx, id) MUST filter `WHERE deleted_at IS NULL` so
//     soft-deleted rows return *NotFoundError to the caller. The
//     handler maps NotFoundError to HTTP 404.
//   - SelectAllByOrg(ctx, orgID, limit) MUST filter `WHERE
//     deleted_at IS NULL` and order by `created_at DESC, id DESC` for
//     stable pagination. The limit caps the slice at N (caller picks
//     N; the repo MUST NOT silently truncate beyond the cap).
//   - UpdateName(ctx, id, name) MUST filter on live rows only; returns
//     *NotFoundError if no live row matches. The unique violation on
//     (organization_id, name) WHERE deleted_at IS NULL is translated to
//     *ConflictError so the handler can map to HTTP 409.
//   - SoftDelete(ctx, id) MUST set `deleted_at = now()` AND hard-delete
//     every row in `workspace_repository WHERE workspace_id = id`.
//     Both SQL statements MUST run inside one transaction so a
//     crash mid-delete cannot leave linked repos dangling. Returns
//     *NotFoundError if no live row matches.
//   - AddLinkedRepo(ctx, repo) MUST set repo.ID + repo.AddedAt from the
//     database's own clock. The unique violation on
//     (workspace_id, github_id) is translated to *ConflictError. FK
//     violations (workspace_id does not exist) are translated to
//     *InternalError so the handler can map to HTTP 500.
//   - RemoveLinkedRepo(ctx, workspaceID, repoID) MUST return
//     *NotFoundError if 0 rows affected.
//   - SelectLinkedRepos(ctx, workspaceID) returns rows ordered by
//     `added_at ASC, id ASC` (chronological). Returns an empty slice
//     (NOT nil) when zero rows exist.
//   - All methods honour the context.
type WorkspaceRepository interface {
	Insert(ctx context.Context, w *Workspace) (*Workspace, error)
	SelectByID(ctx context.Context, id int64) (*Workspace, error)
	SelectAllByOrg(ctx context.Context, orgID int64, limit int) ([]Workspace, error)
	UpdateName(ctx context.Context, id int64, name string) (*Workspace, error)
	SoftDelete(ctx context.Context, id int64) error
	AddLinkedRepo(ctx context.Context, repo *LinkedRepository) (*LinkedRepository, error)
	RemoveLinkedRepo(ctx context.Context, workspaceID, repoID int64) error
	SelectLinkedRepos(ctx context.Context, workspaceID int64) ([]LinkedRepository, error)
}

// ---------------------------------------------------------------------------
// Locked error message vocabulary (spec §4). Single source of truth
// for every error response. A typo in one of these constants will
// fail a test; do not duplicate these strings in the handler.
// ---------------------------------------------------------------------------

const (
	// MsgPrimaryRepoRequired is the field-level error message for a
	// missing or incomplete primary_repository payload.
	MsgPrimaryRepoRequired = "Primary repository is required (github_id, full_name, owner, name)."
	// MsgRepoNotAccessible is the field-level error message returned
	// when the user's OAuth token does not grant access to the selected
	// GitHub repository (see spec R-WS-006 / design T7). The frontend
	// shows this in the repo picker so the user can pick a different
	// repo.
	MsgRepoNotAccessible = "Selected repository is not accessible."
	// MsgWorkspaceConflict is the field-level error message returned when
	// the workspace name collides with an existing live workspace.
	MsgWorkspaceConflict = "A workspace with this name already exists."
	// MsgWorkspaceNotFound is the message shown when a workspace does
	// not exist (or is soft-deleted — callers see the same 404 either
	// way).
	MsgWorkspaceNotFound = "Workspace not found."

	// CodeGitHubNotConnected is the locked vocabulary for the case
	// where the user's OAuth roundtrip did not produce a non-NULL
	// access_token (signed in before PR1a, or scope was not granted).
	// The frontend renders the "Reconnect GitHub" banner when it sees
	// this code on any response (R-WS-017).
	//
	// Distinct from CodeServer so the handler can route to a 401 with
	// the reconnect message instead of a generic 500.
	CodeGitHubNotConnected = "github_not_connected"
)

// ---------------------------------------------------------------------------
// Validation regex / length caps (spec §2.3 for workspaces).
// ---------------------------------------------------------------------------

const (
	workspaceNameMinLen = 3
	workspaceNameMaxLen = 60
)

// ValidateCreateWorkspace runs every rule from spec R-WS-001 against
// the input and returns a *ValidationError with the failing fields
// populated when at least one rule fails. When every rule passes,
// Validate returns nil. The function is pure: no I/O, no clock, no
// logging.
//
// On failure, the returned *ValidationError's Fields map contains ONE
// entry per failing field. The handler reads Fields[fc] to render the
// field-level error envelope. Order of rules does not matter — the
// consumer iterates the map.
//
// Validation rules (locked by workspace_test.go):
//   - name: required, 3..60 chars inclusive
//   - primary_repository: required (all 4 fields non-empty, github_id != 0)
//
// Note: this function does NOT verify that the primary repo is in the
// user's accessible set. That check is the application layer's job
// (workspace_service.go Create) because it requires the GitHub REST
// call + cached token lookup — both out of scope for a pure validator.
func ValidateCreateWorkspace(in CreateWorkspaceInput) error {
	fields := map[string]string{}

	name := trim(in.Name)
	if name == "" {
		fields["name"] = MsgNameRequired
	} else if len(name) < workspaceNameMinLen || len(name) > workspaceNameMaxLen {
		fields["name"] = MsgNameLength
	}

	if in.PrimaryRepo.GitHubID == 0 ||
		in.PrimaryRepo.FullName == "" ||
		in.PrimaryRepo.Owner == "" ||
		in.PrimaryRepo.Name == "" {
		fields["primary_repository"] = MsgPrimaryRepoRequired
	}

	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: fields}
}

// GitHubNotConnectedError signals that the user's OAuth roundtrip did
// not produce a non-NULL access_token. The handler maps this to a 401
// with `error: "github_not_connected"` + the reconnect message so the
// frontend can render the "Reconnect GitHub" banner (R-WS-017).
//
// Distinct from InternalError so the handler does not return 500 for a
// recoverable user-state condition. The Code() value is locked
// (`github_not_connected`) — see CodeGitHubNotConnected.
type GitHubNotConnectedError struct{}

func (e *GitHubNotConnectedError) Error() string {
	return "github not connected: user has no access_token; reconnect required"
}

// Code returns CodeGitHubNotConnected (locked vocabulary, distinct
// from CodeServer so the handler can route to a 401).
func (e *GitHubNotConnectedError) Code() string { return CodeGitHubNotConnected }
