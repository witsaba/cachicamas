# Spec — The cancellation tree (`agent-cancellation-tree`)

> **Change**: `cachicamas-agent-cancellation-tree` · **AG-14** (Layer 2, Wave 3) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-14--build-the-cancellation-tree), `0003:1371-1442`
> **Nodes**: AG-14.1 `[leaf]` interrupt (`0003:1391-1414`) · AG-14.2 `[leaf]` shutdown (`0003:1415-1428`) · AG-14.3 `[leaf]` bounded wind-down (`0003:1429-1442`)
> **Status**: **new capability**. This file is the normative text; per the AG-09 / AG-10 / AG-11 / AG-12 / AG-13 precedent it is promoted to `openspec/specs/agent-cancellation-tree/spec.md` at archive. Five cross-cut deltas ship beside it and are promoted into [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md), [`../agent-run-driver/spec.md`](../agent-run-driver/spec.md), [`../agent-permission-protocol/spec.md`](../agent-permission-protocol/spec.md), [`../agent-tool-scheduler/spec.md`](../agent-tool-scheduler/spec.md) and [`../agent-history/spec.md`](../agent-history/spec.md).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && make test`.
> **IDs**: requirements `R-CAN-0NN`, scenarios `S-CAN-0NN` (bites carry the same `S-CAN-` prefix and are marked **(bite)**). Append-only. Distinct from `R-AEV-`/`R-AGE-`/`R-AGP-`/`R-AGM-`/`R-AGV-`/`R-ATT-`/`R-TLS-`/`R-APE-`/`R-PRH-`/`R-LSK-`/`R-AMT-`/`R-APP-`/`R-HIS-`/`R-RUN-`.
> **Allocated IDs**: `R-CAN-001` through `R-CAN-008`, and `S-CAN-001` through `S-CAN-008` plus the bites `S-CAN-010`, `S-CAN-011` and `S-CAN-012`. This header states the **allocated range and never a total**, because a total is defended by no test and goes silently false the moment a later milestone appends. Coverage per requirement is stated at each requirement, which is where a reader can check it against that requirement's own scenarios.
> **Evidence gate**: `cd backend/agent && make test`, plus `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check` (`vuln-check` is **not** in `make all`). No CI exists.
> **Sources**: charter `0003:1371-1442`; this change's SDD proposal and design document, whose three decisions are closed and are not re-opened here.
> **Ownership boundary**: this capability owns the cancellation *signals*, their propagation, the wind-down algorithm and its bound. It does not own the harness's ordinary run algorithm (`agent-run-driver`), the tool-scheduler's rejoin (`agent-tool-scheduler`), the permission protocol (`agent-permission-protocol`) or the transcript (`agent-history`); it amends each of those through its own delta in this change's `specs/` tree.

## Coverage

| Charter leaf | Requirements | Scenarios |
|---|---|---|
| AG-14.1 interrupt (`0003:1391-1414`) | `R-CAN-001`…`R-CAN-004` | `S-CAN-001`, `S-CAN-002`, `S-CAN-003`, `S-CAN-004`, plus bites `S-CAN-010`, `S-CAN-011` |
| AG-14.2 shutdown (`0003:1415-1428`) | `R-CAN-005`, `R-CAN-007` | `S-CAN-005`, `S-CAN-007` |
| AG-14.3 bounded wind-down (`0003:1429-1442`) | `R-CAN-006` | `S-CAN-006`, plus bite `S-CAN-012` |
| Cross-cut (package-wide contract row) | `R-CAN-008` | `S-CAN-008` |

The Requirements and Scenarios columns name IDs rather than counts, for the reason the **Allocated IDs** header line records.

**Charter Gherkin → spec** (all five charter scenarios mapped, none reduced):

| Charter scenario | Owning requirement | Scenario(s) |
|---|---|---|
| `0003:1396-1400` "interrupt aborts the turn and keeps the session" | `R-CAN-001` (first Then), `R-CAN-002` (second Then) | `S-CAN-001` + bite `S-CAN-010`; `S-CAN-002` + bite `S-CAN-011` |
| `0003:1402-1405` "interrupt during a suspension aborts it typed" | `R-CAN-003` | `S-CAN-003` |
| `0003:1407-1410` "interrupt is idempotent" | `R-CAN-004` | `S-CAN-004` |
| `0003:1420-1424` "shutdown winds down and then refuses new work" | `R-CAN-005`, `R-CAN-007` | `S-CAN-005`, `S-CAN-007` |
| `0003:1434-1439` "a cancellation-deaf tool cannot hold the run hostage" | `R-CAN-006` | `S-CAN-006` + bite `S-CAN-012` |

## Purpose

AG-13 shipped a run that iterates and no way to stop one honestly: `wrapHarnessFailure` builds `ai.FailureCategoryUnavailable` unconditionally (`harness.go:184-187`), `failRun` emits `RunOutcomeFailed` for every non-nil `Turn` error (`harness.go:202`), `Turn`'s body contains no context observation at all so a provider stream that closes bare is normalized to `ai.FinishReasonStop` (`loop.go:425-426`), and `executeCall` hands `tool.Run` a `context.Background()` (`scheduler.go:462`) so no executing tool can observe cancellation. This capability makes stopping a run a typed, bounded, two-signal operation.

## Requirements

### R-CAN-001 — Two named signals share one propagation mechanism

The package MUST expose exactly the signal vocabulary named here, and MUST name its members rather than count them: an interrupt sentinel and a shutdown sentinel, each an `errors.Is`-matchable package-level error value, plus a typed post-shutdown prompt refusal that wraps the shutdown sentinel, plus a typed detached-call carrier (`R-CAN-006`). No other cancellation sentinel is introduced by this change.

Propagation MUST be by **cancellation cause**, not by a second channel and not by a second context threaded alongside `ctx`. `Run` MUST derive the run's own context once at entry from the caller's `ctx` and hold its cancel-cause function; the signal methods MUST do nothing but invoke that function with the matching sentinel. `Turn`, `Schedule`, `executeCall` and `tool.Run` MUST NOT gain a new parameter — each already receives a `ctx`.

Every site that observes cancellation MUST read the **cause**, not the bare context error. Three sites are named because each is a behavior change, not a refactor: `Turn`'s closed-channel branch (`loop.go:417-427`), which gains its first cancellation observation and MUST take it **before** the `turn.finish == 0` normalization (`loop.go:425-426`); the permission gate's two abort arms (`scheduler.go:639`, `scheduler.go:665`), which today build their typed failure from `ctx.Err()`; and the harness's own iteration boundary, which MUST consult the cause before starting another `Turn` so a cancelled run makes no further provider call.

**Scope line, normative.** Only the two named sentinels receive cancellation-typed outcomes. A bare cancellation of the *caller*'s context, taken without going through the signal methods, MUST keep AG-13's existing routing (`R-RUN-011`) and MUST NOT be reported as interrupted or as shutdown. A deadline is not a signal of this capability (`0003:1373`); none is introduced.

**The two observation points are reached separately, and the scenario below is split for that reason.** A provider stream and a tool call are never simultaneously in flight within one turn: the loop schedules tools only once the stream has ended in a `Completion` (`R-LSK-006`, "one cycle per turn"; `finishContinuationTurn` calls `Schedule` under `t.finish == ai.FinishReasonToolCalls`, `loop.go:493-500`). A Given that asks for both at once is unsatisfiable by construction, so `S-CAN-001` MUST be verified as two independent arms rather than one — and each arm MUST be independently non-vacuous, since a single arm would leave the other mechanism unproven while the scenario still reported green.

#### Scenarios

- **S-CAN-001** — **Charter AG-14.1, scenario 1, first Then.** Verified as two independent arms, each proving one of the two cancellation-observation mechanisms.
  - **Arm A — the provider stream, over a transcript carrying a genuinely open call.** Given a harness-driven run whose provider stream is held mid-stream, and whose transcript was seeded with a tool call left open by an earlier turn, when the interrupt signal fires, then the provider stream is cancelled per the Layer 1 fake's cancellation-fidelity contract (`R-CNF-011`/`R-CNF-012`: the channel closes bare, synthesizing no terminal event), the turn closes aborted with a cancellation-category failure, **orphan synthesis repairs the open call — its entry reports origin `synthesized` while every entry committed before the signal still reports `appended`** (`R-HIS-007`), the run-end event carries the interrupted run outcome with a nil failure, the run's returned error satisfies `errors.Is` against the interrupt sentinel, and `CheckStream` accepts the whole emitted stream **unmodified**. The seeded open call is what makes the synthesis assertion non-vacuous: without it the open set is empty, synthesis is a documented no-op, and an assertion over zero synthesized entries would pass while proving nothing.
  - **Arm B — the in-flight tool.** Given a harness-driven run whose scheduled call is executing a cancellation-observing tool, when the interrupt signal fires, then that tool observes cancellation through the `ctx` it now receives (`R-TLS-013`) and returns rather than running to completion, the transcript closes with **no open call left behind**, and the run ends with the same interrupted outcome and sentinel as Arm A.
  - **Arm B does not assert synthesized origins, and the reason is structural.** A call that reached the scheduler is rejoined and committed synchronously by `finishContinuationTurn` (`loop.go:493-513`) whether it succeeded or failed, so its transcript entry origin is `appended` — the call was resolved, not orphaned. Orphan synthesis (`R-HIS-007`) covers calls that never reached a result at all; demanding `synthesized` origins here would demand an observation this path cannot produce, and would be satisfiable only by breaking the rejoin. The property Arm B actually owns is **no open call survives the wind-down**. Arm B defends it *indirectly*: it asserts every entry's origin is `appended`, which is only reachable when the call was resolved rather than left open. The direct check — `CloseTurn` rejecting an unpaired call (`R-HIS-003`) — is exercised by Arm A and by the shutdown wind-down, not by Arm B. Stated precisely here rather than claiming Arm B asserts it outright, because an indirect defence and a direct assertion fail under different regressions.
  - Neither arm alone certifies the requirement: Arm A exercises `Turn`'s closed-channel cause check and Arm B exercises the threaded tool context, and bite `S-CAN-010` proves Arm A's mechanism is load-bearing rather than incidental. *(Also certifies `R-CAN-002`.)*
- **S-CAN-010** — **(bite)** RED-first. Given a scratch tree in which `Turn`'s cancellation-cause observation at the closed-channel branch is deleted, when `S-CAN-001` runs, then it FAILS reporting a normally completed turn — the bare stream close was normalized to `ai.FinishReasonStop` and the interrupt was recorded as success. RED-recorded BEFORE `S-CAN-001` is GREEN, then reverted.

### R-CAN-002 — An interrupt ends the run, not the harness value

An interrupt MUST wind the run down and MUST NOT end the harness value. The run MUST terminate with the **interrupted** run outcome and, per `RunEnd.validate`'s failure-iff-`Failed` rule (`run_events.go:156-161`), with **no** `*Failure` attached. It MUST NOT terminate as failed and MUST NOT carry `ai.FailureCategoryUnavailable`.

Wind-down MUST be, in order: synthesize orphans over the transcript; close the turn; emit the run-close event through the existing public run-event constructor; return an error matching the firing signal's sentinel. Orphan synthesis is idempotent (`R-HIS-008`), so a turn whose results were already committed synthesizes nothing and the ordering is safe on every path.

**The turn bracket MUST close before the run bracket does.** `CheckStream` rejects a run-close emitted while a turn bracket is still open, so the turn that observed the cancellation MUST emit its own turn-close first, carrying the **aborted** turn outcome and a `*Failure` of the pre-existing category `ai.FailureCategoryCancellation` — mirroring the mid-stream-fatal branch's existing precedent rather than introducing a path of its own. No new `TurnOutcome` member is introduced: `turn_events.go` stays byte-unchanged (`S-LSK-018`). This is the **turn's** bracket only and does not weaken the rule above: the **run's** close still carries no `*Failure`. Recorded here because the two brackets carry deliberately different failure shapes, and a reader who assumes the run's nil-failure rule also governs the turn's close would read the emitted stream as malformed.

After an interrupted run has returned, a subsequent `Run` on the **same harness value** MUST be accepted and MUST be able to complete normally, with the steering queue reopened at `Run` entry (`R-RUN-001` delta). This is the only cross-run behavior this change introduces beyond the terminal shutdown flag of `R-CAN-005`; cross-run transcript state remains AG-21's.

#### Scenarios

- **S-CAN-002** — **Charter AG-14.1, scenario 1, second Then.** Given a harness value whose first run ended through an interrupt as in `S-CAN-001`, when a second `Run` is driven on that same value against a fresh script, then the second run emits its own complete run bracket, ends with the completed run outcome, returns a nil error, accepts a `Steer` call with a nil return during that second run, and `CheckStream` accepts the second run's stream unmodified.
- **S-CAN-011** — **(bite)** RED-first. Given a scratch tree in which the harness's cause-aware routing is reverted so every non-nil `Turn` error again reaches the unconditional failure path (`harness.go:200-207`), when `S-CAN-001` and `S-CAN-005` run, then both FAIL reporting the failed run outcome and a non-nil failure of category `Unavailable` — proving the cancellation carve-out of the `R-RUN-011` delta is what produces the typed outcome. RED-recorded BEFORE those scenarios are GREEN, then reverted.

### R-CAN-003 — An interrupt during a permission suspension aborts it typed

When a signal fires while a call is parked on the permission gate, the parked call MUST resolve into a typed abort whose `*Failure` carries `ai.FailureCategoryCancellation` and whose unwrap chain reaches the firing signal's sentinel by `errors.Is`. The raw `context.Canceled` value MUST NOT be the only thing the failure carries: the typed value MUST name *which signal* aborted the call.

Every guarantee `R-APP-009` already makes MUST continue to hold unchanged: both waits honour cancellation, each releasing call deregisters its own parked entry so a late wake observes `ErrStrayDecision`, the rejoin slice returned by `Schedule` is fully populated in ordinal slots, and no gate goroutine waits forever. The transcript MUST close cleanly afterward — orphan synthesis plus turn close leave no open call.

#### Scenarios

- **S-CAN-003** — **Charter AG-14.1, scenario 2.** Given a run whose permission policy defers a call, and a consumer that has read the decision-required event off the run stream (the stream is the synchronization; no wall clock), when the interrupt signal fires, then the parked call's ordinal slot carries an execution-failure result whose `*Failure` reports category `Cancellation` and whose unwrap chain satisfies `errors.Is` against the interrupt sentinel, a subsequent wake for that call identity returns `ErrStrayDecision`, the transcript reads back with no open call after the wind-down, and `CheckStream` accepts the stream unmodified.

### R-CAN-004 — A second signal changes nothing and panics nothing

A signal fired while a run is already winding down MUST be a no-op with respect to observable behavior: the emitted event stream, the run-end outcome and the returned error MUST be the same as if it had never fired, and no code path may panic. Harness signal state MUST be guarded such that a concurrent second signal is safe under `-race`. No signal may close a channel, so no double-close is reachable by construction.

Where two *different* signals race, the first cause MUST win the run-end outcome and the error chain, and a shutdown that loses that race MUST still latch its terminal refusal flag (`R-CAN-005`).

#### Scenarios

- **S-CAN-004** — **Charter AG-14.1, scenario 3.** Given a run already winding down from an interrupt, when a second interrupt fires from a separate goroutine concurrent with the first's wind-down and the suite runs under `-race`, then the test process reports no panic and no data race, the run-end outcome and the returned error are the same values `S-CAN-001` observed, and the emitted event sequence is accepted by `CheckStream` unmodified.

### R-CAN-005 — Shutdown winds down identically and then terminally refuses new prompts

Shutdown MUST perform the same wind-down as an interrupt (`R-CAN-002`), differing in exactly two observable ways: the run-end event MUST carry the **shutdown** run outcome (`R-CAN-007`), and the returned error MUST satisfy `errors.Is` against the shutdown sentinel and MUST NOT satisfy it against the interrupt sentinel.

Shutdown MUST additionally latch a **terminal, one-way** refusal flag on the harness value. A `Run` invoked after shutdown MUST return the typed post-shutdown refusal (`R-CAN-001`), MUST emit no event whatsoever, and MUST leave the consumer sink in a state the caller can drain to completion. The flag MUST never resume a run and MUST hold no transcript: it is per-value terminal bookkeeping and does not pre-empt AG-21's cross-run state (`agent-run-driver`'s `R-RUN-001`, its one-run-at-a-time clause).

The two signals MUST stay distinguishable **through both nouns of `0003:1379`** — the run-end outcome on the stream and the Go error chain — so a consumer that reads only the stream, such as a Layer 3 TUI holding no Go error, can still tell them apart.

#### Scenarios

- **S-CAN-005** — **Charter AG-14.2.** Given a run in flight with the same shape `S-CAN-001` uses, when the shutdown signal fires, then the wind-down produces the same synthesized-origin closures and the same nil-failure run-close shape, the run-end outcome value is the shutdown member and is different from both the interrupted and the failed members, the returned error satisfies `errors.Is` against the shutdown sentinel and fails it against the interrupt sentinel; and when a second `Run` is then invoked on that same harness value, then it returns the typed post-shutdown refusal, that error satisfies `errors.Is` against the shutdown sentinel, the consumer observes no event at all, and `CheckStream` accepts the first run's stream unmodified.

### R-CAN-006 — Wind-down is bounded, and the overrun call is reported typed

Wind-down MUST complete within a **documented bound**. The bound MUST be a package constant serving as the default — `100ms`, chosen an order of magnitude above goroutine-scheduling jitter under `-race` (`agenttest`'s own settle window is 50 ms, `stream_kit_leak.go:77`) — and MUST be overridable through a zero-default field on the caller-owned `Scheduler` value (`agent-tool-scheduler` delta), so tests can inject a smaller bound through the continuation seam without new plumbing.

The bound MUST be **armed only by cancellation**. On an uncancelled path no timer may be created and the join MUST stay exactly as unbounded as it is today, so `R-RUN-010`'s "no third path and no timeout" (`agent-run-driver/spec.md:200`) remains literally true for a run nobody cancelled, and `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` (`permission_protocol_test.go:1573-1600`) MUST pass with its source **file-unchanged**.

A call still running when the bound expires MUST be reported **typed**, naming the tool, the call identity, and the fact that it is still running. The report MUST ride the **existing execution-failure path**: the call's ordinal slot carries an execution-failure `Result` whose `*Failure` reports `ai.FailureCategoryCancellation` and whose cause is the detached-call carrier, `errors.As`-extractable through the failure's preserved unwrap chain. **No new `EventKind` may be registered** and no new `Result` outcome may be introduced — the stream carries the existing `tool_end_execution_failure` kind.

**"Detached and named" — the scope distinction, stated rather than left implicit.** The charter's second and third Thens (`0003:1438-1439`) are simultaneously true because they range over **disjoint sets**:

| Set | Members | Obligation |
|---|---|---|
| Third-party code | the tool's own `tool.Run` frame | MAY still be running after the bound. Go has no goroutine-kill primitive, so the harness cannot end it. It is **named** by the typed report — tool name plus call identity — never silently abandoned, and it MUST be able to complete its result send and exit whenever the third-party code returns, without writing to any structure the harness has since closed. |
| Harness-owned tasks | `runDispatcher` (`scheduler.go:267`), the per-turn event forwarder (`harness.go:274-279`), each `scheduleRead`/`scheduleSerialized` call goroutine **including the wrapper that runs the detach select**, and `Run`'s own control flow | MUST all exit within the bound. The wrapper is harness-owned and exits at the bound even when the tool's own frame does not; the frame `tool.Run` occupies is the only task in neither row's scope, which is what keeps the two rows disjoint. Every writer of the rejoin slice and of the emissions channel MUST remain a harness-owned goroutine that provably exits, so abandoning the join is forbidden: it would race the still-running call's later slot write and could send on a closed emissions channel. |

Panic containment (`R-TLS-011`) MUST be preserved: a panic raised by third-party code after detachment MUST NOT crash the process, and a panic raised before it MUST still surface as the typed execution failure `R-TLS-011` already requires.

#### Scenarios

- **S-CAN-006** — **Charter AG-14.3, all three Thens.** Given a run whose scheduled call uses a cancellation-deaf scripted tool blocking on a channel the test never closes during the run, and a caller-owned scheduler injected with a small wind-down bound, when the interrupt signal fires, then the run **returns** — observed by a read on the run's completion channel, never by a wall-clock assertion; and the stream carries a `tool_end_execution_failure` for that call whose `*Failure` `errors.As`-extracts a detached-call value reporting that tool's name and that call's identity; and when the same scenario is repeated under the package's goroutine-leak harness, each iteration releasing its own blocked tool after the run returned, then no harness-owned goroutine survives the wind-down and the released third-party goroutine is **accounted for** — proven alive past the bound by the typed report and proven to exit once the third-party code returns — rather than excluded from the count. The leak-harness test MUST NOT call `t.Parallel()` (`stream_kit_leak.go:80`, `:110`).
- **S-CAN-012** — **(bite)** RED-first. Given a scratch tree with the armed bound removed so the wind-down waits on the deaf tool unconditionally, when `S-CAN-006` runs, then it does not return and the failure is evidenced by `go test -timeout` reporting the hung test — proving the bound, not some other mechanism, is what ends the run. RED-recorded BEFORE `S-CAN-006` is GREEN, then reverted.

### R-CAN-007 — The shutdown run outcome is a stream-level discriminator

The run-outcome vocabulary MUST gain one member, the shutdown outcome, so a stream consumer holding no Go error can distinguish shutdown from interrupt. The addition MUST be confined to the member and its `String()` rendering, `"shutdown"`. The pre-existing members MUST keep their current values — `RunOutcomeCompleted`/`RunOutcomeInterrupted`/`RunOutcomeFailed` at 1/2/3 (`run_events.go:100-118`) — which the insertion before `runOutcomeLimit` achieves.

`RunEnd.validate`'s failure-iff-`Failed` rule (`run_events.go:156-161`) and `NewRunEnd`'s signature (`run_events.go:172`) MUST NOT change: both the interrupted and the shutdown outcomes are non-`Failed` and therefore carry no `*Failure`, so the discriminator rides the outcome member itself rather than an attached payload. `stream_check.go` MUST be byte-unchanged; `CheckStream` delegates outcome membership to `RunEnd.validate`'s own bound check (`run_events.go:153`), which admits the new member once the limit moves. The substrate release authorising the edit is recorded in the `agent-loop-skeleton` delta of this change (`R-LSK-004`).

#### Scenarios

- **S-CAN-007** — Given the run-outcome vocabulary after this change, when the shutdown member is rendered, then it reads `"shutdown"`; and when a run-close event is constructed with the shutdown outcome and no failure, then construction succeeds; and when one is constructed with the shutdown outcome and a non-nil failure, then it is rejected with the misplaced-failure rule class the validator already applies to every non-`Failed` outcome; and when `CheckStream` is run over a complete, otherwise-valid stream ending in a shutdown run-close, then it accepts the stream with `stream_check.go` byte-unchanged.

### R-CAN-008 — `L2C-08`: cancellation is a package-wide contract row

AG-12's discriminator — does the milestone declare a new package-wide guarantee, or implement behavior inside one already declared? — resolves **yes** here, and the answer is recorded rather than left to silence. `L2C-03`/`L2C-04` govern the upward stream and `L2C-07` the transcript; nothing existing governs **downward control or goroutine lifetime**. Bounded wind-down and two-signal distinguishability bind every future tool and turn family, including AG-19's subagents and AG-21's leak sweep.

The package documentation MUST therefore carry an `L2C-08` row matching the guard's grammar, and its `expectedLayer2ContractRows` entry MUST land in the **same pull request**, per the guard's own rule (`doc_contract_guard_test.go:19-22`). The row's normative content MUST state: cancellation is a bounded, typed, two-signal tree; interrupt aborts the run and keeps the harness; shutdown aborts the run and terminally refuses new prompts; both propagate down through loop, provider and tools as one context cancellation cause; both stay `errors.Is`-distinguishable in the error chain and distinct in the run-end outcome; after the documented wind-down bound only third-party tool code may remain running, reported typed by tool and call identity, and every goroutine the package itself owns has exited.

#### Scenarios

- **S-CAN-008** — Given the package documentation and its `expectedLayer2ContractRows` expectation table after this change, when the doc-contract guard runs, then it passes with the `L2C-08` row present in both and matching the guard's row grammar, with every pre-existing row byte-unchanged and none removed or reworded — the guard is the verifier, not this spec's prose. *(ID note: `S-CAN-008` is outside the design's Testing Strategy assignments, which fix `S-CAN-001`…`S-CAN-007` and the bites `S-CAN-010`/`011`/`012`; `008` was free and no assigned ID is reused.)*

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-CAN-001** | External-package verifiability: every scenario above MUST be verifiable by `cd backend/agent && make test`, and every behavioral test MUST live in `package agent_test`. A behavior reachable only from inside the package is, for this spec, not reachable at all. |
| **NFR-CAN-002** | Determinism and race cleanliness: every test MUST be hermetic and MUST pass under `-race`, including repeated runs per `NFR-APP-002`'s discipline where raciness applies. Synchronization MUST be by `agenttest.Gate`, channel reads and channel closes. **The wind-down bound is the only legitimate use of clock time in this change**; no test may synchronize by sleep, timeout or wall-clock ordering, and a test that needs a sleep to pass is a design failure. |
| **NFR-CAN-003** | Ambient authority: production sources added by this change MUST NOT import `os`, process execution, `syscall` or legacy I/O; the ambient-authority guard (`ambient_authority_test.go:73-94`) MUST pass with **zero** change over both closures, as MUST the import-boundary guard. `time` is not in the forbidden set and is used only for the bound. |
| **NFR-CAN-004** | Coverage: line coverage on `backend/agent/src/agent/loop.go` MUST remain ≥ 80% under `make test`, including the new cancellation branch (`R-LSK-005` carry). |
| **NFR-CAN-005** | Review budget: this change ships as a **single** pull request under a pre-authorised `size:exception` extended for AG-14 beyond the 1000-line budget. The pull-request description MUST state why the change does not fit the default budget. |

## Explicit non-requirements — what this spec does NOT claim

Stated so that no test, guard or acceptance line is written as if AG-14 closes more than it does.

| Not claimed | Owner |
|---|---|
| Subagent cancellation inheritance | **AG-19.2.** Charter "Out of scope" line (`0003:1381`); no subagent tool ships in v1 (`0003:1794`) |
| A keypress, `SIGINT` or any other frontend signal reaching the harness | **Layer 3.** AG-14 defines the mechanism; the composition root calls it |
| Retry or failover on a cancelled turn | **AG-15.** A cancellation is never retried; `R-RUN-011`'s no-retry rule extends verbatim to the cancellation path. **CONFIRMED CLOSED by AG-15, and closed by mechanism rather than by promise.** AG-15 keeps the existing cause check at its existing site as gate **G0** of `R-RTY-001`'s ordered, first-match-wins predicate, evaluated **before** G1–G5, so no cancelled turn can reach a retry, a backoff wait, or the failover consult. AG-15 adds no second cancellation check, moves no existing one, and introduces no cancellation vocabulary. A cancellation arriving **during a backoff wait** is the one new arrival point, and it routes to the same wind-down (`R-RTY-008`) with the same interrupted-or-shutdown run outcome and the same nil `*Failure` — one more way *into* the wind-down, never a way out of it. A bare cancellation of the caller's own context, matching neither sentinel, keeps AG-14's scope-line routing and is likewise never retried: it reaches the failure surface through the untyped or non-retryable gates, not through the retry gate |
| Cost accounting for an interrupted turn | **AG-16** |
| Compaction cancellation-safety | **AG-17 / AG-18** |
| The package-wide goroutine-leak sweep | **AG-21** (`0003:2178`). AG-14 proves only its own goroutines and makes the detached-call report precise enough for that sweep (`0003:1439`) |
| A **deadline** as a third signal | **Not this milestone.** `0003:1373` is a distinctness claim, not a mandate; AG-14.3's bound is a wind-down bound, not a run deadline |
| Concurrent runs on one harness value | **Not this milestone.** `R-CAN-002` re-scopes the rule to *one run at a time*, not to concurrency |
| Cross-run transcript state beyond the terminal shutdown flag | **AG-21** (`agent-run-driver`'s `R-RUN-001`, its one-run-at-a-time clause) |
| A new `EventKind`, a new `TurnOutcome`, or a new exported `History` method | **Not this milestone**, and forbidden under this change |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2.** Layer 1 is consumed, never edited |
