# `agent-retry-failover` Specification

> **Change**: `cachicamas-agent-retry-failover` · **AG-15** (Layer 2, Wave 3), `0003:1444-1525`
> **New capability.** Becomes `openspec/specs/agent-retry-failover/spec.md` at archive. IDs `R-RTY-0NN` / `S-RTY-0NN`; the allocated range is `R-RTY-001`…`R-RTY-012` and `S-RTY-001`…`S-RTY-015`, of which `S-RTY-010`, `S-RTY-011` and `S-RTY-012` are the three **bites**. No total is stated: a count goes silently false the moment a later milestone appends (`S-LSK-020`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && make test`.
> **Sources**: charter `0003:1444-1525`; [`../../proposal.md`](../../proposal.md); [`../../design.md`](../../design.md), whose eight architecture decisions (AD-1…AD-8) are **closed** and are not re-opened here; [`../../explore.md`](../../explore.md).
> **Ownership boundary**: this capability owns the *retry decision*, the attempt bound, the backoff and its timing seam, the failover seam, and the exhausted-retry terminal report. It does not own `Turn`'s per-path emission contract (`agent-loop-skeleton`), the run algorithm and its bracket rules (`agent-run-driver`), the turn's typed termination (`agent-turn-termination`), or the cancellation signals (`agent-cancellation-tree`); it amends each of those through its own delta in this change's `specs/` tree.
> **Every `file:line` below was opened against `main@bf482b0a` in this worktree during this phase.**

## Purpose

Layer 1 classifies; Layer 2 decides — and today Layer 2 decides nothing. Every non-cancellation `Turn` error routes unconditionally to `h.failRun(sink, stamper, runID, terr)` (`harness.go:469`), one line after the AG-14 cancellation carve-out (`harness.go:460-462`), and `R-RUN-011` states that as a requirement rather than an accident. Three consequences, all read in the shipped tree:

1. **A transient failure kills the run.** `(*ai.Failure).Retryable()` is populated by Layer 1 and read by nothing in `backend/agent/src/agent`.
2. **The terminal report lies about why the run failed.** `wrapHarnessFailure` hardcodes `Category: ai.FailureCategoryUnavailable` for every cause (`harness.go:250-254`), one event after the loop preserved the true category on `turn_end` by wrapping the raw `*ai.Failure` (`loop.go:387-388`).
3. **A retry is not even expressible on the stream.** `Turn` emits `turn_start` unconditionally (`loop.go:296-301`) and all three pre-stream failure paths — `buildLoopRequest` (`loop.go:304-308`), the pre-request hook (`loop.go:317-328`), `provider.Stream` (`loop.go:332-338`) — call `closeSink` and return with no `turn_end`, so a second `turn_start` would be rejected by `CheckStream`. That gap is closed by this change's `agent-loop-skeleton` delta, which this capability depends on and does not own.

## Coverage — the seven charter scenarios, each mapped

Every charter Gherkin leaf traces to at least one `S-RTY-0NN`. No leaf is reduced and no scenario is orphaned.

| # | Charter scenario | Lines | Owning requirement | Scenario(s) |
|---|---|---|---|---|
| 1 | AG-15.1 "retryable-with-no-output retries visibly" | `0003:1470-1474` | `R-RTY-002` (with `R-RTY-001` G4) | `S-RTY-002`, bite `S-RTY-011` |
| 2 | AG-15.1 "partial output forbids automatic retry" | `0003:1476-1479` | `R-RTY-003` (with `R-RTY-001` G3) | `S-RTY-003`, bite `S-RTY-010` |
| 3 | AG-15.1 "non-retryable surfaces immediately" | `0003:1481-1484` | `R-RTY-004` (with `R-RTY-001` G2) | `S-RTY-004` |
| 4 | AG-15.2 "retry-after wins and backoff waits on the context" | `0003:1494-1498` | `R-RTY-007` (retry-after), `R-RTY-008` (the wait) | `S-RTY-007`, `S-RTY-008` |
| 5 | AG-15.2 "the harness bound holds above any lower-layer retrying" | `0003:1500-1504` | `R-RTY-005` (bound), `R-RTY-009` (ceiling) | `S-RTY-005`, `S-RTY-009`, bite `S-RTY-012` |
| 6 | AG-15.3 "the retry path consults the failover seam before giving up" | `0003:1514-1517` | `R-RTY-010` | `S-RTY-013` |
| 7 | AG-15.3 "the none implementation changes nothing (pin)" | `0003:1519-1522` | `R-RTY-011` | `S-RTY-014` |

Cross-cut requirements carrying no charter leaf of their own: `R-RTY-001` (the predicate, which every AG-15.1 leaf reads through), `R-RTY-006` (the timing seam, which `NFR-RUN-002`/`NFR-CAN-002` force), `R-RTY-012` (the terminal report, proposal Decision 4).

## Fixture constraint, binding on every scenario below

`agenttest.Provider.Stream` fails before producing a channel in exactly three situations — a zero request (`fake_provider.go:75-77`), an already-cancelled context (`fake_provider.go:85-97`), and script exhaustion (`fake_provider.go:99-102`). **A `Script` therefore cannot express an arbitrary retryable pre-stream failure.** Every scenario that needs one MUST use a test-local provider wrapper that fails its first N calls with a scripted pre-stream `*ai.Failure`, captures its own requests, and delegates to an inner `agenttest.Provider` afterward — the `errorProvider` precedent (`loop_test.go:1408-1421`). No scenario in this spec may be written as if `agenttest` had been widened; `agenttest` is not modified by this change.

## Requirements

### R-RTY-001 — The retry decision is a pure, ordered-gate predicate over typed evidence

The decision whether to retry a failed turn MUST be a **pure function** of the turn's returned error, the attempt number and the attempt bound — no I/O, no clock read, no emission — so the whole gate table is table-driven-testable without driving a run.

The gates MUST be evaluated in the order below with **first match wins**. The ordering is itself the requirement: a table whose rows are all present but evaluated in another order is a different, and wrong, predicate.

| Gate | Condition | Verdict |
|---|---|---|
| **G0** | the run context's cause matches an `R-CAN-001` cancellation sentinel | **wind down** — never retried. Pre-existing and unchanged (`harness.go:460-462`) |
| **G1** | the error is not an `*ai.Failure` | **surface** — no typed evidence exists, so no evidence-driven decision is possible; fail closed |
| **G2** | `Retryable() == false` | **surface immediately**, regardless of delivery position |
| **G3** | `Retryable() == true` **and** `PartialOutput() == true` | **surface** with its partial content; the run ends failed |
| **G4** | `Retryable() == true`, `PartialOutput() == false`, attempts so far `< H` | **retry** after the backoff wait |
| **G5** | `Retryable() == true`, `PartialOutput() == false`, attempts so far `== H` | **consult the failover seam**; on decline, surface the exhausted-retry report |

**G0 MUST stay ordered ahead of G1–G5** and MUST keep its existing shape and site; this change adds no second cancellation check. A cancelled turn is never retried — `R-RUN-011`'s carve-out extends verbatim (`agent-run-driver/spec.md:248`), and the ordering, not a separate prohibition, is what enforces it.

**G3 fires *after* G2 has already said "retryable", and that is the point.** The naive "retry if retryable" predicate is exactly what the charter forbids (`0003:1479`).

**`Delivery()` MUST NOT be a gate.** It is read for the *report* (`R-RTY-012`), never for the decision: Layer 1 states that delivery alone cannot distinguish the two mid-stream shapes (`provider_failure.go:522-527`), while `PartialOutput()` answers the charter's actual question. A consequence, stated so it is not discovered later: a **mid-stream** failure that emitted no output is retryable under G4, and that is stream-legal because the mid-stream path already emits `turn_end(Aborted)` and closes the bracket (`loop.go:398-400`).

The predicate governs only the error a `Turn` invocation returns. Plain errors raised by the harness's own transcript writes (`harness.go:423-425`, `:472-474`) MUST keep their existing direct routing to the failure path and MUST NOT enter the predicate.

#### Scenarios

- **S-RTY-001** — Given the predicate as a directly callable pure function and a table of cases covering every gate, when each case is evaluated, then each yields the verdict its gate names; a case that is **retryable *and* carries partial output** yields the surface verdict, proving G3 is reached and G4 is not; a case that is **non-retryable *and* carries partial output** yields the surface verdict at G2, proving G2 is evaluated before G3; a plain Go error yields the surface verdict at G1 without any type assertion panicking; and the same evidence evaluated at attempts `< H` and at attempts `== H` yields the retry verdict and the failover-consult verdict respectively.
- **S-RTY-010** — **(bite)** RED-first. Given a scratch tree with gate G3 deleted from the predicate, when `S-RTY-003` runs, then it FAILS reporting that the provider recorded more than one request — the partial-output failure was retried, which is the exact defect G8 exists to prevent. RED-recorded BEFORE `S-RTY-003` is GREEN, then reverted.

### R-RTY-002 — A retryable pre-output failure retries over a byte-identical transcript, and every attempt is visible

When G4 fires, the harness MUST re-invoke `Turn` for the **same logical turn** over a transcript that is **byte-identical** to the one the failed attempt was built from. Byte-identity MUST be **proved by the recorded requests**, not assumed: every request the provider captured for that logical turn MUST compare equal as `ai.Request` values.

Nothing may be written to the transcript between two attempts of one logical turn. In particular the failed attempt MUST leave no partial entry behind, the steering drain MUST NOT run between two attempts of one logical turn (a steered message belongs to the *next* logical turn, `R-RUN-008`), and the harness MUST NOT call `CloseTurn` on a failed attempt.

Each attempt MUST be **visible on the stream as its own turn bracket**: its own `turn_start`/`turn_end` pair, its own freshly minted turn identity, on the one contiguous 1-based lane the run's single shared `LaneStamper` owns (`R-RUN-003`). Attempts are visible, not silent.

**No new `EventKind` is introduced.** A retried attempt is distinguished from a new turn by transcript identity — a retry's request transcript equals the failed attempt's, which is exactly what this requirement already guarantees — not by a labelled event. `CheckStream` MUST accept the recorded multi-attempt stream **unmodified**, with `stream_check.go` byte-unchanged (`agent-run-driver/spec.md:125`).

#### Scenarios

- **S-RTY-002** — **Charter AG-15.1 scenario 1.** Given a provider wrapper (per the fixture constraint above) scripted to fail its first `H − 1` calls with a retryable pre-stream `*ai.Failure` reporting no partial output, backed by an inner script that then completes the turn normally, and a recording sleep function that returns nil without waiting, when the run is driven, then the wrapper recorded exactly `H` requests; every recorded request compares equal to every other as an `ai.Request`; the emitted stream carries `H` turn brackets for that logical turn — the first `H − 1` closing with the aborted turn outcome and a non-nil `*Failure`, the last closing normally — with **pairwise distinct turn identities** on one contiguous 1-based lane inside a single run bracket; the run-close carries the completed run outcome; and `CheckStream` accepts the recorded stream unmodified.
- **S-RTY-011** — **(bite)** RED-first. Given a scratch tree in which the pre-stream `turn_end(Aborted)` emission of this change's `agent-loop-skeleton` delta is reverted, when `S-RTY-002` runs, then `CheckStream` REJECTS the recorded stream with `ai.ErrMisplaced` at the second `turn_start` (`stream_check.go:141-143`) and the scenario FAILS — proving the stream-validity assertion is load-bearing rather than incidental. RED-recorded BEFORE `S-RTY-002` is GREEN, then reverted.

### R-RTY-003 — A failure after emitted output is surfaced, never retried

**The G8 sentence, now at the harness.** When a turn fails after a normalized output event has already been emitted — `PartialOutput() == true` — the harness MUST NOT retry, MUST NOT back off, and MUST NOT route to a fallback provider, **even when the failure reports itself retryable**. The typed failure MUST surface on the stream carrying its partial content, and the run MUST end failed.

The observable that proves it is a **count, and the count is the contract**: the provider MUST have recorded exactly one request for that logical turn. Any second request is a duplicated observable output, which is the defect this requirement exists to prevent.

#### Scenarios

- **S-RTY-003** — **Charter AG-15.1 scenario 2.** Given a provider scripted to deliver text content and then a terminal mid-stream failure whose report is marked **retryable** and whose output-preceded discriminator is true, with a further script queued so that a retry would be observable rather than an error, when the run is driven, then `len(provider.Requests()) == 1`; the turn-close carries the aborted turn outcome and a `*Failure` reporting `PartialOutput() == true` whose message content is the content delivered before the failure; the run-close carries the failed run outcome; `Run` returns a non-nil error; and the queued second script was never consumed.

### R-RTY-004 — A non-retryable failure surfaces immediately, regardless of position

When the typed evidence reports `Retryable() == false`, the harness MUST surface the failure immediately: no retry, no backoff, no failover consult, and **no wait of any kind**. This MUST hold identically for a failure delivered **before** the stream opened and for one delivered **after** output was emitted — the position is irrelevant once G2 has matched.

The predicate MUST read the retryability flag, never a category allowlist: `Retryable` is a caller-set classification field independent of `Category` (`provider_failure.go:282-285`), and a category-based reimplementation would drift from Layer 1's classification the first time a category is added.

#### Scenarios

- **S-RTY-004** — **Charter AG-15.1 scenario 3.** Given two runs — one whose provider wrapper fails pre-stream with a non-retryable `*ai.Failure`, one whose script fails mid-stream after emitted output with a non-retryable `*ai.Failure` — each with a further script available so a retry would be observable, and each driven with a recording sleep function, when each run is driven, then each recorded exactly one request, each run-close carries the failed run outcome, `Run` returns a non-nil error in both, and the recording sleep function was called **zero** times in both — no backoff wait was even entered.

### R-RTY-005 — The attempt bound `H` counts total attempts, and the convention is stated wherever a number is

The harness MUST bound the attempts it makes for one logical turn. The bound `H` MUST count **total attempts** — total `Turn` invocations for one logical turn — and MUST NOT count retries-after-the-first.

**This diverges consciously from Layer 1's convention** (`retry.go:15-18` states its own budget as retries *after* the initial request) for three recorded reasons: the charter's observable is the provider's call count, which is a total; the composed ceiling of `R-RTY-009` multiplies two totals, so a budget convention here would corrupt its arithmetic; and re-importing "3 means 4" at a second layer doubles an off-by-one trap Layer 1's own documentation already has to disclaim.

**The divergence MUST be stated adjacent to every number.** Wherever this capability's production documentation states `H`, states the default, or states the composed ceiling, it MUST also state that the number counts total attempts, so no reader has to guess whether 3 means 3 or 4. A number written without its convention is a defect under this requirement even when the number is right.

The default MUST be `H = 3` — three `Turn` invocations for one logical turn, i.e. one initial attempt plus at most two retries — carried as a **zero-default exported field** on the caller-owned `Harness` value, where a non-positive value selects the default (the `Scheduler.WindDownBound` idiom, `scheduler.go:611-613`). G4 MUST fire while attempts so far are `< H`; G5 MUST fire at `== H`.

#### Scenarios

- **S-RTY-005** — **Charter AG-15.2 scenario 2, first Then.** Given a provider wrapper scripted to fail retryably pre-stream on **more** calls than `H` — the charter's "fail pre-stream forever", satisfied by strictly exceeding the bound, which is a stronger proof than literal infinity — backed by an inner provider that must never be reached, and a recording sleep function, when the run is driven to its terminal, then the wrapper recorded **exactly `H`** requests (here the count is the contract: it is the only observable that proves the harness's own bound held); the inner provider recorded none; the run-close carries the failed run outcome; and when the same fixture is re-run on a harness whose attempt-bound field is set to a different positive value `H'`, then the wrapper recorded exactly `H'` requests — proving the bound is the field's value and not a hardcoded constant.

### R-RTY-006 — Backoff timing is injected, and no test may reach a wall clock

The backoff MUST take its clock and its wait through an **injected timing seam** carried as a zero-default value field on the caller-owned `Harness`, whose zero value resolves to production defaults so an unconfigured harness behaves correctly without configuration.

The seam MUST expose, at minimum, a now-function (which seeds jitter), a wait-function taking a context and a duration, a base delay, a maximum delay, and a jitter seed. Its shape MUST mirror Layer 1's configuration struct (`retry.go:25-45`) and MUST be **freshly declared in the `agent` package**: `backend/agent/src/ai/internal/retry` is `internal` to `backend/agent/src/ai`, and `backend/agent/src/agent` is a sibling, so Go's internal-visibility rule forbids the import at compile time. **No file under `backend/agent/src/agent` may import that package.**

Computed backoff MUST grow exponentially with jitter and MUST be clamped to the maximum delay. Every scenario in this capability MUST synchronize through the injected wait-function, channel reads or `agenttest.Gate`; **no test may synchronize by sleep, timeout or wall-clock ordering** (`NFR-RUN-002`, `NFR-CAN-002`). A test that needs a real sleep to pass is a design failure under this requirement, not a flaky test.

#### Scenarios

- **S-RTY-006** — Given a harness whose timing seam is left at its zero value, when the delay computation is exercised directly across successive attempt numbers with a fixed jitter seed, then the delays are reproducible for that seed, strictly non-decreasing in expectation as the attempt number rises, and never exceed the configured maximum delay; and given the full test suite for this capability, when its sources are scanned, then no test file in it calls a wall-clock sleep and no production file added by this change imports the Layer 1 retry package.

### R-RTY-007 — A reported retry-after overrides computed backoff

When the typed evidence reports a retry-after value, the harness MUST wait **that** duration rather than the computed backoff, clamped to the configured maximum delay. Retry-after MUST be read presence-typed through `(*ai.Failure).RetryAfter()` (`provider_failure.go:471-476`), so *absent*, *reported zero* and *reported non-zero* stay three distinguishable readings; an absent value MUST fall back to the computed backoff, and a reported zero MUST NOT be treated as absent.

#### Scenarios

- **S-RTY-007** — **Charter AG-15.2 scenario 1, first Then.** Given a provider wrapper failing retryably pre-stream with a failure carrying a **non-zero** retry-after value, and a recording wait-function capturing every duration requested, when the run is driven, then the first duration the wait-function was asked for equals the reported retry-after value and differs from the computed backoff the same attempt number would have produced; and given the identical fixture whose failure carries **no** retry-after, when the run is driven, then the requested duration equals the computed backoff for that attempt number.

### R-RTY-008 — The backoff wait selects on the run context, and an aborted wait winds the run down

The backoff wait MUST be given the **run's own context** — the cancel-cause context the run derives at entry (`harness.go:418-420`, `:460-462` read it) — and never a background context and never a bare timer with no cancellation arm. It MUST return promptly when that context is cancelled.

When the wait returns because of cancellation, the harness MUST re-consult the run context's cause: a match against an `R-CAN-001` sentinel MUST route to the wind-down path exactly as a signal observed at an iteration boundary does, producing the interrupted or shutdown run outcome and, per `RunEnd.validate`'s failure-iff-`Failed` rule, **no** `*Failure`. A bare cancellation of the caller's own context, matching neither sentinel, MUST fall through to the failure surface — parity with the existing scope rule (`R-CAN-001`, scope line). **No new cancellation vocabulary is introduced**: the interrupted-during-backoff case is the existing wind-down, reached from one more place.

#### Scenarios

- **S-RTY-008** — **Charter AG-15.2 scenario 1, second Then.** Given a run whose provider wrapper fails retryably pre-stream and whose injected wait-function signals on a channel that it has entered the wait and then blocks until its context is done, when the test reads that signal and fires the interrupt, then the wait returns without the test advancing any clock, the run-close carries the **interrupted** run outcome with a **nil** `*Failure`, no run-close carrying the failed outcome appears anywhere on the stream, the provider recorded no request after the signal, `Run` returns an error satisfying `errors.Is` against the interrupt sentinel, and `CheckStream` accepts the stream unmodified; and given the production default wait-function driven directly with an already-cancelled context, when it is called, then it returns immediately with a non-nil error and creates no observable delay.

### R-RTY-009 — The composed ceiling is stated in this layer's words and its divergence is a test failure

This capability's own production documentation MUST state the composed worst-case ceiling for one logical turn as **harness attempts × Layer 1 attempts**, MUST state the concrete value it evaluates to at the shipped defaults — `H = 3` total attempts × `4` wire requests per logical provider call = **12** wire requests — MUST state that `H` counts total attempts (`R-RTY-005`), and MUST identify `backend/agent/src/ai/internal/retry`'s package documentation as the Layer 1 source of the multiplier.

This closes the Layer 2 half of `R-AIS-044 / S-2` (`ai-stream-lifecycle/spec.md:646-650`), whose third clause is the operative one: **a divergence between the two layers' wording MUST be observable as a test failure**, not as a stale comment. A test MUST therefore assert both sides' wording. The Layer 1 file is **read, never edited** — every file under `backend/agent/src/ai/` stays byte-unchanged (`R-RUN-012`), and reading a file is not importing a package, so the internal-visibility rule of `R-RTY-006` is not engaged.

#### Scenarios

- **S-RTY-009** — **Charter AG-15.2 scenario 2, second Then.** Given the Layer 1 retry helper's package documentation read as a file at a repository-relative path and this capability's own policy documentation, when the cross-layer test runs, then the Layer 1 sentences it cites are present verbatim in the Layer 1 file, this capability's documentation states the composed formula in the same wording together with the value the shipped defaults evaluate to and the total-attempts convention, and the Layer 1 file is byte-unchanged against the merge base.
- **S-RTY-012** — **(bite)** RED-first. Given a scratch tree in which one of the cited Layer 1 sentences is perturbed **in the test's own expectation set** (`wantLayer1RetryDocSentences`), when `S-RTY-009` runs, then it FAILS reporting the divergence — proving the cross-layer contract is enforced by a test rather than asserted by a comment. RED-recorded BEFORE `S-RTY-009` is GREEN, then reverted.

  **Why the expectation operand and not the Layer 1 file.** `R-RUN-012` forbids editing anything under `backend/agent/src/ai/**`, transient and reverted mutations included, so this bite cannot perturb `ai/internal/retry/doc.go` even temporarily. The proving power is unchanged: the check is `strings.Contains(fileContent, want)`, symmetric in its two operands, so perturbing either side breaks the same comparison and yields the same observable failure. Note the resulting guarantee is stronger than a revert — the Layer 1 file is byte-unchanged because it is never written at all.

### R-RTY-010 — The failover seam is a named injection point consulted exactly once at exhaustion

The package MUST expose a **failover policy interface** with a single consult method returning a **typed verdict value**, plus a typed prompt carrying, at minimum, the number of attempts made and the final typed failure. It MUST be carried as a **nil-default field on the caller-owned `Harness`**, not on `TurnOptions`: exhaustion is a run-driver concept — `Turn` knows nothing of attempts — and a seam belongs where its consumer is.

The seam MUST be consulted **exactly once** per logical turn, at G5, after the attempt bound is reached and **before** the terminal report is emitted. A nil policy MUST never be called and MUST behave as a decline.

**Declining MUST be the verdict's zero value.** v1 ships **no acceptance field at all**, so an accepting verdict is unconstructible by any implementation — a stronger guarantee than an ignored acceptance flag, and the same unconstructible-cell posture Layer 1 uses (`provider_failure.go:604-609`). A later version adds route and re-budget fields non-breakingly, and every existing implementation returning the zero verdict keeps compiling as a decliner.

The package MUST ship **one concrete declining implementation** — an installable value, which a nil field is not (the `NoOpPermissionPolicy` shape, `S-LSK-010`). The interface's documentation MUST state what a real, post-v1 implementation is obliged to handle: **re-counting the context budget and restarting the cache prefix** (`0003:1517`). That obligation is documentation, not code: the implementation behind the seam stays deferred (`AGS-D`, `agent-v1-scope/spec.md:128,133`).

#### Scenarios

- **S-RTY-013** — **Charter AG-15.3 scenario 1.** Given a recording failover policy installed on the harness and a provider wrapper failing retryably pre-stream on more calls than `H`, when the run is driven to its terminal, then the policy's consult method was called **exactly once** (here the count is the contract: "consulted once when retries exhaust" is the charter's own claim), it was called with an attempt count equal to `H` and with the final attempt's typed failure, its returned zero verdict was read as a decline, the run-close carries the failed run outcome, and the policy was not consulted at all in a second run whose failure is non-retryable and therefore surfaces at G2; and given the shipped declining implementation and the interface's documentation, when the documentation is read, then it names re-counting the context budget and restarting the cache prefix as the real implementation's obligations.

### R-RTY-011 — The seam's existence changes nothing (inertness pin)

A run driven with **no** failover policy installed and the identical run driven with the shipped declining implementation installed MUST be **indistinguishable to a consumer**: the same event kinds in the same order with the same outcomes and the same failure evidence, and the same returned error. The seam adds an injection point, never a behavior.

#### Scenarios

- **S-RTY-014** — **Charter AG-15.3 scenario 2.** Given one failing fixture defined once and driven twice — first on a harness whose failover field is nil, then on a harness carrying the shipped declining implementation — when both runs reach their terminals, then the two recorded event streams are equal event for event in kind, outcome, turn and run identity provenance, and attached failure evidence; the two returned errors are equal in type, in `errors.Is` behavior against every sentinel this capability's tests check, and in the failure category they carry; and both streams are accepted by `CheckStream` unmodified.

### R-RTY-012 — The exhausted-retry terminal report carries the true evidence, whole

When the run ends through the failure path with a cause that **is** an `*ai.Failure`, the run-close event's `*Failure` MUST preserve that evidence in full — category, retryability, retry-after, partial-output, delivery, every opaque provider-attribution field, and the cause chain — because the reported value MUST wrap the **identical** `*ai.Failure`, not a report reconstructed from it. Reconstruction is what loses fields; wrapping cannot.

The identity MUST be observable: unwrapping the run-close's failure MUST reach the same `*ai.Failure` value the final attempt failed with.

When the cause is **not** an `*ai.Failure`, the report MUST keep today's behavior byte-identically — the `Unavailable` category built by the existing helper (`harness.go:250-259`), which this change MUST NOT modify. The preservation MUST therefore arrive as a **conditionally routed sibling** beside that helper (the AG-14 precedent, `scheduler.go:1088-1099`), never as a rewrite of it.

Why the full extent and not merely the category: an exhausted-retry report is the only place a stream consumer learns *why* it failed (category), *whether re-prompting can help and when* (retryability, retry-after), *whether output was already rendered* — the same question G3 answers, and the one a consumer needs in order not to duplicate content on a manual re-prompt (partial output) — and *whether anything crossed the carrier at all* (delivery). Preserving delivery is legal here: `RunEnd.validate` checks only outcome vocabulary and the failure-iff-`Failed` rule (`run_events.go:161-172`) and never inspects delivery, and Layer 1's pre-stream rejection governs its own stream-terminal error event, which a run-close is not.

#### Scenarios

- **S-RTY-015** — **Proposal Decision 4.** Given a run exhausting its attempts against a provider wrapper whose final pre-stream failure reports a category other than unavailable, a true retryability flag, a retry-after value, no partial output, and pre-stream delivery, when the run-close event is read, then its `*Failure` reports that same category — not the unavailable category — the same retryability, the same retry-after, the same partial-output reading and the same delivery reading, and unwrapping it reaches the identical `*ai.Failure` value the final attempt failed with; and given a run failed by a **plain Go error** cause, when its run-close is read, then it reports the unavailable category exactly as it does today, with the existing wrapping helper's source byte-unchanged.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-RTY-001** | External-package verifiability: every scenario above MUST be verifiable by `cd backend/agent && make test`, and every behavioral test MUST live in `package agent_test`. A behavior reachable only from inside the package is, for this spec, not reachable at all. The predicate's own table-driven test MAY exercise it through whatever surface `sdd-design` fixed, provided every behavioral claim above is also observable externally. |
| **NFR-RTY-002** | Determinism and race cleanliness: every test MUST be hermetic and MUST pass under `-race`. Synchronization MUST be by the injected wait-function, `agenttest.Gate`, channel reads and channel closes. **No test may synchronize by sleep, timeout or wall-clock ordering**, and this change introduces no legitimate use of real clock time at all. |
| **NFR-RTY-003** | Ambient authority and boundaries: production sources added by this change MUST NOT import `os`, process execution, `syscall` or legacy I/O; the ambient-authority guard and the import-boundary guard MUST pass with **zero** change. `time` is used only for durations. No file under `backend/agent/src/agent` may import `backend/agent/src/ai/internal/retry`. |
| **NFR-RTY-004** | Substrate: every file named by `R-LSK-004` MUST be byte-unchanged, `stream_check.go`, `turn_events.go`, `failure.go`, `event.go` and `run_events.go` included, as MUST `history.go`, every file under `backend/agent/src/ai/`, and `go.mod`/`go.sum`. The every-kind-constructible guard MUST pass at its committed kind count; this change registers none. |
| **NFR-RTY-005** | Coverage: line coverage on `backend/agent/src/agent/loop.go` MUST remain ≥ 80% under `make test`, including the new pre-stream emission branches (`R-LSK-005` carry). |
| **NFR-RTY-006** | Review budget: this change ships as a **single** pull request under a pre-authorised `size:exception` extended beyond the 1000-line budget. The pull-request description MUST state why the change does not fit the default budget. If `sdd-tasks` forecasts beyond that extension, the slicing boundary is the charter's own DAG — AG-15.1, then AG-15.2 and AG-15.3 as siblings (`0003:1459-1460`). |

## Explicit non-requirements — what this spec does NOT claim

Stated so that no test, guard or acceptance line is written as if AG-15 closes more than it does.

| Not claimed | Owner and why the deferral is safe |
|---|---|
| Wire-level retry mechanics below `provider.Stream` | **Layer 1's AI-35**, shipped. Charter "Out of scope" line (`0003:1454`). AG-15 consumes the documented multiplier; it does not re-implement, re-plumb or override it |
| Overriding Layer 1's wire attempt count from Layer 2 | **Not this milestone.** The call site passes a zero-value configuration and no caller plumbs an override; adding one would be a Layer 1 edit, forbidden by `R-RUN-012`. `R-RTY-009` states the ceiling as a fact about the default, not as a knob |
| The failover **implementation** — choosing a fallback route, re-budgeting tokens and prices, restarting the cache prefix | **Post-v1**, seam 8 (`0003:1454`). `agent-v1-scope`'s `AGS-D` already names AG-15.3 as the placeholder node. AG-15 ships the seam and its documented contract, nothing behind it. *(Still true at AG-16, which ships no route selection and no re-budgeting; the "re-budgeting tokens and prices" obligation stays documentation on the seam, and AG-16's token figures are an observation of spend, never a budget)* |
| Retry of a **cancelled** turn | **Never.** Enforced by G0's ordering in `R-RTY-001`, not by a separate prohibition; `R-RUN-011`'s carve-out extends verbatim. *(Still true at AG-16)* |
| Retry **after** emitted output | **Never.** `R-RTY-003`, the G8 sentence at the harness. *(Still true at AG-16)* |
| `RetryAfter()` on the stream-observable `agent.Failure` envelope | **Deliberately not added (proposal Decision 5, D5).** The harness's decision path reads the raw `*ai.Failure` before any Layer 2 wrapping (`loop.go:411`, `:414`, `:338`, `:327`), so no scenario in this spec exercises the wrapper accessor. Adding it would require editing `failure.go`, `R-LSK-004` substrate released **for AG-11 only** (`agent-loop-skeleton/spec.md:86`), for a surface AG-15 does not use. **The omission is recorded here rather than left to be rediscovered**: a stream consumer holding only an `*agent.Failure` must reach retry-after through `Unwrap()`, and a later milestone that needs it stream-side takes its own recorded release. *(Still not added at AG-16, which edits `failure.go` not at all)* |
| Cost accounting for retried attempts | **AG-16, parallel. CLOSED by AG-16 — and closed by dissolving the question rather than by picking a reading.** The charter's parenthetical "a retried attempt's tokens are real spend" (`0003:1535`, `:1552`) admits two readings, and only one is buildable. **A *failed* attempt's own token spend is not captured, and cannot be**: `ai.Failure` carries nine fields and **no usage** (`provider_failure.go:320-330`), so no data source exists on any failure path, and no Layer 1 edit is in scope for Layer 2 (`R-RUN-012`). Compounding it, gate G3 only ever retries when `PartialOutput() == false` (`retry_policy.go:111-126`), so a retryable failure by construction emitted nothing. **What AG-16 guarantees instead is stronger than the alternative reading and needs no retry-awareness at all**: cumulative is defined as **the sum over every `cost_turn` event emitted within the run bracket** — not "the sum over the turns that succeeded". Because `R-RTY-002` already requires each attempt to be its own `Turn` invocation with its own visible turn bracket and its own freshly minted identity (`harness.go:502-521`), any attempt that reaches a `Completion` emits its own `cost_turn` and is therefore counted, **with no special retry handling anywhere in the accumulator**. The framing is also forward-compatible: if Layer 1 ever attaches usage to a failure, that attempt begins emitting a `cost_turn` and the rule already includes it — no spec change, no accumulator change. **This is a charter reinterpretation and is recorded as one** in doc 0003's AG-16 reconciliation note, following the AG-06 precedent (`0003:613`). Owned by `agent-cost-events` (`R-CST-004` / `S-CST-008`, with bite `S-CST-021` proving the sum is enforced rather than assumed). **Not closed here**: a distinct figure for a failed attempt's own spend, which stays unbuildable rather than deferred |
| Subagent-scoped retry | **AG-19.** No subagent tool ships in v1 (`0003:1794`). *(Still true at AG-16; parent-scoped cost aggregation over delegated runs is likewise AG-19's, `0003:1533`)* |
| A new `EventKind`, a new `TurnOutcome`, or a new exported `History` method | **Not this milestone**, and forbidden under this change. *(Still true at AG-16, which registers none: AG-06 minted `cost_turn` and `cost_session`, and AG-16 only emits them)* |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2.** Layer 1 is consumed, never edited (`R-RUN-012`). `ai/internal/retry`'s package documentation is **read** by `S-RTY-009`, never written. *(Still true at AG-16, which reads `ai.Usage` and `ai.TokenCount` and writes nothing under that tree)* |
| Widening `agenttest` | **Not this milestone.** The pre-stream-failing fixture is test-local (the fixture constraint above); `agenttest` is byte-unchanged. *(Still true at AG-16, which reuses the same test-local `errorProvider` wrapper precedent for its retry-bearing cumulative scenario and leaves `agenttest` byte-unchanged)* |

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active.

- All three charter leaves are behavior, so every scenario is RED-first.
- The three bites `S-RTY-010`, `S-RTY-011` and `S-RTY-012` MUST each be RED-recorded with failing output in `apply-progress.md` **before** their partner scenario is GREEN, then reverted.
- `R-RTY-003` closes **only on `S-RTY-010`**: without the bite, a passing partial-output scenario proves nothing about whether G3 exists.
- `R-RTY-002` closes **only on `S-RTY-011`**: the pre-stream emission is a companion delta, and a green `CheckStream` assertion with the emission already present does not prove the assertion would catch its absence.
- `R-RTY-009` closes **only on `S-RTY-012`**: the promoted `R-AIS-044 / S-2` requires divergence to be observable as a test failure, and a comment cannot satisfy it.
- `S-RUN-100` (`harness_test.go:1870-1919`) MUST pass with its source **byte-unchanged**, and for the strongest reason rather than incidentally: its scripted failure leaves the retryability flag unset, so G2 catches it. `sdd-verify` MUST re-check that against the shipped tree rather than trusting this sentence.

## Acceptance criteria

1. Every `S-RTY-001`…`S-RTY-015` has recorded evidence; all three bites are RED-recorded before their GREEN.
2. All **seven** charter Gherkin scenarios (`0003:1470-1474`, `:1476-1479`, `:1481-1484`, `:1494-1498`, `:1500-1504`, `:1514-1517`, `:1519-1522`) are mapped in the Coverage table and closed; none is reduced.
3. `cd backend/agent && make test` green under `-race`; `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check` all clean (`vuln-check` is **not** in `make all`).
4. `CheckStream` accepts the multi-attempt run stream **unmodified**, with `stream_check.go` byte-unchanged.
5. Every AG-13 and AG-14 harness, cancellation and loop test passes with its source **byte-unchanged**; any exception is an enumerated, signed-off amendment recorded in `apply-progress.md`.
6. Both substrate filters carry an identical exact-filename entry set, one entry per file this change introduces, with no wildcard, prefix or directory pattern (`R-LSK-004` delta).
7. All six spec deltas beside this file are written, and every line they cite is re-read against the shipped change by `sdd-verify`.
8. doc 0003's AG-15 checklist is ticked, R-15/G8 back-annotated, and the milestone counter bumped.

## Traceability

| Requirement | Charter node | Primary evidence |
|---|---|---|
| `R-RTY-001` | AG-15.1 (all three) | pure-predicate table + G3 bite |
| `R-RTY-002` | AG-15.1 sc.1 | `H` identical requests, `H` visible turn brackets, `CheckStream` clean |
| `R-RTY-003` | AG-15.1 sc.2 | exactly one request; partial content on the typed failure |
| `R-RTY-004` | AG-15.1 sc.3 | one request in both positions; zero waits |
| `R-RTY-005` | AG-15.2 sc.2 | provider count equals `H` exactly, and tracks the field |
| `R-RTY-006` | AG-15.2 sc.1 | injected timing; no wall clock anywhere |
| `R-RTY-007` | AG-15.2 sc.1 | requested duration equals retry-after, not computed backoff |
| `R-RTY-008` | AG-15.2 sc.1 | interrupt during backoff → interrupted run-close, nil failure |
| `R-RTY-009` | AG-15.2 sc.2 | cross-layer wording test + perturbation bite |
| `R-RTY-010` | AG-15.3 sc.1 | consulted once at exhaustion with `H` and the final failure |
| `R-RTY-011` | AG-15.3 sc.2 | nil vs declining implementation: identical streams |
| `R-RTY-012` | cross-cut (Decision 4) | true category and pointer identity on the run-close |
