// Package httpiface — github_handler.go implements the server-side proxy
// for `GET /github/repos`. The handler reads the user's OAuth
// access_token from the request context (populated by the auth
// middleware in PR1c-i), checks/writes the in-memory cache, and proxies
// to api.github.com when needed.
//
// Security contract (PR1c-i, R-WS-009 + R-WS-010):
//   - The HTTP response body MUST NOT contain the access_token. The
//     `reposResponse` projection is `Repo` only — no token-shaped field.
//   - When the auth middleware provides an empty token (the user signed
//     in before PR1a), the handler returns 401 with code
//     "github_not_connected" — distinct from generic 401s so the
//     frontend can show the "Reconnect GitHub" banner (R-WS-017).
//   - When GitHub returns 401 (token expired / revoked), we return 502
//     "github_unauthorized" — the frontend should sign out + sign in
//     again to refresh.
//   - When GitHub returns 403 (rate-limited), we return 502
//     "github_rate_limited".
package httpiface

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/cachicamas/backend/database_administrator/src/infrastructure/github"
	"github.com/cachicamas/backend/database_administrator/src/interfaces/http/tokenctx"
)

// GitHubHandler wires the GitHub repos proxy endpoint. Its dependencies
// (cache + client + user identity accessor) are passed in by the
// composition root in cmd/server/main.go.
type GitHubHandler struct {
	Cache  *github.RepoCache
	Client *github.Client
	// UserIDFromRequest returns the identity.user.id associated with the
	// authenticated request. The handler uses it as the cache key and as
	// the audit identifier in slog lines. The auth middleware populates
	// the identity in Echo's context; this closure extracts the id.
	UserIDFromRequest func(c *echo.Context) (int64, bool)
	Logger            *slog.Logger
}

// NewGitHubHandler constructs the handler with the required dependencies.
func NewGitHubHandler(
	cache *github.RepoCache,
	client *github.Client,
	userIDFromRequest func(c *echo.Context) (int64, bool),
	logger *slog.Logger,
) *GitHubHandler {
	if cache == nil {
		panic("NewGitHubHandler: cache must not be nil")
	}
	if client == nil {
		panic("NewGitHubHandler: client must not be nil")
	}
	if userIDFromRequest == nil {
		panic("NewGitHubHandler: userIDFromRequest must not be nil")
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &GitHubHandler{
		Cache:             cache,
		Client:            client,
		UserIDFromRequest: userIDFromRequest,
		Logger:            logger,
	}
}

// RegisterGitHubRoutes wires the GitHub-adjacent routes on the given
// Echo instance. Currently a single endpoint: GET /github/repos.
//
// Deprecated: this surface is unsafe. Use
// RegisterAuthenticatedWorkspaceRoutes in workspace_handler.go, which
// mounts /github/repos AND the 8 workspace endpoints behind the auth
// chain as a single compile-time guarantee. Wiring /github/repos
// here without the auth middleware reintroduces the bug fixed in
// commit on feat/2026-07-06-workspaces.
func RegisterGitHubRoutes(e *echo.Echo, h *GitHubHandler) {
	e.GET("/github/repos", h.ListRepos)
}

// reposResponse is the wire shape returned by ListRepos. It is locked:
// the frontend (lib/api.ts `listGitHubRepos`) decodes this exact shape.
// page + perPage + hasNext echo the request fields; the `repositories`
// array is the GitHub projection.
type reposResponse struct {
	Repositories []github.Repo `json:"repositories"`
	Page         int           `json:"page"`
	PerPage      int           `json:"per_page"`
	HasNext      bool          `json:"has_next"`
}

// ListRepos handles GET /github/repos. It honours `?page=N&per_page=N`
// pagination and `?bust_cache=true` cache bypass (S-WS-084).
//
// Response codes:
//
//	200 — success (repos + hasNext hint).
//	401 — no auth context (the auth middleware should not let this reach
//	       here, but defense-in-depth applies).
//	401 with `error=github_not_connected` — user has no access_token
//	       (signed in before PR1a).
//	502 with `error=github_unauthorized` — GitHub returned 401 (token
//	       expired/revoked).
//	502 with `error=github_rate_limited` — GitHub returned 403.
//	502 with `error=github_unreachable` — transport error.
func (h *GitHubHandler) ListRepos(c *echo.Context) error {
	ctx := c.Request().Context()

	userID, ok := h.UserIDFromRequest(c)
	if !ok {
		return writeEnvelope(c, http.StatusUnauthorized,
			"unauthorized", "authentication required", nil)
	}
	userIDKey := userIDString(userID)

	token, tokenOK := tokenctx.GitHubTokenFromContext(ctx)
	if !tokenOK || token == "" {
		// The cached repo data may still be served IF the user has a
		// cached entry from before their token was lost (e.g., token
		// revoked). In practice this is edge-case; for v1 we choose
		// the simpler behaviour: no token → 401 reconnect.
		return writeEnvelope(c, http.StatusUnauthorized,
			"github_not_connected",
			"Reconnect GitHub to list repositories.",
			nil,
		)
	}

	// Pagination + bust cache from query string.
	page := atoiOr(c.QueryParam("page"), 1)
	perPage := atoiOr(c.QueryParam("per_page"), 30)
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30
	}
	if c.QueryParam("bust_cache") == "true" {
		h.Cache.Bust(userIDKey)
	}

	// Cache lookup. The cache holds repos for the user regardless of
	// page index in v1 — workspace creation only checks the first page.
	// If the requested page > 1, we still consult GitHub. If the cache
	// is warm and page == 1, we serve it.
	if page == 1 {
		if cached, hit := h.Cache.Get(userIDKey); hit && len(cached) > 0 {
			return c.JSON(http.StatusOK, reposResponse{
				Repositories: cached,
				Page:         page,
				PerPage:      perPage,
				HasNext:      len(cached) >= perPage,
			})
		}
	}

	// Cache miss or pagination — fetch from GitHub.
	repos, hasNext, err := h.Client.ListUserRepos(ctx, token, page, perPage)
	if err != nil {
		return h.mapUpstreamError(c, err)
	}

	// Populate cache only on page 1 (the cached value IS page 1).
	if page == 1 {
		h.Cache.Set(userIDKey, repos)
	}

	return c.JSON(http.StatusOK, reposResponse{
		Repositories: repos,
		Page:         page,
		PerPage:      perPage,
		HasNext:      hasNext,
	})
}

// mapUpstreamError translates a GitHub client error into the locked HTTP
// envelope. Centralized so the wire shape stays consistent across the
// workspace + proxy paths.
func (h *GitHubHandler) mapUpstreamError(c *echo.Context, err error) error {
	ctx := c.Request().Context()
	if _, ok := github.AsUnauthorized(err); ok {
		h.Logger.WarnContext(ctx, "github.repos unauthorized",
			slog.String("error_kind", "github_unauthorized"),
		)
		return writeEnvelope(c, http.StatusBadGateway,
			"github_unauthorized",
			"GitHub session expired. Reconnect GitHub.",
			nil,
		)
	}
	if _, ok := github.AsRateLimited(err); ok {
		h.Logger.WarnContext(ctx, "github.repos rate_limited")
		return writeEnvelope(c, http.StatusBadGateway,
			"github_rate_limited",
			"GitHub rate limit reached. Try again in a few minutes.",
			nil,
		)
	}
	var pe *github.ParseError
	if errors.As(err, &pe) {
		h.Logger.ErrorContext(ctx, "github.repos parse_error",
			slog.String("error_kind", "github_parse"),
		)
		return writeEnvelope(c, http.StatusBadGateway,
			"github_unreachable",
			"GitHub returned an unexpected response.",
			nil,
		)
	}
	h.Logger.ErrorContext(ctx, "github.repos transport_error",
		slog.String("error", err.Error()),
	)
	return writeEnvelope(c, http.StatusBadGateway,
		"github_unreachable",
		"Couldn't reach GitHub. Try again.",
		nil,
	)
}

// writeEnvelope is a small helper that wraps the {error, message, fields}
// envelope consistently. The locked error vocabulary comes from the
// existing writeError in organization_handler.go but is duplicated here
// to avoid a dependency from the github handler on the org handler.
func writeEnvelope(c *echo.Context, status int, code, message string, fields map[string]string) error {
	body := map[string]any{
		"error":   code,
		"message": message,
	}
	if len(fields) > 0 {
		body["fields"] = fields
	}
	return c.JSON(status, body)
}

// atoiOr returns the parsed int or the fallback if parsing fails.
func atoiOr(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return fallback
		}
		n = n*10 + int(ch-'0')
		if n > 1_000_000 {
			return fallback // guard against absurd values.
		}
	}
	return n
}

// userIDString converts an int64 user id to a string cache key. We avoid
// strconv to keep imports tight; the int is small enough that the
// formatting is bounded.
func userIDString(id int64) string {
	if id == 0 {
		return "0"
	}
	neg := id < 0
	if neg {
		id = -id
	}
	var buf [20]byte
	i := len(buf)
	for id > 0 {
		i--
		buf[i] = byte('0' + id%10)
		id /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// Compile-time guards.
var _ interface {
	ListRepos(c *echo.Context) error
} = (*GitHubHandler)(nil)

// silence unused import warning if errors is dropped later.
var _ = errors.New
var _ = context.Background
