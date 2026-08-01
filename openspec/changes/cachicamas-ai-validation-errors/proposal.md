# Proposal — the validation error taxonomy

> **Change**: `cachicamas-ai-validation-errors`
> **Milestone**: AI-04 — Define the validation error taxonomy
> **Nodes**: AI-04.1 `[decision]` · AI-04.2 `[leaf]` · AI-04.3 `[leaf]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: `openspec/changes/cachicamas-ai-validation-errors/`, a two-row append to AI-01's register, and **one production file plus one test file** under `backend/agent/src/ai/`. No `go.mod` change, no new dependency
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-00, AI-01, AI-02, AI-03 (wave 0, merged)
> **Blocks**: AI-05 … AI-13, AI-19

---

## Intent

Give every later Layer 1 contract **one** way to report that a caller broke a rule, and prove the three properties that make it usable: a failure is matchable through wrapping, its position is extractable, and its message carries no caller content. Nine milestones consume it; none of them should have to think about it.

doc 0002 schedules the milestone first on a recorded cost. The retired plan defined this taxonomy seventeenth, after ten milestones had each invented their own sentinels, and then spent a milestone rationalizing them. Defined before the first validating contract, that debt is not repaid — it is never taken on.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can tell the inherited from the decided.

1. **The taxonomy is AI-01's.** `V-FAIL-01` … `V-FAIL-04` are used with their register definitions and are cited, never paraphrased. This change *implements* them.
2. **The boundary rule is AI-01's.** Register § 6.3's decidability-without-I/O test is applied, not re-derived.
3. **Validation runs once, before any I/O** — `V-REQ-22`. Every caller-contract failure is a returned value; none is ever an event.
4. **The redaction posture starts here** — `V-FAIL-13`, owned by AI-19, states it explicitly: "at the first thing in the package that formats caller data — a validation failure — not at the hardening milestone."
5. **The module carries zero requires** until AI-24 and AI-37, each behind its own ADR gate. Two landed tests hold it there.
6. **Strict TDD.** `openspec/config.yaml` sets `apply.tdd: true`, and doc 0002's behavior-leaf grammar requires every test-list item to be taken red → green → refactored, in order.
7. **The evidence gate is recorded green `make test`** in `backend/agent/` — that is `go test -race -v ./...`.

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `explore.md` | What exists, what the register already fixes, the four design tensions, the prior-art scan, the vocabulary gap |
| `proposal.md` | This file |
| `decision.md` | **AI-04.1's deliverable.** Four decisions, each with the rejected alternative argued at full strength first, four borderline cases, and the conceded costs |
| `specs/ai-validation-errors/spec.md` | `R-AIE-001` … `R-AIE-012`, requirements over runtime behavior with WHEN/THEN scenarios |
| `design.md` | The Go shapes — type, function and sentinel names, and the reasoning behind each spelling |
| `tasks.md` | AI-04.1/.2/.3 as phases, one task per test-list item, with the red→green record and the review workload forecast |
| `backend/agent/src/ai/validation.go` | **AI-04.2 and AI-04.3's deliverable.** The failure value, the class set, positional context, the ordered-rule mechanism |
| `backend/agent/src/ai/validation_test.go` | The six test-list items, in external test package `ai_test` |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **Amended**: two appended `V-FAIL` rows, a dated amendment blockquote, and updated counts |

### The four decisions this proposal commits to

One line each, so a reviewer can accept or reject the substance before reading `decision.md`.

1. **The boundary** — register § 6.3's rule applied, with the clause that is routinely dropped made load-bearing: *decidable without I/O* is necessary but not sufficient; the fact must also be a property of the **request value**. Four borderline cases resolved, three of them new: a provider cap smaller than the documented breakpoint cap, a tool result that reports the tool failed, and a call signal that has already expired.
2. **Granularity — one sentinel per rule class, reusable across types.** doc 0002's default, taken because per-instance sentinels *are* the recorded defect rather than merely resembling it. Consequence: `errors.Is` answers "which class", `errors.As` answers "where", and the two axes never overlap.
3. **Ordered first-failure.** doc 0002's default, taken — but not on the allocation argument, which this proposal concedes is the weakest of the three. It is taken because a joined error matches a sentinel when *any* member matches, which takes back everything decision 2 bought.
4. **One failure type carrying an always-present, possibly empty position.** Positions are built only from filtered structural names and integer indices; the rendered message never quotes the error it was given, only the registered rule class it matches. Redaction becomes a property of the type rather than of everybody's discipline.

### Out of scope (explicit, and deferred-but-related is called out)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Provider status codes, categories, retryability, partial output, the terminal error event | AI-19 | Everything after a valid request leaves the process. AI-04's charter says so |
| Any validator for a role, message, content part, tool declaration, tool call or request | AI-05 … AI-13 | Those types do not exist. Writing their validators here freezes their design from outside their own milestone |
| The repo-wide adversarial redaction scan | **AI-36** | Deferred deliberately. `V-FAIL-14` is AI-36's; AI-04 owns the posture at the point of formatting, and AI-36 proves it holds everywhere |
| Aggregate reporting of every violation | **AI-04.1, rejected** | Rejected with the reversal path recorded: an "every violation" sibling remains addable without changing the failure type |
| A rule class for conflicting values | **the milestone that meets it** | No citable case exists yet. The append discipline is what makes waiting cheap |
| An external-package proof under `src/agenttest/` | **AI-06.2** | That file's own comment reserves the first real external-readability proof for the first content part. AI-04's tests live in the external test package `ai_test`, which already has no access to unexported state |
| Structured logging, error codes, wire rendering | doc 0003 / doc 0004 | Layer 1 returns values; it does not render them |

## Amendment to AI-01's register (scoped, two rows)

Register § 9 rule 2: *"A missing term is appended, never invented … in the same pull request that needs it."* This change exercises it for the third time.

| New id | Term | Why AI-04 cannot proceed without it |
| --- | --- | --- |
| `V-FAIL-16` | **validation rule** | The word "rule" appears undefined inside `V-FAIL-01`, `V-FAIL-02` and `V-FAIL-04`. This is exactly `V-STR-23` **backpressure**'s situation, which the register's own amendment history treats as a defect rather than an acceptable elision |
| `V-FAIL-17` | **rule class** | doc 0002's checklist states the default as "one sentinel per rule **class**", and the register has no term for the thing a sentinel is one of. Without it, decision 2 cannot state its own unit of granularity |

Both take the next free `V-FAIL` ordinals after `V-FAIL-15`, both are owned by AI-04, and both defer their substance to AI-04: the register says what a rule class *is*, never which classes exist. No existing row is edited, renumbered or reworded.

## Approach

1. **Implement the register, do not redesign it.** Every property AI-04.2 and AI-04.3 prove is a clause already present in `V-FAIL-01` … `V-FAIL-04`. The tests are those clauses made mechanical.
2. **Argue each decision from the strongest opposing case.** doc 0002 requires the SDD to record *why*. Each of the four sections in `decision.md` states the rejected option affirmatively and at length before rejecting it, and each rejection cites a source rather than a preference.
3. **Make redaction structural.** The type is designed so that caller data has nowhere to go: positions carry filtered structural names and integers, and the message renders the registered class's own words rather than the supplied error's. AI-04.3's pin then bites on a real refactor instead of documenting an intention.
4. **Ship only the surface the charter promises.** "One way to report a rule violation" is a failure value, a class set, a position, and the ordered-rule mechanism that decides which rule fires. Anything type-specific is a later milestone's.
5. **Take the restrictive option where reversal is one-way.** First-failure can grow an aggregate sibling; an aggregate contract cannot be narrowed after AI-40 freezes the surface.
6. **Prove determinism by repetition and by race**, not by inspection. AI-04.3 item 1 runs the same multi-violation check many times and concurrently, because "no map iteration decides which rule fires" is a claim about behavior under a scheduler.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-validation-errors/` | Six new markdown files | None — new directory |
| `openspec/specs/ai-contract-vocabulary/spec.md` | Two appended rows, one dated blockquote, counts updated | Low — append-only, no existing row touched |
| `backend/agent/src/ai/validation.go` | New file — the first non-guard Go in Layer 1 | Medium — nine milestones depend on the shape. Mitigated by AI-04.1 and by `spec.md` |
| `backend/agent/src/ai/validation_test.go` | New file | None |
| `backend/agent/go.mod`, `go.work` | **None** — zero requires preserved | — |
| `backend/agent/src/agenttest/` | **None** — reserved for AI-06.2 | — |
| doc 0002 | **None expected.** Any amendment required by the revert-and-record clause lands in this same PR and is reported prominently | — |
| `docker-compose.yaml`, `infra/`, the other two backend modules | **None** | — |

## Rollback plan

The change is additive: one new package file, one new test file, one new change directory, and an append-only amendment. Nothing existing is modified except the register. Rollback is `git revert` of the commit range; no build depends on the markdown, and `src/ai` returns to exporting nothing, which is its state today.

Partial rollback has a shape worth stating: reverting the register amendment alone would leave `decision.md` citing `V-FAIL-16` and `V-FAIL-17`, which would not resolve. If the amendment is rejected in review, the correct move is to reject the change and re-propose — a decision citing a term nobody defined is the exact failure AI-01 exists to prevent.

Post-merge reversal is a different matter and is why the milestone is scheduled first. Once AI-05 … AI-13 report through the taxonomy, changing its shape is a change to nine contracts. That multiplier is the whole reason doc 0002 moved this milestone from seventeenth to first.

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Speculative validators leak in for types that do not exist | Medium | High — freezes AI-05 … AI-13's design from outside | `R-AIE-011` makes it a checkable property: the package exports no rule bound to a Layer 1 type. The exercise surface is the ordered-rule mechanism itself |
| The class set is wrong — too coarse or too fine | High | Low | Wrong is the expected case. The append discipline (§ 3.5 of `decision.md`, `R-AIE-003`) makes correction a one-row change in the pull request that meets it. *Unappendable* would be the failure |
| The redaction claim rests on discipline rather than construction | Medium | High — the leak surface is every log line above Layer 2 | Two structural constraints (`R-AIE-006`), and AI-04.3's pin feeds a distinctive sentinel body through both dynamic channels |
| First-failure is regretted at AI-10, where requests are large | Medium | Low | The reversal is one-way in the safe direction, and `decision.md` § 4.3 records the addable sibling |
| A red step is skipped because Go will not compile a test against a symbol that does not exist | Medium | Medium — it is the discipline the whole repo is built on | `design.md` § 6 states the rule: the red step lands the narrowest declaration that compiles and fails the assertion, never one that passes. Every red run is recorded in `tasks.md` |
| The milestone grows past its review budget | Low | Medium | The surface is fixed by `design.md` before the first test. `tasks.md` carries the forecast, and doc 0002's split triggers are checked against it |

## Dependencies

- **AI-00** — the module, the package, and both import guards. Both must still pass afterwards.
- **AI-01** — the register. Every noun used here is one of its rows, and this change amends it.
- **AI-02**, **AI-03** — merged; they fix the delivery split and make typed failure delivery a required capability, neither of which this change reopens.
- **No new Go dependency, and no ADR required.** The module stays dependency-free; `errors` and `strings` are standard library.

## Success criteria

1. All four closing-checklist items of AI-04.1 are answered in `decision.md`, each with the rejected alternative argued at full strength before it is rejected.
2. At least one borderline case beyond the register's own is resolved. *(Three are.)*
3. Every test-list item of AI-04.2 and AI-04.3 is taken red → green → refactored, in order, with both outputs recorded.
4. `make test` in `backend/agent/` is green with `-race`, and its output is recorded.
5. Both import-boundary guards still pass and `go.mod` still carries zero requires.
6. A failure is matchable by `errors.Is` through at least one layer of wrapping.
7. A failure's position is extractable by `errors.As` and names which message, which content-part index, and which tool, unambiguously.
8. No formatted message can carry a content body, an argument payload or a credential — by construction, proven with a distinctive sentinel body.
9. A value violating several rules reports the first in the documented order, identically across many runs and under `-race`.
10. No input causes a panic, including nil, empty, deeply nested and maximum-size values.
11. The register amendment lands in the same pull request, append-only, with correct counts.
12. The package exports no validator bound to a type AI-05 … AI-13 has not yet defined.

## Notes for the following phases

- **`spec.md`** — unlike wave 0, the system under test is **runtime behavior**, not the artifact. Requirement IDs `R-AIE-0NN`, scenarios `S-AIE-0NN`, every scenario independently verifiable by a test.
- **`design.md`** — owns every Go spelling: the failure type, the class set, the position, the ordered-rule mechanism, and the two structural redaction constraints. It also states how a red step is taken in a statically typed language.
- **`tasks.md`** — three phases, one per node; one task per test-list item; the red and green evidence recorded per item; the verification pass last.
