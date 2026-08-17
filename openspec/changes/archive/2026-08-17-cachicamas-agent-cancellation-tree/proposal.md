# Proposal: AG-14 — Build the cancellation tree

> **Change**: `cachicamas-agent-cancellation-tree` · **Milestone**: AG-14 (Layer 2 Wave 3, milestone 14 of 24; doc 0003 lines 1371–1442)
> **Branch**: `feat/agent-layer2-wave3-ag14` · **Artifact store**: hybrid (Engram + filesystem)
> **Pre-authorized**: `size:exception` against the 1000-line PR review budget, extended for AG-14 specifically
> **TDD**: strict, RED-first (`cd backend/agent && make test`)
> **Closes**: R-08's wind-down half; consumes R-09 (`0003:2257`, `0003:2207`)
> **Depends on**: AG-13 (archived, merged `52701436`) · **Parallel with**: AG-15, AG-16
> **Exploration**: `exploration.md` · Engram `sdd/cachicamas-agent-cancellation-tree/explore`

## Intent

AG-13 gave the package a run that iterates. It gave it no way to stop one honestly. Three concrete defects follow from that, all verified in the shipped code:

1. **Cancellation is silently mis-typed.** `wrapHarnessFailure` builds `ai.FailureReport{Category: ai.FailureCategoryUnavailable}` **unconditionally** (`harness.go:184-187`), and `failRun` emits `NewRunEnd(runID, RunOutcomeFailed, failure)` for **every** non-nil `Turn` error (`harness.go:202`). A cancelled run reaches the consumer as *failed / unavailable* — indistinguishable from a provider outage. `RunOutcomeInterrupted` exists (`run_events.go:106-108`) and is used nowhere in production.
2. **An interrupt that races a bare channel close reads as success.** When the provider stream closes with no `Completion` — exactly what Layer 1's fake does on mid-stream cancellation (`agenttest/conformance_cancellation.go`, R-CNF-011/R-CNF-012) — `Turn` normalizes the finish reason to `ai.FinishReasonStop` (`loop.go:425-426`). `Turn`'s body contains **zero** `ctx.Err()`/`ctx.Done()` checks. A user interrupt can therefore be recorded as a normal completed turn.
3. **A tool can hold the run hostage forever.** `executeCall` passes `context.Background()` to `tool.Run` (`scheduler.go:462`), so no executing tool can observe run cancellation — only a *parked* one can, via the permission gate's `ctx.Done()` arms (`scheduler.go:637`, `:657`). `Schedule`'s `wg.Wait()` (`scheduler.go:205`) is unbounded, so a non-returning call goroutine blocks `Schedule`, therefore `Turn`, therefore `Run`, with no bound anywhere in the chain.

R-RUN-010 names the successor explicitly: "richer cancellation vocabulary is **AG-14**'s" (`agent-run-driver/spec.md:200`). **What ticks**: doc 0003 line 2171 — "Interrupt and shutdown are distinguishable end to end; wind-down is bounded — closed by AG-14."

## Scope

### In

- **AG-14.1** — an **interrupt** signal on the harness, sitting beside `Steer` (`harness.go:149-151`) on the same non-privileged upward path (R-RUN-006). Mid-turn: the provider call is cancelled per the fake's cancellation-fidelity contract, in-flight tools observe cancellation, `History.SynthesizeOrphans` (`history.go:270-300`) runs, and the run ends with `RunOutcomeInterrupted` — not `RunOutcomeFailed`.
- **AG-14.1** — **a new prompt on the same harness works afterward.** Interrupt ends the run, not the harness value.
- **AG-14.1** — **interrupt during a permission suspension aborts it typed** and history closes cleanly. The plumbing is 80 % present: both parked-wait arms already call `typedExecutionFailureFromError(call.ID(), ctx.Err())` (`scheduler.go:639`, `:665`) and `parked.remove`; AG-14 makes the typed value carry *which signal*, not raw `context.Canceled`.
- **AG-14.1** — **idempotency**: a second interrupt on a run already winding down changes nothing and panics nothing. No double channel close, no race against the first signal's cleanup.
- **AG-14.2** — a **shutdown** signal: the same wind-down, a run-end outcome that says *shutdown*, and a harness that afterwards **refuses new prompts with a typed rejection**. This is the first cross-run state any Layer 2 milestone has held on a `Harness` value.
- **AG-14.2** — the two signals stay distinguishable **through every layer they cross**: on the stream (run-end outcome) *and* in the Go error chain (`errors.Is`-matchable sentinels).
- **AG-14.3** — a **documented bound** on wind-down. A scripted cancellation-deaf tool cannot prevent the run from ending within it. `BlockingScriptedTool` (`scripted_tool_test.go:70-83`) is the ready-made fixture: its script discards `ctx` and blocks on an unreleased channel.
- **AG-14.3** — the overrun call is **reported typed** — which tool, which call ID, still running — with its goroutine **detached and named, not silently abandoned**.
- **AG-14.3** — **no task belonging to the harness itself remains** after the wind-down: `runDispatcher` (`scheduler.go:267`), the per-turn forwarder (`harness.go:274-279`) and `Schedule`'s own bookkeeping all unwind. Proven with `agenttest.RequireNoGoroutineLeak` (`agenttest/stream_kit_leak.go:107`) — the first Layer-2 use. It is **serial-only** (`stream_kit_leak.go:80`): these tests MUST NOT call `t.Parallel()`.
- A `R-RUN-011` delta carving cancellation-caused errors out of "any error → `RunOutcomeFailed` with a non-nil failure" (`agent-run-driver/spec.md:211-217`).
- Substrate-guard filter widening in `loop_test.go` and `loop_hook_test.go` by **exact filename**, no wildcards, both filters byte-in-sync — the AG-11/AG-13 discipline (`agent-loop-skeleton/spec.md:90-92`).

### Out — deferred, with the owner named

| Deferred | Owner and why deferral is safe |
|---|---|
| Subagent cancellation inheritance | **AG-19.2.** Charter "Out of scope" line (`0003:1381`), binding. No subagent tool ships in v1 (`0003:1794`). |
| Frontend / TUI signal wiring (a keypress or `SIGINT` reaching the harness) | **Layer 3.** Charter line (`0003:1381`). AG-14 defines the mechanism; `cachicamas_coding`'s composition root calls it. |
| Retry / failover on a cancelled or failed turn | **AG-15**, parallel. A cancellation is never retried; `R-RUN-011`'s no-retry rule already draws that line. |
| Cost accounting for an interrupted turn | **AG-16**, parallel. |
| Compaction cancellation-safety ("recoverable if interrupted", v2 § 4.2) | **AG-17/AG-18.** AG-14 makes the ordinary turn/tool path cancellation-aware, nothing else. |
| The **package-wide** goroutine-leak sweep | **AG-21** (`0003:2178`). Deferred by the milestone doc's own words: "the later package-wide leak sweep depends on this report being precise" (`0003:1439`). AG-14 proves only its own goroutines. |
| Deadline / timeout as a **third** signal | **Not this milestone.** The charter's subtitle is "Interrupt ≠ shutdown, and both ≠ deadline" (`0003:1373`) — that is a *distinctness* claim, not a mandate to implement a deadline signal. AG-14.3's bound is a wind-down bound, not a run deadline. |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2.** Layer 1 is consumed, never edited. |

## Open decisions — framed here, resolved by `sdd-design`

### Decision 1 — the substrate-release scope

**Question**: `run_events.go`, `turn_events.go` and `failure.go` are named `R-LSK-004` substrate (`agent-loop-skeleton/spec.md:84`; the latter two were "**released for AG-11 only** … the release does not extend to any milestone after AG-11 without its own recorded delta", `:86`). Does AG-14 request its own scoped release, and how wide?

| Option | Tradeoff |
|---|---|
| **A — no release. Both signals share `RunOutcomeInterrupted`; the distinction lives only in the Go error chain.** | Zero substrate surface. But the charter says "Both distinguishable in run-end **outcomes and error chains**" (`0003:1379`) — plural, both nouns — and AG-14.2's Then says "the run-end outcome **says shutdown**" (`0003:1423`). A stream consumer (Layer 3's TUI) sees no Go error at all; under A it cannot tell the two apart. **Fails the charter as written.** |
| **B — release `run_events.go` only, confined to one new `RunOutcome` member and its `String()` case.** ⭐ **Recommended.** | Structurally forced, exactly as AG-11's paragraph argues for `TurnOutcome`: the `iota` block (`run_events.go:100-118`), the `runOutcomeLimit` bound check inside `RunEnd.validate` (`:153`) and the `String()` switch (`:123-136`) are all **local to `run_events.go`** — a member declared elsewhere would be rejected by the validator and render as the `runoutcome(N)` placeholder. **No `validate` change is needed**: the failure-iff-`Failed` rule (`:156-161`) forbids a `*Failure` on any non-`Failed` outcome, and both `Interrupted` and shutdown are non-`Failed`, so the discriminator rides the outcome member itself rather than an attached payload. Appending before `runOutcomeLimit` leaves `Completed`/`Interrupted`/`Failed` at 1/2/3. `turn_events.go` and `failure.go` stay **frozen — no release requested**. |
| **C — release `run_events.go` *and* amend `RunEnd.validate` to permit a `*Failure` on `Interrupted`.** | Wider than needed under B and directly contradicts AG-11's own scoping sentence, which held `TurnEnd.validate`'s failure-iff-aborted rule and `NewTurnEnd`'s signature immutable even *inside* a granted release (`:86`). Reviewers will read it as re-litigating AG-04's frozen vocabulary. **Not recommended.** |

**Binding on whichever wins**: the release paragraph `sdd-spec` writes into `R-LSK-004` MUST be scoped as tightly as AG-11's ("confined to `R-ATT-001` … and `R-ATT-006`"; `NewTurnEnd`'s signature and `validate`'s rule MUST NOT change) and must state, in the same paragraph, that `turn_events.go`, `failure.go`, `stream_check.go`, `event.go`, `doc.go` and `doc_contract_guard_test.go` stay byte-unchanged. Under B, `NewRunEnd`'s signature and `RunEnd.validate` do not change either.

**Open sub-question `sdd-design` must close**: `CheckStream`/`String()` coverage for the new member without touching `stream_check.go` (`stream_check_test.go:147-153` proves the validator already accepts `RunOutcomeInterrupted`; the test file is not substrate, the validator is).

### Decision 2 — `tool.Run`'s context source

**Question**: `executeCall` passes `context.Background()` (`scheduler.go:462`). Does AG-14 thread the real run `ctx`, or defer it and rely solely on AG-14.3's bound?

| Option | Tradeoff |
|---|---|
| **A — thread the real `ctx` into `tool.Run`.** ⭐ **Recommended.** | The charter demands propagation "down through loop, provider **and tools**" (`0003:1377`), and AG-14.1's first Then requires "in-flight tools **observe cancellation**" (`0003:1399`) — unreachable by construction under B. It also makes AG-14.3 non-vacuous: a "cancellation-deaf tool" is only a meaningful *special* fixture if the ordinary tool is cancellation-*aware*. Blast radius verified as low: no test asserts the identity of `executeCall`'s context. The `context.Background()` occurrences in `agent`'s test closure are (a) callers passing it *into* `Turn`/`Schedule` (`loop_tool_dispatch_test.go:114`, `scheduler_test.go:536`, `permission_protocol_test.go:160`), and (b) direct `tool.Run` unit calls that never go through the scheduler (`tool_test.go:57`, `:82`, `:124`). Both are unaffected. |
| **B — defer; tools stay context-blind, the bound alone protects the run.** | Smaller diff and zero risk to existing tool behavior. But it silently redefines AG-14.1's first scenario into something weaker than its own Gherkin, and pushes a change with the same blast radius onto a later milestone with less reason to make it. |

**Binding**: whichever wins, `sdd-tasks` MUST re-run the grep above against the final tree before apply. The one behavioral consequence to state in writing: under A, a tool that *does* read `ctx` starts returning early on cancellation where it previously ran to completion — that is the intent, and `R-TLS-010`'s disjoint-return-channel rule (`scheduler.go:464-473`) already types the outcome.

### Decision 3 — who owns the bound, and when it is armed

**Question**: AG-14.3's "documented bound" has no existing implementation to extend — there is **zero** `time.After`/`time.NewTimer`/`context.WithTimeout` in `backend/agent/src/agent` production code (verified; the only production timer in the module is `ai/internal/retry/retry.go:199`). What is its value, who owns the timer, and what is "detached and named" concretely?

| Option | Tradeoff |
|---|---|
| **A — an always-on bound on `Schedule`'s `wg.Wait()` (`scheduler.go:205`).** | Simple, one place. **But it breaks a locked-in approval test**: `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` (`permission_protocol_test.go:1573-1600`) deliberately drives `Schedule` with a plain `context.Background()` — "no deadline, no cancel. Nothing but an explicit `WakeParked` can ever unblock this call" (`:1597-1598`) — and exists specifically to freeze D-B's two-production-paths contract. An always-on bound adds a third path. **Not recommended.** |
| **B — a bound *armed by cancellation*: `wg.Wait()` races the bound only once `ctx` is done; on the uncancelled path the wait stays exactly as unbounded as today.** ⭐ **Recommended.** | Preserves `permission_protocol_test.go:1573` file-unchanged and green, and preserves R-RUN-010's stated closed resolution contract ("the harness adds **no third path and no timeout**", `agent-run-driver/spec.md:200`) for the non-cancelled case — the bound is part of *wind-down*, which is what R-RUN-010 already names as path (b). Cost: two code paths through the join instead of one. |
| **C — the harness owns the timer and abandons the whole `Turn` call.** | Leaves `Schedule` leaking its own dispatcher and forwarder goroutines — directly contradicts AG-14.3's third Then (`0003:1439`). **Not recommended.** |

**Sub-decisions `sdd-design` MUST close in writing**, none of which the exploration settles:
1. **The bound's value and its configurability** — a package constant, or a `Harness` field with a documented default? A test asserting "ends within the bound" must synchronize through `agenttest.Gate`, never a wall-clock sleep; the bound itself is the one legitimate use of clock time in this milestone and must be short enough that the suite stays fast.
2. **The typed report's carrier** — a new `Result` outcome, a dedicated event, or a field on the existing execution-failure path. A new **event kind** would touch `event.go`/`event_registry_test.go`, both substrate that Decision 1 does **not** release; prefer a carrier that needs no new kind.
3. **"Detached and named"** — the scope distinction that makes AG-14.3's second and third Thens simultaneously true: the detached goroutine is the *tool's*, whose lifetime is third-party code the harness cannot kill (Go has no goroutine-kill primitive); "the harness's own tasks" means `runDispatcher`, the forwarder and `Schedule`'s control flow, all of which MUST exit. Write that distinction down; do not leave it implicit.

### Cross-cutting — the `L2C-08` question

`doc.go` and `doc_contract_guard_test.go` are substrate (`agent-loop-skeleton/spec.md:84`), and the guard requires a `doc.go` row and its expectation-table row to land in the **same** PR (`doc_contract_guard_test.go:19-22`, `:37`). AG-13 applied AG-12's discriminator — "does this milestone declare a new package-wide guarantee, or implement behavior inside one already declared?" — and concluded no new row. **AG-14 must apply it fresh and record the answer either way**: a bounded wind-down that every future tool and turn family must respect is a stronger candidate for a package-wide row than AG-13's bracketing was. If the answer is yes, Decision 1's release must widen to name both files explicitly.

## Capabilities

### New

- **`agent-cancellation-tree`** — the two-signal cancellation tree: signal vocabulary and propagation, interrupt semantics, shutdown semantics and post-shutdown refusal, bounded wind-down, the detached-call report, and harness-goroutine liveness. IDs `R-CAN-0NN` / `S-CAN-0NN`, bites `S-CAN-0NN`. Prefix `CAN` verified free: zero `[RSN]-CAN-[0-9]` matches repo-wide. Becomes `openspec/specs/agent-cancellation-tree/spec.md` at archive.

### Modified — deltas required

| Capability | What changes |
|---|---|
| `agent-loop-skeleton` | `R-LSK-004` gains AG-14's scoped release paragraph (Decision 1), and its filter-widening rule extends to AG-14's new filenames. |
| `agent-run-driver` | `R-RUN-011` (`:211-217`) needs the cancellation carve-out: a cancellation-caused `Turn` error routes to an interrupted/shutdown outcome with **no** `*Failure`, not `RunOutcomeFailed`. `R-RUN-010`'s "richer cancellation vocabulary is AG-14's" (`:200`) is closed here and needs back-annotation, including whether the wind-down bound counts as the "third path" that sentence forbids. |
| `agent-permission-protocol` | The parked-wait abort path (`scheduler.go:637-643`, `:657-669`) becomes signal-aware; the typed failure's shape changes from raw `ctx.Err()`. |
| `agent-tool-scheduler` | Decision 2 (`tool.Run`'s context source) and Decision 3 (the armed bound, the detached-call report) are both tool-scheduler surface. `R-TLS-010`'s disjoint-return-channel rule must be restated as still-true. |
| `agent-history` | Back-annotation only: `SynthesizeOrphans` gets its first production caller. No new exported `History` route — `history_surface_guard_test.go` stays green untouched. |

## Approach — exploration's recommendation, adopted

`context.WithCancelCause` (Go 1.20+, stdlib — no new dependency, satisfies the R-01 import guard) as the propagation mechanism:

1. `Run` derives the run's own context from the caller's `ctx` once at entry, holding the `context.CancelCauseFunc`.
2. `Interrupt()` / `Shutdown()` sit beside `Steer()` and do nothing but invoke that cancel func with a typed sentinel (`errors.Is`-matchable). They reach the loop through **no privileged channel** (R-RUN-006) — they flip a context the loop already observes. A second call is a documented Go no-op; a `sync.Once` or atomic flag guards the harness-level state.
3. Every site that today reads `ctx.Err()` reads `context.Cause(ctx)` instead — `loop.go`'s closed-channel branch (`:417-427`, gaining its first cancellation check), the two permission-gate arms (`scheduler.go:639`, `:665`), and `failRun`'s routing (`harness.go:200-207`).
4. Wind-down: orphan synthesis, the armed bound (Decision 3), the typed detached-call report, then the run-end outcome (Decision 1).
5. Shutdown additionally sets the harness's refuse-new-prompts flag, checked at `Run` entry.

**Zero new parameters** on `Turn`, `Schedule`, `executeCall` or `tool.Run` — every one of them already receives a `ctx`. **Rejected — a second channel or a second context threaded alongside `ctx`**: signature churn against AG-13's no-signature-change precedent, for no capability `context.Cause` does not already give.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/harness.go` | **Modified** | `Interrupt()`/`Shutdown()`, the cancel-cause holder, post-shutdown refusal state, `failRun`/`wrapHarnessFailure` cause-aware routing (`:183-207`) |
| `backend/agent/src/agent/loop.go` | **Modified** | The closed-channel branch (`:417-427`) gains its first cancellation-cause check before the `FinishReasonStop` normalization |
| `backend/agent/src/agent/scheduler.go` | **Modified** | `tool.Run`'s context source (`:462`, Decision 2); the armed bound around `wg.Wait()` (`:205`, Decision 3); the detached-call typed report; cause-aware parked-wait aborts (`:639`, `:665`) |
| `backend/agent/src/agent/run_events.go` | **Modified — substrate** | **Only under Decision 1 option B**: one `RunOutcome` member + its `String()` case. Requires the scoped `R-LSK-004` release. `RunEnd.validate` and `NewRunEnd`'s signature unchanged. |
| New production file(s) for the signal vocabulary / sentinels | **New** | Names fixed by `tasks.md`; each must be added to both substrate filters by exact filename |
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | **New** | AG-14.1's three scenarios, AG-14.2's one, AG-14.3's one, plus bites. Serial-only where `RequireNoGoroutineLeak` is used. |
| `loop_test.go`, `loop_hook_test.go` | **Modified** | Substrate filter widening by exact filename, byte-in-sync |
| `openspec/specs/{agent-loop-skeleton, agent-run-driver, agent-permission-protocol, agent-tool-scheduler, agent-history}` | **Delta** | Five deltas, per the Capabilities table |
| `docs/architecture/milestones/0003-…md:2171`, `:2207`, `:2257` | **Modified** | Checklist tick, R-08/R-09 back-annotation, milestone counter to 14/24 |
| `turn_events.go`, `failure.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `history.go`, `doc.go`, `doc_contract_guard_test.go`, `event_registry_test.go`, `reconstruction_test.go`, `go.mod`, `go.sum`, `backend/agent/src/ai/**` | **NOT TOUCHED** | No second substrate file, no new event kind, no History surface change, no Layer 1 edit, no new dependency. `doc.go`/`doc_contract_guard_test.go` move to *Modified* only if the `L2C-08` question resolves yes. |

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | Substrate-release scope creep — a wide release reads as re-litigating AG-04's frozen vocabulary | High | Decision 1 option B opens **one** file for **one** member + its `String()` case, tighter than AG-11's two-file release. `validate` and the constructor signature stay unchanged, and the release paragraph says so verbatim. |
| 2 | `tool.Run`'s context change alters every tool's observable behavior | Med | Grep-verified: no test asserts `executeCall`'s context identity. `sdd-tasks` re-runs the grep against the final tree before apply. |
| 3 | The bound breaks `TestPermission_WakeParked_..._NoDeadline` (`permission_protocol_test.go:1573`) or R-RUN-010's "no third path" contract | **High if Decision 3A is taken** | Decision 3B arms the bound only on cancellation, keeping the uncancelled join byte-behavior-identical. That test MUST pass **file-unchanged**; `sdd-verify` asserts the file is unchanged, not merely that the suite is green. |
| 4 | AG-14.3's second and third Thens read as contradictory — one goroutine survives, none may remain | Med | The scope distinction (harness-owned vs. third-party tool goroutine) is written into the spec, not left implicit. `RequireNoGoroutineLeak` is scoped to the harness's own goroutines with the detached one explicitly accounted for. |
| 5 | A wind-down test passes by wall clock and goes flaky under `-race` | Med | Every synchronization point is an `agenttest.Gate`. The bound is the single legitimate clock use; a test that needs a sleep to pass is a design failure. |
| 6 | The `L2C-08` question is answered by silence rather than by decision | Med | Named as an explicit obligation with a recorded answer either way, mirroring AG-13's `sdd-design` treatment. |
| 7 | Post-shutdown harness state is the first cross-run state in Layer 2 and collides with AG-21's charter (`agent-loop-skeleton/spec.md:106`) | Med | AG-14's flag is *terminal and one-way* — it never resumes a run and holds no transcript. `sdd-design` must state that boundary so AG-21's cross-run state is not pre-empted. |
| 8 | Five spec deltas; one gets missed and a spec goes stale | Med | Enumerated with file and line above. This repo's known staleness clusters are owning-spec omission and un-back-annotated merges — both in play. `sdd-verify` re-reads each cited line against the shipped change. |
| 9 | `agenttest/tracetest` was named in the task brief but never opened during exploration | Low | `sdd-design` MUST Read it directly before assuming any shape, or record that AG-14 does not use it. |
| 10 | Review budget: forecast exceeds even the raised 1000-line bar | High (certain) | `size:exception` pre-authorized and extended for AG-14 specifically. See the forecast below. |

## Rollback Plan

Single revert of the AG-14 merge commit. The new signal-vocabulary file and the new tests are deleted; `harness.go` returns to unconditional `FailureCategoryUnavailable` + `RunOutcomeFailed` routing with no `Interrupt`/`Shutdown` surface; `loop.go`'s closed-channel branch returns to the unconditional `FinishReasonStop` normalization; `scheduler.go` returns to `context.Background()` at `:462` and an unbounded `wg.Wait()` at `:205`; `run_events.go` returns to three `RunOutcome` members; both substrate filters return to their pre-AG-14 filename lists; the five spec deltas are dropped; doc 0003's checklist line un-ticks and the counter returns to 13/24.

The revert is clean: nothing persists, no data migrates, no `go.mod`/`go.sum` change, nothing outside `backend/agent`. The one **externally visible** removal is the new `RunOutcome` member — but no consumer outside this change constructs it, because Layer 3 does not exist yet (`0003:110`), so the revert cannot orphan a live consumer. Re-running `cd backend/agent && make test` at the parent commit confirms zero regression.

Forward-looking cost: AG-14 blocks nothing directly (AG-15 and AG-16 are parallel), but AG-19's nested cancellation and AG-21's leak sweep both build on it. A revert is a scheduling consequence, not a correctness one.

## Review-workload forecast

| Component | Estimate |
|---|---|
| Signal vocabulary + sentinels (new file, this package's doc density) | 120–200 |
| `harness.go` — signals, cancel-cause holder, refusal state, cause-aware `failRun` | 150–250 |
| `loop.go` — cancellation-cause check at the closed-channel branch | 30–70 |
| `scheduler.go` — armed bound, detached-call report, `ctx` threading, cause-aware aborts | 150–280 |
| `run_events.go` — one member + `String()` case (Decision 1B) | 10–20 |
| Test files — 5 charter scenarios + leak assertions + bites | 600–900 |
| `loop_test.go` / `loop_hook_test.go` — filter widening | 20–50 |
| Doc 0003 checklist + traceability | ~10 |
| **Production + test subtotal (Go)** | **1090–1780** |
| SDD markdown — proposal, spec, **5 spec deltas**, design, tasks, apply-progress, verify-report | **800–1200** |
| **Total (authored, additions + deletions)** | **1890–2980** |

`Decision needed before apply: No` — `size:exception` is **PRE-AUTHORIZED** and extended beyond 1000 lines for AG-14 specifically, recorded here, one PR. `Chained PRs recommended: No` — but if `sdd-tasks` forecasts above ~2900, slice at the leaf boundary: AG-14.1 is independently deliverable, AG-14.2 depends only on it, and AG-14.3 depends on both (`0003:1442`). `400-line budget risk: High` — and high against the raised budget too.

The SDD markdown counts toward the attempt budget. `sdd-tasks` MUST forecast against the **full** diff, not the Go diff.

## Dependencies

- **AG-13** (archived, merged `52701436`) — `Harness`, `Run`, `Steer`, `failRun`, the continuation seam, the injected `*Scheduler`, `LeaveSinkOpen`.
- **AG-12** (archived) — `History.SynthesizeOrphans` (`history.go:270-300`), `CloseTurn`, the pairing invariant, the closed-route surface guard.
- **AG-10** (archived) — the parked-call lifecycle and its two `ctx.Done()` arms (`scheduler.go:637`, `:657`), `WakeParked`, `parked.remove`.
- **AG-09 / AG-07** — `Schedule`, `executeCall`, `Tool`, `Result`, `typedExecutionFailureFromError`, `Turn`, `CheckStream`.
- **AG-04** — `RunOutcome`, `RunEnd`, `NewRunEnd` (`run_events.go`).
- **Layer 1** — `ai.FailureCategoryCancellation`, and the fake's cancellation-fidelity contract R-CNF-011/R-CNF-012 (`agenttest/conformance_cancellation.go`, `agenttest/fake_provider.go:192-210`): on mid-stream cancellation the fake closes the channel **bare**, synthesizing no terminal event.
- **`agenttest`** — `NewProvider`, `Script`/`Step`/`Emit`/`Hold`, `Gate`, `RequireNoGoroutineLeak` (`stream_kit_leak.go:107`, serial-only per `:80`).
- **doc 0003:1371-1442** — the AG-14 charter and its three Gherkin leaves; **doc 0003:114-137** — the evidence gate and the test-substrate binding.

## Success Criteria — restated as verifiable checks

- [ ] `cd backend/agent && make test` green with `-race`; all five charter scenarios closed with recorded evidence
- [ ] **AG-14.1** Interrupt mid-turn with a provider stream and tools in flight: the provider call is cancelled per the fake's cancellation-fidelity contract, in-flight tools observe cancellation, orphan synthesis runs, and the run ends with the **interrupted** outcome — never `RunOutcomeFailed`, never `FailureCategoryUnavailable`
- [ ] **AG-14.1** A new prompt on the **same harness value** completes normally after an interrupt
- [ ] **AG-14.1** Interrupt during a permission suspension aborts it with a **typed** outcome naming the signal, and history closes cleanly with every open call resolved
- [ ] **AG-14.1** A second interrupt during wind-down changes nothing and panics nothing — asserted under `-race`, including no double channel close
- [ ] **AG-14.2** Shutdown winds down identically and the run-end outcome **says shutdown** — a value distinct from both interrupted and failed
- [ ] **AG-14.2** A prompt submitted after shutdown fails **typed**; the error is `errors.Is`-distinguishable from the interrupt sentinel, and the stream outcome is distinguishable too (both nouns of `0003:1379`)
- [ ] **AG-14.3** A cancellation-deaf `BlockingScriptedTool` whose release channel is never closed does not prevent the run from ending within the **documented bound**, with the bound's value stated in the spec
- [ ] **AG-14.3** The overrun call is reported typed — which tool, which call ID, still running — through a carrier that adds **no new `EventKind`**
- [ ] **AG-14.3** `RequireNoGoroutineLeak` proves no harness-owned goroutine survives the wind-down, with the one detached tool goroutine explicitly accounted for, not merely excluded; the test does **not** call `t.Parallel()`
- [ ] Bites, RED-recorded before GREEN: (a) delete the loop's cancellation check → an interrupt is recorded as a normal completed turn; (b) revert `failRun`'s cause routing → a cancellation reappears as failed/unavailable; (c) remove the bound → the deaf-tool test hangs
- [ ] `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` (`permission_protocol_test.go:1573`) passes **file-unchanged**; every AG-09/AG-10 scheduler test and `TestTurn_TwoSequentialTurnsShareNothing` pass file-unchanged
- [ ] `CheckStream` accepts every AG-14 stream **unmodified**; `stream_check.go` byte-unchanged
- [ ] Exactly the substrate files named in Decision 1's release differ; every other `R-LSK-004` file and every file under `backend/agent/src/ai/` is byte-unchanged; `go.mod`/`go.sum` unchanged
- [ ] Both substrate filters carry an identical exact-filename entry set with no wildcard, prefix, or directory pattern
- [ ] The `L2C-08` question is answered **in writing** in `design.md`, either way
- [ ] No new exported `History` method (`history_surface_guard_test.go` green untouched); import guard and ambient-authority guard pass with **zero** changes over both closures
- [ ] All five spec deltas written; each cited line re-read against the shipped change by `sdd-verify`
- [ ] `make lint` clean (after `golangci-lint cache clean`), `make build` clean, `make vuln-check` clean — `vuln-check` is **not** in `make all`
- [ ] doc 0003 line 2171 ticked, `:2207`/`:2257` back-annotated, milestone counters bumped to 14/24
