# Archive report — the v1 capability set and optional-capability discovery

> **Change**: `cachicamas-ai-minimum-capabilities`
> **Milestone**: AI-03 of [doc 0002](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-03--decide-the-v1-capability-set-and-optional-capability-discovery) — Decide the v1 capability set and optional-capability discovery
> **Node**: AI-03.1 — The capability matrix `[decision]`
> **Phase**: archive
> **Status**: **ARCHIVED**
> **Date**: 2026-07-31
> **Pull request**: #95 (`feat/2026-07-31-cachicamas-ai-layer1-wave-0` → `main`)
> **Merge commit**: `a831c06` · **Base**: `origin/main` @ `b6c59e6`
> **Change commit on `main`**: `d4ea0d7` — the last node of wave 0 — corrected by `96af943` before merge. The verify report cites the pre-rebase hash `f701e58`.
> **Closes**: the Layer 1 half of the concern doc 0001 and ADR 0005 track as **G3**
> **Verify verdict**: **PASS** — see [`verify-report.md`](verify-report.md)
> **Canonical spec**: [`openspec/specs/ai-minimum-capabilities/spec.md`](../../../specs/ai-minimum-capabilities/spec.md)

---

## 1. Charter acceptance

| # | Charter clause | Outcome |
| --- | --- | --- |
| 1 | A recorded capability matrix with a **required/optional column** and a **discovery mechanism** | **PASS** — three closed lists (5 required, 3 optional, 4 excluded) with stable identifiers, plus § 9's mechanism |
| 2 | **AI-23's suite can mark each case required or optional from this list alone** | **PASS**, and passed by a route the checklist does not name |
| 3 | A provider lacking an optional capability is **fully conformant** and records "absent" rather than skipping silently | **PASS** — `absent` is one of four closed outcome values, and the verdict rule makes it a pass while making `not exercised` inconclusive |
| 4 | doc 0002's note: token counting is optional and discovered by assertion on the provider value, **not part of the provider interface** | **PASS** — § 9 keeps the core contract unwidened and names AI-20.5 item 3's pin as the mechanical form |

Clause 2 is the one that determines whether the artifact works, and it was met by identifying that the obvious reading is unsatisfiable. The required list is five capability-shaped entries while the contract has many more required cases — every ordering invariant, every ownership rule, the pre-stream contract, redaction, translation totality. A suite marking cases by lookup against the required list would leave all of those **unmarked**, and the natural default for an unmarked case is the dangerous one. § 11 supplies what is satisfiable instead: a biconditional over the *optional* list with a **required** default, stated over the capability a case exercises rather than over the suite's node structure.

---

## 2. What was delivered

One `[decision]` leaf: `decision.md`, 581 lines, no production code, no `make test` gate.

**Required — a conformant adapter exhibits every one**

| Id | Capability |
| --- | --- |
| `CAP-R-01` | streaming text |
| `CAP-R-02` | tool calls |
| `CAP-R-03` | completion metadata |
| `CAP-R-04` | cancellation |
| `CAP-R-05` | typed failures with the partial-output distinction |

Each entry carries what it obliges, **what it does not oblige**, and the admission-test clauses that give it its standing. The negative clause is structural rather than editorial: the expensive defect in a required list is not an omission but an entry that forces an honest adapter either to fail conformance for something that is not a defect or to fabricate — and a fabricated answer is indistinguishable from a real one at every layer above. Two such readings hide inside `CAP-R-03` alone, and both are closed explicitly (not every finish-reason value must be emitted; not every token count must be populated). `CAP-R-04` and `CAP-R-05` **cite** AI-02 §§ 5 and 7 for their observable shapes rather than re-deciding them.

**Optional — an adapter that lacks one is fully conformant**

| Id | Capability | Why optional rather than required |
| --- | --- | --- |
| `CAP-O-01` | reasoning content | Providers differ irreconcilably in what they emit, and a consumer that receives none loses nothing it needs for correctness |
| `CAP-O-02` | token counting | A required count forces an adapter whose vendor has none either to fail conformance or to fabricate — and a fabricated count corrupts a compaction decision **silently**, where an absent one degrades to a **visible** estimate |
| `CAP-O-03` | honoring cache-boundary markers | Markers are advisory by contract; an adapter for an auto-caching provider is conformant while ignoring every one of them |

Each also states what a consumer does on a recorded absence, which is what makes optionality survivable rather than merely permitted. "And anything else v1 admits" resolves to **nothing**, with five candidates recorded and the clause each fails.

**Excluded for v1 — not a capability at all, in either list**

`CAP-X-01` multimodal content beyond text · `CAP-X-02` embeddings · `CAP-X-03` batch APIs · `CAP-X-04` server-side tool execution. Each carries "why not optional" as well as "why excluded", under one separating rule: **an optional capability has a defined absence; an excluded one has no defined presence.**

**The four rules**

1. **Discovery** (§ 9) — an optional capability is an additional, separately-asserted contract on the provider value. One contract per capability. The core provider interface never widens. An adapter advertises **by satisfying the contract and by no other means**; an adapter that lacks the capability **declares nothing at all**, which is the asymmetry that removes every incentive to fabricate. A consumer asks the **provider value**, at the point of use, and observes either the capability or a **clean absence** — "not an error and not a zero".
2. **Marking** (§ 11) — a conformance case is optional **if and only if** the capability it exercises appears in the optional list. Every other case is required.
3. **The record** (§ 10) — total over both closed lists, one entry per capability carrying the capability, its **standing** (from this decision, never from the run) and one **outcome** from a closed four-value set: `satisfied`, `absent`, `failed`, `not exercised`. The verdict rule makes a record containing any `not exercised` entry **inconclusive**, which is what stops the four-value set collapsing to three the moment a verdict is computed.
4. **No substitutes** (§ 6.2, § 13 rule 4) — Layer 1 never supplies a fallback for an absent capability. A default that estimates is a fabrication with better provenance.

Two further contributions beyond the checklist. **The nine-row leakage cross-check** (§ 8) runs all nine of doc 0001 § 3.3's documented provider divergences through the admission tests, and the finding is the value: nine documented divergences produce **zero** optional capabilities on their own. Two rows have a capability *adjacent* to a contract item, and in both the capability is the emitting or honoring half, never the neutral shape. **The wrapper forwarding rule** (§ 9) appears in no source document and is a direct consequence of choosing silent absence: a wrapper that satisfies the core contract and forwards nothing silently removes every optional capability of the value it wraps, invisibly. AI-37 is named as the first milestone it binds, before AI-37 writes a wrapper.

### 2.1 Register amendment landed with this change

Three nouns were appended to AI-01's register in this same pull request, under its § 9 rule 2:

| Appended | Owner | Why the register lacked it |
| --- | --- | --- |
| `V-PRV-16` **capability** | AI-03 | Closes a gap **AI-01 identified in its own § 7 preamble**: that preamble names five terms AI-03's charter is not writable without, and the table delivered four |
| `V-PRV-17` **token counting** | AI-03 | The phrase **already appeared inside `V-OUT-06`'s definition undefined**, where it silently collapses into `V-MET-09` **usage** — a report about an output standing in for a question about an input |
| `V-PRV-18` **capability outcome** | AI-03 | The word "outcome" **already appeared inside `V-PRV-09` undefined**, and the distinction that row exists to protect — a recorded absence is not an unrun case — is a distinction *between outcome values* |

Measured: five added lines and one replacement, the register's own arithmetic, with **no existing row renumbered, reworded, reordered or removed**. Register total 111 → 114. All three definitions **defer their substance to AI-03 by name**, in the pattern `V-PRV-08` already established for the discovery mechanism, which is what keeps AI-01 § 9 rule 5 intact — a register row stating which outcome values exist would be AI-01 deciding AI-03's matrix retroactively.

---

## 3. Where the contract now lives

The delta spec `specs/ai-minimum-capabilities/spec.md` and the capability matrix were promoted to [`openspec/specs/ai-minimum-capabilities/spec.md`](../../../specs/ai-minimum-capabilities/spec.md).

That canonical spec **carries the substance in its own text** — `CAP-R-01` … `CAP-R-05` each with its obligations, negative clause and test clauses; `CAP-O-01` … `CAP-O-03` each with its reason and its recorded-absence behavior; `CAP-X-01` … `CAP-X-04` each with its reason and its why-not-optional; the admission tests; the nine-row cross-check; the advertise/ask discovery mechanism with its six rejected alternatives and the wrapper forwarding rule; the marking rule with its seven worked cases; and the capability-record shape with its closed four-value outcome set and verdict rule. It does not merely point at this archive.

The reason is § 13 rule 1. The three lists are **closed but not frozen**: a fourth optional capability arrives by amendment *to this contract*, in the pull request that needs it, having applied § 4's test 2 and § 8's divergence rule. Filing the lists in an immutable archive would have removed the route the decision itself prescribes. AI-20.5, AI-23, AI-24, AI-37 and AI-38 also cite this contract by section number and by `CAP-*` identifier, so both the section numbering `§ 1` … `§ 14` and the entry identifiers are preserved in the canonical spec.

| Artifact | Role |
| --- | --- |
| `openspec/specs/ai-minimum-capabilities/spec.md` | **The contract.** Live, amendable under § 13 rule 1, cited by AI-11, AI-13, AI-19, AI-20.5, AI-23, AI-24, AI-29.0, AI-37, AI-38 and AI-40.2 |
| `openspec/changes/archive/2026-07-31-cachicamas-ai-minimum-capabilities/decision.md` | **The historical record of how the contract was decided** — the same lists and rules as they stood at merge, with AI-03.1's closing-checklist verification. Immutable |

**Deltas promoted**

| Kind | Identifiers |
| --- | --- |
| Requirements | `R-AIC-001` … `R-AIC-015` |
| Scenarios | `S-AIC-001` … `S-AIC-059` |

Change voice was rewritten into standing voice without moving an identifier. `R-AIC-001` now states that exactly one statement of this contract is normative and it is the canonical spec, instead of naming this change's `decision.md` path; `R-AIC-015`'s scope fence and vocabulary discipline now bind every amendment rather than only this change; `S-AIC-012` now points at the canonical stream-lifecycle spec for the cited observable shapes; `S-AIC-059`'s diff-hygiene scenario now reads against "any change that states or amends this contract".

---

## 4. Findings recorded at verify, and their disposition

`verify-report.md` § 7 records three. None blocked the verdict.

| # | Finding | Disposition |
| --- | --- | --- |
| 7.1 | § 3 opens with a block quotation of `V-PRV-16`'s definition — the exact shape a violation of the "no local definition of an appended noun" rule would take | **Holds.** Compared word for word against the register: identical, and § 3 frames it as a quotation (*"`V-PRV-16` names the unit. Its content, for this decision's purposes"*). `CAP-O-02`'s opening does the same for `V-PRV-17`. Recorded because verifying it required a text comparison rather than a reading |
| 7.2 | **MINOR** — seven cross-references pointed at unnumbered headings. `decision.md` cited its own `§ 6.1`, `§ 6.2` and `§ 6.3` seven times, including § 1's reading guide, while those three subsections were headed by their capability identifier alone. Every reference was correct **by position**, but § 1 sends a reviewer to "§ 6.2", which could not be found by searching for that string | **Fixed before merge**, in commit `96af943`. The three subsection headings are now numbered **6.1** (`CAP-O-01`), **6.2** (`CAP-O-02`) and **6.3** (`CAP-O-03`), so all seven references resolve by search as well as by position. § 6.4 is unchanged, and § 10's similar asymmetry was left as-is because § 10's siblings are not cross-referenced by number, so numbering them would add noise without removing a defect. The numbering is carried into the canonical spec |
| 7.3 | `CAP-R-01` … `CAP-X-04` are twelve identifiers this decision introduces that the register does not carry | **Not a violation.** They are **entry identifiers for lists this decision owns**, not Layer 1 nouns, and AI-01 § 9 rule 2 binds Layer 1 *nouns*. The nouns the entries are made of — `V-PRV-06` required capability, `V-PRV-07` optional capability, `V-PRV-16` capability — are all in the register and all cited. Recorded once here because AI-23, AI-24 and AI-38 will each cite a `CAP-*` identifier, and what they are citing is a row in **this** contract rather than in the register |

---

## 5. Deliberately not done

Verified absent in `verify-report.md` § 9, each deliberate.

- **Nothing under `backend/`.** The change's commit touches it zero times; the ten `backend/` paths in the wave's range all belong to AI-00.
- **doc 0002 not amended.** No node added, none renumbered, no stated claim corrected.
- **No vendor named or guessed.** § 12: *"Not inherited, and deliberately not pre-empted: which vendor. This decision names none and guesses at none."* Confirmed by inspection — no vendor name appears anywhere in the change. Which capabilities a given vendor has is AI-24.1's.
- **AI-29.0 not pre-empted.** Whether the first adapter emits reasoning or records a capability absence remains AI-29.0's; this decision's only contribution is making **both** answers legal. § 13 rule 7 states the abstention as a rule — *"a sentence here that answered either would delete a decision node"* — and `tasks.md`'s review focus 4 told the reviewer to hunt for a violation of it.
- **No contract declared.** Every declaration — the provider contract, the optional contracts, and their spellings — is AI-20's.
- **No conformance assertion written.** This decision supplies the list and the marking rule, not the cases. Every assertion is AI-23's.
- **No failure category** (AI-19), **no finish-reason value definition** (AI-13), **no marker cap number** (AI-11).
- **No Layer 1 default implementation of an optional capability** — the abstention that follows from `CAP-O-02`'s own argument.

### 5.1 A source defect found and neutralised rather than propagated

§ 8 records that doc 0001 § 3.3's preamble says "Three require a contract change; the rest are absorbed inside an adapter", while its own table marks rows 8 and 9 as carrying a Layer 1 contract half. The artifact resolves it: the preamble counts **G12**'s three rows only, while rows 8 and 9's contract halves belong to **G4** and are scheduled separately at AI-10 and AI-11. It then states the consequence of getting it wrong — *"Reading 'the rest are adapter-local' as covering rows 8 and 9 entirely would delete AI-11 from the plan."*

---

## 6. Cross-artifact consistency at merge

A wave of three coupled decision artifacts fails at the joints rather than in the middle, so `verify-report.md` § 8 checked both directions. All 60 distinct `V-*` citations resolve. `CAP-R-04` reproduces AI-02 § 5's three cancellation obligations exactly and names `V-STR-09`'s sanctioned loss path as the only exception; `CAP-R-05` requires classification through one vocabulary on both delivery paths, which is AI-02 § 7's boundary in substance; `CAP-R-03` is scoped to "a stream that finishes **normally**", leaving AI-02's cancellation ending and its sanctioned bare close untouched; and `CAP-X-03` excludes batch APIs *because* every clause of AI-02's lifecycle assumes a live cancellable stream — an exclusion derived from the predecessor rather than in tension with it. No contradiction was found in either direction.

---

## 7. Lifecycle

`explore → proposal → spec → design → tasks → decide → verify → archive` — all phases delivered. `tasks.md` records six tasks plus a twelve-check verification pass, all `[x]`.

| Phase | File |
| --- | --- |
| Explore | `explore.md` |
| Proposal | `proposal.md` |
| Spec (delta) | `specs/ai-minimum-capabilities/spec.md` |
| Design | `design.md` |
| Tasks | `tasks.md` |
| Decision | `decision.md` — **superseded as the live contract by the canonical spec; retained here as the historical record** |
| Verify | `verify-report.md` |
| Archive | `archive-report.md` (this file) |

**Unblocked by this decision:** AI-20 (`cachicamas-ai-model-provider`, node AI-20.5), AI-23 (`cachicamas-ai-conformance-suite`), AI-24 (`cachicamas-ai-first-provider-decision`) — and, through them, AI-29.0, AI-37, AI-38 and AI-40. Wave 1 (AI-04 … AI-13) depends on nothing decided here.

**Wave 0 closes here.** doc 0002's exit condition for wave 0 — *"The module exists, both import directions bite, and vocabulary, stream lifecycle, carrier and capability scope are recorded decisions"* — holds on this merge.
