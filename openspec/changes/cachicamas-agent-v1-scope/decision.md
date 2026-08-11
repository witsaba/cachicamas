# Decision — the Layer 2 v1 capability scope

> **Change**: `cachicamas-agent-v1-scope`
> **Milestone**: AG-02 — Decide the Layer 2 v1 capability scope
> **Node**: AG-02.1 — The scope decision `[decision]`
> **Status**: decided
> **Date**: 2026-08-11
> **Project**: cachicamas (witsaba) · **Target module**: `backend/agent/` (Layer 2) — this change touches none of it
> **Closes**: doc 0003's AG-02.1 closing checklist, two items
> **Sources**: [doc 0003 — Layer 2 task graph](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md), §§ AG-02 charter, Scope boundary, Explicitly deferred, Traceability spine · [doc 0001 — agent stack v2](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) § 6, § 7 · [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d4--v1-scope-for-cross-cutting-concerns) · [doc 0004 — Layer 3 task graph](../../../docs/architecture/milestones/0004-cachicamas-coding-layer-3-task-graph.md) (the Layer 3 orphan check)
> **Binding vocabulary**: [AG-00's Layer 2 register](../cachicamas-agent-contract-vocabulary/specs/agent-contract-vocabulary/spec.md) — every Layer 2 noun below is one of its rows, cited by identifier, never redefined here
> **Binding predecessor**: [AG-01's decision](../cachicamas-agent-event-delivery/decision.md) — event delivery and the observer model, cited rather than re-decided wherever a verdict below touches delivery, the observer model, or the upward path
> **Precedent**: [AI-03's decision](../archive/2026-07-31-cachicamas-ai-minimum-capabilities/decision.md) — the Layer 1 analogue, followed in shape and depth

> [!IMPORTANT]
> **This artifact decides scope, not code.** No Go type name, field name, method name, or package identifier appears here — doc 0003's authoring constraint. Concern names use AG-00's vocabulary, cited by `VL2-*` identifier rather than paraphrased. Citations resolve to a contract document, an ADR, doc 0003, doc 0004, or the shipped Layer 1 surface — never to Layer 2 code, which does not exist.

---

## 1. How to use this document

AG-02.1 is a `[decision]` leaf. It ships no production code: per doc 0003's node grammar (defined in doc 0002 and applied here verbatim), a `[decision]` leaf produces no production code and closes when the decision artifact answers every listed question and is merged. No `make test` gate applies, and nothing under `backend/` is touched by this change.

**Two identifier namespaces, and they never overlap.** This document's own verdict identifiers — `AGS-I-NN`, `AGS-S-NN`, `AGS-D-NN`, `AGS-X-NN` — are a *different* namespace from `specs/agent-v1-scope/spec.md`'s requirement and scenario identifiers — `R-AGS-0NN`, `S-AGS-0NN`. A verdict identifier always carries a class letter (`I`/`S`/`D`/`X`) directly after `AGS`; a spec identifier always carries the `R-`/`S-` prefix and a three-digit ordinal. `AGS-I-01` is a verdict entry below; `S-AGS-044` is a scenario in the spec file. A downstream citation of `AGS-I-01` is never a citation of a requirement, and the reverse.

**Seven consumers**, each reading this artifact as data, not prose:

| If you are writing… | Read | What you get |
| --- | --- | --- |
| **AG-17** (context strategy seam, token accounting) | § 5 (`AGS-S-01`), § 11 | The shipped default context strategy is "never compact" (seam 5) and token counting falls back to an estimate (seam 6); the strategy seam's contract is settled here |
| **AG-18** (compaction) | § 5 (`AGS-I-02`, `AGS-D-03`), § 4, § 11 | The compaction mechanism is implement-now; its leaves are obligations, not options; only compaction quality is deferred |
| **AG-19** (delegation and re-entrancy) | § 5 (`AGS-I-05`, `AGS-S-02`, `AGS-D-01`), § 11 | Prove re-entrancy (implement-now); ship the delegation seam with "no subagents" (seam 12); ship no production subagent tool (deferred) |
| **AG-09** (concurrency policy, ordered rejoin) | § 5 (`AGS-I-04`), § 11 | G5's parallel-tool verdict — no seam citation, implement-now, discharged already |
| **AG-15** (retry and failover) | § 5 (`AGS-I-06`, `AGS-S-03`, `AGS-D-02`), § 11 | Retry is implement-now; failover is a seam with trivial implementation "none"; a real failover is deferred |
| **AG-16** (cost and usage events) | § 5 (`AGS-I-07`), § 11 | Cost events are implement-now, token-only; pricing is Layer 3's |
| **AG-20** (hook taxonomy) | § 5 (`AGS-I-08`), § 11 | Taxonomy complete; pre-request and post-turn live; pre-compact live with AG-18; session-start emitted for Layer 3's use |

**If you are reviewing this artifact:** § 13 walks AG-02.1's two closing-checklist items against it, item by item, with the evidence location for each. § 4 and § 9 are where a defect is most expensive — § 4's splitting-rule defenses and § 9's two audit passes are the two places a reader should look first for a hole in the argument.

**If a term below does not resolve:** it is a genuine gap in AG-00's register, corrected there by amendment under that register's own rules — never defined locally here (§ 12).

---

## 2. What was decided

Before any argument, for the reader who came for one row.

### 2.1 The total walk — all thirteen register rows, no subset

Doc 0001 § 7's forward-requirements register runs G1 through G13. Every row is walked here in register order, with a recorded outcome — omission would be visible as a missing row, not a silent gap.

| G | Concern (short) | Owner column | Layer‑2‑owned? | Outcome |
| --- | --- | --- | --- | --- |
| G1 | Permission suspendable protocol | L2 protocol, L3 policy | **Yes — protocol half** | `AGS-I-01` |
| G2 | Sandboxed tool execution | L3 only | No | `AGS-X-01` |
| G3 | Context compaction | L2 | **Yes** | `AGS-I-02`, `AGS-S-01`, `AGS-D-03` |
| G4 | Prompt caching (breakpoint placement / prefix stability) | L1 places, L2 stabilises | **Yes — L2 half** | `AGS-I-03` |
| G5 | Parallel tool execution, ordered rejoin | L2 | **Yes** | `AGS-I-04` |
| G6 | Dynamic / supervised tool sources | L3 only | No | `AGS-X-02` |
| G7 | Subagents (delegation) | L2 | **Yes — structural half** | `AGS-I-05`, `AGS-S-02`, `AGS-D-01` |
| G8 | Typed errors, retry, failover | L1 taxonomy, L2 policy | **Yes — L2 half** | `AGS-I-06`, `AGS-S-03`, `AGS-D-02` |
| G9 | Per-request options + escape hatch | L1 only | No | `AGS-X-03` |
| G10 | Cost / usage as first-class events | L2 emits, L3 prices | **Yes — L2 half** | `AGS-I-07` |
| G11 | Hook taxonomy | L2 + L3 | **Yes** | `AGS-I-08` |
| G12 | Provider leakage register | split, L1/adapter-local | No | `AGS-X-04` |
| G13 | Stream carrier | L1 only | No | `AGS-X-05` |

### 2.2 The ownership test and the resulting counts

The test applied to every row (stated as a rule in § 3): a register row is Layer‑2‑owned if and only if its Owner column assigns any part of the concern to Layer 2; the owned part is exactly the part this decision verdicts.

Applying it to the thirteen rows above yields two counts, stated so a missing row changes a number rather than passing unnoticed:

- **Eight Layer‑2‑owned rows**, named individually: **G1, G3, G4's Layer 2 half, G5, G7, G8's Layer 2 half, G10's Layer 2 half, G11**.
- **Five non‑Layer‑2 rows**, named individually: **G2, G6, G9, G12, G13**.
- **8 + 5 = 13.** Every register row is accounted for exactly once.

### 2.3 Cross-check against AG-02's charter defaults — one to one

Doc 0003's AG-02 charter states exactly eight documented defaults, quoted: "G1 protocol (documented default: implement — it is unretrofittable), G3 compaction (default: implement, AG-18 …), G4's Layer 2 half — prefix stability (default: implement, AG-08.2 …), G5 parallel tools (default: implement), G7 delegation (default: prove re-entrancy, ship no subagent tool …), G8's Layer 2 half (default: retry in v1, failover as a named seam …), G10's Layer 2 half (default: implement cost events), G11 hooks (default: taxonomy complete …)." Eight bullets, eight distinct G-numbers, corresponding one to one with § 2.1's eight owned rows: none is missing, none is extra.

### 2.4 The four closed lists

The splitting rule (§ 4) turns eight owned rows into **fourteen** verdict identifiers — three rows (G3, G7, G8) split into more than one entry, each half named. This is stated once here so the reader does not mistake "eight owned rows" for "eight identifiers" (§ 4 states why the counts differ).

**`AGS-I` — implement-now (8 entries)**

| Id | Register row | One line |
| --- | --- | --- |
| `AGS-I-01` | G1 | Permission protocol |
| `AGS-I-02` | G3 | Compaction mechanism |
| `AGS-I-03` | G4 | Prefix stability (Layer 2 half) |
| `AGS-I-04` | G5 | Parallel tool scheduling, ordered rejoin |
| `AGS-I-05` | G7 | Re-entrancy (structural half) |
| `AGS-I-06` | G8 | Retry with bounded backoff |
| `AGS-I-07` | G10 | Cost events, token-only (Layer 2 half) |
| `AGS-I-08` | G11 | Hook taxonomy |

**`AGS-S` — seam with a named trivial implementation (3 entries)**

| Id | Register row | Seam | Trivial implementation |
| --- | --- | --- | --- |
| `AGS-S-01` | G3 | 5 (+ 6) | Never compacts; token count falls back to an estimate |
| `AGS-S-02` | G7 | 12 | No subagents exist |
| `AGS-S-03` | G8 | 8 | No failover occurs |

**`AGS-D` — deferred (3 entries)**

| Id | Register row | Held by |
| --- | --- | --- |
| `AGS-D-01` | G7 | The proven re-entrancy substrate (AG-19) |
| `AGS-D-02` | G8 | The failover seam (AG-15.3) |
| `AGS-D-03` | G3 | The injected summarization instruction (AG-18.1) |

**`AGS-X` — not owned by Layer 2, a cross-check rather than a verdict (5 entries)**

| Id | Register row | Actual owner |
| --- | --- | --- |
| `AGS-X-01` | G2 | Layer 3 — doc 0004 `CO-04.1` |
| `AGS-X-02` | G6 | Layer 3 — doc 0004 `CO-02.1` |
| `AGS-X-03` | G9 | Layer 1 — `AI-12` |
| `AGS-X-04` | G12 | Layer 1 — `AI-07`/`AI-13`/`AI-18` |
| `AGS-X-05` | G13 | Layer 1 — `AI-02` |

**Total: 14 verdicts + 5 cross-checks = 19 entries**, over 13 register rows.

---

## 3. The ownership test

A register row is **Layer‑2‑owned** if and only if its Owner column (doc 0001 § 7) assigns any part of the concern to Layer 2, in whole or in part; the owned part is exactly the part this decision verdicts. The unowned part, if any, is named with its actual owner (§ 6) rather than dropped.

The test is stated as a rule so a later reader can re-apply it to a new row without re-deriving it: read the Owner column; if "L2" appears anywhere in it (alone, or paired with "L1" or "L3"), the row (or that named half) is Layer‑2‑owned and gets a verdict here; if "L2" does not appear at all, the row gets a cross-check entry (§ 6) naming its actual owner. This test prevented both silent omission (§ 2's total walk demonstrates it was applied to all thirteen rows) and scope creep into G2/G6/G9/G12/G13, none of which is verdicted here.

---

## 4. The taxonomy and the splitting rule

### 4.1 The four inclusion tests

Each class keys on a different observable, which is what keeps the classes mutually exclusive — no concern half can plausibly answer two tests at once.

| Class | Inclusion test | Observable it keys on |
| --- | --- | --- |
| `AGS-I` — implement-now | v1 ships the behavior, and a doc 0003 milestone below this decision has tests that fail if the behavior is wrong | What v1 tests exercise: the behavior itself |
| `AGS-S` — seam with trivial implementation | v1 ships only the point of insertion plus a named trivial default; replacing the default later changes no call site | What v1 tests exercise: the slot and its default, never the real behavior |
| `AGS-D` — deferred | No v1 node schedules any work for it; its place is held by an existing seam or substrate, and doc 0003's "Explicitly deferred" table cites it to AG-02 | What v1 schedules: nothing |
| `AGS-X` — not owned by Layer 2 | The register's Owner column assigns no part of the concern to Layer 2; the entry names the actual owner's node | The Owner column, not any Layer 2 artifact |

**The implement-versus-seam test**, the one that does the real work: *if the trivial default were the permanent answer, would any v1 milestone fail its acceptance criterion?* No → the concern, or that half of it, is a seam. Yes → it is an implementation, whatever doc 0001 § 7's disposition column says (§ 8, finding F3). This test is applied visibly at every one of the three splitting rows below.

### 4.2 The splitting rule

Stated once, as a rule, before the lists: **a verdict attaches to a concern half, not to a register row; a row whose halves answer § 4.1's tests differently yields one entry per half, and every half is named — never resolved by picking one.** The un-named half is the half that gets re-litigated. This is why § 2.4's fourteen verdict identifiers exceed § 2.2's eight owned rows: three rows split.

| Row | Halves | Classes | Argued in full |
| --- | --- | --- | --- |
| G8 | retry / failover seam / failover implementation | `AGS-I` / `AGS-S` / `AGS-D` | Retry is knowable from the typed error alone — implementation. Failover re-opens token budgets, the price table and the cache prefix (seam 8's own rationale, doc 0001 § 6), so the seam ships with a trivial "no failover occurs" default — a seam. The real cross-provider switch is a third, separate object that doc 0003's own "Explicitly deferred" table cites to AG-02 by name — deferred |
| G3 | compaction mechanism / shipped default strategy / compaction quality | `AGS-I` / `AGS-S` / `AGS-D` | If invariant-safe surgery, recording, and interruption recovery were absent, `AG-18.1`–`18.5` fail — that is implementation. "Never compact" forever satisfies `AG-17.1`'s pin — that is a seam. What makes a compaction summary *good* is a third, separate object from both, decided by no v1 node; doc 0003's own "Explicitly deferred" table cites it to AG-02 by name — deferred. The mechanism, its default, and its quality are three different objects, argued in § 4.3 |
| G7 | structural property / delegation seam / production tool | `AGS-I` / `AGS-S` / `AGS-D` | If re-entrancy were unproven, `AG-19.1`–`19.3` fail — implementation. "No subagents" satisfies seam 12's v1 posture — a seam. No v1 node schedules a shipped subagent tool, and doc 0003's own "Explicitly deferred" table cites it to AG-02 — deferred. Three-way because v2 § 8 makes subagents a v1 non-goal while re-entrancy is the part that cannot be added later |

**All three splitting rows are three-way for the same underlying reason.** Wherever doc 0003's own "Explicitly deferred" table separately cites a row to AG-02 (`R-AGS-006`), the seam's trivial default (`AGS-S`) and its eventual full replacement (`AGS-D`) are two different objects, not one — the same pattern that already made G7 a three-way split. § 2.4, § 4.3 and § 5 apply this consistently: G3, G7 and G8 each yield exactly three verdict identifiers, never two.

### 4.3 G3's split — the conflation this decision defeats four times

**The dangerous misreading, stated so it can be recognised.** A reader opens this decision, finds `AGS-S-01` ("shipped default context strategy: never compact, seam 5"), and reasons: *the default never compacts → compaction is a seam → a seam reserves the place and no further work happens in v1 (doc 0001 § 7's own rule for "seam now") → AG-18's five leaves are optional.* Every step is locally plausible; the chain descopes an entire milestone. The countermeasure is structural, not editorial, and it is present at four different places a reader might arrive from:

1. **At the lists (§ 2.4)**: G3 yields three entries across three different lists — the compaction **mechanism** (`AGS-I-02`, discharged by `AG-18.1`–`18.5`), the shipped **default strategy** (`AGS-S-01`, "never compact", seam 5, `AG-17.1`), and deferred **compaction quality** (`AGS-D-03`). There is no single "G3 verdict" to cite — least of all one merging the mechanism and the default strategy, which is the specific misreading this defense blocks.
2. **At the entry (§ 5)**: `AGS-S-01`'s negative clause states, in its own text, that a trivial default strategy does **not** make the mechanism optional, names the misreading, and cross-references `AGS-I-02` by identifier.
3. **At the audit (§ 9)**: the forward pass lists `AG-18.1`–`18.5` as the required evidence for `AGS-I-02`'s row — a reader checking the audit finds AG-18 as what the verdict's validity depends on, not as discretionary work.
4. **At the inheritance table (§ 11)**: AG-18's row states, in that milestone's own terms, that its leaves are not optional.

**The stated count discipline that makes this checkable.** Because G3, G7 and G8 split, `AGS-I` + `AGS-S` + `AGS-D` = 8 + 3 + 3 = 14 verdict identifiers, not 8. The artifact states this explicitly (§ 2.4) so eight owned rows are never mistaken for eight identifiers.

---

## 5. The verdict entries (closing-checklist item 1)

Every `AGS-I` and `AGS-D` entry carries six parts: identifier, register-row citation, class, what it obliges, what it does **not** oblige (normative — an entry that fails to state this is the defect a downstream milestone converts into unagreed scope), and the discharging doc 0003 node(s). `AGS-S` adds two: the trivial implementation, named as concrete behavior, and the seam number from doc 0001 § 6.

### 5.1 `AGS-I` — implement-now

#### `AGS-I-01` — Permission suspendable protocol (G1, protocol half)

- **Register row**: doc 0001 § 7 G1.
- **Obliges**: the ask–suspend–resume permission protocol (`VL2-LOOP-07`) — ask the injected policy; if it defers, emit decision-required and suspend that call without blocking any other call or event delivery; resume on the decision. All four outcomes are obliged: allow-once, allow-always (with the resolution reported remembered-eligible), deny (a typed denial result rejoins in call order), modify-input (executes with the modified arguments, both recorded). A decision addressed to an unknown or already-decided call is a typed protocol error, never a silent no-op.
- **Does not oblige**: permission *policy content* — which calls are allowed, and whether an answer should be remembered — that is `VL2-OUT-01`, owned by Layer 3's permission-policy port (doc 0004 `CO-03.1`, `CO-03.2`), named but not decided here. Nor does it oblige cross-session persistence of a remembered resolution (`VL2-OUT-05`, doc 0004 `CO-16.1`). This entry is one of finding F3's four rows (§ 8): ADR 0005 § D4's cell for G1 reads "Seam now, implement in L2," cited there as the controlling-source rebuttal of doc 0001 § 7's bare "seam now" disposition.
- **Discharging nodes**: `AG-06.1` (the permission event family), `AG-10.1` (ask and suspend), `AG-10.2` (the four outcomes), `AG-10.3` (suspension blocks nothing else), `AG-10.4` (remembered resolutions reported).

#### `AGS-I-02` — Context compaction mechanism (G3, mechanism half)

- **Register row**: doc 0001 § 7 G3.
- **Obliges**: a compaction call (`VL2-SEAM-08`) with its own provider, cost and cancellation, triggered by the context strategy seam, that replaces a compactable span (`VL2-SEAM-04`) with a summary entry (`VL2-SEAM-06`) while protecting recent turns (`VL2-SEAM-05`) and never orphaning a call/result pair (`VL2-COR-11`); recorded on the stream via the compaction event family; recoverable if interrupted, leaving the pre-compaction transcript intact and usable; invocable on demand at a turn boundary (`VL2-SEAM-07`).
- **Does not oblige**: what makes a *good* summary — compaction **quality** (`VL2-OUT-04`) is deferred separately (`AGS-D-03`); nor does it oblige that compaction ever actually runs in v1 — whether the strategy *triggers* it is the separate, seam-classed verdict `AGS-S-01`. Conflating this entry with `AGS-S-01` is the exact misreading § 4.3 defeats. This entry is one of finding F3's four rows (§ 8): ADR 0005 § D4's cell for G3 reads "Seam now, implement in L2," cited there as the controlling-source rebuttal of doc 0001 § 7's bare "seam now" disposition.
- **Discharging nodes**: `AG-06.4` (the compaction event family), `AG-18.1` (the compaction call), `AG-18.2` (invariant-safe surgery), `AG-18.3` (recorded on stream), `AG-18.4` (interruption recovery), `AG-18.5` (the on-demand entry point).

#### `AGS-I-03` — Prefix stability (G4, Layer 2 half)

- **Register row**: doc 0001 § 7 G4.
- **Obliges**: that unchanged inputs (system material, tools, hook) yield a byte-stable prefix across turns — the tool and system regions of the captured request are byte-identical turn to turn, and the message region grows strictly by append (`VL2-LOOP-02`). A silent prefix break is a silent order-of-magnitude cost regression this obligation exists to prevent.
- **Does not oblige**: cache-breakpoint *placement* — Layer 1 already made breakpoints expressible (`AI-10`, `AI-11`, shipped); concrete placement per the selected provider's caching capability is Layer 3's payoff, doc 0004 `CO-24.1` (§ 10). Nor does it oblige history's own append-only discipline, which `AG-12.1` separately provides and this obligation depends on.
- **Discharging nodes**: `AG-08.2` (prefix stability); `AG-12.1` (history append-only) contributes.

#### `AGS-I-04` — Parallel tool execution, ordered rejoin (G5)

- **Register row**: doc 0001 § 7 G5.
- **Obliges**: a concurrency policy (`VL2-LOOP-04`) — reads may run concurrently up to a documented bounded fan-out, mutating and execute-class calls serialize in call order — and an ordered rejoin (`VL2-LOOP-05`): tool results rejoin the transcript in call order regardless of completion order, because several providers reject results that do not correspond positionally to their calls.
- **Does not oblige**: any statement about which tools exist, or under what confinement they execute — the tool source (`VL2-OUT-03`) and sandbox semantics (`VL2-OUT-02`) are both Layer 3's. Nor does it reproduce doc 0001 § 7's own Seam citation for G5 — see finding F1 (§ 8): this verdict is taken from the clean discharging-node mapping below, not from doc 0001's seam cell. Separately, this entry is one of finding F3's four rows (§ 8): ADR 0005 § D4's cell for G5 reads "Seam now, implement in L2," cited there as the controlling-source rebuttal of doc 0001 § 7's bare "seam now" disposition — a distinct defect from F1's wrong seam *number*.
- **Discharging nodes**: `AG-09.2` (concurrency policy), `AG-09.3` (ordered rejoin).

#### `AGS-I-05` — Re-entrancy, the structural property (G7, structural half)

- **Register row**: doc 0001 § 7 G7.
- **Obliges**: that the harness is invocable from within a tool execution — re-entrancy (`VL2-SEAM-09`) — with a nested run, nested cancellation, nested cost aggregation, and events carrying a parent identifier (`VL2-EVT-13`); proven, not merely documented: sibling child harnesses interleave without cross-talk when run concurrently.
- **Does not oblige** — the sharpest negative clause in this artifact — **no shipped subagent tool**. v2 § 8 states plainly that subagents are a v1 non-goal ("Sandboxing, MCP, and subagents. Seams 3, 4 and 12 exist; the implementations do not."). Proving re-entrancy is the structural property that cannot be added later; a production tool built on top of it is a different, deferred concern (`AGS-D-01`). This entry is **not** one of finding F3's four rows (§ 8): ADR 0005 § D4's cell for G7 reads bare "Seam now," with no "implement in L2" suffix, so this entry does not cite that fuller phrasing. Its standing as implement-now rests on doc 0003's own AG-02 charter default ("prove re-entrancy, ship no subagent tool") and on `AG-19`'s realized milestones, not on § D4's wording.
- **Discharging nodes**: `AG-06.3` (the delegation event family), `AG-19.1` (nested run and events), `AG-19.2` (nested cancellation), `AG-19.3` (nested cost and derived permission scope).

#### `AGS-I-06` — Retry with bounded backoff (G8, retry half)

- **Register row**: doc 0001 § 7 G8.
- **Obliges**: a pre-output retryable failure retries within a documented bound as a fresh provider call over an identical transcript, each attempt visible on the stream (`VL2-HAR-06`); any failure after emitted output is surfaced, never silently retried; a terminal category never retries; the loop itself never decides whether to retry (`VL2-COR-21`) — that decision is the harness's alone, over Layer 1's typed evidence.
- **Does not oblige**: switching to a different model or provider — that is the separate, seam-classed verdict `AGS-S-03`. Nor does it oblige Layer 1's own error taxonomy, which already shipped (Layer 1's half of G8).
- **Discharging nodes**: `AG-15.1` (the harness predicate), `AG-15.2` (bounded backoff); corroborated by `AG-11.2` (the loop never retries).

#### `AGS-I-07` — Cost and usage events, token-only (G10, Layer 2 half)

- **Register row**: doc 0001 § 7 G10.
- **Obliges**: every turn emits a cost event (`VL2-COR-16` cost scope) — per-turn and cumulative token figures for input, output, cache-read, cache-write and reasoning tokens; cumulative includes retried attempts and compaction spend; absent usage on any turn yields a cost event reporting absence, never an invented zero; any figure emitted before the stream's final usage update is labelled estimate.
- **Does not oblige**: converting tokens into money — price and money (`VL2-OUT-06`) is Layer 3 enrichment, doc 0004 `CO-05.1`/`CO-18.1` (§ 10). This entry does **not** take the literal "deferred to L2/L3" reading of doc 0001's disposition column at face value — see finding F2 (§ 8).
- **Discharging nodes**: `AG-06.2` (the cost event family), `AG-16.1` (per-turn and cumulative cost); compaction spend flows through `AG-18.1`.

#### `AGS-I-08` — Hook taxonomy (G11)

- **Register row**: doc 0001 § 7 G11.
- **Obliges**: the four hook points and the discipline that governs them (`VL2-SEAM-13`) — pre-request and pre-compact may mutate, post-turn and session-start only observe; every hook point fires at its documented moment with its documented payload; hooks at one point run in registration order; a mutating hook's failure is typed and attributed. AG-02's charter default names the v1 posture precisely: taxonomy complete; pre-request and post-turn live; pre-compact live with AG-18; session-start emitted for a Layer 3 application's use.
- **Does not oblige**: any concrete hook *implementation* — cache-boundary marker placement, session enumeration content, or persistence checkpointing are Layer 3's, doc 0004 `CO-24.1`/`CO-24.2` (§ 10). This entry is one of finding F3's four rows (§ 8) — but see § 8's precision note: its rebuttal does not attribute § D4's fuller phrasing to it.
- **Discharging nodes**: `AG-08.1` (the pre-request hook, itself seam 1), `AG-20.1` (the remaining hook points), `AG-20.2` (observer asynchrony as a mechanical test).

### 5.2 `AGS-S` — seam with a named trivial implementation

#### `AGS-S-01` — Shipped default context strategy (G3, default-strategy half)

- **Register row**: doc 0001 § 7 G3.
- **Obliges**: the context strategy seam (`VL2-SEAM-01`) exists, consulted before every provider call.
- **Trivial implementation, named as behavior**: the shipped default **never compacts** — using it changes nothing observable: no compaction event, no history mutation. Token accounting (`VL2-SEAM-02`) falls back to an estimate where the optional Layer 1 capability is absent, and the estimate is labelled as such everywhere it is consumed.
- **Seam number**: 5 (context strategy), with seam 6 (token accounting) as the companion leaf of the same milestone.
- **Does not oblige**: that the compaction **mechanism** (`AGS-I-02`) is itself a seam. **This is the misreading § 4.3 names explicitly**: conflating "the shipped default strategy never compacts" with "compaction is a seam" would make AG-18's five leaves (`AGS-I-02`'s discharging nodes) look optional. They are not — `AGS-I-02` is implement-now, unconditionally, regardless of whether the strategy that triggers it ever fires in v1.
- **Discharging nodes**: `AG-17.1` (the strategy seam), `AG-17.2` (token accounting).

#### `AGS-S-02` — Delegation seam (G7, delegation-seam half)

- **Register row**: doc 0001 § 7 G7.
- **Obliges**: the delegation seam exists — the harness is invocable from within a tool execution, with derived permission scope (`VL2-SEAM-10`): what the parent's policy already allowed flows down without asking again, and what it would ask about is asked on the parent's stream.
- **Trivial implementation, named as behavior**: **no subagents exist** — every call executes at the top level; there is no delegated participant to invoke the seam against.
- **Seam number**: 12 (delegation).
- **Does not oblige**: a shipped, production subagent tool, or any delegation-depth-limit enforcement — both are `AGS-D-01`, deferred. The seam and the structural property (`AGS-I-05`) share their discharging nodes because the same proof (re-entrancy works) is what makes "no subagents by default" a meaningful, safe posture rather than an untested claim.
- **Discharging nodes**: `AG-19.1`, `AG-19.2`, `AG-19.3` (the same leaves that discharge `AGS-I-05`, since re-entrancy and the delegation seam's v1 posture are proven together).

#### `AGS-S-03` — Failover seam (G8, failover half)

- **Register row**: doc 0001 § 7 G8.
- **Obliges**: the failover seam (`VL2-HAR-07`) exists — a named injection point consulted once the harness's own retries exhaust.
- **Trivial implementation, named as behavior**: **no failover occurs** — the v1 implementation declines, and observable behavior with or without the seam installed is identical.
- **Seam number**: 8 (failover policy).
- **Does not oblige**: an actual cross-provider switch. A real implementation would re-count the token budget and restart the cache prefix — this obligation is documented on the seam now (`VL2-HAR-07`'s own definition) precisely so it need not be rediscovered later, but building it is `AGS-D-02`'s, deferred.
- **Discharging nodes**: `AG-15.3` (the failover seam).

### 5.3 `AGS-D` — deferred

Each entry below restates one of the exactly three rows doc 0003's own "Explicitly deferred" table cites to `AG-02` — verified directly: the table's "Decided by" column reads "AG-02's G7 verdict; v2 § 8", "AG-02's G8 verdict; seam 8's rationale", and "AG-02's G3 verdict" for these three rows and no others; its remaining three rows are decided by "v2 § 7 G1 disposition", "v2 § 8" directly, and "ADR 0004's layer split" — none of which is AG-02, so none of those three needs an entry here.

#### `AGS-D-01` — A production subagent tool, with delegation depth limits (G7, production-tool half)

- **Register row**: doc 0001 § 7 G7.
- **Obliges**: nothing in v1 — no v1 milestone schedules a shipped subagent tool or depth-limit enforcement.
- **Does not oblige**: revisiting `AGS-I-05`'s proof or `AGS-S-02`'s seam — both ship unconditionally; only the production tool built atop them is deferred.
- **Held by (doc 0003 "Explicitly deferred")**: the proven re-entrancy substrate, `AG-19`.

#### `AGS-D-02` — Failover implementation (G8, the real switch)

- **Register row**: doc 0001 § 7 G8.
- **Obliges**: nothing in v1 — no v1 milestone builds the cross-provider switch itself.
- **Does not oblige**: revisiting `AGS-I-06`'s retry obligation or `AGS-S-03`'s seam — both ship unconditionally; only the real failover behavior is deferred, because it re-opens token budgets and cache prefixes and needs its own design.
- **Held by (doc 0003 "Explicitly deferred")**: the failover seam, `AG-15.3` — the same node that discharges `AGS-S-03`, here cited as the placeholder rather than as a shipped behavior.

#### `AGS-D-03` — Compaction quality (G3, what makes a good summary)

- **Register row**: doc 0001 § 7 G3.
- **Obliges**: nothing in v1 — no v1 milestone decides what makes a compaction summary good; summary quality (`VL2-OUT-04`) is Layer 3-configurable and iterative.
- **Does not oblige**: revisiting `AGS-I-02`'s mechanism — the compaction machinery ships unconditionally; only the *content* of the injected summarization instruction is deferred.
- **Held by (doc 0003 "Explicitly deferred")**: the injected summarization instruction, `AG-18.1` — the same node that discharges part of `AGS-I-02`, here cited as the slot the deferred content will occupy, not as already-decided content.

---

## 6. The cross-check entries (closing-checklist item 1, completeness)

`AGS-X` entries are **not verdicts** — they exist so "eight verdicted rows" is a demonstrated count rather than a claim, and so that a register row Layer 2 does not own is recorded with its actual owner rather than silently dropped. Each carries an identifier, the register-row citation, the class label "cross-check, not a verdict," the actual owner, and the owning node instead of a discharging node.

#### `AGS-X-01` — Sandboxed tool execution (G2)

Owner: **Layer 3**, the sandbox port (`VL2-OUT-02`). Owning node: doc 0004 `CO-04.1` (the confinement vocabulary and the "none" implementation). Layer 2 carries only the opaque policy slot (`VL2-LOOP-06`) the execution call passes through unread.

#### `AGS-X-02` — Dynamic / supervised tool sources (G6)

Owner: **Layer 3**, the tool-source port (`VL2-OUT-03`). Owning node: doc 0004 `CO-02.1` (the port and the static v1 source). Not L2-owned in any part; seam 4 is correctly absent from R-18 (§ 7).

#### `AGS-X-03` — Per-request options and the provider escape hatch (G9)

Owner: **Layer 1**. Owning node: `AI-12` (per-request options and the typed-but-opaque pass-through), already shipped. `AG-08.1` (Layer 2's pre-request hook) *consumes* this contract but does not own G9.

#### `AGS-X-04` — Provider leakage register (G12)

Owner: **Layer 1**, split with adapter-local mapping. Owning nodes: `AI-07` (reasoning round-trip token), `AI-13` (refusal/pause finish reasons), `AI-18` (delta-optional tool calls) — all shipped. Layer 2 only consumes the resulting typed values via events.

#### `AGS-X-05` — Stream carrier: channel versus iterator (G13)

Owner: **Layer 1**. Owning node: `AI-02` (decision only, default keep channels), already shipped. **Footnote, consistent with finding F4 (§ 8)**: `AG-01`'s own charter text calls itself "the G13 of this layer" — that is an analogy, not a discharge. G13's Owner column is "L1 only," decided at `AI-02`; `AG-01` makes a *different* decision — Layer 2's own event-carrier choice, closing R-05 invariant 3 and R-09 — and does not verdict this register row.

---

## 7. The seam account

R-18 (verified verbatim against doc 0003 § Sources and research): *"Seams 1, 2, 3, 5, 6, 7, 8 and 12 of v2 § 6 exist in v1, each with its default named."* Doc 0001 § 6 defines exactly twelve numbered seams. Eight are required by R-18; four are omitted, for **two distinct reasons**, not one.

### 7.1 The eight required seams, mapped node-for-node against doc 0003's own R-18 spine row

Doc 0003's Traceability spine states R-18 as: *"Seam 1 → AG-08.1 · seam 2 → AG-10.1 · seam 3 → AG-09.1 (semantics → doc 0004) · seam 5 → AG-17.1 · seam 6 → AG-17.2 · seam 7 → AG-11.2, AG-15.1 · seam 8 → AG-15.3 · seam 12 → AG-19.1, AG-19.2, AG-19.3."* This decision's mapping (built independently, from § 5's entries) matches it node for node:

| Seam | Name | Bearing node(s) | Verdict entry it belongs to |
| --- | --- | --- | --- |
| 1 | Pre-request hook | `AG-08.1` | `AGS-I-08` (hook taxonomy) |
| 2 | Permission decision | `AG-10.1` | `AGS-I-01` |
| 3 | Sandbox policy | `AG-09.1` (semantics → doc 0004) | `AGS-I-04`'s policy-slot pass-through; semantics are `AGS-X-01`'s |
| 5 | Context strategy | `AG-17.1` | `AGS-S-01` |
| 6 | Token counting | `AG-17.2` | `AGS-S-01` |
| 7 | Retry classification | `AG-11.2`, `AG-15.1` | `AGS-I-06` |
| 8 | Failover policy | `AG-15.3` | `AGS-S-03` |
| 12 | Delegation | `AG-19.1`, `AG-19.2`, `AG-19.3` | `AGS-S-02` (and `AGS-I-05`) |

### 7.2 The four omitted seams — two distinct reasons

**Reason 1 — Layer 1 contract items, already shipped.** Doc 0001 § 6 states its own grouping: *"Seams 1, 9, 10 and 11 sit on Layer 1 contracts and are therefore the urgent ones… they appear in § 3.2 with milestone numbers while the rest are recorded as forward requirements below."* (Seam 1 is required and mapped above; seams 9, 10 and 11 are the ones this grouping actually excludes from Layer 2's v1 obligation.)

| Seam | Name | Shipped milestone |
| --- | --- | --- |
| 9 | Provider escape hatch | `AI-12` (`AGS-X-03`'s owning node) |
| 10 | Cache breakpoints | `AI-10` / `AI-11` (Layer 1's half of G4) |
| 11 | Reasoning round-trip token | `AI-07` (`AGS-X-04`'s owning node) |

**Reason 2 — a different reason entirely: Layer 3's, not Layer 1's.**

| Seam | Name | Owner |
| --- | --- | --- |
| 4 | Tool source | Layer 3 — G6's owner, `AGS-X-02`'s owning node, doc 0004 `CO-02.1` |

Seam 4 is the **exception** to doc 0001 § 6's own Layer‑1‑urgency grouping — it is omitted for a reason unrelated to that grouping, so a later reader does not search for it inside a Layer 2 milestone. Lumping all four omissions under one reason would hide this exception.

### 7.3 The count

**8 required + 4 omitted = 12**, the full catalog of doc 0001 § 6. No seam is unaccounted for.

---

## 8. Findings F1 – F4

AG-02's acceptance criterion requires that *"any verdict that diverges from a documented default rebuts it explicitly."* Each finding below states the opposing reading affirmatively first, with its own citation, before the disposition. F1–F4 are defects in doc 0001's **evidentiary wording**, not mismatches inside doc 0003's own graph (§ 9 distinguishes the two).

### F1 — doc 0001 § 7's G5 row cites Seam "2" — an ambiguity between two readings, not a confirmed defect

**The opposing reading, stated first — seam 2 legitimately sits on G5's own site.** Doc 0001 § 6's own catalog entry for seam 2 places its "Lives on" column at *"the tool-scheduling path in the loop"* (`docs/architecture/0001-cachicamas-agent-stack-v2.md:654`) — the exact site G5 ("Parallel tool execution with deterministic, call-ordered re-join and per-tool concurrency policy") is about. Doc 0003 corroborates the connection twice, independently of doc 0001's disputed cell. AG-09's own charter — the milestone whose `Closes:` field names G5 — reads: *"Closes: **G5** (R-13); seams 2 and 3's Layer 2 anchor: the execution call carries a policy parameter it does not interpret"* (`docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:904`). Doc 0003's own "Nodes trace back to scope" table independently states the same pairing: *"AG-09 | R-13, R-18 (seams 2–3)"* (line 2252). Two separate doc 0003 statements, in two different tables, both name AG-09 — G5's own milestone — as anchoring seam 2. Under this reading, G5's Seam column value of `2` is not miscopied from G1's row; it reflects a real, source-corroborated association between G5's concern and seam 2's site.

**The case for the defect reading.** Doc 0001 § 6's seam 2 catalog entry is titled "Permission decision," and its "Why it cannot be added later" column argues an approval/suspension rationale — *"If approval is not a suspension in the loop, it happens out of band and the event stream stops being a complete description of the session"* — with no textual reference to concurrency, ordering, or fan-out, which are G5's actual concerns. More materially, doc 0003's own R-18 Traceability-spine row — the single most granular, seam-by-seam mapping doc 0003 states, and the one this decision's own § 7.1 built independently and verified clean — assigns seam 2 to `AG-10.1` only: *"seam 2 → AG-10.1"* (line 2216), where AG-10 is the milestone that closes **G1**'s protocol half, not G5's. So even inside doc 0003 itself, two different tables answer "which node bears seam 2?" differently: the spine row says `AG-10.1` (G1's milestone) alone, while AG-09's charter and the reverse table both name AG-09 (G5's milestone) as an anchor. G1's row (line 695) and G5's row (line 699) also carry byte-identical Seam cells — confirmed by opening both lines directly — which is consistent with a copy/paste origin.

**Disposition: an ambiguity between two coherent readings, neither defeated by the other on the evidence gathered here.** "Anchor," in AG-09's charter, may mean the call-shape site where a policy parameter threads through without being interpreted — a role AG-09 already plays for seam 3 (sandbox) per its own acceptance criterion — rather than the seam's decision-bearing node, which the spine row assigns to `AG-10.1` for seam 2 specifically. On that reading, AG-09 legitimately touches seam 2 without doc 0001 § 7's G5 row being wrong to cite it, and without contradicting the spine row. The case for a defect (the catalog's permission-specific rationale, the byte-identical cell, the spine row's single `AG-10.1` assignment) and the case against it (seam 2's own "Lives on" site, and two independent doc 0003 statements naming AG-09) are both real and both source-cited; this record states both rather than picking one. Resolving which reading is correct is an architecture decision about doc 0001's seam catalog, and AG-02.1's acceptance criterion — scoped to Layer 2 verdicts — grants no authority to make it.

**`AGS-I-04`'s verdict does not depend on which reading is correct.** The Traceability spine's R-13 row (which closes G5) reads "`AG-09.2`, `AG-09.3`; ordinal from doc 0002 AI-30" and cites no seam at all; § 7.1's seam-account table above maps seam 2 to `AGS-I-01` (G1), not to `AGS-I-04` (G5). `AGS-I-04` is verdicted from that clean mapping regardless of how the ambiguity above resolves.

**The follow-up route is named, so the ambiguity is not merely noted and forgotten.** A doc 0001 amendment, in its own change, under doc 0001's own dated-blockquote amendment convention, is where this is disambiguated — not where it is presumed already answered. Three outcomes are open to whoever holds that authority, and they are different architectural claims, not equivalent rewordings of one typo: confirm seam 2 legitimately covers both G1's permission decision and G5's tool-scheduling site, and extend § 6's catalog entry to say so; leave G5's cell empty (as G13's already is, per doc 0001 § 7); or grow a new numbered seam for parallel-tool scheduling specifically. § 12's standing amendment rules add the closing half of the loop: when that amendment lands, this record receives a dated cross-reference to it.

No edit to doc 0001 ships in this change (confirmed by this change's own diff scope, § 13; verified again by `git status --short` in the verification pass).

### F2 — G10's "deferred to L2/L3" disposition is a terminology trap

**The literal reading, stated first.** Doc 0001 § 7's own stated rule for its disposition column: *"A row marked seam now reserves the place and no further work happens in v1. A row marked in v1 has a milestone."* G10's disposition reads "deferred to L2/L3" — read against that rule, literally, "deferred" implies no v1 work for either layer.

**The rebuttal, citing doc 0001's own disambiguating paragraph.** Doc 0001 § 7's "Two dispositions worth their reasoning" section, immediately below the register table, states: *"G10 needs no Layer 1 work of its own, contrary to first appearances. The usage record is required to carry the cache and reasoning token counts an adapter would report, and AI‑13.3 already owes that. What is missing is entirely above it: a per‑turn and per‑session cost event on the Layer 2 stream, and a Layer 3 price table…"* — "deferred" here means deferred **from Layer 1**, because Layer 1's usage record already supplies everything Layer 1 owes; it does not mean deferred past v1 for Layer 2.

**Disposition.** `AGS-I-07` verdicts G10's Layer 2 half implement-now, token-only, discharged by `AG-06.2` and `AG-16.1` — both real v1 milestones, not placeholders.

### F3 — the "seam now" terseness recurs for G1, G3, G5 and G11 — one rebuttal, referenced four times

**The literal reading, stated first.** Doc 0001 § 7 marks G1, G3, G5 and G11 all "seam now." Doc 0001 § 7's own definition of that phrase — "reserves the place and no further work happens in v1" — is contradicted by the facts on the ground for all four: `AG-10` (the four-outcome permission protocol), `AG-18` (the five-leaf compaction mechanism), `AG-09` (the concurrency scheduler), and `AG-08` plus `AG-20` (all four hook points, three of them live) are substantial v1 implementations, not placeholders.

**The rebuttal, citing the controlling source — verified per-citation against ADR 0005 § D4's actual table text, not the exploration's phrasing.** Doc 0001 § 7 itself names ADR 0005 § D4 as the verdict authority: *"Verdicts are decided in ADR 0005 § D4… this table adds the architectural detail behind each."* § D4's own "v1 verdict" column was read directly, row by row:

| Concern | § D4's exact "v1 verdict" text |
| --- | --- |
| G1 | "Seam now, implement in L2" |
| G3 | "Seam now, implement in L2" |
| G5 | "Seam now, implement in L2" |
| G11 | "Seam now" |

**The fuller phrasing "Seam now, implement in L2" is present in § D4's cells for G1, G3 and G5 only.** It is cited as controlling evidence for those three. Corroborating evidence, independent of § D4's wording: doc 0003's realized milestones — `AG-10` for G1, `AG-18` for G3, `AG-09` for G5 — each schedule and close substantial v1 work.

**G11's § D4 cell reads bare "Seam now," with no "implement in L2" suffix.** Its rebuttal therefore rests on a different, non-borrowed basis: AG-02's own charter default (taxonomy complete; pre-request and post-turn live; pre-compact live with AG-18; session-start emitted for Layer 3's use) plus the realized milestones `AG-08` and `AG-20`. § D4's fuller phrasing is not cited for G11 because the source does not contain it for G11 — a citation the source does not contain would satisfy `S-AGS-044` falsely.

**Disposition.** `AGS-I-01`, `AGS-I-02`, `AGS-I-04` and `AGS-I-08` are each implement-now, each referencing this rebuttal by identifier rather than restating it. `AGS-I-08`'s own entry (§ 5.1) carries the same precision note: it is one of this finding's four rows, but its evidence is the charter default plus `AG-08`/`AG-20`, not § D4's fuller phrasing.

**G7 is not one of this finding's four rows, and needs a separate caution, not this one.** G7's § D4 cell also reads bare "Seam now" (verified in the same table read above), but `AGS-I-05`'s own entry (§ 5.1) already states plainly that its standing rests on AG-02's charter default and `AG-19`'s realized milestones — it was never argued from § D4's fuller phrasing in the first place, so there is no borrowed citation to correct. The two cautions are distinct: G11 is corrected here, inside F3, because F3's rebuttal *could* be misread as covering it uniformly; G7 requires no correction inside F3 because F3 never claimed G7 at all.

### F4 — AG-01's "the G13 of this layer" is an analogy, not a discharge

**The literal reading, stated first.** `AG-01`'s own charter text in doc 0003 describes it as "The G13 of this layer, decided before any event exists" — read literally, this could suggest `AG-01` discharges register row G13.

**The rebuttal.** G13's Owner column is "L1 only," and it is already decided at `AI-02` (default: keep channels) — not L2-owned, so it is not this decision's to verdict, and it is not `AG-01`'s to discharge either. `AG-01` makes a **different** decision: Layer 2's own event-carrier choice, closing R-05 invariant 3 and R-09 (the upward path), using the same "documented default: keep channels, for symmetry with the Layer 1 carrier decision" pattern — a distinct decision that merely resembles G13's in shape.

**Disposition.** Recorded as a footnote to `AGS-X-05` (§ 6): AI-02, not AG-01, is named as G13's deciding node, consistent with § 2's total walk.

---

## 9. The graph-consistency audit (closing-checklist item 2)

Both passes below were **executed**, not asserted: their reproduction procedures are stated exactly, and the recorded outcome is derived from the tables that follow, not asserted ahead of them.

### 9.1 Forward pass — every verdict has its evidence, corroborated independently

**Procedure, as executed.** For every `AGS-I`, `AGS-S` and `AGS-D` identifier: open doc 0003's Traceability spine ("Requirements → closing nodes") at the R-row this verdict's register row maps to, and compare its listed node identifiers against this decision's own declared discharging nodes.

| Verdict | Class | Evidence kind | This decision declares | Traceability-spine row (verified, quoted) | Match |
| --- | --- | --- | --- | --- | --- |
| `AGS-I-01` | implement | ≥1 doc 0003 milestone | `AG-06.1`, `AG-10.1`–`10.4` | R-10: "AG-06.1, AG-10.1, AG-10.2, AG-10.3, AG-10.4; policy half → doc 0004" | **clean** |
| `AGS-I-02` | implement | ≥1 doc 0003 milestone | `AG-06.4`, `AG-18.1`–`18.5` | R-11: "AG-06.4, AG-17.1, AG-17.2, AG-18.1, AG-18.2, AG-18.3, AG-18.4, AG-18.5" | **clean** — R-11 is one undifferentiated row spanning both of G3's split verdicts; `AGS-I-02` ∪ `AGS-S-01`'s declared nodes equal R-11's full set exactly (8 of 8) |
| `AGS-I-03` | implement | ≥1 doc 0003 milestone | `AG-08.2` (+ `AG-12.1` contributes) | R-12: "AG-08.2; history append-only AG-12.1; concrete breakpoint placement → doc 0004 CO-24" | **clean** |
| `AGS-I-04` | implement | ≥1 doc 0003 milestone | `AG-09.2`, `AG-09.3` | R-13: "AG-09.2, AG-09.3; ordinal from doc 0002 AI-30" | **clean** |
| `AGS-I-05` | implement | ≥1 doc 0003 milestone | `AG-06.3`, `AG-19.1`–`19.3` | R-14: "AG-06.3, AG-19.1, AG-19.2, AG-19.3; production tool deferred" | **clean** — R-14's own text independently corroborates `AGS-D-01`'s existence |
| `AGS-I-06` | implement | ≥1 doc 0003 milestone | `AG-15.1`, `AG-15.2` | R-15: "AG-15.1, AG-15.2, AG-15.3; loop-never-retries AG-11.2" | **clean** — `AGS-I-06` ∪ `AGS-S-03`'s declared nodes equal R-15's full set exactly (3 of 3) |
| `AGS-I-07` | implement | ≥1 doc 0003 milestone | `AG-06.2`, `AG-16.1` | R-16: "AG-06.2, AG-16.1; compaction spend AG-18.1; pricing → doc 0004" | **clean** |
| `AGS-I-08` | implement | ≥1 doc 0003 milestone | `AG-08.1`, `AG-20.1`, `AG-20.2` | R-17: "AG-08.1, AG-20.1, AG-20.2" | **clean** |
| `AGS-S-01` | seam | named seam-bearing node | `AG-17.1`, `AG-17.2` | R-11 (shared with `AGS-I-02`, see above) | **clean** |
| `AGS-S-02` | seam | named seam-bearing node | `AG-19.1`–`19.3` | R-14 (shared with `AGS-I-05`, see above) | **clean** |
| `AGS-S-03` | seam | named seam-bearing node | `AG-15.3` | R-15 (shared with `AGS-I-06`, see above) | **clean** |
| `AGS-D-01` | deferred | row in "Explicitly deferred" | held by `AG-19` | Deferred-table row "A production subagent tool… \| The proven re-entrancy substrate (AG-19) \| AG-02's G7 verdict; v2 § 8" (verified verbatim) | **clean** |
| `AGS-D-02` | deferred | row in "Explicitly deferred" | held by `AG-15.3` | Deferred-table row "Failover to another model \| The failover seam (AG-15.3) \| AG-02's G8 verdict; seam 8's rationale" (verified verbatim) | **clean** |
| `AGS-D-03` | deferred | row in "Explicitly deferred" | held by `AG-18.1` | Deferred-table row "Compaction quality… \| The injected summarization instruction (AG-18.1) \| AG-02's G3 verdict" (verified verbatim) | **clean** |

**Result: all fourteen rows clean.** No `AGS-I` row names no milestone; no `AGS-S` row names no node; no declared mapping is uncorroborated by the spine or the deferred table. The Traceability-spine column is a second, independent source for the same mapping — this decision's own text asserts nothing the spine does not also state.

### 9.2 Reverse pass — every `Closes:`-bearing milestone, matched both ways

**Procedure, as executed.**

```
grep -n 'Closes:' docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md
```

This returned **exactly twenty lines** (real output, one per milestone charter's `SDD change:` line): `AG-00, AG-03, AG-04, AG-05, AG-06, AG-07, AG-08, AG-09, AG-10, AG-11, AG-12, AG-13, AG-15, AG-16, AG-17, AG-18, AG-19, AG-20, AG-22, AG-23`. The table below has exactly twenty rows — every returned milestone appears in the table, and every table row corresponds to a returned line.

| Milestone | `Closes:` names | Register row it matches, or base-architecture id |
| --- | --- | --- |
| `AG-00` | "R-20's start condition" | base **R-20** |
| `AG-03` | "R-01, R-02, R-03 mechanically" | base **R-01, R-02, R-03** |
| `AG-04` | "R-04 (lifecycle families), R-05" | base **R-04, R-05** |
| `AG-05` | "R-04 (the two high-volume families)" | base **R-04** |
| `AG-06` | "R-04 — the four families… G1, G10, G7, G3's visible halves" | base **R-04**, and names **G1, G10, G7, G3** — each already covered by a verdict (`AGS-I-01`, `AGS-I-07`, `AGS-I-05`, `AGS-I-02`) |
| `AG-07` | "R-06's stateless core" | base **R-06** |
| `AG-08` | "R-12; seam 1" | matches **G4** (`AGS-I-03`) and seam 1 (§ 7.1) |
| `AG-09` | "**G5** (R-13)" | matches **G5** (`AGS-I-04`) |
| `AG-10` | "**G1**'s protocol half (R-10)" | matches **G1** (`AGS-I-01`) |
| `AG-11` | "R-08's typed mid-stream path" | base **R-08** |
| `AG-12` | "R-07's boundary enforcement" | base **R-07** |
| `AG-13` | "R-08's driving loop" | base **R-08** |
| `AG-15` | "**G8**'s Layer 2 half (R-15)" | matches **G8** (`AGS-I-06`, `AGS-S-03`) |
| `AG-16` | "**G10**'s Layer 2 half (R-16)" | matches **G10** (`AGS-I-07`) |
| `AG-17` | "seams 5 and 6 (R-18)" | matches **G3**'s seam half (`AGS-S-01`) |
| `AG-18` | "**G3** (R-11)" | matches **G3** (`AGS-I-02`) |
| `AG-19` | "**G7**'s structural half (R-14)" | matches **G7** (`AGS-I-05`, `AGS-S-02`, `AGS-D-01`) |
| `AG-20` | "**G11** (R-17)" | matches **G11** (`AGS-I-08`) |
| `AG-22` | "R-19" | base **R-19** |
| `AG-23` | "R-21" | base **R-21** |

**Result: all twenty rows matched, in both directions, with no milestone closing an uncovered G-concern.** Ten of the twenty close a base-architecture identifier only — `AG-00, AG-03, AG-04, AG-05, AG-07, AG-11, AG-12, AG-13, AG-22, AG-23` (R-01…R-09, R-19…R-21 family) — the expected shape because doc 0001 § 7's register is the cross-cutting concerns with no home, not the whole layer; ten name a register row — `AG-06, AG-08, AG-09, AG-10, AG-15, AG-16, AG-17, AG-18, AG-19, AG-20` — and every one of those rows is already covered by a verdict above — none names a G-concern this decision leaves unverdicted. Both counts are read directly off the twenty-row table above, not carried over independently.

**A precision note, recorded rather than smoothed over.** Spec `S-AGS-035` lists "the foundational milestones AG-03, AG-04, AG-05, AG-12, AG-13, **AG-14**, **AG-21**, AG-22, AG-23" as milestones whose reverse-pass rows are expected to name a base-architecture identifier. The same `grep -n 'Closes:'` command, run against the actual source, does **not** return `AG-14` or `AG-21` — their charter lines read "SDD change: `cachicamas-agent-cancellation-tree` · Interrupt ≠ shutdown, and both ≠ deadline" and "SDD change: `cachicamas-agent-concurrency-hardening` · The AI-33/AI-34 of this layer, over the whole assembly" respectively, with no `Closes:` field at all — confirmed by reading both charters directly. Per this pass's own reproduction procedure (§ this section), the reverse-pass table's domain is defined exactly as "one row per doc 0003 milestone carrying a `Closes:` field," so `AG-14` and `AG-21` correctly have **no row** here. This is **not** a reverse-pass mismatch under § 9.3's disposition rule: it is not a disagreement between this decision's own two independent sources, since neither source (the forward pass's declared column, or the literal grep) ever claimed `AG-14` or `AG-21` in the first place.

**`AG-14` and `AG-21` are not the same case as `AG-01` and `AG-02`, and doc 0003's own graph carries a small, genuine asymmetry — recorded here as an observation about doc 0003, not a defect in this decision.** `AG-01` and `AG-02` also carry no `Closes:` field, but they are a different phenomenon, checked directly. `AG-01.1` appears in the **forward** "Requirements → closing nodes" table under **both** R-05 (line 2203, invariant 3) and R-09 (line 2207, "decided") — fully two-way — while `AG-02` traces to "v2 § 7's register" (line 2245), not to any R-row at all, so it has no forward-table counterpart to be missing from in either direction; that absence is expected, not asymmetric. `AG-14` and `AG-21` are neither case. Doc 0003's own "Nodes trace back to scope" table states `AG-14 → R-08` (wind-down), v2 § 4.2 (line 2257) and `AG-21 → R-05` (invariant 3 under pressure), the assembled whole (line 2264) — but neither direction agrees cleanly with the forward table. R-08's own forward row (line 2206: "AG-17.1 (context check) · AG-08.1 (pre-request hook) · AG-11.2, AG-15.1 (typed mid-stream path) · AG-10 (suspension) · AG-09.2, AG-09.3 (parallel + ordered rejoin) · AG-16.1 (cost event)") never names `AG-14` at all — `AG-14.1` appears in the forward table only under **R-09** (line 2207, "consumed"), a different requirement from the one the reverse table names, so `AG-14`'s two directions are directionally mismatched rather than simply missing. `AG-21` is a plainer case: R-05's own forward row (line 2203: "AG-04.3, AG-05.1 (invariant 1); AG-04.1, AG-19.1 (invariant 2); AG-01.1, AG-20.2 (invariant 3); AG-04.3, AG-11.2 (invariant 4)") never names it, and no `AG-21.x` appears anywhere in the forward table at all — verified directly — so `AG-21`'s link is one-way. Doc 0003's own table of contents (line 47) advertises the spine as "Requirement → node, **two-way**"; on this evidence it is one-way for `AG-21` and directionally mismatched for `AG-14`.

**Not closure-blocking, and not this decision's to repair.** `R-AGS-010`'s rule is scoped to a disagreement between this decision's own two forward-pass columns, or a mismatch in this decision's own reverse pass — neither obtains: `AG-14` and `AG-21` close no G-concern and carry no verdict here, so no verdict's evidence depends on either. The asymmetry is doc 0003's own, between two of its own tables, and belongs to a doc 0003 fix in doc 0003's own change — the same route F1 (§ 8) names for doc 0001 — not to a correction fabricated into this pass's table, which would repeat exactly the citation-fabrication error findings F1 and F3 exist to prevent.

### 9.3 The disposition rule, and the derived result

A disagreement between the forward pass's two columns, or a reverse-pass mismatch (a returned milestone missing from the table, or a table row with no matching grep line), **is** the defect closing-checklist item 2 requires fixed before this decision closes — not a note, not a risk carried forward. Neither occurred: § 9.1's fourteen rows are all clean, and § 9.2's twenty rows match the grep exactly, both ways. **Both passes are clean.** This conclusion is stated here, after the tables, because it is their result — not an opening claim the tables merely illustrate.

Findings F1–F4 (§ 8) are not audit failures under this rule: they are defects in doc 0001's evidentiary wording, not mismatches inside doc 0003's own graph, and this item's test is about doc 0003's graph.

---

## 10. The Layer 3 orphan check

Every concern this decision assigns to Layer 3 — through an `AGS-X` entry, a split row's Layer 3-adjacent half, or an `AGS-D` entry's Layer-3-owned neighbor — is checked here against doc 0004 directly, so "nothing is deferred to an owner that does not exist" is verifiable by opening doc 0004 rather than by trusting this decision. Every node below was opened and confirmed present in doc 0004.

| Concern | Layer 3 assignment | doc 0004 node(s) | Confirmed |
| --- | --- | --- | --- |
| G1's policy half | `AGS-I-01`'s "does not oblige" | `CO-03.1` (rule evaluation), `CO-03.2` (in-session memory); persistence `CO-16.1`; UI `CO-20.2` | **yes** — doc 0004's own Traceability spine independently states "G1 — permission policy half… CO-03.1, CO-03.2; persistence CO-16.1; UI CO-20.2" |
| G2 (sandbox) | `AGS-X-01` | `CO-04.1`; process-tree kill `CO-08.2` | **yes** — doc 0004 spine: "G2 — sandbox seam… CO-04.1; consulted by CO-06…CO-08; process-tree kill CO-08.2" |
| G6 (tool sources) | `AGS-X-02` | `CO-02.1` | **yes** — doc 0004 spine: "G6 — dynamic tool sources… CO-02.1" |
| G10's Layer 3 half | `AGS-I-07`'s "does not oblige" | `CO-05.1` (price table), `CO-18.1` (priced enrichment) | **yes** — doc 0004 spine: "G10 — Layer 3 half: price table, money on stream \| CO-05.1, CO-18.1" |
| G11's Layer 3 half | `AGS-I-08`'s "does not oblige" | `CO-24.1`, `CO-24.2` (concrete hooks) | **yes** — doc 0004 spine: "G11 — Layer 3 half: concrete hooks \| CO-24.1, CO-24.2" |
| G4's payoff | `AGS-I-03`'s "does not oblige" | `CO-24.1` (breakpoint placement) | **yes** — doc 0004 spine: "G4 — payoff: breakpoint placement per provider capability \| CO-24.1" |

**No orphan found.** Every Layer 3 assignment this decision makes resolves to a doc 0004 node that exists, independently corroborated by doc 0004's own Traceability spine — not merely asserted by this decision's cross-reference.

---

## 11. What each blocked milestone inherits

Three full rows for the `Blocks:` milestones, each in that milestone's own terms; four pointer rows for the charter-dependents named in AG-02's charter description of its own defaults.

| Milestone | Row kind | What it inherits |
| --- | --- | --- |
| **AG-17** | full | The shipped default context strategy is "never compact" (`AGS-S-01`, seam 5) and token counting falls back to an estimate (seam 6); the strategy seam's *contract* is settled here — its *quality* (which is not a Layer 2 concept at all) is not this milestone's concern. |
| **AG-18** | full | The compaction **mechanism** is implement-now (`AGS-I-02`); its five leaves (`AG-18.1`–`18.5`) are **obligations, not options** — this holds regardless of whether the context strategy ever triggers compaction in a given v1 run (§ 4.3's countermeasure). Compaction **quality alone** is deferred (`AGS-D-03`), attached to `AG-18.1`'s injected instruction. |
| **AG-19** | full | Prove re-entrancy (`AGS-I-05`, implement-now); ship the delegation seam with "no subagents" (`AGS-S-02`, seam 12); ship no production subagent tool (`AGS-D-01`, deferred) — three identifiers, three distinct postures, on one register row. |
| **AG-09** | pointer | `AGS-I-04` — G5's verdict (parallel scheduling, ordered rejoin, no seam citation — see finding F1). |
| **AG-15** | pointer | `AGS-I-06` (retry, implement-now) and `AGS-S-03` (failover seam, trivial implementation "none") — two of G8's three identifiers; the third, `AGS-D-02` (the real failover switch), is held at `AG-15.3` as a placeholder rather than an active obligation this pointer carries. |
| **AG-16** | pointer | `AGS-I-07` — G10's identifier (cost events implement-now, token-only; pricing is Layer 3's, `CO-05.1`/`CO-18.1`). |
| **AG-20** | pointer | `AGS-I-08` — G11's identifier (taxonomy complete; observers never block; concrete hooks are Layer 3's, `CO-24.1`/`CO-24.2`). |

Every milestone that cites AG-02 in its own charter — the three `Blocks:` milestones plus the four charter-dependents above — has at least one governing identifier named here.

---

## 12. Standing amendment rules

Two revision routes, kept distinct.

**A new Layer‑2‑owned concern** arrives by **dated amendment to this document**, never by a local verdict inside a downstream milestone. Following AI-03 § 13's discipline — itself exercised once, 2026-08-10, appending `CAP-O-04` with a dated blockquote, superseded text struck through rather than deleted, every count updated. That exercised amendment lives in the **promoted** spec, not in the archived decision this document's header cites for AI-03's shape and depth: `openspec/specs/ai-minimum-capabilities/spec.md:23`'s dated blockquote, `:84`'s `CAP-O-04` row, and `:604`'s `~~all three~~ **all four**` struck-through count — the archived `decision.md` is immutable at its 2026-07-31 merge and predates the 2026-08-10 amendment it would otherwise need to carry. The same discipline recurs here:

- Identifiers are **append-only**: a new entry takes the next free ordinal in its class (`AGS-I`, `AGS-S`, `AGS-D` or `AGS-X`); existing identifiers are never renumbered, reused, or reordered.
- A superseded entry keeps its identifier, with its old text struck through and visible, never deleted.
- Every stated count in § 2 (the thirteen-row walk, the eight/five split, the fourteen/five verdict-and-cross-check totals) and § 7 (the twelve-seam total) is updated in the same amendment.
- The amendment is introduced by a dated blockquote stating what was appended, by which node, and why this decision lacked the concern.

**An existing verdict disproven by implementation** follows doc 0003's living-graph clause (which adopts doc 0002's revert-and-record rule verbatim): revert to green, record the discovery as graph structure, land the amendment in the resuming pull request. This document adds one binding rule of its own: the verdict revision and the doc 0003 graph change it implies travel in the **same** pull request, and the affected forward-pass and reverse-pass rows (§ 9) are re-derived in that same amendment — so the audit never silently goes stale against a revised verdict.

**F1's specific follow-up** (§ 8) is a third, narrower case: a defect in doc 0001's own text, repaired in doc 0001's own change, under doc 0001's own dated-blockquote convention. When that lands, F1's record above receives a dated cross-reference to it; F1's record is the tracking mechanism, not a closed matter.

---

## 13. Closing-checklist verification

AG-02.1's two closing-checklist items, re-walked against the merged artifact.

| # | Closing-checklist item | Answered in | Evidence |
| --- | --- | --- | --- |
| 1 | One verdict per register row owned by L2, each citing the register and stating implement-now / seam-with-trivial-impl / deferred, with the trivial implementation named where applicable | § 5, with § 2 (the total walk and counts), § 4 (the splitting rule and G3's four defenses), § 6 (the five cross-checks completing the thirteen-row total) | Every Layer-2-owned row (G1, G3, G4-L2, G5, G7, G8-L2, G10-L2, G11) has at least one verdict, each citing `doc 0001 § 7 G-NN`, classed `AGS-I`/`AGS-S`/`AGS-D`, with every `AGS-S` entry naming its trivial implementation as concrete behavior and its seam number (§ 5.2). **Satisfied.** |
| 2 | The verdicts are consistent with doc 0003's own graph: implement ⇒ milestones exist; seam ⇒ a named seam-bearing node exists; a mismatch is fixed before closing | § 9, both passes, executed and recorded | § 9.1's forward pass: fourteen of fourteen verdict rows clean against the Traceability spine, independently corroborated. § 9.2's reverse pass: twenty of twenty `Closes:`-bearing milestones matched both ways against the literal grep, with one precision note recorded (not smoothed over) rather than fabricated into a false match. § 9.3: no disagreement found; both passes derived clean, after the tables. **Satisfied.** |

**Milestone acceptance, restated from doc 0003 and checked**: *"Every G-concern owned by L2 in v2 § 7 has a verdict here; any verdict that diverges from a documented default rebuts it explicitly."* The first clause is § 2's total walk plus § 5's fourteen entries. The second is § 8's four findings, each with the opposing reading stated first and the controlling source cited, verified per-citation against the actual source text rather than inherited from `explore.md`'s phrasing.

**Node status.** AG-02.1 closes on merge of this artifact. Per doc 0003's node grammar, a `[decision]` leaf produces no production code and closes when the decision artifact answers every listed question and is merged. No `make test` gate applies; nothing under `backend/` is touched by this change.

**Wave 0 closes here.** With AG-00 (vocabulary), AG-01 (event delivery) and AG-02 (this decision, v1 scope) all merged in the same pull request, waves 1 through 6 can build without any of the three being reopened. AG-17, AG-18 and AG-19 (wave 4) and the four charter-dependents (AG-09, AG-15, AG-16, AG-20) cite this artifact's verdict identifiers from their own SDD changes onward, per § 11.
