# Spec — the first provider and its transport

> **Change**: `cachicamas-ai-first-provider-decision`
> **Milestone**: AI-24 · **Nodes**: AI-24.1, AI-24.2, both `[decision]`
> **Phase**: spec (delta — new capability)
> **Canonical spec**: `openspec/specs/ai-first-provider-decision/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-08-03
> **Requirement IDs**: `R-APD-0NN` · **Scenario IDs**: `S-APD-0NN`
> **Binding inputs**: [the v1 capability set](../../../../specs/ai-minimum-capabilities/spec.md) — the closed lists `CAP-R-01 … CAP-R-05` and `CAP-O-01 … CAP-O-03`, § 10's four-value outcome set and totality rule, § 12's AI-24 inheritance · [cache-boundary markers](../../../../specs/ai-cache-breakpoints/spec.md) — the advisory contract · [doc 0002](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) — AI-24's charter and both closing checklists · [`proposal.md`](../../proposal.md)

---

## ADDED Requirements

## Purpose

AI-24.1 and AI-24.2 are `[decision]` leaves. They ship no Go, so there is no runtime behavior to specify. **The subject of this spec is the artifact** — the recorded first-provider and transport choice. Every requirement below constrains that document, and every scenario is a property a reviewer can check by inspection, deterministically, without running anything. This follows the model AI-01, AI-02 and AI-03 established.

Three distinctions shape the requirements and are stated once here.

**The argument is specified, not only the conclusion.** Eight milestones — AI-25 … AI-32 — build against this decision, and at least one later milestone will reverse or extend part of it. A list of verdicts with no reasons cannot be audited, extended or reversed on grounds anyone can check. So several requirements below constrain *reasons*: the seven-axis comparison must be total, the losses of the choice must be priced in the same document as the win, and every framing claim must carry its source.

**A discharged gate and an absent gate are different facts.** The ADR gate of `openspec/AGENTS.md` rule 5 resolves to a no-op under this choice, because the chosen transport adds no dependency. A gate that resolves to nothing and is therefore never mentioned is indistinguishable from a gate nobody applied. `R-APD-012` exists to make that distinction inspectable.

**A decision node this artifact does not own must survive it.** The evidence points strongly at one answer for AI-29.0, and an artifact that records the indicated answer as settled would delete a downstream node. `R-APD-010` makes pre-emption a defect.

Requirement count: **19**.

## Requirement ownership by node

| Node | Requirements |
| --- | --- |
| AI-24.1 — the provider decision | `R-APD-001` … `R-APD-010` |
| AI-24.2 — the transport decision | `R-APD-011` … `R-APD-015` |
| Both, plus the living-graph obligations | `R-APD-016` … `R-APD-019` |

## Definitions used by this spec

- **The decision artifact** — `openspec/changes/cachicamas-ai-first-provider-decision/decision.md`.
- **The closing checklists** — AI-24.1's four items and AI-24.2's three items in doc 0002.
- **The seven charter axes** — capability fit, streaming quality, testability, dependency weight, endpoint configurability, maintenance, and the credential-handling boundary, as doc 0002's AI-24 charter deliverable names them.
- **A candidate** — the chosen vendor or any alternative the artifact records as rejected.
- **The closed lists** — `CAP-R-01 … CAP-R-05` and `CAP-O-01 … CAP-O-03`, owned by the [v1 capability set](../../../../specs/ai-minimum-capabilities/spec.md).
- **An expected outcome** — a prediction the artifact records for one capability, before any adapter exists, against which AI-38.2 asserts the generated record.
- **A spec-mandated framing claim** — a claim about the streaming framing that the WHATWG HTML Living Standard § 9.2 states.
- **A dialect-conventional framing claim** — a claim about the streaming framing that § 9.2 does not state and that rests on the chosen vendor's documented or observed behavior.
- **An amendment** — a dated blockquote under a touched heading of doc 0002, per that document's revert-and-record clause rule 4.

---

## R-APD-001 — The artifact exists, is singular, and answers all seven checklist items

The change MUST produce exactly one decision artifact. It MUST answer every item of AI-24.1's four-item and AI-24.2's three-item closing checklists. No other file of this change MAY restate a decision as normative.

### Scenarios

- **S-APD-001** — Given the change directory, when a reviewer lists its files, then exactly one decision artifact is present AND every other artifact of the change refers to it rather than restating a decision as normative.
- **S-APD-002** — Given the artifact, when a reviewer walks the seven closing-checklist items in order, then each resolves to a passage that states an answer and its basis, and no item resolves to a forward reference alone.

---

## R-APD-002 — One vendor is named, and the comparison is total over the seven axes

The artifact MUST name exactly one vendor dialect as the first adapter's target. It MUST record every rejected alternative it considered. The comparison MUST be total over the product of the seven charter axes and the recorded candidates: every axis MUST carry a stated entry for every candidate.

An empty or omitted entry MUST be treated as a defect rather than as an implied equivalence between candidates.

### Scenarios

- **S-APD-003** — Given the artifact, when a reviewer looks for the first adapter's target, then exactly one vendor dialect is named, unambiguously, and no second candidate is left co-equal.
- **S-APD-004** — Given the comparison, when a reviewer forms the product of the seven charter axes and every recorded candidate, then every cell of that product carries a stated entry.
- **S-APD-005** — Given the comparison, when a reviewer removes any single axis from it, then at least one candidate loses a stated basis for its verdict — that is, no axis is decorative, and a missing axis is detectable by inspection rather than by inference.
- **S-APD-006** — Given each rejected alternative, when a reviewer reads its row, then the reason for rejection is stated on at least one named axis rather than asserted as a summary verdict.

---

## R-APD-003 — The argument is grounded, and the losses of the choice are priced in the same artifact

Each axis entry MUST rest on a property of the candidate that a reader can check against a cited source or an in-repo fact, rather than on an unsupported assertion.

The artifact MUST record what the chosen candidate costs. Specifically, it MUST state which already-shipped contract items lose their first exercising consumer under this choice, and MUST state that those items remain contract-mandatory regardless.

### Scenarios

- **S-APD-007** — Given any axis entry, when a reviewer asks on what it rests, then a cited vendor document, a cited spec clause, or a stated in-repo mechanical fact is present.
- **S-APD-008** — Given the artifact, when a reviewer looks for the cost of the choice, then the reasoning round-trip token's signature-preservation path and the breakpoint cap's adapter-side enforcement are each named as losing their first exercising consumer, with the reason.
- **S-APD-009** — Given each priced loss, when a reviewer asks whether the affected contract item is thereby deprecated, then the artifact states that it is not, and that its neutral shape remains contract-mandatory independently of who emits it.

---

## R-APD-004 — The four cross-provider divergences are each answered, with the node each drives

The artifact MUST answer, explicitly and for the named vendor, all four divergences AI-24.1 item 2 enumerates:

1. how the vendor expresses cache breakpoints, **or** that it caches automatically;
2. whether a tool result is a block inside a user-role message, a distinct role, or a nested object;
3. whether an explicit output-token limit is mandatory;
4. whether the vendor assigns tool-call identifiers at all.

Each answer MUST name the downstream node it drives and the consequence for that node.

### Scenarios

- **S-APD-010** — Given the artifact, when a reviewer looks for each of the four divergences in turn, then each carries an explicit answer for the named vendor, and none is left as "not applicable" without a stated reason.
- **S-APD-011** — Given the caching answer, when a reviewer reads it, then it states either the vendor's breakpoint expression or that caching is automatic, AND names the consequence for the request-translation node and for the expected outcome of `CAP-O-03`.
- **S-APD-012** — Given the tool-call-identifier answer, when a reviewer reads it, then it states whether the vendor assigns identifiers, AND names what follows for the node that owns synthetic identifier minting.
- **S-APD-013** — Given the output-token-limit answer, when a reviewer reads it, then it states whether an explicit limit is mandatory, AND, if it is not, records the affected downstream branch as a deliberate no-op rather than leaving it unmentioned.

---

## R-APD-005 — The two further questions are answered, with their downstream weighting

The artifact MUST answer both questions doc 0002 adds at AI-24.1 item 3: whether the vendor streams tool-call arguments in fragments or whole, and whether it signs reasoning blocks. Each answer MUST state the consequence for the node it drives.

### Scenarios

- **S-APD-014** — Given the artifact, when a reviewer reads the tool-call-argument answer, then it states fragmented or whole delivery for the named vendor, AND states which case the tool-call-stream node should weight as primary and which as the edge case.
- **S-APD-015** — Given the artifact, when a reviewer reads the reasoning-signature answer, then it states whether the dialect signs reasoning blocks, AND distinguishes a signed block from any opaque count the vendor reports about reasoning.

---

## R-APD-006 — The expected capability report is total over both closed lists

The artifact MUST record an expected capability report carrying **one entry per capability across both closed lists** — `CAP-R-01 … CAP-R-05` and `CAP-O-01 … CAP-O-03`, eight entries. A capability with no entry MUST be treated as a defect in the record, per the capability contract's § 10 totality rule, not as an absence.

Each entry MUST carry the capability identifier, its standing taken from the capability contract, and one expected outcome.

### Scenarios

- **S-APD-016** — Given the expected report, when a reviewer enumerates its entries against both closed lists, then every identifier in both lists has exactly one entry and no entry names an identifier outside them.
- **S-APD-017** — Given any single entry, when a reviewer reads its parts, then the capability identifier, the standing and the expected outcome are each present and separately identified.
- **S-APD-018** — Given the artifact, when a reviewer asks where an entry's standing comes from, then it is stated to come from the capability contract and never from this decision or from a run.

---

## R-APD-007 — Only `satisfied` and `absent` are legitimate expectations

Every expected outcome in the report MUST be drawn from `satisfied` or `absent`. The artifact MUST NOT record `failed` or `not exercised` as an expectation for any capability, because both are run results rather than predictions.

The artifact MUST NOT record `absent` for a required capability, because `absent` is legal for optional capabilities only.

### Scenarios

- **S-APD-019** — Given the expected report, when a reviewer scans every expected outcome, then each is `satisfied` or `absent`, and an entry predicting `failed` or `not exercised` is a defect.
- **S-APD-020** — Given a required-capability entry, when a reviewer reads its expected outcome, then it is `satisfied`, and an expected `absent` on a required capability is a defect.
- **S-APD-021** — Given the artifact, when a reviewer asks why the two run-only values are excluded from a prediction, then the reason is stated as the capability contract's distinction between a conclusion and the absence of a conclusion, not as a stylistic preference.

---

## R-APD-008 — The five required capabilities are confirmed cleared as a floor on vendor selection

The artifact MUST confirm, capability by capability, that the named vendor clears all five required capabilities, and MUST state that a candidate failing any one of them cannot be the first adapter.

Where clearing a required capability depends on an adapter-side request option rather than on the vendor's default behavior, the artifact MUST record that dependency as an implementation obligation on the named downstream nodes, because a silently unset option would produce an expected-illegal outcome that hides an adapter defect as a vendor limitation.

### Scenarios

- **S-APD-022** — Given the artifact, when a reviewer walks `CAP-R-01` through `CAP-R-05`, then each carries a stated basis on which the named vendor clears it.
- **S-APD-023** — Given the artifact, when a reviewer asks what would follow from a candidate failing one required capability, then it states that the candidate cannot be the first adapter, citing the capability contract's floor clause.
- **S-APD-024** — Given a required capability whose clearance depends on an adapter-side request option, when a reviewer reads its entry, then the option is named, the failure mode of leaving it unset is stated, and the obligation is assigned to at least one named downstream node.

---

## R-APD-009 — Every expected outcome carries its basis

Each entry of the expected report MUST carry the basis on which its outcome is predicted, so that AI-38.2's entry-by-entry comparison against the generated record can classify any difference — in either direction — as a finding rather than as a discrepancy of unknown origin.

### Scenarios

- **S-APD-025** — Given each entry of the expected report, when a reviewer asks why that outcome is expected, then a stated basis is present.
- **S-APD-026** — Given the artifact, when a reviewer reads what a difference between the expected and the generated record means, then it states that an unexpected `absent` and an unexpected `satisfied` are both findings.
- **S-APD-027** — Given an entry whose basis is not yet confirmed against the exact backend a later milestone will use, when a reviewer reads it, then the entry is marked as pending that confirmation rather than presented as settled.

---

## R-APD-010 — The artifact does not pre-empt a downstream decision node

The artifact MUST NOT resolve a question doc 0002 assigns to a downstream `[decision]` node, and MUST NOT propose deleting such a node on the ground that the evidence indicates its answer.

Where the evidence strongly indicates one answer, the artifact MUST record the indication as an input to that node, MUST name the node as the owner, and MUST record at least one concrete reason the indication could fail against the exact backend that node will judge.

### Scenarios

- **S-APD-028** — Given the artifact, when a reviewer looks for a ruling on whether the first adapter emits reasoning or records a capability absence, then no ruling is made and AI-29.0 is named as the owner.
- **S-APD-029** — Given the reasoning indication, when a reviewer reads it, then it is stated as indicated rather than decided, AND it names the concrete case that could overturn it — that some servers sharing the chosen dialect emit a non-standard reasoning field that is not part of the shared dialect.
- **S-APD-030** — Given the artifact's amendment proposals, when a reviewer inspects them, then none deletes or strikes a downstream `[decision]` node, and any note attached to one is explicitly labelled a note rather than a verdict.

---

## R-APD-011 — The transport is decided with evidence, and the dependency fact is mechanical

The artifact MUST decide the transport between the standard library's HTTP client and a vendor SDK, with evidence stated on the charter axes rather than as a preference. It MUST state the resulting dependency consequence for the agent module as a mechanically checkable fact rather than as a claim.

### Scenarios

- **S-APD-031** — Given the artifact, when a reviewer reads the transport decision, then one transport is named, and each rejected alternative carries a stated reason on at least one charter axis.
- **S-APD-032** — Given the dependency consequence, when a reviewer asks how to check it, then the artifact states a fact a reviewer can verify by command against the module file, not only in prose.
- **S-APD-033** — Given the artifact, when a reviewer reads the effect on the forward import guard, then it states whether that guard's allowlist gains an entry under this decision, and records the outcome either way.

---

## R-APD-012 — The ADR gate is recorded as discharged, even when it resolves to a no-op

The artifact MUST record the outcome of the ADR gate that doc 0002's AI-24 acceptance clause and `openspec/AGENTS.md` rule 5 impose. Where the chosen transport adds no dependency and the gate therefore fires no obligation, the artifact MUST record the gate as **evaluated and resolved to a no-op**, with the fact that discharges it.

Silence about the gate MUST be treated as a defect, because a gate that resolved to nothing and a gate nobody applied are indistinguishable from an artifact that mentions neither.

### Scenarios

- **S-APD-034** — Given the artifact, when a reviewer searches for the ADR gate, then it is named, its outcome is stated, and the fact that discharges it is stated with it.
- **S-APD-035** — Given a no-op outcome, when a reviewer reads it, then the artifact states that no ADR is required *because* the gate was applied and fired nothing — not that no ADR was thought about.
- **S-APD-036** — Given the artifact, when a reviewer asks which milestone the gate next binds, then a named milestone is stated, so the gate is transferred rather than closed.

---

## R-APD-013 — The streaming framing is named precisely enough to be encoded as fixtures

The artifact MUST name the streaming framing precisely enough that the framing-decoder milestone can encode fixtures from it without consulting the vendor. At minimum it MUST state: the response content type; the field grammar, including how a field name and its value are separated and what leading whitespace is stripped; how multiple data fields of one event combine and what separator joins them; the disposition of comment lines; the disposition of the identifier and reconnection-time fields; the treatment of a leading byte-order mark; the accepted line terminators; and the terminal-termination convention.

### Scenarios

- **S-APD-037** — Given the artifact, when a reviewer attempts to write a byte-level fixture of a complete stream from it alone, then every listed framing element has a stated answer and none requires consulting the vendor's documentation.
- **S-APD-038** — Given the multi-line data rule, when a reviewer reads it, then both the joining separator and the disposition of the buffer's own trailing separator at dispatch are stated.
- **S-APD-039** — Given a framing element the chosen dialect never emits in practice, when a reviewer reads its entry, then the artifact still states the decoder's required behavior for it and states that the decoder is specified independently of one vendor's habits.

---

## R-APD-014 — Spec-mandated and dialect-conventional framing claims are separately labelled

Every framing claim in the artifact MUST carry a label identifying it as spec-mandated or dialect-conventional. A claim labelled spec-mandated MUST be one the WHATWG HTML Living Standard § 9.2 states.

The terminal sentinel convention and the data-only shape — a stream carrying no event-type lines — MUST both be labelled dialect-conventional, because § 9.2 states neither. The artifact MUST state that both are to be pinned by explicit fixture at the framing-decoder milestone and MUST NOT be inferred from the specification.

Attributing a dialect convention to the specification MUST be treated as a defect, because a decoder built on that attribution conforms to something no specification states and breaks against a compliant server.

### Scenarios

- **S-APD-040** — Given every framing claim, when a reviewer reads its label, then each is labelled spec-mandated or dialect-conventional, and no claim is unlabelled.
- **S-APD-041** — Given the terminal sentinel and the data-only shape, when a reviewer reads their labels, then both are dialect-conventional and neither is attributed to § 9.2.
- **S-APD-042** — Given each dialect-conventional claim, when a reviewer reads it, then the artifact states the explicit-fixture obligation it places on the framing-decoder milestone.
- **S-APD-043** — Given any claim labelled spec-mandated, when a reviewer checks it against § 9.2, then § 9.2 states it; a spec-mandated label on a claim § 9.2 does not state is a defect.

---

## R-APD-015 — The credential-handling boundary is stated as an enforceable subject

The artifact MUST state the credential-handling boundary in three parts: what the adapter **receives**, what it **never reads**, and **where the value originates**.

What the adapter never reads MUST be enumerated as observable behaviors — not as an intention — so that the ambient-authority guard downstream has a subject it can enforce mechanically. The origin MUST be attributed to a layer outside Layer 1.

### Scenarios

- **S-APD-044** — Given the artifact, when a reviewer reads the credential boundary, then all three parts are present and separately stated.
- **S-APD-045** — Given what the adapter receives, when a reviewer reads it, then it is an opaque secret value supplied at construction, explicitly not a variable name and explicitly not a path.
- **S-APD-046** — Given what the adapter never reads, when a reviewer reads it, then at minimum reading an environment variable, opening a file, and spawning a process are each named as forbidden behaviors of the adapter's own source.
- **S-APD-047** — Given the origin, when a reviewer reads it, then it is attributed to the composition root of a layer above Layer 1, and the artifact states that the origin's own mechanism is out of Layer 1's scope.

---

## R-APD-016 — Living-graph amendments are dated, visible, and never silent

Every change this decision forces on doc 0002 MUST land as an amendment: a dated blockquote under the touched heading, stating what changed, which node changed it, and why, with superseded text struck through and left visible. All amendments MUST land in the same pull request as the decision artifact. Editing doc 0002 without a dated amendment block MUST be treated as a defect.

The amendment block MUST cover, at minimum:

| Target | What the amendment records |
| --- | --- |
| AI-24's charter, on the module's dependency state | The ADR gate resolved to a no-op, and which later milestone the zero-dependency state now holds until |
| **AI-25.2 item 1** | The stated guard mechanism is corrected: the referenced forward guard is an import-path scan, and the chosen transport transitively imports the very package the guard would deny, so that mechanism cannot bite. The corrected mechanism is a call-site scan over the adapter's own source files |
| AI-26.5 item 2 | Synthetic tool-call-identifier minting has no subject for this adapter; the requirement text stays for a future adapter and this adapter's conformance marks the branch not-applicable rather than skipping it silently |
| AI-26.2 item 3 | The vendor-cap branch has no subject; the drop-whole branch is the one this adapter takes, exercising the advisory contract |
| AI-26.7 item 2 | The mandatory-output-limit branch is a deliberate no-op, recorded as such |
| AI-29.0 | A note only, per `R-APD-010` |

### Scenarios

- **S-APD-048** — Given the doc 0002 diff, when a reviewer inspects every changed heading, then each carries a dated blockquote naming what changed, which node changed it, and why, and no heading changed without one.
- **S-APD-049** — Given any superseded claim, when a reviewer reads it, then the old text is struck through and left visible rather than deleted.
- **S-APD-050** — Given the AI-25.2 amendment, when a reviewer reads it, then it names the mechanism mismatch concretely: that the cited forward guard scans import paths, that the chosen transport transitively imports the denied package, and that the consequence is a guard that either false-positives on legitimate use or misses a narrow ambient-authority call.
- **S-APD-051** — Given the AI-25.2 amendment, when a reviewer asks what mechanism replaces the corrected one, then a call-site scan scoped to the adapter's own source files is named, so the downstream node inherits a mechanism rather than only a warning.
- **S-APD-052** — Given each of the AI-26.5, AI-26.2 and AI-26.7 amendments, when a reviewer reads it, then the branch that loses its subject is marked not-applicable or deliberate-no-op with its reason, and the requirement text it belongs to is stated to remain in force for a future adapter.
- **S-APD-053** — Given every amendment, when a reviewer checks the date, then it reads **2026-08-03**, and every amendment lands in the same pull request as the decision artifact.

---

## R-APD-017 — The Wave-2 carryovers are assigned to an appended, edged milestone

The artifact MUST record the two unassigned Wave-2 carryovers — the emission-boundary checker's fourth-rule failure-path coverage gap, and the missing redacting debug-rendering method on the provider-failure payload — and MUST assign them to a newly appended milestone rather than leaving them unassigned for a third wave.

The appended milestone MUST take the next free milestone ordinal, MUST carry one leaf per carryover, MUST declare its blocking edge to the milestone whose scope would otherwise overlap it, and MUST state the wave it is scheduled into with the reason.

Retrofitting either carryover into an archived milestone MUST NOT be proposed.

### Scenarios

- **S-APD-054** — Given the amendment block, when a reviewer looks for the carryovers, then a milestone `AI-41` is appended with two leaves, one per carryover.
- **S-APD-055** — Given the appended milestone, when a reviewer reads its edges, then `Blocks: AI-36` is declared, with the reason stated as the scope overlap that would otherwise leave one behavior owned by two nodes.
- **S-APD-056** — Given the appended milestone, when a reviewer asks why it is not scheduled in the current wave, then the artifact states that neither carryover blocks any node of the current wave and states the review-load reason.
- **S-APD-057** — Given the artifact, when a reviewer checks whether any archived milestone gains a retrofitted node identifier, then none does, consistent with the append-only identifier rule.

---

## R-APD-018 — Every blocked milestone receives a statement of what it inherits

The artifact MUST state, for each milestone this decision blocks — AI-25 through AI-32 — what that milestone receives from this decision, written in that milestone's own terms.

Each statement MUST be sufficient for that milestone to be planned without reopening this decision. Where a milestone receives an obligation rather than only a fact, the obligation MUST be stated as such.

### Scenarios

- **S-APD-058** — Given the artifact, when a reviewer looks for each of AI-25 … AI-32 in turn, then each has an inheritance statement written in that milestone's own terms.
- **S-APD-059** — Given any inheritance statement, when a reviewer asks whether that milestone could be planned from it without reopening this decision, then the statement supplies the named vendor dialect, transport, framing or credential boundary the milestone needs, rather than deferring back to this decision.
- **S-APD-060** — Given a milestone that receives an implementation obligation, when a reviewer reads its statement, then the obligation is labelled as one and its failure mode is stated.

---

## R-APD-019 — Scope, authoring constraint, and artifact hygiene

This change MUST NOT decide anything owned by AI-25 through AI-32, AI-38 or AI-41, and MUST NOT modify any requirement of any existing capability spec; existing specs are cited by identifier only.

Per doc 0002's `[decision]` node grammar and its authoring constraint, the artifact MUST NOT declare or name any Go type, field, method, interface or package identifier belonging to the Layer 1 contract. Naming the standard library package that constitutes the chosen transport, the module manifest, and the vendor's wire-level field names is permitted, because both closing checklists require them.

The change MUST add only markdown, MUST add nothing under `backend/`, and MUST modify no build, module or infrastructure file.

### Scenarios

- **S-APD-061** — Given the artifact, when a reviewer looks for an adapter declaration, a fixture, a decoder rule, or a generated capability report, then none is present and the owning milestone is named in each case.
- **S-APD-062** — Given the change diff, when a reviewer inspects it, then no requirement of any file under `openspec/specs/` is added, modified, removed or renamed; only a prose status line recording that the carryovers are now assigned may change.
- **S-APD-063** — Given the artifact, when a reviewer scans for a Layer 1 contract identifier — a single-token camel-case name, a method-shaped name, or a Layer 1 package path — then none is found, and every Layer 1 noun is a noun phrase with spaces or a cited `V-*` or `CAP-*` identifier.
- **S-APD-064** — Given the change diff, when a reviewer inspects the file list, then it contains only markdown, adds nothing under `backend/`, and modifies no build, module or infrastructure file.

---

## Acceptance criteria

The decision holds when:

1. `R-APD-001` through `R-APD-019` hold, each verified by its scenarios.
2. All four items of AI-24.1's closing checklist and all three of AI-24.2's are answered in the decision artifact, with a verification recorded item by item.
3. The expected capability report is total over both closed lists, with every expected outcome drawn only from `satisfied` and `absent`.
4. The ADR gate's outcome is recorded, and its no-op resolution is stated as discharged rather than omitted.
5. Every framing claim carries a spec-mandated or dialect-conventional label, and the two dialect conventions carry their fixture-pin obligation.
6. No downstream `[decision]` node is pre-empted, struck or deleted by this artifact.
7. Every doc 0002 change is a dated amendment, and `AI-41` is appended with its two leaves and its `Blocks: AI-36` edge.
8. No Go identifier of the Layer 1 contract appears anywhere in the change, and no file under `backend/` is touched.
