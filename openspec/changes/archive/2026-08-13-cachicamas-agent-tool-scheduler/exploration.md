# Exploration — `cachicamas-agent-tool-scheduler` (AG-09)

> Milestone AG-09 (Layer 2 Wave 2, milestone 9 of 24; doc 0003, lines 902–1004). SDD change slug: `cachicamas-agent-tool-scheduler`. Artifact store: **HYBRID** (Engram + OpenSpec). Engram topic key: `sdd/cachicamas-agent-tool-scheduler/explore`. Branch `feat/agent-layer2-wave2-ag09` based at `e27e8411` (post-AG-08 PR #168 merge). Strict TDD ACTIVE. 1000-line budget; `size:exception` pre-authorized (AG-04/05/07/08 standing rule).

## Identity

- **Slug**: `cachicamas-agent-tool-scheduler`
- **Milestone**: AG-09 (Layer 2 Wave 2; doc 0003 § AG-09, lines 902–1004)
- **Branch**: `feat/agent-layer2-wave2-ag09`
- **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag09`
- **Store**: HYBRID — `sdd/cachicamas-agent-tool-scheduler/explore` + filesystem
- **Mode**: automatic (A2 — gatekeeper between phases; no user interruption)
- **Strict TDD**: ACTIVE (carried from AG-04/05/06/07/08 cycle; `openspec/AGENTS.md` `## Strict TDD is on`)
- **Review budget**: 1000 lines, `size:exception` pre-authorized
- **Closes**: **G5** (R-13, parallel tool execution with deterministic call-ordered rejoin + per-tool concurrency policy); seams 2 and 3's Layer 2 anchor (seam 3 lives on the **execution call** as an opaque per-call policy slot Layer 2 never reads — the sandbox seam)
- **Depends on**: AG-05 (tool event family merged), AG-07 (walking skeleton merged at `93077c07`), AG-08 (pre-request hook merged at `e27e8411`)
- **Parallel with**: none (AG-08 just merged; AG-09 follows immediately)
- **Blocks**: AG-10 (permission protocol wraps the scheduler), AG-11 (turn termination types the scheduler's outcomes)
- **Out of scope**: Permission protocol (AG-10 wraps the scheduler around each call); what any tool does (Layer 3 built-in tools per doc 0004); sandbox semantics (Layer 3 interprets the policy slot); the harness that loops the model ↔ tool-result ↔ model turns (AG-13's `Harness`)

## The wording trap (doc 0003:109)

> **"The loop executes tools" is too broad.** The loop *schedules* execution against an injected execution contract and drives the permission protocol around it. What a tool does, whether it is allowed, and under what confinement it runs are all decided above; the loop owns ordering, concurrency, suspension, and the rejoin.

This sentence names AG-09's exact scope. AG-09 ships the scheduler; AG-10 wraps it; AG-13 wraps `Turn` in a harness that loops the provider ↔ scheduler ↔ provider cycle. Every word in the trap is load-bearing for the design choices below.

## Current State

### What exists today

- **Walking skeleton** `agent.Turn(ctx, provider, system, transcript, opts, sink) (msg, finish, err)` at `backend/agent/src/agent/loop.go:133-229`. Stateless across calls (R-LSK-002). Emits events on `sink` via `emitStamped(sink, stamper, ev)`. Closes `sink` before returning. Calls `provider.Stream(ctx, req)` once, drains the channel, translates each `ai.Event` into an agent bracket, emits `turn_end` and `run_end` on completion, returns `(msg, finish, nil)`.
- **Pre-request hook seam** (AG-08, `loop.go:80`, `:175`, `:286-295`): `TurnOptions.PreRequestHook func(ctx, req) (req, error)` invoked between `buildLoopRequest` (`loop.go:162`) and `provider.Stream` (`loop.go:190`); nil = identity default; hook failure aborts before I/O via `*ai.PreStreamFailure`. **2nd consecutive milestone where `loop.go` was modified**; the **envelope/descriptor/validator substrate stays byte-untouched for the 6th consecutive milestone**.
- **Tool execution event family** (AG-05.2, `backend/agent/src/agent/tool_event.go`, 501 lines): 5 event kinds (`ToolStart`, `ToolProgress`, `ToolEndSuccess`, `ToolEndResultFailure`, `ToolEndExecutionFailure`), all `PlacementTurn`, `Terminal:false`. Three typed outcomes via `ToolOutcome` enum (`ToolOutcomeSuccess / ToolOutcomeResultFailure / ToolOutcomeExecutionFailure`, closed at `tool_event.go:227-246`). The call ordinal is payload-side (R-AMT-007), not envelope-side (R-13 → doc 0002 AI-30). `ToolEndExecutionFailure` carries a required typed `*Failure` (R-AEV-008); the other two end states do not.
- **`translate()` default branch** (loop.go:468-472): tool-call and reasoning events currently drop on the floor. Per the comment, "Phase 3 widens the switch (S-LSK-005)" — but reasoning was widened in AG-07.2 (loop.go:402-447). **Tool-call events from the provider stream are still dropped**. AG-09 is the natural widening that consumes `ai.EventKindToolCallStart / Delta / End` (AI-18, `tool_call_event.go:1-336`).
- **Layer 1 substrate**:
  - `ai.ToolCall` (`tool_call.go:45`) — opaque value type with `id`, `name`, `arguments string`. Constructor `NewToolCall(id, name, arguments)` validates `id != ""`, `name != ""`, `arguments` is well-formed JSON. `ToolCalls(content []Part) []ToolCall` derives ordinals from a content sequence's tool-call parts (V-STR-21).
  - `ai.Tool` (`tool.go:20`) — declaration only: `name`, `description`, `schema`, `cacheBoundary bool`. **Does NOT carry an effect class** — that is AG-09's Layer 2 decision.
  - `ai.ToolResult` (`tool_result.go:22`) — `callID`, `content`, `failed bool`. Two constructors: `NewToolResult(id, content)` and `NewToolFailure(id, content)`. "A failing tool is a normal outcome the model must see and reason about" — the register's exact phrasing.
  - AI-18 streamed tool-call events (`tool_call_event.go`): `ToolCallStart` (carries `block`, `id`, `name`), `ToolCallDelta` (carries `block`, `fragment`), `ToolCallEnd` (carries `block`, `arguments`). The ordinal is NOT here (R-ATC-012) — derived by walking the recorded stream.
- **AG-05.3 reconstruction property** (`reconstruction_test.go`): interleaved streams reconstruct independently and completely. Tool call streams and message streams are addressable separately.
- **Test substrate** (`backend/agent/src/agenttest/`):
  - `fake_provider.go:157-161` — `Provider.Requests() []ai.Request` (capture history)
  - `fake_script.go` — `Script{Steps: []Step}` of `Emit(ai.Event)` or `Hold(gate)`
  - `conformance_tool_call.go` — tool-call reconstruction cases (fragmented interleaving, zero-delta whole calls, ordinal distinguishes same name, mixed text + tool)
  - **No scripted-tool fixture** — `agenttest` has the provider and the events but no in-memory `agent.Tool` implementation that test authors can drive (the AG-09 charter says: "tools in tests are scripted in-memory implementations of the AG-09 contract").

### What is missing

1. **A `Tool` interface** — what a tool "is" to Layer 2. Charter phrasing: "an executable with a declaration, an effect class, and a typed failure mode."
2. **An `EffectClass` type** — closed vocabulary of read / mutating / execute. Charter: "at minimum read, mutating, execute — that the scheduler consumes."
3. **A `PolicySlot` type** — opaque per-call policy value the scheduler forwards without reading. Charter: "the execution call carries a policy parameter it does not interpret" (seam 3).
4. **A scheduler** — function or method that:
   - Receives a set of requested calls (id + name + args + effect + policy)
   - Emits `ToolStart` synchronously at execution start (NOT at rejoin — charter scenario AG-09.2 #3)
   - Runs reads concurrently with bounded fan-out (charter AG-09.2 #2)
   - Runs mutating/execute serialized in call order (charter AG-09.2 #1)
   - Rejoins results in call order regardless of completion order (charter AG-09.3 #1)
   - Preserves Layer 1 call/result correlation identities (charter AG-09.3 #2)
   - Isolates panics: one bad tool → siblings complete, typed result in call position (charter AG-09.4 #1 + #2)
5. **A `Schedule` function signature** that takes the requested calls + per-call policy slot + a sink (or returns typed results the loop emits) and returns the typed results in call order.
6. **Loop wire-up**: `Turn`'s `translate()` must convert AI-18's `ToolCallStart / Delta / End` events into an internal `[]ScheduledCall` list, hand them to `Schedule` after the provider stream closes with `FinishReasonToolCalls`, emit the five AG-05.2 events, and return. **Or**: AG-09 ships the scheduler as a standalone callable the loop wires in; the loop iteration between model ↔ tools ↔ model is AG-13's `Harness`'s job.
7. **`TurnOptions.Tools` field** — non-breaking zero-value extension. Zero value = no tools (model's tool calls would surface as protocol error typed at the loop level; out of AG-09's strict scope — see Risks §R3).

## Affected Areas (file-level)

### New files (AG-09 will CREATE)

| Path | Purpose | Forecast lines |
|------|---------|----------------|
| `backend/agent/src/agent/tool.go` | `Tool` interface (`Name()`, `EffectClass()`, `Run(ctx, args, policy) (Result, error)`); `EffectClass` typed enum (3 members); `PolicySlot` named type (`type PolicySlot any`); `Result` value type (3 outcomes — mirrors `ToolOutcome`); ctor validations | ~180 |
| `backend/agent/src/agent/tool_test.go` | Per-charter R-TLS-001..004 / S-TLS-001..003: contract-from-outside; policy passthrough opaque; result vs execution-failure distinct | ~280 |
| `backend/agent/src/agent/scheduler.go` | `Schedule(ctx, calls, policy, sink) []Result` — the runner. Reads concurrent (bounded), mutating+execute serialized (call order), result slots indexed by ordinal, panic containment per call, start events emitted before goroutine spawn | ~340 |
| `backend/agent/src/agent/scheduler_test.go` | R-TLS-005..010 / S-TLS-004..010: bounded fan-out, reads concurrent, writes serialized, ordered rejoin, correlation survives, panic isolation, typed result in call position | ~900 |
| `backend/agent/src/agent/loop_tool_dispatch_test.go` | Loop wire-up: provider streams tool-call events → loop calls `Schedule` → emits tool events on `sink` (AG-09.2 #3 start-event-at-execution-start; AG-09.3 #1 rejoin) | ~280 |
| `backend/agent/src/agenttest/scripted_tool.go` | In-memory `agent.Tool` implementation for tests: configurable behavior per call (success / result-failure / execution-failure / panic / delayed completion), records start time and policy value for assertions | ~180 |
| `backend/agent/src/agenttest/scripted_tool_test.go` | Self-test: scripted tool's own discipline (start recording, policy pass-through, outcome discrimination) | ~120 |

**Subtotal new**: ~2,280 lines. **Single-PR size is unlikely at AG-09 scope** — see Risks §R1.

### Modify

| Path | Why |
|------|-----|
| `backend/agent/src/agent/loop.go` | Add `Tools map[string]Tool` field to `TurnOptions` (or `Tools []Tool`); extend `translate()` to convert AI-18 `ToolCallStart/Delta/End` into a `[]ScheduledCall` list; between `provider.Stream` close and `finalize`, call `Schedule(...)` and emit its returned events via the existing `emitStamped(sink, stamper, ev)` path; update doc comment to record the scheduler insertion. AG-08 already added `PreRequestHook`; AG-09 adds `Tools`. |
| `backend/agent/src/agent/loop_test.go` | Widen `TestTurn_SubstrateUntouched`'s file filter to also exclude `loop_tool_dispatch_test.go` (the AG-07 W3 + AG-08 W3 fix pattern). Filter widening is the right escape hatch; the envelope/descriptor/validator substrate list (21 files) stays unchanged. |
| `backend/agent/src/agent/agent_test_helpers_test.go` | One small addition: `ordinalFromToolStart(events []agent.Event, callID string) uint32` helper (reads the payload-side ordinal, AG-08's analogous `loopRequestSystemText` is the precedent at `loop_hook_test.go:48`). |

### NOT touched (substrate preservation — 6th consecutive milestone)

- `event.go`, `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `event_registry_test.go`, `doc.go`, `doc_contract_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`.

**Streak status**: AG-09 is the **6th consecutive "envelope/descriptor/validator substrate untouched" milestone** (AG-04 → AG-05 → AG-06 → AG-07 → AG-08 → AG-09). Two files outside the strict substrate list ARE touched (`loop.go`, `loop_test.go` filter widening) — both precedent from AG-08.

### Line-count budget posture (compare to AG-08)

| Milestone | Code+test added | Notes |
|-----------|-----------------|-------|
| AG-05 | ~2,479 | tool + message event family |
| AG-06 | ~1,500–2,400 | permission + cost + delegation + compaction families |
| AG-07 | ~1,816 | walking skeleton (loop.go + loop_test.go) |
| AG-08 | 996 | pre-request hook seam + tests; under 1000 budget; `size:exception` accepted |
| **AG-09 (forecast)** | **~2,280 new + ~100 modified = ~2,380** | 4 leaves × 2-3 charter scenarios each + bites + cross-cuts. Likely OVER 1000 budget. |

## Approaches (with pros / cons / effort)

### D1 — `Tool` interface shape

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D1a** | `type Tool interface { Name() string; EffectClass() EffectClass; Run(ctx context.Context, args []byte, policy PolicySlot) (Result, error) }` — three methods; `Run` returns typed `Result` for success/result-failure and `error` only for execution-failure (the substrate already does this with `ToolOutcome`) | Mirrors `ToolOutcome`'s typed-result posture; explicit policy slot in signature (the charter's "execution call carries a policy parameter" demand); `Name()` enables map-keying; `EffectClass()` makes the scheduler's branching obvious; no name parsing needed | Three methods vs. two is a small surface tax | **Low** |
| D1b | `type Tool interface { Name() string; Run(ctx context.Context, args []byte) (Result, error) }` — `EffectClass` carried on `Result` or supplied via a separate `ToolInfo` struct the loop holds | Smaller interface | The scheduler needs the effect BEFORE it schedules — it cannot call `Run` to learn the class. Either a separate `ToolInfo` map (two sources of truth) or `Run` is called speculatively (waste + side effects) | Medium |
| D1c | `type Tool struct { Name string; Effect EffectClass; Run func(...) (Result, error) }` — value type, not interface | Composable; no interface dispatch | Layer 2's `Tools` map would be `map[string]Tool` (values), but the scheduler's call site would need to dereference each call; less idiomatic for "things that execute"; the charter says "an executable" — interface is the natural Go shape | Medium |

**Recommendation: D1a.** The interface with three methods matches the substrate's posture (`ToolOutcome` is typed, not flagged) and makes the policy slot an explicit parameter rather than a hidden field. Mirrors the AG-08 `PreRequestHook` callable surface (function-form), but a `Tool` needs identity + class to participate in the scheduler's branching.

### D2 — `EffectClass` shape

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D2a** | Typed enum `type EffectClass uint8` with three members: `EffectClassRead`, `EffectClassMutating`, `EffectClassExecute`. Closed vocabulary, `String()` method, mirrors `ToolOutcome`'s posture | Matches substrate discipline (closed vocabularies, distinct by kind not convention); a future ADR can add `EffectClassNetwork` or `EffectClassIdempotent` without breaking signatures; bitmask unnecessary | Three-way switch in the scheduler | **Low** |
| D2b | String constant (`const EffectRead = "read"` etc.) | Easy to inspect; no enum needed | String compare at every scheduling decision; not substrate-shaped (Layer 1's effect classes would also be strings if any future Layer 1 effect class arrives) | Low |
| D2c | Bitmask struct (`type EffectClass struct { Read, Mutating, Execute bool }`) | "Read AND non-idempotent" can be expressed | Two members set means the scheduler has to define which wins; not how the charter reads ("read, mutating, execute" — three disjoint classes) | Medium |
| D2d | Struct of three booleans with at-most-one rule | Same as D2c | Same | Medium |

**Recommendation: D2a.** Closed enum matches `ToolOutcome` (tool_event.go:227-246). The charter's floor is "at minimum read, mutating, execute" — three disjoint members. A future "network" effect (if introduced) gets a new ordinal without breaking signatures. AG-10 will likely add `EffectClassPermission` or extend the type — keep it additive.

### D3 — `PolicySlot` shape (the sandbox seam)

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D3a** | `type PolicySlot any` (or `type PolicySlot = any`) — named type over `interface{}` | Documents the semantics at the call site (`policy agent.PolicySlot = mySandbox`); the "opaque to Layer 2" promise is part of the type's doc; future ADR can tighten without breaking signatures | A future tightening (e.g., to an interface or to `[]byte`) would be a breaking change | **Low** |
| D3b | `type PolicySlot []byte` (opaque bytes) | Simple, mirrors `ToolStart.Arguments` | Forces Layer 3 to serialize its sandbox value; loses type information; the doc 0001 § 5.1 quote "confinement is a property of the call site, not of the code being called" is honored by `any` not `[]byte` | Low |
| D3c | Untyped `any` / `interface{}` parameter on `Tool.Run` | Maximum flexibility | No name at the call site — a future audit cannot grep for "PolicySlot" or "policy"; less self-documenting | Low |
| D3d | Typed interface (`type PolicySlot interface { ... }`) | Constrains the surface | Defines a vocabulary Layer 2 doesn't need to know about — exactly the seam 3 violation the charter warns against | Medium |

**Recommendation: D3a.** `type PolicySlot any` (or alias) — the named type is a documentation device and a future-tightening affordance. The charter's "Layer 2 never reads" promise is enforced by the scheduler (which does NOT type-assert the value) and asserted by the test (R-TLS-002, S-TLS-002: scripted tool receives the exact value injected, no transform). Doc 0001 § 5.1 quote: "If the execution seam has nowhere to put a policy, adding one later is a rewrite of every tool — which is why the seam is named now even though the first implementation will be 'none'." Named `any` is the minimal seam.

### D4 — Scheduler concurrency primitives

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D4a** | Hand-rolled: `chan struct{}` semaphore for reads (bounded); a single goroutine + channel-of-one for mutating+execute (serialized); indexed result slice `[]Result` per ordinal for rejoin; `defer/recover` per call goroutine for panic containment; `emitStamped(sink, stamper, ev)` called from inside each goroutine | Stdlib-only (no new deps; honors NFR-PRH-002); explicit fan-out bound (matches "bounded fan-out" charter); explicit serialization order (matches "mutating/execute serialized in call order"); the indexed-result pattern is the simplest "rejoin in call order regardless of completion order" | Larger code than errgroup; careful mutex discipline needed for the result slice (writes happen from N goroutines) | **Medium** |
| D4b | `golang.org/x/sync/errgroup` + a serialized sub-group | Smaller code; first-error cancellation built in | New top-level dep (`errgroup`) — the project forbids new deps without an ADR (`openspec/AGENTS.md` `## Hard rules`); first-error semantics CONFLICT with the charter's "siblings complete" requirement (AG-09.4 #1) | Medium |
| D4c | Worker pool with semaphore + task channel | Bounded concurrency "by construction" | Harder to express "reads concurrent, writes serialized" (two classes need two pools); more code than D4a | Medium |

**Recommendation: D4a.** No new deps (NFR-PRH-002 carry); explicit concurrency shape matches the charter's three policies (read=concurrent-bounded, mutate=serialized, execute=serialized); the indexed-result pattern is the cleanest "rejoin in call order regardless of completion order" with the smallest test surface. AG-09.2 #2's bounded-fan-out test is one semaphore acquire + a `runtime.NumGoroutine()` assertion (AG-07 W1 carry pattern).

### D5 — Rejoin shape (charter AG-09.3 #1)

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D5a** | Indexed result slice: `results := make([]Result, len(calls))`; each call goroutine writes `results[call.ordinal] = res`; the scheduler returns `results` in slice order | O(n) memory; trivial rejoin loop; the "rejoin in call order" property is the slice's own iteration order; call identity (`callID`) is preserved inside each `Result` | If a call panics and the defer doesn't write, the slot stays zero — the scheduler must write a typed execution-failure result even on panic | **Low** |
| D5b | Sorted-channel receive (priority queue keyed by ordinal) | Memory-frugal | Reintroduces a heap dependency; over-engineered at n < ~100 | Medium |
| D5c | `sync.WaitGroup` + atomic counter | Standard pattern | Rejoin order requires an extra sort step; race-prone if not careful | Medium |

**Recommendation: D5a.** Indexed slice is the natural data structure for "rejoin in call order regardless of completion order". The panic containment (D7) writes a typed result before returning — every ordinal slot is always populated. AG-09.3 #2's correlation test asserts `results[i].CallID() == calls[i].callID` after the scheduler returns.

### D6 — Panic containment (charter AG-09.4 #2)

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D6a** | Explicit `defer func() { if r := recover(); r != nil { results[ordinal] = Result{Outcome: ExecutionFailure, Failure: NewFailure(MidStreamFailure{...})} } }()` in every call goroutine | Stdlib-only; the typed failure is constructed before the recover returns; honors "under the race detector, the panic is contained" — no second goroutine dies | Slightly more code than `errgroup`; the recover path must itself not panic (use simple constructor calls) | **Low** |
| D6b | `errgroup` with `recover()` middleware | Built-in | Same dep / cancellation conflict as D4b | Medium |
| D6c | `runtime/debug.Stack()` capture into the failure | Useful for debugging | Adds runtime/debug dep, which the AG-03.3 ambient-authority guard may scrutinize | Medium |

**Recommendation: D6a.** Stdlib-only. The recovery path constructs a typed `ToolEndExecutionFailure` event with `*Failure{Category: ai.FailureCategoryUnavailable, Delivery: ai.DeliveryMidStream, Retryable: false}` (mirrors AG-04's typed failure surface at `failure.go:24-79`). AG-09.4 #2's race-detector proof: a script tool that calls `panic("boom")` produces (a) the typed event on the sink, (b) sibling results in their ordinal slots, (c) the scheduler returns the slice — `runtime.NumGoroutine()` returns to baseline.

### D7 — Spec prefix

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D7a** | `R-TLS-` (tool-scheduler) — new prefix | Two-letter match to slug; precedent is per-milestone prefix (AG-04 R-AEV, AG-05 R-AMT, AG-06 R-APE, AG-07 R-LSK, AG-08 R-PRH) | None at AG-09 scope | **Low** |
| D7b | Extend `R-LSK-` | One less prefix | AG-08 already opened `R-PRH-` for the same reason (separate concern from the skeleton); consistency favors per-milestone prefix | Low |
| D7c | `R-SCH-` (scheduler) | Generic name | Doesn't name the milestone's concern | Low |

**Recommendation: D7a.** `R-TLS-` (tool-scheduler) / `S-TLS-NNN`. Per-milestone prefix continues the AG-04/05/06/07/08 pattern.

### D8 — Loop wire-up shape

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D8a** | `Turn` calls `Schedule` once per turn, between `provider.Stream` close and `finalize`. The model iterates: provider emits tool-call events → `translate()` accumulates them → on `ai.Completion{FinishReasonToolCalls}` the loop calls `Schedule` → emits the five tool events → returns `(msg, FinishReasonToolCalls, nil)`. **AG-13 wraps `Turn` in a `Harness` that loops the cycle (model ↔ tools ↔ model).** | Narrow AG-09 scope (one cycle's tools, no iteration logic); substrate-preserving (no new kinds, no envelope edits); aligns with the wording trap (loop schedules, doesn't iterate) | The loop returning `FinishReasonToolCalls` is a NEW behavior — AG-09 must extend `Turn`'s post-completion path; AG-11 will formalize the dispatch (R-TLS-007 carries forward) | **Medium** |
| D8b | `Turn` loops: model → tools → model until `FinishReasonStop / Refusal / Pause`. | Smaller AG-13 | AG-09 owns two-iteration logic + state + re-emit logic; "loop" starts being a misnomer; substrate changes; the wording trap is violated (loop iterates, doesn't just schedule) | High |
| D8c | `Schedule` is a pure function; the loop emits events into a slice, calls `Schedule` on the slice, then re-emits the events from the returned `[]Result` on the sink. No scheduler goroutines; the loop does the goroutines. | Simpler scheduler | The scheduler IS the orchestration — extracting goroutine management to the loop blurs the seam | Medium |

**Recommendation: D8a.** The wording trap is the whole design. AG-09 ships the scheduler; AG-09 also wires `Turn` to call it ONCE per turn for the tools that the model requested. AG-13 (harness) owns the iteration. AG-11 owns the finish-reason dispatch (which already cites AG-09 as a dependency at doc 0003:1150). This split keeps AG-09 ≤ 1000 lines (with `size:exception`) and substrate-preserving.

### D9 — `TurnOptions.Tools` shape

| Option | Description | Pros | Cons | Effort |
|--------|-------------|------|------|--------|
| **D9a** | `Tools map[string]Tool` — keyed by name; `Tools[name]` resolves a model request to an implementation | O(1) resolution; matches the model's "name → tool" lookup shape; `nil` zero value = no tools | Layer 2 doesn't own name validation (Layer 1 `ai.ToolCall.name` rule); collision on duplicate names is the caller's responsibility | **Low** |
| D9b | `Tools []Tool` (slice, scheduler scans by name) | Order-preserving | O(n) resolution; the scheduler would scan linearly per call | Low |
| D9c | `ToolSource` interface (G6 seam) — `Tools() []Tool` returns the available tools per turn | Honors doc 0003 § 5.1 port definition; Layer 3 supplies tools dynamically | AG-06 already implemented permission family for the AG-09 charter; the `ToolSource` port is G6 / doc 0004's territory, not AG-09's | Medium |

**Recommendation: D9a.** `map[string]Tool` keeps AG-09's scope narrow. The `ToolSource` port (G6) is a future AG-13 widening; the loop's `Tools` field can be populated by either an injected map or a `ToolSource`-backed implementation without changing the field's signature.

## Recommendation

**Ready for Proposal: YES** (with the pre-commit confirmations below).

AG-09's scope is large (4 leaves, 11 charter Gherkin scenarios, 6 substrate-preserving files untouched + 2 modified). The recommended approach surface is:

1. **D1a — `Tool` interface with three methods** (`Name()`, `EffectClass()`, `Run(ctx, args, policy) (Result, error)`). Mirrors `ToolOutcome`'s typed-result discipline.
2. **D2a — `EffectClass` typed enum** (3 members: Read / Mutating / Execute; closed; `String()`; mirrors `ToolOutcome`).
3. **D3a — `PolicySlot any`** (named type over `any`; the "Layer 2 never reads" promise is the type's doc + a test).
4. **D4a — hand-rolled scheduler** (`chan struct{}` semaphore + serialized channel + indexed `[]Result` + `defer/recover`). No new top-level deps (NFR-PRH-002).
5. **D5a — indexed result slice** (the call ordinal is the slice index; rejoin is `for i, r := range results { … }`).
6. **D6a — explicit `defer/recover` per call goroutine** (writes typed `ToolEndExecutionFailure` into the result slot before returning).
7. **D7a — `R-TLS-` / `S-TLS-NNN`** spec prefix. Per-milestone convention.
8. **D8a — `Turn` calls `Schedule` once per turn; AG-13 owns iteration**. Honors the wording trap.
9. **D9a — `Tools map[string]Tool`** on `TurnOptions` (zero value = no tools; nil-safe).

### Carry-forwards AG-09 inherits from AG-04/05/06/07/08

| Source | Finding | AG-09 mitigation |
|--------|---------|------------------|
| **AG-08 W1** | Back-pressure path unproven (only one unbuffered-sink test, AG-08 S-PRH-007) | **MUST** add at least one scheduler test with unbuffered `sink` + concurrent consumer + `runtime.NumGoroutine()` baseline. The scheduler's fan-out is the natural place to prove concurrent reads under back-pressure |
| **AG-08 W2** | NFR-PRH-002 spec prose drift (corrected at archive time) | NFR-TLS-001 inherits: every behavioral test in `package agent_test` |
| **AG-08 W3** | Substrate-untouched env-var fallback (`AG08_BASE_REF`) | NFR-TLS-003 inherits: `AG09_BASE_REF` env-var + dynamic `git merge-base HEAD origin/main` |
| **AG-08 SUGG 1** | `drainSink` (`loop_test.go:147`) has no timeout | **MUST** carry — the scheduler's results drain via the sink; add `select` deadline inside `drainSink` so future tests fail fast on regressions (AG-09's first named consumer) |
| **AG-08 SUGG 2** | `loopRequestSystemText` / `systemIncludesSegment` helpers could be promoted | Defer — AG-09 introduces its own helpers (`ordinalFromToolStart`) local to `loop_tool_dispatch_test.go`; promotion at AG-23 archive time |
| **AG-08 SUGG 3** | `applyPreRequestHook` 100% covered by bites | `Schedule` should be similarly covered by S-TLS-004a/b bites before S-TLS-004 GREEN — defense against vacuous scheduling |
| **AG-05 W1** | Vacuous reconstruction helper | Apply bite pattern to R-TLS-006 (rejoin in call order): bite (a) out-of-order completions → rejoin still in order; bite (b) inverted completions → rejoin still in order; both bites RED-recorded BEFORE S-TLS-006 GREEN |
| **AG-04 W9** | Scenario count drift | State identically: **11 charter scenarios → ~14–18 spec + ~6 bites** |
| **AG-04 W8** | Loop coverage ≥ 80% | Forecast ~150 lines added to `loop.go`; current coverage 86.13% (AG-08 final); AG-09 must hold ≥ 80% |
| **AG-07 W4** | `mintLoopMessageID` swallows errors | Not AG-09's scope (carries to AG-23) |
| **AG-07 SUGG 4** | `translate()` method form | AG-09 **widens** the existing `translate()` switch with `ai.EventKindToolCallStart / Delta / End` cases — exactly what the SUGG predicted (the hook seam made `translate()` "naturally" need a method form). Defer the method-form refactor to AG-23 or later; AG-09's scope is widening the switch, not refactoring it |

### Reference: AG-08 substrate-untouched test pattern (NFR-PRH-003 → NFR-TLS-003)

```go
// AG-08's pattern — lifted from loop_hook_test.go (5cca989f)
envRef := os.Getenv("AG08_BASE_REF")
if envRef == "" {
    out, err := exec.Command("git", "merge-base", "HEAD", "origin/main").CombinedOutput()
    if err != nil { t.Skip("no AG08_BASE_REF and no merge-base available: ", err) }
    envRef = strings.TrimSpace(string(out))
}
substrate := []string{ /* 21 paths */ }
out, _ := exec.Command("git", "diff", envRef, "--", append(substrate, "go.mod", "go.sum", "Makefile", ".golangci.yml")...).CombinedOutput()
if len(out) != 0 { t.Fatalf("substrate touched:\n%s", out) }
```

**AG-09's filter widening**: `TestTurn_SubstrateUntouched` (AG-07) currently excludes `loop.go` and `loop_test.go`; AG-08 widened it to also exclude `loop_hook_test.go`. **AG-09 widens again** to exclude `loop_tool_dispatch_test.go` (the new family file). The 21-path substrate list itself is unchanged.

## Risks

### Quantified risks

#### R1 — Size (forecast over budget) [WARNING]

**Forecast**: ~2,280 new + ~100 modified = ~2,380 lines for the cycle (compare AG-08's 996). The 1000-line `size:exception` budget is **likely insufficient** even with `size:exception` accepted (AG-08 landed at 996 — 4 lines under). Two options:

- **R1a — Single PR with `size:exception` at 2,500+ lines**: possible but review-burnout territory. AG-05/AG-06 precedent of single 2,000+ line PRs holds, but review focus suffers.
- **R1b — Chained PRs**: AG-09.1 + AG-09.2 as PR #1 (the contract + scheduler + reads-concurrent + writes-serialized = ~1,400 lines); AG-09.3 + AG-09.4 as PR #2 (ordered rejoin + panic containment + wire-up tests = ~900 lines). Each PR is reviewable in 30–60 minutes. **Recommended.** braejan's standing rule (`work-unit-commits` skill) supports chained PRs for > 1000 line work.

**Mitigation**: surface R1b in the proposal. The chained split mirrors the doc 0003 mermaid (AG-09.1 → AG-09.2 → AG-09.3, AG-09.1 → AG-09.4). The two PRs share the same branch; AG-09's spec/design/tasks split accordingly.

#### R2 — Layer 1 effect class collision [WARNING]

Layer 1's `ai.Tool` declaration (`tool.go:20`) does NOT carry an effect class. If a future Layer 1 milestone (e.g., a tool-source / tool-schema milestone in doc 0002 wave 3) adds an effect class to `ai.Tool`, AG-09's Layer 2 `EffectClass` would either (a) collide, or (b) become a redundant copy. **Two paths**:
- (a) AG-09's `Tool.EffectClass()` method reads from a Layer 2-internal effect class registry (the registry lives in package `agent`; `ai.Tool` stays untouched).
- (b) A future Layer 1 amendment moves effect class up to `ai.Tool`, and AG-09's contract becomes the canonical source.

**Mitigation**: surface in the proposal that AG-09 owns the effect class registry (option a). A future Layer 1 amendment can move it up if the design warrants — this is recorded as an ADR-able cross-layer concern.

#### R3 — Zero-value `Tools` (model emits tool call, no tool registered) [WARNING]

If `opts.Tools` is nil and the provider stream emits AI-18 tool-call events, the loop's `translate()` accumulates calls into a `[]ScheduledCall` with no implementation. The scheduler will return a typed result for each (R-TLS-NFR: no tool = `ToolOutcomeExecutionFailure` with `*Failure{Category: ai.FailureCategoryUnsupportedCapability}`). The result rejoins in call order; the loop emits `ToolEndExecutionFailure` events on the sink.

**Scope question**: is "no tool registered → typed failure" AG-09's job, or AG-13's? **Recommendation**: AG-09 ships the typed-failure path (the scheduler's contract is "produce a typed result for every call, including a no-implementation case"). This is consistent with the "one bad tool does not abort the turn" charter scenario AG-09.4 #1 — a "no tool" case is structurally identical to an execution failure.

#### R4 — Substrate-untouched streak: filter widening [SUGGESTION]

The filter widening pattern (file-granularity escape hatch for new files in the same family) has been carried forward at every milestone since AG-07 W3. AG-09 widens again. **Cumulative**: by AG-23 (archive phase), the substrate-untouched test will have been widened 5 times. **Mitigation**: a single `agent.SubstrateUntouched(baseRef)` helper that takes the filter set as a parameter would consolidate; defer to AG-23 (matches AG-08 SUGG 2 → AG-23 deferral).

#### R5 — `drainSink` timeout carry-forward [SUGGESTION]

`drainSink` (`loop_test.go:147`) has no timeout — same observation as AG-07 SUGG 1 and AG-08 SUGG 1. AG-09 is the **first named consumer of the scheduler's results via `sink`**; the AG-09 unbuffered-sink test will need its own deadline. **Mitigation**: add the `select` deadline inside `drainSink` once — closes AG-07 SUGG 1, AG-08 SUGG 1, and AG-09's need. Tiny refactor; the substrate-untouched test does not need widening for this change (it's `loop_test.go`, not in the 21-path list).

#### R6 — Wire-up scope creep into AG-13 territory [CRITICAL if not surfaced]

AG-09's charter says "schedule requested calls" — not "iterate the loop between model and tools". The wording trap reinforces this. **Risk**: an AG-09 implementation that wires `Turn` to loop model ↔ tools ↔ model will OWN harness behavior, eating AG-13's scope.

**Mitigation**: D8a (one cycle's tools; AG-13 iterates). Surface in the proposal with explicit "AG-09 ships ONE cycle; iteration is AG-13's job." The scheduler function signature `Schedule(ctx, calls, sink) []Result` is the seam — it can be called from `Turn` (AG-09) or from `Harness` (AG-13) or both.

#### R7 — Policy slot opaque-to-Layer-2 promise [WARNING]

The charter says the policy slot is opaque. The scheduler must NOT type-assert the value. The test (R-TLS-002, S-TLS-002) proves a scripted tool receives the EXACT value injected. **Future risk**: a future contributor adds a "convenience" check like `if p, ok := policy.(*SandboxPolicy); ok { ... }` inside the scheduler. **Mitigation**: the test asserts no transformation (`bytes.Equal(policyBytes, observedBytes)`); plus an explicit guard test that scans the scheduler's source for any type assertion on `PolicySlot`. **Pattern**: AG-04's "no payload-convention inference" guard for `ToolEndSuccess` vs `ToolEndResultFailure` (R-AMT-006, S-AMT-050).

#### R8 — Concurrent reads under `-race` [WARNING]

The AG-22 sweep test pattern requires every test to pass under `-race`. The scheduler's concurrent reads will fire `-race` warnings if any shared state is touched without synchronization. **Mitigation**: the `[]Result` slice is the only shared state. Writes happen via `atomic.StoreUintptr` (per-slot) or via a mutex-guarded slice. Read after `Schedule` returns is race-free (the `sync.WaitGroup` inside the scheduler releases before the function returns). The scripted tool's own state uses a `sync.Mutex`. Each emit goes through `emitStamped(sink, stamper, ev)` — but the `LaneStamper` is single-owner to the loop, so concurrent goroutines MUST NOT stamp their own events. **Mitigation**: the scheduler emits a single goroutine (the dispatcher), not each call goroutine — preserves the single-writer invariant from `sequence.go:8-24`.

### Unknowns to verify before design

- **U1**: Whether AG-09.1's "effect class at minimum read, mutating, execute" is a closed vocabulary or an open one. **Mitigation**: D2a (closed enum) is the substrate-shaped default; the design notes "the floor, not the ceiling" — a future ADR can add `EffectClassNetwork` or `EffectClassIdempotent` without breaking signatures.
- **U2**: Whether the scheduler's call goroutines share a context or each gets its own derived `context.WithCancel`. **Mitigation**: one shared `ctx` is simpler and the cancellation story is cleaner (cancelling the turn cancels all in-flight calls). AG-11 (typed failure) and AG-13 (cancellation tree) will formalize; AG-09 ships the single-ctx pattern.
- **U3**: Whether the scheduler returns `[]Result` or emits events directly on a passed `chan<- *Event`. **Mitigation**: `[]Result` is cleaner for tests (assertion surface is a slice), keeps the scheduler free of envelope concerns (the loop emits the events). The loop calls `emitStamped(sink, stamper, ev)` for each result in rejoin order. This keeps the stamper single-owner.
- **U4**: Whether `Schedule`'s panic containment writes a typed `Failure` with `Category: ai.FailureCategoryUnavailable` (the substrate's closest match) or introduces a new `ai.FailureCategoryPanic` category. **Mitigation**: **introduce a new category** if `Unavailable` doesn't fit semantically (a panic is not "unavailable"). Surface as U4; recommend an ADR for the new category if needed, but AG-09 can ship reusing `Unavailable` as a placeholder and the ADR can land later without breaking signatures.

## Ready for Proposal

**YES.** The substrate is well-understood (AG-07 walking skeleton stable, AG-08 pre-request hook stable, AI-12 `Request.With` stable, AI-18 streamed tool-call events stable). AG-08's warnings are explicit carry-forwards; AG-09's specific risks are quantified; the open decisions (D1–D9) are listed with cited evidence.

The orchestrator should launch `sdd-propose` next, with the following **pre-commit confirmations to surface to braejan before proposing**:

1. **D1 (tool interface shape)** — three methods (`Name()`, `EffectClass()`, `Run(ctx, args, policy) (Result, error)`), NOT a struct value, NOT a two-method interface.
2. **D2 (effect class shape)** — typed enum `EffectClass` (Read / Mutating / Execute), NOT a string, NOT a bitmask.
3. **D3 (policy slot shape)** — `type PolicySlot any` (named type over `any`), NOT raw `any`, NOT `[]byte`.
4. **D4 (scheduler primitives)** — hand-rolled (`chan struct{}` semaphore + serialized channel + indexed `[]Result`), NOT `errgroup` (forbidden new dep + cancellation conflict).
5. **D7 (spec prefix)** — open `R-TLS-` (tool-scheduler), distinct from `R-LSK-` (skeleton) and `R-PRH-` (pre-request hook).
6. **D8 (loop wire-up)** — `Turn` calls `Schedule` once per turn; **AG-13 owns iteration**. The wording trap is the design.
7. **R1 (size)** — chained PRs (AG-09.1 + AG-09.2 in PR #1, AG-09.3 + AG-09.4 in PR #2). Each PR ≤ 1,400 lines.

The orchestrator should also confirm:
- **U4 (panic failure category)** — reuse `ai.FailureCategoryUnavailable` as AG-09's placeholder, file ADR-able cross-layer concern for a future `FailureCategoryPanic` if the semantic gap matters.
- **R3 (zero-value `Tools` path)** — model emits tool call, no tool registered → typed execution failure (consistent with AG-09.4 #1 "one bad tool does not abort the turn").

## Evidence (every claim cites file:line or doc section)

### Charter and architecture

- **Charter**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:902-1004` (AG-09, 11 charter Gherkin scenarios in 4 leaves, lines 928-1001)
- **Charter mermaid**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:914-921` (AG-09.1 → AG-09.2 → AG-09.3; AG-09.1 → AG-09.4)
- **Wording trap**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:107-112` ("the loop *schedules* execution against an injected execution contract")
- **v2 architecture § 7 G5 row**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:699` ("parallel tool execution with deterministic, call-ordered re-join and per-tool concurrency policy")
- **v2 architecture § 6 seam 2 + seam 3**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:651-665` (seam 2 = permission decision on scheduling path; seam 3 = sandbox policy on the execution call, "confinement is a property of the call site, not of the code being called")
- **v2 architecture § 5.1 sandbox quote**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:613-616` ("`Sandbox` must be a parameter of tool *execution*, not of a tool. ... If the execution seam has nowhere to put a policy, adding one later is a rewrite of every tool")
- **v2 architecture § 4.1 loop owns**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:475-486` ("scheduling tool execution, including concurrency policy ... bounded concurrency, and the ordered re-join")
- **Forward sequence diagram (fan-out, rejoin)**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:309-321` (the L→L "fan out — reads parallel, writes and shell serialized" + "re-join in CALL order, not completion order" lines)

### Layer 2 substrate (AG-04/05/06/07/08)

- **AG-05.2 tool execution family**: `backend/agent/src/agent/tool_event.go:1-501` (5 kinds, 3 typed outcomes, payload-side ordinal)
- **ToolOutcome enum**: `backend/agent/src/agent/tool_event.go:227-246` (closed 3-member vocabulary)
- **Walking skeleton**: `backend/agent/src/agent/loop.go:133-229` (`Turn`); `:162` (`buildLoopRequest`); `:175-186` (AG-08 hook seam); `:190` (`provider.Stream`); `:468-472` (`translate()` default branch — tool-call events still dropped)
- **AG-08 hook surface**: `backend/agent/src/agent/loop.go:53-81` (`TurnOptions.PreRequestHook`); `:175` (call site); `:286-295` (`applyPreRequestHook`)
- **AG-08 substrate-untouched filter widening pattern**: `backend/agent/src/agent/loop_hook_test.go` (substrate test uses `AG08_BASE_REF` env-var fallback + dynamic `git merge-base`)
- **AG-08 carry-forwards**: `openspec/changes/archive/2026-08-13-cachicamas-agent-pre-request-hook/archive-report.md:84-91` (W1 → unbuffered sink; SUGG 1 → drainSink timeout; SUGG 2 → helper promotion)
- **AG-07 carry-forwards**: `openspec/changes/archive/2026-08-13-cachicamas-agent-loop-skeleton/archive-report.md` (W1 → back-pressure path; W6 → external posture; bite pattern)
- **AG-05 W1 bite pattern**: `backend/agent/src/agent/reconstruction_test.go:180-277` (`S-AMT-071`/`S-AMT-072` RED-recorded BEFORE `S-AMT-051` GREEN)
- **Typed failure surface**: `backend/agent/src/agent/failure.go:24-79` (`Failure` thin wrap over `*ai.Failure`)
- **Envelope identity pattern**: `backend/agent/src/agent/event.go:454-462` (`Event{payload, seq, run, turn, hasTurn, parent, hasParent}`)
- **Sequence per-lane (single-writer invariant)**: `backend/agent/src/agent/sequence.go:8-24` ("a [LaneStamper] belongs to exactly one lane and is touched only by that lane's own forwarding activity — exactly one goroutine")
- **Stream validator two-level scope**: `backend/agent/src/agent/stream_check.go:92-185`

### Layer 1 substrate

- **AI-09.1 ToolCall**: `backend/agent/src/ai/tool_call.go:45` (opaque value type); `:227-234` (`ToolCalls` derives ordinals)
- **AI-08 Tool declaration**: `backend/agent/src/ai/tool.go:20-25` (`name`, `description`, `schema`, `cacheBoundary bool` — NO effect class)
- **AI-09.3 ToolResult**: `backend/agent/src/ai/tool_result.go:22-25` (`callID`, `content`, `failed bool`); `:77-79` (`NewToolResult`); `:105-107` (`NewToolFailure` — "a failing tool is a normal outcome the model must see")
- **AI-18 streamed tool-call events**: `backend/agent/src/ai/tool_call_event.go:74-336` (`ToolCallStart` carries `id` + `name`; `ToolCallDelta` carries `fragment`; `ToolCallEnd` carries `arguments`); R-ATC-012 (ordinal NOT here — derived stream-side)
- **AI-12 copy-on-write rebuild**: `backend/agent/src/ai/request.go:325-336` (`With(...)` derives from `r.options`)

### Test substrate (AI-21, AI-22)

- **Captured request history**: `backend/agent/src/agenttest/fake_provider.go:157-161` (`Provider.Requests() []ai.Request`)
- **Scripted stream**: `backend/agent/src/agenttest/fake_script.go` (`Script{Steps: []Step}` of `Emit(ai.Event)` or `Hold(gate)`)
- **Conformance tool-call cases**: `backend/agent/src/agenttest/conformance_tool_call.go:21-26` (4 cases: fragmented interleave, zero-delta whole, ordinal distinguishes same name, mixed text+tool)
- **Stream kit**: `backend/agent/src/agenttest/stream_kit_iter.go`, `stream_kit_ordering.go`, `stream_kit_leak.go`, `stream_kit_record.go`, `stream_kit_diff.go`

### Layer 1 ↔ Layer 2 substrate discipline

- **No new top-level Go deps**: `openspec/AGENTS.md` `## Hard rules` ("Do not propose new top-level Go dependencies without an ADR")
- **Strict TDD**: `openspec/AGENTS.md` `## Strict TDD is on`
- **Layer 2 import boundary**: `backend/agent/src/agent/import_boundary_test.go` (production closure: stdlib + `src/ai` + OTel API)
- **Layer 2 ambient authority**: `backend/agent/src/agent/ambient_authority_test.go` (no `os`/`os/exec`/`net/http` in non-test sources)

## Key Learnings

1. The wording trap (doc 0003:109) is the whole design — AG-09 *schedules* execution, AG-10 wraps permission around each call, AG-13 iterates the model � tools ↔ model cycle. AG-09 must NOT ship iteration logic; AG-13 must NOT own scheduling.
2. The AG-08 hook-seam pattern (a single callable on `TurnOptions`, typed result from a closure, nil = identity default) is the precedent for AG-09's `Tools map[string]Tool` field. Non-breaking zero-value extension; AG-09 widens the surface, doesn't reshape it.
3. `Request.With(...)` (AI-12) copy-on-write rebuild is irrelevant to AG-09 — the scheduler does NOT modify the request. AG-09 is the second live consumer of the AI-18 streamed tool-call events (after AG-05.3's reconstruction property); the AG-09 dispatch must widen `translate()`'s switch on `ai.EventKindToolCallStart / Delta / End` (currently dropped at `loop.go:468-472`).
4. The "policy slot opaque" promise (charter AG-09.1 #2 + v2 § 5.1) is enforced by (a) `type PolicySlot any` documentation, (b) the scheduler never type-asserts the value, (c) the test asserts scripted tool receives the exact value injected byte-for-byte, (d) a future guard test scans the scheduler's source for any type assertion on `PolicySlot`.
5. AG-09 will be the **6th consecutive "envelope/descriptor/validator substrate untouched" milestone** (AG-04 → AG-09). The substrate-untouched test widens its filter (file-granularity escape hatch) one more time to exclude `loop_tool_dispatch_test.go`. The 21-path list itself is unchanged.
6. The `LaneStamper` single-writer invariant (`sequence.go:8-24`) means concurrent goroutines MUST NOT stamp their own events. The scheduler's emit path is one dispatcher goroutine, not per-call — preserves the invariant from AG-04.2 (W3 latent-trap guard).
7. `errgroup` is forbidden (new top-level dep + first-error cancellation conflicts with AG-09.4 #1's "siblings complete"). Hand-rolled `chan struct{}` semaphore + serialized channel is the substrate-shaped alternative. Mirrors the AG-08 hook-seam precedent (no new deps).
8. The `drainSink` timeout carry-forward (AG-07 SUGG 1 → AG-08 SUGG 1 → AG-09) lands at AG-09: AG-09 is the first named consumer of the scheduler's results via `sink`, and a back-pressure test needs a deadline to fail fast. A 1-line `select` refactor closes the carry forward.
9. AG-09's 4 leaves + 11 charter scenarios forecast 2,000-2,400 lines — likely over the 1000-line budget even with `size:exception`. Chained PRs (AG-09.1 + AG-09.2 in PR #1, AG-09.3 + AG-09.4 in PR #2) are the substrate-shaped delivery. Mirrors the doc 0003 mermaid exactly.
10. AG-09 must NOT add a Layer 1 effect class to `ai.Tool` (Layer 1 stays untouched, AG-04/05/06/07/08/09 substrate preservation rule). The Layer 2 `agent.EffectClass` lives in package `agent`; `Tool.EffectClass()` method returns it per-implementation. A future Layer 1 amendment can move the type up.
11. The `[]Result` indexed-slice rejoin pattern is the cleanest "rejoin in call order regardless of completion order" data structure — O(n) memory, O(n) rejoin loop, panic containment writes a typed result into the slot before returning. The bite pattern (R-TLS-006: out-of-order completions → rejoin still in order) is the load-bearing property test.
12. AG-08's substrate-untouched pattern (env-var `AG08_BASE_REF` fallback + dynamic `git merge-base HEAD origin/main`) is the precedent for AG-09's `AG09_BASE_REF`. Cumulative filter widening is intentional per AG-07's per-milestone-author pattern; consolidation at AG-23 archive time.
