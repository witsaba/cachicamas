# Spec — `cachicamas-chat-tool-source` (CH-09)

> **Change**: `cachicamas-chat-tool-source` · **CH-09** (Wave 3, 10 of 12) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-09--offer-tools-through-a-tool-source-port) (`0005:923-934`) · leaves CH-09.1 `[leaf]`, CH-09.2 `[leaf]`, CH-09.3 `[leaf]`, CH-09.4 `[leaf]`, CH-09.5 `[guard]`
> **Closes**: R-03 (the tool seam's real answer; the empty answer at CH-00 § 3 row 4 of `decision.md` is replaced), R-09's seam (the chat-owned `ToolSource` port) · **Depends on**: CH-05 · **Blocks**: CH-10 (`cachicamas-chat-permission`, the policy port).
> **Status**: **new capability**, promoted verbatim to `openspec/specs/cachicamas-chat-tool-source/spec.md` at archive. Additive amendments land in `chat-conversation-store` (R-CCS-015/016, NFR-CCS-008, S-CCS-019..022) and `frontend-chat-layer1` (REQ-8..11, S-FCL-012..017); identifier-append-only per CH-07 / CH-08 precedent.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable. The `S-CTS-NNN` scenarios are transcribed verbatim from the explore report's child-node Gherkin (CH-09.1..5); the `S-CTT-NNN` scenarios (CH-09.1's CurrentTimeTool) are transcribed verbatim from the same source — both prefixes live in this file because the tool-under-test is named at scenario granularity.
> **IDs**: requirements `R-CTS-NNN`, scenarios `S-CTS-NNN` (port, wire projection, frontend mirror, guards) and `S-CTT-NNN` (CurrentTimeTool itself, a sub-prefix), non-functional `NFR-CTS-NNN`. **Append-only.**
> **Allocated ranges**: `R-CTS-001` … `R-CTS-099`; `S-CTS-001` … `S-CTS-199`; `NFR-CTS-001` … `NFR-CTS-099`. Sub-prefix `S-CTT-001` … `S-CTT-099` for CurrentTimeTool-specific scenarios. The range is reserved; ranges, never totals.
> **Prefix verification**: `CTS` verified collision-free across `openspec/specs/`, `openspec/changes/`, `openspec/changes/archive/` and `docs/` (proposal #3959). `CTT` is the explore's verbatim sub-prefix; no spec file outside this one uses it.
> **Evidence gate**: `cd backend/agent && make test` (uncached; race-clean) plus `make lint` and `make build/chat`. Frontend: `pnpm --filter @cachicamas/frontend test:ci` plus `pnpm lint` and `pnpm build.types`. Postgres cross-process test (`S-CCS-021`) gated `INTEGRATION=1`.

## Purpose

CH-00.1 (`decision.md:88`, *"Empty through Wave 1 and Wave 2: the registry resolves no name, so every scheduled call yields a typed unresolved-name failure rather than a crash. CH-09 replaces it with a real tool-source port and at least one tool the model can call"*) recorded the chat archetype's tool seam as deliberately empty through Wave 1 and Wave 2. CH-09 is the milestone that closes that emptiness: it ships the chat-owned `chat.ToolSource` port (D-1), the first tool (`chat.CurrentTimeTool`, D-2), the wire shape that carries tool calls and results to the browser (D-3 + D-6 collapse), the persistence widening that keeps the transcript faithful across reload (D-5), and the frontend delta that renders tool entries from both the live stream and the reload surface (D-4 + D-7 union widening). Substrate preservation (NFR-TLS-003): zero files under `backend/agent/src/agent/` are modified — chat depends on `agent.Registry` by interface, never by import-of-internals.

The acceptance criterion is observable end-to-end (`0005:931`): given a conversation whose tool source offers one tool, when the model calls it, then the call and its result appear on the participant's stream and the turn continues with the result in the transcript. The Gherkin at CH-09.1..5 in this spec is the binding contract; the scenario names (`S-CTS-001..024`, `S-CTT-001..003`) are the test names in `chat/tool_source_test.go`, `chat/current_time_test.go`, `chat/projection_tool_test.go`, `chat/store_tool_roundtrip_test.go`, `frontend/src/lib/chat-types.spec.ts`, `frontend/src/lib/chat-api.spec.ts`, `frontend/src/components/chat/use-chat-stream.spec.tsx`, and `frontend/src/components/chat/chat-app.spec.tsx`. The CTS-* frontend mirror scenarios (S-CTS-013, S-CTS-014, S-CTS-019..022) are also re-stated as **S-FCL-012..017** in the `frontend-chat-layer1` amendment for completeness — `S-CTS-*` is the source of truth, `S-FCL-*` references it.

## Coverage — charter and decisions traced

| Charter / decision | Requirements | Scenarios |
|---|---|---|
| Charter `0005:931` acceptance (one tool → model calls it → call + result on stream → transcript continues) | R-CTS-001, R-CTS-002, R-CTS-003, R-CTS-004, R-CTS-005, R-CTS-006 | S-CTS-001..003, S-CTT-001..003, S-CTS-006..012, S-CCS-019..022 |
| D-1 port shape (`chat.ToolSource` wraps `agent.Registry` by interface) | R-CTS-001, R-CTS-002, NFR-CTS-001 | S-CTS-001..003 |
| D-2 first tool (`current_time`, RFC3339, injectable `NowFunc`) | R-CTS-003, NFR-CTS-001 | S-CTT-001..003 |
| D-3 wire shape (new SSE event names + DTOs; **new variants, not new fields**) | R-CTS-004, R-CTS-005, REQ-8..11 in `frontend-chat-layer1` amendment | S-CTS-006, S-CTS-007, S-CTS-013, S-CTS-014 |
| D-4 rendering (separate transcript entry between turns) | R-CTS-005, REQ-8..11 in `frontend-chat-layer1` amendment | S-CTS-013, S-CTS-019, S-CTS-020 |
| D-5 persistence (persist both call and result; `Exchange` widens) | R-CTS-006, R-CTS-007, R-CCS-015/016 in `chat-conversation-store` amendment | S-CCS-019..022 |
| D-6 wire collapse (5 Layer-2 events → 2 chat-side events; `EventKindToolProgress` dropped at the chat wire) | R-CTS-004, R-CTS-005, NFR-CTS-002 | S-CTS-009..012 |
| D-7 union widening (`REQ-7` forbids new fields on existing variants; CH-09 adds new variants on the closed `ChatStreamEvent` union) | REQ-8..11 in `frontend-chat-layer1` amendment (explicit "new variants, not new fields" wording) | S-CTS-024 |
| D-8 re-billing deferred (v1 static source; cache prefix byte-stable per conversation) | R-CTS-008 | (no scenario — accepted by construction at `cmd/chat/main.go:227-245` factory closure) |
| Substrate preservation (NFR-TLS-003) | NFR-CTS-003 | S-CTS-023 |
| Wire-fragmentation guard (parseTranscript switch covers all 9 event names) | R-CTS-004 (rationale), NFR-CTS-002 | S-CTS-024 |

## Requirements

### R-CTS-001 — `chat.ToolSource` is the chat package's tool-source port

The system MUST expose a `chat.ToolSource` interface in `backend/agent/src/chat/tool_source.go` (D-1). `Config.ToolSource` MUST be added to `chat.Config` (`conversation.go:34-63`) and `NewConversation` MUST reject a nil `ToolSource` with `ErrNilToolSource` (typed, never panic). The interface shape MUST be `Resolve(name string) (agent.Tool, bool)` — byte-identical signature to `agent.Registry.Resolve` (`backend/agent/src/agent/tool.go:267-269`) so the composition root can hand the chat port an `agent.Registry` value through `chat.FromAgentRegistry`. The chat package MUST depend on `agent.Registry` by interface, not by import-of-internals (NFR-CTS-001, CH-08 precedent at `chat-conversation-store` `R-CCS-002`).

#### Scenario: S-CTS-001 — `Config{ToolSource: nil}` is refused, never panics (Gherkin verbatim, explore #3952)

- Given a `Config{Provider, Store, ParticipantID, ToolSource: nil}`
- When `NewConversation(cfg)` runs
- Then it returns `(*Conversation, ErrNilToolSource)` typed, never panicking

### R-CTS-002 — `chat.FromAgentRegistry(agent.Registry)` adapts Layer 2's registry to chat's port

The system MUST expose `chat.FromAgentRegistry(agent.Registry) chat.ToolSource` in `backend/agent/src/chat/tool_source.go`. The adapter MUST be nil-safe: `FromAgentRegistry(nil)` returns a `ToolSource` whose `Resolve` always returns `(nil, false)` without panic. The adapter MUST hold the registry internally and delegate `Resolve` to it byte-for-byte (`S-CTS-003`). The composition-root factory closure at `cmd/chat/main.go:227-245` gains exactly one line — `ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(map[string]agent.Tool{"current_time": chat.NewCurrentTimeTool(time.Now)}))` — and is otherwise byte-unchanged.

#### Scenario: S-CTS-002 — `FromAgentRegistry(nil)` is nil-safe (Gherkin verbatim, explore #3952)

- Given `FromAgentRegistry(agent.NewMapRegistry(nil))`
- When `Resolve("any")` runs
- Then it returns `(nil, false)` and does not panic

#### Scenario: S-CTS-003 — `FromAgentRegistry` delegates to the wrapped registry byte-equal (Gherkin verbatim, explore #3952)

- Given `FromAgentRegistry(agent.NewMapRegistry(map[string]agent.Tool{"echo": &EchoTool{}}))`
- When `Resolve("echo")` runs
- Then it returns `(*EchoTool, true)` byte-equal to the wrapped map's value

### R-CTS-003 — `chat.CurrentTimeTool` implements `agent.tool` with injectable `NowFunc`

The system MUST expose `chat.CurrentTimeTool` (`backend/agent/src/chat/current_time.go`, D-2) implementing `agent.Tool` (`backend/agent/src/agent/tool.go:182-186`). The constructor MUST be `NewCurrentTimeTool(now func() time.Time) *CurrentTimeTool`. `Name()` MUST return the literal `"current_time"`. `EffectClass()` MUST return `agent.EffectClassRead`. `Run(ctx, args, policy)` MUST accept `args []byte` whose JSON-decoded shape is `{}` (no fields; a non-empty JSON object MUST yield `Result{Outcome: ToolOutcomeResultFailure, Content: <error>}` — never silently ignored); on the empty-object path it MUST return `Result{Outcome: ToolOutcomeSuccess, Content: now().Format(time.RFC3339)}`. The `NowFunc` injection point is the testability seam: every test injects a fixed `time.Time`.

#### Scenario: S-CTT-001 — `CurrentTimeTool.Run` returns RFC3339 with empty args (Gherkin verbatim, explore #3952)

- Given a `CurrentTimeTool{NowFunc: func() time.Time { return fixedTime }}`
- When `Run(ctx, []byte("{}"), nil)` runs
- Then `Result.Content` is `fixedTime.Format(time.RFC3339)` and `Result.Outcome == ToolOutcomeSuccess` and `Result.CallID()` is empty (the scheduler fills it)

#### Scenario: S-CTT-002 — `CurrentTimeTool.Run` refuses unknown args (Gherkin verbatim, explore #3952)

- Given a `CurrentTimeTool`
- When `Run(ctx, []byte(`{"timezone":"UTC"}`), nil)` runs
- Then it returns `Result{Outcome: ToolOutcomeResultFailure, Content: <error message>}` (args JSON-validated against `{}` schema), never silently ignored

#### Scenario: S-CTT-003 — `CurrentTimeTool` exposes the closed-typed surface (Gherkin verbatim, explore #3952)

- Given a `CurrentTimeTool`
- When `EffectClass()` runs
- Then it returns `agent.EffectClassRead`
- And when `Name()` runs, then it returns `"current_time"`

### R-CTS-004 — `chat.WireEvent` widens additively with four new variants

The system MUST add four new variants to the closed `chat.WireEvent` interface (`backend/agent/src/chat/wire.go:13-15`):

- `ToolCallStart{WireCallID string, Tool string, Arguments string}` — projected from `agent.EventKindToolStart` (`event.go:161-162`); `WireCallID` carries `tool_call_id`, `Tool` carries `tool_name`, `Arguments` carries the `[]byte → string` projection (`S-CTS-009`, `S-CTS-010`).
- `ToolResult{WireCallID string, Tool string, Outcome string, Content string, FailureCategory string}` — projected from the three `ToolEnd*` kinds (`S-CTS-010..012`). `Outcome` is one of `"success"`, `"result_failure"`, `"execution_failure"` (the closed vocabulary at `tool_event.go:227-246`); `FailureCategory` is non-empty ONLY when `Outcome == "execution_failure"`, mirroring R-CCP-008 / D6's "no provider text leaks" rule.
- `ToolCallDelta{WireCallID string, Delta string}` — **reserved for future progress-bearing tools** (D-6); v1 does not emit this variant. See `chat/wire.go` and `NFR-CTS-002`.
- `ToolCallEnd{WireCallID string, Outcome string}` — **reserved** for v2 dynamic-source surfaces; v1 collapses `ToolEnd*` into `ToolResult` (D-6). See `NFR-CTS-002`.

Each new variant MUST carry an `isWireEvent()` marker (mirroring `wire.go:25, 35, 46, 57, 69`). The four new `wireFrameName` cases (`tool.call.start`, `tool.call.delta`, `tool.call.end`, `tool.result`) MUST extend the exhaustive switch at `eventsource.go:31-48`; a new variant without a case is a runtime panic from the `default` branch (Go 1.26 does NOT enforce type-switch exhaustiveness at compile time when a `default` arm exists — the structural RED holds at runtime; see `S-CTS-007`).

#### Scenario: S-CTS-006 — `ToolCallStart` serialises to the closed SSE shape (Gherkin verbatim, explore #3952)

- Given a `ToolCallStart{WireCallID: "c1", Tool: "current_time", Arguments: "{}"}`
- When `writeFrame(w, flusher, ev)` serializes via `httptest.ResponseRecorder`
- Then the SSE bytes are exactly `event: tool.call.start\ndata: {"wireCallId":"c1","tool":"current_time","arguments":"{}"}\n\n` (lowercase JSON keys, per the closed `ExchangeDTO` precedent at `chat-types.ts:152-167`)

#### Scenario: S-CTS-007 — a new variant without a `wireFrameName` case panics at runtime (Gherkin verbatim, explore #3952; **F-CHT-9.2 wording amendment**)

- Given a new variant added to `WireEvent` without updating `wireFrameName`
- When `wireFrameName(newVariant{})` runs at runtime
- Then it panics naming the missing case (the `default` branch fires; **Go 1.26 does NOT enforce type-switch exhaustiveness at compile time when a `default` arm exists** — the build passes with a missing case; the runtime panic from the `default` branch is the binding invariant; the test `TestWire_AllNewVariants_SerialiseViaWireFrameName` enforces it by round-tripping every variant). `S-CTS-024` binds the inverse — adding a case without a parser arm surfaces as a frontend drop.

### R-CTS-005 — The chat projector collapses Layer 2's tool bracket into chat-side events

The chat projector at `backend/agent/src/chat/projection.go:47-105` MUST add five new case arms: `agent.EventKindToolStart` → `ToolCallStart{WireCallID, Tool, Arguments}`; `agent.EventKindToolProgress` → **dropped at the chat wire** (D-6); `agent.EventKindToolEndSuccess` → `ToolResult{WireCallID, Tool, "success", Result, ""}`; `agent.EventKindToolEndResultFailure` → `ToolResult{WireCallID, Tool, "result_failure", Result, ""}`; `agent.EventKindToolEndExecutionFailure` → `ToolResult{WireCallID, Tool, "execution_failure", "", FailureCategory.String()}` (no provider text leaks — R-CCP-008 / D6 mirror). The "dropped at the chat wire" posture is non-negotiable: a chat wire `ToolCallDelta` from `ToolProgress` would be a leak (`S-CTS-009`). `ToolResult` MUST carry the original `WireCallID` byte-equal to `ToolCallStart.WireCallID` so the frontend can correlate (S-CTS-010..012).

#### Scenario: S-CTS-009 — `ToolStart` alone yields exactly one chat-side event (Gherkin verbatim, explore #3952)

- Given a recorded Layer 2 stream `[ToolStart(callID="c1", name="current_time", args="{}")]`
- When the projector drains it
- Then the wire channel emits exactly one `ToolCallStart` and no `ToolCallDelta` or `ToolCallEnd` — `ToolStart` carries no content, so a chat wire `ToolCallDelta` would be a leak

#### Scenario: S-CTS-010 — `ToolStart` + `ToolEndSuccess` collapse to two chat-side events (Gherkin verbatim, explore #3952)

- Given `[ToolStart, ToolEndSuccess(callID="c1", content="2026-08-25T07:17:00Z")]`
- When projected
- Then the wire carries `ToolCallStart` and `ToolResult{Outcome: "success", Content: "2026-08-25T07:17:00Z"}` and NO `ToolCallEnd` (End becomes Result)

#### Scenario: S-CTS-011 — `ToolStart` + `ToolEndResultFailure` collapse to two chat-side events (Gherkin verbatim, explore #3952)

- Given `[ToolStart, ToolEndResultFailure(callID="c1", content="bad args")]`
- When projected
- Then the wire carries `ToolResult{Outcome: "result_failure", Content: "bad args"}`

#### Scenario: S-CTS-012 — `ToolStart` + `ToolEndExecutionFailure` collapse to two chat-side events with typed failure category (Gherkin verbatim, explore #3952)

- Given `[ToolStart, ToolEndExecutionFailure(callID="c1", failure=CategoryInvalidArgument)]`
- When projected
- Then the wire carries `ToolResult{Outcome: "execution_failure", FailureCategory: "invalid_argument"}` (no provider text leaks — R-CCP-008 / D6 mirror)

### R-CTS-006 — `chat.Exchange` widens with `[]ToolCallRecord` and `[]ToolResultRecord`

The `chat.Exchange` struct (`backend/agent/src/chat/store.go:41-50`) MUST widen additively with two new fields — `ToolCalls []ToolCallRecord` and `ToolResults []ToolResultRecord` — where `ToolCallRecord` carries `WireCallID, Tool, Arguments` and `ToolResultRecord` carries `WireCallID, Tool, Outcome, Content, FailureCategory` (port-side projections of the wire DTOs). The eight pre-existing fields MUST remain byte-unchanged (`R-CCS-010` closed-port posture; identifier-append-only per CH-07 / CH-08). `MemoryConversationStore.Append` MUST round-trip the new fields; `MemoryConversationStore.Load` MUST return a defensive copy of both slices (NFR-CCS-008 carries NFR-CCS-004 forward). `PostgresConversationStore` MUST persist them via sibling tables `chat_tool_calls` + `chat_tool_results` keyed by `(participant_id, exchange_position)` (NFR-CCS-006 forward-only — no `ALTER` of pre-existing tables).

### R-CTS-007 — `RunConversationStoreScenarios` extends with tool-call round-trip scenarios

The shared scenario helper `RunConversationStoreScenarios` (`backend/agent/src/chat/store_scenarios_test.go`, R-CCS-012 / CH-08.2 precedent) MUST be extended with the four scenarios transcribed in the `chat-conversation-store` amendment (`S-CCS-019..022`). The scenarios MUST run against BOTH `MemoryConversationStore` and `PostgresConversationStore` with scenario text unchanged (R-CCS-012 pattern). `S-CCS-021` is gated `INTEGRATION=1` (postgres cross-process round-trip); `S-CCS-019, S-CCS-020, S-CCS-022` run unconditionally.

### R-CTS-008 — v1's tool source is static; the cache prefix is byte-stable per conversation

The composition-root factory closure at `cmd/chat/main.go:227-245` MUST construct the `chat.ToolSource` exactly once per process, via `chat.FromAgentRegistry(agent.NewMapRegistry(...))`. Across N turns in the same conversation, the registry's tool list MUST be byte-stable (V-REQ-14 / `ai.ToolSet` deterministic ordering). The v2 dynamic-source surface (MCP tool source where `Resolve` may return different tools between turns) is **deferred** to a future spec per doc 0005 register row 5 + ADR 0009 § D4; this spec records the v1 answer and registers the v2 attachment point. Acceptance is by construction (no scenario): the factory closure's `ToolSource:` line is constructed at process start; the harness's `Tools` field at `Harness.Turn` (`harness.go:62`) is set once in `NewConversation` and never re-set.

## Non-functional requirements

### NFR-CTS-001 — `chat.ToolSource` wraps `agent.Registry` by interface

The chat package MUST depend on `agent.Registry` (`backend/agent/src/agent/tool.go:267-269`) by interface, never by import of Layer 2's concrete `mapRegistry` type (`tool.go:280-286`). A static import-boundary check or its equivalent (the existing `backend/agent/src/agent/import_boundary_test.go` admits the chat package importing `agent`; CH-09 does NOT widen the allowlist) MUST prove the absence of an `agent.mapRegistry` reference from `chat/`.

### NFR-CTS-002 — Wire collapse is documented at the seam

`EventKindToolProgress` events from Layer 2 MUST NOT reach the browser at v1: `chat/projection.go`'s switch drops them via the `default` arm at `:80-84`, not via an explicit `case` (an explicit `case ToolProgress:` MUST NOT exist in the file). The four `WireEvent` variants from `R-CTS-004` are the complete chat-side vocabulary for tool calls; future tools needing progress (e.g. a long-running MCP tool) reserve `ToolCallDelta` (REQ-9 in `frontend-chat-layer1`) and `ToolCallEnd` (REQ-10) for that surface — those variants are reserved but unused at v1.

### NFR-CTS-003 — Substrate preservation

Zero files under `backend/agent/src/agent/` MUST be modified. The ten-file substrate list (`event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`) survives byte-unchanged. Verified by `git diff --stat main..HEAD -- backend/agent/src/agent/` (must be empty; `S-CTS-023`). New code lives under `backend/agent/src/chat/` + `cmd/chat/`.

## Scenarios — frontend and substrate guards

The frontend-mirror scenarios (`S-CTS-013, S-CTS-014, S-CTS-019..022`) are also stated as **S-FCL-012..017** in the `frontend-chat-layer1` additive amendment; `S-CTS-*` is the source of truth and the amendment references it.

#### Scenario: S-CTS-013 — `parseTranscript` parses `tool.call.start` into a typed variant (Gherkin verbatim, explore #3952)

- Given `parseTranscript` reading a `tool.call.start` frame with payload `{"wireCallId":"c1","tool":"current_time","arguments":"{}"}`
- When the frame is parsed
- Then the resulting `ChatStreamEvent` is `{kind: "tool.call.start", wireCallId: "c1", tool: "current_time", arguments: "{}"}` (typed and exhaustive)

#### Scenario: S-CTS-014 — a malformed `tool.call.start` JSON is silently dropped (Gherkin verbatim, explore #3952)

- Given a `tool.call.start` frame with malformed JSON
- When parsed
- Then the frame is silently dropped (S-1.b mirror at `chat-types.ts:206-210`)

#### Scenario: S-CTS-019 — `exchangesToEntries` renders tool entries from `ExchangeDTO` (Gherkin verbatim, explore #3952 — F-CHT-9.1 aligned)

- Given an `ExchangeDTO{ToolCalls: [c1], ToolResults: [r1]}`
- When `exchangesToEntries([exchange])` runs
- Then the returned array contains exactly one `kind: "tool"` entry with `tool: c1.tool`, `args: parseArgs(c1.arguments)`, `result: r1.content`, `state: "done"` — appended AFTER the assistant said entry

#### Scenario: S-CTS-020 — a live `tool.call.start` SSE frame opens a "running" tool entry (Gherkin verbatim, explore #3952)

- Given a `tool.call.start` SSE frame arriving mid-turn
- When the page receives it
- Then a new transcript entry with `kind: "tool"`, `state: "running"`, the tool name and parsed args is appended

#### Scenario: S-CTS-021 — a `tool.result` with `execution_failure` closes the entry as "failed" (Gherkin verbatim, explore #3952)

- Given the matching `tool.result` frame with `outcome: "execution_failure"`
- When received
- Then the tool entry's `state` becomes `"failed"` and `result` carries the failure phrase (no provider text — R-CCP-008 / D6 mirror on the wire)

#### Scenario: S-CTS-022 — `turn.end` after a tool execution allows the next assistant bubble to stream tool-result-aware text (**F-CHT-9.3 wording amendment** — "by construction" via assistantId-keyed delta accumulation; the `finishReason: "tool_calls"` field is the model's signal but is not explicitly gated in the hook)

- Given a tool call whose model emits `turn.end` after the `tool.result` frame
- When the page receives the `turn.end` frame (any `finishReason`, including `"tool_calls"`)
- Then the assistant text bubble that follows the tool entry continues to accumulate any subsequent `message.delta` frames — keyed on the original `assistantId` captured in `submit()`'s closure at `use-chat-stream.ts:163-181`. The entry's `state` resolves to `"final"` on `turn.end` (handler at `use-chat-stream.ts:269-281`), but `message.delta` accumulation at `use-chat-stream.ts:202-209` is **finishReason-agnostic** — the next delta appends to the same entry rather than opening a new bubble. This is the "by construction" behaviour: any continuation delta lands in the prior assistant entry because the delta-accumulation switch matches on `e.id === assistantId`, never on `finishReason`. A future contributor adding an early-return on `finishReason === "tool_calls"` would silently break the scenario; the covering test at `use-chat-stream.spec.tsx` (`S-CTS-022: turn.end{finishReason: "tool_calls"} does NOT cancel assistantId-keyed delta accumulation`) is the trip.

#### Scenario: S-CTS-023 — substrate-untouched guard (Gherkin verbatim, explore #3952)

- Given the merged change
- When `make test` runs
- Then the substrate-unaffected test passes (analogous to `TestTurn_SubstrateUntouched` in `loop_test.go` but for the chat-archetype path)
- And `git diff --stat main..HEAD -- backend/agent/src/agent/` is empty

#### Scenario: S-CTS-024 — wire-fragmentation guard (Gherkin verbatim, explore #3952)

- Given the merged `chat-types.ts`
- When `pnpm test:ci` runs
- Then the wire-fragmentation test passes (the `parseTranscript` switch covers all 9 known event names: `message.start`, `message.delta`, `message.end`, `turn.end`, `error`, `tool.call.start`, `tool.call.delta`, `tool.call.end`, `tool.result`; a new variant without a case is a TypeScript compile error from `assertNever`)

## Acceptance — proposal success criteria mapped to scenarios

| Acceptance criterion | Evidence |
|---|---|
| Charter `0005:931` acceptance (model calls tool → call + result on stream → transcript continues) | S-CTS-001..003, S-CTT-001..003, S-CTS-006..012, S-CCS-019..022 |
| D-1 port shape (chat owns `ToolSource`; wraps `agent.Registry` by interface) | S-CTS-001..003 |
| D-2 first tool (`current_time`, RFC3339, injectable clock) | S-CTT-001..003 |
| D-3 wire shape (4 new `WireEvent` variants; lowercase JSON keys; compile-fail on missing `wireFrameName` case) | S-CTS-006, S-CTS-007 |
| D-4 rendering (separate transcript entry between turns) | S-CTS-013, S-CTS-019, S-CTS-020 |
| D-5 persistence (Exchange widens with `[]ToolCallRecord` + `[]ToolResultRecord`) | S-CCS-019..022 |
| D-6 wire collapse (5 → 2 chat-side events per call; `ToolProgress` dropped) | S-CTS-009..012 |
| D-7 union widening (4 new variants on closed `ChatStreamEvent`, NOT new fields on existing variants) | S-CTS-013, S-CTS-014, S-CTS-024 (and REQ-8..11 in `frontend-chat-layer1` amendment) |
| D-8 re-billing deferred (v1 static source; cache prefix byte-stable per conversation) | R-CTS-008 (acceptance by construction at `cmd/chat/main.go:227-245` factory closure) |
| Substrate preservation (NFR-TLS-003 — 10-file list byte-unchanged) | NFR-CTS-003, S-CTS-023 |
| Wire-fragmentation guard (parseTranscript covers all 9 event names) | NFR-CTS-002, S-CTS-024 |
| `cd backend/agent && make test` race-clean (uncached; cached runs are not evidence) | (whole spec) |
| `cd backend/agent && make lint && make build/chat` clean | (whole spec) |
| Frontend `pnpm --filter @cachicamas/frontend test:ci` green; `pnpm lint` 0/0; `pnpm build.types` clean | (whole spec) |
| Postgres cross-process (`S-CCS-021`) gated `INTEGRATION=1` | S-CCS-021 |

## Explicit non-requirements

| Not required here | Why, and who owns it |
|---|---|
| MCP tool sources (registry row 5; ADR 0009 § D4 attaches here) | Charter `0005:933` — deferred against this port. Future spec |
| Sandboxing / policy injection (`agent.Tool.PolicySlot` interpretation) | Charter `0005:933`; v2 § 6 seam 3. CH-10 owns the policy |
| The coding archetype's tools | Owned by doc 0004 (`cachicamas-ai-coding-archetype`) — out of scope |
| Re-billing question (v2 § 5.1:624-627; tool-source changes mid-session invalidating cache prefix) | D-8: v1's static source makes the question moot; deferred to a future spec |
| Widening the closed `agent.Registry` interface (`tool.go:267-269`) | D-1: chassis is `chat.ToolSource`; `agent.Registry` is the executor-side seam, untouched |
| Touching any of the 10 substrate files under `backend/agent/src/agent/` | NFR-CTS-003, NFR-TLS-003 |
| Adding new top-level Go dependencies | CH-07's `pgx/v5` + `pressly/goose/v3` cover the postgres surface; no new deps at CH-09 |
| A second constructor `NewConversationWithToolSource` | D-1: `Config.ToolSource` keeps the choice in the factory closure |
| A dynamic tool-source surface (mid-conversation tool set changes) | D-8: v1 is static; v2 attaches here per ADR 0009 § D4 |
| `tool.call.delta` / `tool.call.end` emission at v1 | NFR-CTS-002: reserved variants for future progress-bearing tools |
| Branching / renaming / searching conversation records | Out of scope for tool surface; deferred |
| A live acceptance test against a real provider | CH-04.3 owns the only live check; CH-09 stays deterministic |
| Repairing the citation defect noted in the **Spec defects** section below | Recorded, not repaired (CH-00 `F-1`/`F-2`/`F-3` pattern) |

## Spec defects found (recorded, not repaired)

- **F-CHT-9.1 (state name discrepancy) — RESOLVED at design phase (#3965 §13)**. `S-CTS-019` originally said the rendered entry's `state` is `"complete"`, but the existing `TranscriptLine` component at `frontend/src/components/chat/transcript-line.tsx:42` and `frontend/src/lib/mock/chat.ts:42` declare `"done"` — spec text and implementation disagreed. The design phase resolved this by aligning the spec text from `"complete"` to `"done"` (one-line amendment in the explore report's scenario wording). This spec carries the aligned wording verbatim. No code ripple; tests assert `"done"` per the implementation.
- **F-CHT-9.2 (S-CTS-007 wording) — RESOLVED at verify recovery (this commit)**. `S-CTS-007` originally read "When `go build ./backend/agent/...` runs, Then compile fails naming the default branch", implying compile-time exhaustiveness. Go 1.26 does NOT enforce type-switch exhaustiveness at compile time when a `default` arm exists — the build passes with a missing case; the runtime panics from the `default` branch. The amendment aligns the scenario wording to "panics naming the missing case" + an explicit note about Go 1.26's runtime-not-compile-time exhaustiveness + a reference to the runtime probe test `TestWire_AllNewVariants_SerialiseViaWireFrameName` that enforces the invariant. The structural binding holds at runtime; only the spec wording was imprecise. Mirrored in `frontend-chat-layer1/spec.md` via the underlying `REQ-8..11` rationale lines (no S-FCL-* mirror of S-CTS-007 needed because the type-switch exhaustiveness is a Go-side concern).
- **F-CHT-9.3 (S-CTS-022 wording) — RESOLVED at verify recovery (this commit)**. `S-CTS-022` originally read "When the page receives `turn.end` with `finishReason: 'tool_calls'`, Then the assistant text bubble that follows the tool entry is allowed to stream and accumulate the model's tool-result-aware text", implying an explicit gate. Implementation is "by construction" via assistantId-keyed delta accumulation at `use-chat-stream.ts:202-209` and `use-chat-stream.ts:269-281` — the `finishReason: "tool_calls"` value carries the model's signal but is not explicitly checked; the continuation works because the delta-accumulation switch matches on `e.id === assistantId`, not on `finishReason`. The amendment aligns the scenario wording to "any `finishReason`" + the "by construction" rationale + the covering test (`use-chat-stream.spec.tsx` `S-CTS-022`) that trips any future early-return on `finishReason === "tool_calls"`. Mirrored in `frontend-chat-layer1/spec.md` S-FCL-017.

## Cross-references

- Doc 0005 § CH-09 charter (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:923-934`).
- CH-00.1 empty tool seam answer at `openspec/changes/archive/2026-08-23-cachicamas-chat-vocabulary-and-scope/decision.md:88` — CH-09 replaces it.
- `backend/agent/src/agent/tool.go:267-269` — `agent.Registry` interface that `chat.ToolSource` wraps.
- `backend/agent/src/agent/tool_event.go:61-501` — the 5-event tool family at Layer 2.
- `backend/agent/src/agent/event.go:161-176` — the closed tool event kinds.
- `backend/agent/src/chat/wire.go:13-69` — the closed 5-variant `WireEvent` union CH-09 widens.
- `backend/agent/src/chat/projection.go:80-84` — current `default` arm where tool events are dropped; CH-09 replaces four of the five drops with explicit `case` arms (the fifth, `EventKindToolProgress`, stays dropped per D-6).
- `backend/agent/src/chat/eventsource.go:31-48` — `wireFrameName` exhaustive switch; CH-09 adds four cases.
- `backend/agent/src/chat/store.go:41-50` — closed 8-field `Exchange`; CH-09 widens with two new fields.
- `backend/agent/src/chat/store.go:106-131` — closed two-method `ConversationStore` interface (R-CCS-010 additively widened to N+1 by R-CCS-013); CH-09 widens `Exchange`, not the port.
- `backend/agent/src/chat/store.go:174-225` — `MemoryConversationStore`; CH-09 extends Append/Load with the two new fields.
- `backend/agent/src/chat/conversation.go:34-63` — `Config` struct; CH-09 adds `ToolSource` field.
- `backend/agent/src/cmd/chat/main.go:227-245` — composition-root factory closure; CH-09 gains one `ToolSource:` line.
- `frontend/src/lib/chat-types.ts:8-32` — closed `ChatStreamEvent` union preamble; REQ-7 forbids new fields on existing variants; CH-09 widens with new variants only (D-7).
- `frontend/src/lib/chat-types.ts:212-279` — `parseTranscript` switch; CH-09 extends with four new `case` arms + the new variants on the union.
- `frontend/src/lib/chat-api.ts:285-291` — `KNOWN_EVENTS` constant; CH-09 extends the four new event names.
- `frontend/src/components/chat/transcript-line.tsx:120-161` — `tool` variant branch (`state: "running" | "done" | "denied" | "failed"`); CH-09 feeds that variant.
- `frontend/src/components/chat/chat-app.tsx:66-87` — `exchangesToEntries`; CH-09 extends to render tool entries from `ExchangeDTO.ToolCalls` + `ToolResults` after the assistant said entry.
- `openspec/specs/chat-conversation-store/spec.md:9` — identifier-append-only rule that CH-09 follows.
- `openspec/specs/chat-conversation-store/spec.md:307-313` — CH-08 additive widening precedent (R-CCS-013/014); CH-09 mirrors.
- `openspec/specs/chat-archetype-contract/spec.md:121` — citation discipline for `tool.go:265-266` ("wraps and completes", not "truncated"); CH-09 inherits.
- `openspec/AGENTS.md` "Substrate preservation in `backend/agent` (NFR-TLS-003)" — the 10-file list CH-09 leaves byte-unchanged; CH-09 appends its one-line pointer here per the AG-10 / AG-11 / CH-07 / CH-08 convention.
- `docs/adr/0010-add-pgx-and-goose-to-backend-agent.md` — ADR justifying the postgres deps CH-07 admitted; CH-09 inherits without widening.