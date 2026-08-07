# Archive Report — `cachicamas-ai-retry-policy` (AI-35)

> **Change**: `cachicamas-ai-retry-policy` · **Milestone**: AI-35 (doc 0002 § 2077–2144) — Define retry and idempotency policy · **Wave**: 5 — Harden  
> **Phase**: archive (final) · **Status**: CLOSED  
> **Date**: 2026-08-07 · **Branch**: `feat/ai-35-retry-policy` @ `6bfc266`  
> **Project**: cachicamas (witsaba)

---

## Executive Summary

AI-35 (Layer 1 retry and idempotency policy) is archived. All 21/21 scenarios PASS (R-AIS-041..044 at 14 scenarios + R-CNF-019 at 7 scenarios), 0 CRITICAL, 8 implementation commits landed on `feat/ai-35-retry-policy` @ `6bfc266`, aggregate diff 1264 insertions / 61 deletions across 15 files. Spec deltas for `ai-stream-lifecycle` and `ai-provider-conformance-suite` merged into canonical specs with amendment blockquotes dated 2026-08-07. Change folder moved to `openspec/changes/archive/2026-08-07-cachicamas-ai-retry-policy/` at archive time. Three acknowledged deviations (D-A1 conformance case body placement, D-A2 size:exception 1096→1264 lines, D-A3 lint fix rename) documented below.

---

## Phase Artifact Inventory

| Artifact | Status | Observation ID | Location |
| --- | --- | --- | --- |
| **Proposal** | PASS | #2643 | Engram `sdd/cachicamas-ai-retry-policy/proposal` |
| **Spec (delta)** | PASS | #2644 | Engram `sdd/cachicamas-ai-retry-policy/spec` |
| **Design** | PASS | #2645 | Engram `sdd/cachicamas-ai-retry-policy/design` |
| **Tasks** | PASS | #2646 | Engram `sdd/cachicamas-ai-retry-policy/tasks` |
| **Apply-progress** | PASS | (inline in worktree) | `openspec/changes/cachicamas-ai-retry-policy/apply-progress.md` |
| **Verify-report** | PASS | #2650 | Engram `sdd/cachicamas-ai-retry-policy/verify-report` |
| **Archive-report** | ACTIVE | (this document) | `openspec/changes/archive/2026-08-07-cachicamas-ai-retry-policy/archive-report.md` |

---

## Verification Verdict

**PASS — 21/21 scenarios, 0 CRITICAL, 0 WARNING, 2 SUGGESTION**

Per `verify-report` (Engram #2650):
- **Requirements verified**: R-AIS-041, R-AIS-042, R-AIS-043, R-AIS-044 (Layer 1 stream lifecycle spec, 14 scenarios)
- **Conformance verified**: R-CNF-019 (provider conformance suite, 7 scenarios); R-CNF-017 (modified for totality over 9 capabilities, structural)
- **Test suite**: `make test` PASSES 899 PASS / 0 FAIL / 4 pre-existing SKIP across 7 packages under `-race`
- **Lint**: `make lint` reports 0 issues
- **Build**: `make build` exits 0
- **Forward guard**: `TestLayer1_ModuleHasNoDependencies_ZeroRequires` PASS; helper is stdlib-only + own-module imports
- **Validator**: `gentle-ai sdd-verify-validate` admits the report with `verdict: pass`

---

## Implementation Commits

Eight commits on `feat/ai-35-retry-policy` branch:

| # | SHA | Subject | WU | Lines | Purpose |
| --- | --- | --- | --- | --- | --- |
| 1 | `5b51a20` | `feat(ai): add internal/retry helper package` | WU-1 | +195 | Helper package foundation (Loop, Config, AttemptReport, DefaultMaxAttempts) |
| 2 | `94c8fed` | `feat(ai): wire retry.Loop into openaicompat Stream` | WU-2 | +214 | Stream integration + AI-35.1 predicate tests |
| 3 | `839e83b` | `fix(ai): preserve typed cancellation at retry seam` | WU-2 fixup | +8 | Context cancellation propagates as typed failure, not bare ctx.Err() |
| 4 | `eed2071` | `feat(ai): prove bounded context-aware retry backoff` | WU-3 | +220 | AI-35.2 backoff mechanics (Retry-After, bounded, context-aware) |
| 5 | `cecab38` | `feat(ai): prove replayability and partial-output boundary` | WU-4 | +240 | AI-35.3 replayability + boundary marker tests |
| 6 | `daddba1` | `feat(agent): add CapRetry conformance capability and case body` | WU-5 | +237 | CapRetry enum + R-CNF-019 conformance case (7 sub-tests) |
| 7 | `ceb85c8` | `fix(agent): keep retry conformance case body in the openaicompat test package` | WU-5 fixup | +10 | Deviation D-A1: move conformance case body from agenttest to openaicompat |
| 8 | `6bfc266` | `fix(ai): rename retry.AfterReader to avoid package-level stutter (lint)` | Lint fix | +8 | Deviation D-A3: lint fix rename (RetryAfterReader → AfterReader) |

**Aggregate**: 1264 insertions / 61 deletions across 15 files (15 files changed per `git diff --shortstat 238b9fa..HEAD`).

---

## Specification Amendment Summary

Two canonical specs amended with dated blockquotes at archive time:

### `openspec/specs/ai-stream-lifecycle/spec.md`

**Amendment location**: § 7 "Decision 5 — failure delivery", after "Who inherits it" section.

**Blockquote added**: "Amended 2026-08-07 (AI-35)" documenting:
- Four requirements added: R-AIS-041 (predicate), R-AIS-042 (backoff), R-AIS-043 (replayability + boundary), R-AIS-044 (composed-bound ceiling)
- The seam: in-adapter loop, factored into helper invoked by adapter's execute-once function
- No modifications to existing failure-delivery contract (handover boundary remains unpolluted by retry)
- Cross-reference section added naming each requirement's scenarios and conformance pins

**Acceptance criteria updated**: Items 9–11 added for AI-35, documenting the four requirements, 14 scenarios, verify-report verdict (PASS, 21/21 scenarios), and three acknowledged deviations.

### `openspec/specs/ai-provider-conformance-suite/spec.md`

**Amendment location**: After "What design resolved" section, at end of file.

**Blockquote added**: "Amended 2026-08-07 (AI-35)" documenting:
- One optional capability added: `CapRetry` (CAP-O-04)
- `Capability` enum grows from [8] to [9]
- One requirement added: R-CNF-019 (auto-retry retryable pre-stream failures up to documented bound)
- Seven scenarios (S-CNF-069..075) verify R-CNF-019
- R-CNF-017 (totality) modified to reflect 9 capability entries (mechanical one-line rebuild)
- New `Factory.Retry *bool` field for capability declaration

**Note**: The conformance suite spec has no separate "Requirements" cross-reference section like the stream-lifecycle spec. The amendment blockquote provides the summary; full details are archived at the delta spec.

---

## Acknowledged Deviations

Per `apply-progress.md` (lines 30–34) and verified against final state:

### D-A1 — Conformance case body placement

**Deviation**: `conformance_retry_test.go` placed in `backend/agent/src/ai/openaicompat/` instead of `backend/agent/src/agenttest/`.

**Reason**: Design D-R4 identified risk that openaicompat as dependency of agenttest could be problematic. Resolved by placing case body in the package it tests (openaicompat), consistent with the pattern where conformance cases live alongside their subjects.

**Acceptance**: No new imports in agenttest; openrouter wrapper inherits helper transparently via `*openaicompat.Client` embed; R-CNF-019 still satisfied. Both openaicompat and openrouter exercise identical behavior, so a future OpenRouter conformance roll-up (AI-38) can extend the same case body without rewriting assertions.

**Evidence**: Commit `ceb85c8`; `agenttest/doc.go` updated to note the package boundary (3 lines).

### D-A2 — Aggregate exceeds 1000-line budget (size:exception 1096→1264)

**Deviation**: Forecast at tasks phase was ~1096 lines; actual landed at 1264 lines (1000-line budget with 96-line overrun approved; actual is +168 from forecast, +264 total overrun).

**Why the drift**: 
- Conformance case body grew to 374 lines (vs ~150 forecast) due to detailed seven-scenario implementation (S-CNF-069..075) plus helper instrumentation for transport mocking
- `execute_once.go` extracted as separate method file (25 lines, vs bundled with stream.go in forecast)  
- Bridge test updates (10 lines)
- Lint-fix commit (8 lines)

**Reason for approval**: User approved `size:exception` at ~1096 line forecast (Engram #2647). Orchestrator's acquire set `--max-changed-lines 1200`. Actual 1264 is within the extended exception cap (1500). No further trimming required.

**Acceptance**: The overrun is concentrated in test files (conformance case + real-producer tests) whose density is mandated by strict-TDD discipline and AI-33/AI-34 precedent. No test documentation density was sacrificed; all scenarios are fully documented per spec.

**Evidence**: Commits 1–8 above; `apply-progress.md` § 1; user approval Engram #2647; orchestrator ledger extended by user.

### D-A3 — Lint-fix commit after apply phase closed

**Deviation**: Apply sub-agent was interrupted before settling the ledger; lint fix landed as commit `6bfc266` after interruption.

**Issue fixed**: Package-level name stutter in `retry.RetryAfterReader` (revive linter). Renamed to `retry.AfterReader`; field `cfg.RetryAfterReader` → `cfg.RetryAfter` for consistency.

**Reason for placement at archive**: The rename was mechanical and did not change any test behavior. Verify phase re-ran `make test` (PASS) and `make lint` (0 issues) on the final commit. No spec changes required; R-AIS-042 references `RetryAfter()` method which is behavior-level, not identifier-level per the no-Go-identifier contract.

**Acceptance**: Verify report confirms the lint fix does not change spec compliance; `make test` and `make lint` both pass with the final commit.

**Evidence**: Commit `6bfc266`; `verify-report` § "Deviations Confirmed" item D-A3.

---

## Spec Promotion Evidence

Both delta specs merged into canonical specs following the amendment-blockquote pattern established by AI-33 and AI-34.

### Lines changed in canonical specs

**`openspec/specs/ai-stream-lifecycle/spec.md`**:
- Lines 517–520: Amendment blockquote added after § 7 "Who inherits it"
- Lines 829–844: New requirement cross-reference section "R-AIS-041 through R-AIS-044"
- Lines 848–850: Acceptance criteria items 9–11 added (AI-35)
- Total net: +25 lines

**`openspec/specs/ai-provider-conformance-suite/spec.md`**:
- Lines 330–334: Amendment blockquote added after "What design resolved" section
- Total net: +4 lines

**Amendment blockquotes** both cite the archived delta specs at `openspec/changes/archive/2026-08-07-cachicamas-ai-retry-policy/specs/` for full requirement text and scenarios. Canonical specs pin the high-level change summary and cross-references only.

---

## Change Folder Archive

**Moved from**: `openspec/changes/cachicamas-ai-retry-policy/`  
**Moved to**: `openspec/changes/archive/2026-08-07-cachicamas-ai-retry-policy/`

Archived folder contains:
- `proposal.md` — initial problem statement, two open flags resolved
- `explore.md` — architectural exploration, helper location and seam analysis
- `specs/ai-stream-lifecycle/spec.md` — delta spec with 4 requirements, 14 scenarios
- `specs/ai-provider-conformance-suite/spec.md` — delta spec with 1 requirement, 7 scenarios, modified R-CNF-017
- `design.md` — concrete Go shape, 10 decisions, type signatures for apply phase
- `tasks.md` — 5 work units, strict TDD ordering, 1096-line forecast (later 1264 actual)
- `apply-progress.md` — apply phase record, 8 commits, 3 acknowledged deviations
- `verify-report.md` — verify phase record, 21/21 scenarios PASS, 0 CRITICAL
- `archive-report.md` — this document

---

## Recommended Next Step

No follow-up SDD required. AI-35 is complete and closed.

**Next milestone**: AI-36 (doc 0002 line 2142) — the next Wave 5 milestone after AI-35.

---

## Key Learnings

1. Conformance case placement in the package it tests (openaicompat) rather than the suite package (agenttest) preserves adapter-agnostic posture and avoids circular-dependency risk when multiple adapters exist.

2. Size:exception extension from 1096 to 1264 lines reflects seven-scenario conformance case detail that is load-bearing for future adapter conformance verification; test density is not a luxury that can be trimmed without weakening the spec's verifiability.

3. Partial-output boundary marker testing via `httpClient.Do` count == 1 after emitted event is more durable than asserting on `PartialOutput()` discriminator directly, because the count tests the construction (seam placement) rather than the classification (discriminator value).

4. Cross-layer composed-bound visibility requires explicit documentation in both Layer 1 helper's package doc comment and Layer 2's test — a single-source-of-truth antipattern that is necessary when the bound affects both layers' reviewability and must be found without cross-referencing between specs.

5. The retry discipline landing as a pre-stream seam before carrier handover keeps the failure-delivery axis orthogonal from the partial-output axis, preserving AI-19's ability to classify and AI-35's ability to retry without either reaching into the other's domain.
