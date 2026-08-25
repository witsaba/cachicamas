# Proposal — `cachicamas-chat-tool-source` (CH-09)

> **Mirror of engram observation #3959** (`sdd/cachicamas-chat-tool-source/proposal`).
> **Change**: `cachicamas-chat-tool-source` · **CH-09** (Wave 3) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md) (`0005:923-934`) · **Closes**: R-03 (the tool seam's real answer), R-09's seam · **Depends on**: CH-05 · **Blocks**: CH-10.
>
> **Source artifacts**: explore #3952, decisions #3947, preflight #3948, CH-08 learnings #3945.

## Intent

Replace CH-00.1's recorded empty tool answer (`decision.md` of `cachicamas-chat-vocabulary-and-scope` § 3 row 4, *"Empty through Wave 1 and Wave 2: the registry resolves no name, so every scheduled call yields a typed unresolved-name failure rather than a crash. CH-09 replaces it with a real tool-source port and at least one tool the model can call"*) with: the chat archetype's `ToolSource` port, one tool the model can call, and the tool-call events reaching the browser as transcript entries. CH-09 ships the seam materialization CH-00 deferred; CH-10 attaches the permission policy.

**Charter acceptance verbatim (`0005:931`)**: *Given a conversation whose tool source offers one tool, When the model calls it, Then the call and its result appear on the participant's stream and the turn continues with the result in the transcript.*

## Scope

### In scope

5 leaves + 1 guard (`#3952` § Child Node Design): **CH-09.1** port + adapter + `current_time` tool + composition-root wire; **CH-09.2** 4 new `WireEvent` variants + projector + SSE framing; **CH-09.3** persistence widening; **CH-09.4** frontend delta; **CH-09.5** substrate + wire-fragmentation guards. ~24 scenarios (S-CTS-001..024, S-CCS-019..022). 8 new files + ~15 modified (`#3952` § Affected Areas). New spec `openspec/specs/cachicamas-chat-tool-source/spec.md`; additive amendments to `chat-conversation-store` and `frontend-chat-layer1`.

### Out of scope

MCP tool sources (charter `0005:933`; ADR 0009 § D4 attaches here, deferred per `#3947`); sandboxing (v2 § 6 seam 3, deferred at CH-00.1); coding-archetype tools (`doc 0004`); re-billing question (v2 § 5.1 — v1's static source makes it moot, deferred to a future spec); widening the closed `Registry` interface (chassis is `chat.ToolSource`, Layer 2's `agent.Registry` is `tool.go:267`, untouched); touching the 10-file substrate under `backend/agent/src/agent/` (NFR-TLS-003).

## Resolved Decisions

| ID | Decision | Citation | Rejected alternative |
|----|----------|----------|----------------------|
| D-1 | **Port shape**: chat-owned `chat.ToolSource` interface wrapping Layer 2's `agent.Registry` (`tool.go:267`); chat depends on `agent.Registry` by interface, not by import-of-internals | `#3947`; `chat-package-boundary` R-CPB-001 | Chat directly importing Layer 2's concrete `mapRegistry` — would couple chat to Layer 2 internals and break the substrate promise |
| D-2 | **First tool**: `current_time` returning RFC3339 with injectable `NowFunc time.Time` | `#3947`; `#3952` § CH-09.1 scenarios | Business-meaningful tool (search, lookup) — proves nothing about the seam; time injection is the testability seam |
| D-3 | **Wire shape**: new SSE event names (`tool.call.start`/`delta`/`end`/`tool.result`) + adjacent DTOs (`ToolCallDTO`/`ToolResultDTO`); **new variants on the union, not new fields on existing variants** | `#3947`; `#3955`; `chat-types.ts:8-20` (preamble), `:27-32` (closed union) | New fields on existing wire events — would violate REQ-7's field-addition clause literally |
| D-4 | **Rendering**: separate transcript entry between turns (system-message-like); `TranscriptLine`'s `tool` branch already exists (`transcript-line.tsx:120-161`) | `#3947`; `#3952` § CH-09.4 | Inline rendering into the assistant bubble — would break `parseTranscript` switch and lose the discrete tool-state shape |
| D-5 | **Persistence**: persist both call and result; `Exchange` gains `[]ToolCallRecord` + `[]ToolResultRecord` (port-side projections of DTOs); `chat-conversation-store` spec gets R-CCS-015/016 widening | `#3947`; `#3952` § CH-09.3 | Persist only the result — reload (CH-08.1) would lose the call and the transcript would be unfaithful |
| D-6 | **Wire collapse (deliberate surfacing)**: chat wire collapses Layer 2's 5-event-per-call bracket model (`EventKindToolStart` + `ToolProgress*` + `ToolEnd*` at `event.go:161-176`) into 2 chat-side events per call (`tool.call.start` + `tool.result`); `EventKindToolProgress` is dropped at the chat wire | `#3953`; `#3952` § Risk 5 | Preserve 1:1 mapping — would put free-form `ToolProgress` bytes onto a wire that renders one transcript line per call, breaking the rendering decision |
| D-7 | **REQ-7 widening language (deliberate surfacing)**: `frontend-chat-layer1/spec.md` REQ-7 forbids *new fields on existing variants*; CH-09 widens with *new variants on the union*. Proposal lifts to REQ-8..11 with explicit "new variants, not new fields" wording so a future reviewer cannot misread REQ-7 as forbidding it | `#3955`; `frontend-chat-layer1/spec.md:83-94` REQ-7 | Silently add variants and let the spec text stay ambiguous — would let a future reviewer reject the PR on a misreading |
| D-8 | **Re-billing question deferred (deliberate surfacing)**: v1's `chat.FromAgentRegistry(agent.NewMapRegistry(...))` is constructed once per conversation; the cache prefix is byte-stable per conversation (V-REQ-14 / `ai.ToolSet` deterministic ordering). v2 dynamic-source surface (MCP) registers the question, not solves it | doc 0005:925; v2 § 5.1:624-627; ADR 0009 § D4 | Solve re-billing now — would overbuild v1 against a static source; the answer for v1 is one sentence, not a design |

## Capabilities

### New capabilities

- **`cachicamas-chat-tool-source`** — `CTS` prefix verified collision-free across `openspec/specs/`, `openspec/changes/`, `openspec/changes/archive/`, and `docs/` (no pre-existing `R-CTS-` / `S-CTS-` / `NFR-CTS-` hits). Carries the headline requirements: port + adapter + first tool + wire shape + frontend rendering + persistence widening + substrate guards. Identifier ranges: `R-CTS-001..099`, `S-CTS-001..199`, `NFR-CTS-001..099`.

### Modified capabilities (additive, identifier-append-only per CH-07/CH-08 precedent)

- **`chat-conversation-store`** — additive widening: **R-CCS-015** (Exchange widens with `[]ToolCallRecord` + `[]ToolResultRecord`), **R-CCS-016** (reload replays records in issuance order), **NFR-CCS-008** (defensive copy on Load extends NFR-CCS-004 forward), **S-CCS-019..022**. Append-only; no identifier renumber; existing R-CCS-001..014 + NFR-CCS-001..007 + S-CCS-001..018 byte-unchanged.
- **`frontend-chat-layer1`** — additive widening: **REQ-8** (tool.call.start), **REQ-9** (tool.call.delta), **REQ-10** (tool.call.end), **REQ-11** (tool.result) + **S-FCL-012..017**. Each REQ carries explicit *"new variants on the union, not new fields on existing variants"* wording (D-7). REQ-1..7 byte-unchanged.

## Approach

### Order of work (5 leaves + 1 guard)

Strict TDD per `#1583`: every scenario is one RED → GREEN → refactor cycle.

1. **CH-09.1** (port + tool + wire) — RED scaffolds `chat/tool_source.go` and `chat/current_time.go`; `chat.FromAgentRegistry(agent.NewMapRegistry(map[string]agent.Tool{"current_time": chat.NewCurrentTimeTool(time.Now)}))` is the composition-root line in `cmd/chat/main.go` factory closure (otherwise byte-unchanged from CH-08). GREEN: S-CTS-001..005, S-CTT-001..003.
2. **CH-09.2** (wire projection) — RED scaffold: `wireFrameName` exhaustive compile-fail first (`#3952` Risk 1 + CH-08 WU-1/2 precedent). GREEN: S-CTS-006..012. Bracket collapse documented in D-6.
3. **CH-09.3** (persistence widening) — `chat.Exchange` widening; postgres migration `chat/migrations/0002_tool_records.sql` (sibling tables `chat_tool_calls` + `chat_tool_results`, NOT columns on `chat_exchanges` — per `#3952` Risk 6, NFR-CCS-006 forward-only); both adapters implement via shared `RunConversationStoreScenarios`. GREEN: S-CCS-019..022 (with `S-CCS-021` gated `INTEGRATION=1`).
4. **CH-09.4** (frontend delta) — `chat-types.ts` adds `ToolCallDTO` + `ToolResultDTO` adjacent to closed union; 4 new variants; `parseTranscript` switch extended (assertNever-style exhaustiveness); `KNOWN_EVENTS` extended; `use-chat-stream` accumulates tool entries; `exchangesToEntries` extension renders tool entries from `ExchangeDTO.ToolCalls` + `ExchangeDTO.ToolResults` after the assistant said entry. GREEN: S-CTS-013/014, S-CTS-019..022 (frontend-scenario names).
5. **CH-09.5** (guards) — substrate-untouched test (`S-CTS-023`); wire-fragmentation test (`S-CTS-024`, asserts `parseTranscript` switch covers all 9 event names); AGENTS.md CH-09 pointer appended (CH-07/08 convention).

### Verification surface

- **Backend**: `cd backend/agent && make test` (uncached; race-clean); `make lint`; `make build/chat` produces `./bin/chat`; substrate test `S-CTS-023` passes; postgres integration test (`S-CCS-021`) gated `INTEGRATION=1`.
- **Frontend**: `pnpm --filter @cachicamas/frontend test:ci`; `pnpm lint` 0 errors / 0 warnings; `pnpm build.types` clean; wire-fragmentation test `S-CTS-024` passes.
- **Spec/doc**: `openspec/specs/cachicamas-chat-tool-source/spec.md` new; `chat-conversation-store/spec.md` and `frontend-chat-layer1/spec.md` amended; doc 0005 status bumped to "10 of 12 shipped"; CH-09.1..5 ticked at `:992-993` (CH-08 location); AGENTS.md CH-09 pointer appended.

### Substrate preservation discipline (NFR-TLS-003)

Zero files under `backend/agent/src/agent/` modified. The 10-file substrate list survives byte-clean: `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`. Verified with `git diff --stat main..HEAD -- backend/agent/src/agent/` (must be empty per CH-08 `S-CTS-023` precedent). New code lives under `backend/agent/src/chat/` + `cmd/chat/`. Chat depends on `agent.Registry` by interface (`tool.go:267`), not by import-of-internals — `agent.tool.go` itself is untouched.

## Affected Areas

See `#3952` § Affected Areas for the master list. Verbatim summary (cite the explore for line-level detail):

**New files** (8): `backend/agent/src/chat/tool_source.go`, `tool_source_test.go`, `current_time.go`, `current_time_test.go`, `projection_tool_test.go`, `store_tool_roundtrip_test.go` (extends `RunConversationStoreScenarios`); `openspec/specs/cachicamas-chat-tool-source/spec.md`.

**Modified files** (~15): `backend/agent/src/chat/store.go` (Exchange widening), `store_postgres.go` (sibling tables, columns round-trip test), `chat/migrations/0002_*.sql` (forward-only ALTER/CREATE, NFR-CCS-006), `wire.go` (4 new variants + `isWireEvent`), `projection.go` (5 new arms), `eventsource.go` (`wireFrameName` extended), `conversation.go` (`Config.ToolSource`, nil rejection), `cmd/chat/main.go` (one factory-closure line); `frontend/src/lib/chat-types.ts` (2 DTOs + 4 variants + `parseTranscript`), `chat-types.spec.ts`, `chat-api.ts` (`KNOWN_EVENTS` extended), `chat-api.spec.ts`, `mock/chat.ts` (tool variant fields), `use-chat-stream.ts` (+ spec), `chat-app.tsx` (`exchangesToEntries` extension + spec), `transcript-line.tsx` (`failed` state extension); `openspec/specs/chat-conversation-store/spec.md` (+R-CCS-015/016, +NFR-CCS-008, +S-CCS-019..022), `openspec/specs/frontend-chat-layer1/spec.md` (+REQ-8..11, +S-FCL-012..017); `docs/architecture/milestones/0005-…md` (status bump + ticks); `openspec/AGENTS.md` (CH-09 pointer).

**Untouched** (NFR-TLS-003 substrate): all 10 files under `backend/agent/src/agent/`; `backend/agent/src/agent/tool.go` (Layer 2's `Tool`/`Registry` interfaces); `loop.go` (`TurnOptions.Tools` — chat hands it a `Registry` value through the chat-owned `ToolSource` adapter).

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| **D-6 wire collapse surprises a future tool engineer** — `EventKindToolProgress` is dropped at the chat wire; a future long-running MCP tool needs progress and discovers the wire has no slot | Medium | D-6 surfaces explicitly in `cachicamas-chat-tool-source/spec.md` rationale and in `openspec/AGENTS.md` CH-09 pointer; future tool asks for a 5th chat-side event (`tool.call.delta` is reserved by REQ-9 for that case) |
| **D-7 closed-union widening misread as forbidden by REQ-7** — a future reviewer reads the original REQ-7 field-addition clause and rejects the PR | Medium | REQ-8..11 carry explicit "new variants on the union, not new fields on existing variants" wording; spec amendment header records "additive, identifier-append-only" precedent (CH-07/CH-08) |
| **D-8 re-billing question resurfaces before v2 dynamic-source surface lands** — a future reader wonders whether the cache prefix is invalidated | Low | `cachicamas-chat-tool-source/spec.md` states v1's static-source answer in one sentence and registers the v2 surface as a future spec (ADR 0009 § D4 attachment point) |
| **Wrapper direction confusion** — `chat.ToolSource` is the inverse of `ConversationStore`: chat owns the port, internally adapts back to `agent.Registry` for `Harness.Tools`. Different from `ConversationStore` where adapters live behind the chat port | Low | `#3954` documents the wrapper direction; spec rationale states it; composition-root line at `cmd/chat/main.go` makes the translation visible |
| **Substrate preservation drift** — a contributor adds a `Tool` helper inside `backend/agent/src/agent/tool.go` instead of `chat/current_time.go` | Low | `S-CTS-023` (`git diff --stat main..HEAD -- backend/agent/src/agent/` empty) runs in `make test`; CH-08 substrate test pattern (`loop_test.go`) carried forward |
| **ToolCallStart `[]byte` → `string` args conversion** — `agent.ToolStart.Arguments()` (`tool_event.go:131`) is `[]byte`; chat wire stores `string`. JSON-validity invariant lives at Layer 1's `ai.NewToolCall` (V-REQ-17) | Low | Decoder lives at `chat/projection.go` seam; tests assert byte-equality (S-CTS-009, S-CTS-010) |
| **Postgres migration is forward-only but reaches an existing table** — adding columns to `chat_exchanges` would violate NFR-CCS-006 | Medium | Sibling tables `chat_tool_calls` + `chat_tool_results` referenced by `chat_exchanges.exchange_id` (per `#3952` Risk 6); no DROP/ALTER of pre-existing tables |
| **CH-08 `R-CCS-013/014` precedent violation** — additive widening of `Exchange` is byte-unchanged on existing 8 fields, adds 2 fields | Low | Same `RunConversationStoreScenarios` shared helper pattern (R-CCS-012 / CH-08.2 precedent); both adapters implement; CH-06/07/08 scenarios pass unchanged |

## Rollback Plan

Single PR revert of `feat/chat-tool-source-ch09`. The additive widening of `Exchange` (new fields on the existing struct) is removable by dropping the fields and reverting the migration's sibling-table DDL; the additive widening of `ChatStreamEvent` (new variants on the closed union) is removable by reverting the 4 variant declarations and the `parseTranscript` switch arms. After revert: `cd backend/agent && make test` race-clean (CH-06/07/08 tests still pass; R-CCS-010 + R-CCS-013 closed-port preserved); `make lint && make build/chat` clean; frontend `pnpm --filter @cachicamas/frontend test:ci` green; the CH-09 amendment headers in `chat-conversation-store/spec.md` and `frontend-chat-layer1/spec.md` revert (CH-08 additive-amendment header pattern); the 10-file substrate list is restored (the AGENTS.md CH-09 pointer line reverts; ten-file substrate unchanged). **If already merged**, the amending path is mandatory: a follow-up PR with an additive amendment header naming this proposal, the proposed amendment, and the disposition, mirroring CH-00's `F-1`/`F-2`/`F-3` recorded-not-repaired pattern.

## Dependencies

- **CH-05** (merged) — wire types (`ChatStreamEvent`, `ChatTurnRequest/Response`), `useChatStream.reset(entries)` API.
- **CH-08** (merged) — additive port widening pattern (R-CCS-013/014), shared scenario helper (R-CCS-012), page-mount discipline.
- **No new top-level Go deps** (CH-08 pattern). `pgx/v5/stdlib` + `pressly/goose/v3` already admitted by CH-07.
- **No schema migration outside chat-owned tables**. New sibling tables `chat_tool_calls` + `chat_tool_results` carry the `chat_` prefix (ADR 0009 § D6 + NFR-CCS-005).

## Success Criteria

Traceability matrix (locked decisions + deliberate surfacings → CTS-prefixed requirements):

| Source | Requirement / scenario | Locked decision / surfacing |
|--------|-----------------------|-----------------------------|
| D-1 (port shape) | R-CTS-001 (port + adapter), R-CTS-002 (chat depends on `agent.Registry` by interface), S-CTS-001/002/003 | D-1 |
| D-2 (first tool) | R-CTS-003 (RFC3339 + injectable clock), NFR-CTS-001 (EffectClass = EffectClassRead), S-CTT-001/002/003 | D-2 |
| D-3 (wire shape) | R-CTS-004 (4 new `WireEvent` variants + lowercase JSON), S-CTS-006/007 | D-3 |
| D-4 (rendering) | R-CTS-005 (separate transcript entry), S-CTS-013/014/019/020 | D-4 |
| D-5 (persistence) | R-CTS-006 (Exchange widens with `[]ToolCallRecord` + `[]ToolResultRecord`), R-CTS-007 (defensive copy), S-CTS-019/020/021/022 | D-5 |
| D-6 (wire collapse) | NFR-CTS-002 (chat wire collapses 5 → 2 events per call; `ToolProgress` dropped; 5th event reserved as future extension), R-CTS-004 rationale | D-6 surfacing |
| D-7 (REQ-7 widening) | REQ-8..11 (frontend-chat-layer1), each with explicit "new variants on the union, not new fields on existing variants" wording; S-FCL-012..017 | D-7 surfacing |
| D-8 (re-billing deferred) | R-CTS-008 (v1 source is static; cache prefix is byte-stable per conversation); v2 dynamic-source surface registered as a future spec | D-8 surfacing |
| Substrate preservation | NFR-TLS-003 (10-file substrate untouched); S-CTS-023; AGENTS.md CH-09 pointer | Substrate discipline |

**Acceptance**: `cd backend/agent && make test` (uncached; race-clean), `make lint`, `make build/chat`, `pnpm --filter @cachicamas/frontend test:ci`, `pnpm lint`, `pnpm build.types` all green; `S-CTS-001..024` + `S-CCS-019..022` + `S-FCL-012..017` COMPLIANT; substrate diff empty; doc 0005 status bumped to 10 of 12; CH-10 (permission) unblocked.

## Proposal Question Round (auto mode — recorded, not asked)

Auto mode is `auto` per `#3948`. The orchestrator already resolved the 5 main product forks at `#3947`. CH-09 has no remaining open product forks surfaced by the propose phase:

- **None.** The 5 locked forks cover the headline product decisions (port, tool, wire, render, persist). The 3 deliberate surfacings (D-6, D-7, D-8) are recorded as architectural commitments, not open forks. The CH-08 `R-CCS-013/014` additive widening pattern and the CH-07 sibling-table migration pattern are the binding precedents; the proposal mirrors them. The "no substrate touch" and "no new top-level Go deps" rules are repo invariants, not forks.

If a future reader surfaces a fork the orchestrator missed, it lands in the next round's engram decisions observation; the spec phase will then widen the CTS-prefixed requirement set.

## Artifacts

- **Engram** topic `sdd/cachicamas-chat-tool-source/proposal` — this document.
- **Source observations**: #1583 (sdd-init), #3945 (CH-08 SDD flow learnings), #3946 (CH-08 final outcome), #3947 (CH-09 product decisions), #3948 (CH-09 preflight), #3952 (CH-09 explore report), #3953 (wire collapse discovery), #3954 (`chat.ToolSource` wrapper direction), #3955 (closed-union widening), #3956 (Layer 2 already wired).
- **Source spec files**: `openspec/specs/frontend-chat-layer1/spec.md` (REQ-7), `openspec/specs/chat-conversation-store/spec.md` (R-CCS-010..014), `openspec/specs/chat-archetype-contract/spec.md` (`tool.go:265-266` note).
- **Source code citations**: `backend/agent/src/chat/wire.go:13` (sealed `WireEvent`), `eventsource.go:31-48` (`wireFrameName` exhaustive switch), `projection.go:80-84` (current drop), `conversation.go:108-113` (`agent.Harness{...}`), `cmd/chat/main.go:164-166` (composition root factory closure); `backend/agent/src/agent/tool.go:267` (`Registry` interface), `tool.go:265-266` (`ToolSource` port comment), `event.go:161-176` (5-event tool family), `loop.go:112` (`TurnOptions.Tools`); `frontend/src/lib/chat-types.ts:8-20` (preamble), `:27-32` (closed union), `frontend/src/components/chat/transcript-line.tsx:120-161` (`tool` variant branch).