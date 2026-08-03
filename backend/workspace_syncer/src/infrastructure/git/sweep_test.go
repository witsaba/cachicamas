package git

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

// sweepTestLog returns a slog.Logger that drops every record.
func sweepTestLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		Level: slog.LevelError + 1,
	}))
}

func TestSweep_RemovesOrphansKeepsLive(t *testing.T) {
	dataDir := t.TempDir()

	// Three workspace directories + one non-numeric dir + one
	// stray file. The sweep MUST:
	//   - remove orphan numeric dirs (2)
	//   - keep live numeric dirs (1, 3)
	//   - SKIP non-numeric dirs (defensive: someone dropped a dir by hand)
	//   - SKIP files (the sweep only walks directories)
	mustMkdir(t, filepath.Join(dataDir, "1"))
	mustMkdir(t, filepath.Join(dataDir, "2"))
	mustMkdir(t, filepath.Join(dataDir, "3"))
	mustMkdir(t, filepath.Join(dataDir, "not-a-number"))
	mustWriteFile(t, filepath.Join(dataDir, "stray.txt"), "ignore me")

	// Live IDs: 1 and 3 (so 2 is orphan).
	live := map[int64]bool{1: true, 3: true}

	if err := Sweep(context.Background(), dataDir, live, OSFS{}, sweepTestLog()); err != nil {
		t.Fatalf("Sweep: %v", err)
	}

	mustExist(t, filepath.Join(dataDir, "1"))
	mustExist(t, filepath.Join(dataDir, "3"))
	mustNotExist(t, filepath.Join(dataDir, "2"))
	// Defensive entries are NOT removed; the sweep logs a warning.
	mustExist(t, filepath.Join(dataDir, "not-a-number"))
	mustExist(t, filepath.Join(dataDir, "stray.txt"))
}

func TestSweep_NoDataDirIsNoOp(t *testing.T) {
	// Fresh container has no /data/workspaces. Sweep must
	// succeed (not error) and return nil.
	if err := Sweep(context.Background(), "/nonexistent/path/that/does/not/exist", nil, OSFS{}, sweepTestLog()); err != nil {
		t.Errorf("Sweep on missing dir returned error: %v", err)
	}
}

func TestSweep_Idempotent(t *testing.T) {
	dataDir := t.TempDir()
	mustMkdir(t, filepath.Join(dataDir, "5"))

	if err := Sweep(context.Background(), dataDir, map[int64]bool{}, OSFS{}, sweepTestLog()); err != nil {
		t.Fatalf("Sweep first call: %v", err)
	}
	if err := Sweep(context.Background(), dataDir, map[int64]bool{}, OSFS{}, sweepTestLog()); err != nil {
		t.Fatalf("Sweep second call: %v", err)
	}
	mustNotExist(t, filepath.Join(dataDir, "5"))
}

func TestSweep_TimeoutAborts(t *testing.T) {
	// Construct a context that is already cancelled. Sweep
	// must abort cleanly (log a warning and return nil).
	dataDir := t.TempDir()
	mustMkdir(t, filepath.Join(dataDir, "1"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Sweep(ctx, dataDir, map[int64]bool{}, OSFS{}, sweepTestLog()); err != nil {
		t.Errorf("Sweep with cancelled ctx returned error: %v", err)
	}
}

func TestSweep_EmptyDataDirIsNoOp(t *testing.T) {
	dataDir := t.TempDir()
	if err := Sweep(context.Background(), dataDir, map[int64]bool{}, OSFS{}, sweepTestLog()); err != nil {
		t.Errorf("Sweep on empty dir returned error: %v", err)
	}
}

// TestSweep_RefusesOutOfTreeSymlink is the regression test for
// audit finding L-1: the prior sweep called RemoveAll on every
// numeric dir under dataDir without resolving symlinks. An
// attacker who could plant a symlink at /data/workspaces/123
// pointing to e.g. /etc would have caused the startup sweep to
// follow the link and delete arbitrary host files. The hardened
// sweep EvalSymlinks the candidate and refuses removal if the
// resolved target escapes dataDir.
func TestSweep_RefusesOutOfTreeSymlink(t *testing.T) {
	dataDir := t.TempDir()

	// Plant a victim directory outside the data dir.
	outsideDir := t.TempDir()
	canary := filepath.Join(outsideDir, "do_not_delete.txt")
	if err := os.WriteFile(canary, []byte("important"), 0o644); err != nil {
		t.Fatalf("seed canary: %v", err)
	}

	// Plant a symlink inside dataDir pointing outside.
	symlinkPath := filepath.Join(dataDir, "123")
	if err := os.Symlink(outsideDir, symlinkPath); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	if err := Sweep(context.Background(), dataDir, map[int64]bool{}, OSFS{}, sweepTestLog()); err != nil {
		t.Fatalf("Sweep returned error: %v", err)
	}

	// The symlink must NOT have been followed-and-deleted.
	// We accept either: (a) the symlink is still present
	// (because the sweep refused), OR (b) the symlink was
	// removed but the canary survives (because the sweep
	// removed only the symlink, not its target). The
	// canary itself MUST survive in both cases.
	if _, err := os.Stat(canary); err != nil {
		t.Errorf("canary file was deleted by the sweep: %v (L-1 regression: the sweep followed the symlink)", err)
	}

	// The symlink may still exist (the sweep refused to
	// remove it) OR it may have been removed as a symlink
	// (the sweep calls RemoveAll which on a symlink removes
	// the link, NOT the target — this is safe). Either is OK.
}

// TestSweep_RefusesPrefixCollision guards the pathContains
// defense-in-depth check: a candidate that is a sibling
// directory whose name starts with dataDir's name MUST be
// rejected even if its absolute path looks similar. We use a
// directory "/data/workspaces" with a planted victim at
// "/data/workspaces_evil/123" — the unhardened code would have
// classified the latter as "inside /data/workspaces" via naive
// string prefix; the hardened code calls EvalSymlinks which
// canonicalizes the path.
func TestSweep_RefusesPrefixCollision(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("running as root; symlink containment is moot for root")
	}
	dataDir := t.TempDir() // /tmp/TestX/...
	evil := dataDir + "_evil"
	if err := os.MkdirAll(evil, 0o755); err != nil {
		t.Fatalf("mkdir evil: %v", err)
	}
	defer func() { _ = os.RemoveAll(evil) }()

	canary := filepath.Join(evil, "do_not_delete.txt")
	if err := os.WriteFile(canary, []byte("important"), 0o644); err != nil {
		t.Fatalf("seed canary: %v", err)
	}

	symlinkPath := filepath.Join(dataDir, "456")
	if err := os.Symlink(evil, symlinkPath); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	if err := Sweep(context.Background(), dataDir, map[int64]bool{}, OSFS{}, sweepTestLog()); err != nil {
		t.Fatalf("Sweep returned error: %v", err)
	}

	if _, err := os.Stat(canary); err != nil {
		t.Errorf("canary file was deleted by the sweep: %v (pathContains prefix collision)", err)
	}
}

// TestPathContains_RejectsPrefixCollision is the unit test for
// the pathContains helper. Asserts that "/data" is NOT
// considered a parent of "/data_evil/x".
func TestPathContains_RejectsPrefixCollision(t *testing.T) {
	if pathContains("/data_evil/x", "/data") {
		t.Errorf("pathContains(/data_evil/x, /data) = true; must reject the sibling-prefix collision")
	}
	if !pathContains("/data/workspaces/123", "/data/workspaces") {
		t.Errorf("pathContains(/data/workspaces/123, /data/workspaces) = false; legitimate descendants must pass")
	}
	if !pathContains("/data/workspaces", "/data/workspaces") {
		t.Errorf("pathContains equal paths = false; the directory itself must be considered 'inside'")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func mustExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected %s to exist: %v", path, err)
	}
}

func mustNotExist(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected %s to NOT exist", path)
	}
}