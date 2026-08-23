# Design: CH-00 — Record the archetype's vocabulary, seam answers and v1 scope

## Technical Approach

Docs-only change (proposal D-1): two authored artifacts — `decision.md` (the readable record, complete answers) and the delta spec `specs/chat-archetype-contract/spec.md` (falsifiable shape assertions, prefix `CHT`) — plus two bookkeeping lines in doc 0005. The record follows AG-23's archived precedent (`openspec/changes/archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md`) adapted from its nine-category walk to CH-00's seven questions. Decisions below are **D-A..D-F**, distinct from the proposal's binding D-1..D-4, which are not re-opened here.

## Architecture Decisions

### D-A — Record structure and division of labour

**Choice**: `decision.md` section order follows the seven questions of `0005:219-225` verbatim: header block (change/milestone/sources, plus an `[!IMPORTANT]` rules block mirroring AG-23 `decision.md:13` restating (a) the generic-client boundary — chat concepts allowed, coding-archetype concepts forbidden (`agent-layer3-handoff/spec.md:183`), and (b) the `PreRequestHook` frozen-and-superseded prohibition (`archive/2026-08-21-…/decision.md:45`)) → § 1 how to use → § 2 vocabulary (Q1) → § 3 seam answers (Q2) → § 4 gap findings (D-D) → § 5 identity (Q3) → § 6 persistence (Q4) → § 7 frontend (Q5) → § 8 deferral register (Q6, D-E) → § 9 substitution rule (Q7) → § 10 inconsistency register (D-F) → § 11 closing-checklist verification (mirrors AG-23 § 6).
**Division of labour**: `decision.md` carries every answer completely — a reader answers any of the seven questions from it alone, without the spec and without source (`S-L3H-029` posture). The spec carries only (i) shape assertions about the record (counts, both-halves rule, marking devices, absence rules) and (ii) the contract claims later CH milestones cite as requirements. No answer lives only in the spec.
**Alternatives**: spec-as-primary with `decision.md` as commentary — rejected: CH-00's charter (`0005:208`) makes the record the citation target. Free-form section order — rejected: question order makes checklist closure verifiable by walking, not reconstructing.

### D-B — Mapped vs coined vs inherited nouns: structural, not typographic, separation

**Choice**: two tables under two headings. **Mapped nouns**: columns `Noun | Maps onto (exactly one VL2-* id) | Layer 2 term | Citation` — the ten mapped rows from the proposal (run `:135`, turn `:136`, transcript, the loop, the harness, upward path, steering, message lifecycle, run outcome, turn outcome). **Coined nouns**: columns `Noun | Gap that forces the coinage | Nearest Layer 2 row and why it is NOT a mapping` — the coined table structurally has **no** "Maps onto" column, so a coinage cannot carry a `VL2-*` id in a mapping position; `VL2-*` ids appear there only as evidence of absence (`VL2-OUT-07` is an exclusion assigned to Layer 3, `VL2-COR-14` is the subagent sense of "participant").
**"Conversation" is a coinage, not a mapping onto `VL2-COR-04` run.** Cardinality decides it: a run is one harness invocation; CH-06 persists conversations across turns and process restarts, so one conversation is realized by one-or-more runs (continuation via the frozen transcript seam), and `0005:115` calls it "the archetype's model of one". A mapping must preserve identity; 1:N is containment.
**Alternatives**: bold/italic or symbol marking within one table — rejected: typography is not mechanically assertable and survives copy-paste badly. "Conversation" as mapping — rejected: collapses under CH-06's persistence; would claim a mapping the register does not support (proposal question 3).

> **Amended 2026-08-22 (verify round 1, CRITICAL C-1).** Two tables became **three**. The record uses the noun **message**, which Layer 2 does not define — its only message-headed row, `VL2-EVT-04` *message lifecycle*, is an event family and is already mapped under its own name — while Layer 1 does define it, as `V-REQ-02` (`openspec/specs/ai-contract-vocabulary/spec.md:111`). Neither block written above was truthful for it: a coined row asserts an invention that did not happen, a mapped row asserts a Layer 2 term that does not exist. D-B's reasoning is unchanged and is what forced the widening — the separation stays structural, so a third provenance needs a third table rather than a prose caveat inside an existing one. The **inherited nouns** table therefore carries columns `Noun | Inherited from (exactly one Layer 1 V-* id) | Layer 1 term | Why it is neither a mapping nor a coinage`, and has no "Maps onto" column, for the same reason the coined table has none. `R-CHT-001` was widened to three blocks with its exhaustiveness clause preserved, and `S-CHT-004` added for the new table's shape. Precedent: 19 promoted specs already open with a `> **Binding vocabulary**` header naming the Layer 1 rows they inherit by identifier (`openspec/specs/ai-message-roles/spec.md:9`), so inherited-as-a-distinct-block is this repository's existing convention, not a new device.

### D-C — The 11-seam table layout

**Choice**: one table, rows in AG-23 source order (`archive/2026-08-21-…/decision.md:31-53`), columns: `# | Seam (AG-23's name) | Status | Injection point | v1 answer | Owner milestone`. Exactly 11 rows (8 + 3). `Status` is the literal token `frozen` or `experimental — not frozen` (echoing the heading at `:49`) — a column, not a footnote, so the spec can count both classes and a reader cannot mistake one for the other. A deliberately empty answer is written as a full sentence in the `0005:116` form — what empty means operationally, where it is injected, and which milestone (or "no owner") ends the emptiness — never the bare token "none"/"n/a"/empty cell.
**Alternatives**: prose paragraphs like AG-23 — rejected: `S-L3H-028`'s no-row-without-both-halves rule becomes assertable only by parsing prose. Separate experimental table — rejected: splits the "exactly 11, none omitted" count into two assertions that can drift independently.

### D-D — Gap findings: structurally incapable of reading as seam answers

**Choice**: own H2 immediately after § 3, titled "Gap findings against Layer 2's own seam set — findings, not seam answers". Opens by stating the frozen enumeration is exactly the 11 rows above and this section extends nothing. Columns: `Finding | Where Layer 2 names it | AG-23 status | Disposition` — deliberately **no** injection-point and **no** v1-answer columns, so a finding row is structurally unreadable as a seam answer (same device as D-B). Rows (all re-resolved this phase): v2 § 6 seam 3 Sandbox (`0001-…v2.md:670`; rides `PolicySlot any`, `tool.go:117`), seam 7 Retry classification (`:674`), `Harness.RetryTiming` (doc comment `harness.go:96` "the injected clock and wait-function seam", field `:100`), `RetryAttempts` (`:94`), `ContextBudget` (`:120`), `System` (`:56`), and the `ToolSource` note (`tool.go:265-266` — the sentence completes as "AG-20's widening."; the record cites it as wrapped, not truncated).
**Alternatives**: append gap rows to the seam table marked "unenumerated" — rejected: that IS extending the frozen enumeration. Omit — rejected by binding D-2.

### D-E — Deferral register: reproduce doc 0005's table; AG-23's register is cited, never merged in

**Choice**: § 8 reproduces doc 0005's "Explicitly deferred until after v1" table (`0005:997-1011`) in substance, citing `0005:997` as source, every row keeping its attaching seam — because the record must answer Q6 directly, and it is the citation target for twelve milestones. AG-23's known-limitations register (`archive/2026-08-21-…/decision.md:103-108`) is **cited as inherited Layer 2 input, not reproduced**: its four rows are Layer 2's limitations, already inherited via that record's § 5; restating them here would create a second count-bearing copy that drifts. The record adopts § 4.3's rule (heading `:110`) for its own register, quoting the archived text verbatim: a defect "appears **nowhere** in the register above. Laundering it as a limitation would make this statement false on the day it lands." — therefore F-1/F-2/F-3 live only in the inconsistency register, never as deferral rows.
**Alternatives**: cite-only — rejected: fails the direct-answer rule. Narrow to "chat-relevant" rows — rejected: all rows are chat-relevant by construction and narrowing invites silent drift against the source table.

### D-F — Inconsistency register: one table, both sides cited, disposition stated

**Choice**: § 10, last content section before closing verification, mirroring doc 0005's own register (`0005:93-100`). Columns: `# | Side A (cited) | Side B (cited) | Disposition`. Exactly rows F-1, F-2, F-3 with the proposal's IDs and citations unchanged; header line states verbatim intent: "Recorded, not repaired. No promoted spec is modified by this change." F-1 and F-2's rows note the shared landing spot (`agent-contract-vocabulary/spec.md`) so one future change closes both. F-3's row carries the corrected guard-chain location (`frontend/src/routes/home/layout.tsx:30-34`, doc comment `:21-29`, imports `:15-17`).
**Alternatives**: inline findings at point of use — rejected: later milestones need one checkable place; doc 0005's precedent is a single register.

## Data Flow

Not applicable — no runtime component changes. Artifact dependency only:

    AG-23 archive decision.md ──→ decision.md § 3 (seam order)
    VL2 archive register ────────→ decision.md § 2 (mapped rows, defect F-1)
    doc 0005 :997-1011 ──────────→ decision.md § 8
    decision.md ─────────────────→ specs/chat-archetype-contract/spec.md (shape assertions)

`rules.design` diagrams/config rules: N/A — no cross-service change, no configuration change (before/after diff rationale not applicable).

## File Changes

| File | Action | Description |
|---|---|---|
| `openspec/changes/cachicamas-chat-vocabulary-and-scope/decision.md` | Create | The record, per D-A structure |
| `openspec/changes/cachicamas-chat-vocabulary-and-scope/specs/chat-archetype-contract/spec.md` | Create | Delta spec, `CHT` requirements/scenarios |
| `openspec/changes/cachicamas-chat-vocabulary-and-scope/design.md`, `tasks.md`, `verify-report.md` | Create | SDD artifacts |
| `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` | Modify | CH-00.1 checklist row ticked; status → 1 of 12 |
| `backend/**`, `frontend/**` | Unchanged | D-1 — zero source edits |

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Regression | Nothing regressed | `cd backend/agent && go test -race -count=1 ./...` — uncached, wall-clock recorded; a `(cached)` or sub-second result is not evidence |
| Lint/build | Repo gates | `golangci-lint cache clean && make lint`; `make build`; `make vuln-check`; **`make all` MUST NOT run** |
| Spec | Record shape | Verify walks each CHT scenario against `decision.md` manually (no CLI, no CI) |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Rollback per proposal: revert the single PR; post-CH-01 defects amend, never revert.

## What the spec must turn into falsifiable scenarios

Each item names the design decision it enforces; sdd-spec must assert these shapes, not re-derive them:

1. **(D-B, as amended)** Every archetype noun appears in exactly one of the **three** vocabulary tables; every mapped row's "Maps onto" cell contains exactly one `VL2-*` id; **neither** the coined table **nor** the inherited table carries a "Maps onto" column; every coined row names the gap that forces it, and every inherited row names exactly one Layer 1 `V-*` id with its citation plus why the noun is neither a mapping nor a coinage. "Conversation" and "participant" are in the coined table; "message" is in the inherited table — see D-B's 2026-08-22 amendment above for why two tables became three.
2. **(D-C)** The seam table has exactly 11 rows in AG-23 source order; every row has a non-empty `Injection point` and non-empty `v1 answer` cell (`S-L3H-028`); exactly 8 rows carry `frozen` and exactly 3 carry `experimental — not frozen`; no `v1 answer` cell is "none", "n/a", or empty.
3. **(D-C, S-L3H-029)** Every seam answer is stated directly — the scenario asserts no cell defers to "see source" or a bare file path as its answer.
4. **(D-D)** The gap-findings section exists, is separately labelled as findings-not-answers, carries no injection-point/v1-answer columns, and its opening sentence states the frozen enumeration is exactly the 11 above.
5. **(D-A rules block)** The record contains no coding-archetype concept (files, shells, skills, terminals as capabilities of this archetype) and nowhere describes `TurnOptions.PreRequestHook` as removed or deprecated.
6. **(D-E)** The deferral register reproduces doc 0005's table with every row carrying an attaching seam; no row is a bare exclusion; no F-* defect appears in it.
7. **(D-F)** The inconsistency register carries a row for each found defect (F-1, F-2, F-3), each with both sides cited and a disposition; the record states "recorded, not repaired"; F-3 cites `home/layout.tsx:30-34` as the correct location.

   > **Amended 2026-08-22 (spec phase).** This item originally read "carries exactly F-1, F-2, F-3". `R-CHT-012` asserts a row per named defect plus per-row completeness instead of a closed total, because a bare-total assertion in a spec that is promoted and later appended to is a known drift class in this repository (`agent-observability-boundary/spec.md:9` — the header states ranges and never totals). The set of defects is still exactly the three named in `proposal.md`; what changed is that no requirement asserts the number.
8. **(D-1)** The change's evidence gate: promoted spec plus green uncached race run — the headline acceptance scenario from `0005:210` (a reader gets any seam's answer with injection point, without reading source).
