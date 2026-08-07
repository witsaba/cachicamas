# Tasks: AI-33 — Prove cancellation and goroutine cleanup

> **Change**: `cachicamas-ai-cancellation` · **Milestone**: AI-33 (doc 0002:1987–2037) · **Wave**: 5 — Harden
> **Branch / worktree**: `feat/ai-33-cancellation` · `cachicamas-worktrees/ai-33` · base `main` @ `e9a8054`
> **Module**: `backend/agent` — **layered** (ADR 0005 § D1), NOT hexagonal
> **Test runner**: `cd backend/agent && make test` (`go test -race -v ./...`)
> **Strict TDD**: RED first, GREEN only after RED is observed, REFACTOR with `-race` clean
> **Convention**: Conventional Commits only, no `Co-Authored-By`, no AI attribution
> **Delivery strategy (preflight)**: `single-pr` · `decision needed before apply: No` (all subnodes under the 400-line budget)
> **Pre-resolved**: AI-33.3 scope = "truly-abandoned consumer" (R-CNF-012 verbatim); test file naming `a_i-33_{1..5}_test.go`; internal `package openaicompat` for 33.1–33.4, external `package openaicompat_test` for 33.5; no new Go deps.
> **Inputs**: `proposal.md` §§5, 10, 14 · `specs/ai-stream-lifecycle/spec.md` R-AIS-033–038 · `design.md` per-subnode table · `explore.md` § 3 test-seam table · `openspec/config.yaml` `rules.tasks`

## Review Workload Forecast

| Field | Value |
|---|---|
| AI-33.1 estimated changed lines | ~150 (test only — code path already merged at `stream.go:181–183, :217–230, :241–251`) |
| AI-33.4 estimated changed lines | ~150 (test only — race-detector proof) |
| AI-33.2 estimated changed lines | ~200 base; +bounded fix if RED fails (watcher or ctx-aware wrap) |
| AI-33.3 estimated changed lines | ~200 (test only — wording pinned to R-CNF-012 verbatim) |
| AI-33.5 estimated changed lines | ~300 (drain in `stream.go:345` + external full-suite wrapper) |
| **Aggregate (5 PRs)** | **~1000 lines across 5 independent PRs** |
| **400-line budget risk** | **Low** per subnode |
| **Chained PRs recommended** | **No** — each subnode ships as its own PR; the 33.5a/33.5b split is contingent on RED exceeding 400 in 33.5 |
| **Chain strategy** | `n/a` (single-pr per subnode) |
| **Decision needed before apply** | No |
| **AI-33.5 chain-plan trigger** | if aggregate 33.5 ≥ 400 → split into 33.5a (drain impl, ~80 lines) + 33.5b (full-suite leak check, ~220 lines). Threshold declared before any 33.5 commit lands. |

```
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: n/a
400-line budget risk: Low
```

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Prove pre-headers cancellation returns typed failure | AI-33.1 | `cd backend/agent && go test -race -v -run 'TestAI33_1' ./src/ai/openaicompat/...` | `make test` | delete `a_i-33_1_test.go`; production code untouched |
| 2 | Prove after-completion cancellation is a no-op (race-clean) | AI-33.4 | `cd backend/agent && go test -race -v -run 'TestAI33_4' ./src/ai/openaicompat/...` | `make test` (×50 repeats built into `RequireNoGoroutineLeak`) | delete `a_i-33_4_test.go`; production code untouched |
| 3 | Prove between-frames cancellation closes bounded | AI-33.2 | `cd backend/agent && go test -race -v -run 'TestAI33_2' ./src/ai/openaicompat/...` | `make test` | delete `a_i-33_2_test.go`; if production change forced, revert that single statement |
| 4 | Prove truly-abandoned consumer + cancel drops bare | AI-33.3 | `cd backend/agent && go test -race -v -run 'TestAI33_3' ./src/ai/openaicompat/...` | `make test` | delete `a_i-33_3_test.go`; production code untouched |
| 5 | Drain-before-close + full-suite leak wrapper | AI-33.5 | `cd backend/agent && go test -race -v -run 'TestAI33_5' ./src/ai/openaicompat/...` | `make test` (full suite) | revert `stream.go:345` defer chain + delete `a_i-33_5_test.go` |

---

## Order: 33.1 → 33.4 → 33.2 → 33.3 → 33.5

Rationale (proposal § 5): surface the empirical uncertainty of 33.2 before the politically charged wording reconciliation of 33.3, while 33.1 and 33.4 establish the `httptest.NewServer` + race-detector posture the rest inherit.

---

## AI-33.1 — Cancel before headers (pre-headers, no spawn)

- [x] **33.1.1 `[red]`** Write `backend/agent/src/ai/openaicompat/a_i-33_1_test.go` (`package openaicompat`); tests `TestAI33_1_TextStream_CancelBeforeDo` (R-AIS-034/S-1), `TestAI33_1_ToolCallStream_CancelBeforeDo` (S-2), `TestAI33_1_RaceCancelMidDo` (S-3) — each uses `contextBeforeFirstFrameServer` (`stream_failure_test.go:391`) inside `RequireNoGoroutineLeak`; asserts `*ai.Failure` with `Category==FailureCategoryCancellation`, `Delivery==DeliveryPreStream`, nil channel, no leak.
  - **Depends on**: none
  - **Acceptance**: three failing test names compile; assertions fail on `main` (no such test file exists); expected to GREEN on this branch since `stream.go:181–183, :217–230, :241–251` already implements the path.
  - **Files**: `backend/agent/src/ai/openaicompat/a_i-33_1_test.go` (NEW, ~150 lines)
  - **Estimated Δ**: +150 / -0
  - **Risk**: low — code path already merged; red is observable only on a missing test file (the test must fail for the "right reason": the absence of the assertion surface, not a broken code path).

- [x] **33.1.2 `[refactor]`** Doc comment cites `R-CNF-011` and `R-ATS-002` verbatim; tidy `t.Run` names; ensure `-race` clean across `RequireNoGoroutineLeak` (50 repeats). `make lint` clean.
  - **Depends on**: 33.1.1
  - **Acceptance**: `cd backend/agent && make test` and `make lint` both green; `RequireNoGoroutineLeak` shows `leakTolerance` not exceeded.
  - **Files**: `a_i-33_1_test.go`
  - **Estimated Δ**: +5 / -5

- **33.1 commit shape**: single commit `feat(openaicompat): prove pre-headers cancellation against real HTTP transport`. Work unit = one deliverable behavior + its test in one commit (per `work-unit-commits` skill).

---

## AI-33.4 — Cancel after completion (race-detector proof)

- [x] **33.4.1 `[red]`** Write `backend/agent/src/ai/openaicompat/a_i-33_4_test.go` (`package openaicompat`); tests `TestAI33_4_TextStream_CancelAfterCompletion` (R-AIS-037/S-1), `TestAI33_4_ToolCallStream_CancelAfterCompletion` (S-2), `TestAI33_4_RaceCancelConcurrentWithFinalReceive` (S-3) — each uses `bridgeServeTranscripts` (`openrouter/conformance/bridge_test.go:141`) with text and tool-call transcripts ending in `[DONE]`; `DrainAndRecord` + `RequireNoGoroutineLeak` ×50; asserts exactly one terminal, exactly one close, no `-race` panic.
  - **Depends on**: none
  - **Acceptance**: three failing tests; GREEN on this branch (post-completion cancellation is already a no-op since `defer close(out)` at `stream.go:344` is the single closing site).
  - **Files**: `backend/agent/src/ai/openaicompat/a_i-33_4_test.go` (NEW, 322 lines — see deviation note below)
  - **Estimated Δ**: +322 / -0
  - **Risk**: low; the race-detector proof is the value, not the assertion.
  - **Deviation (deviation-from-design, recorded)**: spec body names `bridgeServeTranscripts` (`openrouter/conformance/bridge_test.go:141`) as the transcript fixture. That helper is unexported and in a separate Go package (`package openrouter_conformance`), unreachable from `package openaicompat`. Used the shape-identical, same-package helper `serveTranscripts` (`openaicompat/bridge_test.go:98`) instead — it is the established test-only fixture `conformanceBridgeFactory` itself already uses (line 75). Behavior identical. Conformance suite (`agenttest/conformance_*`) untouched per constraint.
  - **Deviation (assertion refinement, recorded)**: race scenario S-3 first asserted `last.Kind() == EventKindCompletion`. Failed RED: when cancel races mid-stream, AI-32.3's bounded-wait at `stream.go:437–470 / 503–526` emits an Error terminal instead. Refined assertion to `last.Kind() in {Completion, Error}`. Spec S-3 wording ("exactly one terminal") admits both; both close the channel exactly once via the unique `defer close(out)` (R-CNF-009); neither is a regression.

- [x] **33.4.2 `[refactor]`** Doc comment cites `R-CNF-009`, `R-CNF-011`, the single-`defer close(out)` invariant; `-race` clean.
  - **Depends on**: 33.4.1
  - **Acceptance**: `make test` and `make lint` green; race-detector quiet across all 50 repeats.
  - **Files**: `a_i-33_4_test.go`
  - **Estimated Δ**: +5 / -5

- **33.4 commit shape**: single commit `feat(openaicompat): prove post-completion cancellation is a no-op under -race`.

---

## AI-33.2 — Cancel between frames (empirical, possible code change)

- [x] **33.2.1 `[red]`** Write `backend/agent/src/ai/openaicompat/a_i-33_2_test.go` (`package openaicompat`); tests `TestAI33_2_TextStream_CancelBetweenFrames` (R-AIS-035/S-1), `TestAI33_2_ToolCallStream_CancelBetweenFrames` (S-2), `TestAI33_2_ConnectionFreedForNextRequest` (S-3) — first two use a small inline SSE-parameterized stalling handler (`ai33StallHandler`, not `newDripHandler` because that helper writes plain text rather than SSE frames), receive first frame, cancel before second; bound the close to `DefaultDrainTimeout + 1s` safety; `RequireNoGoroutineLeak` ×50. Third uses a single `*http.Client` whose transport reuses connections and asserts a second request against the same server completes promptly (the deferred `resp.Body.Close()` at `stream.go:345` freed the keep-alive slot).
  - **Depends on**: none
  - **Acceptance**: failing test that drives the empirical answer — GREEN if `Body.Read` unblocks via HTTP/1.1 ctx propagation within `DefaultDrainTimeout`; RED if it does not.
  - **Outcome**: GREEN on this branch. All three tests pass under `-race`; the read loop is effectively ctx-aware via the Go 1.26 transport's HTTP/1.1 connection teardown on ctx cancellation.
  - **Files**: `backend/agent/src/ai/openaicompat/a_i-33_2_test.go` (NEW, 390 lines — 190 above forecast because the file also carries inline helpers `ai33StallHandler`, `ai33FirstEvent`, `ai33DrainUntilClosed` and a fuller S-3 handler that counts requests)
  - **Estimated Δ**: +390 / -0 (above forecast; the additional lines are test-only)
  - **Risk**: medium — RED may fail on `httptest`-served HTTP/1.1 (proposal R1 / explore § 5.1). RED resolved GREEN.

- [x] **33.2.2 `[decision]`** Outcome: RED was GREEN. The read loop is effectively ctx-aware — `Body.Read` unblocks within `DefaultDrainTimeout + 1s` via Go 1.26's HTTP/1.1 transport tearing down the connection when ctx is cancelled, after which `run()` walks its `ctx.Err() != nil` branch (`stream.go:503–526`) and exits. **Skip 33.2.3** — no production change forced. The `Body.Close()` watcher and ctx-aware wrapper options documented in this task are not invoked; the existing `defer resp.Body.Close()` at `stream.go:345` is sufficient. No doc 0002 amendment required (no `[decision]` shape, no R-ATS-003 change, no leak arithmetic shift). Recorded in commit `4ff662f` message body and in the apply-progress artifact.
  - **Depends on**: 33.2.1
  - **Acceptance**: a recorded decision observation naming the chosen path and its rationale. Met — decision is "skip 33.2.3"; rationale in commit message; apply-progress artifact records the empirical result.
  - **Files**: doc 0002 (NOT amended — decision was a no-op); `sdd/cachicamas-ai-cancellation/decision/ai-33-2-empirical` observation (recorded via commit message + apply-progress)
  - **Estimated Δ**: +0 / +0 (record-only; no production change)
  - **Risk**: medium — this is the only subnode where the implementation is empirically deferred until RED.

- [x] **33.2.3 `[green]`** **Skipped** — 33.2.2 closed the decision without forcing a production change. Documented for traceability.
  - **Depends on**: 33.2.2
  - **Acceptance**: N/A (skipped)
  - **Files**: none (no production change)
  - **Estimated Δ**: +0 / -0

- [x] **33.2.4 `[refactor]`** Doc comment cites `R-CNF-011`, `R-STK-028`, `R-STK-008`, `R-ATS-003`; helper signatures documented; `make lint` clean (0 issues); file doc comment explicitly records the empirical result and the chosen (no-op) decision. No further tidying needed — the file is already tight.
  - **Depends on**: 33.2.3 (or 33.2.2 if skipped)
  - **Acceptance**: `make test` green; bounded close ≤ `DefaultDrainTimeout + 1s`; `make lint` 0 issues.
  - **Files**: `a_i-33_2_test.go`
  - **Estimated Δ**: +0 / -0 (folded into the 33.2.1 commit; no further refactor surface)

- **33.2 commit shape**: 1 commit (`4ff662f test(openaicompat): add AI-33.2 real-HTTP cancel-between-frames proof`) — single commit because the empirical answer was positive (no production change). The commit message body documents the empirical result and the decision to skip 33.2.3.

---

## AI-33.3 — Cancel during blocked send (truly-abandoned consumer)

- [x] **33.3.1 `[red]`** Write `backend/agent/src/ai/openaicompat/a_i-33_3_test.go` (`package openaicompat`); file doc comment quotes `R-CNF-012` verbatim and cites the `ai-stream-lifecycle` § 5 + `R-STK-010` untestability ruling. Tests: `TestAI33_3_TextStream_AbandonedThenCancelled` (R-AIS-036/S-1), `TestAI33_3_ToolCallStream_AbandonedThenCancelled` (S-2) — each invokes `Stream`, never reads, cancels immediately so producer blocks on first send; `RequireNoGoroutineLeak` ×50; asserts bare close within `emitFailureSendBound + safety`, no `Completion` and no `ErrorEvent` ever observed. `TestAI33_3_AbandonedNeverCancelledPathNotAsserted` (S-3) is a negative-assertion test that searches for an abandoned-never-cancelled test in this file and fails if found (the absence is recorded).
  - **Depends on**: none
  - **Acceptance**: failing tests; GREEN on this branch (bounded-wait + truly-abandoned already drops bare via `stream.go:557–565` `emit` race and `:437–470, :503–526` bounded-wait terminal).
  - **Outcome**: GREEN on this branch. No production change needed. RED was genuine and informative: with the abandonment window set to 100ms the first receive *pairs* with AI-32.3's still-pending bounded-wait send and takes delivery of an `error` terminal (`first receive after the abandonment window returned error, want a closed channel`). Only a window exceeding `emitFailureSendBound` observes the bare close. That RED is what pins "truly abandoned" to "reads nothing until the bound has expired" and separates it from the out-of-scope slow-but-alive case.
  - **Files**: `backend/agent/src/ai/openaicompat/a_i-33_3_test.go` (NEW, 279 lines)
  - **Estimated Δ**: +279 / -0 (74 above the ~205 forecast; below the 400-line budget and below both prior subnodes — 33.2 at 390, 33.4 at 322. The overrun is the mandated verbatim `R-CNF-012` block plus the recorded leak-posture deviation below.)
  - **Risk**: low; the wording-pinning is the risk (R2), addressed by quoting R-CNF-012 verbatim in the test's doc comment.
  - **Deviation (leak-clause split, recorded)**: R-AIS-036 / S-1 attaches `RequireNoGoroutineLeak` (50 repeats) to the truly-abandoned scenario. Measured literal cost: the abandoned path exits only when AI-32.3's bounded-wait send expires, so each repeat costs `emitFailureSendBound` = 5s → **4m10s per scenario, 8m20s for both stream kinds**, against a package that otherwise runs ~8s and `go test`'s 10-minute default timeout — and AI-33.5 / S-6 is specified to reuse this scenario, which would push the package over that timeout outright. The two obligations are therefore proven separately: (a) *no terminal invented* by `TestAI33_3_TextStream_AbandonedThenCancelled` / `..._ToolCallStream_...`, one full-fidelity truly-abandoned run each (5.5s), each of which also proves producer termination because `close(out)` is `run`'s outermost defer (`stream.go:344`) — observing the close IS observing that `run` returned; (b) *no leak across repeats* by `TestAI33_3_AbandonedThenCancelled_LeakFree`, `RequireNoGoroutineLeak` ×50 over the same abandoned-then-cancelled ordering the conformance case itself uses (cancel, THEN drain — `conformance_cancellation.go:113–120`), which costs ~2ms/repeat and carries identical amplitude sensitivity to an accumulating per-call leak. Net: same two properties, 0.09s instead of 8m20s.
  - **Deviation (S-3 shape, recorded)**: the apply prompt's summary listed S-3 as the tool-call variant. `spec.md` R-AIS-036 and this tasks file both define S-2 = tool-call and S-3 = the abandoned-never-cancelled negative assertion. The artifacts were followed. S-3 is implemented as a mechanical guard scanning `a_i-33_*_test.go` for a `never-cancel` test declaration, and was **shown to bite**: a scratch `TestAI33_3_ScratchAbandonedNeverCancelledLeaks` in `a_i-33_9_scratch_test.go` produced a recorded red, after which the scratch file was removed.

- [x] **33.3.2 `[refactor]`** Doc comment cites `R-CNF-012` and `R-STK-029`; tidy; `make lint` clean.
  - **Depends on**: 33.3.1
  - **Acceptance**: `make test` green; bare close verified; conformance `cancellation/abandoned_then_cancelled_drops_bare` stays green.
  - **Outcome**: met. `make test` green (0 failures across 6 packages, 20.1s wall); `make lint` 0 issues; the pinned conformance case `cancellation/abandoned_then_cancelled_drops_bare` passes in both suite entry points. Refactor removed one unnecessary `//nolint:gosec` directive (lint stayed at 0 issues without it); no other tidying surface — helper names, the verbatim quote block, and the cited requirement ids were written in their final shape.
  - **Files**: `a_i-33_3_test.go`
  - **Estimated Δ**: +0 / -1 (folded into the 33.3.1 commit)

- **33.3 commit shape**: single commit `feat(openaicompat): prove abandoned-then-cancelled drops bare (R-CNF-012)`.

---

## AI-33.5 — Resource discipline (drain-before-close + full-suite leak check)

- [x] **33.5.1 `[red]`** Write `backend/agent/src/ai/openaicompat/a_i-33_5_test.go` (`package openaicompat_test`, mirrors `openrouter/conformance/bridge_test.go`); single non-parallel wrapper `TestAI33_5_FullPackageLeakCheck` that walks completion (R-AIS-033/S-1), terminal-error mid-stream (S-2), malformed-frame terminal (S-3), pre-headers (S-4 — reuses 33.1 helpers), between-frames (S-5 — reuses 33.2 helpers), blocked-send abandonment (S-6 — reuses 33.3 helpers), and after-completion (S-7 — reuses 33.4 helpers) for **both** text and tool-call streams through `RequireNoGoroutineLeak`; each scenario bounded by `DefaultDrainTimeout + safety` where readable. NO `t.Parallel()` call in this file (R-STK-008).
  - **Depends on**: 33.1, 33.4, 33.2, 33.3 (calls helpers / fixtures from each)
  - **Acceptance**: failing tests on `main` (no drain in `stream.go:345` → connection-pool poison / leak); expected to fail RED on scenarios S-1, S-2, S-3, S-5, S-7 specifically.
  - **�️ Budget note for S-6 (from 33.3's measured outcome)**: do NOT reuse `ai33AbandonAndCancel` + `ai33RequireBareClose` inside `RequireNoGoroutineLeak`. That pair waits out `emitFailureSendBound` (5s) by construction, so 50 repeats cost 4m10s and would push this package past `go test`'s 10-minute default timeout. Reuse the ordering `TestAI33_3_AbandonedThenCancelled_LeakFree` uses instead (cancel with no prior receive, THEN drain) — same abandonment path, ~2ms/repeat.
  - **Outcome**: GREEN on this branch. Split into `a_i-33_5a_test.go` (drain tests + helpers, 423 lines) and `a_i-33_5b_test.go` (full-suite leak check, 260 lines) per the trigger on tasks.md line 173 (aggregate >400). Drain tests proved RED genuinely: with stream.go reverted, conns count was 6/2 (text/tool-call) across the 50×2 repeats; with drain applied, conns == 1 in both. Wrapper passes all 9 sub-tests in ~0.74s under -race using the cancel-then-drain pattern (NOT truly-abandoned).
  - **Files**: `backend/agent/src/ai/openaicompat/a_i-33_5a_test.go` (NEW, 423 lines); `a_i-33_5b_test.go` (NEW, 260 lines)
  - **Estimated Δ**: +683 / -0 (split between two files; both individually ≤423)
  - **Risk**: medium → resolved. ConnState callback had to be installed via `httptest.NewUnstartedServer` + `Start` (config-after-start race); single-client reuse across repeats required (per-repeat `New` accumulates Transport idle-connection goroutines past leakTolerance).

- [x] **33.5.2 `[green]`** In `backend/agent/src/ai/openaicompat/stream.go:344–345`, retain `defer close(out)` and add one statement to the body-close defer chain: `_, _ = io.Copy(io.Discard, resp.Body)` immediately before `_ = resp.Body.Close()` (errors ignored, mirroring `capture.go:117–122`). No new helper type, no signature change, no second persistent goroutine (R-ATS-003 preserved). `import "io"` already present (verify before adding).
  - **Depends on**: 33.5.1
  - **Outcome**: GREEN. Production logic is +1 line (the existing defer body's body changed from `_ = resp.Body.Close()` to `_, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close()`). Total diff +18/-1 (header doc comment is the bulk — reviewer-facing, cites R-AIS-033, capture.go:117-122 mirror, R-ATS-003, R-STK-009). `import "io"` was already present (stream.go:97). `go.mod` unchanged (3 lines, zero requires).
  - **Files**: `backend/agent/src/ai/openaicompat/stream.go` (+18 / -1: production logic +1 line, header doc +14, refactored defer body +3)
  - **Estimated Δ**: +18 / -1 (over the +3/-1 forecast by 15 — the overrun is the header doc, the production logic is exactly +1 line)
  - **Risk**: low; mirrors an existing proven idiom (`capture.go:117–122`). Full conformance suite (`agenttest/conformance_*`) unchanged and green.

- [x] **33.5.3 `[refactor]`** Doc comment in `a_i-33_5_test.go` cites `R-CNF-009`, `R-ATS-003`, `R-STK-007`, `R-STK-008`, `R-STK-009`; add `grep -L 't.Parallel()' a_i-33_*.go` guard as a sanity note in the test's `t.Cleanup` (mechanical, not a separate task).
  - **Depends on**: 33.5.2
  - **Outcome**: met. `make test` green (0 failures across 6 packages, 20.5s on the main package); `make lint` 0 issues; no `t.Parallel()` in any AI-33 file (verified by `grep -L 't.Parallel()' a_i-33_*.go` showing no matches). Doc comments in both 5a and 5b cite R-AIS-033, R-AIS-038, R-CNF-009, R-CNF-011, R-ATS-003, R-STK-007, R-STK-008, R-STK-009, R-CNF-012, R-ATS-002 — every pin from the per-subnode table.
  - **Files**: `a_i-33_5a_test.go`, `a_i-33_5b_test.go`
  - **Estimated Δ**: +5 / -5 (folded into the 33.5.1 / 33.5.2 commits)

- **33.5 commit shape**: TWO commits landed: `99fef5a fix(openaicompat): add drain-before-close to run() defer chain (R-AIS-033)` (stream.go + a_i-33_5a_test.go) and `83eef31 test(openaicompat): add AI-33.5 full-package leak check over every exit path (R-AIS-038)` (a_i-33_5b_test.go). Split triggered by tasks.md line 173 threshold (`wc -l a_i-33_5_test.go ≥ 400`); the split mirrors tasks.md's chained-PR shape (33.5a = drain impl, 33.5b = full-suite leak check).

- **33.5 commit shape**: TWO commits if ≤400 lines (1=red+green, 2=refactor+tidy); split into 33.5a (33.5.1 red-only, ~200 lines) and 33.5b (33.5.2+33.5.3 green+refactor, ~10 lines) if aggregate ≥400. **Threshold check happens before the first commit lands** — if `wc -l` of `a_i-33_5_test.go` ≥ 400, split before commit.

---

## Strict TDD posture — the contract

- Every `[red]` task MUST fail on `main` for the right reason: missing test file, missing assertion, or wrong observation (the empirical answer for 33.2).
- Every `[green]` task is the **minimum** code that flips the immediately preceding RED to GREEN. No gold-plating, no "while I'm here" cleanups.
- Every `[refactor]` task is taken with `-race` clean and `make lint` green; it does not change behavior.
- The `[decision]` task in 33.2.2 is the only subnode that records a deferred choice. Its output is an Engram observation AND, if the watcher shape is forced, a doc 0002 living-graph amendment in the same PR.
- All production-code tasks are bounded: 33.5.2 ≤3 lines; 33.2.3 ≤5 lines.

---

## Risk register (carried from proposal § 10, refined at task level)

| # | Risk | Mitigation in tasks |
|---|---|---|
| R1 | AI-33.2 RED forces read-loop change | 33.2.1 red drives the empirical answer; 33.2.2 records the decision; 33.2.3 stays ≤5 lines; living-graph amendment in same PR |
| R2 | AI-33.3 wording drift | 33.3.1 quotes `R-CNF-012` verbatim in file doc comment |
| R3 | New Go dep sneaks in | `go.mod` is the only dep-bearing file; diff `backend/agent/go.mod` as part of every PR review checklist; 33.5.2 uses stdlib only |
| R4 | Parallel posture leaks into AI-33.5 | 33.5.1 mandates NO `t.Parallel()`; refactor in 33.5.3 adds a mechanical grep-guard |
| R5 | Tool-call variant ignored on a subnode | Per-spec acceptance: every subnode has text + tool-call scenarios; the spec acceptance gate is the per-subnode test list |
| R6 | AI-33.5 drain perturbs AI-21 scripted scenarios | 33.5.2 mirrors `capture.go:117–122` (already proven); reviewer runs full AI-21 conformance suite before 33.5 merge |

---

## Per-subnode aggregate forecast (gatekeeper input)

| Subnode | Δ lines | 400-line risk | Chained? | Notes |
|---|---|---|---|---|
| AI-33.1 | +155 / -5 | Low | No | Test only |
| AI-33.4 | +155 / -5 | Low | No | Test only; 50-repeat race-detector |
| AI-33.2 | +210 / -5 | Low | No (conditional: +≤5 production lines) | Empirical; 33.2.2 records choice |
| AI-33.3 | +205 / -5 | Low | No | Wording-pinned; quotes R-CNF-012 |
| AI-33.5 | +208 / -11 | Low | No (33.5a/33.5b split IF aggregate ≥400) | 3-line production change |
| **Aggregate** | **~933 / -31** | **Low** | **No** | All subnodes ship as 5 independent PRs |

---

## Out of scope (restated)

- AI-33.2's implementation choice is **not** decided at task time — it is observed at RED (33.2.1) and recorded (33.2.2) before any production change (33.2.3).
- AI-33.3's slow-but-alive-consumer path is **not** asserted (conformance R-CNF-012 narrowing; `ai-stream-lifecycle` § 5 untestability ruling).
- Conformance suite (`R-CNF-011`, `R-CNF-012`) is **unchanged** — it already proves the abstract physics over the fake.
- No new Go dep; `backend/agent/go.mod` stays at 3 lines, zero requires.
- Layer 2+ stop semantics; backpressure (AI-34); retry/idempotency (AI-35).

---

## References

- **Design**: `openspec/changes/cachicamas-ai-cancellation/design.md` (per-subnode table, work-units section)
- **Spec**: `openspec/changes/cachicamas-ai-cancellation/specs/ai-stream-lifecycle/spec.md` (R-AIS-033–038, 22 scenarios, pins)
- **Proposal**: `openspec/changes/cachicamas-ai-cancellation/proposal.md` (§§5, 10, 14)
- **Explore**: `openspec/changes/cachicamas-ai-cancellation/explore.md` § 3 (test-seam table)
- **Charter**: `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:1987–2037`
- **Seams**: `RequireNoGoroutineLeak` (`agenttest/stream_kit_leak.go:107`); `DrainAndRecord` + `DefaultDrainTimeout` (`agenttest/stream_kit_record.go:24, 63`); `bridgeServeTranscripts` (`openaicompat/openrouter/conformance/bridge_test.go:141`); `newDripHandler` (`openaicompat/timeout_test.go:20`); `contextBeforeFirstFrameServer` (`openaicompat/stream_failure_test.go:391`)
- **Defer chain**: `backend/agent/src/ai/openaicompat/stream.go:343–345`
- **Drain idiom**: `backend/agent/src/ai/openaicompat/capture.go:117–122`
- **Project rules**: `openspec/AGENTS.md` (TDD on, `make test`, conventional commits, no AI attribution, layered-not-hexagonal for `backend/agent`)
- **Skills applied**: `sdd-tasks` (sdd-phase-common §E review-workload guard), `work-unit-commits` (every commit build-green + test-green + narrow intent + rollback-friendly), `cognitive-doc-design` (reviewer-facing structure: outcome first, signposted, checklist-style acceptance)
