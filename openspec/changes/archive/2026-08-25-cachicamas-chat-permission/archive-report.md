# Archive Report — cachicamas-chat-permission (CH-10)

> **Change**: `cachicamas-chat-permission` · **CH-10** (Wave 3, 11 of 12) of doc 0005
> **Archived**: 2026-08-25
> **Branch**: `feat/chat-permission-ch10` · based at `b4562280` · 12 commits through `b26bde3c`
> **Evidence basis**: apply-phase self-report ONLY (engram `#3994`) — **sdd-verify was skipped by explicit user decision**, transmitted to this archive pass by the orchestrator. No independent verification exists. Nothing in this report should be read as an independently confirmed PASS; where this pass could confirm a fact structurally on disk (files, commits, diffs), it says so explicitly.

## What shipped

- **Chat-owned permission port** (R-CPM-001 / D-1): `chat.PermissionPolicy` declared as a type alias of `agent.PermissionPolicy`; `Config.PermissionPolicy` required with typed `ErrNilPermissionPolicy` refusal; composition-root-only construction of `chat.NewDefaultPermissionPolicy(deferToolNames)` at `cmd/chat/main.go`.
- **Default v1 policy** (R-CPM-002 / D-2): per-tool defer rule (`summarize_conversation` defers; everything else allows synchronously); `AllowAlways → AllowOnce` and `ModifyInput → Defer` collapses inside the default; `ErrDecisionAlreadyMade` typed on double-decision.
- **Two new SSE wire variants** (R-CPM-003 / D-3): `permission.decision.required` + `permission.decision.made` on the closed `WireEvent` union; chat-owned 2-value outcome vocabulary via the D-12 Layer-2 collapse (4 → 2); `permission_resolution_remembered` dropped at the wire; `wireFrameName` switch extended to 11 arms.
- **HTTP reverse-channel** (R-CPM-004 / D-10): `POST /api/agent/turns/:id/permissions/:callID` behind `identityMiddleware` with cross-participant 403, stray-callID 404, malformed-body 422, double-click 409 naming `callID` (S-CPM-017a.5 design-time discovery), then `Conversation.Scheduler().WakeParked(callID)` (D-11 method-only accessor).
- **Frontend affordance** (R-CPM-005 / D-4): inline `hold` row wired end-to-end — `parseTranscript` extended to 11 arms, `KNOWN_EVENTS` to 11 entries, hold-entry accumulation + by-`wireCallId` correlation in `use-chat-stream.ts`, `exchangesToEntries` reload projection at issuance position, `onDecide$` plumbed to `<TranscriptLine>`.
- **Persistence widening** (R-CPM-006 / R-CCS-017/018 / NFR-CCS-009 / D-5+D-9): `Exchange.PermissionDecisions []PermissionDecisionRecord`; 4th additive port method `UpdateSummary`; forward-only migrations `0003_summarize.sql` (ADD COLUMN nullable) + `0004_permission_decisions.sql` (sibling table, FK CASCADE, lookup index); defensive copy via `copyPermissionDecisionRecords`.
- **F-CPM-001 projector-accumulator fix** (R-CPM-007 / D-6, T-07): three parallel accumulators (`toolCalls`, `toolResults`, `permissionDecisions`) threaded into `buildTerminalExchange` so production reload surfaces tool + permission activity — without it both CH-09's and CH-10's widening were theatrical. **Code shipped; its proof test did not (see Unverified claims).**
- **D-8 deny-state collapse** (R-CPM-008): projector suppresses the matching `tool.result` after `permission.decision.made{outcome: "deny"}` via a `deniedSet` accumulator; frontend mirrors the correlation for the reload path. Closes F-CPM-002/003.
- **Guards** (T-08): substrate guard runs inside `make test` (`git diff --stat main..HEAD -- backend/agent/src/agent/` must be empty); wire-fragmentation lockstep test (11 names across Go switch / TS switch / KNOWN_EVENTS); AGENTS.md CH-10 pointer appended.
- **Doc 0005 closure** (T-09): status bumped to "11 of 12 milestones shipped"; charter row ticked at `:1002`; completion-checklist rows ticked; archive folder created.

## Evidence gate (as reported by apply in #3994, NOT independently re-verified)

| Gate | Status | Note |
|---|---|---|
| backend `make test` | GREEN | reported "17/17 packages (**cached**)" — note: cached runs are NOT evidence under this repo's own discipline; no uncached re-run is on record |
| backend `make lint` | GREEN | 0 issues |
| backend `make build/chat` | GREEN | `./bin/chat` produced |
| frontend `pnpm test:ci` | GREEN | vitest, 594/594 |
| frontend `pnpm lint` | GREEN | 0 errors |
| frontend `pnpm build.types` | GREEN | tsc clean |
| INTEGRATION postgres round-trip | NOT_EXERCISED | postgres unavailable in this environment |
| substrate diff empty | YES | independently confirmed by this archive pass: `git diff --stat main..HEAD -- backend/agent/src/agent/` is empty |

## Scenarios covered

Per apply's report (#3994): all scenarios covered, stated there as "40". The distinct scenario IDs named by the tasks/spec artifacts are **37**:

S-CPM-001, S-CPM-002, S-CPM-003, S-CPM-004, S-CPM-005, S-CPM-006, S-CPM-007, S-CPM-008, S-CPM-009, S-CPM-010, S-CPM-011, S-CPM-012, S-CPM-013, S-CPM-014, S-CPM-015, S-CPM-016, S-CPM-017, S-CPM-018, S-CPM-019, S-CPM-020, S-CPM-021, S-CPM-022, S-CPM-023, S-CPM-024, S-CPM-017a.5, S-CCS-023, S-CCS-024, S-CCS-025, S-FCL-018, S-FCL-019, S-FCL-020, S-FCL-021, S-FCL-022, S-CTT-101, S-CTT-102, S-CTT-103, S-CTT-104.

**Recorded contradiction (not silently resolved)**: #3994 states "All 40 scenarios covered"; the spec header (#3989) computes 26 CPM-side scenarios; strict ID enumeration yields 37 distinct IDs. The arithmetic differs across sources; none of the three numbers was independently audited because verify was skipped.

**On-disk contradiction with #3994's coverage claim**: S-CPM-017/S-CPM-018 have **no tests on disk**. The planned `projection_accumulators_test.go` does not exist; T-07 commit `b918abba` modified only `projection.go` (production code); no file under `backend/agent/src/chat/` references those scenario IDs in a test, and no test calls `buildTerminalExchange`. Every other scenario family has a located test surface (permission policy / HTTP handler / projection SSE shapes / store round-trip / frontend specs). See Unverified claims.

## Spec defects resolved during this change

- **F-CPM-001** (CH-09 inherited projector-accumulator defect, CRITICAL) — closed at T-07 (WU-5, commit `b918abba`): accumulators threaded into `buildTerminalExchange`. Code landed; closure proof test missing (see Unverified claims).
- **F-CPM-002/003** (deny-state mapping) — closed at design time via D-8 collapse; implemented in T-04 (projector suppression, `deniedSet`) and T-06 (frontend correlation). Test coverage exists: `TestProjection_DenySuppressesMatchingToolResult`, `use-chat-stream.spec.tsx` deny paths.
- **S-CPM-001 type alias byte-identity (F-CPM-004)** — apply corrected the test logic (compare the two interface declarations' reflect.Types directly, not `reflect.TypeOf` of an assigned value, which returns the dynamic concrete type); spec wording preserved at apply time and amended NOW at close-out in `cachicamas-chat-permission/spec.md` § S-CPM-001 + defect register entry. Mirrors F-CHT-9.2's wording-only resolution shape.
- **S-CTS-007 mirror "compile error" claim (F-CPM-005)** — Go 1.26 does not enforce type-switch exhaustiveness when a `default` arm exists; a missing `wireFrameName` case panics at runtime (apply's RED scaffold #1 landed as runtime panic, commit `75724077`). Spec wording amended NOW at close-out in R-CPM-003 + NFR-CPM-003 + defect register, mirrored into the REQ-12 rationale line of the `frontend-chat-layer1` CH-10 amendment. TypeScript-side `assertNever` compile-error claims are correct and unchanged.

## Spec sync performed at archive

- Primary spec `openspec/specs/cachicamas-chat-permission/spec.md`: F-CPM-004 + F-CPM-005 wording amendments applied; defect register extended; synced byte-identical into the archive folder's `specs/cachicamas-chat-permission/spec.md`.
- Additive amendments verified present and complete: `chat-conversation-store/spec.md` (R-CCS-017/018 + NFR-CCS-009 + S-CCS-023..025 + CH-10 amendment header + untemporal-invariant register addition) and `frontend-chat-layer1/spec.md` (REQ-12/13 with D-7 "new variants, not new fields" wording + S-FCL-018..022 + register addition). Existing requirements byte-unchanged per the amendment headers.
- **Note**: all three spec files were UNCOMMITTED working-tree state until this archive commit — no branch commit had touched `openspec/specs/` despite T-09's "spec promotion" description and the tasks artifact's claim that specs were "already present at b4562280 (committed during spec phase)". They existed only as untracked/unstaged files in the worktree. This archive commit is their first landing on the branch. Recorded here so the audit trail explains why `git log main..HEAD -- openspec/specs/` was empty before it.

## Unverified claims (carry-forward risk)

Attributed to their source and time — these ride into CH-11 (end-to-end acceptance); if CH-11 catches a defect here, trace back to CH-10's T-04 / T-05b / T-07:

1. **F-CPM-001 closure proof (S-CPM-017/018) — NO unit-level test exists.** Per apply-progress #3994 (at apply time), T-07 was marked DONE and the live-transcript-byte-equals-reload claim was asserted; on-disk inspection at archive time found no `projection_accumulators_test.go`, no test referencing S-CPM-017/018, and no test exercising `buildTerminalExchange`. This is stronger risk than "unit-level tests pass": there is no unit-level proof at all. The orchestrator's launch assumption that unit tests cover these scenarios was not borne out by the repository.
2. **Live transcript byte-equals reload transcript end-to-end (S-CPM-016/017 family)** — untested for the same reason; only the projector-side SSE shape tests and store round-trip fixtures exist.
3. **Postgres sibling-table round-trip across processes (S-CCS-023 INTEGRATION variant, S-CCS-021-style)** — INTEGRATION-gated; postgres unavailable in this environment; never exercised.
4. **D-8 deny-collapse under concurrent `Schedule`** — unit-level projector suppression test passes per #3994; full concurrency proof never exercised.
5. **Backend `make test` ran CACHED at final verification** ("17/17 packages (cached)" per #3994) — violates the repo's "cached runs are NOT evidence" rule; no uncached final run is on record.
6. **Double-click race under real concurrency (S-CPM-017a.5)** — asserted at unit level (sequential second POST); true race window never exercised under `-race` with goroutines, per available records.

## Follow-up debt

- **Missing test file**: `backend/agent/src/chat/projection_accumulators_test.go` (or equivalent) implementing S-CPM-017/018 as planned in tasks #3992 T-07. Recommended: land in CH-11's opening PR before the deterministic acceptance, or as an immediate follow-up commit on this branch.
- **Uncached evidence re-run**: one uncached `make test` + frontend suite against the merged branch would convert #3994's cached claims into valid evidence. Cheap; recommended for the PR pipeline run.
- **Deferral register reconciliation** (done here, recorded): doc 0005's deferred register carried only the remembered-decisions row from CH-10's two charter out-of-scope items; the "permission mode selector" row was added at archive close-out (this commit), attributed inline.
- **Scenario-count arithmetic**: reconcile the 40-vs-37-vs-26 counts across tasks/spec/apply artifacts when CH-11 builds its acceptance matrix, so the completion checklist cites one audited number.

## Charter closure

- Charter acceptance (`0005:944`) implemented across R-CPM-001..008 per the mapping table in the primary spec; defer → approve → tool result on stream; decline → refusal recorded, no execution (projector-suppressed tool entry).
- Doc 0005 status line reads "**11 of 12** milestones shipped"; CH-10 charter row ticked at `:1002` with the full deliverable summary; completion checklist rows through CH-10 ticked; CH-11 rows remain open.
- Traceability spine updated: `R-15` and research item 6 map to CH-10 at `:1076` region (pre-existing from planning; verified present).
- Deferral register: remembered-decisions row pre-existing; mode-selector row added at close-out (see Follow-up debt).

## Artifact lineage (engram observation IDs)

preflight/decisions `#3983` · explore `#3985` · proposal `#3987` + `#3988` · spec `#3989` · design `#3990` + `#3991` · tasks `#3992` · apply-progress FINAL `#3994` · archive (this report, persisted as `sdd/cachicamas-chat-permission/archive-report`). No verify-report observation exists — verify phase skipped by user decision. Pattern precedents: CH-09 `#3964` (F-CHT-9.1), `#3965` (design), `#3967` (tasks), `#3945` (apply discipline), `#3974-#3977` (verify recovery pattern that would have applied here had verify run).

## Next milestone

CH-11 (`cachicamas-chat-v1-completion`) is unblocked and is precisely the phase whose deterministic acceptance retires the unverified claims listed above.
