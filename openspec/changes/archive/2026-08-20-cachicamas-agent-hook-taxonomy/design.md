# Design: AG-20 — Complete the hook taxonomy

> Change `cachicamas-agent-hook-taxonomy` · builds on `proposal.md` (D1–D7 settled; this design lands the shapes they deferred and overturns **two** sub-details on stated evidence: D3's reporter-invocation site, AD-4, and D4's gate-release ordering, AD-9/T2 — both declared at their decision, neither silent). Every `file:line` below was opened and verified in this worktree at `main@2a138b59`.

**Deviation from the 800-word budget**: the orchestrator's launch contract explicitly requires the D4 snapshot mechanics, the D6 idempotence proof, the full D7 firing table, the bite-(b) shape, and the by-name file layout; those override the size cap for this change (the AG-19 precedent).

## Technical Approach

One new production file, `hooks.go`, carries the whole taxonomy: the `Hooks` value type, the two function-type families (mutating with returns, observing without), the three payload value types, the typed `ObserverStall` report, and the per-run observer lane (unbounded FIFO, one drain goroutine, registration-order dispatch, panic recovery, terminal-boundary snapshot). `loop.go` widens `applyPreRequestHook` into chain composition; `harness.go` gains the `Hooks` field, the `sessionStarted` latch, the transport assignment beside `turnOpts.Continuation` (`harness.go:640-646`), four post-turn fire sites, and the lane lifecycle; `compaction.go` gains the pre-compact splice between `markSpan` (`compaction.go:285`) and `buildLoopRequest` (`:292`), landing failures on the existing `emitCompactionFailedArm` (`:126`). No new `EventKind` (guard at 25, `scope_fence_test.go:87`), no `Harness` method (`scope_fence_test.go:102-105`, `harness_test.go:1031` both pass unedited), no substrate release requested (every splice file — `loop.go`, `harness.go`, `compaction.go` — is already in both filters; the AG-17/AG-19 template).

## Architecture Decisions

### AD-1 — Registration surface and transport (D1 confirmed; mechanics decided)

**Choice**: `Harness.Hooks Hooks` (field, zero-value inert) is the one registration surface; `TurnOptions.Hooks Hooks` is the transport `Run` fills on its per-turn copy, beside `Continuation`. Because `Hooks` holds func fields, it is **not comparable** — `h.Turn.Hooks != (Hooks{})` does not compile. An unexported `func (h Hooks) isZero() bool` (all four slices `len == 0` and `StallReporter == nil`) decides both refusal sites:

- `Run` entry: non-zero `h.Turn.Hooks` → typed refusal `ai.Invalid(ai.ErrMisplaced, ai.At("turn.hooks"))`, checked in the same pre-identity block as the shutdown check (`harness.go:411-436`), before any event — the `validateContinuation`/`S-LSK-014` posture.
- `runCompaction`: `req.Options.Hooks` joins `Tools`/`Continuation` in the misplaced-options rejection (`compaction.go:269-272`). The pre-compact chain reaches `runCompaction` as one new unexported parameter from the harness.

A direct `Turn` caller who sets observer families on `TurnOptions.Hooks` gets inertness, not refusal: `Turn` consumes only the `PreRequest` family (it has no lane). Stated for `sdd-spec`, not left to inference.

**Rejected**: comparability via pointer-to-Hooks (breaks the zero-value house pattern); a `RegisterHook` method (unavailable by fence).

### AD-2 — The two families: exact declarations, wrong thing unconstructible

```go
// hooks.go — the exported surface, in full.
type Hooks struct {
    PreRequest    []PreRequestHook       // mutating, chained; element i's output feeds i+1
    PreCompact    []PreCompactHook       // mutating, chained
    PostTurn      []PostTurnObserver     // observing
    SessionStart  []SessionStartObserver // observing
    StallReporter ObserverStallReporter  // nil-default: reports nothing (Failover posture)
}

type PreRequestHook func(context.Context, ai.Request) (ai.Request, error)
type PreCompactHook func(context.Context, CompactionPlan) (CompactionPlan, error)
type PostTurnObserver func(context.Context, PostTurnReport)
type SessionStartObserver func(context.Context, SessionStartReport)
type ObserverStallReporter func(ObserverStall)
```

Unconstructibility, three times over: (1) an observer type has **no results** — `func(...) error` or `func(...) PostTurnReport` fails to compile when placed in `PostTurn`; there is no channel through which an observer can signal a mutation or a failure. (2) Every payload (`CompactionPlan`, `PostTurnReport`, `SessionStartReport`, `ObserverStall`) is a value with unexported fields and read-only accessors (the `TurnEnd`/`CostFigures` shape), so an observer cannot reach shared state through its argument. (3) The invocation context is **value-stripped** (AD-3): `DelegationSeamFrom` (`delegation_seam.go:101-104`) is a plain context-value lookup, and `DelegationSeam.Publish` is the one sanctioned door back onto a parent's event lane — in exactly the delegated-child scenario AD-6 blesses, the child's `runCtx` inherits the hosting tool call's seam, so a value-preserving observer ctx would hand an observing hook that door. Observers get a fresh `context.Background()`-rooted context, so the seam (and every other ctx-carried capability) is unreachable by construction, not by convention. `CompactionPlan` alone carries a derivation method, `WithCut(int) CompactionPlan` (the `ai.Request.With` posture): a pre-compact hook can re-designate the cut but cannot forge a span — the span is derived, never settable.

Attribution vocabulary (resolves the D1 "element zero" vs success-criteria "feeds element 0" ambiguity): failures are attributed by **source name**, `TurnOptions.PreRequestHook` (the kept singular field, which runs first) or `Hooks.PreRequest[i]` / `Hooks.PreCompact[i]` by index. Unambiguous under later insertion; no ordinal renumbering ever lies.

### AD-3 — The observer lane

**Choice**: `Run` creates one lane iff `len(PostTurn)+len(SessionStart) > 0`, after `runID` is minted (attribution needs it). The lane is a mutex-guarded slice queue plus one drain goroutine; **enqueue is a lock-append and never blocks** — that, not scheduling luck, is the structural non-blocking property. The drain goroutine pops FIFO, records the in-flight invocation under the mutex, invokes the observer, clears in-flight. Registration order = enqueue order = dispatch order at every point (serial lane; no fan-out — `R-AGE-009`'s consumer machinery stays out of scope). Observers are invoked with a fresh `context.Background()`-rooted context — **never** `runCtx` and deliberately **not** `context.WithoutCancel(runCtx)`, because `WithoutCancel` preserves context **values**: in a hosted child run the child's `runCtx` inherits the hosting tool call's delegation seam (installed by `executeCall` via `withDelegationSeam`; retrievable by the plain value lookup `DelegationSeamFrom`, `delegation_seam.go:101-104`), and a value-preserving observer ctx would let an observing hook `Publish` onto the **parent's streaming lane**, asynchronously, after `Run` returned — an observer mutating a stream, the exact thing AG-20 exists to forbid. Value-stripping keeps AD-2's "unconstructible" literally true and costs nothing: observers may run after `Run` returned, so the run's cancellation and values would misreport anyway. Mutating hooks keep the live ctx exactly as AG-08 shipped. The lane closes for enqueue at the terminal snapshot (structurally no fire site follows it) and its goroutine exits when the queue drains — a released observer leaves no steady-state goroutine.

Zero-value inertness (`R-HKS-012`): no observers → no lane, no goroutine, no channel — the nil path is byte-for-byte today's path on every arm.

### AD-4 — "Eventually": the terminal-boundary snapshot (D4 mechanics; one D3 sub-detail overturned)

**Choice**: a `defer lane.reportOutstanding()` registered in `Run` immediately after lane creation. LIFO places it **first** among Run's defers — after every `return`'s `run_end` has been sent (all three arms send it inside the returning call: success `harness.go:776-779`, `windDownRun` `:346-348`, `failRun` `:369-371`), and before `queue.close`, the cancel-clear, and `close(sink)`. What is snapshotted, under the lane mutex: the in-flight invocation (dispatched, unreturned) plus every queued invocation. Each becomes one `ObserverStall{point, index, run, reason}` with a **three-valued** discriminator:

| Reason | Meaning | Delivered when/where |
|---|---|---|
| `StallOutstanding` | dispatched, unreturned at the terminal boundary | snapshot, **Run's goroutine** |
| `StallQueued` | never dispatched, still queued behind a stall | snapshot, Run's goroutine |
| `StallPanicked` | observer panicked | at recovery, **lane goroutine** (AD-5) |

The third value is a refinement of D3's two (queued victims are not stalled culprits; collapsing them misattributes). **The overturn, on stated evidence**: D3 said the reporter is "invoked on the observer lane". For the snapshot half that is undeliverable — in the motivating case the lane's single drain goroutine is blocked **inside the stalled observer**, so a lane-queued report can never run. Snapshot reports therefore run synchronously on Run's goroutine at the terminal boundary, strictly after the terminal `run_end` was sent. `R-AGE-008`/`S-AGE-010` is preserved for **event delivery**: no event follows `run_end`, so a post-terminal reporter can impede no delivery and has no path onto the stream. The channel-close half is stated too, not implied: because the snapshot defer precedes `close(sink)` (`harness.go:446`) in LIFO order, a stalling reporter delays both `Run`'s return **and** the sink close — a consumer ranging the sink has already received `run_end` but never observes termination. This also gives the determinism the test needs: reporter invocation happens-before `Run` returns. Panic reports stay on the lane (there the drain goroutine is free — the panicking hook returned by panicking). All reporter invocations are serialized by one dedicated mutex. No wall clock, no timeout, no join, anywhere (`R-RUN-010` followed; `NFR-{CST,CTX,CMP,RUN,CAN,RTY}-002` untouched).

Consequences carried as spec text: a permanently stalled observer leaks a goroutine **by design** (caller's leak; AG-21 inherits knowingly); an observer finishing just after the snapshot is reported once and completes anyway; a stalling **reporter** stalls both `Run`'s return and the sink close (the two observables above) — caller's code, caller's stall, same posture on both.

**Determinism lemma (named, reused by the tests): snapshot-or-count exhaustiveness.** An enqueued invocation is, at the snapshot, either complete (its side effect is visible to any post-`Run` read — increment → in-flight clear under mu → snapshot read → Run return → test read is a happens-before chain) or outstanding/queued (it is in the report). There is no third state, so "this hook did not fire" is deterministically assertable as *side-effect count unchanged AND report empty* — no sleep, no poll.

### AD-5 — Panic posture (the charter never named it; decided)

Observing hooks: the drain goroutine wraps every invocation in `defer recover()`; a recovered panic is reported as `StallPanicked` (point, index, run) and the lane continues with the next queued invocation — `runToolWithWindDown`'s posture one layer out. Without this, "observers never block the streaming path" would be true in the worst way: an unrecovered panic on the lane kills the process. Reporter invocations (both goroutines) are also recover-wrapped and a reporter's own panic is **discarded** — the reporter is the last resort; it has no meta-reporter; process survival wins. Mutating hooks are **not** recovered: inline on the caller's goroutine exactly as AG-08 shipped (out of scope, stated).

### AD-6 — Session-start (D2 confirmed; placement decided)

`sessionStarted bool` sits beside `shutdown` (`harness.go:126-131`), guarded by the existing `signalMu`. In Run's entry critical section (`:411-436`), **after** the shutdown check (a shut-down value fires nothing) and before the cancel derivation: if unset, set it and record fire-this-run in a local. The enqueue site is pinned **immediately after lane creation and before the `NewRunStart` construction at `:480`** — not merely "before the `sendStamped` at `:484`": the construction-error return at `:481-483` sits between those two, and an enqueue placed after it would let a run that fails `NewRunStart` **consume the latch without ever firing** — silencing session-start for the value's entire remaining lifetime. Pinned before `:480`, the degenerate error run still fires (fired-and-consumed is consistent; consumed-without-firing is the bug). Session-start is invocation #1 on the lane, ordered before any post-turn of the run. Payload `SessionStartReport{run}` carries the firing run's id. `Harness` already embeds mutexes, so `go vet`'s copylocks makes a post-use value copy a build-time finding — the latch cannot be laundered by copying. Serial reuse: one fire across N runs; a delegated child is a distinct `Harness` value (`nested_run_test.go:27`, the child struct literal handed to `delegatingTool`) and fires its own.

### AD-7 — Post-turn: fourteen exits, four call sites (D7 completed)

Two pure reads join the existing per-attempt forwarder (`harness.go:616-638`), beside `capturedTurnID` and `total.add`: a **per-logical-turn** `turnCost costAccumulator` (a local of the outer loop iteration — never a `Harness` field, `R-CST-004`) and `capturedOutcome`, read from the forwarded `turn_end` payload (no new `TurnOutcome`). Every fire site is downstream of `<-forwarderDone` (`:649`), so all reads inherit the close-of-channel happens-before that `total`'s own comment states. Payload: `PostTurnReport{run, turn, outcome, cost, attempts}`; cost is the **sum over the logical turn's attempts** (each completed attempt emitted its own `cost_turn`; aborted attempts contributed none).

The complete exit enumeration — every row is normative for `sdd-spec`; the repo's bugs hide between two satisfied requirements:

| # | Exit from the logical-turn loop | Site (`harness.go`) | Fires? | Outcome in payload |
|---|---|---|---|---|
| 1 | success, before `closeTurnMarked` | after `:649`/loop break, pre-`:733` | **yes** | forwarded (Finished) |
| 2 | `closeTurnMarked` error → `failRun` | `:734` | **yes** (already fired at row 1's site — the fire precedes the close) | Finished |
| 3 | turn failed (G1–G3/G5) → `failRun` | `:723` | **yes** | Aborted (forwarded) |
| 4 | mid-turn signal (G0) → `windDownRun` | `:667-669` | **yes** | Aborted |
| 5 | backoff-sleep signal → `windDownRun` | `:693-695` | **yes** | Aborted |
| 6 | backoff bare-cancel → break → `failRun` | `:696` → `:723` | **yes** (row 3's site) | Aborted |
| 7 | iteration-boundary signal, no turn ran | `:502-504` | no | — |
| 8 | post-compaction signal, no turn ran | `:565-567` | no | — |
| 9 | steering-drain append error | `:507-509` | no | — |
| 10 | prompt append error, pre-loop | `:486-488` | no | — |
| 11 | terminal-decision steer append error | `:752-754` | no **extra** (row 1 already fired) | — |
| 12 | `NewRunStart` construction error | `:481-483` | no | — |
| 13 | shutdown refusal at entry | `:420-423` | no (latch untouched — checked after shutdown) | — |
| 14 | compaction bracket | `compaction.go` | no — pre-compact is its hook | — |

Fourteen exits collapse to **four** enqueue call sites: (i) after the attempt loop with `terr == nil`, **before** `closeTurnMarked` (rows 1, 2, 11); (ii) before `failRun` at `:723` (rows 3, 6); (iii) before `windDownRun` at `:667` (row 4); (iv) before `windDownRun` at `:693` (row 5). Four sites, each one line, each preceding the exit call — no exit path can fire twice and no firing row can be skipped by an early return above it.

The other three points across the same exit families — no cell implied:

| Exit family | SessionStart | PreRequest (chain) | PreCompact (chain) |
|---|---|---|---|
| Shutdown refusal at entry (`:420-423`) | no — latch checked after the shutdown check, untouched | no | no |
| `NewRunStart` error (`:481-483`) | **yes** — enqueued pre-`:480` (AD-6); the degenerate run fires-and-consumes | no | no |
| Normal completion / turn failure / retry exhaustion | once per `Harness` value, on its first non-refused run | once per **attempt** that reaches the request build, between `buildLoopRequest` and `provider.Stream` — including attempts that then fail or are retried | only inside a compaction bracket with a non-empty chain, pre-provider |
| Interrupt / shutdown (mid-turn, backoff, or boundary) | same — already enqueued at entry; the signal cannot un-fire it | only for attempts that reached the build before the signal; none for the never-started attempt | no new firing; an in-flight compaction's chain already fired pre-provider |
| Compaction bracket (either door, success or failure) | — (not a run entry) | **no** — the compaction provider call is not a `Turn` and never passes the pre-request seam (`runCompaction` calls `buildLoopRequest` `:292` → `Provider.Stream` `:297` directly); PreCompact is this bracket's hook, stated so nobody infers a second pre-request seam | **yes** — before the provider call, both doors, AD-8 |

### AD-8 — Pre-compact splice and the `resolveCut` idempotence proof (D6 obligations discharged)

**Splice**: between `markSpan` (`compaction.go:285-289`) and the prefix build (`:291`), guarded by `len(chain) > 0` (inertness). The hook receives `CompactionPlan{cut, CompactionSpan{start, end}}`. Chain failure → `emitCompactionFailedArm(..., TurnOutcomeAborted, err)` — before the provider call, the "no completion" arm, so `R-CMP-003`'s iff and `R-CMP-010`/`R-CMP-013` hold unamended. The returned plan's cut is then **unconditionally** re-resolved through `resolveCut` (`:51-79`) and the span re-derived through `markSpan` (a pure lookup pre-commit); a re-resolution to 0 fails through a **new** `emitCompactionFailedArm(..., TurnOutcomeAborted, ai.Invalid(ai.ErrEmpty, ai.At("cut")))` call at the splice — the upstream empty-cut arm at `:275-278` is **not reachable** from the post-splice position; the arm's shape is reused, the call site is new.

**Idempotence, proven rather than assumed** — for unchanged plans the second resolution must be a no-op. Structural argument, two lemmas over an unmutated `hist` (nothing between `:274` and the commit at `:331` writes it, and the hook holds a value type that cannot reach it): (L1) `markBoundaryAtOrBefore` is idempotent on its own image — it returns the nearest mark boundary at-or-before its input, and a mark boundary maps to itself; (L2) the open-pair scan is a pure function of `(messages, cut)`. Any non-zero `c = resolveCut(hist, n)` therefore satisfies `markBoundaryAtOrBefore(c) == c` and `!open(messages[:c])`, so `resolveCut(hist, c)` returns `c` on its first pass. The proof is asserted **directly**: `compaction_surgery_test.go` is `package agent` and already calls `resolveCut` first-hand under `NFR-CMP-001`'s carve-out ("Pure helpers with no external surface MAY be tested internally", quoted in that file's own header, `compaction_surgery_test.go:1-19`); it gains a fixed-point table — `resolveCut(hist, resolveCut(hist, n)) == resolveCut(hist, n)` over the same class of naive cuts (0, on-mark, mid-mark, straddling an open pair, beyond length). The carve-out's own condition ("every claim about what a caller observes is also asserted externally") keeps the observational half in force: the identical-plan scenario (a hook returning its input unchanged) must produce a stream and committed history byte-identical to the hookless run — if the second resolution moved the cut, byte-identity breaks. The forward-adjustment scenario exercises the other direction: the hook re-designates the cut forward; resolution retracts from the new request exactly as before; the committed prefix is pairing-closed. `R-CMP-004`'s "MUST NOT expand the cut forward" (`agent-compaction/spec.md:115`) is disambiguated in the delta as governing **resolution**, never the **request** — stated, so a reviewer reading a forward-adjusting hook against the shipped MUST finds the sentence scoped, not violated.

### AD-9 — Bite (b): the literal sabotage cannot fail as an assertion; the recorded shape is goroutine placement

**Impossibility, stated with its proof sketch** (this is a finding, not a shrug): under the "dispatch synchronously" sabotage, the observer's return is ordered happens-before every post-fire-point observable — subsequent sends, the terminal snapshot, and `Run`'s return. A test witnessing asynchrony must observe a post-fire-point observable *while a blocked observer is unreturned*; under the sabotage no such observable ever occurs, so every faithful wait is unbounded. The only bounded conversion of "never" is a clock, and six shipped NFRs ban it. A gate-holding bite under this sabotage therefore deadlocks by construction — and a deadlocking bite cannot have its RED recorded.

**The recorded shape instead**: the asynchrony property *is* a goroutine-placement property, so the bite asserts placement directly. A **non-blocking** post-turn observer captures its own stack (`runtime.Stack(buf, false)`); after `Run` returns, the test asserts the recorded stack contains neither `(*Harness).Run` nor the forwarder's frame — the lane's drain root only. Under the shipped lane this passes deterministically; under the synchronous sabotage the observer runs on Run's goroutine and the assertion **fails as an assertion**, with the offending stack in the failure message. Deterministic both ways, no clock, no gate, no hang. (`runtime.NumGoroutine` sampling is already house practice; stack self-inspection is the same runtime-introspection grade as the shipped reflection pins.)

The liveness half keeps its own GREEN test (T2 below): gated observer, **buffered sink sized to the script's full event count** so `Run` needs no live consumer, reporter **records only** — it never touches the gate. Deterministic, and in the proposal's own order (D4 bullet 4: "the run returns before the gate is released"): the snapshot fires the reporter on Run's goroutine, `Run` returns **with the gate still held** (the snapshot never joins), the test observes `<-runDone`, asserts one `StallOutstanding` report, stream byte-identity, and `CheckStream` validity, and **only then** releases the gate — with `t.Cleanup(gate.Release)` as the AD-10 backstop. Its vacuity guard is bite (a) (no stall ⇒ empty report) plus the gate-entered flag.

### AD-10 — Goroutine-leak discipline: release-before-baseline

Named rule for this milestone's own suite: **a test that gates an observer asserts no goroutine baseline, period**; `runtime.NumGoroutine()` sampling lives only in ungated tests. Every gate is released — inline once its assertions no longer need the hold, and unconditionally via `t.Cleanup(gate.Release)` registered at construction — so no gate outlives its test and the lane goroutine drains and exits before the next test's baseline. The permanently-stalled leak is exercised only under a cleanup-released gate. AG-21 inherits the rule by name.

## Data Flow

    Harness.Hooks ──(Run: turnOpts.Hooks = h.Hooks, beside Continuation)──▶ Turn
        │                                   └─ composePreRequest: singular field → PreRequest[0..n] → provider.Stream
        │
        ├─(unexported param)──▶ runCompaction: resolveCut → markSpan → [PreCompact chain → resolveCut → markSpan] → provider.Stream
        │                                              failure ──▶ emitCompactionFailedArm (existing arm)
        │
        └─▶ observer lane (lock-append enqueue, never blocks) ──drain goroutine──▶ SessionStart / PostTurn observers
                    │                                                └─ panic ⇒ recover ⇒ reporter(StallPanicked)  [lane goroutine]
                    └─ terminal boundary (defer, after run_end) ⇒ snapshot ⇒ reporter(StallOutstanding/Queued)  [Run's goroutine]

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/hooks.go` | Create | `Hooks`, 5 hook/reporter types, 3 payloads + `ObserverStall`/`HookPoint`/`StallReason`, lane, snapshot (~260-380) |
| `backend/agent/src/agent/loop.go` | Modify | `TurnOptions.Hooks` field; `applyPreRequestHook` → chain composition with source-name attribution |
| `backend/agent/src/agent/harness.go` | Modify | `Hooks` field; `sessionStarted` latch; typed refusal; transport assignment; `turnCost`/`capturedOutcome` reads; 4 fire sites; lane lifecycle + snapshot defer |
| `backend/agent/src/agent/compaction.go` | Modify | Splice + re-resolution; `Hooks` in misplaced-options; one unexported parameter |
| `backend/agent/src/agent/hooks_test.go` | Create | Lane units, ordering, panic, snapshot, T2 liveness, bites (a) (b) (c) |
| `backend/agent/src/agent/hooks_harness_test.go` | Create | Session-start (reuse, child, shutdown), post-turn firing table, bite (e) |
| `backend/agent/src/agent/hooks_compaction_test.go` | Create | Splice on both doors, identical-plan byte-identity, forward adjustment, bite (d) |
| `backend/agent/src/agent/loop_test.go` | Modify | **Filter entries only** — `filterOutLoopFiles` (`:837`) gains the four new exact-filename suffixes. No assertion changes. |
| `backend/agent/src/agent/loop_hook_test.go` | Modify | **Filter entries only** — `filterOutLoopHookFiles` (`:907`) gains the identical four suffixes, byte-in-sync. AG-08's assertions are untouched. |
| `failure.go`, `event.go`, `event_descriptor.go`, `agenttest/**`, `scope_fence_test.go`, every other existing `*_test.go`, `go.mod`/`go.sum`, `src/ai/**` | NOT TOUCHED | The stall types live in `hooks.go`, **not** `failure.go` — editing `failure.go` would pass the mechanical guard silently while violating R-LSK prose; declined explicitly |

**Substrate filters** (`filterOutLoopFiles` in `loop_test.go`, `filterOutLoopHookFiles` in `loop_hook_test.go`): AG-20 requests **no release** — every edited file is already in both. Editing these two files is *mandatory*, not optional: a new `hooks.go` under `backend/agent/src/agent/` would otherwise appear unstripped in `TestTurn_SubstrateUntouched`'s diff (`loop_test.go:1329-1335`). They are therefore listed as Modify above; the edit is confined to the filter lists and changes no assertion. It adds, **by exact filename and never by count, the identical set to both filters**: `hooks.go`, `hooks_test.go`, `hooks_harness_test.go`, `hooks_compaction_test.go`. No `agenttest` entry (the filters do not govern that directory; `Gate` is used, not widened).

## Interfaces / Contracts

The whole exported surface is AD-2's block plus:

```go
type CompactionPlan struct{ /* unexported */ }   // Cut() int · Span() CompactionSpan · WithCut(int) CompactionPlan
type PostTurnReport struct{ /* unexported */ }   // Run() RunID · Turn() TurnID · Outcome() TurnOutcome · Cost() CostFigures · Attempts() int
type SessionStartReport struct{ /* unexported */ } // Run() RunID
type ObserverStall struct{ /* unexported */ }    // Point() HookPoint · Index() int · Run() RunID · Reason() StallReason
type HookPoint uint8   // HookPointSessionStart, HookPointPostTurn
type StallReason uint8 // StallOutstanding, StallQueued, StallPanicked
```

`CostFigures` is carried unchanged — `cost_events_test.go:199-231` pins it by reflection and stays green unedited. Export justification: every type above is consumed from `package agent_test` and later by Layer 3 registrations; nothing else is exported and `Harness` gains no method.

## Testing Strategy

Runner: `cd backend/agent && go test -race -count=1 ./...` — a cached run is not evidence; record wall-clock (~170s uncached).

| Test | Proves | RED staging |
|---|---|---|
| T1 lane units: ordering at all four points, N observers registration-order | R-HKS-006 | bite (c): reverse dispatch ⇒ order assertion fails |
| T2 liveness (AD-9): gated observer, buffered sink, reporter releases | R-HKS-007/008, invariant 3 | bite (a): observer returns immediately ⇒ report set MUST be empty (anti-vacuity) |
| T3 placement (AD-9): stack self-inspection | R-HKS-007 | **bite (b)**: synchronous dispatch ⇒ fails as an assertion, stack in message |
| T4 session-start: two serial Runs = one fire; child fires own; shutdown fires none | R-HKS-005 | bite (e): fire per Run ⇒ count==2 **or** run-2 report non-empty (snapshot-or-count lemma — deterministic either way) |
| T5 post-turn: every yes-row of AD-7's table, multi-attempt cost sum, no-fire rows via count+empty-report | R-HKS-004 | wire only the success site first ⇒ rows 3-6 RED |
| T6 pre-compact: both doors, identical-plan byte-identity (idempotence), forward adjustment pairing-closed, hook failure on existing arm, run continues | R-HKS-003, R-CMP-004/010/013 | bite (d): skip re-resolution ⇒ committed prefix splits a pair |
| T7 chain: AG-08's assertions **unchanged** and green (its file gains filter entries only); zero-value byte-identity; singular-first; element-attributed abort | R-HKS-002, R-PRH-002/003/005 as amended | element failure without attribution ⇒ attribution assertion RED |
| T8 panic: panicking observer reported `StallPanicked`, lane continues, process survives, stream byte-identical | AD-5 | recovery removed ⇒ crash; record from isolated `go test -run` (AG-19 bite-(c) precedent) |
| T9 inertness + fences: zero-value `Hooks` byte-identical on every path; `EventKinds()==25`, `NumMethod()==5`, `scope_fence_test.go` byte-unchanged; `loop_test.go`/`loop_hook_test.go` differ only by the four filter entries and stay byte-in-sync | R-HKS-012, R-HKS-010/011 | new-surface RED per house overlay protocol |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Hooks are in-process function values invoked over in-process value types.

## Migration / Rollout

No migration. Rollback is the proposal's single revert: delete `hooks.go` and the three test files, drop the fields/latch/fire-sites/splice, drop the four filter entries.

## Open Questions

None blocking. Downstream notes for `sdd-spec`/`sdd-tasks`: (1) the third `StallReason` value and the reporter's two invocation sites (AD-4) amend D3's wording — carry into R-HKS-008; (2) bite (b)'s redefined shape (AD-9) replaces the proposal's literal sabotage — carry into the task's RED script; (3) attribution names sources (`TurnOptions.PreRequestHook` / `Hooks.PreRequest[i]`), settling the D1 "element zero" phrasing — carry into R-PRH-003's delta; (4) direct-`Turn` observer inertness (AD-1) needs one sentence in the new capability spec.
