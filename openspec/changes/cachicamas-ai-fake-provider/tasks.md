# Tasks: The scripted fake provider (AI-21)

> Test command for every RED/GREEN step: `go test -race -run <Test...> ./...` from `backend/agent/`.
> Milestone gate: `make test` (`go test -race -v ./...`) + `make lint` from `backend/agent/`.
> Size-budget note: this doc exceeds the skill's default 530-word guidance by design — AI-21 is
> foundational (20 requirements/59 scenarios/8 leaves) and NFR-AFP-F mandates a recorded RED/GREEN/
> REFACTOR per test-list item; compressing that would violate S-AFP-059.

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (AI-21 alone) | ~900–1400 (impl ~300–350: `fake_provider.go`/`fake_script.go`/`fake_gate.go`; tests ~600–1000 across 8 `_test.go` files; `doc.go` ~15–20) |
| Shared wave budget | 5000 lines across AI-21+AI-22+AI-23 (session-cached) |
| 400-line budget risk | High (standalone), acceptable against the wave's 5000-line ceiling |
| Chained PRs recommended | No — wave `delivery_strategy=single-pr` already fixes one PR for AI-21+AI-22+AI-23 |
| Suggested split | Single PR; 9 leaf-scoped commits (below) as the reviewable/rollback boundary |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

```text
Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

Flag to orchestrator: AI-21 alone is expected to consume a large share of the wave's 5000-line
budget (foundational: core physics + 8 script vocabularies). Track cumulative burn before AI-22/AI-23.

### Suggested Work Units (commits within the one PR)

| Unit | Goal | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| 1 | AI-21.1 skeleton core + text script | `go test -race -run TestProvider ./...` | `go test -race -v ./...` (agenttest) | revert `fake_provider.go`,`fake_script.go`,`fake_text_test.go` |
| 2 | AI-21.2 tool call | `go test -race -run TestToolCall ./...` | same | revert `fake_tool_call_test.go` |
| 3 | AI-21.3 terminal error | `go test -race -run TestTerminalError ./...` | same | revert `fake_error_test.go` |
| 4 | AI-21.4 gate/delays | `go test -race -run TestGate ./...` | same | revert `fake_gate.go`,`fake_gate_test.go` |
| 5 | AI-21.5 cancellation | `go test -race -run TestCancel ./...` | same | revert `fake_cancellation_test.go` |
| 6 | AI-21.6 request capture | `go test -race -run TestCapture ./...` | same | revert `fake_request_capture_test.go` |
| 7 | AI-21.7 reasoning | `go test -race -run TestReasoning ./...` | same | revert `fake_reasoning_test.go` |
| 8 | AI-21.8 queue/exhaustion/totality | `go test -race -run TestQueue ./...` | same | revert `fake_queue_test.go` |
| 9 | NFRs: doc.go reframe + full gate | `make test && make lint` | full `go test -race -v ./...` | revert `doc.go` diff only |

---

## Phase 1 — AI-21.1 Walking skeleton (`fake_provider.go`, `fake_script.go`, `fake_text_test.go`)

- [x] 1.1 RED — S-001–003 (R-AFP-001): external-pkg test constructs fake, assigns to `ai.ModelProvider`, asserts no network/clock dependence, byte-identical repeat run. RED output: `go test -race -run 'TestProvider_ConstructedExternally|TestProvider_Type_IsExportedFromTheAgenttestPackage' ./...` → compile failure: `src/agenttest/fake_text_test.go:30:38: undefined: agenttest.Provider`, `...:85:50: undefined: agenttest.Script`, `...:104:45: undefined: agenttest.Step`, `...:105:13: undefined: agenttest.Emit`, `...:119:44: undefined: agenttest.NewProvider` (+ "too many errors"). `FAIL github.com/cachicamas/backend/agent/src/agenttest [build failed]`.
- [x] 1.2 GREEN — implement `Provider`, `NewProvider` satisfying `ai.ModelProvider` in `fake_provider.go`. GREEN output: same command → `--- PASS: TestProvider_Type_IsExportedFromTheAgenttestPackage (0.00s)`, `--- PASS: TestProvider_ConstructedExternally_SatisfiesModelProviderAndRunsWithNoNetwork (0.00s)`, `PASS`, `ok github.com/cachicamas/backend/agent/src/agenttest 1.499s`. Implemented full pre-stream contract (validation/nil-ctx/cancellation), mutex-guarded queue pop + capture, `ErrScriptsExhausted` sentinel (typed-error path not yet scenario-tested — deferred to Phase 8), per-call `ai.Stamper` sequencing, and scriptProvider-shaped producer (goroutine/defer-close/select-on-`ctx.Done()`) — ahead of 1.5's nominal split, since S-AFP-002's "fully scripted stream drained to close, byte-identical repeat" needed a working end-to-end `Stream()`. `Emit`'s malformed-event panic (S-AFP-006) and `ai.CheckStream`-based terminal-exclusivity (S-AFP-021) deliberately deferred to 1.4/1.5 and 3.4/3.5 — not yet driven by a failing test.
- [x] 1.3 REFACTOR — note: none needed; `go vet ./...` clean, full suite green (`go test -race ./...`), code already minimal on first pass.
- [x] 1.4 RED — S-004–006 (R-AFP-002): scripted text drains exact/ordered/sequenced (1…N, no gap/repeat); malformed script event fails loud at the fake. RED output: `go test -race -run 'TestProvider_ScriptedTextResponse_DrainsExactlyAsScriptedAndSequencedOneToN|TestEmit_EventNeverConstructed' -v ./...` → `TestProvider_ScriptedTextResponse_...` **PASS immediately** (S-AFP-004/005: sequencing was already implemented as part of 1.2's front-loaded `Stamper` — honestly recorded, no forced failure, per the tasks doc's own Key Learning); `TestEmit_EventNeverConstructed_...` **FAIL** (genuine RED): `fake_text_test.go:210: Emit(ai.Event{}) did not panic, want a loud failure at the fake (S-AFP-006)`, `--- FAIL`.
- [x] 1.5 GREEN — implement `Script`,`Step`,`Emit` (`fake_script.go`), `Stream()` text path + fresh `ai.Stamper` per call (`fake_provider.go`). GREEN output: added `Emit`'s zero-`Kind()` panic check → same command → `--- PASS: TestEmit_EventNeverConstructed_...`, `--- PASS: TestProvider_ScriptedTextResponse_...`, `PASS`.
- [x] 1.6 REFACTOR — note: none; the panic check is a 3-line guard, already minimal.
- [x] 1.7 RED — S-007–008 (R-AFP-003): two fakes streamed concurrently sequence independently under `-race`; one fake streamed twice restarts at 1. RED output: `go test -race -run TestProvider_ConcurrentAndRepeatedStreams_SequenceIndependently -v ./...` → **PASS immediately** (both subtests), no gap found: per-call `ai.Stamper` isolation from 1.2/1.5 already satisfies this — honestly recorded per Key Learning #5, no forced failure.
- [x] 1.8 GREEN — confirm per-call `Stamper` isolation (from 1.5) satisfies; extend only if a gap surfaces. GREEN output: no extension needed; same command, same PASS output as RED.
- [x] 1.9 REFACTOR — note: none.
- [x] 1.10 RED — S-009–011 (R-AFP-004): exactly one producer goroutine, one closing site across complete/error/cancel under `-race`; producer exits (no forever-block) when consumer stops and caller cancels. RED output: `go test -race -run TestProvider_MidStreamPhysics_OneClosingSiteAcrossCompletionErrorAndCancellation -v ./...` → **PASS immediately**, all 23 subtests (completion / terminal-error / cancel-with-no-reader / 20× repeated-cancel-under-`-race`) — including the terminal-error path, an early cross-confirmation of Phase 3's design (`Emit(ai.ErrorEvent(f))` as a plain last step). No gap found: 1.2/1.5's producer (`go produce(...)`, `defer close(out)`, per-send `select` on `ctx.Done()`) already satisfies R-AFP-004 non-negotiably.
- [x] 1.11 GREEN — implement producer goroutine, `defer close`, per-send `select` on `ctx.Done()` in `fake_provider.go`. GREEN output: no extension needed; same PASS output as RED.
- [x] 1.12 REFACTOR — note: none; `go vet ./...` clean. Post-hoc fix during Phase 2's full-suite check: the "cancelling closes the channel..." subtest was flaky (~3/5 runs failed) because it called `drainFake` immediately after `cancel()`, racing the producer's own `select` exactly as `ai/provider_test.go`'s `settleAfterCancel` doc comment predicts. Added `settleAfterFakeCancel` (documented settling step, S-AFP-054-permitted) between `cancel()` and the confirmation drain; 8/8 repeat runs green after the fix. This is a test-only fix — no production code changed.

## Phase 2 — AI-21.2 Scripted tool call (`fake_tool_call_test.go`)

- [x] 2.1 RED — S-012–013 (R-AFP-005): tool call start→arg-deltas→end reconstructs to exact scripted bytes, identity+name preserved. RED output: `go test -race -run 'TestProvider_ScriptedToolCall|TestProvider_InterleavedToolCalls' -v ./...` → `TestProvider_ScriptedToolCall_DeltaCarrying_...` **PASS immediately**; `TestProvider_ScriptedToolCall_ZeroDelta_...` **FAIL** — but the failure was a genuine bug in the test helper itself (`mustToolCallScript` derived the end event's arguments by concatenating fragments, so a zero-fragment call produced empty arguments instead of the intended full string), not a production gap: `fake_tool_call_test.go:176: zero-delta call's arguments = "", want "{\"q\":\"weather\"}"`.
- [x] 2.2 GREEN — confirm Phase 1's generic `Emit`/`Stream()` already carries tool-call events untouched; extend only if a gap surfaces. GREEN output: no production code changed. Fixed the test helper to take `arguments` independently of `fragments` (matching `NewToolCallEnd`'s own contract: end bytes are never derived from deltas) → same command → all 3 tests `PASS`.
- [x] 2.3 REFACTOR — note: `gofmt -w` re-aligned the `reconstructedToolCall` struct tags after the helper edit; no logic change.
- [x] 2.4 RED — S-014–015 (R-AFP-006): zero-delta start→end tool call reconstructs identically to a delta-carrying call with the same arguments. RED output: covered by the same file/run as 2.1 (`TestProvider_ScriptedToolCall_ZeroDelta_...`, see 2.1's genuine test-helper RED above).
- [x] 2.5 GREEN — confirm/extend as in 2.2. GREEN output: covered by 2.2 — same fix, same passing run.
- [x] 2.6 REFACTOR — note: none beyond 2.3's gofmt pass.
- [x] 2.7 RED — S-016–017 (R-AFP-007): two interleaved tool calls keep distinct ordinals and reconstruct independently, no cross-contamination. RED output: `TestProvider_InterleavedToolCalls_KeepDistinctOrdinalsAndReconstructIndependently` **PASS immediately** — no gap, generic machinery already keeps events and their block indices untouched and in order.
- [x] 2.8 GREEN — confirm/extend as in 2.2. GREEN output: no extension needed.
- [x] 2.9 REFACTOR — note: none.

## Phase 3 — AI-21.3 Scripted terminal error (`fake_error_test.go`)

- [x] 3.1 RED — S-018–019 (R-AFP-008): every AI-19 failure category scriptable with and without prior output; partial-output discriminator matches the actual stream state. RED output: `go test -race -run TestProvider_ScriptedTerminalError_EveryCategory_BothPartialOutputStates -v ./...` → **PASS immediately** on first try for all 9 categories × 2 partial-output states (18 subtests) — `Emit(ai.ErrorEvent(f))` as a plain last step already carries every category untouched, confirming Phase 1's uniform-vocabulary design. (A later full-suite run in 3.5 exposed an unrelated pre-existing test-fixture bug in this same test file — see 3.5's note; not a RED/GREEN reversal of this item.)
- [x] 3.2 GREEN — confirm `Emit(ai.ErrorEvent(f))` as terminal step (per design's uniform-vocabulary decision) already satisfies; extend only if a gap surfaces. GREEN output: no extension needed.
- [x] 3.3 REFACTOR — note: none.
- [x] 3.4 RED — S-020–021 (R-AFP-009): nothing follows a terminal error (closed-stream next receive); an event scripted after a terminal fails loudly and names the misplaced event. RED output: `go test -race -run 'TestProvider_AfterTerminalError|TestProvider_ScriptWithEventAfterTerminalError' -v ./...` → `TestProvider_AfterTerminalError_NextReceiveReportsClosedStream` **PASS immediately** (S-AFP-020, natural consequence of existing physics); `TestProvider_ScriptWithEventAfterTerminalError_...` **FAIL** (genuine RED, S-AFP-021): `fake_error_test.go:162: Stream did not panic for a script with an event after a terminal error, want a loud failure naming the misplaced event`.
- [x] 3.5 GREEN — implement/confirm terminal-exclusivity enforcement (rely on `ai.Stamper`/`CheckEmit` where possible; add fake-side check only if uncovered). GREEN output: added a synchronous `ai.CheckStream(stepEvents(steps))` call in `Stream()`, before spawning the producer (uses `ai.CheckStream`, not `ai.CheckEmit` — `CheckStream` is AI-14.4's ordering-invariant checker over a finite slice, the exported mechanism that actually answers "is this event terminal / already-terminated" without the fake special-casing kinds by name; `CheckEmit` stays unused per design's explicit decision) → panics naming the violation, test passes. **Side effect caught by this same GREEN run**: two earlier test fixtures (Phase 1's physics test and 3.1's category test) used an unclosed `ai.NewTextBlockStart` as a stand-in for "some prior output", which `ai.CheckStream`'s block-ordering rule correctly rejects as an unterminated block — a genuine bug in the test fixtures, not the fake. Fixed by adding `mustPriorOutputEvent` (a non-block `ai.NewResponseStart`) and using it in both places. Full command output after both fixes: `PASS`, `ok github.com/cachicamas/backend/agent/src/agenttest 1.369s`.
- [x] 3.6 REFACTOR — note: extracted `stepEvents(steps []Step) []ai.Event` in `fake_provider.go` for the CheckStream call; `go vet ./...` clean, full suite green.

## Phase 4 — AI-21.4 Delays and the blocked stream (`fake_gate.go`, `fake_gate_test.go`)

- [x] 4.1 RED — S-022–024 (R-AFP-010): held stream blocks a second receive until test-controlled release; releasing drains remaining events in order; no sleep used as coordination anywhere in the assertions. RED output: `go test -race -run 'TestProvider_HeldStream|TestProvider_UnreadConsumer_DeterministicallySaturatesBuffer' ./...` → compile failure: `fake_gate_test.go:26:20: undefined: agenttest.NewGate`, `...:33:13: undefined: agenttest.Hold`. `FAIL [build failed]`.
- [x] 4.2 GREEN — implement `Gate` (`Reached()`,`Release()`, `sync.Once`-guarded) and producer `Hold` handling (`select` on released/`ctx.Done()`) in `fake_gate.go`/`fake_provider.go`. GREEN output: added `fake_gate.go` (`Gate`,`NewGate`,`Reached`,`Release`,`markReached`), extended `Step` with `gate`/`isHold`, added `Hold(g *Gate) Step` in `fake_script.go`, and taught `stampSteps`/`stepEvents`/`produce` in `fake_provider.go` to skip/handle Hold steps → same command → `--- PASS: TestProvider_UnreadConsumer_DeterministicallySaturatesBuffer_...`, `--- PASS: TestProvider_HeldStream_...`, `PASS`. Every wait in the test file is on `gate.Reached()` or `time.After(boundedFakeTimeout)` as a failure deadline — no sleep used as coordination.
- [x] 4.3 REFACTOR — note: none; `go vet ./...` clean.
- [x] 4.4 RED — S-025–026 (R-AFP-011): unread consumer deterministically saturates a buffered script (`Script.Buffer`); a slow-but-resumed (non-cancelling) consumer still receives every event, none dropped. RED output: covered by the same compile-failure RED as 4.1 (`TestProvider_UnreadConsumer_DeterministicallySaturatesBuffer_...` was in the same file, same undefined symbols).
- [x] 4.5 GREEN — implement per-script `Buffer` capacity (`cap(ch)` exactly) in `fake_script.go`/`fake_provider.go`. GREEN output: `Buffer` → `make(chan ai.Event, script.Buffer)` was already wired since Phase 1 (task 1.2); this task's own new work was the Gate/Hold machinery from 4.2, without which the saturation recipe (`n` emits + `Hold` + late emit) could not run at all — genuinely GREEN together with 4.2, not a pass-for-free case. Same PASS output as 4.2.
- [x] 4.6 REFACTOR — note: none.

## Phase 5 — AI-21.5 Cancellation fidelity (`fake_cancellation_test.go`)

- [x] 5.1 RED — S-027–029 (R-AFP-012): mid-script cancel closes within bounded time under `-race`; saturated path drops remaining events and closes bare (no terminal); no send after close. RED output: `go test -race -run 'TestProvider_MidScriptCancellation|TestProvider_PreStreamCancellation' -v -count=1 ./...` → `TestProvider_MidScriptCancellation_...` **PASS immediately**, all 20 `-race` iterations — no gap, Phase 1's per-send `select` on `ctx.Done()` already covers the bare-close-on-saturated-cancel path.
- [x] 5.2 GREEN — confirm/extend Phase 1's per-send `select` on `ctx.Done()` covers the bare-close-on-saturated-cancel path. GREEN output: no extension needed.
- [x] 5.3 REFACTOR — note: none.
- [x] 5.4 RED — S-030–032 (R-AFP-013): already-cancelled ctx + valid request → typed pre-stream failure (cancellation category), no carrier, no extra goroutine under `-race`; already-cancelled ctx + zero-value request → validation failure reported, not cancellation. RED output: same command → `TestProvider_PreStreamCancellation_...`'s failure/category/delivery/validation-ordering assertions **PASS immediately** (Phase 1's pre-stream check ordering already correct); but the naive S-AFP-031 "goroutines counted immediately before and after" check was genuinely flaky (~3/3 failed) — traced to `runtime.NumGoroutine()` being unreliable in this package's own busy, heavily-`t.Parallel()` binary (sibling tests' goroutines skew a single before/after snapshot), not a fake defect; confirmed the ground-truth `ai/provider_test.go` doesn't attempt this numeric check for its own analogous scenario either.
- [x] 5.5 GREEN — confirm/order pre-stream checks in `Stream()`: `IsZero()` validation before `ctx.Err()` cancellation check (`fake_provider.go`). GREEN output: no production code changed (ordering already correct from Phase 1). Rewrote the goroutine-count assertion as a repeated-call amplification (50 pre-stream-failing calls; a per-call leak would grow the count by ~50, background jitter does not, so a `before+25` bound catches a real leak without racing sibling tests) → 5/5 repeat full-suite runs green (`go test -race ./...`).
- [x] 5.6 REFACTOR — note: `gofmt`/`go vet` clean.

## Phase 6 — AI-21.6 Request capture (`fake_request_capture_test.go`)

- [x] 6.1 RED — S-033–035 (R-AFP-014): every request field readable after the call; three ordered calls yield three ordered capture entries; a pre-stream-rejected call (invalid request/cancelled ctx/exhausted queue) is absent and `Requests()[i]` ↔ script `i` correspondence holds. RED output: `go test -race -run 'TestProvider_Requests_' -v ./...` → all 3 tests **PASS immediately** — no gap; capture-on-consume (mutex-guarded, only for a call that pops a script) was already built in Phase 1 (task 1.2). Reused `request_test.go`'s existing `buildFullRequest`/`requireRequestsEqual` for the every-field proof.
- [x] 6.2 GREEN — implement capture-on-consumption (mutex-guarded slice, captured only for calls that pass all pre-stream checks) + `Requests()` in `fake_provider.go`. GREEN output: no extension needed.
- [x] 6.3 REFACTOR — note: none.
- [x] 6.4 RED — S-036–037 (R-AFP-015): caller mutation of its own request/slices after the call does not alter capture history; two reads of the history are independent. RED output: covered by the same run — `TestProvider_Requests_LaterCallerMutation_DoesNotAlterCaptureHistory` **PASS immediately** — no gap; `Requests()`'s `slices.Clone` on the captured slice, combined with `ai.Request`'s own immutable accessors, already gives both properties for free.
- [x] 6.5 GREEN — implement clone-on-read for `Requests()` (`slices.Clone` per AI-10.6 accessor pattern). GREEN output: no extension needed.
- [x] 6.6 REFACTOR — note: none; `go vet ./...` clean, full suite green.

## Phase 7 — AI-21.7 Scripted reasoning (`fake_reasoning_test.go`)

- [x] 7.1 RED — S-038–039 (R-AFP-016): reasoning deltas + terminal round-trip token drain byte-exact, including non-text-valid bytes surviving unchanged. RED output: `go test -race -run 'TestProvider_ScriptedReasoning|TestProvider_MixedReasoningAndText' -v ./...` → `TestProvider_ScriptedReasoning_DeltasAndRoundTripToken_ByteExact` **PASS immediately**, including a token of invalid-UTF-8 bytes (`{0xff,0xfe,'a',0x00,'b'}`) surviving byte-exact — no gap.
- [x] 7.2 GREEN — confirm Phase 1's generic `Emit`/`Stream()` carries reasoning events untouched; extend only if a gap surfaces. GREEN output: no extension needed.
- [x] 7.3 REFACTOR — note: none.
- [x] 7.4 RED — S-040–042 (R-AFP-017): redacted block (no visible fragment) and signature-only block scriptable and drainable; a visible fragment inside a redacted block fails loudly, naming the violation. RED output: `TestProvider_ScriptedReasoning_RedactedAndSignatureOnlyShapes` (3 subtests) **PASS immediately** — the misplaced-fragment case is caught by `ai.NewReasoningDelta`'s own construction-time `ErrMisplaced` rule before a test can even build the `Step`, exactly matching design's "events come from ai constructors, already validated" decision; no fake-side check needed or added.
- [x] 7.5 GREEN — confirm/extend as in 7.2; rely on `ai` constructors' own shape validation. GREEN output: no extension needed.
- [x] 7.6 REFACTOR — note: none.
- [x] 7.7 RED — S-043–044 (R-AFP-018): a stream mixing reasoning and text deltas — no reasoning fragment/token in collected text events, and no text fragment in collected reasoning events. RED output: `TestProvider_MixedReasoningAndText_NeverCrossesIntoTheOtherEventKind` **PASS immediately** — no gap; distinct marker strings (`REASONING_ONLY_FRAGMENT`/`TEXT_ONLY_FRAGMENT`) confirmed absent from the wrong-kind event's own accessors.
- [x] 7.8 GREEN — confirm the type-level wall (distinct `ai` event types) holds; extend only if a gap surfaces. GREEN output: no extension needed — the wall is structural (a payload type assertion cannot succeed across kinds).
- [x] 7.9 REFACTOR — note: none; `go vet ./...` clean, full suite green.

## Phase 8 — AI-21.8 Sequential-call scripting, exhaustion, totality (`fake_queue_test.go`)

- [x] 8.1 RED — S-045–047 (R-AFP-019): consecutive calls on one fake consume consecutive scripts in enqueue order, each exactly once; queue inspection after N calls shows the correct remainder. RED output: `go test -race -run 'TestProvider_ConsecutiveCalls|TestProvider_CallAgainstExhaustedQueue|TestProvider_ExtremeInputs' -v ./...` → `TestProvider_ConsecutiveCalls_...` (3 subtests) **PASS immediately** — no gap; the mutex-guarded pop-once index from Phase 1 already satisfies this. "Queue inspection" has no dedicated accessor in design's exported surface, so the 3rd subtest observes remainder through the queue's own behavior: a 3rd call on a 3-script queue succeeds (proving one remained), a 4th fails with `ErrScriptsExhausted` (proving none do).
- [x] 8.2 GREEN — implement mutex-guarded queue index advance (pop-once, no reorder, no replay) in `fake_provider.go`. GREEN output: no extension needed.
- [x] 8.3 REFACTOR — note: none.
- [x] 8.4 RED — S-048–050 (R-AFP-020): a call against an exhausted queue fails loudly and immediately (bounded deadline never reached), names the fake and the exhaustion, does not replay the last script, does not return a clean-closing empty stream. RED output: same command → `TestProvider_CallAgainstExhaustedQueue_...` **PASS immediately** — no gap; `ErrScriptsExhausted`/`fmt.Errorf` path from Phase 1 already satisfies it, proven here via a goroutine + bounded-deadline race (the exhausted call returns before the deadline, `ch2` is nil, `errors.Is(err2, agenttest.ErrScriptsExhausted)`).
- [x] 8.5 GREEN — implement `var ErrScriptsExhausted = errors.New(...)`; `Stream()` returns `nil, fmt.Errorf("...call %d of %d...: %w", ..., ErrScriptsExhausted)`, checked after the three contract checks (validation, ctx, and exhaustion last). GREEN output: no extension needed.
- [x] 8.6 REFACTOR — note: none.
- [x] 8.7 RED — S-056–057 (NFR-AFP-D totality): table of extreme inputs (zero-value request, nil ctx, cancelled ctx, empty script, never-populated queue) through every exported entry point → no panic except the documented `ErrScriptsExhausted` path; doc surface states the exhaustion behavior explicitly. RED output: same command → `TestProvider_ExtremeInputs_...` (6 subtests covering all 5 documented extreme inputs plus a fresh-`Requests()` check) **PASS immediately** for the no-panic behavior — no code gap. The doc-surface half of S-AFP-057 (exhaustion stated explicitly) was genuinely incomplete: `ErrScriptsExhausted`'s doc comment named the mechanism but not the full observable behavior (nil channel, non-replay, non-clean-close, immediacy).
- [x] 8.8 GREEN — close any panic gap found; add exhaustion-behavior doc comment on the fake's exported surface (e.g. `Stream()`/`ErrScriptsExhausted`). GREEN output: no panic gap to close. Added an "Exhaustion behavior, stated explicitly (NFR-AFP-D)" section to `ErrScriptsExhausted`'s doc comment (nil channel, no goroutine, no replay, no clean empty close, error format) and a cross-reference from `Stream()`'s own doc comment. Verified rendering with `go doc ./src/agenttest ErrScriptsExhausted`.
- [x] 8.9 REFACTOR — note: none; `gofmt`/`go vet` clean, full suite green.

## Phase 9 — Non-functional requirements and final gate

- [ ] 9.1 Verify NFR-A/S-051 — `backend/agent/go.mod` still zero `require`; both AI-00 import guards pass; `agenttest` imports only stdlib + `ai`.
- [ ] 9.2 Verify NFR-B/S-052–053 — no diff under `backend/agent/src/ai/`; AI-20 signature guard passes; `src/ai` tests pass unaffected.
- [ ] 9.3 Modify NFR-E/S-058 — `backend/agent/src/agenttest/doc.go`: state both roles (Layer-1 proof + importable testing library), retain the ADR 0005 Guard C sibling-layout note, state the contract-faithful-not-convenient rule.
- [ ] 9.4 Verify NFR-C/S-054–055 — run the full suite twice under `-race`, results identical; grep new `fake_*_test.go` files for `time.Sleep` used as scheduling (only bounded deadlines/documented settle steps allowed).
- [ ] 9.5 Final gate/NFR-F/S-059 — `make test` green, `make lint` clean from `backend/agent/`; confirm every RED/GREEN/REFACTOR field above is filled before closing the milestone.

## Key Learnings

1. Spec assigns granularity by requirement group, so a tasks.md test-list item may legitimately span several scenario IDs.
2. Several AI-21 leaves (tool call, terminal error, reasoning) are expected to pass immediately on the generic `Emit`/`Stream()` skeleton from AI-21.1, since the design's uniform-Emit-vocabulary decision makes new event shapes structurally free.
3. The wave's single-pr delivery strategy makes leaf-scoped commits, not separate PRs, the correct chaining unit for this change.
4. NFR-AFP-D's totality requirement folds naturally into AI-21.8's test file since its one exception is that leaf's own exhaustion failure.
5. Strict TDD here permits a RED phase that passes immediately without new code — that outcome must be recorded honestly, not forced into a false failure.
