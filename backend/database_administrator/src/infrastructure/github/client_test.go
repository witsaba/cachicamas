// Package github client_test.go — strict TDD coverage for the GitHub
// REST client. Uses httptest.Server to mock api.github.com/user/repos.
//
// Tasks covered (sdd/2026-07-06-workspaces/tasks §PR1c-i):
//
//	T-WS-1Ci-004 RED     — happy-path parsing
//	T-WS-1Ci-005 GREEN   — Client + ListUserRepos
//	T-WS-1Ci-006 TRIA    — 401 + 403 + malformed JSON
//	T-WS-1Ci-007 RED     — IsRepoAccessible finds id
//	T-WS-1Ci-008 GREEN   — IsRepoAccessible implemented
package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	gh "github.com/cachicamas/backend/database_administrator/src/infrastructure/github"
)

const sampleReposJSON = `[
  {
    "id": 123,
    "name": "hello",
    "full_name": "octocat/hello",
    "private": false,
    "description": "first repo",
    "html_url": "https://github.com/octocat/hello",
    "updated_at": "2026-01-15T12:00:00Z",
    "stargazers_count": 42,
    "owner": { "login": "octocat" }
  },
  {
    "id": 456,
    "name": "world",
    "full_name": "octocat/world",
    "private": true,
    "description": "",
    "html_url": "https://github.com/octocat/world",
    "updated_at": "2026-02-01T08:30:00Z",
    "stargazers_count": 7,
    "owner": { "login": "octocat" }
  }
]`

// newMockGH spins up an httptest.Server that captures the bearer token,
// serves the given status + body when /user/repos is hit, and counts
// request volume. Returns the server, the atomic request counter, and
// the captured bearer token string.
func newMockGH(t *testing.T, status int, body string) (*httptest.Server, *atomic.Int64, *string) {
	t.Helper()
	var calls atomic.Int64
	var capturedToken string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		capturedToken = r.Header.Get("Authorization")
		if r.URL.Path != "/user/repos" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv, &calls, &capturedToken
}

func TestClient_ListUserRepos_HappyPath(t *testing.T) {
	// RED for T-WS-1Ci-004, GREEN for T-WS-1Ci-005.
	srv, calls, token := newMockGH(t, http.StatusOK, sampleReposJSON)
	c := gh.NewClientWithBase(srv.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	repos, hasNext, err := c.ListUserRepos(ctx, "test-token", 1, 30)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if repos[0].ID != 123 || repos[0].FullName != "octocat/hello" || repos[0].OwnerLogin != "octocat" {
		t.Errorf("first repo wrong: %+v", repos[0])
	}
	if repos[1].Private != true {
		t.Errorf("second repo should be private, got %+v", repos[1])
	}
	if hasNext {
		t.Errorf("hasNext should be false (page returned < perPage)")
	}
	if calls.Load() != 1 {
		t.Errorf("expected 1 GitHub call, got %d", calls.Load())
	}
	if !strings.HasPrefix(*token, "Bearer ") {
		t.Errorf("expected Bearer header, got %q", *token)
	}
	if *token != "Bearer test-token" {
		t.Errorf("bearer token wrong: %q", *token)
	}
}

func TestClient_ListUserRepos_PaginationHint(t *testing.T) {
	// TRIANGULATE: perPage=2 and response has 2 items → hasNext=true.
	srv, _, _ := newMockGH(t, http.StatusOK, sampleReposJSON)
	c := gh.NewClientWithBase(srv.URL)
	repos, hasNext, err := c.ListUserRepos(context.Background(), "t", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(repos))
	}
	if !hasNext {
		t.Errorf("expected hasNext=true when response == perPage")
	}
}

func TestClient_ListUserRepos_UnauthorizedReturnsTypedError(t *testing.T) {
	// TRIANGULATE (a) for T-WS-1Ci-006.
	srv, _, _ := newMockGH(t, http.StatusUnauthorized, `{"message":"bad token"}`)
	c := gh.NewClientWithBase(srv.URL)
	_, _, err := c.ListUserRepos(context.Background(), "expired", 1, 30)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	u, ok := gh.AsUnauthorized(err)
	if !ok {
		t.Fatalf("expected *UnauthorizedError, got %T (%v)", err, err)
	}
	if u == nil {
		t.Fatal("UnauthorizedError wrapper nil")
	}
}

func TestClient_ListUserRepos_RateLimitedReturnsTypedError(t *testing.T) {
	// TRIANGULATE (b) for T-WS-1Ci-006.
	srv, _, _ := newMockGH(t, http.StatusForbidden, `{"message":"rate limit"}`)
	c := gh.NewClientWithBase(srv.URL)
	_, _, err := c.ListUserRepos(context.Background(), "t", 1, 30)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if _, ok := gh.AsRateLimited(err); !ok {
		t.Fatalf("expected *RateLimitedError, got %T (%v)", err, err)
	}
}

func TestClient_ListUserRepos_MalformedJSONReturnsParseError(t *testing.T) {
	// TRIANGULATE (c) for T-WS-1Ci-006.
	srv, _, _ := newMockGH(t, http.StatusOK, "not-json{")
	c := gh.NewClientWithBase(srv.URL)
	_, _, err := c.ListUserRepos(context.Background(), "t", 1, 30)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *gh.ParseError
	if !asParse(err, &pe) {
		t.Fatalf("expected *ParseError, got %T (%v)", err, err)
	}
}

// asParse is a tiny helper to keep the import surface tight: errors.As
// directly would force the test file to import "errors".
func asParse(err error, target **gh.ParseError) bool {
	if pe, ok := err.(*gh.ParseError); ok {
		*target = pe
		return true
	}
	return false
}

func TestClient_ListUserRepos_EmptyTokenRejected(t *testing.T) {
	// Defense-in-depth: empty token must not even hit the network.
	// We point at an httptest server that counts calls; if it receives
	// a hit, the test fails.
	srv, calls, _ := newMockGH(t, http.StatusOK, `[]`)
	c := gh.NewClientWithBase(srv.URL)
	_, _, err := c.ListUserRepos(context.Background(), "", 1, 30)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
	if calls.Load() != 0 {
		t.Errorf("empty token should not hit the network, got %d calls", calls.Load())
	}
}

func TestClient_IsRepoAccessible_Found(t *testing.T) {
	// RED for T-WS-1Ci-007, GREEN for T-WS-1Ci-008.
	srv, _, _ := newMockGH(t, http.StatusOK, sampleReposJSON)
	c := gh.NewClientWithBase(srv.URL)
	ok, err := c.IsRepoAccessible(context.Background(), "test-token", 456)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Errorf("expected repo id=456 to be found")
	}
}

func TestClient_IsRepoAccessible_NotInFirstPage(t *testing.T) {
	// TRIANGULATE: an id that isn't in the mocked response → false.
	srv, _, _ := newMockGH(t, http.StatusOK, sampleReposJSON)
	c := gh.NewClientWithBase(srv.URL)
	ok, err := c.IsRepoAccessible(context.Background(), "test-token", 9999)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Errorf("expected id=9999 not to be found")
	}
}

func TestClient_IsRepoAccessible_ZeroIDRejected(t *testing.T) {
	// TRIANGULATE: zero / negative id rejected before hitting GitHub.
	srv, calls, _ := newMockGH(t, http.StatusOK, sampleReposJSON)
	c := gh.NewClientWithBase(srv.URL)
	_, err := c.IsRepoAccessible(context.Background(), "t", 0)
	if err == nil {
		t.Fatal("expected error for githubID=0")
	}
	if calls.Load() != 0 {
		t.Errorf("githubID=0 should not hit the network, got %d calls", calls.Load())
	}
}
