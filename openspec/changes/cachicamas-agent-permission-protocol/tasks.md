# Tasks: AG-10 — Implement the permission protocol

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 2500–3500 |
| 400-line budget risk | High |
| Chained PRs recommended | No (size:exception pre-authorized) |
| Suggested split | single PR |
| Delivery strategy | single-pr (size:exception approved) |
| Chain strategy | size-exception (NOT chained) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Focused test command | Runtime harness | Rollback boundary |
|------|------|----------------------|-----------------|-------------------|
| 1 | Spec/design chore | `cd backend/agent && make test` | N/A | revert chore |
| 2 | RED bites | `cd backend/agent && go test -run "TestPermission_.*Bite" -race ./src/agent/...` | N/A | revert bites |
| 3 | AG-10.1: port + gate | `cd backend/agent && go test -run "TestPermission_AskSuspend" -race ./src/agent/...` | `agenttest` | revert AG-10.1 |
| 4 | AG-10.2: four outcomes | `cd backend/agent && go test -run "TestPermission_FourOutcomes" -race ./src/agent/...` | `agenttest` | revert AG-10.2 |
| 5 | AG-10.3: suspension+cancel | `cd backend/agent && go test -run "TestPermission_SuspensionDoesNotBlock" -race ./src/agent/...` | `agenttest` `chan` | revert AG-10.3 |
| 6 | AG-10.4: remembered | `cd backend/agent && go test -run "TestPermission_RememberedResolutions" -race ./src/agent/...` | `agenttest` rule | revert AG-10.4 |
| 7 | Loop wire-up + R-TLS-008 | `cd backend/agent && make test` | scripted turn | revert wire-up |
| 8 | Docs + apply-progress | `cd backend/agent && make lint && make build` | N/A | revert docs |

## Phase 1: Foundation

- [x] 1.1 Create `permission_protocol.go` — `PermissionPolicy` interface, `PermissionVerdict`, parked-set, scaffolding. (75ecd566, 20b481a8; `WakeParked`/`rememberedSet` added 34d295eb, aaa5c7b4; `parkedSet` shutdown-sweep cleanup 22297d84)
- [x] 1.2 Create `permission_protocol_test.go` (`package agent_test`) — `t.Run` skeleton for AG-10.1..AG-10.4. (52c8c9be — table-driven `TestPermission_FourOutcomes`, 4 subtests; `TestPermission_AllowAlways_Remember_Branches` also table-driven, aaa5c7b4)
- [x] 1.3 Update `openspec/AGENTS.md` — AG-10 pointer + NFR-TLS-003 invariant. (e18095e5 — see note below: no prior AG-07/08/09 precedent existed in this file to "match"; this is the first pointer ever recorded there)

## Phase 2: RED bites

- [x] 2.1 S-PPB-001 `TestPermission_ImmediateAllow_NoEvent`: sync `AllowOnce` → no `decision_required`, call executes. (5f03d600 RED, 20b481a8 GREEN — pre-existing, reverified passing this session)
- [x] 2.2 S-PPB-002 `TestPermission_DeferEmitsBeforePark`: decision_required reaches `sink` BEFORE the parked wait blocks (R-APP-002/D4 ordering, not just "before a sibling's own emission"). (e05e01cf — renamed from `TestPermission_DeferEmitsAndParks`; required an implementation change, an ack channel on the decision_required emission, because the buffered-`emissions`-channel design could not otherwise guarantee the ordering — see design note in apply-progress)
- [x] 2.3 S-PPB-003 `TestPermission_StrayDecisionIsTypedError`: wake unknown `callID-X` → typed `ErrStrayDecision`. (34d295eb — re-pointed at the real `Scheduler.WakeParked`; the original `ResolveStrayDecision` stub was deleted)
- [x] 2.4 S-PPB-004 `TestPermission_RememberedCardinality_SecondEmissionRejected`: second `resolution_remembered` per `toolName` fails S-APE-082. (ba1e8777 — re-pointed at a real `Schedule` run via a deterministic forced-race barrier; documents a genuine concurrent-remember race the validator backstops, see Risks)

## Phase 3: Core implementation

- [x] 3.1 AG-10.1 GREEN: `Resolve` immediate (AllowOnce → execute), `Defer` (emit → park), stray-rejection on wake. (20b481a8 base; wake re-evaluates the verdict via recursion into `runPermissionGate`, 34d295eb)
- [x] 3.2 AG-10.2 GREEN: four outcomes — `Deny` → `Result{ExecutionFailure, typedDenial}`; `ModifyInput` substitutes args, defers `ToolStart`. (6ec55de7; swallowed-constructor-error defect fixed 756c3e76)
- [x] 3.3 AG-10.3 GREEN: wait = `select { <-parkCh; <-ctx.Done() }`; cancel walks set, aborts, no leaks. (9052381c; dead `closeAll()` shutdown sweep removed 22297d84 once proven unreachable)
- [x] 3.4 AG-10.4 GREEN: invoke `policy.Remember`; emit `resolution_remembered` only on `true`; suppress on `false` (`CardinalityAtMostOne`). (aaa5c7b4 — includes the `remembered` per-Schedule suppression state for R-APP-010)

## Phase 4: Integration

- [x] 4.1 Extend `Scheduler.Schedule` signature with `policy PermissionPolicy` param; thread parked-set; preserve rejoin + AG-09's three sub-paths byte-clean. (20b481a8 base; `remembered` param + `ctx`-first reorder added aaa5c7b4)
- [x] 4.2 Modify `loop.go:240-244`: pass `policy` into `Schedule`; wake = `Scheduler.WakeParked(callID)`; preserve `finalize → Schedule → closeSink` order. (34d295eb — `TurnOptions.PermissionPolicy` field + wiring; wake is `WakeParked`, not a bare `close(parkCh)`, since D-A moved the parked set behind the Scheduler)
- [x] 4.3 R-TLS-008 source guard: 3 parked + 2 immediate through 4 outcomes — rejoin populated, ordinal order preserved, no leak under `-race`. (1adf6350 — `TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin`, deterministic settle-based goroutine check, `-count=10` clean)
- [x] 4.4 `TestTurn_SubstrateUntouched` filter widens to exclude `permission_protocol.go` + `permission_protocol_test.go`; NFR-TLS-003 7th carry. (pre-existing at session start, `loop_test.go` `filterOutLoopFiles`; reverified passing)

## Phase 5: Verification

- [x] 5.1 `cd backend/agent && make test` green with `-race`; 4 leaves + bites + R-TLS-008 pass. (verified: 1176 `--- PASS` lines, 0 `FAIL`, exit 0, whole module — not just `src/agent`)
- [x] 5.2 `cd backend/agent && make lint` clean after `cache clean`. (verified: `0 issues.` across 5 independent cache-cleaned runs, whole module. `ctx`-first reorder in `aaa5c7b4`; vestigial `_ = abortFailure` no-op branch removed `29a2dcfd`)
- [x] 5.3 `cd backend/agent && make build` clean. (verified: `go build -trimpath ./...` exit 0)
- [ ] 5.4 `cd backend/agent && make vuln-check` clean. **NOT clean — pre-existing, out of AG-10's scope.** 5 Go stdlib vulnerabilities at `go1.26.5` (GO-2026-6218, -6090, -6089, -5972, -5026), all fixed in `go1.26.6`, all traced exclusively through `src/ai/openaicompat/**` (Layer 1) — zero AG-10 files (`src/agent/**`) appear in any trace. A toolchain upgrade is out of scope for this milestone per explicit instruction; reported as an accepted WARNING, not fixed.
- [x] 5.5 Zero edits to 10 substrate files (NFR-TLS-003 7th carry). (verified: `git diff main -- <10 files>` = 0 lines; `TestTurn_SubstrateUntouched` PASS)

## Phase 6: Remediation (post-verify, orchestrator-scoped)

`sdd-verify` returned PASS WITH WARNINGS (0 CRITICAL, 8 WARNING, 5 SUGGESTION;
`verify-report.md`, Engram `#3040`). The orchestrator selected 5 of those items as real
gaps to close now; W5/W6 (archive-phase promoted-spec transform), W7 (accepted
pre-existing `vuln-check` stdlib advisories), and S3/S5 (accepted/cosmetic) were
explicitly carried forward, not fixed here.

- [x] R1 (W3) — structurally guaranteed lost wakeup on the wake surface.
  `parked.park(callID)` now runs BEFORE the `decision_required` emission, not after
  (`scheduler.go`'s `runPermissionGate` Defer branch reordered). RED:
  `TestPermission_WakeParked_SynchronousOnDecisionRequired_NoRetry` — a synchronous,
  no-retry `WakeParked` the instant `decision_required` is observed, ~27% failure over 30
  runs pre-fix (flaky, not the literal 100% the verify finding estimated, but real
  non-vacuous evidence), deterministic 15/15 GREEN post-fix (a genuine happens-before
  chain, not a scheduling race, now guarantees this). `wakeParkedWithRetry` deleted — the
  helper that existed only to paper over this gap — and its 4 callers re-pointed at
  direct `sched.WakeParked(id)` calls.
  `TestPermission_DeferEmitsBeforePark` (S-PPB-002) rewritten: its OLD assertion
  ("WakeParked must keep failing until sink delivers the event") is now the WRONG
  invariant by design once registration precedes emission — re-targeted at the invariant
  that still matters: `decision_required` is still genuinely delivered to `sink` even
  when a wake races ahead of it and closes `parkCh` before the gate ever reaches its
  parked wait.
- [x] R2 (W4) — `<-reqAck` was not cancellation-aware. Now
  `select { case <-reqAck: ; case <-ctx.Done(): parked.remove(callID); <typed abort> }`,
  reusing the same typed-abort shape the pre-existing `parkCh`-select's `ctx.Done()`
  branch produces (which now also calls the new `parked.remove(callID)` — every exit path
  deregisters: normal wake, cancel-while-parked, cancel-while-ack-waiting). New
  `parkedSet.remove` method (delete-without-close, so it never races `WakeParked`'s own
  close of the same entry). RED:
  `TestPermission_AbandonedAckWithCancel_GateDeregistersPromptly` — isolated from R1 by
  temporarily reverting only the ack-select to a bare receive (keeping R1's reorder): 5/5
  deterministic FAIL (the entry stays registered and reachable, so a post-cancel
  `WakeParked` spuriously "succeeds" instead of returning `ErrStrayDecision`). Restored:
  15/15 deterministic PASS.
- [x] R3 (W1 + W2) — concurrent-remember race. `rememberedSet.remember` replaced by
  `rememberedSet.rememberIfAbsent` — a compare-and-set under the set's own mutex; the
  scheduler's AllowAlways branch now gates the `resolution_remembered` emission on the
  CAS's own return value (loser suppresses its emission, still executes its call).
  `TestPermission_RememberedCardinality_SecondEmissionRejected` (S-PPB-004) RESTORED
  byte-identical to its pre-`ba1e8777` hand-built `CheckStream` form, so
  `CardinalityAtMostOne` validator coverage is not lost now that the scheduler stops
  producing the invalid stream. New
  `TestPermission_RememberedCardinality_ConcurrentRace_AtMostOneEmission` re-points the
  real-`Schedule` forced-race bite at the real invariant (`resolution_remembered count !=
  1` fails, not `!= 2`): RED 10/10 FAIL (count=2) pre-CAS, GREEN 25/25 PASS (count=1)
  post-CAS.
- [x] R4 (S1) — `TestPermission_ImmediateAllow_NoEvent` (S-PPB-001) now also asserts
  `policy.resolveInvocations() == 1`, closing the vacuity gap the verify audit found (the
  bite previously would have passed identically with the whole gate deleted).
- [x] R5 (S2) — `scheduler.go`'s `resolution_remembered` constructor-failure site (the
  same lines R3 touches) deliberately kept best-effort rather than upgraded to defect 6's
  typed-abort shape, with a code comment explaining why: by that point the primary
  decision (`decision_made{AllowAlways}`) is already durably recorded and the state
  change (`rememberIfAbsent`) already took effect and won; aborting here would retract
  execution of a call the stream already told the model is allowed. The failure is also
  unreachable in practice (same validated `run`, `call.Name()`'s non-empty invariant, a
  hardcoded valid outcome constant).
- [x] R6 (W8 + S4) — design.md's undelivered Turn()-level E2E and `agenttest` decorator.
  New `TestTurn_PermissionPolicy_E2E_DeferDenyModify` (`loop_permission_e2e_test.go`)
  drives Deny + ModifyInput to full completion through `Turn()`'s public surface (not
  just `Schedule()` — widening `TestTurn_PermissionPolicy_WiredToSchedule`'s single-Deny
  wiring proof); the Defer call produces `decision_required` then aborts via ctx-cancel,
  since `Turn()` exposes no handle to reach `WakeParked` for a call parked inside it
  (`loop.go`'s own comment: "the loop's own upward-path wake wiring ... is AG-13's
  scope") — recorded as a DEVIATION from design.md's literal wake-resume wording, not a
  silently-dropped scenario: widening `Turn`'s public surface to expose a wake handle is
  an AG-13-scoped design decision this remediation round has no mandate to make.
  `NoOpPermissionPolicy` (the no-op pass-through decorator) added in NEW file
  `permission_policy_helpers_test.go`, package `agent_test` — NOT `agenttest` as
  design.md literally says: `agenttest`'s own doc.go declares it "imports only the
  standard library and this module's own src/ai" (R-STK-009), enforced by
  `src/ai/import_boundary_test.go`'s Layer 1 closure guard; a `PermissionPolicy`
  implementation necessarily references Layer 2 types (`agent.PermissionVerdict`,
  `agent.PermissionOutcome`) and cannot compile there without breaking that guard. Same
  precedent `scripted_tool_test.go` already documents for `ScriptedTool` — "Layer 1's
  agenttest package cannot import the agent package".
- [x] R7 (discovered during gate re-verification, not one of the five assigned items) —
  `make test` failed after R6 landed: `TestTurn_SubstrateUntouched` and
  `TestTurn_PreRequestHook_SubstrateUntouched` both flag any new `src/agent/` file their
  `filterOutLoopFiles` / `filterOutLoopHookFiles` allowlists do not yet know about — the
  same mechanism task 4.4 already widened once for `permission_protocol.go` /
  `permission_protocol_test.go`. Widened both filters (`loop_test.go`, `loop_hook_test.go`
  — neither is one of the 10 NFR-TLS-003 substrate files) to also exclude
  `loop_permission_e2e_test.go` and `permission_policy_helpers_test.go`.

### Gates re-verified after remediation

| Gate | Result |
|---|---|
| `make test` (whole module, `-race`) | PASS — 1181 `--- PASS`, 0 `FAIL`, exit 0, 12 `ok` packages |
| `make lint` (after `cache clean`) | PASS — `0 issues.` (one `revive` empty-block finding in the W4 test's drain loop found and fixed during this round) |
| `make build` | PASS — `go build -trimpath ./...` exit 0 |
| `go test -race -count=10 -run TestPermission ./src/agent/` | PASS — 230 `--- PASS`, 0 `FAIL`, 0 `DATA RACE`, exit 0 |
| `make vuln-check` | FAIL — accepted, unchanged: same 5 pre-existing Go stdlib advisories at `go1.26.5`, zero `src/agent/` traces |
| `TestTurn_SubstrateUntouched` + merge-base diff | PASS — 0 lines changed across all 10 substrate files against merge-base `6de08335` |