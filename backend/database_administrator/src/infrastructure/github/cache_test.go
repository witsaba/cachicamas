// Package github cache_test.go — strict TDD coverage for RepoCache.
//
// Tasks covered (sdd/2026-07-06-workspaces/tasks §PR1c-i):
//
//	T-WS-1Ci-001 RED    — missing-returns-false
//	T-WS-1Ci-002 GREEN  — Set then Get returns the value within TTL
//	T-WS-1Ci-003 TRIANGULATE — Bust + race + TTL expiry
package github_test

import (
	"sync"
	"testing"
	"time"

	gh "github.com/cachicamas/backend/database_administrator/src/infrastructure/github"
)

func TestRepoCache_MissingReturnsFalse(t *testing.T) {
	// RED for T-WS-1Ci-001: before Set, Get must return (nil, false).
	cache := gh.NewRepoCache(5 * time.Minute)
	repos, ok := cache.Get("user-1")
	if ok {
		t.Errorf("expected miss for unknown key, got entry with %d repos", len(repos))
	}
	if repos != nil {
		t.Errorf("expected nil repos on miss, got %v", repos)
	}
}

func TestRepoCache_SetThenGetReturnsValue(t *testing.T) {
	// GREEN for T-WS-1Ci-002.
	cache := gh.NewRepoCache(5 * time.Minute)
	want := []gh.Repo{
		{ID: 1, FullName: "octocat/hello", OwnerLogin: "octocat", Name: "hello"},
		{ID: 2, FullName: "octocat/world", OwnerLogin: "octocat", Name: "world"},
	}
	cache.Set("user-1", want)
	got, ok := cache.Get("user-1")
	if !ok {
		t.Fatalf("expected hit after Set, got miss")
	}
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("index %d: got %+v want %+v", i, got[i], want[i])
		}
	}
}

func TestRepoCache_BustRemovesEntry(t *testing.T) {
	// TRIANGULATE (a) for T-WS-1Ci-003.
	cache := gh.NewRepoCache(5 * time.Minute)
	cache.Set("user-1", []gh.Repo{{ID: 1}})
	cache.Bust("user-1")
	if _, ok := cache.Get("user-1"); ok {
		t.Errorf("expected miss after Bust, got hit")
	}
}

func TestRepoCache_ConcurrentGetIsRaceClean(t *testing.T) {
	// TRIANGULATE (b) for T-WS-1Ci-003. Must be run with -race.
	cache := gh.NewRepoCache(5 * time.Minute)
	cache.Set("user-1", []gh.Repo{{ID: 1}})
	const goroutines = 10
	const reads = 100
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for g := 0; g < goroutines; g++ {
		go func() {
			defer wg.Done()
			for i := 0; i < reads; i++ {
				if _, ok := cache.Get("user-1"); !ok {
					t.Errorf("unexpected miss under concurrent read")
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestRepoCache_TTLExpiry(t *testing.T) {
	// TRIANGULATE (c) for T-WS-1Ci-003. 1-second TTL, sleep past expiry.
	cache := gh.NewRepoCache(1 * time.Second)
	cache.Set("user-1", []gh.Repo{{ID: 1}})
	// Immediate read should hit.
	if _, ok := cache.Get("user-1"); !ok {
		t.Fatalf("immediate read should hit")
	}
	// After 1.1s the entry should be expired.
	time.Sleep(1100 * time.Millisecond)
	if _, ok := cache.Get("user-1"); ok {
		t.Errorf("expected miss after TTL expiry, got hit")
	}
}

func TestRepoCache_DifferentUsersDoNotCollide(t *testing.T) {
	// Defense-in-depth: even though the design scope is single-tenant
	// (one user per install), the cache key is per-user so a future
	// multi-tenant migration does not silently share entries.
	cache := gh.NewRepoCache(5 * time.Minute)
	cache.Set("user-1", []gh.Repo{{ID: 1}})
	cache.Set("user-2", []gh.Repo{{ID: 2}})
	got1, _ := cache.Get("user-1")
	got2, _ := cache.Get("user-2")
	if len(got1) != 1 || got1[0].ID != 1 {
		t.Errorf("user-1 entry wrong: %+v", got1)
	}
	if len(got2) != 1 || got2[0].ID != 2 {
		t.Errorf("user-2 entry wrong: %+v", got2)
	}
}
