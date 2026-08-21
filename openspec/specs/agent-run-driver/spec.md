# Spec — The multi-turn run driver (`agent-run-driver`)

> **Change**: `cachicamas-agent-run-driver` · **AG-13** (Layer 2, Wave 3) of [doc 0003](../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-13--drive-the-multi-turn-run), `0003:1294-1370`
> **Nodes**: AG-13.1 `[leaf]` run to completion (`0003:1314-1336`) · AG-13.2 `[leaf]` steering input (`0003:1338-1355`) · AG-13.3 `[leaf]` pause resumption (`0003:1357-1369`)
> **Status**: **new capability**. This file is the normative text; per the AG-09 / AG-10 / AG-11 / AG-12 precedent it is promoted to `openspec/specs/agent-run-driver/spec.md` at archive. Five cross-cut deltas live beside it under `../agent-loop-skeleton/`, `../agent-permission-protocol/`, `../agent-history/`, `../agent-turn-termination/`, `../agent-tool-scheduler/`.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && make test` (`go test -race -v ./...`).
> **Identifier convention**: requirements `R-RUN-0NN`, scenarios `S-RUN-0NN` (bites carry the same `S-RUN-` prefix and are marked **(bite)**). Append-only. Distinct from `R-AEV-`/`R-AGE-`/`R-AGP-`/`R-AGM-`/`R-AGV-`/`R-ATT-`/`R-TLS-`/`R-APE-`/`R-PRH-`/`R-LSK-`/`R-AMT-`/`R-APP-`/`R-HIS-`; the `RUN` prefix was verified collision-free repo-wide before minting (`[RSN]-RUN-[0-9]` matches only this change's `proposal.md`).
> **Evidence gate**: `cd backend/agent && make test`, plus `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check` (`vuln-check` is **not** in `make all`). No CI exists.
> **Authoring constraint**: this spec states obligations and observable behavior. It names the Layer 2 identifiers that the SDD design document resolved as **binding** (`Harness`, `Run`, `Steer`, `TurnOptions.Continuation`, `Scheduler.LeaveSinkOpen`) because those resolutions are inputs to this spec, not open questions; the internal shape of each remains design's. Layer 1 identifiers already shipped (`ai.Message`, `ai.FinishReason`, `ai.Invalid`, `ai.ErrMisplaced`, `ai.At`) are cited as consumed contracts, never as Layer 2's own vocabulary.
> **Inherited decisions (CLOSED, not re-openable here)**: Decision 1 resolves to the nil-default `TurnOptions.Continuation` seam; Decision 2 resolves to injecting a caller-owned `*Scheduler`; the `R-LSK-006` reconciliation resolves in the charter's favour — **the harness never calls `Schedule`** (see the Reconciliation section below). No `L2C-08` doc-contract row lands.
>
> **Amended 2026-08-20 (AG-20)**: `R-RUN-013` ADDED, alongside `R-RUN-001`, `R-RUN-003`, `R-RUN-004` and `R-RUN-010`, all four of which are byte-unchanged. AG-20 adds one exported configuration field (`Hooks`) and no exported method, one unexported one-way session-start latch of the shipped shutdown flag's class, two pure reads beside the per-attempt forwarder plus a run-frame-local cost accumulator, and one per-run observer lane created only when at least one observing hook is registered — with **NO** third path, **NO** timeout and **NO** harness method. See delta spec at `openspec/changes/cachicamas-agent-hook-taxonomy/specs/agent-run-driver/spec.md`.
>
> **Amended 2026-08-20 (AG-21 archive)**: `R-RUN-001` MODIFIED (back-annotation only, no new identifier minted). The clause "Cross-run transcript state remains AG-21's" is discharged by an enumerated inventory with an absence assertion, and "Concurrent runs on one value stay out of scope" is restated as still true. See delta spec at `openspec/changes/cachicamas-agent-concurrency-hardening/specs/agent-run-driver/spec.md`.

## Coverage

| Charter leaf | Requirements | Spec scenarios | Bites |
|---|---|---|---|
| AG-13.1 (`0003:1314-1336`) | `R-RUN-001`…`007` | 13 | 1 |
| AG-13.2 (`0003:1338-1355`) | `R-RUN-008` | 4 | 0 |
| AG-13.3 (`0003:1357-1369`) | `R-RUN-009` | 2 | 0 |
| Fourth acceptance clause (`0003:1302`) | `R-RUN-010` | 1 | 1 |
| Cross-cut (failure path, seams) | `R-RUN-011`…`012` | 5 | 0 |

**This capability: requirements allocated `R-RUN-001` through `R-RUN-013`; scenarios allocated `S-RUN-001` through `S-RUN-113`, of which the bites are `S-RUN-061` and `S-RUN-091`.** Each milestone that appends records its own additions in its delta; this line states the allocated range and never a total, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`).

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

### R-RUN-001 — The `Harness` is a value-form driver with a named four-method surface

The run driver MUST be a value-form struct with exported configuration fields, **no constructor** and **no interface**, mirroring the `Scheduler` precedent and the AG-04..AG-12 rule that a concrete boundary type stays a struct until a second implementation actually arrives.

Its public surface MUST be exactly the following methods, **enumerated by name rather than by count** so the pin stays meaningful as the type evolves:

| Method | Contract |
|---|---|
| `Run` | drives one run to its terminal and returns the last turn's `(ai.Message, ai.FinishReason, error)` |
| `Steer` | offers a user message to the in-flight run |
| `Interrupt` | fires the interrupt signal (`R-CAN-002`) |
| `Shutdown` | fires the shutdown signal and latches the terminal refusal (`R-CAN-005`) |

A method not named in that table MUST be observable as a guard failure, never as an unaudited addition. `Interrupt` and `Shutdown` sit beside `Steer` on the **same non-privileged upward path** (`R-RUN-006`): they reach the loop through no privileged channel and hold no handle into it — they flip a context the loop already observes.

**A consequence stated so `sdd-apply` is not read as a silent test rewrite:** the reflection subtest `"exactly two exported methods"` (`harness_test.go:1018-1024`, whose `want` is `[]string{"Run", "Steer"}` at `:1027`) changes as a direct consequence of this requirement. That edit is **conscious and delta-backed**, not incidental; its new expectation MUST be the four names above, sorted, and its subtest name MUST stop asserting a count.

Nil-valued optional fields MUST be resolved to defaults at `Run` entry into locals. The harness MUST NOT mutate the caller's fields, with exactly one recorded exception: it MAY set the sink-ownership flag of `R-RUN-012` once, on the scheduler it drives, before the first turn. AG-14 adds **no second caller-field mutation**: the wind-down bound is a caller-owned zero-default field on the `Scheduler` the caller already injects, which the harness reads and never writes.

`Steer` MUST guarantee **zero drops**: a `Steer` returning nil means the message enters the transcript before a subsequent provider call. After the run's terminal decision has been taken, `Steer` MUST return a typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))` — never a silent drop and never a nil return.

**One run at a time per harness value** — re-scoped from "one `Run` per harness value", because AG-14.1's charter requires the reuse the old wording forbade (`0003:1400`: "a new prompt on the same harness works afterward"). A run that has ended, whether completed, failed or interrupted, MAY be followed by another `Run` on the **same value**, and the steering queue MUST reopen at `Run` entry so the second run's `Steer` calls are accepted with the zero-drop guarantee above rather than meeting the closed queue the first run left behind. **Concurrent runs on one value stay out of scope** and are not made safe by this change.

**Cross-run transcript state: CLOSED by AG-21, and closed by an enumerated inventory with an absence assertion rather than by a citation.** The clause this requirement carried — *"Cross-run transcript state remains AG-21's"* — is discharged here. The claim AG-21 proves is **not** "no state carries over": that is false, and it would forbid a continuing conversation. It is that **the only state outliving a run is state the caller explicitly owns or a shipped requirement already enumerates, and the harness itself retains nothing.** The inventory is closed, and every row is asserted:

| State | Outlives a run? | Owner |
|---|---|---|
| the terminal shutdown flag | **yes** — terminal, one-way, holds no transcript, never resumes a run | `R-CAN-005` |
| the session-start latch | **yes** — one-way, per-value bookkeeping only, holds no transcript | `R-RUN-013` consequence 2 |
| the caller-supplied transcript | **only if the caller set it**; unset ⇒ a fresh transcript per run | the **caller** |
| the run's cancel function | **no** — cleared at exit | `R-CAN-001` |
| the steering queue | **no** — reopened at entry, closed on every exit | this requirement, and `R-RUN-011` |

Three things about the discharge are stated so no later reader re-derives them:

- **Both branches are correct, and neither was asserted anywhere before AG-21.** With a caller-supplied transcript, an adversarial run's wind-down writes orphan synthesis and the turn close into it, so the next run's first provider request legitimately carries them — the caller's continuing conversation. With none supplied, each run resolves a fresh transcript and nothing carries. AG-21 asserts both.
- **The transcript row is proven as an ABSENCE with a defeat test, because a presence assertion proves nothing here.** `S-RUN-003` above and `S-CAN-002` both assert presence — that a second run is accepted, that a steer reaches its transcript, that it reaches its own terminal — and **neither asserts that anything from run 1 is absent from run 2**. Either would stay green with the fresh-per-run mechanism fully defeated. `S-CNH-014` names a **uniquely minted** run-1 artifact, asserts it appears nowhere in run 2's captured request, first proves the needle findable in run 1's own read-back, and carries its defeat — the same assertion re-run against a deliberately shared transcript, which MUST go RED.
- **"Concurrent runs on one value stay out of scope" is restated as STILL TRUE, unweakened.** AG-21 runs the whole combined matrix under the race detector and every cell drives exactly **one** `Run` at a time; signals fire from the test goroutine, which is `R-CAN-004`'s already-decided shape, not concurrency (`R-CNH-002`). A `-race`-clean concurrent-`Run` cell would publish a guarantee this clause and `agent-cancellation-tree/spec.md:204` both deny, so AG-21 fences it by construction rather than leaving it to the fixture.

(Previously: the surface was pinned as "exactly two methods", `Run` and `Steer`, by a bare count; and the requirement closed with "One `Run` per harness value. Cross-run state is **AG-21**'s.", which forbade the same-value reuse AG-14.1 requires.)

(Previously, at AG-21: the requirement ended *"**Cross-run transcript state remains AG-21's**: the only state AG-14 lets outlive a run is the terminal, one-way shutdown flag of `R-CAN-005`, which holds no transcript and never resumes a run."* — an open forward deferral, and an inventory of exactly one row written before the session-start latch of `R-RUN-013` existed. A reader could not tell from this requirement what the complete set of run-outliving state was, nor whether anything from an adversarially ended run reached the next run's provider request, and no scenario in this capability asserted either way.)

#### Scenarios

- **S-RUN-001** — Given a harness value constructed as a struct literal with only its required provider field set, when `Run` drives a scripted single-turn conversation from an external test package, then the run completes without any constructor call, the caller-visible field values are unchanged after `Run` returns except for the recorded sink-ownership flag, and the type's exported method set read by reflection is equal, in both directions, to `{Run, Steer, Interrupt, Shutdown}` — an extra exported method fails the assertion and so does a missing one.
- **S-RUN-002** — Given a run that has taken its terminal decision and returned, when `Steer` is called with a well-formed user message, then it returns an error that satisfies the typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`, the transcript is unchanged, and no further event reaches the consumer sink.
- **S-RUN-003** — **AG-14 serial reuse.** Given a harness value whose first `Run` has returned through the interrupt wind-down of `R-CAN-002`, when a second `Run` is invoked on that same value, then it is accepted, a `Steer` issued during that second run returns nil and its message reaches the second run's transcript before the next provider call, and the second run reaches its own terminal — proving the queue reopened rather than staying closed from the first run. Cross-referenced to `S-CAN-002`. *(AG-21 update: unchanged in claim, and its limit is now recorded rather than assumed away — this scenario asserts **presence** only. The absence half of the cross-run claim is `S-CNH-014`'s, with its defeat test; the two are separate and must not be conflated.)*

*(AG-21: the cross-run inventory above is discharged by `S-CNH-014`, `S-CNH-015` and the mandatory defeat bite `S-CNH-016` in `agent-concurrency-hardening`. No `S-RUN-` scenario is minted for it, so this capability keeps exactly one implementing test per claim.)*

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

**AG-15 makes "one per `Turn` invocation" load-bearing, and it is stated so no reader mistakes a retry for a violation.** A retry re-invokes `Turn` for the **same logical turn** (`R-RTY-002`), so a run of M logical turns whose attempts total N produces **N** turn brackets, not M. Each attempt's bracket carries its own freshly minted turn identity, and all of them lie on the one contiguous lane inside the one run bracket. The rule that actually binds is therefore unchanged and now stated in its stronger form: **one turn bracket per `Turn` invocation, opened and closed, with no invocation exempt.** That is precisely why every pre-stream failure path must close its bracket (`R-LSK-001`, AG-15's amendment): an unclosed bracket makes the very next attempt's `turn_start` a validator violation.

**AG-16 puts two new event kinds on that lane, and the bracket and lane rules above are restated as still-true rather than assumed to be.** A `cost_turn` is turn-placed and rides **inside** a turn bracket (`R-CST-001`); a `cost_session` is run-placed and rides **between** turn brackets or immediately before the run-close (`R-CST-005`). Four consequences follow, each stated because a reader could otherwise infer a violation:

- **The bracket counts are unchanged.** AG-16 opens no bracket and closes none. A cost event is a payload on an existing bracket, never a bracket of its own, so "exactly one run-open/run-close pair" and "one turn bracket per `Turn` invocation" hold at AG-16 unmodified.
- **The lane stays contiguous, and single-writer discipline holds.** Every cost event is stamped from the same shared `LaneStamper`. The `cost_turn` is stamped **inside** the turn, by the turn, exactly as that turn's other events are; each `cost_session` is stamped by the harness **between** turns or after the last one, which is the only window `NFR-RUN-003` permits the harness to touch the lane. No second writer is introduced at any point, and the sequence still starts at 1 and increases by exactly 1 with no gap, repeat or restart across the whole run.
- **Aggregation is a pure read on the existing single forwarding path.** The harness accumulates by reading each `cost_turn` as it forwards it; the event is forwarded exactly once and unmodified. No event is duplicated, re-stamped, re-ordered or withheld.
- **Cost events do not distinguish an attempt from a logical turn.** A retried attempt is still distinguished by transcript identity (`R-RTY-002`), never by a labelled event. Every attempt that reaches a `Completion` carries its own `cost_turn` inside its own bracket, and that is exactly what makes `R-CST-004`'s cumulative retry-inclusive by construction.

`CheckStream` MUST accept the recorded stream **unmodified**, across any number of attempts and with cost events present. `stream_check.go` MUST be byte-unchanged; this requirement is satisfied by the existing state machine, not by relaxing it, and neither AG-15 nor AG-16 requests a release for it.

(Previously: the requirement stated the bracket, lane and validator rules against a stream carrying no cost events, so nothing recorded whether a turn-placed and a run-placed cost payload were expected to ride the shared lane or to be exempt from it.)

#### Scenarios

- **S-RUN-020** — Given the complete event slice recorded from the two-turn run of `S-RUN-010`, when it is passed to `CheckStream` unmodified from an external test package, then it is accepted with no violation; and when the slice is walked, then it contains exactly one run-open and exactly one run-close, exactly two turn-open/turn-close brackets both nested between them, and a sequence that starts at 1 and increases by exactly 1 with no gap, no repeat and no restart across the whole run.
- **S-RUN-021** — **AG-15 attempts are brackets too.** Given a run whose first logical turn fails retryably before any output and then succeeds on a later attempt, when the complete event slice is recorded, then it contains exactly one run-open and one run-close; the number of turn-open/turn-close brackets equals the total number of `Turn` invocations the run made rather than the number of logical turns; every bracket is closed before the next one opens; the turn identities of the attempts belonging to one logical turn are pairwise distinct; the sequence starts at 1 and increases by exactly 1 with no gap, repeat or restart across the whole run including across the retry boundary; and `CheckStream` accepts the slice unmodified with `stream_check.go` byte-unchanged. Cross-referenced to `R-RTY-002` / `S-RTY-002`.
- **S-RUN-022** — **AG-16 cost events ride the existing brackets and the existing lane.** Given a multi-turn run whose stream carries both cost kinds, when the complete event slice is recorded and walked, then it still contains exactly one run-open and one run-close and one turn bracket per `Turn` invocation; every `cost_turn` lies strictly between a `turn_start` and its matching `turn_end`; every `cost_session` lies inside the run bracket and outside every turn bracket; the sequence starts at 1 and increases by exactly 1 with no gap, repeat or restart across the whole run with the cost events included in that walk; every event on the slice, cost events included, carries the run's own identity; each `cost_turn` appears exactly once on the stream, neither duplicated by the forwarding path nor withheld from it; and `CheckStream` accepts the slice unmodified with `stream_check.go` byte-unchanged. Cross-referenced to `R-CST-001` / `R-CST-005` / `NFR-RUN-003`.

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

The resolution contract is closed and MUST be stated where a reader cannot miss it: a parked set lives exactly one `Schedule` call, so a suspension inside a harness-driven turn resolves by exactly one of (a) an external `WakeParked` on the injected scheduler, or (b) cancellation of the run context, which the harness propagates unmodified into every turn. The harness adds **no third path and no timeout**. A run whose policy defers and whose owner neither wakes nor cancels does not terminate — **by design**; `R-APP-009` is the safety net.

**The forward reference "richer cancellation vocabulary is AG-14's" is CLOSED by AG-14**, and the question it left open is answered explicitly rather than by silence: **the wind-down bound of `R-CAN-006` is NOT the third path this requirement forbids.** Two independent reasons, both checkable:

1. It is **part of path (b)**. The bound is armed only by cancellation of the run context; without a cancellation it never exists, so it adds no way for a parked call to resolve that (a) and (b) did not already provide. A parked call still resolves by a wake or by a cancellation, and by nothing else.
2. It creates **no timer on the uncancelled path**. On a run nobody cancelled, the join waits on the call's own completion alone, exactly as it does today. `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` (`permission_protocol_test.go:1573-1600`), which deliberately drives `Schedule` with a bare `context.Background()` precisely to freeze this contract, MUST therefore pass with its source **file-unchanged**. A change that makes that file need an edit has taken the third path and MUST be rejected.

What AG-14 *does* change on path (b) is the **type** of the abort, not its existence: the parked call's typed failure now names which signal cancelled it (`R-CAN-003`), where it previously carried a raw context error.

Synchronization MUST be the stream itself: because registration precedes emission and the emission's acknowledgement precedes the parked wait (`R-APP-002`), a consumer that has read the decision-required event off the run stream may wake with a guaranteed-live entry. No wall clock, no timeout, no sleep.

This requirement also claims the **known gap `agent-permission-protocol/spec.md:172` carried to AG-13**, quoted verbatim: "**Known gap (carried to AG-13)**: the `R-APP-002` acknowledgement itself currently has no non-vacuous guard — deleting the acknowledgement leaves the package green. The behaviour is present and correct in production; the missing bite must observe the parked **wait**, not the registration." The bite below MUST observe the **wait**. A bite that observes registration re-encodes the very gap it closes and MUST be rejected.

(Previously: the requirement deferred the question with "richer cancellation vocabulary is **AG-14**'s" and left unstated whether a wind-down bound would violate its own "no third path and no timeout" clause.)

#### Scenarios

- **S-RUN-090** — Given a permission policy that defers the first resolution and allows the second, a caller-owned scheduler injected into the run, and a test reading the consumer sink event by event, when the decision-required event is read, then the run has not returned (checked by a non-blocking read of the run's completion channel) and the tool has not been invoked; and when `WakeParked` is called for that call identity, then it returns nil, the call executes, the run completes with the completed run outcome, the suspension's events lie **inside turn one's bracket**, and `CheckStream` accepts the whole stream unmodified.
- **S-RUN-091** — **(bite)** Given a scratch edit that replaces the parked **wait** with an immediate re-resolution, when a scenario whose policy's second resolution asserts a flag the test sets immediately before `WakeParked` runs, then it FAILS because the flag is unset at re-resolution — proving the assertion observes the parked wait rather than the registration. RED-recorded over repeated runs per `NFR-APP-002`'s `-count=15` discipline, then reverted.
- **S-RUN-092** — **AG-14 adds no third path.** Given the AG-14 branch, when the suite runs and `git diff` is taken against the merge base, then `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` passes **and its source file is byte-unchanged**; and when a run whose policy defers is driven with neither a wake nor a signal, then it does not terminate on its own within any bound — the bound is unarmed because nothing cancelled the run.

### R-RUN-011 — A failed turn ends the run typed, with no append and no close — cancellation carved out, genuine retryable pre-output failure retried, and the run's real spend reported

When a `Turn` invocation returns a non-nil error **that is not cancellation-caused and is not eligible for retry under `R-RTY-001`**, the harness MUST emit a run-close event carrying the failed run outcome and a non-nil typed failure, built through the public constructors, then close the consumer sink and return that error.

On that path the harness MUST NOT append anything to the transcript and MUST NOT call `CloseTurn`.

**AG-16 adds exactly one emission to this path, immediately before the run-close.** The harness MUST emit a `cost_session` carrying `CostLabelFinal` with the run's cumulative token figures at close time (`R-CST-005`, `R-CST-006`), because **tokens spent before a failure are real spend**: a failed run that reports nothing is indistinguishable at the consumer from a run that spent nothing. The insertion is bounded by four constraints, stated so it cannot be read as a relaxation of anything above:

- **The no-append/no-`CloseTurn` posture is untouched.** A cost emission writes no transcript, closes no turn, and synthesizes nothing. This requirement's second paragraph holds verbatim.
- **The run-close is unchanged and remains last.** Same failed run outcome, same non-nil typed failure with the true evidence of `R-RTY-012`, same returned error, same sink close. `cost_session` is `PlacementRun`, so the position is validator-legal, and `CheckStream` MUST accept the stream **unmodified** with `stream_check.go` byte-unchanged.
- **It is best-effort.** If the payload cannot be constructed, the emission is skipped and this path proceeds exactly as before. A cost emission MUST NOT convert one run outcome into another and MUST NOT alter the failure evidence.
- **The failed attempt itself contributes nothing.** A turn that closes aborted emits no `cost_turn` (`R-CST-001`), so the cumulative reported here is the real spend of the turns that closed **before** the failure — never five invented zeros for the one that did not.

**The retry carve-out, added by AG-15 — the narrowest possible removal of this requirement's original no-retry fence.** The original sentence — "MUST NOT retry, back off, or route to a fallback provider. Retry and failover are **AG-15**'s" — is **removed for exactly one class of error**: a `Turn` error that is an `*ai.Failure` reporting `Retryable() == true` and `PartialOutput() == false`, taken while the run context carries no cancellation cause and the attempt bound is not yet reached. For that class, and only that class, the harness MUST take the retry path of `R-RTY-001`'s gate G4 instead of the failure path above. The fence is **kept verbatim** everywhere else, and the two survivals are stated separately because they fail under different regressions:

- **Cancellation, verbatim.** A cancelled turn is never retried. The carve-out below is evaluated **first** — gate G0 — and no retry, backoff or failover consult may be reached from it.
- **Emitted output, verbatim.** A failure reporting `PartialOutput() == true` MUST NOT be retried, backed off, or routed to a fallback provider, **even when it reports itself retryable**. This is the G8 sentence at the harness (`R-RTY-003`), and it is a prohibition, not a default.
- **Exhaustion.** Once the attempt bound is reached, the failover seam is consulted exactly once and, on its decline, this requirement's failure path is taken (`R-RTY-010`).
- **Plain errors.** A `Turn` error that is not an `*ai.Failure`, and every plain error raised by the harness's own transcript writes, keeps this requirement's failure routing unchanged — no typed evidence exists, so no evidence-driven decision is possible.

When the failure path is finally taken, the run-close's typed failure MUST carry the **true** evidence rather than a hardcoded unavailable category, per `R-RTY-012`; a plain-Go-error cause keeps today's unavailable report byte-identically.

**The cancellation carve-out, added by AG-14.** A `Turn` error whose cause matches one of the cancellation sentinels of `R-CAN-001`, and equally a signal observed at an iteration boundary before another `Turn` is started, MUST NOT take the failure path above. It MUST take the wind-down path instead, and on that path the harness:

- **MUST** synthesize orphans over the transcript and **MUST** call `CloseTurn` — the "no append, no close" rule above is hereby **re-scoped to genuine failures**. This is not a relaxation: an interrupted transcript with open calls is exactly what `R-HIS-007` exists to repair, and leaving it open would strand the pairing invariant;
- **MUST** emit the run-scoped final cost figure and then a run-close carrying the interrupted or the shutdown run outcome and, per `RunEnd.validate`'s failure-iff-`Failed` rule (`run_events.go:156-161`), **no** `*Failure`. The cost emission's position in the wind-down's enumerated order is `R-CAN-002`'s, amended by AG-16 and not re-homed here;
- **MUST NOT** report `ai.FailureCategoryUnavailable` for the run — a cancellation is not a provider outage, and the two must not be indistinguishable at the consumer;
- **MUST NOT** retry, back off, or route to a fallback provider. The no-retry rule extends to cancellations **verbatim**: a cancelled turn is never retried. **AG-15 does not weaken this in any way** — a cancellation observed *during a backoff wait* also takes this path (`R-RTY-008`), so the wait is one more place the wind-down is reached from, never a place a retry escapes it.

A bare cancellation of the *caller*'s context, not routed through the signal methods, keeps this requirement's original failure routing (`R-CAN-001`, scope line), and is likewise never retried: it reaches the failure path through G1 or G2, not through G4. **It therefore takes this requirement's failure close, cost emission included** — the cost figure follows the close that is actually taken, never the signal that was intended.

This failure path MUST also close the run's steering queue as part of the same termination, under the queue's own mutex — the same critical section `R-RUN-002`'s atomic terminal decision uses. **Every** `Run` exit — this failure path, the cancellation wind-down path, a rejected prompt or steered-message append that reaches it, and the terminal-decision success path — MUST leave the queue closed before `Run` returns, so a `Steer` call that reaches the harness afterward always observes it closed and receives `R-RUN-001`'s typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`. A `Run` exit that leaves the queue open makes every later `Steer` return nil forever — indistinguishable, from the caller's side, from the silent drop `R-RUN-001` forbids. The reopen of `R-RUN-001` happens at the *next* `Run`'s entry, never at an exit. **A retry is not an exit**: the queue MUST stay open across attempts of one logical turn, and the steering drain MUST NOT run between two attempts of one logical turn — a message steered during a failing attempt belongs to the next **logical** turn, not to the next attempt, because a retry's transcript must be byte-identical to the failed attempt's (`R-RTY-002`).

**Two exits emit no cost event at all, and the carve-out is stated rather than inferred from silence**: the post-shutdown refusal, which `R-CAN-005` requires to emit no event whatsoever, and a `Run` that fails before its run-open event exists. Neither has a run bracket, so neither has anything to close.

(Previously: **every** non-cancellation `Turn` error routed unconditionally to the failed run outcome, and the requirement stated flatly that the harness "MUST NOT retry, back off, or route to a fallback provider", naming retry and failover as AG-15's — so a rate-limit hiccup before a single token was emitted ended the run exactly as a malformed response did, and the run-close reported the hardcoded unavailable category for every cause. **And no exit of any kind reported the run's token spend**, so a run that failed after several completed turns reported nothing and read, at the consumer, exactly like a run that spent nothing.)

#### Scenarios

- **S-RUN-100** — Given a provider scripted with a terminal mid-stream failure on turn one and a further script available so that a retry would be observable, when the run is driven, then the consumer observes the turn's typed closing brackets followed by a run-close carrying the failed run outcome with a non-nil failure and then the sink close; `Run` returns a non-nil error; the transcript holds only what was committed before the failure, with no entry appended after it; and the provider recorded exactly one request. *(AG-16 note: this run's only turn closes aborted, so it emits no `cost_turn`; the `cost_session(Final)` that now precedes the run-close reports every figure **absent** rather than five zeros, and the run-close's own position, outcome and failure evidence are unchanged. Asserted by `S-RUN-104`.)*
- **S-RUN-101** — Given a run that has ended through this requirement's failure path, when `Steer` is called after `Run` has returned, then it returns `R-RUN-001`'s typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`, never nil — the queue closes on every `Run` exit, not only the terminal-decision success path.
- **S-RUN-102** — **AG-14 carve-out.** Given a run interrupted mid-turn with a further script available so that a retry would be observable, when the run winds down, then the run-close carries the interrupted run outcome with a **nil** failure, no run-close carrying the failed outcome appears anywhere on the stream, the transcript's previously open calls are closed by synthesized-origin entries and the turn is closed, the provider recorded no request after the signal, and a `Steer` issued after `Run` returned receives the typed rejection — the queue closed on this exit too. Cross-referenced to `S-CAN-001` and `S-CAN-011`.
- **S-RUN-103** — **AG-15 carve-out, and the fence that survives it.** Given a run whose first logical turn fails retryably before any output and then succeeds, when the run is driven, then the run reaches the completed run outcome rather than the failed one, no run-close carrying the failed outcome appears anywhere on the stream, the transcript holds no entry written by the failed attempt, and `CloseTurn` was not called for it; and given the identical harness driven against a failure reporting itself **retryable but with output already emitted**, when the run is driven, then this requirement's failure path is taken, the provider recorded exactly one request, and the run ends failed — proving the fence was removed for one class of error and kept for the other. Cross-referenced to `R-RTY-002` / `S-RTY-002` and `R-RTY-003` / `S-RTY-003`.
- **S-RUN-104** — **AG-16 the failed run reports its real spend.** Given a run whose first logical turn completes normally with scripted usage and whose second fails into this requirement's failure path, when the stream is recorded, then a `cost_session` carrying `CostLabelFinal` appears **immediately before** the run-close; its figures equal, figure for figure, the cumulative over the `cost_turn` events observed **on that same stream** — an equality against the observed events, never against a hand-computed literal; the failing turn contributed no `cost_turn` and no invented zeros; the run-close still carries the failed run outcome with its non-nil typed failure whose evidence is unchanged (`R-RTY-012`) and is still the run bracket's last event; nothing was appended to the transcript and `CloseTurn` was not called; `Run` returns the same error it returned before this change; and `CheckStream` accepts the stream unmodified with `stream_check.go` byte-unchanged. And given a run that fails on its **first** turn before any usage is reported, when its stream is recorded, then the `cost_session(Final)` reports every figure **absent** rather than zero. Cross-referenced to `R-CST-006` / `S-CST-012`.

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

### R-RUN-013 — AG-20 adds one unexported latch, two pure forwarder reads, one per-run lane — and NO third path, NO timeout, and NO harness method

**AG-20 is the first milestone to add a goroutine class to the run's lifetime that is not an event carrier.** A new goroutine landing inside a requirement that never mentions it is this repository's known spec-breakage shape, so the four things AG-20 adds to the run driver are enumerated with their verdicts rather than left to be re-derived.

1. **One exported configuration field and NO exported method.** `Harness` gains a `Hooks` field — a zero-value-inert value type, the house pattern every other optional seam already uses. `R-RUN-001`'s surface is enumerated **by name**, so a field costs it nothing and a method would cost it two shipped pins at once. AG-20 adds none.
2. **One unexported, one-way session-start latch, of the shipped shutdown flag's class.** It is per-value bookkeeping only, holds **no transcript**, is **never resumed**, and **does not pre-empt AG-21's cross-run state** — the same sentence the shutdown flag already carries, with one word changed. It is set at `Run` entry under the existing exclusion, **after** the shutdown check, so a shut-down value leaves it untouched. Because the type already embeds an exclusion primitive, a post-use copy of a `Harness` value is a **build-time** finding under the standard vet check, so the latch cannot be laundered by copying.
3. **Two pure reads beside the per-attempt forwarder, and a run-frame-local accumulator.** The forwarded turn-close outcome and a per-logical-turn cost accumulation are read where the existing captures already sit. The accumulator is a **local of the run's frame** and MUST NOT be a `Harness` field: `R-CST-004` forbids the latter and the existing cumulative accumulator's own comment states the rule. Every read is downstream of the forwarder's completion, so all of them inherit the close-of-channel happens-before the forwarder already relies on. `R-RUN-003` holds unamended: nothing is rewritten, synthesized, suppressed or reordered.
4. **One per-run observer lane, created only when at least one observing hook is registered, and a terminal-boundary snapshot — with NO wall clock anywhere.** The lane emits nothing. Its terminal snapshot runs on `Run`'s own goroutine, **after** the run-close has been sent on every returning arm and **before** the consumer sink is closed. **`R-RUN-010`'s prohibition is honoured literally**: AG-20 adds no timeout, no deadline, no sleep, no poll, and **no join on a stalled observer**. A run whose observer stalls forever still returns; the observer's goroutine leaks, **by design and as the caller's leak**, and AG-21 inherits that knowingly (`NFR-HKS-005`).

**Two observable consequences of the snapshot's position MUST be stated rather than discovered**, because both are things a consumer can see and neither is a defect:

- A **stalling reporter** delays `Run`'s return **and** the consumer sink's close. A consumer ranging the sink has already received the run-close but never observes termination. Caller's code, caller's stall.
- An observer that finishes just after the snapshot is **reported once and completes anyway**. The report is an honest statement about the terminal boundary, not a verdict about the hook.

**On the inert path nothing above exists**: no lane, no goroutine, no queue, no snapshot. The nil path is byte-for-byte today's path on every arm (`R-HKS-012`).

#### Scenarios

- **S-RUN-113** — **AG-20: the method set is exactly FIVE, matching what is actually shipped — corrected by `sdd-verify` (MAJOR-7) from a stale four-name count — and the absences are asserted rather than claimed.** Given the merged AG-20 change, when `Harness`'s exported surface is enumerated from `package agent_test`, then its method set is exactly the **five** names `Compact, Interrupt, Run, Shutdown, Steer` — measured, not `R-RUN-001`'s own **four**-name enumeration, which is a pre-existing staleness AG-18 left when it added `Compact` and never back-annotated; AG-20 did not create that staleness, but restating the OLD count here would have made it a fresh, current claim, so this scenario asserts the true, current count instead. `harness_test.go`'s named-method assertion and `scope_fence_test.go`'s method assertion are **byte-unchanged** and green, and the only exported addition AG-20 itself makes is one configuration field. When the run driver's production source is read as raw bytes, then it declares **no** per-logical-turn cost accumulator as a `Harness` field — the accumulator is a local of the run's frame — and it contains **no** timer, deadline, sleep, poll or join on an observing hook. When `git diff` is taken against the merge base, then the module's no-deadline permission pin is **file-unchanged** and passes. When a run whose observer is held indefinitely by the test gate executes, then `Run` **returns** with the gate still held, its recorded stream's **event-KIND sequence** is equal to the same script's with no hooks installed *(precision added by `sdd-verify` MINOR-5 and MADE TRUE in round 2, W-1: kind-sequence equality via `kindsOfHKS`/`kindsEqualHKS`, not a payload-field or full-event comparison — the identical mechanism `S-HKS-017` uses for this same fixture. Round 2 found the test performed only a length comparison, which this sentence over-claimed; the TEST was strengthened to the claim rather than the claim weakened to the test, because kind-sequence equality is the stronger property and both helpers already existed.)*, and `CheckStream` accepts it unmodified. When a run with a **zero-value** `Hooks` executes on each of the success, turn-failure, retry-exhaustion, interrupt, shutdown and compaction arms, then `runtime.NumGoroutine()` returns to its baseline after every one — no lane goroutine exists on the inert path. And no assertion in this scenario reads elapsed time. Cross-referenced to `R-HKS-004`, `R-HKS-005`, `R-HKS-008`, `R-HKS-012` and `S-HKS-026`.

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
| Cross-event run-identity consistency enforced by `CheckStream` | **AG-19.** AG-13 upholds the property by construction and asserts it in its own tests (`R-RUN-004`); the validator is not strengthened and `stream_check.go` stays byte-unchanged. *(Still true at AG-14 and at AG-16: `stream_check.go` stays byte-unchanged, and AG-16's cost events carry the run's identity by construction, asserted in this capability's own tests at `S-RUN-022`. Still true at AG-17, which emits no event at all and leaves `stream_check.go` byte-unchanged)* |
| Retry or failover on a failed turn | **AG-15. CLOSED by AG-15.** Turn-level retry ships in the run driver, owned by `agent-retry-failover` (`R-RTY-001`…`012`) and carved out of `R-RUN-011` above for exactly one class of error: an `*ai.Failure` that is retryable, carries no partial output, is not cancellation-caused, and is within the attempt bound. The **failover implementation** remains deferred — AG-15.3 ships the named seam and its documented contract, and its v1 implementation declines (`agent-v1-scope`'s `AGS-D`) |
| A compaction check between turns | **AG-17** inserts it at AG-13's turn boundary; **AG-18** implements compaction. Nothing here anticipates its shape. *(Still unclaimed at AG-16, which inserts a cost emission at the turn boundary and no compaction check; the two are independent insertions and AG-16 pre-empts nothing about AG-17's shape)* **CLOSED by AG-17 — but only the check half, and the split is the point.** AG-17 ships the check exactly where this row placed it: a nil-default `ContextStrategy` seam on the caller-owned harness, consulted in `Run`'s outer per-logical-turn loop between the transcript resolution (`harness.go:512`) and the attempt loop (`harness.go:562`). **Three properties of the insertion are recorded here rather than left to be rediscovered.** *(1) Its cardinality is counted in LOGICAL TURNS, not attempts*: a run of N logical turns consults it exactly N times however many provider calls those turns issue, because a seam consulted inside the attempt loop could — once AG-18 gives it teeth — mutate the transcript between two attempts of one logical turn, defeating the very argument `R-RTY-002`'s comment relies on (`harness.go:601-612`). This row's own phrase "at AG-13's turn boundary" is what settles it. *(2) It emits nothing and mutates nothing*: no `EventKind` is registered, no compaction event is emitted, no history entry is written, and a run with the shipped never-compact default installed produces a **byte-identical** event stream and a **byte-identical** history read-back to the same run with the field nil. *(3) Compaction stays wholly AG-18's, enforced by the compiler rather than by intent*: AG-17's verdict type ships with **no field**, so a verdict requesting compaction is unconstructible by any strategy a caller could write, and AG-18 adds compaction fields non-breakingly on the `FailoverVerdict` extension path (`failover_policy.go:57-62`). Owned by `agent-context-strategy` (`R-CTX-001`, `R-CTX-003`, `R-CTX-004`). **Not closed here**: compaction itself — summarisation, transcript surgery, protected recent turns and the compaction event family's first emission — which remains AG-18's whole substance. **NOW CLOSED by AG-18 — the second half, and the row is complete.** AG-18 acts on the verdict captured at that exact seam, in the **same** turn-boundary gap and at most once per boundary, and does so **without moving, duplicating or re-timing the check**: `R-CTX-001`'s once-per-logical-turn cardinality is untouched, and a second seam consultation was rejected for exactly that reason. What ships: a summarisation call on an **injected** provider with injected options and an injected instruction, on the run's own context; a **prefix** replacement through history's single commit primitive at a mark-aligned, pairing-closed boundary; the protected tail preserved value-identical in Layer 1 values and origins; the compaction family's **first production emission**, inside its own dedicated turn bracket so `stream_check.go` stays byte-unchanged; and an on-demand door refusing typed whenever a run is in flight. **The `R-RTY-002` hazard this row named is closed rather than inherited**: the harness rebuilds the transcript from the mutated history **before** entering the attempt loop, so every attempt of the following logical turn receives the same rebuilt slice by reference. Owned by `agent-compaction` (`R-CMP-001`…`R-CMP-014`). **Not closed here**: *when* to compact — any threshold arithmetic over budget against accounting — which remains Layer 3's |
| Cancellation semantics — interrupt vs. shutdown vs. deadline | **AG-14.** AG-13 propagates the context unmodified and stops; it defines no cancellation vocabulary and adds no timeout. **PARTIALLY CLOSED by AG-14**: interrupt and shutdown ship here, owned by `agent-cancellation-tree` (`R-CAN-001`…`008`). **Deadline remains unclaimed** — `0003:1373`'s "both ≠ deadline" is a distinctness claim, not a mandate, and AG-14.3's bound is a wind-down bound, not a run deadline |
| Cost aggregation across turns | **AG-16. CLOSED by AG-16**, and the rule it closes on is stated rather than left to be looked up, because its exact wording is what makes it retry-safe. Cumulative is defined as **the sum over every `cost_turn` event emitted within the run bracket** — not "the sum over the turns that succeeded" — which is retry-inclusive **by construction**, since each attempt of a retried logical turn is its own `Turn` invocation with its own bracket (`R-RUN-003` above, `R-RTY-002`) and any attempt reaching a `Completion` emits its own counted `cost_turn`. **No retry-awareness exists in the accumulator.** The cumulative is emitted as a `cost_session` immediately before the run-close on **every** run close — success, failure (`R-RUN-011` above) and cancellation wind-down (`R-CAN-002`) — with running `Estimate`-labelled figures between logical turns. It is **run-scoped**: it lives on `Run`'s own stack frame, is never a harness field, and outlives nothing. Owned by `agent-cost-events` (`R-CST-004`, `R-CST-005`, `R-CST-006`). **Parent-scoped aggregation over delegated runs remains deferred to AG-19** and is not closed here. *(Still true at AG-17, whose token accounting is a separate provenance type sharing no field with `ai.Usage`: no estimate is ever converted into an `ai.TokenCount` or routed into a cost figure, and `cost_events.go` and `cost_usage.go` are byte-unchanged — `R-CTX-008`, `S-CTX-017`)* |
| A production subagent tool, nested or child runs | was: *"**AG-19.** No subagent tool ships in v1 (`0003:1794`)"* — **SPLIT by AG-19, because the row bundles two things with different fates, and closing it whole would be a false claim.** **Nested and child runs: PROVEN by AG-19.** A child `Harness` value runs to completion inside a parent's tool call, on a context derived from the tool's own `ctx`, with its events mirrored onto the parent's stream through a publishing seam and its own stream captured and validated separately. Four structural properties are proven under it: attribution by a one-hop parent walk, sibling children interleaving with no cross-talk under `-race`, cancellation inherited leaf-first through the **existing** tree, and cost and permission crossing correctly. **The run driver itself is unchanged**: it gained no requirement, no scenario, no field and no bracket — a hosted child run drives its **own** `Harness.Run`, and the parent's run driver never learns a child exists. `R-RUN-003`'s closed sequences hold **by mechanism, not fixture luck**: the seam refuses every bracket-role and terminal kind (`R-DEL-002` gates 2 and 4), so a child's `run_start`/`run_end` and `turn_start`/`turn_end` can never reach the parent's lane — bite `S-DEL-021` admits one deliberately and watches the parent's `CheckStream` fail with a duplicate run-open, so the gate is asserted rather than assumed; and a mirrored event is stamped by the parent's single dispatcher goroutine, re-stamping discarding the prior sequence on a copy (`sequence.go:50-58`), so both lanes stay independently contiguous and are never merged. `R-RUN-011`, `R-RUN-012` and `R-RUN-001` likewise hold — the seam is revoked when the hosting call completes on every exit path (`R-DEL-003`), it mutates no caller field, and AG-19's sibling scenario uses **two distinct `Harness` values**, never two concurrent `Run` calls on one value. **A production subagent tool: STILL NOT SHIPPED, and still post-v1** (`0003:1803`). AG-19's every subagent concept lives in `package agent_test`, which production code cannot import, and the seam's concrete type and installer are unexported so no code outside the package can mint one. *(A reader must not take "nested runs proven" as "a subagent tool shipped". The row's first half closes; its second half does not, and its second half is the one AG-02's verdict governs.)* Any milestone that later admits a kind AG-19's rule refuses, or that lets a publish survive its hosting tool call, MUST re-check every rule named above before doing so |
| Multi-turn state beyond a single run | **AG-21** (`agent-run-driver`'s `R-RUN-001`, its one-run-at-a-time clause, this capability's own re-scoped clause in `R-RUN-001`). AG-13 owns one run's iteration; AG-14 adds serial reuse of a harness value and one terminal, transcript-free shutdown flag, and pre-empts nothing else. **Pointer corrected:** this row previously cited `agent-loop-skeleton/spec.md:106`, which is `R-LSK-005` coverage text and has nothing to do with cross-run state. That was a **pre-existing citation defect**, recorded here and corrected rather than propagated; the same stale pointer appears in this change's proposal risk row 7 and MUST NOT be copied forward. *(Still true at AG-16: the cumulative cost figure is run-scoped and carries nothing across runs. Still true at AG-17: the budget is a harness-scoped configuration value Layer 3 states, never run state the harness accumulates, and the accounting is recomputed per logical turn and retained nowhere)* |
| The four-hook taxonomy and a `ToolSource` port widening | **AG-20.** The AG-13 charter never mentions tool sources; `agent-tool-scheduler/spec.md:138` is re-homed by this change's delta |
| Persistence or session reload of a run | **Layer 3.** The harness holds state in memory and never touches a file (`0003:110`) |
| A real provider or a real tool | **Never in Layer 2.** `agenttest` scripts only (`0003:123`) |
| Any edit under `backend/agent/src/ai/**` | **Not this milestone.** Layer 1 is consumed, never edited. *(Still true at AG-16, which reads `ai.Usage` and writes nothing under that tree. Still true at AG-17, which type-asserts the shipped `ai.TokenCounter` and reads `ai.TokenCount`, and whose diff under that tree is **empty** — declaring a Layer 2 counting interface instead would have violated `R-AMP-017`)* |
| A new `EventKind`, a new `TurnOutcome`, or a new exported `History` method | **Not this milestone**, and forbidden under this change. *(Still true at AG-14: it registers no `EventKind`, adds no `TurnOutcome`, and adds no exported `History` method. It does add one `RunOutcome` member under the recorded `R-LSK-004` release — a different vocabulary from the two named here. Still true at AG-16, which adds none of the three: AG-06 minted `cost_turn` and `cost_session` and AG-16 only emits them. Still true at AG-17, which adds none of the three: the compaction family was minted by AG-06 and **AG-18** is its first caller, and the seam reads history through the existing `transcriptFromHistory` route rather than a new exported one)* **BREACHED, deliberately and in exactly one third, by AG-18 — recorded here rather than discovered in a diff.** AG-18 adds **one** new exported `History` method: the **prefix replacement** of `R-HIS-001` as amended, dispatched from the same single validating commit primitive so `R-HIS-004`'s "exactly one commit path, no privileged bypass" stays literally true (`history.go:343` declares `ReplacePrefix`; its dispatch through the single commit primitive is `history.go:434-449`, re-verified in this worktree). The breach was **forecast by this row's own AG-17 annotation**, which named AG-18 as the compaction family's first caller; it is confined to one prefix-shaped route with no mid-span variant; and it is fenced by `R-HIS-001`'s replacement invariants, named clause by clause in this change's [`../agent-history/spec.md`](../agent-history/spec.md) delta. **The exported-surface guard is updated deliberately, in the same commit as the route**: before this change the exported surface is exactly `Append`, `CloseTurn`, `SynthesizeOrphans`, `Entries` and `Len`, and the replacement is the sixth member; `S-HIS-030`'s route enumeration gains it in the same commit, with `S-HIS-102` as the bite that proves the enumeration closed over the widened surface. **The other two-thirds stay TRUE at AG-18 and are restated rather than assumed**: AG-18 registers **no** `EventKind` — AG-06 minted the compaction family and AG-18 is only its first emitter — and adds **no** `TurnOutcome`; the compaction bracket uses the existing outcome members through the existing constructors. AG-18 also adds **no** `CostLabel`. The row is **not** re-opened for any later milestone: a further exported `History` route needs its own recorded delta |
| An `L2C-08` doc-contract row | **Not this milestone.** AG-13 declares no new package-wide guarantee; it rides `L2C-03`/`L2C-04`/`L2C-05`/`L2C-07` at run scope. AG-19's child runs re-open the question. **CLOSED by AG-14, earlier than forecast**: AG-14 declares a package-wide liveness and control guarantee no existing row covers, so the `L2C-08` row ships here, owned by `R-CAN-008`. *(AG-16 declares no new package-wide guarantee and adds no row; `doc.go` and `doc_contract_guard_test.go` are byte-unchanged. AG-17 likewise declares none and adds none — it adds no upward surface, emitting no event and adding no exported history route — so the table stays at its eight committed rows and both files stay byte-unchanged, on the AG-15 precedent)* |

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
