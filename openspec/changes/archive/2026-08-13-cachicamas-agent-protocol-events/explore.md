# Exploration — `cachicamas-agent-protocol-events` (AG-06)

> Milestone **AG-06** (Layer 2 Wave 1, the closing milestone) of [doc 0003](../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-06--add-permission-cost-delegation-and-compaction-event-families), lines 602–712. SDD change slug: `cachicamas-agent-protocol-events`. Artifact store: hybrid (Engram topic key `sdd/cachicamas-agent-protocol-events/explore` + OpenSpec filesystem).
>
> This is the **third** Wave-1 milestone on the `cachicamas-agent-event-envelope` substrate. The other two (AG-04, AG-05) are MERGED (PRs #163, #164; main HEAD `6b4a3468`).

---

## 1. Identity

| Field | Value |
|---|---|
| **Slug** | `cachicamas-agent-protocol-events` |
| **Milestone** | AG-06 (Layer 2 Wave 1, milestone 6 of 24; doc 0003 § AG-06, lines 602–712) |
| **Branch** | `feat/agent-layer2-wave1-ag06` |
| **Worktree** | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave1-ag06` |
| **Store** | HYBRID (Engram `sdd/cachicamas-agent-protocol-events/explore` + filesystem `openspec/changes/cachicamas-agent-protocol-events/explore.md`) |
| **Mode** | automatic (gatekeeper between phases; no user interruption) |
| **Strict TDD** | enabled (`openspec/AGENTS.md` § "Strict TDD is on"; `openspec/config.yaml` `apply.tdd: true`) |
| **Review budget** | 1000 lines, exception pre-authorized open-endedly (braejan's AG-04 standing instruction, Engram obs #2957 archive-report § "Size and Complexity Record" and AG-05 precedent) |
| **Closes** | R-04 — the four families [doc 0001 § 4.3](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md#43-the-agent-event-envelope) marks **absent** from the v1 design: G1, G10, G7, G3's visible halves (`0003:604`) |
| **Depends on** | AG-04 (merged) |
| **Parallel with** | AG-05 (already merged; PR #164) |
| **Out of scope** | Emission (AG-10, AG-16, AG-18, AG-19 respectively, per `0003:612`) |

---

## 2. Context

AG-06 is the **last Wave-1 milestone** for Layer 2. AG-04 shipped the envelope and validator with the two lifecycle families (run, turn); AG-05 extended the registry to 15 kinds with the message and tool families; AG-06 closes the loop by adding the **four remaining families** doc 0001 § 4.3 marks absent, lifting the registry to **19 kinds** total (4 + 11 + 4 = 19).

### What AG-04 shipped (constraints on AG-06)

| Substrate piece | Lives at | AG-06 use |
|---|---|---|
| Envelope + derived kind + per-lane `LaneStamper` | `backend/agent/src/agent/event.go` (`event.go:328-336`), `sequence.go` | Reuse unchanged; AG-06's events inherit the `{payload, seq, run, turn, hasTurn, parent, hasParent}` shape |
| Six → seven-step "adding a kind" procedure (post-AG-05 `c203f25c` fix) | `event_descriptor.go:13-46` (file comment) | Follow exactly. **Step 5a (`Terminal: false` explicit)** is now part of the procedure (S1 lesson from AG-05 verify) |
| `PlacementTurn` seam | `event_descriptor.go:88-100` (proven by `stream_check.go:161`) | AG-06's permission/cost/delegation/compaction kinds MAY need a new `Placement` value; current options are `PlacementRun` (zero) or `PlacementTurn`. **A new placement is an AG-06 ADR decision, not silently added** |
| `CardinalityAtMostOne` seam (reserved for AG-06) | `event_descriptor.go:103-120` (referenced by `S-AEV-112`, `S-AEV-090`'s retightening notes) | **AG-06's first exercise.** Several AG-06 kinds are "at most once per stream" candidates (e.g. `decision_required` for a single tool call, `subagent_ended` for one delegated run, `compaction_started`/`compaction_finished` for one compaction per run); the seam exists, the engine reads it (`stream_check.go:166-171`), and AG-06 opts kinds in via the descriptor row |
| `Terminal` field | `event_descriptor.go:142-144`, read by engine at `stream_check.go:173-175` (post-AG-04 fix `c203f25c`) | AG-06's `compaction_failed` MAY be `Terminal: true` if the engine's "nothing follows the terminal" rule should apply; otherwise leave `false`. W3 latent-trap guard is held by the engine now |
| `BracketRoleNone` for non-bracket kinds | `event_descriptor.go:46-48` | All four AG-06 families are `BracketRoleNone` (none open or close a bracket — they live inside one) |
| Typed-failure wrap | `failure.go` (80 lines) | AG-06's `decision_made` (deny outcome) and `compaction_failed` MAY carry a typed `*Failure`; reuse `R-AEV-008`'s surface, no new category |
| Stream-contract validator | `stream_check.go` (193 lines, production-exported) | **No edit**. AG-04.4's `S-AEV-092` extensibility experiment + AG-05's `R-AEV-012` proven path both hold. AG-06.5's guard extension is a **scope-fence retightening + witness-table extension**, not a validator edit |
| Parent identifier field | `event.go:362-366` (`Event.Parent() (RunID, bool)`) | **AG-06's first user.** AG-06.3 delegation events MUST carry the parent run identity; the field exists from AG-04.1's R-AEV-003 (`0003:450-453`). Note `NewDelegatedRunStart` is currently the ONLY public door that sets parent (`run_events.go:60-76`) — AG-06.3's `NewDelegatedRunStart`-equivalent needs to be defined for its own kinds, OR the existing constructor is reused |
| Doc-matrix guard `L2C-04` row + iterator | `doc.go:23`, `doc_contract_guard_test.go:56-62` | The guard iterates **every registered kind** (`TestLayer2DocContract_MatchesTheCommittedTable`); AG-06 must verify the guard's row count and prose extension are honored in the same commit. **No new guarded row proposed yet** — AG-05 added `L2C-05` only for reconstruction; AG-06 has no equivalent cross-cutting criterion that warrants a new row |
| Every-kind-constructible guard | `event_registry_test.go` | **Extends.** Witness table grows from 15 → 19. Scope-fence `S-AEV-090` retightens from "exactly 15" to "exactly 19" (4 AG-04 + 11 AG-05 + 4 AG-06) in the **same commit** as the new kinds. `TestEventKinds_ScopeFence_BitesByCountOnSixteenthKind` (`event_registry_test.go:411-425`) must be updated to "want 19" with a 20th-scratch bite scenario |
| Identifier convention | `openspec/specs/agent-event-envelope/spec.md:9` (`R-AEV-`/`S-AEV-`); `openspec/specs/agent-message-tool-events/spec.md:6` (`R-AMT-`/`S-AMT-`) | **AG-06 needs a new prefix.** Natural pick: `R-APE-`/`S-APE-` (matches slug `cachicamas-agent-protocol-events`; "P" = protocol). Distinct from AG-01 (`R-AGE-`), AG-04 (`R-AEV-`), AG-05 (`R-AMT-`). The proposal must commit to a prefix, not defer |
| Forbidden names list | `event_registry_test.go:326` (`{"permission", "cost", "delegation", "compaction"}`) | **Retires** at AG-06: these are the four families AG-06 owns. The list becomes empty (or the AG-06 names are moved to the per-prefix witness assertions) |

### AG-05 carry-forward warnings to AG-06 (verbatim from AG-05 verify-report)

| Source | Finding | AG-06 mitigation |
|---|---|---|
| AG-04 W1 (carry-forward) | R-AEV-006 position-naming untested (PARTIAL scenarios) | AG-06 must write position assertions, not name-assertions, in its new scenarios |
| AG-04 W2 (carry-forward) | Position reports the sequence value, not the slice index | AG-06's bites against the scope-fence and validator must use **slice index** (`ai.AtIndex("event", i)`) and **test the slice index**, not the sequence |
| AG-04 W3 (carry-forward) | `Terminal` field inert (fixed in `c203f25c`, but AG-05 prose claim was wrong) | **Write `Terminal: false` explicitly** in every AG-06 descriptor row (S1 lesson). If AG-06's `compaction_failed` declares `Terminal: true`, the engine actually honors it now — bite test must confirm |
| AG-04 W4 (carry-forward) | `Event.Turn()` hardcoded survived | AG-06.3 delegation events MUST carry parent identity; the reflection pin must verify **field name AND field type** (S2 lesson from AG-05) — `parent RunID` typed as `RunID`, not as a proxy under a different name |
| AG-05 W1 (carry-forward) | `reconstructString` (`message_text_test.go:309-339`) is a vacuous round-trip | AG-06 must NOT write a vacuous helper for any of its four families. If a helper is needed (e.g. for cost accumulation, or delegation tree walk), it MUST be bite-tested RED before the property is GREEN |
| AG-05 W2 (carry-forward) | `TestEventKinds_AG05AllRegisterPlacementTurn` is a name-prefix test | AG-06 must **split into two tests**: (a) name check — "the 4 new kinds have these registered names"; (b) placement check — `descriptor.Placement` reflects the structural intent (e.g. `compaction_started` outside a turn is rejected by `stream_check.go:161`) |
| AG-05 S1 (carry-forward) | `Terminal: false` zero-value vs explicit | Already covered above (AG-04 W3) |
| AG-05 S2 (carry-forward) | Reflection pin checks name only | Already covered above (AG-04 W4) |
| AG-04 W7 (inherited) | `S-AEV-054` is a doc-phrase check, not a rule enumeration | AG-06's `R-APE-NNN` scenarios must assert behavior, not phrases — especially the cost family's "every figure labelled estimate or final" rule |
| AG-04 W8 (inherited) | Coverage 69.7% in AG-04; AG-06 must keep coverage green as it adds kinds | AG-06 has 4 new kinds × (constructor + accessor + validate) = at least 12 new code paths; the every-kind-constructible guard exercises one path per kind = 4 more. All 16 paths must be exercised; the byte-counting `agent.Failure.Retryable()` W8 gap remains an open issue |
| AG-04 W9 (inherited) | Scenario count drift propagates 3 artifacts deep | **State scenario count identically** in proposal, tasks, apply-progress. AG-06's count: **9 charter Gherkin scenarios → ~14–22 spec scenarios after per-scenario expansion + bites**. The proposal must commit to a number; the design and tasks must restate it |
| AG-05 S2 (inherited) | `TestToolCallOrdinal_*` reflection pin (`tool_event_test.go:283-289`) | AG-06 will likely have its own ordinal-like fields (e.g. `compaction_id`, `subagent_id`); pin both name AND type |
| AG-05 W7 (inherited) | `CardinalityAtMostOne` seam reserved at `event_descriptor.go` (referenced by `S-AEV-112`) | **AG-06 opts kinds in.** This is one of AG-06's structural decisions |

### Process discipline carried from AG-04/AG-05

- **Strict TDD, RED-first, real command-verified break-and-restore.** AG-05's deviation-note ("production skeletons preceded the test assertions at scale, with RED shown afterward by break-and-restore; for guards that is legitimate and the only meaningful proof") applies. AG-06 should follow the same disclosure.
- **`make vuln-check` is NOT in `make all`** (Engram `obs #2944`). Run it explicitly at apply and verify.
- **`golangci-lint cache clean` before treating any lint finding as real** (AG-04 `6c821c0a` precedent).
- **Scenario counts will be re-verified by `sdd-verify`**. The proposal/tasks/apply-progress should each restate the count: **9 charter Gherkin scenarios in 5 leaves** → expected ~14–22 spec scenarios in `R-APE-`/`S-APE-` ids after per-rule expansion, witness-table bite, and `S-AEV-090`-retightening bite.
- **Identifier convention** must be resolved at proposal time, not deferred: **`R-APE-`/`S-APE-`** is the natural pick (matches the slug; "protocol" is the noun the slug uses). The proposal should commit to it in the success criteria.
- **`sdd-archive` MUST run in the worktree (the PR branch), NOT in the main checkout** (Engram `obs #2963`). The orchestrator's archive step must `cd` into the worktree, commit the archive changes, push to the PR branch. AG-05 had a sequencing bug here; AG-06 must not repeat it.

---

## 3. The problem — what AG-06 closes (R-04)

AG-06 closes **R-04**: the four families [doc 0001 § 4.3](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md#43-the-agent-event-envelope) marks **absent** from the v1 design — G1, G10, G7, G3's visible halves (`0003:604`).

| Register row | G-finding | What the v1 design says | What AG-06 closes |
|---|---|---|---|
| **G1** | Permission as a suspendable protocol, with allow-once / allow-always / deny / modify-input, remembered per session and derived for subagents | "Seam now, implement in L2" (`doc 0005 § D4`) — the seam exists at AG-04.1 (`PermissionPolicy` port at `doc 0001 § 5.1`), but the event family does not | The **3 kinds of the permission family**: `decision_required` (carries call identity + tool name + arguments); `decision_made` (4 typed outcomes: `allow_once`, `allow_always`, `deny`, `modify_input` — the last carries modified arguments); `resolution_remembered` (a fact about future calls, distinct from `decision_made`) |
| **G10** | Cost and usage as first-class events rather than something each frontend re-derives | "Deferred — the Layer 1 obligation is already met" (`doc 0005 § D4`) — Layer 1 owes nothing more; Layer 2 owes per-turn and per-session cost events; Layer 3 prices them | The **cost family payload shape**: token-only (`input`, `output`, `cache_read`, `cache_write`, `reasoning`); per-turn and cumulative; every figure labelled `estimate` or `final` (because reasoning tokens only arrive on the final usage update — `doc 0001 § 7 G10` "Two dispositions worth their reasoning", line 711) |
| **G7** | Subagents as a harness invoked from a tool, with nested cancellation, cost and permission scope, and parent-identified events | "Seam now" (`doc 0005 § D4`) — the seam exists at AG-01 (parent identifier from AG-04.1); delegation mechanics are AG-19 | The **2 kinds of the delegation family**: `subagent_started`, `subagent_ended`, parent-linked (every event attributable to its place in the tree) |
| **G3** | Context compaction: protect recent turns, never orphan a call/result pair, record what was removed, survive interruption | "Seam now, implement in L2" (`doc 0005 § D4`) — the seam exists at AG-04.1 (`compaction_id` carrier); mechanics are AG-16 | The **3 kinds of the compaction family**: `compaction_started`, `compaction_finished` (carries removed span + summary identity), `compaction_failed` (distinct from finished — recovery has nothing to reason from otherwise) |

**Total: 3 + 2 + 2 + 3 = 10 new event kinds** (the user prompt's charter said "permission family (decision required / decision made / resolution remembered); cost family (per-turn and cumulative, labelled estimate vs final); delegation family (subagent started / ended, parent-linked); compaction family (started / finished / failed / what was removed)" — the cost and delegation families are described functionally but the kind count for cost is **1 kind with a labelled-figure discriminator** rather than 2 or 3 distinct kinds; this is one of the design-time calls the proposal must make — see § 4 below).

### Why this matters — the "if it is not on the stream" criterion

`doc 0003:418` (the AG-04 acceptance, restated as `L2C-04` in `doc.go:23`) makes the membership criterion explicit: **"if it is not on the stream, no frontend can render it and no log can reconstruct it"**. AG-06's four families are the ones AG-04 deferred ("absent — G1, G10, G7, G3") precisely because their mechanisms (permission port, price table, subagent harness, context strategy) ship later; AG-06 makes their **events** constructible so AG-10, AG-16, AG-18, AG-19 can emit rather than invent.

### One v2 conflict, already reconciled by doc 0003

[v2 § 4.3](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md#43-the-agent-event-envelope)'s cost-family row reads *"per-turn and cumulative tokens, cache hits and money"*, while [ADR 0005 § D4](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d4--v1-scope-for-cross-cutting-concerns) and v2 § 7 rule "L2 emits, L3 prices". The verdict wins: **the Layer 2 payload is token-only, and money joins the stream as Layer 3 enrichment** (`0003:613`). AG-06.2's "no field that could carry money" (scenario 2 at `0003:659`) is the spec-level restatement.

---

## 4. Key decisions surfaced (open questions for proposal)

These are decisions AG-06's `sdd-propose` phase must resolve, with cited evidence, before `sdd-spec` can write the requirements.

### D1 — Cost family kind count (1 vs 2+)

The charter names a single function ("cost family (per-turn and cumulative, labelled estimate vs final)"). Two readings:

- **D1a (single kind `cost`)** — one `Cost` event kind carrying a typed `CostFigures` payload with `(scope: turn|session, label: estimate|final, input, output, cache_read, cache_write, reasoning tokens)`. The label and scope are typed values inside the payload, not kinds. **1 new kind.**
- **D1b (two kinds `cost_turn_estimate`, `cost_session_final`, etc.)** — kinds-by-discriminator, mirroring `RunOutcome`/`TurnOutcome`/`ToolOutcome`'s pattern. **2+ new kinds** (one per (scope, label) cell).
- **D1c (one kind per scope)** — `cost_turn` and `cost_session`, label as payload field. **2 new kinds.**

**Evidence**:
- Charter `0003:651-666` uses singular "cost family"; scenario `0003:656` says "a cost event" (singular); scenario `0003:662` says "any figure emitted before the final update is labelled estimate and the final figure is labelled final".
- AG-04's `RunOutcome` (3 typed outcomes) and AG-05's `ToolOutcome` (3 typed outcomes) are **typed values inside one kind**, not kinds-by-outcome.
- The cost family's discriminator (label: estimate/final) is orthogonal to the discriminator (scope: turn/session), so a single payload with two typed-value fields is the natural shape.

**Recommendation (surface, not pre-pick)**: **D1a** is the most likely winner (1 kind with two typed-value discriminators), but this is a proposal decision the user must own.

### D2 — Permission outcome vocabulary (typed enum vs separate kinds)

The charter names "decision-made events for each outcome — allow once, allow always, deny, modify input" (`0003:638-641`). Two readings:

- **D2a (single kind `decision_made` with typed `DecisionOutcome`)** — one kind, 4 typed outcome values; the `modify_input` outcome carries the modified arguments as a payload field. Mirrors `ToolOutcome`'s pattern.
- **D2b (4 kinds `decision_allow_once`, `decision_allow_always`, `decision_deny`, `decision_modify_input`)** — kinds-by-outcome.

**Evidence**:
- AG-04's `RunOutcome` (3 typed values in one kind) and AG-05's `ToolOutcome` (3 typed values in one kind) both chose the typed-enum pattern.
- The charter says "each outcome is a distinct typed value" (`0003:640`) — wording aligned with D2a, not D2b.

**Recommendation (surface, not pre-pick)**: **D2a**, but the proposal must confirm.

### D3 — Compaction terminal semantics

The charter names `compaction_failed` as "distinct from finished — or recovery has nothing to reason from" (`0003:693-696`). Two readings:

- **D3a (`compaction_failed` declares `Terminal: true`)** — the engine's existing "nothing follows the terminal" rule (`stream_check.go:173-175`) applies; nothing may follow `compaction_failed` on the stream. Matches the recovery-isn't-coming semantic.
- **D3b (`compaction_failed` declares `Terminal: false`)** — recovery COULD happen later (a subsequent `compaction_started` after a `compaction_failed`); the engine does not enforce "nothing follows".

**Evidence**:
- `doc 0001 § 7 G3` says "survive interruption" — recovery is the explicit design intent. D3b honors that.
- AG-04's `run_end` declares `Terminal: true` because run-end is genuinely terminal; AG-05's message and tool kinds declare `Terminal: false` because they're not. `compaction_failed` is in between.

**Recommendation (surface, not pre-pick)**: **D3b** is more honest with the design intent ("survive interruption"), but the proposal must confirm — the user owns the call.

### D4 — `resolution_remembered` kind placement

The charter names `resolution_remembered` as "a fact about future calls the session log needs, or it cannot explain why later calls were never asked" (`0003:643-646`). Two readings:

- **D4a (own kind, `PlacementTurn`, `CardinalityAny`)** — a kind in its own right, like any other event, with a payload carrying the remembered fact (tool name + remembered outcome).
- **D4b (own kind, `PlacementTurn`, `CardinalityAtMostOne`)** — same as D4a but a tool name can be remembered at most once per run (the "remembered" semantic suggests one row per tool name).

**Evidence**:
- The "remembered" semantic is per-tool-name, not per-event — a tool name remembered once is remembered forever (within the run). D4b honors that.
- AG-04.3 reserved `CardinalityAtMostOne` at `event_descriptor.go:103-120` for AG-06; this is one of its first uses.

**Recommendation (surface, not pre-pick)**: **D4b**, but the proposal must confirm.

### D5 — `compaction_finished` payload shape (span identity)

The charter says compaction-finished "identifies the replaced span of transcript entries and the summary identity — enough for a session log to persist the operation" (`0003:688-691`). Two readings:

- **D5a (span identity = `[firstSeq uint64, lastSeq uint64]`)** — the span by sequence numbers. A consumer reconstructs the span from the stream by sequence range.
- **D5b (span identity = `[firstTurnID TurnID, lastTurnID TurnID]`)** — the span by turn identity. A consumer reconstructs the span from the stream by turn bracket.
- **D5c (span identity = `[firstEventOrdinal uint32, lastEventOrdinal uint32]`)** — a payload-side ordinal (mirroring AG-05's tool-call ordinal pattern).

**Evidence**:
- AG-04's `Sequence` is per-lane, not per-run; `seq` alone is not enough to identify a span in a multi-lane stream (the AG-04 design `0003:445-448`).
- AG-05's `ordinal uint32` is payload-side for the tool family (R-AMT-007, AD-3). A "compaction ordinal" would be the same pattern for compaction events.

**Recommendation (surface, not pre-pick)**: **D5b** is the cleanest (turns are the unit of "recent turns" the charter says compaction protects, `doc 0001 § 7 G3` "protect recent turns"). But this is a proposal decision the user owns.

### D6 — Spec prefix

Per the AG-05 precedent (`R-AMT-`/`S-AMT-` chosen at proposal), AG-06 needs a new prefix. Three candidates:

- **`R-APE-`/`S-APE-`** — matches the slug `cachicamas-agent-protocol-events`. The "P" stands for "protocol" (the noun in the slug).
- **`R-APR-`/`S-APR-`** — same P, but R for "protocol-events". Less distinct from `R-AEV-`.
- **`R-APP-`/`S-APP-`** — "agent-protocol-Permission/cost/etc.".

**Evidence**: AG-01 = `R-AGE-`, AG-04 = `R-AEV-`, AG-05 = `R-AMT-`. The convention is two-letter prefixes; the third letter often distinguishes the family within "agent". AG-06 owns four families — a single `APE` prefix for all four is consistent with AG-04's single `AEV` for both run and turn.

**Recommendation (surface, not pre-pick)**: **`R-APE-`/`S-APE-`** is the natural pick, but the proposal must commit to it (and the user owns the call).

### D7 — Single PR vs chained PRs (size:exception)

AG-04 and AG-05 both landed as **single PR** with `size:exception` (AG-04: 2.7× the 1000-line budget, 2889 backend Go insertions; AG-05: 2.5× the budget, 2479 insertions). AG-06's four families with per-kind (constructor + accessor + validate + bites) plus AG-06.5's guard extension is forecast at **1500–2400 lines** (AG-05 was 2479; AG-06 has 4 new kinds vs AG-05's 11).

**Evidence**: AG-05.3's reconstruction property references both message and tool kinds, so chained PRs were not safe. **AG-06 has no analogous cross-family property** — the four families are independent in their semantics. Chaining COULD be safe (one PR per leaf: AG-06.1, AG-06.2, AG-06.3, AG-06.4, AG-06.5 guard).

**Recommendation (surface, not pre-pick)**: Either single PR (AG-05 precedent) or 2-PR chained (AG-06.1+AG-06.2+AG-06.3 as one; AG-06.4+AG-06.5 as another — the guard extension naturally follows compaction but logically extends over all four). The user owns the call.

### D8 — Per-family file split vs single combined file

AG-05 used **per-family file split** (`message_text.go` + `message_reasoning.go` + `tool_event.go`) as the AG-04 default. AG-06 has four families — four files (`permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`), or one combined `protocol_events.go`.

**Evidence**: AG-05's deviation #1 (in `apply-progress.md`) recorded `run_events.go` as the file-per-family default; AG-05 used 3 files. AG-06's per-family discipline argues for 4 files. AG-04.3 used 2 files (run + turn).

**Recommendation (surface, not pre-pick)**: 4 files, mirroring AG-05's per-family discipline. The proposal should commit to it (and the user owns the call).

---

## 5. Substrate inventory

### Files AG-06 will modify

| File | Touch | Why |
|---|---|---|
| `backend/agent/src/agent/event.go` | **MODIFIED** | Extend the `EventKind` const block + move `eventKindEnd` past the 4 new kinds (D6 prefix decision); extend `eventRegistry` with one row per new kind (4–10 rows per D1/D2/D4); update the `EventKind` documented list (`event.go:50-60`). **No edit to `Kind()`, `Event`, `CheckEmit`, identity accessors, or the `EventKinds()` post-AG-04-1 decision** |
| `backend/agent/src/agent/event_descriptor.go` | UNCHANGED unless D1/D2 require new descriptor field | The seven-step procedure is the AG-06 entry point (per `event_descriptor.go:13-46`). If a new descriptor field is needed (e.g. a `Indexed bool` for cost span ordinals, or a `Labeled bool` for cost's estimate/final discriminator), that is an AG-06 ADR amendment with its own evidence — not a silent addition. `CardinalityAtMostOne` is already declared at `event_descriptor.go:118-120`; AG-06 opts kinds in |
| `backend/agent/src/agent/event_registry_test.go` | **MODIFIED** | Extend witness table 15 → 19 (or 19 + N for D1/D2); retighten scope-fence `S-AEV-090` from "exactly 15" to "exactly 19"; add a 20th-scratch-kind bite scenario. **Split name-check + placement-check** tests per AG-05 W2 carry-forward |
| `backend/agent/src/agent/doc.go` | **MODIFIED** | Add prose paragraphs for the four families' semantics (mirroring AG-04's ordering-invariants prose at `doc.go:22-43` and AG-05's `L2C-05` reconstruction prose). **No new guarded row proposed yet** — AG-06's families are judged by `L2C-04` (membership criterion) and the per-family semantics belong in prose, not in a guarded row, per AG-04.3's AD-3 forward-fix preference |
| `backend/agent/src/agent/doc_contract_guard_test.go` | UNCHANGED unless a new guarded row is added | The guard iterates the table; if AG-06 adds no row, no change |
| `backend/agent/src/agent/agent_test_helpers_test.go` | UNCHANGED | `requireViolationPosition` and `funcDeclSignature` are reusable; AG-06 may add its own helper signatures |

### Files AG-06 will NOT touch (the AG-04/AG-05 bet extended)

| File | Why |
|---|---|
| `backend/agent/src/agent/event_descriptor.go` | UNCHANGED unless new descriptor field is needed. `EventDescriptor` struct shape is the substrate's extensibility bet |
| `backend/agent/src/agent/stream_check.go` | UNCHANGED. AG-04.4's `S-AEV-092` extensibility experiment + AG-05's `R-AEV-012` proven path. The engine reads `PlacementTurn`, `CardinalityAtMostOne`, `Terminal` from descriptor data — AG-06's kinds plug in via the registry table |
| `backend/agent/src/agent/failure.go` | UNCHANGED. `R-AEV-008`'s `agent.Failure` wrap is reused by `decision_made` (deny outcome) and `compaction_failed` if they carry a typed failure. No new failure category |
| `backend/agent/src/agent/sequence.go` | UNCHANGED. Envelope identity-shape preserved; any ordinal (compaction span, subagent ordinal) is payload-side |
| `backend/agent/src/agent/{import_boundary,ambient_authority}_test.go` | UNCHANGED. AG-03 guards pass with zero logic edits |
| `backend/agent/{go.mod,go.sum,Makefile,.golangci.yml}` | UNCHANGED. No new dep, no tooling change |
| `backend/agent/src/ai/**` | UNCHANGED. AG-06 reads `ai.MessageID` if it correlates compaction spans to messages; otherwise no Layer 1 read |
| `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` | UNCHANGED in AG-06. Status header changes once AG-06 is merged, mirroring `dbd7a33e`/`c203f25c`/`6b4a3468` pattern |

### New files AG-06 will create (4 files per D8)

| New file | Mirrors | Owns |
|---|---|---|
| `backend/agent/src/agent/permission_events.go` | `tool_event.go` (typed outcomes + Failure wrap) | `PermissionDecisionRequired` / `PermissionDecisionMade` / `PermissionResolutionRemembered` payload types; `PermissionOutcome` (closed: `AllowOnce` / `AllowAlways` / `Deny` / `ModifyInput`, zero not a member — mirroring `ToolOutcome`); constructors `NewPermissionDecisionRequired(run, turn, callID, name, arguments)`, `NewPermissionDecisionMade(run, turn, callID, outcome, ...modifiedArgs)`, `NewPermissionResolutionRemembered(run, name, outcome)` |
| `backend/agent/src/agent/cost_events.go` | `tool_event.go` (per-family file) | `Cost` payload type with typed `CostScope` (`Turn` / `Session`) and `CostLabel` (`Estimate` / `Final`) discriminators; fields `inputTokens`, `outputTokens`, `cacheReadTokens`, `cacheWriteTokens`, `reasoningTokens` (all `uint64`); constructor `NewCost(run, turn, scope, label, figures)` — or per-D1, `NewCostTurn(run, turn, label, figures)` and `NewCostSession(run, label, figures)` |
| `backend/agent/src/agent/delegation_events.go` | `tool_event.go` (per-family file) | `SubagentStarted` / `SubagentEnded` payload types; both carry `parent RunID` (set by envelope, like `NewDelegatedRunStart`); constructors `NewSubagentStarted(run, parent, subagentID, ...)`, `NewSubagentEnded(run, parent, subagentID, ...)` |
| `backend/agent/src/agent/compaction_events.go` | `tool_event.go` + `failure.go` (typed outcomes + Failure wrap) | `CompactionStarted` / `CompactionFinished` / `CompactionFailed` payload types; `CompactionFinished` carries span identity (per D5) + `summaryID`; `CompactionFailed` carries typed `*Failure` per `R-AEV-008`; constructors `NewCompactionStarted(run, compactionID, ...)`, `NewCompactionFinished(run, compactionID, span, summaryID)`, `NewCompactionFailed(run, compactionID, failure)` |
| `backend/agent/src/agent/permission_events_test.go` | `tool_event_test.go` | Per-family scenarios + per-rule expansion + bite(s) |
| `backend/agent/src/agent/cost_events_test.go` | `tool_event_test.go` | Same |
| `backend/agent/src/agent/delegation_events_test.go` | `tool_event_test.go` | Same — and the "tree walkable" property test (mirror AG-05.3's reconstruction property test shape) |
| `backend/agent/src/agent/compaction_events_test.go` | `tool_event_test.go` + `failure.go` | Same — including the recovery-after-failure bite |
| `openspec/changes/cachicamas-agent-protocol-events/specs/agent-protocol-events/spec.md` | AG-05's spec shape | `R-APE-`/`S-APE-` requirements and scenarios |
| `openspec/changes/cachicamas-agent-protocol-events/specs/agent-event-envelope/spec.md` | AG-05's delta shape | `R-AEV-010` MODIFIED (scope-fence 15 → 19); `R-AEV-012` MODIFIED (extensibility experiment path AG-06 took); `R-AEV-013` ADDED if AG-06 introduces a new descriptor field; scope-fence bite scenario added |
| `openspec/changes/cachicamas-agent-protocol-events/{proposal,design,tasks,apply-progress,verify-report}.md` | AG-05's per-phase artifacts | — |

### Substrate line counts (snapshot at worktree HEAD `6b4a3468`)

```
agent_test_helpers_test.go        170
ambient_authority_test.go         380
doc.go                             52
doc_contract_guard_test.go        131
envelope_test.go                  485
event.go                          418
event_descriptor.go               145
event_registry_test.go            444
failure.go                         80
import_boundary_test.go           400
invariant_pin_test.go             298
message_reasoning.go              223
message_text.go                   243
message_text_test.go              347
reconstruction_test.go            424
run_events.go                     206
sequence.go                        58
stream_check.go                   193
stream_check_test.go              561
tool_event.go                     501
tool_event_test.go                315
turn_events.go                    170
─────────────────────────────────
TOTAL                            6244 lines
```

The AG-06 substrate additions (4 new files × ~100–200 lines per kind × 3 constructors + 4 accessors) are forecast at **1500–2400 lines** per D1/D2/D5/D8. This is at the upper bound of AG-05's 2479 actual.

---

## 6. Carry-forward list — AG-04/AG-05 warnings applied to AG-06

Each finding maps to a **concrete mitigation** the proposal must commit to.

| Source | Finding | Concrete AG-06 mitigation |
|---|---|---|
| AG-04 W1 (carry-forward) | R-AEV-006 position-naming untested | AG-06's bite scenarios (`S-APE-NNN` bites) must assert **slice index** in the violation report, not sequence value |
| AG-04 W2 (carry-forward) | Position reports sequence value, not slice index | AG-06's bites assert `ai.AtIndex("event", i)`; the test verifies the slice index, not the sequence value (the W2 fix is in `c203f25c`; AG-06 inherits the corrected behavior) |
| AG-04 W3 (carry-forward) | `Terminal` field inert (now read by engine post-`c203f25c`) | **Write `Terminal: false` explicitly** in every AG-06 descriptor row, even if `false` is the zero value (S1 lesson). If AG-06's `compaction_failed` declares `Terminal: true` (per D3a), the bite must confirm the engine honors it |
| AG-04 W4 (carry-forward) | `Event.Turn()` hardcoded survived | AG-06.3 delegation events' parent identity MUST be tested both directions (parent present, parent absent). Reflection pin must verify **field name AND field type** (S2 lesson) |
| AG-05 W1 (carry-forward) | Vacuous reconstruction helper | AG-06's delegation tree-walk test (if any) MUST be bite-tested RED before the property is GREEN. No vacuous helpers |
| AG-05 W2 (carry-forward) | Name-prefix placement test | Split into (a) name check + (b) placement check tests. AG-06's `S-AEV-090` extends to "exactly 19"; structural pin on `descriptor.Placement`, not on a name prefix |
| AG-05 S1 (carry-forward) | `Terminal: false` zero-value vs explicit | Already covered above (AG-04 W3) |
| AG-05 S2 (carry-forward) | Reflection pin name-only | Already covered above (AG-04 W4) |
| AG-04 W7 (inherited) | `S-AEV-054` is a doc-phrase check | AG-06's "every figure labelled estimate or final" rule MUST have a behavioral test, not a phrase check |
| AG-04 W8 (inherited) | Coverage 69.7% in AG-04 (below 80%) | AG-06's 4 new kinds × (constructor + accessor + validate) = ~16 new code paths; the every-kind-constructible guard exercises one path per kind = 4 more. All 20 paths must be exercised. The byte-counting `agent.Failure.Retryable()` gap remains an open issue for AG-06 to close (if AG-06.1's `decision_made` (deny) or AG-06.4's `compaction_failed` use the failure wrap, exercise `Retryable()`) |
| AG-04 W9 (inherited) | Scenario count drift | **State scenario count identically** in proposal, tasks, apply-progress. AG-06's count: **9 charter Gherkin scenarios in 5 leaves** → ~14–22 spec scenarios after per-scenario expansion + bites. The proposal must commit to a number; the design and tasks must restate it |
| AG-05 W7 (inherited) | `CardinalityAtMostOne` seam reserved | AG-06 opts `resolution_remembered` (per D4b) and possibly `subagent_ended` in via the descriptor row |
| AG-05 S2 (inherited) | Reflection pin field type | Already covered above (AG-04 W4) |
| AG-04 W4 (inherited) | S-AEV-004 PARTIAL (turn identity unproven) | AG-06's parent-identity tests must verify both directions: parent present returns the parent's run identity, parent absent returns distinguishable `(RunID(""), false)` |
| AG-05 W0 (inherited) | Orchestrator brief artifact path | Cosmetic — already named. AG-06's explore artifact goes to worktree path; no action |

---

## 7. Risks & unknowns

### Risks (quantified)

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| **1** | **Spec prefix collision** — `R-APE-` may collide with a future AG-23 (Layer 3 readiness kit) prefix. | Low | The proposal commits to `R-APE-`/`S-APE-` and notes the future collision risk in `agent-package-scaffold` amendment if needed. AG-23 will use a different prefix (`R-L3R-` or similar) since it's Layer 3 |
| **2** | **Cost kind shape (D1) under-specified** — the charter says "per-turn and cumulative, labelled estimate vs final" but the kind-count is open. D1a (1 kind), D1b (multiple kinds), D1c (1 kind per scope) all defensible. | Medium | The proposal MUST resolve D1 with evidence (cited charter text). The chosen shape drives the spec scenario count and the registry size |
| **3** | **`compaction_failed` terminal semantics (D3) under-specified** — if `Terminal: true`, the engine's existing "nothing follows the terminal" rule applies; if `false`, recovery can re-emit a `compaction_started` later. The charter's "survive interruption" wording suggests D3b but is silent on whether a stream may carry events after a failed compaction. | Medium | The proposal MUST resolve D3 with evidence (`doc 0001 § 7 G3`). Bite-test both shapes if necessary |
| **4** | **Resolution-remembered cardinality (D4) under-specified** — `CardinalityAtMostOne` makes structural sense ("a tool name remembered once is remembered forever") but the spec scenario at `0003:643-646` doesn't enforce it. | Medium | The proposal MUST resolve D4 (likely D4b per the `CardinalityAtMostOne` seam reservation) |
| **5** | **Span identity for compaction-finished (D5) under-specified** — span by sequence, by turn bracket, or by payload-side ordinal? AG-04's `Sequence` is per-lane (not per-run), so sequence-only doesn't work for multi-lane streams. | Medium | The proposal MUST resolve D5 with evidence. D5b (turn bracket) is the natural pick but the proposal must confirm |
| **6** | **Single PR vs chained PR (D7) under-specified** — AG-04/AG-05 precedent is single PR with `size:exception`; AG-06 has no AG-05.3-style cross-family property, so chaining COULD be safe (one PR per leaf). | Medium | The proposal MUST resolve D7. The user's preference between single PR (continuity, AG-05 precedent) and chained (review focus, `chained-pr` skill recommendation for >400-line changes) is theirs to make |
| **7** | **Permission outcome shape (D2) under-specified** — single kind with typed enum (AG-04/AG-05 precedent) vs 4 kinds by outcome. | Low | The charter's "each outcome is a distinct typed value" wording aligns with D2a; the proposal should commit to D2a with cited evidence |
| **8** | **Review budget 1000 lines exceeded 2–3x** — AG-06 forecast 1500–2400 lines, matching AG-05 (2479 actual) at the upper bound. Single PR + `size:exception` is the AG-04/AG-05 precedent. | High | Pre-authorize `size:exception` at the proposal, citing braejan's standing instruction. Chained PRs not strictly necessary (no cross-family property test) but possible if user prefers |
| **9** | **`sdd-archive` worktree-vs-main sequencing** (Engram `obs #2963`) — AG-05 had a sequencing bug; AG-06 must run archive in worktree, commit to branch, push to PR. | Low (known, named, mitigation documented) | Orchestrator's archive step mirrors the implementation step's location. After archive commit + push, do NOT modify main checkout — let merge propagate |
| **10** | **`golangci-lint cache` stale finding** (AG-04 `6c821c0a` precedent) — AG-04 had a phantom lint finding that was a stale cache artifact. AG-06's apply/verify must `golangci-lint cache clean` before each lint gate. | Low | Apply and verify agents explicitly told to run cache clean before lint |
| **11** | **Scenario count drift** (AG-04 W9) — proposal/tasks/apply-progress must restate the count identically. | Low (known, named) | State in all three artifacts: **9 charter Gherkin scenarios in 5 leaves → ~14–22 spec scenarios** |
| **12** | **`make vuln-check` not in `make all`** (Engram `obs #2944`) — apply and verify must run vuln-check explicitly. | Low | Apply and verify agents explicitly told to run `make vuln-check` |
| **13** | **`sdd-attempt settle` flag set incomplete** (Engram `obs #2961`) — settle requires `--diagnosis`, `--cleanup-evidence`, `--process-evidence` in addition to `--outcome`, `--harness-disposition`, `--evidence-revision`. | Low (known, named) | Apply and verify agents explicitly told to pass all six flags |
| **14** | **`test-driven-development` skill gap** (Engram `obs #2962`) — the skill referenced in `openspec/AGENTS.md` does not exist; RED-GREEN-REFACTOR discipline forwarded inline from `openspec/AGENTS.md`. | Low (known, named) | Apply and verify agents explicitly told the skill doesn't exist and to load `sdd-verify/strict-tdd-verify.md` for verification |

### Unknowns to verify before the design phase

| # | Unknown | What to verify |
|---|---|---|
| **U1** | Whether the cost family's "estimate or final" label is a payload field, a typed value, or a kind-level discriminator | Read `doc 0001 § 7 G10` "Two dispositions worth their reasoning" (lines 709–717) verbatim; the cost family's per-turn/per-session + estimate/final axes are orthogonal |
| **U2** | Whether `permission.events.decision_required` carries arguments as `[]byte` (AG-05's tool precedent, ADR 0005 § D1 row 2) or as a parsed structure | AG-05's `ToolStart.arguments []byte` (tool_event.go:65) is the precedent; permission arguments are the same byte-fidelity case |
| **U3** | Whether `delegation.events.subagent_ended` is `CardinalityAtMostOne` (per-delegated-run, one ended per started) or `CardinalityAny` | The charter says "subagent started / ended" without specifying cardinality; D4b's logic (one-ended-per-started) argues for `CardinalityAtMostOne`, but the engine's `CardinalityAtMostOne` rule is per-kind, not per-spawn (i.e. one subagent_ended event per kind per stream, not per delegated run) — the semantics may not match |
| **U4** | Whether `compaction.events.compaction_failed` should also declare `CardinalityAtMostOne` (one failure per compaction) | If yes, the proposal must commit; the engine's rule reads the kind-level cardinality, not per-instance |
| **U5** | Whether the parent identifier for delegation events is set by `NewDelegatedRunStart`-equivalent (`run_events.go:60-76`) or by a new `NewDelegatedSubagent*` family of constructors | The existing `NewDelegatedRunStart` sets parent on `RunStart`; AG-06.3 may need a sibling that sets parent on `SubagentStarted`/`SubagentEnded` specifically. The parent field already exists on `Event` (`event.go:362-366`) |
| **U6** | Whether the spec scenarios at `0003:631-665` for permission and cost are complete or whether AG-06's spec must add behavioral scenarios beyond the Gherkin text | The AG-05 precedent (7 charter → 15 spec) suggests 9 charter → ~14–22 spec is a reasonable expansion. The proposal must commit to the count |

---

## 8. Recommendation for next phase

**Recommendation**: **Proceed to `sdd-propose`** for AG-06 with the following user-decisions to be presented (not pre-picked):

### User decisions to put in front of the user (proposal phase)

1. **D1 — Cost kind count**: D1a (1 kind `cost` with typed `CostScope` + `CostLabel` discriminators) is the recommended choice; the proposal should commit to it with cited charter evidence. Alternative: D1b/c if the user prefers kinds-by-discriminator.
2. **D2 — Permission outcome vocabulary**: D2a (single kind `decision_made` with typed `PermissionOutcome`) is the recommended choice; the proposal should commit to it with cited charter evidence.
3. **D3 — `compaction_failed` terminal semantics**: D3b (`Terminal: false`, recovery may follow) is the recommended choice, honoring `doc 0001 § 7 G3` "survive interruption"; the proposal should commit to it with cited evidence. The bite must confirm the engine honors the explicit declaration.
4. **D4 — `resolution_remembered` kind placement**: D4b (`CardinalityAtMostOne`) is the recommended choice, exercising the AG-04.3-reserved seam; the proposal should commit to it with cited evidence.
5. **D5 — Span identity for `compaction_finished`**: D5b (turn bracket `[firstTurnID, lastTurnID]`) is the recommended choice, matching `doc 0001 § 7 G3` "protect recent turns"; the proposal should commit to it with cited evidence.
6. **D6 — Spec prefix**: `R-APE-`/`S-APE-` is the recommended choice, matching the slug `cachicamas-agent-protocol-events` and the AG-04/AG-05 prefix convention; the proposal should commit to it.
7. **D7 — Single PR vs chained PRs**: Single PR with `size:exception` (AG-04/AG-05 precedent) is the recommended choice; chained PRs are also safe (no AG-05.3-style cross-family property test). The user owns the call.
8. **D8 — Per-family file split**: 4 files (`permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`), mirroring AG-05's per-family discipline; the proposal should commit to it.

### Process discipline carried forward (no user decision)

- Strict TDD with RED-first bite-and-restore (AG-05 deviation note precedent).
- `make vuln-check` explicit (Engram `obs #2944`).
- `golangci-lint cache clean` before each lint gate (AG-04 `6c821c0a`).
- Scenario count stated identically across proposal, tasks, apply-progress (AG-04 W9 lesson).
- `sdd-archive` runs in worktree, commits to PR branch (Engram `obs #2963`).
- All six `sdd-attempt settle` flags passed (Engram `obs #2961`).
- `test-driven-development` skill doesn't exist; discipline forwarded inline (Engram `obs #2962`).

### What the orchestrator should communicate before the proposal

1. **AG-06 will exceed the 1000-line review budget** (forecast 1500–2400 lines, matching AG-05's 2479 actual). Pre-authorize `size:exception` the same way AG-04/AG-05 were.
2. **AG-06 closes Layer 2 Wave 1** — the last of the Wave-1 milestones. Doc 0003 stands at 6 of 24 milestones after AG-06 merges.
3. **AG-06 is the first kind-set to exercise `CardinalityAtMostOne`** (the AG-04.3-reserved seam at `event_descriptor.go:118-120`); the engine reads it (`stream_check.go:166-171`), so no engine edit is needed.
4. **AG-06.3 delegation events are the first to set the parent identifier outside `NewDelegatedRunStart`** — the parent field exists (`event.go:362-366`); AG-06.3's constructors must set it.
5. **No new dependencies, no `go.mod`/`go.sum` edit, no AG-03 guard edit, no `event_descriptor.go`/`stream_check.go`/`failure.go`/`sequence.go` edit.** AG-06's value is in demonstrating the substrate's extensibility for the second time (after AG-05).
6. **AG-06.5's guard extension is the canonical example of "the every-kind-constructible guard bites on a 20th scratch kind"** — `TestEventKinds_ScopeFence_BitesByCountOnSixteenthKind` (`event_registry_test.go:411-425`) becomes `BitesByCountOnTwentiethKind` and retightens from 15 to 19.
7. **Scenario count to commit to in proposal**: **9 charter Gherkin scenarios in 5 leaves → ~14–22 spec scenarios** after per-scenario expansion + bites. State identically in proposal, tasks, apply-progress.
8. **All five `sdd-attempt settle` flags** must be passed at apply time: `--outcome`, `--harness-disposition`, `--evidence-revision`, `--diagnosis`, `--cleanup-evidence`, `--process-evidence` (Engram `obs #2961`).

### Ready for Proposal

**Yes — the orchestrator can launch `sdd-propose` next.** The substrate is well-understood, the AG-04/AG-05 lessons are explicit, the open decisions are listed (not silently picked), the risks are quantified, and the unknowns are named with verification steps.

The proposal's first sentence should commit to the answer (per `cognitive-doc-design`):

> *"AG-06 ships the four missing Layer 2 event families — permission (3 kinds), cost (1 kind with typed discriminators), delegation (2 kinds, parent-linked), compaction (3 kinds, recovery-distinct) — on the AG-04/AG-05 substrate, with the every-kind-constructible guard biting on a 20th scratch kind before AG-06 lands, in a single PR with `size:exception`."*

Then expand.

---

## Key Learnings (for Engram passive capture)

1. **AG-06 is the first kind-set to exercise `CardinalityAtMostOne`** (the AG-04.3-reserved seam at `event_descriptor.go:118-120`). The engine reads it at `stream_check.go:166-171`, so AG-06's `resolution_remembered` (and possibly `subagent_ended`) opt in via the descriptor row only.
2. **AG-06.3 delegation events are the first consumers of the parent identifier field outside `NewDelegatedRunStart`** (`run_events.go:60-76`). The field exists from AG-04.1's R-AEV-003; AG-06.3's constructors must set it on `SubagentStarted`/`SubagentEnded`.
3. **The cost family's "estimate or final" discriminator is orthogonal to "per-turn / per-session"** — both are typed values inside one kind (D1a), not kinds-by-discriminator. This mirrors AG-04's `RunOutcome` and AG-05's `ToolOutcome` patterns.
4. **`compaction_failed` declares `Terminal: false`** to honor `doc 0001 § 7 G3` "survive interruption" — recovery may emit a subsequent `compaction_started`. The engine's "nothing follows the terminal" rule (`stream_check.go:173-175`) does not bind compaction-failed events.
5. **AG-06 closes the doc 0001 § 4.3 family-table gap**: permission (G1), cost (G10), delegation (G7), compaction (G3 visible halves). Each family is judged by the `L2C-04` membership criterion ("if it is not on the stream, no frontend can render it and no log can reconstruct it") — AG-06 makes the events constructible so AG-10, AG-16, AG-18, AG-19 emit rather than invent.
6. **AG-06 closes Layer 2 Wave 1**. Doc 0003 stands at 6 of 24 milestones after AG-06 merges. The next milestone is AG-07 (Loop-Level Emission), which inherits the 19-kind registry unchanged.
7. **The scope-fence `S-AEV-090` retightens from "exactly 15" to "exactly 19"** in the same commit as the new kinds land. `TestEventKinds_ScopeFence_BitesByCountOnSixteenthKind` becomes `BitesByCountOnTwentiethKind`.
8. **The forbidden-names list (`event_registry_test.go:326`) retires** — permission, cost, delegation, compaction are no longer "forbidden", they are AG-06's own. The list becomes empty or is repurposed for AG-07+ forward-fence.
9. **AG-06 has no AG-05.3-style cross-family property test** — the four families are independent in their semantics. Chaining COULD be safe (one PR per leaf); the AG-05 precedent is single PR with `size:exception`. The user owns the call.
10. **Scenario count discipline** (AG-04 W9 carry-forward): state `9 charter → ~14–22 spec` identically in proposal, tasks, apply-progress.

---

## Evidence discipline

Every claim in this artifact cites a file:line or Engram observation:

- Charter + G-finding rows: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:602-712`
- v2 architecture: `docs/architecture/0001-cachicamas-agent-stack-v2.md:523-540` (§ 4.3 envelope), `:695-705` (§ 7 register rows G1, G3, G7, G10)
- ADR 0005 dependency rule: `docs/adr/0005-promote-agent-stack-to-own-module.md:156-179` (§ D1), `:230-261` (§ D3), `:263-312` (§ D4)
- Substrate: `backend/agent/src/agent/event.go:91-156` (EventKind), `:328-336` (Event), `:362-366` (Parent); `event_descriptor.go:46-48, 88-100, 103-120, 142-144`; `stream_check.go:166-175`; `run_events.go:60-76` (NewDelegatedRunStart); `sequence.go:30-58`; `failure.go:24-79`
- Spec pattern: `openspec/specs/agent-event-envelope/spec.md:8, 113, 160, 185`; `openspec/specs/agent-message-tool-events/spec.md:6`
- AG-04 verify warnings: `openspec/changes/archive/2026-08-12-cachicamas-agent-event-envelope/verify-report.md:219-229` (W1-W9)
- AG-05 archive carry-forwards: `openspec/changes/archive/2026-08-12-cachicamas-agent-message-tool-events/archive-report.md:59-75` (W0-W2, S1-S2)
- Engram observations: #2940 (AG-04 merge), #2957 (AG-05 verify), #2958 (AG-05 archive), #2961 (settle flags), #2962 (TDD skill gap), #2963 (archive-in-worktree), #2964 (AG-05 merged), #2965 (AG-05 session summary)
