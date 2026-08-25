# Design: cachicamas-chat-resume-in-browser (CH-08)

> Archived copy of the live design observation #3928 in engram (topic_key `sdd/cachicamas-chat-resume-in-browser/design`).
> Source: engram `sdd/cachicamas-chat-resume-in-browser/design` (`obs-049976540d1eb02a`).
> Charter: CH-08 of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-08--resume-a-conversation-in-the-browser) (`0005:842-899`); leaves CH-08.1 `[leaf]` (`0005:855-876`) and CH-08.2 `[leaf]` (`0005:878-899`).

## Technical Approach

Two additive GET endpoints (`GET /api/agent/conversations/:id`, `GET /api/agent/conversations`) behind the existing `identityMiddleware(resolver)` plus a page-mount delta via Qwik `useVisibleTask$` firing both GETs in parallel. The `ConversationStore` port widens with a third method `List(participantID)` (R-CCS-010 anticipatory text; per decisions #3925, spec R-CCS-013). Backend reads `chat_conversations` parent rows; frontend reuses the existing `useChatStream.reset(entries)` API to seed the buffer and re-mounts `ConversationList` against the wire shape. Maps proposal #3926 + spec #3927 1:1; no schema migration; closed `ChatStreamEvent` union untouched (REQ-7).

## Architecture Decisions

| # | Choice | Alternatives | Rationale |
|---|--------|--------------|-----------|
| 1 | Additive third method `List(participantID) ([]ConversationSummary, error)` on the same `ConversationStore` declaration | (a) Separate `ConversationIndex` port; (b) embed list into `Load` (BREAKING) | R-CCS-010 verbatim anticipates "CH-08.2 widens the port to a list by extending, not replacing"; shared scenario helper (R-CCS-012 precedent) extends with S-CCS-015..018 scenarios that run against both adapters unchanged; `chat_conversations` parent + `updated_at` already encode the query. |
| 2 | Exchanges only — `[]ExchangeDTO` mirroring `chat.Exchange` field-for-field (D-7) | (a) Hydrated `ConversationView` envelope | D-2; minimum surface that closes S-CRI-001/S-CRI-002; future-proofs a header the charter doesn't require; frontend reuses existing Exchange shape. |
| 3 | `conversationID == participantID` (one conversation per participant) | (a) Separate `conversation_id` UUID with FK to participant | D-1; schema (`chat_conversations.participant_id` PK) already encodes this; zero migration; future many-per-participant is a future milestone's schema change. |
| 4 | Full slice, `chat_conversations ORDER BY updated_at DESC`, no pagination | (a) Top-N constant | D-3; charter explicit deferral of ordering/paging/search; v1's per-participant conversation count is bounded. |
| 5 | Qwik `useVisibleTask$` fires both GETs in parallel; `useChatStream.reset(entries)` seeds; `ConversationList` re-mounted; `?with=<slug>` stays inert | (a) IndexedDB mirror | Backend is source of truth; `useChatStream` already accepts `initialEntries`; rail `data-testid`s survive (spec drives them); D-4 keeps `?with=` inert. |
| 6 | `:id` MUST equal caller's `participantID`; 403 `not_found` on mismatch; `ErrConversationNotFound` → 404 `not_found` | (a) Always 404 (leaks existence); (b) always 403 (refuses valid); (c) JSON envelope with codes | R-CHS-004.b precedent (CH-03); refuses without probing existence; matches project's frozen error envelope. |

## Data Flow

```
page (useVisibleTask$ on mount)
  ├─ GET /api/agent/conversations/:id
  │   ├─ identityMiddleware → getIdentity(c).ParticipantID()
  │   │   reject if :id != participantID → 403 not_found
  │   └─ HandleReloadConversation(store)
  │       store.Load(participantID) → []Exchange
  │     → 200 []ExchangeDTO  (or 404 not_found on ErrConversationNotFound)
  └─ GET /api/agent/conversations
      ├─ identityMiddleware → getIdentity(c).ParticipantID()
      └─ HandleListConversations(store)
          store.List(participantID) → []ConversationSummary
        → 200 []ConversationSummaryDTO  (empty slice on miss)
```

Postgres List path (D-1 keeps `WHERE participant_id = $1` returning 0 or 1 row at v1 scale; correlated subquery reads the contiguous position counter from `chat_exchanges`):

```
SELECT participant_id, updated_at,
       (SELECT COALESCE(MAX(position)+1,0)
        FROM chat_exchanges WHERE participant_id = $1) AS turn_count
  FROM chat_conversations
 WHERE participant_id = $1
 ORDER BY updated_at DESC
```

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `backend/agent/src/chat/store.go` | Modify | Add `List(participantID)` to interface; add `ConversationSummary` struct. Append/Load byte-unchanged. |
| `backend/agent/src/chat/store_postgres.go` | Modify | Implement `PostgresConversationStore.List` against `chat_conversations`. |
| `backend/agent/src/chat/http.go` | Modify | Add `HandleReloadConversation(store)` + `HandleListConversations(store)`; register both routes. |
| `backend/agent/src/cmd/chat/main.go` | Modify | Pass `chatStore` to two new handlers. Factory closure byte-unchanged. |
| `backend/agent/src/chat/store_test.go` | Modify | Add `MemoryConversationStore.List` micro-tests. |
| `backend/agent/src/chat/store_scenarios_test.go` | Modify | Extend with S-CCS-015..018 list scenarios (R-CCS-012 pattern, unchanged text across adapters). |
| `backend/agent/src/chat/store_postgres_test.go` | Modify | Add `INTEGRATION=1` List scenario. |
| `backend/agent/src/chat/http_test.go` | Modify | Sub-tests per handler: 200 happy, 403 cross-participant, 404 unknown, 200 empty list. |
| `frontend/src/lib/chat-types.ts` | Modify | Add `ConversationSummary` + `ExchangeDTO` adjacent to (NOT extending) closed `ChatStreamEvent` union. |
| `frontend/src/lib/chat-api.ts` | Modify | Add `listConversations()` + `loadConversation(id)` helpers reusing `envelopeToResult`. |
| `frontend/src/components/chat/chat-app.tsx` | Modify | `useVisibleTask$` fires both GETs; seeds `useChatStream`; re-mounts rail. |
| `frontend/src/components/chat/conversation-list.tsx` | Modify | Accept `ConversationSummary[]`; render `id`/`age`/`turnCount`. |
| `frontend/src/routes/chat/index.tsx` | Modify | Pass `participantID` from `requireSession` into `ChatApp`. |
| `frontend/src/components/chat/conversation-list.spec.tsx` | Modify | Assert wire-driven props. |
| `frontend/src/lib/chat-api.spec.ts` | Modify | Add tests for `listConversations`/`loadConversation`. |
| `frontend/src/routes/chat/index.spec.tsx` | Modify | E2E: page on reload fetches and renders. |
| `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` | Modify | Tick CH-08.1, CH-08.2 (`:992-993`); `:1069`; bump `:3` status to 9 of 12. |
| `openspec/AGENTS.md` | Modify | Append CH-08 pointer to substrate section. |
| `openspec/specs/cachicamas-chat-resume-in-browser/spec.md` | Create (archive) | Promote CH-08 spec from change folder. |
| `openspec/specs/chat-conversation-store/spec.md` | Modify (archive) | Append CH-08 amendment header + `## ADDED Requirements` (R-CCS-013/014, NFR-CCS-007, S-CCS-015..018). |
| `openspec/specs/frontend-chat-layer1/spec.md` | Modify (archive) | Append CH-08 amendment header + `## ADDED Requirements` (REQ-8/9, S-FCL-008..011). |

NFR-TLS-003: ten files under `backend/agent/src/agent/` remain byte-unchanged. New code lives only under `backend/agent/src/chat/` + `cmd/chat/`.

## Interfaces / Contracts

```go
// backend/agent/src/chat/store.go (additive widening — R-CCS-013)
type ConversationStore interface {
    Append(participantID string, exchange Exchange) error
    Load(participantID string) ([]Exchange, error)
    List(participantID string) ([]ConversationSummary, error)  // new
}

type ConversationSummary struct {
    ConversationID string
    LastActivityAt time.Time
    TurnCount      int
}

// backend/agent/src/chat/http.go
func HandleReloadConversation(store ConversationStore) echo.HandlerFunc
func HandleListConversations(store ConversationStore) echo.HandlerFunc
// Both registered via RegisterRoutes under /api/agent/conversations[/:id]
```

```ts
// frontend/src/lib/chat-types.ts (adjacent to ChatStreamEvent, never additions)
export interface ExchangeDTO {
  position: number; promptText: string; assistantText: string;
  partial: boolean; terminalKind: "completed" | "cancelled" | "failed";
  failureCategory: string; finishReason?: string; messageIDs: readonly string[];
}
export interface ConversationSummary {
  conversationID: string; lastActivityAt: string; turnCount: number;
}
```

## Testing Strategy

| Layer | What to Test | Approach |
|-------|--------------|----------|
| Unit (Go) | `MemoryConversationStore.List` iteration order, empty slice | `store_test.go` micro-tests |
| Scenario (Go) | CH-06 + CH-08.2 scenarios against both adapters, unchanged text | `store_scenarios_test.go` helper (R-CCS-012) |
| Integration (Go) | `PostgresConversationStore.List` cross-process | `store_postgres_test.go` INTEGRATION=1 |
| HTTP (Go) | 200 happy, 403 cross-participant, 404 unknown, 200 empty list | `http_test.go` sub-tests |
| Unit (TS) | `listConversations`/`loadConversation` helpers | vitest `chat-api.spec.ts` |
| Component (TS) | `ConversationList` renders `ConversationSummary[]` | vitest `conversation-list.spec.tsx` |
| E2E (TS) | Page on reload fetches + renders | vitest `routes/chat/index.spec.tsx` |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. New handlers ride on Echo v5's existing `identityMiddleware`; no new authn surface, no new subprocess execution, no new executable-file detection.

## Migration / Rollout

No migration required. Schema unchanged (`chat_conversations`, `chat_exchanges`); `Append` already bumps `chat_conversations.updated_at` at `store_postgres.go:266-271` (per #3924). Composition root is one-line addition per handler. Rollback: revert PR — additive `List` disappears, the two new GET routes disappear, CH-03's frozen three-route surface restored.

## Open Questions

None — all four product questions resolved (#3925).
