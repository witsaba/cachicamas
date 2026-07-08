// Package domain contains the core business types of the database
// administrator service. This file defines the workspace aggregate
// for the 1:1 model (post-2026-07-08-workspaces-simplify):
//
//   - Workspace           (one row of the workspace table)
//   - Repository          (the four GitHub fields that identify the
//                          workspace's only repo; previously called
//                          PrimaryRepository when the model was 1:many)
//   - WorkspaceRepository (the hexagonal port the application layer
//                          uses to read and persist Workspace rows)
//   - CreateWorkspaceInput, UpdateWorkspaceInput
//   - validation, locked error vocabulary, typed errors.
//
// Per design §4, the handler maps each error's Code() to the locked
// HTTP envelope; the application layer propagates errors as-is (do not
// wrap further — the handler uses errors.As).
//
// See src/domain/workspace_test.go for the locked validation rules +
// error code contract (TDD discipline: the test file landed before
// this one).
//
// 2026-07-08-workspaces-simplify changelog:
//   - dropped LinkedRepository (the workspace_repository table no
//     longer exists)
//   - dropped AddLinkedRepo / RemoveLinkedRepo / SelectLinkedRepos
//     from the WorkspaceRepository port
//   - renamed PrimaryRepository -> Repository (no secondary exists
//     in the 1:1 model; the "primary_" prefix was misleading)
//   - renamed PrimaryRepo* fields on Workspace to Repo* (same
//     rationale; the DB columns were renamed in the same migration
//     wave)
//   - renamed CreateWorkspaceInput.PrimaryRepo -> Repository
//   - renamed MsgPrimaryRepoRequired -> MsgRepoRequired
package domain

import (
	"context"
	"time"
)

// Workspace is the domain entity for one row of the `workspace`
// table (DDL: migration/sql/20260706120002_workspaces.sql, plus the
// column renames in migration/sql/20260708120000_drop_workspace_repository.sql).
// The struct mirrors the DDL column-for-column; the `db` tags tell
// the pgx adapter which column each field reads from, the `json`
// tags control wire serialization.
//
// `OwnerUserID` is a *int64 with NO `omitempty` because the locked
// contract is that a soft-NULL (user deleted -> SET NULL) is a
// distinct state from "owner unknown at create time". See
// workspace_test.go for the error-code contract that depends on this
// distinction.
//
// `DeletedAt` is a *time.Time that lets callers tell live rows from
// soft-deleted rows. The partial unique index
// `workspace_org_name_live_key` keeps live-row name uniqueness
// cheap; the application layer is responsible for filtering
// `WHERE deleted_at IS NULL` on every list/get call.
type Workspace struct {
	ID             int64      `db:"id"             json:"id"`
	OrganizationID int64      `db:"organization_id" json:"organization_id"`
	OwnerUserID    *int64     `db:"owner_user_id"  json:"owner_user_id"`
	Name           string     `db:"name"           json:"name"`
	RepoGitHubID   int64      `db:"repo_github_id"   json:"repo_github_id"`
	RepoFullName   string     `db:"repo_full_name"   json:"repo_full_name"`
	RepoOwner      string     `db:"repo_owner"       json:"repo_owner"`
	RepoName       string     `db:"repo_name"        json:"repo_name"`
	CreatedAt      time.Time  `db:"created_at"     json:"created_at"`
	UpdatedAt      time.Time  `db:"updated_at"     json:"updated_at"`
	DeletedAt      *time.Time `db:"deleted_at"    json:"deleted_at"`

	// 2026-07-08-workspace-sync-clone (PR-1 migration
	// 20260708120200_sync_job.sql). All four fields are NULL until the
	// first sync attempt resolves. They are denormalized from the latest
	// sync_job row on the callback; the sync_job table remains the
	// single source of truth.
	LastSyncedAt        *time.Time `db:"last_synced_at"        json:"last_synced_at"`
	LastSyncedCommitSHA *string    `db:"last_synced_commit_sha" json:"last_synced_commit_sha"`
	DefaultBranch       *string    `db:"default_branch"         json:"default_branch"`
	LastSyncJobID       *int64     `db:"last_sync_job_id"       json:"last_sync_job_id"`
}

// Repository captures the four GitHub fields that identify the
// workspace's only repo. Stored as 4 columns on the workspace row
// (rather than a JSONB blob) so a future composite index on
// (repo_github_id) stays cheap and the FK-less natural key is
// available to client queries via simple SELECT/WHERE.
//
// `Owner` and `Name` are denormalized out of `FullName` (which
// equals "owner/name") for the same reason: simpler WHERE clauses,
// simpler UPDATEs, no string-splitting in the Go layer.
//
// 2026-07-08-workspaces-simplify: renamed from PrimaryRepository.
// In the 1:1 model there is no longer a contrast with a "secondary"
// repo, so the `Primary` prefix is misleading.
type Repository struct {
	GitHubID int64  `json:"github_id"`
	FullName string `json:"full_name"`
	Owner    string `json:"owner"`
	Name     string `json:"name"`
}

// CreateWorkspaceInput is the handler-facing payload for the POST
// /workspaces use case. The handler decodes JSON or form-encoded
// bodies into this shape, then hands it to
// application.WorkspaceService.Create. The validation rules below
// are the single source of truth on the server side; the frontend
// Zod schema (PR2-iii) mirrors them byte-for-byte.
//
// `OrganizationID` and `OwnerUserID` are filled in by the handler
// from the session — the wire body MUST NOT carry them. The handler
// is the trust boundary that ensures every workspace is scoped to
// the install's single organization and attributed to the signed-in
// user.
//
// 2026-07-08-workspaces-simplify: PrimaryRepo renamed to
// Repository; same semantics.
type CreateWorkspaceInput struct {
	OrganizationID int64
	OwnerUserID    *int64
	Name           string
	Repository     Repository
}

// UpdateWorkspaceInput is the handler-facing payload for PATCH
// /workspaces/:id. Per design T9, the repository is NOT updateable —
// the repo is the workspace's identity. The service silently drops
// the field even if the caller sends it.
type UpdateWorkspaceInput struct {
	Name *string // nil = do not change
}

// WorkspaceRepository is the hexagonal port the application layer
// uses to read and persist Workspace rows. Implementations live
// under src/infrastructure/postgres/ (PR1b-ii.a). The interface is
// deliberately small so a fake is trivial to write in tests (see
// application/workspace_service_test.go).
//
// Contract:
//   - Insert(ctx, w) MUST set w.ID, w.CreatedAt, w.UpdatedAt from the
//     database's own clock (Postgres DEFAULT now()) and return the
//     populated w. The application layer never sets timestamps itself.
//   - SelectByID(ctx, id) MUST filter `WHERE deleted_at IS NULL` so
//     soft-deleted rows return *NotFoundError to the caller. The
//     handler maps NotFoundError to HTTP 404.
//   - SelectAllByOrg(ctx, orgID, limit) MUST filter `WHERE
//     deleted_at IS NULL` and order by `created_at DESC, id DESC`
//     for stable pagination. The limit caps the slice at N (caller
//     picks N; the repo MUST NOT silently truncate beyond the cap).
//   - UpdateName(ctx, id, name) MUST filter on live rows only;
//     returns *NotFoundError if no live row matches. The unique
//     violation on (organization_id, name) WHERE deleted_at IS NULL
//     is translated to *ConflictError so the handler can map to
//     HTTP 409.
//   - SoftDelete(ctx, id) MUST set `deleted_at = now()`. Returns
//     *NotFoundError if no live row matches. (No linked-repos
//     cleanup needed in the 1:1 model — there is no
//     workspace_repository table.)
//   - All methods honour the context.
//
// 2026-07-08-workspaces-simplify: dropped AddLinkedRepo /
// RemoveLinkedRepo / SelectLinkedRepos because the
// workspace_repository table no longer exists.
type WorkspaceRepository interface {
	Insert(ctx context.Context, w *Workspace) (*Workspace, error)
	SelectByID(ctx context.Context, id int64) (*Workspace, error)
	SelectAllByOrg(ctx context.Context, orgID int64, limit int) ([]Workspace, error)
	UpdateName(ctx context.Context, id int64, name string) (*Workspace, error)
	SoftDelete(ctx context.Context, id int64) error

	// 2026-07-08-workspace-sync-clone (PR-3b): denormalize the
	// outcome of a successful sync onto the workspace row so the UI
	// can render the card without a second query. The sync_job row
	// is updated independently (see SyncService.ProcessSyncCallback).
	// The two writes happen in the same Tx.
	MarkSynced(ctx context.Context, id int64, commitSHA, defaultBranch string) error
}

// ---------------------------------------------------------------------------
// Locked error message vocabulary (spec §4). Single source of truth
// for every error response. A typo in one of these constants will
// fail a test; do not duplicate these strings in the handler.
// ---------------------------------------------------------------------------

const (
	// MsgRepoRequired is the field-level error message for a missing
	// or incomplete `repository` payload.
	//
	// 2026-07-08-workspaces-simplify: renamed from
	// MsgPrimaryRepoRequired. The 1:1 model calls the field
	// `repository`, not `primary_repository`.
	MsgRepoRequired = "Repository is required (github_id, full_name, owner, name)."
	// MsgRepoNotAccessible is the field-level error message returned
	// when the user's OAuth token does not grant access to the selected
	// GitHub repository. The frontend shows this in the repo picker so
	// the user can pick a different repo.
	MsgRepoNotAccessible = "Selected repository is not accessible."
	// MsgWorkspaceConflict is the field-level error message returned
	// when the workspace name collides with an existing live workspace.
	MsgWorkspaceConflict = "A workspace with this name already exists."
	// MsgWorkspaceNotFound is the message shown when a workspace does
	// not exist (or is soft-deleted — callers see the same 404 either
	// way).
	MsgWorkspaceNotFound = "Workspace not found."

	// CodeGitHubNotConnected is the locked vocabulary for the case
	// where the user's OAuth roundtrip did not produce a non-NULL
	// access_token (signed in before PR1a, or scope was not granted).
	// The frontend renders the "Reconnect GitHub" banner when it sees
	// this code on any response.
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
// On failure, the returned *ValidationError's Fields map contains
// ONE entry per failing field. The handler reads Fields[fc] to
// render the field-level error envelope. Order of rules does not
// matter — the consumer iterates the map.
//
// Validation rules (locked by workspace_test.go):
//   - name: required, 3..60 chars inclusive
//   - repository: required (all 4 fields non-empty, github_id != 0)
//
// Note: this function does NOT verify that the repo is in the
// user's accessible set. That check is the application layer's job
// (workspace_service.go Create) because it requires the GitHub REST
// call + cached token lookup — both out of scope for a pure
// validator.
func ValidateCreateWorkspace(in CreateWorkspaceInput) error {
	fields := map[string]string{}

	name := trim(in.Name)
	if name == "" {
		fields["name"] = MsgNameRequired
	} else if len(name) < workspaceNameMinLen || len(name) > workspaceNameMaxLen {
		fields["name"] = MsgNameLength
	}

	if in.Repository.GitHubID == 0 ||
		in.Repository.FullName == "" ||
		in.Repository.Owner == "" ||
		in.Repository.Name == "" {
		fields["repository"] = MsgRepoRequired
	}

	if len(fields) == 0 {
		return nil
	}
	return &ValidationError{Fields: fields}
}

// GitHubNotConnectedError signals that the user's OAuth roundtrip
// did not produce a non-NULL access_token. The handler maps this to
// a 401 with `error: "github_not_connected"` + the reconnect message
// so the frontend can render the "Reconnect GitHub" banner.
//
// Distinct from InternalError so the handler does not return 500
// for a recoverable user-state condition. The Code() value is
// locked (`github_not_connected`) — see CodeGitHubNotConnected.
type GitHubNotConnectedError struct{}

func (e *GitHubNotConnectedError) Error() string {
	return "github not connected: user has no access_token; reconnect required"
}

// Code returns CodeGitHubNotConnected (locked vocabulary, distinct
// from CodeServer so the handler can route to a 401).
func (e *GitHubNotConnectedError) Code() string { return CodeGitHubNotConnected }

// WorkspaceNotFoundError is the locked not-found error returned
// by the workspace handlers and the sync handler (when a
// /workspaces/:id/sync GET is called for a workspace that does
// not exist). Distinct from the generic NotFoundError so callers
// can branch on Code() == CodeWorkspaceNotFound.
type WorkspaceNotFoundError struct{}

func (e *WorkspaceNotFoundError) Error() string { return MsgWorkspaceNotFound }
func (e *WorkspaceNotFoundError) Code() string  { return CodeNotFound }

// ---------------------------------------------------------------------------
// 2026-07-08-workspace-sync-clone: locked error codes for the sync flow.
// The Code() returned by each error is the stable contract the frontend
// branch on (work-unit-commits: never change a code once it ships).
// ---------------------------------------------------------------------------

const (
	// CodeSyncAlreadyRunning is the locked vocabulary for a second
	// concurrent POST /workspaces/:id/sync against a workspace that
	// already has a pending or running sync_job (R-WS-019 S-WS-193).
	// The HTTP status is 409.
	CodeSyncAlreadyRunning = "sync_already_running"

	// CodeSyncInsufficientPermissions is the locked vocabulary for a
	// callback or workspace_check that determines the user's OAuth
	// token has permissions.push === false. The HTTP status is 422.
	// The frontend renders the "Reconnect GitHub" banner.
	CodeSyncInsufficientPermissions = "sync_insufficient_permissions"

	// CodeSyncTokenExpired is the locked vocabulary for a callback or
	// workspace_check that determines the OAuth token is expired.
	// The HTTP status is 401.
	CodeSyncTokenExpired = "sync_token_expired"
)

// ErrSyncAlreadyRunning is the typed error returned by
// SyncService.EnqueueSync on a single-flight hit AND by
// /workspaces/:id/sync when a second concurrent request arrives.
// The Code() is CodeSyncAlreadyRunning.
type ErrSyncAlreadyRunning struct {
	JobID int64
}

func (e *ErrSyncAlreadyRunning) Error() string {
	return "sync already running for this workspace"
}
func (e *ErrSyncAlreadyRunning) Code() string { return CodeSyncAlreadyRunning }

// ErrInsufficientPermissions is the typed error returned by the
// permission-validation use case when the OAuth token has
// permissions.push === false. The Code() is
// CodeSyncInsufficientPermissions.
type ErrInsufficientPermissions struct {
	GitHubID int64
}

func (e *ErrInsufficientPermissions) Error() string {
	return "github token has insufficient permissions (push) for this repository"
}
func (e *ErrInsufficientPermissions) Code() string { return CodeSyncInsufficientPermissions }

// ErrTokenExpired is the typed error returned when the OAuth
// token is past its expires_at. The Code() is CodeSyncTokenExpired.
type ErrTokenExpired struct{}

func (e *ErrTokenExpired) Error() string {
	return "github access token is expired; reconnect required"
}
func (e *ErrTokenExpired) Code() string { return CodeSyncTokenExpired }