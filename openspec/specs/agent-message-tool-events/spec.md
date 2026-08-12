# Spec — Message lifecycle and tool execution event families (`agent-message-tool-events`)

> **Change**: `cachicamas-agent-message-tool-events` · **Milestone**: AG-05 (Layer 2, Wave 1) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-05--add-message-and-tool-execution-event-families), `0003:518-600`
> **Nodes**: AG-05.1 `[leaf]` (message) · AG-05.2 `[leaf]` (tool) · AG-05.3 `[leaf]` (reconstruction)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario independently verifiable.
> **Identifier convention**: requirements `R-AMT-0NN`, scenarios `S-AMT-0NN`. Append-only. Distinct from `R-AEV-`/`S-AEV-` (AG-04 envelope) and `R-AGE-`/`S-AGE-` (AG-01 delivery).
> **Co-closure map** (`0003:2203`): envelope invariant 1 (indexed deltas) closed by AG-04.3 **+ AG-05.1**; invariants 2, 3, 4 untouched.
> **Evidence gate**: `cd backend/agent && make test` (`go test -race -v ./...`), plus `make lint`. No CI exists. `make vuln-check` is NOT in `make all`; it MUST be run explicitly (Engram `obs #2944`).

## Coverage

| Charter scenarios | Requirements | Scenarios | Bites |
|---|---|---|---|
| **7 of 7** | 9 added | **15 of 12–18** | 4 (S-AMT-021, S-AMT-071, S-AMT-072, S-AMT-081) |

Charter → spec mapping: charter 1 → `R-AMT-001` + `R-AMT-002`; charter 2 → `R-AMT-003`; charter 3 → `R-AMT-004`; charter 4 → `R-AMT-005`; charter 5 → `R-AMT-006`; charter 6 → `R-AMT-007`; charter 7 → `R-AMT-008`. Cross-cut (every-kind-constructible extends) → `R-AMT-009`. Reconstruction sub-scenarios (drop-a-delta, double-a-delta) live under `R-AMT-008`.

## Purpose

Register the two high-volume Layer 2 families AG-05 owns — message lifecycle (`VL2-EVT-04`) and tool execution (`VL2-EVT-05`) — as constructible and validated event kinds, with the reconstruction property proven before any producer exists (`0003:418` acceptance). AG-05 is the **first kind-set to exercise `PlacementTurn`**, the seam `event_descriptor.go:78-85` reserved at AG-04.3.

## Requirements

### R-AMT-001 — Message text events bracket a text message; Layer 1 content identities readable

Text message events MUST start, carry deltas, and end a text message in declaration order, and each event payload MUST carry the Layer 1 content identity (`ai.MessageID`) readable from an external package per `S-AGV-019`. A reasoning fragment MUST NOT appear in a message-text event payload — reasoning and text are segregated at the event-kind level.

#### Scenarios
- **S-AMT-001** — Given `message_start_text`, `message_delta_text`, `message_end_text` events for one assistant message, when an external test package reads them, then the Layer 1 content identity is readable on each and no reasoning fragment appears in any payload.
- **S-AMT-002** — Given a `message_text` event payload, when inspected, then it carries no field typed for reasoning fragments; the kind-level segregation is asserted mechanically, not by comment.

### R-AMT-002 — Reasoning has its own bracket lifecycle

Reasoning MUST open and close its own bracket (`message_start_reasoning` … `message_end_reasoning`), independent of the text bracket, so reasoning streams do not need text events to bracket them.

#### Scenarios
- **S-AMT-010** — Given `message_start_reasoning`, `message_delta_reasoning`, `message_end_reasoning` events with no `message_text` events present, when validated, then the reasoning bracket is accepted as a complete cycle, distinct from text brackets at the event-kind level.

### R-AMT-003 — Message deltas honor the index pin (co-closes invariant 1 with AG-04.3)

A message delta event MUST carry a positional index and the new fragment only. The construction surface MUST offer **no route** by which a delta can be attached to an accumulated snapshot — co-closing envelope invariant 1 (`VL2-EVT-12`, `R-AEV-007`) with AG-04.3's structural pin.

#### Scenarios
- **S-AMT-020** — Given a fragmented message expressed as `message_delta_text` events, when each is validated, then it carries its index and the new fragment only, and no delta carries an accumulated snapshot.
- **S-AMT-021** — **(bite)** Given the package's construction surface, when every route that would attach a snapshot to a delta is enumerated, then none exists; the co-closure with `R-AEV-007` is asserted mechanically rather than by comment.

### R-AMT-004 — Whole message and equivalent fragmented message reconstruct equally

A consumer reconstructing a message delivered whole (`start` + `end`) and a message delivered as deltas (`start` + N `delta` + `end`) MUST produce equal reconstructions — the per-family shape of the `L2C-04` membership criterion for messages.

#### Scenarios
- **S-AMT-030** — Given one message delivered whole (`start` + `end`) and an equivalent message delivered as deltas, when a consumer reconstructs both, then the reconstructions are equal.

### R-AMT-005 — Tool start carries what a frontend needs; progress is indexed

A `tool_start` event MUST carry the tool call identity (`ai.ToolCallID`), the tool name, and the arguments — sufficient to render "running tool X with these arguments" live. `tool_progress` events for the same call MUST carry an integer index.

#### Scenarios
- **S-AMT-040** — Given a `tool_start` event for one call, when a consumer inspects it, then it carries the call identity, the tool name, and the arguments as distinct fields, and any `tool_progress` event for that call carries an integer index.

### R-AMT-006 — The three tool end states are typed outcomes distinct by kind

Success, result-reports-failure, and execution-itself-failed MUST be three typed outcomes (`ToolOutcome`), distinct by kind, not by convention over payload contents. `tool_end_execution_failure` MUST carry an `agent.Failure` typed value (reusing `R-AEV-008`'s wrap surface) — no code path assigns meaning to a message string.

#### Scenarios
- **S-AMT-050** — Given tool events for one call with each of the three end states, when inspected, then each outcome is reachable as a typed value, and no payload-convention inference is required to tell them apart.
- **S-AMT-051** — Given a `tool_end_execution_failure` constructed without an `agent.Failure`, when validated, then it is REJECTED; the typed-failure wrap is required, not optional.

### R-AMT-007 — Tool events carry a call ordinal correlating to call order regardless of completion order

A tool event's payload MUST carry a call ordinal correlating the event to the call's issuance order, independent of completion order. The ordinal MUST be a payload field, not an envelope field, to preserve the envelope's existing shape — R-13 traces the ordinal to Layer 1 payload-side (`doc 0002 AI-30`).

#### Scenarios
- **S-AMT-060** — Given tool events for several calls issued in one turn with completions arriving out of order, when inspected, then each event's call ordinal correlates to its issuance order, not its arrival order.

### R-AMT-008 — Interleaved message and tool streams reconstruct independently and completely (AG-05.3 property)

A scripted interleaved stream of message deltas and tool progress MUST reconstruct each message and each tool outcome independently and completely — the property a session log will depend on, proven before any producer exists (`0003:594-597`). The reconstruction helper MUST be bite-tested before the property test is GREEN; a vacuous helper is the documented failure mode (proposal risk 2).

#### Scenarios
- **S-AMT-070** — Given a scripted event sequence interleaving two messages' deltas and two tools' progress, when a consumer reconstructs each message and each tool outcome, then every reconstruction is independent and complete.
- **S-AMT-071** — **(bite)** Given the reconstruction helper applied to a sequence with one delta dropped, when the reconstruction is compared to the original, then the comparison FAILS — proving the helper is not vacuous. RED-recorded before `S-AMT-070` is GREEN.
- **S-AMT-072** — **(bite)** Given the reconstruction helper applied to a sequence with one delta duplicated, when the reconstruction is compared, then the comparison FAILS. RED-recorded before `S-AMT-070` is GREEN.

### R-AMT-009 — All 11 new kinds register under PlacementTurn; the every-kind-constructible invariant extends to 15

All 11 new kinds MUST register with `Placement: PlacementTurn` — AG-05's first exercise of the seam reserved at `event_descriptor.go:78-85`. The every-kind-constructible guard MUST iterate over all 15 registered kinds (4 AG-04 + 11 AG-05) through the public surface; the scope-fence `S-AEV-090` retightens from "exactly 4" to "exactly 15" in the **same commit** as the new kinds.

#### Scenarios
- **S-AMT-080** — Given the every-kind-constructible guard, when it runs, then it constructs at least one instance of every registered kind (4 AG-04 + 11 AG-05) through the public surface from an external test package, and reports having constructed 15 kinds.
- **S-AMT-081** — **(bite)** Given a 16th scratch kind planted following the documented six-step procedure, when the guard runs, then it FAILS by count before the name scan runs; the scope-fence bites before AG-06 lands. RED-recorded; scratch kind absent from the merged diff.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-AMT-001** | External-package verifiability: every scenario verifiable by `cd backend/agent && make test`. Every behavioral test in an external test package, per NFR-AEV-001. |
| **NFR-AMT-002** | Boundary guards stay green untouched: AG-03's `import_boundary_test.go` and `ambient_authority_test.go` MUST pass with zero changes to their own logic. |
| **NFR-AMT-003** | Determinism and race cleanliness: every test added MUST be deterministic, hermetic, and pass under `-race`. |
| **NFR-AMT-004** | Review budget: single PR under pre-authorised `size:exception` against 1000-line budget, forecast 1500–2200 lines — citing braejan's standing instruction in AG-04's `session_summary`. Chained PRs NOT safe (AG-05.3 reconstruction references both families). |

## Explicit non-requirements

- **No edit** to `event_descriptor.go`, `stream_check.go`, `failure.go`, or `sequence.go` — AG-04's rule engine stays untouched; AD-1, AD-2, AD-5 hold.
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit.
- **Permission, cost, delegation, compaction families** (`VL2-EVT-06`…`09`) — **AG-06**.
- **Live-loop emission** — **AG-07, AG-09**. AG-05 ships no producer.
- **Tool execution contract** (scheduling, rejoin) — **AG-09.1**. **Permission events around tools** — **AG-06, AG-10**.
- **Test-convenience wrappers** in `backend/agent/src/agenttest` — **AG-23**.
- **CI** — every gate runs when a human runs `make test` in `backend/agent/`.

## Evidence discipline

`openspec/config.yaml` declares `apply.tdd: true`; strict TDD active (`openspec/AGENTS.md`).

- All three leaves are behavior, so all three are RED-first.
- **`R-AMT-003` closes only on its recorded bite** (`S-AMT-021`): the co-closure with `R-AEV-007` is asserted mechanically.
- **`R-AMT-008` closes only on its recorded bites** (`S-AMT-071`, `S-AMT-072`): the reconstruction helper bites RED twice before `S-AMT-070` is GREEN. The helper is test-only code; the **test is the property**.
- **`R-AMT-009` closes only on its recorded bite** (`S-AMT-081`): the 16th kind fails by count RED, scratch file absent from the merged diff.
- **W3 latent-trap guard** — `Terminal: false` MUST be stated explicitly in the six-step procedure restated by `sdd-design`; AG-04.4's `S-AEV-092` extensibility experiment pattern is the documented path AG-05 followed (per delta spec's `R-AEV-012`).

## Acceptance criteria

1. Every `S-AMT-001`…`S-AMT-081` has recorded evidence.
2. `cd backend/agent && make test`, `make lint`, `make build`, and `make vuln-check` are all green.
3. `backend/agent/go.mod` and `go.sum` byte-unchanged.
4. The every-kind-constructible guard constructs all 15 kinds (4 + 11); scope-fence bites on a 16th.
5. AG-05.3 reconstruction property test GREEN with helper bites recorded RED.
6. The 7 charter Gherkin scenarios (`0003:543-583`, `0003:594-597`) are covered; none reduced.
7. AG-03's two boundary guards pass with zero changes to their own logic.
8. `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go` byte-unchanged.

## Traceability

| Requirement | Charter node | Register rows |
|---|---|---|
| `R-AMT-001` | AG-05.1 | `VL2-EVT-04` (text) |
| `R-AMT-002` | AG-05.1 | `VL2-EVT-04` (reasoning) |
| `R-AMT-003` | AG-05.1 | `VL2-EVT-12` (joint with AG-04.3) |
| `R-AMT-004` | AG-05.1 | `VL2-EVT-04` (reconstruction) |
| `R-AMT-005` | AG-05.2 | `VL2-EVT-05` (start, progress) |
| `R-AMT-006` | AG-05.2 | `VL2-EVT-05` (end states) |
| `R-AMT-007` | AG-05.2 | `VL2-EVT-05` (call ordinal) |
| `R-AMT-008` | AG-05.3 | `VL2-EVT-04`, `VL2-EVT-05` (reconstruction) |
| `R-AMT-009` | AG-05 cross-cut | `VL2-EVT-04`, `VL2-EVT-05` (registry) |

All 7 charter Gherkin scenarios are represented; none is reduced.
