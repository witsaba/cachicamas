# Proposal — tool calls and tool results

> **Change**: `cachicamas-ai-tool-messages`
> **Milestone**: AI-09 — Define tool calls and tool results
> **Nodes**: AI-09.1 `[leaf]` · AI-09.2 `[leaf]` · AI-09.3 `[leaf]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: `openspec/changes/cachicamas-ai-tool-messages/`, **two new production files and one appended** under `backend/agent/src/ai/`, **two new test files** and **one appended** there. No `go.mod` change, no new dependency, **no new sentinel**, **no register amendment applied**
> **Predecessor artifacts**: `explore.md` (this change) · AI-06's `decision.md` (inherited whole)
> **Depends on**: AI-00 … AI-06
> **Blocks**: AI-10, AI-18, AI-26

---

## Intent

Land the two content-part variants that carry a model's tool invocation and the answer to it, so that
the ordinal a provider needs, the argument bytes an adapter must not rewrite, and the correlation a
session must be able to reload all exist **before** the first request type (AI-10) and the first
adapter (AI-26) are written against them.

Three properties carry the milestone, and each is one downstream milestone's precondition:

1. **Argument bytes are byte-exact.** AI-26 translates them onto a wire and AI-30 reassembles them
   from a stream; both compare bytes. A contract that re-marshalled would make a correct adapter
   impossible to write rather than merely awkward.
2. **The call ordinal is observable.** Several providers reject tool results that do not correspond
   positionally to their calls (doc 0001 § 2.3 item 5), and Layer 2's parallel-execution re-join is
   ordered by it (§ 7 **G5**).
3. **A tool that failed is content, not an error.** `V-REQ-18` says so in the register's own words. A
   failing tool is a normal outcome the model must see and reason about; routing it through AI-04
   would make a routine turn look like a caller's bug.

## Locked constraints (inherited, not proposed)

1. **The part strategy is AI-06's, entire.** `decision.md` § 9's inheritance table is cited, not
   re-derived. One `Part` type, kind derived from payload, `(T, bool)` accessors, rules on the
   payload, registration in `partKindNames` plus the guarded GoDoc list.
2. **The failure vocabulary is AI-04's five landed classes.** `explore.md` § 4 walks every rule this
   milestone has against them and appends none.
3. **The terms are AI-01's.** `V-REQ-16`, `V-REQ-17`, `V-REQ-18` and `V-STR-21` are used with their
   register definitions and cited, never paraphrased. One candidate clarification to `V-REQ-17` is
   **reported, not applied** — `explore.md` § 7.
4. **The module carries zero requires.** `encoding/json` is the standard library and the
   deny-by-default import guard admits it; `go.mod` is untouched.
5. **Strict TDD.** `openspec/config.yaml` sets `apply.tdd: true`. Every test-list item is taken
   red → green → refactored, in order, with discovered cases appended to the owning leaf's list.
6. **The evidence gate is recorded green `make test`** in `backend/agent/` (`go test -race -v ./...`),
   with `make lint` clean before every commit.
7. **Files owned by other milestones are not edited.** `validation.go`, `role.go`, `message.go`,
   `text_content.go` and every AI-07/AI-08 file are read-only here. `content_part.go` and the AI-06.4
   guard's witness table are **appended to** at their three and one designated points respectively,
   per AI-06 `decision.md` § 8's five-step procedure — twice, once per new kind.

## Scope

### In scope — one PR, five commits on the leaf boundary

| Artifact | Content |
| --- | --- |
| `explore.md` | The inheritance, the register check, the sentinel walk, the three candidate homes for the ordinal, the identity contrast with AI-05, the AI-08 § 8 reading, the budget |
| `proposal.md` | This file |
| `specs/ai-tool-messages/spec.md` | `R-ATM-001` … `R-ATM-011` with Given/When/Then scenarios |
| `design.md` | Every Go name, the ordinal's home and its consequence, the argument-encoding decision and its conceded cost, the failure representation, the file layout, how a red step is taken |
| `tasks.md` | AI-09.1/.2/.3 as phases, one task per test-list item, verbatim red and green evidence, the Review Workload Forecast |
| `src/ai/tool_call.go` | `ToolCall`, `NewToolCall`, `Part.ToolCall`, `ID`/`Name`/`Arguments`, the redacting renderers, and `ToolCalls` — the ordinal derivation |
| `src/ai/tool_result.go` | `ToolResult`, `NewToolResult`, `NewToolFailure`, `Part.ToolResult`, `CallID`/`Content`/`Failed`, the redacting renderers |
| `src/ai/content_part.go` | **Appended only**: two constants at the end of the `PartKind` block with `partKindEnd` moved, two `partKindNames` entries, two documented-list lines |
| `src/ai/content_part_registry_test.go` | **Appended only**: two witnesses, three legs each |
| `src/ai/tool_call_test.go` | AI-09.1 and AI-09.2, in `ai_test` |
| `src/ai/tool_result_test.go` | AI-09.3, in `ai_test` |

### Out of scope, each with the node that owns it

| Deliberately not done | Owner |
| --- | --- |
| Validating argument bytes against the declared tool's schema | Nobody in Layer 1 — `V-REQ-13`, and `ErrMalformed`'s own GoDoc says so |
| Executing a tool, resolving a name to an implementation, scheduling a result | `V-OUT-04`; Layer 2 schedules, Layer 3 confines |
| Enforcing that every call in a transcript has a result | `V-OUT-02` — Layer 2's history invariant |
| Enforcing that a result correlates to a call **within one request** | AI-10.3 item 4, by name |
| Whether a given role may carry a tool call or a tool result | AI-10.3 item 3, by name |
| Uniqueness of call identities across a collection | AI-10.3 — `explore.md` § 4 records why it is not stretched onto an existing class here |
| A byte ceiling on argument bytes or result content | Deferred with reasons — `design.md` § 7 |
| Tool-call **delta** events and their optionality | AI-18 (leakage row 1) |
| Any wire representation, and the synthetic-identifier mapping itself | AI-26, AI-30; leakage row 7 is adapter-local |
| The reasoning kind | AI-07, landing concurrently against the same three append points |
| Tool **declarations**, the tool set and tool choice | AI-08, landing concurrently |

## Approach

Three leaves, in dependency order, each closing on recorded green `make test`.

- **AI-09.1** builds `ToolCall` as an exported opaque value type under AI-06 `decision.md` § 6.3, its
  constructor, its accessor, and its three rules. The byte-fidelity item uses a fixture whose key
  order and interior whitespace a JSON marshal round trip would rewrite, so a canonicalizing
  implementation fails rather than passes for the wrong reason.
- **AI-09.2** derives the ordinal from position rather than storing it, and exports one function over
  a content sequence. `explore.md` § 5 walks the three candidates; `design.md` § 5 states the choice
  and both of its consequences.
- **AI-09.3** builds `ToolResult` with two constructors — success and failure — so that a failing
  tool is a *state on a value* rather than an error anyone has to route.

Registration is the last step of each of the two kind-adding commits, because AI-06.4's guard is what
proves the five-step procedure was completed and it must be seen to bite before it is seen to pass.

## Risks and rollback

| Risk | Mitigation |
| --- | --- |
| **Merge collision with AI-07**, which appends its own kind to the same three points in `content_part.go` and to the same witness table | Everything is appended at the end of each list; nothing existing is renumbered, reordered or reworded. AI-06.4's guard fails on a bad merge in either direction — a constant without a table entry, a table entry without a documented line, a witness without a constant — which is the reason the guard exists |
| **The canonical empty-argument form is a normalization**, and `V-REQ-17` forbids normalizing | Resolved narrowly and recorded: fidelity binds *supplied* bytes; *absent* arguments are not supplied bytes. `R-ATM-003` pins both halves with tests, and the register clarification is reported to the orchestrator rather than applied here |
| **The JSON encoding is hard-coded**, so a future non-JSON argument dialect is a breaking change | Conceded openly in `design.md` § 4. It is the same cost AI-08 declined to pay for schemas, paid here for the reason doc 0002 and AI-04's `ErrMalformed` GoDoc both give: argument bytes are model-produced and stream-reassembled |
| **The ordinal derivation is a package-level function**, a shape AI-06 `decision.md` § 4.3 argued against for *accessors* | The objection there was two call shapes for one question about **one part**. A sequence has no `Part` method to hang it on, and this milestone does not own `message.go`. `design.md` § 5 records the trade and the more-general shape it buys |

**Rollback** is `git revert` of this change's commits. Nothing outside `backend/agent/src/ai/`
imports the new surface; `go.mod`, the module graph and every landed contract are untouched, and the
three appended points in `content_part.go` revert to exactly AI-06's text.

## Acceptance

Doc 0002's charter, restated as a checklist:

- [ ] Argument bytes round-trip **byte-equal** from an external package, through a message.
- [ ] A result's correlation to its call round-trips exactly, including a synthetic identity.
- [ ] Ordinal position is observable, and stable across reads and copies.
- [ ] Construction rules fail through AI-04 sentinels; a call with empty arguments is constructible.
- [ ] A tool failure is distinguishable from a success without an error.
- [ ] Both kinds carry a three-leg witness in AI-06.4's guard, and `make test` is green.
- [ ] `make lint` is clean and `go.mod` still carries zero requires.
