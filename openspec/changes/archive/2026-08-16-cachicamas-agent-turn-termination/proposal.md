# Proposal: AG-11 — Complete turn termination and typed failure reporting

> **Change**: `cachicamas-agent-turn-termination` · **Milestone**: AG-11 (Layer 2 Wave 2, milestone 11 of 24; doc 0003 lines 1113–1176)
> **Branch**: `feat/agent-layer2-wave2-ag11` based at `main` `b8eb7d75`
> **Artifact store**: hybrid (Engram + filesystem)
> **Pre-authorized**: `size:exception` against the 1000-line PR review budget (forecast 900–1500 lines; the budget is extendable for this milestone alone, which carries only AG-11)
> **TDD**: strict, RED-first (test runner `cd backend/agent && make test`)
> **Closes**: R-08's typed mid-stream path · R-05 envelope invariant 4 (jointly with AG-04.3, per `agent-event-envelope/spec.md:247`) · R-06 decide-retry · R-15 loop-never-retries · R-18 seam 7 · doc 0003:2167–2168.
> **Exploration**: `exploration.md` · Engram `sdd/cachicamas-agent-turn-termination/explore` (#3059).

## Intent

The loop terminates every turn as if it succeeded, and terminates a failing turn silently.

1. **Every finish reason collapses to one outcome.** `finalize()` hardcodes `NewTurnEnd(t.runID, t.turnID, TurnOutcomeFinished, nil)` and `NewRunEnd(t.runID, RunOutcomeCompleted, nil)` (`loop.go:613,617`) regardless of `t.finish`. A refusal, a content-filter stop, a length cut-off, a paused turn and an unknown finish reason all reach a consumer as `turn_end(finished)`. The only dispatch on `ai.FinishReason` anywhere in the `agent` package is the equality check at `loop.go:258`, which drives scheduling, not outcome typing. `finish_reason.go`'s own package doc calls this collapsing "a loop-termination defect above Layer 1, not a cosmetic gap".
2. **The mid-stream-fatal path emits nothing.** At `loop.go:270-276` the loop drains the provider, closes `sink`, and returns `ai.Message{}, 0, turn.fatal` — **no `turn_end`, no `run_end`**. A frontend draining `sink` sees the channel close abruptly with no typed event explaining why, and the partial assistant content already accumulated in `turnAccumulator` is discarded. `NewTurnEnd(..., TurnOutcomeAborted, failure)` is dead code for this path today.
3. **The typed-failure surface cannot carry the discriminator.** `agent.Failure` exposes only `Category()` (`failure.go:44`), `Delivery()` (`:54`), `Retryable()` (`:64`) and `Unwrap()` (`:75`). `ai.Failure.PartialOutput()` exists at `provider_failure.go:515` but is unreachable through Layer 2, so AG-11.2's charter phrase "carrying category, retryability, and partial output" is unsatisfiable on the current surface.

AG-11 decides completion correctly for every finish reason and reports every provider failure as a typed value upward — **deciding nothing about retries**.

## Scope

### In
- **AG-11.1** — extend `TurnOutcome` (`turn_events.go:72-89`, two members today) to **one distinct member per `ai.FinishReason`**; `outcomeForFinish(ai.FinishReason) TurnOutcome` exhaustive dispatch in `loop.go`; `finalize()` calls it instead of hardcoding `TurnOutcomeFinished`
- **AG-11.1** — a third instance of the exhaustiveness-pin idiom, this time inside `package agent_test`, walking `0..255` as `finish_reason_test.go:277-320` and `agenttest/conformance_capabilities.go:247,265` already do
- **AG-11.2** — mid-stream-fatal path emits `NewTurnEnd(run, turn, TurnOutcomeAborted, failure)` + `NewRunEnd(run, RunOutcomeFailed, failure)` on `sink` before `closeSink`, and returns the reconstructed partial message instead of `ai.Message{}`
- **AG-11.2** — `agent.Failure.PartialOutput() bool`, nil-safe, mirroring `Category`/`Delivery`/`Retryable`
- **AG-11.2** — the never-retries pin: `len(provider.Requests()) == 1` on every failing turn (`agenttest/fake_provider.go:157-161`)
- Widening `filterOutLoopFiles` (`loop_test.go:831-871`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907-943`) in sync

### Out
- **Acting on retryability** — AG-15. The loop reports `Retryable()`; it never retries.
- **Acting on pause** — AG-13. The harness resumes; AG-11 makes pause *visible as its own outcome* and stops there.
- **Pairing enforcement for turns failing mid-tool-calls** — jointly AG-11.2 + AG-12's history boundary and orphan synthesis. This change owns only the dispatch and the typed emission.
- **The `_ = sched.Schedule(...)` tool-result-loss gap at `loop.go:265`** — pre-existing, adjacent to AG-11's edit site, explicitly **untouched**. Care is required not to silently "fix" or further entrench it.
- New `EventKind`, `event_descriptor.go` or `event_registry_test.go` edit — `failure.go:6-7` already settled this: "AG-04 registers no separate error kind; failures ride the typed outcomes".

## Resolved design fork — 1:1, not a subset

The exploration left the extended vocabulary's cardinality open (its risk 5). **Resolved: the literal reading of AG-11.1's Gherkin, which is the acceptance contract — "each maps to a distinct typed turn outcome under an exhaustive dispatch" (doc 0003:1140).** Every one of the seven `ai.FinishReason` members gets its own `TurnOutcome` member. The doc 0003:2167 checklist row ("refusal, pause, and unknown finish reasons produce three distinct behaviors") is a weaker **subset** requirement that a 1:1 mapping satisfies automatically. A collapsed subset is NOT proposed. `sdd-spec` and `sdd-design` inherit this decision; they do not re-open it.

Planned members (8 total + `turnOutcomeLimit`): `TurnOutcomeFinished` (existing, ← `FinishReasonStop`), `TurnOutcomeToolCalls`, `TurnOutcomeLengthLimited`, `TurnOutcomeContentFiltered`, `TurnOutcomeRefused`, `TurnOutcomePaused`, `TurnOutcomeUnknown`, `TurnOutcomeAborted` (existing, the mid-stream-fatal path — not a finish reason).

## Capabilities

### New
- **`agent-turn-termination`** — finish-reason dispatch + typed failure upward. IDs `R-ATT-0NN` / `S-ATT-0NN`, bites `S-TTB-0NN`. Becomes `openspec/specs/agent-turn-termination/spec.md` at archive.

### Modified
- **`agent-loop-skeleton`** — `R-LSK-004` (`spec.md:57`) names `turn_events.go` and `failure.go` in its normative "SHALL NOT modify" substrate list; AG-11 must edit both, so that requirement takes a delta. `R-LSK-001`'s return contract also changes: the fatal path now returns the reconstructed partial `msg`, not `ai.Message{}`.
- **`agent-event-envelope`** — `R-AEV-008` (`spec.md:126`) is invariant 4's requirement and already records "4 — typed errors | AG-04.3 **+ AG-11.2**" (`spec.md:247`). The delta adds the partial-output discriminator to the typed-failure surface and the loop-level emission obligation.
- **`agent-permission-protocol`** — `R-APP-012` (`spec.md:118`) names `failure.go` in AG-10's own no-edit list. The delta scopes that clause to AG-10 and records AG-11's release of it.

## Approach (exploration Approach 1, recommended)

Extend in place. `TurnOutcome`'s bound-check `e.outcome >= turnOutcomeLimit` (`turn_events.go:120`) and its `String()` switch (`:93-104`) are private to `turn_events.go`; Go permits declaring constants of a named type elsewhere, but the validator would reject and `String()` would mis-render anything appended outside that file. **`turn_events.go` must be edited — this is structural, not preference.** `NewTurnEnd`'s "Failure iff Aborted" rule (`:123-128`) needs **zero change**: only `Aborted` ever carries one.

`loop.go`'s fatal branch (`:270-276`) is rewritten to (a) wrap `t.fatal` via `agent.NewFailure` when it type-asserts to `*ai.Failure` from `ev.ErrorPayload()`; (b) reconstruct the partial message from `t.textBracket`/`t.reasoningBracket` the way `finalize()` already does (`:627-648`); (c) emit the two closing brackets before `closeSink`; (d) return from `provider.Stream` exactly once. AG-11.2's scenario tests reuse `agenttest/conformance_terminal.go:105-161`'s existing script (start, delta, end, then `ai.ErrorEvent(ai.MidStreamFailure(..., true))`) directly against `agent.Turn`.

**Rejected — Approach 2** (keep two members, add a `FinishReason()` accessor on `TurnEnd`): does not satisfy "distinct typed outcome"; a consumer still switches on `ai.FinishReason` to tell refusal from pause. **Rejected — Approach 3** (parallel outcome type in a new file): `TurnEnd.validate()` learns the new field's rules anyway, so `turn_events.go` is edited regardless, for two parallel vocabularies on one event.

## Deliberate substrate deviation (recorded, not silent)

`filterOutLoopFiles`/`filterOutLoopHookFiles` currently exclude 13 filenames, all of them files a milestone **added** (`loop_test.go:849-863`). AG-11 is the **first milestone to widen the filters for an EXISTING substrate file that is genuinely modified** — `turn_events.go` and `failure.go`. AG-08, AG-09 and AG-10 only ever appended new files. This is a new *kind* of exception and is recorded here as a deliberate, unavoidable deviation, with the same explicitness AG-09/AG-10 used in `apply-progress.md`. The reason it is unavoidable is Go const-block, `validate()` and `String()` locality (above) — not convenience. The widening MUST be exact-filename suffixes with no wildcard, prefix, or directory-level relaxation, per `R-APP-012`'s own discipline, and both filters MUST stay in sync.

## Requirement IDs and traceability

| Planned ID | Obligation | Traced to |
|---|---|---|
| `R-ATT-001` | `TurnOutcome` grows to one distinct member per `ai.FinishReason`; the zero value stays a non-member; `Failure` iff `Aborted` unchanged | AG-11.1#1 (`0003:1137-1141`) |
| `R-ATT-002` | Exhaustive `outcomeForFinish` dispatch; `finalize()` consumes it | AG-11.1#1; R-08 |
| `R-ATT-003` | Agent-level exhaustiveness pin over `0..255` — an eighth `ai.FinishReason` fails this suite too | AG-11.1#1 second clause |
| `R-ATT-004` | Refusal and pause diverge: refusal = turn over; pause = resumption expected, visible as its own outcome, received content replayed verbatim | AG-11.1#2 (`0003:1143-1147`); doc 0003:2167 |
| `R-ATT-005` | Mid-stream terminal error emits `turn_end(aborted, failure)` + `run_end(failed, failure)` before sink close | AG-11.2#1 (`0003:1158-1162`); R-05 invariant 4 (`0003:2162,2203`); R-18 seam 7 (`0003:2216`) |
| `R-ATT-006` | `agent.Failure.PartialOutput()` — nil-safe, delegating to `provider_failure.go:515` | AG-11.2#1; `R-AEV-008` |
| `R-ATT-007` | The failing turn's partial assistant content reaches the caller, not `ai.Message{}` | AG-11.2#1 ("and any partial assistant content") |
| `R-ATT-008` | The loop never issues a second provider call: `len(provider.Requests()) == 1` | AG-11.2#2 (`0003:1164-1167`); R-06 decide-retry, R-15 (`0003:2204,2213`) |
| `R-ATT-009` | Substrate deviation recorded; both filters widened by exact filename only | this proposal's deviation section |

`invariant_pin_test.go:1-8` already declares "invariant 4 closes jointly with AG-11.2". `R-ATT-006`'s scenario belongs in that file's family so the joint-closure claim stays auditable.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/agent/turn_events.go` | **Modified (substrate)** | `TurnOutcome` const block `:74-89`, `String()` `:93-104`; `validate()` `:119-130` and `NewTurnEnd` `:135-147` unchanged |
| `backend/agent/src/agent/failure.go` | **Modified (substrate)** | `PartialOutput() bool` added after `Retryable()` `:64-69` |
| `backend/agent/src/agent/loop.go` | Modified | `outcomeForFinish`; fatal branch `:270-276` rewritten; `finalize()` `:613,617` consumes the dispatch |
| `backend/agent/src/agent/loop_test.go` `:831-871` | Modified | `filterOutLoopFiles` widened (exact filenames) |
| `backend/agent/src/agent/loop_hook_test.go` `:907-943` | Modified | `filterOutLoopHookFiles` widened, kept byte-in-sync |
| `backend/agent/src/agent/invariant_pin_test.go` | Modified | `PartialOutput` scenario, invariant-4 joint closure |
| New AG-11 test files (`package agent_test`) | New | Exhaustiveness pin + AG-11.1/AG-11.2 scenarios; each added to both filters |
| `event_descriptor.go`, `event.go`, `stream_check.go`, `sequence.go`, `run_events.go`, `tool_event.go`, `event_registry_test.go`, `doc.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`, `ambient_authority_test.go` | **NOT TOUCHED** | Scope-fence stays at 25 kinds; no new dependency |

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | Widening both filters for **existing** substrate files is a new kind of exception | High (certain) | Recorded above and in `apply-progress.md`; exact-filename suffixes only; `R-LSK-004`/`R-APP-012` deltas amend the owning specs so the rule is changed, not quietly broken |
| 2 | Invariant 4's joint-closure claim becomes unauditable | Med | `R-ATT-006` scenario lands in `invariant_pin_test.go`'s family, cross-referenced from the `agent-event-envelope` delta |
| 3 | `t.fatal` is not uniformly `*ai.Failure` — JSON/construction errors (`loop.go:454,467,477,494,507,524,573`) share the field and the branch | Med | `sdd-design` decides: type-assert to `*ai.Failure` for the typed path; the internal-error branch's treatment is an explicit design decision, not an accident |
| 4 | Silently "fixing" or entrenching the `loop.go:265` result-discard while editing next to it | Med | Named out-of-scope above; `sdd-verify` diffs `loop.go:257-266` against base |
| 5 | `finalize()` runs before `finish` is normalized (`loop.go:286-290`), so `outcomeForFinish` can see a zero `ai.FinishReason` on the no-Completion path | Med | `sdd-design` must normalize to `FinishReasonStop` **before** `finalize()`, or give `outcomeForFinish` an explicit zero case; either way it is a decided rule, not a default |
| 6 | `S-LSK-001` pins `turn_end (TurnOutcomeFinished)` on the walking-skeleton turn | Low | That turn ends in `FinishReasonStop`, which maps to `TurnOutcomeFinished` — pin holds unchanged; verified at spec `agent-loop-skeleton/spec.md:31` |
| 7 | Review budget: forecast 900–1500 lines against 1000 | High | `size:exception` pre-authorized; the budget is extendable because this PR carries only AG-11 |
| 8 | `doc_contract_guard_test.go` if `Turn`'s documented return shape changes | Low | `Turn`'s **signature** is unchanged; only the fatal path's returned `msg` value changes. Flagged for `sdd-design` to confirm no `doc.go` row moves |

## Rollback Plan

Single revert of the AG-11 merge commit. `TurnOutcome` returns to its two members and `finalize()` to the hardcoded `TurnOutcomeFinished`/`RunOutcomeCompleted` pair; `agent.Failure` loses `PartialOutput()`; `loop.go`'s fatal branch returns to `return ai.Message{}, 0, turn.fatal`; both substrate filters return to their 13-filename lists; the new test files are deleted. `NewTurnEnd`'s validator, the 25-kind scope-fence, `go.mod`/`go.sum` and every AG-04..AG-10 invariant are untouched by AG-11 and therefore untouched by its revert. No migrations, no data, no consumer outside `backend/agent`. Re-running `cd backend/agent && make test` at the parent commit confirms zero regression.

## Dependencies

- **AG-07** (archived) — `Turn`, `turnAccumulator`, `finalize()`, the substrate guard
- **AG-09** (archived) — the scheduler wire-up at `loop.go:257-266` this change must not disturb
- **AG-04.2/AG-04.3** (archived) — `TurnOutcome`, `TurnEnd`, `RunEnd`, `agent.Failure`, `invariant_pin_test.go`
- **Layer 1 AI-13 / AI-19** — `ai.FinishReason`'s seven members (`finish_reason.go:41-114`) and the failure taxonomy (`provider_failure.go`); consumed, never edited
- **doc 0003:1113-1176** — the AG-11 charter and its two Gherkin leaves

## Success Criteria

- [ ] `cd backend/agent && make test` green with `-race`; both AG-11.1 scenarios and both AG-11.2 scenarios closed
- [ ] Every `ai.FinishReason` member maps to a distinct `TurnOutcome`; the agent-level exhaustiveness pin RED-records before it is GREEN
- [ ] Refusal and pause are distinguishable from each other and from `Finished` by outcome value alone, with no `ai.FinishReason` inspection required by the consumer
- [ ] A mid-stream terminal error after partial content produces `turn_end(aborted)` carrying a `*agent.Failure` whose `Category()`, `Retryable()` and `PartialOutput()` are all inspectable, plus `run_end(failed)`, plus the partial assistant content on the returned `msg`
- [ ] `len(provider.Requests()) == 1` on every failing turn
- [ ] Both substrate filters widened by exact filename only, byte-in-sync; the deviation recorded in `apply-progress.md`
- [ ] `R-LSK-004`, `R-AEV-008` and `R-APP-012` deltas written — the substrate rule is amended in its owning specs, not silently broken
- [ ] `make lint` clean (after `golangci-lint cache clean`), `make build` clean, `make vuln-check` clean
- [ ] Zero edits to `event_descriptor.go`, `event.go`, `stream_check.go`, `sequence.go`, `run_events.go`, `tool_event.go`, `event_registry_test.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`; scope-fence still 25 kinds
- [ ] `loop.go:265`'s `_ = sched.Schedule(...)` byte-unchanged
