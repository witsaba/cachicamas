# Explore — finish reasons and usage

> **Change**: `cachicamas-ai-completion-metadata`
> **Milestone**: AI-13 — Define finish reasons and usage
> **Nodes**: AI-13.1 `[leaf]` · AI-13.2 `[leaf]` · AI-13.3 `[leaf]` · AI-13.4 `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Reads**: doc 0002 § AI-13, doc 0001 § 3.2 · § 3.3 row 3 · § 7 **G10**, the register § 5 `V-MET`, `openspec/specs/ai-minimum-capabilities/spec.md` `CAP-R-03`, `backend/agent/src/ai/validation.go`

---

## 1. What already exists, and what it settles

Wave 0 and AI-04 have landed. Four things are therefore **not open questions** in this change:

| Landed | Where | What it removes from this change's scope |
| --- | --- | --- |
| The term register | `openspec/specs/ai-contract-vocabulary/spec.md` § 5 | `V-MET-01` … `V-MET-12` already define the finish-reason vocabulary, usage, the token-count field, absence-versus-zero and the cost formula. This change **implements** them; it does not get to redefine them |
| The v1 capability set | `openspec/specs/ai-minimum-capabilities/spec.md` `CAP-R-03` | Completion metadata is a required capability, and its second precision clause states that requiring the usage record does **not** require any count to be populated: "requiring a populated count is requiring a fabricated one" |
| The validation taxonomy | `backend/agent/src/ai/validation.go` | Every rule violation here reports through `Invalid` / the five sentinels / `At` / `FirstFailure`. No new sentinel, no second error type |
| The import boundary and the zero-require rule | `import_boundary_test.go`, `go.mod` | Standard library only. No enum-generation tool, no exhaustiveness linter, no assertion library |

The package exports the validation taxonomy and nothing else. `Usage` and `FinishReason` are the **first Layer 1 domain values** — the first types in this package that model something a model actually returns. That is worth naming, because two of the four decisions below are decisions about *how Layer 1 spells a domain value*, and every later milestone in Wave 1 and Wave 2 will copy whatever this one does.

## 2. What the register already fixes, verbatim

Cited so that no design tension below is a tension the register already resolved.

- `V-MET-01` — the finish reason is "the closed vocabulary value stating why generation stopped. **Closed and complete from birth**, because collapsing distinct stop conditions into a fallback is a loop-termination defect above Layer 1".
- `V-MET-06` / `V-MET-07` / `V-MET-08` — refusal, pause-turn and unknown are three rows, and `V-MET-08` says it out loud: "Those three are three states with three different correct responses."
- `V-MET-09` — usage is "input, output, cache-read, cache-write and reasoning token counts. Layer 1 **reports** usage; it never prices it".
- `V-MET-10` — "Each is independently present or absent, because providers report different subsets."
- `V-MET-11` — "'Not reported' and 'reported as nought' are different facts, and a consumer that cannot tell them apart writes a wrong cost formula and a wrong compaction estimate."
- `V-MET-12` — the cost formula is "the unambiguous arithmetic a consumer can write over usage fields **without guessing whether a count is inclusive of another**".
- `V-OUT-10` — loop termination is Layer 2's; "`V-MET-01` finish reason is the **input** to that decision, not the decision".

`V-MET-12`'s clause is the whole of AI-13.4: the register states the property and explicitly hands the choice down. `V-OUT-10` is the fence around AI-13.2 — see tension 4.

## 3. Five design tensions

### T1 — the zero value of a finish reason

A Go enum over an integer has a zero value whether or not the designer wants one. Two candidates:

- **Zero is unknown.** Simplest. `NormalizeFinishReason` returns the zero value for an unrecognised string and nothing has to be constructed.
- **Zero names nothing.** The unknown value is an explicit non-zero constant, and the zero value is outside the vocabulary.

The second is the one the repository's own history argues for. Retired defect **C1** was "an exported value satisfied the contract directly, so its zero value passed validation and bypassed every construction rule", and doc 0002 answers it at AI-06.1 with "zero-value-invalid". Making zero mean *unknown* recreates C1's shape in miniature: a completion event whose finish reason was never set would read as a recorded, deliberate "I do not recognise this provider string", which is a lie of exactly the kind `V-MET-08` forbids. It is also the finish-reason analogue of `V-MET-11`: an unset value is not the same fact as a recorded one.

### T2 — how a token count says "absent"

Three representations, all workable:

| Representation | Absent is | Cost |
| --- | --- | --- |
| `*int64` | `nil` | A pointer field is aliasable and mutable through a copy of the record; two usage records can share a count. Also invites `*u.Input` at every call site, which panics on absent |
| `int64` with `-1` as absent | a sentinel integer | The sentinel is a value in the domain of the type. Every consumer must know it, and a provider reporting a negative count (which is already a caller-contract failure) collides with it |
| A small value type carrying the count and a presence flag | the zero value | One more type in the package. Every read is two results |

The third is the only one where **the zero value is already correct**: an unset count is absent, which is exactly what `CAP-R-03` clause 2 requires of an adapter that reports nothing. It also matches a shape the package already uses — `Step.Index() (int, bool)` — so the "two results" cost is a convention this package has already paid for once.

### T3 — is the input count inclusive of the cache counts? *(the expensive one)*

The two vendors that matter disagree, and this is not a subtlety — it is a silent multiplier on every cached call:

- **Anthropic** reports `input_tokens` as the *uncached* input, with `cache_read_input_tokens` and `cache_creation_input_tokens` alongside it. The three are **disjoint**; the total input is their sum.
- **OpenAI** reports `prompt_tokens` as the total, with `prompt_tokens_details.cached_tokens` as a **subset breakdown** of it.

Summing "input + cache-read" against an OpenAI record double-counts every cached token. Taking "input" alone against an Anthropic record under-reports by up to 90 % of the prompt on a cache hit. Both mistakes are invisible until the invoice arrives, which is doc 0001 § 3.2's exact complaint about breakpoints, one field over.

The same question has a second half nobody asks: **is the reasoning count inside the output count?** Here both vendors agree — Anthropic bills thinking tokens as output tokens, OpenAI reports `reasoning_tokens` as a breakdown of `completion_tokens`. So the honest answer differs between the two sides of the record, and a design that applies one rule to both sides will be wrong about one of them.

### T4 — how much of "the obligation" is Layer 1's

AI-13.2 item 2 asks for "the documented obligation attached to each value … stated and testable where it is testable". The temptation is a method — `Resumable() bool`, or a `NextAction` enum. `V-OUT-10` forbids it: loop termination is Layer 2's decision and the finish reason is *the input to* that decision. A Layer 1 method that answers "should the loop resume?" moves the decision down a layer and makes every Layer 2 that disagrees a fork rather than a policy.

The clause "where it is testable" is the register's own escape. What is testable inside Layer 1 is that the three values are three values, that each has its own stable identity, and that a consumer *can* write three different responses. What is not testable inside Layer 1 is whether a consumer writes them correctly — that is AI-23.8's conformance case and AI-31.1's mapping.

### T5 — an exhaustiveness pin with no linter

`go vet` has no exhaustiveness check for constant enums and the module may not add one (`go.mod` carries zero requires). A pin that says "a new value must extend the normalization table and the string form" therefore has to be built from the language.

The lever is that the type's underlying integer is small and enumerable. A test can walk every value the type can hold, ask the package which of them are in the vocabulary, and compare that set against the constants it names by hand. Adding a constant then fails twice: the discovered set grows past the named set, and the new value has no string form and no normalization round-trip. Both failures are mechanical, and neither depends on anyone remembering to update a list.

## 4. Prior-art scan

Values observed across the three candidate vendors, which is what the neutral vocabulary has to cover. Per-vendor mapping is **AI-31.1's**, not this milestone's; the point of the scan is only to prove the vocabulary is not short a value.

| Neutral value | Strings seen in the wild |
| --- | --- |
| natural stop | `stop`, `end_turn`, `stop_sequence`, `STOP` |
| length | `length`, `max_tokens`, `MAX_TOKENS` |
| tool calls | `tool_calls`, `tool_use`, `function_call` |
| content filter | `content_filter`, `SAFETY`, `RECITATION`, `PROHIBITED_CONTENT` |
| refusal | `refusal` |
| pause turn | `pause_turn` |
| unknown | `OTHER`, and anything not listed above |

Two observations that shape the normalizer: casing differs by vendor (one vendor's values are upper snake case), and a vendor may add a value at any time without notice. The first makes trimming and lowercasing a requirement rather than a courtesy; the second is why item 4 of AI-13.1 exists at all — the normalizer must be **total**, because the alternative is a Layer 1 crash caused by a vendor release note nobody read.

## 5. Vocabulary gap

The register covers this milestone's nouns completely: `V-MET-01` … `V-MET-12` name the finish reason, each of its seven values, usage, the token-count field, absence-versus-zero and the cost formula. **No amendment is required**, which is the first time in Wave 1 that has been true.

One phrase is used inside a definition without being defined: `V-MET-08` speaks of "a provider stop condition the vocabulary does not recognise", and the register has no term for the raw vendor string itself. This is the same shape as `V-STR-23` **backpressure** and `V-PRV-17` **token counting**, both of which the register's own amendment history treats as defects worth appending. It is **not** blocking — this change cites `V-MET-08`'s phrase as written — and it is recorded here so that whoever next amends the register can decide.

## 6. What this change will not do

| Not here | Owner | Why |
| --- | --- | --- |
| The completion **event** that carries these two values | AI-15.2 | The envelope does not exist until AI-14 |
| Mapping any vendor's stop values | AI-31.1 | No adapter exists; the neutral table here is the target of that mapping, not a substitute for it |
| Verifying the inclusive/exclusive decision against a real transcript | AI-31.3 | doc 0002 already assigns it: "the semantics pinned at AI-13.4 are verified against a real cache-hit transcript" |
| Money, price tables, per-turn cost events | Layer 2 / Layer 3 (`V-OUT-07`, `V-OUT-08`) | doc 0002's out-of-scope clause for AI-13.4, verbatim |
| Any message, role or content-part type | AI-05, AI-06 | Being built concurrently; the file sets are disjoint by construction |
