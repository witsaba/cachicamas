// Package archetype_test — version_test.go is the TDD contract for the
// version-aware rebuild helper (REQ-CCVP-001/002/003, design AD-3).
//
// Tests the `VersionTracker` helper in isolation. The helper is the
// unit-of-rebuild; future archetypes (or the chat Conversation) wrap
// it inside their runtime slot. By isolating the helper here, the
// tests do not require a full Provider / Store / ToolSource /
// PermissionPolicy setup.
//
// Five scenarios from spec #4077:
//
//	- Test_VersionTracker_FirstLoad_RecordsVersion
//	- Test_VersionTracker_VersionMismatch_AppliesNewPrompt
//	- Test_VersionTracker_VersionMatch_NoRebuild
//	- Test_VersionTracker_AbsentRow_DefaultsToVersion1
//	- Test_VersionTracker_NilLoader_NoOp
package archetype_test

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/archetype"
)

// fakeVersionedLoader satisfies archetype.Loader and lets the test
// control what LoadByKindAndOrg returns per call.
type fakeVersionedLoader struct {
	mu     sync.Mutex
	result archetype.ArchetypeConfig
	found  bool
	err    error
}

func (f *fakeVersionedLoader) LoadByKindAndOrg(_ context.Context, _ archetype.ArchetypeKind, _ string) (archetype.ArchetypeConfig, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.result, f.found, f.err
}

func (f *fakeVersionedLoader) WithTx(_ *sql.Tx) archetype.Loader { return f }

func (f *fakeVersionedLoader) withCurrent(cfg archetype.ArchetypeConfig, found bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.result = cfg
	f.found = found
}

// recordingApply captures every prompt the tracker applies. Tests use
// it to assert that the rebuild callback fires only on mismatch.
type recordingApply struct {
	mu      sync.Mutex
	prompts []string
}

func (r *recordingApply) apply(p string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.prompts = append(r.prompts, p)
}

func (r *recordingApply) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.prompts)
}

func (r *recordingApply) last() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.prompts) == 0 {
		return ""
	}
	return r.prompts[len(r.prompts)-1]
}

// Test_VersionTracker_FirstLoad_RecordsVersion — REQ-CCVP-001 /
// Scenario "first-load records the version". Given the Loader returns
// version=3, when the tracker is constructed, then the recorded
// version is 3.
func Test_VersionTracker_FirstLoad_RecordsVersion(t *testing.T) {
	t.Parallel()

	loader := &fakeVersionedLoader{
		result: archetype.ArchetypeConfig{
			Kind:           archetype.KindChat,
			OrgID:          "user_alice",
			SystemPrompt:   "first prompt",
			ToolAllowlist:  []string{"current_time"},
			DeferToolNames: []string{"summarize_conversation"},
			Version:        3,
		},
		found: true,
	}
	apply := &recordingApply{}

	vt, err := archetype.NewVersionTracker(context.Background(), loader, archetype.KindChat, "user_alice", apply.apply)
	if err != nil {
		t.Fatalf("NewVersionTracker: %v", err)
	}
	if got := vt.RecordedVersion(); got != 3 {
		t.Errorf("RecordedVersion = %d, want 3", got)
	}
	// First load records the version but does NOT call applyPrompt —
	// the prompt is already in place at construction time.
	if apply.count() != 0 {
		t.Errorf("applyPrompt called %d time(s) on first load, want 0 (the prompt is set at construction)", apply.count())
	}
}

// Test_VersionTracker_VersionMismatch_AppliesNewPrompt — REQ-CCVP-002
// / Scenario "version mismatch rebuilds the prompt on next Send".
func Test_VersionTracker_VersionMismatch_AppliesNewPrompt(t *testing.T) {
	t.Parallel()

	loader := &fakeVersionedLoader{
		result: archetype.ArchetypeConfig{
			Kind:         archetype.KindChat,
			OrgID:        "user_alice",
			SystemPrompt: "first prompt",
			Version:      3,
		},
		found: true,
	}
	apply := &recordingApply{}

	vt, err := archetype.NewVersionTracker(context.Background(), loader, archetype.KindChat, "user_alice", apply.apply)
	if err != nil {
		t.Fatalf("NewVersionTracker: %v", err)
	}

	loader.withCurrent(archetype.ArchetypeConfig{
		Kind:         archetype.KindChat,
		OrgID:        "user_alice",
		SystemPrompt: "second prompt after config change",
		Version:      4,
	}, true)

	if err := vt.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := vt.RecordedVersion(); got != 4 {
		t.Errorf("RecordedVersion = %d, want 4", got)
	}
	if apply.count() != 1 {
		t.Errorf("applyPrompt called %d time(s), want 1", apply.count())
	}
	if got := apply.last(); got != "second prompt after config change" {
		t.Errorf("last applied prompt = %q, want %q", got, "second prompt after config change")
	}
}

// Test_VersionTracker_VersionMatch_NoRebuild — REQ-CCVP-002 / Scenario
// "version match does not re-read".
func Test_VersionTracker_VersionMatch_NoRebuild(t *testing.T) {
	t.Parallel()

	loader := &fakeVersionedLoader{
		result: archetype.ArchetypeConfig{
			Kind:         archetype.KindChat,
			OrgID:        "user_alice",
			SystemPrompt: "first prompt",
			Version:      3,
		},
		found: true,
	}
	apply := &recordingApply{}

	vt, err := archetype.NewVersionTracker(context.Background(), loader, archetype.KindChat, "user_alice", apply.apply)
	if err != nil {
		t.Fatalf("NewVersionTracker: %v", err)
	}
	// No mutation; same version 3.
	if err := vt.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if got := vt.RecordedVersion(); got != 3 {
		t.Errorf("RecordedVersion = %d, want 3", got)
	}
	if apply.count() != 0 {
		t.Errorf("applyPrompt called %d time(s), want 0 (no change on match)", apply.count())
	}
}

// Test_VersionTracker_AbsentRow_DefaultsToVersion1 — REQ-CACS-003 +
// REQ-CCVP-001.
func Test_VersionTracker_AbsentRow_DefaultsToVersion1(t *testing.T) {
	t.Parallel()

	loader := &fakeVersionedLoader{
		result: archetype.DefaultConfig(archetype.KindChat, "user_alice", []string{"current_time", "summarize_conversation"}),
		found:  false,
	}
	apply := &recordingApply{}

	vt, err := archetype.NewVersionTracker(context.Background(), loader, archetype.KindChat, "user_alice", apply.apply)
	if err != nil {
		t.Fatalf("NewVersionTracker: %v", err)
	}
	if got := vt.RecordedVersion(); got != 1 {
		t.Errorf("RecordedVersion = %d, want 1 (defaults are version=1 per design AD-2)", got)
	}
	// Reload on still-default config is a no-op.
	if err := vt.Reload(context.Background()); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if apply.count() != 0 {
		t.Errorf("applyPrompt called %d time(s), want 0 (defaults stay defaults)", apply.count())
	}
}

// Test_VersionTracker_NilLoader_NoOp — defence in depth.
func Test_VersionTracker_NilLoader_NoOp(t *testing.T) {
	t.Parallel()

	apply := &recordingApply{}
	vt, err := archetype.NewVersionTracker(context.Background(), nil, archetype.KindChat, "user_alice", apply.apply)
	if err != nil {
		t.Fatalf("NewVersionTracker: %v", err)
	}
	if got := vt.RecordedVersion(); got != 0 {
		t.Errorf("RecordedVersion = %d, want 0 (nil loader → no version)", got)
	}
	if err := vt.Reload(context.Background()); err != nil {
		t.Errorf("Reload with nil loader returned err=%v, want nil", err)
	}
	if apply.count() != 0 {
		t.Errorf("applyPrompt called %d time(s), want 0 (no loader → no apply)", apply.count())
	}
}
