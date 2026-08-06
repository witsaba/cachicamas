# Tasks: the reasoning stream — a recorded capability absence

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | ~150–250 (1 test file ~120 LOC + decision.md ~200 lines + doc 0002 amendments ~40 lines) |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | single PR |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: feature-branch-chain
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | decision.md + doc 0002 amendments + reasoning_absence_test.go (2 functions) | PR 1 (single) | `go test -race -count=1 -run 'ReasoningExtensionField\|TestConformanceFactory_DeclaresReasoningExplicitlyFalse' ./...` from `backend/agent/` | `go test -race -count=1 ./...` full suite from `backend/agent/` | `git revert` single commit — nothing generates from or imports these files |

> **Branch point**: apply branches from the feature/tracker chain's current tip at apply time — an AI-28 corrective (`feat/ai-28-8-d8-close-discipline`) lands in parallel and is expected ahead of `feat/ai-28-7-pre-decode-checks`. Verify the D8 corrective is merged into the chain before branching; do not branch from `ai-28-7` directly.
> **First-commit scope**: banks only THIS change's planning artifacts (`proposal.md`, `specs/ai-provider-reasoning-stream/spec.md`, `design.md`, `tasks.md`). Never `git add -A`; never stage another change's uncommitted `openspec/` folder.

## Phase 1: `decision.md` (R-ARS-001…014)

- [x] 1.1 Write §1–§4 (how-to-use, verdict = absence, evidence-rule labels, C7/C8 grounds) per design §2 — R-ARS-001, 002, 008.
- [x] 1.2 Write §5's five-row mechanism table verbatim per design §2 (mechanism, location, named proving test function each); row 2 (`bridge_test.go:46-73`, factory declares `Reasoning:false`) notes "no landed test — proven by this change's `TestConformanceFactory_DeclaresReasoningExplicitlyFalse`" — R-ARS-003, 014.
- [x] 1.3 Write §6–§8 (deadlock/pinned-dialect basis routed to AI-38.2; price restated in absence terms, AI-07/AI-17 frozen not deprecated; `CAP-O-01` confirmed not re-derived) — R-ARS-004, 006, 007.
- [x] 1.4 Write §9–§11 (two reopen triggers with named owners; AI-38/AI-40 inheritance incl. item-6 restate-and-publish via AI-40.2; closing-checklist verification table for AI-29.0 items 1/2) — R-ARS-005, 012, 013.

## Phase 2: doc 0002 amendments B1–B8 (R-ARS-009…013)

- [x] 2.1 B1 — `### AI-29` charter (~1743): strike acceptance's second half; dated blockquote restates absence-terms acceptance, enumerates B7's relabel.
- [x] 2.2 B2 — `#### AI-29.0` (~1764): blockquote — checklist closed with absence, `strongly indicates` upgraded, deadlock/routing/triggers named.
- [x] 2.3 B3 — AI-29.1/29.2/29.3 (~1773–1791): strike test-list bodies (`~~…~~`, legible), blockquote — no subject for this adapter, un-strike condition.
- [x] 2.4 B4 — lines 554, 1247, 1429: inline-strike the dangling pointer only; blockquote re-points each at `decision.md`, naming what to consult.
- [x] 2.5 B5 — line ~2347: strike+restate the superseded Wave-2-close sentence (`~~Item 6 also stays open… AI-29.2~~`); blockquote — not-exercisable-in-v1, published via **AI-40.2 — Capability matrix and examples**, no node appended.
- [x] 2.6 B6 — `G12(b)` row (2402) + checklist→nodes mapping (2415): strike `AI-29.2` references; blockquote restates not-exercisable-in-v1 with AI-40.2 as publishing owner.
- [x] 2.7 B7 — mermaid graph (1755–1762): relabel `X1`/`X2`/`X3` node text with `(struck)`, e.g. `AI-29.1 (struck)<br/>reasoning is never text`; no text deleted.
- [x] 2.8 B8 — `#### AI-40.2 — Capability matrix and examples` (2298): additive dated blockquote recording the inherited item-6 publication duty.
- [x] 2.9 Verify — run the S-ARS-025 grep gate against the actual diff; confirm the six-hit disposition matches design §3's table exactly:

| Line | Disposition |
|---|---|
| 554 | Re-pointed — B4 |
| 1247 | Re-pointed — B4 |
| 1429 | Re-pointed — B4 |
| 2347 | Struck and restated — B5 |
| 2402 | Struck and restated — B6 |
| 2415 | Struck, mapping restated — B6 |

None dangles.

## Phase 3: `backend/agent/src/ai/openaicompat/reasoning_absence_test.go`

- [x] 3.1 RED — write the pin test function (S-ARS-036…039): fixture A (delta 1 content + `reasoning_content` sentinel `RSN-EXT-SENTINEL-7f3a` once, delta 2 content after it asserted intact/in-order, comparative twin per S-ATS-071); fixture B (delta carrying only the extension field); leak-assertion helper (no text event carries a sentinel byte, no reasoning-typed event); drop/not-leak/not-fail on both fixtures. Capture compile/first-run RED.
- [x] 3.2 RED — write `TestConformanceFactory_DeclaresReasoningExplicitlyFalse` (S-ARS-033): assert the bridge factory's three optional-capability declarations, reasoning `false` explicit non-nil.
- [x] 3.3 GREEN — run both; per design §1 both are expected GREEN against landed code (`decodeChunk` has no `DisallowUnknownFields`, ignores undeclared delta members on every path). If genuinely RED: stop, add zero production Go, log the finding, re-scope the fix to a corrective AI-28 change — do not fix chunk decode here.
- [x] 3.4 Non-vacuity — add the permanently-executable self-inversion sub-test (S-ARS-042): feed the leak-assertion helper a synthetic event list with the sentinel routed into a text delta; assert the helper reports failure. Separately perform one staged-and-reverted production mutation at apply time; record both in the evidence log.

## Phase 4: Gates

- [x] 4.1 Zero-production-Go check (R-ARS-017, S-ARS-043/044): diff restricted to `backend/` shows exactly one added `_test.go`; `go.mod`/`go.work` byte-identical.
- [x] 4.2 `go test -race -count=1 ./...` from `backend/agent/`, run twice for flake detection.
- [x] 4.3 `go vet ./...` and project lint pass clean.
- [x] 4.4 `gofmt -l` reports zero files.
- [x] 4.5 Inspection verification pass — walk all 40 `[inspection]` scenarios against the merged `decision.md` + doc 0002 diff, grouped by requirement:

| Requirements | Scenarios |
|---|---|
| R-ARS-001…013 | S-ARS-001…032 |
| R-ARS-014 | S-ARS-033…035 |
| R-ARS-017, 018 | S-ARS-043…047 |

`sdd-apply` performs this pass first; `sdd-verify` re-performs it independently.
