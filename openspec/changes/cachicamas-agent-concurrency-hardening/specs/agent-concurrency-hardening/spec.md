# Spec — Concurrency hardening under combination (`agent-concurrency-hardening`)

> **Change**: `cachicamas-agent-concurrency-hardening` · **AG-21** (Layer 2, Wave 6, milestone 22 of 24) of [doc 0003](../../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-21--harden-concurrency-backpressure-and-leaks), charter `0003:1963-2043`
> **Nodes**: AG-21.1 `[leaf]` cancellation-and-failure matrix (`0003:1983-2010`) · AG-21.2 `[leaf]` slow-consumer pressure (`0003:2012-2028`) · AG-21.3 `[leaf]` package leak sweep (`0003:2030-2041`)
> **Status**: **new capability**. This file is the normative text; per the AG-14 / AG-19 / AG-20 precedent it is promoted to `openspec/specs/agent-concurrency-hardening/spec.md` at archive. Six cross-cut deltas ship beside it under this change's `specs/` tree.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && go test -race -count=1 ./...`.
> **IDs**: requirements `R-CNH-0NN`, scenarios `S-CNH-0NN` (bites carry the same `S-CNH-` prefix and are marked **(bite)**), non-functional `NFR-CNH-0NN`. **Append-only.** The `CNH` prefix was re-verified collision-free in this phase: `grep -rn "CNH" openspec/specs/` returns nothing.
> **Allocated IDs**: `R-CNH-001` through `R-CNH-008`; `S-CNH-001` through `S-CNH-017`; `NFR-CNH-001` through `NFR-CNH-005`. This header states the **allocated range and never a total**, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`).
> **Evidence gate**: `cd backend/agent && go test -race -count=1 ./...` with the wall-clock duration recorded, plus `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check` (`vuln-check` is **not** in `make all`; `make all`'s fmt step rewrites committed files and MUST NOT be run). No CI exists.
> **Sources**: charter `0003:1963-2043`; this change's `proposal.md` (decisions D-A, D-B, D1–D5, binding) and `design.md` (AD-1…AD-10, binding).
> **Ownership boundary**: this capability owns AG-21's **assembly** claims — the combined matrix, the two pressure obligations, the sweep, the detached-tool accounting, the cross-run inventory and AG-21's own scope fence. It owns no single-feature behavior; every requirement it stresses is owned elsewhere and is cited, never restated as this capability's own.

> **Note on length.** The `sdd-spec` skill sets a 650-word budget. This spec exceeds it deliberately, on the recorded precedent of `agent-v1-scope/spec.md:326-328` and `agent-cancellation-tree`: twelve cells, two pressure scenarios, a sweep and an absence claim with its own defeat test do not compress without dropping content `openspec/config.yaml` requires be independently verifiable.

## Purpose

Twenty milestones each proved one feature against one fixture. AG-21 asks whether they hold **together**, under adversarial schedules, when a consumer stops reading. The charter states the gap in its own words: the matrix exercises *"the interactions no single-milestone test exercises"* (`0003:1991`).

This capability adds **no production mechanism**. What it adds is falsification: assertions that fail when a combination of shipped behaviors is wrong, where each behavior alone is right.

## Coverage

| Charter leaf | Requirements | Scenarios |
|---|---|---|
| AG-21.1 matrix (`0003:1983-2010`) | `R-CNH-001`, `R-CNH-002` | `S-CNH-001`…`S-CNH-006` |
| AG-21.2 pressure (`0003:2012-2028`) | `R-CNH-003`, `R-CNH-004` | `S-CNH-007`…`S-CNH-010` |
| AG-21.3 sweep (`0003:2030-2041`) | `R-CNH-005`, `R-CNH-006` | `S-CNH-011`…`S-CNH-013` |
| Cross-run state (proposal D2) | `R-CNH-007` | `S-CNH-014`…`S-CNH-016` |
| Scope fence | `R-CNH-008` | `S-CNH-017` |

The columns name IDs rather than counts, for the reason the **Allocated IDs** header line records.

## Requirements

### R-CNH-001 — Every combined state survives every signal, and the two cells the charter's uniform Then cannot describe are named rather than forced

For each of the twelve cells below, the run MUST reach its own terminal with a transcript that reads back valid and a stream that `CheckStream` accepts **unmodified**, `stream_check.go` byte-unchanged. The four assertions are uniform across every cell and MUST be carried by one shared assertion core, so no cell can be weakened alone:

1. **The run returned** — observed by a read on the run's own completion channel, never by a wall-clock assertion.
2. **Contract-ordered events** — `CheckStream` over the full drained stream reports no violation, and the stream is not modified to make it pass.
3. **Valid history, no open call** — every assistant tool-call part has a matching result entry, real or `R-HIS-007`-synthesized, and no turn is left open (`R-HIS-003`'s `CloseTurn` semantics).
4. **Outcome and error match the firing signal** — the last event is the run-close; an interrupt yields the interrupted run outcome with `errors.Is` against the interrupt sentinel and **not** against the shutdown sentinel, and symmetrically for shutdown (`R-CAN-002`, `R-CAN-005`, `R-CAN-007`); a provider failure yields the failed run outcome with a typed `*Failure` on the `R-RUN-011` path — **except the compaction mid-run and child-harness-active cells**, whose Then the recorded deviation below replaces.

| State | interrupt | shutdown | provider failure |
|---|---|---|---|
| a suspension pending | uniform Then | uniform Then | uniform Then, plus a park-boundary check (`S-CNH-005`) |
| steering queued | uniform Then | uniform Then | uniform Then |
| compaction mid-run | uniform Then | uniform Then | **adjusted Then** — see the deviation below |
| a child harness active | uniform Then | uniform Then | **adjusted Then**, plus containment (`S-CNH-002`) — see the deviation below |

**Recorded charter deviation — two cells, evidence-cited, disposed at the orchestrator level and NOT an open question.** The charter's uniform Then reads *"the run completes its wind-down"* (`0003:1991`). Two cells MUST NOT assert assertion 4's uniform outcome (`RunOutcomeFailed` plus a typed `*Failure`), because two shipped requirements forbid the behavior it would imply and changing that behavior is outside AG-21's charter (*"Performance targets — correctness under pressure only"*, `0003:1973`; AG-21 ships no production mechanism, D1):

- **compaction mid-run × provider failure.** `R-CMP-010`'s own heading is *"Atomic-or-absent by ordering, and a failed compaction never winds the run down"* (`agent-compaction/spec.md:231`), and its text states *"A compaction failure MUST NOT route through the run's cancellation wind-down"* and *"the run MUST continue at its next turn boundary against the uncompacted transcript"* (`:237-239`). The adjusted Then is therefore: the **compaction bracket** closes failed, the **run** continues to its own terminal with a **nil** error and neither the interrupted nor the shutdown outcome. A cell asserting a wind-down here would assert a defect.
- **a child harness active × provider failure.** The CHILD's own provider stream fails, not the parent's, and the failure is **contained** rather than propagated: `finishContinuationTurn` (`loop.go:525-566`) invokes the scheduler's `Schedule` (`:542`), which returns `[]Result` and **no error** at all — a contained tool-execution failure becomes a `Result` carrying an execution-failure outcome, converted by `toolResultMessage` (`:578-598`) into a tool-failure message and appended to history, and can therefore never surface as a Turn-level error on the parent. The adjusted Then is: the parent's stream carries a **closed, contained** subagent bracket (`S-CNH-002`), and the parent reaches its own terminal with a **nil** error and neither the interrupted nor the shutdown outcome.

**A suspension pending × provider failure is NOT one of the two exceptions, despite the timing note in its own scenario (`S-CNH-005`).** The park changes *when* the failure can fire — only at the first provider boundary reachable by resolving the park, never *what* the outcome is — so the cell still asserts the full uniform outcome (`RunOutcomeFailed` plus a typed `*Failure`) at that terminal, plus one additional, narrower check that no provider request was recorded while parked. This is recorded explicitly because an earlier draft of this table and its deviation section named suspension, not child-harness-active, as the second exception — a defect caught by reading the shipped assertion the cell actually runs, not its label.

Neither adjustment weakens assertions 1–3 (run returned, `CheckStream` valid, history paired) — those hold unweakened for both compaction and child-harness-active. What each adjustment replaces is assertion 4 alone, and where it is checked. Recorded here so `sdd-verify` and `sdd-archive` inherit the disposition rather than read it as a defect, and so no later reader takes AG-21 as having quietly narrowed the charter.

#### Scenarios

- **S-CNH-001** — **Charter AG-21.1, the ten uniform-Then cells.** Given a run driven to each of the four pending states — a permission-deferred call parked on the gate; a steering message accepted with a nil return while the turn's provider stream is held; a compaction call in flight; a child harness whose own provider stream is held — where each pending state is proven by a happens-before edge the production code itself provides (a gate-reached close, a synchronous nil `Steer` return, or an event read off the sink) and **never** by elapsed time or by goroutine scheduling, when the cell's signal fires strictly after that edge, then all four assertions of this requirement hold for that cell; and each cell runs under `-race` with its own harness, provider, sink and gates, sharing nothing with any sibling cell.
- **S-CNH-002** — **A child's failure is contained on the parent's lane.** Given a parent run whose tool hosts a child harness with its own held provider stream, when the child's stream fires a terminal provider failure, then the parent's stream carries a **closed** subagent bracket with only `R-DEL-002`-admitted kinds between its markers, the parent reaches its own terminal, and the parent's stream and the child's stream are each `CheckStream`-valid **validated separately, never concatenated** (`R-DEL-005`, `S-AGE-028`); and given the same arrangement signalled by interrupt or by shutdown, then the child winds down before the parent, asserted by observed event index and channel closes, never by elapsed time (`R-DEL-006`, `S-CAN-015`).
- **S-CNH-003** — **(bite)** RED-first, non-vacuity. Given the steering-queued × interrupt cell with its `Interrupt()` call deleted so the held gate simply releases and the run completes, when the cell runs, then it FAILS on the outcome assertion — reporting the completed run outcome where the interrupted one was required — a clean mismatch rather than a hang. RED-recorded with `-count=1` and its wall-clock duration BEFORE that cell is GREEN, then reverted. A cell that passes with no signal fired proves nothing, and this bite is what proves it does not.
- **S-CNH-004** — **The compaction × provider-failure cell, at its adjusted Then.** Given a run whose compaction call is in flight against an injected provider held at a gate, when that provider's script fires a terminal failure, then the `compaction_failed` event is emitted and no `compaction_finished` is (`R-CMP-010`), the transcript reads back byte-identical to its pre-attempt state, **the run does not wind down** — it continues and reaches its own terminal — and this requirement's four assertions hold at that terminal. A run-close carrying the interrupted or shutdown outcome anywhere on this cell's stream FAILS the scenario.
- **S-CNH-005** — **The suspension × provider-failure cell, uniform in outcome and adjusted only in timing.** Given a run whose call is parked on the permission gate, proven parked by the consumer's read of the `permission_decision_required` event, when the park is resolved and the next provider boundary the run reaches fires a terminal failure, then this requirement's four assertions hold at the resulting terminal with the failed run outcome and a typed `*Failure` — the uniform Then, not one of the two recorded exceptions above; and the scenario asserts that **no** provider request was recorded between the park and its resolution — the evidence that no main-provider stream was live to fail while the call was parked, rather than an assumption that none was.

### R-CNH-002 — Cells compose existing fixtures only, and no cell drives concurrent runs

Every cell, pressure scenario and cross-run driver MUST be composed from primitives that already ship: the module's test gate, its fake provider and script/step vocabulary using only the existing emit and hold shapes, and the existing `package agent_test` local fixtures. **No new fixture mechanism, no new step shape, and no `agenttest` widening**: `backend/agent/src/agenttest/` MUST be byte-unchanged (D5). At most **one** new assertion helper may be added, and it MUST live in `package agent_test`, never in `agenttest`.

**Every cell MUST drive exactly one `Run` at a time on one harness value.** Signals fire from the test goroutine, which is `R-CAN-004`'s already-decided shape. Concurrent `Run` calls on one value stay out of scope and are not made safe by this change (`agent-run-driver/spec.md:87`, restated at `agent-cancellation-tree/spec.md:204`); a `-race`-clean concurrent-`Run` cell would publish a guarantee two shipped specs deny, so the fence is a requirement rather than a convention.

#### Scenarios

- **S-CNH-006** — Given the merge base of this change's branch with `origin/main`, when `git diff` is taken over `backend/agent/src/agenttest/`, then the diff is **empty**; and when every file this change adds under `backend/agent/src/agent/` is read, then each declares `package agent_test`, none declares a new fixture type in `agenttest`, and no cell, pressure driver or cross-run driver invokes `Run` on a harness value while another `Run` on that same value has not returned — asserted by construction over the drivers' own source, not by a runtime probe that a scheduling accident could satisfy.

### R-CNH-003 — A stalled consumer loses nothing, and the sanctioned loss path is the only loss admitted

A consumer that stops receiving MUST NOT cause the loss of any event describing a fact already committed to the harness's history (`R-AGE-006`, `agent-event-delivery/spec.md:92-94`). "Stalled" MUST be **structural** — an unbuffered sink with no receiver — and never temporal: no sleep, no timeout and no poll may create or detect the stall.

Loss MUST be measured against **committed facts**, never against a raw event total (proposal D-Q4). `R-AGE-005`'s sanctioned loss lives at the loop-internal boundary and fires only on cancellation with a saturated buffer (`agent-event-delivery/spec.md:82-84`); in a scenario that cancels nothing that path is unreachable, so the only admissible observation is **zero loss** and any absence is unsanctioned and MUST fail.

#### Scenarios

- **S-CNH-007** — **Charter AG-21.2 scenario 1.** Given an unbuffered sink whose consumer reads exactly k events and then blocks on a test-owned resume channel — so the producer is genuinely blocked at its unconditional send, by construction rather than by timing — and a run that is never cancelled, when the resume channel is closed and the consumer drains to completion, then the run completes, `CheckStream` accepts the whole stream unmodified, and **every** fact committed to the transcript has its describing event present on the stream, checked against the scripted event identity set (count, kinds and call identities) so that a single missing committed-fact event is a divergence; and because nothing was cancelled, **zero** events are absent — an absence excused as "sanctioned" FAILS this scenario.
- **S-CNH-008** — **(bite)** RED-first, committed-fact loss. Given a scratch tree in which one committed-fact emission is removed from the wind-down's tail, when `S-CNH-007` runs, then it FAILS naming the missing event; and the bite additionally records that `CheckStream` alone reports **no** violation against that same scratch tree — the removed kind is not bracket-required — proving the completeness assertion does independent work rather than restating the validator. RED-recorded with `-count=1` BEFORE `S-CNH-007` is GREEN, then reverted.

### R-CNH-004 — Cancellation unblocks a stalled stream within the documented bound, observed structurally

When a consumer stall is in progress and cancellation fires, the run MUST unblock and wind down within the documented bound of `R-CAN-006` (`agent-cancellation-tree/spec.md:130-136`). **"Within the bound" MUST be asserted by the run RETURNING** — observed by a read on the run's completion channel — and **never** by a wall-clock assertion, matching `S-CAN-006`'s normative precedent (`agent-cancellation-tree/spec.md:154`). The bound MUST be injected small through the caller-owned scheduler's zero-default field; it is a **ceiling**, never a synchronization point (`NFR-CAN-002`, `agent-cancellation-tree/spec.md:186`).

#### Scenarios

- **S-CNH-009** — **Charter AG-21.2 scenario 2.** Given a consumer stalled structurally as in `R-CNH-003` after reading up to the tool-start of a cancellation-deaf tool, and a caller-owned scheduler injected with a small wind-down bound, when the interrupt signal fires and the consumer resumes draining, then the run **returns**, observed on its completion channel; the detached call is reported typed on the existing execution-failure kind, its `*Failure` reporting the cancellation category and `errors.As`-extracting a detached-call value that names the tool and the call identity (`R-CAN-006`); `CheckStream` accepts the stream unmodified; and no assertion in this scenario reads elapsed time.
- **S-CNH-010** — **(bite)** RED-first, bound proof. Given a scratch tree with the armed bound removed so the wind-down waits on the deaf tool unconditionally, when `S-CNH-009` runs, then it does **not** return and the failure is evidenced by `go test -timeout` reporting the hung test — proving the bound, and not some other mechanism, is what ends the run. The `S-CAN-012` shape (`agent-cancellation-tree/spec.md:155`), taken over a combined cell. RED-recorded BEFORE `S-CNH-009` is GREEN, then reverted.

### R-CNH-005 — The leak sweep is ONE new non-parallel wrapper over this change's own scenarios

AG-21.3's *"wholesale"* (`0003:2037`) MUST be built as **one new top-level, non-parallel test** whose scenario callable runs, serially, the drivers this change adds — the twelve cells, the two pressure scenarios and the cross-run driver — passed to the module's goroutine-leak helper. It MUST NOT wrap the pre-existing suite. Four grounds, each verified:

1. **Mechanically impossible.** The helper pins a sentinel through `tb.Setenv` as its first act (`stream_kit_leak.go:110`), which the testing package panics on when the calling test or an ancestor has already called `t.Parallel()` (`:96-106`). The existing suite is overwhelmingly parallel.
2. **Already normative.** `S-CAN-006` requires it: *"The leak-harness test MUST NOT call `t.Parallel()`"* (`agent-cancellation-tree/spec.md:154`).
3. **Computationally infeasible.** The helper repeats its scenario 50 times (`stream_kit_leak.go:61`).
4. **It would fail by construction.** AG-20 ships tests that gate an observing hook open on purpose; the resulting lane goroutine is a caller-owned leak **by design** (`agent-run-driver/spec.md:321`).

**`NFR-HKS-005` binds this requirement by name and is honoured, not merely cited** (`agent-hook-taxonomy/spec.md:321`, inheritance recorded at `:265`). Three obligations follow and MUST all hold: no driver in the sweep's scenario samples the process goroutine count — the only sampler is the helper itself; **every gate a driver constructs is released inline before that driver returns**, and additionally carries a cleanup release registered at its construction so no abort path leaves a gate held; and the permanently-stalled-observer leak is **not** exercised by the sweep at all — it is AG-20's, and the sweep's scenario set structurally excludes it.

#### Scenarios

- **S-CNH-011** — **Charter AG-21.3.** Given one new top-level test that does **not** call `t.Parallel()`, whose scenario callable invokes this change's cell, pressure and cross-run drivers serially, when it is passed to the module's goroutine-leak helper, then the helper passes; each iteration fully quiesces before returning — every sink drained, every run result received, every gate released and every tool-release channel closed; no driver reads the process goroutine count; and no gate constructed by any driver survives its driver's return, proven by the cleanup release registered at construction as well as by the inline release.
- **S-CNH-012** — **(bite)** RED-first, leak plant. Given the sweep's scenario with one goroutine blocked forever on a never-closed channel planted **once per iteration** — so growth is approximately the repeat count of 50, well past the tolerance of 25 (`stream_kit_leak.go:61`, `:68`) — when the sweep runs, then the helper FAILS. A sub-tolerance plant proves nothing and is explicitly not accepted as this bite. RED-recorded with `-count=1` BEFORE the sweep is GREEN, then reverted.

### R-CNH-006 — The detached-tool report is accounted, not leaked, over a combined cell

A call still running when the wind-down bound expires MUST be **accounted for** rather than excluded from the sweep's count: proven alive past the bound **by the typed report**, and proven to exit **by releasing it after the run returned** (`S-CAN-006`'s discipline, `agent-cancellation-tree/spec.md:154`). The sweep MUST include the driver that produces such a report; excluding it to keep the count clean is forbidden, because it would make the sweep pass by not asking.

#### Scenarios

- **S-CNH-013** — Given the sweep's scenario including the pressure-2 driver, when each iteration runs, then that iteration releases its cancellation-deaf tool **after** the run has returned; the typed detached-call report naming the tool and the call identity is observed on that iteration's stream; and across the helper's repeats the goroutine count stays within tolerance — so the third-party goroutine is counted and shown to exit, never excluded from the count. Removing the release from the driver MUST make the sweep fail, which is what distinguishes accounting from exclusion.

### R-CNH-007 — The harness itself retains nothing across runs beyond the enumerated inventory

The claim is **not** "no state carries over" — that is false and would forbid a continuing conversation. It is: **the only state that outlives a run is state the caller explicitly owns or a shipped requirement already enumerates, and the harness itself retains nothing.** The inventory is closed and MUST be asserted row by row:

| State | Outlives a run? | Owner |
|---|---|---|
| the terminal shutdown flag | **yes** — terminal, one-way, holds no transcript, never resumes a run | `R-CAN-005` (`agent-cancellation-tree/spec.md:119`) |
| the session-start latch | **yes** — one-way, per-value bookkeeping only | `R-RUN-013` consequence 2 (`agent-run-driver/spec.md:319`) |
| the caller-supplied transcript | **only if the caller set it**; unset ⇒ a fresh transcript per run | the **caller** |
| the run's cancel function | **no** — cleared at exit | `R-CAN-001` |
| the steering queue | **no** — reopened at entry, closed on every exit | `R-RUN-001`, `R-RUN-011` |

**The transcript row MUST be asserted as an ABSENCE, and the absence MUST carry its own defeat test.** A presence-phrased assertion passes with the mechanism fully defeated — the exact shape that let `S-RUN-003` (`agent-run-driver/spec.md:95`) and `S-CAN-002` (`agent-cancellation-tree/spec.md:88`) look sufficient for two milestones while asserting nothing about carry-over. Run 1 MUST be **adversarial** — interrupted mid-turn with a genuinely open call — so the wind-down really writes orphan synthesis and the turn close into the transcript it holds; otherwise the branch under test is never taken.

#### Scenarios

- **S-CNH-014** — **The absence half, with a minted needle, and the defeat this scenario REQUIRES rather than ships as permanent inline code.** Given a harness value with **no** caller-supplied transcript, whose first run is interrupted mid-turn over a genuinely open call and which seeded that run with a **uniquely minted** nonce — a call identity and message text minted per test execution, never a shape or a substring that another fixture could also produce — when a second run is driven on that same value, then the minted nonce appears **nowhere** in the second run's captured provider requests; **and, as the anti-ghost floor, the same nonce IS found in the first run's own transcript read-back**, proving the needle is findable at all before its absence is believed. **This scenario additionally REQUIRES a defeat**: the identical absence assertion, re-run against a deliberately caller-**shared** transcript, MUST go RED because the nonce then does reach the second run's request. That defeat is `S-CNH-016`'s own bite — a transient, RED-then-revert apply-time proof (this repo's established `(bite)` convention, recorded in `tasks.md`/`apply-progress`, never shipped as permanent code alongside this scenario's own implementing test, `cross_run_state_test.go:70-75`) — not inline code co-located with this scenario. An implementation in which the absence assertion stays green under `S-CNH-016`'s defeat FAILS this scenario, whatever the first two clauses report.
- **S-CNH-015** — **The legitimate-carry half, and the inventory row by row.** Given a harness value whose transcript the caller **did** supply, when the adversarial first run has wound down and a second run is driven, then the caller's transcript contains the synthesized orphan result of `R-HIS-007` and the second run's captured provider request **does** contain it — the legitimate carry, owned by the caller and not by the harness; and on the same value: a session-start observing hook fired exactly **once** across both runs; an interrupt issued **between** the two runs is a no-op and the second run still reaches its own terminal, proving the run's cancel function was cleared rather than latched; a `Steer` issued after the first run returned is rejected typed and a `Steer` during the second run returns nil, proving the queue closed and reopened (`R-RUN-001`); and the shutdown flag remains unlatched on the interrupt path, so the second run is accepted rather than refused.
- **S-CNH-016** — **(bite)** RED-first, the cross-run defeat. Given `S-CNH-014`'s nil-transcript absence assertion executed against a deliberately shared transcript, when it runs, then it FAILS on the absence clause. RED-recorded with `-count=1` BEFORE `S-CNH-014` is GREEN, then reverted. This bite is mandatory: without it the absence assertion is unproven, and an unproven absence assertion is indistinguishable from a vacuous one.

### R-CNH-008 — The scope fence: what AG-21 adds is tests, and the boundaries are asserted

AG-21 MUST register **no** new event kind (the registry stays at its committed count), add **no** new turn outcome member, **no** new cost label and **no** new exported `History` method; MUST add **no** third-party dependency, so `go.mod` and `go.sum` are byte-unchanged (the `goleak` rejection of AI-22.4 stands and is not reversed here, D5); MUST leave every file under `backend/agent/src/ai/` byte-unchanged — Layer 1 is consumed, never edited; and MUST change **no production Go source** absent D1's contingency.

**D1's contingency, if it fires, is delta-backed and never silent.** If a cell or pressure scenario falsifies current production behavior, the fix ships in this same change with its own delta spec. Where such a fix extends an enumeration, it MUST **widen, never reword** — the precedent is `R-CAN-006`'s three-row table and `R-CAN-008`'s row, each extended with the pre-existing rows unchanged in strength and in text (`agent-cancellation-tree/spec.md:150`, `:174`). A cell whose assertion was weakened between its RED and its GREEN is a finding, not a fix.

**No cell may assert a duration, a throughput or a latency** — the charter's own out-of-scope line, verbatim: *"Performance targets — correctness under pressure only"* (`0003:1973`).

#### Scenarios

- **S-CNH-017** — Given the merge base of this change's branch with `origin/main`, when `git diff` is taken over `backend/agent/`, then the diff under `backend/agent/src/ai/` and under `backend/agent/src/agenttest/` is **empty**; the `go.mod`/`go.sum` diff is empty; the event-kind guard passes at its committed kind count with this change registering none; no turn outcome member and no cost label member was added; `Turn`'s and `Run`'s signatures and `Harness`'s exported method set are unchanged; and every production Go source under `backend/agent/src/agent/` is byte-unchanged **unless** a delta in this change names it, in which case that delta records the falsifying cell and the widening it took. And when every file this change adds is read, then no assertion anywhere in it compares a duration, a throughput or a latency against a threshold.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-CNH-001** | **External-package verifiability.** Every scenario above MUST be verifiable from `package agent_test` by `cd backend/agent && go test -race -count=1 ./...`. A behavior reachable only from inside the package is, for this spec, not reachable at all (`NFR-RUN-001`, `NFR-CAN-001`, `NFR-HKS-001`). |
| **NFR-CNH-002** | **Determinism, race cleanliness, and NO CLOCK.** Every test MUST be hermetic and MUST pass under `-race`. Synchronization MUST be by the module's test gate, channel reads and channel closes. **No test may synchronize by sleep, timeout, deadline, poll or wall-clock ordering** — seven shipped NFRs already ban it (`NFR-CAN-002`, `NFR-HKS-002`, `NFR-CST-002`, `NFR-CTX-002`, `NFR-CMP-002`, `NFR-RUN-002`, `NFR-RTY-002`). The documented wind-down bound is the **one** legitimate use of clock time and is a ceiling, never a synchronization point. Evidence MUST be recorded with **`-count=1`** and its wall-clock duration; the real uncached suite for this module is ~170s, so a sub-second or otherwise implausibly fast pass is a cache artifact, not evidence. |
| **NFR-CNH-003** | **Goroutine-leak discipline: release before baseline — INHERITED BY NAME.** `NFR-HKS-005` (`agent-hook-taxonomy/spec.md:321`) states *"AG-21 inherits this rule by name"*, and AG-20 recorded the inheritance as a consequence (`:265`). It binds here without weakening: a test that gates an observing hook MUST NOT assert a goroutine baseline; every gate MUST be released inline once its assertions no longer need the hold **and** unconditionally through a cleanup release registered at the gate's construction; the permanently-stalled leak MUST be exercised only under a cleanup-released gate, and AG-21's sweep does not exercise it at all. |
| **NFR-CNH-004** | **No performance claim.** No scenario, helper or assertion added by this change may assert a duration, a throughput, a latency or any other performance target (`0003:1973`). Correctness under pressure only. |
| **NFR-CNH-005** | **Review budget.** This change ships as a **single** pull request under a pre-accepted `size:exception` against a 1000-line budget counted **excluding every path under `openspec/`**, which the user designated a working folder, with an extension pre-granted for AG-21 specifically. The pull-request description MUST state why the change does not fit the default budget. The reserve slicing boundary, held but not taken (D-A): **U1** the twelve cells · **U2** the two pressure scenarios · **U3** the sweep and the cross-run scenario. |

## Explicit non-requirements — what this spec does NOT claim

Stated so that no test, guard or acceptance line is written as if AG-21 closes more than it does.

| Not claimed | Owner, and why the exclusion is safe |
|---|---|
| Performance targets of any kind | **Never in AG-21.** The charter's own out-of-scope line, verbatim (`0003:1973`) |
| Concurrent `Run` calls on one harness value | **Not this milestone.** `agent-run-driver/spec.md:87` and `agent-cancellation-tree/spec.md:204` both deny it; `R-CNH-002` fences it by construction so a `-race`-clean cell never publishes a guarantee two specs deny |
| Closing envelope invariant 3 | **Already CLOSED by AG-01.1 + AG-20.2** (`agent-event-envelope/spec.md:269`). AG-21 proves it **under pressure** and closes nothing; doc 0003's forward "Requirements → closing nodes" row for R-05 (`0003:2204`) is left naming `AG-01.1, AG-20.2` |
| A new `EventKind`, `TurnOutcome`, `CostLabel` or exported `History` method | **Never in AG-21.** `R-CNH-008` |
| Any third-party leak-detection dependency | **Rejected by AI-22.4 on recorded grounds; the rejection is reversible but is NOT reversed here.** No ADR is written by this change, so `go.mod`/`go.sum` stay byte-unchanged |
| Widening `agenttest` | **Not this milestone.** The gate, provider, script/step and leak helper are **used**, not widened (`R-CNH-002`) |
| The abandoned-never-cancelled path | **Ruled untestable to termination and deliberately unasserted** (`ai-stream-lifecycle` § 5). AG-21 proves the abandoned-**then**-cancelled path (`R-CNH-004`) and claims nothing about the other |
| A permanently stalled observer's goroutine | **AG-20's, by design, and the caller's leak** (`agent-run-driver/spec.md:321`). Excluded from the sweep by construction (`R-CNH-005`) |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2.** Layer 1 is consumed, never edited |
| Telemetry or spans over the hardened assembly | **AG-22** (`0003:2043-2069`) |
| The published Layer 3 readiness contract and the scripted-harness kit | **AG-23** |
