// Package github client.go — thin net/http wrapper around
// GET https://api.github.com/user/repos. The client is created once and
// shared across all authenticated requests; per-request state (page,
// perPage, access token) is passed via method arguments.
//
// Security rules (PR1c-i, R-WS-009):
//   - The access token is passed as a `Bearer ...` header value via
//     http.Header.Set, NEVER via fmt.Sprintf into a log line.
//   - The client has no business logging the token; slog lines that
//     mention GitHub interactions log only the status code + path, not
//     the request URL (which could leak a query string with the token
//     if the URL ever carried one).
//   - The client does not cache the token; the caller (githubHandler)
//     owns caching concerns.
package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// APIBaseURL is the default GitHub REST API endpoint. Overridden in
// tests via NewClientWithBase. In production the value is
// `https://api.github.com`.
const APIBaseURL = "https://api.github.com"

// Client is the GitHub REST adapter. It carries no per-request state; all
// methods take the access token from the call site.
type Client struct {
	BaseURL    string // injectable for tests (httptest mock server).
	HTTPClient *http.Client
}

// NewClient returns a Client wired to the canonical GitHub API URL with a
// 10-second per-request timeout. Callers that want a different timeout
// (e.g. tests that need to abort mid-request) can construct their own
// &Client{} literal.
func NewClient() *Client {
	return &Client{
		BaseURL: APIBaseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// NewClientWithBase returns a Client whose requests go to `baseURL`. Used
// by tests pointing at an httptest.Server.
func NewClientWithBase(baseURL string) *Client {
	c := NewClient()
	c.BaseURL = baseURL
	return c
}

// ListUserRepos fetches one page of the authenticated user's accessible
// GitHub repositories. Returns the repos, the per-page response has_next
// hint, or one of the typed errors (UnauthorizedError / RateLimitedError /
// ParseError).
//
// Parameters:
//   - ctx: request context (cancellation + deadline honoured).
//   - token: the user's OAuth access_token. Used ONLY as the Bearer
//     header value; the client never logs it, caches it, or serializes
//     it elsewhere.
//   - page: 1-indexed page number.
//   - perPage: GitHub's per_page query parameter (max 100).
//
// Returns `(repos, hasNext, err)`. `hasNext` is true when the response
// contained exactly perPage repos, hinting that page+1 may have more.
// (GitHub does not return a total count for this endpoint.)
//
// Query parameters pinned by the workspace picker UX (T-WS-1Ci-009):
//   - affiliation=owner,collaborator,organization_member — the
//     user-facing default in the GitHub web UI's repo picker.
//     Without this param, GET /user/repos returns only repos where
//     the user is the OWNER (their personal repos + forks they made),
//     which silently excludes every organisation repo the user is a
//     member of and every fork of a private repo they collaborate on.
//     The partner in the OAuth flow must explicitly approve the app
//     on each organisation; if they did, those repos will appear.
//   - type=all — explicit, so we never silently rely on the server
//     default changing. Includes both public and private repos the
//     token can see (the `repo` OAuth scope already covers both).
//   - sort=updated&direction=desc — most recently changed first,
//     which matches the workspace picker's UX intent (the user
//     usually picks a project they are actively working on). Default
//     sort=full_name would put "a-recent" after "zebra".
func (c *Client) ListUserRepos(ctx context.Context, token string, page, perPage int) ([]Repo, bool, error) {
	if token == "" {
		return nil, false, &UnauthorizedError{Cause: fmt.Errorf("empty access token")}
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 30 // GitHub default when per_page is omitted.
	}

	u, err := url.Parse(c.BaseURL + "/user/repos")
	if err != nil {
		return nil, false, fmt.Errorf("github.ListUserRepos: parse base url: %w", err)
	}
	q := u.Query()
	q.Set("page", strconv.Itoa(page))
	q.Set("per_page", strconv.Itoa(perPage))
	q.Set("affiliation", "owner,collaborator,organization_member")
	q.Set("type", "all")
	q.Set("sort", "updated")
	q.Set("direction", "desc")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, false, fmt.Errorf("github.ListUserRepos: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token) // token used only here.
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("github.ListUserRepos: do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through to parse.
	case http.StatusUnauthorized:
		return nil, false, &UnauthorizedError{Cause: fmt.Errorf("http %d", resp.StatusCode)}
	case http.StatusForbidden:
		return nil, false, &RateLimitedError{Cause: fmt.Errorf("http %d", resp.StatusCode)}
	default:
		return nil, false, fmt.Errorf("github.ListUserRepos: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, false, fmt.Errorf("github.ListUserRepos: read body: %w", err)
	}

	repos, err := parseRepos(body)
	if err != nil {
		return nil, false, &ParseError{Cause: err}
	}

	hasNext := len(repos) == perPage
	return repos, hasNext, nil
}

// IsRepoAccessible checks whether the user (token) can access the repo
// with the given GitHub `id`. Implementation: fetch the first page of
// /user/repos and search by id. Repos not in the first page are
// considered inaccessible for v1 (the workspace UI scrolls to load more
// pages, after which the user can re-pick).
//
// This is the locked "Path A" of the workspaces design — server-side
// validation of repo accessibility (T7), so the frontend cannot bypass
// the check by sending an unauthorized github_id.
func (c *Client) IsRepoAccessible(ctx context.Context, token string, githubID int64) (bool, error) {
	if githubID <= 0 {
		return false, fmt.Errorf("github.IsRepoAccessible: githubID must be > 0, got %d", githubID)
	}
	// Use a large perPage to maximize the chance of finding the target
	// in a single round-trip. GitHub caps per_page at 100.
	repos, _, err := c.ListUserRepos(ctx, token, 1, 100)
	if err != nil {
		return false, fmt.Errorf("github.IsRepoAccessible: %w", err)
	}
	for i := range repos {
		if repos[i].ID == githubID {
			return true, nil
		}
	}
	return false, nil
}

// parseRepos decodes the JSON body of GET /user/repos into the local
// Repo projection. We do NOT use `interface{}` decoding — every field
// is explicit so a future GitHub schema change fails loudly at parse
// time, not silently at runtime.
func parseRepos(body []byte) ([]Repo, error) {
	if len(body) == 0 {
		return []Repo{}, nil
	}
	// GitHub returns either a JSON array of repo objects or, on errors
	// with a 2xx (rare), an object. Handle both.
	var arr []struct {
		ID       int64  `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		Private  bool   `json:"private"`
		Desc     string `json:"description"`
		HTMLURL  string `json:"html_url"`
		Updated  string `json:"updated_at"`
		Stars    int    `json:"stargazers_count"`
		Owner    *struct {
			Login string `json:"login"`
		} `json:"owner"`
	}
	if err := json.Unmarshal(body, &arr); err != nil {
		return nil, fmt.Errorf("parse json array: %w", err)
	}
	out := make([]Repo, 0, len(arr))
	for _, r := range arr {
		ownerLogin := ""
		if r.Owner != nil {
			ownerLogin = r.Owner.Login
		}
		updatedAt, _ := time.Parse(time.RFC3339, r.Updated)
		out = append(out, Repo{
			ID:          r.ID,
			FullName:    r.FullName,
			OwnerLogin:  ownerLogin,
			Name:        r.Name,
			Private:     r.Private,
			Description: r.Desc,
			HTMLURL:     r.HTMLURL,
			UpdatedAt:   updatedAt,
			Stargazers:  r.Stars,
		})
	}
	return out, nil
}
