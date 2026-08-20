# Tasks: AG-21 — Harden concurrency, backpressure and leaks

> **Change**: `cachicamas-agent-concurrency-hardening` · Milestone 22 of 24 (Layer 2, Wave 6)
> **Size note**: exceeds the phase skill's 530-word default deliberately — 17 `S-CNH-*` scenarios, 7 bites and 6 delta discharges each need their own traceable row (`design.md`'s own size note is the precedent).
> **Runner (every evidence line)**: `cd backend/agent && go test -race -count=1 ./...` — `-count=1` MANDATORY. Real uncached suite ≈170s; a sub-second or otherwise implausibly fast pass is a **cache artifact**, not evidence. Cite the exact command and its real output/duration on every row below. Never mark a row done on a claim — only on a command.
> **Strict TDD**: every behavioral row is RED-first. Where a bite is named (a)–(g), it is its own row with its own RED/GREEN evidence line, never a clause folded into another row.

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (excl. `openspec/**`) | 830–1,495 |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR (reserve boundary U1/U2/U3 held, not taken) |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

Counted against the **1000-line budget excluding `openspec/**`**, extension pre-granted for AG-21 (user pre-authorized). Forecast is honest, not trimmed.

### Suggested Work Units (reserve boundary, held but NOT taken — D-A)

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| U1 | 12 combined-state × signal cells | single PR | `go test -race -count=1 -run TestCombinedMatrix ./...` | `agenttest.Gate`/`Provider`/`Script` fakes, no clock | revert `combined_matrix_test.go` + `combined_state_fixtures_test.go` + their 2 filter entries |
| U2 | 2 pressure scenarios | single PR | `go test -race -count=1 -run TestSlowConsumerPressure ./...` | unbuffered sink + resume channel, small injected `WindDownBound` | revert `slow_consumer_pressure_test.go` + its filter entries |
| U3 | leak sweep + cross-run scenario | single PR | `go test -race -count=1 -run 'TestConcurrencyHardening_PackageLeakSweep|TestCrossRun' ./...` | `agenttest.RequireNoGoroutineLeak` (50×), minted-nonce driver | revert `combined_leak_sweep_test.go` + `cross_run_state_test.go` + their filter entries |

---

## Phase 0 — AD-9 branch-scope fix (MUST be the first `sdd-apply` commit)

- [x] 0.1 **RED baseline (bite g, part 1).** On this branch, before any fix, run `go test -race -count=1 -run TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters ./...`; record the pre-existing three `t.Errorf` at `hooks_test.go:1990` as the fix's own RED baseline (`R-LSK-008`, `S-LSK-032` AG-21 amendment). **Evidence**: `cd backend/agent && go test -race -count=1 -run TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters ./...` → `FAIL`, exactly `hooks_test.go:1990: expected pre-existing non-test file "loop.go"/"harness.go"/"compaction.go" to have changed, it did not` (three lines), `FAIL github.com/cachicamas/backend/agent/src/agent 0.905s`.
- [x] 0.2 **GREEN fix.** Edit `hooks_test.go:1951-1993` — wrap BOTH the `allowed[]` loop (`:1977-1981`) and the `wantPreExisting` presence loop (`:1982-1992`) with two required guards, `t.Logf` + fall-through (never `t.Skip`): (i) `nonTestChanged` empty; (ii) `nonTestChanged` non-empty but missing the `hooks.go` authorship signature. Filter-entry half (`:1995-2010`) stays unconditional. **Done**: `switch { case len(nonTestChanged)==0: t.Logf(...); case !containsString(nonTestChanged,"hooks.go"): t.Logf(...); default: <original two loops> }`.
- [x] 0.3 **GREEN evidence.** Re-run the same `-run` filter with `-count=1`; record GREEN + wall-clock. **Evidence**: `cd backend/agent && go test -race -count=1 -run TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters -v ./src/agent/...` → `hooks_test.go:1991: AG-21: nonTestChanged is empty on this branch — ...`, `--- PASS (0.06s)`, `ok ... 1.513s`.
- [x] 0.4 **Bite (g), part 2.** Confirm the filter-entry half still executes on this non-AG-20 branch — cross-referenced to bite (f) (task 6.1) tripping it; record the confirmation once 6.1 runs. **Confirmed**: the filter-entry half (loopEntries/hookEntries assertions) ran unconditionally on every single test invocation in this apply batch (it is outside the branch-scoped `if !hksBaseRefIsHEAD` block); bite (f)'s own two-independent-REDs proof at task 6.1 is the formal load-bearing confirmation this row was reserved for — see Phase 6.

## Phase 1 — Fixtures & signal drivers (`R-CNH-002`)

- [x] 1.1 Create `combined_state_fixtures_test.go`: 4 state arrangers (suspension-pending, steering-queued, compaction-mid-run, child-harness-active) + 3 signal firers (interrupt, shutdown, provider-failure) + `runCombinedCell` core + the ONE new helper `requireHistoryPaired`, `package agent_test`.
- [x] 1.2 **Same commit — AD-10.** Add `/combined_state_fixtures_test.go` exact-suffix entry to BOTH `filterOutLoopFiles` (`loop_test.go:837`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907`).
- [x] 1.3 Prove each fixture against its own single-feature baseline before composition; record `-race -count=1` pass per fixture. **Evidence**: each fixture has no independent Test entry point (they are arrange-only helpers with no assertions of their own beyond fixture-construction `t.Fatalf`s), so "single-feature baseline" is discharged by Phase 2's own 12 cells, each of which drives exactly ONE state combined with exactly ONE signal — see task 2.7's full run, all 12 green.

## Phase 2 — 12-cell matrix (`R-CNH-001`, `R-CNH-002`; `S-CNH-001`…`004`, `005`)

- [x] 2.1 Create `combined_matrix_test.go`: `cellCase` table (12 rows) + `t.Run(state+"/"+signal, ...)` with `t.Parallel()`, driven through `runCombinedCell` — **S-CNH-001**.
- [x] 2.2 **Same commit — AD-10.** Add `/combined_matrix_test.go` entry to BOTH filters.
- [x] 2.3 Compaction×failure cell at its AD-2 adjusted Then: bracket closes failed, run continues to its own terminal — **S-CNH-004**. Implemented as `adjustedThenCompactionFailure`.
- [x] 2.4 Suspension×failure cell at its adjusted Then: failure at the first reachable provider boundary post-park; assert NO provider request recorded while parked — **S-CNH-005**. Implemented as `adjustedThenSuspensionFailure`, asserting `arr.requestsAtPark == 1`.
- [x] 2.5 Child-harness-active cell with containment: `validateStreamsSeparately` + `R-DEL-002` kind fence, leaf-first wind-down on interrupt/shutdown — **S-CNH-002**. Implemented as `containmentChildFailure` (provider-failure) + `containmentLeafFirst` (interrupt/shutdown). **Deviation discovered and recorded**: the child×provider-failure cell's parent run reaches `RunOutcomeCompleted` with a nil error, NOT `RunOutcomeFailed` — verified directly against the shipped `loop.go`/`scheduler.go` mechanics (Schedule's own return value is discarded by the loop, so a contained tool-execution failure never surfaces as a Turn-level error) and confirmed empirically by the passing test below; `adjustedThenChildFailure` encodes this rather than the table's uniform-Then shorthand, which describes the state whose OWN driving provider fails (suspension, steering), not a contained CHILD failure.
- [x] 2.6 **Bite (a), RED-first.** Delete `h.Interrupt()` from the steering×interrupt cell; run; record RED (`RunOutcomeCompleted != RunOutcomeInterrupted`, clean mismatch, not a hang) with `-count=1` + wall clock; revert; record GREEN — **S-CNH-003**. **RED**: `go test -race -count=1 -run 'TestCombinedMatrix/steering_queued/interrupt' -v ./src/agent/...` → `combined_matrix_test.go:61: steering_queued/interrupt: Run error = <nil>, want errors.Is(_, agent.ErrInterrupted)` / `run_end outcome = completed, want RunOutcomeInterrupted`, `FAIL ... 0.346s` (no hang). **Design correction discovered**: the naive sabotage (no-op the whole fire closure) hangs (nothing ever unblocks the held gate) instead of giving a clean mismatch; fixed by making the fire closure `h.Interrupt(); gate.Release()` (sequential, gate release as an always-fires safety net) and adding an always-present, ordinarily-unreached turn-two script — verified race-clean across 20 repeats (`-race -count=20`) before sabotage, then RED, then reverted. **GREEN**: `go test -race -count=1 -run TestCombinedMatrix -v ./src/agent/...` → all 12 `PASS`, `ok ... 1.329s`.
- [x] 2.7 Full-package evidence: all 12 cells green under `-race -count=1`, wall-clock recorded. **Evidence**: `cd backend/agent && go test -race -count=1 -run TestCombinedMatrix -v ./src/agent/...` → 12/12 `PASS`, `ok github.com/cachicamas/backend/agent/src/agent 1.487s` (wall clock via `time`: 1.695s total).

## Phase 3 — Stalled-consumer pressure (`R-CNH-003`, `R-CNH-004`; `S-CNH-007`, `009`)

- [x] 3.1 Create `slow_consumer_pressure_test.go`: unbuffered sink; consumer goroutine reads k events then blocks on a test-owned `resume` channel. Implemented as the predicate-driven `cnhStalledConsumer` (shared by both scenarios via `cnhStopAfterN`/`cnhStopAfterToolStart`), plus a `stalled` channel proving the consumer has genuinely stopped reading before any signal fires.
- [x] 3.2 **Same commit — AD-10.** Add `/slow_consumer_pressure_test.go` entry to BOTH filters.
- [x] 3.3 Scenario 1 (never cancelled): close `resume`, drain to completion; assert `CheckStream` valid and every committed-fact event present against the scripted identity set (count, kinds, call IDs); zero absence admitted — **S-CNH-007**. `TestSlowConsumerPressure_NeverCancelled_LosesNothing`.
- [x] 3.4 **Bite (b), RED-first.** Scratch-tree removal of one `sendStamped` from `windDownRun`'s tail; run; record RED naming the missing event, AND record `CheckStream` alone reports NO violation on the same scratch tree (independent-work proof); revert; record GREEN — **S-CNH-008**. **RED**: `go test -race -count=1 -run TestSlowConsumerPressure_CancelledUnblocksWithinBound -v ./src/agent/...` → `slow_consumer_pressure_test.go:282: pressure/cancelled: cost_session(Final) count = 0, want exactly 1 ...`, `FAIL ... 0.329s` — exactly one failure line (the `CheckStream` assertion above it in the same test did NOT fire, confirming `CheckStream` alone misses this class of loss). **GREEN after revert**: both pressure tests `PASS`, `ok ... 1.573s`; `git diff --stat harness.go` empty (clean revert).
- [x] 3.5 Scenario 2: consumer reads until `tool_start` of a cancellation-deaf `BlockingScriptedTool`, stalls; `h.Interrupt()` fires; `Scheduler{WindDownBound: smallWindDownBound}` injected; assert run **returns** (completion-channel read), typed `tool_end_execution_failure` / `*agent.DetachedCallError` — **S-CNH-009**. `TestSlowConsumerPressure_CancelledUnblocksWithinBound`.
- [x] 3.6 **Bite (e), RED-first.** Scratch tree with the armed bound removed; run under `go test -timeout`; record the hang as RED evidence; revert; record GREEN — **S-CNH-010**. **RED**: `runToolWithWindDown`'s `<-ctx.Done()` arm reduced to an unconditional `reply = <-resCh` (the timer/bound removed); `go test -race -count=1 -timeout 15s -run TestSlowConsumerPressure_CancelledUnblocksWithinBound -v ./src/agent/...` → `panic: test timed out after 15s`, goroutine dump shows `runToolWithWindDown` parked at the bare `<-resCh` (scheduler.go), `FAIL ... 15.400s`. **GREEN after revert**: `git diff --stat scheduler.go` empty; both pressure tests `PASS`, `ok ... 1.573s`; full `go test -race -count=1 ./src/agent/...` → `ok ... 7.138s`.

## Phase 4 — Cross-run state (`R-CNH-007`; also discharges `agent-run-driver` `R-RUN-001`, `agent-cancellation-tree` `R-CAN-002`/`R-CAN-005` + non-req `:202`/`:205`, `agent-turn-termination`'s multi-turn-state closure, `agent-loop-skeleton`'s `:263` closure — `S-CNH-014`…`016`)

- [x] 4.1 Create `cross_run_state_test.go`.
- [x] 4.2 **Same commit — AD-10.** Add `/cross_run_state_test.go` entry to BOTH filters.
- [x] 4.3 Nil-`History` branch: run 1 adversarial (interrupted mid-turn, genuinely open call) seeded with a uniquely minted nonce; run 2 on the same value; assert the nonce is absent from run 2's captured provider requests AND present in run 1's own read-back (anti-ghost floor) — **S-CNH-014**. `TestCrossRunState_NilHistory_AbsenceWithAntiGhostFloor`. **Mechanism discovery recorded**: a detached (cancellation-deaf, wind-down-bound-expired) call is NOT left "open" in `History` for `SynthesizeOrphans` to repair — `finishContinuationTurn` (loop.go) always commits the assistant message AND every call's result (including a detached call's own execution-failure result) atomically, together, before `Harness.Run`'s own iteration-boundary wind-down check is ever reached. `SynthesizeOrphans` only ever does real work on a call that a run's own turn machinery never touched at all (verified directly: `cancellation_interrupt_test.go`'s own two sub-tests, one seeded/synthesized, one in-flight/self-resolving, and the file's own comment on the second: "distinct from orphan synthesis, which repairs only a call Schedule never returned a result for"). The nil-History absence test therefore uses a genuinely in-flight (not necessarily orphan-synthesized) call; the shared-History test below uses the seeded-open-call shape specifically, since S-CNH-015 names R-HIS-007 synthesis explicitly.
- [x] 4.4 **Bite (d), RED-first.** Re-run the identical absence assertion against a deliberately shared `History`; record RED (nonce reaches run 2's request); revert; record GREEN — **S-CNH-016**. **RED**: `go test -race -count=1 -run TestCrossRunState_NilHistory_AbsenceWithAntiGhostFloor -v ./src/agent/...` → `cross_run_state_test.go:137: run 2's captured request(s) reference run 1's minted call identity 2 time(s), want 0 ...`, `FAIL ... 0.339s`. **GREEN after revert**: both cross-run tests `PASS`, `ok ... 1.316s`.
- [x] 4.5 Shared-`History` branch + inventory: caller-set `History` carries the `R-HIS-007` synthesized orphan into run 2's request; `sessionStarted` fires exactly once across both runs; an interrupt between runs is a no-op and run 2 completes (`cancelRun` cleared); `Steer` after run 1 is rejected typed and reopens for run 2; `shutdown` stays false on the interrupt path — **S-CNH-015**. `TestCrossRunState_SharedHistory_LegitimateCarryAndInventory`, using the seeded-open-call shape so `SynthesizeOrphans` genuinely fires; `shutdown`-stays-unlatched is proven implicitly (run 2's own acceptance — a latched flag would refuse it typed and emit nothing), since the field is unexported and unreachable from `package agent_test`. Both tests repeated `-race -count=20` clean, no flakiness.

## Phase 5 — Leak sweep (`R-CNH-005`, `R-CNH-006`; also discharges `agent-cancellation-tree` `R-CAN-008` + non-req `:202`; `agent-v1-scope` `S-AGS-067`'s sweep-exclusion clause — `S-CNH-011`…`013`)

- [x] 5.1 Create `combined_leak_sweep_test.go`: `TestConcurrencyHardening_PackageLeakSweep`, **NO `t.Parallel()`**, scenario callable running all 12 cells + both pressure drivers + the cross-run driver serially via `agenttest.RequireNoGoroutineLeak`. **Refactor required and done**: the 12 cells were already plain-function-driven (`runCombinedCell`, called directly, no subtest wrapper needed); the two pressure tests and two cross-run tests were extracted into callable `cnhDrivePressureNeverCancelled`/`cnhDrivePressureCancelledUnblocks`/`cnhDriveCrossRunNilHistory`/`cnhDriveCrossRunSharedHistory`, with their original `Test*` functions reduced to thin wrappers (`t.Parallel(); cnhDriveXxx(t)`) so existing standalone evidence is unchanged.
- [x] 5.2 **Same commit — AD-10 (5th/final entry, 10 total).** Add `/combined_leak_sweep_test.go` entry to BOTH filters.
- [x] 5.3 Enforce `NFR-HKS-005`/`NFR-CNH-003`: no driver samples `runtime.NumGoroutine()`; every gate released inline **and** carries `t.Cleanup(gate.Release)` at construction; permanently-stalled-observer leak structurally excluded — **S-CNH-011**. **Gap found and fixed during this phase**: the compaction and child-harness arrangers' interrupt/shutdown fire closures previously relied solely on `ctx.Done()` to unblock their own gate (harmless for the goroutine itself, since `ctx.Done()` already unblocks the `Hold`, but not an explicit inline release) — each now also calls its own gate's `.Release()` explicitly, inline, alongside `t.Cleanup`. The pressure-2 and cross-run-nil drivers' own deaf-tool `release` channels were similarly moved from `t.Cleanup`-only to an explicit `close(release)` immediately after the run returns — required for the sweep specifically, since a `t.Cleanup` registered on every one of 50 repeats would not fire until the whole sweep test ends, making every iteration's own detached goroutine look leaked at snapshot time. The observer-hook leak is excluded by construction: none of the four driver families gates a `SessionStart`/`PostTurn` observer open.
- [x] 5.4 Include the pressure-2 driver, releasing its deaf tool AFTER the run returns; assert the typed report observed + goroutine count stays within tolerance across the helper's repeats — **S-CNH-013**. Evidence: `TestConcurrencyHardening_PackageLeakSweep` passes with `cnhDrivePressureCancelledUnblocks` (its own typed-report assertions unchanged) inside the scenario set.
- [x] 5.5 **Bite (c), RED-first.** Plant `go func(){ <-neverClosed }()` once per iteration inside the sweep's scenario; run; record `RequireNoGoroutineLeak`'s `Fatalf` as RED (growth ≈50 > tolerance 25); revert; record GREEN — **S-CNH-012**. **RED**: `go test -race -count=1 -run TestConcurrencyHardening_PackageLeakSweep -v ./src/agent/...` → `combined_leak_sweep_test.go:56: agenttest: RequireNoGoroutineLeak: goroutine count grew from 2 to 52 across 50 repeats (tolerance 25), want no per-call leak (R-STK-007)`, `FAIL ... 2.922s` — growth of exactly 50, matching the design's own prediction precisely. **GREEN after revert**: `PASS (2.60s)`, `ok ... 3.916s`; `grep -n "SABOTAGE\|neverClosed"` on the file returns nothing.
- [x] 5.6 Record the sweep's measured wall-clock; if it exceeds 60s apply AD-5's recorded fallback (split into three non-parallel wrappers, one per leaf U1/U2/U3) — otherwise record the measurement as evidence. **Measured**: `time go test -race -count=1 -run TestConcurrencyHardening_PackageLeakSweep -v ./src/agent/...` → subtest `2.59-2.61s` across 4 repeated runs, package `ok` line `3.91-4.07s`, real wall-clock `5.6s`. Well under the 60s threshold — **AD-5's fallback is NOT applied**; the single-wrapper shape stands.

## Phase 6 — AD-10 exactness proof, scope fence, and cross-delta traceability (`S-LSK-034`, `S-CNH-006`, `S-CNH-017`; discharges `agent-event-delivery` `S-AGE-031` and `agent-v1-scope` `S-AGS-067`/`R-AGS-009`)

- [x] 6.1 **Bite (f), RED-first.** Delete ONE new AG-21 entry (e.g. `/combined_matrix_test.go`) from `filterOutLoopFiles` **only**; run suite; record TWO independent REDs — `TestTurn_SubstrateUntouched` (`"substrate was edited (R-LSK-004 violated)"`) AND `S-LSK-031`'s identical-entry-set failure; restore the entry; record GREEN — **S-LSK-034**. **RED**: `go test -race -count=1 -run 'TestTurn_SubstrateUntouched|TestScopeFence_S_LSK_031_SubstrateFiltersByteInSyncExactWidening' -v ./src/agent/...` → `loop_test.go:1360: substrate was edited (R-LSK-004 violated):` (FAIL) AND `scope_fence_test.go:153: filterOutLoopFiles has 78 entries, filterOutLoopHookFiles has 79, want the identical count` (FAIL) — both independent REDs observed together. **GREEN after revert**: `git diff --stat loop_test.go` empty; both `PASS`. This ALSO formally discharges task 0.4 (bite g, part 2): the filter-entry half executed on every invocation throughout this apply batch, including this one.
- [x] 6.2 **S-CNH-006.** Verify by construction (reading the drivers' own source): `agenttest/` diff against merge-base is empty; every new file declares `package agent_test`; no driver invokes `Run` on a harness value while another `Run` on that value has not returned. **Verified**: `git diff 54476ded -- backend/agent/src/agenttest/` empty; all 5 new files' first `package` line reads `package agent_test`; read every driver's own source directly — the only place any driver calls `.Run()` twice on the same harness value is `cross_run_state_test.go`'s two cross-run drivers, and in both, run 2's own goroutine is started strictly after `got1 := <-resultCh1` has already received run 1's own return.
- [x] 6.3 **S-CNH-017 / R-CNH-008.** Verify: empty diff under `src/ai/` and `agenttest/`; `go.mod`/`go.sum` byte-unchanged; event-kind guard at committed count; no new `TurnOutcome`/`CostLabel`; every production Go source under `src/agent/` byte-unchanged EXCEPT `hooks_test.go`/`loop_test.go`/`loop_hook_test.go` (delta-recorded); no assertion anywhere compares a duration, throughput or latency. **Verified**: `git diff 54476ded -- backend/agent/src/ai/` and `-- backend/agent/go.mod backend/agent/go.sum` both empty; `git diff 54476ded --name-only -- backend/agent/src/agent/` lists EXACTLY 8 files, all `_test.go` — the 5 new files plus `hooks_test.go`/`loop_test.go`/`loop_hook_test.go` — no production `.go` file appears at all (stronger than the required exception list); pre-existing `len(agent.EventKinds()) != 25` guards in `hooks_test.go` (unedited assertions) keep passing; `turn_events.go`/`cost_events.go` do not appear in the changed-file list, so no new `TurnOutcome`/`CostLabel` is possible; `grep -n "time\."` across all 5 new files shows only defensive `time.After` deadlines (never the signal itself) and the one legitimate `WindDownBound` ceiling — no assertion compares a duration/throughput/latency.
- [x] 6.4 **Traceability: `S-AGE-031`.** Confirm it is discharged by the cross-referenced `S-CNH-007`/`S-CNH-008` evidence (Phase 3) — no separate test file needed; record the cross-reference. **Recorded per this row's own instruction. Discrepancy flagged, not silently resolved**: the promoted delta text for `S-AGE-031` (read in full during this apply's required reading) states the stalled consumer must be observed while the run is "simultaneously in one of the four combined states of R-CNH-001" — my Phase 3 scenarios (`S-CNH-007`/`S-CNH-008`) are standalone stalled-consumer runs, not combined with a suspension/steering/compaction/child state. This row's own instruction is followed as assigned (no new test file was authorized or built here), but `sdd-verify` should confirm whether the delta's own literal text or this task's discharge claim is authoritative — this apply batch did not unilaterally expand scope to build the combined variant.
- [x] 6.5 **Traceability: `S-AGS-067`.** Confirm the four inherited postures are discharged by the shipped `S-CNH-011`/`012`/`013` (sweep exclusion + gate discipline, Phase 5) and `S-CNH-015` (session-start-latch-once, Phase 4) evidence — no separate test file needed; record the cross-reference. **Verified**: `grep -n "runtime\.NumGoroutine\|PreRequest:\|PreCompact:"` across all 5 new files returns nothing (postures 1-3 hold: no baseline sampling, no mutating-hook registration anywhere); exactly one `SessionStart:` registration exists, in `cross_run_state_test.go`'s shared-history driver, whose own assertions already prove the once-across-two-runs latch (posture 4).
- [x] 6.6 **Traceability: `R-AGS-009`/`S-AGS-035`.** Confirm doc 0003's forward "Requirements → closing nodes" row for R-05 (`0003:2204`) is left naming `AG-01.1, AG-20.2` and is NOT edited — AG-21 traces, it does not close. **Verified**: read `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` at the forward table — the R-05 row still reads exactly `AG-04.3, AG-05.1 (invariant 1); AG-04.1, AG-19.1 (invariant 2); AG-01.1, AG-20.2 (invariant 3); AG-04.3, AG-11.2 (invariant 4)`, AG-21 absent; the reverse "Nodes trace back to scope" table's own AG-21 row already reads `R-05 (invariant 3 under pressure), the assembled whole`. Doc 0003 not yet touched by this apply batch (reserved for Phase 8's own narrow edit).

## Phase 7 — Full-suite green + quality gates

- [ ] 7.1 `cd backend/agent && go test -race -count=1 ./...` — full green, wall-clock duration recorded (sub-170s is suspect).
- [ ] 7.2 `golangci-lint cache clean` then `make lint` — clean.
- [ ] 7.3 `make build` — clean.
- [ ] 7.4 `make vuln-check` — clean. **NOT** part of `make all`; do **NOT** run `make all` (its fmt step rewrites committed files).

## Phase 8 — doc 0003 update (same PR)

- [ ] 8.1 Read the Wave 0–6 table of contents (doc's own ToC, `:43-45`) to derive the shipped count — never increment the printed figure blindly; confirm it yields **22 of 24** after AG-21.
- [ ] 8.2 Append AG-21's outcome to the running status paragraph (`0003:3`), following the AG-19/AG-20 append pattern; update **"21 of 24"** → **"22 of 24"**.
- [ ] 8.3 Flip the checklist row at `0003:2179`: `- [ ]` → `- [x] The package is race-clean and leak-free under the combined-scenario matrix — closed by AG-21.`

## Phase 9 — `sdd-archive` promotion obligations (rows it discharges)

- [ ] 9.1 Promote the NEW capability `agent-concurrency-hardening` into `openspec/specs/agent-concurrency-hardening/spec.md` verbatim (AG-14/19/20 precedent).
- [ ] 9.2 `agent-event-delivery`: extend the scenario-ID header line to cover `S-AGE-031` (range, never a total). No other header line changes.
- [ ] 9.3 `agent-run-driver`: **NO header change** — Allocated-IDs line stays `R-RUN-001..013` / `S-RUN-001..113` verbatim.
- [ ] 9.4 `agent-cancellation-tree`: **NO header change** — no new `S-CAN-`/`R-CAN-` identifier minted.
- [ ] 9.5 `agent-turn-termination`: **NO header/ID change.**
- [ ] 9.6 `agent-v1-scope`: add `S-AGS-067` to the scenario range (range, never a total). No other header line changes.
- [ ] 9.7 `agent-loop-skeleton`: extend the scenario range to `S-LSK-034`; append the AG-21 amendment line (`R-LSK-009` ADDED, `R-LSK-008` MODIFIED, no substrate release requested, both filters widened by exact suffix for the 5 new files, one narrow normative release on `hooks_test.go`'s `S-LSK-032` block).

## Phase 10 — Post-archive verification (AG-20's truncation defect MUST NOT repeat)

- [ ] 10.1 `wc -l` each promoted spec target against its delta source (all 7: the new capability + 6 deltas) — confirm line counts consistent with a real body, not a truncated stub.
- [ ] 10.2 `grep -rln "Content continues|^## Key Learnings|skill_resolution|next_recommended|executive_summary" openspec/specs/` — MUST return empty (or only pre-existing matches unrelated to this change). This is the exact defect that shipped into `openspec/specs/agent-hook-taxonomy/spec.md` at AG-20 and was repaired in `c46b696b`; this row exists so AG-21 cannot repeat it.

---

## Traceability closure

Every requirement/scenario touched by this change's 7 spec files, mapped to its discharging task. Walked in reverse after drafting — **none unclaimed**.

| Capability | Element | Discharged by |
|---|---|---|
| `agent-concurrency-hardening` | `R-CNH-001`…`008` / `S-CNH-001`…`017` (all 17) | Phases 1–6 (see per-row IDs above) |
| `agent-event-delivery` | `R-AGE-008` back-annotation / `S-AGE-031` | 3.3, 3.4, 6.4 |
| `agent-event-delivery` | `S-AGE-010`, `S-AGE-011`, `S-AGE-030` | Unchanged — verified byte-identical (no task) |
| `agent-run-driver` | `R-RUN-001` cross-run inventory | Phase 4 |
| `agent-run-driver` | `S-RUN-003` (presence-only, unchanged) | Unchanged — verified byte-identical (no task) |
| `agent-cancellation-tree` | `R-CAN-002`, `R-CAN-005` cross-run clauses | Phase 4 |
| `agent-cancellation-tree` | `R-CAN-008` sweep-shipped text | Phase 5 |
| `agent-cancellation-tree` | non-req `:202` (sweep), `:205` (cross-run) | Phases 5, 4 |
| `agent-turn-termination` | multi-turn-state closure (`:181`) | Phase 4 |
| `agent-v1-scope` | `R-AGS-009`/`S-AGS-035` (R-05 trace, not closure) | 6.6 |
| `agent-v1-scope` | `R-AGS-016`/`S-AGS-067` | 6.5 |
| `agent-loop-skeleton` | `R-LSK-008`/`S-LSK-032` AG-21 amendment | Phase 0 |
| `agent-loop-skeleton` | `R-LSK-009`/`S-LSK-034` | Phases 1–6 (filter entries) + 6.1 (bite f) |
| `agent-loop-skeleton` | non-req `:263` closure | Phase 4 |

## What NOT to do (binding)

- No code, spec or design decisions — this artifact is a checklist only.
- No task beyond what design.md/specs call for.
- No PR-split or chaining recommendation.
- No task asserting a duration or wall-clock bound (`NFR-CNH-004`).
