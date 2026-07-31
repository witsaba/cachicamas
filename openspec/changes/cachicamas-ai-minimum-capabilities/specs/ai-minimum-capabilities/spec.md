# Spec — the v1 capability set and optional-capability discovery

> **Change**: `cachicamas-ai-minimum-capabilities`
> **Milestone**: AI-03 · **Node**: AI-03.1 `[decision]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-minimum-capabilities/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-AIC-0NN` · **Scenario IDs**: `S-AIC-0NN`
> **Binding inputs**: [AI-01's register](../../../cachicamas-ai-contract-vocabulary/decision.md) · [AI-02's decision](../../../cachicamas-ai-stream-lifecycle/decision.md)

---

## Purpose

AI-03.1 is a `[decision]` leaf. It ships no Go, so there is no runtime behavior to specify. The subject of this spec is the **artifact**: `decision.md`, the recorded capability matrix and discovery mechanism. Every requirement below constrains that document, and every scenario is a property a reviewer can check against it by inspection, deterministically, without running anything.

Two distinctions shape the requirements and are stated once here.

**The argument is specified, not only the conclusion.** A capability list is consumed by milestones that will need to *extend* it — AI-24 records an expectation against it, AI-23 marks cases against it, and some later milestone will propose a fourth optional capability. A list of verdicts with no reasons cannot be extended consistently. So several requirements below constrain reasons: every optional capability carries the reason it is optional rather than required, every required capability carries what it does **not** require, and every exclusion carries its reason.

**Absence of a rule is a defect this spec must catch.** doc 0002's acceptance criterion depends on rules that its own checklist does not name — how an unclassified conformance case is marked, and what happens to an optional capability across a wrapper. `R-AIC-009` and `R-AIC-011` exist because a decision artifact that answered only the five checklist items would still leave AI-23 unable to satisfy its criterion.

---

## Definitions used by this spec

- **The decision artifact** — `openspec/changes/cachicamas-ai-minimum-capabilities/decision.md`.
- **The closing checklist** — AI-03.1's five items in doc 0002.
- **The register** — AI-01's vocabulary artifact.
- **A register citation** — a `V-*` identifier appearing in the decision artifact.
- **A capability identifier** — an identifier the decision artifact assigns to one entry of one of its three lists, so that a downstream milestone can cite an entry rather than paraphrase it.
- **The required list / the optional list / the excluded list** — the three enumerations the closing checklist requires.
- **A standing** — required or optional, as assigned by the decision artifact.
- **An outcome value** — one member of the closed set the capability record's entries may carry.
- **An inheritance statement** — a sentence naming a downstream milestone and what it receives from this decision.

---

## R-AIC-001 — The artifact exists, is singular, and answers all five items

The change MUST produce exactly one decision artifact, at `openspec/changes/cachicamas-ai-minimum-capabilities/decision.md`. It MUST answer every item of AI-03.1's closing checklist. No other file in this change MAY restate a decision as normative.

### Scenarios

- **S-AIC-001** — Given the change directory, when a reviewer lists its files, then exactly one file named `decision.md` is present AND every other artifact of the change refers to it rather than restating a decision as normative.
- **S-AIC-002** — Given the decision artifact, when a reviewer walks AI-03.1's five closing-checklist items in order, then each resolves to a section of the artifact that states a decision and its rationale.

---

## R-AIC-002 — Classification is governed by stated tests, not by assertion

The artifact MUST state the tests by which a candidate behavior is classified as not-a-capability, required, optional or excluded. Every entry in every list MUST be justified by reference to those tests, and every candidate the artifact considers and rejects MUST name the test clause it fails.

### Scenarios

- **S-AIC-003** — Given the artifact, when a reviewer looks for the basis of a classification, then a stated admission test is present, applying to all three lists, AND it is expressed so that a later milestone proposing a new entry can apply it without re-deriving it.
- **S-AIC-004** — Given each entry of the required and optional lists, when a reviewer checks its justification, then the justification names the test clauses it satisfies rather than asserting the verdict.
- **S-AIC-005** — Given the artifact, when a reviewer looks for candidates that were considered and not admitted, then at least one such candidate is recorded for the optional list, each with the reason it was not admitted.

---

## R-AIC-003 — A capability is distinguished from the three things it is not

The artifact MUST state what a capability is, and MUST distinguish it from a contract obligation binding every adapter identically, from an adapter-local mapping obligation, and from a contract property that is optional for every adapter.

### Scenarios

- **S-AIC-006** — Given the artifact, when a reviewer reads the definition of the unit being classified, then all three near-neighbours are named and each is given an example drawn from a source document.
- **S-AIC-007** — Given tool-call deltas, which doc 0001 § 3.3 row 1 makes optional, when a reviewer consults the artifact, then they are classified as a contract property optional for **every** adapter and explicitly **not** as an optional capability, with the reason that no consumer may require a delta from any provider.

---

## R-AIC-004 — The required capabilities are enumerated, and each states what it does not require

The artifact MUST enumerate the required capabilities, covering at minimum streaming text, tool calls, completion metadata (finish reason and usage), cancellation, and typed failures with the partial-output distinction. Each MUST carry an explicit statement of what it does **not** oblige, where a careless reading would oblige an adapter to fabricate behavior. Each MUST be citable by a stable identifier.

### Scenarios

- **S-AIC-008** — Given the artifact, when a reviewer reads the required list, then all five named capabilities are present, each with an identifier, a statement of what it obliges, and a statement of what it does not.
- **S-AIC-009** — Given the completion-metadata capability, when a reviewer reads its precision, then the artifact states that requiring the usage record does **not** require any particular token count to be populated, citing the register rows that make an absent count legal and distinguishable from zero.
- **S-AIC-010** — Given the completion-metadata capability, when a reviewer reads its precision, then the artifact states that requiring the finish-reason vocabulary to be reachable does **not** require every value in it to be emitted by every provider, and that an unrecognised provider stop condition maps to the unknown value as a conformant outcome.
- **S-AIC-011** — Given the streaming-text and tool-call capabilities, when a reviewer reads their precisions, then a block delivered whole with zero deltas is stated to be conformant, AND an adapter that mints identifiers for a vendor that assigns none is stated to be satisfying the required capability rather than lacking one.
- **S-AIC-012** — Given the cancellation and typed-failure capabilities, when a reviewer reads them, then their observable shapes are cited from AI-02's decision rather than re-decided.

---

## R-AIC-005 — Token counting is optional, and the argument is recorded in full

The artifact MUST classify token counting as an optional capability. It MUST record the argument, including the reading under which counting would be required and why that reading is insufficient. It MUST state that a required count would force an adapter without one either to fail conformance or to fabricate a figure, and MUST state why a fabricated count is worse than an absent one. The artifact MUST NOT rest this classification on convenience or on implementation effort.

### Scenarios

- **S-AIC-013** — Given the artifact, when a reviewer reads the token-counting entry, then the reason for optionality is stated as a consequence for a consumer — a fabricated count corrupts a compaction decision silently while an absent one degrades to a visible estimate — rather than as an appeal to adapter effort.
- **S-AIC-014** — Given the artifact, when a reviewer looks for the opposing reading, then the case that counting should be required is stated affirmatively — a consumer genuinely needs a real count — before it is defeated.
- **S-AIC-015** — Given the artifact, when a reviewer asks whether Layer 1 may supply a substitute for an absent capability, then the artifact states that it may not, and gives the reason as the same argument applied to Layer 1 itself.

---

## R-AIC-006 — Every optional capability carries the reason it is optional rather than required

The artifact MUST enumerate the optional capabilities, covering at minimum reasoning content, token counting, and honoring cache-boundary markers. Each entry MUST carry the reason it is optional rather than required. The artifact MUST state whether the list is closed, and MUST state how a new entry is admitted.

### Scenarios

- **S-AIC-016** — Given each entry of the optional list, when a reviewer reads it, then a reason for optionality is present, and it identifies either a real provider that lacks the capability or a legitimate adapter choice to omit it.
- **S-AIC-017** — Given each entry of the optional list, when a reviewer reads it, then the artifact also states what a consumer does on a recorded absence, so that optionality is shown to be survivable rather than merely permitted.
- **S-AIC-018** — Given the optional list, when a reviewer asks whether a fourth entry may be added, then the artifact states the route — an amendment in the pull request that needs it — and states the cost that makes speculative entries wrong.

---

## R-AIC-007 — The nine-row leakage register is cross-checked, row by row

The artifact MUST cross-check its optional list against doc 0001 § 3.3's nine provider-leakage rows, and MUST state for each row whether it yields a capability. Where a row is an adapter-local mapping obligation or a Layer 1 contract item rather than a capability, the artifact MUST say so.

### Scenarios

- **S-AIC-019** — Given the artifact, when a reviewer walks doc 0001 § 3.3's nine rows against it, then each row has a stated verdict, and the rows that are adapter-local mapping obligations are distinguished from the rows that are Layer 1 contract items.
- **S-AIC-020** — Given the artifact, when a reviewer reads the cross-check's conclusion, then it states that a documented provider divergence is evidence for an adapter's mapping table rather than for the optional list.
- **S-AIC-021** — Given the reasoning row and the caching row, when a reviewer reads them, then the artifact separates the neutral shape, which is a contract item, from the emitting or honoring behavior, which is the capability.

---

## R-AIC-008 — Honoring cache-boundary markers is classified with its argument

The artifact MUST state why honoring cache-boundary markers is a capability rather than an adapter-local mapping obligation, given that markers are advisory by contract and an adapter may ignore them wholesale while remaining conformant. It MUST state that expressing markers on a request is a Layer 1 contract item available against every provider, and is not itself optional.

### Scenarios

- **S-AIC-022** — Given the artifact, when a reviewer reads the marker entry, then the case for treating it as adapter-local is stated before it is answered.
- **S-AIC-023** — Given the artifact, when a reviewer reads the answer, then it identifies a consumer-visible consequence that differs between a provider that honors markers and one that does not.
- **S-AIC-024** — Given the artifact, when a reviewer asks whether cache-boundary markers themselves are optional, then the artifact states that they are not: the request-side expression is a contract item owned by another milestone, and only honoring is discovered.

---

## R-AIC-009 — The marking rule makes the acceptance criterion satisfiable

The artifact MUST state the rule by which a conformance case is marked required or optional. The rule MUST be total over cases — including a case that exercises no listed capability — and its default MUST be *required*.

### Scenarios

- **S-AIC-025** — Given a conformance case that exercises no capability named in either list — for example an ordering invariant or the pre-stream contract — when a reviewer applies the artifact's marking rule, then the case is marked required, unambiguously and without consulting any other document.
- **S-AIC-026** — Given the artifact, when a reviewer reads the marking rule, then it is stated as a biconditional over the optional list, AND the artifact states why the required list alone cannot serve as the marking source.

---

## R-AIC-010 — The discovery mechanism states both how an adapter advertises and how a consumer asks

The artifact MUST decide the discovery mechanism. It MUST state that an optional capability is an additional, separately-asserted contract on the provider value, and that the core provider interface never widens. It MUST state how an adapter advertises a capability and how an adapter without it declares nothing. It MUST state how a consumer asks and what it observes on absence. It MUST name and reject the alternatives.

### Scenarios

- **S-AIC-027** — Given the artifact, when a reviewer reads the discovery section, then both halves are present: an adapter advertises by satisfying the additional contract and by no other means, AND a consumer asks the provider value at the point of use.
- **S-AIC-028** — Given an adapter that lacks an optional capability, when a reviewer consults the artifact, then it states that the adapter declares nothing at all — no flag, no negative answer, no entry — and remains fully conformant.
- **S-AIC-029** — Given a consumer that asks for an absent capability, when a reviewer consults the artifact, then the observed result is stated to be a clean absence, explicitly not an error and explicitly not a zero.
- **S-AIC-030** — Given the artifact, when a reviewer looks for rejected alternatives, then at minimum a widened core contract, a declared capability list returned by the provider, a configuration- or catalog-driven capability table, and a single aggregate optional contract are each named and rejected with their reasons.
- **S-AIC-031** — Given the artifact, when a reviewer asks whether a provider may advertise a capability and then decline to honor it, then the artifact states that this is non-conformance rather than absence.
- **S-AIC-032** — Given the artifact, when a reviewer asks what the discovery question is asked *of*, then the artifact states the provider value, and states that a model identity, a configuration entry or a catalog is not an acceptable subject.
- **S-AIC-033** — Given the artifact, when a reviewer looks for the mechanical guarantee that the core contract does not widen, then AI-20.5's pin is named as its form.

---

## R-AIC-011 — Wrapping is addressed

The artifact MUST state what happens to an optional capability when a provider value is wrapped, and MUST state the obligation that applies to a wrapper. It MUST name the milestone that introduces the first wrapper.

### Scenarios

- **S-AIC-034** — Given a wrapper that satisfies the core provider contract and forwards no optional contract, when a reviewer consults the artifact, then it states that every optional capability of the wrapped value is removed AND that the removal is invisible by construction.
- **S-AIC-035** — Given the artifact, when a reviewer reads the wrapper obligation, then it states that a wrapper either forwards the optional contracts the wrapped value satisfies or documents which it removes, AND AI-37 is named as the first milestone the rule binds.

---

## R-AIC-012 — The capability record's shape is sketched and is total

The artifact MUST sketch the shape of the capability record AI-23.6 emits and AI-38.2 asserts. The record MUST carry one entry per capability in the decision's closed lists, required and optional alike. Each entry MUST carry the capability, its standing, and one outcome value. The standing MUST come from this decision rather than from the run.

### Scenarios

- **S-AIC-036** — Given the artifact, when a reviewer reads the record sketch, then it states that the record is total over the closed lists AND that a capability with no entry is a defect in the run rather than an absence.
- **S-AIC-037** — Given one record entry, when a reviewer reads its parts, then the capability, its standing and its outcome are each present and separately identified.
- **S-AIC-038** — Given the artifact, when a reviewer asks where an entry's standing comes from, then it is stated to come from this decision, so that a run cannot demote a required capability by recording it as optional.
- **S-AIC-039** — Given the artifact, when a reviewer asks what the record must not carry, then at minimum model content, credentials and raw provider text are excluded, consistent with the register's safe-metadata and redaction rows, because the record is published into package documentation.

---

## R-AIC-013 — "Absent" and "not exercised" are distinct outcome values

The artifact MUST define a closed set of outcome values, and that set MUST distinguish a recorded absence from a case that did not run. The artifact MUST state a verdict rule over a record, and under that rule a record containing any not-exercised entry MUST NOT be a pass.

### Scenarios

- **S-AIC-040** — Given the artifact, when a reviewer reads the outcome set, then it is closed and enumerated, and it contains distinct values for a recorded absence and for a case that did not run.
- **S-AIC-041** — Given a record in which one optional capability is recorded absent, when a reviewer applies the artifact's verdict rule, then the record is a pass and the provider is fully conformant.
- **S-AIC-042** — Given a record in which one entry is not exercised, when a reviewer applies the verdict rule, then the record is not a pass, and the artifact states that it is inconclusive rather than failing.
- **S-AIC-043** — Given the artifact, when a reviewer reads why the two values are distinct, then the reason is stated as the fact that a skipped case and a recorded absence are typographically identical in an ordinary test report while being opposite facts.
- **S-AIC-044** — Given an optional capability that is advertised and then fails its cases, when a reviewer applies the artifact's outcome set, then the outcome is failure rather than absence, consistent with the rule that advertising binds.

---

## R-AIC-014 — The exclusions are enumerated with their reasons, and the excluded/optional line is a rule

The artifact MUST enumerate what is excluded for v1, covering at minimum multimodal content beyond text, embeddings, batch APIs, and server-side tool execution, each with the reason it is excluded. It MUST state the rule that separates an exclusion from an optional capability.

### Scenarios

- **S-AIC-045** — Given the artifact, when a reviewer reads the excluded list, then all four named exclusions are present, each with an identifier and a reason.
- **S-AIC-046** — Given the multimodal exclusion, when a reviewer reads its reason, then it states that enabling it requires per-provider capability detection v1 does not model, and it is consistent with the register row that names image and audio in the content vocabulary with no producer.
- **S-AIC-047** — Given the server-side tool-execution exclusion, when a reviewer reads its reason, then the reason is principled rather than merely scope-based: it names the layer boundary it would violate and the protocol it would route around.
- **S-AIC-048** — Given the artifact, when a reviewer asks why an exclusion is not simply an optional capability that no provider offers, then the artifact states the distinguishing rule: an optional capability has a defined absence, while an excluded one has no defined presence.

---

## R-AIC-015 — Scope, vocabulary discipline, inheritance, and artifact hygiene

The artifact MUST NOT decide anything owned by AI-11, AI-13, AI-19, AI-20, AI-23, AI-24 or AI-29. Every Layer 1 noun used in a normative sentence MUST resolve to a register row, cited by identifier. A noun the register lacks MUST be appended to the register in this same pull request, with the next free ordinal in its category and a dated amendment blockquote, and MUST NOT be defined locally. The artifact MUST state what each blocked milestone inherits. No Go type, field, method, interface or package identifier MAY appear in any file of this change.

### Scenarios

- **S-AIC-049** — Given the artifact, when a reviewer looks for a failure category, a finish-reason value's definition, a marker cap number, or the declaration of any contract, then none is decided, and the owning milestone is named in each case.
- **S-AIC-050** — Given the artifact, when a reviewer looks for a statement about which optional capabilities the first vendor supports, then none is made, and AI-24.1 is named as the owner of that record.
- **S-AIC-051** — Given the artifact, when a reviewer looks for a decision on whether the first adapter emits reasoning, then none is made, and AI-29.0 is named, with this decision's contribution stated as making both answers legal.
- **S-AIC-052** — Given the artifact, when a reviewer collects every Layer 1 noun it uses in a normative sentence, then each resolves to exactly one register row, cited by identifier.
- **S-AIC-053** — Given a noun the register lacked, when a reviewer inspects the change, then the term appears as a new row in AI-01's artifact with the next free ordinal in its category, under a dated amendment blockquote, AND no definition of it appears in this change's own artifacts other than as a citation.
- **S-AIC-054** — Given the register amendment, when a reviewer diffs AI-01's artifact, then no existing row is renumbered, reworded or removed, AND the register's own term counts are updated to remain consistent.
- **S-AIC-055** — Given each appended register row, when a reviewer reads its definition, then it settles the word without deciding this milestone's substance — consistent with AI-01 § 9 rule 5 — and defers the substance to AI-03 by name.
- **S-AIC-056** — Given the artifact, when a reviewer reads its closing section, then AI-20.5, AI-23 and AI-24 each have an inheritance statement in that milestone's own terms.
- **S-AIC-057** — Given the artifact, when a reviewer looks for the distinction between an absent optional capability and the unsupported-capability failure category, then both are named and separated, with the failure stated to arise at request time and the absence stated to be discoverable before any request.
- **S-AIC-058** — Given every file of this change, when a reviewer scans for a single-token camel-case name, a package path, or a method-shaped name, then none is found; every term is a noun phrase with spaces, and language mechanisms are named descriptively.
- **S-AIC-059** — Given the change's diff, when a reviewer inspects it, then it contains only markdown under `openspec/changes/`, adds nothing under `backend/`, and modifies no build, module or infrastructure file.
