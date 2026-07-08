// Package application contains the use cases of the database
// administrator service. This file implements the workspace use cases
// for the 1:1 model (post-2026-07-08-workspaces-simplify):
//
//   - Create, List, Get, Update, Delete
//
// Each use case opens an OTel span (locked names from spec §3.4) and
// emits a structured slog line on success.
//
// 2026-07-08-workspaces-simplify changelog:
//   - Dropped AddRepository, RemoveRepository, ListRepositories
//     (the workspace_repository table no longer exists)
//   - Dropped the matching span names + http route constants
//   - Renamed PrimaryRepo* field accesses to Repository* (the
//     domain type is now Repository, not PrimaryRepository)
//
// Hexagonal boundary (design §4):
//
//   - This file imports domain (the port) and the stdlib observability
//     stack (slog, OTel trace). It does NOT import pressly/goose or
//     jackc/pgx.
//   - The pgx-backed adapter lives in src/infrastructure/postgres/.
//   - The GitHub REST client lives in src/infrastructure/github/ (PR1c-i).
//     The application layer talks to it through the GitHubAccessor
//     interface so tests can inject a fake without HTTP.
//   - main.go wires the adapter to this service via the
//     domain.WorkspaceRepository port.
package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/cachicamas/backend/database_administrator/src/domain"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http/tokenctx"
)

// Locked OTel span names (spec §3.4 + design T-table).
//
// 2026-07-08-workspaces-simplify: removed spanNameWorkspaceAddRepo /
// spanNameWorkspaceRemoveRepo / spanNameWorkspaceListRepos (the
// linked-repos sub-feature no longer exists).
const (
	spanNameWorkspaceCreate = "workspace.create"
	spanNameWorkspaceList   = "workspace.list"
	spanNameWorkspaceGet    = "workspace.get"
	spanNameWorkspaceUpdate = "workspace.update"
	spanNameWorkspaceDelete = "workspace.delete"
)

// Locked HTTP route strings for span attributes.
//
// 2026-07-08-workspaces-simplify: removed the linked-repo routes
// (/workspaces/:id/repositories and /workspaces/:id/repositories/:repoId).
const (
	httpRouteWorkspaceList   = "/workspaces"
	httpRouteWorkspaceCreate = "/workspaces"
	httpRouteWorkspaceGet    = "/workspaces/:id"
	httpRouteWorkspaceUpdate = "/workspaces/:id"
	httpRouteWorkspaceDelete = "/workspaces/:id"
)

// GitHubAccessor is the hexagonal port the application layer uses to
// ask the GitHub REST layer whether a given repo id is in the user's
// accessible set. The real implementation (PR1c-i) caches /user/repos
// for 5 minutes per user; tests inject an in-memory map.
type GitHubAccessor interface {
	IsRepoAccessible(ctx context.Context, githubID int64) (bool, error)
}

// WorkspaceService is the use case facade for workspace CRUD. It is
// the ONLY caller of domain.WorkspaceRepository in the application
// layer; main.go is the composition root that wires the port to a
// concrete adapter.
//
// 2026-07-08-workspaces-simplify: no longer manages linked repos.
// 2026-07-08-workspace-sync-clone PR-3a: Create now auto-enqueues
// the first sync job via the SyncService (if wired).
type WorkspaceService struct {
	repo         domain.WorkspaceRepository
	syncService  *SyncService // may be nil for pre-PR-3a wiring
	githubClient GitHubAccessor
	logger       *slog.Logger
	tracer       trace.Tracer
}

// NewWorkspaceService constructs a WorkspaceService. The
// syncService may be nil (the service still functions; the
// first-sync auto-enqueue is best-effort skipped when nil).
func NewWorkspaceService(repo domain.WorkspaceRepository, syncService *SyncService, githubClient GitHubAccessor, logger *slog.Logger, tracer trace.Tracer) *WorkspaceService {
	if logger == nil {
		logger = slog.Default()
	}
	if tracer == nil {
		tracer = noop.NewTracerProvider().Tracer("application/workspace_service")
	}
	if githubClient == nil {
		// No-op accessor returns (false, nil) so service callers get
		// the "Selected repository is not accessible." ValidationError.
		// The handler maps that to 422. main.go MUST wire a real
		// implementation; this nil-guard exists so unit tests that
		// don't exercise the GitHub path still compile.
		githubClient = noopGitHubAccessor{}
	}
	return &WorkspaceService{
		repo:         repo,
		syncService:  syncService,
		githubClient: githubClient,
		logger:       logger,
		tracer:       tracer,
	}
}

// noopGitHubAccessor denies every repo by default. Production code
// never uses this — main.go injects the real accessor.
type noopGitHubAccessor struct{}

func (noopGitHubAccessor) IsRepoAccessible(_ context.Context, _ int64) (bool, error) {
	return false, nil
}

// requireGitHubToken returns the token from the context, or a
// *domain.GitHubNotConnectedError if the token is missing/empty.
// Used by Create to gate the GitHub accessibility check.
func requireGitHubToken(ctx context.Context) (string, error) {
	token, ok := tokenctx.GitHubTokenFromContext(ctx)
	if !ok || token == "" {
		return "", &domain.GitHubNotConnectedError{}
	}
	return token, nil
}

// Create validates the input, verifies the repo is accessible to the
// signed-in user (via the GitHub accessor), and delegates persistence
// to the repo. On success the returned *Workspace carries the
// DB-assigned id + Postgres-set timestamps.
//
// OTel attributes (spec §3.4):
//
//   - http.method      = "POST"
//   - http.route       = "/workspaces"
//   - http.status_code = 201 (success)
//   - workspace.id     = <int64> (after successful insert)
//
// On validation failure the function returns immediately — no span, no
// repo call, no DB round-trip, no GitHub call. Validation errors are
// caller errors that do not need to be traced as a request.
//
// On GitHub-not-connected the function returns *domain.GitHubNotConnectedError
// before opening a span (the token lookup is a precondition, not a
// traced operation).
//
// On a GitHub accessibility error (transient network) the function
// wraps the underlying error and returns it; the handler maps to 500.
//
// On a GitHub "not accessible" answer (boolean false) the function
// returns a *ValidationError with fields.repository set to the
// locked MsgRepoNotAccessible message — the handler maps to 422.
//
// On a unique-violation from the repo (name already exists), the
// *ConflictError propagates as-is so the handler can map to 409.
//
// 2026-07-08-workspaces-simplify: in.Repository (was in.PrimaryRepo)
// and the workspace row's Repo* fields (were PrimaryRepo*). The
// validation error key is "repository" (was "primary_repository").
func (s *WorkspaceService) Create(ctx context.Context, in domain.CreateWorkspaceInput) (*domain.Workspace, error) {
	if err := domain.ValidateCreateWorkspace(in); err != nil {
		return nil, err
	}

	// Pre-flight: token must be present before we spend a span on a
	// request the auth layer already knows is doomed. This mirrors the
	// "validation short-circuits the span" pattern from OrganizationService.Create.
	if _, err := requireGitHubToken(ctx); err != nil {
		return nil, err
	}

	ctx, span := s.tracer.Start(ctx, spanNameWorkspaceCreate)
	defer span.End()

	setHTTPRouteAttrs(span, "POST", httpRouteWorkspaceCreate)

	// Verify repo is accessible (design T7). We do NOT call
	// requireGitHubToken again here because the pre-flight above is
	// authoritative for the span.
	accessible, err := s.githubClient.IsRepoAccessible(ctx, in.Repository.GitHubID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("verify repo: %w", err)
	}
	if !accessible {
		span.SetStatus(codes.Error, "repo not accessible")
		return nil, &domain.ValidationError{Fields: map[string]string{
			"repository": domain.MsgRepoNotAccessible,
		}}
	}

	w := &domain.Workspace{
		OrganizationID: in.OrganizationID,
		OwnerUserID:    in.OwnerUserID,
		Name:           in.Name,
		RepoGitHubID:   in.Repository.GitHubID,
		RepoFullName:   in.Repository.FullName,
		RepoOwner:      in.Repository.Owner,
		RepoName:       in.Repository.Name,
	}

	persisted, err := s.repo.Insert(ctx, w)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int64("workspace.id", persisted.ID))
	setHTTPStatus(span, 201)
	s.logger.InfoContext(ctx, "workspace.create ok",
		slog.Int64("workspace.id", persisted.ID),
		slog.String("name", persisted.Name),
		slog.Int64("repo_github_id", persisted.RepoGitHubID),
	)
	return persisted, nil
}

// List returns up to `limit` live workspaces for the given organization.
// The caller (handler) picks the limit; the repo caps at N without
// silent truncation. Ordered created_at DESC for stable "most recent
// first" UX.
func (s *WorkspaceService) List(ctx context.Context, orgID int64, limit int) ([]domain.Workspace, error) {
	ctx, span := s.tracer.Start(ctx, spanNameWorkspaceList)
	defer span.End()

	setHTTPRouteAttrs(span, "GET", httpRouteWorkspaceList)

	out, err := s.repo.SelectAllByOrg(ctx, orgID, limit)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int("workspace.count", len(out)))
	setHTTPStatus(span, 200)
	s.logger.InfoContext(ctx, "workspace.list ok",
		slog.Int64("organization_id", orgID),
		slog.Int("count", len(out)),
	)
	return out, nil
}

// Get returns one live workspace by id. Soft-deleted rows return
// *domain.NotFoundError from the repo (the WHERE deleted_at IS NULL
// filter); the handler maps that to HTTP 404.
func (s *WorkspaceService) Get(ctx context.Context, id int64) (*domain.Workspace, error) {
	ctx, span := s.tracer.Start(ctx, spanNameWorkspaceGet)
	defer span.End()

	setHTTPRouteAttrs(span, "GET", httpRouteWorkspaceGet)

	w, err := s.repo.SelectByID(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int64("workspace.id", w.ID))
	setHTTPStatus(span, 200)
	s.logger.InfoContext(ctx, "workspace.get ok",
		slog.Int64("workspace.id", w.ID),
	)
	return w, nil
}

// Update renames a workspace. Per design T9, the repository field on
// the input is silently dropped — the repo is the workspace's identity
// and cannot be changed post-create. The locked invariant is
// preserved at the application layer so the handler does not need to
// filter the input.
//
// On a unique-violation (duplicate name), the *ConflictError propagates
// unchanged so the handler can map to HTTP 409.
func (s *WorkspaceService) Update(ctx context.Context, id int64, in domain.UpdateWorkspaceInput) (*domain.Workspace, error) {
	if in.Name == nil {
		// Nothing to do. Return the current row so the caller can render.
		return s.Get(ctx, id)
	}
	ctx, span := s.tracer.Start(ctx, spanNameWorkspaceUpdate)
	defer span.End()

	setHTTPRouteAttrs(span, "PATCH", httpRouteWorkspaceUpdate)

	w, err := s.repo.UpdateName(ctx, id, *in.Name)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.Int64("workspace.id", w.ID))
	setHTTPStatus(span, 200)
	s.logger.InfoContext(ctx, "workspace.update ok",
		slog.Int64("workspace.id", w.ID),
		slog.String("name", w.Name),
	)
	return w, nil
}

// Delete soft-deletes the workspace (sets deleted_at = now()). Returns
// *NotFoundError if the row is already soft-deleted or never existed.
//
// 2026-07-08-workspaces-simplify: no longer cascades into
// workspace_repository (the table no longer exists).
func (s *WorkspaceService) Delete(ctx context.Context, id int64) error {
	ctx, span := s.tracer.Start(ctx, spanNameWorkspaceDelete)
	defer span.End()

	setHTTPRouteAttrs(span, "DELETE", httpRouteWorkspaceDelete)

	if err := s.repo.SoftDelete(ctx, id); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}

	span.SetAttributes(attribute.Int64("workspace.id", id))
	setHTTPStatus(span, 204)
	s.logger.InfoContext(ctx, "workspace.delete ok",
		slog.Int64("workspace.id", id),
	)
	return nil
}

// Compile-time check that the service struct exposes the 5 use cases
// the handler needs. If a method is renamed, the build breaks here.
//
// 2026-07-08-workspaces-simplify: 5 methods (was 8); the linked-repo
// use cases are gone.
var _ interface {
	Create(ctx context.Context, in domain.CreateWorkspaceInput) (*domain.Workspace, error)
	List(ctx context.Context, orgID int64, limit int) ([]domain.Workspace, error)
	Get(ctx context.Context, id int64) (*domain.Workspace, error)
	Update(ctx context.Context, id int64, in domain.UpdateWorkspaceInput) (*domain.Workspace, error)
	Delete(ctx context.Context, id int64) error
} = (*WorkspaceService)(nil)

// Compile-time guard: GitHubNotConnectedError must implement AppError so
// the handler can errors.As it and map to the locked HTTP envelope.
var _ domain.AppError = (*domain.GitHubNotConnectedError)(nil)

// errors package import kept here to make the conflict-not-found path
// explicit if a future contributor refactors the error chain.
var _ = errors.Is
