// Package tokenctx defines the context-key contract for carrying the
// GitHub OAuth access_token from the auth middleware (PR1c-i) to the
// workspace service (PR1b-ii.b).
//
// Why this lives in a tiny standalone subpackage:
//   Putting WithGitHubToken / GitHubTokenFromContext inside interfaces/http
//   would create an import cycle — application/workspace_service.go needs
//   to read the token, but application must not depend on interfaces/http
//   (the http layer is the composition root's neighbour, not the app
//   layer's dependency). A neutral subpackage with no other dependencies
//   lets both layers import it without forming a cycle.
//
// PR1c-i will:
//   1. Move/replace the production middleware loader (DB read of
//      identity.account.access_token + WithGitHubToken injection on
//      every authenticated request).
//   2. Re-export the same `WithGitHubToken` and `GitHubTokenFromContext`
//      functions so callers in application/ keep working unchanged.
//
// The key type is unexported (struct{}) so external packages cannot
// construct or read the context value — only this package can write
// via WithGitHubToken, and only callers that have access to these
// helpers can read via GitHubTokenFromContext.
package tokenctx

import "context"

// githubTokenKey is the unexported context-key type. External packages
// cannot construct a value of this type, so they cannot poison or read
// the token via context.WithValue using a colliding key.
type githubTokenKey struct{}

// WithGitHubToken returns a context carrying the user's GitHub OAuth
// access token. The auth middleware (PR1c-i) calls this once per
// authenticated request after loading the token from identity.account.
//
// Production wiring (PR1c-i):
//
//	token, _ := auth.LoadToken(ctx, userID)
//	ctx = tokenctx.WithGitHubToken(ctx, token)
//	c.SetRequest(c.Request().WithContext(ctx))
//
// Tests inject the token via:
//
//	ctx = tokenctx.WithGitHubToken(context.Background(), "test-token")
func WithGitHubToken(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, githubTokenKey{}, token)
}

// GitHubTokenFromContext returns the GitHub OAuth access token stored in
// the context (if any). The boolean mirrors the (value, ok) shape of
// context.Value consumers; ok=false means either the auth middleware did
// not run (anonymous request) or the user signed in before PR1a and has
// no persisted token.
//
// Workspace service callers MUST treat ok=false OR token=="" as
// *domain.GitHubNotConnectedError so the handler can return 401 with
// the reconnect message (R-WS-017).
func GitHubTokenFromContext(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(githubTokenKey{}).(string)
	return t, ok
}