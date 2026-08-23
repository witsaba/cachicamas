# Design: CH-01 — Scaffold the archetype package and make its import boundary bite

Change `cachicamas-chat-package-scaffold`. Executes proposal D-1…D-7 as concrete Go shapes, plus one later user decision (Engram #3741, below) that amends D-1's OTel list. Every `file:line` cites the **merge-base tree** (`419a4291`) — check 6 is inserted mid-file, so post-apply line numbers below the insertion point shift (verify must re-resolve, per `an-insertion-invalidates-citations-below-it`). Unattributed `:NNN` cites are into `backend/agent/src/agent/import_boundary_test.go`; `adr:NNN` cites are into `docs/adr/0005-promote-agent-stack-to-own-module.md`.

## Technical Approach

One in-place extension to `backend/agent/src/agent/import_boundary_test.go`: **check 6**, a per-file, zero-hop, deny-by-default AST scan over the archetype tree, plus its own forbidden table, allowlist, resolver, and vacuity floor. Two new files: `src/chat/doc.go`, `src/cmd/chat/main.go`. No other Go file changes. `go.mod`/`go.sum` are byte-frozen by D-1's nine shipped assertions; `Makefile`/`.golangci.yml` are unchanged by the proposal's own out-of-scope rows (no test asserts them — the constraint is D-1's decision, not an existing assertion).

### Amendment to D-1 (user decision, Engram #3741, 2026-08-23)

`go.opentelemetry.io/contrib/bridges/otelslog` is **permitted** to the archetype and joins the allowlist. D-1's original denial list copied Layer 2's forbidden rows, but ADR 0005 § D3 marks otelslog ❌ for L1/L2 and ✅ for L3 `coding` and `cmd/` (`adr:243`); under the ADR 0009 § D2 substitution that grant belongs to the archetype, and doc 0005:241's forbidden closure names only "the OTel SDK", never the bridge. `otel/sdk/…` and `…/exporters/…` stay denied at the archetype (❌ at L3, `adr:242`) and are permitted only in `src/cmd/chat`, which is outside the scanned set.

## Architecture Decisions

### AD-1 — Check 6's signature, resolution, and walk scope

**Choice**:

```go
func TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault(t *testing.T)
```

Follows check 1's four-segment shape `Test<Subject>_<Scope>_<Behavior>_<Expectation>` (`TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault`, `:277`). Check 4 (`:581`) uses a three-segment form with no expectation clause; the four-segment form is chosen because the expectation — `DenyByDefault` — is the load-bearing property here. The check calls `t.Parallel()`, as all five existing checks do (`:278`, `:323`, `:464`, `:582`, `:728`).

- **Tree resolution** — never an assumed working directory: a static map keyed by import path, mirroring `layer2SiblingTreeDirs` (`:624-631`), rooted in `runtime.Caller(0)` via the existing `layer2AgentPackageDir` (`:545-552`):

```go
func chatArchetypeDirs(t *testing.T) map[string]string {
	t.Helper()
	srcDir := filepath.Dir(layer2AgentPackageDir(t))
	return map[string]string{modulePath + "/src/chat": filepath.Join(srcDir, "chat")}
}
```

`modulePath + "/src/cmd/chat"` is **deliberately absent** — that absence *is* scenario 2's "the same import inside the composition root passes" (D-1). The map's doc comment records D-4's test-closure asymmetry (`layer3handoff` admissible to a future *test* closure; CH-02 decides) so CH-02 amends rather than invents.

- **Walk scope: recursive** (`filepath.WalkDir` per map entry), skipping directories named `testdata` (the toolchain's own exclusion). **This is a stated design-level amendment to proposal D-6**, whose mechanism says "for every `.go` file directly inside the archetype tree" (D-7 likewise treats subdirectory walking as a hypothetical). Reason for amending: checks 4/5 scan single flat packages by design, but the archetype tree grows subpackages at CH-02, and a top-level-only scan would silently release the first subpackage from the guard — exactly the escape class this milestone exists to close. A reviewer can overturn this back to D-6's literal top-level scan; doing so re-opens the subpackage escape at CH-02 and must assign that gap an owner.
- **Production files only**: skip `strings.HasSuffix(name, "_test.go")`, exactly as check 4's `layer2ProductionSourceFiles` does (`:566`; check 5's own helper, which deliberately *includes* test files, is `:675-693`). At CH-01 the tree ships zero test files; scenario 3's no-second-guard clause is asserted over the merged tree in step H, not by this check.
- **Parse shape**: `parser.ParseFile(fset, path, nil, parser.ImportsOnly)`, verbatim from checks 4/5 (`:592`, `:742`); parse error → `t.Fatalf`. Import paths read as `strings.Trim(imp.Path.Value, `"`)` (`:597`). Reads bytes, resolves nothing — the D-1 mechanism that keeps `go.mod` untouched.

### AD-2 — The two tables, final rule strings, and helper reuse policy

Every rule string below is **final text**, compliant with D-3's immutable citation contract: ADR 0005 § D1 row 3 (or § D3) plus the ADR 0009 § D2 substitution, written out in full. No rule string references a transient SDD id (`D-4`, change slugs), because these strings ship in Go source and the SDD artifacts move to `openspec/changes/archive/` at archive.

**Forbidden table**:

```go
var chatArchetypeForbiddenPrefixes = []struct {
	prefix string
	rule   string
}{
	{modulePath + "/src/ai/openaicompat", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: the archetype reaches row 1 only through the src/ai contract — the vendor adapter subtree is denied by name"},
	{modulePath + "/src/agenttest", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: rows 1–2 are importable as production contract surface only — Layer 2's test substrate is not part of that surface"},
	{modulePath + "/src/apptest", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: Layer 2's packaged test kit is consumer-proof machinery, not row 2's production surface"},
	{modulePath + "/src/layer3handoff", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: production never imports the consumer-proof tree — a future test closure decides its own admission (CH-02)"},
	{modulePath + "/src/cmd", "ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: nothing below the composition root imports src/cmd"},
}
```

**Allowlist** (deny-by-default; every entry cites its authority in place, as `allowedProductionPrefixes:219-244` requires):

```go
var chatArchetypeAllowedPrefixes = []string{
	modulePath + "/src/agent", // ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: rows 1–2 are importable
	modulePath + "/src/ai",    // same row; the vendor adapter subtree carved back out by chatArchetypeForbiddenPrefixes above
	"go.opentelemetry.io/otel/trace",     // ADR 0005 § D3 (adr:240): tracing API permitted at every layer
	"go.opentelemetry.io/otel/attribute", // ADR 0005 § D3 (adr:241)
	"go.opentelemetry.io/otel/codes",     // ADR 0005 § D3 (adr:241)
	"go.opentelemetry.io/otel/semconv",   // forced closure of otel/trace (see allowedProductionPrefixes' own semconv entry)
	// ADR 0005 § D3 (adr:243): otelslog is permitted at L3 coding, read as any
	// Layer 3 archetype under ADR 0009 § D2 — user decision, Engram #3741.
	// Authorised ahead of its first import by that decision; a deliberate,
	// recorded deviation from Layer 2's same-commit amendment convention.
	"go.opentelemetry.io/contrib/bridges/otelslog",
}
```

- **Ordering invariant: forbidden first.** The allowlist admits `…/src/ai` as a prefix; an allowlist-first pass would silently admit `…/src/ai/openaicompat` — the exact load-bearing reason recorded at `:170-176` (S-AGP-029). `otel/sdk/…` and `…/exporters/…` need no forbidden rows: no allowlist prefix reaches either (`otelslog`'s prefix does not cover `otel/sdk`), so default denial catches them — and scenario 2 requires only file-naming, no framing clause.
- **Helper policy**: check 6 must NOT reuse `matchForbidden` (`:757-764`) — it is hardcoded to Layer 2's `forbiddenPrefixes`, whose `:201-202` rows are exactly what the proposal keeps *out* of the archetype's table; sharing would re-couple the tables and break AD-4. A new 6-line sibling `chatArchetypeMatchForbidden(importPath)` over the chat table is the honest duplication cost; drift risk is bounded because the tables are *supposed* to differ. The generic `matchesPrefix(importPath, allowlist)` (`:789-796`) IS reused for the allowlist — table-parameterized, exact semantics. `isAllowed` (`:779-787`) is NOT reused: its `_test`-suffix retry handles a `go list -deps -test` synthesis shape that a literal source import can never be.

### AD-3 — The message contract (the one thing this change fixes once)

Because check 6 walks recursively (AD-1), `filepath.Base` cannot name a file unambiguously once CH-02 adds subpackages — `chat.go` in `src/chat/` and in `src/chat/port/` would render identically, and this change freezes the message contract in `CPB`. The file is therefore named by its **tree-relative path** from the declared root (`filepath.Rel(rootDir, path)`; a `Rel` error → `t.Fatalf`). For a root-level file this renders identically to `filepath.Base`.

**Literal format string** for the deny-by-default branch (`relPath` is the `filepath.Rel` result):

```go
t.Errorf("%s directly imports %q, which the chat archetype's deny-by-default allowlist does not admit\n"+
	"  rule: deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2) — "+
	"this path is neither the Go standard library nor a package this archetype's allowlist admits.\n"+
	"  No forbidden prefix names it, and that is not a licence to add it: adding a dependency "+
	"needs its own recorded design decision in chatArchetypeAllowedPrefixes.",
	relPath, importPath)
```

**Worked example** — planted `src/chat/scratch_forbidden_module.go` importing `github.com/cachicamas/backend/workspace_syncer/src/domain` (a real Go package of that module — `backend/workspace_syncer/src/domain/clone.go` exists):

```
scratch_forbidden_module.go directly imports "github.com/cachicamas/backend/workspace_syncer/src/domain", which the chat archetype's deny-by-default allowlist does not admit
  rule: deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2) — this path is neither the Go standard library nor a package this archetype's allowlist admits.
  No forbidden prefix names it, and that is not a licence to add it: adding a dependency needs its own recorded design decision in chatArchetypeAllowedPrefixes.
```

A future subpackage file renders as `port/conversation.go directly imports …` — unambiguous within the declared root.

Clause provenance, explicit: from check 1 (`:296-303`) it carries forward "this path is neither the Go standard library nor a package … admits" (the deny-by-default framing) and "No forbidden prefix names it, and that is not a licence to add it: adding a dependency needs its own recorded design decision" (the anti-missing-allowlist-entry clause); from check 4 (`:598-601`) it carries forward the `"%s directly imports %q"` file-naming shape — deliberately rendered through the tree-relative path rather than check 4's `filepath.Base`, because checks 4/5 scan flat directories where base names are unique by construction and check 6 does not. It drops check 4's family clause (no family list here) and check 1's § D1 *row 2* citation, substituting row 3 plus the ADR 0009 § D2 substitution per D-3. Forbidden-branch hits use check 4's simpler shape with the same relative rendering: `t.Errorf("%s directly imports %q\n  rule: %s", relPath, importPath, rule)`. `sdd-spec` asserts the three properties (file, path, framing) as separate `CPB` scenarios against these strings' properties, not their bytes.

### AD-4 — D-6's load-bearing absence: confirmed

Confirmed with evidence, not challenged. The forbidden-first ordering (AD-2, mandatory for the `openaicompat` carve-out) means any row matching the planted path fires *before* the allowlist branch. Layer 2's `:201-202` rows would match `github.com/cachicamas/backend/workspace_syncer/src/domain` and emit `rule: ADR 0005 § D1 row 2: no package of another backend module` — a named-prefix citation, not "the deny-by-default rule" scenario 1's second `Then` requires. The only mechanism that makes scenario 1's message true is those rows' absence from the archetype's table; R-07 stays enforced because no allowlist prefix reaches either module. This is why AD-2 forbids sharing `matchForbidden`.

### AD-5 — Vacuity floor, reconciled with D-7's per-directory clause, and how the floor is proven

D-7 says "if the scan walks subdirectories, the floor is per directory, following S-AGP-070's per-pattern rule". Reconciliation, stated rather than silently overridden: S-AGP-070's mechanism (`:313-321`, check 5's per-tree form `:734-739`) protects **declared inputs** — a mistyped pattern or map entry must fail by name. Under a recursive walk, subdirectories are *discovered*, not declared; a mistyped subdirectory simply does not exist and is never visited, so a per-subdirectory floor can never fire on the failure S-AGP-070 exists to catch, and would false-positive on legitimately Go-free directories. The floor therefore binds each **declared map entry** — exactly the granularity S-AGP-070 protects, and at CH-01 exactly one: `WalkDir` error → `t.Fatalf` naming the directory and import path; zero collected `.go` files under an entry → `t.Fatalf` "the scan would pass vacuously", mirroring check 4's `:586-588`. A `doc.go`-only tree satisfies the floor with one file — file-count-based, never dependency-count-based (D-6's deferral reason). **Proof**: one recorded run with the map entry mistyped to `…/chatx` → fatal captured in `apply-progress.md`, then reverted (TDD step F).

### AD-6 — Standard-library classification: `go list std`, never a heuristic

The AST scan sees literal stdlib imports (checks 1–2 never do — `listNonStdlibDeps` filters them, `:808-813`). A new helper runs `exec.Command("go", "list", "std")` once per check-6 run, builds a `map[string]struct{}`, and admits exact members — toolchain-owned classification, honoring the file's own recorded warning against maintained lists/heuristics (`:805-807`). **On subprocess failure the helper calls `t.Fatalf`** with the command's stderr, matching every existing `go list` failure path in this file (`:282`, `:329`, `:468`). Fixed argv, no user input, same shape as the file's three existing `exec.Command("go", …)` sites. Evaluation order per import: forbidden table → stdlib set → allowlist → **deny** (forbidden absolutely first; forbidden and stdlib are disjoint, so the invariant is preserved trivially).

### AD-7 — Package-comment amendment row

Two edits to the package comment, matching its own convention (enumeration updated in place, history appended unstruck, `:116-128`): (1) item 1's "FIVE checks" (`:8`) becomes "SIX", appending check 6 to the enumeration with CH-01 attribution; (2) a new appended section `# CH-01 amendment — the chat archetype's own boundary (check 6)` recording: the per-file deny-by-default scan over `src/chat/...` (recursive, AD-1's stated amendment); the D-3 citation rule; `src/cmd/chat` outside the scanned set by design; the deliberate absence of `database_administrator`/`workspace_syncer` rows and why (AD-4); the otelslog grant (`adr:243` under the ADR 0009 § D2 substitution, Engram #3741); and the zero-hop gap with its owner (transitive closure check → CH-02, D-6).

### AD-8 — The two new files

- **`src/chat/doc.go`** — header `// CH-01.1 — …`, `package chat`, doc comment only, zero declarations. The comment MUST state: the chat archetype occupies the Layer 3 *position* (ADR 0009 § D2), created by CH-01.1 of doc 0005 (`chat-archetype-contract` R-CHT-007), ships empty of policy until CH-02; the position's forbidden closure so a reader meets it in the package itself — OTel SDK and exporters anywhere but the composition root (ADR 0005 § D3, `adr:242`), any `src/cmd/…` package, any Go package of `database_administrator`/`workspace_syncer` (network only, R-07), Layer 2's test substrate (`agenttest`, `apptest`, `layer3handoff`) and the vendor adapter (`openaicompat`); the permitted OTel surface (the API packages and the otelslog bridge, `adr:243`, under the ADR 0009 § D2 substitution); and check 6 in `src/agent/import_boundary_test.go` as the deny-by-default enforcement.
- **`src/cmd/chat/main.go`** — `package main`, doc comment, `func main() {}` with a body comment. The comment MUST state: this is the archetype's composition root, the only package of the archetype permitted to install the OTel SDK and exporters — ADR 0005 § D3's `cmd/` column (`adr:242`), which is generic and needs no substitution, together with CH-00 seam row 8; it is deliberately outside check 6's scanned set; **CH-04.1 owns its wiring** — `main` is intentionally empty until then, and running the binary doing nothing is a recorded decision (proposal D-5, fail-fast stub rejected). Where the comment references the substitution convention at all, it writes "ADR 0009 § D2" in full — bare "§ D2" after an ADR 0005 citation would read as ADR 0005's own § D2 section.

### AD-9 — The R-AGP-006 analogue: D-7's table plus two executable proofs

Proposal success criterion 7 requires **both** halves; the executable proofs supplement D-7's five-row table, they do not replace it. At verify: (0) **re-verify D-7's table** row by row against the post-apply tree (Layer 1 guard's closed pattern list `src/ai/import_boundary_test.go:59-61`; checks 1/3/4 scoped to `layer2Pattern` `:150`; check 2's pattern set `:165-168`; check 5's directory map `:627-630`; the `src/cmd` forbidden row `:204` now armed). Then:

1. **Hunk audit**: `git diff 419a4291 -- backend/agent/src/agent/import_boundary_test.go` — every hunk lies in the package comment or in wholly-new `chatArchetype*`/check-6 declarations; zero hunks inside the five existing `Test*` funcs, their tables (`forbiddenPrefixes`, `allowedProductionPrefixes`, `allowedTestPrefixes`, `networkOrFilesystemPackages`, `forcedStandardLibraryImporters`, both family lists) or shared helpers. Plus `git diff 419a4291 -- backend/agent/src/ai/import_boundary_test.go` → empty. This diffs guard **bodies**, per `releasing-a-file-from-its-freeze-removes-its-reviewer`.
2. **Re-plant bite**: after the extension lands, plant `src/agent/scratch_os_exec.go` containing `import _ "os/exec"` → checks 3 and 4 RED, recorded, file deleted (TDD step G). Checks 1/2/5 armed-ness rests on proofs 0–1 (bodies byte-identical to shipped, already-bitten code). Insertion point for all new declarations: after check 5 (`:755`), before `matchForbidden` — checks stay contiguous, shared helpers stay last.

## Data Flow

```
chatArchetypeDirs ──► WalkDir(root, skip testdata, skip _test.go) ──► per .go file:
  parser.ImportsOnly ──► relPath = filepath.Rel(root, path) ──► per import path:
    chatArchetypeForbiddenPrefixes?  ──hit──► Errorf(relPath, path, row rule)
    go-list-std set?                 ──hit──► admit
    matchesPrefix(allowlist)?        ──hit──► admit
    otherwise                        ───────► Errorf(relPath, path, deny-by-default AD-3 message)
Floor: walk error OR zero files per declared root ──► Fatalf
```

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/chat/doc.go` | Create | AD-8; `package chat`, doc only |
| `backend/agent/src/cmd/chat/main.go` | Create | AD-8; `package main`, empty documented `main` |
| `backend/agent/src/agent/import_boundary_test.go` | Modify | Check 6 + two tables + resolver + files helper + stdlib helper + floor + comment amendment (~140–160 lines, vs. proposal's 135 estimate — AD-6's stdlib helper is the delta) |
| `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` | Modify | `0005:981` checklist row ticked; `0005:3` status line → **2 of 12** (proposal in-scope item 7, success criterion 11) |
| `openspec/changes/cachicamas-chat-package-scaffold/design.md` | Create | This document |

Spec deltas (`agent-package-scaffold` per D-2; new `chat-package-boundary`) are `sdd-spec`'s deliverable, shaped by AD-3's contract.

## Testing Strategy — TDD sequencing (strict; every run recorded)

**Run shape (corrected this round — Engram #3739)**: the repo root's `go.work` (`go.work:3-7`) uses all three backend modules, so in workspace mode a sibling module's packages import without a `require` in `backend/agent/go.mod`, and the build list is the union of all used modules' requirements — `go.opentelemetry.io/otel/sdk v1.44.0` is importable from `backend/agent` because `backend/database_administrator/go.mod:15` requires it. Both planted scratches therefore **build** (verified by probe: `import _ ".../workspace_syncer/src/domain"` and `import _ "go.opentelemetry.io/otel/sdk/trace"` each compile, exit 0), and a full `./...` run does reach the guard. The scoped command below is used for the recorded bites anyway — for **focus** (fast, and it isolates the check under test from the rest of the suite), never out of necessity. All scratch files use **blank imports** (`import _ "…"`) so they compile under any run shape without a use site; `parser.ImportsOnly` and `go list -deps` both still see them. Note separately: the `go.mod` byte-freeze still binds **committed** code — `go mod tidy` would add a `require` for a real committed import, drifting the frozen file.

Scoped bite command: `go test -race -count=1 -run TestChatArchetype -v ./src/agent/` (from `backend/agent`).

| Step | Action | Expected | Recorded in `apply-progress.md` |
|---|---|---|---|
| A | CH-01.1: write both files; `go build ./...` + `go build ./src/cmd/chat` | green | build output |
| B (RED-1) | Plant `src/chat/scratch_forbidden_module.go` (`import _ ".../workspace_syncer/src/domain"`); write check 6 + tables; scoped run | FAIL with AD-3's message — read for all three properties | full output |
| C | Delete scratch; scoped run | PASS | output |
| D (RED-2a) | Plant `src/chat/scratch_otel_sdk.go` (`import _ "go.opentelemetry.io/otel/sdk/trace"`); scoped run | FAIL naming file + path | full output |
| E (2b, negative control) | Plant identical import in `src/cmd/chat/scratch_otel_sdk.go`; scoped run | check 6 PASS with root scratch present | full output — the control is evidence in its own right |
| F (floor) | Delete scratches; mistype map entry to `…/chatx`; scoped run; revert | FATAL naming the mistyped dir | full output |
| G (AD-9.2) | Plant `src/agent/scratch_os_exec.go` (`import _ "os/exec"`); run `go test -race -count=1 -run 'TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage|TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport' -v ./src/agent/`; delete | both RED | full output |
| H (scenario 3) | Clean tree: `go clean -testcache`; `cd backend/agent && go test -race -count=1 ./...` (wall-clock); `make lint`; `gofmt -l` empty; AD-9's proofs 0–1; **scenario 3 second clause**: `git ls-files -- backend/agent/src/chat backend/agent/src/cmd` outputs exactly `backend/agent/src/chat/doc.go` and `backend/agent/src/cmd/chat/main.go` (zero `_test.go`, no second guard), and `git status --porcelain -- backend/agent/src/chat backend/agent/src/cmd` is empty (no untracked leftovers) | all green/empty/exact | outputs + wall-clock |

`make all` is never run. A `(cached)` result is not evidence.

## Threat Matrix

Reviewed against `references/threat-matrix.md`: **all rows N/A.** No routing, VCS/PR automation, executable-file classification, or process-integration boundary changes. The only subprocess added is test-scope `exec.Command("go", "list", "std")` — fixed argv, zero user input, identical in shape to the file's three existing `go list` sites; documentation-path, git-selection, commit, push, and PR rows have no subject here.

## Migration / Rollout

No migration. Rollback per proposal: single-PR revert before CH-02; partial revert (check 6 alone) available; packages-alone revert unavailable (the floor requires the tree).

## Open Questions

None. All forks were settled by D-1…D-7 (Engram #3725), the explore Resolutions table, and the otelslog decision (Engram #3741); this design binds their Go shapes and records its two stated amendments (AD-1's recursive walk over D-6's "directly inside"; #3741's allowlist entry over D-1's original OTel list).
