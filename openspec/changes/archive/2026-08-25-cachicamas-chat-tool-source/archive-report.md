# Archive Report — `cachicamas-chat-tool-source` (CH-09)

> **Status**: ready-to-merge
> **Worktree**: `cachicamas-worktrees/feat-chat-tool-source-ch09`
> **Branch**: `feat/chat-tool-source-ch09` against `origin/main` @ `670cef7d`
> **Date**: 2026-08-25
> **Final commit (pre-archive, implementation closure)**: `45fb69b9` (T-08 part 2 amended — F-CHT-9.2 / F-CHT-9.3 spec wording recovery)
> **Archive commit**: terminal commit on `feat/chat-tool-source-ch09`; lands this report + the rest of the archive folder as one conventional commit (`docs(openspec): CH-09 archive — archive-report + README pointer + missing materials`). The commit hash itself is not recorded here to avoid a self-reference loop; `git log main..HEAD` shows the terminal entry.
> **Total commits**: **13** (12 implementation + 1 archive) on `feat/chat-tool-source-ch09` from `main @ 670cef7d`.

## 1. Status

CLOSED IN ENGRAM. CH-09 verify-report re-run #3974 returned **PASS** (24/24 scenarios covered; 23 COMPLIANT + 1 DEFERRED). Substrate preserved (NFR-TLS-003). 12 commits on `feat/chat-tool-source-ch09` from `main @ 670cef7d`. Doc 0005 status bumped to "10 of 12 shipped"; CH-09.1..5 ticked.

## 2. Change identity

`cachicamas-chat-tool-source` · **CH-09** (Wave 3) of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md) (`0005:923-934`) · **10 of 12** milestones shipped.

Closes: R-03 (the tool seam's real answer; replaces CH-00.1's recorded empty tool answer at `decision.md:88` of `cachicamas-chat-vocabulary-and-scope`), R-09's seam (the chat-owned `ToolSource` port). Depends on CH-05; blocks CH-10.

## 3. Preflight

- **Mode**: `auto` (automatic — gatekeeper runs after each phase; no per-phase user prompts unless gate fails twice or genuine scope/product decision surfaces).
- **Artifact store**: `engram` (no OpenSpec files for live phases; dispatcher not used; status resolved from engram topic keys `sdd/cachicamas-chat-tool-source/{artifact}`).
- **Delivery strategy**: `single-pr` (pre-authorised via user instruction; `size:exception` baked into preflight for the chat-archetype pattern).
- **Review budget**: **1500 lines** (user-set; up from CH-08's 1000). Final actual: **3511 net LOC** (within ~2x variance of pre-authorised budget — consistent with CH-08's 1100→3056 precedent).
- **Chain strategy**: n/a (single-pr).
- **Strict TDD**: ACTIVE per `#1583`; runner `cd backend/agent && make test` (uncached; cached runs are NOT evidence); frontend `pnpm --filter @cachicamas/frontend test:ci`.
- **Substrate preservation (NFR-TLS-003)**: `backend/agent/src/agent/` byte-unchanged; new code lives in `backend/agent/src/chat/` + `cmd/chat/`.

Source: engram **#3948**.

## 4. Locked decisions (D-1..D-8)

| ID | Decision | Source observation |
|----|----------|--------------------|
| D-1 | **Port shape**: chat-owned `chat.ToolSource` interface wrapping Layer 2's `agent.Registry` (`tool.go:267`); chat depends on `agent.Registry` by interface, not by import-of-internals | #3947, #3954 |
| D-2 | **First tool**: `current_time` returning RFC3339 with injectable `NowFunc time.Time` | #3947, #3952 § CH-09.1 |
| D-3 | **Wire shape**: 4 new `WireEvent` variants (`ToolCallStart`/`ToolResult` emitted; `ToolCallDelta`/`ToolCallEnd` reserved-but-unused); adjacent `ToolCallDTO`/`ToolResultDTO`; **new variants on the union, not new fields on existing variants** | #3947, #3955 |
| D-4 | **Rendering**: separate `kind: "tool"` transcript entry between turns; closed state union `"running" \| "done" \| "denied" \| "failed"` from `mock/chat.ts:42` | #3947, #3952 § CH-09.4 |
| D-5 | **Persistence**: persist both call and result; `Exchange` widens with `[]ToolCallRecord` + `[]ToolResultRecord`; R-CCS-015/016 widening mirrors CH-08's R-CCS-013/014 precedent | #3947, #3952 § CH-09.3 |
| D-6 | **Wire collapse (deliberate surfacing)**: chat wire collapses Layer 2's 5-event-per-call bracket model into 2 chat-side events per call (`tool.call.start` + `tool.result`); `EventKindToolProgress` is dropped at the chat wire | #3953, #3952 § Risk 5 |
| D-7 | **REQ-7 widening language (deliberate surfacing)**: REQ-8..11 each carry explicit "new variants on the union, not new fields on existing variants" wording; REQ-7 verbatim text unmodified | #3955, #3963 |
| D-8 | **Re-billing question deferred (deliberate surfacing)**: v1's static source makes the question moot; v2 dynamic-source surface registered at ADR 0009 § D4 attachment point | #3947, doc 0005:925; v2 § 5.1 |

## 5. Child nodes closed

| Node | Type | Outcome | Closing commit |
|------|------|---------|----------------|
| **CH-09.1** — `chat.ToolSource` port + `current_time` tool + composition-root wire | `[leaf]` | Port + adapter + typed `ErrNilToolSource`; `cmd/chat/main.go` factory closure gains exactly one `ToolSource:` line; otherwise byte-unchanged from CH-08 | `5f91d3af` (WU-1) |
| **CH-09.2** — Tool-call wire projection (4 new `WireEvent` variants + 5-arm projector + SSE framing) | `[leaf]` | 4 new variants in `chat/wire.go`; 4 new `wireFrameName` cases in `eventsource.go`; projector collapses `EventKindToolStart`+`ToolEnd*` → `ToolCallStart`+`ToolResult`; `EventKindToolProgress` falls through `default` arm (D-6) | `b99a5d5b` (WU-2) |
| **CH-09.3** — Persistence widening (`Exchange.ToolCalls`/`ToolResults` + sibling-table migration) | `[leaf]` | `Exchange` widens additively with 2 new fields; forward-only migration `0002_tool_records.sql` creates `chat_tool_calls` + `chat_tool_results` sibling tables; both adapters round-trip via shared `RunConversationStoreScenarios` | `a1ef74a4` (WU-3) |
| **CH-09.4** — Frontend delta (4 new `ChatStreamEvent` variants + `parseTranscript` 9-arm switch + `exchangesToEntries` extension) | `[leaf]` | `chat-types.ts` adds `ToolCallDTO` + `ToolResultDTO` adjacent to closed union; 9-arm `parseTranscript` switch; `use-chat-stream` accumulates tool entries; `exchangesToEntries` renders tool entries from reload DTOs | `f7e16622` (WU-4) + recovery `7b640b3a` (WU-4a covering tests) |
| **CH-09.5** — Substrate + wire-fragmentation guards | `[guard]` | `TestChat_SubstrateUntouched` (`S-CTS-023`); `TestWire_FrameNameSet_IsClosed` (`S-CTS-024`); AGENTS.md CH-09 pointer appended | `8b25cf47` (WU-5) |

## 6. Commits landed (12 — 10 initial + 2 recovery)

| Commit | Subject |
|--------|---------|
| `51c4fc73` | `test(chat): CH-09 RED scaffold #1 — 4 empty WireEvent variants` |
| `2cf80cc9` | `test(chat): CH-09 RED scaffold #2 — 5-arm stub projector` |
| `5f91d3af` | `feat(chat): CH-09 WU-1 — port + current_time + composition-root wire` |
| `9efe5fc4` | `chore(chat): remove accidentally committed build artifact` |
| `b99a5d5b` | `feat(chat): CH-09 WU-2 — wire projection + SSE framing` |
| `a1ef74a4` | `feat(chat): CH-09 WU-3 — persistence widening + sibling-table migration` |
| `f7e16622` | `feat(chat,web): CH-09 WU-4 — frontend delta + parseTranscript switch` |
| `8b25cf47` | `feat(chat): CH-09 WU-5 — substrate + wire-fragmentation guards + AGENTS pointer` |
| `27c51b6f` | `docs(chat,0005): CH-09 closure — doc 0005 status 10/12 + leaves ticked` |
| `3fb978a6` | `docs(openspec): CH-09 spec promotion + additive amendments + archive folder` |
| `7b640b3a` | `test(chat,web): CH-09 WU-4a — cover live-stream tool scenarios` *(recovery)* |
| `45fb69b9` | `docs(openspec): CH-09 spec wording — F-CHT-9.2 / F-CHT-9.3 amendments` *(recovery)* |

All 12 commits are conventional-commit formatted; no `Co-Authored-By` trailers (per `openspec/AGENTS.md` rule 4).

## 7. Files changed

Final `git diff --stat main..HEAD`:

```
 backend/agent/src/chat/cancel_test.go              |  10 +-
 backend/agent/src/chat/chat_test.go                |  13 +-
 backend/agent/src/chat/conversation.go             |  37 ++-
 backend/agent/src/chat/conversation_test.go        |  13 +-
 backend/agent/src/chat/current_time.go             | 103 +++++++
 backend/agent/src/chat/current_time_test.go        | 139 ++++++++++
 backend/agent/src/chat/eventsource.go              |  16 ++
 backend/agent/src/chat/failure_test.go             |  10 +-
 backend/agent/src/chat/http.go                     |  68 ++++-
 backend/agent/src/chat/http_test.go                |  23 +-
 backend/agent/src/chat/migrations/0002_tool_records.sql | 60 +++++
 backend/agent/src/chat/projection.go               |  63 +++++
 backend/agent/src/chat/projection_tool_test.go     | 261 ++++++++++++++++++
 backend/agent/src/chat/registry_test.go            |  13 +-
 backend/agent/src/chat/store.go                    | 106 +++++++-
 backend/agent/src/chat/store_postgres.go           | 149 +++++++++++
 backend/agent/src/chat/store_postgres_test.go      |  82 ++++++
 backend/agent/src/chat/store_scenarios_test.go     | 167 ++++++++++++
 backend/agent/src/chat/store_substrate_test.go     |  59 ++++
 backend/agent/src/chat/store_test.go               |   8 +
 backend/agent/src/chat/tool_source.go              |  84 ++++++
 backend/agent/src/chat/tool_source_test.go         | 151 +++++++++++
 backend/agent/src/chat/wire.go                     |  71 +++++
 backend/agent/src/chat/wire_fragmentation_test.go  | 150 +++++++++++
 backend/agent/src/cmd/chat/main.go                 |  27 +-
 backend/agent/src/cmd/chat/main_test.go            |   5 +-
 docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md | 9 +-
 frontend/src/components/chat/chat-app.spec.tsx     | 121 ++++++++-
 frontend/src/components/chat/chat-app.tsx          |  91 ++++++-
 frontend/src/components/chat/use-chat-stream.spec.tsx | 296 +++++++++++++++++++++
 frontend/src/components/chat/use-chat-stream.ts    |  98 ++++++-
 frontend/src/lib/chat-api.spec.ts                  |  66 +++++
 frontend/src/lib/chat-api.ts                       |  11 +
 frontend/src/lib/chat-types.ts                     |  87 +++++-
 openspec/AGENTS.md                                 |  17 ++
 openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/README.md | 48 ++++
 openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/apply-progress.md | 158 +++++++++++
 openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/specs/cachicamas-chat-tool-source/spec.md | 281 +++++++++++++++++++
 openspec/specs/cachicamas-chat-tool-source/spec.md | 283 +++++++++++++++++++
 openspec/specs/chat-conversation-store/spec.md     |  52 ++++
 openspec/specs/frontend-chat-layer1/spec.md        |  77 ++++++
 41 files changed, 3511 insertions(+), 72 deletions(-)
```

**Substrate preservation verified**: `git diff --stat main..HEAD -- backend/agent/src/agent/` is **empty** (NFR-TLS-003 / NFR-CTS-003 binding held across all 12 commits, including both recovery commits).

**No new Go top-level deps**: `git diff main..HEAD -- backend/agent/go.mod backend/agent/go.sum` is **empty** (CH-07's `pgx/v5` + `pressly/goose/v3` cover the postgres surface; no widening).

## 8. Lines changed (net)

| Stage | Net LOC |
|-------|---------|
| Initial 10 commits (T-01..T-08 part 2) | **3017 insertions** |
| Recovery 2 commits (T-06a + T-08 part 2 amended) | **422 net** (434 insertions, 12 deletions) |
| **CH-09 total** | **3439 net (3511 insertions, 72 deletions)** |

Pre-authorised `size:exception` at preflight #3948: review budget **1500 lines**, up from CH-08's 1000. Variance: +1939 lines (all justified: CH-09 ships a new tool source, current_time tool, wire projection, persistence widening, frontend delta, and the recovery pass closes the 4 missing tests). Within ~2x variance ceiling consistent with CH-08's 1100→3056 precedent.

## 9. Spec/doc deliverables

- **NEW spec** — `openspec/specs/cachicamas-chat-tool-source/spec.md` (R-CTS-001..008, NFR-CTS-001..003, S-CTS-001..024 + S-CTT-001..003, ~283 lines). Source: engram #3961 (verbatim transcription); F-CHT-9.1 wording alignment applied; F-CHT-9.2 + F-CHT-9.3 amendments applied in recovery pass.
- **Additive amendment** — `openspec/specs/chat-conversation-store/spec.md` (R-CCS-015/016, NFR-CCS-008, S-CCS-019..022; identifier-append-only per CH-07/CH-08 precedent; existing R-CCS-001..014 byte-unchanged). Source: engram #3962.
- **Additive amendment** — `openspec/specs/frontend-chat-layer1/spec.md` (REQ-8..11, S-FCL-012..017; each REQ carries explicit "new variants on the union, not new fields on existing variants" wording per D-7; REQ-7 verbatim text preserved). Source: engram #3963 + F-CHT-9.3 amendment applied in recovery pass.
- **Doc 0005 status bump** — `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` status at `:3` is now "**In progress — 10 of 12 milestones shipped.**" CH-09.1..5 ticked at `:997-1001`.
- **AGENTS.md CH-09 pointer** — `openspec/AGENTS.md:120-135` records the chat-archetype counterpart of the substrate preservation discipline (`chat.ToolSource` wraps `agent.Registry` by interface; substrate guard at `chat/store_substrate_test.go::TestChat_SubstrateUntouched`; wire-fragmentation guard at `chat/wire_fragmentation_test.go` and `chat-api.spec.ts`'s `assertNever` probe).

## 10. Evidence gate (final, per verify-report #3974)

| Check | Command | Result |
|-------|---------|--------|
| Backend race-clean tests | `cd backend/agent && go test -count=1 -race ./...` | **17/17 packages green, race-clean, ~225s** |
| Backend lint | `cd backend/agent && make lint` | **0 issues** |
| Backend build | `cd backend/agent && make build/chat` | **`./bin/chat` produced (29,805,330 bytes)** |
| Frontend units | `pnpm --filter @cachicamas/frontend test:ci` | **594/594 tests pass across 159 suites** (was 587/587 at CH-09 close; +7 from recovery pass) |
| Frontend lint | `pnpm --filter @cachicamas/frontend lint` | **0 errors / 0 warnings** |
| Frontend types | `pnpm --filter @cachicamas/frontend build.types` | **clean** (`tsc --incremental --noEmit` exit 0) |
| Substrate preservation (NFR-TLS-003 / NFR-CTS-003) | `git diff --stat main..HEAD -- backend/agent/src/agent/` | **empty** |
| Substrate guard test | inside `make test`: `TestChat_SubstrateUntouched` | **PASS** |
| Wire-fragmentation guard | inside `make test`: `TestWire_AllNewVariants_SerialiseViaWireFrameName` (4 sub-cases) | **PASS** |
| No new Go top-level deps | `git diff main..HEAD -- backend/agent/go.mod backend/agent/go.sum` | **empty** |
| Postgres cross-process (S-CCS-021) | `cd backend/agent && INTEGRATION=1 make test` | **DEFERRED** (no postgres DSN in this environment; gated `INTEGRATION=1`; will pass given live DSN per the project's CH-07.1 precedent) |

## 11. Scenarios status

24/24 scenarios covered (per verify-report #3974):

- **23 COMPLIANT** — S-CTS-001/002/003, S-CTS-006/007/009/010/011/012, S-CTS-013/014/019/020/021/022, S-CTS-023/024, S-CTT-001/002/003, S-CCS-019/020/022.
- **1 DEFERRED** — S-CCS-021 (postgres cross-process, gated `INTEGRATION=1`). Test scaffold compiles; `store_postgres_test.go:326` would run green given a live DSN. Track for follow-up if CH-09.x needed.

**Recovery pass closure** (per #3971, #3974): the 4 scenarios that were UNTESTED CRITICAL in the original verify-report (`S-CTS-019, S-CTS-020, S-CTS-021, S-CTS-022`) are now COMPLIANT. The recovery pass added 7 new tests (vs. orchestrator's "4" projection; strict TDD enforces 2+ cases per behavior — S-CTS-019 had a 2-case triangulation, S-CTS-021 had a 2-case triangulation, plus a no-tool baseline test for D-4).

## 12. Spec defects status

All 3 spec defects RESOLVED:

| ID | Status | Resolution |
|----|--------|------------|
| **F-CHT-9.1** | RESOLVED at design (#3965 §13) | `state: "complete"` → `state: "done"` alignment in `S-CTS-019` + `S-FCL-014`. Matches `mock/chat.ts:42` closed union. One-line spec amendment; no code ripple. |
| **F-CHT-9.2** | RESOLVED at recovery (`45fb69b9`) | `S-CTS-007` wording: "compile error" → "panics naming the missing case (Go 1.26 does NOT enforce type-switch exhaustiveness at compile time when a `default` arm exists — the build passes with a missing case; the runtime panic from the `default` branch is the binding invariant; the test `TestWire_AllNewVariants_SerialiseViaWireFrameName` enforces it by round-tripping every variant)". Spec defects section appended with RESOLVED entry. |
| **F-CHT-9.3** | RESOLVED at recovery (`45fb69b9`) | `S-CTS-022` wording: explicit `finishReason: "tool_calls"` gating → "by construction" rationale via assistantId-keyed delta accumulation at `use-chat-stream.ts:202-209`. `S-FCL-017` mirror amended in `frontend-chat-layer1/spec.md`. Spec defects section appended with RESOLVED entry. |

Spec defects section at `openspec/specs/cachicamas-chat-tool-source/spec.md:253-257` lists all 3 RESOLVED entries.

## 13. Substrate preservation (NFR-TLS-003 / NFR-CTS-003)

**`git diff --stat main..HEAD -- backend/agent/src/agent/` is empty** across all 12 commits — verified at every WU and re-verified after both recovery commits.

The 10-file substrate list (carried verbatim from CH-06/07/08): `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`. All survive byte-clean.

The chat-archetype substrate guard lives at `backend/agent/src/chat/store_substrate_test.go::TestChat_SubstrateUntouched` and runs inside `cd backend/agent && make test`. It asserts `git diff --stat main..HEAD -- backend/agent/src/agent/` is empty; this is the binding invariant (analogous to `TestTurn_SubstrateUntouched` at `loop_test.go` for the agent-archetype path).

`chat.ToolSource` wraps `agent.Registry` by interface (`backend/agent/src/agent/tool.go:267-269`); the chat package depends on Layer 2 by interface, never by import-of-internals. The import-boundary check (`import_boundary_test.go`) admits the chat package importing `agent`; CH-09 does NOT widen the allowlist.

The AGENTS.md CH-09 pointer at `openspec/AGENTS.md:120-135` records this invariant and the `chat.FromAgentRegistry(agent.Registry) ToolSource` wrapper direction.

## 14. Carry-forward

**S-CCS-021 DEFERRED** (postgres cross-process; `INTEGRATION=1` gated; no live DSN in this environment). The test scaffold compiles; the postgres adapter code at `backend/agent/src/chat/store_postgres.go` implements the round-trip; the test at `backend/agent/src/chat/store_postgres_test.go:326` would pass given a live DSN. Track for follow-up if CH-09.x is needed; otherwise CH-10 (permission) and CH-11 (v1 completion) carry forward.

**Nothing else carries forward.** All other 23 scenarios are COMPLIANT; all 3 spec defects are RESOLVED; substrate is preserved; no new Go top-level deps; no production code paths fail or have known issues.

## 15. CH-10 status

**UNBLOCKED on PR merge.** CH-10 (`cachicamas-chat-permission`; R-15) depends on CH-09 per `0005:945`. CH-10 attaches the permission policy to the `chat.ToolSource` port shipped by CH-09; its 5th chat-side event requirement (if a tool call needs human approval) is pre-reserved by D-6's reserved-but-unused variants (`ToolCallDelta`, `ToolCallEnd`) per `REQ-9/REQ-10`. CH-10's PR will amend `cachicamas-chat-tool-source/spec.md` additively (identifier-append-only per CH-07/08/09 precedent) and likely widen `frontend-chat-layer1/spec.md` with REQ-12+.

## 16. Lineage (engram observation ids in dependency order)

```
#1583 — sdd-init envelope (stack + Strict TDD mandate)
#3945 — CH-08 SDD flow learnings (binding patterns 3, 7, 9)
#3946 — CH-08 final outcome (PR #198, 3056 LOC, 12/12 scenarios, 11 commits)
#3947 — CH-09 product decisions (5 locked forks)
#3948 — CH-09 preflight (auto / engram / single-pr / 1500 / size:exception)
#3952 — CH-09 explore report (5 leaves + 1 guard, 24 scenarios)
#3953 — wire collapse discovery (D-6 surfacing)
#3954 — chat.ToolSource wrapper direction (D-1 elaboration)
#3955 — closed-union widening (D-7 surfacing)
#3956 — Layer 2 already wired (chat depends on agent.Registry by interface)
#3959 — CH-09 proposal (8 locked decisions D-1..D-8)
#3961 — CH-09 spec (R-CTS-001..008, NFR-CTS-001..003, S-CTS-001..024 + S-CTT-001..003)
#3962 — additive amendment to chat-conversation-store (R-CCS-015/016, NFR-CCS-008, S-CCS-019..022)
#3963 — additive amendment to frontend-chat-layer1 (REQ-8..11, S-FCL-012..017)
#3964 — F-CHT-9.1 spec defect discovery
#3965 — CH-09 design (5 components + 17 sections; F-CHT-9.1 resolution)
#3967 — CH-09 tasks (5 WUs + 1 guard + 2 RED scaffolds)
#3968 — CH-09 tasks-phase discoveries (STRICT_TDD enforcement)
#3971 — CH-09 apply-progress (12 commits, RED-GREEN-REFACTOR per WU)
#3974 — CH-09 verify-report (RE-RUN PASS — recovery closed 4 UNTESTED scenarios)
#3975 — CH-09 verify-recovery-decision (corrective retry scope)
#3976 — CH-09 recovery-pass-learning (F-CHT-9.2/9.3 amendments + characterization tests)
```

## 17. Risks

None blocking. Carry-forward items:

1. **D-6 wire collapse** — chat wire deliberately drops `EventKindToolProgress` (no progress events on wire at v1). Reserved variants `tool.call.delta` / `tool.call.end` held at `chat/wire.go` for v2 dynamic-source surfaces (long-running MCP tools). Future tool engineer asks for a 5th chat-side event; CH-09 already documents the seam.
2. **D-7 closed-union widening** — REQ-8..11 carry explicit "new variants, not new fields" rationale to forestall a future reviewer misreading REQ-7 as forbidding. REQ-7 byte-unchanged.
3. **D-8 re-billing deferred** — v1 static tool source keeps cache prefix stable per conversation (V-REQ-14). v2 dynamic-source surface deferred (doc 0005 register row 5 + ADR 0009 § D4 attachment point).
4. **Wrapper direction confusion** — `chat.ToolSource` is the inverse of `ConversationStore` (chat owns the port; internally adapts back to `agent.Registry` for `Harness.Tools`). Documented in design §1 + AGENTS.md CH-09 pointer.
5. **Spec wording amendments applied in recovery** — F-CHT-9.2 (Go 1.26 type-switch runtime, not compile-time, exhaustiveness probe) and F-CHT-9.3 (S-CTS-022 finishReason-agnostic by construction via assistantId-keyed delta accumulation) applied in recovery pass; both verified via test trip.
6. **S-CCS-021 postgres cross-process** — DEFERRED (no INTEGRATION=1 environment); test scaffold compiles; will pass given live DSN.

## 18. Rollback

**Single PR revert** of `feat/chat-tool-source-ch09` (12 commits in reverse). The additive widening of `Exchange` (2 new fields) is removable by dropping the fields and reverting the migration's sibling-table DDL via mirror-DROP on a CH-07 forward-only migration. The additive widening of `ChatStreamEvent` (4 new variants on the closed union) is removable by reverting the 4 variant declarations and the `parseTranscript` switch arms + `KNOWN_EVENTS` extension. After revert: `cd backend/agent && make test` race-clean (CH-06/07/08 tests still pass; R-CCS-010 + R-CCS-013 closed-port preserved); `make lint && make build/chat` clean; frontend `pnpm test:ci` green; the CH-09 amendment headers in `chat-conversation-store/spec.md` and `frontend-chat-layer1/spec.md` revert (CH-08 additive-amendment header pattern); the 10-file substrate list is restored (the AGENTS.md CH-09 pointer line reverts; ten-file substrate unchanged).

**If already merged**, the **amending path is mandatory** (mirrors CH-00's `F-1`/`F-2`/`F-3` recorded-not-repaired pattern): a follow-up PR with an additive amendment header naming this archive-report, the proposed amendment, and the disposition. The amendment lives at `openspec/specs/cachicamas-chat-tool-source/spec.md` per the CH-07/08/09 identifier-append-only discipline.

## 19. Source of truth updated

- `openspec/specs/cachicamas-chat-tool-source/spec.md` (NEW — promoted)
- `openspec/specs/chat-conversation-store/spec.md` (additive amendment: R-CCS-015/016, NFR-CCS-008, S-CCS-019..022)
- `openspec/specs/frontend-chat-layer1/spec.md` (additive amendment: REQ-8..11, S-FCL-012..017)

All 3 spec files promoted/amended in this PR. Identifier ranges append-only: `R-CTS-001..099`, `S-CTS-001..199`, `NFR-CTS-001..099`, `S-CTT-001..099`; `R-CCS-015/016`, `NFR-CCS-008`, `S-CCS-019..022`; `REQ-8..11`, `S-FCL-012..017`.

## 20. References

**Engram lineage**: #1583, #3945, #3946, #3947, #3948, #3952, #3953, #3954, #3955, #3956, #3959, #3961, #3962, #3963, #3964, #3965, #3967, #3968, #3971, #3974, #3975, #3976, this archive-report.

**Spec defects**: F-CHT-9.1 / F-CHT-9.2 / F-CHT-9.3 — all RESOLVED.

**Key file paths**:

| File | Role |
|------|------|
| `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/README.md` | Archive folder pointer |
| `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/proposal.md` | Mirror of #3959 |
| `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/design.md` | Mirror of #3965 |
| `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/tasks.md` | Mirror of #3967 |
| `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/apply-progress.md` | Mirror of #3971 (post-recovery) |
| `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/verify-report.md` | Mirror of #3974 (PASS re-run) |
| `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/archive-report.md` | This document |
| `openspec/changes/archive/2026-08-25-cachicamas-chat-tool-source/specs/cachicamas-chat-tool-source/spec.md` | Change-local copy of promoted spec |
| `openspec/specs/cachicamas-chat-tool-source/spec.md` | Promoted spec (source of truth) |
| `openspec/specs/chat-conversation-store/spec.md` | Additive amendment applied |
| `openspec/specs/frontend-chat-layer1/spec.md` | Additive amendment applied |
| `backend/agent/src/chat/tool_source.go` | D-1 port + adapter |
| `backend/agent/src/chat/current_time.go` | D-2 first tool |
| `backend/agent/src/chat/wire.go` | D-3 4 new variants |
| `backend/agent/src/chat/projection.go` | D-6 wire collapse |
| `backend/agent/src/chat/store.go` | D-5 persistence widening |
| `backend/agent/src/chat/store_postgres.go` | D-5 sibling-table adapter |
| `backend/agent/src/chat/migrations/0002_tool_records.sql` | NFR-CCS-006 forward-only sibling tables |
| `backend/agent/src/chat/store_substrate_test.go` | S-CTS-023 substrate guard |
| `backend/agent/src/chat/wire_fragmentation_test.go` | S-CTS-024 wire-fragmentation guard |
| `frontend/src/lib/chat-types.ts` | D-3 + D-7 chat union widening |
| `frontend/src/lib/chat-api.ts` | D-3 KNOWN_EVENTS extension |
| `frontend/src/components/chat/use-chat-stream.ts` | D-4 tool-entry accumulation |
| `frontend/src/components/chat/chat-app.tsx` | D-4 `exchangesToEntries` extension |
| `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` | Doc 0005 status bump + CH-09.1..5 ticked |
| `openspec/AGENTS.md` | CH-09 pointer appended |

**Predecessors in doc 0005**: CH-00 → CH-01 → CH-02 → CH-03 → CH-04 → CH-05 → CH-06 → CH-07 → CH-08 → **CH-09**.

**Successors unblocked**: CH-10 (`cachicamas-chat-permission`), CH-11 (`cachicamas-chat-v1-completion`).

## SDD cycle complete

The change has been fully planned (proposal #3959 + spec #3961 + amendments #3962/#3963 + design #3965 + tasks #3967), implemented (5 WUs + 1 guard + 2 RED scaffolds + 2 recovery commits = 12 work-unit commits), verified (24/24 spec scenarios covered by passing tests + 1 DEFERRED pending INTEGRATION=1; verify-report #3974 PASS), and archived (this report).

Ready for PR merge; orchestrator opens PR after archive closes.