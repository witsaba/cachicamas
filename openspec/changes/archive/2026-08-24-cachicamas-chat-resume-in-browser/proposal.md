# Proposal: Resume a conversation in the browser (CH-08)

## Intent

The chat page (`chat-app.tsx:36`) mounts `useChatStream([])` with an empty seed — a reload or revisit lands on a blank transcript, even when the durable `ConversationStore` has the conversation recorded. CH-08 closes that seam: the page fetches the participant's recorded transcript on mount and re-mounts the conversation rail against the real wire shape, making wave-2 durability visible to the human it was built for.

## Scope

### In Scope

- Backend: additive third method `List(participantID) ([]ConversationSummary, error)` on `ConversationStore` (per R-CCS-010's anticipatory clause).
- Backend: `GET /api/agent/conversations/:id` → `[]ExchangeDTO` (CH-08.1). `GET /api/agent/conversations` → `[]ConversationSummary` (CH-08.2).
- Frontend: `chat-types.ts` adds `ConversationSummary` + `ExchangeDTO` (closed `ChatStreamEvent` union untouched). `chat-api.ts` adds typed `listConversations()` + `loadConversation(id)`. `chat-app.tsx` mounts both GETs via `useVisibleTask$`, seeds `useChatStream.reset(entries)`, re-mounts `ConversationList`. `routes/chat/index.tsx` passes `participantID` from `requireSession`.
- Tests: extend shared `store_scenarios_test.go` helper with list scenarios (R-CCS-012 pattern); per-handler HTTP tests.
- Docs: tick CH-08.1 + CH-08.2 in `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md`; append CH-08 pointer to `openspec/AGENTS.md`.

### Out of Scope

- Resuming the live stream when a tab closes mid-stream (explicit charter "out of scope").
- Branching / renaming / deleting / searching / titles / pagination / filtering.
- `conversation_id` schema migration (many-per-participant model) — deferred.
- SSE replay / `Last-Event-ID`, IndexedDB mirror, offline reads.
- Widening the closed `ChatStreamEvent` union (REQ-7 preserved).

## Capabilities

### New Capabilities

- `cachicamas-chat-resume-in-browser`: covers both new GET endpoints, the additive port widening (`List`), the read-side wire types, and the frontend page-mount delta.

### Modified Capabilities

- `chat-conversation-store`: additive amendment appending `List(participantID)` (R-CCS-013 / R-CCS-014 / S-CCS-015 / S-CCS-016 / S-CCS-017 / S-CCS-018 / NFR-CCS-007). Same declaration extended, not replaced. Identifier append-only per CH-07 precedent.
- `frontend-chat-layer1`: additive amendment appending REQ-8 (page survives a reload) + REQ-9 (page never claims streaming on a reload that catches an in-flight turn). Closed `ChatStreamEvent` union unchanged (REQ-7 enforced).

## Approach

Per the four user-resolved product decisions (#3925): one conversation per participant (`conversationID == participantID`); reload returns `[]ExchangeDTO` only (no metadata envelope); list returns full slice most-recent-first (no pagination/sort); `?with=<slug>` deep-link stays inert (resolves to `agentSlug` but does not drive which conversation loads). Backend implements `List` against `chat_conversations.updated_at DESC` — cheap, no new index at v1. Frontend fires both GETs in parallel on mount; `reset(entries)` seeds the hook; rail re-mounts against the wire. No schema migration, no factory-closure change, no new env vars.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/chat/store.go` | Modified | Adds `List(participantID)` to interface |
| `backend/agent/src/chat/store_postgres.go` | Modified | Implements `List` (`SELECT … FROM chat_conversations ORDER BY updated_at DESC`) |
| `backend/agent/src/chat/store_scenarios_test.go` | Modified | S-CCS-015..017 added (shared helper, R-CCS-012 pattern) |
| `backend/agent/src/chat/store_test.go` + `store_postgres_test.go` | Modified | Per-adapter List micro-tests + `INTEGRATION=1` scenario |
| `backend/agent/src/chat/http.go` | Modified | `HandleReloadConversation`, `HandleListConversations`, two new route registrations |
| `backend/agent/src/chat/http_test.go` | Modified | Per-handler sub-tests (happy + cross-participant + 404) |
| `backend/agent/src/cmd/chat/main.go` | Modified | Passes `chatStore` to new handlers |
| `frontend/src/lib/chat-api.ts` | Modified | New typed `listConversations()`, `loadConversation(id)` |
| `frontend/src/lib/chat-types.ts` | Modified | Adds `ConversationSummary`, `ExchangeDTO`; closed union unchanged |
| `frontend/src/components/chat/chat-app.tsx` | Modified | `useVisibleTask$` mounts both GETs; seeds hook; re-mounts rail |
| `frontend/src/components/chat/conversation-list.tsx` + spec | Modified | Accepts `ConversationSummary[]`; `id = participantID`; relative-time `age` helper |
| `frontend/src/routes/chat/index.tsx` | Modified | Passes `participantID` from `requireSession` |
| `openspec/specs/chat-conversation-store/spec.md` | Modified | Additive amendment (R-CCS-013..017) |
| `openspec/specs/frontend-chat-layer1/spec.md` | Modified | Additive amendment (REQ-8, REQ-9) |
| `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` | Modified | Tick CH-08.1 + CH-08.2 |
| `openspec/AGENTS.md` | Modified | One-line CH-08 substrate pointer |

## Risks

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| Port widening erodes "closed" guarantee | Low | Spec records "third method, additive"; future widens keep the additive pattern |
| Cross-participant reload probes existence | Low | Both endpoints refuse non-owner as 403 not_found (R-CHS-004.b shape) |
| Contributor auto-subscribes on reload → false streaming | Low | REQ-9 forbids; `useChatStream` only opens EventSource on `submit()` |
| `(updated_at DESC)` index missing at scale | Medium | v1 ships without; deferred to future milestone |
| Re-mounted rail reintroduces CH-05 D-3 mock incoherence | Low | Rail is wire-driven; mock surfaces stay isolated in `lib/mock/chat.ts` |

## Rollback Plan

Single PR revert of `feat/chat-resume-ch08`. The additive `List` is a new method on the same declaration — existing `Append`/`Load` untouched. After revert: `cd backend/agent && make test` race-clean (CH-06/CH-07 tests still pass; R-CCS-010 closed-port preserved); `make lint && make build/chat` clean; frontend `pnpm --filter frontend test` green; substrate list in `openspec/AGENTS.md` restored (only the CH-08 pointer line reverts; ten-file substrate unchanged); the two new GET routes disappear and the CH-03 frozen three-route surface is restored.

## Dependencies

- **CH-07** — `PostgresConversationStore` + additive port amendment pattern.
- **CH-05** — wire types (`ChatTurnRequest/Response`, `ChatStreamEvent`); `useChatStream.reset(entries)` API.
- **CH-03** — frozen `/api/agent` surface + `identityMiddleware`.

## Success Criteria

- [ ] Both Gherkin scenarios from doc 0005 lines 859-873 (CH-08.1) and 880-895 (CH-08.2) pass, transcribed verbatim.
- [ ] Substrate list in `openspec/AGENTS.md` unchanged (only the CH-08 pointer line added).
- [ ] CH-07.2 closed-port guard preserved under R-CCS-013 additive widening.
- [ ] `cd backend/agent && make test` race-clean (full suite green).
- [ ] `cd backend/agent && make lint && make build/chat` clean.
- [ ] Frontend `pnpm --filter frontend test` (Vitest) + `pnpm --filter frontend test:e2e` (Playwright) green.
- [ ] Cross-participant reload refused as 403 not_found; unknown conversation as 404 not_found.
- [ ] Empty list returns 200 with `[]` (not 404) per S-CCS-016.
