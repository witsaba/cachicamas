// Package httpiface — workspace_github_adapter.go adapts the
// concrete *github.Client (which takes the OAuth token as a method
// argument) to the application.GitHubAccessor interface (which only
// takes the GitHub repo id and reads the token from the request
// context via tokenctx).
//
// Why this lives in interfaces/http and not in infrastructure/github:
// the application.GitHubAccessor port is hexagonal — the application
// layer must not depend on the HTTP layer. But the token lives in the
// request context, which is constructed by the HTTP middleware. The
// adapter is therefore the boundary: it knows about both
// tokenctx (infrastructure-agnostic) and *github.Client (which it
// adapts).
package httpiface

import (
	"context"

	"github.com/cachicamas/backend/database_administrator/src/infrastructure/github"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http/tokenctx"
)

// WorkspaceGitHubAccessor wraps a *github.Client to satisfy
// application.GitHubAccessor. It reads the OAuth access token from the
// request context (populated by the auth middleware) and forwards the
// call to the client.
type WorkspaceGitHubAccessor struct {
	client *github.Client
}

// NewWorkspaceGitHubAccessor constructs the adapter.
func NewWorkspaceGitHubAccessor(client *github.Client) *WorkspaceGitHubAccessor {
	return &WorkspaceGitHubAccessor{client: client}
}

// IsRepoAccessible reads the token from ctx and delegates to
// client.IsRepoAccessible(ctx, token, githubID).
//
// Returns (false, *domain.GitHubNotConnectedError-shaped error) when
// the token is absent — but the application layer's
// requireGitHubToken() pre-flight short-circuits the call before it
// ever reaches here, so in practice this branch is defensive only.
func (a *WorkspaceGitHubAccessor) IsRepoAccessible(ctx context.Context, githubID int64) (bool, error) {
	token, ok := tokenctx.GitHubTokenFromContext(ctx)
	if !ok || token == "" {
		return false, errNoGitHubToken
	}
	return a.client.IsRepoAccessible(ctx, token, githubID)
}

// errNoGitHubToken is a sentinel returned when the request context
// has no OAuth access token. In practice the application layer's
// pre-flight catches this before IsRepoAccessible is called, so this
// branch only fires on direct unit tests of the adapter.
var errNoGitHubToken = noGitHubTokenError{}

type noGitHubTokenError struct{}

func (noGitHubTokenError) Error() string {
	return "no GitHub OAuth access token in request context"
}