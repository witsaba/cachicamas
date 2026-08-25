# Tasks: cachicamas-chat-resume-in-browser (CH-08)

> Archived copy of the live tasks observation #3929 in engram (topic_key `sdd/cachicamas-chat-resume-in-browser/tasks`).
> Source: engram `sdd/cachicamas-chat-resume-in-browser/tasks` (`obs-aff438913e6632ef`).
> Charter: CH-08 of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-08--resume-a-conversation-in-the-browser) (`0005:842-899`); leaves CH-08.1 `[leaf]` (`0005:855-876`) and CH-08.2 `[leaf]` (`0005:878-899`).

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1100-1600 |
| 400-line budget risk | High |
| Chained PRs recommended | No (size-exception) |
| Suggested split | Single PR, 10 work-unit commits |
| Delivery strategy | single-pr (exception-ok) |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| WU-1 | RED scaffold `MemoryConversationStore.List` cases | PR 1 | `cd backend/agent && go test -race -count=1 -run TestMemoryConversationStore_List ./src/chat/...` | N/A (unit) | Revert `chat/store_test.go` |
| WU-2 | GREEN `MemoryConversationStore.List` impl | PR 1 | same as WU-1 | N/A | Revert `chat/store.go` |
| WU-3 | RED+GREEN `PostgresConversationStore.List` + INTEGRATION | PR 1 | `cd backend/agent && INTEGRATION=1 go test -race -count=1 -run TestPostgresConversationStore_List ./src/chat/...` | INTEGRATION=1 cross-process | Revert `chat/store_postgres.go` + `chat/store_postgres_test.go` |
| WU-4 | Extend `store_scenarios_test.go` with list scenarios (S-CCS-015..018) | PR 1 | `cd backend/agent && go test -race -count=1 -run TestRunConversationStoreScenarios ./src/chat/...` | N/A | Revert `chat/store_scenarios_test.go` |
| WU-5 | RED HTTP handler sub-tests (200/403/404/empty) | PR 1 | `cd backend/agent && go test -race -count=1 -run 'TestReload\|TestList' ./src/chat/...` | N/A | Revert `chat/http_test.go` |
| WU-6 | GREEN HTTP handlers + `RegisterRoutes` + composition root | PR 1 | `cd backend/agent && make test && make build/chat` | `make build/chat` produces `./bin/chat` | Revert `chat/http.go` + `cmd/chat/main.go` |
| WU-7 | Frontend RED: `listConversations()` + `loadConversation(id)` + DTOs | PR 1 | `cd frontend && pnpm test:ci -- chat-api.spec.ts` | N/A (vitest) | Revert `chat-api.ts` + `chat-types.ts` |
| WU-8 | Frontend GREEN: `useVisibleTask$` mount, `reset(entries)`, rail re-mount | PR 1 | `cd frontend && pnpm test:ci && pnpm test:e2e -- routes/chat` | Vitest | Revert frontend component files |
| WU-9 | Doc 0005 closure + AGENTS.md pointer | PR 1 | `cd backend/agent && make test && make lint` | N/A | Revert `0005-*.md` + `openspec/AGENTS.md` |
| WU-10 | Spec promotion + additive amendments + archive | PR 1 | `cd backend/agent && make test` (regression) | N/A | Revert `openspec/specs/*` + `openspec/changes/archive/...` |

## Phase 1: Backend store widening (RED → GREEN per WU)

- [x] 1.1 WU-1 — RED: add `TestMemoryConversationStore_List` (own-list, empty, cross-participant-absent) in `backend/agent/src/chat/store_test.go`; fails for the right reason. **Commit `4b6f06c1`**
- [x] 1.2 WU-2 — GREEN: implement `MemoryConversationStore.List` in `backend/agent/src/chat/store.go`; iterate `m` map; return `[]ConversationSummary{}` on miss. **Commit `b09a83b7`**
- [x] 1.3 WU-3 — RED+GREEN: implement `PostgresConversationStore.List` in `backend/agent/src/chat/store_postgres.go` (one `SELECT … FROM chat_conversations ORDER BY updated_at DESC`); INTEGRATION=1 scenario in `chat/store_postgres_test.go`. **Commit `94205343`**
- [x] 1.4 WU-4 — extend `backend/agent/src/chat/store_scenarios_test.go` with S-CCS-015..018; unchanged text across both adapters per R-CCS-012. **Commit `7f689eab`**

## Phase 2: HTTP handlers + composition root

- [x] 2.1 WU-5 — RED: add per-handler sub-tests in `chat/http_test.go` (200 happy, 403 cross-participant, 404 unknown, 200 empty list). **Commit `11027e6a`**
- [x] 2.2 WU-6 — GREEN: implement `HandleReloadConversation(store)` + `HandleListConversations(store)` in `chat/http.go`; register both routes; pass `chatStore` in `cmd/chat/main.go`. **Commit `588a15ce`**

## Phase 3: Frontend delta (page-mount + helpers)

- [x] 3.1 WU-7 — RED: `listConversations()` + `loadConversation(id)` typed helpers in `frontend/src/lib/chat-api.ts`; `ConversationSummary` + `ExchangeDTO` in `frontend/src/lib/chat-types.ts` (adjacent to closed `ChatStreamEvent`, never additions). **Commit `926d8682`**
- [x] 3.2 WU-8 — GREEN: `useVisibleTask$` in `chat-app.tsx` fires both GETs in parallel; seeds `useChatStream.reset(entries)`; re-mounts `ConversationList`; pass `participantID` from `requireSession` in `routes/chat/index.tsx`; `conversation-list.tsx` accepts `ConversationSummary[]`. **Commit `f29dee6a`**

## Phase 4: Documentation + spec promotion + archive

- [x] 4.1 WU-9 — Doc 0005 closure: tick CH-08.1, CH-08.2 (`0005:992-993`); bump status to 9 of 12 (`0005:3`); append CH-08 pointer to `openspec/AGENTS.md` (per CH-07 carve-out convention). **Commit `eba2330c`**
- [x] 4.2 WU-10 — promote `openspec/specs/cachicamas-chat-resume-in-browser/spec.md`; additive `## ADDED Requirements` blocks in `chat-conversation-store/spec.md` (R-CCS-013/014, NFR-CCS-007, S-CCS-015..018) and `frontend-chat-layer1/spec.md` (REQ-8/9, S-FCL-008..011); archive under `openspec/changes/archive/2026-08-24-cachicamas-chat-resume-in-browser/`. **Commit `6ff75115`**
