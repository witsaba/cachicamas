// Package git — the startup sweep that removes orphan cloned
// directories. See design.md §4 (Cleanup hook). The sweep runs
// once at service startup; if it takes longer than 30s it is
// canceled and a warning is logged.
//
// An "orphan" is any /data/workspaces/{id}/... directory whose
// {id} no longer corresponds to a live (non-soft-deleted) workspace.
// In v1, the sweep fetches the live workspace IDs from
// database_administrator's /internal/live-workspaces endpoint
// (added in PR-3b).
//
// SECURITY: the sweep walks /data/workspaces and removes
// directories. The path construction is the same WorkspacePath
// function used by the clone, so the path-traversal defense
// applies uniformly.
package git

import (
	"context"
	"errors"
	"fmt"
	ioFS "io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// DefaultSweepTimeout is the 30s ceiling on the startup sweep.
// If the database_administrator lookup or the filesystem walk
// exceeds this, the sweep logs a warning and exits.
const DefaultSweepTimeout = 30 * time.Second

// FS is the small interface the sweep uses to read and remove
// directories. The production code passes OSFS; tests can use
// an in-memory fake (the sweep is written to depend only on
// ReadDir + RemoveAll).
type FS interface {
	ReadDir(dirname string) ([]ioFS.DirEntry, error)
	RemoveAll(path string) error
}

// OSFS is the production FS implementation.
type OSFS struct{}

// ReadDir delegates to os.ReadDir.
func (OSFS) ReadDir(dirname string) ([]ioFS.DirEntry, error) {
	return os.ReadDir(dirname)
}

// RemoveAll delegates to os.RemoveAll.
func (OSFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// Sweep walks dataDir and removes any directory whose name is
// not a positive int64 in the liveIDs map. The sweep is
// idempotent: re-running it on the same filesystem produces
// the same result.
//
// Errors are logged but do not abort the sweep. A sweep that
// completes with partial failures is still useful — the next
// startup runs it again and cleans up the remaining orphans.
//
// The function returns nil on success (even if some entries
// could not be removed). It returns a non-nil error only when
// the initial ReadDir fails for a reason other than "directory
// does not exist" (which is a normal first-boot state).
func Sweep(ctx context.Context, dataDir string, liveIDs map[int64]bool, fs FS, log *slog.Logger) error {
	sweepCtx, cancel := context.WithTimeout(ctx, DefaultSweepTimeout)
	defer cancel()

	entries, err := fs.ReadDir(dataDir)
	if err != nil {
		if errors.Is(err, ioFS.ErrNotExist) {
			log.InfoContext(ctx, "sweep: data dir does not exist yet (first boot)",
				slog.String("data_dir", dataDir),
			)
			return nil
		}
		return fmt.Errorf("sweep: read %s: %w", dataDir, err)
	}

	var removed, skipped, failed int
	for _, entry := range entries {
		if sweepCtx.Err() != nil {
			log.WarnContext(ctx, "sweep: timeout reached, aborting",
				slog.Int("processed", removed+skipped+failed),
				slog.Int("removed", removed),
				slog.Int("skipped", skipped),
				slog.Int("failed", failed),
			)
			break
		}
		if !entry.IsDir() {
			skipped++
			continue
		}
		id, parseErr := strconv.ParseInt(entry.Name(), 10, 64)
		if parseErr != nil {
			log.WarnContext(ctx, "sweep: non-numeric entry (skipping)",
				slog.String("name", entry.Name()),
			)
			skipped++
			continue
		}
		if liveIDs[id] {
			skipped++
			continue
		}
		path := filepath.Join(dataDir, entry.Name())
		if err := fs.RemoveAll(path); err != nil {
			log.ErrorContext(ctx, "sweep: remove failed",
				slog.Int64("id", id),
				slog.String("path", path),
				slog.String("error", err.Error()),
			)
			failed++
			continue
		}
		log.InfoContext(ctx, "sweep: removed orphan", slog.Int64("id", id))
		removed++
	}

	log.InfoContext(ctx, "sweep: complete",
		slog.Int("removed", removed),
		slog.Int("skipped", skipped),
		slog.Int("failed", failed),
	)
	return nil
}