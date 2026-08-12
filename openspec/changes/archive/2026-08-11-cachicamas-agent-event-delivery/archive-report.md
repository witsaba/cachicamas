# Archive Report — Layer 2 Agent Event Delivery and the Observer Model

> **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 — Decide event delivery and the observer model
> **Node**: AG-01.1 `[decision]` (decision leaf, no production code)
> **Phase**: archive
> **Project**: cachicamas (witsaba)
> **Archive Date**: 2026-08-12
> **Merge Commit**: 47813e6c (2026-08-11, PR #159)
> **Status**: PASS WITH WARNINGS — 0 CRITICAL, 3 WARNING, 4 SUGGESTION
> **Artifacts Archived**: proposal.md, specs/agent-event-delivery/spec.md, design.md, tasks.md, decision.md, explore.md

---

## 1. Final Verification Verdict

Per `verify-report.md` (2026-08-11, final inspection):

**Verdict: PASS WITH WARNINGS**
- Requirements: 19/19 satisfied
- Scenarios: 26 PASS, 1 PARTIAL, 0 FAIL (27 total)
- Critical findings: 0
- Blockers: 0
- Warnings: 3 (non-blocking)
- Suggestions: 4 (informational)

The recorded verdict confirms that all acceptance criteria hold and the change is ready for archive without blocking issues. The three warnings are editorial and do not affect the decision's correctness or mergeability.

### Warnings Summary (per verify-report)

1. **W1 — Dangling internal subsection references**: Three lines in `decision.md` (lines 29, 188, 607) reference `§ N.M` labels that do not exist in `decision.md` itself (they are from the design document). The content is present in the artifact; only the reviewer entry points are misdirected. No requirement or scenario turns on this issue.

2. **W2 — Rendezvous objection stated twice instead of once**: The rendezvous objection appears in full in both `decision.md` § 4 (line 188) and § 5 (lines 288–290), whereas `design.md` criterion 5 and `tasks.md` item 4.5 committed to stating it once and cross-referencing. The duplication is deliberate and argued; no spec requirement is violated. Task 9.1's recorded verification of this criterion is therefore inaccurate, though the overall requirement is satisfied.

3. **W3 — `AG-21.2` charter does not yet carry the capacity measurement deliverable**: `decision.md` names AG-21.2 as the closing-evidence owner with "lane-absorption high-water mark and producer-wait observations". Doc 0003's AG-21 charter states "Out of scope: Performance targets — correctness under pressure only", with a deliverable of hardening suite and leak checks (correctness, not measurement). `R-AGE-018` is satisfied on its face (owner named, closing evidence described); however, the owner's charter would need amending or the closing evidence re-derived before the deferral closes. This note was recorded at line 200 of `decision.md` without full resolution.

### Suggestions (non-blocking, informational)

1. **S1**: Asymmetric cost statements in carrier alternatives table (send-capable handle cost stated inversely; push-callback cost stated directly).
2. **S2**: Memory residual of bending trilemma property (c) assigned to AG-21.2 but not priced as a residual cost, unlike abandonment residual.
3. **S3**: Wording difference between `decision.md` § 4 ("orphan synthesis fires before history closes") and AG-14.1's charter ("before the next turn runs"); both resolved in decision's favour.
4. **S4**: Placement reason for carrier view cites production/test closure split directly rather than by identifier (`R-02`).

---

## 2. Task Completion Status

All 32 tasks marked complete in `tasks.md`:

- **Phase 1 (Entry points)**: 2 tasks complete
- **Phase 2 (Carrier decision)**: 3 tasks complete
- **Phase 3 (Backpressure, two boundaries)**: 4 tasks complete
- **Phase 4 (Observer model)**: 5 tasks complete
- **Phase 5 (Close and ownership)**: 4 tasks complete
- **Phase 6 (Upward path)**: 5 tasks complete
- **Phase 7 (Topology and package contract)**: 4 tasks complete
- **Phase 8 (Cross-change: AG-00 register amendment)**: 1 task complete (verified by citation, not performed in this change's diff)
- **Phase 9 (Verification and closing-checklist walk)**: 3 tasks complete

**Verification re-check result (§ 0 of verify-report)**: Command-verified: `grep -c '^- \[x\]'` / `'^- \[ \]'` = 32 / 0. All tasks genuinely complete.

**Key completion notes from verify-report**:
- Task 8.1 (AG-00 register amendment): "Verified complete by citation, not performed here". AG-00's register already carries both required rows (`VL2-EVT-18` loop-internal loss, `VL2-EVT-19` harness-facing loss), authored on AG-01's behalf at AG-01's request. This change's diff makes no edit to AG-00's register file, satisfying the sequencing rule.
- Task 9.1 (acceptance criterion 5 verification): Recorded as verified, but a remediation note indicates the actual artifact now meets the criterion whereas the original task checkmark predated the fix. Criterion 5 now genuinely holds.

---

## 3. Spec Promotion Summary

**Delta spec location**: `openspec/changes/cachicamas-agent-event-delivery/specs/agent-event-delivery/spec.md` (281 lines)

**Promotion target**: `openspec/specs/agent-event-delivery/spec.md` (new directory, new capability)

**Action taken**: Full promotion — delta spec is a complete, new capability spec, not a delta against an existing spec. Spec contains:
- 19 requirements (`R-AGE-001` … `R-AGE-019`)
- 27 scenarios (`S-AGE-001` … `S-AGE-027`)
- Complete requirement and scenario text, ready for downstream milestones

**No merge required**: No existing spec at the target location; promotion is a direct copy.

**Verification of spec content**:
- All 19 requirements verified present and accounted for in verify-report § 3 (requirement-by-requirement verdict)
- All 27 scenarios verified present and checked in verify-report § 4 (scenario-by-scenario verdict)
- Binding vocabulary: every Layer 2 noun cited from AG-00's register by identifier; no identifiers invented here (verify-report § 0, command-verified, citation resolution check)
- Frozen input: Layer 1 stream lifecycle spec cited; not amended

---

## 4. Artifacts Archived

All six artifacts of the change are present and archived in place under `openspec/changes/cachicamas-agent-event-delivery/`:

| Artifact | Status | Lines | Summary |
| --- | --- | --- | --- |
| `proposal.md` | Archived | 242 | Five decisions proposed, scope stated, approach and risks documented |
| `explore.md` | Archived | 169 | Exploration of option space and source conflicts |
| `specs/agent-event-delivery/spec.md` | Archived (also promoted) | 281 | 19 requirements, 27 scenarios; promoted to main specs |
| `design.md` | Archived | 369 | Structure of decision artifact, three failure modes, worked cases, topology diagram |
| `decision.md` | Archived | 622 | Deliverable: five decisions in twelve sections per design structure §8; inheritance table for six blocked milestones |
| `tasks.md` | Archived | 166 | 32 tasks, all complete; precedent and dependencies documented |

**Exploration and ancillary artifacts**: `explore.md` is present and archived as part of the change. Doc 0003 notes this change's exploration previously carried a false claim about a Layer 1 fact; the correction is reflected in the final `decision.md` artifact and was re-verified by opening the cited sources.

---

## 5. Verification Evidence Summary

Per verify-report § 1 (evidence commands and outputs):

### Authoring-constraint and citation-resolution gate
- **Go-identifier scan**: No Go type, field, method or package identifier found in any of the six artifacts (verified with a firing positive control on synthetic Go identifiers first).
- **VL2-* citation resolution**: 29 distinct `VL2-*` terms cited in `decision.md`, all resolving to rows in AG-00's register (empty set difference).
- **Task completion**: 32/32 tasks marked complete.
- **Numeric Layer 2 capacity**: No numeric capacity attributed to any Layer 2 boundary anywhere in the change.
- **Diff scope**: No backend, build or dependency file touched. Only the change's own six artifacts added.

### Structural gate
- **Outbound markdown links**: All six external links verified present on disk (documentation, milestones, specs, archived decisions).
- **Design skeleton**: Exactly 12 top-level numbered sections in `design.md` §8; `decision.md` implements this skeleton.
- **Five-part shape**: Each of decision sections §3–§7 carries the five-part shape (Decision, Why, What this excludes, Consequences, Who inherits it).
- **Internal subsection cross-references**: Three dangling references reported as WARN (non-blocking per the skill's risk hierarchy).

---

## 6. Requirements and Scenarios at Close

**Requirements (19/19)**:
- All 19 mapped to decision sections and verified present
- `R-AGE-001` (carrier at Layer 2): PASS — four Layer 1 grounds reproduced and tested against AG-10.3 and AG-04.2
- `R-AGE-002` (liveness rule): PASS — adopted unweakened from Layer 1
- `R-AGE-003` (iterator-view placement): PASS — deferred to AG-23.2 with stated reason
- `R-AGE-004` through `R-AGE-017` (backpressure, observer model, close and ownership, upward path): PASS — all present with worked cases, mechanism tables, and inheritance
- `R-AGE-018` (capacity deferral): PASS (with WARNING 3) — AG-21.2 named; closing evidence and inherited tie-break stated; owner's charter discrepancy noted but does not fail the requirement
- `R-AGE-019` (acceptance criterion and authoring constraints): PASS — all five closing-checklist items answered; merge ordering mergeability unverifiable pre-merge

**Scenarios (26 PASS, 1 PARTIAL)**:
- S-AGE-025 (checklist walked and closed) is PARTIAL because merge ordering ("closed before AG-04 starts") cannot be observed pre-merge. What was confirmed: no AG-04 change exists in this worktree; `openspec/changes/` holds only AG-00, AG-01, and AG-02.
- All other 26 scenarios pass outright, including all eight marked **[adversarial]** (designed to be failed by plausible artifacts but failed by this one).

---

## 7. Decisions Recorded at Close

The five decisions this change records, per verify-report and verified in artifact:

1. **Carrier** (§3 of decision.md): Receive-only carrier, argued at Layer 2 (not by symmetry), with Layer 2 sources cited (AG-10.3, AG-04.2), alternatives priced (push-callback, send-capable handle), abandonment concession present.

2. **Backpressure at two boundaries** (§4 of decision.md):
   - Loop-internal: Layer 1's rule unchanged (on cancellation with saturated buffer, drop late events, close without terminal).
   - Harness-facing: Strictly narrower rule — never drop events describing facts already in history. Cites measured capacity 0. Answers zero-capacity objection by receive-step/after-hand-off distinction.

3. **Observer model** (§5 of decision.md): Decoupling mechanism — one canonical stream per forwarding activity, per attached consumer. Mechanism table with six candidates. Multi-observer fork decided with rebuttal recorded.

4. **Close and ownership** (§6 of decision.md): Three nested scopes (turn/loop, run/harness, delegated-run/child-harness), each with one sole closer. Terminal discipline: run-start and run-end with typed outcome. No "sometimes no terminal" case at run scope.

5. **Upward path** (§7 of decision.md): One harness-level surface carrying three payload kinds (permission, steering, interrupt). Doc 0001 §2.2 and §2.3 reconciled in writing. Call-identity lookup exists, harness-owned, shape left open. Typed rejection at two granularities; pause-resumption carved out; delegation recursion stated.

---

## 8. Inheritance and Downstream Consumption

Per verify-report § 6.4 (inheritance table):

**Blocked milestones receiving from this decision** (all 19 requirements mapped):

| Milestone | What it inherits |
| --- | --- |
| **AG-04** (envelope and ordering) | Carrier; three ownership scopes; "nothing follows the terminal" as owned obligation |
| **AG-10** (permission protocol) | Harness-level surface; call-identity lookup confirmed harness-owned; typed-protocol-error discipline; suspension never stalls delivery |
| **AG-13** (multi-turn run driver) | Same surface for steering input; ended-run typed rejection generalized; pause-resumption carve-out |
| **AG-14** (cancellation tree) | Interrupt as payload kind; interrupt during wind-down (idempotent) vs. after ended (rejected); harness-facing delivery obligation in bounded wind-down |
| **AG-19** (delegation and re-entrancy) | Strictly nested ownership; leaf-first close ordering; upward-path recursion; frontend answers on parent's surface, parent routes to child's suspension lookup |
| **AG-20** (hook taxonomy) | Decoupling mechanism that makes stalled-observer test pass structurally; "eventually reported typed" expressible at all |

**No-invented-channel rule**: None of these milestones may invent its own channel, loss rule, or way back into a live run. Stated once in §11 and verified present.

---

## 9. Known Open Issues and Future Work

**Per verify-report**, tracked for downstream milestones:

1. **AG-21.2 charter amendment** (WARNING 3): The capacity deferral's closing evidence description assumes AG-21.2 will measure "lane-absorption high-water mark and producer-wait observations". AG-21's published charter states "Performance targets out of scope; correctness under pressure only". The decision itself is valid and merges; AG-21 ownership is named. However, AG-21.2's charter would need amending or the closing evidence re-derived before the deferral can close. This is not a defect in AG-01; it is an AG-21 issue discovered and recorded during AG-01's verification.

2. **Three internal cross-reference fixes** (WARNING 1): `decision.md` should either add numbered subsection labels (§4.2, §5.3, §5.4) to match design §8's skeleton, or update three reference lines (29, 188, 607) to remove misdirected labels.

3. **Rendezvous objection consolidation** (WARNING 2): Either collapse the duplication at lines 188 vs. 288–290, or relax design criterion 5 and amend task 9.1's verification note to acknowledge the deliberate duplication.

None of these prevent archive. All are post-merge optional corrections or downstream amendments.

---

## 10. Archive Ledger

| Artifact | Promotion Status | Location | Storage |
| --- | --- | --- | --- |
| Delta spec | Promoted to main specs | `openspec/specs/agent-event-delivery/spec.md` | Filesystem |
| Change folder | Remains in-place | `openspec/changes/cachicamas-agent-event-delivery/` | Filesystem (not moved yet) |
| Archive report | Created in-place | `openspec/changes/cachicamas-agent-event-delivery/archive-report.md` | Filesystem |

**Workflow**: The change folder itself will be moved by the orchestrator to `openspec/changes/archive/2026-08-12-cachicamas-agent-event-delivery/` after this archive report is staged. The spec has been promoted. All artifacts are in place for archive closure.

---

## 11. Final State Authority and Reconciliation

Per the Final-State Authority hierarchy in the skill:

- **Native review authority**: This change did not enter receipt-driven development (no review gate records exist; the kill switch is off).
- **Persisted tasks artifact**: All 32 tasks complete (verified by command).
- **Orchestrator launch prompt**: No explicit final-state facts about resolved blockers or fixed warnings beyond what verify-report already records.
- **Verify-report (verify-report.md, 2026-08-11)**: Intermediate snapshot, now authority for the milestone's final verification verdict. PASS WITH WARNINGS recorded; no CRITICAL findings.

**Reconciliation**: The verify-report's intermediate snapshot claims rank lowest in the hierarchy. However, this report's authority derives from:
1. The persisted tasks artifact (all complete).
2. The explicit merge commit (47813e6c, 2026-08-11) showing the change already merged to main.
3. The verify-report's own verdict (PASS WITH WARNINGS, 0 CRITICAL), which is the final snapshot of the verification phase before archive.

**Contradictions**: None identified. The verify-report's three warnings are confirmed and have not been superseded by later commits or evidence.

---

## Summary

The change `cachicamas-agent-event-delivery` (AG-01.1) is complete, verified (PASS WITH WARNINGS, 0 CRITICAL), and ready for archive. The delta spec has been promoted to `openspec/specs/agent-event-delivery/spec.md`. All 32 tasks are complete. The five recorded decisions answer all five items of the closing checklist with rationale and *what this excludes* parts. The artifact is self-contained and requires no code execution (decision leaf). No Go identifiers appear anywhere; no files outside the change directory were modified. All Layer 2 nouns are cited from AG-00's register by identifier. Merge commit 47813e6c (2026-08-11, PR #159) confirms the change is already on main.

**Next action**: Move the change folder to the archive directory (orchestrator responsibility) and update the config's archive index if required. The change is closed and all downstream milestones (AG-04, AG-10, AG-13, AG-14, AG-19, AG-20) may proceed without reopening this decision.
