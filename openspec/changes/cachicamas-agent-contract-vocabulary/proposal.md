# Proposal — Layer 2 contract vocabulary

> **Change**: `cachicamas-agent-contract-vocabulary`
> **Milestone**: AG-00 — Record the Layer 2 contract vocabulary
> **Node**: AG-00.1 — The vocabulary decision `[decision]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Driver**: braejan
> **Scope**: documentation only — `openspec/changes/cachicamas-agent-contract-vocabulary/`. **Zero Go. Zero `go.mod` change. Zero files under `backend/`.**
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: the doc 0002 wave 2 contract surface — frozen; AI-40 merged `7326a813` on 2026-08-10
> **Blocks**: AG-01, AG-02, AG-03 — and through them every Layer 2 milestone
> **Authoring constraint**: doc 0003's authoring constraint binds every artifact of this change. The vocabulary is conceptual; no Go identifier appears anywhere.

---

## Intent

Layer 2 does not exist yet. `backend/agent/src/agent/` has zero files on disk, and [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) is deliberately authored before the first one so that the seams are built in rather than retrofitted. AG-00 is the first milestone in that sequence and its entire product is a set of nouns. Doc 0003 names it directly: *"the AI-01 of this layer: names before code."*

**Why a vocabulary milestone comes first — the Layer 2 evidence, not Layer 1's.** Layer 1's AI-01 argued from four shipped-then-corrected contract defects (C1–C4). Layer 2 has shipped nothing, so it cannot have repeated them; copying that argument here would be borrowed evidence. The evidence that exists at this layer sits one level up, in the documents, and it is recorded rather than hypothetical:

| # | Recorded correction | What it cost | Source |
| --- | --- | --- | --- |
| 1 | *"the portable brain"* → **"the portable agent runtime"** | A name that described the layer as *thinking* survived for weeks in the architecture reference — inviting exactly the policy-inside-Layer-2 mistake §§ 4.1–4.2 spend two pages forbidding. The correction changed no boundary, no seam, no dependency rule: only a word | doc 0001 § 4 amendment, 2026-08-10 |
| 2 | **"Layer 3" ≠ `cachicamas_coding`** | § 5 read as though Layer 3 *were* the coding agent. Left unfixed, a later milestone could have written a contract, test name, or acceptance criterion in terms of a *coding* agent — a boundary violation with the same weight as an import violation | doc 0001 § 5 amendment, 2026-08-10; doc 0003 wording trap 4 |
| 3 | Four defects in the **plan that names this milestone** — a too-narrow import allowlist, a parallelism claim contradicting its own dependency edge, charter dependencies missing from the node graph, one-way `Blocks:` edges | Found by the 2026-08-11 pre-implementation audit of doc 0003 v1, before any milestone started | doc 0003 § "Sources and research", inconsistency register rows 1, 2, 5, 6 |

Two conflicts are **live in doc 0003 right now**, and neither has a design alternative, so neither can be deferred to AG-01 or AG-02 (§ "Decisions taken", items 4–5 below): the delegation term is spelled three ways across milestones that will ship in different waves, and the turn / provider call relationship is stated only inside AG-00's own charter prose, nowhere citable.

**Why now.** AG-00 costs one documentation PR, depends on nothing unmerged, and blocks AG-01, AG-02 and AG-03 — everything. After AG-03 merges, a term found missing is renegotiated inside a pull request under implementation pressure, which is how the corrections above were produced one level up. The generalization this milestone records is stronger than AI-01's: not only contracts, but *plans about contracts*, drift when a noun's definition is still open when they are written.

**Why a decision artifact rather than package documentation.** There is no package. More importantly, doc 0002's node grammar — which doc 0003 adopts verbatim — makes a `[decision]` leaf a first-class node with a closing checklist and a merge gate, so the vocabulary is reviewed as a decision rather than absorbed as prose into a milestone that also ships behavior.

---

## Locked constraints (inherited, not proposed)

Settled upstream. Recorded for traceability; no later phase re-litigates them.

1. **No Go identifiers.** Doc 0003's authoring constraint: the document "never invents type names, field names, or signatures — each milestone's SDD cycle owns those." A term is a noun phrase with spaces and a definition; it is never an API.
2. **No production code.** AG-00.1 is a `[decision]` leaf: "a recorded choice with a closing checklist. No production code." Zero files under `backend/`, zero `go.mod` change.
3. **Append-only identifiers.** Doc 0003's milestone and node ids are append-only; the register adopts the same discipline for term identifiers — never renumbered, never reused, never reordered.
4. **Amendment convention.** The living-graph clause: a dated blockquote under the touched heading, struck-through text for superseded claims, never a silent edit.
5. **Citation surface.** Citations may point at contract documents, ADRs, or the **shipped Layer 1 surface** — never at Layer 2 code, which does not exist.
6. **Language.** All artifacts are English, neutral professional register.

---

## Scope

### In scope — one PR, documentation only

| Artifact | Purpose |
| --- | --- |
| `explore.md` | Why naming late is this layer's failure mode; the walked term inventory; reuse-vs-wrap grounding; boundary cases; the conflict register |
| `proposal.md` | This file — intent, locked constraints, scope, decisions, rollback |
| `specs/agent-contract-vocabulary/spec.md` | RFC 2119 requirements `R-AGV-0NN` and verifiable artifact properties `S-AGV-0NN`, **and the register itself** — promoted at archive to `openspec/specs/agent-contract-vocabulary/spec.md` |
| `design.md` | How the register is structured, and why one definition + one owning milestone per term prevents the class of argument it targets |
| `tasks.md` | The single leaf AG-00.1 as the phase, with its closing checklist as the task list |
| **`decision.md`** | **The deliverable — AG-00.1's argument and the state of the register on the day it merged** |

Six markdown files. No other path in the repository is created, modified, or deleted.

### Out of scope (explicit; deferred-but-related is called out)

| # | Excluded | Owner / when |
| --- | --- | --- |
| 1 | **Any decision with a design alternative** — AG-00's own out-of-scope clause | AG-01 and AG-02 |
| 2 | The carrier at the agent-event boundary; backpressure/loss posture; the observer decoupling *mechanism*; close/ownership rules; the upward-path surface | **Deferred but related** — AG-01.1 items 1–5. This change fixes the *nouns* ("observer", "upward path", "steering") so AG-01 argues about them on their merits |
| 3 | Every G1/G3/G4/G5/G7/G8/G10/G11 **verdict** (implement-now / seam-with-trivial-impl / deferred) | **Deferred but related** — AG-02.1. This change needs the nouns ("failover seam", "subagent"), never their v1 disposition |
| 4 | The § D3 telemetry attribute vocabulary extension for Layer 2 spans | AG-22.1 — its own `[decision]` node. AG-00's charter deliverable list names no telemetry term; that omission is confirmed correct |
| 5 | Any Go type, field, method, or package name | Each milestone's own SDD chooses spellings |
| 6 | Any production code, test, `go.mod`, or `Makefile` change | Nothing in Layer 2 exists to change |
| 7 | **Defining** any Layer 3 or above concept — permission policy content, sandbox semantics, tool source, summary quality, cross-session rule persistence, price/money, session persistence, frontends, catalogs | Named with an owner in the exclusion register; never defined |
| 8 | **Re-paraphrasing** any Layer 1 term | The live `ai-contract-vocabulary` register owns them; this register cites the `V-*` row |
| 9 | Amending doc 0001 or doc 0003 | The living-graph clause governs those separately; this change cites, never edits |
| 10 | Mirroring the vocabulary into package documentation | AG-23 at the v1 freeze, if it is wanted then |

---

## Capabilities

### New Capabilities

- `agent-contract-vocabulary`: the single definitive register of every noun a Layer 2 milestone charter, SDD, or PR description may use — one definition, one owning milestone, and provenance per term, plus the exclusion register naming what Layer 2 does not own and who does.

### Modified Capabilities

- None. `ai-contract-vocabulary` is **cited, never edited** by this change.

---

## Decisions taken in this proposal (ratifiable in review)

### 1. The register lives in the canonical tree, live and appendable

The register text is authored as this change's spec delta and **promoted at archive** to `openspec/specs/agent-contract-vocabulary/spec.md`, where it stays live. An immutable `decision.md` stays in the archived change folder as the historical record of *how* the register was first decided. This mirrors Layer 1 exactly.

Three reasons, each grounded rather than inferred:

1. `openspec/AGENTS.md` names `openspec/specs/` "source of truth (main specs) — populated as changes land." A register every later milestone must cite belongs there.
2. **Freezing it in the archive would freeze an artifact AG-01 … AG-23 must still write to.** The Layer 1 register was appended to **twice inside its own Wave 0** — `V-STR-22`/`V-STR-23` by AI-02.1 and `V-PRV-16`/`17`/`18` by AI-03.1, taking it from 109 to 114 terms. AG-01 and AG-02 are the exact structural analogues of those two milestones and are the first consumers of this register; the same thing should be expected here, and it only works if the register lives somewhere a later PR can write.
3. `decision.md` and the register answer different questions. One is an argument made on a date; the other is text that must still be correct in six months.

**Amendment rules the register carries** (Layer 1's six standing rules, with layer identifiers substituted):

| # | Rule |
| --- | --- |
| 1 | **Append, never invent.** A Layer 2 noun this register lacks is added *here*, never defined locally in a downstream SDD |
| 2 | **Next free ordinal in its category.** Ids are append-only: never renumbered, never reused, never reordered |
| 3 | **A dated amendment blockquote under the touched category heading**, stating what was appended, by which node, and **why the register lacked the term** — the last part is what makes the rule self-enforcing |
| 4 | **No silent edit.** A superseded definition keeps its identifier with its old text struck through and visible, so citations from merged charters keep resolving |
| 5 | **Update the counts** — per-category figures, the sum, and any extended identifier range |
| 6 | **In the same pull request that needs it** — the amendment lands with the milestone that discovered the gap, not afterwards |

### 2. Identifier shape: `VL2-<CAT>-nn` — disjoint from Layer 1 by construction

Both registers will sit side by side under `openspec/specs/`, and doc 0003 already cites Layer 1 rows (`V-REQ-*`, `V-MET-*`, `V-STR-*`) **inside Layer 2 milestone prose**. Reusing a bare `V-` prefix would make a citation ambiguous about which register resolves it.

| Candidate | Verdict |
| --- | --- |
| `V-<CAT>-nn` (reuse L1's prefix) | **Reject** — a plain-text search for `V-` matches both registers, and a reader's default expectation is that every `V-*` id is Layer 1's |
| `AG-<CAT>-nn` | **Reject** — `AG-` is already the milestone/node namespace of doc 0003 |
| `V2-<CAT>-nn` | **Reject** — textually disjoint, but `2` reads as a *version* of the vocabulary rather than the *layer* it describes, and it forces `V3-` on doc 0004, which reads worse |
| **`VL2-<CAT>-nn`** | **Adopt** — `V` keeps continuity with the register family, `L2` names the **layer**, and the pair is textually disjoint from `V-` so no search or citation can cross registers. It generalizes: Layer 3's register becomes `VL3-` without re-reading as a version bump |

### 3. Six categories, sized to a runtime rather than an adapter

Layer 1's axes (request in, stream out, metadata, failure, provider surface, exclusions) describe a *model adapter*. Layer 2 is a loop plus a harness, so the axes differ. Same count, disjoint content:

| Code | Category |
| --- | --- |
| `VL2-COR` | Core identity — runtime, loop, harness, run, turn, provider call, attempt, transcript, pairing invariant, steering, suspension, delegation, cost scope |
| `VL2-EVT` | The event envelope — kind, the eight families, the four invariants, run and turn outcomes, the stream-contract validator |
| `VL2-LOOP` | Loop mechanics — pre-request hook, prefix stability, tool execution contract, effect class, ordered rejoin, policy slot, permission protocol, finish-reason dispatch, typed turn failure |
| `VL2-HAR` | Harness mechanics — history, orphan synthesis, run driver, interrupt vs shutdown, bounded wind-down, retry policy, failover seam, composed bounds, cost aggregation |
| `VL2-SEAM` | Cross-cutting seams — context strategy, token accounting, compaction and its artifacts, re-entrancy, derived permission scope, hook taxonomy, observer asynchrony, the readiness contract |
| `VL2-OUT` | Excluded — named, attributed to its owning layer, never defined here |

### 4. Conflict resolution — the delegation term

| Side | Says | Where |
| --- | --- | --- |
| A | **subagent** — the event-kind names `subagent-started` / `subagent-ended` are fixed | doc 0003 § AG-06 (node AG-06.3); doc 0003 § AG-02 charter ("no subagent tool ships in v1", R-14); doc 0001 § 6 seam 12 |
| B | **child harness** / **nested run** — "a child harness runs inside a tool", "sibling children interleave", "the child winds down first"; the milestone is titled *delegation readiness*, not *subagent readiness* | doc 0003 § AG-19 scenarios and heading |

**Resolution — one canonical noun, two recorded synonyms, one scope rule:**

- **`subagent`** is canonical for the delegated participant. It wins because it is the word that appears **on the wire**: it is already fixed in the event-kind names AG-06 ships, and an envelope name is a public surface that later milestones cannot cheaply rename.
- **`delegation`** is the term for the *relationship* and for the event family — which is why AG-19's title is correct as it stands.
- **`child harness`** (the re-entrancy mechanism) and **`nested run`** (the run a subagent drives) are recorded as admissible synonyms, each with the sense it carries.
- **Scope rule:** any name that ships — an event kind, a scenario id, a test name, an acceptance criterion — uses **subagent** or **delegation**. Prose may use a synonym only where this register lists it.

Without this, AG-06 (which ships the event-kind names, Wave 1) and AG-19 (which ships the structural proof, Wave 5) land four waves apart using different words for one relationship — the exact defect class this milestone exists to prevent.

### 5. Conflict resolution — turn vs provider call vs attempt

| Side | Says | Where |
| --- | --- | --- |
| A | One loop invocation issues **exactly one** provider call — "the loop never issues a second provider call"; the sequence diagram draws one loop-to-provider interaction per iteration | doc 0003 § AG-11 (node AG-11.2); doc 0001 § 2.3 |
| B | One **turn** can span several provider calls — retry re-invokes the loop for "a fresh provider call over an identical transcript"; cumulative cost "includes the retried attempt's tokens" | doc 0003 § AG-15, § AG-16 (node AG-16.1) |

**Resolution — three rows, not one, because the two sides speak at different scopes and therefore do not actually contradict:**

| Term | Definition | Scope |
| --- | --- | --- |
| **turn** | One assistant response plus its tool results | harness-scoped |
| **provider call** | One Layer 1 stream. Exactly one per loop invocation | loop-scoped |
| **attempt** | A provider call made in service of a turn already begun. Attempts beyond the first exist **only** because the harness re-invoked the loop under retry | harness-scoped |

The reconciling statement, recorded as a citable row rather than left in AG-00's charter prose: *a turn spans one or more provider calls; the count exceeds one only via harness retry; within a single loop invocation the loop never issues a second provider call.* Side A is a statement about the loop; side B is a statement about the turn. AG-00's charter already contains the sentence ("one turn may span several provider calls only via retry") but doc 0001 does not, and no register row carries it — so AG-11 and AG-15 would each re-derive it.

### 6. A fifth boundary case is added to the closing checklist

AG-00.1 item 1 names three boundary cases. A fourth surfaced in exploration and is directly analogous: **is a compaction call a turn?** Answer: **no — it is a provider call but not a turn.** AG-00's own definition of turn (an assistant *response* plus its tool results) does not describe what compaction produces, and doc 0003 consistently calls compaction "a model call with its own provider, cost and cancellation" (AG-18.1), never a turn — but nowhere states the exclusion as vocabulary. It is added rather than deferred because AG-16's cost aggregation and AG-18's mechanics both depend on the answer.

### 7. Three further standing rules the register carries

- **Cross-register citation.** A term Layer 2 reuses unchanged from Layer 1 cites the `V-*` row; it is never re-paraphrased here. A paraphrase is how a definition drifts.
- **The wording traps are quoted verbatim**, not summarized — all four of doc 0003's, following the Layer 1 precedent whose `decision.md` quotes its two "because each has already caused one wrong decision."
- **Exactly one owning milestone per term**, as a column. Double ownership then becomes visible by inspection rather than discovered when two milestones ship contradicting definitions.

---

## Approach

1. Walk AG-00's charter and every AG-01 … AG-23 charter, closing checklist and Gherkin scenario, collecting every concept named.
2. Assign each to exactly one owning milestone — the one that *defines* it, not any that merely uses it.
3. Write one definition per concept, stated as what it **is** and what it deliberately **is not**, in conceptual language.
4. Attach provenance: the doc 0001 section, the doc 0003 requirement (`R-01` … `R-21`) or forward-requirement (`G1` … `G11`), and the shipped Layer 1 file where a reused identity is grounded.
5. Record the reuse-vs-wrap split against the shipped Layer 1 surface (item 2 of the checklist).
6. Restate the loop's six must-nevers and the harness's one as citable obligations, each paired with the guard that enforces it (item 3).
7. Fix the layer's name with both exclusions and the "a Layer 3 application" term (item 4).
8. Build the exclusion register: cite Layer 1's `V-OUT-*` rows for concerns it already assigned; add the Layer-2-specific exclusions with their owning layer.
9. Quote all four wording traps verbatim; record the two reconciled conflicts (cost payload; the two "too broad" readings) and the two resolved here (§ decisions 4–5).
10. Verify against AG-00.1's closing checklist plus the added fifth boundary case, item by item, in the tasks phase.

---

## Affected areas

| Area | Impact | Description |
| --- | --- | --- |
| `openspec/changes/cachicamas-agent-contract-vocabulary/` | New | Six markdown files |
| `openspec/specs/agent-contract-vocabulary/` | New (at archive) | The register, promoted from this change's spec delta; live and appendable thereafter |
| `openspec/specs/ai-contract-vocabulary/spec.md` | Unchanged | Cited by row id; never edited |
| `backend/` (any module) | **Untouched** | No Go, no `go.mod`, no `Makefile` |
| `docs/` | **Untouched** | Cited, not edited |
| Build, tests, containers | **Unaffected** | Nothing compiles differently |

---

## Risks

| # | Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- | --- |
| 1 | A term is missing and gets invented inside a later PR instead of appended by amendment | Medium | High — the exact failure this milestone targets | Amendment rule 1 is normative in the spec and checkable; every downstream SDD inherits it |
| 2 | The delegation resolution (decision 4) is ignored and AG-06 / AG-19 ship different words | Medium | High | The scope rule binds *shipping* names, not prose, so it is checkable at review: an event kind, scenario id or test name that says "child" fails it |
| 3 | Two milestones both believe they own a term | Low | High | Ownership is a single column with exactly one value; duplicate ownership becomes a checkable artifact property |
| 4 | A Go identifier leaks in and pre-empts a milestone's naming | Medium | Medium | Forbidden normatively; the register uses noun phrases with spaces, which makes a leak visually obvious. Note: `explore.md` in this change deliberately **de-identifies** the Layer 1 grounding it inherited, citing files rather than exported names |
| 5 | The register over-reaches into AG-01, AG-02 or AG-22 territory | Medium | Medium | The out-of-scope table names all three with the node that owns each; the noun-versus-decision line is restated in `design.md` |
| 6 | Review burden — 60–70 terms is a long artifact | Medium | Medium | Six categories with fixed row shape make it scannable; `size:exception` is pre-accepted for this change and the artifact is prose, not code |
| 7 | The register is written and never read | Low | Medium | AG-00's acceptance criterion makes every later charter expressible in these terms, which makes it load-bearing for AG-01's SDD immediately |
| 8 | A citation resolves to the wrong register | Low | High | Decision 2's `VL2-` prefix makes every citation self-describing; no search string matches both registers |

---

## Rollback plan

Six markdown files added; nothing modified. Stated at three levels because the third is the interesting case.

**Level 1 — revert before anything depends on it.** `git revert` the merge commit, or delete `openspec/changes/cachicamas-agent-contract-vocabulary/`. No build, test, container, or migration is affected, because none was touched. One commit. Correct while AG-01 has not started.

**Level 2 — amend rather than revert, once downstream milestones cite terms.** Once AG-01, AG-02 or any AG-03+ charter cites a `VL2-*` id, a wholesale revert strands those citations. The supported correction is the living-graph convention applied to the register: append a dated blockquote under the touched category, strike through the superseded definition, land the amendment **in the same PR that needs it**. Ids are never reused and never renumbered; a missing term is appended with the next free ordinal in its category.

**Level 3 — supersede the artifact wholesale.** If the categorization itself proves wrong — for example if AG-01's delivery decision makes "observer" and "consumer" one concept rather than two — the artifact is superseded by a new decision node appended under AG-00 (next free ordinal, `AG-00.2`), with the edge recorded in doc 0003 per the living-graph clause. The original `decision.md` stays in place with superseded sections struck through, so links from merged charters keep resolving. This is the only path that requires editing doc 0003, and doc 0003's own clause already prescribes it.

**Note on promotion.** Before archive, the register exists only inside the change folder and Level 1 is a folder delete. After archive it is live at `openspec/specs/agent-contract-vocabulary/spec.md`; rollback from that point is Level 2 or Level 3, never a deletion of the live file.

**Blast radius, worst case:** every downstream artifact affected is itself a markdown document. No runtime, no data, no build. Nothing to migrate and nothing to restore.

---

## Dependencies

- **The doc 0002 wave 2 contract surface** — frozen. AI-40 merged as `7326a813` on 2026-08-10; Layer 1 closed 42 of 42. This is the gate doc 0003 records as satisfied.
- **doc 0003** (v2, restructured 2026-08-11) — the milestone charters this vocabulary assigns ownership from.
- **doc 0001** (v2, as amended 2026-08-10) — the architectural source of the concepts and of the `G1` … `G11` identifiers.
- **ADR 0004 as amended by ADR 0005** — the layer boundary the exclusion register enforces. **ADR 0006**, **ADR 0007** — cited, not changed.
- **The live `ai-contract-vocabulary` register** — cited by row id for every reused Layer 1 identity.
- No new tooling, no new dependency, no ADR required by this change.

---

## Success criteria

- [ ] All four of AG-00.1's closing-checklist items, plus the fifth boundary case added by decision 6, are answered in `decision.md`, verifiable item by item.
- [ ] Every term in the register has exactly one definition and exactly one owning milestone.
- [ ] Every term AG-00's charter names as a minimum deliverable is present: the runtime and its two parts; run, turn, provider call; transcript and the pairing invariant; the loop/harness responsibility split restated testably; suspension and resumption; steering; delegation and the parent relationship; the cost event's token-only scope.
- [ ] The reuse-vs-wrap split is stated for message identity, tool-call identity, finish reasons and usage (reused) and for events, ordering and failure (wrapped).
- [ ] The loop's six must-nevers and the harness's one appear as citable obligations, each naming the guard that enforces it.
- [ ] The layer's name is fixed with both exclusions, and "a Layer 3 application" is fixed as the consumer's term.
- [ ] Both open conflicts are resolved with both sides cited by document and section; both reconciled conflicts are recorded with their disposition.
- [ ] All four wording traps appear verbatim.
- [ ] No Go identifier appears in any artifact of this change.
- [ ] AG-01's and AG-02's charters, re-read after this change, are expressible using only these terms — the concrete instance of AG-00's acceptance criterion.
- [ ] `git status` after the change shows six added files and nothing else.

---

## Open questions carried to review

These do not block the spec and design phases; each has a stated default that review may overturn.

| # | Question | Default taken here |
| --- | --- | --- |
| 1 | Is `VL2-` the right prefix, or is a shorter one preferred? | `VL2-` (decision 2) — layer-marked, search-disjoint, generalizes to `VL3-` |
| 2 | Should `attempt` be its own row or a second sense of `provider call`? | Its own row (decision 5) — AG-16's cost scenario counts attempts, not calls |
| 3 | Should the fifth boundary case amend AG-00.1's checklist in doc 0003, or only be answered in the register? | Answered in the register; a doc 0003 amendment is proposed separately if review wants the checklist to carry it |
| 4 | Is `subagent` the right canonical term, given AG-19's title says *delegation*? | Yes for the participant, with `delegation` canonical for the relationship (decision 4) |

---

## Notes for the following phases

- **`sdd-spec`**: requirements are properties of the **artifact**, not of runtime behavior — there is no runtime. Scenarios are Given/When/Then over the document: "given the register, when a reviewer selects any row, then it names exactly one owning milestone." Requirement ids `R-AGV-0NN`, scenario ids `S-AGV-0NN`, mirroring Layer 1's `AIV` convention. The register text is authored here and promoted at archive.
- **`sdd-design`**: the design question is why a register with one owner per term prevents an argument class, not how to implement anything. Include the register's structural contract, the `VL2-` identifier scheme, and the amendment protocol.
- **`sdd-tasks`**: exactly one phase — AG-00.1 — whose task list is its closing checklist plus the added fifth boundary case. Single PR; `size:exception` is pre-accepted for this change. The artifact is prose and is reviewed as prose.
