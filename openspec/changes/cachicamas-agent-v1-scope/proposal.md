# Proposal — the Layer 2 v1 capability scope

> **Change**: `cachicamas-agent-v1-scope`
> **Milestone**: AG-02 — Decide the Layer 2 v1 capability scope
> **Node**: AG-02.1 — The scope decision `[decision]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Driver**: braejan
> **Scope**: documentation only — `openspec/changes/cachicamas-agent-v1-scope/`. **Zero Go. Zero `go.mod` change. Zero files under `backend/`. Zero edits to any merged document.**
> **Predecessor artifact**: [`explore.md`](./explore.md) (this change)
> **Depends on**: AG-00 (`cachicamas-agent-contract-vocabulary`), AG-01 (`cachicamas-agent-event-delivery`) — both decided in parallel in this same wave and pull request
> **Blocks**: AG-17, AG-18, AG-19
> **Precedent**: [AI-03](../archive/2026-07-31-cachicamas-ai-minimum-capabilities/proposal.md) — the Layer 1 analogue, merged in PR #95
> **Authoring constraint**: doc 0003's authoring constraint binds every artifact of this change. No type name, field name, method name, or package identifier appears anywhere.

---

## Intent

Close doc 0003's AG-02.1 closing checklist — one verdict per Layer-2-owned row of the forward-requirements register, and a graph-consistency audit of those verdicts against doc 0003's own graph — in one merged artifact, so that **AG-17, AG-18 and AG-19 can each read their scope off a table instead of re-deriving it from doc 0001 § 7**, and so that no wave-4 or wave-5 milestone re-litigates whether a concern is in v1.

AG-02 is the last node of wave 0. After it, the vocabulary is fixed (AG-00), the event-delivery model is fixed (AG-01), and the boundary of v1 is fixed (AG-02). Waves 1 through 6 can then be built without any of the three being reopened.

The milestone belongs in wave 0 rather than beside the milestones it scopes, for the reason worth stating: **a scope verdict changed after the milestone below it exists invalidates that milestone's own acceptance criterion.** Promoting a seam to an implementation after AG-15 ships re-opens token budgets and cache prefixes; demoting an implementation to a seam after AG-18 ships strands five leaves. Both are free today and expensive from wave 4 onward.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can separate the inherited from the decided.

1. **The register is doc 0001 § 7's**, G1…G13. AG-02 assigns verdicts to rows; it does not add, remove, or re-own a row.
2. **The verdict authority is [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md)**, which doc 0001 § 7 itself names ("Verdicts are decided in ADR 0005 § D4; this table adds the architectural detail behind each"). Where the two are worded differently, ADR 0005 is the verdict and doc 0001 is the detail.
3. **Eight documented defaults are already stated in AG-02's charter** — G1, G3, G4's Layer 2 half, G5, G7, G8's Layer 2 half, G10's Layer 2 half, G11. This proposal adopts all eight; the work is the argument, the identifiers, and the audit, not a different answer.
4. **Subagents are a v1 non-goal** (v2 § 8). G7's verdict is the structural property only.
5. **Layer 3 owns sandbox semantics, permission policy content, and pricing.** Naming Layer 3 as the owner is in scope; deciding how it works is not.
6. **No production code, no type names.** doc 0003's authoring constraint and its `[decision]` node grammar: *a `[decision]` leaf produces no production code and closes when the decision artifact answers every listed question and is merged. No `make test` gate applies.*

## Scope

### In scope — one pull request (shared with AG-00 and AG-01)

| Artifact | Content |
| --- | --- |
| `explore.md` | The exploration, transcribed into the repository: the thirteen-row G-concern table, the twelve-seam mapping with R-18's eight, the graph-consistency audit, findings F1–F4, the AI-03 precedent shape, the Layer 3 orphan check |
| `proposal.md` | This file |
| `specs/agent-v1-scope/spec.md` | `R-AGS-0NN` — each a checkable property of the decision artifact |
| `design.md` | The structure `decision.md` implements and the reasoning rules it applies |
| `tasks.md` | The single leaf AG-02.1: one task per closing-checklist item, plus the audit pass and the verification pass |
| `decision.md` | **The deliverable.** The verdict lists, the four rebuttals, the two-pass audit, and what each blocked milestone inherits |

### Out of scope (explicit, and deferred-but-related is called out)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Sandbox semantics; what a policy means | **Layer 3** — doc 0004 CO-04.1 | AG-02's charter excludes it by name. AG-09.1 carries the opaque slot; Layer 2 never interprets it |
| Permission **policy content** (which tool is allowed, remembered where) | **Layer 3** — doc 0004 CO-03.1, CO-03.2, CO-16.1 | AG-02 verdicts G1's *protocol* half only |
| Pricing — converting tokens into money | **Layer 3** — doc 0004 CO-05.1, CO-18.1 | AG-02 verdicts G10's *emission* half only; the cost events are token-only |
| Dynamic and supervised tool sources, including MCP | **Layer 3** — doc 0004 CO-02.1 | G6's owner is L3 only; seam 4 is correctly absent from R-18 |
| The Layer 1 halves of G4, G8, G9 and G12 | **Layer 1** — shipped (AI-10/AI-11, AI-19/AI-32/AI-35, AI-12, AI-07/AI-13/AI-18) | Already built; AG-02 consumes them |
| G13, the stream carrier | **Layer 1** — decided at AI-02 | Not L2-owned. AG-01 makes a *different* carrier decision — see F4 |
| **A production subagent tool** and delegation depth limits | The proven re-entrancy substrate (AG-19) | Deferred by G7's verdict; already a row in doc 0003's "Explicitly deferred" table |
| **Failover implementation** | The failover seam (AG-15.3) | Deferred by G8's verdict; re-opens token budgets and cache prefixes, so it needs its own design |
| **Compaction quality** — what makes a good summary | The injected summarization instruction (AG-18.1) | Deferred by G3's verdict; prompt engineering is iterative and Layer 3-configurable |
| Any edit to doc 0001, doc 0003, doc 0004, or an ADR | — | See "The doc 0001 question" below. Zero merged documents are touched by this change |
| Any code, test, `go.mod` entry, or file under `backend/` | Waves 1–6 | AG-02.1 is a `[decision]` leaf |

## Capabilities

### New capabilities

- `agent-v1-scope`: the Layer 2 v1 scope verdicts — one per Layer-2-owned register row — the four source-document rebuttals, and the two-pass graph-consistency audit that makes the verdicts re-derivable rather than asserted.

### Modified capabilities

- None. No existing spec's requirements change. `agent-module-scaffold` and the `ai-*` specs are read, not amended.

## The decisions this proposal commits to

One line each, so a reviewer can accept or reject the substance before reading the argument.

### 1. Adopt AI-03's proven verdict shape, and say so

`decision.md` follows the structure AI-03 shipped and PR #95 reviewed: closed lists with stable identifiers up front, then the argument; per-entry **obliges / does not oblige / why**; a cross-check section; a mechanism section; a "what each blocked milestone inherits" table; standing amendment rules; a closing-checklist verification table. The negative clause is not editorial — an entry that fails to name what it does *not* oblige is the defect that gets re-litigated at AG-18.

### 2. Four closed lists with stable identifiers

The checklist names three verdict values. A fourth list is the not-a-verdict cross-check that keeps the five non-L2 rows visible rather than silently dropped.

| Prefix | Meaning | Entries |
| --- | --- | --- |
| `AGS-I-NN` | **implement-now** | G1 protocol half · G3 compaction mechanism · G4's Layer 2 half (prefix stability) · G5 parallel tools · G7's structural half (re-entrancy) · G8's retry half · G10's Layer 2 half (cost events, token-only) · G11 hook taxonomy |
| `AGS-S-NN` | **seam with a named trivial implementation** | G8's failover half — trivial implementation *none*, seam 8, node AG-15.3 · G3's shipped default context strategy — *never compact*, seam 5, node AG-17.1 · G7's delegation seam — *no subagents*, seam 12, nodes AG-19.1–19.3 |
| `AGS-D-NN` | **deferred** | The production subagent tool · the failover implementation · compaction quality — the three rows doc 0003's own "Explicitly deferred" table already cites to AG-02 |
| `AGS-X-NN` | **not owned by Layer 2** (cross-check, not a verdict) | G2 → doc 0004 CO-04.1 · G6 → CO-02.1 · G9 → Layer 1 AI-12 · G12 → Layer 1 AI-07/AI-13/AI-18 · G13 → Layer 1 AI-02 |

**The splitting rule, stated so eight rows are not mistaken for eight identifiers.** Every Layer-2-owned row carries at least one verdict. Three rows split, and each split is named on both halves rather than resolved by picking one: **G8** (retry `AGS-I` / failover `AGS-S`), **G3** (mechanism `AGS-I` / shipped default strategy `AGS-S`), **G7** (structural property `AGS-I` / delegation seam `AGS-S` / production tool `AGS-D`). G3's split is the one most likely to be misread — the compaction *mechanism* is fully implemented; only the *default strategy* is trivial, and a reader who conflates the two concludes that AG-18's five leaves are optional.

### 3. Every entry cites its register row and names its discharging node

Per entry: the register row (`doc 0001 § 7 G-NN`), what the verdict obliges, what it does **not** oblige, the seam number and its trivial implementation where the verdict is `AGS-S`, and the doc 0003 milestone or node that discharges it. An `AGS-X` entry names its owner and the doc 0004 or Layer 1 node instead.

### 4. R-18's eight seams, plus the four omissions with distinct reasons

Seams 1, 2, 3, 5, 6, 7, 8 and 12 are each mapped to the doc 0003 node that bears it (AG-08.1 · AG-10.1 · AG-09.1 · AG-17.1 · AG-17.2 · AG-11.2 with AG-15.1 · AG-15.3 · AG-19.1–19.3). The four omissions are recorded with **two different reasons, not one**: seams 9, 10 and 11 are Layer 1 contract items already shipped (AI-12, AI-10/AI-11, AI-07), which is doc 0001 § 6's own stated grouping; **seam 4 is omitted for a different reason — it is Layer 3's** (G6's owner), not a Layer 1 item. Lumping all four together is the misreading this entry prevents.

### 5. Four rebuttals, each citing both sides

AG-02's acceptance criterion requires that any verdict diverging from a documented default rebuts it explicitly. Two of these divergences are invisible unless stated, because the charter resolves them *silently* by not repeating doc 0001's wording.

| Finding | Both sides | Proposed disposition |
| --- | --- | --- |
| **F1** — doc 0001 § 7's G5 row cites Seam `2`, byte-identical to G1's, while § 6 has no seam covering parallel-tool scheduling (verified at `docs/architecture/0001-cachicamas-agent-stack-v2.md:699` against `:695`) | *For the citation*: it is in a merged architecture reference. *Against*: seam 2 is defined as "Permission decision … the tool-scheduling path in the loop", a different concern; § 6's catalog has no parallel-tools entry | Treat as a probable copy/paste defect in doc 0001. Verdict G5 from R-13's clean mapping (AG-09.2, AG-09.3), which cites no seam at all; **do not reproduce doc 0001's seam cell for G5**. Record the defect and its follow-up route; ship no edit — see "The doc 0001 question" |
| **F2** — G10's disposition reads "deferred to L2/L3", which under § 7's own rule ("a row marked *seam now* reserves the place and no further work happens in v1") implies no v1 work | *For "no v1 work"*: § 7's stated rule, read literally. *Against*: AG-16 is a real milestone, and doc 0001's own "Two dispositions worth their reasoning" paragraph immediately below the table says the deferral is *from Layer 1*, because AI-13.3's usage record already supplies everything Layer 1 owes | Rebut the literal reading, citing doc 0001's own disambiguating paragraph. `AGS-I` for G10's Layer 2 half, token-only |
| **F3** — G1, G3, G5 and G11 all read "seam now", contradicting § 7's own definition given AG-10, AG-18, AG-09 and AG-08 + AG-20 | *For "seam now"*: doc 0001 § 7's disposition column. *Against*: [ADR 0005 § D4](../../../docs/adr/0005-promote-agent-stack-to-own-module.md) — the document § 7 itself names as the verdict authority — says "**Seam now, implement in L2**", and doc 0003 schedules substantial v1 work for all four | Cite ADR 0005 § D4's fuller phrasing as the accurate source, with doc 0003's realized milestone counts as corroborating evidence. `AGS-I` for all four, rebutted once and referenced from each entry |
| **F4** — AG-01 calls itself "the G13 of this layer" | *For*: the milestone's own charter text. *Against*: G13's owner is L1 only and it was decided at AI-02; AG-01 decides Layer 2's own event carrier, closing R-05 invariant 3 and R-09 | Record as an analogy, not a discharge. `AGS-X` for G13 names AI-02, **not** AG-01. A footnote, because the milestone's own wording invites the misreading |

### 6. The graph-consistency audit is a re-derivable procedure, not a claim

Closing-checklist item 2 is an acceptance criterion the decision must **pass**, not an assertion it makes. The exploration reports doc 0003's internal graph is clean; the decision's job is to record *how a reviewer re-derives that* without trusting either document. Two passes, both mechanically reproducible:

| Pass | Row | Columns | What a defect looks like |
| --- | --- | --- | --- |
| **Forward** — every verdict has its evidence | one per `AGS-I` / `AGS-S` / `AGS-D` identifier | verdict class · the evidence kind its class requires (`AGS-I` ⇒ at least one doc 0003 milestone below it; `AGS-S` ⇒ a named seam-bearing node; `AGS-D` ⇒ a row in doc 0003's "Explicitly deferred" table) · the node ids · **the doc 0003 Traceability-spine R-row that states the same mapping independently** · status | An `AGS-I` with no milestone; an `AGS-S` with no named node; a mapping the Traceability spine does not corroborate |
| **Reverse** — every milestone has its register row | one per doc 0003 milestone carrying a `Closes:` field | the milestone · the G-concern or R-id its `Closes:` field names · the register row it matches, **or** the base-architecture R-id (R-01…R-09, R-19…R-21) that explains why it closes no G-concern | A milestone closing a G-concern no register row covers |

The reverse pass is the one that makes the audit checkable: a reviewer greps doc 0003 for `Closes:` and compares the result to the table, rather than taking the decision's word for it. The forward pass's Traceability-spine column is a second, independent source for the same mapping — a disagreement between the two columns is the defect the checklist asks to be fixed before closing.

**Expected result, to be re-derived rather than assumed:** both passes clean. The foundational milestones (AG-03…AG-05, AG-12, AG-13, AG-14, AG-21, AG-22, AG-23) close base-architecture R-ids, not G-concerns, which is the expected shape — v2 § 7's register is the cross-cutting concerns with no home, not the whole layer.

### 7. The Layer 3 orphan check is an acceptance criterion

Every `AGS-X` entry, and every `AGS-D` entry whose owner is Layer 3, names a doc 0004 node. The exploration reports no orphan (G1 policy → CO-03.1/CO-03.2/CO-16.1/CO-20.2 · G2 → CO-04.1/CO-08.2 · G6 → CO-02.1 · G10 pricing → CO-05.1/CO-18.1 · G11 concrete hooks → CO-24.1/CO-24.2 · G4 payoff → CO-24.1). The decision records the check as a table with the node ids, so **"nothing is deferred to an owner that does not exist"** is verifiable by opening doc 0004, not by trusting this proposal.

### 8. Standing amendment rules

Following AI-03 § 13, which was itself amended once (2026-08-10, AI-35 appending `CAP-O-04`, dated blockquote, superseded text struck through rather than deleted, every count updated): a later Layer-2-owned concern arrives by a dated amendment to `decision.md`, never by a local verdict inside a downstream milestone. Identifiers are append-only; no existing entry is renumbered.

## The doc 0001 question — recommendation: record, do not amend

F1–F4 are defects in a merged document, so the proposal must state whether a correction ships here. **Recommendation: rebut all four inside `decision.md`; ship zero edits to doc 0001 in this pull request.** The argument, since this should not be assumed:

1. **F2, F3 and F4 need no edit at all.** They are terseness and analogy, not error. The acceptance criterion asks for an explicit rebuttal in the decision — which is exactly what the decision's own text is for. Editing doc 0001 would satisfy a requirement nobody stated.
2. **F1 is a real error, but its repair is architectural rather than typographical.** Two repairs are available and they are not equivalent: G5's seam cell becomes empty (as G13's already is), or § 6 grows a thirteenth seam for parallel-tool scheduling. The second changes a catalog that R-18, doc 0004 and ADR 0005 all cite **by number**. Choosing between them is an architecture decision; AG-02.1's acceptance criterion is scoped to Layer 2 verdicts and gives it no authority to make one.
3. **The AI-03 precedent does not transfer.** AI-03 amended AI-01's register in the same pull request under **AI-01 § 9 rule 2 — an explicit amendment route addressed to downstream milestones** — append-only, and AI-03 literally could not proceed without the three appended terms. Doc 0001 § 7 offers AG-02 no such route, and AG-02 proceeds fine: F1 changes no verdict, because R-13 cites no seam.
4. **Doc 0001 is shared by this wave.** AG-00, AG-01 and AG-02 land in one pull request and all three read it. An in-place edit inside one change's artifact set is a coordination hazard for no gain here.

**What ships instead:** `decision.md` records F1 as a named source-document defect with its evidence (both line references), states that G5's verdict does not depend on it, and names the follow-up route — a doc 0001 amendment under doc 0001's own dated-blockquote convention, in its own change, where the empty-cell-versus-thirteenth-seam choice can be argued. The finding is recorded, not lost, and the blast radius of this pull request stays at one new directory.

## Approach

1. **Verdict from the authority, not from the terse column.** ADR 0005 § D4 is the verdict; doc 0001 § 7 is the architectural detail. Every entry cites the register row and, where the two are worded differently, says so and names which is controlling. This is F3's rebuttal applied as a method rather than a one-off.
2. **State the negative clause on every entry.** An `AGS-I` that does not say what it fails to oblige is how AG-18 acquires a leaf nobody agreed to. G7's entry is the sharp case: implementing the structural property obliges a proven re-entrant harness and obliges **no shipped subagent tool**.
3. **Air the strongest opposing reading before rebutting it.** AI-03 did this for token counting. AG-02's analogues are F3 (a literal reading of § 7 makes four implementations into placeholders) and G8's retry-versus-failover line (why one half is implemented and the other is a seam, when both are in the same register row).
4. **Run all thirteen rows through the ownership test, in the artifact.** The five non-L2 rows are recorded with their owners rather than dropped — this is AI-03 § 8's nine-row cross-check at the concern level, and it is what makes "eight verdicts" a demonstrated count rather than a claim.
5. **Make the audit reproducible before making it detailed.** The reverse pass is grep-able against doc 0003's `Closes:` fields; the forward pass carries an independent Traceability-spine column. A reviewer can disprove either without reading this proposal.
6. **Close by inheritance.** The artifact ends with what AG-17, AG-18 and AG-19 each receive, in that milestone's own terms, so the acceptance criterion is checkable from one table.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-agent-v1-scope/` | Six new markdown files | None — new directory |
| `openspec/specs/` | **None** during this change; `agent-v1-scope` is promoted at archive time | — |
| `docs/architecture/0001-...`, `docs/architecture/milestones/0003-...`, `0004-...`, `docs/adr/**` | **None** — read-only. F1 is recorded, not repaired | Low — see "The doc 0001 question" |
| `backend/agent/`, `go.mod`, `go.work`, `docker-compose.yaml`, `infra/` | **None** | — |

## Rollback plan

The change is **purely additive**: one new directory, six new markdown files, zero edits to any existing file. Rollback is `git revert` of this change's commits, or deleting the directory. Nothing is generated from these files, nothing imports them, and no build depends on them. `make test` and `make lint` are unaffected because no Go file changes.

**No partial-rollback hazard** — this is the one place AG-02 is simpler than AI-03, which had to warn that reverting AI-01's three appended rows alone would leave `decision.md` citing unresolvable identifiers. AG-02 amends nothing, so no such coupling exists. The pull request is shared with AG-00 and AG-01; reverting AG-02 alone means reverting its directory, and the two siblings stand without it (both are `Depends on:` inputs to AG-02, not consumers of it).

**Post-merge reversal is the expensive direction, and it is the reason for the wave-0 schedule.** Once AG-17, AG-18 and AG-19 cite verdict identifiers as their scope authority, changing a verdict changes those milestones' acceptance criteria. Reverting after wave 4 has started is not a revert; it is a re-plan.

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| A later milestone reads G3's "never compact" default as evidence that compaction itself is a seam, and treats AG-18's five leaves as optional | Medium | **High** — the whole of AG-18 | The split is stated explicitly as a rule, with `AGS-I` for the mechanism and `AGS-S` for the shipped default strategy, and the misreading is named in the entry. Decision § on the splitting rule |
| A reader takes doc 0001 § 7's "seam now" literally for G1, G3, G5, G11 and re-opens scope at wave 2 or 4 | Medium | **High** — four milestones | F3 rebutted in full, once, citing ADR 0005 § D4 as § 7's own named verdict authority; each of the four entries references the rebuttal |
| "Deferred" in G10's row is read as "no v1 work" and AG-16 is descoped | Low | High | F2 rebutted, citing doc 0001's own disambiguating paragraph |
| The decision quotes doc 0001's G5 seam cell verbatim and propagates the F1 defect into a second document | Medium | Medium — a wrong citation acquires a second source | The decision verdicts G5 from R-13's mapping and states explicitly that it does not reproduce the seam cell; F1 recorded with both line references |
| The graph-consistency audit is asserted ("we checked, it is clean") and a later mismatch is discovered with no way to tell which document drifted | Medium | Medium | Two-pass audit with an independent corroborating column and a grep-able reverse pass; both are re-derivable by a reviewer |
| The four seam omissions (4, 9, 10, 11) are recorded with one reason, hiding that seam 4 is Layer 3's rather than Layer 1's | Medium | Low–Medium — a later reader looks for seam 4 in a Layer 2 milestone | Two distinct reasons stated; seam 4's row calls out that it is the exception to doc 0001 § 6's Layer-1-urgency grouping |
| Over-reach into Layer 3 — the decision starts saying *how* a policy or price table works | Medium | Medium | AI-01 § 9 rule 5's deletion test applied literally: if a sentence were deleted, would Layer 3 have more options? If yes, delete it. Every Layer 3 mention is an owner name plus a doc 0004 node id |
| AG-00 or AG-01 lands a vocabulary or carrier decision that contradicts wording used here | Low | Medium | All three are in one pull request and reviewed together; AG-02 cites concepts, never spellings, and doc 0003's reconcile-or-flag duty applies |

## Dependencies

- **AG-00** (`cachicamas-agent-contract-vocabulary`) — **hard.** Every concern name this decision uses is AG-00's; definitions are cited, never paraphrased.
- **AG-01** (`cachicamas-agent-event-delivery`) — **hard.** G1's protocol verdict and G10's cost-event verdict both presuppose AG-01's delivery model and its upward path. F4's footnote also depends on AG-01's own text.
- Both are decided in parallel in this same wave and pull request. Neither consumes AG-02, so the edge is one-directional and the pull request is internally orderable.
- No new dependency of any kind. No ADR required: this change adds nothing to build.

## Success criteria

- [ ] Both AG-02.1 closing-checklist items are answered in `decision.md`.
- [ ] **Item 1**: every Layer-2-owned register row has at least one verdict, each citing its register row, each stating implement-now / seam-with-trivial-impl / deferred, and each `AGS-S` naming its trivial implementation and its seam number.
- [ ] **Item 2**: the two-pass audit is recorded and clean, with the reverse pass re-derivable from doc 0003's `Closes:` fields.
- [ ] The eight Layer-2-owned concerns (G1, G3, G4's L2 half, G5, G7, G8's L2 half, G10's L2 half, G11) are all verdicted, and the three splits (G8, G3, G7) are named on both halves.
- [ ] The five non-L2 rows (G2, G6, G9, G12, G13) are recorded as `AGS-X` with their owners, and each Layer-3-owned one names a doc 0004 node — **no orphan**.
- [ ] R-18's eight seams are each mapped to a doc 0003 node, and the four omissions carry their two distinct reasons.
- [ ] **AG-02's acceptance criterion is met**: every Layer-2-owned G-concern has a verdict, and every divergence from a documented default — F1, F2, F3, F4 — is rebutted explicitly, citing both sides.
- [ ] AG-17, AG-18 and AG-19 each have a stated inheritance, in that milestone's own terms.
- [ ] Standing amendment rules are stated, so a future concern arrives by amendment rather than by a local downstream verdict.
- [ ] The change adds one directory and six markdown files, edits zero existing files, and contains no type name, field name, method name, or package identifier.

## Open questions for the driver

Answer, correct, or skip — the proposal proceeds on the stated assumption if there is no reply.

1. **Identifier scheme.** The proposal introduces `AGS-I` / `AGS-S` / `AGS-D` / `AGS-X`, mirroring AI-03's `CAP-R` / `CAP-O` / `CAP-X`. Downstream milestones will cite these forever. *Assumption: accepted.* Is there a house prefix you would rather see?
2. **F1's follow-up.** The recommendation records the defect and ships no doc 0001 edit, because the repair (empty cell versus a thirteenth seam) is an architecture choice. *Assumption: record now, repair in its own change.* Do you want it repaired here instead — and if so, which repair?
3. **The `AGS-D` list.** It restates three rows doc 0003's "Explicitly deferred" table already cites to AG-02. *Assumption: restate them with identifiers so downstream milestones cite one source.* Or would you rather the decision point at doc 0003's table and add no identifiers?
4. **How far the inheritance table goes.** `Blocks:` names AG-17, AG-18 and AG-19 only, but AG-09, AG-15, AG-16 and AG-20 also depend on these verdicts through their charters. *Assumption: a full inheritance row for the three blocked milestones, and a one-line pointer for the other four.* Extend it to all seven?

## Notes for the following phases

- **`spec.md`** — the system under test is the artifact, as it was for AI-01 and AI-03. Requirement ids `R-AGS-0NN`, scenario ids `S-AGS-0NN`, Given/When/Then, RFC 2119 keywords per `openspec/config.yaml`. Every scenario must be checkable by inspection, without running anything. Several requirements constrain the *argument* rather than the conclusion — a rebuttal that cites both sides, a negative clause on every entry — because a list of verdicts with no reasons passes a spec that checks only verdicts and is then re-litigated anyway.
- **`design.md`** — owns the structure of `decision.md` and the reasoning rules it applies: the Layer-2-ownership test, the splitting rule, the implement-versus-seam test, and the two-pass audit procedure.
- **`tasks.md`** — two tasks for the two checklist items, plus the four rebuttals, the audit pass, and the verification pass. Delivery is `single-pr` shared with AG-00 and AG-01, with `size:exception` pre-accepted against a 1000-line budget.
- **`decision.md`** — the deliverable. Ends with the inheritance table and the closing-checklist verification table, because AG-02's acceptance is stated in terms of what AG-17, AG-18 and AG-19 can do without reopening this milestone.
- **On length**: the `sdd-propose` skill sets a 450-word budget. This proposal exceeds it deliberately, following the repository's own merged precedent (AI-03's proposal, PR #95) and `openspec/config.yaml`'s proposal rules; the surface AG-02 must settle — thirteen register rows, twelve seams, four rebuttals, a two-pass audit and a document-amendment recommendation — is not compressible to 450 words without dropping content the orchestrator's phase brief requires.
