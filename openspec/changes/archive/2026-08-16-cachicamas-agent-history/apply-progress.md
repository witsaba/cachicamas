# Apply Progress: AG-12 — History and the pairing invariant

**Change**: `cachicamas-agent-history` · **Mode**: Strict TDD · **Status**: 43/44 tasks complete
**Worktree**: `feat/agent-layer2-wave3-ag12` · **HEAD**: `9ffd8b6c` · **Commits**: 10

## Budget overrun (report prominently, not silently absorbed)

`git diff --stat origin/main` = **2916 insertions + 7 deletions = 2923 changed lines**, 19 files.

This **exceeds the recorded runtime attempt authority ceiling of 2400** by 523 lines (~22%), and
exceeds the pre-apply forecast (1100–1600, `tasks.md` Review Workload Forecast) by ~1.8x.

| Component | Lines | Forecast bucket | Notes |
|---|---|---|---|
| SDD change-folder markdown (proposal.md 178, design.md 346, exploration.md 119, tasks.md 193) | 836 | "SDD markdown: 450–700" | Already exceeded its own forecast bucket before apply began; these were pre-existing untracked files at session start, not authored during apply |
| Promoted `openspec/specs/` copies (agent-history/spec.md 207 new file, 2 delta amendments 14 lines) | 221 | **not itemized** in the original forecast at all | Inherent cost of Task 11.2's promotion step — a near-duplicate of the 208-line change-folder spec is unavoidable once promoted |
| Go production + test code (history.go 432, history_test.go 554, history_synthesis_test.go 240, history_surface_guard_test.go 331) | 1557 | "history.go 250–350, history_test.go 350–500" | Design.md's own file list mandates exactly these 3 test files; the forecast blended them into one bucket that undercounted the closed-route guard's own weight |
| Substrate/doc.go touches (doc.go 1, doc_contract_guard_test.go 4, loop_test.go 21, loop_hook_test.go 21) | 47 | "~8 + ~6" | Includes the apply-phase substrate-filter correction (see below) |
| docs/architecture status bump | 4 | — | |

None of the three largest drivers (SDD markdown, spec promotion, the 3-file test split) were
things this apply session could have shrunk without either violating `design.md`'s committed file
list or leaving a `tasks.md`-mandated task (11.2) undone. Flagged here for the orchestrator/user to
explicitly acknowledge, request a chained/stacked split for a *future* similar milestone, or accept
the overrun for this PR as `size:exception` already covers in spirit ("extend if needed") even
though the specific 2400-line numeric ceiling was not met.

## Task completion

43/44 tasks `[x]` in `tasks.md`. Task 12.2 (openspec archive move) is **deliberately not performed**
— owned by `sdd-archive` after `sdd-verify` passes, per this apply launch's explicit scope.

## TDD Cycle Evidence

| Task(s) | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1–1.4 | `history_test.go` | Unit | N/A (new file) | ✅ Written — compile error `undefined: agent.NewHistory` | ✅ Passed | ✅ 2 scenarios (order, no-alias) | ➖ None needed |
| 2.1–2.4 | `history_test.go` | Unit | ✅ 2/2 (Phase 1 tests) | ✅ Written — `got nil error, want an *ai.Violation...` | ✅ Passed | ✅ orphan-reject + accepted-after-match | ➖ None needed |
| 2.3 (bite S-HIS-012) | `history.go` scratch | Unit | ✅ 4/4 | ✅ Written+applied — re-FAILS the same assertion after weakening identity check | ✅ Reverted, re-confirmed green | N/A (bite) | N/A |
| 3.1–3.3 | `history_test.go` | Unit | ✅ 4/4 | ✅ Written — `h.CloseTurn undefined` | ✅ Passed | ✅ 3 scenarios (unclosed/all-closed/two-unclosed-first-position) | ➖ None needed |
| 4.1–4.4 | `history_test.go` | Unit | ✅ 7/7 | ✅ Written — `undefined: agent.NewSeededHistory` (×4 sites) | ✅ Passed | ✅ 5 scenarios (valid/orphan/zero-unusable/signature/open-call-accepted) | ➖ None needed |
| 5.1–5.2 | `history_test.go` | Unit | ✅ 12/12 | ➖ Already-GREEN pin (no new production code; Phase 1/4 already implement the behavior) — explicitly recorded, not a fabricated RED | ✅ Passed on first run | ✅ 2 scenarios (read-back, identity stability across reads+seed) | ➖ None needed |
| 6.1–6.3 | `history_synthesis_test.go` | Unit | N/A (new file) | ✅ Written — `h.SynthesizeOrphans undefined` (×3 sites) | ✅ Passed | ✅ 3 scenarios (closes-every-orphan, distinguishable-by-origin-only, turn-closes-after) | ➖ None needed |
| 7.1–7.2 | `history_synthesis_test.go` | Unit | ✅ 17/17 | ➖ Already-GREEN pin (Phase 6's open-set logic already covers it) — recorded explicitly | ✅ Passed on first run | ✅ 2 scenarios (exactly-N, second-application-no-op) | ➖ None needed |
| 8.1–8.3 | `history_surface_guard_test.go` | Unit | N/A (new file) | ➖ No production RED possible — pure closed-audit test over an already-complete surface (Phases 1–7); recorded honestly, mirrors the Phase 5/7 pin pattern rather than fabricating a RED | ✅ Passed on first run | ✅ 4 driver-map routes (Append/NewSeededHistory/CloseTurn/SynthesizeOrphans) + zero-value proof each | ➖ None needed |
| 8.4–8.5 (bite S-HIS-031) | `history.go` scratch (`ScratchAppend`) | Unit | ✅ 22/22 | ✅ Written+applied — FAILS naming `ScratchAppend` as unenumerated | ✅ Reverted; `git diff` byte-empty vs prior commit, re-confirmed green | N/A (bite) | N/A |
| 9.1–9.3 | `doc_contract_guard_test.go` + `doc.go` | Unit | ✅ (existing guard passing) | ✅ Written — `found 6 of 7 rows` | ✅ Passed at 7 rows | ➖ Single scenario (structural table entry) | ➖ None needed |
| 9.4 (bite S-HIS-081) | `doc_contract_guard_test.go` scratch | Unit | ✅ | ✅ Written+applied — FAILS `found 7 of 6 rows` naming the unexpected row | ✅ Restored; re-confirmed green | N/A (bite) | N/A |
| 10.1–10.4 | N/A (verification only) | N/A | N/A | N/A | ✅ All 4 sub-checks confirmed by direct command | N/A | N/A |
| 11.1–11.3 | N/A (openspec promotion) | N/A | N/A | N/A | ✅ Diffs verified scoped to named blocks only | N/A | N/A |
| 12.1 | N/A (doc edit) | N/A | N/A | N/A | ✅ Diff verified exactly 2 lines | N/A | N/A |
| 13.1–13.4 | N/A (final gates) | N/A | N/A | N/A | ✅ test/lint/build/vuln-check all green | N/A | N/A |
| Apply-phase substrate fix | `loop_test.go`/`loop_hook_test.go` | N/A | ✅ (discovered via genuine RED) | ✅ Discovered — `TestTurn_SubstrateUntouched`/`TestTurn_PreRequestHook_SubstrateUntouched` genuinely FAILED after the doc.go/doc_contract_guard_test.go edit, verbatim diffs captured | ✅ Widened both filters, re-confirmed green | N/A | N/A |
| Final lint cleanup | `history.go`, `history_synthesis_test.go` | N/A | ✅ (`make lint` after cache clean) | ✅ 2 real findings (`revive: package-comments`, `unused`) confirmed not cache-stale | ✅ Fixed both, `0 issues.` | N/A | N/A |

### Test Summary

- **Total top-level test functions added**: 22 (history_test.go: 14, history_synthesis_test.go: 5, history_surface_guard_test.go: 2 top-level + 4 sub-tests via `t.Run`)
- **Total tests passing**: every test in `backend/agent/src/agent` (200+), full module 12/12 packages, all under `-race`
- **Layers used**: Unit only (History is pure in-memory, Threat Matrix N/A per design.md)
- **Bites**: 3 (S-HIS-012, S-HIS-031, S-HIS-081) — all RED-recorded then reverted, diff-verified clean
- **Pure functions created**: `resolveOpenSet` (the core pairing-validation function) is a pure function — no receiver mutation, returns the would-be delta

## Final gate results (verbatim)

- `go test -race -count=1 ./...` (whole module, non-cached): 12/12 packages `ok`, including `ai/openaicompat` at ~173s (network-independent fixtures)
- `make lint` (after `./bin/golangci-lint cache clean`): `0 issues.`
- `make build`: clean, no output (success)
- `make vuln-check`: exit code 0, zero `finding` entries across the whole `./...` closure — cleaner than the AG-10/AG-11 precedent's "pre-existing stdlib advisories" WARNING, plausibly due to the earlier go1.26.5→1.26.6 bump (`d2b8228d`) that already cleared 8 stdlib `GO-2026-*` advisories

## Non-negotiable constraints — confirmed held

- `git diff origin/main -- backend/agent/src/ai/ backend/agent/src/agent/loop.go backend/agent/src/agent/scheduler.go backend/agent/go.mod backend/agent/go.sum` → 0 lines
- Exactly 4 new files under `backend/agent/src/agent/`: `history.go`, `history_test.go`, `history_synthesis_test.go`, `history_surface_guard_test.go`
- Both substrate filters byte-in-sync (verified via diff)

## Apply-phase discoveries (deviations from the plan, noted not silently absorbed)

1. **Substrate filter gap for `doc.go`/`doc_contract_guard_test.go`** (Phase 9): the tasks.md
   "Substrate Filter Closure" section only anticipated the 4 brand-new files. Landing `L2C-07`
   modifies the PRE-EXISTING `doc.go`/`doc_contract_guard_test.go`, which genuinely tripped
   `TestTurn_SubstrateUntouched` and `TestTurn_PreRequestHook_SubstrateUntouched` — the first doc-row
   amendment since those guards existed (introduced at AG-08; AG-04/05/06 predate them). Fixed by
   widening both filters with `/doc.go` and `/doc_contract_guard_test.go`, "released for AG-12
   only", mirroring AG-11's own `turn_events.go`/`failure.go` precedent exactly. Recorded as its own
   Engram discovery observation for future milestones that add an L2C row.
2. **Stale coverage table** (Phase 11, per the apply launch's own "one extra fix to fold in"): the
   change-folder spec's per-node coverage table said 14 AG-12.1 scenarios; the authoritative count
   (recounted from the file) is 16. Corrected in both the source and the promoted copy before
   promotion, so they agree.
3. **Budget overrun**: see the dedicated section above.

## Commit list

1. `b947d962` chore(agent): AG-12 track SDD change artifacts
2. `2918b71a` feat(agent): AG-12.1 history skeleton — order preservation + no-alias reads
3. `64e563a1` feat(agent): AG-12.1 pairing invariant — orphan-result + turn-close rejection
4. `cb50fd6b` feat(agent): AG-12.1 seeded construction + read-only identity pins
5. `b9234654` feat(agent): AG-12.2 orphan synthesis — idempotent and total
6. `ebcdb322` test(agent): AG-12 closed-route guard — R-HIS-004 enumeration, S-HIS-030/042
7. `c1c46f5b` docs(agent): AG-12 L2C-07 doc-contract row + substrate filter correction
8. `62d8c7bb` docs(openspec): AG-12 promote agent-history + apply cross-cut spec deltas
9. `b7bc542c` docs(architecture): AG-12 bump the doc 0003 milestone counters
10. `9ffd8b6c` fix(agent): AG-12 final-gate lint cleanup
