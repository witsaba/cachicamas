# Proposal: AG-15 — Implement retry policy and the failover seam

> **Change**: `cachicamas-agent-retry-failover` · **Milestone**: AG-15 (Layer 2 Wave 3, milestone 15 of 24; doc 0003 lines 1444–1525)
> **Branch**: `feat/agent-layer2-wave3-ag15` · base `main@bf482b0a` · **Worktree**: `cachicamas-worktrees/ag-15-retry-failover`
> **Artifact store**: hybrid (Engram + filesystem) · **Delivery**: single PR, `size:exception` pre-authorized against the 1000-line budget
> **TDD**: strict, RED-first (`cd backend/agent && make test`)
> **Closes**: **G8**'s Layer 2 half (R-15); seam 7 consumed, seam 8 reserved
> **Depends on**: AG-11, AG-13 (both archived) · **Parallel with**: AG-14 (merged `6485b937`), AG-16
> **Exploration**: `explore.md` · Engram `sdd/cachicamas-agent-retry-failover/explore`

---

## Intent

Layer 1 classifies. Layer 2 decides. Today Layer 2 decides nothing: every non-cancellation `Turn` error routes unconditionally to `h.failRun(sink, stamper, runID, terr)` (`harness.go:469`), and `R-RUN-011` states the prohibition as a requirement — "MUST NOT retry, back off, or route to a fallback provider. Retry and failover are **AG-15**'s" (`agent-run-driver/spec.md:241`).

Three concrete consequences follow, all verified in the shipped tree:

1. **A transient failure kills the run.** `ai.FailureReport.Retryable` is a first-class classification (`provider_failure.go:282-285`) that Layer 1 populates and Layer 2 reads through `(*ai.Failure).Retryable()` (`:456-461`) — and no production code in `backend/agent/src/agent` acts on it. A rate-limit hiccup before a single token was emitted ends the run exactly as a malformed-response does.
2. **The run's terminal report lies about why it failed.** `wrapHarnessFailure` hardcodes `Category: ai.FailureCategoryUnavailable` (`harness.go:250-254`) for *every* cause. The loop preserves the true category on `turn_end` (it wraps the raw `*ai.Failure` directly, `loop.go:387-388`; `turn_failure_test.go:344` pins `FailureCategoryRateLimit` surviving); the harness then discards it one event later on `run_end`. That asymmetry is invisible today because nothing reads it — an exhausted-retry report makes it load-bearing.
3. **A retry is not even expressible on the stream.** Confirmed, not conjectured — see Decision 1.

**What ticks**: `R-RUN-011`'s no-retry prohibition converts from a fence into a policy; `agent-turn-termination/spec.md:153` ("Acting on retryability — AG-15") and `agent-run-driver/spec.md:308` both close; `R-AIS-044 / S-2` — the already-promoted Layer 2 half of the composed-bound contract (`ai-stream-lifecycle/spec.md:646-650`) — gets its first implementation.

---

## Scope

### In

- **AG-15.1** — a **turn-level retry predicate** in `Harness.Run`, inserted between the AG-14 cancellation carve-out (`harness.go:460-462`) and the unconditional `failRun` (`:469`). Decides from typed evidence only: category, retryability, partial-output, delivery. See Decision 2's table.
- **AG-15.1** — **each attempt visible on the stream**: a retry re-invokes `Turn` over an unmodified transcript, producing its own `turn_start`/`turn_end` pair on the same contiguous lane (`R-RUN-003`), distinguishable by sequence number and turn identity. **No new `EventKind`** — the `R-RUN-012` / `R-LSK-004` posture that AG-13 and AG-14 both held.
- **AG-15.1 (mandatory companion)** — `Turn`'s three pre-stream failure paths emit `turn_end(TurnOutcomeAborted, failure)` before closing the sink. Without this, AG-15.1's first scenario is unsatisfiable. See Decision 1.
- **AG-15.2** — **bounded backoff** with exponential growth and jitter, a documented harness attempt bound, and a **clock/sleep injection seam** in the `agent` package so every test synchronizes deterministically (`NFR-RUN-002`, `NFR-CAN-002`: no test may synchronize by sleep or wall clock).
- **AG-15.2** — **retry-after overrides computed backoff** when the failure reports one, read presence-typed via `(*ai.Failure).RetryAfter()` (`provider_failure.go:471-476`).
- **AG-15.2** — **the backoff wait selects on the run context**, so `Interrupt`/`Shutdown` abort it immediately and route to `windDownRun`, exactly as an interrupt at an iteration boundary does.
- **AG-15.2** — **the composed ceiling** `H × 4` stated and tested. See "The composed ceiling".
- **AG-15.3** — the **failover seam**: a named injection point, consulted once retries exhaust, whose v1 implementation declines and whose contract documents what a real implementation must handle (re-counting the context budget, restarting the cache prefix). See Decision 3.
- **AG-15.3** — the **inertness pin**: identical observable behavior with and without the seam installed.
- **AG-15** — the exhausted-retry terminal report carries the **true** failure category. See Decision 4.
- Substrate-guard filter widening in `loop_test.go` (`:831-871`) and `loop_hook_test.go` (`:907-943`) by **exact filename suffix**, both filters byte-in-sync — the AG-11/AG-13/AG-14 discipline (`agent-loop-skeleton/spec.md:92-96`).

### Out — deferred, with the owner named

| Deferred | Owner and why deferral is safe |
|---|---|
| **Wire-level retry mechanics** (attempt scheduling below `provider.Stream`) | **Layer 1's AI-35**, shipped. `ai/internal/retry` exists and runs today at `openaicompat/stream.go:240`. Charter "Out of scope" line (`0003:1454`), binding. AG-15 consumes its documented multiplier; it does not re-implement, re-plumb, or override it. |
| **Model failover implementation** (choosing a fallback model, re-budgeting tokens and prices, restarting the cache prefix) | **Post-v1**, seam 8's rationale (`0003:1454`): it crosses into AG-17/Layer 3 territory. `agent-v1-scope/spec.md:128,133` already records this as `AGS-D` with **AG-15.3 named as its placeholder node**. AG-15 ships the seam and its documented contract, nothing behind it. |
| Retry of a **cancelled** turn | **Never.** `R-RUN-011`'s carve-out extends the no-retry rule to cancellations *verbatim* (`agent-run-driver/spec.md:248`; `agent-cancellation-tree/spec.md:168`). The existing carve-out at `harness.go:460-462` stays **ordered ahead** of the retry decision. |
| Retry **after** emitted output | **Never.** The G8 sentence, now at the harness. AG-15.1 scenario 2 forbids it; `V-FAIL-15` forbids the Layer 1 counterpart for the same reason (duplicate observable output). |
| Overriding Layer 1's wire attempt count from Layer 2 | **Not this milestone.** The call site is a zero-value `retry.Config{}` (`openaicompat/stream.go:240`) with no plumbing from any caller; adding one is a Layer 1 edit, forbidden by `R-RUN-012`. AG-15 states the ceiling as a *fact about the default*, not as a knob. |
| Retry-after on the **stream-observable** envelope (`agent.Failure.RetryAfter()`) | **Deferred — see Decision 5.** The harness's own decision path does not need it. |
| Cost accounting for retried attempts | **AG-16**, parallel. |
| Subagent-scoped retry | **AG-19.** No subagent tool ships in v1 (`0003:1794`). |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2.** Layer 1 is consumed, never edited (`R-RUN-012`). Includes `ai/internal/retry/doc.go`, which AG-15 **reads** to satisfy `R-AIS-044 / S-2`. |

---

## Decision 1 — the pre-stream `turn_end` gap **(DECIDED — mandatory companion delta)**

**Confirmed by direct reading, not inferred.** `Turn` emits `turn_start` unconditionally at `loop.go:296-301`. All three pre-stream failure paths then call `closeSink(sink)` and return the bare error with **no `turn_end` emitted**:

| Path | Site | Returned error |
|---|---|---|
| `buildLoopRequest` error | `loop.go:304-308` | plain Go error |
| `PreRequestHook` error | `loop.go:317-328` | `*ai.Failure` (pre-stream, `UnsupportedCapability`, `Retryable` unset → `false`) |
| `provider.Stream` error | `loop.go:332-338` | whatever the provider returned (comment names it "V-PRV-04", the AG-07 walking-skeleton behavior) |

The harness's own comment already concedes the shape: "the turn's own typed closing brackets (`turn_end(Aborted)`, **or nothing at all on a pre-stream failure**)" (`harness.go:465-467`).

`CheckStream` hard-rejects the consequence. On `BracketRoleOpensTurn`: `if turnOpen { return violation(ai.ErrMisplaced, ai.AtIndex("event", i)) }` (`stream_check.go:141-143`). A harness retry re-invoking `Turn` after a pre-stream failure emits a second `turn_start` while `turnOpen` is still true → the recorded stream is rejected with `ai.ErrMisplaced`. **This is a hard violation, not a tolerance question.**

**Decision: `Turn`'s three pre-stream paths MUST emit `turn_end(TurnOutcomeAborted, failure)` before `closeSink`, mirroring the mid-stream path (`loop.go:398-400`, `R-ATT-005`).** This lands as a **mandatory delta on `agent-loop-skeleton`'s `R-LSK-001`**, which is the per-path emission-and-return contract for `Turn` (it already states the mid-stream obligation at `:40`), and as a **cross-reference amendment on `agent-turn-termination`'s `R-ATT-005`**, whose title scopes it to "a mid-stream terminal error" — without the cross-reference a reader of that capability would not find the pre-stream rule, which is this repo's known owning-spec-omission staleness shape.

**The fix lands on the emitter, never the validator.** `R-RUN-003` requires `CheckStream` to accept the recorded stream *unmodified* with `stream_check.go` **byte-unchanged** (`agent-run-driver/spec.md:125`), and `stream_check.go` is `R-LSK-004` substrate that has **never** been released to any milestone (`agent-loop-skeleton/spec.md:84`). AG-15 requests **no** release for it.

**Sub-questions `sdd-design` MUST close in writing:**

1. The `buildLoopRequest` path returns a **plain Go error**, not an `*ai.Failure`. `NewTurnEnd(runID, turnID, TurnOutcomeAborted, failure)` needs a non-nil `*Failure` under `TurnEnd.validate`'s failure-iff-aborted rule (`turn_events.go:123-128`, which `R-LSK-004` forbids changing). Design must name the wrap used on that path — the `typedFailureFromError` sibling shape (`scheduler.go:1062-1072`) is the ready precedent.
2. `turn_events.go` and `failure.go` are `R-LSK-004` substrate **released for AG-11 only**, and "the release does not extend to any milestone after AG-11 without its own recorded delta" (`:86`). AG-15's emission uses the **existing** `NewTurnEnd` constructor and adds no `TurnOutcome` member, so **no release is required** — design must state that conclusion explicitly rather than leave it inferred.
3. Does the run-close also need emitting on the nil-continuation pre-stream path (as `R-ATT-005` requires for mid-stream), or only the turn-close? The harness always drives the continuation path, where `R-LSK-001` point 2 suppresses the run-close; but the nil path is the one every existing standalone-`Turn` test exercises. Design must answer for **both** paths and prove no existing test changes.

---

## Decision 2 — the retry predicate **(DECIDED — stated as a table)**

The charter is explicit that the naive predicate is wrong: "the naive *retry if retryable* predicate is exactly what this scenario forbids" (`0003:1479`). The predicate is therefore stated as **ordered gates over typed evidence, first match wins**, evaluated on `terr` at `harness.go:447`.

**Evidence source.** `Turn` returns its fatal error **raw** — `turn.fatal` at `loop.go:411`/`:414`, `streamErr` at `:338`, the typed `*ai.PreStreamFailure` at `:327`. So `terr` is type-assertable to `*ai.Failure` **before** any Layer-2 wrapping, giving the predicate `Category()`, `Retryable()`, `RetryAfter()`, `PartialOutput()` and `Delivery()` directly. No `agent.Failure` accessor is needed at the decision point (Decision 5).

| Gate | Condition on `terr` | Verdict | Charter mapping |
|---|---|---|---|
| **G0** | `context.Cause(runCtx)` matches `ErrInterrupted` / `ErrShutdown` | **wind down** — never retried | Existing, unchanged (`harness.go:460-462`). `R-RUN-011` carve-out, verbatim. |
| **G1** | not an `*ai.Failure` (plain Go error: build error, `hist.Append` error, constructor error) | **surface** — no retry | No typed evidence exists, so no evidence-driven decision is possible. Fail closed. |
| **G2** | `Retryable() == false` | **surface immediately** | **AG-15.1 scenario 3** — "a terminal category … surfaces immediately regardless of position". `Retryable` is a caller-set classification field (`provider_failure.go:282-285`), independent of `Category`; the predicate reads the flag, not a category allowlist. |
| **G3** | `Retryable() == true` **and** `PartialOutput() == true` | **surface** with its partial content; run ends failed | **AG-15.1 scenario 2** — the G8 sentence at the harness. This gate is what makes the predicate non-naive: it fires *after* G2 has already said "retryable". |
| **G4** | `Retryable() == true`, `PartialOutput() == false`, attempt count `< H` | **retry**: re-invoke `Turn` over the unmodified transcript after the backoff wait | **AG-15.1 scenario 1** — "a turn failing retryable before any output … each attempt is a fresh provider call over an identical transcript". |
| **G5** | `Retryable() == true`, `PartialOutput() == false`, attempt count `== H` | **consult the failover seam**; on decline, surface the exhausted-retry report | **AG-15.3 scenario 1** — "when retries exhaust, the seam is consulted". |

**Each of AG-15.1's three scenarios maps to a distinct row**: scenario 1 → G4, scenario 2 → G3, scenario 3 → G2. No two scenarios share a gate, and no gate is unreachable.

**`Delivery()` is deliberately not a gate.** Layer 1 states why: "Independent of `PartialOutput`: delivery alone cannot distinguish the two mid-stream shapes" (`provider_failure.go:522-527`). `PartialOutput()` answers the charter's actual question — did a normalized output event precede this failure (`:618-620`) — and `Delivery()` does not. `Delivery()` is read for the **report**, not the decision. Consequence, stated so it is not discovered later: a **mid-stream** failure that emitted no output is retryable under G4, and this is stream-legal because the mid-stream path already emits `turn_end(Aborted)` (`loop.go:398-400`) and closes the bracket.

**Existing pins hold, checked rather than asserted.** `S-RUN-100`'s fixture (`harness_test.go:1870-1919`) asserts "the provider recorded exactly 1 request" for a scripted mid-stream failure. Its script builds `ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)` (`loop_test.go:1266-1268`) — the `Retryable` field is **unset**, therefore `false`, therefore **G2 catches it** and no retry occurs. `S-RUN-100` and its file stay green **byte-unchanged**, and it stays green for the strongest reason (the first gate), not incidentally. `sdd-verify` must re-check this against the final tree rather than trusting this paragraph.

**Binding on `sdd-design`:** the "identical transcript" claim of AG-15.1 scenario 1 must be **proved, not assumed**. `R-RUN-011`'s failure path appends nothing and never calls `CloseTurn`, and `Turn`'s fatal branch returns at `loop.go:411` before the finalize/history-wiring section — but design must open that path and demonstrate that no partial `History` entry survives a failed attempt, and that attempt *n+1*'s request body is byte-identical to attempt *n*'s. Layer 1's own byte-identical-replay scenario (`R-AIS-043 / S-1`, `ai-stream-lifecycle/spec.md:606-612`) is the shape to mirror at harness scope.

---

## Decision 3 — the failover seam's shape **(POSITION TAKEN: interface with a typed verdict, nil-default inert)**

Two precedents were weighed, both real and both citable.

| Precedent | Shape | Fit |
|---|---|---|
| **(a) AG-08 `PreRequestHook`** | `TurnOptions.PreRequestHook func(ctx, req) (ai.Request, error)`, nil = identity (`loop.go:655-675`, `R-PRH-005`) | Fits a **pure transform** of one value into a derived value of the same type. Failover is not a transform. |
| **(b) AG-10 `PermissionPolicy`** | `interface { Resolve(ctx, call) PermissionVerdict; Remember(...) bool }` (`permission_protocol.go:80-94`), nil = bypass, plus a shipped `NoOpPermissionPolicy` concrete stand-in (`S-LSK-010`) | Fits a **policy consulted for a verdict**, where the verdict is a typed value with more than one field and Layer 3 owns the implementation. |

**Position: (b), with (a)'s nil-default inertness kept.** Justification against the charter's own wording:

- "the seam is **consulted**" (`0003:1517`) — consultation returning a verdict, not a value transformed in place.
- "its **contract** documents what a real implementation must handle — re-counting the context budget, restarting the cache prefix" (`0003:1517`) — a *contract with documented implementer obligations* is a Go interface with a doc comment; a bare func field carries the doc but names no contract.
- The verdict is **not boolean**. A real failover must say *which* model/provider to switch to and how the budget is re-counted. A typed verdict struct extends non-breakingly (the AG-04 zero-value-extension posture used package-wide); `func(...) bool` does not.
- "the failover seam is a named injection point whose **v1 implementation declines**" (`0003:1452`) — "implementation" is the interface's word, and it requires an *installable value*, which a nil func field is not.
- AG-15.3 scenario 2 compares "with and without the seam installed" (`0003:1521`). This needs **two installable states**: nil (the identity default) and a shipped declining implementation. (b) provides both; (a) provides only nil-vs-a-closure.

**Concretely proposed** (names are indicative; `sdd-design` fixes them): a `FailoverPolicy` interface with one method returning a typed decline/accept verdict, carried as a nil-default field on the caller-owned `Harness` value (the `Scheduler.WindDownBound` zero-default idiom, `scheduler.go:611-613`), plus one shipped concrete declining implementation. Nil and the declining implementation MUST produce byte-identical event streams.

**`sdd-design` MUST close:** the verdict type's exact fields (an accept arm that no v1 implementation can take still needs a shape a v2 can fill without a breaking change); whether the seam lives on `Harness` or on `TurnOptions`; and whether "declines" is a distinct verdict member or the zero value.

---

## Decision 4 — `wrapHarnessFailure`'s discarded category **(POSITION TAKEN: preserve, via the AG-14 sibling precedent)**

`wrapHarnessFailure` (`harness.go:250-259`) builds `ai.FailureReport{Category: ai.FailureCategoryUnavailable, Cause: cause}` **unconditionally**, then `ai.MidStreamFailure(report, false)`. Every field of the real failure — category, retryability, retry-after, partial-output, delivery — is discarded for the `run_end` payload.

**Position: preserve the true evidence when `cause` is an `*ai.Failure`; keep `Unavailable` as the documented fallback for plain Go errors.**

Justification:

- AG-15's exhausted-retry report is meaningless without it. "Rate-limited, exhausted after H attempts" and "unavailable" are different operational facts, and the run-close is the only place a stream consumer can learn either.
- AG-14 already ruled on the identical principle one requirement earlier: a cancellation "MUST NOT report `ai.FailureCategoryUnavailable` for the run — a cancellation is not a provider outage, and the two must not be indistinguishable at the consumer" (`agent-run-driver/spec.md:247`). AG-15 extends the same rule to the failure path AG-14 explicitly left alone.
- The asymmetry is already visible in the tree: the loop preserves the category on `turn_end` by wrapping the raw `*ai.Failure` (`loop.go:387-388`; `turn_failure_test.go:344` pins `FailureCategoryRateLimit` surviving). Only the harness loses it, one event later.

**Shape: follow AG-14's sibling precedent, not a rewrite.** AG-14 did not change `typedFailureFromError`'s hardcoded `Unavailable`; it added `typedCancellationFailure` beside it and routed to it conditionally (`scheduler.go:1088-1114`, whose own doc comment names the hardcoding sibling at `:1101-1103`). AG-15 should do the same at `wrapHarnessFailure`: an `*ai.Failure` cause routes to a preserving construction, everything else keeps today's behavior byte-identically.

**`sdd-design` MUST close:** *how much* is preserved. Category is required. Retryability and retry-after are recommended (a consumer deciding whether to re-prompt wants both). **Partial-output and delivery are open**: `MidStreamFailure`'s second argument is `outputPreceded` (`provider_failure.go:622-624`), currently hardcoded `false`; and the wrapper builds mid-stream delivery even for a pre-stream cause. Preserving delivery would need a construction fork, and `ErrorEvent`'s pre-stream rejection (`:631-637`) does **not** apply here because `run_end` is not an `ErrorEvent` — design must verify that reading before relying on it. **Design must also grep for any existing test asserting the wrapped `run_end` category** and report the answer either way.

---

## Decision 5 — deferrals, each with a stated reason

| Explore risk | Disposition | Reason |
|---|---|---|
| **#1** pre-stream `turn_end` gap | **DECIDED NOW** — Decision 1 | Confirmed blocker; AG-15.1 scenario 1 is unsatisfiable without it. |
| **#2** `agent.Failure` has no `RetryAfter()` | **DECIDED NOW — do not add it.** | The harness's decision path type-asserts the raw `*ai.Failure` (Decision 2, evidence source), so nothing in AG-15 needs the wrapper accessor. Adding it would require editing `failure.go`, which is `R-LSK-004` substrate **released for AG-11 only** (`agent-loop-skeleton/spec.md:86`) — a new recorded release, for a surface no AG-15 scenario exercises. Recorded instead as an explicit non-requirement naming the omission and the reach-through that makes it non-blocking, so a later milestone finds the decision rather than rediscovering the gap. |
| **#3** no clock/timing seam in `agent`/`agenttest` | **DEFERRED to `sdd-design`** — shape only, existence is in scope | The seam must exist (AG-15.2's "backoff runs with injected timing"). Its *shape* is a design call: mirror `retry.Config`'s `NowFunc`/`SleepFunc`/`BaseDelay`/`MaxDelay`/`JitterSeed` (`retry.go:25-45`) freshly in the `agent` package. **Hard constraint, non-negotiable**: `backend/agent/src/ai/internal/retry` is an `internal` package rooted at `.../src/ai`, and `.../src/agent` is a **sibling**, so Go's internal-visibility rule forbids the import at compile time. **Mirror the shape, never import the package.** |
| **#4** `wrapHarnessFailure` discards the category | **DECIDED NOW** — Decision 4 | Position taken; extent of preservation deferred. |
| **#5** composed-ceiling documentation is net-new | **PARTLY RESOLVED — the explore pass missed that half of it already ships.** | See "The composed ceiling". |
| **#6** failover-seam shape | **DECIDED NOW** — Decision 3 | Position taken and justified against the charter's wording; field names deferred. |

Additional open item for `sdd-design`, not in the explore register: **the attempt bound `H` itself** — its value, whether it is a package constant or a zero-default `Harness` field (the `Scheduler.WindDownBound` idiom, `scheduler.go:611-613`), and whether it counts *attempts* or *retries after the first*. Layer 1 chose "retries after the initial request" (`retry.go:15-18`); Layer 2 choosing the other convention silently would corrupt the composed ceiling's arithmetic.

---

## The composed ceiling

**The number.** Layer 1's `DefaultMaxAttempts = 3` is the retry budget *after* the initial request (`retry.go:15-18`), and the call site is a zero-value `retry.Config{}` with no override plumbing from any caller (`openaicompat/stream.go:240`). So one logical `provider.Stream` call issues at most **4** wire requests. With harness attempts `H`, the worst-case provider-connection count for one logical harness retry sequence is **H × 4**.

**Where it is documented — the Layer 1 half already ships.** The explore pass recorded this as entirely net-new; it is not. `R-AIS-044` — "Composed-bound ceiling (cross-layer contract)" (`ai-stream-lifecycle/spec.md:634-650`) — is already promoted, and its Layer 1 scenario is already satisfied: `ai/internal/retry/doc.go:3-6` reads

> `DefaultMaxAttempts = 3` is the retry budget: a logical call may issue at most `N+1 = 4` wire requests when every pre-stream failure is retryable. The composed ceiling for Layer 2 is harness attempts × Layer 1 attempts. AG-15.2 (doc 0003, line 718) consumes this documented Layer 1 multiplier.

**AG-15.2 owes the Layer 2 half**, `R-AIS-044 / S-2` (`:646-650`): "the test cites the Layer 1 multiplier **verbatim** from the helper's package documentation, AND the composed-bound formula matches the helper's wording, AND **a divergence between the two layers' wording is observable as a test failure**."

Three obligations follow, and they are stronger than "write the number in a doc comment":

1. AG-15's own policy documentation states `H`, restates the formula in the helper's wording, and identifies `ai/internal/retry/doc.go` as its source.
2. A Layer 2 test asserts the ceiling by **provider call count** — `provider.Requests()` (`agenttest/fake_provider.go:157-161`) returns captured requests in call order — mirroring `R-ATT-008` / `S-ATT-010`.
3. A Layer 2 test makes divergence **detectable**: if `ai/internal/retry/doc.go`'s wording changes, the Layer 2 test fails. Since Layer 2 cannot import the internal package, `sdd-design` must choose the mechanism (reading the source file at a relative path is the obvious candidate) and prove it fails on a wording change. **`ai/internal/retry/doc.go` is read, never edited** — `R-RUN-012` requires every file under `backend/agent/src/ai/` byte-unchanged.

**Test-substrate note.** `agenttest.Provider` has **no "fail forever" mode**: `Stream` past the last scripted call fails loudly with `ErrScriptsExhausted` (`fake_provider.go:27-47`, `:101`). The charter's "scripted to fail pre-stream forever" is satisfied by queuing strictly more failing scripts than `H` and asserting `len(provider.Requests()) == H` exactly — which is a *stronger* proof of the harness's own bound than literal infinity would be.

---

## Capabilities

> Contract with `sdd-spec`. Existing names taken verbatim from `openspec/specs/`.

### New

- **`agent-retry-failover`** — the harness's retry policy and the failover seam: the predicate over typed evidence, attempt bounding, backoff with retry-after precedence, the cancellation-abortable wait, the injected timing seam, the composed-ceiling contract's Layer 2 half, the failover seam and its declining v1 implementation, and the exhausted-retry terminal report. IDs `R-RTY-0NN` / `S-RTY-0NN`, bites `S-RTY-0NN`. **Prefix `RTY` verified free**: zero `[RSN]-(RTY|RET|FLO|FVR)-[0-9]` matches repo-wide. Becomes `openspec/specs/agent-retry-failover/spec.md` at archive.

### Modified — deltas required

| Capability | What changes | Mandatory? |
|---|---|---|
| `agent-loop-skeleton` | **(1)** `R-LSK-001` gains the pre-stream `turn_end(Aborted)` emission obligation for all three paths (Decision 1), stated per continuation-path *and* nil path. **(2)** `R-LSK-004` records that AG-15 requests **no** substrate release — `turn_events.go`, `failure.go`, `stream_check.go` all stay byte-unchanged — and its filter-widening rule extends to AG-15's new filenames, byte-in-sync across both filters. | **Yes — blocking.** Without (1), AG-15.1 scenario 1 cannot pass `CheckStream`. |
| `agent-run-driver` | `R-RUN-011`'s "MUST NOT retry, back off, or route to a fallback provider" (`:241`) is the fence AG-15 removes for genuine, retryable, pre-output failures — and **keeps verbatim** for cancellations (`:248`) and for post-output failures. `R-RUN-003`'s bracket/lane rule is restated as still-true across N retry attempts. `:308`'s deferred-register row ("Retry or failover on a failed turn — **AG-15**") is back-annotated as closed. | **Yes.** |
| `agent-turn-termination` | `R-ATT-005`'s title scopes it to "a mid-stream terminal error"; a cross-reference to Decision 1's pre-stream rule keeps the capability from going stale by omission. `:153`'s deferred row ("Acting on retryability — AG-15") is back-annotated as closed. | **Yes.** |
| `ai-stream-lifecycle` | Back-annotation only: `R-AIS-044 / S-2` (`:646-650`) gets its Layer 2 consumer named. **No requirement text changes and no Layer 1 file is touched.** | Yes — annotation. |
| `agent-v1-scope` | Back-annotation only: the `AGS-D` failover entry (`:128`, `:133`) records that its placeholder node AG-15.3 has shipped the seam, with the implementation still deferred. | Yes — annotation. |
| `agent-cancellation-tree` | Back-annotation only: `:168`'s "Retry or failover on a cancelled turn — **AG-15**" row is confirmed closed by the gate ordering (G0 ahead of G2–G5), not weakened. | Yes — annotation. |

---

## Approach

1. **Close the bracket gap first** (Decision 1). Nothing else in AG-15.1 is testable until a retry produces a `CheckStream`-valid stream. This is the RED-first ordering constraint, not a preference.
2. **Extract the predicate as a pure function** over typed evidence — `(terr error, attempt int, bound int) → verdict` — so the whole table of Decision 2 is table-driven-testable without driving a run. The gates are ordered and first-match-wins; the ordering is the requirement.
3. **Insert the decision** at `harness.go`'s failure branch, strictly *after* the AG-14 cancellation re-check (`:460-462`) and *before* `failRun` (`:469`). G0 keeps its current position and code; AG-15 adds no second cancellation check.
4. **Backoff as a small, injectable waiter** in the `agent` package: `NowFunc`/`SleepFunc` mirroring `retry.Config`'s shape (`retry.go:25-45`) but **freshly implemented** — the internal-import boundary forbids reuse. The wait `select`s on `runCtx.Done()` alongside the timer, mirroring `defaultSleep` (`retry.go:192-207`) and `Scheduler.WindDownBound`'s wind-down timer (`scheduler.go:611-619`); an aborted wait routes to `windDownRun`, adding no new cancellation vocabulary.
5. **Retry-after overrides computed backoff** when present, mirroring `retry.Loop`'s own precedence (`retry.go:102-110`), read presence-typed so absent / reported-zero / reported-nonzero stay three distinguishable readings.
6. **The failover seam** is consulted exactly once, at G5, before the terminal report. Nil and the declining implementation are byte-identical on the stream.
7. **The terminal report** preserves the true category via the AG-14 sibling shape (Decision 4).

**Zero new `EventKind`.** Each attempt is already distinguishable by its own `turn_start`/`turn_end` pair, sequence number and fresh `TurnID` on the shared contiguous lane. `TurnStart{}` carries no payload (`turn_events.go:28-31`) and gains none. If a consumer must distinguish "a retry of the same logical turn" from "a new turn from a `ToolCalls` continuation", the design-owned convention is transcript-identity — a retried attempt's request transcript is byte-identical to the failed one, which is exactly what AG-15.1 scenario 1 already requires. **Rejected: a `turn_retry` event kind** — it would touch `event.go` and `event_registry_test.go`, both `R-LSK-004` substrate never released, and would break `R-RUN-012`'s no-new-kind pin for a distinction the charter never asks to be labeled.

---

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/loop.go` | **Modified** | Pre-stream `turn_end(Aborted, failure)` on all three paths (`:304-308`, `:317-328`, `:332-338`); the plain-error wrap for the build path |
| `backend/agent/src/agent/harness.go` | **Modified** | The retry decision between `:462` and `:469`; the attempt counter; the backoff call; the failover consult; category-preserving `wrapHarnessFailure` routing (`:250-259`) |
| New production file — retry policy, predicate, backoff, timing seam, failover seam + declining impl | **New** | Names fixed by `tasks.md`; each added to **both** substrate filters by exact filename suffix |
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | **New** | AG-15.1's 3 scenarios, AG-15.2's 2, AG-15.3's 2, plus bites and the predicate's table-driven unit coverage |
| `loop_test.go` (`:831-871`), `loop_hook_test.go` (`:907-943`) | **Modified** | Substrate filter widening by exact filename suffix, byte-in-sync |
| `openspec/specs/{agent-loop-skeleton, agent-run-driver, agent-turn-termination, ai-stream-lifecycle, agent-v1-scope, agent-cancellation-tree}` | **Delta** | Six deltas — three normative, three back-annotation |
| `docs/architecture/milestones/0003-…md` | **Modified** | AG-15 checklist tick, R-15/G8 back-annotation, milestone counter to 15/24 |
| `stream_check.go`, `stream_check_test.go`, `turn_events.go`, `failure.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `history.go`, `doc.go`, `doc_contract_guard_test.go`, `event_registry_test.go`, `reconstruction_test.go`, `go.mod`, `go.sum`, **all of `backend/agent/src/ai/**`** | **NOT TOUCHED** | No substrate release requested, no new event kind, no `History` surface change, no Layer 1 edit, no new dependency. `ai/internal/retry/doc.go` is **read** by a test, never written. |

---

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | The pre-stream `turn_end` emission changes behavior every existing standalone-`Turn` test observes on a failure path | **High** | The change is *additive on the stream* (one event before a close that already happens) but existing tests may assert exact event sequences on pre-stream failures. `sdd-design` MUST enumerate every test touching `loop.go:304-338`'s paths **before** apply, and `sdd-verify` MUST report which test files changed and why — a test edited to accommodate the new event is a conscious amendment needing sign-off, not a quiet fix. |
| 2 | A retry duplicates observable output — the exact defect G8 exists to prevent | Med | G3 gates on `PartialOutput()`, the accessor whose documented meaning is "did a normalized output event precede this failure" (`provider_failure.go:618-620`). A **bite** must prove the gate is non-vacuous: delete G3 → the partial-output scenario retries and fails. |
| 3 | `S-RUN-100`'s "exactly 1 request" pin breaks | Low | Verified: its fixture leaves `Retryable` unset → `false` → G2 catches it (`loop_test.go:1266-1268`). `sdd-verify` re-checks the file is **byte-unchanged**, not merely that the suite is green. |
| 4 | A backoff test synchronizes by wall clock and goes flaky under `-race` | Med | The injected timing seam is the whole point of AG-15.2's first scenario. `NFR-RUN-002`/`NFR-CAN-002` forbid sleep-based synchronization; a test needing a real sleep is a design failure, and `sdd-verify` must grep for `time.Sleep` in the new test files. |
| 5 | Someone imports `ai/internal/retry` from `agent` | Low (compile-time barrier) | Go's internal rule fails the build. Recorded here so no design or task proposes it and then discovers it. |
| 6 | The `R-AIS-044 / S-2` divergence-detection obligation is satisfied by a comment instead of a failing test | Med | The promoted scenario says "a divergence … is **observable as a test failure**" (`:650`). A **bite** must prove it: perturb the cited wording → the Layer 2 test fails. |
| 7 | Six spec deltas; one gets missed and a spec goes stale | Med | Enumerated with file and line above. This repo's known staleness clusters are owning-spec omission and un-back-annotated merges — **both** in play (Decision 1's `R-ATT-005` cross-reference is precisely the owning-spec-omission shape). `sdd-verify` re-reads each cited line against the shipped change. |
| 8 | Decision 4 changes a shipped helper's behavior and an unenumerated test pins the old category | Med | AG-14's sibling shape confines the change to the `*ai.Failure` arm; plain-error causes keep today's `Unavailable` byte-identically. Design must grep for `run_end` category assertions and report the answer either way. |
| 9 | The composed ceiling is documented as a sentence nobody can falsify | Med | It is tested twice: by `provider.Requests()` count against `H`, and by the divergence bite of risk 6. |
| 10 | Review budget exceeds even the raised 1000-line bar | **High (near-certain)** | `size:exception` pre-authorized at 1000 lines and extendable. See the forecast; a leaf-boundary slicing plan is stated there. |
| 11 | AG-16 is parallel and also touches `harness.go` | Low-Med | AG-15 and AG-16 are declared parallel by the charter (`0003:1453`). Both edit `harness.go`; conflicts are textual, not semantic. The worktree isolation holds; the merging orchestrator resolves. |

---

## Rollback Plan

Single revert of the AG-15 merge commit. The new retry-policy production file and its tests are deleted; `harness.go` returns to the unconditional `failRun` at `:469` and to the hardcoded `FailureCategoryUnavailable` wrap at `:250-254`; `loop.go`'s three pre-stream paths return to `closeSink`-and-return with no `turn_end`; both substrate filters return to their pre-AG-15 filename lists; the six spec deltas are dropped; doc 0003's AG-15 checklist line un-ticks and the counter returns to 14/24.

The revert is clean: nothing persists, no data migrates, no `go.mod`/`go.sum` change, nothing outside `backend/agent`, and no Layer 1 file was ever touched. The one **externally visible** removal is the `FailoverPolicy` interface — but Layer 3 does not exist yet (`0003:110`), so no live consumer is orphaned. Re-running `cd backend/agent && make test` at the parent commit confirms zero regression.

Forward-looking cost: AG-15 blocks nothing directly (AG-14 and AG-16 are parallel). Reverting re-opens G8's Layer 2 half and re-defers seam 7. It is a scheduling consequence, not a correctness one.

---

## Review-workload forecast

| Component | Estimate (authored, additions + deletions) |
|---|---|
| `loop.go` — pre-stream `turn_end` on three paths + the plain-error wrap | 60–110 |
| `harness.go` — retry decision, attempt counter, backoff call, failover consult, preserving wrap | 130–220 |
| New production file — predicate, bound, backoff + jitter, timing seam, failover seam + declining impl (this package's doc density) | 250–400 |
| Test files — 7 charter scenarios + the predicate's table-driven unit coverage + bites + the divergence test | 650–950 |
| `loop_test.go` / `loop_hook_test.go` — filter widening | 20–50 |
| Existing-test accommodation for the new pre-stream event (risk 1) | 0–120 |
| Doc 0003 checklist + traceability | ~10 |
| **Go subtotal** | **1110–1860** |
| SDD markdown — proposal, spec, **6 spec deltas**, design, tasks, apply-progress, verify-report | **850–1300** |
| **Total** | **1960–3160** |

`Decision needed before apply: No` — `size:exception` is **PRE-AUTHORIZED** at 1000 review lines and extendable, recorded here, one PR.
`Chained PRs recommended: No` — but if `sdd-tasks` forecasts above ~3000, slice at the leaf boundary, which the charter's own DAG already draws: **AG-15.1 → {AG-15.2, AG-15.3}** (`0003:1459-1460`). AG-15.1 (with Decision 1's companion delta) is independently deliverable and independently valuable; AG-15.2 and AG-15.3 each depend only on it and on nothing in each other, so they can be two sibling child PRs.
`400-line budget risk: High` — and high against the raised budget too.

**Work-unit breakdown** (the slicing plan if it is needed):

| Unit | Contents | Independently deliverable? |
|---|---|---|
| **U1** | Decision 1's pre-stream `turn_end` delta + its tests. Closes a real stream-validity defect on its own merits, independent of retry. | Yes |
| **U2** | AG-15.1 — the predicate, the attempt bound, the three scenarios, the G3 bite. | Yes, on U1 |
| **U3** | AG-15.2 — timing seam, backoff, retry-after precedence, context-abortable wait, composed ceiling + divergence test. | Yes, on U2 |
| **U4** | AG-15.3 — failover seam, declining implementation, inertness pin. | Yes, on U2 (parallel with U3) |

The SDD markdown counts toward the attempt budget. `sdd-tasks` MUST forecast against the **full** diff, not the Go diff.

---

## Dependencies

- **AG-13** (archived) — `Harness`, `Run`'s iteration and failure branch (`harness.go:410-470`), `failRun`, `wrapHarnessFailure`, the continuation seam, the shared `LaneStamper`.
- **AG-14** (merged `6485b937`) — `Interrupt`/`Shutdown`, `context.WithCancelCause` on the run (`harness.go:360-362`), the sentinels, `windDownRun`, the iteration-boundary cause check (`:418-420`), and the category-preserving sibling precedent (`scheduler.go:1088-1114`).
- **AG-11** (archived) — `R-ATT-005`'s mid-stream typed closing brackets (`loop.go:387-411`), `R-ATT-006`'s `PartialOutput()` on `agent.Failure` (`failure.go:76-81`), `R-ATT-007`'s partial content on the fatal path.
- **AG-07/AG-08** — `Turn`, its pre-stream paths, `TurnOptions`, the `PreRequestHook` nil-default precedent.
- **AG-10** — `PermissionPolicy`, the interface-with-typed-verdict precedent and its `NoOp` concrete stand-in.
- **AG-04** — `TurnEnd`/`NewTurnEnd`, `TurnOutcomeAborted`, the failure-iff-aborted validate rule, `CheckStream`.
- **Layer 1 (AI-19, AI-35)** — `ai.Failure` and its five accessors (`provider_failure.go:456-533`), `FailureReport.Retryable`/`RetryAfter` (`:277-310`), `retry.DefaultMaxAttempts = 3` (`retry.go:18`), the zero-value call site (`openaicompat/stream.go:240`), and the already-shipped Layer 1 half of `R-AIS-044` (`ai/internal/retry/doc.go:3-6`).
- **`agenttest`** — `NewProvider`, `Script`, `Requests()` (`fake_provider.go:157-161`), `ErrScriptsExhausted` (`:27-47`), `Gate`.
- **doc 0003:1444-1525** — the AG-15 charter and its three Gherkin leaves; **doc 0003:114-137** — the evidence gate and test-substrate binding.

---

## Success Criteria — restated as verifiable checks

- [ ] `cd backend/agent && make test` green with `-race`; all seven charter scenarios closed with recorded evidence
- [ ] **Decision 1** — all three pre-stream failure paths emit `turn_end(TurnOutcomeAborted, non-nil *Failure)` before closing the sink, on the continuation path **and** the nil path, with the answer for each path stated in the spec
- [ ] **AG-15.1 / G4** — a retryable pre-output failure retries up to the documented bound; each attempt is a fresh provider call over a **byte-identical** transcript; `CheckStream` accepts the recorded multi-attempt stream **unmodified**, with `stream_check.go` byte-unchanged
- [ ] **AG-15.1 / G3** — a failure after partial output is **not** retried; the typed failure surfaces on the stream carrying its partial content; the run ends failed; `provider.Requests()` has length exactly 1
- [ ] **AG-15.1 / G2** — a non-retryable failure surfaces immediately regardless of delivery position
- [ ] **Bites, RED-recorded before GREEN**: (a) delete G3 → the partial-output scenario retries and the assertion fails; (b) revert the pre-stream `turn_end` emission → the retry stream is rejected by `CheckStream` with `ai.ErrMisplaced`; (c) perturb `ai/internal/retry/doc.go`'s cited wording → the Layer 2 composed-bound test fails
- [ ] **AG-15.2** — retry-after **overrides** computed backoff when present, proven with injected timing and **zero** wall-clock sleeps in the test
- [ ] **AG-15.2** — an interrupt during backoff aborts the wait immediately and routes to `windDownRun`, with the run-close carrying the interrupted outcome and a **nil** `*Failure`
- [ ] **AG-15.2** — with more failing scripts queued than `H`, `len(provider.Requests()) == H` **exactly**; the composed ceiling `H × 4` is stated in the policy's documentation, cites `ai/internal/retry/doc.go` as its Layer 1 source, and matches its wording
- [ ] **AG-15.3** — the seam is consulted exactly once when retries exhaust; the v1 implementation declines; its contract documents re-counting the context budget and restarting the cache prefix
- [ ] **AG-15.3 (pin)** — the same failures run with the seam nil and with the declining implementation installed produce **identical** observable behavior
- [ ] **Decision 4** — the exhausted-retry `run_end` carries the **true** failure category, not `FailureCategoryUnavailable`; a plain-Go-error cause still reports `Unavailable` byte-identically
- [ ] `S-RUN-100` (`harness_test.go:1870`) and every AG-13/AG-14 harness and cancellation test pass with their source files **byte-unchanged**; any exception is an enumerated, signed-off amendment
- [ ] Zero `R-LSK-004` substrate files differ; `stream_check.go`, `turn_events.go`, `failure.go`, `event.go`, `run_events.go`, `history.go` and **every file under `backend/agent/src/ai/`** are byte-unchanged; `go.mod`/`go.sum` unchanged; the every-kind-constructible guard passes at its committed kind count (AG-15 adds zero)
- [ ] Both substrate filters carry an identical exact-filename entry set, one entry per file AG-15 introduces, with no wildcard, prefix, or directory pattern
- [ ] No `ai/internal/retry` import appears anywhere under `backend/agent/src/agent`
- [ ] Import guard and ambient-authority guard pass with **zero** changes
- [ ] All six spec deltas written; each cited line re-read against the shipped change by `sdd-verify`
- [ ] `make lint` clean (after `golangci-lint cache clean`), `make build` clean, `make vuln-check` clean — `vuln-check` is **not** in `make all`
- [ ] doc 0003's AG-15 checklist ticked, R-15/G8 back-annotated, milestone counter bumped to 15/24
