# Spec — The multi-turn run driver (`agent-run-driver`)

> **Change**: `cachicamas-agent-run-driver` · **AG-13** (Layer 2, Wave 3) of [doc 0003](../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-13--drive-the-multi-turn-run), `0003:1294-1370`
> **Nodes**: AG-13.1 `[leaf]` run to completion (`0003:1314-1336`) · AG-13.2 `[leaf]` steering input (`0003:1338-1355`) · AG-13.3 `[leaf]` pause resumption (`0003:1357-1369`)
> **Status**: **new capability**. This file is the normative text; per the AG-09 / AG-10 / AG-11 / AG-12 precedent it is promoted to `openspec/specs/agent-run-driver/spec.md` at archive. Five cross-cut deltas live beside it under `../agent-loop-skeleton/`, `../agent-permission-protocol/`, `../agent-history/`, `../agent-turn-termination/`, `../agent-tool-scheduler/`.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && make test` (`go test -race -v ./...`).
> **Identifier convention**: requirements `R-RUN-0NN`, scenarios `S-RUN-0NN` (bites carry the same `S-RUN-` prefix and are marked **(bite)**). Append-only. Distinct from `R-AEV-`/`R-AGE-`/`R-AGP-`/`R-AGM-`/`R-AGV-`/`R-ATT-`/`R-TLS-`/`R-APE-`/`R-PRH-`/`R-LSK-`/`R-AMT-`/`R-APP-`/`R-HIS-`; the `RUN` prefix was verified collision-free repo-wide before minting (`[RSN]-RUN-[0-9]` matches only this change's `proposal.md`).
> **Evidence gate**: `cd backend/agent && make test`, plus `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check` (`vuln-check` is **not** in `make all`). No CI exists.
> **Authoring constraint**: this spec states obligations and observable behavior. It names the Layer 2 identifiers that the SDD design document resolved as **binding** (`Harness`, `Run`, `Steer`, `TurnOptions.Continuation`, `Scheduler.LeaveSinkOpen`) because those resolutions are inputs to this spec, not open questions; the internal shape of each remains design's. Layer 1 identifiers already shipped (`ai.Message`, `ai.FinishReason`, `ai.Invalid`, `ai.ErrMisplaced`, `ai.At`) are cited as consumed contracts, never as Layer 2's own vocabulary.
> **Inherited decisions (CLOSED, not re-openable here)**: Decision 1 resolves to the nil-default `TurnOptions.Continuation` seam; Decision 2 resolves to injecting a caller-owned `*Scheduler`; the `R-LSK-006` reconciliation resolves in the charter's favour — **the harness never calls `Schedule`** (see the Reconciliation section below). No `L2C-08` doc-contract row lands.

## Coverage

| Charter leaf | Requirements | Spec scenarios | Bites |
|---|---|---|---|
| AG-13.1 (`0003:1314-1336`) | `R-RUN-001`…`007` | 13 | 1 |
| AG-13.2 (`0003:1338-1355`) | `R-RUN-008` | 4 | 0 |
| AG-13.3 (`0003:1357-1369`) | `R-RUN-009` | 2 | 0 |
| Fourth acceptance clause (`0003:1302`) | `R-RUN-010` | 1 | 1 |
| Cross-cut (failure path, seams) | `R-RUN-011`…`012` | 5 | 0 |

**This capability: 12 requirements → 26 scenarios, of which 2 are bites** (`S-RUN-061`, `S-RUN-091`). This count MUST be stated identically in `tasks.md`, `apply-progress.md` and `verify-report.md`.

> **Verify remediation (MAJOR-1)**: `S-RUN-101` was added under `R-RUN-011` after `sdd-verify` found the harness's `Run` left the steering queue open on every failure exit, so `Steer` returned nil forever after an `R-RUN-011` failure instead of `R-RUN-001`'s typed rejection. See `R-RUN-011`'s own text below.

**Charter Gherkin → spec** (all six charter scenarios mapped, none reduced):

| Charter scenario | Owning requirement | Scenario(s) |
|---|---|---|
| `0003:1319-1323` "a two-turn conversation runs to its terminal" | `R-RUN-002` (run), `R-RUN-005` (history) | `S-RUN-010`, `S-RUN-040` |
| `0003:1325-1328` "the event stream is the complete story" | `R-RUN-007` | `S-RUN-060` + bite `S-RUN-061` |
| `0003:1330-1333` "the harness holds no privileged channel into the loop" | `R-RUN-006` | `S-RUN-050` |
| `0003:1343-1346` "a mid-turn user message queues to the boundary" | `R-RUN-008` | `S-RUN-070` |
| `0003:1348-1352` "queued messages keep arrival order and are never dropped" | `R-RUN-008` | `S-RUN-071`, `S-RUN-072` |
| `0003:1362-1366` "pause resumes with verbatim replay" | `R-RUN-009` | `S-RUN-080`, `S-RUN-081` |

## Purpose

Twelve milestones built the parts of a conversation and none of them holds one. `Turn` runs exactly one `model → tools → finalize` cycle and stops (`R-LSK-006`); `History` stores a transcript and wires into nothing (`NFR-HIS-003`); AG-11 makes a pause visible and refuses to act on it (`agent-turn-termination/spec.md:146`); AG-10 parks a deferred call and deliberately exposes no wake handle (`agent-permission-protocol/spec.md:144`).

`agent-run-driver` is where those deferrals come due at once. It owns the **run**: appending the user-side messages, iterating `Turn` until a terminal finish reason, bracketing the whole iteration with exactly one run bracket over N turn brackets on one contiguous lane, queueing steering input at turn boundaries, resuming a paused turn verbatim, and keeping the run alive across a permission suspension that spans a turn.

It adds no new `EventKind`, no new `TurnOutcome`, no new exported `History` method, and no new Go dependency. What it adds is two nil/zero-default seams into the existing one-turn surface and a driver above it.

## Reconciliation — the harness drives through the public one-turn surface and never schedules

`R-LSK-006` (`agent-loop-skeleton/spec.md:83`) reads "the scheduler's `Schedule` function is the seam: callable from `Turn` (AG-09) or from `Harness` (AG-13)". AG-13.1's third charter scenario (`0003:1330-1333`) requires that "the harness holds no privileged channel into the loop … it goes through the same public one-turn surface the skeleton's external tests use". The two statements are in genuine tension and this spec resolves it rather than leaving it to implementation.

**The charter wins.** The `Harness` MUST NOT invoke `Schedule`. It owns the `*Scheduler` **value** — it constructs or receives it, injects it through the continuation seam, and calls `WakeParked` on it — and nothing else. `WakeParked` is not the `Schedule` seam and not a channel into the loop; it is the upward-path wake surface `agent-permission-protocol/spec.md:144` explicitly reserves for AG-13.

`R-LSK-006`'s operative clause — "`Turn` MUST call `Schedule` at most once per invocation" — remains **TRUE** under AG-13. The harness iterates by calling `Turn` repeatedly; each `Turn` invocation still schedules at most once. The losing phrase ("callable … from `Harness` (AG-13)") is re-scoped, not deleted, by the delta in [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md).

## Reconciliation — exactly one appender per message class

The run's transcript is written from two places and this spec keeps them apart, because a message written twice and a message written nowhere are the same defect wearing different clothes.

- **User-side messages** — the initial prompt and every steered message — are appended by the **harness**, at run entry and at turn boundaries.
- **The turn's assistant message and its `RoleTool` result messages** are appended by the **continuation-mode loop**, inside `Turn`, before it returns.

No message class has two appenders and none has zero. `CloseTurn` is the harness's call at each turn boundary, and it succeeds because the loop appended calls-then-results, leaving the open-call set empty (`R-HIS-003`).

## Requirements

### R-RUN-001 — The `Harness` is a value-form driver with a two-method surface

The run driver MUST be a value-form struct with exported configuration fields, **no constructor** and **no interface**, mirroring the `Scheduler` precedent and the AG-04..AG-12 rule that a concrete boundary type stays a struct until a second implementation actually arrives. Its public surface MUST be exactly two methods: `Run`, which drives one run to its terminal and returns the last turn's `(ai.Message, ai.FinishReason, error)`, and `Steer`, which offers a user message to the in-flight run.

Nil-valued optional fields MUST be resolved to defaults at `Run` entry into locals. The harness MUST NOT mutate the caller's fields, with exactly one recorded exception: it MAY set the sink-ownership flag of `R-RUN-012` once, on the scheduler it drives, before the first turn.

`Steer` MUST guarantee **zero drops**: a `Steer` returning nil means the message enters the transcript before a subsequent provider call. After the run's terminal decision has been taken, `Steer` MUST return a typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))` — never a silent drop and never a nil return.

One `Run` per harness value. Cross-run state is **AG-21**'s.

#### Scenarios

- **S-RUN-001** — Given a harness value constructed as a struct literal with only its required provider field set, when `Run` drives a scripted single-turn conversation from an external test package, then the run completes without any constructor call, the caller-visible field values are unchanged after `Run` returns except for the recorded sink-ownership flag, and the type exposes no third exported method beyond `Run` and `Steer`.
- **S-RUN-002** — Given a run that has taken its terminal decision and returned, when `Steer` is called with a well-formed user message, then it returns an error that satisfies the typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`, the transcript is unchanged, and no further event reaches the consumer sink.

### R-RUN-002 — The run algorithm and its finish-reason dispatch

`Run` MUST execute the following, in order: (1) resolve defaults and validate the prompt; (2) emit the run-open event on the consumer sink through the public run-event constructor and the run's shared lane stamper; (3) append the prompt to the transcript; (4) iterate — drain the steering queue FIFO and append each message, build the request transcript from the committed entries, call `Turn` once through the public one-turn surface with continuation options, forward every event the turn emits to the consumer sink unmodified, then close the turn; (5) dispatch on the returned finish reason; (6) terminate by emitting the run-close event and closing the consumer sink.

The dispatch MUST be total over `ai.FinishReason`'s vocabulary and MUST partition it exactly as follows:

| Finish reason | Dispatch |
|---|---|
| `ai.FinishReasonToolCalls` | **Iterate.** The loop already executed the calls and appended their results. |
| `ai.FinishReasonPauseTurn` | **Iterate.** The partial message is already appended; the next transcript replays it (`R-RUN-009`). |
| `ai.FinishReasonStop`, `Length`, `ContentFilter`, `Refusal`, `Unknown` | **Terminal candidate**, resolved by the atomic queue check below. |

A terminal candidate MUST be resolved **atomically under the steering queue's mutex**: if the queue is non-empty, the harness takes those messages and iterates — **a message queued during the final turn yields a new turn, never a dropped message**; if the queue is empty, the harness marks the queue closed in the same critical section and terminates. A check-then-close without the lock would drop a `Steer` that had already been accepted with a nil return, violating `R-RUN-001`'s zero-drop guarantee, and is therefore forbidden.

The harness MUST forward events; it MUST NOT rewrite, synthesize, suppress or reorder any event a turn emitted.

Forwarding MUST NOT depend on the turn returning: the harness MUST relay each turn's events to the consumer sink while the turn is still in flight, so that a `permission_decision_required` is observable to a consumer before the turn that emitted it returns. That observability is what makes `R-RUN-010` reachable.

#### Scenarios

- **S-RUN-010** — **Charter AG-13.1 sc.1.** Given a fake provider scripted with turn one requesting a tool call and turn two answering with final text, when one prompt drives the run and the consumer drains the sink to close, then the consumer observes, in order: run open, turn one open, turn one's message and tool events including the tool's execution and result, turn one close carrying the tool-calls outcome, turn two open, turn two's message events, turn two close carrying the finished outcome, run close carrying the completed outcome; and `Run` returns the second turn's message and finish reason with a nil error.
- **S-RUN-011** — Given a fake provider scripted so that a single turn ends in each terminal-candidate member of `ai.FinishReason`'s vocabulary in turn, when each run is driven with an empty steering queue, then each run ends after exactly one turn, the run-close event carries the completed outcome, and no candidate falls through to a second provider call.
- **S-RUN-012** — Given a run whose final turn is held at an `agenttest.Gate` and a user message offered through `Steer` while that turn is in flight, when the gate is released and the turn returns a terminal candidate, then `Steer` had returned nil, the queued message is appended, an **additional** turn bracket appears on the stream, and the run terminates only after that additional turn — proving the terminal decision and the queue close are one atomic step.

### R-RUN-003 — One run bracket, N turn brackets, one contiguous 1-based lane

The stream a consumer observes for one run MUST carry **exactly one** run-open/run-close pair, **N** turn-open/turn-close brackets nested inside it (one per `Turn` invocation), and **one contiguous 1-based sequence** across the entire run, owned by a **single shared `LaneStamper`** that the harness creates once and injects into every turn.

The harness MUST own the run bracket; `Turn` MUST keep ownership of the turn brackets. `Turn` cannot know that a turn is the last one — a terminal finish reason with a queued steering message continues the run — so conditional run-bracket emission inside `Turn` is forbidden by construction, not merely discouraged.

`CheckStream` MUST accept the recorded stream **unmodified**. `stream_check.go` MUST be byte-unchanged; this requirement is satisfied by the existing state machine, not by relaxing it.

#### Scenarios

- **S-RUN-020** — Given the complete event slice recorded from the two-turn run of `S-RUN-010`, when it is passed to `CheckStream` unmodified from an external test package, then it is accepted with no violation; and when the slice is walked, then it contains exactly one run-open and exactly one run-close, exactly two turn-open/turn-close brackets both nested between them, and a sequence that starts at 1 and increases by exactly 1 with no gap, no repeat and no restart across the whole run.

### R-RUN-004 — Run identity is harness-minted, provenance-distinct, and value-consistent

The harness MUST mint the run identity itself, with the provenance-distinct prefix `run-hrn-` and a package-local monotonic counter, rather than reusing the loop's `run-lsk-` minter. A reader of any single event MUST therefore be able to tell a harness-driven run from a bare one-turn `Turn` run by identity provenance alone.

Every event in one run — from the run-open through every turn's events to the run-close — MUST carry the **same** run identity value. This property MUST be asserted by this capability's own tests. It MUST NOT be delegated to `CheckStream`, which validates bracket structure and never compares identity values; strengthening `CheckStream` with cross-event identity consistency is **AG-19**'s (`stream_check.go` is a named `R-LSK-004` substrate file, and AG-19.1's parent/child separation is the first place the check is load-bearing). AG-13 MUST NOT exploit the gap: it upholds the property by construction and proves it by test.

#### Scenarios

- **S-RUN-030** — Given the complete event slice recorded from a multi-turn run, when every event's run identity is read, then all values are equal to one another and equal to the identity carried by the run-open event, and the assertion is made by this capability's own test rather than by `CheckStream`.
- **S-RUN-031** — Given a harness-driven run and a bare `Turn` invocation recorded side by side, when their run identities are read, then the harness-driven identity carries the `run-hrn-` prefix, the bare invocation's carries the loop's own prefix, and the two prefixes are different.

### R-RUN-005 — History wiring at the turn boundary, through existing routes only

The run's transcript MUST hold the full alternating conversation with **every tool call matched to its result**, written through the appender split of the reconciliation above. Specifically:

- the harness MUST append the prompt before the first turn and each steered message at a turn boundary before the next request transcript is built;
- the continuation-mode loop MUST append the turn's assistant message (skipping it when the turn produced no content) and then one tool-result message per rejoin result, in call order, before returning — that half is owned by `R-HIS-010` in [`../agent-history/spec.md`](../agent-history/spec.md), not restated as an obligation here;
- the harness MUST close the turn at each boundary, and that close MUST succeed because the open-call set is empty.

The capability MUST add **no new exported `History` method**. `history_surface_guard_test.go` is a closed-route guard and MUST stay green **file-unchanged**; the wiring uses `Append`, `CloseTurn` and `Entries` only.

Consecutive same-role entries MUST be accepted — the neutral transcript carries no role-alternation rule (`0003:1346`), and no History change is needed to permit them.

#### Scenarios

- **S-RUN-040** — **Charter AG-13.1 sc.1, second Then.** Given the completed two-turn run of `S-RUN-010`, when the transcript is read back through the public read route, then it holds, in order, the prompt, turn one's assistant message carrying the tool call, the tool result message for that call, and turn two's assistant message; every tool call has a matching result; and the read-back sequence contains no duplicated and no missing message.
- **S-RUN-041** — Given the closed-route history surface guard and the import/ambient-authority guards, when the full suite runs against this change, then all three pass with their source files byte-unchanged, and the enumerated exported History route set is unchanged in both directions.

### R-RUN-006 — The harness holds no privileged channel into the loop

**Charter AG-13.1 sc.3.** The harness MUST reach the loop through the same public one-turn surface the skeleton's external tests use. It MUST NOT reference any loop internal — the turn accumulator, the loop's identity minters, its stamped-emission helper, its sink-close helper, its request builder, or its finish-reason mapper — and it MUST NOT contain a `Schedule` call site (the reconciliation above).

Every test for this capability MUST live in an external test package (`package agent_test`) and MUST observe only the event stream and the public transcript surface. An in-package test could reach the private surface directly and would prove nothing about the boundary.

This is `L2C-03` ("callers observe the stream, they never reach into the loop") at run scope, not a new package-wide guarantee; **no `L2C-08` doc-contract row lands**, and `doc.go` and `doc_contract_guard_test.go` stay byte-unchanged.

#### Scenarios

- **S-RUN-050** — Given the run driver's production source read as raw bytes by a source-scan guard (the `scheduler_test.go` regex precedent), when it is scanned, then it contains no reference to any enumerated loop internal and no `Schedule` call site; and given the capability's behavioral tests, when their package clause is read, then every one of them is in an external test package and reaches the driver only through its two public methods, the event stream, and the public transcript reads.

### R-RUN-007 — Run-scope reconstruction: the stream is the complete story

**Charter AG-13.1 sc.2.** Every message and every tool outcome the transcript holds MUST be reconstructable from the replayed event stream, **asserted at run scope** — over the whole multi-turn stream partitioned by turn bracket — not per turn. This extends `L2C-04`/`L2C-05` to the multi-turn story; it declares no new guarantee.

The assertion MUST be non-vacuous, and non-vacuity MUST be discharged by a **bite**, not by prose. The bite MUST drop an event belonging to **turn two** and MUST prove that the comparator reports divergence — mirroring `S-LSK-003a`/`S-LSK-003b`. A run-scope assertion that still passes with a turn-two event removed is the AG-05 W1 vacuous-helper failure mode and MUST fail this requirement.

#### Scenarios

- **S-RUN-060** — Given a completed multi-turn run and the complete slice of events it emitted, when the events are partitioned by turn bracket and each turn's messages and tool outcomes are reconstructed from them, then the reconstructed run-scope result is deep-equal to the transcript read back through the public route, message for message and outcome for outcome.
- **S-RUN-061** — **(bite)** Given a copy of that event slice with exactly one **turn-two** event removed, when the run-scope reconstruction of `S-RUN-060` runs against it, then the comparator REPORTS divergence and the scenario FAILS — proving the run-scope assertion is non-vacuous rather than passing on turn one alone. RED-recorded BEFORE `S-RUN-060` is GREEN.

### R-RUN-008 — Steering: queued at the boundary, arrival-ordered, never dropped

**Charter AG-13.2, both scenarios.** A user message offered while a turn is in flight MUST queue. The in-flight turn MUST complete **untouched** — its event sequence MUST be identical to the sequence the same turn produces with no steering at all. The queued message MUST enter the transcript at the turn boundary **before the next provider call**, and that ordering MUST be proven by evidence from a request-recording provider wrapper, not by inference from transcript order alone.

Multiple queued messages MUST enter in **arrival order** with **zero drops**, including messages offered concurrently from another goroutine. A message queued during the **final** turn MUST yield a new turn rather than a dropped message (`R-RUN-002`'s atomic terminal decision).

Consecutive same-role entries MUST be accepted.

Synchronization in every scenario below MUST be by `agenttest.Gate` and channel reads. A test that needs a wall-clock sleep or a timeout to pass is a design failure, not a flaky test.

#### Scenarios

- **S-RUN-070** — **Charter AG-13.2 sc.1.** Given a run whose turn-one script holds at an `agenttest.Gate`, when a user message is offered through `Steer` on reaching the gate and the gate is then released, then the in-flight turn's emitted event sequence is unchanged from the un-steered baseline, the steered message appears in the transcript between turn one's messages and turn two's, and the request the provider recorded for turn two contains that message — "before the next provider call" proven by the recorded request.
- **S-RUN-071** — **Charter AG-13.2 sc.2, first Then.** Given a gate-held turn and N user messages offered through `Steer` from a second goroutine in a test-determined order, when the gate is released and the run continues, then all N messages appear in the transcript in that same arrival order, none is missing, and none is duplicated.
- **S-RUN-072** — **Charter AG-13.2 sc.2, second Then.** Given a gate-held **final** turn whose script would otherwise end the run, and a script available for one further turn, when a message is offered through `Steer` before the gate is released, then `Steer` returns nil, an additional turn bracket appears on the stream, the steered message is in the transcript, and the run terminates after that additional turn.
- **S-RUN-073** — Given a run that has terminated, when `Steer` is called, then it returns the typed rejection of `R-RUN-001` and the transcript is unchanged — the queue is closed, not merely empty.

### R-RUN-009 — Pause resumption: the partial is replayed verbatim, and the pause stays visible

**Charter AG-13.3.** A turn returning `ai.FinishReasonPauseTurn` MUST resume. The `ai.Message` that turn returned — the partial assistant content reconstructed from what was received — MUST be appended to the transcript and re-included in the next request's transcript **byte-verbatim**, including any opaque reasoning round-trip token. It MUST NOT be discarded, re-synthesized, re-serialized, truncated, or merged into the resumed turn's message.

The pause MUST remain visible on the stream as its own turn outcome (`TurnOutcomePaused`) rather than being silently absorbed. The harness MUST forward that turn-close event unchanged; it MUST NOT rewrite the outcome and MUST NOT suppress the bracket. No new `TurnOutcome` and no new `EventKind` is introduced — AG-11's existing mapping already covers it.

The run MUST continue past the pause to a **real terminal** finish reason.

#### Scenarios

- **S-RUN-080** — **Charter AG-13.3, first Then.** Given a provider scripted with turn one delivering partial text and reasoning carrying a non-empty round-trip token and completing with `ai.FinishReasonPauseTurn`, and turn two delivering final text with a terminal finish reason, when the run is driven, then the transcript entry for the paused turn is deep-equal to the message `Turn` returned including the round-trip token bytes, the request recorded for turn two contains that entry byte-verbatim, and the run ends with the completed run outcome.
- **S-RUN-081** — **Charter AG-13.3, second Then.** Given the same run, when the event stream is read, then turn one's turn-close event carries the paused turn outcome, that outcome value is different from both the finished and the aborted outcome values, and no assertion in the scenario reads an `ai.FinishReason` to establish it.

### R-RUN-010 — The run survives a permission suspension spanning a turn

**Fourth acceptance clause (`0003:1302`).** A `PermissionDefer` verdict issued inside a harness-driven turn parks the call. The harness MUST hold a live handle on the `*Scheduler` that turn is blocked inside — obtained by injecting a caller-owned scheduler through the continuation seam — and MUST be able to release the parked call with `WakeParked` **while that turn is still in flight**. The run MUST then complete.

The resolution contract is closed and MUST be stated where a reader cannot miss it: a parked set lives exactly one `Schedule` call, so a suspension inside a harness-driven turn resolves by exactly one of (a) an external `WakeParked` on the injected scheduler, or (b) cancellation of the run context, which the harness propagates unmodified into every turn. The harness adds **no third path and no timeout**. A run whose policy defers and whose owner neither wakes nor cancels does not terminate — **by design**; `R-APP-009` is the safety net and richer cancellation vocabulary is **AG-14**'s.

Synchronization MUST be the stream itself: because registration precedes emission and the emission's acknowledgement precedes the parked wait (`R-APP-002`), a consumer that has read the decision-required event off the run stream may wake with a guaranteed-live entry. No wall clock, no timeout, no sleep.

This requirement also claims the **known gap `agent-permission-protocol/spec.md:172` carried to AG-13**, quoted verbatim: "**Known gap (carried to AG-13)**: the `R-APP-002` acknowledgement itself currently has no non-vacuous guard — deleting the acknowledgement leaves the package green. The behaviour is present and correct in production; the missing bite must observe the parked **wait**, not the registration." The bite below MUST observe the **wait**. A bite that observes registration re-encodes the very gap it closes and MUST be rejected.

#### Scenarios

- **S-RUN-090** — Given a permission policy that defers the first resolution and allows the second, a caller-owned scheduler injected into the run, and a test reading the consumer sink event by event, when the decision-required event is read, then the run has not returned (checked by a non-blocking read of the run's completion channel) and the tool has not been invoked; and when `WakeParked` is called for that call identity, then it returns nil, the call executes, the run completes with the completed run outcome, the suspension's events lie **inside turn one's bracket**, and `CheckStream` accepts the whole stream unmodified.
- **S-RUN-091** — **(bite)** Given a scratch edit that replaces the parked **wait** with an immediate re-resolution, when a scenario whose policy's second resolution asserts a flag the test sets immediately before `WakeParked` runs, then it FAILS because the flag is unset at re-resolution — proving the assertion observes the parked wait rather than the registration. RED-recorded over repeated runs per `NFR-APP-002`'s `-count=15` discipline, then reverted.

### R-RUN-011 — A failed turn ends the run typed, with no append, no close, and no retry

When a `Turn` invocation returns a non-nil error, the harness MUST emit a run-close event carrying the failed run outcome and a non-nil typed failure, built through the public constructors, then close the consumer sink and return that error.

On that path the harness MUST NOT append anything to the transcript, MUST NOT call `CloseTurn`, and MUST NOT retry, back off, or route to a fallback provider. Retry and failover are **AG-15**'s, and this requirement is what makes that separation checkable rather than asserted.

This failure path MUST also close the run's steering queue as part of the same termination, under the queue's own mutex — the same critical section `R-RUN-002`'s atomic terminal decision uses. Every `Run` exit — this failure path, a rejected prompt or steered-message append that reaches it, and the terminal-decision success path — MUST leave the queue closed before `Run` returns, so a `Steer` call that reaches the harness afterward always observes it closed and receives `R-RUN-001`'s typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`. A `Run` exit that leaves the queue open makes every later `Steer` return nil forever — indistinguishable, from the caller's side, from the silent drop `R-RUN-001` forbids.

#### Scenarios

- **S-RUN-100** — Given a provider scripted with a terminal mid-stream failure on turn one and a further script available so that a retry would be observable, when the run is driven, then the consumer observes the turn's typed closing brackets followed by a run-close carrying the failed run outcome with a non-nil failure and then the sink close; `Run` returns a non-nil error; the transcript holds only what was committed before the failure, with no entry appended after it; and the provider recorded exactly one request.
- **S-RUN-101** — Given a run that has ended through this requirement's failure path, when `Steer` is called after `Run` has returned, then it returns `R-RUN-001`'s typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`, never nil — the queue closes on every `Run` exit, not only the terminal-decision success path.

### R-RUN-012 — The seams are nil/zero-default and the substrate holds

The two seams this capability needs MUST be additive and default-inert:

- the one-turn surface's continuation seam MUST be **nil-default**: with it nil, every path through `Turn` is byte-stable pre-AG-13 behavior;
- the scheduler's sink-ownership flag MUST be **zero-default**: with it false, `Schedule` behaves exactly as it did at AG-09. The harness sets it to true on the scheduler it drives so that the continuation path can emit the turn-close event after the rejoin.

Every file named by `R-LSK-004` — including `stream_check.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `turn_events.go`, `doc.go`, `doc_contract_guard_test.go`, `event_registry_test.go`, `reconstruction_test.go` — MUST be byte-unchanged, as MUST `history.go` and every file under `backend/agent/src/ai/`. `go.mod` and `go.sum` MUST be unchanged. No new `EventKind` is registered.

Both substrate filters — `filterOutLoopFiles` (`loop_test.go:831`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907`) — MUST be widened by **exact filename suffix only** for each file this change introduces: no wildcard, no prefix match, no directory-level relaxation, and the two filters MUST carry an identical entry set.

#### Scenarios

- **S-RUN-110** — Given the merge base of the AG-13 branch with `origin/main`, when `git diff` is taken over `backend/agent/src/agent/` and over `go.mod`/`go.sum`, then only allowlisted files differ, every file named by `R-LSK-004` is byte-unchanged, `history.go` is byte-unchanged, the `go.mod`/`go.sum` diff is empty, the every-kind-constructible guard still passes at its committed kind count (AG-13 adds zero), and both substrate guards pass.
- **S-RUN-111** — Given the two substrate filters, when their entry sets are compared, then they are identical to each other, every entry is an exact filename suffix with no wildcard, prefix or directory pattern, and each file this change introduces is named in both.
- **S-RUN-112** — Given every existing test that asserts unconditional run brackets or a per-call sequence restart, and every AG-09/AG-10 scheduler test, when the suite runs against this change, then all of them pass with their source files **byte-unchanged**, because each passes a zero-value options value and therefore takes the nil continuation path.

## Non-functional requirements

### NFR-RUN-001 — External-package verifiability

Every scenario above MUST be verifiable by `cd backend/agent && make test`. Every behavioral test MUST live in an external test package (`package agent_test`); a behavior reachable only from inside the package is, for the purposes of this spec, not reachable at all. `R-RUN-006` in particular is meaningless from inside the package.

### NFR-RUN-002 — Determinism, race cleanliness, and no wall clock

Every test added by this change MUST be deterministic and hermetic — no network, no filesystem, no environment dependence — and MUST pass under `-race`. Synchronization MUST be by `agenttest.Gate`, channel reads and channel closes. **No test may synchronize by sleep, timeout, or wall-clock ordering**, and no production source added by this change may read a clock or perform I/O (the ambient-authority guard).

### NFR-RUN-003 — Single-writer discipline on the shared lane

The harness MUST touch the run's shared lane stamper only **between** turns; during a turn only the loop and the scheduler's dispatcher touch it. The phases are strictly sequential and MUST be race-free under `-race`.

### NFR-RUN-004 — Coverage

Line coverage on `backend/agent/src/agent/loop.go` MUST remain ≥ 80% under `make test`, including the new continuation branches (`R-LSK-005` carry).

### NFR-RUN-005 — Review budget

`openspec/config.yaml` forecasts a 400-line review budget. This change ships as a **single** pull request under a **pre-authorised `size:exception`**, forecast at 1630–2660 changed lines including SDD markdown. The pull request description MUST state why the change does not fit the default budget.

## Explicit non-requirements — what this spec does NOT claim

Stated so that no test, guard or acceptance line is written as if AG-13 closes more than it does. Each row names its owning milestone.

| Not claimed | Owner |
|---|---|
| Cross-event run-identity consistency enforced by `CheckStream` | **AG-19.** AG-13 upholds the property by construction and asserts it in its own tests (`R-RUN-004`); the validator is not strengthened and `stream_check.go` stays byte-unchanged |
| Retry or failover on a failed turn | **AG-15.** Charter "Out of scope" line, binding (`R-RUN-011`) |
| A compaction check between turns | **AG-17** inserts it at AG-13's turn boundary; **AG-18** implements compaction. Nothing here anticipates its shape |
| Cancellation semantics — interrupt vs. shutdown vs. deadline | **AG-14.** AG-13 propagates the context unmodified and stops; it defines no cancellation vocabulary and adds no timeout |
| Cost aggregation across turns | **AG-16** |
| A production subagent tool, nested or child runs | **AG-19.** No subagent tool ships in v1 (`0003:1794`) |
| Multi-turn state beyond a single run | **AG-21** (`agent-loop-skeleton/spec.md:106`). AG-13 owns one run's iteration |
| The four-hook taxonomy and a `ToolSource` port widening | **AG-20.** The AG-13 charter never mentions tool sources; `agent-tool-scheduler/spec.md:138` is re-homed by this change's delta |
| Persistence or session reload of a run | **Layer 3.** The harness holds state in memory and never touches a file (`0003:110`) |
| A real provider or a real tool | **Never in Layer 2.** `agenttest` scripts only (`0003:123`) |
| Any edit under `backend/agent/src/ai/**` | **Not this milestone.** Layer 1 is consumed, never edited |
| A new `EventKind`, a new `TurnOutcome`, or a new exported `History` method | **Not this milestone**, and forbidden under this change |
| An `L2C-08` doc-contract row | **Not this milestone.** AG-13 declares no new package-wide guarantee; it rides `L2C-03`/`L2C-04`/`L2C-05`/`L2C-07` at run scope. AG-19's child runs re-open the question |

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active.

- All three charter leaves are behavior, so all three are RED-first.
- **`R-RUN-007` closes only on its bite** (`S-RUN-061`): the run-scope reconstruction must RED with a dropped **turn-two** event BEFORE `S-RUN-060` is GREEN. This is the AG-05 W1 vacuous-helper failure mode restated at run scope.
- **`R-RUN-010` closes only on its bite** (`S-RUN-091`), which must observe the parked **wait**. Note the staleness finding recorded in [`../agent-permission-protocol/spec.md`](../agent-permission-protocol/spec.md): the AG-10 remediation test `TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery` may already guard the acknowledgement at `Schedule` level, so `:172`'s "leaves the package green" claim may be stale. That MUST be settled by **running the scratch at apply time**, not assumed in either direction.
- **`R-RUN-002`'s atomic terminal decision** is proven by `S-RUN-012`, not by prose.
- **No new event kinds** — the every-kind-constructible guard stays at its committed count.

## Acceptance criteria

1. Every `S-RUN-001`…`S-RUN-112` has recorded evidence; both bites (`S-RUN-061`, `S-RUN-091`) are RED-recorded with failing output before their GREEN.
2. All **six** charter Gherkin scenarios (`0003:1319-1323`, `:1325-1328`, `:1330-1333`, `:1343-1346`, `:1348-1352`, `:1362-1366`) are mapped in the table above and closed; none is reduced.
3. `cd backend/agent && make test` green under `-race`; `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check` all clean.
4. `CheckStream` accepts the multi-turn run stream **unmodified**, with `stream_check.go` byte-unchanged.
5. Every existing test asserting unconditional run brackets or per-call sequence restart, and every AG-09/AG-10 scheduler test, passes **file-unchanged**.
6. Both substrate filters are byte-in-sync and widened by exact filename suffix only.
7. All five cross-cut deltas are written and every line they cite is re-read against the shipped change by `sdd-verify`.
8. `docs/architecture/milestones/0003-…md:2170` is ticked and the doc-0003 milestone counters are bumped.

## Traceability

| Requirement | Charter node | Primary evidence |
|---|---|---|
| `R-RUN-001` | AG-13.1 / AG-13.2 | value-form surface and typed steering rejection |
| `R-RUN-002` | AG-13.1 sc.1 | two-turn run to terminal; atomic terminal decision |
| `R-RUN-003` | AG-13.1 sc.1 | one run bracket, N turn brackets, one lane |
| `R-RUN-004` | AG-13.1 cross-cut | run-identity value consistency and provenance |
| `R-RUN-005` | AG-13.1 sc.1 | alternating transcript, every pair matched |
| `R-RUN-006` | AG-13.1 sc.3 | source-scan guard + external-package tests |
| `R-RUN-007` | AG-13.1 sc.2 | run-scope reconstruction + turn-two bite |
| `R-RUN-008` | AG-13.2 sc.1, sc.2 | boundary entry, arrival order, zero drops |
| `R-RUN-009` | AG-13.3 | verbatim replay, pause visible |
| `R-RUN-010` | 4th acceptance clause | suspension survives the turn; parked-wait bite |
| `R-RUN-011` | cross-cut | failed-run close, no retry |
| `R-RUN-012` | cross-cut | nil/zero-default seams, substrate held |
