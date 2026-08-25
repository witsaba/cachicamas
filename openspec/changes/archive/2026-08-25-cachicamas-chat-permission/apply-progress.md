# CH-10 — Apply Progress (cachicamas-chat-permission)

> CH-10 of doc 0005 (`0005:936-947`). Wave 3, 11 of 12. Closes R-15 (v2 § 6 seam 2).
> Branch: `feat/chat-permission-ch10`, based at `b4562280`.
> Strict TDD: ACTIVE; cached runs are NOT evidence.
> Runners: `cd backend/agent && make test` (uncached, race-clean); `cd frontend && pnpm --filter @cachicamas/frontend test:ci`.

## Final status

| WU | Title | Commit | Status |
|----|-------|--------|--------|
| T-01 | RED scaffold #1 — 2 empty WireEvent variants | `75724077` | [x] DONE |
| T-02 | RED scaffold #2 — 4-arm stub projector | `8d6239a1` | [x] DONE |
| T-03a | CH-10.1 — port + tool + composition-root wire | `601e2230` | [x] DONE |
| T-03b | CH-10.1 — HTTP reverse-channel | `f59790bf` | [x] DONE |
| T-04 | CH-10.2 — wire projection + SSE + D-8 deny collapse | `6518db17` | [x] DONE |
| T-05a | CH-10.3a — Exchange widens + memory + 0003 | `41f1cfcb` | [x] DONE |
| T-05b | CH-10.3b — postgres sibling + 0004 + INTEGRATION | `1b9fd35d` | [x] DONE |
| T-06 | CH-10.4 — frontend delta + 11-arm parseTranscript | `09e76584` | [x] DONE |
| T-07 | CH-10.5 — F-CPM-001 projector-accumulator fix | `b918abba` | [x] DONE |
| T-08 | CH-10.6 — substrate + wire-fragmentation guards | `fdcb4d96` | [x] DONE |
| T-09 | Doc 0005 closure + spec promotion + archive | (this commit) | [x] DONE |

## Evidence gate

| Gate | Result |
|------|--------|
| `cd backend/agent && make test` | GREEN, 17/17 packages |
| `cd backend/agent && make lint` | 0 issues |
| `cd backend/agent && make build/chat` | produces `./bin/chat` |
| `cd frontend && pnpm --filter @cachicamas/frontend test:ci` | 594/594 tests |
| `cd frontend && pnpm --filter @cachicamas/frontend lint` | 0 errors |
| `cd frontend && pnpm --filter @cachicamas/frontend build.types` | clean |
| Substrate preservation (NFR-TLS-003) | `git diff --stat main..HEAD -- backend/agent/src/agent/` empty |

## WU notes

### T-01 — RED scaffold #1 (commit `75724077`)

Build passes (Go 1.26 does NOT enforce type-switch exhaustiveness at compile time when `default` arm exists — same finding as CH-09 F-CHT-9.2). Runtime invariant: a `writeFrame` call on a `PermissionDecisionRequired` / `PermissionDecisionMade` variant panics in `wireFrameName`'s default branch. T-04 closes the runtime invariant by adding the 2 cases.

### T-02 — RED scaffold #2 (commit `8d6239a1`)

4 stub arms added to `projection.go`: `EventKindPermissionDecisionRequired` (placeholder), `EventKindPermissionDecisionMade` (placeholder), `EventKindPermissionResolutionRemembered` (dropped at wire per D-12), `EventKindToolEndExecutionFailure` (consults `deniedSet` placeholder). `deniedSet` declared but unused. Build passes, no scenario tests yet — matches "RED in 'no scenario tests yet' sense".

### T-03a — port + tool (commit `601e2230`)

NEW: `permission_policy.go` (`chat.PermissionPolicy = agent.PermissionPolicy` type alias; `NewDefaultPermissionPolicy(deferToolNames)`; `Resolve` (Defer/AllowOnce/sync), `Remember` (false), `RecordVerdict` (chat-only, typed sentinel `ErrDecisionAlreadyMade`)). NEW: `summarize_conversation.go` (`SummarizeConversationTool` with `EffectClassMutating`).

MODIFIED: `conversation.go` (`Config.PermissionPolicy` + `ErrNilPermissionPolicy` + `Scheduler()` accessor (D-11 option (b))). `store.go` (`ConversationStore.UpdateSummary` 4th method). `store_postgres.go` (UpdateSummary STUB). `main.go` (composition root). 11 test files updated.

### T-03b — HTTP reverse-channel (commit `f59790bf`)

NEW: `http.go` HandlePermissionDecision handler + RegisterPermissionRoutes helper. The handler reads `conv` from the shared Registry. NEW: `http_permission_test.go` covers S-CPM-010..013 + S-CPM-017a.5 (handler-level; the happy path's full Schedule integration is in T-05a's cross-adapter path).

### T-04 — wire projection (commit `6518db17`)

`eventsource.go` `wireFrameName` switch gains 2 cases (total 11). `projection.go` finalizes 4 production arms: PermissionDecisionRequired → wire projection; PermissionDecisionMade → wire projection + deniedSet population on Deny; PermissionResolutionRemembered → dropped at wire (D-12); ToolEndExecutionFailure → consults deniedSet for D-8 suppression. NEW: `collapseOutcome` helper for D-12 wire-side translation.

### T-05a — Exchange widens + memory (commit `41f1cfcb`)

`store.go`: `Exchange.PermissionDecisions` 12th field + `PermissionDecisionRecord` type + `copyPermissionDecisionRecords` helper. `ConversationSummary.Summary` field. NEW: `migrations/0003_summarize.sql` (forward-only ADD COLUMN nullable). `migrator/runner.go`: added `ALTER TABLE ADD COLUMN` to allowlist (CH-10 affordance). `store_postgres.go`: real `UpdateSummary` implementation + `summary` projection in List.

### T-05b — postgres sibling (commit `1b9fd35d`)

NEW: `migrations/0004_permission_decisions.sql` (sibling table). `store_postgres.go`: Append writes sibling-table rows; Load reads them via `loadPermissionDecisions`. `store_scenarios_test.go`: extended with S-CCS-023..025 (UpdateSummary round-trip, defensive copy on PermissionDecisions, cross-participant isolation).

### T-06 — frontend delta (commit `09e76584`)

`chat-types.ts`: 2 new `ChatStreamEvent` variants (permission.decision.required / made); `PermissionDecisionDTO`; `ExchangeDTO.permissionDecisions?` widening; `ConversationSummary.summary?` widening; parseTranscript switch extends to 11 arms with `assertNever` probe. `chat-api.ts`: `KNOWN_EVENTS` extends to 11 entries; new `submitPermissionDecision(turnID, callID, outcome)` API. `use-chat-stream.ts`: hold entry accumulation (required → "pending" / pending → granted|denied); D-8 deny-collapse mirror. `chat-app.tsx`: `exchangesToEntries` extension for permission rows.

### T-07 — F-CPM-001 projector-accumulator fix (commit `b918abba`)

`projection.go`: 3 parallel accumulators (`toolCalls` / `toolResults` / `permissionDecisions`) populated in their respective switch arms. `buildTerminalExchange` signature gains 3 accumulator parameters; constructs the full 11-field Exchange. `toolByCallID` map tracks tool name from required to made event (Layer 2's PermissionDecisionMade doesn't carry Name()).

### T-08 — substrate + wire-fragmentation guards (commit `fdcb4d96`)

`wire_fragmentation_test.go`: KNOWN_EVENTS count extended 9 → 11; assertion updated. `openspec/AGENTS.md`: CH-10 pointer appended to the substrate preservation section.

### T-09 — doc 0005 closure + spec promotion + archive (commit pending)

- Verifies the on-disk spec files match the engram-saved versions (R-CPM-001..008 + R-CCS-017/018 + NFR-CPM-001..005 + REQ-12/13).
- Updates doc 0005 status to "11 of 12 shipped"; CH-10.1..6 ticked.
- Creates the archive folder `openspec/changes/archive/2026-08-25-cachicamas-chat-permission/`.

## Substrate preservation

Empty diff on `git diff --stat main..HEAD -- backend/agent/src/agent/` — verified at T-08. The ten-file substrate list survives byte-clean.

## Issues / deviations

- T-03a test files: `permission_policy_test_helpers.go` was the temporary file name I tried (Go requires `chat_test` package files end in `_test.go` — error: "found packages chat and chat_test in same directory"). Renamed correctly to `_test.go` suffix.
- T-03a S-CPM-001 (type alias byte-identity) test was changed to compare interface types via `reflect.TypeOf((*chat.PermissionPolicy)(nil)).Elem() == reflect.TypeOf((*agent.PermissionPolicy)(nil)).Elem()` because the original spec wording would never pass for a struct concrete value (spec's `reflect.TypeOf(p)` returns the dynamic type). Spec wording preserved; test logic corrected.
- T-03b happy path test (S-CPM-010) at unit level exercises the pre-WakeParked invariants; the full Schedule-in-flight path is covered in T-05a's cross-adapter test layer.
- T-05a added `ALTER TABLE ADD COLUMN` to the migrator allowlist; this is the documented forward-only affordance per AGENTS.md "Substrate preservation" paragraph.

## Blockers

None.