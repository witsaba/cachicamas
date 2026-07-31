# Spec — Layer 1 stream lifecycle, ownership and carrier

> **Change**: `cachicamas-ai-stream-lifecycle`
> **Milestone**: AI-02 · **Node**: AI-02.1 `[decision]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-stream-lifecycle/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-AIS-0NN` · **Scenario IDs**: `S-AIS-0NN`
> **Binding input**: [AI-01's register](../../../cachicamas-ai-contract-vocabulary/decision.md)

---

## Purpose

AI-02.1 is a `[decision]` leaf. It ships no Go, so there is no runtime behavior to specify. The subject of this spec is the **artifact**: `decision.md`, the recorded stream lifecycle decision. Every requirement below constrains that document, and every scenario is a property a reviewer can check against it by inspection, deterministically, without running anything.

The distinction is stated once here so no later phase mistakes it: a scenario reads "given the decision artifact, when …, then …". The artifact is the system under test.

A second distinction matters as much. Several requirements below constrain the **argument**, not only the conclusion — that the strongest opposing case is present, that a rejected alternative is named, that a number carries its falsification criteria. doc 0002 requires the SDD to "record why it chose what it chose"; a conclusion with no argument does not satisfy that, and would pass a spec that only checked conclusions.

---

## Definitions used by this spec

- **The decision artifact** — `openspec/changes/cachicamas-ai-stream-lifecycle/decision.md`.
- **The closing checklist** — AI-02.1's five items in doc 0002.
- **The register** — AI-01's vocabulary artifact.
- **A register citation** — a `V-*` identifier appearing in the decision artifact.
- **The package contract** — the prose obligations the decision requires AI-20 to publish alongside the provider interface.
- **An inheritance statement** — a sentence in the decision artifact naming a downstream milestone and what that milestone receives from this decision.

---

## R-AIS-001 — The artifact exists, is singular, and answers all five items

The change MUST produce exactly one decision artifact, at `openspec/changes/cachicamas-ai-stream-lifecycle/decision.md`. It MUST answer every item of AI-02.1's closing checklist. No other file in this change MAY restate a decision as normative.

### Scenarios

- **S-AIS-001** — Given the change directory, when a reviewer lists its files, then exactly one file named `decision.md` is present AND every other artifact of the change refers to it rather than restating a decision as normative.
- **S-AIS-002** — Given the decision artifact, when a reviewer walks AI-02.1's five closing-checklist items in order, then each resolves to a section of the artifact that states a decision, its rationale, and its consequences.

---

## R-AIS-002 — The carrier decision is argued, and rests on no cost-of-change argument

The artifact MUST decide the carrier and MUST record the rationale. It MUST present the strongest case for the rejected option before rejecting it. It MUST NOT rest any part of the carrier rationale on the cost of invalidating existing work, because doc 0002 records that no such work exists and declares that argument void.

### Scenarios

- **S-AIS-003** — Given the artifact's carrier section, when a reviewer reads it, then a case **for** the range-over-func iterator is stated affirmatively and at length before any rebuttal, AND that case includes at minimum: the structural elimination of the stranded-producer hazard, abandonment becoming a supported operation, and the dissolution of the buffering question.
- **S-AIS-004** — Given the artifact, when a reviewer searches the carrier rationale for any appeal to shipped guards, merged scenarios, existing signatures, or migration cost, then none is found as a supporting ground; the only permitted mention is the explicit statement that this argument is void.
- **S-AIS-005** — Given each ground on which the iterator case is defeated, when a reviewer checks it, then it cites a specific source — doc 0001, doc 0002, doc 0003, or a register row — rather than asserting a preference.
- **S-AIS-006** — Given the artifact, when a reviewer looks for a third option, then "offer both carriers at the boundary" is named and rejected with its costs enumerated.

---

## R-AIS-003 — The carrier decision states its delegation and its graph consequence

Whichever carrier is chosen, the artifact MUST state the consequence doc 0002 attaches to that branch.

If the receive-only channel is chosen, the artifact MUST state that the iterator-ergonomics requirement is delegated to AI-22.5 and MUST state that doc 0002's waves 2–5 therefore require no amendment nodes. If the iterator is chosen, the artifact MUST instead state that doc 0002's waves 2–5 gain amendment nodes under the living-graph clause, and MUST name AI-22.5's inversion.

### Scenarios

- **S-AIS-007** — Given the artifact's carrier section, when a reviewer reads its consequences, then exactly one of the two branches above is stated, matching the decision taken, AND the other is not left implied.
- **S-AIS-008** — Given the channel branch, when a reviewer reads the delegation, then it states that the delegated view is a convenience and never a second contract, AND it names AI-22.5's pin as the mechanical form of that claim.

---

## R-AIS-004 — Ownership is stated structurally

The artifact MUST state what "exactly once" binds, in structural terms a reader can check against a producer: how many goroutines may send, how many closing sites may exist, and when the closing site runs relative to the last send attempt.

### Scenarios

- **S-AIS-009** — Given the artifact's ownership section, when a reviewer reads it, then it states that exactly one goroutine sends on a stream AND that exactly one closing site exists in the producer AND that the closing site runs after the last send attempt and never before.
- **S-AIS-010** — Given an adapter that internally reads a transport on one goroutine and translates on another, when a reviewer consults the artifact, then the artifact states that internal fan-in happens below the boundary and that exactly one goroutine ever sends on the carrier.

---

## R-AIS-005 — "Exactly once" is stated for all three paths, not implied

The artifact MUST state the closing obligation separately for the completion path, the terminal-error path and the cancellation path. It MUST NOT state the obligation once for the completion path and leave the other two to be inferred.

### Scenarios

- **S-AIS-011** — Given the artifact, when a reviewer looks for the closing obligation, then the completion path, the terminal-error path and the cancellation path are each named explicitly and each carries its own statement of what is emitted and when the close happens.
- **S-AIS-012** — Given an unwinding exit from the producer, when a reviewer consults the artifact, then the close is stated to run on that path too.

---

## R-AIS-006 — Nothing but the producer closes

The artifact MUST state that the consumer never closes the stream, and MUST enumerate the other parties that also never close it.

### Scenarios

- **S-AIS-013** — Given the artifact, when a reviewer reads the ownership section, then the consumer, the test kit, and any consumer above Layer 1 are each named as parties that do not close the stream, consistent with `V-STR-04` and doc 0001 § 9.

---

## R-AIS-007 — Cancellation obligations are stated and each is classified as testable or statable

The artifact MUST state that the caller owns a cancellable context, that every send waits on both the stream and cancellation, and that cancellation closes the stream within bounded time. For each obligation the artifact MUST say whether it is provable by test and, where it is, name the node that proves it.

### Scenarios

- **S-AIS-014** — Given the artifact's cancellation section, when a reviewer reads each of the three obligations, then each carries either a naming of the node that proves it or an explicit statement that it is not provable by test.
- **S-AIS-015** — Given the bounded-close obligation, when a reviewer reads what "bounded" excludes, then the artifact states that after cancellation is observable the producer begins no new blocking wait on the network or on the consumer, and that a backoff waits on the signal rather than sleeping.

---

## R-AIS-008 — Abandonment is a contract violation, stated in the package contract

The artifact MUST state that a consumer which stops reading and never cancels is a documented contract violation rather than a supported mode. It MUST state that this sentence belongs in the package contract, MUST state why — it cannot be tested to termination — and MUST name AI-40.3 as its restatement at the v1 freeze.

### Scenarios

- **S-AIS-016** — Given the artifact, when a reviewer reads the abandonment clause, then it is stated as a violation AND it is assigned to the package contract AND the reason given is that no test can prove a goroutine never exits AND AI-40.3 is named.
- **S-AIS-017** — Given the artifact, when a reviewer asks what a consumer *may* legally do to end a stream early, then the complete set of legal endings is enumerated, so that "violation" has a complement.
- **S-AIS-018** — Given AI-20.1's third test item — that the interface documentation states who closes the stream, who owns the context, and what abandoning without cancelling means — when a reviewer checks the artifact, then all three of those statements are present and quotable.

---

## R-AIS-009 — The buffer is bounded and carries a decided starting capacity

The artifact MUST state that the buffer is bounded. It MUST choose one starting capacity, expressed as a number. It MUST justify that number against the alternatives at both ends of the range, and MUST state that concurrency multiplies the cost.

### Scenarios

- **S-AIS-019** — Given the artifact's buffering section, when a reviewer looks for the capacity, then exactly one number is stated as the starting capacity AND it is not expressed as a range, a preference, or a deferral.
- **S-AIS-020** — Given the chosen number, when a reviewer reads its justification, then the justification states what a materially smaller capacity would cost and what a materially larger one would cost.
- **S-AIS-021** — Given concurrent streams, when a reviewer consults the artifact, then it states that capacity is paid per live stream and names the concurrency sources that make this real.

---

## R-AIS-010 — The capacity is falsifiable and AI-34 inherits the criteria

The artifact MUST present the capacity as a starting hypothesis, MUST state what measurement would confirm or change it, and MUST state which direction each result implies. It MUST NOT decide whether the capacity is a constant or configurable — that is AI-34.1's.

### Scenarios

- **S-AIS-022** — Given the artifact, when a reviewer reads the buffering section, then it names the measurements AI-34.1 should take and states, for each, what result moves the number up and what result moves it down.
- **S-AIS-023** — Given the artifact, when a reviewer looks for a decision on constant versus configurable, then the artifact explicitly defers it to AI-34.1 rather than deciding it.

---

## R-AIS-011 — Backpressure is lossless and exactly one loss path is sanctioned

The artifact MUST state that a full buffer makes the producer wait and never drop. It MUST state the single sanctioned loss path in full — cancellation with a saturated buffer drops late events and closes without a terminal event — MUST state that this is the only such path, and MUST name AI-20.3 as where it is proven.

### Scenarios

- **S-AIS-024** — Given the artifact, when a reviewer reads the backpressure posture, then waiting is stated as the behavior of a full buffer AND dropping is stated as not being that behavior.
- **S-AIS-025** — Given the sanctioned loss path, when a reviewer reads it, then all three of its elements are present — cancellation, saturation, and closing without a terminal event — AND the artifact states that a consumer treating a missing terminal after its own cancellation as corruption is the party in error.
- **S-AIS-026** — Given a stream that closes without a terminal event and was never cancelled, when a reviewer consults the artifact, then that case is stated to be a producer defect, not a second loss path.

---

## R-AIS-012 — The two delivery paths are separated at an observable moment

The artifact MUST state what a caller observes when the request never becomes a stream (`V-FAIL-11`) and what a caller observes when a stream dies mid-flight (`V-FAIL-12`). It MUST identify the moment that separates them as an observable event, MUST name and reject the alternative boundary, and MUST state that the delivery axis is orthogonal to the partial-output discriminator (`V-FAIL-09`).

### Scenarios

- **S-AIS-027** — Given the artifact, when a reviewer reads the pre-stream path, then it states that the failure is returned directly AND that no stream and no producer are created — not an empty stream, not a stream that immediately yields a failure.
- **S-AIS-028** — Given the artifact, when a reviewer reads the mid-stream path, then it states that the failure arrives as the terminal error event and that the stream then closes, AND that no second route exists by which a caller learns of a mid-stream failure.
- **S-AIS-029** — Given a stream that is handed to the caller and then fails before emitting any content, when a reviewer consults the artifact, then that case is classified as mid-stream delivery, AND the artifact states that classifying it as pre-stream is the conflation with `V-FAIL-09` that doc 0001 § 7 **G8** records.
- **S-AIS-030** — Given the artifact, when a reviewer looks for the rejected boundary, then "the first event" is named as the rejected alternative with the reason it fails.
- **S-AIS-031** — Given AI-19.5's third test item — a caller that only inspects the returned failure and a caller that only inspects the terminal event can each classify every failure — when a reviewer checks the artifact, then the artifact states the property that makes this satisfiable: the same category, retryability and partial-output discriminator are reachable on both paths.

---

## R-AIS-013 — The artifact stays inside its own scope

The artifact MUST NOT decide anything owned by AI-14, AI-19, AI-20, AI-22.4, AI-34 or AI-35. AI-01 § 9 rule 5's test applies: if a sentence were deleted, would a later milestone have more options? If yes, and that milestone is not AI-02, the sentence does not belong.

### Scenarios

- **S-AIS-032** — Given the artifact, when a reviewer looks for an event kind, an event payload, a sequence rule or an ordering invariant, then none is decided; the artifact constrains the container and defers contents to AI-14.
- **S-AIS-033** — Given the artifact, when a reviewer looks for a failure category, a retryability rule, or the shape of a terminal error payload, then none is decided; the artifact decides delivery only and defers classification to AI-19.
- **S-AIS-034** — Given the artifact, when a reviewer looks for a leak-detection mechanism, then none is chosen; AI-22.4 is named as its owner.
- **S-AIS-035** — Given the artifact, when a reviewer looks for a retry rule, then none is stated; AI-35 is named as its owner.

---

## R-AIS-014 — Vocabulary discipline, inheritance, and artifact hygiene

Every Layer 1 noun used by the artifact MUST resolve to a register row. A noun the register lacks MUST be appended to the register in this same pull request, with the next free ordinal in its category and a dated amendment blockquote, and MUST NOT be defined locally. The artifact MUST state what each blocked milestone inherits. No Go type, field, method, interface or package identifier MAY appear in any file of this change.

### Scenarios

- **S-AIS-036** — Given the artifact, when a reviewer collects every Layer 1 noun it uses in a normative sentence, then each resolves to exactly one register row, cited by identifier.
- **S-AIS-037** — Given a noun the register lacked, when a reviewer inspects the change, then the term appears as a new row in AI-01's artifact with the next free ordinal in its category, under a dated amendment blockquote, AND no definition of it appears in this change's own artifacts other than as a citation.
- **S-AIS-038** — Given the register amendment, when a reviewer diffs AI-01's artifact, then no existing row is renumbered, reworded or removed, AND the register's own term counts are updated to remain consistent.
- **S-AIS-039** — Given the artifact, when a reviewer reads its closing section, then AI-14, AI-20, AI-21 and AI-22 each have an inheritance statement in that milestone's own terms, AND AI-34 has a stated starting point.
- **S-AIS-040** — Given every file of this change, when a reviewer scans for a single-token camel-case name, a package path, or a method-shaped name, then none is found; every term is a noun phrase with spaces, and language and standard-library shapes are named descriptively.
- **S-AIS-041** — Given the change's diff, when a reviewer inspects it, then it contains only markdown under `openspec/changes/`, adds nothing under `backend/`, and modifies no build, module or infrastructure file.
