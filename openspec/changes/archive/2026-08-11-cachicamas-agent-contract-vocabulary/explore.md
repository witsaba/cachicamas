# Explore — Layer 2 contract vocabulary

> **Change**: `cachicamas-agent-contract-vocabulary`
> **Milestone**: AG-00 — Record the Layer 2 contract vocabulary
> **Node**: AG-00.1 — The vocabulary decision `[decision]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Target package**: `backend/agent/src/agent/` — **does not exist yet; no code is written by this change**
> **Sources**: [doc 0003 — Layer 2 task graph](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) (v2, restructured 2026-08-11) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0004](../../../docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md) · [ADR 0005](../../../docs/adr/0005-promote-agent-stack-to-own-module.md) · [ADR 0006](../../../docs/adr/0006-resolve-skill-and-prompt-source-of-truth.md) · [ADR 0007](../../../docs/adr/0007-adopt-dag-convention-for-task-graphs.md) · doc 0002 (node grammar, § "How to read this document") · Layer 1 precedent: `openspec/changes/archive/2026-07-31-cachicamas-ai-contract-vocabulary/{explore,decision}.md` · `openspec/specs/ai-contract-vocabulary/spec.md` (the live L1 register, 114 terms) · shipped `backend/agent/src/ai/`, `backend/agent/src/agenttest/`, `backend/agent/src/handoff/` (read-only, for identity-reuse grounding) · `openspec/project.md`, `openspec/AGENTS.md`, `openspec/config.yaml`
> **Authoring constraint**: doc 0003's authoring constraint binds this whole change. The vocabulary is **conceptual**. No Go type name, field name, or signature appears in any artifact of this change. Citations may point at contract documents or at shipped **Layer 1** code; never at Layer 2 code, which does not exist (confirmed: `backend/agent/src/agent/` has zero files on disk).

---

## 1. What this milestone is, in one paragraph

AG-00 is explicitly named in doc 0003 as "the AI-01 of this layer: names before code." It ships zero Go and zero files under `backend/`. Its deliverable is a recorded vocabulary artifact that fixes, once, the meaning of every noun the 23 remaining Layer 2 milestones (AG-01 … AG-23) will use, so that no later milestone's SDD re-litigates what a *run*, a *turn*, a *transcript entry*, a *tool call/result pair*, a *suspension*, or a *steering message* is. Every later AG milestone's charter must be able to cite a vocabulary entry instead of defining a term inline (AG-00's acceptance criterion, doc 0003 § "Wave 0 — Decide").

## 2. Why naming late is the failure mode this milestone exists to prevent — and evidence it already happened twice at this layer

The Layer 1 precedent (AI-01) proved the pattern with four shipped-then-corrected defects (C1–C4) that were all "a contract frozen around a noun whose definition was still open." Layer 2 has not shipped any code yet, so it cannot repeat C1–C4 literally — but the **architecture reference itself** (doc 0001) already needed two vocabulary corrections after the fact, which is the same failure mode one level up:

1. **"the portable brain" → "the portable agent runtime"** (doc 0001, amendment dated 2026-08-10). The spike narrative's *brain* metaphor claimed the layer thinks/reasons/decides, which is exactly the design mistake §§ 4.1–4.2 forbid (policy leaking into Layer 2). The correction changed no boundary, no seam, no dependency rule — only a name that had been describing the wrong thing for weeks.
2. **"Layer 3" conflated with `cachicamas_coding`** (doc 0001, amendment 2, same date). § 5 previously read as though Layer 3 *were* the coding agent. It is not — Layer 3 is a position in the stack; `cachicamas_coding` is its first occupant. This conflation, if left unfixed, would have let a later milestone write a contract, test name, or acceptance criterion in terms of a *coding* agent specifically — precisely the leak AG-23's consumer proof exists to catch structurally.

Both corrections are folded into doc 0003's body already, and AG-00.1 item 4 exists specifically to make sure a third occurrence never gets that far: the name and its exclusions are recorded **before** AG-03 writes the first line of code, not after a milestone ships around the wrong word.

A third, milestone-plan-level instance: doc 0003's own **inconsistency register** (§ "Sources and research", rows 1, 2, 5, 6) records defects found in the *v1* restructure of this exact document — a too-narrow import allowlist, a parallelism claim that contradicted a dependency edge, charter-declared dependencies missing from the node graph, and one-way `Blocks:` edges — all naming/consistency defects in the plan that names the vocabulary milestone itself. The generalization holds one level further down than AI-01 stated it: not just contracts, but *plans about contracts* drift when a noun's definition is still open when the plan is written.

## 3. The term inventory

### 3.1 Proposed category scheme

Layer 1's six categories (`V-REQ`/`V-STR`/`V-MET`/`V-FAIL`/`V-PRV`/`V-OUT`) map onto the shape of a *model adapter* (request in, stream out, metadata, failure, provider surface, exclusions). Layer 2 is a *runtime* (loop + harness), so its natural axes are different. Proposed six-category scheme, offered as a recommendation for AG-00.1/the proposal phase to adopt or amend — **not a final ID assignment**, which is decision-phase work:

| Proposed code | Category | Rough L1 analog |
| --- | --- | --- |
| `V2-COR` | Core identity — runtime, loop, harness, run, turn, provider call/attempt, transcript | none (new to L2) |
| `V2-EVT` | The event envelope — kind, the eight families, ordering invariants, lifecycle outcomes | `V-STR` (content half) |
| `V2-LOOP` | Loop mechanics — pre-request hook, tool execution contract, permission protocol, termination | none (new to L2) |
| `V2-HAR` | Harness mechanics — history, pairing invariant, steering, cancellation tree, retry/failover, cost | none (new to L2) |
| `V2-SEAM` | Cross-cutting seams — context strategy, compaction, delegation, hook taxonomy | `V-PRV` (proving/seam apparatus) |
| `V2-OUT` | Excluded — named, attributed to Layer 3, never defined here | `V-OUT` |

**Naming the prefix is itself a decision AG-00.1 should make explicitly.** Reusing `V-` with new category letters risks an id like `V-EVT-01` colliding in meaning (not in text — Layer 1 already used `V-STR`) with a reader's expectation that all `V-*` ids are Layer 1's. A distinct prefix (`V2-`, `W-`, or similar) keeps a citation self-describing about which layer's register it resolves against, which matters because doc 0003 already cites Layer 1 rows (`V-REQ-*`, `V-STR-*`, etc.) directly inside Layer 2 milestone text (e.g. AG-07.2 "the reasoning round-trip token" = `V-REQ-11`). Two disjoint prefixes make every citation unambiguous without a lookup.

### 3.2 The terms, walked from AG-00's charter and every AG-01…AG-23 charter/scenario

This is not a final ID table (that is AG-00.1's own output); it is the walked inventory the next phase should turn into one. Grouped by the proposed categories above, each with its defining source and the milestone that will consume it as a precondition.

**V2-COR — core identity**

| Term | Working definition (from sources, not invented here) | First consumer(s) |
| --- | --- | --- |
| the (portable agent) runtime | The loop-plus-harness assembly; not a third thing wrapping them, not a synonym for either alone (doc 0001 § 4 amendment; doc 0003 "Outcome first") | every milestone |
| the loop | The stateless part: one assistant turn, schedules, no state between calls (doc 0001 § 4.1) | AG-03…AG-11 |
| the harness | The stateful part: history, suspension, cancellation tree, delegation, compaction trigger (doc 0001 § 4.2) | AG-12…AG-20 |
| run | The multi-turn unit the harness drives, bracketed by run-start/run-end (doc 0003 AG-04.2, AG-13) | AG-04, AG-13 |
| turn | One assistant response plus its tool results (doc 0003 AG-00 charter). Terminal turn may have zero tool calls (AG-00.1 item 1) | AG-04, AG-07, AG-11 |
| provider call / attempt | One Layer 1 stream. AG-00's charter states "one turn may span several provider calls only via retry" — a distinction not made explicit in doc 0001 itself; see § 8 conflicts below | AG-11, AG-15, AG-16 |
| transcript | The harness's ordered history of messages, distinct from Layer 1's `message` (`V-REQ-02`) and from `V-OUT-02` (Layer 1's own exclusion naming this as Layer 2's) | AG-12 |
| pairing invariant | Every tool call has a matching result, enforced at the history boundary (doc 0001 § 4.2; doc 0003 R-07) | AG-12 |
| steering (message) | A mid-run user message queued while a turn is in flight, entering history at the next turn boundary (doc 0003 AG-13.2) | AG-13 |
| suspension / resumption | A scheduled call paused on the permission protocol, resumed by a decision arriving through the decided upward path (doc 0003 AG-10, AG-01.1 item 5) | AG-01, AG-10 |
| delegation / parent relationship | A subagent = a harness invoked from within a tool; nested cancellation/cost/permission scope; parent-identified events (doc 0001 § 4.2, § 6 seam 12; doc 0003 AG-19) | AG-06.3, AG-19 |
| the cost event('s token-only scope) | Per-turn/cumulative token figures only; money is Layer 3 enrichment (doc 0003 AG-06's reconciled note, AG-16) | AG-06, AG-16 |

**V2-EVT — the event envelope**

| Term | Definition source | Consumer |
| --- | --- | --- |
| agent event envelope | The only contract between Layer 2 and everything above it (doc 0001 § 4.3; doc 0003 R-04) | AG-04 |
| event kind / eight families | run lifecycle, turn lifecycle, message, tool execution, permission, cost, delegation, compaction (doc 0001 § 4.3 table) | AG-04, AG-05, AG-06 |
| the four envelope invariants | indexed deltas, explicit nesting (parent id), non-blocking observers, typed errors (doc 0001 § 4.3) | AG-04.3, AG-19.1, AG-20.2, AG-11.2 |
| stream-contract validator | The reusable checker run over any hand-built or produced event sequence (doc 0003 AG-04 deliverable) | AG-04, AG-23 |
| run outcome | completed / interrupted / failed, carried on run-end (doc 0003 AG-04.2) | AG-04, AG-14 |
| turn outcome | "model finished" vs "turn aborted", typed (doc 0003 AG-04.2) | AG-04, AG-11 |
| decision-required / decision-made / resolution-remembered | the three permission-family event kinds (doc 0003 AG-06.1) | AG-06, AG-10 |
| subagent-started / subagent-ended | the delegation-family event kinds — note the naming tension with "child harness", flagged in § 8 | AG-06.3, AG-19 |
| compaction-started / -finished / -failed | the compaction-family event kinds (doc 0003 AG-06.4) | AG-06, AG-18 |

**V2-LOOP — loop mechanics**

| Term | Definition source | Consumer |
| --- | --- | --- |
| pre-request hook | The seam immediately before the provider call — the last moment the outgoing request exists as data (doc 0001 § 6 seam 1; L1's `V-REQ-29` request rebuild is the mechanism it stands on) | AG-08 |
| byte-stable prefix / prefix stability | Tool and system regions byte-identical across turns; message region append-only (doc 0003 AG-08.2, **G4**'s L2 half) | AG-08 |
| tool execution contract | What a tool is to Layer 2: declaration + effect class + typed failure mode (doc 0003 AG-09) | AG-09 |
| effect class (read / mutating / execute) | The scheduler's concurrency-policy discriminator (doc 0003 AG-09.1) | AG-09 |
| ordered rejoin / call ordinal | Results rejoin in call order regardless of completion order; ordinal = L1's `V-STR-21`/`V-REQ`'s call ordinal, restated at L2 (doc 0003 AG-09.3) | AG-09 |
| the (per-call) policy slot | The opaque sandbox-policy parameter the execution call carries and never reads (doc 0001 § 6 seam 3; doc 0003 AG-09.1) | AG-09 |
| permission protocol (ask–suspend–resume) | The protocol, never the policy answer (doc 0001 § 4.1; doc 0003 AG-10) | AG-10 |
| the four permission outcomes | allow-once / allow-always / deny / modify-input (doc 0003 AG-10.2) | AG-10 |
| finish-reason dispatch | Exhaustive dispatch over L1's closed finish-reason vocabulary (`V-MET-01`…`08`) into distinct typed turn outcomes (doc 0003 AG-11.1) | AG-11 |
| typed turn failure | Category, retryability, partial-output discriminator carried upward — L1's `V-FAIL-*` restated at turn scope (doc 0003 AG-11.2) | AG-11, AG-15 |

**V2-HAR — harness mechanics**

| Term | Definition source | Consumer |
| --- | --- | --- |
| history | The append-only, boundary-validated transcript store (doc 0003 AG-12) | AG-12 |
| orphan synthesis | Synthesizing typed results for tool calls orphaned by interruption, before the next turn (doc 0003 AG-12.2) | AG-12, AG-18 |
| the run driver | Runs the loop repeatedly until a terminal finish reason (doc 0003 AG-13) | AG-13 |
| interrupt vs shutdown | Two distinguishable signals: abort-turn-keep-session vs flush-and-exit (doc 0003 AG-14) | AG-14 |
| bounded wind-down | Cancellation completes within a documented bound even against a cancellation-deaf tool (doc 0003 AG-14.3) | AG-14, AG-21 |
| retry policy (harness half) | Consumes L1's typed evidence; partial-output failures are never silently retried (doc 0003 AG-15, R-15) | AG-15 |
| failover seam | Named injection point, v1 implementation declines (doc 0001 § 6 seam 8; doc 0003 AG-15.3) | AG-15 |
| composed bounds | harness attempts × wire-level (L1) attempts — the combined retry ceiling (doc 0003 AG-15.2 note) | AG-15 |
| cost aggregation | Per-turn and cumulative, estimate-vs-final labelled, retries and compaction spend included (doc 0003 AG-16) | AG-16 |

**V2-SEAM — cross-cutting seams**

| Term | Definition source | Consumer |
| --- | --- | --- |
| context strategy (seam) | Consulted with transcript + budget before every provider call; v1 default never compacts (doc 0001 § 6 seam 5; doc 0003 AG-17) | AG-17 |
| token accounting / capability-discovered counting | Optional provider capability, type-asserted; documented estimate fallback (doc 0001 § 6 seam 6; doc 0003 AG-17.2; L1's `V-PRV-17` token counting) | AG-17 |
| compaction | A model call with its own provider/cost/cancellation; protects recent turns; never orphans a pair; recorded; interruption-safe (doc 0001 § 4.2, § 7 G3; doc 0003 AG-18) | AG-18 |
| compactable span / protected turns | The transcript region compaction may replace vs the region it must leave byte-identical (doc 0003 AG-18.2) | AG-18 |
| summary entry (compaction artifact) | Typed transcript entry distinguishable from a model message (doc 0003 AG-18.2) — resolves the boundary case in § 5 below | AG-18 |
| on-demand entry point | The `/compact`-equivalent invocation, same mechanics as strategy-triggered, refused mid-turn (doc 0003 AG-18.5) | AG-18 |
| re-entrancy / child harness / subagent | The harness invocable from inside a tool — structural property, no v1 subagent *tool* (doc 0001 § 6 seam 12; doc 0003 AG-19). See naming-consistency flag in § 8 | AG-19 |
| derived permission scope | A child's policy scope derived from the parent's (doc 0003 AG-19.3) | AG-19 |
| the hook taxonomy | pre-request, pre-compact, post-turn, session-start; mutating vs observing (doc 0001 § 7 G11; doc 0003 AG-20) | AG-08, AG-20 |
| observer asynchrony | Observers never synchronous on the streaming path — envelope invariant 3, mechanically (doc 0003 AG-20.2) | AG-01, AG-20 |
| the § D3 attribute vocabulary (Layer 2 extension) | Span/attribute names for run, turn, tool-execution, compaction spans — **AG-22.1's own decision, not AG-00's**; flagged out of scope in § 8 | AG-22 |
| the Layer 3 readiness contract / consumer proof | The frozen v1 surface and the generic (non-coding) test that proves it sufficient (doc 0003 AG-23) | AG-23 |

**V2-OUT — excluded, named here only**

Layer 1's own `V-OUT-01`…`V-OUT-17` already assign most Layer-2-and-above concerns an owner (agent turn → L2, transcript → L2, session → L3, tool execution → L2 schedules/L3 confines, permission → L2 protocol/L3 policy, compaction → L2, cost → L2 emits, price → L3, frontend → above L3, and eight more in § 8.2 of the L1 register). AG-00 does not redefine these; it should **cite them by their `V-OUT-*` id** rather than re-paraphrase, exactly as AI-01.1's own rule states ("a paraphrase is how a definition drifts"). New Layer-2-specific exclusions this milestone should name (attributed, not defined):

| Excluded term | Owner | Why it is not Layer 2's | Source |
| --- | --- | --- | --- |
| permission policy content | Layer 3 (the permission-policy port) | Layer 2 owns only the ask–suspend–resume protocol | doc 0001 § 5.1, § 7 G1 |
| sandbox / confinement semantics | Layer 3 | The execution call carries an opaque policy slot Layer 2 never reads | doc 0001 § 5.1 note, § 6 seam 3 |
| tool source (built-ins, MCP) | Layer 3 (the tool-source port) | Layer 2 receives a tool set; provenance and dynamism are above it | doc 0001 § 5.1, § 6 seam 4 |
| compaction quality (summary content) | Layer 3 (injected instruction) | Layer 2 owns mechanics; the summarization instruction is injected, never authored (doc 0003 AG-18 deliverable) | doc 0003 AG-18 out-of-scope |
| permission-rule persistence across sessions | Layer 3 (session) | The policy port's remembered-resolution report is Layer 2's only obligation | doc 0003 AG-10.4, completion checklist |
| price / money | Layer 3 (the price-table port) | Layer 2 reports tokens only | doc 0001 § 5.1, § 7 G10 |
| session persistence, frontends, catalogs | Layer 3 / above | The event stream and seeded history are Layer 2's whole surface | doc 0001 § 5, doc 0003 completion checklist |

## 4. Reuse vs wrap — grounded against the shipped Layer 1 surface

AG-00.1 item 2 requires stating which Layer 1 identities Layer 2 **reuses as-is** and which it **wraps**. Grounded by reading the shipped code (not inferred):

**Reused as-is** (Layer 2 never redefines these; it cites the `V-*` row):

- **Message identity** — `backend/agent/src/ai/request.go` exports the message-identity value the pairing and history logic reduces over, together with its zero-value test. Layer 1's `V-REQ-03`.
- **Tool-call identity** — `backend/agent/src/ai/tool_call.go` defines the tool-call value carrying id, name and argument bytes, constructed through a validating constructor and read back on the result side. Layer 1's `V-REQ-16`/`V-REQ-17`.
- **Finish reasons** — `backend/agent/src/ai/finish_reason.go` defines the closed finish-reason vocabulary AG-11.1's exhaustive dispatch consumes directly. Layer 1's `V-MET-01`…`V-MET-08`.
- **Usage** — `backend/agent/src/ai/usage.go` carries the absence-vs-zero-honoring token fields AG-16.1 reports verbatim. Layer 1's `V-MET-09`…`V-MET-12`.
- **The Layer 1 stream event** (`backend/agent/src/ai/event.go`) is what the loop *drains and re-emits from* (AG-07.1 "re-emits normalized content as agent message events") — Layer 2 does not invent a second representation of L1 content, it carries it.

**Wrapped, not reused** — Layer 2's envelope is its own:

- **Events.** The Layer 1 event (`V-STR-10`) is stream-scoped (one provider response). Layer 2's agent event envelope (AG-04) is run/turn-scoped, adds explicit parent nesting (invariant 2) that has no Layer 1 analog, and carries four families (permission, cost, delegation, compaction) that do not exist at Layer 1 at all. AG-00.1 item 2's own phrasing is exact: "events — Layer 2's envelope is its own, carrying Layer 1 payloads."
- **Sequence/ordering.** L1's `V-STR-13` sequence is per-*stream*. Layer 2 needs an independent per-consumer-stream ordering at the *agent* level (AG-04.1's "ordering is per-consumer-stream and 1-based from birth" scenario) — this is a new instance of the same *pattern* C3 taught, not a reuse of the L1 counter.
- **Failure.** L1's `V-FAIL-*` taxonomy (category, retryability, partial-output discriminator) is carried *into* AG-11.2's typed turn failure, but the turn-failure type itself is Layer 2's own — it adds turn-scoped context (which turn, which call) that L1's taxonomy has no field for.

## 5. The boundary cases AG-00.1 item 1 names explicitly

All three have answers recoverable from doc 0003 as written; AG-00.1's job is to make each an explicit, citable vocabulary statement rather than an implication a reader has to reconstruct:

1. **Is a turn with zero tool calls still a turn?** **Yes — the terminal one.** AG-00's own charter states this directly ("one turn = one assistant response plus its tool results…"; the checklist item answers "yes — the terminal one"). AG-07.1's walking-skeleton scenario proves it operationally: a scripted text-only response with no tool calls still produces turn-start/message events/turn-end — a complete turn.
2. **Is a compaction summary a transcript entry or metadata about entries?** **A transcript entry** — specifically a typed one. AG-18.2's scenario is explicit: "the summary entry is typed as a compaction artifact, **distinguishable from a model message**" and it must pass AG-12's history boundary validation post-compaction (AG-18.2's other scenario: "the resulting transcript passes the history boundary validation"). It is not metadata sitting beside history; it occupies a slot in history, typed differently from an ordinary message.
3. **Is a steering message part of the current turn or the next?** **The next turn.** AG-13.2 is unambiguous: "the current turn completes untouched, and the message enters history at the turn boundary **before the next provider call**." A message queued during the *final* turn still yields a new turn rather than being dropped (AG-13.2's second scenario) — so "next turn" holds even at the edge of run termination.

**A fourth boundary case surfaced during this exploration, not named by AG-00.1's checklist, worth flagging for the decision phase**: is a *compaction call* itself "a turn"? AG-00's own definition of turn ("one assistant response plus its tool results") does not describe what a compaction call produces (a summary, not a response to the user), and doc 0003 consistently calls compaction "a model call with its own provider, cost, and cancellation" (AG-18.1) rather than "a turn" — but never states the exclusion as vocabulary. Recommend AG-00.1 add this as an explicit fifth boundary answer: a compaction call is a provider call but not a turn.

## 6. The must-nevers, restated as vocabulary-level obligations

Doc 0001 states these as prose prohibitions; AG-00.1 item 3 asks for them restated as obligations later scenarios (and AG-03.2/03.3's guards) can cite by name.

**The loop's six must-nevers** (doc 0001 § 4.1) — each already has (or will have) a mechanical guard:

| # | Must-never | Vocabulary-level obligation | Mechanical enforcement |
| --- | --- | --- | --- |
| 1 | persist anything | The loop is stateless between calls: no field, no side channel, survives across two sequential turns | AG-07.2 "two sequential turns share nothing" |
| 2 | read the filesystem or the environment | No-ambient-authority: zero env reads, zero filesystem calls, zero process spawns | AG-03.3 guard (AST scan) |
| 3 | render anything | The loop's only output is the event stream; no direct write to any display surface | AG-03.2 import guard (denies rendering-adjacent imports) + doc's stream-only contract |
| 4 | decide *whether* a tool is allowed | The loop asks (emits decision-required) and executes the answer; it never evaluates policy | AG-10.1 "an immediate answer needs no event" / policy injected, not decided |
| 5 | decide *whether* to retry | The loop reports a typed failure upward and stops; retry is harness-only | AG-11.2 "the loop never issues a second provider call" |
| 6 | know which frontend is attached | No frontend-identifying type or parameter crosses the loop's public surface | AG-03.2 import guard (Layer 2 cannot import a frontend) |

**The harness's one must-never** (doc 0001 § 4.2): **must never dictate the loop's logic** — "a different termination rule is a different harness, not a different loop." Vocabulary-level obligation: the harness may vary *when* and *how often* the loop runs (retry, steering, compaction insertion) but may never reach inside a single loop invocation to change how it decides turn completion. Mechanical enforcement: AG-13.1's "the harness holds no privileged channel into the loop… it goes through the same public one-turn surface the skeleton's external tests use."

## 7. The name fixation (AG-00.1 item 4)

- **"The portable agent runtime"** denotes the loop-plus-harness assembly (doc 0001 § 4 amendment; doc 0003 "Outcome first" section, verbatim: "Every verb above is mechanism, and that is the definition of the word *runtime* in this stack").
- **Exclusion (a) — no cognitive/biological metaphor for any Layer 2 concept.** The retired *brain* framing is named, not silently dropped, "so the reason survives the rename, not just the result" (AG-00.1 item 4's own wording). The reason: a metaphor implying cognition invites the exact design mistake §§ 4.1–4.2 spend two pages forbidding — putting policy inside Layer 2.
- **Exclusion (b) — "the runtime" never abbreviates Go's `runtime` package.** Doc 0003's own scope-boundary section already states the disambiguation rule operationally: "Nothing in `backend/agent` imports Go's `runtime`, and AG-03.2's allowlist would have to be widened deliberately for that to change, so the collision is a reading cost only. When a sentence needs the distinction, write 'the agent runtime.'" AG-00.1 should adopt this exact phrasing rule rather than re-deriving it.
- **"A Layer 3 application"** is fixed as the term for the runtime's consumer — deliberately *not* "the coding agent" or "cachicamas_coding" — so that no later milestone writes a contract, test name, or acceptance criterion in terms of a coding agent specifically (doc 0001 § 5's amendment; doc 0003 scope-boundary wording trap 4). AG-23's consumer proof is the mechanical check this definition makes possible: AG-23.1 explicitly builds "a generic Layer 3 application in miniature," never a coding-agent miniature.

## 8. Conflicts to reconcile or flag (AG-00's acceptance criterion)

Four candidates found by walking doc 0001/0002/0003/ADRs for the same noun used two ways. The first two are *already reconciled in doc 0003's own text*, so AG-00.1's job is to cite the resolution, not re-derive it; the last two are open and should be resolved (or explicitly deferred with a reason) by AG-00.1 or the design phase.

1. **Cost payload: tokens-and-money vs tokens-only — reconciled.**
   - Side A: doc 0001 § 4.3's cost-family table row and the § 2.3 turn-sequence diagram both say the harness reports "tokens, cache hits **and money**."
   - Side B: ADR 0005 § D4 (row G10) and doc 0001 § 7 (G10 disposition) both say "L2 emits, L3 prices" — token-only at Layer 2.
   - **Disposition** (already executed in doc 0003 AG-06's charter note): "The verdict wins: the Layer 2 payload is token-only, and money joins the stream as Layer 3 enrichment." AG-00.1 should record this as the vocabulary statement for "cost event," citing both sides, exactly as doc 0003 already does.

2. **"The loop executes tools" / "the harness holds state" — two wording traps, already flagged as traps rather than genuine cross-document conflicts.** Doc 0003's own Scope Boundary section states both explicitly as *plausible-but-wrong* readings ("too broad") with the corrected phrasing ("the loop schedules execution against an injected execution contract"; "the harness holds the conversation in memory… a harness that touches a file has crossed the boundary"). AG-00.1 should absorb these two traps verbatim into the vocabulary artifact, the same way AI-01.1 absorbed Layer 1's two traps into its § 2 — precedent: L1's `decision.md` § 2 quotes its traps "verbatim… because each has already caused one wrong decision."

3. **"Turn" vs "provider call" vs "attempt" — open, not yet stated as vocabulary anywhere.** doc 0001 § 2.3's sequence diagram draws exactly one loop-to-provider interaction per loop iteration, and AG-11.2 pins "the loop never issues a second provider call" (i.e., one loop invocation = exactly one provider call). But AG-15 (harness-level retry) re-invokes the loop for "a fresh provider call over an identical transcript" on the *same* turn, and AG-16.1 requires "cumulative equals the per-turn sum… **including the retried attempt's tokens**" — implying one turn can span multiple provider calls, contradicting a naive reading of AG-11.2 in isolation. AG-00's own charter already contains the resolving sentence ("one turn may span several provider calls only via retry"), but this sentence does not appear in doc 0001 itself and is not yet a named, citable vocabulary row. **Recommend AG-00.1 add "provider call" / "attempt" as first-class terms distinguishing them from "turn," citing AG-00's charter as the source, since doc 0001 conflates the two at the loop's single-invocation granularity.**

4. **"Subagent" vs "child harness" vs "nested run" / "delegated run" — open, an internal doc 0003 naming inconsistency.** The event-family and event-kind names are fixed as "subagent-started" / "subagent-ended" (AG-06.3), and AG-02's charter and doc 0001 § 6 seam 12 both say "subagent(s)." But AG-19's own scenarios consistently say "child harness" and "nested run" ("a child harness runs inside a tool," "sibling children interleave," "the child winds down first"), and AG-19's title is "delegation readiness," not "subagent readiness." No single term is declared canonical anywhere; a reader has to infer that all four phrases name the same relationship. **Recommend AG-00.1 pick one primary term (the event-kind name "subagent" is the strongest candidate, since it is the one that will actually appear on the wire/in the envelope) and record the others as synonyms used in prose**, closing this before AG-06 and AG-19 both ship using different words for the same concept.

## 9. Where the artifact lives — recommendation, with the Layer 1 precedent's reasoning

**Recommend the exact Layer 1 shape**: a live, appendable register at `openspec/specs/agent-contract-vocabulary/spec.md` (capability slug `agent-contract-vocabulary`, paralleling the existing `openspec/specs/ai-contract-vocabulary/spec.md`) as the canonical, citable text, plus an immutable `decision.md` under `openspec/changes/<archived-slug>/` recording AG-00.1's argument and the state of the register on the day it merged.

Reasoning, transplanted from the L1 register's own § "Status" section (which states this explicitly, not as inference):

1. `openspec/AGENTS.md` names `specs/` as "source of truth (main specs) — populated as changes land." A register that every later milestone must cite and append to belongs in the canonical tree, not the archive.
2. The archived `decision.md` becomes the historical record of *how* the register was first decided, immutable from that point; the live `spec.md` is the register itself, appended to in the same PR that discovers a missing term.
3. Archiving the register (leaving it only in the change folder) would freeze an artifact AG-01 … AG-23 must still write to. The Layer 1 register was appended to twice inside its own Wave 0 (`V-STR-22/23` by AI-02.1, `V-PRV-16/17/18` by AI-03.1) — direct evidence that a Wave-0-adjacent vocabulary register gets amended before the wave even closes, which only works if it lives somewhere writable by later PRs, i.e. `specs/`, not the archive.

The `openspec/specs/` directory listing confirms the naming convention is live and consistent: every Layer 1 capability spec follows `ai-<topic>`; `agent-module-scaffold` already exists (Layer 1's AI-00, the *module* scaffold — not to be confused with Layer 2's own future `agent-package-scaffold`, AG-03). `agent-contract-vocabulary` is free and matches the pattern.

**Amendment rules**: adopt the L1 register's six standing rules verbatim, substituting layer/document identifiers (append-only ids, next-free-ordinal-in-category, dated amendment blockquote stating *why* the register lacked the term, no silent edits, updated counts, amendment lands in the same PR that needed the term).

## 10. Open questions the proposal must settle, and items that belong to AG-01/AG-02 instead

**Open questions for the proposal/decision phase:**

1. Final category-code letters and identifier prefix (`V2-*` vs a distinct prefix such as `W-*`) — recommend a prefix visually disjoint from Layer 1's `V-*` (see § 3.1).
2. The canonical delegation term — "subagent" vs "child harness" (§ 8, conflict 4) — this is a pure naming question with no design alternative, so it appears to be squarely AG-00's to decide, not deferred.
3. Whether "provider call" and "attempt" become two separate rows or one row with two senses (§ 8, conflict 3).
4. Whether the fourth boundary case found here (is a compaction call a turn — § 5) should be added to AG-00.1's closing checklist as a fifth item, given it was not named by doc 0003 but is directly analogous to the three that were.
5. Whether the register should also state, once, that citations from later AG milestones must reference **both** registers where a term is reused unchanged (e.g. "message" cites `V-REQ-02`, not a re-paraphrase) — the same discipline the L1 register enforces on itself.

**Explicitly belongs to AG-01, not AG-00** (AG-00's own out-of-scope clause: "Any decision with a design alternative"):

- The stream carrier at the agent-event boundary (channels vs iterator) — AG-01.1 item 1.
- The backpressure/loss posture for agent events — AG-01.1 item 2.
- The observer decoupling *mechanism* — AG-01.1 item 3 (AG-00 only needs the *term* "observer," already reused from the L1 vocabulary's shape).
- The upward-path surface a frontend calls — AG-01.1 item 5.

**Explicitly belongs to AG-02, not AG-00:**

- Every G1/G3/G4/G5/G7/G8/G10/G11 *verdict* (implement-now / seam-with-trivial-impl / deferred) — AG-00 only needs the *nouns* (e.g. "subagent," "failover seam"), never their v1 disposition, which is AG-02.1's closing checklist.

**Explicitly belongs to AG-22, not AG-00:**

- The § D3 telemetry attribute vocabulary extension for Layer 2 spans (span names, attribute names) — this is AG-22.1's own `[decision]` node, structurally identical in shape to AG-00.1 but scoped to observability only. AG-00's charter's deliverable list does not mention telemetry terms at all; this exploration confirms that omission is correct and should stay that way.

## 11. Approaches — structural options for the register (mirrors the L1 precedent's own comparison)

| Option | Shape | Verdict |
| --- | --- | --- |
| A — glossary only, alphabetical | term → definition, no ownership column | Reject — cannot express which milestone owns a term, so two milestones can double-define one (the retired-plan failure mode) |
| B — per-milestone appendix | each AG-NN's own SDD defines its terms | Reject — this is the exact failure this milestone exists to prevent, one layer up |
| C — categorized register, stable IDs, one owner per row, six categories | mirrors the Layer 1 register's proven shape | **Recommended** — already validated twice by amendment inside Layer 1's own Wave 0; keeps ownership and provenance as first-class columns; the only change from L1's shape is the category axis (§ 3.1) and the id prefix (§ 3.1, § 9) |
| D — machine-checkable schema (YAML/JSON) + generated prose | structured data, prose rendered | Reject for v1, same reason L1 rejected it: no CI exists in this repo (ADR 0005 § Enforcement: ".github/workflows/ is absent"), so a machine check would never run; audience is humans writing charters |

## 12. Recommendation

Adopt Option C, structurally identical to the Layer 1 register, with three deltas: (1) a disjoint category scheme (§ 3.1) sized to a runtime rather than an adapter; (2) a disjoint id prefix so a citation is self-describing about which layer's register it resolves against; (3) explicit resolution of the two open naming conflicts (§ 8, items 3–4) before the register is declared closed, since — unlike Layer 1's two post-hoc amendments — both are visible now, before any downstream milestone has started, and are cheaper to fix in this PR than to amend later. The artifact lives at `openspec/specs/agent-contract-vocabulary/spec.md` (live, appendable) with `decision.md` archived under the change folder, exactly per § 9.

## 13. Risks

- **Term-count blast radius.** This exploration found on the order of 60–70 candidate terms across six categories (vs Layer 1's 114 across the same six-category count) — smaller, but AG-00.1 should budget review time accordingly; a register this size is itself a >250-line artifact and will not fit the milestone's own review-budget guidance if written carelessly.
- **The delegation-naming conflict (§ 8, item 4) is live in doc 0003's own text right now.** If AG-00.1 does not resolve "subagent" vs "child harness" explicitly, AG-06 (which ships the event-kind names) and AG-19 (which ships the structural proof) are likely to land using different vocabulary for the same relationship, which is precisely the class of defect this milestone exists to prevent.
- **The turn/provider-call/attempt distinction (§ 8, item 3) is currently stated only inside AG-00's own charter prose**, not as a citable vocabulary row anywhere. If AG-00.1 does not promote it to a first-class term, AG-11 (turn termination) and AG-15 (retry) risk re-deriving slightly different implicit definitions of "how many provider calls make a turn."
- **Category-code collision risk with Layer 1's `V-*` register is low but real** if AG-00.1 reuses the `V-` letter without a disambiguating layer marker, since both registers will sit side by side under `openspec/specs/` and doc 0003 already cross-cites Layer 1 rows directly inside Layer 2 milestone prose.

## 14. Ready for Proposal

**Yes.** Every input the proposal phase needs is grounded and cited: the term inventory (§ 3), the reuse/wrap grounding against shipped code (§ 4), the three named boundary cases plus one discovered case (§ 5), the must-nevers as citable obligations (§ 6), the name-fixation material verbatim from its two sources (§ 7), two reconciled and two open conflicts with both sides cited (§ 8), a grounded artifact-location recommendation with the precedent's own reasoning (§ 9), and an explicit list of what does *not* belong in this milestone (§ 10). The two open conflicts (§ 8, items 3–4) are the only items that need a decision before AG-00.1 can close its checklist; both are cheap to resolve because doc 0003 has not yet shipped any milestone downstream of AG-00.
