# Spec — Layer 2 contract vocabulary

> **Change**: `cachicamas-agent-contract-vocabulary`
> **Milestone**: AG-00 · **Node**: AG-00.1 `[decision]`
> **Phase**: spec · **Capability**: `agent-contract-vocabulary` (new)
> **Canonical spec**: `openspec/specs/agent-contract-vocabulary/spec.md` — created by `sdd-archive` from this file, live and appendable thereafter
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-08-11
> **Requirement IDs**: `R-AGV-0NN` · **Scenario IDs**: `S-AGV-0NN`

---

## Purpose

AG-00.1 is a `[decision]` leaf. It ships no Go, so there is no runtime behavior to specify. The subject of this spec is an **artifact**: the Layer 2 contract vocabulary register. Every requirement below constrains that document, and every scenario is a property a reviewer can check against it by inspection, deterministically, without running anything.

Stated once so no later phase mistakes it: a scenario here reads "given the register, when …, then …" — **the register is the system under test.**

The register text itself is not authored in this phase. `sdd-apply` writes it, appended below these requirements in this same file, and `sdd-archive` promotes the whole to the canonical path.

---

## Definitions used by this spec

- **The register** — the categorized table of Layer 2 terms, comprising all six categories.
- **A row** — one term: identifier, term name, definition, owning milestone, provenance.
- **Owning milestone** — the current doc 0003 identifier `AG-NN` whose charter *defines* the term, as opposed to any charter that merely uses it.
- **Excluded term** — a term in the exclusion category which Layer 2 deliberately does not own; its owner is a layer or a composition root, never a Layer 2 milestone.
- **Wording trap** — one of the four sentences quoted from doc 0003's *Scope boundary* section.
- **The historical record** — `decision.md` in this change folder: AG-00.1's argument and the state of the register on the day it merged.

---

## R-AGV-001 — The register is live, canonical, and singular

The register MUST be authored in this change's spec delta and promoted by archive to `openspec/specs/agent-contract-vocabulary/spec.md`, where it stays **live and appendable** by later milestones. It MUST NOT be frozen inside the archived change folder. `decision.md` MUST remain in the archived change folder as the immutable historical record and MUST NOT be the citation target for a term. No other file in the repository MAY carry a competing normative definition of a Layer 2 term.

### Scenarios

- **S-AGV-001** — Given the merged change, when a reviewer resolves any `VL2-*` citation, then it resolves to `openspec/specs/agent-contract-vocabulary/spec.md` AND that file is writable by a later pull request AND no `VL2-*` citation in any charter or SDD points into `openspec/changes/archive/`.
- **S-AGV-002** — Given `decision.md` and the promoted register, when a reviewer compares them, then `decision.md` presents the argument and the merge-day snapshot AND the register presents the normative rows AND neither restates the other as a second normative source.

---

## R-AGV-002 — Fixed row shape

Every row MUST carry all five fields — **identifier · term · definition · owning milestone · provenance** — in that fixed shape, so a reviewer can scan one column at a time. A row MUST NOT omit a field or fold two fields into one cell. Provenance MUST name a document and a section (doc 0001 §, doc 0003 requirement `R-01`…`R-21` or forward-requirement `G1`…`G11`, an ADR, or a shipped Layer 1 register row), never a line number.

### Scenarios

- **S-AGV-003** — Given the register, when a reviewer reads every row across all six categories, then each row has a non-empty value in all five fields.
- **S-AGV-004** — Given any provenance cell, when a reviewer follows it, then it names a document and a section that exists AND no provenance cell cites a line number or a Layer 2 source file.

---

## R-AGV-003 — Exactly one owning milestone per term

Every non-excluded row MUST name **exactly one** owning milestone, expressed as a current `AG-NN` identifier from doc 0003. A row MUST NOT name two owners, MUST NOT name a range (`AG-04…AG-06`), MUST NOT say "all milestones", and MUST NOT leave the owner unstated. Two rows MUST NOT claim ownership of the same term.

### Scenarios

- **S-AGV-005** — Given the register, when a reviewer reads the owner column of every non-excluded row, then each cell holds exactly one `AG-NN` identifier AND that identifier exists as a milestone heading in doc 0003.
- **S-AGV-006** — Given a term used pervasively — for example *the runtime*, which every milestone references — when a reviewer reads its owner cell, then it names the single milestone whose charter *defines* it (AG-00), not the set of milestones that use it.
- **S-AGV-007** — Given any milestone cited as an owner, when a reviewer opens that milestone's charter in doc 0003, then the charter's goal or deliverable covers the owned term — an owner whose charter does not cover the term is a defect in this register, not in doc 0003.

---

## R-AGV-004 — Identifier discipline: `VL2-<CAT>-nn`, append-only, register-disjoint

Every term MUST carry a stable identifier of the form `VL2-<CAT>-nn`, where `<CAT>` is one of the six category codes and `nn` is an ordinal within that category. Identifiers MUST be append-only: a later term takes the next free ordinal in its category, and existing identifiers MUST NOT be renumbered, reused, or reordered. A superseded term MUST retain its identifier with its old definition struck through and visible. The scheme MUST be textually disjoint from Layer 1's `V-<CAT>-nn`, so that no plain-text search or citation can cross registers.

### Scenarios

- **S-AGV-008** — Given the register, when a reviewer reads any category's identifiers in document order, then ordinals are unique within the category AND no ordinal is shared between a superseded row and a live row.
- **S-AGV-009** — Given a plain-text search for `V-` across `openspec/specs/`, when the results are partitioned by register, then every Layer 1 hit belongs to `ai-contract-vocabulary` AND a search for `VL2-` returns only Layer 2 rows AND no identifier is ambiguous between the two.
- **S-AGV-010** — Given a term discovered missing after this change merges, when it is added, then it receives the next free ordinal in its category AND no existing identifier changes value or position.

---

## R-AGV-005 — Six categories, each term in exactly one

The register MUST be partitioned into exactly six categories — core identity (`COR`), event envelope (`EVT`), loop mechanics (`LOOP`), harness mechanics (`HAR`), cross-cutting seams (`SEAM`), and exclusions (`OUT`). Every term MUST belong to exactly one category. Each category heading MUST carry a term count, and the register MUST state the sum.

### Scenarios

- **S-AGV-011** — Given the register, when a reviewer lists its category headings, then exactly six are present with the codes above AND each carries a count AND the counts sum to the stated total AND the total equals the number of rows.
- **S-AGV-012** — Given every term name in the register, when a reviewer groups them by category, then no term name appears under two category headings AND no row's category code disagrees with the heading it sits under.

---

## R-AGV-006 — One observable definition per term, including the boundary cases

Every term MUST carry exactly one definition, phrased **observably** — stated so a later test, guard, or acceptance criterion could cite it — and stating what the term **is** and what it deliberately **is not**. A term MUST NOT be defined twice, in two categories, or with two wordings; a recurrence in a later milestone MUST be a cross-reference from the single owning row.

The register MUST answer, as explicit citable rows rather than as implications a reader reconstructs, these boundary cases:

| # | Boundary case | Required answer |
| --- | --- | --- |
| 1 | Is a turn with zero tool calls still a turn? | Yes — the terminal one |
| 2 | Is a compaction summary a transcript entry or metadata about entries? | A transcript entry, typed as a compaction artifact and distinguishable from a model message |
| 3 | Does a steering message belong to the current turn or the next? | The next — the current turn completes untouched and the message enters history at the turn boundary |
| 4 | Is a compaction call itself a turn? | No — it is a provider call but not a turn |

### Scenarios

- **S-AGV-013** — Given the register, when a reviewer collects every term name across all six categories, then no term name appears in more than one row AND no two rows carry the same definition text.
- **S-AGV-014** — Given boundary case 1, when a reviewer reads the *turn* row, then it states that a response with zero tool calls is a complete turn AND identifies it as the terminal turn — an answer left implicit in a scenario elsewhere does not satisfy this.
- **S-AGV-015** — Given boundary case 2, when a reviewer reads the compaction-artifact row, then it places the summary **in** the transcript as a typed entry AND states it is distinguishable from a model message AND does not describe it as metadata beside history.
- **S-AGV-016** — Given boundary case 3, when a reviewer reads the *steering* row, then it assigns the message to the next turn AND states the edge case that a message queued during the final turn yields a new turn rather than being dropped.
- **S-AGV-017** — Given boundary case 4, when a reviewer reads the *compaction call* row, then it states the call is a provider call and not a turn AND names the definitional reason (it produces a summary, not an assistant response with its tool results).
- **S-AGV-018** — Given the three rows *turn*, *provider call*, and *attempt*, when a reviewer reads them, then each has its own row with its own scope label AND the register carries the reconciling statement that a turn spans one or more provider calls, exceeding one only via harness retry, while a single loop invocation never issues a second provider call — with both conflicting sides cited by document and section.

---

## R-AGV-007 — The reuse-versus-wrap split is stated (checklist item 2)

The register MUST state which Layer 1 identities Layer 2 **reuses as-is** — at minimum message identity, tool-call identity, finish reasons, and usage — and which it **wraps** — at minimum events, ordering, and failure. A reused identity MUST be recorded by citation to its Layer 1 `V-*` row and MUST NOT be re-paraphrased here. A wrapped identity MUST state what Layer 2 adds that the Layer 1 identity has no place for.

### Scenarios

- **S-AGV-019** — Given the reused identities, when a reviewer reads each entry, then it cites a `V-*` row identifier from the live `ai-contract-vocabulary` register AND carries no independent definition sentence of its own.
- **S-AGV-020** — Given the wrapped identities, when a reviewer reads the events entry, then it states that the Layer 1 event is stream-scoped while the Layer 2 envelope is run/turn-scoped, names the parent-nesting and event-family additions that have no Layer 1 analog, and does not present the wrap as a reuse.
- **S-AGV-021** — Given the ordering entry, when a reviewer reads it, then it states that Layer 2 ordering is an independent agent-level counter and explicitly not the Layer 1 per-stream sequence — an entry that merely cites the Layer 1 sequence row fails this scenario.

---

## R-AGV-008 — The must-nevers become citable obligations (checklist item 3)

The register MUST restate the loop's **six** must-nevers and the harness's **one** must-never as vocabulary-level obligations. Each MUST be a citable row or named obligation, phrased so an AG-03 guard or a later Gherkin scenario can reference it by name, and each MUST name the mechanical enforcement that bites on it. A prose paragraph that restates the prohibitions without giving each a citable handle does not satisfy this requirement.

### Scenarios

- **S-AGV-022** — Given the register, when a reviewer counts the loop obligations, then exactly six are present — no persistence, no filesystem or environment access, no rendering, no tool-permission decision, no retry decision, no frontend knowledge — AND each names its enforcing guard or scenario.
- **S-AGV-023** — Given the harness obligation, when a reviewer reads it, then it states that the harness may vary when and how often the loop runs but may never reach inside a single loop invocation to change how it decides turn completion AND names its mechanical enforcement.
- **S-AGV-024** — Given AG-03's guard work as later written, when its author needs to cite the obligation a guard enforces, then a `VL2-*` identifier or named obligation exists for it AND the guard's description needs no new prohibition wording of its own.

---

## R-AGV-009 — The layer's name is fixed with both exclusions and the consumer term (checklist item 4)

The register MUST fix **the portable agent runtime** as the name of the loop-plus-harness assembly, stating that the runtime *is* the loop plus the harness — not a third thing wrapping them, and not a synonym for either alone. It MUST state two exclusions:

- **(a)** No cognitive or biological metaphor is used for any Layer 2 concept. The retired *brain* framing MUST be named explicitly, together with the reason it was retired, so the reason survives the rename and not merely the result.
- **(b)** "The runtime" never abbreviates Go's `runtime` package; where a sentence needs the distinction, it writes "the agent runtime".

It MUST also fix **"a Layer 3 application"** as the term for the runtime's consumer, deliberately not "the coding agent" and not any product name.

### Scenarios

- **S-AGV-025** — Given the name row, when a reviewer reads it, then the runtime is defined as loop-plus-harness AND both negative clauses (not a third wrapping thing, not a synonym for either part) are present.
- **S-AGV-026** — Given exclusion (a), when a reviewer reads it, then the retired *brain* framing is named AND the recorded reason is that a cognition metaphor invites placing policy inside Layer 2 — an exclusion that states the rule without naming the retired framing fails this scenario.
- **S-AGV-027** — Given exclusion (b), when a reviewer reads it, then the disambiguation rule is stated operationally as the phrase to write when the distinction is needed.
- **S-AGV-028** — Given the consumer term, when a reviewer scans every row, definition, and obligation in the register, then the consumer is named "a Layer 3 application" throughout AND no row, test name, or acceptance criterion is phrased in terms of a coding agent specifically.

---

## R-AGV-010 — The delegation vocabulary is canonical, with a scope rule

The register MUST declare **subagent** canonical for the delegated participant and **delegation** canonical for the relationship and the event family. It MUST record **child harness** and **nested run** as admissible synonyms, each with the sense it carries. It MUST state the scope rule: any name that ships — an event kind, a scenario identifier, a test name, an acceptance criterion — uses *subagent* or *delegation*; prose MAY use a synonym only where the register lists it. Both sides of the conflict MUST be cited by document and section.

### Scenarios

- **S-AGV-029** — Given the delegation rows, when a reviewer reads them, then exactly one participant term is marked canonical AND the two synonyms are listed as synonyms rather than as alternative definitions.
- **S-AGV-030** — Given a later pull request that introduces an event kind, scenario identifier, or test name containing "child" or "nested", when a reviewer applies the scope rule, then the name fails review and the register row is the citable basis for the rejection.

---

## R-AGV-011 — Excluded terms are named, attributed, and never defined

The exclusion category MUST list every concept Layer 2 deliberately does not own — at minimum permission policy content, sandbox semantics, tool source, summary quality, cross-session rule persistence, price and money, session persistence, frontends, and catalogs — each attributed to the layer or composition root that owns it. An excluded term MUST NOT carry a Layer 2 definition. Where Layer 1's register already assigned a concern, the row MUST cite the Layer 1 `V-OUT-*` row rather than re-attributing it.

### Scenarios

- **S-AGV-031** — Given the exclusion category, when a reviewer reads every row, then each names an owner that is a layer, a port, or the composition root — never an `AG-NN` milestone.
- **S-AGV-032** — Given any excluded term, when a reviewer looks for a definition of it, then none is present — the row states ownership, and where the exclusion is easy to misread, the Layer 2 concept it is commonly confused with.
- **S-AGV-033** — Given the cost exclusion, when a reviewer reads it, then the Layer 2 cost payload is recorded as token-only with money as Layer 3 enrichment AND both sides of the reconciled conflict are cited by document and section.

---

## R-AGV-012 — The four wording traps are quoted verbatim

The register MUST reproduce all four of doc 0003's *Scope boundary* wording traps **verbatim**, not paraphrased, each accompanied by its corrected phrasing and the record that it is a plausible-but-wrong reading.

### Scenarios

- **S-AGV-034** — Given the register, when a reviewer compares each trap quotation against doc 0003's *Scope boundary* section, then all four match character for character, including qualifying clauses.
- **S-AGV-035** — Given the trap "the loop executes tools", when a reviewer reads the accompanying text, then the corrected phrasing states that the loop schedules execution against an injected execution contract AND the register's tool-related rows are consistent with that correction.
- **S-AGV-036** — Given the trap "the harness holds state", when a reviewer reads the accompanying text, then the corrected phrasing states that the harness holds the conversation in memory AND that a harness which touches a file has crossed the boundary.

---

## R-AGV-013 — Growth is by amendment, never by invention

After this change merges, a Layer 2 term found missing, wrong, or double-owned MUST be corrected by an amendment to the register, landed **in the same pull request that needs it**. A downstream milestone MUST NOT introduce a new Layer 2 term in its own SDD without appending it here. An amendment MUST take the next free ordinal in its category, MUST be introduced by a **dated blockquote under the touched category heading** stating what was appended, by which node, and **why the register lacked the term**, MUST update the per-category count and the sum, and MUST NOT silently edit existing text: a superseded definition keeps its identifier with its old text struck through and visible.

### Scenarios

- **S-AGV-037** — Given a downstream milestone whose SDD needs a term the register does not carry, when that SDD is written, then the same pull request appends the term with the next free ordinal AND a dated amendment blockquote records the addition, the appending node, and the reason the register lacked it.
- **S-AGV-038** — Given an amendment blockquote that records only what was appended and by whom, when a reviewer applies this requirement, then it fails — the reason the register lacked the term is a required element, and its absence is what lets the same gap recur.
- **S-AGV-039** — Given a superseded definition, when a reviewer reads its row, then the old text is struck through and remains visible AND the replacing text is present AND the identifier is unchanged.
- **S-AGV-040** — Given any amendment, when a reviewer reads the touched category heading, then its count and the register's stated sum both reflect the appended row.

---

## R-AGV-014 — Downstream charters are expressible in these terms

The register MUST be complete enough that every milestone charter from AG-01 through AG-23 can be written using only its terms plus ordinary English, so that a later charter cites a register entry instead of defining a term inline. Conflicting uses of a noun across doc 0001, doc 0002, and doc 0003 MUST be either reconciled with the disposition recorded, or explicitly flagged with the node that will resolve them. This is AG-00's acceptance criterion, restated normatively so it is checkable rather than aspirational.

### Scenarios

- **S-AGV-041** — Given AG-01's charter as written in doc 0003, when a reviewer maps each domain noun in it to the register, then every one resolves — specifically observer, upward path, steering, carrier, and interrupt.
- **S-AGV-042** — Given AG-02's charter as written in doc 0003, when a reviewer maps each domain noun in it to the register, then every one resolves — specifically failover seam, subagent, and the forward-requirement identifiers it disposes.
- **S-AGV-043** — Given any charter from AG-03 through AG-23, when a reviewer finds a domain noun that does not resolve, then that finding is recorded as a defect in this register and closed by amendment under `R-AGV-013` — not by inventing the term in that milestone's SDD.
- **S-AGV-044** — Given every noun the register reconciles across documents, when a reviewer reads its row, then the disposition is recorded with both sides cited, or the row explicitly flags the conflict as open and names the node that owns the resolution.

---

## R-AGV-015 — No Go identifiers, no production code, no module change

No artifact of this change MAY contain a Go type name, field name, method name, interface name, or package identifier belonging to the future Layer 2 surface. Terms MUST be expressed as conceptual noun phrases written as ordinary English — never camel-case, never Pascal-case, never underscore-joined, and never styled as a code identifier in any other way. The test is the **surface form a reader sees**, not whether a compiler would accept the characters: an ordinary lowercase English word such as *run*, *turn* or *transcript* satisfies this requirement, because the rule's purpose is to make a leaked Go identifier visually anomalous (`design.md` § 3), and an English word in running prose is not. Naming the *spelling* of a term is each owning milestone's own SDD decision. The change MUST NOT create, modify, or delete any file under `backend/`, any `go.mod`, `go.sum`, `Makefile`, or any build or container configuration.

> **Amendment (2026-08-11, during `sdd-apply` remediation, before this change merged).** The original sentence required noun phrases "containing spaces." That over-hardened Layer 1's own descriptive row-shape note — `openspec/specs/ai-contract-vocabulary/spec.md` line 56's table cell, "A conceptual noun phrase, always with spaces. Never a Go identifier" — into a MUST here; but Layer 1's own normative sentence, the same file's line 456, requires only "conceptual noun phrases," never "always with spaces." Three of this register's own `VL2-COR` terms — *run*, *turn*, *transcript* — are genuinely single words, named as single words in AG-00's own charter Goal, and padding them into artificial phrases would distort the vocabulary to satisfy a typo rather than the rule's actual purpose: making a leaked Go identifier visually anomalous (`design.md` § 3). The requirement is corrected to that purpose. No register row is renamed by this correction.
>
> **Second amendment (2026-08-11, same remediation, after re-verification).** The first amendment replaced "containing spaces" with "never a single token that could itself compile as a type, field, method, or package name" — which re-encoded the same prohibition, because every ordinary lowercase English word *is* a legal Go identifier. Re-verification measured the effect: twelve of the thirteen single-token rows still failed, and only *re-entrancy* was cleared, by its hyphen. The requirement now tests the **surface form** instead, which is what `design.md` § 3 actually argues for. The claim that the thirteen rows "were always conceptual noun phrases and never Go identifiers" is withdrawn: as a syntactic claim it is false, and it is replaced by the surface-form scope above. `S-AGV-046` is amended in the same pass so the scenario and its requirement agree — the first amendment left them disagreeing, which is how the defect survived.

### Scenarios

- **S-AGV-045** — Given any artifact of this change, when a reviewer scans for camel-case or Pascal-case single-token names, struct or interface declarations, or field lists, then none is present.
- **S-AGV-046** — Given any term row, when a reviewer reads the term name, then its surface form is ordinary English rather than a code identifier — not camel-case, not Pascal-case, not underscore-joined, not otherwise styled as code — AND the definition describes what the concept must be able to express and what nothing in Layer 2 may do with it, stating no name, shape, or signature.
- **S-AGV-047** — Given the change's diff, when a reviewer lists changed paths, then every path is under `openspec/changes/cachicamas-agent-contract-vocabulary/` AND every changed file has a `.md` suffix.
- **S-AGV-048** — Given the repository before and after the change, when the build and test targets are run in every module, then their outcomes are identical — the change is provably inert with respect to the build.

---

## Non-functional requirements

### NFR-AGV-A — Reviewability

- The register MUST be readable in one sitting by someone who has read doc 0003's *Scope boundary* section and nothing else.
- Every row MUST fit the pattern `identifier · term · definition · owner · provenance` so a reviewer can scan one column at a time.
- The register MUST NOT require the reader to open doc 0001 to understand a definition; citations are for provenance, not for comprehension.

### NFR-AGV-B — Traceability

- Every doc 0003 requirement `R-01`…`R-21` and every forward requirement `G1`…`G11` carrying a Layer 2 obligation MUST appear at least once in the provenance column, so the vocabulary is auditable against doc 0003's traceability spine.
- Every citation MUST name a document and a section, never a line number.

### NFR-AGV-C — Durability

- No citation MAY point at Layer 2 code, which does not exist. Citations resolve to a contract document, an ADR, the architecture reference, or the **shipped Layer 1 surface**.

---

## Acceptance criteria

The change is accepted when:

1. `R-AGV-001` through `R-AGV-015` hold, each verified by its scenarios.
2. All four items of AG-00.1's closing checklist, plus the fourth boundary case added by the proposal, are answered in the register and in `decision.md`, and `tasks.md` records the verification item by item.
3. Both open conflicts (delegation; turn versus provider call versus attempt) are resolved with both sides cited, and both reconciled conflicts are recorded with their disposition.
4. All four wording traps appear verbatim.
5. No Go identifier appears anywhere in the change, and the diff contains markdown under this change directory only.
6. AG-01's and AG-02's charters are demonstrably expressible in the register's terms (`S-AGV-041`, `S-AGV-042`) — the handoff those two milestones' SDDs consume.

---

## The register

> Authored by `sdd-apply` under `R-AGV-001` … `R-AGV-015` and appended below this heading. Promoted with this file to `openspec/specs/agent-contract-vocabulary/spec.md` at archive, where it remains live and appendable under `R-AGV-013`.

Hold the single, definitive vocabulary of Layer 2 of the cachicamas agent stack: every noun a milestone charter, an SDD, or a pull-request description from AG-01 through AG-23 may use about Layer 2, with exactly one definition, exactly one owning milestone, and its provenance.

The register is **86 terms in six categories**. It settles words, not behavior: it names the carrier without choosing where it lives beyond AG-01's own decision, names the failover seam without implementing it, names the compaction call without deciding what a good summary says. Each of those is a later milestone's decision.

### Status — this file is the live register, and it is appendable

`openspec/AGENTS.md` calls `openspec/specs/` the *source of truth (main specs)*. The register therefore lives **here**, in the canonical tree once promoted, and not in the archive — mirroring the Layer 1 register exactly (`openspec/specs/ai-contract-vocabulary/spec.md`).

1. **This file is the register.** It carries the definitive text of every Layer 2 term. A milestone that needs a Layer 2 noun resolves it here, cites the row identifier, and — when the noun is missing — appends it here.
2. **`decision.md` is the historical record of how the register was first decided**, not a second normative copy. It holds AG-00.1's argument, its closing-checklist verification, and the state of the register on the day AG-00.1 merged. It is immutable; nothing is ever added to it, and it is never the citation target for a term (`R-AGV-001`).
3. **Archiving the register would freeze an artifact AG-01 … AG-23 must still write to.** Two rows are pre-added below on AG-01's behalf (`VL2-EVT-18`, `VL2-EVT-19`) precisely so its own SDD satisfies its amendment duty by citation rather than by a second append.

### How to amend this register

The six standing rules, in operational form:

| # | Rule |
| --- | --- |
| 1 | **Append, never invent.** A Layer 2 noun this register lacks is added *here*, never defined locally in a downstream SDD. |
| 2 | **Next free ordinal in its category.** `VL2-<CAT>-nn`, taking the next unused `nn` for that `<CAT>`. Identifiers are append-only: never renumbered, never reused, never reordered. |
| 3 | **A dated amendment blockquote under the touched category heading.** It states what was appended, by which node, and *why the register lacked the term*. |
| 4 | **No silent edit.** A superseded definition keeps its identifier with its old text struck through and visible, so citations from merged charters keep resolving. |
| 5 | **Update every count.** A term's addition updates the intro sentence's stated total, its own category heading's count, and the `## Term count` section's per-category figures and sum — every location a count appears, not only the total below. |
| 6 | **In the same pull request that needs it.** The amendment lands with the milestone that discovered the gap, not afterwards. |

### Row shape and categories

Every term carries five fields — **identifier · term · definition · owning milestone · provenance** — in that fixed order. Provenance names a document and a section, never a line number; where a term grounds a claim in shipped Layer 1 code, provenance names the file path and the definition describes the value conceptually, never by exported name.

| Code | Category | Test order (design.md § 2.2) |
| --- | --- | --- |
| `VL2-COR` | Core identity | 3 |
| `VL2-EVT` | The event envelope | 2 |
| `VL2-LOOP` | Loop mechanics | 4 |
| `VL2-HAR` | Harness mechanics | 5 |
| `VL2-SEAM` | Cross-cutting seams | 6 |
| `VL2-OUT` | Excluded — named, attributed, never defined | 1 |

### Wording traps

Quoted verbatim from doc 0003 § Scope boundary — not paraphrased, because paraphrase is how Layer 1's own two traps caused a wrong decision, and Layer 2's four are recorded here before any Layer 2 code exists to get it wrong.

**Trap 1 — the tool trap**

> **"The loop executes tools" is too broad.** The loop *schedules* execution against an injected execution contract and drives the permission protocol around it. What a tool does, whether it is allowed, and under what confinement it runs are all decided above; the loop owns ordering, concurrency, suspension, and the rejoin.

Corrected phrasing: the loop schedules execution against `VL2-LOOP-03` the tool execution contract; it never executes, never decides permission, never decides confinement. `VL2-LOOP-03` … `VL2-LOOP-06` are consistent with this correction throughout.

**Trap 2 — the state trap**

> **"The harness holds state" does not mean the harness persists state.** The harness holds the conversation *in memory* and exposes it for a Layer 3 session to persist. A harness that touches a file has crossed the boundary; the no-I/O guard (AG-03.3) exists to make that mechanical.

Corrected phrasing: `VL2-COR-03` the harness holds the transcript in memory only; persistence is `VL2-OUT-07` session persistence, a Layer 3 concern. `VL2-COR-18` no ambient authority is the mechanical guard this trap depends on.

**Trap 3 — the runtime-package trap**

> **"The runtime" is this layer, never Go's `runtime` package.** The term names the loop-plus-harness assembly — the thing that *runs* an agent conversation. Nothing in `backend/agent` imports Go's `runtime`, and AG-03.2's allowlist would have to be widened deliberately for that to change, so the collision is a reading cost only. When a sentence needs the distinction, write "the agent runtime". No cognitive or biological metaphor names any Layer 2 concept — the retired *brain* framing is recorded in AG-00.1 so the reason survives the rename.

Corrected phrasing: every use of "the runtime" in this register and in later Layer 2 artifacts means `VL2-COR-01`; where a sentence could be misread as Go's standard-library package, it is written "the agent runtime". Trap 3's own verbatim-quoted sentence above is this register's one unavoidable mention of Go's `runtime` package *inside a quoted block* — required by `R-AGV-012`'s verbatim-quote rule, not chosen. That is the only counted exception: Go's `runtime` package was never part of "the future Layer 2 surface" `R-AGV-015` forbids, so the corrected phrasing above and every other disambiguating use of the word elsewhere in this register and this change's other artifacts remain independently legitimate under `R-AGV-009(b)`, as many times as clarity requires — none of them is a second exception to count. See the note under `VL2-COR-01`.

**Trap 4 — the Layer 3 trap**

> **"Layer 3" in this document means the layer, not the coding agent.** Every out-of-scope line below that hands something to Layer 3 — policy content, pricing, persistence, prompt material, frontends — hands it to *whichever application is standing on the runtime*, which today is `cachicamas_coding` and tomorrow may not be. A test, a contract, or a milestone that only makes sense for a coding agent has put an application's assumption inside the runtime; that is a boundary violation with the same weight as an import violation, and AG-23's consumer proof is where it surfaces.

Corrected phrasing: every `VL2-OUT` row below names "Layer 3" or "a Layer 3 application" as owner, never "the coding agent" and never a product name. `AG-23`'s consumer proof (`VL2-SEAM-14` readiness contract) is the mechanical check that no row, test name, or acceptance criterion drifted into a coding-agent assumption.

### Reuse versus wrap

**Reused as-is** — Layer 2 never redefines these; every row below cites the Layer 1 `V-*` row and adds no independent definition of its own.

| Layer 2 use | Layer 1 identity reused | Grounding (shipped Layer 1 file, conceptual) |
| --- | --- | --- |
| Message identity, the value pairing and history logic (`VL2-HAR-01`) reduce over | `V-REQ-03` message identity | `backend/agent/src/ai/message.go` |
| Tool-call identity, carried through the tool execution contract (`VL2-LOOP-03`) | `V-REQ-16` tool call, `V-REQ-17` argument bytes | `backend/agent/src/ai/tool_call.go` |
| Finish reasons, consumed by finish-reason dispatch (`VL2-LOOP-08`) | `V-MET-01` … `V-MET-08` | `backend/agent/src/ai/finish_reason.go` |
| Usage, reported verbatim by cost aggregation (`VL2-HAR-09`) | `V-MET-09` … `V-MET-12` | `backend/agent/src/ai/usage.go` |
| The Layer 1 stream event, what the loop drains and re-emits from | `V-STR-10` event, `V-STR-11` event kind | `backend/agent/src/ai/event.go` |

**Wrapped, never reused unchanged** — each states the Layer 2 addition the Layer 1 identity has no field for.

| Layer 2 term | What it wraps | The addition |
| --- | --- | --- |
| The agent event envelope (`VL2-EVT-01` and its families, `VL2-EVT-02` … `VL2-EVT-09`) | `V-STR-10` event | Layer 1's event is stream-scoped (one provider response); Layer 2's envelope is run/turn-scoped, adds explicit parent nesting (`VL2-EVT-13`) with no Layer 1 analog, and carries four families (permission, cost, delegation, compaction) that do not exist at Layer 1 at all — the wrap is not a reuse |
| Ordering (the sequence the carrier, `VL2-EVT-17`, assigns) | `V-STR-13` sequence | Layer 1's sequence is per-stream. Layer 2 needs an independent, per-consumer-stream, agent-level ordering — a new instance of the same pattern, explicitly **not** the Layer 1 per-stream counter |
| Typed turn failure (`VL2-LOOP-09`) | `V-FAIL-05` … `V-FAIL-10` | Layer 1's taxonomy (category, retryability, partial-output discriminator) is carried in unchanged; the turn-failure type adds turn-scoped context (which turn, which call) Layer 1's taxonomy has no field for |

---

## 1. Core identity — `VL2-COR` (23 terms)

Owners AG-00, AG-01. The runtime and its two parts; the run's timeline units; delegation; the seven must-never obligations that constitute the loop/harness split.

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `VL2-COR-01` | **the portable agent runtime** | The loop-plus-harness assembly that executes an agent conversation: the stateless loop (`VL2-COR-02`) plus the stateful harness (`VL2-COR-03`) together, and nothing else — not a third thing wrapping them, not a synonym for either part alone. Every verb it performs is mechanism (schedule, stream, suspend, resume, append, count, cancel); it decides no policy. Two naming exclusions bind every Layer 2 artifact: **(a)** no cognitive or biological metaphor names any Layer 2 concept — the retired "the portable brain" framing is named here, not silently dropped, because a metaphor implying cognition invites exactly the mistake of putting policy inside Layer 2, which is why it was retired; **(b)** "the runtime" never abbreviates Go's `runtime` package — a sentence needing the distinction writes "the agent runtime". The runtime's consumer is fixed as **"a Layer 3 application"** throughout this register and every later Layer 2 artifact — deliberately not "the coding agent" and not any product name, so no contract, test name, or acceptance criterion is phrased for a coding agent specifically. | AG-00 | doc 0001 § 4 amendment (2026-08-10); doc 0003 "Outcome first", R-01, R-02, R-03, R-20 (AG-00 itself closes R-20's start condition, doc 0003's own SDD-change annotation for this milestone) |
| `VL2-COR-02` | **the loop** | The stateless half of the runtime. Given a system instruction, a transcript, a tool set and options, it runs one assistant turn and emits events; it holds no state between calls and knows nothing of sessions, users, frontends, or persistence. It must remain callable directly from a test with a scripted provider. Its must-never obligations are `VL2-COR-17` … `VL2-COR-22`. | AG-00 | doc 0001 § 4.1; doc 0003 R-06 |
| `VL2-COR-03` | **the harness** | The stateful half of the runtime. It holds the conversation across turns — history, suspension, the cancellation tree, delegation, the compaction trigger — and is what a Layer 3 application interacts with. Its one must-never obligation is `VL2-COR-23`. | AG-00 | doc 0001 § 4.2; doc 0003 R-07 |
| `VL2-COR-04` | **run** | The multi-turn unit the harness drives, bracketed by exactly one run-start and one run-end. A run ends with a typed outcome — completed, interrupted, or failed — never silently and never absent. | AG-00 | doc 0003 AG-00 charter ("one run = many turns"), AG-04.2, R-04, R-08 |
| `VL2-COR-05` | **turn** | One assistant response plus its tool results. A turn with zero tool calls is still a complete turn — the terminal one — not a degenerate or partial case. Harness-scoped: the unit the run driver advances one at a time. It may span more than one provider call (`VL2-COR-06`) only through harness retry — see `VL2-COR-08`. | AG-00 | doc 0003 AG-00 charter, AG-00.1 closing checklist item 1, R-08; proved by AG-07.1 |
| `VL2-COR-06` | **provider call** | One Layer 1 stream. Exactly one per loop invocation — a single loop invocation never issues a second provider call. Loop-scoped: the unit the loop, not the harness, produces. | AG-00 | doc 0001 § 2.3 (one loop-to-provider interaction per iteration); doc 0003 AG-11.2 ("the loop never issues a second provider call"), R-06, R-08 |
| `VL2-COR-07` | **attempt** | A provider call made in service of a turn already begun. Attempts beyond the first exist only because the harness re-invoked the loop under retry — harness-scoped, even though each individual attempt is one loop-scoped provider call. Distinct from a compaction call (`VL2-SEAM-08`), which is never made in service of a turn. | AG-00 | doc 0003 AG-15.1 ("each attempt is a fresh provider call"), AG-16.1 ("cumulative counts every attempt"), R-15, R-16 |
| `VL2-COR-08` | **turn provider-call span** | The reconciling statement between `VL2-COR-05` turn, `VL2-COR-06` provider call, and `VL2-COR-07` attempt: a turn spans one or more provider calls; the count exceeds one only via harness retry; within a single loop invocation the loop never issues a second provider call. Both conflicting readings this reconciles are cited: the loop-scoped reading (one invocation, one call) and the harness-scoped reading (a turn may retry) speak at different scopes and do not actually contradict. | AG-00 | doc 0001 § 2.3 (the loop-scoped side); doc 0003 AG-15 charter, AG-16.1 "including the retried attempt's tokens" (the harness-scoped side); doc 0003 AG-00 charter ("one turn may span several provider calls only via retry") |
| `VL2-COR-09` | **upward path** | The one decided surface through which a permission decision, a steering message, or an interrupt re-enters a live run — the only upward arrow in the layer view. The harness is the surface: the stable, addressable thing a Layer 3 application can hold across a whole run, resolving identity at two levels (run identity, then call identity for a suspension) before routing downward. A message that loses the race against a run or call ending returns a typed rejection at the matching granularity, never a silent drop. | AG-01 | doc 0003 R-09, AG-01.1 closing checklist item 5 (the upward-path decision) |
| `VL2-COR-10` | **transcript** | The harness's ordered history of messages across a run. Distinct from a Layer 1 message (`V-REQ-02`), which is Layer 1's unit of attribution within one request; a transcript is Layer 2's collection of messages across many turns, with its ordering and its repair after interruption or compaction. See `VL2-HAR-01` history for the store that holds it. | AG-00 | doc 0001 § 4.2; doc 0003 AG-00 charter, R-07; Layer 1 `V-OUT-02` |
| `VL2-COR-11` | **pairing invariant** | Every tool call has a matching result. Enforced at the history boundary (`VL2-HAR-01`), not patched at each call site — including after compaction, the point at which it is most often violated and where `VL2-SEAM-04`/`VL2-SEAM-06` protect it. | AG-00 | doc 0001 § 4.2; doc 0003 AG-00 charter, R-07 |
| `VL2-COR-12` | **steering (message)** | A user message that arrives while a turn is in flight. It belongs to the **next** turn, never the current one: the current turn completes untouched, and the message enters history at the turn boundary, before the next provider call. Edge case: a steering message queued during the final turn yields a new turn rather than being dropped. | AG-00 | doc 0003 AG-00 charter, AG-00.1 closing checklist item 1, AG-13.2, R-09 |
| `VL2-COR-13` | **suspension and resumption** | A scheduled tool call paused on the permission protocol (`VL2-LOOP-07`) is suspended; it resumes on a decision arriving through the upward path (`VL2-COR-09`). Suspension must not block sibling calls or event delivery — other calls and in-flight message deltas continue while one call is suspended. | AG-00 | doc 0003 AG-00 charter, AG-01.1 closing checklist item 5, AG-10.1, AG-10.3, R-09, G1 |
| `VL2-COR-14` | **subagent** | The delegated participant: a harness invoked from within a tool execution. Canonical for the participant — the term wins because it already appears on the wire in the delegation event-kind names (`VL2-EVT-08`). Two admissible synonyms are recorded, for prose only, each with its own sense: **child harness** (the re-entrancy mechanism a subagent stands on, `VL2-SEAM-09`) and **nested run** (the run a subagent drives). Scope rule: any name that ships — an event kind, a scenario id, a test name, an acceptance criterion — uses *subagent* or *delegation* (`VL2-COR-15`); prose may use a synonym only where this row lists it. | AG-00 | doc 0003 § AG-06 (node AG-06.3, event-kind names), § AG-02 charter ("no subagent tool ships in v1", R-14); doc 0001 § 6 seam 12 — against the losing reading: doc 0003 § AG-19 scenarios and heading ("child harness", "delegation readiness") |
| `VL2-COR-15` | **delegation** | The relationship between a parent harness and a subagent (`VL2-COR-14`) it invokes from within a tool, and the name of the event family reporting it (`VL2-EVT-08`). Canonical for the relationship and the event family; AG-19's title "delegation readiness" is correct as it stands under this term. | AG-00 | doc 0001 § 4.2, § 6 seam 12; doc 0003 § AG-19, R-14, G7 |
| `VL2-COR-16` | **cost scope** | The Layer 2 cost event's scope is token-only: per-turn and cumulative figures for input, output, cache-read, cache-write and reasoning tokens. It carries no field that could hold money — money is Layer 3 enrichment, added above Layer 2. Reconciles doc 0001 § 4.3's row (harness reports "tokens, cache hits and money") against ADR 0005 § D4 and doc 0001 § 7 G10 ("L2 emits, L3 prices"): the verdict wins. | AG-00 | doc 0001 § 4.3, § 7 G10; ADR 0005 § D4 row G10; doc 0003 AG-06 charter note, R-16 |
| `VL2-COR-17` | **loop statelessness** | Must-never 1 of 6. The loop must never persist anything: no field, no side channel survives across two sequential turns on one loop value. Proved, not merely disciplined: two sequential turns on the same loop value are independent, with fresh ordering and no residue. | AG-00 | doc 0001 § 4.1; doc 0003 R-06; enforced/proved by AG-07.2 |
| `VL2-COR-18` | **no ambient authority** | Must-never 2 of 6. The loop must never read the filesystem or the environment. Enforced mechanically: zero environment reads, zero filesystem calls, zero process spawns anywhere in the loop's closure. | AG-00 | doc 0001 § 4.1; doc 0003 R-02, R-06; enforced by AG-03.3 (the no-ambient-authority guard) |
| `VL2-COR-19` | **stream-only output** | Must-never 3 of 6. The loop must never render anything; its only output is the event stream. Enforced by import denial (nothing rendering-adjacent is reachable from the loop's closure) plus the stream-only contract every event family upholds. | AG-00 | doc 0001 § 4.1; doc 0003 R-06; enforced by AG-03.2 (forward import guard) and AG-04's stream-only contract |
| `VL2-COR-20` | **no permission decision** | Must-never 4 of 6. The loop must never decide *whether* a tool is allowed; it asks the injected policy and executes the answer it is given. An immediate policy answer needs no suspension event at all — silence on the stream means nobody needed to be asked. | AG-00 | doc 0001 § 4.1; doc 0003 R-06, G1; enforced/proved by AG-10.1 |
| `VL2-COR-21` | **no retry decision** | Must-never 5 of 6. The loop must never decide *whether* to retry; it reports a typed failure upward and stops. Proved as a count, not merely a rule: for any failing turn, the fake provider's call count is exactly one. | AG-00 | doc 0001 § 4.1; doc 0003 R-06, R-15; proved by AG-11.2 |
| `VL2-COR-22` | **no frontend knowledge** | Must-never 6 of 6. The loop must never know which frontend is attached. No frontend-identifying type or parameter crosses the loop's public surface. Enforced by import denial: nothing naming a frontend is reachable from the loop's closure. | AG-00 | doc 0001 § 4.1; doc 0003 R-06; enforced by AG-03.2 (forward import guard) |
| `VL2-COR-23` | **harness non-interference** | The harness's one must-never. It must never dictate the loop's logic. It may vary *when* and *how often* the loop runs — retry, steering, compaction insertion — but may never reach inside a single loop invocation to change how it decides turn completion. A different termination rule is a different harness, not a different loop. Proved structurally: the harness holds no privileged channel into the loop — it goes through the same public one-turn surface the loop's own external tests use. | AG-00 | doc 0001 § 4.2; doc 0003 R-07; proved by AG-13.1 |

---

## 2. The event envelope — `VL2-EVT` (19 terms)

Owners AG-01, AG-04, AG-05, AG-06. Kind and the eight families; the four invariants; run/turn outcomes; the stream-contract validator; the carrier and the two named loss postures decided at AG-01.

> Rows `VL2-EVT-18` and `VL2-EVT-19` are authored here on AG-01's behalf, at AG-01's own recorded request (AG-01.1's own design record, "the two loss postures must be nameable distinctly"), so that AG-01's own amendment duty is satisfied by citation rather than a second append.

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `VL2-EVT-01` | **event kind** | The discriminator naming what an agent event carries, derived from the event's payload rather than set alongside it, so a kind cannot disagree with its contents. The kind set is closed; every registered kind has a constructible payload, mechanically enforced. | AG-04 | doc 0001 § 4.3; doc 0003 AG-04.1, R-04; enforced by AG-04.4 |
| `VL2-EVT-02` | **run lifecycle (event family)** | The event family bracketing a run: exactly one run-start precedes all other events, exactly one run-end follows them, carrying the run's typed outcome (`VL2-EVT-10`). | AG-04 | doc 0001 § 4.3; doc 0003 AG-04.2, R-04 |
| `VL2-EVT-03` | **turn lifecycle (event family)** | The event family bracketing a turn: turn-start/turn-end pairs nest strictly inside the run bracket and never overlap; turn-end carries the turn's typed outcome (`VL2-EVT-11`) on every exit path, not only the happy one. | AG-04 | doc 0001 § 4.3; doc 0003 AG-04.2, R-04 |
| `VL2-EVT-04` | **message lifecycle (event family)** | The event family for one assistant message: start, deltas, end, with reasoning distinguished from text at the event-kind level. A message reconstructed from its deltas is equal, as a Layer 1 message value, to the message the loop holds directly — one source of truth. | AG-05 | doc 0001 § 4.3; doc 0003 AG-05.1, R-04 |
| `VL2-EVT-05` | **tool execution (event family)** | The event family for one tool call: start (call identity, tool name, arguments), optional indexed progress, end. The end event distinguishes three typed outcomes — success-with-result, the tool ran but its result reports failure, and execution itself failed — by type, never by convention over payload contents. | AG-05 | doc 0001 § 4.3; doc 0003 AG-05.2, R-04, R-13 |
| `VL2-EVT-06` | **permission (event family)** | The event family reporting the permission protocol (`VL2-LOOP-07`) on the stream: a decision is required, a decision was made (carrying the full four-outcome vocabulary), and a resolution was remembered — distinct from decision-made because it is a fact about *future* calls a session log needs. | AG-06 | doc 0001 § 4.3 (absent — G1); doc 0003 AG-06.1, R-04, G1 |
| `VL2-EVT-07` | **cost (event family)** | The event family carrying `VL2-COR-16` cost scope onto the stream: per-turn and cumulative token figures, each labelled estimate or final. | AG-06 | doc 0001 § 4.3 (absent — G10); doc 0003 AG-06.2, R-04, R-16, G10 |
| `VL2-EVT-08` | **delegation (event family)** | The event family making the delegation tree walkable: subagent-started and subagent-ended, parent-identified, so a consumer separates a delegated conversation from the main one by walking parent identifiers. | AG-06 | doc 0001 § 4.3 (absent — G7); doc 0003 AG-06.3, R-04, R-14, G7 |
| `VL2-EVT-09` | **compaction (event family)** | The event family reporting what a compaction operation (`VL2-SEAM-03`) did: started, finished (identifying the replaced span and the summary identity), and failed — distinct from finished, so recovery has something to reason from. | AG-06 | doc 0001 § 4.3 (absent — G3); doc 0003 AG-06.4, R-04, R-11, G3 |
| `VL2-EVT-10` | **run outcome** | The typed value a run-end event carries: completed, interrupted, or failed. Never absent — a run has no "sometimes no terminal at all" case, because `VL2-EVT-19` protects run-end specifically. | AG-04 | doc 0003 AG-04.2, R-05 invariant 4; AG-01.1's decision (the run-end protection clause) |
| `VL2-EVT-11` | **turn outcome** | The typed value a turn-end event carries, distinguishing "model finished" from "turn aborted" — turn-end is emitted on every exit path, not only the happy one. | AG-04 | doc 0003 AG-04.2, R-05 invariant 4 |
| `VL2-EVT-12` | **indexed deltas (envelope invariant)** | Envelope invariant 1: a delta carries an index and only the new fragment, never a snapshot of the accumulated message. No route through the envelope's public construction surface can attach an accumulated-message payload to a delta. | AG-04 | doc 0001 § 4.3 invariant 1; doc 0003 AG-04.3, AG-05.1, R-05 |
| `VL2-EVT-13` | **explicit nesting (envelope invariant)** | Envelope invariant 2: an event belonging to a delegated harness carries its parent identifier; a top-level event carries none. The field exists from the envelope's birth, before any delegation mechanism exists, because explicit nesting cannot be retrofitted. | AG-04 | doc 0001 § 4.3 invariant 2; doc 0003 AG-04.1, AG-19.1, R-05 |
| `VL2-EVT-14` | **non-blocking observers (envelope invariant)** | Envelope invariant 3: an observer attached to the stream is never synchronous on the streaming path — a slow observer must not stall token delivery. This row states the requirement the envelope must satisfy; `VL2-SEAM-12` observer asynchrony is the structural mechanism that makes it true and is cross-referenced from here, never restated. | AG-04 | doc 0001 § 4.3 invariant 3; doc 0003 R-05; mechanism decided at AG-01.1, tested at AG-20.2 |
| `VL2-EVT-15` | **typed errors and outcomes (envelope invariant)** | Envelope invariant 4: a failure is a typed value reachable through the typed-failure surface, never a message string a consumer must parse; setup failures (pre-stream) and stream failures (mid-stream) are distinguishable. | AG-04 | doc 0001 § 4.3 invariant 4; doc 0003 AG-04.3, AG-11.2, R-05 |
| `VL2-EVT-16` | **stream-contract validator** | The reusable checker that runs over any hand-built or produced agent event sequence and accepts it only if every envelope invariant and lifecycle bracket holds. Exists so the invariants are assertable before any producer exists, and is reused wholesale by the Layer 3 readiness contract's kit (`VL2-SEAM-14`). | AG-04 | doc 0003 AG-04 charter deliverable, R-04; reused at AG-23 |
| `VL2-EVT-17` | **carrier** | The mechanism by which a consumer receives agent events at the Layer 2 boundary — decided independently of Layer 1's own `V-STR-02` carrier because it applies at a different boundary (agent events, not model-response events). Documented default: channels, for symmetry with the Layer 1 decision and the same caller-owns-the-context liveness rule. | AG-01 | doc 0003 AG-01.1 closing checklist item 1, R-09; doc 0001 § 2.3; Layer 1 `V-STR-02` (the analogous, non-identical decision) |
| `VL2-EVT-18` | **loop-internal turn-scoped loss** | The loss posture at the boundary between the loop and the harness: Layer 1's rule, unchanged. On the harness's own cancellation of a turn with a send in flight, late turn events drop and the turn stream closes without its terminal. What may be lost: turn events the harness had not yet received. Nothing more is promised — the loss fires only on the harness's own cancellation, and only the harness is a consumer at this boundary. | AG-01 | AG-01.1's decision (the loop-harness loss posture); Layer 1 `V-STR-09` sanctioned loss path (the rule this posture reuses unchanged) |
| `VL2-EVT-19` | **harness-facing history-guarded truncation** | The loss posture at the boundary between the harness and its attached consumers: Layer 1's mechanism, narrowed. The loss path never discards an event describing a fact already committed to history (`VL2-HAR-01`), and bounded wind-down (`VL2-HAR-05`) finishes delivering everything history holds, as part of or immediately preceding run-end. What may be lost: facts the harness never learned before cancellation — in-flight deltas of a turn cut short, progress of tools that never completed, everything of turns that never began. What may never be lost: any event describing a committed history fact, and run-end itself. | AG-01 | AG-01.1's decision (the harness-consumer loss posture); doc 0003 AG-14.3 (the time bound this posture depends on) |

---

## 3. Loop mechanics — `VL2-LOOP` (9 terms)

Owners AG-08, AG-09, AG-10, AG-11. What exists entirely within one loop invocation.

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `VL2-LOOP-01` | **pre-request hook** | The seam immediately before the provider call — the last moment the outgoing request exists as data. An installed hook observes the fully-assembled request and its return value is what the provider receives; the identity default (no hook installed) changes nothing; a failing hook aborts the turn before any I/O, attributing the hook in a typed error. | AG-08 | doc 0001 § 6 seam 1; doc 0003 AG-08.1, R-12, R-18 seam 1, G11; Layer 1 `V-REQ-29` request rebuild (the mechanism this seam stands on) |
| `VL2-LOOP-02` | **prefix stability** | The property that unchanged inputs (system material, tools, hook) yield a byte-stable prefix across turns: the tool and system regions of the captured request are byte-identical turn to turn, and the message region grows strictly by append. Closes the Layer 2 half of cache-breakpoint discounting — a silent prefix break is a silent order-of-magnitude input-cost regression. | AG-08 | doc 0001 § 3.2; doc 0003 AG-08.2, R-12, G4 |
| `VL2-LOOP-03` | **tool execution contract** | What a tool is to Layer 2: a Layer 1 tool declaration, an effect class (`VL2-LOOP-04`), and a typed failure mode, exposed so a scheduler in another package can consume it without acting on the tool itself. The loop schedules against this contract; it never executes a tool's own behavior — the tool trap, applied. | AG-09 | doc 0001 § 4.1; doc 0003 AG-09.1, R-13, R-18 seams 2–3; doc 0003 § Scope boundary trap 1 |
| `VL2-LOOP-04` | **effect class** | The scheduler's concurrency-policy discriminator on a tool: at minimum read, mutating, or execute. Reads may run concurrently, up to a documented bounded fan-out — no more than the bound run at once, and all complete; mutating and execute-class calls serialize among themselves in call order. | AG-09 | doc 0001 § 4.1; doc 0003 AG-09.1, AG-09.2, R-13 |
| `VL2-LOOP-05` | **ordered rejoin** | The rule that tool results rejoin the transcript in call order, regardless of completion order — several providers reject results that do not correspond positionally to their calls. Layer 1's call ordinal (`V-STR-21`) is what makes the reordering possible; correlation identities, including any synthetic ones an adapter minted, survive the rejoin exactly. | AG-09 | doc 0001 § 2.3 item 5, § 7 G5; doc 0003 AG-09.3, R-13; Layer 1 `V-STR-21` call ordinal |
| `VL2-LOOP-06` | **policy slot** | The opaque per-call parameter the tool execution call carries and never reads. Confinement is a property of the call site, not of the tool; the loop passes the exact value injected and interprets none of it — the seam a Layer 3 sandbox implementation fills. | AG-09 | doc 0001 § 6 seam 3; doc 0003 AG-09.1, R-18 seam 3, G2 |
| `VL2-LOOP-07` | **permission protocol** | The ask–suspend–resume protocol around every scheduled call, never the policy answer: ask the injected policy; if it defers, emit decision-required and suspend that call without blocking anything else; resume on the decision. Four outcomes complete the protocol — allow-once (executes as scheduled), allow-always (executes, resolution reported remembered-eligible to the policy), deny (execution skipped, a typed denial result rejoins in call order), and modify-input (executes with the modified arguments, both original and modified recorded). A decision addressed to an unknown or already-decided call is a typed protocol error, never a silent no-op. | AG-10 | doc 0001 § 4.1, § 6 seam 2; doc 0003 AG-10.1, AG-10.2, R-10, G1 |
| `VL2-LOOP-08` | **finish-reason dispatch** | The loop's exhaustive dispatch over Layer 1's closed finish-reason vocabulary into distinct typed turn outcomes: tool-calls continues into scheduling, natural-stop completes, refusal completes-with-refusal (a decline is a final answer), pause yields a suspended-resumable outcome (resume, not stop or guess) — refusal and pause never collapse into one outcome. A finish reason with no mapped outcome fails the build or the suite until the loop handles it. | AG-11 | doc 0001 § 3.2, § 3.3 row 3, G12(c); doc 0003 AG-11.1, R-08; Layer 1 `V-MET-01` … `V-MET-08` |
| `VL2-LOOP-09` | **typed turn failure** | Layer 1's failure taxonomy carried into turn scope: category, retryability, and the partial-output discriminator, plus any partial assistant content, inspectable as typed values and reported upward on the corresponding typed failure event — never a message string. | AG-11 | doc 0001 § 4.3 invariant 4; doc 0003 AG-11.2, R-08, R-18 seam 7; Layer 1 `V-FAIL-05` … `V-FAIL-10` |

---

## 4. Harness mechanics — `VL2-HAR` (10 terms)

Owners AG-12, AG-13, AG-14, AG-15, AG-16, AG-21. What the harness does between or around loop invocations, invariantly across harness configurations, plus the combined-adversarial proof (`VL2-HAR-10`) that those invariants hold together, not merely singly.

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `VL2-HAR-01` | **history** | The harness's append-only-within-a-run, boundary-validated transcript store. It rejects any commit that would orphan a tool call — a tool-result entry whose call identity has no prior call, or a turn closing with a call that has no result — through exactly one commit path, with no privileged bypass for internal callers. Seeded construction, over a pre-existing transcript, validates by the same rule. | AG-12 | doc 0001 § 4.2; doc 0003 AG-12.1, R-07 |
| `VL2-HAR-02` | **orphan synthesis** | Completing every tool call orphaned by interruption with a synthesized result, typed as an interruption artifact and distinguishable from a real result, before the next turn runs. Idempotent and total: applying synthesis to N orphans closes exactly N pairs on the first application and changes nothing on the second. | AG-12 | doc 0001 § 4.2; doc 0003 AG-12.2, R-07 |
| `VL2-HAR-03` | **run driver** | The harness's control loop over the loop: append the user message, run one turn, execute and append its results, repeat, until a terminal finish reason, emitting run lifecycle events and accepting queued steering input throughout. It reaches the loop only through the loop's own public one-turn surface — no privileged channel (`VL2-COR-23`). | AG-13 | doc 0001 § 4.2; doc 0003 AG-13.1, R-08, R-09 |
| `VL2-HAR-04` | **interrupt vs shutdown** | Two distinguishable cancellation signals, never conflated. Interrupt aborts the in-flight turn and keeps the session — a new prompt on the same harness works afterward. Shutdown performs the same wind-down and then refuses new prompts, typed. Both are idempotent and distinguishable through every layer they cross, including run-end outcomes and error chains. | AG-14 | doc 0001 § 4.2; doc 0003 AG-14.1, AG-14.2 |
| `VL2-HAR-05` | **bounded wind-down** | The guarantee that cancellation, either signal, completes within a documented time bound, even against a tool that ignores cancellation entirely. An offending call is reported typed — which tool, still running — with its task detached and named, never silently abandoned; nothing belonging to the harness itself survives the wind-down. | AG-14 | doc 0003 AG-14.3, R-08 |
| `VL2-HAR-06` | **retry policy (harness half)** | The harness's decision, over Layer 1's typed evidence, whether to retry a failed turn: a pre-output retryable failure retries within a documented bound as a fresh provider call over an identical transcript, each attempt visible on the stream; any failure after emitted output is surfaced, never silently retried; a terminal category never retries; backoff waits on the context, and retry-after overrides computed backoff when the failure carries one. | AG-15 | doc 0001 § 6 seams 7, 8, § 7 G8; doc 0003 AG-15.1, AG-15.2, R-15; Layer 1 `V-FAIL-07`, `V-FAIL-09`, `V-FAIL-15` |
| `VL2-HAR-07` | **failover seam** | The named injection point consulted once the harness's own retries exhaust. Its v1 implementation declines, and observable behavior with and without the seam installed is identical; a real implementation must re-count the token budget and restart the cache prefix, and its contract documents that obligation now so it need not be discovered later. | AG-15 | doc 0001 § 6 seam 8; doc 0003 AG-15.3, R-15, G8 |
| `VL2-HAR-08` | **composed bounds** | The combined retry ceiling: harness attempts multiplied by Layer 1's own wire-level attempts beneath them. Stated and tested explicitly, in the documentation both layers' readers will find, so the first rate-limit storm is not the first time anyone computes it. | AG-15 | doc 0003 AG-15.2 note, R-15 |
| `VL2-HAR-09` | **cost aggregation** | Maintaining cumulative cost figures across a run from Layer 1 usage: cumulative equals the per-turn sum including retried attempts (a retried attempt's tokens are real spend) and compaction spend; absent usage on any turn yields a cost event that reports absence, never an invented zero; any figure emitted before the stream's final usage update is labelled estimate and corrected by the final, labelled-final figure. | AG-16 | doc 0003 AG-16.1, R-16; Layer 1 `V-MET-11` absence versus zero |
| `VL2-HAR-10` | **hardening suite** | The combined-adversarial proof that every wave 2–5 harness feature — suspension, interrupt, steering, compaction, delegation — survives running together, not merely singly, over the whole assembled harness: no goroutine leaks on any exit path, checked package-wide by reusing Layer 1's AI-22 leak-detection mechanism wholesale, never a Layer-2-authored replacement; a slow consumer loses nothing and observes contract order, proving `VL2-SEAM-12` observer asynchrony and the `VL2-EVT-18`/`VL2-EVT-19` loss postures hold under pressure and in combination — cited here, never restated; an abandoned consumer who cancels winds down within the documented bound, proving `VL2-HAR-05` holds combined rather than only singly. It proves properties this register already defines and defines none of its own; performance is explicitly out of scope — correctness under pressure only. | AG-21 | doc 0003 AG-21 charter (Goal, Deliverable, Acceptance), R-05, R-08, R-17; Layer 1's AI-22 leak-detection mechanism (reused, not redefined) |

---

## 5. Cross-cutting seams — `VL2-SEAM` (16 terms)

Owners AG-00, AG-01, AG-17, AG-18, AG-19, AG-20, AG-22, AG-23. Injected variation points and cross-cutting contracts whose v1 implementation may decline while the name must persist.

> The exact attribute vocabulary — span names and the decided per-span attribute list for run, turn, tool-execution and compaction spans — is deliberately deferred to `AG-22.1`'s own `[decision]` node (doc 0003 AG-22.1 closing checklist item 1), not decided by this register. `VL2-SEAM-15` and `VL2-SEAM-16` fix only the boundary `AG-22.1` must decide inside. The deferral is recorded here, in the live register, rather than only in this change's own `proposal.md` out-of-scope table, so that a later charter cites the register and not an archived change folder.

| Id | Term | Definition | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `VL2-SEAM-01` | **context strategy** | The injected seam consulted before every provider call, with the current transcript and the model's budget, deciding whether and how to compact. The v1 default never compacts, and using it changes nothing observable — no compaction event, no history mutation. | AG-17 | doc 0001 § 6 seam 5; doc 0003 AG-17.1, R-11, R-18 seam 5, G3 |
| `VL2-SEAM-02` | **token accounting** | Answering how many tokens a request would consume before it is sent, distinct from Layer 1's own `V-MET-09` usage, which reports what a response already consumed. Discovered as an optional Layer 1 capability by type assertion; where absent, an estimate is used and is documented and labelled as an estimate everywhere it is consumed — never treated as exact. | AG-17 | doc 0001 § 6 seam 6; doc 0003 AG-17.2, R-18 seam 6, G3; Layer 1 `V-PRV-17` token counting |
| `VL2-SEAM-03` | **compaction** | A model call with its own provider, model, cost and cancellation, triggered by the context strategy (`VL2-SEAM-01`), that replaces a compactable span (`VL2-SEAM-04`) with a summary while protecting recent turns (`VL2-SEAM-05`) and never orphaning a call/result pair. Recoverable if interrupted: an interrupted compaction leaves the pre-compaction transcript intact and usable, with a typed failure event on the stream. | AG-18 | doc 0001 § 4.2, § 6 seams 5–6, § 7 G3; doc 0003 AG-18.1, AG-18.4, R-11 |
| `VL2-SEAM-04` | **compactable span** | The region of the transcript compaction may replace. Its boundary never splits a call/result pair: where a naive boundary would split one, the boundary moves to include the whole pair, by construction — the resulting transcript still passes history's boundary validation (`VL2-HAR-01`). | AG-18 | doc 0003 AG-18.2, R-11, R-07 |
| `VL2-SEAM-05` | **protected turns** | The recent turns a compaction configuration protects from replacement. After compaction, protected turns are byte-identical to their pre-compaction form. | AG-18 | doc 0001 § 4.2; doc 0003 AG-18.2, R-11 |
| `VL2-SEAM-06` | **summary entry (compaction artifact)** | Answers boundary case 2. A compaction summary is a transcript entry, not metadata beside history: it occupies a slot in history, typed as a compaction artifact and distinguishable from a model message, and it must pass history's own boundary validation (`VL2-HAR-01`) after compaction. | AG-18 | doc 0003 AG-18.2, AG-00.1 closing checklist item 1, R-06, R-11 |
| `VL2-SEAM-07` | **on-demand entry point** | The invocation of compaction outside its strategy trigger, at a turn boundary, using the same mechanics as a strategy-triggered compaction — the two are one path observed two ways, and their event sequences are equal. Refused typed when demanded mid-turn: compaction happens only at turn boundaries, never queued silently and never racing the loop. | AG-18 | doc 0003 AG-18.5, R-11 |
| `VL2-SEAM-08` | **compaction call** | Answers boundary case 4 (the fifth boundary case). A compaction call is a provider call (`VL2-COR-06`) but not a turn (`VL2-COR-05`): it produces a summary, not an assistant response with tool results, so it fails the turn definition while satisfying the provider-call definition. It carries its own provider, cost and cancellation, and is never an attempt (`VL2-COR-07`), because it is never made in service of a turn already begun. | AG-00 | doc 0003 § "A fifth boundary case is added to the closing checklist" (proposal decision 6); doc 0003 AG-18.1 ("a model call with its own provider, cost and cancellation") |
| `VL2-SEAM-09` | **re-entrancy** | The structural property that the harness is invocable from within a tool execution: a nested run, nested cancellation, nested cost, parent-identified events, and a permission scope derived from the parent (`VL2-SEAM-10`). Proven, not merely documented: sibling child harnesses interleave without cross-talk when run concurrently. | AG-19 | doc 0001 § 6 seam 12; doc 0003 AG-19.1, AG-19.2, R-14, G7 |
| `VL2-SEAM-10` | **derived permission scope** | A subagent's policy scope, derived from its parent's: what the parent's policy already allowed flows down to the child without asking again, and what it would ask about is asked on the parent's stream — one place a human watches, never a second permission surface. | AG-19 | doc 0003 AG-19.3, R-14, G7 |
| `VL2-SEAM-11` | **observer** | A consumer attached at the observer attachment point alongside a run's primary consumer — for example a session logger or a cost meter. No observer is privileged at the mechanism level; the run's primary frontend is one more attached consumer, and privilege among them is Layer 3 policy, never a Layer 2 property. | AG-01 | AG-01.1's decision (the observer attachment point); doc 0003 AG-01.1 closing checklist item 3 |
| `VL2-SEAM-12` | **observer asynchrony** | The structural guarantee, not a convention, that a stalled observer (`VL2-SEAM-11`) cannot stall the streaming path: every path from a stalled consumer's own receive terminates at that consumer's own delivery lane, never backing up to the producer or to any other consumer. This is the mechanism that makes envelope invariant 3 (`VL2-EVT-14`) true; it is cited from that row, never restated there. | AG-01 | doc 0001 § 4.3 invariant 3; doc 0003 AG-01.1 closing checklist item 3, AG-20.2, R-05, R-17; AG-01.1's decision (the observer-asynchrony mechanism) |
| `VL2-SEAM-13` | **hook taxonomy** | The four hook points and the discipline that governs them: pre-request and pre-compact may mutate; post-turn and session-start only observe. Every hook point fires at its documented moment with its documented payload, and hooks registered at one point run in registration order, deterministically. A mutating hook's failure is typed and attributed to that hook. | AG-20 | doc 0001 § 7 G11; doc 0003 AG-20.1, R-17, G11 |
| `VL2-SEAM-14` | **readiness contract** | The frozen v1 surface a Layer 3 application may consume — every seam's injection point and its v1 default — proved sufficient by an external-package consumer proof that builds a generic Layer 3 application in miniature, never a coding-agent miniature, plus the deterministic scripted-harness test kit that application is handed to build sessions on. | AG-23 | doc 0001 § 4 amendment (2026-08-10, "a Layer 3 application"); doc 0003 AG-23.1, AG-23.3, R-21 |
| `VL2-SEAM-15` | **the observability boundary** | The injected discipline governing every OTel span the runtime emits for a run, a turn, a tool execution, or a compaction: telemetry crosses the boundary only through the OTel **API**, never the SDK — the SDK and every exporter are the composition root's own concern (AG-22's own charter: "Exporters, SDK, dashboards — composition root"), never Layer 2's. With no tracer configured, a traced run's observable behavior is identical to the same run untraced, and nothing panics — the same declines-safely pattern `VL2-SEAM-01` context strategy and `VL2-HAR-07` failover seam already establish. `R-19` states Layer 2's attribute vocabulary is a recorded extension of ADR 0005 § D3's allowlist, which covers Layer 1 spans only; deciding that extension — the exact span names and attributes — is deliberately deferred to `AG-22.1`'s own `[decision]` node, never invented here or in any downstream SDD (see the note under this category's heading). `VL2-SEAM-16` content denylist states what the boundary forbids absolutely, regardless of which attributes `AG-22.1` later decides to allow. | AG-22 | doc 0003 AG-22 charter (Goal, Deliverable, Out of scope), R-01, R-19; ADR 0005 § D3 |
| `VL2-SEAM-16` | **content denylist** | The absolute prohibition on Layer 2 telemetry, inherited from ADR 0005 § D3 and restated here, never weakened: no prompt, completion, reasoning, tool-argument or tool-result text, HTTP header, or credential may appear in any span the runtime emits, proven by scanning a full-featured run's recorded telemetry for absence — never merely by omission from an allowlist. Layer 1 carries the identical discipline as its own milestone (redaction); Layer 2's obligation under `VL2-SEAM-15` is the same rule applied to its own spans, never a separate or weaker one. | AG-22 | ADR 0005 § D3 (attribute denylist, absolute); doc 0003 AG-22 charter Acceptance, R-19 |

---

## 6. Excluded — `VL2-OUT` (9 terms)

Named and attributed, deliberately **not** defined here — defining a Layer 3 concept inside a Layer 2 artifact is the first step toward Layer 2 implementing it. Where a Layer 1 register row already assigns the concern, it is cited rather than re-attributed.

| Id | Excluded term | Definition (attribution only, never Layer 2's own) | Owner | Provenance |
| --- | --- | --- | --- | --- |
| `VL2-OUT-01` | **permission policy content** | Which calls are allowed, and whether an answer should be remembered. Layer 2 owns only the ask–suspend–resume protocol (`VL2-LOOP-07`); easy to misread as Layer 2's because the protocol lives there — the content of the answer never does. | Layer 3 — the permission-policy port | doc 0001 § 5.1, § 7 G1; Layer 1 `V-OUT-05` permission |
| `VL2-OUT-02` | **sandbox semantics** | Under what confinement a tool call executes. Layer 2 carries only the opaque policy slot (`VL2-LOOP-06`) the execution call passes through unread; what the slot means, and what it confines, is never defined here. | Layer 3 — the sandbox port | doc 0001 § 5.1, § 6 seam 3, § 7 G2; Layer 1 `V-OUT-16` sandbox |
| `VL2-OUT-03` | **tool source** | Where the tool set offered on a request came from, and whether it changes between turns. Layer 2 receives a tool set and schedules against it (`VL2-LOOP-03`); it never resolves, discovers, or filters tool sources. | Layer 3 — the tool-source port | doc 0001 § 5.1, § 6 seam 4, § 7 G6; Layer 1 `V-OUT-17` tool source |
| `VL2-OUT-04` | **summary quality** | What makes a good compaction summary. Layer 2 owns compaction's mechanics (`VL2-SEAM-03`); the summarization instruction itself arrives injected, never authored by the runtime, and its quality is never this artifact's concern. | Layer 3 — the injected summarization instruction | doc 0003 AG-18 charter deliverable ("the summarization instruction arrives injected"), out-of-scope clause |
| `VL2-OUT-05` | **cross-session rule persistence** | Remembering a permission resolution across sessions. Layer 2's only obligation is the remembered-resolution report on the current run, part of `VL2-LOOP-07`'s four-outcome vocabulary; storing it across sessions is never this artifact's concern. | Layer 3 — session persistence | doc 0001 § 7 G1 disposition; doc 0003 AG-10.4 out-of-scope clause |
| `VL2-OUT-06` | **price and money** | Converting tokens into money. Layer 2's cost event is token-only (`VL2-COR-16`); money joins the stream as Layer 3 enrichment. Both sides of the reconciled conflict are cited. | Layer 3 — the price-table port | doc 0001 § 4.3 (the losing reading: "tokens, cache hits and money"); ADR 0005 § D4 row G10, doc 0001 § 7 G10 ("L2 emits, L3 prices" — the reading that wins); Layer 1 `V-OUT-08` price |
| `VL2-OUT-07` | **session persistence** | Append-only session records with parent chains, under a Layer 3 application's own storage. Layer 2 exposes a transcript (`VL2-COR-10`) for a session to persist; it never writes one itself — `VL2-COR-18` no ambient authority makes this mechanical, not merely documented. | Layer 3 — session persistence | doc 0001 § 5.2; Layer 1 `V-OUT-03` session |
| `VL2-OUT-08` | **frontends** | Any renderer of the runtime's event stream — a TUI, print mode, a future IDE integration. Layer 2 knows none of them exist (`VL2-COR-22` no frontend knowledge) and emits the same event stream regardless of which, if any, is attached. | above Layer 3 — consumes events only | doc 0001 § 2.2, § 8; Layer 1 `V-OUT-09` frontend |
| `VL2-OUT-09` | **catalogs** | Provider catalogs, model selection, and credential resolution. Layer 2 receives a model identity and calls the provider it is given; it never resolves a catalog, selects a model, or reads a credential. | the composition root — provider catalog and credential resolution | doc 0001 § 5.2; Layer 1 `V-OUT-14` provider catalog / credential resolution |

---

## Term count

`VL2-COR` 23 · `VL2-EVT` 19 · `VL2-LOOP` 9 · `VL2-HAR` 10 · `VL2-SEAM` 16 · `VL2-OUT` 9 = **86 terms**.

**Consumers of this register.** AG-01 (`cachicamas-agent-event-delivery`) and AG-02 (`cachicamas-agent-v1-scope`) are written from it next, in this same wave. Every AG-01 domain noun (observer, upward path, steering, carrier, interrupt) and every AG-02 domain noun (failover seam, subagent, the forward-requirement identifiers it disposes) resolves against a row above.
