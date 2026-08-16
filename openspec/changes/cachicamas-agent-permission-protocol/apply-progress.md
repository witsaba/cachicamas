# Apply Progress: AG-10 — Implement the permission protocol

**Change**: `cachicamas-agent-permission-protocol` · Layer 2 Wave 2, milestone 10/24
**Mode**: Strict TDD · **Store**: hybrid (Engram + OpenSpec filesystem)
**Session**: resumed apply (prior partial work: 4 commits + ~301 uncommitted lines, re-validated by orchestrator audit before this session — Engram observation #3038)

## Status

20/20 tasks complete (all `[x]` in `tasks.md`), with one explicit exception: task 5.4
(`make vuln-check` clean) is **not** clean, for reasons entirely outside AG-10's scope
(see Gates below). Nothing is left uncommitted. Ready for `sdd-verify`.

## Commits (this session, oldest first)

| SHA | Message |
|---|---|
| `87abd9f8` | chore(agent): AG-10 track SDD change artifacts |
| `9052381c` | feat(agent): AG-10.3 GREEN — sibling isolation + cancellation wind-down |
| `34d295eb` | feat(agent): AG-10 wake path + policy wiring (D-A, D-C, tasks 3.1/4.2) |
| `22297d84` | refactor(agent): AG-10 D-B — delete unreachable parked-set shutdown sweep |
| `aaa5c7b4` | feat(agent): AG-10.4 — Remember gate + remembered-suppression (task 3.4) |
| `1adf6350` | test(agent): AG-10 R-TLS-008 source guard (task 4.3) |
| `756c3e76` | fix(agent): AG-10 defect 6 — stop swallowing decision_made constructor errors |
| `e05e01cf` | fix(agent): AG-10 R-APP-002 ordering + park-test cancel-after-return bug (task 2.2, defects 8/9) |
| `ba1e8777` | test(agent): AG-10 re-point remembered-cardinality bite at a real Schedule run (task 2.4) |
| `52c8c9be` | refactor(agent): AG-10 table-driven t.Run skeleton for the four-outcome matrix (task 1.2) |
| `e18095e5` | docs(agent): AG-10 pointer + NFR-TLS-003 substrate list in openspec/AGENTS.md (task 1.3) |
| `2ec851c6` | chore(agent): AG-10 back-annotate tasks.md + record apply-progress |
| `29a2dcfd` | refactor(agent): AG-10 remove vestigial abortFailure no-op branch (task 5.2) |

Pre-session commits (already landed, re-validated, preserved byte-clean where the
orchestrator's audit marked them DONE): `75ecd566`, `5f03d600`, `20b481a8`, `6ec55de7`.

## Design Decisions Applied (orchestrator-resolved, implemented as directed)

- **D-A (wake path)**: `parkedSet` remains internally threaded through `Schedule`'s call
  goroutines unchanged; a *separate* mutex-guarded `Scheduler.parked` field publishes the
  same set for external reach. `WakeParked(callID)` closes the parked channel; on wake the
  gate recurses into `runPermissionGate`, re-entering `policy.Resolve` and processing the
  fresh verdict (design data flow "wake: re-evaluate verdict"). `ResolveStrayDecision`
  deleted; the stray-decision bite re-points at the real surface.
- **D-B (shutdown ordering)**: deleted the unreachable `parked.closeAll()` sweep,
  `errParkedSetShutdown`, and the dead `wakeErr != nil` branch — proven dead because
  `wg.Wait()` cannot return while any call is still parked (a parked goroutine defers
  `wg.Done()` until release), so nothing is ever left for a post-`wg.Wait()` sweep to
  unblock. `parkedSet`'s channel reverted `chan error` → `chan struct{}` once the
  wake-vs-shutdown distinction that motivated `chan error` no longer existed. An approval
  test (`TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline`, no
  context deadline at all) recorded PASS before and after the refactor.
- **D-C (policy injection)**: `TurnOptions.PermissionPolicy` field added; `loop.go`
  forwards it byte-exact to `Schedule` instead of the literal `nil` the AG-13-deferral
  comment described. Nil remains a permanent, legitimate bypass — existing nil-policy
  tests are untouched. A new loop-level test
  (`TestTurn_PermissionPolicy_WiredToSchedule`, `loop_tool_dispatch_test.go`) proves the
  field actually reaches `Schedule` rather than merely existing on the struct.

## TDD Cycle Evidence

| Task / Defect | Test | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|
| D-A/D-C wake path + policy wiring (tasks 3.1/4.2) | `TestPermission_WakeParked_ResumesAndCompletes`, `TestPermission_WakeParked_UnknownCallID_TypedRejection_NoTouch`, `TestPermission_StrayDecisionIsTypedError` (repointed), `TestTurn_PermissionPolicy_WiredToSchedule` | ✅ compile-error RED, proven by reverting `tool.go`/`scheduler.go`/`loop.go` and running `go vet` (`unknown field PermissionPolicy in struct literal`) | ✅ 4/4 PASS, 1.52s | ✅ 2 wake scenarios (success path + stray-rejection-with-no-touch) | ➖ none needed |
| D-B shutdown cleanup | `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` | N/A — approval test (strict-tdd.md "Approval Testing for refactoring"); recorded PASS on the pre-refactor baseline | ✅ PASS unchanged after the refactor | ➖ single scenario is the whole property (no-deadline context) | ✅ deleted ~70 lines of dead code (`closeAll`, `errParkedSetShutdown`, dead branch) |
| AG-10.4 Remember (task 3.4) | `TestPermission_AllowAlways_Remember_Branches` (2 subtests), `TestPermission_RememberedSuppressesSubsequentAsk` | ✅ assertion-failure RED: `policy.Remember invocation count = 0, want 1`; `policy.Resolve invocation count = 3, want 2` | ✅ 3/3 PASS, 1.51s | ✅ true/false Remember branches + suppressed/non-suppressed tool names | ➖ none needed |
| R-TLS-008 source guard (task 4.3) | `TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin` | N/A — integration test over already-implemented, already-tested behavior (no new production code); labeled honestly rather than forcing a synthetic RED | ✅ `-count=10` clean, 1.30s | ➖ single comprehensive scenario (5 calls, mixed lanes, all 4 outcomes) is the task's own definition | ✅ deleted genuinely-dead `parkedSet.parkedCount()` (unreachable from this package's external-test-only posture) |
| Defect 6 (swallowed constructor errors) | `TestPermission_ModifyInput_ConstructorFailure_DoesNotSilentlyProceed` | ✅ assertion-failure RED: `results[0].Outcome = success, want ExecutionFailure`; `modify_bad invocations = 1, want 0` | ✅ PASS | ➖ ModifyInput is the one policy-controlled trigger; AllowOnce/AllowAlways/Deny fixed uniformly but not independently RED-tested (see Risks) | ➖ none needed |
| R-APP-002 ordering + defects 8/9 (task 2.2) | `TestPermission_DeferEmitsBeforePark` (renamed), `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot`, `TestPermission_CancellationMidPark_PopulatesRejoinAndNoLeak` | ✅ `TestPermission_DeferEmitsBeforePark` RED: "WakeParked succeeded before anything read from sink" — proven against the pre-ack implementation | ✅ all three PASS, 0.30s / 0.00s / 0.00s (was 2.00s×3) | ✅ discovered + fixed a second race (ack proves delivery, not registration) across 4 call sites via `wakeParkedWithRetry`, caught by a `-count=8` full-suite repeat run | ✅ added the missing goroutine-leak assertion to the cancellation test |
| Cardinality bite re-pointed (task 2.4) | `TestPermission_RememberedCardinality_SecondEmissionRejected` | N/A — re-pointing an existing bite at real code, not new behavior | ✅ `-count=25` clean, deterministic forced-race barrier | ➖ single scenario | ✅ replaced hand-built stream with real `Schedule`-emitted events sharing one `LaneStamper` |
| Table-driven t.Run (task 1.2) | `TestPermission_FourOutcomes` (4 subtests) | N/A — approval-testing refactor (behavior unchanged) | ✅ 4/4 subtests PASS | N/A | ✅ 4 functions → 1 table, net −97 lines |

### Test Summary

- **Total new/modified test functions this session**: 13 (2 wake-path, 1 D-B approval, 2
  Remember, 1 R-TLS-008, 1 defect-6, 3 park-test fixes + rename, 1 cardinality re-point, 1
  loop-wiring, 1 table-driven consolidation replacing 4)
- **Total tests passing**: whole `backend/agent` module — 1176 `--- PASS` lines, 0 `FAIL`,
  `go test -race -v ./...` exit 0
- **Layers used**: Unit/Integration (all AG-10 tests exercise the real `Scheduler`/`Turn`
  surface end-to-end within the package boundary — no mocks beyond the existing
  `ScriptedTool`/`scriptedPermissionPolicy` test doubles)
- **Approval tests**: 2 (D-B shutdown cleanup, table-driven refactor)
- **Flakiness verification**: `TestPermission_` group run `-count=20` clean;
  `TestPermission_RTLS008_...` `-count=10` clean; cardinality re-point `-count=25` clean;
  full module `-count=1` repeated 8× clean

## Work Unit Evidence (all 11 commits)

Every commit message above records its own focused-test command and result inline (`git
log` on the branch is the durable record). Runtime harness: `agenttest`-scripted
`ai.ModelProvider` via `TestTurn_PermissionPolicy_WiredToSchedule` and the existing AG-09
tool-dispatch tests (`loop_tool_dispatch_test.go`) exercise the real loop→scheduler
boundary; N/A beyond that — AG-10 adds no new runtime/process boundary (design.md's
Threat Matrix: "N/A"). Rollback boundary: each commit is independently revertible
(`git revert <sha>`) without touching unrelated work, since each is scoped to one
deliverable behavior per the work-unit-commits discipline.

## Gates

| Gate | Result |
|---|---|
| `make test` (whole module, `-race`) | **PASS** — 1176 `--- PASS`, 0 `FAIL`, exit 0 |
| `make lint` (after `cache clean`) | **PASS** — `0 issues.`, reproduced across 5 independent cache-cleaned runs (see Risks — one non-reproducing transient finding at session start) |
| `make build` | **PASS** — `go build -trimpath ./...` exit 0 |
| `make vuln-check` | **FAIL — accepted, out of scope.** 5 Go stdlib vulnerabilities (`GO-2026-6218`, `-6090`, `-6089`, `-5972`, `-5026`) at `go1.26.5`, fixed in `go1.26.6`. Every trace runs exclusively through `src/ai/openaicompat/**` (Layer 1: `execute_once.go`, `stream.go`, `openrouter/transport.go`, `conformancetest/retry.go`) — zero AG-10 files (`src/agent/**`) appear in any trace. Per explicit instruction: no toolchain upgrade attempted. |
| `TestTurn_SubstrateUntouched` (NFR-TLS-003) | **PASS** — `git diff main` against the 10 named substrate files is 0 lines |

## Deviations from Design

1. **`Scheduler` struct fields live in `tool.go`, not `scheduler.go`.** design.md's File
   Changes table lists only `scheduler.go` as modified for AG-10, but `type Scheduler
   struct` is declared in `tool.go` (AG-09.1's file). `WakeParked`'s `parkedMu`/`parked`
   fields were added there (2 fields, ~10 lines); `WakeParked` the method itself lives in
   `scheduler.go` alongside `Schedule`. Minimal, mechanical, and the only place the fields
   could go without moving the whole struct declaration.
2. **`openspec/AGENTS.md` had no AG-07/08/09 precedent to "match".** Verified via
   `git log` and the three milestones' own archived `tasks.md`: none of AG-06.1, AG-08, or
   AG-09 ever added a per-milestone pointer to this file (AG-08/AG-06.1 only *read* it).
   Task 1.3 as originally written assumed a precedent that does not exist. Implemented
   task 1.3's literal ask (an AG-10 pointer + the NFR-TLS-003 invariant) honestly, framed
   as the *first* such pointer rather than falsely claiming continuity.
3. **`parkedSet`'s channel type churned `chan struct{}` → `chan error` → `chan
   struct{}`** across this session's commits (visible in `9052381c` then `22297d84`). The
   `chan error` design was already sitting in the working tree from a prior session
   (AG-10.3 partial) before D-B's analysis proved its underlying rationale
   (distinguishing wake from a shutdown sweep) moot. Committed the AG-10.3 state as
   received, then reverted it in a clearly-separated D-B commit rather than silently
   rewriting history — the diff tells this story accurately.

## Risks / Discovered Issues Not Fixed (flagging, not hiding)

1. **A genuine, if narrow, concurrent-remember race exists.** Two calls to the *same*
   tool name scheduled on the *same* (unbounded-relative-to-each-other) read-class lane
   can both pass the `remembered.remembered(name)` pre-check before either's
   `remembered.remember(name)` write lands, independently reach `AllowAlways`, and each
   independently emit `permission_resolution_remembered` — a real double-emission the
   `CardinalityAtMostOne` validator (not the scheduler) backstops.
   `TestPermission_RememberedCardinality_SecondEmissionRejected` now demonstrates this
   deterministically via a forced synchronization barrier (not left to scheduling luck).
   R-APP-010's own wording ("subsequent calls") is about sequential/temporal precedence,
   not concurrent races, so this is not a spec violation — but it is a real edge case a
   future milestone (or a defensive "reserve-before-resolve" lock) could close. Not fixed
   here: out of the assigned scope, and closing it risks changing read-class concurrency
   semantics AG-09.2 charters.
2. **`make lint`'s one non-reproducing transient finding.** The very first `make lint`
   run of this session (after an explicit `cache clean`) reported a 9th issue —
   `src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go:17:9:
   var-naming: don't use an underscore in package name` — in a file this session never
   touched (confirmed via `git status`/`git diff`, last commit `bbde0221`, unrelated
   AI-40.2 milestone). Four subsequent independent `cache clean` + full-module lint runs
   all reported `0 issues.`, including the file's own package in isolation. Treating the
   original 9-issue count as a stale/transient artifact rather than a real pre-existing
   defect, consistent with this repo's own documented gotcha ("A lint cache artifact can
   masquerade as a finding"); `make lint` is genuinely clean, not merely "clean of
   AG-10's own issues".
3. **AllowOnce/AllowAlways/Deny's `decision_made` constructor-failure paths (defect 6)
   are fixed but not independently RED-tested.** Only ModifyInput has a
   policy-controlled way to force `NewPermissionDecisionMade` to fail (empty
   `ModifiedArgs`); the other three outcomes' constructors only fail on an empty
   `runID`/`turnID`, which would require a malformed `Schedule` caller, not a malformed
   policy — a much less natural test to construct without bypassing this file's own
   helpers. The defensive fix is applied uniformly to all four branches on the strength
   of the pattern already proven correct at the `decision_required` constructor site
   (`scheduler.go:580-590`) and at ModifyInput.

## Files Changed (this session)

| File | Action | What |
|---|---|---|
| `backend/agent/src/agent/permission_protocol.go` | Modified | `WakeParked`-supporting doc updates, `ResolveStrayDecision` deleted, `rememberedSet` added, `closeAll`/`errParkedSetShutdown`/`parkedCount` deleted, `chan error` reverted to `chan struct{}` |
| `backend/agent/src/agent/scheduler.go` | Modified | `WakeParked` method, `Scheduler.parked`/`parkedMu` publication, wake-recursion, ack-gated `decision_required` emission, `remembered` threading + `ctx`-first reorder (3 functions), swallowed-error fixes (4 branches), D-B dead-code deletion |
| `backend/agent/src/agent/tool.go` | Modified | `Scheduler` struct gains `parkedMu sync.Mutex` + `parked *parkedSet` fields |
| `backend/agent/src/agent/loop.go` | Modified | `TurnOptions.PermissionPolicy` field; `Schedule` call forwards it instead of literal `nil` |
| `backend/agent/src/agent/permission_protocol_test.go` | Modified | ~1450 net new lines: wake-path tests, D-B approval test, Remember tests, R-TLS-008 test, defect-6 test, park-test fixes, cardinality re-point, table-driven four-outcomes |
| `backend/agent/src/agent/loop_tool_dispatch_test.go` | Modified | `TestTurn_PermissionPolicy_WiredToSchedule` + `wiringTestPolicy` helper |
| `openspec/AGENTS.md` | Modified | NFR-TLS-003 substrate-list section + AG-10 pointer |
| `openspec/changes/cachicamas-agent-permission-protocol/*.md` | Tracked | proposal/design/tasks/exploration/specs — were entirely untracked at session start |

## Remaining Tasks

None. 20/20 `tasks.md` items are `[x]`, with task 5.4 explicitly annotated as failing for
reasons outside AG-10's scope (see Gates).

## Workload / PR Boundary

- Mode: **single PR**, `size:exception` pre-authorized (Review Workload Forecast:
  2500–3500 estimated lines, High budget risk, `size-exception` chain strategy —
  unchanged from `sdd-tasks`).
- Current work unit: N/A — this batch completes the change; no further slicing needed.
- Boundary: this apply batch starts from the orchestrator-audited partial state
  (`6ec55de7` + ~301 uncommitted lines) and ends with all 20 tasks complete, all gates
  green except the pre-existing `vuln-check` WARNING.
- Estimated review budget impact: large (~3050 net insertions this session on top of the
  ~1150 already landed pre-session) — consistent with the pre-authorized exception; no
  further decision needed before `sdd-verify`.

## Status

20/20 tasks complete. Ready for `sdd-verify`.

---

# Remediation Round (post-verify)

**Session**: scoped remediation on top of the FINAL state above (`c0fcc6e5`). `sdd-verify`
returned PASS WITH WARNINGS (0 CRITICAL, 8 WARNING, 5 SUGGESTION; `verify-report.md`,
Engram `#3040`). The orchestrator selected 5 of those items (W1, W2, W3, W4, W8, plus S1,
S2, S4 — the sub-items those five decompose into) as real gaps to close now; W5/W6 (the
archive-phase promoted-spec transform), W7 (accepted pre-existing `make vuln-check`
stdlib advisories), and S3/S5 (accepted/cosmetic) were explicitly carried forward per the
orchestrator's scoping, not fixed here. Nothing outside this list was touched.

## Commits (this round, oldest first)

| SHA | Message |
|---|---|
| `2fcf8fb4` | docs(agent): AG-10 track verify-report.md |
| `c41ed864` | fix(agent): AG-10 close the wake-registration race and the concurrent-remember race (W1, W2, W3, W4, S1, S2) |
| `9908313c` | test(agent): AG-10 add the Turn()-level permission E2E and a no-op policy helper (W8, S4) |
| `b378cc76` | fix(agent): AG-10 widen the substrate filters for the two new remediation test files |

## TDD Cycle Evidence

| Item | RED (pre-fix) | GREEN (post-fix) | REFACTOR |
|---|---|---|---|
| W3 — wake-registration race | `TestPermission_WakeParked_SynchronousOnDecisionRequired_NoRetry`: flaky, 8/30 FAIL against unfixed code (non-vacuous, though not the literal 100% the verify finding estimated) | 15/15 deterministic PASS after reordering `parked.park` before the emission | `wakeParkedWithRetry` helper deleted (no longer needed); 4 callers re-pointed at direct calls; `TestPermission_DeferEmitsBeforePark` re-targeted at the invariant that still holds |
| W4 — `<-reqAck` not cancellation-aware | `TestPermission_AbandonedAckWithCancel_GateDeregistersPromptly`, isolated from W3 by temporarily reverting only the ack-select to a bare receive: 5/5 deterministic FAIL | 15/15 deterministic PASS after `select{<-reqAck;<-ctx.Done()}` + `parked.remove` | New `parkedSet.remove` also added to the pre-existing `parkCh`-select's cancel branch for exit-path completeness |
| W1/W2 — concurrent-remember race | `TestPermission_RememberedCardinality_ConcurrentRace_AtMostOneEmission`: 10/10 deterministic FAIL (count=2) against pre-CAS code | 25/25 deterministic PASS (count=1) after `rememberedSet.rememberIfAbsent` (CAS) | Original hand-built `CheckStream` validator test restored byte-identical under its original name (S-PPB-004), so validator coverage is not lost |
| S1 — S-PPB-001 vacuity | N/A — test-only strengthening of already-passing behavior, no production change | Confirmed still passing after the addition | — |
| S2 — best-effort telemetry site | N/A — deliberate no-change decision, justified by code comment | Existing tests continue to pass | — |
| W8 — Turn()-level E2E | N/A — pure test addition over already-implemented Schedule-level behavior (RTLS-008 already covers the mixed-outcome matrix at Schedule level) | `TestTurn_PermissionPolicy_E2E_DeferDenyModify` 10/10 deterministic PASS | — |
| S4 — no-op policy decorator | N/A — pure test-support addition | `TestNoOpPermissionPolicy_AllowsEverySynchronously` PASS | — |
| R7 — substrate filter (discovered, not assigned) | `TestTurn_SubstrateUntouched` / `TestTurn_PreRequestHook_SubstrateUntouched` FAIL after the W8/S4 commit landed | Both PASS after widening `filterOutLoopFiles` / `filterOutLoopHookFiles` | — |

## Work Unit Evidence

| Work unit | Focused test command / result | Runtime harness | Rollback boundary |
|---|---|---|---|
| `2fcf8fb4` (track verify-report) | N/A — docs only | N/A | revert commit |
| `c41ed864` (W1–W4, S1, S2) | `go test -race -count=10 -run TestPermission ./src/agent/` — 230 `--- PASS`, 0 `FAIL`, 0 `DATA RACE`, exit 0 | Real `Schedule`/`Turn` calls via `agenttest`-free scripted policies (Schedule-level, no fake provider needed for these items) | revert commit — touches only `permission_protocol.go`, `scheduler.go`, `permission_protocol_test.go` |
| `9908313c` (W8, S4) | `go test -race -count=10 ./src/agent/...` — all green | `agent.Turn` + `agenttest.NewProvider` scripted fake — genuine E2E through the loop's public surface | revert commit — two new files only, no existing file touched |
| `b378cc76` (substrate filter) | `go test -race -run "TestTurn_SubstrateUntouched\|TestTurn_PreRequestHook_SubstrateUntouched" -v ./src/agent/...` — both PASS | N/A (a git-diff-based guard, not a runtime path) | revert commit — two filter functions only |

## Gates (final re-verification, this round)

| Gate | Result |
|---|---|
| `make test` (whole module, `-race`) | **PASS** — 1181 `--- PASS`, 0 `FAIL`, exit 0, 12 `ok` packages |
| `make lint` (after `cache clean`) | **PASS** — `0 issues.` (one `revive` empty-block finding in the W4 test's drain loop found and fixed within this round, before the commit) |
| `make build` | **PASS** — `go build -trimpath ./...` exit 0 |
| `go test -race -count=10 -run TestPermission ./src/agent/` | **PASS** — 230 `--- PASS`, 0 `FAIL`, 0 `DATA RACE`, exit 0 |
| `make vuln-check` | **FAIL — accepted, unchanged.** Same 5 pre-existing Go stdlib advisories at `go1.26.5`, zero `src/agent/` traces (verified: `grep -c "src/agent/"` over the output = 0) |
| Substrate (10 named files) | **PASS** — `git diff --stat $(git merge-base main HEAD) -- <10 files>` = 0 lines against the true merge-base `6de08335`; `TestTurn_SubstrateUntouched` PASS |

Note on the merge-base check: a PLAIN `git diff main -- <files>` in this worktree shows a
non-zero diff on `Makefile`/`go.mod`, because the shared `main` branch advanced to
`b7ec9906` (an unrelated upstream merge, PR #170) after this branch's base commit
`6de08335` — NOT because of any edit made in this session. The correct, merge-base-relative
diff (matching the method `verify-report.md` itself used) is the authoritative check and
is 0 lines.

## Design Deviations (this round)

1. **W8's E2E does not include a wake-resume cycle for the deferred call.** `Turn()`
   constructs its `*Scheduler` as an unexported local variable and returns only
   `(ai.Message, ai.FinishReason, error)` — nothing outside the package can reach
   `WakeParked` for a call parked inside a `Turn()` invocation. `loop.go`'s own comment
   states this is deliberate: "the loop's own upward-path wake wiring
   (`Scheduler.WakeParked`) is AG-13's scope." Widening `Turn`'s public surface to expose
   a wake handle is an AG-13-scoped design decision this remediation round has no mandate
   to make. The E2E instead proves all three outcomes reachable from `Turn()`: Deny and
   ModifyInput complete fully with the exact stream shape design.md specifies; Defer
   produces `decision_required` and aborts via context cancellation — the one release
   path actually reachable from outside `Turn()` today.
2. **S4's decorator lives in `agent_test` (package `agent_test`, file
   `permission_policy_helpers_test.go`), not in `agenttest`.** `agenttest`'s own doc.go
   states it "imports only the standard library and this module's own `src/ai`"
   (R-STK-009), enforced as a build failure by `src/ai/import_boundary_test.go`'s Layer 1
   closure guard. A `PermissionPolicy` implementation necessarily references Layer 2
   types (`agent.PermissionVerdict`, `agent.PermissionOutcome`), so it cannot compile
   inside `agenttest`. `scripted_tool_test.go` already documents the identical
   constraint for `ScriptedTool`, verbatim: "Layer 1's `agenttest` package cannot import
   the `agent` package."

## Explicitly Out of Scope (per orchestrator instruction, not re-litigated)

- **W5, W6** — the promoted-spec transform (`openspec/specs/agent-permission-protocol/spec.md`)
  and its Given/When/Then scenarios. The archive phase owns this; no file was written
  under `openspec/specs/` in this round.
- **W7** — `make vuln-check`'s 5 pre-existing stdlib advisories at `go1.26.5`. Accepted;
  no toolchain upgrade attempted.
- **S3, S5** — accepted (design.md E2E note superseded by W8's delivery) / cosmetic
  (PASS-line counting method).

## Remaining Tasks

None. All 5 assigned remediation items done, plus one necessarily-coupled substrate-filter
fix discovered during gate re-verification. Ready for `sdd-verify` (re-run) or archive, at
the orchestrator's discretion — this executor does not push, open a PR, or archive.

## Status (remediation round)

5/5 assigned items done (W1, W2, W3, W4 as one group; W8+S4 as one group), plus the
substrate-filter correction. All gates green except the explicitly-accepted
`make vuln-check`. Nothing left uncommitted.

---

# Second Remediation Round (W9, S6 — terminal re-verification)

**Session**: tightly scoped remediation on top of the round-1 remediation state
(`ecfebcd4`). `sdd-verify`'s terminal re-verification pass returned PASS WITH WARNINGS (0
CRITICAL, 4 WARNING, 3 SUGGESTION; `verify-report.md`). Two of those findings came from
round 1's own changes: **W9** (a new WARNING — the R-APP-002/D4 ack lost its only
non-vacuous test guard) and **S6** (a SUGGESTION — the test round 1's fix designated as
the W3 RED bite does not actually catch a reverted reorder). The orchestrator scoped this
round to exactly those two findings. W5/W6 (archive-phase promoted-spec transform), W7
(accepted pre-existing `make vuln-check` stdlib advisories), and S7/S8 (cosmetic) were
explicitly out of scope and untouched. No production-code behavior change was made or
required: the ack and the W3 reorder are both correct; this round adds/corrects test
guards only.

## Commits (this round, oldest first)

| SHA | Message |
|---|---|
| `e7073a19` | docs(agent): AG-10 record terminal re-verification findings (W9, S6) |
| `90fde05f` | test(agent): AG-10 pin the R-APP-002/D4 ack ordering guard (W9) |
| `7ab11679` | docs(agent): AG-10 correct the false W3 RED-bite claim on the no-retry test (S6) |

Plus this docs commit (`tasks.md` + `apply-progress.md`, both backends).

## W9 — the R-APP-002/D4 ack lost its only non-vacuous test guard

**Root cause of the gap.** `TestPermission_DeferEmitsBeforePark`'s round-1 rewrite (the
W3 fix) proves `decision_required` is genuinely delivered even when a wake races ahead of
it, and proves registration precedes emission — but post-W3, an early wake succeeds
*purely* from registration timing, with or without the ack. That test's early-wake
polling loop therefore cannot distinguish "ack present" from "ack deleted": both shapes
let it pass. The ack's own, narrower job — keeping the gate goroutine from reaching the
parked-WAIT select (and therefore from completing an already-arrived wake's re-Resolve,
emitting `decision_made`, and letting the tool run) until `decision_required` has
actually reached `sink` — had no test observing it at all.

**Fix: new test, no production change.**
`TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery`
(`permission_protocol_test.go`) wakes early using the same technique as
`TestPermission_DeferEmitsBeforePark` (poll `WakeParked` for up to 500ms without ever
reading `sink` — post-W3 registration is fast enough that this reliably succeeds), then —
BEFORE reading anything from the still-unbuffered, still-unread `sink` — polls for 300ms
asserting the tool has NOT run and `Schedule` has NOT returned. With the ack present this
is not a timing race to get lucky on: the dispatcher is structurally blocked on
`sink <- &stamped` (nothing has read it), so `close(reqAck)` cannot have happened, so the
gate cannot have left the ack-select, so the tool cannot have run — true for the entire
window, however long, not merely likely. Only afterward does the test read
`decision_required` off `sink`, unblocking the dispatcher's ack close and letting the
already-woken gate complete normally.

**Defeat-test evidence (full package, this round).**

RED — `reqAck` deleted entirely in `scheduler.go` (`emissions <- emission{ev: reqEv}`
with no ack field, no wait — the exact deletion the verify finding described): 1
in-process full-package run + 5 separate full-package process invocations
(`go test -race ./src/agent/...`, no `-run` filter), **6/6 failed**, all with
`ack_gate_tool invocations = 1, want 0 before decision_required reached sink`. Only the
new test failed each time; nothing else regressed from this specific deletion.

GREEN — `scheduler.go` restored via `git restore` (confirmed byte-identical to HEAD via
`git diff --stat`): 1 in-process full-package run + 3 separate full-package process
invocations, **4/4 clean**; `go test -race -count=15 ./src/agent/` also clean (zero
flakes, 7.596s).

## S6 — the designated W3 bite does not actually catch the W3 revert

**Root cause.** `TestPermission_WakeParked_SynchronousOnDecisionRequired_NoRetry` (round
1's R1 cited this as the W3 RED bite) does not reliably reproduce that failure. Its
`drainUntilDecisionRequired` consumer buffers `sink` (capacity 64) and hands the "ready"
signal through a *separate goroutine* plus a `sync.Once`-gated channel close — that extra
scheduling latency reliably gives `park()` enough of a head start to have already run by
the time the test's `WakeParked` call actually executes, regardless of whether
registration precedes or follows the emission. The verify pass measured 0/40 isolated,
0/20 full-package under a reverted reorder; re-measured this round (full package,
unfiltered, 4 runs): 3 PASS, 1 FAIL — still fundamentally unreliable, not the
deterministic bite its own doc comment and round 1's `tasks.md` R1 claimed.

**Fix chosen: documentation correction, not a strengthened test.** The invariant is
already, factually, better pinned by a different, existing test:
`TestPermission_DeferEmitsBeforePark` (S-PPB-002). Its early wake runs directly in the
polling loop's own goroutine against an *unbuffered* sink with no buffering and no extra
hop, so pre-fix it reliably loses (rejected for the entire poll window, because the old
ordering gates `park()` behind an ack that cannot close while sink stays unbuffered and
unread) and post-fix it reliably wins. Building a second, differently-shaped
"unbuffered + synchronous consumer" test to *also* chase this same race was considered
and rejected: it would still be a probabilistic, scheduler-dependent assertion in
character (even if empirically very reliable), sitting redundantly next to a test that is
already structurally deterministic for the identical property. Corrected instead:

1. `TestPermission_WakeParked_SynchronousOnDecisionRequired_NoRetry`'s doc comment
   (`permission_protocol_test.go`) — dropped the false "RED before the fix... GREEN
   after" claim, named `TestPermission_DeferEmitsBeforePark` as the real, deterministic
   reorder guard, and reframed this test as what it legitimately still proves: given the
   realistic buffered/async consumption pattern, a single `WakeParked` attempt made
   immediately upon observing `decision_required` succeeds with no retry needed (a
   no-retry/latency characterization test, not a defeat test for the W3 reorder).
2. `tasks.md` Phase 6 R1 — added a `**CORRECTED in Phase 7 R9**` note pointing at the
   same facts, without deleting the historical record of what round 1 actually did.

No test body and no production code changed for this item.

**Defeat-test re-confirmation (full package, this round, reorder temporarily reverted in
`scheduler.go`: `parked.park(call.ID())` moved back to run AFTER the ack-select, matching
the pre-W3 shape).**

RED — `TestPermission_DeferEmitsBeforePark`: **4/4 full-package runs FAILED**
deterministically (matching round 1's 20/20). Same revert also incidentally failed
`TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery` (expected: that
test also depends on early registration succeeding, an orthogonal property to the one
under test in this defeat run) and, non-deterministically across the 4 runs, a few other
D-A wake-path/RTLS-008 tests that also depend on the W3 ordering —
`TestPermission_WakeParked_SynchronousOnDecisionRequired_NoRetry` itself failed in only 1
of the 4 runs, confirming its unreliability as a guard rather than disproving the revert
worked.

GREEN — `scheduler.go` restored via `git restore` (confirmed byte-identical to HEAD): 3
fresh full-package runs, **3/3 clean**.

## Gates (final re-verification, this round)

| Gate | Result |
|---|---|
| `make test` (whole module, `-race`) | **PASS** — 1182 `--- PASS`, 0 `FAIL`, exit 0, 12 `ok` packages (was 1181; +1 for the new W9 test) |
| `make lint` (after `cache clean`) | **PASS** — `0 issues.` |
| `make build` | **PASS** — `go build -trimpath ./...` exit 0 |
| `go test -race -count=15 ./src/agent/` | **PASS** — zero flakes, 7.596s |
| `make vuln-check` | **FAIL — accepted, unchanged.** Same 5 pre-existing Go stdlib advisories (`GO-2026-5026`, `-5972`, `-6089`, `-6090`, `-6218`) at `go1.26.5`, zero `src/agent/` traces |
| `TestTurn_SubstrateUntouched` + `TestTurn_PreRequestHook_SubstrateUntouched` | **PASS** |
| Substrate (10 named files) merge-base diff | **PASS** — `git diff --stat $(git merge-base main HEAD) -- <10 files>` = 0 lines against merge-base `6de08335` |

## Files changed this round

`backend/agent/src/agent/permission_protocol_test.go` (modified: 1 new test for W9, 1
doc-comment-only correction for S6 — no other test bodies or production files touched),
`openspec/changes/cachicamas-agent-permission-protocol/verify-report.md` (tracked, was
uncommitted), `openspec/changes/cachicamas-agent-permission-protocol/tasks.md` and this
`apply-progress.md` (this docs commit).

## Explicitly out of scope (per orchestrator instruction)

- **W5, W6** — the promoted-spec transform. Archive phase owns it; still untouched.
- **W7** — `make vuln-check`'s 5 pre-existing stdlib advisories. Accepted, unchanged.
- **S7, S8** — cosmetic (design.md back-annotation note; wall-clock sleep margins in an
  unrelated test).

## Remaining Tasks

None. Both assigned items (W9, S6) done. All gates green except the explicitly-accepted
`make vuln-check`. Nothing left uncommitted. Ready for `sdd-verify` re-run or archive at
the orchestrator's discretion — this executor does not push, open a PR, or archive.

## Status (second remediation round)

2/2 assigned items done (W9, S6). Zero production-code changes. All gates green except
the explicitly-accepted `make vuln-check`. NFR-TLS-003 substrate guard intact (0-line
diff on all 10 files). Nothing left uncommitted.
