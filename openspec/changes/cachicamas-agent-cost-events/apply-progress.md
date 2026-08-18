# Apply progress: AG-16 — Emit cost and usage events (`cachicamas-agent-cost-events`)

> Worktree `cachicamas-worktrees/ag16`, branch `feat/agent-layer2-wave3-ag16`, base `main@09bb30e1`. Strict TDD active (`cd backend/agent && make test`). Single PR, `size:exception` pre-accepted.

## Status: Phases 0-6 complete and committed. Phases 7-9 in progress / remaining.

## Task 0.1 — Substrate filter tail re-grep (do not trust cited line numbers)

Verified by fresh grep, NOT by design.md's citations:

- `filterOutLoopFiles` (`loop_test.go`): starts `:831` (matches design). Its AG-15 tail's last entry (`/failover_policy_test.go`) was at `:958` pre-AG-16, if-block closing `}` at `:959`, function closing `}` at `:966` — matches design's own citation exactly.
- `filterOutLoopHookFiles` (`loop_hook_test.go`): starts `:907`. Its AG-15 tail's last entry (`/failover_policy_test.go`) was at `:1019` pre-AG-16 (design did not cite this one; it was flagged "not re-measured" and required a fresh grep, which this was).

Both filters now carry an identical 46-entry set after AG-16's widening (verified task 6.2, below).

## Phase 0 — Foundation (commits `dc4f1a54`, docs in `4cf845b3`)

- [x] 0.1 Filter tails re-grepped (above).
- [x] 0.2 `usage ai.Usage` field added to `turnAccumulator` (`loop.go`); populated at the Completion case beside `finish`/`finishOk`. Inert — no emission wired yet. `make test` stayed green.
- [x] 0.3 RED: `cost_usage_test.go` — `TestCostFromUsage_TableDriven` (S-CST-007) referencing undefined `costFromUsage`/`costPresence`. RED = compile failure (`undefined: costPresence`, 10 errors), confirmed by running `go test ./src/agent/... -run TestCostFromUsage`.
- [x] 0.4 GREEN: `cost_usage.go` created — `costPresence{input,output,cacheRead,cacheWrite,reasoning bool}`, pure `costFromUsage(ai.Usage) (CostFigures, costPresence)`. Table test passes (8 sub-cases: empty, fully-reported, all-zeros, 5×mixed-single-figure, pairwise-distinct) plus a dedicated zero-value-payload test.
- [x] 0.5 `presence costPresence` field added to `CostTurn`/`CostSession` (`cost_events.go`) beside `figures`; 10 paired accessors (`InputTokens()` etc. × 5 figures × 2 types). `NewCostTurn`/`NewCostSession` set presence via `allReportedCostPresence` internally — exact signatures unchanged, `cost_events_test.go` untouched. Doc comments at the `CostLabelEstimate`/`CostLabelFinal` consts (originally `:96-103`) restated for the run-scoped label axis — the ONLY other edit inside the bounded release; the `CostLabel` type's own doc comment (`:85-89`) was deliberately left in its old wire-level framing, since the bounded release names only "the doc comments at `:96-103`" and nothing else (see Risks).
- [x] 0.6 `newCostTurnFromUsage`/`newCostSessionFromTotals` added to `cost_usage.go` (AG-14 `typedCancellationFailure` sibling precedent) — package-private, construct payload literals directly with real presence.
- [x] 0.7 `make test` green; `git diff` confirmed pure addition, no call site wired, no test assertion changed. Substrate filters widened for `cost_events.go`/`cost_usage.go`/`cost_usage_test.go` in the SAME commit (task 6.1's "land each entry in the same commit as the file it names" applied from Phase 0 onward, not deferred to a literal "Phase 6" commit).

**Deviation from tasks.md's literal phase order, recorded rather than silent**: Phase 6's substrate-filter widening was NOT done as one deferred batch. Each filter entry landed in the commit that introduced the file it names (Phase 0: `cost_events.go`/`cost_usage.go`/`cost_usage_test.go`; Phase 1+2: `cost_turn_emission_test.go`; Phase 3: `cost_session_test.go`) — this is what task 6.1's own text requires ("Land each entry in the same commit as the file it names first appears in") and is necessary for `TestTurn_SubstrateUntouched`/`TestTurn_PreRequestHook_SubstrateUntouched` to stay green throughout, rather than leaving a known-red window open across four phases.

## Phase 1 — `cost_turn` emission (commit `2e35ba78`)

- [x] 1.1 RED: `cost_turn_emission_test.go` — `TestTurn_CostTurn_FiguresExactPerTurn` (S-CST-001). 2-turn harness run (PauseTurn then Stop), pairwise-distinct 5-figure usage per turn. RED: `turn "turn-lsk-4": event immediately before turn_end is message_end_text, want cost_turn`.
- [x] 1.2 GREEN: `finalize()` (`loop.go`) emits `newCostTurnFromUsage(t.runID, t.turnID, CostLabelFinal, t.usage)` as its first emission, before `turn_end`. All Phase-1 RED tests confirmed GREEN (`TestTurn_CostTurn_FiguresExactPerTurn`, `TestTurn_NoCompletion_CostTurnAllAbsent`, `TestTurn_CostTurn_LabelAlwaysFinal`, `TestTurn_CostTurn_AbsenceVsReportedZero`, `TestTurn_CostTurn_MixedRecordPartialPresence`, `TestTurn_Standalone_NoCostSession`).
- [x] 1.3 `TestTurn_NoCompletion_CostTurnAllAbsent` (S-CST-002) — provider closes without Completion; `cost_turn` still emitted, all 5 absent.
- [x] 1.4 `TestTurn_AbortedTurn_NoCostTurn` (S-CST-003) — 5 subtests: mid-stream fatal, 3 pre-stream aborts (build error, hook error, provider.Stream error), cancellation mid-turn (Gate + `Harness.Interrupt()`). All PASS from the start (aborted paths never reach `finalize()`) — non-vacuous by construction/companion to S-CST-001's positive proof (see Evidence table note on vacuity).
- [x] 1.5 `TestTurn_CostTurn_LabelAlwaysFinal` (S-CST-004) — extended over Phase 3's retry-bearing stream at task 3.9.
- [x] 1.6 `TestTurn_CostTurn_AbsenceVsReportedZero` (S-CST-005).
- [x] 1.7 `TestTurn_CostTurn_MixedRecordPartialPresence` (S-CST-006).
- [x] 1.8 Bite S-CST-020: `costFromUsage` mutated to a whole-record `anyReported` flag (scratch). Re-ran 1.7: FAILED — `OutputTokens() = (0, true), want (0, false)` (and same for CacheRead/CacheWrite/Reasoning). RED recorded, reverted; `git diff --stat` confirmed clean revert.
- [x] 1.9 `TestTurn_Standalone_NoCostSession` (S-CST-011).
- [x] 1.10 `Turn`'s exported signature confirmed unchanged by reading the shipped signature (matches `agent-turn-termination/spec.md:113`/`NFR-ATT-004` verbatim) — no test edit, body-only change.

## Phase 2 — Blast-radius remediation (commit `2e35ba78`, same as Phase 1)

- [x] 2.1 Discovery grep for `len(got)`, `wantKinds`, `wantOrder` run BEFORE any edit. Full site list:
  - `loop_test.go:350-361` (S-LSK-001, `TestTurn_WalkingSkeleton_EmitsContractEventOrder`) — AFFECTED, amended.
  - `loop_test.go:1152-1165` — **NOT** the tool-dispatch test design.md/the loop-skeleton delta cite as "S-LSK-009"; it is actually `TestTurn_ReasoningPassThroughByteExact` (S-LSK-005, reasoning interleaving). Both design.md and the `agent-loop-skeleton` delta's own S-LSK-009 scenario annotation mis-cite this location — a genuine citation defect in the upstream planning artifacts, not a git-drift issue (verified against the same `main@09bb30e1` base both were written against). AFFECTED regardless of the label error; amended.
  - `harness_steering_test.go:103-126` (turn-one `wantKinds`) — AFFECTED, amended after determining (below) that the window does NOT cross into `cost_session`'s.
  - `retry_policy_test.go:110-131` (`assertPreStreamAbortSequence`) — NOT affected (pre-stream aborts never reach `finalize()`). Confirmed unedited, still green.
  - `harness_test.go:151-160,194-209` (`TestSchedule_LeaveSinkOpen*`) — NOT affected: these test `Scheduler.Schedule` directly, no `Turn`/`finalize()` involvement at all. Not previously enumerated by design.md's table but discovered by this phase's own grep and confirmed harmless.
  - `event_registry_test.go:409-417` — NOT affected: static kind-registry count, not a stream sequence; also a byte-unchanged-pinned file regardless.
- [x] 2.2 `loop_test.go`'s S-LSK-001 `wantKinds` amended: `EventKindCostTurn` inserted before `EventKindTurnEnd`. Two now-stale `//nolint:gosec // i+1 is always in [1,9]` comments (this test and the unrelated `TestTurn_TwoSequentialTurnsShareNothing`, S-LSK-004, whose second turn also gains a `cost_turn`) corrected to `[1,10]` — comment-only, no assertion change; `TestTurn_TwoSequentialTurnsShareNothing` itself needed no assertion edit since it checks `Sequence()` by array position, not by kind or fixed length.
- [x] 2.3 `loop_test.go`'s `wantOrder` (the S-LSK-005 reasoning test, actual location of the mis-cited "S-LSK-009") amended: `EventKindCostTurn` inserted before `EventKindTurnEnd`.
- [x] 2.4 `harness_steering_test.go`'s turn-one `wantKinds` — determination recorded: the collection loop breaks as soon as it observes `turn_end` (`if ev.Kind() == EventKindTurnEnd && inTurnOne { break }`), and `cost_session(Estimate)` is emitted by the harness strictly AFTER that `turn_end` (in `Run`'s own switch statement, after `<-forwarderDone`). The window therefore captures `cost_turn` (emitted before `turn_end`, inside the still-open window) but never reaches `cost_session` (emitted after the window already closed). `EventKindCostTurn` inserted; `EventKindCostSession` correctly NOT inserted.
- [x] 2.5 Every site NOT amended re-ran green, file-unchanged (confirmed via `git status --short` showing none of them modified, plus the full-package green run).
- [x] 2.6 Recorded above with reason and owning delta for each edited file.

## Phase 3 — Cumulative accumulator + estimate/final (commit `8b3b548d`)

- [x] 3.1 `costAccumulator{figures CostFigures; presence costPresence}` added to `cost_usage.go` with `add(c CostTurn)` (plain `+=` per figure, presence OR) and `sessionEvent(run, label) (Event, error)`.
- [x] 3.2 Forwarder in `Harness.Run` (`harness.go`) intercepts: `if ct, ok := ev.CostTurn(); ok { total.add(ct) }` before `sink <- ev`. `var total costAccumulator` declared as a **local**, moved to right after `stamper := &LaneStamper{}` (before `NewRunStart`/`hist.Append(prompt)`) rather than after `lastFinish` as first drafted — needed so it is in scope for the very first `failRun` call site (the `hist.Append(prompt)` failure, which precedes any turn).
- [x] 3.3 RED: `cost_session_test.go` — `TestHarness_CostSession_CumulativeEqualsEmittedCostTurns` (S-CST-008). Multi-turn run: logical turn one fails retryably (via `preStreamFailingProvider`, `retry_policy_test.go`'s own fixture — reused directly, not re-implemented) then succeeds (PauseTurn), logical turn two succeeds (Stop). RED: `recorded 0 cost_session events, want at least 1` (the cost_turn count of 2 and CheckStream both already passed, confirming the retry fixture itself was correct BEFORE the emission existed).
- [x] 3.4 GREEN: `cost_session(Estimate)` emitted before both `continue`s in `Run`'s switch (the ToolCalls/PauseTurn arm and the steered-message arm); `cost_session(Final)` emitted immediately before the success run-close. `failRun`/`windDownRun` both widened with an unexported `total costAccumulator` parameter and their own best-effort Final emission (front-loaded from Phase 4's tasks 4.1/4.2 since the widened signature had to compile against ALL call sites at once — see Phase 4 notes).
- [x] 3.5 Bite S-CST-021: `costAccumulator.add` mutated from `+=` to `=` (overwrite). Re-ran 3.3: FAILED — every figure read back as turn-two's value only (e.g. `InputTokens() = (19, true), want (719, true)`), proving turn-one's contribution was silently dropped. RED recorded, reverted; `git diff --stat` confirmed clean revert (diff against last commit shows only the legitimate Phase-3 additions).
- [x] 3.6 `TestHarness_CostSession_EstimateThenFinal` (S-CST-009) — PASSED immediately (GREEN already present from 3.4): ≥1 Estimate strictly between the two turn brackets, terminal Final equals cumulative, no estimate exceeds the final on any figure.
- [x] 3.7 `TestHarness_SingleTurnRun_FinalOnly` (S-CST-010) — PASSED immediately: sole `cost_session` is Final, none is Estimate.
- [x] 3.8 Bite S-CST-022: the success-close Final emission's label mutated to `CostLabelEstimate` (scratch). Re-ran 3.6 and 3.7: BOTH FAILED (`cost_session immediately before run_end has label estimate, want Final`; `sole cost_session label = estimate, want Final`). RED recorded, reverted.
- [x] 3.9 Label-axis assertion (S-CST-004) extended over 3.3's retry-bearing stream inline (loop at the end of `TestHarness_CostSession_CumulativeEqualsEmittedCostTurns`): every `cost_turn` observed reads Final.
- [x] 3.10 S-RUN-022 structural walk (`assertCostEventsRideExistingBracketsAndLane`) added and run on 3.3's same stream: exactly one run_start/run_end pair; every `cost_turn` strictly inside its own turn bracket; every `cost_session` inside the run bracket and outside every turn bracket; sequence 1..N contiguous with cost events included; every event carries the run's identity; each `cost_turn` exactly once.

**Correction to design.md's own citation**: `windDownRun` has **3** call sites (`:461` — the iteration-boundary cancellation check before any Turn attempt in a logical turn; plus the two inner-loop sites design.md did cite), not the "2 call sites (`:540`, `:566`)" tasks.md's task 4.2 states. All three were widened — Go's compiler enforces this (the widened signature would not build otherwise), so the undercount had no chance to ship silently.

## Phase 4 — Non-happy run closes (commit `bb167957`)

- [x] 4.1 `failRun` widened with `total costAccumulator` (5 call sites, not 1 as tasks.md's task 4.1 states — see correction above applied equally here: `:463` after `hist.Append(prompt)`, `:494` after a mid-run steering append, `:633` the R-RUN-011 main failure path, `:637` after `hist.CloseTurn()` fails, `:650` after a steered-message append fails). `cost_session(Final)` emitted best-effort immediately before `NewRunEnd(runID, RunOutcomeFailed, failure)`, nested inside the same `if failure, ferr := ...; ferr == nil` block as a fully independent best-effort step (its own construction can fail without preventing the run-close, and vice versa).
- [x] 4.2 `windDownRun` widened with `total costAccumulator` (3 call sites — see correction above). `cost_session(Final)` emitted best-effort after `hist.CloseTurn()`/`SynthesizeOrphans()` and immediately before `NewRunEnd(runID, outcome, nil)`, preserving R-CAN-002's amended enumerated order.
- [x] 4.3 `TestHarness_CostSession_FinalOnFailedRun` (S-CST-012, S-RUN-104) — two subtests: "usage reported before the failure" (turn one succeeds, turn two fails non-retryably via a new `failAfterProvider` fixture — the mirror image of `preStreamFailingProvider`) and "no usage reported: fails on the first turn" (figures read absent, not zero). Both PASS immediately (implementation front-loaded into Phase 3).
- [x] 4.4 `TestHarness_CostSession_FinalOnInterruptedRun` (S-CST-013/S-CAN-012) — Gate-held second turn, `h.Interrupt()`. Asserts the full order (aborted turn-close with no cost_turn → cost_session(Final) → run_end(Interrupted, nil failure) → close), figures equal cumulative, AND a second `Run` on the same harness value (with a fresh provider swapped onto the same `h` value) reports only its own turn's figure (40 from run 1 never carries into run 2's 9). PASS immediately.
- [x] 4.5 `TestHarness_CostSession_FinalOnShutdownRun` (S-CST-013/S-CAN-013) — same shape with `h.Shutdown()`, `RunOutcomeShutdown`; a third `Run` invocation after the shutdown flag latches observes ZERO events (`errors.Is(err, agent.ErrPromptAfterShutdown)`, `len(drainSink(...)) == 0`). PASS immediately.

## Phase 5 — Scope fence & substrate verification (commit `449566a0`)

- [x] 5.1 `TestCost_ScopeFence` (S-CST-014) added: owns only the `git diff` half (`backend/agent/src/ai/` and `go.mod`/`go.sum` empty against `merge-base(HEAD, origin/main)` or `$AG16_BASE_REF`). The token-only reflection scan and every-kind-constructible guard are proven by `cost_events_test.go`'s and `event_registry_test.go`'s own UNEDITED tests continuing to pass in the same `make test` run — not duplicated.
- [x] 5.2 Byte-unchanged verification against `main@09bb30e1` — all 11 files empty diff: `cost_events_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `turn_events.go`, `failure.go`, `run_events.go`, `history.go`, `doc.go`, `doc_contract_guard_test.go`.
- [x] 5.3 `git diff --name-only 09bb30e1 -- backend/agent/src/agent/` excluding `_test.go` = exactly `{cost_events.go, cost_usage.go, harness.go, loop.go}` — matches S-LSK-025 exactly.

## Phase 6 — Substrate filter widening (done incrementally in Phases 0/1+2/3, verified here)

- [x] 6.1 Both filters widened by exact filename suffix, each entry landed in the commit introducing the file it names: `/cost_events.go`, `/cost_usage.go`, `/cost_usage_test.go` (Phase 0); `/cost_turn_emission_test.go` (Phase 1+2); `/cost_session_test.go` (Phase 3). `/cost_events_test.go` and `/stream_check_test.go` deliberately absent from both.
- [x] 6.2 Both filters' complete entry sets extracted and diffed: 46 entries each, byte-identical (`diff` reports no differences). AG-16's 5 entries present in both; `cost_events_test.go`/`stream_check_test.go` confirmed absent from both (0 grep matches). No wildcard/prefix/directory pattern anywhere (every entry is `strings.HasSuffix(path, "/exact-name.go")`).

## Lint fix (bundled into Phase 5 commit)

`golangci-lint cache clean && make lint` surfaced 2 issues after Phase 0-4's work: `cost_usage.go`'s doc comment didn't start with "Package agent ..." (revive `package-comments` — fixed by adopting the same "Package agent is Layer 2..." form `cancellation.go`/`tool.go`/`scheduler.go` already use), and a De Morgan's-law simplification in `cost_session_test.go`'s S-CST-009 position check (staticcheck QF1001). Both fixed; `make lint` now reports `0 issues` repo-wide.

## Remaining (Phases 7-9, this batch continues)

- [ ] Phase 7 — Documentation: doc 0003 reconciliation note, status line, counter bump; checklist row 7.3 stays deliberately unticked (AG-18.1 not shipped).
- [ ] Phase 8 — OpenSpec promotion (6 delta merges + 1 new capability spec) + `git mv` archive.
- [ ] Phase 9 — Final gate: full `make test -race`, `make lint`, `gofmt -l`, `make build`, `make vuln-check`, tasks.md self-check.

## TDD Cycle Evidence

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 0.3-0.4 | `cost_usage_test.go` | Unit (internal) | N/A (new) | Written (compile fail) | Passed | 9 cases | None needed |
| 1.1-1.2 | `cost_turn_emission_test.go` | Integration (Turn/Harness-driven) | ✅ 258/258 (full pkg baseline) | Written | Passed | 6 companion scenarios | None needed |
| 1.4 | `cost_turn_emission_test.go` | Integration | — | Vacuously true pre-GREEN (no emission existed at all); non-vacuous once paired with 1.1's positive proof + bite 1.8's mutation | Passed | 5 sub-conditions | None needed |
| 1.8 (bite) | `cost_usage.go` (scratch) | — | — | ✅ Written, FAILED as predicted | Reverted | — | — |
| 3.3-3.4 | `cost_session_test.go` | Integration | ✅ (Phase 1+2 green) | Written | Passed | 3 companion scenarios (3.6, 3.7, 3.10) | None needed |
| 3.5 (bite) | `cost_usage.go` (scratch) | — | — | ✅ Written, FAILED as predicted | Reverted | — | — |
| 3.8 (bite) | `harness.go` (scratch) | — | — | ✅ Written, FAILED as predicted (both 3.6 and 3.7) | Reverted | — | — |
| 4.3-4.5 | `cost_session_test.go` | Integration | ✅ (Phase 3 green) | Written | Passed immediately (implementation front-loaded into Phase 3) | 2 (failed-with-usage / failed-without) + 2 (interrupted/shutdown) | None needed |
| 5.1 | `cost_turn_emission_test.go` | Integration (git-diff check) | ✅ | Written | Passed | N/A — single deterministic check | None needed |

### Test Summary
- **Total new test functions**: 16 top-level (`TestCostFromUsage_TableDriven`, `TestCostFromUsage_ZeroValuePayloadReportsEveryFigureAbsent`, `TestTurn_CostTurn_FiguresExactPerTurn`, `TestTurn_NoCompletion_CostTurnAllAbsent`, `TestTurn_AbortedTurn_NoCostTurn` [5 subtests], `TestTurn_CostTurn_LabelAlwaysFinal`, `TestTurn_CostTurn_AbsenceVsReportedZero`, `TestTurn_CostTurn_MixedRecordPartialPresence`, `TestTurn_Standalone_NoCostSession`, `TestCost_ScopeFence`, `TestHarness_CostSession_CumulativeEqualsEmittedCostTurns`, `TestHarness_CostSession_EstimateThenFinal`, `TestHarness_SingleTurnRun_FinalOnly`, `TestHarness_CostSession_FinalOnFailedRun` [2 subtests], `TestHarness_CostSession_FinalOnInterruptedRun`, `TestHarness_CostSession_FinalOnShutdownRun`)
- **Amended existing tests**: 3 (`TestTurn_WalkingSkeleton_EmitsContractEventOrder`, `TestTurn_ReasoningPassThroughByteExact`, `TestHarness_MidTurnSteer_EntersAtBoundaryBeforeNextProviderCall`) + 2 stale-comment corrections (`TestTurn_TwoSequentialTurnsShareNothing`'s `[1,9]`→`[1,10]`)
- **Bites**: 3 (S-CST-020, S-CST-021, S-CST-022), all RED-recorded then reverted, verified clean via `git diff --stat`
- **Layers used**: Unit (1 — the internal `costFromUsage` table test), Integration (15 — `Turn`/`Harness.Run`-driven, external `agent_test` package)
- **Pure functions created**: `costFromUsage`, `costAccumulator.add`, `costAccumulator.sessionEvent` (thin constructor wrapper)

## Known risks / discrepancies carried forward from design.md and tasks.md, corrected during apply

1. **`windDownRun` has 3 call sites, not design/tasks.md's cited 2.** The third is the iteration-boundary cancellation check (`harness.go`, before any `Turn` attempt in a logical turn). Widened at all three; Go's compiler made an undercount impossible to ship silently.
2. **The `loop_test.go:1152-1165` "S-LSK-009 tool dispatch" citation in design.md and in the `agent-loop-skeleton` delta's own S-LSK-009 scenario text is wrong.** That location is actually `TestTurn_ReasoningPassThroughByteExact` (S-LSK-005). The REAL S-LSK-008/009 tool-dispatch test lives in `loop_tool_dispatch_test.go` and uses kind-filtered counting (`ToolStart`/`ToolEndSuccess` counts), which is unaffected by AG-16 regardless. The implementation fix (insert `EventKindCostTurn` before `EventKindTurnEnd` in the closed sequence at the cited line range) is correct either way since the line range itself is accurate — only the scenario LABEL attached to it is wrong. Not corrected in the spec prose during this apply (out of scope for `sdd-apply`; flagged for `sdd-verify`/a future spec correction).
3. **`CostLabel`'s own type-level doc comment (`cost_events.go:85-89`) still reads the pre-AG-16 wire-level framing** ("distinguishes figures emitted before the stream's final usage update..."). This is deliberate: the bounded R-LSK-004 release for AG-16 names only "the doc comments at `cost_events.go:96-103`" (the two constant doc comments) as an allowed edit, "and nothing else." Restating the type doc too would exceed the recorded release scope. Flagged for a future milestone's own recorded release if it needs fixing.
4. **`errorProvider` precedent citation drift**: the `agent-retry-failover` spec text and design.md cite `loop_test.go:1408-1421` for the retry fixture; the actual fixture used (and the one that matches the described behavior) is `retry_policy_test.go`'s `preStreamFailingProvider`, at whatever its current line is. `loop_test.go`'s own `errorProvider` (currently `:1458-1480`) is a simpler always-fails fixture serving a different test. Neither citation was load-bearing for implementation — `preStreamFailingProvider` was identified and reused correctly by reading its own doc comment and call sites directly.
5. **Two pre-existing files I touched (`loop.go`, `loop_test.go`, `loop_hook_test.go`) were already `gofmt`-dirty on `main` before this change**, unrelated to AG-16 (confirmed via `git stash` + `gofmt -l` against the pre-edit tree). Since all three are files this milestone legitimately modifies, `gofmt -w` was applied to them specifically (never a repo-wide `make fmt`/`make all`); the resulting diffs include a few incidental pre-existing whitespace/doc-comment-list reformattings bundled in with the intentional AG-16 changes. Confirmed comment-only / whitespace-only, never behavioral, by diffing each file's non-AG-16 hunks individually.
6. **Not yet independently verified by a command in this progress snapshot**: `make build` and `make vuln-check` (scheduled for Phase 9). `go build ./...` has been run repeatedly throughout and always succeeded.
