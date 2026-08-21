```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:214acb72b9de9808055bfd72774faa68312a7a54b56e844460f2ec104cd042b5
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 17/17
scenarios: 69/69
test_command: cd backend/agent && go clean -testcache && make test
test_exit_code: 0
test_output_hash: sha256:efe16903cee0158a413395e38ba3e991dab8b8bf876a416c628f7818b5c9347b
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report — Round 3

**Change**: `cachicamas-agent-layer3-handoff` (AG-23 — Publish the Layer 3 readiness contract; Wave 6; 24 of 24 — Layer 2's exit)
**Version**: `agent-layer3-handoff` new full spec + 7 amendment/back-annotation deltas
**Mode**: Strict TDD (active)
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ag23-layer3-handoff`
**Branch**: `feat/agent-layer2-wave6-ag23` @ `e827766c`, base `main` @ `9c55eeda`, 22 commits
**Round**: 3

> **Method**. Nothing was accepted from the round-2 remediation's own record. The evidence suite was started on a tree first proved byte-identical to `HEAD` (`git status --porcelain` empty) and no plant existed at any moment while it ran. Every plant afterwards was reverted and byte-identity re-confirmed with `git diff --stat HEAD`. The read-only base checkout was confirmed untouched at the end.

---

## Round-1 and round-2 history (preserved for the archive trail)

**Round 1 — FAIL, 1 CRITICAL, 13 WARNING, 3 SUGGESTION.**
CRITICAL-1: `S-RUN-115` asserted defeat A "crashes on its first iteration"; the shipped code deadlocks. Remediated in 6 commits. W-3's closure (re-scoping `S-L3H-056` around a lint finding) became round 2's blocker.

**Round 2 — FAIL, 1 CRITICAL (C-1), 9 WARNING, 3 SUGGESTION.**
C-1: `S-L3H-056`/`NFR-L3H-B` narrowed around a `src/ai/` lint finding that does not exist, embedding a false claim that `sdd-archive` would promote into `openspec/specs/`. W-R2-1…W-R2-9 covered a mis-cited scenario, the vocabulary guard's plural/camelCase blind spot, a stale cross-reference, two wrong seam injection points, a structurally impossible clause in doc 0003, a stale gloss in `tasks.md`, and three recorded-not-fixed facts. Remediated in 3 commits (`4a3bd74f`, `9bb5c372`, `e827766c`).

Round 3 verifies each of those closures adversarially and re-verifies the whole milestone.

---

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 58 |
| Tasks complete | 53 |
| Tasks incomplete | 5 |

The 5 open tasks are **WU-10.1 … WU-10.5 (archive)** only — spec promotion and the change-folder move, which run *after* this gate by design. Every behavioural unit (WU-1…WU-9) and every remediation unit (R-1…R-5, round-2 R-1…R-2) is `[x]`, and each was checked against code state rather than against the record.

---

### Build & Tests Execution

**Build**: PASS — `cd backend/agent && make build` (`go build -trimpath ./...`), exit **0**.

**Tests**: PASS — **14 packages `ok`, 0 FAIL, exit 0, 172 s wall-clock**.

```text
cd backend/agent && go clean -testcache && make test     # make test = go test -race -v ./...
ok  .../src/agent                                         13.317s
ok  .../src/agenttest                                      2.231s
ok  .../src/agenttest/sweep                                1.677s
ok  .../src/agenttest/tracetest                            1.915s
ok  .../src/ai                                             5.552s
ok  .../src/ai/internal/retry                              2.309s
ok  .../src/ai/openaicompat                               170.955s
ok  .../src/ai/openaicompat/conformancetest                2.039s
ok  .../src/ai/openaicompat/openrouter                     2.892s
ok  .../src/ai/openaicompat/openrouter/conformance         6.474s
?   .../src/ai/openaicompat/openrouter/conformance/fixtures [no test files]
ok  .../src/ai/openaicompat/openrouter/internal/smoke      2.911s
ok  .../src/apptest                                        2.590s
ok  .../src/handoff                                        2.612s
ok  .../src/layer3handoff                                  2.934s
EXIT=0    DURATION=172s
```

**Uncachedness proven, not asserted.** `grep -c "(cached)"` over the full output = **0**. `grep -c "^--- FAIL"` = **0**. `src/ai/openaicompat` at **170.955 s** matches the documented ~170 s honest uncached figure. `git status --porcelain` was empty immediately before the run started and immediately after it finished; the run overlapped **no** plant.

**Coverage**: not measured — no coverage threshold is configured for this module; coverage is informational and never blocking under this skill.

---

## A. C-1's closure — confirmed by reading AND by measurement

### A.1 The spec text was genuinely restored

`specs/agent-layer3-handoff/spec.md`:

- **`NFR-L3H-B`, line 350** — "…`make lint` reporting zero issues **module-wide**, and `make build` exiting zero."
- **`S-L3H-056`, line 360** — "…`make lint` reports zero issues **module-wide** (measured after `golangci-lint cache clean`, since a stale cache has previously masqueraded as a finding in this repo), and build exits zero."

The false parenthetical ("module-wide lint carries one pre-existing finding … inside `src/ai/`") is **gone**. The requirement is now **stronger** than the round-1 text, not narrower.

### A.2 The claim measured true, by me

```text
$ bin/golangci-lint --version
golangci-lint has version 2.9.0 built with go1.26.0 from 72798d34 on 2026-02-10T23:13:22Z
$ bin/golangci-lint cache clean
cleaned
$ cd backend/agent && make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
LINT EXIT=0
$ git diff --stat main...HEAD -- .golangci.yml
(empty — byte-unchanged)
```

Whole-module, after a cache clean, with the pinned v2.9.0 binary and an unmodified config. **`0 issues.`, exit 0.** The restored claim is true.

### A.3 No sibling artifact repeats the nonexistent finding as a live claim

Grepped the whole repository for `pre-existing finding`, `module-wide lint carries`, `change's own packages`.

| Location | Reads as | Verdict |
|---|---|---|
| `apply-progress.md:203` (original apply G.2) | history, with an explicit **"Round-2 correction"** annotation stating "this apparent finding does not reproduce … it is not current evidence and MUST NOT be cited as such" | **Correct as history** |
| `apply-progress.md:267`, `:299` | inside the `# Round-1 remediation (sdd-verify)` section (line 226), recording what *that* pass did | **Correct as history**, superseded by the round-2 section below it |
| `apply-progress.md:342` | the round-2 correction record | Correct |
| Two archived changes (`agent-event-envelope`, `agent-run-driver`) | unrelated, pre-existing | Out of scope |

**No artifact states the nonexistent finding as present-tense fact. C-1 is CLOSED.**

---

## B. Round-2 finding-by-finding closure audit

Every row confirmed by opening the actual line or by running the command — never from the remediation's summary of itself.

| # | Round-2 finding | Round-3 verdict | Evidence obtained by this phase |
|---|---|---|---|
| **C-1** | `S-L3H-056`/`NFR-L3H-B` narrowed around a nonexistent lint finding | **CLOSED** | § A — spec text read, `0 issues.` measured, sibling sweep audited |
| **W-R2-1** | `agent-run-driver/spec.md:25` mis-cited `S-RUN-115` | **CLOSED, correction verified true** | Line 25 now names `S-RUN-117`. Opened `S-RUN-117` (line 70): it does assert, for "every non-panicking arm … its recorded event sequence is **identical** to the same script's at the merge base". The new citation resolves to a scenario that says what the row claims |
| **W-R2-2** | Vocabulary guard blind to plural/camelCase | **CLOSED for 2 of 3 inflection classes; 3rd disclosed** | Proved by uncontaminated probe — see § C. Both the fix and its stated residual limit verified |
| **W-R2-3** | `agent-loop-skeleton/spec.md:39` stale "identical discipline" cross-reference | **CLOSED, all three new clauses verified** | (i) the AG-21 half is reproduced byte-identically from `openspec/specs/agent-loop-skeleton/spec.md:295` on `main`; (ii) `NFR-L3H-B` does reserve `-count=1` for focused runs and require `go clean -testcache` for whole-module runs (spec lines 344-349); (iii) `openspec/specs/agent-run-driver/spec.md:6` **does** pin `make test` as `go test -race -v ./...`, so the stated reason is true |
| **W-R2-4** | Two of eleven seam injection points wrong | **CLOSED — all 11 injection points and all 11 defaults re-verified by me** | § D |
| **W-R2-5** | doc 0003's structurally impossible clause | **CLOSED in doc 0003 — but the identical clause survives in `decision.md:112`** | § E and **W-R3-1** |
| **W-R2-6** | `tasks.md:38` stale gloss | **CLOSED** | Row now reads `go clean -testcache && make test`; `grep -rn "count=1 \./\.\.\."` over `tasks.md` returns nothing |
| **W-R2-7** | Review budget | **RECORDED, grown to 2389** | § F |
| **W-R2-8** | `gofmt -l` non-empty | **RE-CONFIRMED pre-existing** | § F |
| **W-R2-9** | stdlib-intermediary gap | **Open by construction, disclosed** | Unchanged; compliant as `R-L3H-002` specifies ("by name") |
| **S-R2-1 / S-R2-2** | declined with reasons | Reasonable | Recorded, not re-litigated |
| **S-R2-3** | `len(l3hGuardDeniedVocabulary()) == 8` floor | **IMPLEMENTED and proven to bite** | § C.3 |

**Commit-scope audit.** `9bb5c372` (the guard widening) touched **exactly one file** — `generic_client_guard_test.go`, +62/−6. No guard-of-the-guard was relaxed in the same commit, so the widening kept its own reviewer. `4a3bd74f` touched 6 documentation files with 8 insertions / 8 deletions; `decision.md`'s share of that commit is 2 lines (the two seam corrections) — which is exactly why the `:112` clause was never in its scope.

---

## C. The disclosed residual gap in the vocabulary guard — probed directly

### C.1 The mechanism, read from source

`l3hGuardWordBoundaryPatterns` (`generic_client_guard_test.go:109-126`) compiles, per needle:

```
(^|[^a-zA-Z])(?i:<needle>[|<irregular plural>])(?:[^a-zA-Z]|$|(?i:s)(?:[^a-zA-Z]|$)|[A-Z])
```

The **left** arm is unchanged (no letter immediately before). The **right** side gained two arms: `s` + boundary (regular plural), and a case-**sensitive** `[A-Z]` (a following camelCase segment). Two needles ending in `y` carry a literal `…ies` alternate.

### C.2 True coverage, established by probe — not by reading the comment

Round 2's own probe was contaminated (it contained the bare plural `files`), so its reported `file` needle could have come from that rather than from `readFileContents`. I built a probe containing **only** non-initial camelCase embeddings, with **no bare and no plural form anywhere in the file**, and machine-verified that property against the exact regex before planting it:

```go
package layer3handoff_test

func zzReadFileContents() string    { return "" }
func zzRunShellCommand() string     { return "" }
func zzOpenTerminalSession() string { return "" }
func zzUseEditorPane() string       { return "" }
func zzCloneGitRemote() string      { return "" }
func zzTheRepositoryRoot() string   { return "" }
func zzTheFilesystemRoot() string   { return "" }
func zzTheDirectoryNode() string    { return "" }
```

```text
$ go test -race -count=1 -run TestGenericClientBoundary_VocabularyScan ./src/layer3handoff/
ok  	github.com/cachicamas/backend/agent/src/layer3handoff	1.568s
```

**All eight denied concepts leak green.** The disclosed gap is real and is exactly eight concepts wide.

**The scan is live on that same file**, proven by two controls on the identical probe:

```text
# control 1 — one identifier changed to zzRead_fileContents (left boundary satisfied)
generic_client_guard_test.go:230: tree "layer3handoff" carries denied vocabulary: [{base:zz_r3_probe_test.go needle:file}]

# control 2 — a bare plural appended in a comment ("// control: terminals")
generic_client_guard_test.go:230: tree "layer3handoff" carries denied vocabulary: [{base:zz_r3_probe_test.go needle:terminal}]
```

So the clean probe's green is the camelCase gap, not a dead scan.

**Precise coverage statement** (verified form by form against the compiled pattern):

| Shape | Example | Result |
|---|---|---|
| bare singular | `file` | **CAUGHT** |
| regular plural | `files`, `shells`, `terminals`, `editors`, `filesystems` | **CAUGHT** |
| irregular plural | `repositories`, `directories` | **CAUGHT** |
| underscore-joined tool name | `read_file` | **CAUGHT** |
| needle **opens** an identifier, camelCase follows | `fileContents`, `gitStatus`, `shellCommand`, `directoryNode` | **CAUGHT** |
| needle in a **later** camelCase segment (preceded by a lowercase letter) | `readFileContents`, `runShellCommand`, `useGitRemote`, `theRepositoryRoot`, `listDirectoryEntries`, `openTerminalView`, `textEditorPane` | **ESCAPES** |
| ordinary prose false positives | `legitimate`, `digit`, `profile` | correctly **not** flagged |
| the guard's own necessary imports | `"path/filepath"`, `filepath.Join`, `os.ReadDir` | correctly **not** flagged |

**The stated reason for leaving it open is true.** The comment (lines 100-108) says widening the left boundary would re-trip the guard's own `os.ReadFile` call. Simulating a left-widened pattern `(^|[^a-zA-Z]|[a-z])`:

```text
'os.ReadFile(path)'      -> ['file']       <- self-trip
'os.ReadDir(dir)'        -> clean
'"path/filepath"'        -> clean
```

Confirmed: the hazard is real and is the identical self-trip one boundary over.

### C.3 The count floor bites

Dropped the `editor` needle from `l3hGuardDeniedVocabulary()`:

```text
--- FAIL: TestGenericClientBoundary_VocabularyScan_PassesOverBothTreesIncludingItself (0.00s)
    generic_client_guard_test.go:220: len(l3hGuardDeniedVocabulary()) = 7, want 8
FAIL
```

Reverted; `git diff --stat HEAD` empty.

### C.4 Judgement: does the gap hollow out `R-L3H-010`?

**No, and the disclosure is honest.**

- `R-L3H-010`'s binding clause is that *"a committed source scan MUST additionally search both new trees' bytes, comments included, for the denied vocabulary."* It does. Its closing clause — *"The vocabulary set MUST exclude terms that occur in ordinary prose about streams, runs and events, so the guard denies concepts rather than syllables"* — constrains the **set** against false positives, which is satisfied (8 application-specific concepts, verified not to trip on `legitimate`/`digit`/`profile`).
- The realistic leak the requirement targets — a fixture named after an application-specific tool — is caught in this module's own naming convention (underscore-joined, `read_file`) and in leading-camelCase (`fileContents`), and prose leaks in comments use bare or plural forms, all caught.
- The escape class is genuine and material (`readFileContents` is an ordinary Go shape), it is **stated in the guard's own source** with an accurate reason, and it is stated in the same disclose-don't-hide posture this codebase already uses for the stdlib-intermediary gap.
- **Under a stricter reading** — "denies concepts" meaning every inflection — `R-L3H-010` would be **PARTIAL** rather than compliant. I record that reading explicitly rather than resolving the ambiguity in the change's favour silently. Under either reading no scenario is falsified: none of `S-L3H-044…047` asserts inflection coverage.

Carried as **W-R3-2**, non-blocking.

---

## D. All eleven seams re-verified against shipped source

Round 2 checked two injection points. I checked **all eleven injection points and all eleven v1 defaults**, against `harness.go` / `loop.go` / `scheduler.go`, not against the design.

`Harness` declares: `Provider, System, Turn, Hooks, Scheduler, History, RetryAttempts, RetryTiming, Failover, ContextStrategy, ContextBudget, TracerProvider`.
`TurnOptions` declares **exactly seven** fields (lines 66-157): `Model, MaxTokens, PreRequestHook, Tools, PermissionPolicy, Continuation, Hooks`.

| # | Seam | Stated injection point | Real location | Stated v1 default | Verified |
|---|---|---|---|---|---|
| 1 | model provider | required field on the run value | `Harness.Provider` (`harness.go:53`) | none; caller supplies | ✅ / ✅ |
| 2 | decision port | optional **per-run** field | `TurnOptions.PermissionPolicy` (`loop.go:130`) | unset bypasses; all allowed | ✅ / ✅ — `scheduler.go:776` `if policy == nil { return true, nil, nil }` |
| 3 | caller-owned wake handle | optional field on the run value | `Harness.Scheduler` (`harness.go:81`) | unset constructs one | ✅ / ✅ — "Nil constructs one" at `harness.go:78-81` |
| 4 | tool registry | **per-run** field | `TurnOptions.Tools` (`loop.go:112`) | typed unresolved-name failure, not a crash | ✅ / ✅ — `loop.go:97-101` "an unresolved name yields a typed `Result{Outcome: ExecutionFailure…}` in that call's ordinal slot" |
| 5 | transcript | optional field on the run value | `Harness.History` (`harness.go:85`) | unset constructs an empty transcript | ✅ / ✅ — `harness.go:457` `hist = NewHistory()` |
| 6 | hook-family registration surface | one registration value on the run | `Harness.Hooks` (`harness.go:74`) | zero value inert | ✅ / ✅ |
| 7 | singular pre-request field | **per-run** field, composed as element zero of the chain | `TurnOptions.PreRequestHook` (`loop.go:93`) | unset changes nothing | ✅ / ✅ — `applyPreRequestHook` invokes the singular field **FIRST**, its output feeding `chain[0]`; nil singular + empty chain is the identity default |
| 8 | tracing-API provider | optional field on the run value | `Harness.TracerProvider` (`harness.go:131`) | unset → the tracing API's own no-op | ✅ / ✅ — `tracerFromHarness(h.TracerProvider)` at `harness.go:637` |
| 9 | in-frame delegation door | installed internally per scheduled call; **never** a field on the run value | no such field on `Harness` or `TurnOptions` | nothing ships in v1 | ✅ / ✅ |
| 10 | **failover policy** | optional field on the run value, consulted **once at retry-bound exhaustion** | `Harness.Failover` (`harness.go:107`) | unset never called; behaves as decline | ✅ **corrected** / ✅ — `harness.go:959` `if verdict == verdictExhausted && h.Failover != nil` |
| 11 | **context-reduction policy** | optional field on the run value, consulted **once per logical turn** | `Harness.ContextStrategy` (`harness.go:115`) | unset → no reduction ever attempted | ✅ **corrected** / ✅ — `harness.go:701`, nil-guarded, at the turn boundary; source comment cites `R-CTX-001` "consulted exactly once per logical turn" |

**11/11 injection points resolve to a real field (or, for the delegation door, to a real absence of one). 11/11 v1 defaults are true of the shipped code.** W-R2-4 is closed and introduced no new false statement. One terminological observation is recorded as **S-R3-1**.

---

## E. Every clause of doc 0003's AG-23 status paragraph, checked

The doc-0003 diff is **9 insertions / 9 deletions**: 8 checkbox flips and 1 status paragraph. All 8 flipped rows are **byte-identical modulo the `[ ]`→`[x]` character** — no annotation text was rewritten.

| Clause | Verdict |
|---|---|
| "closes Wave 6 and discharges **R-21**" | ✅ doc 0003:75 defines R-21; :2097 "Closes: R-21"; :2220 and :2267 map AG-23→R-21 |
| "the RED watched failing **before** the fix existed, and both defeat directions planted and watched failing **after** the fix landed, each reverting one half of it" | ✅ **corrected clause is true** — I re-planted both directions myself (§ G); each requires the shipped fix in place |
| "a dedicated abort channel plus a deferred join, registered so it always runs before the consumer sink closes on every exit path" | ✅ `defer close(sink)` at `harness.go:533`, the join defer registered **immediately after** at `:557-563`, so LIFO runs the join first |
| "including a panic through the deliberately unrecovered pre-request-hook seam" | ✅ `S-RUN-114`'s test drives exactly that seam |
| "the import-boundary guard gains a fifth check … with its own per-pattern vacuity floor proven non-vacuous by a planted bite" | ✅ `TestLayer2_SiblingTrees_NoDirectForbiddenFamilyImport` PASS; per-tree floor at `generic_client_guard_test.go:188` |
| "a new importable, scriptable kit ships at `src/apptest` — never inside `src/agenttest`, which stays byte-unchanged" | ✅ kit present, no build tags, compile-time pins at `permission.go:92` / `tool.go:75`; `git diff --numstat main...HEAD -- backend/agent/src/agenttest/` **empty** |
| "consumer proof at `src/layer3handoff` … all seven acceptance capabilities … in one sequential run" | ✅ seven `t.Run` stages `01_…`–`07_…`, zero `t.Parallel` in the proof |
| "a mechanical, non-self-tripping vocabulary guard proven to bite on a planted application-specific tool name" | ✅ re-proven twice by my own controls (§ C.2) |
| "four runnable examples, compiled and run by the ordinary test command, cover building a harness, driving a run, consuming its event stream, and handling a permission suspension" | ✅ `ExampleHarness`, `ExampleHarness_Run`, `ExampleHarness_events`, `ExampleHarness_suspension`, each with a mandatory `// Output:` block, all four `--- PASS` in the evidence run |
| "registers the four named limitations … with the forwarder race explicitly excluded from that register as a fixed defect" | ✅ `decision.md` § 4.2 register + § 4.3's explicit exclusion; § 6 row 3 "answered" |
| "`scheduler.go` is restored to the fence, and `import_boundary_test.go`'s release is made permanent" | ✅ `hksScopeFenceByteUnchangedFiles()` lists `scheduler.go` and does **not** list `import_boundary_test.go` |
| "`TurnOptions.PreRequestHook` … kept, unamended in behavior, carries no deprecation marker" | ✅ present at `loop.go:93`, no deprecation marker |
| "The three shipped specs that deferred their own test-convenience-wrapper rows … back-annotated closed" | ✅ `agent-loop-skeleton`, `agent-protocol-events`, `agent-message-tool-events` deltas all present |
| "**24 of 24** milestones shipped" | ✅ derived: `AG-00`…`AG-23` = **24 distinct** milestone tokens counted off the document's own wave table; the string "24 of 24" occurs exactly once |

**Checklist state**: `grep -c '^- \[ \]'` = **0**; `grep -n '\[ \]'` over the whole document returns **nothing**. 21 checked rows.

**Every clause of the reworded paragraph is true.** The correction introduced no new false statement in doc 0003.

**But the same clause survives one file over.** See **W-R3-1**.

---

## F. Full milestone re-verification

### Spec compliance — authoritative tally

| Delta | Requirements | Scenarios (AG-23's own) | Reproduced as context |
|---|---|---|---|
| `agent-layer3-handoff` | 13 (`R-L3H-001…011`, `NFR-L3H-A`, `NFR-L3H-B`) | 56 (`S-L3H-001…056`) | — |
| `agent-package-scaffold` | 1 (`R-AGP-003`) | 6 (`S-AGP-067…072`) | 19 (`S-AGP-020…038`) |
| `agent-run-driver` | 1 (`R-RUN-014`) | 4 (`S-RUN-114…117`) | — |
| `agent-hook-taxonomy` | 1 (`R-HKS-010`) | 2 (`S-HKS-027, 028`) | 1 (`S-HKS-024`) |
| `agent-v1-scope` | 1 (`R-AGS-016`) | 1 (`S-AGS-069`) | 4 (`S-AGS-065…068`) |
| `agent-loop-skeleton` | 0 (back-annotation) | 0 | — |
| `agent-protocol-events` | 0 (back-annotation) | 0 | — |
| `agent-message-tool-events` | 0 (back-annotation) | 0 | — |
| **Total** | **17** | **69** | 24 (verified at their own milestones; not re-litigated) |

**Requirements: 17/17 compliant. Scenarios: 69/69 compliant.**

Movements from round 2:

| Scenario | Round 2 | Round 3 | Reason |
|---|---|---|---|
| `S-L3H-056` | FAILING | **COMPLIANT** | Module-wide claim restored and measured true (§ A) |
| `S-L3H-028` | PARTIAL | **COMPLIANT** | All 11 injection points and all 11 defaults verified against shipped source (§ D) |

Requirement-level: `R-L3H-006` and `NFR-L3H-B` move to fully compliant. `R-L3H-010` returns to COMPLIANT with the residual gap recorded as **W-R3-2** and the stricter reading stated explicitly (§ C.4).

### Consumer proof — seven stages, stream-level where the criterion requires it

| Stage | Capability | Asserted through the drained stream? |
|---|---|---|
| `01_construction_from_injected_fakes` | construction | N/A by design — `S-L3H-001` is a package/directory and construction property; asserts all five injected fakes are non-nil on the public value |
| `02_multiturn_and_tool_execution` | multi-turn + tool | **Yes** — `turn_start` count ≥ 2, `tool_start` and `tool_end_success` for the exact call ID, all read from `run1Events`; `agent.CheckStream(run1Events)` clean (`S-L3H-003`) |
| `03_scripted_permission_suspension` | suspension resolved by script | **Yes** — `permission_decision_required` index strictly before `permission_decision_made`, both scanned out of `run1Events`; queue-consumption asserted separately (`S-L3H-004`) |
| `04_interrupt` | interrupt | **Yes** — last event of `run2Events` is `run_end` with `RunOutcomeInterrupted`; drain report clean (`S-L3H-005`) |
| `05_resumed_prompt` | resumed prompt | **Yes** — last event of `run3Events` is `run_end` with `RunOutcomeCompleted`; drain report clean (`S-L3H-006`) |
| `06_second_harness_over_seeded_transcript` | seeded second harness | **Yes** — transcript moved only via `History.Entries()` → `Entry.Message()` → `agent.NewSeededHistory`; seeded drain report clean, stream non-empty (`S-L3H-007`) |
| `07_closing_validation_every_drain_clean` | closing validation | **Yes** — four `Violation()` checks over four distinct drained streams |

No `t.Parallel()` anywhere in the proof — the stage-6 hand-off is ordered by construction (`AD-4`).

### Kit

`DrainAndCheck` is 6 lines and returns `agent.CheckStream(events)`'s report unmodified — zero re-implemented validation (`S-L3H-017`). Compile-time interface pins in **production** files (`permission.go:92`, `tool.go:75`). No build tags anywhere in `src/apptest` (`S-L3H-014`, `S-L3H-015`). The exhaustion latch falls back to `AllowOnce` and latches rather than blocking (`S-L3H-016`).

### Surface freeze

| Check | Result | Evidence |
|---|---|---|
| No new event kind | PASS | `git diff --numstat main...HEAD -- src/agent/event.go` **empty**; `EventKinds()` derives from `eventKindFirst`..`eventKindEnd` in that unchanged file; **six** independent `== 25` assertions green (`invariant_pin_test.go:67`, `compaction_stream_test.go:98,173`, `scope_fence_test.go:87`, `hooks_test.go:1658,1776,2234`) |
| `TurnOptions.PreRequestHook` retained | PASS | `loop.go:93`, no deprecation marker — the user's declined-removal decision honoured |
| `go.mod` / `go.sum` byte-unchanged | PASS | 0 changed files |
| `src/ai/` byte-unchanged | PASS | 0 changed files |
| `src/agenttest/` byte-unchanged | PASS | 0 changed files |
| `del024ByteUnchangedFiles()` | PASS | `--- PASS: TestScopeFence_S_DEL_024_ByteUnchangedFilesAndNoNewKind (0.14s)` |
| `hksScopeFenceByteUnchangedFiles()` | PASS (fence ran; function SKIPs afterwards) | All 17 fence entries — **including the restored `scheduler.go`** — plus `src/ai/`, `src/agenttest/`, `go.mod`/`go.sum`, **and** the four reflection invariants (25 kinds, 5 `*Harness` methods, `Turn` arity 6, `Run` arity 4) run **before** the `t.Skip`; the skip covers only the `loop.go` anti-vacuity branch |
| Import-boundary, 5 checks | PASS | All five `TestLayer2_*` green |

### Doc 0003

| Check | Result |
|---|---|
| `grep -c '^- \[ \]'` | **0**; and `[ ]` occurs nowhere in the file |
| Counter derivation | `AG-00`…`AG-23` = **24 distinct** milestones counted off the document's own wave table; "24 of 24" derived, not asserted |
| Diff scope | 9 insertions / 9 deletions = 8 checkbox flips (all byte-identical modulo the checkbox character) + 1 status paragraph |
| AG-23 status paragraph | **Every clause true** (§ E) |

### Completion-checklist walk (`decision.md` § 3)

21 rows, each with its own distinct closing node and per-row evidence — no blanket sweep (`S-L3H-033`). Only row 21 carries "(this change)" (`S-L3H-035`). Merge-base citations spot-checked **on `main`**: lines 2161, 2164, 2169 each carry their pre-existing "— closed by AG-NN." annotation at the merge base, so the evidence does not resolve to anything this change created (`S-L3H-034`).

### `S-L3H-047` — the statement carries no application-specific term

Letter-boundary scan of `decision.md` for `file`, `shell`, `skill`, `terminal` (singular + plural): each occurs on **line 140 only** — the captioned, verbatim quotation of doc 0003's own checklist wording, already adjudicated at round 1's W-8. Zero leaks elsewhere.

---

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|---|---|---|
| Forwarder bounded exit (`R-RUN-014`) | Implemented | Both halves re-proven load-bearing by independent defeat this round (§ G) |
| Guard extension (`R-AGP-003`) | Implemented | Per-pattern and per-tree vacuity floors; check 5 scans test files; `(tree, filename)`-keyed self-exclusion |
| Kit (`R-L3H-003`) | Implemented | No build tags, compile-time pins, non-wedging exhaustion latch, wholesale drain delegation |
| Determinism (`R-L3H-004`) | Implemented | Static (no clock import in either tree) + dynamic repeat-run structural equality with the identity difference itself asserted |
| Consumer proof (`R-L3H-001`) | Implemented | Seven named stages, stream-level assertions, sequential by construction |
| Vocabulary guard (`R-L3H-010`) | Implemented, with a disclosed residual class | § C.4 |
| Evidence discipline (`NFR-L3H-B`) | Implemented | Module-wide zero-issue lint restored and measured true |
| Stream freeze (`R-L3H-011`) | Implemented | 25 kinds, `PreRequestHook` retained, four byte-freezes hold |

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| AD-1 — abort channel + deferred join, LIFO before `close(sink)` | Yes | Registration order read from source at `harness.go:533`/`:557`; both halves defeat-proven this round |
| AD-2 — per-check pattern split, per-pattern floor, fifth check | Yes | Plus per-tree floor and pair-keyed exclusion |
| AD-3 — kit delegates wholesale | Yes | `DrainAndCheck` returns `agent.CheckStream`'s report unmodified |
| AD-4 — sequential consumer proof | Yes | No `t.Parallel()` in the proof or any stage |
| AD-6 — checklist walk scoped to the property | Yes | The scan is a search for the unchecked marker, not a row-count comparison |
| Deviation 1 — letter boundary instead of `\b` | **Justified; now partially widened and honestly bounded** | Necessary (`\b` treats `_` as a word character). The right side is widened; the left is not, and the cost is now stated in the guard's own source with a verified reason |
| Deviation 2 — self-exclusion for the vocabulary guard | Yes | Narrow in intent and in effect |

---

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | PASS | Per-work-unit RED/GREEN/defeat/rollback rows for WU-1…9, round-1 R-1…R-5, and round-2 W-R2-2 / S-R2-3 |
| All tasks have tests | PASS | Every behavioural unit carries a focused command; documentation-only findings correctly marked N/A |
| RED confirmed (tests exist) | PASS | Every named test file exists and was read |
| GREEN confirmed (tests pass) | PASS | 14/14 packages on independent uncached execution |
| Triangulation adequate | PASS | 2 tests in the new trees + 4 runnable examples + 7 proof stages + 8 kit unit tests |
| Safety Net for modified files | PASS | Substrate filters widened in the same commits as the files they cover |
| **Bites watched failing** | **PASS — 4/4 re-planted this round** | Defeat A, defeat B, the camelCase-escape probe with two controls, and the denied-vocabulary count floor. All reverted; byte-identity confirmed after each |

**TDD Compliance**: 7/7 checks passed.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|---|---|---|---|
| Unit (kit behaviour) | 8 | 3 | `go test -race` |
| Integration (live `Harness` runs) | 4 | 3 | `go test -race` |
| Guard / source-scan | 2 | 2 | AST + regexp |
| Runnable examples | 4 | 1 | example runner |
| **Total** | **18** | **9** | |

### Changed File Coverage

Coverage analysis skipped — no coverage threshold is configured for this module, and coverage is informational (never blocking) under this skill.

### Assertion Quality

Every new and remediation-modified test file re-scanned.

**Assertion quality: all assertions verify real behaviour.** Specifically re-cleared this round:

- The stage-2 and stage-3 stream assertions read `run1Events` and were proven to fail without the events (round 2 planted; unchanged since).
- The per-tree vacuity floor and the denied-set count floor both **fail on a planted regression** — the count floor re-planted and watched failing this round.
- Stage 07 is not a placeholder: four `Violation()` checks over four distinct drained streams.
- No tautologies, no ghost loops (both loops over possibly-empty collections sit behind explicit non-empty floors), no orphan empty-collection assertions, no smoke-test-only assertions.

### Quality Metrics

| Gate | Result |
|---|---|
| **Linter** | **`0 issues.`, exit 0, whole module**, after `golangci-lint cache clean`, pinned `bin/golangci-lint` v2.9.0, `.golangci.yml` byte-unchanged vs `main` |
| **Vet** | Clean (`make lint`'s first stage, `go vet ./...`) |
| **Vulnerabilities** | `make vuln-check` exit 0; **0** `"type": "finding"` entries across 170 scanned OSV records |
| **gofmt** | `gofmt -l backend/agent/src` lists **15** files. All 15 verified **byte-identical to `main`**; the intersection with this branch's **19** changed files under `src/` is **empty**. Pre-existing repository condition |

---

## G. Both forwarder defeats re-planted by this phase

### G.1 Defeat A — termination removed (`S-RUN-115`)

Plant: `select { case sink <- ev: case <-forwarderAbort: return }` → bare `sink <- ev`; deferred join kept.

```text
panic: test timed out after 45s
	running tests:
		TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink

goroutine 8 [chan receive]:
	.../src/agent/harness.go:560     # the deferred join, <-forwarderDone
goroutine 9 [chan send]:
	.../src/agent/harness.go:863     # the now-unabandonable forwarder send
FAIL	github.com/cachicamas/backend/agent/src/agent	45.475s
```

**Matches `S-RUN-115`'s corrected text exactly** — a **deadlock** surfacing as a test timeout naming exactly two blocked goroutines: the join blocked on a channel receive and the forwarder blocked on its own channel send. No send-on-closed-channel panic occurred, exactly as the scenario says is structurally unreachable on this defeat.

### G.2 Defeat B — the ordering edge removed (`S-RUN-116`)

Plant: `<-forwarderDone` deleted from `Run`'s defer; `close(forwarderAbort)` kept.

**4/4 runs FAILED.** Three of the four surfaced the literal panic:

```text
WARNING: DATA RACE
      .../src/agent/harness.go:533     # close(sink)
panic: send on closed channel
	.../src/agent/harness.go:861       # the forwarder's send
FAIL	github.com/cachicamas/backend/agent/src/agent	0.489s
```

**Matches `S-RUN-116` exactly** — `close(sink)` racing the forwarder's send with the happens-before edge removed.

**Both directions watched failing, in both directions, this round. `R-RUN-014` is load-bearing in both halves.**

### G.3 Plant hygiene

After every plant: `git status --porcelain` **empty**, `git diff --stat HEAD` **empty**. The read-only base checkout `/Users/braejan/workspace/witsaba/repositories/cachicamas` confirmed `git status --porcelain` **empty** at the end — no write ever reached it.

---

### Issues Found

#### CRITICAL

**None.**

#### WARNING

**W-R3-1 — `decision.md:112` still carries the exact falsified clause W-R2-5 corrected in doc 0003.**
§ 4.3 of the compatibility statement reads:

> …was found and **fixed** this milestone (`agent-run-driver`'s `R-RUN-014`), RED-first, **both defeat directions planted and watched failing before the fix**.

That is the same structurally impossible claim W-R2-5 identified and the round-2 remediation corrected — one file over. Defeat A and defeat B are each *defined* as reverting one half of the shipped fix (`tasks.md` 1.4/1.5, `S-RUN-115`/`S-RUN-116`), so neither can be planted before the fix exists; I re-planted both this round and each required the fix in place. Only the RED (`S-RUN-114`) was watched failing before the fix existed.

The remediation's own record scoped its check to *"every other clause in that paragraph"* — the doc-0003 paragraph — and never grepped the phrase across the change folder, even though commit `4a3bd74f` edited `decision.md` in the same commit as doc 0003. A repository-wide grep finds **exactly one** surviving occurrence: this one. This is the third appearance in this change of the documented "correcting a requirement leaves its sibling wrong" mechanism.

**Why WARNING and not CRITICAL, stated explicitly**: round 2 graded the *identical sentence* in doc 0003 — a canonical, non-archiving document with wider readership — as WARNING, and grading a lower-reach instance higher would be inconsistent. `decision.md` archives inside the change folder rather than promoting into `openspec/specs/`, so it lacks C-1's blast radius. And **no requirement or scenario is falsified by it**: `S-RUN-115`/`S-RUN-116` are correct, `S-L3H-039` ("the defect appears in the register nowhere") is true, and `NFR-L3H-B` requires bites "recorded watched failing" without ordering them relative to the fix. It is a one-line documentation-only fix and should ride the same pass as any other pre-archive edit.

**W-R3-2 — The vocabulary guard's non-initial-camelCase class escapes; verified, bounded, and honestly disclosed.**
Proved by an uncontaminated probe (§ C.2): eight of eight denied concepts pass green as `readFileContents`-shaped identifiers, while bare, regular-plural, irregular-plural, underscore-joined and leading-camelCase forms are all caught. The guard's own comment (lines 100-108) states this class and its reason, and I verified the reason is true: a left-widened boundary re-trips the guard's own `os.ReadFile` call. Round 2's probe was contaminated by a bare plural, so this is the first clean measurement of the true boundary. Non-blocking; recorded so the next reader does not rediscover it as a defect, and so a future durable fix knows the exact shape and the exact obstacle.

**W-R3-3 — Review budget: 2389 counted lines against a 1000-line budget. Recorded as fact, not resolved.**
Independently measured: `git diff --numstat main...HEAD -- . ':!openspec'` = **2308 insertions + 81 deletions = 2389** across **20** files, matching apply's own figure. Including `openspec/` the full branch diff is **4382 + 81 = 4463** across 33 files. The user granted `size:exception` up front. **Deliberately not resolved by narrowing scope or deleting tests** — the overage is the honest cost of the spec's own bite-and-defeat requirements, and every remediation round added detection coverage rather than removing it.

**W-R3-4 — `gofmt -l backend/agent/src` is not empty; it lists 15 files.**
All 15 verified **byte-identical to `main`**; the intersection with this branch's 19 changed files under `src/` is **empty**. Pre-existing repository condition. Recorded because `scheduler.go` is simultaneously gofmt-dirty **and** a byte-unchanged fence entry this change deliberately restored — running `make fmt` would break the fence this milestone mandates. Never run `make all` on this branch.

**W-R3-5 — The stdlib-intermediary gap remains open by construction, disclosed.**
Check 5 denies only direct imports of the five named families; check 2's closure listing filters the standard library out before its allowlist runs. A standard-library intermediary such as `text/template` (which itself reaches `os`) is invisible to both. `R-L3H-002` requires denial "by name", so this is compliant as specified. No action.

**W-R3-6 — `decision.md` § 3's heading names a row count ("The twenty-one-row … walk").**
`R-L3H-007` explicitly warns that a count assertion "goes silently false the moment a later editor appends a row". The **requirement** is correctly property-scoped and `S-L3H-032`'s scan is expressed as a search for the unchecked marker — so the requirement itself is compliant, and the count is true today (21 rows, measured). The heading is a snapshot label in an artifact that freezes at archive, so the exposure is small. Recorded for completeness because this repository has the drift class on record.

#### SUGGESTION

- **S-R3-1** — `decision.md` uses two nearly synonymous phrasings that mean different structs: *"a field on the run value"* for `Harness` fields (6 seams) and *"a per-run field"* for `TurnOptions` fields (3 seams). All 11 are accurate under that convention, but the convention is nowhere stated, and `S-L3H-029` requires a Layer 3 reader to get the answer "without reading source". One sentence declaring the convention — or one consistent phrasing — would close the last of W-R2-4's class.
- **S-R3-2** — A future durable fix for W-R3-2 should carry an explicit allowlist for the guard's own `os.ReadFile` / `path/filepath` uses so the left boundary can be widened safely, and should cover the markdown surface at the same time (round-2's S-R2-1).
- **S-R3-3** — Round-1's S-3 is still unaddressed: the consumer proof rebuilds `l3hToolCallScript` / `l3hTextScript` byte-for-byte alongside `determinism_test.go`'s equivalents. Promoting them into the `apptest` kit would remove the duplication and further demonstrate the kit's sufficiency. Declined twice with reasons; recorded as a follow-up.

---

### Verdict

**PASS WITH WARNINGS — 0 CRITICAL, 6 WARNING, 3 SUGGESTION.**

**This is a clean signal, not a manufactured one.** Round 2's blocker is genuinely closed: I read the restored spec text, then measured `make lint` myself after a cache clean with the pinned v2.9.0 binary and an unmodified config — `0 issues.`, exit 0, module-wide — and audited every sibling occurrence of the nonexistent-finding claim, confirming the one surviving mention reads as annotated history and explicitly forbids being cited as current evidence. Every other round-2 finding was re-verified by opening the actual line rather than reading the remediation's summary, and each correction was independently checked for introducing a *new* falsehood, which is the exact mechanism that produced round 2's blocker out of round 1's remediation.

The substantive engineering is sound and was re-proven, not accepted. Both forwarder defeats were re-planted this round and each produced exactly the mechanism its scenario names — defeat A a two-goroutine deadlock with the join blocked at `harness.go:560` and the forwarder at `:863`, defeat B a `close(sink)`/`sink <- ev` race surfacing the literal `panic: send on closed channel`. All eleven seam injection points and all eleven v1 defaults were checked against shipped source, closing W-R2-4 completely. The whole 14-package suite is green uncached under `-race` in 172 s with zero `(cached)` markers, `make build` and `make vuln-check` exit 0, every byte-freeze holds, no new event kind lands, `TurnOptions.PreRequestHook` survives at `loop.go:93` as the user decided, and doc 0003 reaches a genuinely derived 24 of 24 with zero unchecked rows. Scenario compliance moves from 67/69 to **69/69** and requirement compliance from 14/17 to **17/17**.

**One real finding remains, and it is documentary, one line, and does not block archive.** `decision.md:112` still carries the structurally impossible "both defeat directions planted and watched failing **before the fix**" clause that W-R2-5 corrected in doc 0003 — the round-2 remediation edited both files in the same commit and swept only one of them. It falsifies no requirement and no scenario, and by round 2's own grading of the identical sentence it is a WARNING; it should nonetheless be corrected in the same one-line documentation pass before the change archives, because the compatibility statement is the artifact this milestone exists to publish.

The most substantive engineering observation is **W-R3-2**: the vocabulary guard's true coverage, measured for the first time with an uncontaminated probe, catches bare, plural, irregular-plural, underscore-joined and leading-camelCase forms and misses only a needle preceded by a lowercase letter. That class is real, it is stated in the guard's own source with a reason I verified to be true, and it does not hollow out `R-L3H-010` — though I record explicitly that a stricter reading of "denies concepts rather than syllables" would leave that requirement PARTIAL rather than compliant.

**Review-budget overage recorded as fact**: 2389 counted lines against a 1000-line budget (4463 including `openspec/`), under the user's granted `size:exception`. Not resolved by narrowing scope or deleting tests.

No production code change is required to clear this gate.
