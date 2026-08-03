# Spec — the provider conformance suite

> **Milestone**: AI-23 — Create the provider conformance suite (Wave 3 "Prove", the wave's **final** milestone) · **Nodes** (8 leaves): AI-23.1 pluggable skeleton · AI-23.2 text + lifecycle · AI-23.3 tool calls · AI-23.4 terminal + error · AI-23.5 cancellation + closure · AI-23.7 redaction · AI-23.8 optional capabilities · AI-23.6 capability record
> **Introduced by**: `openspec/changes/archive/2026-08-03-cachicamas-ai-conformance-suite/`, merged to `main` in PR #107 (commit `0d5fd91`)
> **Status**: **live** — this file carries the contract; later milestones cite it and amend it
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/agenttest/` (Layer 1 testing infrastructure)
> **Closes**: doc 0002's **AI-23** milestone in full — the wave's final deliverable, and the artifact AI-24 (first vendor adapter, Wave 4) and AI-38 (transcript replay) are judged against
> **Requirement IDs**: `R-CNF-0NN` · **Scenario IDs**: `S-CNF-0NN`
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Binding predecessors, cited by identifier and never modified**:
> [`ai-minimum-capabilities`](../ai-minimum-capabilities/spec.md) (AI-03) — § 5 `CAP-R-01…05`, § 6 `CAP-O-01…03`, § 9 discovery, § 10 the record and its four-value outcome set, § 11 the marking rule, § 12 what AI-23 inherits, § 13 standing rules;
> [`ai-model-provider`](../ai-model-provider/spec.md) (AI-20) — `R-AMP-009` … `R-AMP-012` mid-stream physics, AI-20.3 saturated drop, AI-20.4 signature guard, `R-AMP-017` the single askable optional contract;
> [`ai-stream-lifecycle`](../ai-stream-lifecycle/spec.md) — § 4 ownership, § 5 cancellation, § 6 buffering, § 7 failure delivery;
> [`ai-event-envelope`](../ai-event-envelope/spec.md) (AI-14) — the envelope, the kind vocabulary, `CheckStream`'s ordering invariants;
> [`ai-fake-provider`](../ai-fake-provider/spec.md) (AI-21) — `Script`/`Step`/`Emit`/`Hold`/`Gate`, the suite's scenario language and its first subject;
> [`ai-stream-testkit`](../ai-stream-testkit/spec.md) (AI-22) — `R-STK-001` drain, `R-STK-003` diff, `R-STK-005`/`R-STK-006` ordering and contiguity, `R-STK-007`/`R-STK-008` leak detection
> **Depends on**: AI-03, AI-19, AI-20, AI-21, AI-22 (all shipped) · **Blocks**: AI-24, AI-38
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) §§ AI-23.1 … AI-23.8

---

## Purpose

Every Layer 1 rule today is proven only against the `ai` package's own tests. Nothing states what a **concrete adapter** must do, so AI-24's first vendor adapter would invent its own assertions and AI-38 would have no artifact to assert against.

This spec constrains what the suite MUST guarantee: that any provider factory plugs in and runs the whole case set with **zero copied assertions**, that every case is marked required or optional by AI-03 § 11 **alone**, and that the emitted capability record is **total** over both closed lists so an omission is a defect in the run rather than a silent pass. It is not a general-purpose test framework and it judges no vendor.

## Status — this file is the canonical home of the contract

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The conformance suite's eighteen requirements — the factory seam, the declared-capability tri-state, the mechanical marking rule, and the total eight-entry record — therefore live here, not only as a pointer into the archive. The archived change folder at [`openspec/changes/archive/2026-08-03-cachicamas-ai-conformance-suite/`](../../changes/archive/2026-08-03-cachicamas-ai-conformance-suite/) is the historical record of how AI-23 was explored, proposed, designed, applied and verified — including the `Factory` bool-to-`*bool` correction caught before apply (see below).

**`R-CNF-002` and `R-CNF-017`/`R-CNF-018` carry the most weight against erosion.** `R-CNF-002` is what makes every optional capability askable even when only one (`CAP-O-02`) has a real discovery seam; weakening the tri-state back to a bare bool silently reintroduces the exact defect caught during this wave's own `sdd-tasks` phase. `R-CNF-017`/`R-CNF-018` are what make the capability record trustworthy as AI-24's and AI-38's judge — a verdict rule that lets `not exercised` read as a pass would certify an adapter the suite never actually exercised.

## Delta status against existing specs

`openspec/specs/` was re-read at spec time and carries no conformance-suite, factory-seam or capability-record capability: `ai-provider-conformance-suite` is **new**, so this is a full capability spec with **no MODIFIED, no REMOVED and no RENAMED requirement**. The binding predecessors above are read and **cited by identifier, never modified** — including `ai-completion-metadata`, whose conditional delta the proposal reserved: risk **R3 is resolved without one** (`NFR-CNF-B`, `R-CNF-016`).

## Definitions used by this spec

- **The suite** — the exported conformance case set this milestone ships, runnable against any subject through a factory.
- **A subject** — the provider value under conformance test, together with the factory that builds it.
- **The factory** — the caller-supplied seam that turns a suite-authored scenario into a live provider, plus that subject's declared capability expectation.
- **A capability expectation** — which optional capabilities (`CAP-O-01…03`) the subject's owner states it offers. It is an input to the run, never a result of it.
- **A case** — one independently verifiable conformance assertion, keyed by the single capability it exercises (AI-03 § 11).
- **The record** — AI-03 § 10's capability record as this suite emits it: subject, run, and one entry per capability in both closed lists.
- **The sentinel** — a caller-planted value that must never reach any observable output.

---

## AI-23.1 — Pluggable skeleton `[leaf]`

### R-CNF-001 — The suite runs against any subject through one factory seam, with zero copied assertions

The suite MUST accept a caller-supplied factory that builds a provider from a suite-authored scenario, and MUST run its whole case set against that subject without the caller writing a single assertion. The scenario language MUST be AI-21's existing `Script`/`Step`/`Emit`/`Hold`/`Gate` vocabulary; the suite MUST NOT define a second scenario vocabulary. A subject that cannot express a scenario MUST fail that case with an attributable message, and MUST NOT cause the run to hang or abort the remaining cases.

#### Scenarios

- **S-CNF-001** — Given AI-21's fake wrapped as a factory, when the suite runs against it, then every required case executes and passes, and the wrap contributes no assertion of its own.
- **S-CNF-002** — Given the landed suite source, when a reviewer looks for a second scenario vocabulary or a re-implemented Layer 1 assertion, then none exists — cases delegate to AI-22's kit and to `ai.CheckStream`.
- **S-CNF-003** — Given a factory that cannot realise one scenario, when the suite runs, then that case fails naming the subject and the scenario, and the remaining cases still run to completion.

### R-CNF-002 — *(non-negotiable)* The factory declares its capability expectation, so `absent` is askable for every optional capability

Only `CAP-O-02` is discoverable by asking the provider value (`R-AMP-017`); `CAP-O-01` and `CAP-O-03` have no askable seam. The factory MUST therefore carry a declared capability expectation over `CAP-O-01…03` alongside the provider it builds, so every optional capability has a stated position before the run starts. The suite MUST treat a declared non-offer as a **conclusion** and record `absent`, and MUST NOT record it as `not exercised`. WHEN the declaration and an askable discovery disagree — a subject declaring `CAP-O-02` absent while satisfying the token-counting contract, or the reverse — the suite MUST fail that entry rather than trusting either side silently.

#### Scenarios

- **S-CNF-004** — Given a factory declaring `CAP-O-03` not offered, when the suite runs, then the record carries `CAP-O-03` = `absent` and the run is not inconclusive on that entry.
- **S-CNF-005** — Given a factory declaring `CAP-O-02` offered by a provider that does not satisfy the token-counting contract, when the suite runs, then the entry is `failed` and the failure names the contradiction between declaration and discovery.
- **S-CNF-006** — Given a factory that declares nothing for one optional capability, when the suite runs, then it fails at construction naming the undeclared capability, rather than producing a record with a `not exercised` entry.

### R-CNF-003 — Every case is marked from AI-03 § 11 alone, and an unclassified case defaults to required

Each case MUST be keyed to the single capability it exercises, and its standing MUST be derived mechanically: **optional if and only if** that capability is in `CAP-O-01…03`; **every other case required**. Standing MUST come from AI-03, never from the suite's node structure and never from the run. A case whose capability is unclassified MUST default to **required** and fail loudly.

#### Scenarios

- **S-CNF-007** — Given the usage absent-versus-zero case, which lives in the AI-23.8 node but exercises `CAP-R-03`, when its standing is computed, then it is **required** (AI-03 § 11's last worked row).
- **S-CNF-008** — Given the exactly-one-terminal and the redaction cases, which exercise no listed optional capability, when their standing is computed, then both are required by the default rather than by an entry.
- **S-CNF-009** — Given a case keyed to a capability outside both closed lists, when the suite runs, then it is marked required and fails naming the unclassified capability.

### R-CNF-004 — A skipped optional case is reported, never silent

WHEN an optional case is not exercised because its capability is absent, the suite MUST report the skip with the capability, the case, and the reason. It MUST NOT emit a silently empty result, and MUST NOT let a skipped case read as a pass. A required case MUST NOT be skippable for any reason; there are no waivers.

#### Scenarios

- **S-CNF-010** — Given a subject offering no optional capability, when the suite runs, then the run output names each skipped optional case with its capability and the absence reason.
- **S-CNF-011** — Given the suite invoked with an attempt to skip a required case, when it runs, then the attempt fails and names the required case rather than honouring it.

---

## AI-23.2 — Text and lifecycle cases `[leaf]`

### R-CNF-005 — Text arrives ordered, contiguous, and reconstructs exactly

The suite MUST assert that a text interaction produces block start → deltas → block end in that order, that sequence numbers run 1..N contiguously with no gap and no repeat, and that concatenating the deltas reconstructs the source text **byte-exactly**. Reconstruction MUST be proven with a multi-byte rune deliberately split across two deltas, so a subject that reassembles per-rune rather than per-byte fails. Ordering MUST be asserted by delegating to the shipped checker and contiguity to AI-22's `R-STK-006` helper; neither MUST be reimplemented.

#### Scenarios

- **S-CNF-012** — Given a scripted text interaction, when the suite drains and asserts it, then kinds appear in block-start/delta/block-end order and sequences run 1..N with no gap.
- **S-CNF-013** — Given a text whose multi-byte rune is split across two adjacent deltas, when the deltas are concatenated, then the result equals the source byte-for-byte.
- **S-CNF-014** — Given a subject whose deltas arrive reordered, when the suite runs, then the case fails carrying the shipped checker's own verdict.

### R-CNF-006 — An empty completion is legal and must not be treated as a defect

The suite MUST assert that a normally-finished interaction producing **no** text content is conformant: it terminates normally, carries its terminal event, and MUST NOT be reported as a failure, as an incomplete stream, or as a contiguity violation. The suite MUST NOT require any minimum event count.

#### Scenarios

- **S-CNF-015** — Given a subject that finishes normally having emitted no text delta, when the suite runs the empty-completion case, then it passes and the terminal is present.
- **S-CNF-016** — Given the same interaction, when the contiguity assertion runs over its recording, then it reports no violation for the absent text block.

---

## AI-23.3 — Tool-call cases `[leaf]`

### R-CNF-007 — A tool call is reconstructible whole, fragmented or not, and carries an observable ordinal

The suite MUST assert that a tool call arrives with identity, name, exact argument bytes and an observable ordinal (`CAP-R-02`). It MUST cover a call whose argument bytes arrive **fragmented across several deltas interleaved with another call's**, reconstructing each call's bytes exactly and attributing every fragment to the right call. It MUST also cover a **whole call delivered with zero deltas**, which MUST be accepted rather than rejected for missing incremental delivery. Ordinals MUST be observable and MUST distinguish two calls to the same tool name.

#### Scenarios

- **S-CNF-017** — Given two concurrent tool calls whose argument fragments interleave, when the suite reconstructs them, then each call's argument bytes match its source exactly and no fragment is misattributed.
- **S-CNF-018** — Given a tool call delivered whole with zero argument deltas, when the suite runs, then the case passes — the zero-delta shape is mandatory to support.
- **S-CNF-019** — Given two calls to the same tool name, when their ordinals are read, then they differ and order the two calls unambiguously.

### R-CNF-008 — Mixed text and tool content ends on the tool-call finish reason

The suite MUST assert an interaction that emits text content **and** a tool call, terminating normally with the finish reason that denotes a tool call. Text and tool events MUST both survive with their ordering invariants intact, and the finish reason MUST come from the closed vocabulary rather than from free text.

#### Scenarios

- **S-CNF-020** — Given an interaction emitting a text block and then a tool call, when it finishes, then both are present, ordering holds, and the finish reason is the tool-call value from the closed vocabulary.

---

## AI-23.4 — Terminal and error cases `[leaf]`

### R-CNF-009 — Exactly one terminal, and the partial-output discriminator is always answerable

The suite MUST assert that every stream carries **exactly one** terminal event and that nothing follows it — on the normal path and on both failure-delivery paths. It MUST assert that each failure states whether content preceded it (`CAP-R-05`), and MUST cover both sides: a failure after content emitted and a failure with none. A stream carrying two terminals, or a terminal followed by any further event, MUST fail.

#### Scenarios

- **S-CNF-021** — Given a normally-finished stream, a pre-stream failure, and a mid-stream failure, when each is asserted, then each carries exactly one terminal and nothing follows it.
- **S-CNF-022** — Given a failure raised after two text deltas and a failure raised before any event, when the discriminator is read, then the first reports content preceded and the second reports none.
- **S-CNF-023** — Given a subject emitting a text delta after its terminal, when the suite runs, then the case fails naming the post-terminal event.

### R-CNF-010 — All nine failure categories are iterated exhaustively against the shipped enumerator

The suite MUST iterate **every** member of the failure-category vocabulary by consuming AI-19's exported enumerator, not a hand-written list, and MUST assert each category is expressible and classifiable on both delivery paths. A new category added upstream MUST cause the suite to fail until a case covers it; the exhaustiveness MUST be enforced mechanically, never by reviewer vigilance.

#### Scenarios

- **S-CNF-024** — Given the nine enumerated failure categories, when the suite runs, then each is exercised and classified through the one vocabulary, on both the pre-stream and mid-stream paths.
- **S-CNF-025** — Given a hypothetical tenth category added to the enumerator without a case, when the suite's exhaustiveness check runs, then it fails naming the uncovered category.

---

## AI-23.5 — Cancellation and closure cases `[leaf]`

### R-CNF-011 — Cancellation closes within bounded time and leaks no goroutine

The suite MUST assert that cancelling the caller-owned signal ends the stream within a bounded deadline (`CAP-R-04`) and that the stream is closed exactly once by its producer. Leak freedom MUST be asserted through AI-22's opt-in, amplitude-based `R-STK-007` helper; the suite MUST NOT install an implicit global leak check and MUST NOT call `t.Parallel()` in any test that uses it.

#### Scenarios

- **S-CNF-026** — Given a stream cancelled mid-consumption, when the suite waits with a bounded deadline, then the stream closes before the deadline and is closed exactly once.
- **S-CNF-027** — Given the cancellation scenario repeated under the leak helper, when it completes, then no goroutine growth beyond the helper's stated tolerance is observed.
- **S-CNF-028** — Given every suite test using the leak helper, when a reviewer enumerates them, then none calls `t.Parallel()`.

### R-CNF-012 — The abandoned-then-cancelled saturated path drops cleanly, with no terminal invented

The suite MUST assert AI-20.3's saturated-drop physics: a consumer that stops reading until the buffer saturates, whose caller then cancels, MUST see the stream close **bare** — no terminal event invented, no undelivered event forced through, no leak. The suite MUST NOT assert the **abandoned-never-cancelled** path; that narrowing is inherited from `ai-stream-lifecycle` § 5's untestability ruling and is recorded here rather than left silently absent.

#### Scenarios

- **S-CNF-029** — Given a consumer that stops reading until the buffer saturates and a caller that then cancels, when the stream terminates, then it closes bare with no terminal event and the events already delivered intact.
- **S-CNF-030** — Given that scenario repeated under the leak helper, when it completes, then no goroutine growth beyond tolerance is observed.
- **S-CNF-031** — Given the landed suite, when a reader looks for an abandoned-never-cancelled case, then it is stated out of scope with the § 5 reason cited.

---

## AI-23.7 — Redaction cases `[leaf]`

### R-CNF-013 — A planted sentinel appears in no event, no error string, and no failure output

The suite MUST accept a sentinel value on the factory's configuration and MUST require each subject's factory adapter to plant it where that subject carries sensitive material. It MUST then assert the sentinel appears in **no** event payload, in **no** error string, and in **no** test-failure output the suite itself produces — including AI-22's bounded payload summaries and first-divergence diffs. This case is **required** (no listed optional capability, AI-03 § 11). The suite MUST reuse the established planted-sentinel pattern rather than inventing a second one, and MUST NOT require new AI-21 surface to plant it.

#### Scenarios

- **S-CNF-032** — Given a sentinel planted into the subject's failure-report fields, when the suite scans every emitted event and every error string, then the sentinel appears in none.
- **S-CNF-033** — Given a deliberately failing case whose recording carries the sentinel in a payload, when the suite renders its diff and payload summaries, then the sentinel appears in no part of the failure output.
- **S-CNF-034** — Given a subject that does leak the sentinel into an error string, when the suite runs, then the redaction case fails and names where the leak was observed, without reprinting the sentinel itself.

---

## AI-23.8 — Optional-capability cases `[leaf]`

### R-CNF-014 — `CAP-O-01` reasoning content: emitted whole, and never leaking into text

WHEN a subject offers reasoning content, the suite MUST assert that reasoning arrives as its own blocks, that reasoning **never** appears in the text channel, and that a signature the subject carries round-trips byte-exactly (AI-03 § 6.1: advertised is advertised whole). Both the plain and the redacted reasoning doors MUST be covered, and the redacted bit MUST carry forward into every delta and end built from that start. WHEN the subject does not offer it, the entry MUST be `absent` and the skip MUST be reported.

#### Scenarios

- **S-CNF-035** — Given a subject offering reasoning, when the suite runs the case, then reasoning arrives in its own blocks and no reasoning byte appears in any text event.
- **S-CNF-036** — Given a subject carrying a reasoning signature, when the block round-trips, then the signature is byte-identical to the source.
- **S-CNF-037** — Given a redacted reasoning block start, when its deltas and end are read, then each carries the redacted bit forward.
- **S-CNF-038** — Given a subject declaring reasoning not offered, when the suite runs, then the case is skipped with a report and the record entry is `absent`.

### R-CNF-015 — `CAP-O-02` token counting: asked of the provider value, and a clean absence

The suite MUST ask for token counting **of the provider value**, never of a model identity, a configuration entry or a catalog. WHEN the capability is present it MUST answer a count and the suite MUST assert the answer. WHEN it is absent, the observation MUST be a clean absence — **not an error and not a zero** — and MUST NOT be reported through the failure vocabulary. An advertised counter that declines to answer MUST be `failed`, not `absent` (AI-03 § 9's advertising-binds rule).

#### Scenarios

- **S-CNF-039** — Given a subject satisfying the token-counting contract, when the suite asks, then it receives the capability and the returned count is asserted.
- **S-CNF-040** — Given a subject not satisfying it, when the suite asks, then the result is a clean absence: no error raised and no zero count substituted.
- **S-CNF-041** — Given a subject that satisfies the contract and then declines to answer, when the suite runs, then the entry is `failed` rather than `absent`.

### R-CNF-016 — `CAP-O-03` cache-boundary honoring, and the required cases living in this node

The suite MUST cover `CAP-O-03` through the declared capability expectation of `R-CNF-002`: honoring MUST be asserted as consumer-visible behaviour when offered, and recorded `absent` when the factory declares it is not. This node MUST additionally carry two cases that are **required** because they exercise `CAP-R-03`: (a) a normally-finished stream ends carrying a finish reason from the closed vocabulary, iterated **exhaustively over all seven values**; and (b) usage honours absent-versus-zero per field, so an absent count is distinguishable from a reported zero. Because the finish-reason vocabulary exports no enumerator, the seven values MUST be hand-listed in the suite **behind a drift guard** that fails when the upstream vocabulary gains or loses a member; the suite MUST NOT modify `src/ai` to obtain an enumerator.

#### Scenarios

- **S-CNF-042** — Given a subject declaring cache-boundary honoring offered, when the suite exercises it, then honoring is observed as consumer-visible behaviour; given one declaring it not offered, the entry is `absent` with a reported skip.
- **S-CNF-043** — Given all seven finish reasons, when the suite iterates them, then each is reachable on a normally-finished stream and each is a closed-vocabulary value.
- **S-CNF-044** — Given a hypothetical eighth finish reason added upstream without a suite case, when the drift guard runs, then it fails naming the uncovered value; given a value removed, it likewise fails.
- **S-CNF-045** — Given a usage record with one count absent and another reported as zero, when the suite reads them, then the two are distinguishable and neither is coerced into the other.
- **S-CNF-046** — Given the standing of cases (a) and (b), when it is computed, then both are **required** despite living in the optional-capability node.

---

## AI-23.6 — The capability record `[leaf, depends on .2 .3 .4 .5 .7 .8]`

### R-CNF-017 — The record is total over both closed lists, with standing taken from AI-03

The emitted record MUST identify its subject and its run, and MUST carry **exactly one entry per capability** in `CAP-R-01…05` and `CAP-O-01…03` — eight entries, always. Each entry's **standing** MUST be taken from AI-03 § 5 and § 6, never from the run. Each entry's **outcome** MUST come from the closed four-value set — `satisfied`, `absent`, `failed`, `not exercised` — with `absent` legal for **optional entries only**. A capability with no entry MUST be a defect in the run, not an absence. The record MUST carry no capability-specific detail, no model content, no credential and no raw provider text.

#### Scenarios

- **S-CNF-047** — Given any completed run, when the record is read, then it carries exactly eight entries, one per capability in the two closed lists, each naming its capability, standing and outcome.
- **S-CNF-048** — Given a run whose subject offers no optional capability, when the record is read, then the three optional entries are `absent` and no entry is `not exercised`.
- **S-CNF-049** — Given a run in which a required capability's cases failed, when the record is read, then that entry is `failed` — never `absent`, for which required standing has no legal value.
- **S-CNF-050** — Given a run that recorded a required capability as optional, when the record is validated, then it fails: standing is not the run's to supply.
- **S-CNF-051** — Given a published record, when it is inspected for capability-specific detail, model content or credentials, then it carries none.

### R-CNF-018 — The verdict rule: `not exercised` is inconclusive, and a failed required entry cannot pass

The suite MUST compute the verdict mechanically: a record is a **pass** when every required entry is `satisfied`, every optional entry is `satisfied` or `absent`, and **no** entry is `not exercised`. A record containing any `not exercised` entry MUST be **inconclusive** — neither a pass nor a failure. The four-value set MUST NOT collapse to three at verdict time, and a skipped case MUST NOT read as a pass. Two records MUST be comparable entry by entry, so a difference in either direction is a finding.

#### Scenarios

- **S-CNF-052** — Given a record whose required entries are all `satisfied` and whose optional entries are `satisfied` or `absent`, when the verdict is computed, then it is a pass.
- **S-CNF-053** — Given a record with one `not exercised` entry and every other entry `satisfied`, when the verdict is computed, then it is **inconclusive** — not a pass and not a failure.
- **S-CNF-054** — Given a record with one `failed` required entry, when the verdict is computed, then it is a failure and no optional result offsets it.
- **S-CNF-055** — Given two records for the same subject differing on one entry, when they are compared, then the comparison names that entry and the direction of the difference.
- **S-CNF-056** — Given AI-21's fake as the first subject, when the suite runs end to end, then the verdict is a pass and every required entry is `satisfied`.

---

## Non-functional requirements

### NFR-CNF-A — Dependency purity

This change MUST add no module dependency. The suite MUST import only the standard library and the module's own `ai` and `agenttest` packages. Both AI-00 import guards MUST still pass and `backend/agent/go.mod` MUST still declare zero requires.

- **S-CNF-057** — Given the change merged, when `go.mod` is read and both import guards run, then it declares no require and both guards pass.

### NFR-CNF-B — *(non-negotiable)* Layer 1 is read, never modified

This change MUST NOT modify `src/ai` — including `finish_reason.go`. `R-CNF-016`'s exhaustive seven-value iteration MUST be satisfied by the hand-list-plus-drift-guard mechanism, proving exhaustiveness behaviourally against a vocabulary the suite cannot introspect, in the manner AI-22's `R-STK-004` already established. Adding an exported enumerator to `ai` is **rejected for this change**, and the rejection is reversible by a later milestone. `src/ai` MUST compile and pass identically with or without this change. AI-20.4's signature guard MUST pass **unmodified**.

- **S-CNF-058** — Given the merged diff, when a reviewer looks for an edit under `src/ai/`, then none exists.
- **S-CNF-059** — Given the merged change, when the signature guard runs, then it passes unmodified and still observes the single decided method.
- **S-CNF-060** — Given the change reverted in isolation, when `src/ai`'s, AI-21's and AI-22's tests run, then they pass exactly as before.

### NFR-CNF-C — Package placement follows the established convention

The suite MUST ship **inside `src/agenttest/`**, not in a new sibling package — the convention AI-21 established and AI-22 extended. Its files MUST use a `conformance_` prefix, distinct from AI-21's `fake_` and AI-22's `stream_kit_` prefixes. `src/agenttest/doc.go` MUST name the conformance suite alongside the fake and the kit, and MUST retain their existing framing and the dependency-free pin.

- **S-CNF-061** — Given the merged change, when the package layout is read, then the suite lives in `src/agenttest/` under `conformance_`-prefixed files and no new sibling package was created.
- **S-CNF-062** — Given the landed package documentation, when a reader looks for the package's contents, then it names the fake, the kit and the conformance suite.

### NFR-CNF-D — AI-21's fake and AI-22's kit are consumed unchanged

This change MUST NOT modify AI-21's or AI-22's shipped source or test files. Every assertion MUST delegate to AI-22's kit or to the shipped checker; the suite MUST NOT reimplement a drain, a diff, an ordering check, a contiguity check or a leak detector.

- **S-CNF-063** — Given the merged diff, when a reviewer looks for edits to `fake_*.go` or `stream_kit_*.go`, then none exists.
- **S-CNF-064** — Given each conformance case, when a reader looks for what it asserts, then each names the Layer 1 requirement identifier it proves and delegates its mechanics to the kit.

### NFR-CNF-E — Determinism, no network, and totality

No behaviour this spec requires MUST depend on network access, credentials, or elapsed wall-clock time for correctness. Bounded deadlines proving a call does **not** hang are permitted; a sleep used to order events is not. No exported suite entry point MUST panic for any input — including a nil factory, an empty case selection, a nil subject and an undeclared capability expectation — and MUST instead fail the test with an attributable message.

- **S-CNF-065** — Given the milestone's whole test set run repeatedly under `-race`, when results are compared, then they are identical across runs and no test performs I/O.
- **S-CNF-066** — Given a table of extreme inputs passed through every exported entry point, when each runs, then none panics and each failure names the entry point and the offending input.

### NFR-CNF-F — Evidence

Every test-list item of AI-23.1 … AI-23.8 MUST be taken red → green → refactored **in order**, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green `make test` (`go test -race -v ./...`) from `backend/agent/` and clean `make lint`.

- **S-CNF-067** — Given `tasks.md`, when a reviewer walks the milestone's test-list items, then each carries recorded red output, recorded green output, and a refactor note.

---

## Acceptance criteria

1. A provider factory runs the whole case set with **zero copied assertions**, in AI-21's existing scenario vocabulary, and AI-21's fake is the first subject and passes (`R-CNF-001`, `S-CNF-056`).
2. The factory declares its capability expectation, so `absent` is a conclusion for all three optional capabilities and a declaration/discovery contradiction fails (`R-CNF-002`).
3. Every case is marked from AI-03 § 11 alone, an unclassified case defaults to required, and every skipped optional case is reported (`R-CNF-003`, `R-CNF-004`).
4. Text reconstructs byte-exactly across a split multi-byte rune, sequences are 1..N contiguous, and an empty completion is legal (`R-CNF-005`, `R-CNF-006`).
5. Fragmented interleaved tool calls reconstruct exactly, a zero-delta whole call is accepted, ordinals are observable, and mixed content ends on the tool-call finish reason (`R-CNF-007`, `R-CNF-008`).
6. Exactly one terminal on all three paths, the partial-output discriminator answered both ways, and all **nine** failure categories iterated against the shipped enumerator (`R-CNF-009`, `R-CNF-010`).
7. Cancellation closes within a bounded deadline with no leak, and the abandoned-then-cancelled saturated path closes bare with the abandoned-never-cancelled narrowing recorded (`R-CNF-011`, `R-CNF-012`).
8. A planted sentinel appears in no event, no error string and no test-failure output, including bounded summaries and diffs (`R-CNF-013`).
9. All three optional subjects are covered — reasoning, token counting, cache-boundary honoring — and the finish-reason and usage cases in that node are marked **required**, with all **seven** finish reasons iterated behind a drift guard and no `src/ai` edit (`R-CNF-014` … `R-CNF-016`, `NFR-CNF-B`).
10. The record carries exactly eight entries with standing from AI-03 and the four-value outcome set; any `not exercised` entry is **inconclusive** and a failed required entry cannot pass (`R-CNF-017`, `R-CNF-018`).
11. `make test` green under `-race`, `make lint` clean, AI-00 import guards and AI-20.4's signature guard passing unmodified, `go.mod` unchanged, `src/ai` and AI-21/AI-22 sources untouched (`NFR-CNF-A` … `NFR-CNF-F`).

## What design resolved

> **Resolved at archive 2026-08-03.** This section originally left four items to design, deliberately undecided at spec time. All four are settled and shipped.

`R-CNF-001`'s factory and `R-CNF-002`'s capability-expectation value travel together as one `Factory` struct: `New func(testing.TB, ...Script) ai.ModelProvider` plus `Reasoning`, `TokenCounting`, `CacheBoundary *bool`. **The `*bool` choice was itself a correction, not the original design.** The first design pass used bare `bool` fields, which cannot distinguish "declared not offered" (must record `absent`) from "never declared" (spec's `S-CNF-006` requires this to fail construction loudly) — a bare bool's zero value is `false` either way. This was caught during `sdd-tasks`, before apply, and fixed in `design.md`: `nil` fails construction naming the undeclared capability; non-nil `false` records `absent`; non-nil `true` is cross-checked against askable discovery (only `CAP-O-02` is askable, via `ai.TokenCounter`) and records `satisfied` or `failed`.

`R-CNF-003`'s marking and `R-CNF-017`'s totality are mechanical by construction: an 8-member `Capability` enum with an exhaustive enumerator, entries initialized to `OutcomeNotExercised` in a fixed `[8]`-element array that cannot be appended to. `R-CNF-016`'s finish-reason drift guard hand-lists the seven values and probes `ai.FinishReason(n).String()` upward to the first `"invalid"`, failing if the count diverges — proving exhaustiveness behaviorally against a vocabulary `src/ai` exports no enumerator for, the same technique AI-22's `R-STK-004` established for event kinds. `R-CNF-013`'s sentinel travels on the factory's own configuration (`Factory.Sentinel`); the fake's factory adapter plants it into `Failure.Cause` and `Failure.RequestID` (deliberately not `RawLabel`, whose bounded head is sanctioned rendering, not a leak) — zero new AI-21 surface, exactly as the proposal predicted.
