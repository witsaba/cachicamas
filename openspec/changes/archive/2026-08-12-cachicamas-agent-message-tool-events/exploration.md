# Exploration — `cachicamas-agent-message-tool-events` (AG-05)

> Milestone AG-05 (Layer 2 Wave 1) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-05--add-message-and-tool-execution-event-families), lines 518–600. SDD change slug: `cachicamas-agent-message-tool-events`. Artifact store: hybrid (Engram + OpenSpec). Engram topic key: `sdd/cachicamas-agent-message-tool-events/explore`.

## What AG-05 owes the substrate it inherits

AG-05 is the second Wave-1 milestone on the `cachicamas-agent-event-envelope` substrate. It adds the two high-volume event families (message and tool) that doc 0001 § 4.3 names "in the v1 design" alongside the two lifecycle families AG-04 already shipped. The substrate is:

| Substrate piece | Lives at | AG-05 use |
| --- | --- | --- |
| Envelope + derived kind + per-lane `LaneStamper` | `backend/agent/src/agent/event.go`, `sequence.go` | Reuse unchanged |
| Six-step "adding a kind" procedure | `event_descriptor.go:13-31` (file comment) | Follow exactly; AG-05 documents the same procedure for the Wave 1 reader |
| `PlacementTurn` seam (designed for AG-05) | `event_descriptor.go:78-85` | **First use.** Message and tool events MUST register with `Placement: PlacementTurn` (a `message_start` outside an open turn is rejected by the validator — by design, per `stream_check.go:161`) |
| `Cardinality` (default `Any` is zero value) | `event_descriptor.go:95-104` | All AG-05 kinds use the zero `CardinalityAny` (none are "at most one") |
| `Terminal` (post-W3 fix in `c203f25c`) | `event_descriptor.go:126-128`, `stream_check.go:108` | AG-05.2 `ToolEnd` MAY be `Terminal: true` if it closes the stream (out of charter per `0003:613-614`'s pattern); otherwise leave false. **AG-05 inherits W3's fix already**, no new work |
| `BracketRoleNone` for non-bracket kinds | `event_descriptor.go:46-48` | Both message and tool events use this — they neither open nor close a bracket, they live inside a turn |
| Failure wrap | `failure.go` (AD-2) | AG-05.2's `ToolEnd` "execution itself failed" reuses `agent.Failure` exactly as `RunEnd`/`TurnEnd` do; no parallel vocabulary |
| Stream-contract validator | `stream_check.go` | No edit. AG-05.3 reconstruction test exercises the validator as-is |
| Doc-matrix guard `L2C-04` row | `doc.go:20`, `doc_contract_guard_test.go:58` | Unchanged. AG-05's families are judged by `L2C-04` (stream membership) — the criterion `0003:418` makes explicit |
| Every-kind-constructible guard | `event_registry_test.go` | **Extends.** Witness table grows from 4 to 6 (or 9, see Design fork A1); scope-fence `S-AEV-090` tightens from "exactly 4" to "exactly N" with N = AG-05's count |
| Identifier convention | `specs/agent-event-envelope/spec.md:8` | `R-AEV-`/`S-AEV-` taken by AG-04; `R-AGE-`/`S-AGE-` taken by `agent-event-delivery` (`openspec/specs/agent-event-delivery/spec.md`). **AG-05 needs a new prefix** — `R-AMT-`/`S-AMT-` is the natural pick (matches slug `cachicamas-agent-message-tool-events` and avoids stutter) |

## Current State

Layer 2's `agent` package, after AG-04 (PR #163, merge `967d043f`), has:

- **4 registered event kinds** (`event.go:64-83`, `eventRegistry` at `:99-115`): `EventKindRunStart`, `EventKindRunEnd`, `EventKindTurnStart`, `EventKindTurnEnd`. Documented list at `event.go:50-60`; the four are mirrored in the every-kind-constructible guard's `expectedKinds` set in `event_registry_test.go` (the scope fence, S-AEV-090).
- **Closed `EventDescriptor` vocabulary** with two unused seams the design expressly reserved for AG-05/AG-06: `PlacementTurn` (AG-05's; per `event_descriptor.go:72-85`'s own comment "AG-05's seam; no kind AG-04 registers uses it") and `CardinalityAtMostOne` (AG-06's). AG-05 is the first kind-set to exercise `PlacementTurn`.
- **11 charter Gherkin scenarios** for AG-04, mapped to 11 requirements (`R-AEV-001`…`R-AEV-011`) and 45 unique `S-AEV-0NN` scenario ids in `openspec/specs/agent-event-envelope/spec.md` (plus 6 restated `S-AGP-010`…`S-AGP-015` from the `agent-package-scaffold` delta — 51 spec scenarios total across the change; W9 closed).
- **Validator with two-level scope engine** (`stream_check.go`), production-exported, slice-index position naming (W1/W2 fix in `c203f25c`), and `Terminal` field actually read (W3 fix in same commit). All 8 of AG-04's `BLOCKED ON AG-04` warnings from verify are now closed in the merged main.
- **One guarded `L2C-04` row** in `doc.go:20`, machine-checked by `doc_contract_guard_test.go:58`. The criterion is *"if it is not on the stream, no frontend can render it and no log can reconstruct it"* (`0003:418`) — the AG-05.3 reconstruction property is the literal expression of "a log can reconstruct it."
- **Six-step "adding a kind" procedure** documented at `event_descriptor.go:13-31` and proven by `S-AEV-092`'s extensibility experiment: registering a 5th kind following the procedure exactly required zero edits to `stream_check.go` or `event_registry_test.go`'s own logic. AG-05 extends the same procedure — same `Scope fence` and `W3 latent trap` apply (an AG-05/AG-06 kind declaring `Terminal: true` with `BracketRoleNone` is **silently non-terminal** until verify closes W3 across the engine, which `c203f25c` already did at AG-04's W3 fix).

## Charter (AG-05, doc 0003:518-600)

| Node | Type | Charter Gherkin scenarios | Closes |
| --- | --- | --- | --- |
| **AG-05.1** — Message family `[leaf]` | start, deltas, end with reasoning distinguished from text | 3 (S charter: `0003:543-558`) | R-04 (message half), envelope invariant 1 (jointly with AG-04.3, per `0003:2203`) |
| **AG-05.2** — Tool execution family `[leaf]` | start, progress, end with 3 distinct typed outcomes; call ordinal correlates events to call order regardless of completion order | 3 (S charter: `0003:567-583`) | R-04 (tool half), R-13 (ordinal) |
| **AG-05.3** — Reconstruction property `[leaf]` | interleaved streams reconstruct independently and completely — *"the property a session log will depend on, proven before any producer exists"* (load-bearing sentence) | 1 (S charter: `0003:594-597`) | R-04 acceptance, the reconstruction-half of the doc 0001 § 4.3 membership test |

**Total normative behavior**: 7 Gherkin scenarios in 3 leaves. AG-04 had 11 in 4 leaves (and ended at 45 `S-AEV` scenario ids via per-rule scenarios + bites + reconstruction). AG-05's per-rule count will be similar: 1 bite for AG-05.1's no-delta-to-accumulated-route pin, 1 bite for the every-kind-constructible guard's scope-fence, plus the reconstruction scenarios. **Forecast: 8–12 spec scenarios** (one prefix across both families, mirroring AG-04's one-prefix-per-capability discipline).

**Out of scope** (charter `0003:527-528`): producing these events from a live loop (AG-07, AG-09); permission events around tools (AG-06, AG-10). The acceptance criterion — *"a consumer accumulating deltas reconstructs the message exactly"* and *"a tool event stream distinguishes the three end states (success, result-reports-failure, execution failed) by type, not by convention"* — is the spec-side proof of the property, **before** the loop emits anything.

## Affected Areas

### Already-shipped files that AG-05 touches

| File | Touch | Why |
| --- | --- | --- |
| `backend/agent/src/agent/event.go` | **MODIFIED** | Extend the `EventKind` const block + move `eventKindEnd` past the new kinds; extend `eventRegistry` with one row per new kind (6 rows minimum: 4 message + 3 tool, possibly more per Design fork A1); update the `EventKind` documented list (`event.go:50-60`). **No edit to `Kind()`, `Event`, `CheckEmit`, identity accessors, or the `EventKinds()` post-AG-04-1 decision ("declared constant space, not the registry")** |
| `backend/agent/src/agent/event_descriptor.go` | UNCHANGED unless Design fork A1 reopens the descriptor shape | The six-step procedure is the AG-05 entry point. If a new descriptor field is needed (e.g. `Indexed bool` for delta kinds), that is an AG-05 amendment to AD-1 with its own evidence — not a silent addition. The two existing AG-05/AG-06 seams (`PlacementTurn`, `CardinalityAtMostOne`) are sufficient for everything in `0003:543-583` as currently drafted |
| `backend/agent/src/agent/stream_check.go` | **UNTOUCHED** | AG-04.4's S-AEV-092 extensibility experiment proved the engine reads only descriptor fields + envelope identity. AG-05.1/AG-05.2/AG-05.3 all execute on the unchanged engine; AG-05.3's reconstruction test asserts the property the validator already enforces |
| `backend/agent/src/agent/doc.go` | **MODIFIED** | Add prose paragraphs for the message/tool family semantics (mirroring AG-04's ordering-invariants prose at `doc.go:22-43`); **no new guarded row** — `L2C-04` already carries the criterion these families are judged by; adding `L2C-05`/`L2C-06` rows would either overclaim or need rewording when AG-06 lands (same forward-fix preference as AD-3) |
| `backend/agent/src/agent/doc_contract_guard_test.go` | **UNTOUCHED** | Unless a new guarded row is added (it isn't) |
| `backend/agent/src/agent/event_registry_test.go` | **MODIFIED** | Extend witness table; retighten scope-fence `S-AEV-090` from "exactly 4" to "exactly N" (N = 6 or 9 per Design fork A1); add a witness entry for each new kind following the same two-leg shape (constructor + payload accessor) |
| `backend/agent/src/agent/agent_test_helpers_test.go` | UNCHANGED | `requireViolationPosition` and `funcDeclSignature` are reusable; AG-05.3's reconstruction helper is a new test-only function in AG-05's own test file |
| `backend/agent/src/agent/envelope_test.go`, `stream_check_test.go`, `invariant_pin_test.go` | UNCHANGED | AG-04's scenario files. AG-05 ships its own scenario files (mirror AG-04's pattern) |

### New files AG-05 creates

Following AG-04's file-per-family discipline (each family has one `<family>_events.go` with the kind constants, payloads, constructors, accessors):

| New file | Mirrors | Owns |
| --- | --- | --- |
| `backend/agent/src/agent/message.go` | `run_events.go`'s structure (markers, typed outcomes) | `MessageStart`/`MessageEnd` payload types; `MessageKind` enum (text vs reasoning) if Design fork A1 resolves to per-message reasoning; constructors `NewMessageStart`/`NewMessageDelta`/`NewMessageEnd` and `NewReasoningDelta` (or symmetric reasoning-start/end if A1 resolves to 6 kinds) |
| `backend/agent/src/agent/message_events.go` (or merged) | `run_events.go:25-76` (one file with both `Start` and `End` halves) | Per AG-04's reconciliation (deviation #1 in `apply-progress.md:21-25`): the file-per-family decision is a design call; the AG-04 default is one file per family |
| `backend/agent/src/agent/tool_event.go` | `run_events.go` + `failure.go` (typed outcomes + Failure wrap) | `ToolStart`/`ToolProgress`/`ToolEnd` payload types; `ToolOutcome` (uint8, closed: `Succeeded` / `ResultReportsFailure` / `ExecutionFailed`, zero not a member — mirroring `RunOutcome`); constructor `NewToolStart(run, turn, callID, name, args)`; `NewToolProgress(run, turn, callID, index, fragment)`; `NewToolEnd(run, turn, callID, outcome, result *Failure, payload []byte)` |
| `backend/agent/src/agent/tool_events.go` (or merged) | Same merge decision as above | — |
| `backend/agent/src/agent/reconstruction_test.go` (test only) | AG-05.3's load-bearing property test | A reconstruction consumer (the "session log" surrogate) that groups events by (message id, tool call id) and asserts completeness regardless of interleaving. **The test, not the helper, is the deliverable** — the helper is test-only code and never ships in production |
| `backend/agent/src/agent/message_events_test.go` | `envelope_test.go`'s pattern | AG-05.1's 3 charter scenarios + 1 bite (no-delta-to-accumulated-route) + per-rule expansion |
| `backend/agent/src/agent/tool_events_test.go` | `stream_check_test.go`'s pattern | AG-05.2's 3 charter scenarios + per-rule expansion |
| `openspec/changes/cachicamas-agent-message-tool-events/specs/agent-message-tool-events/spec.md` | AG-04's spec shape | `R-AMT-0NN` / `S-AMT-0NN` requirements and scenarios |
| `openspec/changes/cachicamas-agent-message-tool-events/{proposal,design,tasks,apply-progress,verify-report}.md` | AG-04's per-phase artifacts | — |

### Files AG-05 explicitly does NOT touch

- `backend/agent/src/agent/{event_descriptor,sequence,failure,stream_check}.go` — design AD-1/AD-2/AD-5 hold. AG-05.3's reconstruction test exercises them as shipped.
- `backend/agent/src/agent/{import_boundary,ambient_authority,doc_contract_guard}_test.go` — AG-03's guards, must stay green with zero edits to their own logic (NFR-AEV-003 inherited).
- `backend/agent/src/agent/doc_contract_guard_test.go` — same.
- `backend/agent/{go.mod,go.sum,Makefile,.golangci.yml}` — no new dep, no tooling change.
- `backend/agent/src/ai/**` — frozen Layer 1 surface (AG-12 guards it later); AG-05 **reads** `ai.MessageID` and `ai.ToolCallID` (via `ai.ToolCall`) per S-AGV-019, edits nothing.
- `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` status header — changes once AG-05 is merged, mirroring `dbd7a33e`/`c203f25c` pattern.

## Approaches

| # | Approach | Brief | Pros | Cons | Effort |
| --- | --- | --- | --- | --- | --- |
| **1** | **Mirror AG-04 exactly** — one `<family>_events.go` per family, six-step procedure, witness table, scope fence retightened | Two new family files (message, tool); `event.go` extends the registry; `event_registry_test.go` extends the witness table; new reconstruction test file exercises the validator unchanged | Reuses AG-04's verified pattern; same convention-closed design (AD-1 mirror); same scope-fence shape; `make test`/`make lint` discipline unchanged. **Lowest-risk for review**: every reviewer already read AG-04 | File count grows (mirrors AG-04); reasoning-vs-text subdecision (A1) must be resolved before apply | **Low** |
| **2** | **Single combined `content_events.go`** for both message and tool | One file instead of two; both families' payloads + constructors share a file | Fewer files; one round of imports | Mixes two families' semantics in one file; AG-04's per-family file discipline broken; harder to read the third family (cost, AG-06) into the same convention | **Low** |
| **3** | **Reuse Layer 1's `ai.TextEvent` / `ai.ToolCallEvent` directly** | Carry Layer 1 events into Layer 2's envelope | Less code | **Violates S-AGV-020** explicitly (*"events, ordering, and failure are wrapped, not reused as-is"*); would also break the AG-04 envelope's per-lane `Sequence` (Layer 1's `ai.Sequence` is per-stream, Layer 2's is per-lane) — and is exactly the kind of cross-layer shortcut AG-04's whole envelope was designed to prevent. **REJECTED** | — |

**A1 (sub-decision under approach 1)** — message-kind shape:

| # | Sub-approach | Kinds registered | Pros | Cons |
| --- | --- | --- | --- | --- |
| **A1a** | **4 kinds** — `MessageStart`, `MessageDelta` (text only), `MessageEnd`, `ReasoningDelta` (standalone, no reasoning start/end) | 4 + 3 tool = **7 kinds** total | Mirrors the simplest reading of `0003:543-558`; reasoning is a fragment-only stream, the loop emits reasoning-start inline as a reasoning-delta or as part of `MessageStart` metadata | Reasoning may need its own start/end later — likely a forward-fix amendment |
| **A1b** | **6 kinds** — symmetric text + reasoning, each with start/delta/end | 6 + 3 tool = **9 kinds** total | Symmetric, easier to reason about; no per-message-kind subtyping | Larger registry; the "reasoning distinguished from text at the event-kind level" reading of the charter is satisfied either way |

**Recommendation: A1a** — 7 kinds total. The charter's "start, deltas, end — reasoning distinguished from text" reading is satisfied by a `ReasoningDelta` kind alongside `MessageStart`/`MessageDelta`/`MessageEnd`. A1b is the symmetric alternative if a future milestone needs reasoning to open and close its own bracket (it doesn't, at AG-05). **The proposal should resolve A1, not defer it** — it sets the scope-fence count and the `R-04` acceptance language.

## Recommendation

**Approach 1 (mirror AG-04) with sub-approach A1a (7 kinds total).** Concretely:

1. **Extend the kind registry** (`event.go`) with 7 new kinds, in declaration order: `MessageStart`, `MessageDelta`, `MessageEnd`, `ReasoningDelta`, `ToolStart`, `ToolProgress`, `ToolEnd`. All use `BracketRoleNone` (they don't open/close brackets) and `PlacementPlacementTurn` (they live inside an open turn — exercising the seam `event_descriptor.go:78-85` was designed for). All use `CardinalityAny` (default zero value). `Terminal: false` on all of them; AG-04.4's W3 fix (`c203f25c`) made the engine read `Terminal` from descriptor data, so any future kind declaring `Terminal: true` is honored — but no AG-05 kind needs it.
2. **Two new family files** (`message_events.go`, `tool_events.go`), mirroring `run_events.go`'s structure. Each carries one closed vocabulary where the charter requires one (`ToolOutcome` for the 3 typed end states; no outcome enum for messages, since the charter does not name a message-level outcome).
3. **Reuse `agent.Failure`** in `ToolEnd` for "execution itself failed" — same wrap, same `Category()`/`Delivery()`/`Retryable()` accessors (W8's `Retryable()` 0% coverage from AG-04 is a real gap; AG-05's tool path is one place that can close it without expanding AG-04's scope).
4. **Reuse `ai.MessageID` and `ai.ToolCallID`** (S-AGV-019 — reused as-is). `ToolStart` carries call identity + tool name + arguments as distinct fields; the call ordinal is a payload field (not envelope), correlating completion order to call order regardless of arrival order.
5. **Reconstruction property test** (`reconstruction_test.go`) is the AG-05.3 deliverable: a test-only consumer that groups a scripted interleaved stream (2 messages × 3 deltas + 2 tool calls × 2 progress events + 1 end, all interleaved) by id, reconstructs each, and asserts independence + completeness. The helper is test-only; the **test is the property**.
6. **Scope-fence retightening** in `event_registry_test.go`: change `S-AEV-090` from "exactly 4 kinds, none named message/tool/..." to "exactly 7 kinds, none named permission/cost/delegation/compaction". This is the bite AG-04.4 reserved for AG-05/AG-06 — registering a 6th message kind or 4th tool kind fails by count before the name scan runs.
7. **No edit** to `stream_check.go`, `event_descriptor.go`, `failure.go`, `sequence.go`, `doc_contract_guard_test.go`, or any AG-03 guard. AG-05's value is in the substrate's extensibility, demonstrated.

### PR strategy

- Session preflight: `PR strategy: single-pr`, `Review budget: 1000 lines`. AG-04's pre-authorized 1000-line budget was exceeded 2.7x (ended at 2687 lines) and braejan's standing instruction (*"authorization to do an exception if the PR is bigger"*, recorded in Engram `sdd/cachicamas-agent-event-envelope/session_summary`) was applied without re-asking. **AG-05 will exceed 1000 lines by the same reasoning**: 7 kinds, each with constructor + accessor + validate, plus two new family files, plus the reconstruction test (which is the most code-heavy single piece — independent script + helper + assertions), plus the witness-table extension. **Forecast: 1500–2200 lines** of code; propose `size:exception` for AG-05 the same way AG-04 was.
- AG-05.1 and AG-05.2 can land in parallel DAG-wise; AG-05.3 depends on both. **Single PR, three internal commits (one per leaf)** — same as AG-04's four commits in one PR. Chaining is not safe because the reconstruction test references both message and tool kinds; splitting would either duplicate the kind registry across PRs or defer the load-bearing property past merge.
- **The proposal should explicitly state the `size:exception` request and reference braejan's standing instruction**, not invent a new exception process.

### Process discipline carried from AG-04

- **Strict TDD, RED-first, real command-verified break-and-restore.** AG-04's deviation-note ("RED evidence is a targeted break-and-restore cycle against already-implemented behavior throughout, disclosed once here rather than repeated per row", `apply-progress.md:62`) is the honest version of RED at this scale. AG-05 should follow the same disclosure, not a fictional chronological RED.
- **`make vuln-check` is NOT in `make all`** (Engram `obs #2944`). Run it explicitly at apply and verify.
- **`golangci-lint cache clean` before treating any lint finding as real** (same memory). The W4 retracted finding was a stale cache; the experiment that closed it was a pristine-worktree lint run.
- **Scenario counts will be re-verified by `sdd-verify`** (W9's "51 vs 45" mistake propagated three artifacts deep). The proposal/tasks/apply-progress should each restate the count: **7 charter Gherkin scenarios in 3 leaves** → expected ~12–18 spec scenarios in `R-AMT-`/`S-AMT-` ids after per-rule expansion, witness-table bite, and the reconstruction property's independent sub-scenarios. State this number explicitly in the proposal and tasks; do not let a "7 vs 12" mistake propagate.
- **Identifier convention** must be resolved at proposal time, not deferred: `R-AMT-`/`S-AMT-` is the natural pick; the proposal should commit to it in the success criteria.

## Risks

| # | Risk | Likelihood | Mitigation |
| --- | --- | --- | --- |
| **1** | **W3 latent trap re-opens under AG-05** if a future AG-05/AG-06 kind declares `Terminal: true` with `BracketRoleNone`. W3 was closed in `c203f25c` for the 4 AG-04 kinds (`c203f25c` made the engine read `Terminal`), so AG-05 ships into a clean state — but the trap remains for any new kind. | Low (AG-05's 7 kinds all declare `Terminal: false`) | The proposal restates AD-1's six-step procedure with the `Terminal` row explicit (step 5 already names the `EventDescriptor` row; the proposal can call out `Terminal bool` separately for kinds that end the stream). The every-kind-constructible guard's bite can be extended to assert that a kind declaring `Terminal: true` is in fact the last event in any sequence that includes it — a behavioral test, not a structural one. **Recorded for the proposal, not silently inherited.** |
| **2** | **Reasoning-vs-text subdecision (A1)** is unresolved at the exploration level. A1a (4 message kinds) vs A1b (6 message kinds) sets the registry size, the scope-fence count, and the "reasoning distinguished at the event-kind level" reading of the charter. | Medium if deferred past proposal | **The proposal must resolve A1** with evidence — either a citation from `0003:543-558` that 4 kinds satisfy "start, deltas, end — reasoning distinguished from text", or a 6-kind symmetric reading with cited support. The resolution is recorded, not silent |
| **3** | **Call ordinal placement (AG-05.2)** — the call ordinal correlates the event to call order regardless of completion order (`0003:582-584`). AG-05 must decide: payload field on every tool event, or a new envelope field. A new envelope field would grow the envelope (currently `{payload, seq, run, turn, hasTurn, parent, hasParent}`) and require editing `event.go`'s accessors, the witness table, and AG-04's identity-tests. | Medium if mis-placed as envelope | **Recommendation: payload field, not envelope field.** The ordinal is meaningful only for tool events, not for run/turn/message — making it envelope would over-shape the envelope for a single family's need. R-13 (`0003:2211`) traces the ordinal to `doc 0002 AI-30` (Layer 1's tool-call ordinal, payload-side). A1-style proposal resolution required |
| **4** | **Scope creep into AG-06's families** — the guard may register a placeholder or `failure-message-string convention` for a family AG-05 does not own. AG-04's own scope-fence shape (`S-AEV-090`) is the precedent. | Medium | The guard's `S-AEV-090` retightens from "exactly 4, no message/tool/permission/cost/delegation/compaction" to "exactly 7 (per A1a), no permission/cost/delegation/compaction" — the message/tool/permission/cost/delegation/compaction name scan drops the two names AG-05 owns and adds nothing. Reviewer checklist item: a 5th message kind, 4th tool kind, or any AG-06 name fails by count before the name scan. Bite test in AG-05.1's own RED log: plant `EventKindScratchCost` and confirm the guard fails naming it |
| **5** | **Reconstruction test (AG-05.3) is a structural test against the validator's own contract** — if the reconstruction helper is too weak, the test passes vacuously. The load-bearing sentence is *"the property a session log will depend on, proven before any producer exists"*; a vacuous pass is the failure mode. | Medium | The reconstruction helper is itself tested: an obvious-loss script (drop a delta; expect a non-equality) bites; an obvious-double script (interleave a delta twice; expect non-equality) bites. Both bites are RED-recorded before the property test itself is GREEN. The helper's own correctness is the precondition for the property test's meaning |
| **6** | **Review budget 1000 lines is exceeded 2–3x by AG-05** for the same reason AG-04 was: 7 kinds, two new family files, a reconstruction test that is the most code-heavy single piece, witness-table extension. | High (against default 400-line budget) / High (against session's 1000-line budget) | Pre-authorize `size:exception` at the proposal, citing braejan's standing instruction. The four-node AG-04 precedent (1.4–2.2x its own 1400–2200 forecast, ended 2.7x) is the empirical anchor. Single PR; **chained PRs are not safe** because AG-05.3's reconstruction test references both message and tool kinds, and the kind registry must be a single committed thing. State this in the proposal, not at apply |
| **7** | **`sdd-archive` truncation risk** (Engram `obs #2944`'s fourth-consecutive archive intervention) — the proposal.md or design.md may be silently truncated by the archive agent; only a byte-diff catches it. | Low (known, named, mitigation documented) | After every archive, byte-diff the artifact against the working-copy size. Same posture as AG-04's archive repair; recorded as standing instruction |

## Open decisions for the next phase (carried forward, not resolved here)

| # | Decision | Why it is genuinely open |
| --- | --- | --- |
| **A1** | Message-kind shape: 4 (`MessageStart`/`MessageDelta`/`MessageEnd`/`ReasoningDelta`) or 6 (symmetric text + reasoning start/delta/end) | Sets the registry size and the charter's "reasoning distinguished at the event-kind level" reading. The proposal must resolve this with evidence from `0003:543-558` |
| **A2** | Call ordinal placement: payload field (recommended) vs new envelope field | Payload preserves the envelope's existing shape; envelope widens AG-04's identity tests. R-13 traces the ordinal to Layer 1 payload-side (`doc 0002 AI-30`), which is evidence toward payload |
| **A3** | Per-family file split: two files (`message_events.go`, `tool_events.go`) or one combined `content_events.go` | AG-04's deviation #1 recorded that `run_events.go` was extended mid-cycle to host both `RunStart` and `RunEnd` halves; AG-05's per-family file discipline is the same default but is revisable. Two files preserve the AG-04 default and keep AG-06's four families (permission/cost/delegation/compaction) in their own future files |
| **A4** | Reasoning vs text: are reasoning deltas a kind of message (A1a's reading) or a separate "stream within a message" (a sub-typing that may need its own start/end)? | A1a is the simplest reading; A1b is the symmetric alternative. Either is charter-defensible |
| **A5** | `size:exception` pre-authorization: assumed yes per braejan's standing instruction, but the proposal should make it explicit and offer the user a chance to chain instead | AG-04's exception was 2.7x; AG-05 will be similar. Chaining is not safe (AG-05.3's reconstruction test needs both families present), so the choice is "single PR + exception" or "defer AG-05" |

## Ready for Proposal

**Yes — the orchestrator can launch `sdd-propose` next.** The substrate is well-understood, the AG-04 lessons are explicit, the open decisions are listed (not silently picked), and the risks are quantified.

What the user should know before the proposal:

1. **AG-05 will exceed the 1000-line review budget**, probably by 1.5–2.2x (forecast 1500–2200 lines). Pre-authorize `size:exception` the same way AG-04 was, or chain — but chaining is not safe because AG-05.3's reconstruction property test references both message and tool kinds in one PR.
2. **AG-05 needs a new spec prefix** — `R-AMT-`/`S-AMT-` is the natural pick (matches the slug `cachicamas-agent-message-tool-events`). The proposal should commit to it.
3. **Five open decisions** (A1–A5) are listed above. The proposal must resolve **A1** (message-kind shape — 4 or 6) and **A5** (single PR + exception) explicitly; A2/A3/A4 are design-phase calls and can be deferred to `sdd-design` with evidence.
4. **No new dependencies, no `go.mod`/`go.sum` edit, no AG-03 guard edit, no `event_descriptor.go`/`stream_check.go` edit.** AG-05's value is in demonstrating the substrate's extensibility — the proposal should make this explicit in the success criteria.
5. **AG-05.3's reconstruction property test is the load-bearing deliverable**, not the helper. The proposal should name it in the acceptance criteria; the helper is test-only code.
6. **`make vuln-check` is not in `make all`** — the apply and verify agents must be told to run it explicitly. The proposal can carry this in the success criteria.
7. **Scenario counts will be re-verified by `sdd-verify`** (W9's three-artifact-deep propagation). The proposal's stated count: **7 charter Gherkin scenarios in 3 leaves**, expected ~12–18 spec scenarios after per-rule expansion + bite + reconstruction property's independent sub-scenarios. State this number explicitly and restate it in `tasks.md` and `apply-progress.md`.

The proposal's first sentence should commit to the answer (per `cognitive-doc-design`): *"AG-05 ships the message and tool execution families on the AG-04 substrate, with reconstruction proven before any producer exists, in a single PR with `size:exception`."* Then expand.

## Key Learnings (for Engram passive capture)

1. AG-05 is the first kind-set to exercise the `PlacementTurn` seam reserved at AG-04.3's design — registering message and tool events with `Placement: PlacementTurn` makes the existing two-level scope engine reject a `message_start` outside an open turn without any code edit.
2. AG-04's W3 latent trap (`Terminal` field declared but unread) was closed in `c203f25c` for AG-04's 4 kinds; AG-05 inherits a clean state but the trap remains for any future kind declaring `Terminal: true` with `BracketRoleNone`.
3. The reconstruction property is the literal expression of the `L2C-04` membership criterion ("a log can reconstruct it"); the test is the deliverable, the helper is test-only.
4. The every-kind-constructible guard's scope-fence must be retightened from "exactly 4" to "exactly 7" (or 9, per A1a/A1b) when AG-05 lands; this is the bite that fails a 5th message kind or 4th tool kind before the name scan.
5. Scenario count errors propagate three artifacts deep (W9); AG-05 should state "7 charter scenarios in 3 leaves, ~12–18 spec scenarios after expansion" in proposal, tasks, and apply-progress, identically.
