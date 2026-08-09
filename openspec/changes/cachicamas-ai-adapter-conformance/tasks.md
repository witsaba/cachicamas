# Tasks: AI-38 — Run full deterministic adapter conformance

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 1170–1970 authored (design); goldens excluded from authored count, included in snapshot identity |
| 400-line budget risk | High |
| Chained PRs recommended | No — pre-approved `size:exception` covers the single-PR decision |
| Suggested split | Single PR, 9 internal work-unit commits (WU1–WU9) |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

Decision needed before apply: No — pre-approved
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| WU | Goal | Focused test | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | RED baseline driver | `go test ./openrouter/conformance/... -run FullConformance -race` | same command (real unscoped run) | delete driver, restore skips |
| 2 | Recorder + testdata + drift guard | `go test ./openrouter/conformance/... -run Recorder -race` | `UPDATE_CONFORMANCE_FIXTURES=1` against local `httptest.Server` | delete recorder_test.go, testdata/*.sse; restore literal fixtures |
| 3 | Cancellation amendment | `go test ./agenttest/... -run Cancel -race` | full conformance run | revert conformance_cancellation.go + self-tests |
| 4 | Dialect seam (finish-reason + mid-stream) | `go test ./agenttest/... -run "Dialect|FinishReason|Terminal" -race` | full conformance run | revert conformance_suite/_capabilities/_terminal.go |
| 5 | Bridge rendering (error+usage) | `go test ./openrouter/conformance/... -run "Usage|Terminal|Redaction" -race` | full conformance run | revert renderScript changes |
| 6 | Retry extraction + parity | `go test ./openaicompat/... -run Retry -race` | full conformance run, both binaries | delete conformancetest pkg, revert Retry flips |
| 7 | Capability record generation | `go test ./openrouter/conformance/... -run CapabilityRecord -race` | full conformance run | revert capability_record_test.go |
| 8 | Boundary sweep | `go test ./openrouter/conformance/... -run BoundarySweep -race` | full offset sweep, measured runtime | delete boundary_sweep_test.go |
| 9 | Spec traceability check | n/a — doc only | N/A, no runtime behavior | revert specs/**/spec.md |

## Phase 0: RED Baseline (WU1)

- [x] 0.1 `openrouter/conformance/run_for_test.go`: replace 3 `t.Skip` drivers with `TestOpenRouterAdapter_FullConformance`, unscoped `agenttest.RunConformance`, no `t.Parallel()`. RED: fails ≥5 families (cancellation, terminal, finish_reason/refusal, usage/absent_vs_zero, redaction) — this run IS the binding evidence. Maps R-ACR-001/S-ACR-001-003; R-CNF-028/S-CNF-085-086.
- [x] 0.2 Record exact failing case names/messages as resolution input for Phases 2–6. Maps R-ACR-002/S-ACR-004-005.

  **RED baseline evidence** (`go test ./src/ai/openaicompat/openrouter/conformance/... -run TestOpenRouterAdapter_FullConformance -race -v`, exit 1) — exactly the five predicted families, no divergence:
  1. **cancellation** — `cancellation/bounded_close_leak_free` (`conformance_cancellation.go:80`: "received N further event(s) after cancelling an unbuffered stream, want 0") and `cancellation/abandoned_then_cancelled_drops_bare` (`:123`: "an error terminal was invented for the abandoned-then-cancelled path, want none"); both also trip `RequireNoGoroutineLeak` (`conformance_suite.go:510`) as a downstream symptom of the same extra-terminal-event root cause, re-measured after the Phase 2 amendment.
  2. **finish_reason/refusal** — `finish_reason/all_seven_values_reachable_drift_guarded/{refusal,pause_turn,unknown}` (`conformance_capabilities.go:280/283`: "event 1 kind = error, want completion").
  3. **usage/absent_vs_zero** — `usage/absent_vs_zero_distinguishable` (`conformance_suite.go:510`: "Usage().Output reports absent, want present — an explicit Tokens(0) was scripted").
  4. **redaction** — `redaction/sentinel_absent_from_every_rendering/*` (all nine categories; `conformance_redaction.go:202`: "bridge: renderScript: unsupported event kind error — outside this bridge's scope").
  5. **terminal** — `terminal/exactly_one_terminal_every_path/mid_stream_failure` (`conformance_terminal.go:80`), `terminal/partial_output_discriminator_both_states/{content_preceded,none_preceded}` (`:121`/`:145`), `terminal/all_nine_failure_categories_exhaustive/*/mid_stream` (`:195`, all nine categories) — all the same `bridge: renderScript: unsupported event kind error` root cause as redaction.

  Everything else passes: `text/*`, `tool_call/*`, `finish_reason/{stop,length,tool_calls,content_filter}`, `terminal/exactly_one_terminal_every_path/{normal_finish,pre_stream_failure}`, `terminal/all_nine_failure_categories_exhaustive/*/pre_stream`. `reasoning`/`token_counting`/`cache_boundary` are declared-absent skips (expected, not failures). **No divergence from the design's predicted set — proceeding to Phase 1 per the maintainer's escalation rule.**

## Phase 1: Recorder + Fixtures (WU2)

- [x] 1.1 `openrouter/conformance/recorder_test.go` (new): `recordTranscript(tb, url, cred, body) []byte`, raw `io.ReadAll`, zero parsing. RED: no recorder exists → GREEN: `UPDATE_CONFORMANCE_FIXTURES=1` writes `testdata/*.sse`; unset asserts byte-identity. Maps R-ACR-003/S-ACR-006-007/009.
- [x] 1.2 Prove drift guard: hand-edit one committed byte in a throwaway run, confirm named failure, revert. GREEN: guard names drifted file. Maps S-ACR-008.
- [x] 1.3 `fixtures/*.go`: string literals → `go:embed testdata/*.sse`, accessors kept. Maps R-OR-06 (regenerable transcripts).

  **Evidence**: RED — `go vet` on the new `recorder_test.go` failed `undefined: recordTranscript` before the function was added. GREEN — `go test ./.../conformance/... -run TestRecordTranscript_RegeneratesEveryFixture -race -v`: both `text_stream` and `tool_call` subtests PASS (real HTTP capture byte-identical to the committed, embedded golden). Triangulated: ran again with `UPDATE_CONFORMANCE_FIXTURES=1` — wrote both files, `md5sum` before/after identical (idempotent regeneration), `git status` reported no change. Drift guard proof (1.2): hand-corrupted one byte in `fixtures/testdata/text_stream.sse` (`alpha`→`alphX`), reran → `--- FAIL ... text_stream`, message "fixtures/testdata/text_stream.sse has drifted from the recording helper's output (R-ACR-003 drift guard, S-ACR-008)" — names the exact drifted file; reverted the byte, reran → PASS again.

  **Path correction from design**: `testdata/*.sse` lives at `fixtures/testdata/*.sse`, not `conformance/testdata/*.sse` as design's file-changes table listed — Go's `//go:embed` directive cannot reference a parent directory (no `..` path elements permitted), so the embedding files in `fixtures/*.go` require `testdata/` nested under `fixtures/`. Noted here per "note design gaps, don't silently deviate"; behavior and accessor signatures are unaffected.

  **Scope note**: only `text_stream` and `tool_call` fixtures are recorder-verified in this work unit. `with_usage.sse` stays outside the round-trip until Phase 5/WU5 lands D6's usage rendering (the current `bridgeWriteTerminalChunk` renders no `usage` key at all, so a live capture would not match the existing all-fields-populated golden). `reasoning_extension.sse` is permanently outside the round-trip — `reasoning_details`/`reasoning` are vendor wire-extension fields with no `ai.Event` equivalent, so `bridgeRenderScript` cannot produce them from any `agenttest.Script`. Both fixtures still moved to `go:embed` storage (task 1.3's literal instruction); both package doc comments explain the exclusion. Bootstrap of the initial `testdata/*.sse` byte content used a throwaway `_test.go` in `fixtures/` (real Go compilation dumped the existing literal vars to disk, then the throwaway file was deleted) — avoids any hand-transcription risk of the escaped string literals.

  Pre-existing, repo-wide gofmt "missing trailing newline" convention observed across `backend/agent` (dozens of files, confirmed present before this branch via `git stash`); `.golangci.yml` enables only `govet, errcheck, staticcheck, unused, revive` — no formatting linter — so this does not affect `make lint`. Left unchanged outside the files this work unit substantively modifies.

## Phase 2: Cancellation (WU3)

- [x] 2.1 `agenttest/conformance_cancellation.go`: add `requireCancellationTail(t, rec)` — bare close OR one cancellation-category terminal (+ optional block-close events), replacing :79-81/:121-128. RED: Phase 0 cancellation failure → GREEN: bounded_close/abandoned pass. Maps R-CNF-011/012, S-CNF-026-031/082-083.
- [x] 2.2 `agenttest/conformance_suite_test.go`: self-tests against hand-built doubles (two terminals, wrong category, post-terminal event all rejected); fake stays green. Maps S-CNF-082/083.

  **Evidence**: GREEN — `go test ./.../conformance/... -run "TestOpenRouterAdapter_FullConformance/cancellation" -race -v`: both `bounded_close_leak_free` and `abandoned_then_cancelled_drops_bare` PASS (stable across 3 repeated `-count=1` runs). Safety net: `go test ./src/agenttest/... -race` green before and after (0 → 0 failures), including the two pre-existing fake-factory regression tests. 6 new self-tests in `conformance_suite_test.go` (`TestRequireCancellationTail_*`) via `probeTB`, all PASS: bare-close admitted, single correctly-placed/categorized terminal admitted, two terminals rejected (names count), wrong category rejected (names both), event-after-terminal rejected (names non-final position), invented Completion rejected (names it).

  **Design refinement (discovered via Safety Net, not silent)**: design D4's literal text — "bare close OR exactly one terminal, optionally preceded ONLY by block-closing end events" — regressed two pre-existing green fake-factory tests and, empirically, does not hold against the real bridge either. Root cause: `cancellationAbandonedThenCancelledCase`'s zero-buffer, zero-prior-read construction means the very first scripted/decoded event(s) can race the producer noticing `ctx.Done()` (Go `select` has no case priority) — confirmed on AI-21's fake (an unscripted `TextBlockStart`/`ResponseStart` can precede a bare close) AND on the real adapter (a small transcript can arrive and decode multiple events from one HTTP read before the post-cancel drain starts, so ordinary content — not just block-closes — can precede the terminal). `requireCancellationTail` was refined to enforce exactly the properties both R-CNF-011/012's text and the LOCKED decision actually require — no invented Completion, at most one error terminal, that terminal (if any) is cancellation-category and final — while tolerating an ordinary non-terminal prefix on either side of the race. This is a narrowing of the ADMISSION SHAPE, not of the rejection guarantees: every case task 2.2 enumerates (two terminals, wrong category, post-terminal event, invented Completion) still fails exactly as specified.

  **Additional discovery required for GREEN (bridge_test.go, outside D4's own file list)**: `RequireNoGoroutineLeak` calls `scenario` (which calls `Factory.New`) 50 times against the same `t`, and `tb.Cleanup` callbacks all run at subtest end, not incrementally. Against `conformanceBridgeFactory`'s real `httptest.Server`-per-`New()`-call shape, this left up to 50 concurrently-open servers alive at the leak measurement — a test-harness artifact (`RequireNoGoroutineLeak: goroutine count grew from 4 to 204`/`4 to 54`), never observable before AI-38 since AI-21's in-process fake has no such per-call resource cost. Fixed by having `New` close the previous server it built (under a closure-local mutex) before returning the new one — safe because every case's scenario is fully synchronous before the next `New` call — with exactly one `tb.Cleanup` for the final server. Zero production code touched; test-only, contained to `bridge_test.go`.

## Phase 3: Dialect Seam (WU4)

- [ ] 3.1 `agenttest/conformance_suite.go`: add nil-safe `Dialect *DialectConstraints{UnreachableFinishReasons, MidStreamCategoryCollapse}` on `Factory`. GREEN: existing factories unaffected.
- [ ] 3.2 `agenttest/conformance_capabilities.go`: unreachable-declared reasons assert negative (one typed failure terminal, no Completion); subset-of-hand-list check intact. RED: Phase 0 finish_reason/refusal → GREEN: dialect reasons pass negative assertion. Maps R-CNF-016/S-CNF-042-046/084.
- [ ] 3.3 `agenttest/conformance_terminal.go`: mid-stream collapse branch + `requireFailureCategoryCoverage` counts collapse covered; pre-stream half unchanged. RED: Phase 0 terminal-category failures → GREEN: exactly one typed terminal at declared collapse. Maps R-CNF-010/S-CNF-024-025/087.
- [ ] 3.4 `openrouter/conformance/bridge_test.go`: declare `Dialect{Unreachable:{Refusal,PauseTurn,Unknown}, Collapse:Unknown}` matching `failureFromErrorFrame`.
- [ ] 3.5 `agenttest/conformance_suite_test.go`: self-tests proving S-CNF-084/087 anti-escape (dialect-expressible-but-unproduced still fails).

## Phase 4: Bridge Rendering (WU5)

- [ ] 4.1 `openrouter/conformance/bridge_test.go` `renderScript`: `EventKindError` → in-band error frame (fixed label per category, unwrapped cause+request id), stop rendering; sentinels stay off wire except sanctioned `RawLabel`. RED: Phase 0 redaction Fatalf → GREEN: redaction/leak-scan + terminal/exactly_one pass. Maps D6, redaction/terminal cases.
- [ ] 4.2 Same file: `writeTerminalChunk` renders `usage` present-fields-only (absent ⇒ key omitted, zero ⇒ `0`). RED: Phase 0 usage/absent_vs_zero → GREEN: distinguishable. Maps R-CNF-016/S-CNF-045.

## Phase 5: Retry Parity (WU6)

- [ ] 5.1 New pkg `openaicompat/conformancetest/`: move `retryAutoRetryUpToBoundCase` + helpers, register via `agenttest.RegisterConformanceCase` in `init()`; verify `import_boundary_test.go` admits it (fallback: duplicate + drift test). Maps R-ACR-006.
- [ ] 5.2 `openaicompat/conformance_retry_test.go`: shrink to thin `TestRetryCaseBody_RunsDirectly` importing `conformancetest`.
- [ ] 5.3 `openaicompat/bridge_test.go` + `openrouter/conformance/bridge_test.go`: flip `Retry` → `&true` both factories; blank-import `conformancetest` in OpenRouter binary. RED: factory disagreement/CAP-O-04 NotExercised → GREEN: identical declarations, CAP-O-04 exercised. Maps R-ACR-006/S-ACR-017-018.

## Phase 6: Capability Record (WU7)

- [ ] 6.1 `openrouter/conformance/capability_record_test.go`: `expectedOpenRouterRecord()` (CAP-R-01-05 Satisfied, CAP-O-01/02/03 Absent, CAP-O-04 Satisfied); assert nil diffs + `Verdict()==VerdictPass`; shrink pointer checks to factory-shape sanity. RED: no generated-record assertion at Phase 0 → GREEN: full-run record matches. Maps R-ACR-004/S-ACR-010-013; R-OR-05/06.
- [ ] 6.2 Force a temporary `CAP-O-01=Satisfied` stub to prove the block fires naming the AI-29 trigger, then revert. Maps S-ACR-013.

## Phase 7: Boundary Sweep (WU8)

- [ ] 7.1 `openrouter/conformance/boundary_sweep_test.go` (new): event-level, every offset 0…len per transcript via split-write `httptest.Server`; drained events equal canonical replay; anchor ≥1 transcript to a hard-coded expected event list. RED: no sweep exists → GREEN: every offset matches, anchor non-vacuous. Maps R-ACR-005/S-ACR-014-015.
- [ ] 7.2 **Measure** full offset cross-product runtime under `-race` first. If <30s added, keep exhaustive record-level replay at every offset; only if measured budget is exceeded, fall back to sampled offsets `{1, len/2, len-1}` and record the bound + sampling rule. Maps R-ACR-005 sampling clause/S-ACR-016.
- [ ] 7.3 Local `checkConformanceSweepBound`: ≤1024B/transcript, mirrors `openaicompat_test`'s unexported `checkSweepFixtureBound`. Maps S-ACR-016.

## Phase 8: Cross-Cutting Verification

- [ ] 8.1 `make test` (repo root, `-race`): zero `t.Skip` in conformance drivers, AI-21 fake still green. Maps NFR-ACR-A/S-ACR-019, R-ACR-001/S-ACR-002.
- [ ] 8.2 `make lint`; confirm `backend/agent/go.mod` zero-`require`; confirm no edit under `openspec/specs/**`. Maps S-ACR-020/021.
- [ ] 8.3 If Phase 0 evidence diverges from the predicted 5-family set, escalate per design before continuing — do not silently widen resolutions.
