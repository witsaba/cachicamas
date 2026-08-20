# Proposal: AG-20 — Complete the hook taxonomy

> **Change**: `cachicamas-agent-hook-taxonomy` · **Milestone**: AG-20 (Layer 2 Wave 5, milestone 20 of 24; doc `0003:1864-1918`) — the **last** milestone of Wave 5
> **Worktree**: `cachicamas-worktrees/ag20-hook-taxonomy` · base `main@2a138b59` (PR #181, AG-19)
> **Artifact store**: hybrid (Engram + filesystem) · **Execution mode**: `auto`
> **Delivery**: `exception-ok` — a **single PR** carrying the change, the doc 0003 milestone-document update and the OpenSpec archive together (the AG-16/AG-17/AG-18/AG-19 house pattern)
> **Review budget**: 1000 changed lines, counted **excluding everything under `openspec/`** — the user designated `openspec/` a working folder. `sdd-tasks` and `sdd-apply` inherit this counting rule verbatim.
> **TDD**: strict, RED-first. Canonical runner: `cd backend/agent && go test -race -count=1 ./...`; the real uncached agent-module suite is ~170s, so a sub-second "pass" is a cache artifact, not evidence.
> **Closes**: **G11** (**R-17**) and **envelope invariant 3** jointly with AG-01.1; v2 § 6 seam 1 widens to the full taxonomy.
> **Depends on**: AG-08, AG-13, AG-18 (all merged); reads AG-01.1's decided mechanism and AG-19's shipped scope fence.
> **Exploration**: `explore.md` · Engram `sdd/cachicamas-agent-hook-taxonomy/explore`
> **ID prefix**: `R-HKS-` / `S-HKS-` / `NFR-HKS-` — **verified free**: zero occurrences worktree-wide (`sdd-spec` re-verifies before minting).

---

## Intent

Layer 2 has **one** hook point and **four** promised ones. `TurnOptions.PreRequestHook` shipped in AG-08 (`loop.go:90`), and AG-08's own spec made a commitment on this milestone's behalf, verbatim: *"the seam is a single callable on `TurnOptions` … **AG-20 widens to chain composition**"* (`agent-pre-request-hook/spec.md:19`). Three hook points — pre-compact, post-turn, session-start — exist today only as prose in `ai-contract-vocabulary/spec.md:331` (`V-OUT-13`, the row that carries **G11**).

The gap is not "three missing callbacks". It is that **the discipline that makes hooks safe has never been made mechanical**. `agent-event-envelope/spec.md:268` reserves envelope invariant 3 — *non-blocking observers* — as closed by "**AG-01.1 + AG-20.2**", and `R-AGE-008` (`agent-event-delivery/spec.md:123`) states the standard AG-20.2 must meet:

> "A statement of the obligation, a convention, a review rule, or documentation alone MUST NOT satisfy this requirement."

Today nothing in Layer 2 enforces that an observer cannot stall the streaming path, because Layer 2 has no observers. The moment three of them land, the property becomes falsifiable — and a Layer 3 application (doc 0004 CO-24) will wire concrete hooks into it. AG-20 is the last point at which the discipline can be built into the *shape* of the surface rather than asked for in a doc comment. Wave 5 exits here, and AG-21 ("concurrency, backpressure, leaks") inherits whatever posture this milestone freezes.

Two facts make the deadline real:

| Fact | Where | Why it forecloses a later fix |
|---|---|---|
| The observing families' function types are the enforcement | this proposal, D5 | Once a hook type with a return value ships, removing that return is a **breaking** change to every Layer 3 hook |
| The stalled-observer report's carrier is chosen once | D3 | Seven live assertions fix the event registry at 25 kinds; a 26th minted later costs the same seven deltas plus a Layer 3 migration |

---

## Resolved decisions — the six the exploration left open, plus one it did not find

`sdd-design` may overturn any of these **on stated evidence**; it may not resolve them by silence.

### D1 — The registration surface, and the fate of AG-08's shipped field

**Decided: one exported `Hooks` value type, registered on `Harness.Hooks`, transported into `Turn` through a new `TurnOptions.Hooks` field that the harness fills in the same derivation it already uses for `Continuation`. A non-zero `Harness.Turn.Hooks` is refused typed at `Run` entry. `TurnOptions.PreRequestHook` is kept, unamended, as chain element zero.**

Three constraints, each **checked in this worktree rather than assumed**, jointly force this shape:

| Constraint | Evidence | Consequence |
|---|---|---|
| `*Harness` has **exactly five** exported methods, asserted twice | `scope_fence_test.go:102-105` (`NumMethod() != 5`, AG-19's `R-DEL-009`) and `harness_test.go:1031` (the exact named set) | A `RegisterHook(...)` API is **unavailable by fence**. The surface must be a field. |
| "A seam belongs where its consumer is" | `agent-retry-failover/spec.md:169` — `Failover` was put on `Harness` and **explicitly not** on `TurnOptions`, with that reason | Session-start, post-turn and pre-compact are consumed by `Harness`; registering them on `TurnOptions` would contradict a shipped requirement's stated rule |
| `TurnOptions` is the established zero-value extension point | AG-08 `PreRequestHook`, AG-09 `Tools`, AG-10 `PermissionPolicy`, AG-13 `Continuation` (`loop.go:63-141`) | Pre-request's consumer **is** `Turn`; its transport belongs there, and a fifth zero-value field is the house pattern |

The transport is not a second registration surface, and that is enforced rather than documented. `Run` already overwrites `turnOpts.Continuation` on a copy of `h.Turn` (`harness.go:640-646`); it will assign `turnOpts.Hooks = h.Hooks` at the same site. So that the overwrite can never be **silent**, a non-zero `h.Turn.Hooks` is a typed refusal before any identity is minted and any event is emitted — the exact posture `validateContinuation` already ships for a half-configured continuation (`S-LSK-014`). Symmetrically, `CompactionRequest.Options.Hooks` joins `Tools` and `Continuation` in `runCompaction`'s misplaced-options rejection (`compaction.go:269-272`): the pre-compact chain reaches `runCompaction` as an explicit unexported parameter from the harness, never smuggled through the request's own options.

**The shipped field is kept, and AG-08's promise is honoured additively.** `TurnOptions.PreRequestHook`, when non-nil, runs **first**, and its output feeds `Hooks.PreRequest[0]`. Every AG-08 test stays byte-stable; `R-PRH-001` and `R-PRH-005`'s nil-default byte-identity survive as the "no singular hook **and** empty chain" case. The field is marked **superseded in prose only** — not with Go's `Deprecated:` marker, which would make every existing internal reference a lint finding for no behavioural gain. Removal is `AG-23`'s (the frozen Layer 3 surface), recorded below as deferred, not decided here.

**Rejected: a `Hooks` field on `Harness` only.** Pre-request fires inside `Turn`, which never sees the harness value. That shape needs a second carrier anyway, and the second carrier is what "one registration surface" forbids.

### D2 — Session-start's "once per harness"

**Decided: a one-way `sessionStarted` latch on the `Harness` value, set at `Run` entry under the existing `signalMu`, after the shutdown check and before the run bracket opens. "Once per harness" becomes literally true, and the spec MUST say "once per `Harness` **value**".**

The exploration is right that no harness identity exists: `Harness` is value-form with no constructor, and `Run` mints a fresh `run-hrn-` id each call (`harness.go:40-42`, `:467`). But the charter does not need an *identity* — it needs a *latch*, and this type already carries one of exactly the same class:

> `shutdown` is the terminal, one-way refusal flag `Shutdown` latches (`R-CAN-005`). **It is per-value bookkeeping only — no transcript, never resumed — and does not pre-empt AG-21's cross-run state** (`harness.go:126-130`).

`sessionStarted` is that sentence with one word changed. It reuses the mutex the same comment argues for over `sync.Once` (`harness.go:111-117`), and `Harness` already embeds a `sync.Mutex`, so `go vet`'s copylocks check already makes a post-use copy of a `Harness` value a build-time finding — the latch cannot be laundered by copying.

**The reuse case is the motivating case, not an edge case.** `History` is caller-owned and survives across `Run` calls; a serially reused harness is a shipped, tested shape (`R-CAN-002`'s queue reopen at `harness.go:431-433`). That reuse is exactly *one session, many runs*. Firing session-start per `Run` would make the hook indistinguishable from `run_start` and would make the charter sentence false; option (a) from the exploration is rejected for that reason.

**Nested and delegated runs: session-start fires for the child, once.** A child run is a brand-new `Harness` struct literal (`delegation_seam_test.go:341`), so it is a distinct value with its own latch. This is the right answer product-side as well as mechanically: a subagent **is** its own session, with its own transcript and its own `run-hrn-` identity, and AG-19 already decided that the two conversations are separated by walking parents rather than merged. A child that inherits no `Hooks` value fires nothing, so nothing surprising happens to AG-19's shipped fixtures.

**No charter amendment is needed** — only a disambiguation the spec must carry: *"once per `Harness` value"*, with the two consequences (serial reuse does not re-fire; a delegated child fires its own) stated as requirement text, not left to inference.

### D3 — The stalled-observer typed report is **not** stream-visible

**Decided: no 26th `EventKind`. The report is a Go-side typed value delivered off the streaming path.**

The count argument is real and larger than the exploration recorded — there are **seven** sites, and the seventh is a *live Go test on `main`*, not a spec sentence:

| # | Site | Shape |
|---|---|---|
| 1 | `R-AEV-010` / `S-AEV-090` | "exactly 25 kinds … and no further kind of any name" |
| 2 | `R-AEV-013` / `S-AEV-120` | the same, restated for AG-06 |
| 3 | `agent-event-envelope/spec.md:144` + `S-AEV-075` | "No new `EventKind` is registered for failure … the guard stays at **25 kinds**" |
| 4 | `R-CST-007` (`agent-cost-events/spec.md:174`) | AG-16's fence: token figures, **no new kind** |
| 5 | `R-DEL-009` / `R-DEL-010` (`agent-delegation-readiness/spec.md:236`, `:252`) | AG-19's fence and its closed-sequence table |
| 6 | `S-AEV-092` / `S-AMT-081` / `S-APE-081` | the bites that make the fence bite by count |
| 7 | **`scope_fence_test.go:87`** | `if got := len(agent.EventKinds()); got != 25 { t.Errorf(...) }` — **shipped, running on every `make test`** |

Site 7 alone means a 26th kind turns `main` red on the day it lands. Minting one also requires editing `event.go`, `event_descriptor.go` and `event_registry_test.go`, all three on `R-LSK-004`'s forbidden list **and** on AG-19's `del024ByteUnchangedFiles()` (`scope_fence_test.go:21-29`), whose merge-base diff guard is live on this branch.

**But the decisive argument is structural, not budgetary, and it must be the one the spec records.** `R-AGE-008`'s closure test is that *"every path from that stalled consumer's receive back toward the canonical producer terminates at the forwarding activity that privately owns that consumer's carrier, and none reaches the canonical producer"* (`S-AGE-010`). An event announcing the stall **is a path from the stalled observer back onto the producer's stream**. Publishing it would contradict the very invariant AG-20.2 exists to close. A budget argument can be overturned by a budget; this one cannot.

Precedent agrees in both directions: AG-11 carried typed failure on the existing `turn_end`/`run_end` outcomes; AG-16 carried cost figures with no new kind.

**Carrier instead**: a typed value (working name `ObserverStall`) carrying the hook point, the registration index, the run identity and the stall's discriminator (stalled / panicked), delivered to a nil-defaultable reporter registered on the same `Hooks` value, invoked **on the observer lane** and never on the streaming path. A nil reporter reports nothing — the standard nil-default posture of `Failover`, `ContextStrategy` and `PermissionPolicy`, and `sdd-spec` must state it in exactly those terms so "eventually reported" is not read as an unconditional emission promise.

### D4 — What "eventually" means

**Decided: structural, not temporal, following `R-RUN-010` explicitly. No wall clock, no timeout, no deadline, no join on the stalled observer.**

`R-RUN-010` (`agent-run-driver/spec.md:229`) forbids exactly the temporal answer on a structurally similar suspended-call seam — *"The harness adds **no third path and no timeout**. A run whose policy defers and whose owner neither wakes nor cancels does not terminate — **by design**"* — and `:238` adds *"No wall clock, no timeout, no sleep."* Six capabilities carry an NFR banning sleep/timeout synchronization in tests (`NFR-CST-002`, `NFR-CTX-002`, `NFR-CMP-002`, `NFR-RUN-002`, `NFR-CAN-002`, `NFR-RTY-002`). There is no wall-clock timeout anywhere in the module. Departing here would be the first, and it would be arbitrary: a "slow" threshold is a Layer 3 policy number, and `R-AGS-015` forbids Layer 2 deciding Layer 3's content.

> **The definition.** *Eventually* = **at the run's terminal boundary** — the same structural moment `run_end` is constructed — the observer lane's **outstanding set** is snapshotted: every observing-hook invocation that has been dispatched and has not returned, plus every invocation still queued behind it. Each is reported typed and attributed by hook point, registration index and run identity. The run neither waits for nor joins any of them.

This is deterministically testable with the primitive the module already mandates:

- given a post-turn observer held open by `agenttest.Gate` — entering the gate **proves** the invocation started and **proves** it has not returned;
- when the run reaches its terminal boundary and the consumer observes `run_end`;
- then the reporter received exactly one typed report naming that hook point and index, and the recorded stream is byte-identical to the same script with no hooks installed and is `CheckStream`-valid;
- and only then is the gate released, after `Run` has already returned.

Two consequences MUST be stated rather than discovered:

1. **A permanently stalled observer leaks a goroutine, by design, and that is the caller's leak.** Any goroutine-baseline assertion in this milestone's own tests (the `S-PRH-007` / AG-14 leak-test pattern) MUST release its gate before sampling `runtime.NumGoroutine()`. AG-21 inherits this knowingly.
2. **An observer that finishes just after the snapshot is reported once and completes anyway.** The report is an honest statement about the terminal boundary, not a verdict about the hook.

**Mandatory anti-vacuity bite**: with the stall removed (the observer returns immediately), the report set MUST be **empty**. Without it, an implementation that reports every observer would pass.

### D5 — The mutation contract, made unconstructible rather than discouraged

**Decided: enforcement by type. An observing hook's function type has no return values at all.**

```
mutating   PreRequestHook  func(context.Context, ai.Request)      (ai.Request, error)   // AG-08's shape, unchanged
mutating   PreCompactHook  func(context.Context, CompactionPlan)  (CompactionPlan, error)
observing  PostTurnObserver     func(context.Context, PostTurnReport)        // no return
observing  SessionStartObserver func(context.Context, SessionStartReport)    // no return
```

An observer **cannot** signal a mutation or a failure back to the runtime, because there is no channel through which to signal one — the wrong thing is unconstructible, matching this repo's stated house rule ("unreachable rather than merely discouraged", AG-18/AG-19). The payload types are values with unexported fields and read-only accessors, the module's established payload shape (`TurnEnd`, `CostFigures`), so an observer cannot reach shared state through its argument either. `CostFigures` is **carried unchanged** — `cost_events_test.go:199-231` pins its field names, count and kinds by reflection, and this change must not disturb that pin.

**Sub-decision, not in the charter and not in the exploration: a panicking observer is recovered.** An observing hook runs on the observer lane's own goroutine; an unrecovered panic there kills the process. "Observers never block the streaming path" would then be true in the worst possible way. The lane recovers, reports through the **same** typed surface as a stall (discriminated), and continues with the next queued invocation — the scheduler's own tool panic-recovery posture (`runToolWithWindDown`), applied one layer out. **Mutating hooks are not recovered**: they run inline on the caller's goroutine exactly as AG-08 shipped, and changing that is out of scope and stated as such.

**Mutating-hook failure is typed and attributed on both points.** Pre-request keeps `R-PRH-003`'s `*ai.PreStreamFailure` abort, widened so the attribution names *which chain element* failed. Pre-compact fails through the compaction bracket's existing failure arm (`emitCompactionFailedArm(..., TurnOutcomeAborted, ...)`), so `R-CMP-010`/`R-CMP-013`'s "a compaction failure is visible on the stream and nowhere else, and never interrupts the run" holds unamended.

### D6 — The pre-compact splice point, chosen so scenario 2 needs no new validation code

**Decided: inside `runCompaction`, after the naive cut is resolved and the span derived, before the provider call — and the adjusted plan is re-resolved through `resolveCut` before use.**

The exploration proposed splicing at the harness's verdict gap (`harness.go:550`). That is worse on two counts: it is **outside** the compaction bracket, so a hook failure has no typed arm to land on; and it can only hand the hook an `int`, not a span. Splicing inside `runCompaction` between `hist.markSpan(cut)` (`compaction.go:285`) and `buildLoopRequest` (`:292`) gives the hook a real `CompactionSpan`, keeps it strictly before `req.Provider.Stream` (`:297`), reaches **both** doors (`R-CMP-001`'s strategy door and on-demand door) through the one operation, and lands a hook failure on the arm that already exists.

Charter scenario 2 — *"the adjusted span is revalidated by the invariant-safe surgery before use"* — is then **literal**: the adjusted cut is fed back through `resolveCut` (`compaction.go:51-79`), the retraction-only, structurally terminating surgery `R-CMP-004` specifies. No new validation code is written.

Two obligations this creates, both for `sdd-design`:

1. **`resolveCut` idempotence must be proven, not assumed.** On the unchanged-plan path the second resolution must be a no-op. It is plausible (`resolveCut` retracts to a fixed point) but nothing asserts it today.
2. **`R-CMP-004`'s "MUST NOT expand the cut forward under any condition" must be read correctly, and the spec must say so.** That sentence governs **resolution**, not the **request**. A mutating pre-compact hook re-designates the boundary; resolution then retracts from the new request exactly as before. Left unstated, a reviewer will read a forward-adjusting hook as violating a shipped MUST — and would be right to.

### D7 — Post-turn's payload and, harder, its firing sites

**Decided: `PostTurnReport{ Run, Turn, Outcome, Cost, Attempts }`, fired once per *logical* turn, and the spec MUST enumerate every exit from the logical-turn loop with a yes/no for each.**

The report is assembled from data the harness already has or already reads:

| Field | Source | Note |
|---|---|---|
| `Turn` | `capturedTurnID`, captured in the per-attempt forwarder (`harness.go:623-625`) | existing pure read |
| `Cost` | a **per-logical-turn** accumulator beside `total`, folded in the same forwarder (`:633-635`) | a **local in `Run`'s frame**, never a `Harness` field — `R-CST-004` forbids the latter and `total`'s own comment (`harness.go:470-478`) states the rule |
| `Outcome` | the forwarded `turn_end`'s `TurnOutcome` (`turn_events.go:202`) | existing vocabulary; **no new member** |
| `Attempts` | the inner attempt loop's counter (`:613`) | makes the retry semantics of `Cost` legible |

**Retries make "the turn's cost" ambiguous and the spec must not leave it so**: a logical turn may make several `Turn` invocations, each emitting its own `cost_turn` inside its own bracket. The turn's cost is the **sum over that logical turn's attempts**. Stated, because a reviewer would otherwise reasonably read it as the last attempt's.

**The firing sites are where this milestone can hide a bug between two satisfied requirements.** Coverage of a failure path proves the path runs, not that it does the right thing. `sdd-spec` MUST enumerate all of them:

| Exit from the logical-turn loop | `harness.go` | Fires? |
|---|---|---|
| Success — after `hist.closeTurnMarked` | `:733-737` | **yes**, with the turn's outcome |
| Turn failed — before `failRun` | `:700-723` | **yes**, outcome aborted |
| Interrupt/shutdown with a turn in flight — before `windDownRun` | `:667-669`, `:693-695` | **yes** (proposal's decision; `sdd-design` may overturn on evidence) |
| Iteration-boundary signal, **no** turn ran this iteration | `:502-504` | **no** — there is no turn to report |
| Compaction's own bracket | `compaction.go` | **no** — compaction is not a logical turn; pre-compact is its hook |
| History-append failure before the first turn | `:486-488`, `:506-510` | **no** |

---

## What changes

### Production (small, additive, and the whole of it)

- **`hooks.go`** (new): the `Hooks` type, the four function types, the three payload types, the typed observer report, the observer lane (unbounded FIFO, one drain goroutine per run, registration-order dispatch, panic recovery), and the terminal-boundary snapshot.
- **`loop.go`** (modified): `TurnOptions.Hooks` field; `applyPreRequestHook` (`:739-748`) becomes chain composition with per-element attribution.
- **`harness.go`** (modified): `Harness.Hooks` field; the `sessionStarted` latch beside `shutdown`; the typed refusal of a non-zero `h.Turn.Hooks`; `turnOpts.Hooks` assignment beside `turnOpts.Continuation`; a per-logical-turn cost accumulator and outcome capture in the existing forwarder; the post-turn fire sites of D7; lane creation and terminal snapshot.
- **`compaction.go`** (modified): the pre-compact splice of D6; `Hooks` added to the misplaced-options rejection; `runCompaction` gains one unexported parameter.

### Not changed, and this is assertable

`event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `cost_usage.go`, `turn_events.go`, `failure.go`, `sequence.go`, `run_events.go`, `doc.go`, `doc_contract_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, `scheduler.go`, `delegation_seam.go`, `go.mod`, `go.sum`, **all of `backend/agent/src/ai/`**, and `agenttest` (byte-unchanged by default; `agenttest.Gate` is used, not widened). No new `EventKind` — the guard passes at 25. No new `TurnOutcome`, no new `CostLabel`. `Harness` gains **no** exported method; `Turn` and `Run` keep their signatures.

---

## Scope

### In

- **AG-20.1** — the three remaining hook points at the splice points of D2/D6/D7; the one `Hooks` registration surface of D1; the pre-request chain honouring AG-08's promise; deterministic registration-order dispatch at every point; pre-compact adjustment revalidated by `resolveCut`; typed, element-attributed failure on both mutating points.
- **AG-20.2** — observer asynchrony as a mechanical test against `agenttest.Gate`; the structural definition of "eventually" (D4); the typed stall/panic report (D3/D5); envelope invariant 3 closed jointly with AG-01.1.
- The new capability spec, the deltas below, the doc 0003 milestone-document update (AG-20 ticked, counter to 20/24, Wave 5 complete), and the OpenSpec archive — **same PR**.

### Out — quoting the charter verbatim

> "**Out of scope:** Any concrete hook implementation (Layer 3 wires them)." (`0003:1874`)

| Also deferred | Owner, and why the deferral is safe |
|---|---|
| Any concrete hook (cache breakpoints, compaction policy, telemetry) | **Layer 3**, doc 0004 CO-24.1/CO-24.2 — `agent-v1-scope` `S-AGS-048` already names those nodes |
| Removing `TurnOptions.PreRequestHook` | **AG-23** (the frozen Layer 3 surface). Kept and superseded here, never broken |
| Multi-consumer fan-out / observer attachment machinery | **AG-01.1 decided it; nobody has built it.** AG-20 makes a *hook* non-blocking; it does not ship `R-AGE-009`'s consumer fan-out |
| Hook deregistration, priority, filtering, or conditional registration | **Not this milestone.** Registration order is the whole ordering contract (`0003:1898-1901`) |
| Any wall-clock timeout, deadline or "slow hook" threshold | **Never in Layer 2.** `R-RUN-010`; a threshold is Layer 3 policy (`R-AGS-015`) |
| Panic recovery for **mutating** hooks | **Not this milestone / AG-21.** They run inline on the caller's goroutine, exactly as AG-08 shipped |
| Cross-run hook state, or a session concept outliving a `Harness` value | **AG-21.** The latch is per-value bookkeeping, the `shutdown` shape (`harness.go:126-130`) |
| A new `EventKind`, `TurnOutcome` or `CostLabel` | **Never in AG-20** (D3) |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2** (`R-RUN-012`) |

---

## Capabilities and delta-spec plan

`sdd-spec` finalises the requirement text; this proposal fixes the shape.

### New

- **`agent-hook-taxonomy`** — kebab-case, matching the `agent-*` convention; directory does not exist. IDs `R-HKS-0NN` / `S-HKS-0NN` / `NFR-HKS-0NN`, prefix verified free. **State the allocated range, never a total** — a total is defended by no test and goes silently false on the next append (`S-LSK-020`, and this repo's known count-assertion drift class).

Expected families: `R-HKS-001` the one registration surface and the two type families · `R-HKS-002` the pre-request chain and AG-08 compatibility · `R-HKS-003` pre-compact, its splice and its revalidation · `R-HKS-004` post-turn, its payload and its complete firing-site enumeration · `R-HKS-005` session-start, once per `Harness` value · `R-HKS-006` deterministic registration-order dispatch · `R-HKS-007` observer asynchrony as structure · `R-HKS-008` "eventually" and the typed stall/panic report · `R-HKS-009` mutating-hook failure, typed and element-attributed · `R-HKS-010` the scope fence (the `R-DEL-009` pattern) · `R-HKS-011` closed-sequence safety (the `R-DEL-010` / `R-CMP-014` table pattern) · `R-HKS-012` inertness: a zero-value `Hooks` is byte-identical to today on every path.

### Spec blast radius — every requirement touched, with its delta operation named

Grepped this phase for absolute quantifiers ("exactly N", "only", "iff", closed enumerations) across `openspec/specs/`. **Certainty is stated per row; `sdd-spec` MUST open each cited line.**

| Capability | Requirement | Op | Why |
|---|---|---|---|
| `agent-pre-request-hook` | `R-PRH-002` | **MODIFIED** | *"The hook's returned `ai.Request` (**and only that value**) is what flows to `provider.Stream`"* (`:35`) **goes false** under a chain: element *n*'s output flows to element *n+1*. Re-scoped to "the chain's final output". **Certain** |
| `agent-pre-request-hook` | `R-PRH-005` | **MODIFIED** | *"When `opts.PreRequestHook` is nil, the system SHALL skip the seam"* (`:59`) goes false when the chain is non-empty. Condition becomes nil singular **and** empty chain. **Certain** |
| `agent-pre-request-hook` | `R-PRH-003` | **MODIFIED** | Failure attribution widens to name the failing chain element. **Certain** |
| `agent-pre-request-hook` | `R-PRH-007`, Purpose `:19`, `:98`, `:113` | **MODIFIED** | Determinism extends over the composed chain; the `:19` promise and the `:98` non-requirement close; `:113` "Blocks: AG-20" back-annotated. **Certain** |
| `agent-pre-request-hook` | `R-PRH-001`, `R-PRH-004`, `R-PRH-006` | **unamended, shown still true** | Field kept; value-semantics and prefix stability unaffected. **Certain** |
| `agent-event-envelope` | invariant-3 row `:268` | **MODIFIED** | Closes jointly with AG-01.1. **Certain** |
| `agent-event-delivery` | `R-AGE-008` / `S-AGE-010` | **MODIFIED** | AG-20.2 delivers the mechanical half; the "no path back to the producer" trace is what D3 preserves. **Certain** |
| `agent-compaction` | `R-CMP-004` | **MODIFIED** | Distinguish the **requested** cut (hook-adjustable, either direction) from **resolution** (retraction only, `:115` unchanged). **Certain** |
| `agent-compaction` | `R-CMP-001`, `R-CMP-013`, `R-CMP-014` | **MODIFIED** | One operation, two doors, one splice; the misplaced-options rejection gains `Hooks`; the no-new-vocabulary fence and the inertness claim re-verified for a hook-carrying path. **Certain** |
| `agent-run-driver` | `R-RUN-010` | **MODIFIED (back-annotation)** | Record that AG-20 adds **no** timeout and no third path, and that `TestPermission_WakeParked_…_NoDeadline` stays file-unchanged. **Certain** |
| `agent-run-driver` | `R-RUN-001`, `R-RUN-003`, `R-RUN-004` | **MODIFIED** | The value-form posture gains one unexported latch (the `shutdown` class, `:72`); the forwarder gains two pure reads and still rewrites/synthesizes/suppresses/reorders nothing; harness-vs-run identity is disambiguated by D2. **Certain** |
| `agent-cost-events` | `R-CST-004` | **unamended, shown still true** | The per-logical-turn accumulator is a `Run`-frame local, not a harness field; no second writer on the event path. **Certain** |
| `agent-turn-termination` | `TurnOutcome` vocabulary | **MODIFIED (back-annotation)** | Consumed by `PostTurnReport`; **no member added**. **Certain** |
| `agent-delegation-readiness` | `R-DEL-009`, `R-DEL-010` | **MODIFIED (back-annotation)** | AG-20 keeps the registry at 25, adds no `Harness` method, and leaves all seven fenced files byte-unchanged — so `scope_fence_test.go` stays green **unedited**. **Certain** |
| `agent-v1-scope` | G11 / **R-17** row | **MODIFIED** | Discharged. **Certain** |
| `agent-loop-skeleton` | `R-LSK-002`, `R-LSK-004` | **MODIFIED (back-annotation) / CONDITIONAL** | Nil-path byte-stability re-affirmed via `R-HKS-012`. `R-LSK-004` needs an exact-filename release **only if** design elects to edit `doc.go` for a package-guarantee row — the AG-15/AG-17/AG-19 shape is to request none, and this proposal requests none. **Conditional** |
| `ai-contract-vocabulary` | `V-OUT-13` row `:331` | **MODIFIED (back-annotation)** | The row's four hook points and "observers never synchronous" clause are realised. Spec text only — no `src/ai/` code is touched, so `R-RUN-012` holds. **Likely** |
| `agent-context-strategy` | `:302` | **no delta** | *"The pre-request hook derives its request downstream of this seam"* remains true for a chain. **Certain** |

---

## Approach

1. **Write the new capability spec and the deltas first.** `R-PRH-002` and `R-PRH-005` go false the moment the chain lands; the amendment must precede the code, not chase it.
2. **`hooks.go`** — the types, the observer lane, the snapshot — tested in isolation: registration order deterministic; a no-return type cannot signal; a panicking observer is reported and the lane continues.
3. **The pre-request chain** in `loop.go`, then re-run AG-08's whole suite **unedited** as the compatibility proof.
4. **Session-start + the latch + post-turn's six firing sites** in `harness.go`, with the reuse and delegated-child scenarios.
5. **Pre-compact** in `compaction.go`, with the `resolveCut` idempotence proof and the forward-adjustment scenario.
6. **AG-20.2** — the gate-held observer, the byte-identical stream, the terminal snapshot, and the anti-vacuity bite.
7. **Doc 0003 update + archive**, same PR.

**Bites are not optional.** At minimum: (a) observer returns immediately ⇒ the stall report MUST be **empty** (D4's anti-vacuity); (b) invoke the observer synchronously on the streaming path ⇒ the "delivery unimpeded" assertion MUST fail as an **assertion**, not a hang — design owes a shape that fails rather than deadlocks; (c) dispatch the chain in reverse ⇒ the ordering scenario MUST fail; (d) skip the second `resolveCut` after a hook adjustment ⇒ the invariant-safe scenario MUST fail with a split pair; (e) fire session-start per `Run` ⇒ the serial-reuse scenario MUST fail with two fires. Each RED-recorded **before** its GREEN, with `-count=1`, then reverted.

---

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/hooks.go` (new) | **New** | `Hooks`, four function types, three payloads, typed observer report, observer lane, terminal snapshot |
| `backend/agent/src/agent/loop.go` | **Modified** | `TurnOptions.Hooks`; `applyPreRequestHook` → chain composition with attribution |
| `backend/agent/src/agent/harness.go` | **Modified** | `Harness.Hooks`; `sessionStarted` latch; typed refusal; `turnOpts.Hooks`; per-turn cost + outcome capture; post-turn fire sites; lane lifecycle |
| `backend/agent/src/agent/compaction.go` | **Modified** | Pre-compact splice; `Hooks` in the misplaced-options rejection; one new unexported parameter |
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | **New** | Both leaves' scenarios and five bites |
| `openspec/specs/agent-hook-taxonomy/` | **New capability** | Via the change's delta folder |
| `openspec/specs/{agent-pre-request-hook, agent-event-envelope, agent-event-delivery, agent-compaction, agent-run-driver, agent-cost-events, agent-turn-termination, agent-delegation-readiness, agent-v1-scope, agent-loop-skeleton, ai-contract-vocabulary}` | **Delta** | Eleven; `agent-loop-skeleton`'s `R-LSK-004` release conditional and **not** requested |
| `docs/architecture/milestones/0003-…md` | **Modified** | AG-20 status, delivery table, checklist row `:2163`; counter to 20/24; Wave 5 complete |
| `event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `turn_events.go`, `failure.go`, `doc.go`, `scheduler.go`, `agenttest/`, `go.mod`, `go.sum`, **all of `src/ai/`** | **NOT TOUCHED** | No new kind, no Layer 1 edit, no new dependency, no `agenttest` widening |

---

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | **The chain lands and `R-PRH-002`/`R-PRH-005` are left saying the opposite of the code.** Both are phrased as closed claims ("and only that value", "when nil … skip the seam") and both go false on the first chained call | **High** | Named MODIFIED above with the falsifying clause quoted. `sdd-spec` writes the amendment **before** step 3; `sdd-verify` opens `:35` and `:59` against the shipped code — a citation is not evidence |
| 2 | **A 26th `EventKind` is minted for the stall report**, turning `main` red at `scope_fence_test.go:87` and forcing seven deltas | **Med-High** | D3, decided structurally (an event about the stall **is** the path back to the producer that `S-AGE-010` forbids), not on budget |
| 3 | **"Eventually" is implemented as a timeout**, becoming the module's first wall clock and contradicting `R-RUN-010` | **Med-High** | D4's snapshot definition is normative before any code, with the gate-based test shape and bite (a) |
| 4 | **Post-turn is wired only on the success path**, so aborted and wound-down turns silently never fire — a bug that hides between two individually satisfied requirements | **High** | D7's six-row enumeration is normative; `sdd-spec` MUST carry it as requirement text with a scenario per firing row |
| 5 | The stalled-observer test asserts by elapsed time or flakes under `-race` | Med | `agenttest.Gate` only; ordering asserted by observed sequence on the serial lane; six shipped NFRs already ban sleep/timeout synchronization |
| 6 | **A permanently stalled observer leaks a goroutine and a leak assertion in this milestone's own suite goes red** | **Med-High** | D4 consequence 1: release the gate before sampling `NumGoroutine()`; stated so AG-21 inherits it knowingly |
| 7 | `resolveCut` is assumed idempotent and the second resolution silently moves the cut | Med | D6 obligation 1: proven with its own scenario, not assumed |
| 8 | A forward-adjusting pre-compact hook is read as violating `R-CMP-004`'s "MUST NOT expand forward" | **Med-High** | D6 obligation 2: the request/resolution distinction is stated in the delta, not left to a reviewer |
| 9 | A `Harness` method is added for the report or for registration, breaking **two** shipped pins at once | Med | D1: the surface is a field; `scope_fence_test.go:102-105` and `harness_test.go:1031` must both pass **unedited** |
| 10 | One of eleven deltas is missed — **no Go test enforces a back-annotation** | **Med-High** | Enumerated with file and line above. `agent-compaction`, `agent-turn-termination`, `agent-run-driver` and `ai-contract-vocabulary` were **not** in `explore.md` and were found by grep in this phase |
| 11 | A panicking observing hook kills the process; nobody specified it because the charter did not name it | Med | D5 sub-decision: recovered on the lane, reported through the same typed surface, discriminated from a stall |
| 12 | Evidence recorded from a **cached** run | Med | `-count=1` in the header and every acceptance line; record wall-clock duration |
| 13 | The registration surface is judged "not one surface" because `TurnOptions.Hooks` also exists | Low-Med | D1: one **type**; the transport is filled by the harness and a caller-set `h.Turn.Hooks` is a **typed refusal**, not a silent overwrite |

---

## Rollback plan

**Single revert of the AG-20 merge commit.** `hooks.go` and all new test files are deleted; `loop.go` loses one field and `applyPreRequestHook` returns to its AG-08 four-line form; `harness.go` loses one field, one latch and the fire sites; `compaction.go` loses the splice and one parameter; the deltas are dropped; doc 0003's AG-20 line un-ticks and the counter returns to 19/24.

**The revert is clean, and the reason is structural.** Nothing persists across processes. No data migrates. No `go.mod`/`go.sum` change. No Layer 1 file touched. No `EventKind` added or removed. No exported signature narrowed — every addition is a zero-value field or a new type, so no existing caller compiled against a removed symbol. The only consumer of the hook surface is `package agent_test`; doc 0004's Layer 3 wiring does not exist yet.

**Forward-looking cost**: reverting re-opens G11 / R-17, returns envelope invariant 3 to its "AG-01.1 + AG-20.2" deferred row, leaves AG-08's `:19` promise outstanding, and blocks Wave 5's exit. Scheduling consequence, not correctness.

---

## Review-workload forecast

**Counting rule**: additions + deletions, **excluding every path under `openspec/`**. SDD markdown still counts toward the **attempt budget** — a different budget; `sdd-tasks` must not conflate them.

| Component | Estimate (authored, non-`openspec`) |
|---|---|
| `hooks.go` — types, payloads, lane, snapshot, report | 260–380 |
| `loop.go` — field + chain composition + attribution | 40–70 |
| `harness.go` — field, latch, refusal, capture, six fire sites, lane lifecycle | 90–150 |
| `compaction.go` — splice, misplaced-options, parameter | 40–70 |
| AG-20.1 scenarios (three charter scenarios across four points) | 450–700 |
| AG-20.2 scenarios (asynchrony, snapshot, panic) | 200–320 |
| Five bites | 180–300 |
| Doc 0003 status line, delivery table, checklist | 25–45 |
| **Counted total** | **1285–2035** |
| *Uncounted (attempt-budget relevant)*: proposal, new capability spec, 11 deltas, design, tasks, apply-progress, verify-report, archive report | *1200–1900* |

`Decision needed before apply: No` — `exception-ok`, single PR, `size:exception` pre-accepted at 1000 counted lines with a user-pre-accepted extension if the milestone genuinely needs it.
`Chained PRs recommended: No` — held in reserve; the slicing boundary is the Approach ordering: **U1** = `hooks.go` + the pre-request chain + AG-08 compatibility · **U2** = session-start + post-turn · **U3** = pre-compact + AG-20.2.
`400-line budget risk: High`

---

## Dependencies

- **AG-08** (archived, PR #167) — `TurnOptions.PreRequestHook` (`loop.go:90`), `applyPreRequestHook` (`:739-748`), `R-PRH-001…007`, and the `:19` chain-composition promise this change discharges.
- **AG-13** (archived) — `Harness.Run`'s logical-turn loop, the per-attempt forwarder (`harness.go:616-638`), `TurnContinuation`, `validateContinuation`'s typed-refusal precedent.
- **AG-18** (merged, PR #180) — `runCompaction` (`compaction.go:252-342`), `resolveCut` (`:51-79`), the misplaced-options rejection (`:269-272`), `markSpan`, `CompactionSpan`, `R-CMP-001…014`.
- **AG-01.1** (archived) — `R-AGE-008`'s decided decoupling mechanism and `S-AGE-010`'s no-path-back trace, the standard AG-20.2 is measured against.
- **AG-14 / AG-16 / AG-19** (merged) — the `shutdown` latch precedent (`harness.go:126-130`), `R-CST-004`'s run-frame-local rule, and the live scope fence (`scope_fence_test.go`) this change must leave green **unedited**.
- **`agenttest.Gate`** (`fake_gate.go:20`) — the mandated no-sleep synchronization primitive. **Byte-unchanged** unless design proves otherwise.
- **doc `0003:1864-1918`** — the AG-20 charter and its two Gherkin leaves.

---

## Success criteria — restated as verifiable checks

- [ ] `cd backend/agent && go test -race -count=1 ./...` green; wall-clock duration recorded as evidence
- [ ] **One registration surface** — `Hooks` is one exported type; `Harness` gains **no** exported method; `scope_fence_test.go` and `harness_test.go:1031` both pass **unedited**; a non-zero `h.Turn.Hooks` is refused typed before any event is emitted
- [ ] **Pre-request chain** — AG-08's entire suite passes **unedited**; a zero-value `TurnOptions` still produces a byte-identical request (`R-PRH-005` as amended); the singular field runs first and its output feeds element 0; a failing element aborts before I/O with `*ai.PreStreamFailure` naming **which** element
- [ ] **Session-start** — fires exactly once across two serial `Run` calls on one `Harness` value, before the first turn; fires again for a distinct child `Harness`; a shut-down value fires none
- [ ] **Post-turn** — fires once per logical turn on the success arm, the failed arm and the wind-down arm; does **not** fire on the no-turn iteration-boundary exit or for a compaction bracket; the payload's cost is the **sum over that turn's attempts** with a multi-attempt fixture proving it; the outcome is read from `turn_end`, and **no `TurnOutcome` member is added**
- [ ] **Pre-compact** — fires before `provider.Stream` on **both** doors with a real `CompactionSpan`; an adjusted plan is re-resolved by `resolveCut` and the resulting prefix is pairing-closed; `resolveCut` idempotence proven; a hook failure lands on the existing `compaction_failed` arm and the run continues (`R-CMP-010`)
- [ ] **Ordering** — with N hooks at one point, dispatch is registration order, deterministically, at every one of the four points
- [ ] **AG-20.2 / asynchrony** — with an observer held by `agenttest.Gate`, the recorded stream is **byte-identical** to the same script with no hooks installed, `CheckStream` accepts it, and the run returns before the gate is released
- [ ] **AG-20.2 / eventually** — the terminal-boundary snapshot reports the outstanding observer typed and attributed (point, index, run); **no timeout, no deadline, no sleep anywhere in production or test**; `TestPermission_WakeParked_…_NoDeadline` stays **file-unchanged**
- [ ] **Panic** — a panicking observer is reported through the same typed surface, the lane continues, and the process survives
- [ ] **Bites RED-recorded before GREEN**: no-stall ⇒ empty report; synchronous observer ⇒ delivery assertion fails; reversed dispatch ⇒ ordering fails; skipped re-resolution ⇒ split pair; per-`Run` session-start ⇒ two fires
- [ ] **Scope fence** — no new `EventKind` (guard at 25), no new `TurnOutcome`, no new `CostLabel`; `Turn` and `Run` signatures unchanged; the twenty-one files listed under "Not changed" byte-unchanged; `agenttest` byte-unchanged; `go.mod`/`go.sum` byte-unchanged; all of `src/ai/` byte-unchanged
- [ ] **Guards green unchanged** — ambient-authority, import-boundary, doc-contract, `PolicySlot` source guard, `S-DEL-024` scope fence
- [ ] **Deltas shipped** — eleven, each cited line opened by `sdd-verify`; `agent-loop-skeleton`'s `R-LSK-004` release **not** requested, and that conclusion recorded rather than inferred
- [ ] `make lint` (after `golangci-lint cache clean`), `make build`, `make vuln-check` all clean (`vuln-check` is **not** in `make all`; **do not run `make all`** — its fmt step rewrites committed files)

---

## Proposal question round

Execution mode is `auto`, so these were not asked interactively. Each changes the shape of the product, not the harness. **Answering any of them before `sdd-design` moves the recommendation above.**

1. **Is "one session = one `Harness` value" the product's meaning of a session?** Assumed **yes** (D2), because `History` is caller-owned and survives across `Run` calls, so a reused harness is one conversation. If Layer 3 intends a session to outlive a `Harness` value — surviving a process restart, or spanning several harnesses — that is a durable-identity feature and it belongs to AG-21/AG-23, not here. Say so now, because the latch is what makes the charter sentence true.
2. **Should a delegated child run fire its own session-start?** Assumed **yes** (D2), on the grounds that a subagent is its own session with its own transcript. If the product wants one session-start per *user-visible* conversation, the child must be suppressed — and that requires a notion of "child harness" that Layer 2 deliberately does not have (AG-19 kept every subagent concept in `package agent_test`).
3. **Does a stalled observer need to be visible to a *human watching the stream*, or only to the *program that registered the hook*?** Assumed **the program only** (D3). If an operator must see it live, the answer is still not a 26th kind — it is a Layer 3 consumer that receives the typed report and renders it. Confirm, because the opposite answer reopens the seven-site fence.
4. **Is "reported at the run's terminal boundary" soon enough?** Assumed **yes** (D4). A long run with a hook that stalls in turn 1 learns about it only at `run_end`. The alternative — reporting at each hook point's next firing — is also structural and also testable, and it is strictly more informative; it was not chosen because it fires a report repeatedly for one stall. If earlier notice matters more than a single report per stall, say so.
5. **Should post-turn fire during wind-down?** Assumed **yes** (D7 row 3). It means an interrupted run delivers a post-turn observation for a turn the user cancelled. If Layer 3 would rather see nothing for a cancelled turn, that is a one-row change now and a breaking change later.
6. **Is a forward-adjusting pre-compact hook acceptable?** Assumed **yes** (D6): the hook is *mutating* by charter, and `R-CMP-004`'s forward-expansion ban governs resolution, not the request. If the product wants the strategy's boundary to be inviolable, pre-compact becomes an observing hook and the charter's own "which hooks may mutate" sentence needs amending — which is a bigger change than it looks.
