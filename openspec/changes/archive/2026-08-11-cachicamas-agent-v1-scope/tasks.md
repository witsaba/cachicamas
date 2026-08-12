# Tasks — the Layer 2 v1 scope verdicts

> **Change**: `cachicamas-agent-v1-scope`
> **Milestone**: AG-02 · **Node**: AG-02.1 — The scope decision `[decision]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Inputs**: `explore.md`, `proposal.md`, `specs/agent-v1-scope/spec.md`, `design.md`
> **Deliverable**: `openspec/changes/cachicamas-agent-v1-scope/decision.md`
> **Precedent**: [AI-03's tasks](../archive/2026-07-31-cachicamas-ai-minimum-capabilities/tasks.md)

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 900–1,400 (decision.md alone; shared PR also carries AG-00 and AG-01's own artifacts) |
| 400-line budget risk | High |
| Chained PRs recommended | No — `size:exception` is the chosen route, not chaining |
| Suggested split | Single PR, shared with AG-00 and AG-01 |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No — `size:exception` already accepted by the user in session preflight, against the 1,000-line shared-PR budget the proposal records.
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Splitting `decision.md`'s sixteen sections across chained PRs is not viable: the four lists, the two audit passes and the inheritance table are one classification process over one closed set of rows — a chain would leave a register row verdicted in one PR and audited in another, reopening exactly the re-litigation surface AG-02 exists to close. `size:exception` is the correct route for a generated-decision artifact of this kind, per the skill's own guidance.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | `decision.md`, all sixteen sections, complete | PR 1 (shared with AG-00, AG-01) | N/A — `[decision]` leaf, no test gate; verification is inspection against the checklist in Phase 12 below | N/A — no runtime harness exists for a documentation-only node | `git revert` of the change's commits, or delete `openspec/changes/cachicamas-agent-v1-scope/` — purely additive, no partial-rollback hazard per `proposal.md` |

---

## Traceability — every requirement maps to a task

| Requirement | Task(s) |
|---|---|
| R-AGS-001 | 0.1, 0.2, 11.1, 11.2 |
| R-AGS-002 | 1.1, 1.2, 1.3 |
| R-AGS-003 | 3.1, 3.2 |
| R-AGS-004 | 2.2, 3.3, 3.4, 3.5 |
| R-AGS-005 | 4.1 |
| R-AGS-006 | 3.6 |
| R-AGS-007 | 5.1, 5.2 |
| R-AGS-008 | 7.1 |
| R-AGS-009 | 7.2 |
| R-AGS-010 | 7.3 |
| R-AGS-011 | 6.1, 6.2, 6.3, 6.4 |
| R-AGS-012 | 8.1 |
| R-AGS-013 | 9.1 |
| R-AGS-014 | 10.1 |
| R-AGS-015 | 0.3, 11.3, 12.6 |

---

## Phase 0: Scaffold and namespace statement (design § 5, § 1)

- [x] 0.1 Create `decision.md` with the sixteen-section spine (§ 1–§ 13 of `design.md` § 5), each section carrying a stated placement reason where `design.md` § 5 gives one.
- [x] 0.2 Write § 1 "How to use this document": the seven consumers, and the spec's two disjoint identifier namespaces (`R-AGS-0NN`/`S-AGS-0NN` vs. `AGS-I/S/D/X-NN`), restated so a downstream citation of `AGS-I-01` is never read as citing a requirement.
- [x] 0.3 State in § 1 that AG-02.1 is a `[decision]` leaf: no production code, closes on merge, no `make test` gate.

## Phase 1: The total walk and the four lists, before any argument (design § 5 § 2)

- [x] 1.1 Write § 2 as a walk of all thirteen register rows G1…G13, register order, each with a recorded outcome — no subset.
- [x] 1.2 State the ownership test used to sort the walk (Owner-column rule, re-appliable to a new row) and the resulting counts: eight Layer-2-owned rows (G1, G3, G4's L2 half, G5, G7, G8's L2 half, G10's L2 half, G11) and five non-Layer-2 rows (G2, G6, G9, G12, G13) — named individually, summing to thirteen.
- [x] 1.3 Cross-check the eight owned rows against AG-02's charter defaults in doc 0003, one to one, before drafting any entry.

## Phase 2: Method sections — the ownership test and the taxonomy (design § 3, § 4, § 8 rules 1–2)

- [x] 2.1 Write § 3, the ownership test, as a restatable rule (design § 8 rule 1).
- [x] 2.2 Write § 4: the four inclusion tests (`AGS-I`/`AGS-S`/`AGS-D`/`AGS-X`, design § 3 table) and the splitting rule stated once, before the lists: a verdict attaches to a concern half, not a row; every half is named, never resolved by picking one.

## Phase 3: The verdict entries — closing-checklist item 1 (design § 5 § 5)

- [x] 3.1 Draft every `AGS-I`/`AGS-S`/`AGS-D` entry with the six-part shape (design § 5.1): identifier, register-row citation, class, obliges, does-not-oblige, discharging node(s).
- [x] 3.2 For each `AGS-S` entry, add the two additional parts: the trivial implementation named as concrete behavior, and the seam number from doc 0001 § 6.
- [x] 3.3 Draft G8's two-halved split (retry `AGS-I` / failover `AGS-S`) with the full argument aired before rebuttal (design § 4): retry is knowable from the typed error alone; failover reopens token budgets, the price table and the cache prefix.
- [x] 3.4 Draft G3's two-halved split with all four of design § 2.1's structural defenses present and cross-consistent: (a) two identifiers in two lists, no single "G3 verdict" citable; (b) the `AGS-S` entry's negative clause names the misreading in its own text and cross-references the sibling `AGS-I` entry; (c) confirm the forward-pass row (Phase 7) lists AG-18's leaves as required evidence, not discretionary work; (d) confirm the inheritance row (Phase 9) states the leaves are not optional.
- [x] 3.5 Draft G7's three-way split (structural property `AGS-I` / delegation seam `AGS-S` / production tool `AGS-D`) with the sharpest negative clause in the artifact: proving re-entrancy obliges **no** shipped subagent tool, citing v2 § 8's non-goal. Verify G7's own § D4 row before drafting — it reads bare "Seam now" in ADR 0005 § D4, so this entry cites no fuller phrasing for G7 (it is not an F3 row; do not borrow F3's citation here).
- [x] 3.6 Draft the three `AGS-D` entries restating doc 0003's own "Explicitly deferred" rows cited to AG-02 (subagent tool → AG-19; failover → AG-15.3; compaction quality → AG-18.1), each traceable to its deferred-table row.

## Phase 4: The cross-check entries (design § 5 § 6)

- [x] 4.1 Draft the five `AGS-X` entries (G2, G6, G9, G12, G13), each naming its actual owner and owning node — at minimum G2→CO-04.1, G6→CO-02.1, G9→AI-12, G12→AI-07/AI-13/AI-18, G13→AI-02 — and state that these are cross-checks, not verdicts, including G13's footnote that AG-01's "the G13 of this layer" self-description is an analogy, not a discharge.

## Phase 5: The seam account (design § 5 § 7)

- [x] 5.1 Map R-18's eight required seams (1, 2, 3, 5, 6, 7, 8, 12) to their doc 0003 nodes, verified node-for-node against the Traceability-spine row for R-18.
- [x] 5.2 Record the four omitted seams (4, 9, 10, 11) under **two distinct reasons**, not one: seams 9, 10, 11 as Layer 1 contract items already shipped (AI-12, AI-10/AI-11, AI-07); seam 4 as Layer 3's (G6's owner), stated as the exception to doc 0001 § 6's Layer-1-urgency grouping. Confirm the count: eight required plus four omitted equals twelve.

## Phase 6: Findings F1–F4 (design § 5 § 8, § 7)

- [x] 6.1 Draft F1 as the four-step argument from `design.md` § 7: the defect is real (both line references cited); the repair is architectural, not typographical, and out of this milestone's authority; G5's verdict does not depend on the repair (taken from R-13's clean mapping, states it does not reproduce doc 0001's seam cell); the follow-up route is named (a doc 0001 amendment in its own change).
- [x] 6.2 Draft F2: the literal "no v1 work" reading stated first, then rebutted by citing doc 0001's own disambiguating paragraph; G10's Layer 2 half verdicted `AGS-I`, token-only.
- [x] 6.3 Draft F3, one rebuttal covering G1, G3, G5 and G11, each referencing rather than restating it. Before writing, verify each citation against ADR 0005 § D4's actual table text — do not copy the exploration's phrasing: the fuller phrasing "Seam now, implement in L2" is present in § D4's cells for **G1, G3 and G5 only**, and may be cited for those three. **G11's § D4 cell reads bare "Seam now"** — its rebuttal rests on AG-02's charter default (taxonomy complete, pre-request and post-turn live, pre-compact live with AG-18, session-start emitted) plus the realized milestones AG-08 and AG-20, never on § D4's fuller phrasing. A citation the source does not contain would satisfy `S-AGS-044` falsely — reject the entry and redraft if this check fails.
- [x] 6.4 Draft F4 as a footnote: recorded as an analogy, not a discharge, naming AI-02 (not AG-01) as G13's deciding node, consistent with 4.1.

## Phase 7: The graph-consistency audit — closing-checklist item 2 (design § 6, § 5 § 9)

- [x] 7.1 Build the forward pass: one row per `AGS-I`/`AGS-S`/`AGS-D` identifier, with class, required evidence kind, the verdict's own declared node identifiers, and — in a separate column — the doc 0003 Traceability-spine R-row stating the same mapping independently. Execute the reviewer's cross-reference for every row before recording status: open doc 0003's "Traceability spine" section at the cited R-row and compare node identifiers with the verdict's column.
- [x] 7.2 Build the reverse pass: one row per doc 0003 milestone carrying a `Closes:` field. Run `grep -n 'Closes:' docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md`, compare the result set to the table both ways (every returned milestone in the table; every table row corresponds to a returned line), and record the milestone, what its `Closes:` field names, and either the matched register row or the base-architecture requirement identifier (R-01…R-09, R-19…R-21) it closes instead.
- [x] 7.3 State the disposition rule and derive the outcome from the tables, after them: a disagreement between the forward pass's two columns, or a reverse-pass mismatch, **is** the defect closing-checklist item 2 requires fixed before closure — not a note, not a risk carried forward. If Phase 7.1 or 7.2 finds a disagreement, repair it (in `decision.md` if the transcription is wrong, or flag doc 0003's own living-graph clause if the spine disagrees with doc 0003's own milestones) before proceeding to Phase 11.

## Phase 8: The Layer 3 orphan check (design § 5 § 10)

- [x] 8.1 Build the orphan-check table: every Layer 3 assignment this decision makes (via `AGS-X`, a split row's Layer 3 half, or an `AGS-D` owner) with its doc 0004 node — at minimum G1's policy half→CO-03.1/CO-03.2/CO-16.1, G2→CO-04.1, G6→CO-02.1, G10's pricing half→CO-05.1/CO-18.1, G11's concrete hooks→CO-24.1/CO-24.2, G4's payoff→CO-24.1. Open `docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md` and confirm each node exists.

## Phase 9: What each blocked milestone inherits (design § 5 § 11, § 9)

- [x] 9.1 Draft the seven-row inheritance table: three full rows (AG-17, AG-18, AG-19, each in that milestone's own terms) plus four pointer rows (AG-09, AG-15, AG-16, AG-20, each naming at least the governing verdict identifier). Confirm AG-18's row states its five leaves are not optional, consistent with 3.4(d).

## Phase 10: Standing amendment rules (design § 5 § 12, § 10)

- [x] 10.1 Draft the amendment rules: a later Layer-2-owned concern arrives by dated amendment to `decision.md`, never by a local downstream verdict; identifiers append-only; no renumbering; superseded text struck through; every count updated — citing AI-03 § 13 and its exercised 2026-08-10 amendment as precedent. State the second route separately: a verdict disproven by implementation follows doc 0003's living-graph clause, and the verdict revision and the doc 0003 graph change it implies travel in the same PR, with the affected forward- and reverse-pass rows re-derived in that same amendment.

## Phase 11: Closing-checklist verification and hygiene (design § 5 § 13)

- [x] 11.1 Draft the closing verification table mapping AG-02.1's two checklist items to the sections that answer them (item 1 → § 5; item 2 → § 9), each with a status.
- [x] 11.2 Confirm no other artifact of the change (`explore.md`, `proposal.md`, `spec.md`, `design.md`) restates a verdict as normative; each refers to `decision.md` instead.
- [x] 11.3 Run the hygiene checks: no type name, field name, method name or package identifier anywhere in the change; every Layer 2 concern name resolves to AG-00's vocabulary by citation; no sentence fails the deletion test for Layer 3 scope (design § 8 rule 4); the diff contains only markdown under `openspec/changes/cachicamas-agent-v1-scope/`, nothing under `backend/`, no edited merged document.

## Phase 12: Verification pass (closes the milestone — design § 14, ranked by cost of a missed defect)

Every check is inspection; nothing executes.

- [x] 12.1 G3's split: two identifiers, the misreading named, AG-18's inheritance row consistent (R-AGS-004, S-AGS-051) — highest cost if missed.
- [x] 12.2 The thirteen-row walk with counts summing to thirteen (R-AGS-002).
- [x] 12.3 F3 rebutted once from the controlling source, referenced four times, G11 cited precisely against § D4's actual text — not the exploration's phrasing (R-AGS-011).
- [x] 12.4 Both audit passes present with their exact reproduction procedures and disposition rule (R-AGS-008/009/010).
- [x] 12.5 Every entry's negative clause present; G7's names "no shipped subagent tool" (R-AGS-003).
- [x] 12.6 Seam account total (8 + 4 = 12), two omission reasons, seam 4 the stated exception (R-AGS-007).
- [x] 12.7 F1 recorded with both line references, no seam-cell reproduction, follow-up route named (S-AGS-042, S-AGS-046).
- [x] 12.8 Orphan-check table complete against doc 0004 (R-AGS-012).
- [x] 12.9 Inheritance: three full rows + four pointers (R-AGS-013).
- [x] 12.10 Amendment rules + living-graph binding stated (R-AGS-014).
- [x] 12.11 Hygiene: no Go identifiers, no `backend/` files, no merged-document edits, namespaces distinct (R-AGS-015).

## Phase 13: Final closing-checklist re-walk (mandatory, closes the milestone)

- [x] 13.1 Independently re-walk AG-02.1's two closing-checklist items against the merged `decision.md` and record the evidence for each: item 1 — every Layer-2-owned row has at least one verdict, cited, classed, with `AGS-S` entries naming trivial implementation and seam; item 2 — both audit passes are clean, derived from the tables rather than asserted. Explicitly record that a disagreement between the forward pass's two corroborating columns (or a reverse-pass mismatch) found at this step is a **closure-blocking defect**, not a note to carry forward — if found, return to Phase 7 and repair before this task is checked complete.

---

## Acceptance criteria for the milestone

1. All sixteen sections of `design.md` § 5 are present in `decision.md`, each with its stated placement reason where one is given.
2. Every `R-AGS-001`…`R-AGS-015` requirement holds, per the traceability table above.
3. The verification pass (Phase 12) and the final re-walk (Phase 13) are both recorded complete.
4. The change adds one directory and (across AG-00/AG-01/AG-02 combined) six markdown files under `openspec/changes/cachicamas-agent-v1-scope/`; it edits zero existing files.
5. No Go identifier, no file under `backend/`, no build/module/infrastructure edit.

## Next

`decision.md` — the deliverable. On merge, wave 0 closes: AG-00 fixes vocabulary, AG-01 fixes event delivery, AG-02 fixes v1 scope. AG-17, AG-18 and AG-19 (wave 4) and the four charter-dependents (AG-09, AG-15, AG-16, AG-20) cite this artifact's verdict identifiers from their own SDD changes onward.

---

**Note on length**: the `sdd-tasks` skill sets a 530-word budget. This tasks artifact exceeds it deliberately, following this repository's merged precedent (AI-03's tasks, PR #95, the Layer 1 analogue of comparable depth) and the orchestrator's explicit brief for this phase: sixteen sections each tasked individually, a full requirement-traceability table, the thirteen-row walk, both audit passes with their exact reproduction procedures, the F1–F4 rebuttals with per-citation verification against ADR 0005 § D4, and a final closing-checklist re-walk — none of which compresses to 530 words without dropping content the milestone's acceptance criterion requires.
