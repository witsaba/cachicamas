# Proposal — tool declarations

> **Change**: `cachicamas-ai-tool-declarations`
> **Milestone**: AI-08 — Define tool declarations
> **Nodes**: AI-08.1 `[leaf]` · AI-08.2 `[leaf]` · AI-08.3 `[leaf]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Scope**: `openspec/changes/cachicamas-ai-tool-declarations/`, and **three new production files plus three new test files** under `backend/agent/src/ai/`. No `go.mod` change, no new dependency, **no register amendment**
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-04, AI-05
> **Blocks**: AI-10, AI-18, AI-26

---

## Intent

Land the provider-neutral transport representation of the tools a model may call — a declaration whose schema bytes survive **byte-identically**, a set whose iteration order is the caller's on every read, and a closed tool-choice vocabulary that is cross-validated against the declared set before any I/O happens.

Two of those three are properties that cannot be retrofitted, which is what makes the milestone worth its size. Byte fidelity is the one AI-26.4's stable cache prefix stands on, and adding it after fixtures exist breaks every fixture. Iteration determinism is the one whose absence produces **no failure at all** — just a cache prefix that differs per call and an input bill roughly ten times larger than it should be. Both are cheap now and expensive later.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can tell the inherited from the decided.

1. **The four terms are AI-01's.** `V-REQ-12` tool declaration, `V-REQ-13` schema bytes, `V-REQ-14` tool set, `V-REQ-15` tool choice are used with their register definitions and cited, never paraphrased. This change *implements* them.
2. **The failure vocabulary is AI-04's.** Every rule violation reports through `Invalid` with one of the five landed rule classes, at a position, composed by `FirstFailure`. No new sentinel, no local `errors.New`.
3. **The closed-vocabulary pattern is AI-05's.** Tool choice reuses it — `iota + 1`, an indexed table, an enumeration over the constant space, an exhaustiveness pin — and extends it by exactly the widening AI-05's `design.md` § 3.2 pre-authorised.
4. **Trap 1 binds.** Layer 1 carries the transport representation and does nothing with it: no execution, no name resolution, no schema validation.
5. **The module carries zero requires** until AI-24 and AI-37, each behind its own ADR gate. Two landed tests hold it there.
6. **Strict TDD.** Every test-list item taken red → green → refactored, **in order**; pins are exempt from red-first and are shown to bite.
7. **The evidence gate is recorded green `make test`** in `backend/agent/` — that is `go test -race -v ./...`.
8. **Tests live in the external test package `ai_test`.** `src/agenttest/` is not touched: its own file comment reserves the first cross-package readability proof for AI-06.2, which another agent is landing concurrently.

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `explore.md` | What exists, the four terms, the four tensions, the prior-art scan, the vocabulary check |
| `proposal.md` | This file |
| `specs/ai-tool-declarations/spec.md` | `R-ATD-001` … `R-ATD-012`, requirements over runtime behavior with WHEN/THEN scenarios |
| `design.md` | The Go shapes — every type, function and constant name; the name rule and its justification; the payload-carrying extension to the vocabulary pattern |
| `tasks.md` | AI-08.1/.2/.3 as phases, one task per test-list item, with red/green evidence, the pin's bite, and the budget reassessment |
| `backend/agent/src/ai/tool.go` | **AI-08.1's deliverable.** `Tool`, `NewTool`, the accessors, the name rules |
| `backend/agent/src/ai/tool_test.go` | AI-08.1's items, in `ai_test` |
| `backend/agent/src/ai/tool_set.go` | **AI-08.2's deliverable.** `ToolSet`, `NewToolSet`, ordered readback, duplicate rejection |
| `backend/agent/src/ai/tool_set_test.go` | AI-08.2's items, in `ai_test` |
| `backend/agent/src/ai/tool_choice.go` | **AI-08.3's deliverable.** `ToolChoiceMode` and its vocabulary, `ToolChoice`, the two constructors, the cross-validator |
| `backend/agent/src/ai/tool_choice_test.go` | AI-08.3's items and the pin, in `ai_test` |

**Not touched:** `validation.go`, `role.go`, `message.go`, `doc.go`, `import_boundary_test.go`, their tests, `src/agenttest/`, `go.mod`, `go.work`, and `openspec/specs/ai-contract-vocabulary/spec.md`.

### The four choices this proposal commits to

AI-08 has no `[decision]` node, so there is no `decision.md`, and every choice below is answerable to a scenario in `spec.md`. One line each, so a reviewer can accept or reject the substance before reading `design.md`.

1. **The tool name rule is the intersection of what real providers accept, not the union.** 1–64 bytes; first byte an ASCII letter or `_`; every byte an ASCII letter, digit, `_` or `-`. The reason is specific to names rather than to validation generally: **the name comes back**. A model's tool call names the tool, and Layer 2 resolves it — so a name an adapter had to rewrite outbound would need un-rewriting inbound, a mapping AI-26 would own and AI-30 would have to invert.
2. **Schema bytes are copied in, copied out, and never touched.** No re-marshalling, no canonicalization, no key reordering, no whitespace normalization. Byte-equality is asserted against JSON whose key order and spacing would not survive a marshal/unmarshal round trip.
3. **The tool set is a slice, and its zero value is the legal empty set.** Order is the caller's; duplicate names are rejected with the index of the *second* occurrence; nothing in the type iterates a map to decide anything.
4. **Tool choice is a payload-free closed vocabulary plus a value type that carries a member and its payload.** The table row widens with an arity column; the pin asserts that column against both constructors. This is the extension AI-05's `design.md` § 3.2 pre-authorised, not a departure from the pattern.

### Out of scope (explicit, and deferred-but-related is called out)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Validating a tool's schema against a **meta-schema** | Nobody in Layer 1 | `V-REQ-13`: "Layer 1 transports it and never validates it against a meta-schema". AI-08's charter repeats it |
| Validating schema bytes as **syntactically well-formed** | Deliberately unlanded | Doc 0002 asks for it on AI-09.1's argument bytes and pointedly does not on AI-08.1's schema bytes. `design.md` § 8 records both sides rather than deciding silently |
| Executing a tool; resolving a name; deciding whether a call may run; confining execution | Layers 2 and 3 | `V-OUT-04`, `V-OUT-05`, `V-OUT-16` — trap 1's second half |
| Where the tool set came from; whether it changes between turns | Layer 3 | `V-OUT-17` |
| **Attaching tools or tool choice to a request** | **AI-10** | The request does not exist yet. AI-10.3 re-runs AI-08.3's cross-validation at the request boundary |
| Whether a request may omit a tool choice entirely | **AI-10.3** | This milestone validates a tool-choice *value*; absence is a request-shape question |
| The tool **call** and its argument bytes | **AI-09** | A content variant, with its own identity, ordinal and byte-fidelity property |
| Cache-boundary markers on a declaration | **AI-11.1** | Markers are advisory and land on three regions at once |
| Any content part, message type or request type | AI-06 / AI-05 / AI-10 | This milestone adds none, and touches none of their files |

## Approach

1. **Implement the register, do not redesign it.** Every property proven below is a clause already in `V-REQ-12` … `V-REQ-15`. The tests are those clauses made mechanical.
2. **Make byte fidelity the first thing proven, against input designed to break it.** The fixture is JSON with non-alphabetical keys and irregular whitespace — a shape a marshal/unmarshal round trip would silently rewrite. An implementation that "helpfully" normalizes fails on the second test of the milestone, not in AI-26.
3. **Make determinism structural.** The set holds a slice; nothing decides anything from a map. The proof is a set large enough that Go's map randomization would show, iterated many times, compared against the caller's order — not just against itself.
4. **Extend the closed-vocabulary pattern by the one move it already allows.** Widen the table row; keep `iota + 1` and the constant-space enumeration untouched. Both of those are the load-bearing rules per AI-05's `design.md` § 3.2, and neither has a degree of freedom.
5. **Reuse the name rule where a name appears.** A tool choice naming a syntactically impossible tool fails at construction, not at cross-validation — one rule set, two call sites.
6. **Land nothing without a citable case.** No `Tool.String`, no `ToolChoice.String`, no `ToolSet.Contains` beyond what cross-validation needs, no equality, no encoding. Each is plausible; none has a case in doc 0002 or the register.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-tool-declarations/` | Five new markdown files | None — new directory |
| `src/ai/tool.go`, `tool_test.go` | New files | **Medium** — byte fidelity is inherited by AI-26.4 and cannot be added later |
| `src/ai/tool_set.go`, `tool_set_test.go` | New files | **Medium** — an order defect here is invisible except on a bill |
| `src/ai/tool_choice.go`, `tool_choice_test.go` | New files | Low — a closed vocabulary is a well-understood shape, and the pin guards its growth |
| `src/ai/validation.go`, `role.go`, `message.go`, `doc.go` | **None** — read-only. AI-06 and AI-13 are being built concurrently against neighbouring files | — |
| `src/agenttest/` | **None** — reserved for AI-06.2 | — |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **None.** `explore.md` § 6 records the check and the three near-misses | — |
| `backend/agent/go.mod`, `go.work` | **None** — zero requires preserved | — |
| doc 0002 | **None expected.** Any amendment required by the revert-and-record clause lands in this same PR and is reported prominently | — |

## Rollback plan

The change is additive: six new Go files, one new change directory, nothing modified. Rollback is `git revert` of the commit range; no build depends on the markdown, and `src/ai` returns to exporting AI-04's and AI-05's surface alone.

Post-merge reversal has an asymmetry worth stating. The name rule is cheap to **loosen** later (a name that was rejected becomes accepted; nothing that worked stops working) and expensive to **tighten** (a name a caller shipped stops constructing). That asymmetry is the argument for landing the intersective rule now rather than the permissive one: starting narrow leaves the cheap move available, starting wide does not.

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| An implementation normalizes schema bytes "helpfully" | Medium | **High** — AI-26.4's cache prefix depends on it and every fixture would break | `R-ATD-002`'s fixture is chosen so normalization is *detectable*: non-alphabetical keys, irregular whitespace. The test also asserts the returned slice does not alias the stored one |
| The tool set's order is right by accident and regresses later | Medium | **High** — no error, no crash, a tenfold input bill | The order test iterates a 64-element set 100 times and compares against the **caller's** order, not against a previous read. A map-backed implementation fails within the first few iterations |
| The name rule is wrong — too strict for a provider we later add | Medium | Low | Loosening is additive; § "Rollback plan" argues the asymmetry. `design.md` § 3 names which half is verified from a primary source |
| The vocabulary extension quietly breaks AI-05's pattern for AI-13 | Low | Medium | `design.md` § 5 states the extension as a rule AI-13 can read, and the pin asserts the arity column rather than a hard-coded member list |
| The pin passes vacuously | Low | Medium | It is **shown to bite** — a scratch member declared without a table entry, its failure recorded, then removed. `tasks.md` carries the output |
| The milestone grows past its review budget | **High** | Medium | Expected, and forecast in `explore.md` § 7 rather than discovered. `tasks.md` carries the reassessment; the commit split already falls on the node boundary |
| A red step is skipped because Go will not compile a test against a missing symbol | Medium | Medium | AI-04's convention, restated in `design.md` § 9: land the narrowest declaration that compiles and **fails**, record the failure, then implement |

## Dependencies

- **AI-04** — the taxonomy. Every failure here reports through it; no new sentinel.
- **AI-05** — the closed-vocabulary pattern, and the copy-in/copy-out habit.
- **AI-01** — the register. Every noun used here is one of its rows; **no amendment is needed**, and `explore.md` § 6 records why.
- **No new Go dependency, and no ADR required.** `bytes`, `slices`, `strconv` and `strings` are standard library.

## Success criteria

1. Every test-list item of AI-08.1, AI-08.2 and AI-08.3 is taken red → green → refactored, in order, with both outputs recorded. The pin is exempt from red-first and is **shown to bite**.
2. `make test` in `backend/agent/` is green with `-race`, and its output is recorded. `make lint` is clean.
3. Both import-boundary guards still pass and `go.mod` still carries zero requires.
4. A declaration's name, description and schema bytes read back exactly from an external package.
5. Schema bytes are **byte-identical** on readback for input whose key order and whitespace a marshal round trip would rewrite, and the returned slice does not alias the stored one.
6. Construction fails through AI-04 sentinels for: an empty name, an over-long name, a name outside the alphabet, and empty schema bytes — in the documented order, identically across runs.
7. An **empty tool set is legal** and constructible, and its zero value is that same empty set.
8. Duplicate tool names in one set are rejected with an AI-04 sentinel positioned at the second occurrence.
9. A 64-tool set iterated 100 times yields the caller's order every time.
10. Each of the four tool-choice members is constructible and round-trips through parse-and-render; the payload-carrying member carries its name back.
11. A tool choice naming an undeclared tool fails with `ErrUnresolvedReference`; any choice other than "none" against an empty set fails with `ErrEmpty`; "none" against an empty set succeeds.
12. The exhaustiveness pin fails when a member is declared without a table entry — recorded, then removed.

## Notes for the following phases

- **`spec.md`** — requirement IDs `R-ATD-0NN`, scenarios `S-ATD-0NN`. The system under test is runtime behavior, as in AI-04 and AI-05; every scenario is independently verifiable by a test.
- **`design.md`** — owns every Go spelling, the exact name rule with its primary-source citation, the payload-carrying extension stated as a rule AI-13 can reuse, and the documented rule order for each constructor and for cross-validation.
- **`tasks.md`** — three phases, one per node; one task per test-list item; red and green evidence per item; the pin's bite proof; the budget reassessment; the verification pass last.
