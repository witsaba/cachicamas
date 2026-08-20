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
// diffs doc.go's parsed rows against. It mirrors doc.go's six rows
// byte-for-byte; a divergence in either direction is what proves the
// comparison is closed, not a containment check (S-AGP-013, S-AGP-014).
//
// L2C-04 (added by AG-04, agent-event-envelope) landed in the same commit
// as doc.go's own row, per this guard's own rule: count equality first,
// then per-row byte-exact diff (this file's own package comment, above).
// L2C-05 (added by AG-05, agent-message-tool-events) follows the same
// pattern: doc.go's row and the guard's expected table grow together.
// L2C-06 (added by AG-06, agent-protocol-events) follows the same
// pattern, with per-family semantics delegated to prose and per-family
// files rather than the guarded row (R-AGP-002's closed-amendment rule).
// L2C-07 (added by AG-12, agent-history) follows the same pattern: history
// is a second upward surface no existing row covers, so its own row and
// this table entry land in the same pull request (R-HIS-009).
// AG-18 amends the L2C-07 row's two independently falsifiable clauses —
// the route enumeration (append, seeded construction, orphan synthesis,
// prefix replacement) and the identity-stability clause (stable within a
// transcript generation, not unqualified) — confined to those two
// clauses, byte-in-sync with doc.go, landed in the same pull request
// (R-HIS-009's closed-amendment rule; S-HIS-104 proves both moved
// together).
// AG-20 amends the L2C-08 row's goroutine-exit clause — discovered
// during sdd-verify (CRITICAL-2), not planned at design time — on the
// identical AG-18/L2C-07 precedent: AG-20 introduces the observer lane
// (agent-hook-taxonomy delta, R-HKS-008), a package-owned goroutine
// that MAY legitimately outlive the wind-down bound while blocked
// inside a caller-registered observing hook, exactly as third-party
// tool code already may. The row is widened to name that second,
// disjoint carve-out — reported typed by hook point and registration
// index, mirroring the tool carve-out's "reported typed by tool and
// call identity" — rather than left silently false of the shipped
// code. Every other clause is unchanged. Full account: the
// agent-cancellation-tree delta of cachicamas-agent-hook-taxonomy
// (R-CAN-006, R-CAN-008, S-CAN-016, S-CAN-017).
var expectedLayer2ContractRows = []contractRow{
	{id: "L2C-01", text: "Imports: the Go standard library, github.com/cachicamas/backend/agent/src/ai and its measured transitive closure — nothing else, deny-by-default (ADR 0005 § D1 row 2; import_boundary_test.go)."},
	{id: "L2C-02", text: "No I/O of its own: no environment read, no filesystem access, no network call, no process spawn (ADR 0005 § D1 row 2; ambient_authority_test.go and import_boundary_test.go)."},
	{id: "L2C-03", text: "The event stream is the only upward contract: Layer 2 reports exclusively through emitted events; callers observe the stream, they never reach into the loop (AG-00 vocabulary)."},
	{id: "L2C-04", text: "Stream membership: a fact belongs on the event stream or it does not exist upward — if it is not on the stream, no frontend can render it and no log can reconstruct it. This is the criterion every later event family is judged by (doc 0003 AG-04 acceptance; agent-event-envelope)."},
	{id: "L2C-05", text: "Message and tool families are reconstructable: a session log carrying a message-text bracket, a reasoning bracket, or a tool-call bracket reconstructs each message and each tool outcome independently and completely, so a frontend can render \"what the model said\" and \"what the tools did\" without losing interleaving order (doc 0003 AG-05 acceptance; agent-message-tool-events). Reconstruction helpers are test-only code; the property is the test."},
	{id: "L2C-06", text: "Permission, cost, delegation, and compaction families are constructible on the event stream: a session log receives each family's events with their typed payloads and identity fields, and a frontend renders \"may this call proceed?\", the model's token usage, the delegation tree, and the compaction lifecycle without losing interleaving order (doc 0003 AG-06 acceptance; agent-protocol-events). Per-family semantics — the closed PermissionOutcome enum (R-APE-002), the token-only CostFigures shape (R-APE-004), the parent identifier envelope field (R-APE-006), the turn-bracket CompactionSpan (R-APE-007) — belong in this package's prose and per-family files, not in the guarded row."},
	{id: "L2C-07", text: "History is the second upward surface and has exactly one commit path: the transcript a run accumulates is read back as unmodified Layer 1 values with ordinal entry identity stable within a transcript generation, and every route that can extend it — append, seeded construction, orphan synthesis, prefix replacement — funnels through one validating commit primitive enforcing the pairing invariant (every tool call has a matching result) at the boundary, with no privileged bypass for internal callers; a synthesized interruption result is distinguishable from a real one by envelope origin alone (doc 0003 AG-12 acceptance; agent-history)."},
	{id: "L2C-08", text: "Cancellation is a bounded, typed, two-signal tree: interrupt aborts the run and keeps the harness; shutdown aborts the run and terminally refuses new prompts; both propagate down through loop, provider and tools as one context cancellation cause, stay errors.Is-distinguishable in the error chain and distinct in the run-end outcome; after the documented wind-down bound only third-party tool code — reported typed by tool and call identity — or a permanently stalled third-party observing-hook invocation — reported typed by hook point and registration index — may remain running, and every other goroutine the package itself owns has exited (doc 0003 AG-14 acceptance; agent-cancellation-tree)."},
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

// TestDocContract_L2C07BothClausesTogether — S-HIS-104, AG-18. The
// amended L2C-07 row's text in doc.go and in
// expectedLayer2ContractRows is byte-identical between the two; the
// route clause names all FOUR routes including the prefix replacement;
// the identity clause states stability within a transcript generation
// rather than unqualified stability; and the pairing-invariant,
// no-privileged-bypass and origin-distinguishability clauses are
// preserved verbatim. The row cannot be half-amended: this scenario
// checks the route clause and the identity clause as two INDEPENDENT
// assertions, so a scratch edit touching only one of them would fail
// the other — by construction, not by a separate mutation drill (this
// row's amendment is not one of the four mandated bites).
func TestDocContract_L2C07BothClausesTogether(t *testing.T) {
	t.Parallel()

	path := docGoPath(t)
	rows := parseLayer2ContractRows(t, path)

	var docRow, wantRow contractRow
	foundDoc, foundWant := false, false
	for _, r := range rows {
		if r.id == "L2C-07" {
			docRow, foundDoc = r, true
		}
	}
	for _, r := range expectedLayer2ContractRows {
		if r.id == "L2C-07" {
			wantRow, foundWant = r, true
		}
	}
	if !foundDoc || !foundWant {
		t.Fatalf("L2C-07 row missing: found in doc.go=%v, found in expectedLayer2ContractRows=%v", foundDoc, foundWant)
	}
	if docRow.text != wantRow.text {
		t.Fatalf("L2C-07 text differs between doc.go and expectedLayer2ContractRows:\n  doc.go:    %q\n  committed: %q", docRow.text, wantRow.text)
	}

	if !strings.Contains(docRow.text, "append, seeded construction, orphan synthesis, prefix replacement") {
		t.Error("L2C-07 route clause does not name all four routes including prefix replacement")
	}
	if !strings.Contains(docRow.text, "ordinal entry identity stable within a transcript generation") {
		t.Error("L2C-07 identity clause does not state generation-scoped stability")
	}
	for _, mustContain := range []string{
		"enforcing the pairing invariant (every tool call has a matching result) at the boundary",
		"no privileged bypass for internal callers",
		"a synthesized interruption result is distinguishable from a real one by envelope origin alone",
	} {
		if !strings.Contains(docRow.text, mustContain) {
			t.Errorf("L2C-07 no longer preserves the clause %q verbatim", mustContain)
		}
	}
}

// TestDocContract_L2C08CarveOutClause — S-CAN-017, AG-20 (sdd-verify
// CRITICAL-2). The amended L2C-08 row's text in doc.go and in
// expectedLayer2ContractRows is byte-identical between the two; the
// pre-existing clauses — the two-signal tree, the propagation-as-one-
// cancellation-cause clause, the errors.Is-distinguishability clause,
// and the original third-party-tool carve-out ("reported typed by
// tool and call identity") — are all preserved verbatim; and the row
// additionally states a SECOND, disjoint carve-out naming the class
// R-CAN-006's widened table admits at AG-20: a permanently stalled
// third-party observing-hook invocation, reported typed by hook point
// and registration index. Mirrors TestDocContract_L2C07BothClausesTogether's
// shape exactly (the AG-18/L2C-07 precedent this amendment follows):
// both the preserved half and the new half are checked as independent
// assertions, so a scratch edit touching only one would fail the
// other.
func TestDocContract_L2C08CarveOutClause(t *testing.T) {
	t.Parallel()

	path := docGoPath(t)
	rows := parseLayer2ContractRows(t, path)

	var docRow, wantRow contractRow
	foundDoc, foundWant := false, false
	for _, r := range rows {
		if r.id == "L2C-08" {
			docRow, foundDoc = r, true
		}
	}
	for _, r := range expectedLayer2ContractRows {
		if r.id == "L2C-08" {
			wantRow, foundWant = r, true
		}
	}
	if !foundDoc || !foundWant {
		t.Fatalf("L2C-08 row missing: found in doc.go=%v, found in expectedLayer2ContractRows=%v", foundDoc, foundWant)
	}
	if docRow.text != wantRow.text {
		t.Fatalf("L2C-08 text differs between doc.go and expectedLayer2ContractRows:\n  doc.go:    %q\n  committed: %q", docRow.text, wantRow.text)
	}

	for _, mustContain := range []string{
		"Cancellation is a bounded, typed, two-signal tree",
		"interrupt aborts the run and keeps the harness",
		"shutdown aborts the run and terminally refuses new prompts",
		"both propagate down through loop, provider and tools as one context cancellation cause",
		"stay errors.Is-distinguishable in the error chain and distinct in the run-end outcome",
		"only third-party tool code — reported typed by tool and call identity —",
	} {
		if !strings.Contains(docRow.text, mustContain) {
			t.Errorf("L2C-08 no longer preserves the pre-existing clause %q verbatim", mustContain)
		}
	}

	if !strings.Contains(docRow.text, "a permanently stalled third-party observing-hook invocation — reported typed by hook point and registration index") {
		t.Error("L2C-08 does not state the AG-20 observing-hook carve-out by hook point and registration index")
	}
	if !strings.Contains(docRow.text, "every other goroutine the package itself owns has exited") {
		t.Error("L2C-08's blanket clause no longer scopes to \"every OTHER goroutine\" now that a second carve-out exists")
	}
}

// TestDocContract_RowCountReScoped — S-HIS-080. The
// expectedLayer2ContractRows table contains the L2C-07 row, present
// and in its committed position, with row text referencing history's
// single validated commit path as a package-wide guarantee — scoped
// OFF the literal row count, the repeated count-assertion drift class
// this repository has hit before (AG-14 appended L2C-08 without any
// test ever asserting "7 rows", and this scenario itself asserts no
// literal row count either). S-HIS-081's bite (a scratch row appended
// to doc.go with no matching expectedLayer2ContractRows entry, FAILS
// naming the unexpected row) is unchanged by this change: it is
// TestLayer2DocContract_MatchesTheCommittedTable's own pre-existing
// count-mismatch detection, untouched here.
func TestDocContract_RowCountReScoped(t *testing.T) {
	t.Parallel()

	const wantIndex = 6 // 0-based: L2C-01..L2C-06 precede it, L2C-08 follows.
	if wantIndex >= len(expectedLayer2ContractRows) {
		t.Fatalf("expectedLayer2ContractRows has only %d row(s), want at least %d", len(expectedLayer2ContractRows), wantIndex+1)
	}
	row := expectedLayer2ContractRows[wantIndex]
	if row.id != "L2C-07" {
		t.Fatalf("expectedLayer2ContractRows[%d].id = %q, want %q", wantIndex, row.id, "L2C-07")
	}
	if !strings.Contains(row.text, "History is the second upward surface and has exactly one commit path") {
		t.Error("L2C-07's row text no longer references history's single validated commit path as a package-wide guarantee")
	}

	path := docGoPath(t)
	rows := parseLayer2ContractRows(t, path)
	if len(rows) <= wantIndex || rows[wantIndex].id != "L2C-07" {
		t.Errorf("doc.go's own row at index %d is not L2C-07", wantIndex)
	}
}
