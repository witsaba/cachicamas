# Apply progress — AG-23: Publish the Layer 3 readiness contract

**Status**: WU-1 through WU-9 complete, all committed, all final gates green. **WU-10 (archive) deliberately NOT run**, per the orchestrator's explicit instruction — the change folder stays in place and the delta specs stay un-promoted for `sdd-verify` to review next.

**Mode**: Strict TDD (active, per session config).

---

## Critical finding: the counted review budget is significantly exceeded

**Report up front, not buried.** `git diff --numstat main...HEAD -- . ':!openspec'` (production + tests + `docs/**`, `openspec/**` excluded, matching the session's own counting rule) totals **2216 lines** (2136 insertions + 80 deletions across 20 files). This is **more than double** both the original 861–1002 forecast and the pre-authorized ~1150 extension (which the tasks artifact reserved specifically for WU-1, "never for demoting a discharged obligation to the known-limitations register").

**Per-file breakdown against the tasks artifact's own per-file estimates** (`git diff --numstat`, sorted descending):

| File | Actual | Forecast | Over by |
|---|---|---|---|
| `src/layer3handoff/layer3handoff_test.go` | 437 | 220–260 | ~1.7–2x |
| `src/apptest/determinism_test.go` | 267 | **not in the original file list at all** | fully unbudgeted |
| `src/agent/example_test.go` | 233 | 120–160 | ~1.5–2x |
| `src/agent/import_boundary_test.go` | 246 (210+36) | 70–100 | ~2.5–3x |
| `src/layer3handoff/generic_client_guard_test.go` | 166 | 60–80 | ~2–2.7x |
| `src/apptest/permission_test.go` | 131 | (part of 130–160 for all three `_test.go`) | — |
| `src/apptest/tool_test.go` | 102 | (part of the same 130–160) | — |
| `src/apptest/drain_test.go` | 102 | (part of the same 130–160) | — |
| `src/agent/harness_forwarder_panic_test.go` | 96 | 75–95 | roughly in range |
| `src/apptest/permission.go` | 92 | (part of 130–170 for all four `.go`) | — |
| `src/agent/harness.go` | 121 (88+33) | 25–35 | ~3.5–4.8x |
| `src/apptest/tool.go` | 75 | (part of the same 130–170) | — |
| `src/agent/hooks_test.go` | 35 | 3–8 | ~4–12x |
| `src/apptest/doc.go` | 24 | (part of the same 130–170) | — |
| `src/agent/loop_test.go` | 18 | **not in the original file list at all** | fully unbudgeted |
| `src/agent/loop_hook_test.go` | 17 | **not in the original file list at all** | fully unbudgeted |
| `src/apptest/drain.go` | 17 | (part of the same 130–170) | — |
| `src/agent/doc.go` | 10 | 8–14 | in range |
| `docs/architecture/milestones/0003-...md` | 18 (9+9) | 15–25 | in range |
| `src/layer3handoff/doc.go` | 7 | 5–10 | in range |

**Two entirely unbudgeted additions, both apply-time discoveries recorded below, account for a meaningful share:**
- `src/apptest/determinism_test.go` (267 lines) — the repeat-run structural-equality proof AD-3/task 3.7 explicitly requires; it was never itemized as its own file in the tasks artifact's file table (folded conceptually into the `apptest` test estimate but not sized on its own).
- `src/agent/loop_test.go` + `loop_hook_test.go` widenings (35 lines total) — discovered during apply: `TestTurn_SubstrateUntouched` / `TestTurn_PreRequestHook_SubstrateUntouched` are LIVE guards (not historical), and every new file created directly inside `src/agent/` must be added to both filters by exact name in the same commit, or the guard fails once the file is committed. Neither `harness_forwarder_panic_test.go` nor `example_test.go` was anticipated as needing this widening at planning time.

**Beyond that, nearly every individual estimate ran over**, most severely `import_boundary_test.go` (guard extension: the fifth check plus per-pattern floor plus the self-exclusion discovery, at ~2.5–3x its estimate) and `harness.go` (the D-5 fix's own defer/select mechanism plus its documentation comments, at ~3.5–4.8x).

**What I did NOT do in response**: I did not remove rigor, shorten evidence, or silently narrow scope to force the number down — every test, every bite, every defeat-direction proof, and every documentation section this milestone's own requirements (`R-L3H-001..011`, `NFR-L3H-A/B`) demand is present and green. I also did not stop mid-implementation to request a new workload decision once individual files began running over their per-file estimates, which in hindsight I should have done once `layer3handoff_test.go` and `import_boundary_test.go` were clearly landing at roughly double their forecasts — that is a genuine process gap on my part, reported here rather than hidden.

**What this means for delivery**: the `single-pr` / `size:exception` decision the user pre-authorized ("1000 review lines and extend if needed") does not, on its own wording, clearly cover a **more-than-double** overage of the specific ~1150 ceiling that was reserved for one work unit. This is a decision point for the orchestrator/user: accept the larger single-PR exception as delivered, or reconsider a chained/stacked split before this goes to review. I am flagging it rather than deciding it.

---

## Work Unit Evidence (all work units)

| WU | Focused test command | Result | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | `go test -race -count=1 ./src/agent/... -run TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink -v` | PASS, 100/100 iterations, 1.324s (post-fix) | The RED test itself drives a real `Harness.Run` through the panic-unwind path — this test IS the harness | Revert `harness.go`, delete `harness_forwarder_panic_test.go` |
| 2 | `go test -race -count=1 ./src/agent/... -run TestLayer2 -v` | PASS, all TestLayer2\* green, 1.4–1.7s | N/A — AST-only guard | Revert `import_boundary_test.go` only |
| 3 | `go test -race -count=1 ./src/apptest/...` | PASS, 1.3–2.5s | Real scripted permission/tool scenario against a live `agent.Harness` (`determinism_test.go`'s `detRunScenario`) | Delete `src/apptest/` |
| 4 | `go test -race -count=1 ./src/layer3handoff/...` | PASS, 6 sequential stages + vocabulary guard, 1.5–2.6s | Real, full 7-capability `agent.Harness` run sequence — this test IS the runtime harness | Delete `src/layer3handoff/` |
| 5 | `go test -race -count=1 ./src/agent/... -run Example -v` | PASS, all 4 examples, output verified, identical in isolation and full suite | The examples themselves execute a live `Harness` | Delete `example_test.go` |
| 6 | N/A — documentation | `TestDocContract_*` green, row count unchanged | N/A | Revert `doc.go`'s new section and delete `decision.md` |
| 7 | `go test -race -count=1 ./src/agent/... -run TestHooks_ScopeFence` and `scope_fence_test.go` suite | PASS (2 green, 2 correctly SKIP on their own documented anti-vacuity floor since `loop.go`/`scheduler.go` are unchanged by this branch) | N/A — guard-list edit | Revert `hooks_test.go`'s two edits independently |
| 8 | N/A — markdown only | `rg '^- \[ \]'` over the whole document returns zero matches | N/A | Revert the single doc 0003 commit |
| 9 | N/A — OpenSpec delta files | N/A | N/A | Revert the three delta files independently |

---

## WU-1 — AD-1 forwarder-race fix (`harness.go`, RED-first)

**Tasks 1.1–1.6: all complete.**

- **1.1 (enumeration, apply obligation)**: `Turn`'s own turnSink sends all happen on the harness's own attempt-loop goroutine (synchronous call to `Turn(...)`); `Schedule`'s internal dispatcher goroutines are joined synchronously before `Schedule`/`Turn` return (confirmed via `TestSchedule_LeaveSinkOpenSet_CallerOwnsClose`'s own comment: "Schedule is synchronous — it returns only after the dispatcher goroutine has exited"). No producer other than the per-attempt forwarder itself can be stranded by an abort.
- **1.2 RED, watched failing before the fix existed** (`go test -race -count=1 ./src/agent/... -run TestRun_PanicUnwind_ForwarderNeverSendsOnClosedSink -v`):
  ```
  panic: send on closed channel
  goroutine 6 [running]:
  github.com/cachicamas/backend/agent/src/agent.(*Harness).Run.func3()
      .../harness.go:815 +0xbc
  created by github.com/cachicamas/backend/agent/src/agent.(*Harness).Run in goroutine 5
      .../harness.go:781 +0x1dcc
  FAIL  github.com/cachicamas/backend/agent/src/agent  0.487s
  ```
  Also caught by `-race`: a genuine data race between the forwarder's send and `Run`'s own `close(sink)` defer read, both at the exact lines the fix targets. This is the send-on-closed-channel reason specifically — not a timeout, not a compile error.
- **1.3 GREEN**: dedicated `forwarderAbort` channel + hoisted `forwarderDone` var + a defer registered immediately after `close(sink)`'s own registration (LIFO: runs immediately before `close(sink)`), doing `close(forwarderAbort); if forwarderDone != nil { <-forwarderDone }`. Forwarder's receive AND send both select on `forwarderAbort`. Re-run: **PASS, 100/100 iterations, 1.324s**.
- **1.4 Defeat A (cancellation-termination removed)**: reverted the send-side select to a bare `sink <- ev` (keeping the receive-side select and Run's own join defer). Re-ran the exact focused command with `-timeout 20s`: **FAILED via `panic: test timed out after 20s`**, full goroutine dump showing goroutine 8 blocked in `chan receive` at `harness.go:560` (the join) and goroutine 9 blocked in `chan send` at `harness.go:863` (the now-unabandonable forwarder send) — a genuine, diagnosable deadlock, not the literal "crash on iteration 1" design predicted, but unambiguous FAIL evidence naming both exact blocking points. Reverted; `git diff --stat` showed exactly the intended fix restored (88+/33-), no `DEFEAT-A` marker remaining (`grep` exit 1).
- **1.5 Defeat B (happens-before join removed)**: deleted the `<-forwarderDone` wait from Run's own defer, keeping only `close(forwarderAbort)`. Re-ran: **FAILED**, `panic: send on closed channel` at `harness.go:860` (inside the double-select's send case) on iteration ~117 of the 100-iteration inner loop range (i.e., within the run, not necessarily iteration 1 — matches the design's own "roughly half the iterations" prediction), 0.456s. Reverted; confirmed clean.
- **1.6**: full `src/agent` suite re-run, uncached (`go clean -testcache`), `-race -count=1`: **PASS, 10.187s**.

**Commit**: `fix(agent): AG-23 -- close the forwarder panic race on the sink-close unwind`.

---

## WU-2 — AD-2 import-boundary guard extension

**Tasks 2.1–2.7: all complete** (2.7 confirmed retroactively once WU-3's `ScriptedTool` existed — no `time` import, no `startAt` field, by construction).

- Added `layer2TestOnlyTreePatterns` (const set), extended `allowedTestPrefixes`, converted check 2 into a per-pattern loop with the vacuity floor moved inside it, added check 5 (AST scan over both new trees including test files, families `{os, net, syscall, io/fs, time}`).
- **Bite 2.4 (mistyped pattern)**: planted `src/apptest_BOGUS_BITE_2_4/...` in place of the real apptest pattern. Re-ran `TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction`: **FAILED** — `go list -test returned no packages for pattern "github.com/cachicamas/backend/agent/src/apptest_BOGUS_BITE_2_4/..."; the guard would pass vacuously for this pattern.` Reverted; confirmed clean.
- **Bite 2.6 (time + os imports)**: planted a `time` import in a scratch file inside `src/apptest`, ran `TestLayer2_SiblingTrees_NoDirectForbiddenFamilyImport`: **FAILED** — `scratch_bite_2_6_test.go directly imports "time", which is in the forbidden "time" family`. Reverted. Planted an `os` import in a scratch file inside `src/layer3handoff`: **FAILED** — same message naming `"os"`. Reverted both; confirmed clean after each.
- Full `TestLayer2*` sweep and full `src/agent` suite re-run green after every revert.

**Commit**: `test(agent): AG-23 -- extend the import-boundary guard for two new sibling trees` (bundles the two new packages' minimal `doc.go` stubs, created early so check 2/5 have real, compliant trees to scan rather than nonexistent directories — WU-3/WU-4 built the rest of each package on top in their own commits).

**Apply-time discovery, own commit**: `harness_forwarder_panic_test.go` (WU-1's new file) tripped `TestTurn_SubstrateUntouched` / `TestTurn_PreRequestHook_SubstrateUntouched` once committed — these are LIVE, currently-enforced guards (`git diff <merge-base-with-origin/main>`, which on a feature branch shows exactly that branch's own new files), not a one-time AG-07 historical check as the surrounding prose might suggest. Both `loop_test.go`'s `filterOutLoopFiles` and `loop_hook_test.go`'s `filterOutLoopHookFiles` were widened by the file's exact name, mirroring every prior milestone's own precedent (`harness.go` since AG-13, `import_boundary_test.go` since AG-22 — neither needed re-declaring for this change's own edits to them). Fixed in commit `fix(agent): AG-23 -- widen substrate-untouched guards for the forwarder-panic RED test`, then again for `example_test.go` in WU-5's own commit.

---

## WU-3 — AD-3 `src/apptest` kit

**Tasks 3.1–3.8: all complete.**

- `permission.go` (`ScriptedPermissionPolicy`, FIFO queue under mutex, `AllowOnce` exhaustion default + latch), `tool.go` (`ScriptedTool`, mutex-guarded invocation/argument recording, no wall clock), `drain.go` (`DrainAndCheck`, delegates to `agent.CheckStream` wholesale).
- `permission_test.go`, `tool_test.go`, `drain_test.go`: FIFO order, exhaustion latch (`Resolve #3 (exhausted) outcome = AllowOnce`, `Exhausted() = true` only after the 3rd call against a 2-entry queue), invocation recording (with an explicit copy-safety check), `DrainAndCheck` blocking-until-close proof (via synchronization on the unbuffered channel's own sends, no wall clock) and delegation-not-reimplementation proof (a deliberately malformed stream still surfaces `report.Violation() != nil`).
- **Determinism proof** (`determinism_test.go`, `TestApptestKit_RepeatRun_StructurallyEqualModuloMintedIdentities`): two independent live `agent.Harness` runs of an identical scripted scenario (tool call + permission `AllowOnce` + text completion), projected onto kind + identity-independent payload (`RunOutcome`/`ToolStart.Name`/`ToolEndSuccess.Result`/`PermissionDecisionMade.Outcome`/`MessageDeltaText.Fragment`), asserted equal — **and** the raw `RunID`/`TurnID` values asserted to **differ** between the two runs (the anti-vacuity check). A companion test, `TestApptestKit_RepeatRun_WidenedToWholeEventValues_FailsOnMintedIdentities`, widens the comparison to whole `Event.String()` values (which embed the minted `RunID`) and confirms it **does** find a difference — proving the narrower projection is doing real work, not passing vacuously. Both PASS.
- Full `src/agent` + `src/apptest` suite green together, uncached.

**Commit**: `feat(apptest): AG-23 -- build the packaged test kit`.

---

## WU-4 — AD-4 `src/layer3handoff` consumer proof

**Tasks 4.1–4.6: all complete.**

- `TestLayer3Handoff_ConsumerProof`: one sequential driver, six `t.Run` stages (capabilities 2+3 combined into one stage since they share one `Run` call — documented explicitly in the stage's own comment rather than left implicit):
  1. `01_construction_from_injected_fakes` — asserts every public field is populated from fakes only.
  2. `02_multiturn_and_scripted_permission_suspension` — Run #1: a tool-call turn whose permission decision is deferred then resolved via `Scheduler.WakeParked` (mirroring `src/agent`'s own `harness_suspension_test.go` mechanism one layer up), followed by a text-completion turn. This stage's own read loop performs exactly what `apptest.DrainAndCheck` does internally (accumulate to close, then `agent.CheckStream`) but inlined, because it must intercept mid-stream to call `WakeParked`; documented as such in the stage's own comment.
  3. `03_interrupt` — Run #2 over an `agenttest.Hold` gate; `Interrupt()` once `gate.Reached()`; `apptest.DrainAndCheck` used directly (no mid-stream interception needed here, since the gate signal is a separate channel).
  4. `04_resumed_prompt` — Run #3 on the same `Harness` value.
  5. `05_second_harness_over_seeded_transcript` — `hist.Entries()` → `Entry.Message()` → `agent.NewSeededHistory` → a fresh `Harness{History: seeded}` → one more run.
  6. `06_closing_validation_every_drain_clean` — re-confirms every one of the four `StreamReport`s is clean.
- `generic_client_guard_test.go`: a letter-boundary (not regexp's own underscore-permissive `\b`) scan for eight terms, assembled at run time via `l3hGuardConcat`. **Bite 4.5**: planted `scratch_bite_4_5_test.go` in `src/layer3handoff` containing `func read_file() {}`. Ran `TestGenericClientBoundary_VocabularyScan_PassesOverBothTreesIncludingItself`: **FAILED** — `tree "layer3handoff" carries denied vocabulary: [{base:scratch_bite_4_5_test.go needle:file}]`. Reverted; confirmed clean, re-ran green.
- **Apply-time discovery**: my own hook-fixture attempt with regexp's own `\b` word boundary did NOT catch `read_file` (underscore counts as a word character in `\b`), which would have silently defeated the whole bite. Rewrote the boundary check to `(^|[^a-zA-Z])needle([^a-zA-Z]|$)` (a letter boundary, not a word boundary) before the bite passed.
- **Apply-time discovery, own commit**: the guard itself needs `os.ReadDir`/`os.ReadFile` to scan its sibling trees, which check 5 (its own scope) denies inside `src/layer3handoff`. Excluded `generic_client_guard_test.go` by exact name from check 5's own scan (`layer2SiblingTreeSelfExcludedFile` in `import_boundary_test.go`), with the identical reasoning `import_boundary_test.go`'s own `os.ReadDir` use over Layer 2 production already carries: verification tooling reading already-committed source is not part of the runtime surface it scans. Recorded in both the guard source and `generic_client_guard_test.go`'s own package comment.
- `apptest/doc.go`'s own wording was revised (avoiding "file"/"filesystem" as whole words) after the guard first flagged its own prose.

**Commit**: `feat(layer3handoff): AG-23 -- build the consumer proof and the vocabulary guard`.

---

## WU-5 — AD-5 runnable examples

**Tasks 5.1–5.3: all complete.**

- `example_test.go`, `package agent_test`, four `Example*` functions: `ExampleHarness` (building), `ExampleHarness_Run` (driving, prints `finish reason: stop`), `ExampleHarness_events` (consuming, prints the seven ordered kind names for a text-only turn), `ExampleHarness_suspension` (handling a suspension, a self-contained local `exampleDeferOncePolicy`, prints `requested: call-example-004` then `resolved: allow_once`).
- `go test -race -count=1 ./src/agent/... -run Example -v`: **PASS**, all four, in isolation and as part of the full package suite (re-run twice, identical output both times).
- No example prints a minted `RunID`/`TurnID` — verified by construction (every print statement names a kind, a finish reason, a permission outcome, or a caller-supplied literal).
- **Apply-time discovery**: `example_test.go` is a new file directly inside `src/agent`, so both substrate filters were widened by its exact name in this same commit (the same fix class WU-1's own file needed).

**Commit**: `test(agent): AG-23 -- add four runnable package examples`.

---

## WU-9 — Three back-annotation deltas (falsified-requirement closure)

**Tasks 9.1–9.4: all complete.** Delta files written for `agent-loop-skeleton` (`spec.md:294`), `agent-protocol-events` (`spec.md:163`), `agent-message-tool-events` (`spec.md:109`), each reproducing that spec's own `Explicit non-requirements` list in full and closing only the "Test-convenience wrappers" row: **DELIVERED, RELOCATED to `backend/agent/src/apptest`**, citing this change (`cachicamas-agent-layer3-handoff`) by name and never `backend/agent/src/agenttest`. No code, no tests — documentation only, matching the tasks artifact's own "N/A" focused-test line.

**Commit**: `docs(openspec): AG-23 -- close the three test-convenience-wrapper back-annotations`.

---

## WU-6 — AD-6 compatibility statement

**Tasks 6.1–6.4: all complete.**

- `decision.md`: six sections mirroring AI-40's own precedent shape exactly (read in full from the archived `cachicamas-ai-layer2-handoff/decision.md` before writing). § 2 enumerates nine frozen capabilities plus three marked experimental, by capability only — verified zero Go identifiers beyond the four permitted classes (a named test function, a named production function cited as evidence, and a package import path). § 3 walks all 21 (not 18 — this milestone's own checklist is longer than AI-40's) completion-checklist rows; the 7 rows this change flips are explicitly marked as a **documentation-defect closure, not new evidence**, so the walk stays scoped to the property rather than reading as self-certification. § 4 states the abandoned-consumer contract inherited from Layer 1, the four named limitations with their post-v1 paths, and explicitly states the forwarder defect is fixed and appears in the register **nowhere**. § 5/§6 as designed.
- **Vocabulary check** (R-L3H-006's own {file, shell, skill, terminal} constraint — distinct from R-L3H-010's eight-word guard, and not automated by the Go guard since `decision.md` is markdown, not scanned): grepped manually with the same letter-boundary pattern. Found and fixed two genuine violations (the `[!IMPORTANT]` box's own sentence literally named the four forbidden words while stating the rule; § 4.4 used "file" for source-code files in the guard-disposition discussion) — reworded both, avoiding the words entirely (using "source unit"/"compilation unit" and a more abstract rule statement). Final grep: only "terminal" remains, in its ordinary adjectival sense ("terminal decision", "terminal outcome", "terminal figure"), matching this codebase's own pre-existing, unrelated shipped-spec usage (`agent-run-driver`'s own `R-RUN-001`: "drives one run to its terminal decision") — not the console/interface sense the rule targets.
- `src/agent/doc.go` gains one pointer GoDoc section, worded identically to `src/ai/doc.go`'s own AI-40 pointer ("see that change's decision.md for..."), never a hardcoded path (robust to the pre-archive/post-archive path change). `doc_contract_guard_test.go`'s row-count guard re-run green, unchanged.

**Commit**: `docs: AG-23 -- publish the Layer 2 compatibility statement`.

---

## WU-7 — W-6 closure (guard restoration)

**Tasks 7.1–7.4: all complete.**

- `scheduler.go` restored as the 17th entry in `hksScopeFenceByteUnchangedFiles()`. `import_boundary_test.go` NOT restored — its release made PERMANENT, with the category-error reasoning recorded both beside the list (a substantial new comment block) and in `decision.md` § 4.4.
- **Bite 7.1**: appended a one-line scratch comment to `scheduler.go`. Ran `TestHooks_ScopeFence_ByteUnchangedFilesAndNoNewKind`: **FAILED** — `backend/agent/src/agent/scheduler.go is not byte-unchanged against 9c55eeda...`, diff shown naming the exact appended line. Reverted via `git checkout --`; confirmed clean.
- **7.3**: confirmed `harness.go` was never on either list (16 pre-AG-23 entries in `hksScopeFenceByteUnchangedFiles()`, 7 in `del024ByteUnchangedFiles()`) — no widening needed for WU-1's own edit.
- **7.4**: re-ran both scope-fence test files: `TestHooks_ScopeFence_ByteUnchangedFilesAndNoNewKind` and `TestScopeFence_S_TLS_020_...` correctly **SKIP** on their own documented anti-vacuity floor (empty diff over `loop.go`/`scheduler.go` respectively, since this branch does not itself touch either — the established `t.Skip`-on-empty-diff pattern, not a failure); `TestScopeFence_S_DEL_024_...` and `TestScopeFence_S_LSK_031_...` **PASS** outright.

**Commit**: `test(agent): AG-23 -- close W-6, restore scheduler.go and permanently release the import guard`.

---

## WU-8 — Docs: doc 0003 completion

**Tasks 8.1–8.5: all complete.**

- Re-grepped `^- \[ \]` at edit time (not trusting the cached count): exactly 8 rows, same 8 line numbers as at planning time (2161, 2162, 2164, 2165, 2166, 2167, 2169, 2181) — no drift.
- Flipped the 7 stale rows first, each already citing its closing milestone; flipped AG-23's own row (2181) **last**, after every other work unit's commit.
- Updated the status line from "23 of 24" to "24 of 24, Layer 2 complete" and appended one long AG-23 narrative sentence in the house run-on style, naming: the forwarder-race fix, the guard's fifth check, the `src/apptest` kit (never `agenttest`), the `src/layer3handoff` consumer proof, the four runnable examples, the compatibility statement, the scope-fence disposition, the declined-and-frozen `PreRequestHook` ruling, and the three back-annotated specs — closing with **Wave 6 is complete. Layer 2 is complete.**
- Confirmed the sentence states the `PreRequestHook` ruling as declined/frozen (never removed/deprecated) and the wrapper relocation as delivered to `src/apptest` (never `agenttest`).
- Post-edit: `rg '^- \[ \]'` over the whole document returns zero matches.

**Commit**: `docs: AG-23 -- flip Layer 2 completion checklist to 24 of 24, Wave 6 complete`.

---

## Final verification gates

- **G.1** — `cd backend/agent && go clean -testcache && make test`: **PASS**, exit 0, zero `FAIL` lines, all 14 packages `ok`, confirmed uncached (`src/ai/openaicompat` at 168.089s / 171.613s across two separate full runs — matching the documented ~170s uncached duration, never a `(cached)` result). Total wall-clock ≈ 2:49–2:52 both times.
- **G.2** — `golangci-lint cache clean && make lint`: found and fixed 5 genuine `revive` findings in this change's own new code (two empty drain loops in `example_test.go`, an unused `ctx`/`req` pair in the RED test's panicking hook, two unused `ctx` parameters on `ScriptedPermissionPolicy`). One additional, **pre-existing, out-of-scope** finding (`src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go:17`, a package name with an underscore) reproduces consistently across repeated cache-cleaned runs when scanning the whole module; confirmed via `git diff --stat main -- <that file>` (empty) that it is byte-identical to `main` and therefore not introduced by this branch — fixing it would additionally violate `src/ai/`'s own byte-freeze this milestone's guards require. Scoped to only this change's own packages (`./src/agent/... ./src/apptest/... ./src/layer3handoff/...`): **0 issues**, confirmed reproducibly.
- **G.3** — `make vuln-check`: exit 0, zero `"type": "finding"` entries in the JSON output (only informational `osv`/`progress` records for advisories already known and inert against packages already present before this change) — clean.
- **G.4** — `gofmt -l backend/agent/src`: the dirty-file set after this change is a strict subset of, and identical to, the dirty-file set confirmed present on `main` before any of this change's own edits (spot-checked `scheduler.go`, `tool.go` against `main@9c55eeda` directly). The intersection of `gofmt -l`'s output with this change's own changed-file list is empty — zero newly-dirty files.
- **G.5** — import-boundary (5 checks, all green), no-ambient-authority (green, part of the `TestLayer2Agent_*` sweep), doc-contract row count (green, `TestDocContract_RowCountReScoped` unchanged), `hksScopeFenceByteUnchangedFiles()` (17 entries, restored `scheduler.go` confirmed via bite), `del024ByteUnchangedFiles()` (7 entries, untouched), and `git diff main -- backend/agent/go.mod backend/agent/go.sum backend/agent/src/ai/` (0 lines) plus `git diff main -- backend/agent/src/agenttest/` (0 lines, beyond AG-22's own already-released `tracetest` filter) — both empty.
- **G.6** — all five planted bites (WU-1's defeat A, defeat B; WU-2's mistyped-pattern bite, time/os-import bite; WU-4's `read_file` bite) were watched FAILING for the documented reason, then reverted, with a clean tree confirmed after each (see per-WU sections above for the literal failure text).

**Base checkout safety**: `git -C /Users/braejan/workspace/witsaba/repositories/cachicamas status --porcelain` confirmed empty throughout — no misdirected write ever reached the read-only base checkout.

---

## Deviations from design

1. **`generic_client_guard_test.go`'s own needle boundary** uses a letter boundary (`(^|[^a-zA-Z])needle([^a-zA-Z]|$)`), not `regexp`'s own `\b` (which treats an underscore as a word character and would silently miss `read_file`). Discovered while proving the bite; not anticipated in design.
2. **`layer2SiblingTreeSelfExcludedFile`** — check 5 excludes `generic_client_guard_test.go` by exact name from its own scan, for the reason recorded in both `import_boundary_test.go` and the guard file itself (§ WU-4 above). Not anticipated in design; discovered when the guard's own `os` import tripped its own scope.
3. **`determinism_test.go`** was not itemized as its own file in the tasks artifact's per-file table (folded into the general `apptest` test estimate). It exists exactly as AD-3/task 3.7 require.
4. **The review budget is exceeded** — see the dedicated section at the top of this document. This is the one deviation I could not resolve within apply and am surfacing as a decision point rather than deciding myself.

## Issues found

None beyond the budget overage and the pre-existing, out-of-scope lint finding, both already detailed above.
