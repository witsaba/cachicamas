# Delta for `agent-run-driver` — AG-14 widens the surface, re-scopes run reuse, and carves cancellation out of the failure path

> **Change**: `cachicamas-agent-cancellation-tree` · **AG-14** (Layer 2, Wave 3), `0003:1371-1442`
> **Modifies**: `agent-run-driver` ([`../../../../specs/agent-run-driver/spec.md`](../../../../specs/agent-run-driver/spec.md)) — `R-RUN-001` (`spec.md:64-77`), `R-RUN-010` (`spec.md:196-209`), `R-RUN-011` (`spec.md:211-222`), and the Explicit non-requirements table (`spec.md:267-281`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES each block in the main spec with the MODIFIED block below; **full-block preservation is mandatory** and every scenario of every touched requirement is reproduced here verbatim unless explicitly amended.
> **Why this delta**: three things AG-13 wrote as closed are re-opened by AG-14 and must be re-scoped in writing rather than silently broken. (a) `R-RUN-001` pins the public surface as "exactly two methods" and `S-RUN-001` asserts "no third exported method"; AG-14 adds `Interrupt` and `Shutdown` on the same non-privileged upward path as `Steer` (`R-RUN-006`). (b) "One `Run` per harness value" (`spec.md:72`) forbids exactly the reuse AG-14.1's charter requires (`0003:1400`). (c) `R-RUN-011`'s "any error → `RunOutcomeFailed` with a non-nil failure" would mis-type every cancellation, and `R-RUN-010`'s forward reference — "richer cancellation vocabulary is **AG-14**'s" (`spec.md:200`) — is closed here and needs back-annotating, including an explicit answer on whether the wind-down bound is the "third path" that sentence forbids.
> **Ownership**: the signal semantics themselves — vocabulary, propagation, wind-down order, the bound, the detached-call report — are owned by [`../agent-cancellation-tree/spec.md`](../agent-cancellation-tree/spec.md) (`R-CAN-001`…`008`). This delta owns only what the *run driver's* existing requirements must now say.

## MODIFIED Requirements

### R-RUN-001 — The `Harness` is a value-form driver with a named four-method surface

The run driver MUST be a value-form struct with exported configuration fields, **no constructor** and **no interface**, mirroring the `Scheduler` precedent and the AG-04..AG-12 rule that a concrete boundary type stays a struct until a second implementation actually arrives.

Its public surface MUST be exactly the following methods, **enumerated by name rather than by count** so the pin stays meaningful as the type evolves:

| Method | Contract |
|---|---|
| `Run` | drives one run to its terminal and returns the last turn's `(ai.Message, ai.FinishReason, error)` |
| `Steer` | offers a user message to the in-flight run |
| `Interrupt` | fires the interrupt signal (`R-CAN-002`) |
| `Shutdown` | fires the shutdown signal and latches the terminal refusal (`R-CAN-005`) |

A method not named in that table MUST be observable as a guard failure, never as an unaudited addition. `Interrupt` and `Shutdown` sit beside `Steer` on the **same non-privileged upward path** (`R-RUN-006`): they reach the loop through no privileged channel and hold no handle into it — they flip a context the loop already observes.

**A consequence stated so `sdd-apply` is not read as a silent test rewrite:** the reflection subtest `"exactly two exported methods"` (`harness_test.go:1018-1024`, whose `want` is `[]string{"Run", "Steer"}` at `:1027`) changes as a direct consequence of this requirement. That edit is **conscious and delta-backed**, not incidental; its new expectation MUST be the four names above, sorted, and its subtest name MUST stop asserting a count.

Nil-valued optional fields MUST be resolved to defaults at `Run` entry into locals. The harness MUST NOT mutate the caller's fields, with exactly one recorded exception: it MAY set the sink-ownership flag of `R-RUN-012` once, on the scheduler it drives, before the first turn. AG-14 adds **no second caller-field mutation**: the wind-down bound is a caller-owned zero-default field on the `Scheduler` the caller already injects, which the harness reads and never writes.

`Steer` MUST guarantee **zero drops**: a `Steer` returning nil means the message enters the transcript before a subsequent provider call. After the run's terminal decision has been taken, `Steer` MUST return a typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))` — never a silent drop and never a nil return.

**One run at a time per harness value** — re-scoped from "one `Run` per harness value", because AG-14.1's charter requires the reuse the old wording forbade (`0003:1400`: "a new prompt on the same harness works afterward"). A run that has ended, whether completed, failed or interrupted, MAY be followed by another `Run` on the **same value**, and the steering queue MUST reopen at `Run` entry so the second run's `Steer` calls are accepted with the zero-drop guarantee above rather than meeting the closed queue the first run left behind. **Concurrent runs on one value stay out of scope** and are not made safe by this change. **Cross-run transcript state remains AG-21's**: the only state AG-14 lets outlive a run is the terminal, one-way shutdown flag of `R-CAN-005`, which holds no transcript and never resumes a run.

(Previously: the surface was pinned as "exactly two methods", `Run` and `Steer`, by a bare count; and the requirement closed with "One `Run` per harness value. Cross-run state is **AG-21**'s.", which forbade the same-value reuse AG-14.1 requires.)

#### Scenarios

- **S-RUN-001** — Given a harness value constructed as a struct literal with only its required provider field set, when `Run` drives a scripted single-turn conversation from an external test package, then the run completes without any constructor call, the caller-visible field values are unchanged after `Run` returns except for the recorded sink-ownership flag, and the type's exported method set read by reflection is equal, in both directions, to `{Run, Steer, Interrupt, Shutdown}` — an extra exported method fails the assertion and so does a missing one.
- **S-RUN-002** — Given a run that has taken its terminal decision and returned, when `Steer` is called with a well-formed user message, then it returns an error that satisfies the typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`, the transcript is unchanged, and no further event reaches the consumer sink.
- **S-RUN-003** — **AG-14 serial reuse.** Given a harness value whose first `Run` has returned through the interrupt wind-down of `R-CAN-002`, when a second `Run` is invoked on that same value, then it is accepted, a `Steer` issued during that second run returns nil and its message reaches the second run's transcript before the next provider call, and the second run reaches its own terminal — proving the queue reopened rather than staying closed from the first run. Cross-referenced to `S-CAN-002`.

### R-RUN-010 — The run survives a permission suspension spanning a turn

**Fourth acceptance clause (`0003:1302`).** A `PermissionDefer` verdict issued inside a harness-driven turn parks the call. The harness MUST hold a live handle on the `*Scheduler` that turn is blocked inside — obtained by injecting a caller-owned scheduler through the continuation seam — and MUST be able to release the parked call with `WakeParked` **while that turn is still in flight**. The run MUST then complete.

The resolution contract is closed and MUST be stated where a reader cannot miss it: a parked set lives exactly one `Schedule` call, so a suspension inside a harness-driven turn resolves by exactly one of (a) an external `WakeParked` on the injected scheduler, or (b) cancellation of the run context, which the harness propagates unmodified into every turn. The harness adds **no third path and no timeout**. A run whose policy defers and whose owner neither wakes nor cancels does not terminate — **by design**; `R-APP-009` is the safety net.

**The forward reference "richer cancellation vocabulary is AG-14's" is CLOSED by AG-14**, and the question it left open is answered explicitly rather than by silence: **the wind-down bound of `R-CAN-006` is NOT the third path this requirement forbids.** Two independent reasons, both checkable:

1. It is **part of path (b)**. The bound is armed only by cancellation of the run context; without a cancellation it never exists, so it adds no way for a parked call to resolve that (a) and (b) did not already provide. A parked call still resolves by a wake or by a cancellation, and by nothing else.
2. It creates **no timer on the uncancelled path**. On a run nobody cancelled, the join waits on the call's own completion alone, exactly as it does today. `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` (`permission_protocol_test.go:1573-1600`), which deliberately drives `Schedule` with a bare `context.Background()` precisely to freeze this contract, MUST therefore pass with its source **file-unchanged**. A change that makes that file need an edit has taken the third path and MUST be rejected.

What AG-14 *does* change on path (b) is the **type** of the abort, not its existence: the parked call's typed failure now names which signal cancelled it (`R-CAN-003`), where it previously carried a raw context error.

Synchronization MUST be the stream itself: because registration precedes emission and the emission's acknowledgement precedes the parked wait (`R-APP-002`), a consumer that has read the decision-required event off the run stream may wake with a guaranteed-live entry. No wall clock, no timeout, no sleep.

This requirement also claims the **known gap `agent-permission-protocol/spec.md:172` carried to AG-13**, quoted verbatim: "**Known gap (carried to AG-13)**: the `R-APP-002` acknowledgement itself currently has no non-vacuous guard — deleting the acknowledgement leaves the package green. The behaviour is present and correct in production; the missing bite must observe the parked **wait**, not the registration." The bite below MUST observe the **wait**. A bite that observes registration re-encodes the very gap it closes and MUST be rejected.

(Previously: the requirement deferred the question with "richer cancellation vocabulary is **AG-14**'s" and left unstated whether a wind-down bound would violate its own "no third path and no timeout" clause.)

#### Scenarios

- **S-RUN-090** — Given a permission policy that defers the first resolution and allows the second, a caller-owned scheduler injected into the run, and a test reading the consumer sink event by event, when the decision-required event is read, then the run has not returned (checked by a non-blocking read of the run's completion channel) and the tool has not been invoked; and when `WakeParked` is called for that call identity, then it returns nil, the call executes, the run completes with the completed run outcome, the suspension's events lie **inside turn one's bracket**, and `CheckStream` accepts the whole stream unmodified.
- **S-RUN-091** — **(bite)** Given a scratch edit that replaces the parked **wait** with an immediate re-resolution, when a scenario whose policy's second resolution asserts a flag the test sets immediately before `WakeParked` runs, then it FAILS because the flag is unset at re-resolution — proving the assertion observes the parked wait rather than the registration. RED-recorded over repeated runs per `NFR-APP-002`'s `-count=15` discipline, then reverted.
- **S-RUN-092** — **AG-14 adds no third path.** Given the AG-14 branch, when the suite runs and `git diff` is taken against the merge base, then `TestPermission_WakeParked_SchedulerReturnsAfterExplicitWake_NoDeadline` passes **and its source file is byte-unchanged**; and when a run whose policy defers is driven with neither a wake nor a signal, then it does not terminate on its own within any bound — the bound is unarmed because nothing cancelled the run.

### R-RUN-011 — A failed turn ends the run typed, with no append, no close, and no retry — cancellation carved out

When a `Turn` invocation returns a non-nil error **that is not cancellation-caused**, the harness MUST emit a run-close event carrying the failed run outcome and a non-nil typed failure, built through the public constructors, then close the consumer sink and return that error.

On that path the harness MUST NOT append anything to the transcript, MUST NOT call `CloseTurn`, and MUST NOT retry, back off, or route to a fallback provider. Retry and failover are **AG-15**'s, and this requirement is what makes that separation checkable rather than asserted.

**The cancellation carve-out, added by AG-14.** A `Turn` error whose cause matches one of the cancellation sentinels of `R-CAN-001`, and equally a signal observed at an iteration boundary before another `Turn` is started, MUST NOT take the failure path above. It MUST take the wind-down path instead, and on that path the harness:

- **MUST** synthesize orphans over the transcript and **MUST** call `CloseTurn` — the "no append, no close" rule above is hereby **re-scoped to genuine failures**. This is not a relaxation: an interrupted transcript with open calls is exactly what `R-HIS-007` exists to repair, and leaving it open would strand the pairing invariant;
- **MUST** emit a run-close carrying the interrupted or the shutdown run outcome and, per `RunEnd.validate`'s failure-iff-`Failed` rule (`run_events.go:156-161`), **no** `*Failure`;
- **MUST NOT** report `ai.FailureCategoryUnavailable` for the run — a cancellation is not a provider outage, and the two must not be indistinguishable at the consumer;
- **MUST NOT** retry, back off, or route to a fallback provider. The no-retry rule above extends to cancellations **verbatim**: a cancelled turn is never retried.

A bare cancellation of the *caller*'s context, not routed through the signal methods, keeps this requirement's original failure routing (`R-CAN-001`, scope line).

This failure path MUST also close the run's steering queue as part of the same termination, under the queue's own mutex — the same critical section `R-RUN-002`'s atomic terminal decision uses. **Every** `Run` exit — this failure path, the cancellation wind-down path, a rejected prompt or steered-message append that reaches it, and the terminal-decision success path — MUST leave the queue closed before `Run` returns, so a `Steer` call that reaches the harness afterward always observes it closed and receives `R-RUN-001`'s typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`. A `Run` exit that leaves the queue open makes every later `Steer` return nil forever — indistinguishable, from the caller's side, from the silent drop `R-RUN-001` forbids. The reopen of `R-RUN-001` happens at the *next* `Run`'s entry, never at an exit.

(Previously: every non-nil `Turn` error routed to the failed run outcome with a non-nil failure, and the "no append, no `CloseTurn`" rule was unconditional — so a cancelled run reached the consumer as failed/unavailable and left its transcript's open calls unrepaired.)

#### Scenarios

- **S-RUN-100** — Given a provider scripted with a terminal mid-stream failure on turn one and a further script available so that a retry would be observable, when the run is driven, then the consumer observes the turn's typed closing brackets followed by a run-close carrying the failed run outcome with a non-nil failure and then the sink close; `Run` returns a non-nil error; the transcript holds only what was committed before the failure, with no entry appended after it; and the provider recorded exactly one request.
- **S-RUN-101** — Given a run that has ended through this requirement's failure path, when `Steer` is called after `Run` has returned, then it returns `R-RUN-001`'s typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`, never nil — the queue closes on every `Run` exit, not only the terminal-decision success path.
- **S-RUN-102** — **AG-14 carve-out.** Given a run interrupted mid-turn with a further script available so that a retry would be observable, when the run winds down, then the run-close carries the interrupted run outcome with a **nil** failure, no run-close carrying the failed outcome appears anywhere on the stream, the transcript's previously open calls are closed by synthesized-origin entries and the turn is closed, the provider recorded no request after the signal, and a `Steer` issued after `Run` returned receives the typed rejection — the queue closed on this exit too. Cross-referenced to `S-CAN-001` and `S-CAN-011`.

## MODIFIED Explicit non-requirements

The table is reproduced in full; three rows are back-annotated and none is removed.

Stated so that no test, guard or acceptance line is written as if AG-13 closes more than it does. Each row names its owning milestone.

| Not claimed | Owner |
|---|---|
| Cross-event run-identity consistency enforced by `CheckStream` | **AG-19.** AG-13 upholds the property by construction and asserts it in its own tests (`R-RUN-004`); the validator is not strengthened and `stream_check.go` stays byte-unchanged. *(Still true at AG-14: `stream_check.go` stays byte-unchanged.)* |
| Retry or failover on a failed turn | **AG-15.** Charter "Out of scope" line, binding (`R-RUN-011`). *(Still open at AG-14, which extends the no-retry rule to cancellations rather than implementing retry.)* |
| A compaction check between turns | **AG-17** inserts it at AG-13's turn boundary; **AG-18** implements compaction. Nothing here anticipates its shape |
| Cancellation semantics — interrupt vs. shutdown vs. deadline | **AG-14.** AG-13 propagates the context unmodified and stops; it defines no cancellation vocabulary and adds no timeout. **PARTIALLY CLOSED by AG-14**: interrupt and shutdown ship here, owned by `agent-cancellation-tree` (`R-CAN-001`…`008`). **Deadline remains unclaimed** — `0003:1373`'s "both ≠ deadline" is a distinctness claim, not a mandate, and AG-14.3's bound is a wind-down bound, not a run deadline |
| Cost aggregation across turns | **AG-16** |
| A production subagent tool, nested or child runs | **AG-19.** No subagent tool ships in v1 (`0003:1794`) |
| Multi-turn state beyond a single run | **AG-21** (`agent-run-driver`'s `R-RUN-001`, its one-run-at-a-time clause, this capability's own re-scoped clause in `R-RUN-001`). AG-13 owns one run's iteration; AG-14 adds serial reuse of a harness value and one terminal, transcript-free shutdown flag, and pre-empts nothing else. **Pointer corrected:** this row previously cited `agent-loop-skeleton/spec.md:106`, which is `R-LSK-005` coverage text and has nothing to do with cross-run state. That was a **pre-existing citation defect**, recorded here and corrected rather than propagated; the same stale pointer appears in this change's proposal risk row 7 and MUST NOT be copied forward |
| Persistence or session reload of a run | **Layer 3.** The harness holds state in memory and never touches a file (`0003:110`) |
| A real provider or a real tool | **Never in Layer 2.** `agenttest` scripts only (`0003:123`) |
| Any edit under `backend/agent/src/ai/**` | **Not this milestone.** Layer 1 is consumed, never edited |
| A new `EventKind`, a new `TurnOutcome`, or a new exported `History` method | **Not this milestone**, and forbidden under this change. *(Still true at AG-14: it registers no `EventKind`, adds no `TurnOutcome`, and adds no exported `History` method. It does add one `RunOutcome` member under the recorded `R-LSK-004` release — a different vocabulary from the two named here.)* |
| An `L2C-08` doc-contract row | **Not this milestone.** AG-13 declares no new package-wide guarantee; it rides `L2C-03`/`L2C-04`/`L2C-05`/`L2C-07` at run scope. AG-19's child runs re-open the question. **CLOSED by AG-14, earlier than forecast**: AG-14 declares a package-wide liveness and control guarantee no existing row covers, so the `L2C-08` row ships here, owned by `R-CAN-008` |
