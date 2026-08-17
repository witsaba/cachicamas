# Delta for `agent-tool-scheduler` — AG-14 gives tools the run context and bounds the wind-down

> **Change**: `cachicamas-agent-cancellation-tree` · **AG-14** (Layer 2, Wave 3), `0003:1371-1442`
> **Modifies**: `agent-tool-scheduler` ([`../../../../specs/agent-tool-scheduler/spec.md`](../../../../specs/agent-tool-scheduler/spec.md)) — adds `R-TLS-013` and `R-TLS-014`; restates `R-TLS-010` (`spec.md:101-108`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES the restated block in the main spec with the MODIFIED block below; **full-block preservation is mandatory**.
> **Why this delta**: two of the design's three closed decisions land on scheduler surface. `executeCall` passes `context.Background()` to `tool.Run` (`scheduler.go:462`), so no executing tool can observe run cancellation and AG-14.1's first Then (`0003:1399`) is unreachable **by construction**. And `Schedule`'s `wg.Wait()` (`scheduler.go:205`) is unbounded, so a non-returning call goroutine blocks `Schedule`, therefore `Turn`, therefore `Run`. `R-TLS-010` is restated because threading a real context adds callers to its failure branch, and a reader must be able to see that its disjointness rule survived rather than infer it.
> **Ownership**: the signal vocabulary, the wind-down algorithm and the "detached and named" scope distinction are owned by `R-CAN-001`/`R-CAN-006` in [`../agent-cancellation-tree/spec.md`](../agent-cancellation-tree/spec.md). This delta owns only the scheduler's own obligations.

## ADDED Requirements

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

## MODIFIED Requirements

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

## Non-functional carry

`NFR-TLS-002` (determinism, race cleanliness, no ambient authority) holds unchanged. `time` is **not** a member of the ambient-authority guard's forbidden set (`ambient_authority_test.go:73-94`, which names `os`, process execution, `syscall` and legacy I/O), so the bound's timer is admissible in production source; the guard MUST pass with **zero** change over both closures. `NFR-TLS-003`'s substrate list is unaffected by this delta — the release AG-14 needs is recorded in [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md) under `R-LSK-004`.
