# Tasks: AG-11 — Complete turn termination and typed failure reporting

> Scenario count: **4 charter → 12 spec + 5 bites = 17 total** across 9 requirements (`R-ATT-001..009`) plus three MODIFIED deltas (`R-LSK-001/004`, `R-APP-012`, `R-AEV-008`). Design decisions `D1..D8` are binding; this file does not re-derive or contradict them.

## Substrate Filter Closure (authoritative — closes R-ATT-009 / S-LSK-012 / S-APP-014)

`filterOutLoopFiles` (`loop_test.go`) and `filterOutLoopHookFiles` (`loop_hook_test.go`) MUST widen with exactly these five exact-filename suffixes, byte-in-sync between the two functions, no wildcard/prefix/directory pattern:

```
/turn_events.go
/failure.go
/invariant_pin_test.go
/turn_termination_test.go
/turn_failure_test.go
```

This is the concrete filename list `R-LSK-004` and `R-APP-012` delegate to `tasks.md`. Any file outside this set MUST still fail both guards.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ≈940 (design estimate; spec `NFR-ATT-005` forecasts 900–1500) |
| 400-line budget risk | High (session-extended review budget for this milestone: **1000 lines — HOLDS**, 940 ≤ 1000) |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | exception-ok (`size:exception` pre-authorized) |
| Chain strategy | size-exception (NOT chained) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | Chore: this tasks.md, filter closure notation | N/A | N/A | revert chore commit |
| 2 | `TurnOutcome` vocab (`turn_events.go`) + `PartialOutput()` (`failure.go`) + filter widening | `go test -run "TestTurnOutcome_|TestFailure_PartialOutput" -race ./src/agent/...` | N/A (unit) | revert `turn_events.go`/`failure.go`/filter diffs |
| 3 | Exhaustive dispatch + `finalize()` wiring + zero-normalization move (D2) | `go test -run "TestTurn_FinishReasonDispatch|TestTurn_NoCompletionPath" -race ./src/agent/...` | `agenttest` scripted provider | revert dispatch + `finalize()` diffs |
| 4 | Agent-level exhaustiveness pin (0..255 walk) | `go test -run TestTurn_ExhaustivenessPin -race ./src/agent/...` | N/A (unit) | revert `turn_termination_test.go` pin |
| 5 | Refusal/pause divergence (R-ATT-004) | `go test -run "TestTurn_RefusalPauseFinished|TestTurn_PauseReplaysVerbatim" -race ./src/agent/...` | `agenttest` scripted provider | revert refusal/pause scenarios |
| 6 | Fatal-path rewrite (D1/D6/D7) + failure-identity pin + partial content | `go test -run "TestTurn_FatalPath|TestTurn_PartialContentSurvives|TestTurn_ExactlyOneProviderCall|TestTurn_InternalErrorArm" -race ./src/agent/...` | `agenttest` `conformance_terminal.go` script | revert `loop.go` fatal branch + `turn_failure_test.go` |
| 7 | Cross-cut invariants: `S-LSK-011/012`, `S-APP-014`, `S-ATT-011`, `loop.go:265` byte-check | `cd backend/agent && make test` + `git diff origin/main -- backend/agent/src/agent/` | N/A | revert cross-cut assertions |
| 8 | Spec promotion (ADDED + 3 MODIFIED deltas → canonical `openspec/specs/`) | N/A (docs) | N/A | revert promotion commit |
| 9 | Docs: `openspec/AGENTS.md` NFR-TLS-003 note, milestone doc checkbox, openspec archive | N/A (docs) | N/A | revert docs/archive commit |
| 10 | Final gates: `make test`/`lint`/`build`/`vuln-check` | `cd backend/agent && make test && make lint && make build && make vuln-check` | N/A | N/A (verification only, no code) |

## Phase 1: Foundation — substrate vocabulary (Work Unit 2)

- [ ] 1.1 RED — `turn_termination_test.go` (create, `package agent_test`): `TestTurnOutcome_DistinctMemberPerFinishReason` (S-ATT-001) walks `1..turnOutcomeLimit-1` via `NewTurnEnd`, asserts distinct non-placeholder `String()` forms. Expect FAIL: compile error / short vocabulary (only 2 members exist today).
- [ ] 1.2 RED — same file: `TestTurnOutcome_ZeroAndFailureRuleUnchanged` (S-ATT-002) — zero value rejected, failure-iff-Aborted rule holds. Expect FAIL until 1.3 exists (vacuous pass risk without 1.1's member growth — assert against extended vocab).
- [ ] 1.3 GREEN — `turn_events.go`: append six `TurnOutcome` members after `TurnOutcomeAborted` per D4 (`LengthLimited, ToolCalls, ContentFiltered, Refused, Paused, Unknown`), six `String()` cases; `turnOutcomeLimit` moves last. `validate()`/`NewTurnEnd` untouched (design-confirmed zero-change).
- [ ] 1.4 RED — `invariant_pin_test.go` (modify): `TestFailure_PartialOutput_ReachableAsTypedValue` (S-ATT-008 / S-AEV-074) — true/false/nil-safe cases via `agent.NewFailure`. Expect FAIL: compile error, `PartialOutput()` undefined.
- [ ] 1.5 GREEN — `failure.go`: add nil-safe `PartialOutput() bool` after `Retryable()`, delegates to `f.wrapped.PartialOutput()`, mirrors `Category()`/`Delivery()`/`Retryable()` shape exactly.
- [ ] 1.6 Widen filters — `loop_test.go` `filterOutLoopFiles` and `loop_hook_test.go` `filterOutLoopHookFiles`: add the five exact suffixes from the Substrate Filter Closure section, byte-in-sync. Land in the **same commit** as 1.3–1.5 so `make test` is never red for the unrelated substrate-guard reason.
- [ ] 1.7 GREEN confirm — `go test -run "TestTurnOutcome_|TestFailure_PartialOutput" -race ./src/agent/...` passes; `TestTurn_SubstrateUntouched` and `TestTurn_PreRequestHook_SubstrateUntouched` pass.

## Phase 2: Exhaustive dispatch (Work Unit 3)

- [ ] 2.1 RED — `turn_termination_test.go`: `TestTurn_FinishReasonDispatch_EachProducesDistinctOutcome` (S-ATT-003) — 7 scripted turns (one per `ai.FinishReason` member), assert each `turn_end.Outcome()` matches D4's table and all 7 are pairwise distinct. Expect FAIL: `finalize()` still hardcodes `TurnOutcomeFinished`.
- [ ] 2.2 GREEN — `loop.go`: add `outcomeForFinish(ai.FinishReason) TurnOutcome` (exhaustive switch, defensive `default: return 0`, per Interfaces/Contracts). `finalize()` (`loop.go:613`) consumes it instead of the hardcoded value.
- [ ] 2.3 RED bite S-TTB-001 — temporarily alias/remove one dispatch-table entry in the test's expectation map (not production code), confirm `TestTurn_FinishReasonDispatch_EachProducesDistinctOutcome` fails with a duplicate/missing-outcome message; capture the failing output in `apply-progress.md`; revert. RED-recorded BEFORE 2.1 is GREEN.
- [ ] 2.4 RED — `TestTurn_NoCompletionPath` (S-ATT-012) — provider stream closes without `ai.Completion`; assert `turn_end` emitted, member outcome, `finish == ai.FinishReasonStop`, no `NewTurnEnd` violation. Expect FAIL: `finalize()` at `loop.go:286` currently sees zero finish (per D2's confirmed no-completion gap) before the normalization at `:288-290`.
- [ ] 2.5 GREEN (D2) — `loop.go`: move `if turn.finish == 0 { turn.finish = ai.FinishReasonStop }` to run **ahead of** the `finalize()` call at `:286`; delete the post-hoc block at `:288-290`. Covers BOTH `finalize()` call sites: `loop.go:257` (completion path, already non-zero by `ai.NewCompletion` validation) and `loop.go:286` (no-completion path, now normalized before the call).
- [ ] 2.6 GREEN confirm — `go test -run "TestTurn_FinishReasonDispatch|TestTurn_NoCompletionPath" -race ./src/agent/...` passes.

## Phase 3: Agent-level exhaustiveness pin (Work Unit 4)

- [ ] 3.1 RED — `turn_termination_test.go`: `TestTurn_ExhaustivenessPin` (S-ATT-004) — walk `ai.FinishReason(0..255)`; membership by `.Validate()` (NOT the unexported `finishReasonLimit` — `outcomeForFinish` is also unexported, so this is a membership-walk + behavioral pin per `finish_reason_test.go:277-320`'s idiom); every validating candidate must appear in a hand-written `dispatchVocabulary` map and counts must match; for each of the 7, script a turn and assert the dispatched `turn_end.Outcome()` equals the table. Expect FAIL: no such test exists yet.
- [ ] 3.2 GREEN — pin passes once Phase 2's `outcomeForFinish` is complete (no production change needed if 2.2 is total).
- [ ] 3.3 RED bite S-TTB-002 — force a probe value to validate as an `ai.FinishReason` while absent from `dispatchVocabulary`; confirm the pin fails and names the unhandled value; capture output; revert. RED-recorded BEFORE 3.1 is GREEN.
- [ ] 3.4 GREEN confirm — `go test -run TestTurn_ExhaustivenessPin -race ./src/agent/...` passes.

## Phase 4: Refusal/pause divergence (Work Unit 5)

- [ ] 4.1 RED — `TestTurn_RefusalPauseFinished` (S-ATT-005) — three scripted turns (`Refusal`, `PauseTurn`, `Stop`); assert pairwise-different `turn_end` outcomes with no `ai.FinishReason` read to distinguish them. Expect FAIL until Phase 2 dispatch names `Refused`/`Paused` distinctly (should already be GREEN from Phase 2/D4 — this scenario adds the "no `ai.FinishReason` inspection" behavioral proof).
- [ ] 4.2 RED — `TestTurn_PauseReplaysVerbatim` (S-ATT-006) — interleaved reasoning/text deltas, non-empty reasoning round-trip token, `FinishReasonPauseTurn` completion; assert byte-identical `msg`/deltas/token and `Paused` outcome (not `Refused`/`Finished`). Expect FAIL: verify no re-serialization defect exists in the current reconstruction path for this case.
- [ ] 4.3 GREEN — if 4.1/4.2 fail only on assertion wiring (not missing behavior), adjust test scripting; no additional production code expected beyond Phase 2/6's dispatch and reconstruction (Phase 6 lands `reconstructMessage()`; if 4.2 needs it early, pull D6's extraction forward into this unit).
- [ ] 4.4 RED bite S-TTB-003 — alias the paused outcome onto refused (or `Finished`) in the test's expectation, confirm `TestTurn_RefusalPauseFinished` reports the collapse; capture output; revert. RED-recorded BEFORE 4.1 is GREEN.
- [ ] 4.5 GREEN confirm — `go test -run "TestTurn_RefusalPauseFinished|TestTurn_PauseReplaysVerbatim" -race ./src/agent/...` passes.

## Phase 5: Fatal-path rewrite and typed failure (Work Unit 6)

- [ ] 5.1 RED — `turn_failure_test.go` (create, `package agent_test`): `TestTurn_FatalPath_EmitsTypedBrackets` (S-ATT-007) — provider scripted to message-start/delta/end then terminal `ai.ErrorEvent` carrying `*ai.Failure`; assert consumer observes `turn_end(Aborted, non-nil *Failure)` → `run_end(Failed, *Failure)` → channel close, in order; `Turn` returns non-nil error. Expect FAIL: `loop.go:270-276` today drains, closes, returns nothing emitted.
- [ ] 5.2 GREEN (D1/D6/D7) — `loop.go`: rewrite the fatal branch — type-assert `turn.fatal` to `*ai.Failure`; on match, wrap via `NewFailure`, emit `turn_end(Aborted, failure)` then `run_end(Failed, failure)` on `sink` before `closeSink`, return `reconstructMessage()` result. On the internal-error arm (plain Go error), keep byte-identical behavior: drain, `closeSink`, `return ai.Message{}, 0, turn.fatal` — no emission. Extract `finalize()`'s reconstruction body (`loop.go:627-676`) into `reconstructMessage()`, called from both `finalize()` and the fatal path (D6).
- [ ] 5.3 **Failure-identity pin** (orchestrator carry-forward item 3, pins D1's single-construction intent) — in the same test, assert `turnEnd.Failure()` and `runEnd.Failure()` are the **identical `*agent.Failure` pointer** (`==` comparison, not `reflect.DeepEqual` value equality). This proves one `NewFailure` call feeds both emissions, not two independent wraps.
- [ ] 5.4 RED bite S-TTB-004 — temporarily restore the fatal branch to its pre-AG-11 shape (drain/close/return only); confirm the consumer observes close with neither event and the scenario fails; capture output; revert. RED-recorded BEFORE 5.1 is GREEN.
- [ ] 5.5 RED — `TestTurn_PartialContentSurvives` (S-ATT-009) — text content then terminal mid-stream failure; assert returned `msg` carries content byte-for-byte (not `ai.Message{}`), deltas reconstruct to the same message via the AG-05.3 helper, `*Failure.PartialOutput() == true`. Expect FAIL until 5.2 lands.
- [ ] 5.6 RED bite S-TTB-005 — temporarily force the fatal branch to return `ai.Message{}`; confirm the scenario fails on the empty-message assertion; capture output; revert. RED-recorded BEFORE 5.5 is GREEN.
- [ ] 5.7 RED — `TestTurn_ExactlyOneProviderCall` (S-ATT-010, D7) — fake provider scripted with a terminal mid-stream failure and a second available script; assert `len(provider.Requests()) == 1` on turn completion, including a turn failing before any content. Expect FAIL only if a stray second call exists (design states none does — treat as a regression pin, may be GREEN immediately).
- [ ] 5.8 RED — `TestTurn_InternalErrorArm_EmitsNothing` (D1 pin) — one of the plain-Go-error construction sites (`loop.go:454,467,477,494,507,524,573`); assert no `turn_end`/`run_end` observed, matching pre-AG-11 behavior byte-for-byte. Proves D1's type-fork, not just the provider-`*ai.Failure` arm.
- [ ] 5.9 RED — `TestTurn_TypedFailureFullyInspectable` (S-AEV-075) — partial content then mid-stream failure; assert `turn_end(Aborted)`/`run_end(Failed)` each carry a non-nil `*Failure` whose `Category()`/`Delivery()`/`Retryable()`/`PartialOutput()` are all inspectable; guard stays at 25 kinds; no message-string assertion.
- [ ] 5.10 GREEN confirm — `go test -run "TestTurn_FatalPath|TestTurn_PartialContentSurvives|TestTurn_ExactlyOneProviderCall|TestTurn_InternalErrorArm|TestTurn_TypedFailureFullyInspectable" -race ./src/agent/...` passes.

## Phase 6: Cross-cut invariants (Work Unit 7)

- [ ] 6.1 `TestTurn_SignatureUnchanged` (S-LSK-011) — assert `Turn`'s exported signature is unchanged (doc-contract guard); mid-stream-fatal returned `msg` carries content, `err` non-nil.
- [ ] 6.2 `loop.go:265` byte-unchanged check — `git diff origin/main -- backend/agent/src/agent/loop.go` and confirm the `_ = sched.Schedule(...)` line is untouched (diff context only, no `+`/`-` on that exact line).
- [ ] 6.3 Merge-base diff scenario (S-LSK-012 / S-ATT-011 / S-APP-014) — `git diff <merge-base> -- backend/agent/src/agent/` and `go.mod`/`go.sum`: only pre-existing non-test files differing are `turn_events.go`, `failure.go`, `loop.go`; every other file named forbidden by `R-LSK-004`/`R-APP-012` byte-unchanged; `go.mod`/`go.sum` diff empty; `NewTurnEnd` signature and `TurnEnd.validate`'s failure-iff-aborted rule unchanged; both filters carry the identical five-suffix set from the Substrate Filter Closure section; every-kind-constructible guard passes at 25 kinds.
- [ ] 6.4 `failure.go` scoped diff (S-APP-014) — confirm the ONLY change in `failure.go` is the `PartialOutput()` addition; `NewFailure`, `Category`, `Delivery`, `Retryable`, `Unwrap` byte-unchanged.
- [ ] 6.5 `cd backend/agent && make test` (whole module, `-race`) green.

## Phase 7: Spec promotion (Work Unit 8)

- [ ] 7.1 Create `openspec/specs/agent-turn-termination/spec.md` from `openspec/changes/cachicamas-agent-turn-termination/specs/agent-turn-termination/spec.md` (ADDED delta) — strip the change-scoped promotion-note header line, keep all 9 requirements / 12 scenarios / 5 bites verbatim.
- [ ] 7.2 Apply the `agent-loop-skeleton` MODIFIED delta: replace `R-LSK-001` and `R-LSK-004` blocks in `openspec/specs/agent-loop-skeleton/spec.md` with the full MODIFIED text (including `S-LSK-011`, `S-LSK-012`); leave every other requirement in that file byte-unchanged.
- [ ] 7.3 Apply the `agent-permission-protocol` MODIFIED delta: replace `R-APP-012` in `openspec/specs/agent-permission-protocol/spec.md` with the full MODIFIED text (including `S-APP-014`); leave every other requirement byte-unchanged.
- [ ] 7.4 Apply the `agent-event-envelope` MODIFIED delta: replace `R-AEV-008` in `openspec/specs/agent-event-envelope/spec.md` with the full MODIFIED text (including `S-AEV-074`, `S-AEV-075`); leave every other requirement byte-unchanged.
- [ ] 7.5 Verification (guards spec risk 5 — the un-back-annotated merge): diff each promoted/amended canonical spec file against its change-folder delta; confirm ONLY the four named requirement blocks changed in each of the three MODIFIED files, and confirm the new file is byte-identical to the ADDED delta minus the header note. Record the diff evidence in `apply-progress.md`.

## Phase 8: Docs and archive (Work Unit 9)

- [ ] 8.1 `openspec/AGENTS.md` — append an AG-11 pointer under "Substrate preservation in `backend/agent` (NFR-TLS-003)": record that `failure.go` (one of the ten NFR-TLS-003 files) was modified for AG-11 only, for the structural reason in `R-LSK-004`/`R-APP-012`, following the AG-10-pointer convention already in that section.
- [ ] 8.2 `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` — flip the line-2167 acceptance checkbox (`Refusal, pause, and unknown finish reasons produce three distinct behaviors — closed by AG-11.1`) to `[x]`, sole owner AG-11.1. Do NOT flip line 2162 or 2168 — both list sibling milestones (AG-19.1/AG-20.2/AG-04.3; AG-10.1/AG-03.3) not yet confirmed complete; leave `[ ]` with no change.
- [ ] 8.3 openspec archive — move `openspec/changes/cachicamas-agent-turn-termination/` to `openspec/changes/archive/2026-08-16-cachicamas-agent-turn-termination/` (or the actual merge date) per the AG-09/AG-10 precedent naming.

## Phase 9: Final verification (Work Unit 10)

- [ ] 9.1 `cd backend/agent && make test` (race-gated) — all 17 scenarios + 5 bites recorded evidence, whole module green.
- [ ] 9.2 `cd backend/agent && make lint` after `golangci-lint cache clean` — 0 issues (per repo's cache-artifact pitfall).
- [ ] 9.3 `cd backend/agent && make build` — clean.
- [ ] 9.4 `cd backend/agent && make vuln-check` — record result; if the pre-existing stdlib advisories from AG-10 persist and trace exclusively outside `src/agent/**`, accept as WARNING per prior precedent, not a blocker.
- [ ] 9.5 Confirm `loop.go:265` byte-unchanged (cross-ref 6.2) and every-kind-constructible guard still 25 kinds.
- [ ] 9.6 Confirm scenario count `4 charter → 12 spec + 5 bites = 17 total` stated identically in `apply-progress.md` and `verify-report.md`.

## Risks

- **R-ATT-004 (4.1–4.3)**: design does not explicitly state whether `reconstructMessage()`'s extraction (D6, scheduled Phase 5) must land BEFORE `TestTurn_PauseReplaysVerbatim` (Phase 4) can go GREEN. If the pause-content path depends on the extracted helper, Phase 4 and Phase 5's D6 step may need reordering during apply; sequencing here is a best-effort ordering, not a hard design mandate — apply MUST NOT silently reorder D1/D2/D3 decisions themselves, only task sequencing within this file.
- **Filter-widening landmine**: every RED test added under `backend/agent/src/agent/` trips both substrate guards until its filter suffix lands. Tasks 1.6, and the filter-inclusive commits in Phases 2/3/5, MUST widen filters in the SAME commit as the file that needs them — never in a later, separate commit — or `make test` reports an unrelated failure alongside the intended RED signal.
- **`make vuln-check`**: AG-10 carried 5 pre-existing Go stdlib advisories at `go1.26.5` traced exclusively through `src/ai/openaicompat/**`, accepted as out-of-scope WARNING. If unresolved by AG-11's start, the same acceptance applies; do not attempt a toolchain upgrade under this milestone.
