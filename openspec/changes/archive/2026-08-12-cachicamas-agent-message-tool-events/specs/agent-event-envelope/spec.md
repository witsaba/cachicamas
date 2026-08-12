# Delta for `agent-event-envelope`

> **Change**: `cachicamas-agent-message-tool-events` · **Milestone**: AG-05 (Layer 2, Wave 1) — doc 0003 lines 518–600
> **Delta target**: `openspec/specs/agent-event-envelope/spec.md` (the AG-04 spec; merged `967d043f`)
> **Identifier convention**: append-only. New requirement ID `R-AEV-012`. New scenario IDs `S-AEV-110`…`S-AEV-112`. Existing `R-AEV-007` and `R-AEV-010` are MODIFIED; their existing scenarios are preserved unchanged where the behavior does not shift. **MODIFIED blocks carry the full requirement + ALL scenarios**, edited in place — partial MODIFIED blocks lose content at archive.
> **Co-closure map** (`0003:2203`): envelope invariant 1 (indexed deltas) is closed by AG-04.3 **+ AG-05.1**; the AG-05.1 half lands on the new `R-AEV-007` text below. Invariants 2, 3, 4 are not touched by AG-05.

## Coverage

| Charter scenarios | Modified requirements | Added requirements | Added scenarios |
|---|---|---|---|
| **7 of 7** in the parent change (counted in the new spec) | 2 (`R-AEV-007`, `R-AEV-010`) | 1 (`R-AEV-012`) | 3 (`S-AEV-110`…`S-AEV-112`) |

This delta extends the AG-04 spec's scope-fence (`S-AEV-090`) from "exactly 4" to "exactly 15" (4 AG-04 + 11 AG-05 kinds), restates the AG-04.3 co-closure of envelope invariant 1 to include AG-05.1, and records that AG-05.2 followed the AG-04.4 extensibility experiment pattern (`S-AEV-092`).

## ADDED Requirements

### Requirement: R-AEV-012 — AG-05 followed the AG-04.4 extensibility experiment pattern

AG-05's 11 new kinds MUST be registered by following the six-step "adding a kind" procedure documented at `event_descriptor.go:13-31`, with no edit to `stream_check.go`, `event_registry_test.go`, or `failure.go`. The AG-04.4 extensibility experiment (`S-AEV-092`) is the documented path AG-05 took — proven by the `S-AMT-081` bite (a 16th scratch kind fails the scope-fence by count) and by `R-AMT-009`'s placement assertion (`S-AMT-080`, all 11 kinds register under `PlacementTurn`).

#### Scenarios
- **S-AEV-110** — Given AG-05's 11 new kinds registered following the six-step procedure, when the validator checks a hand-built sequence containing a `message_start_text` outside an open turn, then it is REJECTED naming the `PlacementTurn` rule — proving the seam AG-04.3 reserved was actually exercised.
- **S-AEV-111** — Given the registry after AG-05, when `stream_check.go` and `event_registry_test.go` are diffed against the AG-04 merge (`967d043f`), then the diffs are empty for both files; AG-05's value is in the substrate's extensibility, demonstrated.
- **S-AEV-112** — Given the six-step procedure doc, when it is read, then it states that any future kind declaring `Terminal: true` MUST be honored by the validator (the W3 latent-trap guard) and that the `CardinalityAtMostOne` seam is reserved for AG-06.

## MODIFIED Requirements

### Requirement: R-AEV-007 — Invariant pin 1: a delta kind has no route to an accumulated-message payload (joint with AG-05.1)

The envelope's public construction surface MUST offer **no route** by which a delta kind can be attached to an accumulated-message payload (`VL2-EVT-12`, envelope invariant 1, `decision.md:173`). A delta carries an index and the new fragment only.

AG-04 registers no message family, so this requirement is a **structural pin on the construction surface**, not an assertion about a message-delta kind AG-05 will introduce. AG-05.1 introduces `message_delta_text` and `message_delta_reasoning`; AG-05.1's per-kind construction surface inherits this pin. Envelope invariant 1 is therefore closed **jointly by AG-04.3 + AG-05.1** (`0003:2203`); AG-05.1's co-closure is asserted by `S-AMT-021` (bite, in the new spec). The pin remains structural rather than instance-based; AG-05.1's two delta kinds do not weaken it — they extend the surface it guards.
(Previously: requirement text referenced AG-05.1 as forthcoming; co-closure is now joint and the bite lives in the AG-05 spec.)

#### Scenarios
- **S-AEV-060** — Given the envelope's public construction surface, when every exported route from a delta kind to a payload is enumerated, then none accepts or produces an accumulated-message payload, and the pin is asserted mechanically rather than by comment.
- **S-AEV-061** — Given the package documentation, when the delta rule is read, then it states that a delta carries an index and the new fragment only, and that the absence of an accumulated-snapshot route is the mechanism AG-05 inherits.

### Requirement: R-AEV-010 — AG-04 registers exactly two families; AG-05 adds two more; the scope-fence now stands at 15

The package MUST register exactly the four event families its milestones own: AG-04 owns run lifecycle (`VL2-EVT-02`) and turn lifecycle (`VL2-EVT-03`); AG-05 owns message lifecycle (`VL2-EVT-04`) and tool execution (`VL2-EVT-05`). It MUST NOT register, stub, placeholder or reserve any kind of the permission (`VL2-EVT-06`), cost (`VL2-EVT-07`), delegation (`VL2-EVT-08`) or compaction (`VL2-EVT-09`) families — those belong to AG-06 (`0003:420`). The scope-fence `S-AEV-090` retightens from "exactly 4" to "exactly 15" in the same commit as the AG-05 kinds land.

The registry and the guard MUST remain **structurally extensible**: AG-06 MUST be able to add a kind by following the documented procedure without editing the validator's rule engine, mirroring the AG-04.4 extensibility experiment (`S-AEV-092`, restated in the new requirement `R-AEV-012`).
(Previously: scope-fence held at "exactly 4"; forbidden-names list included message/tool; permission/cost/delegation/compaction unchanged.)

#### Scenarios
- **S-AEV-090** — Given the package's registered kind set, when it is enumerated, then it contains exactly the run, turn, message and tool lifecycle kinds (15 kinds total: 4 AG-04 + 11 AG-05) and no permission, cost, delegation or compaction kind, under any name.
- **S-AEV-091** — Given the package documentation, when its "adding a kind" procedure is read, then it states the ordered steps a later milestone follows to register a new kind, and states that following them requires no edit to the validator's rule engine.
- **S-AEV-092** — Given the registry, when a new kind is added following the documented procedure in a scratch experiment, then the every-kind-constructible guard and the validator both continue to compile and run without edits to their own logic. Recorded, then reverted.

## REMOVED Requirements

None. No AG-04 requirement is fully superseded by AG-05.

## RENAMED Requirements

None.

## Notes for the archive step

- **MODIFIED blocks preserve all original scenarios.** `S-AEV-060` and `S-AEV-061` (under `R-AEV-007`) and `S-AEV-090`, `S-AEV-091`, `S-AEV-092` (under `R-AEV-010`) are unchanged in their assertions; only the requirement text above each block was edited.
- **ADDED `R-AEV-012` carries 3 scenarios** (`S-AEV-110`…`S-AEV-112`); no overlap with existing scenario IDs (`S-AEV-001`…`S-AEV-102`).
- **`R-AEV-009` (every-kind-constructible guard) is not MODIFIED** — its text already says "every event kind the package registers", which auto-extends to 15 kinds without textual change. The witness table physically grows in the same commit as the new kinds land.
- **Scenario count for this delta**: 2 requirements modified (preserving 5 existing scenarios unchanged), 1 requirement added (with 3 new scenarios). Net spec-scenario count after AG-05 archive: 48 (45 AG-04 + 3 AG-05) on the envelope spec, plus the 15 on the new `agent-message-tool-events` spec.
