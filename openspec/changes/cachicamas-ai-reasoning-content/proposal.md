# Proposal — reasoning content and its round-trip token

> **Change**: `cachicamas-ai-reasoning-content`
> **Milestone**: AI-07 — Define reasoning content with its round-trip token
> **Nodes**: AI-07.1 `[leaf]` · AI-07.2 `[leaf]` · AI-07.3 `[leaf]` · AI-07.4 `[leaf]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: `openspec/changes/cachicamas-ai-reasoning-content/`, **one new production file** and **three appends to one existing file** under `backend/agent/src/ai/`, **one new test file** and **one append to the AI-06.4 guard's witness table**. No `go.mod` change, no new dependency, **no register amendment, no new rule class**
> **Predecessor artifacts**: `explore.md` (this change) · [AI-06's `decision.md`](../cachicamas-ai-content-parts/decision.md), inherited whole
> **Depends on**: AI-06
> **Blocks**: AI-17 (reasoning blocks on the stream), AI-29 (their wire normalization)

---

## Intent

Land the **second content-part kind**: reasoning, carrying a state, optional text, and an opaque provider token that survives construction and readback **byte for byte**. Absorb what the retired plan deferred to a later breaking change (**G12(b)**), on the milestone that first states the contract rather than on one that would have to break it.

The token is not metadata. Doc 0001 § 3.2 puts it in the register of things that cannot wait, and doc 0001 § 6 makes it seam 11 — a place where a decision can be inserted later without reshaping everything around it, whose trivial v1 implementation is "empty" and whose reason for existing now is *"Correctness, not metadata: without it, multi-turn extended thinking with tool use fails."*

## Locked constraints (inherited, not proposed)

1. **The strategy is AI-06's, cited and not re-derived.** `decision.md` § 9 is a table written for this milestone to read a row from. One part type, one accessor shape, one registration procedure, one rule set with two entry points.
2. **The three terms are AI-01's.** `V-REQ-09`, `V-REQ-10`, `V-REQ-11` are used with their register definitions and cited, never paraphrased.
3. **The failure vocabulary is AI-04's.** No new sentinel, and no stretching of an existing one: `explore.md` § 6 records that the one violation which would have needed a new class is designed out rather than reported.
4. **The five-step procedure is AI-06.4's**, and its guard is the thing that fails when a step is missed.
5. **The module carries zero requires**, until AI-24 and AI-37, each behind its own ADR gate.
6. **Strict TDD**, `apply.tdd: true`. Every test-list item red → green → refactored, in order, with discovered cases appended rather than chased.
7. **The evidence gate is recorded green `make test`** in `backend/agent/` (`go test -race -v ./...`), plus `make lint`.

## Scope

### In scope

| Artifact | Content |
| --- | --- |
| `explore.md` | What exists, the four questions AI-07 actually owns, the prior-art scan, the vocabulary check, the budget forecast |
| `proposal.md` | This file |
| `specs/ai-reasoning-content/spec.md` | `R-ARC-001` … `R-ARC-009`, requirements over runtime behavior with Given/When/Then scenarios |
| `design.md` | The Go shapes, the derivation of the state, the absence-versus-empty representation, the rule order, and how each red step is taken |
| `tasks.md` | AI-07.1 … AI-07.4 as phases, one task per test-list item, with the verbatim red→green record and the budget reassessment |
| `src/ai/reasoning_content.go` | **The kind.** `ReasoningState` and its closed vocabulary, `Reasoning`, `NewReasoning`, `NewRedactedReasoning`, `Part.Reasoning`, `MaxReasoningTokenLen` |
| `src/ai/content_part.go` | **Three appends only** — the constant, the table entry, the documented list line. Nothing existing is renumbered or reordered |
| `src/ai/reasoning_content_test.go` | All four leaves, in `ai_test` |
| `src/ai/content_part_registry_test.go` | **One append** — the witness entry with its three legs |

### Out of scope, each with the node that owns it

| Deliberately not done | Owner |
| --- | --- |
| The token's survival through request rebuild | AI-12.1 |
| The token's survival through the wire, outbound and inbound | AI-26.6, AI-29.2 |
| Reasoning blocks and deltas on a stream | AI-17 |
| Reasoning token **counts** in usage — a different meaning of the word | AI-13.3 |
| Which role may carry a reasoning part | AI-10.3 |
| Tool call and tool result kinds | AI-09 |
| Any interpretation, verification or decryption of a token | nobody — `V-REQ-11` forbids it at every layer |

## What changes on the exported surface

**Added**

| Symbol | Why |
| --- | --- |
| `PartKindReasoning` | `V-REQ-05`. The second member of the kind vocabulary |
| `ReasoningState`, `ReasoningStateText`, `ReasoningStateRedacted`, `ReasoningStateTokenOnly`, `ReasoningStates()` | `V-REQ-10`. The closed state vocabulary, in the shape `role.go` documents as "the pattern every later closed vocabulary in this package reuses" |
| `Reasoning` | `V-REQ-09`. The payload value type, opaque, with unexported fields — `decision.md` § 6.3's rule for a payload that carries structure |
| `Reasoning.State()`, `Reasoning.Text()`, `Reasoning.Token() ([]byte, bool)` | The payload's readers. The second result of `Token` is `V-REQ-11`'s absent-is-not-empty |
| `NewReasoning(text string, token []byte) (Part, error)` | The door for the text and token-only states, whose state is derived from what it is given |
| `NewRedactedReasoning(token []byte) (Part, error)` | The door for the one state that cannot be derived from the bytes |
| `Part.Reasoning() (Reasoning, bool)` | `V-REQ-06`. The accessor shape `decision.md` § 6.1 named for this milestone, spelled exactly as it wrote it |
| `MaxReasoningTokenLen` | The documented sanity bound whose breach is `ErrOutOfRange` |

**Removed / changed**: nothing. Every edit to `content_part.go` is an append at the end of a list.

## Two doors, and why not one

A single `NewReasoning(state, text, token)` would take a state the caller supplies, and would then have to reject the combinations that contradict themselves — a redacted part carrying plaintext, a token-only part carrying text. Reporting a contradiction between two fields needs a rule class **AI-04's five do not name**, so that design's real cost is a sentinel append for a state the type never needed to be in.

Deriving the state removes it. Text presence decides between *text* and *token-only*; redaction is not derivable from bytes and gets its own door. This is `decision.md` § 5 applied one level down — *the two cannot disagree, because there is only one of them* — and it is why this change appends nothing to `validation.go`.

## Risks, and what each is traded against

| Risk | Mitigation |
| --- | --- |
| **A concurrent milestone edits the same three places.** AI-09 adds two kinds to the same constant block, the same table and the same GoDoc list | Every edit is an **append at the end**; nothing existing is renumbered or reordered. AI-06.4's guard cross-checks the declared constant space against `PartKinds()` and `partKindNames` in both directions, so a merge that drops or misorders an entry fails the build rather than shipping |
| **The token aliases the caller's buffer.** A stored `[]byte` shares its backing array with whatever the caller reused | Proven red before it is fixed — AI-07.3 item 2 exists for this, and AI-08 hit the identical bug on schema bytes. The fix must hold on **both** paths: construction and readback |
| **A `[]byte` field makes `Part` non-comparable**, and `content_part.go` documents that "equality with `==` is defined and compares payloads" | A landed property of `Part` must not regress. `design.md` § 3.3 records the storage that keeps it |
| **The state vocabulary is derived, so a caller cannot set it** | That is the point, and it is the same trade `Part.Kind()` already made. `design.md` § 2.3 states the concession openly |
| **A documented sanity bound looks like a provider limit** | It is not, and `MaxTextLen`'s GoDoc already says so for text. The reasoning bound repeats the reasoning rather than assuming a reader will carry it across files |

## Rollback plan

1. **Full rollback** — revert the commits in reverse order. `reasoning_content.go` and its test disappear; `content_part.go` loses three appended lines and the guard loses one witness. The package returns to AI-06's landed state.
2. **Partial** — there is none worth having. A kind whose constant is declared without its file fails AI-06.4 by design.
3. **No data, no migration, no deployment.** The module has no `main` package, no dependency, and nothing persisted.

## Acceptance

Doc 0002's charter, restated as the conditions this change must meet:

1. A reasoning part is constructed and read back through AI-06's strategy, **exactly like text** — one part type, one accessor shape, no special case.
2. Reasoning and text are structurally distinct: no accessor yields one as the other.
3. All three reasoning states are constructible, and the vocabulary is closed, enumerable and immutable by a consumer.
4. A token's **absence** is distinguishable from an **empty** token, through the exported surface.
5. Tokens that are not valid UTF-8, not valid JSON and not printable are stored and returned **byte-identically**, through a message, across a copy, and after the caller mutates the slice it supplied.
6. The redacted and signature-only shapes are constructible, valid, byte-exact, and confusable with nothing.
7. AI-06.4's guard is green with a witness carrying all three legs; `make test` and `make lint` are clean; both AI-00 import guards pass; `go.mod` still carries zero requires.
