# Proposal — Layer 1 stream lifecycle, ownership and carrier

> **Change**: `cachicamas-ai-stream-lifecycle`
> **Milestone**: AI-02 — Decide stream lifecycle, ownership, and the carrier
> **Node**: AI-02.1 — Lifecycle, ownership and carrier `[decision]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: documentation only — `openspec/changes/cachicamas-ai-stream-lifecycle/`, plus a two-row amendment to AI-01's register. **Zero Go. Zero `go.mod` change. Zero files under `backend/`.**
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-01 (`cachicamas-ai-contract-vocabulary`)
> **Blocks**: AI-14, AI-20, AI-21, AI-22 — and, through them, AI-19, AI-23, AI-33, AI-34, AI-40
> **Authoring constraint**: doc 0002's authoring constraint binds every artifact of this change. No Go identifier appears anywhere.

---

## Intent

Close doc 0002's AI-02.1 closing checklist — carrier, ownership, cancellation, buffering, failure delivery — in one merged artifact, so that AI-14 through AI-20 can be written without reopening any of the five, and so that AI-34's buffer measurement has a stated starting point to confirm or change.

The milestone exists because a stream contract that is settled *after* an adapter exists costs roughly three times as much to settle. doc 0001 § 3.2 records that multiplier for every wave-0 contract change, and doc 0001 § 7's disposition on **G13** applies it specifically to the carrier. The window is open now and closes when AI-24 lands.

## Locked constraints (inherited, not proposed)

These are not this proposal's to relitigate. They are listed so that a reviewer can tell the inherited from the decided.

1. **The vocabulary is AI-01's.** `V-STR-01` … `V-STR-09`, `V-FAIL-08` … `V-FAIL-12`, `V-STR-18`, `V-STR-20` are used with their register definitions. A definition is cited, never paraphrased.
2. **No Go identifiers.** doc 0002's authoring constraint; AI-14 and AI-20 choose spellings.
3. **No production code.** A `[decision]` leaf produces none, by doc 0002's node grammar.
4. **The stream ends with exactly one terminal event**, and nothing follows it — `V-STR-18`, restated by AI-14.4, not decided here.
5. **Cancellation is a context, not a polled flag**, and backoff waits on it rather than sleeping — doc 0001 § 9.
6. **Exactly one sanctioned loss path exists**; any other loss is a defect — `V-STR-09`.
7. **Validation runs once, before any I/O** — `V-REQ-22`. This is what keeps caller-contract failures out of the mid-stream path, and it is why AI-01's failure grid has an empty lower-left cell.

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `explore.md` | The option space, argued; the four defeats of the iterator case; the vocabulary gap |
| `proposal.md` | This file |
| `specs/ai-stream-lifecycle/spec.md` | `R-AIS-001` … `R-AIS-014`, each a checkable property of the decision artifact |
| `design.md` | The structure `decision.md` implements, and the reasoning method it applies |
| `tasks.md` | The single leaf AI-02.1, one task per closing-checklist item, plus the verification pass |
| `decision.md` | **The deliverable.** Five decisions, each with rationale, consequences, and the milestones that inherit it |
| AI-01's `decision.md` | **Amended**: two appended `V-STR` rows and their dated amendment blockquote, per AI-01 § 9 rule 2 |

### The five decisions this proposal commits to

Stated here in one line each so that a reviewer can accept or reject the substance before reading the full artifact. Each is argued in `decision.md`.

1. **Carrier — a receive-only channel.** Decided on four grounds, none of which is cost of change: Layer 2 must wait on a stream *and* on other things at once and an iterator cannot be waited on; the consumer must not be the socket reader; the terminal event is an element and an iterator has nowhere clean to put it; the iteration shape connotes a repeatable collection walk and a stream is single-use. The iterator-ergonomics requirement is **delegated to AI-22.5**, and doc 0002's waves 2–5 therefore need **no** amendment nodes.
2. **Ownership — one owning goroutine, one closing site, every exit path.** "Exactly once" means the closing site is unique in the producer, runs after the last send attempt, runs on completion, on terminal error and on cancellation alike, and runs on an unwinding exit. Nothing else closes: not the consumer, not the test kit, not the harness.
3. **Cancellation — caller-owned, waited on by every send, bounded close, abandonment is a violation.** The complete set of legal consumer endings is *drain to close* and *cancel*. Anything else is abandonment, which is a documented contract violation rather than a supported mode, stated in the package contract because it cannot be tested to termination and restated by AI-40.3 at the freeze.
4. **Buffering — bounded, starting capacity 64, falsifiable at AI-34.** Backpressure means waiting, never dropping. The single sanctioned loss path is cancellation with a saturated buffer: late events are dropped and the stream closes without a terminal event. Everything else is lossless.
5. **Failure delivery — the boundary is the handover of the carrier.** Before handover, a failure is returned directly and no stream and no producer ever exist (`V-FAIL-11`). After handover, every failure arrives as the terminal error event, including one that arrives before any content (`V-FAIL-12`). Whether content preceded the failure is a *different* fact (`V-FAIL-09`) and the two are never conflated.

### Out of scope (explicit, and deferred-but-related is called out)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Event kinds, payloads, sequence rules, ordering invariants | AI-14 | This milestone decides the container's behavior, not its contents |
| The provider interface declaration and its signature guard | AI-20 | AI-20 declares; this decision supplies what it must document |
| Failure categories, retryability, the constructible terminal payload | AI-19 | Classification is AI-19's; only *delivery* is decided here |
| Whether the buffer capacity is a constant or configurable | **AI-34.1** | Deferred deliberately. Deciding configurability now removes an option from the milestone that will hold the measurements |
| The measured capacity | **AI-34.1** | 64 is a starting hypothesis with stated falsification criteria, not a result |
| Goroutine-leak detection mechanism | AI-22.4 | Constrained by the module's dependency-free rule until AI-24 |
| Cancellation proven against a real transport | AI-33 | This decision states the obligation; AI-20.3 proves it against a fake, AI-33 against a transport |
| Layer 2's own carrier and observer model | doc 0003 AG-01.1 | This decision is an input to that one. AG-01's charter already records the default as "matching AI-02" |
| Retry policy | AI-35 | `V-FAIL-15`; the partial-output clause is promoted there, not here |

## Amendment to AI-01's register (scoped, two rows)

AI-01 § 9 rule 2: *"A missing term is appended, never invented. When a milestone needs a Layer 1 noun this register lacks, the term is appended there — next free ordinal in its category, dated amendment blockquote under the category heading — in the same pull request that needs it."* This change exercises that rule for the first time.

| New id | Term | Why AI-02 cannot proceed without it |
| --- | --- | --- |
| `V-STR-22` | **carrier view** | Decision 1 delegates ergonomics to AI-22.5 and must say what that thing *is* — a convenience adaptation of a stream into a different iteration shape, offered outside the package boundary, which is never a second contract. Without the noun, every downstream restatement says "iterator view", a phrase welded to one carrier choice and therefore wrong the day the choice is revisited |
| `V-STR-23` | **backpressure** | Decision 4 turns on "backpressure means waiting, never dropping" (the property AI-34.2 asserts). The word already appears *inside* `V-STR-08`'s definition without being defined, which is precisely the drift AI-01 exists to prevent — someone implements "backpressure" as dropping and cites the same word |

Both are container-side, both take the next free ordinals after `V-STR-21`, and both are owned by AI-02. No existing row is edited, renumbered or reworded; AI-01 § 9 rule 3 (append-only) holds.

## Approach

1. **Argue the carrier from the strongest opposing case.** doc 0002 requires the SDD to record *why* it chose what it chose. A decision that restates a default has not been made. `explore.md` § 4.2 therefore builds the iterator case at full strength before defeating it, and each defeat cites a document, not an intuition — doc 0001 § 4.1 and doc 0003 AG-10.3 for the multiplexing requirement, `V-STR-18`/`V-FAIL-10` for the terminal-event shape, `V-PRV-10` for the wrong-physics hazard.
2. **State ownership structurally, not procedurally.** "One owning goroutine, one closing site" is checkable by reading a producer. "Close it exactly once" is not.
3. **Separate the testable from the statable.** Cancellation's three obligations are testable and are handed to AI-20.3 as such. Abandonment is not, and is handed to the package contract and to AI-40.3 instead. Mixing the two is how an untestable clause acquires a test that proves something weaker than it claims.
4. **Make the capacity falsifiable.** The number is presented with the measurement that would move it and the direction each result implies, so AI-34.1 inherits a hypothesis rather than a preference to overturn.
5. **Draw the delivery boundary at an observable moment.** Handover, not first event. The alternative fails on contact with `V-FAIL-09`.
6. **Close by inheritance.** The artifact ends with what each blocked milestone receives, in that milestone's own terms, so a reviewer can check the acceptance criterion — AI-14 … AI-20 writable without reopening — by reading one table.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-stream-lifecycle/` | Six new markdown files | None — new directory |
| `openspec/changes/cachicamas-ai-contract-vocabulary/decision.md` | Two appended rows, one dated blockquote, two counts updated | Low — append-only, no existing row touched |
| `backend/agent/` | **None** | — |
| `go.mod`, `go.work`, `docker-compose.yaml`, `infra/` | **None** | — |
| doc 0002 | **None.** Channels won, so the default holds and waves 2–5 gain no amendment nodes | — |

## Rollback plan

The change is additive documentation with one append-only amendment. Rollback is `git revert` of the single commit; nothing is generated from these files, nothing imports them, and no build depends on them.

Partial rollback is possible and its shape matters: reverting only the two appended register rows would leave `decision.md` citing identifiers that do not resolve. If the amendment is rejected in review, the correct move is to reject the whole change and re-propose, not to strip the rows — a decision that cites a term nobody defined is the exact failure AI-01 was built to prevent.

Post-merge reversal is a different matter and is the reason the milestone is scheduled here: after AI-14 and AI-20 land, the carrier is embedded in an envelope contract, an interface, a signature guard, a fake provider and a conformance suite. doc 0001 § 7 prices that at roughly three times this milestone.

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| The carrier decision reads as a restated default rather than an argued one | Medium | High — doc 0002 explicitly requires the argument | `explore.md` § 4.2 states the opposing case at full strength; every defeat cites a source; `spec.md` `R-AIS-002` makes "the sunk-cost argument is absent" a checkable property |
| The chosen capacity is wrong | High | Low | It is labelled a hypothesis with stated falsification criteria and an owning milestone (AI-34.1). Being wrong is the expected case; being *unfalsifiable* is the failure |
| The delivery boundary is drawn at the first event by a later reader | Medium | High — reintroduces the **G8** retry defect | The decision states the boundary as handover, states the rejected alternative, and states why `V-FAIL-09` is a different axis. `R-AIS-011` checks it |
| Over-reach into AI-14, AI-19, AI-20 or AI-34 | Medium | Medium | AI-01 § 9 rule 5's test is applied literally: if a sentence were deleted, would a later milestone have more options? If yes and the milestone is not AI-02, the sentence is cut. `R-AIS-013` |
| Abandonment clause is quietly dropped because no test enforces it | Medium | Medium | It is required in the package contract by `R-AIS-008`, inherited by AI-20.1 item 3, and restated at AI-40.3 |
| Layer 2 diverges and picks the other carrier | Low | Medium | doc 0003 AG-01.1 item 1 already names channels as its default "matching AI-02". This decision's cancellation and ownership rules are stated so AG-01 can mirror them rather than re-derive them |

## Dependencies

- **AI-00** (`cachicamas-agent-module-scaffold`) — the module. Not touched by this change, but the milestone chain runs through it.
- **AI-01** (`cachicamas-ai-contract-vocabulary`) — **hard.** Every noun used here is one of its rows, and this change amends it.
- No new Go dependency. No ADR required: the module stays dependency-free until AI-24, and this change adds nothing.

## Success criteria

1. All five closing-checklist items of AI-02.1 are answered in `decision.md`, each with rationale.
2. The carrier decision records the strongest opposing case and why it loses, and rests on no argument from cost of change.
3. Delegation to AI-22.5 is stated explicitly, and the absence of doc 0002 amendment nodes is stated as a consequence.
4. "Exactly once" is stated across completion, error and cancellation — not implied from the happy path.
5. The abandonment clause is present, marked as belonging in the package contract, and attributed to AI-40.3 for restatement.
6. A starting buffer capacity is chosen, justified, and accompanied by what would falsify it.
7. Pre-stream and mid-stream delivery are separated at an observable moment, and kept orthogonal to the partial-output discriminator.
8. AI-14, AI-20, AI-21 and AI-22 each have a stated inheritance, and AI-34 has a stated starting point.
9. The change adds six markdown files and amends exactly one existing file, append-only.
10. No Go identifier appears in any artifact.

## Notes for the following phases

- **`spec.md`** — the system under test is the artifact, as it was for AI-01. Requirement IDs `R-AIS-0NN`, scenarios `S-AIS-0NN`. Every scenario must be checkable by inspection, without running anything.
- **`design.md`** — owns the structure of `decision.md` and the two reasoning methods it applies: the strongest-opposing-case rule for the carrier, and the falsifiable-hypothesis rule for the capacity.
- **`tasks.md`** — five tasks, one per closing-checklist item, plus the register amendment and the verification pass.
- **`decision.md`** — the deliverable. Ends with the inheritance table, because the acceptance criterion is stated in terms of what downstream milestones can do without reopening this one.
