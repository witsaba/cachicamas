# Apply Progress: CH-01 — Scaffold the archetype package and make its import boundary bite

**Change**: `cachicamas-chat-package-scaffold` · **Mode**: Strict TDD (guard-test adaptation — see below) · **Date**: 2026-08-23
**Branch**: `feat/chat-archetype-wave0-ch01` · **Merge base**: `419a4291` · **Worktree**: `cachicamas-worktrees/feat-chat-archetype-wave0-ch01`

All commands below were run from `backend/agent` unless a `cd` is shown. Every scoped bite command is `go test -race -count=1 -run TestChatArchetype -v ./src/agent/`, used **for focus and isolation of the check under test, and for that reason only** (R-CPB-010, S-CPB-091) — the repo's `go.work` puts every used module's requirements in the build list, so an unscoped `./...` run reaches the guard too; no evidence below claims otherwise.

## Strict TDD adaptation for a guard task (recorded, not silently applied)

`strict-tdd.md`'s generic cycle assumes a test that references *production code that does not exist yet*. CH-01.2 is a `[guard]` leaf: the spec's own **Evidence discipline** section states guard requirements "are RED-first by construction. The closing condition is the recorded red run against a deliberate violation, followed by green" — the "production code under test" is not a function but the **archetype tree's own import statements**, exercised by planting scratch files. This is the same convention this exact guard file used to build checks 1–5 historically (its own package comment records AG-22 being *"discovered by `sdd-verify`"* via exactly this plant/bite/revert cycle). The RED/GREEN/TRIANGULATE mapping below follows that established, project-specific convention rather than the generic function-first shape.

### TDD Cycle Evidence

| Task | Test file / subject | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1–1.4 (CH-01.1, `[mechanical]`) | `src/chat/doc.go`, `src/cmd/chat/main.go` | Build/check evidence | N/A (new files) | N/A — exempt from red-green by design (tasks.md Phase 1 header, spec "Evidence discipline") | ✅ `go build ./...` + `go build ./src/cmd/chat` exit 0 | N/A | N/A — doc-only / empty-`main` files, nothing to refactor |
| 2.1 (RED-1) | `TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault` (check 6, written complete per design AD-1…AD-9) | Unit (Go test) | ✅ baseline: check 6 written + both new files present → scoped run PASS (0 violations) before any scratch is planted | ✅ `src/chat/scratch_forbidden_module.go` planted → scoped run FAIL, all 3 message properties in one block | — (revert happens across 2.2–2.4, not immediately) | — | — |
| 2.2 (defeat test) | same | Unit | — | ✅ drafted forbidden-table row → re-run FAILS the deny-by-default *property* (S-CPB-042), proving the omission is checked not assumed | ✅ table reverted, scratch (2.1's) still present | — | — |
| 2.3 (ordering bite) | same | Unit | — | ✅ `scratch_openaicompat.go` planted (2.1's scratch still present too) → FAILS on forbidden-prefix rule, not deny-by-default | — | ✅ 2nd distinct code path exercised (forbidden-table branch, not allowlist branch) | — |
| 2.4 | same | Unit | — | — | ✅ both `src/chat` scratches deleted → scoped run PASS | — | — |
| 2.5 (RED-2a) | same | Unit | — | ✅ `scratch_otel_sdk.go` in `src/chat` → FAIL naming file + path (deny-by-default branch, distinct from forbidden-table branch) | — | ✅ 3rd distinct code path (default-deny with an OTel path, no forbidden row involved) | — |
| 2.6 (negative control) | same | Unit | — | (2.5's RED already recorded) | ✅ `src/chat` scratch removed, identical import planted **only** in `src/cmd/chat` → scoped run PASS — proves the composition root is outside the scanned set | ✅ 4th distinct case: same import path, different (excluded) root | — |
| 2.7 (vacuity floor) | `chatArchetypeDirs` | Unit | — | ✅ map entry mistyped to `…/chatx` → FATAL naming the mistyped dir + import path | ✅ entry reverted → scoped run PASS | ✅ 5th distinct code path (the floor itself, not the per-import evaluation) | — |
| 2.8–2.9 (proofs 0–1) | package-level (no code change) | Read/diff evidence | — | N/A — evidence proofs, not RED/GREEN | N/A | N/A | N/A |
| 2.10 (re-plant bite, AD-9.2) | `TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage`, `TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport` (pre-existing checks 3+4) | Unit | ✅ both pass on the clean, post-apply tree before planting (see 2.8/4.1) | ✅ `src/agent/scratch_os_exec.go` (`import _ "os/exec"`) → **both** RED | ✅ scratch deleted → **both** PASS again | — (existing checks, not this task's own triangulation) | — |
| Whole file | `import_boundary_test.go` | — | — | — | — | — | ✅ `gofmt -w` + `gofmt -l` clean; `go vet` clean; `make lint` 0 issues — formatting/lint pass with tests green throughout |

### Test Summary
- **Total distinct RED runs captured**: 7 (2.1, 2.2's drafted-table run, 2.3, 2.5, 2.7, 2.10×2 checks)
- **Total distinct PASS/GREEN runs captured**: 6 (baseline pre-scratch, 2.4, 2.6, 2.7 revert, 2.10 revert, final full-suite `./...`)
- **Layers used**: Unit (Go `testing`) only — no integration/E2E harness applies to a backend import guard (NFR — see Work Unit Evidence)
- **Approval tests**: None — no refactoring of pre-existing behavior; checks 1–5's bodies are proven byte-identical to the merge base (proof 1) rather than approval-tested
- **Pure functions created**: `chatArchetypeMatchForbidden`, `chatArchetypeDirs`, `chatArchetypeStandardLibraryPackages` (the last is not pure — shells to `go list std` — but is deterministic and hermetic per NFR-CPB-001)

## Work Unit Evidence (mandatory, all modes)

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test -race -count=1 -run TestChatArchetype -v ./src/agent/` from `backend/agent` — PASS on the clean tree (0.14s–1.9s per run across the sequence below); every bite run recorded in full below |
| Runtime harness command/scenario and exact result | N/A — this is a backend unit-test import guard; no compose/browser/E2E harness applies (proposal's own Suggested Work Units table: "N/A — backend unit-test guard, no compose/browser harness applies") |
| Rollback boundary | Pre-merge: discard branch. Post-merge: single-PR revert — no `go.mod`/`go.sum`/`Makefile`/`.golangci.yml`, no schema, no wire, no runtime behavior is touched (confirmed empty diffs, Phase 3 below) |

---

## Phase 1 — CH-01.1 package scaffold (check evidence, not red-green)

### 1.1 / 1.2 — files created
- `backend/agent/src/chat/doc.go` — `package chat`, doc comment only, zero declarations.
- `backend/agent/src/cmd/chat/main.go` — `package main`, doc comment, `func main() {}` with a body comment only.

Both siblings of `backend/agent/src/agent/`, never nested (confirmed by path: `backend/agent/src/chat`, `backend/agent/src/cmd/chat`, vs. `backend/agent/src/agent`).

### 1.3 — build evidence

```
$ cd backend/agent && gofmt -l src/chat/doc.go src/cmd/chat/main.go
(empty — clean)
$ go build ./...
(exit 0, no output)
$ go build ./src/cmd/chat
(exit 0, no output)
```

### 1.4 — R-CPB-002 demonstration (not committed as a test)

Planted `backend/agent/src/chat/scratch_import_cmd_root.go`:
```go
package chat

import _ "github.com/cachicamas/backend/agent/src/cmd/chat"
```

```
$ go build ./...
src/chat/scratch_import_cmd_root.go:3:8: import "github.com/cachicamas/backend/agent/src/cmd/chat" is a program, not an importable package
exit=1
```

Toolchain rejection confirmed — the property is a Go-language guarantee, not a guard. File deleted immediately after; confirmed via `go build ./...` → exit 0 again and `git status --porcelain -- src/chat src/cmd` showing only the two shipped files (untracked at that point).

---

## Phase 2 — CH-01.2 guard TDD sequence

Baseline (check 6 written complete, both tables complete, no scratch planted yet):

```
$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
=== RUN   TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
=== PAUSE TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
=== CONT  TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
--- PASS: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.39s)
PASS
ok  	github.com/cachicamas/backend/agent/src/agent	1.902s
```

`gofmt -l src/agent/import_boundary_test.go` → empty. `go vet ./src/agent/...` → exit 0.

### 2.1 — RED-1 (S-CPB-040…046, S-CPB-096 half 1)

Planted `backend/agent/src/chat/scratch_forbidden_module.go`:
```go
package chat

import _ "github.com/cachicamas/backend/workspace_syncer/src/domain"
```

Probe build (S-CPB-096 half 1 — proves the workspace-mode reachability claim, distinct from the go.mod-drift claim):
```
$ go build ./...
(exit 0, no output)
```

Scoped run:
```
$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
=== RUN   TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
=== PAUSE TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
=== CONT  TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
    import_boundary_test.go:1102: scratch_forbidden_module.go directly imports "github.com/cachicamas/backend/workspace_syncer/src/domain", which the chat archetype's deny-by-default allowlist does not admit
          rule: deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2) — this path is neither the Go standard library nor a package this archetype's allowlist admits.
          No forbidden prefix names it, and that is not a licence to add it: adding a dependency needs its own recorded design decision in chatArchetypeAllowedPrefixes.
--- FAIL: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.14s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/agent	0.433s
FAIL
```

All three S-CPB-005 properties read off this **one** failure block (S-CPB-045): file (`scratch_forbidden_module.go`, tree-relative), path (`"github.com/cachicamas/backend/workspace_syncer/src/domain"`), deny-by-default framing (no named forbidden prefix cited). Byte-for-byte match with design AD-3's worked example.

Line-count tracking (design's ~140–160-line estimate): the actual net insertion into `import_boundary_test.go` is **362 lines added, 6 lines changed** (`git diff --numstat`, see Phase 4). This exceeds the estimate — the doc-comment burden to satisfy every `S-CPB-023/034/035/038/043/046/052/053/060/062/082` per-declaration comment requirement, plus the ~59-line package-comment amendment section, is larger than the design's own estimate anticipated. Recorded plainly, not narrowed.

### 2.2 — Defeat test (S-CPB-044)

With 2.1's scratch still present, temporarily drafted a `database_administrator`/`workspace_syncer` row into `chatArchetypeForbiddenPrefixes`:
```go
// S-CPB-044 DEFEAT-TEST DRAFT ROW — TEMPORARY, MUST BE REVERTED.
{"github.com/cachicamas/backend/workspace_syncer", "ADR 0005 § D1 row 2: no package of another backend module"},
```

```
$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
=== RUN   TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
=== PAUSE TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
=== CONT  TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
    import_boundary_test.go:1095: scratch_forbidden_module.go directly imports "github.com/cachicamas/backend/workspace_syncer/src/domain"
          rule: ADR 0005 § D1 row 2: no package of another backend module
--- FAIL: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.36s)
FAIL
FAIL	github.com/cachicamas/backend/agent/src/agent	0.669s
FAIL
```

The failure now cites a **named forbidden prefix** instead of deny-by-default framing — property 3 (S-CPB-042) genuinely FAILS under this drafted table, proving the recorded absence of `database_administrator`/`workspace_syncer` rows is **checked**, not assumed. Table reverted to its recorded shape (row removed) immediately after this run.

### 2.3 — openaicompat ordering bite (S-CPB-033)

With 2.1's scratch still present, additionally planted `backend/agent/src/chat/scratch_openaicompat.go`:
```go
package chat

import _ "github.com/cachicamas/backend/agent/src/ai/openaicompat"
```

```
$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
    import_boundary_test.go:1102: scratch_forbidden_module.go directly imports "github.com/cachicamas/backend/workspace_syncer/src/domain", which the chat archetype's deny-by-default allowlist does not admit
          rule: deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2) — this path is neither the Go standard library nor a package this archetype's allowlist admits.
          No forbidden prefix names it, and that is not a licence to add it: adding a dependency needs its own recorded design decision in chatArchetypeAllowedPrefixes.
    import_boundary_test.go:1093: scratch_openaicompat.go directly imports "github.com/cachicamas/backend/agent/src/ai/openaicompat"
          rule: ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2: the archetype reaches row 1 only through the src/ai contract — the vendor adapter subtree is denied by name
--- FAIL: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.14s)
FAIL
```

The `scratch_openaicompat.go` line cites the **forbidden-prefix** rule, not the allowlist/deny-by-default rule — proving forbidden-first ordering carves `…/src/ai/openaicompat` out before the `…/src/ai` allowlist prefix would otherwise admit it. `scratch_openaicompat.go` deleted immediately after.

### 2.4 — GREEN (design step C)

Both `src/chat` scratches (`scratch_forbidden_module.go`, `scratch_openaicompat.go`) deleted.

```
$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
--- PASS: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.14s)
PASS
ok  	github.com/cachicamas/backend/agent/src/agent	1.456s
```
`src/chat/` confirmed to contain only `doc.go` at this point.

### 2.5 — RED-2a (S-CPB-050, S-CPB-096 half 2)

Planted `backend/agent/src/chat/scratch_otel_sdk.go`:
```go
package chat

import _ "go.opentelemetry.io/otel/sdk/trace"
```

Probe build (S-CPB-096 half 2):
```
$ go build ./...
(exit 0, no output)
```

Scoped run:
```
$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
    import_boundary_test.go:1102: scratch_otel_sdk.go directly imports "go.opentelemetry.io/otel/sdk/trace", which the chat archetype's deny-by-default allowlist does not admit
          rule: deny-by-default allowlist (ADR 0005 § D1 row 3, read as any Layer 3 archetype under ADR 0009 § D2) — this path is neither the Go standard library nor a package this archetype's allowlist admits.
          No forbidden prefix names it, and that is not a licence to add it: adding a dependency needs its own recorded design decision in chatArchetypeAllowedPrefixes.
--- FAIL: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.14s)
FAIL
```

FAILs naming the file (tree-relative) and the path, via the default-deny branch (no forbidden row exists for `otel/sdk`, by design — R-CPB-004).

### 2.6 — Negative control (S-CPB-051) — corrected sequence, recorded

**Correction discovered during apply**: tasks.md 2.6 and the proposal's bite table both say "check 6 PASSES with both scratch files present" / "root scratch present." A literal simultaneous-both-present state cannot produce a PASS: `src/chat/scratch_otel_sdk.go` (2.5's plant) is **inside** the scanned set and is a genuine violation for as long as it exists — the whole Go test reports FAIL if any file under a scanned root violates, regardless of what else is planted elsewhere. Confirmed empirically:

```
$ (both src/chat/scratch_otel_sdk.go AND src/cmd/chat/scratch_otel_sdk.go present)
$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
    import_boundary_test.go:1102: scratch_otel_sdk.go directly imports "go.opentelemetry.io/otel/sdk/trace", which the chat archetype's deny-by-default allowlist does not admit
          ...
--- FAIL: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.14s)
FAIL
```

This FAIL is entirely attributable to `src/chat`'s own scratch (the message names only `scratch_otel_sdk.go` at the `src/chat` root — `src/cmd/chat` is never walked, by construction of `chatArchetypeDirs`, so its file could not have produced this line regardless). Design step E's own literal phrasing — "check 6 PASS **with root scratch present**" — and spec S-CPB-051's own literal phrasing — "Given a scratch file in `.../src/cmd/chat/`... PASSES **with that file present**" — both use singular framing naming only the root's own file, consistent with this being the correct reading. Deleted `src/chat/scratch_otel_sdk.go`, kept only `src/cmd/chat/scratch_otel_sdk.go` (identical import), reran:

```
$ (only src/cmd/chat/scratch_otel_sdk.go present)
package main

import _ "go.opentelemetry.io/otel/sdk/trace"

$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
=== RUN   TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
=== PAUSE TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
=== CONT  TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault
--- PASS: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.14s)
PASS
ok  	github.com/cachicamas/backend/agent/src/agent	1.591s
```

**PASS recorded as evidence in its own right** — the identical import, only in the composition root, does not trip the guard: proves `src/cmd/chat` is genuinely outside the scanned set. Both otel scratch files deleted after; confirmed `src/chat/` = `{doc.go}`, `src/cmd/chat/` = `{main.go}`, scoped run PASS again.

### 2.7 — Vacuity floor bite (S-CPB-061)

Temporarily mistyped `chatArchetypeDirs`' map entry:
```go
// S-CPB-061 VACUITY-FLOOR BITE — TEMPORARY MISTYPE, MUST BE REVERTED.
return map[string]string{modulePath + "/src/chat": filepath.Join(srcDir, "chatx")}
```

```
$ go clean -testcache && go test -race -count=1 -run TestChatArchetype -v ./src/agent/
    import_boundary_test.go:1074: filepath.WalkDir("/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/feat-chat-archetype-wave0-ch01/backend/agent/src/chatx") error = lstat /Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/feat-chat-archetype-wave0-ch01/backend/agent/src/chatx: no such file or directory, want nil (chat archetype root github.com/cachicamas/backend/agent/src/chat)
--- FAIL: TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault (0.36s)
FAIL
```

FATAL (single message, no further loop output — confirms `t.Fatalf` halted the test immediately), naming both the mistyped directory and the archetype's import path. Entry reverted to `filepath.Join(srcDir, "chat")`; scoped run PASS confirmed again after revert.

### 2.8 — Proof 0: D-7 table re-verified row by row against the post-apply tree (S-CPB-075)

Line numbers re-resolved by reading the **current** file (1263 lines total, up from 907 at merge base):

| Row | Post-apply citation | Verified claim |
|---|---|---|
| Layer 1's forward guard | `src/ai/import_boundary_test.go:58-61` (`layer1Patterns`) | Closed, fully-qualified list: `src/ai/...`, `src/agenttest/...`, `src/handoff/...`. No pattern touches `src/chat` or `src/cmd/chat`; file byte-unchanged (Phase 2.9) |
| Checks 1, 3, 4 (Layer 2) | `src/agent/import_boundary_test.go:213` (`layer2Pattern` const), used at `:343` (check 1), `:529`/`:547` (check 3); check 4 uses `layer2AgentPackageDir`, untouched | Still scoped to `layer2Pattern` alone, deliberately unwidened |
| Check 2's pattern set | `layer2TestOnlyTreePatterns` at `:228`, used at `:388` | Unchanged — still only `src/apptest/...`, `src/layer3handoff/...` |
| Check 5's directory map | `layer2SiblingTreeDirs` at `:687` | Unchanged — still only `src/apptest`, `src/layer3handoff` |
| `…/src/cmd` forbidden row (Layer 2) | `:267` (shifted from merge-base `:204`) | Byte-identical text (confirmed below), now **armed** — `src/cmd/chat` exists on disk where nothing did before |

### 2.9 — Proof 1: hunk audit (S-CPB-070, S-CPB-071, S-CPB-073, S-CPB-074)

```
$ git diff --stat 419a4291 -- src/agent/import_boundary_test.go
 backend/agent/src/agent/import_boundary_test.go | 368 +++++++++++++++++++++++-
 1 file changed, 362 insertions(+), 6 deletions(-)

$ git diff 419a4291 -- src/ai/import_boundary_test.go | wc -l
0

$ git diff -U0 419a4291 -- src/agent/import_boundary_test.go | grep -n '^@@'
5:@@ -8 +8 @@
8:@@ -18 +18 @@
11:@@ -21,4 +21,8 @@
24:@@ -128,0 +133,59 @@
84:@@ -756,0 +820,293 @@ func TestLayer2_SiblingTrees_NoDirectForbiddenFamilyImport(t *testing.T) {
```

Classification of all 5 hunks:
- Hunks 1–3 (`-8+8`, `-18+18`, `-21,4+21,8`): all inside the package comment's item-1 enumeration (lines 1–129 range) — "FIVE"→"SIX" plus the sixth-check clause.
- Hunk 4 (`-128,0 +133,59`): pure 59-line insertion at old line 128 (immediately before `package agent_test`) — the new "# CH-01 amendment" section. Zero deletions.
- Hunk 5 (`-756,0 +820,293`): pure 293-line insertion immediately after check 5's closing brace (old line 756), before `func matchForbidden`. Zero deletions, zero context overlap with any existing function body, table, or helper.

Arithmetic cross-check: deletions 1+1+4=6 (matches "6 deletions" in `--stat`); insertions 1+1+8+59+293=362 (matches "362 insertions"). **Zero hunks fall inside any existing `Test*` function, table, or shared helper** — every hunk lies either in the package comment or in a wholly-new declaration, exactly as AD-9.1/S-CPB-070 requires.

`src/ai/import_boundary_test.go` diff against merge base: **empty** (0 lines) — Layer 1's forward guard is byte-unchanged (S-CPB-071).

Existing `…/src/cmd` row confirmed byte-identical at its shifted location:
```
$ diff <(git show 419a4291:backend/agent/src/agent/import_boundary_test.go | sed -n '204p') <(sed -n '267p' src/agent/import_boundary_test.go)
(no output — IDENTICAL)
```
Both print: `	{modulePath + "/src/cmd", "ADR 0005 § D1 row 2: Layer 2 must not import the composition root"},`

### 2.10 — Proof 2: re-plant bite (AD-9.2, S-CPB-072)

Planted `backend/agent/src/agent/scratch_os_exec.go`:
```go
package agent

import _ "os/exec"
```

```
$ go clean -testcache && go test -race -count=1 -run 'TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage|TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport' -v ./src/agent/
=== RUN   TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage
=== PAUSE TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage
=== RUN   TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport
=== PAUSE TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport
=== CONT  TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage
=== CONT  TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport
    import_boundary_test.go:662: scratch_os_exec.go directly imports "os/exec", which is in the forbidden "os" family
          rule: ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no network, filesystem or process package, reached directly or through any intermediary
--- FAIL: TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport (0.01s)
=== NAME  TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage
    import_boundary_test.go:567: Layer 2's production closure imports "os" via unexpected direct importer(s) [path/filepath os/exec] (vetted: [go.opentelemetry.io/otel/trace fmt])
          rule: ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no filesystem/environment package
    import_boundary_test.go:567: Layer 2's production closure imports "io/fs" via unexpected direct importer(s) [path/filepath os/exec] (vetted: [os internal/filepathlite])
          rule: ADR 0005 § D1 row 2: Layer 2 performs no I/O of its own — no filesystem package
--- FAIL: TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage (0.06s)
FAIL
```

**Both existing checks RED**, confirming checks 3 and 4's armed-ness independent of their code being byte-identical (proof 1). Note: `path/filepath` also appears as a newly-unvetted importer of `os`/`io/fs` — `os/exec`'s own transitive dependency on `path/filepath` pulls it into the closure for the first time; this is the guard correctly catching everything the plant newly exposes, not a spurious result. Scratch deleted; re-run confirms both PASS again:
```
--- PASS: TestLayer2_ProductionSources_NoDirectForbiddenFamilyImport (0.01s)
--- PASS: TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage (0.04s)
PASS
```

---

## Phase 3 — Non-code invariant (R-CPB-009, S-CPB-080)

```
$ cd backend/agent && for f in go.mod go.sum Makefile .golangci.yml; do git diff 419a4291 -- "$f" | wc -l; done
0
0
0
0
$ cd .. && git diff 419a4291 -- backend/agent/go.mod backend/agent/go.sum backend/agent/Makefile backend/agent/.golangci.yml | wc -l   # repo-root cross-check
0
```
All four paths byte-unchanged against the merge base with `origin/main`, confirmed both cwd-relative and repo-root-relative.

---

## Phase 4 — Final gate

### 4.1 — Zero scratch files; full uncached suite (S-CPB-081, S-CPB-093, R-CPB-010)

Housekeeping finding, recorded: several `go build ./...` probe-evidence runs above (1.4's demonstration was pre-plant; 2.1's and 2.5's probes were post-plant) leave a compiled binary named `chat` in `backend/agent/` (Go's default `go build ./...` output-to-cwd behavior for a `main` package when no `-o` is given). Found via `git status --porcelain` showing `?? backend/agent/chat`, confirmed via `file chat` → `Mach-O 64-bit executable arm64`, removed with `rm -f chat`. Not part of any deliverable; not committed.

```
$ go clean -testcache
$ go test -race -count=1 ./...
ok  	github.com/cachicamas/backend/agent/src/agent	12.319s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.231s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep	1.881s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest	1.670s
ok  	github.com/cachicamas/backend/agent/src/ai	3.927s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	2.091s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	169.519s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest	1.847s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	2.378s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	5.435s
?   	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures	[no test files]
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke	2.610s
ok  	github.com/cachicamas/backend/agent/src/apptest	2.337s
?   	github.com/cachicamas/backend/agent/src/chat	[no test files]
?   	github.com/cachicamas/backend/agent/src/cmd/chat	[no test files]
ok  	github.com/cachicamas/backend/agent/src/handoff	2.210s
ok  	github.com/cachicamas/backend/agent/src/layer3handoff	2.031s
```

**Wall-clock: real 172.18s** (`/usr/bin/time -p`; user 72.50s, sys 9.44s) — an uncached, genuine race-detector run (not a `(cached)`/sub-second result). `src/chat` and `src/cmd/chat` both report `[no test files]`, trivially confirming zero `_test.go` files under either tree. This run also exercises the nine shipped `go.mod`/`go.sum` merge-base-diff-empty assertions inside `src/agent`'s 12.319s.

### 4.2 — Lint and format (S-CPB-094)

```
$ gofmt -l src/chat/doc.go src/cmd/chat/main.go src/agent/import_boundary_test.go
(empty — clean, scoped to this change's 3 touched files)

$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

**Finding, recorded**: `gofmt -l .` over the **whole module** additionally lists 15 files this change never touches: `compaction_events_test.go`, `cost_events_test.go`, `delegation_events_test.go`, `envelope_test.go`, `event_registry_test.go`, `permission_events.go`, `permission_events_test.go`, `permission_protocol_test.go`, `protocol_events_test.go`, `reconstruction_test.go`, `scheduler.go`, `scheduler_test.go`, `scripted_tool_test.go`, `tool.go`, `tool_test.go`. Confirmed **pre-existing and unrelated to CH-01**:
```
$ git diff 419a4291 -- <all 15 files> | wc -l
0
```
byte-identical to the merge base. Sample (`tool.go`) shows a pure column-alignment difference on adjacent one-line method bodies (`gofmt -d` output), consistent with a toolchain/gofmt-version drift, not a content change. Per design's explicit "No other Go file changes" and the standing rule that `make all`'s mutating fmt step must never run, these files are **not** touched by this change. `make lint`'s 5-linter set (`govet`, `errcheck`, `staticcheck`, `unused`, `revive`) has no formatting linter enabled, so `make lint`'s "0 issues" and this pre-existing `gofmt -l` drift are independent facts, not contradictory.

### 4.3 — File listing exactness (R-CPB-003, R-CPB-010, S-CPB-020…022, S-CPB-039, S-CPB-092)

The two new files were staged (`git add`, not committed — apply does not commit) so `git ls-files` reflects the intended merge-state listing:
```
$ git add backend/agent/src/chat/doc.go backend/agent/src/cmd/chat/main.go
$ git ls-files -- backend/agent/src/chat backend/agent/src/cmd
backend/agent/src/chat/doc.go
backend/agent/src/cmd/chat/main.go
```
**Exactly** the two shipped files — no `_test.go`, no second guard.

```
$ git status --porcelain -- backend/agent/src/chat backend/agent/src/cmd
A  backend/agent/src/chat/doc.go
A  backend/agent/src/cmd/chat/main.go
```
**Note on S-CPB-092's literal "empty" wording**: that scenario is stated over "the merged tree," i.e. post-commit. Apply does not commit (explicit instruction: "I handle commits"), so the strongest pre-commit evidence obtainable is exactly this — both paths show a clean `A ` (staged addition of exactly the two expected files), with **no** `??` (untracked scratch) and **no** `M` (unexpected modification) for either path. Once the user commits, `git status --porcelain` for these two paths will read empty, as the scenario states.

```
$ git diff --stat 419a4291 -- backend/agent/src/agent/
 backend/agent/src/agent/import_boundary_test.go | 368 +++++++++++++++++++++++-
 1 file changed, 362 insertions(+), 6 deletions(-)
```
`import_boundary_test.go` is the only changed file under `src/agent`. Since the two new files import nothing, no archetype allowlist entry is exercised by a committed import at this merge state (S-CPB-039).

### 4.4 — Budget gate

```
$ git diff 419a4291 --numstat   # from repo root; includes the already-committed planning docs (5dd0e99f) + this apply's work, single-PR
362	6	backend/agent/src/agent/import_boundary_test.go
30	0	backend/agent/src/chat/doc.go
28	0	backend/agent/src/cmd/chat/main.go
2	2	docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md
189	0	openspec/changes/cachicamas-chat-package-scaffold/design.md
155	0	openspec/changes/cachicamas-chat-package-scaffold/explore.md
301	0	openspec/changes/cachicamas-chat-package-scaffold/proposal.md
59	0	openspec/changes/cachicamas-chat-package-scaffold/specs/agent-package-scaffold/spec.md
290	0	openspec/changes/cachicamas-chat-package-scaffold/specs/chat-package-boundary/spec.md
94	0	openspec/changes/cachicamas-chat-package-scaffold/tasks.md
```
Subtotal before `apply-progress.md` itself: **1510 insertions + 8 deletions = 1518 counted lines** (this numstat was captured *after* all `tasks.md` checkbox/annotation edits and both doc-0005 edits, so it is already final for those two files).

`apply-progress.md` itself (this document, wholly new, untracked at the time of writing): **482 lines** (`wc -l`), all additions.

**Final total: 1518 + 482 = 2000 counted lines** — recorded plainly against the 1000-line figure named at session start, not narrowed to look closer to it. This lands almost exactly at the top of the session's own ~1900–2000 projection (proposal/tasks.md Review Workload Forecast). `size:exception` was pre-authorised by the user for this session; this is not a stop condition. `verify-report.md` (not produced by this phase) will add further lines at `sdd-verify`.

---

## Phase 5 — Doc 0005 bookkeeping

Re-resolved by reading the file directly (both citations confirmed exact before editing):
- `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:981` — `- [ ] The package and its composition root exist, and the import guard is shown to bite on the forbidden closure — closed by CH-01.2` → `- [x] ...` (unchanged text otherwise).
- `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:3` — status line `**1 of 12**` → `**2 of 12**`.

---

## Deviations from design / findings summary

1. **Task 2.6 wording correction** (not a design defect — a task-list imprecision): "both scratch files present" cannot literally produce a PASS; the correct, self-consistent reading (matching design.md step E and spec S-CPB-051's own singular phrasing) is "the root's scratch alone present, `src/chat`'s own scratch removed first." Evidence recorded under both readings above (the FAIL with both present, then the true negative-control PASS with only the root's scratch present).
2. **Package-comment doc-comment volume** exceeded the design's ~140–160-line estimate (362 insertions total for the check-6 extension) because every `S-CPB` scenario requiring an in-place comment (10+ distinct properties) was honored literally and in full, per-declaration as well as at the package-comment level for the whole-table deviation and otelslog grant (redundant on purpose, for legibility from either entry point — NFR-CPB-002).
3. **Stray build artifact** (`backend/agent/chat`, a compiled binary) appeared after `go build ./...` probe-evidence runs; this is `go build`'s own default output-to-cwd behavior for a `main` package, not a defect in the change. Removed before every evidence-gathering step that mattered and before the final gate; never staged or committed.
4. **Pre-existing `gofmt` drift** in 15 unrelated files, confirmed byte-identical to the merge base — not caused by, and not repaired by, this change (design's "No other Go file changes"; `make all`'s mutating fmt step is never run per this session's explicit instruction).
5. **S-CPB-092's "git status --porcelain → empty"** is a post-commit property; apply does not commit. The two new files are staged (not committed) with a clean `A ` status and nothing else under either path — the strongest pre-commit evidence obtainable.

No design decision (AD-1…AD-9), proposal decision (D-1…D-7), or spec requirement was weakened, narrowed, or reopened. No dependency was added. No existing check, table, or shared helper was modified (proof 1, Phase 2.9).

## Status

**20/20 tasks complete.** All three code files (`src/chat/doc.go`, `src/cmd/chat/main.go`, `src/agent/import_boundary_test.go`) written and green. Doc 0005 bookkeeping applied. `tasks.md` marked `[x]` throughout. Ready for `sdd-verify`.
