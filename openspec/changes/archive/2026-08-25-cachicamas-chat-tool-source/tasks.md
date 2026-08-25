# Tasks — `cachicamas-chat-tool-source` (CH-09)

> **Mirror of engram observation #3967** (`sdd/cachicamas-chat-tool-source/tasks`).
> Source observations: `#1583` (sdd-init), `#3945` (CH-08 learnings), `#3946` (CH-08 outcome), `#3947` (decisions), `#3948` (preflight), `#3952` (explore), `#3953`–`#3956` (discoveries), `#3959` (proposal), `#3961` (spec), `#3962` (CCS amendment), `#3963` (FCL amendment), `#3964` (F-CHT-9.1 defect), `#3965` (design).
>
> Charter: CH-09 of doc 0005 (`0005:923-934`); leaves CH-09.1 `[leaf]`, CH-09.2 `[leaf]`, CH-09.3 `[leaf]`, CH-09.4 `[leaf]`, CH-09.5 `[guard]`.
>
> Strict TDD: ACTIVE per `#1583`; runner `cd backend/agent && make test` (uncached). Cached runs are NOT evidence.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1500-2500 (CH-08 forecast 1100-1600, actual 3056; +33% inflation expected) |
| 400-line budget risk | High |
| Chained PRs recommended | No (size:exception pre-authorised at preflight #3948) |
| Suggested split | Single PR, ~9 commits (2 RED scaffolds + 5 main WUs + 2 doc/spec) |
| Delivery strategy | single-pr (exception-ok per preflight) |
| Chain strategy | size-exception |

**Decision needed before apply**: No
**Chained PRs recommended**: No
**Chain strategy**: size-exception
**400-line budget risk**: High

Pre-authorisation: `size:exception` baked into preflight #3948 (review budget 1500 lines, up from CH-08's 1000).

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| T-01 | RED scaffold: 4 empty `WireEvent` variants (wireFrameName exhaustive compile-fail) | PR 1 | `cd backend/agent && go build ./...` | N/A (compile probe) | Revert `chat/wire.go` |
| T-02 | RED scaffold: 5-arm stub projector (build passes; no scenario tests yet) | PR 1 | `cd backend/agent && make test` | N/A | Revert `chat/projection.go` |
| T-03 (CH-09.1) | Port + `current_time` tool + composition-root wire | PR 1 | `cd backend/agent && make test` | `make build/chat` produces `./bin/chat` | Revert `chat/tool_source.go` + `chat/current_time.go` + `cmd/chat/main.go` |
| T-04 (CH-09.2) | Wire projection + SSE framing (closes T-01 RED) | PR 1 | `cd backend/agent && make test` | N/A | Revert `chat/wire.go` + `chat/projection.go` + `chat/eventsource.go` |
| T-05 (CH-09.3) | Persistence widening + sibling-table migration | PR 1 | `cd backend/agent && make test` + `INTEGRATION=1 make test` | INTEGRATION=1 postgres cross-process | Revert `chat/store.go` + `chat/store_postgres.go` + `chat/migrations/0002_*.sql` |
| T-06 (CH-09.4) | Frontend delta + parseTranscript switch + tool-entry accumulation | PR 1 | `cd frontend && pnpm test:ci && pnpm lint && pnpm build.types` | N/A (vitest) | Revert frontend files in `chat-types.ts`/`chat-api.ts`/`use-chat-stream.ts`/`chat-app.tsx`/`transcript-line.tsx` |
| T-07 (CH-09.5) | Substrate + wire-fragmentation guards + AGENTS.md pointer | PR 1 | `cd backend/agent && make test` (S-CTS-023 in test suite) | N/A | Revert `chat/store_substrate_test.go` + `chat/wire_fragmentation_test.go` + `openspec/AGENTS.md` |
| T-08 | Doc 0005 closure + spec promotion + additive amendments + archive | PR 1 | `cd backend/agent && make test` (regression) | N/A | Revert `docs/.../0005-*.md` + `openspec/specs/cachicamas-chat-tool-source/spec.md` + `openspec/specs/{chat-conversation-store,frontend-chat-layer1}/spec.md` + archive folder |

## Work Units

#### T-01 — RED scaffold #1: 4 empty `WireEvent` variants

- **Id**: T-01 · **Title**: `test(chat): CH-09 RED scaffold #1 — 4 empty WireEvent variants`
- **Scope**: Declare four new structs (`ToolCallStart`, `ToolCallDelta`, `ToolCallEnd`, `ToolResult`) implementing `isWireEvent()` in `backend/agent/src/chat/wire.go`. Each carries empty fields; this commit does not add `wireFrameName` cases.
- **Strict TDD**: RED only. Build fails naming `wireFrameName`'s `default` branch (T-01 is the FAIL state). No GREEN yet.
- **Scenarios**: none closed; pre-empts S-CTS-007 (a new variant without a `wireFrameName` case is a compile error).
- **Files**: `backend/agent/src/chat/wire.go` (modified — 4 new variant declarations + `isWireEvent` markers).
- **Verification**: `cd backend/agent && go build ./...` → **fails** naming the missing case.
- **Depends on**: none (first commit on `feat/chat-tool-source-ch09`).
- **Out of scope**: production projection logic; composition-root wire; tests.

#### T-02 — RED scaffold #2: 5-arm stub projector

- **Id**: T-02 · **Title**: `test(chat): CH-09 RED scaffold #2 — 5-arm stub projector`
- **Scope**: Extend `chat/projection.go` with five `case agent.EventKindTool{Start,Progress,EndSuccess,EndResultFailure,EndExecutionFailure}:` arms. Each arm produces zero events (placeholder body, e.g. `out <- ToolResult{...}` for the result arms, no-op for `ToolProgress`). Builds; no scenarios cover tool variants yet.
- **Strict TDD**: RED in the sense of "no scenario tests yet"; `make test` runs but the new code paths are unexercised. Wire shape half-built.
- **Scenarios**: pre-empts S-CTS-009..012 (Layer-2 → chat wire translation). Closed in T-04.
- **Files**: `backend/agent/src/chat/projection.go` (modified — 5 new `case` arms; default arm unchanged).
- **Verification**: `cd backend/agent && go build ./...` passes; `cd backend/agent && make test` (uncached) passes unchanged baseline.
- **Depends on**: T-01 (variants declared).
- **Out of scope**: SSE framing changes; production tests; frontend; persistence.

#### T-03 (CH-09.1) — Port + first tool + composition-root wire

- **Id**: T-03 · **Title**: `feat(chat): CH-09 WU-1 — port + current_time + composition-root wire`
- **Scope**: Add chat-owned `ToolSource` interface + `FromAgentRegistry(agent.Registry) ToolSource` adapter + `ErrNilToolSource` typed sentinel. Add `chat.CurrentTimeTool` implementing `agent.Tool` with injectable `NewCurrentTimeTool(now func() time.Time)`. `chat.Config` gains `ToolSource ToolSource` field; `NewConversation` rejects nil. `cmd/chat/main.go` factory closure gains exactly one `ToolSource:` line: `ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(map[string]agent.Tool{"current_time": chat.NewCurrentTimeTool(time.Now)}))` — closure body otherwise byte-unchanged from CH-08.
- **Strict TDD**: RED scaffold #1 closes here (T-02's projector stub isn't touched; this WU adds port+tool files; the RED fails for the right reason at T-01's wireFrameName exhaustiveness probe and clears when T-04 finalizes). New tests first: `tool_source_test.go` and `current_time_test.go` start RED.
- **Scenarios**: **S-CTS-001** (`Config{ToolSource: nil}` → `ErrNilToolSource`, never panic); **S-CTS-002** (`FromAgentRegistry(nil)` → `(nil, false)`); **S-CTS-003** (`FromAgentRegistry(reg)` delegates byte-equal); **S-CTT-001** (`CurrentTimeTool.Run(ctx, []byte("{}"), nil)` → RFC3339 content + success); **S-CTT-002** (non-empty args → `ToolOutcomeResultFailure`, never silently ignored); **S-CTT-003** (`EffectClass() == EffectClassRead`; `Name() == "current_time"`).
- **Files**: NEW `backend/agent/src/chat/tool_source.go`, `backend/agent/src/chat/tool_source_test.go`, `backend/agent/src/chat/current_time.go`, `backend/agent/src/chat/current_time_test.go`. MODIFIED `backend/agent/src/chat/conversation.go` (`Config.ToolSource`; nil check in `NewConversation`), `backend/agent/src/cmd/chat/main.go` (one `ToolSource:` line in factory closure).
- **Verification**: `cd backend/agent && go build ./...` → ok; `cd backend/agent && make test` (uncached) → race-clean with 6/6 new tests GREEN; `cd backend/agent && make lint` → 0 issues; `cd backend/agent && make build/chat` → produces `./bin/chat`.
- **Depends on**: T-02 (projector stub doesn't break wireFrameName exhaustiveness).
- **Threats → RED tests**: "Args JSON validation in `current_time`" → S-CTT-002 (typed refusal on non-empty args).
- **Out of scope**: wire projection (T-04); persistence (T-05); frontend (T-06); substrate guard (T-07).

#### T-04 (CH-09.2) — Wire projection + SSE framing

- **Id**: T-04 · **Title**: `feat(chat): CH-09 WU-2 — wire projection + SSE framing`
- **Scope**: Finalize four new `WireEvent` variants in `chat/wire.go` (full field set per `R-CTS-004`). Replace T-02's stub projector arms with production projection: `EventKindToolStart` → `ToolCallStart`; `EventKindToolProgress` → **dropped at the chat wire** (no explicit `case EventKindToolProgress:`; falls through `default` arm per `NFR-CTS-002`); `EventKindToolEnd{Success,ResultFailure,ExecutionFailure}` → `ToolResult{Outcome, Content, FailureCategory}`. `chat/eventsource.go`'s `wireFrameName` switch gains four cases (`tool.call.start`, `tool.call.delta`, `tool.call.end`, `tool.result`). Closes T-01's RED scaffold (build passes only after the four cases land).
- **Strict TDD**: RED scaffold closes; new tests in `chat/projection_tool_test.go` start RED (write S-CTS-006/007/009/010/011/012 against the not-yet-finalized bodies; production arms finalize GREEN).
- **Scenarios**: **S-CTS-006** (`ToolCallStart` serializes to `event: tool.call.start\ndata: {"wireCallId":"c1","tool":"current_time","arguments":"{}"}\n\n` exact bytes); **S-CTS-007** (a 5th variant without a `wireFrameName` case is a compile error); **S-CTS-009** (`[ToolStart]` → exactly one `ToolCallStart`); **S-CTS-010** (`[ToolStart, ToolEndSuccess]` → `ToolCallStart` + `ToolResult{Outcome:"success", Content:...}`, no `ToolCallEnd`); **S-CTS-011** (`[ToolStart, ToolEndResultFailure]` → `ToolResult{Outcome:"result_failure", Content:"bad args"}`); **S-CTS-012** (`[ToolStart, ToolEndExecutionFailure]` → `ToolResult{Outcome:"execution_failure", FailureCategory:"invalid_argument"}`, no provider text — R-CCP-008 mirror).
- **F-CHT-9.1 alignment** (RESOLVED at design, applied at T-04 via spec amendment in T-08): the spec wording `S-CTS-019` uses `state: "complete"`; design resolution (`#3965` §13) aligns spec text from `"complete"` to `"done"` to match the existing closed `state` union at `frontend/src/lib/mock/chat.ts:42` and `transcript-line.tsx:120-161`. T-08 amends the spec; T-06 frontend test asserts `"done"` verbatim per the implementation.
- **Files**: MODIFIED `backend/agent/src/chat/wire.go` (4 variants finalized), `backend/agent/src/chat/projection.go` (5 arms finalized; `EventKindToolProgress` falls through `default`), `backend/agent/src/chat/eventsource.go` (4 new `wireFrameName` cases). NEW `backend/agent/src/chat/projection_tool_test.go` (S-CTS-006/007/009/010/011/012).
- **Verification**: `cd backend/agent && make test` (uncached) → race-clean with 6/6 new tests GREEN; `cd backend/agent && make lint` → 0 issues.
- **Depends on**: T-03 (port wired; `NewConversation` accepts `ToolSource`; Harness passes `Tools` to Layer 2).
- **Threats → RED tests**: "Provider text leaks on failure" → S-CTS-012 (no provider text in `Content`; only typed category); "Tool arguments `[]byte` ↔ `string` mismatch" → S-CTS-009/010 (byte-equal projection); "Wire-shape mismatch across the bridge" → S-CTS-007 (compile-fail on missing `wireFrameName` case).
- **Out of scope**: persistence (T-05); frontend (T-06); substrate guard (T-07).

#### T-05 (CH-09.3) — Persistence widening

- **Id**: T-05 · **Title**: `feat(chat): CH-09 WU-3 — persistence widening + sibling-table migration`
- **Scope**: Widen `chat.Exchange` (`store.go:41-50`) additively with `ToolCalls []ToolCallRecord` and `ToolResults []ToolResultRecord`. New types `ToolCallRecord{WireCallID, Tool, Arguments}` and `ToolResultRecord{WireCallID, Tool, Outcome, Content, FailureCategory}` (port-side projections of wire DTOs). `MemoryConversationStore.Append`/`Load` round-trip the new fields; defensive copy on `Load` (NFR-CCS-008 carries NFR-CCS-004 forward). Postgres migration `chat/migrations/0002_tool_records.sql` creates sibling tables `chat_tool_calls` + `chat_tool_results` keyed by `chat_exchanges.id` (FK ON DELETE CASCADE, no ALTER of pre-existing tables per NFR-CCS-006). `PostgresConversationStore.Append`/`Load` extend with the sibling-table INSERT/SELECT. `RunConversationStoreScenarios` (`store_scenarios_test.go`) extends with S-CCS-019/020/022 (run against BOTH adapters, scenario text unchanged per R-CCS-012). `store_postgres_test.go` adds S-CCS-021 gated `INTEGRATION=1`.
- **Strict TDD**: RED scaffold for the migration apply (`go test ./chat/...` fails to apply the new migration without the new store methods; design §16 precedent at CH-08 WU-3). New tests first: `store_tool_roundtrip_test.go` starts RED (in-memory cases), postgres INTEGRATION cases start RED.
- **Scenarios**: **S-CCS-019** (turn with two tool calls round-trips through store in issuance order; re-append preserves); **S-CCS-020** (defensive copy on `Load` extends to new fields — caller-side mutation does not corrupt the store); **S-CCS-021** (INTEGRATION=1, postgres adapter round-trip across processes); **S-CCS-022** (tool records never leak across participants — `Load("p2")` sees no records from `p1`).
- **Files**: NEW `backend/agent/src/chat/migrations/0002_tool_records.sql` (sibling tables), `backend/agent/src/chat/store_tool_roundtrip_test.go` (extends `RunConversationStoreScenarios`). MODIFIED `backend/agent/src/chat/store.go` (`Exchange` widening + Append/Load round-trip + defensive copy), `backend/agent/src/chat/store_postgres.go` (sibling-table INSERT/SELECT + defensive copy), `backend/agent/src/chat/store_postgres_test.go` (S-CCS-021 INTEGRATION gated), `backend/agent/src/chat/store_scenarios_test.go` (S-CCS-019/020/022 extension).
- **Verification**: `cd backend/agent && make test` (uncached) → race-clean with 3/3 unconditional new tests GREEN; `cd backend/agent && INTEGRATION=1 make test` → S-CCS-021 GREEN; `cd backend/agent && make lint` → 0 issues; `cd backend/agent && make build/chat` → produces `./bin/chat`.
- **Depends on**: T-04 (wire shape final; `ToolCallRecord`/`ToolResultRecord` projections stable).
- **Threats → RED tests**: "Cross-participant tool records" → S-CCS-022.
- **Out of scope**: frontend (T-06); substrate guard (T-07); doc promotion (T-08).

#### T-06 (CH-09.4) — Frontend delta

- **Id**: T-06 · **Title**: `feat(chat,web): CH-09 WU-4 — frontend delta + parseTranscript switch + tool-entry accumulation`
- **Scope**: `frontend/src/lib/chat-types.ts` gains `ToolCallDTO` + `ToolResultDTO` adjacent to the closed union (mirroring `ExchangeDTO`/`ConversationSummary` placement at `:152-161`); four new `ChatStreamEvent` variants (`tool.call.start`, `tool.call.delta` reserved, `tool.call.end` reserved, `tool.result`) per D-3 + D-7; `parseTranscript` switch extends to 9 arms with `assertNever` probe in default. `frontend/src/lib/chat-api.ts:285-291` `KNOWN_EVENTS` extends from 5 to 9 entries. `frontend/src/components/chat/use-chat-stream.ts` accumulates tool entries: `tool.call.start` opens `{kind:"tool", state:"running", tool, args: parseArgs(arguments), id: unique}`; `tool.result` closes entry to `state: "done"` (success / result_failure) or `"failed"` (execution_failure). `frontend/src/components/chat/chat-app.tsx:66-87` `exchangesToEntries` extends to render tool entries from `ExchangeDTO.ToolCalls` + `ExchangeDTO.ToolResults` AFTER the assistant said entry. `frontend/src/components/chat/transcript-line.tsx` `failed` state path extended if not already covered at `:141-149`. `frontend/src/lib/mock/chat.ts:42` already declares the closed state union (`"running" | "done" | "denied" | "failed"`) — no change required (F-CHT-9.1).
- **Strict TDD**: RED scaffold (`frontend/src/lib/chat-types.spec.ts` or internal assertions; runtime-error RED, per CH-08 WU-7 precedent). New tests first: parseTranscript switch arms + `exchangesToEntries` extension.
- **Scenarios**: **S-CTS-013** (`parseTranscript` parses `tool.call.start` into typed variant); **S-CTS-014** (malformed JSON silently dropped); **S-CTS-019** (`exchangesToEntries` renders tool entries from `ExchangeDTO` — `state: "done"` per F-CHT-9.1 resolution); **S-CTS-020** (live `tool.call.start` SSE frame opens `"running"` tool entry); **S-CTS-021** (matching `tool.result` with `outcome: "execution_failure"` closes entry as `"failed"`); **S-CTS-022** (`finishReason: "tool_calls"` allows assistant bubble to continue streaming).
- **Files**: MODIFIED `frontend/src/lib/chat-types.ts` (DTOs + 4 variants + parseTranscript switch + `assertNever`), `frontend/src/lib/chat-api.ts` (`KNOWN_EVENTS` extended), `frontend/src/components/chat/use-chat-stream.ts` (tool-entry accumulation), `frontend/src/components/chat/chat-app.tsx` (`exchangesToEntries` extension), `frontend/src/components/chat/transcript-line.tsx` (failed state if needed). NEW/MODIFIED specs: `frontend/src/lib/chat-types.spec.ts`, `frontend/src/lib/chat-api.spec.ts`, `frontend/src/components/chat/use-chat-stream.spec.tsx`, `frontend/src/components/chat/chat-app.spec.tsx`.
- **Verification**: `cd frontend && pnpm --filter @cachicamas/frontend test:ci` (vitest) → 584+ baseline + new scenarios GREEN; `cd frontend && pnpm --filter @cachicamas/frontend lint` → 0 errors / 0 warnings; `cd frontend && pnpm --filter @cachicamas/frontend build.types` (tsc) → clean.
- **Depends on**: T-05 (Exchange carries ToolCalls/ToolResults; reload DTOs available).
- **Threats → RED tests**: "Wire-shape mismatch across the bridge" → S-CTS-024 (covered in T-07; `parseTranscript` covers all 9 event names; missing case = TypeScript compile error).
- **Out of scope**: substrate guard (T-07); doc promotion (T-08); chat-side changes.

#### T-07 (CH-09.5) — Substrate + wire-fragmentation guards

- **Id**: T-07 · **Title**: `feat(chat): CH-09 WU-5 — substrate + wire-fragmentation guards + AGENTS pointer`
- **Scope**: Add `chat/store_substrate_test.go` asserting `git diff --stat main..HEAD -- backend/agent/src/agent/` is empty (analogous to `TestTurn_SubstrateUntouched` in `loop_test.go`). Add `chat/wire_fragmentation_test.go` (or extend `frontend/src/lib/chat-types.spec.ts`) asserting `parseTranscript` switch covers all 9 known event names (`message.start`, `message.delta`, `message.end`, `turn.end`, `error`, `tool.call.start`, `tool.call.delta`, `tool.call.end`, `tool.result`). Append CH-09 pointer line to `openspec/AGENTS.md` substrate section per CH-07/08 convention.
- **Strict TDD**: RED scaffold per design §16 (CH-08 WU-5 pattern). Substrate test runs `git diff --stat main..HEAD -- backend/agent/src/agent/` and FAILS on non-empty output (pre-merge hook enforcement); wire-fragmentation test enumerates KNOWN_EVENTS + parseTranscript arms and asserts lockstep.
- **Scenarios**: **S-CTS-023** (substrate-untouched guard — `git diff` empty); **S-CTS-024** (wire-fragmentation guard — parseTranscript switch covers all 9 event names).
- **Files**: NEW `backend/agent/src/chat/store_substrate_test.go`, `backend/agent/src/chat/wire_fragmentation_test.go` (or equivalent frontend assertion). MODIFIED `openspec/AGENTS.md` (one-line CH-09 pointer appended to substrate section).
- **Verification**: `cd backend/agent && make test` (uncached) → race-clean with S-CTS-023 GREEN; `cd frontend && pnpm test:ci` → S-CTS-024 GREEN; `cd backend/agent && git diff --stat main..HEAD -- backend/agent/src/agent/` → empty.
- **Depends on**: T-06 (frontend parseTranscript switch finalized; substrate test runs against the merged change).
- **Threats → RED tests**: "Substrate drift" → S-CTS-023.
- **Out of scope**: doc 0005 status bump (T-08); spec promotion (T-08); archive folder (T-08).

#### T-08 — Doc promotion + spec amendments + archive

- **Id**: T-08 · **Title**: `docs(chat,0005): CH-09 — doc promotion + spec amendments + archive`
- **Scope**: NEW `openspec/specs/cachicamas-chat-tool-source/spec.md` (R-CTS-001..008, NFR-CTS-001..003, S-CTS-001..024 + S-CTT-001..003, transcribed verbatim from `#3961`). Additive amendment to `openspec/specs/chat-conversation-store/spec.md` (R-CCS-015/016, NFR-CCS-008, S-CCS-019..022) per `#3962`. Additive amendment to `openspec/specs/frontend-chat-layer1/spec.md` (REQ-8..11, S-FCL-012..017, each REQ carries explicit "new variants, not new fields" wording per D-7) per `#3963`. F-CHT-9.1 alignment: change `state: "complete"` → `state: "done"` in `S-CTS-019` (in the new spec) and `S-FCL-014` (in the FCL amendment) — single-line spec amendment per `#3965` §13 resolution. `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` status bumped to "10 of 12 shipped"; CH-09.1..5 ticked at `:992-993`. CH-10 unblocked. `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/` archive folder created (per CH-08 archive pattern: README + proposal + design + tasks + apply-progress + verify-report + archive-report + specs/cachicamas-chat-tool-source/spec.md).
- **Strict TDD**: N/A (documentation/spec work; evidence gate is regression on prior WUs).
- **Scenarios**: none — covers spec/doc promotion only.
- **Files**: NEW `openspec/specs/cachicamas-chat-tool-source/spec.md`, `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/{README,proposal,design,tasks,apply-progress,verify-report,archive-report}.md` + `specs/cachicamas-chat-tool-source/spec.md`. MODIFIED `openspec/specs/chat-conversation-store/spec.md` (additive header + R-CCS-015/016 + NFR-CCS-008 + S-CCS-019..022), `openspec/specs/frontend-chat-layer1/spec.md` (additive header + REQ-8..11 + S-FCL-012..017 + F-CHT-9.1 line), `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` (status bump + CH-09.1..5 ticked).
- **Verification**: `cd backend/agent && make test` (regression — all prior WUs GREEN); `cd frontend && pnpm test:ci` (regression — all prior WUs GREEN); doc 0005 status to "10 of 12 shipped" confirmed at `:3`; CH-09.1..5 ticked confirmed at `:992-993`.
- **Depends on**: T-07 (substrate guard closes; final commit shape).
- **Out of scope**: implementation work (closed in T-01..T-07); CH-10 (next milestone, unblocked on archive closure).

### Commit discipline (CH-08 pattern carry-forward, per design §16)

1. `test(chat): CH-09 RED scaffold #1 — 4 empty WireEvent variants` (T-01)
2. `test(chat): CH-09 RED scaffold #2 — 5-arm stub projector` (T-02)
3. `feat(chat): CH-09 WU-1 — port + current_time + composition-root wire` (T-03 GREEN)
4. `feat(chat): CH-09 WU-2 — wire projection + SSE framing` (T-04 GREEN; closes T-01 RED)
5. `feat(chat): CH-09 WU-3 — persistence widening + sibling-table migration` (T-05 GREEN)
6. `feat(chat,web): CH-09 WU-4 — frontend delta + parseTranscript switch` (T-06 GREEN)
7. `feat(chat): CH-09 WU-5 — substrate + wire-fragmentation guards` + AGENTS.md pointer (T-07 GREEN)
8. `docs(chat,0005): CH-09 — doc 0005 closure + status to 10/12` (T-08 part 1)
9. `docs(openspec): CH-09 spec promotion + additive amendments + archive folder` (T-08 part 2)

Conventional commits only — no `Co-Authored-By`, no AI attribution (per `openspec/AGENTS.md` rule 4).

## Scenarios coverage matrix

| Scenario | WU | Files | Strict TDD phase |
|----------|-----|-------|------------------|
| S-CTS-001 | T-03 | `tool_source.go`+`conversation.go`+`tool_source_test.go` | GREEN |
| S-CTS-002 | T-03 | `tool_source.go`+`tool_source_test.go` | GREEN |
| S-CTS-003 | T-03 | `tool_source.go`+`tool_source_test.go` | GREEN |
| S-CTS-006 | T-04 | `wire.go`+`eventsource.go`+`projection_tool_test.go` | GREEN |
| S-CTS-007 | T-04 | `wire.go`+`eventsource.go`+`projection_tool_test.go` | GREEN (compile-fail probe) |
| S-CTS-009 | T-04 | `projection.go`+`projection_tool_test.go` | GREEN |
| S-CTS-010 | T-04 | `projection.go`+`projection_tool_test.go` | GREEN |
| S-CTS-011 | T-04 | `projection.go`+`projection_tool_test.go` | GREEN |
| S-CTS-012 | T-04 | `projection.go`+`projection_tool_test.go` | GREEN |
| S-CTS-013 | T-06 | `chat-types.ts`+`chat-types.spec.ts` | GREEN |
| S-CTS-014 | T-06 | `chat-types.ts`+`chat-types.spec.ts` | GREEN |
| S-CTS-019 | T-06 | `chat-app.tsx`+`chat-app.spec.tsx` (state `"done"` per F-CHT-9.1) | GREEN |
| S-CTS-020 | T-06 | `use-chat-stream.ts`+`use-chat-stream.spec.tsx` | GREEN |
| S-CTS-021 | T-06 | `use-chat-stream.ts`+`use-chat-stream.spec.tsx` | GREEN |
| S-CTS-022 | T-06 | `use-chat-stream.ts`+`use-chat-stream.spec.tsx` | GREEN |
| S-CTS-023 | T-07 | `chat/store_substrate_test.go` | GREEN |
| S-CTS-024 | T-07 | `chat/wire_fragmentation_test.go` (or `chat-types.spec.ts`) | GREEN |
| S-CTT-001 | T-03 | `current_time.go`+`current_time_test.go` | GREEN |
| S-CTT-002 | T-03 | `current_time.go`+`current_time_test.go` | GREEN |
| S-CTT-003 | T-03 | `current_time.go`+`current_time_test.go` | GREEN |
| S-CCS-019 | T-05 | `store.go`+`store_scenarios_test.go`+`store_tool_roundtrip_test.go` | GREEN (both adapters) |
| S-CCS-020 | T-05 | `store.go`+`store_scenarios_test.go`+`store_tool_roundtrip_test.go` | GREEN (both adapters) |
| S-CCS-021 | T-05 | `store_postgres.go`+`store_postgres_test.go` (INTEGRATION=1) | GREEN (gated) |
| S-CCS-022 | T-05 | `store.go`+`store_scenarios_test.go`+`store_tool_roundtrip_test.go` | GREEN (both adapters) |

Coverage: 24 scenarios across 8 WUs; 0 orphans; 0 duplicates. S-FCL-012..017 mirrors CTS-013/014/019..022 (spec amendment header at `#3963`); no separate WU needed (covered by T-08 spec amendment + T-06 implementation).

## Dependencies graph

```
T-01 (RED scaffold #1: 4 empty WireEvent variants)
  └── T-02 (RED scaffold #2: 5-arm stub projector)
        └── T-03 (CH-09.1 — port + current_time + composition-root wire)
              └── T-04 (CH-09.2 — wire projection + SSE framing; closes T-01 RED)
                    └── T-05 (CH-09.3 — persistence widening + sibling-table migration)
                          └── T-06 (CH-09.4 — frontend delta + parseTranscript switch)
                                └── T-07 (CH-09.5 — substrate + wire-fragmentation guards + AGENTS pointer)
                                      └── T-08 (Doc promotion + spec amendments + archive)
```

Linear chain per design §10. Each WU unblocks the next; no parallelizable work (every WU's tests depend on the prior WU's production shape).

## PR body outline (CH-08 pattern carry-forward)

**Title**: `feat(chat): CH-09 — tool-source port + current_time tool + wire projection + persistence widening`

**Body sections** (per `#3945` lesson 3 — chat milestone PR convention):

- **Charter & acceptance** — opens with `[CH-09 of doc 0005]` (`0005:923-934`); acceptance verbatim: *Given a conversation whose tool source offers one tool, When the model calls it, Then the call and its result appear on the participant's stream and the turn continues with the result in the transcript.*
- **What ships** — port + `current_time` + wire projection (4 new events) + persistence widening + frontend delta + substrate guards. Single PR, ~9 commits, 1500-2500 LOC.
- **Evidence gate** — `cd backend/agent && make test` (uncached, race-clean), `make lint`, `make build/chat`, `cd frontend && pnpm test:ci` + `pnpm lint` + `pnpm build.types`, INTEGRATION=1 postgres scenario. Substrate diff empty.
- **Review budget** — 1500 lines pre-authorised at preflight #3948; `size:exception` baked in. Variance to actual ~2x consistent with CH-08 (1100 forecast → 3056 actual).
- **Risks** — D-6 wire collapse (`EventKindToolProgress` dropped at chat wire; reserved variants for future tools needing progress); D-7 closed-union widening (REQ-8..11 explicit "new variants, not new fields"); D-8 re-billing deferred (v1 static source; v2 surface registered); wrapper direction confusion (`chat.ToolSource` inverse of `ConversationStore`); substrate drift (S-CTS-023).
- **Rollback** — single PR revert; additive widening of `Exchange` removable by dropping the two new fields; additive widening of `ChatStreamEvent` removable by reverting the 4 variant declarations and the 9-arm switch's 4 new arms; sibling-table migration reverses via mirror-DROP on a CH-07 forward-only migration; if already merged, amend header pattern (CH-00 `F-1`/`F-2`/`F-3` recorded-not-repaired).
- **Files changed** — ~8 new + ~15 modified (see per-WU scope above; full diff in PR).
- **Commits** — 9 (per commit discipline table above; each commit a reviewable unit, conventional commits only).
- **References** — observations `#1583`, `#3945`–`#3948`, `#3952`–`#3956`, `#3959`, `#3961`–`#3965`; doc 0005 `:923-934`; spec files `cachicamas-chat-tool-source/spec.md`, additive amendments to `chat-conversation-store/spec.md` and `frontend-chat-layer1/spec.md`.

## Constraints (binding, from preflight #3948 + AGENTS.md)

- **No new Go top-level deps** — `pgx/v5/stdlib` + `pressly/goose/v3` (admitted by CH-07, ADR 0010) cover the postgres surface.
- **No file under `backend/agent/src/agent/` modified** (NFR-TLS-003 substrate preservation; S-CTS-023 enforces).
- **Spec identifiers append-only** — never renumber.
- **Spec defect F-CHT-9.1 resolved at design** (`#3965` §13); T-08 applies the alignment on disk (`state: "complete"` → `state: "done"` in `S-CTS-019` and `S-FCL-014`).
- **Strict TDD** — every WU has a RED commit first (or a RED scaffold that pre-empts it).
- **Cached `make test` is NOT evidence** — every WU runs uncached.
- **All scenarios from explore `#3952` accounted for** — 24 scenarios, 0 orphans.

## Next phase: `sdd-apply`

`sdd-apply` produces 9 commits on `feat/chat-tool-source-ch09` (from `main @ 670cef7d`). Each WU follows RED → GREEN → refactor under strict TDD. After each WU, persist `apply-progress.md` (mirror of CH-08 apply-progress.md) to engram at topic `sdd/cachicamas-chat-tool-source/apply-progress`. Carry forward CH-08 patterns from `#3945`:

1. **Truncation recovery**: if `sdd-apply` sub-agent return truncates, recover via `git log main..HEAD` + `git diff --stat main..HEAD` from the worktree + persisted `apply-progress` engram. Re-run evidence gate. Continue to `sdd-verify` without re-running apply.
2. **Auto-mode is not "no prompts ever"** — surface real product forks via `question` tool if any resurface during implementation (none expected per proposal #3959).
3. **Cached runs are not evidence** — `make test` always uncached.
4. **Substrate guard runs in test suite** — `S-CTS-023` lives inside `make test`; not a separate harness call.

`apply-progress` merge pattern from CH-07/08: write incremental `apply-progress.md` to the worktree after each WU; final commit merges it into the archive folder with the doc promotion commit.

PR opens after `sdd-verify` confirms evidence gate (sister phase).