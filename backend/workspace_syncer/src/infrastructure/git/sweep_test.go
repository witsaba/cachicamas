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