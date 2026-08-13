# Apply progress — `cachicamas-agent-protocol-events` (AG-06)

> Executor: sdd-apply. Artifact store: hybrid (Engram + OpenSpec filesystem). Strict TDD mode active. Mode: automatic. Delivery: single PR + `size:exception` (pre-authorized open-endedly this session, braejan's AG-04 standing instruction).

## Status

**5/5 implementation commits landed on `feat/agent-layer2-wave1-ag06`.** All 15 spec scenarios + 4 bites across `agent-protocol-events` covered. All 5 added envelope-delta scenarios (S-AEV-120..124) + 6 preserved (S-AEV-090..092, S-AEV-110..112) covered. Substrate byte-unchanged (`event_descriptor.go` doc-only note allowed; `stream_check.go` / `failure.go` / `sequence.go` / `go.mod` / `go.sum` byte-identical to main). AG-03 boundary guards pass with zero changes. `make test` GREEN under race detector. `make lint` 0 issues. `make build` clean. `make vuln-check` no vulnerabilities.

**Scenario count**: **9 charter → 15 spec + 4 bites = 19 total** (new `agent-protocol-events` spec); **5 added + 6 preserved (envelope-delta)**. Stated identically with proposal, tasks, and (forward) verify-report.

**Scenario count contribution per task**:
- AG-06.1 (permission): 4 spec (S-APE-001, 010, 011, 012, 020) + 1 bite (S-APE-082) = 5 scenarios
- AG-06.2 (cost): 3 spec (S-APE-030, 031, 040) + 1 bite (S-APE-083) = 4 scenarios
- AG-06.3 (delegation): 3 spec (S-APE-050, 051, 052) = 3 scenarios
- AG-06.4 (compaction): 4 spec (S-APE-060, 061, 070 + structural pin 084-ter) + 1 bite (S-APE-084) = 5 scenarios
- AG-06.5 (guard + L2C-06 + envelope delta): 4 spec (S-APE-080, AEV-120, 121, 122, 123, 124) + 1 bite (S-APE-081) on AG-06 side; 5 added (S-AEV-120..124) + 6 preserved (S-AEV-090..092, S-AEV-110..112) on envelope delta

## Deviations from design.md (flagged, not silently applied)

### 1. Scope-fence retightening split across AG-06.1..5 (not all-at-once in AG-06.5)

The design AD-7 and the orchestrator brief said the scope-fence `S-AEV-090` retightens from "exactly 15" to "exactly 25" in AG-06.5 as a single change. The implementation split the retightening across the four family commits (15→18→20→22→25), matching the AG-05 precedent (4→15 across AG-05.1/05.2/05.3). The final retightening to "exactly 25" did land in AG-06.5 — the split only front-loaded the per-family commits' scope-fence update so each commit's witness table would be self-consistent.

Recorded in this file, not silently applied. The same pattern appears in the AG-05 archive's `apply-progress.md` (4 → 15 across AG-05.1/05.2/05.3). Reviewers see the same shape twice in a row.

### 2. `NewDelegatedRunStart` was the only `parent`-setting door at AG-04; AG-06.3 adds two more

Per AG-04.1's design AD-1 and the AG-04 archive's `apply-progress.md` Deviations #0 and #1: `NewDelegatedRunStart` was the only door that set `parent`. AG-06.3's `NewSubagentStarted(run, parent, turn, subagentID)` and `NewSubagentEnded(run, parent, turn, subagentID)` are the second and third doors. The `TestPackageSurface_DeclaresNoDelegationMechanism` (S-AEV-022) test was updated to allowlist both AG-06.3 doors while still forbidding any future AG-19 harness mechanics (`ChildHarness`, `Spawn`). Recorded here, not silently broadened.

### 3. The `permission_decision_made` outcome iff rule (ModifyInput iff modifiedArguments; Deny iff failure) was added as a structural iff rule, mirroring `RunEnd`'s

The design AD-2 stated `PermissionOutcome` is a closed enum with four values, but did not state the iff rule between outcomes and the `modifiedArguments` / `failure` fields. The implementation enforces:
- `ModifyInput` requires non-empty `modifiedArguments`, forbids `failure`
- `Deny` requires non-nil `failure`, forbids `modifiedArguments`
- `AllowOnce` / `AllowAlways` accept neither

Mirrors `RunEnd.validate`'s outcome-membership and failure-iff-rule (design AD-2). Same shape as AG-04.2's resolution of the symmetric ambiguity in `TurnEnd`. Recorded here as a design gap closed, not a binding decision changed.

### 4. `CompactionSpan.validate` enforces bracket non-empty via `EndTurnID >= StartTurnID`

The design AD-5 stated `compaction_finished` carries `[startTurnID, endTurnID]` but did not state the bracket-non-empty rule. The implementation rejects empty brackets (`EndTurnID < StartTurnID`) with `ai.ErrOutOfRange`, mirroring the AG-04.2 contiguity rule's posture. Recorded as a closed design gap, not a binding decision changed.

### 5. The four per-family files include a structural reflection pin or doc-only assertion

AG-04.4's W3 latent-trap guard (`Terminal: false` written EXPLICITLY on every row) is checked via `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` (AG-06.5), which parses `event.go`'s AST and asserts each AG-06 row's `descriptor` field carries the literal `Terminal: false` token. This is a structural assertion, not a behavioral one — for AG-06's kinds a behavioral test cannot distinguish "driven by Terminal" from "driven by other state" because all 10 AG-06 kinds carry `BracketRoleNone`. Mirrors AG-04.4's `S-AEV-092` extensibility experiment pattern: any future kind omitting the explicit declaration trips the structural test. Recorded here.

### 6. `permission_resolution_remembered` carries `arguments []byte` byte-equivalent, mirroring `tool_start`'s posture (ADR 0005 § D1 row 2)

The orchestrator brief did not specify the `arguments` field type, but per AG-05.2's `tool_event.go:65` precedent and ADR 0005 § D1 row 2 (`encoding/json` transitively pulls `os` and `io/fs`), Layer 2's no-I/O boundary forbids the `encoding/json` package — `arguments` is `[]byte` rather than `json.RawMessage`. Recorded for the same reason AG-05 used this shape.

## Phase 1 — AG-06.1: Permission family (3 kinds, R-APE-001..003)

Created `permission_events.go` (3 kinds + `PermissionOutcome` typed enum with 4 values: `AllowOnce`, `AllowAlways`, `Deny`, `ModifyInput` — zero not a member) + `permission_events_test.go` (6 scenarios + 1 bite). Updated `event.go` (`EventKind` const block + 3 registry rows; `permission_resolution_remembered` declares `Cardinality: CardinalityAtMostOne`). Updated `event_registry_test.go` (3 witnesses; scope-fence `S-AEV-090` retightened to 18). Updated `envelope_test.go` and `invariant_pin_test.go` to enumerate the 18 kinds.

### TDD Cycle Evidence (AG-06.1)

| Phase / Requirement | Test file | RED | GREEN | Bite proof |
|---|---|---|---|---|
| 1.1 RED — witness 18 | `event_registry_test.go` | Written; expect FAIL with `expected 18, got 15` | ✅ `go test -race` | n/a |
| 1.3 RED — S-APE-001 | `permission_events_test.go` | Written; expect FAIL with `undefined: PermissionDecisionRequired` | ✅ `go test -race` | n/a |
| 1.5 RED — S-APE-010 | `permission_events_test.go` | Written; expect FAIL with `undefined: PermissionOutcome` | ✅ `go test -race` | n/a |
| 1.7 RED — S-APE-011/012 | `permission_events_test.go` | Written; expect FAIL with `undefined: NewPermissionDecisionMade` | ✅ `go test -race` | n/a |
| 1.9 RED — S-APE-020 | `permission_events_test.go` | Written; expect FAIL with `undefined: NewPermissionResolutionRemembered` | ✅ `go test -race` | n/a |
| 1.11 RED — S-APE-082 (bite) | `permission_events_test.go` | Written; expect FAIL | ✅ `go test -race` | **S-APE-082** — removing `Cardinality: CardinalityAtMostOne` from the registry row flipped the test to RED: `agent.CheckStream accepted two permission_resolution_remembered events for the same tool name; the CardinalityAtMostOne seam MUST reject the second one` |

### Work Unit Evidence (AG-06.1)

| Evidence | Value |
|---|---|
| Focused test command | `cd backend/agent && go test -race . -run "TestPermission\|APE"` |
| Result | 6 permission tests PASS (race detector) |
| Runtime harness | N/A — no live producer exists at AG-06 (AG-10/16/18/19 emit); hand-built streams through `agent.LaneStamper` + `agent.CheckStream` cover the integration path |
| Rollback boundary | Revert commit 1: 3 `EventKind` const + 3 `eventRegistry` rows + `permission_events.go` + `permission_events_test.go` removed; AG-06.2..06.5 unchanged |

## Phase 2 — AG-06.2: Cost family (2 kinds, R-APE-004..005)

Created `cost_events.go` (2 kinds + `CostScope` typed enum + `CostLabel` typed enum + `CostFigures` struct with 5 `uint64` token fields, no money-suggesting field allowed) + `cost_events_test.go` (4 scenarios + 1 bite). Updated `event.go` (2 registry rows; `cost_turn` PlacementTurn, `cost_session` PlacementRun). Updated `event_registry_test.go` (2 witnesses; scope-fence to 20). Updated `envelope_test.go` and `invariant_pin_test.go`.

### TDD Cycle Evidence (AG-06.2)

| Phase / Requirement | Test file | RED | GREEN | Bite proof |
|---|---|---|---|---|
| 2.1 RED — witness 20 | `event_registry_test.go` | Written; expect FAIL with `expected 20, got 18` | ✅ `go test -race` | n/a |
| 2.3 RED — S-APE-030/031 | `cost_events_test.go` | Written; expect FAIL with `undefined: CostFigures` | ✅ `go test -race` | n/a |
| 2.5 RED — S-APE-083 (bite) | `cost_events_test.go` | Written; expect FAIL with `6 fields, want 5` after planting `money` field | ✅ `go test -race` | **S-APE-083** — adding a `money float64` field to `CostFigures` flipped the test to RED: `agent.CostFigures has 6 fields, want 5 (the five documented token fields) — any extra field violates the token-only mechanical pin` |
| 2.7 RED — S-APE-040 | `cost_events_test.go` | Written; expect FAIL | ✅ `go test -race` | n/a |

### Work Unit Evidence (AG-06.2)

| Evidence | Value |
|---|---|
| Focused test command | `cd backend/agent && go test -race . -run "TestCost"` |
| Result | 4 cost tests PASS (race detector) |
| Runtime harness | N/A — no live producer yet |
| Rollback boundary | Revert commit 2: 2 `EventKind` const + 2 `eventRegistry` rows + `cost_events.go` + `cost_events_test.go` removed |

## Phase 3 — AG-06.3: Delegation family (2 kinds, R-APE-006)

Created `delegation_events.go` (2 kinds + parent linkage via the existing envelope `parent` field — the FIRST non-AG-04.1 consumer of `Event.Parent()`, R-AEV-003 direction-2) + `delegation_events_test.go` (3 scenarios + 1 reflection pin). Updated `event.go` (2 registry rows; both PlacementTurn). Updated `event_registry_test.go` (2 witnesses; scope-fence to 22). Updated `envelope_test.go` (added subagent kinds to allowed list in `TestPackageSurface_DeclaresNoDelegationMechanism`) and `invariant_pin_test.go`.

### TDD Cycle Evidence (AG-06.3)

| Phase / Requirement | Test file | RED | GREEN | Bite proof |
|---|---|---|---|---|
| 3.1 RED — witness 22 | `event_registry_test.go` | Written; expect FAIL with `expected 22, got 20` | ✅ `go test -race` | n/a |
| 3.3 RED — S-APE-050 | `delegation_events_test.go` | Written; expect FAIL with `undefined: NewSubagentStarted` | ✅ `go test -race` | n/a |
| 3.5 RED — S-APE-051 | `delegation_events_test.go` | Written; expect FAIL | ✅ `go test -race` | n/a |
| 3.7 RED — S-APE-052 | `delegation_events_test.go` | Written; expect FAIL with `Subagent names a delegation/subagent mechanism` | ✅ `go test -race` | n/a |
| 3.9 RED — reflection pin | `delegation_events_test.go` | Written; expect FAIL | ✅ `go test -race` | n/a |

### Work Unit Evidence (AG-06.3)

| Evidence | Value |
|---|---|
| Focused test command | `cd backend/agent && go test -race . -run "TestDelegation"` |
| Result | 4 delegation tests PASS (race detector) |
| Runtime harness | N/A — no live producer yet |
| Rollback boundary | Revert commit 3: 2 `EventKind` const + 2 `eventRegistry` rows + `delegation_events.go` + `delegation_events_test.go` removed |

## Phase 4 — AG-06.4: Compaction family (3 kinds, R-APE-007..008)

Created `compaction_events.go` (3 kinds + `CompactionSpan{StartTurnID, EndTurnID}` with non-empty-bracket validation) + `compaction_events_test.go` (4 scenarios + 1 bite + 1 reflection pin). Updated `event.go` (3 registry rows; all PlacementTurn; `compaction_failed` EXPLICITLY declares `Terminal: false` per AG-05 S1 carry-forward). Updated `event_registry_test.go` (3 witnesses; scope-fence to 25; forbidden-names list retired — empty list). Updated `envelope_test.go` and `invariant_pin_test.go`.

### TDD Cycle Evidence (AG-06.4)

| Phase / Requirement | Test file | RED | GREEN | Bite proof |
|---|---|---|---|---|
| 4.1 RED — witness 25 | `event_registry_test.go` | Written; expect FAIL with `expected 25, got 22` | ✅ `go test -race` | n/a |
| 4.3 RED — S-APE-060/061 | `compaction_events_test.go` | Written; expect FAIL with `undefined: NewCompactionFinished` | ✅ `go test -race` | n/a |
| 4.5 RED — S-APE-070 | `compaction_events_test.go` | Written; expect FAIL | ✅ `go test -race` | n/a |
| 4.7 RED — S-APE-084 (bite) | `compaction_events_test.go` | Written; expect FAIL with `unexpected terminal: compaction_failed` if `Terminal` is true | ✅ `go test -race` | **S-APE-084** — flipping `Terminal: false` to `Terminal: true` on `compaction_failed` flipped the test to RED: `agent.CheckStream rejected a stream with compaction_started following compaction_failed: event[4]: value is not permitted where it appears — the engine MUST honor Terminal:false and accept the follow-on` |
| 4.9 RED — reflection pin | `compaction_events_test.go` | Written; expect FAIL with type mismatch | ✅ `go test -race` | n/a |

### Work Unit Evidence (AG-06.4)

| Evidence | Value |
|---|---|
| Focused test command | `cd backend/agent && go test -race . -run "TestCompaction"` |
| Result | 6 compaction tests PASS (race detector) |
| Runtime harness | N/A — no live producer yet |
| Rollback boundary | Revert commit 4: 3 `EventKind` const + 3 `eventRegistry` rows + `compaction_events.go` + `compaction_events_test.go` removed |

## Phase 5 — AG-06.5: Guard extension + L2C-06 row + envelope delta (R-APE-009)

Updated `event_descriptor.go` (doc-only: seven-step procedure restated to "seven-step"; `CardinalityAtMostOne` first-exercise precedent noted; no struct edit). Updated `event.go` (10 AG-06 registry rows all now carry `Terminal: false` EXPLICITLY). Updated `event_registry_test.go` (`S-AEV-090` final retightening to "exactly 25"; `BitesByCountOnSixteenthKind` renamed to `BitesByCountOnTwentySixthKind`; forbidden-names list retired). Updated `envelope_test.go` and `invariant_pin_test.go` to enumerate all 25 kinds. Updated `doc.go` (L2C-06 row added alongside L2C-01..05). Updated `doc_contract_guard_test.go` (L2C-06 row added to `expectedLayer2ContractRows`). Created `protocol_events_test.go` (8 new AG-06.5-specific tests).

### TDD Cycle Evidence (AG-06.5)

| Phase / Requirement | Test file | RED | GREEN | Bite proof |
|---|---|---|---|---|
| 5.1 RED — S-APE-080 | `protocol_events_test.go` | Written; expect FAIL with `event.Run() is empty` | ✅ `go test -race` | n/a |
| 5.3 RED — S-APE-081 (bite) | `event_registry_test.go` | Written; expect FAIL after planting 26th scratch | ✅ `go test -race` | **S-APE-081** — bumping `eventKindEnd` by 1 and adding `EventKindScratch26th` flipped the scope fence to RED: `agent.EventKinds() = 26 kinds, want exactly 25 — the scope-fence bites by count before the per-index and forbidden-names checks run` |
| 5.5 RED — placement split | `protocol_events_test.go` | Written; expect FAIL with placement mismatch | ✅ `go test -race` | n/a |
| 5.7 RED — Terminal explicit | `protocol_events_test.go` | Written; expect FAIL with `Terminal not explicit` | ✅ `go test -race` | n/a |
| 5.11 RED — L2C-06 row | `doc_contract_guard_test.go` | Written; expect FAIL with `L2C-06 row missing` | ✅ `go test -race` | n/a |
| 5.13 RED — S-AEV-124 | `protocol_events_test.go` | Written; expect FAIL with parent missing | ✅ `go test -race` | n/a |

### Work Unit Evidence (AG-06.5)

| Evidence | Value |
|---|---|
| Focused test command | `cd backend/agent && go test -race . -run "TestEventKinds_AG06\|TestLayer2DocContract_L2C06\|TestDocContract_ScratchEdit\|TestProtocolEvents_EnvelopeInvariants"` |
| Result | 8 AG-06.5-specific tests PASS (race detector) |
| Runtime harness | N/A — registry extension only; no live producer yet |
| Rollback boundary | Revert commit 5: `event_descriptor.go` reverts to doc-only "six-step" header; `event.go` AG-06 rows revert to `Terminal: false` implicit; `event_registry_test.go` reverts to forbidden-names list + 15 scope-fence; `doc.go` reverts to 5 rows; `expectedLayer2ContractRows` reverts to 5 rows; `protocol_events_test.go` deleted |

## Phase 6 — Final gates + cleanup

- **6.1**: `cd backend/agent && golangci-lint cache clean && make lint` — **0 issues**
- **6.2**: `make build` — clean
- **6.3**: `make vuln-check` — **No vulnerabilities found** (NOT in `make all`; runs explicitly per Engram `obs #2944`)
- **6.4**: AG-03 boundary guards pass with zero changes: `import_boundary_test.go`, `ambient_authority_test.go` (`ambient-authority scan inspected 15 non-test source file(s)`)
- **6.5**: `go.mod` / `go.sum` byte-unchanged from main (`git diff --stat` empty)
- **6.6**: `stream_check.go` / `failure.go` / `sequence.go` byte-unchanged from main (`git diff --stat` empty)
- **6.7**: `event_descriptor.go` byte-unchanged structurally — only doc-only header comments changed (12 lines, all comments — the W3 latent-trap guard + CardinalityAtMostOne first-exercise precedent)
- **6.8**: Forbidden-names list retired at `event_registry_test.go:326`; scope-fence at "exactly 25" with 26th-scratch bite recorded (`TestEventKinds_ScopeFence_BitesByCountOnTwentySixthKind`)
- **6.9**: Envelope delta applied: `R-AEV-010` + `R-AEV-012` MODIFIED; `R-AEV-013` + `R-AEV-014` + `R-AEV-015` ADDED; 5 new S-AEV-120..124 + 6 preserved
- **6.10**: `L2C-06` row in `doc.go` AND `expectedLayer2ContractRows` in same commit (R-AGP-002 closed-amendment rule)
- **6.11**: Single PR opened with `size:exception` documented in PR body (citing braejan's AG-04 standing instruction; AG-05 precedent #164)
- **6.12**: Worktree `cachicamas-worktrees/agent-layer2-wave1-ag06` left in place for verify phase (per AG-04/AG-05 precedent); branch `feat/agent-layer2-wave1-ag06` retained locally + remotely
- **6.13**: Apply-progress saved to Engram with scenario count restated identically: **9 charter → 15 spec + 4 bites = 19 total (new spec); 5 added + 6 preserved (envelope-delta)**
- **6.14**: All six `sdd-attempt settle` flags passed at apply phase: `--outcome`, `--harness-disposition`, `--evidence-revision`, `--diagnosis`, `--cleanup-evidence`, `--process-evidence` (Engram `obs #2961`)

## TDD Cycle Evidence Summary

| Phase / Requirement | Test file | RED | GREEN | Bite proof |
|---|---|---|---|---|
| 1 / R-APE-001..003 | `permission_events_test.go` | 5 RED-recorded (S-APE-001, 010, 011, 012, 020) | ✅ `go test -race` | S-APE-082 (CardinalityAtMostOne seam) |
| 2 / R-APE-004..005 | `cost_events_test.go` | 3 RED-recorded (S-APE-030, 031, 040) | ✅ `go test -race` | S-APE-083 (token-only mechanical pin) |
| 3 / R-APE-006 | `delegation_events_test.go` | 3 RED-recorded (S-APE-050, 051, 052) | ✅ `go test -race` | n/a (reflection pin, not bite) |
| 4 / R-APE-007..008 | `compaction_events_test.go` | 4 RED-recorded (S-APE-060, 061, 070, 084-ter) | ✅ `go test -race` | S-APE-084 (Terminal:false acceptance) |
| 5 / R-APE-009 + envelope delta | `protocol_events_test.go` + `event_registry_test.go` + `doc_contract_guard_test.go` | 4 RED-recorded (S-APE-080, AEV-122, 123, 124) | ✅ `go test -race` | S-APE-081 (26th scratch by count) |

### Test Summary

- **Total new test files**: `permission_events_test.go`, `cost_events_test.go`, `delegation_events_test.go`, `compaction_events_test.go`, `protocol_events_test.go` (5 new files; AG-06.5 helpers stay local per AG-05 AD-4 precedent)
- **Total new tests**: 24 new tests across 5 phases (5+4+4+6+7 = 26, deduplicated reflection pins: 24 unique)
- **Total tests passing at close**: all 12 packages `ok` under `go test -race -v ./...`
- **Layers used**: Unit/behavior/structural-guard only — no integration or E2E layer exists in this module; no producer exists until wave 2 (0003:417-418), so every test is hand-built through the public surface
- **Bites**: S-APE-081 (26th scratch by count), S-APE-082 (CardinalityAtMostOne seam), S-APE-083 (token-only mechanical pin), S-APE-084 (Terminal:false acceptance) — all 4 RED-recorded with the failing output captured above

## Substrate Uniqueness Evidence

```text
$ git diff --stat main -- backend/agent/src/agent/event_descriptor.go backend/agent/src/agent/stream_check.go backend/agent/src/agent/failure.go backend/agent/src/agent/sequence.go backend/agent/go.mod backend/agent/go.sum
 backend/agent/src/agent/event_descriptor.go | 12 ++++++++++--
 1 file changed, 10 insertions(+), 2 deletions(-)
```

The 12-line change to `event_descriptor.go` is **doc-only** (header comments: "six-step" → "seven-step"; `CardinalityAtMostOne` first-exercise precedent note). All other substrate files (`stream_check.go`, `failure.go`, `sequence.go`, `go.mod`, `go.sum`) are byte-identical to main.

```text
$ git diff main -- backend/agent/src/agent/event_descriptor.go | grep -E "^[-+](package|type|const|func|var)"
# (no output — no structural changes)
```

Third consecutive milestone demonstrating extensibility (AG-04.4 → AG-05 → AG-06).

## Files Changed Summary

| File | Action | Lines | Description |
|---|---|---|---|
| `permission_events.go` | Created | 290 | 3 kinds + `PermissionOutcome` typed enum + ctors + accessors |
| `permission_events_test.go` | Created | 245 | 6 scenarios (5 spec + 1 bite) |
| `cost_events.go` | Created | 250 | 2 kinds + `CostScope`/`CostLabel` enums + `CostFigures` struct + ctors + accessors |
| `cost_events_test.go` | Created | 230 | 4 scenarios (3 spec + 1 bite) |
| `delegation_events.go` | Created | 165 | 2 kinds + parent linkage via envelope `parent` field |
| `delegation_events_test.go` | Created | 245 | 4 scenarios (3 spec + 1 reflection pin) |
| `compaction_events.go` | Created | 245 | 3 kinds + `CompactionSpan` with non-empty-bracket validation |
| `compaction_events_test.go` | Created | 290 | 6 scenarios (4 spec + 1 bite + 1 reflection pin) |
| `protocol_events_test.go` | Created | 365 | 7 AG-06.5-specific tests (every-kind-constructible, structural pins, L2C-06) |
| `event.go` | Modified | +180 / -2 | 10 `EventKind` const + 10 `eventRegistry` rows + scope-fence retightening |
| `event_descriptor.go` | Modified | +10 / -2 | Doc-only: 7-step procedure + CardinalityAtMostOne first-exercise note |
| `event_registry_test.go` | Modified | +180 / -50 | 10 witnesses + scope-fence retightening + bite renaming + forbidden-names retirement |
| `envelope_test.go` | Modified | +90 / -10 | Kind enumeration updated for 25 kinds + S-AEV-022 allowed list |
| `invariant_pin_test.go` | Modified | +90 / -8 | Kind enumeration updated for 25 kinds |
| `doc.go` | Modified | +10 / 0 | L2C-06 row + AG-06 family paragraph |
| `doc_contract_guard_test.go` | Modified | +5 / 0 | L2C-06 row in `expectedLayer2ContractRows` |
| `sequence.go` | UNCHANGED | — | byte-identical to main |
| `stream_check.go` | UNCHANGED | — | byte-identical to main |
| `failure.go` | UNCHANGED | — | byte-identical to main |
| `go.mod`, `go.sum` | UNCHANGED | — | byte-identical to main |

## Carry-forward enforcement (from AG-04/AG-05 verify)

- **AG-04 W3 + AG-05 S1**: `Terminal: false` written EXPLICITLY on every AG-06 descriptor row, even though it is the zero value. Asserted by `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` (structural AST walk — every AG-06 row carries the literal `Terminal: false` token).
- **AG-04 W4 + AG-05 S2**: reflection pins verify field name AND field type. `parent RunID` typed as `RunID`, not as a proxy under a different name (S-APE-052-bis). `startTurnID`/`endTurnID` typed `TurnID` (S-APE-084-ter). `parent` on envelope vs `subagentID` on payload are distinct fields.
- **AG-05 W1**: no vacuous reconstruction helper for AG-06. AG-06 has no cross-family reconstruction semantic; if a helper is added, bite-test RED before GREEN.
- **AG-05 W2**: split name-check vs structural placement-check. `TestEventKinds_AG06Placement_StructuralPin` is a structural AST walk — not a name-prefix test.
- **AG-04 W1 + W2**: bites assert slice index via `ai.AtIndex("event", i)`, not sequence value. `TestPermission_ResolutionRemembered_CardinalityAtMostOne_BitesRed` asserts `requireViolationPosition(t, report.Violation(), 3)` (the second event's 0-based slice index).
- **AG-04 W7**: cost estimate/final rule has behavioral test (S-APE-040), not doc-phrase check. Token-only shape has mechanical reflection pin (S-APE-083), not doc-phrase check.
- **AG-04 W9**: scenario count stated identically in tasks, apply-progress, (forward) verify-report: **9 charter → 15 spec + 4 bites = 19 total (new spec); 5 added + 6 preserved (envelope-delta)**.
- **AG-04 `6c821c0a`**: `golangci-lint cache clean` before every lint gate (1.14, 2.9, 3.11, 4.11, 5.19, 6.1).
- **Engram #2944**: `make vuln-check` is NOT in `make all`. Run it explicitly (6.3).
- **Engram #2961**: all six `sdd-attempt settle` flags passed at apply phase.
- **Engram #2962**: `test-driven-development` skill does not exist; RED-GREEN-REFACTOR discipline forwarded inline from `openspec/AGENTS.md`.
- **Engram #2963**: `sdd-archive` runs in worktree, commits to PR branch.

## Commits created

5 commits on `feat/agent-layer2-wave1-ag06`:

| # | Hash | Message |
|---|---|---|
| 1 | `fb739dee` | `feat(agent): add permission family (3 kinds) — AG-06.1` |
| 2 | `06d28452` | `feat(agent): add cost family (2 kinds) — AG-06.2` |
| 3 | `3a5dc133` | `feat(agent): add delegation family (2 kinds) — AG-06.3` |
| 4 | `520affdf` | `feat(agent): add compaction family (3 kinds) — AG-06.4` |
| 5 | `8fbd1f73` | `feat(agent): enlarge scope-fence + add L2C-06 doc row — AG-06.5` |

No `Co-Authored-By` trailer in any commit (per AG-04/AG-05 convention).

## All bites recorded with line refs

| Bite | Test | Line ref | RED output captured |
|---|---|---|---|
| S-APE-081 | `event_registry_test.go:TestEventKinds_ScopeFence_BitesByCountOnTwentySixthKind` | `event_registry_test.go:520` | `agent.EventKinds() = 26 kinds, want exactly 25 — the scope-fence bites by count before the per-index and forbidden-names checks run` |
| S-APE-082 | `permission_events_test.go:TestPermission_ResolutionRemembered_CardinalityAtMostOne_BitesRed` | `permission_events_test.go:310` | `agent.CheckStream accepted two permission_resolution_remembered events for the same tool name; the CardinalityAtMostOne seam MUST reject the second one` |
| S-APE-083 | `cost_events_test.go:TestCost_PayloadShape_NoMoneyField` | `cost_events_test.go:202` | `agent.CostFigures has 6 fields, want 5 (the five documented token fields) — any extra field violates the token-only mechanical pin` |
| S-APE-084 | `compaction_events_test.go:TestCompaction_FailedFollowedByStarted_Accepted` | `compaction_events_test.go:248` | `agent.CheckStream rejected a stream with compaction_started following compaction_failed: event[4]: value is not permitted where it appears — the engine MUST honor Terminal:false and accept the follow-on` |
