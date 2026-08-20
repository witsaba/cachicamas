# Spec — The tool execution contract and scheduler (`agent-tool-scheduler`)

> **Change**: `cachicamas-agent-tool-scheduler` · **AG-09** (Layer 2, Wave 2, milestone 9 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-09--define-the-tool-execution-contract-and-scheduler), `0003:902-1004`
> **Nodes**: AG-09.1 `[leaf]` execution contract · AG-09.2 `[leaf]` concurrency policy · AG-09.3 `[leaf]` ordered rejoin · AG-09.4 `[leaf]` failure isolation
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario independently verifiable.
> **IDs**: `R-TLS-NNN` / `S-TLS-NNN`. Distinct from `R-AEV-`/`S-AEV-`, `R-AMT-`/`S-AMT-`, `R-APE-`/`S-APE-`, `R-AGE-`/`S-AGE-`, `R-LSK-`/`S-LSK-`, `R-PRH-`/`S-PRH-`.
> **Scenario count** (AG-04 W9): **11 charter → 12 spec + 6 bites = 18 total** in 11 requirements (+ 1 cross-cut scenario in the `agent-loop-skeleton` delta, `S-LSK-008`).
> **Traces to**: G5 (R-13 — parallel tool execution with deterministic call-ordered rejoin + per-tool concurrency policy); v2 § 6 seams 2 & 3 (seam 3 = the opaque per-call policy slot Layer 2 carries but never reads, the sandbox seam).
> **Depends on**: AG-05 (tool event family merged, `tool_event.go:1-501`); AG-07 (walking skeleton merged at `93077c07`); AG-08 (`PreRequestHook` precedent merged at `e27e8411`). **Parallel with**: none. **Blocks**: AG-10 (permission protocol wraps the scheduler), AG-11 (turn termination types the scheduler's outcomes).

## Coverage

| Charter | Requirements | Spec | Bites |
|---|---|---|---|
| **11 of 11** | 11 (`R-TLS-001`–`011`) | **12** | **6** (`S-TLS-002a`, `S-TLS-005a`, `S-TLS-006a`, `S-TLS-006b`, `S-TLS-010a`, `S-TLS-011a`) |

Charter → spec mapping: AG-09.1 #1 → `R-TLS-001` + `R-TLS-003` / `S-TLS-001`; AG-09.1 #2 → `R-TLS-002` / `S-TLS-002` + bite; AG-09.1 #3 → `R-TLS-007` / `S-TLS-007` (typed-result distinct outcomes); AG-09.2 #1 → `R-TLS-004` / `S-TLS-004`; AG-09.2 #2 → `R-TLS-005` / `S-TLS-005` + bite; AG-09.2 #3 → `R-TLS-006` / `S-TLS-006` + bites; AG-09.3 #1 → `R-TLS-008` / `S-TLS-008` (call-order rejoin); AG-09.3 #2 → `R-TLS-009` / `S-TLS-009` (correlation identities); AG-09.4 #1 → `R-TLS-010` / `S-TLS-010` + bite; AG-09.4 #2 → `R-TLS-011` / `S-TLS-011` + bite (panic containment). Cross-cut (AG-09 → AG-13 boundary) → `S-LSK-008` in the delta spec.

## Purpose

Define what a tool **is** to Layer 2 (a `Tool` interface: `Name()`, `EffectClass()`, `Run(ctx, args, policy)`) and the **scheduler** that runs a turn's requested tool calls under three concurrency policies — reads concurrent with bounded fan-out, mutating + execute serialized in call order — and rejoins results in **call order** regardless of completion order, with one bad tool yielding a typed execution-failure result in its call position while siblings complete. AG-09 also ships the **opaque per-call `PolicySlot`** Layer 2 forwards byte-exact: seam 3 of v2 § 6 — "confinement is a property of the call site, not of the code being called" (`0001:613-616`). AG-09 schedules; it does **not** iterate the model ↔ tools ↔ model cycle — iteration is AG-13's `Harness` job (the wording trap, `0003:107-112`).

## Requirements

### R-TLS-001 — `Tool` interface: declaration + effect class + run (D1)

The system SHALL expose a `Tool` interface with three methods: `Name() string`, `EffectClass() EffectClass`, and `Run(ctx context.Context, args []byte, policy PolicySlot) (Result, error)`. `Name()` MUST return a non-empty string matching `ai.ToolCall.name`; `EffectClass()` MUST be readable WITHOUT invoking `Run`; `Run` MUST return a typed `Result` for success/result-failure and a non-nil `error` only when execution itself fails (mirroring AG-05.2's `ToolOutcome` at `tool_event.go:227-246`).

#### Scenarios

- **S-TLS-001** — AG-09.1 #1 contract from outside. Given an in-test scripted `Tool` whose `Name()` returns `"read_file"`, `EffectClass()` returns `EffectClassRead`, and `Run` returns `(Result{Outcome: Success, Content: …}, nil)`, when an external package reads the three methods, then each returns the expected typed value, `Name()` is non-empty, and `EffectClass()` is reachable without calling `Run`.

### R-TLS-002 — `PolicySlot` is opaque to the scheduler (D3, seam 3)

The system SHALL define `PolicySlot` as a named type over `any` (`type PolicySlot any`). The scheduler MUST NOT type-assert, type-switch, or read the value of `PolicySlot`; it MUST forward the exact bytes/identity of the injected value to the tool's `Run`. This is enforced by (a) source guard test scanning `scheduler.go` for any type assertion on `PolicySlot`, and (b) scripted-tool byte-equality assertion via `bytes.Equal`.

**Back-annotation (AG-19) — a second per-call value now rides the tool's context, and seam 3 is untouched by it.** AG-19 installs a publishing seam onto the context handed to `tool.Run`, inside `executeCall` and around the wind-down call. Five constraints keep this requirement literally true, and each is checkable rather than asserted:

- **`PolicySlot` is not the carrier.** The seam has its own unexported context key and its own type. Reusing `PolicySlot` would overload a seam whose single documented meaning is the permission slot (`scheduler.go:466-471`) — considered and rejected outright.
- **`scheduler.go` gains ZERO type assertions of any kind.** The seam's single assertion lives in the seam's own file, on the seam's own type; even the `context.WithValue` call is placed behind an unexported installer so the scheduler source names neither the key nor the type. The source guard passes with **zero change to its own source**.
- **The forwarded policy value is untouched.** `executeCall` still hands `PolicySlot(call.ID())` through unread and unmodified; `S-TLS-002`'s byte-equality assertion holds.
- **The scheduler learns nothing about delegation.** It installs an opaque value and revokes it; it never inspects what a tool publishes, never counts children, and holds no delegation state.
- **The addition is per-call and bounded.** The seam exists for exactly one `tool.Run` frame and is revoked on **every** exit path — normal return, detached return and re-panic — so no scheduler-owned state outlives the call (`R-DEL-003`).

(Previously: the requirement stated the opacity rule against a scheduler that installed nothing on the tool's context at all, so a reader meeting AG-19's `scheduler.go` diff had no recorded basis for deciding whether seam 3 had been overloaded.)

#### Scenarios

- **S-TLS-002** — AG-09.1 #2 policy slot passes through opaquely. Given a caller injecting `PolicySlot(sandboxBytes)` (e.g., a `[]byte` payload a Layer 3 sandbox would interpret), when the scripted tool's `Run` records the received `policy` value, then `bytes.Equal(recorded, injected)` is true byte-for-byte and the scheduler source contains zero type assertions on `PolicySlot`. *(AG-19 update: the assertion is unchanged and now also runs against a scheduler that installs a delegation seam on the same call's context — the two per-call values are independent, and this scenario proves the policy one is still forwarded byte-exact.)*
- **S-TLS-002a** — **(bite)** RED-first. Given a scheduler implementation that strips the type tag (`policy = PolicySlot(underlying)`), when the byte-equality assertion runs, then it FAILS for the right reason (tag stripped) — proves the property is non-vacuous. RED-recorded BEFORE `S-TLS-002` is GREEN.
- **S-TLS-020** — **AG-19: the seam rides beside `PolicySlot`, not inside it, and the guard proves it.** Given the merged AG-19 change, when `scheduler.go` is read as raw bytes, then it contains **no** type assertion of any kind added by this change and **no** occurrence of the seam's context key or interface type; when the `PolicySlot` source guard runs, then it passes with its own source byte-unchanged; and when a tool records both the `policy` value it received and the seam it obtained from its context, then the policy value is byte-identical to the injected one and the seam is a distinct value obtained through its own accessor. Cross-referenced to `R-DEL-001` / `S-DEL-003`.

### R-TLS-003 — `EffectClass` closed vocabulary: read / mutating / execute (D2)

The system SHALL define `EffectClass` as a typed enum (`type EffectClass uint8`) with exactly three members: `EffectClassRead`, `EffectClassMutating`, `EffectClassExecute`. The zero value MUST NOT be a member (mirroring `ToolOutcome` discipline, `tool_event.go:227-246`). `String()` MUST render each member distinctly.

#### Scenarios

- **S-TLS-003** — AG-09.1 #3 effect class vocabulary. Given the three `EffectClass` values, when `String()` is called on each, then the rendered strings are distinct and the zero value renders as `"unset"` (mirroring `ToolOutcome.String()` posture).

### R-TLS-004 — Concurrency policy: reads concurrent, mutating + execute serialized (D4, D5)

The system SHALL schedule calls as follows: a call with `EffectClassRead` runs concurrently with other reads (subject to fan-out bound); a call with `EffectClassMutating` or `EffectClassExecute` runs in a single-goroutine serialized channel, in issuance order among themselves. `Schedule` returns the typed `[]Result` slice in **call order** (rejoin), regardless of completion order.

#### Scenarios

- **S-TLS-004** — AG-09.2 #1 reads concurrent, mutations serialize. Given 3 read-class and 2 mutating-class calls, when the scheduler runs them and the scripted tools record their start times, then the 3 reads overlap (concurrent start), and the 2 mutatings start in strict issuance order with no overlap.

### R-TLS-005 — Bounded fan-out (D4)

The system SHALL bound the number of concurrently running read-class calls to a documented constant `MaxReadFanOut` (default 8). More read-class calls than the bound SHALL queue, never exceed.

#### Scenarios

- **S-TLS-005** — AG-09.2 #2 bounded fan-out. Given 12 read-class calls against `MaxReadFanOut = 8`, when the scheduler runs them and the scripted tools record start timestamps + a global concurrent-counter, then no snapshot exceeds 8 in-flight reads, and all 12 complete.
- **S-TLS-005a** — **(bite)** RED-first. Given a scheduler without a fan-out bound, when the concurrent-counter scenario runs, then it FAILS for the right reason (unbounded concurrency observed) — proves the bound is non-vacuous.

### R-TLS-006 — Start events emit at execution start, not at rejoin (D6)

The system SHALL emit `ToolStart` events (per AG-05.2 / R-AMT-005) for each scheduled call at the moment execution starts (before the call goroutine blocks on I/O), NOT at rejoin. A frontend observing `sink` MUST see `ToolStart` events in the order the calls began executing, not in completion order.

#### Scenarios

- **S-TLS-006** — AG-09.2 #3 start events at execution start. Given 3 read-class calls with staggered scripted delays (call 0 fast, call 1 slow, call 2 medium), when the scheduler runs them and a consumer drains `sink`, then the `ToolStart` events arrive in execution-start order (0, 2, 1) — NOT in call order or completion order.
- **S-TLS-006a** — **(bite)** RED-first. Given completions in inverted order (call 2 finishes first, call 0 last), when the consumer compares the `ToolStart` arrival order against completion order, then they differ — proves `ToolStart` is not emitted at rejoin. RED-recorded BEFORE `S-TLS-006` is GREEN.
- **S-TLS-006b** — **(bite)** RED-first. Given completions in strict issuance order, when `ToolStart` events are observed before any `ToolEnd*`, then they precede their corresponding end events by ≥ one observed timestamp tick — proves start-before-end. RED-recorded BEFORE `S-TLS-006` is GREEN.

### R-TLS-007 — `Result` typed outcomes distinct from execution error (AG-09.1 #3)

The system SHALL define `Result` as a value type with three typed outcomes mirroring AG-05.2's `ToolOutcome` (`Success`, `ResultFailure`, `ExecutionFailure`) — distinct by kind, not by convention over payload contents. A `Run` that returns a `Result{Outcome: Success}` or `Result{Outcome: ResultFailure}` MUST NOT return a non-nil `error`; a `Run` that returns a non-nil `error` MUST NOT also return a populated `Result`. The two channels are disjoint.

#### Scenarios

- **S-TLS-007** — AG-09.1 #3 result and execution failure are distinct typed outcomes. Given one scripted tool returning `(Result{Outcome: Success, Content: ...}, nil)` and one scripted tool returning `(Result{Outcome: ExecutionFailure, Failure: ...}, nil)`, when both outcomes are inspected at the contract level, then they are distinguishable by the typed `Result.Outcome` discriminant without inspecting payload contents, and the second outcome's `Failure` is a typed `*Failure` (R-AMT-006 carry).

### R-TLS-008 — Ordered rejoin (D5)

The system SHALL return `[]Result` whose element `i` is the result for call `i` (the call with ordinal `i`), regardless of the order in which calls completed. The slice MUST be fully populated; every ordinal slot MUST contain a non-zero `Result`.

#### Scenarios

- **S-TLS-008** — AG-09.3 #1 completions rejoin in call order. Given scripted tools completing in deliberately inverted order (call 0 last, call 1 first, call 2 middle), when the scheduler returns, then `results[0].CallID() == calls[0].callID`, `results[1].CallID() == calls[1].callID`, `results[2].CallID() == calls[2].callID` — positional match regardless of completion order.

### R-TLS-009 — Correlation identities survive the rejoin

The system SHALL preserve Layer 1 call/result correlation identities (`ai.ToolCall.id`) byte-exact through the rejoin. The `Result.CallID()` accessor MUST return the same `ai.ToolCallID` value the caller supplied, including synthetic IDs minted by an adapter.

#### Scenarios

- **S-TLS-009** — AG-09.3 #2 correlation identities preserved. Given 3 calls whose IDs include one synthetic ID minted by an adapter (`"adapter-mint-1"`) and two natural IDs from the provider, when the scheduler returns, then each `Result.CallID()` byte-equals the corresponding `ai.ToolCall.id` from the input.

### R-TLS-010 — One bad tool does not abort the turn (D6) — still true under cancellation

The system SHALL isolate execution failures: a call whose `Run` returns a non-nil `error` (or whose result is `Result{Outcome: ExecutionFailure}`) SHALL NOT abort the scheduler or affect sibling calls' outcomes. The failing call's ordinal slot SHALL contain a `Result` with `Outcome: ExecutionFailure` and a typed `*Failure`; siblings SHALL complete in their ordinal slots.

**The category is corrected here, because the name this requirement has carried since AG-09 does not exist.** `R-TLS-010` named `ai.FailureCategoryExecution`, and `ai.FailureCategory` has never had such a member: its vocabulary is `Authentication`, `Authorization`, `RateLimit`, `Unavailable`, `Timeout`, `Cancellation`, `MalformedResponse`, `UnsupportedCapability`, `Unknown` (`backend/agent/src/ai/provider_failure.go:58-96`, read directly, not cited). The behavior the requirement was describing has always been `ai.FailureCategoryUnavailable`, hardcoded in `typedFailureFromError` (`scheduler.go:1062-1066`) — the requirement described a category the code could not produce, and no test caught it because `S-TLS-016` and its siblings assert the failure's cause chain rather than its category. The corrected rule: an ordinary tool error carries **`ai.FailureCategoryUnavailable`**, and a scheduler-detected cancellation overrun carries **`ai.FailureCategoryCancellation`** (`R-TLS-013`, `R-TLS-014`). The category names the cause, and the isolation rule is the same for both.

A tool that returns early *because* it observed cancellation is **not** the second case: its error reaches the scheduler through the ordinary `runErr != nil` branch like any other tool error, so it carries `Unavailable`. The scheduler cannot distinguish "this tool errored because it saw cancellation" from "this tool errored for its own reasons" — only the bound overrun, which the scheduler itself detects, is typed `Cancellation`. AG-14 does not add a mechanism to tell them apart, and a scenario MUST NOT assert `Cancellation` on a tool's own self-reported error.

**The disjoint-return-channel rule survives AG-14 unchanged, and the reason is recorded rather than asserted.** `tool.Run` returns `(Result, error)`, and no site ever returns both channels populated: a context-observing tool returns a **non-nil error**, which routes through the pre-existing `runErr != nil` branch (`scheduler.go:469-473`) into `Result{Outcome: ExecutionFailure, Failure: non-nil}` with the tool's own `Result` discarded. Cancellation therefore adds **callers to an existing branch**; it adds no new return shape, no second populated channel and no new outcome member. The same is true of the bound: an overrunning call's slot is written by the scheduler through that same shape, not by the tool.

(Previously: the requirement named `ai.FailureCategoryExecution` — a member that does not exist in `ai.FailureCategory` — as the only category, and said nothing about cancellation-caused failures or about whether threading a real context into `tool.Run` disturbs the disjointness rule. The nonexistent name is a pre-existing AG-09 defect, not one AG-14 introduced; it is corrected here because AG-14 re-opens this requirement anyway, and leaving it would promote a name no reader can resolve.)

#### Scenarios

- **S-TLS-010** — AG-09.4 #1 one bad tool, siblings complete. Given 3 calls where call 1's scripted tool returns an execution error, when the scheduler runs, then `results[0]` is success, `results[1].Outcome == ExecutionFailure` with a typed `*Failure`, `results[2]` is success — the scheduler returns the full slice and no goroutine leaks (`runtime.NumGoroutine()` returns to baseline).
- **S-TLS-010a** — **(bite)** RED-first. Given a scheduler using `errgroup` whose first error cancels the group, when the failing-call scenario runs, then `results[2]` is the zero `Result` (sibling aborted) — proves the "siblings complete" property is non-vacuous. RED-recorded BEFORE `S-TLS-010` is GREEN.
- **S-TLS-019** — **AG-14: disjointness under cancellation.** Given a batch mixing a cancellation-observing tool, a cancellation-deaf tool and an ordinary succeeding tool, when the run is cancelled mid-batch, then every ordinal slot is populated, no slot carries both a tool-supplied `Result` and a scheduler-supplied `*Failure`, and the succeeding sibling's own result is unaffected. The category split MUST follow the requirement above rather than the intuition that "cancelled call ⇒ `Cancellation`": only the **deaf** tool's slot, whose failure the scheduler itself synthesized at the bound, reports `ai.FailureCategoryCancellation`; the **observing** tool returned its own error through the ordinary `runErr != nil` branch and therefore reports `ai.FailureCategoryUnavailable`, indistinguishable to the scheduler from any other tool error. A scenario asserting `Cancellation` on the observing tool's slot would contradict its own requirement and MUST NOT be written.

### R-TLS-011 — Panic containment under `-race` (D6)

The system SHALL contain a panicking tool's panic to that call's goroutine. The scheduler SHALL convert the panic into a typed `Result{Outcome: ExecutionFailure, Failure: NewFailure(...)}` in the panicking call's ordinal slot. After `Schedule` returns, `runtime.NumGoroutine()` SHALL return to its pre-`Schedule` baseline under `-race`.

#### Scenarios

- **S-TLS-011** — AG-09.4 #2 panicking tool contained. Given a scripted tool whose `Run` calls `panic("boom")`, when the scheduler runs under `go test -race`, then the panic does not propagate, `results[callOrdinal]` is `ExecutionFailure` with a typed `*Failure`, sibling results are populated, and `runtime.NumGoroutine()` returns to baseline.
- **S-TLS-011a** — **(bite)** RED-first. Given a scheduler without `defer/recover` in the call goroutine, when the panic scenario runs under `-race`, then `go test` reports the panic as an unhandled goroutine failure and the test process aborts — proves the recovery path is non-vacuous. RED-recorded BEFORE `S-TLS-011` is GREEN.

### R-TLS-012 — Sink ownership is a caller-selectable, zero-default contract

`Schedule` today closes the caller's `sink` after the rejoin, unconditionally. That is correct when `Schedule` is the last writer to the sink, and wrong when it is not — and on the continuation path of `R-LSK-001` it is not, because the schedule-before-finalize reorder puts the turn-close event **after** the rejoin. Closing there would make that emission a send on a closed channel.

The `Scheduler` MUST therefore carry a **sink-ownership flag whose zero value preserves AG-09 behavior exactly**: with the flag unset, `Schedule` closes the sink after the rejoin, as it always has. With the flag set, `Schedule` MUST leave the sink open after the rejoin and the **caller** becomes responsible for closing it exactly once. Every other step of `Schedule` — the parked-set clear, the emissions close, the dispatcher join, the ordered rejoin — MUST be unchanged in behavior and in order; only the close is conditional.

The flag MUST be a field on the `Scheduler` value rather than a new `Schedule` parameter, because a parameter would break the public `Schedule` signature that AG-09's and AG-10's suites are written against. Because every existing construction site builds the scheduler with a keyed struct literal, a new field is invisible to them and every AG-09/AG-10 scheduler test MUST pass with its source **byte-unchanged**.

The flag MUST NOT change how many times the sink is closed overall. Exactly one close MUST happen per sink, by exactly one owner. A caller that sets the flag and never closes, or closes twice, is a caller defect; the scheduler makes ownership selectable, it does not make it optional.

Setting the flag MUST NOT affect the dispatcher's need for a reader: the pre-existing AG-09 hazard that `sink <- &stamped` requires a live consumer is unchanged by this seam.

#### Scenarios

- **S-TLS-013** — Given a `Scheduler` constructed with the sink-ownership flag at its zero value and a set of scheduled calls, when `Schedule` runs and a consumer drains `sink`, then the ordered rejoin is fully populated, the consumer observes the sink close after the last tool event, and the emitted event sequence is byte-identical to the pre-AG-13 sequence for the same input.
- **S-TLS-014** — Given a `Scheduler` constructed with the sink-ownership flag set, when `Schedule` runs to its rejoin, then the sink is **not** closed, a subsequent send on it by the caller succeeds and is observed by the consumer, and the consumer observes the close only when the caller closes it; and the ordered rejoin is fully populated exactly as in `S-TLS-013`.
- **S-TLS-015** — Given every existing AG-09 and AG-10 scheduler test, when the suite runs against this change, then all of them pass with their source files **byte-unchanged**, because each constructs the scheduler with a keyed struct literal and therefore leaves the new field at its zero value.

### R-TLS-013 — `tool.Run` receives the run context

The scheduler MUST pass the context it was called with down into `tool.Run`. It MUST NOT substitute a background or otherwise detached context. No signature changes: `Schedule`, `executeCall` and `tool.Run` each already receive a `ctx`, and none may gain a parameter for this.

**The behavioral consequence is stated rather than discovered later:** a tool that reads its context now returns early when the run is cancelled, where it previously ran to completion. **That is the intent** — it is what makes AG-14.1's "in-flight tools observe cancellation" reachable at all, and what makes AG-14.3's cancellation-*deaf* tool a meaningful special fixture rather than a description of every tool.

A tool that ignores its context is not a protocol violation. It is the case `R-TLS-014` bounds.

#### Scenarios

- **S-TLS-016** — Given a scripted tool that returns a distinguishable typed error as soon as its context is done, and a run cancelled while that call is executing, then the call's ordinal slot carries an execution-failure result attributable to that tool's own early return, the tool's recorded work-completed flag is false, and its sibling calls still occupy their own ordinal slots — the context reached the tool and the scheduler did not abort the batch. *(Falsifiable independently of this requirement's own text: the tool records whether it ran to completion, an observation the scheduler does not produce.)*
- **S-TLS-017** — Given every existing AG-09 and AG-10 scheduler test and every direct `tool.Run` unit call in the package's test closure, when the suite runs against this change, then all of them pass with their source files **byte-unchanged** — none asserts the identity of the context `executeCall` supplies, so threading the real one is invisible to them.

### R-TLS-014 — Wind-down is bounded per call, and an overrunning call is detached and reported

The scheduler MUST carry a **wind-down bound** as a field on the `Scheduler` value whose **zero value resolves to a documented package default**. A caller that never sets it MUST get the default; a caller that sets it MUST get its own value. The field MUST NOT become a new `Schedule` parameter — a parameter would break the public signature AG-09's and AG-10's suites are written against — and it follows the sink-ownership precedent of `R-TLS-012`: because every existing construction site builds the scheduler with a keyed struct literal, a new field is invisible to them.

The bound MUST be **armed only by cancellation**. While the call's context is live, the scheduler MUST wait on the call's completion **alone** and MUST create no timer, so an uncancelled `Schedule` behaves exactly as it does today (`R-RUN-010`, `R-APP-009`). Once the context is done, the scheduler MUST wait at most the bound for that call's execution to return.

Detachment MUST be **per call**, not around the join. Abandoning `Schedule`'s wait for its call goroutines is forbidden, because a still-running goroutine would later write its ordinal slot — a data race under `-race` — and could send on an emissions channel `Schedule` has already closed (`scheduler.go:226`). Every writer of the rejoin slice and of the emissions channel MUST therefore remain a scheduler-owned goroutine that provably exits within the bound; the only frame that may outlive the bound is the third-party `tool.Run` frame itself, which MUST be able to deliver its eventual result and exit **without writing to any structure the scheduler has since closed**.

A call still running when the bound expires MUST be reported typed through the **existing execution-failure path**: its ordinal slot carries an execution-failure `Result` whose `*Failure` reports `ai.FailureCategoryCancellation` and whose cause is an `errors.As`-extractable value naming the **tool** and the **call identity** and stating the call is still running past the bound. **No new `EventKind` and no new `Result` outcome may be introduced** — the stream carries the existing `tool_end_execution_failure` kind.

`Schedule`'s remaining steps MUST be unchanged in behavior and in order: the parked-set clear, the emissions close, the dispatcher join and the ordered rejoin. Panic containment (`R-TLS-011`) MUST be preserved on both sides of the bound: a panic raised before detachment MUST still surface as that call's typed execution failure, and a panic raised by third-party code **after** detachment MUST NOT crash the process.

#### Scenarios

- **S-TLS-018** — Given a `Scheduler` constructed with the wind-down bound at its zero value and an uncancelled `Schedule` call whose tool blocks until an explicit release, when the release happens after an arbitrary delay, then `Schedule` returns with a fully populated rejoin, no bound-derived failure appears in any ordinal slot, and no timer was created — proving the bound is unarmed rather than merely generous.
- **S-CAN-006** *(owned by [`../agent-cancellation-tree/spec.md`](../agent-cancellation-tree/spec.md))* certifies the armed side: a cancellation-deaf tool does not prevent the run from ending, the overrun call is reported typed with the tool name and call identity, and no scheduler-owned goroutine survives the wind-down. It is referenced rather than duplicated so the two capabilities cannot drift into two different bounds.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-TLS-001** | External-package verifiability: every scenario verifiable by `cd backend/agent && make test`. Every behavioral test in `package agent_test` or another external package (AG-07 W6 carry; NFR-PRH-001 inheritance). |
| **NFR-TLS-002** | Determinism, race cleanliness, no ambient authority (AG-03.3 carry): every test MUST be deterministic, hermetic, pass under `-race`. Zero `net/http`, `os`, `os/exec` in **non-test sources** added by AG-09. |
| **NFR-TLS-003** | Substrate byte-unchanged (6th consecutive milestone, AG-08 NFR-PRH-003 carry): the 21 files in AG-07 `R-LSK-004` — `event.go`, `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `event_registry_test.go`, `doc.go`, `doc_contract_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, plus `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` — are byte-unchanged. Verified via `AG09_BASE_REF` env-var fallback (AG-07 W3 → AG-08 W3 → AG-09 pattern) + dynamic `git merge-base HEAD origin/main`. `TestTurn_SubstrateUntouched` widens its filter to also exclude `loop_tool_dispatch_test.go` (the AG-09 new test file). |
| **NFR-TLS-004** | Review budget: single PR under pre-authorised `size:exception` against 1000-line budget, forecast 2200–2500 lines (compare AG-08's 996). |

## Explicit non-requirements

- **No edit** to any substrate file in `NFR-TLS-003`. AG-09 is the 6th consecutive extensibility demonstration. *(Still true at AG-13: `tool.go` and `scheduler.go` are not members of that list — it names `tool_event.go`, a different file — so AG-13's edits to them need no release. Verified against `agent-loop-skeleton/spec.md:60`.)*
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit. Hand-rolled `chan struct{}` semaphore + serialized channel + `defer/recover` per call goroutine; `errgroup` is FORBIDDEN (new top-level dep + first-error cancellation conflicts with `R-TLS-010`). *(Still true at AG-13: the run driver adds no dependency.)*
- **Permission protocol around the scheduler** — AG-10. AG-09's scheduler accepts a single `policy PolicySlot` per call; AG-10 wraps that with decision-required / decision-made events.
- **Iteration of the model ↔ tools ↔ model cycle** — AG-13. AG-09 schedules one cycle; AG-13 iterates. **CLOSED by AG-13**: iteration ships in the run driver, which calls `Turn` repeatedly (`R-RUN-002`). The scheduler is unchanged in this respect — it still schedules exactly one cycle's calls per `Schedule` invocation, and the run driver never calls `Schedule` itself (`R-LSK-006`, as reconciled).
- **What any tool does** — Layer 3 built-in tools (doc 0004). AG-09 ships the contract; doc 0004 implements built-ins against it.
- **Sandbox semantics** — Layer 3 interprets `PolicySlot`. AG-09 forwards it byte-exact (R-TLS-002); AG-09 does not read it.
- **Subagent tool** — was: *"v1 non-goal per doc 0003 § 8."* **STILL A V1 NON-GOAL, and AG-19 is not its delivery — this row is REAFFIRMED rather than closed.** AG-19 installs a publishing seam in `executeCall` and proves the harness is re-entrant, but it ships no subagent tool, no subagent configuration and no depth limit; those stay post-v1 on the substrate AG-19 proved (`0003:1803`). The enforcement is structural: the seam names no subagent concept, its concrete type and installer are unexported so no code outside the package can mint one, and every subagent concept lives in `package agent_test`, which production code cannot import. **A reader must not take AG-19's `executeCall` diff as this row closing.** What the scheduler gained is a per-call opaque context value and its revocation — nothing that knows what a subagent is.
- **The four-hook taxonomy** (AG-20) — AG-09 ships `Tools` as a map; AG-20 widens to a `ToolSource` port if/when needed.
- **`ToolSource` port** (G6) — **AG-20** widening, not AG-09 and **not AG-13**. Re-homed by this change for two recorded reasons: the AG-13 charter (`0003:1294-1370`) never mentions tool sources anywhere in its goal, deliverable, acceptance clauses, or three leaves; and the immediately preceding line already assigns the four-hook taxonomy and the `ToolSource` widening to AG-20, so the two adjacent lines contradicted each other. AG-13 implements no part of it. **The identical claim also lives in code**, as a comment on the `Registry` interface at `backend/agent/src/agent/tool.go:239-240` — "`ToolSource` port (G6) is AG-13's widening" — and that comment MUST be re-homed to AG-20 in the **same pull request** as this delta. Spec and code drifting apart on the same sentence is the failure mode this repo has recorded before; changing one and not the other is not an acceptable outcome of this change.

## Dependencies

- **Depends on**: AG-05 (tool event family — `tool_event.go:1-501`, `ToolOutcome` enum at `tool_event.go:227-246`); AG-07 (walking skeleton — `loop.go:133-229`, merged at `93077c07` PR #167); AG-08 (`PreRequestHook` precedent — `loop.go:53-81`, `:175`, `:286-295`, merged at `e27e8411` PR #168).
- **Depends on**: AI-18 (streamed tool-call events at `tool_call_event.go:74-336`).
- **Depends on**: AI-21 (`agenttest.Provider.Requests()` at `fake_provider.go:157-161`); AI-22 (`agenttest.Script`, `Emit`, `Hold`, `NewIter`).
- **Closes**: G5 (R-13); v2 § 6 seams 2 & 3 (Layer 2 anchor: the execution call carries a policy parameter it does not interpret).
- **Blocks**: AG-10 (permission protocol wraps the scheduler); AG-11 (turn termination types the scheduler's outcomes); doc 0004 built-in tools implement this contract.

## Verification approach

- `cd backend/agent && make test` — full `-race -v ./...` run; all 12 spec scenarios + 6 bites green; AG-03 boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) stay green untouched.
- `cd backend/agent && make lint` — `golangci-lint` clean (cache clean if stale).
- `cd backend/agent && make build` — compile `./bin/database_administrator` clean.
- `cd backend/agent && make vuln-check` — no vulnerabilities.
- `cd backend/agent && make test/cover` — `loop.go` ≥ 80% line coverage (AG-04 W8 carry).
- Substrate-untouched check: NFR-TLS-003 verified via `AG09_BASE_REF` env-var fallback + dynamic merge-base.
- AG-08 W1 closed: at least one scheduler test uses unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline.
- AG-08 SUGG 1 closed: `drainSink` (`loop_test.go:147`) gains a `select` deadline once.

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active (`openspec/AGENTS.md`).

- **All four leaves are behavior**, so all four are RED-first.
- **`R-TLS-002` closes only on its recorded bite** (`S-TLS-002a`): the policy slot byte-equality bites RED before `S-TLS-002` is GREEN.
- **`R-TLS-005` closes only on its recorded bite** (`S-TLS-005a`): the unbounded-concurrency scenario bites RED before `S-TLS-005` is GREEN.
- **`R-TLS-006` closes only on its recorded bites** (`S-TLS-006a`, `S-TLS-006b`): the start-event-at-execution-start property bites RED twice before `S-TLS-006` is GREEN.
- **`R-TLS-010` closes only on its recorded bite** (`S-TLS-010a`): the "siblings complete" property bites RED with an `errgroup`-shaped mock before `S-TLS-010` is GREEN.
- **`R-TLS-011` closes only on its recorded bite** (`S-TLS-011a`): the panic containment property bites RED with a `panic("boom")` test before the `defer/recover` path is implemented.
- **No new event kinds** registered by AG-09 — the every-kind-constructible guard stays at 25 (AG-07 → AG-08 invariant).
- **Scope-fence remains at 25.** AG-09 extends via `NFR-TLS-003`'s 6th consecutive extensibility demonstration, not by editing the registry.

## Acceptance criteria

1. Every `S-TLS-001`…`S-TLS-011` has recorded evidence (plus `S-LSK-008` in the delta spec).
2. `cd backend/agent && make test`, `make lint` (after `cache clean`), `make build`, and `make vuln-check` are all green.
3. `backend/agent/go.mod` and `go.sum` byte-unchanged (NFR-TLS-004).
4. The 21 substrate files in `NFR-TLS-003` are byte-unchanged (6th consecutive milestone).
5. `loop.go` line coverage ≥ 80% (AG-04 W8).
6. The every-kind-constructible guard still passes at 25 kinds; AG-03's two boundary guards pass with zero changes.
7. The 11 charter `AG-09.1`/`AG-09.2`/`AG-09.3`/`AG-09.4` Gherkin scenarios are covered; none reduced.
8. All 6 bites RED-recorded with failing output in `apply-progress.md`.
9. `11 charter → 12 spec + 6 bites = 18 total` scenario count stated identically with the proposal, tasks, apply-progress, and verify-report (plus `S-LSK-008` cross-cut in the delta spec).
10. The `Turn` does not iterate (one cycle per turn, `S-LSK-008`) — the wording trap from `0003:107-112` is asserted mechanically in the delta spec.
11. AG-08 W1 carry: ≥ 1 scheduler test uses unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline.
12. AG-08 SUGG 1 closed: `drainSink` has a `select` deadline.

## Traceability

| Requirement | Charter node | Decisions cited | Charter scenario (`0003`) |
|---|---|---|---|
| `R-TLS-001` | AG-09.1 | D1 | `:928-931` (contract from outside) |
| `R-TLS-002` | AG-09.1 | D3 | `:933-936` (policy slot opaque) |
| `R-TLS-003` | AG-09.1 | D2 | `:928-931` (effect class vocabulary) |
| `R-TLS-007` | AG-09.1 | R-AMT-006 carry | `:938-941` (result and execution failure distinct) |
| `R-TLS-004` | AG-09.2 | D4, D5 | `:950-954` (concurrency policy) |
| `R-TLS-005` | AG-09.2 | D4 | `:956-959` (bounded fan-out) |
| `R-TLS-006` | AG-09.2 | D6 | `:961-964` (start events at execution start) |
| `R-TLS-008` | AG-09.3 | D5 | `:974-977` (ordered rejoin) |
| `R-TLS-009` | AG-09.3 | D5 | `:979-982` (correlation identities) |
| `R-TLS-010` | AG-09.4 | D6 | `:992-995` (one bad tool) |
| `R-TLS-011` | AG-09.4 | D6 | `:997-1000` (panic containment) |
| `S-LSK-008` (cross-cut, in delta spec) | AG-09 → AG-13 boundary | D8 (one cycle's tools; AG-13 iterates) | wording trap `0003:107-112` |

All 11 charter Gherkin scenarios are represented; none is reduced. Scenario count stated identically with the proposal (`11 charter → 12 spec + 6 bites = 18 total` in this spec, plus `S-LSK-008` cross-cut in the delta spec).