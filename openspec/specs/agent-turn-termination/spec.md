# Spec — Complete turn termination and typed failure reporting (`agent-turn-termination`)

> **Change**: `cachicamas-agent-turn-termination` · **AG-11** (Layer 2, Wave 2, milestone 11 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-11--complete-turn-termination-and-typed-failure-reporting), `0003:1113-1176`
> **Nodes**: AG-11.1 `[leaf]` (finish-reason dispatch, `0003:1132-1151`) · AG-11.2 `[leaf]` (typed failure upward, `0003:1153-1170`)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario independently verifiable by `cd backend/agent && make test`.
> **IDs**: `R-ATT-0NN` / `S-ATT-0NN`, bites `S-TTB-0NN`. Append-only. Distinct from `R-AEV-`/`R-AMT-`/`R-APE-`/`R-AGE-`/`R-LSK-`/`R-TLS-`/`R-APP-`.
> **Scenario count**: allocated `S-ATT-001` through `S-ATT-015`, plus the five bites `S-TTB-001`…`S-TTB-005`, across requirements allocated `R-ATT-001` through `R-ATT-010`. Each milestone that appends records its own additions in its delta; this header states the allocated range and never a total, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`).
> **Inherited decision (CLOSED, not re-openable here)**: the extended vocabulary is **1:1** — each of the seven `ai.FinishReason` members maps to its own distinct `TurnOutcome` member (`proposal.md` "Resolved design fork"). A collapsed subset is not specified.
>
> **Amended 2026-08-20 (AG-20)**: `R-ATT-010` ADDED, alongside `R-ATT-001` and `R-ATT-009`, both of which are byte-unchanged. BACK-ANNOTATION ONLY: no outcome member is added, removed or renumbered; no `String()` form changes; no constructor signature changes; the failure-iff-aborted rule is untouched. `turn_events.go` and `failure.go` — released for AG-11 only — are forbidden again at AG-20 and stay byte-unchanged. AG-20 is the outcome vocabulary's first non-stream consumer, reading the outcome from the forwarded turn-close payload. See delta spec at `openspec/changes/cachicamas-agent-hook-taxonomy/specs/agent-turn-termination/spec.md`.

## Coverage

| Charter leaf | Requirements | Spec scenarios | Bites |
|---|---|---|---|
| AG-11.1 (`0003:1137-1148`) | `R-ATT-001`…`004` | 7 (incl. `S-ATT-012`) | 3 |
| AG-11.2 (`0003:1158-1168`) | `R-ATT-005`…`008` | 4 | 2 |
| Cross-cut (substrate deviation) | `R-ATT-009` | 1 | 0 |

Charter → spec: `0003:1137-1141` → `R-ATT-001`/`002`/`003`; `0003:1143-1147` → `R-ATT-004`; `0003:1158-1162` → `R-ATT-005`/`006`/`007`; `0003:1164-1167` → `R-ATT-008`.

## Purpose

Today the loop terminates every turn as if it succeeded and terminates a failing turn silently: `finalize()` hardcodes `TurnOutcomeFinished`/`RunOutcomeCompleted` (`loop.go:613,617`, verified via the caller at `loop.go:257,286`), and the mid-stream-fatal branch returns `ai.Message{}, 0, turn.fatal` after `closeSink` with **no** `turn_end` and **no** `run_end` (`loop.go:270-276`, read verbatim). AG-11 makes turn completion a decision over the whole finish-reason vocabulary and makes provider failure a typed value that reaches the consumer — **deciding nothing about retries** (`0003:1119`).

## Requirements

### R-ATT-001 — `TurnOutcome` carries one distinct member per finish reason

`TurnOutcome` MUST grow from its two current members (`TurnOutcomeFinished`, `TurnOutcomeAborted`; `turn_events.go:74-89`) to a vocabulary in which **every** `ai.FinishReason` member (`finish_reason.go:41-114`: `Stop`, `Length`, `ToolCalls`, `ContentFilter`, `Refusal`, `PauseTurn`, `Unknown`) is represented by a **distinct** `TurnOutcome` value, plus `TurnOutcomeAborted`, which is not a finish reason. No two finish reasons MAY share an outcome value.

The vocabulary's existing structural properties MUST be preserved unchanged: the zero value MUST remain a non-member rejected by `TurnEnd.validate` (`turn_events.go:120`), every member MUST render a distinct non-placeholder `String()` form (`turn_events.go:93-104`), and the "a `Failure` is present **iff** the outcome is `TurnOutcomeAborted`" rule (`turn_events.go:123-128`) MUST hold verbatim — only `TurnOutcomeAborted` ever carries one.

#### Scenarios

- **S-ATT-001** — Distinct member per finish reason. Given the extended `TurnOutcome` vocabulary, when every value in `1..turnOutcomeLimit-1` is enumerated from an external test package, then each is accepted by `NewTurnEnd` (with a `*Failure` supplied exactly for the aborted member), each renders a `String()` form that is neither `"unset"` nor the `turnoutcome(N)` placeholder, and the set of rendered forms has no duplicates.
- **S-ATT-002** — The zero value stays a non-member and the failure rule is unchanged. Given the extended vocabulary, when `NewTurnEnd(run, turn, 0, nil)` is called, then it returns a validation violation; and when `NewTurnEnd(run, turn, o, f)` is called with a non-nil `f` for any non-aborted member `o`, then it returns a violation; and when it is called with a nil `f` for the aborted member, then it returns a violation.

### R-ATT-002 — Exhaustive dispatch consumed by the loop's terminator

The loop MUST decide the turn's outcome by dispatching on the turn's normalized `ai.FinishReason` through a single, exhaustive mapping function, and the loop's terminator (`finalize()`) MUST consume that mapping instead of hardcoding `TurnOutcomeFinished` (`loop.go:613`). The mapping MUST be total over the finish-reason vocabulary: it MUST NOT fall through to a success outcome for a reason it does not name.

A finish reason that the mapping does not name MUST be observable as a failure of the suite (per `R-ATT-003`), never as a silently-`Finished` turn.

#### Scenarios

- **S-ATT-003** — Each finish reason produces its own `turn_end`. Given a provider scripted to complete with finish reason `r`, for each `r` in the seven-member vocabulary in turn, when `Turn` runs and the consumer drains `sink`, then the `turn_end` event's outcome is the distinct outcome assigned to `r` by `R-ATT-001`, and the outcomes observed across the seven runs are pairwise different.
- **S-TTB-001** — **(bite)** Given the dispatch table with one finish reason's entry removed or aliased onto another reason's outcome, when `S-ATT-003` runs, then it reports a duplicate or missing outcome — proving the per-reason assertion is non-vacuous. RED-recorded BEFORE `S-ATT-003` is GREEN.

### R-ATT-003 — Agent-level exhaustiveness pin (the third instance of the idiom)

The `agent` package's external test surface MUST carry an exhaustiveness pin that walks `ai.FinishReason` candidates `0..255`, admits every candidate the Layer 1 package validates, and asserts that the loop's dispatch names each admitted value. An **eighth** `ai.FinishReason` added upstream MUST fail this suite (or the build) until the loop handles it.

This is the **third** instance of an idiom already established at two layers — `ai/finish_reason_test.go:277-320` (walks `0 <= candidate <= 255`, skips values failing `Validate()`, and errors on any validating value the test does not name) and `agenttest/conformance_capabilities.go:247,265` — extended here to the agent-loop layer. The pin MUST be closed in both directions: a value the dispatch names but Layer 1 does not validate MUST also fail.

#### Scenarios

- **S-ATT-004** — Bidirectional pin over `0..255`. Given the loop's finish-reason dispatch, when candidates `0..255` are walked and each candidate accepted by `ai.FinishReason.Validate()` is dispatched, then every accepted candidate yields a member-valued `TurnOutcome`, the count of accepted candidates equals the count of reasons the pin names, and any name in the pin that Layer 1 does not validate is reported as a failure.
- **S-TTB-002** — **(bite)** Given a probe value forced to validate as an `ai.FinishReason` while absent from the loop's dispatch, when `S-ATT-004` runs, then the pin fails and its message names the unhandled value — proving a future eighth reason cannot pass unnoticed. RED-recorded BEFORE `S-ATT-004` is GREEN.

### R-ATT-004 — Refusal and pause diverge, and both diverge from finished

A turn ending in `FinishReasonRefusal` and a turn ending in `FinishReasonPauseTurn` MUST be distinguishable **from each other and from a normally finished turn by the `TurnOutcome` value alone**. A consumer MUST NOT need to inspect any `ai.FinishReason` to tell them apart.

Refusal MUST mean the turn is over: it is a complete answer of the form "no" (`finish_reason.go:71-80`), so the loop MUST close the turn's brackets and MUST NOT mark it resumable. Pause MUST be visible as its own **suspended-resumable** outcome carrying the obligation recorded at `finish_reason.go:82-91` — the turn expects resumption with the content already received replayed **verbatim**. The content the loop returns and emits for a paused turn MUST therefore be byte-identical to the content received, with no re-serialisation, re-ordering, or stripping of an opaque round-trip token.

**Acting on** a pause — performing the resumption — was AG-13's, not this capability's (`0003:1123`), and **AG-13.3 has now done it**. The division of labour is recorded here so no later reader re-derives it: this capability owns the pause's **visibility**, AG-13 owns the **action**. Concretely, AG-13 imposes two obligations on this capability that are already satisfied and MUST stay satisfied:

- the paused turn's `turn_end` MUST continue to carry the paused outcome, and a caller driving a run MUST forward it unchanged rather than rewriting or absorbing it — a resumption that hid the pause would defeat this requirement's whole point;
- the partial `ai.Message` this capability requires the loop to return MUST remain the exact value a resumption replays. AG-13 appends **that value** and re-includes it verbatim in the next request's transcript; it never re-synthesizes an equivalent one. If this capability ever stopped returning byte-identical content, `R-RUN-009` would break, so the byte-identity clause above is now load-bearing for two capabilities rather than one.

A refusal MUST still terminate. Refusal is a terminal candidate for the run driver's dispatch (`R-RUN-002`), never an iteration trigger — the divergence this requirement pins is what makes that dispatch correct.

(Previously: the requirement deferred the pause's resumption to AG-13 with no statement of what AG-13 would depend on, so the byte-identity clause read as a property of one capability rather than as a contract two capabilities share.)

#### Scenarios

- **S-ATT-005** — Refusal, pause and finished are three values. Given three turns scripted to end in `FinishReasonRefusal`, `FinishReasonPauseTurn` and `FinishReasonStop` respectively, when each runs and its `turn_end` outcome is read, then the three outcome values are pairwise different, and no assertion in the scenario reads an `ai.FinishReason` to distinguish them.
- **S-ATT-006** — Pause replays received content verbatim. Given a turn scripted with interleaved reasoning and text deltas, a non-empty `[]byte` reasoning round-trip token, and a completion carrying `FinishReasonPauseTurn`, when `Turn` runs, then the returned `msg` and the emitted delta sequence carry the received fragments byte-for-byte, the reasoning round-trip token is byte-equal to the script's, and the `turn_end` outcome is the paused member — not the refused member and not `TurnOutcomeFinished`.
- **S-TTB-003** — **(bite)** Given the pause outcome aliased onto the refusal outcome (or onto `TurnOutcomeFinished`), when `S-ATT-005` runs, then it reports the collapse — the loop-termination defect AI-13 exists to prevent (`0003:1121`) is detected, not assumed absent. RED-recorded BEFORE `S-ATT-005` is GREEN.
- **S-ATT-013** — **AG-13 forwards the pause, and the partial survives the run.** Given a harness-driven run whose first turn ends in `FinishReasonPauseTurn` and whose second ends in a terminal reason, when the run's consumer stream is read, then turn one's `turn_end` carries the paused outcome unrewritten and unabsorbed; and when the transcript entry for that turn is compared to the message `Turn` returned, then the two are byte-identical including the reasoning round-trip token. Owned jointly with `R-RUN-009` / `S-RUN-080` / `S-RUN-081`.

### R-ATT-005 — A mid-stream terminal error emits typed closing brackets

When the provider stream ends in a terminal error, the loop MUST emit, on `sink` and **before** the sink is closed, a `turn_end` carrying `TurnOutcomeAborted` and a non-nil `*Failure`, followed by a `run_end` carrying `RunOutcomeFailed` and the same `*Failure`. The consumer MUST NOT observe the sink close without those two typed events (today it does: `loop.go:270-276` drains, closes and returns with nothing emitted).

The loop MUST NOT register a new `EventKind` for this: failures ride the typed outcomes already registered (`failure.go:6-7`), and the every-kind-constructible guard MUST still pass at its committed kind count.

**Cross-reference, added by AG-15 — the pre-stream half of the same obligation lives in `R-LSK-001`.** This requirement governs the **mid-stream** terminal error only, and a reader MUST NOT infer from its scope that a failure occurring *before* the stream opens may close the sink with the turn bracket still open. Since AG-15, `Turn`'s three pre-stream failure paths — the request-build error, the pre-request-hook error, and the `provider.Stream` error — carry the **same** obligation in the same shape: `turn_end(TurnOutcomeAborted)` with a non-nil `*Failure` before the sink closes, plus `run_end(RunOutcomeFailed)` on the nil-continuation path and **not** on the continuation path, where the run bracket is the caller's (`R-LSK-001`, `R-RUN-011`). That rule is owned by `agent-loop-skeleton`, not re-homed here; this requirement records the pointer so the pair of rules cannot be read as a single rule with a hole in it.

The two halves stay **distinguishable** in one respect worth stating, because it changes what a consumer may conclude: a mid-stream abort's `*Failure` may report partial output and mid-stream delivery, while a pre-stream abort's reports neither — nothing crossed the carrier. Both are `TurnOutcomeAborted`; the evidence, not the outcome member, is what separates them, and `R-RTY-001`'s gate G3 is the first consumer that depends on the difference.

(Previously: the requirement stated the mid-stream obligation alone, with no statement of what a pre-stream failure owed the stream — so the capability that owns typed turn termination was silent about the failure class AG-15 makes reachable in a retry loop.)

#### Scenarios

- **S-ATT-007** — Typed brackets on the fatal path. Given a provider stream scripted to emit a message start, one or more deltas, a message end, and then a terminal `ai.ErrorEvent` carrying a mid-stream `*ai.Failure`, when `Turn` runs and the consumer drains `sink` to close, then the consumer observes `turn_end` with `TurnOutcomeAborted` carrying a non-nil `*Failure`, then `run_end` with `RunOutcomeFailed` carrying a `*Failure`, then the channel close, in that order; and `Turn` returns a non-nil error.
- **S-TTB-004** — **(bite)** Given the fatal branch restored to its pre-AG-11 shape (drain, close, return), when `S-ATT-007` runs, then the consumer observes the channel close with neither `turn_end` nor `run_end` and the scenario fails — proving the emission assertion is non-vacuous. RED-recorded BEFORE `S-ATT-007` is GREEN.
- **S-ATT-014** — **AG-15 cross-reference is live, not decorative.** Given the three pre-stream failure conditions of `R-LSK-001`'s AG-15 amendment driven through `Turn` with a nil continuation, when each consumer drains `sink` to close, then each observes a `turn_end` carrying `TurnOutcomeAborted` with a non-nil `*Failure` before the close, exactly as this requirement's mid-stream path does; and when each aborting `*Failure` is inspected, then it reports `PartialOutput() == false` and pre-stream delivery, while `S-ATT-007`'s mid-stream failure reports mid-stream delivery — so the two abort classes stay distinguishable by evidence while sharing one outcome member. Owned jointly with `R-LSK-001` / `S-LSK-021`; the emission itself is `agent-loop-skeleton`'s obligation, and this scenario exists so that a regression in it fails **this** capability's suite too.

### R-ATT-006 — `PartialOutput()` on Layer 2's typed-failure surface (invariant 4, jointly with AG-04.3)

`agent.Failure` MUST expose `PartialOutput() bool`, delegating unchanged to `(*ai.Failure).PartialOutput()` (`provider_failure.go:515-520`) and, like its three siblings `Category()` (`failure.go:44`), `Delivery()` (`:54`) and `Retryable()` (`:64`), returning the zero value (`false`) for a nil receiver rather than panicking.

Without it, AG-11.2's charter phrase "including the partial-output discriminator" (`0003:1161`) is unsatisfiable through Layer 2, because Layer 1's accessor is unreachable from a consumer holding an `*agent.Failure`. This requirement closes envelope invariant 4 **jointly with AG-04.3**, as `agent-event-envelope/spec.md:132` and `invariant_pin_test.go:1-8` already declare ("invariant 1 closes jointly with AG-05.1; invariant 4 closes jointly with AG-11.2"). Its scenario MUST land in that file's family so the joint-closure claim stays auditable.

#### Scenarios

- **S-ATT-008** — The discriminator is inspectable through Layer 2, nil-safe. Given a `*ai.Failure` constructed on the mid-stream path with the partial-output flag set and a second with it unset, when each is wrapped via `agent.NewFailure` and inspected from an external test package, then `PartialOutput()` reports `true` and `false` respectively, a nil `*agent.Failure` reports `false` without panicking, and no assertion reads a failure message string.

### R-ATT-007 — The failing turn's partial assistant content reaches the caller

On the mid-stream-fatal path the loop MUST return the assistant message reconstructed from the content accumulated before the failure, not the zero `ai.Message{}` it returns today (`loop.go:275`). The `*Failure` carried by the `turn_end` and `run_end` of `R-ATT-005` MUST report `PartialOutput() == true` when such content exists, so the discriminator and the content agree.

`Turn`'s **signature** MUST NOT change; only the value returned on this path changes.

#### Scenarios

- **S-ATT-009** — Partial content survives the failure. Given a provider stream scripted to deliver text content and then a terminal mid-stream failure, when `Turn` returns, then the returned `msg` carries the delivered content byte-for-byte (not the zero message), the emitted deltas reconstruct to that same message via the AG-05.3 reconstruction helper, and the `*Failure` on the `turn_end` reports `PartialOutput() == true`.
- **S-TTB-005** — **(bite)** Given the fatal branch returning `ai.Message{}`, when `S-ATT-009` runs, then the returned message is empty and the scenario fails — proving the content assertion is non-vacuous. RED-recorded BEFORE `S-ATT-009` is GREEN.

### R-ATT-008 — The loop never issues a second provider call

On **any** failing turn the loop MUST wind the turn down after exactly one provider call. Verified from the loop side: the fake provider's captured request log (`agenttest/fake_provider.go:157-161`, `Requests()` returns one entry per consumed script, in call order) MUST have length exactly `1`.

Retry is the harness's decision (AG-15), and this requirement is what makes that separation checkable rather than asserted.

#### Scenarios

- **S-ATT-010** — Exactly one call on a failing turn. Given a fake provider scripted with a terminal mid-stream failure (and a second script available, so a retry would be observable rather than an error), when `Turn` runs to completion, then `len(provider.Requests()) == 1`, and the same holds for a turn failing before any content is delivered.

### R-ATT-009 — Substrate deviation: exact-filename widening only, recorded in the owning specs

AG-11 is the first milestone to modify **pre-existing** substrate files (`turn_events.go`, `failure.go`) rather than only appending new ones; the reason is structural (Go const-block, `validate()` bound-check at `turn_events.go:120` and `String()` at `:93-104` are local to `turn_events.go`), not convenience. The substrate guards' allowlists MUST therefore widen by **exact filename suffixes only** — no wildcard, no prefix, no directory-level relaxation — and `filterOutLoopFiles` (`loop_test.go:831-871`) and `filterOutLoopHookFiles` (`loop_hook_test.go:907-943`) MUST stay in sync with each other.

Widening the filters alone would leave merged code contradicting two canonical specs. The owning requirements MUST be amended by the deltas shipped with this change: `R-LSK-004` (`agent-loop-skeleton`, which names both `turn_events.go` and `failure.go`), `R-APP-012` (`agent-permission-protocol`, which names `failure.go` but **not** `turn_events.go`), and `R-AEV-008` (`agent-event-envelope`, for the typed-failure surface's growth).

#### Scenarios

- **S-ATT-011** — Only the released files differ, and the specs say so. Given the merge base of the AG-11 branch with `origin/main`, when `git diff` is taken over `backend/agent/src/agent/` and over `go.mod`/`go.sum`, then the only pre-existing non-test files that differ are `turn_events.go`, `failure.go` and `loop.go`; every other substrate file named by `R-LSK-004` and `R-APP-012` is byte-unchanged; the `go.mod`/`go.sum` diff is empty; the every-kind-constructible guard still passes at 25 kinds; both substrate guards pass; both filters carry the identical set of exact-filename entries; and this change's spec deltas for `R-LSK-004`, `R-APP-012` and `R-AEV-008` exist.

### R-ATT-010 — AG-20 CONSUMES the turn-outcome vocabulary and adds no member, and the consumption is on FOURTEEN enumerated exits

**AG-20 is the outcome vocabulary's first non-stream consumer**, and that is the whole of its interaction with this capability. The per-logical-turn observation report carries the turn's outcome as a payload field, read from the turn-close event the per-attempt forwarder already handles. Three consequences MUST be stated rather than inferred:

1. **No member is added, and the reason is that none is needed.** Every exit AG-20 reports on maps onto an outcome the vocabulary already carries: a completed logical turn reports the finished outcome, and a failed, interrupted or wound-down one reports the aborted outcome, in each case the value the turn's own close event carried. An outcome member meaning "observed by a hook" would be a category error — the report is *about* a turn, it is not a turn state.
2. **The read is a PURE READ downstream of the forwarder, so the outcome the report carries is by construction the outcome the STREAM carries.** They cannot diverge: there is one source. A reviewer checking a report against a recorded stream is checking an identity, not a correspondence, and `S-ATT-015` asserts it that way.
3. **The consumption is total over the logical-turn loop's exits, and the enumeration is normative in `R-HKS-004`.** Fourteen exits are enumerated there with an explicit fires/does-not-fire verdict each, collapsing to four enqueue sites so that no exit can fire twice and no firing exit can be skipped by an early return above it. This capability's contribution is the guarantee the report leans on: **a logical turn that ran has exactly one turn-close event carrying exactly one outcome**, so "the turn's outcome" is well defined for every yes-row and undefined — and therefore not reported — for every no-row.

**This requirement adds no obligation to any producer.** It records what AG-20 reads, so that a later milestone contemplating a new outcome member knows a second consumer exists.

#### Scenarios

- **S-ATT-015** — **AG-20: the reported outcome IS the streamed outcome, and no member was added.** Given six runs driving the finished, failed, interrupted and wound-down logical-turn exits, with a post-turn observer recording every report, when each run completes and its lane has drained, then each report's outcome is **identical** to the outcome carried by that turn's own close event on the recorded stream, compared value-to-value rather than through a rendered string; and when the outcome vocabulary is enumerated through the public surface, then its member set and each member's rendered form are exactly what they were at the merge base, `turn_events.go` and `failure.go` are byte-unchanged, and the turn-close constructor's signature and the failure-iff-aborted rule are unchanged. Cross-referenced to `R-HKS-004` / `S-HKS-010` and `R-HKS-010` / `S-HKS-024`.

## The no-`Completion` path (a REQUIRED observable, with the fix left to design)

`finalize()` is invoked at `loop.go:286` **before** the zero-finish-reason normalization at `loop.go:288-290` (read verbatim). A dispatch placed inside `finalize()` would therefore see a zero `ai.FinishReason` on the provider-closed-without-`Completion` path, and zero is not a vocabulary member (`finish_reason.go:42`). This spec pins the observable, not the remedy; `sdd-design` chooses between normalizing before `finalize()` and giving the dispatch an explicit zero case.

- **S-ATT-012** — Required observable, belonging to `R-ATT-002`. Given a provider stream that emits content and then closes **without** any `ai.Completion` event, when `Turn` runs, then a `turn_end` **is** emitted, its outcome is a member of the closed vocabulary (accepted by `NewTurnEnd`, neither the zero value nor `>= turnOutcomeLimit`), it is the outcome assigned to `FinishReasonStop` per `R-ATT-001`, the returned `finish` is `ai.FinishReasonStop`, and no `NewTurnEnd` validation violation occurs.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-ATT-001** | External-package verifiability: every scenario verifiable by `cd backend/agent && make test`; every behavioural test in `package agent_test` or another external package. |
| **NFR-ATT-002** | Determinism and race cleanliness: every test deterministic, hermetic, green under `-race`. |
| **NFR-ATT-003** | No new top-level Go dependency, **no new `EventKind`**, no `event_descriptor.go`/`event_registry_test.go` edit; scope-fence stays at 25 kinds. |
| **NFR-ATT-004** | `Turn`'s exported signature and its documented contract rows are unchanged; only the fatal path's returned `msg` value changes. |
| **NFR-ATT-005** | Review budget: single PR under pre-authorised `size:exception` against the 1000-line budget (forecast 900–1500 lines). |

## Explicit non-requirements

The following are **out of scope** and MUST NOT be implemented by this change:

- **Acting on retryability** — AG-15. The loop reports `Retryable()` through the typed surface and MUST NOT retry, back off, or route to a fallback provider. *(Still open at AG-13: the run driver ends a failed run typed and retries nothing, `R-RUN-011`.)* **CLOSED by AG-15**, and the constraint on the **loop** is unchanged and still binding: acting on retryability ships in the **harness**, owned by `agent-retry-failover` (`R-RTY-001`…`012`), and `Turn` itself still MUST NOT retry, back off, or route to a fallback provider. A retry is the harness calling `Turn` again; `Turn` remains a one-turn function with no knowledge that it is being retried, which is exactly why `R-ATT-008` stays true unamended.
- **Acting on pause** — AG-13. AG-11 makes pause visible as its own outcome and stops there; the loop MUST NOT attempt a resumption. **CLOSED by AG-13**: the resumption ships in the run driver, owned by `R-RUN-009`. The constraint on `Turn` is unchanged and still binding — **the loop still MUST NOT resume**. Resumption is the driver calling `Turn` again with a transcript that already contains the partial; `Turn` itself remains a one-turn function with no knowledge that it is resuming anything.
- **Pairing enforcement for turns that fail mid-tool-calls** — owned jointly by AG-11.2's typed path and AG-12's history boundary and orphan synthesis (`0003:1151`). This capability owns only the dispatch and the typed emission.
- **The `_ = sched.Schedule(...)` tool-result discard at `loop.go:265`** — a pre-existing gap immediately adjacent to this change's edit site. It MUST come out of AG-11 **byte-unchanged**: neither fixed nor further entrenched. **DISCHARGED by AG-13**, and the scope of the fix is stated exactly: on the **continuation path** the rejoin results are captured and committed to the run's transcript (`R-HIS-010`), so the discard no longer exists there. On the **nil-continuation path** the discard remains — a bare `Turn` invocation still has no transcript to commit to, and changing it would alter behavior no caller asked to change. The reservation is therefore discharged for the path AG-13 drives and left standing for the path it does not. *(Still standing at AG-16, which does not touch it: the cost emission reads the turn's usage, not its rejoin results.)*
- **A new `EventKind` for failure** — `failure.go:6-7` already settled it: failures ride the typed outcomes. *(Still true at AG-13: the run driver's failed run-close reuses the existing run outcome and the existing typed failure surface. Still true at AG-16: an aborted turn emits no cost event and no failure-carrying kind is added.)*
- **Multi-turn state, cancellation tree, cost events** — AG-12…AG-18. *(AG-13 ships the multi-turn **run**; state that outlives a run is AG-21's, the cancellation tree is AG-14's, cost aggregation is AG-16's.)* **The cost-events half is CLOSED by AG-16**, and the constraint on **this** capability is unchanged and still binding. `Turn` gains exactly one obligation, owned by `R-LSK-001`'s AG-16 amendment and not re-homed here: a turn that closes **non-aborted** emits a `cost_turn` inside its own bracket before `turn_end`. Every termination path this capability owns is **unchanged** — the mid-stream fatal path, the three pre-stream paths and the cancellation path all close aborted and emit **no** cost event, so no enumerated order in this capability is altered. `Turn`'s signature is unchanged (`NFR-ATT-004`, `R-ATT-007`), which is what made the emission expressible without touching this capability at all. Run-scoped aggregation is the harness's (`R-CST-004`), and a bare `Turn` caller aggregates nothing. **Multi-turn state beyond a run remains AG-21's** and is not closed here.

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active (`openspec/AGENTS.md`).

- Both leaves are behaviour, so every scenario is RED-first.
- The five bites (`S-TTB-001`…`S-TTB-005`) MUST each be RED-recorded with failing output in `apply-progress.md` BEFORE their partner scenario is GREEN.
- The substrate deviation of `R-ATT-009` MUST be recorded in `apply-progress.md` with the same explicitness AG-09 and AG-10 used.

## Acceptance criteria

1. Every `S-ATT-001`…`S-ATT-012` has recorded evidence; all five bites RED-recorded first.
2. `cd backend/agent && make test` (race-gated), `make lint` (after `golangci-lint cache clean`), `make build`, `make vuln-check` all green.
3. Every `ai.FinishReason` member maps to a distinct `TurnOutcome`; refusal, pause and finished are distinguishable by outcome value alone.
4. A mid-stream terminal error after partial content produces `turn_end(aborted)` + `run_end(failed)` carrying a `*agent.Failure` whose `Category()`, `Retryable()` and `PartialOutput()` are inspectable, plus the partial content on the returned `msg`.
5. `len(provider.Requests()) == 1` on every failing turn.
6. Both substrate filters widened by exact filename only and byte-in-sync; `R-LSK-004`, `R-APP-012` and `R-AEV-008` deltas shipped.
7. `loop.go:265` byte-unchanged; scope-fence still 25 kinds; `go.mod`/`go.sum` byte-unchanged.
8. Scenario count `4 charter → 12 spec + 5 bites = 17 total` stated identically in proposal, tasks, apply-progress and verify-report.

## Traceability

| Requirement | Charter node | Charter lines (`0003`) | Closes |
|---|---|---|---|
| `R-ATT-001` | AG-11.1 | `:1137-1141` | R-08 typed termination |
| `R-ATT-002` | AG-11.1 | `:1137-1141` | R-08 |
| `R-ATT-003` | AG-11.1 | `:1141` | exhaustiveness pin, 3rd instance |
| `R-ATT-004` | AG-11.1 | `:1143-1147`, `:2167-2168` | the AI-13 loop-termination defect |
| `R-ATT-005` | AG-11.2 | `:1158-1162`, `:2162`, `:2203` | R-05 invariant 4, R-18 seam 7 (`:2216`) |
| `R-ATT-006` | AG-11.2 | `:1161` | R-05 invariant 4, jointly with AG-04.3 |
| `R-ATT-007` | AG-11.2 | `:1161` | partial assistant content |
| `R-ATT-008` | AG-11.2 | `:1164-1167`, `:2204`, `:2213` | R-06 decide-retry, R-15 loop-never-retries |
| `R-ATT-009` | cross-cut | `proposal.md` deviation section | substrate rule amended, not broken |
