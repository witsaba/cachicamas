# Proposal — the normalized request core

> **Change**: `cachicamas-ai-model-request`
> **Milestone**: AI-10 — Define the normalized request core
> **Nodes**: AI-10.1 `[leaf]` · AI-10.2 `[leaf]` · AI-10.3 `[leaf]` · AI-10.4 `[leaf]` · AI-10.5 `[leaf]` · AI-10.6 `[leaf]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: `openspec/changes/cachicamas-ai-model-request/`, **two new production files plus two new test files** under `backend/agent/src/ai/`, and **one appended rule class** in `validation.go` with its two registry mirrors. No `go.mod` change, no new dependency, **no register amendment**
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-04, AI-05, AI-06, AI-07, AI-08, AI-09
> **Blocks**: AI-11, AI-12, AI-20, AI-26

---

## Intent

Land `V-REQ-20` — the one value a provider receives. Model identity, an **ordered, segmented** system instruction, ordered messages, the tool set and tool choice, and generation options, in a sealed immutable value that validates **once, before any I/O** and exposes every region to a reader in another package.

Three of those properties cannot be retrofitted, and each was a shipped defect or a named gap before this plan existed:

1. **Segmented from birth.** Doc 0001 § 3.2 lists a flat system instruction first among the things that must change before an adapter exists, because a flat string has nowhere to put a cache boundary and collapses *absent* into *empty*. Segments cost nothing today and a breaking change tomorrow.
2. **Readable from another package.** Defect **C2** made request translation structurally impossible in the retired Layer 1. AI-06 made readability constitutive of a part; AI-10 makes it constitutive of the whole request, proven from `ai_test` on day one.
3. **The cross-region rules decided here.** Whether a tool result may sit in a user message, whether an orphan result is legal, and whether two calls may share an identity are three questions that become askable only when a request exists. Left undecided, the first adapter answers them by accident and every later adapter inherits the accident.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can tell the inherited from the decided.

1. **The terms are AI-01's.** `V-REQ-19` … `V-REQ-22`, `V-REQ-26` are used with their register definitions and cited, never paraphrased. This change *implements* them.
2. **The failure vocabulary is AI-04's.** Every violation reports through `Invalid` at a position, composed by `FirstFailure`. One class is **appended** under AI-04's own append rule; no local `errors.New`, no second failure type.
3. **The content rules are AI-06's, called not copied.** `validateContent(prefix, content)` is the request path's content validator, with the prefix AI-06.3 item 3 pinned.
4. **The tool-choice cross-validation is AI-08.3's, called not copied.** `ToolChoice.ValidateAgainst(ToolSet)`.
5. **There is no system role.** AI-05 landed three roles deliberately. The system instruction is its own region; rendering it as a message is an adapter's job.
6. **Trap 1 binds.** Layer 1 carries transport representations; it executes nothing, resolves no name, validates no schema.
7. **The module carries zero requires.** Two landed guards hold it there.
8. **Strict TDD**, `Test<Subject>_<Behavior>_<Expectation>`, banner comments citing the leaf ID, behavioral tests in the external package `ai_test`.
9. **The evidence gate is recorded green `make test`** in `backend/agent/` (`go test -race -v ./...`), plus clean `make lint`.

## Scope

### In scope

| Artifact | Content | Status |
| --- | --- | --- |
| `explore.md` | What exists, the two cashed seams, the four tensions, prior art, the vocabulary check | landed |
| `proposal.md` | This file | landed |
| `specs/ai-model-request/spec.md` | `R-AMR-001` … `R-AMR-016` over runtime behavior, with scenarios | landed |
| `design.md` | Every Go shape; the documented rule order; the three AI-10.3 dispositions with their reasoning; the option admission table; the seams left for AI-11 and AI-12 | landed |
| `tasks.md` | Six phases, one per leaf, one task per test-list item, with red/green evidence and the budget reassessment | landed, evidence through AI-10.3 |
| `src/ai/system_instruction.go` | **AI-10.2's deliverable.** `Segment`, `NewSegment`, `SystemInstruction`, `NewSystemInstruction`, `NewSystemText`, accessors, redaction | **implemented** |
| `src/ai/system_instruction_test.go` | AI-10.2's items, in `ai_test` | **implemented** |
| `src/ai/request.go` | **AI-10.1 and AI-10.3's deliverable.** `Request`, `NewRequest`, `RequestOption` and its seven constructors, the accessors, the validation order, the cross-region rules, redaction | **implemented through AI-10.3** |
| `src/ai/request_test.go` | AI-10.1's and AI-10.3's items, in `ai_test` | **implemented** |
| `src/ai/validation.go` | `ErrMisplaced` appended to the closed class set and to `ruleClasses` | **implemented** |
| `src/ai/validation_registry_internal_test.go`, `validation_test.go` | The two mirrors the internal guard requires, updated in the same commit | **implemented** |

### Deliberately **not yet** implemented — the second half of the milestone

This milestone is worked as two chained halves on the leaf boundary, because it is the largest composition in Layer 1 and two prior attempts died on transcript size. **AI-10.4, AI-10.5 and AI-10.6 are planned in full below and in `design.md`, and are not implemented.** The resuming agent's entry point is `tasks.md` § "Phase AI-10.4"; everything it depends on is landed and green.

| Node | What remains | Where the plan is |
| --- | --- | --- |
| AI-10.4 | First-failure determinism at request scope; totality over the regions; the no-I/O dependency-closure guard in the AI-00.3 style | `design.md` § 9, `tasks.md` phase AI-10.4 |
| AI-10.5 | The whole-request round trip from `src/agenttest/`, plus the exhaustive-kind pin over `PartKinds()` | `design.md` § 10, `tasks.md` phase AI-10.5 |
| AI-10.6 | Mutation of what a reader returned; mutation of what the constructor was passed; documented equality of two requests built from identical inputs | `design.md` § 11, `tasks.md` phase AI-10.6 |

### The six choices this proposal commits to

AI-10 has no `[decision]` node, so there is no `decision.md`; every choice is made in `design.md` and answerable to a scenario in `spec.md`. One line each.

1. **Required regions are parameters; optional regions are functional options.** `NewRequest(model string, messages []Message, opts ...RequestOption)`. Absence of an option is structural, so "temperature unset" and "temperature zero" are different requests. It is also the seam AI-12 needs: a per-request override is the same option applied again, and rebuild is one added method.
2. **The system instruction is its own value type holding ordered segments.** `NewSystemText(text)` produces a value **equal** to `NewSystemInstruction(seg)`; absent is zero segments; an empty or whitespace-only segment is rejected, so "one empty segment" is unrepresentable rather than merely equivalent to absence.
3. **Role versus content kind is enforced, from a table.** `tool_result` only in `tool`; `tool_call` and `reasoning` only in `assistant`; `text` in `user` and `assistant`. The reason is that an adapter maps *from* the neutral shape (doc 0001 § 3.3 row 4), and a neutral shape with two legal placements makes every adapter handle both. Violations report the appended `ErrMisplaced`.
4. **An orphan tool result is rejected; an orphan tool call is legal.** A result whose correlation matches no call anywhere in the request fails with `ErrUnresolvedReference` — the class's definition verbatim. The converse is deliberately legal, because a call awaiting its result is the ordinary mid-turn state and repairing orphaned *calls* is Layer 2's (`V-OUT-02`).
5. **Tool-call identities are unique across the request; duplicates are rejected with `ErrDuplicate`.** AI-09's `design.md` § 3.2 deferred this here and named the class. The reason is not tidiness: choice 4's rule resolves a result to a call *by identity*, so uniqueness is a precondition of a rule this milestone lands, not a preference.
6. **Four generation options, each admitted by `V-REQ-26`'s own test.** Maximum output tokens, temperature, top-p, stop sequences. Every rejected candidate is named in `design.md` § 8 with the provider that fails it, and its owner is AI-12's escape hatch.

### Out of scope (explicit)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Cache-boundary markers on segments, declarations or messages | **AI-11.1** | `design.md` § 12 records the seam left for them: markers attach to `Segment`, `Tool` and `Message`, and no request-level shape changes |
| The breakpoint cap and the tools → system → messages read order | **AI-11.2** | Markers do not exist yet |
| Copy-on-write rebuild, per-request overrides, the escape hatch | **AI-12** | `design.md` § 12 records the seam: `Request.With(...RequestOption)` and nothing reshaped |
| Translation to any wire shape | **AI-26** | AI-10.5's round trip is the property AI-26 consumes, not the translation |
| Executing a tool, resolving a tool name, judging a schema | Layers 2 and 3 | Trap 1 |
| The transcript invariant "every tool call has a matching result" | **Layer 2** | `V-OUT-02`. Only the request-level fragment — a result whose call is absent — is this milestone's, per doc 0002 AI-09's out-of-scope clause |
| Whether a call must appear *before* the result that correlates to it | **Deliberately unlanded**, recorded | `design.md` § 6.3. Doc 0002 scopes item 4 to existence "anywhere in the request"; ordering repair is `V-OUT-02`'s. Pinned by a test so the disposition is asserted rather than assumed |
| An upper bound on temperature | **Deliberately unlanded**, recorded | `design.md` § 8.3. Providers disagree (1.0 versus 2.0); a bound that is right for one is a caller-contract failure invented by this package for the other |
| A catalog check on the model identity | Nobody in Layer 1 | The register's own worked borderline case: an unrecognised model is a **provider** failure, not a caller-contract one. Only emptiness is decidable here |

## Approach

1. **Compose, do not re-derive.** Every rule this change runs on a region is the region's own rule, called. The only new rules are the ones that need two regions in the same value.
2. **Take the walking skeleton first and widen it.** AI-10.1 is one model identity and one user text message, readable from `ai_test`. AI-10.2 widens with the system region, AI-10.3 with messages, tools and choice; nothing opens a second unintegrated front.
3. **Make absence structural everywhere it appears.** Zero segments for an absent instruction, an unapplied option for an absent option, `ToolChoice.Mode() == 0` for an absent choice. No sentinel value means "unset" anywhere in this file.
4. **Decide the cross-region questions in writing before writing the test.** Each of the three has a table row, a reason drawn from a cited document, and a test that names the reason in its banner.
5. **Put the redaction posture on the type, not on the reviewer.** The request carries every payload in the package at once, which makes it the highest-value leak in Layer 1. `String` and `GoString` name regions and counts; a test asserts that a secret placed in each of six regions appears in none of the four fmt verbs.
6. **Land nothing without a citable case.** No JSON tags, no `Equal` beyond what AI-10.6 needs, no `Validate()` exported separately from construction, no builder type.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-model-request/` | Five new markdown files | None — new directory |
| `src/ai/system_instruction.go`, its test | New files | Medium — the segment shape is what AI-11.1 attaches markers to |
| `src/ai/request.go`, its test | New files | **High** — every later request milestone and every adapter depends on the shape |
| `src/ai/validation.go` | **One appended rule class**, `ErrMisplaced`, plus its `ruleClasses` row | Medium — the first class appended since AI-04 landed. Both registry mirrors updated in the same commit, as the internal guard demands |
| `src/ai/validation_registry_internal_test.go`, `validation_test.go` | The two mirrors | Low — mechanical, and the guard fails when they drift |
| `src/ai/message.go`, `content_part.go`, `tool*.go`, `role.go`, `doc.go` | **None** — read-only | — |
| `src/agenttest/` | **None in this half.** AI-10.5 adds the round-trip proof there | — |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **None.** The one gap found is reported below, not edited | — |
| `backend/agent/go.mod`, `go.work` | **None** — zero requires preserved | — |
| doc 0002 | **None.** No revert-and-record amendment was required; every leaf's first red test drove green in small steps | — |

## Vocabulary check — one reported gap, not acted on

`V-REQ-26` **generation option** defines the concept and its admission test but enumerates no members. Doc 0002 AI-10.1 item 3 says "the neutral vocabulary decided in AI-01, no more", and AI-01's closing checklist item 1 required the *term* to be defined, which it is. So the list is not in the register and this change did not put it there.

**Recommended register amendment, for the milestone that owns the register:** append to `V-REQ-26`'s row, or as a sub-list beneath § 3, the v1 generation-option set — *maximum output tokens, temperature, top-p, stop sequences* — with the note that the list grows only by re-applying the admission test, and that every rejected candidate belongs to `V-REQ-28`. `design.md` § 8 carries the table an amendment would cite.

## Rollback plan

The change is additive apart from six lines in `validation.go` and its two mirrors. Rollback is `git revert` of the commit range; nothing outside `src/ai` depends on it, and the appended rule class has no consumer other than this change.

The asymmetry worth stating: the three cross-region rules are cheap to **loosen** later (a request that was rejected becomes accepted) and expensive to **tighten** (a request a caller shipped stops constructing). That asymmetry is the argument for deciding them strictly now rather than deferring to "the provider will judge".

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| A second content validator appears | Medium | **High** | `request.go` calls `validateContent` and holds no per-kind logic; the composed position is asserted |
| The role/kind table is wrong for a provider added later | Medium | Medium | Loosening is additive; the table is one `var` and one test row per cell |
| `ErrMisplaced` is a class the package did not need | Low | Medium | `explore.md` § 3.4 walks all six existing classes and shows each fails the fit; AI-04's append rule is followed literally, both mirrors updated |
| The request leaks a payload through a fmt verb | Medium | **High** | Four verbs × six regions, one table test |
| The option list grows per provider | Medium | Medium | The admission table names the rejected candidates and their owner |
| The milestone busts the review budget | **Certain** | Medium | Forecast before the fact; leaf boundaries are commit boundaries; the milestone is split across two agents at the .3/.4 line |
| The second agent resumes in the wrong place | Medium | Medium | `tasks.md` phases AI-10.4 … AI-10.6 carry unchecked boxes and a written entry point; `proposal.md` marks them not-implemented |

## Dependencies

- **AI-04** taxonomy — extended by one appended class, per its own rule.
- **AI-05** roles and messages, **AI-06** parts, **AI-07** reasoning, **AI-08** tools, **AI-09** tool messages — all consumed, none modified.
- **No new Go dependency and no ADR required.** `slices`, `strconv`, `strings`, `unicode` are standard library.

## Success criteria

1. Every test-list item of AI-10.1, AI-10.2 and AI-10.3 taken red → green → refactored, in order, both outputs recorded.
2. `make test` green with `-race`, `make lint` clean, both import guards passing, `go.mod` at zero requires.
3. A minimal request — model identity, one user text message — constructs, validates and reads both back from `ai_test`.
4. An empty model identity and an empty message list each fail with `ErrEmpty` at the named region.
5. All four generation options carry through construction and read back, and an unset option is distinguishable from one set to its zero value.
6. Segment order and content round-trip exactly; `NewSystemText` equals the one-segment build; absence is legal and unrepresentable as an empty segment; empty and whitespace-only segments fail with `ErrEmpty`.
7. Message order and intra-message content order are preserved exactly.
8. `ToolChoice.ValidateAgainst` runs at the request boundary, with AI-08.3's three rules and positions unchanged.
9. The role/kind table is enforced for all twelve cells, reporting `ErrMisplaced`.
10. An orphan tool result fails with `ErrUnresolvedReference`; an orphan tool call succeeds; a duplicate call identity fails with `ErrDuplicate` positioned at the second occurrence.
11. A request holding a secret in every region renders it through none of `%v`, `%s`, `%+v`, `%#v`.
12. `validation.go`'s appended class is mirrored in both registries, and the internal guard passes.

## Notes for the following phases

- **`spec.md`** — requirement IDs `R-AMR-0NN`, scenario IDs `S-AMR-0NN`; the system under test is runtime behavior; every scenario verifiable by one test.
- **`design.md`** — owns every Go spelling, the documented rule order, the three dispositions, the option admission table, and the AI-11/AI-12 seams.
- **`tasks.md`** — six phases; red and green evidence per item for AI-10.1 … AI-10.3; unchecked plans for AI-10.4 … AI-10.6; the budget reassessment.
