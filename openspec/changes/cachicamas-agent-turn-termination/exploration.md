# Exploration: `cachicamas-agent-turn-termination` (AG-11)

> **Change**: `cachicamas-agent-turn-termination` · **AG-11** (Layer 2 Wave 2, milestone 11 of 24; doc 0003 lines 1113-1176)
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag11`
> **Branch**: `feat/agent-layer2-wave2-ag11` based at `main` `b8eb7d75`
> **Closes**: R-08's typed mid-stream path; envelope invariant 4 (R-05, jointly with AG-04.3); doc 0003:2167-2168.
> **Format**: read-only investigation; no production code or test written.
> **Engram mirror**: `sdd/cachicamas-agent-turn-termination/explore` (observation #3059)

---

### Current State

**Termination today (`backend/agent/src/agent/loop.go`)**: `Turn()` (loop.go:179-292) has exactly three exit paths, and only one of them is "correct" by AG-11's charter:

1. **Completion event** (loop.go:246-269): `turn.translate(ev)` returns `true` on `ai.EventKindCompletion` (loop.go:530-534), capturing `t.finish = completion.FinishReason()` but emitting nothing onto `sink` at that point. The loop then calls `turn.finalize()` (loop.go:608-677). `finalize()` **unconditionally** emits `NewTurnEnd(runID, turnID, TurnOutcomeFinished, nil)` and `NewRunEnd(runID, RunOutcomeCompleted, nil)` (loop.go:613,617) — *regardless of what `t.finish` actually is*. A refusal, a content-filter stop, an unknown finish reason, or a paused turn are ALL reported as `TurnOutcomeFinished` today. There is no dispatch on `ai.FinishReason` anywhere in the `agent` package except the single equality check `turn.finish == ai.FinishReasonToolCalls` at loop.go:258 (which drives scheduling, not outcome typing).
2. **Mid-stream fatal error** (loop.go:270-276): `turn.fatal != nil` after `translate()` returns `false`. Set at loop.go:591-593 (`case ai.EventKindError: ... t.fatal = payload` where `payload` is the `*ai.Failure` returned by `ev.ErrorPayload()`) or at various JSON/construction-failure sites (loop.go:454,467,477,494,507,524,573 — plain Go errors from `NewMessageStartText` etc., NOT `*ai.Failure`). On this path the loop **drains the provider, closes `sink`, and returns `(ai.Message{}, 0, turn.fatal)` — with NO `turn_end`/`run_end` emitted at all.** A consumer watching `sink` sees the channel close abruptly with no typed event explaining why. Partial assistant content already accumulated in `turnAccumulator` (text/reasoning fragments) is discarded (`ai.Message{}`), not reconstructed. This is the concrete gap AG-11.2 must close.
3. **Provider closes with no Completion** (loop.go:279-291): treated as "ran to empty", `finalize()` is called (still emits `TurnOutcomeFinished`/`RunOutcomeCompleted`), and `finish` defaults to `ai.FinishReasonStop` if zero.

**The tool_calls result-loss gap (pre-existing, out of scope but worth flagging)**: at loop.go:257-266, `finalize()` runs and reconstructs `msg` BEFORE `Schedule` runs, and the scheduler's returned `[]Result` is discarded (`_ = sched.Schedule(...)`) — `turn.toolResults` is never populated even though `finalize()` reads it (loop.go:654). Not AG-11's charter, but adjacent to the "outcome per finish reason" work; the design phase should confirm this stays untouched.

### The finish-reason vocabulary (Layer 1, `backend/agent/src/ai/finish_reason.go`)

Seven closed values (finish_reason.go:41-114): `FinishReasonStop` (47), `FinishReasonLength` (53), `FinishReasonToolCalls` (58), `FinishReasonContentFilter` (69), `FinishReasonRefusal` (80), `FinishReasonPauseTurn` (91), `FinishReasonUnknown` (101), bounded by `finishReasonLimit` (113). `String()`/`Validate()`/`finishReasonNames` array follow the "array sized by the bound" idiom.

**The existing exhaustiveness pin** the charter says AG-11 must EXTEND is `TestFinishReason_AddingAValue_FailsWithoutATableAndAStringForm` (`backend/agent/src/ai/finish_reason_test.go:277-320`): it walks `FinishReason(0..255)`, asks `Validate()` which are members, and cross-checks against a hand-written `theVocabulary` list — a value that validates but isn't named fails three assertions. A second, sibling instance of the SAME idiom already lives one layer up in `backend/agent/src/agenttest/conformance_capabilities.go`: `finishReasonDriftGuardAgainst` (line 247) + `finishReasonExhaustivenessCase` (line 265) prove all seven reasons are *reachable on a stream* via the fake provider, with a drift guard against a shrunk/grown hand-list (`handListedFinishReasons`).

Neither of these two existing pins tests the **agent-loop's own dispatch**. AG-11.1 must add a THIRD instance of this idiom, this time inside `backend/agent/src/agent` (new test file), walking the same 0..255 space and asserting the loop's exhaustive switch has a named, distinct case for every value — so an eighth `ai.FinishReason` added upstream fails this new agent-level test too, not just the two Layer-1 ones.

### The Layer 1 failure taxonomy and how it reaches the loop

`ai.Failure` (`backend/agent/src/ai/provider_failure.go:320-330`): unexported fields `category FailureCategory`, `retryable bool`, `retryAfter RetryDelay`, `rawLabel string`, `statusClass int`, `requestID string`, `cause error`, `delivery DeliveryPath`, `partialOutput bool`. Constructors: `PreStreamFailure` (line 610) and `MidStreamFailure(r FailureReport, outputPreceded bool)` (line 622) — the `outputPreceded` argument IS the partial-output discriminator, stored as `partialOutput`. Accessor `PartialOutput()` exists at line 515 (25 callers across `agenttest`/`openaicompat` conformance suites). `ErrorEvent(f *Failure)` (line 644) wraps `f` as the stream's terminal error payload; `Event.ErrorPayload()` (line 661) is the typed accessor the loop already calls.

`agent.Failure` (`backend/agent/src/agent/failure.go:1-80`, AG-04.2/AG-04.3): a thin wrap `struct{ wrapped *ai.Failure }`. Exposes `Category()` (44), `Delivery()` (54), `Retryable()` (64), `Unwrap()` (75) — **but NOT `PartialOutput()`**. This is a real gap: AG-11.2's charter line ("typed turn failure carrying category, retryability, and partial output") cannot be satisfied by the current `agent.Failure` surface. `NewFailure(f *ai.Failure) (*Failure, error)` (line 33) is the only constructor; a nil `f` is rejected.

**How a mid-stream error reaches the loop today**: `translate()`'s `case ai.EventKindError` (loop.go:583-594) captures `ev.ErrorPayload()` — a `*ai.Failure` — directly into `t.fatal error`. It is never wrapped as `agent.Failure`, never attached to a `TurnEnd`/`RunEnd`, and never re-emitted onto `sink`.

### The typed turn outcome — ALREADY EXISTS, but coarse (major finding)

`TurnOutcome` (`backend/agent/src/agent/turn_events.go:72-89`, from AG-04.2) is NOT something AG-11 must invent — it already exists as a closed `uint8` enum with exactly **two** members: `TurnOutcomeFinished` (79) and `TurnOutcomeAborted` (84). `TurnEnd.validate()` (119-130) enforces "Failure required iff Aborted, forbidden otherwise" — the same `RunOutcome`/`RunEnd` pattern exists in `run_events.go` (`RunOutcomeCompleted`/`RunOutcomeInterrupted`/`RunOutcomeFailed`, three members). `NewTurnEnd(run, turn, outcome, f *Failure)` (135) is the single door in.

AG-11.1's charter ("every normalized finish reason maps to a **distinct** typed turn outcome"; doc 0003:2167 "refusal, pause, and unknown finish reasons produce three distinct behaviors") requires growing this 2-member vocabulary. Because `TurnOutcome`'s bound (`turnOutcomeLimit`) and its `validate()`/`String()` switches live entirely inside `turn_events.go`'s own `const` block, **a new file cannot add members** — Go constants of the same named type can be declared elsewhere, but `validate()`'s `e.outcome >= turnOutcomeLimit` bound-check and `String()`'s switch are private to that file and would reject/mis-render anything appended outside it. `turn_events.go` itself MUST be edited. This is load-bearing for the design phase.

### The event surface — no new event kind needed (major finding)

`failure.go`'s own package comment states the design already made this decision: "AG-04 registers no separate error kind; failures ride the typed outcomes" (turn_events.go / run_events.go both already carry an optional `*Failure` field, validated iff the outcome says so). AG-11.2 therefore needs **zero** new `EventKind`, no `event_descriptor.go` edit, no `event_registry_test.go` addition. It only needs to actually USE the existing `NewTurnEnd(run, turn, TurnOutcomeAborted, failure)` + `NewRunEnd(run, RunOutcomeFailed, failure)` machinery on the mid-stream-fatal path — which today is dead code for that path (never called).

### `invariant_pin_test.go` already names AG-11.2 as a joint closer (major finding)

`backend/agent/src/agent/invariant_pin_test.go:1-8`: "Non-claim, restated per risk 4 (0003:2203): nothing in this file asserts that AG-04 alone closes envelope invariant 1 or invariant 4. Invariant 1 closes jointly with AG-05.1; **invariant 4 closes jointly with AG-11.2**." Invariant 4 = "typed errors" (doc 0003:2162). The file already contains `TestFailure_CategoryAndCause_ReachableAsTypedValues` (107), `TestFailure_Delivery_DistinguishesPreStreamFromMidStream` (131), `TestFailure_CategoryMapping_IsIdentityDeclaredInSource` (169, via `reflect` on `Category()`'s return type), `TestSuite_NeverAssertsOnFailureMessageStringContent` (200, scans every `_test.go` in the package for the literal `ai.Failure.Error()` rendering prefix — R-AEV-008: never assert on message-string content). **None of these test `PartialOutput()`** — confirming the gap above and telling AG-11.2 exactly where its own sibling scenario (e.g. `TestFailure_PartialOutput_ReachableAsTypedValue`) belongs, in the SAME package/file family, to close invariant 4 for real.

### Test harness reality

`agenttest.Provider` (`backend/agent/src/agenttest/fake_provider.go:55-161`): `NewProvider(scripts ...Script)`, `Stream(ctx, req)`, and — critically — `Requests() []ai.Request` (line 157-161, mutex-guarded clone). **`len(provider.Requests())` already gives the exact call count** — no new helper needed for "the fake provider's call count shows exactly ONE call". A mid-stream terminal error after partial content is ALREADY a scripted pattern: `agenttest/conformance_terminal.go:105-161` (`terminalDiscriminatorCase`) does exactly this — `Emit(textBlockStart)`, `Emit(delta)`, `Emit(end)`, `Emit(ai.ErrorEvent(ai.MidStreamFailure(ai.FailureReport{...}, true)))` in one `Script`, then asserts `payload.PartialOutput()`. AG-11.2's own tests reuse this exact idiom directly against `agent.Turn`.

### Known landmines — verified

1. **`TestTurn_SubstrateUntouched`** (`loop_test.go:1125-1150`, `filterOutLoopFiles` at `loop_test.go:831-871`) and **`TestTurn_PreRequestHook_SubstrateUntouched`** (`loop_hook_test.go:812-843`, `filterOutLoopHookFiles` at `loop_hook_test.go:907-943`): both diff `backend/agent/src/agent/` against an AG-07-era base ref and fail if ANYTHING outside an explicit per-milestone exclusion list changed. The current exclusion list (both filters, kept in sync) is: `loop.go, loop_test.go, loop_hook_test.go, loop_tool_dispatch_test.go, tool.go, tool_test.go, scheduler.go, scheduler_test.go, scripted_tool_test.go, permission_protocol.go, permission_protocol_test.go, loop_permission_e2e_test.go, permission_policy_helpers_test.go`. **`turn_events.go` and `failure.go` are NOT on this list.** Since AG-11.1 needs to edit `turn_events.go` (extend `TurnOutcome`) and AG-11.2 needs to edit `failure.go` (add `PartialOutput()`), BOTH filters must be widened to add `turn_events.go` and `failure.go` for the first time — every prior milestone (AG-08 through AG-10) only ever ADDED new files to the exclusion list; AG-11 is the first to need to exclude an EXISTING substrate file that gets genuinely modified. This is a deliberate, unavoidable deviation that the design/tasks phase must record explicitly, the same way AG-09/AG-10 recorded their own deviations in `apply-progress.md`. Any new AG-11 test file also needs adding to both filters, per the standing rule.
2. **`import_boundary_test.go`** (AG-03.2): three closure checks (production-only allowlist, test-inclusive allowlist, network/filesystem denylist). AG-11 introduces no new third-party or cross-module import, so this guard should pass unmodified.
3. **`doc_contract_guard_test.go`** (AG-03.1): asserts `doc.go`'s committed layer-contract table matches row-for-row. Flag for the design phase if the return-type shape of `Turn` changes.
4. **`invariant_pin_test.go`** (AG-04.3): the joint-closure site for invariant 4; likely to gain a new `TestFailure_PartialOutput_...` scenario, and is NOT on the substrate-exclusion list either.
5. **`ambient_authority_test.go`** (AG-03.3): forbids ambient I/O call sites by package. AG-11 adds no I/O; should pass unmodified.

### Approaches

1. **Extend the existing `TurnOutcome` enum in `turn_events.go` to mirror `ai.FinishReason`, one member per distinct behavior, plus keep `Aborted` for the mid-stream-error path** *(recommended)*
   Add `TurnOutcomeToolCalls, TurnOutcomeLengthLimited, TurnOutcomeContentFiltered, TurnOutcomeRefused, TurnOutcomePaused, TurnOutcomeUnknown` alongside the existing `TurnOutcomeFinished`/`TurnOutcomeAborted`, all still `Failure`-forbidden except `Aborted`. `loop.go` gains a small `outcomeForFinish(ai.FinishReason) TurnOutcome` pure function with an exhaustive `switch`, plus the new agent-level drift-guard test walking the same value space. `finalize()` calls it instead of hardcoding `TurnOutcomeFinished`.
   - **Pros**: reuses the exact idiom `RunOutcome`/`TurnOutcome` already establish; the `NewTurnEnd` validator's existing "Failure iff Aborted" rule needs zero change since only `Aborted` ever carries one; `String()` gets a case per member for free diagnostics; the exhaustiveness pin naturally falls out of the `switch`.
   - **Cons**: forces editing `turn_events.go`, a substrate file untouched since AG-04 — requires widening both substrate filters for an EXISTING file, a new kind of exception.
   - **Effort**: Medium.

2. **Keep `TurnOutcome` at 2 members; carry the finish-reason distinction only via a new `FinishReason()` accessor added to `TurnEnd`, sourced from `t.finish`**
   - **Pros**: no change to `TurnOutcome`'s closed vocabulary or validator; smaller diff.
   - **Cons**: does not satisfy "each maps to a **distinct typed outcome**" directly — a consumer still has to switch on `ai.FinishReason` itself to tell refusal from pause from unknown, which is exactly the "collapsing distinct stop conditions into a fallback" defect `finish_reason.go`'s own package doc calls "a loop-termination defect above Layer 1, not a cosmetic gap". Still requires editing `turn_events.go`.
   - **Effort**: Low-Medium.

3. **Introduce a brand-new sealed outcome type in a new file, carried as a second field on `TurnEnd`**
   - **Pros**: avoids editing `turn_events.go`'s existing `const` block.
   - **Cons**: `TurnEnd.validate()` must still learn the new field's rules, so `turn_events.go` is edited anyway; produces two parallel outcome vocabularies on the same event. Rejected as needless duplication.
   - **Effort**: Medium-High, for a worse fit.

### Recommendation

**Approach 1.** It is the only option that satisfies "each finish reason maps to a distinct typed outcome" literally, reuses the `RunOutcome`/`TurnOutcome` idiom this package already established twice (AG-04.1/AG-04.2), and keeps `NewTurnEnd`'s existing "Failure iff Aborted" validation rule completely unchanged. The mid-stream-fatal path in `loop.go` should be rewritten to: (a) wrap `t.fatal` (when it is genuinely an `*ai.Failure` from `ev.ErrorPayload()`, not an internal construction error) via `agent.NewFailure`; (b) reconstruct the partial message the same way `finalize()` already does (do not discard `t.textBracket`/`t.reasoningBracket` fragments); (c) emit `NewTurnEnd(run, turn, TurnOutcomeAborted, failure)` + `NewRunEnd(run, RunOutcomeFailed, failure)` on `sink` before `closeSink`; (d) still return exactly once from `provider.Stream` — no retry, matching `Requests()==1`. `agent.Failure` gains a `PartialOutput() bool` accessor (mirroring `Category`/`Delivery`/`Retryable`'s nil-safe pattern) to close invariant 4 for real.

The design phase must explicitly decide and record: (a) the exact new `TurnOutcome` member names/count (7 total members mirroring `ai.FinishReason` 1:1, or a coarser subset — doc 0003:2167 only strictly requires refusal/pause/unknown to be 3 distinct behaviors); (b) whether the non-`*ai.Failure` branch of `t.fatal` (internal construction errors) also gets wrapped into a typed `Aborted` outcome, or stays a bare Go `error` return; (c) the exact widening additions to `filterOutLoopFiles`/`filterOutLoopHookFiles`.

### Size forecast

Estimated changed lines (code + tests), based on AG-09/AG-10 precedent's dense doc-comment convention:

| Target | Estimate |
| --- | --- |
| `turn_events.go` | +40 to +80 |
| `failure.go` | +10 to +20 |
| `loop.go` | +80 to +160 |
| New agent-level exhaustiveness/drift-guard test | +120 to +200 |
| AG-11.1 scenario tests | +200 to +350 |
| AG-11.2 scenario tests | +250 to +400 |
| Substrate filter widening (both files, in sync) | +10 to +20 |
| Deviation documentation | +30 to +60 |

**Total forecast: roughly 900-1500 changed lines.** Inside the session's 1000-line review budget only at the low end; the design/tasks phase should plan for the budget's stated extension rather than assume it fits at 1000 unmodified.

### Risks

1. Widening `filterOutLoopFiles`/`filterOutLoopHookFiles` to exclude `turn_events.go` and `failure.go` (existing substrate, not new files) is a new KIND of exception vs. every prior milestone's widening. Must be recorded explicitly in the design/tasks docs as a deliberate deviation.
2. `invariant_pin_test.go` is itself the joint-closure site for envelope invariant 4 (R-05) and already declares "invariant 4 closes jointly with AG-11.2" — any new `PartialOutput` scenario should land there (or a clearly cross-referenced sibling) to keep the joint-closure claim auditable.
3. `t.fatal` is not uniformly `*ai.Failure` — internal construction errors share the same field and the same `turn.fatal != nil` branch. The design phase must decide whether AG-11.2's typed-Aborted treatment applies only when `t.fatal` type-asserts to `*ai.Failure`, or covers both cases.
4. The AG-09/AG-10-era tool-call-result-loss gap (`_ = sched.Schedule(...)`, loop.go:265) sits directly adjacent to the code AG-11 must touch — care is needed not to silently "fix" or further entrench it; it is explicitly out of this charter's scope.
5. Deciding the exact cardinality of the extended `TurnOutcome` vocabulary is a real design fork the charter text does not fully resolve — flag explicitly for sdd-propose/sdd-spec, do not silently pick one in apply.

### Ready for Proposal

**Yes.** AG-11 depends only on AG-07 and AG-09 (both archived) and has no unresolved external blocker.

---

## Cross-references

- **AG-10 exploration** (Engram `sdd/cachicamas-agent-permission-protocol/explore`) — already flagged `loop.go:265`'s `_ = sched.Schedule(...)` result-discard; still open, adjacent to but out of AG-11's scope.
- **doc 0003 lines 1113-1176** — AG-11 charter and two Gherkin leaves.
- **doc 0003 lines 2162, 2167-2168, 2203-2206, 2213, 2216** — envelope invariant 4 (R-05), the three-distinct-behaviors checklist row, R-06/R-08/R-15/R-18 cross-references.
- **`backend/agent/src/agent/invariant_pin_test.go:1-8`** — explicit joint-closure statement for invariant 4 with AG-11.2.
- **`backend/agent/src/ai/finish_reason.go`, `finish_reason_test.go:277-320`** — the seven-member closed vocabulary and its existing exhaustiveness pin.
- **`backend/agent/src/agenttest/conformance_terminal.go:105-161`, `fake_provider.go:157-161`** — the scripting and call-count assertion primitives AG-11.2's tests reuse directly.
