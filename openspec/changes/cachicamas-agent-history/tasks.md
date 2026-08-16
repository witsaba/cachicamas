# Tasks: AG-12 — History and the pairing invariant

> Scenario count: **22 spec scenarios + 3 bites = 25 total** across 9 requirements (`R-HIS-001..009`) plus 5
> non-functional requirements, and 2 MODIFIED deltas (`R-AEV-014`, `R-AGP-002`). This corrects the per-node
> table in `spec.md`'s Coverage section (14/2, 5/0, 1/1), which predates the correction round's addition of
> `S-HIS-054`; requirement-by-requirement counts below are the authoritative ones. Design decisions in
> `design.md`'s Architecture Decisions table are binding; this file does not re-derive or contradict them.

## Substrate Filter Closure (authoritative — closes `R-HIS-004`/`NFR-HIS-004`)

`filterOutLoopFiles` (`loop_test.go:831`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907`) MUST widen
with exactly these four exact-filename suffixes, byte-in-sync between the two functions, no
wildcard/prefix/directory pattern (the AG-11 recorded lesson):

```
/history.go
/history_test.go
/history_synthesis_test.go
/history_surface_guard_test.go
```

Both filters currently list the AG-10 (`permission_protocol.go`/`permission_protocol_test.go`) and AG-10
remediation (`loop_permission_e2e_test.go`/`permission_policy_helpers_test.go`) suffixes; AG-12 appends this
four-entry list with an AG-12 widening comment mirroring the existing AG-10 comment block. Land in the SAME
commit as the file that first needs it.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1100–1600 (proposal's `Changed-line forecast`; `history.go` 250–350, `history_test.go` 350–500, `doc.go`+guard ~8, filters ~6, SDD markdown 450–700) |
| Default repo budget (`openspec/config.yaml`) | 400 |
| Session-extended pre-authorized ceiling | 1000 (`size:exception`, extend-if-needed) |
| 400-line budget risk | High — exceeds even the pre-authorized 1000-line ceiling |
| Chained PRs recommended | No |
| Suggested split | single PR — AG-12 only |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

All units ship in ONE PR (`size:exception` pre-authorized). Runtime harness is N/A across the board: `History`
is a pure in-memory type, no I/O (Threat Matrix: N/A, design.md).

| Unit | Goal | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | Foundation: skeleton + order preservation + substrate filters (Phase 1) | `go test -run "TestHistory_OrderPreserved|TestHistory_ReadDoesNotAlias" -race ./src/agent/...` | N/A | revert `history.go`/`history_test.go` skeleton + filter diffs |
| 2 | Pairing invariant: orphan-result + turn-close rejection (Phase 2–3) | `go test -run "TestHistory_OrphanedResult|TestHistory_ResultAfterMatchingCall|TestHistory_UnclosedCallAtTurnClose|TestHistory_AllCallsClosed|TestHistory_TwoUnclosedCalls" -race ./src/agent/...` | N/A | revert rule-3/rule-4 + `CloseTurn` diffs |
| 3 | Seeded construction + read-only views (Phase 4–5) | `go test -run "TestNewSeededHistory_|TestHistory_ReadBack|TestHistory_EntryIdentity" -race ./src/agent/...` | N/A | revert `NewSeededHistory` diff |
| 4 | Orphan synthesis + idempotence (Phase 6–7) | `go test -run "TestHistory_SynthesizeOrphans|TestHistory_SynthesizedVsReal" -race ./src/agent/...` | N/A | revert `history_synthesis_test.go` + `SynthesizeOrphans` diff |
| 5 | Closed-route guard (Phase 8) | `go test -run "TestHistoryRouteGuard_|TestHistory_NoBypass_" -race ./src/agent/...` | N/A | revert `history_surface_guard_test.go` |
| 6 | `L2C-07` doc contract (Phase 9) | `go test -run TestLayer2DocContract_MatchesTheCommittedTable -race ./src/agent/...` | N/A | revert `doc.go`/`doc_contract_guard_test.go` diffs |
| 7 | Cross-cut, spec promotion, docs, archive (Phase 10–12) | `cd backend/agent && make test` + `git diff <merge-base> -- backend/agent/src/agent/` | N/A | revert promotion/docs/archive commits |
| 8 | Final gates (Phase 13) | `cd backend/agent && make test && make lint && make build && make vuln-check` | N/A | verification only, no code |

## Phase 1: Foundation — package skeleton, order preservation, substrate registration

- [ ] 1.1 RED — create `history_test.go` (`package agent_test`): `TestHistory_OrderPreserved` (S-HIS-001), `TestHistory_ReadDoesNotAliasInternalStorage` (S-HIS-002). Expect FAIL: compile error, `agent.History`/`NewHistory`/`Append`/`Entries` undefined.
- [ ] 1.2 GREEN — create `history.go`: `EntryOrigin`, `EntryID`, `Entry` + read-only accessors, `openCall`, `History`, `commitOp`, `commit` (rules 1–2 only: `!constructed` → `ai.ErrEmpty` at `"history"`; message not `ai.NewMessage`-constructed → `ai.ErrEmpty` at `messages[i]`), `NewHistory()`, `Append()`, `Entries()` (fresh slice each call), `Len()`.
- [ ] 1.3 Register substrate filters — SAME commit as 1.1/1.2: widen `filterOutLoopFiles` and `filterOutLoopHookFiles` per the Substrate Filter Closure section, byte-in-sync.
- [ ] 1.4 GREEN confirm — `go test -run "TestHistory_OrderPreserved|TestHistory_ReadDoesNotAlias" -race ./src/agent/...` passes; `TestTurn_SubstrateUntouched`/`TestTurn_PreRequestHook_SubstrateUntouched` still pass.

## Phase 2: Orphan-result rejection (`R-HIS-002`)

- [ ] 2.1 RED — `history_test.go`: `TestHistory_OrphanedResult_RejectedTyped` (S-HIS-010), `TestHistory_ResultAfterMatchingCall_Accepted` (S-HIS-011). Expect FAIL: no pairing rule yet, orphan silently accepted.
- [ ] 2.2 GREEN — `commit` rule 3: track the open set; a `ToolCall` part joins it; a `ToolResult` part must name an open call, else `ai.Invalid(ai.ErrUnresolvedReference, ai.At("messages"), ai.AtIndex(i), ai.At("content"), ai.AtIndex(j))`; a duplicate result for an already-answered call fails by the same rule.
- [ ] 2.3 RED bite S-HIS-012 — scratch-weaken rule 3 to accept any result once ≥1 call exists (ignore identity); rerun `TestHistory_OrphanedResult_RejectedTyped`; confirm FAIL reporting the accepted orphan; record output in `apply-progress.md`; revert.
- [ ] 2.4 GREEN confirm — `go test -run "TestHistory_OrphanedResult|TestHistory_ResultAfterMatchingCall" -race ./src/agent/...`.

## Phase 3: Turn-close rejection (`R-HIS-003`)

- [ ] 3.1 RED — `TestHistory_UnclosedCallAtTurnClose_RejectedTyped` (S-HIS-020), `TestHistory_AllCallsClosed_TurnCloseSucceeds` (S-HIS-021), `TestHistory_TwoUnclosedCalls_NamesFirstOffendingPosition` (S-HIS-022). Expect FAIL: `CloseTurn` undefined.
- [ ] 3.2 GREEN — `history.go`: `commitCloseTurn` op, `CloseTurn()` method, rule 4 (open set must be empty, else `ai.Invalid(ai.ErrEmpty, ai.At("messages"), ai.AtIndex(i), ai.At("content"), ai.AtIndex(j), ai.At("result"))` at the first-issued open call); empty open set → no-op, returns nil.
- [ ] 3.3 GREEN confirm — `go test -run "TestHistory_UnclosedCallAtTurnClose|TestHistory_AllCallsClosed|TestHistory_TwoUnclosedCalls" -race ./src/agent/...`.

## Phase 4: Seeded construction (`R-HIS-006`)

- [ ] 4.1 RED — `TestNewSeededHistory_ValidSeed_Accepted` (S-HIS-050), `TestNewSeededHistory_OrphanedResult_RejectedFirstOffendingPosition` (S-HIS-051, result direction only), `TestNewSeededHistory_RejectedSeed_ZeroValueUnusable` (S-HIS-052), `TestNewSeededHistory_SignatureAcceptsOnlyMessages` (S-HIS-053), `TestNewSeededHistory_EndsInOpenCall_AcceptedThenCloseTurnRejects` (S-HIS-054). Expect FAIL: `NewSeededHistory` undefined.
- [ ] 4.2 GREEN — `history.go`: `NewSeededHistory(messages []ai.Message) (*History, error)` — `NewHistory()`, then `commit(commitAppend, m, EntryOriginAppended)` per message in order; first violation aborts and returns `(nil, err)`, positioned via `ai.AtIndex(i)`; accepts a seed ending in open calls (rule 3 only — `commitCloseTurn`/rule 4 is NEVER applied at seed time); rejects only an orphaned RESULT.
- [ ] 4.3 Pin S-HIS-054 against the corrected reading: a seed ending in an open call is ACCEPTED; `CloseTurn` on that seeded history THEN rejects with the identical rule class and positional shape as S-HIS-020.
- [ ] 4.4 GREEN confirm — `go test -run "TestNewSeededHistory_" -race ./src/agent/...`.

## Phase 5: Read-only views, stable identity (`R-HIS-005`, excl. S-HIS-042)

- [ ] 5.1 RED — `TestHistory_ReadBack_UnmodifiedValuesInOrder` (S-HIS-040), `TestHistory_EntryIdentity_StableAcrossReadsAndSeed` (S-HIS-041 — mutate a returned `Entries()` slice, re-read unchanged; construct a second `History` via `NewSeededHistory` over the same ordered seed, assert identical `EntryID`s to the appended original). Pin explicitly even if already GREEN from 1.2/4.2.
- [ ] 5.2 GREEN confirm — `go test -run "TestHistory_ReadBack|TestHistory_EntryIdentity" -race ./src/agent/...`.

## Phase 6: Orphan synthesis (`R-HIS-007`)

- [ ] 6.1 RED — create `history_synthesis_test.go` (`package agent_test`): `TestHistory_SynthesizeOrphans_ClosesEveryOrphanDistinguishableByOrigin` (S-HIS-060), `TestHistory_SynthesizedVsReal_DistinguishableByOriginOnly` (S-HIS-061 — byte-identical content, `Failed()` set identically on both, only `Origin()` differs; assertion reads neither content nor `Failed()`), `TestHistory_SynthesizeOrphans_TurnClosesAfterSynthesis` (S-HIS-062). Expect FAIL: `SynthesizeOrphans` undefined.
- [ ] 6.2 GREEN — `history.go`: `SynthesizeOrphans() (int, error)` — snapshot the open set in issuance order; build ALL N `ai.NewToolFailure(callID, synthesizedInterruptionContent)` wrapped in `ai.RoleTool` messages BEFORE committing any; commit each via `commit(commitAppend, m, EntryOriginSynthesized)`; add the unexported `synthesizedInterruptionContent` constant.
- [ ] 6.3 GREEN confirm — `go test -run "TestHistory_SynthesizeOrphans|TestHistory_SynthesizedVsReal" -race ./src/agent/...`.

## Phase 7: Idempotent and total synthesis (`R-HIS-008`)

- [ ] 7.1 RED — `TestHistory_SynthesizeOrphans_ClosesExactlyN` (S-HIS-070, N≥2 orphans interleaved with already-matched calls), `TestHistory_SynthesizeOrphans_SecondApplicationNoOp` (S-HIS-071). Pin explicitly even if already GREEN from 6.2's open-set logic.
- [ ] 7.2 GREEN confirm — `go test -run "TestHistory_SynthesizeOrphans_ClosesExactlyN|SecondApplicationNoOp" -race ./src/agent/...`.

## Phase 8: Closed-route guard (`R-HIS-004`, `S-HIS-042`)

- [ ] 8.1 RED — create `history_surface_guard_test.go` (`package agent_test`): `routeClass`/`historyRoute` types, `expectedHistoryRoutes` (11 rows per `design.md`), method-set enumeration via `reflect.TypeOf(*History)`/`reflect.TypeOf(Entry)`, package-function enumeration via `go/parser` + `runtime.Caller(0)`. `TestHistoryRouteGuard_SurfaceMatchesExpectedTable` — set-equal diff both directions, plus asserting every `method:Entry` row is `routeReadOnly`-only (closes S-HIS-042).
- [ ] 8.2 RED — same file: `TestHistory_NoBypass_EveryMutatingRouteRejectsOrphaningSequence` (S-HIS-030) — driver map keyed off `expectedHistoryRoutes`' `routeMutating` rows (`Append`, `NewSeededHistory`, `CloseTurn`, `SynthesizeOrphans`); each driver builds an orphaning sequence through that route and asserts typed rejection plus byte-unchanged state; also drives `new(History)` through each route; fails naming any mutating row with no driver.
- [ ] 8.3 GREEN confirm — both tests pass against the complete surface (Phases 1–7 already implement every route).
- [ ] 8.4 RED bite S-HIS-031 — scratch-add `func (h *History) ScratchAppend(m ai.Message) error { h.entries = append(h.entries, Entry{message: m}); return nil }` bypassing `commit`; rerun `TestHistoryRouteGuard_SurfaceMatchesExpectedTable`; confirm FAILS naming `ScratchAppend` as an unenumerated surface route; record output in `apply-progress.md`; revert.
- [ ] 8.5 GREEN confirm — guard file green after revert.

## Phase 9: `L2C-07` doc-contract row (`R-HIS-009`)

- [ ] 9.1 RED — `doc_contract_guard_test.go`: append the 7th entry `{id: "L2C-07", text: "..."}` (exact text from `design.md`'s "The `L2C-07` Row" section) to `expectedLayer2ContractRows`. Expect FAIL: `TestLayer2DocContract_MatchesTheCommittedTable` reports "found 6 of 7 rows" — `doc.go` still has 6 rows.
- [ ] 9.2 GREEN — `doc.go`: append the `L2C-07` tab-indented row (identical exact text) to the guarded doc-comment table, in the SAME PR as 9.1 (`R-AGP-002`).
- [ ] 9.3 GREEN confirm — `TestLayer2DocContract_MatchesTheCommittedTable` passes at 7 rows.
- [ ] 9.4 RED bite S-HIS-081 — scratch-strip the `L2C-07` entry from `expectedLayer2ContractRows` while `doc.go`'s row remains; rerun the guard; confirm FAILS naming the unexpected `doc.go` row; record output; restore the table entry.

## Phase 10: Cross-cut substrate invariants

- [ ] 10.1 Confirm byte-unchanged: `git diff origin/main -- backend/agent/src/ai/ backend/agent/src/agent/loop.go backend/agent/src/agent/scheduler.go backend/agent/go.mod backend/agent/go.sum` empty (`NFR-HIS-003`).
- [ ] 10.2 Merge-base diff scenario: `git diff <merge-base> -- backend/agent/src/agent/` — only files differing: new `history.go`/`history_test.go`/`history_synthesis_test.go`/`history_surface_guard_test.go`, modified `doc.go`/`doc_contract_guard_test.go`/`loop_test.go`/`loop_hook_test.go`; both substrate filters carry the identical 4-suffix set.
- [ ] 10.3 `cd backend/agent && make test` (whole module, `-race`) green — all 22 scenarios + 3 bites recorded.
- [ ] 10.4 Confirm all three new test files are `package agent_test` (`NFR-HIS-001`); confirm every added test is deterministic/hermetic, no network, no filesystem outside `t.TempDir()` (`NFR-HIS-002`).

## Phase 11: Spec deltas and promotion

- [ ] 11.1 Confirm `specs/agent-event-envelope/spec.md` (`S-AEV-122`/`123`, MODIFIED `R-AEV-014`) and `specs/agent-package-scaffold/spec.md` (`S-AGP-010..015`, MODIFIED `R-AGP-002`) already carry the amended text (authored in `sdd-spec`); no further edit needed here.
- [ ] 11.2 At promotion: create `openspec/specs/agent-history/spec.md` from the change-folder ADDED delta (strip the promotion-note header line, keep all 9 requirements / 22 scenarios / 3 bites verbatim); apply the two MODIFIED deltas into `openspec/specs/agent-event-envelope/spec.md` (`R-AEV-014` block only) and `openspec/specs/agent-package-scaffold/spec.md` (`R-AGP-002` block only), leaving every other requirement byte-unchanged.
- [ ] 11.3 Verification: diff each promoted/amended canonical file against its change-folder delta; confirm only the named requirement blocks changed; record diff evidence in `apply-progress.md`.

## Phase 12: Docs and archive

- [ ] 12.1 `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` — flip the AG-12.1 and AG-12.2 acceptance checkboxes, sole owner AG-12; bump the Layer 2 wave/milestone status counters (11/24 → 12/24); do NOT flip any sibling milestone's unrelated checkbox.
- [ ] 12.2 openspec archive — move `openspec/changes/cachicamas-agent-history/` to `openspec/changes/archive/<merge-date>-cachicamas-agent-history/` per AG-09/AG-10/AG-11 precedent naming, AFTER `sdd-verify` (not mid-apply, per AG-11's recorded deviation).

## Phase 13: Final gates

- [ ] 13.1 `cd backend/agent && make test` (race-gated) — all 22 scenarios + 3 bites recorded evidence, whole module green.
- [ ] 13.2 `cd backend/agent && make lint` after `golangci-lint cache clean` — 0 issues.
- [ ] 13.3 `cd backend/agent && make build` — clean.
- [ ] 13.4 `cd backend/agent && make vuln-check` (NOT in `make all`) — record result; accept pre-existing stdlib advisories outside `src/agent/**` as WARNING per AG-10/AG-11 precedent, not a blocker.

## Coverage Table — every `S-HIS-` scenario mapped to a task

| Requirement | Scenario | Task |
|---|---|---|
| R-HIS-001 | S-HIS-001, S-HIS-002 | 1.1, 1.2 |
| R-HIS-002 | S-HIS-010, S-HIS-011 | 2.1, 2.2 |
| R-HIS-002 | S-HIS-012 (bite) | 2.3 |
| R-HIS-003 | S-HIS-020, S-HIS-021, S-HIS-022 | 3.1, 3.2 |
| R-HIS-004 | S-HIS-030 | 8.2 |
| R-HIS-004 | S-HIS-031 (bite) | 8.4 |
| R-HIS-005 | S-HIS-040, S-HIS-041 | 5.1 |
| R-HIS-005 | S-HIS-042 | 8.1 |
| R-HIS-006 | S-HIS-050, S-HIS-051, S-HIS-052, S-HIS-053 | 4.1, 4.2 |
| R-HIS-006 | S-HIS-054 | 4.1, 4.3 |
| R-HIS-007 | S-HIS-060, S-HIS-061, S-HIS-062 | 6.1, 6.2 |
| R-HIS-008 | S-HIS-070, S-HIS-071 | 7.1 |
| R-HIS-009 | S-HIS-080 | 9.1, 9.2 |
| R-HIS-009 | S-HIS-081 (bite) | 9.4 |
| NFR-HIS-001..005 | (cross-cutting) | 10.4, Review Workload Forecast |
| R-AEV-014 (delta) | S-AEV-122, S-AEV-123 | 11.1, 11.2, 11.3 |
| R-AGP-002 (delta) | S-AGP-010..015 | 11.1, 11.2, 11.3 |

## Risks

- **Ordering dependency (Phase 4 before Phase 5)**: `S-HIS-041` needs `NewSeededHistory` to construct the
  second history over the same seed, so seeded construction (Phase 4) MUST land before the read-only-views
  scenario that cross-checks it (Phase 5) — this reorders `R-HIS-005` ahead of `R-HIS-006` relative to the
  spec's own numbering; apply MUST NOT silently skip this dependency.
- **Closed-route guard depends on every route existing** (Phase 8 last among behavior phases): the
  `S-HIS-030` driver map exercises `Append`, `NewSeededHistory`, `CloseTurn` and `SynthesizeOrphans` — all four
  must already be implemented before the guard's audit test is meaningful, so Phase 8 is sequenced after
  Phases 1–7 despite `R-HIS-004` appearing earlier in the spec.
- **Filter-widening landmine**: every RED test added under `backend/agent/src/agent/` trips both substrate
  guards until its filter suffix lands. Task 1.3 MUST widen filters in the SAME commit as 1.1/1.2, never in a
  later separate commit, or `make test` reports an unrelated failure alongside the intended RED signal.
- **`S-HIS-051` reading**: pin strictly against the CORRECTED text — orphaned RESULT only rejects at seed
  time; an unclosed call is legal in a seed and rejects only via `CloseTurn` (`S-HIS-054`). Do not regress to
  the superseded stricter reading.
- **`ai/validation.go`, `loop.go`, `scheduler.go` MUST NOT be modified** — Task 10.1 is the mechanical proof;
  any diff there is an automatic apply-phase failure, not a judgment call.
