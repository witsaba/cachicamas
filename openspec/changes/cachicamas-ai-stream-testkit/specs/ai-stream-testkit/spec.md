# Spec — the stream test kit

> **Change**: `cachicamas-ai-stream-testkit` · **Milestone**: AI-22 — Build stream recording and assertion helpers (Wave 3 "Prove", the wave's **second** milestone)
> **Nodes**: AI-22.1 `[leaf]` timeout-safe drain and record · AI-22.2 `[leaf]` readable event diffs · AI-22.3 `[leaf]` ordering and gap assertions · AI-22.4 `[decision]` leak-detection mechanism · AI-22.5 `[leaf]` carrier view
> **Capability**: `ai-stream-testkit` — **new**. Promoted to `openspec/specs/ai-stream-testkit/spec.md` at archive.
> **Predecessor**: [`proposal.md`](../../proposal.md) · Engram `sdd/cachicamas-ai-stream-testkit/proposal` (obs #2394), `sdd/cachicamas-ai-stream-testkit/explore` (obs #2393)
> **Requirement IDs**: `R-STK-0NN` · **Scenario IDs**: `S-STK-0NN` — prefix re-verified unused across `openspec/specs/` and `openspec/changes/` at spec time (siblings in use: `AFP` AI-21, `AIP`, `AMP`, `AMR`, `AIE`, `ARC`, `ACP`, `ATD`, `REX`, `WSY`). AI-23 must not reuse `STK`.
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Vocabulary owned by this milestone**: [`ai-contract-vocabulary`](../../../../specs/ai-contract-vocabulary/spec.md) `V-PRV-11` **stream test kit** (AI-22) and `V-STR-22` **carrier view** (appended by AI-02.1, ergonomics delegated to AI-22.5). Both terms are used as the register defines them; neither is renamed and no alternate name is coined.
> **Binding predecessors, cited by identifier and never modified**:
> [`ai-model-provider`](../../../../specs/ai-model-provider/spec.md) (AI-20) — `R-AMP-009` … `R-AMP-012` mid-stream physics, AI-20.4's signature guard;
> [`ai-stream-lifecycle`](../../../../specs/ai-stream-lifecycle/spec.md) — § 3 carrier, § 4 ownership, § 5 cancellation (the abandoned-never-cancelled path it rules untestable), § 6 buffering, § 7 failure delivery;
> [`ai-event-envelope`](../../../../specs/ai-event-envelope/spec.md) (AI-14) — the envelope, its kind vocabulary, `CheckStream`'s ordering invariants (AI-14.4) and per-stream sequencing;
> [`ai-fake-provider`](../../cachicamas-ai-fake-provider/specs/ai-fake-provider/spec.md) (AI-21, **not yet archived** — read from its active change folder) — the producer these helpers are proved against
> **Depends on**: AI-21 (shipped on this branch) · **Blocks**: AI-23, AI-33, doc 0003's hardening wave
> **Sources**: [doc 0002 — Layer 1 task graph](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) §§ AI-22.1 … AI-22.5 · design D10 (`CheckStream` defers contiguity to AI-22.3) · G13 (carrier ergonomics are AI-22.5)

---

## Purpose

Wave 3 has a producer and no concise way to assert against it. The same timeout-safe drain loop is already written twice — once inside `ai`, once inside `agenttest` — because no importable version exists, and AI-23 would write it a third time. Nothing anywhere asserts sequence **contiguity**: `ai.CheckStream` disclaims it in its own doc comment and names AI-22.3 as its owner.

This spec constrains what the `V-PRV-11` stream test kit MUST guarantee: that a broken producer fails on a **deadline** rather than hanging the suite, that a failed assertion is **readable** rather than a wall of bytes, and that every helper asserts something a Layer 1 contract already states. It is deliberately not a general-purpose testing framework — that is the charter's stated non-goal.

## Delta status against existing specs

`openspec/specs/` was re-read at spec time. It carries no test-kit, recorder, differ or leak-detection capability: `ai-stream-testkit` is **new**, so this is a full capability spec with **no MODIFIED, no REMOVED and no RENAMED requirement**. `ai-model-provider`, `ai-stream-lifecycle`, `ai-event-envelope`, `ai-contract-vocabulary` and the sibling `ai-fake-provider` are read and **cited by identifier, never modified**.

**Explicitly not absorbed here**, and recorded so it reads as decided rather than forgotten:

- **W1/W2, the two Wave-2 carryovers AI-21 parked** (the `CheckEmit` rule 4 failure-path gap, the missing redacting `GoString()` on the failure payload). Still unassigned; doc 0002 assigns neither to AI-22.
- **Migration of the two existing ad hoc drain helpers** (`requireClosedWithin` in `ai/provider_test.go`, `drainFake` in `agenttest/fake_text_test.go`). This change is **additive tooling, not a refactor** of already-shipped, already-verified AI-21 and AI-20 test files. Both local helpers stay exactly where they are. Migration is a later tidy-up.
- **A third-party leak detector.** See `R-STK-009`: rejected here as this change's own recorded decision, not forbidden forever.

## Definitions used by this spec

- **The kit** — the exported helpers this milestone ships in `src/agenttest/`, importable from any package in the module.
- **A recording** — the finite, ordered `[]Event` a drain produced, together with how the stream terminated. Immutable once produced.
- **A bounded deadline** — a test-supplied maximum wait after which the helper fails the test with an attributable message. It is a **failure** deadline, never a coordination mechanism.
- **First divergence** — the lowest index at which two recordings differ, by kind, by sequence, or by payload.
- **A bounded payload summary** — a rendering of an event's payload capped to a fixed length with an explicit elision marker. Never the payload verbatim.
- **A scenario under leak test** — a closure the caller supplies that performs one complete stream interaction (call, consume or abandon, terminate), repeated N times by the leak helper.
- **The carrier view** (`V-STR-22`) — an iteration shape over a stream the consumer already holds. It never owns the stream and never closes it.

---

## AI-22.1 — Timeout-safe drain and record `[leaf]`

### R-STK-001 — Draining is bounded by a deadline and never hangs

The kit MUST expose a drain helper that receives from a stream until the stream closes **or** a caller-supplied deadline expires, whichever happens first. On deadline expiry it MUST fail the test with a message naming the deadline and how many events were received before it expired. It MUST NOT hang, MUST NOT block past the deadline, and MUST NOT silently return a partial recording as though the stream had closed.

#### Scenarios

- **S-STK-001** — Given a producer that emits two events and never closes, when a test drains it with a short deadline, then the helper fails within that deadline and the failure names the deadline and the two events received.
- **S-STK-002** — Given a fully scripted stream that closes normally, when a test drains it with a generous deadline, then the helper returns the complete recording and the deadline is never reached.
- **S-STK-003** — Given a stream that closes bare after cancellation with events undelivered, when a test drains it, then the helper reports normal closure with the events actually delivered — a bare close is a close, not a deadline failure.

### R-STK-002 — One recording backs many assertions without re-draining

A recording MUST be reusable: every kit assertion MUST accept an already-produced recording rather than a live stream, so several assertions run over **one** drain. The recording MUST preserve receive order exactly. Reading a recording MUST NOT consume, mutate or reorder it, and a caller mutating what a read returned MUST NOT alter the recording.

#### Scenarios

- **S-STK-004** — Given one recording of a four-event stream, when three different kit assertions run over it in turn, then all three observe the same four events in the same order and no re-drain occurred.
- **S-STK-005** — Given a recording read twice, when the caller mutates the slice the first read returned, then the second read is unaffected.
- **S-STK-006** — Given a recording of a scripted stream, when its events are compared against the script, then they appear in scripted order with none added, reordered, coalesced or omitted.

---

## AI-22.2 — Readable event diffs `[leaf]`

### R-STK-003 — A difference is reported as the FIRST divergence, located precisely

WHEN two recordings differ, the kit MUST report the **first** divergence — the lowest differing index — and MUST name that index, the event kind on each side, and the sequence number on each side. It MUST NOT report only "not equal", MUST NOT report an arbitrary or last divergence, and MUST NOT dump both recordings in full as its primary output. WHEN the recordings differ only in length, the report MUST name the index at which one ended and what the other still held.

#### Scenarios

- **S-STK-007** — Given two four-event recordings differing at index 2, when they are diffed, then the failure names index 2 and both sides' kind and sequence, and does not name index 3 as the divergence.
- **S-STK-008** — Given two recordings that are element-wise equal, when they are diffed, then the diff reports equality and produces no failure.
- **S-STK-009** — Given a three-event recording and a five-event recording sharing their first three events, when they are diffed, then the report names index 3 as where the shorter ended and names the two extra events' kinds.

### R-STK-004 — Payload rendering is bounded, and no event kind renders as nothing

Any payload the diff renders MUST be bounded to a fixed maximum length with an explicit elision marker when truncated. The kit MUST NOT print payload bytes verbatim and MUST NOT offer an unbounded rendering. Every registered event kind MUST have a summary; a newly registered kind MUST NOT be able to reach the differ and render as empty, unknown or unlabelled — the exhaustiveness MUST be enforced mechanically against the kind registry, not by reviewer vigilance.

#### Scenarios

- **S-STK-010** — Given an event whose payload greatly exceeds the cap, when it is summarised, then the rendering is at most the cap and carries the elision marker.
- **S-STK-011** — Given one event of every registered kind, when each is summarised, then each produces a non-empty rendering that names its kind, and none renders as unknown.
- **S-STK-012** — Given a hypothetical new event kind added to the registry without a summary, when the kit's exhaustiveness check runs, then it fails and names the unsummarised kind.

---

## AI-22.3 — Ordering and gap assertions `[leaf]`

### R-STK-005 — Ordering is delegated to the shipped checker, never reimplemented

Kind ordering, block start → delta → end ordering, terminal exclusivity and terminal placement MUST be asserted by **delegating to `ai.CheckStream`** (AI-14.4). The kit MUST NOT reimplement those invariants. A violation the shipped checker reports MUST surface through the kit's failure with the checker's own verdict preserved, so a future descriptor change is absorbed in exactly one place.

#### Scenarios

- **S-STK-013** — Given a recording carrying a terminal event followed by a text delta, when the kit's ordering assertion runs, then it fails and the failure carries the shipped checker's verdict for that violation.
- **S-STK-014** — Given a well-formed recording, when the ordering assertion runs, then it passes and reports no violation.
- **S-STK-015** — Given the landed kit source, when a reviewer looks for a re-implementation of the ordering invariants, then none exists — the assertion calls the shipped checker.

### R-STK-006 — Sequence contiguity is asserted, and a gap is named precisely

The kit MUST assert that a recording's sequence numbers start at **1** and increase by exactly one with no gap and no repeat. This check is new: the shipped checker deliberately does not perform it. WHEN a gap exists, the failure MUST name the **missing** sequence number and the two events it falls between (their indices and sequence numbers). WHEN a sequence repeats or decreases, the failure MUST name the offending index and both sequence values.

#### Scenarios

- **S-STK-016** — Given a recording sequenced 1, 2, 3, 4, when the contiguity assertion runs, then it passes.
- **S-STK-017** — Given a recording sequenced 1, 2, 4, when the assertion runs, then it fails naming missing sequence 3 and the two neighbouring events carrying 2 and 4.
- **S-STK-018** — Given a recording whose first event carries sequence 2, when the assertion runs, then it fails naming the start-at-1 violation rather than reporting a mid-stream gap.
- **S-STK-019** — Given a recording sequenced 1, 2, 2, when the assertion runs, then it fails naming the repeated sequence and its index.

---

## AI-22.4 — Leak-detection mechanism `[decision]`

### R-STK-007 — Leak detection is opt-in and amplitude-based, never an implicit global check

The kit MUST expose leak detection as an **explicit, opt-in** helper the caller wraps a scenario in. It MUST NOT be applied automatically to every stream test, MUST NOT be installed as an implicit package-level or suite-level hook, and MUST NOT change the behaviour of a test that does not call it. Detection MUST work by repeating the caller's scenario a stated number of times and comparing live-goroutine counts before and after, so a per-iteration leak grows with the repeat count while background jitter does not. A single before/after snapshot MUST NOT be the mechanism.

#### Scenarios

- **S-STK-020** — Given a scenario that leaks one goroutine per call, when it is wrapped in the leak helper at N repetitions, then the helper fails and its message names the observed growth against the repeat count.
- **S-STK-021** — Given a scenario that leaks nothing, when it is wrapped at the same repeat count, then the helper passes and reports no growth beyond its stated tolerance.
- **S-STK-022** — Given an existing stream test that does not call the leak helper, when the kit is merged, then that test's behaviour, runtime and outcome are unchanged.

### R-STK-008 — *(non-negotiable)* The leak helper is serial-only, and says so on its own surface

A test that uses the leak helper MUST NOT call `t.Parallel()`. The helper's own documentation MUST state this incompatibility explicitly and state **why**: the live-goroutine count it reads is process-wide, so it cannot distinguish a sibling parallel test's goroutines from the goroutines of the scenario under test. The kit MUST NOT attempt to attribute goroutines to a specific test, and MUST NOT paper over the problem by widening its tolerance band until the check stops detecting real leaks. This resolves the open `t.Parallel()` question by **scoping around** concurrent goroutine attribution rather than claiming to solve it.

#### Scenarios

- **S-STK-023** — Given the leak helper's documentation, when a reader looks for its concurrency behaviour, then it states that a test using it MUST NOT call `t.Parallel()` and states the process-wide-count reason.
- **S-STK-024** — Given every test this milestone adds that uses the leak helper, when a reviewer enumerates them, then none calls `t.Parallel()`.
- **S-STK-025** — Given the helper's stated tolerance, when a reviewer checks it against the repeat count, then the tolerance still fails a scenario leaking one goroutine per iteration — it was not widened into uselessness.

### R-STK-009 — The mechanism decision, including the rejected alternative, is recorded

This node is a `[decision]`. The landed change MUST record that a hand-rolled, standard-library-only mechanism was chosen and that a third-party leak detector was **rejected for this change**, with the reason stated: a new top-level dependency requires its own ADR, and the package is documented as dependency-free. The change MUST add no module dependency. The rejection MUST be recorded as reversible — a later milestone MAY revisit it by writing that ADR.

#### Scenarios

- **S-STK-026** — Given the merged change, when the module's dependency declaration is read, then it is unchanged and declares no new requirement.
- **S-STK-027** — Given the landed decision record, when a reader looks for why no third-party detector was adopted, then the alternative, its ADR trigger and the dependency-free pin are all stated, and the rejection is scoped to this change rather than declared permanent.

### R-STK-010 — Leak assertions cover the cancellation and abandoned-then-cancelled paths only

This milestone MUST carry leak assertions on the **cancellation** path and on the **abandoned-then-cancelled** path — a consumer that stops reading and whose caller then cancels. The **abandoned-never-cancelled** path MUST NOT be asserted here, and the narrowing MUST be recorded: `ai-stream-lifecycle` § 5 rules that path untestable because an unreached goroutine state cannot be observed deterministically. Doc 0002's charter phrase "the abandoned-consumer and cancellation paths" is narrowed accordingly, deliberately and on the record.

#### Scenarios

- **S-STK-028** — Given a stream cancelled mid-consumption, when the scenario is repeated under the leak helper, then no goroutine growth beyond tolerance is observed.
- **S-STK-029** — Given a consumer that stops reading and a caller that then cancels, when the scenario is repeated under the leak helper, then no goroutine growth beyond tolerance is observed.
- **S-STK-030** — Given the landed spec and change record, when a reader looks for the abandoned-never-cancelled path, then it is stated as out of scope with the § 5 untestability reason cited, rather than being silently absent.

---

## AI-22.5 — Carrier view `[leaf]`

### R-STK-011 — The view is a view: it never owns and never closes the stream

The kit MUST offer a `V-STR-22` carrier view — an iteration shape over a stream the consumer already holds. It MUST NOT close the stream, MUST NOT take ownership of it, and MUST NOT become a second contract: the provider boundary keeps speaking the decided carrier. The view MUST be usable over a stream the caller obtained itself, and abandoning the view MUST leave the stream in the same state the underlying contract puts it in.

#### Scenarios

- **S-STK-031** — Given a stream and a view over it, when iteration finishes, then the view performed no close — the stream's own producer closed it exactly once.
- **S-STK-032** — Given a view abandoned before the stream ends, when the caller then cancels, then the stream terminates exactly as it does without a view interposed.

### R-STK-012 — A terminal error is surfaced after the loop, and cancellation is respected during it

An iteration-shaped loop over a stream MUST surface the stream's terminal error to the caller **after** the loop ends, so a terminal failure cannot be silently swallowed by a loop that simply finished. Iteration MUST respect context cancellation: a cancelled context MUST end iteration promptly rather than blocking. A stream that completes normally MUST surface no error after the loop.

#### Scenarios

- **S-STK-033** — Given a stream ending in a terminal error, when a caller iterates it to the end and then inspects the view's error, then the terminal failure is reported with its category intact.
- **S-STK-034** — Given a stream that completes normally, when a caller iterates it to the end and inspects the view's error, then none is reported.
- **S-STK-035** — Given a mid-iteration cancellation, when the caller waits with a bounded deadline, then iteration ends before the deadline rather than blocking.

### R-STK-013 — The view lives outside the provider package and the signature guard passes unmodified

The carrier view MUST live in the test-kit package, not in the provider's own package. AI-20.4's signature guard MUST pass **unmodified** — no edit to the guard and no edit to the file it parses. The provider interface MUST still declare exactly its one decided method with its decided parameters and results.

#### Scenarios

- **S-STK-036** — Given the merged diff, when a reviewer looks for an edit to the provider source file or to the signature guard, then none exists.
- **S-STK-037** — Given the merged change, when the signature guard runs, then it passes and still observes the single decided method with its decided carrier result.

---

## Non-functional requirements

### NFR-STK-A — Dependency purity

This change MUST add no module dependency. The kit MUST import only the standard library and the module's own Layer 1 package. Both AI-00 import guards MUST still pass and `backend/agent/go.mod` MUST still declare zero requires.

- **S-STK-038** — Given the change merged, when `go.mod` is read and both import guards run, then it declares no require and both guards pass.

### NFR-STK-B — Layer 1 is read, never modified

This change MUST NOT modify `src/ai`. The shipped checker, the sequence stamper and the provider interface are read-only dependencies. `src/ai` MUST compile and pass identically with or without this change.

- **S-STK-039** — Given the merged diff, when a reviewer looks for an edit under `src/ai/`, then none exists.
- **S-STK-040** — Given the change reverted in isolation, when `src/ai`'s tests and AI-21's tests are run, then they pass exactly as before.

### NFR-STK-C — Existing ad hoc helpers are left in place

The two existing local drain helpers MUST remain unchanged and in their current packages. This change MUST NOT rewrite an AI-20 or AI-21 test file to consume the new kit. Adopting the kit in those files is a later change with its own review.

- **S-STK-041** — Given the merged diff, when a reviewer looks for edits to `ai/provider_test.go` or to AI-21's `fake_*_test.go` files, then none exists and both local helpers still compile and run.

### NFR-STK-D — Every helper cites the contract it asserts

This change MUST NOT grow into a general-purpose testing framework. Every exported helper MUST document the Layer 1 requirement identifier it asserts. A helper that asserts nothing a Layer 1 contract states MUST NOT ship in this milestone.

- **S-STK-042** — Given each exported helper's documentation, when a reader looks for what it asserts, then each names a Layer 1 requirement identifier.

### NFR-STK-E — Determinism and totality

No behaviour this spec requires MUST depend on elapsed wall-clock time for correctness. Bounded deadlines proving a call does **not** hang are permitted; a sleep used to order events is not. No exported helper MUST panic for any input — including a nil stream, an empty recording, a zero deadline and a recording of length one — and MUST instead fail the test with an attributable message.

- **S-STK-043** — Given the milestone's whole test set run repeatedly under `-race`, when results are compared, then they are identical across runs.
- **S-STK-044** — Given a table of extreme inputs passed through every exported entry point, when each runs, then none panics and each failure names the helper and the offending input.

### NFR-STK-F — The package documentation names the kit

`src/agenttest/doc.go` MUST name the stream test kit alongside the fake, MUST retain AI-21's existing framing, and MUST state the dependency-free pin that `R-STK-009` depends on.

- **S-STK-045** — Given the landed package documentation, when a reader looks for the package's contents, then it names both the fake and the test kit and retains the dependency-free pin.

### NFR-STK-G — Evidence

Every test-list item of AI-22.1 … AI-22.5 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) from `backend/agent/` and clean `make lint`.

- **S-STK-046** — Given `tasks.md`, when a reviewer walks the milestone's test-list items, then each carries recorded red output, recorded green output, and a refactor note.

---

## Acceptance criteria

1. A producer that never closes fails the drain helper on a deadline; the run never hangs (`R-STK-001`).
2. One recording backs several assertions without re-draining, order preserved and reads non-destructive (`R-STK-002`).
3. Two differing recordings produce a failure naming the **first** divergence by index, kind and sequence, with payload bounded and every registered kind summarised (`R-STK-003`, `R-STK-004`).
4. Ordering is delegated to the shipped checker; 1-based contiguity is new, and a seeded gap names the missing sequence and its two neighbours (`R-STK-005`, `R-STK-006`).
5. Leak detection is opt-in, amplitude-based, documented serial-only with its `t.Parallel()` incompatibility and reason, with the third-party alternative recorded as rejected and reversible (`R-STK-007` … `R-STK-009`).
6. Cancellation and abandoned-then-cancelled paths carry leak assertions; the abandoned-never-cancelled narrowing is recorded with its reason (`R-STK-010`).
7. The carrier view never closes the stream, surfaces the terminal error after the loop, respects cancellation, and AI-20.4's signature guard passes **unmodified** (`R-STK-011` … `R-STK-013`).
8. `make test` green under `-race`, `make lint` clean, AI-00 import guards passing, `go.mod` unchanged, `src/ai` and AI-21's test files untouched (`NFR-STK-A` … `NFR-STK-C`, `NFR-STK-G`).

## What this file deliberately leaves to design

Three items are **real design work**, deliberately not decided here:

1. The exact repeat count and tolerance band of `R-STK-007`'s leak helper — the *shape* (amplitude-based, opt-in, serial-only) is fixed by `R-STK-007` and `R-STK-008`; the numbers are design's.
2. The concrete mechanism enforcing `R-STK-004`'s per-kind exhaustiveness against the kind registry.
3. The concrete signature of `R-STK-011`'s carrier view, subject to the fixed constraint that it lives outside the provider package.

The conformance suite that consumes these helpers is AI-23 — a sibling in this wave, a separate change, a consumer of this capability rather than part of it.
