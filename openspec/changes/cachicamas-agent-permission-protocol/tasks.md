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

- [ ] 1.1 Create `permission_protocol.go` — `PermissionPolicy` interface, `PermissionVerdict`, parked-set, scaffolding.
- [ ] 1.2 Create `permission_protocol_test.go` (`package agent_test`) — `t.Run` skeleton for AG-10.1..AG-10.4.
- [ ] 1.3 Update `openspec/AGENTS.md` — AG-10 pointer + NFR-TLS-003 invariant.

## Phase 2: RED bites

- [ ] 2.1 S-PPB-001 `TestPermission_ImmediateAllow_NoEvent`: sync `AllowOnce` → no `decision_required`, call executes.
- [ ] 2.2 S-PPB-002 `TestPermission_DeferEmitsBeforePark`: A `Defer` + B `AllowOnce` → A's `decision_required` reaches `sink` before B.
- [ ] 2.3 S-PPB-003 `TestPermission_StrayDecisionIsTypedError`: wake unknown `callID-X` → typed `ErrStrayDecision`.
- [ ] 2.4 S-PPB-004 `TestPermission_RememberedCardinality`: second `resolution_remembered` per `toolName` fails S-APE-082.

## Phase 3: Core implementation

- [ ] 3.1 AG-10.1 GREEN: `Resolve` immediate (AllowOnce → execute), `Defer` (emit → park), stray-rejection on wake.
- [ ] 3.2 AG-10.2 GREEN: four outcomes — `Deny` → `Result{ExecutionFailure, typedDenial}`; `ModifyInput` substitutes args, defers `ToolStart`.
- [ ] 3.3 AG-10.3 GREEN: wait = `select { <-parkCh; <-ctx.Done() }`; cancel walks set, aborts, no leaks.
- [ ] 3.4 AG-10.4 GREEN: invoke `policy.Remember`; emit `resolution_remembered` only on `true`; suppress on `false` (`CardinalityAtMostOne`).

## Phase 4: Integration

- [ ] 4.1 Extend `Scheduler.Schedule` signature with `policy PermissionPolicy` param; thread parked-set; preserve rejoin + AG-09's three sub-paths byte-clean.
- [ ] 4.2 Modify `loop.go:240-244`: pass `policy` into `Schedule`; wake = `close(parkCh)` keyed on `callID`; preserve `finalize → Schedule → closeSink` order.
- [ ] 4.3 R-TLS-008 source guard: 3 parked + 2 immediate through 4 outcomes — rejoin populated, ordinal order preserved, no leak under `-race`.
- [ ] 4.4 `TestTurn_SubstrateUntouched` filter widens to exclude `permission_protocol.go` + `permission_protocol_test.go`; NFR-TLS-003 7th carry.

## Phase 5: Verification

- [ ] 5.1 `cd backend/agent && make test` green with `-race`; 4 leaves + bites + R-TLS-008 pass.
- [ ] 5.2 `cd backend/agent && make lint` clean after `cache clean`.
- [ ] 5.3 `cd backend/agent && make build` clean.
- [ ] 5.4 `cd backend/agent && make vuln-check` clean.
- [ ] 5.5 Zero edits to 10 substrate files (NFR-TLS-003 7th carry).