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