# Apply Progress — `cachicamas-chat-resume-in-browser` (CH-08)

> Archived copy of the live apply-progress observation #3932 in engram (topic_key `sdd/cachicamas-chat-resume-in-browser/apply-progress`).

**Mode**: Strict TDD (`strict_tdd: true`, runner `cd backend/agent && make test`)
**Branch**: `feat/chat-resume-ch08` from `main` @ `084ad61a`
**Delivery**: single-pr (size:exception per preflight #3923); 10 WUs land in one PR.

## Final evidence gate

Backend:
- `cd backend/agent && make test` → ok
- `cd backend/agent && make lint` → 0 issues
- `cd backend/agent && make build/chat` → produced `./bin/chat`

Frontend:
- `cd frontend && pnpm test:ci` (vitest) → 584 tests passing
- `cd frontend && pnpm lint` → 0 errors, 0 warnings
- `cd frontend && pnpm build.types` (tsc) → clean

Substrate:
- No file under `backend/agent/src/agent/` modified (NFR-TLS-003)
- `git diff main -- backend/agent/src/agent/` → empty

## Strict TDD Cycle Evidence

| WU | RED (test + commit SHA) | GREEN (impl + commit SHA) | REFACTOR | Tests result |
|----|--------------------------|---------------------------|----------|--------------|
| WU-1 | RED scaffold `4b6f06c1` (3 S-CCS-NNN sub-tests) | n/a | n/a | 3 RED compile-fail |
| WU-2 | n/a | MemoryConversationStore.List + ConversationSummary `b09a83b7` | defensive copy + activity map | 3/3 GREEN |
| WU-3 | (combined) | PostgresConversationStore.List + INTEGRATION scenarios `94205343` | n/a | unit-GREEN; INTEGRATION scenarios gated |
| WU-4 | (combined) | RunConversationStoreScenarios + S-CCS-017/018 + helper micro-test `7f689eab` | shared helper latent-bug fix | 7/7 GREEN |
| WU-5 | RED HTTP sub-tests `11027e6a` (5 cases) | n/a (WU-6) | n/a | 5 RED compile-fail |
| WU-6 | n/a | HandleReloadConversation + HandleListConversations + RegisterResumeRoutes + main.go wire `588a15ce` | n/a | 5/5 GREEN |
| WU-7 | RED frontend helpers `926d8682` (7 cases) | n/a (WU-8) | n/a | 7 RED runtime-error |
| WU-8 | n/a | types + helpers + page-mount delta + rail re-mount `f29dee6a` | n/a | 7/7 GREEN |
| WU-9 | n/a (docs) | doc 0005 closure + AGENTS pointer `eba2330c` | n/a | N/A |
| WU-10 | n/a (archive) | spec promotion + additive amendments + archive folder (this WU) | n/a | N/A |

## Work Unit Evidence (focused test commands + result)

| WU | Focused test command | Result | Runtime harness | Rollback boundary |
|----|------------------------|--------|------------------|-------------------|
| WU-1 | `go test -race -count=1 -run TestConversationStore_List ./src/chat/...` | 3 RED compile-fail | N/A (unit) | Revert store_test.go |
| WU-2 | same | 3/3 GREEN | N/A | Revert store.go (List impl) |
| WU-3 | `go test -race -count=1 -run TestPostgresConversationStore_List ./src/chat/...` | unit-GREEN | INTEGRATION=1 cross-process | Revert store_postgres.go |
| WU-4 | `go test -race -count=1 -run TestMemoryConversationStore_CH08Scenarios_PassUnchanged ./src/chat/...` | 7/7 GREEN | N/A | Revert store_scenarios_test.go |
| WU-5 | `go test -race -count=1 -run 'TestHandleReloadConversation\|TestHandleListConversations' ./src/chat/...` | 5 RED compile-fail | N/A | Revert http_test.go |
| WU-6 | same | 5/5 GREEN | N/A | Revert http.go + main.go |
| WU-7 | `pnpm test:ci -- chat-api.spec.ts` | 7 RED runtime-error | N/A (vitest) | Revert chat-api.spec.ts |
| WU-8 | `pnpm test:ci` (chat-app, conversation-list, routes/chat specs) | 584 GREEN (574 baseline + 10 net) | N/A | Revert chat-app.tsx, conversation-list.tsx, routes/chat/index.tsx, chat-types.ts, chat-api.ts |
| WU-9 | N/A | N/A (docs) | N/A | Revert docs + AGENTS.md |
| WU-10 | N/A | N/A (archive) | N/A | Revert openspec/ |

## Files Changed (cumulative, 23 files)

### Backend (Go)

| File | Action | WU |
|------|--------|----|
| backend/agent/src/chat/store_test.go | Modified (added TestConversationStore_List + 3 sub-tests) | WU-1 |
| backend/agent/src/chat/store.go | Modified (ConversationSummary + List on port + activity map) | WU-2 |
| backend/agent/src/chat/store_postgres.go | Modified (List stub→impl) | WU-2 + WU-3 |
| backend/agent/src/chat/store_postgres_test.go | Modified (List tests) | WU-3 |
| backend/agent/src/chat/store_scenarios_test.go | Modified (S-CCS-017/018 + helper micro-test + helper latent-bug fix) | WU-4 |
| backend/agent/src/chat/http_test.go | Modified (5 RED scaffold sub-tests) | WU-5 |
| backend/agent/src/chat/http.go | Modified (Handlers + RegisterResumeRoutes + DTO projections) | WU-6 |
| backend/agent/src/cmd/chat/main.go | Modified (RegisterResumeRoutes wire) | WU-6 |

### Frontend (TypeScript / Qwik)

| File | Action | WU |
|------|--------|----|
| frontend/src/lib/chat-types.ts | Modified (ConversationSummary + ExchangeDTO adjacent to closed union) | WU-8 |
| frontend/src/lib/chat-api.ts | Modified (listConversations + loadConversation helpers) | WU-8 |
| frontend/src/lib/chat-api.spec.ts | Modified (7 RED scaffold sub-tests) | WU-7 (tests only) |
| frontend/src/components/chat/format-relative-time.ts | Created (rail age helper) | WU-8 |
| frontend/src/components/chat/conversation-list.tsx | Modified (wire-shaped prop) | WU-8 |
| frontend/src/components/chat/conversation-list.spec.tsx | Modified (rewired against ConversationSummary[]) | WU-8 |
| frontend/src/components/chat/chat-app.tsx | Modified (useVisibleTask$ mount + reset(entries) + rail re-mount + participantID prop) | WU-8 |
| frontend/src/components/chat/chat-app.spec.tsx | Modified (rail-mounted assertion + participantID prop) | WU-8 |
| frontend/src/routes/chat/index.tsx | Modified (passes participantID from requireSession) | WU-8 |
| frontend/src/routes/chat/index.spec.tsx | Modified (new participantID-flow test + retired-rail test updated) | WU-8 |

### Docs + Specs

| File | Action | WU |
|------|--------|----|
| docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md | Modified (status 9/12 + CH-08 shipped prose) | WU-9 |
| openspec/AGENTS.md | Modified (CH-08 substrate pointer appended) | WU-9 |
| openspec/specs/cachicamas-chat-resume-in-browser/spec.md | Created (new capability promoted verbatim) | WU-10 |
| openspec/specs/chat-conversation-store/spec.md | Modified (additive `## ADDED Requirements` block) | WU-10 |
| openspec/specs/frontend-chat-layer1/spec.md | Modified (additive `## ADDED Requirements` block) | WU-10 |
| openspec/changes/archive/2026-08-24-cachicamas-chat-resume-in-browser/ | Created (README + proposal + specs/cachicamas-chat-resume-in-browser/spec.md) | WU-10 |

## Commits landed (chronological)

1. WU-1: `4b6f06c1` test(chat): scaffold ConversationStore.List + 3 RED cases (WU-1)
2. WU-2: `b09a83b7` feat(chat): ConversationStore.List + ConversationSummary (GREEN WU-2)
3. WU-3: `94205343` feat(chat): PostgresConversationStore.List GREEN + INTEGRATION scenario (WU-3)
4. WU-4: `7f689eab` refactor(chat): extend RunConversationStoreScenarios with List scenarios (WU-4)
5. WU-5: `11027e6a` test(chat): RED — reload + list HTTP handler sub-tests (WU-5)
6. WU-6: `588a15ce` feat(chat): HandleReloadConversation + HandleListConversations + composition-root wire (WU-6)
7. WU-7: `926d8682` test(chat,web): RED — listConversations + loadConversation typed helpers (WU-7)
8. WU-8: `f29dee6a` feat(chat,web): page-mount delta — reload + list wired through useChatStream.reset (WU-8)
9. WU-9: `eba2330c` docs(chat,0005): CH-08 closure — completion-checklist check-off + AGENTS pointer (WU-9)
10. WU-10: this WU's commit — docs(openspec): CH-08 archive — spec promotion + additive amendments + archive folder

## Deviations from Design

- **WU-4 latent-bug fix:** the CH-07-era `RunConversationStoreScenarios` had `S-CCS-007` and `S-CCS-008` both using participant id `"alice"`, so calling the helper at unit level (which CH-07's INTEGRATION-gated runners avoided) leaked state. Renamed to `"scn-007-alice"` / `"scn-008-alice"`. Documented in commit WU-4. Strict pre-existing bug that the new unit-level runner at WU-4 exposed; fix is targeted and does not change the CH-06/CH-07 contract.
- **WU-6 stub-then-replace pattern:** `PostgresConversationStore.List` shipped with a `(nil, errNotImplemented)` stub in WU-2 to keep the package compiling while the WU-2 in-memory test was GREEN. WU-3 replaced the stub with the production SELECT in a single commit, alongside the INTEGRATION-gated scenario. This mirrors the CH-07 WU-5/6 stub pattern (the existing test `TestPostgresConversationStore_StubReturnsNotImplemented` was the WU-5 RED scaffold for that prior milestone); CH-08 condensed it into one RED+GREEN commit for the chat surface.

## Issues Found

None. Every WU landed at its committed design; the closed `ChatStreamEvent` union is byte-unchanged (REQ-7 enforced); the ten-file substrate list under `backend/agent/src/agent/` is byte-unchanged (NFR-TLS-003).

## Workload / PR Boundary

- **Mode**: single PR (size:exception per preflight #3923, user pre-granted)
- **Commits**: 10 work-unit commits + 1 archive doc commit (this WU-10) = 11 commits
- **Boundary**: branch `feat/chat-resume-ch08` against `main` @ `084ad61a`
- **Estimated changed lines** (per design forecast): 1100–1600 lines; the user explicitly granted extension at preflight

## Status

**10/10 WUs complete. Ready for sdd-verify.**
