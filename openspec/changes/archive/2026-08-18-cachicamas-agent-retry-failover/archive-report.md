# Archive Report: AG-15 — Implement retry policy and the failover seam

> **Change**: `cachicamas-agent-retry-failover` · **Milestone**: AG-15 (Layer 2 Wave 3, **15 of 24**)
> **Branch**: `feat/agent-layer2-wave3-ag15` · **Base**: `main@bf482b0a`
> **Closes**: **G8**'s Layer 2 half (R-15); seam 7 consumed, seam 8 reserved
> **Verify verdict**: PASS WITH WARNINGS — 0 CRITICAL, 8 WARNING, 4 SUGGESTION, **0 blockers**
> **Delivery**: single PR, `size:exception` pre-authorized by the user
> **Receipt-driven development**: off (clone-local) — delivery under ordinary repository policy, reported `disabled/unmanaged`

## What shipped

The harness now decides what the loop refused to: whether to retry a failed turn.

- **The retry predicate** — ordered gates G0–G5, first match wins. G0 cancellation wind-down stays ahead of everything; G2 reads `Retryable()`; **G3 reads `PartialOutput()` and fires only after G2 already said retryable**. That ordering is the whole point: the naive "retry if retryable" predicate is exactly what the G8 sentence forbids, and inverting the two gates would still pass a careless test. `Delivery()` is deliberately not a gate.
- **The inner attempt loop** in `Harness.Run`, reusing the transcript slice built before the loop — which converts "each attempt is a fresh provider call over an identical transcript" from an assumption into a structural guarantee.
- **Bounded backoff** via a fresh `RetryTiming` seam mirroring `retry.Config`'s shape without importing it (Go's internal-visibility rule forbids the import from the sibling `agent` package). Retry-after overrides computed backoff, clamped to `[0, MaxDelay]`. The wait selects on the run context, so an interrupt during backoff aborts immediately rather than hanging.
- **The failover seam** — `FailoverPolicy` interface with a typed `FailoverVerdict`, nil-default inert, plus a shipped `NoOpFailoverPolicy` decliner whose empty verdict makes acceptance unconstructible in v1. Carried on `Harness`, not `TurnOptions`: exhaustion is a run-driver concept, and `Turn` knows nothing of attempts.
- **True-category preservation** on the exhausted-retry `run_end`, via a sibling beside `wrapHarnessFailure` that wraps the identical `*ai.Failure` pointer rather than reconstructing a report.

### The blocker found before any code was written

Exploration flagged, and the orchestrator confirmed by reading the file the explore pass had not opened, that **AG-15.1's first charter scenario was not satisfiable against the substrate as it stood**:

- `loop.go:304-338` — all three of `Turn`'s pre-stream failure paths closed the sink and returned without ever emitting `turn_end`, leaving the turn bracket open.
- `stream_check.go:141-143` — on `BracketRoleOpensTurn`, `if turnOpen { return violation(ai.ErrMisplaced, ...) }`.

A harness retry re-invoking `Turn` after a pre-output failure therefore emitted a second `turn_start` while a turn was still open, and `CheckStream` rejected the stream. This forced a mandatory companion delta on `agent-loop-skeleton`. Because `R-RUN-003` pins `stream_check.go` byte-unchanged, the fix landed on the emitter, never the validator — and bite `S-RTY-011` now reverts the emission and proves `CheckStream` rejects, turning the defect into a permanent regression guard.

## The composed ceiling

**H = 3 total harness attempts × 4 wire requests = 12.**

Layer 2 counts **total attempts**; Layer 1 counts **retries after the first** (`DefaultMaxAttempts = 3` meaning 4 requests). The conventions differ deliberately, and `R-RTY-005` makes *stating the convention adjacent to every number* itself falsifiable — so the first rate-limit storm is not the first time anyone computes the product. `R-AIS-044` already carried the Layer 1 half from AI-35; AG-15 owed only Layer 2's, and `S-RTY-009` enforces the cross-layer wording by test rather than by comment.

## Commits (10)

| SHA | Subject |
|---|---|
| `288aa154` | `feat(agent): emit turn_end on Turn's pre-stream failure paths` |
| `cc1cab86` | `feat(agent): retry a pre-output failure across an inner attempt loop` |
| `0baec9b1` | `docs(openspec): AG-15 SDD planning artifacts and batch-1 progress` |
| `38e06ef2` | `feat(agent): inject retry timing, honor retry-after, state the composed ceiling` |
| `790f63e8` | `feat(agent): consult the failover seam once at retry exhaustion` |
| `473f605c` | `fix(agent): preserve the true failure category on an exhausted-retry run_end` |
| `2f2cae67` | `docs(0003): tick AG-15, bump milestone counter to 15/24` |
| `a6a3f515` | `style(agent): separate retry_policy.go/failover_policy.go headers from the package doc` |
| `48ccc50b` | `docs(openspec): AG-15 batch-2 apply-progress (final, merged with batch 1)` |
| `f3915eb1` | `fix(agent): inject retry timing so the visible-attempts test stops reaching the wall clock` |

## Post-verify remediation (`f3915eb1`) — final state

Two verify findings were fixed before archive rather than carried forward:

1. **A test was reaching the production wall clock.** `TestHarness_RetryVisibleAttempts` omitted `RetryTiming`, falling through to the production `DefaultRetrySleep` and genuinely sleeping `BaseDelay` between attempts — 0.43s against 0.00s for every sibling. This violated `S-RTY-002`'s own Given, `R-RTY-006` and `NFR-RTY-002`. The drift mechanism is worth recording: the test was authored in Phase 1 *before* the timing seam existed, and began sleeping silently when Phase 2 landed a 100 ms default. A `grep "time.Sleep("` returned clean and structurally could not see it, because the clock is reached transitively through production `time.NewTimer`. Now 0.00s.

2. **A bite scenario described a mechanism that never ran.** `S-RTY-012` claimed it perturbs the cited wording in Layer 1's package documentation; what actually executed perturbs the test's own expectation slice. Apply's choice was correct — `R-RUN-012` forbids editing `backend/agent/src/ai/**` even transiently — but the scenario text and `ai-stream-lifecycle`'s `R-AIS-044` back-annotation both asserted the unexecuted mechanism, and **both promote verbatim at archive**. Both were corrected to describe what runs, with the reasoning recorded: the check is `strings.Contains`, symmetric in its operands, so proving power is identical, and the resulting guarantee is stronger than a revert because the Layer 1 file is never written at all.

## Capabilities promoted

| Capability | Kind | What promoted |
|---|---|---|
| `agent-retry-failover` | **NEW** | `R-RTY-001`…`012`, `S-RTY-001`…`015` (incl. 3 bites). 262 lines. |
| `agent-loop-skeleton` | MODIFIED | `R-LSK-001` pre-stream `turn_end` obligation; `R-LSK-004` substrate rules; new `S-LSK-021/022/023`; **header allocated range extended to `S-LSK-023`** |
| `agent-run-driver` | MODIFIED | `R-RUN-003` restated across N attempts; `R-RUN-011` retry carve-out (cancellations and post-output failures kept verbatim); `S-RUN-103`; deferred row closed |
| `agent-turn-termination` | MODIFIED | `R-ATT-005` cross-reference to the pre-stream rule; `S-ATT-014`; deferred row closed |
| `ai-stream-lifecycle` | back-annotation | `R-AIS-044 / S-2` Layer 2 consumer named. No requirement obligation changed; no Layer 1 file touched. |
| `agent-v1-scope` | back-annotation | `AGS-D` records AG-15.3 shipped the seam, implementation still deferred |
| `agent-cancellation-tree` | back-annotation | "retry/failover on a cancelled turn" confirmed closed **by gate G0's ordering**, a mechanism a later reader can check, not a reassurance |

## Verification at close

Independently re-run by the orchestrator, not inherited from a phase report:

- `go test -race -count=1 ./...` from `backend/agent/` — **12/12 packages `ok`**, zero FAIL
- `make lint` after `golangci-lint cache clean` — **`0 issues`**; `go vet` clean; `make build` clean
- `loop.go` statement coverage 89.1% (277/311), satisfying `NFR-RTY-005`
- `tasks.md`: 35 `[x]`, 0 `[ ]`
- Pinned substrate byte-unchanged vs `main`: `stream_check.go`, `stream_check_test.go`, `turn_events.go`, `failure.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `history.go`, `harness_test.go`, `go.mod`, `go.sum`, and **all of `backend/agent/src/ai/`**
- `ai/internal/retry` absent from `./src/agent`'s dependency closure (`go list -deps`) — the shape was mirrored, never imported
- 25 `EventKind`s unchanged
- All three bites reverted; no mutation survives in any commit
- Base checkout clean on `main` at `bf482b0a` throughout

## Carried forward as follow-ups (non-blocking)

1. **`tasks.md`'s coverage table mapped only `S-RTY-*`** and omitted every delta-spec scenario. Six clauses are true in fact but not test-enforced: `src/ai/` byte-unchanged, `S-RTY-006`'s source scan, `S-RUN-103`'s transcript/`CloseTurn`, `S-LSK-021`'s nil-path `CheckStream`, `S-LSK-023`'s filter-set equality, `S-ATT-014`'s discriminator. Root cause is structural — nothing in the apply loop was ever pointed at delta-spec scenarios. Worth fixing in the tasks template, not here.
2. **Pre-existing flake, NOT caused by AG-15**: `TestAI33_1_RaceCancelMidDo` in `src/ai/openaicompat` failed once under the full parallel `-race` suite, then passed 5/5 in isolation. Proven unrelated by dependency closure — `openaicompat` has zero `src/agent` entries, so Layer 1 cannot see Layer 2, and AG-15 changed nothing under `src/ai/`. Deserves its own issue.
3. **AG-16 runs parallel and also edits `harness.go`.** The inner attempt loop restructures `Run`'s turn-invocation block, so the merge conflict will be structural rather than line-level.
4. Recorded implementation deviations, all within design latitude: `retry_decision_internal_test.go` as an internal-package test (justified by `NFR-RTY-001`'s carve-out for the predicate's own table test); `DefaultRetrySleep` exported (forced by the external-test rule, whose carve-out covers `S-RTY-001` but not `S-RTY-008`); `S-RTY-007` asserting a mathematically-guaranteed range rather than RNG-replicated exact values.

## State at close

Layer 2 stands at **15 of 24**. AG-16 (cost and usage events) is next and depends on AG-15; AG-17 can start as soon as its other edges close.
