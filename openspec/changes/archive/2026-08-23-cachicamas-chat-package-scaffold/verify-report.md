```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:8076e26eed6906e0b4672b827340b143aa0e96a318e2ea2f3136c1f0cf93cde6
verdict: fail
blockers: 1
critical_findings: 1
requirements: 12/13
scenarios: 51/53
test_command: cd backend/agent && go clean -testcache && go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:a6ceba57c9c2bbccddec29208b4f2b6317f09343f8e3fa81a86d5d8c0fccd6c5
build_command: cd backend/agent && go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: `cachicamas-chat-package-scaffold` (CH-01, Wave 0 of doc 0005)
**Branch**: `feat/chat-archetype-wave0-ch01` · **HEAD**: `61aeca83` · **Merge base**: `419a4291`
**Mode**: Strict TDD (guard-test adaptation)
**Method**: every claim re-executed in this session. No `go test -overlay` was used anywhere — check 6 reads the real tree via `filepath.WalkDir` and the guard file shells out to `git`/`go list`, so overlay would report a false green. All plants were real files, all reverted; the worktree is clean.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 21 |
| Tasks complete | 21 |
| Tasks incomplete | 0 |

`apply-progress.md` and the Engram tasks artifact both state "20/20". `grep -cE '^- \[x\]' tasks.md` = **21** (1.1–1.4, 2.1–2.10, 3.1, 4.1–4.4, 5.1–5.2). Undercount only; every task is in fact checked.

### Build & Tests Execution

**Build**: PASSED — `go build ./...` exit 0, empty output.

**Tests**: PASSED — full uncached race suite, **170 s wall clock** (`2:50.24 total`), exit 0.

```text
$ cd backend/agent && go clean -testcache && go test -race -count=1 ./...
ok  .../src/agent                                12.604s
ok  .../src/ai/openaicompat                      169.196s
?   .../src/chat                                 [no test files]
?   .../src/cmd/chat                             [no test files]
(all 15 packages ok; 0 fail)
go test -race -count=1 ./...  65.63s user 8.35s system 43% cpu 2:50.24 total
```

Genuinely uncached: cache cleared first, 170 s measured against the ~170 s honest baseline. `src/chat` and `src/cmd/chat` reporting `[no test files]` independently confirms zero `_test.go` under either tree.

**Lint**: `make lint` → `0 issues.` `make all` was NOT run.
**Coverage**: not applicable — the change ships no behavioural code; its subject is an import guard.

### Spec Compliance Matrix — every `S-CPB` scenario

Evidence key: **[re-run]** = I reproduced it in this session; **[read]** = verified by reading the cited target.

| Scenario | Result | Evidence |
|---|---|---|
| S-CPB-001 | COMPLIANT | `git ls-files` → `src/chat/doc.go`; declares `package chat` [re-run] |
| S-CPB-002 | COMPLIANT | `go doc .../src/chat` emits only the doc comment, zero declarations [re-run] |
| S-CPB-003 | COMPLIANT | `go build ./src/cmd/chat` exit 0; `func main()` body holds one comment, no statement [re-run] |
| S-CPB-004 | COMPLIANT | `doc.go:1-29` carries all six required clauses incl. the otelslog grant + guard name [read] |
| S-CPB-005 | COMPLIANT | `main.go:1-21` carries composition root, `adr:242` SDK grant, scanned-set exclusion, CH-04.1 wiring, recorded empty `main`, "ADR 0009 § D2" written in full [read] |
| S-CPB-006 | COMPLIANT | `chat`/`main` match directory basename derivation [read] |
| **S-CPB-010** | **NON-COMPLIANT** | Clause 1 holds (no such test exists). **Clause 2 fails**: `main.go` never records the Go language rule. See CRITICAL-1 |
| S-CPB-011 | COMPLIANT | Recorded once at apply; not committed as a test [read] |
| S-CPB-012 | COMPLIANT | `import_boundary_test.go:855` carries the `…/src/cmd` row [read] |
| S-CPB-020 | COMPLIANT | `git ls-files -- backend/agent/src/chat backend/agent/src/cmd` = exactly `doc.go`, `main.go` [re-run] |
| S-CPB-021 | COMPLIANT | Same listing filtered for `_test.go` → empty [re-run] |
| S-CPB-022 | COMPLIANT | `git diff --name-only 419a4291` → `import_boundary_test.go` is the only file under `src/agent/` [re-run] |
| S-CPB-023 | COMPLIANT | Package comment `:134-191` carries every required clause incl. recursion, whole-table deviation, zero-hop gap [read] |
| S-CPB-030 | COMPLIANT | `:845-855` — five rows, each annotated with its rule [read] |
| S-CPB-031 | COMPLIANT | `:893-905` — every entry carries its `adr:NNN` in place [read] |
| S-CPB-032 | COMPLIANT | Every archetype-scoped rule string names ADR 0005 § D1 row 3 (or § D3) **and** ADR 0009 § D2. Grep over `:841-905` for `"ADR 0005` lines lacking `ADR 0009 § D2` → none. Layer 2's own row-2 strings at `:361`/`:406` correctly do not carry it [re-run] |
| S-CPB-033 | COMPLIANT | **[re-run]** planted `src/chat/scratch_openaicompat.go` → FAIL at `:1093` (forbidden-table branch), citing the vendor-adapter rule, **not** deny-by-default. Forbidden-first ordering proven |
| S-CPB-034 | COMPLIANT | `:848-854` records the production/test asymmetry and names CH-02 [read] |
| S-CPB-035 | COMPLIANT | No `otel/sdk` or `exporters` row exists; `:886-892` states default denial is the mechanism [read] |
| S-CPB-036 | COMPLIANT | `:904` present, `:900-903` cites `adr:243` + the substitution [read] |
| S-CPB-037 | COMPLIANT | **[re-run]** planted `import _ "go.opentelemetry.io/contrib/bridges/otelslog"` in `src/chat` → **PASS**. The grant is genuinely reachable, not shadowed. (Apply asserted this by reading; I proved it executably.) |
| S-CPB-038 | COMPLIANT | `:866-884` states the deviation over the whole table with declaration-vs-incremental reasoning and both rejected alternatives [read] |
| S-CPB-039 | COMPLIANT | Only two committed files exist and neither imports anything → no entry is exercised [re-run] |
| S-CPB-040 | COMPLIANT | **[re-run]** `src/chat/scratch_forbidden_module.go` → FAIL at `:1102` naming the file. **Strengthened**: I additionally planted `src/chat/port/scratch_sub.go`, which reported `port/scratch_sub.go` — proving `filepath.Rel` rather than `filepath.Base`. Apply only ever bit at the root, where the two shapes are indistinguishable |
| S-CPB-041 | COMPLIANT | Same block names `"github.com/cachicamas/backend/workspace_syncer/src/domain"` in full [re-run] |
| S-CPB-042 | COMPLIANT | Same block frames deny-by-default and cites no named forbidden prefix [re-run] |
| S-CPB-043 | COMPLIANT | Neither module appears in `chatArchetypeForbiddenPrefixes`; `:831-840` states why [read] |
| S-CPB-044 | COMPLIANT | **[re-run]** drafted both rows in → failure moved to `:1095` and cited `rule: DEFEAT-TEST ROW: …`, i.e. a **named prefix**; property 3 genuinely FAILED. Guard reverted, `git diff --stat` back to 362/6. The absence is checked, not assumed |
| S-CPB-045 | COMPLIANT | All three properties read off **one** failure block [re-run] |
| S-CPB-046 | COMPLIANT | `:1084-1087` derives via `filepath.Rel` with a fatal on error; `:1001-1010` states why a base name is inadmissible [read] |
| S-CPB-050 | COMPLIANT | **[re-run]** `src/chat/scratch_otel_sdk.go` → FAIL naming file + path via the default-deny branch. Probe `go build ./...` exit 0 first |
| S-CPB-051 | COMPLIANT | **[re-run]** identical import placed **only** in `src/cmd/chat/` → **PASS**. Negative control holds |
| S-CPB-052 | COMPLIANT | `:930-938` records the exclusion as deliberate and cites `adr:242` [read] |
| S-CPB-053 | COMPLIANT | **[re-run]** the `src/chat/port/…` and `src/chat/a/b/c/…` plants were both caught, proving recursion; `testdata` and `_test.go` skips verified by plant (both correctly unscanned); `:991-999` records the amendment [re-run] |
| S-CPB-060 | COMPLIANT | **[re-run]** both halves fired: walk error at `:1073`, and — **not bitten by apply** — I moved `doc.go` aside to fire the zero-file half at `:1076` (`the scan would pass vacuously`). Restored |
| S-CPB-061 | COMPLIANT | **[re-run]** root mistyped to `…/src/chatx` → FATAL naming the mistyped dir and the archetype import path. Reverted |
| S-CPB-062 | COMPLIANT | `:1031-1045` states the file-count basis and the `doc.go`-only reasoning [read] |
| S-CPB-070 | COMPLIANT | See "Guard-body proof" below — stronger than a hunk classification |
| S-CPB-071 | COMPLIANT | `git diff 419a4291 -- backend/agent/src/ai/import_boundary_test.go | wc -c` → **0** [re-run] |
| S-CPB-072 | COMPLIANT | **[re-run]** baseline PASS → planted `src/agent/scratch_os_exec.go` → checks 3 **and** 4 both RED (`:662`, `:567`×2) → removed → both PASS again |
| S-CPB-073 | COMPLIANT | Both trees are siblings of `src/agent/`, neither nested [re-run] |
| S-CPB-074 | COMPLIANT | The `…/src/cmd` row is byte-identical at the shifted location (base `:204` → post `:267`) and now names an existing tree [re-run] |
| S-CPB-075 | COMPLIANT | All five D-7 rows re-resolved against the **post-apply** tree — see below |
| S-CPB-080 | COMPLIANT | `git diff 419a4291 --` for `go.mod`, `go.sum`, `Makefile`, `.golangci.yml` → **0 bytes each** [re-run] |
| S-CPB-081 | COMPLIANT | Full uncached suite green; `src/agent` (which hosts those assertions) ok in 12.604s [re-run] |
| S-CPB-082 | COMPLIANT | `:1080` uses `parser.ImportsOnly`; `:1012-1017` states it resolves nothing. Independently proven: my adversarial plant of the non-existent path `…/src/ai-evil` was still evaluated and denied [re-run] |
| S-CPB-090 | COMPLIANT | Full failing output recorded for every bite, in `apply-progress.md` and reproduced here [re-run] |
| S-CPB-091 | COMPLIANT | Scoping is recorded as focus/isolation only; no evidence claims an unscoped run would fail at build. Independently true — my plants all built (`go build ./...` exit 0) [re-run] |
| S-CPB-092 | COMPLIANT | `git ls-files` = exactly the two files; `git status --porcelain` over both trees → **empty** (now genuinely empty post-commit, where apply could only show staged-clean) [re-run] |
| S-CPB-093 | COMPLIANT | 170 s uncached, exit 0, recorded above [re-run] |
| **S-CPB-094** | **PARTIAL** | `make lint` → 0 issues [re-run]. `gofmt -l` scoped to this change's 3 files → empty [re-run]. **Whole-module `gofmt -l` prints 15 files** — all confirmed byte-unchanged vs `419a4291`. See WARNING-1 |
| S-CPB-095 | COMPLIANT | Every plant used `import _ "…"`; all compiled with no use site [re-run] |
| S-CPB-096 | COMPLIANT | Both halves: sibling-module blank import built (exit 0), OTel SDK blank import built (exit 0); the `go mod tidy` drift claim is recorded separately, not conflated [re-run] |

**Compliance summary**: 51 COMPLIANT / 1 PARTIAL / 1 NON-COMPLIANT of 53.

**NFRs**: NFR-CPB-001 COMPLIANT (`t.Parallel()` at `:1047`; roots resolved from `runtime.Caller`, not cwd; passes under `-race`). NFR-CPB-002 COMPLIANT (failures name file + path + rule). NFR-CPB-003 COMPLIANT (`go list std` at `:964`, fatal with subprocess stderr at `:969`).

### Guard-body proof — no existing guard was disarmed

Rather than classify hunks, I reconstructed the merge base by deleting **only** the two inserted regions and diffing:

```text
$ git show 419a4291:backend/agent/src/agent/import_boundary_test.go > base.go
$ sed '820,1112d;133,191d' backend/agent/src/agent/import_boundary_test.go > reconstructed.go
$ diff base.go reconstructed.go
8c8, 18c18, 21,24c21,28   ← three lines inside the package doc comment (FIVE→SIX enumeration) only
```

Every other byte of the file is identical. That covers **all five existing checks**, `forbiddenPrefixes`, `allowedProductionPrefixes`, `allowedTestPrefixes`, `networkOrFilesystemPackages`, `forcedStandardLibraryImporters`, both family lists, `layer2Pattern`, `layer2TestOnlyTreePatterns`, `layer2SiblingTreeDirs`, and every shared helper (`matchForbidden`, `matchesPrefix`, `listNonStdlibDeps`, `layer2AgentPackageDir`). Zero hunks fall inside checks 1–5, their tables, or the helpers — proven by byte identity of the whole remainder, not by inspection of hunk boundaries.

`backend/agent/src/ai/import_boundary_test.go` is byte-unchanged (0-byte diff).

### D-7's five-row table, re-resolved against the post-apply tree

Insertion shifts: merge-base `L` → post-apply `L+4` for `24 < L ≤ 128`, `L+63` for `L > 128`.

| D-7 row | Merge-base cite | Post-apply | Verified |
|---|---|---|---|
| Layer 1 forward guard `layer1Patterns` | `src/ai:59-61` | unchanged (file frozen) | closed list `src/ai/…`, `src/agenttest/…`, `src/handoff/…`; no pattern reaches `src/chat` or `src/cmd/chat` |
| Layer 2 checks 1, 3, 4 — `layer2Pattern` | `:150` | `:213` | `const layer2Pattern = modulePath + "/src/agent/..."` — byte-identical |
| Layer 2 check 2 — `layer2TestOnlyTreePatterns` | `:165-168` | `:228-231` | byte-identical |
| Layer 2 check 5 — directory map | `:627-630` | `:690-693` | byte-identical |
| `…/src/cmd` forbidden row | `:204` | `:267` | byte-identical, now **armed** |

Supporting cites also re-resolved: sibling-nesting comment `:159-164`→`:222-227`; check 4's floor `:586-588`→`:649-651`. All five rows hold at the post-apply tree.

### Frozen files and merged-tree cleanliness

```text
$ git diff 419a4291 -- backend/agent/go.mod backend/agent/go.sum backend/agent/Makefile backend/agent/.golangci.yml
(0 bytes for each of the four)
$ git ls-files -- backend/agent/src/chat backend/agent/src/cmd
backend/agent/src/chat/doc.go
backend/agent/src/cmd/chat/main.go
$ git status --porcelain -- backend/agent/src/chat backend/agent/src/cmd
(empty)
```

Zero `_test.go` under either tree. `git diff --name-only 419a4291` lists 13 files — 3 Go/doc-comment files, doc 0005, and the 9 SDD artifacts. **No scratch file, no stray `chat` build binary, nothing else.**

### Spec delta audit — four deltas

| Check | agent-package-scaffold | chat-archetype-contract | ai-observability-boundary |
|---|---|---|---|
| Cited target text resolves verbatim | YES (`:35`, `:41`) | YES (`:145-155`, `:226`, `:19`, `:256`) | YES (`:228`, `:26` anchor, `:220-234` table) |
| Mints no identifier | YES — 8 ids in delta, 0 absent from target's 67 | YES — 12 in delta, 0 absent from 64 | YES — 11 in delta, 0 absent from 54 |
| Renumbers nothing | YES | YES | YES (touches no identifier at all) |
| Modified requirement's scenario set complete | YES — S-AGP-001…005, matches target exactly | YES — S-CHT-060/061/062 and S-CHT-120/121/122 match target exactly | N/A |
| "Byte-identical" claims true | YES — S-AGP-001/002/004/005 verified byte-equal | YES — S-CHT-060/120/121/122 verified byte-equal | YES — Item cell unchanged; other rows untouched |
| Link depth correct for the promoted location | YES | YES | YES — `../../../docs/adr/…` matches the target file's own convention |
| Promotion instruction explicit | YES | YES for both the amendment note **and** the prose corrections ("The archive executor MUST apply them in the same promotion") | **PARTIAL** — see WARNING-2 |

**Scenario-id collision sweep across the whole `openspec/` tree** (the AG-23 failure mode): `grep -rEno '\b(R|S|NFR)-CPB-[0-9]+' openspec/ docs/ backend/` outside the change directory returns **only** citations inside `import_boundary_test.go`. No promoted spec uses the `CPB` prefix; `openspec/specs/chat-package-boundary/` does not exist, so the new capability collides with nothing. No duplicate `S-CPB` id within the new spec (`sort | uniq -d` → empty). The three deltas mint nothing, so they cannot collide under a sibling requirement.

### Coverage, both directions

Computed by set difference, not by row count:

```text
declared scenarios: 53   referenced in tasks: 53
DECLARED but not reachable from any task: (none)
referenced in tasks but not declared:     (none)
```

All 10 `R-CPB` requirements are referenced by at least one task. Four tasks cite no scenario id — 2.4 (the GREEN half of the bite sequence, serving R-CPB-010), 4.4 (budget gate), 5.1 and 5.2 (doc 0005 bookkeeping). Those are process/bookkeeping steps, legitimately outside the spec's scenario space.

### Doc 0005 bookkeeping

Checklist row `:981` ticked `[x]`, "closed by CH-01.2" text unchanged. Status line `:3` reads **2 of 12**, and its former clause "No Layer 3 archetype exists yet on disk" is replaced.

**Sibling-sentence sweep**: grepped doc 0005 for `does not exist|do not exist|exists yet|no Layer 3|nothing on disk|no archetype|no code`. Three further hits, none falsified — `:13` is a generic authoring constraint about code that does not exist at HEAD, `:64` is R-08's schema-ownership rule, `:501` is a Gherkin `When` about cancelling a nonexistent turn identity. **No other sentence in doc 0005 is falsified by the two packages existing.**

### Adversarial pass against check 6 — what I actually tried

| Attempt | Result |
|---|---|
| Aliased import (`evil "…"`) | **CAUGHT** — `imp.Path.Value` is alias-independent |
| Dot import (`. "…"`) | **CAUGHT** |
| Prefix-boundary probe `…/src/ai-evil` against the `…/src/ai` entry | **CAUGHT** — `matchesPrefix` requires exact match or `prefix+"/"`, so it is segment-safe |
| Prefix-boundary probe `go.opentelemetry.io/otel/tracex` against the `otel/trace` entry | **CAUGHT** |
| `go.opentelemetry.io/otel/sdk` (bare, no subpath) | **CAUGHT** |
| Build-tagged file (`//go:build ignore`) | **CAUGHT** — `parser.ParseFile` ignores build constraints, so the check is stricter than the compiler |
| Subpackage `src/chat/port/…` | **CAUGHT**, reported as `port/scratch_sub.go` |
| Deep nesting `src/chat/a/b/c/…` | **CAUGHT**, reported as `a/b/c/adv_f.go` |
| Multiple violations in one file | **CAUGHT** — all four reported (`t.Errorf`, not `Fatalf`) |
| `_test.go` inside `src/chat/` | **ESCAPES** — by design, skipped at `:1066`. Not a live hole: R-CPB-003 forbids `_test.go` at CH-01 and there are none. The test closure is owned by CH-02 and the asymmetry is recorded at `:848-854` |
| `src/chat/testdata/*.go` | **ESCAPES** — skipped at `:1060`. Not a real escape: the Go tool ignores `testdata` entirely, so such a file is never compiled into the archetype |
| Symlinked directory into `src/chat/` | **ESCAPES the check** — `filepath.WalkDir` does not follow symlinks. **But it is not a real escape**: I verified `go list ./src/chat/...` returns only `src/chat` and `go build ./...` exits 0 without compiling it, so the toolchain does not follow it either. Code the compiler never sees cannot enter the archetype's closure |
| Allowlist branch genuinely reachable | Planted `src/agent`, `src/ai`, `otel/trace`, `otel/attribute`, `otel/codes`, `otelslog` → **PASS**. The allowlist branch is exercised, answering the spec's own "a branch no evidence reaches" concern |

**I could not construct an import that should be denied and is not.** The one structural gap outside the scan — everything under `backend/agent/src/cmd/chat/`, including any future subpackage — is unscanned by construction (R-CPB-006), and no requirement or comment bounds it to the composition-root package itself. Recorded as SUGGESTION-2.

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|---|---|---|
| R-CPB-001 | Implemented | Both packages exist, declare nothing, build |
| R-CPB-002 | **Partially implemented** | No test ships (correct); the mechanism is **not** recorded in `main.go` — CRITICAL-1 |
| R-CPB-003 | Implemented | One guard extended in place; no second guard; zero `_test.go` added |
| R-CPB-004 | Implemented | Both tables complete, forbidden-first ordering proven by bite |
| R-CPB-005 | Implemented | All three properties off one block, defeat test confirms the absence is checked |
| R-CPB-006 | Implemented | Exclusion + negative control both re-run |
| R-CPB-007 | Implemented | Both floor halves fired |
| R-CPB-008 | Implemented | All three proofs; guard bodies proven byte-identical |
| R-CPB-009 | Implemented | Four files 0-byte diff |
| R-CPB-010 | Implemented | RED recorded for every bite, green on clean tree, all scratches removed |

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| AD-1 — `src/cmd/chat` outside scanned set | Yes | `chatArchetypeDirs` single entry; negative control proves it |
| AD-2 — separate forbidden table + matcher | Yes | `chatArchetypeMatchForbidden` is its own function; rationale at `:907-914` |
| AD-3 — tree-relative message | Yes | Proven by the subpackage plant |
| AD-5 — file-count floor | Yes | Both halves bitten |
| AD-6 — forbidden → stdlib → allowlist → deny | Yes | Ordering proven by the openaicompat bite |
| AD-7 — package-comment amendment | Yes | FIVE→SIX plus the 59-line CH-01 section |
| AD-9 — three proofs | Yes | Strengthened here by whole-remainder byte identity |
| D-6 amendment to recursion | Yes, stated | Recorded at `:991-999`; recursion proven by two nested plants |

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | YES | `apply-progress.md` carries a full TDD Cycle Evidence table |
| Guard-adaptation recorded | YES | Recorded explicitly, not silently applied |
| RED confirmed | YES | 7 distinct REDs re-executed in this session, all reproduced |
| GREEN confirmed | YES | Clean-tree PASS + negative-control PASS + full suite reproduced |
| Triangulation adequate | YES | 5+ distinct code paths: forbidden-table, deny-by-default, stdlib, allowlist-admit, walk-error floor, zero-file floor |
| Safety net | YES | Checks 3+4 verified passing before the re-plant, and again after |

**Assertion quality**: no test-file assertions were authored by this change beyond check 6 itself, which is a guard whose failure paths I fired directly. No tautology, no orphan-empty assertion, no ghost loop — the `for` loop over `files` is protected by the zero-file floor at `:1075`, which I fired to confirm it cannot silently iterate zero times.

### Issues Found

**CRITICAL**

**CRITICAL-1 — `S-CPB-010` is false at its own merge state; `R-CPB-002`'s recording obligation is unmet.**
`backend/agent/src/cmd/chat/main.go:1-22` — the package comment never records the mechanism. `R-CPB-002` (`specs/chat-package-boundary/spec.md:45`) states the property "is enforced by the **Go language**, not by a guard: a `package main` is not importable", and `S-CPB-010` (`:51`) requires that mechanism to be "recorded in `backend/agent/src/cmd/chat/main.go`'s own comment". It is not. The comment covers the composition root, the `adr:242` OTel grant, the check-6 exclusion and the empty `main`, but says nothing about importability.

Demonstrated by:
```text
$ grep -niE "import|not importable|language" backend/agent/src/cmd/chat/main.go
11:// backend/agent/src/agent/import_boundary_test.go's check 6
12:// (TestChatArchetype_..._DenyByDefault):
13:// sweeping it would deny the very import this position exists to grant.
```
Three hits, all about check 6's scanned set; none about importability.

Root cause — the clause fell between two tasks. `tasks.md:1.2` enumerates what the comment must carry (composition root, SDK grant, exclusion, CH-04.1 wiring, substitution wording) and maps to `R-CPB-001, S-CPB-003, S-CPB-005, S-CPB-006` — it never mentions the Go-language rule and never maps to `S-CPB-010`. `tasks.md:1.4` maps to `S-CPB-010` and asserts the property is "stated in 1.2's comment" — but 1.2 was never asked to state it. Both tasks are checked, and the requirement was satisfied by neither.

This blocks archive: promoting `chat-package-boundary` today would publish a scenario that is false against the tree it is promoted from — precisely the defect class this change's own *Untemporal-invariant register* exists to close. Fix is one sentence in `main.go`; it changes no behaviour and touches no frozen file.

**WARNING**

**WARNING-1 — `S-CPB-094`'s `gofmt -l` clause is unsatisfiable at the merged tree as written.**
`specs/chat-package-boundary/spec.md:234` says "when `make lint` and `gofmt -l` run, then the first exits 0 with no reported issue and **the second prints nothing**." The scenario names no operand. Under the whole-module reading it is false:
```text
$ cd backend/agent && gofmt -l . | wc -l
15
$ for f in $(gofmt -l .); do echo "$f -> $(git diff 419a4291 -- "$f" | wc -c) bytes"; done
(all 15 → 0 bytes)
```
All 15 are byte-unchanged against `419a4291` — pre-existing gofmt-version drift the change is explicitly forbidden to repair ("No other Go file changes"). Scoped to this change's three files, `gofmt -l` is empty and `make lint` is 0 issues. Apply recorded this honestly. The defect is in the scenario's scope, not the code: pin the operand (e.g. "`gofmt -l` over the files this change touches") before archive, or the promoted scenario archives false for the same reason CRITICAL-1 does.

**WARNING-2 — the `ai-observability-boundary` delta's normative change carries no archive-executor imperative.**
`specs/ai-observability-boundary/spec.md:54-62` — the delta's entire subject is one table row, and it is presented under `## MODIFIED content — the Out of scope row at spec.md:228` with the declarative lead-in "The row becomes:". There is **no** `## MODIFIED Requirements` section in this delta at all, so an archive executor that promotes by scanning requirement blocks would find nothing to do and could silently drop the whole delta. By contrast the same file's amendment note (`:66`) does carry "The archive executor MUST append…", and the `chat-archetype-contract` delta's out-of-requirement prose corrections carry "The archive executor MUST apply them in the same promotion" (`:87`). Mitigated but not closed by `V-1`/`V-2`/`V-5`, which would detect the drop only after archive. Recommend adding an explicit imperative before the row.

**SUGGESTION**

**SUGGESTION-1 — task count mis-stated as 20/20.** `tasks.md` contains 21 checked rows (`grep -cE '^- \[x\]'` = 21); `apply-progress.md` and the Engram tasks/apply artifacts all say "20/20". Every task is genuinely complete; only the total is wrong.

**SUGGESTION-2 — the composition-root subtree is unbounded, and nothing says so.** `chatArchetypeDirs` excludes `…/src/cmd/chat` (correctly, per R-CPB-006), but the exclusion is whole-subtree: a future `src/cmd/chat/internal/wire/` would be unscanned by every guard in the module. `R-CPB-006` and the comment at `:930-938` both justify the exclusion by the composition root's own OTel grant, which is a property of the root package, not of arbitrary subpackages beneath it. Worth an owner (CH-04.1 is the natural one) before wiring lands.

**SUGGESTION-3 — one apply transcript line number is off by one.** `apply-progress.md`'s S-CPB-061 transcript cites `import_boundary_test.go:1074` for the WalkDir fatal; the committed file emits it from `:1073` (confirmed by re-running the bite). Cosmetic, but it means that transcript was captured against a tree that is not the committed one.

**Not findings, stated plainly so the absence is a result rather than a silence**: the four frozen files are byte-identical; `src/ai/import_boundary_test.go` is byte-identical; no existing check, table or helper was touched; the merged tree carries no scratch file or stray binary; no `S-CPB` id collides anywhere in `openspec/`; no delta mints or renumbers an identifier; every "byte-identical" claim in every delta is true; all five D-7 rows re-resolve correctly post-apply; no sentence in doc 0005 besides the one already repaired is falsified; and I could not defeat check 6.

### Verdict

**FAIL** — one CRITICAL: `S-CPB-010`'s second clause is unmet in `backend/agent/src/cmd/chat/main.go`, so promoting the spec would archive a scenario false at its own merge state. Everything else verified green under re-execution: 51 of 53 scenarios COMPLIANT, 1 PARTIAL, guard bodies proven byte-identical, every bite reproduced, and no adversarial escape found.

---

## Post-verify addendum — findings resolved, and one audit weakness recorded

Added by the orchestrator after the verify round, against the tree at `61aeca83` plus the corrections below.

### CRITICAL-1 — resolved

`backend/agent/src/cmd/chat/main.go`'s package comment now records the mechanism `S-CPB-010`'s second clause requires: a `package main` cannot be imported, the rule is the Go language's, and that is why this change ships no test asserting it. Re-run after the edit: `gofmt -l` on the file prints nothing; `cd backend/agent && go clean -testcache && go test -race -count=1 ./...` exits 0 in **2:54.76** wall clock — consistent with the two prior uncached runs (2:52.95 and apply's 172.18s), so not a cache hit.

### WARNING-1 and WARNING-2 — resolved

`S-CPB-094`'s `gofmt -l` operand is pinned to this change's own three Go files, with the 15 pre-existing drifting files named, proven byte-unchanged against `419a4291`, attributed to a toolchain version bump, and marked as forbidden to repair here by `R-CPB-009`. `make lint` and `S-CPB-093`'s suite run stay module-wide, because both are genuine obligations of this change rather than artefacts of its scope. The `ai-observability-boundary` and `chat-archetype-contract` deltas now carry explicit `## PROMOTION INSTRUCTION` sections with numbered BEFORE/AFTER text, so an archive executor that scans only requirement blocks cannot silently drop a normative change that is not shaped like one.

### Recorded, not repaired — the coverage audit's own unsoundness

`tasks.md`'s coverage table maps scenarios to tasks as **ten ranges** (`S-CPB-001…006` through `S-CPB-090…096`), never as explicit per-scenario entries. Four scenarios are named by no individual task: `S-CPB-090` and `S-CPB-095` appear only inside a range, and `S-CPB-039` and `S-CPB-091` are claimed by two tasks with neither singly accountable.

Verified by command: `grep -n "S-CPB-090" tasks.md` returns only the range row at `:92`; `grep -oE 'S-CPB-[0-9]+…[0-9]+' tasks.md | sort -u` returns all ten ranges.

**All four are COMPLIANT under re-execution**, so this is not a live defect in the shipped change. What it is, is a defect in the audit: this verify round reported *declared 53 / referenced 53, none missing*, and could not have reported anything else. Expanding a range manufactures the very references the set difference then looks for, so the check is empty by construction regardless of how many members no task mentions.

That matters because it is the same mechanism as this round's one CRITICAL, which survived two green tasks — task 1.2 enumerated `main.go`'s required comment content and mapped to `S-CPB-003/005/006`, and task 1.4 mapped to `S-CPB-010` while asserting 1.2 had already delivered it. A clause fell between two tasks, each assuming the other carried it, and a range-based coverage table is structurally unable to see that.

`tasks.md` is left unedited: its tasks are applied and checked, and rewriting closed work would falsify its own history. The convention itself — that a coverage mapping must name every identifier explicitly, and that a referenced set must be built only from identifiers a task names inline — belongs to whoever owns the task-graph method, not to this change.
