# Proposal — Add permission, cost, delegation, and compaction event families

> **Slug**: `cachicamas-agent-protocol-events` · **Milestone**: AG-06 (Layer 2 Wave 1, **closing**) — [doc 0003:602-712](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-06--add-permissioncostdelegation-and-compaction-event-families)
> **Closes**: R-04 — the four families [doc 0001 § 4.3](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md#43-the-agent-event-envelope) marks absent (G1, G10, G7, G3 visible halves)
> **Traces to**: doc 0001 § 7 G1/G3/G7/G10 (`0001:695-705`); ADR 0005 § D1, D3, D4
> **Depends**: AG-04 (merged `967d043f`), AG-05 (merged PR #164) · **Blocks**: AG-07, AG-09, AG-10, AG-16, AG-18, AG-19
> **Delivery**: single PR + `size:exception` (forecast 1500–2400 lines; pre-authorized per braejan's AG-04 standing instruction, Engram `#2957`)
> **Spec prefix**: `R-APE-`/`S-APE-` (matches slug; distinct from `R-AEV-`/`R-AMT-`/`R-AGE-`)
> **Scenario count** (state identically across all downstream artifacts — AG-04 W9): **9 charter → ~14–22 spec + 4 bites**

## Intent

AG-04 shipped 4 kinds (run + turn); AG-05 added 11 (message + tool); **AG-06 closes Layer 2 Wave 1 with 10 more kinds** across the four families doc 0001 § 4.3 marks absent: **permission (3), cost (2), delegation (2), compaction (3)**, judged by `L2C-04`'s "if it is not on the stream, no frontend can render it" (`0003:418`). AG-06 makes events constructible so AG-10/16/18/19 emit rather than invent. **First exercise of `CardinalityAtMostOne`** (AG-04.3 seam at `event_descriptor.go:118-120`; engine reads at `stream_check.go:166-171`). **AG-06.3 is the first consumer of parent identifier outside `NewDelegatedRunStart`** (`run_events.go:60-76`).

## Scope

**In scope.** Register 10 new kinds: `permission_decision_required`, `permission_decision_made`, `permission_resolution_remembered`, `cost_turn`, `cost_session`, `subagent_started`, `subagent_ended`, `compaction_started`, `compaction_finished`, `compaction_failed`. Four per-family files (D8). Scope-fence `S-AEV-090` retightened 15 → 25. Forbidden-names list at `event_registry_test.go:326` retires. New `L2C-06` doc-contract row.

**Out of scope.** Live emission (AG-10, AG-16, AG-18, AG-19). Mechanism owners — AG-06 ships **events only**. No new Go deps; no `event_descriptor.go`/`stream_check.go`/`failure.go`/`sequence.go` edit.

## Capabilities

### New Capabilities

- `agent-protocol-events`: full spec at `openspec/changes/cachicamas-agent-protocol-events/specs/agent-protocol-events/spec.md`. **9 requirements** (`R-APE-001`–`009`), **~14–22 scenarios** + **4 bites**.

### Modified Capabilities

- `agent-event-envelope`: delta spec at `openspec/changes/cachicamas-agent-protocol-events/specs/agent-event-envelope/spec.md`. Witness table extends 15 → 25 (`R-AEV-010` MODIFIED). `R-AEV-012` restated (AG-06's extensibility path). New `L2C-06` row.

## Approach

Extend the registry table (`event.go:91-156`). Follow the 7-step procedure (`event_descriptor.go:13-46`); write `Terminal: false` explicitly on every row (AG-05 S1). **`compaction_failed` declares `Terminal: false`** (D3b) — honors `doc 0001 § 7 G3` "survive interruption"; engine accepts follow-on events but does NOT auto-recover. **`permission_resolution_remembered` declares `CardinalityAtMostOne`** (D4b). **`compaction_finished` carries `[startTurnID, endTurnID]`** (D5b) — turns are the protected unit, not sequence (per-lane) or ordinal. **Cost: 2 kinds** (D1b) `cost_turn`, `cost_session`; every figure carries `CostLabel` (`Estimate`/`Final`) as payload field, token-only (no money — ADR 0005 § D4 per `0003:613`). **Permission: 3 kinds with typed `PermissionOutcome`** (D2a: `AllowOnce`/`AllowAlways`/`Deny`/`ModifyInput`). **Delegation: 2 kinds** (`subagent_started`, `subagent_ended`), parent-linked via `Parent()` (`event.go:362-366`); new constructors set parent. All `PlacementTurn` (or `PlacementRun` for `cost_session`); all `BracketRoleNone`. `make vuln-check` explicit (Engram `#2944`). `golangci-lint cache clean` before lint (`6c821c0a`). All six `sdd-attempt settle` flags (Engram `#2961`).

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/{permission,cost,delegation,compaction}_events.go` | New (4) | Per-family payloads + constructors; `compaction_failed` carries `*Failure`; delegation sets parent |
| `backend/agent/src/agent/{permission,cost,delegation,compaction}_events_test.go` | New (4) | Per-family scenarios + bites; split name + placement per AG-05 W2 |
| `backend/agent/src/agent/event.go` | Modified | `EventKind` const block + `eventRegistry` (10 new rows); move `eventKindEnd` |
| `backend/agent/src/agent/event_registry_test.go` | Modified | Witness 15 → 25; `S-AEV-090` retightens; `BitesByCountOnSixteenthKind` → `BitesByCountOnTwentySixthKind`; forbidden-names list retires |
| `backend/agent/src/agent/doc.go` | Modified | New `L2C-06` row + per-family prose |
| `backend/agent/src/agent/doc_contract_guard_test.go` | Modified | Guard iterates new row count |
| `openspec/changes/cachicamas-agent-protocol-events/specs/agent-protocol-events/spec.md` | New | `R-APE-`/`S-APE-` 9 reqs + ~14–22 scenarios + 4 bites |
| `openspec/changes/cachicamas-agent-protocol-events/specs/agent-event-envelope/spec.md` | New | Delta: witness 15→25, `R-AEV-012` restated |
| `backend/agent/src/agent/event_descriptor.go` | UNCHANGED | `CardinalityAtMostOne` seam exists from AG-04.3 |
| `backend/agent/src/agent/stream_check.go` | UNCHANGED | Engine reads descriptor data; AG-04.4 `S-AEV-092` + AG-05 `R-AEV-012` proven path |
| `backend/agent/src/agent/failure.go` | UNCHANGED | `*Failure` reused; no new category |
| `backend/agent/src/agent/sequence.go` | UNCHANGED | Span identity is payload-side (turn bracket) |
| `backend/agent/{go.mod,go.sum,Makefile,.golangci.yml}` | UNCHANGED | No new deps, no tooling change |
| `backend/agent/src/ai/**` | UNCHANGED | Layer 1 untouched |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Spec prefix `R-APE-` collides with future AG-23 | Low | AG-23 uses `R-L3R-` (Layer 3); post-Wave-3 |
| Cost shape (D1b 2 kinds) overlap `cost_session` ↔ cumulative-on-turn-end | Medium | Cite AG-16.1; bite-test both; spec disambiguates |
| `compaction_failed` `Terminal: false` misread as auto-recovery | Medium | Spec: engine accepts follow-on events but does NOT synthesize recovery |
| `resolution_remembered` cardinality conflict (per-tool vs per-stream) | Medium | `CardinalityAtMostOne` is per-stream; spec scenario separately asserts per-tool |
| Span identity (D5b) couples turn IDs to compaction | Low | Turn ID already in envelope (`S-AEV-004`) |
| Single PR (D7) vs chained | Medium | AG-05 precedent; no AG-05.3-style cross-family property |
| Review budget 2–3× over (forecast 1500–2400 vs 1000) | High | `size:exception` pre-authorized open-endedly (AG-04 standing instruction) |
| Scenario count drift (AG-04 W9) | Low | State identically: **9 charter → ~14–22 spec + 4 bites** |
| `sdd-archive` worktree-vs-main sequencing (Engram `#2963`) | Low | Archive runs in worktree, commits to PR branch |
| `golangci-lint cache` stale finding (`6c821c0a`) | Low | `cache clean` before each lint gate |
| `make vuln-check` not in `make all` (Engram `#2944`) | Low | Run `vuln-check` explicitly |
| `sdd-attempt settle` flags incomplete (Engram `#2961`) | Low | All 6: `--outcome --harness-disposition --evidence-revision --diagnosis --cleanup-evidence --process-evidence` |
| `test-driven-development` skill gap (Engram `#2962`) | Low | Discipline forwarded inline; verify loads `sdd-verify/strict-tdd-verify.md` |
| Vacuous reconstruction helper in delegation tree-walk (AG-05 W1) | Low | If helper added, bite-test RED before GREEN |

## Rollback Plan

Single PR. Revert the AG-06 merge commit. 10 kinds removed; `S-AEV-090` reverts to "exactly 15"; forbidden-names list restored; `L2C-06` row removed. Substrate (`event_descriptor.go`/`stream_check.go`/`failure.go`/`sequence.go`) byte-unchanged — rollback trivial. `R-AEV-012` reverts to AG-05 wording. No persisted state, no schema, no migration, no `go.mod`/`go.sum` unwind. Cite AG-04 + AG-05 archive patterns.

## Dependencies

**Hard**: AG-04 (`967d043f`), AG-05 (`6b4a3468`). **Soft downstream**: AG-07, AG-09, AG-10, AG-16, AG-18, AG-19. **Toolchain unchanged**: Go 1.26.3, `golangci-lint v2.9.0`. **No new Go deps** (`openspec/AGENTS.md` rule 5).

## Success Criteria

- [ ] 10 new kinds registered (3 permission + 2 cost + 2 delegation + 3 compaction), constructible, validated, guarded
- [ ] `agent-protocol-events` spec promoted: 9 reqs + ~14–22 scenarios + 4 bites (stated identically in proposal, tasks, apply-progress)
- [ ] `agent-event-envelope` delta: `S-AEV-090` retightens 15 → 25; `R-AEV-012` restated; new `L2C-06` row
- [ ] `cost_turn` + `cost_session` are 2 distinct kinds (D1b); each figure carries `CostLabel`
- [ ] `permission_decision_made` carries typed `PermissionOutcome` (4 values, zero not a member)
- [ ] `permission_resolution_remembered` declares `CardinalityAtMostOne`
- [ ] `compaction_failed` declares `Terminal: false`; engine accepts follow-on events
- [ ] `compaction_finished` carries `[startTurnID, endTurnID]`
- [ ] `subagent_started`/`subagent_ended` set `Parent()` on envelope
- [ ] `event_descriptor.go`/`stream_check.go`/`failure.go`/`sequence.go` byte-unchanged
- [ ] Forbidden-names list retired; `go.mod`/`go.sum` byte-unchanged
- [ ] `make test`, `make lint` (after cache clean), `make build`, `make vuln-check` all green
- [ ] All six `sdd-attempt settle` flags passed; `sdd-archive` runs in worktree
- [ ] Single PR merged with `size:exception` documented; prefix `R-APE-`/`S-APE-` used consistently

## Notes for the Following Phases

- **`sdd-spec`**: write `specs/agent-protocol-events/spec.md` + `agent-event-envelope` delta. Restate 9 charter as Given/When/Then; do not reduce. Witness extension, `CardinalityAtMostOne` exercise, parent-identifier consumption, recovery-after-failure bite are spec requirements. State count "9 → ~14–22 + 4 bites" identically. No envelope-invariant co-closure.
- **`sdd-design`**: D1b/D2a/D3b/D4b/D5b/D6/D7/D8 **locked at proposal** — design cannot silently re-open. Mechanics (file naming, helper signatures, exact turn-bracket wire format) are design's call.
- **`sdd-tasks`**: five phases matching node graph — AG-06.1 permission, AG-06.2 cost, AG-06.3 delegation, AG-06.4 compaction, AG-06.5 guard (depends on all four). Single PR, five commits. State scenario count identically. Restate `sdd-phase-common.md` § E guard lines (`Decision needed: No`, `Chained PRs: No`, `400-line budget risk: High`, `Forecast: 1500–2400`).
