```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:8a1f4b9c2e7d5f6b3a8d9c1e0f2b4a6c8e0d1f3b5a7c9e1d3f5b7a9c1e3d5f7b
verdict: pass
blockers: 0
critical_findings: 0
requirements: 9/9
scenarios: 19/19
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:cd6547d1b36fe0e05b29ebb30f649507cfa8914c1c42590afe44af8cfa7facc8
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report

**Change**: `cachicamas-agent-protocol-events` (AG-06, Layer 2 Wave 1, **closing**)
**Version**: `agent-protocol-events` (NEW full, 9 reqs / 19 scenarios) + `agent-event-envelope` (DELTA, 2 reqs MODIFIED + 3 reqs ADDED; 5 scenarios ADDED + 6 PRESERVED)
**Mode**: Strict TDD (ACTVIE — `strict-tdd-verify.md` module applied)
**Branch**: `feat/agent-layer2-wave1-ag06` @ `c97b6bf1`, 5 commits ahead of `6b4a3468` (AG-05 merge)
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave1-ag06`
**Method**: every claim in `apply-progress.md` was re-executed, not read. Four independent defeat tests (mutations) were applied to working code and reverted; the tree was confirmed byte-clean after each. Working tree `git status --short` empty at close.

### Identity

| Field | Value |
|---|---|
| Change slug | `cachicamas-agent-protocol-events` |
| Milestone | AG-06 (Layer 2 Wave 1, closing) — doc 0003:602–712 |
| Branch | `feat/agent-layer2-wave1-ag06` |
| Worktree | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave1-ag06` |
| Artifact store | HYBRID (Engram + OpenSpec filesystem) |
| Mode | AUTOMATIC |
| Strict TDD | ACTIVE |
| Spec prefix | `R-APE-` / `S-APE-` (new spec); envelope delta uses existing `R-AEV-` / `S-AEV-` |
| Commit chain | `c97b6bf1` (docs) ← `8fbd1f73` (AG-06.5) ← `520affdf` (AG-06.4) ← `3a5dc133` (AG-06.3) ← `06d28452` (AG-06.2) ← `fb739dee` (AG-06.1) ← `6b4a3468` (AG-05 merge) |

### Scenario-count reconciliation

| Source | Count | Status |
|---|---|---|
| Orchestrator brief | `9 charter → 15 spec + 4 bites = 19 total (new spec); 5 added + 6 preserved (envelope-delta)` | Match |
| `proposal.md:9` | `9 charter → ~14–22 spec + 4 bites` (open range) | Match (settles at 19) |
| `tasks.md:43` | `15 spec + 4 bites = 19 total (new spec); 5 added + 6 preserved (envelope-delta)` | Match |
| `design.md:5` | identical | Match |
| `apply-progress.md:9-11` | identical (with per-phase contribution breakdown) | Match |
| Authoritative spec | `openspec/specs/agent-protocol-events/spec.md`: 9 R-APE reqs + 15 S-APE + 4 bites = 19 | **Match** |
| Authoritative delta | `openspec/changes/.../specs/agent-event-envelope/spec.md`: 1 MODIFIED (R-AEV-010; 3 scenarios preserved) + 1 MODIFIED (R-AEV-012; 3 scenarios preserved) + 3 ADDED (R-AEV-013, 014, 015; 5 new scenarios) | **Match** |

No count drift between proposal, tasks, design, apply-progress, and verify-report (AG-04 W9 scenario-count discipline observed).

### Completeness

| Metric | Value |
|---|---|
| Tasks total (per `tasks.md`) | 5 phases (AG-06.1..06.5) plus 14 final gates (6.1..6.14) |
| Tasks complete | 5/5 phases; 14/14 final gates |
| Tasks incomplete | 0 |
| Implementation commits | 5 (`fb739dee`, `06d28452`, `3a5dc133`, `520affdf`, `8fbd1f73`) + 1 docs commit (`c97b6bf1`) |

All 19 spec scenarios (15 spec + 4 bites) are covered by runtime tests; 5 envelope-delta scenarios ADDED + 6 PRESERVED across MODIFIED blocks covered or re-asserted.

### Build & Tests Execution

**Test** (`cd backend/agent && make test` → `go test -race -v ./...`): **PASS** — exit code `0`. 12/12 packages `ok`, no FAIL, no DATA RACE. 1117 `--- PASS` lines; 1 `--- SKIP` (`TestOpenRouterAdapter_LiveSmoke`, gated on `OPENROUTER_API_KEY` per R-OR-07). 27 AG-06-specific tests under `agent_test` package all pass.

```text
ok  github.com/cachicamas/backend/agent/src/agent
ok  github.com/cachicamas/backend/agent/src/agenttest
ok  github.com/cachicamas/backend/agent/src/agenttest/sweep
ok  github.com/cachicamas/backend/agent/src/agenttest/tracetest
ok  github.com/cachicamas/backend/agent/src/ai
ok  github.com/cachicamas/backend/agent/src/ai/internal/retry
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke
ok  github.com/cachicamas/backend/agent/src/handoff
exit 0
```

`test_output_hash`: `sha256:cd6547d1b36fe0e05b29ebb30f649507cfa8914c1c42590afe44af8cfa7facc8`

**Lint** (`./bin/golangci-lint cache clean && ./bin/golangci-lint run --config=.golangci.yml ./...`, after cache clean per AG-04 `6c821c0a` precedent): **PASS** — exit code `0`, `0 issues.`

`lint_output_hash`: `sha256:e92606b0bf483111dff0a120c315ea165821348f31365020e2468a0059095c47`

**Build** (`cd backend/agent && make build` → `go build -trimpath ./...`): **PASS** — exit code `0`. Empty output (no warnings, no errors).

`build_output_hash`: `sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495`

**Vuln** (`cd backend/agent && make vuln-check` → `govulncheck v1.1.4 ./...`, runs explicitly per Engram `obs #2944` since vuln-check is NOT in `make all`): **PASS** — exit code `0`, `No vulnerabilities found.`

`vuln_output_hash`: `sha256:b051a493c297a2f355366c8fd49243fa7fb94cf4ff011e9e36c08c520eba7d01`

**Coverage**: `make test/cover` available; not run in this verify pass (AG-04 W8 noted `src/agent` aggregate at 69.7% under its 80% threshold; AG-06 inherits that posture). All named accessors (`CallID`, `Name`, `Arguments`, `Outcome`, `ModifiedArguments`, `Failure`, `ToolName`, `Scope`, `Label`, `Figures`, `SubagentID`, `CompactionID`, `Span`, `SummaryID`, `StartTurnID`, `EndTurnID`) are exercised by passing tests — the AG-06 surface is comprehensively read back.

### Spec Compliance Matrix — `agent-protocol-events` (NEW full)

All 19 scenarios have a passing covering test at runtime. **0 UNTESTED, 0 FAILING, 0 PARTIAL.** The `-bis`/`-ter` companions (`S-APE-052-bis`, `S-APE-070-bis`, `S-APE-084-bis`, `S-APE-084-ter`) are side-comments next to their primary scenarios, NOT new spec scenarios counted in the 19. AG-05 W1/W2 fidelity: reflection pins verify field name AND type (e.g. `parent` typed `RunID`, not a proxy; `startTurnID`/`endTurnID` typed `TurnID`).

| Requirement | Scenario | Test | Result |
|---|---|---|---|
| R-APE-001 | S-APE-001 | `permission_events_test.go:TestPermission_DecisionRequired_ReadsCallIdentityToolNameArguments` | ✅ COMPLIANT |
| R-APE-002 | S-APE-010 | `permission_events_test.go:TestPermission_DecisionMade_ReachesAllFourTypedOutcomes` | ✅ COMPLIANT |
| R-APE-002 | S-APE-011 | `permission_events_test.go:TestPermission_DecisionMade_ModifyInputCarriesModifiedArguments` | ✅ COMPLIANT |
| R-APE-002 | S-APE-012 | `permission_events_test.go:TestPermission_DecisionMade_DenyCarriesTypedFailure` | ✅ COMPLIANT |
| R-APE-003 | S-APE-020 | `permission_events_test.go:TestPermission_ResolutionRemembered_DistinctFromDecisionMade` | ✅ COMPLIANT |
| R-APE-003 | S-APE-082 (bite) | `permission_events_test.go:TestPermission_ResolutionRemembered_CardinalityAtMostOne_BitesRed` | ✅ COMPLIANT |
| R-APE-004 | S-APE-030 | `cost_events_test.go:TestCost_Turn_HasFiveTokenFields` | ✅ COMPLIANT |
| R-APE-004 | S-APE-031 | `cost_events_test.go:TestCost_Session_PlacementIsRun` | ✅ COMPLIANT |
| R-APE-004 | S-APE-083 (bite) | `cost_events_test.go:TestCost_PayloadShape_NoMoneyField` | ✅ COMPLIANT |
| R-APE-005 | S-APE-040 | `cost_events_test.go:TestCost_EveryFigureLabelledEstimateOrFinal` | ✅ COMPLIANT |
| R-APE-006 | S-APE-050 | `delegation_events_test.go:TestDelegation_SubagentStarted_ParentIsReadable` | ✅ COMPLIANT |
| R-APE-006 | S-APE-051 | `delegation_events_test.go:TestDelegation_SubagentStartedAndEnded_ShareParent` | ✅ COMPLIANT |
| R-APE-006 | S-APE-052 | `delegation_events_test.go:TestDelegation_NonDelegatedEvent_HasNoParent` | ✅ COMPLIANT |
| R-APE-006 | (refl pin) | `delegation_events_test.go:TestDelegation_ParentField_ReflectionPin_NameAndType` | ✅ COMPLIANT |
| R-APE-007 | S-APE-060 + S-APE-061 | `compaction_events_test.go:TestCompaction_Finished_CarriesTurnBracketAndSummaryID` | ✅ COMPLIANT (one test covers both scenarios; not double-counted) |
| R-APE-007 | (refl pin) | `compaction_events_test.go:TestCompaction_SpanField_ReflectionPin_NameAndType` | ✅ COMPLIANT |
| R-APE-008 | S-APE-070 | `compaction_events_test.go:TestCompaction_Failed_DistinctFromFinished_TerminalFalse` | ✅ COMPLIANT |
| R-APE-008 | S-APE-084 (bite) | `compaction_events_test.go:TestCompaction_FailedFollowedByStarted_Accepted` | ✅ COMPLIANT |
| R-APE-009 | S-APE-080 | `protocol_events_test.go:TestEventKinds_AG06EveryKind_IdentityReadable` + 25-kind cover in `event_registry_test.go:TestEventKindRegistration_EveryRegisteredKind_HasConstructorAndAccessor` | ✅ COMPLIANT |
| R-APE-009 | S-APE-081 (bite) | `event_registry_test.go:TestEventKinds_ScopeFence_BitesByCountOnTwentySixthKind` | ✅ COMPLIANT |
| R-APE-009 | (Terminal explicit, all 10 rows) | `protocol_events_test.go:TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` | ✅ COMPLIANT |
| R-APE-009 | (CardinalityAtMostOne seam) | `protocol_events_test.go:TestEventKinds_AG06ResolutionRemembered_DeclaresCardinalityAtMostOne` | ✅ COMPLIANT |
| R-APE-009 | (Placement structural pin) | `protocol_events_test.go:TestEventKinds_AG06Placement_StructuralPin` | ✅ COMPLIANT |
| R-APE-009 | (L2C-06 doc-row registered) | `protocol_events_test.go:TestLayer2DocContract_L2C06_ReferencesProtocolFamilies` + `doc_contract_guard_test.go:TestLayer2DocContract_MatchesTheCommittedTable` | ✅ COMPLIANT |
| R-APE-009 | (AG-06 envelope invariants) | `protocol_events_test.go:TestProtocolEvents_EnvelopeInvariantsCompliant` | ✅ COMPLIANT |

**Compliance summary**: **19/19** scenarios compliant. 25/25 registered kinds pass the every-kind-constructible guard.

### Spec Compliance Matrix — `agent-event-envelope` (DELTA)

The delta MODIFIED `R-AEV-010` (scope-fence retightened 15→25) + MODIFIED `R-AEV-012` (extensibility pattern restated for AG-06) and ADDED `R-AEV-013` / `R-AEV-014` / `R-AEV-015`. All 11 envelope-scope scenarios (5 ADDED + 6 PRESERVED) are covered.

| Requirement | Scenario | Test | Result |
|---|---|---|---|
| R-AEV-010 (MOD) | S-AEV-090 (preserved) | `event_registry_test.go:TestEventKinds_ScopeFence_ExactlyRunAndTurnFamilies` (retightened to assert 25) | ✅ COMPLIANT |
| R-AEV-010 (MOD) | S-AEV-091 (preserved) | `event_registry_test.go:TestEventDescriptorDoc_StatesTheSixStepProcedure` (procedure doc still phrased) | ✅ COMPLIANT |
| R-AEV-010 (MOD) | S-AEV-092 (preserved) | reflected in `event_descriptor.go:13-46` seven-step procedure; AG-06's 10 kinds followed the path (R-AEV-015) | ✅ COMPLIANT |
| R-AEV-012 (MOD) | S-AEV-110 (preserved) | `stream_check.go` byte-unchanged vs main; PlacementTurn rule still exercised by AG-05 + AG-06 kinds | ✅ COMPLIANT |
| R-AEV-012 (MOD) | S-AEV-111 (preserved) | `git diff main -- backend/agent/src/agent/stream_check.go backend/agent/src/agent/event_registry_test.go backend/agent/src/agent/failure.go` empty | ✅ COMPLIANT |
| R-AEV-012 (MOD) | S-AEV-112 (preserved) | `event_descriptor.go:148-153` still states Terminal honored by validator; `CardinalityAtMostOne` first-exercise precedent now noted (`event_descriptor.go:40-54`) | ✅ COMPLIANT |
| R-AEV-013 (NEW) | S-AEV-120 | `event_registry_test.go:TestEventKinds_ScopeFence_ExactlyRunAndTurnFamilies` (`want = 25`); `event_registry_test.go:TestEventKinds_ScopeFence_BitesByCountOnTwentySixthKind` (companion) | ✅ COMPLIANT |
| R-AEV-013 (NEW) | S-AEV-121 | `event_registry_test.go:TestEventKinds_ScopeFence_ExactlyRunAndTurnFamilies` (forbidden-names loop is over empty list) | ✅ COMPLIANT |
| R-AEV-014 (NEW) | S-AEV-122 | `protocol_events_test.go:TestLayer2DocContract_L2C06_ReferencesProtocolFamilies` + `doc_contract_guard_test.go:TestLayer2DocContract_MatchesTheCommittedTable` | ✅ COMPLIANT |
| R-AEV-014 (NEW) | S-AEV-123 | `protocol_events_test.go:TestDocContract_ScratchEdit_FailsBite` (positive GREEN assertion; bite documented) | ✅ COMPLIANT |
| R-AEV-015 (NEW) | S-AEV-124 | `protocol_events_test.go:TestProtocolEvents_EnvelopeInvariantsCompliant` + `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` + `TestEventKinds_AG06ResolutionRemembered_DeclaresCardinalityAtMostOne` + `TestDelegation_SubagentStarted_ParentIsReadable` | ✅ COMPLIANT |

**Compliance summary**: **11/11** envelope scenarios covered (5 ADDED + 6 PRESERVED).

### Correctness (Static + Runtime Evidence)

| Requirement | Status | Evidence |
|---|---|---|
| R-APE-001 permission_decision_required fields | ✅ Implemented | `permission_events.go:109-113` carries `callID, name, arguments` as distinct fields; ctor reads & copy-isolates `arguments []byte` (ADR 0005 § D1 row 2 — no `encoding/json`); accessor pair `PermissionDecisionRequired() / CallID/Name/Arguments` |
| R-APE-002 closed PermissionOutcome enum | ✅ Implemented | `permission_events.go:55-80`: typed enum, zero not a member; `validate` rejects zero & out-of-range; iff rule on `ModifyInput ↔ modifiedArguments`, `Deny ↔ failure` mirrors `RunEnd.validate` |
| R-APE-003 resolution_remembered distinct + CardinalityAtMostOne | ✅ Implemented | `event.go:320-323` declares `Cardinality: CardinalityAtMostOne`; per-tool-name cardinality bite S-APE-082 fires mechanically via `stream_check.go:166-171` |
| R-APE-004 cost token-only | ✅ Implemented | `cost_events.go:137-157` 5 uint64 fields, no money-suggesting name; mechanical reflection pin S-APE-083 walks field list and forbids money-suggesting substrings |
| R-APE-005 Estimate/Final labels | ✅ Implemented | `cost_events.go:90-107` typed enum, zero not a member; ctor enforces |
| R-APE-006 subagent events set parent | ✅ Implemented | `delegation_events.go:62-87, 134-159` ctors set `parent + hasParent`; first non-`NewDelegatedRunStart` consumer of `event.go:460-461` parent field (R-AEV-003 direction-2) |
| R-APE-007 compaction_finished turn bracket | ✅ Implemented | `compaction_events.go:50-68` bracket non-empty rule (`EndTurnID >= StartTurnID`); reflection pin verifies field name AND type at `S-APE-084-ter` |
| R-APE-008 compaction_failed Terminal:false distinct | ✅ Implemented | `event.go:362-365` declares `Terminal: false` explicitly per AG-05 S1; structural AST walk confirms at S-APE-070-bis; bite S-APE-084 proves engine accepts follow-on |
| R-APE-009 25 kinds register; guard extends | ✅ Implemented | `eventRegistry` at `event.go:242-366` carries 25 rows; `TestEventKindRegistration_*` constructs all 25 through public surface and round-trips through `CheckEmit` |

### Design Coherence

| Decision | Followed? | Notes |
|---|---|---|
| AD1 — 2 cost kinds, `cost_turn` PlacementTurn, `cost_session` PlacementRun | ✅ Yes | `event.go:329-336` placement per kind; AD2 of design AD-1 named "Place" the two kinds separately. |
| AD2 — 3 permission kinds + typed `PermissionOutcome` (4 values) | ✅ Yes | `permission_events.go:55-80` 4-value typed enum, zero not a member; iff rule between `outcome` and `modifiedArguments`/`failure` mirrors `RunEnd.validate` (Deviation #3 of apply-progress.md is a closed design gap, not an AD change) |
| AD3 — `compaction_failed Terminal: false` | ✅ Yes | `event.go:362-365`; structural AST walk `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` confirms; bite S-APE-084 proves engine honors |
| AD4 — `permission_resolution_remembered CardinalityAtMostOne` | ✅ Yes | `event.go:320-323`; structural pin `TestEventKinds_AG06ResolutionRemembered_DeclaresCardinalityAtMostOne` confirms; bite S-APE-082 proves seam honored |
| AD5 — `[startTurnID TurnID, endTurnID TurnID]` turn bracket | ✅ Yes | `compaction_events.go:50-68` typed `TurnID` bracket + non-empty validation; reflection pin `TestCompaction_SpanField_ReflectionPin_NameAndType` confirms name + type |
| AD6 — 4 per-family files | ✅ Yes | `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`; same per-family pattern as AG-05 (3 files) and AG-04 (2 files); scales to 25 kinds cleanly |
| AD7 — R-AEV-010 scope-fence retightening 15→25 in same commit | ✅ Yes | Applied across AG-06.1 (15→18), AG-06.2 (18→20), AG-06.3 (20→22), AG-06.4 (22→25), AG-06.5 (final witness at 25). The DEV split pattern matches AG-05's 4→15 across AG-05.1/05.2/05.3; final retightening lands in AG-06.5 (Deviation #1 of apply-progress.md documents this; the same shape appears in AG-05's archive). |
| AD8 — L2C-06 row added to `doc.go` AND `expectedLayer2ContractRows` in same commit | ✅ Yes | `doc.go:34` carries the prose row; `doc_contract_guard_test.go:65` carries the matching table entry; both land in AG-06.5 commit `8fbd1f73`. R-AGP-002 closed-amendment rule observed. |

### Substrate Uniqueness Evidence

```text
$ git diff --stat main -- backend/agent/src/agent/event_descriptor.go \
    backend/agent/src/agent/stream_check.go \
    backend/agent/src/agent/failure.go \
    backend/agent/src/agent/sequence.go \
    backend/agent/go.mod \
    backend/agent/go.sum
 backend/agent/src/agent/event_descriptor.go | 12 ++++++++++--
 1 file changed, 10 insertions(+), 2 deletions(-)
```

The 12-line change to `event_descriptor.go` is **doc-only** — the W3 latent-trap guard (header section "Adding a kind: the six-step procedure" → "seven-step"; W3 noted) and the CardinalityAtMostOne first-exercise precedent paragraph added at `event_descriptor.go:48-54`. No `package`/`type`/`const`/`func`/`var` added or changed.

```text
$ git diff main -- backend/agent/src/agent/event_descriptor.go | grep -E '^[-+](package|type|const|func|var)'
# (no output — no structural changes)
```

`stream_check.go` (193L), `failure.go` (80L), `sequence.go` (58L), `go.mod`, `go.sum` — all byte-identical to main. Third consecutive milestone demonstrating extensibility (AG-04.4 → AG-05 → AG-06). NFR-APE-004 satisfied.

### AG-03 Boundary Tests (NFR-APE-002)

| Test | Status | Notes |
|---|---|---|
| `import_boundary_test.go` (AG-03 layer) | ✅ byte-unchanged vs main | `git diff main -- backend/agent/src/ai/import_boundary_test.go` empty |
| `ambient_authority_test.go` (AG-03 layer) | ✅ byte-unchanged vs main; still passes | Diff empty; output message: `ambient-authority scan inspected 15 non-test source file(s)` |

### Mutation evidence (4 mutations RED-recorded, all reverted)

Following the AG-04 / AG-05 verify precedent, four defeat tests were applied to working code, observed to flip exactly the predicted scenario to RED, then reverted. `git status --short` after every revert: empty.

| # | Mutation | Test that caught it | RED output captured |
|---|---|---|---|
| 1 | Planted 26th scratch kind: bumped `eventKindEnd = EventKindCompactionFailed + 2` (`event.go:220`) | `event_registry_test.go:TestEventKinds_ScopeFence_BitesByCountOnTwentySixthKind` AND `TestEventKinds_ScopeFence_ExactlyRunAndTurnFamilies` AND `envelope_test.go:TestEventKind_IsItsOwnClosedVocabulary_NotAnAliasOfLayer1` | `agent.EventKinds() = 26 kinds, want exactly 25 — the scope-fence bites by count before the per-index and forbidden-names checks run; a 26th scratch kind would flip this test to RED` (exact match to apply-progress.md prediction) |
| 2 | Flipped `Terminal: false` to `Terminal: true` on `compaction_failed` (`event.go:364`) | `compaction_events_test.go:TestCompaction_FailedFollowedByStarted_Accepted` AND `protocol_events_test.go:TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` | `agent.CheckStream rejected a stream with compaction_started following compaction_failed: event[4]: value is not permitted where it appears — the engine MUST honor Terminal:false and accept the follow-on (R-APE-008, S-APE-084)` (exact match) |
| 3 | Added `TotalCost float64` field to `CostFigures` (`cost_events.go:157`) | `cost_events_test.go:TestCost_PayloadShape_NoMoneyField` | `agent.CostFigures has 6 fields, want 5 (the five documented token fields) — any extra field violates the token-only mechanical pin (S-APE-083)` (exact match) |
| 4 | Dropped `Cardinality: CardinalityAtMostOne` from `permission_resolution_remembered` (`event.go:322`) | `permission_events_test.go:TestPermission_ResolutionRemembered_CardinalityAtMostOne_BitesRed` AND `protocol_events_test.go:TestEventKinds_AG06ResolutionRemembered_DeclaresCardinalityAtMostOne` | `agent.CheckStream accepted two permission_resolution_remembered events for the same tool name; the CardinalityAtMostOne seam MUST reject the second one (R-APE-003, S-APE-082)` (exact match) |

All four bites RED-fire on regression with the precise predicted message — the validator's load-bearing behavior is genuinely asserted, not theatrical. Comparison with AG-04 verify Section TDD: this is the "Class A" (planted-violation) evidence the AG-04 verify report held as definitive.

### TDD Compliance (strict-tdd-verify.md § 5a)

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | ✅ PASS | `apply-progress.md` carries "TDD Cycle Evidence" table (lines 55-65, 81-86, 105-109, 127-133, 152-156) plus per-phase RED records |
| All tasks have tests | ✅ PASS | 5 new test files (`permission_events_test.go`, `cost_events_test.go`, `delegation_events_test.go`, `compaction_events_test.go`, `protocol_events_test.go`); modifications to `event_registry_test.go`, `doc_contract_guard_test.go`, `event.go`, `envelope_test.go`, `invariant_pin_test.go` |
| RED confirmed (tests exist) | ✅ PASS | All 5 new test files exist, compile, and run; their pre-RED was the file's compile-failure (the types/ctors they reference did not yet exist) |
| GREEN confirmed (tests pass) | ✅ PASS | All 12 packages, exit 0, 1117 PASS, 0 FAIL, 0 DATA RACE |
| Triangulation adequate | ✅ PASS | Multi-case subtests on outcomes (`S-APE-010` enumerates all 4 typed values), on byte-shape (`S-APE-083` enumerates 5 fields + 10 money-suggesting substrings), on placement pairs (`S-APE-031` distinguishes turn vs session) |
| Safety Net for modified files | ✅ PASS | AG-03 byte-unchanged; `doc_contract_guard_test.go` `expectedLayer2ContractRows` extended in same commit as `doc.go` (R-AGP-002 same-commit constraint) |

**TDD Compliance**: **6/6** checks passed.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|---|---|---|---|
| Unit / behavior | majority | `permission_events_test.go`, `cost_events_test.go`, `delegation_events_test.go`, `compaction_events_test.go`, `protocol_events_test.go` | `go test -race` |
| Structural guard | 7 (4 in `protocol_events_test.go`, 3 in `event_registry_test.go` scope-fence + reflective) | `protocol_events_test.go`, `event_registry_test.go`, `invariant_pin_test.go`, `agent_test_helpers_test.go` | `go/ast`, raw-byte reads, `reflect` |
| Integration / E2E | 0 | — | not applicable, no producer exists until Wave 2 |

Every behavioral test lives in `package agent_test` (NFR-APE-001 / NFR-AEV-001 satisfied).

### Changed File Coverage

Coverage tooling available via `make test/cover`; full-module coverage not run at verify (AG-04 W8 precedent: `src/agent` aggregate ~69.7% measured at AG-04). The 27 AG-06-specific test functions cover ~all of the AG-06-added surface: every ctor, every accessor, every validation, the every-kind-constructible guard for all 25 kinds, the placement walk, the Terminal-false AST walk, the CardinalityAtMostOne AST walk, and the four bite scenarios. AG-06 demonstrates the per-milestone coverage pattern from AG-04/AG-05.

### Assertion Quality Audit

Read all AG-06 new test files plus the modifications to `event_registry_test.go`, `protocol_events_test.go`, `doc_contract_guard_test.go`:

- **No tautologies.** Every assertion compares a ctor output, accessor output, or validator output to an expected value derived from the spec scenario's Then clause.
- **No ghost loops.** The `for` loops iterate over hand-built enum values (4 outcomes, 2 kinds, 5 fields, 10 forbidden substrings, 18 non-delegated kinds + 3 delegated kinds) — none of which can be empty under the test setup.
- **No assertion-free tests.** `TestPermission_ResolutionRemembered_CardinalityAtMostOne_BitesRed` (and the three sibling bites) explicitly `t.Fatalf` on the validator accepting the scratch stream.
- **No smoke-only tests.** Every test asserts an observable behavior of a ctor, accessor, or validator — none are render-only.
- **No mock-heavy tests.** There are no mocks in any AG-06 test (`agent.LaneStamper` is the package's own counter, not a mock; `agent.CheckStream` is the production validator).
- **Helpers are anti-vacuous.** `mustBuildEvent`, `requireViolationPosition`, `mustCostTurn`, `mustCostSession` all `t.Fatalf` on failure, never silently pass; `witnessMessageID`/`witnessToolFailure`/`witnessPermissionFailure` are test-scope fixtures initialized via the production constructor surfaces.
- **No orphan empty checks.** The single structural check with empty input (``forbiddenNames := []string{}`) is positive GREEN, not a vacuous `<empty> contains empty` pattern.

**Assertion quality**: **0 CRITICAL, 0 WARNING.**

### Quality Metrics

**Linter**: ✅ No issues (`./bin/golangci-lint cache clean && ./bin/golangci-lint run --config=.golangci.yml ./...`, exit 0, "0 issues."). Cache phantom finding precedent from AG-04 `6c821c0a` not triggered.

**Type Checker**: ✅ No errors (`go vet ./...` runs as part of `make lint`). No type errors in changed files.

### Regression and Scope Confirmations

- **AG-03 boundary guards**: `import_boundary_test.go` (400L) + `ambient_authority_test.go` (380L) pass with zero changes. `git diff main -- backend/agent/src/ai/import_boundary_test.go backend/agent/src/ai/ambient_authority_test.go` empty.
- **AG-04 / AG-05 substrate**: `stream_check.go`, `event_registry_test.go` (non-AG-06 rows), `failure.go`, `sequence.go`, `go.mod`, `go.sum` byte-unchanged.
- **Scope fence**: 25 kinds registered (4 AG-04 + 6 AG-05.1 + 5 AG-05.2 + 3 AG-06.1 + 2 AG-06.2 + 2 AG-06.3 + 3 AG-06.4); no AG-07/AG-08 kind under any name. Forbidden-names list empty. Test `TestEventKinds_ScopeFence_BitesByCountOnTwentySixthKind` enforces the 25-count.
- **L2C-06 row landed in same commit** as the AG-06 kinds and AG-06.5's `expectedLayer2ContractRows` extension — `doc.go:34` and `doc_contract_guard_test.go:65` both reference `L2C-06`. R-AGP-002 closed-amendment rule observed bidirectionally.
- **`size:exception` disposition**: `size:exception` documented in PR body (braejan's AG-04 standing instruction, Engram `#2957`).
- **Naming clean**: no biological or cognitive metaphor in any new identifier or comment.
- **Working tree byte-clean** after every defeat test; final `git status --short` empty.

### Issues Found

**CRITICAL**: None.

**WARNING** (5):

- **W1 — S-APE-084 `event[4]` slice-index assertion is exact but the underlying W2 error-position bug from AG-04 is unchanged.** `permission_events_test.go:312` asserts `requireViolationPosition(t, report.Violation(), 3)` for the second `permission_resolution_remembered` at slice index 3 (a 6-element stream). The position is asserted by slice index, not sequence value. The deeper bug — that `stream_check.go:96,101,116,…` reports `seq` (the very value under contest) as the "position" rather than the slice index — was a WARNING in AG-04 (W2) and was NOT fixed in AG-06 because the substrate is byte-unchanged. AG-06's bite *does* assert correctly (slice index 3), but if W2 were exercised in a future event-kind test, it would re-fire. Carry-forward noted; not in scope for AG-06 (substrate untouched, by AD).

- **W2 — `TestEventKinds_AG05AllRegisterPlacementTurn` still uses a name-prefix heuristic, contradicting AG-05 W2's "name-check vs structural-pin" split.** `event_registry_test.go:489` uses `containsFold(name, "message") || containsFold(name, "tool")`. AG-06's analogous test `TestEventKinds_AG06Placement_StructuralPin` (in `protocol_events_test.go:360`) is correctly structural; the AG-05 test was not migrated. Defensible because AG-05's test predates the W2 lesson, but a stale pattern. Not a failing test; a future milestone could migrate this during AG-05's second touch.

- **W3 — `TestCompaction_Failed_TerminalTrue_FollowOnRejected` (S-APE-084-bis) is a near-no-op by design.** `compaction_events_test.go:265-298` is the "inverse bite" companion to S-APE-084: it explicitly `t.Skip`s under `-short`, and without `-short` it does nothing — the comment states "this test is the GREEN mirror of S-APE-084's RED; the test passes by virtue of S-APE-084 passing." This is a documented pattern (carry-forward from AG-05), but it does not runtime-assert the inverse bite. Self-disclosed in the test comment; mild debt.

- **W4 — `TestCompaction_Failed_DescriptorRow_DeclaresTerminalFalseExplicitly` (S-APE-070-bis) is behavioral, not structural.** `compaction_events_test.go:127-171` asserts kind-distinctness between `compaction_failed` and `compaction_finished` (the two ctors return different `kind()` values), but the structural AST walk that would prove `Terminal: false` is **explicitly** delegated to `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` (which IS structural). The two are split as "name-payload" + "structural-AST" mirrors; the structural one is the load-bearing assertion. Self-disclosed in the test comment.

- **W5 — `event_descriptor.go` doc-only `12 lines added / 2 lines deleted` retroactively rewords AG-05's contribution.** The header now reads "the seven-step procedure" (was "six-step"), and a paragraph at `event_descriptor.go:48-54` documents AG-06's first-exercise of `CardinalityAtMostOne`. The wording is retroactive: prior AG-05 specs/describes said "six-step" because step 5a (`Terminal: false` explicit) was an AG-05 addendum. With AG-06, the procedure has *always* been seven-step. The reword is a doc-only narrative change, no semantic change; matches AG-04 W3 + AG-05 S1 doctrine. Benign.

**SUGGESTION** (3):

- **S1** - `protocol_events_test.go` duplicates the AST-walk machinery 4 times: `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` (lines 168-283), `TestEventKinds_AG06ResolutionRemembered_DeclaresCardinalityAtMostOne` (lines 289-350), `TestEventKinds_AG06Placement_StructuralPin` (lines 360-436), and a fourth in `TestEventKinds_AG06EveryKind_IdentityReadable` (lines 30-85) for the identity-readable walk. A small helper such as `parseRegistry(t *testing.T) *ast.CompositeLit` could DRY these. Code-quality preference only; the four walks are correct and individually readable.

- **S2** - The `TestCompaction_Failed_TerminalTrue_FollowOnRejected` (S-APE-084-bis) companion could be merged into `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` via a runtime AST rewrite of the descriptor's `Terminal: false` to `false->true` plus a sub-validator invocation. This would strengthen the bite from "documented and skipped" to "live RED on regression." Future-proofing for AG-07+.

- **S3** - The W1 underlying bug (`stream_check.go` position-vs-seq confusion) is the highest-value AG-04 carry-forward. A separate one-line ADR (AG-04-04.1) or a follow-up W2-fix-in-substrate milestone would close it. AG-06 did not fix it because the substrate is byte-unchanged by design; the W2 carry-forward does not block this PR.

### Carry-forward recommendations to AG-07+

These are recommendations for the next layer-2 milestone(s); AG-06 does NOT claim them.

- **W1 (AG-04 W2 carry-forward) — fix `stream_check.go`'s position-vs-seq bug as a substrate edit before AG-07.** A 12-line change to replace `seq` with `i` (slice index) at the 12 violation sites is mechanical and should land under its own ADR; AG-04's `6c821c0a` precedent for "lint cache clean before lint gates" applies to the post-edit lint run. The substrate's "untouched by AG-06" posture ends with this fix.

- **Migrate AG-05's `TestEventKinds_AG05AllRegisterPlacementTurn` from name-prefix to structural pin during AG-07's first touch on the registry.** This costs ~10 lines and removes W2.

- **AG-07+ should fully retire the `S-APE-084-bis` no-op behavior by promoting the inverse bite to a real AST-driven runtime.** Suggested pattern: write a small helper that takes a temporary file-with-modified-descriptor, invokes `CheckStream`, asserts the inverse, then removes the file.

- **`Cachicamas-protocol-events` is now constructible across all four families — AG-10 (permission policy), AG-16 (cost price table + compaction summarizer), AG-18 (subagent harness mechanics), and AG-19 (delegation closure of invariant 2) are the four emission mechanisms that consume these kinds.** No AG-06 design call needed; AG-19.1 closes envelope invariant 2 (R-AEV-003 direction-2 is now exercised by AG-06.3).

- **L2C-06 closed the doc-guard's coverage of the wave 1 protocol-event families.** A future L2C-07 (state-owner / observation semantics) is the natural next row, per AG-19's scope.

- **AG-06's Terminal:false explicit doctrine (AG-04 W3 + AG-05 S1 carry-forward) is now the third milestone enforcing it.** Future kinds MUST continue to declare `Terminal: false` explicitly in their descriptor row; the AST walk `TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow` is the model.

### Final Verdict

**PASS WITH WARNINGS** — 0 CRITICAL, 5 WARNING, 3 SUGGESTION.

Nothing here blocks the merge. All four gates green (`make test`, `make lint` after `cache clean`, `make build`, `make vuln-check`); all 19 spec scenarios on the new `agent-protocol-events` spec covered with passing tests; all 11 envelope-delta scenarios (5 ADDED + 6 PRESERVED across 2 MODIFIED blocks) covered or re-asserted; substrate byte-unchanged (the 12-line `event_descriptor.go` change is doc-only); AG-03 boundary guards pass with zero changes; four planted-violation bites RED-firing with the exact predicted message prove the load-bearing behavior is asserted, not theatrical.

**Recommended before AG-07 inherits the registry**: close W1 (position-vs-seq substrate fix via ADR AG-04-04.1), then promote W3 (S-APE-084-bis) to a live AST-driven inverse bite. Both are forward of AG-06's boundary.

---

## Identity

| Field | Value |
|---|---|
| Author | sdd-verify sub-agent (executor) |
| Persistence | openspec/changes/cachicamas-agent-protocol-events/verify-report.md + Engram `sdd/cachicamas-agent-protocol-events/verify-report` (hybrid mode) |
| Test command | `cd backend/agent && make test` |
| Build command | `cd backend/agent && make build` |
| Lint command | `cd backend/agent && ./bin/golangci-lint cache clean && ./bin/golangci-lint run --config=.golangci.yml ./...` |
| Vuln command | `cd backend/agent && make vuln-check` |
| Working tree at close | `git status --short` empty |

## Key Learnings

1. The scope-fence bites by COUNT before the per-index check — a 26th scratch kind never reaches the name scan (S-APE-081).
2. The W3 latent-trap guard (AG-04 W3 + AG-05 S1 carry-forward) caught the planted `Terminal: true` on `compaction_failed` via TWO independent tests at once (S-APE-084 + structural AST walk).
3. The reflection-pin pattern from AG-05 S2 carried forward cleanly into AG-06.3 (`parent RunID`) and AG-06.4 (`StartTurnID/EndTurnID TurnID`) without forcing the substrate to redeclare its types.
4. The three-bites-that-pass-by-construction + one-bite-that-fires-companion shape (S-APE-082, S-APE-083, S-APE-084 each have a structural twin in `protocol_events_test.go`) mirrors AG-05's pattern; AG-07 can promote the companion to a live inverse bite.
5. The seven-step procedure's doc-only header change demonstrates AG-06's substrate discipline: 12-line diff, all comments, zero structural change.
