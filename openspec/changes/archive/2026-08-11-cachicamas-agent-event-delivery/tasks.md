# Tasks — Layer 2 agent event delivery and the observer model

> **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 · **Node**: AG-01.1 — Event delivery and the observer model `[decision]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Inputs**: `explore.md`, `proposal.md`, `specs/agent-event-delivery/spec.md` (`R-AGE-001`…`R-AGE-019`, `S-AGE-001`…`S-AGE-027`), `design.md`
> **Precedent**: `openspec/changes/archive/2026-07-31-cachicamas-ai-stream-lifecycle/tasks.md` (AI-02.1) — same node type, same discipline
> **Depends on**: AG-00 (`cachicamas-agent-contract-vocabulary`) — decided in parallel, same wave, same pull request
> **Blocks**: AG-02, AG-04 — and through them AG-10, AG-13, AG-14, AG-19, AG-20

---

## Node type and what it means for this task list

AG-01.1 is a **`[decision]` leaf**. No production code, no test list, no red-green-refactor cycle, no `make test` evidence gate — a decision leaf closes when the decision artifact answers every closing-checklist item and merges. `openspec/config.yaml` sets `apply.tdd: true` for **Go service code**; this change writes none. The whole milestone is one phase, one node: the PR-chain question degenerates to a single-PR decision, made explicit in the forecast below.

**Deliverable of the whole phase:** `openspec/changes/cachicamas-agent-event-delivery/decision.md`, structured per `design.md` §8's twelve sections, plus one amendment to AG-00's register in the same pull request.

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,600–2,200 (decision.md alone, given five worked-trace decisions, three diagrams, six tables) + ~40–60 (AG-00 register amendment) + this file. **0 Go** |
| 400-line budget risk | **High** — but the 400-line code-review budget targets authored risk in executable code; this diff is entirely markdown decision prose under `openspec/changes/`, per the AI-02 precedent's own accounting |
| Chained PRs recommended | **No** |
| Suggested split | Single PR |
| Delivery strategy | `single-pr` |
| Chain strategy | `size-exception` |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

**Why no chaining, even at High risk**: `design.md` §12 states the five decisions are load-bearing on each other — the carrier gives the send discipline its object, the observer mechanism presumes the carrier, the harness-facing rule presumes the harness owns the run scope, the upward path presumes run-end delivery. Splitting a coupled decision across pull requests reproduces the partial-decision defect the precedent change's `explore.md` names as the reason the milestone exists as one node. `size:exception` was already accepted by the user for this session; it is recorded here as accepted, not requested.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | `decision.md` (all twelve sections) + AG-00 register amendment, merged together | PR 1 (only PR) | Inspection walk against V-1…V-8 below; no executable test exists for a decision leaf | N/A — no runtime, no build target touches this change; verification is inspection-only by construction (`design.md` §13) | `git revert` of the single commit; complete by construction (new directory, nothing imports it) — but AG-00's amendment reverts together with it or not at all, per `design.md` §12 |

---

## Traceability — every requirement maps to at least one task

| Requirement | Task(s) |
| --- | --- |
| `R-AGE-001` | 2.1 |
| `R-AGE-002` | 2.2, 7.2 |
| `R-AGE-003` | 2.3 |
| `R-AGE-004` | 3.1 |
| `R-AGE-005` | 3.2, 8.1 |
| `R-AGE-006` | 3.3 |
| `R-AGE-007` | 3.4, 7.2 |
| `R-AGE-008` | 4.1, 4.2, 4.3 |
| `R-AGE-009` | 4.4 |
| `R-AGE-010` | 5.1, 5.2 |
| `R-AGE-011` | 5.3 |
| `R-AGE-012` | 5.4 |
| `R-AGE-013` | 6.1 |
| `R-AGE-014` | 6.2 |
| `R-AGE-015` | 6.3 |
| `R-AGE-016` | 6.4 |
| `R-AGE-017` | 6.5, 7.3 |
| `R-AGE-018` | 3.5 |
| `R-AGE-019` | 9.1, 9.2, 9.3 |

---

## Phase 1 — `decision.md` §1–§2: entry points and the settled verdict

- [x] 1.1 Write §1 "How to use this document" — one entry point per blocked milestone (AG-04, AG-10, AG-13, AG-14, AG-19, AG-20), each answerable without reading the other four decisions.
- [x] 1.2 Write §2 "What was decided" — the five one-line conclusions, stated before any argument, in checklist order.

## Phase 2 — Decision 1: the carrier (§3) — `R-AGE-001`…`R-AGE-003`

- [x] 2.1 Draft the carrier decision in the five-part shape (Decision · Why · What this excludes · Consequences · Who inherits it): reproduce Layer 1's four carrier grounds, argue each against a named Layer 2 source (AG-10.3 for the multiplexing ground, AG-04.2 for the terminal-event ground), price the two alternatives (push-callback, send-capable sink handle), and state the abandonment concession as a residual cost. **[R-AGE-001; S-AGE-001, S-AGE-002]**
- [x] 2.2 State the caller-owned liveness rule unweakened: cancellable signal on the creating call, every send waits on destination and cancellation including the terminal send, the two legal endings (drain-to-close, cancel), bounded-time close with signal-driven backoff. **[R-AGE-002; S-AGE-003]**
- [x] 2.3 **Choice — carrier-view convenience owner.** State that any carrier-view convenience lives outside the Layer 2 runtime package, in a test-support sibling; never owns or closes the carrier it views; is never a second contract; and name **AG-23.2** (test kit + examples, wave 6) as the milestone that owns whether such a view is ever built at all. **[R-AGE-003; S-AGE-004]**

## Phase 3 — Decision 2: backpressure, two boundaries (§4) — `R-AGE-004`…`R-AGE-007`, `R-AGE-018`

- [x] 3.1 State the losslessness claim with named exceptions only: every message and tool event arrives, in order, except at the two boundary rules below — no third loss circumstance anywhere in the artifact. **[R-AGE-004; S-AGE-005]**
- [x] 3.2 State the loop-internal boundary rule verbatim from Layer 1 (cancellation with a saturated buffer drops late events, closes without a terminal), argue why inheritance is safe (the loss fires only on the harness's own cancellation), and record the hazard that the two boundaries now carry different loss postures — hand the naming to Phase 8's cross-change task. **[R-AGE-005; S-AGE-006]**
- [x] 3.3 State the harness-facing boundary rule strictly narrower than Layer 1's, carrying `design.md` §4.2's worked case (a committed tool side effect silently absent from the stream while history holds it) and §4.3's composition table; cite the **measured** capacity 0 and its consequence (saturation is the ordinary condition, not the exceptional one) rather than the superseded hypothesis; answer the "consumer pauses by design" objection by the receive-step-versus-after-hand-off distinction. **[R-AGE-006; S-AGE-007, S-AGE-008]**
- [x] 3.4 State the droppable set positively (facts the harness never learned before cancellation) and why a fully lossless harness-facing boundary is unavailable at any price (an unbounded wait would contradict AG-14.3's bounded wind-down). **[R-AGE-007; S-AGE-009]**
- [x] 3.5 **Choice — capacity-measurement owner.** Defer every numeric boundary capacity explicitly; name **AG-21.2** (slow-consumer pressure) as the owner, with closing evidence = lane-absorption high-water mark and producer-wait observations under AG-21.2's stalled-consumer scenarios, Layer 1's smaller-wins tie-break inherited; state why no starting hypothesis is named, unlike AI-02. **[R-AGE-018; S-AGE-024]**

  **Remediation note (verify pass, 2026-08-11).** The closing evidence named here — "lane-absorption high-water mark and producer-wait observations" plus a smaller-wins tie-break — read as a performance measurement, which AG-21's charter explicitly excludes ("Out of scope: Performance targets — correctness under pressure only"). `decision.md` §4's "Choice — the capacity-measurement owner" was corrected to name AG-21.2's own two acceptance scenarios (a stalled consumer loses nothing; cancellation unblocks within bounds) as the closing evidence instead — a correctness gate consistent with AG-21.2's actual charter, not a number. AG-21.2 remains the owner; only what closes the deferral changed, and a failure of either scenario now routes to a dated §11 rule 4 amendment rather than to a follow-up measurement task.

## Phase 4 — Decision 3: the observer model (§5) — `R-AGE-008`, `R-AGE-009`

- [x] 4.1 Draw the convention-based failure trace: one stalled observer's inline fan-out backing up through the harness's receive into the loop's send into the Layer 1 transport read, freezing delivery for every consumer including the primary frontend — to motivate why a documented "must not block" rule is insufficient. **[supports R-AGE-008]**
- [x] 4.2 **Load-bearing task.** Write the run-extent-bounded lane absorption resolution (`design.md` §5.2–§5.3): one canonical internal stream whose only receiver is the distribution step; per attached consumer, a lane fed by its own forwarding activity that privately owns that consumer's receive-only carrier and applies the full send discipline; each lane's absorption bounded by the run's own extent (memory-only, freed at run end, no persistence per R-02); the terminating trace proving every path from a stalled consumer's receive ends at its own forwarding activity and never reaches the canonical producer or any other consumer. **This is the task without which `S-AGE-010`'s stalled-observer trace has no defence in the merged artifact.** **[R-AGE-008; S-AGE-010]**
- [x] 4.3 Tabulate every rejected mechanism by what it makes **impossible**, not what it discourages: bounded per-consumer buffer with drop-on-overflow (rejected against `R-AGE-004`), pull-based replay record, and the blocking synchronous multicast row recording that it makes **nothing** impossible — the named example of "conventional, not structural". **[R-AGE-008; S-AGE-011]**
- [x] 4.4 Decide the multi-observer-versus-single-hand-off fork on the page: verdict (more than one attached consumer, no consumer privileged at the mechanism level), the single-hand-off reading written down with its rejection grounds, Layer 3's further fan-out stated as a second additional stage rather than an alternative. **[R-AGE-009; S-AGE-012]**
- [x] 4.5 Answer the rendezvous objection on its merits, once, cross-referenced rather than repeated: a rendezvous is intolerant of a pause on the receive step and indifferent to a pause on work performed after hand-off; anchor the argument in AG-10.3's off-the-receive-path requirement. **[supports R-AGE-006, R-AGE-008]**

## Phase 5 — Decision 4: close and ownership (§6) — `R-AGE-010`…`R-AGE-012`

- [x] 5.1 State the three nested scopes with their sole owner and sole closer (loop / turn, harness / run, child harness / delegated run), and why the turn-scope and run-scope closers differ (the loop is stateless, re-instantiated per turn, and does not know the run boundary). **[R-AGE-010; S-AGE-013]**
- [x] 5.2 State the delegated-run case: leaf-first close order, the child's stream fully closed before the parent's re-emission, neither party ever closing the other's stream. **[R-AGE-010; S-AGE-014]**
- [x] 5.3 State exactly-one-terminal discipline: one run-start, one run-end with typed outcome (completed / interrupted / failed), turn-end on every exit path (normal, typed failure, cancellation) distinguishing finished from aborted, and that Layer 2 has **no** "sometimes no terminal at all" case at run scope — citing `R-AGE-006` as what makes that true. **[R-AGE-011; S-AGE-015]**
- [x] 5.4 State the per-outcome consumer assumption: on completed, the received story is complete; on interrupted or failed, the received prefix is trustworthy in §4.3's specific sense (nothing committed to history is missing; what had not yet happened is truncated). **[R-AGE-012; S-AGE-016]**

## Phase 6 — Decision 5: the upward path (§7) — `R-AGE-013`…`R-AGE-017`

- [x] 6.1 Name the harness as the one inbound surface for three typed payload kinds (permission decision, steering input, interrupt); reconcile doc 0001 §2.2 (harness-level arrow) with §2.3 (loop-level suspension narration) in writing, stating why the harness is the stable addressable surface and the loop is the routing destination. **[R-AGE-013; S-AGE-017]**
- [x] 6.2 State that the call-identity-to-suspension lookup exists, is harness-owned, and lives for the suspension's lifetime — without fixing its shape, structure or storage. **[R-AGE-014; S-AGE-018]**
- [x] 6.3 Require typed rejection, never a silent drop, at both granularities (call identity within a live run; run identity once fully ended); state the two carve-outs — an interrupt during bounded wind-down is silently idempotent, and pause-resumption is model-initiated, harness-internal, and not an instance of the upward path at all. **[R-AGE-015; S-AGE-019, S-AGE-020, S-AGE-021]**
- [x] 6.4 State the upward-path recursion under delegation: the frontend answers through the parent's surface; the parent's routing must reach the nested child's own suspension lookup. **[R-AGE-016; S-AGE-022]**
- [x] 6.5 Close §7 with the inheritance table naming AG-04, AG-10, AG-13, AG-14, AG-19, AG-20 in each one's own terms, and the no-invented-channel rule stated once for all of them. **[R-AGE-017; S-AGE-023]**

## Phase 7 — `decision.md` §8–§11: topology, package contract, inheritance, standing rules

- [x] 7.1 Reproduce `design.md` §3's delivery topology as one settled diagram, roles and messages only, plus the invariant-binding table. **[supports R-AGE-008, R-AGE-010]**
- [x] 7.2 State the package contract's abandonment clause: a consumer that abandons without cancelling is a documented, untestable-to-termination contract violation — the Layer 2 equivalent of AI-40.3's Layer 1 clause. **[R-AGE-002, R-AGE-007]**
- [x] 7.3 Consolidate §10's single inheritance table across all five decisions so it is the one place a reviewer checks each blocked milestone's inheritance — verify it does not silently diverge from the per-decision inheritance stated in Phases 2–6. **[R-AGE-017]**
- [x] 7.4 State §11 standing rules: amendment terms for this decision, and the no-invented-channel rule restated once at the document level.

## Phase 8 — Cross-change: AG-00 register amendment (sequence after AG-00's register exists)

- [x] 8.1 **Choice — the two loss-posture rows.** **Sequencing**: start only once `openspec/changes/cachicamas-agent-contract-vocabulary/specs/agent-contract-vocabulary/spec.md`'s register section exists (AG-00.1's own `sdd-apply` output), since this task appends to that file, not to this change's own artifacts. Under `R-AGV-013`'s amendment procedure, append two rows at the next free ordinals in their category, each introduced by a dated blockquote naming what was appended, by AG-01.1, and why the register lacked the term: **(a)** the loop-internal loss posture — Layer 1's sanctioned loss, unchanged, at turn scope; **(b)** the harness-facing loss posture — history-guarded truncation. Update the touched category's count and the register's stated sum. `decision.md` §4 cites both rows by identifier; it does not define them. **[R-AGE-005; S-AGE-006 — also closes AG-00's `R-AGV-013`, `S-AGV-037`, `S-AGV-040` for these two rows]**

  **Verified complete by citation, not performed here.** AG-00's register (`openspec/changes/cachicamas-agent-contract-vocabulary/specs/agent-contract-vocabulary/spec.md`, `VL2-EVT` category) already carries both rows: `VL2-EVT-18` **loop-internal turn-scoped loss** and `VL2-EVT-19` **harness-facing history-guarded truncation**, each with content matching `design.md` §4.2–§4.3, introduced by the category header note "Rows `VL2-EVT-18` and `VL2-EVT-19` are authored here on AG-01's behalf, at AG-01's own recorded request... so that AG-01's own amendment duty is satisfied by citation rather than a second append." `decision.md` §4 and §12 cite both rows by identifier and record that they were authored at AG-00 on AG-01's request. This change's own diff makes no edit to AG-00's register file.

## Phase 9 — Verification and the closing-checklist walk

- [x] 9.1 Verify `design.md` §14's eight design-phase acceptance criteria against the merged `decision.md` (section order, five-part shape, §4.2/§4.3 on the page, §5.1/§5.3 traces with impossibility table, rendezvous objection answered once, §6/§7 diagrams present, §10 inheritance complete, AG-00 amendment follows §10's procedure).

  **Remediation note (verify pass, 2026-08-11).** Criterion 5 ("rendezvous objection answered once, cross-referenced rather than repeated") was recorded as verified here, but at that time `decision.md` stated the objection's full proof — including the verbatim Layer 1 quote — in both §4 and §5. Corrected by trimming §4's version to its conclusion plus a cross-reference to §5, where the full proof now lives alone. Criterion 5 now genuinely holds; this note records that the original checkmark predated the fix, so a future reader does not trust an unearned checkmark over this note.
- [x] 9.2 Run the eight-item verification pass from `design.md` §13, ordered by cost of a missed defect: (1) harness-facing narrowing + worked case + droppable set; (2) carrier grounds cite Layer 2 sources; (3) stalled-observer trace terminates at the lane + multicast row states "nothing impossible"; (4) rendezvous objection answered by the receive-step distinction; (5) three scopes, one closer each, leaf-first order; (6) one surface, two rejection granularities, both carve-outs, the recursion; (7) deferrals named (AG-23.2, AG-21.2, the two AG-00 rows); (8) deletion test, no Go identifiers, vocabulary cited by identifier.
- [x] 9.3 **Final task.** Re-walk AG-01.1's five closing-checklist items one by one against the merged artifact — carrier, backpressure, observer model, close and ownership, upward path — and record, for each, its evidence location (the `decision.md` section) and its rationale-plus-*what-this-excludes* pair, confirming `R-AGE-019`'s acceptance criterion and that the change merges before AG-04's first node begins. **[R-AGE-019; S-AGE-025, S-AGE-026, S-AGE-027]**

---

## Review focus

1. **The stalled-observer trace (4.2).** This is the single task whose absence fails `S-AGE-010` outright. Check that the trace names the forwarding activity as the termination point, not a convention or a documented rule.
2. **The harness-facing worked case (3.3).** Confirm the artifact reasons from the measured capacity 0, not the superseded 64-hypothesis, and that "saturated is ordinary, not exceptional" is stated explicitly.
3. **The three named choices (2.3, 3.5, 8.1).** Each is a claim `sdd-apply` must carry unmodified and a reviewer must be able to challenge on its own terms, independent of the rest of the decision.
4. **Sequencing of 8.1.** If AG-00's register section does not yet exist when this task starts, it blocks — do not invent the rows locally in this change's artifacts.
5. **Leaked Go identifiers.** No camelCase or PascalCase single-token names, no struct/interface/field-shaped language, anywhere in `decision.md` or the register amendment.

---

## PR forecast and review budget

| PR | Content | Forecast | Depends on |
| --- | --- | --- | --- |
| 1 | Six markdown artifacts under this change's directory, plus an append-only amendment to AG-00's register | ~1,600–2,200 lines of prose, **0 Go** | AG-00's register section merged or landing in the same PR |

`size:exception` accepted for this session; no further decision is required before `sdd-apply`.

## Acceptance criteria for the milestone

1. All five closing-checklist items (Phases 2–6) are answered in `decision.md`, each with rationale and *what this excludes*.
2. The AG-00 register amendment (8.1) is merged in the same pull request.
3. The verification pass (9.1, 9.2) is recorded complete, and the closing-checklist walk (9.3) is recorded with evidence per item.
4. `spec.md`'s `R-AGE-001`…`R-AGE-019` hold.
5. No Go identifier appears anywhere in the change; no file under `backend/`; no build or dependency file touched.

## Next

`sdd-apply` writes `decision.md` against Phases 1–8, then closes with Phase 9's verification and checklist walk. Then AG-02, AG-04, and through them AG-10, AG-13, AG-14, AG-19, AG-20 — none of which may invent its own channel, loss rule, or way back into a live run.
