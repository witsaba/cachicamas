# Tasks — AG-06 — Add permission, cost, delegation and compaction event families

## Identity
- **Change**: `cachicamas-agent-protocol-events`
- **Milestone**: AG-06 (Layer 2 Wave 1, **closing**) — doc 0003:602–712
- **Branch**: `feat/agent-layer2-wave1-ag06`
- **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave1-ag06`
- **Store**: hybrid (Engram + OpenSpec filesystem)
- **Mode**: automatic
- **Strict TDD**: enabled (per `openspec/AGENTS.md`; `apply.tdd: true` in `openspec/config.yaml`)
- **Spec prefix**: `R-APE-`/`S-APE-`; envelope delta uses existing `R-AEV-`/`S-AEV-`

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 1500–2400 (backend/agent Go insertions + openspec planning) |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR with 5 internal commits (AG-06.1 → AG-06.5) |
| Delivery strategy | `single-pr` |
| Chain strategy | `size-exception` |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Rationale: `size:exception` pre-authorized open-endedly this session (braejan's AG-04 standing instruction, Engram #2957/2958). No AG-05.3-style cross-family property test in AG-06 — the four families are independent — but single PR matches AG-04/AG-05 precedent. Substrate byte-unchanged (event_descriptor.go / stream_check.go / failure.go / sequence.go / go.mod / go.sum).

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 (AG-06.1) | Permission family — 3 kinds, `PermissionOutcome` enum (4 typed values), `resolution_remembered` declares `CardinalityAtMostOne` (first AG-06 exercise of AG-04.3 seam at `event_descriptor.go:103-120`) | Same PR, commit 1 | `cd backend/agent && make test` (`-run APE-001\|APE-002\|APE-003\|APE-082`) | N/A — no live producer yet | Revert commit 1: 3 `EventKind` const + `eventRegistry` rows + `permission_events.go` + `permission_events_test.go` removed; AG-06.2..06.5 unchanged |
| 2 (AG-06.2) | Cost family — 2 kinds (`cost_turn` PlacementTurn, `cost_session` PlacementRun), 5 token fields, `CostLabel` (Estimate/Final) discriminator; byte-level structural pin on token-only shape (no money field) | Same PR, commit 2 | `cd backend/agent && make test` (`-run APE-004\|APE-005\|APE-083`) | N/A — no live producer yet | Revert commit 2: 2 `EventKind` const + `eventRegistry` rows + `cost_events.go` + `cost_events_test.go` removed |
| 3 (AG-06.3) | Delegation family — 2 kinds (`subagent_started`, `subagent_ended`) set `Parent()` on envelope; **first non-`NewDelegatedRunStart` consumer** of parent identifier (`event.go:362-366`); reflection pin verifies field name AND type (AG-05 S2 lesson) | Same PR, commit 3 | `cd backend/agent && make test` (`-run APE-006`) | N/A — no live producer yet | Revert commit 3: 2 `EventKind` const + `eventRegistry` rows + `delegation_events.go` + `delegation_events_test.go` removed |
| 4 (AG-06.4) | Compaction family — 3 kinds (`compaction_started`, `compaction_finished` with `[startTurnID, endTurnID]` bracket + `summaryID`, `compaction_failed` `Terminal:false` carries typed `*Failure`); recovery-after-failure bite confirms engine honors `Terminal:false` | Same PR, commit 4 | `cd backend/agent && make test` (`-run APE-007\|APE-008\|APE-084`) | N/A — no live producer yet | Revert commit 4: 3 `EventKind` const + `eventRegistry` rows + `compaction_events.go` + `compaction_events_test.go` removed |
| 5 (AG-06.5) | Guard extension — every-kind-constructible guard iterates all 25 registered kinds; scope-fence `S-AEV-090` retightens 15 → 25; `BitesByCountOnSixteenthKind` → `BitesByCountOnTwentySixthKind`; forbidden-names list at `event_registry_test.go:326` retires; `L2C-06` row added to `doc.go` + `doc_contract_guard_test.go`; envelope delta `R-AEV-010` + `R-AEV-012` MODIFIED, `R-AEV-013/014/015` ADDED (split name-check vs placement-check per AG-05 W2) | Same PR, commit 5 | `cd backend/agent && make test` (`-run APE-009\|APE-081\|AEV-120\|AEV-121\|AEV-122\|AEV-123\|AEV-124`) + `golangci-lint cache clean && make lint` + `make build` + `make vuln-check` | N/A — registry extension only; no live producer yet | Revert commit 5: witness table reverts to 15; scope-fence reverts to "exactly 15"; forbidden-names list restored; L2C-06 row removed |

## Scenario count commitment (W9 lesson from AG-04/AG-05)

**9 charter Gherkin scenarios in 5 leaves → 15 spec + 4 bites = 19 total** (on new `agent-protocol-events` spec). **5 added + 6 preserved (envelope-delta scenarios across MODIFIED blocks)** on the envelope delta. This count MUST be restated identically in `apply-progress.md` and `verify-report.md`. AG-04's W9 finding was a scenario-count error that propagated three artifacts deep before verify caught it.

Bite distribution: S-APE-081 (26th scratch fails by count, AG-06.5); S-APE-082 (per-tool `CardinalityAtMostOne` fires, AG-06.1); S-APE-083 (scratch with `money *decimal` → diff FAIL, AG-06.2); S-APE-084 (compaction_started after compaction_failed accepted, AG-06.4).

## Phase 1: AG-06.1 — Permission family (3 kinds, R-APE-001..003)

- [x] 1.1 RED: extend `event_registry_test.go` witness-table to assert exactly 18 kinds (4 AG-04 + 11 AG-05 + 3 AG-06.1); run, expect FAIL with `expected 18, got 15`
- [x] 1.2 GREEN: create `permission_events.go` with `PermissionOutcome` typed enum (`PermissionOutcomeAllowOnce/AllowAlways/Deny/ModifyInput`, zero not a member), 3 payloads (`PermissionDecisionRequired{toolCallID, name, arguments []byte}`, `PermissionDecisionMade{callID, outcome, modifiedArguments []byte, failure *Failure}`, `PermissionResolutionRemembered{toolName, outcome}`), 3 ctors, 3 `EventDescriptor` rows (`Terminal: false` explicit on every row per AG-05 S1); witness passes at 18
- [x] 1.3 RED: write `permission_events_test.go` `TestPermission_DecisionRequired_ReadsCallIdentityToolNameArguments` (S-APE-001); run, expect FAIL with `undefined: PermissionDecisionRequired`
- [x] 1.4 GREEN: implement `NewPermissionDecisionRequired(run, turn, callID, name, args) (Event, error)` with `[]byte` argument (ADR 0005 § D1 row 2 — encoding/json transitively imports forbidden packages); S-APE-001 passes
- [x] 1.5 RED: write `TestPermission_DecisionMade_ReachesAllFourTypedOutcomes` (S-APE-010); run, expect FAIL
- [x] 1.6 GREEN: implement `NewPermissionDecisionMade(run, turn, callID, outcome, ...)` enforcing closed enum (zero not a member); S-APE-010 passes
- [x] 1.7 RED: write `TestPermission_DecisionMade_ModifyInputCarriesModifiedArguments` (S-APE-011) + `TestPermission_DecisionMade_DenyCarriesTypedFailure` (S-APE-012); run, expect FAIL
- [x] 1.8 GREEN: extend `PermissionDecisionMade` payload with `modifiedArguments []byte` + `failure *Failure`; S-APE-011 + S-APE-012 pass (satisfies R-APE-002)
- [x] 1.9 RED — bite: write `TestPermission_ResolutionRemembered_DistinctFromDecisionMade` (S-APE-020); run, expect FAIL with `kind collision`
- [x] 1.10 GREEN: implement `NewPermissionResolutionRemembered(run, name, outcome)`; S-APE-020 passes (satisfies R-APE-003)
- [x] 1.11 RED — bite: hand-build stream with two `permission_resolution_remembered` for same tool name; write `TestPermission_ResolutionRemembered_CardinalityAtMostOne_BitesRed` (S-APE-082) asserting REJECTED with `CardinalityAtMostOne` rule and second offending position via `ai.AtIndex("event", i)` (AG-04 W2 carry-forward); run, expect FAIL
- [x] 1.12 RED: write `TestPermission_DescriptorRow_ResolutionRememberedDeclaresCardinalityAtMostOne` (split name-check vs placement-check per AG-05 W2); run, expect FAIL
- [x] 1.13 GREEN: set `Cardinality: CardinalityAtMostOne` on the `permission_resolution_remembered` row; S-APE-082 bites RED; S-APE-020 descriptor assertion passes
- [x] 1.14 `make test` clean (race detector); `golangci-lint cache clean && make lint` clean (per AG-04 `6c821c0a` precedent)

**Scenario count contribution**: 4 spec scenarios (S-APE-001, 010, 011, 012, 020) + 1 bite (S-APE-082) = 5 scenarios total in this phase.

## Phase 2: AG-06.2 — Cost family (2 kinds, R-APE-004..005)

- [x] 2.1 RED: extend witness-table to assert exactly 20 kinds (4 + 11 + 5 AG-06.1+2); run, expect FAIL with `expected 20, got 18`
- [x] 2.2 GREEN: create `cost_events.go` with `CostScope` (`CostScopeTurn`, `CostScopeSession`, zero not a member) + `CostLabel` (`CostLabelEstimate`, `CostLabelFinal`, zero not a member) typed enums; `Cost` payload with 5 `uint64` fields (`inputTokens`, `outputTokens`, `cacheReadTokens`, `cacheWriteTokens`, `reasoningTokens`); 2 ctors `NewCostTurn(run, turn, label, figures)` and `NewCostSession(run, label, figures)`; 2 `EventDescriptor` rows (`cost_turn` PlacementTurn, `cost_session` PlacementRun); `Terminal: false` explicit; witness at 20
- [x] 2.3 RED: write `cost_events_test.go` `TestCost_Turn_HasFiveTokenFields` (S-APE-030) + `TestCost_Session_PlacementIsRun` (S-APE-031); run, expect FAIL
- [x] 2.4 GREEN: enforce `PlacementRun` on `cost_session` row via separate ctor; S-APE-030 + S-APE-031 pass (satisfies R-APE-004)
- [x] 2.5 RED — bite: hand-build scratch payload adding `money *decimal` field; write `TestCost_PayloadShapeDiff_NoMoneyField` (S-APE-083) parsing via reflection and asserting diff FAILS naming unknown field; run, expect FAIL with `unknown field money`; revert scratch
- [x] 2.6 GREEN: assert structural shape via reflection in `cost_events_test.go` (mechanical pin, not doc-phrase per AG-04 W7 carry-forward inverted); S-APE-083 bites RED
- [x] 2.7 RED: write `TestCost_EveryFigureLabelledEstimateOrFinal` (S-APE-040) constructing pre-final + post-final cost events; run, expect FAIL
- [x] 2.8 GREEN: enforce `CostLabel` typed enum (zero not a member) in ctors; S-APE-040 passes (satisfies R-APE-005)
- [x] 2.9 `make test` clean; `golangci-lint cache clean && make lint` clean

**Scenario count contribution**: 3 spec scenarios (S-APE-030, 031, 040) + 1 bite (S-APE-083) = 4 scenarios total in this phase.

## Phase 3: AG-06.3 — Delegation family (2 kinds, R-APE-006)

- [x] 3.1 RED: extend witness-table to assert exactly 22 kinds (4 + 11 + 7 AG-06.1..3); run, expect FAIL with `expected 22, got 20`
- [x] 3.2 GREEN: create `delegation_events.go` with `SubagentStarted{subagentID string}` + `SubagentEnded{subagentID string}` payloads; 2 ctors `NewSubagentStarted(run, parent, subagentID)` and `NewSubagentEnded(run, parent, subagentID)` (set envelope `parent` + `hasParent` via internal helper, mirroring `NewDelegatedRunStart` at `run_events.go:60-76`); 2 `EventDescriptor` rows PlacementTurn; `Terminal: false` explicit; witness at 22
- [x] 3.3 RED: write `delegation_events_test.go` `TestDelegation_SubagentStarted_ParentIsReadable` (S-APE-050); run, expect FAIL
- [x] 3.4 GREEN: helper `delegationEventParent(run RunID) (RunID, bool)` invoked inside ctors; S-APE-050 passes
- [x] 3.5 RED: write `TestDelegation_SubagentStartedAndEnded_ShareParent` (S-APE-051) asserting both return same `(parentID, true)`; run, expect FAIL
- [x] 3.6 GREEN: helper invoked consistently; S-APE-051 passes
- [x] 3.7 RED: write `TestDelegation_NonDelegatedEvent_HasNoParent` (S-APE-052) asserting AG-04/AG-05 events return `(RunID(""), false)` (R-AEV-003 direction-2, AG-04 W4 carry-forward); run, expect FAIL
- [x] 3.8 GREEN: parent field exists from AG-04.1 — only delegation ctors set it; S-APE-052 passes (satisfies R-APE-006)
- [x] 3.9 RED: write `TestDelegation_ParentField_ReflectionPin_NameAndType` verifying `parent` field is named "parent" AND typed `RunID` (AG-05 S2 carry-forward — not proxy under different name); run, expect FAIL with type mismatch
- [x] 3.10 GREEN: ensure `Event.Parent() (RunID, bool)` signature preserved; pin passes
- [x] 3.11 `make test` clean; `golangci-lint cache clean && make lint` clean

**Scenario count contribution**: 3 spec scenarios (S-APE-050, 051, 052) = 3 scenarios total in this phase.

## Phase 4: AG-06.4 — Compaction family (3 kinds, R-APE-007..008)

- [x] 4.1 RED: extend witness-table to assert exactly 25 kinds (4 + 11 + 10 AG-06.1..4); run, expect FAIL with `expected 25, got 22`
- [x] 4.2 GREEN: create `compaction_events.go` with `CompactionSpan{StartTurnID TurnID, EndTurnID TurnID}`; 3 payloads (`CompactionStarted{compactionID string}`, `CompactionFinished{compactionID string, span CompactionSpan, summaryID string}`, `CompactionFailed{compactionID string, failure *Failure}`); 3 ctors `NewCompactionStarted(run, compactionID)`, `NewCompactionFinished(run, compactionID, span, summaryID)`, `NewCompactionFailed(run, compactionID, failure)`; 3 `EventDescriptor` rows PlacementTurn; `Terminal: false` explicit on every row; witness at 25
- [x] 4.3 RED: write `compaction_events_test.go` `TestCompaction_Finished_CarriesTurnBracketAndSummaryID` (S-APE-060 + S-APE-061); run, expect FAIL
- [x] 4.4 GREEN: enforce span bracket non-empty (`EndTurnID >= StartTurnID`); S-APE-060 + S-APE-061 pass (satisfies R-APE-007)
- [x] 4.5 RED: write `TestCompaction_Failed_DistinctFromFinished_TerminalFalse` (S-APE-070) asserting distinct kind + `Terminal: false` declaration + typed `*Failure` reachable; run, expect FAIL
- [x] 4.6 GREEN: enforce `Terminal: false` explicitly on `compaction_failed` row (AG-05 S1 carry-forward — do NOT rely on zero value); S-APE-070 passes (satisfies R-APE-008)
- [x] 4.7 RED — bite: hand-build stream where `compaction_started` follows `compaction_failed`; write `TestCompaction_FailedFollowedByStarted_Accepted` (S-APE-084) asserting validator ACCEPTS via `CheckStream`; run, expect FAIL with `unexpected terminal: compaction_failed`
- [x] 4.8 GREEN: `stream_check.go:173-175` honors explicit `Terminal: false` (post-AG-04 fix `c203f25c`); S-APE-084 bites RED before GREEN (engine accepts per explicit declaration); revert scratch stream
- [x] 4.9 RED: write `TestCompaction_SpanField_ReflectionPin_NameAndType` verifying `startTurnID` / `endTurnID` named AND typed `TurnID` (AG-05 S2 carry-forward); run, expect FAIL with type mismatch
- [x] 4.10 GREEN: ensure payload field types preserved; pin passes
- [x] 4.11 `make test` clean; `golangci-lint cache clean && make lint` clean

**Scenario count contribution**: 3 spec scenarios (S-APE-060, 061, 070) + 1 bite (S-APE-084) = 4 scenarios total in this phase.

## Phase 5: AG-06.5 — Guard extension + L2C-06 row + envelope delta (R-APE-009)

- [x] 5.1 RED: write `event_registry_test.go` `TestEventKinds_EveryKindConstructible_Iterates25` (S-APE-080) constructing one of each registered kind through public surface from `agent_test` external package; run, expect FAIL with `expected 25, got 25` (count is correct, but identity fields assertion on AG-06 kinds fails first)
- [x] 5.2 GREEN: ensure all 10 AG-06 ctors return `Event` with readable identity fields (`Run()`, `Turn()`, `Parent()`); S-APE-080 passes
- [x] 5.3 RED — bite: plant 26th scratch kind following seven-step procedure in scratch test file; write `TestEventKinds_ScopeFence_BitesByCountOnTwentySixthKind` (S-APE-081) asserting guard FAILS by count before name scan runs; run, expect FAIL
- [x] 5.4 GREEN: update `S-AEV-090` scope-fence in `event_registry_test.go` from "exactly 15" to "exactly 25" (same commit); rename `BitesByCountOnSixteenthKind` → `BitesByCountOnTwentySixthKind`; S-APE-081 bites RED; revert scratch file (NOT in merged diff per AG-05 precedent)
- [x] 5.5 RED: write `TestEventKinds_AG06RegisterPlacementTurn_NameCheck` (name only — split per AG-05 W2) + `TestEventKinds_AG06PlacementTurn_StructuralPin` (asserts `descriptor.Placement == PlacementTurn` via registry inspection, not name); run, expect FAIL
- [x] 5.6 GREEN: both tests pass; structural pin does not rely on name prefix
- [x] 5.7 RED: write `TestEventKinds_TerminalFalseExplicitOnAG06Rows` asserting each AG-06 row has explicit `Terminal: false` (AG-05 S1 carry-forward — not zero-value); run, expect FAIL with `compaction_failed row Terminal not explicit`
- [x] 5.8 GREEN: add `Terminal: false` explicit to all 10 AG-06 rows; test passes
- [x] 5.9 RED: write `TestForbiddenNames_Retired` asserting forbidden-names list at `event_registry_test.go:326` is empty/absent (per R-AEV-010 MODIFIED, R-AEV-013); run, expect FAIL
- [x] 5.10 GREEN: retire forbidden-names list; test passes (R-AEV-013 S-AEV-121)
- [x] 5.11 RED: extend `expectedLayer2ContractRows` table with `L2C-06` row text referencing 4 protocol families; write `TestLayer2DocContract_L2C06_ReferencesProtocolFamilies` (S-AEV-122); run, expect FAIL with `L2C-06 row missing`
- [x] 5.12 GREEN: add `L2C-06` row to `doc.go` prose AND `expectedLayer2ContractRows` in `doc_contract_guard_test.go` in same commit (per R-AGP-002 closed-amendment rule); S-AEV-122 passes; also satisfies S-AEV-123 scratch (revert scratch)
- [x] 5.13 RED: write `TestProtocolEvents_EnvelopeInvariantsCompliant` (S-AEV-124) asserting all 10 AG-06 kinds have readable identity fields, `subagent_*` report `(parentID, true)`, `resolution_remembered` declares `CardinalityAtMostOne`, `compaction_failed` declares `Terminal: false`; run, expect FAIL
- [x] 5.14 GREEN: all assertions hold via reflection pin; S-AEV-124 passes
- [x] 5.15 update `event.go` `EventKind` const block + `eventRegistry` table (the 10 new rows total from AG-06.1..4 commits; this commit may be a no-op if events go module lands them)
- [x] 5.16 update `event_descriptor.go` seven-step procedure doc note to record `CardinalityAtMostOne` first-exercise precedent (doc-only, no struct edit per AG-05 S1 carry-forward)
- [x] 5.17 update `doc.go` to carry `L2C-06` prose alongside `L2C-01..05`
- [x] 5.18 verify `agent_test_helpers_test.go` reusable; AG-06 helpers stay local to per-family `_test.go` files (AG-05 AD-4 precedent — helpers in test file that uses them)
- [x] 5.19 `make test` clean (race detector); `golangci-lint cache clean && make lint` clean

**Scenario count contribution**: 2 spec scenarios (S-APE-080, AEV-122, AEV-123, AEV-124) + 1 bite (S-APE-081) on AG-06 side; 5 added (S-AEV-120..124) + 6 preserved (S-AEV-090..092, S-AEV-110..112) on envelope delta = 8 net scenarios in this phase.

## Phase 6: Final gates + cleanup

- [x] 6.1 `golangci-lint cache clean && make lint` (full pass; cite `6c821c0a` precedent)
- [x] 6.2 `make build` clean
- [x] 6.3 `make vuln-check` clean (NOT in `make all`; runs explicitly per Engram `obs #2944`)
- [x] 6.4 AG-03 boundary guards pass with zero changes: `import_boundary_test.go`, `ambient_authority_test.go`
- [x] 6.5 `go.mod` / `go.sum` byte-unchanged from main (`git diff main -- backend/agent/go.mod go.sum` returns empty)
- [x] 6.6 `stream_check.go` / `failure.go` / `sequence.go` byte-unchanged (the AG-06 bet — third consecutive milestone demonstrating extensibility)
- [x] 6.7 `event_descriptor.go` byte-identical EXCEPT for doc-only procedure note (no struct edit per AG-05 S1 carry-forward)
- [x] 6.8 forbidden-names list retired at `event_registry_test.go:326`; scope-fence at "exactly 25" with 26th-scratch bite recorded
- [x] 6.9 envelope delta applied: `R-AEV-010` + `R-AEV-012` MODIFIED; `R-AEV-013` + `R-AEV-014` + `R-AEV-015` ADDED; 5 new S-AEV-120..124 + 6 preserved
- [x] 6.10 `L2C-06` row in `doc.go` AND `expectedLayer2ContractRows` in same commit (R-AGP-002 closed-amendment)
- [x] 6.11 Single PR opened with `size:exception` documented in PR body (citing braejan's AG-04 standing instruction; AG-05 precedent #164)
- [x] 6.12 Worktree `cachicamas-worktrees/agent-layer2-wave1-ag06` left in place for verify phase (per AG-04/AG-05 precedent); branch `feat/agent-layer2-wave1-ag06` retained locally + remotely
- [x] 6.13 Apply-progress saved to Engram with scenario count restated identically: **9 charter → 15 spec + 4 bites = 19 total (new spec); 5 added + 6 preserved (envelope-delta)**
- [x] 6.14 All six `sdd-attempt settle` flags passed at apply phase: `--outcome`, `--harness-disposition`, `--evidence-revision`, `--diagnosis`, `--cleanup-evidence`, `--process-evidence` (Engram `obs #2961`)

## Commit Graph (work-unit-commits skill)

5 commits, one per node:

| Commit | Conventional message | Scope | Files |
|---|---|---|---|
| 1 | `feat(agent): add permission family (3 kinds, CardinalityAtMostOne) — AG-06.1` | Permission family | `permission_events.go` + `permission_events_test.go` + 3 `eventRegistry` rows + 3 `EventKind` const |
| 2 | `feat(agent): add cost family (2 kinds, token-only) — AG-06.2` | Cost family | `cost_events.go` + `cost_events_test.go` + 2 `eventRegistry` rows + 2 `EventKind` const |
| 3 | `feat(agent): add delegation family (2 kinds, parent-linked) — AG-06.3` | Delegation family | `delegation_events.go` + `delegation_events_test.go` + 2 `eventRegistry` rows + 2 `EventKind` const |
| 4 | `feat(agent): add compaction family (3 kinds, recovery-distinct) — AG-06.4` | Compaction family | `compaction_events.go` + `compaction_events_test.go` + 3 `eventRegistry` rows + 3 `EventKind` const |
| 5 | `feat(agent): enlarge scope-fence + L2C-06 row + envelope delta — AG-06.5` | Guard + envelope delta + L2C-06 | `event.go` + `event_registry_test.go` + `doc.go` + `doc_contract_guard_test.go` + `event_descriptor.go` (doc-only) |

## Carry-forward enforcement (from AG-04/AG-05 verify)

- **AG-04 W3 + AG-05 S1**: `Terminal: false` written EXPLICITLY on every AG-06 descriptor row, even though it is the zero value. Reviewers should see intent at a glance.
- **AG-04 W4 + AG-05 S2**: reflection pins verify field name AND field type. `parent` field on `Event` typed `RunID`, not as a proxy under a different name. `startTurnID`/`endTurnID` typed `TurnID`. `parent` on envelope vs `subagentID` on payload are distinct fields.
- **AG-05 W1**: no vacuous reconstruction helper for AG-06. AG-06 has no reconstruction property test (no cross-family reconstruction semantic); if a helper is added, bite-test RED before GREEN.
- **AG-05 W2**: split name-check vs structural placement-check. `TestEventKinds_AG06RegisterPlacementTurn_NameCheck` AND `TestEventKinds_AG06PlacementTurn_StructuralPin` are two separate tests, not one name-prefix test.
- **AG-04 W1 + W2**: bites assert slice index via `ai.AtIndex("event", i)`, not sequence value.
- **AG-04 W7**: cost's "estimate or final" rule has behavioral test (S-APE-040), not doc-phrase check. Token-only shape has mechanical reflection pin (S-APE-083), not doc-phrase check.
- **AG-04 W9**: scenario count `9 charter → 15 spec + 4 bites = 19 total (new spec); 5 added + 6 preserved (envelope-delta)` stated identically in tasks, apply-progress, verify-report.
- **AG-04 `6c821c0a`**: `golangci-lint cache clean` before every lint gate.
- **Engram #2944**: `make vuln-check` is NOT in `make all`. Run it explicitly.
- **Engram #2961**: all six `sdd-attempt settle` flags passed.
- **Engram #2962**: `test-driven-development` skill does not exist; RED-GREEN-REFACTOR discipline forwarded inline from `openspec/AGENTS.md`.
- **Engram #2963**: `sdd-archive` runs in worktree (the PR branch), commits to branch, pushes. After merge, main fast-forwards.

## Out-of-scope explicitly

- AG-10 / AG-16 / AG-18 / AG-19 (mechanisms that emit these kinds) — defer per charter `0003:612`
- `event_descriptor.go` struct (doc-only note allowed in AG-06.5; no struct edit)
- `stream_check.go` / `failure.go` / `sequence.go` (substrate untouched — third consecutive milestone demonstrating extensibility)
- `go.mod` / `go.sum` / `Makefile` / `.golangci.yml` (no new deps, no tooling change)
- AG-03 boundary tests (`import_boundary_test.go`, `ambient_authority_test.go`) — must remain green, NOT modified
- `backend/agent/src/agenttest` (test-convenience wrappers — AG-23's scope)
- Live emission — AG-06 ships events constructible; mechanisms are AG-10/16/18/19's scope

## Forecast

- **Changed lines**: 1500–2400 (forecast; AG-05 precedent 2,479 actual)
- **Files**: 4 new production + 4 new test + 5 modified (event.go, event_registry_test.go, doc_contract_guard_test.go, doc.go, event_descriptor.go doc-only) = 13 files (AG-05 was 13)
- **Commits**: 5 internal commits (AG-06.1 → AG-06.5)
- **Risk**: Low (proven substrate; AG-04/AG-05 bet pays for the 3rd time)
- **`size:exception`**: pre-authorized open-endedly (braejan's AG-04 standing instruction)

## Risks

- W3 latent trap (carried from AG-04/AG-05): any AG-06 kind declaring `Terminal: true` with `BracketRoleNone` would slip past per-leaf phases. Mitigated by 1.13 + 4.6 + 5.8 (every descriptor row has explicit `Terminal: false` asserted).
- Cost shape (D1b 2 kinds) overlap `cost_session` ↔ cumulative-on-turn-end. Mitigated by S-APE-030 + S-APE-031 placement assertion (cost_session PlacementRun distinct from cost_turn PlacementTurn).
- `compaction_failed` `Terminal: false` misread as auto-recovery. Mitigated by spec scenario S-APE-070 + bite S-APE-084 (engine accepts follow-on events but does NOT synthesize recovery).
- `resolution_remembered` cardinality conflict (per-tool vs per-stream). Mitigated by S-APE-082 bite (per-tool `CardinalityAtMostOne` enforced mechanically via stream validator).
- Span identity (D5b turn bracket) couples turn IDs to compaction. Mitigated: TurnID already in envelope (S-AEV-004); span by turn bracket is the natural shape.
- Spec prefix `R-APE-` may collide with future AG-23. Low risk; AG-23 will use `R-L3R-` (Layer 3); post-Wave-3.
- Lint cache phantom finding (AG-04 `6c821c0a`). Mitigated: `golangci-lint cache clean` before every lint gate (1.14, 2.9, 3.11, 4.11, 5.19, 6.1).
- `make vuln-check` not in `make all` (Engram `obs #2944`). Mitigated: run explicitly at 6.3.
- `sdd-attempt settle` flags incomplete (Engram `obs #2961`). Mitigated: all six flags passed at 6.14.
- `sdd-archive` worktree-vs-main sequencing (Engram `obs #2963`). Mitigated: orchestrator's archive step mirrors implementation step's location (worktree, PR branch).
- Scenario count drift (AG-04 W9). Mitigated: stated identically at 6.13 with cross-reference to proposal.
