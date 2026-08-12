// AG-03.1 — the doc-contract guard: Layer 2's layer contract is
// machine-checked, not remembered (R-AGP-002, design AD-4).
//
// # Substitution, not a match
//
// AG-03.1's cited "doc-guard byte-suffix convention" (doc 0003, tracing to
// doc 0002:155) exists nowhere in doc 0002 or doc 0003 as a worked
// mechanism — the only repo-wide hit is a one-line parenthetical example in
// a node-grammar table cell. This guard SUBSTITUTES the repository's real
// working precedent instead:
// backend/agent/src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go
// (AI-40.2, R-L2H-004). It resolves doc.go from this test file's own
// location via runtime.Caller(0), reads its raw bytes, matches a pinned
// tab-indented row grammar with a regexp, and diffs the parsed rows
// entry-by-entry and in order against a committed expectation table
// declared in this file — never by importing the package or reading
// `go/doc` output.
//
// A later milestone that appends a guarded paragraph to doc.go's contract
// MUST add its row here, to expectedLayer2ContractRows, in the SAME pull
// request — that is what turns an omitted amendment into a failing test
// instead of silent drift.
package agent_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// layer2ContractRowPattern is the Layer 2 contract's row grammar, pinned
// (design AD-4): a tab-indented doc-comment line naming one L2C-NN contract
// clause, mirroring AI-40.2's `^//\tCAP-[RO]-\d\d\b` shape, retargeted.
var layer2ContractRowPattern = regexp.MustCompile(`^//\tL2C-\d\d\t`)

// contractRow is one parsed row of doc.go's layer contract: the L2C-NN id
// exactly as written, and its clause text exactly as written.
type contractRow struct {
	id   string
	text string
}

// expectedLayer2ContractRows is the committed source of truth this guard
// diffs doc.go's parsed rows against. It mirrors doc.go's four rows
// byte-for-byte; a divergence in either direction is what proves the
// comparison is closed, not a containment check (S-AGP-013, S-AGP-014).
//
// L2C-04 (added by AG-04, agent-event-envelope) landed in the same commit
// as doc.go's own row, per this guard's own rule: count equality first,
// then per-row byte-exact diff (this file's own package comment, above).
var expectedLayer2ContractRows = []contractRow{
	{id: "L2C-01", text: "Imports: the Go standard library, github.com/cachicamas/backend/agent/src/ai and its measured transitive closure — nothing else, deny-by-default (ADR 0005 § D1 row 2; import_boundary_test.go)."},
	{id: "L2C-02", text: "No I/O of its own: no environment read, no filesystem access, no network call, no process spawn (ADR 0005 § D1 row 2; ambient_authority_test.go and import_boundary_test.go)."},
	{id: "L2C-03", text: "The event stream is the only upward contract: Layer 2 reports exclusively through emitted events; callers observe the stream, they never reach into the loop (AG-00 vocabulary)."},
	{id: "L2C-04", text: "Stream membership: a fact belongs on the event stream or it does not exist upward — if it is not on the stream, no frontend can render it and no log can reconstruct it. This is the criterion every later event family is judged by (doc 0003 AG-04 acceptance; agent-event-envelope)."},
}

// docGoPath resolves src/agent/doc.go relative to THIS test file's own
// source location — doc_matrix_guard_test.go's runtime.Caller(0)-based
// resolution, mirrored here. No "../" hops are needed: doc.go and this
// guard share the same directory.
func docGoPath(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to report this test file's own path — cannot resolve src/agent/doc.go")
	}
	return filepath.Join(filepath.Dir(thisFile), "doc.go")
}

// parseLayer2ContractRows reads path's raw bytes and returns every row
// matching layer2ContractRowPattern, in file order. It fails the test
// loudly, naming path, on an unresolvable file or a matched row that does
// not carry an id and a text field separated by a tab —
// doc_matrix_guard_test.go's posture, restated for this guard: never
// skips, never passes vacuously.
func parseLayer2ContractRows(t *testing.T, path string) []contractRow {
	t.Helper()

	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("doc-contract guard: %q is not resolvable — src/agent/doc.go must stay reachable at this location: %v", path, err)
	}

	var rows []contractRow
	for i, line := range strings.Split(string(contents), "\n") {
		if !layer2ContractRowPattern.MatchString(line) {
			continue
		}
		rest := strings.TrimPrefix(line, "//\t")
		fields := strings.SplitN(rest, "\t", 2)
		if len(fields) != 2 {
			t.Fatalf("doc-contract guard: %q line %d matches the row pattern but does not carry an id and text separated by a tab: %q", path, i+1, line)
		}
		rows = append(rows, contractRow{id: fields[0], text: fields[1]})
	}
	return rows
}

// TestLayer2DocContract_MatchesTheCommittedTable is the drift guard itself
// (R-AGP-002, S-AGP-012..014): it fails when the parsed row count differs
// from expectedLayer2ContractRows' count (added, missing, or both) —
// naming every found and every wanted row so the divergence is legible —
// and it fails naming the differing row, its expected text and its found
// text when a single row no longer matches, entry-for-entry, in order.
func TestLayer2DocContract_MatchesTheCommittedTable(t *testing.T) {
	t.Parallel()

	path := docGoPath(t)
	rows := parseLayer2ContractRows(t, path)

	if len(rows) != len(expectedLayer2ContractRows) {
		t.Fatalf("doc-contract guard: found %d of %d rows in %q — doc.go must carry exactly one row per committed contract entry, entry-for-entry with expectedLayer2ContractRows.\n  found rows: %+v\n  want rows:  %+v",
			len(rows), len(expectedLayer2ContractRows), path, rows, expectedLayer2ContractRows)
	}

	for i, want := range expectedLayer2ContractRows {
		got := rows[i]
		if got != want {
			t.Errorf("doc-contract guard: row %d = id %q text %q, want id %q text %q — doc.go has drifted from the committed table",
				i+1, got.id, got.text, want.id, want.text)
		}
	}
}
