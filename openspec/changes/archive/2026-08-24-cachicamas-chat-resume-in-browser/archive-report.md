# Archive Report — `cachicamas-chat-resume-in-browser` (CH-08)

> **Status**: ready-to-merge
> **Worktree**: `cachicamas-worktrees/feat-chat-resume-ch08`
> **Branch**: `feat/chat-resume-ch08` against `origin/main` @ `084ad61a`
> **Date**: 2026-08-24
> **Final commit**: `{last WU commit on PR — `6ff75115` (WU-10)}` — full WU map below

## Outcome

CH-08 closes the resume-in-browser half of doc 0005's Wave 2 chat archetypes. The charter (`0005:842-853`) called for the durability of wave 2 to become visible to the human it was built for, and the four Gherkin scenarios at `0005:859-873` (CH-08.1) and `0005:880-895` (CH-08.2) now pass end-to-end. The closure retires two open items on the record: **R-14** (the listing half, CH-08.2), and **R-16** (the visible half — durable state independent of transport, CH-08.1). Two leaves land together: CH-08.1 reload-restore; CH-08.2 list-my-conversations.

## What shipped

### Code

- `backend/agent/src/chat/store.go` (+89 lines, modified) — added `ConversationSummary` struct + `List(participantID)` to the `ConversationStore` port (R-CCS-013 additive widening); `Append` and `Load` byte-unchanged.
- `backend/agent/src/chat/store_postgres.go` (+64 lines, modified) — `PostgresConversationStore.List` against `chat_conversations ORDER BY updated_at DESC` (single SELECT, no new index at v1).
- `backend/agent/src/chat/http.go` (+212 lines, modified) — `HandleReloadConversation(store)` and `HandleListConversations(store)` behind the existing `identityMiddleware`; both registered via `RegisterResumeRoutes` in `RegisterRoutes` under `/api/agent/conversations/:id` and `/api/agent/conversations`; DTO projections for `[]ExchangeDTO` and `[]ConversationSummaryDTO`.
- `backend/agent/src/cmd/chat/main.go` (+12 lines, modified) — wires `chatStore` to the two new handlers; factory closure byte-unchanged.
- `backend/agent/src/chat/store_test.go` (+155 lines) — `TestConversationStore_List` + 3 sub-tests (S-CCS-017, S-CCS-018, NFR-CCS-007 leak-detection loop).
- `backend/agent/src/chat/store_scenarios_test.go` (+124 lines, modified) — extended `RunConversationStoreScenarios` with S-CCS-017/018 + helper micro-test; helper latent-bug fix.
- `backend/agent/src/chat/store_postgres_test.go` (+86 lines, modified) — `TestPostgresConversationStore_List` + INTEGRATION scenarios.
- `backend/agent/src/chat/http_test.go` (+276 lines) — 5 sub-tests per handler (happy / cross-participant 403 / unknown 404 / empty 200).
- `frontend/src/lib/chat-types.ts` (+39 lines) — `ConversationSummary` + `ExchangeDTO` adjacent to (NOT extending) the closed `ChatStreamEvent` union (REQ-7 preservation).
- `frontend/src/lib/chat-api.ts` (+106 lines, modified) — `listConversations()` + `loadConversation(id)` typed helpers reusing `envelopeToResult`.
- `frontend/src/components/chat/chat-app.tsx` (+141 lines, modified) — `useVisibleTask$` mount fires both GETs in parallel; `useChatStream.reset(entries)` seeds; `ConversationList` re-mounted; `participantID` prop wired.
- `frontend/src/components/chat/conversation-list.tsx` (+43 lines, modified) — wire-shaped prop `ConversationSummary[]`; renders `id` / `age` / `turnCount`.
- `frontend/src/components/chat/format-relative-time.ts` (NEW, 42 lines) — relative-time helper for rail `age`.
- `frontend/src/routes/chat/index.tsx` (+39 lines, modified) — passes `participantID` from `requireSession` into `ChatApp`.

### Spec

- `openspec/specs/cachicamas-chat-resume-in-browser/spec.md` (NEW, 144 lines) — promoted verbatim from engram at archive.
- `openspec/specs/chat-conversation-store/spec.md` (+57 lines) — additive amendment: R-CCS-013/014, NFR-CCS-007, S-CCS-015..018. Follows the CH-07 amendment-header pattern verbatim (identifier-append-only, byte-unchanged existing R-CCS-001..012).
- `openspec/specs/frontend-chat-layer1/spec.md` (+45 lines) — additive amendment: REQ-8/9, S-FCL-008..011. REQ-7 closed `ChatStreamEvent` union unchanged.

### Doc 0005 bookkeeping

- Line `:3` — status updated to "9 of 12 shipped" (was 8 of 12 after CH-07).
- Lines `:992-993` — CH-08.1, CH-08.2 ticked.
- `openspec/AGENTS.md` § "Substrate preservation in `backend/agent`" — appended a one-line CH-08 pointer following the CH-07 pattern.

## Evidence gate

| Check | Command | Result |
|---|---|---|
| All 4 Gherkin scenarios pass (transcribed verbatim from `0005:859-873`, `0005:880-895`) | `cd backend/agent && go test -race -count=1 -run 'TestConversationStore_List\|TestHandleReload\|TestHandleList\|TestMemoryConversationStore_CH08' ./src/chat/...` | PASS (15/15 covering sub-tests) |
| Full backend suite | `cd backend/agent && make test` | GREEN, 18/18 packages, -race clean |
| Linter (backend) | `cd backend/agent && make lint` | 0 errors / 0 warnings |
| Build | `cd backend/agent && make build/chat` | `./bin/chat` produced (29,750,818 bytes) |
| Frontend units | `cd frontend && pnpm test:ci` | 584/584 passing across 157 suites |
| Linter (frontend) | `cd frontend && pnpm lint` | 0 errors / 0 warnings |
| Type-check (frontend) | `cd frontend && pnpm build.types` | clean (`tsc --incremental --noEmit`) |
| Import-boundary | `git diff --stat 084ad61a HEAD -- backend/agent/src/agent/import_boundary_test.go` | empty (R-CCS-002 / NFR-CCS-002 carried forward) |
| 10-file substrate | `git diff --stat 084ad61a HEAD -- backend/agent/src/agent/` | empty (NFR-TLS-003 satisfied) |
| Spec promotion | `git diff --stat 084ad61a HEAD -- openspec/specs/` | 3 files: 1 new (`chat-resume-in-browser/spec.md`); 2 amended (`chat-conversation-store/spec.md`, `frontend-chat-layer1/spec.md`) — additive only |
| Frontend wire-substrate | `git diff --stat 084ad61a HEAD -- frontend/src/components/chat/` | includes new `format-relative-time.ts` + rewired `chat-app.tsx` / `conversation-list.tsx`; REQ-7 closed `ChatStreamEvent` union preserved (verified via `git diff` showing no widening) |

## Work-unit commit map

| WU | Commit | Subject |
|---|---|---|
| WU-1 | `4b6f06c1` | `test(chat): scaffold ConversationStore.List + 3 RED cases` |
| WU-2 | `b09a83b7` | `feat(chat): ConversationStore.List + ConversationSummary (GREEN)` |
| WU-3 | `94205343` | `feat(chat): PostgresConversationStore.List GREEN + INTEGRATION scenario` |
| WU-4 | `7f689eab` | `refactor(chat): extend RunConversationStoreScenarios with List scenarios` |
| WU-5 | `11027e6a` | `test(chat): RED — reload + list HTTP handler sub-tests` |
| WU-6 | `588a15ce` | `feat(chat): HandleReloadConversation + HandleListConversations + composition-root wire` |
| WU-7 | `926d8682` | `test(chat,web): RED — listConversations + loadConversation typed helpers` |
| WU-8 | `f29dee6a` | `feat(chat,web): page-mount delta — reload + list wired through useChatStream.reset` |
| WU-9 | `eba2330c` | `docs(chat,0005): CH-08 closure — completion-checklist check-off + AGENTS pointer` |
| WU-10 | `6ff75115` | `docs(openspec): CH-08 archive — spec promotion + additive amendments + archive folder` |

## Strict TDD evidence

RED scaffold commits verified to contain only failing tests:

- `4b6f06c1` — 3 S-CCS-NNN sub-tests in `store_test.go` (RED compile-fail)
- `11027e6a` — 5 sub-tests in `http_test.go` (RED compile-fail)
- `926d8682` — 7 sub-tests in `chat-api.spec.ts` (RED runtime-error)

GREEN verified by `make test` (62 chat tests, race-clean) and `pnpm test:ci` (584 frontend tests). All RED scaffolds now GREEN.

## SUGGESTION (informational, not blocking)

S-CRI-002 / S-FCL-009 ("reload during a streaming turn shows what was recorded; the page does not claim streaming") lack a single end-to-end Gherkin-scoped test that combines (in-flight turn + page reload + recorded exchanges shown + idle status asserted). The behavior is enforced structurally — `useChatStream.reset(entries)` does NOT open an EventSource (`chat-app.tsx:122-126`); REQ-9 forbids auto-subscribe; the reload endpoint contract returns recorded exchanges only (R-CRI-001). A future milestone could add `TestPage_ReloadDuringStreamingTurn_ShowsRecordedAndDoesNotClaimStreaming` for explicit coverage.

## Source of truth updated

- `openspec/specs/cachicamas-chat-resume-in-browser/spec.md` (NEW — promoted)
- `openspec/specs/chat-conversation-store/spec.md` (additive amendment)
- `openspec/specs/frontend-chat-layer1/spec.md` (additive amendment)

## Rollback

Revert the 10 commits in reverse order (`6ff75115` → `4b6f06c1`). The revert is a single PR; the additive port widening means removing the `List` method returns the port to its CH-07 closed two-method shape; the additive spec amendments are identifier-append-only and revert cleanly.

## SDD cycle complete

The change has been fully planned (proposal + spec + design + tasks), implemented (10 WUs / 10 work-unit commits), verified (12/12 spec scenarios covered by passing tests), and archived.
