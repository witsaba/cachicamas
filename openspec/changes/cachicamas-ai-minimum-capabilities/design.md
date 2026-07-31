# Design — the v1 capability set and optional-capability discovery

> **Change**: `cachicamas-ai-minimum-capabilities`
> **Milestone**: AI-03 · **Node**: AI-03.1 `[decision]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-minimum-capabilities/spec.md`
> **Output**: the structure and the reasoning rules that `decision.md` implements
> **Diagrams**: ASCII (project convention — the sibling AI-01 and AI-02 changes use no mermaid in their proposal, spec or design artifacts)

---

## 1. What is being designed

Not a capability. A **list that other milestones will compute against**, which is a materially different design problem from AI-01's (a register consulted for meanings) and AI-02's (a decision consulted for physics).

Three consumers, and each treats the artifact as data rather than prose:

| Consumer | What it computes | What breaks if the list is imprecise |
| --- | --- | --- |
| **AI-23** | Marks every conformance case required or optional | A case falls through unmarked and is skipped silently — the exact failure `V-PRV-09` exists to prevent |
| **AI-24.1** | Records the expected outcome for the chosen vendor, before any adapter exists | The expectation cannot be written, so AI-38.2 has nothing to assert against |
| **AI-38.2** | Emits a record and asserts it against that expectation | The generated record and the expectation are not comparable |

So three properties follow, and they are not the ones a prose decision usually optimises for:

- **The lists must be closed and citable.** A downstream milestone must be able to iterate the entries and cite one, which means identifiers. AI-02 needed none — its five decisions are cited by section. AI-03's entries are cited individually by three milestones, so they get identifiers (`CAP-R-nn`, `CAP-O-nn`, `CAP-X-nn`).
- **Totality must be designed in, not asserted.** A record can only be total over a closed list. Closing the lists is therefore not tidiness; it is the precondition for the property item 5 requires.
- **Every classification must carry the reason.** Not for the reader's satisfaction but because at least one downstream milestone will propose a new entry, and a list of verdicts cannot be extended consistently. The reason is the extension mechanism.

## 2. The three failure modes this design targets

### 2.1 The requirement that forces a fabrication

The obvious risk is a *missing* required capability. The expensive one is a **present** required capability that some honest adapter cannot satisfy. It has two effects, and both are worse than the omission it was meant to prevent:

1. The adapter fails conformance for a reason that is not a defect, so the required list becomes a filter on which vendors may be adapted rather than a definition of correctness.
2. Or the adapter satisfies the requirement by fabricating the behavior — and a fabricated answer is indistinguishable from a real one at every layer above.

doc 0002 names the second effect for token counting ("implement it or lie") but the same defect hides in two places inside the required list where nobody is looking for it: requiring a *populated* token count inside usage, and requiring every finish-reason value to be *emitted*. Both would force an adapter to invent data, and both are one careless sentence away.

The countermeasure is a rule about the shape of a required entry:

> **The fabrication test.** Every required capability states what it does **not** oblige. If a plausible reading of the obligation would make an honest adapter invent data it does not have, the negative clause that excludes that reading is mandatory, not optional.

`R-AIC-004`, `S-AIC-009` and `S-AIC-010` make it checkable.

### 2.2 The optional list that absorbs the leakage register

doc 0001 § 3.3 is nine documented ways providers differ. Read quickly, every row is a candidate optional capability — and an optional list of nine is nine suite cases, nine record entries and nine recorded expectations for capabilities no consumer asked for.

The countermeasure is to run the register through the admission tests **in the artifact**, row by row, and to record the finding as a rule rather than a table. The finding is strong enough to be a rule: nine documented divergences yield zero optional capabilities on their own, and the two rows that have a capability adjacent to them (reasoning, caching) have it on the emitting or honoring half, never on the neutral shape.

> **The divergence rule.** A documented provider divergence is evidence for an adapter's mapping table, not for the optional list. Promoting one requires showing that the difference is consumer-visible — that a consumer would behave differently knowing it.

### 2.3 The silently skipped case wearing the clothes of a recorded absence

This is `V-PRV-09`'s whole content, restated four times across doc 0002 because it is genuinely hard to see. In an ordinary test report, "we did not run this because the provider does not offer it" and "we did not run this because the harness errored" render identically: a case with no result.

The countermeasure is not a warning. It is a data structure:

> **The distinct-values rule.** A recorded absence and an unrun case are different values of the same closed outcome set, and the verdict rule treats them differently — one is a pass, the other makes the record inconclusive. A report cannot express them with the same mark, because the mark is a value, not a formatting choice.

## 3. Structure of `decision.md`

```
  §1  How to use this document          ← who reads it, for what
  §2  What was decided                  ← the matrix, before any argument
  §3  What a capability is              ← the unit, and the three things it is not
  §4  The admission tests               ← the method, stated before it is applied
  §5  Required capabilities             ← checklist item 1
  §6  Optional capabilities             ← checklist item 2
  §7  Excluded for v1                   ← checklist item 3
  §8  The nine-row leakage cross-check  ← the guard on §6, and the divergence rule
  §9  The discovery mechanism           ← checklist item 4
  §10 The capability record             ← checklist item 5
  §11 The marking rule                  ← the acceptance criterion's real requirement
  §12 What each blocked milestone inherits
  §13 Standing rules
  §14 Closing-checklist verification
```

Sections 5 … 10 are the closing checklist in its own order, so a reviewer can walk doc 0002 and the artifact in parallel — the same property AI-02's structure has. Four sections sit outside that spine and each earns its place:

- **§ 3 and § 4 come first** because they are the method. AI-02 could put its argument inside each decision because its five decisions were independent; AI-03's three lists are produced by **one** classification process, and stating it three times inside three sections would be three chances to state it differently.
- **§ 8 sits after the three lists** because it is a check on them, not an input. Placing it before § 6 would read as a source of entries, which is precisely the misreading § 2.2 targets.
- **§ 11 sits after the mechanism** because the marking rule is what makes doc 0002's acceptance criterion satisfiable, and a reviewer checking that criterion should find it in one place rather than inferring it from the lists.

### 3.1 The shape of a capability entry

Every entry in § 5 and § 6 carries the same four parts:

| Part | Content | Why it is mandatory |
| --- | --- | --- |
| **Identifier and name** | `CAP-R-nn` / `CAP-O-nn`, plus a noun phrase | Three downstream milestones cite entries individually |
| **What it obliges** | The behavior, in register terms, cited by identifier | The suite tests this |
| **What it does not oblige** | The negative clause | § 2.1's fabrication test. Required entries always; optional entries where a reading would over-claim |
| **Why this standing** | The admission-test clauses it satisfies | The extension mechanism. Item 2 requires it explicitly for optional entries; this design requires it for required ones too |

Excluded entries in § 7 carry three parts: identifier and name, the reason for exclusion, and — where the exclusion is easy to mistake for an optional capability — the clause of the excluded/optional rule it fails.

Uniformity here is the same argument AI-02 made for its five-part decision sections, with one addition: a uniform entry shape is what lets a reviewer check the list *mechanically*, which is what a list consumed as data needs.

### 3.2 Where the argument's weight goes

```
   contested                                                     settled
   |------------------------------------------------------------|
   token counting    marker honoring   exclusions   discovery    cancellation
   (the load-        (genuinely        (sourced,    (mechanism   + typed failures
    bearing case;     arguable         but the      family is    (AI-02 already
    doc 0002          both ways;       optional/    inherited;   fixed the shapes;
    predicts the      §7's rule        excluded     the five     this milestone
    reopening)        decides it)      line is      sub-questions only assigns
                                       new)         are free)    the standing)
```

Argument length tracks that gradient. Token counting gets the full treatment including the opposing reading at its strongest, because doc 0002 predicts the reopening in its own charter note. Cancellation gets three sentences and two citations, because arguing a settled thing at length invites a reader to think it is unsettled.

### 3.3 Why the required list is not the marking source

```
        everything a conformant adapter must do  (V-PRV-04 + V-PRV-05 + the five)
        ┌──────────────────────────────────────────────────────────────┐
        │  ordering invariants · ownership · pre-stream contract ·      │
        │  sequence · redaction · translation totality · …              │
        │                                                              │
        │        ┌────────────────────────────────────┐                │
        │        │  the five required capabilities    │  ← capability-  │
        │        │  CAP-R-01 … CAP-R-05               │    shaped only  │
        │        └────────────────────────────────────┘                │
        └──────────────────────────────────────────────────────────────┘

        ┌────────────────────────────────┐
        │  CAP-O-01 … CAP-O-03           │  ← the only cases that are NOT required
        └────────────────────────────────┘
```

The picture is the argument. The required list is a **subset** of what is required, so marking by membership in it would leave most required cases unmarked. The optional list is **exactly** the set of non-required cases, so marking by membership in it is total. Hence the biconditional, with *required* as the default — and the default direction matters: an unclassified case that defaults to required fails loudly, while one that defaults to optional is skipped silently, which is the failure mode of § 2.3 relocated from the record to the suite.

### 3.4 Why the record is designed before it is described

Item 5 asks for a *sketch*, and a sketch invites prose. Prose cannot express the property the item is actually about. So § 10 designs three things and describes none:

```
  record
    ├── subject          which provider, which run   → comparability
    └── entries          ONE PER CAPABILITY IN THE CLOSED LIST → totality
          ├── capability       identifier from §5 / §6
          ├── standing         required | optional   ← from THIS decision, not the run
          └── outcome          satisfied | absent | failed | not exercised
                                          ▲                        ▲
                                          │                        │
                                a conclusion            the ABSENCE of a conclusion
                                (pass, conformant)      (record is inconclusive)
```

Three properties, three structural choices:

1. **Totality** ← one entry per capability in a closed list. A missing entry is a defect, not an absence.
2. **Standing integrity** ← the standing is copied from this decision, not derived from the run. A run cannot demote a required capability by recording it optional, which is what AI-23.6 item 2 needs.
3. **Absence is a conclusion** ← *absent* and *not exercised* are distinct values, and the verdict rule reads them differently.

The record deliberately carries **no** capability-specific detail — not the marker cap, not a counting method, not timings. Two reasons: the record answers "what did this run observe about this capability", and any additional field is a second thing to keep comparable across AI-24.1's expectation and AI-38.2's generated record; and AI-40.2 publishes it into package documentation, where the register's safe-metadata and redaction posture (`V-FAIL-13`, `V-FAIL-14`) binds anything published.

### 3.5 Why the mechanism's asymmetry gets its own treatment

The chosen mechanism has one property that drives everything else: **an adapter without a capability writes nothing.** No flag, no negative answer, no registration.

That asymmetry is the mechanism's central virtue — absence costs zero lines, so no adapter is ever tempted to fabricate — and it is the source of both sharp edges, which is why the design keeps them beside it rather than in a footnote:

```
  absence is silent ──┬──► virtue:  an adapter never has to declare a lack,
                      │             so the optional list is cheap to grow
                      │
                      ├──► edge 1:  an advertised capability that declines
                      │             looks like a widened contract wearing
                      │             a disguise           → advertising binds
                      │
                      └──► edge 2:  a wrapper that forwards nothing removes
                                    capabilities invisibly → forwarding rule
```

Edge 2 appears in no source document. It is a consequence of the mechanism, the first wrapper is AI-37, and stating it costs two sentences today against a Layer 2 investigation later.

## 4. Reasoning rules the artifact applies

Five, each with the failure it prevents.

1. **The admission tests** (`explore.md` § 4, restated in the artifact's § 4) — prevents a list nobody can extend consistently.
2. **The fabrication test** (§ 2.1) — prevents a required capability that forces an honest adapter to invent data.
3. **The divergence rule** (§ 2.2) — prevents the optional list absorbing the leakage register.
4. **The distinct-values rule** (§ 2.3) — prevents a skipped case reading as a conformance pass.
5. **The deletion test for scope.** AI-01 § 9 rule 5, applied verbatim: if a sentence were deleted, would a later milestone have more options? If yes and that milestone is not AI-03, cut it. Prevents over-reach into AI-19, AI-20, AI-23, AI-24 and AI-29.

Rule 5 has teeth here in a way it did not for AI-02, because this milestone's blocked milestones include two decision nodes — AI-24.1 and AI-29.0 — whose whole content is a choice this decision could accidentally make for them. AI-29.0 is the sharper case: it decides whether the first adapter emits reasoning, and this decision's only legitimate contribution is *making both answers legal*. A sentence recommending either answer removes AI-29.0's reason to exist.

## 5. The line between deciding a standing and deciding a behavior

AI-01 settled words. AI-02 settled the stream's physics. AI-03 needs the next line along:

> **This milestone decides which behaviors are required, which are optional, and how optionality is observed. It decides no behavior's content, and it judges no provider.**

Worked cases, because the line is easier to state than to apply:

| Question | Whose? | Why |
| --- | --- | --- |
| Must an adapter emit a finish reason? | **AI-03** | A standing |
| Which finish reasons exist, and what each means? | **AI-13** | Content |
| Must an adapter be cancellable? | **AI-03** | A standing |
| What does cancellation oblige of a producer? | **AI-02** | Behavior, already decided; cited, not restated |
| Is token counting optional? | **AI-03** | A standing, and the milestone's central one |
| What does a consumer do when it is absent? | **Layer 2** (`V-OUT-06`) | Consumer policy. Layer 1 states the absence and supplies nothing |
| How is an optional capability discovered? | **AI-03** | The observation mechanism for optionality |
| How is the optional contract declared? | **AI-20** | A declaration |
| Which optional capabilities does the first vendor have? | **AI-24.1** | A judgement about a provider, and no provider is chosen |
| Does the first adapter emit reasoning? | **AI-29.0** | A judgement about an adapter. This decision only makes both answers legal |
| What is in the capability record? | **AI-03** | The shape is the mechanical form of `V-PRV-09` |
| What format is the record emitted in? | **AI-23.6** / **AI-40.2** | Emission and publication |

## 6. Handling of the register amendment

`explore.md` § 11 found three nouns the register lacks. AI-01 § 9 rule 2 governs and AI-02 set the procedure; this change follows it exactly:

1. The rows are appended to AI-01's artifact — not defined here. This change's artifacts cite them by identifier only.
2. Ordinals are the next free ones in the `V-PRV` category. No existing identifier moves.
3. A dated amendment blockquote is placed under the category heading, per the convention.
4. The register's own term counts are updated, because an artifact that contradicts its own arithmetic invites the reader to distrust the rest.
5. The owner of all three rows is AI-03.
6. **Each definition defers its substance to AI-03 by name**, exactly as `V-PRV-08` defers the discovery mechanism. This is the clause that keeps the amendment inside AI-01 § 9 rule 5: the register settles words, and a register row that stated which outcome values exist, or that token counting is optional, would be AI-01 deciding AI-03's matrix retroactively.

One of the three is different in kind from the other two and the difference is worth recording: `V-PRV-16` **capability** closes a gap AI-01 identified in its own text. Its § 7 preamble names five terms "AI-03's charter is not writable without" and the table delivers four. So this row is not a discovery by AI-03; it is AI-01's own list, completed.

## 7. Verification approach

Every requirement in `spec.md` is checkable by inspection. There is nothing to run, and that is a property of the node type rather than an exemption being taken quietly — `openspec/config.yaml` sets `apply.tdd: true` for Go service code, and this change writes none.

The verification pass in `tasks.md` is ordered by cost of a missed defect, not by document order:

| Rank | Check | Cost if missed |
| --- | --- | --- |
| 1 | Token counting is optional, with the full argument and the no-substitute corollary | A future adapter fabricates a count; compaction corrupts a transcript silently |
| 2 | *Absent* and *not exercised* are distinct outcome values, and the verdict rule separates them | A skipped case reads as a conformance pass — `V-PRV-09` defeated at the first use |
| 3 | The marking rule is total with a required default | AI-23 cannot satisfy its acceptance criterion; unclassified cases are skipped |
| 4 | Every required capability states what it does not oblige | An honest adapter fails conformance, or fabricates to pass |
| 5 | Every optional capability carries its reason | The list cannot be extended consistently; item 2 unsatisfied |
| 6 | Advertising binds, and wrappers forward | The widened contract returns in disguise; capabilities vanish across AI-37 |
| 7 | The nine-row cross-check is present with per-row verdicts | The optional list grows to nine, and three milestones pay for it |
| 8 | Scope: the deletion test, especially against AI-24.1 and AI-29.0 | A decision node inherits its answer instead of its question |
| 9 | Register discipline: citations resolve, amendment is append-only, definitions defer | Vocabulary drift, two milestones in |
| 10 | No Go identifiers | doc 0002's authoring constraint |

## 8. Acceptance criteria for the design phase

1. `decision.md`'s sections 5 … 10 follow AI-03.1's closing-checklist order; the four sections outside that spine each have a stated reason to be where they are.
2. Every required and optional entry carries all four parts of § 3.1.
3. The fabrication test is visibly applied to every required entry.
4. The divergence rule is visibly applied, with the nine-row cross-check present and per-row verdicts.
5. The distinct-values rule is visibly applied: the outcome set is closed and enumerated, with a verdict rule.
6. The marking rule is stated as a biconditional over the optional list with a required default, and § 3.3's reason is given.
7. The mechanism's asymmetry and both of its edges are stated together.
8. Section 12 of the artifact states an inheritance for AI-20.5, AI-23 and AI-24.
9. The register amendment follows § 6 exactly, including the deferral clause in each definition.

## 9. Next phase

`tasks.md` — five tasks, one per closing-checklist item, plus the register amendment and the verification pass. Then `decision.md`, the deliverable.
