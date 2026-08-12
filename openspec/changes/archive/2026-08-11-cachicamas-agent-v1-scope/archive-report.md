# Archive Report — the Layer 2 v1 capability scope

> **Change**: `cachicamas-agent-v1-scope`
> **Milestone**: AG-02 · **Node**: AG-02.1 — The scope decision `[decision]`
> **Status**: archived
> **Archive date**: 2026-08-11 (merge date: 2026-08-11, commit 47813e6c)
> **Project**: cachicamas (witsaba)
> **Scope**: documentation only — six markdown files under one new directory, zero Go, zero `go.mod` change, zero files under `backend/`

---

## Executive Summary

The SDD change `cachicamas-agent-v1-scope` has been completed, verified with zero CRITICAL findings, and archived. The Layer 2 v1 capability scope verdicts are closed: eight register rows verdicted (implement-now, seam-with-trivial-implementation, or deferred), five non-Layer-2 rows recorded as cross-checks, both audit passes clean. The decision artifact answers both items of AG-02.1's closing checklist. The delta spec has been promoted to the canonical location `openspec/specs/agent-v1-scope/spec.md`.

---

## Artifacts Archived

All artifacts present in `openspec/changes/cachicamas-agent-v1-scope/` on merge date 2026-08-11:

| Artifact | Status | Observations | Phase |
| --- | --- | --- | --- |
| `proposal.md` | ✅ Archived | 225 lines; scope, approach, dependencies, acceptance criteria | proposal |
| `spec.md` | ✅ Promoted | 259 lines; 15 requirements, 60 scenarios for the decision artifact | spec |
| `design.md` | ✅ Archived | 291 lines; structure of `decision.md`, reasoning rules, verification approach | design |
| `tasks.md` | ✅ Archived | 162 lines; 13 phases across 42 checked tasks, no unchecked tasks | tasks |
| `decision.md` | ✅ Archived | 554 lines; fourteen verdict identifiers (8 implement-now, 3 seam, 3 deferred); two audit passes, inheritance table | apply/verify |
| `explore.md` (delta spec source) | ✅ Archived | (linked from proposal.md; predecessor artifact) | proposal |
| `specs/agent-v1-scope/spec.md` | ✅ Promoted | (copied to main specs) | spec |

---

## Verification Summary

**Verify report verdict**: `PASS WITH WARNINGS` (2026-08-11, sdd-verify phase)
- **Critical findings**: 0 (archive proceeds)
- **Closure blockers**: 0
- **Warnings**: 5 (W1–W5; precision defects in derived counts and spec wording, not verdict defects)
- **Suggestions**: 4 (S1–S4; cross-reference precision, not closure-blocking)

**Checklist verification**:
1. **Item 1** — one verdict per Layer-2-owned register row, citing the register, classed, trivial implementation named where applicable: **SATISFIED.** All eight owned rows verified against `doc 0001 § 7 G-NN`; all three `AGS-S` entries name trivial behavior and seam number.
2. **Item 2** — verdicts consistent with doc 0003's graph; mismatch fixed before closing: **SATISFIED.** Forward pass 14/14 clean against Traceability spine; reverse pass 20/20 against `grep 'Closes:'`. No disagreement between corroborating columns.

---

## Spec Promotion

The delta spec at `openspec/changes/cachicamas-agent-v1-scope/specs/agent-v1-scope/spec.md` was promoted to the canonical location `openspec/specs/agent-v1-scope/spec.md` on archive. This is a new capability — no pre-existing main spec was merged against. The 259-line spec file contains 15 requirements and 60 scenarios defining what the decision artifact must satisfy.

**Promotion method**: file copy (verbatim, no transformation)

---

## Verify-Report Verdict Details

Per `verify-report.md` (sdd-verify phase, 2026-08-11):

**Substantive findings**:
- **19 verdict identifiers** confirmed (8 `AGS-I` / 3 `AGS-S` / 3 `AGS-D` / 5 `AGS-X`; apply phase stated 20, corrected to 19)
- **ADR 0005 § D4 citations**: verified row by row; fuller phrasing "Seam now, implement in L2" cited only for G1/G3/G5 (where the source supports it); G7 and G11 correctly refuse the fuller phrasing (sources read bare "Seam now")
- **Thirteen register rows** walked in order; 8 owned + 5 cross-checked = 13; charter cross-check one-to-one
- **Both audit passes**:
  - **Forward pass**: 14 rows (one per verdict identifier); all 14 clean against Traceability spine `R-10…R-17`; all 32 cited `AG-NN.N` nodes exist in doc 0003
  - **Reverse pass**: 20 milestones carrying `Closes:` field; my own grep `grep -n 'Closes:' docs/architecture/milestones/0003-…md` returns identical set; both directions match perfectly
- **G3 split defenses** (the critical misreading protection):
  1. Two identifiers in two lists (no single "G3 verdict" to cite)
  2. `AGS-S-01` negative clause names the misreading in bold; cross-references `AGS-I-02`
  3. Forward pass lists AG-18's leaves as the required evidence for mechanism's `AGS-I` row
  4. Inheritance table states AG-18's five leaves are "obligations, not options"
- **Seam account**: 8 required seams node-for-node against R-18; 4 omissions under two distinct reasons (seams 9/10/11 are Layer 1 shipped; seam 4 is Layer 3's, stated as exception)
- **Orphan check**: all 11 `CO-*` nodes exist in doc 0004; doc 0004's own spine independently corroborates all six mappings
- **No Go identifiers**: zero camel/Pascal-case names in any of the six artifacts (positive control tested against real Go file first)
- **Closing-checklist verification**: item 1 satisfied (eight owned rows verdicted); item 2 satisfied (both audit passes clean, no disagreement)

**Warnings** (5; non-closure-blocking):
- **W1**: § 9.2 derived count (9/11 split) contradicts its own table (should be 10/10); reverse pass still passes
- **W2**: `S-AGS-014` per-row counts (2/2/3) unachievable due to `R-AGS-006`'s three `AGS-D` entries; artifact correctly resolved in favor of enumerated requirement (3/3/3); spec wording needs decision at archive
- **W3**: `S-AGS-035` premise false for AG-14/AG-21 (no `Closes:` field, so no row expected); specification imprecision, not decision defect; conclusion (not closure-blocking) upheld
- **W4**: F1 does not state opposing reading affirmatively first; does not engage strongest counter-evidence (doc 0003's own seam-2 association for G5). Disposition still safe (verdict taken from R-13's clean mapping), but rhetorical structure weaker
- **W5**: Vocabulary dependency is unmerged (AG-00 in same wave but untracked); hard delivery coupling in wave 0 design

**Suggestions** (4; informational):
- **S1**: `AGS-I-06` union description imprecise (says "3 of 3" against R-15's 4 nodes; nothing unaccounted, but wording unclear)
- **S2**: F2/F3 disposition-rule quotation compressed; meaning preserved, but italics imply verbatim
- **S3**: AI-03 precedent link points to archived change.md, not promoted canonical spec where `CAP-O-04` amendment lives
- **S4**: `S-AGS-058` "package path" clause stricter than authoring constraint it derives from (S-AGS-058 disallows paths; doc 0003 authoring constraint does not)

---

## Task Completion

All 42 implementation tasks checked complete (✅). No unchecked tasks remain.

Phases completed:
- Phase 0 (scaffold): 3 tasks ✅
- Phase 1 (total walk): 3 tasks ✅
- Phase 2 (method): 2 tasks ✅
- Phase 3 (verdicts): 6 tasks ✅
- Phase 4 (cross-checks): 1 task ✅
- Phase 5 (seams): 2 tasks ✅
- Phase 6 (findings): 4 tasks ✅
- Phase 7 (audit): 3 tasks ✅
- Phase 8 (orphan check): 1 task ✅
- Phase 9 (inheritance): 1 task ✅
- Phase 10 (amendments): 1 task ✅
- Phase 11 (verification): 3 tasks ✅
- Phase 12 (verification pass): 6 tasks ✅
- Phase 13 (final re-walk): 1 task ✅

---

## Change Closure

This change closes AG-02.1's two closing-checklist items, decided in milestone AG-02 — "Decide the Layer 2 v1 capability scope". The decision artifact is singular, complete, and merged to `main` as part of PR #159 (merge commit 47813e6c, 2026-08-11), shared with AG-00 (`cachicamas-agent-contract-vocabulary`) and AG-01 (`cachicamas-agent-event-delivery`).

Wave 0 closes: AG-00 fixes vocabulary, AG-01 fixes event delivery, AG-02 fixes v1 scope. Waves 1 through 6 proceed with verdicts fixed. Downstream milestones AG-17, AG-18, AG-19 (wave 4) and the four charter-dependents AG-09, AG-15, AG-16, AG-20 read their scope from this decision's verdict identifiers (`AGS-I/S/D-NN`).

---

## Final Authority

Archive report reflects final state per Final-State Authority hierarchy in the skill:
- Native review authority: not applicable (this is a `[decision]` leaf, no code to review)
- Persisted tasks artifact: all 42 checked complete
- Explicit final-state facts from launch prompt: merge date 2026-08-11 (commit 47813e6c), PR #159 shared with AG-00 and AG-01, 0 CRITICAL
- Verify-report snapshot: `PASS WITH WARNINGS`, 0 CRITICAL, 0 blockers, both checklist items satisfied

**Verdict remains**: PASS WITH WARNINGS, 0 CRITICAL, 0 closure-blocking. Archive proceeds.
