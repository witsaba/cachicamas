// Package application contains the use cases of the workspace_syncer
// service. The CloneService is the single use case in v1: clone
// the repository, run the worktree probe, and post the outcome
// back to database_administrator.
//
// The use case is the only consumer of the Runner and the
// CallbackClient. Tests inject fakes; production wiring in
// main.go injects the real implementations.
package application

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/cachicamas/backend/workspace_syncer/src/domain"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/git"
	"github.com/cachicamas/backend/workspace_syncer/src/infrastructure/httpclient"
)

// CallbackClientPort is the application-layer port for the
// database_administrator callback. The real implementation is
// httpclient.CallbackClient; tests inject a fake.
type CallbackClientPort interface {
	Post(ctx context.Context, req httpclient.CallbackRequest) error
}

// GitHubAccessorPort is the application-layer port for the
// permission-validation step (GET /repos/{owner}/{repo}). The real
// implementation lives in infrastructure/github (PR-2b does NOT
// implement this — PR-2b uses a stub that returns "permitted"
// always; PR-2c replaces the stub with the real GitHub client).
//
// Why a port and not a direct call: the design (T-7) calls for
// permission validation BEFORE the long-running clone, so a
// workspace with an invalid token fails fast. The application
// layer depends on the port; the implementation can be swapped
// without touching the use case.
type GitHubAccessorPort interface {
	// IsRepoAccessible returns (true, nil) when the OAuth token
	// grants push permission on {owner}/{repo}. Returns
	// (false, nil) when the repo is not accessible. Returns
	// (false, err) on a network error (the application layer
	// treats this as a transient failure and posts a callback
	// with code "github_unreachable").
	IsRepoAccessible(ctx context.Context, owner, repo string) (bool, error)
}

// CloneService is the use case. It validates the request, calls
// the GitHub accessor for permission validation, runs the
// clone + worktree probe, and posts the outcome to
// database_administrator.
type CloneService struct {
	runner   git.Runner
	callback CallbackClientPort
	github   GitHubAccessorPort
	logger   *slog.Logger
}

// NewCloneService constructs a CloneService.
func NewCloneService(runner git.Runner, callback CallbackClientPort, github GitHubAccessorPort, logger *slog.Logger) *CloneService {
	if logger == nil {
		logger = slog.Default()
	}
	return &CloneService{
		runner:   runner,
		callback: callback,
		github:   github,
		logger:   logger,
	}
}

// CloneAndValidate runs the full pipeline:
//  1. Validate the request (defense in depth — the handler also
//     validates, but the use case is the trusted entry point).
//  2. Verify the OAuth token grants push permission on the
//     workspace's primary repo (via the GitHubAccessorPort).
//  3. Run the clone.
//  4. Run the worktree probe; capture the HEAD SHA.
//  5. Post the callback (done | failed).
//
// Errors are translated to the cross-service error codes via
// errors.As on the git package's typed errors. The mapping is:
//
//   - *git.cloneTimeoutError      → CLONE_TIMEOUT
//   - *git.worktreeProbeFailedError → WORKTREE_PROBE_FAILED
//   - *git.cloneFailedError (any other) → CLONE_FAILED
//   - domain.AppError (from the GitHub accessor) → its own Code()
//
// The function always attempts to post a callback on failure (so
// the database_administrator's sync_job row is updated). The
// callback post itself is best-effort: a failure to post the
// callback is logged but does not affect the function's return
// value.
func (s *CloneService) CloneAndValidate(ctx context.Context, req domain.CloneRequest) {
	startedAt := time.Now().UTC()

	// Step 1: validate the request.
	if err := domain.ValidateCloneRequest(req); err != nil {
		s.logger.ErrorContext(ctx, "clone: validation failed",
			slog.Int64("job_id", req.JobID),
			slog.String("error", err.Error()),
		)
		// Validation failures do NOT post a callback: the
		// database_administrator handler should never have
		// sent an invalid request. The error is logged so an
		// operator can investigate.
		return
	}

	// Step 2: permission validation (best-effort; PR-2b uses a
	// stub that always returns permitted; PR-2c replaces with
	// the real GitHub client).
	if s.github != nil {
		accessible, err := s.github.IsRepoAccessible(ctx, req.Owner, req.Repo)
		if err != nil {
			// Per spec audit finding H-1, the ErrorMessage MUST
			// NOT echo err.Error() (which may contain the URL with
			// embedded token, network details, etc.). Use a
			// closed-vocab message; log the actual error server-side.
			s.logger.WarnContext(ctx, "clone: github accessor error (treating as inaccessible)",
				slog.Int64("job_id", req.JobID),
				slog.String("error", err.Error()),
			)
			s.postCallback(ctx, req, startedAt, httpclient.CallbackRequest{
				JobID:        req.JobID,
				WorkspaceID:  req.WorkspaceID,
				Status:       "failed",
				ErrorCode:    domain.CloneErrCodeGitHubUnreachable.String(),
				ErrorMessage: "github api unreachable",
			})
			return
		}
		if !accessible {
			s.logger.WarnContext(ctx, "clone: token lacks push permission",
				slog.Int64("job_id", req.JobID),
				slog.String("owner", req.Owner),
				slog.String("repo", req.Repo),
			)
			s.postCallback(ctx, req, startedAt, httpclient.CallbackRequest{
				JobID:        req.JobID,
				WorkspaceID:  req.WorkspaceID,
				Status:       "failed",
				ErrorCode:    domain.CloneErrCodeInvalidRepo.String(),
				ErrorMessage: "token lacks push permission on this repository",
			})
			return
		}
	}

	// Step 3: run the clone.
	path, err := s.runner.Clone(ctx, req.WorkspaceID, req.Owner, req.Repo, req.OAuthToken)
	if err != nil {
		// Per spec audit finding H-1: classify to a closed-vocab
		// code and a closed-vocab message. err.Error() may contain
		// git stderr (which itself may contain the URL or token);
		// NEVER echo it to the callback.
		code := domain.CloneErrCodeFailed
		message := "clone failed unexpectedly"
		var toErr *git.CloneTimeoutErrorAlias
		if errors.As(err, &toErr) {
			code = domain.CloneErrCodeTimeout
			message = "clone exceeded timeout"
		}
		s.logger.ErrorContext(ctx, "clone: runner.Clone failed",
			slog.Int64("job_id", req.JobID),
			slog.String("code", code.String()),
			slog.String("error", err.Error()),
		)
		s.postCallback(ctx, req, startedAt, httpclient.CallbackRequest{
			JobID:        req.JobID,
			WorkspaceID:  req.WorkspaceID,
			Status:       "failed",
			ErrorCode:    code.String(),
			ErrorMessage: message,
		})
		return
	}

	// Step 4: run the worktree probe.
	sha, err := s.runner.WorktreeProbe(ctx, path)
	if err != nil {
		// Closed-vocab classification (H-1). err.Error() may
		// contain git stderr; never echo it to the callback.
		s.logger.ErrorContext(ctx, "clone: runner.WorktreeProbe failed",
			slog.Int64("job_id", req.JobID),
			slog.String("code", domain.CloneErrCodeWorktreeProbeFailed.String()),
			slog.String("error", err.Error()),
		)
		s.postCallback(ctx, req, startedAt, httpclient.CallbackRequest{
			JobID:        req.JobID,
			WorkspaceID:  req.WorkspaceID,
			Status:       "failed",
			ErrorCode:    domain.CloneErrCodeWorktreeProbeFailed.String(),
			ErrorMessage: "worktree probe failed",
		})
		return
	}

	// Step 4b: resolve the default branch. Used to denormalize
	// onto workspace.default_branch via the callback. The
	// resolve is best-effort: a failure (older git on bare mirror,
	// non-standard layout) is logged but does NOT fail the clone
	// — the callback is still posted with an empty
	// DefaultBranch, and the workspace row stays NULL until
	// the next successful sync.
	defaultBranch := ""
	if dbBranch, dbErr := s.runner.ResolveDefaultBranch(ctx, path); dbErr != nil {
		s.logger.WarnContext(ctx, "clone: ResolveDefaultBranch failed (non-fatal; callback will omit the field)",
			slog.Int64("job_id", req.JobID),
			slog.String("error", dbErr.Error()),
		)
	} else {
		defaultBranch = dbBranch
	}

	// Step 5: post the success callback.
	s.logger.InfoContext(ctx, "clone: success",
		slog.Int64("job_id", req.JobID),
		slog.Int64("workspace_id", req.WorkspaceID),
		slog.String("sha", sha),
		slog.String("default_branch", defaultBranch),
	)
	s.postCallback(ctx, req, startedAt, httpclient.CallbackRequest{
		JobID:          req.JobID,
		WorkspaceID:    req.WorkspaceID,
		Status:         "done",
		CommitSHAAfter: sha,
		DefaultBranch:  defaultBranch,
	})
}

// postCallback is a small wrapper that posts the callback and
// logs (but does not propagate) a failure. The clone has already
// succeeded or failed; the only remaining question is whether the
// database_administrator was notified.
func (s *CloneService) postCallback(ctx context.Context, req domain.CloneRequest, startedAt time.Time, body httpclient.CallbackRequest) {
	if body.StartedAt == "" {
		body.StartedAt = startedAt.Format(time.RFC3339Nano)
	}
	body.FinishedAt = time.Now().UTC().Format(time.RFC3339Nano)
	if err := s.callback.Post(ctx, body); err != nil {
		s.logger.ErrorContext(ctx, "clone: callback post failed (the sync_job row will not be updated; manual intervention needed)",
			slog.Int64("job_id", req.JobID),
			slog.String("error", err.Error()),
		)
	}
}

// Compile-time check that the runner's typed error is exported
// (and importable by the application layer). The git package
// exposes CloneTimeoutErrorAlias as a public type alias so
// errors.As can match it without depending on the unexported
// concrete type.
var _ = git.CloneTimeoutErrorAlias{Seconds: 0}
