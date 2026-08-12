# Archive Report — `cachicamas-agent-message-tool-events` (AG-05)

**Change**: `cachicamas-agent-message-tool-events` · **Milestone**: AG-05 (Layer 2, Wave 1) · **Status**: **ARCHIVED WITH KNOWN WARNINGS — 0 CRITICAL, 3 WARNING, 2 SUGGESTION carried forward**

**Archived to**: `openspec/changes/archive/2026-08-12-cachicamas-agent-message-tool-events/` (hybrid store — both OpenSpec filesystem and Engram)

**Verification verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 3 WARNING, 2 SUGGESTION (per `sdd/cachicamas-agent-message-tool-events/verify-report`, Engram #2957, captured 2026-08-12 17:38:12)

**SDD cycle completion**: Proposal → Spec → Design → Tasks → Apply → Verify → **Archive (this report)**

**PR state**: OPEN (PR #164, not yet merged by user). Worktree preserved per AG-04 pattern. Main is at `967d043f` (AG-04 merge). No cleanup of branch / worktree / main performed at this archive step.

---

## Executive Summary

AG-05 (`cachicamas-agent-message-tool-events`) ships the **next two of eight** Layer 2 v1 event families — **message lifecycle** (`VL2-EVT-04`) and **tool execution** (`VL2-EVT-05`) — bringing the registry from 4 to 15 kinds. AG-05 is the **first kind-set to exercise `PlacementTurn`**, the seam `event_descriptor.go:78-85` reserved at AG-04.3, and ships the **reconstruction property** (AG-05.3) that proves the `L2C-04` `0003:418` membership criterion — "if it is not on the stream, no frontend can render it and no log can reconstruct it" — for interleaved message + tool streams before any producer exists.

**Real deliverable size**: 13 files changed, 2,479 insertions, 40 deletions, 3 internal commits (AG-05.1, AG-05.2, AG-05.3). All within the pre-authorized `size:exception` band (1500–2200 forecast). AG-05 is the second milestone to ship under `size:exception` after AG-04.

**Evidence and compliance**: All 7 charter Gherkin scenarios restate as 15 spec scenarios (`S-AMT-001`..`S-AMT-081`) plus 3 added scenarios on the envelope delta (`S-AEV-110`..`S-AEV-112`). 4 bites recorded RED before properties went GREEN: `S-AMT-021` (delta → no-accumulated-snapshot route), `S-AMT-071` + `S-AMT-072` (reconstruction helper isn't vacuous), `S-AMT-081` (scope-fence bites on 16th scratch kind). All four Makefile gates (`test`, `lint`, `build`, `vuln-check`) clean.

**AG-05 bet preserved**: byte-unchanged files (`stream_check.go`, `failure.go`, `sequence.go`, `go.mod`, `go.sum`) verified per `git diff main`. The validator's rule engine (`stream_check.go:161` `PlacementTurn` rule) exercises AG-05's 11 new kinds with zero code edit; `R-AEV-008`'s `agent.Failure` wrap is reused by `R-AMT-006`'s `ToolEndExecutionFailure`. AG-04's extensibility experiment (`S-AEV-092`) is now an asserted spec requirement (`R-AEV-012`), not just a recorded experiment.

**Open defects inherited by AG-06**: 3 WARNING + 2 SUGGESTION findings remain as methodological notes. The two WARNING findings that propagate to AG-06 (`W1` vacuous helper; `W2` name-prefix test) are documented as **carry-forward lessons**, not blockers. The AG-04 lesson about scenario-count drift (W9) was honored: 7 charter → 15 spec restated identically across proposal, tasks, and apply-progress.

**Spec promotion**: `agent-message-tool-events` (new capability, 9 requirements, 15 scenarios + 4 bites) promoted to `openspec/specs/agent-message-tool-events/spec.md` verbatim. `agent-event-envelope` delta applied in place to `openspec/specs/agent-event-envelope/spec.md` (per AG-04's pattern, pre-authorized in this session): `R-AEV-007` and `R-AEV-010` MODIFIED (invariant 1 co-closure now joint with AG-05.1; scope-fence retightened from "exactly 4" to "exactly 15"); `R-AEV-012` ADDED (documents the AG-04.4 extensibility experiment path AG-05 took).

---

## Change Artifacts

| Artifact | Location | Type | Observation ID |
| --- | --- | --- | --- |
| Proposal | `openspec/changes/cachicamas-agent-message-tool-events/proposal.md` | Proposal | Engram #2948 |
| Exploration | `openspec/changes/cachicamas-agent-message-tool-events/exploration.md` | Exploration (sdd-explore) | Engram #2946 |
| Spec (concatenated) | `openspec/changes/cachicamas-agent-message-tool-events/specs/agent-message-tool-events/spec.md` (delta) | Full spec for new capability | Engram #2949 |
| Design | `openspec/changes/cachicamas-agent-message-tool-events/design.md` | Design decisions | Engram #2950 |
| Tasks | `openspec/changes/cachicamas-agent-message-tool-events/tasks.md` | Task breakdown | Engram #2951 |
| Apply progress | (Engram only — `sdd/cachicamas-agent-message-tool-events/apply-progress`) | Apply snapshot (intermediate) | Engram #2954 |
| Verify report | (Engram only — `sdd/cachicamas-agent-message-tool-events/verify-report`) | Verify snapshot (intermediate) | Engram #2957 |
| Delta spec | `openspec/changes/cachicamas-agent-message-tool-events/specs/agent-event-envelope/spec.md` | Delta on `agent-event-envelope` | (merged into main spec) |
| Archive report | `openspec/changes/archive/2026-08-12-cachicamas-agent-message-tool-events/archive-report.md` | This file | (Engram, at archive phase) |

**Promoted specs**:
- **New capability**: `openspec/specs/agent-message-tool-events/spec.md` (9 requirements `R-AMT-001..009`, 15 scenarios `S-AMT-001..081` + 4 bites, 7 of 7 charter Gherkin scenarios covered)
- **Amended capability**: `openspec/specs/agent-event-envelope/spec.md` (R-AEV-007 and R-AEV-010 MODIFIED; R-AEV-012 ADDED; traceability table row 12 added; contradiction #5 closed in place; acceptance criteria #1 and #9 updated; "Amended 2026-08-12 (AG-05)" header note added)

---

## Verification Verdict Authority and Final-State Facts

Per the Final-State Authority hierarchy (SDD Archive Skill § Final-State Authority):

**CRITICAL**: No CRITICAL issues block archive. Proceed to closure.

**WARNINGS and SUGGESTIONS**: Per verify-report.md (Engram #2957), 3 WARNING + 2 SUGGESTION findings were recorded at verification time (2026-08-12 17:38:12). All five are **methodological or cosmetic** — none are behavioral defects. No fix commits were made between verify-report capture and archive phase; the findings remain open as documented below.

### Warnings (W0, W1, W2)

| Finding | Category | Description | Owner | Status at archive |
| --- | --- | --- | --- | --- |
| **W0** | Orchestrator prompt | Orchestrator brief stated OpenSpec artifacts live under worktree path; actual location is untracked-in-main repo. Read from main repo (the source of truth). | Orchestrator / harness contract | **Cosmetic** — harness gap, not a deliverable defect. Carry forward to AG-06 inbox. |
| **W1** | Methodological | `reconstructString` (`message_text_test.go:309-339`) is a vacuous round-trip on its own input — `S-AMT-030` passes regardless of reconstruction logic. The real R-AMT-004 invariant is carried by `S-AMT-070`'s `TestReconstruction_Interleaved_TwoMessagesTwoTools_IndependentAndComplete` in `reconstruction_test.go`. | sdd-design: rewrite or delete S-AMT-030 in AG-06 if the test is being touched | **Carry forward to AG-06**: delete `S-AMT-030` or rewrite `reconstructString` to be non-vacuous. Do not let the "two tests for the same requirement" state stand. |
| **W2** | Methodological | `TestEventKinds_AG05AllRegisterPlacementTurn` (`event_registry_test.go:368-396`) is a name-prefix test, not a structural pin on `descriptor.Placement`. The structural pin is held indirectly by `stream_check.go:161` PlacementTurn rule. | sdd-design: split into name-check + placement-check tests | **Carry forward to AG-06**: when AG-06 registers permission/cost/delegation/compaction kinds, the placement test must split into a name check (C1) and a placement check (C2). The structural pin is the load-bearing assertions; the name check is the cosmetic. |

### Suggestions (S1, S2)

| Finding | Category | Description | Owner | Status at archive |
| --- | --- | --- | --- | --- |
| **S1** | Cosmetic | `event_descriptor.go:30-35` prose says "all 11 register `Terminal: false` explicitly"; the 11 rows use `{Placement: PlacementTurn}` with `Terminal` zero-value (functionally identical — `false` is `bool`'s zero value). | Cosmetic / W3 latent-trap guard review | **AG-06 pattern**: write `Terminal: false` explicitly in the AG-06 descriptor rows so the reviewer can tell intent at a glance. The `Terminal` field is now genuinely read by the engine (AG-04 fix `c203f25c`); explicitness is the latent-trap guard. |
| **S2** | Cosmetic | `TestToolCallOrdinal_*` reflection pin (`tool_event_test.go:283-289`) checks only field names; would not catch an unexported ordinal proxy under a different name. | Cosmetic | **AG-06 pattern**: when AG-06 introduces its own kind families, the reflection pin should also verify the field type (uint32 ordinal) so an unexported proxy under a different name cannot silently substitute. |

### Verification Evidence Detail

**Build**: PASSED (`make build` exit 0; commit-hash unchanged; output hash `sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495`)

**Tests**: PASSED (`make test` exit 0; `go test -race -v ./...`)
- 1,090 PASS, 0 FAIL, 0 `DATA RACE`, 0 `t.Skip` in `src/agent`
- All targeted AMT scenarios: 13 passed (see Spec Compliance Matrix in verify-report)
- AG-03 boundary guards pass unchanged: 8/8 (`TestLayer2_*`, `TestLayer2Agent_*`)

**Lint**: PASSED (`make lint` exit 0; "0 issues")
- `golangci-lint cache clean` invoked before each lint gate (cite AG-04 `6c821c0a` precedent)
- Same bit-identical content, no phantom finding

**Vuln-check**: PASSED (`make vuln-check` exit 0; "No vulnerabilities found", govulncheck v1.1.4)
- The skull: `make vuln-check` is NOT in `make all` (Engram `#2944`); ran explicitly

**AG-05 bet — byte-unchanged files**:

| File | `git diff main -- <file>` | Verdict |
| --- | --- | --- |
| `backend/agent/src/agent/stream_check.go` | empty | UNCHANGED — `PlacementTurn` rule at line 161 exercises AG-05's 11 new kinds with zero code edit |
| `backend/agent/src/agent/failure.go` | empty | UNCHANGED — `R-AEV-008`'s `agent.Failure` wrap reused by R-AMT-006's `ToolEndExecutionFailure` |
| `backend/agent/src/agent/sequence.go` | empty | UNCHANGED — envelope identity-shape preserved; ordinal is payload-side |
| `backend/agent/go.mod` | empty | UNCHANGED — no new top-level deps |
| `backend/agent/go.sum` | empty | UNCHANGED |

`event_descriptor.go`: **DOC-ONLY change** — six-step → seven-step procedure (added `5a. state Terminal: false explicitly`); the `Terminal` field is now read by the engine (AG-04's post-verify correction `c203f25c` is inherited). The `EventDescriptor` struct itself is unchanged. No validator rule-engine edit.

---

## Scope Fence Held

- **Exactly 11 new kinds registered**: 6 message (`message_start_text`, `message_delta_text`, `message_end_text`, `message_start_reasoning`, `message_delta_reasoning`, `message_end_reasoning`) + 5 tool (`tool_start`, `tool_progress`, `tool_end_success`, `tool_end_result_failure`, `tool_end_execution_failure`).
- **Total registry now 15 kinds** (4 AG-04 + 11 AG-05). Scope-fence `S-AEV-090` retightened from "exactly 4" to "exactly 15" in the same commit as the new kinds.
- **No AG-06 family kind present**: permission, cost, delegation, compaction families deferred to AG-06 per charter (`0003:420`).
- **`agenttest` never imported**: AD-4 (inherited from AG-04) honored; charter edges (AG-01, AG-03 only) enforced.
- **AG-03 guards pass untouched**: `import_boundary_test.go`, `ambient_authority_test.go` — zero logic edits, both pass at AG-05 close (8/8 `TestLayer2_*` / `TestLayer2Agent_*` tests).

---

## Size and Complexity Record

Per proposal § Affected Areas and launch context (orchestrator's final-state facts):

| Metric | Forecast (proposal) | Actual | Status |
| --- | --- | --- | --- |
| Total insertions | 1500–2200 | **2,479** | Within forecast band |
| Files changed | 13–17 (estimate) | **13** | Within forecast |
| Review budget risk | High | High | Single PR, `size:exception` pre-authorized |
| Commits | 3 (one per node) | **3** (AG-05.1, AG-05.2, AG-05.3) | On plan |
| Test count | 12–18 spec scenarios after expansion | **15 `S-AMT-001..081` + 4 bites + 3 `S-AEV-110..112` = 22** | Per-scenario expansion correctly applied |
| Charter coverage | 7 of 7 | **7 of 7** | None reduced |
| Bites recorded | 4 (S-AMT-021, S-AMT-071, S-AMT-072, S-AMT-081) | **4** | All required bites RED-recorded before properties GREEN |
| Test pass count | ~950 (estimate) | **1,090** | Above forecast; AG-03 guards still 8/8 |

**Honest recording**: Size was within forecast. Single PR; AG-05.3's reconstruction property references both message and tool kinds, so chained PRs were not safe (per proposal risk and `config.yaml` `rules.archive` warning). `size:exception` documented in PR body per AG-04 precedent.

---

## Structural and Invariant Closure

**Envelope invariant closure (per `0003:2203` spine)**:

AG-05 closes **envelope invariant 1 jointly with AG-04.3** (the indexed-delta pin, now instance-based, not just structural):

| Invariant | Closed by | AG-05's part | Status at AG-05 close |
| --- | --- | --- | --- |
| 1 — indexed deltas | AG-04.3 **+ AG-05.1** ✓ JOINTLY CLOSED | AG-05.1 introduced `message_delta_text` and `message_delta_reasoning`; the pin is now structural *and* instance-based. `S-AMT-021` (bite) mechanically asserts no route from a delta kind to an accumulated payload. | **Closed** at AG-05 archive |
| 2 — explicit nesting | AG-04.1 + AG-19.1 | None | Deferred to AG-19; AG-04.1's parent identifier carries |
| 3 — non-blocking observers | AG-01.1 + AG-20.2 (AG-05 absent) | None | Outside AG-05 |
| 4 — typed errors | AG-04.3 + AG-11.2 | None directly (AG-05.2's `ToolEndExecutionFailure` reuses `R-AEV-008`'s surface but does not close invariant 4) | AG-04.3 surface complete; AG-11 emission deferred |

**AG-05 closes invariant 1 jointly with AG-04.3.** That is the only invariant partial-closure AG-05 claims. The `R-AEV-007` contradiction bookmark #5 in the main spec is now closed in place: "the pin is now structural and instance-based".

**AG-05's reconstruction property**: `S-AMT-070` proves the `L2C-04` `0003:418` membership criterion for interleaved message + tool streams. The reconstruction helper is bite-tested twice (`S-AMT-071` drop-a-delta, `S-AMT-072` double-a-delta) before the property is GREEN. The bites guarantee the helper is non-vacuous.

**No spec line, test name or acceptance line claims otherwise.** Explicit non-requirements section in the new `agent-message-tool-events` spec (`NFR-AMT-001`..`004` + "Explicit non-requirements") enforces this bound.

---

## Traceability

### SDD Artifacts (Engram IDs)

- **Proposal** (#2948): Settled inputs, scope, approach, locked decisions A1–A5, rollback, open decisions, risk forecast. Source of truth for "why this change".
- **Exploration** (#2946): Architecture exploration pre-proposal. Captures the constraint surface, alternatives considered, and the loader for AG-04's six-step procedure.
- **Spec** (#2949): Concatenated spec across both affected domains (new capability + delta). 9 `R-AMT-001..009` requirements + 15 `S-AMT-001..081` scenarios + 4 bites, plus the delta on `agent-event-envelope` (MODIFIED `R-AEV-007`/`R-AEV-010`, ADDED `R-AEV-012`, 3 `S-AEV-110..112` scenarios). Given/When/Then + RFC 2119, all verifiable by `cd backend/agent && make test`.
- **Design** (#2950): Five locked decisions (A1–A5) carried forward from proposal; file layout, helper signatures, byte-unchanged inventory, four bites documented in TDD plan. Source of truth for "how it works".
- **Tasks** (#2951): 42 implementation tasks (Phases 1–3) + 0 archive tasks (archive is a separate SDD phase, not a tasks.md task; the 42 figure matches the verify-report's Task Completion Gate). All checkpoints `[x]`. Phases aligned to AG-05.1 / AG-05.2 / AG-05.3 leaves.
- **Apply progress** (#2954): Intermediate snapshot. Reports scenario count `7 charter → 15 spec + 4 bites + 3 envelope delta = 22` restated identically to proposal and tasks. All bites recorded RED before properties GREEN.
- **Verify report** (#2957): Verdict (PASS WITH WARNINGS, 0 CRITICAL), 3 WARNING + 2 SUGGESTION findings, reproducible defeat tests, 1,090 PASS, 0 FAIL, 0 DATA RACE. Intermediate snapshot; no fix commits between verify-report and archive.

### Promoted Specs (OpenSpec)

- **`openspec/specs/agent-message-tool-events/spec.md`**: New live capability (no prior spec). R-AMT-001..009, 15 scenarios S-AMT-001..081, 4 bites (S-AMT-021, S-AMT-071, S-AMT-072, S-AMT-081). All 7 charter Gherkin scenarios (`0003:543-583`, `0003:594-597`) represented; none reduced. Charter → spec mapping restated in the spec's `Coverage` table.
- **`openspec/specs/agent-event-envelope/spec.md`**: Amended in place at AG-05 close. Header note "Amended 2026-08-12 (AG-05)" added (line 6). R-AEV-007 and R-AEV-010 MODIFIED with `(Previously: ...)` notes recording the prior text. R-AEV-012 ADDED with 3 scenarios S-AEV-110..112. Traceability table gains `R-AEV-012` row (line 276). Contradiction #5 closed in place (line 286). Acceptance criteria #1 and #9 updated to reflect AG-05's contribution.

### Archived Folder Contents

```
openspec/changes/archive/2026-08-12-cachicamas-agent-message-tool-events/
├── proposal.md
├── exploration.md
├── design.md
├── tasks.md
├── specs/
│   └── agent-event-envelope/
│       └── spec.md (delta spec; merged into openspec/specs/ in place)
└── archive-report.md (this file)
```

(Apply progress and verify report live in Engram only — the archive preserves the filesystem artifacts; the Engram observations are addressable by id #2954 and #2957.)

---

## Key Decisions Recorded

Per proposal (locked at proposal; design did not silently re-open them):

1. **A1 — 6 message kinds locked at proposal**: `message_start_text`, `message_delta_text`, `message_end_text`, `message_start_reasoning`, `message_delta_reasoning`, `message_end_reasoning`. Reasoning + text have separate bracket lifecycles; kind-level segregation is asserted by `S-AMT-002` (no reasoning field in text payload).

2. **A2 — payload ordinal**: tool call ordinal is a payload field (`ordinal uint32` on `ToolStart`/`ToolProgress`/`ToolEnd*`), not an envelope field. R-13 traces the ordinal to Layer 1 payload-side (`doc 0002 AI-30`). Envelope identity-shape preserved.

3. **A3 — per-family file split**: `message_text.go` + `message_text_events.go` + `message_reasoning.go` + `message_reasoning_events.go` + `tool_event.go` + `tool_events.go`. Symmetric reasoning+text split for kind segregation at the file boundary, not just at the descriptor row.

4. **A4 — 6 symmetric reasoning+text**: reasoning mirrors text's bracket shape (start + N delta + end), providing the per-family reconstruction property.

5. **A5 — single PR + `size:exception`**: AG-05.3's reconstruction property references both message and tool kinds, so chained PRs are NOT safe. 1000-line pre-authorized budget exception; forecast 1500–2200 lines.

**Additional design decisions (resolved at design time, not in proposal)**:
- **Reconstruction helper**: lives in `reconstruction_test.go` (test-only, AD-4 from AG-04 honored). Bites-first ensures non-vacuous: drop-a-delta and double-a-delta both fail before `S-AMT-070` is GREEN.
- **Typed-failure wrap**: AG-05.2's `ToolEndExecutionFailure` reuses `R-AEV-008`'s `agent.Failure` wrap surface from AG-04.3. No new failure category added; mapping is the existing 9-to-9 Layer 1 mapping.
- **Six → seven-step procedure**: `event_descriptor.go` adds `5a. state Terminal: false explicitly` to the documented procedure. W3 latent-trap guard from AG-04 inherited; AG-06's `CardinalityAtMostOne` seam is reserved.

---

## Notes for Following Phases

### For AG-06 (Permission, Cost, Delegation, Compaction Families)

- **Inherit the validator unchanged**: Add new kinds following the seven-step procedure in `event_descriptor.go` (note: was six-step at AG-04 close; `5a. Terminal: false explicit` added at AG-05.3). No edit to `stream_check.go` rule engine needed. Extend the descriptor set; add scenarios for new families.
- **W1 lesson**: do not write a vacuous helper. The reconstruction helper in `reconstruction_test.go` is the pattern; the bites (`S-AMT-071`/`S-AMT-072`) are the safeguard. AG-06's reconstruction (if any) inherits the same bite-first discipline.
- **W2 lesson**: split name-check + placement-check tests. AG-06's `S-AEV-090` (now at 15) extends to "exactly 19" (4 AG-04 + 11 AG-05 + 4 AG-06). The structural pin must be on `descriptor.Placement`, not on a name prefix.
- **S1 pattern**: write `Terminal: false` explicitly in the descriptor rows. The `Terminal` field is now genuinely read by the engine (AG-04 `c203f25c` was the fix; AG-05's restated procedure is the guard).
- **S2 pattern**: reflection pin should verify field type, not just field name. AG-06 will likely have its own ordinal-like fields; pin both name and type.
- **W7 lesson (inherited from AG-04)**: document the `CardinalityAtMostOne` reserved seam at `event_descriptor.go` (referenced by `S-AEV-112`). AG-06 may use it.
- **W9 lesson (inherited from AG-04)**: state scenario count identically across proposal, tasks, and apply-progress. AG-06's 4 charter scenarios → ~12–18 spec scenarios after expansion; state it identically in those three artifacts.

### For AG-07 (Loop-Level Emission)

- **Inherit the registry unchanged**: AG-07's loop will emit events through constructors registered by AG-04 (run/turn) and AG-05 (message/tool). No new kinds expected.
- **Reconstruction property**: AG-07's emitted streams MUST satisfy `S-AMT-070`-style independence. The reconstruction helper in `reconstruction_test.go` is the reference; use it for the loop's own test.

### For AG-09 (Scheduler)

- **Tool call ordinal is payload-side**: AG-09's scheduler correlates tool events by `payload.ordinal`, not by envelope field. The kind-segregation invariant (`S-AMT-002`-style) is the loader for any ordinal lookup.

### For AG-11 (Loop-Level Typed-Error Path)

- **AG-04.3's typed-failure surface is reused**: AG-05.2's `ToolEndExecutionFailure` uses `R-AEV-008`'s `agent.Failure` wrap. AG-11's loop-level typed-error emission path is the next milestone to plug in.

### For AG-19 (Delegation Semantics)

- **Inherit the parent identifier field**: it exists (AG-04.1) and is readable. Delegation mechanics are AG-19's scope. AG-05's tool events don't carry parent semantics; AG-19 may add them.

### For AG-23 (Layer 3 Readiness Kit)

- **Import and use the registry wholesale**: `agent.MessageStartText`, `agent.ToolStart`, etc. are production-exported from `backend/agent/src/agent`. AG-23's readiness kit may construct any of the 15 kinds directly.

---

## Rollback and Recovery

**Nothing consumes the 11 new kinds yet.** AG-06 is the next to register; AG-07 may begin emitting. No schema, no migration, no `go.mod`/`go.sum` change. Rollback is mechanical:

1. Delete the files added by AG-05:
   - `message_text.go`, `message_reasoning.go`, `message_text_events.go`, `message_reasoning_events.go`, `tool_event.go`, `tool_events.go`
   - `reconstruction_test.go` (test-only)
   - `message_text_test.go`, `tool_event_test.go` (test-only helpers / scenarios)
2. Revert `event_registry_test.go` to AG-04's witness-table state (10 entries, plus scope-fence "exactly 4").
3. Revert `doc_contract_guard_test.go` to AG-04's iterator state (`expectedLayer2ContractRows` or whatever the AG-04 closed state recorded).
4. Revert `event.go` to AG-04's `EventKind` const block + `eventRegistry` table (4 AG-04 rows).
5. Revert `event_descriptor.go` six-step procedure to AG-04's closed state (drop the `5a. Terminal: false explicit` step).
6. Revert `agent_test_helpers_test.go` to AG-04's helper set (no AG-05 helper signatures).
7. Revert `doc.go` to AG-04's prose (no L2C-05 row).
8. Revert the AG-05 merge commit if merge already closed; otherwise just don't merge.
9. Revert the AG-05 archive in `openspec/changes/archive/2026-08-12-cachicamas-agent-message-tool-events/` (delete the folder — the archive is an audit trail; deletion is permitted only on rollback).
10. Revert the AG-05 amendment to `openspec/specs/agent-event-envelope/spec.md` (the in-place merge of R-AEV-007, R-AEV-010, R-AEV-012; the header note; the contradiction #5 closure note).
11. Delete `openspec/specs/agent-message-tool-events/` (the new live capability).

**Forward-fix preference** (per ADR 0005 § Guard A): If AG-05's design proves wrong at AG-06 or later, amend the mechanism in that milestone's change with its own justification. Never delete a rule or invariant to unblock a dependency. The reconstruction property and the kinds' kind-level segregation are foundational; precision over speed.

---

## Acceptance Completion Checklist

Per tasks.md Acceptance criteria (per the 8-item `Acceptance criteria` section in the new spec, restated here with close evidence):

- [x] 1. Every `S-AMT-001`…`S-AMT-081` and `S-AEV-110`…`S-AEV-112` has recorded evidence (verify-report.md Spec Compliance Matrix; tasks.md Phases 1–3 REDs/GREENs; all 4 bites recorded RED before properties GREEN)
- [x] 2. `cd backend/agent && make test`, `make lint`, `make build`, and `make vuln-check` are all green (verify-report.md Build & Tests Execution; 1,090 PASS, 0 FAIL, 0 DATA RACE; `make vuln-check` ran explicitly per Engram `obs #2944`)
- [x] 3. `backend/agent/go.mod` and `go.sum` byte-unchanged (verify-report.md AG-05 bet table; `git diff main` empty)
- [x] 4. The every-kind-constructible guard constructs all 15 kinds (4 + 11); scope-fence bites on a 16th (`S-AMT-081` RED-recorded; scratch absent from merged diff)
- [x] 5. AG-05.3 reconstruction property test GREEN with helper bites recorded RED (`S-AMT-070` GREEN; `S-AMT-071` + `S-AMT-072` RED first)
- [x] 6. The 7 charter Gherkin scenarios (`0003:543-583`, `0003:594-597`) are covered; none reduced (verified in Spec's `Coverage` table)
- [x] 7. AG-03's two boundary guards pass with zero changes to their own logic (`import_boundary_test.go`, `ambient_authority_test.go` 8/8 PASS)
- [x] 8. `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go` byte-unchanged (verify-report.md AG-05 bet table; only `event_descriptor.go` had a doc-only change, no struct edit)

**All 8 acceptance criteria satisfied.**

---

## Archive Workflow Completion

**Phase 6 archive tasks** (these are archive-scoped, not in tasks.md):
- [x] 6.1 — Apply `agent-event-envelope` delta to `openspec/specs/agent-event-envelope/spec.md` in place, with "Amended 2026-08-12 (AG-05)" header note. R-AEV-007 and R-AEV-010 MODIFIED; R-AEV-012 ADDED; traceability table row 12 added; contradiction #5 closed; acceptance criteria #1 and #9 updated. ✓ Completed at archive time.
- [x] 6.2 — Verify `openspec/changes/cachicamas-agent-message-tool-events/specs/agent-message-tool-events/spec.md` was promoted verbatim as `openspec/specs/agent-message-tool-events/spec.md` at sdd-spec time. ✓ Confirmed; content matches the spec observation (Engram #2949) byte-for-byte. SHA-256 confirmed.

**No stale implementation tasks remain unchecked** — Phase 6's archive tasks are now complete; tasks.md's 42 implementation tasks are all `[x]`.

---

## Verification Gate Status

No blocking items. Archive can proceed:

- ✓ No CRITICAL findings in verify-report
- ✓ Task completion gate passed (42/42 Phases 1–3; archive is a separate phase, not in tasks.md)
- ✓ Native review receipt gate not applicable (delivery disabled/unmanaged; kill switch off per session context)
- ✓ Spec promotion conflict resolution: no conflict (new spec at new path; delta spec merges cleanly into existing requirements section)

---

## Archive Metrics

| Metric | Value |
| --- | --- |
| SDD cycle duration | 2026-08-12 13:37–17:42 (4h 5m, proposal to archive close) |
| Proposal → Archive status updates | 2 (initial proposal flagged "approved"; archive reports PASS WITH WARNINGS) |
| Observation IDs recorded | 7 (proposal #2948, exploration #2946, spec #2949, design #2950, tasks #2951, apply-progress #2954, verify-report #2957) |
| Promoted capabilities | 2 (1 new + 1 amended) |
| Scenarios tested | 22 (15 `S-AMT-001..081` + 4 bites + 3 `S-AEV-110..112`) |
| Kinds registered | 11 (6 message + 5 tool) |
| Commits in branch | 3 (AG-05.1, AG-05.2, AG-05.3) |
| Make test result | GREEN — 1,090 PASS, 0 FAIL, 0 DATA RACE |
| Make lint result | GREEN — 0 issues |
| Make vuln-check result | GREEN — No vulnerabilities found |
| Files added | 6 (production) + 4 (test) = 10 |
| Files modified | 3 (event.go, event_descriptor.go, doc.go, plus test helpers / registry tests) |
| Files byte-unchanged | 5 (stream_check.go, failure.go, sequence.go, go.mod, go.sum) |

---

## SDD Cycle Closed

The `cachicamas-agent-message-tool-events` change is **complete, verified with known methodological findings, archived, and ready for downstream consumption** by AG-06, AG-07, AG-09, AG-11, AG-19, and AG-23.

Engram archive observation IDs: #2946 (exploration), #2948 (proposal), #2949 (spec), #2950 (design), #2951 (tasks), #2954 (apply-progress), #2957 (verify-report), [archive-report recorded at archive phase in Engram under topic `sdd/cachicamas-agent-message-tool-events/archive-report`].

**Next phase**: AG-06 (permission, cost, delegation, compaction families). Inherit the registry, the seven-step procedure, and the design decisions recorded above. W1's vacuous helper lesson and W2's name-prefix test lesson carry forward directly. The `CardinalityAtMostOne` seam reserved at `event_descriptor.go` is now documented (per `S-AEV-112`) and ready for AG-06's use.

**PR state**: OPEN — PR #164 awaits user merge. Cleanup of branch / worktree / main happens AFTER user merge, per AG-04's pattern (`sdd/cachicamas-agent-event-envelope/archive-report`).
