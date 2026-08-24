# Archive Report — `cachicamas-chat-conversation-store` (CH-06)

> **Status**: ready-to-merge
> **Worktree**: `cachicamas-worktrees/feat-chat-conversation-store-ch06`
> **Branch**: `feat/chat-conversation-store-ch06` against `origin/main` @ `e3c717a4`
> **Date**: 2026-08-24
> **Final commit**: this archive commit on PR #196 (SHA retrievable via `git log -1 --format='%H'` on `feat/chat-conversation-store-ch06` after push). The archive commit's working SHA is recorded in the engram `sdd/cachicamas-chat-conversation-store/archive-report` observation's sync_id, captured by `sdd-archive` at push time.

## Outcome

CH-06 closes the conversation-durability half of doc 0005's Wave 1 chat archetypes. The charter (`0005:711-722`) called for a `ConversationStore` port with two leaves — CH-06.1 (`0005:724-751`) the port + in-memory adapter, CH-06.2 (`0005:753-778`) the reload path through the same port — and that port is now realized end-to-end on this branch. The closure retires four open items on the record: **R-04** (define your own ports), **R-16** (durable state independent of transport), **inconsistency register row 3** (home-directory sessions are not a layer rule), and **research finding 5** (the stream is not the store of record). Conversation durability is no longer carried by the scene (the chat HTTP handler) — it is a hexagonal port the archetype owns (`ConversationStore` in `backend/agent/src/chat/store.go`) with a single v1 adapter (`MemoryConversationStore`) sitting behind it; the composition root (`backend/agent/src/cmd/chat/main.go`) is the only place that knows which adapter is in use, and that one line is CH-07's swap point. Evidence: code-only diff ~1029 LOC (29 lines above the user-pre-granted 1000-line ceiling, absorbed), full backend suite green (`16/16 packages`, `-race` clean, wall-clock 2:50.44), `make lint` clean (0 issues), `make build/chat` produces `./bin/chat`, 10/10 substrate/scope fence gates empty, 9/9 spec scenarios COMPLIANT, 8/8 design decisions coherent, 27/27 tasks `[x]`, verify-report verdict PASS (0 CRITICAL, 0 WARNING, 4 SUGGESTION informational). Two leaves land together: CH-06.1 the port + in-memory adapter; CH-06.2 reload through the port.

## What shipped

### Code

- **`backend/agent/src/chat/store.go`** (231 lines, new) — the `ConversationStore` port + first adapter. Defines: `Exchange` struct (8 fields per D-7: `Position int`, `PromptText string`, `AssistantText string`, `Partial bool`, `TerminalKind TerminalKind`, `FailureCategory ai.FailureCategory`, `FinishReason *ai.FinishReason`, `MessageIDs []string`); `TerminalKind` int enum (Completed / Cancelled / Failed) with `String()`; the `ConversationStore` interface (two methods only: `Append(participantID string, exchange Exchange) error` and `Load(participantID string) ([]Exchange, error)`); sentinel errors (`ErrConversationNotFound`, `ErrNilStore`, `ErrEmptyParticipantID`); the `MemoryConversationStore` v1 adapter guarded by `sync.Mutex`; the `NewMemoryConversationStore()` constructor; the `ExchangesToHistory` helper that round-trips recorded exchanges back into `*agent.History`; the `buildTerminalExchange` package-private helper called from `projection.go`. Imports only `errors`, `sync`, `agent`, `ai` — stdlib-only for the port's adapter contract (NFR-CCS-002).
- **`backend/agent/src/chat/store_test.go`** (610 lines, new) — the 9 sub-tests in `t.Run("S-CCS-NNN …", …)` format, each independently runnable: `TestConversationStore_AppendPersists` (S-CCS-007), `TestConversationStore_LoadReturnsSliceInOrder` (S-CCS-008), `TestConversationStore_LoadUnknownReturnsErrConversationNotFound` (S-CCS-009), plus the six `Send`-driven scenarios transcribed from `0005:728-775` (`TestConversationStore_RecordsTwoTurnsInOrder` S-CCS-001, `TestConversationStore_CancelledTurnCarriesPartialText` S-CCS-002, `TestConversationStore_FailedTurnAppendsLater` S-CCS-003, `TestConversationStore_LoadSeedsHistoryForThirdTurn` S-CCS-004, `TestConversationStore_IdentifierMintedDuringTurnSurvivesReload` S-CCS-005, `TestConversationStore_LoadUnknownRefused` S-CCS-006).
- **`backend/agent/src/chat/conversation.go`** (+72 lines, modified) — `Config.Store ConversationStore`, `Config.ParticipantID string`, `Config.InitialHistory *agent.History` added; `Conversation` struct gains `store` and `participantID` fields; `NewConversation` now requires both `Store` (else `ErrNilStore`) and a non-empty `ParticipantID` (else `ErrEmptyParticipantID`); `cfg.InitialHistory` is the reload seam (threaded to harness directly when non-nil, else fall back to `agent.NewHistory()`).
- **`backend/agent/src/chat/projection.go`** (+85 lines, modified) — `assistantText` accumulates `delta.Fragment()` at `MessageDeltaText`; `messageIDs` pushes `start.MessageID().String()` at `MessageStartText`; between the terminal wire event and the `inFlight = false` clear, the terminal exchange is built via `buildTerminalExchange` and recorded via `c.store.Append(c.participantID, exchange)` — *before* the in-flight flag drops, per D-6.
- **`backend/agent/src/cmd/chat/main.go`** (+22 lines, modified) — `chatStore := chat.NewMemoryConversationStore()` declared near line 152 (CH-07's swap point); the factory closure consults `chatStore.Load(participantID)`, maps `errors.Is(err, chat.ErrConversationNotFound)` to a fresh slice, threads `chat.ExchangesToHistory(exchanges)` as `Config.InitialHistory`.
- **Test-fixture carry** — the seven pre-existing chat test files now pass a `Store` + `ParticipantID` to `Config`: `cancel_test.go` (+5), `chat_test.go` (+7), `conversation_test.go` (+7), `failure_test.go` (+5), `http_test.go` (+13), `registry_test.go` (+7), `cmd/chat/main_test.go` (+5). Net test-fixture churn: ~49 lines of additions, no semantic edits.

### Spec

- **`openspec/specs/chat-conversation-store/spec.md`** (NEW — promoted from `openspec/changes/.../specs/chat-conversation-store/spec.md`).
- 10 requirements (`R-CCS-001..010`), 9 scenarios (`S-CCS-001..009`), 4 NFRs (`NFR-CCS-001..004`), `CCS` prefix verified collision-free against the `chat-frontent-layer1`, `chat-archetype-contract`, and `chat-package-boundary` registered scopes.
- No promoted spec is amended; this is a brand-new capability.

### Doc 0005 bookkeeping

- Line `:3` — status updated to **7 of 12** (was 6 of 12).
- Lines `:992` and `:993` — CH-06.1, CH-06.2 ticked.
- Line `:1044` — register row 3 annotated: "closed by CH-06 (this PR); CH-07 carries the postgres adapter".
- Line `:1064` — close-by mapping annotated: "CH-06.1, CH-06.2 → R-04, R-16, research 5, register 3".

## Evidence gate

Per `0005:719` acceptance:

| Check | Command | Result |
|---|---|---|
| All 6 Gherkin scenarios pass | `cd backend/agent && go test -race -count=1 -run "TestConversationStore_RecordsTwoTurnsInOrder\|TestConversationStore_CancelledTurnCarriesPartialText\|TestConversationStore_FailedTurnAppendsLater\|TestConversationStore_LoadSeedsHistoryForThirdTurn\|TestConversationStore_IdentifierMintedDuringTurnSurvivesReload\|TestConversationStore_LoadUnknownRefused" ./src/chat/...` | 6/6 PASS (each scenario's `t.Run("S-CCS-…")` passed at runtime) |
| 3 in-memory micro-tests | `cd backend/agent && go test -race -count=1 -run "TestConversationStore_AppendPersists\|TestConversationStore_LoadReturnsSliceInOrder\|TestConversationStore_LoadUnknownReturnsErrConversationNotFound" ./src/chat/...` | 3/3 PASS |
| Full backend suite | `cd backend/agent && make test` (`go test -race -v ./...`) | GREEN, 16/16 packages, -race clean, wall-clock 2:50.44 |
| Linter | `cd backend/agent && make lint` | 0 errors / 0 warnings (`golangci-lint` v2.9.0) |
| Build | `cd backend/agent && make build/chat` | `./bin/chat` produced (23,303,730 bytes) |
| Import boundary | `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/import_boundary_test.go` | empty (NFR-CCS-002 satisfied) |
| 10-file substrate | `git diff --stat e3c717a4 HEAD -- backend/agent/src/agent/` | empty (NFR-TLS-003 satisfied; ten files byte-unchanged) |
| Frontend untouched | `git diff --stat e3c717a4 HEAD -- frontend/` | empty (CH-05 evidence preserved) |
| Spec promotion | `git diff --stat e3c717a4 HEAD -- openspec/specs/` | exactly one new file: `chat-conversation-store/spec.md` (194 LOC) |
| Chat-wire substrate | `git diff --stat e3c717a4 HEAD -- backend/agent/src/chat/{http.go,eventsource.go,wire.go,identity.go,registry.go,doc.go,errortext.go}` | empty (CH-03 frozen wire preserved) |
| Promote-target spec amendments | `git diff --stat e3c717a4 HEAD -- openspec/specs/chat-archetype-contract/spec.md openspec/specs/chat-package-boundary/spec.md` | empty (D-5 satisfied) |

## Work-unit commit map

| WU | Commit | Subject |
|---|---|---|
| WU-1 | `cdd6c7f9` | `test(chat): scaffold store port + 9 RED cases` |
| WU-2 | `8d6aa1f3` | `feat(chat): ConversationStore port + MemoryConversationStore adapter (GREEN micro-tests)` |
| WU-3 | `ec00398f` | `feat(chat): wire terminal site to ConversationStore.Append (GREEN record-side)` |
| WU-4 | `8062dea2` | `feat(chat): reload path consults store in factory closure (GREEN reload)` |
| WU-5 | `e84927d4` | `test(chat): identifier survival + unknown-refused (GREEN tail)` |
| WU-6 | `f5f7e788` | `docs(0005): CH-06 shipped + chat-conversation-store spec promoted` |
| (bookkeeping) | `de860bd7` | `docs(openspec): CH-06 change-folder artifacts (design, spec mirror, tasks)` |
| (apply-progress) | `de842da8` | `docs(openspec): CH-06 apply-progress — all 27 tasks [x] + commit SHAs + TDD evidence` |
| (archive — this PR) | TBD | `docs(openspec): CH-06 archived — chat-conversation-store promoted + spec delta closes R-04/R-16/Register 3` |

## Substrate / scope fence

All gates verified via `git diff --stat e3c717a4 HEAD -- <path>` returning "no changes":

- `backend/agent/src/agent/` (NFR-TLS-003 ten-file substrate) — empty.
- `backend/agent/src/agent/import_boundary_test.go` (NFR-CCS-002) — empty.
- `backend/agent/src/agent/history.go` (D-3 — `NewSeededHistory` consumed unchanged) — empty.
- `frontend/` (CH-05 evidence remains valid per PR #195) — empty.
- `backend/agent/src/chat/{http.go,eventsource.go,wire.go,identity.go,registry.go,doc.go,errortext.go}` (CH-03 frozen wire + registry) — empty.
- `openspec/specs/chat-archetype-contract/spec.md`, `openspec/specs/chat-package-boundary/spec.md` (D-5 — no amendment) — empty.
- `docker-compose.yaml`, `infra/` — empty.
- `backend/database_administrator/`, `backend/workspace_syncer/` — empty.

## Apply-progress final state

Per `sdd/cachicamas-chat-conversation-store/apply-progress` (engram `obs-dc7cf219e26f765b`):

- 27/27 tasks complete (every `[x]` in `tasks.md`'s persisted artifact).
- TDD Cycle Evidence: 9/9 scenarios RED → GREEN → TRIANGULATE → REFACTOR documented per `strict-tdd.md`.
- 8 commits on the branch from the apply phase (6 WUs + 2 bookkeeping); this archive commit is the 9th.

## Verify-report final state

Per `sdd/cachicamas-chat-conversation-store/verify-report` (engram `obs-b80ab18007789d99`):

- **Verdict: PASS** (0 CRITICAL, 0 WARNING, 4 SUGGESTION informational).
- 9/9 spec scenarios COMPLIANT.
- 8/8 design decisions coherent.
- 10/10 scope fence gates empty.
- 5/5 doc 0005 audit lines confirmed (`grep` + `awk` on `0005-cachicamas-chat-archetype-task-graph.md`).
- Spec promotion: exactly one new file, no amended promoted spec.
- TDD compliance: 6/6 checks (per-WU RED existence, GREEN existence, TRIANGULATE multi-assertion, REFACTOR roll-up, strict-TDD-verify `step 5a/5d/5f`).
- Strict-TDD-verify evidence captured: `test_output_hash: sha256:27b12aff9f364d50d4f99d15f508b6725edba7305e1b3daf0616c1037818c8c4`, `build_output_hash: sha256:722bd11de69aa9dec87de6c107d12e07d3feccc15e9c2472b12c6cf19defa6da`.

## Documented deviations

The apply phase surfaced four deviations; all are within the orchestrator's pre-granted envelope:

1. **`strings.TrimSpace(cfg.ParticipantID) == ""`** validation (D-1 strict-form variant) — design D-1 said "empty"; whitespace-only is functionally empty, and `TrimSpace` is the strict form. Acceptable; recorded for reviewer audit. The deviation preserves D-1's intent and is strictly stricter than the bare check.
2. **PR carries 9 commits** — the design stated 6 WUs + the archive commit = 7 commits. Apply landed 8 (1 extra for the apply-progress observation; WU-5 is an empty verification commit); this archive adds the 9th. The chat-frontend-wire precedent at PR #195 followed the same `+1 bookkeeping` shape.
3. **Code-only diff is ~1029 LOC**, slightly over the user's pre-granted 1000-line ceiling (29 lines over). The user's "1000 review lines and extend if needed" pre-authorization absorbs this. The PR body calls this out explicitly.
4. **No CRITICAL/WARNING** at verify time — the 4 SUGGESTIONs are informational only and non-blocking.

## Open follow-ups (deferred)

1. **CH-07** (`cachicamas-chat-store-adapter`) — Postgres adapter that satisfies the same `ConversationStore` port with a single constructor swap at `cmd/chat/main.go` (the `chatStore := chat.NewMemoryConversationStore()` declaration near line 152 is the swap point). Closes R-08.
2. **CH-08** (`cachicamas-chat-resume`) — browser surface for resume + listing per `0005:778`. The `ConversationStore` key (currently `participantID`) is just a `string`; the widening to a list-ready interface stays an extension, not a redesign.

## Source of truth updates

| Path | Action |
|---|---|
| `openspec/specs/chat-conversation-store/spec.md` | NEW — promoted from `openspec/changes/archive/2026-08-24-cachicamas-chat-conversation-store/specs/chat-conversation-store/spec.md`. 10 requirements, 9 scenarios, 4 NFRs, `CCS` prefix verified collision-free. |
| `openspec/changes/cachicamas-chat-conversation-store/` | MOVED → `openspec/changes/archive/2026-08-24-cachicamas-chat-conversation-store/` via `git mv`. |
| `openspec/specs/chat-archetype-contract/spec.md` | UNCHANGED (D-5 satisfied; no amendment recorded). |
| `openspec/specs/chat-package-boundary/spec.md` | UNCHANGED (D-5 satisfied; no amendment recorded). |

## PR closure

PR https://github.com/witsaba/cachicamas/pull/196 carries all 9 commits (8 apply + 1 archive) on `feat/chat-conversation-store-ch06`. User reviews and merges. After merge:

- `main` gains `chat-conversation-store` as a new promoted capability.
- `0005-cachicamas-chat-archetype-task-graph.md` status updates to **7 of 12**.
- Milestone doc closes CH-06 (`0005:711-778`); CH-07 is unblocked.

## Archive invariants preserved

- The change folder is moved (`git mv`), preserving `git log` history; nothing deleted.
- All 5 filesystem-resident phase artifacts land in the archive folder: `design.md`, `tasks.md`, `verify-report.md`, `specs/chat-conversation-store/spec.md`, this `archive-report.md`. (Proposal and explore phases were not surfaced to the filesystem; their substance lives in engram topics `sdd/cachicamas-chat-conversation-store/proposal` and `sdd/cachicamas-chat-conversation-store/explore`.)
- 9 engram observations carry the side-channel cross-session memory for this change: `proposal` (`obs-648ba59aa8e4609f`), `spec` (`obs-838962697be27cf1`), `design` (`obs-072c7e2b19a4d211`), `tasks` (`obs-e42ad1836076ebd8`), `apply-progress` (`obs-dc7cf219e26f765b`), `verify-report` (`obs-b80ab18007789d99`), `preflight` (`sdd/ch-06/preflight`, `obs-0dcc794c464b705a`), `delivery-decision` (`sdd/ch-06/delivery-decision`, `obs-ba59e7cb6072670d`), this `archive-report` (sync_id TBD).
- `chat-archetype-contract`'s append-only identifier rule (`spec.md:9`) holds — no requirement was renumbered, none was added to it.
- Verify-evidence bytes are stable across the apply → verify → archive boundary (the test and build output hashes are reproduced verbatim from the verify-report frontmatter).

## Cycle complete

The CH-06 SDD cycle closes with PR #196. Future CH-07 can `mem_search` the engram topic `sdd/cachicamas-chat-conversation-store/archive-report` to recover the prior context (proposal D-1..D-8, spec R-CCS-001..010 / S-CCS-001..009 / NFR-CCS-001..004, doc 0005 closure notes).
