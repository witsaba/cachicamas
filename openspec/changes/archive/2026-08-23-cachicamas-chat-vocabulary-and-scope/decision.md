# Decision — CH-00: the chat archetype's vocabulary, seam answers and v1 scope

> **Change**: `cachicamas-chat-vocabulary-and-scope`
> **Milestone**: CH-00 — Record the archetype's vocabulary, seam answers and v1 scope (Wave 0, **1 of 12** milestones of [doc 0005](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md))
> **Node**: CH-00.1 — Answer every question the record must close `[decision]` — the milestone's only node
> **Status**: decided
> **Project**: cachicamas (witsaba) · **Target package**: none — this `[decision]` node ships no code. `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` do not exist yet (§ 5 below)
> **Closes**: doc 0005's CH-00.1 closing checklist — the seven questions at `0005:219-225`, walked in § 11 below — together with R-02, R-03, and doc 0005's own inconsistency-register rows 3, 5, 6 (`0005:96-99`)
> **Sources**: [doc 0005 — the chat archetype's milestones](../../../docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md), charter `0005:202-240` · [ADR 0009](../../../docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md) § D6 (table ownership, `0009:152`; the quoted sentence at `:154-155`) and § D7(a) (the read-with-substitution rule, `0009:169-176`) · [ADR 0005](../../../docs/adr/0005-promote-agent-stack-to-own-module.md) § D2 (target package locations, `0005-adr:198-218`) · [the `agent-layer3-handoff` spec](../../../openspec/specs/agent-layer3-handoff/spec.md) (`R-L3H-006`, `S-L3H-028`, `S-L3H-029`, the generic-client boundary at `:183`) · [AG-23's own `decision.md`](../archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md) — the frozen eleven-seam enumeration this record answers (§ 2, `:31-53`) and the known-limitations register this record inherits (§ 4.2-4.3, `:99-112`)
> **This change's own artifacts, binding and not re-opened here**: `proposal.md` (decisions D-1…D-4), `design.md` (decisions D-A…D-F), `explore.md`, and the delta spec [`chat-archetype-contract/spec.md`](specs/chat-archetype-contract/spec.md) (`R-CHT-001…R-CHT-013`, `NFR-CHT-001…002`) this record satisfies

> [!IMPORTANT]
> **Two constraints inherited from Layer 2's own exit bind every sentence below. Each is restated here in this record's own words, not merely cited.**
>
> **(a) The generic-client boundary.** This record names no coding-archetype concept — no file, no shell, no skill, no terminal, no repository — as a capability of this archetype. This archetype is a **chat** archetype, and chat concepts (a conversation, a message, a participant, a turn) are named freely throughout. A coding-archetype concept is named nowhere below except where a sentence is quoting or citing the boundary rule itself, exactly as it is quoted in full in § 3 (`openspec/specs/agent-layer3-handoff/spec.md:183`).
>
> **(b) `TurnOptions.PreRequestHook` is frozen-and-superseded — never removed, never deprecated.** Wherever this record names that field — § 3 (seam row 7) and § 9 (the `agent-v1-scope:317` citation), the two sections outside this block that name it — it names it as kept, unamended in behavior, carrying no deprecation marker. It is never described as removed and never described as deprecated, anywhere in this record, including footnotes and register rows (`openspec/changes/archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:45`).

---

## 1. How to use this document

**If you are closing one of the seven questions at `0005:219-225`:** § 11 names the section that answers each one. You do not need `chat-archetype-contract/spec.md` in hand and you do not need to open any Go source to close a question — this record answers directly and names the injection point, exactly as the charter's own acceptance requires (`0005:210`, quoted verbatim in § 11).

**If you are opening a later CH milestone (CH-01 … CH-11):** § 3 is the seam table you build against. A row's `v1 answer` is this archetype's committed answer for wave 0 through the whole of v1 scope — *implementing* it is that later milestone's job, never this record's (`0005:212`). § 8 is the deferral register: if what you need is listed there, it is deliberately out of v1, not forgotten.

**If you are reviewing this artifact:** § 4 is where a Layer 2 gap could be misread as a seam answer if it were not structurally separated from § 3 — check that it carries no `Injection point` and no `v1 answer` column before trusting a row there. § 10 is where a found inconsistency would be quietly repaired rather than recorded if this record were not disciplined about it — check the header line reads "recorded, not repaired" before trusting any row there.

---

## 2. Vocabulary — what this archetype calls a conversation, a turn, a message and a participant

*(Closes Q1, `0005:219`.)*

Every noun this archetype uses sits in exactly one of three structurally distinct tables. The **mapped** table carries a `Maps onto` column naming exactly one `VL2-*` identifier; the **coined** table has **no** `Maps onto` column at all, so a coinage cannot be mistaken for a mapping by a reader skimming columns rather than prose; the **inherited** table carries an `Inherited from` column naming exactly one Layer 1 `V-*` identifier and likewise has no `Maps onto` column, so a noun this archetype takes from Layer 1 can be mistaken neither for a Layer 2 mapping nor for a coinage of this archetype's own.

Three tables and not two, because this archetype sits above **both** lower layers. Layer 1's own register draws the boundary this section has to record on both sides of: `V-OUT-02` (`openspec/specs/ai-contract-vocabulary/spec.md:313`) hands *transcript* to Layer 2 while keeping *message* in Layer 1 — "Layer 1 owns the unit, never the collection, its ordering across turns, or its repair." § 2.1 below maps *transcript*; § 2.3 inherits *message*. Both halves of that one sentence are in this record, and a two-table split could hold only one of them.

### 2.1 Mapped nouns — this archetype's own name for a Layer 2 concept it did not invent

| Noun | Maps onto | Layer 2 term | Citation |
| --- | --- | --- | --- |
| run | `VL2-COR-04` (`:135`) | The multi-turn unit the harness drives, bracketed by exactly one run-start and one run-end, ending with a typed outcome — completed, interrupted, or failed | `openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md:135` |
| turn | `VL2-COR-05` (`:136`) | One assistant response plus its tool results. A turn with zero tool calls is still a complete turn — the last one in the run, not a degenerate or partial case | `:136` |
| the loop | `VL2-COR-02` (`:133`) | The stateless half of the runtime; runs one assistant turn and emits events; holds no state between calls | `:133` |
| the harness | `VL2-COR-03` (`:134`) | The stateful half of the runtime; holds the conversation across turns — history, suspension, the cancellation tree, delegation, the compaction trigger | `:134` |
| upward path | `VL2-COR-09` (`:140`) | The one decided surface through which a permission decision, a steering message, or an interrupt re-enters a live run | `:140` |
| transcript | `VL2-COR-10` (`:141`) | The harness's ordered history of messages across a run | `:141` |
| steering | `VL2-COR-12` (`:143`) | A user message that arrives while a turn is in flight; belongs to the **next** turn, never the current one | `:143` |
| message lifecycle | `VL2-EVT-04` (`:165`) | The event family for one assistant message: start, deltas, end, with reasoning distinguished from text | `:165` |
| run outcome | `VL2-EVT-10` (`:171`) | The typed value a run-end event carries: completed, interrupted, or failed. Never absent | `:171` |
| turn outcome | `VL2-EVT-11` (`:172`) | The typed value a turn-end event carries, distinguishing "model finished" from "turn aborted" | `:172` |

Every mapped row's `Maps onto` cell names exactly one `VL2-*` identifier — none names two, none hedges with prose, none is empty.

### 2.2 Coined nouns — this archetype's own vocabulary, with the gap that forced it

| Noun | Gap that forces the coinage | Nearest Layer 2 row and why it is NOT a mapping |
| --- | --- | --- |
| conversation | **Cardinality.** `VL2-COR-04` run (`decision.md:135`) is one harness invocation. This archetype's conversation is continued across turns and, from CH-06 onward, across process restarts, through the frozen transcript seam (`VL2-COR-10`, § 3 row 5). One conversation is therefore realized by **one or more runs** | `VL2-COR-04` run is the nearest row, and is cited above only to show it is *not* the mapping target: a mapping must preserve identity, and a conversation's relation to a run is containment (1-to-N), not identity. Doc 0005's own gloss is the operative one: "the archetype's model of one" conversation, never a component in a browser (`0005:115`) |
| participant | **Sense.** `VL2-COR-14` (`:145`) names "the delegated participant" — the **subagent** sense, fixed because that word already ships in the delegation event-kind names | `VL2-COR-14` is the nearest row and is cited only to rule it out: it is a different sense of "participant" than the human who holds a conversation with this archetype. Layer 2 defines no term for a human participant at all — see below |

Stated plainly, because a clean two-table split can still read as though Layer 2 blessed vocabulary it never defined: **Layer 2 defines no term for a session, no term for a human participant, and no term for a "part."** `VL2-OUT-07` (`decision.md:252`) — "append-only session records with parent chains, under a Layer 3 application's own storage. Layer 2 exposes a transcript … it never writes one itself" — is quoted here as an **exclusion** Layer 2 assigns to Layer 3, never as a term this archetype inherits or maps onto. Under ADR 0009 § D7(a)'s substitution rule (§ 9 below), that quoted "a Layer 3 application" is read as "a Layer 3 archetype" — this archetype, specifically, for session persistence (delivered at CH-06/CH-07).

### 2.3 Inherited nouns — Layer 1's own term, used unchanged, because Layer 2 defines none

| Noun | Inherited from | Layer 1 term | Why it is neither a mapping nor a coinage |
| --- | --- | --- | --- |
| message | `V-REQ-02` (`openspec/specs/ai-contract-vocabulary/spec.md:111`) | "The smallest addressable unit of a transcript: one role plus ordered content. Layer 1's unit of attribution and ordering *within* a request. Not a turn, not a transcript, not a history …" | **Not a mapping**: Layer 2's register defines no term for a message, and says so in its own words — `VL2-COR-10` **transcript** (`archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md:141`) defines a transcript as "the harness's ordered history of messages across a run. **Distinct from a Layer 1 message (`V-REQ-02`)**, which is Layer 1's unit of attribution within one request …" — Layer 2's register deferring to Layer 1 for this noun explicitly. Two `VL2-*` rows carry "message" in their term cell and neither is this noun: `VL2-EVT-04` **message lifecycle** (`:165`) is an **event family** — start, deltas, end — and `VL2-COR-12` **steering (message)** (`:143`) is a user message arriving mid-turn. Both are already mapped under their own names in § 2.1. **Not a coinage**: this archetype invents nothing here. `V-REQ-02` is a row of a promoted, citable Layer 1 spec, cited by identifier from exactly four other promoted specs — `ai-message-roles`, `ai-content-parts`, `ai-tool-messages` and `ai-cache-breakpoints` (the complete set at spec level: five promoted specs contain the identifier, four excluding the one that defines it) — and this archetype uses it unchanged |

That Layer 2 supplies no message term is a fourth Layer 2 absence, and it is deliberately **not** listed with the three in § 2.2. Those three force a coinage, because no layer supplies the noun. This one forces nothing: Layer 1 supplies it, so the honest record is an inheritance. Listing "message" among the coinages would assert that this archetype minted a word Layer 1 already owns and four other promoted Layer 1 specs already cite by identifier (`ai-message-roles`, `ai-content-parts`, `ai-tool-messages`, `ai-cache-breakpoints`) — the precise error this section's three-table structure exists to prevent.

---

## 3. Seam answers — this archetype's v1 answer to every seam AG-23 named

*(Closes Q2, `0005:220`, and Q2b's gap-findings disclosure at § 4.)*

**The generic-client boundary, quoted in full, per the rules block above:** *"without reference to files, shells, skills or terminals"* (`openspec/specs/agent-layer3-handoff/spec.md:183`).

AG-23's own frozen enumeration (`archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:31-53`) names **eleven** seams — eight marked frozen, three marked experimental under the heading at `:49`. That count belongs to AG-23's own archived, immutable source; it is not restated here as a property of this record's own open-ended set. Every row below carries a non-empty `Injection point` and a non-empty `v1 answer` — no row is listed with only one half (`S-L3H-028`, `agent-layer3-handoff/spec.md:195`) — and every `v1 answer` is a direct statement this archetype makes, not a bare citation standing in for one (`S-L3H-029`, `:196`). Rows appear in AG-23's own source order.

| # | Seam | Status | Injection point | v1 answer | Owner milestone |
| --- | --- | --- | --- | --- | --- |
| 1 | The run's model provider | frozen | `Harness.Provider` — a required field on the run value (`harness.go:53`) | CH-04.1's composition root resolves exactly **one** Layer 1 vendor adapter from the environment at startup, from a provider credential and a model selection. No per-participant provider or model picker ships in v1 (deferred, § 8) | CH-04.1 |
| 2 | The decision port | frozen | `TurnOptions.PermissionPolicy` — an optional per-turn field (`loop.go:130`) | CH-10 supplies a real policy: every scheduled tool call is asked, and a call the policy defers becomes a suspension the participant resolves from the browser, carried on the same event stream (`0005:938-940`). Remembered decisions and rule sets over tool names/arguments are deferred (§ 8) | CH-10 |
| 3 | The caller-owned wake handle | frozen | `Harness.Scheduler` — an optional field on the run value (`harness.go:81`); nil constructs one internally | CH-10's HTTP surface supplies this field so the participant's browser decision can reach and resume a suspended call from outside the run's own request — the exact mechanism row 2's approval round-trip rides | CH-10 |
| 4 | The tool registry | frozen | `TurnOptions.Tools` — a per-run field resolving a call's name to an implementation (`loop.go:112`); the `Registry` interface (`tool.go:267`) | Empty through Wave 1 and Wave 2: the registry resolves no name, so every scheduled call yields a typed unresolved-name failure rather than a crash. CH-09 replaces it with a real tool-source port and at least one tool the model can call | CH-09 |
| 5 | The transcript | frozen | `Harness.History` — an optional field on the run value (`harness.go:85`); nil constructs an empty one internally | CH-02.1 holds it inside the conversation object for in-process continuation across turns. CH-06/CH-07 extend the same seam so a freshly constructed `Harness` seeds its `History` from the conversation-store port, surviving a page reload and a process restart | CH-02.1 (in-process); CH-06/CH-07 (durable) |
| 6 | The wider hook-family registration surface | frozen | `Harness.Hooks` — one registration value composed once per turn (`harness.go:74`); the zero value is inert — no lane, no goroutine, no queue | Empty throughout the twelve-milestone v1 graph — this archetype leaves the field at its zero value. No CH milestone's charter states a need to mutate the pre-compaction transcript, observe post-turn, or latch at session start; nothing in v1's scope calls for policy injected here. **No owner in v1** for that reason | none — no owner in v1 |
| 7 | The singular pre-request field | frozen | `TurnOptions.PreRequestHook` — a per-run field composed as element zero of the wider taxonomy's own chain (`loop.go:93`), kept and **frozen-and-superseded**, unamended in behavior and carrying no deprecation marker | Empty throughout v1 — unset, changing nothing. No CH milestone needs to rebuild the outgoing request: no cache-breakpoint requirement and no injected-context requirement is named for this archetype in v1. **No owner in v1** for that reason. The field itself is kept, unamended in behavior, carrying no deprecation marker — never described as removed or deprecated (`archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:45`) | none — no owner in v1 |
| 8 | The tracing-API provider | frozen | `Harness.TracerProvider` — an optional field on the run value (`harness.go:131`); nil resolves to the tracing API's own no-op provider | CH-04.1 is the **only** package in this archetype permitted to install the OpenTelemetry SDK (`0005:549`). It wires a real tracer provider into this exact field; everything below the composition root passes it through unmodified | CH-04.1 |
| 9 | The in-frame delegation door | **experimental — not frozen** | Installed internally per scheduled call, onto that call's own frame; the `DelegationSeam` interface (`delegation_seam.go:46`) is unconstructible from outside the package (`:18-19`) | No coordination tool is shipped or configured. This archetype is one participant, one model, a sequence of turns (`0005:107`); no CH milestone in the twelve-milestone graph configures a subagent tool. **No owner in v1**: ADR 0009 § D3 reserves cross-archetype coordination for a layer above Layer 3 and deliberately does not design it (§ 8, final row) | none — no owner in v1 |
| 10 | The failover policy | **experimental — not frozen** | `Harness.Failover` — an optional field on the run value, consulted exactly once at the retry bound's exhaustion (`harness.go:107`) | Unset throughout v1 — the shipped default declines unconditionally, byte-identical to no policy installed. No CH milestone configures a real failover policy. **No owner in v1**: a real policy re-opens the token budget and the cache prefix and needs its own design — an explicit Layer 2 limitation this archetype inherits unchanged (`archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:106`) | none — no owner in v1 |
| 11 | The context-reduction policy | **experimental — not frozen** | `Harness.ContextStrategy` — an optional field on the run value, consulted once per logical turn (`harness.go:115`) | Unset throughout v1 — the run never compacts, exactly as it behaved before the seam existed. No CH milestone configures a real compaction policy. **No owner in v1**: what triggers a reduction and the instruction driving it is real design work no CH milestone's charter claims; the archetype inherits Layer 2's never-compact default unchanged (`archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:107`) | none — no owner in v1 |

---

## 4. Gap findings against Layer 2's own seam set — findings, not seam answers

*(Closes Q2b, design D-2/D-D disclosure.)*

The frozen enumeration this record answers is exactly the eleven rows in § 3 above. This section **extends nothing** in that enumeration — a finding here is never appended to § 3 under any marking, because appending to § 3 would itself extend a frozen source. The table below has deliberately **no** `Injection point` column and **no** `v1 answer` column, so a finding row is structurally unreadable as a seam answer.

| Finding | Where Layer 2 names it | AG-23 status | Disposition |
| --- | --- | --- | --- |
| Sandbox policy | v2 § 6 seam 3 (`docs/architecture/0001-cachicamas-agent-stack-v2.md:670`); rides the opaque `PolicySlot any` (`backend/agent/src/agent/tool.go:117`), which the scheduler carries without reading | Not in AG-23's eleven-row enumeration | This archetype's v1 tool execution passes no sandbox policy — the same "none" trivial implementation Layer 2's own § 6 table ships (`:670`). A Layer 3 sandbox is a future decision this record does not make |
| Retry classification | v2 § 6 seam 7 (`:674`) — the typed error a provider returns classifies its own retryability | Not in AG-23's eleven-row enumeration | Not a per-archetype seam: Layer 2's harness (`VL2-HAR-06`) already consumes this classification internally. This archetype makes no separate choice about it |
| `Harness.RetryTiming` | Its own doc comment (`backend/agent/src/agent/harness.go:96-97`) calls it "the injected clock and wait-function seam"; the field itself declares at `:100` | Settable but not in AG-23's eleven-row enumeration | Left at its zero value throughout v1 — production defaults resolve via `applyRetryTimingDefaults`; no CH milestone overrides it |
| `Harness.RetryAttempts` | Settable field (`harness.go:94`) — the attempt bound for one logical turn | Settable but not in AG-23's eleven-row enumeration | Left at its zero-default (`defaultRetryAttempts`) throughout v1; no CH milestone sets a different bound |
| `Harness.ContextBudget` | Settable field (`harness.go:120`) — Layer 3's own stated token budget, possibly absent | Settable but not in AG-23's eleven-row enumeration | Left at its zero value (which reads as absent, never as a budget of zero) throughout v1; no CH milestone supplies a budget |
| `Harness.System` | Settable field (`harness.go:56`) — the system prompt every turn is built with | Settable but not in AG-23's eleven-row enumeration | This archetype's own system prompt is assembled and injected here; its content is CH-02's to write, not this record's |

**The `Registry` interface's doc comment.** Where this record notes `backend/agent/src/agent/tool.go:265-266`, the sentence **wraps across the two lines and completes there**: "`ToolSource` port (G6) is AG-20's widening." Reading `:265` alone stops mid-sentence; the sentence itself is complete once `:266` is read together with it.

---

## 5. Identity — name, package path, composition root

*(Closes Q3, `0005:221`.)*

This archetype's name is `cachicamas_chat`. Its package path is `backend/agent/src/chat/`; its composition root's path is `backend/agent/src/cmd/chat/` — citing doc 0005's own header (`0005:8`) and ADR 0005 § D2's location mapping (`docs/adr/0005-promote-agent-stack-to-own-module.md:198-218`).

**Neither path exists yet, and there is no `backend/agent/src/cmd/` directory of any kind on disk.** Stating the paths as though they already existed would make this record false on the day it lands. Their creation is **CH-01.1**'s to own (`0005:228`), not this record's — CH-00.1's own out-of-scope clause says exactly this (`0005:228`).

---

## 6. Persistence — does this archetype write to a database, and to whose tables

*(Closes Q4, `0005:222`.)*

Yes. This archetype persists conversations server-side, per participant, in **tables it owns** — the answer names the **owner**, not merely the intent, per `0005:222`'s own instruction and ADR 0009 § D6 (`docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md:152`, the quoted sentence at `:154-155`): *"Each business system owns its own tables; no archetype writes to another system's schema."*

The conversation-store port and its adapters are CH-06's to define; the tables themselves — owned exclusively by this archetype, never shared with or written by another business system — are CH-07's to migrate and populate. Neither exists yet; § 5's "neither path exists" statement applies equally to their contents.

---

## 7. Frontend — which one attaches, and what of its wire is already frozen

*(Closes Q5, `0005:223`.)*

The **existing web chat frontend**, the first real occupant of the `CUSTOM` consumer slot that `docs/architecture/0001-cachicamas-agent-stack-v2.md` §2.2 reserves on the agent event stream — §2.2's frontend subgraph names three nodes, `TUI`, `Print mode` and `CUSTOM["Future: IDE / RPC"]`, and does not itself name this frontend — attaches to this archetype. Its wire is frozen by the promoted capability `frontend-chat-layer1`, `REQ-1` through `REQ-7` — every one of the seven is **inherited, and not this document's to design**:

| Requirement | What it freezes | Status here |
| --- | --- | --- |
| `REQ-1` | Submit-a-turn happy path: `POST /api/agent/turns` → `{turn_id, stream_url}`; `EventSource(stream_url, {withCredentials:true})`; accumulate `message.delta` into one assistant bubble; close on `turn.end` | Inherited |
| `REQ-2` | Mid-stream cancellation: `DELETE /api/agent/turns/:id` in addition to `EventSource.close()` when the stream closes for any reason other than a natural `turn.end` | Inherited |
| `REQ-3` | The auth guard chain, described below | Inherited |
| `REQ-4` | A typed backend error renders inline; the client never auto-retries | Inherited |
| `REQ-5` | The dev-mode offline literal `"backend not wired — see PR for backend wire"` | Inherited; retired only by a recorded spec delta at CH-05.2, never a silent deletion (§ 8) |
| `REQ-6` | Assistant text renders through `renderSanitizedMarkdown`; raw model HTML is never injected | Inherited |
| `REQ-7` | One colocated spec per source unit, on the frontend chat tree | Inherited |

**The auth guard chain — corrected location.** Where this record cites the guard chain `REQ-3` mandates, it cites `frontend/src/routes/home/layout.tsx`: the `onRequest` handler at `:30`, its three calls at `:31-33` (`setSsrCookieHeader` → `requireAuthRedirect` → `await requireOwnboarding`), its doc comment at `:21-29`, and its imports at `:15-17`. The promoted `frontend-chat-layer1/spec.md`'s own citation of this chain — at its lines `9`, `40` and `134` — names `frontend/src/routes/home/index.tsx:39-51` instead. That source carries no guard chain of any kind; its own header comment says so directly ("The guard chain lives in this section's layout," `home/index.tsx:5-6`). This defect is recorded, not repaired, in § 10 below as **F-3**.

No frozen-wire element is proposed for change here. The `REQ-5` literal is retired only by a recorded spec delta at **CH-05.2**, never by a silent deletion (`0005:635`).

---

## 8. Explicitly deferred until after v1

*(Closes Q6, `0005:224`.)*

Reproduced row for row from doc 0005's own "Explicitly deferred until after v1" table, source `0005:997-1011`. Every row carries the seam it attaches to — no row here is a bare exclusion.

| Capability | Seam where it attaches later | Decided by |
| --- | --- | --- |
| Resumable mid-turn reconnect — replaying events a dropped subscriber missed | The turn's event stream, which would need a durable ordered log behind it. Reload fidelity (CH-06/CH-08) restores what was *recorded between turns*; it does not replay a stream — the failure mode this deferral must not be mistaken for having fixed is the one [opencode #25657](https://github.com/anomalyco/opencode/issues/25657) shipped | Research findings 3 and 4; recorded at CH-03.2 |
| Multi-device and multi-tab delivery of one turn | The same event stream and log | Research finding 3 |
| MCP tool sources | The CH-09 tool-source port — the seam ADR 0009 § D4 attaches to when this archetype reaches a business system | ADR 0009 § D4; doc 0005's own register row 5 |
| Sandboxing of tool execution | The execution seam Layer 2 already carries (v2 § 6 seam 3); v1's answer is "none," stated — see § 4 above | CH-00.1 (this record) |
| Remembered permission decisions and rule sets | The CH-10 permission policy port | CH-10's charter |
| Provider and model selection by the participant | The composition root's resolution point (CH-04.1); v1 selects one model from the environment (§ 3 row 1) | CH-04's charter |
| Cost and usage display | The runtime's accounting seam and the price-table port doc 0004 defines; promoting it to shared Layer 3 code is a separate decision | v2 § 7 G10 |
| Conversation branching, renaming, deletion and search | The CH-06 store port and CH-08.2's listing | CH-08's charter |
| A websocket transport | The CH-03 surface. One-directional agent output does not justify it; only a bidirectional need would | Research finding 2 |
| Promoting any part of this archetype into shared Layer 3 code | These parts already sit behind their own ports and read nothing ambient, so promotion stays a move rather than a rewrite. The honest time to decide is when a second archetype exists to be measured against | Doc 0004's deferred register, inherited |
| Cross-archetype coordination | A layer above 3, whose position ADR 0009 § D3 reserves and deliberately does not design | ADR 0009 § D3 (R-17) |

**AG-23's own known-limitations register is cited as inherited Layer 2 input, and is not reproduced here.** Its four rows — no production coordination tool for nested runs, a failover policy that always declines, no default context-reduction policy, and the abandoned-consumer contract inherited from Layer 1 — live at `archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:103-108`. Reproducing them here would create a second count-bearing copy that drifts against its source; § 3 rows 10 and 11 above already cite the two of them this archetype's own seam answers touch.

This register adopts AG-23 § 4.3's own rule for itself (heading at `:110`), quoting the archived text verbatim rather than paraphrasing it: a defect "appears **nowhere** in the register above. Laundering it as a limitation would make this statement false on the day it lands." No `F-1`, `F-2` or `F-3` row appears in this deferral register — each lives only in § 10's inconsistency register below.

---

## 9. The substitution rule for "a Layer 3 application"

*(Closes Q7, `0005:225`, register row 6.)*

ADR 0009 § D7(a) has not yet landed as its own SDD change. Until it does, the following cited Layer 2 artifacts still say "a Layer 3 application" rather than "a Layer 3 archetype":

- `openspec/specs/agent-contract-vocabulary/spec.md:146` (the `R-AGV-009` consumer-term clause)
- `openspec/specs/agent-contract-vocabulary/spec.md:153` (`S-AGV-028`)
- `openspec/specs/agent-contract-vocabulary/spec.md:335` (Trap 4's corrected phrasing)
- `openspec/specs/agent-layer3-handoff/spec.md:17`
- `openspec/specs/agent-layer3-handoff/spec.md:34` (the consumer-proof definition — "a future Layer 3 application in miniature")
- `openspec/specs/agent-layer3-handoff/spec.md:177`
- `openspec/specs/agent-layer3-handoff/spec.md:196` (`S-L3H-029`)
- `openspec/specs/agent-v1-scope/spec.md:317` (the seam-1 widening clause, which is also where AG-23's declining of the `PreRequestHook` removal is recorded — the field is kept and **frozen-and-superseded**, unamended in behavior and carrying no deprecation marker, and that cited clause states it in those words: "frozen-and-superseded with a post-v1 removal path")

**How this list was derived, so a later reader can re-check it rather than trust it.** The list is the result of searching `openspec/specs/` for the phrase `Layer 3 application`, **not** for the exact string `a Layer 3 application`. Measured against the tree this change is cut from, the two patterns differ by exactly one occurrence:

```
grep -rn 'Layer 3 application'   openspec/specs/   →  8 hits across 3 specs
grep -rn 'a Layer 3 application' openspec/specs/   →  7 hits across the same 3 specs
```

The occurrence the exact string misses is `agent-layer3-handoff/spec.md:34`, which carries an intervening modifier — "a **future** Layer 3 application" — so the looser pattern is the correct one and any later re-derivation of this list MUST use it.

**Two distinct failures produced the short list this paragraph replaces, and they are named separately here so a later reader defends against both rather than conflating them.** The first is the pattern error above.

The second is neither a pattern error nor a scope error. This record has already named its cause wrongly once, so the evidence is quoted here rather than summarised. This change's own `explore.md:88` records the exploration's grep as matching *"exactly the cited set, plus two additional unlisted occurrences that don't contradict"* — and, as originally written, enumerated **one** of those two (`agent-layer3-handoff/spec.md:34`) while leaving the other unnamed. That note has since been corrected in place to name both; the account here describes the state that produced the short list, not how that note reads today. Six cited plus two unlisted is eight — the ground truth — and the loose pattern over the two specs the cited six live in returns only seven, so the unnamed eighth is reachable only from `agent-v1-scope/spec.md:317`. The search saw the whole tree and saw that spec.

The failure was **carrying the six-item list forward past an acknowledged, unresolved remainder**. A note reading "plus two additional" while listing one is an open count; an open count that is never closed silently drops whatever was never written down, and every artifact downstream inherits the short list with no signal that anything is missing. That is the mechanism to guard against here — not a mis-scoped search, and not the pattern error above, which would not have hidden `:317` in any case since the exact string does match it.

ADR 0009 § D7(a)'s substitution rule, quoted verbatim (`docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md:174-176`):

> "Until that change lands, 'a Layer 3 application' in shipped Layer 2 artifacts is read as 'a Layer 3 archetype', exactly as ADR 0004's amendment established read-with-substitution before it."

Every citation of one of the eight occurrences above, anywhere in this record or this change's other artifacts, is read under this substitution rule — this archetype is what "a Layer 3 application" resolves to. This change does **not** perform the rename in any promoted spec; that remains ADR 0009 § D7(a)'s own SDD change to make (`0005:212`'s scope fence).

---

## 10. Inconsistency register — recorded, not repaired. No promoted spec is modified by this change.

*(Design D-F.)*

| # | Side A (cited) | Side B (cited) | Disposition |
| --- | --- | --- | --- |
| **F-1** | `openspec/specs/agent-contract-vocabulary/spec.md:339` carries a placeholder — "[REGISTER CONTINUES WITH 86 ROWS IN 6 CATEGORIES — SEE ARCHIVED DECISION.MD FOR THE COMPLETE SNAPSHOT]" — and the 86 `VL2-*` rows exist only in `openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md` | `R-AGV-001` (`agent-contract-vocabulary/spec.md:36`) and `S-AGV-001` (`:40`) forbid the archive as a citation target for a term | Recorded. Repair is a future change against `agent-contract-vocabulary` |
| **F-2** | ADR 0009 § D7(a) (`0009:169-183`) is still open — no change has renamed "a Layer 3 application" to "archetype" in Layer 2's promoted specs | The **eight** occurrences listed in § 9 above (`agent-contract-vocabulary/spec.md:146, :153, :335`; `agent-layer3-handoff/spec.md:17, :34, :177, :196`; `agent-v1-scope/spec.md:317`) still say "a Layer 3 application" | Recorded. F-2 spans **three** promoted specs — `agent-contract-vocabulary`, `agent-layer3-handoff` and `agent-v1-scope`. **F-1 and F-2 both land on `agent-contract-vocabulary/spec.md`, so one future change can close both there**; closing F-2 completely also requires the other two |
| **F-3** | `openspec/specs/frontend-chat-layer1/spec.md` cites the auth guard chain at `:9`, `:40` and `:134` as `frontend/src/routes/home/index.tsx:39-51` | That source carries no guard chain — the correct location, cited by this record throughout § 7, is `frontend/src/routes/home/layout.tsx` (`onRequest` `:30`, its three calls `:31-33`, doc comment `:21-29`, imports `:15-17`) | Recorded. Repair is a future change against `frontend-chat-layer1` |

**Recorded, not repaired. No promoted spec is modified by this change.** This change's own "Modified capabilities" is **None** — F-1, F-2 and F-3 are found and cited against **four** promoted specs — `agent-contract-vocabulary` (F-1 and F-2), `agent-layer3-handoff` (F-2), `agent-v1-scope` (F-2) and `frontend-chat-layer1` (F-3) — and repaired by none of them here (proposal D-3). The targets are listed with the rows they belong to, so that a shortfall shows up against the register three lines above rather than only inside a numeral. That is what listing buys and no more — it does not make the count self-maintaining, and a fifth target would falsify the numeral exactly as `agent-v1-scope` falsified the previous one when it joined F-2 and moved the union from three to four. This record's own § 9 is the case study in an enumeration going short in silence.

---

## 11. Closing verification

*(Mirrors AG-23 `decision.md` § 6.)*

Every one of the seven closing questions (`0005:219-225`), walked in order, naming where its answer lives:

| # | Question | Answered in |
| --- | --- | --- |
| 1 | Conversation, turn, message, participant, and for each the Layer 2 term it maps onto — or, where Layer 2 supplies none, the Layer 1 term it inherits (`message`, § 2.3) or the gap that forced the coinage | § 2 |
| 2 | Every AG-23 seam's v1 answer and injection point, deliberately empty answers listed never omitted | § 3 |
| 2b | Gap findings against Layer 2's own seam set | § 4 |
| 3 | Archetype name, package path, composition root | § 5 |
| 4 | Database, and to whose tables | § 6 |
| 5 | Which frontend attaches, and what of its wire is frozen | § 7 |
| 6 | What is out of v1, and the seam each deferral attaches to | § 8 |
| 7 | Which artifacts still say "a Layer 3 application," and the substitution rule | § 9 |

**The acceptance statement, mirroring `0005:210` verbatim:** Given the record, when a reader asks what this archetype's answer is to any seam Layer 2 names, then the record answers it directly and names the seam's injection point, without the reader consulting source.

**Self-review.** Taking seam 6 of § 3 ("the wider hook-family registration surface") at random: the injection point is `Harness.Hooks`, a registration value composed once per turn; the v1 answer is that it is left at its zero value throughout v1, with no owning milestone, because nothing in the twelve-milestone graph calls for policy injected there. Both halves are supplied without opening `harness.go`. Every one of the seven questions above names the section its answer lives in, and none of those seven answers appears only in `chat-archetype-contract/spec.md` — the spec asserts shape; this record carries every answer completely (design D-A).

**Confirmed with the spec set aside.** Re-reading § 2 through § 9 with `chat-archetype-contract/spec.md` deliberately out of hand: every one of the seven questions above still resolves from this record alone, and no answer disappears.
