# Proposal — roles and message identity

> **Change**: `cachicamas-ai-message-roles`
> **Milestone**: AI-05 — Define roles and message identity
> **Nodes**: AI-05.1 `[leaf]` · AI-05.2 `[leaf]` · AI-05.3 `[leaf]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: `openspec/changes/cachicamas-ai-message-roles/`, and **two production files plus two test files** under `backend/agent/src/ai/`. No `go.mod` change, no new dependency, **no register amendment**
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-00, AI-01, AI-02, AI-03, AI-04
> **Blocks**: AI-06 … AI-10

---

## Intent

Land the smallest addressable unit of a transcript — **one role, ordered content, a stable identity, and no way for a caller to mutate it from outside** — and land it as the first consumer of AI-04's taxonomy rather than as a second, parallel way of saying a caller broke a rule.

Two things make this milestone worth more than its size. It is the **walking skeleton of wave 1's domain**: every later contract in the wave (AI-06 … AI-13) either goes inside a message or goes beside one, and each inherits the shape decided here. And it is the milestone that has to hold a seam open for AI-06 rather than closing it — see § "The content-part seam" below, which is the part of this proposal a reviewer should read first.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can tell the inherited from the decided.

1. **The three terms are AI-01's.** `V-REQ-01` role, `V-REQ-02` message, `V-REQ-03` message identity are used with their register definitions and cited, never paraphrased. This change *implements* them.
2. **The failure vocabulary is AI-04's.** Every rule violation reports through `Invalid` with one of the five landed rule classes, at a position, composed by `FirstFailure`. No new sentinel, no local `errors.New`.
3. **`V-REQ-04` content part is AI-06's**, and both of its constitutive properties — external readability and zero-value invalidity — are decided **together, before any code exists**, by AI-06.1.
4. **The module carries zero requires** until AI-24 and AI-37, each behind its own ADR gate. Two landed tests hold it there.
5. **Strict TDD.** `openspec/config.yaml` sets `apply.tdd: true`; doc 0002's behavior-leaf grammar requires every test-list item taken red → green → refactored, **in order**.
6. **The evidence gate is recorded green `make test`** in `backend/agent/` — that is `go test -race -v ./...`.
7. **Tests live in the external test package `ai_test`.** AI-04's convention, and the reason it exists: a contract that validates cleanly only from inside its own package is the retired defect **C2** in miniature.

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `explore.md` | What exists, the three terms, the four tensions, the prior-art scan, the vocabulary check |
| `proposal.md` | This file |
| `specs/ai-message-roles/spec.md` | `R-AMR-001` … `R-AMR-011`, requirements over runtime behavior with WHEN/THEN scenarios |
| `design.md` | The Go shapes — every type, function and constant name, the seam, and the reusable closed-vocabulary pattern |
| `tasks.md` | AI-05.1/.2/.3 as phases, one task per test-list item, with the red→green record and the budget reassessment |
| `backend/agent/src/ai/role.go` | **AI-05.1's deliverable.** The closed role vocabulary, its rendering, its parser and its enumeration |
| `backend/agent/src/ai/role_test.go` | AI-05.1's four test-list items, in `ai_test` |
| `backend/agent/src/ai/message.go` | **AI-05.2 and AI-05.3's deliverable.** The content seam, message identity, the message and its construction rules |
| `backend/agent/src/ai/message_test.go` | AI-05.2's and AI-05.3's five test-list items, in `ai_test` |

**Not touched:** `validation.go`, `validation_test.go`, `doc.go`, `import_boundary_test.go`, `src/agenttest/`, `go.mod`, `go.work`, and `openspec/specs/ai-contract-vocabulary/spec.md`.

### The five choices this proposal commits to

AI-05 has no `[decision]` node, so there is no `decision.md` and every choice below is answerable to a test in `spec.md`. One line each, so a reviewer can accept or reject the substance before reading `design.md`.

1. **Three roles: user, assistant, tool — and deliberately not `system`.** Each of the three has a citable case; `system` has none, because `V-REQ-19` gives the system instruction its own region of the request, owned by AI-10.2. The vocabulary is append-only, so a milestone that meets the case adds a constant; AI-04 set this precedent with its sixth rule class.
2. **The vocabulary is closed at the boundary, not merely documented.** A role outside it fails construction with `ErrNotInVocabulary`, and the parser accepts only the exact lowercase rendering — not a case-folded, trimmed or aliased one.
3. **Identity is minted by the constructor, opaque and comparable.** A caller cannot supply, forge or recompute it. This makes "distinguished from another" structural rather than a rule someone else writes, and it makes AI-05.2 item 1 a real assertion rather than a tautology.
4. **Copy in and copy out, both proven.** Construction clones the content sequence; every read returns a fresh one. These are two mechanisms and a design that does one produces the same confusing failure from the other side.
5. **Content is held through a marker seam that carries no payload, no kind, no accessor and no constructor.** It names a position in the contract and nothing else, so AI-06.1 inherits both of its properties undecided. This is the choice § "The content-part seam" defends.

### Out of scope (explicit, and deferred-but-related is called out)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| What a content part **is** — payload, kind, accessor, construction rules, sealing | **AI-06** | AI-06.1 decides readability and zero-value invalidity together, before any code exists. Anything landed here makes that clause false |
| Rejecting a nil or unconstructed content element | **AI-06.3** item 1 | It is that item verbatim, and it cannot be written before "unconstructed" has a meaning |
| Whether a role may carry a given content kind; whether a tool result may appear in a non-tool role | **AI-10.3** | AI-05.1's own out-of-scope clause. A request-level rule that needs kinds |
| Message collections, their order across a request, their repair | **AI-10**, Layer 2 | `V-OUT-02`: Layer 1 owns the unit, never the collection |
| Uniqueness of identity across a request | **AI-10** | AI-05 makes two messages distinguishable; "no duplicates in this request" is a composition rule at request scope |
| Equality of two messages | **AI-10.6** item 3 | "the documented equality" is named there and is defined over a request's regions |
| Merging consecutive same-role messages | adapter (**AI-26**) | doc 0001 § 3.3 row 5 marks it adapter-local |
| A `system` role | **the milestone that meets the case** | No citable case. Additive to add, breaking to remove |

## The content-part seam — read this before `design.md`

AI-05 owns "ordered content". AI-06 owns what a content part is. doc 0002 puts them in that order deliberately, and the risk it creates is that AI-05 defines a content type in passing and hands AI-06 a conclusion instead of a question.

**What AI-05 lands is a marker interface with a single unexported method and nothing else** — no payload, no kind, no accessor, no constructor, no rendering, no length. It names the position a content part occupies in a message's ordered content, and it is the whole of what AI-05.2's ordering test and AI-05.3's copy test require.

Three claims a reviewer should check, because the seam is only worth what they are worth:

1. **It leaves `V-REQ-06` readability undecided.** The seam exposes nothing to read. AI-06 may put accessors on the interface, on concrete variants, or on neither and use a walk — all three remain open.
2. **It leaves `V-REQ-07` sealing undecided.** The seam validates nothing. A nil element is accepted today because AI-06.3 item 1 is the assertion that rejects it, and it needs AI-06.1's strategy to know what it is rejecting.
3. **It is not itself a seal, and that is deliberate.** An unexported method is satisfiable from another package by embedding the interface — AI-05's own external tests do it, in four lines. AI-06.3 item 2 says the seal may be "the compiler or validation, whichever the AI-06.1 strategy chose"; landing the compile-time half here would take that choice. `design.md` § 4 records the bypass in the seam's own GoDoc so AI-06 inherits a known-open door rather than discovering it.

If a reviewer concludes the seam does decide something for AI-06, the correct response is doc 0002's revert-and-record clause — a new node, its edge, and a dated amendment in this same PR — not a smaller seam bolted on afterwards.

## Approach

1. **Implement the register, do not redesign it.** Every property proven below is a clause already in `V-REQ-01`, `V-REQ-02` or `V-REQ-03`. The tests are those clauses made mechanical.
2. **Borrow the standard library's closed-vocabulary shape and refuse its tolerance.** Integer defined type, `iota` constants, an **indexed slice** table, ugly rendering for an unregistered value — then, unlike `time.Month`, reject the unregistered value at every boundary it can enter through.
3. **Make the exhaustiveness pin structural, and design it for reuse.** The pin enumerates the *constant space*, not the name table, so a constant added without a table entry is caught. AI-07, AI-08 and AI-13 each land a closed vocabulary; `design.md` § 3 states the pattern as four rules so they inherit it instead of re-deriving it.
4. **Prove copy semantics in both directions and by the mechanism that actually fails.** The variadic slice-spread aliasing hazard is the specific bug; the red test reproduces it rather than asserting immutability in the abstract.
5. **Keep the seam thinner than it is comfortable to.** Every method not on the seam is a decision left to AI-06. The discomfort is the design working.
6. **Land nothing without a citable case.** Three roles, two rules, one identity type. `system`, an `IsValid` predicate, a message equality and a content accessor are each plausible, each have no citable case, and none is landed.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-message-roles/` | Five new markdown files | None — new directory |
| `backend/agent/src/ai/role.go`, `role_test.go` | New files | Low — a closed vocabulary is a well-understood shape, and the pin guards its growth |
| `backend/agent/src/ai/message.go`, `message_test.go` | New files | **Medium** — five milestones build on the message, and the seam is inherited by the keystone milestone |
| `backend/agent/src/ai/validation.go`, `validation_test.go` | **None** — read-only. AI-13 is being built concurrently and shares those files | — |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **None.** `explore.md` § 6 records the check and the three near-misses | — |
| `backend/agent/go.mod`, `go.work` | **None** — zero requires preserved | — |
| `backend/agent/src/agenttest/` | **None** — reserved for AI-06.2 | — |
| doc 0002 | **None expected.** Any amendment required by the revert-and-record clause lands in this same PR and is reported prominently | — |
| `docker-compose.yaml`, `infra/`, the other two backend modules | **None** | — |

## Rollback plan

The change is additive: four new Go files, one new change directory, and nothing modified. Rollback is `git revert` of the commit range; no build depends on the markdown, and `src/ai` returns to exporting AI-04's surface alone.

Post-merge reversal has the shape worth stating. The role vocabulary is cheap to grow and expensive to shrink, which is why `system` is not landed. The seam is cheap to replace *by AI-06 specifically* — it has no implementations outside tests, so widening or replacing it is a change to one file — and expensive to replace after AI-06 has built four variants on it. That asymmetry is why the seam is thin: AI-06 is the last moment it is free.

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| The seam decides AI-06.1 by accident | Medium | **High** — it is the keystone of wave 1, and C1/C2 both came from deciding these properties separately | `R-AMR-009` makes it checkable: the seam exposes no payload, kind, accessor, constructor or rendering. The three claims in § "The content-part seam" are each a review step |
| The role vocabulary is wrong — `system` is needed | Medium | Low | Additive. The pin forces the table update in the same commit as the constant, so a half-added role cannot merge |
| Copy semantics proven in one direction only | Medium | **High** — doc 0002 names it the most confusing class of failure in a streaming package | Two separate test-list items, two separate mechanisms, and the red test reproduces the variadic aliasing hazard specifically |
| Minted identity is read as defect **C3** returning | Medium | Low | `design.md` § 5.2 argues the distinction: C3's contract was about the counter's *value* per stream; `V-REQ-03` says nothing about the value, only that two messages differ |
| Identity minting is a data race under `-race` | Low | Medium | `sync/atomic`, and a concurrent construction subtest under the race detector rather than an inspection claim |
| The milestone grows past its review budget | **High** | Medium | Expected, as in AI-04. The surface is fixed by `design.md` before the first test; `tasks.md` carries the reassessment and the commit split already falls on the node boundary |
| A red step is skipped because Go will not compile a test against a missing symbol | Medium | Medium | AI-04's `design.md` § 6 convention, restated in `design.md` § 8: land the narrowest declaration that compiles and **fails**, record the failure, then implement |

## Dependencies

- **AI-00** — the module, the package, both import guards. Both must still pass afterwards.
- **AI-01** — the register. Every noun used here is one of its rows; **no amendment is needed**, and `explore.md` § 6 records why.
- **AI-04** — the taxonomy. This change is its first consumer and uses it exactly as its `tasks.md` § Next predicted.
- **No new Go dependency, and no ADR required.** `slices`, `strconv`, `strings` and `sync/atomic` are standard library.

## Success criteria

1. Every test-list item of AI-05.1, AI-05.2 and AI-05.3 is taken red → green → refactored, in order, with both outputs recorded. The pin is exempt from red-first and is **shown to bite**.
2. `make test` in `backend/agent/` is green with `-race`, and its output is recorded.
3. `make lint` is clean.
4. Both import-boundary guards still pass and `go.mod` still carries zero requires.
5. A message is constructible with each vocabulary role and the role reads back exactly.
6. A role outside the vocabulary fails construction with `ErrNotInVocabulary` at the role's position.
7. Each role's rendering is stable, lowercase, and round-trips through parse-and-render; a rendering that is not exactly a registered one does not parse.
8. The exhaustiveness pin fails when a role constant is added without a table entry — recorded, then removed.
9. A message's identity is comparable, non-zero, distinct per message, and identical across repeated reads.
10. Content order round-trips exactly, including a sequence holding the same element more than once.
11. A message with no content fails with `ErrEmpty` at the content position.
12. Mutating the slice a caller passed, or the slot it passed it in, does not change the message; mutating what a read returned does not change the message.
13. The seam exposes no payload, kind, accessor, constructor or rendering, and no register term is defined locally.

## Notes for the following phases

- **`spec.md`** — requirement IDs `R-AMR-0NN`, scenarios `S-AMR-0NN`. The system under test is runtime behavior, as in AI-04; every scenario is independently verifiable by a test.
- **`design.md`** — owns every Go spelling, the seam's exact shape and its GoDoc, and the four-rule closed-vocabulary pattern AI-07, AI-08 and AI-13 reuse. It also restates AI-04's red-step convention.
- **`tasks.md`** — three phases, one per node; one task per test-list item; red and green evidence per item; the pin's bite proof; the budget reassessment; the verification pass last.
