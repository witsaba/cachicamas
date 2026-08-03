> **Archived 2026-08-03.** This is the delta spec as written during the SDD cycle. The **promoted live contract** is [`openspec/specs/ai-fake-provider/spec.md`](../../../../../specs/ai-fake-provider/spec.md) — read that file, not this one, for the current requirement text, including the resolution of `R-AFP-014`'s pre-apply correction and design's resolution of the two items this delta left open.

# Spec — the scripted fake provider

> **Change**: `cachicamas-ai-fake-provider` · **Milestone**: AI-21 — Build a scripted fake provider (Wave 3 "Prove", the wave's **first** milestone)
> **Nodes**: AI-21.1 `[leaf]` walking skeleton · AI-21.2 `[leaf]` tool call · AI-21.3 `[leaf]` terminal error · AI-21.4 `[leaf]` delays and the blocked stream · AI-21.5 `[leaf]` cancellation fidelity · AI-21.6 `[leaf]` request capture · AI-21.7 `[leaf]` scripted reasoning · AI-21.8 `[leaf]` sequential-call scripting
> **Capability**: `ai-fake-provider` — **new**. Promoted to `openspec/specs/ai-fake-provider/spec.md` at archive.
> **Predecessor**: [`proposal.md`](../../proposal.md) · Engram `sdd/cachicamas-ai-fake-provider/proposal` (obs #2370), `sdd/cachicamas-ai-fake-provider/explore` (obs #2369)
> **Requirement IDs**: `R-AFP-0NN` · **Scenario IDs**: `S-AFP-0NN` — prefix re-verified unused across `openspec/specs/` and `openspec/changes/` at spec time; siblings AI-22 and AI-23 must not reuse it
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Binding predecessors, cited by identifier and never modified**:
> [`ai-model-provider`](../../../../../specs/ai-model-provider/spec.md) (AI-20) — `R-AMP-003` external implementability, `R-AMP-005` … `R-AMP-008` the pre-stream contract, `R-AMP-009` … `R-AMP-012` the mid-stream contract, `R-AMP-013` the local single-purpose producer that this milestone succeeds;
> [`ai-stream-lifecycle`](../../../../../specs/ai-stream-lifecycle/spec.md) — § 3 carrier, § 4 ownership, § 5 cancellation, § 6 buffering, § 7 failure delivery, § 9 the eight statements;
> [`ai-event-envelope`](../../../../../specs/ai-event-envelope/spec.md) (AI-14) — the envelope, its kind vocabulary, `CheckEmit`, and per-stream sequencing;
> [`ai-text-events`](../../../../../specs/ai-text-events/spec.md) (AI-16), [`ai-reasoning-events`](../../../../../specs/ai-reasoning-events/spec.md) (AI-17), [`ai-tool-call-events`](../../../../../specs/ai-tool-call-events/spec.md) (AI-18), [`ai-provider-errors`](../../../../../specs/ai-provider-errors/spec.md) (AI-19) — the four script vocabularies;
> [`ai-model-request`](../../../../../specs/ai-model-request/spec.md) (AI-10.5 readability, AI-10.6 immutability) — what request capture asserts on
> **Depends on**: AI-20 (shipped) · **Blocks**: AI-22, AI-23, doc 0003 wave C
> **Sources**: [doc 0002 — Layer 1 task graph](../../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) §§ AI-21.1 … AI-21.8 (lines 1168–1251) · ADR 0005 § D2 / Guard C (the `src/agenttest` sibling layout)

---

## Purpose

Layer 1 has a provider interface and no provider. Everything above it — AI-22's stream test kit, AI-23's conformance suite, and every Layer 2 agent-loop test — needs a producer it can script, and today the only one is unexported inside `src/ai`'s own tests (`R-AMP-013`). This spec constrains that producer's promotion into an importable testing library: **what a script can express, what draining a scripted stream must yield, and which physics the fake is forbidden to simplify.**

The governing constraint is doc 0002's own: the fake MUST be **contract-faithful, not convenient**. Layer 2 will build on what the fake does, not on what the document says. Where the real contract drops events, this spec states the drop as a **positive requirement** — a fake that closes cleanly there teaches the wrong physics permanently, and it does so silently.

## Delta status against existing specs

`openspec/specs/` was re-read at spec time. It carries no fake, double, or test-library capability: `ai-fake-provider` is **new**, so this is a full capability spec with **no MODIFIED, no REMOVED and no RENAMED requirement**. `ai-model-provider`, `ai-stream-lifecycle`, `ai-event-envelope`, `ai-text-events`, `ai-reasoning-events`, `ai-tool-call-events`, `ai-provider-errors` and `ai-model-request` are read and **cited by identifier, never modified**.

**Two Wave 2 carryovers are out of scope and stay unassigned**: the `CheckEmit` rule 4 failure-path coverage gap (`ai-event-envelope`) and the missing redacting `GoString()` on the failure payload (`ai-provider-errors`, `R-AIP-009`). Both are recorded in those specs as "owned by Wave 3" **generically**; doc 0002 assigns neither to AI-21, and neither appears in AI-21's charter or its eight leaves. Absorbing either into this change is scope creep, and this paragraph is the record that it was decided rather than forgotten.

## Definitions used by this spec

- **The fake** — the exported provider implementation this milestone ships in `src/agenttest/`, satisfying the AI-20 provider interface.
- **A script** — the test-authored description of one call's behaviour: the events it emits, in order, and its termination.
- **The script queue** — the ordered sequence of scripts a single fake value holds, one consumed per streaming call (AI-21.8).
- **Exhaustion** — a streaming call made against a fake whose queue holds no unconsumed script.
- **A synchronization point** — a test-controlled release mechanism that gates a scripted hold. It is satisfied by a signal from the test, **never** by elapsed wall-clock time.
- **Loud failure** — an immediate, attributable, observable failure that names the fake and the exhausted queue. Its mechanism is design's; that it is neither a hang nor a silent success is required here.
- **Contract-faithful** — reproducing an AI-20 behaviour including its unattractive parts, rather than the nearest behaviour that makes a test simpler to write.

---

## AI-21.1 — Walking skeleton: a scripted text response `[leaf]`

### R-AFP-001 — The fake is importable from outside `ai` and needs no network

The fake MUST be exported from a package that is not `ai`, MUST be constructible and drivable by a test in any other package in the module, and MUST satisfy the AI-20 provider interface. It MUST perform no network I/O, no filesystem I/O and no clock-dependent work on any scripted path. Its existence MUST itself stand as a second proof of `R-AMP-003` — that the interface is externally implementable.

#### Scenarios

- **S-AFP-001** — Given a test in a package that is neither `ai` nor the fake's own package, when it constructs the fake and assigns it to the provider interface, then it compiles and satisfies the interface without naming anything unexported from `ai`.
- **S-AFP-002** — Given a fully scripted stream drained to close, when the run is inspected for external effects, then no socket, no file handle and no wall-clock dependence was involved, and repeating the run yields byte-identical events.
- **S-AFP-003** — Given the landed package, when a reader looks for the fake's declaration, then it is exported production code in `src/agenttest/`, a direct sibling of `src/ai`, and the AI-20 signature guard still passes unchanged.

### R-AFP-002 — A scripted text response drains exactly as scripted

WHEN a test scripts a response start, text deltas and a completion, THEN draining the returned carrier MUST yield **exactly** those events, in the scripted order, with no event added, reordered, coalesced or omitted. Sequence numbers MUST run 1…N over the stream with no gap and no repeat, and the stream MUST terminate by the carrier's close.

#### Scenarios

- **S-AFP-004** — Given a script of "start, two text deltas, complete", when a consumer drains the carrier to close, then it observes exactly four events in the scripted order, sequenced 1, 2, 3, 4, followed by close.
- **S-AFP-005** — Given the same script, when the consumer inspects each delta's bytes, then each equals the scripted fragment byte-for-byte and no fragment was merged with its neighbour.
- **S-AFP-006** — Given a script whose events would violate the envelope's emission rules, when the fake is driven, then the violation surfaces as a loud failure at the fake rather than as an invalid event delivered to the consumer.

### R-AFP-003 — Concurrent streams sequence independently

Every streaming call MUST sequence its own stream from 1, independently of every other stream, including streams from **the same** fake value and streams running concurrently. Per-stream sequencing MUST be observable at the consumer and MUST hold under the race detector.

#### Scenarios

- **S-AFP-007** — Given two fakes each scripted with a three-event response, when both are streamed concurrently and both are drained under `-race`, then each consumer observes 1, 2, 3 and the race detector reports nothing.
- **S-AFP-008** — Given one fake value streamed twice, when both carriers are drained, then each stream's sequence starts at 1 rather than continuing the previous stream's count.

### R-AFP-004 — *(non-negotiable)* The fake reproduces AI-20's stream physics, it does not approximate them

The fake MUST reproduce the AI-20 mid-stream contract exactly: exactly **one** producer goroutine per stream, exactly **one** closing site that runs on every exit path and never before the last send attempt, and **every** send — the terminal one included — waiting on cancellation as well as on the stream. The fake MUST NOT substitute a more convenient behaviour for a contract behaviour, and MUST NOT close cleanly on a path where the contract drops events.

#### Scenarios

- **S-AFP-009** — Given the fake's producer, when a reviewer counts its sending goroutines and its closing sites, then there is exactly one of each, per `R-AMP-009`.
- **S-AFP-010** — Given the fake driven separately to completion, to a terminal error and to cancellation, when each run finishes under `-race`, then the stream closed exactly once on each path and no send occurred after the close.
- **S-AFP-011** — Given a consumer that stops reading, when the caller cancels the context, then the producer exits rather than blocking forever on a send.

---

## AI-21.2 — Scripted tool call `[leaf]`

### R-AFP-005 — A delta-carrying tool call reconstructs to exact argument bytes

A script MUST be able to express a tool call as start → argument deltas → end. The drained events MUST reconstruct to the **exact** argument bytes the test scripted, with the tool identity and name preserved.

#### Scenarios

- **S-AFP-012** — Given a tool call scripted with a name, an identity and arguments split across three fragments, when the consumer drains and reconstructs, then the result equals the scripted arguments byte-for-byte and carries the scripted identity and name.
- **S-AFP-013** — Given the same script, when the consumer inspects the drained events, then it observes a start, exactly the scripted deltas in order, and an end — none synthesised and none dropped.

### R-AFP-006 — A zero-delta tool call is scriptable and indistinguishable after reconstruction

A script MUST be able to express a tool call as start → end with **zero** deltas. After reconstruction a consumer MUST NOT be able to distinguish it from a delta-carrying call bearing the same arguments.

#### Scenarios

- **S-AFP-014** — Given a zero-delta tool call and a delta-carrying tool call scripted with identical arguments, when both are drained and reconstructed, then the two reconstructions are equal.
- **S-AFP-015** — Given the zero-delta script, when the consumer drains it, then it observes a start and an end with no delta between them, and reconstruction still yields the full arguments.

### R-AFP-007 — Interleaved tool calls keep their ordinals and reconstruct independently

A script MUST be able to interleave the events of two or more concurrent tool calls on one stream. Each call MUST keep its own ordinal, and each MUST reconstruct independently of the interleaving.

#### Scenarios

- **S-AFP-016** — Given two tool calls whose starts, deltas and ends are scripted interleaved, when the consumer drains and reconstructs per ordinal, then each call yields its own scripted arguments with no cross-contamination.
- **S-AFP-017** — Given that same stream, when the consumer inspects each event's ordinal, then every event carries the ordinal of the call it belongs to, unchanged by the interleaving.

---

## AI-21.3 — Scripted terminal error `[leaf]`

### R-AFP-008 — Any failure category is scriptable, in both partial-output states

A script MUST be able to end in a terminal error carrying **any** AI-19.2 category, and MUST be able to do so both **with** prior output and **without** it. The partial-output discriminator the drained failure carries MUST match the state the script actually produced; the fake MUST NOT hard-code one state or infer one that contradicts the stream.

#### Scenarios

- **S-AFP-018** — Given, for each category in the failure vocabulary, a script that ends in a terminal error of that category after two text deltas, when each is drained, then the terminal event carries that exact category and reports that output preceded it.
- **S-AFP-019** — Given a script that ends in a terminal error having emitted nothing, when it is drained, then the terminal event reports that no output preceded it, and it is still delivered mid-stream rather than as a pre-stream failure.

### R-AFP-009 — Terminal exclusivity is observable at the consumer

After a scripted terminal error the stream MUST close and **nothing** MUST follow it — no completion, no further content event, no second terminal. A script that names events after a terminal error MUST fail loudly at the fake rather than deliver them.

#### Scenarios

- **S-AFP-020** — Given a script ending in a terminal error, when the consumer keeps receiving past that event, then the next receive reports a closed stream.
- **S-AFP-021** — Given a script that places a text delta after a terminal error, when the fake is driven, then it fails loudly and names the misplaced event rather than emitting it.

---

## AI-21.4 — Delays and the blocked stream `[leaf]`

### R-AFP-010 — A stream can be held open and released through a synchronization point, never a sleep

A script MUST be able to hold the stream **open without emitting**, so a consumer's own timeout can be tested, and MUST be releasable on demand by the test. The hold MUST be released by a test-controlled synchronization point. Neither the fake's mechanism nor any assertion proving it MUST depend on a wall-clock sleep.

#### Scenarios

- **S-AFP-022** — Given a script that emits one event and then holds, when the consumer attempts a second receive, then it blocks until the test releases the hold, and only then observes the next scripted event.
- **S-AFP-023** — Given a held stream, when the test releases it and then drains, then the stream completes and closes with the remaining scripted events intact and in order.
- **S-AFP-024** — Given the landed tests for this leaf, when a reviewer searches their assertions for a wall-clock sleep used as coordination, then none exists — every wait is on a synchronization point or a bounded test deadline.

### R-AFP-011 — A script can deterministically saturate an unread consumer

A script MUST be able to emit faster than an unread consumer drains, saturating the stream's buffer **deterministically** rather than by timing luck. This is the setup AI-20.3's sanctioned loss path requires, and the fake MUST reach it reproducibly.

#### Scenarios

- **S-AFP-025** — Given a script with more events than the stream's buffer admits and a consumer that reads nothing, when the test observes the producer, then it is blocked on a send with the buffer saturated, reached without any wall-clock assumption.
- **S-AFP-026** — Given a slow consumer that pauses and resumes **without** cancelling, when it eventually drains to close, then it receives every scripted event in order and none was dropped — backpressure waited, it did not drop.

---

## AI-21.5 — Cancellation fidelity `[leaf]`

### R-AFP-012 — Mid-stream cancellation behaves exactly as AI-20.3 requires, including the drop

WHEN the consumer cancels mid-script, the fake MUST close the stream within bounded time, MUST perform no send after the close, and on the **saturated** path MUST drop the remaining scripted events and close **bare** — with no terminal event. The fake MUST NOT force a terminal event, a synthetic cancellation event or a clean completion onto that path. The property MUST hold under `-race`.

#### Scenarios

- **S-AFP-027** — Given a mid-script cancellation and a test deadline, when the test waits for the stream to close under `-race`, then it closes before the deadline and the race detector reports nothing.
- **S-AFP-028** — Given a saturated stream cancelled by the caller, when the consumer drains what remains, then the stream closes with **no** terminal event and the undelivered scripted events are absent.
- **S-AFP-029** — Given that same run, when the test asserts no send occurred after the close, then the assertion holds.

### R-AFP-013 — Cancellation before the call takes the pre-stream path

WHEN the context is already cancelled at call time and the scripted request is valid, the call MUST fail on the pre-stream path with a typed error carrying the cancellation category, MUST return no usable carrier, and MUST start no producer goroutine. WHEN the request is invalid **and** the context is already cancelled, the reported failure MUST be the validation failure — the fake MUST preserve AI-20.2's ordering rather than short-circuiting on cancellation.

#### Scenarios

- **S-AFP-030** — Given a valid scripted request and a context cancelled before the call, when the call is made, then it returns a typed failure carrying the cancellation category and no usable carrier.
- **S-AFP-031** — Given that same call under `-race` with goroutines counted immediately before and after, when it returns, then no additional goroutine is live.
- **S-AFP-032** — Given a zero-value request and an already-cancelled context, when the call is made, then the failure reported is the validation failure, not the cancellation category.

---

## AI-21.6 — Request capture `[leaf]`

### R-AFP-014 — Everything a request carried is assertable after the call

WHEN a test streams a request through the fake, it MUST afterwards be able to assert on **every** field that request carried — model, message content, tools, tool choice, options, cache markers and pass-through values — through the request's own readability. Capture MUST occur exactly for a call that consumes a script — i.e., one that passes every pre-stream check. A call rejected on the pre-stream path (an invalid request, a cancelled context, or an exhausted queue) MUST NOT be captured: there is no script for it to correspond to, and the capture history's positional correspondence to the script queue (`Requests()[i]` ↔ script `i`) MUST hold exactly.

#### Scenarios

- **S-AFP-033** — Given a request populated in every field and streamed through the fake, when the test reads the captured request, then every field it asserts on equals what it sent.
- **S-AFP-034** — Given three calls made in order, when the test reads the capture history, then it holds three entries in call order, each matching its own call.
- **S-AFP-035** — Given a call rejected on the pre-stream path (invalid request, cancelled context, or exhausted queue), when the test reads the capture history afterwards, then that request is absent and the history's length and positional correspondence to consumed scripts (`Requests()[i]` ↔ script `i`) are unaffected.

### R-AFP-015 — Later caller mutation cannot corrupt recorded history

A captured request MUST be a copy or otherwise immutable. Any mutation a caller performs on its own request value, or on any slice reachable from it, **after** the call MUST NOT alter what the capture history reports.

#### Scenarios

- **S-AFP-036** — Given a request streamed through the fake, when the caller afterwards derives a modified request from its own value and mutates every slice it can reach, then the capture history still reports the original values.
- **S-AFP-037** — Given the capture history read twice, when the test mutates what the first read returned, then the second read is unaffected.

---

## AI-21.7 — Scripted reasoning `[leaf]`

### R-AFP-016 — Reasoning deltas and a round-trip token are scriptable and byte-exact

A script MUST be able to stream reasoning content — a block start, deltas, and a terminal shape carrying a round-trip token. The drained events MUST carry the token **byte-exact**, so Layer 2 can prove round-tripping without a vendor.

#### Scenarios

- **S-AFP-038** — Given a reasoning block scripted with two deltas and a terminal token, when the consumer drains it, then the deltas arrive in order with byte-exact fragments and the terminal event carries the scripted token byte-for-byte.
- **S-AFP-039** — Given a token containing bytes that are not valid text, when it is scripted and drained, then it survives unchanged and is not normalised, escaped or truncated.

### R-AFP-017 — Redacted and signature-only reasoning shapes are scriptable

A script MUST be able to express a **redacted** reasoning block and a **signature-only** block, so Layer 2 tests can exercise both without a vendor. Each shape's own construction rules MUST hold: a script that violates one MUST fail loudly at the fake rather than emit an invalid event.

#### Scenarios

- **S-AFP-040** — Given a redacted reasoning block scripted with a terminal token and no visible fragment, when the consumer drains it, then it observes the redacted shape and no reasoning text.
- **S-AFP-041** — Given a signature-only block, when the consumer drains it, then it carries the signature and no content fragment.
- **S-AFP-042** — Given a script that places a visible fragment inside a redacted block, when the fake is driven, then it fails loudly and names the violation.

### R-AFP-018 — Scripted reasoning never appears in a text event

Reasoning content scripted through the fake MUST NOT appear in any text event, in any shape — not as a delta, not merged into a text block, not as a prefix. The fake MUST enforce the same wall the real adapter is required to hold.

#### Scenarios

- **S-AFP-043** — Given a script mixing reasoning deltas and text deltas, when the consumer drains and collects every text event, then no scripted reasoning fragment and no reasoning token appears among them.
- **S-AFP-044** — Given the same stream, when the consumer collects every reasoning event, then no scripted text fragment appears among them — the wall holds in both directions.

---

## AI-21.8 — Sequential-call scripting `[leaf]`

### R-AFP-019 — Consecutive calls consume consecutive scripts, in order, exactly once each

WHEN consecutive streaming calls hit **one** fake value, they MUST consume consecutive scripts from its queue, in the order the test enqueued them, each consumed exactly once. A script MUST NOT be replayed for a later call, and the queue MUST NOT be reordered. This is the multi-turn shape every Layer 2 agent-loop test is made of.

#### Scenarios

- **S-AFP-045** — Given a fake queued with a tool-call script then a text script, when a test makes two calls and drains each, then call one yields the tool call and call two yields the text, in that order.
- **S-AFP-046** — Given a fake queued with two identical-length scripts, when both calls are drained, then each call consumed its own script and neither observed the other's events.
- **S-AFP-047** — Given a fake queued with three scripts, when the test inspects the queue after two calls, then exactly one unconsumed script remains.

### R-AFP-020 — Exhaustion fails loudly; it never hangs and never repeats

WHEN a streaming call is made against an exhausted queue, the fake MUST fail loudly and immediately. It MUST NOT hang, MUST NOT block the caller, MUST NOT return an empty stream that closes as though the call succeeded, and MUST NOT repeat the last consumed script. The failure MUST name the fake and the exhaustion; the exact mechanism is design's.

#### Scenarios

- **S-AFP-048** — Given a fake whose single script has been consumed, when a second streaming call is made, then the call fails loudly within a bounded test deadline and the failure names queue exhaustion.
- **S-AFP-049** — Given that same exhausted call, when the test observes what it returned, then it did not receive a replay of the previous script and did not receive a stream that closed as a clean empty success.
- **S-AFP-050** — Given a bounded test deadline around the exhausted call, when the deadline expires, then it was never reached — the call had already failed rather than blocked.

---

## Non-functional requirements

### NFR-AFP-A — Dependency purity

This change MUST add no module dependency. The fake MUST import only the standard library and the module's own Layer 1 package. `backend/agent/go.mod` MUST still declare zero requires, and both AI-00 import guards MUST still pass.

- **S-AFP-051** — Given the change merged, when `go.mod` is read and both import guards run, then it declares no require and both guards pass.

### NFR-AFP-B — Layer 1 is read, never modified

This change MUST NOT modify `src/ai`. The local single-purpose producer inside `ai/provider_test.go` MUST remain in place, unexported and unreplaced, per `R-AMP-013`'s own record. The AI-20 signature guard MUST still pass unchanged, and `src/ai` MUST compile and pass identically with or without this change.

- **S-AFP-052** — Given the merged diff, when a reviewer looks for an edit under `src/ai/`, then none exists, and the signature guard still passes.
- **S-AFP-053** — Given the change reverted in isolation, when `src/ai`'s tests are run, then they pass exactly as before.

### NFR-AFP-C — Determinism and no wall-clock coordination

No behaviour this spec requires MUST depend on elapsed wall-clock time for its correctness. Bounded test deadlines used to prove a call does **not** hang are permitted; a sleep used to sequence scripted events is not.

- **S-AFP-054** — Given every test this milestone adds, when a reviewer enumerates each use of a sleep or a timer, then each is a failure deadline or a documented settling step, and none is the mechanism that orders scripted events.
- **S-AFP-055** — Given the milestone's whole test set run repeatedly under `-race`, when the results are compared, then they are identical across runs.

### NFR-AFP-D — Totality, with one stated exception

No exported function or method of the fake MUST panic for any input — including a zero-value request, a nil or cancelled context, an empty script and a script queue that was never populated — **except** the loud-failure path of `R-AFP-020`, whose observable form is design's and which MUST be documented on the fake's own surface wherever it can terminate a test.

- **S-AFP-056** — Given a table of those extreme inputs, when each is passed through every exported entry point, then none panics except where `R-AFP-020` requires a loud failure.
- **S-AFP-057** — Given the fake's documentation, when a reader looks for how exhaustion behaves, then it is stated explicitly rather than discovered by a test that dies.

### NFR-AFP-E — The package documentation is reframed

`src/agenttest/doc.go` MUST be updated: the package is no longer only Layer 1's external-consumer proof, it is also an importable testing library. The documentation MUST state both roles, MUST retain the ADR 0005 Guard C sibling-layout note, and MUST state the contract-faithful-not-convenient rule so a later contributor does not simplify a physics behaviour for convenience.

- **S-AFP-058** — Given the landed package documentation, when a reader looks for the package's purpose, then it names both the proof role and the library role, retains the sibling-layout note, and states the contract-faithful rule.

### NFR-AFP-F — Evidence

Every test-list item of AI-21.1 … AI-21.8 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) from `backend/agent/` and clean `make lint`.

- **S-AFP-059** — Given `tasks.md`, when a reviewer walks the milestone's test-list items, then each carries recorded red output, recorded green output, and a refactor note.

---

## Acceptance criteria

1. A test in another package scripts start + two text deltas + complete and drains exactly those events, sequenced 1…N, terminated by close, with no network (`R-AFP-001`, `R-AFP-002`).
2. Two fakes streaming concurrently sequence independently under `-race` (`R-AFP-003`).
3. The fake has one producer goroutine, one closing site, and every send waits on cancellation (`R-AFP-004`).
4. Tool calls reconstruct to exact argument bytes; a zero-delta call is indistinguishable after reconstruction; interleaved calls keep their ordinals (`R-AFP-005` … `R-AFP-007`).
5. A terminal error of any category is scriptable in both partial-output states, and nothing follows it (`R-AFP-008`, `R-AFP-009`).
6. A stream can be held open and released, and can saturate an unread consumer, both without a wall-clock sleep in any assertion (`R-AFP-010`, `R-AFP-011`).
7. Mid-stream cancellation closes in bounded time with no send after close and a bare close on the saturated path; pre-call cancellation takes the pre-stream path with a typed error, after validation (`R-AFP-012`, `R-AFP-013`).
8. Every field a request carried is assertable after the call, and later caller mutation cannot alter recorded history (`R-AFP-014`, `R-AFP-015`).
9. Reasoning deltas, a byte-exact round-trip token, and redacted / signature-only shapes are scriptable, and none appears in a text event (`R-AFP-016` … `R-AFP-018`).
10. Consecutive calls consume consecutive scripts; an exhausted queue fails loudly, never hangs, never repeats (`R-AFP-019`, `R-AFP-020`).
11. `make test` green under `-race`, `make lint` clean, both AI-00 import guards and the AI-20 signature guard still passing, `go.mod` still zero requires (`NFR-AFP-A`, `NFR-AFP-B`, `NFR-AFP-F`).

## What this file deliberately leaves to design and to later milestones

Two items are **real design work**, deliberately not decided here: the observable form of `R-AFP-020`'s loud failure given the provider interface's fixed return shape, and the exact shape of `R-AFP-010`'s synchronization primitive. Both are without precedent in the module and MUST be settled in `design.md` rather than improvised during apply.

Vendor wire-format mocking remains AI-27's fixtures and AI-38's transcripts. The stream recording and assertion helpers are AI-22 and the conformance suite is AI-23 — siblings in this wave, separate changes, both consumers of this capability rather than parts of it.
