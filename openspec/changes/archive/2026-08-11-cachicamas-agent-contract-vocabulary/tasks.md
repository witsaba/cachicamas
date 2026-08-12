# Tasks — Layer 2 contract vocabulary

> **Change**: `cachicamas-agent-contract-vocabulary`
> **Milestone**: AG-00 · **Node**: AG-00.1 `[decision]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Inputs**: `proposal.md`, `specs/agent-contract-vocabulary/spec.md`, `design.md`
> **Precedent**: `openspec/changes/archive/2026-07-31-cachicamas-ai-contract-vocabulary/tasks.md`

---

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,600–2,200 (register ~70–80 rows across six categories + `decision.md`; spec/design/proposal already largely authored) |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR — `size:exception` |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

This change is documentation-only and ships as part of a three-change single PR whose combined total exceeds 1,000 lines. `size:exception` was already accepted by the user at session preflight — recorded here as accepted, not re-asked. Splitting the register across PRs would reproduce the partial-definition state this milestone exists to prevent (same rationale as the Layer 1 precedent).

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Register authored in spec delta + `decision.md` written + verification pass recorded | Single PR (`size:exception`) | N/A — inspection only, no test command; nothing under `backend/` changes | N/A — `[decision]` leaf, no runtime, no `make test` gate (doc 0002 evidence gate) | `git revert` the merge commit or delete `openspec/changes/cachicamas-agent-contract-vocabulary/` — valid while no `VL2-*` id is cited downstream (proposal rollback level 1) |

---

## Node type and what it means for this task list

AG-00.1 is a **`[decision]` leaf**: a recorded choice with a closing checklist, no production code, closing when the artifact answers every listed question and is merged. There is no red-green-refactor cycle and no `make test` gate — `openspec/config.yaml`'s `apply.tdd: true` binds Go service code, and this change writes none. The whole milestone is one phase, one node: the forecast above is degenerate to a single PR.

---

## Phase AG-00.1 — Register authoring (closing checklist)

- [x] **T-AGV-1** — Write the register skeleton into `specs/agent-contract-vocabulary/spec.md` under the reserved `## The register` heading: six category headings (`VL2-COR`, `VL2-EVT`, `VL2-LOOP`, `VL2-HAR`, `VL2-SEAM`, `VL2-OUT`) each with a term count, the register-wide sum, and the six standing amendment rules (append-never-invent; next free ordinal; dated blockquote naming what/who/why; no silent edit — struck-through supersession; update counts; land in the same PR). → `R-AGV-001, R-AGV-004, R-AGV-005, R-AGV-013`
- [x] **T-AGV-2** — Author `VL2-COR` rows: runtime, loop, harness, run, turn, provider call, attempt (plus the reconciling statement that a turn spans one or more provider calls, exceeding one only via harness retry), transcript, pairing invariant, steering, suspension, delegation, cost scope — plus **seven** must-never obligation rows (six loop, one harness), each naming its enforcing guard. Fixed five-field row shape; single owning-milestone id per row. → `R-AGV-002, R-AGV-003, R-AGV-005, R-AGV-008`
- [x] **T-AGV-3** — Author `VL2-EVT` rows: event kind, the eight event families, the four invariants, run and turn outcomes, the stream-contract validator. → `R-AGV-002, R-AGV-003, R-AGV-005`
- [x] **T-AGV-4** — Author `VL2-LOOP` rows: pre-request hook, prefix stability, tool execution contract, effect class, ordered rejoin, policy slot, permission protocol, finish-reason dispatch, typed turn failure. → `R-AGV-002, R-AGV-003, R-AGV-005`
- [x] **T-AGV-5** — Author `VL2-HAR` rows: history, orphan synthesis, run driver, interrupt vs shutdown, bounded wind-down, retry policy, failover seam, composed bounds, cost aggregation. → `R-AGV-002, R-AGV-003, R-AGV-005`
- [x] **T-AGV-6** — Author `VL2-SEAM` rows: context strategy, token accounting, compaction and its artifacts (the compaction summary as a typed transcript entry, distinguishable from a model message), re-entrancy, derived permission scope, hook taxonomy, observer asynchrony (cross-referencing the `VL2-EVT` invariant row, never restating it), readiness contract. → `R-AGV-002, R-AGV-003, R-AGV-005`
- [x] **T-AGV-7** — Author the `VL2-OUT` exclusion register: permission policy content, sandbox semantics, tool source, summary quality, cross-session rule persistence, price and money (cost payload reconciliation — Layer 2 cost event is token-only, money is Layer 3 enrichment, both sides cited by document and section), session persistence, frontends, catalogs. Each row names an owning layer/port/composition root, never an `AG-NN` milestone; each row states no definition; citations to Layer 1's `V-OUT-*` rows where a concern is already assigned. → `R-AGV-002, R-AGV-005, R-AGV-011`
- [x] **T-AGV-8** — Author the four boundary-case rows from the spec plus the fifth from the proposal's decision 6, each phrased observably (what it is / what it deliberately is not): (1) a zero-tool-call turn is a complete, terminal turn; (2) the compaction summary is a transcript entry typed as a compaction artifact, distinguishable from a model message, not metadata beside history; (3) a steering message belongs to the next turn, with the edge case that a message queued during the final turn yields a new turn rather than being dropped; (4) a compaction call is a provider call but not a turn, with the definitional reason named; (5, added by the proposal) same mechanism as (4), stated once and cross-referenced. → `R-AGV-006`
- [x] **T-AGV-9** — Author the reuse-vs-wrap statement: reused identities (message identity, tool-call identity, finish reasons, usage) each cited to a `V-*` row of the live `ai-contract-vocabulary` register with no re-paraphrase; wrapped identities (events — stream-scoped vs run/turn-scoped envelope; ordering — independent agent-level counter, explicitly not the Layer 1 per-stream sequence; failure) each stating the Layer 2 addition the Layer 1 identity has no place for. → `R-AGV-007`
- [x] **T-AGV-10** — Author the name-fixation row: "the portable agent runtime" fixed as loop-plus-harness, not a third wrapping thing, not a synonym for either alone; exclusion (a) names the retired "brain" framing and the cognition-invites-policy reason it was retired; exclusion (b) states the "the agent runtime" disambiguation rule against Go's `runtime` package; "a Layer 3 application" fixed as the consumer term, used throughout the register with no row phrased as "the coding agent". → `R-AGV-009`
- [x] **T-AGV-11** — Author the delegation vocabulary rows: `subagent` canonical for the participant, `delegation` canonical for the relationship and event family, `child harness` and `nested run` recorded as synonyms each with its precise sense, the scope rule (shipped names — event kinds, scenario ids, test names, acceptance criteria — use only `subagent`/`delegation`), both sides of the conflict cited by document and section. → `R-AGV-010`
- [x] **T-AGV-12** — Quote all four of doc 0003's *Scope boundary* wording traps verbatim, each with its corrected phrasing and the record that it is a plausible-but-wrong reading. → `R-AGV-012`
- [x] **T-AGV-13** — Write `openspec/changes/cachicamas-agent-contract-vocabulary/decision.md`: AG-00.1's argument (why a vocabulary-first milestone, the three recorded corrections from the proposal), both resolved conflicts with their losing readings recorded, and the merge-day snapshot of the register. States the argument only — carries no competing normative definition. → `R-AGV-001`

---

## Verification pass (closes the milestone)

Run after T-AGV-1 … T-AGV-13. Every check is inspection; nothing executes.

- [x] **V-1** — No term name appears under two category headings and no two rows carry the same definition text; every recurrence across categories is a cross-reference, never a second definition. → `R-AGV-006, R-AGV-005`
- [x] **V-2** — Every non-excluded row's owner cell holds exactly one current `AG-NN` id that resolves against a doc 0003 milestone heading, and that milestone's charter covers the owned term. → `R-AGV-003`
- [x] **V-3** — Ordinals are unique within each category, no ordinal is shared between a superseded and a live row, and a plain-text search for `V-` versus `VL2-` across `openspec/specs/` partitions cleanly with no ambiguous hit. → `R-AGV-004`
- [x] **V-4** — Each category's stated count and the register's stated sum both equal the actual number of rows. → `R-AGV-005`
- [x] **V-5** — Scan every artifact of this change for camel-case/Pascal-case single-token names, struct or interface declarations, or field lists — none present. → `R-AGV-015`
- [x] **V-6** — `git status`/diff for this change touches only `.md` files under `openspec/changes/cachicamas-agent-contract-vocabulary/`, plus `specs/agent-contract-vocabulary/spec.md` at archive — no `backend/`, `go.mod`, `go.sum`, `Makefile`, or build/container path. → `R-AGV-015`
- [x] **V-7** — Walk AG-01's charter (doc 0003) noun by noun; every domain noun resolves against the register — observer, upward path, steering, carrier, interrupt. → `R-AGV-014, S-AGV-041`
- [x] **V-8** — Walk AG-02's charter noun by noun; every domain noun resolves — failover seam, subagent, and the forward-requirement identifiers it disposes. → `R-AGV-014, S-AGV-042`
- [x] **V-9** — Spot-check further charters (AG-03 and at least one later milestone) for an unresolvable domain noun; any finding is recorded as a register defect to be closed by amendment under `R-AGV-013`, never invented in the downstream SDD. → `R-AGV-014, S-AGV-043`

---

## Final task — re-walk the closing checklist and record evidence

- [x] **T-AGV-14** — Re-walk AG-00.1's four closing-checklist items plus the fifth boundary case added by the proposal, and record the evidence for each in `decision.md`'s verification table:

| # | Closing-checklist item | Register section | Evidence |
|---|---|---|---|
| 1 | Boundary cases (three original + the fifth added) | `VL2-COR` boundary-case rows (T-AGV-8) | Each row states is/is-not observably; `S-AGV-014`…`S-AGV-017` hold |
| 2 | Reuse-vs-wrap split stated | Reuse-vs-wrap statement (T-AGV-9) | Reused rows cite `V-*` only; wrapped rows state the Layer 2 addition; `S-AGV-019`…`S-AGV-021` hold |
| 3 | Must-nevers restated as citable obligations | `VL2-COR` must-never rows (T-AGV-2) | Seven rows, each naming its guard; `S-AGV-022`…`S-AGV-024` hold |
| 4 | Layer's name fixed, with both exclusions and the consumer term | Name-fixation row (T-AGV-10) | Both exclusions present; "a Layer 3 application" used throughout; `S-AGV-025`…`S-AGV-028` hold |
| 5 | Fifth boundary case (compaction call vs turn) | `VL2-COR` boundary-case rows (T-AGV-8) | States provider-call-not-turn with the definitional reason; cross-referenced from case 4 |

---

## Traceability — every requirement maps to a task

| Requirement | Task(s) |
|---|---|
| `R-AGV-001` | T-AGV-1, T-AGV-13 |
| `R-AGV-002` | T-AGV-2, T-AGV-3, T-AGV-4, T-AGV-5, T-AGV-6, T-AGV-7 |
| `R-AGV-003` | T-AGV-2 … T-AGV-7, V-2 |
| `R-AGV-004` | T-AGV-1, V-3 |
| `R-AGV-005` | T-AGV-1 … T-AGV-7, V-1, V-4 |
| `R-AGV-006` | T-AGV-8, V-1 |
| `R-AGV-007` | T-AGV-9 |
| `R-AGV-008` | T-AGV-2 |
| `R-AGV-009` | T-AGV-10 |
| `R-AGV-010` | T-AGV-11 |
| `R-AGV-011` | T-AGV-7 |
| `R-AGV-012` | T-AGV-12 |
| `R-AGV-013` | T-AGV-1 |
| `R-AGV-014` | V-7, V-8, V-9 |
| `R-AGV-015` | V-5, V-6 |

No requirement is left without a task.

---

## Acceptance criteria for the milestone

1. `R-AGV-001` … `R-AGV-015` hold, each verified by its scenarios (per the traceability table above).
2. T-AGV-14's evidence table is complete: all four closing-checklist items plus the fifth boundary case are answered in the register and in `decision.md`.
3. Both open conflicts (delegation; turn vs provider call vs attempt) are resolved with both sides cited; both reconciled conflicts (pairing invariant, steering, cost) are recorded with their disposition.
4. All four wording traps appear verbatim (T-AGV-12).
5. V-5 and V-6 confirm no Go identifier and markdown-only diff.
6. V-7 and V-8 confirm AG-01's and AG-02's charters are expressible in the register's terms.

## Next

- **AG-01** — consumes: observer, upward path, steering, carrier, interrupt.
- **AG-02** — consumes: failover seam, subagent, disposed forward-requirement identifiers.
- **AG-03** — consumes the seven must-never obligation rows for its guard work.
