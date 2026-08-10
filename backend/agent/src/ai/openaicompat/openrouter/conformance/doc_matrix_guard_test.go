// AI-40.2 — the doc-matrix drift guard (R-L2H-004, design AD-4).
//
// This file protects the capability matrix AI-40.2 publishes in
// src/ai/doc.go against silent drift: it parses the published rows back
// out of doc.go's own bytes and compares them, entry-by-entry and in
// order, to expectedOpenRouterRecord() — the same committed expectation
// capability_record_test.go already asserts the REAL generated record
// against (run_for_test.go's TestOpenRouterAdapter_FullConformance is
// that generator). It lives in this package, not a new one, because
// expectedOpenRouterRecord is unexported in a _test.go file and this is
// the only place that can reach it directly.
//
// It is read-only: it neither calls nor modifies anything in the
// conformance suite, any adapter, or the committed expectation itself —
// it only reads doc.go's bytes and expectedOpenRouterRecord()'s already-
// existing return value.
package openrouter_conformance

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// publishedMatrixRowPattern is the published matrix's row grammar,
// pinned (design AD-3/AD-4): a tab-indented doc-comment line naming one
// capability from either closed list.
var publishedMatrixRowPattern = regexp.MustCompile(`^//\tCAP-[RO]-\d\d\b`)

// matrixRow is one parsed row of the published capability matrix: the
// capability token exactly as Capability.String() renders it, the
// standing token exactly as Standing.String() renders it, and the
// outcome token exactly as Outcome.String() renders it.
type matrixRow struct {
	capability string
	standing   string
	outcome    string
}

// docGoPath resolves src/ai/doc.go relative to THIS test file's own
// source location — provider_signature_guard_test.go's
// runtime.Caller(0)-based resolution (ADR 0005 § D2, Guard C), mirrored
// here for the sibling doc.go three directories up:
// .../openrouter/conformance/ -> .../openrouter/ -> .../openaicompat/ ->
// .../ai/doc.go.
func docGoPath(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to report this test file's own path — cannot resolve src/ai/doc.go")
	}
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "doc.go")
}

// parsePublishedMatrix reads path and returns every row matching
// publishedMatrixRowPattern, in file order. It fails the test loudly,
// naming path, on an unresolvable file or a matched row that does not
// carry all three fields — R-AMP-015's posture, restated for this guard:
// never skips, never passes vacuously.
func parsePublishedMatrix(t *testing.T, path string) []matrixRow {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("doc-matrix guard: %q is not resolvable — src/ai/doc.go must stay reachable at this relative path (ADR 0005 § D2): %v", path, err)
	}

	var rows []matrixRow
	for i, line := range strings.Split(string(contents), "\n") {
		if !publishedMatrixRowPattern.MatchString(line) {
			continue
		}
		fields := strings.Fields(strings.TrimPrefix(line, "//"))
		if len(fields) < 3 {
			t.Fatalf("doc-matrix guard: %q line %d matches the row pattern but does not carry three fields (capability, standing, outcome): %q", path, i+1, line)
		}
		rows = append(rows, matrixRow{capability: fields[0], standing: fields[1], outcome: fields[2]})
	}
	return rows
}

// TestPublishedCapabilityMatrix_MatchesTheCommittedExpectation is the
// drift guard itself (R-L2H-004, S-L2H-017..019): it fails when the
// published row count differs from the committed expectation's nine
// entries (added, missing, or both), and it fails naming the differing
// row when a single row's capability, standing, or outcome no longer
// matches — entry-for-entry, in Capabilities()' order.
func TestPublishedCapabilityMatrix_MatchesTheCommittedExpectation(t *testing.T) {
	t.Parallel()

	path := docGoPath(t)
	rows := parsePublishedMatrix(t, path)

	want := expectedOpenRouterRecord().Entries()
	if len(rows) != len(want) {
		t.Fatalf("doc-matrix guard: found %d of %d rows in %q — the published matrix must carry exactly one row per capability, entry-for-entry with the committed expectation", len(rows), len(want), path)
	}

	for i, entry := range want {
		row := rows[i]
		wantCapability := entry.Capability.String()
		wantStanding := entry.Standing.String()
		wantOutcome := entry.Outcome.String()
		if row.capability != wantCapability || row.standing != wantStanding || row.outcome != wantOutcome {
			t.Errorf("doc-matrix guard: row %d = %q/%q/%q, want %q/%q/%q (capability/standing/outcome) — the published matrix has drifted from the committed expectation",
				i+1, row.capability, row.standing, row.outcome, wantCapability, wantStanding, wantOutcome)
		}
	}
}
