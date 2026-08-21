# Proposal: AG-21 — Harden concurrency, backpressure and leaks

> **Change**: `cachicamas-agent-concurrency-hardening` · **Milestone**: AG-21 (Layer 2, Wave 6, milestone 22 of 24; charter `0003:1963-2043`)
> **Worktree**: `cachicamas-worktrees/ag21-concurrency-hardening` · base `main@54476ded` (PR #182, AG-20)
> **Artifact store**: hybrid (Engram + filesystem) · **Execution mode**: `auto`
> **Delivery**: `single-pr` with `size:exception` **pre-authorized by the user before this phase** — the change, the doc 0003 update and the OpenSpec archive in one PR (the AG-16…AG-20 house pattern)
> **Review budget**: 1000 changed lines, counted **excluding every path under `openspec/`** (a working folder). Extension pre-granted for AG-21 specifically. `sdd-tasks` and `sdd-apply` inherit this rule verbatim and MUST NOT re-ask.
> **TDD**: strict, RED-first. Canonical runner `cd backend/agent && go test -race -count=1 ./...`. `make test` is `go test -race -v ./...` with **no `-count=1`** (`backend/agent/Makefile:109-110`), so a cached pass is not evidence; the real uncached suite is ~170s.
> **Depends on**: AG-14, AG-15, AG-16, AG-18, AG-19, AG-20 (`0003:1972`) — all merged.
> **Exploration**: `explore.md` · Engram `sdd/cachicamas-agent-concurrency-hardening/explore`
> **ID prefix**: `R-CNH-` / `S-CNH-` / `NFR-CNH-` — **verified free this phase**: zero occurrences worktree-wide for `[RSN]-CNH-[0-9]`. `sdd-spec` re-verifies before minting.
> **Every `file:line` below was opened in this worktree during this phase.** Where the exploration and a primary source disagree, the primary source wins and the disagreement is recorded in [Corrections](#corrections-to-the-exploration-and-to-the-launch-brief).

---

## Intent

Twenty milestones each proved one feature against one fixture. AG-21 is the first that asks whether they hold **together**, under adversarial schedules, when a consumer stops reading.

The charter states the gap in its own words: the 12-cell matrix exercises "**the interactions no single-milestone test exercises**" (`0003:1991`). That is literally true. Every state and every signal has a ready-made driver — suspension (`harness_suspension_test.go`), steering (`harness_steering_test.go`), compaction (`compaction_call_test.go`), a child harness (`nested_run_test.go`), interrupt (`cancellation_interrupt_test.go`), shutdown (`cancellation_shutdown_test.go`), provider failure (`turn_failure_test.go`'s emitted `ai.MidStreamFailure`) — and **no test composes any of the twelve pairs**.

The deadline is real, and it is not scheduling. AG-21 is the last milestone before the package is handed outward: AG-22 adds telemetry inside a boundary and AG-23 publishes the **Layer 3 readiness contract** (`0003:2043-2069`, `0003:1941-1947`). Whatever concurrency posture AG-21 fails to falsify becomes a published guarantee that a Layer 3 application will build on.

Two further facts make this milestone the forced point, and both were found in this phase rather than inherited:

| Fact | Where | Why it forecloses a later fix |
|---|---|---|
| **Nine live spec sites defer run-outliving state to AG-21 by name**, one explicitly marked *"Still open"* | `agent-loop-skeleton/spec.md:263`; `agent-run-driver/spec.md:87`, `:319`; `agent-cancellation-tree/spec.md:76`, `:119`, `:170`, `:202`, `:205`; `agent-turn-termination/spec.md:181` | AG-22 and AG-23 have no leaf that could discharge them. Left open, they ship into the Layer 3 contract as an unowned deferral |
| **AG-20 handed AG-21 a named, inherited test rule** | `NFR-HKS-005`: *"The permanently-stalled leak MUST be exercised only under a cleanup-released gate. **AG-21 inherits this rule by name.**"* | The leak sweep's shape is already constrained by a shipped NFR. Designing it without that rule produces a sweep that fails by construction |

---

## Resolved decisions

`sdd-design` and `sdd-spec` may overturn any of these **on stated evidence**; neither may resolve one by silence.

### D-A — No split. One change, one PR, the whole milestone

**Decided (pre-settled by the user, recorded here so no later phase re-asks): the charter's own `Split if:` escape hatch (`0003:2010`) is NOT exercised.** All three leaves — AG-21.1's twelve cells, AG-21.2's two pressure scenarios, AG-21.3's leak sweep — ship together under a `size:exception` accepted **up front** against the forecast in [Review-workload forecast](#review-workload-forecast). `sdd-tasks` records `Decision needed before apply: No`. The reserve slicing boundary is kept but not taken: **U1** = the twelve matrix cells · **U2** = the two pressure scenarios · **U3** = the leak sweep and the cross-run scenario.

### D-B — AG-21.3's "wholesale" is a new non-parallel wrapper, not the whole `go test` invocation

**Decided: build ONE new top-level, non-parallel test that runs the combined-scenario matrix and the pressure cells *this change adds* as its `scenario func()`, and pass THAT to `agenttest.RequireNoGoroutineLeak`.** The literal reading — wrapping the pre-existing suite — is rejected on **four** independent grounds, each verified in this phase:

1. **Mechanically impossible.** `RequireNoGoroutineLeak` calls `tb.Setenv(leakSerialOnlySentinel, "1")` as its first act (`stream_kit_leak.go:110`), and its own doc comment states the reason: the `testing` package "panics on when the calling test (or an ancestor) has already called `t.Parallel()`" (`:102-106`). The existing suite carries **1,912 `t.Parallel()` calls across 207 files** under `backend/agent/src` (measured this phase). A wholesale wrapper panics.
2. **Already normative.** `S-CAN-006` does not merely describe this — it requires it: *"The leak-harness test MUST NOT call `t.Parallel()` (`stream_kit_leak.go:80`, `:110`)"* (`agent-cancellation-tree/spec.md:154`).
3. **Computationally infeasible.** The helper repeats its callback `leakRepeats = 50` times (`stream_kit_leak.go:61`). Fifty times a ~170s uncached suite is not a test.
4. **It would fail by construction, and this ground is new.** AG-20 ships tests that gate an observing hook open on purpose; the resulting lane goroutine is a **caller-owned leak by design** (`agent-run-driver/spec.md:321`). `NFR-HKS-005` therefore bans baseline sampling in gated tests outright. A wholesale sweep would sample a baseline across exactly those tests and report AG-20's designed behavior as AG-21's leak.

The design MUST additionally honour `NFR-HKS-005` inside the new wrapper: any gate the matrix cells construct is cleanup-released, and no gated cell samples `runtime.NumGoroutine()`.

### D1 (Q1) — AG-21 ships no production code, and the contingency is pre-authorized rather than discovered

**Hypothesis: AG-21 is a pure test-hardening milestone.** Every primitive the matrix needs already exists in `agenttest` and composes without new mechanism: `Gate` (`fake_gate.go`), `Provider` + `Script`/`Step` (`fake_provider.go`, `fake_script.go`), the `delegatingTool` + second-`Harness` pattern (`nested_run_test.go:14-40`), and provider failure as an emitted `ai.MidStreamFailure` + `ai.ErrorEvent` step (`turn_failure_test.go:32-61`). No `Step` shape beyond `Emit` and `Hold` is required, and none is added.

**Contingency, decided now so it is never a mid-flight negotiation: if any of the twelve cells or either pressure scenario fails against current production code, AG-21 fixes it IN PLACE under this same change, and the fix carries its own delta spec.** It is not deferred to AG-22, not filed as a follow-up, and not absorbed silently into a test adjustment.

The precedent is this project's own and it is one milestone old: AG-20's `sdd-verify` discovered CRITICAL-2 — the observer lane is a goroutine class `R-CAN-006` had never enumerated — and the response was to **widen** `R-CAN-006` from two disjoint sets to three and `R-CAN-008`'s `L2C-08` row alongside it (`agent-cancellation-tree/spec.md:150`, `:174`). A milestone whose whole purpose is "prove it under combination" is precisely the milestone that surfaces such a class.

**Budget consequence, stated rather than discovered.** If the contingency fires, the counted diff grows by the production fix (small — the AG-20 precedent was a doc row and a table row) and the delta spec is `openspec/` and therefore **uncounted**. The pre-granted `size:exception` absorbs it; `sdd-apply` does not stop.

**The widen-not-reword rule binds any such fix.** `R-CAN-006`'s three-row table and `R-CAN-008`'s row were each extended with the pre-existing rows and obligations *unchanged in strength and in text* (`:150`, `:174`). A fourth goroutine class discovered by the matrix takes the identical treatment.

### D2 (Q2) — Cross-run state is **OWED**, not discharged by citation. Option (b)

**Decided: AG-21 adds an explicit scenario proving that state does not carry over between two serial runs on one harness value under adversarial (interrupted / shut-down / failed) conditions. It attaches to AG-21.1** — it is a state × signal cell in everything but name: the state is "a harness value whose previous run wound down adversarially", the signals are the three the matrix already fires.

Option (a) — "stale wording, corrected at archive" — is **falsified by four checks**, all made against the promoted specs rather than the exploration's summary.

**Check 1: it is not one reference; it is nine, across five capabilities.** The exploration found one (`agent-run-driver/spec.md:87`). A repo-wide grep this phase found:

| Site | Text |
|---|---|
| `agent-run-driver/spec.md:87` | *"**Cross-run transcript state remains AG-21's**"* — inside the live `R-RUN-001` |
| `agent-run-driver/spec.md:319` | AG-20's session-start latch *"does not pre-empt AG-21's cross-run state"* |
| `agent-cancellation-tree/spec.md:76` | *"cross-run transcript state remains AG-21's"* — inside `R-CAN-002` |
| `agent-cancellation-tree/spec.md:119` | the shutdown flag *"does not pre-empt AG-21's cross-run state"* — inside `R-CAN-005` |
| `agent-cancellation-tree/spec.md:205` | non-requirement row: *"Cross-run transcript state beyond the terminal shutdown flag \| **AG-21**"* |
| `agent-cancellation-tree/spec.md:202` | non-requirement row: *"The package-wide goroutine-leak sweep \| **AG-21**"* |
| `agent-cancellation-tree/spec.md:170` | `R-CAN-008`: `L2C-08` binds *"AG-19's subagents and **AG-21's leak sweep**"* |
| `agent-loop-skeleton/spec.md:263` | *"**Multi-turn state** — AG-21. … **(Still open.** AG-13 owns one run's iteration; state that outlives a run remains AG-21's.)"* |
| `agent-turn-termination/spec.md:181` | *"**Multi-turn state beyond a run remains AG-21's** and is not closed here."* |

**Check 2: one of them says "Still open" in its own words.** `agent-loop-skeleton/spec.md:263` carries an explicit open marker. That capability's siblings use the same slot to write *"CLOSED by AG-13"* when a deferral is discharged (`:262`, `:264`). A wording judged stale by AG-21 would be contradicting a spec that just declared itself unresolved.

**Check 3: `S-RUN-003` and `S-CAN-002` assert presence, and the obligation is absence.** Both were opened. `S-RUN-003` (`agent-run-driver/spec.md:95`) proves a second `Run` "is accepted", that a `Steer` returns nil and "its message reaches the second run's transcript", and that the second run "reaches its own terminal" — *"proving the queue reopened rather than staying closed"*. `S-CAN-002` (`agent-cancellation-tree/spec.md:88`) proves the second run "emits its own complete run bracket", ends completed, and is `CheckStream`-valid. **Neither asserts that anything from run 1 is absent from run 2.** An absence obligation needs a defeat test; a presence assertion passes with the mechanism fully defeated.

**Check 4: the mechanism is real, observable, and unasserted.** `Run` resolves the transcript at entry: `hist := h.History; if hist == nil { hist = NewHistory() }` (`harness.go:418-421`). So `Harness.History` is a caller-owned **exported configuration field**, and on the adversarial path run 1's wind-down *writes into it* — `windDownRun` calls `hist.SynthesizeOrphans()` and `hist.CloseTurn()` (`harness.go:353-354`). With a caller-supplied `History`, run 2's first provider request therefore contains run 1's synthesized orphan entries. With a nil `History`, run 2 gets a fresh one and nothing carries. **Both branches are correct; neither is asserted anywhere.**

The project has already answered the adjacent slice in exactly this shape, which is the template: `S-CAN-013` proves the **cost** figure does not carry over (*"its own final figure counts only its own turns"*, `agent-cancellation-tree/spec.md:90`). Transcript is the remaining unproven slice.

**The scenario's obligation, stated so `sdd-spec` cannot write a vacuous one.** The claim is *not* "no state carries over" — that is false and would forbid a continuing conversation. It is: **the only state that outlives a run is state the caller explicitly owns or the spec already enumerates, and the harness itself retains nothing.** `sdd-spec` MUST enumerate against the measured inventory of run-outliving `Harness` state:

| Field | Outlives a run? | Owner |
|---|---|---|
| `h.shutdown` | **yes** — terminal, one-way | `R-CAN-005` (`harness.go:457-461`) |
| `h.sessionStarted` | **yes** — one-way latch | `R-HKS-005` (`harness.go:480-483`) |
| `h.History` | **only if the caller set it**; nil ⇒ fresh per run | the caller (`harness.go:418-421`) |
| `h.cancelRun` | **no** — cleared at exit under `signalMu` | `R-CAN-001` (`harness.go:498-503`) |
| `h.queue` | **no** — reopened at entry, closed on every exit | `R-RUN-001`, `R-RUN-011` (`harness.go:468-470`, `:515`) |

**Who applies the corrections.** The nine sites are discharged by back-annotation in this change's delta specs, promoted by `sdd-archive`. `agent-loop-skeleton:263` and `agent-turn-termination:181` change from *"Still open"* / *"not closed here"* to a **CLOSED-by-AG-21** row citing the new scenario, in the shape `agent-loop-skeleton:262` already uses for AG-13. The `agent-cancellation-tree` non-requirement rows `:202` and `:205` gain their *"CLOSED by AG-21"* dispositions the same way `:197` and `:199` carry AG-19's and AG-15's.

### D3 (Q3) — R-05 closes by **back-annotation** on `R-AGE-008`, plus one ADDED scenario

**Decided: a back-annotation paragraph on the existing `R-AGE-008` in `agent-event-delivery`, plus one ADDED `S-AGE-0NN` carrying the empirical under-pressure proof. NOT a new normative obligation, and NOT a requirement in a new capability.**

The launch brief's premise — that `0003:2265` names AG-21 as R-05's closing node — **does not survive opening the file.** doc 0003 carries two different tables and `:2265` is the wrong one:

| Table | Heading | R-05 / AG-21 row |
|---|---|---|
| `0003:2196-2221` | **"Requirements → closing nodes"** | `:2204` — R-05 closes by *"AG-04.3, AG-05.1 (invariant 1); AG-04.1, AG-19.1 (invariant 2); **AG-01.1, AG-20.2** (invariant 3); AG-04.3, AG-11.2 (invariant 4)"*. **AG-21 is absent.** |
| `0003:2238-2267` | **"Nodes trace back to scope"** — *"Every milestone's purpose, for the reverse direction of the spine"* (`:2240`) | `:2265` — *"AG-21 \| R-05 (invariant 3 under pressure), the assembled whole"* |

A milestone **tracing to** a requirement is not a milestone **closing** it. And the owning spec settles it independently: `agent-event-envelope/spec.md:269` records invariant 3 as **`AG-01.1 + AG-20.2 — CLOSED`**, with `:272` adding *"invariant 3 is now closed **jointly**"*. **AG-21 cannot close what is already closed.**

So the only shape consistent with both sources is the one AG-19 and AG-20 already used on this exact requirement: a back-annotation recording what a later milestone **proved** about an already-decided mechanism. `R-AGE-008` carries AG-20's as a titled block (`agent-event-delivery/spec.md:125-134`) with a *"What the back-annotation does NOT claim"* paragraph and a `(Previously: …)` record; `S-AGE-010` and `S-AGE-011` carry `*(AG-20 update: …)*` parentheticals (`:143`, `:151`). AG-21 takes the identical shape.

Two constraints the annotation MUST respect, both from the requirement's own text:

- **`R-AGE-008` bars satisfaction by prose**: *"A statement of the obligation, a convention, a review rule, or documentation alone MUST NOT satisfy this requirement"* (`:123`). AG-21's contribution is therefore the **ADDED scenario**, not the paragraph. The paragraph records; the scenario proves.
- **The stalled-consumer and stalled-hook claims must not be conflated.** `S-AGE-010` is about a stalled **consumer**; `S-AGE-030` is AG-20's parallel claim for a stalled **hook**, and `:143` says explicitly *"the two are separate and must not be conflated."* AG-21's scenario is the **third** member of that family: the same no-path-back trace asserted **under load and in combination** — a stalled consumer while a run is simultaneously suspended, steering, compacting or hosting a child, and then signalled. That is what "under pressure" means, and it is falsifiable rather than restated.

### D4 (Q4) — One NEW capability, plus deltas on five existing ones

**Decided: `agent-concurrency-hardening` is a NEW capability holding AG-21's own obligations, with cross-cutting deltas.** The precedent is uniform — AG-14 minted `agent-cancellation-tree`, AG-19 `agent-delegation-readiness`, AG-20 `agent-hook-taxonomy` — and the test is whether the milestone owns obligations no existing capability owns. It does: the combined-matrix obligation, the two pressure obligations and the sweep obligation are each about the **assembly**, and no single-feature capability can own an assembly claim without over-reaching its own boundary.

**New**

- **`agent-concurrency-hardening`** — kebab-case, matching the change name; directory does not exist. IDs `R-CNH-0NN` / `S-CNH-0NN` / `NFR-CNH-0NN`, **append-only**, prefix verified free. The header **states the allocated range and never a total** (`S-LSK-020`, and this repo's known count-assertion drift class).

Expected requirement families — `sdd-spec` finalises the text, this proposal fixes the shape:

| ID | Owns |
|---|---|
| `R-CNH-001` | The combined-state matrix: every state × signal cell winds down with valid history and contract-ordered events (AG-21.1) |
| `R-CNH-002` | Cell composition uses existing `agenttest` primitives only; no new fixture mechanism, no new `Step` shape |
| `R-CNH-003` | A stalled consumer loses nothing, and the sanctioned loss path is the **only** loss observed (AG-21.2 sc. 1) |
| `R-CNH-004` | Cancellation unblocks a stalled stream within the documented bound (AG-21.2 sc. 2) |
| `R-CNH-005` | The leak sweep: the new non-parallel wrapper of D-B, honouring `NFR-HKS-005` |
| `R-CNH-006` | The detached-tool report is **accounted, not leaked** — empirically, over the combined cells |
| `R-CNH-007` | Cross-run state: the harness itself retains nothing beyond the enumerated inventory (D2) |
| `R-CNH-008` | The scope fence: no new `EventKind`, no new `TurnOutcome`, no production change absent a fired contingency, no `agenttest` widening, no third-party dependency |

**Modified — five capabilities**

| Capability | Requirement | Op | Why |
|---|---|---|---|
| `agent-event-delivery` | `R-AGE-008` + ADDED `S-AGE-0NN` | **MODIFIED (back-annotation) + ADDED scenario** | D3. Invariant 3 proven under pressure; the requirement's own no-prose bar is cleared by the scenario, not the paragraph. **Certain** |
| `agent-run-driver` | `R-RUN-001` (`:87`), `R-RUN-011` (`:319` context) | **MODIFIED (back-annotation)** | D2. *"Cross-run transcript state remains AG-21's"* is discharged and the inventory recorded. `R-RUN-001`'s *"Concurrent runs on one value stay out of scope"* is restated as **still true** — see D5. **Certain** |
| `agent-cancellation-tree` | non-requirement rows `:202`, `:205`; `R-CAN-002` (`:76`), `R-CAN-005` (`:119`), `R-CAN-008` (`:170`) | **MODIFIED (back-annotation)** | Five AG-21-naming sites discharged in one delta, the shape `:197`/`:199` already use for AG-19/AG-15. **Certain** |
| `agent-cancellation-tree` | `R-CAN-006` disjoint-set table, `R-CAN-008` `L2C-08` row | **CONDITIONAL — MODIFIED (widen)** | Fires **only** if a matrix cell surfaces a fourth goroutine class. Widen-not-reword, per `:150`/`:174`. **Not requested by this proposal.** |
| `agent-loop-skeleton` | `:263` *"Multi-turn state — AG-21 … (Still open)"* | **MODIFIED (back-annotation)** | Closed by D2's scenario. **No `R-LSK-004` substrate release is requested** — AG-21 opens no forbidden file. **Certain** |
| `agent-turn-termination` | `:181` *"Multi-turn state beyond a run remains AG-21's"* | **MODIFIED (back-annotation)** | Closed by the same scenario. **Certain** |
| `agent-v1-scope` | `:317` (the four postures AG-21 inherits), `S-AGS-035` (`:210`) | **MODIFIED (back-annotation)** | Records that all four inherited postures were honoured; `S-AGS-035`'s reverse-pass row for AG-21 resolves to R-05 per `0003:2265`. **Likely** |
| ~~`agent-hook-taxonomy`~~ | ~~the whole promoted file~~ | **RESOLVED — no delta owed** | The promoted spec was a truncated placeholder. **Repaired by the orchestrator in commit `c46b696b`** before `sdd-design` ran, byte-identical to the change's own archived delta (355 lines). AG-21 owes this capability **no delta**; `sdd-archive` must not look for a seventh one. The restored `NFR-HKS-005` is instead an *input* to AG-21 — it ends "AG-21 inherits this rule by name" and is adopted verbatim as `NFR-CNH-003`. |

### D5 (Q5) — Explicit non-requirements

Stated in the house style (`agent-cancellation-tree/spec.md:191-208`) so that no test, guard or acceptance line is written as if AG-21 closes more than it does.

| Not claimed | Owner, and why the exclusion is safe |
|---|---|
| **Performance targets** | **Never in AG-21** — the charter's own out-of-scope line, verbatim: *"Performance targets — correctness under pressure only"* (`0003:1973`). No cell may assert a duration, a throughput or a latency |
| **Concurrent `Run` calls on one harness value** | **Not this milestone.** `R-RUN-001`: *"Concurrent runs on one value stay out of scope and are not made safe by this change"* (`agent-run-driver/spec.md:87`), restated at `agent-cancellation-tree/spec.md:204`. **The race-detector matrix MUST NOT silently start testing it** — a `-race`-clean concurrent-`Run` cell would publish a guarantee two specs deny |
| **Any new `EventKind`, `TurnOutcome`, `CostLabel` or exported `History` method** | **Never in AG-21.** The registry is pinned at 25 kinds by a live test (`scope_fence_test.go`, via `R-DEL-009`); AG-21 emits nothing and registers nothing |
| **Any third-party leak-detection dependency** | **Rejected by decision, and the rejection is scoped and reversible but NOT reversed here.** AI-22.4 weighed `go.uber.org/goleak` and rejected it on two recorded grounds — `openspec/AGENTS.md` rule 5 (new top-level dependency ⇒ ADR first) and the package's dependency-free pin (`stream_kit_leak.go:1-25`). AG-21 writes no such ADR, so `go.mod`/`go.sum` are byte-unchanged |
| **A wall clock, timeout, deadline, sleep or poll as synchronization** | **Never in Layer 2.** `R-RUN-010`; and `NFR-CAN-002`, `NFR-HKS-002`, `NFR-CST-002`, `NFR-CTX-002`, `NFR-CMP-002`, `NFR-RUN-002`, `NFR-RTY-002` each already ban it. **The documented wind-down bound is the one legitimate use of clock time**, and it is a ceiling, never a synchronization point (`NFR-CAN-002`) |
| **Widening `agenttest`** | **Not this milestone.** `Gate`, `Provider`, `Script`/`Step` and `RequireNoGoroutineLeak` are **used, not widened**; `backend/agent/src/agenttest/` is byte-unchanged unless design proves otherwise on the record |
| **The abandoned-never-cancelled path** | **Ruled untestable to termination and deliberately unasserted** (`ai-stream-lifecycle` § 5, restated at `stream_kit_leak.go:32-45`). AG-21 does not pretend otherwise; the abandoned-**then**-cancelled path is what AG-21.2 sc. 2 proves |
| **A permanently stalled observer's goroutine** | **AG-20's, by design, and the caller's leak** (`agent-run-driver/spec.md:321`). `NFR-HKS-005` binds AG-21 by name: gates are cleanup-released and gated tests sample no baseline |
| **Any edit under `backend/agent/src/ai/**`** | **Never in Layer 2.** Layer 1 is consumed, never edited |

---

## Scope

### In

- **AG-21.1** — twelve combined-state × signal cells composed from existing primitives, under `-race`, each proving wind-down with valid history and contract-ordered events accepted by `CheckStream` unmodified.
- **AG-21.2** — a stalled-consumer harness proving zero loss beyond the sanctioned path of `R-AGE-005`/`R-AGE-007`, and cancellation unblocking a stalled stream within `R-CAN-006`'s bound.
- **AG-21.3** — the new non-parallel leak-sweep wrapper of D-B over this change's own cells, with the detached-tool report accounted rather than excluded.
- **The cross-run state scenario** of D2, and the nine back-annotations that discharge it.
- The new capability spec, the deltas above, the doc 0003 update (AG-21 ticked, checklist row `0003:2179`, counter to 22/24) and the OpenSpec archive — **same PR**.

### Out

Every row of D5's non-requirements table, plus:

| Also deferred | Owner |
|---|---|
| Telemetry / spans over the hardened assembly | **AG-22** (`0003:2043-2069`) |
| The published Layer 3 readiness contract and scripted-harness kit | **AG-23** |
| A production subagent tool, failover, compaction quality, permission persistence, cross-run cost aggregation | **After Layer 2 v1** (`0003:2183-2192`) |

---

## Approach

1. **Write the new capability spec and the deltas first.** The nine back-annotation sites and D3's shape are settled here; `sdd-spec` writes them before any test lands, so no promoted spec spends a commit contradicting the code.
2. **Build the four state fixtures and the three signal drivers as reusable local helpers** in `package agent_test`, each proven against its own single-feature baseline before any pair is composed. A cell that fails must fail for the *combination*, not for a mis-built fixture.
3. **The twelve cells**, one `t.Run` per cell, table-driven by state × signal. Each asserts: wind-down completes, the transcript reads back with no open call, the stream is `CheckStream`-valid unmodified, and the run-end outcome and error chain match the firing signal.
4. **The two pressure scenarios** — a `Gate`-blocked sink read as the stalled consumer; loss measured against `R-AGE-006`'s committed-fact rule, not against a total.
5. **The cross-run scenario** of D2, in both branches (nil `History` and caller-shared `History`).
6. **The leak sweep** of D-B, last, over everything above.
7. **Doc 0003 update + archive**, same PR.

**Bites are not optional.** At minimum: (a) delete a cell's signal and the wind-down assertion MUST fail — a cell that passes with no signal fired proves nothing; (b) drop one committed-fact event from the stalled-consumer stream and the loss assertion MUST report divergence; (c) plant a deliberately leaked goroutine inside the sweep's `scenario` and `RequireNoGoroutineLeak` MUST fail — the helper's tolerance is `leakRepeats/2 = 25` (`stream_kit_leak.go:68`), so a sub-tolerance plant would pass and prove nothing, and the plant MUST be one-per-iteration; (d) reuse a caller-supplied `History` across two runs and the cross-run scenario MUST distinguish that from the nil-`History` branch. Each RED-recorded **before** its GREEN, with `-count=1`, then reverted.

---

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | **New** | The twelve cells, two pressure scenarios, cross-run scenario, leak sweep, four bites |
| `openspec/specs/agent-concurrency-hardening/` | **New capability** | Via this change's delta folder |
| `openspec/specs/{agent-event-delivery, agent-run-driver, agent-cancellation-tree, agent-loop-skeleton, agent-turn-termination, agent-v1-scope}` | **Delta** | Six; `agent-cancellation-tree`'s `R-CAN-006`/`R-CAN-008` widening **conditional and not requested** |
| `docs/architecture/milestones/0003-…md` | **Modified** | AG-21 status, delivery table, checklist row `:2179`; counter to 22/24 |
| Production Go sources | **NOT TOUCHED** — unless D1's contingency fires | No new mechanism is anticipated |
| `backend/agent/src/agenttest/`, `event.go`, `stream_check.go`, `doc.go`, `doc_contract_guard_test.go`, `go.mod`, `go.sum`, **all of `src/ai/`** | **NOT TOUCHED** | No fixture widening, no new kind, no dependency, no Layer 1 edit |

---

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | **`openspec/specs/agent-hook-taxonomy/spec.md` is a 22-line truncated placeholder** — line 14 reads *"[Content continues: the full 354-line spec follows exactly as read above…]"* and line 16 is a `## Key Learnings` block. AG-20's entire normative surface (`R-HKS-001…012`, `S-HKS-*`, `NFR-HKS-001…006`) is **absent from the promoted specs tree**. AG-21 depends on AG-20 and cites into it — `R-CAN-006:144` cites `R-HKS-008`, `S-CAN-016:156` is *discharged by* `S-HKS-017`/`S-HKS-019`, `agent-run-driver:321` cites `NFR-HKS-005` — and **every one of those citations is currently unresolvable from `openspec/specs/`** | **Certain — verified, not predicted** | Content survives intact at `openspec/changes/archive/2026-08-20-cachicamas-agent-hook-taxonomy/specs/agent-hook-taxonomy/spec.md` (≥326 lines; `NFR-HKS-005` at `:319`). **This is AG-20 remediation, not AG-21 scope** — flagged to the orchestrator for a decision rather than absorbed silently. Repair is `openspec/`-only and therefore **uncounted** against the review budget. Recommendation: repair it in this PR as a prerequisite, recorded as such |
| 2 | **A cell fails and the failure is absorbed as a test adjustment** rather than surfaced as a production defect | **Med-High** | D1's contingency is pre-authorized *and* pre-budgeted. `sdd-verify` must treat any cell whose assertion was weakened between RED and GREEN as a finding |
| 3 | **The matrix silently starts testing concurrent `Run` calls** on one harness value, publishing a guarantee two specs deny | **Med** | D5 row 2, quoting both `agent-run-driver:87` and `agent-cancellation-tree:204`. Every cell drives **one** run at a time; signals fire from another goroutine, which is `R-CAN-004`'s already-decided shape, not concurrency |
| 4 | **The leak sweep is built as a wholesale suite wrapper**, panics on `tb.Setenv`, or reports AG-20's designed leak as AG-21's | **Med-High** | D-B, decided on four verified grounds, with `NFR-HKS-005`'s release-before-baseline rule named as binding |
| 5 | **The cross-run scenario is written as a presence assertion** and passes with the mechanism defeated — the exact failure mode that let `S-RUN-003`/`S-CAN-002` look sufficient | **High** | D2's enumerated inventory is normative; bite (d) is mandatory; the assertion must name a uniquely minted run-1 artifact and be watched to FAIL before it is believed |
| 6 | **The nine back-annotation sites are partially applied — no Go test enforces a back-annotation** | **Med-High** | Enumerated with file and line in D2. Eight of the nine were **not** in `explore.md` and were found by grep in this phase; `sdd-verify` opens each cited line |
| 7 | **Evidence recorded from a cached run.** `make test` has no `-count=1` (`Makefile:109-110`) | **Med** | `-count=1` in every acceptance line; wall-clock duration recorded. AG-21 adds 15+ scenarios on top of a ~170s baseline, so the run will be materially longer, not shorter |
| 8 | **A cell needs a sleep to pass** and one is added | **Med** | Seven shipped NFRs ban it. Synchronization is `agenttest.Gate`, channel reads and closes only; the wind-down bound is the sole clock and is a ceiling |
| 9 | **A fourth goroutine class is discovered and `R-CAN-006` is reworded rather than widened**, silently weakening the first three rows | Med | D1's widen-not-reword rule, citing the two precedents at `:150` and `:174` |
| 10 | `R-AGE-008` is treated as satisfiable by the back-annotation paragraph alone | **Med** | D3: the requirement's own `:123` bar. The ADDED scenario is the discharge; the paragraph only records |
| 11 | The counted diff materially exceeds the forecast | **Med-High** | `size:exception` pre-accepted with extension. Reserve slicing boundary U1/U2/U3 held but not taken (D-A) |

---

## Rollback plan

**Single revert of the AG-21 merge commit.** All new test files are deleted; the deltas are dropped; the new capability directory is removed at un-archive; doc 0003's AG-21 line un-ticks and the counter returns to 21/24. If D1's contingency fired, the production fix and its delta revert with it.

**The revert is clean, and the reason is structural.** AG-21's default shape adds **no production code, no exported symbol, no `EventKind`, no `go.mod` entry and no Layer 1 edit**, so no caller compiled against anything that disappears. Nothing persists across processes; no data migrates. The only consumer of everything added is `package agent_test`.

**Forward-looking cost**: reverting re-opens the nine cross-run deferrals, returns R-05's "under pressure" trace row to unproven, leaves the combined-scenario checklist row (`0003:2179`) unticked, and blocks AG-22 and AG-23, which both depend on AG-21.3 (`0003:1950-1952`). Scheduling consequence, not correctness — **except** where the contingency fired, in which case the revert restores a real defect.

---

## Review-workload forecast

**Counting rule**: additions + deletions, **excluding every path under `openspec/`**. SDD markdown still counts toward the **attempt budget** — a different budget; `sdd-tasks` MUST NOT conflate them.

| Component | Estimate (authored, non-`openspec`) |
|---|---|
| Four state fixtures + three signal drivers, as reusable local helpers | 120–220 |
| Twelve matrix cells (AG-21.1) | 600–1100 |
| Two pressure scenarios + the stalled-consumer harness (AG-21.2) | 150–300 |
| Cross-run state scenario (D2, both branches) | 80–150 |
| Leak sweep wrapper (AG-21.3) | 80–150 |
| Four bites | 100–180 |
| Doc 0003 status line, delivery table, checklist row | 10–25 |
| **Counted total** | **1140–2125** |
| *Uncounted (attempt-budget relevant)*: proposal, new capability spec, 6–7 deltas, design, tasks, apply-progress, verify-report, archive report, `agent-hook-taxonomy` repair | *1400–2400* |

`Decision needed before apply: No` — `size:exception` pre-accepted at 1000 counted lines with a user-pre-granted extension for AG-21 specifically.
`Chained PRs recommended: No` — reserve boundary U1 / U2 / U3 held but not taken (D-A).
`400-line budget risk: High`

*(This forecast is higher than the exploration's 960–1795 because it adds D2's scenario and the fixture-helper layer, both of which the exploration did not scope.)*

---

## Dependencies

- **AG-14** (merged) — `R-CAN-001…008`, the wind-down bound (`agent-cancellation-tree/spec.md:132`: `100ms`, package constant, armed only by cancellation, overridable via a zero-default `Scheduler` field), the detached-call carrier, `S-CAN-006`'s leak-harness shape.
- **AG-16** (merged) — `R-CST-004`'s run-frame-local rule and `S-CAN-013`, the template for D2's absence claim.
- **AG-18** (merged) — `runCompaction` and the compaction bracket, the "compaction mid-run" state's driver.
- **AG-19** (merged) — the child-harness pattern and `R-DEL-005`'s separate-lane rule, the "child harness active" state's driver.
- **AG-20** (merged, PR #182) — the observer lane, `R-HKS-008`, and **`NFR-HKS-005`, which binds AG-21 by name**. See Risk 1: its promoted spec is currently truncated.
- **AG-01.1** (archived) — `R-AGE-008`'s decided mechanism and `S-AGE-010`'s no-path-back trace, the standard D3's scenario is measured against.
- **AI-22.4 / `agenttest`** — `RequireNoGoroutineLeak` (`stream_kit_leak.go:107`), `Gate`, `Provider`, `Script`/`Step`. **Byte-unchanged** unless design proves otherwise.
- **doc `0003:1963-2043`** — the AG-21 charter and its three Gherkin leaves.
- **The wind-down bound's Go symbol is NOT yet pinned to a `file:line`.** The spec names it as *"a package constant"* without naming the identifier. `sdd-design` MUST grep `scheduler.go`/`harness.go` and pin it before writing; this proposal deliberately invents no name.

---

## Success criteria — restated as verifiable checks

- [ ] `cd backend/agent && go test -race -count=1 ./...` green; wall-clock duration recorded as evidence (a sub-170s pass is a cache artifact)
- [ ] **All twelve cells present and each independently non-vacuous** — bite (a) RED-recorded: remove a cell's signal, the wind-down assertion FAILS
- [ ] Every cell: transcript reads back with **no open call**; `CheckStream` accepts the stream **unmodified**; the run-end outcome and the returned error match the firing signal by `errors.Is`; `stream_check.go` byte-unchanged
- [ ] **No cell drives concurrent `Run` calls on one harness value**, asserted by construction and stated in the spec
- [ ] **Stalled consumer** — no committed-fact event lost (`R-AGE-006`); the only loss observed is `R-AGE-005`'s sanctioned path; bite (b) RED-recorded
- [ ] **Cancellation unblocks the stall** within the documented bound, observed by a channel read and **never by a wall-clock assertion**
- [ ] **Leak sweep** — one new non-parallel top-level test over this change's own cells; it does **not** call `t.Parallel()`; every gate is cleanup-released and no gated cell samples a baseline (`NFR-HKS-005`); bite (c) RED-recorded with a **one-per-iteration** plant that exceeds the tolerance of 25
- [ ] **Detached-tool report accounted, not leaked** — proven alive past the bound by the typed report and proven to exit once the third-party code returns
- [ ] **Cross-run state** — the harness retains nothing beyond the enumerated inventory; both the nil-`History` and caller-shared-`History` branches asserted; bite (d) RED-recorded
- [ ] **Nine back-annotation sites discharged**, each cited line opened by `sdd-verify`; `agent-loop-skeleton:263` and `agent-turn-termination:181` no longer read "Still open" / "not closed here"
- [ ] **R-05 recorded correctly** — the back-annotation states AG-21 **proves invariant 3 under pressure** and does **not** claim to close it; `0003:2204` is left naming `AG-01.1, AG-20.2`
- [ ] **Scope fence** — no new `EventKind` (registry at 25), no new `TurnOutcome`, no new `CostLabel`; `agenttest/` byte-unchanged; `go.mod`/`go.sum` byte-unchanged; all of `src/ai/` byte-unchanged; no production change absent a fired and delta-backed contingency
- [ ] **Guards green unchanged** — ambient-authority, import-boundary, doc-contract, scope fence
- [ ] **No wall clock** — no sleep, timeout, deadline or poll used for synchronization anywhere in the change; the wind-down bound is the only clock and is a ceiling
- [ ] `make lint` (after `golangci-lint cache clean`), `make build`, `make vuln-check` all clean (`vuln-check` is **not** in `make all`; **do not run `make all`** — its fmt step rewrites committed files)

---

## Corrections to the exploration and to the launch brief

Recorded because a primary source outranks a paraphrase, and because each of these would have changed a downstream phase's work.

| # | Claim | Primary source | Correction |
|---|---|---|---|
| 1 | Launch brief: *"doc `0003:2265` names AG-21 as R-05's closing node"* | `0003:2196-2204` vs `0003:2238-2265`; `agent-event-envelope/spec.md:269` | **Wrong.** `:2265` is the reverse **"Nodes trace back to scope"** table. The forward **"Requirements → closing nodes"** table at `:2204` does **not** name AG-21 for R-05, and invariant 3 is already marked **CLOSED** by `AG-01.1 + AG-20.2`. AG-21 stresses it; it does not close it. This is what makes D3 a back-annotation |
| 2 | Exploration §8: the cross-run reference is **one** live site, at `agent-run-driver/spec.md:82` | repo-wide grep, this phase | **Undercounted, and mis-cited.** The line is `:87`, not `:82`, and there are **nine** AG-21-naming sites across five capabilities — one of them (`agent-loop-skeleton:263`) explicitly marked *"Still open"*. This is what falsifies option (a) and settles D2 |
| 3 | Exploration §10.3 gives **two** grounds against the literal "wholesale" reading | `NFR-HKS-005` (`…/archive/…/specs/agent-hook-taxonomy/spec.md:319`); `agent-run-driver/spec.md:321` | **Incomplete.** A third and a fourth exist: `S-CAN-006` already makes the non-parallel rule **normative**, and a wholesale sweep would sample a baseline across AG-20's deliberately-gated observer tests, reporting a **by-design caller-owned leak** as AG-21's. Ground 4 is the strongest of the four |
| 4 | Exploration §9: no production code anticipated, so the whole diff is tests | `harness.go:418-421`, `:353-354` | **Stands, but under-scoped.** It omitted the cross-run scenario and the fixture-helper layer entirely; the forecast is raised accordingly |
| 5 | Not observed by the exploration at all | `openspec/specs/agent-hook-taxonomy/spec.md:14` | **`openspec/specs/agent-hook-taxonomy/spec.md` is a truncated placeholder.** AG-20's entire normative surface is missing from the promoted tree, and three AG-21-relevant citations point into it. See Risk 1 |

*Minor, non-blocking*: `agent-event-envelope/spec.md:263` cites the traceability spine as `0003:2203`, which is R-04's row; R-05's is `:2204`. A one-line citation drift of the class an insertion above it produces. Worth a correction at archive; it changes nothing normative.

---

## Proposal question round

Execution mode is `auto`, so these were not asked interactively. Each changes the shape of the product, not the harness. **Answering any of them before `sdd-design` moves a recommendation above.**

1. **Should AG-21 repair `openspec/specs/agent-hook-taxonomy/spec.md` (Risk 1), or is that a separate AG-20 remediation?** Assumed **repair it here, as a recorded prerequisite**, because AG-21 depends on AG-20 and three of its own citations are currently unresolvable. The repair is `openspec/`-only and therefore free against the review budget. The counter-argument is real: absorbing another milestone's remediation makes this PR's story two stories. Say which you want — this is the one question whose answer changes what lands.
2. **What does "cross-run state" mean to the product — a continuing conversation, or a fresh one?** Assumed **the caller decides, and the harness retains nothing** (D2). With `Harness.History` set, a serially reused harness is one continuing conversation and run 1's synthesized orphans are legitimately in run 2's request; with it nil, each run starts clean. If the product intends a `Harness` value to be single-conversation-only, the nil branch is the only supported one and `R-CNH-007` becomes a prohibition rather than an enumeration.
3. **After an adversarial run, should a caller-shared transcript still carry run 1's synthesized orphan entries into run 2's provider request?** Assumed **yes** — `R-HIS-007` synthesizes them precisely so the pairing invariant survives, and dropping them at the run boundary would strand it. If the product would rather a cancelled turn leave no trace in the next request, that is a real behavior change and it belongs to AG-21, not later.
4. **Is "loses nothing" measured against committed facts, or against every event the run produced?** Assumed **committed facts** — `R-AGE-006`'s rule, with `R-AGE-005`'s sanctioned path the only loss. If a consumer is expected to observe in-flight events of work cut short, that contradicts `R-AGE-007` and needs an amendment there, not a scenario here.
5. **If a matrix cell surfaces a genuine production defect, does it ship fixed in this PR or as a follow-up?** Assumed **fixed here, delta-backed** (D1). The AG-20 precedent is one milestone old. If you would rather see AG-21 report and defer, say so now — it changes the contingency, the budget forecast and what `sdd-apply` is allowed to do without stopping.
