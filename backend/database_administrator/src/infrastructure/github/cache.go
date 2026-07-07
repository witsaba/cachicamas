// Package github contains the adapter that proxies the GitHub REST API on
// behalf of authenticated users. The package is split into three files:
//
//   - cache.go      — 5-min in-memory RepoCache keyed by user_id (this file).
//   - client.go     — thin net/http wrapper around GET /user/repos.
//   - errors.go     — Unauthorized + RateLimited sentinel error types.
//
// Why a separate package from postgres/: the GitHub adapter is a third-party
// outbound dependency, not a database. Keeping it on its own prevents an
// accidental import of pgx from the github/client.go (mirror of
// postgres/organization_repo.go's rule).
//
// Security (PR1c-i, R-WS-009):
//   - The cache key is `userID`, NOT the OAuth token. The token never enters
//     this package at all.
//   - The cache values are slices of public `Repo` structs (id, full_name,
//     owner.login, …); they contain no secrets.
//   - No token-shaped field is ever serialized into the cached value.
package github

import (
	"sync"
	"time"
)

// Repo is the minimal projection of a GitHub repository the workspaces
// feature needs. It deliberately omits private fields (e.g. webhook URLs,
// tokens) so the struct can be safely cached + serialized to the frontend.
type Repo struct {
	ID          int64     `json:"id"`
	FullName    string    `json:"full_name"`
	OwnerLogin  string    `json:"owner_login"`
	Name        string    `json:"name"`
	Private     bool      `json:"private"`
	Description string    `json:"description"`
	HTMLURL     string    `json:"html_url"`
	UpdatedAt   time.Time `json:"updated_at"`
	Stargazers  int       `json:"stargazers_count"`
}

// cacheEntry is the in-memory cache row. `repos` is the value; `expiresAt`
// is the wall-clock TTL marker. The struct is unexported so callers cannot
// bypass the Get/Set/Bust API.
type cacheEntry struct {
	repos     []Repo
	expiresAt time.Time
}

// RepoCache is the in-memory, per-user cache for `/user/repos` results.
// One entry per user; each entry expires `ttl` after the Set call.
//
// Concurrency: protected by a sync.RWMutex. Reads (Get) take a read lock;
// writes (Set, Bust) take a write lock.
//
// TTL semantics: an entry is expired as soon as time.Now() is at or after
// its expiresAt marker. An expired entry behaves as a miss; the caller is
// expected to re-populate via Set.
type RepoCache struct {
	mu   sync.RWMutex
	data map[string]cacheEntry
	ttl  time.Duration
}

// NewRepoCache constructs an empty cache with the given TTL. The TTL MUST
// be > 0; a zero TTL would behave as "never cache", which is not the
// intended use (the design locks TTL = 5 minutes).
func NewRepoCache(ttl time.Duration) *RepoCache {
	if ttl <= 0 {
		panic("github.NewRepoCache: ttl must be > 0 (use a small positive duration for tests)")
	}
	return &RepoCache{
		data: make(map[string]cacheEntry),
		ttl:  ttl,
	}
}

// Get returns the cached repos for userID. The boolean is true iff the
// entry exists and is not expired. Expired entries are returned as miss
// but are NOT auto-evicted (Bust or the next Set overwrites them).
func (c *RepoCache) Get(userID string) ([]Repo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.data[userID]
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.repos, true
}

// Set replaces the cached repos for userID with a fresh entry whose TTL
// starts now. Callers should normally call this only on a cache miss
// (after a successful GitHub fetch).
func (c *RepoCache) Set(userID string, repos []Repo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[userID] = cacheEntry{
		repos:     repos,
		expiresAt: time.Now().Add(c.ttl),
	}
}

// Bust removes the entry for userID. Subsequent Get calls return miss
// until a new Set re-populates.
//
// Bust is the cache eviction primitive used by the `?bust_cache=true`
// query parameter in github_handler.go. The query param exists so the UI
// can offer a "Refresh repos" button that always goes to GitHub instead
// of serving the stale cache.
func (c *RepoCache) Bust(userID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, userID)
}

// Size returns the number of entries currently held. Test-only helper —
// the runtime path never needs it.
func (c *RepoCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}
