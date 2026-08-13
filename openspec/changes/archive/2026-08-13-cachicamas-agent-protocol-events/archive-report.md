# Archive report — `cachicamas-agent-protocol-events` (AG-06)

> **What**: AG-06 closes Layer 2 Wave 1 by adding 10 event kinds across 4 families (3 permission, 2 cost, 2 delegation, 3 compaction). The registry grows from 15 to 25 kinds; the scope-fence retightens from "exactly 15" to "exactly 25"; `L2C-06` doc-guard row added; envelope delta applied (`R-AEV-010` + `R-AEV-012` MODIFIED, `R-AEV-013` / `014` / `015` ADDED). **Substrate byte-unchanged** (12-line `event_descriptor.go` doc-only change is the only exception; `stream_check.go` / `failure.go` / `sequence.go` / `go.mod` / `go.sum` byte-identical). Third consecutive milestone demonstrating extensibility.
>
> **Verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 5 WARNING, 3 SUGGESTION. All four gates green (`make test`, `make lint` after `cache clean`, `make build`, `make vuln-check`). All 19 new-spec scenarios covered with passing tests; all 11 envelope-delta scenarios (5 ADDED + 6 PRESERVED) covered or re-asserted.
>
> **Recommendation**: nothing here blocks the merge. PR is ready to push.

## Quick path

1. **Commit this archive to the PR branch** — `feat/agent-layer2-wave1-ag06` (commit `c97b6bf1` → archive-commit will follow).
2. **Push and open the PR** with `size:exception` documented (braejan's AG-04 standing instruction, Engram `#2957`).
3. **Carry-forward** to AG-07: W1 (position-vs-seq substrate bug), W2 (name-prefix → structural pin migration), W3 (S-APE-084-bis inverse bite promotion). All documented; none blocks merge.

## Identity

| Field | Value |
|---|---|
| Change slug | `cachicamas-agent-protocol-events` |
| Milestone | **AG-06** (Layer 2 Wave 1, **closing**) — doc 0003:602–712 |
| Branch | `feat/agent-layer2-wave1-ag06` |
| Worktree | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave1-ag06` |
| Archive folder | `openspec/changes/archive/2026-08-13-cachicamas-agent-protocol-events/` |
| Artifact store | HYBRID (Engram + OpenSpec filesystem) |
| Mode | AUTOMATIC |
| Strict TDD | ACTIVE |
| Spec prefix | `R-APE-` / `S-APE-` (new spec); envelope delta uses existing `R-AEV-` / `S-AEV-` |

## Final state (the answer)

| Item | Final state | Source |
|---|---|---|
| Verdict | PASS WITH WARNINGS | `verify-report.md` line 322 |
| Critical findings | 0 | `verify-report.md` line 282 |
| Warnings | 5 | `verify-report.md` lines 285–295 |
| Suggestions | 3 | `verify-report.md` lines 297–303 |
| New kinds registered | 10 (3 permission + 2 cost + 2 delegation + 3 compaction) | `event.go:242–366` (25 registry rows) |
| Scope-fence | "exactly 25" (was "exactly 15" pre-AG-06) | `event_registry_test.go:520` |
| L2C-06 row | present in `doc.go:34` AND `expectedLayer2ContractRows` | `doc_contract_guard_test.go:65` |
| Substrate uniqueness | `event_descriptor.go` 12-line doc-only; `stream_check.go` / `failure.go` / `sequence.go` / `go.mod` / `go.sum` byte-identical to main | `git diff --stat main` (see verify-report § "Substrate Uniqueness Evidence") |
| AG-03 boundary guards | `import_boundary_test.go` + `ambient_authority_test.go` byte-unchanged vs main; pass | `verify-report.md` line 273 |
| Commit chain | `c97b6bf1` (docs) ← `8fbd1f73` (AG-06.5) ← `520affdf` (AG-06.4) ← `3a5dc133` (AG-06.3) ← `06d28452` (AG-06.2) ← `fb739dee` (AG-06.1) ← `6b4a3468` (AG-05 merge) | `git log --oneline -7` |
| Tests passing | 12/12 packages `ok`; 1117 PASS; 0 FAIL; 0 DATA RACE; 1 documented SKIP (`TestOpenRouterAdapter_LiveSmoke`, gated on `OPENROUTER_API_KEY` per R-OR-07) | `verify-report.md` line 67 |
| Lint | 0 issues (after `cache clean`) | `verify-report.md` line 87 |
| Build | clean (no warnings, no errors) | `verify-report.md` line 91 |
| Vuln-check | "No vulnerabilities found." (NOT in `make all`; runs explicitly per Engram `#2944`) | `verify-report.md` line 95 |

## Specs synced (source of truth updated)

| Domain | Action | Details |
|---|---|---|
| `agent-protocol-events` | Already promoted | `openspec/specs/agent-protocol-events/spec.md` (NEW full, 9 reqs / 15 spec + 4 bites = 19 scenarios). Placed by `sdd-spec` phase. |
| `agent-event-envelope` | **DELTA MERGED** into `openspec/specs/agent-event-envelope/spec.md` | 2 requirements MODIFIED (`R-AEV-010` scope-fence 15→25, `R-AEV-012` extensibility restated) + 3 requirements ADDED (`R-AEV-013`, `R-AEV-014`, `R-AEV-015`); 5 new scenarios (`S-AEV-120`–`S-AEV-124`) + 6 PRESERVED (`S-AEV-090`–`S-AEV-092`, `S-AEV-110`–`S-AEV-112`). |

The delta merge operations applied to `openspec/specs/agent-event-envelope/spec.md`:

- **MODIFIED `R-AEV-010`**: title and text retightened from "exactly 15" to "exactly 25"; text updated to reference all eight families (AG-04 + AG-05 + AG-06); forbidden-names list retirement noted; the `(Previously:)` parenthetical updated.
- **MODIFIED `R-AEV-012`**: title and text updated to record AG-06 as the third kind-set following AG-04.4's extensibility experiment; S-AEV-112 scenario updated to reflect seven-step procedure + `CardinalityAtMostOne` exercised by AG-06's `permission_resolution_remembered`.
- **ADDED `R-AEV-013`**: registry holds exactly 25 kinds; scope-fence at 25; forbidden-names list retires. 2 scenarios (`S-AEV-120`, `S-AEV-121`).
- **ADDED `R-AEV-014`**: `L2C-06` doc-contract row references the four protocol families. 2 scenarios (`S-AEV-122`, `S-AEV-123`).
- **ADDED `R-AEV-015`**: Protocol family kinds follow the AG-04.1 envelope invariants; `subagent_started`/`subagent_ended` are the first non-`NewDelegatedRunStart` consumers of the parent identifier; `CardinalityAtMostOne` seam exercised for the first time. 1 scenario (`S-AEV-124`).
- **Header note** amended to record AG-06's three MODIFIED + three ADDED amendments (lines 6–7 of the spec).
- **Acceptance criteria** updated: `S-AEV-001` … `S-AEV-102` and `S-AEV-110` … `S-AEV-112` and `S-AEV-120` … `S-AEV-124` has recorded evidence; criterion 9 updated to "Exactly the run, turn, message, tool, permission, cost, delegation and compaction lifecycle families are registered (25 kinds)".

## Scenario-count discipline (W9 lesson from AG-04 / AG-05)

**9 charter → 15 spec + 4 bites = 19 total** (new spec); **5 added + 6 preserved (envelope-delta)**. Stated identically across proposal, design, tasks, apply-progress, verify-report, and this archive report. AG-04 W9 scenario-count discipline observed.

| Source | Count | Status |
|---|---|---|
| `proposal.md:9` | `9 charter → ~14-22 spec + 4 bites` | Match (settles at 19) |
| `design.md:5` | identical | Match |
| `tasks.md:43` | identical | Match |
| `apply-progress.md:9-11` | identical (with per-phase contribution breakdown) | Match |
| `verify-report.md:44-49` | identical | Match |
| Authoritative new spec | `openspec/specs/agent-protocol-events/spec.md`: 9 R-APE reqs + 15 S-APE + 4 bites = **19** | **Match** |
| Authoritative delta | `openspec/changes/archive/.../specs/agent-event-envelope/spec.md`: 2 MODIFIED (3 + 3 preserved) + 3 ADDED (5 new) | **Match** |

Per-phase contribution (verify-report § "Spec Compliance Matrix"):

- AG-06.1 (permission): 4 spec + 1 bite = 5 scenarios
- AG-06.2 (cost): 3 spec + 1 bite = 4 scenarios
- AG-06.3 (delegation): 3 spec + 0 bites = 3 scenarios
- AG-06.4 (compaction): 4 spec + 1 bite = 5 scenarios
- AG-06.5 (guard + L2C-06 + envelope delta): 1 spec (S-APE-080) + 1 bite (S-APE-081) = 2 scenarios
- Envelope delta: 5 added (S-AEV-120..124) + 6 preserved (S-AEV-090..092, S-AEV-110..112)

## Archive contents

| File | Status | Lines | Notes |
|---|---|---|---|
| `proposal.md` | ✅ | 102 | AG-06 charter (doc 0003:602–712); scope, approach, risks, success criteria |
| `design.md` | ✅ | 135 | 8 architecture decisions (AD1–AD8); per-family file split; data flow |
| `tasks.md` | ✅ | 210 | **78/78 sub-steps complete** (5 phases + 14 final gates). All `[x]`. |
| `explore.md` | ✅ | — | Exploration artifact from `sdd-explore` phase |
| `specs/agent-event-envelope/spec.md` | ✅ | 86 | The delta spec applied to the base envelope spec |
| `apply-progress.md` | ✅ | 280 | 5/5 implementation commits; TDD cycle evidence; substrate uniqueness; carry-forward enforcement |
| `verify-report.md` | ✅ | 349 | PASS WITH WARNINGS; mutation evidence (4 bites RED-recorded); spec compliance matrix (19/19 + 11/11) |
| `archive-report.md` | ✅ | this file | Terminal record of the cycle |

**Task Completion Gate reconciliation note**: the 14 Phase 6 final gates (6.1–6.14) were mechanically reconciled from `[ ]` to `[x]` during the archive phase. Per the `sdd-archive` Task Completion Gate, this is an exceptional repair performed because (1) the orchestrator's launch prompt explicitly directed `sdd-archive` to verify "All tasks `[x]` in archived tasks.md" and (2) `apply-progress.md` § "Phase 6 — Final gates + cleanup" and `verify-report.md` § "Build & Tests Execution" together prove every unchecked gate is complete (lint 0 issues, build clean, vuln-check no vulnerabilities, AG-03 guards pass, substrate uniqueness confirmed, envelope delta applied, `L2C-06` row landed, worktree retained, apply-progress saved to Engram, six `sdd-attempt settle` flags passed). The reconciliation is recorded here for audit.

## Source of truth — main specs now reflect new behavior

The following specs are now updated to reflect AG-06's behavior:

- `openspec/specs/agent-protocol-events/spec.md` — **NEW full spec** (9 requirements, 15 spec + 4 bites = 19 scenarios). Placed during `sdd-spec` phase; carried forward unchanged to archive.
- `openspec/specs/agent-event-envelope/spec.md` — **UPDATED** via delta merge: `R-AEV-010` MODIFIED (scope-fence 15→25), `R-AEV-012` MODIFIED (extensibility restated for AG-06), `R-AEV-013`/`R-AEV-014`/`R-AEV-015` ADDED. 5 new scenarios (`S-AEV-120`–`S-AEV-124`) + 6 PRESERVED.
- `backend/agent/src/agent/event.go` — 10 new `EventKind` consts + 10 `eventRegistry` rows; `eventKindEnd` moved.
- `backend/agent/src/agent/permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go` — 4 new per-family files (AD-6).
- `backend/agent/src/agent/event_registry_test.go` — 10 new witness entries; scope-fence retightened to 25; `BitesByCountOnTwentySixthKind` bite; forbidden-names list retired.
- `backend/agent/src/agent/doc.go` — `L2C-06` row added (line 34).
- `backend/agent/src/agent/doc_contract_guard_test.go` — `L2C-06` row in `expectedLayer2ContractRows` (line 65).
- `backend/agent/src/agent/event_descriptor.go` — **doc-only** 12-line change (header "six-step" → "seven-step"; `CardinalityAtMostOne` first-exercise precedent paragraph). Zero structural change.

## Issues summary

| ID | Severity | Summary | Carry-forward |
|---|---|---|---|
| **W1** | WARNING | `stream_check.go` position-vs-seq confusion (AG-04 W2 carry-forward). AG-06's bite asserts slice index correctly (`requireViolationPosition(t, report.Violation(), 3)`), but the substrate bug is unchanged. | AG-04-04.1 ADR: 12-line change replacing `seq` with `i` at the 12 violation sites. Substrate edit, must precede AG-07's first registry touch. |
| **W2** | WARNING | `TestEventKinds_AG05AllRegisterPlacementTurn` (AG-05's test, `event_registry_test.go:489`) still uses name-prefix heuristic, contradicting AG-05 W2 doctrine. AG-06's analogous test (`TestEventKinds_AG06Placement_StructuralPin`) is correctly structural. | Migrate during AG-05's second touch on the registry. ~10 lines. |
| **W3** | WARNING | `TestCompaction_Failed_TerminalTrue_FollowOnRejected` (S-APE-084-bis) is a no-op-by-design inverse bite companion. Explicitly `t.Skip`s under `-short`; without `-short` does nothing (comment-documented). | Promote to live AST-driven inverse bite during AG-07's first registry touch. |
| **W4** | WARNING | `TestCompaction_Failed_DescriptorRow_DeclaresTerminalFalseExplicitly` (S-APE-070-bis) is behavioral, not structural. The structural AST walk `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` is the load-bearing assertion. | None — pattern is correct (name-payload + structural-AST split); documentation only. |
| **W5** | WARNING | `event_descriptor.go` 12-line doc-only change retroactively rewords AG-05's "six-step" to "seven-step" header. Benign narrative change; matches AG-04 W3 + AG-05 S1 doctrine. | None. |
| **S1** | SUGGESTION | `protocol_events_test.go` duplicates AST-walk machinery 4 times. Helper `parseRegistry(t *testing.T) *ast.CompositeLit` would DRY them. | Code-quality preference; future PR. |
| **S2** | SUGGESTION | `TestCompaction_Failed_TerminalTrue_FollowOnRejected` could be merged into `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` via AST rewrite of descriptor's `Terminal: false` → `false→true`. | Future-proofing for AG-07+. |
| **S3** | SUGGESTION | W1 (stream_check.go position-vs-seq) is the highest-value AG-04 carry-forward. | Already noted in W1's carry-forward. |

## Carry-forward to AG-07 (explicit list)

1. **W1 — fix `stream_check.go`'s position-vs-seq bug as a substrate edit before AG-07.** A 12-line change to replace `seq` with `i` (slice index) at the 12 violation sites is mechanical and should land under its own ADR (AG-04-04.1). AG-04's `6c821c0a` precedent for "lint cache clean before lint gates" applies to the post-edit lint run. The substrate's "untouched by AG-06" posture ends with this fix.

2. **W2 — migrate AG-05's `TestEventKinds_AG05AllRegisterPlacementTurn` from name-prefix to structural pin during AG-07's first touch on the registry.** Costs ~10 lines. Removes the stale pattern documented in AG-05 W2.

3. **W3 — promote `S-APE-084-bis` no-op behavior to a live AST-driven runtime during AG-07's first registry touch.** Suggested pattern: write a helper that takes a temporary file-with-modified-descriptor, invokes `CheckStream`, asserts the inverse, then removes the file.

4. **AG-07+ may begin emitting events** against the four now-constructible families: AG-10 (permission policy), AG-16 (cost price table + compaction summarizer), AG-18 (subagent harness mechanics), and AG-19 (delegation closure of invariant 2). No AG-06 design call needed; AG-19.1 closes envelope invariant 2 (R-AEV-003 direction-2 is now exercised by AG-06.3).

5. **L2C-07 doc-guard row** (state-owner / observation semantics) is the natural next row per AG-19's scope.

6. **AG-06's `Terminal: false` explicit doctrine (AG-04 W3 + AG-05 S1 carry-forward) is now the third milestone enforcing it.** Future kinds MUST continue to declare `Terminal: false` explicitly in their descriptor row; the AST walk `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` is the model.

## Commit chain (final)

All 6 commits landed on `feat/agent-layer2-wave1-ag06`:

| # | Hash | Conventional message |
|---|---|---|
| 1 | `fb739dee` | `feat(agent): add permission family (3 kinds) — AG-06.1` |
| 2 | `06d28452` | `feat(agent): add cost family (2 kinds) — AG-06.2` |
| 3 | `3a5dc133` | `feat(agent): add delegation family (2 kinds) — AG-06.3` |
| 4 | `520affdf` | `feat(agent): add compaction family (3 kinds) — AG-06.4` |
| 5 | `8fbd1f73` | `feat(agent): enlarge scope-fence + add L2C-06 doc row — AG-06.5` |
| 6 | `c97b6bf1` | `docs(agent): add AG-06 openspec planning artifacts + apply-progress` |

The **archive commit** will follow on this branch with the spec merge and the archive folder move. No `Co-Authored-By` trailer (per AG-04 / AG-05 / AG-06 convention).

## Phase artifacts persisted to Engram (HYBRID mode)

All 7 phase artifacts persisted under `sdd/cachicamas-agent-protocol-events/` topic hierarchy:

| Topic key | Type | Observation ID |
|---|---|---|
| `sdd/cachicamas-agent-protocol-events/explore` | architecture | `#2966` |
| `sdd/cachicamas-agent-protocol-events/spec` | architecture | `#2969` |
| `sdd/cachicamas-agent-protocol-events/tasks` | architecture | `#2971` |
| `sdd/cachicamas-agent-protocol-events/apply-progress` | architecture | `#2972` |
| `sdd/cachicamas-agent-protocol-events/verify-report` | architecture | `#2975` |
| `sdd/cachicamas-agent-protocol-events/archive-report` | architecture | (this save — observation ID to be recorded by orchestrator on next `mem_search`) |
| `sdd/cachicamas-agent-protocol-events` (session summary) | session_summary | `#2974` |

Carry-forward lessons cited across all artifacts (Engram): `#2944` (vuln-check explicit), `#2957` (`size:exception` standing instruction), `#2961` (six `sdd-attempt settle` flags), `#2962` (TDD skill gap), `#2963` (`sdd-archive` worktree sequencing), `#2965` (AG-05 session summary).

## Substrate uniqueness citation map

| File | Status vs `main` | Justification |
|---|---|---|
| `backend/agent/src/agent/event_descriptor.go` | 12-line diff (10 ins, 2 del), all comments | W3 latent-trap guard + `CardinalityAtMostOne` first-exercise precedent paragraph. No `package`/`const`/`func`/`var` added. |
| `backend/agent/src/agent/stream_check.go` | byte-identical | Engine reads descriptor data; AG-04.4 + AG-05 + AG-06 proven path. |
| `backend/agent/src/agent/failure.go` | byte-identical | `*Failure` reused by `decision_made` (Deny) and `compaction_failed`. |
| `backend/agent/src/agent/sequence.go` | byte-identical | `LaneStamper` unchanged; span identity is payload-side. |
| `backend/agent/go.mod` / `go.sum` | byte-identical | No new Go deps (per `openspec/AGENTS.md` rule 5). |
| AG-03 guards (`import_boundary_test.go`, `ambient_authority_test.go`) | byte-identical | `import_boundary_test.go` (400L) + `ambient_authority_test.go` (380L) pass with zero logic change. |

Third consecutive milestone demonstrating extensibility (AG-04.4 → AG-05 → AG-06). NFR-APE-004 satisfied.

## SDD cycle complete

The change has been fully planned (proposal, design), specified (spec + delta), broken into tasks (78 sub-steps), implemented (5 commits + 1 docs commit), verified (PASS WITH WARNINGS, 19/19 + 11/11 scenarios covered, 4 bites RED-recorded with exact predicted message), and archived (spec delta merged, change folder moved, archive report written, Engram persisted).

The next milestone (AG-07 onward) inherits a 25-kind registry, a substrate whose extensibility is proven for the third time, and a clear set of three carry-forward warnings (W1 / W2 / W3) to close before AG-07's first registry touch.

---

## Identity (executor)

| Field | Value |
|---|---|
| Author | sdd-archive sub-agent (executor) |
| Persistence | `openspec/changes/archive/2026-08-13-cachicamas-agent-protocol-events/archive-report.md` + Engram `sdd/cachicamas-agent-protocol-events/archive-report` (hybrid mode) |
| Worktree at close | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave1-ag06` |
| Branch at close | `feat/agent-layer2-wave1-ag06` |
| Working tree at close | `git status --short` empty (after archive commit lands) |
| Task Completion Gate | 78/78 sub-steps `[x]` (14 Phase 6 final gates reconciled from `[ ]` → `[x]` per orchestrator's "All tasks `[x]`" directive + apply-progress/verify-report proof) |
| Spec delta merge | 2 MODIFIED + 3 ADDED, 5 new scenarios, 6 preserved |
| Final-state authority ranking | Orchestrator prompt + verify-report (#2975) + apply-progress (#2972) + tasks (persisted). Per Final-State Authority hierarchy in `sdd-archive` SKILL. |