# Design — `cachicamas-agent-protocol-events` (AG-06)

> Inputs: `proposal.md` (D1b/D2a/D3b/D4b/D5b/D6/D7/D8 locked), `specs/agent-protocol-events/spec.md` (15 spec + 4 bites), `changes/.../specs/agent-event-envelope/spec.md` (delta), AG-05 substrate at `6b4a3468`. Gate: `make test` + `make lint` (after `cache clean`) + `make build` + `make vuln-check` (NOT in `make all`, Engram `#2944`).
>
> **Scenario count**: **9 charter → 15 spec + 4 bites = 19 total** (new spec); **5 added + 6 preserved (envelope-delta)**.
>
> **Substrate UNTOUCHED**: `event_descriptor.go` (145L, `CardinalityAtMostOne` `:103-120`); `stream_check.go` (193L, `:161`/`166-171`/`173-175`); `failure.go` (80L); `sequence.go` (58L); `go.mod`/`go.sum`; AG-03 guards.

## 1. Technical Approach

AG-06 extends the AG-04/AG-05 registry by **10 kinds** (3 permission + 2 cost + 2 delegation + 3 compaction) via the seven-step procedure (`event_descriptor.go:13-46`), with `Terminal: false` explicit on every AG-06 row (AG-05 S1). Per-family file split (AG-05 AD-6): `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`. `eventRegistry` grows 15→25 in one commit; scope-fence `S-AEV-090` retightens to "exactly 25". AG-06 is the **first** kind-set to exercise `CardinalityAtMostOne` (`permission_resolution_remembered`, R-APE-003) and the **first** non-`NewDelegatedRunStart` consumer of the parent identifier (AG-06.3). Forbidden-names list at `event_registry_test.go:326` retires. **No edit** to substrate — third consecutive milestone demonstrating extensibility (AG-04.4 + AG-05 + AG-06).

## 2. Architecture Decisions

| # | Decision | Choice | Rationale |
|---|---|---|---|
| **AD1** | Cost kind shape | **D1b**: 2 kinds (`cost_turn` PlacementTurn + `cost_session` PlacementRun); label as payload field | Charter "per-turn and cumulative" maps to scope. Mirrors AG-04 `RunOutcome` / AG-05 `ToolOutcome` (typed-value-in-one-kind > kinds-by-discriminator). |
| **AD2** | Permission family shape | **3 kinds + typed `PermissionOutcome`** (`AllowOnce`/`AllowAlways`/`Deny`/`ModifyInput`, zero not a member) | Charter `"each outcome is a distinct typed value"` (`0003:638-642`) aligns with D2a. `ModifyInput` carries modified args; `Deny` carries typed `*Failure` (R-AEV-008). |
| **AD3** | `compaction_failed` terminal | **`Terminal: false`** | Honoring `doc 0001 § 7 G3` "survive interruption" — recovery may follow. Engine accepts but does NOT synthesize recovery. Bite `S-APE-084`. |
| **AD4** | `resolution_remembered` cardinality | **`CardinalityAtMostOne`** | "Remembered" = per-tool-name within a run (once = forever). The AG-04.3-reserved seam at `event_descriptor.go:103-120` is engineered exactly for this. First AG-06 exercise (bite `S-APE-082`). |
| **AD5** | `compaction_finished` span identity | **`[startTurnID TurnID, endTurnID TurnID]` turn bracket** | Turn is the protected unit per `doc 0001 § 7 G3` "protect recent turns". `Sequence` is per-lane — sequence-only fails for multi-lane streams. |
| **AD6** | Per-family file split | **4 files** | AG-05 AD-6 per-family discipline; AG-04 used 2, AG-05 used 3, AG-06 uses 4 — scales to 25 kinds cleanly. |
| **AD7** | R-AEV-010 scope-fence retightening | **"exactly 15" → "exactly 25"** in MODIFIED `R-AEV-010` (same commit) | Single-line MODIFIED preserves the bite. Six existing scenarios preserved. |
| **AD8** | L2C-06 doc-guard row | **One new row references the 4 families**; same commit as kind registrations (R-AGP-002 closed-amendment) | Per-family semantics belong in `doc.go` prose, not in the guarded row (AG-04.3 forward-fix preference). |

## 3. Data Flow

```
producer (AG-10/16/18/19) → per-family ctor
  → envelope {payload, run, turn, parent, seq}
    → eventRegistry[i] (25 kinds) → descriptor {Placement, Bracket, Cardinality, Terminal:false}
      → CheckEmit → CheckStream → consumer
```

Per-family ctors: `NewPermissionDecisionMade(outcome, modifiedArgs, failure)`, `NewCostTurn/Session(label, five token fields)`, `NewSubagentStarted/Ended(run, parent, subagentID)` (sets `parent`/`hasParent`), `NewCompactionStarted/Finished(span, summaryID)/Failed(failure)`.

## 4. File Changes

| File | Action | Description |
|---|---|---|
| `permission_events.go` | Create | 3 kinds + `PermissionOutcome` + ctors. `resolution_remembered` → `CardinalityAtMostOne`. |
| `cost_events.go` | Create | `Cost` + `CostScope` + `CostLabel` + 2 ctors. Token-only. |
| `delegation_events.go` | Create | `SubagentStarted`/`Ended` + `delegationEventParent` helper + 2 ctors. |
| `compaction_events.go` | Create | `CompactionStarted`/`Finished`/`Failed` + `CompactionSpan` + 3 ctors. |
| `*_events_test.go` (×4) | Create | Per-family scenarios + bites. |
| `event.go` | Modify | 10 `eventRegistry` rows; move `eventKindEnd`. |
| `event_registry_test.go` | Modify | Witness 15→25; `S-AEV-090` retightens; `BitesByCountOnSixteenthKind`→`BitesByCountOnTwentySixthKind`. |
| `doc_contract_guard_test.go` | Modify | `L2C-06` row. |
| `doc.go` | Modify | `L2C-06` prose. |
| `event_descriptor.go` | **UNTOUCHED** | `CardinalityAtMostOne` seam `:103-120`; seven-step procedure `:13-46`. |
| `stream_check.go` | **UNTOUCHED** | PlacementTurn `:161`; CardinalityAtMostOne `:166-171`; Terminal `:173-175`. |
| `failure.go` / `sequence.go` | **UNTOUCHED** | `*Failure` reused; `LaneStamper` unchanged. |
| `go.mod`/`go.sum`/`Makefile`/`.golangci.yml` | **UNTOUCHED** | No new deps. |
| AG-03 guards | **UNTOUCHED** | `import_boundary_test.go` + `ambient_authority_test.go` pass with zero logic change. |

## 5. Interfaces / Contracts

```go
type PermissionOutcome uint8
const (_ PermissionOutcome = iota; PermissionOutcomeAllowOnce; PermissionOutcomeAllowAlways; PermissionOutcomeDeny; PermissionOutcomeModifyInput)
type CostScope uint8; const (CostScopeTurn CostScope = iota+1; CostScopeSession)
type CostLabel uint8; const (CostLabelEstimate CostLabel = iota+1; CostLabelFinal)

type PermissionDecisionRequired    struct{ callID, name string; arguments []byte }
type PermissionDecisionMade        struct{ callID string; outcome PermissionOutcome; modifiedArguments []byte; failure *Failure }
type PermissionResolutionRemembered struct{ toolName string; outcome PermissionOutcome }
type Cost struct{ scope CostScope; label CostLabel; inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens uint64 }
type SubagentStarted struct{ subagentID string }
type SubagentEnded   struct{ subagentID string }
type CompactionSpan struct{ StartTurnID, EndTurnID TurnID }
type CompactionStarted  struct{ compactionID string }
type CompactionFinished struct{ compactionID string; span CompactionSpan; summaryID string }
type CompactionFailed   struct{ compactionID string; failure *Failure }

// Ctors: NewPermissionDecisionRequired/DecisionMade/ResolutionRemembered,
//        NewCostTurn/Session, NewSubagentStarted/Ended (sets parent/hasParent),
//        NewCompactionStarted/Finished/Failed -> (Event, error)
```

Descriptor rows (all `Terminal: false` explicit per AG-05 S1): 10 rows. `permission_resolution_remembered` → `CardinalityAtMostOne`. `cost_session` → `PlacementRun`; rest → `PlacementTurn`. All `BracketRoleNone`.

## 6. Testing Strategy

| Layer | What | How |
|---|---|---|
| **Unit** | 10 ctors + accessors + validate + `CheckEmit` | External `agent_test`; mirror AG-05 witness-table. Reflection pin verifies **field name AND type** (AG-04 W4 + AG-05 S2). |
| **Bite** | `S-APE-081` (26th scratch fails by count); `S-APE-082` (per-tool `CardinalityAtMostOne` fires); `S-APE-083` (scratch with `money *decimal` → diff FAIL); `S-APE-084` (`compaction_failed` then `compaction_started` accepted) | All 4 RED before GREEN. Apply: `golangci-lint cache clean` before lint (AG-04 `6c821c0a`). |
| **Integration** | Ctor + `CheckStream` round-trip | Hand-built sequences via `LaneStamper` + `CheckStream`. |
| **Doc-guard** | `L2C-06` row registered | `doc_contract_guard_test.go` extension. |
| **Cross-family** | Scope-fence `S-APE-081` (26th scratch) | `event_registry_test.go`; bites by count before name scan. |
| **Boundary** | AG-03 guards unmodified | `import_boundary_test.go` + `ambient_authority_test.go` pass with zero changes. |

## 7. Threat Matrix

`N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. AG-06 is a pure table extension + type definition milestone; no I/O, no concurrency, no subprocess.`

## 8. Migration / Rollout

`No migration required. Registry grows 15 → 25 in one commit. L2C-06 row lands in same commit (R-AGP-002 closed-amendment).`

## 9. Commit Splits (Work Units — per `work-unit-commits`)

| Commit | Scope | Verification |
|---|---|---|
| **AG-06.1** | Permission family — `permission_events.go` + `_test.go` + `event.go` constants | `make test` |
| **AG-06.2** | Cost family — `cost_events.go` + `_test.go` + `event.go` constants | `make test` |
| **AG-06.3** | Delegation family — `delegation_events.go` + `_test.go` + `event.go` constants | `make test` |
| **AG-06.4** | Compaction family — `compaction_events.go` + `_test.go` + `event.go` constants | `make test` |
| **AG-06.5** | Guard + scope-fence retighten + 10 `eventRegistry` rows + `L2C-06` row + `doc.go` + `doc_contract_guard_test.go` + `event_registry_test.go` | `make test` + `make lint` (after `cache clean`) + `make build` + `make vuln-check` |

Single PR, 5 commits. Substrate byte-unchanged across all 5 — third consecutive milestone demonstrating extensibility.

## 10. Open Questions

`None. All 8 design-time decisions (D1b/D2a/D3b/D4b/D5b/D6/D7/D8) locked at proposal. Forward note to AG-10/AG-16/AG-18/AG-19: emission mechanisms will exercise these kinds; no design call needed now. AG-06.3's `subagent_started`/`subagent_ended` are the first non-`NewDelegatedRunStart` consumers of the parent identifier field (`event.go:362-366`, R-AEV-003) — AG-19.1 closes envelope invariant 2 fully.`

## Substrate Uniqueness Citation Map

| Claim | Citation |
|---|---|
| `event_descriptor.go` byte-unchanged | 145L; `CardinalityAtMostOne` `:103-120`; seven-step procedure `:13-46` |
| `stream_check.go` byte-unchanged | 193L; PlacementTurn `:161`; CardinalityAtMostOne `:166-171`; Terminal `:173-175` |
| `failure.go` byte-unchanged | 80L; `*Failure` reused by `decision_made` (Deny) and `compaction_failed` |
| `sequence.go` byte-unchanged | 58L; `LaneStamper` unchanged; `compaction_id`/`subagent_id` payload-side |
| `go.mod`/`go.sum` byte-unchanged | No new deps; Go 1.26.3, `golangci-lint v2.9.0` |
| AG-03 guards untouched | `import_boundary_test.go` (400L) + `ambient_authority_test.go` (380L) pass with zero changes |
| Argument bytes `[]byte` | `tool_event.go:65`; ADR 0005 § D1 row 2; `encoding/json` transitively pulls `os`/`io/fs`, forbidden |
| `parent` field from AG-04.1 | `event.go:362-366` (`Event.Parent() (RunID, bool)`), R-AEV-003 |
| Scope-fence bite precedent | `BitesByCountOnSixteenthKind` at `event_registry_test.go:411-425` → `BitesByCountOnTwentySixthKind` |
| Parent identifier precedent | `NewDelegatedRunStart` at `run_events.go:60-76` — only prior door that sets parent; AG-06.3's `delegationEventParent` is the second |
| Engram references | `#2944` (vuln-check explicit), `#2961` (six `sdd-attempt settle` flags), `#2962` (TDD skill gap), `#2963` (archive sequencing), `#2949` (AG-05 spec), `#2950` (AG-05 design) |

## Note on `4 + 11 + 10 = 25` vs spec wording

The proposal's `Affected Areas` and `Risks` refer to the AG-06-ADDED count (10); the design and delta spec refer to the TOTAL count (4 AG-04 + 11 AG-05 + 10 AG-06 = 25). Both are correct in their own framing. The scope-fence `S-AEV-090` retightens to `exactly 25` — same wording as the delta spec's `R-AEV-013` and the modified `R-AEV-010`.
