# Exploration — `cachicamas-agent-loop-skeleton` (AG-07)

> Milestone AG-07 (Layer 2 Wave 2, the opening milestone) of doc 0003, lines 771-832. SDD change slug: `cachicamas-agent-loop-skeleton`. Artifact store: HYBRID (Engram + OpenSpec). Engram topic key: `sdd/cachicamas-agent-loop-skeleton/explore`. Branch `feat/agent-layer2-wave2-ag07` based at `8420b2c4` (post-AG-06 merge).

## Identity

- **Slug**: `cachicamas-agent-loop-skeleton`
- **Milestone**: AG-07 (Layer 2 Wave 2, milestone 7 of 24; doc 0003 § AG-07, lines 771-832)
- **Branch**: `feat/agent-layer2-wave2-ag07`
- **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag07`
- **Store**: HYBRID (Engram topic key `sdd/cachicamas-agent-loop-skeleton/explore` + filesystem `openspec/changes/cachicamas-agent-loop-skeleton/exploration.md`)
- **Mode**: automatic (gatekeeper between phases; no user interruption)
- **Strict TDD**: enabled
- **Review budget**: 1000 lines, `size:exception` pre-authorized for AG-07's walking-skeleton scope (braejan's standing rule from AG-04 / AG-05)
- **Closes**: R-06's stateless core (the loop's six must-nevers — statelessness is the load-bearing part); R-20 (substrate-bound tests)
- **Depends on**: AI-40 (frozen Layer 1 surface), AI-21 (fake provider), AI-22 (stream kit), AG-04 (envelope + ordering), AG-05 (message + tool families)
- **Blocks**: AG-08 (pre-request hook seam), AG-09 (tool execution contract), AG-11 (turn termination + typed failure)
- **Out of scope**: tools (AG-09), hooks (AG-08), errors beyond typed pass-through (AG-11), reasoning display policy (frontend concern)

## Context — the AG-04/AG-05/AG-06 substrate AG-07 consumes

AG-07 is the FIRST milestone where Layer 2 produces events from a live loop. Everything AG-04/05/06 built is consumed unchanged.

| Substrate piece | Lives at | AG-07 use |
|---|---|---|
| `Event` envelope (Payload, seq, run, turn, hasTurn, parent, hasParent) | `backend/agent/src/agent/event.go:454-462` | Produced by the loop; each event stamped by a per-lane `LaneStamper` |
| Derived kind (no `Kind` field stored) | `event.go:469-474` | Each emitted event's kind comes from its payload constructor's `kind()` method |
| `CheckEmit` (R-AEV-001, sequence non-zero, payload validate) | `event.go:526-543` | The loop's emission boundary; every event the loop produces must pass |
| `eventRegistry` of 25 kinds (post-AG-06 merge) | `event.go:242-366` | Loop emits `run_start`, `run_end`, `turn_start`, `turn_end`, `message_start_text`, `message_delta_text`, `message_end_text`, and (AG-07.2 #2) the reasoning set; **no other kinds in walking skeleton** |
| `LaneStamper` per-canvas (one goroutine, no mutex) | `sequence.go:48-58` | The loop owns one stamper, stamps each event after construction |
| `EventDescriptor` (Bracket, Placement, Cardinality, Terminal) | `event_descriptor.go:136-153` | All 25 rows already register the descriptors; AG-07 is pure consumer |
| `CheckStream` (the validator) | `stream_check.go:92-185` | AG-07's test must verify the loop's emitted events pass `CheckStream` (the AG-04.4's S-AEV-092 extensibility proof) |
| `RunStart` / `RunEnd` ctors | `run_events.go:53-182` | Loop emits `run_start` at the start and `run_end` at the end (or none — see U3) |
| `TurnStart` / `TurnEnd` ctors | `turn_events.go:42-147` | Loop emits `turn_start` at the start of one turn and `turn_end` at termination; `TurnOutcome` is `TurnOutcomeFinished` for AG-07's walking skeleton |
| `MessageStart{Text,Reasoning}` / `MessageDelta{Text,Reasoning}` / `MessageEnd{Text,Reasoning}` ctors | `message_text.go:72-235`, `message_reasoning.go:58-205` | Loop translates each provider stream entry into a bracket on the envelope |
| `Failure` wrap (Layer 2 typed-failure) | `failure.go:28-79` | Walking skeleton carries no typed failure (out of scope: AG-11) |
| `Completion` event (Layer 1; `reason FinishReason`, `usage Usage`) | `backend/agent/src/ai/completion.go:34-93` | Finish reason is **read from the provider's `Completion` event** in the script; AG-07 surfaces it via the loop's return value (see D3) |
| `AgentProvider` interface `Stream(ctx, req) (<-chan ai.Event, error)` | `backend/agent/src/ai/provider.go:96-100` | The loop's one call: the provider returns a receive-only channel of normalized `ai.Event` |
| `agenttest.Provider` (scripted) + `agenttest.Script` + `agenttest.Emit` + `agenttest.Hold` + `agenttest.NewGate` | `backend/agent/src/agenttest/fake_provider.go:55-211`, `fake_script.go:15-52`, `fake_gate.go:20-50` | All walking-skeleton tests script on this fake, no network |
| `agenttest.NewIter(ch)` (carrier view) | `backend/agent/src/agenttest/stream_kit_iter.go:21-80` | Reuse in tests for iterative drain of the provider stream |
| `agenttest.RequireSameEvents(tb, got, want)` | `backend/agent/src/agenttest/stream_kit_diff.go:27-52` | Reuse in tests for byte-equal diffs of the *emitted* agent stream (`got`) vs a hand-built expected (`want`) |
| `reconstructMessage` helper (AG-05.3) | `backend/agent/src/agent/reconstruction_test.go:54-114` | Reuse or write a sibling for the loop's emitted events; AG-07.1 #3's "one source of truth" test defends against a vacuous reconstruction (W1 lesson) |
| Every-kind-constructible guard (25 kinds, post-AG-06) | `event_registry_test.go:54-251` | Untouched by AG-07 (no new kinds) |
| AG-01 carrier decision (`receive-only channel of agent events`) | `openspec/changes/archive/2026-08-11-cachicamas-agent-event-delivery/decision.md:53-94` | AG-07 MUST emit on a channel — the loop's `Turn()` returns a channel of `agent.Event` (or a wrapper), not a callback iterator |
| Import boundary / ambient authority guards | `backend/agent/src/agent/import_boundary_test.go`, `ambient_authority_test.go` | Untouched; AG-07's loop code must stay within the allowed imports (stdlib + `src/ai` + `src/agenttest` for tests) |

## Charter (AG-07, doc 0003:771-832)

| Node | Type | Charter scenarios | Closes |
|---|---|---|---|
| AG-07.1 — One text turn `[leaf]` | walking-skeleton end-to-end; consumer drains event sequence; context respected; one source of truth | 3 (0003:795-808) | R-06's "no persistence", R-20's substrate-bound tests |
| AG-07.2 — Statelessness and reasoning pass-through `[leaf]` | two sequential turns share nothing; reasoning and text distinguished by kind; reasoning round-trip token byte-exact | 2 (0003:819-828) | R-06's statelessness; the segmented-by-kind delivery from AG-05.1 R-AMT-002 |

**Total**: 5 charter Gherkin scenarios in 2 leaves. Forecast ~8-15 spec scenarios after per-scenario expansion + bites (AG-05 precedent: 7 charter → 15 spec).

**The single most important node in doc 0003** (per the AG-07 charter header, `0003:773`): "the first time Layer 1 and Layer 2 meet."

## Key decisions surfaced — D1-D6

| # | Decision | Cited evidence | Recommendation (surface, not pre-pick) |
|---|---|---|---|
| **D1** | Loop's public one-turn surface: `func Turn(ctx, provider, system, transcript, opts, sink) (msg, finish, err)` vs `Turner` value with `Turn(...)` method vs builder-style struct | AG-05 precedent uses builder-style helpers (e.g. `RunEnd` factory-style; `MessageStartText` direct ctors); AG-13's harness will need a value-style `Harness`; walking-skeleton precedent favors simple `func(...)` for now | **Surface D1a** — `func Turn(ctx, provider, system, transcript, opts, sink) (msg ai.Message, finish ai.FinishReason, err error)` with `opts Options` and `sink chan<- agent.Event` — the simplest shape that lets AG-07.2's "two sequential turns on one loop value" scenario be asserted directly (since the loop value is the function, not a value). AG-13 introduces the `Harness` value that wraps `Turn` in a stateful shell. |
| **D2** | How the loop emits events to the consumer: `<-chan agent.Event` (channel) vs `iter.Seq[agent.Event]` (iterator view) vs `agent.EventSink` interface | AG-01 decision § 3 chose "**receive-only carrier of agent events**" at Layer 2's package boundary — iterator ergonomics deferred to AG-23.2. AI-22.5 (`stream_kit_iter.go`) demonstrates the iterator view is **for Layer 1 streams in tests**, not for Layer 2's package boundary | **Surface D2a** — the loop returns a `<-chan agent.Event` (the harness-facing channel). The tests use `agenttest.NewIter` for ergonomic drains. No `EventSink` interface yet — it would be premature. |
| **D3** | How finish reason propagates: on `TurnEnd`'s typed outcome (the AG-04 substrate has `TurnOutcome` but NOT `FinishReason`) vs on the return value of `Turn()` vs both | `TurnEnd` payload is `{outcome: TurnOutcome, failure: *Failure}` (`turn_events.go:109-112`) — it does NOT carry `FinishReason`. Adding a `finish Reason` field to `TurnEnd` would be an envelope change (R-AEV-004 expansion), explicitly out of scope per the AG-07 charter. The charter's "the consumer drains... turn-end carrying the model's finish reason" reads as "the consumer drains the events AND sees the finish reason observed", not "the finish reason is on the turn-end event itself" | **Surface D3a** — finish reason propagates on the return value of `Turn()`. The walking skeleton returns `(ai.Message, ai.FinishReason, error)`. Later milestones (AG-11 typed-failure path) can wrap this in a typed `Result` value; AG-07 keeps the bare destructure. |
| **D4** | How reasoning interleaving works in the scripted fake: builder-style `ScriptedReasoningTextResponse` vs extending `agenttest.Script` to take a mix of text and reasoning events | `agenttest.Provider` consumes `agenttest.Script` (a sequence of `Step`s — `Emit` or `Hold`). For AG-07.2 #2, the script needs to emit `ai.NewReasoningBlockStart(1)` + `ai.NewReasoningDelta(...)` + `ai.NewReasoningBlockEnd(..., token)` interleaved with text events. The fake already supports this — `fake_reasoning_test.go` proves it (`TestProvider_MixedReasoningAndText_NeverCrossesIntoTheOtherEventKind`) | **Surface D4a** — script the provider directly with `agenttest.Emit(...)` calls in interleaved order. No new scripted-response helper needed; the substrate's `agenttest.Script` is already the door. |
| **D5** | "The caller's context is respected" — pass `ctx` straight through vs derive (`context.WithoutCancel`) vs impose a timeout | Walking-skeleton preference: pass `ctx` straight through with one derived value — `ctx` for the provider's `Stream(ctx, req)` call. `go.mod` is Go 1.26.5 per `wc -l` probe; `context.WithoutCancel` is in stdlib since 1.21. The provider's pre-stream contract (`provider.go:74-103`) already handles nil-ctx and already-cancelled-ctx. The only derivation needed is for the loop's own bookkeeping (none in walking skeleton) | **Surface D5a** — pass `ctx` through to `provider.Stream(ctx, req)` unchanged. No derived context. The walking skeleton test asserts (a) a passed-through non-cancelled `ctx` lets the stream run to completion, and (b) a cancelled `ctx` is honored at the producer boundary (this is the AI-20 mid-stream physics the fake already proves — AG-07 inherits the proof). |
| **D6** | Spec prefix: `R-LSK-` (loop skeleton) vs `R-LP-` (loop) vs `R-AG07-` | AG-05 chose `R-AMT-` (message-tool); AG-06 chose `R-APE-` (protocol-events). Two-letter match to slug | **Surface D6a** — `R-LSK-` (loop-skeleton). Matches AG-04/AG-05/AG-06 two-letter convention. Scenarios `S-LSK-NNN`. |

**Note for orchestrator**: D1, D3, D6 are the load-bearing decisions. Surface them to braejan before proposing. The other three (D2, D4, D5) inherit directly from AG-01 / AI-22 / AI-20 and need no confirmation.

## Substrate inventory (line counts)

Total `backend/agent/src/agent/`: **9,190 lines** (post-AG-06 merge at `8420b2c4`).

- AG-07 will MODIFY: nothing. AG-07 is the first milestone to exercise the substrate without amending it.
- AG-07 will NOT touch: `event.go`, `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `event_registry_test.go`, `doc.go`, `doc_contract_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`. **NO** modifications to envelope/descriptor/validator. **NO** register row additions. **NO** envelope variant extension.
- AG-07 will likely CREATE: `loop.go` (the turn runner, ~80-150 lines), `loop_test.go` (the AG-07.1 scenarios + AG-07.2 scenarios, ~250-400 lines). Possibly a small helper in `agenttest/` if the stub text/response builder proves its own shape (forecast: 0-50 lines if needed, 0 if not).
- AG-07 will NOT change: `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, AG-03 boundary guards.

**Forecast**: 400-700 lines added (smaller than AG-05's 2,479 or AG-06's 1,500-2,400). Walking-skeleton scope is small. The 1000-line budget with `size:exception` pre-authorized is sufficient with margin.

## Carry-forward list (AG-04/AG-05/AG-06 warnings applied to AG-07)

| Source | Finding | AG-07 mitigation |
|---|---|---|
| AG-04 W1 (post-verify fix c203f25c) | Position naming: `CheckStream` was emitting `sequence value` instead of `slice index`; post-verify fix landed | AG-07 tests that use `CheckStream` rely on the fix; the validator's emitted position is the slice index. AG-07's tests assert the slice index via `agenttest.RequireSameEvents` and via direct `CheckStream` invocations. |
| AG-04 W2 (post-verify fix c203f25c) | `ai.AtIndex("event", i)` in violation sites | Same as W1 — AG-07 benefits from the fix. The "walking skeleton" test (AG-07.1 #1) consumes the validator's report at the position the loop emitted to. |
| AG-04 W3 (post-verify fix; engine reads `d.Terminal`) | `Terminal` field was inert; the engine now reads it at `stream_check.go:173-175` | AG-07 emits events through the substrate's constructors, which already pass `CheckStream`. AG-07's most-stream test asserts the validator accepts the produced stream. |
| AG-04 W4 (carried) | `Event.Turn()` hardcoded survived | AG-07.2 #2 reasoning interleaving test sets `Event.Turn()` evidence on every event (the test reads `t, ok := e.Turn()` and checks identity). If a latent bug surfaces here, it's an AG-04 carry-forward bug AG-07 surfaces (not creates). |
| AG-04 W5 | S-AEV-050 doesn't satisfy its `Given` clause | Pre-existing; not AG-07's scope. |
| AG-04 W6 | S-AEV-003 rejection branch unreachable by design | Not AG-07's scope. |
| AG-04 W7 (carried) | S-AEV-054 is a doc-phrase check | AG-07's tests don't rely on ambiguous doc phrasing. |
| AG-04 W8 (carried) | Coverage 69.7% in AG-04 | AG-07's walking skeleton MUST hit >= 80% in `loop.go` (the only new file). |
| AG-04 W9 (carried) | Scenario count drift | State identically in proposal, tasks, apply-progress: **5 charter → ~8-15 spec**. |
| AG-05 W1 (carried) | Vacuous reconstruction helper (`message_text_test.go:309-339`) | **AG-07.1 #3's "one source of truth" test MUST bite before it satisfies.** Write two failing bites (drop a delta, double a delta) BEFORE the GREEN — same pattern as S-AMT-071 / S-AMT-072 (`reconstruction_test.go:180-277`). |
| AG-05 W2 (carried) | `TestEventKinds_AG05AllRegisterPlacementTurn` is a name-prefix test, not a structural pin | If AG-07 touches any kind's name or placement, split into name-check + placement-check. |
| AG-05 S1 (carried) | `Terminal: false` zero-value vs explicit | AG-07 doesn't add kinds, so this carries forward unchanged. |
| AG-05 S2 (carried) | Reflection pin name-only | If AG-07 adds any reflection pin (unlikely — see D1), check name AND type. |
| AG-06 W7 (carried) | CardinalityAtMostOne seam reserved | AG-07 doesn't add kinds; this is inherited. |
| AG-06 (recent) | Spec prefix `R-APE-` | AG-07's `R-LSK-` continues the convention. |
| AG-06 (recent) | Scope-fence `S-AEV-090` retightens from 25 to N | AG-07 does NOT retighten (no new kinds). |
| AG-06 (recent) | Per-family file split | AG-07's `loop.go` is a single-file node (no family split — one turn runner). |
| Project-wide (out of date) | `test-driven-development` skill does not exist at `~/.claude/skills/` | RED-GREEN-REFACTOR discipline is forwarded inline from `openspec/AGENTS.md` (which has the project's authoritative version). The `openspec/AGENTS.md` rule says "test-driven-development" is the path, but the file is missing — AG-07 must cite the AGENTS.md `## Strict TDD is on` block directly. |

## Risks & unknowns

### Quantified risks

1. **R1 — Loop surface conflict (medium)**: AG-13's harness will need a value-form surface. AG-07's `func Turn(...)` choice has to compose with AG-13's later shape without a redesign. **Mitigation**: surface D1 in the proposal with the AG-13 prediction; commit to "the function-form composes via a `Harness` struct that wraps `Turn` (no signature change)". AG-13's own distance is the precedent.
2. **R2 — `TurnEnd` does not carry `FinishReason` (low→medium)**: Charter says "turn-end carrying the model's finish reason", but the substrate's `TurnEnd` payload is `{outcome, failure}` only. **Mitigation**: D3 surfaces this with the cited evidence; the proposal commits to the return-value path. The walk-skeleton test asserts the consumer observes the finish reason via the return value, not on the turn-end event itself.
3. **R3 — Review-budget 1000 lines: low at 400-700 forecast, but 1.5-2× possible (low)**: AG-07's walking skeleton is small, but extensive Strict TDD bites + per-scenario expansion could push toward 1000. **Mitigation**: `size:exception` pre-authorized. Chain PRs are not needed (AG-05 precedent: 2,479 lines => single PR, exception accepted).
4. **R4 — `engram/protocol.md` vs `openspec/AGENTS.md` divergence on TDD skill path (low)**: AGENTS.md lists `test-driven-development` at `~/.claude/skills/test-driven-development/SKILL.md`, but the file does not exist there. **Mitigation**: read AGENTS.md `## Strict TDD is on` block directly; cite `## Hard rules` rather than the missing skill.
5. **R5 — Substrate has no `Turn()` for the loop value to be a value (low)**: AG-07.2 #2 wants "two sequential turns on one loop value are independent". If the loop is a function, the "loop value" is the function (and two calls share nothing by construction). If the loop is a `*Turner` value, the test mutates one field. **Mitigation**: D1a (function form) is the simpler path; the test asserts two `Turn(...)` calls share no state by constructing fresh slices / no closure captures.
6. **R6 — Reasoning round-trip token byte-exact (AG-07.2 #2) requires Layer 1 round-trip preservation (low)**: AG-05.3's reconstruction property (R-AMT-008) covers text fragments. The reasoning round-trip token is Layer 1's `ai.NewReasoningBlockEnd(..., []byte t)` payload — fake_reasoning_test.go:42-94 proves byte-exactness. The loop's job is to emit the `message_end_reasoning` with the token unchanged. **Mitigation**: write the test as a bytes.Equal of the reasoning-end event's payload's token vs the script's token.
7. **R7 — `agenttest.Script.Buffer` (channel capacity) selection (low)**: The fake's `Buffer` field sizes the channel capacity. AG-07's test wants to drain events deterministically; unbuffered (Buffer=0) is the safest choice. **Mitigation**: chosen in the proposal via the AI-21.1 precedent (`fake_text_test.go:131-136`).
8. **R8 — `MessageID` minting across turns (low)**: Two turns may mint matching `MessageID`s if not handled. The agent's `MessageStartText` requires a non-zero `msgID` (per `message_text.go:62-66`). The walking skeleton is text-only (AG-07.1) and reasoning+text (AG-07.2). **Mitigation**: the loop mints per turn via `ai.NewMessage(ai.RoleAssistant, ai.NewText(fragments...))` after the deltas accumulate, or the loop carries the `msgID` through from the running accumulation. Detail in the proposal.
9. **R9 — Pass-through of `ctx` cancellation (low)**: AI-20 mid-stream physics proven by the fake (`fake_provider.go:192-211`); AG-07's test asserts the consumer's `ctx.Done()` honors the loop's downstream drain. **Mitigation**: AG-07.1 #2's "context respected" test asserts (a) a non-cancelled `ctx` lets the stream run, then (b) a cancelled mid-stream `ctx` is honored at the producer's send-select.
10. **R10 — No `agenttest` scripted text-response helper (low)**: AG-07.1 #1 needs a one-line scripted response. AG-21.1's `mustTextDeltaScript` (`fake_text_test.go:112-137`) is the precedent inside `agenttest_test`. **Mitigation**: AG-07's test uses the same script-building pattern, adapted to its package. No new `agenttest` helper needed.

### Unknowns to verify before design

- **U1**: Whether the `Loop` surface (D1) takes `provider ai.ModelProvider` as an interface or as a concrete `*agenttest.Provider` for the test. Likely interface (the only realistic answer, otherwise the loop is untestable with a real provider).
- **U2**: Whether the loop's `Turn()` returns the assistant `ai.Message` reconstructed from the deltas, or a `[]string` fragments slice, or the per-lane `[]agent.Event` it emitted. D3 says "the assistant message" — that's an `ai.Message`, consistent with R-AMT-004 and reconstruction_test.go.
- **U3**: Whether the loop emits `run_start` / `run_end` per turn, or whether the walking skeleton is turn-only (no run bracket). The validators require a complete run bracket (`stream_check.go:178-183`). **Two paths**: (a) emit `run_start` → `turn_start` → ... → `turn_end` → `run_end` per turn (one run per turn); (b) the loop emits only `turn_start` → ... → `turn_end` and the harness (AG-13) wraps it in a run bracket. Path (a) is simpler for the walking skeleton; path (b) makes the loop's "statelessness" assertion direct. Confirm in the proposal.
- **U4**: Whether the assistant `ai.Message` carries the `RoleAssistant` and the accumulated text fragments, or whether the loop emits the message-end as a typed event only. AG-05.1 substrate: `MessageStartText` / `MessageDeltaText` / `MessageEndText` all carry the `msgID`. The `Message` itself is constructed on the loop side from the fragments.
- **U5**: Whether the consumer's channel close is the loop's responsibility (loop closes on turn-end) or the consumer's responsibility (loop only closes the channel when the turn itself ends). R-AFP-007's one closing site belongs to the producer. The loop IS the producer. **Mitigation**: loop closes its channel after `turn_end` + a `run_end` (if U3 = path a).
- **U6**: Whether the `Context` derived value (D5) needs `context.WithoutCancel` for any reason. **Mitigation**: pass-through is the simplest, propose it.

## Recommendation for next phase

**Ready for Proposal: YES.** The substrate is well-understood; the AG-04/AG-05/AG-06 lessons are explicit carry-forwards; the open decisions (D1-D6) are listed with cited evidence; the risks are quantified; the unknowns are named with verifications.

The orchestrator should launch `sdd-propose` next, with the following pre-commit confirmations it should surface to braejan before proposing:

1. **D1 (loop surface)** — `func Turn(ctx, provider, system, transcript, opts, sink) (msg ai.Message, finish ai.FinishReason, err error)` — confirm function form over `Turner` value, since AG-13 will introduce the value-form `Harness` later.
2. **D3 (finish reason propagation)** — return value, NOT on the `TurnEnd` payload. AG-07's charter reads "the consumer drains... turn-end carrying the model's finish reason" as the consumer observing the finish reason (via the loop's return), not the reason being on the event itself.
3. **D6 (spec prefix)** — `R-LSK-` / `S-LSK-`.
4. **U3 (run bracket per turn)** — confirm path (a) `run_start` ... `run_end` per turn, OR path (b) turn-only emission with the harness wrapping in a run bracket. Path (a) is the walking-skeleton preference; the proposal should commit to it unless braejan prefers path (b).

## Key Learnings

1. AG-07 is the FIRST milestone where Layer 2 emits events from a live loop (AG-04/05/06's stream-validator tests fed the validator HAND-BUILT events). AG-07 owns the producer side of AG-01's decision.
2. The AG-04/05/06 substrate is COMPLETE for AG-07's walking skeleton: constructors, descriptor, validator, LaneStamper, every-kind-constructible guard, typed-failure wrap, per-family file split. AG-07 is a pure consumer — no register row addition, no envelope variant extension (`event.go:454-462` carries the envelope data; `turn_events.go:109-112` carries `TurnOutcome` only).
3. The charter's "turn-end carrying the model's finish reason" is interpretable as the consumer observing the finish reason via the loop's return value, not as the finish reason being on the `TurnEnd` payload. The substrate's `TurnEnd` has no `FinishReason` field (`turn_events.go:109-112`); adding one would be an envelope change (R-AEV-004 expansion), explicitly out of scope.
4. AG-01's carrier decision (`receive-only channel of agent events`) lands at the layer-2 package boundary, NOT at the loop's internal emission. AG-07 commits to D2a: the loop returns a `<-chan agent.Event` to the consumer. Iterator ergonomics live in `agenttest.stream_kit_iter` for the test side.
5. AG-07's test substrate (the `agenttest` package) already supports both walking-skeleton needs: scripted text (`fake_text_test.go`) and scripted reasoning with byte-exact round-trip tokens (`fake_reasoning_test.go:42-94`). No new `agenttest` helper is needed.
6. The ratchet rule for AG-07's strict TDD: write `reconstructMessage`-style bites (drop a delta, double a delta) BEFORE the AG-07.1 #3 property test goes green. The W1 vacuous-helper trap from AG-05 is the live precedent (`reconstruction_test.go:180-277`).
7. The `Loop` surface is a function, not a value, in the walking skeleton. AG-07.2 #2's "two sequential turns on one loop value are independent" is asserted by calling `Turn(...)` twice and observing no shared state (no closure captures over the first call's results).
8. Scope-fence `S-AEV-090` retightens from "exactly 4" to "exactly 25" with AG-06; AG-07 does NOT retighten (no new kinds). The 25-kinds count holds at merge time.
9. The substrate's `ai.Stamper` (`backend/agent/src/ai`) and `agent.LaneStamper` (`backend/agent/src/agent`) are independent per-stream counters; AG-07's loop owns one `LaneStamper` per turn (per `sequence.go:48-58`'s "one goroutine per lane" rule).
10. `go.mod` is `go 1.26.5` (post-AG-06). `context.WithoutCancel` is available since Go 1.21 — AG-07's pass-through-`ctx` (D5a) does not need any derived context for the walking skeleton.
11. The 1000-line review budget with `size:exception` pre-authorized carries forward from AG-04 / AG-05 standing rule. AG-07's 400-700 forecast lands well under the limit even with the strict TDD bites.
12. The `test-driven-development` skill path listed in `openspec/AGENTS.md` does not exist on disk. AG-07 (and AG-08 onward) must read the `## Strict TDD is on` block in `openspec/AGENTS.md` directly and cite by section, not by skill path.

## Evidence

Every claim cites file:line or Engram observation:

- **Charter**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:771-832` (AG-07, lines 771-832; 5 charter Gherkin scenarios in 2 leaves, lines 794-829)
- **v2 architecture**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:450-551` (Layer 2 § 4.1 "the loop's six must-nevers", § 4.2 "the harness's one must-never", § 4.3 envelope invariants)
- **AG-01 carrier decision**: `openspec/changes/archive/2026-08-11-cachicamas-agent-event-delivery/decision.md:53-94` (§ 3 decision 1 — receive-only carrier)
- **Substrate, Layer 2 envelope**: `backend/agent/src/agent/event.go:454-462` (Event struct), `:469-474` (derived Kind), `:526-543` (CheckEmit), `:242-366` (eventRegistry 25 kinds)
- **Substrate, ordering**: `backend/agent/src/agent/event_descriptor.go:136-153` (EventDescriptor), `stream_check.go:92-185` (CheckStream), `sequence.go:30-58` (LaneStamper, design AD-5)
- **Substrate, constructors**: `backend/agent/src/agent/run_events.go:53-182` (RunStart/End), `turn_events.go:42-147` (TurnStart/End, `TurnOutcome` finished/aborted), `message_text.go:72-243` (MessageStartText/DeltaText/EndText), `message_reasoning.go:58-223` (MessageStartReasoning/DeltaReasoning/EndReasoning), `failure.go:28-79` (Failure wrap)
- **Substrate, Layer 1**: `backend/agent/src/ai/provider.go:96-100` (ModelProvider interface), `backend/agent/src/ai/completion.go:34-93` (Completion carries FinishReason), `backend/agent/src/ai/finish_reason.go:38-114` (FinishReason closed vocabulary)
- **Substrate, agenttest**: `backend/agent/src/agenttest/fake_provider.go:55-211` (Provider; Stream returns `<-chan ai.Event`), `fake_script.go:15-52` (Emit / Hold / Script), `fake_gate.go:20-50` (Gate), `fake_text_test.go:112-137` (mustTextDeltaScript), `fake_reasoning_test.go:42-94` (round-trip token byte-exact), `stream_kit_iter.go:21-80` (carrier view), `stream_kit_diff.go:27-52` (RequireSameEvents)
- **Substrate, AG-05 reconstruction pattern**: `backend/agent/src/agent/reconstruction_test.go:54-114` (helper), `:180-277` (S-AMT-071 drop-a-delta + S-AMT-072 double-a-delta bites)
- **Substrate, AG-06 registry**: `backend/agent/src/agent/event_registry_test.go:54-522` (25-kinds witness table; scope-fence S-AEV-090 retightened to "exactly 25")
- **Substrate, signature guard**: `backend/agent/src/agenttest/provider_signature_guard_test.go:55-73` (ModelProvider exactly one method, Stream)
- **AG-05 carry-forwards**: Engram `#2960` session summary, AG-05's 3 WARNING + 2 SUGGESTION (W1: vacuous reconstruction helper; W2: name-prefix placement test; S1: Terminal:false explicit; S2: reflection pin name+type)
- **AG-04 verify findings (post-verify fixes applied)**: Engram `#2935` verify-report, W1-W9 (W1/W2 position naming fix c203f25c; W3 Terminal engine-read fix; W4 Event.Turn() hardcoded carried; coverage 69.7% carried)
- **AG-06 explore template**: Engram `#2966` (structural model for this artifact)
- **Doc 0003 authorship**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:222-242` (AG-01 carrier decision note "keep channels"), `:518-600` (AG-05 charter), `:602-712` (AG-06 charter), `:2204` (R-06 closes mapping), `:2250` (AG-07 ↔ R-06, R-20)
- **Project AGENTS.md**: `openspec/AGENTS.md:1-100` (stack, strict TDD, sub-agent launch contract, hard rules)
- **ADR 0005 dependency rule**: `docs/adr/0005-promote-agent-stack-to-own-module.md` (Layer 2's import allowlist)
