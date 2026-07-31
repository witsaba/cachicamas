# Proposal — Layer 1 contract vocabulary

> **Change**: `cachicamas-ai-contract-vocabulary`
> **Milestone**: AI-01 — Record the Layer 1 contract vocabulary
> **Node**: AI-01.1 — The vocabulary `[decision]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: documentation only — `openspec/changes/cachicamas-ai-contract-vocabulary/`. **Zero Go. Zero `go.mod` change. Zero files under `backend/`.**
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-00 (`cachicamas-agent-module-scaffold`)
> **Blocks**: AI-02, AI-03, and every contract milestone AI-04 … AI-40
> **Authoring constraint**: doc 0002's authoring constraint binds every artifact of this change. The vocabulary is conceptual; no Go identifier appears anywhere.

---

## Intent

Layer 1 does not exist yet. Not one file of `backend/agent/src/ai/` is written, and doc 0002 is deliberately authored before the first file so that every contract is built correct at birth rather than corrected afterwards. AI-01 is the first content milestone in that sequence, and its entire product is a set of nouns.

The reason it comes before AI-02 and every contract milestone is recorded evidence, not preference. The retired 2026-07-30 plan produced four self-contradicting contracts (**C1**–**C4**) and four breaking-change gaps (**G4**, **G9**, **G12(b)**, **G12(c)**). Every one of the eight is, underneath, a contract frozen around a noun whose definition was still open — "content part" without readability (**C2**), "content part" without sealing (**C1**), "sequence" defined before "stream" (**C3**), "terminal error event" declared before "provider failure" had a constructible referent (**C4**), and four gaps whose carrier noun simply did not exist yet. `explore.md` § 2 traces each.

This change records the vocabulary: one definition per term, one owning milestone per term, and an explicit register of the terms that are deliberately *not* Layer 1's, each with the layer that owns it. It also restates verbatim the two wording traps from doc 0002's *Layer boundary* section, because both have already caused one wrong decision each.

**Why now.** AI-01 costs one document and blocks nothing else in the schedule; it is the cheapest milestone in Layer 1. After the first contract milestone merges, a term that turns out to be missing or double-owned is renegotiated inside a pull request under implementation pressure — which is exactly how C1 through C4 were produced. doc 0002 prices the alternative: each of the four gaps became "roughly three times the work" once a surface froze around it.

**Why a decision artifact rather than package documentation.** Package documentation cannot exist yet; there is no package. More importantly, doc 0002's node grammar makes a `[decision]` leaf a first-class node with a closing checklist and a merge gate, so the vocabulary is reviewed as a decision rather than absorbed as prose into a milestone that also ships behavior.

---

## Locked constraints (inherited, not proposed)

These are settled upstream. This proposal records them for traceability; no later phase re-litigates them.

1. **No Go identifiers.** doc 0002's authoring constraint: the document "never invents Go type names, field names, or signatures — each milestone's SDD cycle owns those." AI-01's charter repeats it as an explicit out-of-scope clause.
2. **No production code.** AI-01.1 is a `[decision]` leaf. doc 0002's node grammar: a decision leaf is "a recorded choice with a closing checklist. No production code."
3. **The module stays dependency-free.** `go.mod` carries zero requires from AI-00 until AI-24 selects a transport. This change touches no module file at all.
4. **Append-only identifiers.** doc 0002's milestone and node identifiers are append-only. The vocabulary adopts the same discipline for term identifiers.
5. **Amendment convention.** Doc 0002's living-graph clause: a dated blockquote under the touched heading, struck-through text for superseded claims, never a silent edit.
6. **Language.** All artifacts are English, neutral professional register.

---

## Scope

### In scope — one PR, documentation only

| Artifact | Purpose |
| --- | --- |
| `explore.md` | Why naming late is the failure mode; where each term comes from; the trap terms; structural options considered |
| `proposal.md` | This file — intent, scope, rollback, out-of-scope |
| `specs/ai-contract-vocabulary/spec.md` | RFC 2119 requirements `R-AIV-0NN` and verifiable artifact properties `S-AIV-0NN` |
| `design.md` | How the register is structured, and why one definition + one owning milestone per term prevents the class of argument it targets |
| `tasks.md` | The single leaf AI-01.1 as the phase, with the closing checklist as its task list |
| **`decision.md`** | **The deliverable — the vocabulary itself** |

Six markdown files. No other path in the repository is created, modified, or deleted.

### Out of scope (explicit, and deferred-but-related is called out)

| # | Excluded | Owner / when |
| --- | --- | --- |
| 1 | Any Go type, field, method, or package name | Each milestone's own SDD chooses spellings — AI-01 charter |
| 2 | Any production code, any test, any `go.mod` or `Makefile` change | Nothing in Layer 1 exists to change; AI-24 is the first milestone permitted to add a dependency |
| 3 | Choosing the stream **carrier** (receive-only channel versus range-over-func iterator) | **Deferred but related** — AI-02.1 item 1. This change defines the noun so AI-02 can argue about it on its merits, which doc 0002 states is a genuinely free choice there and only there |
| 4 | Stream ownership, cancellation semantics, buffer capacity, and the sanctioned loss path | **Deferred but related** — AI-02.1 items 2–4. Nouns only here |
| 5 | The required/optional capability matrix and the discovery mechanism | **Deferred but related** — AI-03.1. Nouns only here |
| 6 | The content-part strategy that reconciles readability with sealing | **Deferred but related** — AI-06.1. This change records that both properties are constitutive; it does not choose the mechanism |
| 7 | Validation-error granularity, aggregate-versus-short-circuit, positional-context shape | **Deferred but related** — AI-04.1 items 2–4. This change fixes only the caller-contract / provider-transport boundary as a definitional line |
| 8 | The finish-reason mapping table for any concrete provider | AI-31.1 |
| 9 | Retry policy, backoff mechanics, redaction sweep, observability attributes | AI-35, AI-36, AI-37 |
| 10 | Amending doc 0001 or ADR 0005 | doc 0002 already records ADR 0005's Context/Migration narrative as stale and amending it as a separate change |
| 11 | Defining any Layer 2 or Layer 3 concept | Named with an owner in the exclusion register; never defined |
| 12 | Mirroring the vocabulary into package documentation | AI-40.3 at the v1 freeze, if it is wanted then |

---

## Decisions taken in this proposal (ratifiable in review)

1. **The register carries six categories, not four.** AI-01.1's checklist enumerates request-side, stream-side, metadata, failure, and excluded. A sixth — provider surface and proving apparatus — is added because AI-01's *acceptance* criterion is stronger than its checklist: "every subsequent milestone's charter can be written using only these terms." AI-03, AI-21, AI-22 and AI-23 charters are not writable without **capability**, **capability discovery**, **fake provider**, **stream test kit** and **conformance suite**.
2. **Every term gets a stable identifier** (`V-REQ-nn`, `V-STR-nn`, `V-MET-nn`, `V-FAIL-nn`, `V-PRV-nn`, `V-OUT-nn`), append-only, so later charters and SDDs cite a term rather than re-paraphrasing it, and so an amendment appends rather than renumbers.
3. **Ownership is a column, not prose.** Exactly one owning milestone per term. Double ownership is then visible by inspection rather than discovered when two milestones ship contradicting definitions.
4. **`call ordinal` is grouped stream-side, per the closing checklist, but owned by AI-09.** The checklist lists it among stream-side terms; the concept originates in the tool-call content contract and is restated at AI-18.3 and AI-30.5. One definition, one owner, grouped where the checklist puts it, cross-referenced where it recurs.
5. **The vocabulary defines the caller-contract / provider-transport boundary but not the taxonomy inside either side.** The boundary is definitional and belongs to a vocabulary; granularity and aggregation are design decisions that belong to AI-04.1 and AI-19.2.
6. **The two wording traps are quoted verbatim**, not paraphrased. Paraphrase is how "Layer 1 does not know what a tool is" became too broad in the first place.

---

## Approach

1. Walk doc 0002's charters AI-02 … AI-40 and collect every concept a charter, closing checklist, or test list names.
2. Assign each concept to exactly one owning milestone, using the charter that *defines* it rather than any charter that merely uses it.
3. Write one definition per concept, stated as what the concept **is** and what it deliberately is **not**, in conceptual language.
4. Attach provenance: the doc 0001 section a term derives from, and the `C1`–`C4` / `G1`–`G13` identifier where the term exists to close a specific defect or gap.
5. Build the exclusion register from doc 0001 §§ 4.1, 4.2, 4.3, 5.1, 5.2 and § 7, naming the owning layer for each excluded term.
6. Quote the two wording traps verbatim from doc 0002's *Layer boundary*.
7. Verify the artifact against AI-01.1's six checklist items, one by one, in the tasks phase.

---

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-contract-vocabulary/` | six new markdown files | none |
| `backend/agent/` | **untouched** | none |
| `backend/database_administrator/`, `backend/workspace_syncer/` | **untouched** | none |
| `docs/` | **untouched** — cited, not edited | none |
| `go.mod` (any module) | **untouched** | none |
| Build, tests, containers | **unaffected** — nothing compiles differently | none |

---

## Rollback plan

The change adds six markdown files and modifies nothing. Rollback is correspondingly cheap, and is stated at three levels because the third one is the interesting case.

**Level 1 — revert before anything depends on it.** `git revert` the merge commit, or delete `openspec/changes/cachicamas-ai-contract-vocabulary/`. No build, test, container, or migration is affected, because none was touched. Recovery time is one commit. This is the correct move if AI-02 has not yet started.

**Level 2 — amend rather than revert, once downstream milestones cite terms.** Once AI-02, AI-03 or any AI-04+ charter cites a `V-*` identifier, a wholesale revert would strand those citations. The supported correction is doc 0002's living-graph convention applied to the vocabulary: append a dated blockquote under the touched term, strike through the superseded definition, and land the amendment **in the same PR** that needs it. Term identifiers are never reused and never renumbered. A term that turns out to be missing is appended with the next free ordinal in its category.

**Level 3 — supersede the artifact wholesale.** If the categorization itself proves wrong — for example, if AI-06.1's chosen part strategy makes "content part" and "content-part kind" one concept rather than two — the artifact is superseded by a new decision node appended under AI-01 (next free ordinal, `AI-01.2`), with an edge recorded in doc 0002 per the living-graph clause. The original `decision.md` stays in place with its superseded sections struck through, so links from merged charters keep resolving. This is the only rollback path that requires editing doc 0002, and doc 0002's own clause already prescribes it.

**Blast radius, worst case:** every downstream artifact affected is itself a markdown document. No runtime, no data, no build. There is nothing to migrate and nothing to restore.

---

## Risks

| # | Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- | --- |
| 1 | A term is missing and gets invented inside a later PR instead of appended by amendment | medium | high — it is the exact failure C1–C4 came from | AI-01's acceptance criterion is written as a rule the reviewer applies: a missing term is appended by amendment. `spec.md` `R-AIV-011` states it normatively; every downstream SDD inherits it |
| 2 | Two milestones both believe they own a term | low | high | Ownership is a single column with exactly one value; `S-AIV-003` makes duplicate ownership a checkable property of the artifact |
| 3 | A Go identifier leaks into the vocabulary and pre-empts a milestone's naming | medium | medium | `R-AIV-009` forbids it normatively; `S-AIV-009` is the review check. The register uses noun phrases with spaces, which makes a leaked identifier visually obvious |
| 4 | The vocabulary over-reaches and decides something AI-02, AI-03, AI-04 or AI-06 owns | medium | medium | The out-of-scope table above names all four explicitly with the node that owns each; `design.md` § 5 states the noun-versus-decision line |
| 5 | The artifact is written and never read | medium | medium | Every downstream milestone's charter is required to be expressible in these terms (`R-AIV-012`), which makes the vocabulary load-bearing for AI-02's own SDD immediately, not eventually |
| 6 | doc 0001 uses retired milestone identifiers (AI-40 … AI-47) that no longer mean what they say | **high** | high — a wrong citation propagates into every later charter | Every citation in `decision.md` is translated through doc 0002's identifier map and expressed in current identifiers. `S-AIV-008` checks it |

---

## Dependencies

- **AI-00** merged: `backend/agent/` exists with both import guards. AI-01's charter lists it as the only dependency. Nothing in this change writes into the module, but the milestone ordering is doc 0002's.
- **doc 0002** merged (2026-07-30) — the milestone charters this vocabulary assigns ownership from.
- **doc 0001** merged — the architectural source of the concepts and of the `C1`–`C4` / `G1`–`G13` identifiers.
- **ADR 0004 as amended by ADR 0005** — the layer boundary the exclusion register enforces.
- No new tooling, no new dependency, no ADR required by this change.

---

## Success criteria

1. All six of AI-01.1's closing-checklist items are answered in `decision.md`, verifiable item by item.
2. Every term in the register has exactly one definition and exactly one owning milestone.
3. Every one of the nine terms AI-01.1 item 5 names as excluded appears in the exclusion register with a named owner.
4. Both wording traps appear verbatim.
5. No Go identifier appears in any artifact of this change.
6. AI-02's and AI-03's charters, re-read after this change, are expressible using only these terms — the concrete instance of AI-01's acceptance criterion, and the handoff the AI-02 and AI-03 SDDs consume.
7. `git status` after the change shows six added files and nothing else.

---

## Notes for the following phases

- **`sdd-spec`**: requirements are properties of the **artifact**, not of runtime behavior. There is no runtime. Scenarios are Given/When/Then over the document — "given the register, when a reviewer selects any row, then it names exactly one owning milestone."
- **`sdd-design`**: the design question is why a register with one owner per term prevents an argument class, not how to implement anything. Include the register's structural contract and the amendment protocol.
- **`sdd-tasks`**: exactly one phase — AI-01.1 — whose task list is the closing checklist. No PR chain; the forecast is one documentation PR well under the 250-line preference for *code*, and reviewed as prose.
