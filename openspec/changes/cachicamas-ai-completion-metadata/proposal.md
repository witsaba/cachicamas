# Proposal — finish reasons and usage

> **Change**: `cachicamas-ai-completion-metadata`
> **Milestone**: AI-13 — Define finish reasons and usage
> **Nodes**: AI-13.1 `[leaf]` · AI-13.2 `[leaf]` · AI-13.3 `[leaf]` · AI-13.4 `[leaf]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: `openspec/changes/cachicamas-ai-completion-metadata/`, and **four new Go files** under `backend/agent/src/ai/`. No `go.mod` change, no new dependency, no register amendment
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-00 … AI-03 (wave 0, merged), AI-04 (merged on the Wave 1 branch)
> **Blocks**: AI-15.2, AI-31

---

## Intent

Define the two values a response carries when it is over: **why generation stopped**, and **what it consumed**. Both are small. Both are also the two places Layer 1 has historically lied to Layer 2 — once by collapsing distinct stop conditions into a fallback (doc 0001 **G12(c)**), and once by presenting an unreported token count as a zero (`V-MET-11`). This milestone lands them so that neither lie is expressible.

The milestone absorbs **G12(c)**. The retired plan reached the same vocabulary by adding two values to a frozen enum in a corrective milestone; here refusal and pause-turn are in the vocabulary from the first commit, and the corrective milestone never exists.

## Locked constraints (inherited, not proposed)

1. **The vocabulary is AI-01's.** `V-MET-01` … `V-MET-12` are used with their register definitions and cited, never paraphrased. This change *implements* them. No register amendment is needed — the first Wave 1 change for which that is true.
2. **The capability precision is AI-03's.** `CAP-R-03` clause 2: requiring the usage record does not require any count to be populated. An adapter that reports nothing produces a valid all-absent record.
3. **Every rule violation reports through AI-04.** The five sentinels, `Invalid`, `At`/`AtIndex`, `Rule`, `FirstFailure`. No new sentinel and no second failure type.
4. **Loop termination is Layer 2's** — `V-OUT-10`. The finish reason is the input to that decision, never the decision.
5. **The module carries zero requires.** Standard library only; no exhaustiveness linter, no assertion library.
6. **Strict TDD**, `openspec/config.yaml` `apply.tdd: true`, and doc 0002's behavior-leaf grammar: every test-list item red → green → refactored, in order.
7. **The evidence gate is recorded green `make test`** in `backend/agent/` — `go test -race -v ./...`.

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `explore.md` | What is landed, what the register fixes, five design tensions, the vendor scan, the (empty) vocabulary gap |
| `proposal.md` | This file |
| `specs/ai-completion-metadata/spec.md` | `R-ACM-001` … `R-ACM-012` over runtime behavior, with Given/When/Then scenarios |
| `design.md` | The Go surface — every type, constant, method and its reasoning — plus the two semantic decisions and the pin mechanisms |
| `tasks.md` | Four leaves as phases, one task per test-list item, with the recorded red and green output |
| `backend/agent/src/ai/finish_reason.go` | **AI-13.1 and AI-13.2's deliverable.** The closed vocabulary, the total normalizer, the string form, the validation rule |
| `backend/agent/src/ai/finish_reason_test.go` | AI-13.1's five items and AI-13.2's two, in external test package `ai_test` |
| `backend/agent/src/ai/usage.go` | **AI-13.3 and AI-13.4's deliverable.** The token count with its presence, the usage record, the two documented totals |
| `backend/agent/src/ai/usage_test.go` | AI-13.3's three items and AI-13.4's two |

Four Go files rather than two: the milestone has two graph chains that share no symbol, and doc 0002 records that "two people could work it concurrently without conflict — then it was two nodes all along". Splitting the files along the chain boundary is that observation made physical, and it keeps a reviewer's diff for the cost-formula decision separate from the diff for the stop vocabulary.

### The two decisions this proposal commits to

Stated in one line each so a reviewer can accept or reject the substance before reading `design.md` § 4 and § 5.

1. **Cached input tokens are EXCLUDED from the input count.** The five fields are mutually exclusive on the input side: `input`, `cache read` and `cache write` never count the same token twice, and the total input is their sum. Vendors that report an inclusive figure subtract in their adapter. Chosen because the three have three different prices — so no consumer can ever sum them at one rate anyway — and because absence then degrades gracefully: an absent cache-read leaves the input field still true of what it names, where under inclusive semantics an absent cache field silently corrupts the arithmetic.
2. **Reasoning tokens are INCLUDED in the output count.** The opposite rule on the opposite side, and deliberately so: both vendors bill reasoning at the output rate, so it is a breakdown of a billed quantity rather than a billed quantity of its own. This also keeps a mid-stream usage update honest — `V-MET-12` records that the reasoning count arrives only on the final update, and under this rule the output count is complete at every update while the attribution fills in later.

The rule behind both, stated once so a later field knows which side it lands on: **a count priced differently from every other is a term; a count priced the same as another is a breakdown of it.**

### Out of scope (explicit, and deferred-but-related is called out)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Money, price tables, per-turn and per-session cost events | Layer 2 / Layer 3 — `V-OUT-07`, `V-OUT-08` | doc 0002's own out-of-scope clause for AI-13.4. Layer 1's obligation ends at honest token counts |
| The completion event that carries a finish reason and a usage record | **AI-15.2** | The event envelope does not exist until AI-14. AI-15.2's test list already restates AI-13.3's property as one it must preserve across the event boundary |
| Mapping any vendor's stop strings | **AI-31.1** | The neutral synonym table here is what that mapping targets. AI-31.1 is where "every vendor stop value maps to its normalized reason, never to unknown" is proven against a real adapter |
| Verifying the inclusive/exclusive decision against a real cache-hit transcript | **AI-31.3** | doc 0002 assigns it in those words. This milestone decides and pins the neutral semantics; the adapter proves the semantics are true of it |
| Any resumption mechanism for a paused turn | **Layer 2**, mapped at AI-31.1 | `V-OUT-10`. The obligation is *recorded* here, in GoDoc and in this change's spec, for AI-31.1 and Layer 2 to honor |
| A `Resumable()`/`NextAction` method on the finish reason | **rejected** — see `design.md` § 4.3 | It would move loop termination into Layer 1, which `V-OUT-10` reserves for Layer 2 |
| An aggregate "every violated rule" report | **AI-04.1, already rejected** | Inherited. Usage validation reports the first violated rule in the documented field order |
| Any role, message or content-part type | AI-05, AI-06 | Concurrent work; the file sets are disjoint by construction |

## Register amendment

**None.** Register § 9 rule 2 obliges a milestone to append a Layer 1 noun it needs and cannot find; this milestone needs none. `V-MET-01` … `V-MET-12` cover every noun in the charter.

One non-blocking observation is recorded in `explore.md` § 5 for whoever amends next: `V-MET-08` uses the phrase "provider stop condition" for the raw vendor string without the register defining it, the same shape as the two elisions the register has already amended. This change cites the phrase as written rather than inventing a local definition.

## Approach

1. **Implement the register, do not redesign it.** Every property proven below is a clause already in `V-MET-01` … `V-MET-12`. The tests are those clauses made mechanical.
2. **Make the two lies inexpressible rather than discouraged.** The unknown value is not the zero value, so a finish reason nobody set cannot masquerade as a recorded one; an absent count is the zero value of its own type, so a record nobody populated cannot masquerade as a record of zeros.
3. **Total normalization.** `NormalizeFinishReason` returns a value for every string, including the empty one. There is no error return, because there is no failure — an unrecognised string is a recorded outcome (`V-MET-08`), not a fault.
4. **Pin exhaustiveness with the language, not a linter.** The vocabulary's underlying integer is enumerable; the pin walks it, compares the discovered set against the hand-named set, and requires each member to round-trip through its own string form.
5. **Pin the formula to the field set.** The cost formula's term list and the record's field list are the same list. A reflection-based pin asserts the field set exactly, so a sixth field cannot land without the author meeting the formula.
6. **Stay behind `V-OUT-10`.** The three-way distinguishability is proven by an exhaustive consumer-shaped switch **in the test**, which demonstrates that three different responses are writable without Layer 1 writing them.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-completion-metadata/` | Five new markdown files | None — new directory |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **None** — no term is missing | — |
| `backend/agent/src/ai/finish_reason.go` + test | New files | **Medium** — AI-31.1 maps every vendor onto this vocabulary, and AI-40 freezes it |
| `backend/agent/src/ai/usage.go` + test | New files | **Medium** — the inclusive/exclusive decision is the one a wrong answer makes expensive |
| `backend/agent/src/ai/validation.go` and its test | **None** — read only | — |
| `backend/agent/go.mod`, `go.work` | **None** — zero requires preserved | — |
| `backend/agent/src/agenttest/` | **None** — reserved for AI-06.2 | — |
| doc 0002 | **None expected.** Any amendment required by the revert-and-record clause lands in this same PR and is reported prominently | — |

## Rollback plan

Purely additive: four new Go files and one new change directory. Nothing existing is modified. Rollback is `git revert` of the commit range, and `src/ai` returns to exporting only the validation taxonomy.

Post-merge reversal is where the cost sits, and it is asymmetric between the two chains. The finish-reason vocabulary can grow a value additively for as long as consumers switch exhaustively — that is precisely what AI-13.1's pin protects. The **inclusive/exclusive decision cannot be reversed quietly**: flipping it changes the meaning of a field without changing its name or type, so every adapter and every consumer keeps compiling and starts being wrong. If it is to be reversed, it must be reversed before the first adapter lands (AI-31), and the reversal must rename the fields so that the change is compile-visible.

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| The inclusive/exclusive decision is the wrong one | Medium | **High** — silent under- or over-reporting on every cached call | Decided with both vendors' behavior recorded (`explore.md` T3), pinned by test, and scheduled for verification against a real transcript at AI-31.3. The rollback note above states what a reversal costs and how to make it visible |
| A vendor ships a stop value the vocabulary lacks | **High** — it is a certainty over time | Low | The normalizer is total by test (AI-13.1 item 4): an unrecognised string becomes the unknown value, never a panic and never a silent substitution for refusal or pause |
| A later value is added without extending the table or the string form | Medium | High — a value that normalizes to nothing is the collapse G12(c) exists to prevent | AI-13.1 item 5's pin, shown to bite against a scratch value before it lands |
| A later field is added to usage and the formula quietly stops being true | Medium | High | AI-13.4 item 2's pin asserts the field set exactly |
| The three-way distinction is collapsed by a well-meaning simplification | Low | **High** — doc 0001 calls it a loop-termination bug | The three constants are named individually in the tests, so removing one fails to compile; AI-13.2's exhaustive switch has a failing `default` |
| A `Resumable()` convenience is added later and drags loop termination into Layer 1 | Medium | Medium | `V-OUT-10` cited on the declaration itself, and the rejection recorded in `design.md` § 4.3 |
| The milestone grows past its review budget | **High** — four leaves in one milestone | Medium | The surface is fixed by `design.md` before the first test; `tasks.md` carries the forecast and the reassessment against doc 0002's split triggers |

## Dependencies

- **AI-04** — the validation taxonomy. Both validators report through it, and no sentinel is added.
- **AI-01** — the register. Every noun used here is one of its rows.
- **AI-03** — `CAP-R-03`'s second clause is the reason the all-absent usage record is *valid* rather than deficient.
- **AI-00** — the module and both import guards; both must still pass afterwards.
- **No new Go dependency and no ADR required.** `strings`, `strconv`, `errors`, `slices`, `reflect` (tests only) are standard library.

## Success criteria

1. Every test-list item of all four leaves is taken red → green → refactored, in order, with both outputs recorded in `tasks.md`.
2. Both pins are shown to **bite** — a scratch violation is added, the failure recorded, the scratch removed.
3. `make test` in `backend/agent/` is green with `-race`, and `make lint` is clean; both outputs recorded.
4. Both import-boundary guards still pass and `go.mod` still carries zero requires.
5. The vocabulary contains seven values including refusal and pause-turn, and the zero value is not one of them.
6. Refusal and content filter are distinct values, with the line between them documented on the declarations.
7. Every string in the vendor scan normalizes to its value after trimming and lowercasing, and every unrecognised string — including the empty one — becomes unknown without error.
8. A usage record distinguishes absent from zero on all five counts, is constructible with any subset present, and is readable field by field from an external package.
9. The inclusive/exclusive semantics of the input count are documented on the declaration and pinned against a constructed cache-hit record.
10. The cost formula is asserted in the test and pinned to the record's field set.
11. No input causes a panic: not the zero value, not an out-of-range value, not an empty or over-long provider string.

## Notes for the following phases

- **`spec.md`** — the system under test is runtime behavior. Requirement IDs `R-ACM-0NN`, scenario IDs `S-ACM-0NN`, each scenario independently verifiable by one test.
- **`design.md`** — owns every Go spelling, the two semantic decisions with their rejected alternatives argued first, both pin mechanisms, and the statement of how a red step is taken in a statically typed language (inherited from AI-04's `design.md` § 6).
- **`tasks.md`** — four phases, one per leaf; one task per test-list item; red and green evidence recorded per item; the order of the two chains stated and defended.
