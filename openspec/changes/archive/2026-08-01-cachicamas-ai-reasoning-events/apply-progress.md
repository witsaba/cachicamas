# Apply Progress: cachicamas-ai-reasoning-events (AI-17)

> **Status**: implementation complete. 14/14 tasks done across Phases 0–4. `make test` and `make lint` both green. Ready for `sdd-verify`.

## Goal

Implement the AI-17 streamed reasoning block event family: three event kinds
(reasoning block start, reasoning delta, reasoning block end) registered in
AI-14's event registry, a producer-stamped shared block-index space with
AI-16's text blocks, fragment-only deltas with byte-exact reconstruction, no
public accumulator, AI-07's opaque round-trip token carried whole on the
block-end event, and redacted/signature-only stream support. Source: spec
#2204, design #2205, tasks #2206.

## Session context — this was a resume

A prior `sdd-apply` run for this exact change disconnected mid-flight (API
connection error) partway through Phase 3. Verified state at the start of
this session:

- Commit `91168d0` already landed Phases 0–2 (lifecycle + token delivery).
- `reasoning_event_test.go` had an uncommitted, genuine RED state: exactly 2
  tests failing (`TestReasoningBlockEnd_RedactedWithNoToken_IsRejected`,
  `TestReasoningDelta_NonEmptyFragmentOnARedactedBlock_IsRejected`) —
  everything else in the package passed.
- Sibling milestones AI-16 and AI-18 had also landed in this worktree
  (commits `4f63977`, `2b3bcee`, `eaaf7b8`, `5cd0b91`, `20a483d`) — not
  touched by this session; only `reasoning_event.go` / `reasoning_event_test.go`
  were in scope.

This session picked up from that exact RED state: implemented GREEN for the
two failing tests, REFACTOR, then Phase 4 (NFR closeout).

## Branch & base

| Field | Value |
|---|---|
| Branch | `feat/2026-08-01-cachicamas-ai-layer1-wave-2` |
| Worktree | `cachicamas-worktrees/ai-wave-2` |
| Module / package | `backend/agent`, `src/ai` |
| Base for Phases 0–2 | `91168d0` (already committed before this session) |
| Delivery strategy | `exception-ok`, single PR, `size:exception` (per tasks #2206 Review Workload Forecast) |
| Test runner | `make test` from `backend/agent/` (`go test -race -v ./...`) |
| Strict TDD | ACTIVE |

## Commits made (this session)

| # | SHA | Message | Scope |
|---|---|---|---|
| 1 | `8337573` | `feat(ai): land the AI-17.3 redacted and signature-only reasoning streams (AI-17.3)` | GREEN for S-ARE-035/036 against the already-present RED tests for S-ARE-031..039 |
| 2 | `4ceb77c` | `feat(ai): close out AI-17 NFRs - totality and a clean make lint (AI-17 NFR)` | S-ARE-041 totality test, S-ARE-042 failure-reporting test, one-line pre-existing lint fix |

Prior session: `91168d0` (Phases 0–2, already committed, not touched this session).

## Files changed (this session)

| File | Action | What was done |
|---|---|---|
| `backend/agent/src/ai/reasoning_event.go` | Modified | `ReasoningDelta.validate`: added `redacted ∧ non-empty fragment → ErrMisplaced` at "fragment" (S-ARE-036), ordered before the `MaxTextLen` bound check. `ReasoningBlockEnd.validate`: added `redacted ∧ !hasToken → ErrEmpty` at "token" (S-ARE-035), ordered before the `MaxReasoningTokenLen` bound check. Updated `NewReasoningDelta`/`NewReasoningBlockEnd` doc comments (removed stale "once AI-17.3 lands" future tense). Fixed a pre-existing lint gap: added the blank line every sibling file carries between its header comment and `package ai` (revive `package-comments`, 1 issue → 0). |
| `backend/agent/src/ai/reasoning_event_test.go` | Modified | Confirmed the already-present RED tests for S-ARE-031..039 (written pre-disconnect) now pass. Added `TestReasoningEvents_Totality_NoExportedEntryPointPanics` (S-ARE-041, NFR-ARE-B) and `TestReasoningEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel` (S-ARE-042, NFR-ARE-C — reuses `assertViolation` from `tool_call_test.go`, same `ai_test` package, rather than redefining it). No new imports added to either file. |
| `openspec/changes/cachicamas-ai-reasoning-events/tasks.md` | Modified | All 14 tasks marked `[x]`; appended an Evidence log section with the RED→GREEN→REFACTOR table for this session's scope (3.1–4.3) and final verification output. |

`reasoning_content.go` / `reasoning_content_test.go`: **untouched** (NFR-ARE-A) — confirmed `git diff main` empty for both.

## TDD Cycle Evidence

Strict-TDD gate: ACTIVE. Runner: `make test` (`go test -race -v ./...`) from
`backend/agent/`. This session's scope is Phase 3 (3.1 already RED at session
start; 3.2 GREEN + 3.3 REFACTOR done this session) and Phase 4 (4.1–4.3, all
done this session). Phases 0–2 (0.1–2.3) were already GREEN/REFACTOR-complete
in commit `91168d0` before this session; included below for completeness per
the merge protocol.

| Task | Test File | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 0.1–0.2 | `reasoning_event.go`, `event.go` | Unit | N/A (new) | n/a (structural) | ✅ `91168d0` | ➖ n/a | ➖ n/a |
| 1.1–1.3 | `reasoning_event_test.go` | Unit | N/A (new file) | ✅ `91168d0` | ✅ `91168d0` | ✅ multiple cases (S-ARE-001..022) | ✅ `91168d0` |
| 2.1–2.3 | `reasoning_event_test.go` | Unit | ✅ full suite green after 1.x | ✅ `91168d0` | ✅ `91168d0` | ✅ multiple cases (S-ARE-023..030) | ✅ `91168d0` |
| 3.1 | `reasoning_event_test.go` | Unit | ✅ confirmed at session start: `go test -run Reasoning` → exactly 2 FAIL (`...RedactedWithNoToken...`, `...NonEmptyFragmentOnARedactedBlock...`), rest PASS | ✅ Written pre-disconnect (this session verified, did not author) | — (RED-only step) | — | — |
| 3.2 | `reasoning_event.go` | Unit | (same baseline as 3.1) | see 3.1 | ✅ `go test -race -run 'TestReasoningBlockEnd_RedactedWithNoToken_IsRejected\|TestReasoningDelta_NonEmptyFragmentOnARedactedBlock_IsRejected' -v ./...` → both PASS | ✅ S-ARE-031..039 as a group (11 tests) all PASS against the same GREEN change — no Fake It, real conditional logic gated on `r.redacted` | ✅ full-package `go test -race ./...` → PASS, 0 FAIL |
| 3.3 | `reasoning_event_test.go` | Unit | ✅ (see 3.2) | n/a | n/a | n/a | ✅ Confirmed `reconstructFragments`/`reconstructReasoningState` already unexported test-local helpers (S-ARE-021/022 guard this); no production duplication extracted — the two redacted-gated rules use different sentinels/fields, so a shared helper would not reduce duplication |
| 4.1 | `reasoning_event_test.go` | Unit | ✅ full suite green after 3.x | ✅ Written this session, against the already-landed surface (closeout assertions, not new production behavior) | ✅ `go test -race -run 'TestReasoningEvents_Totality_NoExportedEntryPointPanics\|TestReasoningEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel' -v ./...` → both PASS, all subtests PASS | ✅ totality test covers 4 constructors × multiple edge-input tables; failure-reporting test covers 8 rejection scenarios across all 3 kinds | ➖ none needed — reused `assertViolation` (DRY, matches AI-18 precedent) instead of writing new helpers |
| 4.2 | n/a (verification) | n/a | n/a | n/a | ✅ `go.mod` 2 lines, zero `require`; `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` + `TestLayer1_ModuleHasNoDependencies_ZeroRequires` PASS; `git diff main -- reasoning_content.go reasoning_content_test.go` empty | n/a | n/a |
| 4.3 | n/a (final gate) | n/a | n/a | n/a | ✅ `make test` exit 0, 0 FAIL | n/a | ✅ `make lint` exit 0, `0 issues.` (after the blank-line fix) |

### Test Summary

- **Total tests written this session**: 0 new test functions for 3.1 (pre-existing); 2 new test functions for 4.1/4.2 (`TestReasoningEvents_Totality_NoExportedEntryPointPanics`, `TestReasoningEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel`), each with multiple sub-tests/table rows.
- **Total tests passing**: full `backend/agent` module — `ok` for `src/agenttest` and `src/ai`, 0 FAIL.
- **Layers used**: Unit only (pure contract package, no I/O — matches design.md's Testing Strategy table).
- **Approval tests**: None — no refactoring-of-existing-behavior tasks this session.
- **Pure functions touched**: `ReasoningDelta.validate`, `ReasoningBlockEnd.validate` (both pure, no side effects).

## Work Unit Evidence

| Evidence | Value |
|---|---|
| Focused test command and exact result | `go test -race -run 'TestReasoningBlockEnd_RedactedWithNoToken_IsRejected\|TestReasoningDelta_NonEmptyFragmentOnARedactedBlock_IsRejected' -v ./src/ai/...` → `PASS`, both tests green, `ok` in 1.5s |
| Runtime harness command/scenario and exact result | N/A — pure contract package, no I/O, no routing/shell/subprocess boundary (per design.md Threat Matrix: "N/A") |
| Rollback boundary | Revert commits `8337573` and `4ceb77c` (or `git revert` in reverse order); no other files touched. `reasoning_content.go`/`reasoning_content_test.go` and all AI-16/AI-18 files are untouched and unaffected. |

## Deviations from design

None — implementation matches `design.md`'s validation order table exactly:
delta validates `block_index` → redacted-gated `ErrMisplaced` at "fragment" →
`MaxTextLen` bound; end validates `block_index` → redacted-gated `ErrEmpty`
at "token" → `MaxReasoningTokenLen` bound. `R-ARE-013`'s resolution (redacted
block rejects non-empty text via `ErrMisplaced`, per design.md's "KEPT"
decision) is implemented as specified.

## Issues found and fixed

One pre-existing lint gap, not part of the original design/tasks scope:
`reasoning_event.go` (from `91168d0`) was missing the blank line every
sibling file in this package carries between its header comment and
`package ai`, which made `revive`'s `package-comments` rule treat the header
as an incorrectly-worded package doc comment. Fixed with a single blank line
in commit `4ceb77c`. Zero behavior change; `make lint` went from 1 issue to
0.

## Verification results

| Check | Result |
|---|---|
| `go test -race -run Reasoning -v ./...` (before this session's GREEN) | 2 FAIL: `TestReasoningBlockEnd_RedactedWithNoToken_IsRejected`, `TestReasoningDelta_NonEmptyFragmentOnARedactedBlock_IsRejected` |
| Same command, after GREEN | 0 FAIL, all PASS |
| `go test -race ./...` (full module, after Phase 3) | PASS — `agenttest` and `ai` both `ok` |
| `TestReasoningEvents_Totality_NoExportedEntryPointPanics` | PASS |
| `TestReasoningEvents_EveryRejection_ReportsAIsFourViolationWithALandedSentinel` | PASS (8/8 sub-cases) |
| `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` | PASS |
| `TestLayer1_ModuleHasNoDependencies_ZeroRequires` | PASS |
| `git diff main -- reasoning_content.go reasoning_content_test.go` | empty (diff-free) |
| `gofmt -l` (both changed files) | clean (no output) |
| `go vet ./...` | clean |
| `make test` (`go test -race -v ./...`) | exit 0, 0 `--- FAIL` lines |
| `make lint` (`go vet` + `golangci-lint run --config=.golangci.yml ./...`) | exit 0, `0 issues.` |
| `git status` (ai package scope) | clean working tree after both commits |
| `git diff main --stat` (both AI-17 files, full feature) | 1847 insertions across Phases 0–4 (commits `91168d0`, `8337573`, `4ceb77c`) |

## Relevant Files

- `backend/agent/src/ai/reasoning_event.go` — production code, all 3 kinds + constructors + validation.
- `backend/agent/src/ai/reasoning_event_test.go` — S-ARE-001..043 tests, test-local `reconstructFragments`/`reconstructReasoningState` helpers.
- `backend/agent/src/ai/event.go` — AI-14's registry (3 rows appended, landed via AI-16's commit `4f63977`, preserved verbatim).
- `backend/agent/src/ai/reasoning_content.go`, `reasoning_content_test.go` — read-only precedent, untouched (NFR-ARE-A).
- `openspec/changes/cachicamas-ai-reasoning-events/tasks.md` — all 14 tasks `[x]`, evidence log appended.

## Status

14/14 tasks complete. `make test` and `make lint` both green. Ready for `sdd-verify`.

---

> **Archive-time note, 2026-08-01.** The lint fix recorded above (`4ceb77c`) also closed the one finding AI-16's apply run had left open and correctly predicted a sibling would resolve — `WAVE-2-VERIFY.md` catalogues that chain as **D10 → D12**. `make lint` is `0 issues` at wave head. This milestone's one open suggestion at wave close is **S2**: AI-17 is the only block family whose tests never drive `CheckStream` (they call only `CheckEmit`), so reasoning block ordering works by construction rather than by runtime evidence. Not a defect — the checker is demonstrably payload-independent and the descriptors are pinned — but one `CheckStream` table over a reasoning start/delta/end sequence would close it. Owned by Wave 3.
