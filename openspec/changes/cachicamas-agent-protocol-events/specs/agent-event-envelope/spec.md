# Delta for `agent-event-envelope`

> **Change**: `cachicamas-agent-protocol-events` · **Milestone**: AG-06 (Layer 2 Wave 1, **closing**) — doc 0003 lines 602–712
> **Delta target**: `openspec/specs/agent-event-envelope/spec.md` (the AG-04 spec, last amended by AG-05 archive at `6b4a3468`)
> **Identifier convention**: append-only. New requirement IDs `R-AEV-013`, `R-AEV-014`, `R-AEV-015`. New scenario IDs `S-AEV-120`…`S-AEV-124`. Existing `R-AEV-010` is MODIFIED; its existing scenarios are preserved unchanged where the behavior does not shift. **MODIFIED blocks carry the full requirement + ALL scenarios**, edited in place — partial MODIFIED blocks lose content at archive.
> **Scope-fence retightening**: `S-AEV-090` now stands at "exactly 25" (4 AG-04 + 11 AG-05 + 10 AG-06); `R-AEV-012` is restated to record that AG-06 followed the AG-04.4 extensibility pattern. The forbidden-names list at `event_registry_test.go:326` retires.
> **Co-closure map** (`0003:2203`): AG-06 touches no envelope invariant directly; it co-closes invariant 2 (explicit nesting) **partially** by giving `subagent_started`/`subagent_ended` consumers of the parent identifier field. AG-19.1 closes the invariant fully. Invariants 1, 3, 4 are untouched by AG-06.

## Coverage

| Charter scenarios | Modified requirements | Added requirements | Added scenarios |
|---|---|---|---|
| **9 of 9** in the parent change (counted in the new spec) | 1 (`R-AEV-010`) | 3 (`R-AEV-013`, `R-AEV-014`, `R-AEV-015`) | 5 (`S-AEV-120`…`S-AEV-124`) |

This delta extends the AG-04/AG-05 scope-fence (`S-AEV-090`) from "exactly 15" to "exactly 25", adds the `L2C-06` row to the doc-contract guard, restates `R-AEV-012` to record AG-06 as the third kind-set to follow the extensibility pattern, and adds three new requirements documenting the protocol families' relationship to the envelope invariants.

## ADDED Requirements

### Requirement: R-AEV-013 — AG-06's registry holds exactly 25 kinds; the scope-fence now stands at 25

The package MUST register exactly the eight event families its milestones own across Wave 1: AG-04 owns run lifecycle (`VL2-EVT-02`) and turn lifecycle (`VL2-EVT-03`); AG-05 owns message lifecycle (`VL2-EVT-04`) and tool execution (`VL2-EVT-05`); AG-06 owns permission (`VL2-EVT-06`), cost (`VL2-EVT-07`), delegation (`VL2-EVT-08`), and compaction (`VL2-EVT-09`). The registry MUST hold exactly 25 kinds (4 AG-04 + 11 AG-05 + 10 AG-06) — no stub, placeholder, or reservation. The scope-fence `S-AEV-090` retightens from "exactly 15" to "exactly 25" in the **same commit** as the AG-06 kinds land. The forbidden-names list at `event_registry_test.go:326` (the AG-04/AG-05 forward-fence against AG-06's families) retires — permission, cost, delegation, and compaction are AG-06's own.

#### Scenarios

- **S-AEV-120** — Given the package's registered kind set, when it is enumerated, then it contains exactly **25 kinds** (4 AG-04 + 11 AG-05 + 10 AG-06) — the four protocol families included — and no further kind of any name; the forbidden-names list is empty or absent.
- **S-AEV-121** — Given the every-kind-constructible guard, when it runs, then it constructs at least one instance of every registered kind (25 total) through the public surface from an external test package; the forbidden-names check does not flag any AG-06 family name.

### Requirement: R-AEV-014 — L2C-06 doc-contract row references the four protocol families

The doc-contract guard's `expectedLayer2ContractRows` table MUST carry an `L2C-06` row added in the same commit as the AG-06 kinds and the `L2C-06` row in `doc.go`. The row's text states that the four protocol families (permission, cost, delegation, compaction) are constructible on the event stream and that the per-family semantics belong in `doc.go` prose, not in the guarded row, per `R-AGP-002`'s closed-amendment rule. The guardian of this amendment is the same `doc_contract_guard_test.go` that guards `L2C-01`..`L2C-05`.

#### Scenarios

- **S-AEV-122** — Given the `expectedLayer2ContractRows` table in `doc_contract_guard_test.go`, when it is read, then it contains 6 rows (`L2C-01`..`L2C-06`) — the new `L2C-06` row is present, in order, with row text referencing the four protocol families.
- **S-AEV-123** — Given a scratch edit that appends an `L2C-06` row to `doc.go` without adding its entry to `expectedLayer2ContractRows`, when the doc-contract guard runs, then it FAILS naming the unexpected row — the closed-amendment rule is observed, not bypassed. RED-recordable.

### Requirement: R-AEV-015 — Protocol family kinds follow the AG-04.1 envelope invariants

Every AG-06 kind MUST follow the AG-04.1 envelope invariants: derived kind from payload (`R-AEV-001`); identity fields (run, turn, parent) readable from an external package (`R-AEV-001`/`R-AEV-003`); and per-lane ordering 1-based and contiguous (`R-AEV-002`). `subagent_started` and `subagent_ended` are the **first** non-`NewDelegatedRunStart` consumers of the parent identifier field (`event.go:362-366`, `R-AEV-003`) — the field has existed from AG-04.1 and AG-06.3 is its first extension. AG-06 closes invariant 2 (explicit nesting) **partially**; AG-19.1 closes it fully. The `CardinalityAtMostOne` seam reserved at AG-04.3 (`event_descriptor.go:103-120`) is exercised for the first time by `permission_resolution_remembered` (R-APE-003).

#### Scenarios

- **S-AEV-124** — Given the 10 new AG-06 kinds registered following the seven-step procedure, when the every-kind-constructible guard constructs each through the public surface and inspects them, then every kind's identity fields (run, turn, parent) are readable from an external package; `subagent_started`/`subagent_ended` distinguishably report `(parentID, true)`; `permission_resolution_remembered` declares `CardinalityAtMostOne`; `compaction_failed` declares `Terminal: false` — the three AG-06 substrate exercises are mechanically asserted.

## MODIFIED Requirements

### Requirement: R-AEV-010 — AG-04/AG-05/AG-06 register exactly six families; the scope-fence now stands at 25

The package MUST register exactly the eight event families its milestones own across Wave 1: AG-04 owns run lifecycle (`VL2-EVT-02`) and turn lifecycle (`VL2-EVT-03`); AG-05 owns message lifecycle (`VL2-EVT-04`) and tool execution (`VL2-EVT-05`); AG-06 owns permission (`VL2-EVT-06`), cost (`VL2-EVT-07`), delegation (`VL2-EVT-08`), and compaction (`VL2-EVT-09`). The registry MUST hold exactly 25 kinds (4 AG-04 + 11 AG-05 + 10 AG-06) — no stub, placeholder, or reservation outside these eight families. The scope-fence `S-AEV-090` retightens from "exactly 15" to "exactly 25" in the same commit as the AG-06 kinds land; the forbidden-names list at `event_registry_test.go:326` retires (permission, cost, delegation, compaction are now AG-06's own).

The registry and the guard MUST remain **structurally extensible**: AG-07 onward MUST be able to add a kind by following the documented procedure without editing the validator's rule engine, mirroring the AG-04.4 extensibility experiment (`S-AEV-092`, restated in `R-AEV-012`; AG-06 followed the same path, recorded by `R-AEV-015`).
(Previously: scope-fence held at "exactly 15"; AG-06's four families were forbidden; AG-06 retightens to "exactly 25" and the forbidden-names list retires.)

#### Scenarios

- **S-AEV-090** — Given the package's registered kind set, when it is enumerated, then it contains exactly the run, turn, message, tool, permission, cost, delegation and compaction lifecycle kinds (25 kinds total: 4 AG-04 + 11 AG-05 + 10 AG-06) and no further kind of any name.
- **S-AEV-091** — Given the package documentation, when its "adding a kind" procedure is read, then it states the ordered steps a later milestone follows to register a new kind, and states that following them requires no edit to the validator's rule engine.
- **S-AEV-092** — Given the registry, when a new kind is added following the documented procedure in a scratch experiment, then the every-kind-constructible guard and the validator both continue to compile and run without edits to their own logic. Recorded, then reverted.

### Requirement: R-AEV-012 — AG-04/AG-05/AG-06 followed the AG-04.4 extensibility experiment pattern

AG-04's 4 kinds, AG-05's 11 kinds, and AG-06's 10 kinds MUST all be registered by following the seven-step "adding a kind" procedure documented at `event_descriptor.go:13-46`, with no edit to `stream_check.go`, `event_registry_test.go`, or `failure.go`. The AG-04.4 extensibility experiment (`S-AEV-092`) is the documented path all three kind-sets took — proven by AG-04's `S-AEV-082` bite, AG-05's `S-AMT-081` bite (a 16th scratch kind fails the scope-fence by count), AG-06's `S-APE-081` bite (a 26th scratch kind fails by count), AG-05's `R-AMT-009` placement assertion (`S-AMT-080`, all 11 AG-05 kinds register under `PlacementTurn`), and AG-06's `R-AEV-015` invariant compliance assertion (`S-AEV-124`, AG-06's parent-identifier and `CardinalityAtMostOne` exercises).

#### Scenarios

- **S-AEV-110** — Given AG-05's 11 new kinds registered following the seven-step procedure, when the validator checks a hand-built sequence containing a `message_start_text` outside an open turn, then it is REJECTED naming the `PlacementTurn` rule — proving the seam AG-04.3 reserved was actually exercised.
- **S-AEV-111** — Given the registry after AG-06 merges, when `stream_check.go`, `event_registry_test.go`, and `failure.go` are diffed against the AG-04 merge (`967d043f`), then the diffs are empty for all three files; AG-06's value is in the substrate's extensibility, demonstrated for the third time.
- **S-AEV-112** — Given the seven-step procedure doc, when it is read, then it states that any future kind declaring `Terminal: true` MUST be honored by the validator (the W3 latent-trap guard), that `CardinalityAtMostOne` is exercised by AG-06's `permission_resolution_remembered` (R-APE-003, `S-APE-082`), and that future kinds MAY opt into the same seam via the descriptor row.

## REMOVED Requirements

None. No AG-04 or AG-05 requirement is fully superseded by AG-06.

## RENAMED Requirements

None.

## Notes for the archive step

- **MODIFIED blocks preserve all original scenarios.** `S-AEV-090`, `S-AEV-091`, `S-AEV-092` (under `R-AEV-010`) and `S-AEV-110`, `S-AEV-111`, `S-AEV-112` (under `R-AEV-012`) are unchanged in their assertions; only the requirement text above each block was edited.
- **ADDED `R-AEV-013` carries 2 scenarios** (`S-AEV-120`, `S-AEV-121`); `R-AEV-014` carries 2 (`S-AEV-122`, `S-AEV-123`); `R-AEV-015` carries 1 (`S-AEV-124`). Total 5 new scenarios; no overlap with existing scenario IDs (`S-AEV-001`…`S-AEV-112`).
- **No scrap-from-AG-04.** `R-AEV-001` through `R-AEV-009` and `R-AEV-011` retain their original text. `R-AEV-007` is NOT MODIFIED — AG-06 touches no envelope invariant 1 directly. Only `R-AEV-010` (scope-fence) and `R-AEV-012` (extensibility pattern restated) carry edits.
- **Scenario count for this delta**: 2 requirements modified (preserving 6 existing scenarios unchanged), 3 requirements added (with 5 new scenarios). Net spec-scenario count after AG-06 archive: 56 on the envelope spec (45 AG-04 + 3 AG-05 + 5 AG-06 + 3 preserved across MODIFIED = 56), plus the 19 on the new `agent-protocol-events` spec (15 spec + 4 bites).
- **The `L2C-06` row text** to be added to `doc.go` and `expectedLayer2ContractRows` in the same commit (per `R-AGP-002`'s closed-amendment rule) is named in `R-AEV-014` (`S-AEV-122`); the exact byte text is the design phase's call.
- **Forbidden-names list retirement** is part of `R-AEV-010`'s MODIFIED text — the test file at `event_registry_test.go:326` retires the four-name list and the scope-fence's forbidden-names check either becomes empty or is folded into the kind-count first action.
- **Bite-by-bite cross-references**: AG-06's four bites (`S-APE-081` scope-fence, `S-APE-082` per-tool cardinality, `S-APE-083` token-only mechanical pin, `S-APE-084` recovery-after-failure) are surfaced here as part of the spec discipline; the bites themselves live in `openspec/specs/agent-protocol-events/spec.md` and close on the AG-06 apply phase.
