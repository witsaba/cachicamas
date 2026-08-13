# Proposal: AG-07 — Build the one-turn walking skeleton

## Intent

AG-07 is the FIRST milestone where Layer 2 emits events from a live loop. AG-04/05/06's stream-validator tests fed the validator HAND-BUILT events. AG-07 owns the producer side of AG-01's carrier decision. Per the AG-07 charter (doc 0003:773): "the single most important node in this document: the first time Layer 1 and Layer 2 meet." Without AG-07, AG-08 (hook seam), AG-09 (tool scheduler), AG-11 (typed termination) have nothing to wrap.

## Scope

### In scope
- Stateless turn runner `func Turn(...)` (D1) — thinnest end-to-end path
- Event emission via `chan<- *Event` parameter (D2, AG-01 carrier)
- Finish reason on the return value (D3)
- Reasoning + text interleave via direct `agenttest.Script` (D4)
- Pass-through `ctx` (D5)
- AG-07.1 (3) + AG-07.2 (2) scenarios — **5 charter → 8-15 spec** (AG-04 W9 carry)

### Out of scope
- Tools, hooks, errors beyond typed pass-through, permission, retry, context-check, cost events (deferred to AG-08…AG-18)
- Any envelope/descriptor/validator/substrate edit (4th consecutive "substrate untouched" milestone)

## Capabilities

### New
- `agent-loop-skeleton`: the one-turn walking skeleton. New `openspec/specs/agent-loop-skeleton/spec.md` (full spec).

### Modified
- None.

## Approach

`TurnOptions` struct has trivial/zero fields for AG-07. `Turn` flow: (1) `CheckEmit(run_start)`; (2) `CheckEmit(turn_start)`; (3) `provider.Stream(ctx, req)`; (4) drain, translating each `ai.Event` to a bracket (`MessageStart{Text,Reasoning}` → `MessageDelta{Text,Reasoning}` → `MessageEnd{Text,Reasoning}`); (5) on `ai.Completion`, capture `FinishReason`; (6) `CheckEmit(turn_end)`; (7) `CheckEmit(run_end)` (U3 path a: one run per turn); (8) close `sink`; (9) return `(msg, finish, err)`. Assistant message assembled from accumulated deltas via the AG-05.3 reconstruction pattern; byte-exact reasoning round-trip token preserved (AG-07.2 #2). Strict TDD: AG-07.1 #3 bites RED first (drop-a-delta, double-a-delta) BEFORE the GREEN property test (AG-05 W1 carry).

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/loop.go` | NEW | `func Turn(...)` ~80-150 lines |
| `backend/agent/src/agent/loop_test.go` | NEW | AG-07.1 + AG-07.2 scenarios, ~250-400 lines |
| `backend/agent/src/agent/{event,event_descriptor,stream_check,failure,sequence,run_events,turn_events,message_text,message_reasoning,permission_events,cost_events,delegation_events,compaction_events,tool_event}.go` | UNTOUCHED | substrate preserved |
| `backend/agent/src/agent/{import_boundary,ambient_authority}_test.go` | UNTOUCHED | AG-03 guards |
| `openspec/specs/agent-loop-skeleton/spec.md` | NEW | full spec for new capability |
| `openspec/changes/cachicamas-agent-loop-skeleton/{exploration,proposal}.md` | NEW | phase artifacts |
| `backend/agent/go.mod`, `go.sum` | UNTOUCHED | no new deps |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| Loop surface conflict with AG-13's later `Harness` value | Medium | D1: function form; AG-13 wraps `Turn` without signature change |
| `TurnEnd` carries no `FinishReason` field | Low | D3: finish reason via return value |
| Review budget 1000 lines exceeded 1.5× | Low | Forecast 400-700; `size:exception` pre-authorized |
| Vacuous reconstruction helper (AG-05 W1) | Low | Two failing bites (drop, double) BEFORE GREEN |
| `test-driven-development` skill missing on disk | Low | Cite `openspec/AGENTS.md` `## Strict TDD is on` block directly |
| Scenario count drift (AG-04 W9) | Low | State `5 charter → 8-15 spec` identically across proposal, tasks, apply-progress, verify-report |
| `MessageID` minting collisions across turns (R8) | Low | Per-turn fresh `ai.Message` via accumulated fragments |
| `ctx` cancellation mid-stream (R9) | Low | AI-20 mid-stream physics already proven by fake |

## Rollback plan

Revert PR. New files only: `loop.go` + `loop_test.go` + spec. Substrate byte-unchanged (4th "substrate untouched" bet); clean revert. No data migration, no schema change.

## Dependencies

- AI-40 frozen Layer 1 surface (satisfied 2026-08-10)
- AI-21 fake provider (`backend/agent/src/agenttest/fake_provider.go`)
- AI-22 stream kit (`backend/agent/src/agenttest/stream_kit_*.go`)
- AG-04 envelope + ordering invariants (PR #163)
- AG-05 message + tool families (PR #164)
- AG-06 permission/cost/delegation/compaction families (PR #166, today)

## Success criteria

- [ ] `make test` green in `backend/agent/` with `-race` (no new failures)
- [ ] `make lint` green (after `golangci-lint cache clean` per AG-04 precedent)
- [ ] `make build` green
- [ ] `make vuln-check` clean ("No vulnerabilities found", NOT in `make all`)
- [ ] AG-07's `loop.go` ≥ 80% test coverage (AG-04 W8 carry)
- [ ] Every-kind-constructible guard still passes at 25 kinds (AG-07 adds zero kinds)
- [ ] Import boundary guard + ambient authority guard still pass
- [ ] `go.mod` / `go.sum` byte-identical to main
- [ ] 5 charter → 8-15 spec scenarios, stated identically across proposal, tasks, apply-progress, verify-report
