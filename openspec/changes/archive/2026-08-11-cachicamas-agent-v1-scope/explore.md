# Explore — the Layer 2 v1 capability scope

> **Change**: `cachicamas-agent-v1-scope`
> **Milestone**: AG-02 — Decide the Layer 2 v1 capability scope
> **Node**: AG-02.1 — The scope decision `[decision]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Target module**: `backend/agent/` — **no code is written by this change**
> **Sources**: [doc 0003 — Layer 2 task graph](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md) · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) § 6 and § 7 · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Precedent**: [AI-03 — the v1 capability set](../archive/2026-07-31-cachicamas-ai-minimum-capabilities/decision.md), the Layer 1 analogue, merged in PR #95
> **Predecessors**: AG-00 (`cachicamas-agent-contract-vocabulary`), AG-01 (`cachicamas-agent-event-delivery`) — both decided in parallel in this same wave
> **Authoring constraint**: doc 0003's authoring constraint binds every artifact of this change. **No type name, field name, method name, or package identifier appears anywhere.** "The loop", "the harness", "the envelope" are concept names from the architecture reference.

---

## 1. What this milestone is

> Milestone AG-02, node AG-02.1 `[decision]` in doc 0003. Zero production code — a recorded decision. This document is the complete self-contained exploration; the proposal, spec and design phases should read it whole, not a summary.

## 2. Current state

- **Entry gates satisfied.** AI-40 (Layer 1 readiness contract) merged `7326a813` on 2026-08-10; Layer 1 is 42/42. AG-00 (`cachicamas-agent-contract-vocabulary`) and AG-01 (`cachicamas-agent-event-delivery`) are being decided in parallel in this same wave and pull request; AG-02 `Depends on: AG-00, AG-01` and `Blocks: AG-17, AG-18, AG-19`.
- **AG-02's charter** (doc 0003, "Wave 0 — Decide") states eight documented defaults and one acceptance criterion: *"Every G-concern owned by L2 in v2 § 7 has a verdict here; any verdict that diverges from a documented default rebuts it explicitly."* The closing checklist (AG-02.1) has exactly two items: (1) one verdict per register row owned by L2, each stating implement-now / seam-with-trivial-impl / deferred; (2) verdicts consistent with doc 0003's own graph — implement ⇒ milestones exist; seam ⇒ a named seam-bearing node exists; any mismatch is a bug fixed before closing.
- **The authoritative register** is doc 0001 § 7, "Forward-requirements register", G1…G13, cross-referenced against ADR 0005 § D4 (which restates the same thirteen rows with slightly different "v1 verdict" wording) and against doc 0003's own R-10…R-18 and its Traceability spine.
- **Layer 1 precedent (AI-03, "the AI-03 of this layer")** shipped as `openspec/changes/archive/2026-07-31-cachicamas-ai-minimum-capabilities/`, merged in PR #95 at commit `a831c06`, promoted to `openspec/specs/ai-minimum-capabilities/spec.md`. It decided Layer 1's required / optional / excluded capability lists using the identical discipline AG-02 must replicate at the concern level.

---

## 3. The full G-concern register (v2 § 7), run through the L2-ownership test

Every row of doc 0001 § 7, with Owner, Seam number, doc 0001 disposition, ADR 0005 § D4's "v1 verdict" phrasing, whether AG-02 must verdict it, the AG-02 charter's documented default, and the doc 0003 milestone(s) that discharge it.

| G | Concern | Owner | Seam(s) | doc 0001 § 7 disposition | ADR 0005 § D4 "v1 verdict" | L2-owned? (AG-02 must verdict) | AG-02 documented default | Discharging doc 0003 node(s) |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| G1 | Permission suspendable protocol (ask/suspend/resume; allow-once / allow-always / deny / modify-input) | L2 protocol, L3 policy | 2 | seam now | "Seam now, implement in L2" | **Yes — protocol half** | implement — "it is unretrofittable" | AG-06.1 (event family), AG-10.1–10.4 (ask + suspend, four outcomes, non-blocking suspension, remembered resolutions). L3 policy half → doc 0004 CO-03 |
| G2 | Sandboxed tool execution (whole process tree) | L3 only | 3 | seam now | "Seam now, implement later" | **No** | — (out of scope: "sandbox semantics … Layer 3") | AG-09.1 carries seam 3's opaque per-call policy slot as a structural anchor only — L2 never interprets it. Full seam → doc 0004 CO-04 |
| G3 | Context compaction (protect recent turns, never orphan a pair, record removal, survive interruption) | L2 | 5, 6 | seam now | "Seam now, implement in L2" | **Yes** | implement — "the trigger seam alone is not testable against nothing" | AG-06.4 (event family), AG-17.1/17.2 (strategy seam + token accounting), AG-18.1–18.5 (the call, invariant-safe surgery, recorded on stream, interruption recovery, on-demand entry point). Note: the compaction *mechanism* is fully implemented; the shipped *default strategy* is "never compact" (AG-17.1's pin) — a seam-like triviality inside an otherwise-implement verdict, worth stating explicitly so a reader does not conflate "default strategy never compacts" with "compaction is a seam" |
| G4 | Prompt caching — L1 places breakpoints, L2 stabilises the prefix | L1 places, L2 stabilises | 1, 10 | **in v1 — AI-10 + AI-11** | "In v1, Layer 1, before any adapter" | **Yes — L2 half only** (prefix stability) | implement, AG-08.2 — "Layer 1 made breakpoints expressible; a harness that churns the prefix silently forfeits the discount" | AG-08.2 (prefix stability); AG-12.1 (history append-only) contributes; concrete marker placement → doc 0004 CO-24 |
| G5 | Parallel tool execution, call-ordered rejoin, per-tool concurrency policy | L2 | **2** *(see Finding F1 — likely a copy/paste defect; should have no seam or a distinct one)* | seam now | "Seam now, implement in L2" | **Yes** | implement | AG-05.2 (tool event family), AG-09.2 (concurrency policy: read-parallel / write-serial, bounded fan-out), AG-09.3 (ordered rejoin) |
| G6 | Dynamic / supervised tool sources (including MCP) | L3 only | 4 | seam now | "Seam now" | **No** | — (out of scope) | Not L2's; seam 4 excluded from R-18's required-in-v1 list. Full seam → doc 0004 CO-02.1 |
| G7 | Subagents (harness invoked from a tool): nested cancellation, cost, permission scope, parent-identified events | L2 | 12 | seam now | "Seam now" | **Yes — structural half** | **prove re-entrancy, ship no subagent tool** — v2 § 8 makes subagents a v1 non-goal; "the structural property is the part that cannot be added later" | AG-06.3 (delegation event family), AG-19.1–19.3 (nested run + events, nested cancellation, nested cost + permission scope). Production tool: deferred (doc 0003's own deferred-capability table) |
| G8 | Typed error taxonomy, retry with backoff, provider failover (the partial-output case is the load-bearing one) | L1 taxonomy, L2 policy | 7, 8 | in v1 — AI-19, AI-32, AI-35 *(L1 milestones only — the disposition column does not name the L2 half)* | "In v1 — already scheduled, amended" | **Yes — L2 half, and it splits into two sub-verdicts** | retry: implement in v1; failover: **seam-with-trivial-"none"-implementation**, named | retry → AG-15.1 (harness predicate), AG-15.2 (bounded backoff); failover → AG-15.3 ("Failover seam" — explicitly named node). Loop-never-retries pinned at AG-11.2 |
| G9 | Per-request options + typed-but-opaque provider escape hatch | L1 only | 9 | in v1 — AI-12 | "In v1, Layer 1, before any adapter" | **No** | — | AG-08.1 (pre-request hook) *consumes* the L1 rebuildable-request contract (AI-12) but does not own G9 |
| G10 | Cost / usage as first-class events | L2 emits, L3 prices | 11 | "**deferred to L2/L3**" *(terminology trap — see Finding F2)* | "Deferred — the Layer 1 obligation is already met" | **Yes — L2 half** | implement cost events, token-only | AG-06.2 (cost event family), AG-16.1 (per-turn + cumulative cost). L3 pricing → doc 0004 CO-05.1, CO-18.1 |
| G11 | Hook taxonomy: pre-request, pre-compact, post-turn, session-start; observers never block the stream | L2 + L3 | 1 | seam now | "Seam now" | **Yes** | taxonomy complete; pre-request and post-turn live; pre-compact live with AG-18; session-start emitted for Layer 3's use | AG-08.1 (pre-request — the one hook point that is also a named § 6 seam), AG-20.1 (remaining three hook points), AG-20.2 (observer asynchrony, envelope invariant 3 as a mechanical test) |
| G12 | Provider leakage (nine-row register in doc 0001 § 3.3) | split (L1-contract / adapter-local) | 9, 11 | in v1 — AI-07, AI-13, AI-18 | "Split" | **No** | — | Entirely Layer 1's; L2 only consumes the resulting typed values via events (for example, the reasoning round-trip token flows through AG-07.2) |
| G13 | Stream carrier: channel versus iterator at the package boundary | L1 only | — | decision only — AI-02, default keep channels | "Decision only; default = keep channels" | **No** (already decided at AI-02) | — | **Not** discharged by AG-01 despite AG-01's own charter text calling itself *"the G13 of this layer"* — see Finding F4. AG-01 makes Layer 2's own event-carrier decision (R-05 invariant 3, R-09), a distinct decision from v2 § 7's G13 row |

**Concerns AG-02 does NOT need to verdict** (Owner column excludes L2 entirely): **G2, G6, G9, G12, G13** — five of thirteen. Each is recorded above with the reason; none appears in AG-02's charter default list, and that omission is correct, not an oversight, because AG-02's own acceptance criterion is scoped to concerns "owned by L2."

**Concerns AG-02 DOES verdict, matching the charter's eight defaults exactly:** G1 (protocol half), G3, G4 (L2 half), G5, G7 (structural half), G8 (L2 half, itself splitting into retry + failover), G10 (L2 half), G11. That is eight charter-default bullets covering eight distinct G-numbers — every L2-owned row is covered, none is missing, none is extra.

---

## 4. The twelve seams of v2 § 6, and R-18's eight

R-18: *"Seams 1, 2, 3, 5, 6, 7, 8 and 12 of v2 § 6 exist in v1, each with its default named."*

| Seam # | Name | Lives on | Trivial v1 default | In R-18? | doc 0003 node (per Traceability spine, R-18 row) |
| --- | --- | --- | --- | --- | --- |
| 1 | Pre-request hook | the loop, before the provider call | identity | ✅ | AG-08.1 |
| 2 | Permission decision | tool-scheduling path in the loop | allow-all | ✅ | AG-10.1 |
| 3 | Sandbox policy | the tool execution call | none | ✅ | AG-09.1 (L2 anchor only; semantics → doc 0004) |
| 4 | Tool source | session construction | static list of built-ins | ❌ omitted | **L3-owned (G6), not an L2 v1 seam** |
| 5 | Context strategy | the harness, before each turn | never compact | ✅ | AG-17.1 |
| 6 | Token counting | optional provider capability | absent → estimate | ✅ | AG-17.2 |
| 7 | Retry classification | the typed error a provider returns | everything fatal | ✅ | AG-11.2, AG-15.1 |
| 8 | Failover policy | the harness | none | ✅ | AG-15.3 |
| 9 | Provider escape hatch | the normalized request | empty | ❌ omitted | **L1 contract item (G9), already built at AI-12** |
| 10 | Cache breakpoints | the normalized request | no markers | ❌ omitted | **L1 contract item (part of G4), built at AI-10 / AI-11** |
| 11 | Reasoning round-trip token | reasoning content | empty | ❌ omitted | **L1 contract item, built at AI-07** |
| 12 | Delegation | the harness, invoked from inside a tool | no subagents | ✅ | AG-19.1, AG-19.2, AG-19.3 |

**Why R-18 omits exactly seams 4, 9, 10 and 11**, and why this is a coherent, evidenced pattern rather than an oversight: doc 0001 § 6 itself states *"Seams 1, 9, 10 and 11 sit on Layer 1 contracts and are therefore the urgent ones … they appear in § 3.2 with milestone numbers while the rest are recorded as forward requirements below."* Seams 9, 10 and 11 are Layer 1 contract items (already shipped: AI-12, AI-10 / AI-11, AI-07 respectively) — not something an AG milestone builds, so R-18 correctly excludes them from Layer 2's v1 obligation. Seam 4 (tool source) is the one exception to that "Layer 1 urgency" grouping: it is omitted because it is **Layer 3's** (G6's owner is L3 only, per the register), not because it is a Layer 1 item — a distinct reason from the other three, worth stating explicitly in the decision rather than lumping all four together.

All eight required seams map cleanly onto exactly one doc 0003 node each (or a small named set), and this matches doc 0003's own Traceability spine row for R-18 verbatim — no mismatch found here.

---

## 5. Graph-consistency audit (AG-02.1 closing-checklist item 2)

Built mechanically: G-concern → verdict → discharging doc 0003 node, cross-checked against doc 0003's own Traceability spine ("Requirements → closing nodes" and "Nodes trace back to scope") and its "Layer 2 completion checklist."

**Result: no defect internal to doc 0003's own graph.** Every "implement" verdict AG-02's charter states has milestones below it; every "seam" verdict has a named seam-bearing node (AG-19 for G7, AG-15.3 for G8's failover half). The Traceability spine's R-10…R-18 rows agree with the mapping built independently above, node for node. No milestone in doc 0003 implements a concern that no register row covers — the foundational and architectural milestones (AG-03 through AG-05, AG-12, AG-13, AG-14, AG-21, AG-22, AG-23) all trace to R-01…R-09, R-19, R-20 and R-21 (base architecture, not the G-register's cross-cutting concerns), which is the expected shape: v2 § 7's register is specifically "eight concerns [that] have no home at all" plus five contract-level items (G4 / G8 / G9 / G12 / G13), not the whole layer.

**However, four findings surface from the source documents that the decision should record and, where in scope, rebut or reference** — because the acceptance criterion requires that *"any verdict that diverges from a documented default rebuts it explicitly,"* and two of these are exactly the kind of divergence a careless reading of doc 0001 § 7's terse disposition column would miss.

### F1 — doc 0001 § 7's G5 row cites Seam "2", which is wrong

G5 ("parallel tool execution … call-ordered re-join") is listed with Seam column value `2`, byte-identical to G1's Seam column value. But § 6's own twelve-seam catalog has no entry for parallel-tool scheduling; seam 2 is explicitly "Permission decision … the tool-scheduling path in the loop … If approval is not a suspension in the loop, it happens out of band" — a different concern entirely. This reads as a copy/paste artifact in doc 0001, not a real seam assignment.

It does **not** corrupt doc 0003: R-13 (which closes G5) traces to AG-09.2 / AG-09.3 with no seam citation at all, and R-18's seam list does not include G5 by number. Recorded as a source-document defect outside AG-02's editing scope (doc 0001 is not part of this SDD change); the decision should verdict G5 from R-13's clean mapping and note the citation defect as an evidence-quality risk, not as a graph mismatch requiring a fix inside doc 0003.

**Location verified**: `docs/architecture/0001-cachicamas-agent-stack-v2.md:699` (G5 row) against `:695` (G1 row).

### F2 — "deferred to L2/L3" in doc 0001's G10 disposition column is a terminology trap

Read literally against § 7's own stated rule (*"A row marked* seam now *reserves the place and no further work happens in v1. A row marked* in v1 *has a milestone."*), "deferred" implies no v1 milestone — yet AG-16 is a real, implemented milestone (per-turn plus cumulative cost events), and doc 0001's own "Two dispositions worth their reasoning" section immediately below the table clarifies what "deferred" actually means here: *deferred from Layer 1* (because AI-13.3's usage record already supplies everything Layer 1 owes), not deferred past v1 for Layer 2.

AG-02's charter already states the correct verdict ("implement cost events") and doc 0003's R-16 correctly traces to AG-06.2 / AG-16.1. No doc 0003 defect, but the decision should explicitly rebut a naive "deferred ⇒ no v1 work" reading of G10, citing doc 0001's own disambiguating paragraph, so a future reader of AG-02.1 does not re-litigate this.

### F3 — the same disposition-column terseness recurs for G1, G3, G5 and G11

All four read "seam now" in doc 0001 § 7, while ADR 0005 § D4's parallel table gives the fuller phrase "Seam now, implement in L2" for G1 and G3 (and the same pattern applies to G5 and G11 by the evidence of their actual milestones).

Doc 0001 § 7's own definition of "seam now" — "reserves the place and no further work happens in v1" — is contradicted by the facts on the ground for these four concerns: AG-10 (four-outcome permission protocol), AG-18 (five-leaf compaction mechanism), AG-09 (concurrency scheduler), and AG-08 plus AG-20 (all four hook points, three of them live) are all substantial v1 implementations, not placeholders.

AG-02's charter already resolves this correctly by stating "implement" as the default for all four — but it resolves it *silently*, by simply not repeating doc 0001's disposition wording. Per the acceptance criterion's own rule, this is exactly the kind of divergence-from-a-literal-reading that should be rebutted explicitly in the decision text, citing ADR 0005 § D4's fuller phrasing and doc 0003's realized milestone counts as the evidence, rather than leaving a reader to reconcile doc 0001's "seam now" against doc 0003's actual scope unaided.

### F4 (minor, informational) — AG-01's "the G13 of this layer" is an analogy

AG-01's self-description is not a claim that AG-01 discharges v2 § 7's G13 register row. G13's Owner is "L1 only" and it is already decided at AI-02 (keep channels); AG-01 makes a *distinct* decision — Layer 2's own event-carrier choice, closing R-05 invariant 3 and R-09 (the upward path) — using the same "documented default: keep channels, for symmetry with the Layer 1 carrier decision (AI-02)" pattern. AG-02.1 does not need to verdict G13 (it is not L2-owned), and should not cite AG-01 as discharging it; a footnote avoiding this confusion is worth including given the milestone's own text invites the misreading.

**No fifth finding** of a milestone implementing an uncovered concern, and no finding of a "seam" verdict lacking a named node or an "implement" verdict lacking milestones, was found. The audit is otherwise clean.

---

## 6. The Layer 1 precedent (AI-03) — verdict shape to reuse

From `openspec/changes/archive/2026-07-31-cachicamas-ai-minimum-capabilities/decision.md` and its promoted spec `openspec/specs/ai-minimum-capabilities/spec.md`:

- **Id scheme:** `CAP-R-NN` (required), `CAP-O-NN` (optional / seam-with-trivial-implementation analogue), `CAP-X-NN` (excluded / not-owned analogue) — three closed lists, each entry with a stable identifier cited by every downstream milestone.
- **"What was decided" up front:** the matrix appears in full, as tables, before any argument — "for the reader who came for one row."
- **Per-entry shape:** what it obliges / what it does **not** oblige / why. Test clauses are stated for both directions — the negative clause is explicitly called out as "not editorial," because an incomplete required entry is the expensive defect.
- **The load-bearing entry gets the argument in full, including the strongest opposing reading, before it is rebutted** (§ 6.2's token-counting entry: "the argument is given in full, including the reading that says it should be required."). AG-02's analogue is G8's retry-versus-required tension and the G1 / G3 / G5 / G11 "seam now" wording (F2 and F3 above) — both should get this full-airing-then-rebuttal treatment.
- **A cross-check section** that runs every plausible fourth-list-member candidate through the admission test and shows why nothing else qualifies (§ 8's "nine-row leakage cross-check"). AG-02's analogue is exactly this exploration's G-concern register audit above — every G1…G13 row run through the L2-ownership test, with the five non-L2 rows recorded and reasoned rather than silently dropped.
- **A discovery / mechanism section** decided once and inherited by every downstream consumer (§ 9). AG-02's analogue: how "implement" versus "seam-with-trivial" is decided (the register cited, the ADR 0005 § D4 divergence-of-wording issue rebutted) plus, per doc 0003's own "reconcile-or-flag" duty precedent (used at AG-00), any doc 0001 / ADR 0005 wording conflict is reconciled or flagged rather than silently picked.
- **"What each blocked milestone inherits" table** (§ 12) — written in each downstream milestone's own terms. AG-02's analogue: a table for AG-17, AG-18 and AG-19 (AG-02's own `Blocks:` list) stating exactly what each inherits from this decision.
- **Standing rules section** (§ 13) — durable rules the decision establishes for future amendments (for example, "a fourth entry arrives by amendment to this document, never decided locally downstream"). AI-03's own list was in fact amended once, on 2026-08-10, by AI-35 introducing `CAP-O-04`, with a dated blockquote, superseded text struck through rather than deleted, and every count updated — the exact discipline AG-02's decision should also commit itself to for any future AG-02-owned concern.
- **Closing-checklist verification table** at the end: each closing-checklist item, where it is answered, status. AG-02.1 has exactly two items, so this table has two rows plus a restated milestone-acceptance sentence.
- **Node status / gate note:** *"a `[decision]` leaf produces no production code and closes when the decision artifact answers every listed question and is merged. No `make test` gate applies."* — identical for AG-02.1.
- **Delivery precedent:** PR #95 shipped Layer 1 Wave 0 (AI-00…AI-03) as one pull request — directly supports this session's cached `single-pr`, `size:exception` delivery strategy for Wave 0 (AG-00, AG-01, AG-02) as one pull request.

---

## 7. Layer 3 orphan check (doc 0004)

Grepped doc 0004 (`docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md`) for every G-number and port name. Every concern this exploration marks "not owned by L2" is confirmed owned by a doc 0004 node:

| Concern deferred out of Layer 2 | Layer 3 owner |
| --- | --- |
| **G1** (policy half) | CO-03.1, CO-03.2 (policy port); persistence CO-16.1; UI CO-20.2 |
| **G2** (sandbox seam, implementations deferred) | CO-04.1; consulted by CO-06…CO-08; process-tree kill CO-08.2 |
| **G6** (dynamic tool sources; MCP deferred) | CO-02.1 |
| **G10** (Layer 3 half: price table, money on stream) | CO-05.1, CO-18.1 |
| **G11** (Layer 3 half: concrete hooks) | CO-24.1, CO-24.2 |
| **G4** (payoff: breakpoint placement per provider capability) | CO-24.1 (consuming AI-11 and AG-08.2) |

**No orphan found**: every concern this decision defers to Layer 3 is genuinely picked up by a named doc 0004 node, not left as an unowned gap.

---

## 8. Affected areas (files read; none modified — read-only phase)

- `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` — full document (2271 lines), especially the requirements inventory (R-01…R-21), AG-02's charter and AG-02.1's checklist, every wave 1–6 milestone (for the graph-consistency audit), the Layer 2 completion checklist, "Explicitly deferred," and the Traceability spine.
- `docs/architecture/0001-cachicamas-agent-stack-v2.md` — full document, especially § 6 (twelve seams) and § 7 (forward-requirements register), the source of every G-concern.
- `docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md`, `docs/adr/0005-promote-agent-stack-to-own-module.md` (especially § D4, the v1-scope-verdict source ADR), `docs/adr/0007-adopt-dag-convention-for-task-graphs.md` — read in full for scope and verdict authority and the graph-consistency contract.
- `docs/adr/0006-resolve-skill-and-prompt-source-of-truth.md` — grepped for G-concern overlap; none found (S7, skills and prompts, is orthogonal to AG-02).
- `openspec/changes/archive/2026-07-31-cachicamas-ai-minimum-capabilities/decision.md` and `openspec/specs/ai-minimum-capabilities/spec.md` — the AI-03 precedent, read in full for verdict shape.
- `openspec/AGENTS.md`, `openspec/config.yaml` — project conventions (RFC 2119 keywords, Given/When/Then, 400-line pull-request budget default — though this session's cached budget is 1000 lines with `size:exception` pre-accepted).
- `docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md` — grepped for every G-number and port name, confirming no orphaned deferral.
- `backend/agent/src/{ai,agenttest}/**` — confirmed to exist on disk (287+ files) via glob, corroborating doc 0003's citations to the shipped Layer 1 surface (production tree plus vendor-compatibility subtree plus the test substrate); not read in depth, since AG-02 is a pure decision node touching zero production code.

## 9. Approaches

### Approach 1 — mirror AI-03's structure exactly (recommended)

Three closed lists (implement / seam-with-trivial / not-owned), each entry with the obliges / does-not-oblige / why shape; a cross-check section (this exploration's register audit, restated); a discovery-mechanism section reconciling the doc 0001 versus ADR 0005 wording tension (F2 and F3); a "what each blocked milestone inherits" table for AG-17 / AG-18 / AG-19; standing rules for future amendment; and a closing-checklist verification table.

- **Pros:** proven form, already reviewed and merged once (PR #95); satisfies both AG-02.1 checklist items with a matching design; gives AG-17 / AG-18 / AG-19 an inheritance table they can cite instead of re-deriving scope.
- **Cons:** none material — AG-02's register is smaller (8 verdicted concerns versus AI-03's 12 capabilities), so the artifact will be shorter, which is a benefit rather than a drawback.
- **Effort:** Medium (mostly synthesis of material already gathered in this exploration; no new research needed).

### Approach 2 — minimal table-only decision

Just the verdict table (concern, verdict, discharging node) with a short paragraph per divergence.

- **Pros:** fastest to write.
- **Cons:** does not rebut the F2 and F3 divergences explicitly (violating the acceptance criterion's "any verdict that diverges from a documented default rebuts it explicitly" as literally as doc 0001's own disposition wording requires); does not give AG-17 / AG-18 / AG-19 an inheritance section; breaks precedent with the AI-01 / AI-03 established shape, which later reviewers and downstream milestones will expect.
- **Effort:** Low, but produces a weaker artifact that is more likely to be re-litigated at AG-17 / AG-18 / AG-19.

## 10. Recommendation

**Approach 1.** The G-concern register, the seam mapping, and the graph-consistency audit in this document are already assembled to the level of detail the proposal, spec and design phases need to draft `decision.md` directly, following AI-03's proven shape. The four findings (F1–F4) are the explicit-rebuttal material the acceptance criterion requires; none of them blocks the decision — they are all citation-quality and wording-clarity issues in the *source* documents (doc 0001 § 7's disposition column and its G5 seam number), not defects in doc 0003's own graph, and doc 0003's own R-ids and Traceability spine already carry the correct verdicts independent of the terse disposition wording.

## 11. Risks

| Risk | Detail |
| --- | --- |
| **F1** | doc 0001 § 7's G5 row cites Seam "2" (Permission decision), which does not correspond to any seam covering parallel tool execution in § 6's twelve-seam catalog; it looks like a copy/paste defect from G1's row. It does not corrupt doc 0003's graph (R-13 and R-18 do not depend on it) but is a citation-quality risk if the decision quotes doc 0001's seam column verbatim without correction. |
| **F2** | doc 0001 § 7's G10 disposition ("deferred to L2/L3") reads, against § 7's own stated rule, as "no v1 work," while AG-16 is in fact a real implementation milestone. The decision must state the correct reading explicitly (deferred *from Layer 1 only*) rather than leave it implicit. |
| **F3** | doc 0001 § 7's disposition column marks G1, G3, G5 and G11 all "seam now," which under § 7's own definition ("no further work happens in v1") contradicts the substantial v1 implementations doc 0003 actually schedules for all four (AG-10, AG-18, AG-09, AG-08 plus AG-20). ADR 0005 § D4's fuller phrasing ("Seam now, implement in L2") is the more accurate source and should be cited as the rebuttal evidence for each of these four verdicts. |
| **F4** | AG-01's self-description as "the G13 of this layer" is analogy, not literal discharge of v2 § 7's G13 row; worth a clarifying footnote in AG-02.1 to prevent a future reader from citing AG-01 as G13's discharging node. |
| **General** | None of F1–F4 blocks AG-02.1 from closing — the checklist's own graph-consistency test (item 2) is about doc 0003's internal graph, which is clean. F1–F4 are about the *evidentiary basis cited from doc 0001*, which the decision should address explicitly per the acceptance criterion's divergence-rebuttal clause, but a correct doc 0003-internal graph does not require doc 0001 to be edited (outside this change's scope). |

## 12. Ready for proposal

**Yes.** The G-concern register (13 rows, 8 requiring AG-02 verdicts, 5 explicitly out of scope with reasons), the R-18 seam mapping (8 required plus 4 omitted with reasons), the graph-consistency audit (clean, with four source-document findings to rebut explicitly), the AI-03 precedent shape, and the Layer 3 orphan check are all complete. The proposal phase can draft the change proposal directly from this document; the spec and design phases can draft `decision.md` following the AI-03 structural precedent with the verdict table and findings reproduced from this exploration verbatim.
