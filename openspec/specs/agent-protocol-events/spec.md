# Spec — Permission, cost, delegation, and compaction event families (`agent-protocol-events`)

> **Change**: `cachicamas-agent-protocol-events` · **Milestone**: AG-06 (Layer 2 Wave 1, **closing**) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-06--add-permission-cost-delegation-and-compaction-event-families), `0003:602-712`
> **Nodes**: AG-06.1 `[leaf]` (permission) · AG-06.2 `[leaf]` (cost) · AG-06.3 `[leaf]` (delegation) · AG-06.4 `[leaf]` (compaction) · AG-06.5 `[guard]`
> **Closes**: R-04 — the four families [doc 0001 § 4.3](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md#43-the-agent-event-envelope) marks absent (G1, G10, G7, G3's visible halves)
> **Traces to**: doc 0001 § 7 G1/G3/G7/G10; ADR 0005 § D1, D3, D4
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario independently verifiable.
> **Identifier convention**: requirements `R-APE-0NN`, scenarios `S-APE-0NN`. Append-only. Distinct from `R-AEV-`/`S-AEV-` (AG-04 envelope), `R-AMT-`/`S-AMT-` (AG-05 message/tool), `R-AGE-`/`S-AGE-` (AG-01 delivery).
> **Scenario count** (stated identically across all downstream artifacts — AG-04 W9): **9 charter → ~14–22 spec + 4 bites**. This spec lands at **15 spec + 4 bites = 19 total scenarios**.

## Coverage

| Charter scenarios | Requirements | Spec scenarios | Bites |
|---|---|---|---|
| **9 of 9** | 9 added (`R-APE-001`–`009`) | **15 of 14–22** | **4** (S-APE-081, S-APE-082, S-APE-083, S-APE-084) |

Charter → spec mapping: AG-06.1 charter 1 → `R-APE-001`; charter 2 → `R-APE-002`; charter 3 → `R-APE-003`. AG-06.2 charter 1 → `R-APE-004`; charter 2 → `R-APE-005`. AG-06.3 charter 1 → `R-APE-006`. AG-06.4 charter 1 → `R-APE-007`; charter 2 → `R-APE-008`. AG-06.5 guard bite → `R-APE-009` (cross-cuts all four families).

## Purpose

Register the four Layer 2 event families doc 0001 § 4.3 marks absent — permission (`VL2-EVT-06`, G1), cost (`VL2-EVT-07`, G10), delegation (`VL2-EVT-08`, G7), compaction (`VL2-EVT-09`, G3's visible halves) — as constructible and validated event kinds, on the AG-04/AG-05 substrate, with **no edit** to `event_descriptor.go`, `stream_check.go`, `failure.go`, or `sequence.go`. AG-06 is the **first** kind-set to exercise `CardinalityAtMostOne` (the AG-04.3-reserved seam at `event_descriptor.go:103-120`) and the **first** non-`NewDelegatedRunStart` consumer of the parent identifier field (`event.go:362-366`).

AG-06 closes Layer 2 Wave 1. After AG-06 merges, the registry holds **25 kinds** (4 AG-04 + 11 AG-05 + 10 AG-06), the scope-fence `S-AEV-090` retightens to "exactly 25", and AG-07 onward emit events against the four families already constructible.

## Requirements

### R-APE-001 — Permission: a decision-required event carries the call identity, tool name, and arguments

A `permission_decision_required` event payload MUST carry the tool call identity (`ai.ToolCallID`), the tool name, and the arguments as distinct fields readable from an external package — the surface a frontend needs to ask "may this call proceed?". The kind is distinct from other permission kinds (G1, `0003:631-635`).

#### Scenarios

- **S-APE-001** — Given a `permission_decision_required` event for one scheduled call, when an external test package inspects it, then the call identity, tool name, and arguments are readable as distinct fields; the kind is distinct from `permission_decision_made` and `permission_resolution_remembered`.

### R-APE-002 — Permission: a decision-made event carries the full outcome vocabulary as typed values

A `permission_decision_made` event MUST carry a typed `PermissionOutcome` whose values are `AllowOnce`, `AllowAlways`, `Deny`, or `ModifyInput` — closed enum, zero not a member, mirroring AG-04's `RunOutcome` and AG-05's `ToolOutcome` patterns. The `ModifyInput` outcome MUST carry the modified arguments as a payload field. The `Deny` outcome MUST carry a typed `*agent.Failure` (reusing `R-AEV-008`'s wrap surface) — no code path assigns meaning to a message string (`0003:638-642`).

#### Scenarios

- **S-APE-010** — Given `permission_decision_made` events constructed with each of `AllowOnce`, `AllowAlways`, `Deny`, and `ModifyInput`, when an external test package inspects them, then each outcome is reachable as a distinct typed value; the enum is closed (zero is not a member); no payload-convention inference is required to tell them apart.
- **S-APE-011** — Given a `permission_decision_made` constructed with `ModifyInput`, when inspected, then the modified arguments are present as a distinct payload field — the ruling carries the editorial change.
- **S-APE-012** — Given a `permission_decision_made` constructed with `Deny`, when inspected, then it carries a typed `*agent.Failure` reachable through the typed-failure surface; the failure is not asserted by parsing a message string.

### R-APE-003 — Permission: resolution-remembered is a distinct kind with at most one entry per stream per tool name

A `permission_resolution_remembered` event MUST be a distinct kind from `permission_decision_made` — a fact about future calls the session log needs, or it cannot explain why later calls were never asked (`0003:643-647`). It MUST declare `CardinalityAtMostOne`: at most one such event per stream per tool name. The per-stream cardinality and per-tool-name cardinality are asserted separately.

#### Scenarios

- **S-APE-020** — Given a `permission_resolution_remembered` event, when an external test package inspects it, then it is a distinct kind from `permission_decision_made`; its payload carries the tool name and the remembered outcome as separate fields.
- **S-APE-082** — **(bite)** Given a hand-built stream with two `permission_resolution_remembered` events for the same tool name, when the validator checks it, then it is REJECTED naming the `CardinalityAtMostOne` rule and the second offending position — the per-tool-name cardinality is asserted mechanically, not by comment. RED-recorded BEFORE the registry ships.

### R-APE-004 — Cost: payload is token-only, mirroring Layer 1 usage; no field that could carry money

A `cost_turn` event MUST carry per-turn token figures (`inputTokens`, `outputTokens`, `cacheReadTokens`, `cacheWriteTokens`, `reasoningTokens`) and a `cost_session` event MUST carry cumulative figures of the same fields — mirroring Layer 1's usage taxonomy. Neither payload MAY carry a field that could carry money, currency, or price; the verdict on `doc 0001 § 4.3` vs ADR 0005 § D4 wins (`0003:613`, `0003:656-661`).

`cost_turn` is `Placement: PlacementTurn`; `cost_session` is `Placement: PlacementRun` (a session marker spans the whole run).

**AG-16 adds a per-figure presence discriminator to both payloads — beside the figures, never inside them.** Each of the five figures MUST be readable **together with a discriminator stating whether Layer 1 reported it**, mirroring Layer 1's own house idiom `TokenCount.Count() (int64, bool)` (`usage.go:63-65`): `(0, absent)` and `(0, reported)` MUST be observably different from an external test package, and the paired accessors are the **required** path for judging presence — `Figures()` survives unchanged for this capability's byte-pinned callers and MUST NOT be read to infer absence, nor removed to satisfy this clause. The granularity is **per figure**, because `ai.Usage`'s presence is per field (`usage.go:44-47`): a whole-record flag would report four unreported figures as present zeros for a usage such as `ai.Usage{Input: ai.Tokens(100)}`, which is exactly the invented zero the charter's acceptance text forbids (`0003:1535`).

Three consequences are stated rather than left to be inferred:

- **`CostFigures` is closed and stays closed.** The discriminator MUST NOT be added as a field, a mask, or a bit inside `CostFigures`. That type MUST remain **byte-unchanged**: five fields, in that name order, each `uint64`. `S-APE-083` walks `reflect.TypeOf(agent.CostFigures{})` (`cost_events_test.go:199-234`) and inspects **nothing else**, so it stays green with `cost_events_test.go` **byte-unchanged**. This requirement's "mirroring Layer 1's usage **taxonomy**" clause is about *which buckets exist* — the five-way split — and never obliged presence to live inside the figures; the acceptance criterion independently obliges presence to be observable *on the event*. Both hold simultaneously only when presence travels beside the figures, which is what AG-16 ships.
- **The token-only pin is strengthened, never weakened.** AG-16 adds **zero** fields to `CostFigures` and no field of any name to any payload that could carry money, currency or price. `S-APE-083`'s forbidden-substring scan (`cost_events_test.go:207-224`) MUST pass unchanged. The money question stays Layer 3's (`0003:1537`).
- **The existing constructors keep their exact signatures.** `NewCostTurn` and `NewCostSession` (`cost_events.go:181-193`, `:248-257`) MUST NOT change shape — `cost_events_test.go` calls them and MUST stay byte-unchanged — and a payload built through them MUST report every figure **reported**: a caller handing five plain figures is asserting five figures. The presence-preserving construction path MUST therefore be a **sibling**, not a widening of the existing constructors (the AG-14 `typedCancellationFailure` sibling precedent). A zero-value payload — what a wrong-kind accessor returns (`cost_events.go:199-202`) — MUST report every figure absent, which is the coherent reading rather than a special case.

(Previously: the requirement stated the five token figures and the two placements, with no statement of how a figure Layer 1 never reported is distinguished from a figure Layer 1 reported as zero — so the payloads could express only "five numbers", and any emitter would have had to invent zeros for absent figures.)

#### Scenarios

- **S-APE-030** — Given a `cost_turn` event, when inspected, then it carries `inputTokens`, `outputTokens`, `cacheReadTokens`, `cacheWriteTokens`, and `reasoningTokens` as distinct fields; no field carries money, currency, or price.
- **S-APE-031** — Given a `cost_session` event, when inspected, then it carries the same five fields as cumulative figures; its `Placement` is `PlacementRun`, not `PlacementTurn`.
- **S-APE-083** — **(bite)** Given a scratch cost payload that adds a `money *decimal` field, when the parsed row is diffed against the committed payload shape, then it FAILS naming the unknown field — the token-only contract is asserted mechanically, not by doc-phrase. RED-recorded. *(AG-16 note: **unchanged and untouched**. This bite reflects over `CostFigures` alone, which AG-16 leaves byte-unchanged, so it holds at AG-16 without amendment and its implementing test file is byte-unchanged. It was not re-run against a widened type and it was not relaxed.)*
- **S-APE-085** — **AG-16 presence is per figure and lives beside the figures.** Given a `cost_turn` and a `cost_session` each built through the presence-preserving path from an `ai.Usage` reporting only its input figure, when each is inspected from an external test package, then the input figure reads as reported with its magnitude and the other four read as **absent**; none of the four reads as a present zero; a payload built from an `ai.Usage` whose input figure is `ai.Tokens(0)` reads that figure as **reported at zero** and is observably different from one built from the empty `ai.Usage`; a payload built through the unchanged public constructors reads every figure as reported; the zero-value payload reads every figure as absent; and when `CostFigures` is reflected over, then it still carries exactly five `uint64` fields in their committed name order. Owned jointly with `R-CST-002`; the emission behavior is `agent-cost-events`', and this scenario exists so a regression in the payload surface fails **this** capability's suite too.

### R-APE-005 — Cost: every figure is labelled estimate or final

Every cost figure MUST carry a `CostLabel` discriminator whose value is `Estimate` or `Final`. The discriminator MUST be a typed value, not a free-form string, and the enum MUST stay closed at those two members.

**AG-16 scopes the two members, because the axis they range over was under-determined and read literally is unsatisfiable at turn scope.** The original wording — "any figure emitted **before the stream's final usage update**" — describes a **wire-level** phenomenon: on adaptive-reasoning models the reasoning count arrives only on the last streamed usage update (`doc 0001 § 7 G10`, `0001:719-732`, echoed verbatim at `cost_events.go:96-103`). Layer 1 absorbs that phenomenon **entirely** before Layer 2 sees it: `ai.Completion` is the sole usage carrier, registered `CardinalityAtMostOne, Terminal: true` (`completion.go:92`, `ai/event.go:167-170`), and the decoder folds every wire-chunk update into it (`openaicompat/stream_state.go:328`). Within one turn there is therefore **no earlier figure to label `Estimate`**.

The axis is consequently **run-scoped**, on the charter's own authority — its out-of-scope line already assigns the estimate-labelled event to the *incremental running* figure (`0003:1537`):

| Scope | Label | Meaning |
|---|---|---|
| `cost_turn` | **always `Final`** | per-turn usage is complete whenever it is known at all, absence included — an absent figure is a *final* reading of "nothing was reported", not an estimate |
| `cost_session` emitted as a running total while the run continues | `Estimate` | the run has not concluded; more tokens may follow |
| `cost_session` emitted at the run's terminal | `Final` | corrects every earlier estimate |

**The presence axis and the label axis are orthogonal and MUST NOT collide.** Absence MUST NOT be expressed as a third `CostLabel` member, and a payload's label MUST NOT be read as evidence about whether its figures were reported. The two questions are answered by two independent surfaces (`R-APE-004`).

The emission rules that give these labels their run-time meaning — which events exist, in what order, on which run closes — are owned by [`../agent-cost-events/spec.md`](../agent-cost-events/spec.md) (`R-CST-001`, `R-CST-005`, `R-CST-006`) and are **not** re-homed here. This requirement owns the vocabulary and its scoping; that capability owns the protocol.

(Previously: the requirement stated the label axis in wire terms — "any figure emitted before the stream's final usage update MUST be labelled `Estimate`; the figure emitted on the final usage update MUST be labelled `Final`" — which, read at turn scope against the shipped Layer 1 substrate, describes a figure that cannot exist, and which said nothing about how the label axis relates to the presence axis.)

#### Scenarios

- **S-APE-040** — Given `cost_turn` payloads constructed with each `CostLabel` member, when an external test package inspects their labels, then the payload constructed with `Estimate` reads back `Estimate` and the one constructed with `Final` reads back `Final`; the label is a typed value, not a free-form string; and the enum is closed at those two members. *(Amended by AG-16: the scenario's Given previously spoke of figures "emitted before and after the stream's final usage update", which the amended requirement establishes cannot exist at turn scope. The observable it actually pins — that both members are constructible and read back as distinct typed values — is unchanged, so its implementing test `TestCost_EveryFigureLabelledEstimateOrFinal` (`cost_events_test.go:121`) passes **byte-unchanged**. The wording is corrected here rather than left to certify a mechanism the amended requirement denies.)*
- **S-APE-086** — **AG-16 the label axis is run-scoped and orthogonal to presence.** Given a recorded harness run stream carrying cost events, when every `cost_turn` on it is inspected, then each carries `Final` and none carries `Estimate`, including the one whose figures are entirely absent — proving absence is a final reading rather than an estimate; and when the run's `cost_session` events are inspected, then those emitted while the run continued carry `Estimate` and the one at the run's terminal carries `Final`. Owned jointly with `R-CST-001` / `R-CST-005`.

### R-APE-006 — Delegation: subagent events are parent-linked via the envelope's parent identifier

A `subagent_started` event and a `subagent_ended` event MUST set the envelope's parent identifier (`event.go:362-366`) to the delegating run's `RunID` — making the delegation tree walkable. AG-06.3 is the **first** non-`NewDelegatedRunStart` consumer of the parent field. The absence of a parent on a non-delegated event MUST remain distinguishable (R-AEV-003, `0003:675-679`).

#### Scenarios

- **S-APE-050** — Given a `subagent_started` event for a delegated run, when an external test package reads its parent identifier, then the parent's `RunID` is returned and `Parent()` returns `(parentID, true)`.
- **S-APE-051** — Given a `subagent_ended` event for the same delegated run, when its parent identifier is read, then it returns the same `(parentID, true)` as `subagent_started` — the pair is parent-linked consistently.
- **S-APE-052** — Given a non-delegated event from AG-04 or AG-05, when its parent identifier is inspected, then `Parent()` returns `(RunID(""), false)` — the "no parent" state is unambiguous, not a zero-value trick.

### R-APE-007 — Compaction: a finished event carries the replaced span and summary identity

A `compaction_finished` event MUST carry a span identity `[startTurnID TurnID, endTurnID TurnID]` (the turn bracket — turns are the protected unit, not per-lane sequence) and a `summaryID` identifying the persisted summary — enough for a session log to record the operation (`0003:687-691`, `doc 0001 § 7 G3` "protect recent turns").

#### Scenarios

- **S-APE-060** — Given a `compaction_finished` event emitted after a `compaction_started`, when inspected, then it carries both `startTurnID` and `endTurnID` as typed `TurnID` values forming a non-empty bracket; the span is identified by turn identity, not by sequence.
- **S-APE-061** — Given the same `compaction_finished` event, when inspected, then it carries a `summaryID` distinct from the run identity, so a session log can persist the summary separately from the run record.

### R-APE-008 — Compaction: a failed event is Terminal:false and distinct from finished

A `compaction_failed` event MUST declare `Terminal: false` — honoring `doc 0001 § 7 G3` "survive interruption". The engine accepts follow-on events (a subsequent `compaction_started` MAY re-emit) but does NOT synthesize recovery. The event MUST be a distinct kind from `compaction_finished` and MUST carry a typed `*agent.Failure` (`0003:693-697`).

#### Scenarios

- **S-APE-070** — Given a `compaction_failed` event, when inspected, then it is a distinct kind from `compaction_finished`; it carries a typed `*agent.Failure` reachable through the typed-failure surface; its `Terminal` declaration is `false` (not the zero value — the descriptor row is explicit, per AG-05 S1 carry-forward).
- **S-APE-084** — **(bite)** Given a hand-built stream where a `compaction_started` follows a `compaction_failed`, when the validator checks it, then the stream is ACCEPTED — the engine honors `Terminal: false`. RED-recorded BEFORE the registry ships.

### R-APE-009 — All 10 new kinds register; the constructibility guard extends to 25; the scope-fence bites on the 26th scratch kind

All 10 new kinds (3 permission + 2 cost + 2 delegation + 3 compaction) MUST register following the seven-step "adding a kind" procedure documented at `event_descriptor.go:13-46`, with `Terminal: false` written explicitly on every row (AG-05 S1 carry-forward). The every-kind-constructible guard MUST iterate over all 25 registered kinds (4 AG-04 + 11 AG-05 + 10 AG-06) through the public surface. The scope-fence `S-AEV-090` retightens from "exactly 15" to "exactly 25" in the **same commit** as the new kinds land. The forbidden-names list (`event_registry_test.go:326`) retires — permission, cost, delegation, and compaction are AG-06's own.

#### Scenarios

- **S-APE-080** — Given the every-kind-constructible guard, when it runs, then it constructs at least one instance of every registered kind (25 total: 4 AG-04 + 11 AG-05 + 10 AG-06) through the public surface from an external test package; the `Terminal: false` declaration is asserted on every AG-06 row, not relied on as the zero value.
- **S-APE-081** — **(bite)** Given a 26th scratch kind planted following the documented seven-step procedure, when the guard runs, then it FAILS by count before the name scan runs — the scope-fence bites before AG-07 lands. RED-recorded; the scratch kind is absent from the merged diff.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-APE-001** | External-package verifiability: every scenario verifiable by `cd backend/agent && make test`. Every behavioral test in `package agent_test` or another external package, per NFR-AEV-001. |
| **NFR-APE-002** | Boundary guards stay green untouched: AG-03's `import_boundary_test.go` and `ambient_authority_test.go` MUST pass with zero changes to their own logic. |
| **NFR-APE-003** | Determinism and race cleanliness: every test added MUST be deterministic, hermetic, and pass under `-race`. |
| **NFR-APE-004** | Substrate byte-unchanged: `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml` are byte-unchanged. AG-06's value is in the substrate's extensibility, demonstrated for the third time (after AG-04.4 + AG-05). |
| **NFR-APE-005** | Review budget: single PR under pre-authorised `size:exception` against 1000-line budget, forecast 1500–2400 lines (AG-05 precedent: 2479 actual). |
| **NFR-APE-006** | Doc-contract guarded: the `L2C-04` row's family table is extended to include the four protocol families in the same commit as the `L2C-06` row amendment (per `R-AGP-002` closed amendment rule). |

## Explicit non-requirements

- **No edit** to `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go` — AG-04's rule engine stays untouched; AG-04.4's `S-AEV-092` and AG-05's `R-AEV-012` proven path extend to AG-06. AG-06.3's `CardinalityAtMostOne` opt-in is via the descriptor row, not a new descriptor field. *(Still true at AG-16, which edits none of the four. Still true at AG-18, which edits none of the four either — its compaction bracket is opened through the existing exported turn-event constructors precisely so `stream_check.go`'s turn-placement rule, `stream_check.go:161`, is satisfied with the validator byte-unchanged.)*
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit. No Layer 1 read beyond what AG-05 already uses (`ai.MessageID`, `ai.FailureCategory`). *(AG-16 reads `ai.Usage` and `ai.TokenCount` in addition — a read of an already-imported Layer 1 package, adding no dependency and editing nothing under `backend/agent/src/ai/`. AG-18 reads `ai.Message`, `ai.Completion` and `ai.Usage` in addition, on the same terms: no dependency added, and its diff under `backend/agent/src/ai/` is empty.)*
- **Live emission** — **AG-10, AG-16, AG-18, AG-19**. AG-06 ships events constructible; the mechanisms that emit them land later. **The cost family's half is CLOSED by AG-16**, which is `R-APE-004`/`R-APE-005`'s first emitter (`agent-cost-events`, `R-CST-001`, `R-CST-005`). The permission family's half closed at AG-10. The compaction and delegation families remain unemitted, owned by AG-18 and AG-19 respectively — AG-16 emits **neither**, and no acceptance line may be written as if it did. **The compaction family's half is CLOSED by AG-18**, its first production emitter after twelve milestones constructible-but-unemitted. Three facts are recorded here because a later reader will ask them of the protocol rather than of compaction: *(1)* **the family is emitted inside its own dedicated turn bracket**, opened and closed through the existing exported turn-event constructors — required rather than stylistic, because the family is registered `PlacementTurn` and the validator rejects such an event where no turn is open (`stream_check.go:161`, verified verbatim), and the validator is frozen; *(2)* **`compaction_events.go` is byte-unchanged**, as are `event.go`, `event_descriptor.go` and `event_registry_test.go` — AG-06's shape needed no amendment to accept its first caller, and in particular `CompactionSpan` was **not** made optional, because `R-APE-007`'s turn-identifier axis is precisely what survives a compaction; *(3)* **exactly one of finished or failed is emitted per operation**, never both, and the finished payload's span plus summary identity are together sufficient for a consumer holding only the stream to name the replaced turn brackets and locate the summary entry (`R-CMP-009`). **The delegation family remains unemitted and remains AG-19's**; AG-18 emits nothing from it, and no acceptance line may be written as if it did.
- **Permission policy** (asking, rule sets, mode flags) — the **port** exists at `doc 0001 § 5.1` (`PermissionPolicy`); the implementations are AG-10.
- **Price table** (`doc 0001 § 5.1` `PriceTable`) — **AG-16**. The cost payload is token-only on purpose; money joins the stream as Layer 3 enrichment. **Owner corrected at AG-16: this is NOT AG-16's.** AG-16 emits the token-only payloads and ships **no** money, currency or price field at any scope — its own `R-CST-007` states that as a requirement, and `S-APE-083` enforces it mechanically. The price table is **Layer 3's**, the charter's own out-of-scope line naming it so (`0003:1537`). The row is re-owned rather than deleted so the correction is visible.
- **Subagent harness mechanics** — **AG-19**. AG-06.3 ships the events; the harness is AG-19's. *(Still true at AG-16: parent-scoped cost aggregation is AG-19's by the charter's Goal line, `0003:1533`. Still true at AG-18, whose compaction spend joins the run's own cumulative and nothing above it.)*
- **Compaction strategy** — **AG-16**. AG-06.4 ships the events; the summarizer is AG-16's. **Owner corrected at AG-16: this is NOT AG-16's either.** AG-16's charter is cost and usage emission (`0003:1527-1560`) and it ships no summarizer, no compaction check and no compaction emission. The strategy is **AG-18**'s, with AG-17 inserting the check at the turn boundary (`agent-run-driver/spec.md:324`). Corrected here rather than propagated, following the pointer-correction precedent at `agent-run-driver/spec.md:328`. **CLOSED by AG-18 — with the strategy/mechanism split stated, because the row's word "strategy" invites the wrong reading.** AG-18 ships the **mechanism** a strategy drives: the summarising call, the surgery, the emission, the atomicity and the on-demand door. It does **not** ship a strategy that decides *when* to compact, nor the instruction's content — both are **Layer 3's**, injected, and AG-18's production code MUST NOT author instruction text on any path (`R-CMP-002`, charter `0003:1662`, `0003:1665`).
- **Test-convenience wrappers** in `backend/agent/src/agenttest` — **AG-23**. *(Still true at AG-16, which uses the test-local `errorProvider` wrapper precedent and leaves `agenttest` byte-unchanged. AG-18 adds fixture helpers to `agenttest` for its own scenarios — a mis-aligned-cut transcript builder and a marks-bearing fixture — which are fixtures for this milestone's tests, not the general convenience wrappers this row defers to AG-23.)*
- **CI** — every gate runs when a human runs `make test` in `backend/agent/`.

## Evidence discipline

`openspec/config.yaml` declares `apply.tdd: true`; strict TDD active (`openspec/AGENTS.md`).

- All five leaves are behavior, so all five are RED-first.
- **`R-APE-003` closes only on its recorded bite** (`S-APE-082`): the per-tool-name `CardinalityAtMostOne` semantic is asserted mechanically, mirroring AG-04's `R-AEV-009` bite pattern.
- **`R-APE-004` closes only on its recorded bite** (`S-APE-083`): the token-only contract is a structural pin on the payload shape, not a doc-phrase check (AG-04 W7 carry-forward inverted — mechanical pin instead of phrase check).
- **`R-APE-008` closes only on its recorded bite** (`S-APE-084`): the `Terminal: false` declaration is honored by the engine (`stream_check.go:173-175`).
- **`R-APE-009` closes only on its recorded bite** (`S-APE-081`): the 26th scratch kind fails by count RED, scratch file absent from the merged diff.
- **`Terminal: false` MUST be stated explicitly** in every AG-06 descriptor row (AG-04 W3 + AG-05 S1 carry-forward).
- **Reflection pins verify field name AND field type** (AG-04 W4 + AG-05 S2 carry-forward) — `parent RunID` typed as `RunID`, not as a proxy under a different name.
- **Bites assert slice index, not sequence value** (AG-04 W1 + W2 carry-forward).
- **`make vuln-check` is NOT in `make all`** (Engram `#2944`) — apply and verify MUST run it explicitly.
- **`golangci-lint cache clean` before each lint gate** (AG-04 `6c821c0a` precedent).
- **All six `sdd-attempt settle` flags** — `--outcome`, `--harness-disposition`, `--evidence-revision`, `--diagnosis`, `--cleanup-evidence`, `--process-evidence` (Engram `#2961`).
- **`sdd-archive` runs in the worktree**, commits to the PR branch (Engram `#2963`).
- **Test-driven-development skill gap** (Engram `#2962`) — discipline forwarded inline from `openspec/AGENTS.md`; load `sdd-verify/strict-tdd-verify.md` for verification work.

## Acceptance criteria

The contract holds when:

1. Every `S-APE-001`…`S-APE-084` has recorded evidence.
2. `cd backend/agent && make test`, `make lint` (after `cache clean`), `make build`, and `make vuln-check` are all green.
3. `backend/agent/go.mod` and `go.sum` byte-unchanged.
4. `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go` byte-unchanged.
5. The every-kind-constructible guard constructs all 25 kinds (4 + 11 + 10); scope-fence bites on a 26th.
6. All four bites (`S-APE-081`, `S-APE-082`, `S-APE-083`, `S-APE-084`) RED-recorded with the failing output captured in `apply-progress.md`.
7. The 9 charter Gherkin scenarios (`0003:631-696`, `0003:705-709`) are covered; none reduced.
8. AG-03's two boundary guards pass with zero changes to their own logic.
9. The `L2C-06` row is added to `doc.go` and `expectedLayer2ContractRows` in the same commit (per `R-AGP-002` rule).
10. `permission_decision_made` carries 4 typed outcomes (zero not a member); `permission_resolution_remembered` declares `CardinalityAtMostOne`; `compaction_failed` declares `Terminal: false` and is distinct from `compaction_finished`; `compaction_finished` carries `[startTurnID, endTurnID]`; `subagent_started` and `subagent_ended` set `Parent()` on the envelope.

## Traceability

| Requirement | Charter node | Register rows | Charter scenario (`0003`) |
| --- | --- | --- | --- |
| `R-APE-001` | AG-06.1 | `VL2-EVT-06` (decision_required) | `:631-635` |
| `R-APE-002` | AG-06.1 | `VL2-EVT-06` (decision_made, typed `PermissionOutcome`) | `:638-642` |
| `R-APE-003` | AG-06.1 | `VL2-EVT-06` (resolution_remembered, `CardinalityAtMostOne`) | `:643-647` |
| `R-APE-004` | AG-06.2 | `VL2-EVT-07` (cost token-only) | `:656-661` |
| `R-APE-005` | AG-06.2 | `VL2-EVT-07` (cost labelled estimate/final) | `:662-665` |
| `R-APE-006` | AG-06.3 | `VL2-EVT-08` (parent-linked delegation) | `:675-679` |
| `R-APE-007` | AG-06.4 | `VL2-EVT-09` (compaction_finished span) | `:687-691` |
| `R-APE-008` | AG-06.4 | `VL2-EVT-09` (compaction_failed `Terminal: false`) | `:693-697` |
| `R-APE-009` | AG-06.5 (cross-cut) | `VL2-EVT-06`…`VL2-EVT-09` (registry 25) | `:705-709` |

All 9 charter Gherkin scenarios in 5 leaves are represented; none is reduced. Scenario count stated identically with the proposal (`9 charter → ~14-22 spec + 4 bites`); this spec lands at **15 spec + 4 bites = 19 total**, within the forecast range.
