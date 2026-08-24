# Spec — `cachicamas-chat-resume-in-browser` (CH-08)

> **Change**: `cachicamas-chat-resume-in-browser` · **CH-08** (Wave 2, 9 of 12) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md#ch-08--resume-a-conversation-in-the-browser) (`0005:842-899`) · leaves CH-08.1 `[leaf]` (`0005:855-876`) · CH-08.2 `[leaf]` (`0005:878-899`)
> **Closes**: R-14 (listing half — CH-08.2), R-16 (visible half — CH-08.1, durable state independent of transport)
> **Status**: **new capability**, promoted verbatim to `openspec/specs/cachicamas-chat-resume-in-browser/spec.md` at archive.
> **Depends on**: CH-07 (`cachicamas-chat-store-adapter`), CH-05 (`cachicamas-frontend-chat-layer1`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable; the four Gherkin `S-CRI-NNN` rows are transcribed verbatim from `0005:859-873` and `0005:880-895`.
> **IDs**: requirements `R-CRI-NNN`, scenarios `S-CRI-NNN`, non-functional `NFR-CRI-NNN`. Append-only.
> **Prefix verification**: `CRI` verified collision-free across `openspec/specs/`, `openspec/changes/` and `docs/architecture/milestones/0005-…md`.

## Purpose

CH-06 + CH-07 made conversation durability real on the server. CH-08 closes the visible half: the chat page (`chat-app.tsx:36`) mounts `useChatStream([])` with an empty seed, so a reload or revisit lands on a blank transcript even when `ConversationStore` has the conversation recorded. This capability ships two non-streaming GET endpoints (`/api/agent/conversations/:id` and `/api/agent/conversations`) plus the page-mount delta that consumes them, so wave-2 durability is visible to the human it was built for (charter, `0005:848`). One conversation per participant (`conversationID == participantID`, decisions #3925); reload returns exchanges only; list returns full slice most-recent-first; deep-link `?with=<slug>` stays inert.

## Coverage — charter traced to requirements and scenarios

| Charter clause / decision | Requirements | Scenarios |
|---|---|---|
| CH-08 acceptance `0005:850` (reload shows the conversation as the participant left it) | R-CRI-001, R-CRI-003 | S-CRI-001 |
| CH-08.1 Gherkin `0005:862-866` (reload restores both exchanges in order, input accepts a new prompt) | R-CRI-001, R-CRI-003 | S-CRI-001 |
| CH-08.1 Gherkin `0005:868-872` (reload-during-streaming: exchanges recorded before reload are shown, no false streaming) | R-CRI-001, R-CRI-003 | S-CRI-002 |
| CH-08.2 Gherkin `0005:885-889` (own-list: each participant sees only their own conversations) | R-CRI-002, R-CRI-005, NFR-CRI-003 | S-CRI-003 |
| CH-08.2 Gherkin `0005:891-895` (empty list is success, not not-found) | R-CRI-002 | S-CRI-004 |
| D-1 one-per-participant (`conversationID == participantID`) | R-CRI-001, R-CRI-002 | S-CRI-001, S-CRI-003 |
| D-2 exchanges-only DTO | R-CRI-004 | S-CRI-001 |
| D-3 full list, most-recent-first, no pagination | R-CRI-002 | S-CRI-003, S-CRI-004 |
| D-4 deep-link `?with=<slug>` stays inert | R-CRI-003 | S-CRI-001 |
| Substrate preservation (NFR-TLS-003) | NFR-CRI-001 | (gated by `git diff`) |

---

## Requirements

### R-CRI-001 — `GET /api/agent/conversations/:id` returns the participant's recorded exchanges

The system MUST expose `GET /api/agent/conversations/:id` behind `identityMiddleware(resolver)`. The handler MUST call `store.Load(participantID)` (R-CCS-006, R-CCS-010) and project the resulting `[]Exchange` to a `[]ExchangeDTO` whose fields mirror the port's `Exchange` field-for-field (R-CRI-004, REQ-7 closed-union enforcement). The URL parameter `:id` MUST equal the caller's `participantID` from the identity middleware; a request whose `:id` does not match the authenticated participant's id MUST be refused as `403 not_found` (mirrors R-CHS-004.b — refuse, do not probe). A request whose `:id` matches the authenticated participant but the store returns `ErrConversationNotFound` MUST respond with `404 not_found`. A request whose `:id` matches and the store returns `[]Exchange` (an existing participant with no turns yet, per CH-06.1 D-1's `Append` semantics) MUST respond with `200 []ExchangeDTO{}` — the JSON shape is always a slice. The handler MUST NOT mutate store state.

#### Scenario: S-CRI-001 — reloading the page restores the conversation (Gherkin verbatim, `0005:862-866`)

- Given an employee who has completed two turns and reloads the page
- When the page loads
- Then both exchanges are shown in their original order
- And the input accepts a new prompt that continues the same conversation

#### Scenario: S-CRI-002 — a reload during a streaming turn shows what was recorded (Gherkin verbatim, `0005:868-872`)

- Given an employee who reloads while a turn is streaming
- When the page loads
- Then the exchanges recorded before the reload are shown
- And the page does not claim the turn is still streaming

### R-CRI-002 — `GET /api/agent/conversations` returns the participant's own summaries

The system MUST expose `GET /api/agent/conversations` behind `identityMiddleware(resolver)`. The handler MUST call `store.List(participantID)` (R-CCS-013, the third additive method on the closed two-method port) and respond with `200 []ConversationSummaryDTO`. The response MUST be sorted most-recent-first by `chat_conversations.updated_at DESC` (decisions D-3). A request whose identity middleware returns no identity MUST be refused as `401 not_found`. An authenticated participant whose `List` returns zero rows MUST receive `200 []ConversationSummaryDTO{}` (empty slice) — the empty list is success, not-not-found (mirrors S-CRI-004 and S-CCS-018). The handler MUST NOT widen the closed `ChatStreamEvent` union (REQ-7 preserved).

#### Scenario: S-CRI-003 — a participant sees their own conversations and no others (Gherkin verbatim, `0005:885-889`)

- Given two participants who have each held conversations
- When one of them requests their list
- Then the list contains only their own conversations
- And each entry identifies its conversation well enough to open it

#### Scenario: S-CRI-004 — a participant with no conversations gets an empty list (Gherkin verbatim, `0005:891-895`)

- Given an authenticated participant who has never held a conversation
- When they request their list
- Then the list is empty
- And the response is a success rather than a not-found

### R-CRI-003 — The chat page mounts both endpoints on first paint

The page (`chat-app.tsx`) MUST, on its Qwik `useVisibleTask$` mount, fire `GET /api/agent/conversations/:id` and `GET /api/agent/conversations` in parallel against the participant id surfaced from `requireSession` (`routes/chat/index.tsx`). The page MUST call `useChatStream.reset(entries)` with the recorded exchanges from the reload endpoint (seeding the buffer without opening an `EventSource`); MUST re-mount the `ConversationList` component against the summary list (CH-05 D-3's rail-drop is undone, now wire-driven); and MUST NOT auto-open an `EventSource` for the prior conversation id. The deep-link `?with=<slug>` MUST continue to resolve to an `agentSlug` from `staff.ts` (CH-05 D-6) but MUST NOT drive which conversation loads — the participant's recorded transcript is the source of truth (decisions D-4). The page MUST render in a state where `Composer` accepts a new prompt on completion of the GETs.

### R-CRI-004 — `ConversationSummaryDTO` and `ExchangeDTO` are closed transport projections

`ConversationSummaryDTO` MUST carry exactly `{ conversationID: string, lastActivityAt: ISO8601, turnCount: int }`. `ExchangeDTO` MUST mirror `chat.Exchange` field-for-field (`Position`, `PromptText`, `AssistantText`, `Partial`, `TerminalKind`, `FailureCategory`, `FinishReason`, `MessageIDs` — eight fields per D-7 of CH-06). The DTOs MUST live in `frontend/src/lib/chat-types.ts` (NFR-CRI-002 import-boundary). The wire MUST NOT invent new fields beyond what the port's projection returns (REQ-7 enforcement: the closed `ChatStreamEvent` union is not widened; `ConversationSummaryDTO` and `ExchangeDTO` are new types adjacent to it, never additions to it).

### R-CRI-005 — `ConversationList` is re-mounted against the real wire shape

The `ConversationList` component (`frontend/src/components/chat/conversation-list.tsx` — surviving from CH-05.1, not mounted per D-3) MUST accept a `ConversationSummary[]` prop whose entries carry `conversationID` (the participant id per D-1), `lastActivityAt` (ISO8601, drives a relative-time `age` helper), and `turnCount`. The mock `Conversation` type in `lib/mock/chat.ts` MUST survive for `hero-proof` and `front-desk` surfaces (CH-05 D-3); the rail MUST be driven by `ConversationSummary[]` from the wire.

---

## Non-functional requirements

### NFR-CRI-001 — Substrate preservation

No file under `backend/agent/src/agent/` is modified. The ten-file substrate list (per `openspec/AGENTS.md` § "Substrate preservation in `backend/agent`") is byte-unchanged. NFR-TLS-003.

### NFR-CRI-002 — Import-boundary

Frontend wire types live under `frontend/src/lib/chat-types.ts`. No new top-level frontend dependencies are admitted (CH-05.1 baseline preserved).

### NFR-CRI-003 — Participant-scoped reads under identity middleware

`ConversationStore.List` is a participant-scoped read (mirrors `Load`). Implementations MUST NOT return rows for any other participant even under a corrupted or missing identity. The guard pattern is: identity middleware runs the participant through `getIdentity(c)` before the handler's `List` call; the handler uses that resolved id, not any URL or header value.

### NFR-CRI-004 — Sub-millisecond reload at v1's request rate

`Load` and `List` defensive-copy semantics preserve against caller-side mutation (NFR-CCS-004 carried forward). `MemoryConversationStore` iterates its map under the same `sync.Mutex`; `PostgresConversationStore` uses one `SELECT … FROM chat_conversations ORDER BY updated_at DESC` against the existing schema (no new index at v1).

---

## Acceptance — proposal success criteria mapped to scenarios

| Acceptance criterion | Evidence |
|---|---|
| Both Gherkin scenarios from `0005:859-873` (CH-08.1) pass, transcribed verbatim | S-CRI-001, S-CRI-002 |
| Both Gherkin scenarios from `0005:880-895` (CH-08.2) pass, transcribed verbatim | S-CRI-003, S-CRI-004 |
| Substrate list in `openspec/AGENTS.md` unchanged (only the CH-08 pointer line added) | NFR-CRI-001 |
| CH-07.2 closed-port guard preserved under R-CCS-013 additive widening | R-CRI-002, NFR-CRI-003 |
| `cd backend/agent && make test` race-clean (full suite green) | (whole spec) |
| `cd backend/agent && make lint && make build/chat` clean | (whole spec) |
| Frontend `pnpm --filter frontend test` + `pnpm --filter frontend test:e2e` green | (whole spec) |
| Cross-participant reload refused as 403 not_found | R-CRI-001 |
| Unknown conversation returns 404 not_found | R-CRI-001 |
| Empty list returns 200 with `[]` (not 404) per S-CRI-004 | R-CRI-002 |

## Explicit non-requirements

| Not required here | Owner / reason |
|---|---|
| Resuming the live stream when a tab closes mid-stream | Charter explicit out-of-scope (`0005:852`); "resume is not a transport feature" (research finding cited in doc 0005 preamble). Deferred to a future reconnect milestone. |
| Branching, renaming, deleting, searching conversations | Charter explicit out-of-scope (`0005:852`); each named in the register. Deferred. |
| `conversation_id` schema migration (many-per-participant model) | D-1 resolves one-per-participant; current schema (`chat_conversations.participant_id` PK) is source of truth. |
| SSE replay / `Last-Event-ID`, IndexedDB mirror, offline reads | Deferred. R-CRI-001's reload path is online-only at v1. |
| Pagination, sort, filtering, search on the list endpoint | Charter explicit out-of-scope (`0005:899`); full slice, most-recent-first only at v1. |
| Widening the closed `ChatStreamEvent` union | REQ-7 enforced; `ConversationSummaryDTO` and `ExchangeDTO` are new types adjacent to it. |
| Modifying any file under `backend/agent/src/agent/` | NFR-TLS-003 / NFR-CRI-001. |
| A `ConversationView` envelope (metadata + exchanges) | D-2 — exchanges only at v1. |
| Modifying `openspec/specs/chat-conversation-store/spec.md` or `frontend-chat-layer1/spec.md` outside their additive `## ADDED Requirements` blocks | Identifier-append-only per CH-07 precedent; the deltas are recorded separately in the change folder and synced to the main specs at archive. |

## Cross-references

- Doc 0005 § CH-08 charter (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:842-853`) and the two leaves (`0005:855-876`, `0005:878-899`).
- The Gherkin feature files at `0005:859-873` (CH-08.1) and `0005:882-895` (CH-08.2) are the authoritative scenario wording — the `S-CRI-NNN` scenario lines in this spec are transcribed verbatim.
- `backend/agent/src/chat/store.go:97-115` — the closed two-method port `ConversationStore` (R-CCS-010). CH-08 widens it additively (R-CCS-013, see the additive delta in `chat-conversation-store`).
- `backend/agent/src/chat/http.go` — existing handlers and the `RegisterRoutes` surface; CH-08 adds two GET handlers behind the existing `identityMiddleware`.
- `frontend/src/components/chat/chat-app.tsx:36` — the `useChatStream([])` mount; CH-08 adds the page-mount delta.
- `frontend/src/lib/chat-types.ts` — the closed `ChatStreamEvent` union (REQ-7); CH-08 adds `ConversationSummary` + `ExchangeDTO` adjacent to it.
- `frontend/src/components/chat/conversation-list.tsx` — the surviving (not-mounted) rail from CH-05.1; CH-08 re-mounts it against `ConversationSummary[]`.
- `openspec/specs/chat-conversation-store/spec.md` — the additive `## ADDED Requirements` block for R-CCS-013 / R-CCS-014 / NFR-CCS-007.
- `openspec/specs/frontend-chat-layer1/spec.md` — the additive `## ADDED Requirements` block for REQ-8 / REQ-9.
- `openspec/AGENTS.md` § "Substrate preservation in `backend/agent`" — the ten-file substrate list this spec leaves byte-unchanged.
