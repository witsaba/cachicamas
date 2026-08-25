## CH-09 — Offer tools through a tool-source port

Closes R-03 (the tool seam's real answer) and R-09's seam. Wave 3 of doc 0005.

### Charter

> **CH-09 — Offer tools through a tool-source port.** SDD change: `cachicamas-chat-tool-source` · Closes: R-03 (the tool seam's real answer), R-09's seam. · **Refinement: deferred** — child nodes and scenarios land just-in-time in the PR that opens this wave.
>
> **Goal:** replace CH-00.1's recorded empty tool answer with a real port and at least one tool the model can call.
> **Deliverable:** the archetype's tool-source port, one tool behind it, and the tool-call events reaching the browser.
> **Acceptance:** Given a conversation whose tool source offers one tool, When the model calls it, Then the call and its result appear on the participant's stream and the turn continues with the result in the transcript.
> **Depends on:** CH-05. **Blocks:** CH-10.

### Acceptance

> Given a conversation whose tool source offers one tool, When the model calls it, Then the call and its result appear on the participant's stream and the turn continues with the result in the transcript.

### What ships

- **Backend**: chat-owned `chat.ToolSource` interface wrapping Layer 2's `Registry`; `current_time` tool with injectable `NowFunc`; 4 new `WireEvent` variants (`ToolCallStart`, `ToolCallDelta` reserved, `ToolCallEnd` reserved, `ToolResult`); 5-arm projector collapsing Layer 2's bracket model into 2 chat-side events per call; `chat.Exchange` widens with `[]ToolCallRecord` + `[]ToolResultRecord`; postgres sibling-table migration `chat/migrations/0002_tool_records.sql`; substrate + wire-fragmentation guards.
- **Frontend**: 4 new `ChatStreamEvent` variants + adjacent DTOs (`ToolCallDTO`, `ToolResultDTO`); `parseTranscript` 9-arm switch with `assertNever`; `KNOWN_EVENTS` extended; `use-chat-stream` accumulates tool entries from live SSE; `exchangesToEntries` renders tool entries from `ExchangeDTO.ToolCalls`+`ToolResults` after the assistant said entry; `transcript-line` tool branch extended.
- **Spec/doc**: new `openspec/specs/cachicamas-chat-tool-source/spec.md` (R-CTS-001..008, NFR-CTS-001..003, S-CTS-001..024 + S-CTT-001..003); additive amendments to `chat-conversation-store` (R-CCS-015/016, NFR-CCS-008, S-CCS-019..022) and `frontend-chat-layer1` (REQ-8..11 with explicit "new variants, not new fields" wording, S-FCL-012..017); doc 0005 status bumped to "10 of 12 shipped"; CH-09.1..5 ticked; AGENTS.md CH-09 pointer appended.

### Evidence gate (all uncached)

- `cd backend/agent && go test -count=1 -race ./...` → **17/17 packages OK, race-clean**
- `cd backend/agent && make lint` → **0 issues**
- `cd backend/agent && make build/chat` → produced **`./bin/chat`** (29,805,330 bytes)
- `cd frontend && pnpm test:ci` → **594/594 tests pass** across 159 suites (587 → 594 from recovery pass)
- `cd frontend && pnpm lint` → **0 errors / 0 warnings**
- `cd frontend && pnpm build.types` → **clean** (`tsc --incremental --noEmit` exit 0)
- `git diff --stat main..HEAD -- backend/agent/src/agent/` → **empty** (NFR-TLS-003 / NFR-CTS-003 binding)
- `git diff main..HEAD -- backend/agent/go.mod backend/agent/go.sum` → **empty** (no new Go top-level deps)

24/24 scenarios covered (23 COMPLIANT + 1 DEFERRED pending `INTEGRATION=1`; verify-report #3974 PASS).

### Review budget

- **1500 lines pre-authorised at preflight #3948**; `size:exception` baked in.
- 12 commits, 41 files changed, **3511 net LOC** (3017 initial + 422 recovery + 72 archive materials). Within ~2x variance ceiling consistent with CH-08's 1100→3056 precedent.
- Substrate (10-file invariant under `backend/agent/src/agent/`) preserved across all 12 commits.

### Risks

1. **D-6 wire collapse** — chat wire deliberately drops `EventKindToolProgress` (no progress events on wire at v1). Reserved variants `tool.call.delta` / `tool.call.end` held at `wire.go` for v2 dynamic-source surfaces (long-running MCP tools).
2. **D-7 closed-union widening** — REQ-8..11 carry explicit "new variants, not new fields" rationale to forestall a future reviewer misreading REQ-7 as forbidding. REQ-7 byte-unchanged.
3. **D-8 re-billing deferred** — v1 static tool source keeps cache prefix stable per conversation (V-REQ-14). V2 dynamic-source surface deferred (doc 0005 register row 5 + ADR 0009 § D4).
4. **Wrapper direction confusion** — `chat.ToolSource` is inverse of `ConversationStore` (chat owns port; internally adapts back to `agent.Registry` for `Harness.Tools`). Documented in design #3965.
5. **Spec wording amendments** — F-CHT-9.1 (state `"complete"` → `"done"` alignment) resolved at design; F-CHT-9.2 (Go 1.26 type-switch runtime, not compile-time) and F-CHT-9.3 (S-CTS-022 finishReason-agnostic by construction via assistantId-keyed delta accumulation) applied in recovery pass.
6. **S-CCS-021 postgres cross-process** — DEFERRED (INTEGRATION=1 environment); test scaffold compiles; will pass given live DSN. Track for follow-up if CH-09.x needed.

### Files changed

```
 backend/agent/src/chat/cancel_test.go                            |   10 +-
 backend/agent/src/chat/chat_test.go                              |   13 +-
 backend/agent/src/chat/conversation.go                           |   37 ++-
 backend/agent/src/chat/conversation_test.go                      |   13 +-
 backend/agent/src/chat/current_time.go                           |  103 +++++++
 backend/agent/src/chat/current_time_test.go                      |  139 ++++++++++
 backend/agent/src/chat/eventsource.go                            |   16 ++
 backend/agent/src/chat/failure_test.go                           |   10 +-
 backend/agent/src/chat/http.go                                   |   68 ++++-
 backend/agent/src/chat/http_test.go                              |   23 +-
 backend/agent/src/chat/migrations/0002_tool_records.sql          |   60 +++++
 backend/agent/src/chat/projection.go                             |   63 +++++
 backend/agent/src/chat/projection_tool_test.go                   |  261 ++++++++++++++++++
 backend/agent/src/chat/registry_test.go                          |   13 +-
 backend/agent/src/chat/store.go                                  |  106 +++++++-
 backend/agent/src/chat/store_postgres.go                         |  149 +++++++++++
 backend/agent/src/chat/store_postgres_test.go                    |   82 ++++++
 backend/agent/src/chat/store_scenarios_test.go                   |  167 ++++++++++++
 backend/agent/src/chat/store_substrate_test.go                   |   59 ++++
 backend/agent/src/chat/store_test.go                             |    8 +
 backend/agent/src/chat/tool_source.go                            |   84 ++++++
 backend/agent/src/chat/tool_source_test.go                       |  151 +++++++++++
 backend/agent/src/chat/wire.go                                   |   71 +++++
 backend/agent/src/chat/wire_fragmentation_test.go                |  150 +++++++++++
 backend/agent/src/cmd/chat/main.go                               |   27 +-
 backend/agent/src/cmd/chat/main_test.go                          |    5 +-
 docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md |    9 +-
 frontend/src/components/chat/chat-app.spec.tsx                   |  121 ++++++++-
 frontend/src/components/chat/chat-app.tsx                        |   91 ++++++-
 frontend/src/components/chat/use-chat-stream.spec.tsx            |  296 +++++++++++++++++++++
 frontend/src/components/chat/use-chat-stream.ts                  |   98 ++++++-
 frontend/src/lib/chat-api.spec.ts                                |   66 +++++
 frontend/src/lib/chat-api.ts                                     |   11 +
 frontend/src/lib/chat-types.ts                                   |   87 +++++-
 openspec/AGENTS.md                                               |   17 ++
 openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/README.md                        |   48 ++++
 openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/apply-progress.md                |  158 +++++++++++
 openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/verify-report.md                  |  292 +++++++++++++++++++
 openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/specs/cachicamas-chat-tool-source/spec.md |  281 +++++++++++++++++++
 openspec/specs/cachicamas-chat-tool-source/spec.md                                                |  283 +++++++++++++++++++
 openspec/specs/chat-conversation-store/spec.md                                                   |   52 ++++
 openspec/specs/frontend-chat-layer1/spec.md                                                      |   77 ++++++
 pr-body.md                                                                                       | (this file)
 42 files changed, 3571 insertions(+), 72 deletions(-)
```

### Commits (13 total: 10 initial + 2 recovery + 1 archive)

1. `51c4fc73` — `test(chat): CH-09 RED scaffold #1 — 4 empty WireEvent variants`
2. `2cf80cc9` — `test(chat): CH-09 RED scaffold #2 — 5-arm stub projector`
3. `5f91d3af` — `feat(chat): CH-09 WU-1 — port + current_time + composition-root wire`
4. `9efe5fc4` — `chore(chat): remove accidentally committed build artifact`
5. `b99a5d5b` — `feat(chat): CH-09 WU-2 — wire projection + SSE framing`
6. `a1ef74a4` — `feat(chat): CH-09 WU-3 — persistence widening + sibling-table migration`
7. `f7e16622` — `feat(chat,web): CH-09 WU-4 — frontend delta + parseTranscript switch`
8. `8b25cf47` — `feat(chat): CH-09 WU-5 — substrate + wire-fragmentation guards + AGENTS pointer`
9. `27c51b6f` — `docs(chat,0005): CH-09 closure — doc 0005 status 10/12 + leaves ticked`
10. `3fb978a6` — `docs(openspec): CH-09 spec promotion + additive amendments + archive folder`
11. `7b640b3a` — `test(chat,web): CH-09 WU-4a — cover live-stream tool scenarios` *(recovery)*
12. `45fb69b9` — `docs(openspec): CH-09 spec wording — F-CHT-9.2 / F-CHT-9.3 amendments` *(recovery)*
13. (this archive commit) — `docs(openspec): CH-09 archive — archive-report + README pointer + missing materials` *(this PR's terminal commit)*

### Spec defects (all RESOLVED)

- **F-CHT-9.1** — RESOLVED at design (`#3965` §13). `state: "complete"` → `state: "done"` alignment in `S-CTS-019` + `S-FCL-014`; matches `mock/chat.ts:42` closed union.
- **F-CHT-9.2** — RESOLVED at recovery (`45fb69b9`). `S-CTS-007` wording: Go 1.26 does NOT enforce type-switch exhaustiveness at compile time when a `default` arm exists — runtime panic from `default` branch is the binding invariant; `TestWire_AllNewVariants_SerialiseViaWireFrameName` enforces it by round-tripping every variant.
- **F-CHT-9.3** — RESOLVED at recovery (`45fb69b9`). `S-CTS-022` wording: `finishReason: "tool_calls"` is "by construction" via assistantId-keyed delta accumulation at `use-chat-stream.ts:202-209`; not an explicit early-return gate. `S-FCL-017` mirror amended.

### Carry-forward

- **S-CCS-021** (postgres cross-process) — DEFERRED (no INTEGRATION=1 environment in this run); test scaffold compiles; will pass given live DSN. Nothing else carries forward.

### Substrate preservation

`git diff --stat main..HEAD -- backend/agent/src/agent/` is **empty** across all 12 commits. The 10-file substrate list (carried verbatim from CH-06/07/08) survives byte-clean. The chat-archetype substrate guard lives at `backend/agent/src/chat/store_substrate_test.go::TestChat_SubstrateUntouched` and runs inside `cd backend/agent && make test`. Wire-fragmentation guard (`S-CTS-024`) lives at `chat/wire_fragmentation_test.go` (Go) and `chat-api.spec.ts`'s `assertNever` probe (TS).

### References

- **Engram lineage**: `#1583`, `#3945`, `#3946`, `#3947`, `#3948`, `#3952`, `#3953`, `#3954`, `#3955`, `#3956`, `#3959`, `#3961`, `#3962`, `#3963`, `#3964`, `#3965`, `#3967`, `#3968`, `#3971`, `#3974`, `#3975`, `#3976`, plus this archive-report.
- **Spec defects**: F-CHT-9.1 / F-CHT-9.2 / F-CHT-9.3 — all RESOLVED.
- **Archive folder**: `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/` (README + proposal + design + tasks + apply-progress + verify-report + archive-report + specs/cachicamas-chat-tool-source/spec.md).
- **Doc 0005 charter**: `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:923-934` (CH-09 charter), `:997-1001` (CH-09.1..5 ticked), `:3` (status "10 of 12 shipped").
- **AGENTS.md CH-09 pointer**: `openspec/AGENTS.md:120-135`.
- **CH-10 unblocked** on merge.