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
| `make lint` (after `cache clean`) | **PASS** — `0 issues.`, reproduced across 4 independent cache-cleaned runs (see Risks — one non-reproducing transient finding at session start) |
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
