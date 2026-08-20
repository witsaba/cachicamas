# Spec — the Layer 2 v1 capability scope

> **Change**: `cachicamas-agent-v1-scope`
> **Milestone**: AG-02 · **Node**: AG-02.1 `[decision]`
> **Phase**: spec (new capability — `agent-v1-scope`)
> **Canonical spec**: `openspec/specs/agent-v1-scope/spec.md` — created by `sdd-archive` from this file
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`; every scenario independently verifiable
> **Date**: 2026-08-11
> **Binding inputs**: [`proposal.md`](../../proposal.md) · [`explore.md`](../../explore.md) · doc 0001 § 6 and § 7 · doc 0003 (AG-02 charter, Traceability spine, "Explicitly deferred") · [ADR 0005 § D4](../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Precedent**: [AI-03's spec](../../../archive/2026-07-31-cachicamas-ai-minimum-capabilities/specs/ai-minimum-capabilities/spec.md)
> **Authoring constraint**: doc 0003's authoring constraint binds this file. No type name, field name, method name, or package identifier appears anywhere.

---

## Two identifier namespaces — read this before citing anything

This change uses **two disjoint identifier namespaces**, and a citation that confuses them is ambiguous.

| Namespace | Shape | Lives in | Means |
| --- | --- | --- | --- |
| **Spec** | `R-AGS-0NN` (requirement), `S-AGS-0NN` (scenario) | this file | A checkable property of the decision artifact |
| **Verdict** | `AGS-I-NN`, `AGS-S-NN`, `AGS-D-NN`, `AGS-X-NN` | `decision.md` | One entry of one of the decision's four closed lists |

The two never overlap: a spec identifier always carries the `R-`/`S-` prefix and a three-digit ordinal; a verdict identifier always carries a class letter (`I`/`S`/`D`/`X`) between `AGS` and its ordinal. `R-AGS-004` and `AGS-S-04` are different objects. The decision artifact MUST restate this distinction so that a downstream milestone citing `AGS-I-01` is never read as citing a requirement.

---

## Purpose

AG-02.1 is a `[decision]` leaf. It ships no production code, so there is no runtime behavior to specify. **The subject of this spec is the artifact**: `decision.md`, the recorded Layer 2 v1 scope verdicts. Every requirement below constrains that document, and every scenario is a property a reviewer checks against it by inspection, deterministically, without running anything.

Three distinctions shape the requirements and are stated once here.

**The argument is specified, not only the conclusion.** AG-17, AG-18 and AG-19 will read their scope off this artifact. A verdict list with no reasons and no negative clauses is re-litigated at the first milestone that finds it inconvenient. Several requirements therefore constrain reasoning: what a verdict does **not** oblige, why a divergence from a documented default is rebutted, and which of two differently-worded sources is controlling.

**Completeness is checked, not asserted.** The defect this milestone exists to prevent is a missing register row. A requirement that says "every owned row has a verdict" is satisfiable by a document that simply omits a row and says nothing. `R-AGS-002` therefore specifies the *walk* — all thirteen rows, each with a recorded outcome — so that omission is visible rather than silent.

**A scenario a defective decision would still pass proves nothing.** Where a property is mechanically checkable against a source document, the scenario names the grep or the cross-reference a reviewer runs, and names what a failure looks like.

---

## Definitions used by this spec

- **The decision artifact** — `openspec/changes/cachicamas-agent-v1-scope/decision.md`.
- **The closing checklist** — AG-02.1's two items in doc 0003.
- **The register** — doc 0001 § 7's forward-requirements register, rows G1 through G13.
- **A register row** — one of those thirteen rows, cited as `doc 0001 § 7 G-NN`.
- **A Layer-2-owned row** — a register row whose Owner column names Layer 2, in whole or in part.
- **A verdict** — one entry of the decision's `AGS-I` / `AGS-S` / `AGS-D` lists.
- **A cross-check entry** — one entry of the `AGS-X` list; it records ownership, it is not a verdict.
- **A discharging node** — a doc 0003 milestone or node identifier that carries out what a verdict obliges.
- **The Traceability spine** — doc 0003's "Requirements → closing nodes" and "Nodes trace back to scope" tables.
- **A `Closes:` field** — the grep-able per-milestone line in doc 0003 naming what that milestone closes.
- **The seam catalog** — doc 0001 § 6's twelve seams; **the required eight** are R-18's seams 1, 2, 3, 5, 6, 7, 8 and 12.
- **An inheritance statement** — a sentence naming a downstream milestone and what it receives from this decision, in that milestone's own terms.

---

## R-AGS-001 — The artifact exists, is singular, and answers both checklist items

The change MUST produce exactly one decision artifact, at `openspec/changes/cachicamas-agent-v1-scope/decision.md`. It MUST answer both items of AG-02.1's closing checklist, and MUST close with a verification table mapping each item to the section that answers it. No other file of this change MAY restate a verdict as normative.

### Scenarios

- **S-AGS-001** — Given the change directory, when a reviewer lists its files, then exactly one file named `decision.md` is present, AND every other artifact of the change refers to it rather than restating a verdict as normative.
- **S-AGS-002** — Given the decision artifact, when a reviewer walks AG-02.1's two closing-checklist items in order, then each resolves to a named section that states an answer and its evidence, AND the artifact's closing verification table records both items with a status.
- **S-AGS-003** — Given the decision artifact, when a reviewer looks for the gate that applies to this node, then the artifact states that a `[decision]` leaf produces no production code and closes on merge, with no test gate.

---

## R-AGS-002 — Verdict completeness is demonstrated by a total walk of all thirteen rows

The artifact MUST record an outcome for **every** register row G1 through G13 — never a subset. Each Layer-2-owned row MUST carry at least one verdict; each row Layer 2 does not own MUST carry a cross-check entry naming its actual owner. The artifact MUST state the ownership test it applied, and MUST state the resulting counts so that a missing row changes a number rather than passing unnoticed.

### Scenarios

- **S-AGS-004** — Given the artifact, when a reviewer walks doc 0001 § 7's rows G1 … G13 in order against it, then every one of the thirteen appears with a recorded outcome, AND no row is absent.
- **S-AGS-005** — Given the artifact, when a reviewer reads the ownership test, then it is stated as a rule applied to the register's Owner column, expressed so that a later reader can re-apply it to a new row without re-deriving it.
- **S-AGS-006** — Given the artifact, when a reviewer reads the stated counts, then eight Layer-2-owned rows (G1, G3, G4's Layer 2 half, G5, G7, G8's Layer 2 half, G10's Layer 2 half, G11) and five non-Layer-2 rows (G2, G6, G9, G12, G13) are named individually, AND the two counts sum to thirteen.
- **S-AGS-007** — Given a hypothetical artifact from which one Layer-2-owned row were removed, when a reviewer applies S-AGS-004 and S-AGS-006, then the omission is detected by the row walk and by the count, rather than depending on the reviewer noticing an absent paragraph.
- **S-AGS-008** — Given the artifact, when a reviewer checks the eight Layer-2-owned rows against AG-02's charter defaults in doc 0003, then the charter's eight documented defaults and the artifact's eight owned rows correspond one to one.

---

## R-AGS-003 — Every verdict entry carries the same closed shape

Each `AGS-I`, `AGS-S` and `AGS-D` entry MUST carry: a stable identifier; the register row it decides, cited as `doc 0001 § 7 G-NN`; its class — implement-now, seam-with-trivial-implementation, or deferred; what it obliges; **what it does not oblige**; and the doc 0003 milestone or node that discharges it. Every `AGS-S` entry MUST additionally name its trivial implementation and its seam number. The negative clause is normative, not editorial.

### Scenarios

- **S-AGS-009** — Given each entry of the three verdict lists, when a reviewer reads it, then an identifier, a register-row citation, a class, an obliges clause, a does-not-oblige clause, and a discharging node are all present.
- **S-AGS-010** — Given each `AGS-S` entry, when a reviewer reads it, then the trivial implementation is named as a concrete behavior (for example, that the shipped default never compacts, that no failover occurs, that no subagents exist) AND the seam number from doc 0001 § 6 is cited.
- **S-AGS-011** — Given the entry for G7's structural half, when a reviewer reads its negative clause, then it states that proving re-entrancy obliges **no** shipped subagent tool, citing v2 § 8's non-goal.
- **S-AGS-012** — Given any entry whose negative clause were absent, when a reviewer applies S-AGS-009, then the entry fails this requirement, because an entry that does not say what it fails to oblige is the defect a downstream milestone converts into unagreed scope.

---

## R-AGS-004 — The splitting rule is stated as a rule, and G3's split is stated by name

> **Amended 2026-08-11 (remediation) — G3 and G8 corrected to three-way splits, matching G7's pattern.** This requirement originally described G8 as a two-way split (retry / failover) and G3 as a two-way split (mechanism / shipped default strategy), and `S-AGS-014` expected two verdict entries for each. `R-AGS-006` separately mandates a third, `AGS-D` entry for each of G3, G7 and G8 — the three rows doc 0003's own "Explicitly deferred" table cites to AG-02 by name — and that entry is neither the mechanism/property (`AGS-I`) nor the seam's trivial default (`AGS-S`): it is the seam's eventual full replacement, a third and separate object. `G7` already reflected this correctly, three-way, in this requirement's original text; `G3` and `G8` did not, which put `S-AGS-014` and `R-AGS-006` in conflict — no artifact could satisfy both at once. This amendment corrects `G3` and `G8` to three-way splits, consistent with `G7` and with `R-AGS-006`'s own count, rather than bending the decision artifact — whose fourteen entries were already correct — to fit the smaller, inconsistent count.

The artifact MUST state that one register row MAY yield more than one verdict, and that every split MUST be explicit on **every named half** rather than resolved by picking one. The three splitting rows — G8 (retry / failover seam / failover implementation), G3 (compaction mechanism / shipped default strategy / compaction quality), G7 (structural property / delegation seam / production tool) — MUST each be recorded with every half carrying its own identifier and class. G3's split MUST additionally name the misreading it prevents.

### Scenarios

- **S-AGS-013** — Given the artifact, when a reviewer looks for the splitting rule, then it is stated once as a general rule, before the lists, AND it states that a split row is recorded on every half.
- **S-AGS-014** — Given G8, G3 and G7, when a reviewer counts their verdict entries, then each has three, each with its own identifier and class, AND no half is left implicit.
- **S-AGS-015** — Given G3's mechanism and default-strategy entries, when a reviewer reads them, then the compaction **mechanism** is classed implement-now with AG-18's leaves named, AND the shipped **default strategy** is classed seam-with-trivial-implementation with "never compact" named as the trivial implementation and AG-17.1 as its node.
- **S-AGS-016** — Given G3's entries, when a reviewer looks for the stated misreading, then the artifact states in its own text that conflating "the shipped default strategy never compacts" with "compaction is a seam" would make AG-18's five leaves look optional, and states that this reading is wrong.
- **S-AGS-017** — Given the artifact's stated verdict count, when a reviewer compares it to the count of Layer-2-owned rows, then the artifact states that the two numbers differ because of the splitting rule, rather than leaving eight rows to be mistaken for eight identifiers.

---

## R-AGS-005 — Non-Layer-2 rows are recorded with their owners, never dropped

The artifact MUST record G2, G6, G9, G12 and G13 as `AGS-X` cross-check entries. Each MUST name its actual owner and the doc 0004 node or Layer 1 milestone that owns it. The artifact MUST state that an `AGS-X` entry is a cross-check rather than a verdict, and MUST state why recording these rows is required even though AG-02's acceptance criterion is scoped to Layer-2-owned concerns.

### Scenarios

- **S-AGS-018** — Given the artifact, when a reviewer reads the `AGS-X` list, then G2, G6, G9, G12 and G13 are each present with an identifier, a named owner, and a named owning node — at minimum G2 to doc 0004 CO-04.1, G6 to CO-02.1, G9 to Layer 1 AI-12, G12 to Layer 1 AI-07/AI-13/AI-18, and G13 to Layer 1 AI-02.
- **S-AGS-019** — Given the `AGS-X` list, when a reviewer asks whether these entries are verdicts, then the artifact states that they are not, AND states that they exist so that "eight verdicts" is a demonstrated count rather than a claim.
- **S-AGS-020** — Given G13's entry, when a reviewer reads it, then AI-02 is named as its deciding node, AND a footnote states that AG-01's self-description as "the G13 of this layer" is an analogy and does **not** discharge the register row.

---

## R-AGS-006 — The deferred list restates doc 0003's own deferrals with identifiers

The artifact MUST record as `AGS-D` the three items doc 0003's "Explicitly deferred" table already cites to AG-02: a production subagent tool with delegation depth limits, the failover implementation, and compaction quality. Each MUST name the seam or node that holds its place and the reason for deferral, and each MUST be traceable to its row in that table.

**Back-annotation (AG-15, 2026-08-18) — the failover row's placeholder node has shipped its seam; the deferral stands.** `AGS-D`'s failover entry names AG-15.3 as the node holding the place for the failover **implementation**. AG-15.3 has now shipped that place: a named injection point on the caller-owned harness, consulted exactly once when the retry budget for a logical turn is exhausted, whose v1 implementation **declines** and whose typed verdict makes acceptance unconstructible in v1 (`R-RTY-010`), together with a pin proving the seam's presence changes no observable behavior (`R-RTY-011`). **What remains deferred is unchanged and is the whole of the entry's substance**: choosing a substitute route, re-counting the context budget, re-pricing, and restarting the cache prefix — the obligations the interface's documentation now names for a real implementer, and which cross into AG-17/Layer 3 territory per seam 8's rationale (`0003:1454`). The entry MUST NOT be removed, reworded as closed, or re-owned: a shipped seam with a declining implementation is a held place, not a delivered capability.

**Back-annotation (AG-19) — the SUBAGENT-TOOL row's placeholder node has shipped its substrate; the deferral STANDS, and this is the entry most at risk of being wrongly closed.** `AGS-D`'s subagent-tool entry names **AG-19** as the node holding the place for a production subagent tool with delegation depth limits, and AG-19 has now shipped. What it shipped is the **substrate**, not the tool:

- **Shipped**: a context-carried publishing seam installed for exactly one tool call and revoked on every exit path; an admissibility rule derived from the existing event registry; and proof of four structural properties under a real nested run — parent attribution by a one-hop walk, sibling children interleaving with no cross-talk under `-race`, nested cancellation inherited through the existing tree and completing leaf-first, and nested cost and permission scope.
- **Still deferred, and it is the whole of the entry's substance**: a production subagent tool, subagent configuration, and **delegation depth limits**. AG-19 ships **no** notion of depth at all — not a limit, not a counter, not a field — and its seam holds no child lifecycle, no configuration and no subagent concept in any identifier.
- **The deferral is enforced structurally rather than promised.** Every subagent concept in AG-19 lives in `package agent_test`, which production code cannot import; the seam's concrete type, its constructor and its installer are unexported, so no code outside the package can mint or install one. **"No subagent tool ships in v1" is enforced by the compiler, not by this table.**

The entry MUST NOT be removed, reworded as closed, or re-owned. A proven substrate is not a delivered capability, on exactly the AG-15 argument above.

### Scenarios

- **S-AGS-021** — Given the artifact, when a reviewer greps doc 0003's "Explicitly deferred" table for `AG-02`, then each of the three rows returned corresponds to exactly one `AGS-D` entry, AND no `AGS-D` entry lacks such a row.
- **S-AGS-022** — Given each `AGS-D` entry, when a reviewer reads it, then the placeholder node is named (AG-19 for the subagent tool, AG-15.3 for failover, AG-18.1 for compaction quality) AND the reason is stated in terms of what deferring protects, not in terms of effort. *(AG-15 update: AG-15.3's failover entry now carries an annotation recording that the placeholder node has shipped its seam. AG-19 update: the subagent-tool entry now carries the same shape of annotation — AG-19 has shipped, and the entry is still deferred. The placeholder node named by this scenario is unchanged, so its falsifiable claim is exactly what it was.)*
- **S-AGS-062** — **AG-19: the deferral is checked against the shipped surface, not against the artifact's prose.** Given the merged AG-19 change, when the package's exported surface is enumerated from an external test package, then it declares **no** subagent type, no child-harness type, no delegation depth, no delegation configuration and no subagent lifecycle; no exported identifier names a subagent concept; and no external caller can construct or install a publishing seam the scheduler will honour — so the `AGS-D` subagent-tool entry is still deferred **by construction** and a reviewer can confirm it without reading this artifact. Cross-referenced to `R-DEL-001` / `R-DEL-009` and `S-DEL-002` / `S-DEL-024`.

---

## R-AGS-007 — R-18's eight seams are mapped, and the four omissions carry two distinct reasons

The artifact MUST map each of R-18's required seams — 1, 2, 3, 5, 6, 7, 8 and 12 — to the doc 0003 node that bears it. It MUST record the four omitted seams — 4, 9, 10 and 11 — each with a stated reason, and MUST state **two different reasons**: seams 9, 10 and 11 are Layer 1 contract items already shipped, while seam 4 is Layer 3's. The artifact MUST NOT record the four omissions under a single reason.

**Back-annotation (AG-17, 2026-08-19) — seams 5 and 6 have shipped their bearing nodes; the mapping is unchanged and both rows now carry their discharge.** `S-AGS-023` maps **seam 5 to AG-17.1** ("the check before every turn") and **seam 6 to AG-17.2** ("the optional counting capability"). Both nodes have now shipped, and what they shipped is recorded here as a **mechanism**, not as a reassurance:

- **Seam 5 → AG-17.1, shipped.** A nil-default context-strategy field on the caller-owned harness, consulted at AG-13's turn boundary — in the run driver's outer loop, between the transcript resolution (`harness.go:512`) and the attempt loop (`harness.go:562`) — receiving the coming logical turn's transcript, the stated budget and the measured accounting. **Its cardinality is counted in LOGICAL TURNS, not attempts**: a run of N logical turns consults it exactly N times however many provider calls those turns issue, because a seam consulted inside the attempt loop could, once AG-18 gives it teeth, mutate the transcript between two attempts of one logical turn and defeat the argument `R-RTY-002`'s own comment relies on (`harness.go:601-612`). **`S-AGS-010`'s worked example is now literally true**: it cites *"that the shipped default never compacts"* as the kind of concrete behavior an `AGS-S` entry must name, and AG-17's verdict type ships with **no field**, so a verdict requesting compaction is unconstructible by **any** strategy a caller could write — the guarantee is a property of the type rather than of the one implementation shipped. Installing that default changes nothing observable: the event stream and the history read-back are **byte-identical** to the same run with the field nil.
- **Seam 6 → AG-17.2, shipped.** Token accounting by **type assertion on the shipped Layer 1 contract** (`ai/provider.go:130-134`) and by no other means, with **no** Layer 2 counting interface — declaring one would have violated `R-AMP-017`, one contract per capability. The result carries three type-level provenance states, and the estimate path is reachable **only** from a genuinely absent capability (`R-AMP-018`); an advertised counter that errors, and an advertised counter answering with an absent count and a nil error, **both** resolve to *unavailable*, because `R-AMP-019` makes those non-conformance rather than absence, and estimating there would launder a non-conformant provider into a working one.

**What this back-annotation does not do**: it does not renumber, remove, re-own or reword the seam mapping — seam 5 is still AG-17.1's and seam 6 still AG-17.2's, and the eight-required/four-omitted counts are unchanged. It also does **not** discharge seam 8, whose failover implementation stays deferred under `R-AGS-006`'s `AGS-D` entry, nor R-11, which is **AG-18's**: a shipped seam whose verdict type cannot request compaction is a held place for compaction, not a delivery of it.

**Back-annotation (AG-18, 2026-08-19) — R-11 / G3 is DISCHARGED, and the sentence above that reserved it is now answered.** The paragraph immediately preceding named R-11 as AG-18's and drew the distinction between a held place and a delivery. **AG-18 is the delivery**, and what it delivered is recorded as a mechanism rather than as a reassurance:

- **The held place became teeth, deliberately.** AG-17's verdict type carried no field, so compaction was unconstructible by any strategy a caller could write. AG-18 amends that extension point in writing — the instruction `S-CTX-005` addressed to this milestone — and re-homes the never-compact guarantee from the **type** onto the **zero value**: a verdict requesting nothing still compacts nothing, on every path, and the same byte-identity two-run pin proves it. **This is the strongest form of the guarantee still available once the extension point opens**, and it was chosen over an accept-flag the harness merely ignores, which would have been strictly weaker.
- **The mechanism, in its five parts** (`agent-compaction`, `R-CMP-001`…`R-CMP-014`): a summarisation call on an **injected** provider with injected options and an **injected** instruction — Layer 2 authors no instruction content on any path, asserted over captured request bytes; a **prefix** replacement through history's single validating commit primitive at a boundary that is both turn-mark-aligned and pairing-closed, with the protected tail preserved value-identical in Layer 1 values and origins; the compaction event family's **first production emission** after twelve milestones constructible-but-unemitted, inside its own dedicated turn bracket so the stream validator stays byte-unchanged; atomicity by **ordering** — the commit is unreachable until the call succeeded and the replacement validated, so an interrupted compaction leaves the pre-compaction transcript intact with no journal and no rollback mechanism; and an on-demand door invoking the same operation, refusing typed whenever a run is in flight and **never** queueing.
- **What the discharge cost, stated so it is not read as free.** AG-18 is the first Layer 2 milestone to **remove** something a run committed. Two promoted MUSTs were amended rather than quietly falsified, and each names its replacement: append-only becomes "exactly one prefix-shaped, pairing-closed removal route through the same commit primitive" (`R-HIS-001`), and unqualified identity stability becomes stability **within a transcript generation**, with the turn identifier as the durable cross-generation handle (`R-HIS-005`). A third — `R-CST-001`'s iff — was **not** relaxed: the compaction bracket satisfies it literally on every arm, and only its enumerated per-path table grew.

**What this back-annotation does not do**: it does not renumber, remove, re-own or reword the seam mapping; the eight-required/four-omitted counts are unchanged; seam 8's failover implementation stays deferred; and `R-AGS-006`'s `AGS-D` entry for compaction **quality** stays deferred with AG-18.1 still named as its placeholder node — a delivered mechanism is not a delivered quality bar, and the two MUST NOT be conflated.

**Back-annotation (AG-19) — SEAM 12 is DISCHARGED, and with it G7's structural half (R-14).** `S-AGS-023` maps **seam 12 to AG-19.1–19.3**, and all three nodes have now shipped. What they shipped is recorded here as a **mechanism**, not as a reassurance:

- **Seam 12's argument was never "build the door later".** It was that the *shape* the door needs — a parent identifier on the envelope from birth, a cancel-cause chain already transitive, a cost bracket run-scoped rather than harness-scoped — cannot be retrofitted once consumers exist. That shape was paid for across five milestones (AG-04.1, AG-06.3, AG-14, AG-16) with **zero** production callers. **AG-19 is the milestone that proves it holds under a real nested run**, and in doing so opens the smallest possible door.
- **The door, in one sentence.** A context-carried publishing seam — two methods, one accessor, two sentinels — installed for exactly one `tool.Run` frame and revoked on every exit path including detach and re-panic, riding the scheduler's **existing** emission funnel so no channel, loss rule or second stamping writer is invented (`R-AGE-017`, exercised unamended).
- **The four structural properties, proven rather than documented.** Parent attribution as a **one-hop walk** through the single `subagent_started`, with no per-event parent stamp and no constructor gaining a parent parameter; siblings interleaving through the one dispatcher with no cross-talk under `-race`, on two distinct `Harness` values; nested cancellation inherited through AG-14's **existing** tree, completing leaf-first, asserted by observed event order and never by elapsed time; and nested cost as a **consumer-side reconstruction** with a production-level refusal — the seam refuses the `cost_turn` kind — that makes a Layer-2 fold unreachable rather than discouraged.
- **What the discharge cost, stated so it is not read as free.** Two shipped claims were **amended rather than quietly falsified**, and each names its replacement: `R-CST-005`'s "exactly one final immediately before the run-close" is re-scoped to the run's **own** `cost_session` events discriminated by run identity, because a mirrored child figure lands mid-bracket; and `R-AEV-003`'s "an event belonging to a delegated harness carries its parent identifier" is scoped to **construction**, with the one-hop walk stated explicitly as what replaces the literal reading. A third — `R-AGE-017`'s no-invented-channel rule — was **not** relaxed: AG-19 adds a named accessor onto an existing channel, and that judgement is recorded in the open in this change's `agent-event-delivery` delta rather than settled by silence.

**What this back-annotation does not do**: it does not renumber, remove, re-own or reword the seam mapping — seam 12 is still AG-19.1–19.3's — and the eight-required/four-omitted counts are unchanged. It does **not** discharge seam 8's failover implementation, and it does **not** close `R-AGS-006`'s `AGS-D` entry for the production subagent tool, which stays deferred with AG-19 still named as its placeholder node. **A proven substrate is not a delivered capability, and the two MUST NOT be conflated.**

### Scenarios

- **S-AGS-023** — Given the artifact, when a reviewer compares its seam table to doc 0003's Traceability-spine row for R-18, then all eight required seams map to the same nodes in both, node for node — at minimum seam 1 to AG-08.1, 2 to AG-10.1, 3 to AG-09.1, 5 to AG-17.1, 6 to AG-17.2, 7 to AG-11.2 with AG-15.1, 8 to AG-15.3, and 12 to AG-19.1–19.3. *(AG-17 update: seams 5 and 6 now carry the back-annotation above recording that their bearing nodes have shipped. The mapping asserted by this scenario is unchanged — seam 5 is still AG-17.1's and seam 6 still AG-17.2's — so this scenario's falsifiable claim is exactly what it was, and the annotation adds discharge evidence rather than altering the assertion. AG-18 update: the same holds — AG-18 discharges R-11 and moves no seam, so this scenario's claim is again exactly what it was. AG-19 update: seam 12's bearing nodes have now shipped, so this scenario's mapping is confirmed by delivery; the assertion it makes is exactly what it was, and no seam moved.)*
- **S-AGS-024** — Given the four omitted seams, when a reviewer reads their reasons, then seams 9, 10 and 11 are stated to be Layer 1 contract items with their shipped milestones named (AI-12, AI-10/AI-11, AI-07), AND seam 4 is stated to be omitted for a different reason — that it is Layer 3's, per G6's owner. *(Unchanged by AG-17, which discharges two **required** seams and touches no omission; note that AG-17 consumes Layer 1's already-shipped counting capability rather than reopening any Layer 1 row. Unchanged by AG-18, which touches no omission either. Unchanged by AG-19, which discharges a **required** seam and touches no omission.)*
- **S-AGS-025** — Given seam 4's row, when a reviewer reads it, then the artifact states explicitly that it is the exception to doc 0001 § 6's own Layer-1-urgency grouping, so that a later reader does not search for seam 4 inside a Layer 2 milestone.
- **S-AGS-026** — Given the artifact, when a reviewer counts the seams it accounts for, then eight required plus four omitted equals the twelve of doc 0001 § 6, with no seam unaccounted. *(Unchanged by AG-17: the back-annotation adds no seam, removes none, and moves no count — `R-AGS-014`'s count-consistency rule is satisfied by there being nothing to update. Unchanged by AG-18 for the same reason. Unchanged by AG-19 for the same reason.)*
- **S-AGS-061** — **AG-18: R-11's discharge is auditable, not asserted.** Given the AG-18 back-annotation above, when a reviewer opens each mechanism it names, then the compaction call, the prefix replacement, the compaction-family emission, the atomicity ordering and the on-demand door each exist as a requirement in `agent-compaction` with at least one independently verifiable scenario; and when the reviewer looks for what remains deferred, then compaction **quality** and the compaction **threshold** are each still recorded as deferred with their owners named — a discharge that also closed those would be over-claiming. **Renumbered from the change's own draft ID, corrected rather than propagated:** the `cachicamas-agent-compaction` delta spec drafted this scenario as `S-AGS-053`, which collides with this spec's pre-existing `R-AGS-014` scenario of that number (the "ninth Layer-2-owned concern" amendment-route scenario) — an authoring defect introduced at the `sdd-spec` phase and never caught before promotion, ironic given `R-AGS-014` is itself the append-only-identifier rule this collision would have broken. `S-AGS-061` is the true next-free identifier — every `S-AGS-0NN` slot from 001 through 060 is already allocated, contiguously, with no gap — so the append-only rule is satisfied rather than violated.
- **S-AGS-063** — **AG-19: seam 12's discharge is auditable, not asserted.** Given the AG-19 back-annotation above, when a reviewer opens each mechanism it names, then the publishing seam, the admissibility rule, the revocation latch, the one-hop parent walk, the sibling isolation, the nested cancellation and the cost refusal each exist as a requirement in `agent-delegation-readiness` with at least one independently verifiable scenario; and when the reviewer looks for what remains deferred, then the production subagent tool, subagent configuration and **delegation depth limits** are each still recorded as deferred with AG-19 named as their placeholder node — a discharge that also closed those would be over-claiming.

---

## R-AGS-008 — The forward audit pass records evidence and an independent corroborating source

The artifact MUST record a forward pass with one row per `AGS-I`, `AGS-S` and `AGS-D` identifier. Each row MUST carry: the verdict class; the evidence its class requires (`AGS-I` ⇒ at least one doc 0003 milestone below it, `AGS-S` ⇒ a named seam-bearing node, `AGS-D` ⇒ a row in doc 0003's "Explicitly deferred" table); the node identifiers; **and, in a separate column, the doc 0003 Traceability-spine R-row that states the same mapping independently**; plus a status.

### Scenarios

- **S-AGS-027** — Given the forward pass, when a reviewer counts its rows, then there is exactly one per verdict identifier, AND no verdict is missing from it.
- **S-AGS-028** — Given each `AGS-I` row, when a reviewer opens the named doc 0003 milestone, then that milestone exists and schedules work; an `AGS-I` row naming no milestone is a defect.
- **S-AGS-029** — Given each `AGS-S` row, when a reviewer opens the named node, then the node exists and bears the seam; an `AGS-S` row with no named node is a defect.
- **S-AGS-030** — Given each forward-pass row, when a reviewer opens the cited Traceability-spine R-row in doc 0003, then that R-row names the same discharging nodes the verdict names — for example R-10 for G1's protocol half, R-11 for G3, R-13 for G5, R-14 for G7, R-15 for G8, R-16 for G10, R-17 for G11.
- **S-AGS-031** — Given the Traceability-spine column, when a reviewer asks what it is for, then the artifact states that it is a second, independent source for the same mapping, so the audit does not rest on the decision's own assertion.

---

## R-AGS-009 — The reverse audit pass is keyed to doc 0003's grep-able `Closes:` fields

The artifact MUST record a reverse pass with one row per doc 0003 milestone carrying a `Closes:` field. Each row MUST name the milestone, what its `Closes:` field names, and either the register row it matches **or** the base-architecture requirement identifier (R-01 … R-09, R-19 … R-21) that explains why it closes no register row. The artifact MUST state the exact grep a reviewer runs to reproduce the pass.

### Scenarios

- **S-AGS-032** — Given the artifact, when a reviewer reads the reverse pass, then it names the reproduction procedure explicitly — grep doc 0003 for `Closes:` and compare the result set to the table — rather than asserting the pass was performed.
- **S-AGS-033** — Given the result of that grep, when a reviewer compares it to the reverse-pass table, then every milestone returned appears in the table, AND every table row corresponds to a returned milestone.
- **S-AGS-034** — Given a milestone whose `Closes:` field names a G-concern, when a reviewer checks the table, then that concern maps to a register row covered by a verdict; a milestone closing a G-concern no verdict covers is a defect.
- **S-AGS-035** — Given the foundational milestones AG-03, AG-04, AG-05, AG-12, AG-13, AG-14, AG-21, AG-22 and AG-23, when a reviewer checks their reverse-pass rows, then each names a base-architecture requirement identifier rather than a register row, AND the artifact states that this is the expected shape because the register is the cross-cutting concerns with no home, not the whole layer.

---

## R-AGS-010 — A disagreement between the two corroborating sources is a defect fixed before closing

The artifact MUST state that the forward pass's verdict-declared nodes and its Traceability-spine column are two independent sources for one mapping, and that a disagreement between them **is** the defect closing-checklist item 2 requires to be fixed before the decision closes — not a note, not a risk, and not a discrepancy recorded and carried forward. The artifact MUST state the same for a reverse-pass mismatch. It MUST record the outcome of both passes as clean or not clean, with the outcome derived from the tables rather than asserted ahead of them.

### Scenarios

- **S-AGS-036** — Given the artifact, when a reviewer reads the audit's disposition rule, then it states that a disagreement between the two forward-pass columns, or a reverse-pass mismatch, blocks closure until repaired, naming closing-checklist item 2 as the authority.
- **S-AGS-037** — Given a hypothetical disagreement, when a reviewer applies the stated rule, then the required action is a repair in one of the two documents, AND the artifact states that recording it as an accepted risk does not satisfy item 2.
- **S-AGS-038** — Given both passes, when a reviewer reads the stated result, then the result is presented as the conclusion of the tables — every row clean — rather than as an opening claim the tables illustrate.
- **S-AGS-039** — Given the artifact, when a reviewer asks whether findings F1 through F4 are audit failures, then the artifact distinguishes them: they are defects in the evidentiary wording of doc 0001, not mismatches inside doc 0003's graph, and item 2's test is about doc 0003's graph.

---

## R-AGS-011 — Every divergence from a documented default is rebutted with both sides cited

AG-02's acceptance criterion requires that any verdict diverging from a documented default rebuts it explicitly. The artifact MUST state that obligation as a standing rule, and MUST record findings F1, F2, F3 and F4, each with the case **for** the source document's reading stated affirmatively before it is answered, each with its evidence cited by location, and each with its disposition. Where two source documents are worded differently, the artifact MUST name which is controlling.

### Scenarios

- **S-AGS-040** — Given the artifact, when a reviewer looks for the rebuttal rule, then it is stated once as a standing method — verdict from the named authority, and say so wherever two sources differ — rather than applied ad hoc.
- **S-AGS-041** — Given each of F1 through F4, when a reviewer reads it, then the reading it opposes is stated affirmatively first, with its own supporting citation, before the answer is given.
- **S-AGS-042** — Given F1, when a reviewer reads it, then both line references are cited (doc 0001 § 7's G5 row against its G1 row), the artifact states that G5's verdict is taken from R-13's mapping and does **not** reproduce doc 0001's seam cell, and a follow-up route is named for repairing doc 0001 in its own change.
- **S-AGS-043** — Given F2, when a reviewer reads it, then the literal reading of "deferred to L2/L3" is answered by citing doc 0001's own disambiguating paragraph, AND G10's Layer 2 half is verdicted implement-now, token-only.
- **S-AGS-044** — Given F3, when a reviewer reads it, then ADR 0005 § D4's fuller phrasing is cited as the controlling source — doc 0001 § 7 itself names it as the verdict authority — with doc 0003's realized milestones for AG-10, AG-18, AG-09, AG-08 and AG-20 as corroborating evidence, AND each of G1, G3, G5 and G11 references the rebuttal rather than restating it.
- **S-AGS-045** — Given F4, when a reviewer reads it, then it is recorded as an analogy rather than a discharge, consistent with S-AGS-020.
- **S-AGS-046** — Given the artifact, when a reviewer looks for edits to doc 0001, doc 0003, doc 0004 or any ADR in this change's diff, then none is present, AND the artifact states that F1 is recorded rather than repaired, with the reason.

---

## R-AGS-012 — No concern is assigned to an owner that does not exist

Every concern this decision assigns to Layer 3 — whether through an `AGS-X` entry, a split row's Layer 3 half, or an `AGS-D` entry whose owner is Layer 3 — MUST name a doc 0004 node that owns it. The artifact MUST record this as a table with node identifiers, so that "nothing is deferred to an owner that does not exist" is verifiable by opening doc 0004 rather than by trusting the decision.

### Scenarios

- **S-AGS-047** — Given the artifact's orphan-check table, when a reviewer collects every Layer 3 assignment the decision makes, then each appears in the table with at least one doc 0004 node identifier.
- **S-AGS-048** — Given each node identifier in that table, when a reviewer opens doc 0004 and searches for it, then the node exists — at minimum G1's policy half to CO-03.1/CO-03.2/CO-16.1, G2 to CO-04.1, G6 to CO-02.1, G10's pricing half to CO-05.1/CO-18.1, G11's concrete hooks to CO-24.1/CO-24.2, and G4's payoff to CO-24.1.
- **S-AGS-049** — Given a hypothetical Layer 3 assignment with no doc 0004 node, when a reviewer applies S-AGS-047, then the orphan is detected by the table's own completeness rather than by reading the prose.

---

## R-AGS-013 — Each blocked milestone has an inheritance statement in its own terms

The artifact MUST close with a table stating what AG-17, AG-18 and AG-19 each inherit from this decision, written in that milestone's own terms, so that each can read its scope off the table instead of re-deriving it from doc 0001 § 7. Milestones that depend on these verdicts through their charters rather than through `Blocks:` — at minimum AG-09, AG-15, AG-16 and AG-20 — MUST each carry at least a pointer to the verdict that governs them.

**Back-annotation (AG-18, 2026-08-19) — AG-18's inheritance statement has been discharged, and it was correct.** `S-AGS-051` requires AG-18's statement to say that the compaction mechanism is **implement-now** and that AG-18's leaves are **not optional**. All five leaves shipped: the compaction call with its own injected provider, options and instruction; the invariant-safe prefix surgery; the first production emission of the compaction event family; atomicity by ordering under interruption; and the on-demand entry point with its typed refusal. **The statement is confirmed by delivery, not amended**, and the entry MUST NOT be removed or reworded as closed for AG-19, whose inheritance statement is untouched by this change.

(Previously: the requirement recorded the inheritance table with AG-18's statement outstanding; a reader after the AG-18 merge could not tell from this requirement whether the leaves it called non-optional had shipped.)

### Scenarios

- **S-AGS-050** — Given the artifact's closing section, when a reviewer reads it, then AG-17, AG-18 and AG-19 each have an inheritance statement naming the verdict identifiers they inherit and what those verdicts settle for them.
- **S-AGS-051** — Given AG-18's inheritance statement, when a reviewer reads it, then it states that the compaction mechanism is implement-now and that AG-18's leaves are not optional, consistent with S-AGS-015 and S-AGS-016. *(AG-18 update: all five leaves have shipped, so the statement's claim is confirmed by delivery; the assertion this scenario makes is unchanged.)*
- **S-AGS-052** — Given AG-09, AG-15, AG-16 and AG-20, when a reviewer looks for their governing verdicts, then each is named with at least a pointer, AND no milestone that cites AG-02 in its charter is left without one.

---

## R-AGS-014 — Standing amendment rules make future concerns arrive by amendment

The artifact MUST state the rules by which it is later amended: a later Layer-2-owned concern arrives by a dated amendment to the decision artifact, never by a local verdict inside a downstream milestone; identifiers are append-only; no existing entry is renumbered; superseded text is struck through rather than deleted; and every stated count is updated to remain consistent.

### Scenarios

- **S-AGS-053** — Given the artifact, when a reviewer asks how a ninth Layer-2-owned concern would be admitted, then the amendment route is stated, AND deciding it locally inside a downstream milestone is stated to be prohibited.
- **S-AGS-054** — Given the amendment rules, when a reviewer reads them, then append-only identifiers, no renumbering, struck-through supersession and count updates are all present, citing AI-03 § 13 and its own 2026-08-10 amendment as the precedent.

---

## R-AGS-015 — Scope discipline, vocabulary discipline, and artifact hygiene

The artifact MUST NOT decide anything Layer 3 owns — sandbox semantics, permission policy content, or pricing. Naming Layer 3 as owner is permitted; deciding how it works is not. Every Layer 2 concern name used in a normative sentence MUST be AG-00's, cited rather than paraphrased. No type name, field name, method name, or package identifier MAY appear in any file of this change, and the change MUST add nothing under `backend/` and modify no build, module or infrastructure file.

### Scenarios

- **S-AGS-055** — Given the artifact, when a reviewer looks for a statement of what a sandbox policy means, which tool a permission policy allows, or how tokens convert to money, then none is made, AND the owning doc 0004 node is named in each case.
- **S-AGS-056** — Given every Layer 3 mention in the artifact, when a reviewer applies the deletion test — would Layer 3 have more options if this sentence were removed? — then no sentence fails it; each mention is an owner name plus a node identifier.
- **S-AGS-057** — Given every Layer 2 concern name the artifact uses in a normative sentence, when a reviewer checks it, then it resolves to AG-00's vocabulary artifact by citation, AND no term is defined locally.
- **S-AGS-058** — Given every file of this change, when a reviewer scans for a single-token camel-case name, a package path, or a method-shaped name, then none is found; every term is a noun phrase with spaces.
- **S-AGS-059** — Given the change's diff, when a reviewer inspects it, then it contains only markdown under `openspec/changes/cachicamas-agent-v1-scope/`, adds nothing under `backend/`, modifies no build, module or infrastructure file, and edits no merged document.
- **S-AGS-060** — Given the artifact, when a reviewer reads its identifier conventions, then the spec namespace (`R-AGS-0NN` / `S-AGS-0NN`) and the verdict namespace (`AGS-I-NN` / `AGS-S-NN` / `AGS-D-NN` / `AGS-X-NN`) are stated to be distinct, so a downstream citation is unambiguous.

**Identifier-allocation defect (recorded 2026-08-20, AG-20) — `S-AGS-064` is allocated and MUST NOT be reused.** AG-19's `agent-v1-scope` delta allocated `S-AGS-064` (named in that change's own `tasks.md` task 4.6 and graded PARTIAL in that phase's verify-report), but the AG-19 promotion into this file dropped it: the promoted range holds `S-AGS-060`…`S-AGS-063` and then jumps to the next milestone's additions with no `S-AGS-064` anywhere in between. Per `R-AGS-014`'s append-only rule, a dropped-at-promotion identifier is not a freed one: `S-AGS-064` stays permanently allocated and unreused, whichever way the discrepancy resolves. AG-20's own additions therefore start at `S-AGS-065` rather than reclaiming `064`. This is the same identifier-allocation failure shape `S-AGS-061`'s comment above records for a different collision (a drafted duplicate ordinal); this one is a promotion-time drop rather than a drafting collision, and the fix is the same: never renumber into the gap.

---

## R-AGS-016 — AG-20 discharges G11 / R-17 and widens seam 1; the concrete hooks stay Layer 3's and the two MUST NOT be conflated (AG-20 addition, 2026-08-20)

**G11 / R-17 is DISCHARGED for its Layer 2 half, and what discharged it is recorded as a mechanism rather than as a reassurance.** The forward pass maps **R-17 to G11** (`S-AGS-030`), and doc 0003's AG-20 charter closes it. What AG-20 shipped:

- **The full taxonomy behind one registration surface.** Four hook points — pre-request (AG-08's, now composed), pre-compact, post-turn and session-start — registered on **one** exported value type held as a **field** on the harness, transported to the turn loop on the harness's own per-turn copy. A registration **method** was unavailable by fence and none was added.
- **A uniform mutate-versus-observe contract enforced BY TYPE, not by documentation.** The two mutating families take a payload and return one with an error; the two observing families **have no result parameters at all**, so a hook that could signal a mutation or a failure is unconstructible. Every payload is a value type with unexported fields and read-only accessors.
- **The asynchrony discipline made mechanical**, which is the charter's own acceptance clause and the standard `R-AGE-008` sets: a per-run observer lane whose enqueue never blocks, dispatch on the lane's own goroutine asserted as a **goroutine-placement** property, a value-stripped observer context that makes the delegation publishing seam unreachable from a hook, a stalled observer proven to delay no event delivery by stream byte-identity, and **no wall clock anywhere** in production or test.
- **Typed, source-named failure on both mutating points**, attributed by source name rather than by a chain-wide ordinal, so the attribution survives a later insertion.

**SEAM 1 IS WIDENED, NOT RE-OWNED.** `S-AGS-023` maps seam 1 to **AG-08.1**, and that mapping is unchanged. AG-08 shipped the seam as a single callable and recorded, in its own spec, that *"AG-20 widens to chain composition"* (`agent-pre-request-hook/spec.md:19`). AG-20 discharges that promise **additively**: the shipped field is kept, runs **first**, and feeds the chain's element 0. Its removal is **AG-23**'s and AG-20 does not take it.

**What is NOT discharged, and this is the clause most at risk of being wrongly closed.** `S-AGS-048` records *"G11's concrete hooks to CO-24.1 / CO-24.2"* — doc 0004's Layer 3 nodes. **AG-20 ships zero concrete hooks.** Cache-breakpoint placement, compaction policy and telemetry are Layer 3's, verbatim from the charter, and this milestone's own out-of-scope table names them. **A taxonomy is not a wiring.** The G11 row's Layer 3 half stays open with CO-24.1 / CO-24.2 named as its holders, on exactly the AG-15 argument this artifact already records: *a shipped seam with a declining implementation is a held place, not a delivered capability*.

**Two further absences MUST be stated so they are inherited knowingly rather than discovered:**

1. **No wall-clock timeout, deadline or "slow hook" threshold ships, and none ever will in Layer 2.** `R-RUN-010` forbids the temporal answer on a structurally similar seam, and `R-AGS-015` forbids Layer 2 deciding Layer 3's content — a threshold is Layer 3 policy. AG-20's "eventually" is the run's **terminal boundary**, a structural moment, not a duration.
2. **`R-AGE-009`'s multi-consumer fan-out is NOT shipped.** AG-20 makes a *hook* non-blocking; it does not build consumer attachment machinery. Envelope invariant 3 closes jointly with AG-01.1; `R-AGE-009` remains decided-and-unbuilt.

**Wave 5's exit is a scheduling fact, not a scope claim.** AG-20 is Wave 5's last milestone, and AG-21 inherits this milestone's frozen postures by name: the caller-owned stalled-observer goroutine leak, the release-before-baseline test rule, unrecovered mutating hooks, and cross-run hook state.

### Scenarios

- **S-AGS-065** — **AG-20: the discharge is auditable, not asserted.** Given the AG-20 statement above, when a reviewer opens each mechanism it names, then the one registration surface, the two type families, the pre-request chain, the pre-compact splice and its re-resolution, the post-turn firing enumeration, the session-start latch, the registration-order dispatch, the observer lane's non-blocking enqueue, the terminal-boundary snapshot and the source-named failure attribution each exist as a requirement in `agent-hook-taxonomy` with at least one independently verifiable scenario; and when the reviewer opens doc 0003's Traceability-spine row for **R-17**, then it names the same discharging node the verdict names, with no node moved and no count changed.
- **S-AGS-066** — **AG-20: the Layer 3 half is checked against the shipped surface, not against this artifact's prose.** Given the merged AG-20 change, when the package's exported surface is enumerated from an external test package, then it declares **no** concrete hook, no cache-breakpoint placement, no compaction policy and no telemetry sink; no hook is registered by default and a zero-value registration value is inert on every path; and when the reviewer looks for a wall clock, then no timeout, deadline, sleep or "slow hook" threshold exists in production or in test. And when the reviewer reads `S-AGS-048`'s orphan-check row for G11's concrete hooks, then it still names CO-24.1 / CO-24.2 as their holders — a discharge that also closed those would be over-claiming, on exactly the AG-15 and AG-19 argument this artifact already records.

---

## Note on length

The `sdd-spec` skill sets a 650-word budget. This spec exceeds it deliberately, following this repository's merged precedent — AI-03's spec (PR #95), the Layer 1 analogue with fifteen requirements and fifty-nine scenarios — and `openspec/config.yaml`'s requirement that every scenario be independently verifiable. Thirteen register rows, twelve seams, four rebuttals and a two-pass audit do not compress to 650 words without dropping content AG-02's acceptance criterion requires.
