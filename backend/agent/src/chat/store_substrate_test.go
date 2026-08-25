// CH-09.5 — substrate preservation guard (S-CTS-023, NFR-TLS-003 /
// NFR-CTS-003). Mirrors `TestTurn_SubstrateUntouched` (CH-08
// precedent, in `backend/agent/src/agent/loop_test.go`): runs
// `git diff --stat main..HEAD -- backend/agent/src/agent/` and
// FAILS on non-empty output. The 10-file substrate list is the
// chat-archetype counterpart of Layer 2's own substrate invariant —
// CH-09 must leave it byte-unchanged.
//
// The substrate list, verbatim from `openspec/AGENTS.md`:
//   - event_descriptor.go
//   - stream_check.go
//   - failure.go
//   - sequence.go
//   - event.go
//   - go.mod
//   - go.sum
//   - Makefile
//   - .golangci.yml
//   - import_boundary_test.go
//
// This test runs as part of `cd backend/agent && make test`. A
// future contributor adding a `Tool` helper inside the agent/
// directory trips the guard at PR-time, not at runtime.

package chat_test

import (
	"os/exec"
	"strings"
	"testing"
)

// TestChat_SubstrateUntouched is the chat-archetype substrate
// guard (S-CTS-023, NFR-CTS-003). Asserts that no file under
// `backend/agent/src/agent/` is modified by the merged change.
// Empty diff → test passes; non-empty diff → test fails with the
// full diff attached so the contributor sees exactly what drifted.
func TestChat_SubstrateUntouched(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not available in PATH; substrate guard is opt-in")
	}

	out, err := exec.Command("git", "diff", "--stat", "main..HEAD",
		"--", "backend/agent/src/agent/").CombinedOutput()
	if err != nil {
		t.Fatalf("git diff --stat main..HEAD -- backend/agent/src/agent/: %v\n%s", err, string(out))
	}
	if strings.TrimSpace(string(out)) != "" {
		t.Fatalf("substrate drift under backend/agent/src/agent/ (NFR-CTS-003 / S-CTS-023):\n%s", string(out))
	}
}

// gitAvailable returns true iff `git` is reachable on PATH. The
// guard test skips (rather than fails) when git is unavailable
// so the suite still runs on a tarball checkout without history.
func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}