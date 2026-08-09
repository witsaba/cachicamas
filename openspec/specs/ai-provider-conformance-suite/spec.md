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

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The conformance suite's twenty-one requirements carried in this file — the factory seam, the declared-capability tri-state, the mechanical marking rule, the total nine-entry record, the single inheritable redaction sweep (`R-CNF-027`, added 2026-08-08 by AI-36), and the scoped-run-is-never-evidence rule (`R-CNF-028`, added 2026-08-09 by AI-38) — therefore live here, not only as a pointer into the archive. *(Count corrected 2026-08-08. It read "eighteen requirements" and "eight-entry record"; both were already stale before AI-36 — AI-35 landed `R-CNF-019` and grew the record to nine entries per `R-CNF-017` without updating this sentence. The count is of the requirements **in this file**; see the identifier note at the foot of this document for why `R-CNF-020` … `R-CNF-026` are absent from it.)* The archived change folder at [`openspec/changes/archive/2026-08-03-cachicamas-ai-conformance-suite/`](../../changes/archive/2026-08-03-cachicamas-ai-conformance-suite/) is the historical record of how AI-23 was explored, proposed, designed, applied and verified — including the `Factory` bool-to-`*bool` correction caught before apply (see below).

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

### R-CNF-010 — All nine failure categories are iterated exhaustively against the shipped enumerator, with dialect collapse recorded rather than skipped

The suite MUST iterate **every** member of the failure-category vocabulary by consuming AI-19's exported enumerator, not a hand-written list, and MUST assert each category is expressible and classifiable on both delivery paths. A new category added upstream MUST cause the suite to fail until a case covers it; the exhaustiveness MUST be enforced mechanically, never by reviewer vigilance.

A category a subject's wire dialect **collapses** on the mid-stream path — because the dialect carries no discriminator that could preserve it — MUST still produce exactly one typed terminal, and that terminal's category MUST equal the subject's declared collapse value. The suite MUST record this as a **dialect-aware collapse** naming both the category and the dialect; it MUST NOT read as a skip and MUST NOT read as a pass of the original category. The enumerator-driven exhaustive iteration and the mechanical drift failure MUST remain intact for the vocabulary as a whole; only the per-subject mid-stream classification claim is narrowed. A category the dialect **can** express mid-stream MUST still be classified as itself.
(Previously: every category was required to arrive mid-stream classified as itself on every subject, which no dialect lacking a mid-stream error discriminator can satisfy.)

> **Amended 2026-08-09 (AI-38)** by `cachicamas-ai-adapter-conformance`: the per-subject mid-stream classification claim is narrowed to admit a dialect-aware collapse, resolving a conflict with the shipped OpenRouter adapter's `stream_failure.go`, which hardcodes `FailureCategoryUnknown` for every in-band mid-stream error frame. `S-CNF-024` is restated; `S-CNF-087` is added as the anti-escape twin.

#### Scenarios

- **S-CNF-024** *(modified 2026-08-09, AI-38)* — Given the nine enumerated failure categories, when the suite runs against a subject, then each is exercised and classified through the one vocabulary on the pre-stream path; and on the mid-stream path each category the subject's dialect can express is classified as itself, and each category the dialect collapses arrives as exactly one typed terminal carrying the declared collapse value, recorded as a dialect-aware collapse naming the category and the dialect.
- **S-CNF-025** — Given a hypothetical tenth category added to the enumerator without a case, when the suite's exhaustiveness check runs, then it fails naming the uncovered category.
- **S-CNF-087** *(added 2026-08-09, AI-38)* — Given a category a subject's dialect can express mid-stream that arrives carrying any other category, when the suite runs, then the case fails naming the expected and observed categories — the dialect-aware collapse is not available as a general escape.

---

## AI-23.5 — Cancellation and closure cases `[leaf]`

### R-CNF-011 — Cancellation closes within bounded time, leaks no goroutine, and admits at most one typed cancellation terminal

The suite MUST assert that cancelling the caller-owned signal ends the stream within a bounded deadline (`CAP-R-04`) and that the stream is closed exactly once by its producer. The suite MUST accept **either** a bare close **or** exactly one terminal event whose failure payload is of the cancellation category (`ai-provider-error-mapping` `R-AEM-014`) as conformant cancellation shapes, and MUST reject anything else: more than one terminal, a terminal of any other category, or any event after the terminal. Leak freedom MUST be asserted through AI-22's opt-in, amplitude-based `R-STK-007` helper; the suite MUST NOT install an implicit global leak check and MUST NOT call `t.Parallel()` in any test that uses it.
(Previously: cancellation admitted only a bare close, which conflicted with the promoted `R-AEM-014` typed-terminal obligation; the suite over-specified one of two legal shapes.)

> **Amended 2026-08-09 (AI-38)** by `cachicamas-ai-adapter-conformance`: cancellation now admits either a bare close or exactly one cancellation-category terminal, resolving the conflict with the promoted `ai-provider-error-mapping` `R-AEM-014` / `S-AEM-051…055` typed-terminal obligation (AI-32.3). `S-CNF-082` is added.

#### Scenarios

- **S-CNF-026** — Given a stream cancelled mid-consumption, when the suite waits with a bounded deadline, then the stream closes before the deadline and is closed exactly once.
- **S-CNF-027** — Given the cancellation scenario repeated under the leak helper, when it completes, then no goroutine growth beyond the helper's stated tolerance is observed.
- **S-CNF-028** — Given every suite test using the leak helper, when a reviewer enumerates them, then none calls `t.Parallel()`.
- **S-CNF-082** *(added 2026-08-09, AI-38)* — Given a cancelled stream from a subject that emits one terminal carrying a cancellation-category failure, when the suite asserts cancellation, then the case passes; and given a subject that emits two terminals, or one terminal of any non-cancellation category, or any event after the terminal, then the case fails naming the observed shape.

### R-CNF-012 — The abandoned-then-cancelled saturated path drops cleanly, inventing no terminal beyond the admitted cancellation terminal

The suite MUST assert AI-20.3's saturated-drop physics: a consumer that stops reading until the buffer saturates, whose caller then cancels, MUST see the stream close with **no undelivered event forced through, no already-delivered event discarded, and no leak**. The closure MUST be either bare or carry exactly one cancellation-category terminal; no other terminal MUST be invented and no second terminal MUST appear. The suite MUST NOT assert the **abandoned-never-cancelled** path; that narrowing is inherited from `ai-stream-lifecycle` § 5's untestability ruling and is recorded here rather than left silently absent.
(Previously: the saturated path required a strictly bare close, forbidding the cancellation-category terminal `R-AEM-014` requires of the shipped adapter.)

> **Amended 2026-08-09 (AI-38)** by `cachicamas-ai-adapter-conformance`: the saturated abandoned-then-cancelled path now admits the same two shapes as `R-CNF-011` (bare close or one cancellation-category terminal), replacing the prior bare-close-only requirement. `S-CNF-083` is added.

#### Scenarios

- **S-CNF-029** — Given a consumer that stops reading until the buffer saturates and a caller that then cancels, when the stream terminates, then it closes with no undelivered event forced through and the events already delivered intact.
- **S-CNF-030** — Given that scenario repeated under the leak helper, when it completes, then no goroutine growth beyond tolerance is observed.
- **S-CNF-031** — Given the landed suite, when a reader looks for an abandoned-never-cancelled case, then it is stated out of scope with the § 5 reason cited.
- **S-CNF-083** *(added 2026-08-09, AI-38)* — Given the saturated-then-cancelled path on a subject that closes bare and on a subject that emits one cancellation-category terminal, when the suite runs, then both pass; and given a subject that invents a terminal of another category on that path, then the case fails naming the invented category.

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

A finish-reason value the subject's wire dialect cannot express MUST be recorded as a **dialect-aware absence** — unsatisfiable and unviolated — naming the value and the dialect, and MUST NOT read as a pass and MUST NOT be silently skipped. The exhaustive iteration and the drift guard MUST remain intact for the vocabulary as a whole; only the per-subject reachability claim is narrowed. A value the dialect **can** express but the subject does not produce MUST still fail.
(Previously: `S-CNF-043` claimed every one of the seven finish reasons is reachable on every subject's normally-finished stream, which no strict single-dialect adapter can satisfy.)

> **Amended 2026-08-09 (AI-38)** by `cachicamas-ai-adapter-conformance`: the finish-reason reachability claim is narrowed to admit a dialect-aware absence, resolving a conflict with the shipped OpenRouter adapter, whose single dialect cannot express every one of the seven closed-vocabulary finish reasons. No `ai-provider-completion` `R-ACP-002` reopen. `S-CNF-043` is restated; `S-CNF-084` is added as the anti-escape twin.

#### Scenarios

- **S-CNF-042** — Given a subject declaring cache-boundary honoring offered, when the suite exercises it, then honoring is observed as consumer-visible behaviour; given one declaring it not offered, the entry is `absent` with a reported skip.
- **S-CNF-043** *(modified 2026-08-09, AI-38)* — Given all seven finish reasons and a subject, when the suite iterates them, then each value its dialect can express is reachable on a normally-finished stream and is a closed-vocabulary value, and each value its dialect cannot express is recorded as a dialect-aware absence naming the value and the dialect.
- **S-CNF-044** — Given a hypothetical eighth finish reason added upstream without a suite case, when the drift guard runs, then it fails naming the uncovered value; given a value removed, it likewise fails.
- **S-CNF-045** — Given a usage record with one count absent and another reported as zero, when the suite reads them, then the two are distinguishable and neither is coerced into the other.
- **S-CNF-046** — Given the standing of cases (a) and (b), when it is computed, then both are **required** despite living in the optional-capability node.
- **S-CNF-084** *(added 2026-08-09, AI-38)* — Given a subject whose dialect can express a finish reason but which never produces it, when the suite runs, then the case fails naming the value — the dialect-aware absence is not available as a general escape.

---

## AI-23.6 — The capability record `[leaf, depends on .2 .3 .4 .5 .7 .8]`

### R-CNF-017 — The record is total over both closed lists, with standing taken from AI-03

The emitted record MUST identify its subject and its run, and MUST carry **exactly one entry per capability** in `CAP-R-01…05` and `CAP-O-01…04` — **nine entries, always**. Each entry's **standing** MUST be taken from AI-03 § 5 and § 6, never from the run. Each entry's **outcome** MUST come from the closed four-value set — `satisfied`, `absent`, `failed`, `not exercised` — with `absent` legal for **optional entries only**. A capability with no entry MUST be a defect in the run, not an absence. The record MUST carry no capability-specific detail, no model content, no credential and no raw provider text.

> **Amended 2026-08-07 (AI-35)** by `cachicamas-ai-retry-policy`: previously "eight entries, always" over `CAP-R-01…05` and `CAP-O-01…03`. AI-35 adds `CapRetry` as `CAP-O-04`, the ninth entry and fourth optional capability; S-CNF-047 and S-CNF-048 are restated accordingly and S-CNF-076 is added.

#### Scenarios

- **S-CNF-047** *(modified 2026-08-07, AI-35)* — Given any completed run, when the record is read, then it carries exactly **nine** entries, one per capability in the two closed lists (including `CAP-O-04(retry)`), each naming its capability, standing and outcome.
- **S-CNF-048** *(modified 2026-08-07, AI-35)* — Given a run whose subject offers no optional capability, when the record is read, then the **four** optional entries (`CAP-O-01…04`) are `absent` and no entry is `not exercised`.
- **S-CNF-049** — Given a run in which a required capability's cases failed, when the record is read, then that entry is `failed` — never `absent`, for which required standing has no legal value.
- **S-CNF-050** — Given a run that recorded a required capability as optional, when the record is validated, then it fails: standing is not the run's to supply.
- **S-CNF-051** — Given a published record, when it is inspected for capability-specific detail, model content or credentials, then it carries none.
- **S-CNF-076** *(added 2026-08-07, AI-35; pin: `R-CNF-003`, `R-CNF-018`)* — Given the capability closed lists and their enumerator, when a reviewer inspects them, then the closed list is `CAP-R-01…05` and `CAP-O-01…04` (nine entries, in declaration order), AND the none-marker stays deliberately excluded from the enumerator (unchanged from the eight-member rule), AND the retry capability is declared last so the enumerator's contiguous range grows by exactly one.

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

> **Amended 2026-08-07 (AI-35)** by `cachicamas-ai-retry-policy` (AI-35, Wave 5 — Harden). One optional capability `CapRetry` (CAP-O-04) added to the conformance suite's closed list; the `Capability` enum grows from `[8]` to `[9]`; `R-CNF-017` totality rebuild is mechanical (one new line). One requirement `R-CNF-019` added: every adapter claiming `CapRetry` auto-retries retryable pre-stream failures up to the documented Layer 1 bound, never retries terminal-category failures regardless of position, re-issues with byte-identical wire body across attempts, and exposes the attempt count and final cause via the cause chain.

### R-CNF-019 (added 2026-08-07) — Every adapter claiming `CapRetry` auto-retries retryable pre-stream failures up to a documented bound

An adapter that declares `CapRetry` (CAP-O-04) MUST satisfy all of the following:

1. **Every retryable-flagged pre-stream failure is retried up to the documented Layer 1 bound** (per `R-AIS-044`, meaning up to `N+1` wire requests per logical call, where `N` is documented as 3).
2. **Terminal-category failures — authentication, invalid request — are NEVER retried**, regardless of position.
3. **The per-attempt wire body is byte-identical to the original** across attempts (per `R-AIS-043` / S-1).
4. **The attempt count and the final cause are reachable from the returned error chain** via the cause-chain accessor (per `R-AIS-043` / S-2).

An adapter that does **not** declare `CapRetry` is not bound by `R-CNF-019` but MUST still satisfy `R-CNF-001` … `R-CNF-018`. The `CapRetry` declaration is mandatory input to the run (extending `R-CNF-002`: a factory whose `CapRetry` declaration field is `nil` fails construction per `S-CNF-006`).

#### Scenario: R-CNF-019 / S-CNF-069 — Retryable pre-stream failure is retried up to the documented bound *(pin: `R-AIS-041` / S-1, `R-AIS-042` / S-4, `R-AIS-044`)*

- **GIVEN** a scripted transport that returns rate-limit (429) with retry-after 0, repeated for up to `N+1` attempts, then a 2xx text-stream
- **WHEN** the suite runs the `CapRetry` case against an adapter that declares `CapRetry`
- **THEN** the case asserts every retryable-flagged pre-stream failure was retried up to the documented bound, AND the final stream is the `N+1`-th attempt's 2xx stream, AND the typed failure (if returned) carries the attempt count via the cause chain

#### Scenario: R-CNF-019 / S-CNF-070 — Terminal-category failure is never retried *(pin: `R-AIS-041` / S-2, `R-AEM-008`, `R-AEM-009`)*

- **GIVEN** a scripted transport that returns authentication failure on a single attempt
- **WHEN** the suite runs the `CapRetry` case
- **THEN** the case asserts no retry occurred (wire request count == 1, instrumented), AND the typed failure is returned with retryability marked false, AND no second wire request follows

#### Scenario: R-CNF-019 / S-CNF-071 — No retry after partial output *(pin: `R-AIS-043` / S-4, `R-AIP-010`, `R-AIP-011`, `V-FAIL-15`, `AG-15.1`)*

- **GIVEN** a scripted transport that returns one 2xx text-stream frame, then a retryable-flagged failure mid-stream
- **WHEN** the suite runs the `CapRetry` case
- **THEN** the case asserts wire request count == 1 (no automatic retry after a semantic event has been emitted), AND the typed failure reaches the consumer via the terminal error event with partial output marked true

#### Scenario: R-CNF-019 / S-CNF-072 — Byte-identical replay across attempts *(pin: `R-AIS-043` / S-1, `R-ART-010`)*

- **GIVEN** a scripted transport that records every request body's bytes, and a request the helper retries `N+1` times
- **WHEN** the suite runs the `CapRetry` case
- **THEN** the case asserts every recorded body is byte-identical to every other (no drift, no re-encoding), AND the recorded request bodies match the original request's bytes verbatim

#### Scenario: R-CNF-019 / S-CNF-073 — Attempt count and final cause reachable from error chain *(pin: `R-AIS-043` / S-2, `V-FAIL-15`)*

- **GIVEN** the helper has exhausted its attempt budget
- **WHEN** the suite reads the returned error via the cause-chain accessor
- **THEN** the attempt count equals `N+1` and the final cause is reachable as a typed failure carrying the original failure's category and delivery classification

#### Scenario: R-CNF-019 / S-CNF-074 — CapRetry absent is reported, not silent *(pin: `R-CNF-002`, `R-CNF-004`)*

- **GIVEN** an adapter whose factory carries the `CapRetry` declaration field as false (declared not offered)
- **WHEN** the suite runs against it
- **THEN** the record carries `CapRetry` = `absent` per `R-CNF-004` (skipped optional case reported, never silent), AND the run is not inconclusive on that entry, AND the adapter still satisfies `R-CNF-001` … `R-CNF-018`

#### Scenario: R-CNF-019 / S-CNF-075 — Factory nil Retry fails construction *(pin: `R-CNF-002`, `S-CNF-006`)*

- **GIVEN** a factory whose `CapRetry` declaration field is `nil` (undeclared)
- **WHEN** the suite runs
- **THEN** it fails at construction naming the undeclared capability, per `R-CNF-002` / `S-CNF-006` — the same nil-disallowed discipline `Reasoning`, `TokenCounting`, `CacheBoundary` already enforce

---

> **Amended 2026-08-08 (AI-36)** by [`cachicamas-ai-redaction`](../../changes/archive/2026-08-08-cachicamas-ai-redaction/) (AI-36 — Enforce secret redaction, Wave 5 — Harden). One requirement `R-CNF-027` added: sentinel absence is proven through a **single** inheritable sweep, and every absence claim ships a positive control so it cannot pass vacuously. No requirement is modified, removed or renamed — `R-CNF-001` … `R-CNF-019` and `NFR-CNF-A` … `NFR-CNF-F` stand unchanged. `R-CNF-013`'s planted-sentinel case is the pin this requirement generalises; it is exercised, not amended.
>
> **Identifier note — the gap from `R-CNF-019` to `R-CNF-027` is pre-existing and deliberately left open here.** `R-CNF-020` … `R-CNF-026` are already consumed by two archived changes — [`2026-08-05-cachicamas-ai-conformance-lifecycle-amendment`](../../changes/archive/2026-08-05-cachicamas-ai-conformance-lifecycle-amendment/) (020–024) and [`2026-08-05-cachicamas-ai-conformance-tool-amendment`](../../changes/archive/2026-08-05-cachicamas-ai-conformance-tool-amendment/) (025–026) — whose requirements were never promoted into this file. Those identifiers are nevertheless **live**: `R-CNF-023` and `R-CNF-024` are cited as binding dependencies by [`ai-provider-text-stream`](../ai-provider-text-stream/spec.md) and [`ai-provider-completion`](../ai-provider-completion/spec.md). AI-36 therefore takes `R-CNF-027`, the first genuinely free identifier, and neither renumbers nor backfills the gap. Restoring the seven missing requirements is a separate, recorded repository defect and is not in this milestone's scope.

### R-CNF-027 (added 2026-08-08) — Redaction is proven by one inheritable sweep whose absence claim is falsifiable

The suite MUST prove sentinel absence through a **single** sweep capability that every consumer reaches, rather than through per-consumer re-implementations. Specifically:

1. A distinctive sentinel **credential**, configured into the adapter under test, MUST appear in no error rendering, no wrapped cause reachable through a rendering, no verbose or Go-syntax rendering, and no event metadatum, across **every** failure path the suite can trigger.
2. A distinctive sentinel **content body**, sent through the adapter, MUST be equally absent from every error rendering, every log field and every event metadatum.
3. Exactly **one** sweep implementation MUST exist in the module. The suite's redaction case and the live-smoke sweep MUST both reach it, and both MUST keep their already-published names, so consumers bound to them are not broken.
4. A newly registered failure path MUST inherit the sweep without a second sweep being written.
5. Every absence claim MUST ship a **positive control** proving the sweep detects a planted leak. An absence assertion that cannot fail does not satisfy this requirement.
6. No sweep report MAY reproduce the bytes it matched. A report MUST name the vector, the file, the event position or the verb only.

#### Scenarios

- **S-CNF-077** — Given an adapter configured with a sentinel credential, when every failure path the suite can trigger is exercised and each resulting error rendering, wrapped cause, verbose rendering, Go-syntax rendering and event metadatum is swept, then the sentinel credential appears in none of them.
- **S-CNF-078** — Given a request whose content carries a sentinel body, when the same failure paths are exercised and every error rendering, log field and event metadatum is swept, then the sentinel body appears in none of them.
- **S-CNF-079** — Given a deliberately planted leak of each sentinel in turn, when the sweep runs, then it reports a failure for each, and given that report, when it is inspected, then it names the vector and position and contains neither sentinel.
- **S-CNF-080** — Given the shipped module, when its sweep implementations are enumerated, then exactly one exists; and when the suite's redaction case and the live-smoke sweep are each run, then both reach that one implementation, both keep their previously published names, and the tests bound to those names stay green.
- **S-CNF-081** — Given a provider that echoes the caller's authorization value and the caller's request body back inside a non-streaming response, when the resulting refusal is rendered and swept, then the sentinel credential is absent; and given any disclosure attributable solely to the provider replaying the caller's own content, when the milestone closes, then that residual is recorded in writing as a named residual rather than silently absent.

> **`S-CNF-081`'s named residual, recorded at AI-36's close (2026-08-08).** The hostile-server case fired both of its empirical branches. The **credential** vector bit: a provider echoing the caller's authorization value into a non-streaming response reproduced that credential verbatim inside the refusal's bounded excerpt. It is fixed, and the fix is now binding as [`ai-provider-error-mapping`](../ai-provider-error-mapping/spec.md)'s `R-AEM-019`. The **caller's own prompt content** was observed echoed into the same excerpt on the same run and is **deliberately not suppressed** — suppressing it would defeat the landed obligation that the excerpt stay diagnostically readable (`R-AEM-019` item 2). It is therefore recorded here, and in `R-AEM-019` item 4 and `S-AEM-072`, as an accepted named residual rather than a defect. The credential's absence is proven, not assumed: that guard was empirically defeated and restored.

> **Amended 2026-08-09 (AI-38)** by [`cachicamas-ai-adapter-conformance`](../../changes/archive/2026-08-09-cachicamas-ai-adapter-conformance/) (AI-38 — Run full deterministic adapter conformance, Wave 6 — Hand off). One requirement `R-CNF-028` added: the scoped, per-capability entry point is never presentable as evidence of full conformance — promoted from a doc comment (this file's own `RunConformanceFor` documentation) to a requirement, so the rule is enforced mechanically rather than by reviewer vigilance. `R-CNF-010`, `R-CNF-011`, `R-CNF-012` and `R-CNF-016` are amended in place — see each requirement's own amendment note above. No requirement is removed or renamed.

### R-CNF-028 (added 2026-08-09) — A scoped run is never presentable as evidence of full conformance

The suite's scoped, per-capability entry point MUST remain available as a debugging affordance, and its verdict MUST be reported as non-evidential. A scoped verdict MUST NOT be recorded as a conformance pass, MUST NOT emit a record that a consumer could mistake for a total record, and MUST NOT be cited as acceptance evidence in code, comment, report, or spec. Only the unscoped entry point's run MUST count as full-conformance evidence. This obligation is a requirement of the suite, not a documentation comment.

#### Scenarios

- **S-CNF-085** — Given a scoped per-capability run, when its verdict and any emitted record are read, then both are marked non-evidential and the record is not total over the nine entries.
- **S-CNF-086** — Given an acceptance artifact that cites a scoped run as full-conformance evidence, when the suite's evidence check runs, then it fails naming the citation.
