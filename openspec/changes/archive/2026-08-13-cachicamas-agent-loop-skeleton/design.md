# Design: AG-07 — Build the one-turn walking skeleton

## Technical Approach

The walking skeleton is one function `Turn(ctx, provider, system, transcript, opts, sink)` in `backend/agent/src/agent/loop.go` that produces a sequence of agent events from one assistant turn. The function drains the provider's `ai.Event` stream once, translates each event into a bracket on the Layer 2 envelope (`message_start`/`delta`/`end` for text and reasoning kinds), stamps each event with a per-turn `LaneStamper`, and emits to the consumer's `chan<- *Event`. The run bracket (`run_start` → `run_end`) wraps the entire turn per U3 path-a; the multi-turn run bracket arrives with AG-13's `Harness`. The finish reason propagates via the return value (D3a), not on the `TurnEnd` payload. The substrate is untouched — AG-07 changes are limited to `loop.go` + `loop_test.go`. This delivers R-LSK-001..005 / S-LSK-001..007 with R-LSK-004 (substrate untouched) and R-LSK-005 (≥80% coverage) as cross-cuts.

## Architecture Decisions

### Decision: D1a — Loop surface is function-form `Turn`

| | |
|---|---|
| **Choice** | `func Turn(ctx context.Context, provider ai.ModelProvider, system string, transcript []ai.Message, opts TurnOptions, sink chan<- *Event) (msg ai.Message, finish ai.FinishReason, err error)` |
| **Alternatives** | (1) value-form `Turner` struct with `Turn(...)` method; (2) builder-style `TurnBuilder` accumulating inputs |
| **Rationale** | Walking-skeleton scope is the thinnest end-to-end path; a function is the smallest surface and composes with AG-13's later `Harness` value (which wraps `Turn` without signature change). Function form makes `R-LSK-002` direct — two `Turn(...)` calls share nothing by construction (no closure captures, no shared `LaneStamper` — each turn mints a fresh one). |

### Decision: D2a — Channel carrier per AG-01

| | |
|---|---|
| **Choice** | `chan<- *Event` parameter; consumer drains `<-chan *Event`. The loop owns and closes the channel. |
| **Alternatives** | (1) `iter.Seq[*Event]` iterator view; (2) `EventSink` interface with callbacks; (3) synchronous return slice `[]*Event` |
| **Rationale** | AG-01's decision record chose "receive-only carrier of agent events" at the Layer 2 package boundary. Iterator ergonomics live in `agenttest.stream_kit_iter` for tests. An `EventSink` interface is premature — AG-07 emits to one consumer. A synchronous slice would lose the streaming semantics the substrate's `LaneStamper` and per-event ordering enable. |

### Decision: D3a — Finish reason on return value

| | |
|---|---|
| **Choice** | `(msg ai.Message, finish ai.FinishReason, err error)` — finish reason is a return value, not on `TurnEnd` |
| **Alternatives** | (1) Add `finish Reason` field to `TurnEnd` payload (envelope change); (2) emit typed `CompletionEvent` |
| **Rationale** | `TurnEnd` payload is `{outcome, failure}` only (`turn_events.go:109-112`); adding a `FinishReason` field would be an envelope change (R-AEV-004 expansion), out of scope per charter. The finish reason is read from the provider's `ai.Completion` (`completion.go:34-93`); surfacing via return is the minimal Layer 2 work. AG-11 wraps in `Result` later if needed. |

### Decision: D4a — Direct `agenttest.Script` for interleaved reasoning

| | |
|---|---|
| **Choice** | Test scripts the provider directly via `agenttest.Emit(...)` calls, interleaving `ai.NewMessageStart(Text/Reasoning)` + `ai.NewMessageDelta(Text/Reasoning)` + `ai.NewMessageEnd(Text/Reasoning)` + `ai.NewCompletion(...)`. |
| **Alternatives** | (1) new `ScriptedReasoningTextResponse` helper; (2) extend `agenttest.Script` with kind-mix builder |
| **Rationale** | Substrate already supports this — `fake_reasoning_test.go:42-94` proves byte-exact reasoning round-trip tokens through `agenttest.Script`. A new helper would over-engineer. Same pattern as `mustTextDeltaScript` in `fake_text_test.go:112-137`. |

### Decision: D5a — Pass-through `ctx`

| | |
|---|---|
| **Choice** | `ctx` flows unchanged to `provider.Stream(ctx, req)`. No derived context. |
| **Alternatives** | (1) `context.WithCancel`; (2) `context.WithTimeout`; (3) `context.WithoutCancel` |
| **Rationale** | Walking-skeleton scope is the thinnest path. The provider's pre-stream contract (`provider.go:74-103`) already handles nil-`ctx` and cancelled-`ctx`. If a later milestone (AG-08 hook seam, AG-09 tool scheduler) needs a derived context, it introduces one at that seam. |

### Decision: U3 path-a — One run per turn

| | |
|---|---|
| **Choice** | `Turn` emits `run_start` → `turn_start` → ... → `turn_end` → `run_end` per call |
| **Alternatives** | (1) Turn-only emission; harness wraps in run bracket |
| **Rationale** | The validator (`stream_check.go:92-185`) requires a complete run bracket (`stream_check.go:178-183`). Path (a) satisfies this trivially inside `Turn`. Path (b) would require the test to construct a run bracket by hand. AG-13's `Harness` introduces multi-turn where one run spans N turns. |

## Data Flow

```
Turn(ctx, provider, system, transcript, opts, sink)
  │
  ├─► mint runID, turnID, LaneStamper (fresh)
  ├─► emit run_start → sink
  ├─► emit turn_start → sink
  ├─► req = buildRequest(system, transcript, opts)
  ├─► pCh, pErr = provider.Stream(ctx, req)
  ├─► for ev in pCh:
  │     switch ev.Kind():
  │       MessageStart(Text)       → emit message_start_text       (stamped)
  │       MessageDelta(Text)       → emit message_delta_text       (stamped)
  │       MessageEnd(Text)         → emit message_end_text         (stamped)
  │       MessageStart(Reasoning)  → emit message_start_reasoning  (stamped)
  │       MessageDelta(Reasoning)  → emit message_delta_reasoning  (stamped)
  │       MessageEnd(Reasoning)    → emit message_end_reasoning    (stamped)
  │       Completion               → capture finish
  ├─► emit turn_end (TurnOutcomeFinished) → sink
  ├─► emit run_end   → sink
  ├─► close(sink)
  └─► return (reconstructedMsg, finish, pErr)
```

`msg` is reconstructed from accumulated deltas: per `R-LSK-001` scenario 3 (S-LSK-003 + bites), the loop holds a per-turn `[]ai.ContentPart` (text + reasoning parts interleaved by emission order) and constructs `ai.Message{Role: ai.RoleAssistant, Content: parts}` at turn end. The reasoning round-trip token is preserved byte-exact because each `ai.NewReasoningBlockEnd(..., token)` flows through unchanged to the assistant message's reasoning-content `Token` field.

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/loop.go` | NEW | `Turn` function + `TurnOptions` struct (~80-150 lines) |
| `backend/agent/src/agent/loop_test.go` | NEW | Strict TDD tests: S-LSK-001..007 (~250-400 lines) |
| `backend/agent/src/agent/{event,event_descriptor,stream_check,failure,sequence,run_events,turn_events,message_text,message_reasoning,permission_events,cost_events,delegation_events,compaction_events,tool_event,event_registry_test,doc,doc_contract_guard_test,ambient_authority_test,import_boundary_test,reconstruction_test}.go` | UNTOUCHED | substrate preserved (21 files, R-LSK-004) |
| `backend/agent/go.mod`, `go.sum` | UNTOUCHED | no new deps |
| `openspec/specs/agent-loop-skeleton/spec.md` | NEW | spec (Phase 3) |
| `openspec/changes/cachicamas-agent-loop-skeleton/{exploration,proposal,design}.md` | NEW | phase artifacts |

## Interfaces / Contracts

```go
package agent

// TurnOptions is the options struct for a single assistant turn.
// Walking-skeleton scope: trivial/zero fields; AG-08 introduces hooks,
// AG-09 introduces tool opts, AG-11 introduces retry opts.
type TurnOptions struct {
    // Model is the model identifier passed to the provider. Empty = provider default.
    Model string
    // MaxTokens is the optional max-tokens budget. Zero = provider default.
    MaxTokens int
}

// Turn runs one assistant turn end-to-end and emits the resulting events
// to sink. The function is stateless across calls — two Turn(...) invocations
// share nothing. The function closes sink before returning.
//
// R-LSK-001..005 / S-LSK-001..007 — see openspec/specs/agent-loop-skeleton/spec.md
func Turn(
    ctx context.Context,
    provider ai.ModelProvider,
    system string,
    transcript []ai.Message,
    opts TurnOptions,
    sink chan<- *Event,
) (msg ai.Message, finish ai.FinishReason, err error)
```

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | `Turn` emits exact event sequence per charter scenario | `agenttest.NewScriptedTextProvider(...)` driving `Turn(...)`; `agenttest.RequireSameEvents(t, got, want)` for byte-equal diffs; `agenttest.NewIter` for ergonomic drains |
| Bite | AG-07.1 #3 property non-vacuous (drop-a-delta, double-a-delta) | RED-first: write bites BEFORE property test GREEN, mirroring `S-AMT-071`/`S-AMT-072` pattern (`reconstruction_test.go:180-277`) |
| Integration | `Turn`'s emitted stream passes `CheckStream` | After `Turn` returns, run `agent.CheckStream(emitted)` — validator must accept |
| Coverage | `loop.go` ≥ 80% line coverage | `cd backend/agent && make test` with race detector |

Test file structure (single `loop_test.go`):
- `TestTurn_WalkingSkeleton_EmitsContractEventOrder` — S-LSK-001
- `TestTurn_ProviderStreamDrainedAndCtxRespected` — S-LSK-002
- `TestTurn_OneSourceOfTruthForAssistantMessage_BiteDropDelta` — S-LSK-003a (RED-first)
- `TestTurn_OneSourceOfTruthForAssistantMessage_BiteDoubleDelta` — S-LSK-003b (RED-first)
- `TestTurn_OneSourceOfTruthForAssistantMessage` — S-LSK-003
- `TestTurn_TwoSequentialTurnsShareNothing` — S-LSK-004
- `TestTurn_ReasoningPassThroughByteExact` — S-LSK-005
- `TestTurn_SubstrateUntouched` — S-LSK-006 (uses `git diff` against main `8420b2c4`)
- `TestTurn_CoverageGate` — S-LSK-007 (run with `-cover`)

## Threat Matrix

**N/A** — AG-07 introduces no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The loop is pure Go over the Layer 2 envelope and the Layer 1 provider contract. The substrate's `agenttest.Provider` is the only "external" boundary, operating over in-memory channels. No filesystem, network, environment, or process spawn. (`Event.Turn()` hardcoded per AG-04 W4 is an inherited AG-04 carry-forward; AG-07 does not introduce it.)

## Migration / Rollout

No migration required. AG-07 adds new files (`loop.go`, `loop_test.go`, spec). The substrate is byte-untouched. The PR is a single PR (`feat/agent-layer2-wave2-ag07`) with `size:exception` pre-authorized. Forecast 400–700 lines; well under the 1000-line review budget.

## Open Questions

None. D1a–D6a + U3 path-a are committed in the proposal and restated in this design. The 5 charter scenarios are fully covered by R-LSK-001..005 / S-LSK-001..007. The substrate is preserved (R-LSK-004). The walking-skeleton scope is the minimum end-to-end path.

## Cross-references

- Spec: `openspec/specs/agent-loop-skeleton/spec.md` (R-LSK-001..005, S-LSK-001..007)
- Proposal: `openspec/changes/cachicamas-agent-loop-skeleton/proposal.md` (D1–D6, U3 path-a)
- Exploration: `openspec/changes/cachicamas-agent-loop-skeleton/exploration.md` (D1–D6 surfaced, risks R1–R10)
- Charter: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:771-832` (AG-07)
- Substrate references (read for the loop implementation):
  - `backend/agent/src/agent/event.go:454-462` (Event struct), `:526-543` (CheckEmit), `:242-366` (eventRegistry 25 kinds)
  - `backend/agent/src/agent/event_descriptor.go:136-153` (EventDescriptor)
  - `backend/agent/src/agent/stream_check.go:92-185` (CheckStream), `:173-175` (Terminal engine read), `:178-183` (run-bracket rule)
  - `backend/agent/src/agent/sequence.go:30-58` (LaneStamper)
  - `backend/agent/src/agent/run_events.go:53-182` (RunStart/End), `turn_events.go:42-147` (TurnStart/End, TurnOutcome)
  - `backend/agent/src/agent/message_text.go:62-161` (MessageStart/Delta/End text ctors), `message_reasoning.go:58-157` (reasoning ctors + round-trip token)
  - `backend/agent/src/ai/provider.go:96-100` (ModelProvider interface)
  - `backend/agent/src/ai/completion.go:34-93` (Completion carries FinishReason)
  - `backend/agent/src/agenttest/fake_provider.go:55-211`, `fake_script.go:15-52`
  - `backend/agent/src/agenttest/stream_kit_iter.go:21-80`, `stream_kit_diff.go:27-52`
  - `backend/agent/src/agent/reconstruction_test.go:54-114` (AG-05.3 reconstruction pattern), `:180-277` (S-AMT-071/072 bites)
  - `backend/agent/src/agenttest/fake_reasoning_test.go:42-94` (byte-exact reasoning round-trip), `fake_text_test.go:112-137` (mustTextDeltaScript)