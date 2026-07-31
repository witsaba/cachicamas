# Proposal — the v1 capability set and optional-capability discovery

> **Change**: `cachicamas-ai-minimum-capabilities`
> **Milestone**: AI-03 — Decide the v1 capability set and optional-capability discovery
> **Node**: AI-03.1 — The capability matrix `[decision]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: documentation only — `openspec/changes/cachicamas-ai-minimum-capabilities/`, plus a three-row amendment to AI-01's register. **Zero Go. Zero `go.mod` change. Zero files under `backend/`.**
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-01 (`cachicamas-ai-contract-vocabulary`), AI-02 (`cachicamas-ai-stream-lifecycle`)
> **Blocks**: AI-20.5, AI-23, AI-24 — and, through them, AI-29.0, AI-38, AI-40
> **Authoring constraint**: doc 0002's authoring constraint binds every artifact of this change. No Go identifier appears anywhere.

---

## Intent

Close doc 0002's AI-03.1 closing checklist — required capabilities, optional capabilities with their reasons, exclusions with their reasons, the discovery mechanism, and the capability record — in one merged artifact, so that **AI-23's conformance suite can mark every case required or optional from this list alone**, and so that a provider lacking an optional capability is fully conformant and records "absent" rather than skipping silently.

AI-03 is the last node of wave 0. After it, the vocabulary is fixed (AI-01), the stream's physics are fixed (AI-02), and the definition of a conformant adapter is fixed (AI-03). Wave 1 can then build contracts without any of the three being reopened.

The milestone is scheduled here rather than beside the suite for a reason worth stating: the required list **is** the conformance suite's definition of correctness. Promoting a capability from optional to required after an adapter exists invalidates that adapter's conformance; demoting one after the suite exists invalidates the suite's claim to define correctness. Both are cheap today and expensive from AI-23 onward.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can tell the inherited from the decided.

1. **The vocabulary is AI-01's.** `V-PRV-06` … `V-PRV-09` were assigned to AI-03 by AI-01 itself. Definitions are cited, never paraphrased.
2. **The discovery mechanism *family* is inherited, not chosen.** doc 0002's checklist item 4 states it: an optional capability is an additional, separately-asserted contract on the provider value, so the core provider interface never widens. Unlike AI-02's carrier, this is not a free choice. What is free is everything inside the family, and the proposal decides five such questions.
3. **Token counting is optional and discovered by assertion on the provider value.** doc 0001 § 6 seam 6, doc 0001 § 7 **G3**, ADR 0005 § D4 row G3 and doc 0002's charter note all state it. This proposal supplies the argument, not the verdict.
4. **Cancellation and typed failure delivery already have fixed observable shapes.** AI-02 §§ 5 and 7. This decision marks them required; it does not re-decide them.
5. **Multimodal content beyond text is a v1 non-goal.** doc 0001 § 8 and doc 0002's deferred list, which already names AI-03.1 as the node that records the exclusion.
6. **Layer 1 never executes a tool, resolves a tool name, or decides whether a call may run.** AI-01's trap 1, `V-OUT-04`, `V-OUT-05`, `V-OUT-16`.
7. **No Go identifiers, no production code.** doc 0002's authoring constraint and its `[decision]` node grammar.

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `explore.md` | The admission tests; every candidate run through them; the nine-row leakage cross-check; the five open discovery sub-questions; the three vocabulary gaps |
| `proposal.md` | This file |
| `specs/ai-minimum-capabilities/spec.md` | `R-AIC-001` … `R-AIC-015`, each a checkable property of the decision artifact |
| `design.md` | The structure `decision.md` implements, and the reasoning rules it applies |
| `tasks.md` | The single leaf AI-03.1, one task per closing-checklist item, plus the register amendment and the verification pass |
| `decision.md` | **The deliverable.** The capability matrix, the discovery mechanism, the record shape, and what each blocked milestone inherits |
| AI-01's `decision.md` | **Amended**: three appended `V-PRV` rows and their dated amendment blockquote, per AI-01 § 9 rule 2 |

### The five decisions this proposal commits to

One line each, so a reviewer can accept or reject the substance before reading the argument.

1. **Five required capabilities, closed** — `CAP-R-01` streaming text, `CAP-R-02` tool calls, `CAP-R-03` completion metadata, `CAP-R-04` cancellation, `CAP-R-05` typed failures with the partial-output distinction. Each carries one precision that stops an adapter being failed for the wrong reason — most importantly that requiring the *usage record* is not requiring a *populated count*, and requiring the finish-reason vocabulary to be reachable is not requiring every value to be emitted. Requiring either would force a fabrication, which is the same defect the token-counting argument turns on.
2. **Three optional capabilities, closed** — `CAP-O-01` reasoning content (providers differ irreconcilably, and AI-29.0 is explicitly authorized to record an absence), `CAP-O-02` token counting (the load-bearing case: a required count forces every adapter to implement or fabricate, and a fabricated count corrupts compaction silently where an absent one degrades visibly), `CAP-O-03` honoring cache-boundary markers (markers are advisory by contract, and an auto-caching provider correctly ignores them wholesale). A fourth arrives by amendment, not by improvisation.
3. **Four exclusions, each with its reason** — `CAP-X-01` multimodal content beyond text, `CAP-X-02` embeddings, `CAP-X-03` batch APIs, `CAP-X-04` server-side tool execution. The line between excluded and optional is stated as a rule: **an optional capability has a defined absence; an excluded one has no defined presence.**
4. **Discovery — one separately-asserted contract per capability, declared in Layer 1, enumerable, asked of the provider value.** An adapter advertises by satisfying the additional contract and by nothing else; an adapter without the capability writes nothing at all. A consumer asks at the point of use and observes either the capability or a clean absence — not an error, not a zero. Advertising binds: an advertised capability that declines is non-conformant, not absent. Layer 1 supplies no fallback estimate, ever.
5. **A total capability record with a four-value outcome set** — *satisfied*, *absent*, *failed*, *not exercised*. One entry per capability in the closed list, in every record, carrying the capability, its standing (from this decision, not from the run) and one outcome. **Absent and not exercised are different values**, which is the mechanical form of `V-PRV-09`, and a record containing any *not exercised* entry is inconclusive rather than passing.

### Two rules the decision establishes beyond the checklist

Both are needed for the acceptance criterion and neither is a checklist item, so they are called out here rather than discovered in the artifact.

- **The default-required marking rule.** *A conformance case is optional if and only if the capability it exercises appears in the closed optional list; every other case is required.* doc 0002's acceptance criterion — "AI-23's suite can mark each case required or optional from this list alone" — is not satisfiable from the required list, because the required list names five capability-shaped behaviors while the contract has many more required cases (ordering invariants, ownership, the pre-stream contract, redaction). It is satisfiable from the optional list plus a default, and the default must be *required* so that an unclassified case fails loudly instead of being skipped.
- **The forwarding rule for wrapped providers.** A wrapper that satisfies the core provider contract and forwards no optional contract silently removes every optional capability of the value it wraps — and the removal is invisible, because absence is legitimately silent. No source document mentions this; it is a direct consequence of the mechanism, it constrains AI-37 and every Layer 2 decorator, and it is cheaper to state now than to find in Layer 2.

### Out of scope (explicit, and deferred-but-related is called out)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| The declaration of the provider contract or of any optional contract | AI-20 | AI-20 declares; this decision states what must be expressed and documented |
| Every conformance case | AI-23 | This decision supplies the list and the marking rule, not the assertions |
| Which optional capabilities the first vendor has | **AI-24.1 item 4** | Deferred deliberately — no vendor is chosen. Guessing here would create an expectation AI-38.2 asserts against |
| Whether the first adapter emits reasoning | **AI-29.0** | This decision makes both emission and a recorded absence legal, which is its entire contribution there |
| Failure categories, retryability, terminal payload shape | AI-19 | This decision requires typed failures; it defines no category |
| The finish-reason vocabulary and the usage record | AI-13 | Required here, defined there |
| Cache-boundary markers, their cap, the invalidation cascade | AI-11 | Honoring them is the capability; expressing them is AI-11's contract |
| Compaction, estimation, and any consumer-side fallback | Layer 2 (`V-OUT-06`) | Layer 1 states the absence and supplies no substitute. This is the token-counting argument applied to Layer 1 itself |
| The record's serialised form | AI-23.6 emits, AI-40.2 publishes | The shape and the outcome set are decided; a format is not |
| Multimodal support of any kind | after v1 | `CAP-X-01`. Enabling it requires the per-provider capability-detection model v1 does not have |

## Amendment to AI-01's register (scoped, three rows)

AI-01 § 9 rule 2: a missing term is appended there, next free ordinal in its category, dated amendment blockquote, in the same pull request that needs it. AI-02 set the precedent with two `V-STR` rows; this change appends three `V-PRV` rows.

| New id | Term | Why AI-03 cannot proceed without it |
| --- | --- | --- |
| `V-PRV-16` | **capability** | AI-01's own § 7 preamble lists the terms "AI-03's charter is not writable without" and names five — `capability`, `required capability`, `optional capability`, `capability discovery`, `capability record`. The table delivers the last four. Without the bare noun there is no way to say that a contract obligation, an adapter-local mapping obligation, and a contract property optional for everyone are *not* capabilities — which is the distinction that keeps the optional list from absorbing all nine leakage rows |
| `V-PRV-17` | **token counting** | The phrase appears inside `V-OUT-06`'s definition — "Layer 1's only obligation is that token counting is discoverable and optional" — without being defined. Undefined, it collapses into `V-MET-09` **usage**, and the collapse is not cosmetic: usage reports what a response consumed *after* the fact, while counting answers what a request would consume *before* it is sent. A consumer that substitutes the second for the first has no figure at all at the moment it needs one |
| `V-PRV-18` | **capability outcome** | The word "outcome" already appears inside `V-PRV-09`'s definition, undefined — the same situation `V-STR-23` **backpressure** was appended for. The distinction `V-PRV-09` exists to protect (a recorded absence is not an unrun case) is a distinction *between outcome values*, so it cannot be stated without the noun |

All three are provider-surface terms, all three take the next free ordinals after `V-PRV-15`, and all three are owned by AI-03. Each definition defers its substance to AI-03 the way `V-PRV-08` defers the discovery mechanism, so the amendment settles words without pre-empting this decision. No existing row is edited, renumbered or reworded; AI-01 § 9 rule 3 holds.

## Approach

1. **Decide by a stated test, not by intuition.** Four admission tests (`explore.md` § 4) classify a candidate as not-a-capability, required, optional or excluded. Every entry in every list is justified by naming the test clause it satisfies, and every rejected candidate by naming the clause it fails. A list nobody can extend consistently is a list that will be extended inconsistently.
2. **Attack the required list, not only the optional one.** The expensive defect is a required capability that forces a fabrication. Two are inside `CAP-R-03` — a populated token count and an emitted finish-reason value — and both are the token-counting argument at smaller scale. Each required capability therefore carries an explicit precision naming what it does **not** require.
3. **Run all nine leakage rows through the test.** doc 0001 § 3.3 is nine documented ways providers differ, so it is the natural source of false optional capabilities. The finding — nine rows produce zero optional capabilities on their own, with a capability adjacent to two of them — is recorded as a rule, because "a documented provider divergence is evidence for an adapter's mapping table, not for the optional list" is the mistake a later reader will make.
4. **State the mechanism's asymmetry and then its consequences.** Absence costs an adapter zero lines. That is the mechanism's virtue and the source of its two sharp edges: an advertised capability that declines, and a wrapper that forwards nothing. Both are stated as rules.
5. **Make the record total before making it detailed.** Totality over a closed list is what makes a missing entry a defect; the four-value outcome set is what makes absence a conclusion. Everything else about the record — its format, its detail — is deferred.
6. **Close by inheritance.** The artifact ends with what AI-20.5, AI-23 and AI-24 each receive, in that milestone's own terms, so the acceptance criterion is checkable from one table.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-minimum-capabilities/` | Six new markdown files | None — new directory |
| `openspec/changes/cachicamas-ai-contract-vocabulary/decision.md` | Three appended rows, one dated blockquote, two counts updated | Low — append-only, no existing row touched |
| `backend/agent/` | **None** | — |
| `go.mod`, `go.work`, `docker-compose.yaml`, `infra/` | **None** | — |
| doc 0002 | **None.** Every list this decision closes was already anticipated by AI-03.1's checklist; no node is added, struck or renumbered | — |

## Rollback plan

The change is additive documentation with one append-only amendment. Rollback is `git revert` of the single commit; nothing is generated from these files, nothing imports them, and no build depends on them.

Partial rollback has the same shape AI-02 recorded and the same answer: reverting only the three appended register rows would leave `decision.md` citing identifiers that do not resolve. If the amendment is rejected in review, the correct move is to reject the whole change and re-propose, not to strip the rows.

Post-merge reversal is the expensive direction and is the reason for the schedule. Once AI-20.5 implements the mechanism, AI-23 marks every case against these lists and AI-24.1 records an expectation in this record's vocabulary, changing a standing changes a suite, an adapter's conformance claim and a published matrix at once.

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| A later milestone promotes token counting to required, "because compaction needs it" | Medium | **High** — every future adapter then implements it or fabricates it, and a fabricated count corrupts compaction silently | The argument is stated in full, including why the first reading (loop-necessity) is correct and still insufficient; the corollary that Layer 1 supplies no fallback estimate is stated as a standing rule. `R-AIC-005` |
| The optional list is read as a list of "things some providers do", and grows to absorb the leakage register | Medium | Medium — every entry costs a suite case, a record entry and a recorded expectation | The nine-row cross-check is in the artifact with its verdict per row, and § 2's three near-neighbours are named. `R-AIC-007` |
| "Absent" and "not exercised" collapse into one mark in the record | Medium | **High** — this is the exact failure `V-PRV-09` exists to prevent, and a skipped case then reads as a conformance pass | Four-value outcome set, with the verdict rule making any *not exercised* entry inconclusive. `R-AIC-012`, `R-AIC-013` |
| AI-23 cannot mark a case that exercises no listed capability | Medium | High — the acceptance criterion fails | The default-required marking rule, stated as a rule rather than left as an inference. `R-AIC-009` |
| Honoring cache-boundary markers is later reclassified as adapter-local, or markers themselves are read as optional | Low | Medium | Both directions addressed: the breakpoint cap is what makes honoring consumer-visible, and the artifact states that expressing markers is AI-11's contract for every consumer against every provider. `R-AIC-008` |
| An optional contract is folded into the core provider contract for convenience | Low | High — the promise the mechanism exists to keep | AI-20.5 item 3's pin is named as the mechanical form of the promise, and the aggregate-contract alternative is named and rejected. `R-AIC-010` |
| A wrapper silently removes an optional capability | Medium | Medium — invisible by construction, and AI-37 is the first wrapper | The forwarding rule is stated, with AI-37 named. `R-AIC-011` |
| Over-reach into AI-19, AI-20, AI-23, AI-24 or AI-29 | Medium | Medium | AI-01 § 9 rule 5's deletion test applied literally: if a sentence were deleted, would a later milestone have more options? `R-AIC-014` |

## Dependencies

- **AI-01** (`cachicamas-ai-contract-vocabulary`) — **hard.** `V-PRV-06` … `V-PRV-09` were reserved for this milestone, and this change amends the register.
- **AI-02** (`cachicamas-ai-stream-lifecycle`) — **hard.** `CAP-R-04` and `CAP-R-05` are marked required with the observable shapes AI-02 §§ 5 and 7 already fixed. AI-02's own inheritance table anticipates this: "Cancellation and typed failure delivery are required capabilities whose observable shape is already fixed by §§ 5 and 7, so the matrix can mark them without re-deciding them."
- No new Go dependency. No ADR required: the module stays dependency-free until AI-24, and this change adds nothing.

## Success criteria

1. All five closing-checklist items of AI-03.1 are answered in `decision.md`.
2. Five required capabilities are enumerated, each with the precision that names what it does **not** require.
3. Three optional capabilities are enumerated, **each with the reason it is optional rather than required**, and the list is stated to be closed with an amendment route.
4. Four exclusions are enumerated, each with its reason, and the excluded-versus-optional line is stated as a rule.
5. The token-counting argument is present in full, including the corollary that Layer 1 supplies no fallback estimate.
6. The discovery mechanism states **both** how an adapter advertises and how a consumer asks, plus the four alternatives it excludes.
7. The capability record's shape is sketched, is total over the closed lists, and carries a four-value outcome set in which *absent* and *not exercised* are distinct.
8. The default-required marking rule is stated, so AI-23 can mark every case from this list alone.
9. AI-20.5, AI-23 and AI-24 each have a stated inheritance, in that milestone's own terms.
10. The change adds six markdown files and amends exactly one existing file, append-only. No Go identifier appears anywhere.

## Notes for the following phases

- **`spec.md`** — the system under test is the artifact, as it was for AI-01 and AI-02. Requirement IDs `R-AIC-0NN`, scenario IDs `S-AIC-0NN`. Every scenario must be checkable by inspection, without running anything. Several requirements constrain the *argument* rather than the conclusion — a required capability's precision, an optional capability's reason — because a list of verdicts with no reasons cannot be extended consistently and would pass a spec that checked only verdicts.
- **`design.md`** — owns the structure of `decision.md` and the three reasoning rules it applies: the admission tests, the fabrication test applied to the required list, and the closed-list rule that makes the record total.
- **`tasks.md`** — five tasks, one per closing-checklist item, plus the register amendment and the verification pass.
- **`decision.md`** — the deliverable. Ends with the inheritance table, because the acceptance criterion is stated in terms of what AI-23 can do without reopening this milestone.
