# Exploration — AG-15: retry policy and the failover seam (`cachicamas-agent-retry-failover`)

> Phase: sdd-explore · Change: `cachicamas-agent-retry-failover` · Milestone AG-15, doc 0003 lines 1444–1519
> Worktree: `cachicamas-worktrees/ag-15-retry-failover`, branch `feat/agent-layer2-wave3-ag15`, base `main@bf482b0a`
> Depends on: AG-11 (turn termination, archived), AG-13 (run driver, archived). Parallel with AG-14 (cancellation tree, merged `6485b937`).
> Artifact store: hybrid — this file is normative; Engram topic `sdd/cachicamas-agent-retry-failover/explore` mirrors it.

## 1. Existing typed-evidence surfaces AG-15 must consume

**Layer 1 (`ai.Failure`, `backend/agent/src/ai/provider_failure.go:320-330`)** — full evidence, nil-safe accessors:

- `Category() FailureCategory` — closed 9-member vocabulary
- `Retryable() bool`
- `RetryAfter() (time.Duration, bool)` — `provider_failure.go:471-476`, presence-typed (absent / reported-zero / reported-nonzero are three distinguishable readings)
- `PartialOutput() bool`
- `Delivery() DeliveryPath` — pre-stream vs mid-stream

**Layer 2 wrapper (`agent.Failure`, `backend/agent/src/agent/failure.go:24-92`)** delegates only four of the five: `Category()` (`:44-49`), `Delivery()` (`:54-59`), `Retryable()` (`:64-69`), `PartialOutput()` (`:76-81`), plus `Unwrap() error` (`:87-92`, unwrapping to the raw `*ai.Failure`).

**`RetryAfter()` is NOT delegated** — a genuine gap in the clean Layer-2 envelope type, unlike the other four which mirror `R-ATT-006`'s pattern exactly. A stream consumer holding only `TurnEnd.Failure()` / `RunEnd.Failure()` (the `*agent.Failure` the event payload carries) cannot read retry-after without reaching past the wrapper via `Unwrap()` and a type assertion. **Design constraint:** AG-15 (or a delta on `agent-turn-termination`) should either add `func (f *Failure) RetryAfter() (time.Duration, bool)` mirroring the other four, or explicitly document the harness-only reach below as sufficient.

**The harness's own reach is wider than the wrapper**, which resolves the immediate blocking question: `loop.go:387` type-asserts `turn.fatal.(*ai.Failure)` directly on the mid-stream fatal path, and `Turn` returns that same `turn.fatal` as its raw `error` (`loop.go:411,414`). So `Harness.Run`'s `terr` (`harness.go:447`) is itself type-assertable to `*ai.Failure` **before** any Layer-2 wrapping — giving AG-15's retry predicate full access to `Category()`, `Retryable()`, `RetryAfter()`, `PartialOutput()` at the harness's own decision point. The gap above is only in the **event-stream-observable** envelope, not the harness's internal decision path.

## 2. Where AG-13's run driver ends a turn — the retry insertion point

`Harness.Run` (`backend/agent/src/agent/harness.go:311-500`). The loop body calls `Turn(...)` once per iteration (`:447`). On `terr != nil` (`:450-470`):

- It first re-checks `context.Cause(runCtx)` for the two AG-14 cancellation sentinels and routes to `windDownRun` if matched (`:460-462`). **AG-15 MUST NOT retry a cancelled turn** — this carve-out is already ordered ahead of any retry decision and must stay ahead of it.
- Otherwise it calls `h.failRun(sink, stamper, runID, terr)` unconditionally (`:469`). This is `R-RUN-011`'s failure path and the exact site AG-15.1's predicate must intercept: between the cancellation check (`:460`) and the unconditional `failRun` (`:469`), a retry decision must either loop back to call `Turn` again over an unmodified transcript with an incremented attempt counter, or fall through to `failRun` as today.
- `failRun` (`:293-300`) wraps `terr` via `wrapHarnessFailure` (`:250-259`), which **unconditionally overwrites the category to `ai.FailureCategoryUnavailable`** regardless of the real category. AG-15 must read the real category/retryability from `terr` before this wrapping discards it, or restructure `wrapHarnessFailure` to preserve it for the exhausted-retry terminal report.

**Lane/bracket-ordering constraint (`R-RUN-003`, `agent-run-driver/spec.md:119-129`):** the harness owns exactly one run bracket, N turn brackets (one per `Turn` invocation), one contiguous 1-based lane via one shared `LaneStamper` injected into every `Turn` call. A retry that re-invokes `Turn` naturally produces its own `turn_start`/`turn_end` pair on the same lane — already stream-visible via sequence number and turn identity, with **no new `EventKind` apparently required** for the "each attempt is visible" charter clause (see §3).

### CONFIRMED BLOCKER — pre-stream failures never emit `turn_end`

`Turn` (`loop.go:248-339`) emits `turn_start` unconditionally at `:296-301`, **before** `buildLoopRequest` (`:304`), the pre-request hook (`:317-327`), and `provider.Stream` (`:332`). On every failure at or before `provider.Stream` returns — build error (`:304-308`), hook error (`:317-327`), `streamErr != nil` (`:332-338`) — `Turn` calls `closeSink(sink)` and returns the bare error **with no `turn_end` ever emitted**. The turn bracket is left permanently open on the stream. The comment at `:334-338` names this "V-PRV-04", the original AG-07 walking-skeleton behavior, unchanged since. Contrast the mid-stream fatal path (`:374-415`, `R-ATT-005`), which does emit `turn_end(Aborted)` before returning.

This is exactly the case AG-15.1's first scenario needs ("retryable-with-no-output retries visibly" — a pre-output, hence typically pre-stream, retryable failure).

**Verified against `stream_check.go` (read directly, 193 lines):**

- `stream_check.go:79-81` documents the rejection: "opening the run bracket twice, closing it with a still-open turn, **opening a second turn before the first closes**, or closing a turn that is not open".
- `stream_check.go:141-143` implements it: on `BracketRoleOpensTurn`, `if turnOpen { return violation(ai.ErrMisplaced, ai.AtIndex("event", i)) }`.

So a harness-level retry that re-invokes `Turn` after a pre-stream failure emits a second `turn_start` while `turnOpen` is still true, and `CheckStream` rejects the recorded stream with `ai.ErrMisplaced`. This is **not** a tolerance question — it is a hard violation.

`sdd-design` MUST resolve this. The most likely resolution is extending `Turn`'s pre-stream-failure paths to also emit `turn_end(Aborted, failure)` — a new requirement/delta on `agent-loop-skeleton`, mirroring how AG-11 amended `R-LSK-004` for the mid-stream case. `R-RUN-003` additionally pins `stream_check.go` as byte-unchanged, so the fix must land on the emitter side, never on the validator.

## 3. "Each attempt is visible" in the existing event vocabulary

25 registered `EventKind` values (`event.go:100-221`), unchanged since AG-14 (matches `agent-turn-termination/spec.md`'s "25 kinds" pin). No retry-specific kind exists. `TurnStart{}` (`turn_events.go:28-31`) carries no payload data at all — only envelope identity (run, turn via `Event.Turn()`).

A retry re-invoking `Turn` produces a fresh `TurnID` and its own `turn_start`/`turn_end` pair, distinguishable by sequence number and turn identity alone. This is likely sufficient to satisfy "attempts are visible, not silent" with **zero new `EventKind`**, consistent with `R-RUN-012`'s no-new-EventKind pattern that AG-13 and AG-14 both held to.

Open question for `sdd-design`: does a consumer need to know a `turn_start` is specifically a *retry* of the same logical turn, versus a brand-new turn from a `ToolCalls`/`PauseTurn` continuation? Nothing in the charter explicitly requires that distinction to be labeled. If it is needed, the smallest fix is a design-owned convention: every retried turn's transcript is byte-identical to the failed one ("a fresh provider call over an identical transcript", per the charter's Given clause), so a consumer can infer a retry by request-transcript equality without a new field.

## 4. Layer 1's existing wire-level retry mechanics — the composed ceiling

`backend/agent/src/ai/internal/retry/retry.go` (full file read):

- `DefaultMaxAttempts = 3` (`:18`) — retries permitted **after** the initial request, so a logical call issues at most `3 + 1 = 4` wire requests when every attempt fails retryably.
- `Config{MaxAttempts, NowFunc, SleepFunc, AfterReader, BaseDelay, MaxDelay, JitterSeed}` (`:37-45`) — a full clock/timing injection seam already exists at Layer 1. `NowFunc` seeds jitter, `SleepFunc` waits (context-aware), `AfterReader` reads the retry-after hint from an `*ai.Failure` (defaulting to `f.RetryAfter()`, `:209-211`).
- `Loop(ctx, body, cfg, executeOnce)` (`:72-117`) is the retry driver: exponential backoff with jitter (`computeBackoff`, `:169-190`), and **retry-after overrides computed backoff** when present (`:102-110`) — the exact behavior AG-15.2's charter scenario 1 requires at the harness level too.
- **The call site is fixed, not overridable today:** `openaicompat/stream.go:240` calls `retry.Loop(ctx, body, retry.Config{}, c.executeOnce)` — a zero-value `Config{}`, so `MaxAttempts` always resolves to the default 3 (`applyDefaults`, `:138-158`). There is no plumbing from Layer 2, or any caller, to override the wire-level attempt count.

**Composed ceiling** (what AG-15.2's charter note requires be stated and tested): with harness attempts `H` and the wire-level default of 4 requests per call, the worst-case provider-connection count for one logical harness-level retry sequence is **H × 4**. This number is not documented anywhere in Layer 2 today; AG-15.2 must state and test it.

**Cross-package import constraint (a Go language rule, not a guess):** `backend/agent/src/ai/internal/retry` is an `internal` package rooted at `backend/agent/src/ai`. `backend/agent/src/agent` (Layer 2) is a **sibling** directory, not nested under `backend/agent/src/ai`, so Layer 2 **cannot import** `ai/internal/retry` — Go's internal-visibility rule forbids it at compile time. AG-15's harness-level backoff/clock seam must be independently implemented in the `agent` package, at most mirroring `retry.Config`'s shape as a design precedent, never importing or reusing the Layer 1 package.

## 5. Test substrate for scripting failures and injecting timing

`agenttest.Provider` (`backend/agent/src/agenttest/fake_provider.go`, full file read):

- `NewProvider(scripts ...Script)` (`:64-66`) queues one `Script` per `Stream` call, consumed in order (`consume`, `:138-150`).
- `ErrScriptsExhausted` (`:27-47`): calling `Stream` past the last scripted call fails synchronously and loudly — there is **no built-in "fail forever" mode**. AG-15.2's charter Given ("the fake provider scripted to fail pre-stream forever") must be satisfied by queuing enough failing scripts to exceed the harness's own retry bound; proving the harness's own bound held is then a matter of asserting `provider.Requests()` has length exactly `N`, mirroring `R-ATT-008`'s `S-ATT-010` pattern, not literal infinity.
- `Requests()` (`:157-161`) returns captured requests in call order — the existing mechanism AG-15.2's "the provider call count proves the harness's own bound held" scenario needs. No new capture plumbing required.
- **No clock/timing seam exists in `agenttest` or the `agent` package today.** Every existing AG-13/AG-14 test synchronizes via `agenttest.Gate` and channel reads/closes, per `NFR-RUN-002` and `NFR-CAN-002` (no test may synchronize by sleep or wall clock). AG-15.2's "backoff runs with injected timing" therefore needs **new** plumbing: a `NowFunc`/`SleepFunc`-shaped seam in the `agent` package (mirroring Layer 1's `retry.Config` shape per §4, but freshly implemented — the internal-import boundary forbids reuse) plus a caller-injectable fake clock for tests. This is new surface AG-15.2 must design and build, though the precedent shape is well established.

## 6. The cancellation seam from AG-14 — how backoff must select on it

`Harness.Run` derives one context via `context.WithCancelCause` per run (`harness.go:360-362`), stores the cancel func under `signalMu` (`:337-362`), and both `Interrupt`/`Shutdown` (`:106-127`) invoke it with `ErrInterrupted`/`ErrShutdown` — package-level sentinels matched via `errors.Is` / `context.Cause`. The iteration boundary already re-checks `context.Cause(runCtx)` before every `Turn` call (`:418-420`) and again after a `Turn` error (`:460`).

**Any backoff sleep AG-15.2 adds MUST select on this same `runCtx`** — never a bare `context.Background()` or a timer with no cancellation arm — so `errors.Is(context.Cause(ctx), ErrInterrupted|ErrShutdown)` aborts the wait immediately. This mirrors `defaultSleep` at Layer 1 (`retry.go:192-207`, `select { case <-timer.C: ...; case <-ctx.Done(): return ctx.Err() }`) and `R-CAN-006`'s `Scheduler.WindDownBound` wind-down timer (`scheduler.go:611-619`). An interrupt firing during backoff must route to `windDownRun`, exactly like an interrupt during ordinary iteration — no new cancellation vocabulary, only a new `select` arm inside the new backoff wait.

## 7. Prior art for a "declining v1 implementation" seam

Three established idioms, all directly citable:

**(a) Nil-default function-field seam (AG-08's `PreRequestHook`)** — `TurnOptions.PreRequestHook func(ctx, req) (ai.Request, error)` with `applyPreRequestHook` (`loop.go:655-675`): nil is the identity default (`R-PRH-005`, D4a); a zero-value struct produces byte-identical output to the no-hook case.

**(b) Interface-with-typed-verdict seam (AG-10's `PermissionPolicy`)** — `PermissionPolicy interface { Resolve(ctx, call) PermissionVerdict; Remember(ctx, toolName, outcome) bool }` (`permission_protocol.go:80-94`). Layer 3 implements; Layer 2 must not define rule sets. The interface is the boundary; a concrete `Resolve` returning a fixed verdict is the natural v1 stand-in.

**(c) Zero-default field on a caller-owned struct (AG-14's `Scheduler.WindDownBound`)** — `if bound <= 0 { bound = defaultWindDownBound }` (`scheduler.go:611-613`), the exact shape a backoff-bound override for tests should follow.

Given the charter's language — "the seam is consulted, the v1 implementation declines, and its contract documents what a real implementation must handle" — this reads more like (b): an interface (e.g. `FailoverPolicy`) with a method returning a typed decline/accept verdict, consulted once retries exhaust, whose v1 concrete implementation always declines. This is a genuine open design fork for `sdd-propose`/`sdd-design` to resolve explicitly, citing both (a) and (b), rather than silently picking one.

## 8. Risks and open questions for `sdd-propose` / `sdd-design`

| # | Sev | Risk |
|---|-----|------|
| 1 | **HIGH — CONFIRMED** | Pre-stream turn_end gap (§2). `Turn`'s pre-stream-failure paths (`loop.go:304-338`) never emit `turn_end`, and `CheckStream` hard-rejects a second `turn_start` while a turn is open (`stream_check.go:141-143`). A harness retry over a pre-output failure produces a stream `CheckStream` rejects with `ai.ErrMisplaced`. Must be closed by a companion delta on `agent-loop-skeleton`; `stream_check.go` is pinned byte-unchanged by `R-RUN-003`, so the fix lands on the emitter. |
| 2 | MEDIUM | `agent.Failure` has no `RetryAfter()` accessor (§1). The harness's own decision path bypasses this via a type assertion on the raw error, but the omission is inconsistent with the other four accessors and may need its own spec delta if retry-after should ever be stream-observable. |
| 3 | MEDIUM | No timing/clock seam exists in the `agent` package or `agenttest` (§5). AG-15.2's injected-timing scenario requires new plumbing, not reuse — the Go `internal` import boundary forbids reusing Layer 1's `retry` package (§4). |
| 4 | MEDIUM | `wrapHarnessFailure` (`harness.go:250-259`) discards the real failure category, always reporting `ai.FailureCategoryUnavailable`. AG-15's exhausted-retry terminal report should preserve the true category, which changes this existing helper's behavior and needs an explicit spec decision. |
| 5 | LOW | Composed-ceiling documentation is net-new (§4). No existing doc states "harness attempts × wire attempts"; the wire count (3 retries, 4 requests) is a fixed, non-overridable default at `openaicompat/stream.go:240`. |
| 6 | LOW | Failover-seam shape is an open design fork (§7): nil-default func vs. interface-with-typed-verdict. Charter language leans toward the interface but neither precedent is forced. |

**Exact numbers for citation:** `retry.DefaultMaxAttempts = 3` (`retry.go:18`); zero-value `retry.Config{}` call site (`openaicompat/stream.go:240`); `Scheduler.WindDownBound` default arm (`scheduler.go:611-613`); 25 registered `EventKind`s (`event.go:100-221`); `Harness.Run`'s failure branch (`harness.go:450-470`); `CheckStream`'s second-turn rejection (`stream_check.go:141-143`).

## Ready for proposal

Yes, with the six risks carried forward explicitly. Risk 1 is the one `sdd-design` must resolve first: it determines whether AG-15.1's first scenario is satisfiable at all without a companion delta on `agent-loop-skeleton`.
