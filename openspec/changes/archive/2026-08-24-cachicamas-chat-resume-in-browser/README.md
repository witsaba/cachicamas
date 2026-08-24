# Archive — `cachicamas-chat-resume-in-browser` (CH-08)

This folder archives the CH-08 SDD change — Wave 2, 9 of 12 milestones shipped.

| File | What it carries |
|------|----------------|
| `proposal.md` | Intent, scope, decisions, success criteria. Source: engram `sdd/cachicamas-chat-resume-in-browser/proposal` (#3926). |
| `explore.md` | Charter-vs-current-state inventory and approach comparison. Source: engram `sdd/cachicamas-chat-resume-in-browser/explore` (#3924). |
| `decisions.md` | Four product-decision locks (one-per-participant, exchanges-only DTO, full-list ordering, `?with=` deep-link inert). Source: engram `…/decisions` (#3925). |
| `design.md` | Technical approach — additive third method on `ConversationStore`, two new GET endpoints, page-mount delta. Source: engram `…/design` (#3928). |
| `tasks.md` | 10 work units across 4 phases. Source: engram `…/tasks` (#3929). |
| `apply-progress.md` | Apply-phase evidence — RED-GREEN-REFACTOR per WU, focused test commands and results, files changed. |
| `specs/cachicamas-chat-resume-in-browser/spec.md` | Change-local copy of the promoted spec. |
| `verify-report.md` | Produced by `sdd-verify` after this PR lands (NOT included in this PR per orchestrator). |
| `archive-report.md` | Produced by `sdd-archive` after verify closes (NOT included in this PR per orchestrator). |

## What shipped

CH-08.1 + CH-08.2 — Resume a conversation in the browser.

Two non-streaming GET endpoints behind the existing `identityMiddleware`:

- `GET /api/agent/conversations/:id` returns the participant's recorded exchanges (`[]ExchangeDTO`).
- `GET /api/agent/conversations` returns the participant's own summaries (`[]ConversationSummary`).

A page-mount `useVisibleTask$` on `chat-app.tsx` fires both in parallel, seeds `useChatStream.reset(entries)`, and re-mounts the `ConversationList` rail against the wire shape. The closed `ChatStreamEvent` union is preserved (REQ-7); `ExchangeDTO` and `ConversationSummary` are new types adjacent to it, never additions.

## Notable decisions

1. **Additive `List` method** (`R-CCS-013`) — widens the closed port without replacing its two-method declaration. CH-08's port widening preserves `R-CCS-010`'s anticipatory text: "CH-08.2 widens the port to a list by extending, not replacing, this declaration."
2. **One conversation per participant** — `conversationID == participantID`. Schema's `chat_conversations.participant_id` PK already encodes this; zero migration. List returns 0 or 1 entry.
3. **Exchanges-only DTO** — `ExchangeDTO` mirrors `chat.Exchange` field-for-field (D-7's 8 fields). No `ConversationView` envelope.
4. **`?with=<slug>` deep-link stays inert** — resolves to `agentSlug` from `staff.ts` but does not drive which conversation loads (D-4).

## Substrate preservation

No file under `backend/agent/src/agent/` is modified; the ten-file substrate list survives byte-clean (NFR-TLS-003). The CH-08 pointer in `openspec/AGENTS.md` records this invariant and the additive widening precedent.

## Spec promotion

`openspec/specs/cachicamas-chat-resume-in-browser/spec.md` is created (the new capability). `chat-conversation-store/spec.md` and `frontend-chat-layer1/spec.md` get additive `## ADDED Requirements` blocks (the same pattern CH-07 used) — every prior requirement, scenario, and NFR is byte-unchanged.
