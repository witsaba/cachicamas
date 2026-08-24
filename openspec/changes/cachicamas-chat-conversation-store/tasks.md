# Tasks: CH-06 — Define the conversation store port and its in-memory adapter

Change `cachicamas-chat-conversation-store` · Node **CH-06.1** `[leaf]` (`0005:724-751`) + **CH-06.2** `[leaf]` (`0005:753-778`) — design's six-commit plan collapsed into one PR (`exception-ok`, user-pre-authorised 1000-line ceiling, extendable). All `path:line` cites re-resolved in this worktree at `e3c717a4`. Strict TDD per `openspec/AGENTS.md` (RED-GREEN-REFACTOR); every implementation task is preceded by its RED. Evidence gate verbatim from `proposal.md` *Evidence gate* + *Verification gates*.

**Worktree**: `cachicamas-worktrees/feat-chat-conversation-store-ch06` · **Base**: `e3c717a4` (PR #195, CH-05) · **Branch**: `feat/chat-conversation-store-ch06` · **Delivery**: `exception-ok` (single PR + `size:exception` pre-granted).

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | ~863 code-only (proposal § Review Workload Forecast) — comfortable against the user-pre-granted 1000-line ceiling |
| 400-line budget risk | Low (redefined: this PR's effective ceiling is the user's pre-granted 1000 lines; ~863 sits ~14% under it) |
| Chained PRs recommended | No — single PR with `size:exception` |
| Delivery strategy | `exception-ok` (user pre-authorised) |
| Chain strategy | `size-exception` (the 6 WUs fold into ONE PR; commit slicing only) |

The project's default review budget is 400 lines (`sdd-phase-common.md` § E). The user pre-authorised a 1000-line ceiling for this PR with extension permitted. The code-only diff (~863 LOC) sits comfortably under the user's ceiling; against the default 400 the risk would be High, against the granted 1000 it is Low. Be explicit in review: this PR's effective review budget is the user's pre-granted 1000 lines, not the default 400.

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low
```

### Suggested Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| WU-1 | RED scaffold — empty port + 9 RED cases | commit 1 | `cd backend/agent && go test -race -count=1 -run TestConversationStore ./src/chat/...` (RED) | N/A — no harness yet | revert commit 1; tree returns to `e3c717a4` |
| WU-2 | GREEN port + adapter — 3 micro-tests green | commit 2 | `cd backend/agent && go test -race -count=1 -run "TestConversationStore_AppendPersists\|TestConversationStore_LoadReturnsSliceInOrder\|TestConversationStore_LoadUnknownReturnsErrConversationNotFound" ./src/chat/...` | N/A (in-memory) | revert commit 2 alone; 6 `Send`-driven scenarios fall back to "no store wired" |
| WU-3 | GREEN wire terminal site → store — S-CCS-001/002/003 green | commit 3 | `cd backend/agent && go test -race -count=1 -run "TestConversationStore_RecordsTwoTurnsInOrder\|TestConversationStore_CancelledTurnCarriesPartialText\|TestConversationStore_FailedTurnAppendsLater" ./src/chat/...` | N/A | revert commit 3 alone; reload-path scenarios stay red, port is intact |
| WU-4 | GREEN reload path — S-CCS-004 green | commit 4 | `cd backend/agent && go test -race -count=1 -run TestConversationStore_LoadSeedsHistoryForThirdTurn ./src/chat/...` | N/A (factory closure consults in-memory store) | revert commit 4 alone; WU-3 scenarios stay green |
| WU-5 | GREEN identifier survival + unknown-refused — S-CCS-005/006/009 green | commit 5 | `cd backend/agent && go test -race -count=1 -run "TestConversationStore_IdentifierMintedDuringTurnSurvivesReload\|TestConversationStore_LoadUnknownRefused\|TestConversationStore_LoadUnknownReturnsErrConversationNotFound" ./src/chat/...` | N/A | revert commit 5 alone; reload-side + identifier-side scenarios fall back |
| WU-6 | Spec promotion + doc 0005 bookkeeping | commit 6 | `cd backend/agent && make test && make lint && make build/chat` (full gate) | proven by `git log --oneline feat/chat-conversation-store-ch06 ^e3c717a4` showing 6 commits + `git diff --stat e3c717a4 HEAD -- openspec/specs/` showing exactly one new spec | revert commit 6 alone; code stays |

## Phase 1: WU-1 — RED scaffold (NODE: CH-06.1)

- [ ] **1.1** Create `backend/agent/src/chat/store.go` (new) — `package chat`, imports `errors` + `sync`, empty `type ConversationStore interface{}`, doc comment *"stub for strict TDD red phase"*. **Tag:** RED.
- [ ] **1.2** Create `backend/agent/src/chat/store_test.go` (new, `package chat_test`) — 9 sub-tests in `t.Run("S-CCS-NNN …", …)` format: 3 micro-tests (`TestConversationStore_AppendPersists` S-CCS-007, `TestConversationStore_LoadReturnsSliceInOrder` S-CCS-008, `TestConversationStore_LoadUnknownReturnsErrConversationNotFound` S-CCS-009) assert against `MemoryConversationStore` methods that **don't exist yet** — RED by compile; 6 `Send`-driven scenarios (`S-CCS-001..006`) transcribed from `0005:728-775` — RED by `Config.Store` not existing. **Tag:** RED.
- [ ] **1.3** Run `cd backend/agent && go test -race -count=1 -run TestConversationStore ./src/chat/...` — confirm RED: 6 `Send`-driven fail by compile error, 3 micro-tests fail by assertion with "method Append on interface{}". **Tag:** RED.
- [ ] **1.4** Run `cd backend/agent && make lint` — confirm the stub's empty interface does not break lints. **Tag:** DOC.

## Phase 2: WU-2 — GREEN port + adapter (NODE: CH-06.1)

- [ ] **2.1** Fill `backend/agent/src/chat/store.go` with: imports `errors` + `sync` + `agent` + `ai` only (D-4); `Exchange` struct (8 fields per D-7: `Position int`, `PromptText string`, `AssistantText string`, `Partial bool`, `TerminalKind TerminalKind`, `FailureCategory ai.FailureCategory`, `FinishReason *ai.FinishReason`, `MessageIDs []string`); `TerminalKind` int enum (`TerminalKindCompleted`/`Cancelled`/`Failed`) with `String()`; `ConversationStore` interface (`Append(participantID string, exchange Exchange) error` + `Load(participantID string) ([]Exchange, error)`); sentinels `ErrConversationNotFound` + `ErrNilStore` + `ErrEmptyParticipantID`; `MemoryConversationStore` with `sync.Mutex` + `map[string][]Exchange`; `NewMemoryConversationStore()` constructor; `Append` (lock + slice append + store copy) + `Load` (lock + defensive slice copy). **Tag:** GREEN.
- [ ] **2.2** Add `ExchangesToHistory(exchanges []Exchange) (*agent.History, error)` helper — maps each `Exchange` to one user + one assistant `ai.Message` (assistant's `MessageID` on part metadata when present); calls `agent.NewSeededHistory(messages)`. **Tag:** GREEN.
- [ ] **2.3** Add `buildTerminalExchange(prompt string, runEnd agent.RunEnd, haveRunEnd bool, res runResult, assistantText string, messageIDs []string) Exchange` (package-private; called from `projection.go`). **Tag:** GREEN.
- [ ] **2.4** Run `cd backend/agent && go test -race -count=1 -run "TestConversationStore_AppendPersists|TestConversationStore_LoadReturnsSliceInOrder|TestConversationStore_LoadUnknownReturnsErrConversationNotFound" ./src/chat/...` — confirm GREEN (3 micro-tests pass; 6 `Send`-driven stay RED). **Tag:** GREEN.

## Phase 3: WU-3 — GREEN wire-side append (NODE: CH-06.2)

- [ ] **3.1** Modify `backend/agent/src/chat/conversation.go` — add `Store ConversationStore`, `ParticipantID string`, `InitialHistory *agent.History` to `Config`; add `store ConversationStore`, `participantID string` to `Conversation`; add `ErrNilStore` + `ErrEmptyParticipantID` after `ErrNilProvider` (line 24); in `NewConversation` add `if cfg.Store == nil { return nil, ErrNilStore }` + `if cfg.ParticipantID == "" { return nil, ErrEmptyParticipantID }`; thread `cfg.InitialHistory` to harness (`History: cfg.InitialHistory` when non-nil, else `agent.NewHistory()` at lines 62-69). **Tag:** GREEN.
- [ ] **3.2** Modify `backend/agent/src/chat/projection.go` (lines 33-73) — track `assistantText string` (accumulate `delta.Fragment()` at MessageDeltaText, line 49) and `messageIDs []string` (push `start.MessageID().String()` at MessageStartText, line 45) alongside `msgIndex`; insert between line 68 (`out <- terminalWireEvent(...)`) and line 70 (`c.mu.Lock()`): `exchange := buildTerminalExchange(prompt, runEnd, haveRunEnd, res, assistantText, messageIDs); c.store.Append(c.participantID, exchange)`; pass `prompt string` to `c.project`. **Tag:** GREEN.
- [ ] **3.3** Run `cd backend/agent && go test -race -count=1 -run "TestConversationStore_RecordsTwoTurnsInOrder|TestConversationStore_CancelledTurnCarriesPartialText|TestConversationStore_FailedTurnAppendsLater" ./src/chat/...` — confirm GREEN (S-CCS-001/002/003 pass). **Tag:** GREEN.
- [ ] **3.4** Run `cd backend/agent && make test` — confirm the existing CH-02/CH-03 suite stays green (every existing test populates `Config{Store: …, ParticipantID: …}` per the new requirement, plus CH-04's `cmd/chat/main.go`). **Tag:** GREEN.

## Phase 4: WU-4 + WU-5 — GREEN reload + identifier + refused (NODE: CH-06.2)

- [ ] **4.1** Modify `backend/agent/src/cmd/chat/main.go` factory closure (line ~165) — declare `chatStore := chat.NewMemoryConversationStore()` near line 152 (CH-07's swap point); the closure consults `chatStore.Load(participantID)`, maps `errors.Is(err, chat.ErrConversationNotFound)` to `exchanges := []chat.Exchange{}` else surfaces the err; calls `chat.ExchangesToHistory(exchanges)` and threads through `Config.InitialHistory`. **Tag:** GREEN.
- [ ] **4.2** Run `cd backend/agent && go test -race -count=1 -run TestConversationStore_LoadSeedsHistoryForThirdTurn ./src/chat/...` — confirm GREEN (S-CCS-004: `provider.Requests()[2]` carries both earlier exchanges in original order). **Tag:** GREEN.
- [ ] **4.3** Verify `messageIDs` carries forward through `Exchange.MessageIDs` — assert reload-rebuilt harness emits the same wire-side IDs in `TestConversationStore_IdentifierMintedDuringTurnSurvivesReload` (S-CCS-005 GREEN). **Tag:** GREEN.
- [ ] **4.4** Verify refusal path — `TestConversationStore_LoadUnknownRefused` (S-CCS-006): drive `Load("never-existed")`, assert (a) error == `ErrConversationNotFound`, (b) no side-effect entry created, (c) follow-up `Append` under different `participantID` does not collide. **Tag:** GREEN.
- [ ] **4.5** Verify micro-test version — `TestConversationStore_LoadUnknownReturnsErrConversationNotFound` (S-CCS-009) asserts the same against the in-memory adapter directly. **Tag:** GREEN.

## Phase 5: WU-6 — Evidence gate + spec promotion + doc 0005 (NODE: BOTH)

- [ ] **5.1** Run `cd backend/agent && make test` (full backend suite, race-flagged, uncached) — confirm green. **Tag:** GREEN.
- [ ] **5.2** Run `cd backend/agent && make lint` — confirm clean (`go vet` + `golangci-lint` v2.9.0). **Tag:** DOC.
- [ ] **5.3** Run `cd backend/agent && make build/chat` — confirm `./bin/chat` binary produced. **Tag:** DOC.
- [ ] **5.4** Run `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/import_boundary_test.go` — expect empty (NFR-CCS-002 substrate preservation). **Tag:** DOC.
- [ ] **5.5** Run `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/` — expect the ten-file substrate list empty (NFR-TLS-003). **Tag:** DOC.
- [ ] **5.6** Run `git diff --stat e3c717a4 HEAD -- frontend/` — expect empty (CH-05 evidence remains valid). **Tag:** DOC.
- [ ] **5.7** Copy `openspec/changes/cachicamas-chat-conversation-store/specs/chat-conversation-store/spec.md` to `openspec/specs/chat-conversation-store/spec.md` (archive-time promotion; the apply-progress commits record this). **Tag:** DOC.
- [ ] **5.8** Tick `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md`: lines `:992` (CH-06.1), `:993` (CH-06.2), `:1064` (close-by mapping); annotate `:1044` (Register 3 row → "closed by CH-06"); update line `:3` status to **7 of 12**. **Tag:** DOC.
- [ ] **5.9** Open the single PR (6 commits folded) with title `feat(chat): CH-06 — define the conversation store port and its in-memory adapter`; body cross-links commits to WUs and `0005:711-778`. **Tag:** DOC.
- [ ] **5.10** Save engram apply-progress at `sdd/cachicamas-chat-conversation-store/apply-progress` (type=`architecture`, `capture_prompt=false`) recording per-WU RED→GREEN discipline, commit SHAs, and the PR link. **Tag:** DOC.

## Evidence (record per-task transcripts during apply)

| Check | Command | Expected |
|---|---|---|
| WU-1 RED | `cd backend/agent && go test -race -count=1 -run TestConversationStore ./src/chat/...` | compile/assertion failures on 9 cases |
| WU-2 micro | `cd backend/agent && go test -race -count=1 -run "TestConversationStore_AppendPersists\|…LoadReturnsSliceInOrder\|…LoadUnknownReturnsErrConversationNotFound" ./src/chat/...` | GREEN on 3 micro-tests; 6 Send-driven still RED |
| WU-3 record | `cd backend/agent && go test -race -count=1 -run "TestConversationStore_RecordsTwoTurnsInOrder\|…CancelledTurnCarriesPartialText\|…FailedTurnAppendsLater" ./src/chat/...` | GREEN on S-CCS-001/002/003 |
| WU-3 full | `cd backend/agent && make test` | full suite green, uncached |
| WU-4 reload | `cd backend/agent && go test -race -count=1 -run TestConversationStore_LoadSeedsHistoryForThirdTurn ./src/chat/...` | GREEN on S-CCS-004 |
| WU-5 id+refuse | `cd backend/agent && go test -race -count=1 -run "TestConversationStore_IdentifierMintedDuringTurnSurvivesReload\|TestConversationStore_LoadUnknownRefused" ./src/chat/...` | GREEN on S-CCS-005/006 |
| WU-6 lint | `cd backend/agent && make lint` | exit 0 |
| WU-6 build | `cd backend/agent && make build/chat` | `./bin/chat` produced |
| WU-6 substrate | `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/` | empty (NFR-TLS-003) |
| WU-6 import-boundary | `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/import_boundary_test.go` | empty (NFR-CCS-002) |
| WU-6 frontend | `git diff --stat e3c717a4 HEAD -- frontend/` | empty (CH-05 evidence preserved) |
| WU-6 spec delta | `git diff --stat e3c717a4 HEAD -- openspec/specs/` | exactly one new file: `chat-conversation-store/spec.md` |
| WU-6 scenario count | `grep -c "S-CCS-" backend/agent/src/chat/store_test.go` | ≥ 9 |

## Commit SHAs (work-unit shape — single PR)

| WU | Subject prefix |
|---|---|
| WU-1 | `test(chat): scaffold store port + 9 RED cases` |
| WU-2 | `feat(chat): ConversationStore port + MemoryConversationStore adapter (GREEN micro-tests)` |
| WU-3 | `feat(chat): wire terminal site to ConversationStore.Append (GREEN record-side)` |
| WU-4 | `feat(chat): reload path consults store in factory closure (GREEN reload)` |
| WU-5 | `test(chat): identifier survival + unknown-refused (GREEN tail)` |
| WU-6 | `docs(0005): CH-06 shipped + chat-conversation-store spec promoted` |

## Out-of-scope reminder (the seam fence, design § *Out of scope carryover*)

Implementation MUST NOT touch: `backend/agent/src/agent/**` (NFR-TLS-003 ten-file substrate); `backend/agent/src/agent/import_boundary_test.go` (NFR-CCS-002); `backend/agent/src/agent/history.go` (D-3 — `NewSeededHistory` consumed unchanged); `frontend/**` (CH-05 owns; CH-08.1 mounts the reload endpoint); `backend/agent/src/chat/{http.go,eventsource.go,wire.go,identity.go,registry.go,doc.go,errortext.go}` (CH-03 frozen wire + registry); `openspec/specs/chat-archetype-contract/spec.md` or `openspec/specs/chat-package-boundary/spec.md` (D-5 — no amendment); `docker-compose.yaml`, `infra/`, `backend/database_administrator/**`, `backend/workspace_syncer/**`. Any commit that widens scope fails review.

## Coverage — every scenario maps to a closing task

| Scenario | Source | Closing task(s) |
|---|---|---|
| S-CCS-001 two turns in order | `0005:731-735` | 1.2 (RED), 3.1–3.3 (GREEN) |
| S-CCS-002 cancelled turn partial | `0005:737-741` | 1.2 (RED), 2.1 (D-7 fields), 3.2 (assistantText), 3.3 (GREEN) |
| S-CCS-003 failed turn appends later | `0005:743-747` | 1.2 (RED), 2.1 (D-7 fields), 3.2 (FailureCategory), 3.3 (GREEN) |
| S-CCS-004 reload continues transcript | `0005:760-763` | 1.2 (RED), 2.2 (ExchangesToHistory), 4.1 (factory consult), 4.2 (GREEN) |
| S-CCS-005 identifier survives reload | `0005:765-768` | 1.2 (RED), 3.2 (messageIDs), 4.3 (GREEN) |
| S-CCS-006 unknown reload refused | `0005:770-774` | 1.2 (RED), 4.1 (ErrConversationNotFound mapping), 4.4 (GREEN) |
| S-CCS-007 Append persists | spec R-CCS-002 | 1.2 (RED), 2.1, 2.4 (GREEN) |
| S-CCS-008 Load returns in order | spec R-CCS-002 | 1.2 (RED), 2.1, 2.4 (GREEN) |
| S-CCS-009 Load unknown returns sentinel | spec R-CCS-003 | 1.2 (RED), 2.1, 2.4 + 4.5 (GREEN) |
| NFR-CCS-001 race-free under `-race` | spec NFR | 5.1 (`make test` includes `-race`) |
| NFR-CCS-002 stdlib-only imports | spec NFR | 2.1 (import set), 5.4 (import-boundary test empty) |
| NFR-CCS-003 substrate preservation | spec NFR | 5.5 (`git diff` ten-file list empty) |
| Doc 0005 bookkeeping | proposal § Doc 0005 | 5.8 |
| Spec promotion | repo convention | 5.7 |
| Single PR open | repo convention | 5.9 |
| Apply-progress engram | repo convention | 5.10 |
