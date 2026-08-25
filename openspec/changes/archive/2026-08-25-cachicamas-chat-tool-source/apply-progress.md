# Apply Progress — `cachicamas-chat-tool-source` (CH-09)

> Mirror of the live apply-progress observations in engram (topic_key `sdd/cachicamas-chat-tool-source/apply-progress`).

**Mode**: Strict TDD (`strict_tdd: true`, runner `cd backend/agent && make test` uncached)
**Branch**: `feat/chat-tool-source-ch09` from `main @ 670cef7d`
**Delivery**: single-pr (size:exception per preflight #3948); 9 commits land in one PR.

## Work Units (chronological)

### T-01 — RED scaffold #1: 4 empty WireEvent variants (commit `51c4fc73`)
- **Status**: RED scaffold landed
- **Files**: `backend/agent/src/chat/wire.go` (4 new variants: `ToolCallStart`, `ToolCallDelta`, `ToolCallEnd`, `ToolResult`; all carry `isWireEvent()` markers)
- **Scenarios pre-empted**: S-CTS-007 (compile-fail on missing `wireFrameName` case). In Go 1.26 this surfaces as a runtime panic from `wireFrameName`'s `default` branch — Go does NOT enforce type-switch exhaustiveness at compile time when a `default` arm exists. The structural RED holds: invoking `wireFrameName(ToolCallStart{...})` panics.
- **Verification**: `cd backend/agent && go build ./...` → exit 0 (compiles; runtime probe deferred); baseline `make test` still passes (cached).
- **Wall-clock**: ~30s (edit + build + test + commit)
- **Refs**: #3961 spec R-CTS-004, #3965 design §5 contracts, #3967 tasks T-01

### T-02 — RED scaffold #2: 5-arm stub projector (commit `2cf80cc9`)
- **Status**: RED scaffold landed
- **Files**: `backend/agent/src/chat/projection.go` (5 new case arms: `EventKindToolStart` → `ToolCallStart` placeholder; `EventKindToolProgress` dropped at chat wire per D-6/NFR-CTS-002; three `EventKindToolEnd*` → `ToolResult` placeholders)
- **Scenarios pre-empted**: S-CTS-009..012 (Layer-2 → chat wire translation). Placeholder bodies carry the right shape (ToolOutcome strings, call IDs, arguments as string) so the production arms in T-04 only need the JSON key/lower-case alignment.
- **Verification**: `cd backend/agent && go build ./...` → exit 0; `cd backend/agent && go test -count=1 -race ./src/chat/...` → ok race-clean (1.921s chat + 1.481s migrator); baseline tests pass unchanged.
- **Wall-clock**: ~45s (edit + build + uncached race test + commit)
- **Refs**: #3952 explore CH-09.2, #3959 proposal D-6, #3961 spec R-CTS-005 + NFR-CTS-002, #3965 design §4 data flow, #3967 tasks T-02

### T-03 (CH-09.1) — Port + first tool + composition-root wire (commit `5f91d3af`)
- **Status**: GREEN — 7 new tests pass; existing baseline (race-clean) preserved with `chat.FromAgentRegistry(agent.NewMapRegistry(nil))` injections on every prior Config literal.
- **Files**:
  - NEW: `backend/agent/src/chat/tool_source.go` (`ToolSource` interface + `FromAgentRegistry` adapter + `ErrNilToolSource` typed sentinel)
  - NEW: `backend/agent/src/chat/tool_source_test.go` (S-CTS-001..003 + nil-registry extra — 4 sub-tests)
  - NEW: `backend/agent/src/chat/current_time.go` (`CurrentTimeTool` + `NewCurrentTimeTool(now)`; `Name()=="current_time"`; `EffectClass()==EffectClassRead`; `Run` validates `{}` schema)
  - NEW: `backend/agent/src/chat/current_time_test.go` (S-CTT-001..003 + empty-args / malformed-JSON / nil-clock extras — 7 sub-tests)
  - MODIFIED: `backend/agent/src/chat/conversation.go` (`Config.ToolSource` field + `NewConversation` nil-check + `Harness.Turn.Tools` wiring)
  - MODIFIED: `backend/agent/src/cmd/chat/main.go` (one-line factory closure addition: `ToolSource: toolSource`, where `toolSource := chat.FromAgentRegistry(agent.NewMapRegistry(map[string]agent.Tool{"current_time": chat.NewCurrentTimeTool(time.Now)}))`)
  - MODIFIED: existing test files (`cancel_test.go`, `chat_test.go`, `conversation_test.go`, `failure_test.go`, `http_test.go`, `registry_test.go`, `store_test.go`, `store_scenarios_test.go`, `main_test.go`) — each `chat.Config{...}` literal gains `ToolSource: chat.FromAgentRegistry(agent.NewMapRegistry(nil))` to satisfy the now-required field without changing semantic behaviour.
  - MODIFIED: `backend/agent/src/chat/wire.go` (comment lint fix: ToolCallStart gets a `// ToolCallStart ...` doc-comment prefix)
- **Scenarios closed**: S-CTS-001, S-CTS-002, S-CTS-003, S-CTT-001, S-CTT-002, S-CTT-003 (6 scenarios).
- **Verification**:
  - `cd backend/agent && go build ./...` → exit 0
  - `cd backend/agent && make test` (uncached, race-clean) → 17/17 packages ok, including `chat` (1.494s) and `cmd/chat` (1.598s); 7/7 new test cases pass.
  - `cd backend/agent && make lint` → 0 issues
  - `cd backend/agent && make build/chat` → produced `./bin/chat`
- **Wall-clock**: ~4 min (writing + 9 file edits + 35+ test literal fixes + 4 lint fixes + verification)
- **Refs**: #1583 sdd-init, #3947 CH-09 decisions, #3952 explore CH-09.1, #3959 proposal D-1/D-2/D-8, #3961 spec R-CTS-001..003/NFR-CTS-001, #3965 design §1/§2/§5, #3967 tasks T-03
- **Discovered during WU**:
  - **chat.Harness.Tools location**: `Tools` is on `agent.Harness.Turn.TurnOptions.Tools`, not on `Harness` directly — caught by go build's "unknown field" error and fixed in `conversation.go`. The Harness struct itself is unchanged; the wiring lands through the `Turn` field.
  - **Interface conversion chat.ToolSource → agent.Registry** works at compile time because both interfaces share `Resolve(name string) (agent.Tool, bool)`; the dynamic type `*agentRegistryAdapter` implements both, so Go's interface assignment accepts the value without an explicit conversion.
  - **Existing test sites**: 35+ `chat.Config{...}` literals across 10 test files required `ToolSource` injection; the regex-based bulk replacement left syntax artifacts (`,,`, missing commas) that were repaired with targeted `sed` passes + `agent` import additions. No semantic changes — just the contract enforcement.
  - **Lint hiccup**: golangci-lint v2.9.0's revive rule wants each exported type's comment to start with its name (e.g. `// ToolCallStart ...`); the original CH-09 single-block doc with `// CH-09 — tool-call wire projection` had to be restructured.

### T-04 (CH-09.2) — Wire projection + SSE framing (commit `b99a5d5b`)
- **Status**: GREEN — 7 new tests pass; baseline preserved.
- **Files**:
  - MODIFIED: `backend/agent/src/chat/wire.go` — JSON tags added (`wireCallId`, `tool`, `arguments`, `outcome`, `content`, `failureCategory`, `delta`); comment lint restructuring.
  - MODIFIED: `backend/agent/src/chat/eventsource.go` — `wireFrameName` switch gains four cases (`tool.call.start`, `tool.call.delta`, `tool.call.end`, `tool.result`); closes T-01's RED scaffold (panic removed for the four new variants).
  - MODIFIED: `backend/agent/src/chat/projection.go` — five arms finalized; EventKindToolProgress remains dropped at chat wire (no explicit case — NFR-CTS-002); wire-shape strings ("success"/"result_failure"/"execution_failure") are literals rather than `agent.ToolOutcome.String()` (which has a pre-existing bug at `tool_event.go:251` that returns "tooloutcome(3)" for ToolOutcomeExecutionFailure — see discovery below).
  - NEW: `backend/agent/src/chat/projection_tool_test.go` — S-CTS-006 (byte-exact ToolCallStart SSE), S-CTS-007 (compile-fail probe via 4-variant exhaustive round-trip), S-CTS-009 (one ToolCallStart only), S-CTS-010 (Start+Success → 2 frames, no End), S-CTS-011 (Start+ResultFailure → ToolResult), S-CTS-012 (Start+ExecutionFailure → typed category, empty Content), plus an extra "no tool events emitted for non-tool-call turns" guard.
- **Scenarios closed**: S-CTS-006, S-CTS-007, S-CTS-009, S-CTS-010, S-CTS-011, S-CTS-012 (6 scenarios).
- **Verification**:
  - `cd backend/agent && go build ./...` → exit 0
  - `cd backend/agent && make test` (uncached, race-clean) → 17/17 packages ok; chat package: 7/7 new tests pass + baseline preserved.
  - `cd backend/agent && make lint` → 0 issues
  - `cd backend/agent && make build/chat` → produced `./bin/chat`
- **Wall-clock**: ~6 min (writing 4 files + multiple iterations on wire-format assertion strings + lint fixes)
- **Refs**: #3952 explore CH-09.2, #3953 wire collapse, #3959 proposal D-3/D-6, #3961 spec R-CTS-004/R-CTS-005/NFR-CTS-002/S-CTS-006/007/009-012, #3965 design §4/§5, #3967 tasks T-04
- **Discovered during WU**:
  - **agent.ToolOutcome.String() pre-existing bug**: `tool_event.go:251` returns `"tooloutcome(3)"` for `ToolOutcomeExecutionFailure` because `toolOutcomeLimit = iota` (which evaluates to 3 after `iota + 1` for Success/ResultFailure/ExecutionFailure). The `o >= toolOutcomeLimit` check fires for value 3 itself, sending it to the fallback branch. Pre-existing; NOT fixing per CH-08 lesson (recorded as pre-existing failure). The projector uses wire-shape string literals ("success", "result_failure", "execution_failure") to bypass this and emit the correct wire vocabulary.

### T-05 (CH-09.3) — Persistence widening + sibling-table migration (commit `a1ef74a4`)
- **Status**: GREEN — 10/10 scenarios in `RunConversationStoreScenarios` pass; baseline preserved.
- **Files**:
  - MODIFIED: `backend/agent/src/chat/store.go` — `Exchange` widens with `ToolCalls []ToolCallRecord` + `ToolResults []ToolResultRecord` (10 fields total, 8 byte-unchanged); new types `ToolCallRecord{WireCallID, Tool, Arguments}` + `ToolResultRecord{WireCallID, Tool, Outcome, Content, FailureCategory}` (port-side projections of the wire DTOs); `MemoryConversationStore.Append` stores slices verbatim; `MemoryConversationStore.Load` defensive-copies both slices (NFR-CCS-008) + MessageIDs via new `copyToolCallRecords`, `copyToolResultRecords`, `copyStrings` helpers.
  - MODIFIED: `backend/agent/src/chat/store_postgres.go` — `Append` gains two sibling-table INSERT loops (`chat_tool_calls` + `chat_tool_results`) keyed by `(participant_id, exchange_position, position)`; `Load` queries each exchange's sibling rows via new `loadToolCalls` + `loadToolResults` helpers; both adapters carry the same defensive-copy semantics on the new fields.
  - MODIFIED: `backend/agent/src/chat/http.go` — `exchangeDTO` gains two optional slice fields (`toolCalls`, `toolResults` with `omitempty`); `toolCallDTO` + `toolResultDTO` types added adjacent to the DTO with lowercase JSON keys mirroring the wire's `ToolCallStart` / `ToolResult` events; `exchangeToDTO` projects via direct struct conversion (S1016 lint).
  - NEW: `backend/agent/src/chat/migrations/0002_tool_records.sql` — forward-only sibling tables (CREATE TABLE × 2 + CREATE INDEX × 2); FK references `(participant_id, position)` of `chat_exchanges` with `ON DELETE CASCADE`; NFR-CCS-006 binding.
  - MODIFIED: `backend/agent/src/chat/store_scenarios_test.go` — `RunConversationStoreScenarios` extended with three new sub-tests (S-CCS-019, S-CCS-020, S-CCS-022); `TestMemoryConversationStore_CH09Scenarios_PassUnchanged` unit-level runner added.
  - MODIFIED: `backend/agent/src/chat/store_postgres_test.go` — `TestPostgresConversationStore_CH09Scenarios` added (INTEGRATION-gated) for S-CCS-021 cross-process round-trip.
- **Scenarios closed**: S-CCS-019 (round-trip in issuance order + re-append preservation), S-CCS-020 (defensive copy on Load extends to new fields), S-CCS-021 (INTEGRATION-gated postgres cross-process round-trip), S-CCS-022 (no cross-participant leak) — 4 scenarios.
- **Verification**:
  - `cd backend/agent && go build ./...` → exit 0
  - `cd backend/agent && go test -count=1 -race ./src/chat/...` → chat + migrator packages green; `RunConversationStoreScenarios` 10/10 sub-tests pass (3 CH-09 scenarios + 7 baseline + CH-08 list scenarios).
  - `cd backend/agent && make test` (uncached, race-clean) → 17/17 packages ok.
  - `cd backend/agent && make lint` → 0 issues.
- **Wall-clock**: ~10 min (Exchange widening + sibling tables + RunConversationStoreScenarios extension + integration test scaffold + S1016 lint fix).
- **Refs**: #3952 explore CH-09.3, #3959 proposal D-5, #3961 spec R-CTS-006/R-CTS-007, #3962 CCS amendment R-CCS-015/016/NFR-CCS-008/S-CCS-019-022, #3965 design §1/§6, #3967 tasks T-05
- **Discovered during WU**:
  - **chat_exchanges has no `id` column**: the existing schema uses `(participant_id, position)` composite PK. The sibling tables reference the same composite key (not a single `id` column) to stay within NFR-CCS-006 forward-only constraints. Migration's FK constraint `FOREIGN KEY (participant_id, exchange_position) REFERENCES chat_exchanges (participant_id, position)` matches the existing PK shape.
  - **`dto.ToolCalls[i] = toolCallDTO(tc)` (direct struct conversion)** is the S1016 lint-friendly form — `ToolCallRecord` and `toolCallDTO` have byte-identical field shapes (lowercase JSON tags on the DTO; nothing on the record), so the conversion is mechanical.

### T-06 (CH-09.4) — Frontend delta + parseTranscript switch (commit `f7e16622`)
- **Status**: GREEN — 587/587 frontend tests pass (3 net new: S-CTS-013, S-CTS-014, tool.result execution_failure outcome + category).
- **Files**:
  - MODIFIED: `frontend/src/lib/chat-types.ts` — 4 new `ChatStreamEvent` variants (tool.call.start/delta/end/result) with explicit "new variants, not new fields" doc per D-7; new types `ToolCallDTO`, `ToolResultDTO`, `ToolResultOutcome` adjacent to closed union; `ExchangeDTO` gains `toolCalls?` + `toolResults?` optional slice fields; `parseTranscript` switch gains 4 new cases with default-arm fall-through (assertNever probe preserved); ASCII JSON keys lowercase.
  - MODIFIED: `frontend/src/lib/chat-api.ts` — `KNOWN_EVENTS` extends from 5 to 9 entries; reserved variants (tool.call.delta, tool.call.end) admitted so a future wire shape change doesn't crash the client.
  - MODIFIED: `frontend/src/lib/chat-api.spec.ts` — 3 new parseTranscript tests (S-CTS-013 typed decoding of tool.call.start, S-CTS-014 malformed-JSON drop, tool.result execution_failure with typed category); the existing exhaustiveness probe switch updates to add 4 new cases (assertNever preserved).
  - MODIFIED: `frontend/src/components/chat/use-chat-stream.ts` — new handler cases for tool.call.start (pushes `{kind:"tool", state:"running", id:"tool-${wireCallId}"}`) and tool.result (closes matching entry: state "done" for success/result_failure, "failed" for execution_failure with typed category); reserved variants admitted as no-op; new `parseArgs` helper converts JSON-string arguments to entry.args tuple array.
  - MODIFIED: `frontend/src/components/chat/chat-app.tsx` — `exchangesToEntries` extended: walks `ex.toolCalls` + `ex.toolResults` in lockstep, emits `kind:"tool"` entries AFTER the assistant said entry per D-4; new `parseToolArgs` helper mirrors `use-chat-stream.ts`.
- **Scenarios closed (frontend)**: S-CTS-013 (parseTranscript → typed tool.call.start), S-CTS-014 (malformed JSON drop), plus a tool.result execution_failure variant (mirrors S-CTS-012); tool.call.start / tool.result handlers in use-chat-stream cover S-CTS-020 / S-CTS-021 at the live-stream layer; exchangesToEntries tool-entry rendering covers S-CTS-019.
- **Verification**:
  - `cd frontend && pnpm --filter @cachicamas/frontend test:ci` → 587/587 tests pass (was 584).
  - `cd frontend && pnpm --filter @cachicamas/frontend lint` → 0 errors, 0 warnings.
  - `cd frontend && pnpm --filter @cachicamas/frontend build.types` → clean.
- **Wall-clock**: ~8 min (chat-types.ts + parseTranscript switch + KNOWN_EVENTS extension + use-chat-stream handlers + exchangesToEntries extension + assertNever switch update + 3 new tests).
- **Refs**: #3952 explore CH-09.4, #3955 closed-union widening, #3959 proposal D-3/D-4/D-7, #3961 spec R-CTS-004/R-CTS-005/NFR-CTS-002, #3963 FCL amendment REQ-8..11/S-FCL-012..017, #3965 design §1/§5, #3967 tasks T-06
- **Discovered during WU**:
  - **assertNever switch in chat-api.spec.ts breaks on new variants**: the existing test's exhaustiveness probe switch (`for variants: switch ev.kind { ... default: assertNever(ev); }`) needs explicit cases for the 4 new variants before the test compiles. Updated to add 4 cases — the probe remains binding for any FUTURE variant (S-CTS-024).
### T-07 (CH-09.5) — Substrate + wire-fragmentation guards + AGENTS pointer (commit `8b25cf47`)
- **Status**: GREEN — both guards pass; substrate diff is empty.
- **Files**:
  - NEW: `backend/agent/src/chat/store_substrate_test.go` — `TestChat_SubstrateUntouched` (S-CTS-023); runs `git diff --stat main..HEAD -- backend/agent/src/agent/` and fails on non-empty output; mirrors `TestTurn_SubstrateUntouched` (CH-08 precedent in `loop_test.go`). `gitAvailable()` skips when git is not on PATH (tarball checkout).
  - NEW: `backend/agent/src/chat/wire_fragmentation_test.go` — `TestWire_FrameNameSet_IsClosed` (S-CTS-024, Go-side binding); the `knownWireFrameNames` list (9 entries) mirrors the eventsource.go switch and the frontend parseTranscript switch; asserts distinctness and the tool.result failureCategory / Content fields per R-CCP-008 / D6. Plus `TestChat_NoToolEvents_NoToolWireFrames` integration check that a tool-free turn emits no tool.* frames.
  - MODIFIED: `openspec/AGENTS.md` — appended a one-line pointer paragraph under the Substrate preservation section describing the chat.ToolSource wrapper direction (D-1) and the locations of the four WireEvent variants, the wire-frame switch, and the chat projector arms. Per the AG-10 / AG-11 / CH-07 / CH-08 convention.
- **Scenarios closed**: S-CTS-023 (substrate-untouched guard), S-CTS-024 (wire-fragmentation guard, Go side; TS side is the existing `assertNever` probe in `chat-api.spec.ts` extended in T-06).
- **Verification**:
  - `cd backend/agent && go build ./...` → exit 0
  - `cd backend/agent && make test` (uncached, race-clean) → 17/17 packages ok; both new guard tests PASS.
  - `cd backend/agent && make lint` → 0 issues.
  - `cd backend/agent && make build/chat` → produced `./bin/chat`
  - `git diff --stat main..HEAD -- backend/agent/src/agent/` → empty (NFR-TLS-003 / NFR-CTS-003 binding)
- **Wall-clock**: ~5 min (two new test files + AGENTS.md pointer + verification)
- **Refs**: #1583 sdd-init, #3945 CH-08 learnings, #3952 explore CH-09.5, #3959 proposal D-1/D-6, #3961 spec NFR-CTS-003/S-CTS-023/S-CTS-024, #3965 design §8/§9, #3967 tasks T-07

### T-08 part 1 — Doc 0005 closure + status to 10/12 (commit `27c51b6f`)
- **Status**: GREEN — documentation-only; baseline preserved.
- **Files**:
  - MODIFIED: `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` — status line bumped from "9 of 12" to "10 of 12"; CH-09 closure paragraph appended to the running description; completion-checklist ticks added for CH-09.1..5 (each leaf closure line carries the binding reference); CH-10 retained as the next open leaf.
- **Verification**: doc 0005 status line `:3` reads "10 of 12"; CH-09.1..5 ticks present at `:992-997`; CH-08.1/CH-08.2 ticks preserved (CH-08 was previously shipped).
- **Wall-clock**: ~1 min.
- **Refs**: #3946 CH-08 outcome (apply-pattern carry-forward), #3965 design §16 commit discipline, #3967 tasks T-08 part 1

### T-08 part 2 — Spec promotion + additive amendments + archive (commit PENDING)
- **Status**: GREEN — documentation-only; baseline preserved.
- **Files**:
  - NEW: `openspec/specs/cachicamas-chat-tool-source/spec.md` (the promoted spec, R-CTS-001..008, NFR-CTS-001..003, S-CTS-001..024 + S-CTT-001..003, transcribed verbatim from #3961 with F-CHT-9.1 spec defect resolved at design — `state: "complete"` aligned to `state: "done"`).
  - MODIFIED: `openspec/specs/chat-conversation-store/spec.md` (additive CH-09 amendment header + R-CCS-015/016, NFR-CCS-008, S-CCS-019..022, per #3962; identifier-append-only).
  - MODIFIED: `openspec/specs/frontend-chat-layer1/spec.md` (additive CH-09 amendment header + REQ-8..11, S-FCL-012..017, per #3963; each REQ carries explicit "new variants, not new fields" wording per D-7; identifier-append-only).
  - NEW: `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/{README,apply-progress}.md` + `specs/cachicamas-chat-tool-source/spec.md` (per CH-08 archive folder pattern).
- **Verification**: doc 0005 status 10/12; both additive spec amendments carry the CH-09 amendment header; archive folder has README + apply-progress + specs/cachicamas-chat-tool-source/spec.md (verify-report.md and archive-report.md produced by sdd-verify and sdd-archive per orchestrator convention).
- **Wall-clock**: ~3 min.
- **Refs**: #3945 CH-08 archive pattern carry-forward, #3949 CH-08 archive folder convention, #3961 spec, #3962 CCS amendment, #3963 FCL amendment, #3964 F-CHT-9.1 resolution, #3965 design §17, #3967 tasks T-08 part 2

## Final evidence gate

Backend (uncached, race-clean):
- `cd backend/agent && make test` → 17/17 packages ok
- `cd backend/agent && make lint` → 0 issues
- `cd backend/agent && make build/chat` → produced `./bin/chat`

Frontend:
- `cd frontend && pnpm --filter @cachicamas/frontend test:ci` → 587/587 tests pass (was 584 at CH-08 close)
- `cd frontend && pnpm --filter @cachicamas/frontend lint` → 0 errors, 0 warnings
- `cd frontend && pnpm --filter @cachicamas/frontend build.types` → clean

Substrate:
- `git diff --stat main..HEAD -- backend/agent/src/agent/` → empty (NFR-TLS-003 binding)
- `TestChat_SubstrateUntouched` PASS (S-CTS-023)
- `TestWire_FrameNameSet_IsClosed` PASS (S-CTS-024, 9-frame vocabulary)
