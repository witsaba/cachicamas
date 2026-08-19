# `agent-cost-events` Specification

> **Change**: `cachicamas-agent-cost-events` · **AG-16** (Layer 2, Wave 3), `0003:1527-1560`
> **NEW capability**, minted by this change (AG-16). This file is the normative text, promoted to `openspec/specs/agent-cost-events/spec.md` at archive. IDs `R-CST-0NN` / `S-CST-0NN`; the allocated range is `R-CST-001`…`R-CST-007` and `S-CST-001`…`S-CST-014` plus the bites `S-CST-020`, `S-CST-021`, `S-CST-022`. **The prefix `CST` was verified collision-free** (zero `[RSN]-(CST|COS|CEV|CSE)-[0-9]` matches repo-wide). **No total is stated**: a count goes silently false the moment a later milestone appends (`S-LSK-020`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && make test`.
> **Sources**: charter `0003:1527-1560`; the `cachicamas-agent-cost-events` change's proposal (Decisions 1–5), design (whose decisions **DD1–DD4 are closed** and are not re-opened here), and explore notes, archived at `openspec/changes/archive/2026-08-18-cachicamas-agent-cost-events/`.
> **Ownership boundary**: this capability owns *what a cost event asserts* — per-turn emission and its per-path rule, per-figure presence, the cumulative algebra, and the estimate/final label axis at run scope. It does **not** own `Turn`'s per-path emission contract ([`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md)), the run algorithm and its bracket rules ([`../agent-run-driver/spec.md`](../agent-run-driver/spec.md)), the cost payload *vocabulary* ([`../agent-protocol-events/spec.md`](../agent-protocol-events/spec.md), `R-APE-004`/`R-APE-005`), the retry predicate ([`../agent-retry-failover/spec.md`](../agent-retry-failover/spec.md)), or the cancellation wind-down order ([`../agent-cancellation-tree/spec.md`](../agent-cancellation-tree/spec.md)). It amends each of those through its own delta in this change's `specs/` tree.

## Purpose

AG-06 built the cost vocabulary and nothing emits it: `CostTurn` and `CostSession` are constructible, registered and validated (`cost_events.go:162-165`, `:227-230`) with **zero production call sites**, and the `agent` package never calls `completion.Usage()` once. Layer 1 counts tokens into `ai.Usage` (`usage.go:111-149`); Layer 2 discards them at the accumulator's Completion case, which reads `FinishReason()` and drops the rest (`loop.go:923-927`). A run's token spend is therefore unobservable, and two shipped specs record the debt against AG-16 by name (`agent-run-driver/spec.md:326`, `agent-retry-failover/spec.md:219`).

This capability closes G10's Layer 2 half: **Layer 1 counts; Layer 2 reports.**

## Coverage — the three charter scenarios, each mapped

Every charter Gherkin leaf traces to at least one `S-CST-0NN`. No leaf is reduced and no scenario is orphaned.

| # | Charter scenario (AG-16.1) | Lines | Owning requirement | Scenario(s) |
|---|---|---|---|---|
| 1 | "per-turn cost is exact and honest about absence" | `0003:1545-1549` | `R-CST-001` (emission), `R-CST-002` (presence) | `S-CST-001`, `S-CST-002`, `S-CST-005`, `S-CST-006`, bite `S-CST-020` |
| 2 | "cumulative counts every attempt" | `0003:1550-1553` | `R-CST-004` | `S-CST-008`, bite `S-CST-021` |
| 3 | "late usage corrects an estimate" | `0003:1554-1558` | `R-CST-005` | `S-CST-009`, `S-CST-010`, bite `S-CST-022` |

Cross-cut requirements carrying no charter leaf of their own: `R-CST-003` (the conversion, which every figure claim reads through), `R-CST-006` (the non-happy run closes, forced by `R-RUN-011` and `R-CAN-002`), `R-CST-007` (the scope fence).

## Two charter reconciliations, binding on every requirement below

Both are proposal Decision 5's recorded reconcile-or-flag duties, following the AG-06 precedent (`0003:613`). They are stated here because a reader who applies the charter's literal words to the shipped substrate would write an unbuildable test.

1. **"A retried attempt's tokens are real spend"** (`0003:1535`, `:1552`) does **not** mean a *failed* attempt's own spend is captured: `ai.Failure` carries nine fields and no usage (`provider_failure.go:320-330`), and no Layer 1 edit is in scope (`R-RUN-012`). It is reconciled to the sum-over-emitted-events rule of `R-CST-004`, which is retry-inclusive **by construction**.
2. **"Usage arriving only on the final stream update"** (`0003:1556`) is a wire-level phenomenon Layer 1 absorbs entirely: `ai.Completion` is the sole usage carrier, registered `CardinalityAtMostOne, Terminal: true` (`completion.go:92`, `ai/event.go:167-170`). Within one turn there is no earlier figure to label `Estimate`. It is reconciled to the run-scoped axis of `R-CST-005`, on the charter's own authority (`0003:1537`, which already assigns `Estimate` to the incremental running figure).

## Fixture constraint, binding on every scenario below

`agenttest.Provider.Stream` fails before producing a channel in exactly three situations (`fake_provider.go:75-77`, `:85-97`, `:99-102`), so **a `Script` cannot express an arbitrary retryable pre-stream failure** (`agent-retry-failover/spec.md:36`). Every scenario needing a retried attempt MUST use the test-local `errorProvider` wrapper precedent (`loop_test.go:1408-1421`). `agenttest` is **not** modified by this change. Scripted usage MUST be supplied through `ai.NewCompletion(finishReason, usage)`, which can carry any `ai.Usage` including the empty one.

## Requirements

### R-CST-001 — A `cost_turn` is emitted iff the turn closes non-aborted, inside its own bracket, always labelled final

For **every** turn that closes non-aborted, the system MUST emit a `cost_turn` event carrying that turn's five token figures — input, output, cache-read, cache-write and reasoning — mapped from the `ai.Usage` the turn's `ai.Completion` reported, on the sink, **inside the turn bracket the turn opened and before that bracket's `turn_end`**. The rule is stated as an iff so that neither half is inferred: a non-aborted close always emits one, and an aborted close never does.

The event's `CostLabel` MUST **always** be `Final`. Per-turn usage is complete whenever it is known at all, because `ai.Completion` is the sole, at-most-once, terminal usage carrier. The `Estimate` half of the label axis is run-scoped and belongs to `R-CST-005`; a per-turn `Estimate` is unconstructible behavior under this requirement, not merely unused.

The per-path rule, enumerated rather than left to be inferred (design DD2):

| Path | `cost_turn`? | Why |
|---|---|---|
| Normal completion, on either `Turn` call path | **Yes**, `Final`, exact figures | the turn's usage was reported |
| Provider closes with no `Completion` (`S-ATT-012`) | **Yes**, `Final`, **all five figures absent** | nothing was ever reported; a first-class `R-CST-002` absence case, **not** an error and **not** five zeros |
| Mid-stream fatal (`loop.go:396-430`) | **No** | the turn closes aborted; `Completion` is `Terminal`, so no usage exists |
| Pre-stream abort (`R-LSK-001`'s three paths) | **No** | no data source exists (`provider_failure.go:320-330` carries no usage) |
| Cancellation mid-turn (AG-14 wind-down) | **No** | routes through the fatal path; `turn_end(Aborted)` with no invented figures |

The placement is a validator obligation, not a convention: `cost_turn` is registered `PlacementTurn` (`event.go:329-331`), so `CheckStream` MUST accept the recorded cost-bearing stream **unmodified**, with `stream_check.go` **byte-unchanged**. The fix lands on the emitter, never on the validator (`R-RUN-003`, `agent-run-driver/spec.md:127`).

`Turn`'s exported signature MUST NOT change (`agent-turn-termination/spec.md:113`, `NFR-ATT-004`). The usage reaches the emission site through the turn's own accumulator, never through a returned value.

#### Scenarios

- **S-CST-001** — **Charter AG-16.1 scenario 1, exactness half.** Given a multi-turn run whose provider scripts a distinct `ai.Usage` on each turn's `ai.Completion`, every one of the five figures present and pairwise distinct across turns — cache-read, cache-write and reasoning included, not only input and output — when the run is driven and the stream is recorded, then each turn bracket carries a `cost_turn` positioned after that turn's content events and **before** its `turn_end`; each such event's five figures equal that turn's scripted `ai.Usage` figure for figure; each carries `CostLabelFinal`; and `CheckStream` accepts the recorded stream unmodified with `stream_check.go` byte-unchanged.
- **S-CST-002** — **Charter AG-16.1 scenario 1, the no-`Completion` path.** Given a provider scripted to close its stream without ever emitting an `ai.Completion` (`S-ATT-012`'s shape), when `Turn` runs and the stream is recorded, then the turn still carries a `cost_turn` before its `turn_end`; every one of its five figures **reports absence**; no figure reports a present zero; the turn is not reported as failed by the emission; and `CheckStream` accepts the stream unmodified.
- **S-CST-003** — **The aborted half of the iff.** Given, in turn, a mid-stream fatal failure, each of `R-LSK-001`'s three pre-stream failure conditions, and an interrupt fired mid-turn, when each stream is recorded, then **no** `cost_turn` appears in the aborted turn's bracket in any of them; each bracket still closes exactly as its owning requirement already requires; and `CheckStream` accepts each stream unmodified. Cross-referenced to `R-ATT-005`, `R-LSK-001` and `R-CAN-002`.
- **S-CST-004** — **The label axis at turn scope.** Given every `cost_turn` observed by `S-CST-001`, `S-CST-002` and `R-CST-004`'s retry-bearing run, when each event's label is read, then each is `Final`, and no `cost_turn` anywhere on any recorded stream in this capability's suite carries `Estimate`.

### R-CST-002 — Presence is per figure, travels beside the figures, and absence is never an invented zero

Each of the five figures a cost payload carries MUST be readable **together with a discriminator saying whether Layer 1 reported it**, mirroring Layer 1's own house idiom `TokenCount.Count() (int64, bool)` (`usage.go:63-65`). A reading of `(0, absent)` and a reading of `(0, reported)` MUST be observably different from an external test package. The paired accessors are the **required** path for judging presence: `Figures()` survives unchanged for AG-06's byte-pinned callers (`cost_events_test.go`, pinned byte-unchanged by `NFR-CST-004`), and a reader MUST NOT infer absence from it, because a bare `CostFigures` renders an unreported figure and a reported nought identically. `Figures()` MUST NOT be removed to satisfy this requirement.

**The granularity is the requirement.** `ai.Usage`'s presence is per field (`usage.go:44-47`), so a **whole-record** flag is a defect under this requirement even though it satisfies the charter's literal Given: for `ai.Usage{Input: ai.Tokens(100)}` a record-level "present" reports reasoning as `(0, reported)` — an invented zero for four absent figures, which is the exact defect "absence is reported as absence, never invented zeros" names (`0003:1535`, `:1547`).

The discriminator MUST travel **beside** `CostFigures` and MUST NOT be added inside it. That is not a stylistic preference: `S-APE-083` reflects over `reflect.TypeOf(agent.CostFigures{})` and pins exactly five fields, in that name order, each `uint64` (`cost_events_test.go:199-234`). `CostFigures` MUST be **byte-unchanged** and `cost_events_test.go` MUST be **byte-unchanged and green**. The payload wrapper widening is owned by this change's `agent-protocol-events` delta (`R-APE-004`).

The presence axis and the label axis MUST stay orthogonal. A turn whose usage is absent is still a **final** figure; absence MUST NOT be expressed as a third `CostLabel` member, and the label MUST NOT be read as evidence about presence.

#### Scenarios

- **S-CST-005** — **Charter AG-16.1 scenario 1, absence half.** Given two otherwise identical single-turn runs, one whose `ai.Completion` carries `ai.Usage{}` and one whose `ai.Completion` carries a usage whose input figure is `ai.Tokens(0)`, when both streams are recorded, then both turns emit a `cost_turn`; the first reports its input figure **absent** and the second reports it **present at zero**; the two events are observably different when read through the presence-paired surface; and the assertion reads the discriminator, **never the count alone** — two events whose counts are both `0` must not be able to satisfy it.
- **S-CST-006** — **Charter AG-16.1 scenario 1, the mixed record.** Given a turn whose `ai.Completion` carries `ai.Usage{Input: ai.Tokens(100)}` — one figure reported, four unreported — when its `cost_turn` is read, then the input figure reads `(100, reported)` and the output, cache-read, cache-write and reasoning figures each read **absent**; none of the four reads as a present zero.
- **S-CST-020** — **(bite)** RED-first. Given a scratch tree in which per-figure presence is collapsed to a single whole-record discriminator, when `S-CST-006` runs, then it **FAILS** reporting the four unreported figures as present zeros — proving the per-figure granularity is load-bearing rather than incidental, and that a record-level flag would have shipped the named defect while `S-CST-005` still passed. RED-recorded BEFORE `S-CST-006` is GREEN, then reverted.

### R-CST-003 — The `ai.Usage` → payload conversion is total, pure and presence-preserving

The mapping from `ai.Usage` to a cost payload's figures-and-presence pair MUST be **total** (defined for every `ai.Usage` value, the zero value included), **pure** (no I/O, no clock read, no emission), and **presence-preserving in both directions**: a reported value maps to a reported figure of the same magnitude, and an unreported value maps to an absent figure. `ai.Tokens(0)` MUST map to a **reported nought** and MUST NOT be conflated with absence.

A payload not built from a reported `ai.Usage` — the zero-value payload a wrong-kind accessor returns (`cost_events.go:199-202`) — MUST report every figure absent, which is the coherent reading rather than a special case.

#### Scenarios

- **S-CST-007** — Given the conversion exercised directly across a table covering the empty `ai.Usage`, a fully reported usage, a usage reporting only zeros, a mixed usage with each of the five positions reported alone, and a usage whose figures are pairwise distinct, when each case is converted, then each figure's magnitude and discriminator equal the corresponding `ai.TokenCount`'s reading; no case panics; no case requires a non-negative-value guard beyond what Layer 1 already validates (`usage.go:151-169`); and the zero-value payload reports every figure absent.

### R-CST-004 — Cumulative is the sum over every `cost_turn` emitted within the run bracket

The run-scoped cumulative figure a `cost_session` carries MUST be defined as **the sum over every `cost_turn` event emitted within that run bracket** — not as "the sum over the turns that succeeded".

**This framing is the requirement, not an implementation note.** It makes the charter's "a retried attempt's tokens are real spend" true **by construction**: each attempt of a retried logical turn is its own `Turn` invocation with its own turn bracket and its own freshly minted turn identity on the one contiguous lane (`harness.go:502-521`, `R-RTY-002`, `R-RUN-003`), so any attempt that reaches a `Completion` emits its own `cost_turn` and is therefore counted, with **no retry-awareness anywhere in the accumulator**. It is also forward-compatible: if Layer 1 ever attaches usage to a failure, that attempt begins emitting a `cost_turn` and this rule already includes it, with no spec change and no accumulator change.

The per-figure algebra MUST be, for each of the five figures independently — absence is the additive identity in value, and presence is OR:

| Running total | Contribution | Cumulative count | Cumulative discriminator |
|---|---|---|---|
| absent | absent | 0 | **absent** |
| absent | reported *n* | *n* | reported |
| reported *m* | reported *n* | *m + n* | reported |
| reported 0 | absent | 0 | **reported** (a reported nought survives) |

A figure no turn in the run ever reported MUST therefore report **absent** on the cumulative, never a fabricated `0` — the same "never invented zeros" rule as `R-CST-002`, at run scope.

The cumulative state MUST be **run-scoped**: it MUST NOT outlive one run, and it MUST NOT be carried on the harness value, which is value-form, serially reused, and carries no cross-run state (`R-CAN-002`, `R-RUN-001`'s one-run-at-a-time clause). Accumulation MUST NOT introduce a second writer to the run's event path.

#### Scenarios

- **S-CST-008** — **Charter AG-16.1 scenario 2.** Given a multi-turn run in which one logical turn fails retryably before any output and succeeds on a later attempt (driven through the `errorProvider` wrapper per the fixture constraint), with a distinct scripted `ai.Usage` on each attempt that reaches a `Completion`, when the run reaches its terminal and the stream is recorded, then the run-terminal `cost_session`'s figures equal, figure for figure, the sum computed **over the `cost_turn` events observed on that same recorded stream** — asserted as an equality against the observed events and **never against a hand-computed literal**; the retried logical turn's succeeding attempt contributes its real figures, neither dropped nor zeroed; a figure absent on every contributing event reports absent on the cumulative rather than zero; and `CheckStream` accepts the stream unmodified.
- **S-CST-021** — **(bite)** RED-first. Given a scratch tree whose accumulator skips the `cost_turn` of any logical turn that was retried, when `S-CST-008` runs, then it **FAILS** reporting a cumulative below the observed per-turn sum — proving the sum-over-emitted-events rule is enforced by the assertion rather than assumed by the framing. RED-recorded BEFORE `S-CST-008` is GREEN, then reverted.

### R-CST-005 — The estimate/final protocol is run-scoped: zero or more estimates, then exactly one final at the close

Within any harness run bracket, the `cost_session` events MUST be **zero or more carrying `CostLabelEstimate`, followed by exactly one carrying `CostLabelFinal` emitted immediately before the run-close event**. This invariant MUST hold on every run outcome (`R-CST-006`).

- A `cost_session` emitted as a running total **because the driver decided to run another logical turn** MUST carry `Estimate`: the run has not concluded and more tokens may follow.
- The run-terminal `cost_session` MUST carry `Final` and MUST correct every earlier estimate — its figures are the cumulative of `R-CST-004` at close time.

**The charter's "any earlier figure was labelled estimate" is a conditional, and the conditional is the requirement.** Where an earlier figure exists it is `Estimate`; in a run that never continued past its first logical turn none exists, and the sole `cost_session` is the `Final` one. A run that emits no `Estimate` therefore satisfies this requirement rather than violating it, and no scenario may assert an `Estimate` is always present.

Both kinds are `CardinalityAny`, so multi-emission is already stream-legal, and `cost_session` is `PlacementRun` (`event.go:332-334`), so the between-turns and immediately-before-run-close positions are both legal. `CheckStream` MUST accept the recorded stream **unmodified**, `stream_check.go` byte-unchanged.

`cost_session` is **harness-scoped**. A standalone `Turn` invocation — no harness, nil continuation — emits its `cost_turn` with nothing aggregating it and MUST emit **no** `cost_session`. That is correct behavior, stated here so it is not later read as a gap.

#### Scenarios

- **S-CST-009** — **Charter AG-16.1 scenario 3.** Given a run the driver continues past its first logical turn, when the stream is recorded, then at least one `cost_session` carrying `Estimate` appears between two turn brackets; the last `cost_session` on the stream carries `Final`, is positioned **immediately before** the run-close event, and carries figures equal to the cumulative over the stream's `cost_turn` events; every `cost_session` preceding it carries `Estimate` and none carries figures exceeding the final one on any reported figure; and `CheckStream` accepts the stream unmodified.
- **S-CST-010** — **The conditional half.** Given a run that reaches its terminal on its first logical turn, when the stream is recorded, then its sole `cost_session` carries `Final`, sits immediately before the run-close, and **no** `cost_session` carrying `Estimate` appears anywhere in the run bracket — the conditional reading is satisfied by absence, not by a fabricated estimate.
- **S-CST-011** — **Standalone `Turn` aggregates nothing.** Given `Turn` invoked directly with a zero-value `TurnOptions` and a completing script, when the stream is recorded, then it carries the turn's `cost_turn` before `turn_end` and **no** `cost_session` of either label; and `CheckStream` accepts the stream unmodified.
- **S-CST-022** — **(bite)** RED-first. Given a scratch tree in which the run-terminal `cost_session` is labelled `Estimate`, when `S-CST-009` and `S-CST-010` run, then both **FAIL** reporting a run bracket that closed without a final figure — proving the label assertion reads the terminal event's own label rather than merely counting cost events. RED-recorded BEFORE `S-CST-009` is GREEN, then reverted.

### R-CST-006 — Every run close carries the final figure, including the failed and the cancelled close

The `cost_session` carrying `Final` MUST be emitted immediately before the run-close event on **all three** run closes:

| Close | Requirement it belongs to | Why the figure is owed |
|---|---|---|
| Success | `R-RUN-002`'s terminal decision | the run concluded |
| **Failure** | `R-RUN-011` | tokens spent before the failure are **real spend**; a failed run that reports nothing is indistinguishable from a run that spent nothing |
| **Cancellation wind-down** (interrupt and shutdown) | `R-CAN-002`, `R-CAN-005` | same reason; a cancelled run's spend is not refunded |

The emission MUST be **best-effort**, mirroring the existing posture on those paths: if the payload cannot be constructed, the emission is skipped and the path's original outcome, error and event sequence are returned unchanged. A cost emission MUST NOT convert one run outcome into another, MUST NOT attach a `*Failure` to a close that carries none, and MUST NOT alter `R-RUN-011`'s no-append/no-`CloseTurn` posture — a cost event writes no transcript.

Exactly two paths MUST emit **nothing at all**: the post-shutdown refusal, which `R-CAN-005` requires to "emit no event whatsoever", and a `Run` that fails before its run-open event exists. Neither has a run bracket, so neither has a cost bracket.

#### Scenarios

- **S-CST-012** — **The failed close.** Given a run driven to `R-RUN-011`'s failure path after at least one turn has already reported usage, when the stream is recorded, then a `cost_session` carrying `Final` appears **immediately before** the run-close; its figures equal the cumulative over the `cost_turn` events observed on that same stream; the run-close still carries the failed run outcome with its non-nil typed failure exactly as `R-RUN-011` and `R-RTY-012` require; nothing was appended to the transcript by the cost emission; and `CheckStream` accepts the stream unmodified.
- **S-CST-013** — **The cancelled closes, and the two silent paths.** Given a run interrupted mid-flight after a turn has reported usage, and separately a run shut down the same way, when each stream is recorded, then each carries a `cost_session` carrying `Final` immediately before its run-close; each run-close still carries its own outcome member — interrupted or shutdown — with a **nil** `*Failure`, per `RunEnd.validate`'s failure-iff-`Failed` rule; the wind-down's enumerated order is otherwise unchanged (`R-CAN-002`); `CheckStream` accepts both streams unmodified; and when a `Run` is invoked on the harness value **after** shutdown has latched, then the consumer observes **no event whatsoever**, cost events included.

### R-CST-007 — The scope fence: token-only, no new kind, no money, no parent aggregation

This capability MUST NOT introduce any field that could carry money, currency or price, on any payload, at either scope. The token-only pin (`R-APE-004`, mechanically enforced by `S-APE-083`'s forbidden-substring scan, `cost_events_test.go:207-224`) is **strengthened** by this change, never weakened: `CostFigures` gains nothing. Money is Layer 3's, and the price table is out of scope by the charter's own line (`0003:1537`).

This capability MUST NOT register a new `EventKind`. AG-06 minted `cost_turn` and `cost_session`; AG-16 emits them. `event_descriptor.go` and `event_registry_test.go` MUST be byte-unchanged and the every-kind-constructible guard MUST pass at its committed kind count.

Aggregating a delegated or subagent run's cost into a parent run is **AG-19**'s, by the charter's own Goal line (`0003:1533`). Nothing in this capability may be written as if a parent scope exists.

#### Scenarios

- **S-CST-014** — Given the merged change, when the package's payload surface is scanned by `S-APE-083`'s existing forbidden-substring check and its reflection walk, then both pass with `cost_events_test.go` **byte-unchanged** and `CostFigures` **byte-unchanged**; when the every-kind-constructible guard runs, then it passes at its committed kind count with `event_descriptor.go` and `event_registry_test.go` byte-unchanged; and when the change's diff is taken over `backend/agent/src/ai/` and over `go.mod`/`go.sum`, then it is empty.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-CST-001** | External-package verifiability: every scenario above MUST be verifiable by `cd backend/agent && make test`, and every behavioral test MUST live in `package agent_test`. A behavior reachable only from inside the package is, for this spec, not reachable at all. `R-CST-003`'s conversion MAY additionally be exercised through an internal table test, provided every behavioral claim above is also observable externally. |
| **NFR-CST-002** | Determinism and race cleanliness: every test MUST be hermetic and MUST pass under `-race`. Synchronization MUST be by channel reads, channel closes and `agenttest.Gate`. **No test may synchronize by sleep, timeout or wall-clock ordering** (`NFR-RUN-002`, `NFR-CAN-002`, `NFR-RTY-002`). |
| **NFR-CST-003** | Ambient authority and boundaries: production sources added by this change MUST NOT import `os`, process execution, `syscall` or legacy I/O; the ambient-authority and import-boundary guards MUST pass with **zero** change. |
| **NFR-CST-004** | Substrate: every file named by `R-LSK-004` MUST be byte-unchanged **except** `cost_events.go`, whose release is recorded by this change's `agent-loop-skeleton` delta; `cost_events_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `turn_events.go`, `failure.go`, `run_events.go`, `history.go`, `go.mod`, `go.sum` and every file under `backend/agent/src/ai/` MUST be byte-unchanged. Both substrate filters MUST carry an identical exact-filename entry set, one entry per file this change introduces, with no wildcard, prefix or directory pattern. |
| **NFR-CST-005** | Existing-test amendments are signed off, never quiet. Adding a `cost_turn` to every non-aborted turn changes what a closed-sequence assertion observes. Every test file edited to accommodate it MUST be enumerated in `apply-progress.md` with its reason, and each MUST be tied to the delta that amends the requirement it implements. A test edited to accommodate a new event is an **amendment**, not a fix. |
| **NFR-CST-006** | Review budget: this change ships as a **single** pull request under a pre-authorised `size:exception` extended beyond the 1000-line budget. The pull-request description MUST state why the change does not fit the default budget. If `sdd-tasks` forecasts beyond that extension, the slicing boundary is the proposal's Approach ordering — **U1** capture + conversion (inert, no emission), **U2** `cost_turn` emission, **U3** cumulative and the session axis. |

## Explicit non-requirements — what this spec does NOT claim

Stated so that no test, guard or acceptance line is written as if AG-16 closes more than it does.

| Not claimed | Owner and why the deferral is safe |
|---|---|
| Money, currency, price, or a price table on any payload | **Layer 3 / CO-18.** Charter out-of-scope (`0003:1537`); `R-APE-004` states it; `S-APE-083` enforces it mechanically. AG-16 strengthens the pin by adding zero fields to `CostFigures` |
| A **failed** attempt's own token spend as a distinct figure | **Unbuildable, not deferred by choice.** `ai.Failure` carries no usage (`provider_failure.go:320-330`), and Layer 1 is never edited (`R-RUN-012`). `R-CST-004`'s sum-over-emitted-events rule already includes such a figure the day Layer 1 provides one |
| Delegated / subagent cost aggregation into a parent run | **AG-19**, by the charter's own Goal line (`0003:1533`). No subagent tool ships in v1 (`0003:1794`) |
| Mid-stream incremental cost display | **Frontend / out of scope** (`0003:1537`). The charter itself rules that the estimate-labelled event at minimum covers it, which is `R-CST-005`'s own evidence |
| A per-turn `Estimate` figure | **Unbuildable at this layer.** Layer 1 folds every wire-chunk usage update into one terminal `Completion` (`openaicompat/stream_state.go:328`, `ai/event.go:167-170`), so no earlier per-turn figure exists to label |
| A third `CostLabel` member for absence | **Never.** Presence and estimate/final are orthogonal axes (`R-CST-002`); overloading one enum with both is a category error |
| A new `EventKind`, `TurnOutcome` or `RunOutcome` | **Not this milestone**, and forbidden under this change (`R-CST-007`) |
| Re-opening AG-15's retry predicate, gates G0–G5 | **CLOSED by AG-15.** AG-16 reads the attempt loop's outputs and touches no gate (`retry_policy.go:111-126`) |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2.** Layer 1 is consumed, never edited (`R-RUN-012`) |
| Widening `agenttest` | **Not this milestone.** The retry fixture is test-local (the fixture constraint above); `agenttest` is byte-unchanged |

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active.

- All three charter leaves are behavior, so every scenario is RED-first.
- The three bites `S-CST-020`, `S-CST-021` and `S-CST-022` MUST each be RED-recorded with failing output in `apply-progress.md` **before** their partner scenario is GREEN, then reverted.
- `R-CST-002` closes **only on `S-CST-020`**: a whole-record presence flag passes `S-CST-005` while shipping the exact defect the acceptance criterion names, so a green absence scenario proves nothing about granularity without the bite.
- `R-CST-004` closes **only on `S-CST-021`**: the sum-over-emitted-events framing makes the correct implementation the natural one, which is precisely why a passing equality proves nothing until the bite shows it would catch a dropped contribution.
- `R-CST-005` closes **only on `S-CST-022`**.
- `cost_events_test.go` MUST pass with its source **byte-unchanged**, and `sdd-verify` MUST re-check that against the shipped tree rather than trusting this sentence.
- Assertions about absence MUST read the presence discriminator. A scenario that distinguishes two events by their counts alone is vacuous under `R-CST-002` and MUST be rewritten, not re-run.
- Cumulative MUST be asserted as an equality against the events observed on the same recorded stream, never against a hand-computed literal — this repository's known count-assertion drift class.

## Acceptance criteria

1. Every `S-CST-001`…`S-CST-014` has recorded evidence; all three bites are RED-recorded before their GREEN.
2. All **three** charter Gherkin scenarios (`0003:1545-1549`, `:1550-1553`, `:1554-1558`) are mapped in the Coverage table and closed; none is reduced.
3. `cd backend/agent && make test` green under `-race`; `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check` all clean (`vuln-check` is **not** in `make all`).
4. `CheckStream` accepts the multi-turn, retry-inclusive, cost-bearing stream **unmodified**, with `stream_check.go` byte-unchanged.
5. `CostFigures` and `cost_events_test.go` are **byte-unchanged**; `S-APE-083` is green.
6. `Turn`'s exported signature is unchanged, verified by reading the shipped signature (`NFR-ATT-004`).
7. Every test edited to accommodate the new events is an enumerated, signed-off amendment recorded in `apply-progress.md`, each tied to the delta amending its owning requirement (`NFR-CST-005`).
8. Both substrate filters carry an identical exact-filename entry set, one entry per file this change introduces.
9. All **six** spec deltas beside this file are written, and every line they cite is re-read against the shipped change by `sdd-verify`.
10. Decision 5's reconciliation note exists on doc 0003's AG-16 charter, follows the AG-06 template (`0003:613`), and names **both** reconciliations of the section above.
11. doc 0003's AG-16 checklist is ticked, R-16/G10 back-annotated, and the milestone counter bumped to 16/24.

## Traceability

| Requirement | Charter node | Primary evidence |
|---|---|---|
| `R-CST-001` | AG-16.1 sc.1 | per-turn figures equal the scripted usage; placement inside the bracket; `CheckStream` clean |
| `R-CST-002` | AG-16.1 sc.1 | absent vs reported-nought events differ; mixed record + granularity bite |
| `R-CST-003` | cross-cut | conversion table over the empty, full, zero-valued and mixed usages |
| `R-CST-004` | AG-16.1 sc.2 | equality against the observed `cost_turn` events on the same stream + skip bite |
| `R-CST-005` | AG-16.1 sc.3 | estimates then exactly one final at the close + label bite |
| `R-CST-006` | cross-cut (DD4) | final figure on the failed and both cancelled closes; nothing after shutdown |
| `R-CST-007` | cross-cut (Decision 1) | token-only scan and kind count pass byte-unchanged |
