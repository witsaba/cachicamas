# Design — Layer 1 contract vocabulary

> **Change**: `cachicamas-ai-contract-vocabulary`
> **Milestone**: AI-01 · **Node**: AI-01.1 `[decision]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-contract-vocabulary/spec.md`
> **Output**: the structure and rules that `decision.md` implements
> **Diagrams**: ASCII (project convention — no mermaid in existing proposal/spec artifacts)

---

## 1. What is being designed

Not software. The artifact. This document answers three questions:

1. **How is the vocabulary structured**, such that a reviewer can verify `S-AIV-001` … `S-AIV-031` by inspection?
2. **Why does "one definition + one owning milestone per term" prevent a class of argument**, rather than merely documenting nouns?
3. **Where is the line** between a noun this milestone settles and a decision a later milestone owns?

---

## 2. The argument class this artifact targets

### 2.1 Naming the failure precisely

The arguments AI-01 exists to prevent are not disagreements about behavior. They are disagreements in which **two people use one word for two concepts and neither notices**. That failure has a recognizable signature, and it produced every one of doc 0001's four contract defects:

```
   Milestone M                            Milestone N (later)
   -----------                            -------------------
   needs concept X                        needs concept X'
   picks word "W"                         reads word "W"
   ships a contract using "W"             assumes it means X'
        |                                       |
        |                                       v
        |                            builds against "W" == X'
        |                                       |
        +---------------- collision ------------+
                              |
                              v
              a contract that contradicts its own documentation
```

Applied to the record:

| Defect | W | X (what M meant) | X' (what N assumed) |
| --- | --- | --- | --- |
| **C1** | content part | a discriminated wrapper | any value satisfying the part contract, including one that skipped construction |
| **C2** | content part | an opaque wrapper exposing only its discriminator | a value whose payload an adapter can read from another package |
| **C3** | sequence | a monotonic counter in the package | a per-stream, 1-based, contiguous ordinal |
| **C4** | terminal error event | a declared, mandatory event kind | an event an adapter can actually construct |

In every row, the collision was resolvable at zero cost *before* either milestone started, and cost a corrective milestone afterwards.

### 2.2 Why one definition is the countermeasure

A word with two definitions is a word that carries no constraint. Neither party is wrong; the contract is simply not written down. The countermeasure is not more documentation of behavior — the retired plan documented its behavior fully, and its documentation is what the code contradicted. The countermeasure is to make the word itself carry the constraint:

- **C1 and C2 dissolve** when `content part` is defined as *a value that cannot exist without passing its construction rules **and** whose payload is readable from outside the package that owns it*. The two properties are constitutive, in one sentence, in one place. AI-06.1 then cannot choose a strategy that satisfies one and not the other, because such a strategy fails the definition before it fails a test. doc 0002's own note on AI-06 says exactly this: "Any strategy that satisfies one and not the other has failed this milestone."
- **C3 dissolves** when `stream` is defined before `sequence`, and `sequence` is defined as *a property of a stream*. A process-global counter is then not a defensible implementation of the definition; it is a different concept wearing the same word.
- **C4 dissolves** when `terminal error event` is defined as *the event that ends a stream that failed, constructible by an adapter in another package*. An interface cannot then declare it mandatory before it exists, because "mandatory but unconstructible" contradicts the noun.

### 2.3 Why one owning milestone is the second half

One definition stops the collision. One **owner** stops the renegotiation.

Without a named owner, a term is owned by whoever needs it most urgently — which, in a plan executed milestone by milestone, is always the milestone currently under implementation pressure. That is the mechanism by which a definition drifts: not by a decision, but by a small convenient adjustment made inside a pull request whose reviewer is looking at code.

Naming exactly one owning milestone converts term drift from an invisible edit into a visible cross-milestone change:

```
  without an owner column                 with an owner column
  -----------------------                 --------------------
  AI-14 PR quietly narrows                AI-14 PR needs "delta" narrowed
  what "delta" means                              |
        |                                         v
        v                              amendment to V-STR-16 in the same PR
  AI-16, AI-17, AI-18 inherit                      |
  a definition nobody agreed to                    v
        |                                 reviewer sees AI-14 editing a term
        v                                 owned by AI-14 — legitimate;
  discovered at AI-30, costs a            or a term owned by AI-09 — a
  corrective milestone                    cross-milestone change, reviewed as one
```

The owner column also makes a subtler defect checkable: a term whose owner's charter does not actually cover it (`S-AIV-005`). That is a mis-assignment, and it is exactly the shape of C4 — a term whose owner (the provider interface) could not deliver it, while the milestone that could (the error taxonomy) came later.

### 2.4 The residual risk this design accepts

A register cannot stop a milestone from *needing* a concept the vocabulary lacks. It can only decide what happens next. The design's answer is `R-AIV-011`: the missing term is appended by amendment in the same pull request that needs it, with the next free ordinal, under a dated blockquote. This is doc 0002's living-graph clause applied to nouns instead of nodes, and it is deliberate — the clause exists because "implementation will disprove parts of this plan; that is expected and priced in."

What the design refuses is the alternative: inventing the term inside the milestone's own SDD and leaving the register stale. A stale register is worse than none, because it is trusted.

---

## 3. Structure of the register

### 3.1 Row shape

Every term is one row with five fields, in this order:

```
  V-<CAT>-nn  |  term name  |  definition  |  owning milestone  |  provenance
  ----------     ---------     ----------     -----------------     ----------
  stable id      conceptual    what it IS     exactly one AI-NN     doc 0001 §
  append-only    noun phrase   + what it      (current identifier,  + C1-C4 /
                 with spaces   is NOT         never retired)        G1-G13 where
                                                                    applicable
```

Four properties follow from the shape and each maps to a spec scenario:

- The identifier is **stable and append-only**, so amendment never renumbers (`R-AIV-004`).
- The term name is a **noun phrase with spaces**, which makes a leaked Go identifier visually obvious rather than something a reviewer must hunt for (`R-AIV-009`, and the reason `S-AIV-019` is a cheap check).
- The definition states **what the term is and what it is not**. The negative half is not decoration: C1, C2 and C3 were all failures of an unstated negative — "not a value that skipped construction", "not opaque to other packages", "not a process-wide counter".
- The owner is **exactly one current identifier**, single-valued, which is what makes double ownership visible by scanning one column (`S-AIV-004`).

### 3.2 The six categories

```
  +-----------------------------------------------------------------------+
  |  V-REQ   request-side       what goes to the model                    |
  |          owners AI-05 .. AI-12                       checklist item 1 |
  +-----------------------------------------------------------------------+
  |  V-STR   stream-side        what comes back, and what carries it      |
  |          owners AI-02, AI-09, AI-14 .. AI-18         checklist item 2 |
  +-----------------------------------------------------------------------+
  |  V-MET   metadata           what a response says about itself         |
  |          owner  AI-13                                checklist item 3 |
  +-----------------------------------------------------------------------+
  |  V-FAIL  failure            two vocabularies, two delivery paths      |
  |          owners AI-04, AI-19, AI-35, AI-36           checklist item 4 |
  +-----------------------------------------------------------------------+
  |  V-PRV   provider surface   the interface and the proving apparatus   |
  |          owners AI-03, AI-20 .. AI-24            beyond the checklist |
  +-----------------------------------------------------------------------+
  |  V-OUT   excluded           named, attributed, never defined          |
  |          owners Layer 2 / Layer 3 / composition root checklist item 5 |
  +-----------------------------------------------------------------------+
```

**Why `V-PRV` exists although the checklist does not name it.** AI-01's *acceptance* criterion is stronger than its checklist: "Every subsequent milestone's charter can be written using only these terms." AI-03's charter needs `capability`, `required capability`, `optional capability`, `capability discovery` and `capability record`; AI-21's needs `fake provider`; AI-22's needs `stream test kit`; AI-23's needs `conformance suite`. Omitting them would satisfy the checklist and fail the acceptance criterion, and the acceptance criterion is the normative one (doc 0002's charter rule: "The charter is the normative scope").

**Why `V-OUT` rows have no definition.** Defining a Layer 2 concept in a Layer 1 artifact is the first step toward Layer 1 implementing it. An exclusion row states the term, its owner, and — where the exclusion is easy to misread — the Layer 1 neighbour it is confused with. That third field is the one that does the work: `transcript` excluded is unhelpful; `transcript` excluded, owned by Layer 2's harness, adjacent to Layer 1's `message`, is a rule a reader can apply.

### 3.3 Definition order inside a category is load-bearing

Categories are not alphabetical. Within `V-STR`, the container nouns precede the content nouns:

```
  stream -> carrier -> producer/consumer -> ownership -> cancellation
      |
      +--> sequence          (a property OF a stream)
      +--> event -> kind -> payload
      +--> block -> block index -> delta
      +--> terminal event
```

This is C3 encoded as document order. `sequence` is defined *after* `stream` and *as a property of* it, so the definition itself excludes a process-global counter. Reversing the order re-admits the defect at the definitional level, where no test can catch it.

The same principle applies in `V-FAIL`: the two owners are separated before the two delivery paths, because the owner split answers *whose failure it is* and the delivery split answers *how the caller observes it*. Presenting delivery first invites the reading that pre-stream failures are AI-04's and mid-stream failures are AI-19's — which is wrong, and is precisely the confusion `R-AIV-006` forbids.

### 3.4 Cross-references instead of second definitions

Some concepts genuinely recur. `call ordinal` is created in the tool-call content contract (AI-09.2), asserted again on the stream (AI-18.3), and preserved by the adapter (AI-30.5). Three milestones touch it; one owns it.

The register handles this with a single row that names AI-09 as owner and lists the restatement nodes in its provenance. It does **not** create a second stream-side row. Recurrence is a fact about the plan; ownership is a fact about the vocabulary, and only the second one is this artifact's business.

Grouping follows the closing checklist even where ownership does not: the checklist places `call ordinal` among stream-side terms, so the row sits in `V-STR` with an AI-09 owner. Checklist grouping and ownership are different axes, and the row makes both explicit rather than forcing them to agree.

---

## 4. Provenance and the retired-identifier hazard

### 4.1 The hazard

doc 0001 is merged and normative for architecture, and it cites milestone identifiers that no longer mean what they say. Its § 3.2 assigns cache breakpoints to "AI-43"; § 3.1 assigns C4 to "AI-18". Under doc 0002's renumbering, cache breakpoints are AI-11 and C4 is closed by AI-19 — while **AI-18 still exists** as a different milestone (tool-call delta events). A copied citation is therefore not merely stale; it points at a real, wrong milestone.

This is the highest-impact risk in the change (proposal § Risks row 6), because a wrong owner propagates into every downstream charter that trusts the register.

### 4.2 The translation table this artifact applies

Taken from doc 0002's identifier map, restricted to identifiers doc 0001 actually cites:

| doc 0001 says | Concern | Current owner |
| --- | --- | --- |
| AI-40 | per-stream sequence (C3) | **AI-14** |
| AI-41 | content readable from another package (C2) | **AI-06** |
| AI-42 | content-part construction bypass (C1) | **AI-06** |
| AI-43 | cache breakpoints (G4) | **AI-11** |
| AI-44 | per-request options + escape hatch (G9) | **AI-12** |
| AI-45 | reasoning round-trip token (G12(b)) | **AI-07** |
| AI-46 | refusal and pause finish reasons (G12(c)) | **AI-13** |
| AI-47 | stream carrier (G13) | **AI-02** |
| AI-18 (in § 3.1, § 7 G8) | provider error taxonomy (C4) | **AI-19** |
| AI-16 (in ADR 0005 § D4) | interface signature guard | **AI-20.4** |
| AI-34, AI-35 (in ADR 0005 § D3) | redaction, observability | **AI-36**, **AI-37** |

`R-AIV-008` makes applying this table a requirement, and `S-AIV-018` makes it checkable: every `AI-NN` in `decision.md` must resolve against a doc 0002 milestone heading.

### 4.3 What provenance records

Two things, and deliberately not a third:

- **The doc 0001 section** the concept derives from — so a reader can find the architectural argument.
- **The `C1`–`C4` / `G1`–`G13` identifier**, where the term exists to close a specific recorded defect or gap — so the register is auditable against doc 0002's traceability spine.
- **Not line numbers.** They drift, and a drifted citation is worse than a coarse one (`NFR-AIV-B`).

---

## 5. The line between a noun and a decision

The sharpest risk in this milestone is over-reach: a vocabulary that quietly decides what a later milestone owns. The rule the artifact applies:

> **This milestone settles what a word means. It does not settle what the thing does.**

Worked, for the four milestones most at risk:

| Milestone | AI-01 settles (noun) | AI-NN settles (decision) |
| --- | --- | --- |
| **AI-02** | `carrier` means *the mechanism by which a consumer receives a stream's events at the package boundary* | **which** carrier — receive-only channel or range-over-func iterator. doc 0002 states the choice is genuinely free there and only there |
| **AI-02** | `bounded buffer`, `sanctioned loss path` mean what they say | the starting capacity, and the exact conditions of the one sanctioned loss |
| **AI-03** | `optional capability` means *a capability a conformant adapter may lack, whose absence is a recorded outcome rather than an error* | **which** capabilities are optional, and the discovery mechanism |
| **AI-04** | `caller-contract failure` means *an invalid request; the caller's bug; knowable without I/O* | granularity, aggregate-versus-short-circuit, and how positional context attaches |
| **AI-06** | `content part` carries readability and sealing as constitutive properties | **which strategy** delivers both simultaneously |

The test a reviewer applies: *if this sentence were deleted, would a later milestone have fewer options?* If yes, and the milestone that loses options is not AI-01, the sentence is over-reach.

One deliberate exception, and it is a definitional constraint rather than a design decision: for `content part`, the register states that readability and sealing are **both** constitutive. That removes from AI-06.1 the option of choosing a strategy that satisfies one and not the other — but doc 0002's AI-06 note already removes it ("Any strategy that satisfies one and not the other has failed this milestone"), so the register is recording an upstream constraint, not creating one.

---

## 6. Verification approach

There is nothing to execute. Verification is inspection, and the design makes each check cheap by construction:

| Check | Made cheap by |
| --- | --- |
| One definition per term (`S-AIV-002`) | Terms appear once; recurrence is a cross-reference in the provenance field |
| One owner per term (`S-AIV-004`) | Owner is a single-valued column; scanning it is a single pass |
| No retired identifiers (`S-AIV-018`) | Every identifier is translated at authoring time via § 4.2's table |
| No Go identifiers (`S-AIV-019`) | Term names are noun phrases with spaces; a single-token camel-case name is visually anomalous |
| Checklist coverage (`S-AIV-008`) | Categories are ordered to match closing-checklist items 1–5 |
| Traps verbatim (`S-AIV-029`) | Quoted as a block quotation, character for character, from doc 0002's *Layer boundary* |
| Downstream expressibility (`S-AIV-026`, `S-AIV-027`) | AI-02's and AI-03's charters are walked noun by noun as the last task of the milestone |

No `make test` gate applies. Per doc 0002's evidence gate, a `[decision]` leaf closes when "the decision artifact answers every listed question and is merged"; the recorded green-output requirement is scoped to behavior and guard leaves, and there is no code in `backend/agent/` for this change to exercise.

---

## 7. Amendment protocol

Inherited from doc 0002's living-graph clause, restated here as the operating procedure for the register.

1. A milestone discovers a Layer 1 noun the register does not carry, carries wrongly, or attributes to the wrong owner.
2. **Do not** define it locally in that milestone's SDD.
3. Append or correct the row in `decision.md`, taking the next free ordinal in its category. Never renumber, never reuse.
4. Record the change as a dated blockquote (`> **Amended YYYY-MM-DD** …`) under the touched category heading. Strike through superseded definition text; leave it visible.
5. Land the amendment **in the same pull request** that needs it, so the register is never behind the plan.
6. If the correction changes ownership, the pull request is a cross-milestone change and is reviewed as one.

A wholesale restructuring — for example, if AI-06.1's chosen strategy collapses `content part` and `content-part kind` into one concept — is not an amendment. It is a new decision node appended under AI-01 (`AI-01.2`) with an edge recorded in doc 0002, leaving `decision.md` in place with its superseded sections struck through so merged citations keep resolving. That is proposal § Rollback level 3.

---

## 8. Acceptance criteria for the design phase

1. The structure in § 3 is sufficient to satisfy every requirement in `spec.md` — each requirement maps to a structural property, not to reviewer diligence.
2. The argument in § 2 explains the mechanism, not just the outcome: why a word with one definition and one owner cannot drift silently.
3. The line in § 5 is stated as a test a reviewer can apply, and the one deliberate exception is justified from an upstream constraint.
4. The retired-identifier hazard (§ 4) is closed with a concrete translation table, not a warning.
5. Nothing in this document names a Go identifier, chooses a carrier, chooses a capability set, or chooses a content-part strategy.

## 9. Next phase

`sdd-tasks` — one phase, one node (AI-01.1), the closing checklist as the task list, plus the verification pass of § 6.
