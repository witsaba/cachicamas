# Design — Layer 2 contract vocabulary

> **Change**: `cachicamas-agent-contract-vocabulary`
> **Milestone**: AG-00 · **Node**: AG-00.1 `[decision]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Inputs**: `explore.md`, `proposal.md` (the spec phase runs in parallel; requirement ids are cited by property, not number)
> **Output**: the structure and rules that the register (spec delta) and `decision.md` implement
> **Diagrams**: ASCII (project convention)
> **Precedent**: `openspec/changes/archive/2026-07-31-cachicamas-ai-contract-vocabulary/design.md` — followed in shape; every departure is stated with its reason

---

## 1. What is being designed

Not software. The artifact. This document answers four questions:

1. **How is a candidate term assigned to exactly one category**, so that no term can plausibly sit in two?
2. **Why the identifier scheme is `VL2-<CAT>-nn`**, against the two live alternatives, and how a Layer 3 register extends it without collision.
3. **Why each of the five row fields is load-bearing** — in particular why the owner column holds exactly one current doc 0003 milestone id, never a range, never two.
4. **What makes the amendment rule self-enforcing** in a repository with no CI, rather than merely well-intentioned.

The argument for *why a vocabulary-first milestone exists* is the proposal's (§ Intent) and is not repeated here. Layer 1's design proved the mechanism — one definition stops the collision, one owner stops the renegotiation — from four shipped defects; this layer inherits the mechanism and grounds it in its own evidence: two doc 0001 vocabulary amendments and four doc 0003 plan defects, all recorded before any Layer 2 code exists.

---

## 2. The category scheme and its decision procedure

### 2.1 The defect this section closes

A category scheme in which a term could sit in two categories reproduces, inside the register, the exact ambiguity the register exists to kill: two readers file the same concept in two places and neither notices. The countermeasure is not better category *descriptions*; it is a **decision procedure** — six inclusion tests applied in a fixed order, where the first test that fires wins and later tests are never consulted.

### 2.2 The six tests, in application order

| Order | Category | Inclusion test (one line) |
| --- | --- | --- |
| 1 | `VL2-OUT` | Is the concept defined by another layer, so this register may only name it and its owner? |
| 2 | `VL2-EVT` | Is the term a constituent, family, invariant, or outcome of the agent event envelope — visible to a consumer holding only the stream? |
| 3 | `VL2-COR` | Is the term needed to state the loop/harness responsibility split itself, or a unit or interaction of the run's timeline that both loop-side (AG-07 … AG-11) and harness-side (AG-12 … AG-20) milestones must cite? |
| 4 | `VL2-LOOP` | Does the term describe something that exists entirely within one loop invocation — between receiving a transcript and returning one turn? |
| 5 | `VL2-HAR` | Does the term describe what the harness does between or around loop invocations, **invariantly across harness configurations**? |
| 6 | `VL2-SEAM` | Is the term an injected variation point or cross-cutting contract whose v1 implementation may decline while the name must persist? |

A candidate failing all six tests is either another layer's concept misread as Layer 2's (return to test 1 with the owning layer named) or not a vocabulary term at all — a design decision belonging to AG-01, AG-02 or AG-22, per the proposal's out-of-scope table.

### 2.3 The procedure resolves the known ambiguities

Every ambiguity the exploration surfaced is decided by test order, not judgement:

| Candidate | Plausible second home | Resolution by the procedure |
| --- | --- | --- |
| pairing invariant | `VL2-HAR` — history enforces it at the boundary | Test 3 fires first: the loop constructs the pairs, the harness validates them; both families cite it. `VL2-COR` |
| steering message | `VL2-HAR` — the queue is harness mechanics | Test 3 fires: the term names an interaction of the run timeline (defined by its turn-boundary placement), not the queueing mechanism. `VL2-COR` — the owner is still AG-13; category and ownership are different axes (§ 4.2) |
| compaction | `VL2-HAR` — the harness triggers it | Test 5 **fails**: the v1 default never compacts, so compaction is not invariant across harness configurations — it arrives through the context-strategy seam. Test 6 catches it. `VL2-SEAM` |
| typed turn failure | `VL2-HAR` — retry consumes it | Test 4 fires first: it is produced entirely within one loop invocation by finish-reason dispatch. `VL2-LOOP`; AG-15 appears in provenance as a consumer, never as a second row |
| observer asynchrony vs envelope invariant 3 | apparently one term with two homes | Two terms, each single-homed: the invariant is a property of the envelope (test 2, `VL2-EVT`); observer asynchrony is the attachment contract (test 6, `VL2-SEAM`), and its row **cross-references** the invariant row rather than restating it |
| the delegation cluster | `VL2-COR`, `VL2-EVT` and `VL2-SEAM` all plausible | Three terms, three homes: delegation and subagent (timeline participants, test 3, `VL2-COR`); the two delegation-family event kinds (test 2, `VL2-EVT`); re-entrancy / child harness (the mechanism whose v1 tool declines, test 6, `VL2-SEAM`) |
| cost | three readings | Three terms: the cost event's token-only scope (test 3, `VL2-COR`); the cost event family (test 2, `VL2-EVT`); cost aggregation (test 5 — aggregation is configuration-invariant harness work, `VL2-HAR`) |

The pattern in the last four rows is the important one: where a concept seems to belong in two categories, the correct move is almost always to notice it is **two or three distinct terms**, define each once, and connect them by cross-reference. One term, one row, one home — the procedure forces the split instead of allowing a compromise placement.

---

## 3. The identifier scheme

### Decision: `VL2-<CAT>-nn`

**Choice**: `VL2-` prefix, category code, category-scoped append-only ordinal.

**Alternatives considered**:

| Candidate | Why it loses |
| --- | --- |
| `V-<CAT>-nn` (Layer 1's bare prefix) | Both registers sit side by side under `openspec/specs/`, and doc 0003 already cites Layer 1 rows (`V-REQ-*`, `V-MET-*`, `V-STR-*`) **inside Layer 2 milestone prose**. A bare `V-` citation would be ambiguous about which register resolves it, and a plain-text search for `V-` would match both. Ambiguity in citations is this change's stated failure mode (proposal risk 8) |
| `AG-<CAT>-nn` | `AG-` is already doc 0003's milestone/node namespace; `AG-06` (a milestone) and a hypothetical `AG-EVT-06` (a term) would collide in a reader's pattern recognition |
| `V2-<CAT>-nn` (the exploration's working prefix) | Textually disjoint, but `2` reads as a *version* of the vocabulary rather than the *layer* it describes. The misreading is not hypothetical: doc 0003 refers to doc 0001 throughout as "the v2 reference", so `V2-` ids would visually rhyme with a document-version marker in the very prose that cites them. It also forces `V3-` on the next layer, compounding the version misreading |

**Rationale**: `V` keeps continuity with the register family; `L2` names the layer; the pair is textually disjoint from `V-`, so no search string and no citation can cross registers.

**Extension rule** (recorded now so the next register does not renegotiate it): a layer-*n* register uses `VL<n>-`; its category codes are that register's own choice, sized to that layer's shape. Layer 1's bare `V-` is grandfathered as the family's only unmarked prefix and is never reused. Under this rule every cross-register citation is self-describing by prefix alone, at any number of layers.

---

## 4. Structure of the register

### 4.1 Row shape

Every term is one row with five fields, in this order — Layer 1's shape, unchanged, because both of its Wave 0 amendments worked *within* this shape without straining it:

```
  VL2-<CAT>-nn | term name   | definition   | owning milestone   | provenance
  ------------   ---------     ----------     ----------------     ----------
  stable id      conceptual    what it IS     exactly one AG-NN    doc 0001 § +
  append-only    noun phrase   + what it      (current doc 0003    R-01..R-21 /
                 with spaces   is NOT         id, single-valued)   G1..G11 + V-*
                                                                   row or L1 file
```

Why each field is load-bearing:

- **The id is stable and append-only**, so amendment never renumbers and a citation in a merged charter never dangles. This is doc 0003's own id discipline applied to terms (locked constraint 3).
- **The term is a noun phrase with spaces**, which makes a leaked Go identifier visually anomalous — a single-token camel-case name in a column of noun phrases is caught by a glance, not a hunt. This is what makes the no-Go-identifiers constraint cheap to review.
- **The definition states what the term is and what it is not.** The negative half is constitutive, not decorative: both doc 0001 corrections were failures of an unstated negative — "the portable brain" lacked *does not think or decide*, and "Layer 3" lacked *not the coding agent*. Each row's negative half is where the next such correction is pre-paid.
- **The owner is exactly one current doc 0003 milestone id** — see § 4.2.
- **Provenance carries the doc 0001 section, the doc 0003 requirement (`R-01` … `R-21`) or forward requirement (`G1` … `G11`), and — for a reused Layer 1 identity — the `V-*` row or the shipped Layer 1 file cited by path, never by exported name.** Not line numbers: they drift, and a drifted citation is worse than a coarse one (Layer 1's recorded rule).

### 4.2 Why the owner column is single-valued — never a range, never two ids

Layer 1's design records the mechanism this rule buys, and it transfers unchanged:

- **Two owners is zero owners.** A term owned by "AG-12 and AG-18" is a term either may adjust inside its own PR, which restores exactly the drift mechanism the register exists to remove — the definition renegotiated by whichever milestone is under implementation pressure. With one owner, any other milestone's edit to the row is a visible cross-milestone change, reviewed as one.
- **A range hides mis-assignment.** "Owned by AG-12 … AG-16" cannot be checked against any single charter. One id makes `S-AGV`-class checks possible: *does the owner's charter actually cover this term?* — the check that would have caught Layer 1's C4-shaped mis-assignment.
- **Recurrence is not co-ownership.** A term three milestones touch (the pairing invariant: constructed by the loop, validated by AG-12's boundary, re-validated after AG-18's compaction) gets **one row, one owner, and restatement nodes listed in provenance** — never a second row. Recurrence is a fact about the plan; ownership is a fact about the vocabulary; only the second is this artifact's business. Category grouping is a third, independent axis (§ 2.3's steering row).
- **"Current" id, cheap to keep true.** Layer 1 needed a translation table because doc 0001 cited renumbered doc 0002 milestones. Layer 2's exposure is structurally smaller — doc 0003's ids are append-only from birth and its v2 restructure preserved them — but the check is kept anyway as an artifact property: every `AG-NN` in the register must resolve against a current doc 0003 heading. A cheap check against a hazard that already bit once next door is kept, not argued away.

---

## 5. The two resolved naming conflicts, as decisions

Both are pure naming questions with no design alternative, which is what makes them AG-00's to close (AG-00's out-of-scope clause cuts the other way for everything else). Both losing readings are recorded here because a silently dropped reading is how the argument gets re-had.

### Decision: `subagent` is canonical for the participant; `delegation` for the relationship

**Choice**: `subagent` names the delegated participant; `delegation` names the relationship and the event family; `child harness` (the re-entrancy mechanism) and `nested run` (the run a subagent drives) are recorded synonyms, each with its precise sense. **Scope rule**: any name that ships — an event kind, a scenario id, a test name, an acceptance criterion — uses `subagent` or `delegation`; prose may use a synonym only where the register lists it.

**The losing reading, recorded**: make `child harness` canonical, since AG-19's scenarios use it consistently and its title is "delegation readiness". It loses because the event-kind names AG-06 ships are already fixed as `subagent-started` / `subagent-ended` — the word is **on the wire**, and an envelope name is a public surface later milestones cannot cheaply rename, while prose can be aligned for free. AG-19's title survives intact because *delegation* correctly names the relationship.

**What this prevents**: AG-06 (Wave 1) and AG-19 (Wave 5) landing four waves apart with different words for one relationship — the defect class this milestone exists to prevent, live in doc 0003's own text today.

### Decision: `turn`, `provider call`, and `attempt` are three rows, not one

**Choice**: three rows with a scope column built into each definition — **turn** (one assistant response plus its tool results; harness-scoped), **provider call** (one Layer 1 stream; exactly one per loop invocation; loop-scoped), **attempt** (a provider call made in service of a turn already begun; exists beyond the first only because the harness re-invoked the loop under retry; harness-scoped) — plus the reconciling statement as a citable row: *a turn spans one or more provider calls; the count exceeds one only via harness retry; within a single loop invocation the loop never issues a second provider call.*

**The losing readings, recorded**: (a) one row, "provider call", with "attempt" as a second sense — loses because AG-16's cost scenario counts *attempts*, not calls, so the senses diverge exactly where cost aggregation needs precision; (b) treating doc 0001 § 2.3 ("one loop-to-provider interaction per iteration") and AG-15/AG-16 (retry spans several calls per turn) as a contradiction one side must win — loses because the sides speak at different scopes: side A is a statement about the loop, side B about the turn. The scope split dissolves the conflict; picking a winner would have falsified one true statement.

**What this prevents**: AG-11 (turn termination) and AG-15 (retry) each re-deriving a private answer to "how many provider calls make a turn" — the sentence currently lives only in AG-00's charter prose, citable by nobody.

### The fifth boundary case, same mechanism

*Is a compaction call a turn?* **No — a provider call, but not a turn.** It follows from the rows above: a compaction call produces a summary, not an assistant response, so it fails the turn definition while satisfying the provider-call definition; it carries its own provider, cost and cancellation (AG-18). Recorded as a register statement because AG-16's aggregation and AG-18's mechanics both depend on the answer.

---

## 6. The must-nevers as citable rows

**Choice**: the loop's six must-nevers and the harness's one become **seven individual rows** in `VL2-COR` (they constitute the responsibility split, so test 3 fires). Each row: a noun-phrase name for the obligation (for example *loop statelessness*, *no ambient authority*, *stream-only output*; the exact spellings are the apply phase's work), a definition stating the obligation observably, and — per the proposal's approach step 6 — **the mechanical guard that enforces it, named in the definition** (the AG-03.2 import guard, the AG-03.3 authority scan, AG-07.2's shared-nothing scenario, AG-10.1, AG-11.2, AG-13.1's no-privileged-channel proof).

**Alternative considered**: one composite row per part ("the loop's must-nevers", listing six). Rejected because a guard scenario cites one obligation; a composite row would make every citation ambiguous about which obligation the guard proves, and an ambiguous citation is this artifact's defined failure.

This is what checklist item 3 means by "restated as vocabulary-level obligations that AG-03's guards and later scenarios cite": the guard cites an identifier, the identifier resolves to one obligation, and the pairing of obligation and guard is readable in one row.

---

## 7. The amendment protocol, and what makes it self-enforcing

The register carries Layer 1's six standing rules with layer identifiers substituted (proposal decision 1). The design question is why rule 1 — *a missing term is appended here, never defined locally downstream* — will actually hold, in a repository where no CI can enforce it. The answer is two properties, both structural:

**The compliant path is the cheapest available path.** An amendment is: next free ordinal, one row, one dated blockquote, in the PR already open. Defining a term locally means writing the same definition anyway *plus* defending in review why it cites nothing. Layer 1 is the evidence that the cheap path gets taken: its register was amended twice inside its own Wave 0 (`V-STR-22`/`23` by AI-02.1; `V-PRV-16`/`17`/`18` by AI-03.1) — the rule held not because anyone policed it but because following it was less work than violating it. Two preconditions keep the path cheap, and both are design choices here: the register lives in `openspec/specs/` where a later PR can write (an archived register would make rule 1 unfollowable — a dead letter), and the amendment lands in the same PR that needs it, so compliance never costs a second PR.

**Violation is visible in ordinary review, without special diligence.** AG-00's acceptance criterion makes "the charter cites a vocabulary entry" the norm for every later milestone; once that is the norm, an inline definition is an anomaly a reviewer recognizes by shape — a definition with no `VL2-` citation attached — not by memory of a rule. The promoted spec carries rule 1 as a normative requirement, so every downstream SDD phase reads it as an inherited constraint, not a convention.

One rule does quiet extra work: rule 3's requirement that the amendment blockquote state **why the register lacked the term**. It converts every amendment into recorded feedback on the register's coverage — including on § 2.2's inclusion tests — which is how the scheme improves instead of merely accreting.

---

## 8. Alternatives considered and rejected (register level)

| Option | What it offers | Why it loses |
| --- | --- | --- |
| A — glossary, alphabetical, no owner column | Cheapest to write | Cannot make double-definition or drift checkable; two milestones can define one term and no column pass detects it. The owner column is the enforcement surface (§ 4.2) |
| B — per-milestone appendix | Zero coordination cost now | Is the failure mode itself, promoted to a convention: every SDD renegotiates its nouns under implementation pressure |
| D — machine-checkable schema (YAML/JSON) with generated prose | Mechanical verification | **Verified 2026-08-11**: `.github/workflows/` contains no files in this repository, so no machine would ever run the check — the same ground Layer 1 rejected it on. The audience is humans writing charters, and structured data is worse-shaped prose for them. Re-opened only if CI arrives, by whichever milestone introduces CI — not pre-decided here |
| **C — categorized register, stable ids, one owner per row** | Ownership and provenance as first-class, scannable columns | **Adopted** — Layer 1's proven shape, already validated by two in-wave amendments, with three deltas: the category scheme (§ 2), the id prefix (§ 3), and the two conflicts resolved before the register is declared closed (§ 5) |

---

## 9. File changes

| File | Action | Description |
| --- | --- | --- |
| `openspec/changes/cachicamas-agent-contract-vocabulary/design.md` | Create | This document |
| `openspec/changes/cachicamas-agent-contract-vocabulary/specs/agent-contract-vocabulary/spec.md` | Create (spec phase, parallel) | Requirements, scenarios, and the register text itself |
| `openspec/changes/cachicamas-agent-contract-vocabulary/tasks.md` | Create (tasks phase) | AG-00.1's closing checklist plus the fifth boundary case, as the task list |
| `openspec/changes/cachicamas-agent-contract-vocabulary/decision.md` | Create (apply phase) | The deliverable — AG-00.1's argument and the register's state at merge |
| `openspec/specs/agent-contract-vocabulary/spec.md` | Create at archive | The register, promoted; live and appendable thereafter |

Nothing under `backend/`, `docs/`, or any other path is touched.

## 10. Threat matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Documentation only.

## 11. Rollout and rollback

**Rollout**: the register is authored as this change's spec delta and reviewed as prose in one PR (`size:exception` pre-accepted); at archive it is promoted to `openspec/specs/agent-contract-vocabulary/spec.md` and becomes the live citable surface; `decision.md` stays in the archived change folder, immutable, as the record of how the register was first decided. The rollout test is immediate and concrete: AG-01's and AG-02's SDDs — the structural analogues of the two milestones that amended Layer 1's register in its own wave — must be expressible citing only these rows.

**Rollback**: the proposal's three levels govern. Level 1 (revert or delete the change folder) is valid only while nothing cites a `VL2-*` id. Level 2 (amend under § 7's rules) is the supported correction once any downstream charter cites a row. Level 3 (supersede wholesale via a new decision node `AG-00.2`, with the edge recorded in doc 0003) is for a wrong categorization, and leaves the original artifact in place with superseded sections struck through so merged citations keep resolving. After archive the live file is never deleted — only amended or superseded.

## 12. Verification approach

Nothing executes. Verification is inspection, and each check is cheap **by construction**, not by reviewer diligence:

| Check (spec property) | Made cheap by |
| --- | --- |
| One definition per term | Terms appear once; recurrence is a provenance cross-reference (§ 4.2), so a duplicate is a duplicate row, not a judgement call |
| One owner per term | Single-valued column; one scanning pass |
| No category ambiguity | § 2.2's ordered tests; a disputed placement is re-run through the procedure, not argued |
| No Go identifiers | Noun phrases with spaces make a code identifier visually anomalous |
| Owner ids current | Every `AG-NN` resolves against a doc 0003 heading (§ 4.2) |
| Conflicts resolved, losers recorded | § 5 states both dispositions with both sides cited; the register rows carry them |
| Must-nevers citable | Seven rows, one obligation and one guard each (§ 6) |
| Downstream expressibility | AG-01's and AG-02's charters walked noun by noun as the milestone's last task |

No `make test` gate applies: doc 0002's evidence gate (adopted by doc 0003) closes a `[decision]` leaf when the artifact answers every listed question and is merged; there is no code in `backend/agent/` for this change to exercise.

## 13. Acceptance criteria for the design phase

1. § 2's procedure assigns every term the exploration inventoried to exactly one category, and every known ambiguity is resolved by test order, not judgement.
2. § 3 defends the identifier scheme against both live alternatives and states the Layer 3 extension rule.
3. § 4 justifies every row field from a recorded failure, and the single-owner rule from Layer 1's recorded mechanism.
4. § 5 records both resolved conflicts **with their losing readings**, so neither argument can be silently re-had.
5. § 7 shows the amendment rule holds for structural reasons — cheapest path, visible violation — with Layer 1's two in-wave amendments as evidence, not assertion.
6. Nothing in this document names a Go identifier, defines a Layer 3 concept, or decides anything AG-01, AG-02 or AG-22 owns.

## 14. Next phase

`sdd-tasks` — one phase, one node (AG-00.1), the closing checklist plus the fifth boundary case as the task list, with § 12's inspection pass as verification.
