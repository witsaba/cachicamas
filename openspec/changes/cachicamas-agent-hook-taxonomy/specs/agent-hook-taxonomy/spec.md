# Spec — The complete hook taxonomy (`agent-hook-taxonomy`)

> **Change**: `cachicamas-agent-hook-taxonomy` · **AG-20** (Layer 2, Wave 5, milestone 20 of 24 — the **last** of Wave 5), charter `0003:1864-1918`
> **NEW capability**, minted by this change. Promoted to `openspec/specs/agent-hook-taxonomy/spec.md` at archive.
> **Nodes**: AG-20.1 `[leaf]` (the three remaining hook points) · AG-20.2 `[leaf]` (observer asynchrony)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && go test -race -count=1 ./...`. **`-count=1` is mandatory**: the real uncached agent-module suite is ~170s, so a sub-second pass is a cache artifact and not evidence.
> **IDs**: `R-HKS-0NN` / `S-HKS-0NN` / `NFR-HKS-0NN`, **append-only**. Allocated `R-HKS-001`…`R-HKS-012`, `S-HKS-001`…`S-HKS-026`, `S-HKS-050`…`S-HKS-054` (the five **bites**), and `NFR-HKS-001`…`NFR-HKS-006`. This header states the allocated **range** and never a total, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`). Prefix `HKS` verified free repo-wide before minting and **re-verified in this phase**: zero occurrences under `openspec/specs/`.
> **Sources**: charter `0003:1864-1918`; this change's `proposal.md` (D1–D7, binding) and `design.md` (**AD-1…AD-10, binding and not re-opened here**). Where this spec differs from the proposal it follows the design, and the design declares **two** deliberate overturns which this spec carries as written: **AD-4** overturns D3's "the reporter is invoked on the observer lane" for the snapshot half (the lane's drain goroutine is blocked *inside* the stalled observer, so a lane-queued report can never run), and **AD-9/T2** overturns D4's gate-release ordering and redefines bite (b) (the literal synchronous-dispatch sabotage cannot fail as an assertion without a clock).
> **Ownership boundary**: this capability owns the `Hooks` registration surface, the two function-type families, the three payload types, the typed stall report, the observer lane, the four firing points and the asynchrony discipline. It does **not** own the pre-request seam's shipped field ([`../agent-pre-request-hook/spec.md`](../agent-pre-request-hook/spec.md)), the compaction bracket ([`../agent-compaction/spec.md`](../agent-compaction/spec.md)), the envelope ([`../agent-event-envelope/spec.md`](../agent-event-envelope/spec.md)), the delivery decoupling mechanism ([`../agent-event-delivery/spec.md`](../agent-event-delivery/spec.md)) or the run driver ([`../agent-run-driver/spec.md`](../agent-run-driver/spec.md)).
> **Every `file:line` cited below was opened in this worktree during this phase, at `main@2a138b59`.** No citation is carried forward from `explore.md`, `proposal.md` or `design.md` unresolved.

## Purpose

Layer 2 has **one** hook point and **four** promised ones. AG-08 shipped `TurnOptions.PreRequestHook` (`loop.go:90`) and made a commitment on this milestone's behalf, verbatim: *"the seam is a single callable on `TurnOptions` … **AG-20 widens to chain composition**"* (`agent-pre-request-hook/spec.md:19`). Pre-compact, post-turn and session-start exist today only as prose in `ai-contract-vocabulary/spec.md:331` (`V-OUT-13`, the row carrying **G11** / **R-17**).

The gap is not "three missing callbacks". It is that **the discipline that makes hooks safe has never been made mechanical**. `agent-event-envelope/spec.md:268` reserves envelope invariant 3 — *non-blocking observers* — as closed by "AG-01.1 + AG-20.2", and `R-AGE-008` (`agent-event-delivery/spec.md:123`) states the standard AG-20.2 must meet: *"A statement of the obligation, a convention, a review rule, or documentation alone MUST NOT satisfy this requirement."* Nothing in Layer 2 enforces that an observer cannot stall the streaming path, because Layer 2 has no observers. AG-20 lands three, and the property becomes falsifiable the same day.

## Coverage — the charter's four Gherkin scenarios, each mapped

Every charter leaf traces to at least one requirement **and** at least one scenario. No leaf is reduced and no scenario is orphaned.

| # | Leaf | Charter scenario | Lines | Owning requirement(s) | Scenario(s) |
|---|---|---|---|---|---|
| 1 | AG-20.1 | "each hook fires at its documented moment" | `0003:1888-1891` | `R-HKS-003`, `R-HKS-004`, `R-HKS-005` | `S-HKS-006`, `S-HKS-010`, `S-HKS-012`, `S-HKS-013`, bites `S-HKS-054` |
| 2 | AG-20.1 | "pre-compact adjustments stay invariant-safe" | `0003:1893-1896` | `R-HKS-003` | `S-HKS-007`, `S-HKS-008`, bite `S-HKS-053` |
| 3 | AG-20.1 | "hook ordering is deterministic (pin)" | `0003:1898-1901` | `R-HKS-006` | `S-HKS-016`, bite `S-HKS-052` |
| 4 | AG-20.2 | "a stalled observer cannot stall the stream" (both `Then` clauses) | `0003:1911-1915` | `R-HKS-007` (delivery unimpeded), `R-HKS-008` (eventually reported typed) | `S-HKS-017`, `S-HKS-018`, `S-HKS-019`, bites `S-HKS-050`, `S-HKS-051` |

Cross-cut requirements carrying no charter leaf of their own: `R-HKS-001` (registration surface and the two families), `R-HKS-002` (the pre-request chain, discharging AG-08's `:19` promise), `R-HKS-009` (mutating-hook failure, typed and attributed — the charter's third acceptance clause, `0003:1872`), `R-HKS-010` (scope fence), `R-HKS-011` (closed-sequence safety) and `R-HKS-012` (inertness).

## Two design overturns, recorded as such

Recorded here rather than left to inference, on the AG-19 precedent. A reader holding only `proposal.md` would otherwise read a different mechanism than AG-20 ships.

| Proposal clause | Normative reading (design) | Why the proposal's reading is rejected |
|---|---|---|
| D3: the stall reporter is *"invoked **on the observer lane**"* | **Two invocation sites, discriminated by reason** (AD-4): snapshot reports (`StallOutstanding`, `StallQueued`) run **on `Run`'s goroutine** at the terminal boundary; panic reports (`StallPanicked`) run **on the lane goroutine** at recovery. | In the motivating case the lane's single drain goroutine is blocked **inside** the stalled observer, so a lane-queued snapshot report can never run. A report that is structurally undeliverable in exactly the case it exists for is not a report. |
| D4 bullet 4 read as *"the run returns before the gate is released"* with a lane-side reporter | **AD-9/T2**: the snapshot fires the reporter on `Run`'s goroutine, `Run` returns **with the gate still held**, the test asserts, and **only then** releases the gate, with `t.Cleanup(gate.Release)` as the unconditional backstop. Bite (b) is redefined as a **goroutine-placement** assertion (`runtime.Stack`), not the literal synchronous-dispatch sabotage. | Under the literal sabotage the observer's return is ordered happens-before **every** post-fire-point observable, so a test witnessing asynchrony would have to wait for an event that never occurs. The only bounded conversion of "never" is a clock, and six shipped NFRs ban it — a gate-holding bite would deadlock by construction, and a deadlocking bite cannot have its RED recorded. |

## Requirements

### R-HKS-001 — ONE registration surface, two function-type families, and the wrong thing is unconstructible

The system MUST expose exactly one exported registration type, `Hooks`, carrying the four hook families and one nil-defaultable stall reporter. `Harness.Hooks` MUST be the one registration surface; `TurnOptions.Hooks` MUST be a **transport** the harness fills on its per-turn copy beside `Continuation`, never a second registration surface. **`Harness` MUST gain no exported method**: `scope_fence_test.go:102-105` (`NumMethod() != 5`) and `harness_test.go:1031` (the exact named set) MUST both pass **unedited**.

The two families MUST be distinguished **by type**, not by documentation:

| Family | Members | Signature shape | May signal back? |
|---|---|---|---|
| **Mutating** | `PreRequest`, `PreCompact` | takes a context and a payload, returns `(payload, error)` | yes — a new payload, or a typed failure |
| **Observing** | `PostTurn`, `SessionStart` | takes a context and a report, returns **nothing at all** | **no — unconstructible** |

An observing hook MUST have **no result parameters**, so a function returning an error or a mutated payload fails to compile when placed in an observing slice. Every payload (`CompactionPlan`, `PostTurnReport`, `SessionStartReport`, `ObserverStall`) MUST be a value type with unexported fields and read-only accessors, so an observer cannot reach shared state through its argument. `CompactionPlan` alone MUST carry a cut-derivation method; the span MUST be derived and never settable, so a hook can re-designate a cut but cannot forge a span.

**The transport MUST NOT be silently overwritten.** A non-zero `Harness.Turn.Hooks` MUST be refused typed at `Run` entry, in the same pre-identity block as the shutdown check, **before any event is emitted** — the `validateContinuation` / `S-LSK-014` posture. Because `Hooks` holds function fields it is **not comparable**, so the emptiness test MUST be an explicit predicate over the four slices and the reporter, never an equality against a zero value.

Symmetrically, `CompactionRequest.Options.Hooks` MUST join the existing misplaced-options rejection in the compaction path (`compaction.go:269-272`); the pre-compact chain MUST reach the compaction operation as an explicit unexported parameter, never smuggled through the request's own options.

**A direct `Turn` caller who sets observing families on `TurnOptions.Hooks` gets INERTNESS, not refusal, and this MUST be stated rather than inferred.** `Turn` consumes only the `PreRequest` family; it owns no observer lane and no run identity, so `PostTurn`, `SessionStart` and `StallReporter` set on a directly-constructed `TurnOptions` fire nothing, report nothing, and change nothing observable. This is deliberate: the refusal above protects the **harness's** transport from a silent overwrite, and a direct `Turn` call has no transport to protect.

#### Scenarios

- **S-HKS-001** — **One surface, and the fence proves it — corrected by `sdd-verify` (MAJOR-4) to match `R-HKS-010` consequence 4's own recorded release.** Given the package after this change, when its exported surface is enumerated from `package agent_test`, then `Hooks` is the only hook registration type; `Harness` declares exactly the five exported methods it declared at the merge base, named individually rather than counted; `scope_fence_test.go`'s own `NumMethod()`-and-named-set assertions and `harness_test.go`'s named-method assertion are **byte-unchanged** and green (the assertions, not the whole file: `scope_fence_test.go` itself carries the narrow, recorded anti-vacuity-floor release `R-HKS-010` consequence 4 and `agent-loop-skeleton`'s `R-LSK-008` consequence 6 own in full — 13 changed lines, confined to converting one `t.Fatal` to `t.Skip`, touching neither this assertion nor any other); and no exported hook-registration method exists on `Harness` under any name.
- **S-HKS-002** — **The transport refuses rather than overwrites.** Given a `Harness` whose `Turn.Hooks` carries a non-zero value in **any one** of the five members — each of the four families and the reporter, asserted member by member — when `Run` is called, then it returns a typed misplaced-options refusal identifiable by its typed shape, the consumer sink carries **no event at all** (not even a run-open), and no run identity was minted; and given a `Harness` whose `Turn.Hooks` is entirely zero, when `Run` is called, then the run proceeds normally.
- **S-HKS-003** — **The observing families cannot signal, and a direct `Turn` observer is inert.** Given `package agent_test`, when a function with any result parameter is assigned to an observing slice, then the package **fails to compile** (asserted through the package's own compile-time-refusal fixture pattern, not by a runtime check); and given a direct `Turn` call whose `TurnOptions.Hooks` carries post-turn observers, session-start observers and a stall reporter, when that `Turn` runs, then no observer is invoked, the reporter receives nothing, no goroutine beyond the baseline is created, and the captured request and emitted event slice are byte-identical to the same call with a zero-value `Hooks`.

---

### R-HKS-002 — The pre-request chain: the shipped singular field runs FIRST and its output feeds element 0

The pre-request seam MUST become a composed chain. When both are present, `TurnOptions.PreRequestHook` MUST be invoked **first** and its returned request MUST be the input to `Hooks.PreRequest[0]`; element *i*'s returned request MUST be the input to element *i+1*; and the **final** element's returned request is what reaches `provider.Stream`. Each element MUST be invoked exactly once per `Turn` call that reaches the request build, with the loop's own `ctx`.

**AG-08's shipped field is kept, unamended in behavior, and superseded in prose only.** It MUST NOT be marked with Go's `Deprecated:` convention, which would make every existing internal reference a lint finding for no behavioral gain. Its removal belongs to AG-23 and is **not** decided here.

**The whole of AG-08's test suite MUST pass byte-unchanged**, and that is the compatibility proof rather than a claim about it. `R-PRH-005`'s nil-default byte-identity survives as the "**no** singular hook **and** empty chain" case; this change's [`../agent-pre-request-hook/spec.md`](../agent-pre-request-hook/spec.md) delta carries the amendment, and the amendment MUST precede the code.

#### Scenarios

- **S-HKS-004** — **Composition order is observable, and the singular field is first.** Given a `Harness` with `Turn.PreRequestHook` appending marker `A` to the system instruction and `Hooks.PreRequest` holding three elements appending `B`, `C`, `D` in registration order, when a one-turn run executes, then the request captured at the provider carries all four markers in the order `A, B, C, D`; each element observed as its input exactly the request its predecessor returned, asserted by each element recording the marker set it received; and the provider received exactly one request for that attempt.
- **S-HKS-005** — **AG-08 compatibility is proven by an unedited suite.** Given the merged change, when AG-08's `agent-pre-request-hook` test files are diffed against the merge base, then they are **byte-unchanged**, and when the whole module suite runs with `-count=1`, then every AG-08 scenario is green; and given a zero-value `TurnOptions` with a zero-value `Hooks`, when `Turn` runs against the same script `S-PRH-002` uses, then the captured request is byte-identical to what the hookless skeleton produced.

---

### R-HKS-003 — Pre-compact fires inside the compaction operation, before the provider call, on BOTH doors — and its adjustment is re-resolved

The pre-compact chain MUST be invoked inside the compaction operation, **after** the naive cut has been resolved and the span derived, and **strictly before** the compaction provider call. Because both compaction doors — the strategy-verdict door and the on-demand door of `R-CMP-001` — funnel through that one operation, one splice MUST reach both, and no second splice MAY be added.

The hook MUST receive a plan carrying the **resolved** cut and the derived span, not a bare index. The plan returned by the chain MUST then be fed **unconditionally** back through the existing cut resolution and span derivation of `R-CMP-004` — the retraction-only, structurally terminating surgery — before use. No new validation code is written: the charter's *"the adjusted span is revalidated by the invariant-safe surgery before use"* is literal.

**The forward direction is permitted at the REQUEST and forbidden at RESOLUTION, and this distinction is normative.** A mutating pre-compact hook MAY re-designate the cut forward; resolution then retracts from the new request exactly as it always has. `R-CMP-004`'s *"Resolution MUST NOT expand the cut **forward** under any condition"* (`agent-compaction/spec.md:115`) governs **resolution**, never the **request**, and this change's [`../agent-compaction/spec.md`](../agent-compaction/spec.md) delta states that scoping explicitly so a reviewer reading a forward-adjusting hook against the shipped MUST finds the sentence scoped rather than violated.

**Idempotence MUST be proven, not assumed.** For an unchanged plan the second resolution MUST be a no-op, so that installing a hook which returns its input unchanged produces a stream and a committed history byte-identical to the hookless run. A re-resolution that reaches zero MUST fail typed through the compaction bracket's **existing** failure arm carrying the aborted turn outcome; the upstream empty-cut arm is not reachable from the post-splice position, so the arm's shape is reused at a **new** call site.

A chain element's failure MUST land on that same existing failure arm, **before** the provider call, so `R-CMP-003`'s iff and `R-CMP-010`/`R-CMP-013`'s "a compaction failure is visible on the stream and nowhere else, and never interrupts the run" hold unamended and the run continues.

**No second pre-request seam exists inside a compaction.** The compaction provider call is not a `Turn` and never passes the pre-request seam; `PreRequest` MUST fire zero times for a compaction bracket, and that is stated so nobody infers otherwise.

#### Scenarios

- **S-HKS-006** — **Charter AG-20.1 scenario 1, the pre-compact third — both doors.** Given a run whose compaction is reached through the strategy verdict, and a second run whose compaction is reached through the on-demand entry point, and a `PreCompact` chain of one element recording what it received, when each run executes, then in **both** runs the element was invoked exactly once, the plan it received carried the resolved cut and a derived span whose start and end match the span the compaction bracket used, the invocation is ordered strictly before the compaction provider received any request (asserted by the provider's own recorded call count being zero at hook entry), and `PreRequest` fired **zero** times for that bracket.
- **S-HKS-007** — **Charter AG-20.1 scenario 2, the identity half — re-resolution is a no-op on an unchanged plan.** Given two runs of an identical compacting script — one with no hooks and one with a `PreCompact` element returning its input plan unchanged — when both are recorded, then the two event streams are byte-identical modulo the freshly minted run and turn identifiers, the two committed history read-backs are byte-identical, and both are `CheckStream`-valid; and given the module's internal fixed-point table over the same class of naive cuts (zero, on a mark, mid-mark, straddling an open pair, beyond length), when resolution is applied twice, then the second application returns its input for every row.
- **S-HKS-008** — **Charter AG-20.1 scenario 2, the adjustment half — forward re-designation stays invariant-safe.** Given a transcript whose resolved cut is *r* and a `PreCompact` element re-designating the cut **forward** to an index *f > r* that lands strictly inside an open call/result pair, when compaction proceeds, then the cut actually committed is ≤ *f*, the committed prefix is pairing-closed with no call separated from its result, a fresh history seeded from the post-compaction read-back constructs successfully, and no assertion in the scenario reads elapsed time.
- **S-HKS-009** — **A chain element's failure lands on the existing arm and the run continues.** Given a `PreCompact` chain whose second element returns a non-nil error, when the compaction bracket runs, then the compaction provider received **no** request, the bracket closes with the aborted turn outcome carrying a typed failure, exactly one compaction-failed event is emitted and no compaction-finished event is, the committed transcript is byte-identical to its pre-attempt read-back, the run **continues** to its own terminal, and `CheckStream` accepts the whole stream unmodified.
- **S-HKS-053** — **(bite)** RED-first, **mandated**. Given a scratch tree in which the post-hook re-resolution is skipped and the hook's returned cut is used directly, when `S-HKS-008` runs, then it **FAILS** with the committed prefix splitting a call/result pair — proving the re-resolution is load-bearing rather than incidentally satisfied by a fixture whose adjusted cut already landed cleanly. RED-recorded BEFORE `S-HKS-008` is GREEN, with `-count=1`, then reverted.

---

### R-HKS-004 — Post-turn fires once per LOGICAL turn, and EVERY exit from the logical-turn loop is enumerated with a verdict

Post-turn MUST be enqueued exactly once per logical turn that actually ran a turn, carrying a report of the run identity, the turn identity, the turn's outcome as read from the forwarded turn-close payload, the turn's cost, and the attempt count.

**The cost MUST be the sum over that logical turn's attempts**, not the last attempt's. A logical turn may make several attempts, each emitting its own per-turn cost event inside its own bracket; aborted attempts contribute none. The accumulator MUST be a local of the run's own frame and **MUST NOT** be a `Harness` field — `R-CST-004` forbids the latter, and this change's reading of that requirement is that it holds **unamended** because the accumulator is a pure read added beside the existing per-attempt forwarder, adds no second writer on the event path, and folds no foreign figure.

**No `TurnOutcome` member is added.** The outcome is read from the existing forwarded payload.

The complete exit enumeration is **normative**. This repository's recorded failure shape is a bug hiding between two individually satisfied requirements, so every exit carries an explicit verdict rather than an inference:

| # | Exit from the logical-turn loop | Fires? | Outcome carried |
|---|---|---|---|
| 1 | success, before the turn is closed in history | **yes** | the forwarded finished outcome |
| 2 | the history turn-close fails and the run fails | **yes** — already fired at row 1's site; the fire **precedes** the close | finished |
| 3 | the turn failed and the run fails | **yes** | the forwarded aborted outcome |
| 4 | a mid-turn signal winds the run down | **yes** | aborted |
| 5 | a signal during the retry backoff winds the run down | **yes** | aborted |
| 6 | a bare cancellation during backoff breaks to the failing exit | **yes** — row 3's site | aborted |
| 7 | an iteration-boundary signal with **no** turn run this iteration | **no** | — |
| 8 | a post-compaction signal with **no** turn run this iteration | **no** | — |
| 9 | a steering-drain append error | **no** | — |
| 10 | a prompt append error before the loop | **no** | — |
| 11 | a terminal-decision steering append error | **no extra** — row 1 already fired | — |
| 12 | a run-open construction error | **no** | — |
| 13 | the shutdown refusal at entry | **no** | — |
| 14 | a compaction bracket | **no** — compaction is not a logical turn; pre-compact is its hook | — |

The fourteen exits MUST collapse to **four** enqueue call sites, each preceding its exit call, so that **no exit path can fire twice and no firing row can be skipped by an early return above it**. Every fire site MUST be downstream of the per-attempt forwarder's completion, so the report's reads inherit the forwarder's existing happens-before.

#### Scenarios

- **S-HKS-010** — **Charter AG-20.1 scenario 1, the post-turn third — every yes-row fires exactly once.** Given six runs, one driving each of rows 1–6 of the table above, and a `PostTurn` observer recording every report it receives, when each run completes and its lane has drained, then exactly one report was received per logical turn that ran, its run and turn identities equal the identities observed on that run's own recorded stream, its outcome equals the outcome carried by that turn's close event, and **no run produced two reports for one logical turn** — asserted by report count against the turn-close count on the stream, not by eyeballing a log.
- **S-HKS-011** — **Every no-row fires nothing — two rows proven behaviorally, six by the structural call-site count, corrected by `sdd-verify` (MAJOR-2) to state its actual method rather than an unshipped eight-run claim.** Given **two** behavioral runs — one driving row 13 (the shutdown refusal at entry, trivial and direct) and one driving row 7 (an iteration-boundary signal with no turn run this iteration, via a test-local tool whose `Run` method calls `h.Interrupt()` as a side effect so the signal is set after the tool-calling turn's own stream has already completed but before the next iteration's own boundary check) — when each run returns, then the observer's recorded report count is **zero** *and* the stall reporter's report set is **empty**, jointly, exactly as the snapshot-or-count lemma requires: an enqueued invocation is, at the terminal snapshot, either complete (its side effect is visible to any post-`Run` read) or outstanding/queued (it is in the report), and there is no third state. The remaining six rows (8: post-compaction signal; 9: steering-drain append error; 10: prompt append error pre-loop; 11: terminal-decision steer append error; 12: `NewRunStart` construction error; 14: compaction bracket) are **not** independently driven — each requires forcing an unexported constructor or an internal validation path to fail in a way `package agent_test` cannot reach externally — and are instead proven **structurally**: given `harness.go`'s own source, when it is scanned, then it contains **exactly four** `lane.enqueuePostTurn(` call sites and **exactly one** `lane.enqueueSessionStart(` call site, meaning there is no fifth, hidden fire site any of the six could reach — a real, narrower proof than six additional behavioral runs would be, bounding *where* firing can happen even though it does not independently show each of the six exit paths avoids those four known sites. No assertion sleeps, polls, or reads a clock.
- **S-HKS-012** — **The cost is the sum over the turn's attempts, and the multi-attempt fixture is what makes it non-vacuous.** Given a logical turn scripted to make **three** attempts — two retried after a retryable failure, the third succeeding — each attempt carrying a **distinct, non-zero** usage figure, when the post-turn report arrives, then its cost equals the sum of the per-turn cost events observed inside that logical turn's brackets on the recorded stream, that sum is **strictly greater** than the last attempt's own figure on at least one reported figure, and the report's attempt count equals three.

---

### R-HKS-005 — Session-start fires ONCE PER `Harness` VALUE, and the latch is consumed only by a run that fires

Session-start MUST fire **once per `Harness` value**, not once per run. The wording "once per `Harness` value" is normative: the type is value-form with no constructor and mints a fresh run identity on every `Run` call, so there is no session identity — there is a one-way latch, of exactly the class the shipped terminal shutdown flag already is, and it is per-value bookkeeping only, holds no transcript, is never resumed, and does not pre-empt AG-21's cross-run state.

Two consequences MUST be stated as requirement text rather than left to inference:

1. **Serial reuse does not re-fire.** A caller-owned transcript survives across `Run` calls and a serially reused harness is a shipped, tested shape; that reuse is exactly *one session, many runs*. Firing per `Run` would make the hook indistinguishable from the run-open event and would make the charter sentence false.
2. **A delegated child fires its own.** A child run is constructed on a distinct `Harness` value with its own latch, so it fires once for itself. A child that inherits no `Hooks` value fires nothing, so AG-19's shipped fixtures are undisturbed.

The latch MUST be tested and set at `Run` entry **after** the shutdown check, so a shut-down value fires nothing and leaves the latch untouched. The enqueue MUST be placed **before** the run-open event's construction, not merely before its send: a run that fails to construct its run-open event would otherwise **consume the latch without ever firing**, silencing session-start for the value's entire remaining lifetime. Fired-and-consumed on a degenerate run is consistent; consumed-without-firing is the bug.

Session-start MUST be the lane's **first** invocation for the run that fires it, ordered before any post-turn invocation of that run.

#### Scenarios

- **S-HKS-013** — **Charter AG-20.1 scenario 1, the session-start third — once across two serial runs.** Given one `Harness` value with a `SessionStart` observer and a multi-turn script, when `Run` is called, allowed to complete, and called a **second** time on the same value, then the observer received exactly **one** report across both runs; that report's run identity equals the **first** run's identity as observed on its own recorded stream; the invocation is ordered before that run's first post-turn report on the same lane; and both runs' streams are `CheckStream`-valid.
- **S-HKS-014** — **A delegated child fires its own, and the parent's does not re-fire.** Given a parent `Harness` with a `SessionStart` observer hosting a child run built on a **distinct** `Harness` value that carries its own `SessionStart` observer, when the parent run completes, then the parent's observer received exactly one report carrying the parent's run identity, the child's observer received exactly one report carrying the **child's** run identity, and neither observer received the other's.
- **S-HKS-015** — **Shutdown fires none; a degenerate run still fires.** Given a `Harness` value whose terminal shutdown has latched, when `Run` is called, then it returns the shutdown refusal, the observer received **zero** reports, and the session-start latch is still unconsumed — proven by clearing the shutdown state, calling `Run` again, and observing exactly one report; and given a run whose run-open event fails to construct, when `Run` returns its construction error, then the observer received exactly **one** report — the latch was fired, not silently consumed.
- **S-HKS-054** — **(bite)** RED-first, **mandated**. Given a scratch tree in which session-start is enqueued per `Run` rather than behind the latch, when `S-HKS-013` runs, then it **FAILS** — deterministically either way under the snapshot-or-count lemma: the observed report count is **two**, **or**, if the second invocation is still outstanding at the second run's terminal boundary, the second run's stall report set is non-empty naming the session-start point. Neither outcome requires a wait. RED-recorded BEFORE `S-HKS-013` is GREEN, with `-count=1`, then reverted.

---

### R-HKS-006 — Dispatch is REGISTRATION ORDER at every one of the four points, deterministically

With N hooks registered at one point, the system MUST invoke them in registration order — index 0 first, index N-1 last — at **every** one of the four points, deterministically and with no fan-out. For the two mutating points, registration order is composition order (`R-HKS-002`, `R-HKS-003`). For the two observing points, registration order MUST equal enqueue order MUST equal dispatch order, which the lane guarantees by being **serial**: one drain goroutine popping first-in-first-out.

**Deregistration, priority, filtering and conditional registration are NOT in this milestone.** Registration order is the whole ordering contract (`0003:1898-1901`), and a later milestone adding any of them MUST re-check this requirement first.

#### Scenarios

- **S-HKS-016** — **Charter AG-20.1 scenario 3 — the pin, at all four points.** Given four hooks registered at **each** of the four points, each appending its own index to a per-point recorder, when a run that reaches all four points executes and its lane has drained, then each point's recorded sequence is exactly `[0, 1, 2, 3]`; the recorded sequence is identical across repeated executions of the same script; the session-start entry precedes every post-turn entry on the observing recorder; and the scenario passes under `-race`.
- **S-HKS-052** — **(bite)** RED-first, **mandated**. Given a scratch tree in which dispatch at one point iterates the registered hooks in reverse, when `S-HKS-016` runs, then it **FAILS** reporting the recorded sequence `[3, 2, 1, 0]` for that point — proving the ordering assertion is non-vacuous rather than satisfied by a single-hook fixture. RED-recorded BEFORE `S-HKS-016` is GREEN, with `-count=1`, then reverted.

---

### R-HKS-007 — Observer asynchrony is STRUCTURAL: enqueue never blocks, and dispatch is off the streaming path

An observing hook MUST NOT be invoked on the goroutine that drives the run or on any goroutine that stamps or delivers events. The system MUST create **one** observer lane per run, and only when at least one observing hook is registered. Enqueue MUST be an append under a lock and MUST NOT block on the observer's progress — **that**, and not scheduling luck, is the non-blocking property. Dispatch MUST happen on the lane's own drain goroutine.

The observers' invocation context MUST be freshly rooted and **value-stripped** — neither the run context nor a cancellation-stripped derivation of it. The reason is structural and MUST be recorded rather than rediscovered: a cancellation-stripped derivation **preserves context values**, and in a hosted child run the child's run context inherits the hosting tool call's publishing seam, retrievable by a plain value lookup (`delegation_seam.go:101-104`). A value-preserving observer context would therefore hand an **observing** hook the one sanctioned door onto the **parent's** streaming lane, asynchronously, after `Run` returned — an observer mutating a stream, the exact thing AG-20 exists to forbid. Value-stripping makes `R-HKS-001`'s "unconstructible" literally true and costs nothing, because an observer may run after `Run` returned and the run's cancellation and values would misreport anyway.

**Mutating hooks keep the live context and the caller's goroutine, exactly as AG-08 shipped.** Changing that is out of scope and stated as such.

`R-AGE-008`'s decoupling mechanism MUST hold unamended: no path from a stalled observer reaches the canonical producer. `R-AGE-009`'s multi-consumer fan-out machinery is **not** shipped by this change — AG-20 makes a *hook* non-blocking; it does not build consumer fan-out.

#### Scenarios

- **S-HKS-017** — **Charter AG-20.2, the "delivery is unimpeded" clause.** Given a `PostTurn` observer held open by the module's test gate primitive and a sink **buffered to the script's full event count** so the run needs no live consumer, when the run executes, then entering the gate proves the invocation started and has not returned; the recorded event stream is **byte-identical** to the same script with no hooks installed, modulo the freshly minted run and turn identifiers; `CheckStream` accepts it unmodified; `Run` returns **while the gate is still held**; and the gate is released only after those assertions, with an unconditional cleanup release registered at the gate's construction.
- **S-HKS-018** — **Asynchrony is a goroutine-placement property, so it is asserted as one.** Given a **non-blocking** `PostTurn` observer that captures its own stack at invocation, when `Run` has returned and the lane has drained, then the recorded stack contains **neither** the harness run frame **nor** the per-attempt forwarder frame — only the lane's own drain root; and no assertion in the scenario reads elapsed time, sleeps, or polls.
- **S-HKS-051** — **(bite)** RED-first, **mandated**, and its shape is deliberate. Given a scratch tree in which observing hooks are dispatched **synchronously** at the fire site, when `S-HKS-018` runs, then it **FAILS as an assertion** — not as a hang — reporting a captured stack containing the harness run frame, with the offending stack in the failure message. The literal proposal-era sabotage (a gate-holding synchronous-dispatch bite) is **rejected and the rejection is recorded**: under it the observer's return is ordered happens-before every post-fire-point observable, so any faithful wait is unbounded and the only bounded conversion is a clock, which `NFR-HKS-002` bans. RED-recorded BEFORE `S-HKS-018` is GREEN, with `-count=1`, then reverted.

---

### R-HKS-008 — "Eventually" is the RUN'S TERMINAL BOUNDARY, and the typed report has three reasons delivered at TWO sites

"Eventually reported typed" MUST be **structural, not temporal**. The system MUST add **no wall clock, no timeout, no deadline, no sleep, and no join on a stalled observer**, following `R-RUN-010` (`agent-run-driver/spec.md:229`, `:238`) explicitly. There is no wall-clock timeout anywhere in the module today, and a "slow hook" threshold is a Layer 3 policy number that `R-AGS-015` forbids Layer 2 deciding.

> **The definition.** *Eventually* = **at the run's terminal boundary** — after the run-close event has been sent on every returning arm, and before the lane closes for enqueue, the run's cancellation state is cleared and the consumer sink is closed — the observer lane's outstanding set is snapshotted: every observing-hook invocation dispatched and not returned, plus every invocation still queued behind it. The run neither waits for nor joins any of them.

The report MUST be a Go-side typed value delivered to a nil-defaultable reporter registered on the same `Hooks` value. **No new `EventKind` is registered**, and the reason MUST be recorded as structural rather than budgetary: an event announcing the stall **is a path from the stalled observer back onto the producer's stream**, which is exactly what `S-AGE-010` forbids and exactly the invariant AG-20.2 exists to close. A budget argument can be overturned by a budget; this one cannot.

Each report MUST carry the hook point, the registration index, the run identity, and a **three-valued** reason. The two invocation sites are normative and MUST both be stated, because a spec naming only one contradicts the code:

| Reason | Meaning | Invoked where |
|---|---|---|
| outstanding | dispatched, not returned at the terminal boundary | the snapshot, **synchronously on `Run`'s goroutine** |
| queued | never dispatched, still queued behind a stall | the snapshot, on `Run`'s goroutine |
| panicked | the observer panicked and was recovered | at recovery, on the **lane** goroutine |

The third reason refines the proposal's two: queued victims are not stalled culprits, and collapsing them misattributes. Snapshot reports cannot run on the lane because in the motivating case the lane's drain goroutine is blocked **inside** the stalled observer. `R-AGE-008`/`S-AGE-010` is preserved for **event delivery**: no event follows the run-close, so a post-terminal reporter can impede no delivery and has no path onto the stream. All reporter invocations MUST be serialized by one dedicated exclusion.

**A nil reporter reports nothing**, and this MUST be stated in exactly the terms the other nil-default seams use, so "eventually reported" is never read as an unconditional emission promise.

**Panic posture.** The drain goroutine MUST recover a panicking observing hook, report it with the panicked reason, and **continue with the next queued invocation** — the scheduler's own tool panic-recovery posture, one layer out. Without recovery, "observers never block the streaming path" would be true in the worst way: an unrecovered panic on the lane kills the process. Reporter invocations at **both** sites MUST also be recover-wrapped, and a reporter's own panic MUST be **discarded**: the reporter is the last resort, it has no meta-reporter, and process survival wins. **Mutating hooks are NOT recovered** — they run inline on the caller's goroutine exactly as AG-08 shipped, and that is out of scope and stated as such.

Four consequences MUST be stated rather than discovered:

1. **A permanently stalled observer leaks a goroutine, by design, and it is the caller's leak.** AG-21 inherits this knowingly.
2. **An observer finishing just after the snapshot is reported once and completes anyway.** The report is an honest statement about the terminal boundary, not a verdict about the hook.
3. **A stalling *reporter* stalls both `Run`'s return and the consumer sink's close.** Because the snapshot runs before the sink is closed, a consumer ranging the sink has already received the run-close but never observes termination. Caller's code, caller's stall, same posture on both.
4. **The lane closes for enqueue at the snapshot** — structurally no fire site follows it — and its goroutine exits when the queue drains, so a released observer leaves no steady-state goroutine.

#### Scenarios

- **S-HKS-019** — **Charter AG-20.2, the "eventually reported typed" clause — with a queued victim beside the culprit.** Given a `PostTurn` observer held open by the test gate at index 0 and a second observer at index 1 that is never reached, and a run producing at least two logical turns, when `Run` returns, then the reporter received reports naming **both**: one carrying the post-turn point, index 0, the run identity observed on that run's own stream, and the **outstanding** reason; and at least one carrying the **queued** reason for the invocation stranded behind it; the two reasons are distinguishable by inspection rather than by string; every report was delivered **before** `Run` returned (asserted by reading the reporter's record after `Run` returns, with the gate still held); and no assertion reads elapsed time.
- **S-HKS-020** — **A panicking observer is reported on the lane, the lane continues, the process survives.** Given three `PostTurn` observers registered in order where index 1 panics, when a run with two logical turns executes and the lane drains, then the reporter received a report carrying the post-turn point, index 1 and the **panicked** reason; observers at indices 0 and 2 each recorded their full expected invocation count; the test process survives to assert it; the recorded stream is byte-identical to the same script with no hooks installed; and the report was delivered from the lane goroutine, not from `Run`'s.
- **S-HKS-021** — **A nil reporter reports nothing, and a stalling reporter stalls the two documented observables.** Given a stalled observer and a **nil** `StallReporter`, when the run executes, then `Run` returns normally, the recorded stream is byte-identical to the hookless run, and nothing panics; and given a reporter itself held by a gate, when the run reaches its terminal boundary, then `Run` has **not** returned and the consumer sink has **not** closed while the reporter's gate is held — both asserted by non-blocking reads, never by waiting — and both complete once the reporter's gate is released by the scenario's unconditional cleanup.
- **S-HKS-050** — **(bite)** RED-first, **mandated** — the anti-vacuity bite. Given the shipped tree and an observer that returns **immediately** rather than stalling, when the run completes and `Run` returns, then the reporter's report set MUST be **empty**; and given a scratch tree that reports every enqueued observer regardless of state, when this scenario runs, then it **FAILS** with a non-empty report set — proving the report distinguishes a stalled observer from a completed one, rather than an implementation that reports everything passing `S-HKS-019` for free. RED-recorded BEFORE `S-HKS-019` is GREEN, with `-count=1`, then reverted.

---

### R-HKS-009 — A mutating hook's failure is typed AND attributed BY SOURCE NAME, never by a bare ordinal

When a mutating hook returns a non-nil error the system MUST abort typed **before** the I/O that hook guards, and the failure MUST name **which** hook failed.

**Attribution is by source name, not by a chain-wide ordinal**, and the distinction is normative because it is what stays true under later insertion. The two source names are the shipped singular field and the indexed chain slot — `TurnOptions.PreRequestHook` for the kept field (which runs first), and `Hooks.PreRequest[i]` or `Hooks.PreCompact[i]` for a chain element at index *i*. A bare ordinal over the composed sequence would renumber the moment a caller inserts an element, and a renumbered ordinal is a lie a green suite never catches. This settles the proposal's "element zero" phrasing (D1) against its own success-criteria phrasing ("feeds element 0"): the singular field is **not** element zero of the chain; it is a distinct, separately named source that runs before element zero.

**Pre-request** MUST keep `R-PRH-003`'s abort shape — the turn aborts before the provider is called, the sink closes, and the typed pre-stream failure is returned — widened only so the attribution names the source. **Pre-compact** MUST fail through the compaction bracket's existing failure arm (`R-HKS-003`), so `R-CMP-010`/`R-CMP-013` hold unamended.

#### Scenarios

- **S-HKS-022** — **Pre-request: attribution names the source, and the abort precedes I/O.** Given a `Harness` whose `Turn.PreRequestHook` succeeds and whose `Hooks.PreRequest` holds three elements of which index **1** returns a non-nil error, when the run executes, then the provider recorded **zero** requests for that attempt; the returned error carries the typed pre-stream failure shape; its attribution names `Hooks.PreRequest[1]` and **not** `Hooks.PreRequest[2]`, **not** the singular field, and **not** a bare ordinal over the composed sequence; the element at index 2 was never invoked; and the sink drains unblocked.
- **S-HKS-023** — **The singular field is a distinct named source, and its OWN insertion or removal does not renumber the chain — corrected by `sdd-verify` (MAJOR-3) to state "elsewhere" as `R-HKS-009`'s own prose means it, not as inserting INTO the chain.** Given a run whose **singular** `PreRequestHook` fails, when the run executes, then the attribution names `TurnOptions.PreRequestHook` and not any indexed slot; and given `S-HKS-022`'s failing chain element at index 1, run once with **no** singular field registered and once **with** one registered (running before the chain), when each run executes, then in BOTH runs the attribution names `Hooks.PreRequest[1]` — the chain element's own attribution is unchanged by whether the singular field exists at all, because the singular field sits entirely OUTSIDE `Hooks.PreRequest`'s own indexing and is never counted as "element zero" of it (the exact ambiguity `R-HKS-009` settles against the proposal's own D1 phrasing). **This is deliberately NOT** a claim that inserting an ADDITIONAL element INTO the chain ahead of index 1 leaves that element's own attribution unchanged — it does not, and must not: `Hooks.PreRequest[i]` is that element's own slice index, so an element inserted ahead of it genuinely renumbers it to `Hooks.PreRequest[i+1]`, which is `R-HKS-009`'s own point about a composed ordinal being unstable, not a case this scenario claims stability for.

---

### R-HKS-010 — The scope fence: what AG-20 ships, and what it does not

AG-20 MUST register **no** new `EventKind` (the shipped guard stays at its committed kind count, `scope_fence_test.go:87`), add **no** new turn outcome member and **no** new cost label, and change **no** existing exported signature: `Turn` and `Run` MUST keep their signatures and `Harness` MUST gain no exported method.

`event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `cost_usage.go`, `turn_events.go`, `run_events.go`, `failure.go`, `sequence.go`, `compaction_events.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, `scheduler.go`, `delegation_seam.go`, `go.mod`, `go.sum` and **every file under `backend/agent/src/ai/`** MUST be **byte-unchanged**. `backend/agent/src/agenttest/` MUST be byte-unchanged: the test gate primitive is **used**, not widened.

**The stall types MUST live in the new hook file, not in the typed-failure file.** Editing the typed-failure file would pass the mechanical guard silently while violating `R-LSK-004`'s prose; the placement is declined explicitly rather than left to convenience.

Five consequences MUST be stated here rather than discovered by a later consumer:

1. **No concrete hook ships.** The charter's own out-of-scope line is verbatim: *"Any concrete hook implementation (Layer 3 wires them)"* (`0003:1874`). Layer 3's wiring is doc 0004 CO-24.1/CO-24.2, and `S-AGS-048` already names those nodes.
2. **`TurnOptions.PreRequestHook` is kept and superseded in prose only.** Its removal is AG-23's, and AG-20 does not take it.
3. **AG-21 inherits the stalled-observer goroutine leak and the release-before-baseline test rule knowingly** (`NFR-HKS-005`), rather than discovering them.
4. **`scope_fence_test.go` is deliberately ABSENT from the byte-unchanged list above — a narrow, recorded release, discovered during `sdd-apply`, not planned at design time** (the [`agent-loop-skeleton` delta](../agent-loop-skeleton/spec.md)'s own `R-LSK-008` consequence 6 owns the full account). `TestScopeFence_S_TLS_020_SeamRidesBesidePolicySlotNotInsideIt`'s own `NumMethod()`-and-named-set assertions (`:102-105`, cited above) and the event-kind-count assertion (`:87`, cited above) both stay byte-unchanged and green; the ONE edit converts that same test's own anti-vacuity floor — an empty `git diff` over `scheduler.go` — from `t.Fatal` to `t.Skip`, because an empty diff means the absence it checks holds vacuously on a branch (any branch after AG-19's merge, AG-20 included) that does not itself touch `scheduler.go`. `cancellation_test.go` carries the identical fix for its own mirror (`S-DEL-015`) but was never named byte-unchanged by this requirement to begin with.
5. **`doc.go` and `doc_contract_guard_test.go` are ALSO deliberately ABSENT from the byte-unchanged list above — a second, unrelated narrow release, discovered during `sdd-verify` (CRITICAL-2), not planned at design time and not the same release consequence 4 records.** `L2C-08`'s package-wide contract row (`R-CAN-008`) claimed *"every goroutine the package itself owns has exited"* past the wind-down bound; AG-20's own observer lane goroutine may legitimately still be running, by design, while blocked inside a permanently stalled observer (`R-HKS-008` consequence 1), so the claim was false of the shipped code and no delta recorded it. Fixed by widening `L2C-08`'s existing third-party-tool carve-out to also name a permanently stalled third-party observing-hook invocation, reported typed by hook point and registration index — the identical mechanism the row already uses, not a weakening. The [`agent-cancellation-tree` delta](../agent-cancellation-tree/spec.md) (`R-CAN-006`, `R-CAN-008`, `S-CAN-016`, `S-CAN-017`) owns the full account. Both files are pre-existing entries in both substrate filters since AG-18 — the milestone that took the only PRIOR release of these same two files, for the unrelated reason of amending `L2C-07` — so this release needs no new filter entry either.

#### Scenarios

- **S-HKS-024** — Given the merge base of this change's branch with `origin/main`, when `git diff` is taken over `backend/agent/`, then every file named byte-unchanged above is byte-unchanged; the diff under `backend/agent/src/ai/` and under `backend/agent/src/agenttest/` is **empty**; the `go.mod`/`go.sum` diff is empty; the event-kind guard passes at its committed kind count with AG-20 registering none; the every-kind-constructible guard passes; no turn outcome member and no cost label member was added; and `Turn`'s and `Run`'s signatures and `Harness`'s exported method set are unchanged.

---

### R-HKS-011 — Closed-sequence safety: three new hook points, one new goroutine class, and what each does NOT move

AG-20 adds three firing points and a per-run observer lane. **A new producer or a new goroutine landing inside a requirement that never mentions it is this repository's known spec-breakage shape**, so the reasoning is recorded here rather than left to be rediscovered.

| Rule at risk | Verdict | Mechanism that keeps it true |
|---|---|---|
| `R-AEV-010` / `R-AEV-013` / `R-DEL-009` — the event registry's exact kind count | **holds** | AG-20 registers no kind; the report is a Go-side typed value (`R-HKS-008`) |
| `R-AGE-008` / `S-AGE-010` — no path from a stalled observer back to the producer | **holds, and is DELIVERED mechanically** | the lane is fed by a non-blocking enqueue and owns its own goroutine; the observer context is value-stripped, so the publishing seam is unreachable from an observer (`R-HKS-007`) |
| `R-RUN-003` — one stamping writer, one contiguous lane | **holds** | the observer lane emits **nothing**; it stamps no event and touches no sink |
| `R-CST-004` — the cumulative is the sum over every per-turn cost event in the run bracket | **holds, unamended** | the per-logical-turn accumulator is a run-frame local, is read-only with respect to the event path, and folds no foreign figure (`R-HKS-004`) |
| `R-CST-005` — zero or more estimates then exactly one final immediately before the run-close | **holds** | post-turn enqueues emit no event, and the terminal snapshot runs strictly **after** the run-close was sent |
| `R-CAN-002` / `S-CAN-013` — the enumerated wind-down tail | **holds** | wind-down fire sites enqueue only; the tail's event sequence is unchanged, asserted by `R-HKS-012`'s byte-identity |
| `R-CMP-003`'s iff, `R-CMP-010`, `R-CMP-013` — a compaction failure is stream-visible and never interrupts the run | **holds** | the pre-compact splice is pre-provider and its failures land on the **existing** arm (`R-HKS-003`) |
| `R-CMP-004` — resolution never expands the cut forward | **CLARIFIED** by this change's `agent-compaction` delta | the clause governs **resolution**, never the **request**; a forward-adjusting hook re-designates the request and resolution retracts from it as always |
| `R-PRH-002` — the hook's returned request *"(and only that value)"* reaches the provider | **AMENDED** by this change's `agent-pre-request-hook` delta | under a chain, element *n*'s output flows to element *n+1*; the clause is re-scoped to the chain's **final** output |
| `R-PRH-005` — a nil singular hook skips the seam | **AMENDED** by the same delta | the condition becomes nil singular **and** empty chain |
| `S-LSK-001`'s length-equality sequence, `R-LSK-002`'s statelessness pins | **hold** | all construct a zero-value `TurnOptions`; `R-HKS-012`'s inertness makes the nil path byte-identical |
| `R-RUN-010` — no third path and no timeout | **holds, back-annotated** | AG-20 adds no wall clock, no timeout and no join; the module's no-deadline pin stays **file-unchanged** |
| `R-CAN-006` / `R-CAN-008` (`L2C-08`) — the wind-down bound's harness-owned-tasks table and the package-wide "every goroutine…has exited" contract row | **AMENDED** by this change's `agent-cancellation-tree` delta, discovered during `sdd-verify` (CRITICAL-2), not planned at design time | the table missed its own case at design time: the observer lane's drain goroutine is package-owned and, by design, MAY outlive the bound while blocked inside a stalled observer (`R-HKS-008` consequence 1) — neither the tool's own frame nor an enumerated harness-owned task. The delta widens both requirements to name it as a third, disjoint carve-out — reported typed by hook point and registration index, mirroring the existing third-party-tool carve-out — rather than leave the blanket claim false of the shipped code |

Any milestone that later adds a hook point, a fan-out lane, a deregistration surface or a wall-clock threshold MUST re-check every row above before doing so.

#### Scenarios

- **S-HKS-025** — **The table is checked, not asserted.** Given the merged change, when each row above is evaluated against the shipped code and the shipped suites, then every "holds" row's owning test passes **byte-unchanged**; the three "AMENDED" rows each resolve to a delta in this change carrying the amendment; the "CLARIFIED" row resolves to a delta whose text scopes the clause without weakening it; and no row's owning requirement was edited without a delta in this change naming it.

---

### R-HKS-012 — Inertness: a zero-value `Hooks` is BYTE-IDENTICAL to today on every path

When no hook of any family is registered, the system MUST create no lane, start no goroutine, allocate no queue, and invoke nothing. The nil path MUST be byte-for-byte today's path on **every** arm — success, turn failure, retry exhaustion, interrupt, shutdown, and a compaction bracket on both doors.

`Hooks`'s zero value MUST be inert, matching the house pattern of every other optional seam on the harness. Because `Hooks` holds function fields and is therefore not comparable, the inertness test MUST be an explicit predicate, never an equality against a zero value.

#### Scenarios

- **S-HKS-026** — **The nil-`Hooks` path is deterministic on repetition, and no lane goroutine exists on it — corrected by `sdd-verify` (MAJOR-1) from an overreaching original.** Given SEVEN paired runs of identical scripts — success, turn failure, retry exhaustion, interrupt, shutdown, a compacting run through the strategy-verdict door, and a compacting run through the on-demand door — each pair run **TWICE ON THIS BRANCH** with a zero-value `Hooks`, when both runs of each pair are recorded, then each pair's event-**kind** sequences are equal, and `runtime.NumGoroutine()` returns to its baseline after every one of them — no lane goroutine exists on the inert path. **This is deliberately NOT** "one of each pair on the merge base and one on this change" (the original text's claim, corrected here): AG-19's `inert_path_test.go` earned that comparison explicitly, by showing its own new code is never read on the hookless path at all (a caller-invisible seam behind an opt-in accessor). AG-20 does **not** re-earn it — `applyPreRequestHook`'s chain composition in `loop.go`, the `turnCost`/`capturedOutcome` capture locals in `harness.go`, and the `len(chain) > 0` guard in `compaction.go` all execute **unconditionally** on the hookless path too, so "twice on this branch" and "once on the merge base, once here" are demonstrably not the same experiment for AG-20 the way they were for AG-19. Equivalence to the pre-AG-20 behavior is a **structural** claim, not one this scenario measures behaviorally: every new branch the nil path could reach is gated by an `isZero()`/`len(...) == 0` predicate visible at the call site (`hooks.go`'s own `isZero`, and the three call sites named above), never by a value depending on runtime state. History read-backs and captured provider requests are **not** compared by this scenario — only the event-kind sequence and the goroutine count are (the same "byte-identical" == kind-sequence-equal shorthand this delta's `S-HKS-007`, `S-HKS-017` and `agent-run-driver`'s `S-RUN-113` also use, and no more than they prove either).

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-HKS-001** | **External-package verifiability.** Every behavioral scenario MUST be verifiable from `package agent_test`. A behavior reachable only from inside the package is, for this spec, not reachable at all (`NFR-CMP-001`, `NFR-DEL-001`). The cut-resolution fixed-point table of `S-HKS-007` MAY additionally be exercised internally under `NFR-CMP-001`'s pure-helper carve-out, **provided** the observational half — the identical-plan byte-identity — is asserted externally, which `S-HKS-007` requires. |
| **NFR-HKS-002** | **Determinism, race cleanliness, and NO CLOCK.** Every test MUST be hermetic and MUST pass under `-race`. Synchronization MUST be by the package's test gate primitive, channel reads, channel closes, and the snapshot-or-count lemma of `S-HKS-011`. **No test and no production path may synchronize by sleep, timeout, deadline, poll or wall-clock ordering** — `R-RUN-010` is the precedent and `NFR-CST-002`, `NFR-CTX-002`, `NFR-CMP-002`, `NFR-RUN-002`, `NFR-CAN-002` and `NFR-RTY-002` each already ban it. Evidence MUST be recorded with **`-count=1`** and the wall-clock duration recorded with it; the real uncached suite for this module is ~170s, so a sub-second pass is a cache artifact, not evidence. |
| **NFR-HKS-003** | **Ambient authority and boundaries.** Production sources added by this change MUST NOT import process, filesystem, environment or network facilities; the ambient-authority and import-boundary guards MUST pass with **zero** change to their own sources. |
| **NFR-HKS-004** | **Substrate.** Every file named by `R-LSK-004` MUST be byte-unchanged. **This change releases none of them.** The conclusion, together with the exact-filename widening both substrate filters owe, is recorded in this change's [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md) delta as `R-LSK-008` / `S-LSK-032`. A later phase electing to edit a forbidden file MUST take its own recorded release; it is **not** pre-granted here. |
| **NFR-HKS-005** | **Goroutine-leak discipline: release before baseline.** A test that gates an observing hook MUST NOT assert a goroutine baseline; `runtime.NumGoroutine()` sampling lives only in ungated tests (`S-HKS-026`). Every gate MUST be released — inline once its assertions no longer need the hold, and unconditionally through a cleanup release registered at the gate's construction — so no gate outlives its test and the lane goroutine drains and exits before the next test's baseline. The permanently-stalled leak MUST be exercised only under a cleanup-released gate. **AG-21 inherits this rule by name.** |
| **NFR-HKS-006** | **Review budget.** This change ships as a **single** pull request under a pre-accepted `size:exception` against a 1000-line budget counted **excluding every path under `openspec/`**, which the user designated a working folder. The pull-request description MUST state why the change does not fit the default budget. The reserve slicing boundary is the proposal's Approach ordering: **U1** the hook types, the lane and the pre-request chain · **U2** session-start and post-turn · **U3** pre-compact and AG-20.2. |

## Explicit non-requirements — what this spec does NOT claim

| Not claimed | Owner and why the deferral is safe |
|---|---|
| Any concrete hook — cache breakpoints, compaction policy, telemetry | **Layer 3**, doc 0004 CO-24.1/CO-24.2, verbatim from the charter (`0003:1874`). `S-AGS-048` already names those nodes |
| Removing `TurnOptions.PreRequestHook` | **AG-23** (the frozen Layer 3 surface). Kept and superseded here, never broken |
| Multi-consumer fan-out or observer attachment machinery | **AG-01.1 decided it; nobody has built it.** AG-20 makes a *hook* non-blocking; it does not ship `R-AGE-009`'s consumer fan-out |
| Hook deregistration, priority, filtering or conditional registration | **Not this milestone.** Registration order is the whole ordering contract (`0003:1898-1901`) |
| Any wall-clock timeout, deadline or "slow hook" threshold | **Never in Layer 2.** `R-RUN-010`; a threshold is Layer 3 policy (`R-AGS-015`) |
| Panic recovery for **mutating** hooks | **Not this milestone / AG-21.** They run inline on the caller's goroutine, exactly as AG-08 shipped |
| Cross-run hook state, or a session outliving a `Harness` value | **AG-21.** The latch is per-value bookkeeping, the shipped shutdown-flag shape |
| Bounding, joining or killing a stalled observer | **Never.** It is the caller's goroutine and the caller's leak (`R-HKS-008` consequence 1); AG-21 inherits it knowingly |
| A new `EventKind`, turn outcome or cost label | **Never in AG-20** (`R-HKS-008`, `R-HKS-010`) |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2** (`R-RUN-012`). Layer 1 is consumed, never edited |
| Widening `agenttest` | **Not this milestone.** The gate primitive is used, not widened; a widening requires a recorded reason at apply time |
| Concurrent `Run` calls on **one** `Harness` value | **Not this milestone** (`R-CAN-002`'s one-run-at-a-time clause). The session-start latch is not made concurrency-safe beyond the exclusion it already shares |

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active. Both leaves are behavior, so both are RED-first.

The five bites MUST each be RED-recorded **before** its corresponding scenario is GREEN, with `-count=1`, then reverted:

| Bite | Sabotage | Scenario it must break | Recorded failure |
|---|---|---|---|
| `S-HKS-050` (a) | report every enqueued observer regardless of state | `S-HKS-019` | non-empty report set with no stall present |
| `S-HKS-051` (b) | dispatch observing hooks synchronously at the fire site | `S-HKS-018` | assertion failure naming the harness run frame in the captured stack |
| `S-HKS-052` (c) | dispatch one point in reverse | `S-HKS-016` | recorded sequence `[3, 2, 1, 0]` |
| `S-HKS-053` (d) | skip the post-hook cut re-resolution | `S-HKS-008` | committed prefix splits a call/result pair |
| `S-HKS-054` (e) | fire session-start per `Run` rather than behind the latch | `S-HKS-013` | report count two, **or** a non-empty second-run stall report |

**`S-HKS-020`'s panic RED is a process-crashing panic when recovery is removed.** It MUST be recorded from an **isolated** `go test -run` invocation naming only that scenario, with `-count=1`, in its own process; its evidence is the non-zero process exit status together with the panic trace, **not** a `--- FAIL` line, which will not be produced. This is the `S-DEL-022` precedent, and recording it inside a full-suite run would destroy that run's other evidence.
