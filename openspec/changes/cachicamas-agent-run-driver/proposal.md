# Proposal: AG-13 — Drive the multi-turn run

> **Change**: `cachicamas-agent-run-driver` · **Milestone**: AG-13 (Layer 2 Wave 3, milestone 13 of 24; doc 0003 lines 1294–1370)
> **Branch**: `feat/agent-layer2-wave3-ag13` · **Artifact store**: hybrid (Engram + filesystem)
> **Pre-authorized**: `size:exception` against the 1000-line PR review budget
> **TDD**: strict, RED-first (`cd backend/agent && make test`)
> **Closes**: R-08's driving loop; consumes R-09 · **Blocks**: AG-14, AG-15, AG-16, AG-17, AG-19, AG-20
> **Exploration**: `exploration.md` · Engram `sdd/cachicamas-agent-run-driver/explore` (#3162)

## Intent

Twelve milestones have built the parts of a conversation and none of them holds one. `Turn` runs exactly one model→tools→finalize cycle and stops (`R-LSK-006`, `agent-loop-skeleton/spec.md:83`: "the loop schedules, it does not iterate — iteration across cycles is **AG-13's `Harness` contract**"). `History` stores a transcript and wires into nothing (`agent-history/spec.md:185`: "wiring history into `Turn`/`Schedule` is AG-13's"). AG-11 makes a pause visible and refuses to act on it (`agent-turn-termination/spec.md:146`). AG-10 parks a deferred call and deliberately exposes no wake handle (`agent-permission-protocol/spec.md:144`: "`Turn()` deliberately exposes no scheduler handle and no wake surface at AG-10").

AG-13 is where those five deferrals come due at once. It is the harness's core: append the user message, run a turn, execute and append results through the loop, repeat until a terminal finish reason — one run bracket, N turn brackets, one contiguous event story.

**This is not composition.** The exploration found two genuine gaps that no amount of careful assembly closes:

1. `Turn` mints a fresh `RunID`/`TurnID` per call and unconditionally brackets each call with `RunStart`/`RunEnd` and a per-call `LaneStamper` (`loop.go:141-151, 193-198, 666-681`). `CheckStream` accepts N turn brackets inside one run bracket — neither `turn_start` nor `turn_end` carries `CardinalityAtMostOne` (`event.go:253-257`) and only `run_end` is `Terminal` (`event.go:249`) — but `BracketRoleOpensRun` rejects a second `run_start` with `ErrDuplicate` (`stream_check.go:122-127`) and the sequence check demands one contiguous 1-based lane. Calling `Turn` N times and concatenating fails on both counts.
2. `Turn` builds its `*Scheduler` locally (`loop.go:259`) and never exposes it. `Scheduler.WakeParked` (`scheduler.go:235`) is the only production release for a `PermissionDefer` verdict, and `parkedSet` lives exactly one `Schedule` call — so a suspension must resolve before that `Turn` returns or the call blocks forever. The charter's fourth acceptance clause ("the run survives a permission suspension spanning a turn") is unreachable without an injection seam. `loop.go:260-264` already names this as AG-13's scope.

**What closes**: R-08's driving loop, plus consumption of R-09. **What unblocks**: AG-14 (cancellation), AG-15 (retry/failover), AG-16 (cost), AG-17 (context strategy), AG-19 (delegation), AG-20 (hook taxonomy). **What ticks**: doc 0003 line 2170 — "A multi-turn run completes with steering, pause resumption, and a complete event story — closed by AG-13."

## Scope

### In

- **AG-13.1** — the run driver: a value-form harness over `Turn`, permission protocol, termination, and history. Drives a scripted two-call-then-answer conversation to its terminal, appending the user message, running the turn, executing and appending results **via the loop**, and repeating.
- **AG-13.1** — **one run bracket, N turn brackets, one contiguous sequence**, accepted by `CheckStream` unmodified. The consumer observes run start, turn one with tool execution and result append, turn two, and run end.
- **AG-13.1** — history holds the full alternating transcript with every pair matched, via the existing `Append` / `CloseTurn` / `Entries` routes only. **No new exported `History` method** — the `history_surface_guard_test.go` closed-route guard is set-equal in both directions.
- **AG-13.1** — the run-scope reconstruction property: every message and tool outcome the history holds is reconstructable from the replayed stream, **asserted at run scope**, not per turn. This extends `L2C-04`/`L2C-05` to the multi-turn story rather than declaring a new guarantee.
- **AG-13.1** — the no-privileged-channel proof: the harness reaches the loop through the same public one-turn surface the skeleton's external tests use. Tests live in external `agent_test`; an in-package test would prove nothing.
- **AG-13.2** — a steering queue. A user message arriving mid-turn queues, the in-flight turn completes untouched, and the message is appended at the turn boundary **before the next provider call**. Consecutive same-role entries are legal — `commitAppendOp` carries no role-alternation check, so History already satisfies this and no History change is needed.
- **AG-13.2** — arrival-order preservation with **zero drops**, including a message queued during the final turn, which yields a new turn rather than a dropped message. Synchronization is by `agenttest.Gate`, never wall clock.
- **AG-13.3** — pause resumption. A turn returning `ai.FinishReasonPauseTurn` resumes with the returned partial `ai.Message` replayed **verbatim** — appended and re-included in the next request's transcript, not discarded or re-synthesized — and the run continues to a real terminal.
- **AG-13.3** — the pause is visible on the stream as `TurnOutcomePaused`, its own turn outcome, not silently absorbed. `outcomeForFinish` (`loop.go:331-350`) already maps it; no new `TurnOutcome` and no new `EventKind`.
- **Fourth acceptance clause** — the run survives a permission suspension spanning a turn: a `PermissionDefer` verdict parks, the driver wakes it through a live `Scheduler` handle, and the run completes.
- The `agent-permission-protocol` **known gap carried to AG-13** (`spec.md:172`): the `R-APP-002` acknowledgement has no non-vacuous guard today. The wake seam makes the missing bite constructible — it must observe the parked **wait**, not the registration.
- Substrate filter widening in `loop_test.go` (and `loop_hook_test.go` if it carries the twin filter) by **exact filename**, no wildcards, kept byte-in-sync — the AG-11/AG-12 discipline.

### Out — deferred, with the owner named

| Deferred | Owner and why deferral is safe |
|---|---|
| **Strengthening `CheckStream` with cross-event RunID-value consistency** | **AG-19.** Two independent reasons. (a) `stream_check.go` is a named `R-LSK-004` substrate file; touching it breaks a five-milestone byte-unchanged streak for a check nothing in AG-13 needs. (b) AG-19.1's scenario — "the child's events each parent-identified… a consumer separates the two conversations by walking parents" (`0003:1820-1821`) — is the first place identity-value consistency is *load-bearing* and mechanically forced. Safe because AG-13 **does not exploit the gap**: the driver emits one honest RunID for the whole run, and the design must state that the property is upheld by construction and asserted in AG-13's own tests even though the validator does not yet enforce it. |
| Retry / failover on a failed turn | **AG-15.** Charter "Out of scope" line, binding. |
| The compaction check between turns | **AG-17** inserts it; **AG-18** implements compaction. AG-13's turn boundary is the seam it will be inserted at, and nothing here anticipates its shape. |
| Cancellation semantics — interrupt vs. shutdown vs. deadline | **AG-14.** AG-13 propagates `ctx` and stops; it defines no cancellation vocabulary. |
| A production subagent tool, nested runs | **AG-19.** No subagent tool ships in v1 (`0003:1794`). |
| Persistence / session reload of a run | **Layer 3.** The harness holds state in memory; it never touches a file (`0003:110`). |
| A real provider or a real tool | **Never in Layer 2.** `agenttest` scripts only; a Layer 2 test that opens a socket or a file is a boundary violation (`0003:123`). |
| Cost aggregation across turns | **AG-16.** |
| `ToolSource` port widening | **Flagged, not taken.** `agent-tool-scheduler/spec.md:138` names AG-13 as the widener; the AG-13 charter does not mention tool sources. `sdd-spec` must either re-home that line by delta or record why it stays. Not implemented under this change. |
| Multi-turn *state* beyond the run | **AG-21** (`agent-loop-skeleton/spec.md:106`). AG-13 owns one run's iteration, not cross-run state. |
| Any edit under `backend/agent/src/ai/**` | **Not this milestone.** Layer 1 is consumed, never edited. |

## Open design decisions — framed here, resolved by `sdd-design`

### Decision 1 — run/turn identity continuity across N turns

**Question**: how does a driver get one run bracket, N turn brackets, and one contiguous 1-based sequence out of a `Turn` that today mints fresh identity and brackets unconditionally per call?

| Option | Tradeoff |
|---|---|
| **A — extend `TurnOptions`** with an optional caller-supplied run identity, a continuation flag, and a caller-supplied `LaneStamper`; make `RunStart`/`RunEnd` emission conditional on them. All fields nil/zero-default. | **Exploration's recommendation.** `TurnOptions` already grew across AG-08/09/10 without changing `Turn`'s positional signature — the documented contract at `loop.go:11-14` ("AG-13's later Harness can wrap it without changing the signature") is preserved literally. Cost: conditional emission must be re-proven against **every** existing `loop_test.go` scenario that asserts unconditional `RunStart`/`RunEnd`. |
| **B — change `Turn`'s positional signature** to accept identity directly. | Breaks the documented no-signature-change contract and `S-LSK-011`'s "`Turn`'s exported signature is unchanged" assertion. Rejected unless A proves impossible. |
| **C — driver-side event rewriting**: call `Turn` unchanged, strip inner run brackets, re-synthesize an outer one. | Requires no loop change, but `Event`'s fields are unexported with no rewrite door, so the driver must reconstruct every payload through its own constructors — destroying identity provenance and defeating AG-13.1's "no privileged channel" scenario outright. **Not recommended.** |
| **D — the harness calls `Scheduler.Schedule` directly**, bypassing `Turn`. | `R-LSK-006` explicitly permits it ("callable from `Turn` (AG-09) or from `Harness` (AG-13)"), but it contradicts AG-13.1's third scenario, which requires the harness to go through the **public one-turn surface**. The two spec statements are in tension; `sdd-design` must reconcile them in writing, not silently pick one. |

**Binding constraint on whichever option wins**: `R-LSK-002` / `S-LSK-004` (`TestTurn_TwoSequentialTurnsShareNothing`, `loop_test.go:960`) must stay green **for the nil-default path**. Note the precise scope this leaves: `S-LSK-004`'s Given is "fresh slices, fresh `opts`, fresh `sink`, fresh `provider` script" — the invariant is about two *independent* invocations with *fresh opts*. A continuation-opts invocation is a different scenario, not a violation. `sdd-design` should record that reading explicitly rather than let a reviewer read the new behavior as breaking a locked-in test. If the reading is wrong, the amendment needs conscious sign-off, not a quiet edit.

### Decision 2 — the `Scheduler` / `WakeParked` reachability seam

**Question**: how does the driver obtain a live handle on the `*Scheduler` that a given `Turn` invocation is blocked inside, so it can call `WakeParked` before that invocation returns?

| Option | Tradeoff |
|---|---|
| **A — inject a caller-owned `*Scheduler` through a new `TurnOptions` field**, nil default constructing today's local scheduler byte-stably. | Exploration's recommendation. Symmetric with Decision 1 option A — one extension mechanism, not two. Cost: it changes `Turn`'s internal `Schedule` call site, so every AG-09/AG-10 scheduler test must stay green untouched. |
| **B — return the scheduler** from `Turn`. | Useless: the handle is needed **while** `Turn` is blocked, not after it returns. |
| **C — a wake channel or callback passed through `TurnOptions`** instead of the scheduler pointer. | Narrower surface (the driver gets a wake door, not the whole scheduler), but adds a second concept where `Scheduler` already is the concept. Worth weighing against A on surface-area grounds. |
| **D — the driver owns the scheduler and calls `Schedule` itself** (Decision 1 option D). | Same reconciliation problem with AG-13.1 scenario 3. |

**Binding constraint**: `parkedSet` lives exactly one `Schedule` call. Any suspension must resolve — wake or `ctx.Done()` — before that `Turn` returns. The design must state what happens if it does not, and the test must prove the resolution by synchronization point, never by timeout.

## Capabilities

### New

- **`agent-run-driver`** — the multi-turn run driver: run iteration to a terminal finish reason, one-run/N-turn event bracketing, history wiring at turn boundaries, steering queueing, pause resumption, and permission-suspension survival. IDs `R-RUN-0NN` / `S-RUN-0NN`, bites `S-RUN-0NN`. Prefix `RUN` verified free: zero `[RSN]-RUN-[0-9]` matches repo-wide. Becomes `openspec/specs/agent-run-driver/spec.md` at archive.

### Modified — deltas required

| Capability | What changes |
|---|---|
| `agent-loop-skeleton` | The `TurnOptions` extension and conditional bracket emission are loop-skeleton surface. `R-LSK-002`/`S-LSK-004` needs its scope stated (nil-default path). Non-requirement lines "Value-form `Harness` — AG-13" (`:105`) and "Iteration of the model ↔ tools ↔ model cycle — AG-13" (`:107`) need back-annotation. `R-LSK-006` ("`Turn` MUST call `Schedule` at most once per invocation") stays **true** and must be restated as still-true, not silently assumed. |
| `agent-permission-protocol` | `:144` "The upward-path wake wired into `Turn` — AG-13" is closed here. `:172`'s known gap — the `R-APP-002` non-vacuous bite observing the parked wait — becomes constructible and is claimed by this change. |
| `agent-history` | `:185` / `:201` / `:206` — "`loop.go` and `scheduler.go` MUST be byte-unchanged: wiring history into `Turn`/`Schedule` is AG-13's" and the steering-queueing deferral row. Back-annotation recording that AG-13 did the wiring; the AG-12 statement was scoped to AG-12, not a standing prohibition. |
| `agent-turn-termination` | `:65` / `:146` "Acting on a pause — performing the resumption — is AG-13's". Closed by AG-13.3. |
| `agent-tool-scheduler` | `:133` "AG-13 iterates" back-annotation; `:138`'s `ToolSource` widening line re-homed or justified (see Out table). |

**No `L2C-08` doc-contract row.** Applying AG-12's own discriminator — "does this milestone declare a new package-wide guarantee, or implement behavior inside one already declared?" — AG-13 declares none. "No privileged channel into the loop" *is* `L2C-03` verbatim, extended to multi-turn; run-scope reconstruction *is* `L2C-04`/`L2C-05` at a wider scope; history is already `L2C-07`. *Criterion to overturn in `sdd-design`*: if the one-run/N-turn bracketing turns out to be a package-wide constraint on every future event family rather than behavior of one type, add the row and say so. Note that `doc.go` and `doc_contract_guard_test.go` are `R-LSK-004` substrate files — adding a row is a substrate edit and needs the justification to clear that bar too.

## Approach — exploration Approach 1

A value-form harness in a new file under `backend/agent/src/agent/`, driving `Turn` in a loop:

1. Append the user message to `History`; drain the steering queue in arrival order before building the request transcript.
2. Call `Turn` with continuation options carrying the run identity and lane stamper (Decision 1) and the caller-owned scheduler (Decision 2), forwarding events to the consumer sink unchanged.
3. Dispatch on the returned finish reason: tool calls → the loop already executed and appended results, continue; pause → append the partial message verbatim, continue; terminal → close.
4. `CloseTurn` at each boundary; emit the run bracket exactly once, around the whole iteration.

The driver adds no new `EventKind`, no new `TurnOutcome`, no new exported `History` method, and no new Go dependency. Everything it needs already exists; what it needs is two nil-default seams into `Turn`.

**Rejected — driver-side event rewriting** (Decision 1 option C): destroys identity provenance and contradicts AG-13.1 scenario 3.
**Rejected — a `Harness` interface**: the second implementation does not exist. Every concrete boundary type in `agent/` from AG-04 to AG-12 is a struct until a second implementation actually arrived.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/harness.go` (name TBD by design) | **New** | The run driver, the steering queue, the turn-boundary history wiring, pause resumption |
| `backend/agent/src/agent/harness_test.go` (`package agent_test`) | **New** | AG-13.1's three scenarios, AG-13.2's two, AG-13.3's one, the permission-suspension clause, the run-scope reconstruction assertion, and the `R-APP-002` parked-wait bite |
| `backend/agent/src/agent/loop.go` | **Modified** | `TurnOptions` extension; conditional `RunStart`/`RunEnd`; caller-supplied lane stamper; injected scheduler at the `Schedule` call site (`:259`) |
| `backend/agent/src/agent/scheduler.go` | **Modified (possibly)** | Only if the wake seam needs it; `WakeParked` itself already exists at `:235` |
| `backend/agent/src/agent/loop_test.go` | **Modified** | Substrate filter widened by exact filename; existing unconditional-bracket assertions re-scoped to the nil-default path |
| `openspec/specs/agent-loop-skeleton`, `agent-permission-protocol`, `agent-history`, `agent-turn-termination`, `agent-tool-scheduler` | **Delta** | Five back-annotation / scope deltas |
| `docs/architecture/milestones/0003-…md:2170` | **Modified** | Completion-checklist line ticked |
| `stream_check.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `turn_events.go`, `history.go`, `doc.go`, `doc_contract_guard_test.go`, `event_registry_test.go`, `reconstruction_test.go`, `go.mod`, `go.sum`, `backend/agent/src/ai/**` | **NOT TOUCHED** | No `R-LSK-004` substrate edit, no new event kind, no History surface change, no Layer 1 edit, no new dependency |

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | `R-LSK-002` / `TestTurn_TwoSequentialTurnsShareNothing` (`loop_test.go:960`) is a locked-in, tested invariant preserved through AG-08..AG-12; conditional bracketing is read as breaking it | High | The nil-default path keeps it literally green, and `S-LSK-004`'s Given already scopes it to *fresh `opts`*. `sdd-design` must record that reading in prose; `sdd-verify` re-runs the test unmodified. Any actual amendment needs conscious sign-off, not a quiet edit. |
| 2 | Scheduler injection changes `Turn`'s internal `Schedule` call site; an AG-09/AG-10 scheduler test regresses | Med | Nil default constructs today's local scheduler byte-stably. Every existing scheduler test stays **file-unchanged** and green; `sdd-verify` asserts that, not just a passing suite. |
| 3 | The review budget: forecast 1600–2400 lines against 1000 | High (certain) | `size:exception` pre-authorized by the user for this milestone. See the forecast below. If `sdd-tasks` forecasts above ~2400, slice at the leaf boundary — AG-13.1 is independently deliverable; AG-13.2 and AG-13.3 both depend only on it. |
| 4 | `CheckStream`'s missing cross-event RunID-value consistency check is a latent gap the driver could silently pass through | Med | Deferred to **AG-19** with two stated reasons (substrate + first load-bearing use). AG-13 does **not** exploit it: the driver emits one honest RunID and AG-13's own test asserts identity-value consistency across the whole run, so the property is proven here even though the validator does not enforce it. |
| 5 | Decisions 1D and AG-13.1 scenario 3 are in genuine spec tension — `R-LSK-006` permits the harness to call `Schedule` directly, the charter requires it to go through the public one-turn surface | Med | Named as a reconciliation obligation on `sdd-design`, in writing. This is exactly the parallel-phase divergence class this repo has been bitten by; both `sdd-spec` and `sdd-design` must land on the same reading. |
| 6 | A suspension that resolves neither by wake nor by `ctx.Done()` deadlocks `Turn` forever (`parkedSet` lives one `Schedule` call) | Med | The design must state the resolution contract. Every suspension test synchronizes through `agenttest.Gate`; a test that would need a timeout to pass is a design failure, not a flaky test. |
| 7 | Forgetting the new filenames in the `loop_test.go` substrate filter (and its twin, if present) | Med | AG-11's and AG-12's recorded lesson. Exact-filename suffixes only, no wildcards, both filters byte-in-sync. |
| 8 | Five spec deltas is more back-annotation surface than any prior Layer 2 milestone; one gets missed and a spec goes stale | Med | Enumerated exhaustively above with file and line. This repo's known staleness clusters are owning-spec omission and un-back-annotated merges — both are in play here. `sdd-verify` re-reads each cited line against the shipped change. |
| 9 | The run-scope reconstruction assertion is written per-turn and passes vacuously | Med | AG-05's W1 vacuous-helper failure mode. The assertion must be **run-scope** and needs a bite: rewrite the run's event sequence to drop a turn-two event and prove the reconstruction diverges — mirroring `S-LSK-003a`/`S-LSK-003b`. |
| 10 | `TestPermission_*` and the `R-APP-002` bite: a bite that observes registration rather than the parked wait re-encodes the very gap it closes | Med | `agent-permission-protocol/spec.md:172` states the requirement verbatim — "the missing bite must observe the parked **wait**, not the registration". Quote it in the spec, do not paraphrase. |

## Rollback Plan

Single revert of the AG-13 merge commit. The new harness file and its test are deleted; `loop.go` returns to unconditional bracketing with a locally constructed scheduler and `TurnOptions` loses its new fields; `scheduler.go` returns to its AG-10 state; `loop_test.go`'s substrate filter returns to its pre-AG-13 filename list and its bracket assertions to their unconditional form; the five spec deltas are dropped; doc 0003's checklist line un-ticks.

The revert is clean because every seam is **nil-default and additive**: no consumer of `Turn` outside this change passes the new options, so the pre-AG-13 behavior is exactly what the nil path already does. No migration, no data, no `go.mod`/`go.sum` change, nothing outside `backend/agent`. Re-running `cd backend/agent && make test` at the parent commit confirms zero regression.

Forward-looking cost: AG-13 blocks six milestones (AG-14, AG-15, AG-16, AG-17, AG-19, AG-20). A revert re-blocks all six. That is a scheduling consequence, not a correctness one.

## Review-workload forecast

| Component | Estimate |
|---|---|
| Harness production file (this package's doc density) | 250–400 |
| `loop.go` — `TurnOptions` fields, conditional emission, injected scheduler | 80–150 |
| `scheduler.go` — wake seam, if needed | 0–60 |
| Steering queue (own file or inside the harness) | 60–100 |
| Harness test file — 6 charter scenarios + permission-suspension clause + run-scope reconstruction + bites | 500–750 |
| `loop_test.go` — filter widening + re-scoped bracket assertions | 40–100 |
| Doc 0003 checklist tick | ~2 |
| **Production + test subtotal (Go)** | **930–1560** |
| SDD markdown — proposal, spec, **5 spec deltas**, design, tasks, apply-progress, verify-report | **700–1100** |
| **Total (authored, additions + deletions)** | **1630–2660** |

`Decision needed before apply: No` — `size:exception` is **PRE-AUTHORIZED** by the user for this milestone, recorded here, one PR. `Chained PRs recommended: No` (unless `sdd-tasks` forecasts above ~2400, in which case slice at the AG-13.1 / AG-13.2+13.3 leaf boundary). `400-line budget risk: High` — and equally high against the raised 1000-line budget.

The SDD markdown counts toward the attempt budget. `sdd-tasks` must forecast against the **full** diff, not the Go diff.

## Dependencies

- **AG-10** (archived) — `PermissionPolicy`, `PermissionVerdict`, `PermissionDefer`, `Scheduler.WakeParked`, the parked-call lifecycle.
- **AG-11** (archived) — `TurnOutcomePaused`, `outcomeForFinish`, the finish-reason dispatch, the AG-11 return contract on the fatal path.
- **AG-12** (archived, merged `5590afa0`) — `History`, `NewSeededHistory`, `Append`, `CloseTurn`, `SynthesizeOrphans`, `Entries`, the pairing invariant, the closed-route surface guard.
- **AG-09 / AG-07** — `Turn`, `Scheduler`, `Schedule`, `Tool`, `Result`, `LaneStamper`, `CheckStream`.
- **AI-21 / AI-22** — `agenttest.NewProvider(scripts ...Script)` (multi-script, one per `Turn`), `Script`/`Step`/`Emit`, `Gate{Reached, Release}`.
- **doc 0003:1294-1370** — the AG-13 charter and its three Gherkin leaves; **doc 0003:114-137** — the evidence gate (`make test` in `backend/agent/`, `go test -race -v ./...`) and the test-substrate binding.

## Acceptance criteria — restated as verifiable checks

- [ ] `cd backend/agent && make test` green with `-race`; all six charter scenarios closed with recorded evidence
- [ ] A scripted two-call-then-answer conversation completes: the consumer observes **run start → turn one (tool execution, result append) → turn two → run end**, and `CheckStream` accepts the stream **unmodified**
- [ ] The stream carries **exactly one** `run_start`/`run_end` pair, **N** turn brackets, and one contiguous 1-based lane sequence across the whole run
- [ ] Every event in the run carries the **same** `RunID` value — asserted by AG-13's own test, not delegated to `CheckStream`
- [ ] History holds the full alternating transcript with every tool call matched to its result
- [ ] Run-scope reconstruction: every message and tool outcome in history is reconstructable from the replayed stream, **plus a bite** proving the assertion is non-vacuous (drop a turn-two event → reconstruction diverges), RED-recorded before GREEN
- [ ] The harness reaches the loop only through the public one-turn surface; its tests are in external `agent_test`
- [ ] A user message arriving mid-turn queues; the in-flight turn completes untouched; the message is appended **before the next provider call**; consecutive same-role entries are accepted
- [ ] Multiple queued messages enter in arrival order with zero drops; a message queued during the **final** turn yields a new turn
- [ ] A turn returning `ai.FinishReasonPauseTurn` resumes with the returned partial message replayed **byte-verbatim**, and the run reaches a real terminal
- [ ] The pause appears on the stream as `TurnOutcomePaused` — visible, not absorbed
- [ ] A `PermissionDefer` verdict parks, the driver wakes it via the injected scheduler, and the run completes — synchronized by `agenttest.Gate`, **no timeout, no wall clock**
- [ ] The `R-APP-002` bite observes the parked **wait** and RED-records before GREEN
- [ ] `TestTurn_TwoSequentialTurnsShareNothing` passes **file-unchanged**; every AG-09/AG-10 scheduler test passes file-unchanged
- [ ] Every `R-LSK-004` substrate file — including `stream_check.go`, `doc.go`, `doc_contract_guard_test.go`, `event_registry_test.go`, `reconstruction_test.go` — and every file under `backend/agent/src/ai/` is byte-unchanged; `go.mod`/`go.sum` unchanged
- [ ] No new `EventKind`, no new `TurnOutcome`, no new exported `History` method (`history_surface_guard_test.go` green untouched)
- [ ] The import guard and the ambient-authority guard pass with **zero changes** over both the production and test closures
- [ ] All five spec deltas written; each cited line re-read against the shipped change by `sdd-verify`
- [ ] `make lint` clean (after `golangci-lint cache clean`), `make build` clean, `make vuln-check` clean — `vuln-check` is **not** in `make all`
- [ ] doc 0003 line 2170 ticked, and the doc-0003 milestone counters bumped to 13/24
