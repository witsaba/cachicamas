# Tasks: AG-16 — Emit cost and usage events

> Change: `cachicamas-agent-cost-events`. Binding inputs: `proposal.md` (Decisions 1–5), `design.md` (DD1–DD4), seven delta specs under `specs/` (`agent-cost-events` NEW; `agent-protocol-events`, `agent-run-driver`, `agent-retry-failover`, `agent-loop-skeleton`, `agent-turn-termination`, `agent-cancellation-tree` MODIFIED). Strict TDD: every behavior task is RED-first (`cd backend/agent && make test`), RED output recorded in `apply-progress.md` before GREEN. Ships in **one PR**: implementation, doc 0003 update, and the OpenSpec archive together.

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (Go) | 980–1635 |
| Estimated changed lines (SDD markdown — 7 spec promotions already mostly authored, archive-report.md, apply-progress.md, this file) | 750–1150 |
| Estimated total | 1730–2785 |
| 400-line budget risk | High |
| Chained PRs recommended | No — `size:exception` pre-accepted at 1000 review lines |
| Suggested split (if ever needed) | U1 (inert capture) → U2 (cost_turn emission + blast radius) → U3 (cumulative + estimate/final) → U4 (non-happy closes + scope fence + filters) → U5 (doc 0003) → U6 (OpenSpec archive) |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

Judgment: not genuinely unreviewable as one PR — AG-15 shipped a comparable-size single-PR change (~3160 lines, 10 commits) successfully under the same exception. If a seam is ever needed, it falls exactly at the six work-unit boundaries below.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| U1 | Capture + conversion + presence discriminator (inert, no behavior change) | PR 1 (single-pr) | `go test -race -run TestCostFromUsage ./src/agent/...` | N/A — no observable behavior differs yet | `cost_usage.go`, `cost_usage_test.go`, `turnAccumulator.usage` field |
| U2 | `cost_turn` emission (Scenario 1) + blast-radius remediation | PR 1 | `go test -race -run 'TestTurn_CostTurn|TestTurn_NoCompletion|TestTurn_AbortedTurn'` | `make test` full suite (proves the amendment broke nothing else) | `finalize()` emission line, `cost_turn_emission_test.go`, the 3 amended existing test files |
| U3 | Cumulative + estimate/final protocol (Scenarios 2–3) | PR 1 | `go test -race -run TestHarness_CostSession` | `make test` | forwarder interception, 2 continue-arm + success-close emissions, `cost_session_test.go` (happy-path portion) |
| U4 | Non-happy closes + scope fence + filter widening | PR 1 | `go test -race -run 'TestHarness_CostSession_FinalOn|TestCost_ScopeFence'` | `make lint` + `make build` | `failRun`/`windDownRun` params+emissions, both substrate filters, `cost_events.go` bounded release |
| U5 | Doc 0003 update | PR 1 | N/A — documentation only | N/A — no runtime behavior | doc 0003 reconciliation note + status line + counter |
| U6 | OpenSpec promotion + archive | PR 1 | N/A — spec promotion only | N/A — no runtime behavior | the six delta merges, new `agent-cost-events/spec.md`, `git mv` of the change dir |

## Exact new/modified files

| File | Action | Hosts |
|---|---|---|
| `backend/agent/src/agent/cost_usage.go` | Create | `costPresence`, `costFromUsage`, `newCostTurnFromUsage`, `newCostSessionFromTotals`, `costAccumulator` |
| `backend/agent/src/agent/cost_usage_test.go` | Create | `TestCostFromUsage_TableDriven` (S-CST-007) |
| `backend/agent/src/agent/cost_turn_emission_test.go` | Create | S-CST-001…006, 011, 020(bite) |
| `backend/agent/src/agent/cost_session_test.go` | Create | S-CST-008…013, 021(bite), 022(bite) |
| `backend/agent/src/agent/cost_events.go` | Modify (**bounded R-LSK-004 release**) | `presence costPresence` field + 10 accessors on `CostTurn`/`CostSession`; doc comments `:96-103`. `CostFigures` (`:137-157`) and `cost_events_test.go` byte-unchanged |
| `backend/agent/src/agent/loop.go` | Modify | `usage ai.Usage` on `turnAccumulator` (`:768-810`); capture at `:923-927`; emission first-in-`finalize()` (`:1001`) |
| `backend/agent/src/agent/harness.go` | Modify | forwarder interception (`:507-509`); `var total costAccumulator` local; `Estimate` before `:606`/`:615`; `total` param + `Final` in `failRun` (`:335-342`) / `windDownRun` (`:315-327`); `Final` before `:621-624` |
| `backend/agent/src/agent/loop_test.go` (`filterOutLoopFiles`) | Modify | append exact-filename entries |
| `backend/agent/src/agent/loop_hook_test.go` (`filterOutLoopHookFiles`) | Modify | append exact-filename entries, byte-in-sync |
| `backend/agent/src/agent/harness_steering_test.go` | Modify (conditional) | insert `EventKindCostTurn` and, if the slice crosses `turn_end`, `EventKindCostSession` |

## Substrate filter drift — verified in this phase, DO NOT trust design.md's cited ranges

Verified against the shipped worktree: `filterOutLoopFiles` (`loop_test.go`) starts `:831` (matches design) but its **AG-15 tail** now runs to `:958` (`/failover_policy_test.go`, closing `}` `:959`) — design's cited `:871` is stale, superseded by AG-15's own merged widening. `filterOutLoopHookFiles` (`loop_hook_test.go`) starts `:907`; its tail was not re-measured here and MUST be re-grepped at apply time too.

- **Task 0.1** (below): re-grep both functions for their own AG-15 tail (`strings.HasSuffix(path, "/failover_policy_test.go")`) at apply time — never trust a cited line number, since this file shifts with every milestone that touches it.

## Phase 0 — Foundation: capture usage, conversion, presence discriminator (inert)

- [x] 0.1 Re-locate both filter functions' actual current tail by grep (not by line number); record the true line ranges in `apply-progress.md`.
- [x] 0.2 Add `usage ai.Usage` field to `turnAccumulator` (`loop.go:768-810`); populate it at the Completion case (`:923-927`) beside the existing `finish`/`finishOk` assignment. No emission yet — `make test` stays green, file-diff-only.
- [x] 0.3 RED: `cost_usage_test.go` — `TestCostFromUsage_TableDriven` (S-CST-007, R-CST-003) referencing not-yet-defined `costFromUsage`: empty `ai.Usage`, fully reported, all-zeros, five mixed single-figure cases, one pairwise-distinct case. Record RED (compile failure).
- [x] 0.4 GREEN: create `cost_usage.go`; add unexported `costPresence{input, output, cacheRead, cacheWrite, reasoning bool}` and total, pure `costFromUsage(u ai.Usage) (CostFigures, costPresence)` mirroring `ai.TokenCount.Count()`. `ai.Tokens(0)` maps to a reported nought, never absence.
- [x] 0.5 Add `presence costPresence` beside `figures CostFigures` on `CostTurn`/`CostSession` in `cost_events.go`; ten paired accessors (`InputTokens() (uint64, bool)` × 5 figures × 2 types). Existing `NewCostTurn`/`NewCostSession` set presence **all-reported** internally (their exact signatures unchanged; `cost_events_test.go` untouched). Restate the doc comments at `:96-103` for the run-scoped label axis (R-APE-005/DD4). `CostFigures` (`:137-157`) stays byte-unchanged.
- [x] 0.6 Add package-private siblings to `cost_usage.go`: `newCostTurnFromUsage(run RunID, turn TurnID, label CostLabel, u ai.Usage) (Event, error)` and `newCostSessionFromTotals(run RunID, label CostLabel, f CostFigures, p costPresence) (Event, error)` (AG-14 `typedCancellationFailure` sibling precedent) — construct the payload literal directly with real presence, bypassing the all-reported constructors.
- [x] 0.7 `make test` green; confirm via `git diff` that no call site is wired yet and no test assertion changed (pure addition).

## Phase 1 — `cost_turn` emission (Scenario 1: R-CST-001, R-CST-002, R-CST-003)

- [x] 1.1 RED: `cost_turn_emission_test.go` — `TestTurn_CostTurn_FiguresExactPerTurn` (S-CST-001): multi-turn run, distinct scripted `ai.Usage` per turn including cache-read/write/reasoning; assert `cost_turn` after content events and before `turn_end`, figures equal the scripted usage figure-for-figure, label `Final`, `CheckStream` accepts unmodified. Record RED (no emission today).
- [x] 1.2 GREEN: in `finalize()` (`loop.go:1001-1025`), first emission: `if costTurn, cerr := newCostTurnFromUsage(t.runID, t.turnID, CostLabelFinal, t.usage); cerr == nil { emitStamped(t.sink, t.stamper, costTurn) }`.
- [x] 1.3 `TestTurn_NoCompletion_CostTurnAllAbsent` (S-CST-002): provider closes without `Completion` — assert `cost_turn` still emitted, all five figures absent, no error.
- [x] 1.4 `TestTurn_AbortedTurn_NoCostTurn` (S-CST-003): mid-stream fatal, each of the three pre-stream aborts, and cancellation mid-turn — assert none emit `cost_turn`; each bracket closes exactly as its owning requirement already requires; `CheckStream` accepts each unmodified.
- [x] 1.5 `TestTurn_CostTurn_LabelAlwaysFinal` (S-CST-004): every `cost_turn` observed by 1.1/1.3 (extended over Phase 3's retry-bearing run at 3.3) reads `Final`; none reads `Estimate`.
- [x] 1.6 `TestTurn_CostTurn_AbsenceVsReportedZero` (S-CST-005): `ai.Usage{}` vs `ai.Usage{Input: ai.Tokens(0)}` — input reads `(0, absent)` vs `(0, reported)`; assertion reads the discriminator only, never the count alone.
- [x] 1.7 `TestTurn_CostTurn_MixedRecordPartialPresence` (S-CST-006): `ai.Usage{Input: ai.Tokens(100)}` → input `(100, reported)`, the other four absent.
- [x] 1.8 Bite **S-CST-020** (RED-first, then revert): collapse `costPresence` to one whole-record bool (scratch edit); re-run 1.7; confirm it **FAILS** reporting the four unreported figures as present zeros. Record RED, revert.
- [x] 1.9 `TestTurn_Standalone_NoCostSession` (S-CST-011): standalone `Turn` (nil continuation, zero-value `TurnOptions`) with a completing script — carries `cost_turn` before `turn_end`, no `cost_session` of either label on the sink.
- [x] 1.10 Confirm `Turn`'s exported signature unchanged by reading the shipped signature against `agent-turn-termination/spec.md:113` / `NFR-ATT-004` — no test edit required.

## Phase 2 — Existing-test blast-radius remediation (enumerated amendments, `NFR-CST-005`)

- [x] 2.1 Discovery: grep `backend/agent/src/agent/*_test.go` for `len(got)`, `wantKinds`, `wantOrder`, or any closed-sequence/length-equality assertion over an event-kind slice, **before** editing anything. Record the full site list in `apply-progress.md` — confirming no site beyond the three named below is affected.
- [x] 2.2 Amend `loop_test.go`'s `S-LSK-001` sequence (closed `wantKinds`, `len(got) != len(wantKinds)`, ~`:350-361`): insert `EventKindCostTurn` immediately before `turn_end`. Tied to the `agent-loop-skeleton` delta's `R-LSK-001` amendment; record as a signed-off amendment in `apply-progress.md`, not a quiet fix.
- [x] 2.3 Amend `loop_test.go`'s closed `wantOrder` (`S-LSK-009`, ~`:1152-1165`): insert `EventKindCostTurn` at its position after the rejoin-ordered tool events, before `turn_end`.
- [x] 2.4 Inspect `harness_steering_test.go`'s turn-one `wantKinds` (~`:103-126`, `S-LSK-024`): determine whether the slice crosses `turn_end` into `cost_session(Estimate)`'s window (design's flagged apply-time verification). Insert `EventKindCostTurn` always; insert `EventKindCostSession` only if the window includes it. Record the determination and its evidence.
- [x] 2.5 Re-run every site enumerated in 2.1 that was **not** amended; confirm each stays green, file-unchanged (positional-only assertions per design's table: `harness_test.go:297-305,:1162-1173`, `harness_pause_test.go:94`, `cancellation_interrupt_test.go:97-100,:212-215,:593-596`, `cancellation_shutdown_test.go:183-186`, `retry_policy_test.go:110-131`, `turn_failure_test.go:86-125`).
- [x] 2.6 `apply-progress.md` records every edited test file, its reason, and the delta/requirement it amends.

## Phase 3 — Cumulative accumulator + estimate/final protocol (Scenarios 2–3: R-CST-004, R-CST-005, R-RUN-003)

- [x] 3.1 Add `costAccumulator{figures CostFigures; presence costPresence}` to `cost_usage.go` with `add(c CostTurn)` (algebra: absent+absent=absent; absent+present n=present n; present m+present n=present m+n; present 0+absent=present 0 — reported nought survives) and `sessionEvent(run RunID, label CostLabel) (Event, error)`.
- [x] 3.2 Intercept the forwarder (`harness.go:507-509`): `for ev := range turnSink { if ct, ok := ev.CostTurn(); ok { total.add(ct) }; sink <- ev }`; declare `var total costAccumulator` as a **local** in `Run`'s stack frame, never a `Harness` field.
- [x] 3.3 RED: `cost_session_test.go` — `TestHarness_CostSession_CumulativeEqualsEmittedCostTurns` (S-CST-008): multi-turn run, one logical turn retried via the `errorProvider` wrapper precedent (`loop_test.go:1408-1421`, per the fixture constraint — `agenttest` cannot script an arbitrary retryable pre-stream failure), distinct scripted usage per completing attempt; assert cumulative equals the sum over observed `cost_turn` events on the **same recorded stream** — equality against observed events, never a literal; retried turn's succeeding attempt counted; `CheckStream` accepts unmodified.
- [x] 3.4 GREEN: emit `cost_session(Estimate)` before each `continue` at `harness.go:606` and `:615`; emit `cost_session(Final)` before the success run-close at `:621-624`; both via `total.sessionEvent(...)`.
- [x] 3.5 Bite **S-CST-021** (RED-first, then revert): make `costAccumulator.add` skip a retried turn's `cost_turn` (scratch edit); re-run 3.3; confirm **FAIL** (cumulative below observed sum). Record RED, revert.
- [x] 3.6 `TestHarness_CostSession_EstimateThenFinal` (S-CST-009): multi-turn continued run — ≥1 `Estimate` between turn brackets, last `cost_session` is `Final` immediately before run-close, figures equal cumulative, no estimate exceeds the final on any figure.
- [x] 3.7 `TestHarness_SingleTurnRun_FinalOnly` (S-CST-010): single-turn success run — sole `cost_session` is `Final`, no `Estimate` anywhere.
- [x] 3.8 Bite **S-CST-022** (RED-first, then revert): label the run-terminal `cost_session` `Estimate` (scratch edit); re-run 3.6 and 3.7; confirm both **FAIL**. Record RED, revert.
- [x] 3.9 Extend 1.5's label-axis assertion (S-CST-004) over 3.3's retry-bearing stream — every `cost_turn` `Final`, none `Estimate`.
- [x] 3.10 `S-RUN-022` structural walk on 3.3's recorded stream: every `cost_turn` strictly between its own `turn_start`/`turn_end`; every `cost_session` inside the run bracket and outside every turn bracket; lane sequence 1..N with no gap/repeat/restart including cost events; each `cost_turn` appears exactly once.

## Phase 4 — Non-happy run closes (R-CST-006, R-RUN-011, R-CAN-002, R-CAN-005)

- [x] 4.1 Widen `failRun` (`harness.go:335-342`) with an unexported `total costAccumulator` parameter (1 call site, `:595`); emit `cost_session(Final)` best-effort immediately before `NewRunEnd(runID, RunOutcomeFailed, failure)`.
- [x] 4.2 Widen `windDownRun` (`harness.go:315-327`) with the same parameter (2 call sites, `:540`, `:566`); emit `cost_session(Final)` best-effort after `hist.CloseTurn()` and immediately before `NewRunEnd(runID, outcome, nil)` — preserving `R-CAN-002`'s amended enumerated order: synthesize orphans → close turn → **emit `cost_session(Final)`** → emit run-close → return.
- [x] 4.3 RED→GREEN: `TestHarness_CostSession_FinalOnFailedRun` (S-CST-012, S-RUN-104): first turn completes with scripted usage, second fails into `R-RUN-011`'s path — assert `cost_session(Final)` immediately before run-close, figures equal cumulative over observed `cost_turn` events, failing turn contributed none, run-close still carries the failed outcome + true typed failure (`R-RTY-012`) as last event, no transcript append, `Run` returns the same error, `CheckStream` unmodified. Also: a run failing on its first turn before any usage reports all-absent, not zero.
- [x] 4.4 RED→GREEN: `TestHarness_CostSession_FinalOnInterruptedRun` (S-CST-013 / S-CAN-013): run interrupted after ≥1 turn reported usage — order: aborted turn-close (no `cost_turn`) → `cost_session(Final)` → run-close(`Interrupted`, nil failure, last) → sink close; figures = cumulative over observed events; transcript untouched by the emission; a second `Run` on the same harness value starts its own figure from nothing (`R-CST-004` run-scope). **ID correction (apply-time): the design/delta text calls this "S-CAN-012", which collides with R-CAN-006's pre-existing bite of the same ID in the shipped `agent-cancellation-tree` spec. Renumbered to S-CAN-013 during promotion (task 8.1) — see apply-progress.md.**
- [x] 4.5 RED→GREEN: `TestHarness_CostSession_FinalOnShutdownRun` (S-CST-013 / S-CAN-014): shutdown variant of 4.4 — same shape, `RunOutcomeShutdown`; a `Run` invoked after the shutdown flag latches observes **no event whatsoever**, cost events included. **ID correction (apply-time): the design/delta text calls this "S-CAN-013", renumbered to S-CAN-014 to stay sequential after the S-CAN-012→013 correction above.**

## Phase 5 — Scope fence & substrate verification (R-CST-007, NFR-CST-004)

- [x] 5.1 `TestCost_ScopeFence` (S-CST-014): re-run `S-APE-083`'s forbidden-substring scan and reflection walk (unedited) confirming pass; re-run the every-kind-constructible guard confirming it passes at its committed kind count; `git diff` over `backend/agent/src/ai/` and `go.mod`/`go.sum` is empty.
- [x] 5.2 Byte-unchanged verification: diff `cost_events_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `turn_events.go`, `failure.go`, `run_events.go`, `history.go`, `doc.go`, `doc_contract_guard_test.go` against `main@09bb30e1`; all empty. Record in `apply-progress.md`.
- [x] 5.3 `git diff main -- backend/agent/src/agent/` non-test files (S-LSK-025): confirm the set is exactly `{cost_events.go, cost_usage.go, harness.go, loop.go}`.

## Phase 6 — Substrate filter widening (R-LSK-004, S-LSK-026)

- [x] 6.1 Widen `filterOutLoopFiles` and `filterOutLoopHookFiles` at their re-grepped tail (task 0.1) — append, byte-in-sync: `/cost_events.go` (released), `/cost_usage.go`, `/cost_usage_test.go`, `/cost_turn_emission_test.go`, `/cost_session_test.go`. Deliberately do **not** add `/cost_events_test.go` or `/stream_check_test.go`. Land each entry in the same commit as the file it names first appears in.
- [x] 6.2 Confirm both filters carry an identical entry set (diff the two suffix lists); no wildcard, prefix, or directory pattern.

## Phase 7 — Documentation (part of the deliverable)

- [x] 7.1 `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` — insert the Decision 5 reconciliation note on the AG-16 charter (after `:1537`'s Out-of-scope line), following the AG-06 template verbatim in form (`:613`, "**Note — one v2 conflict, reconciled here (the AG-00 reconcile-or-flag duty, executed)**"): name both reconciliations — (1) "a retried attempt's tokens are real spend" reconciled to the sum-over-emitted-events reading, because `ai.Failure` carries no usage; (2) "usage arriving only on the final stream update" reconciled to the run-scoped estimate/final axis, because Layer 1 folds every wire-chunk update into one terminal `Completion` and the charter's own out-of-scope line already assigns `Estimate` to the running incremental figure.
- [x] 7.2 Same doc, status line (`:3`) — extend "Wave 3 opens with AG-12…AG-15" to include AG-16, following AG-13/14/15's narrative pattern; bump the milestone counter from **15 of 24** to **16 of 24**.
- [x] 7.3 Line `:2173` completion checklist — do **NOT** tick (`- [ ] Every turn emits a cost event; cumulative figures include retries and compaction; estimates are labelled — closed by AG-16.1, AG-18.1.`), since it also names AG-18.1 (compaction), which has not shipped. Grep-confirm this is the only checklist row naming AG-16.1; record why it stays unchecked.
- [x] 7.4 Confirm no `L2C-0N` contract row is owed (task 5.2 covers `doc.go`/`doc_contract_guard_test.go` byte-unchanged); confirm `backend/agent/README.md` needs no edit (grep found no cost-specific section to update).

## Phase 8 — OpenSpec promotion + archive (same PR)

- [x] 8.1 For each of the six delta specs (`agent-protocol-events`, `agent-run-driver`, `agent-retry-failover`, `agent-loop-skeleton`, `agent-turn-termination`, `agent-cancellation-tree`), apply the MODIFIED blocks in place into `openspec/specs/<capability>/spec.md`, replacing the cited requirement blocks verbatim (full-block preservation), including `agent-loop-skeleton`'s header range edit (`S-LSK-001`→`S-LSK-026`) and every non-requirements-table back-annotation. Diff each promoted file against its change-dir delta source (not against a placeholder) to confirm nothing was dropped or truncated.
- [x] 8.2 Create `openspec/specs/agent-cost-events/spec.md` — the new capability, verbatim from `openspec/changes/cachicamas-agent-cost-events/specs/agent-cost-events/spec.md` (adjust only the "Becomes … at archive" framing line, since it now **is** that file).
- [x] 8.3 `git mv openspec/changes/cachicamas-agent-cost-events/ openspec/changes/archive/2026-08-18-cachicamas-agent-cost-events/` (AG-15 archive-directory naming precedent: `2026-08-18-cachicamas-agent-retry-failover/`). Verify every moved file's content against its pre-move git blob (`git show HEAD:<old-path>` diffed against the moved file) — do not trust a "moved" report; diff byte-for-byte (known risk: a prior archive phase in this repo has truncated moved files into placeholders).
- [x] 8.4 Write `openspec/changes/archive/2026-08-18-cachicamas-agent-cost-events/archive-report.md` following AG-15's shape (what shipped, commits, capabilities promoted, verification at close, carried-forward follow-ups, state at close — Layer 2 stands at 16 of 24, AG-17 next).
- [x] 8.5 `openspec/changes/archive/WAVE-3-ARCHIVE.md` is **Layer 1's** Wave 3 tracker (AI-21…AI-23) — confirmed by reading it in this phase; it has no Layer 2 counterpart and AG-15's own archive did **not** touch it. **Do not edit it.** The precedent AG-15 actually followed is doc 0003's status-line update (Phase 7) plus its own self-contained `archive-report.md` (8.4) — that is the correct precedent, not the orchestrator's literal instruction to update `WAVE-3-ARCHIVE.md`.
- [x] 8.6 Update `apply-progress.md` with every RED/GREEN transition, every bite's RED evidence, every signed-off test amendment (Phase 2), and the byte-unchanged verification results (Phase 5), per `NFR-CST-005` and the spec's Evidence discipline section.

## Phase 9 — Final gate

- [x] 9.1 `cd backend/agent && make test` green under `-race`, full suite.
- [x] 9.2 `golangci-lint cache clean && make lint` — 0 issues. **Never `make fmt`/`make all`** (the fmt target rewrites committed files in place and manufactures failures — known repo lesson).
- [x] 9.3 `gofmt -l backend/agent` — empty output.
- [x] 9.4 `make build` clean; `make vuln-check` clean (not part of `make all`).
- [x] 9.5 `tasks.md` self-check: every `[ ]` closed to `[x]`; coverage table below fully mapped.

## Coverage table — every scenario has a task

| Scenario | Task(s) |
|---|---|
| S-CST-001 | 1.1–1.2 |
| S-CST-002 | 1.3 |
| S-CST-003 | 1.4 |
| S-CST-004 | 1.5, 3.9 |
| S-CST-005 | 1.6 |
| S-CST-006 | 1.7 |
| S-CST-007 | 0.3–0.4 |
| S-CST-008 | 3.2–3.4 |
| S-CST-009 | 3.6 |
| S-CST-010 | 3.7 |
| S-CST-011 | 1.9 |
| S-CST-012 | 4.1, 4.3 |
| S-CST-013 | 4.2, 4.4, 4.5 |
| S-CST-014 | 5.1 |
| S-CST-020 (bite) | 1.8 |
| S-CST-021 (bite) | 3.5 |
| S-CST-022 (bite) | 3.8 |
| S-APE-030/031/083/085/086 | 0.5, 1.6–1.7, 3.9 (payload surface exercised by the same tests) |
| S-LSK-001 (amended) | 2.2 |
| S-LSK-009 | 2.3 |
| S-LSK-011/013/014/021 (held) | 1.10, 2.5 |
| S-LSK-024/025/026 | 1.1–1.4, 5.3, 6.1–6.2 |
| S-RUN-020/021/022 | 3.10 |
| S-RUN-100/101/102/103 (held) | 2.5 |
| S-RUN-104 | 4.3 |
| S-CAN-002/005/011 (held) | 2.5 |
| S-CAN-013 (renumbered from the delta's "S-CAN-012" — collision fix) | 4.4 |
| S-CAN-014 (renumbered from the delta's "S-CAN-013") | 4.5 |
| Both substrate filters, exact-filename, byte-in-sync | 0.1, 6.1–6.2 |
| Doc 0003 note / status / counter / checklist | 7.1–7.4 |
| OpenSpec promotion + archive | 8.1–8.6 |

## Known risks carried forward

- `agent-loop-skeleton`'s delta has a `## MODIFIED Header` section outside the usual `MODIFIED Requirements` shape (the allocated-range edit) — archive (8.1) must not drop it, mirroring AG-15's own flagged risk for the same file.
- The substrate filter tail is measured, not cited, at apply time (task 0.1) — design.md's cited `loop_test.go:831-871` is already stale against the shipped AG-15-merged tree; trust nothing but a fresh grep.
- `openspec/changes/archive/WAVE-3-ARCHIVE.md` is unrelated to this milestone (Layer 1, not Layer 2) — task 8.5 records why it is deliberately left untouched rather than edited per the orchestrator's literal (but factually incorrect) instruction.
- `harness_steering_test.go`'s turn-one `wantKinds` crossing `turn_end` into `Estimate`'s window is undetermined until task 2.4 runs — the only open apply-time fact in this plan.
