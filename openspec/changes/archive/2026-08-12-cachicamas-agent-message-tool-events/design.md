# Design — `cachicamas-agent-message-tool-events` (AG-05)

> Inputs: `proposal.md` (locked A1–A5), `specs/agent-message-tool-events/spec.md` (R-AMT-001..009, 15 scenarios + 4 bites), `changes/.../specs/agent-event-envelope/spec.md` (delta MODIFIED R-AEV-007/010, ADDED R-AEV-012), AG-04 substrate at `967d043f`, Engram `sdd/cachicamas-agent-event-envelope/verify-report` (W3 latent-trap guardrail). Evidence gate: `cd backend/agent && make test` plus `make lint`, `make build`, `make vuln-check` (last NOT in `make all`, Engram `obs #2944`).

## Technical Approach

AG-05 extends AG-04 by registering 11 new kinds (6 message + 5 tool) with **zero edits** to `event_descriptor.go` / `stream_check.go` / `failure.go` / `sequence.go` — the AG-05 bet, proven by `S-AEV-110..112` (`R-AEV-012`). The AG-05.3 reconstruction property (`S-AMT-070`, `R-AMT-008`) is the load-bearing acceptance criterion; helpers are bite-tested RED twice (`S-AMT-071` drop, `S-AMT-072` double) before the property test is GREEN.

## Architecture Decisions

| AD | Choice | Rationale |
|----|--------|-----------|
| **AD-1** | Six-step procedure restated with `Terminal: false` explicit (new step between accessor and registry row) | W3 latent trap: `Terminal` was declared but inert — `c203f25c` made the validator read it; AG-05's 11 kinds declare it explicitly. Follows `R-AEV-012` (AG-04.4 extensibility pattern), validated by `S-AEV-110..112`. |
| **AD-2** | All 11 register `PlacementTurn`, `BracketRoleNone` | AG-04.3 reserved `PlacementTurn` at `event_descriptor.go:78-85`; `stream_check.go:161` rejects outside-turn with zero edit. Satisfies `R-AMT-009` (placement assertion, `S-AMT-080`). |
| **AD-3** | Tool call ordinal is a payload field | R-13 traces ordinal to doc 0002 AI-30 payload-side; envelope identity-shape preserved; no edit to `sequence.go`. Satisfies `R-AMT-007` and the joint closure of envelope invariant 1 with `R-AEV-007` + `R-AMT-003` (no-snapshot pin, AG-04.3 + AG-05.1 per `0003:2203`). |
| **AD-4** | Reconstruction helpers INSIDE `reconstruction_test.go` (test-only, single file) | Bite-tested RED twice (`S-AMT-071`, `S-AMT-072`) before property GREEN; vacuous helper is proposal risk 2. Same external test package `agent_test` (NFR-AMT-001). Satisfies `R-AMT-008` (load-bearing acceptance criterion, `S-AMT-070`). |
| **AD-5** | `S-AEV-090` retightened 4 → 15 same commit; doc-matrix guard iterates the registry (auto-extends) | Witness table grows physically; `S-AMT-081` (16th scratch) bites BEFORE merge. Joint with `R-AEV-010` scope-fence — exactly 15 (total kinds registered). |
| **AD-6** | Per-family file: payload + ctors + kind constants + descriptors co-located in ONE file per family | `message_text.go` (3), `message_reasoning.go` (3), `tool_event.go` (5) — each holds payload types, constructors, kind constants, and `EventDescriptor` rows together. Mirrors AG-04's `run_events.go` precedent (RunStart + RunEnd payloads + descriptors co-located). |

## Data Flow

```
constructor (family file, co-located) → Event{payload, run, turn}
  → eventRegistry[i] → descriptor {BracketRoleNone, PlacementTurn, Terminal:false}
    → stream_check.go:161: turn must be open, else ErrMisplaced
    → reconstruction_test.go (helpers co-located): reconstructMessage / reconstructToolOutcome
       (RED `S-AMT-071/072`, then GREEN `S-AMT-070`, per R-AMT-008)
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `backend/agent/src/agent/message_text.go` | Create | Text-kind payloads + ctors + 3 `EventDescriptor` rows + kind constants, all co-located. Kind-segregation invariant — `R-AMT-001`. |
| `backend/agent/src/agent/message_reasoning.go` | Create | Reasoning-kind payloads + ctors + 3 `EventDescriptor` rows + kind constants. Own bracket lifecycle, independent of text — `R-AMT-002`. |
| `backend/agent/src/agent/tool_event.go` | Create | `ToolStart`/`Progress`/`End*` payloads + 5 `EventDescriptor` rows + kind constants. `R-AMT-005` (ToolStart fields + indexed progress), `R-AMT-006` (three typed end states), `R-AMT-007` (ordinal payload-side, AD-3). |
| `backend/agent/src/agent/reconstruction_test.go` | Create | Reconstruction helpers (test-only, inline) + AG-05.3 property `S-AMT-070` + 2 bite tests RED-first `S-AMT-071/072`. Satisfies `R-AMT-008` (whole-vs-fragmented equality, `R-AMT-004`). |
| `backend/agent/src/agent/event.go` | Modify | Extend `EventKind` const block + 11 `eventRegistry` rows. Deltas carry index + fragment only (no-snapshot pin — `R-AMT-003`). |
| `backend/agent/src/agent/event_descriptor.go` | Modify | Restate six-step with `Terminal` explicit (DOC note — `R-AEV-012` extensibility experiment pattern). |
| `backend/agent/src/agent/event_registry_test.go` | Modify | Witness 4 → 15; `S-AEV-090` retightened to exactly 15 (scope-fence — `R-AEV-010`); all 11 register `PlacementTurn` (`R-AMT-009`); `S-AMT-081` 16th-scratch bite. |
| `backend/agent/src/agent/doc_contract_guard_test.go` | Modify | Guard iterates the registered kinds; the iteration auto-extends from 4 to 15 with the new kinds — no textual change to the guard's logic, only the registry it iterates (same scope-fence as `R-AEV-010`). |
| `backend/agent/src/agent/agent_test_helpers_test.go` | Modify | Reuse `requireViolationPosition`; add AG-05 helper. |
| `backend/agent/src/agent/doc.go` | Modify | `L2C-05` prose alongside `L2C-04`. |
| `backend/agent/src/agent/stream_check.go` | **UNCHANGED** | PlacementTurn rule already exercises. |
| `backend/agent/src/agent/failure.go` | **UNCHANGED** | `R-AMT-006` reuses `agent.Failure` typed value. |
| `backend/agent/src/agent/sequence.go` | **UNCHANGED** | Identity-shape preserved (`R-AMT-007` ordinal payload-side). |
| `backend/agent/{go.mod,go.sum}` | **UNCHANGED** | No new deps. |

## Interfaces / Contracts

```go
// message_text.go (R-AMT-001) / message_reasoning.go (R-AMT-002)
func MessageStartText(run RunID, turn TurnID, msgID ai.MessageID) (Event, error)
func MessageDeltaText(run RunID, turn TurnID, msgID ai.MessageID, idx uint32, fragment string) (Event, error)
func MessageEndText(run RunID, turn TurnID, msgID ai.MessageID) (Event, error)
// Reasoning variants: identical signature, own bracket lifecycle.

// tool_event.go (5 kinds; ordinal payload-side per AD-3, R-AMT-007)
func ToolStart(run RunID, turn TurnID, callID string, ordinal uint32, name string, args json.RawMessage) (Event, error)
func ToolProgress(run RunID, turn TurnID, callID string, idx uint32, payload json.RawMessage) (Event, error)
func ToolEndSuccess(run RunID, turn TurnID, callID string, ordinal uint32, result json.RawMessage) (Event, error)
func ToolEndResultFailure(run RunID, turn TurnID, callID string, ordinal uint32, result json.RawMessage) (Event, error)
func ToolEndExecutionFailure(run RunID, turn TurnID, callID string, ordinal uint32, failure *Failure) (Event, error)

// reconstruction_test.go (helpers co-located, test-only; satisfies R-AMT-008)
type Message struct{ ID ai.MessageID; Fragments []string }
type ToolOutcome struct{ CallID string; Ordinal uint32; Result json.RawMessage; Failure *Failure }
func reconstructMessage(events []Event) (Message, error)
func reconstructToolOutcome(events []Event) (ToolOutcome, error)
```

## Testing Strategy (strict TDD — 15 scenarios, all RED-first)

| Layer | What | How |
|-------|------|-----|
| Unit (per-kind) | 11 kinds construct + validate + pass `CheckEmit` | Extend `event_registry_test.go` witness 4 → 15 |
| Unit (placement) | All 11 register `PlacementTurn`; `stream_check.go:161` rejects outside turn | Extend `stream_check_test.go`; reuse `requireViolationPosition` |
| Unit (no-snapshot-route) | `S-AMT-021` — RED-first (bite): no delta → accumulated-payload; joint w/ `R-AEV-007`, satisfies `R-AMT-003` | Extend `invariant_pin_test.go` |
| Unit (bites) | `S-AMT-071` — RED-first (bite): drop-a-delta FAILS; `S-AMT-072` — RED-first (bite): double-a-delta FAILS — RED BEFORE property `S-AMT-070` | `reconstruction_test.go` RED-first |
| Property | `S-AMT-070` interleaved 2-message + 2-tool reconstruction GREEN AFTER bites RED | `reconstruction_test.go` |
| Cross-cut | `S-AMT-081` — RED-first (bite): 16th scratch fails by count RED, scratch absent from merged diff | `event_registry_test.go` |
| Lint hygiene | `golangci-lint cache clean` before any lint gate; `make lint` green against a clean cache | Cites `6c821c0a` (AG-04 retraction commit) — phantom `var-naming` finding was cache garbage |
| Boundary | AG-03's `import_boundary_test.go` + `ambient_authority_test.go` pass with zero changes | Untouched, must stay green |

## Threat Matrix

`N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. AG-05 extends the event-envelope registry; no I/O, no network, no subprocess, no VCS automation.`

## Migration / Rollout

`No migration required. AG-05 ships new kinds and tests; no persisted state, no schema, no go.mod/go.sum change. Single PR + size:exception pre-authorized against 1000-line budget, forecast 1500–2200 lines.`

## Scenario count commitment (W9 lesson from AG-04)

The scenario count `7 charter Gherkin scenarios in 3 leaves → 15 of 12–18 spec scenarios after per-rule expansion + bite + reconstruction sub-scenarios` MUST be restated identically in `tasks.md` and `apply-progress.md`. AG-04's W9 finding was a scenario-count error that propagated three artifacts deep before verify caught it.

## Note on proposal `4 → 11` vs design/delta `4 → 15`

The proposal's Affected Areas and Risks refer to the AG-05-ADDED count (11); the design and delta spec refer to the TOTAL count (4 AG-04 + 11 AG-05 = 15). Both are correct in their own framing. The scope-fence `S-AEV-090` retightens to `exactly 15` (total kinds registered) — same wording as the delta spec.

## Open Questions

`None. All locked decisions (A1–A5) resolved at proposal. Mechanics (file naming, helper signatures) documented above.`
