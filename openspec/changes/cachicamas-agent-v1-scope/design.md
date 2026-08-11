# Design — the Layer 2 v1 scope verdicts

> **Change**: `cachicamas-agent-v1-scope`
> **Milestone**: AG-02 · **Node**: AG-02.1 `[decision]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-11
> **Inputs**: `explore.md`, `proposal.md`, `specs/agent-v1-scope/spec.md`
> **Output**: the structure and the reasoning rules that `decision.md` implements
> **Precedent**: [AI-03's design](../archive/2026-07-31-cachicamas-ai-minimum-capabilities/design.md) — the Layer 1 analogue, followed in shape and depth
> **Authoring constraint**: doc 0003's authoring constraint binds this file. No type name, field name, method name, or package identifier appears anywhere.

---

## 1. What is being designed

Not a scope. A **verdict table that other milestones will compute against** — which is the same design problem AI-03 solved for Layer 1, at the concern level instead of the capability level.

Seven consumers, and each treats the artifact as data rather than prose:

| Consumer | How it consumes | What breaks if the verdicts are imprecise |
| --- | --- | --- |
| **AG-17** (via `Blocks:`) | Reads the shipped default context strategy and the token-counting posture off its inheritance row | The strategy seam is built against a re-derived scope, and seam 5's "never compact" pin is re-argued |
| **AG-18** (via `Blocks:`) | Reads that the compaction mechanism is implement-now — its five leaves are obligations, not options | The G3 conflation (§ 2.1) descopes the milestone silently |
| **AG-19** (via `Blocks:`) | Reads the three-way G7 split: prove re-entrancy, ship the delegation seam, ship no subagent tool | A production tool creeps in, or the structural proof is skipped as "just a seam" |
| **AG-09, AG-15, AG-16, AG-20** (via their charters) | Each cites the verdict identifier that governs it (spec `R-AGS-013`) | The milestone re-derives scope from doc 0001 § 7's terse disposition column — the exact column F2 and F3 show to be misleading |

Three properties follow, and they drive every structural choice below:

- **The lists are closed and citable.** Downstream milestones cite entries individually and forever, so every entry gets a stable identifier (`AGS-I-NN` / `AGS-S-NN` / `AGS-D-NN` / `AGS-X-NN`). The verdict namespace is disjoint from the spec namespace (`R-AGS-0NN`), and the artifact restates the distinction (spec `S-AGS-060`).
- **Totality is demonstrated, not asserted.** Spec `R-AGS-002` obliges a walk of **all thirteen** register rows with stated counts summing to thirteen — so an omitted row changes a number instead of passing unnoticed. The design accepts this stricter-than-proposal obligation: the "what was decided" section opens with the thirteen-row walk, eight owned plus five cross-checked.
- **Every verdict carries its reason and its negative clause.** The reason is the extension mechanism (a later concern is classified by the same tests); the negative clause is the anti-re-litigation mechanism (an entry that does not say what it fails to oblige is the defect a downstream milestone converts into unagreed scope — spec `S-AGS-012`).

## 2. The failure modes this design targets

### 2.1 The conflation that makes AG-18 optional

The dangerous misreading, worked through. A reader opens the decision, finds "shipped default context strategy: never compact, seam 5", and reasons: *the default never compacts → compaction is a seam → a seam reserves the place and no further work happens in v1 (doc 0001 § 7's own rule) → AG-18's five leaves are optional.* Every step is locally plausible; the chain descopes an entire milestone.

The countermeasure is structural, not editorial — the artifact defeats the chain four times, in four different places a reader might arrive from:

1. **At the lists**: G3 yields two entries with two identifiers in two different lists — the compaction *mechanism* (`AGS-I`, discharged by AG-18.1–18.5) and the shipped *default strategy* (`AGS-S`, "never compact", seam 5, AG-17.1). There is no single "G3 verdict" to cite, so the citation that would carry the conflation cannot be written.
2. **At the entry**: the `AGS-S` entry's negative clause states that a trivial default strategy does **not** make the mechanism optional, names the misreading in its own text (spec `S-AGS-016`), and cross-references the sibling `AGS-I` entry by identifier.
3. **At the audit**: the forward pass lists AG-18's leaves as the *required evidence* for the mechanism's `AGS-I` row — a reader checking the audit finds AG-18 as what the verdict's validity depends on, not as discretionary work.
4. **At the inheritance table**: AG-18's row states in that milestone's own terms that its leaves are not optional (spec `S-AGS-051`).

### 2.2 The literal disposition column

Doc 0001 § 7 marks G1, G3, G5 and G11 "seam now", and § 7's own definition of that phrase ("reserves the place and no further work happens in v1") contradicts the substantial v1 work doc 0003 schedules for all four. The countermeasure is the **controlling-source rule** (§ 8, rule 3): the verdict is taken from ADR 0005 § D4 — the authority § 7 itself names — and every entry where the two sources are worded differently says so.

One precision constraint, verified against the sources rather than inherited from `explore.md`: **D4's fuller phrasing "Seam now, implement in L2" covers G1, G3 and G5 directly, but G11's D4 cell reads bare "Seam now".** G11's entry must therefore not cite D4's phrasing as if it covered G11. Its rebuttal rests on the charter default (doc 0003, AG-02's charter: taxonomy complete, pre-request and post-turn live, pre-compact live with AG-18, session-start emitted) and on the realized milestones AG-08 and AG-20 — with D4 cited as authority for the *method* and for the three sibling rows, not as a source for words it does not contain. A citation is not evidence; the decision quotes only what the cited line actually says.

### 2.3 The silently omitted row

A requirement that says "every owned row has a verdict" is satisfiable by a document that omits a row and says nothing. The countermeasure is the total walk (§ 5, artifact § 2): thirteen rows in register order, each with a recorded outcome, closed by two stated counts (eight owned, five cross-checked) that sum to thirteen. Removing any row breaks either the walk or the arithmetic (spec `S-AGS-007`).

### 2.4 The asserted audit

"We checked, it is clean" leaves a later mismatch unattributable. The countermeasure is § 6: two passes a reviewer executes, each with its exact reproduction procedure stated in the artifact, and a disposition rule that names what a disagreement means and what it forces.

### 2.5 The lumped omission reasons

Recording seams 4, 9, 10 and 11 as "omitted from R-18" under one reason hides that seam 4's reason is different in kind. Seams 9, 10 and 11 are Layer 1 contract items already shipped (AI-12, AI-10/AI-11, AI-07) — doc 0001 § 6's own grouping ("seams 1, 9, 10 and 11 sit on Layer 1 contracts"). **Seam 4 is omitted because it is Layer 3's** (G6's owner), the one exception to that grouping. The seam account (artifact § 7) records the four omissions in two labelled groups, and seam 4's row states it is the exception, so a later reader does not search for seam 4 inside a Layer 2 milestone (spec `S-AGS-025`).

## 3. The verdict taxonomy — four classes, four inclusion tests

Each test is one line, and each keys on a **different observable**, which is what makes the classes mutually exclusive: no concern can plausibly fit two, because no object answers two tests at once.

| Class | Inclusion test (one line) | Observable it keys on |
| --- | --- | --- |
| `AGS-I` — implement-now | v1 ships the behavior, and a doc 0003 milestone below this decision has tests that fail if the behavior is wrong | What v1 **tests exercise**: the behavior itself |
| `AGS-S` — seam with trivial implementation | v1 ships only the point of insertion plus a named trivial default; replacing the default later changes no call site | What v1 tests exercise: the **slot and its default**, never the real behavior |
| `AGS-D` — deferred | No v1 node schedules any work for it; its place is held by an existing seam or substrate, and doc 0003's "Explicitly deferred" table cites it to AG-02 | What v1 schedules: **nothing** |
| `AGS-X` — not owned by Layer 2 | The register's Owner column assigns no part of the concern to Layer 2; the entry names the actual owner's node | The **Owner column**, not any Layer 2 artifact |

**The implement-versus-seam line, drawn sharply**, because it is what stops a later milestone re-litigating scope. Ask of the concern: *what arrives later, and what changes when it does?*

- For an `AGS-I`, **nothing arrives later** — the behavior is already there, and its tests already bite. Compaction's invariant-safe surgery, the ordered rejoin, bounded-backoff retry: each has a v1 milestone whose acceptance criterion fails if the behavior is absent.
- For an `AGS-S`, **a value arrives later, into a slot that already exists** — and only the value changes. The failover policy ("none" today), the context strategy ("never compact" today), the delegation posture ("no subagents" today): each v1 acceptance criterion is *satisfied by the trivial default*, and the future replacement touches no call site.

The line is therefore a test a reviewer can run per entry: **if the trivial default were the permanent answer, would any v1 milestone fail its acceptance criterion?** No → the concern (or that half of it) is a seam. Yes → it is an implementation, whatever doc 0001 § 7's disposition column says. This test is stated in the artifact's method section and visibly applied at every boundary case, so the distinction survives the people who wrote it.

Worked boundary cases — the three places a fuzzy line would blur:

| Concern half | Test outcome | Class |
| --- | --- | --- |
| G3 compaction mechanism | If surgery/recording/recovery were absent, AG-18.1–18.5 fail | `AGS-I` |
| G3 shipped default strategy | "Never compact" forever satisfies AG-17.1's pin | `AGS-S` |
| G8 retry | If bounded backoff were absent, AG-15.1/15.2 fail | `AGS-I` |
| G8 failover | "None" forever satisfies AG-15.3's named seam | `AGS-S` |
| G7 structural property | If re-entrancy were unproven, AG-19.1–19.3 fail | `AGS-I` |
| G7 delegation seam | "No subagents" satisfies seam 12's v1 posture | `AGS-S` |
| G7 production tool | No v1 node schedules it; deferred-table row exists | `AGS-D` |

## 4. The splitting rule

Stated once, as a rule, before the lists (spec `S-AGS-013`): **a verdict attaches to a concern half, not to a register row; a row whose halves answer § 3's tests differently yields one entry per half, and every half is named — never resolved by picking one.** The un-named half is the half that gets re-litigated.

Three rows split, and the artifact enumerates them at the rule, so eight owned rows are not mistaken for eight identifiers (spec `S-AGS-017` — the artifact states *why* the verdict count exceeds the row count):

| Row | Halves | Classes |
| --- | --- | --- |
| G8 | retry / failover | `AGS-I` / `AGS-S` |
| G3 | mechanism / shipped default strategy | `AGS-I` / `AGS-S` |
| G7 | structural property / delegation seam / production tool | `AGS-I` / `AGS-S` / `AGS-D` |

G3 is the load-bearing split and gets § 2.1's four structural defenses. G8's split gets the full argument aired before rebuttal (why one half of one register row is implemented and the other is a seam: retry is knowable from the typed error alone, failover re-opens token budgets, the price table and the cache prefix — seam 8's own rationale). G7's is three-way because v2 § 8 makes subagents a v1 non-goal while re-entrancy is the part that cannot be added later; its `AGS-I` entry carries the sharpest negative clause in the artifact — proving re-entrancy obliges **no** shipped subagent tool (spec `S-AGS-011`).

## 5. Structure of `decision.md`

```
  §1  How to use this document        ← consumers, and the two identifier namespaces
  §2  What was decided                ← the four lists + the thirteen-row walk, before any argument
  §3  The ownership test              ← the method: Owner-column rule, re-appliable to a new row
  §4  The taxonomy and the splitting rule   ← § 3 and § 4 of this design, stated as method
  §5  The verdict entries             ← AGS-I, AGS-S, AGS-D          (closing-checklist item 1)
  §6  The cross-check entries         ← AGS-X, five rows with owners
  §7  The seam account                ← twelve rows: R-18's eight mapped, four omitted, two reasons
  §8  Findings F1 … F4                ← each: the opposing reading first, then the rebuttal
  §9  The graph-consistency audit     ← forward + reverse passes      (closing-checklist item 2)
  §10 The Layer 3 orphan check        ← every Layer 3 assignment → a doc 0004 node
  §11 What each blocked milestone inherits  ← three full rows + four pointers
  §12 Standing amendment rules        ← the living-graph connection
  §13 Closing-checklist verification  ← two rows + the acceptance criterion restated
```

Sections 5 and 9 are the closing checklist in its own order, so a reviewer walks doc 0003's two items and the artifact in parallel — AI-03's property, kept. Placement decisions that earn their place:

- **§ 3 and § 4 come before the lists** because they are the method, and the four lists are produced by *one* classification process; stating it inside each list would be four chances to state it differently.
- **§ 8 sits before § 9** because the audit's disposition rule must distinguish an audit defect (a doc 0003-internal mismatch, blocks closure) from an F-finding (a defect in doc 0001's evidentiary wording, recorded and rebutted — spec `S-AGS-039`). The distinction is only checkable if the findings are already on the table when the audit's rule is stated.
- **§ 7 sits between the lists and the findings** because the seam account is where F1's absence has to be visible: G5 appears in the verdict list with *no* seam citation, and the seam account shows why — R-18 does not include a G5 seam, and § 6's catalog has none.
- **§ 11 and § 12 close the artifact** because acceptance is stated in terms of what the blocked milestones can do without reopening this one.

### 5.1 The shape of an entry

Every `AGS-I` / `AGS-S` / `AGS-D` entry carries the same six parts (spec `R-AGS-003`); `AGS-S` adds two; `AGS-X` substitutes ownership for obligation:

| Part | `AGS-I` / `AGS-D` | `AGS-S` adds | `AGS-X` instead |
| --- | --- | --- | --- |
| Identifier | ✅ | — | ✅ |
| Register-row citation (`doc 0001 § 7 G-NN`) | ✅ | — | ✅ |
| Class | ✅ | — | stated as *cross-check, not a verdict* |
| What it obliges | ✅ | — | the actual owner |
| What it does **not** oblige (normative) | ✅ | — | the owning doc 0004 node or Layer 1 milestone |
| Discharging doc 0003 node(s) | ✅ | — | — |
| Trivial implementation, named as behavior | — | ✅ | — |
| Seam number from doc 0001 § 6 | — | ✅ | — |

Uniformity is what lets a reviewer check the lists mechanically — the property a list consumed as data needs.

### 5.2 Where the argument's weight goes

```
   contested                                                      settled
   |--------------------------------------------------------------|
   G3's split        F3 (the four        G8's split    F1 (real     AGS-X rows
   (the misreading   "seam now" rows;    (retry vs     defect;      (owners named,
    that descopes     rebutted once,      failover,     recorded     one citation
    AG-18; § 2.1's    referenced four     both sides    not          each; arguing
    four defenses)    times)              aired)        repaired)    them invites
                                                                     doubt)
```

Argument length tracks the gradient. G3's split and F3 get the full treatment including the opposing reading at its strongest. The `AGS-X` rows get one line and one citation each — arguing a settled ownership at length invites a reader to think it unsettled.

## 6. The graph-consistency audit — a procedure a reviewer executes

Closing-checklist item 2 is an acceptance criterion the decision must *pass*. Both passes state their reproduction procedure in the artifact, so the audit rests on re-derivation, not trust.

**Forward pass** — one row per verdict identifier. Columns: identifier · class · evidence kind the class requires (`AGS-I` ⇒ at least one doc 0003 milestone below; `AGS-S` ⇒ a named seam-bearing node; `AGS-D` ⇒ a row in doc 0003's "Explicitly deferred" table) · the node identifiers the **verdict** declares · the doc 0003 **Traceability-spine R-row** that states the same mapping independently · status. The reviewer's cross-reference, stated exactly: open doc 0003's "Traceability spine" section ("Requirements → closing nodes" table) at the cited R-row and compare node identifiers with the verdict's column — for example, G5's entry declares AG-09.2 and AG-09.3, and the spine's R-13 row reads "AG-09.2, AG-09.3". For `AGS-D` rows the check is the grep `grep -n 'AG-02' docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` restricted to the "Explicitly deferred" table — it returns exactly the three rows whose *Decided by* cites AG-02, one per `AGS-D` entry (spec `S-AGS-021`).

**Reverse pass** — one row per doc 0003 milestone carrying a `Closes:` field. The reproduction procedure, stated exactly in the artifact: run

```
grep -n 'Closes:' docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md
```

and compare the result set to the table both ways — every returned milestone appears in the table, every table row corresponds to a returned line (spec `S-AGS-033`). Each row names the milestone, what its `Closes:` field cites, and either the register row a verdict covers **or** the base-architecture requirement identifier (R-01…R-09, R-19…R-21) explaining why it closes no G-concern. The expected shape — re-derived in the artifact, not assumed — is that the foundational milestones close base-architecture identifiers, because § 7's register is the cross-cutting concerns with no home, not the whole layer (spec `S-AGS-035`).

**What a disagreement means.** The forward pass's two columns are two independent sources for one mapping: the verdict column is this decision's transcription; the spine column is doc 0003's own. A disagreement between them **is** the defect checklist item 2 requires fixing before closure — not a note, not a risk carried forward (spec `S-AGS-036`, `S-AGS-037`). The repair lands in whichever document is wrong: if the decision mis-transcribed, the decision is corrected before merge; if doc 0003's spine disagrees with doc 0003's own milestones, that is a doc 0003-internal bug repaired under its living-graph clause in its own amendment — and this decision does not close until it has. The audit's outcome (clean / not clean) is presented as the conclusion of the tables, after them, never as an opening claim the tables illustrate (spec `S-AGS-038`).

**What a disagreement does not mean.** F1–F4 are defects in doc 0001's evidentiary wording, not mismatches inside doc 0003's graph. Item 2's test is about doc 0003's graph; the findings are handled in § 8 of the artifact, under the rebuttal rule, and do not block the audit (spec `S-AGS-039`).

## 7. F1 — recorded, not repaired, and the follow-up route

The decisive argument, worked out so `decision.md` states it rather than gestures at it:

1. **The defect is real.** Doc 0001 § 7's G5 row cites Seam `2` (line 699), byte-identical to G1's citation (line 695), while § 6's twelve-seam catalog defines seam 2 as the permission decision and contains no parallel-tool-scheduling seam. Both line references appear in the record.
2. **The repair is not typographical, because two non-equivalent repairs exist**: G5's seam cell goes empty (as G13's already is), or § 6 grows a thirteenth seam for parallel-tool scheduling. These are different architectural claims — the first says parallel-tool scheduling needs no insertion point; the second adds one to a catalog that R-18 ("Seams 1, 2, 3, 5, 6, 7, 8 and 12 of v2 § 6"), doc 0004 and ADR 0005 all cite **by number**. Choosing is an architecture decision about doc 0001's seam catalog, and AG-02.1's acceptance criterion — scoped to Layer 2 verdicts — grants no authority to make it.
3. **G5's verdict does not depend on the repair.** R-13's mapping (AG-09.2, AG-09.3) cites no seam, and R-18's list does not include a G5 seam. The verdict is taken from that clean mapping, and the entry states explicitly that it does **not** reproduce doc 0001's seam cell — so the defect acquires no second source (spec `S-AGS-042`).
4. **The follow-up route is named, so the defect is not noted and forgotten.** F1's record in the artifact names the route: a doc 0001 amendment in its own change, under doc 0001's own dated-blockquote amendment convention, where the empty-cell-versus-thirteenth-seam choice is argued by whoever holds the authority. The standing amendment rules (§ 12 of the artifact) add the closing half of the loop: when that amendment lands, F1's record receives a dated cross-reference to it — the record is the tracking mechanism, and it is updated, not abandoned.

F2, F3 and F4 need no edit anywhere — terseness and analogy, not error — and each is recorded with the opposing reading stated affirmatively first, its own citation attached, then the rebuttal and the disposition (spec `S-AGS-041`): F2 answered by doc 0001's own disambiguating paragraph below the register; F3 by the controlling-source rule with § 2.2's G11 precision; F4 as a footnote naming AI-02, not AG-01, as G13's deciding node.

## 8. Reasoning rules the artifact applies

Five, each with the failure it prevents:

1. **The ownership test** — a register row is Layer-2-owned iff its Owner column assigns any part of the concern to Layer 2; the owned part is exactly the part verdicted. Stated as a rule a later reader re-applies to a new row without re-deriving it (spec `S-AGS-005`). Prevents both silent omission and scope creep into G2/G6/G9/G12/G13.
2. **The implement-versus-seam test** — § 3's question: *if the trivial default were permanent, would a v1 milestone fail its acceptance criterion?* Prevents the G3 conflation and every future analogue.
3. **The controlling-source rule** — the verdict is ADR 0005 § D4's (the authority doc 0001 § 7 itself names); wherever two sources are worded differently, the entry says so and names which controls; no source is cited for words it does not contain (§ 2.2's G11 constraint). Prevents the literal-disposition misreading and citation drift.
4. **The deletion test for Layer 3 scope** — AI-01 § 9 rule 5, applied verbatim: if a sentence were deleted, would Layer 3 have more options? If yes, the sentence goes. Every Layer 3 mention is an owner name plus a doc 0004 node identifier (spec `S-AGS-056`). Prevents this decision answering CO-02, CO-03, CO-04, CO-05 questions it has no authority over.
5. **The count discipline** — every count the artifact states (eight owned, five cross-checked, thirteen total; eight seams mapped, four omitted, twelve total; three deferrals) is derivable from a table in the same artifact. Prevents the arithmetic and the prose drifting apart, which is what makes `S-AGS-007`'s omission detection work.

## 9. The inheritance design

Artifact § 11 is one table in two blocks — full rows for the `Blocks:` three, pointer rows for the four charter-dependents (the spec's stricter-than-proposal obligation, `R-AGS-013`, accepted):

| Milestone | Row kind | Content, in that milestone's own terms |
| --- | --- | --- |
| AG-17 | full | The shipped default strategy is "never compact" (seam 5) and token counting falls back to estimation (seam 6); the strategy seam's contract is settled here, its quality deferred |
| AG-18 | full | The compaction mechanism is implement-now; the five leaves are obligations, not options; compaction *quality* alone is deferred, attached to AG-18.1's injected instruction |
| AG-19 | full | Prove re-entrancy (implement-now), ship the delegation seam with "no subagents" (seam 12), ship no production tool (deferred) — three identifiers, three postures |
| AG-09 | pointer | G5's verdict identifier (parallel scheduling, ordered rejoin — no seam citation) |
| AG-15 | pointer | G8's two identifiers (retry implement-now; failover seam, trivial implementation "none", AG-15.3) |
| AG-16 | pointer | G10's identifier (cost events implement-now, token-only; pricing is Layer 3's) |
| AG-20 | pointer | G11's identifier (taxonomy complete, observers never block; concrete hooks are Layer 3's) |

Each full row names the verdict identifiers inherited and what they settle; each pointer names the governing identifier, so no milestone that cites AG-02 in its charter is left without one (spec `S-AGS-052`).

## 10. Amendment rules and the living graph

Two revision routes exist and the artifact distinguishes them:

- **A new Layer-2-owned concern** arrives by dated amendment to `decision.md` — never by a local verdict inside a downstream milestone. AI-03 § 13's discipline, including its own exercised precedent (2026-08-10, AI-35 appending an entry with a dated blockquote, struck-through supersession, counts updated): identifiers append-only, no renumbering, superseded text struck through, every stated count updated (spec `R-AGS-014`).
- **An existing verdict disproven by implementation** follows doc 0003's living-graph clause, which adopts doc 0002's revert-and-record rule verbatim: revert to green, record the discovery as graph structure, land the amendment in the resuming PR. The design binds the two documents' amendments together: a verdict revision and the doc 0003 graph change it implies travel in the same PR, and the affected forward- and reverse-pass rows are re-derived in the same amendment — so the audit never silently goes stale against a revised verdict.

## 11. Alternatives considered and rejected

| Alternative | Why rejected |
| --- | --- |
| Minimal table-only decision (explore § 9, approach 2) | No rebuttals violates AG-02's acceptance criterion literally; no inheritance table sends AG-17/18/19 back to doc 0001 § 7 — the re-litigation this milestone exists to prevent |
| Three lists, no `AGS-X` cross-check | "Eight verdicts" becomes a claim instead of a demonstrated count; a dropped row is invisible (§ 2.3); G13's F4 confusion has nowhere to be recorded |
| One verdict per row, splits as footnotes | The un-identified half cannot be cited, and the uncitable half is the one re-litigated; G3's conflation defense (§ 2.1) depends on two identifiers existing |
| Repair F1 in this PR | § 7's authority argument: the repair is an architecture choice about doc 0001's seam catalog, cited by number by three documents; AI-03's same-PR amendment precedent does not transfer because AI-01 § 9 rule 2 provided an explicit route and doc 0001 offers none |
| Assert the audit, spot-check on review | A later mismatch cannot be attributed to a document (§ 2.4); the reverse pass's grep is cheap and total |
| Inheritance rows for the `Blocks:` three only | Spec `R-AGS-013` is deliberately stricter: AG-09/AG-15/AG-16/AG-20 cite these verdicts through their charters, and a milestone with no pointer re-derives scope from the misleading column |
| Point at doc 0003's deferred table instead of `AGS-D` identifiers | Downstream would cite two sources with no identifier joining them; the deferred rows already cite AG-02 as decider, so AG-02 owes them identifiers |

## 12. Rollout and rollback — for a decision artifact

**Rollout is the merge.** The verdict identifiers become citable the moment the PR lands; AG-17, AG-18 and AG-19 (wave 4) and the four charter-dependents cite them from their SDD changes onward. Nothing is generated, imported or built from the artifact; no flag, phase or migration exists.

**Rollback has three regimes**, stated so the wave-0 placement argument stays visible:

1. **Pre-merge**: delete the directory. The PR is shared with AG-00 and AG-01, but both are inputs to AG-02, not consumers — reverting AG-02 alone is safe.
2. **Post-merge, pre-wave-4**: a verdict is revised only by dated amendment under § 10's rules — never by silent edit — and the revision is cheap because nothing yet cites the identifier.
3. **Post-wave-4**: a changed verdict changes the acceptance criteria of milestones that cite it; that is a re-plan under the living-graph clause, not a revert. This asymmetry is why AG-02 is in wave 0.

## 13. Threat matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The change is six markdown files under one new directory.

## 14. Verification approach

Every spec requirement is checkable by inspection; nothing runs, which is a property of the node type — `openspec/config.yaml` gates TDD on Go service code, and this change writes none. The verification pass in `tasks.md` orders checks by cost of a missed defect:

| Rank | Check | Spec | Cost if missed |
| --- | --- | --- | --- |
| 1 | G3's split: two identifiers, the misreading named, AG-18's inheritance row consistent | R-AGS-004, S-AGS-051 | AG-18 descoped silently — the single most expensive defect |
| 2 | The thirteen-row walk with counts summing to thirteen | R-AGS-002 | A silently omitted row; the milestone's core defect |
| 3 | F3 rebutted once from the controlling source, referenced four times, G11 cited precisely | R-AGS-011 | Scope re-opened at wave 2 or 4 from a literal reading |
| 4 | Both audit passes present with their exact reproduction procedures and disposition rule | R-AGS-008/009/010 | A later mismatch unattributable; item 2 unsatisfied |
| 5 | Every entry's negative clause present; G7's names "no shipped subagent tool" | R-AGS-003 | Unagreed scope acquired downstream |
| 6 | Seam account total (8 + 4 = 12), two omission reasons, seam 4 the stated exception | R-AGS-007 | Seam 4 hunted for inside a Layer 2 milestone |
| 7 | F1 recorded with both line references, no seam-cell reproduction, follow-up route named | S-AGS-042, S-AGS-046 | The defect propagates or is forgotten |
| 8 | Orphan-check table complete against doc 0004 | R-AGS-012 | A concern deferred to an owner that does not exist |
| 9 | Inheritance: three full rows + four pointers | R-AGS-013 | A charter-dependent re-derives scope |
| 10 | Amendment rules + living-graph binding stated | R-AGS-014 | A ninth concern decided locally downstream |
| 11 | Hygiene: no Go identifiers, no `backend/` files, no merged-document edits, namespaces distinct | R-AGS-015 | Authoring-constraint breach; ambiguous citations |

## 15. Acceptance criteria for the design phase

1. `decision.md` follows § 5's spine; sections 5 and 9 answer the two closing-checklist items in order, and every section outside the spine has a stated placement reason.
2. Every entry carries § 5.1's shape; `AGS-S` entries name their trivial implementation as behavior and their seam number.
3. § 3's four inclusion tests appear in the artifact, and the implement-versus-seam test is visibly applied at all three splits.
4. § 2.1's four structural defenses of G3 are all present and cross-consistent.
5. The audit states both reproduction procedures verbatim (the spine cross-reference and the `Closes:` grep) and the disagreement disposition rule, with the outcome derived after the tables.
6. F1's four-step argument (§ 7) appears, including the named follow-up route and the dated-cross-reference rule.
7. The inheritance table has seven rows: three full, four pointers.
8. G11's F3 rebuttal does not attribute D4's fuller phrasing to G11's row.
9. No file of this change contains a type, field, method, or package identifier; the change edits no existing file.

## 16. Next phase

`tasks.md` — one task per closing-checklist item, plus the four rebuttals, the audit passes, and the verification pass ordered per § 14. Then `decision.md`, the deliverable.

---

**Note on length**: the `sdd-design` skill sets an 800-word budget. This design exceeds it deliberately, following the repository's merged precedent — AI-03's design (PR #95), the Layer 1 analogue of comparable depth — because the artifact it structures must survive being computed against by seven milestones, and the reasoning rules are the content, not commentary.
