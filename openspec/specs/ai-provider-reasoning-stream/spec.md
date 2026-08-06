# Spec — the reasoning stream: a recorded capability absence

> **Change**: `cachicamas-ai-provider-reasoning-stream`
> **Milestone**: AI-29 · **Nodes**: AI-29.0 `[decision]`; AI-29.1 · AI-29.2 · AI-29.3 struck by this change
> **Phase**: spec (delta — new capability)
> **Canonical spec**: `openspec/specs/ai-provider-reasoning-stream/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-08-04
> **Requirement IDs**: `R-ARS-001` … `R-ARS-018` · **Scenario IDs**: `S-ARS-001` … `S-ARS-047`
> **Requirement count: 18** · **Scenario count: 47** (`[test]`: 7 · `[inspection]`: 40)
> **Binding inputs**: [`proposal.md`](../../proposal.md) · [AI-24's decision record](../../../cachicamas-ai-first-provider-decision/decision.md) §§ 5, 7, 8, 12, 13.2 · [AI-24's spec](../../../cachicamas-ai-first-provider-decision/specs/ai-first-provider-decision/spec.md) — the `R-APD` artifact-as-subject model this spec follows · [the pinned-dialect citations](../../../cachicamas-ai-provider-text-stream/citations.md) **C7**, **C8** · [the v1 capability set](../../../../specs/ai-minimum-capabilities/spec.md) — `CAP-O-01`, § 6, § 10 · [doc 0002](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) — the AI-29 charter, AI-29.0's closing checklist, the revert-and-record clause rule 4, completion-checklist item 6, the `G12(b)` spine row, the completion-checklist→nodes mapping

---

## ADDED Requirements

## Purpose

AI-29.0 is a `[decision]` leaf and doc 0002's AI-29 charter offers two legal deliverables — a mapping, or a documented capability absence. This change takes the absence branch. It ships **no production Go**: the absence is already implemented by landed code (translation-time refusal, a `false` capability declaration, an `absent` outcome recording, an unmapped token count). **The subject of most of this spec is therefore the artifact** — the recorded verdict, its grounds, its price and its reopen triggers — exactly as `R-APD` treated AI-24's.

One exception exists and is deliberate. AI-29's acceptance clause ("provider reasoning never leaks into text events") is a **behavioral** claim, and an artifact asserting it is weaker than a test pinning it. `R-ARS-015` and `R-ARS-016` therefore specify one behavioral pin — the reopen trigger's own case, a non-standard `reasoning_content`-style extension field inside a delta — discharged by observable stream behavior rather than by the absence of code.

Four distinctions shape the requirements and are stated once here.

**Absence is a posture, not a hole.** A verdict recording only "no reasoning" is indistinguishable from a node nobody implemented. `R-ARS-003` and `R-ARS-014` require the four already-landed mechanisms and their proving tests to be named and their mechanical form verified rather than asserted.

**The decision is made against the pinned dialect, now.** AI-29.0's text defers to "the exact backend chosen for AI-38/AI-39", but doc 0002 line 2221 makes **AI-38 depend on AI-29**: waiting deadlocks the graph. `R-ARS-004` requires that deadlock to be stated and confirmation routed forward to AI-38.2, where AI-24 § 8 already arms an expected-versus-generated comparison in which "a difference in either direction is a finding".

**A decision that strikes three nodes must leave a reversible record.** `R-ARS-005` and `R-ARS-010` make the reopen triggers observable conditions with owners, and keep the struck text legible for a future adapter against a signing dialect.

**A box no node can close must not be left silently open.** Completion-checklist item 6's wire half named AI-26.6 and AI-29.2; AI-26.6 landed as a refusal and this change strikes AI-29.2. `R-ARS-012` resolves it by **restate-and-publish** — no appended node — because a path with no v1 consumer does not earn one.

## Requirement ownership

| Concern | Requirements |
| --- | --- |
| The decision artifact — verdict, grounds, price | `R-ARS-001` … `R-ARS-008` |
| The doc 0002 living-graph amendment | `R-ARS-009` … `R-ARS-013` |
| The already-landed absence machinery | `R-ARS-014` |
| The behavioral pin | `R-ARS-015`, `R-ARS-016` |
| Scope and hygiene | `R-ARS-017`, `R-ARS-018` |

## Definitions used by this spec

- **The decision artifact** — `openspec/changes/cachicamas-ai-provider-reasoning-stream/decision.md`.
- **The pinned dialect** — the vendor dialect schema at the commit AI-24 pinned, as quoted by citations **C7** and **C8**; not any deployment of a server that speaks a superset of it.
- **A wire claim** — any claim this change makes about what the provider's stream does or does not carry.
- **A pinned-dialect citation** — a wire claim resolved to **C7** or **C8** at the pinned commit.
- **A landed-test citation** — a claim resolved to a named test function in a file already on the branch.
- **The four landed mechanisms** — translation-time refusal (`refuseReasoning`, called first inside `Translate`), the conformance factory's `false` reasoning declaration, the suite's up-front `absent` recording for a declared-absent optional capability, and the unmapped reasoning token count.
- **An extension field** — a JSON member inside a streamed delta object that the pinned dialect's delta schema does not declare.
- **An amendment** — a dated blockquote under a touched doc 0002 heading, per that document's revert-and-record clause rule 4.
- **Struck** — superseded text left visible with strikethrough, never deleted.

---

## R-ARS-001 — The artifact exists, is singular, and closes AI-29.0's two-item checklist

The change MUST produce exactly one decision artifact. It MUST answer both items of AI-29.0's closing checklist: whether v1 emits reasoning events or documents a capability absence, and — because absence wins — what follows for AI-29.1 … AI-29.3 and for AI-23.8's capability outcome. No other file of this change MAY restate a verdict as normative.

### Scenarios

- **S-ARS-001** `[inspection]` — Given the change directory, when a reviewer lists its files, then exactly one decision artifact is present AND every other artifact refers to it rather than restating a verdict as normative.
- **S-ARS-002** `[inspection]` — Given the artifact, when a reviewer walks AI-29.0's two checklist items in order, then each resolves to a passage stating an answer and its basis, and neither resolves to a forward reference alone.

---

## R-ARS-002 — The verdict is absence, and it is grounded in the pinned dialect

The artifact MUST record the verdict as a **documented capability absence** for the first adapter, unambiguously, with no co-equal alternative left standing.

The verdict's grounds MUST rest on the pinned dialect's own schema: that the streamed delta schema declares a closed set of properties containing no reasoning-bearing field (**C7**), and that the only reasoning-shaped datum the dialect reports is an optional integer count inside the usage details object (**C8**) — a count, carrying no text, no signature, and nothing replayable.

The artifact MUST state that a count is not a block, so that a reader cannot mistake the reported count for a partial capability.

### Scenarios

- **S-ARS-003** `[inspection]` — Given the artifact, when a reviewer looks for the emission verdict, then absence is stated for the first adapter and no emission branch is left co-equal or conditional.
- **S-ARS-004** `[inspection]` — Given the grounds, when a reviewer asks what the delta schema declares, then the artifact resolves the claim to **C7** at the pinned commit, names the declared property set, and states that none of them is reasoning-bearing.
- **S-ARS-005** `[inspection]` — Given the reasoning token count, when a reviewer reads its entry, then the artifact resolves it to **C8**, states that it is an optional integer inside the usage details object, and states explicitly that a count carries no text, no signature and nothing replayable — so it cannot be mapped to a reasoning event.

---

## R-ARS-003 — The four landed mechanisms are named, each with the test that proves it

The artifact MUST enumerate all four landed mechanisms that already implement the absence, and MUST cite for each the named landed test that proves it. It MUST state that absence is therefore a posture the adapter actively takes, not an unimplemented node.

### Scenarios

- **S-ARS-006** `[inspection]` — Given the artifact, when a reviewer walks the four landed mechanisms, then each is named with the production location that implements it AND a named landed test that proves it, and none is asserted without one.
- **S-ARS-007** `[inspection]` — Given the refusal mechanism, when a reviewer reads its entry, then the artifact states that refusal runs before any body assembly, and cites the landed test proving refusal at every message position and for every reasoning state — not only for a single representative case.

---

## R-ARS-004 — The decision is made against the pinned dialect, and the scheduling deadlock is stated

The artifact MUST state that AI-29.0's "decided against the exact backend chosen for AI-38/AI-39" is **unsatisfiable as written**, citing doc 0002's declared edge that AI-38 depends on AI-29, and MUST state the consequence: waiting for the backend deadlocks the graph.

It MUST therefore record the decision as made against the **pinned dialect**, name the pinned dialect as the only artifact that exists at decision time, and MUST NOT present the decision as confirmed against a backend.

Confirmation MUST be routed forward to **AI-38.2's expected-versus-generated capability comparison**, and the artifact MUST restate that comparison's standing rule: a difference in **either** direction is a finding.

### Scenarios

- **S-ARS-008** `[inspection]` — Given the artifact, when a reviewer asks why the decision is not deferred to the backend, then the dependency edge is cited by its doc 0002 location and the deadlock is stated as a graph fact rather than as a scheduling preference.
- **S-ARS-009** `[inspection]` — Given the verdict, when a reviewer asks what it was decided against, then the pinned dialect is named, and the artifact states that no backend has been named at decision time.
- **S-ARS-010** `[inspection]` — Given the artifact, when a reviewer asks who confirms the verdict, then AI-38.2 is named as the confirming node, its expected-versus-generated comparison is named as the mechanism, and the either-direction finding rule is restated.

---

## R-ARS-005 — Both reopen triggers are stated as observable conditions with named owners

The artifact MUST state exactly two reopen triggers as **observations**, not intentions:

1. the backend selected for AI-38/AI-39 is documented to carry a reasoning-bearing field on its streamed delta;
2. the dialect schema at a future pinned commit declares such a field.

Each trigger MUST name the node or role that would observe it, and the artifact MUST state that either one alone reopens AI-29 and un-strikes AI-29.1 … AI-29.3.

A trigger stated only as a future intention, or without an owner, MUST be treated as a defect, because a trigger nobody watches cannot fire.

### Scenarios

- **S-ARS-011** `[inspection]` — Given the artifact, when a reviewer counts the reopen triggers, then exactly two are stated, each as a condition someone can observe rather than as a plan.
- **S-ARS-012** `[inspection]` — Given each trigger, when a reviewer asks who observes it, then a named node or role is stated, and no trigger is left unowned.
- **S-ARS-013** `[inspection]` — Given either trigger, when a reviewer asks what firing it costs, then the artifact states that AI-29 reopens and AI-29.1 … AI-29.3 are un-struck, so reversal is a stated procedure rather than an inference.

---

## R-ARS-006 — The price of absence is recorded in the same artifact as the verdict

The artifact MUST record what absence costs. Specifically it MUST state that the AI-29 charter's acceptance clause loses its second half in v1 — no reasoning-bearing request is replayed and no reasoning event is emitted by this adapter — and MUST restate the acceptance clause in absence terms so it remains checkable.

It MUST state that the neutral reasoning contract owned by AI-07 and AI-17 remains **contract-mandatory and frozen**, loses only its first exercising consumer, and is not deprecated by this decision. The strike MUST be scoped to this adapter and stated as such.

### Scenarios

- **S-ARS-014** `[inspection]` — Given the artifact, when a reviewer reads the acceptance clause, then it is restated in absence terms — no reasoning event is emitted and no reasoning-bearing request is replayed — and the restatement is checkable without a mapping existing.
- **S-ARS-015** `[inspection]` — Given the neutral reasoning contract, when a reviewer asks whether this decision deprecates it, then the artifact states that it does not, that it remains contract-mandatory independently of who emits it, and that the strike is scoped to this adapter only.

---

## R-ARS-007 — `CAP-O-01` is confirmed, not re-decided

The artifact MUST record that AI-24's expected outcome for `CAP-O-01` is **confirmed** by this decision, and MUST NOT re-derive or re-decide it. It MUST restate that `absent` is a legal outcome for an optional capability and that an adapter offering none of the optional capabilities is fully conformant, citing the capability contract rather than asserting it.

It MUST restate AI-38.2's standing obligation to assert the **generated** record against that expectation.

### Scenarios

- **S-ARS-016** `[inspection]` — Given the artifact, when a reviewer looks for `CAP-O-01`, then its expected outcome is stated as confirmed with AI-24 named as its owner, and no second derivation of the expectation is present.
- **S-ARS-017** `[inspection]` — Given the artifact, when a reviewer asks whether absence is an adapter defect, then the capability contract's clause is cited by identifier, stating that an adapter lacking every optional capability is fully conformant.

---

## R-ARS-008 — Every wire claim carries its evidence label

Every wire claim in this change MUST be resolvable to a **pinned-dialect citation** (**C7** or **C8**) or to a **landed-test citation**, and MUST be labelled as one or the other. A claim that is neither MUST be labelled an **inference** and MUST state what would confirm it.

Attributing to the pinned dialect a behavior **C7**/**C8** do not state MUST be treated as a defect, because the verdict's entire grounds are the pinned dialect's own schema.

### Scenarios

- **S-ARS-018** `[inspection]` — Given every wire claim in the change, when a reviewer reads its label, then each is labelled pinned-dialect citation, landed-test citation, or inference, and no wire claim is unlabelled.
- **S-ARS-019** `[inspection]` — Given any claim labelled a pinned-dialect citation, when a reviewer checks it against **C7** or **C8**, then that citation states it; a pinned-dialect label on a claim neither citation states is a defect.

---

## R-ARS-009 — The doc 0002 amendment is dated, visible, and never silent

Every change this decision forces on doc 0002 MUST land as an amendment: a dated blockquote under the touched heading, stating what changed, which node changed it, and why. Superseded claims MUST be struck through and left visible; text that is **not** superseded MUST NOT be struck. All amendments MUST land in the same pull request as the decision artifact. Editing doc 0002 without a dated amendment block MUST be treated as a defect.

### Scenarios

- **S-ARS-020** `[inspection]` — Given the doc 0002 diff, when a reviewer inspects every changed heading, then each carries a dated blockquote naming what changed, which node changed it, and why, and no heading changed without one.
- **S-ARS-021** `[inspection]` — Given every strikethrough in the diff, when a reviewer reads the struck text, then each struck passage is a claim this decision genuinely supersedes; a restated-but-still-true claim rendered as struck is a defect.
- **S-ARS-022** `[inspection]` — Given every amendment, when a reviewer checks its date, then it reads **2026-08-04**, and every amendment lands in the same pull request as the decision artifact.

---

## R-ARS-010 — The AI-29 charter and AI-29.0 resolve, and AI-29.1 … AI-29.3 are struck legibly

The amendment MUST resolve the AI-29 charter's deliverable to the documented-capability-absence branch and restate its acceptance in absence terms per `R-ARS-006`.

It MUST record AI-29.0's checklist as **closed with absence**, and MUST record that AI-24's 2026-08-03 note — which stated the evidence "strongly indicates" absence without deciding — is now upgraded to a verdict by this node, the node that owns it.

It MUST strike AI-29.1, AI-29.2 and AI-29.3, stating that they have no subject for **this adapter**, and MUST leave their test lists legible for a future adapter against a signing dialect rather than deleting them.

### Scenarios

- **S-ARS-023** `[inspection]` — Given the AI-29.0 amendment, when a reviewer reads it, then the checklist is recorded closed with absence AND the prior `strongly indicates` note is named as upgraded to a verdict by this node, so the two records do not read as contradicting each other.
- **S-ARS-024** `[inspection]` — Given AI-29.1 … AI-29.3 after the amendment, when a reviewer reads them, then each is struck with its reason scoped to this adapter, and each test list is still readable rather than removed.

---

## R-ARS-011 — Cross-references to the struck leaves are re-pointed, not left dangling

The amendment MUST re-point every doc 0002 cross-reference that names AI-29.1 or AI-29.2 as a future owner at this decision instead. At minimum it MUST cover AI-21's item 3, AI-24.1's item 3, and AI-07's out-of-scope line.

A cross-reference left pointing at a struck node MUST be treated as a defect, because a reader following it lands on a node with no subject and no successor.

### Scenarios

- **S-ARS-025** `[inspection]` — Given doc 0002 after the change, when a reviewer searches for every mention of AI-29.1 and AI-29.2 outside the AI-29 section itself, then each either points at this decision or is itself struck, and none is left dangling at a struck node.
- **S-ARS-026** `[inspection]` — Given each re-pointed cross-reference, when a reviewer reads it, then it states what the reader should now consult rather than only recording that the old target is gone.

---

## R-ARS-012 — Completion-checklist item 6's wire half is restated and published, and no node is appended

The amendment MUST record that completion-checklist item 6's wire half — the clause requiring round-trip tokens to survive byte-exact through *the wire* — is **not exercisable in v1**, and MUST state the two facts that make it so: AI-26.6 landed as a refusal, and AI-29.2 is struck by this change. It MUST state that no reasoning exists on this wire to round-trip, so no v1 node can close that half.

The resolution MUST be **restate and publish**: the item's wire clause is restated as not-exercisable-in-v1 and published through **AI-40.2's readiness contract**, which already carries the Layer-2-strips-reasoning duty. The change MUST NOT append a new milestone or leaf to close it, because the path has no v1 consumer.

The stream half of the round-trip token, already closed by AI-17, MUST be stated as unaffected — not struck — so restating the wire half does not read as reopening a closed property.

### Scenarios

- **S-ARS-027** `[inspection]` — Given the amendment, when a reviewer reads item 6's entry, then the wire clause is restated as not-exercisable-in-v1 with both causes named — AI-26.6's refusal and this change's strike of AI-29.2.
- **S-ARS-028** `[inspection]` — Given the resolution, when a reviewer asks where it is published, then AI-40.2's readiness contract is named as the publishing owner, and the obligation is stated as an obligation on that node rather than as a note.
- **S-ARS-029** `[inspection]` — Given the change diff, when a reviewer looks for a new milestone or leaf identifier created to close item 6, then none exists, and the artifact states why appending one was rejected.
- **S-ARS-030** `[inspection]` — Given item 6's stream half, when a reviewer reads the amendment, then AI-17's closure of it is stated as unaffected and is not struck.

---

## R-ARS-013 — The navigational records are moved with the claim they carry

Wherever doc 0002 carries a navigational restatement of item 6's wire half or of AI-29.2's role — at minimum the `G12(b)` traceability-spine row and the completion-checklist→nodes mapping — the amendment MUST update it to match the restated claim.

A tally, spine row or mapping still naming a struck node as a future closer MUST be treated as a defect, because a reader navigating by the spine would re-derive an obligation the decision retired.

### Scenarios

- **S-ARS-031** `[inspection]` — Given the `G12(b)` spine row after the change, when a reviewer reads it, then it no longer presents AI-29.2 as a pending wire proof, and states the not-exercisable-in-v1 resolution with its publishing owner.
- **S-ARS-032** `[inspection]` — Given the completion-checklist→nodes mapping, when a reviewer reads item 6's node list, then it agrees with the amendment; a mapping that still lists a struck node as a closer is a defect.

---

## R-ARS-014 — The AI-23.8 capability-absence record is verified as already mechanical, not rebuilt

AI-29.0's checklist requires AI-23.8 to record absence as an adapter **outcome** rather than a gap. This change MUST verify that the mechanical form already exists in landed code, and MUST NOT add or modify code to produce it.

The verification MUST establish three properties by inspection: the conformance factory declares reasoning **not offered** explicitly rather than by omission; the suite records the absent outcome **up front**, independently of whether any case for that capability is registered; and the skip-and-record path is reachable **only** for an optional capability, never for a required one.

### Scenarios

- **S-ARS-033** `[inspection]` — Given the landed conformance factory, when a reviewer reads its optional-capability declarations, then reasoning is declared with an explicit non-nil false value rather than left unset, so a declaration and an omission are distinguishable.
- **S-ARS-034** `[inspection]` — Given the landed suite, when a reviewer traces how an absent outcome is recorded, then it is applied up front from the declaration, before any case runs, so absence does not depend on a case existing to report it.
- **S-ARS-035** `[inspection]` — Given the landed skip-and-record path, when a reviewer reads its guard, then it returns no-skip for a required capability before consulting any declaration, so a required capability can never be recorded absent.

---

## R-ARS-015 — An extension field inside a delta is ignored, never leaks into text, and never fails the stream

The change MUST add exactly one behavioral test pinning the reopen trigger's own case as a regression rather than asserting it in prose.

The adapter, when a streamed delta object carries a reasoning-bearing **extension field** the pinned dialect does not declare, MUST ignore that field entirely. Specifically it MUST NOT emit the field's value as, or inside, any text event; it MUST NOT emit any reasoning-typed event for it; and it MUST NOT fail, abort or terminate the stream because of its presence.

The behavior MUST be **drop, not leak, and not fail** — and the test MUST pin all three, since dropping and leaking are distinguishable only by asserting on the emitted events, and failing is distinguishable only by draining to a normal terminal.

### Scenarios

- **S-ARS-036** `[test]` — Given a streamed response whose delta carries both declared content and an undeclared reasoning-bearing extension field, when the stream is drained to its terminal, then no emitted text event contains any byte of the extension field's value.
- **S-ARS-037** `[test]` — Given the same stream, when the emitted events are inspected by kind, then no reasoning-typed event is emitted at all.
- **S-ARS-038** `[test]` — Given the same stream, when it is drained, then it reaches its normal terminal with no failure attributable to the extension field, and the terminal is the same one the identical stream without the extension field reaches.
- **S-ARS-039** `[test]` — Given a delta carrying **only** the undeclared extension field and no declared content, when the stream is drained, then it neither fails nor emits an event for that delta, so the ignore path is proven at the boundary as well as alongside content.

---

## R-ARS-016 — The pin test's fixtures must be non-vacuous

Each fixture backing `R-ARS-015` MUST be constructed so that the assertion can fail. At minimum:

- content MUST exist **after** the extension field's position in the stream and MUST be asserted observably intact, so the test cannot pass on a fixture whose remainder is empty;
- the extension field's value MUST be a distinctive sentinel that appears nowhere else in the fixture, so a leak is attributable rather than ambiguous;
- the fixture MUST be able to distinguish correct ignoring from misrouted-but-unobserved handling — an assertion satisfied equally by "handled correctly" and "silently discarded before reaching the assertion point" MUST be treated as vacuous.

A fixture that cannot distinguish implemented from not-implemented MUST be treated as a defect in the test, not as coverage.

### Scenarios

- **S-ARS-040** `[test]` — Given the fixture, when a reviewer inspects the bytes following the extension field, then declared content is present after it AND the test asserts that content arrives intact and in order, so a fixture with no meaningful remainder cannot pass.
- **S-ARS-041** `[test]` — Given the extension field's value, when a reviewer searches the whole fixture for it, then it occurs exactly once and is distinct from every other value in the fixture, so any leak assertion names an unambiguous source.
- **S-ARS-042** `[test]` — Given the test, when the ignore behavior is deliberately inverted so the extension field's value is routed into a text event, then at least one assertion fails; a mutation that leaves every assertion passing proves the test vacuous.

---

## R-ARS-017 — The change adds no production code

The absence is implemented by the four landed mechanisms. This change MUST therefore add and modify **zero** production Go, and MUST NOT change the module manifest or the workspace file.

The only additions under the backend tree MUST be the single behavioral test file required by `R-ARS-015`.

### Scenarios

- **S-ARS-043** `[inspection]` — Given the change diff restricted to the backend tree, when a reviewer lists every added and modified file, then exactly one file appears, it is a `_test.go` file, and no non-test Go file is added or modified.
- **S-ARS-044** `[inspection]` — Given the change diff, when a reviewer inspects the module manifest and the workspace file, then neither is modified, and no dependency is added.

---

## R-ARS-018 — Scope, authoring constraint, and artifact hygiene

This change MUST NOT decide anything owned by AI-38, AI-39 or AI-40, MUST NOT modify any requirement of any existing capability spec — existing specs are cited by identifier only — and MUST NOT re-decide `CAP-O-01`'s expected outcome.

Per doc 0002's `[decision]` node grammar and its authoring constraint, the **decision artifact** MUST NOT declare or name a Go type, field, method, interface or package identifier belonging to the Layer 1 contract. Naming a landed test function, a landed production function whose behavior is being cited as evidence, and the vendor's wire-level field names is permitted, because the closing checklist and the evidence rule require them.

Outside the one test file, the change MUST add only markdown.

### Scenarios

- **S-ARS-045** `[inspection]` — Given the change diff, when a reviewer inspects every file under `openspec/specs/`, then no requirement of any existing capability spec is added, modified, removed or renamed.
- **S-ARS-046** `[inspection]` — Given the decision artifact, when a reviewer scans for a Layer 1 contract identifier that is not a cited test name or a cited evidence location, then none is found, and every Layer 1 noun is a noun phrase with spaces or a cited `CAP-*` / `R-*` identifier.
- **S-ARS-047** `[inspection]` — Given the change diff, when a reviewer inspects the file list, then it contains only markdown plus the single `_test.go` file, and modifies no build or infrastructure file.

---

## Acceptance criteria

The change holds when:

1. `R-ARS-001` through `R-ARS-018` hold, each verified by its scenarios.
2. AI-29.0's two closing-checklist items are answered in the decision artifact, with the verdict recorded as **absence**.
3. Every wire claim resolves to **C7**, **C8**, or a named landed test, or is labelled an inference with its confirming condition.
4. The four landed mechanisms are each named with a proving test, and none is asserted without one.
5. The pinned-dialect basis and the AI-38/AI-39 deadlock are stated, and confirmation is routed to AI-38.2's either-direction comparison.
6. Both reopen triggers are observable conditions with named owners, and the un-strike procedure is stated.
7. doc 0002 carries one dated **2026-08-04** amendment block covering the charter, AI-29.0, the three struck leaves, the re-pointed cross-references, and item 6's restate-and-publish resolution with its navigational records updated.
8. No milestone or leaf identifier is appended to close completion-checklist item 6.
9. The one behavioral test pins drop, not-leak, and not-fail, on fixtures that fail under a deliberate inversion.
10. No production Go, module manifest, workspace file, build file or existing spec requirement changes.
