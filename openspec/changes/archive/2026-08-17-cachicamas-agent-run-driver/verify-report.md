```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:8f4f3976d21227765ac2fd73a653d452c6ee014b647a138853f5d994bd45a974
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 26/26
scenarios: 60/60
test_command: cd backend/agent && go clean -testcache && make test
test_exit_code: 0
test_output_hash: sha256:2d3903e12993e494a93d2ff14168f95dc56db1f5dbc6fbd400fd7ac8eae73451
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

# Verification Report — AG-13 `cachicamas-agent-run-driver`

> **Change**: `cachicamas-agent-run-driver` · **AG-13 — Drive the multi-turn run** (Layer 2, Wave 3)
> **Branch**: `feat/agent-layer2-wave3-ag13` · **HEAD** `227f3e9d` · **Merge base** `5590afa0`
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ag-13`
> **Mode**: `auto` · **Store**: `hybrid` · **Strict TDD**: active · **Artifacts**: proposal, specs (6), design, tasks, apply-progress — all present, all dimensions verified.
> **Scenario count** (per `specs/agent-run-driver/spec.md:22`, MUST be stated identically in `tasks.md`, `apply-progress.md` and this file): `agent-run-driver` itself is **12 requirements → 25 scenarios, of which 2 are bites** (`S-RUN-061`, `S-RUN-091`). The five cross-cut deltas add **18 further scenarios** this change closes. **Total new/changed evidence obligations: 43.** Counted across all six spec files the retrieved set holds **26 requirements and 60 scenarios**, which is the envelope total above.

## Verdict

**PASS WITH WARNINGS** — **0 CRITICAL · 2 MAJOR · 8 MINOR**

Every one of the 60 scenarios in the six spec files has a passing covering test or a passing command-level check. Both bites were independently re-executed and both are genuinely non-vacuous. Two MAJOR findings remain: one demonstrated violation of an `R-RUN-001` MUST clause on the run-failure path, and one recorded-evidence overstatement in `apply-progress.md` that my independent re-run contradicts.

**Envelope note**: the `requirements: 26/26` count above means every requirement has passing covering evidence for every scenario it declares — which is true. It does **not** mean every requirement's normative text is fully honoured: `R-RUN-001` carries MAJOR-1, a clause its own two scenarios do not reach. The validator's schema rejects a passing verdict alongside an incomplete count, so the warning is carried in the findings and risks rather than in the count.

**Verification method**: every claim below was re-executed. No claim was accepted from `apply-progress.md` without a command. Every defeat test was run through `go test -overlay=` or on an off-worktree copy, so the worktree was never mutated — `git status --short` and `git diff --stat` are both empty at the end of this phase (verified).

---

## Completeness

| Dimension | Result | Evidence |
|---|---|---|
| Tasks complete | ✅ 84/84 ticked, 0 unchecked | `grep -cE '^- \[x\] [0-9]+\.' tasks.md` = 84; `grep -cE '^- \[ \]'` = 0 |
| Per-phase | ✅ 0:6 · 1:18 · 2:30 · 3:9 · 4:3 · 5:6 · 6:12 = 84 | `grep -oE "^- \[[x ]\] ([0-9]+)\." tasks.md \| sort -n \| uniq -c` |
| Files changed match design | ✅ exactly the 10 files `design.md:146-158` names | `git diff --stat 5590afa0..HEAD -- backend/agent/` |
| Artifacts present | ✅ proposal, 6 specs, design, tasks, apply-progress | `ls openspec/changes/cachicamas-agent-run-driver/` |

---

## Build / Tests / Coverage

| Gate | Command | Exit | Result |
|---|---|---|---|
| Tests | `cd backend/agent && go clean -testcache && make test` | 0 | **PASS** — 12 packages `ok`, `grep -c "^--- FAIL"` = **0** |
| Package (uncached, `-race`) | `go clean -testcache && go test -race ./src/agent/...` | 0 | `ok … 1.922s` |
| Build | `cd backend/agent && make build` | 0 | `go build -trimpath ./...` clean |
| Vet | `go vet ./src/agent/...` | 0 | clean |
| Lint | `golangci-lint cache clean && golangci-lint run ./...` (v2.12.2 via `bin/` symlink) | 0 | 2 issues, **both in files with a zero-line diff on this branch** — `src/ai/openaicompat/client_test.go:383` (govet), `src/agenttest/cache_boundary_test.go:303` (staticcheck). **Zero findings in any AG-13 file.** Independently reproduces `apply-progress.md:410`. |
| Coverage | `loop.go` 248/282 = **87.94%**; `harness.go` 83/89 = 93.26% | — | ≥ 80% `NFR-RUN-004` floor. Accepted from the orchestrator's own independent measurement, not re-derived. |

---

## The two bites — independently re-executed

### `S-RUN-061` — run-scope reconstruction (`R-RUN-007`) · **NON-VACUOUS, CONFIRMED**

Scratch (via `-overlay`, worktree untouched): `harness_test.go:1697` `joined := strings.Join(textFragments, "")` → `joined := "alphabetagamma"` (the correct constant for the fixture's `alpha`/`beta`/`gamma` deltas — `loop_test.go:48-82`), i.e. the AG-05 W1 vacuous-helper failure mode.

```
go test -overlay=… -race -v -run 'TestHarness_RunStream' ./src/agent/
--- PASS: TestHarness_RunStream_ReconstructsHistoryAtRunScope (0.00s)
    harness_test.go:1851: run-scope reconstruction did NOT report divergence after a turn-two
    event was dropped — the property is vacuous (AG-05 W1 failure mode)
--- FAIL: TestHarness_RunStream_ReconstructionBiteDropsTurnTwoEvent (0.00s)
```

Exactly the claimed shape: `S-RUN-060` alone stays GREEN against a vacuous helper (so `S-RUN-060` cannot detect vacuity on its own), while `S-RUN-061` FAILS for the predicted reason. `shasum harness_test.go` identical before and after (`0a70bb14…`); `git diff` empty. **The bite is real.**

### `S-RUN-091` / `S-APP-016` — the `R-APP-002` parked-wait bite (`R-RUN-010`) · **EFFECTIVE, but the recorded evidence is overstated — see MAJOR-2**

Scratch (via `-overlay`): `scheduler.go:645-669`, the parked-wait `select { case <-parkCh: … case <-ctx.Done(): … }`, replaced with `_ = parkCh; return s.runPermissionGate(...)` — an immediate re-resolution, exactly the spec's named defeat test.

The bite **does** observe the parked **wait**, not the registration: `driveSuspendedRun` (`harness_suspension_test.go:144-147`) sets `wakeIssued` *after* reading `permission_decision_required` and immediately before `WakeParked`; registration (`parked.park`, `scheduler.go:614`) happens strictly before the emission, so no registration-only ordering can make that read true. The failure message the scratch produces is the flag assertion at `harness_suspension_test.go:281`. **The bite is not a re-encoding of the gap it closes.**

Reproducibility, measured (this is the MAJOR-2 discrepancy):

| Invocation | Result |
|---|---|
| `go test -overlay=… -race -count=15 …` × 6 independent invocations | **6/6 command-level FAIL** — the aggregate RED is deterministic |
| per-iteration split, same 6 invocations | PASS/FAIL of 15 = 6/9, 7/8, 4/11, 4/11, 4/11, 2/13 — **2 to 7 of every 15 iterations PASS under the defeat scratch** |
| single-iteration (`-count=1`) × 20 | **PASS=8, FAIL=12** — the defeat test catches the defect ~60% of the time as a single run |
| against **unmodified** code, `-race -count=15` on both suspension tests | **30/30 PASS** — no flakiness in the GREEN direction |

The spec (`S-APP-016`) requires the RED "under `go test -race -count=15` … per `NFR-APP-002`'s repeated-run discipline". That command-level bar **is met, 6/6**. What is not true is `apply-progress.md:244`'s stronger claim: "**Result: 15/15 runs FAIL** — genuine, unconditional RED, exceeding the `S-PPB-002` 20/20 reliability bar proportionally."

---

## Spec compliance matrix

### `agent-run-driver` — `R-RUN-001…012` (25 scenarios, 12 requirements)

| Requirement | Scenario | Covering test | Status |
|---|---|---|---|
| `R-RUN-001` | `S-RUN-001` | `TestHarness_StructLiteralRun_NoConstructorFieldsUnchanged` (`harness_test.go:957`) — 3 sub-tests: nil→locals, `LeaveSinkOpen` exception, `reflect` two-exported-methods | ✅ PASS |
| `R-RUN-001` | `S-RUN-002` | `TestHarness_SteerAfterTerminal_TypedRejectionNoSilentDrop` (`:1039`) | ⚠️ PASS with MINOR-9; **requirement itself violated on the failure path — MAJOR-1** |
| `R-RUN-002` | `S-RUN-010` | `TestHarness_TwoTurnRun_CompletesToTerminal` (`:1119`) | ✅ PASS |
| `R-RUN-002` | `S-RUN-011` | `TestHarness_EachTerminalCandidateFinishReason_EndsAfterOneTurn` (`:1216`), 5 sub-cases | ✅ PASS — vacuity concern **REFUTED**, see below |
| `R-RUN-002` | `S-RUN-012` | `TestHarness_SteerNearTerminal_AtomicQueueCheckYieldsAdditionalTurn` (`:1284`) | ✅ PASS |
| `R-RUN-003` | `S-RUN-020` | `TestHarness_EventStream_OneRunBracketContiguousLane_CheckStreamAccepts` (`:1385`) — `CheckStream` + per-event `Sequence()==i+1` | ✅ PASS |
| `R-RUN-004` | `S-RUN-030`, `S-RUN-031` | `TestHarness_RunIdentity_ConsistentAcrossEventsAndProvenanceDistinct` (`:1445`) | ✅ PASS |
| `R-RUN-005` | `S-RUN-040` | `TestHarness_History_AlternatingTranscriptEveryPairMatched` (`:1503`) | ✅ PASS |
| `R-RUN-005` | `S-RUN-041` | `TestHistoryRouteGuard_SurfaceMatchesExpectedTable`, `import_boundary_test.go`, `ambient_authority_test.go` — all green, all byte-unchanged (verified by per-file `git diff --stat`) | ✅ PASS (command evidence) |
| `R-RUN-006` | `S-RUN-050` | `TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard` (`:1578`) | ✅ PASS — non-vacuity **independently proven**, see below |
| `R-RUN-007` | `S-RUN-060`, `S-RUN-061` | `TestHarness_RunStream_ReconstructsHistoryAtRunScope` (`:1803`) + bite (`:1823`) | ✅ PASS — bite re-executed, non-vacuous |
| `R-RUN-008` | `S-RUN-070` | `TestHarness_MidTurnSteer_EntersAtBoundaryBeforeNextProviderCall` (`harness_steering_test.go:62`) — recorded-request proof at `:148-160` | ✅ PASS |
| `R-RUN-008` | `S-RUN-071` | `TestHarness_SteerBurst_ArrivalOrderZeroDrops` (`:208`) — N=5, second goroutine, tool-calling turn one so the burst genuinely lands in `drain()` | ✅ PASS |
| `R-RUN-008` | `S-RUN-072` | `TestHarness_FinalTurnSteer_YieldsNewTurn` (`:298`) | ✅ PASS |
| `R-RUN-008` | `S-RUN-073` | `TestHarness_SteerAfterTermination_QueueClosedTypedRejection` (`:363`) — called twice | ✅ PASS |
| `R-RUN-009` | `S-RUN-080` | `TestHarness_PauseFinish_ResumesVerbatimToRealTerminal` (`harness_pause_test.go:29`) — byte-exact round-trip token + recorded-request replay | ✅ PASS |
| `R-RUN-009` | `S-RUN-081` | `TestHarness_PauseFinish_TurnEndCarriesPausedOutcomeVisibleAndForwarded` (`:111`) — verified: **no `ai.FinishReason` is read anywhere in the scenario** | ✅ PASS |
| `R-RUN-010` | `S-RUN-090` | `TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake` (`harness_suspension_test.go:171`) — in-bracket placement, `CheckStream`, non-blocking completion read | ✅ PASS |
| `R-RUN-010` | `S-RUN-091` | `TestHarness_PermissionDefer_ParkedWaitObservedBite` (`:272`) | ⚠️ PASS — bite effective; recorded evidence overstated (MAJOR-2) |
| `R-RUN-011` | `S-RUN-100` | `TestHarness_TurnError_RunEndsTypedNoAppendNoCloseNoRetry` (`harness_test.go:1863`) — `hist.Len()==1`, exactly 1 provider request | ✅ PASS |
| `R-RUN-012` | `S-RUN-110` | `git diff 5590afa0..HEAD` — 10 files, all allowlisted; every `R-LSK-004` file + `history.go` + `history_surface_guard_test.go` + `failure.go` + `turn_events.go` byte-unchanged (per-file diff scan, all empty); `go.mod`/`go.sum` diff empty; `src/ai/**` diff empty | ✅ PASS (command evidence) |
| `R-RUN-012` | `S-RUN-111` | Both filters extracted and `diff`ed: **29 entries, identical, in identical order**, every one an exact `/filename` suffix, zero wildcard/prefix/directory patterns, all five `/harness*.go` suffixes present | ✅ PASS (command evidence, MINOR-6) |
| `R-RUN-012` | `S-RUN-112` | Full suite green; `loop_test.go` diff is one hunk at `@@ -890,7 +890,25 @@` (the filter only); `scheduler_test.go`, `permission_protocol_test.go`, `turn_termination_test.go`, `turn_failure_test.go`, `loop_permission_e2e_test.go`, `loop_tool_dispatch_test.go` absent from the branch diff entirely | ✅ PASS |

**Non-functional**: `NFR-RUN-001` ✅ (all five `harness*` files declare `package agent_test`, grep-verified and machine-enforced by `S-RUN-050`'s own directory scan) · `NFR-RUN-002` ✅ (no `time.Sleep`/`net.`/`http.`/`os.Open`/`exec.` in any of the five new files; `harness.go` imports only `context`, `strconv`, `sync`, `sync/atomic`, `src/ai`) · `NFR-RUN-003` ✅ (`-race` green over 100+ iterations; stamper touched only at `harness.go:229` and `:302`, both outside any turn) · `NFR-RUN-004` ✅ 87.94% · `NFR-RUN-005` ✅ single PR under pre-authorised `size:exception`.

### Delta-owned requirements

| Requirement | Scenarios | Covering evidence | Status |
|---|---|---|---|
| `R-LSK-001` (MODIFIED) | `S-LSK-013`, `S-LSK-014` (+ 8 pre-existing reproduced verbatim) | `TestTurn_ContinuationNonNil_NoRunBracketsSharedIdentityAndLane` (`harness_test.go:262`), `TestTurn_ContinuationToolCall_ScheduleBeforeFinalize_EventsInsideTurnBracket` (`:428`), `TestTurn_ContinuationHalfConfigured_RejectedTypedNoEmission` (`:35`, table-driven over all 4 members) | ✅ PASS |
| `R-LSK-002` (MODIFIED) | `S-LSK-015` (+ `S-LSK-004`) | The three enumerated tests re-run individually: all PASS, and `loop_test.go`'s only branch hunk is the filter widening | ✅ PASS |
| `R-LSK-004` (MODIFIED) | `S-LSK-016` (+ `S-LSK-006`, `S-LSK-012`) | Merge-base diff + per-file byte-unchanged scan + filter comparison | ✅ PASS |
| `R-LSK-006` (MODIFIED) | `S-LSK-017` (+ `S-LSK-008`, `S-LSK-008a`) | `grep -rn "\.Schedule(" src/agent/*.go \| grep -v _test` returns **exactly two sites, both in `loop.go`** (`:368` nil path, `:465` continuation path), and they are mutually exclusive — `loop.go:346-348` returns `finishContinuationTurn` before `:368` is reachable. `harness.go` has **zero** `Schedule` occurrences. | ✅ PASS (MINOR-5 on the second half) |
| `R-HIS-010` (ADDED) | `S-HIS-090…094` | `TestTurn_ContinuationCommitsAssistantAndToolResults_OpenSetEmptyAtClose` (`:549`), `…EmptyContent_AppendsNothing` (`:695`), `…MixedOutcomes_OneResultPerCallInOrder` (`:748`), `…AppendFailure_TypedErrorReturned` (`:875`), `…Nil_HistorySurfaceGuardStaysGreen` (`:916`) | ✅ PASS |
| `NFR-HIS-003` (MODIFIED) | `S-HIS-095`, `S-HIS-096` | Merge-base diff; `TestHistoryRouteGuard_SurfaceMatchesExpectedTable` green, source byte-unchanged; `history.go` diff = **0 lines** | ✅ PASS |
| `R-APP-002` (MODIFIED) | `S-APP-015`, `S-APP-016` (+ `S-APP-003`) | `TestHarness_PermissionDefer_RunSurvivesSuspensionAcrossWake`, `…ParkedWaitObservedBite` | ⚠️ PASS (MAJOR-2) |
| `R-ATT-004` (MODIFIED) | `S-ATT-013` (+ `S-ATT-005`, `S-ATT-006`, `S-TTB-003`) | Both `harness_pause_test.go` tests | ✅ PASS |
| `R-TLS-012` (ADDED) | `S-TLS-013`, `S-TLS-014`, `S-TLS-015` | `TestSchedule_LeaveSinkOpenZeroDefault_ClosesSinkUnchanged` (`:119`), `TestSchedule_LeaveSinkOpenSet_CallerOwnsClose` (`:168`); `scheduler_test.go` absent from the branch diff | ✅ PASS |

**Code/spec twin re-home (`agent-tool-scheduler/spec.md:41`, task 6.2)**: ✅ `grep -n "AG-13\|AG-20" backend/agent/src/agent/tool.go` → `tool.go:249` now reads ``` `ToolSource` port (G6) is AG-20's widening ```; the only remaining `AG-13` mentions are the `LeaveSinkOpen` seam comments at `:232` and `:236`. Spec and code agree.

---

## The six charter Gherkin scenarios

| Charter scenario | Closed by | Verdict |
|---|---|---|
| `0003:1319-1323` "a two-turn conversation runs to its terminal" | `TestHarness_TwoTurnRun_CompletesToTerminal` (stream) + `TestHarness_History_AlternatingTranscriptEveryPairMatched` (transcript, every pair matched, `CloseTurn` succeeds) | ✅ **fully closed, not weakened** |
| `0003:1325-1328` "the event stream is the complete story" | `S-RUN-060` **at run scope** (partitioned by turn bracket, both turns) + the re-executed `S-RUN-061` bite | ✅ **fully closed**. Scope caveat, explicitly recorded in the test at `harness_test.go:1797-1802` and legitimate: the comparison excludes the run's initial prompt, which is never emitted as an event — the charter says "every message and tool outcome **the history holds**", and the prompt is harness-appended seed data, not turn output. The exclusion is `[1:]` on the transcript only. Reasoning parts are also out of the reconstruction's scope (documented at `:1670-1674`) — `message_end_reasoning` carries no round-trip token — but `R-RUN-009`/`S-RUN-080` closes reasoning byte-identity directly against the returned message instead. Neither exclusion weakens the charter clause. |
| `0003:1330-1333` "the harness holds no privileged channel into the loop" | `TestHarness_LoopAccess_PublicOneTurnSurfaceOnly_SourceScanGuard` | ✅ **fully closed**, and I proved the guard non-vacuous myself (below) |
| `0003:1343-1346` "a mid-turn user message queues to the boundary" | `TestHarness_MidTurnSteer_EntersAtBoundaryBeforeNextProviderCall` | ✅ **fully closed** — the "before the next provider call" clause is closed by `provider.Requests()[1].Messages()`, i.e. by recorded-request evidence, exactly as `R-RUN-008` demands, not by transcript order |
| `0003:1348-1352` "queued messages keep arrival order and are never dropped" | `TestHarness_SteerBurst_ArrivalOrderZeroDrops` (arrival order, N=5, concurrent) + `TestHarness_FinalTurnSteer_YieldsNewTurn` (final-turn clause) | ✅ **fully closed**, and the two exercise genuinely different mechanisms (`drain()` vs `takeOrClose()`) — the apply agent's Batch-2 Issue #2 correction is real and I confirmed the shapes differ (`heldToolCallScript` ends `FinishReasonToolCalls` → the `continue` branch; `heldTurnScript` ends `Stop` → the atomic branch) |
| `0003:1362-1366` "pause resumes with verbatim replay" | `TestHarness_PauseFinish_ResumesVerbatimToRealTerminal` + `…TurnEndCarriesPausedOutcomeVisibleAndForwarded` | ✅ **fully closed** — both Then-clauses split into their own test; the "not silently absorbed" half asserts on `TurnOutcome` values only |

**No charter scenario is closed by a weaker property than it states.**

---

## Defeat tests I ran myself (beyond the two mandated bites)

Every one below was executed; none is inference.

| # | What I defeated | How | Observed |
|---|---|---|---|
| D1 | Deviation A — mid-stream-fatal `run_end` suppression (`loop.go:400`) | `-overlay`: `if opts.Continuation == nil` → `if true` | **1 test fails**: `TestTurn_ContinuationMidStreamFatal_TurnEndAbortedNoRunEnd`. ✅ genuinely covered |
| D2 | Deviation B — the `R-HIS-010` `History.Append` wiring (`loop.go:470-485`) | `-overlay`: both append blocks removed | **8 tests fail**: `TestHarness_History_AlternatingTranscriptEveryPairMatched`, `…MidTurnSteer…`, `…PauseFinish_ResumesVerbatimToRealTerminal`, `…RunStream_ReconstructsHistoryAtRunScope`, `…SteerBurst…`, `TestTurn_ContinuationAppendFailure…`, `…CommitsAssistantAndToolResults…`, `…MixedOutcomes…`. ✅ strongly covered |
| D3 | `S-RUN-011` vacuity concern (`apply-progress.md:362`) | `-overlay`: `harness.go:284` dispatch made to iterate on `ai.FinishReasonRefusal` | **`TestHarness_EachTerminalCandidateFinishReason_EndsAfterOneTurn/refusal` FAILS.** The concern is about a *stub that could only run one turn* — an artefact of RED sequencing. Against the **shipped** code the test genuinely discriminates a terminal-candidate-routed-to-iterate defect. **REFUTED as a vacuity concern.** |
| D4 | `S-RUN-050` source guard, forbidden-symbol half | off-worktree copy: `_ = emitStamped` inserted into `harness.go` | **FAILS** at `harness_test.go:1599` with the exact violation message. ✅ non-vacuous |
| D5 | `S-RUN-050` source guard, `Schedule` regex half | off-worktree copy: a real `sched.Schedule(...)` call site inserted | **FAILS** at `harness_test.go:1605`. ✅ non-vacuous |
| D6 | Task 5.6 staleness settlement (`apply-progress.md:257-264`) | `-overlay`: `scheduler.go:635-643` ack-wait `select` deleted | **Exactly ONE test fails across the whole package**: `TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery`. **Independently confirms** the apply agent's settlement: `agent-permission-protocol/spec.md:172`'s "deleting the acknowledgement leaves the package green" is **STALE**. |

**Methodology note worth recording**: `-overlay` cannot defeat-test `S-RUN-050`, because the guard reads `harness.go` from **disk** at runtime (`harness_test.go:1581`), not from the compiler's overlay. D4/D5 were therefore run on a full off-worktree copy of `backend/agent`. Any future agent that "proves" this guard with `-overlay` and sees a PASS has proved nothing.

---

## Vacuity concerns raised by the apply agent — verified or refuted

| Concern (`apply-progress.md`) | My finding |
|---|---|
| `S-RUN-011` passes against a single-turn stub (`:362`) | **REFUTED as a defect.** Honest about the RED sequencing, but the shipped test discriminates a real defect (D3). No action needed. |
| `takeOrClose()`'s race-freedom rests on inspection, not a forced race (`:364`) | **ACCEPTED as a scoping choice.** `harness.go:119-129` is a single `Lock`/`defer Unlock` spanning both the length check and the close, and `enqueue` (`:87-95`) takes the same mutex — there is exactly one lock acquisition, so there is no check-then-act window to interleave. Every ordering is correct: if `Steer` wins, `takeOrClose` sees `len>0` and takes; if `takeOrClose` wins, `Steer` sees `closed` and is rejected typed. `-race` is green over 100+ iterations. A synthetically forced race would add no information here. No finding. |
| `ToolOutcomeExecutionFailure`'s content mapping (`Failure.Unwrap().Error()`) is spec-unpinned (`:356`) | **ACCEPTABLE as an implementation choice** — the spec deliberately pins only that the outcome rides the Layer 1 failure form, and `S-HIS-092` asserts exactly that (`Failed()`, `CallID()`). But the *content bytes* are asserted by **no test** — see MINOR-8. |

---

## Design coherence

| Design decision | In code | Status |
|---|---|---|
| Decision 1 — nil-default `TurnOptions.Continuation`, all-or-nothing | `loop.go:127-193` (`TurnContinuation`, `validateContinuation`), validated as the **first statement** of `Turn`'s body (`loop.go:256-259`) before any emission or identity minting | ✅ as designed |
| Decision 2 — inject a caller-owned `*Scheduler` | `harness.go:64`, `:257-262`; `WakeParked` reached by the test, never handed back by `Turn` | ✅ as designed |
| Reconciliation — the harness never calls `Schedule` | zero `Schedule` occurrences in `harness.go`; both call sites in `loop.go`, mutually exclusive | ✅ as designed |
| Schedule-before-finalize, continuation only | `loop.go:456-486`; nil path keeps finalize-first (`:349-380`) | ✅ as designed |
| Exactly one appender per message class | harness appends prompt (`harness.go:231`) + steered (`:240`, `:290`); `Turn` appends assistant + results (`loop.go:471`, `:479`); `Turn` never calls `CloseTurn` (grep-verified) | ✅ as designed |
| `LeaveSinkOpen` zero-default field on `Scheduler` | `tool.go:232`, `scheduler.go:228-234` | ✅ as designed |
| Value form, no constructor, two methods | `reflect`-asserted at `harness_test.go:1018-1031` | ✅ as designed |
| Run algorithm step (1) "validate the prompt" **before** step (2) "emit run-open" | Not literal — see MINOR-7 | ⚠️ benign deviation |

---

## Findings

### CRITICAL — 0

None. No test fails, no task is unchecked, and no spec scenario lacks passing covering evidence.

### MAJOR-1 — `Steer` silently drops a message after the run terminates through any failure path

- **Requirement violated**: `agent-run-driver/spec.md:68` (`R-RUN-001`) — "`Steer` MUST guarantee **zero drops**: a `Steer` returning nil means the message enters the transcript before a subsequent provider call. … never a silent drop and never a nil return."
- **File/lines**: `backend/agent/src/agent/harness.go` — `failRun` (`:187-194`) and every one of its six call sites (`:232`, `:241`, `:274`, `:278`, `:291`) plus the `NewRunStart` early return (`:225-228`). **None of them closes `h.queue`**; only `takeOrClose` (`:127`) ever sets `closed = true`, and it is reachable only on the success path (`:287`).
- **Consequence**: after a run that ended via `R-RUN-011`'s failure path (or a rejected prompt append, a rejected steered append, or a `CloseTurn` failure), `Steer` returns **nil** forever. A nil return is the harness's own promise that the message will enter the transcript before a subsequent provider call. There is no subsequent provider call. That is precisely a silent drop with a nil return.
- **Evidence command** (probe added via `-overlay`, worktree untouched):
  ```
  go test -overlay=… -race -v -run 'TestVerifyProbe' ./src/agent/
  zz_probe_test.go:29: PROBE: Steer after an R-RUN-011-failed run returned: <nil>
  --- FAIL: TestVerifyProbe_SteerAfterFailedRun
  zz_probe2_test.go:50: PROBE: Steer after a prompt-append failure returned: <nil>
  --- FAIL: TestVerifyProbe_SteerAfterPromptAppendFailure
  ```
- **Why no scenario caught it**: `S-RUN-002` and `S-RUN-073` both take the *successful* terminal path. `R-RUN-011` enumerates what the harness must do on the failure path (emit `run_end`, no append, no `CloseTurn`, no retry) and is silent about the queue. The two requirements underdetermine their interaction — this is a spec gap as much as a code gap.
- **Disposition**: this requirement is about to be **promoted to `openspec/specs/agent-run-driver/spec.md`** at archive. Promoting a MUST that the shipped code demonstrably violates is the spec-staleness class this repository has recorded repeatedly. Resolve before archive, either way:
  - **code**: close the queue on every `Run` exit — one line in `failRun` plus one at the `NewRunStart` early return, plus one test; or
  - **spec**: one sentence in `R-RUN-001` scoping the typed rejection to the *terminal-decision* exit and stating explicitly that the failure path leaves the queue open, with the reason.

### MAJOR-2 — `apply-progress.md`'s "15/15 runs FAIL" for the `S-RUN-091` defeat test is not reproducible

- **File/line**: `openspec/changes/cachicamas-agent-run-driver/apply-progress.md:244` — "**Result: 15/15 runs FAIL** — genuine, unconditional RED, exceeding the `S-PPB-002` 20/20 reliability bar proportionally", and `:255` "no run passed", and `:323` "fails the bite 15/15 under `-race -count=15`".
- **Observed**: 6 independent `-race -count=15` invocations under a byte-equivalent scratch produced **2 to 7 PASSING iterations of 15 every time**; 20 independent single-iteration runs produced **8 PASS / 12 FAIL**. Full data in the table above.
- **What is and is not affected**: the scenario's own bar — a `-race -count=15` **command** that goes RED — is met, **6/6**. The bite is real and it does observe the wait. What is wrong is the recorded reliability claim, and the inference drawn from it ("exceeding the `S-PPB-002` 20/20 bar"). It does not exceed that bar; it is materially weaker than `S-PPB-002`'s deterministic 20/20.
- **Why it matters**: `S-APP-016`/`S-RUN-091` is the bite that closes a **known gap carried across three milestones**. A future reader who trusts "15/15, unconditional" will believe the guard is deterministic and will not re-measure. The honest record is "the aggregate `-count=15` command is a reliable RED; a single run is ~60%".
- **Disposition**: correct `apply-progress.md:244`, `:255` and `:323` to the measured numbers. Optionally strengthen the bite so a single run is deterministic — the flakiness is a genuine race between the test setting `wakeIssued` and the scratch's immediate re-resolution, and could be removed by having `driveSuspendedRun` set `wakeIssued` before the sink read rather than after — but that is a design question, not a defect in shipped behavior, and **must not be attempted without re-proving the bite still observes the wait rather than the registration**.

### MINOR findings

| # | Finding | File / line | Evidence |
|---|---|---|---|
| 3 | `apply-progress.md`'s headline says **74/74** tasks; its own per-phase breakdown on the same line sums to **84**, and `tasks.md` holds 84 tasks. All 84 are ticked, so no task is incomplete — the headline number is simply wrong. This is the "count assertions are a drift class" pattern. | `apply-progress.md:8` (also `:5`, `:382`) | `grep -oE "^- \[[x ]\] ([0-9]+)\." tasks.md \| sort -n \| uniq -c` → 6,18,30,9,3,6,12 = 84; `grep -cE '^- \[ \]'` = 0 |
| 4 | `agent-run-driver/spec.md:22` makes it a MUST that the scenario count be "stated identically in `tasks.md`, `apply-progress.md` and `verify-report.md`". `tasks.md:3` states it; **`apply-progress.md` does not state it at all**. | `apply-progress.md` (absent) | `grep -niE "25 scenario\|12 requirement\|are bites\|43" apply-progress.md` → no match |
| 5 | `S-LSK-017`'s second half — "the per-tool schedule-invocation counter … reports exactly one invocation per turn and no invocation attributable to the harness itself" — has **no dedicated schedule-invocation counter in a harness-driven run**. It is discharged compositionally (`tool.Invocations()==1` in `S-RUN-010`, plus the source guard, plus `S-LSK-008`/`008a` on a bare `Turn`). Sound, but the scenario's literal wording is not literally asserted. Honestly recorded at `apply-progress.md:318`. | `harness_test.go:1197` | reading + `apply-progress.md:318` |
| 6 | `S-RUN-111` (the two substrate filters are byte-in-sync) has **no machine guard** — it is a command-level check only, so future drift between `filterOutLoopFiles` and `filterOutLoopHookFiles` is invisible to the suite. This is inherited from AG-11, not introduced here. | `loop_test.go:890`, `loop_hook_test.go:963` | `awk`-extract + `diff` → identical 29 entries; but nothing in the suite compares them |
| 7 | `R-RUN-002` step (1) says "resolve defaults and **validate the prompt**", before step (2) "emit the run-open event". `Run` does not validate the prompt; validation is implicit in the step-(3) `hist.Append(prompt)`. An invalid prompt therefore emits `run_start` **then** `run_end(Failed)` rather than being rejected before any emission. Benign — the resulting stream is well-formed and the error is typed — and arguably better for the consumer, but it is not what the requirement's ordering says, and no scenario covers it. | `harness.go:225-233` vs `spec.md:79` | probe: `zero-value prompt -> err=messages[0]: required value is empty, emitted kinds=[run_start run_end]`; `CheckStream` accepts |
| 8 | The tool-result **content** for `ToolOutcomeExecutionFailure` (`f.Unwrap().Error()`) is asserted by **no test**. `S-HIS-092` asserts only `Failed()` and `CallID()`; `S-RUN-060`'s fixture uses a succeeding tool, so the execution-failure branch of `reconstructRunScope` (`harness_test.go:1756-1762`) is never compared against `toolResultMessage`'s. A regression in that mapping would be invisible. | `loop.go:509-514`; `harness_test.go:1756-1762` | reading + fixture inspection (`runTwoTurnScenarioForReconstruction` uses `EchoScriptedTool`) |
| 9 | `S-RUN-002`'s third clause — "no further event reaches the consumer sink" — is not asserted by `TestHarness_SteerAfterTerminal_TypedRejectionNoSilentDrop`; the sink has already been drained to close before `Steer` is called. The property is true by construction (`defer close(sink)` at `harness.go:220`), but unasserted. | `harness_test.go:1039-1076` | reading |
| 10 | `S-HIS-091` (`TestTurn_ContinuationEmptyContent_AppendsNothing`) is a pure negative assertion and **also passes with the entire `R-HIS-010` wiring removed** (confirmed: it is absent from D2's 8-test failure list). Honestly recorded in the test's own banner (`harness_test.go:690-694`) and in `apply-progress.md:54`. No action needed — it is covered jointly by `S-HIS-090`/`092`/`093`. | `harness_test.go:695` | D2 run |

### Informational — not findings

- **gofmt**: 18 files under `src/agent/` are gofmt-non-conformant under the local Go 1.26 toolchain, including `loop.go`, `scheduler.go` and `tool.go`. **All three were already non-conformant at the merge base** (`gofmt -l` on `git show 5590afa0:…` lists all three), and the `turnAccumulator` struct block AG-13 touched was already in the same misaligned state. `.golangci.yml` enables `govet`/`errcheck`/`staticcheck`/`unused`/`revive` and **no formatter**, so `make lint` correctly passes. The five new AG-13 files are all gofmt-clean. Pre-existing, repo-wide, out of scope.
- **Filesystem access in tests**: `harness_test.go` calls `os.ReadFile`/`os.ReadDir`, which reads oddly against `NFR-RUN-002`'s "no filesystem". It is the source-scan guard `R-RUN-006`/`S-RUN-050` **mandates** ("the run driver's production source read as raw bytes … the `scheduler_test.go` regex precedent"), so it is spec-required, not a violation.
- **`drainSink`'s `time.After(time.Second)`** (`loop_test.go:159`) is a pre-existing bounded failure deadline, not an ordering mechanism; no new test synchronises on it. Consistent with `NFR-RUN-002`.
- **Coverage** was not re-derived; the orchestrator's independent measurement (87.94% weighted on `loop.go`) is carried forward as stated.

---

## Substrate and guard verification (all by command)

```
git diff --stat 5590afa0..HEAD -- backend/agent/src/ai/ backend/agent/go.mod backend/agent/go.sum   →  (empty)
per-file diff over all 22 R-LSK-004 / history / failure / turn_events files                          →  (all empty)
git diff 5590afa0..HEAD -- backend/agent/src/agent/history.go | wc -l                                 →  0
grep -rn "\.Schedule(" src/agent/*.go | grep -v _test.go                                             →  loop.go:368, loop.go:465 only
grep -c "Schedule(" src/agent/harness.go                                                             →  0
filterOutLoopFiles vs filterOutLoopHookFiles entry sets                                              →  IDENTICAL (29 entries, in order)
git status --short ; git diff --stat  (end of phase)                                                 →  (empty — worktree pristine)
```

`TestHistoryRouteGuard_SurfaceMatchesExpectedTable`, `TestEventRegistryDoc_StatesTheGuardsRecordedScope`, `TestDocContract_ScratchEdit_FailsBite`, `TestScheduler_SourceGuard_NoErrgroupImport`, `TestScheduler_SourceGuard_PolicySlotNoTypeAssertion`, `TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin` — all PASS, all source byte-unchanged.

---

## TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | ✅ | Three "TDD Cycle Evidence" tables in `apply-progress.md` (Batches 1–3), 24 task rows |
| All tasks have tests | ✅ | Every behavioral task row names an existing test file; Phase 6's rows are verification-only, correctly marked N/A |
| RED confirmed (tests exist) | ✅ | 24/24 named test functions exist and are reachable by `go test -run` |
| GREEN confirmed (tests pass) | ✅ | Full package suite re-run uncached under `-race`: 0 failures |
| Triangulation adequate | ✅ | Table-driven where the spec has multiple cases: `S-LSK-014` 4 members, `S-RUN-011` 5 terminal candidates, `S-HIS-092` 3 outcome classes, `S-RUN-071` N=5 burst, `S-RUN-073` called twice |
| Safety net for modified files | ✅ | Every batch records a full-package green baseline before starting |
| Composition proofs honestly marked | ✅ | Nine rows marked "➖ composition proof" rather than claimed as RED; six of those carry a separate scratch-revert causality proof. I re-executed three of them (D1, D2, D6) and all held. |

**Assertion quality**: ✅ No tautologies, no orphan empty-collection assertions, no ghost loops, no smoke-only tests, no mock-heavy files across the five new files. Every negative assertion (`no run_start/run_end`, `history.Len()==0`, `sink not closed`) has a companion positive assertion in the same or an adjacent test — with the single acknowledged exception of `S-HIS-091` (MINOR-10). The one assertion pattern worth naming as *strong*: `harness_test.go:91-108` and `:212-251` distinguish "channel empty" from "channel closed" by a non-blocking `select` **plus** a `recover`-guarded `close`, which is the only way to prove both in Go.

**Test layer distribution**: Unit 25 test functions across 5 files (`harness_test.go` 20, `harness_steering_test.go` 4, `harness_pause_test.go` 2, `harness_suspension_test.go` 2 — counting `harness_test.go`'s Phase 0/1 tests). Integration/E2E: none, and none is appropriate — Layer 2 forbids a real provider or a real tool (`0003:123`).

---

## Acceptance criteria (`spec.md:288-297`)

| # | Criterion | Verdict |
|---|---|---|
| 1 | Every `S-RUN-001…112` has recorded evidence; both bites RED-recorded before GREEN | ✅ — both bites independently re-executed by me (MAJOR-2 concerns the *reliability wording* of one RED record, not its existence) |
| 2 | All six charter Gherkin scenarios mapped and closed, none reduced | ✅ |
| 3 | `make test` green under `-race`; `make lint`, `make build`, `make vuln-check` clean | ✅ test/build/lint re-run by me; `vuln-check` accepted from `apply-progress.md:417` (not re-run — network-dependent installer) |
| 4 | `CheckStream` accepts the multi-turn stream unmodified, `stream_check.go` byte-unchanged | ✅ |
| 5 | Every existing bracket/sequence test and every AG-09/AG-10 scheduler test passes file-unchanged | ✅ |
| 6 | Both substrate filters byte-in-sync, exact filename suffix only | ✅ |
| 7 | All five cross-cut deltas written and every cited line re-read against the shipped change | ✅ — all six spec files read in full; every code citation I checked (`loop.go:465`, `scheduler.go:228`, `tool.go:232`/`:249`, `loop_test.go:890`, `loop_hook_test.go:963`, `harness.go:119`) resolves to what the delta claims |
| 8 | Doc 0003 `:2170` ticked and counters bumped to 13/24 | ✅ accepted from the orchestrator's own independent verification |

---

## Risks carried to archive

1. **MAJOR-1 is unresolved.** Archive promotes `agent-run-driver/spec.md` into `openspec/specs/`, making `R-RUN-001`'s zero-drop MUST canonical while the shipped code violates it on every failure path. Decide the disposition (one-line code fix, or one-sentence spec scoping) **before** promotion.
2. **MAJOR-2 is unresolved.** `apply-progress.md`'s reliability claim for the `S-APP-016` bite overstates the measured result. If the change is archived as-is, the overstatement becomes the permanent record for a gap that has already been carried across three milestones.
3. MINOR-3 and MINOR-4 are `apply-progress.md` bookkeeping defects that archive would freeze verbatim.
4. MINOR-6 (no machine guard on filter byte-in-sync-ness) is inherited debt that has now survived three milestones; it will keep surviving unless a milestone owns it.
