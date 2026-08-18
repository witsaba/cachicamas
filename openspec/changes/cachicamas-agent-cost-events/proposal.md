# Proposal: AG-16 — Emit cost and usage events

> **Change**: `cachicamas-agent-cost-events` · **Milestone**: AG-16 (Layer 2 Wave 3, milestone 16 of 24; doc 0003 lines 1527–1560)
> **Branch**: `feat/agent-layer2-wave3-ag16` · base `main@09bb30e1` · **Worktree**: `cachicamas-worktrees/ag16`
> **Artifact store**: hybrid (Engram + filesystem) · **Delivery**: single PR, `size:exception` pre-authorized against the 1000-line budget
> **TDD**: strict, RED-first (`cd backend/agent && make test`)
> **Closes**: **G10**'s Layer 2 half (R-16). Layer 1 already counts; Layer 2 reports.
> **Depends on**: AG-06, AG-13, AG-15 (all archived) · **Parallel with**: AG-14 (merged)
> **Exploration**: `explore.md` · Engram `sdd/cachicamas-agent-cost-events/explore`

---

## Intent

AG-06.2 built the cost vocabulary. Nothing emits it. `CostTurn` and `CostSession` are constructible, registered and validated (`cost_events.go:162-165`, `:227-230`), and **zero production call sites exist** — the whole `agent` package never calls `completion.Usage()` even once. Layer 1 counts tokens into `ai.Usage` (`usage.go:111-149`); Layer 2 throws them away at `turnAccumulator.translate`'s Completion case, which reads `FinishReason()` and drops the rest (`loop.go:923-927`).

Three consequences, all verified in the shipped tree:

1. **A run's token spend is unobservable.** The only usage carrier at the Layer 1→Layer 2 boundary is `Completion.Usage()` (`completion.go:92`), registered `CardinalityAtMostOne, Terminal: true` (`ai/event.go:167-170`). It reaches `Turn`, and `Turn` discards it.
2. **Two shipped specs record this as an open debt against AG-16 by name** — `agent-run-driver/spec.md:326` ("Cost aggregation across turns | **AG-16**") and `agent-retry-failover/spec.md:219` ("Cost accounting for retried attempts | **AG-16**, parallel"). Neither carries a "CLOSED by" annotation. This change is what closes both.
3. **The charter's own acceptance text outruns the shipped substrate in two places.** Absence cannot be expressed by `CostFigures`, and the estimate/final axis it describes does not exist per-turn. Decisions 1 and 4 settle both, and Decision 5 records the reconciliation in doc 0003 rather than leaving the charter silently contradicted.

**What ticks**: `R-APE-004`/`R-APE-005` gain their first emitter; `agent-run-driver:326` and `agent-retry-failover:219` both close; G10's Layer 2 half lands.

---

## Scope

### In

- **AG-16.1** — a `cost_turn` event emitted for **every** turn that reaches a `Completion`, carrying the five token figures mapped from `ai.Usage`, labelled `Final`, with per-figure absence preserved. See Decision 1.
- **AG-16.1** — the usage plumbing seam: `turnAccumulator` captures `completion.Usage()` and `finalize()` emits `cost_turn` inside the turn bracket, before `turn_end`. `Turn`'s exported signature is **unchanged**. See Decision 2.
- **AG-16.1** — run-scoped cumulative accounting in `Harness.Run`, summing **every** emitted `cost_turn` including those of retried logical turns. See Decision 3.
- **AG-16.1** — the estimate/final protocol at run scope: intermediate `cost_session` events labelled `Estimate`, the run-terminal `cost_session` labelled `Final`. See Decision 4.
- **A presence discriminator** on `CostTurn`/`CostSession` — beside `CostFigures`, never inside it, so `S-APE-083`'s reflection pin stays byte-green. See Decision 1.
- **A doc 0003 reconciliation note** on the AG-16 charter, recording Decisions 3 and 4 as executed reconcile-or-flag duties. See Decision 5.
- Substrate-guard filter widening in `loop_test.go` (`:831-871`) and `loop_hook_test.go` (`:907-943`) by **exact filename suffix**, both filters byte-in-sync — the AG-11/AG-13/AG-14/AG-15 discipline.

### Out — deferred, with the owner named

| Deferred | Owner and why deferral is safe |
|---|---|
| **Money, currency, price, any monetary field** | **Layer 3's CO-18.** Charter out-of-scope (`0003:1537`); `R-APE-004` states it as a requirement (`agent-protocol-events/spec.md:56`); `S-APE-083` enforces it mechanically by forbidden-substring scan (`cost_events_test.go:207-224`). AG-16 **strengthens** this pin by adding zero fields to `CostFigures`. |
| **Delegated/subagent cost aggregation into a parent run** | **AG-19**, named by the charter's own Goal line (`0003:1533`, "the aggregation half lands with AG-19"). No subagent tool ships in v1. |
| **Mid-stream incremental cost display** | **Frontend / out of scope** (`0003:1537`). The charter itself rules that "the estimate-labelled event at minimum covers it" — which is load-bearing evidence for Decision 4, not merely a deferral. |
| **A failed attempt's own token spend as a distinct figure** | **Unbuildable, not deferred by choice.** `ai.Failure` carries nine fields and no usage (`provider_failure.go:320-330`). See Decision 3 and Decision 5. |
| **Any new `EventKind`** | **Never in AG-16.** AG-06 minted `cost_turn`/`cost_session`; AG-16 emits them. The scope-fence stays at its committed kind count, holding the `R-RUN-012`/`R-LSK-004` posture AG-13, AG-14 and AG-15 each held. |
| **Any edit under `backend/agent/src/ai/**`** | **Never in Layer 2** (`R-RUN-012`). Layer 1 is consumed, never edited. |
| **Re-opening AG-15's retry predicate (gates G0–G5)** | **CLOSED by AG-15.** AG-16 reads the attempt loop's outputs; it does not touch `retryDecision` (`retry_policy.go:111-126`). |

---

## Decision 1 — how absence is represented **(DECIDED — a presence discriminator beside `CostFigures`, never inside it)**

**The gap, confirmed.** `ai.Usage` is five `TokenCount`s, each a `{count int64; present bool}` pair (`usage.go:44-47`); `Tokens(0)` is a *reported nought*, explicitly distinguishable from absence. `CostFigures` is five plain `uint64` fields (`cost_events.go:137-157`). Converting one to the other **loses exactly the distinction the acceptance criterion demands** — "absence is reported as absence, never invented zeros" (`0003:1535`, `:1547`).

**`CostFigures` itself is closed.** `S-APE-083` walks `reflect.TypeOf(agent.CostFigures{})` and asserts exactly five fields, that exact name order, and `uint64` for every one (`cost_events_test.go:199-234`). The production doc comment states the same constraint prospectively (`cost_events.go:133-136`). Verified by reading the test in full: **it inspects `CostFigures` and nothing else** — it never reflects over `CostTurn` or `CostSession`.

**Does "mirroring" oblige mirroring absence?** `R-APE-004` reads: "mirroring Layer 1's usage **taxonomy**" (`agent-protocol-events/spec.md:56`). Taxonomy is *which buckets exist* — the five-way split of input/output/cache-read/cache-write/reasoning. It is not a claim about presence semantics. So mirroring the taxonomy does **not** oblige putting presence inside `CostFigures`. But the acceptance criterion independently requires the absent-vs-zero distinction to be *observable on the event*. Both hold simultaneously only if presence travels **beside** the figures.

| Option | Verdict |
|---|---|
| **(a)** Add a presence field/mask to `CostFigures` | **Rejected.** Breaks `S-APE-083` by field count. Would require amending a mechanically-pinned scenario from another capability, for which this codebase has **no precedent**, in exchange for nothing option (c) does not give. |
| **(b)** A third `CostLabel` member (`CostLabelAbsent`) | **Rejected.** A category error: it overloads one enum with two orthogonal questions. A turn with absent usage is still a *final* figure, and Decision 4 needs the label axis free to mean estimate-vs-final. The two axes must not collide. |
| **(c)** Skip emitting `cost_turn` when usage is absent | **Rejected.** Scenario 1 says "**its** cost event carries the turn's five token figures … honoring absent-vs-zero" — an event **is** emitted. Absence-by-non-existence also breaks the cumulative sum's turn-count invariant. |
| **(d)** A presence discriminator on `CostTurn`/`CostSession`, `CostFigures` untouched | **RECOMMENDED.** Keeps `S-APE-083` green with `CostFigures` **byte-unchanged**; keeps the token-only pin *strengthened* rather than weakened; leaves the label axis free for Decision 4. `cost_events.go` is **not** in `NFR-APE-004`'s byte-unchanged substrate list (`agent-protocol-events/spec.md:118`, which names only `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, build files), so widening the payload wrapper is structurally permitted. |

**Granularity — the sharper half of this decision.** `ai.Usage`'s presence is **per field**, not per record. A whole-record `usageAbsent bool` would satisfy Scenario 1's literal Given ("one turn with usage absent") while still **inventing zeros** for a mixed record — e.g. `ai.Usage{Input: ai.Tokens(100)}`, where reasoning is absent and would be reported as `0`. That is the precise defect the acceptance criterion names. **Recommendation: per-figure presence**, exposed through an accessor returning a presence-typed reading that mirrors Layer 1's own house idiom `TokenCount.Count() (int64, bool)` (`usage.go:63-65`).

**`sdd-design` MUST close:** the discriminator's exact shape (five-bit mask, `[5]bool`, or a parallel presence struct) and its accessor surface; whether `NewCostTurn`/`NewCostSession` gain a parameter or a sibling constructor (the AG-14 `typedCancellationFailure` sibling precedent applies); and whether an `ai.Usage`→payload conversion helper is exported or package-private.

---

## Decision 2 — the usage plumbing seam **(DECIDED — emit from `finalize()`, intercept in the forwarder; `Turn`'s signature is untouched)**

`Turn(ctx, provider, system, transcript, opts, sink) (ai.Message, ai.FinishReason, error)` (`loop.go:248-255`) returns no usage, so `Harness.Run` cannot reach it at `harness.go:520`.

| Option | Verdict |
|---|---|
| **(a)** Widen `Turn` to return `ai.Usage` | **Rejected — it contradicts a promoted spec.** `agent-turn-termination` states "`Turn`'s **signature** MUST NOT change" (`spec.md:113`) and pins it non-functionally: `NFR-ATT-004` — "`Turn`'s exported signature and its documented contract rows are unchanged" (`:153`). That is shipped, promoted text from AG-11. `loop.go:11-13`'s package doc gives the same design intent independently ("so AG-13's later Harness can wrap it without changing the signature"). Option (a) buys nothing (b) does not. |
| **(b)** `finalize()` emits `cost_turn`; `Harness.Run`'s forwarder intercepts | **RECOMMENDED.** |

**Why (b) fits the substrate rather than fighting it:**

- `finalize()` (`loop.go:1001-1025`) is the *only* point per-turn usage is ever known, and it is already the turn's emission site for `turn_end`. It is reached from exactly two call sites (`loop.go:383`, `finishContinuationTurn` at `:525`), both immediately after `translate` observed a `Completion`.
- `cost_turn` is registered `PlacementTurn` (`event.go:329-331`), so it **must** land inside the turn bracket — before `turn_end` closes it. `finalize()` is exactly where that ordering is expressible.
- The forwarder already exists and stays single-writer: `for ev := range turnSink { sink <- ev }` (`harness.go:507-509`). Reading `ev.CostTurn()` on the way through is a pure read that changes no forwarding behavior.
- It matches the charter's own Deliverable wording — "cost emission in the run driver from Layer 1 usage" (`0003:1534`).
- **It keeps `NFR-ATT-004` true.** `agent-turn-termination` then needs only a back-annotation confirming the pin held, not an amendment.

Implementation shape: add `usage ai.Usage` to `turnAccumulator` (`loop.go:768-810`), populate it at the Completion case (`loop.go:923-927`) alongside the existing `finish`/`finishOk` assignment, and emit from `finalize()`. Cumulative state lives as a **local** in `Run`'s stack frame, not a `Harness` field — `Harness` is value-form and reused serially per `R-CAN-002`, and nothing else on it carries cross-run state.

**`sdd-design` MUST close:** the exact emission order inside `finalize()` relative to `turn_end` and the nil-path `run_end`, proven against `CheckStream` with `stream_check.go` **byte-unchanged**; behavior on the no-`Completion` path (`S-ATT-012`'s provider-closes-without-Completion case — no usage was ever reported, so this is a first-class absence case for Decision 1, not an error); and whether a standalone `Turn` caller (no harness) emitting `cost_turn` with nobody aggregating is acceptable (it is — `cost_session` is harness-scoped — but the spec must say so).

---

## Decision 3 — what "a retried attempt's tokens are real spend" means **(DECIDED — settled explicitly, because the criterion is otherwise untestable)**

The charter's parenthetical (`0003:1535`, `:1552`) admits two readings:

| Reading | Buildable? |
|---|---|
| **(1)** A **failed** attempt's own token spend must be captured and summed | **No.** `ai.Failure` has nine fields — `category, retryable, retryAfter, rawLabel, statusClass, requestID, cause, delivery, partialOutput` — and **no usage field** (`provider_failure.go:320-330`). No data source exists on any failure path. Compounding this, AG-15's G3 gate only ever retries when `PartialOutput() == false` (`retry_policy.go:111-126`), so a retryable failure by construction emitted no output. |
| **(2)** The **succeeding** attempt of a logical turn that needed retries must still contribute its real tokens — a naive accumulator must not drop or zero a turn that took 2+ attempts | **Yes**, entirely with today's substrate. |

**Decision: reading (2) — but framed so the question dissolves rather than being merely picked.**

State the requirement as: **cumulative is the sum over every `cost_turn` event emitted within the run bracket.** Not "the sum over turns that succeeded." This framing is strictly better than choosing a reading:

- It is **correct by construction** for retries today. Each attempt of a retried logical turn gets its own `Turn` invocation and its own `turn_start`/`turn_end` pair with a fresh `TurnID` on one contiguous lane (`harness.go:502-521`, `R-RTY-002`, `agent-retry-failover/spec.md:74`). Any attempt reaching a `Completion` emits its own `cost_turn` and is therefore counted, with no special retry-awareness in the accumulator at all.
- It is **future-proof**. If Layer 1 ever attaches usage to a failure, that attempt starts emitting a `cost_turn` and the cumulative rule already includes it — **no spec change, no accumulator change**. Reading (2) stated as "sum the successful attempts" would need rewriting; stated as "sum the emitted events" it does not.
- It makes the accumulator **untestable-by-omission impossible**: the falsifiable claim becomes an equality between the cumulative figure and the observed per-turn events on the same recorded stream.

**This is a charter reinterpretation and must be recorded as one** — the parenthetical implies failed-attempt spend is countable, and against `ai.Failure`'s shape it is not. See Decision 5.

**`sdd-design` MUST close:** whether an attempt that fails *after* reaching a `Completion` can occur at all (G3 says no); and the per-figure presence arithmetic — summing an absent figure with a present one (recommendation: absent is the additive identity for the running total, but the cumulative figure's own presence bit is set once **any** contributing turn reported that figure, so a run where nothing was ever reported still reports absence rather than a fabricated `0`).

---

## Decision 4 — the estimate/final axis **(DECIDED — run scope, with the charter's own out-of-scope line as evidence)**

**The literal per-turn reading is unsatisfiable.** Scenario 3's Given is "usage arriving only on the final stream update." That describes a **wire-level** phenomenon, and doc 0001 § 7 G10 states its rationale (`0001:719-732`): on adaptive-reasoning models the reasoning count arrives only on the last streamed usage update, so any mid-stream figure is structurally an estimate. `cost_events.go:96-103` inherits that wording verbatim.

**Layer 1 absorbs that phenomenon entirely before Layer 2 sees it.** `ai.Completion` is the sole usage carrier (`completion.go:92`), registered `CardinalityAtMostOne, Terminal: true` (`ai/event.go:167-170`); `openaicompat/stream_state.go:328` folds wire-chunk usage into that single terminal event. **Within one turn there is no earlier figure to label `Estimate`** — a `cost_turn` from `finalize()` always already holds the complete usage. Read literally at per-turn granularity, Scenario 3 asks for an event that cannot exist.

**The charter itself supplies the resolution.** Its out-of-scope line reads: "mid-stream incremental cost display (a frontend may accumulate deltas itself; **the estimate-labelled event at minimum covers it**)" (`0003:1537`). The charter therefore already assigns the `Estimate` label to a *running, incremental* figure — not to a per-stream partial. That is the run-scoped axis, stated by the charter one line above the scenario it appears to contradict.

**Decision:**

- `cost_turn` is **always** `CostLabelFinal`. Per-turn usage is complete whenever it is known.
- `cost_session` emitted as a running total after an intermediate turn is **`CostLabelEstimate`** — the run has not concluded and more tokens may follow.
- The `cost_session` emitted at the run's terminal, immediately before `run_end` (`harness.go:621-624`), is **`CostLabelFinal`** and corrects every earlier estimate.

Scenario 3's "**any** earlier figure was labelled estimate" reads as a **conditional**: where an earlier figure exists it is `Estimate`; in a single-turn run none exists and the sole `cost_session` is `Final`. Both `cost_turn` and `cost_session` are `CardinalityAny`, so the multi-emission shape is already legal.

**`sdd-design` MUST close:** whether `cost_events.go:96-103`'s doc comments are amended to state the Layer 2 axis (permitted — `cost_events.go` is not in `NFR-APE-004`'s byte-unchanged list) and whether `R-APE-005` needs a corresponding delta; and whether a `cost_session(Final)` is emitted on the **failure** and **cancellation** run-closes as well as the success close — `R-RUN-011`'s failure-path posture and AG-14's wind-down path both need an explicit answer, since tokens spent before a failure are still real spend.

---

## Decision 5 — the milestone-doc reconciliation is a deliverable, not a footnote

Decisions 3 and 4 both reinterpret charter text that, read literally against the shipped substrate, describes unbuildable behavior. This repo has an **established, executed precedent** for exactly this: the AG-06 charter carries "**Note — one v2 conflict, reconciled here (the AG-00 reconcile-or-flag duty, executed)**" (`0003:613`), which records a doc-0001-vs-ADR-0005 conflict, names the verdict, and states which side wins and why.

**AG-16 owes the same note**, covering both reconciliations:

1. **"A retried attempt's tokens are real spend"** — reconciled to reading (2), because `ai.Failure` carries no usage (`provider_failure.go:320-330`) and no Layer 1 change is in scope for Layer 2 (`R-RUN-012`). Cumulative is defined over emitted events, which is retry-inclusive by construction and forward-compatible if Layer 1 ever adds the field.
2. **"Usage arriving only on the final stream update"** — reconciled to the run-scoped axis, because Layer 1's decoder absorbs the multi-chunk phenomenon into one terminal `Completion` (`ai/event.go:167-170`), and because the charter's own out-of-scope line already assigns `Estimate` to the incremental running figure (`0003:1537`).

Without this note, doc 0003 would ship contradicting its own merged implementation — this repo's known "un-back-annotated merge" staleness shape.

---

## Capabilities

> Contract with `sdd-spec`. Existing names taken verbatim from `openspec/specs/`.

### New

- **`agent-cost-events`** — cost emission and accounting: per-turn emission from Layer 1 usage, per-figure absence preservation, run-scoped cumulative over emitted events, and the estimate/final label axis. IDs `R-CST-0NN` / `S-CST-0NN`, bites `S-CST-0NN`. **Prefix `CST` verified free**: zero `[RSN]-(CST|COS|CEV|CSE)-[0-9]` matches repo-wide. Becomes `openspec/specs/agent-cost-events/spec.md` at archive.

### Modified — deltas required

| Capability | What changes | Mandatory? |
|---|---|---|
| `agent-protocol-events` | **(1)** `R-APE-004` records the presence discriminator on `CostTurn`/`CostSession` as a **minimal widening** of AG-06's capability, with `CostFigures` byte-unchanged and `S-APE-083` green — the token-only pin is strengthened, never weakened. **(2)** `R-APE-005`'s label semantics gain the Layer 2 axis of Decision 4. | **Yes — blocking.** Decisions 1 and 4 both amend this capability's surface. |
| `agent-run-driver` | Cumulative accounting and the `cost_session` emission sites relative to `run_end`; `:326`'s deferred row ("Cost aggregation across turns — **AG-16**") back-annotated as closed. `R-RUN-003`'s bracket/lane rule restated as still-true with cost events on the lane. | **Yes.** |
| `agent-retry-failover` | `:219`'s deferred row ("Cost accounting for retried attempts — **AG-16**, parallel") back-annotated as closed by Decision 3, with the reading recorded. The predicate's gates G0–G5 stay **CLOSED and untouched**. | **Yes.** |
| `agent-loop-skeleton` | `R-LSK-001` gains `Turn`'s `cost_turn` emission obligation at `finalize()`; `R-LSK-004` records that AG-16 requests **no** substrate release, and its filter-widening rule extends to AG-16's new filenames, byte-in-sync across both filters. | **Yes.** |
| `agent-turn-termination` | Back-annotation only: `NFR-ATT-004` and `:113` confirmed **held, not amended** — Decision 2 leaves `Turn`'s signature unchanged. | Yes — annotation. |

---

## Approach

1. **Capture usage first.** Add `usage ai.Usage` to `turnAccumulator` and populate at `loop.go:923-927`. Inert on its own — no behavior change, no new event — so it lands green and de-risks everything after it.
2. **Convert with presence preserved.** A conversion from `ai.Usage` to `(CostFigures, presence)` that is total, pure and table-driven-testable in isolation: absent stays absent, `Tokens(0)` stays a reported nought, and the two are never conflated. This is the single highest-value unit test in the change.
3. **Emit `cost_turn` from `finalize()`**, inside the turn bracket, before `turn_end`. Verify against `CheckStream` with `stream_check.go` byte-unchanged.
4. **Accumulate in the forwarder.** `Harness.Run` reads `ev.CostTurn()` on the existing forwarding path (`harness.go:507-509`) into a run-local total. No `Harness` field, no second writer.
5. **Emit `cost_session(Estimate)`** after each intermediate turn and `cost_session(Final)` before `run_end`.
6. **Test the retry case with the `errorProvider` wrapper precedent** (`loop_test.go:1408-1421`), not a bare `agenttest.Provider` — the fixture constraint at `agent-retry-failover/spec.md:36` means a `Script` cannot express an arbitrary retryable pre-stream failure.

**Zero new `EventKind`.** AG-06 minted both kinds; AG-16 emits them. `event_descriptor.go` and `event_registry_test.go` stay untouched and the scope-fence holds at its committed count.

---

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/loop.go` | **Modified** | `turnAccumulator.usage` field (`:768-810`); capture at the Completion case (`:923-927`); `cost_turn` emission in `finalize()` (`:1001-1025`) |
| `backend/agent/src/agent/cost_events.go` | **Modified** | Presence discriminator on `CostTurn`/`CostSession` + accessors (Decision 1); label-axis doc comments (`:96-103`, Decision 4). **`CostFigures` (`:137-157`) byte-unchanged** |
| `backend/agent/src/agent/harness.go` | **Modified** | Forwarder-side interception (`:507-509`); run-local cumulative; `cost_session` emissions around `:621-624` |
| New production file — usage→figures conversion, presence shape, cumulative accumulator | **New** | Names fixed by `tasks.md`; added to **both** substrate filters by exact filename suffix |
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | **New** | AG-16.1's 3 scenarios, the conversion's table-driven coverage, bites |
| `loop_test.go` (`:831-871`), `loop_hook_test.go` (`:907-943`) | **Modified** | Substrate filter widening, byte-in-sync |
| `openspec/specs/{agent-protocol-events, agent-run-driver, agent-retry-failover, agent-loop-skeleton, agent-turn-termination}` | **Delta** | Five deltas — four normative, one back-annotation |
| `docs/architecture/milestones/0003-…md` | **Modified** | AG-16 checklist tick, R-16/G10 back-annotation, counter to 16/24, **and Decision 5's reconciliation note** |
| `cost_events_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `turn_events.go`, `failure.go`, `run_events.go`, `history.go`, `go.mod`, `go.sum`, **all of `backend/agent/src/ai/**`** | **NOT TOUCHED** | No substrate release requested, no new event kind, no Layer 1 edit, no new dependency, `S-APE-083` byte-unchanged and green |

---

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | `S-APE-083`'s reflection pin breaks | Low (by design) | Decision 1 puts presence on `CostTurn`/`CostSession`; `CostFigures` gains nothing. `sdd-verify` confirms `cost_events_test.go` is **byte-unchanged** and green, not merely that the suite passes. |
| 2 | A whole-record absence flag ships and silently invents zeros for a **mixed** usage record | **Med-High** | The exact defect the acceptance criterion names, and the easiest to ship by accident. A scenario MUST script `ai.Usage{Input: ai.Tokens(100)}` — one figure present, four absent — and assert reasoning reports **absent**, not `0`. A **bite** must prove non-vacuity: collapse presence to a single bool → that scenario fails. |
| 3 | `Turn` emitting a new event changes what every existing standalone-`Turn` test observes | **High** | Same shape as AG-15's Decision 1 risk. `sdd-design` MUST enumerate every test asserting exact event sequences from `Turn` **before** apply; `sdd-verify` reports which test files changed and why. A test edited to accommodate the new event is a signed-off amendment, not a quiet fix. |
| 4 | `cost_turn` placement violates `CheckStream` | Med | `PlacementTurn` requires it inside the bracket. Proven by `CheckStream` accepting the recorded stream **unmodified** with `stream_check.go` byte-unchanged — the fix lands on the emitter, never the validator. |
| 5 | Absent-vs-zero passes vacuously because the fake scripts a zero `ai.Usage{}` and the assertion cannot tell the two apart | Med | Assertions read the **presence bit**, never the count. A scenario pairs `ai.Usage{}` (absent) against `ai.Usage{Input: ai.Tokens(0)}` (reported nought) and requires the two events to **differ**. |
| 6 | Cumulative asserted against a hand-computed constant that later work silently invalidates | Med | This repo's known count-assertion drift class. Cumulative is asserted as an **equality against the observed per-turn events on the same recorded stream**, never against a literal — which is also exactly what Decision 3's framing makes natural. |
| 7 | Decision 4's reinterpretation is recorded in the spec but not in doc 0003, leaving the charter contradicting merged code | Med | Decision 5 makes the note a deliverable with the AG-06 precedent (`0003:613`) as its template. `sdd-verify` checks the note exists and names both reconciliations. |
| 8 | Five spec deltas; one gets missed | Med | Enumerated with file and line above. Both known staleness clusters are in play (owning-spec omission, un-back-annotated merges). `sdd-verify` re-reads each cited line against the shipped change. |
| 9 | Review budget exceeds the raised 1000-line bar | **High** | `size:exception` pre-accepted. Slicing plan below follows the numbered Approach steps, which are already ordered by dependency. |
| 10 | Textual conflict with AG-15 in `harness.go` | Low-Med | AG-15 is **merged** (`09bb30e1`), so this is a rebase concern only, not a parallel-worktree one. Conflicts are textual, not semantic — AG-16 adds to the forwarder and the run-close, and touches no retry gate. |

---

## Rollback Plan

Single revert of the AG-16 merge commit. The new production file and its tests are deleted; `loop.go` returns to discarding `completion.Usage()` at `:923-927`; `cost_events.go` returns to the two-field `CostTurn`/`CostSession`; `harness.go`'s forwarder returns to a bare pass-through; both substrate filters return to their pre-AG-16 filename lists; the five spec deltas are dropped; doc 0003's AG-16 line un-ticks, the counter returns to 15/24, and the reconciliation note is removed.

The revert is clean: nothing persists, no data migrates, no `go.mod`/`go.sum` change, nothing outside `backend/agent`, no Layer 1 file was touched, and no event kind was added or removed. The one **externally visible** removal is the presence accessor on `CostTurn`/`CostSession` — Layer 3 does not exist yet (`0003:110`), so no live consumer is orphaned.

Forward-looking cost: reverting re-opens G10's Layer 2 half and re-defers `agent-run-driver:326` and `agent-retry-failover:219`. **AG-19** consumes AG-16's cumulative accounting for delegated aggregation, so a revert blocks AG-19's cost half. Scheduling consequence, not correctness.

---

## Review-workload forecast

| Component | Estimate (authored, additions + deletions) |
|---|---|
| `loop.go` — accumulator field, capture, `finalize()` emission | 60–100 |
| `cost_events.go` — presence discriminator, accessors, doc comments | 90–150 |
| `harness.go` — interception, cumulative, two `cost_session` sites | 90–150 |
| New production file — conversion, presence shape, accumulator | 150–250 |
| Test files — 3 charter scenarios + conversion table + bites + retry wrapper | 500–750 |
| Filter widening | 20–40 |
| Existing-test accommodation for the new `Turn` event (risk 3) | 0–150 |
| Doc 0003 tick + reconciliation note + traceability | 20–35 |
| **Go subtotal** | **930–1625** |
| SDD markdown — proposal, spec, **5 deltas**, design, tasks, apply-progress, verify-report | **800–1200** |
| **Total** | **1730–2825** |

`Decision needed before apply: No` — `size:exception` pre-accepted at 1000 review lines, recorded here, one PR.
`Chained PRs recommended: No` — but if `sdd-tasks` forecasts above ~3000, slice along the Approach steps: **U1** (capture + conversion, inert, no emission) → **U2** (`cost_turn` emission + Scenario 1) → **U3** (cumulative + Scenarios 2 and 3). Each is independently deliverable and independently valuable; U1 changes no observable behavior at all.
`400-line budget risk: High`

The SDD markdown counts toward the attempt budget. `sdd-tasks` MUST forecast against the **full** diff, not the Go diff.

---

## Dependencies

- **AG-06** (archived) — `CostTurn`, `CostSession`, `CostFigures`, `CostLabel`, `CostScope`, both constructors, the registry rows (`event.go:329-335`), and `S-APE-083`'s pin.
- **AG-13** (archived) — `Harness`, `Run`'s per-turn loop and forwarder (`harness.go:451-626`), the run-close at `:621-624`, the shared `LaneStamper`.
- **AG-15** (merged `09bb30e1`) — the attempt loop (`harness.go:502-570`), `retryDecision`'s gates (`retry_policy.go:111-126`), and the `errorProvider` test-wrapper precedent.
- **AG-11** (archived) — `NFR-ATT-004`, the signature pin Decision 2 preserves.
- **AG-07** — `Turn`, `turnAccumulator`, `finalize()`.
- **Layer 1 (AI-15, AI-19)** — `ai.Usage`, `TokenCount` and its presence bit (`usage.go:44-65`), `Completion.Usage()` (`completion.go:92`), the `CardinalityAtMostOne, Terminal` registration (`ai/event.go:167-170`), `ai.Failure`'s nine fields (`provider_failure.go:320-330`).
- **`agenttest`** — `Script`, `Emit`, `NewProvider`, `Requests()`; `ai.NewCompletion(finishReason, usage)` scripting any `ai.Usage` including the empty one.
- **doc 0003:1527-1560** — the AG-16 charter and its three Gherkin leaves; **doc 0003:613** — the AG-06 reconciliation-note precedent; **doc 0003:114-137** — the evidence gate and test-substrate binding.

---

## Success Criteria — restated as verifiable checks

- [ ] `cd backend/agent && make test` green under `-race`; all three AG-16.1 scenarios closed with recorded evidence
- [ ] **Scenario 1 / exactness** — per-turn `cost_turn` figures match the fake's scripted usage **exactly**, including cache-read, cache-write and reasoning, for every turn of a multi-turn run
- [ ] **Scenario 1 / absence** — a turn scripted with `ai.Usage{}` emits a `cost_turn` that **reports absence**; a turn scripted with `ai.Tokens(0)` emits one reporting a **present zero**; the two events are **observably different**
- [ ] **Scenario 1 / mixed record** — `ai.Usage{Input: ai.Tokens(100)}` yields input present at 100 and the other four **absent**, not `0` (risk 2)
- [ ] **Scenario 1 / label** — every `cost_turn` carries `CostLabelFinal` (Decision 4)
- [ ] **Scenario 2** — cumulative equals the sum over **every emitted `cost_turn` on the recorded stream**, asserted as an equality against the observed events and **not** against a hand-computed literal (risk 6); a run containing a retried logical turn contributes that turn's real figures, neither dropped nor zeroed
- [ ] **Scenario 3** — a multi-turn run emits intermediate `cost_session(Estimate)` events and exactly one run-terminal `cost_session(Final)` that corrects them; a single-turn run emits the `Final` one alone
- [ ] **Bites, RED-recorded before GREEN**: (a) collapse per-figure presence to a single whole-record bool → the mixed-record scenario fails; (b) make the accumulator skip retried turns → Scenario 2 fails; (c) label the run-terminal `cost_session` `Estimate` → Scenario 3 fails
- [ ] **`CostFigures` is byte-unchanged** and `cost_events_test.go` is **byte-unchanged** and green — `S-APE-083`'s five-field, in-order, `uint64`, no-money pin holds untouched
- [ ] **`Turn`'s exported signature is unchanged** — `NFR-ATT-004` and `agent-turn-termination/spec.md:113` hold, verified by reading the shipped signature
- [ ] `CheckStream` accepts the recorded multi-turn, retry-inclusive, cost-bearing stream **unmodified**, with `stream_check.go` byte-unchanged
- [ ] **No new `EventKind`** — `event_descriptor.go` and `event_registry_test.go` untouched; the every-kind-constructible guard passes at its committed count
- [ ] No payload carries any money-, currency- or price-suggesting field; the forbidden-substring scan passes unchanged
- [ ] Zero files under `backend/agent/src/ai/` differ; `go.mod`/`go.sum` unchanged; import and ambient-authority guards pass with **zero** changes
- [ ] Both substrate filters carry an identical exact-filename entry set, one entry per file AG-16 introduces, no wildcard or prefix pattern
- [ ] All five spec deltas written; each cited line re-read against the shipped change by `sdd-verify`
- [ ] **Decision 5's reconciliation note** exists on doc 0003's AG-16 charter, follows the AG-06 template (`0003:613`), and names **both** the retried-attempt and the estimate/final reconciliations
- [ ] `make lint` clean (after `golangci-lint cache clean`), `make build` clean, `make vuln-check` clean — `vuln-check` is **not** in `make all`
- [ ] doc 0003's AG-16 checklist ticked, R-16/G10 back-annotated, milestone counter bumped to 16/24
