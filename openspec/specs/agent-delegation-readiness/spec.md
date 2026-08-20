# Spec — Re-entrancy proven: the delegation seam (`agent-delegation-readiness`)

> **Change**: `cachicamas-agent-delegation-readiness` · **AG-19** (Layer 2, Wave 5, milestone 19 of 24), charter `0003:1793-1862`
> **NEW capability**, minted by this change (AG-19). Promoted to `openspec/specs/agent-delegation-readiness/spec.md` at archive.
> **Nodes**: AG-19.1 `[leaf]` (nested run and events) · AG-19.2 `[leaf]` (nested cancellation) · AG-19.3 `[leaf]` (nested cost and permission scope)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && make test`. **`-count=1` is mandatory**: the real uncached agent-module suite is ~170s, so a sub-second pass is a cache artifact and not evidence.
> **IDs**: `R-DEL-0NN` / `S-DEL-0NN` / `NFR-DEL-0NN`, **append-only**. Allocated `R-DEL-001`…`R-DEL-010`, `S-DEL-001`…`S-DEL-025` and `NFR-DEL-001`…`NFR-DEL-005`, of which `S-DEL-020`…`S-DEL-023` are the four **bites**. The header states the allocated **range** and never a total, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`). Prefix `DEL` verified free under `openspec/specs/` before minting, re-verified in this phase: zero occurrences.
> **Sources**: charter `0003:1793-1862`; this change's `proposal.md` (Q1–Q6, binding) and `design.md` (**AD-1…AD-7, binding and not re-opened here**). Where this spec differs from the proposal it follows the design — in particular the admissibility predicate is tightened by AD-2's registry-membership gate, and AD-3 decides the child's `cost_session` crossing that the proposal deliberately left open. Archived at `openspec/changes/archive/2026-08-19-cachicamas-agent-delegation-readiness/`.
> **Ownership boundary**: this capability owns the publishing seam, its admissibility rule, its lifetime, and the four structural properties AG-19 proves — parent attribution, two-lane reconstruction, nested cancellation and nested cost/permission. It does **not** own the event vocabulary ([`../agent-protocol-events/spec.md`](../agent-protocol-events/spec.md)), the envelope ([`../agent-event-envelope/spec.md`](../agent-event-envelope/spec.md)), the cost contract ([`../agent-cost-events/spec.md`](../agent-cost-events/spec.md)), the cancellation tree ([`../agent-cancellation-tree/spec.md`](../agent-cancellation-tree/spec.md)), the permission gate ([`../agent-permission-protocol/spec.md`](../agent-permission-protocol/spec.md)) or the scheduler ([`../agent-tool-scheduler/spec.md`](../agent-tool-scheduler/spec.md)).
> **Every `file:line` cited below was opened in this worktree during this phase, against `main@558641f3`.** No citation is carried forward from `explore.md`, `proposal.md` or `design.md` unresolved.

## Purpose

The harness is documented as re-entrant. Nothing proves it, and one thing quietly forbids it: `Tool.Run(ctx, args, policy PolicySlot)` receives no sink, no stamper and no scheduler handle (`tool.go:182-186`). A tool can build a child `Harness` today, and that child's whole conversation would be invisible to the parent's consumer. There is no door.

AG-19 opens the smallest door that makes the substrate's five milestones of preparation checkable, and proves the four structural properties on it. **It ships no subagent tool.** Everything that knows what a subagent *is* lives in `package agent_test`, which production code cannot import — the charter's "no subagent tool ships in v1" is enforced by the compiler, not by a convention.

## Coverage — the charter's five Gherkin scenarios, each mapped

Every charter leaf traces to at least one requirement **and** at least one scenario. No leaf is reduced and no scenario is orphaned.

| # | Leaf | Charter scenario | Lines | Owning requirement(s) | Scenario(s) |
|---|---|---|---|---|---|
| 1 | AG-19.1 | "a child harness runs inside a tool, on the parent's stream" | `0003:1818-1821` | `R-DEL-001`, `R-DEL-002`, `R-DEL-005` | `S-DEL-001`, `S-DEL-004`, `S-DEL-010`, bite `S-DEL-021` |
| 2 | AG-19.1 | "…and a consumer separates the two conversations by walking parents" | `0003:1822` | `R-DEL-004` | `S-DEL-011`, `S-DEL-012` |
| 3 | AG-19.1 | "sibling children interleave without cross-talk" | `0003:1824-1827` | `R-DEL-005` | `S-DEL-013` |
| 4 | AG-19.2 | "the tree cancels leaf-first" | `0003:1837-1840` | `R-DEL-006` | `S-DEL-014`, `S-DEL-015`, bite `S-DEL-023` |
| 5 | AG-19.3 | "child cost aggregates and stays attributable" | `0003:1850-1853` | `R-DEL-007` | `S-DEL-016`, `S-DEL-017`, bite `S-DEL-020` |
| 6 | AG-19.3 | "child permission flows through a derived scope to one place" | `0003:1855-1859` | `R-DEL-008` | `S-DEL-018`, `S-DEL-019` |

Cross-cut requirements carrying no charter leaf of their own: `R-DEL-003` (seam lifetime and revocation — a hazard this change *found* rather than inherited: `S-DEL-006`…`S-DEL-009`, bite `S-DEL-022`), `R-DEL-009` (the scope fence, `S-DEL-024`) and `R-DEL-010` (closed-sequence safety, `S-DEL-025`).

## Two charter reinterpretations, recorded as such

Both are recorded here rather than left to inference, following the AG-06 (`0003:613`) and AG-16 precedent. A later reviewer reading the charter alone would otherwise read a stronger obligation than AG-19 delivers, and would be right to.

| Charter phrase | Normative reading | Why the literal reading is rejected |
|---|---|---|
| "the child's events **each parent-identified**" (`0003:1821`) | **A walkable tree in one hop** (`R-DEL-004`): every child event carries `Run()` = the child run identity, and exactly one `subagent_started` on the parent's stream carries that same `Run()` together with `Parent()` = the parent run identity. No per-event parent field is added. | Only three constructors in a 25-kind vocabulary accept a parent (`delegation_events.go:62,134` and `NewDelegatedRunStart`); every message, tool, permission, cost and turn constructor takes none (`permission_events.go:138,242,326`; `cost_events.go:193,296`). Making the clause literal rewrites the whole construction surface and falsifies `R-AEV-003`'s "a top-level event reports no parent as a distinguishable false". The charter states its own mechanism in the very next clause: *"a consumer separates the two conversations by walking parents."* |
| "child cost **aggregates into the parent's cumulative**" (`0003:1853`) | **A consumer-side reconstruction** (`R-DEL-009`). No Layer-2 fold occurs, and the seam makes the fold *impossible* rather than merely discouraged. | `R-CST-004` is shipped and machine-tested: the cumulative is the sum over every `cost_turn` emitted **within that run bracket**, and a child's `cost_turn` is emitted inside the *child's* bracket. Folding it would ship a violation of a requirement `S-CST-008`/`S-CST-021` already defend. The charter's own qualifier agrees — *"a frontend can show both 'this subagent cost X' and 'the run cost Y'"* is **two displayed numbers**, not one merged Layer-2 event. |

## Requirements

### R-DEL-001 — One context-carried publishing seam, minimal and unmintable, naming no subagent concept

The system MUST expose exactly one sanctioned route from inside a `tool.Run` frame back onto the hosting run's event lane: a **context-carried seam** reachable through a named exported accessor. Its whole exported surface MUST be an interface carrying exactly two methods — one reporting the hosting run and turn identities, one offering a single `Event` and returning an error — plus that accessor and the two sentinel errors of `R-DEL-002` and `R-DEL-003`.

`Parent()` is **unavoidable rather than convenient**: `Tool.Run` receives no run or turn identity, so without it `NewSubagentStarted(childRun, parentRun, parentTurn, id)` is unconstructible. It exposes only what `Event.Run()` / `Event.Turn()` already expose on every delivered event.

The concrete seam value, its context key, its constructor, its installer, its admissibility predicate and its revocation MUST all be **unexported**. No code outside the package MAY construct or install a seam; only the scheduler's per-call installation mints one. The seam MUST name **no** subagent concept in any identifier: it has no notion of depth, configuration, child lifecycle, or subagent anything. It publishes *events*.

The seam MUST NOT introduce a new channel, a new loss rule, or a second stamping writer. It MUST ride the scheduler's existing emission funnel, so the mirrored event is stamped by the **one** dispatcher goroutine that already stamps every sibling tool call's events, and the lane keeps exactly one stamping writer (`R-AGE-017`, `R-RUN-003`, `sequence.go:13-24`).

Installation MUST add **zero** type assertions to the scheduler source. The seam's single type assertion MUST live in the seam's own file, on the seam's own type, so the `PolicySlot` source guard (`scheduler_test.go:616-650`) passes **unchanged** and `PolicySlot`'s single documented meaning (`R-TLS-002`, seam 3) is not overloaded.

#### Scenarios

- **S-DEL-001** — **Charter AG-19.1 scenario 1, the door half.** Given a tool scheduled by the ordinary path, when its `Run` frame asks the context for a seam, then one is present; the identities it reports equal the hosting run's and the hosting turn's, compared against the `run_start` and `turn_start` events observed on that same recorded stream; and an event published through it is observed by the consumer on the parent's stream.
- **S-DEL-002** — **The seam is unmintable from outside.** Given `package agent_test`, when the package's exported surface is enumerated, then it declares the seam interface, its accessor and the two sentinels and **nothing else** of the delegation seam; no exported constructor, installer, context key or admissibility predicate exists; and no external caller can install a seam onto a context that the scheduler will honour.
- **S-DEL-003** — **The guards are green unchanged.** Given the merged change, when the `PolicySlot` source guard, the ambient-authority guard, the import-boundary guard and the doc-contract guard run, then all four pass with **zero** change to their own sources; and when `scheduler.go` is scanned as raw bytes, then it contains **no** type assertion of any kind added by this change.

---

### R-DEL-002 — Admissibility is TOTAL: a registry-membership gate, three descriptor gates, and one requirement-level carve-out

Publishing MUST decide **every** possible `Event` value. An event MUST be admitted iff, evaluated in this order, all five gates pass:

| # | Gate | Refusal reason |
|---|---|---|
| 1 | The event's kind has a registry entry | unregistered or zero kind |
| 2 | The registered descriptor's bracket role is *none* | the event opens or closes a bracket |
| 3 | The registered cardinality is not *at-most-one* | a second occurrence on one validated slice is a duplicate |
| 4 | The registered descriptor is not terminal | everything after a terminal event on the slice is rejected |
| 5 | The kind is not `cost_turn` | `R-CST-004`, and this gate alone is requirement-level |

A refusal MUST return a typed sentinel distinguishable by `errors.Is` and MUST reach no sink.

**Gate 1 is new relative to the proposal, and it is load-bearing.** A zero `Event` reports kind 0 (`event.go:469-474`) and has no registry row (`event.go:372-377`); its zero-value descriptor fields are exactly *bracket-none*, *cardinality-any* and *non-terminal* (`event_descriptor.go:70-72,102-104,115-118`), so gates 2–4 would **admit** it — and `CheckStream` would then silently skip it (`stream_check.go:112-118`), leaving junk on a stream that still validates. Without gate 1 the predicate is not total; with it, every `Event` value is decided.

The rule MUST be **derived from the existing registry**, not from a hand-maintained kind list, so a kind registered by a later milestone is decided by its own descriptor with no edit here. Over the 25 kinds registered today (`event.go:100-221,242-366`) the rule refuses **6** and admits **19**:

| Kind(s) | Descriptor evidence | Verdict |
|---|---|---|
| `run_start` | opens the run bracket (`event.go:243-246`) | **refuse** — gate 2 |
| `run_end` | closes the run bracket, terminal (`event.go:247-250`) | **refuse** — gates 2 and 4 |
| `turn_start`, `turn_end` | open/close the turn bracket (`event.go:251-258`) | **refuse** — gate 2 |
| the 6 message kinds | turn placement, rest zero-valued (`event.go:259-282`) | **admit** |
| the 5 tool kinds | turn placement (`event.go:285-304`) | **admit** |
| `permission_decision_required`, `permission_decision_made` | turn placement, non-terminal (`event.go:312-319`) | **admit** — the ask and its answer cross **together** |
| `permission_resolution_remembered` | cardinality at-most-one (`event.go:320-323`) | **refuse** — gate 3 |
| `cost_turn` | turn placement, non-terminal (`event.go:329-332`) | **refuse** — gate 5 only |
| `cost_session` | run placement, non-terminal, cardinality **any** (`event.go:333-336`) | **admit** — `R-DEL-009` |
| `subagent_started`, `subagent_ended` | turn placement, non-terminal (`event.go:340-347`) | **admit** — the seam's reason to exist |
| the 3 compaction kinds | turn placement, non-terminal (`event.go:354-365`) | **admit** — a child's compaction is legitimately visible to the human |
| unregistered or zero kind | no registry row | **refuse** — gate 1 |

**The `cost_turn` asymmetry MUST be stated loudly wherever this rule is restated.** It is the one refusal `CheckStream` would happily accept: it is a **requirement-level** rule (`R-CST-004`), not a stream-legality rule, and it exists because the funnel terminates in exactly the channel whose per-attempt forwarder folds by payload type with **no run-identity filter** (`harness.go:633-635`). A mirrored child `cost_turn` would be folded into the parent's cumulative **silently**.

Gate 4 is today implied by gate 2 — the only terminal kind is `run_end`, which also closes the run bracket — and MUST be kept anyway, as defence against a future terminal kind with no bracket role.

The mirroring rule MUST NOT be delivered as a hand-written allow-list of kinds. The permission ask and its answer MUST cross together or not at all: a mirrored ask with no mirrored answer is unreadable to the human watching.

#### Scenarios

- **S-DEL-004** — **The rule is total, kind by kind.** Given a table naming every registered kind together with the verdict this requirement's table states, when each kind is published through a live seam, then the observed verdict equals the stated one for **all** of them; each refusal returns the inadmissible sentinel identifiable by `errors.Is` and delivers **nothing** to any sink; and the table's row count equals the number of kinds the registry reports, so a kind registered later cannot be silently absent from the assertion.
- **S-DEL-005** — **The zero `Event` is refused by the membership gate, not admitted by zero-valued descriptor fields.** Given a zero-value `Event`, when it is published, then the publish returns the inadmissible sentinel, nothing reaches any sink, and the parent's recorded stream is byte-identical to the same run whose tool published nothing.
- **S-DEL-021** — **(bite)** RED-first, **mandated**. Given a scratch tree in which gate 2 is removed so the child's `run_start` is admitted, when `S-DEL-010` runs, then it **FAILS** with the parent's `CheckStream` reporting a duplicate run-open (`stream_check.go:122-127`) — proving the bracket gate is what keeps the parent's stream legal rather than a fixture that happened never to mirror one. RED-recorded BEFORE `S-DEL-010` is GREEN, with `-count=1`, then reverted.

---

### R-DEL-003 — The seam's lifetime is exactly the hosting call, and revocation serializes with the send

The seam MUST be installed for exactly the hosting tool call's `Run` frame and MUST be **revoked** when that call completes, on **every** exit path: the normal return, the detached return, and the re-panic path.

Revocation MUST be **serialized with the publish itself under one mutual exclusion**, so that a publish either completes its send strictly before revocation returns, or observes the revoked state and sends nothing. A publish racing revocation MUST return a typed sentinel distinguishable by `errors.Is` from the inadmissibility sentinel — **never a panic, and never a silent drop**. The two reasons MUST be separately identifiable, because a caller must distinguish a programming error (a kind that can never cross — fail the test) from a timing outcome it must tolerate (the call is over — stop publishing).

**Why the weaker primitives are insufficient, recorded so a later reader does not "simplify" it.** An atomic flag makes check-then-send non-atomic: the publisher can be descheduled between the load and the send while revocation and the channel close slide in between, and the send then panics on a closed channel. A `select` on a done-channel does not help either, because Go has no non-panicking send on a closed channel, even inside `select`. **The exclusion must cover the send.**

**The hazard is real and specific to the detached path.** On the ordinary path a tool cannot publish after the channel closes, because the wait-group release is deferred past `tool.Run`. On the detached path the tool's goroutine is abandoned after the wind-down bound (`scheduler.go:627-633`) while the scheduler proceeds to `close(emissions)` (`scheduler.go:240`), so a late publish would send on a closed channel and **panic in a goroutine no recover covers**.

**The lifetime bound is what keeps three other capabilities' closed sequences true**, and MUST be stated here rather than rediscovered: because the seam is dead before the hosting call returns, no mirrored event can ever land after the parent's turn closes. `R-CAN-002`'s enumerated wind-down tail, `S-CAN-013`'s observed order, and `R-CST-005`'s "immediately before the run-close" are therefore untouched by mirroring — not by luck, but by this bound.

Revocation MUST be sequenced so that it runs **before** the call's panic-recovery unwinding completes, so the re-panic path is covered too.

#### Scenarios

- **S-DEL-006** — **Revocation on the normal path.** Given a tool that captures its seam into a value outliving its `Run` frame and publishes an admissible event after `Run` returned, when that publish happens, then it returns the revoked sentinel, delivers nothing, and the parent's run completes normally with a `CheckStream`-valid stream.
- **S-DEL-007** — **Revocation on the detached path.** Given a scheduler with an injected small wind-down bound and a tool that ignores its context and publishes only after a gate the test closes once `Schedule` has returned — so the emission channel is provably already closed (`scheduler.go:240` precedes `:249`) — when that publish happens, then it returns the revoked sentinel, **no panic occurs**, and the test process survives to assert it.
- **S-DEL-008** — **Revocation on the re-panic path.** Given a tool that captures its seam and then panics, when the scheduler's recovery reports the typed panic failure, then a later publish through the captured seam returns the revoked sentinel; and the parent's stream carries the panic's typed tool failure exactly as it does without a seam installed.
- **S-DEL-009** — **The two refusals are distinguishable.** Given a live seam and a revoked seam, when an inadmissible kind is published through the live one and an admissible kind through the revoked one, then the two returned errors match different sentinels by `errors.Is` and neither matches the other's.
- **S-DEL-022** — **(bite)** RED-first, **mandated**, and **isolation is part of the requirement**. Given a scratch tree in which revocation is dropped, when `S-DEL-007` runs, then it **FAILS** by an unrecovered send-on-closed-channel panic in the detached goroutine. **That panic terminates the whole test binary**, so this RED MUST be recorded from an isolated single-test invocation (`go test -run` naming only this scenario, `-count=1`, its own process), and its evidence is the non-zero process exit status together with the panic trace naming the send — **not** a `--- FAIL` line, which will not be produced. Recorded BEFORE `S-DEL-007` is GREEN, then reverted.

---

### R-DEL-004 — Parent identity is a WALKABLE TREE in one hop, not a per-event stamp

A consumer holding the parent's delivered stream and the child's own captured stream MUST be able to attribute **every** child event to the parent by walking exactly one hop, with **no** per-event parent field:

1. every child event carries `Run()` equal to the child run identity;
2. exactly one `subagent_started` on the parent's stream carries `Run()` equal to that same child identity **and** `Parent()` equal to the parent run identity;
3. therefore every child event is attributable to the parent through (2).

No event constructor MAY gain a parent parameter, and `NewSubagentStarted` / `NewSubagentEnded` MUST remain the delegation family's only parent-bearing route into the parent's stream. `R-AEV-003`'s "a top-level event reports no parent as a distinguishable false" MUST stay true unweakened.

A mirrored event MUST be a **distinct value** on the parent's lane: re-stamping discards the prior sequence and returns a copy (`sequence.go:50-58`), so the child's own captured value keeps its own contiguous stamp on the child's lane. `Event` is a value type; publishing a copy cannot perturb the child's lane.

#### Scenarios

- **S-DEL-010** — **Charter AG-19.1 scenario 1, the interleaving half.** Given a test-only tool hosting a child harness running a scripted conversation, when the parent run completes, then the parent's recorded stream carries, in order, exactly one `subagent_started`, the admissible child events of `R-DEL-002`, and exactly one `subagent_ended`, all inside the hosting turn's bracket; `CheckStream` accepts the parent's stream **unmodified** with `stream_check.go` byte-unchanged; and the parent's sequence is contiguous and 1-based across the whole run with no gap, repeat or restart.
- **S-DEL-011** — **Charter AG-19.1 scenario 1, the walking half.** Given those two streams, when a consumer resolves each mirrored child event to a parent using only `Run()` and the single `subagent_started`'s `Parent()`, then **every** mirrored event resolves in one hop; no mirrored event other than `subagent_started`/`subagent_ended` reports a parent of its own; and when the package's event construction surface is enumerated, then no constructor gained a parent parameter in this change.
- **S-DEL-012** — **The two lanes stay independently contiguous.** Given the same run, when the child's own captured stream is walked, then its sequence is contiguous and 1-based independently of the parent's, and the mirrored copies on the parent's lane carry the parent lane's stamps rather than the child's — the two lanes are never merged and no event carries a sequence from the other lane.

---

### R-DEL-005 — The reconstruction is SPLIT-THEN-VALIDATE; the two streams are never concatenated

A consumer MUST validate the parent's delivered stream and the child's own stream **separately**, each with `CheckStream`, and MUST NOT concatenate them into one slice for validation.

The reason is derivable rather than stylistic: `CheckStream` never reads `Event.Run()`; its run bracket, turn-open flag, at-most-once map and terminal latch are **global over the validated slice** (`stream_check.go:92-176`). A concatenation therefore presents two run brackets, two terminals and duplicated at-most-once kinds to a validator that has no way to tell them apart.

`stream_check.go` and `event.go` MUST be **byte-unchanged**: the admissibility rule of `R-DEL-002` exists precisely so the validator needs no amendment.

**The rule MUST hold for N children, not only one.** Sibling tool calls each hosting their own child run MUST produce N+1 independently validatable streams. Cross-talk MUST be impossible **by construction** rather than by test luck: each call holds a *distinct* seam, no seam ever carries another child's events, attribution is `Event.Run()`, and the seams share no memory except the emission channel, whose multi-producer send is the synchronization point the Go memory model already gives every sibling tool call today. Each child harness MUST be a **distinct value** with its own run context, transcript, stamper and sink — concurrent `Run` calls on **one** `Harness` value stay out of scope (`R-CAN-002`'s one-run-at-a-time clause).

#### Scenarios

- **S-DEL-013** — **Charter AG-19.1 scenario 2 — siblings, under `-race`.** Given two sibling read-class tools each hosting its own child run on its own distinct `Harness` value, when both run concurrently within one parent turn, then both children's admissible events appear on the parent's stream; `CheckStream` accepts the parent's stream and **each** child's own stream, all three validated separately and unmodified; the parent's sequence remains contiguous and 1-based; **no** event of either child resolves to the other child's `subagent_started`; and the whole scenario passes under `-race` with no data race reported.

---

### R-DEL-006 — Nested cancellation is inherited through the EXISTING tree and completes leaf-first

A child harness constructed inside a tool MUST derive its run context from the tool's own `ctx`, so the parent's cancellation cause reaches it transitively through the tree AG-14 already built (`harness.go:434`). This change MUST NOT add a nested-cancellation signal, a child-specific cause, a second cancellation primitive or a branch of the tree.

Leaf-first completion MUST be asserted by **observed event order and channel closes**, never by elapsed time:

1. the child's run error matches the interrupt sentinel by `errors.Is` — proving the cause propagated transitively;
2. the child's own stream is `CheckStream`-valid and its `cost_session` carrying the final label precedes its terminal run-close carrying the interrupted outcome (the wind-down order of `R-CAN-002`, `harness.go:343-348`);
3. on the parent's stream, the index of `subagent_ended` is **strictly less than** the index of the parent's run-close;
4. the parent's tool result is **not** a detached-call failure — the call completed inside the bound.

**Assertion 3 is load-bearing, not decorative.** The tool publishes `subagent_ended` only after the child's sink has closed, and the child's sink closes only after its run-close was sent (`harness.go:446`), so parent-lane order **structurally** proves the child's wind-down completed before the parent's did. The parent's lane is single-stamped and contiguous, so index comparison is sound.

The parent's wind-down bound MUST be set **generously** for this scenario. A bound is a ceiling, never a synchronisation point; `NFR-DEL-002`'s no-wall-clock rule forbids synchronising by it, not setting it. The detach framing is deliberately **not** asserted: a detached call produces a typed execution failure (`scheduler.go:1132-1140`) which is a legal stream but a different story from the one the charter describes, and the detach path is already owned and tested by `R-CAN-006` / `R-TLS-014`.

**The structural coupling is recorded so it is inherited knowingly**: a child whose wind-down is slower than the parent's wind-down bound will cause the parent's hosting tool call to detach. That is `R-CAN-006`'s documented behavior, not a defect of this seam, and negotiating the bound for slow children is not in this charter.

#### Scenarios

- **S-DEL-014** — **Charter AG-19.2 scenario.** Given a parent run with an active child harness inside a tool call and a generous parent wind-down bound, when the parent is interrupted mid-flight, then the child's run error matches the interrupt sentinel by `errors.Is`; the child's own stream is `CheckStream`-valid, carries its synthesized orphans, and carries its final cost figure immediately before its run-close carrying the interrupted outcome with a nil failure; the parent's stream is `CheckStream`-valid and `subagent_ended` appears at a strictly smaller index than the parent's run-close; the parent's tool result is **not** a detached-call failure; and no assertion in the scenario reads elapsed time.
- **S-DEL-015** — **The tree is inherited, not rebuilt.** Given the merged change, when the production sources it adds are read, then they introduce no cancel function, no cause value, no deadline and no context derivation of their own; the child's context derivation happens entirely in `package agent_test`; and `S-DEL-014` still passes.
- **S-DEL-023** — **(bite)** RED-first, **mandated**. Given a scratch tree in which the test tool builds its child harness on an independent background context instead of the tool's own `ctx`, when `S-DEL-014` runs, then it **FAILS before assertion 1 is ever reached**: the child's own scripted gate release depends on ctx cancellation, which now never arrives, so the child blocks there permanently, the run never completes, and `drainSink`'s own 1-second timeout fires ("sink did not close within 1s") — a stronger demonstration (total non-termination) than a mismatched sentinel would have been — proving the inheritance is the tool's context derivation rather than an ambient effect. RED-recorded BEFORE `S-DEL-014` is GREEN, with `-count=1`, then reverted.

---

### R-DEL-007 — Child cost is a CONSUMER-SIDE reconstruction; no Layer-2 fold, ever

Layer 2 MUST NOT fold any child figure into any parent figure. The refusal of `R-DEL-002` gate 5 is **production behaviour**, not test discipline: it makes the fold unreachable rather than merely discouraged.

The child's `cost_session` carrying the final label **DOES** cross onto the parent's stream, and it is the natural carrier of the frontend's "this subagent cost X" on the one stream a human watches. Two facts make that safe, both verified rather than assumed:

1. **Stream-legal.** `cost_session` registers run placement, non-terminal, with **zero-value cardinality**, and the zero value is *any*, not *at-most-one* (`event.go:333-336`, `event_descriptor.go:115-118`). Two `cost_session` events on one validated slice trip neither the cardinality rule (`stream_check.go:166-171`) nor placement — the default branch checks only turn placement against the turn-open flag (`stream_check.go:157-163`), and a run-placed event inside the parent's open turn passes.
2. **It cannot reach the parent's cumulative.** The per-attempt forwarder folds only the `cost_turn` payload (`harness.go:633-635`); a `cost_session` payload never matches.

**A consumer MUST discriminate the two by `Event.Run()`, and labels MUST NOT be the discriminator.** Re-stamping rewrites only the sequence (`sequence.go:54-58`), so the crossed child event still reports the **child's** run identity while the parent's own final figure reports the parent's. A consumer summing by label would double-count; a consumer summing by run identity cannot.

The reconstruction a consumer performs MUST be: the parent's cumulative is the sum over the `cost_turn` events on the parent's **own** stream; the combined figure is that sum plus the sum over the child's own stream, joined by the walk of `R-DEL-004`. `R-CST-004` MUST be shown **still true** against the shipped code rather than cited.

#### Scenarios

- **S-DEL-016** — **Charter AG-19.3 scenario 1, and the inequality is what makes it non-vacuous.** Given a child run that provably spends **non-zero** tokens, when the parent's cost events are inspected, then the parent's terminal `cost_session` figures equal the sum over the `cost_turn` events observed on the **parent's own** stream, and are **strictly less than** the parent-plus-child sum on at least one reported figure; a consumer walking both streams by parent identity recovers the combined figure; and no `cost_turn` carrying the child's run identity appears anywhere on the parent's stream.
- **S-DEL-017** — **The crossed `cost_session` is present, attributable and inert.** Given that same run, when the parent's stream is walked, then it carries the child's final `cost_session` reporting the **child's** run identity, positioned inside the hosting turn's bracket; the parent's own final `cost_session` reports the parent's run identity and sits immediately before the parent's run-close; the two are distinguished by run identity and **not** by label; and `CheckStream` accepts the parent's stream unmodified.
- **S-DEL-020** — **(bite)** RED-first, **mandated**. Given a scratch tree in which gate 5 is removed so a child `cost_turn` is admitted, when `S-DEL-016` runs, then it **FAILS**: the parent's reported terminal `cost_session` (92) no longer equals the run-filtered reconstruction of the parent's own `cost_turn` events (15), and a `cost_turn` carrying the child's run identity is found on the parent's stream. The parent's cumulative **is** inflated by exactly the child's spend, but the **strict inequality itself does not break** — the filtered parent-own figure stays pinned at 15 and the parent-plus-child combined figure stays at 92, neither one moving between the shipped tree and this bite, so parent-own remains strictly less than combined regardless. The equality and absence failures alone prove the refusal is what protects `R-CST-004` and that the forwarder's unfiltered fold (`harness.go:633-635`) is a live hazard rather than a hypothetical one. RED-recorded BEFORE `S-DEL-016` is GREEN, with `-count=1`, then reverted.

---

### R-DEL-008 — A derived permission scope NARROWS and never widens; the ask goes up, the decision comes down

A child's permission policy MUST be an ordinary composition of the parent's policy — a value in `package agent_test` implementing the existing policy interface (`permission_protocol.go:80-94`) by delegating to the parent's. This change MUST NOT add a scope type, a rule set, or a mode flag to Layer 2; "scope" is not a Layer 2 concept and this change does not make it one (`permission_protocol.go:77-79`, CO-03).

**"What the parent's policy allowed flows down" MUST be given an operational meaning, and both directions MUST be asserted.** A derived scope MAY only **narrow** — mapping a parent allow to a deny or a defer for a tool outside the child's grant — and MUST NOT **widen** — it MUST NOT map a parent deny or defer to an allow.

**Ask up**: the child's `permission_decision_required` is emitted on the child's stream by the child's own scheduler and is **mirrored** through the seam onto the parent's stream, which is the one place a human watches. Its answering `permission_decision_made` MUST cross with it.

**Decision down**: the human's verdict, given on the parent's surface, MUST reach the child's suspension through the **existing** wake surface (`scheduler.go:264-272`), with **zero** new production routing surface. The gate re-enters resolution on wake (`scheduler.go:772-783`) and the derived scope answers with the human's verdict. This discharges `S-AGE-022`'s pre-stated routing obligation by exercising it, and `R-AGE-017` holds **unamended**: no channel is invented.

#### Scenarios

- **S-DEL-018** — **Charter AG-19.3 scenario 2, the narrowing half, asserted in both directions.** Given a child whose scope is derived from the parent's policy, when a tool the parent allows but the child's grant excludes is called, then the derived scope returns a non-allow outcome; and when a tool the parent denies or defers is called, then the derived scope returns **no** allow for it under any input — the widening direction is asserted as impossible, not merely unused; and when Layer 2's exported surface is enumerated, then it declares no scope type, rule set or mode flag.
- **S-DEL-019** — **Charter AG-19.3 scenario 2, the one-place half.** Given a child call that needs permission, when the child's ask is raised, then both the ask and its answering decision appear on the **parent's** stream in that order; when the test — playing the human on the parent's stream — answers through the child scheduler's existing wake surface, then the child's suspended call resumes with that verdict; both streams remain `CheckStream`-valid validated separately; the child's `permission_resolution_remembered`, if any, appears **only** on the child's stream; and the production diff adds no routing surface.

---

### R-DEL-009 — The scope fence: what AG-19 proves, and what it deliberately does not ship

AG-19 MUST register **no** new `EventKind`, add no new outcome or label member, and change no existing exported signature: `Tool`, `PolicySlot`, `Harness` and the permission policy interface MUST keep their signatures. `event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `go.mod`, `go.sum` and every file under `backend/agent/src/ai/` MUST be **byte-unchanged**, and the every-kind-constructible guard MUST pass at its committed kind count.

**No subagent tool, subagent configuration or depth limit ships** (`0003:1803`). The enforcement is structural: every subagent concept lives in `package agent_test`, which production code cannot import, and the seam is unmintable from outside the package.

Three consequences MUST be stated here rather than discovered by a later consumer:

1. **Any tool holding the seam can publish any admissible event onto its hosting run's stream.** The seam is deliberately not subagent-shaped, so this capability is bounded by the admissibility rule and by nothing else. A narrower, subagent-shaped door would have a smaller blast radius and a larger charter risk; the trade was made knowingly.
2. **The delegation family's zero-production-caller state is exactly what a revert restores.** No data migrates, nothing persists across processes, and no consumer outside the test surface can hold a seam.
3. **The child's own `cost_turn` events never appear on the parent's stream at all.** A consumer that wants per-turn child spend must read the child's stream; the parent's stream carries the child's session-level figure only.

#### Scenarios

- **S-DEL-024** — Given the merge base of this change's branch with `origin/main`, when `git diff` is taken over `backend/agent/`, then every file named byte-unchanged above is byte-unchanged, the diff under `backend/agent/src/ai/` is empty, the `go.mod`/`go.sum` diff is empty, the every-kind-constructible guard passes at its committed kind count with AG-19 registering none, and no exported signature named above changed.

---

### R-DEL-010 — Closed-sequence safety: two newly emitted kinds and a second lane, and what each does NOT move

AG-19 is the delegation family's **first production emission** after thirteen milestones constructible-but-unemitted, and it mirrors up to 19 kinds onto a second lane. **A new event producer landing inside a requirement that never mentions it is this repository's known spec-breakage shape**, so the reasoning is recorded here rather than left to be rediscovered.

| Rule at risk | Verdict | Mechanism that keeps it true |
|---|---|---|
| `R-RUN-003` — exactly one run bracket, N turn brackets, one contiguous lane | **holds** | gate 2 refuses every bracket-opening and bracket-closing kind, so no mirrored event is a bracket; re-stamping by the one dispatcher keeps the lane contiguous |
| `R-AEV-004` — exactly one run-start first and one run-end last | **holds** | gates 2 and 4 |
| `R-CAN-002` / `S-CAN-013` — the enumerated wind-down tail | **holds** | `R-DEL-003`'s lifetime bound: the seam is dead before the hosting call returns, so no mirrored event can land after the parent's turn closes |
| `R-CST-004` — cumulative is the sum over every `cost_turn` in the run bracket | **holds** | gate 5; no foreign `cost_turn` ever reaches the parent's lane |
| `R-CST-005` — zero or more estimates then exactly one final immediately before the run-close | **AMENDED** by this change's `agent-cost-events` delta | a mirrored child final `cost_session` is a second final landing mid-bracket; the requirement is re-scoped to the run's **own** `cost_session` events, discriminated by `Event.Run()` |
| `R-AEV-003` — an event belonging to a delegated harness carries its parent identifier | **AMENDED** by this change's `agent-event-envelope` delta | mirrored child events belong to a delegated harness and carry no parent; the walkable-tree reading of `R-DEL-004` replaces the literal one |
| `S-AEV-022` — the package declares no delegation mechanism | **AMENDED** by the same delta | the seam is named `Delegation…`; the surviving claim is that it declares no **subagent** or child-harness mechanism |
| `R-CST-001`'s iff, `R-CMP-*`'s bracket order, `S-LSK-001`'s length-equality sequence | **hold** | AG-19 opens no bracket and emits nothing on a non-delegating path; a run whose tools publish nothing produces a stream byte-identical to the same run before this change |

Any milestone that later admits a kind this rule refuses, or that publishes through the seam after the hosting call returns, MUST re-check every row above before doing so.

#### Scenarios

- **S-DEL-025** — **The inert path is byte-identical.** Given two runs of an identical multi-turn tool-calling script — one on the merge base and one on this change — where no tool publishes through the seam, when both are recorded, then the two event streams are byte-identical modulo the freshly minted run and turn identifiers, the two history read-backs are byte-identical, and neither carries any delegation-family kind.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-DEL-001** | **External-package verifiability.** Every behavioral scenario MUST be verifiable from `package agent_test`. A behavior reachable only from inside the package is, for this spec, not reachable at all (`NFR-CMP-001`, `NFR-HIS-001`). The admissibility table of `S-DEL-004` MAY additionally be exercised internally, provided every claim about what a **caller** observes is also asserted externally. |
| **NFR-DEL-002** | **Determinism and race cleanliness.** Every test MUST be hermetic and MUST pass under `-race`. Synchronization MUST be by the package's test gate primitive, channel reads and channel closes; **no test may synchronize by sleep, timeout or wall-clock ordering**. The injected wind-down bounds of `R-DEL-006` and `S-DEL-007` are ceilings, never synchronization points. Evidence MUST be recorded with **`-count=1`** and the wall-clock duration recorded with it; the real uncached suite for this module is ~170s, so a sub-second pass is a cache artifact, not evidence. |
| **NFR-DEL-003** | **Ambient authority and boundaries.** Production sources added by this change MUST NOT import process, filesystem, environment or network facilities; the ambient-authority and import-boundary guards MUST pass with **zero** change. |
| **NFR-DEL-004** | **Substrate.** Every file named by `R-LSK-004` MUST be byte-unchanged. **This change releases none of them**: the seam lands in a new file and the installation lands in `scheduler.go`, neither of which is a member of that list. The conclusion, together with the exact-filename widening both substrate filters owe, is recorded in this change's [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md) delta as `R-LSK-007` / `S-LSK-031`. `sdd-design` **declined** the `doc.go` narrative edit, so no release fires; a later phase that elects to edit `doc.go` MUST take its own recorded release first — it is **not** pre-granted here. |
| **NFR-DEL-005** | **Review budget.** This change ships as a **single** pull request under a pre-accepted `size:exception` against a 1000-line budget counted **excluding every path under `openspec/`**. The pull-request description MUST state why the change does not fit the default budget. The reserve slicing boundary is the proposal's Approach ordering: **U1** the seam plus its installation and isolated tests · **U2** AG-19.1 · **U3** AG-19.2 and AG-19.3. |

## Explicit non-requirements — what this spec does NOT claim

| Not claimed | Owner and why the deferral is safe |
|---|---|
| A production subagent tool, subagent configuration, depth limits | **Post-v1** (`0003:1803`, AG-02's verdict). AG-19 proves the substrate; it ships no tool. Enforced by the compiler, not by convention |
| Any Layer-2 fold of child cost into a parent figure | **Never.** `R-CST-004` forbids it and `R-DEL-002` gate 5 makes it unreachable. This is a permanent answer, not a deferral to a later milestone |
| A per-event parent stamp on the other 22 event kinds | **Not built.** `R-DEL-004`'s walkable tree delivers the charter clause, and stamping would falsify `R-AEV-003`. A literal per-event parent field is a vocabulary-wide change and belongs to a different milestone |
| A "scope" type, rule set or mode flag in Layer 2 | **Layer 3 / CO-03** (`permission_protocol.go:77-79`) |
| Concurrent `Run` calls on **one** `Harness` value | **Not this milestone** (`R-CAN-002`'s one-run-at-a-time clause). AG-19's siblings use **two distinct `Harness` values**, noted so a reviewer does not flag a false collision |
| Subagent-scoped retry | **Post-v1.** AG-19 ships no retry surface at all; this change's `agent-retry-failover` delta re-owns the row that named AG-19 as its holder |
| A narrower nested-cancellation signal, a child-specific cause or a nested deadline | **Not built** (`R-DEL-006`). The child participates in the existing tree; it does not get a branch of its own |
| Bound negotiation for a child slower than the parent's wind-down bound | **Not this charter.** The detach that results is `R-CAN-006` / `R-TLS-014`'s documented behavior, recorded in `R-DEL-006` so it is inherited knowingly |
| A quieter parent stream — filtering which admissible child events a human sees | **Consumer-side.** The charter says "the child's events"; a product that wants less applies a consumer filter, not a narrower seam |
| Persistence of a delegation record across processes | **Layer 3 session.** Nothing in this change persists |
| Any new `EventKind`, outcome or label member | **Never in AG-19** (`R-DEL-009`). AG-06.3 minted the delegation family; AG-19 is its first caller |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2** (`R-RUN-012`). Layer 1 is consumed, never edited |
| Widening `agenttest` | **Not this milestone** unless a fixture provably cannot be test-local; `agenttest` is byte-unchanged by default |

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active. All three leaves are behavior, so all three are RED-first. The four bites — `S-DEL-020` (admit `cost_turn`), `S-DEL-021` (admit `run_start`), `S-DEL-022` (drop revocation), `S-DEL-023` (independent child context) — MUST each be RED-recorded **before** its corresponding scenario is GREEN, with `-count=1`, then reverted.

`S-DEL-022` is the one bite whose RED is a **process-crashing panic**. It MUST be run in its own `go test -run` invocation, and its evidence is the process exit status plus the panic trace, not a `--- FAIL` line. Recording it inside a full-suite run would destroy the run's other evidence.
