// CH-10 test helpers — small, production-safe adapters used by the
// chat_test package's S-CPM-003 composition-root discipline test and
// any future white-box test that needs source-tree grep.
//
// productionSymbolMatches counts occurrences of a Go identifier
// (substring match, line-based) across the chat package's
// production source tree (chat/*.go excluding *_test.go files).
// Mirrors the AGENTS.md "composition-root-only" convention recorded
// at openspec/AGENTS.md and applies it programmatically.
//
// The helper uses `git grep` when available; falls back to a stdlib
// file walk when git is unavailable so tarball checkouts without
// history still see the surface.

package chat_test

import (
	"bufio"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

// productionSymbolMatches returns the number of occurrences of sym in
// the chat package's production source files (chat/*.go, excluding
// *_test.go). Uses `git grep` when available; falls back to a stdlib
// walk otherwise.
func productionSymbolMatches(t *testing.T, sym string) int {
	t.Helper()

	chatDir := resolveChatPackageDir(t)

	if gitAvailable() {
		out, err := exec.Command("git", "grep", "-n", "-F", sym, "--",
			filepath.Join(chatDir, "*.go"),
			":!*_test.go").CombinedOutput()
		if err == nil {
			return countLines(out)
		}
		// fall through to stdlib walk on git error
	}

	return countInProductionFiles(chatDir, sym)
}

// resolveChatPackageDir walks the file tree from the test's working
// directory until it finds the chat/*.go source directory. Returns
// the absolute path of the chat directory.
//
// The convention: Go tests run with cwd = package directory. We verify
// by checking that wire.go exists in cwd; if not, walk up.
//
// Renamed from chatPackageDir to avoid colliding with the
// chatPackageDir variable in store_guard_test.go (which exports a
// directory path differently — see store_guard_test.go:85).
func resolveChatPackageDir(t *testing.T) string {
	t.Helper()

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd: %v", err)
	}

	sentinel := filepath.Join(cwd, "wire.go")
	if _, err := os.Stat(sentinel); err == nil {
		return cwd
	}

	// Walk up to find a chat/ subdirectory containing wire.go.
	for dir := cwd; dir != "/" && dir != "."; dir = filepath.Dir(dir) {
		candidate := filepath.Join(dir, "chat")
		if _, err := os.Stat(filepath.Join(candidate, "wire.go")); err == nil {
			return candidate
		}
	}
	t.Fatalf("could not locate chat package directory from cwd=%q", cwd)
	return ""
}

// countLines returns the number of non-empty lines in raw.
func countLines(raw []byte) int {
	count := 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

// countInProductionFiles walks chatDir for *.go files (excluding
// *_test.go) and counts the lines containing sym.
func countInProductionFiles(chatDir, sym string) int {
	count := 0
	entries, err := os.ReadDir(chatDir)
	if err != nil {
		return 0
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") {
			continue
		}
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		//nolint:gosec // test-only read.
		f, err := os.Open(filepath.Join(chatDir, name))
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), sym) {
				count++
			}
		}
		_ = f.Close()
	}
	return count
}

// newEchoServerForTest builds a fresh Echo instance for handler
// tests. Echo v5 + a single route group suffices; the test drives
// the server via httptest.NewRecorder.
func newEchoServerForTest(t *testing.T) *echo.Echo {
	t.Helper()
	e := echo.New()
	return e
}