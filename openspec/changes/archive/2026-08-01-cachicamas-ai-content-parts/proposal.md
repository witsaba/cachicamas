# Proposal — content parts: readable and sealed

> **Change**: `cachicamas-ai-content-parts`
> **Milestone**: AI-06 — Define content parts: readable and sealed · **the keystone of Wave 1**
> **Nodes**: AI-06.1 `[decision]` · AI-06.2 `[leaf]` · AI-06.3 `[leaf]` · AI-06.4 `[guard]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: `openspec/changes/cachicamas-ai-content-parts/`, **two new production files and one modified** under `backend/agent/src/ai/`, **three new test files** there, **one new test file** in `backend/agent/src/agenttest/`, and a mechanical migration of AI-05's two test files. No `go.mod` change, no new dependency, **no register amendment**
> **Predecessor artifacts**: `explore.md`, `decision.md` (this change)
> **Depends on**: AI-00 … AI-05
> **Blocks**: AI-07, AI-08, AI-09, AI-10, and — through readability — AI-26

---

## Intent

Land **one content-part strategy** that makes every variant's payload readable from another package *and* every unconstructed value invalid, prove it on the text variant, and guard the registration so a later kind cannot be half-added.

The retired Layer 1 needed two corrective milestones here — one for readability, one for the construction bypass — because the two properties were decided separately and pulled in opposite directions, and it ended with **two strategies for one contract**: a sealed-but-unreadable wrapper for three payload-carrying types and a readable-but-unsealed direct implementation for reasoning. Doc 0001 § 3.1 records that the two "had to be reconciled". This milestone decides them together, once, before a second variant exists to inherit the wrong answer.

## Locked constraints (inherited, not proposed)

1. **The five terms are AI-01's.** `V-REQ-04` … `V-REQ-08` are used with their register definitions and cited, never paraphrased.
2. **The failure vocabulary is AI-04's.** Every rule violation reports through `Invalid` with one of the five landed rule classes, at a position, composed by `FirstFailure`. **No new sentinel** — `decision.md` § 7.3 records why.
3. **Both properties of `V-REQ-04` are constitutive.** A strategy satisfying one and not the other has failed the milestone; doc 0002 says so in the charter note.
4. **The module carries zero requires** until AI-24 and AI-37, each behind its own ADR gate.
5. **Strict TDD.** `openspec/config.yaml` sets `apply.tdd: true`; every test-list item is taken red → green → refactored, **in order**, with discovered cases appended rather than chased.
6. **The evidence gate is recorded green `make test`** in `backend/agent/` (`go test -race -v ./...`), plus a **recorded red run** against a scratch violation for the guard leaf.
7. **The cross-package proof lands in `src/agenttest/`.** That package's file comment reserves the first cross-package readability proof for AI-06.2 by name.

## Scope

### In scope — one PR, six commits

| Artifact | Content |
| --- | --- |
| `explore.md` | What exists, the seam AI-05 left open, the four tensions, the prior-art scan, the vocabulary check, the budget forecast |
| `decision.md` | **AI-06.1.** The strategy, both properties demonstrated bypass by bypass before any code exists, five alternatives at full strength, the accessor shape, the procedure for adding a kind, the conceded costs |
| `proposal.md` | This file |
| `specs/ai-content-parts/spec.md` | `R-ACP-001` … `R-ACP-012`, requirements over runtime behavior with Given/When/Then scenarios |
| `design.md` | The Go shapes — every type, function and constant name — the seam closure, the validation order, the file layout, and how a red step is taken |
| `tasks.md` | AI-06.2/.3/.4 as phases, one task per test-list item, with the red→green record, the guard's recorded bite, and the budget reassessment |
| `src/ai/content_part.go` | **The strategy.** `Part`, `PartKind`, the kind vocabulary and its table, `Kind()`, `PartKinds()`, the unexported payload contract, and the validation path AI-10 reuses |
| `src/ai/text_content.go` | **The first subject.** `MaxTextLen`, the text payload, `NewText`, `Part.Text()` |
| `src/ai/message.go` | **The seam closure.** `Content` removed; `Message` holds `[]Part`; construction validates each element |
| `src/ai/content_part_test.go` | AI-06.2 items 2–4 and AI-06.3 items 1–2, in `ai_test` |
| `src/ai/content_part_internal_test.go` | AI-06.3 item 3 — the pinned reuse point for AI-10, in `ai` |
| `src/ai/content_part_registry_test.go` | **AI-06.4.** The registration guard and its AST scan of the documented kind list, in `ai` |
| `src/ai/testdata/…` | Two scratch programs the compile-time half of AI-06.3 item 2 builds — one that must fail, one that must succeed |
| `src/agenttest/content_part_test.go` | **AI-06.2 item 1.** The external-package round trip |
| `src/ai/role_test.go`, `src/ai/message_test.go` | Migration: the embedded content helper becomes a constructed text part; `[]ai.Content` becomes `[]ai.Part` |

### Out of scope, each with the node that owns it

| Deliberately not done | Owner |
| --- | --- |
| Reasoning content, its state vocabulary and its round-trip token | AI-07 |
| Tool declarations | AI-08 |
| Tool call and tool result variants | AI-09 |
| The request, its single validation pass, and which role may carry which kind | AI-10.3, AI-10.4 |
| Cache-boundary markers on a part | AI-11 |
| Image and audio kinds | deliberately producerless — `V-PRV-07`, doc 0001 § 8 |
| Any wire encoding, JSON tag or serialization | AI-24 onward |
| A parser for a kind's rendering | no citable case; see `explore.md` § 8 |

## What changes on the exported surface

**Added**

| Symbol | Why |
| --- | --- |
| `Part` | `V-REQ-04`. One opaque value type, every kind |
| `PartKind`, `PartKindText`, `PartKinds()` | `V-REQ-05`. The closed, enumerable discriminator |
| `NewText(string) (Part, error)` | `V-REQ-08`. The only door in |
| `Part.Kind() PartKind` | `V-REQ-05`. Derived from the payload |
| `Part.Text() (string, bool)` | `V-REQ-06`. The accessor shape all later variants inherit |
| `Part.String() string` | Diagnostic rendering that names the kind and **never the payload** — AI-04's redaction posture, applied to the first payload-carrying type in the package |
| `MaxTextLen` | The documented bound whose breach is `ErrOutOfRange` |

**Removed**

| Symbol | Why |
| --- | --- |
| `Content` | It is the seam AI-05 landed **explicitly for this milestone to close**. Leaving it — even as an alias — leaves the embedding bypass alive, which is the defect. AI-05's `design.md` § 4.3 records the door and its dimensions; `decision.md` § 3.3 bypass 4 records the closure |

**Changed**

| Symbol | Before | After |
| --- | --- | --- |
| `NewMessage` | `(Role, ...Content) (Message, error)` | `(Role, ...Part) (Message, error)`, with a third rule: every element is a constructed part |
| `Message.Content()` | `[]Content` | `[]Part` |

## The two properties, and why one PR proves both

Doc 0002 makes them constitutive, so splitting them across PRs would ship a half-contract — which is exactly the state **C1** and **C2** came from. The PR is nevertheless reviewable in commit order, and the commits fall on node boundaries:

1. `docs(sdd)` — the planning artifacts, `decision.md` first among them.
2. `refactor(ai)` — the seam closure and the migration of AI-05's tests, with stub accessors. Green.
3. `feat(ai)` — AI-06.2, the text variant: readable, discriminated, ruled, unaltered.
4. `test(ai)` — AI-06.3, the seal: zero value, hand-rolled, and the pinned request-path reuse.
5. `test(ai)` — AI-06.4, the guard, with its recorded bite.
6. `docs(sdd)` — `tasks.md` with the full red/green record.

## AI-06.3 item 3 — the request path, resolved without building AI-10

Doc 0002 asks that an unconstructed part be rejected "through the request path instead of the message path", and **AI-10 does not exist**. The instruction is explicit that no speculative request type may be built. Two routes were available; this change takes the first, and `tasks.md` records the choice as required.

**Chosen: express the rule as something AI-10 reuses, and pin the reuse point.** The element rule lives in a package-internal function

```go
func validateContent(prefix Path, content []Part) *Violation
```

whose `prefix` parameter exists for exactly one caller that does not exist yet. `NewMessage` calls it with no prefix, producing `content[0]`. AI-10 will call it with `AtIndex("messages", i)`, producing `messages[2].content[0]`. The reuse point is pinned by a test that calls it with a request-shaped prefix and asserts the rendered position, so AI-10 inherits a function with a proven contract rather than a comment asking it to remember one.

**Not chosen: revert-and-record.** The clause fires when a leaf's first red test cannot be driven green in small steps. This one can: AI-10 and AI-06 live in the same package, so "the request path" is not a missing seam, it is a caller that has not arrived. Inventing a `Request` type to satisfy the letter of item 3 would violate the milestone rule "one primary contract or behavior per milestone" and would freeze a shape AI-10 has four leaves to decide. **No doc 0002 amendment is proposed by this change.**

## Risks, and what each is traded against

| Risk | Mitigation |
| --- | --- |
| **Removing `Content` breaks AI-05's tests.** | It is the intended disposition of `R-AMSG-009`'s second scenario, which AI-05 wrote as a placeholder. The migration is mechanical, in this diff, and the tests keep asserting the same properties |
| **Concurrent milestones touch the same package.** AI-08 (tool declarations) and AI-13 (finish reasons, usage) are in flight | Disjoint file sets by construction: this change adds `content_part*.go` and `text_content.go` and modifies only `message.go` and AI-05's two test files. It reads `validation.go` and `role.go` and edits neither |
| **One Go type for every kind removes type-level narrowing.** | Conceded openly — `decision.md` § 10, the largest concession. Layer 1's consumers loop over heterogeneous content, so a narrowed signature is one the loop cannot call |
| **The compile-time proof shells out to the Go toolchain.** | Precedented: AI-00's import guard already runs `go list -deps -test`. The scratch programs live under `testdata/`, which the go tool and golangci-lint both exclude from `./...` |
| **The milestone will exceed the review budget.** | Forecast in `explore.md` § 7 rather than discovered afterwards, reassessed in `tasks.md`, and split across six commits on node boundaries so the PR can be reviewed in passes or chained without rework |

## Rollback plan

Every commit is additive or mechanically reversible, and nothing outside `backend/agent/src/` is touched.

1. **Full rollback** — revert the six commits in reverse order. `git revert` restores `Content`, the `[]Content` seam and AI-05's helper verbatim; the package returns to AI-05's landed state, whose tests are unchanged in substance.
2. **Partial rollback of the guard alone** — delete `content_part_registry_test.go`. The contract stands; only the exhaustiveness check is lost, and AI-07 would re-introduce it.
3. **Partial rollback of the compile proof alone** — delete the `testdata/` programs and their test. AI-06.3 item 2's runtime half (the zero value, however obtained, is rejected) survives on its own.
4. **No data, no migration, no deployment.** The module has no `main` package until doc 0004, no dependency, and nothing persisted. The blast radius is the compile graph of one module that nothing outside the repo imports.

## Acceptance

Doc 0002's charter, restated as the conditions this change must meet:

1. An **external-package** test constructs a text part, places it in a message, and reads the exact text back — byte-equal, no re-encoding.
2. The same external package **cannot** smuggle a zero-value or hand-rolled part into a message: the compiler rejects the hand-rolled shapes and validation rejects the zero value with an AI-04 sentinel.
3. The kind reported by a part always matches the payload it yields, because there is only one of them.
4. The registration guard bites — recorded red against a scratch kind missing one leg, and against a documentation list that drifted from the table.
5. `make test` and `make lint` are clean in `backend/agent/`; both AI-00 import guards pass; `go.mod` still carries zero requires.
