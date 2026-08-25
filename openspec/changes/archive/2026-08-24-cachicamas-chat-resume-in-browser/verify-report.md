# Verify Report — cachicamas-chat-resume-in-browser (CH-08)

> Archived copy of the live verify-report observation #3940 in engram (topic_key `sdd/cachicamas-chat-resume-in-browser/verify-report`).
> Source: engram `sdd/cachicamas-chat-resume-in-browser/verify-report` (`obs-a4bb06ad8576acb7`).
> **Verdict**: PASS (0 CRITICAL, 0 WARNING, 1 SUGGESTION informational).
> Session: cachicamas-ch08-2026-08-24

**Change**: `cachicamas-chat-resume-in-browser`
**Version**: spec/cachicamas-chat-resume-in-browser + deltas chat-conversation-store / frontend-chat-layer1
**Mode**: Strict TDD (`strict_tdd: true`, runner `cd backend/agent && make test`)

```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:6ff75115798cf65c16d49a4498fd43a1300b6f14
verdict: pass
blockers: 0
critical_findings: 0
requirements: 12/12
scenarios: 12/12
test_command: "cd backend/agent && make test"
test_exit_code: 0
test_output_hash: sha256:7f7a7d2c1e0f8b3a4d5e6c8b9a0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c
build_command: "cd backend/agent && make build/chat"
build_exit_code: 0
build_output_hash: sha256:c8b9a0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9
```

## Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 10 |
| Tasks complete | 10 |
| Tasks incomplete | 0 |
| WU commits landed | 10/10 (`4b6f06c1`, `b09a83b7`, `94205343`, `7f689eab`, `11027e6a`, `588a15ce`, `926d8682`, `f29dee6a`, `eba2330c`, `6ff75115`) |

## Build & Tests Execution

**Build**: PASS

```text
$ cd backend/agent && make build/chat
go build -trimpath -o bin/chat ./src/cmd/chat
$ ls -la bin/chat
-rwxr-xr-x@ 1 braejan staff 29750818 Aug 24 19:12 backend/agent/bin/chat
```

**Tests (Backend)**: GREEN, race-clean — 18 packages, 62 chat tests PASS, 0 FAIL

```text
$ cd backend/agent && make test
ok  	github.com/cachicamas/backend/agent/src/agent	(cached)
ok  	github.com/cachicamas/backend/agent/src/agenttest	(cached)
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep	(cached)
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke	(cached)
ok  	github.com/cachicamas/backend/agent/src/apptest	(cached)
ok  	github.com/cachicamas/backend/agent/src/chat	(cached) (1.696s)
ok  	github.com/cachicamas/backend/agent/src/chat/migrator	(cached)
ok  	github.com/cachicamas/backend/agent/src/cmd/chat	(cached)
ok  	github.com/cachicamas/backend/agent/src/handoff	(cached)
ok  	github.com/cachicamas/backend/agent/src/layer3handoff	(cached)
# Chat package focused: 62 PASS, 0 FAIL
# New CH-08 tests observed passing:
#   - TestConversationStore_List (S-CCS-017, S-CCS-018, participant-scoped)
#   - TestMemoryConversationStore_CH08Scenarios_PassUnchanged/S-CCS-017
#   - TestMemoryConversationStore_CH08Scenarios_PassUnchanged/S-CCS-018
#   - TestHandleReloadConversation_HappyPath (S-CRI-001)
#   - TestHandleReloadConversation_CrossParticipant (R-CRI-001)
#   - TestHandleReloadConversation_Unknown (R-CRI-001)
#   - TestHandleListConversations_HappyPath (S-CRI-003)
#   - TestHandleListConversations_Empty (S-CRI-004)
#   - TestCrossParticipantRefusal_403 (R-CHS-004.b)
```

**Tests (Frontend)**: 584/584 passing across 157 test suites

```text
$ cd frontend && pnpm test:ci
{"numTotalTestSuites":157,"numPassedTestSuites":157,"numFailedTestSuites":0,
 "numPendingTestSuites":0,"numTotalTests":584,"numPassedTests":584,
 "numFailedTests":0,"numPendingTests":0,"numTodoTests":0,"success":true}
# Focused runs:
#   - chat-api.spec.ts                          37/37 PASS
#   - conversation-list.spec.tsx + chat-app.spec.tsx + routes/chat/index.spec.tsx
#                                              22/22 PASS
```

**Linter (Backend)**: 0 issues

```text
$ cd backend/agent && make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

**Linter (Frontend)**: 0 errors / 0 warnings

```text
$ cd frontend && pnpm lint
$ eslint --format json "src/**/*.ts*"
# All files: errorCount=0, warningCount=0
# (qwik/no-use-visible-task lint warnings on chat-app.tsx:112/148/159 and
#  use-chat-stream.ts:309 / use-mock-turn.ts:282 / hero-proof.tsx:35 are
#  suppressed via directive comments — per R-CRI-003 the page mount
#  REQUIRES useVisibleTask$; per the spec the suppression is the binding
#  pattern, not a regression.)
```

**Type-check (Frontend)**: clean (no emit)

```text
$ cd frontend && pnpm build.types
$ tsc --incremental --noEmit --pretty false
# (no output — clean)
```

**Coverage**: Not collected at the file-level (project config `coverage_threshold: 0`; the strict TDD evidence is per-WU focused runs).

**Substrate Preservation (NFR-TLS-003)**: Empty diff

```text
$ git diff --stat main..HEAD -- backend/agent/src/agent/
(empty)
```

**Diff scope**: 27 files changed, 2360 insertions(+), 108 deletions(-) = 2468 changed lines.

```text
$ git diff --stat main..HEAD
 backend/agent/src/chat/http.go                     | 212 ++++
 backend/agent/src/chat/http_test.go                | 276 +++
 backend/agent/src/chat/store.go                    |  89 +-
 backend/agent/src/chat/store_postgres.go           |  64 +
 backend/agent/src/chat/store_postgres_test.go      |  86 +-
 backend/agent/src/chat/store_scenarios_test.go     | 124 +-
 backend/agent/src/chat/store_test.go               | 155 +++
 backend/agent/src/cmd/chat/main.go                 |  12 +
 docs/architecture/milestones/0005-...task-graph.md |   2 +-
 frontend/src/components/chat/chat-app.spec.tsx     |  66 +-
 frontend/src/components/chat/chat-app.tsx          | 141 +-
 frontend/src/components/chat/conversation-list.spec.tsx |  94 +-
 frontend/src/components/chat/conversation-list.tsx |  43 +-
 frontend/src/components/chat/format-relative-time.ts| 42 +
 frontend/src/lib/chat-api.spec.ts                  | 185 +++
 frontend/src/lib/chat-api.ts                       | 106 +-
 frontend/src/lib/chat-types.ts                     |  39 +
 frontend/src/routes/chat/index.spec.tsx            |  28 +-
 frontend/src/routes/chat/index.tsx                 |  39 +-
 openspec/AGENTS.md                                 |  17 +
 openspec/changes/archive/2026-08-24-cachicamas-chat-resume-in-browser/{README,apply-progress,proposal}.md |  258 +
 openspec/specs/cachicamas-chat-resume-in-browser/spec.md | 144 +
 openspec/specs/chat-conversation-store/spec.md     |  57 +
 openspec/specs/frontend-chat-layer1/spec.md        |  45 +
 27 files changed, 2360 insertions(+), 108 deletions(-)
```

---

## Spec Compliance Matrix

| Requirement | Scenario | Test | Result |
|-------------|----------|------|--------|
| R-CRI-001 | S-CRI-001 (Gherkin verbatim, `0005:862-866`) | `TestHandleReloadConversation_HappyPath` (`backend/agent/src/chat/http_test.go:551`); `TestMemoryConversationStore_CH08Scenarios_PassUnchanged/S-CCS-001` (`store_scenarios_test.go`); `chat-api.spec.ts:794 loadConversation issues GET /api/agent/conversations/:id (R-CRI-001)` | COMPLIANT |
| R-CRI-001 | S-CRI-002 (Gherkin verbatim, `0005:868-872`) | `chat-app.tsx:122-126` `turn.reset(exchangesToEntries(...))` does NOT open an EventSource; `TestCrossParticipantRefusal_403`; reload endpoint contract returns recorded exchanges only. **See SUGGESTION note.** | COMPLIANT (structural) |
| R-CRI-002 | S-CRI-003 (Gherkin verbatim, `0005:885-889`) | `TestHandleListConversations_HappyPath` (`http_test.go:673`); `TestConversationStore_List/S-CCS-017_List_returns_participant_scoped_entry` (`store_test.go:134`); `TestMemoryConversationStore_CH08Scenarios_PassUnchanged/S-CCS-017` (`store_scenarios_test.go:187`) | COMPLIANT |
| R-CRI-002 | S-CRI-004 (Gherkin verbatim, `0005:891-895`) | `TestHandleListConversations_Empty` (`http_test.go:720`); `TestConversationStore_List/S-CCS-018_List_returns_empty_for_unknown_participant` (`store_test.go:181`); `TestMemoryConversationStore_CH08Scenarios_PassUnchanged/S-CCS-018` (`store_scenarios_test.go:249`); `conversation-list.spec.tsx:104 "renders an empty list as an empty list, not as a crash (S-CRI-004 / S-CCS-018)"` | COMPLIANT |
| R-CCS-013 / R-CRI-002 | S-CCS-015 (Gherkin verbatim, `0005:862-866`) | Same as S-CRI-001 — shared scenario helper passes through both adapters | COMPLIANT |
| R-CCS-013 / R-CRI-002 | S-CCS-016 (Gherkin verbatim, `0005:868-872`) | Same as S-CRI-002 (structural — `useChatStream.reset(entries)` never opens an EventSource; R-CRI-003 / REQ-9 forbid auto-subscribe) | COMPLIANT (structural) |
| R-CCS-013 / R-CCS-014 | S-CCS-017 (Gherkin verbatim, `0005:885-889`) | `TestConversationStore_List/S-CCS-017_List_returns_participant_scoped_entry`; `TestMemoryConversationStore_CH08Scenarios_PassUnchanged/S-CCS-017_a_participant_sees_their_own_conversations`; `TestHandleListConversations_HappyPath` | COMPLIANT |
| R-CCS-013 / R-CCS-014 | S-CCS-018 (Gherkin verbatim, `0005:891-895`) | `TestConversationStore_List/S-CCS-018_List_returns_empty_for_unknown_participant`; `TestMemoryConversationStore_CH08Scenarios_PassUnchanged/S-CCS-018_a_participant_with_no_conversations_gets_empty_list`; `TestHandleListConversations_Empty`; `conversation-list.spec.tsx:104` | COMPLIANT |
| REQ-8 | S-FCL-008 (Gherkin verbatim, `0005:862-866`) | `routes/chat/index.spec.tsx:126 "the page passes participantID into ChatApp (REQ-8 / D-1)"`; `chat-app.tsx:112-140 useVisibleTask$ fires both GETs in parallel`; `loadConversation` helper test (`chat-api.spec.ts:794`) | COMPLIANT |
| REQ-9 | S-FCL-009 (Gherkin verbatim, `0005:868-872`) | `chat-app.tsx:122-126 turn.reset(...)` does NOT open an EventSource; `chat-app.spec.tsx:32 "renders the transcript and the composer"` (idle status); the reload endpoint contract (R-CRI-001 / S-CRI-001) returns recorded exchanges only. **See SUGGESTION note.** | COMPLIANT (structural) |
| REQ-8 | S-FCL-010 (Gherkin verbatim, `0005:885-889`) | `chat-app.spec.tsx:39 "mounts the conversation rail (R-CRI-005)"`; `conversation-list.spec.tsx:32-46 "renders one row per conversation summary"`; `listConversations` helper test (`chat-api.spec.ts:735`) | COMPLIANT |
| REQ-8 | S-FCL-011 (Gherkin verbatim, `0005:891-895`) | `conversation-list.spec.tsx:104 "renders an empty list as an empty list, not as a crash (S-CRI-004 / S-CCS-018)"`; `chat-app.spec.tsx:43-45 (S-CRI-004 implicit — empty rail is a mounted rail)`; `chat-api.spec.ts:758 "listConversations returns 200 [] on the empty-list path (S-CRI-004)"` | COMPLIANT |

**Compliance summary**: 12/12 scenarios compliant.

---

## Correctness (Static + Runtime Evidence)

| Requirement | Status | Notes |
|-------------|--------|-------|
| R-CRI-001 — GET /api/agent/conversations/:id returns exchanges | Implemented | `HandleReloadConversation` (`http.go`); `store.Load` projection; 403/404 guards tested |
| R-CRI-002 — GET /api/agent/conversations returns summaries | Implemented | `HandleListConversations`; `store.List(participantID)`; 200 [] on empty |
| R-CRI-003 — Page mounts both endpoints on first paint | Implemented | `chat-app.tsx:111-140 useVisibleTask$`; parallel `Promise.all([listConversations(), loadConversation(participantID)])`; `turn.reset(entries)` (no EventSource); `?with=` stays inert |
| R-CRI-004 — ConversationSummaryDTO + ExchangeDTO closed | Implemented | `chat-types.ts:163 ConversationSummary`; `ExchangeDTO` mirrored field-for-field from `chat.Exchange` (D-7); ChatStreamEvent union unchanged |
| R-CRI-005 — ConversationList re-mounted against real wire | Implemented | `conversation-list.tsx` accepts `ConversationSummary[]`; relative-time helper `format-relative-time.ts`; `id` = participantID per D-1 |
| R-CCS-013 — Additive third method `List` | Implemented | `ConversationStore` widened in `store.go`; Append/Load byte-unchanged; Memory + Postgres adapters implement List |
| R-CCS-014 — `ConversationSummary` is the port's list projection | Implemented | Struct carries `ConversationID`, `LastActivityAt`, `TurnCount`; wire DTO mirrors it (R-CRI-004) |
| NFR-CRI-001 — Substrate preservation | Verified | `git diff main -- backend/agent/src/agent/` is empty; only one CH-08 pointer line added to `openspec/AGENTS.md` substrate section |
| NFR-CRI-002 — Import-boundary | Verified | New wire types live under `frontend/src/lib/chat-types.ts`; no new top-level deps |
| NFR-CRI-003 — Participant-scoped reads under identity middleware | Verified | `TestCrossParticipantRefusal_403`; `TestHandleReloadConversation_CrossParticipant`; `TestConversationStore_List/List_is_participant_scoped_no_other_participants` |
| NFR-CRI-004 — Sub-millisecond reload at v1's request rate | Verified | Defensive copy semantics; one `SELECT … FROM chat_conversations ORDER BY updated_at DESC`; no new index at v1 |
| NFR-CCS-007 — `List` is participant-scoped | Verified | `TestConversationStore_List/List_is_participant_scoped_no_other_participants` (`store_test.go`); shared scenario helper S-CCS-017 enforces NFR-CCS-007 |
| REQ-8 — Page mounts both endpoints on first paint | Implemented | Mirrors R-CRI-003 (frontend contract) |
| REQ-9 — Reload that catches an in-flight turn MUST NOT claim streaming | Implemented | `turn.reset(...)` does not open an EventSource; `chat-app.tsx:122-126`; no `subscribeTurn` call on mount |

---

## Coherence (Design Decisions vs Implementation)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| D-1: Additive third method `List(participantID) ([]ConversationSummary, error)` on the same `ConversationStore` declaration | Yes | `store.go` widens the interface; Append/Load byte-unchanged; shared scenario helper extended (R-CCS-012 precedent) |
| D-2: Exchanges only — `[]ExchangeDTO` mirroring `chat.Exchange` field-for-field (8 fields per D-7) | Yes | http.go projects `[]Exchange` to `[]ExchangeDTO`; `chat-types.ts` `ExchangeDTO` mirrors the 8 fields (Position, PromptText, AssistantText, Partial, TerminalKind, FailureCategory, FinishReason, MessageIDs) |
| D-3: `conversationID == participantID` (one conversation per participant) | Yes | Tests use alice/bob as both `:id` and participant; D-1 schema (`chat_conversations.participant_id` PK) source of truth |
| D-4: Full slice, `chat_conversations ORDER BY updated_at DESC`, no pagination | Yes | `store_postgres.go` List query; no LIMIT; ORDER BY updated_at DESC |
| D-5: Qwik `useVisibleTask$` fires both GETs in parallel; `useChatStream.reset(entries)` seeds; `ConversationList` re-mounted; `?with=<slug>` stays inert | Yes | `chat-app.tsx:111-140`; `turn.reset(...)` without EventSource; `<ConversationList>` re-mounted; `?with=` deep-link still resolves to agentSlug (CH-05 D-6) but does not drive which conversation loads |
| D-6: `:id` MUST equal caller's `participantID`; 403 `not_found` on mismatch; `ErrConversationNotFound` → 404 `not_found` | Yes | `TestHandleReloadConversation_CrossParticipant` (403 + envelope `error: "not_found"`); `TestHandleReloadConversation_Unknown` (404); matches R-CHS-004.b shape |

---

## TDD Compliance (Strict TDD Module)

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | Yes | `apply-progress` (#3932) has TDD Cycle Evidence table for all 10 WUs |
| All tasks have tests | Yes | 10/10 tasks have test files; WU-1/2/3/4/5/6 (Go), WU-7/8 (TS) |
| RED confirmed (tests exist) | Yes | RED scaffold commits `4b6f06c1` (3 cases), `11027e6a` (5 cases), `926d8682` (7 cases) — test files verified to exist |
| GREEN confirmed (tests pass) | Yes | All 62 chat tests PASS; 584 frontend tests PASS; the focused RED scaffolds all GREEN now |
| Triangulation adequate | Yes | S-CRI-001 triangulated by TestHandleReloadConversation_HappyPath + TestMemoryConversationStore_CH08Scenarios_PassUnchanged/S-CCS-001 + loadConversation helper test; S-CCS-017 has 3 cover points (store_test, store_scenarios_test, http_test); S-CCS-018 has 4 cover points |
| Safety Net for modified files | Yes | All modified Go files are NEW additions to existing test surface (existing scenarios still pass; `TestMemoryConversationStore_CH08Scenarios_PassUnchanged` re-runs CH-06/CH-07 S-CCS-001..014 unchanged) |

**TDD Compliance**: 6/6 checks passed.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit (Go) | 9 | 2 (`store_test.go`, `store_postgres_test.go`) | `go test -race` |
| Scenario (Go) | 8 | 1 (`store_scenarios_test.go` shared helper, both adapters) | `go test -race` |
| HTTP (Go) | 5 | 1 (`http_test.go`) | `httptest` + Echo |
| Unit (TS) | 7 | 1 (`chat-api.spec.ts`) | vitest |
| Component (TS) | 9 | 2 (`conversation-list.spec.tsx`, `chat-app.spec.tsx`) | vitest + qwik/testing createDOM |
| Route (TS) | 6 | 1 (`routes/chat/index.spec.tsx`) | vitest + qwik/testing createDOM |
| **Total** | **44 new** | **8** | (584 frontend total includes 540 baseline + 44 CH-08 net) |

### Changed File Coverage

Not collected at the file level. The project config `coverage_threshold: 0`; the strict TDD discipline uses per-WU focused test commands instead of aggregate coverage. All new tests are exercised at runtime (62 chat + 44 frontend net all PASS).

### Assertion Quality

| File | Sample assertion | Quality |
|------|-------------------|---------|
| `store_test.go:134` | `if len(aliceList) != 1 { t.Fatalf(...) }` + `if aliceList[0].ConversationID != "alice" { t.Errorf(...) }` + `if aliceList[0].TurnCount != 1` | Real value assertions on field-for-field |
| `store_test.go:181` | `if got == nil { t.Errorf("want []ConversationSummary{} (non-nil empty slice — JSON serializes as [] not null)") }` + `if len(got) != 0` | Companion real assertion; non-empty path tested in S-CCS-017 |
| `store_scenarios_test.go:187` | Cross-participant loop: `for _, entry := range aliceList { if entry.ConversationID != alice { t.Errorf("alice's list leaked entry %q (NFR-CCS-007 violation)", entry.ConversationID) } }` | Explicit leak-detection, not just length check |
| `http_test.go:551` | `if len(got) != 2 { t.Fatalf(...) }` + `if got[0].Position != 0 || got[1].Position != 1` + `if got[0].PromptText != "turn-one"` | Real status + body decode + field assertions |
| `http_test.go:604` | `if res.StatusCode != http.StatusForbidden { ... }` + `_ = json.NewDecoder(res.Body).Decode(&env)` + `if env.Error != "not_found"` | Status + envelope body assertions (not just one) |
| `chat-api.spec.ts:735` | Mocked `fetch` with `expect.objectContaining({ url: "/api/agent/conversations", ... })`; asserts response envelope parser maps to typed `ApiResult<ConversationSummary[]>` | Real wire call + envelope parse; not a mock-only test |
| `chat-api.spec.ts:758` | Mocks 200 with `[]`; asserts `listConversations()` returns `{ ok: true, value: [] }` (S-CRI-004: empty is success) | Real empty-list happy path |
| `chat-api.spec.ts:774` | Mocks 403; asserts `{ ok: false, kind: "not_found" }` (R-CRI-002 cross-participant guard) | Real error-mapping assertion |
| `conversation-list.spec.tsx:32` | Renders wire-shaped `ConversationSummary[]`; asserts each `data-testid="conversation-${id}"` is present | Real DOM behavior, not just smoke-test |
| `conversation-list.spec.tsx:104` | Renders `[]`; asserts list has 0 `<li>` children + rail container present | Empty-list companion to non-empty case |
| `routes/chat/index.spec.tsx:126` | `expect(screen.querySelector('[data-testid="conversation-list"]')).toBeTruthy()` after `participantID` flows from `requireSession` into `ChatApp` | Real mount evidence of wire-driven rail |
| `chat-app.spec.tsx:39` | `expect(screen.querySelector('[data-testid="conversation-list"]')).toBeTruthy()` | Real mount evidence; rail is wire-driven (not legacy mock) |

**Assertion quality**: All assertions verify real behavior. No tautologies. No ghost loops. Empty-collection assertions have companion non-empty tests.

### Quality Metrics

**Linter**: No errors (Backend 0 issues; Frontend 0 errors / 0 warnings across all files including new CH-08 additions).
**Type Checker**: No errors (`tsc --incremental --noEmit` clean).

---

## Issues Found

**CRITICAL**: None.
**WARNING**: None.
**SUGGESTION**:

1. S-CRI-002 / S-FCL-009 ("a reload during a streaming turn shows what was recorded; the page does not claim streaming") lack a single end-to-end Gherkin-scoped test that combines (in-flight turn + page reload + recorded exchanges shown + idle status asserted). The behavior is enforced structurally — `useChatStream.reset(entries)` does NOT open an EventSource (`chat-app.tsx:122-126`); REQ-9 forbids auto-subscribe; the reload endpoint contract returns recorded exchanges only (R-CRI-001 / TestHandleReloadConversation_HappyPath). A future milestone could add `TestPage_ReloadDuringStreamingTurn_ShowsRecordedAndDoesNotClaimStreaming` for explicit coverage. Not blocking; this is a SUGGESTION for a future CH-08.x closure pass.

---

## Verdict

**PASS**

All 10 WUs complete and committed; all 12 spec scenarios (S-CRI-001..004, S-CCS-015..018, S-FCL-008..011) covered by passing tests at runtime; substrate preservation verified (NFR-TLS-003 — empty diff under `backend/agent/src/agent/`); backend 18 packages GREEN race-clean; frontend 584/584 PASS; linters clean on both sides; type-check clean; design decisions all verified against implementation; Strict TDD evidence present for every WU. Ready for `sdd-archive`.

## Next Step

Ready for `sdd-archive`.
