# Explore — the normalized request core

> **Change**: `cachicamas-ai-model-request`
> **Milestone**: AI-10 — Define the normalized request core
> **Nodes**: AI-10.1 … AI-10.6, all `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Depends on**: AI-04, AI-05, AI-06, AI-07, AI-08, AI-09
> **Blocks**: AI-11, AI-12, AI-20, AI-26

---

## 1. What this milestone is

AI-10 is the first milestone that *assembles*. Every milestone of wave 1 before it defined a piece in isolation — a failure, a role, a part, a declaration. AI-10 is the thing a provider receives: `V-REQ-20` **normalized request**, "the complete provider-neutral description of one model call: model identity, ordered system-instruction segments, ordered messages, the tool set and tool choice, and generation options."

Nothing here is a new concept. Six register rows already define the whole surface — `V-REQ-19` through `V-REQ-22`, `V-REQ-26`, plus the regions themselves. This milestone's work is composition: putting five already-landed contracts into one value, deciding the rules that only become expressible *once they are in the same value*, and proving the whole thing is readable from another package.

## 2. What already exists, and what each of it gives this milestone

| Landed | File | What AI-10 consumes |
| --- | --- | --- |
| AI-04 taxonomy | `validation.go` | `Invalid`, `At`, `AtIndex`, `FirstFailure`, six rule classes, the `Path`/`Violation` pair, and the redaction posture |
| AI-05 roles and messages | `role.go`, `message.go` | `Role` (three members, **no system role**), `Message`, `NewMessage`, `MessageID`, copy-in/copy-out |
| AI-06 content parts | `content_part.go`, `text_content.go` | `Part`, `PartKind`, `PartKinds()`, and — the load-bearing one — **`validateContent(prefix Path, content []Part)`** |
| AI-07 reasoning | `reasoning_content.go` | `Reasoning`, its state vocabulary, its byte-exact round-trip token, and the redaction posture on an exported payload type |
| AI-08 tools | `tool.go`, `tool_set.go`, `tool_choice.go` | `Tool`, `ToolSet`, `ToolChoice`, and **`ToolChoice.ValidateAgainst(ToolSet)`** |
| AI-09 tool messages | `tool_call.go`, `tool_result.go` | `ToolCall` (id, name, argument bytes), `ToolResult` (correlation, content, failure), `ToolCalls([]Part)` |

Two of those are seams left open *for this milestone by name*, and finding them already cut is the reason AI-10 is composition rather than invention.

### 2.1 The `validateContent` prefix — AI-06.3 item 3, cashed here

`content_part.go` documents the parameter this milestone exists to use:

> The prefix is prepended to every position this function reports. `NewMessage` passes none, so a failure renders as `content[0]`. AI-10 holds a request, whose content sits one level deeper, and will pass `AtIndex("messages", i)` so the same failure renders as `messages[2].content[0]`.

AI-06's `decision.md` § 7.2 states the principle — "one rule set, two callers" — and states the failure mode of the alternative: "a constructor that checks and a boundary that does not". **This change writes no second content validator.** The request path calls `validateContent` with `Path{AtIndex("messages", i)}` and inherits every part rule, at every depth, for every kind that exists now or later.

### 2.2 `ToolChoice.ValidateAgainst` — AI-08.3, cashed here

`tool_choice.go` closes with:

> This is not the only place these rules run. AI-10.3 re-runs them at the request boundary, which is why they live on a method a request can call rather than inlined into a constructor.

The three rules it carries — mode is a member, a non-`none` choice against an empty set is `ErrEmpty`, a specific choice naming an undeclared tool is `ErrUnresolvedReference` — are exactly the request-boundary rules, and their positions (`toolChoice`, `tools`, `toolChoice.name`) are already request-scoped names. **This change reimplements none of them.**

### 2.3 Three questions handed forward to AI-10.3 by name

| Question | Handed over by | Where it is answered |
| --- | --- | --- |
| May a tool-result part appear in a non-tool role? | `role.go` `RoleTool` GoDoc; doc 0002 AI-10.3 item 3 | `design.md` § 5 |
| Is a tool result whose call is absent legal? | doc 0002 AI-09 out-of-scope; AI-10.3 item 4 | `design.md` § 6 |
| Must tool-call identities be unique? | AI-09 `design.md` § 3.2, which names `ErrDuplicate` as the class if the answer is "reject" | `design.md` § 7 |

All three are *cross-region* questions: none is decidable while holding one message, which is why none could be answered before a request existed.

## 3. The four tensions

### 3.1 Optional regions on an immutable value

A request has one required region pair (model identity, messages) and four optional ones (system instruction, tool set, tool choice, generation options), and every generation option is *individually* optional because absence and zero are different facts — `V-MET-11`'s distinction, applied on the request side. A constructor taking six positional parameters is unreadable and cannot grow; a mutable exported struct is not sealed.

Three candidates were weighed:

| Candidate | Verdict |
| --- | --- |
| Positional constructor, one parameter per region | Rejected. Six parameters today, eight after AI-11 and AI-12; every caller of the common case writes four zero values |
| An exported draft struct with exported fields, validated by `NewRequest(draft)` | Plausible. Extends by adding a field, and AI-12's rebuild is "read a draft, mutate, rebuild". Rejected because it puts an unsealed, payload-carrying, mutable type on the exported surface for every caller of the *simple* case to fill in, and because a field whose absence and zero coincide reintroduces the flat-string trap for `Temperature` |
| **Functional options** — `NewRequest(model, messages, ...RequestOption)` | **Selected.** Required regions are parameters; every optional region is an option that is either applied or absent, so absence is structural rather than a sentinel value. The request stays sealed; nothing exported carries a settable field |

The selection also cuts AI-12's seam without AI-10 building any of it: a per-request override is the same `RequestOption` applied again, and copy-on-write rebuild is one method — `Request.With(...RequestOption) (Request, error)` — that AI-12 adds without reshaping a single existing signature.

### 3.2 The system instruction, segmented from birth

Doc 0001 § 3.2 names this the first thing that must change before an adapter exists: "The system instruction is a flat string, so there is nowhere to mark a cache boundary." Doc 0002 prices it: "Ordered segments cost nothing here: a single-segment request is the common case, and the ergonomic constructor for it is part of this milestone's deliverable."

The trap a flat string sets is not only the missing cache boundary. It is that `""` means both *absent* and *empty*, so a request that meant to carry no instruction and one that carries an empty one are the same value. Segments dissolve both at once: absence is zero segments, and an empty segment is rejected at construction, so "one empty segment" is not merely equal to absence — it is **unrepresentable**.

### 3.3 Which generation options are neutral

`V-REQ-26` defines a generation option and states the admission test — "an option that exists to satisfy one provider belongs in the escape hatch instead" — but the register enumerates no members, and AI-01's closing checklist item 1 required the *term*, not the list. So AI-10 owns the list, under a test the register already wrote.

`design.md` § 8 applies the test option by option. Four pass; every candidate that fails is named with the provider that fails it, and its owner is AI-12's escape hatch. This is the one place this milestone decides something the register did not, and `proposal.md` § "Vocabulary check" records it as a register-amendment recommendation rather than acting alone.

### 3.4 A rule class the taxonomy does not have

AI-10.3's role-versus-content-kind rule rejects a value that is *individually valid* because of *where it appears*: a `Reasoning` part in a user message is a perfectly constructed part, and what fails is its placement. None of AI-04's six classes describes that:

| Class | Why it does not fit |
| --- | --- |
| `ErrEmpty` | Something is present |
| `ErrNotInVocabulary` | The kind **is** a vocabulary member; the value is not the problem |
| `ErrOutOfRange` | No bound is crossed |
| `ErrMalformed` | The value is well-formed for its encoding |
| `ErrUnresolvedReference` | It names nothing |
| `ErrDuplicate` | Nothing repeats |

AI-04's own rule governs: the set "is extended by appending a class in the pull request that needs it". `design.md` § 5.3 appends `ErrMisplaced` and states the criterion AI-04 used for `ErrDuplicate` — the fix the consumer is being told to make is different ("move it, or drop it", not "spell it differently"), and `errors.Is` is the only place that difference is readable.

## 4. Prior art scan

- **Vercel AI SDK / LangChain-style unified request objects.** Both carry a flat `system` string. Both then grew a parallel structure for cache control, which is doc 0001 § 3.2's prediction observed in the wild.
- **OpenAI Chat Completions.** The system instruction is a *message* with role `system`. The register deliberately does not: AI-05 landed three roles and no system role, so the instruction is its own region and an adapter renders it into a message on the way out. That decision is inherited here, not re-taken.
- **Anthropic Messages.** The system instruction is a top-level field taking either a string or an ordered array of blocks, each independently markable for caching. This is the shape AI-10.2 + AI-11.1 target, and it is the reason the ergonomic single-segment path must produce a value indistinguishable from the one-segment array.
- **Go idiom.** Functional options are the standard answer to "an immutable value with many optional regions"; `crypto/tls`, `net/http` and most SDKs use either options or a config struct. The sealed-value house style of this package makes options the closer fit, because a config struct would need to be exported and settable.

## 5. Risk register for this milestone

| Risk | Impact | Mitigation |
| --- | --- | --- |
| A second content validator is written by reflex | **High** — recreates C1 from the other direction, the exact failure AI-06 § 7.2 named | `request.go` calls `validateContent` and holds no per-kind logic. `request_test.go` asserts the composed position `messages[1].content[0].text` |
| The cross-region rules are decided by the first adapter instead of here | **High** — a rule discovered at AI-26 is a breaking change to a frozen surface | Three dispositions decided in `design.md` §§ 5–7, each pinned by a test that names the reason |
| The option list grows once per provider | Medium — the unbounded-vocabulary failure doc 0001 § 3.3 closes with | `design.md` § 8 applies `V-REQ-26`'s admission test in a table, and names AI-12's escape hatch as the owner of every rejected candidate |
| The exported request leaks the prompt through `%v` | **High** — the request carries every payload in the package at once | `String`/`GoString` render structural counts only, and the leak test enumerates all four verbs against a request holding a secret in every region |
| The milestone busts the review budget | **High**, and expected | Forecast in `tasks.md` before the fact; the leaf boundary is the commit boundary; the milestone is split across two agents at the .3/.4 line |

## 6. Vocabulary check

Every noun this change uses resolves to a register row: `V-REQ-19` segment, `V-REQ-20` normalized request, `V-REQ-21` model identity, `V-REQ-22` request validation, `V-REQ-26` generation option, plus `V-REQ-01` … `V-REQ-18` for the regions and `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-13`, `V-FAIL-17` for failure.

**One gap, reported and not acted on.** `V-REQ-26` names the concept and its admission test but enumerates no members, so "the neutral vocabulary decided in AI-01" (doc 0002 AI-10.1 item 3) is a test rather than a list. This change applies the test and lands four options; it does **not** edit `openspec/specs/ai-contract-vocabulary/spec.md`. The recommended amendment is recorded in `proposal.md`.

**One taxonomy amendment, made here and reported.** `ErrMisplaced` is appended to `validation.go` under AI-04's own append rule. `V-FAIL-17` states that "which classes exist, and how the set grows, are AI-04's", so the register needs no amendment for it.

## 7. Size forecast

| Slice | Forecast |
| --- | --- |
| SDD artifacts | ~1,000 prose lines |
| AI-10.1 + AI-10.2 + AI-10.3 (this agent) | ~450 Go production, ~800 Go test |
| AI-10.4 + AI-10.5 + AI-10.6 (second agent) | ~150 Go production, ~600 Go test |

Doc 0002's budget is "prefer < 250 changed lines, stop and reassess before 400". **Split trigger 4 fires before the first test is written**, and it is recorded here rather than discovered: AI-10 is six leaves and the largest composition in Layer 1. The reassessment lives in `tasks.md`; the mitigation already applied is that the milestone is worked as two chained halves on the leaf boundary, which is where doc 0002 puts the PR-chain boundary.
