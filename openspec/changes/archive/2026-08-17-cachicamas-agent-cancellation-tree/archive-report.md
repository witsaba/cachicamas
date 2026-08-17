# Archive Report — AG-14: Build the cancellation tree

**Change**: `cachicamas-agent-cancellation-tree`  
**Status**: ARCHIVED  
**Date**: 2026-08-17  
**Verification**: PASS WITH WARNINGS (0 CRITICAL, 0 MAJOR, 0 blockers)  
**Observation IDs**: proposal #3234, spec #3241, design #3236, tasks #3245, verify-report #3263

## Executive Summary

AG-14 delivers a two-signal cancellation tree with bounded wind-down as a package-wide Layer 2 contract. The capability makes stopping a run a typed, bounded, two-signal operation: interrupt aborts the run and keeps the harness value; shutdown aborts the run and terminally refuses new prompts. Both signals propagate downward through loop, provider, and tools as a single context cancellation cause, remain `errors.Is`-distinguishable, and bind AG-19/AG-21. The harness-owned tasks exit within a 100ms wind-down bound; only third-party tool code may survive, named and detached by a typed report.

## Deliverables

### New Capability: `agent-cancellation-tree`

Eight requirements (`R-CAN-001` through `R-CAN-008`) define:

1. **R-CAN-001** — Two named signals (interrupt, shutdown) propagate by cancellation cause through loop, permission gate, and scheduler. No new parameters; real contexts flow down to `tool.Run`.
2. **R-CAN-002** — Interrupt ends the run (not harness), with nil failure on run-close and same-value reuse enabled at next `Run`.
3. **R-CAN-003** — Permission suspension abort typed to name which signal fired.
4. **R-CAN-004** — Second signal no-op, first cause wins, shutdown flag latches.
5. **R-CAN-005** — Shutdown identical wind-down, then terminal refusal flag blocks further `Run` calls.
6. **R-CAN-006** — Wind-down bounded (100ms default, zero-default field override); armed only on cancellation; overrun call typed and detached with tool/call identity report.
7. **R-CAN-007** — `RunOutcomeShutdown` vocabulary member added to `run_events.go` under `R-LSK-004` release.
8. **R-CAN-008** — `L2C-08` contract row declaring cancellation as package-wide liveness/control guarantee.

### Five Cross-Cut Deltas

| Spec | Modifications |
|---|---|
| **agent-loop-skeleton** | `R-LSK-004` expanded to record AG-14 substrate release (`run_events.go`, `doc.go`, `doc_contract_guard_test.go`); scenario count retired, range stated instead; three new filenames added to both filters (`/cancellation.go`, `/cancellation_interrupt_test.go`, `/cancellation_shutdown_test.go`, `/cancellation_winddown_test.go`, `/cancellation_events_test.go`, `/run_events.go`). New scenarios `S-LSK-018`, `S-LSK-019`, `S-LSK-020` record the release and filter bounds. |
| **agent-run-driver** | `R-RUN-001` widened to four methods (Interrupt, Shutdown added); serial reuse of harness value enabled. `R-RUN-010` explicitly answers the wind-down bound is NOT a third path. `R-RUN-011` carved out for cancellation: synthesize orphans, nil failure on run-close, no retry. New scenario `S-RUN-003`, `S-RUN-092`, `S-RUN-102`. Non-requirements table back-annotated: cancellation semantics partially closed, `L2C-08` closed, stale pointer to `agent-loop-skeleton/spec.md:106` corrected. |
| **agent-permission-protocol** | `R-APP-009` typed abort now names which signal cancelled (interrupt vs. shutdown). New scenarios `S-APP-017`, `S-APP-018`. Non-requirements table back-annotated: full cancellation tree closed. |
| **agent-tool-scheduler** | `R-TLS-013` added: context flows to `tool.Run`. `R-TLS-014` added: wind-down bound (zero-default field), armed only on cancellation, detached-call report via existing execution-failure path. `R-TLS-010` restated: category corrected (`ai.FailureCategoryUnavailable`, not nonexistent `ai.FailureCategoryExecution`), disjoint-return-channel rule reaffirmed under cancellation. New scenarios `S-TLS-016`, `S-TLS-017`, `S-TLS-018`, `S-TLS-019`. |
| **agent-history** | `R-HIS-007` back-annotated: synthesis now has its first production caller (harness wind-down). Idempotency load-bearing. New scenarios `S-HIS-097`, `S-HIS-098` prove wind-down caller. Non-requirements table back-annotated: cancellation semantics closed. |

## Substrate Release

**Files released and edited under recorded `R-LSK-004` deltas:**

- **`run_events.go`** (first ever release) — Added `RunOutcomeShutdown` member between `RunOutcomeFailed` and `runOutcomeLimit`; `String()` case `"shutdown"`. `RunEnd.validate` and `NewRunEnd` signature **byte-unchanged**. New member value selected to preserve `Completed`/`Interrupted`/`Failed` at 1/2/3.
- **`doc.go`** — Added `L2C-08` contract row (AG-14 only).
- **`doc_contract_guard_test.go`** — Added `L2C-08` entry to `expectedLayer2ContractRows` in same PR per guard's own rule.

**New files created** (six enumerated by exact filename in filter widening):
- `/cancellation.go` — Signal and sentinel definitions.
- `/cancellation_interrupt_test.go` — Interrupt path tests.
- `/cancellation_shutdown_test.go` — Shutdown path tests.
- `/cancellation_winddown_test.go` — Wind-down bound and detached-call tests.
- `/cancellation_events_test.go` — Stream-level behavior tests.
- **Plus five existing files modified**: `harness.go`, `loop.go`, `scheduler.go`, `tool.go` (non-substrate), `doc.go`, `doc_contract_guard_test.go` (released).

**Files explicitly NOT edited** (substrate preservation):
- `stream_check.go`, `stream_check_test.go` (forbidden; validator coverage in cancellation_events_test.go)
- `turn_events.go`, `failure.go`, `history.go`, `go.mod`, `go.sum`, all files under `backend/agent/src/ai/**`

## Verification Outcome

**Final Verdict**: **PASS WITH WARNINGS** (0 CRITICAL, 0 MAJOR, 0 blockers; 8 MINOR, 4 SUGGESTION)

Re-verify pass at commit `e3dd0cd7` (merge base `52701436`) supersedes first-pass fail (3 MAJOR at `1ec63ccb`). Each closure independently re-proven via `go test -overlay` mutations.

### MAJORs Closed

1. **MAJOR-1 CLOSED** — Nonexistent `ai.FailureCategoryExecution` in `R-TLS-010`. Corrected to `ai.FailureCategoryUnavailable` in delta spec; observing and deaf tool tests (`scheduler_test.go:1449-1450,1465-1466`) verify the distinction.

2. **MAJOR-2 CLOSED** — Synthesized-origin clause only in `S-CAN-001` Arm A (seeded open call). Arm B clarified: property defended *indirectly* by asserting all entries are `appended`, which only succeeds if calls were resolved not orphaned. Non-vacuity proven: deleting `windDownRun`'s synthesis fails Arm A (history count mismatch).

3. **MAJOR-3 CLOSED** — Interrupt/shutdown outcome and failure-presence assertions in turn-close events (`cancellation_interrupt_test.go:126-145`, `cancellation_shutdown_test.go:214-233`). Both halves proven live: mutating failure to `Unavailable` fails both tests; mutating to `TurnOutcomeUnknown, nil` fails both on outcome AND failure-presence.

### MINOR Remediated

**MINOR-3 CLOSED** — Defer registration order in `harness.go:372`. Defect: mutations to `cancelRun` after defer close could race. Fix: register `close(sink)` defer first so LIFO clears `cancelRun` first. Every exit path audited (post-shutdown early return closes sink itself). Regression test: 3/3 RED against inverted order (attempts 17/76/21 of 2000); 0/22,000 green-side false positives. Module test cost: +4.3s (2.853s → 7.109s in 173s suite). Net-positive.

### MINORs Left Open (Known Limitations)

- **MINOR-2** (`S-TLS-018`, unobservable "no timer created") — True but unobservable from outside package; documented in spec.
- **MINOR-4** (`windDownRun` discards transcript errors, `harness.go:274-275`) — Consistent with `failRun` precedent; acceptable trade-off.
- **MINOR-5** (leak snapshot taken after third-party release) — Permanent leak falsifiable; snapshot timing correct.
- **MINOR-6** (`S-LSK-019` final clause untested) — Known limitation flagged for future work.
- **MINOR-7** (apply-progress.md scenario count stale) — Intermediate snapshot; current count is 3267 PASS re-verified 2026-08-17.
- **MINOR-1** (spec.md:96 disjoint-sets wrapper goroutine misclassified) — Promoted verbatim per archive spec sync; not a defect in delivery.

### SuGGESTIONs (Accepted)

- **S-1** — Spec clause rewording for clarity (complied in archive).
- **S-4** — Raise `time.After` deadline (`:717`) from 500ms to 3-5s to remove CI false-RED vector (accepted).
- Others: Documentation and comment improvements.

### Pre-existing Issues Surfaced (Not Fixed)

1. **Go 1.26.6 `gofmt` drift** — 18 files including byte-unchanged ones. Repo-wide issue; needs separate action.
2. **golangci-lint cache version mismatch** — Phantom findings when cache cleaned with version 2.12.2 binary vs. pinned 2.9.0 invocation. Documented; clean with pinned binary always.

## Final Gate Results

All conducted at final commit `e3dd0cd7`:

| Gate | Result | Details |
|---|---|---|
| `go test -race` | **PASS** | 3267 PASS / 0 FAIL / 0 DATA RACE, all 12 packages ok, 2m53s |
| `make lint` | **PASS** | 0 issues (after `./bin/golangci-lint cache clean`) |
| `make build` | **PASS** | Clean build |
| `make vuln-check` | **PASS** | 0 findings |
| `loop.go` coverage | **88.01%** | ≥ 80% per NFR-CAN-004 |
| Substrate byte-identity | **PASS** | 7-file `S-LSK-018` set, 12 forbidden files, 35-entry filter pair, `go.mod`/`go.sum`, `backend/agent/src/ai/**` all byte-unchanged |

**Note**: Final passing-test count (3267) not recorded in archive; counts drift on next append. Verify-report snapshots are intermediate; this archive report records final state per the skill's Final-State Authority section.

## Specifications Promoted

Six canonical specs updated/created and committed under AG-14's changes:

| Spec | Action | Details |
|---|---|---|
| `openspec/specs/agent-cancellation-tree/spec.md` | **CREATED** | 152 lines, 8 requirements, 8 scenarios + 3 bites, new capability |
| `openspec/specs/agent-loop-skeleton/spec.md` | **UPDATED** | `R-LSK-004` expanded; scenario count retired, range stated; new scenarios S-LSK-018/019/020; filters widened by 6 exact filenames |
| `openspec/specs/agent-run-driver/spec.md` | **UPDATED** | `R-RUN-001` four methods; `R-RUN-010` wind-down bound answer; `R-RUN-011` cancellation carve-out; new scenarios S-RUN-003/092/102; non-requirements back-annotated |
| `openspec/specs/agent-permission-protocol/spec.md` | **UPDATED** | `R-APP-009` signal-aware abort typing; new scenarios S-APP-017/018; non-requirements back-annotated |
| `openspec/specs/agent-tool-scheduler/spec.md` | **UPDATED** | `R-TLS-013` context threading; `R-TLS-014` wind-down bound; `R-TLS-010` category correction; new scenarios S-TLS-016/017/018/019 |
| `openspec/specs/agent-history/spec.md` | **UPDATED** | `R-HIS-007` back-annotation; new scenarios S-HIS-097/098; non-requirements back-annotated |

## Traceability

**Observation IDs** (for future reference):
- Proposal: #3234
- Spec:    #3241
- Design:  #3236
- Tasks:   #3245
- Verify:  #3263

**Charter Coverage** (0003:1371-1442):
- AG-14.1 (interrupt) → `R-CAN-001`, `R-CAN-002`, `R-CAN-003`, `S-CAN-001` Arm A/B
- AG-14.2 (shutdown) → `R-CAN-005`, `S-CAN-005`
- AG-14.3 (wind-down) → `R-CAN-006`, `S-CAN-006`
- L2C-08 (doc contract) → `R-CAN-008`, `S-CAN-008`

**Cross-Document Fixes**:
- Stale pointer `agent-run-driver/spec.md:275` corrected to point to `agent-run-driver/spec.md:72` (pre-existing defect, not propagated)
- Scenario count assertion retired in `agent-loop-skeleton/spec.md` per repo's known drift class (fixed by range statement, never a total)

## Release Sign-Off

AG-14 is ready for next wave planning. The change:
- ✅ Delivers two-signal cancellation tree with bounded wind-down
- ✅ Records substrate release (`run_events.go` first ever, `doc.go`/`doc_contract_guard_test.go`)
- ✅ Passes all verification gates (0 CRITICAL, 0 MAJOR, 8 MINOR accepted as known limitations)
- ✅ Promotes specs faithfully into canonical tree
- ✅ Establishes `L2C-08` contract row binding AG-19/AG-21
- ✅ Enables AG-15 (retry) and AG-21 (leak sweep) to build on typed cancellation semantics

**Next milestone**: AG-15 (retry on failed turn — out of scope in this cycle)

---

*Archived by `sdd-archive` executor on 2026-08-17.*  
*Change verification outcome: PASS WITH WARNINGS.*  
*Artifact state authority: review gate approval + post-apply validation.*
