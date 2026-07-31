# Design — Layer 1 stream lifecycle, ownership and carrier

> **Change**: `cachicamas-ai-stream-lifecycle`
> **Milestone**: AI-02 · **Node**: AI-02.1 `[decision]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-stream-lifecycle/spec.md`
> **Output**: the structure and the reasoning rules that `decision.md` implements
> **Diagrams**: ASCII (project convention — the sibling AI-01 change uses no mermaid in its proposal, spec or design artifacts)

---

## 1. What is being designed

Not a stream. A **document** that will be read by six later milestones as the settled physics of a stream, and read once each, usually by an agent or an engineer who has the milestone's own charter open and nothing else. The design question is therefore: what shape makes a five-part decision survive being consulted piecemeal?

Three properties follow from that reading pattern:

- **Each decision must be answerable without reading the other four.** A milestone opens the artifact for one question. If ownership cannot be understood without first reading the carrier section, the reader either reads everything or gets it wrong.
- **Each decision must carry its consequences in the same place as its conclusion.** A consequence recorded in a separate "implications" section at the end is a consequence nobody reads.
- **The argument must be present, not merely the conclusion.** doc 0002 requires it explicitly for the carrier — "the SDD must record why it chose what it chose" — and the same discipline is what stops a later reader from re-deciding a question because the reason looked absent.

## 2. The two failure modes this design targets

### 2.1 The restated default

doc 0002 says the carrier choice is free "here, and only here", and warns that the retired plan's default rested partly on an argument that is now void. The failure mode is subtle: a decision artifact that reaches the same conclusion the default suggested, by asserting it. Such an artifact is indistinguishable, on the page, from one that argued. It passes review. And it fails the first time someone asks *why*, at which point the question is genuinely reopened — except now it is wave 4 and reopening costs three times as much.

The countermeasure is a rule about the shape of the argument, not about its conclusion:

> **The strongest-opposing-case rule.** The rejected option is stated affirmatively, at its strongest, in its own voice, *before* any rebuttal. Each rebuttal then cites a document. A rebuttal that cites nothing is a preference, and preferences do not settle a decision that a later milestone can reopen.

Applied here, this means `decision.md` must be able to make a reader who arrived believing in iterators say "yes, that is my argument" before reading the defeats. `R-AIS-002` and `S-AIS-003` make that a checkable property.

### 2.2 The unfalsifiable constant

The buffer capacity has no right answer available today — there is no adapter, no transport, and no workload. Two ways of handling that are both wrong:

- **Defer it.** doc 0002's acceptance criterion forbids this in as many words: "AI-34's buffer measurement has a stated starting point to confirm or change." A measurement with no hypothesis is a survey.
- **State it and defend it.** A number defended on the page acquires authority it has not earned. AI-34 then has to *overturn* it rather than *test* it, which is a much higher bar and a predictable source of a wrong constant surviving into v1.

The countermeasure:

> **The falsifiable-hypothesis rule.** The number is stated as a hypothesis, together with the measurement that tests it and the direction each possible result implies. AI-34.1 inherits an experiment, not an opinion.

## 3. Structure of `decision.md`

```
  §1  How to use this document        ← who reads it, for what, and the one-line answers
  §2  What was decided                ← five one-line conclusions, before any argument
  §3  Decision 1 — the carrier        ← checklist item 1
  §4  Decision 2 — ownership          ← checklist item 2
  §5  Decision 3 — cancellation       ← checklist item 3
  §6  Decision 4 — buffering          ← checklist item 4
  §7  Decision 5 — failure delivery   ← checklist item 5
  §8  The complete lifecycle          ← the five decisions as one picture
  §9  What the package contract must state
  §10 What each blocked milestone inherits
  §11 Standing rules
  §12 Closing-checklist verification
```

Section order is the closing checklist's order, deliberately, so that a reviewer walking doc 0002 walks this document in parallel. § 2 exists because the most common access pattern is "what did they decide about X" and that reader should not have to enter an argument to find out.

### 3.1 The shape of a decision section

Every one of § 3 … § 7 carries the same five parts, in the same order:

| Part | Content | Why it is mandatory |
| --- | --- | --- |
| **Decision** | One paragraph. What was decided. | The reader who wants only this stops here |
| **Why** | The argument, with sources | doc 0002's requirement; the guard against a restated default |
| **What this excludes** | The alternatives, named and rejected | An unnamed alternative gets re-proposed |
| **Consequences** | What becomes true elsewhere | Consequences separated from conclusions are consequences nobody reads |
| **Who inherits it** | Named nodes, in their own terms | The acceptance criterion is about downstream writability |

Uniformity here is not tidiness. It means a reader who has read one section knows where to look in the other four, which is the difference between an artifact consulted five times and one read once.

### 3.2 Where the argument's weight goes

The five decisions are not equally contested, and the artifact should not pretend they are.

```
   contested                                                  settled
   |------------------------------------------------------------|
   carrier          failure          buffering    cancellation  ownership
   (genuinely       delivery         (number is   (obligations  (V-STR-05
    reopened;       (boundary is     free; the    are stated    already
    doc 0002 says   easy to draw     posture is   in doc 0001   states it;
    so explicitly)  wrong — G8)      not)         § 9)          this states
                                                                what it binds)
```

Argument length should track that gradient. The carrier gets the full opposing case; ownership gets precision rather than persuasion, because nobody disputes it — what is disputed is what it *binds*, and that is a question of statement, not of argument.

### 3.3 Why ownership is stated structurally rather than procedurally

"Close the stream exactly once" is a procedural instruction. It is unverifiable by reading, because "once" is a claim about all executions and a reader sees one text. Restating it structurally makes it readable:

```
  procedural:  "the producer closes the stream exactly once,
                on completion, on error, and on cancellation"
                     ↓  same obligation, checkable form  ↓
  structural:  "one goroutine sends; one closing site exists;
                it runs on every exit path; it runs after the
                last send attempt"
```

The structural form has three properties the procedural form lacks. It is checkable by reading a producer. It implies "no send after close" rather than needing it as a separate rule. And it forecloses the three misreadings `explore.md` § 5 enumerated, because each of them is a *shape* — a second closing site, several senders, a consumer-side close — rather than a behavior.

### 3.4 Why the delivery boundary needs a named rejected alternative

`explore.md` § 8 found that the delivery axis is routinely conflated with the partial-output discriminator, and doc 0001 § 7 **G8** records the resulting defect by name: "retry if nothing completed" is precisely the predicate that gets this wrong. The conflation is attractive because "the first event" is an intuitive boundary and it is wrong in exactly one case — a stream handed over that fails before emitting content.

An artifact that states only the correct boundary does not protect against this, because the reader who would draw it wrongly does not notice that they disagree. The artifact must therefore state the rejected boundary *as* a rejected boundary, together with the one case that separates them. That is `S-AIS-029` and `S-AIS-030`.

```
      call returns a carrier
              |
   pre-stream | mid-stream         ← the delivery axis (V-FAIL-11 / V-FAIL-12)
              |                      decided by THIS moment
   -----------+------------------------------------------------
              |
        (no content) | (content emitted)   ← the partial-output axis (V-FAIL-09)
                     |                       decided by a DIFFERENT fact
```

Two axes, one artifact, and the grid is the countermeasure. It mirrors AI-01 § 6's own two-axis diagram deliberately: a reader who has seen that one recognises the shape.

## 4. Reasoning rules the artifact applies

Four, each with the failure it prevents.

1. **Strongest-opposing-case** (§ 2.1) — prevents the restated default.
2. **Falsifiable-hypothesis** (§ 2.2) — prevents the unearned constant.
3. **Testable versus statable, marked as such.** Every obligation the artifact imposes is labelled with either the node that proves it or an explicit note that no test can. Prevents the failure where an untestable clause acquires a test that proves something weaker — the shape of doc 0001's C3, where "a shipped test documented the resulting gaps as expected behavior".
4. **Deletion test for scope.** AI-01 § 9 rule 5, applied verbatim: if a sentence were deleted, would a later milestone have more options? If yes and that milestone is not AI-02, cut it. Prevents over-reach into AI-14, AI-19, AI-20 and AI-34.

## 5. The line between deciding physics and deciding contents

AI-01 drew a line between settling a word and settling a behavior. AI-02 needs the next line along:

> **This milestone settles how a stream behaves as a container. It does not settle what the container carries.**

Worked cases, because the line is easier to state than to apply:

| Question | Whose? | Why |
| --- | --- | --- |
| How many terminal events may a stream have? | **AI-14** | It is a property of the event sequence. AI-02 cites `V-STR-18`; it does not restate it as its own rule |
| On which paths does the producer close? | **AI-02** | A lifecycle property of the container |
| What does a terminal error event carry? | **AI-19** | Contents |
| *Where* does a mid-flight failure arrive? | **AI-02** | Delivery is a container property, and AI-19.5 implements what AI-02 decides |
| Is a failure retryable? | **AI-19** classifies, **AI-35** acts | Neither is a container property |
| How large is the buffer? | **AI-02** starts, **AI-34** measures | The bound is a container property; the number is a measurement |
| Is the buffer size configurable? | **AI-34.1** | Deciding it now would remove an option from the milestone holding the evidence |
| What does the interface signature look like? | **AI-20** | AI-02 supplies what the documentation must say, not the declaration |

## 6. Handling of the register amendment

`explore.md` § 9 found two nouns the register lacks. AI-01 § 9 rule 2 governs, and this change is the first exercise of it, so the design records the procedure rather than leaving it to be improvised:

1. The rows are appended to AI-01's artifact — not defined here. This change's artifacts cite them by identifier only.
2. Ordinals are the next free ones in the `V-STR` category. No existing identifier moves.
3. A dated amendment blockquote is placed under the category heading, per the convention (`> **Amended YYYY-MM-DD** …`).
4. The register's own term counts and its checklist range reference are updated, because an artifact that contradicts its own arithmetic invites the reader to distrust the rest.
5. The owner of both new rows is AI-02, because AI-02 is the milestone that defines their behavior.

The rollback consequence is recorded in `proposal.md` and matters: the amendment and the decision are one unit. A decision citing an identifier nobody defined is exactly the failure AI-01 exists to prevent.

## 7. Verification approach

Every requirement in `spec.md` is checkable by inspection. There is nothing to run, and that is a property of the node type rather than an exemption being taken quietly — `openspec/config.yaml` sets `apply.tdd: true` for Go service code, and this change writes none.

The verification pass in `tasks.md` is ordered by cost of a missed defect, not by document order:

| Rank | Check | Cost if missed |
| --- | --- | --- |
| 1 | The carrier argument is present and cites sources | The decision is reopened in wave 4 at triple cost |
| 2 | The delivery boundary is handover, and the rejected alternative is named | The **G8** retry defect is rebuilt |
| 3 | "Exactly once" covers all three paths | A double close or a missing close on the error path |
| 4 | The abandonment clause is in the package contract | The clause is dropped by the first reader who notices no test enforces it |
| 5 | The capacity is falsifiable | AI-34 inherits an opinion to overturn instead of a hypothesis to test |
| 6 | Scope: the deletion test over every normative sentence | Over-reach into AI-14 / AI-19 / AI-20 / AI-34 |
| 7 | Register discipline: citations resolve, amendment is append-only | Vocabulary drift, one milestone in |
| 8 | No Go identifiers | doc 0002's authoring constraint |

## 8. Acceptance criteria for the design phase

1. `decision.md`'s section order matches AI-02.1's closing-checklist order.
2. Every decision section carries all five parts of § 3.1, in that order.
3. The strongest-opposing-case rule is visibly applied to the carrier.
4. The falsifiable-hypothesis rule is visibly applied to the capacity.
5. Every obligation is marked testable (with its node) or statable (with its reason).
6. The two-axis grid of § 3.4 appears in the artifact.
7. Section 10 of the artifact states an inheritance for AI-14, AI-20, AI-21, AI-22 and AI-34.
8. The register amendment follows § 6 exactly.

## 9. Next phase

`tasks.md` — five tasks, one per closing-checklist item, plus the register amendment and the verification pass. Then `decision.md`, the deliverable.
