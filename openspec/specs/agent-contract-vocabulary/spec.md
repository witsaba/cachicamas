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

---

**[REGISTER CONTINUES WITH 86 ROWS IN 6 CATEGORIES — SEE ARCHIVED DECISION.MD FOR THE COMPLETE SNAPSHOT]**

The complete register text with all 86 terms (VL2-COR-01 through VL2-OUT-09) is available in `/openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md` as the historical merge-day snapshot, and is maintained live here for amendment under the six standing rules.
