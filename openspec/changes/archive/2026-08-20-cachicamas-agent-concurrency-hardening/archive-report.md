# Archive Report — AG-21 `cachicamas-agent-concurrency-hardening`

> **Status**: ARCHIVED · **Verdict**: `sdd-verify` round 2 **PASS WITH WARNINGS** (0 CRITICAL, 0 MAJOR, 3 MINOR, 2 SUGGESTION), two of the three MINORs closed after that report in `87f92fbe`.
> **Milestone**: AG-21 (Layer 2, Wave 6, milestone 22 of 24) · charter `0003:1963-2043`
> **Branch**: `feat/agent-layer2-wave6-ag21` · base `main@54476ded` (PR #182, AG-20)
> **Worktree**: `cachicamas-worktrees/ag21-concurrency-hardening`

## What shipped

AG-21 proves the assembled harness clean under adversarial schedules. It is a **pure test-hardening milestone**: **zero production Go files changed**. The D1 contingency (fix in place, delta-backed, if a matrix cell surfaced a real defect) was pre-authorized and **did not fire**.

- **AG-21.1** — the 12-cell combined-state × signal matrix (4 states × 3 signals), composed from existing `agenttest` primitives. Every "pending" point is proven by a happens-before edge — a `Gate.Reached()` close, a synchronous nil `Steer` return, or an event read off the sink — and **never by a clock**.
- **AG-21.2** — two standalone pressure scenarios plus, added during the correction round, the **combined** stalled-consumer-with-pending-state driver that genuinely discharges `S-AGE-031`.
- **AG-21.3** — one non-parallel leak sweep wrapping the drivers this change adds, honouring `NFR-HKS-005` (which AG-20's spec addresses to AG-21 by name).
- **Cross-run state** — `agent-run-driver:87`'s long-standing *"Cross-run transcript state remains AG-21's"* is discharged by an enumerated inventory with a genuine **absence** assertion: a per-execution minted nonce, an anti-ghost floor proving the needle is findable in run 1, and a defeat test that must go RED against a deliberately shared transcript.

## Counted diff

**1,903 lines** (1,868 insertions / 35 deletions) excluding `openspec/**`, across 9 files — 5 new `_test.go`, 3 modified `_test.go`, and the milestone doc. Shipped as a **single PR** under the `size:exception` the user pre-authorized ("1000 review lines and extend if needed").

## Evidence

Full uncached suite, `cd backend/agent && go test -race -count=1 ./...` — `-count=1` mandatory, because the real uncached suite is ~175s and a sub-second pass is a cache artifact:

| Run | Result | Wall clock |
|---|---|---|
| Pre-change baseline | **FAIL** (see "Inherited defects" below) | 2:54.94 |
| Post-apply (orchestrator-run) | PASS | 2:56.14 |
| Post-correction (orchestrator-run) | PASS | 2:51.29 |
| Pre-archive (orchestrator-run) | PASS | ~2:54 |

Also clean: `make lint` (golangci-lint 0 issues), `make build`, `make vuln-check` (exit 0). `make all` was never run — its `fmt` step rewrites committed files and fails the substrate guards.

## Three inherited defects, none caused by AG-21

AG-21 could not have shipped without absorbing these. All three predate it.

1. **`openspec/specs/agent-hook-taxonomy/spec.md` was a truncated placeholder on `main`.** 21 lines: a correct header, the literal string `[Content continues: the full 354-line spec follows exactly as read above...]`, then an `sdd-archive` `## Key Learnings` return envelope written into the spec in place of the spec. `R-HKS-001…012`, every `S-HKS-*` and `NFR-HKS-001…006` were absent from the promoted tree while every citation into them still resolved to a plausible identifier. Repaired in **`c46b696b`** from the change's own archived delta, body verified byte-identical by diff. This mattered to AG-21 directly: the restored `NFR-HKS-005` ends *"AG-21 inherits this rule by name."*
2. **`TestHooks_SubstrateFilters_NoReleaseExactWideningInBothFilters` failed on every branch cut from `main` after AG-20.** Its `wantPreExisting` loop asserts `loop.go`/`harness.go`/`compaction.go` appear in the diff against the merge base — true only on AG-20's own authoring branch. Proven decisively: a detached worktree at `54476ded` passes (`ok 1.595s`); the same worktree plus a single `git commit --allow-empty` fails with three errors at `hooks_test.go:1990`. Fixed by AD-9's **two-layer** guard (empty-diff fast path plus AG-20's authorship-signature file), delta-backed as a narrow recorded release under `R-LSK-008`.
3. **`S-LSK-032` carried a false clause.** It claimed neither substrate filter names `failure.go` or `cost_events.go`; both name both (`loop_test.go:877,978`; `loop_hook_test.go:944,1032`), and the nearest guard checks for **duplicates**, not absence, so nothing ever defended it. AG-21's freshly-authored `S-LSK-034` had copied the same wording. Both corrected, with a `(Previously: …)` note recording that the claim was false when written rather than propagating it.

## Verify history

- **Round 1 — FAIL**: 0 CRITICAL, 3 MAJOR, 4 MINOR, 2 SUGGESTION. All nine closed by one scoped correction round (`17d7edf4`..`c68e307c`). **MAJOR-1 was closed by BUILDING the missing `S-AGE-031` conjunction, not by narrowing the requirement text** — the delta diff is one line and every GIVEN/THEN is byte-unchanged, so the requirement's strength is unchanged.
- **Round 2 — PASS WITH WARNINGS**: 0 CRITICAL, 0 MAJOR, 3 MINOR, 2 SUGGESTION; ready for archive.
- **After round 2**, two of the three MINORs were closed in `87f92fbe`: a self-contradicting doc comment left over from the MAJOR-2 correction, and a count-drift clause in `R-CNH-005` that said "the two pressure scenarios" after a third existed — closed on both halves, by enrolling the third driver in the sweep as well as fixing the text.

## Promotion

Seven deltas promoted — one NEW capability plus six MODIFIED — in `48c68e0a`. **Promotion was split across three writers on disjoint capabilities** rather than run as a single pass, precisely because AG-20's single-pass promotion exhausted mid-run and produced defect 1 above.

Audit, run by the orchestrator rather than self-reported:

| Capability | Op | Lines |
|---|---|---|
| `agent-concurrency-hardening` | NEW | 182 |
| `agent-event-delivery` | MODIFIED | 353 → 379 (+27/−1) |
| `agent-run-driver` | MODIFIED | 412 → 434 (+24/−2) |
| `agent-cancellation-tree` | MODIFIED | 207 → 235 (+33/−5) |
| `agent-turn-termination` | MODIFIED | 214 → 218 (+5/−1) |
| `agent-loop-skeleton` | MODIFIED | 303 → 314 (+37/−8) |
| `agent-v1-scope` | MODIFIED | 328 → 351 (+23/−1) |

Every target **grew**; none shrank. The contamination sweep — `grep -rln "Content continues|^## Key Learnings|skill_resolution|next_recommended|executive_summary" openspec/specs/` — returns **empty** across the whole promoted tree. Before AG-21 it returned `agent-hook-taxonomy/spec.md`.

## Charter deviation, recorded rather than discovered

Two of the twelve matrix cells cannot assert the charter's uniform Then, and this is a decision with evidence, not a defect:

- **compaction × provider-failure** — `R-CMP-010`'s own heading reads *"Atomic-or-absent by ordering, and a failed compaction never winds the run down."*
- **child-harness-active × provider-failure** — `Schedule` (`loop.go:542`) returns `[]Result` and no error, so a contained tool-execution failure never surfaces as a Turn-level error; the cell reaches `RunOutcomeCompleted`.

Round 1 of the spec named the wrong second member (suspension rather than child). Corrected across all six sites, with a zero-survivors sweep. **Suspension × failure is uniform in outcome**; what is bounded there is the firing *point*, not the Then — no main-provider stream is live while a call is parked.

## Open follow-ups (non-blocking, not AG-21's)

1. `R-LSK-008`'s requirement body (`agent-loop-skeleton/spec.md:240`) says `failure.go`/`cost_events.go` "are each absent from AG-20's additions … a filter entry would remove the guard that says so". That is AG-20's own text about AG-20's additions and is weaker than the scenario-level falsity AG-21 corrected, but it reads misleadingly now that the scenario states those entries exist. Worth a look when a milestone next touches this capability.
2. `TestAI33_1_RaceCancelMidDo` (`src/ai/openaicompat`) is a **pre-existing flake**. `sdd-verify` settled blame by reachability in one command rather than by re-running: `go list -deps -test ./src/ai/openaicompat` excludes `src/agent`, and AG-21's `src/ai/` diff is empty. A future run may see it.

## Doc 0003

Checklist row `0003:2179` flipped to `[x]`. Status counter updated to *"Wave 5 complete (AG-19…AG-20), Wave 6 opened (AG-21) — **22 of 24** milestones shipped"*, derived from the document's own Wave 0–6 table of contents rather than by incrementing the printed figure.

**Wave 6 remains open**: AG-22 (observability boundary) and AG-23 (Layer 3 readiness contract) are the last two milestones of Layer 2.
