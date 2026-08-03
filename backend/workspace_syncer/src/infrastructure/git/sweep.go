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
// applies uniformly. Per spec audit finding L-1: before
// RemoveAll, the sweep MUST EvalSymlinks on the candidate path
// and refuse the removal if the resolved target escapes
// dataDir. Otherwise a symlink planted inside dataDir could
// redirect RemoveAll to an arbitrary host location (e.g.
// `/etc`).
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

// FS is the small interface the sweep uses to read, stat, and
// remove directories. The production code passes OSFS; tests
// can use an in-memory fake. EvalSymlinks is the security-critical
// addition: the sweep MUST resolve symlinks before RemoveAll so
// a planted symlink cannot redirect the removal outside dataDir.
type FS interface {
	ReadDir(dirname string) ([]ioFS.DirEntry, error)
	RemoveAll(path string) error
	// EvalSymlinks resolves every symlink in path and returns the
	// canonical absolute form. The sweep uses this to verify a
	// candidate removal target is still inside dataDir.
	EvalSymlinks(path string) (string, error)
	// Lstat returns the file-info for path WITHOUT following
	// symlinks. Used to detect symlink entries (the prior code
	// called entry.IsDir(), which silently follows symlinks).
	Lstat(path string) (ioFS.FileInfo, error)
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

// EvalSymlinks delegates to filepath.EvalSymlinks.
func (OSFS) EvalSymlinks(path string) (string, error) {
	return filepath.EvalSymlinks(path)
}

// Lstat delegates to os.Lstat (does NOT follow symlinks).
func (OSFS) Lstat(path string) (ioFS.FileInfo, error) {
	return os.Lstat(path)
}

// Sweep walks dataDir and removes any directory whose name is
// not a positive int64 in the liveIDs map. The sweep is
// idempotent: re-running it on the same filesystem produces
// the same result.
//
// SECURITY: per audit finding L-1, before calling RemoveAll
// the sweep EvalSymlinks the candidate path and refuses the
// removal if the resolved target escapes dataDir. A symlink
// planted inside dataDir pointing outside (e.g. at /etc)
// would otherwise cause RemoveAll to follow the link and
// delete arbitrary host files.
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

	// Resolve the dataDir once so the per-iteration containment
	// check can compare against the canonical absolute form. If
	// EvalSymlinks fails (e.g. dataDir itself does not exist),
	// the subsequent ReadDir returns ErrNotExist and the sweep
	// short-circuits there.
	dataDirCanonical, err := fs.EvalSymlinks(dataDir)
	if err != nil {
		// dataDir may not exist yet (first boot). Let ReadDir
		// produce the canonical ErrNotExist branch.
		dataDirCanonical = dataDir
	}

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

	var removed, skipped, failed, refusedSymlink int
	for _, entry := range entries {
		if sweepCtx.Err() != nil {
			log.WarnContext(ctx, "sweep: timeout reached, aborting",
				slog.Int("processed", removed+skipped+failed+refusedSymlink),
				slog.Int("removed", removed),
				slog.Int("skipped", skipped),
				slog.Int("failed", failed),
				slog.Int("refused_symlink", refusedSymlink),
			)
			break
		}
		path := filepath.Join(dataDir, entry.Name())
		// Lstat (NOT stat) so a symlink is detected as a
		// symlink, not silently followed.
		info, lstatErr := fs.Lstat(path)
		if lstatErr != nil {
			log.WarnContext(ctx, "sweep: lstat failed (skipping)",
				slog.String("name", entry.Name()),
				slog.String("error", lstatErr.Error()),
			)
			skipped++
			continue
		}
		// Skip anything that is not a directory (regular files,
		// symlinks, devices, etc.). The prior entry.IsDir() check
		// silently followed symlinks and was the L-1 footgun.
		if !info.IsDir() {
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

		// SECURITY (audit L-1): before RemoveAll, resolve the
		// candidate through EvalSymlinks and refuse if the
		// resolved target escapes dataDirCanonical. This stops a
		// planted symlink from redirecting the removal.
		resolved, evalErr := fs.EvalSymlinks(path)
		if evalErr != nil {
			log.ErrorContext(ctx, "sweep: EvalSymlinks failed (refusing to remove)",
				slog.Int64("id", id),
				slog.String("path", path),
				slog.String("error", evalErr.Error()),
			)
			refusedSymlink++
			continue
		}
		if !pathContains(resolved, dataDirCanonical) {
			log.ErrorContext(ctx, "sweep: candidate resolves outside dataDir (refusing to remove; possible symlink attack)",
				slog.Int64("id", id),
				slog.String("path", path),
				slog.String("resolved", resolved),
				slog.String("data_dir", dataDirCanonical),
			)
			refusedSymlink++
			continue
		}

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
		slog.Int("refused_symlink", refusedSymlink),
	)
	return nil
}

// pathContains reports whether child is the same path as parent
// or a descendant of parent. Both arguments are expected to be
// canonical absolute paths (the output of EvalSymlinks). The
// comparison is purely textual; on Linux this is sufficient
// because EvalSymlinks resolves ".." and "." components. A
// defense-in-depth trailing-separator check prevents the
// `/data` vs `/data2` boundary collision.
func pathContains(child, parent string) bool {
	if child == parent {
		return true
	}
	// Ensure parent has a trailing separator so prefix match is
	// bounded to the directory boundary.
	parentWithSep := parent
	if parentWithSep == "" || parentWithSep[len(parentWithSep)-1] != '/' {
		parentWithSep += "/"
	}
	return len(child) > len(parentWithSep) && child[:len(parentWithSep)] == parentWithSep
}