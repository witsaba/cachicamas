# Proposal: AG-18 — Implement compaction

> **Change**: `cachicamas-agent-compaction` · **Milestone**: AG-18 (Layer 2 Wave 4, milestone 18 of 24; doc `0003:1655-1760`)
> **Branch**: `feat/agent-layer2-wave4-ag18` · base `origin/main@f6acc0d2` · **Worktree**: `cachicamas-worktrees/ag18`
> **Artifact store**: hybrid (Engram + filesystem) · **Execution mode**: `auto`
> **Delivery**: `exception-ok` — a **single PR** under a pre-accepted `size:exception`, shipping the change, the doc 0003 milestone-document update and the OpenSpec archive together (the AG-16/AG-17 house pattern)
> **Review budget**: 1000 changed lines, counted **excluding everything under `openspec/`** — the user designated `openspec/` a working folder whose lines do not count toward reviewer burden. `sdd-tasks` and `sdd-apply` inherit this counting rule verbatim.
> **TDD**: strict, RED-first. Canonical runner: `cd backend/agent && go test -race -count=1 ./...`. **`-count=1` is mandatory** — the real uncached suite is ~3 minutes (`openaicompat` alone ≈175s); a sub-second "pass" is a cache artifact, not evidence.
> **Closes**: **G3 / R-11** — `agent-v1-scope/spec.md:148` states verbatim that AG-17's back-annotation "does **not** discharge … R-11, which is **AG-18's**".
> **Depends on**: AG-02, AG-06, AG-12, AG-13, AG-15, AG-16, AG-17 (all merged) · **Blocks**: doc 0004's `/compact`
> **Exploration**: `explore.md` · Engram `sdd/cachicamas-agent-compaction/explore` (obs #3469)
> **ID prefix**: `R-CMP-` / `S-CMP-` — verified free: zero occurrences under `openspec/`

---

## Intent

The runtime can **measure** its context and can **ask** whether to shrink it. It cannot shrink it.

AG-17 shipped the seam and stopped there by construction: `ContextVerdict` is `struct{}` (`context_strategy.go:53`), so no strategy a caller could write can request compaction, and the harness discards the verdict it receives (`harness.go:524-530`). The transcript therefore only ever grows. A long session ends when the window ends — there is no graceful degradation, no `/compact`, and G3's "protect recent turns" is a promise with no mechanism behind it.

AG-18 is the mechanism. It is the first milestone in Layer 2 that **removes** something a run committed, and that is why it is not an ordinary implementation milestone.

**Three promoted specs already name AG-18 as the owner of this debt.** Each verified by opening it in this worktree:

| Spec row | Verbatim | Retires as |
|---|---|---|
| `agent-history/spec.md:258` | "The invariant holds after compaction removes entries \| **AG-18.2**, which re-proves it — ordinal-derived identity is a known, named constraint AG-18 inherits" | **CLOSED by AG-18** |
| `agent-run-driver/spec.md:342` | "AG-17 inserts the check, AG-18 implements … **Not closed here**: compaction itself" | **CLOSED by AG-18** — the second half |
| `agent-protocol-events/spec.md:154` | "The compaction and delegation families remain unemitted, owned by **AG-18** and AG-19 respectively" | **CLOSED by AG-18** (compaction half) |

**What ticks**: R-11 / G3 discharge; `agent-v1-scope`'s AG-18 inheritance statement (`S-AGS-050`/`S-AGS-051`) gains its back-annotation; the compaction event family gains its first production caller after twelve milestones of being constructible-but-unemitted; doc 0003 advances to 18/24.

---

## Deliverable 0 — the spec amendments, stated first because they are not a byproduct

**AG-18 cannot be scoped as "implementation against frozen specs."** Its charter requires behavior three promoted requirements presently forbid at MUST level. The amendments are the first deliverable, authored deliberately in `sdd-spec`, not discovered late in `sdd-apply`.

### A1 — `agent-history` `R-HIS-001`: the append-only clause (**certain**)

Verbatim today (`agent-history/spec.md:52`):

> "Within a run, history MUST NOT support removal, reordering, or in-place mutation of an already-committed entry; the only admitted extension is an append through the commit path of `R-HIS-004`."

AG-18.2 replaces a span of entries with one summary entry. **That is removal.** The contradiction is direct and must be resolved by a `MODIFIED` delta, never by shipping code that quietly falsifies the sentence.

**The exception must be fenced narrowly, and something must replace what is relaxed.** A relaxation with nothing put in its place is this change's characteristic failure mode. Proposed shape, for `sdd-spec` to make normative:

| What is relaxed | What replaces it |
|---|---|
| "MUST NOT support removal" becomes: **exactly one** removal operation exists — a span replacement — reachable **only** through the same single validating commit primitive of `R-HIS-004` | Removal MUST be **span-shaped and pairing-closed**: the replaced range MUST NOT split a tool-call/tool-result pair in either direction, and the post-replacement transcript MUST satisfy the **same** append-time pairing rules `NewSeededHistory` (`history.go:216-224`) already enforces. The invariant is not weakened; it is re-proved on a smaller transcript. |
| An ordinary caller's ability to remove, reorder, or rewrite an entry | **Unchanged — still forbidden.** Reordering and in-place mutation of a surviving entry remain MUST NOT on every route. Compaction may only *replace a contiguous span*; it may not permute, edit, or partially rewrite anything it does not replace. |
| Nothing else | Protected trailing entries MUST be preserved **value-identical**: their Layer 1 values and their origin discriminators MUST compare equal before and after. |

### A2 — `agent-history` `R-HIS-004`: the closed route enumeration (**certain**)

`R-HIS-004` (`:91`) reads "Every route that can extend **or mutate** the transcript — append, seeded construction, and orphan synthesis alike — MUST funnel through a single validating commit primitive", and `:93` requires the enumeration be **closed**: "a public mutating route that the enumeration does not name MUST be observable as a failure of the suite."

Compaction is a fourth public mutating route. `R-HIS-004` is not *violated* by AG-18 — it is *satisfied* by routing compaction through the same primitive — but `S-HIS-030`'s enumeration must **name** it, or `S-HIS-031`'s closure bite is the thing that fires. The delta amends the enumeration, not the rule.

### A3 — `agent-history` `R-HIS-005`: ordinal identity across a compaction (**certain, and the subtle one**)

`R-HIS-005` (`:106`) says identity "MUST be **derived deterministically from ordinal position**", no caller may mint one, and "Identity MUST be stable."

Shrinking the transcript shifts every ordinal after the summary. Preserving the old `EntryID`s would require minting non-ordinal identities — **the C1 back door `R-HIS-005` exists to close**. So renumbering is *forced by `R-HIS-005`'s own rule*, not chosen.

The amendment therefore states the generation semantics plainly rather than pretending stability survives:

- Identity remains ordinal-derived and caller-unmintable. **Unchanged.**
- Stability is scoped: identities are stable **within a transcript generation**; a compaction begins a new generation. The same transcript still yields the same identities on every read and across processes.
- **The replacement invariant**: what survives a compaction as a durable handle is **`TurnID`, not `EntryID`** — which is exactly why AG-06.4 built `CompactionSpan` out of `TurnID` (`compaction_events.go:50-53`, `R-APE-007`). The safe axis was chosen twelve milestones ago; the spec now says so.
- Consequence to state, because a reader will otherwise infer the wrong thing from the charter's word "byte-identical": **protected turns are byte-identical in their Layer 1 values and origin discriminators, and are NOT identical in their entry identities.** Any test written as `reflect.DeepEqual` over whole entries including identity will be asserting a false claim.
- Cheap and verifiable today: `EntryID` usage is fully contained to `history.go`/`history_test.go` (grepped repo-wide in `explore.md` §1), so no stored identity elsewhere is silently invalidated.

### A4 — `agent-history` `R-HIS-009` / `S-HIS-080` and the `L2C-07` row (**certain**)

The guarded row, read verbatim in this worktree (`doc_contract_guard_test.go:69`), has **two** clauses AG-18 falsifies:

1. "every route that can extend it — **append, seeded construction, orphan synthesis** — funnels through one validating commit primitive" — a closed three-route enumeration; compaction is a fourth.
2. "stable **ordinal entry identity**" — see A3.

`R-HIS-009` binds the `doc.go` row and its `expectedLayer2ContractRows` entry to land in the **same pull request** (`R-AGP-002`'s closed-amendment rule). Both clauses must be re-worded together.

**Pre-existing drift to correct in the same delta**: `S-HIS-080` (`:182`) claims the table "contains **7 rows** (`L2C-01`..`L2C-07`)". The real table has **8** (`L2C-01`..`L2C-08`; AG-14 added `L2C-08`, and `S-CTX-021` independently pins "eight committed rows"). This is AG-14's un-back-annotated merge, **not** AG-18's defect — but AG-18 is the next change to touch this table, and shipping a ninth row over a stale "7" compounds the drift. Correct it here. *(This is the repo's known count-assertion drift class: re-scope the claim off the literal count rather than re-fixing the number.)*

### A5 — `agent-context-strategy` `R-CTX-003` and `S-CTX-005` (**certain**)

`R-CTX-003` (`:71`) requires the verdict type expose "**no field, no method and no constructor** by which any implementation could request compaction. Its zero value MUST be its only constructible value." `S-CTX-005` (`:81`) closes with: "This scenario is what guards AG-18's extension point: **it must be deliberately amended by AG-18, never silently outgrown.**"

That is an explicit, written instruction addressed to this change. The delta amends `R-CTX-003` from *unconstructible* to *constructible only by an implementation that supplies its own provider, options and instruction* — the never-compact guarantee moves from a property of the type to a property of the **zero value** (a zero `ContextVerdict` still requests nothing), which is the strongest form still available once the extension point opens. `R-CTX-004`'s inertness pin survives unchanged for the nil/never-compact case, but its sentence "it opens no new history route and takes no write path" (`:95`) must be re-scoped to "on any path where no compaction is requested."

### A6 — `agent-loop-skeleton` `R-LSK-004`: the substrate release (**certain**)

`R-LSK-004` (`:103`) names `doc.go` and `doc_contract_guard_test.go` — and `reconstruction_test.go` — in the byte-unchanged substrate list. A4 requires editing the first two, and AG-18.3's reconstruction-style test may touch the third. The house pattern is a **recorded exact-filename release** in an `agent-loop-skeleton` delta (the AG-11 / AG-14 / AG-16 precedent, stated in the requirement's own heading). Both substrate filters must widen by exact filename, byte-in-sync.

### A7 — back-annotations, no normative change (**certain**)

`agent-protocol-events` (`:154`, compaction family's live-emission half), `agent-run-driver` (`:342` compaction half; `:351`'s "a new exported `History` method — **not this milestone, and forbidden under this change**" row, which AG-18 is the first milestone to breach and must record as such), and `agent-v1-scope` (R-11 / G3 discharge; `S-AGS-015`, `S-AGS-050`, `S-AGS-051`). **No Go test enforces a back-annotation** — this is the repo's known un-back-annotated-merge staleness shape. `sdd-verify` must open each cited line.

### A8 — `agent-cost-events` `R-CST-001` (**conditional; newly identified, not in `explore.md`**)

`R-CST-001` (`:42`) is stated as an **iff**: "For **every** turn that closes non-aborted, the system MUST emit a `cost_turn` … mapped from the `ai.Usage` the turn's `ai.Completion` reported … a non-aborted close always emits one, and an aborted close never does."

If D3 lands as recommended — compaction gets its **own** `turn_start`/`turn_end` bracket — then a bracket exists that closes without being a `Turn()` call. Either the word "turn" in `R-CST-001` is scoped to model turns and the compaction bracket named explicitly (a `MODIFIED` delta), or D3 must be re-decided. **This is a decision `sdd-design` must make explicitly and `sdd-spec` must record**; it must not be resolved by silence. `R-CST-007`'s scope fence is *not* an obstacle: its text is scoped to "**This capability**" (AG-16's own change), so it constrains AG-16, not AG-18.

---

## Scope

### In

- **AG-18.1** — a compaction execution path: an injected `ai.ModelProvider` (distinct from `h.Provider`), injected options, and an **injected instruction string**; one non-tool completion against `runCtx`, so it participates in the run's existing cancellation tree; its `ai.Usage` converted through the existing `newCostTurnFromUsage` (`cost_usage.go:70-83`) and folded by the existing accumulator with no retry-awareness added.
- **AG-18.2** — a pure, read-only span-widening helper over `History.Entries()` that expands a naive cut point to a fixed point so neither boundary splits a call/result pair; **one** new `History` span-replacement route through the single commit primitive; a discriminable entry origin for the summary entry.
- **AG-18.3** — the first production call sites for `NewCompactionStarted` / `NewCompactionFinished` / `NewCompactionFailed`, placed so `CheckStream` accepts the recorded stream **unmodified**, and so a resumed session given only `Span()` and `SummaryID()` can name exactly which turns were replaced and by what.
- **AG-18.4** — atomicity by ordering: the complete replacement message set is built in memory **before** any commit is attempted, so an interruption cannot reach the commit step; on any error, `compaction_failed` (never `compaction_finished`) and the run's turn loop proceeds unchanged.
- **AG-18.5** — an on-demand `Harness` entry point invoking the **same** mechanics, gated to turn boundaries, returning a **typed refusal** mid-turn — never queued silently, never racing the loop.
- **Deliverable 0** — the seven certain spec amendments above, plus A8 if D3 stands.
- Substrate-filter widening by exact filename, byte-in-sync (the AG-11…AG-17 discipline).
- Doc 0003 milestone-document update and the OpenSpec archive of this change, **in the same PR**.

### Out — deferred, with the owner named

| Deferred | Owner and why the deferral is safe |
|---|---|
| **When to compact** — any threshold arithmetic over budget vs. accounting | **Layer 3.** Charter `0003:1665`; `agent-context-strategy/spec.md:280`. AG-18 acts on a verdict; it never decides one. |
| **The summarization instruction's content** | **Layer 3, injected.** Charter `0003:1662`: "Layer 2 owns no prompt content, so this milestone's tests script one." AG-18's production code MUST NOT author instruction text on any path — this is an assertable property, not a style note. |
| **Persistence of the compaction record across processes** | **Layer 3 session.** Charter `0003:1665`; `agent-history/spec.md:257`'s standing precedent. |
| **A real tokenizer or exact boundary sizing** | **Out of scope and structurally impossible.** `agent-context-strategy/spec.md:284`: `L2C-01` allows the standard library and `src/ai`'s measured closure only, and `openspec/config.yaml:33` forbids a new top-level dep without an ADR. AG-17's estimate stays a documented heuristic; AG-18 **consumes** it and does not replace it. |
| **Aggregating compaction spend into a parent or delegated run** | **AG-19**, by `R-CST-007`'s own line (`0003:1533`). The run's own cumulative is in scope; nothing above it is. |
| **A narrower "abort compaction only" cancellation signal** | **Not built.** The charter says compaction "cancels via the run's cancellation tree" — i.e. participates in the existing tree, not a new branch of it. No new cancellation primitive; building one speculatively is scope creep. |
| **Any new `EventKind`, `CostLabel`, or `TurnOutcome`** | **Not this change.** The compaction family and its `PlacementTurn` rule are fully pre-provisioned by AG-06 (`event.go:205-220`, 25 registered kinds pinned by `invariant_pin_test.go:47-68`). `event.go`, `event_descriptor.go`, `event_registry_test.go` and `stream_check.go` stay byte-unchanged. |
| **Any edit under `backend/agent/src/ai/**`** | **Never in Layer 2** (`R-RUN-012`). Layer 1 is consumed, never edited. |
| **Re-opening `R-CAN-002`'s wind-down order** | **CLOSED by AG-14 and untouched.** A compaction failure MUST NOT route through `windDownRun` — that order is closed and `CheckStream`-validated. Explicit trap, named so it is avoided rather than discovered. |
| **`S-HIS-080`'s stale row count as a standalone fix** | Corrected **incidentally** here (A4) because this PR touches the same table. Not a separate deliverable. |

---

## Decisions to be settled in `sdd-design` — recommended, not settled here

Each carries a recommendation and the **criterion that would overturn it**. `sdd-design` decides; this proposal does not.

**D1 — how the compactable span is widened so it never splits a pair.**
*Recommend:* a **pure two-directional widening helper** over the public `Entries()` view, iterating to a fixed point (a widened boundary can itself expose a further pair), reusing the ID-correlation logic `resolveOpenSet` (`history.go:408-432`) already proved correct.
*Overturned if:* the fixed-point iteration cannot be shown to terminate on an adversarial interleaving, or if correlation turns out to need private state `Entries()` does not expose — in which case a read-only pairing accessor on `History` is the fallback, at the cost of widening the `history_surface_guard_test.go` enumeration (`S-HIS-030`/`S-HIS-042`).

**D2 — how the summary entry is typed.**
*Recommend:* extend `EntryOrigin` with a third member (e.g. `EntryOriginSummarized`), mirroring the `appended`/`synthesized` split (`history.go:81-92`).
*Overturned if:* a third origin value proves unmintable without a second commit door — which would re-open A1/A2 rather than settle them. **A content-level sentinel is not an option**: `R-HIS-007` (`:139`) already prohibits it by name — "a real tool can emit any content bytes, so a content sentinel is forgeable" — and that argument does not weaken for compaction.

**D3 — where compaction runs, and how atomicity is achieved.**
*Recommend:* **build-then-commit** — compute the widened span and obtain the summary entirely in memory, then perform one span-replacement commit. Atomic-or-absent falls out of **ordering**, not out of a rollback mechanism or a journal. Run it in the "no turn open" gap after `ContextStrategy.Resolve` returns and before the next `Turn()`, inside a **dedicated compaction turn bracket** with its own fresh `TurnID`, so the family's `PlacementTurn` rule (`stream_check.go:161`) is satisfiable at a point where no turn is otherwise open.
*Overturned if:* A8 resolves against a dedicated bracket — i.e. if scoping `R-CST-001`'s iff proves worse than relocating the events. **A pointer swap at `Harness.History` is ruled out by evidence, not by preference**: `harness_test.go:1013` pins `h.History != hist` → "Run must use the caller's history in place, never replace it."

**D4 — how the on-demand entry point refuses a mid-turn demand.**
*Recommend:* a `turnInFlight` flag guarded by the existing `h.signalMu`, mirroring how `Interrupt`/`Shutdown` detect run-in-flight via `h.cancelRun != nil`. Note the granularities differ: today's flag is *run*-in-flight, and AG-18.5 needs *turn*-in-flight.
*Overturned if:* routing the request through the existing steering-queue mechanism makes it inherently only actionable between turns — which would need no new state at all and would be strictly better.

**D5 — whether compaction spend reuses `cost_turn`.**
*Recommend:* **reuse it unmodified**, with `CostLabelFinal` and the compaction bracket's own `TurnID`. Distinguishability comes free from the bracket identity, correlatable through `CompactionFinished.Span()`. No new label, no new kind, `cost_events.go`/`cost_usage.go` byte-unchanged.
*Overturned if:* A8 forces the events out of a dedicated bracket, removing the `TurnID` that carries the distinguishability.

**D6 — how provider, options and instruction are injected without Layer 2 authoring content.**
*Recommend:* **extend `ContextVerdict`** — the carrier its own doc comment (`context_strategy.go:47-52`) already forecasts: "AG-18 adds compaction fields non-breakingly; every implementation returning the zero verdict keeps compiling." Fields: a request discriminator, the injected provider, injected options, the injected instruction, and enough boundary information for D1's helper to compute a naive cut point.
*Overturned if:* the verdict cannot carry a boundary hint without leaking harness state to the strategy (`R-CTX-002`: the strategy "MUST NOT be able to reach the harness's own state"). **A second seam method is rejected**: it would consult the strategy twice per turn boundary and contradict `R-CTX-001`'s once-per-logical-turn cardinality.

**D7 — what "protected turns are byte-identical" means concretely.** *(raised by A3; not in `explore.md`)*
*Recommend:* value-identity over Layer 1 values **and** origin discriminators, explicitly **excluding** entry identity, with the exclusion stated in the requirement rather than left to the test author.
*Overturned if:* a way is found to preserve `EntryID`s across a compaction without minting a non-ordinal identity — which `R-HIS-005:106` forbids, so this is unlikely to be overturnable.

---

## Capabilities

### New Capabilities

- `agent-compaction`: the compaction execution path, invariant-safe span surgery, the compaction bracket on the stream, interruption atomicity, and the on-demand entry point. IDs `R-CMP-0NN` / `S-CMP-0NN`, append-only, prefix verified free. **No total count is stated** — a count goes silently false the moment a later milestone appends (`S-LSK-020`).

### Modified Capabilities

| Capability | What changes | Certainty |
|---|---|---|
| `agent-history` | `R-HIS-001` (A1), `R-HIS-004` enumeration (A2), `R-HIS-005` generation semantics (A3), `R-HIS-009`/`S-HIS-080` + the `L2C-07` row (A4), non-requirements row `:258` | **Certain** |
| `agent-context-strategy` | `R-CTX-003` + `S-CTX-005` (A5), `R-CTX-004` re-scoping, non-requirements rows `:278`/`:282`/`:283` | **Certain** |
| `agent-loop-skeleton` | `R-LSK-004` recorded exact-filename release (A6) | **Certain** |
| `agent-protocol-events` | Back-annotation `:154` — compaction family's live-emission half closed (A7) | **Certain** |
| `agent-run-driver` | Back-annotation `:342` and `:351` (A7) | **Certain** |
| `agent-v1-scope` | Back-annotation — R-11 / G3 discharged; `S-AGS-015`, `S-AGS-050`, `S-AGS-051` (A7) | **Certain** |
| `agent-cost-events` | `R-CST-001`'s iff scoped to model turns (A8) | **Conditional on D3** |

---

## Approach

1. **Author the amendments first** (Deliverable 0). Nothing in `sdd-apply` may proceed against a spec the change is known to falsify.
2. **D1's widening helper** — a pure function, no `History` mutation, testable in isolation against adversarial interleavings. Self-contained and independently deliverable.
3. **D2's origin member + the span-replacement commit route** through the existing single primitive, with `S-HIS-030`'s enumeration extended in the same commit as the route.
4. **D6's `ContextVerdict` extension**, non-breaking: a zero verdict still requests nothing, and `NoOpContextStrategy` keeps compiling untouched.
5. **AG-18.1's execution path** — its own file, its own injected provider, `runCtx`-scoped, `newCostTurnFromUsage` reused, folded by the existing forwarder.
6. **D3's bracket and the three event call sites**, validated by `CheckStream` **unmodified**.
7. **AG-18.4** falls out of step 5's ordering — assert it, do not build a mechanism for it.
8. **AG-18.5's entry point + D4's state**, then the equivalence assertion: strategy-triggered and on-demand streams equal `Kind()`-for-`Kind()`, excluding fresh `RunID`/`TurnID`, reusing `context_strategy_test.go:222-224`'s established exclusion.
9. **Doc 0003 update + archive**, same PR.

**Bites are not optional.** At minimum: (a) narrow the widening to a single direction → the split-pair scenario must FAIL; (b) commit before the model call returns → the atomicity scenario must FAIL under a cancelled `runCtx`; (c) drop the fourth route from `S-HIS-030`'s enumeration → the closure guard must FAIL naming it; (d) let the on-demand path queue instead of refusing → the typed-refusal scenario must FAIL. Each RED-recorded **before** its GREEN, with `-count=1`.

---

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/compaction.go` (new) | **New** | The execution path, the span-widening helper, the bracket and its three event call sites |
| `backend/agent/src/agent/history.go` | **Modified** | The span-replacement route through the single commit primitive; `EntryOrigin`'s third member |
| `backend/agent/src/agent/context_strategy.go` | **Modified** | `ContextVerdict`'s compaction fields (D6) |
| `backend/agent/src/agent/harness.go` | **Modified** | Capture the verdict at `:524-530` instead of discarding it; the on-demand method; D4's turn-in-flight state |
| `backend/agent/src/agent/doc.go`, `doc_contract_guard_test.go` | **Modified** | `L2C-07`'s two falsified clauses, amended together in one PR (A4, A6) |
| `backend/agent/src/agent/*_test.go` (new, `package agent_test`) | **New** | Five leaves' scenarios plus the four bites |
| `history_surface_guard_test.go`, substrate filters | **Modified** | Route enumeration + exact-filename widening, byte-in-sync |
| `openspec/specs/{agent-history, agent-context-strategy, agent-loop-skeleton, agent-protocol-events, agent-run-driver, agent-v1-scope}` | **Delta** | Six certain deltas; `agent-cost-events` conditional (A8) |
| `docs/architecture/milestones/0003-…md` | **Modified** | AG-18 status line, delivery table, checklist; counter to 18/24 |
| `event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `compaction_events.go`, `cost_events.go`, `go.mod`, `go.sum`, **all of `backend/agent/src/ai/**`** | **NOT TOUCHED** | No new kind, no Layer 1 edit, no new dependency; the 25-kind guard passes at its committed count |

---

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | **The review budget is exceeded by a wide margin.** AG-17 shipped 4503 insertions across 29 files for **two** leaves; AG-18 has **five**, plus a new capability spec and 6–7 deltas | **High** | `size:exception` pre-accepted at 1000 lines **excluding `openspec/`**, one PR, recorded here. If `sdd-tasks` forecasts beyond ~3000 non-`openspec` lines, slice along the Approach steps: **U1** = D1's pure helper + D2's route + the `agent-history` delta (no provider call, no events, independently deliverable) → **U2** = AG-18.1/18.4 → **U3** = AG-18.3/18.5. |
| 2 | `sdd-apply` implements the code and **silently falsifies `R-HIS-001`** without the delta | **High** | Deliverable 0 is stated before Scope for exactly this reason. `sdd-verify` MUST open `agent-history/spec.md:52` against the shipped `history.go` and confirm the sentence and the code agree. A citation is not evidence. |
| 3 | The amendment **relaxes `R-HIS-001` with nothing put in its place** | **High** | A1's table names the replacement invariant per relaxed clause. A delta that removes the prohibition without adding the pairing-closed span constraint MUST be rejected in `sdd-verify`. |
| 4 | `L2C-07`'s row is amended for the route enumeration but **not** for "stable ordinal entry identity" | **Med-High** | A4 names **both** clauses. The repo's own lesson applies: correcting a requirement leaves its scenarios wrong — grep every occurrence of the phrase, not just the first. |
| 5 | **A8 is resolved by silence** — a compaction bracket ships and `R-CST-001`'s iff quietly becomes false | **Med-High — newly identified, not in `explore.md`** | `sdd-design` MUST decide D3/A8 explicitly and `sdd-spec` MUST record the decision either way. A new bracket in an existing closed sequence is this repo's known spec-breakage shape. |
| 6 | A test asserts protected turns are `DeepEqual` **including entry identity** and encodes a false claim that still passes because the fixture happens not to shift ordinals | Med-High | D7 + A3. The protection assertion MUST be written over Layer 1 values and origins explicitly, and MUST be exercised on a fixture where the span **does** shift the protected entries' ordinals. |
| 7 | A compaction failure is routed through `windDownRun`, breaking `R-CAN-002`'s closed order | Med | Named as an explicit trap in Scope-Out. AG-18.4's scenario asserts the **next** turn proceeds — a run that wound down cannot satisfy it. |
| 8 | `S-LSK-001`'s length-equality sequence, or a run-level enumerated sequence in `harness_test.go`, breaks on the new bracket | Med | `S-LSK-001` is verified **not** at risk (it asserts a direct `Turn()` call with no `Harness`). Run-level sequences in `harness_test.go` were **not** independently verified in exploration — `sdd-tasks` MUST grep enumerated sequences and length-equality assertions before editing. |
| 9 | Six-to-seven deltas; one is missed | **Med-High** | Enumerated with file and line in Capabilities. **Three of them (`agent-loop-skeleton`, `agent-run-driver`, `agent-v1-scope`) were not in `explore.md`** and were found by reading in this phase. No Go test enforces a back-annotation. |
| 10 | Evidence is recorded from a **cached** test run | Med | `-count=1` mandated in the header and in every acceptance line. A sub-second suite for this module is a cache artifact — the real uncached run is ~3 minutes. |
| 11 | The implementation authors instruction content on some path (a default, a fallback, a prefix) | Med | Charter-level violation. The AG-18.1 scenario asserts the captured request carries the injected instruction **and no runtime-authored content** — assert over the captured bytes via `Provider.Requests()`, not over a comment. |
| 12 | `EntryOrigin`'s third member is added but the discriminator is never asserted **without reading content** | Low-Med | `R-HIS-007`'s existing prohibition carries forward verbatim: no test may distinguish origin by content or `Failed()`. |

---

## Rollback Plan

**Single revert of the AG-18 merge commit.**

`compaction.go` and all new test files are deleted; `history.go` loses the span-replacement route and `EntryOrigin`'s third member; `context_strategy.go`'s `ContextVerdict` returns to `struct{}`; `harness.go` returns to discarding the verdict at `:524-530` and loses the on-demand method and the turn-in-flight state; `doc.go` and `doc_contract_guard_test.go` return to their eight-row `L2C-07` text; both substrate filters return to their pre-AG-18 filename lists; the six-to-seven deltas are dropped; doc 0003's AG-18 line un-ticks and the counter returns to 17/24.

**The revert is clean, and the reason is structural.** Nothing persists across processes (persistence is Layer 3's, and this change adds none). No data migrates. No `go.mod`/`go.sum` change. Nothing outside `backend/agent`. No Layer 1 file touched. No `EventKind` added or removed — the family was minted by AG-06 and returns to having zero production callers, exactly its pre-AG-18 state.

**The one genuinely irreversible thing is not in the code.** A compaction that already ran in a live process has **discarded transcript entries that no longer exist anywhere in memory**. Reverting the code cannot restore them, and this change ships no compaction journal, no undo, and no pre-compaction snapshot. That is acceptable *today* only because Layer 3 does not exist (`0003:110`) and nothing outside tests drives a real run — but it MUST be stated in the spec as a named consequence rather than discovered by the first Layer 3 consumer.

**Forward-looking cost**: reverting re-opens R-11 / G3, re-opens `agent-run-driver:342`'s second half and `agent-history:258`, returns the compaction family to unemitted, and removes the mechanism doc 0004's `/compact` stands on. Scheduling consequence, not correctness.

---

## Review-workload forecast

**Counting rule for this change**: additions + deletions, **excluding every path under `openspec/`**. The user designated `openspec/` a working folder whose lines do not count toward reviewer burden. SDD markdown still counts toward the **attempt budget** — a different budget, and `sdd-tasks` must not conflate them.

| Component | Estimate (counted: authored, non-`openspec`) |
|---|---|
| `compaction.go` — execution path, widening helper, bracket, three call sites | 260–420 |
| `history.go` — span-replacement route, third `EntryOrigin` member | 130–230 |
| `context_strategy.go` — verdict extension | 60–110 |
| `harness.go` — verdict capture, on-demand method, turn-in-flight state | 120–210 |
| `doc.go` + `doc_contract_guard_test.go` — `L2C-07` amendment | 20–40 |
| `agenttest` additions (second scripted provider instance, mis-aligned-cut fixture, verdict helper) | 50–100 |
| Test files — five leaves' scenarios + four bites + the surface-guard enumeration | 900–1500 |
| Substrate filter widening | 20–40 |
| Doc 0003 status line, delivery table, checklist | 25–45 |
| **Counted total (excludes `openspec/`)** | **1585–2695** |
| *Uncounted, but inside the attempt budget*: proposal, new `agent-compaction` spec, 6–7 deltas, design, tasks, apply-progress, verify-report, archive report | *1300–2100* |
| *Full-diff total (attempt-budget relevant)* | *2885–4795* |

`Decision needed before apply: No` — `exception-ok`, `size:exception` pre-accepted at 1000 counted lines, one PR.
`Chained PRs recommended: No` — single PR is the accepted strategy. Slice plan held in reserve (risk 1) if `sdd-tasks` forecasts beyond ~3000 counted lines.
`400-line budget risk: High`

---

## Dependencies

- **AG-06** (archived) — `compaction_events.go`'s three constructors and `CompactionSpan`; `R-APE-007`/`R-APE-008`; the 25-kind registry pinned by `invariant_pin_test.go:47-68`.
- **AG-12** (archived) — `History`, `Entries()`, `NewSeededHistory` (`history.go:216-224`) as the boundary-validation entry point, `resolveOpenSet` (`:408-432`), `EntryOrigin` (`:81-92`).
- **AG-13** (archived) — `Harness.Run`'s outer per-logical-turn loop; `CloseTurn` at `harness.go:668`, which is why the seam sits in a **no-turn-open gap**.
- **AG-14** (merged) — `windDownRun` (`harness.go:329-350`) and `R-CAN-002`'s closed order; `L2C-08`.
- **AG-15** (merged) — the attempt loop and `R-RTY-002`'s by-reference transcript pin (`harness.go:550-561`).
- **AG-16** (merged) — `costAccumulator.add` (`cost_usage.go:118-130`), `newCostTurnFromUsage` (`:70-83`), the unconditional forwarder fold (`harness.go:562-580`).
- **AG-17** (merged, `f6acc0d2`) — the seam, `ContextPrompt`/`ContextVerdict`, `ContextBudget`, `TokenAccounting`, and the call site at `harness.go:524-530`.
- **`agenttest`** — `NewProvider`, `Requests()` (`fake_request_capture_test.go`), `Gate` (`fake_gate.go`, the mandated no-sleep sync primitive), the `stream_kit_*` assertion kit.
- **doc 0003:1655-1760** — the AG-18 charter and its five Gherkin leaves.

---

## Success Criteria — restated as verifiable checks

- [ ] `cd backend/agent && go test -race -count=1 ./...` green; **not** a cached run — record the wall-clock duration as part of the evidence
- [ ] **AG-18.1 / own call** — a compaction provider distinct from `h.Provider` is proven called; its usage reaches the run cumulative; cancelling `runCtx` mid-call aborts the compaction **without** ending the run
- [ ] **AG-18.1 / injection** — the captured request carries **exactly** the injected instruction and no runtime-authored content, asserted over captured bytes
- [ ] **AG-18.2 / no split pair** — a fixture whose naive cut splits a call/result pair yields a widened span containing both, by construction
- [ ] **AG-18.2 / boundary validation** — the post-compaction transcript passes the **same** pairing rules `NewSeededHistory` enforces
- [ ] **AG-18.2 / protection** — protected trailing entries compare equal in Layer 1 values **and** origins, on a fixture where their ordinals **do** shift (D7, risk 6)
- [ ] **AG-18.2 / typed summary** — the summary entry is distinguishable from a model message **without** reading content or `Failed()`
- [ ] **AG-18.3 / stream** — `CheckStream` accepts the recorded stream with `stream_check.go` **byte-unchanged**; a reconstruction given only `Span()` and `SummaryID()` names the replaced turns and the summary
- [ ] **AG-18.4 / atomic-or-absent** — after an interrupted compaction, `hist.Entries()` is byte-identical to its pre-attempt state, `compaction_failed` (not `finished`) was emitted, the next turn runs against the uncompacted transcript, and `windDownRun` was never entered
- [ ] **AG-18.5 / one path** — strategy-triggered and on-demand streams are `Kind()`-for-`Kind()` equal excluding fresh `RunID`/`TurnID`
- [ ] **AG-18.5 / typed refusal** — a mid-turn demand is refused typed, emits **no** event, and the in-flight turn completes unaffected
- [ ] **Bites RED-recorded before GREEN**: single-direction widening; commit-before-return; dropped route from `S-HIS-030`'s enumeration; queueing on-demand instead of refusing
- [ ] **Deliverable 0 shipped** — `R-HIS-001`, `R-HIS-004`, `R-HIS-005`, `R-HIS-009`/`S-HIS-080` + `L2C-07`, `R-CTX-003`/`S-CTX-005`, `R-LSK-004`'s release, and the three back-annotations all present; `sdd-verify` opens each cited line
- [ ] **`S-HIS-080`'s stale "7"** corrected, re-scoped off the literal count
- [ ] **No new `EventKind`, `CostLabel` or `TurnOutcome`**; `event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `compaction_events.go`, `go.mod`, `go.sum` and all of `src/ai/` byte-unchanged; the every-kind guard passes at 25
- [ ] **`h.History` pointer unchanged across `Run`** — `harness_test.go:1013`'s pin still green, untouched

---

## Proposal question round

Execution mode is `auto`, so these were not asked interactively. They are recorded here because each changes the shape of the product, not the harness. **Answer any of them before `sdd-design` and the recommendation above moves.**

1. **Is discarding compacted entries irreversibly acceptable for v1?** This change ships no journal, no undo, no pre-compaction snapshot. Today nothing persists, so the blast radius is one in-memory run. Assumed: **yes, acceptable, and stated as a named consequence in the spec**. If not, a snapshot obligation belongs in scope now, not retrofitted.
2. **May a compaction's `EntryID`s renumber?** A3 argues renumbering is *forced* by `R-HIS-005`'s ordinal rule, and that `TurnID` is the durable handle. Assumed: **yes, and the spec says so explicitly** so no Layer 3 consumer stores an `EntryID` across a compaction.
3. **Should a compaction failure be visible to the caller beyond the event stream?** Today it is stream-only: the run continues and the caller's return value is unchanged. Assumed: **stream-only** (`L2C-03`: the stream is the only upward contract). If a Layer 3 UX needs a synchronous signal, that is a different requirement.
4. **Should the on-demand entry point ever queue?** The charter says "refused typed — never queued silently." Assumed: **never queues, in any form**, including a "queue but tell the caller" variant, which the word *silently* might otherwise be read to permit.
5. **May compaction run more than once per turn boundary?** Nothing in the charter forbids a second compaction immediately after a first. Assumed: **at most one per turn boundary**, since `R-CTX-001` consults the strategy exactly once per logical turn and the verdict is the only request carrier.
