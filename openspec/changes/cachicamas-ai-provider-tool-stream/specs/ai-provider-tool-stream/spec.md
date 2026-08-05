# Spec — translate the tool-call stream

> **Change**: `cachicamas-ai-provider-tool-stream`
> **Milestone**: AI-30 · **Nodes**: AI-30.1 … AI-30.5 — all `[leaf]`
> **Phase**: spec (delta — ONE new capability `ai-provider-tool-stream`, ADDED only; **zero modified capabilities**)
> **Canonical spec**: `openspec/specs/ai-provider-tool-stream/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Date**: 2026-08-04 · **Project**: cachicamas (witsaba)
> **Requirement IDs**: `R-ATL-0NN` · **Scenario IDs**: `S-ATL-0NN`
> **Wire-fact source of record**: [`citations.md`](../../citations.md) — claims **C9.1 … C9.6** against `github.com/openai/openai-openapi` at pinned commit `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439`, the same pin `doc.go` and the AI-28 `C1 … C8` table already carry.
> **Binding predecessors, cited by identifier and never amended**: `ai-tool-call-events` (AI-18: `R-ATC-002` … `R-ATC-012`), `ai-provider-text-stream` (AI-28, unarchived), `ai-provider-errors` (AI-19), `ai-stream-decoder` (AI-27), `ai-provider-conformance-suite` (AI-23.3, `CapToolCalls`), `ai-request-translation` (AI-26), `ai-error-mapping` (AI-32.5's `capture.go`) · [doc 0002](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) §§ AI-30.1 … AI-30.5 · [`proposal.md`](../../proposal.md)
> **Depends on**: AI-18, AI-28, AI-19, AI-23.3 · **Parallel with**: AI-29 (reasoning), AI-31 (usage/finish reason)

---

## ADDED Requirements

## Purpose

AI-26.5 shipped the **request** half of tool calling: `Translate` renders `tool_calls` on the assistant message and `role:"tool"` results, byte-proven. The **response** half is discarded — `wireDelta` decodes exactly one field, `content`, so a model's answer to a tool declaration falls through `R-ATS-017`'s skip rule and vanishes with no event, no error and no trace. This spec constrains the component that closes that arc: the OpenAI-compatible adapter's **tool-call response mapping**.

Four distinctions shape every requirement below and are stated once here.

**Almost nothing about streamed tool calls is specified on the wire (C9.2, C9.3, C9.4, C9.6 — four load-bearing negatives).** `ChatCompletionMessageToolCallChunk` declares four properties and requires exactly one, `index` — which carries **no description at all** at the pinned commit (C9.2). No prose anywhere states that `id`/`function.name` appear only on a call's first element, or whether either may itself fragment (C9.3). `function.arguments` is a plain string whose description is copied verbatim from the non-streaming schema and says nothing about partial delivery or append semantics (C9.4). And **no per-call end signal exists**: completeness is bounded only by the choice-level `finish_reason` and stream end (C9.6). The accumulation behavior every OpenAI-compatible provider actually exhibits is therefore **dialect convention pinned by fixtures**, not translated schema. This spec labels that wholesale and honestly, in the same form `R-ATS-009` revision 2 uses for its split-rune fixtures — and labelling a rule dialect-conventional **never weakens it**: every such scenario stays `[test]` and stays load-bearing.

**The wire is permissive; the neutral events are strict.** Every chunk-element field except `index` is optional (C9.1), so any distribution across elements is schema-valid. `ai.NewToolCallStart` by contrast requires a non-empty `id` **and** a non-empty `name` before any argument byte (`R-ATC-002`). The gap between "schema-valid" and "representable" is real, and this spec resolves it in exactly one direction: **typed failure, never a silent drop and never a minted value** (`R-ATL-010`).

**This package ships no accumulator, and this milestone assembles exactly once.** `ai`'s three tool-call events are fragment-only by contract: `ToolCallDelta` carries a new fragment and never a transcript (`R-ATC-005`), `ToolCallEnd` carries the complete bytes (`R-ATC-006`), and neither parses, canonicalizes nor validates anything — `tool_call_event.go` names **AI-30's reassembly** as the single point where a streamed call's arguments are validated and where empty arguments canonicalize (`R-ATC-007`). This milestone is that point. The **ordinal** stays derived stream-side, never shipped as an accumulator (`R-ATC-012`).

**The tenth failure category is recorded, not absorbed.** `errors.go` states AI-19's nine-member vocabulary may gain a resource-exhaustion member **only once a second consumer exists**, and names **AI-30.1 item 4 — the per-call accumulation cap — as that second consumer**. This change records that the second consumer now exists and stops there (`R-ATL-006`). The cap keeps `ErrFrameTooLarge`'s exact compromise form. Widening `ai.FailureCategory` belongs to the taxonomy owner as its own change.

Requirement count: **16** (`R-ATL-001` … `R-ATL-016`). Scenario count: **73** (`S-ATL-001` … `S-ATL-073`) — **54 [test]**, **19 [inspection]**.

*Counted mechanically over the scenario bullets below, not estimated. The nineteen `[inspection]` scenarios are `S-ATL-005`, `012`, `018`, `027`, `031`, `032`, `043`, `051`, `058`, `061`, `064`, `065`, `066`, `067`, `068`, `069`, `071`, `072`, `073`. Evidence rules key off these markings; a reconciliation finding a different split is reading a stale revision, not a downgraded test.*

## Definitions used by this spec

- **The producer** — `(*Client).Stream`, AI-28's shipped shell. This milestone adds mapping to it; it adds no second entry point.
- **A chunk element** — one item of choice 0's `delta.tool_calls` array, of schema `ChatCompletionMessageToolCallChunk` (C9.1).
- **A call** — the set of chunk elements sharing one `index` value across a stream, plus the neutral events mapped from them.
- **A call's block** — the stream-wide `ai.BlockIndex` this adapter mints for that call. It is **not** the wire's `index` value: `ai.BlockIndex` 0 is invalid at construction (`R-ATC-004`) while wire `index` 0 is legal and routine.
- **The text block** — `textBlockIndex = 1`, `R-ATS-008` rule 1's constant. Reasoning blocks are **absent by AI-29's own decision**, so at this milestone the block-index space is text + tool calls and nothing else.
- **A call's close** — the moment `R-ATL-005`'s completeness rule fires. There is no wire signal for it (C9.6).
- **A call's ordinal** — its 1-based position among the stream's tool-call **start** events, in emission order (V-STR-21, `R-ATC-012`). A distinct axis from the block index and from the wire `index`.
- **The terminal window** — between choice 0's terminal chunk (non-null `finish_reason`, C2) and the terminal sentinel `data: [DONE]` (C5).
- **Scenario kind** — every scenario is marked **[test]** (runnable and failable under `make test` / `go test -race -count=1 ./...`) or **[inspection]** (a reviewer-checkable obligation over the artifact or shipped source, deterministic but not executed by the suite). Under Strict TDD, every **[test]** scenario MUST be demonstrated failing before the code satisfying it exists.

## The citation gate

Every statement of wire shape in this document, in the shipped source and in its tests MUST resolve to a claim in this change's `citations.md` (**C9.1 … C9.6**) or to AI-28's own **C1 … C8**, **or** be explicitly labelled **dialect-conventional** with a **named fixture-pin obligation**. `R-ATL-016` makes a violation detectable rather than merely discouraged. Citing a claim number that does not exist is a spec defect, not a typo.

| Claim | What it establishes |
| --- | --- |
| **C9.1** | `ChatCompletionMessageToolCallChunk`'s complete property set — `index` (integer), `id`, `type` (const-enum `function`), `function{name, arguments}` — and a `required` list containing **only `index`** |
| **C9.2** | `index` is the sole required field and carries **no description whatsoever**; its fragment-correlation role is **not** spec text |
| **C9.3** | **NEGATIVE** — no prose states that `id`/`name` appear only on a first element, repeat, or may themselves fragment; all fields optional, so any distribution is schema-valid |
| **C9.4** | `function.arguments` is a plain string with **zero** stated fragment or append semantics; `type` is a single-member enum; no streaming custom-tool-call element exists |
| **C9.5** | The deprecated streaming `function_call` delta: anonymous inline object, optional `arguments`/`name`, **no `index`**, no `required` list, zero fragmentation prose |
| **C9.6** | **NEGATIVE** — no per-call end marker, count field or done event exists in chat scope; completeness is choice-level `finish_reason` plus stream end only |

### The five dialect-conventional labels this spec carries

Every one is load-bearing, every one is `[test]`-backed, and every one names its pinning fixture. No sixth label may be introduced without naming its fixture in the same place.

| # | Rule | Cited negative | Pinning fixture |
| --- | --- | --- | --- |
| 1 | `index` correlates a call's elements across chunks | C9.2 | the interleaved two-call transcript (`S-ATL-006`, `S-ATL-012`) |
| 2 | `id`/`name` may arrive on any element, typically the first; absence on later elements is normal | C9.3 | the id/name-distribution transcript trio (`S-ATL-013` … `S-ATL-015`, `S-ATL-018`) |
| 3 | `function.arguments` values concatenate, in arrival order, into one payload | C9.4 | the fragment-concatenation transcript (`S-ATL-022`, `S-ATL-038`) |
| 4 | A call closes at the choice's `finish_reason`, or at stream end | C9.6 | the terminal-chunk close transcript and its truncated twin (`S-ATL-024`, `S-ATL-027`, `S-ATL-044`) |
| 5 | The deprecated `function_call` delta is tolerated, ignored, never mapped | C9.5 + C7 | the `function_call`-only transcript (`S-ATL-062`, `S-ATL-064`) |

## Requirement ownership by node

| Node | Requirements | Status |
| --- | --- | --- |
| AI-30.1 — per-call accumulation | `R-ATL-001` … `R-ATL-006` | ready |
| AI-30.2 — empty and zero-fragment calls | `R-ATL-007` | ready |
| AI-30.3 — argument-byte fidelity | `R-ATL-008` | ready |
| AI-30.4 — truncation and malformation | `R-ATL-009`, `R-ATL-010` | ready |
| AI-30.5 — ordinal preservation | `R-ATL-011` | ready |
| Acceptance surface and charter | `R-ATL-012` … `R-ATL-016` | ready |

---

## AI-30.1 — Per-call accumulation `[leaf]`

### R-ATL-001 — The chunk element is decoded to C9.1's declared shape, byte-preservingly, and to nothing else

The producer MUST decode choice 0's `delta.tool_calls` array as items of `ChatCompletionMessageToolCallChunk` (C9.1) and MUST read only that schema's declared fields. Because `required` contains **only `index`** (C9.1), every other field MUST be treated as optional on **every** element: `id`, `type`, `function`, `function.name` and `function.arguments` absent from an element MUST NOT be a failure by itself. `function.arguments` MUST be decoded through the same byte-preserving `json.RawMessage` + `unquoteJSONString` path `delta.content` already uses (chunk.go D2), **reused and never re-invented**, because `encoding/json`'s own string decode substitutes `U+FFFD` for byte sequences that are not well-formed UTF-8. The producer MUST NOT infer a delta-side field name from the request-side `ChatCompletionMessageToolCall` schema by analogy. Fields beyond C9.1's four MUST be ignored, per `R-ATS-017`'s mandatory unknown-field tolerance.

#### Scenarios

- **S-ATL-001** *[test]* — Given a chunk whose choice-0 delta carries a `tool_calls` element with **only** `"index":0` and no other key, when the stream is drained, then no failure is reported, no argument fragment is attributed to that call, and a later element for the same index still maps normally.
- **S-ATL-002** *[test]* — Given two transcripts identical except that one omits `"type"` and the other carries `"type":"function"`, when both are drained, then the emitted event sequences are equal in kind, order and payload — `type` is optional (C9.1) and its single legal value (C9.4) carries no mapping decision.
- **S-ATL-003** *[test]* — Given an element carrying `index`, `id`, `function.name`, `function.arguments` **and** two invented sibling fields, when the stream is drained, then the call maps normally and no failure is reported.
- **S-ATL-004** *[test]* — Given an element whose `function.arguments` wire value is `"{\"q\":\"a\\nb\"}"`, when the emitted delta's `Fragment()` is read, then it is the byte sequence `{"q":"a\nb"}` with the escape decoded exactly once — asserted byte-equal, not string-compared after a re-encode.
- **S-ATL-005** *[inspection]* — Given the shipped source, when a reviewer reads every comment naming a delta-side field, then each cites **C9.1** or **C9.4**, none cites the request-side `ChatCompletionMessageToolCall` schema, and the argument-decode path calls the existing `unquoteJSONString` rather than declaring a second unquoter.

### R-ATL-002 — `index` correlates a call's elements, and mints a stream-wide-unique block that never collides with the text block

**Dialect-conventional premise carrying a fixture-pin obligation (label 1).** C9.2 establishes that `index` is the element's sole required field **and that it carries no description at all** — the spec never states what it indexes. That `index` correlates fragments of the same call across chunks is therefore **dialect convention, not spec text**, and the obligation is discharged at fixture level: a committed transcript MUST exercise two concurrent calls whose elements interleave by `index`, and the tests MUST assert reconstruction from that correlation alone.

On that premise, the producer MUST treat two elements carrying the **same** `index` value as belonging to one call and two elements carrying **different** `index` values as belonging to different calls. For each call it MUST mint one `ai.BlockIndex` that is: (a) **stable** — every start, delta and end event of that call reports the identical value; (b) **stream-wide unique** — no two calls share one, and no call ever shares the text block's index; and (c) **valid** — at least 1, since `ai.BlockIndex` 0 is rejected at construction (`R-ATC-004`) while wire `index` 0 is legal. The minted block index MUST NOT be assumed equal to the wire `index`. An element carrying **no** `index` at all violates C9.1's `required` list and MUST yield a typed malformed-response failure — never a silent default into bucket 0.

#### Scenarios

- **S-ATL-006** *[test]* — Given a transcript whose chunks carry, in order, elements with `index` 0, 1, 0, 1, when the stream is drained, then exactly two calls exist, each carrying the fragments of its own `index` in arrival order.
- **S-ATL-007** *[test]* — Given that same drained stream, when both calls' block indices are read, then they are different from each other.
- **S-ATL-008** *[test]* — Given a transcript whose only call carries wire `index` `0`, when the stream is drained, then that call's events are constructed successfully and its block index is at least 1 — the stream reports no construction failure.
- **S-ATL-009** *[test]* — Given a transcript carrying choice-0 `content` text **and** two tool calls, when the stream is drained, then no tool-call event reports block index `1` (the text block's constant), and the text events are identical in kind, order and payload to a text-only twin transcript.
- **S-ATL-010** *[test]* — Given one call delivered as start, three fragments and a close, when every emitted event for that call is inspected, then all five report the same block index.
- **S-ATL-011** *[test]* — Given a chunk whose `tool_calls` element carries `id` and `function.arguments` but **no** `index` key, when the stream is drained, then the terminal event is a typed malformed-response failure and no call was opened at any index.
- **S-ATL-012** *[inspection]* — Given the shipped source and fixture set, when a reviewer reads them, then the correlation rule is labelled **dialect-conventional** citing **C9.2**'s negative, the interleaved two-call transcript naming it is committed, and no comment claims the vendor documents `index`'s meaning.

### R-ATL-003 — Identity and tool name are readable from the start event before any argument byte, under any legal field distribution

**Dialect-conventional premise carrying a fixture-pin obligation (label 2).** C9.3 is a load-bearing negative: no prose anywhere states that `id`/`name` appear only on a first element, that they repeat, or that either may itself arrive fragmented — and because every field but `index` is optional (C9.1), **any** distribution across elements is schema-valid. The first-element-carries-identity behavior real providers exhibit is undocumented, so the adapter MUST tolerate presence and absence on any element and MUST pin the shapes it supports with committed fixtures.

The producer MUST emit exactly one `ai.ToolCallStart` per call, carrying that call's `id` and `function.name` **byte-exact**, and MUST emit it **before** any `ai.ToolCallDelta` for that call — `R-ATC-002`'s "readable before any argument byte" is the observable. It MUST NOT trim, case-fold, substitute or mint either value. A later element for an established call that **omits** `id`/`name`, or that **repeats them identically**, MUST be tolerated and MUST NOT produce a second start. A later element carrying a **different non-empty** `id` or `name` for an established call MUST yield a typed malformed-response failure: no cited semantics permit reassigning either, and silently keeping the first or overwriting with the second would record inference as fact.

#### Scenarios

- **S-ATL-013** *[test]* — Given an element carrying `"id":"call_A1"`, `"function":{"name":"search","arguments":"{\"q\":"}` in one chunk, when the stream is drained, then the call's start event precedes its first delta, and the start's `ID()` and `Name()` are `call_A1` and `search` byte for byte.
- **S-ATL-014** *[test]* — Given a call whose second and third elements carry `function.arguments` only — no `id`, no `name`, no `type` — when the stream is drained, then exactly one start event exists for that call, no failure is reported, and both fragments are attributed to it.
- **S-ATL-015** *[test]* — Given a call whose every element repeats the identical `id` and `name` alongside its fragment, when the stream is drained, then exactly one start event exists for that call and the drained event sequence is equal in kind, order and payload to `S-ATL-014`'s twin transcript.
- **S-ATL-016** *[test]* — Given a call whose second element carries `"id":"call_B2"` for an index already established as `call_A1`, when the stream is drained, then the terminal event is a typed malformed-response failure, and the same assertion run against a `name`-mismatch twin transcript fails identically.
- **S-ATL-017** *[test]* — Given a call whose elements carry `index` and `function.arguments` but never any `function.name` before the call closes, when the stream is drained, then the terminal event is a typed malformed-response failure and no start event carrying an empty or minted name was emitted.
- **S-ATL-018** *[inspection]* — Given the shipped source and fixture set, when a reviewer reads them, then the identity-distribution rule is labelled **dialect-conventional** citing **C9.3**'s negative, and the three committed transcripts covering first-element-only, omitted and repeated identity are named at the label.

### R-ATL-004 — Fragments accumulate in per-call buffers, with zero cross-contamination

Each call MUST accumulate its argument fragments in its **own** buffer, keyed by its correlation identity, so that a fragment belonging to one call can never be observed in another's reassembled arguments. This MUST hold when calls interleave arbitrarily — across chunks and within a single chunk's `tool_calls` array — because cross-contamination in parallel tool calls is a shipped-SDK bug class rather than a hypothetical.

#### Scenarios

- **S-ATL-019** *[test]* — Given a transcript interleaving two calls whose fragments carry **distinguishable bytes after every interleave point** — call 0 as `{"q":"al` / `pha"}` and call 1 as `{"city":"be` / `ta"}`, alternating — when the stream is drained, then call 0's end `Arguments()` is exactly `{"q":"alpha"}` and call 1's is exactly `{"city":"beta"}`, with neither containing any byte of the other.
- **S-ATL-020** *[test]* — Given a transcript round-robining **three** calls over three fragments each, every fragment carrying a call-unique marker byte, when the stream is drained, then each call's arguments match its own hard-coded expected literal — a fixture in which swapping any two buffers changes at least one expected value.
- **S-ATL-021** *[test]* — Given a **single** chunk whose `tool_calls` array carries two elements with different `index` values and different fragments, when the stream is drained, then each fragment is attributed to its own call and neither call received both.

### R-ATL-005 — A call closes exactly once, at the choice's terminal chunk or at stream end, and assembles exactly once there

**Dialect-conventional premise carrying a fixture-pin obligation (label 4).** C9.6 is a load-bearing negative: the chunk element has no end, done or count field; no per-call event exists in chat scope; the **only** completeness signals on this wire are the choice-level `finish_reason` transitioning from null (C2) and stream end (C5). The completeness rule below is therefore adapter-derived, and it MUST be pinned by committed transcripts — one closing at the terminal chunk, one truncated before it.

On that premise: every open call MUST close when choice 0's terminal chunk arrives — for **any** member of C2's `finish_reason` enum, not only `tool_calls` — or at clean stream end if the terminal chunk never arrived carrying a well-formed completion. At the close, and only there, the producer MUST assemble that call's accumulated fragments **exactly once** and emit exactly one `ai.ToolCallEnd`. It MUST NOT parse, inspect or partially assemble fragments before the close. Ends for multiple open calls MUST be emitted in **ascending ordinal order**, after any text-block end and strictly **before** the completion event. A stream that ends with a call still open and no terminal chunk is `R-ATL-009`'s truncation case, never a clean close.

#### Scenarios

- **S-ATL-022** *[test]* — Given a call receiving four fragments, when the stream is drained, then exactly one `ToolCallEnd` event exists at that call's block index and its `Arguments()` is byte-equal to the ordered concatenation of the four emitted `Fragment()` values.
- **S-ATL-023** *[test]* — Given a transcript whose chunks deliver a start and three fragments and are then followed by a keep-alive but no terminal chunk yet, when the events emitted up to that point are inspected, then zero `ToolCallEnd` events have been emitted.
- **S-ATL-024** *[test]* — Given a transcript with choice-0 text and two open calls followed by a terminal chunk carrying `"finish_reason":"tool_calls"` and the sentinel, when the stream is drained, then the emitted order is `… TextBlockEnd, ToolCallEnd(ordinal 1), ToolCallEnd(ordinal 2), Completion`, and `ai.CheckStream` reports no violation.
- **S-ATL-025** *[test]* — Given a transcript whose open call is followed by a terminal chunk carrying `"finish_reason":"stop"`, when the stream is drained, then the call still closes with exactly one end event and the stream completes normally — completeness is the terminal chunk, not the reason's value.
- **S-ATL-026** *[test]* — Given a call whose individual fragments are each invalid JSON on their own (`{"path":"/e`, `tc/hosts"}`) but concatenate to a valid payload, when the stream is drained, then the stream completes cleanly and the end's `Arguments()` is `{"path":"/etc/hosts"}` — nothing was parsed before the close.
- **S-ATL-027** *[inspection]* — Given the shipped source and fixture set, when a reviewer reads them, then the completeness rule is labelled **dialect-conventional** citing **C9.6**'s negative, both pinning transcripts (terminal-chunk close and truncated twin) are committed and named, and no comment claims the vendor signals per-call completion.

### R-ATL-006 — Per-call accumulation is bounded by a documented cap, and the tenth category's second consumer is recorded

Per-call accumulation MUST be memory-bounded across fragments: a runaway fragment sequence into **one** call MUST hit a **documented cap** and terminate the stream with a typed failure. AI-27.5's `ErrFrameTooLarge` bounds a single frame; this bounds the **sum** across a call. The cap MUST be a documented **unexported** package constant (`captureLimit`'s mold) and the relation MUST be strict, matching `ErrFrameTooLarge`'s recorded reading: accumulation reaching exactly the cap still completes; exceeding it fails. The cause MUST be an **unexported** value wrapping `ai.ErrMalformedResponse` with `%w` (the `errMalformedIdentity` precedent), so `S-ART-054`'s allowlist stays untouched, and MUST keep `ErrFrameTooLarge`'s **exact compromise form** — naming `ai.FailureCategoryMalformedResponse` for an over-cap payload that may be perfectly well-formed, with the compromise disclosed in the same comment shape.

**The second consumer now exists — recorded, not absorbed.** `errors.go` states the resource-exhaustion category may be appended only once a second consumer of the same concept exists, and names **AI-30.1 item 4** as that consumer. This change MUST record, verbatim-adjacent to that existing escalation note, that the second consumer has landed, and MUST stop there. Widening `ai.FailureCategory` MUST NOT happen in this mapping-only milestone; it belongs to the taxonomy owner as a separately-proposable change.

#### Scenarios

- **S-ATL-028** *[test]* — Given a transcript feeding one call a fragment sequence whose total exceeds the documented cap, when the stream is drained, then the terminal event is a failure whose `Category()` is `ai.FailureCategoryMalformedResponse` and whose cause satisfies `errors.Is(cause, ai.ErrMalformedResponse)`.
- **S-ATL-029** *[test]* — Given a transcript feeding one call exactly the cap's worth of argument bytes and no more, when the stream is drained, then the call closes normally and no failure is reported — the relation is strict.
- **S-ATL-030** *[test]* — Given a transcript feeding **two** calls each just under the cap, whose sum far exceeds it, when the stream is drained, then both calls close normally — the cap is per call, not per stream.
- **S-ATL-031** *[inspection]* — Given the shipped source, when a reviewer reads it, then the cap is a named unexported package constant with a doc comment stating its value and rationale, its cause is unexported and wraps `ai.ErrMalformedResponse`, and the compromise comment names `ai.FailureCategoryMalformedResponse` in `ErrFrameTooLarge`'s existing shape.
- **S-ATL-032** *[inspection]* — Given `errors.go` after this change, when a reviewer reads the escalation note, then it records that AI-30.1 item 4's second consumer now exists, states that widening the taxonomy remains a separate change, and `ai.FailureCategories()` is unchanged at nine members before and after.

---

## AI-30.2 — Empty and zero-fragment calls `[leaf]`

### R-ATL-007 — Empty fragments are no-ops, zero-fragment calls canonicalize to `{}`, and a whole call normalizes identically to its fragmented twin

An element whose `function.arguments` is the **empty string** MUST be a no-op — routine in real transcripts, never an error — and MUST NOT disturb the call's subsequent accumulation. An element whose `function.arguments` key is **absent** MUST behave identically to the empty string; C9.1 makes both shapes legal on every element. A call that closes with **zero accumulated bytes** MUST normalize to `ai`'s canonical empty-arguments form — the two bytes `{}` that `ai.NewToolCall` writes for absent arguments — rather than failing to parse empty input, so that every call this adapter emits carries decodable argument bytes. This is `R-ATC-007`'s own allocation: `NewToolCallEnd` deliberately does **not** canonicalize, and names AI-30's reassembly as the place that does. A call delivered **whole** — its complete arguments in a single element — MUST normalize to the same `ai.ToolCallEnd` arguments as its fragmented twin; `R-ATC-010` makes the fragment count unobservable in the normalized result.

#### Scenarios

- **S-ATL-033** *[test]* — Given a call whose elements carry `""`, then `{"q":`, then `""`, then `"a"}`, when the stream is drained, then the end's `Arguments()` is exactly `{"q":"a"}` and no failure is reported.
- **S-ATL-034** *[test]* — Given a call whose only element carries `index`, `id` and `function.name` and closes with zero accumulated bytes, when the end event is inspected, then its `Arguments()` is byte-equal to the arguments of a part built by `ai.NewToolCall(id, name, nil)` — the two bytes `{}`.
- **S-ATL-035** *[test]* — Given two transcripts identical except that one carries `"arguments":""` where the other omits the key entirely, when both are drained, then the emitted event sequences are equal in kind, order and payload.
- **S-ATL-036** *[test]* — Given a call delivered whole as `{"city":"Caracas","units":"c"}` in one element, and its fragmented twin splitting those exact bytes across five elements, when both are drained, then both ends' `Arguments()` are byte-identical to each other and to the source literal, while the delta counts differ (1 versus 5).
- **S-ATL-037** *[test]* — Given the zero-accumulated-bytes call of `S-ATL-034`, when its events are counted, then exactly zero `ToolCallDelta` events were emitted for its block — a start immediately followed by its end is legal and complete (`R-ATC-009`).

---

## AI-30.3 — Argument-byte fidelity `[leaf]`

### R-ATL-008 — Reassembled arguments are byte-identical to the concatenated fragments, and nothing re-marshals them

**Dialect-conventional premise carrying a fixture-pin obligation (label 3).** C9.4 is explicit that `function.arguments`'s description is copied verbatim from the non-streaming schema and states **nothing** about partial delivery or append semantics; `citations.md` records zero hits for `partial JSON`, `accumulate` and chat-scoped `concatenat`. That fragments concatenate in arrival order is therefore dialect convention, pinned by a committed fragment-concatenation transcript.

The bytes a call's `ai.ToolCallEnd` carries MUST be byte-identical to the ordered concatenation of that call's emitted delta fragments, and both MUST be byte-identical to the transcript's own decoded fragment bytes. The producer MUST NOT re-marshal, re-encode, reorder keys, normalize whitespace, round-trip through a numeric type, or validate as UTF-8 anywhere on the argument path. Exotic-but-legal payloads — escape sequences and extreme numeric forms that have crashed shipped SDK parsers — MUST survive unchanged.

#### Scenarios

- **S-ATL-038** *[test]* — Given a call whose fragments concatenate to `{"path":"C:\\tmp\\x","quote":"say \"hi\"","nl":"a\nb","u":"caf\u00e9"}`, when the end event is inspected, then its `Arguments()` is byte-equal to that exact decoded literal.
- **S-ATL-039** *[test]* — Given a call whose fragments concatenate to `{"big":123456789012345678901234567890,"tiny":1e-320,"neg":-0,"prec":3.141592653589793238462643383279}`, when the end event is inspected, then its `Arguments()` reproduces every digit and sign byte for byte — no float round-trip, no exponent renormalization.
- **S-ATL-040** *[test]* — Given a call whose fragments concatenate to `{ "b" : 2,  "a":1 }` — duplicated spacing and non-alphabetical key order — when the end event is inspected, then its `Arguments()` preserves that spacing and order exactly.
- **S-ATL-041** *[test]* — Given a call whose argument text is split **inside a JSON escape** — one element's decoded fragment ending with the single byte `\` and the next beginning `u00e9"}` — when the end event is inspected, then the concatenation carries the intact `\u00e9` escape and equals the expected literal byte for byte.
- **S-ATL-042** *[test]* — Given any of the fixtures above, when both assertions are run, then the end's `Arguments()` equals the fixture literal **and** equals the concatenation of the emitted `Fragment()` values — two independent comparisons, neither derived from the other.
- **S-ATL-043** *[inspection]* — Given the shipped source, when a reviewer searches the tool-argument path, then it contains no `json.Marshal`, no `json.Compact`, no `json.Indent` and no `string`→number→`string` conversion, and it decodes through the existing `unquoteJSONString` idiom.

---

## AI-30.4 — Truncation and malformation `[leaf]`

### R-ATL-009 — Truncation and malformation both yield one typed failure carrying the raw partial fragment, bounded and never rendered

Two shapes MUST terminate the stream with the **same** typed malformed-response failure family: (1) a stream ending with a call still open and no terminal chunk — a length-limit cutoff mid-arguments; and (2) a call whose accumulated bytes are **not well-formed JSON** at its close. The second is a deliberate policy choice recorded here: C9.4 states outright that the model "does not always generate valid JSON", so a real backend can produce this shape legitimately — the adapter nonetheless fails typed rather than forwarding undecodable bytes, because `tool_call.go` states as an invariant that every constructible tool call's argument bytes are well-formed JSON, which AI-26 and this milestone both decode against.

Both failures MUST be `ai.MidStreamFailure` over `ai.DeliveryMidStream`, report `ai.FailureCategoryMalformedResponse`, and report `PartialOutput()` per whether normalized output preceded them. Both MUST **carry the raw partial fragment**, and the disposition is fixed here rather than left to design:

1. The raw bytes MUST be reachable **only** through a typed cause found with `errors.As`/`Unwrap` — the `nonStreamContentType` precedent for reachability.
2. The bytes MUST be **bounded**, reusing `capture.go`'s existing `captureLimit` thinking rather than a second, independent bound.
3. Neither the outer `ai.Failure`'s `Error()` (frozen category-only rendering, `R-AIP-009`) nor the typed cause's own `Error()` MUST reproduce any argument byte. **This picks `capturedBody`'s leak-averse posture (`R-AEM-016`) over `nonStreamContentType`'s rendering one, and the reason is stated: argument bytes are model-generated and routinely echo user input, so a rendered excerpt lands verbatim in every log line that prints the failure — while an HTML error page's excerpt does not carry user content.**

No input sequence MUST cause a panic, and every event already emitted MUST remain delivered, in order and byte-exact, ahead of the terminal failure. A call MUST NEVER be silently discarded.

#### Scenarios

- **S-ATL-044** *[test]* — Given a transcript delivering a start and two fragments for one call and then closing the connection with no terminal chunk and no sentinel, when the stream is drained, then the terminal event is a failure whose `Category()` is `ai.FailureCategoryMalformedResponse`, whose `Delivery()` is `ai.DeliveryMidStream`, and whose `PartialOutput()` is `true`.
- **S-ATL-045** *[test]* — Given that same failure, when `errors.As` is used to reach this package's typed cause, then the cause is found and its accessor returns bytes byte-equal to the two fragments concatenated.
- **S-ATL-046** *[test]* — Given a call whose fragments concatenate to `{"q":"a"` — closed cleanly by a terminal chunk — when the stream is drained, then the terminal event is the same typed malformed-response failure and its typed cause carries those exact raw bytes.
- **S-ATL-047** *[test]* — Given a truncated call whose accumulated bytes far exceed the documented bound, when the typed cause's bytes are read, then their length is no greater than that bound.
- **S-ATL-048** *[test]* — Given both failures above, when the outer failure's `Error()` and the typed cause's `Error()` are each rendered, then neither string contains any byte sequence from the accumulated fragments — asserted against a fixture whose fragments carry a distinctive marker token.
- **S-ATL-049** *[test]* — Given a table of every failing transcript in this milestone — truncation, malformed assembly, cap exceeded, missing `index`, identity mismatch, missing name — driven through one runner with a `recover()` probe, when all rows run, then zero panics are recorded and every row's terminal event is a failure.
- **S-ATL-050** *[test]* — Given a failing transcript preceded by a complete, successfully closed tool call and two text deltas, when the stream is drained, then those events are present, in order, byte-exact, ahead of the terminal failure.
- **S-ATL-051** *[inspection]* — Given the shipped source, when a reviewer reads the failure path, then the raw-fragment bound is a named constant reusing `captureLimit`, the bytes accessor is unexported, and the cause's `Error()` is a fixed string built from no captured byte.

### R-ATL-010 — A tool call the neutral events cannot represent fails typed, never silently

When a wire-legal tool call cannot be represented by AI-18's neutral events, the producer MUST terminate the stream with a typed malformed-response failure. It MUST NOT drop the call, MUST NOT mint a substitute value for a missing field, and MUST NOT emit a partially-populated start. Because C9.3 states no fragmentation prose for `id`/`name`, the producer MUST NOT concatenate name or identity fragments across elements: doing so would record inference as fact.

#### Scenarios

- **S-ATL-052** *[test]* — Given a call whose first element carries `"name":"sea"` and whose second carries `"name":"rch"` for the same `index`, when the stream is drained, then the terminal event is a typed malformed-response failure and no start event named `sea`, `rch` or `search` was emitted.
- **S-ATL-053** *[test]* — Given a call whose elements carry `index` and `function.arguments` only, never an `id`, up to its close, when the stream is drained, then the terminal event is a typed malformed-response failure and no start event carrying an empty identity was emitted.
- **S-ATL-054** *[test]* — Given a table of every unrepresentable shape above driven through one runner, when each row is drained, then each yields a failure **and** the same assertion **fails** if the producer instead completed the stream with the offending call absent — a silent drop is distinguishable from a typed failure in the same test.

---

## AI-30.5 — Ordinal preservation `[leaf]`

### R-ATL-011 — Each call's ordinal position is observable regardless of fragment interleaving

Each normalized call's **ordinal** — its 1-based position among the stream's tool-call start events, in emission order — MUST be observable from a recorded stream regardless of how its fragments interleave with other calls'. Per `R-ATC-012` this package MUST NOT ship an accumulator: the ordinal is derived stream-side by filtering start events in order, exactly as `agenttest`'s own `reconstructToolCalls` already does. The ordinal MUST remain a distinct axis from both the minted block index and the wire `index` value.

#### Scenarios

- **S-ATL-055** *[test]* — Given a transcript whose calls first appear in wire-`index` order `2`, `0`, `1`, when the recorded stream is reconstructed, then the ordinals are `1`, `2`, `3` assigned to wire indices `2`, `0`, `1` respectively — a fixture whose expected mapping is **not** produced by sorting on `index`.
- **S-ATL-056** *[test]* — Given that same transcript with every call's fragments interleaved round-robin after their starts, when the recorded stream is reconstructed, then the ordinal-to-identity pairs equal the same hard-coded expected literals as `S-ATL-055` — interleaving does not perturb them.
- **S-ATL-057** *[test]* — Given a transcript carrying two calls to the **same** tool name, when the recorded stream is reconstructed, then their ordinals are distinct and strictly ordered by start arrival — the calls are distinguishable when the name is not.
- **S-ATL-058** *[inspection]* — Given the shipped production source, when a reviewer searches it, then no ordinal counter, ordinal field or ordinal accessor exists — ordinal derivation appears only in test sources (`R-ATC-012`).

---

## Acceptance surface and charter

### R-ATL-012 — AI-23.3's four `CapToolCalls` conformance cases pass against real transport

The adapter's test-only conformance bridge MUST be extended to render tool-call events into an SSE transcript, and the capability-scoped conformance entry point MUST report all four `CapToolCalls` cases satisfied against a **real** `*Client` speaking real HTTP to a local `httptest.Server`. This is the milestone's acceptance proof: those four cases (`fragmented_interleaved_reconstructs_exactly`, `zero_delta_whole_call_accepted`, `ordinal_distinguishes_same_tool_name`, `mixed_text_and_tool_ends_on_tool_call_finish_reason`) have never been run against transport. The bridge MUST remain adapter-owned and test-only, MUST NOT modify `src/agenttest/`, and MUST NOT weaken, fork or skip a conformance case; if a case cannot pass unmodified, the node **STOPS** and the reconciliation is raised as its own change against `ai-provider-conformance-suite`.

#### Scenarios

- **S-ATL-059** *[test]* — Given the adapter-owned factory with `CapToolCalls` declared, when the capability-scoped conformance entry point is run over the `CapToolCalls` cases, then all four report satisfied and the run performs no outbound network access beyond `127.0.0.1`.
- **S-ATL-060** *[test]* — Given a scripted `ai.ToolCallEnd` carrying `{"q":"a\\nb"}`, when the bridge renders it into a transcript and the producer replays and drains it, then the drained end's `Arguments()` is byte-equal to the scripted bytes — the bridge itself neither re-marshals nor canonicalizes.
- **S-ATL-061** *[inspection]* — Given the shipped sources, when a reviewer reads them, then the bridge exists only in `_test.go` files of `openaicompat`, no file under `src/agenttest/` was modified by this change, and no conformance case was edited, forked or skipped.

### R-ATL-013 — The deprecated `function_call` streaming delta is a recorded skip: tolerated, ignored, never mapped

The producer MUST NOT map the deprecated `function_call` streaming delta in either direction. It MUST tolerate its presence — no failure, no perturbation of adjacent mapping — and MUST emit no tool-call event from it. The disposition is **recorded, not accidental** (dialect-conventional label 5): C9.5 establishes that this object is deprecated, is an anonymous inline object with no named schema, carries **no `index`**, has no `required` list, and — like C9.4 — states zero fragmentation semantics; C7 records the deprecation on the delta schema itself. Without an `index` there is no correlation key, and with no fragmentation prose there is no defensible accumulation rule, so mapping it would be pure invention. **Reopen trigger, stated so it is checkable**: if a real backend is proven to emit `function_call` deltas **exclusively** — no `tool_calls` array at all — this disposition is reopened as its own change, never patched in silently here.

#### Scenarios

- **S-ATL-062** *[test]* — Given a transcript of `content:"alpha"`, then a chunk whose choice-0 delta carries only `"function_call":{"name":"search","arguments":"{}"}`, then `content:"omega"`, when the stream is drained, then exactly two text deltas are emitted in order, no tool-call event of any kind exists, and no failure is reported.
- **S-ATL-063** *[test]* — Given a chunk whose choice-0 delta carries **both** a `tool_calls` array and a `function_call` object, when the stream is drained, then the `tool_calls` half maps normally and the `function_call` half contributes nothing.
- **S-ATL-064** *[inspection]* — Given the shipped source, when a reviewer reads the `function_call` disposition, then it cites **C9.5** and C7's deprecation, states the skip as deliberate rather than unimplemented, and names the exclusive-emission reopen trigger.

### R-ATL-014 — `R-ATS-026`'s tool-call clause and `S-ATS-100`'s tool-call half are discharged, not amended

`R-ATS-026` (sibling change `cachicamas-ai-provider-text-stream`, **unarchived**) reads, verbatim: *"This milestone MUST NOT map `tool_calls` or `function_call` deltas in either direction (AI-30)."* That is a **charter boundary scoped to AI-28's own shipped slice**, and it names AI-30 as the owner of the deferred behavior. This change therefore **discharges** it for `tool_calls`; it MUST NOT amend, weaken or rewrite it, and no promoted spec exists to delta. Its only live artifact is `S-ATS-100`, an inspection scenario — verified to have **no executable guard** encoding it, so nothing flips as `S-APC-030/031` did. The sanctioned reconciliation is documentary: this requirement records the discharge, and `doc.go` carries a correction in the same disclose-a-correction form the package already uses. AI-29 makes the **same-shaped, disjoint** statement for reasoning; the two MUST NOT both rewrite one shared clause.

#### Scenarios

- **S-ATL-065** *[inspection]* — Given `doc.go` after this change, when a reviewer reads the correction, then it quotes `R-ATS-026`'s tool-call clause, names AI-30 as its owner, states the discharge as scoped to `tool_calls` only, and leaves the reasoning half to AI-29's own disjoint statement.
- **S-ATL-066** *[inspection]* — Given `openaicompat/*_test.go` before and after this change, when a reviewer searches them, then no test asserting the absence of tool-call emission existed to flip, none was deleted or skipped, and `S-ATS-100`'s reasoning half remains untouched.

### R-ATL-015 — The charter boundary holds: bytes not meaning, no new sentinel, no widened taxonomy, no new dependency

This milestone MUST NOT validate arguments against a tool's schema and MUST NOT execute anything — bytes, not meaning. It MUST NOT export a new sentinel identity, so `S-ART-054`'s allowlist (today exactly `ErrFrameTooLarge` / `ErrTruncated`) is untouched and AI-32's future `ErrInBandErrorFrame` entry has no position to collide with. It MUST NOT widen `ai.FailureCategory`. It MUST NOT map reasoning content (AI-29) or implement usage/finish-reason mapping (AI-31). `backend/agent/go.mod` MUST remain at **zero** requires. It MUST NOT modify `src/ai` or `src/agenttest`.

#### Scenarios

- **S-ATL-067** *[inspection]* — Given `backend/agent/go.mod` before and after this change, when the two are compared, then the file is byte-identical and its require set is empty.
- **S-ATL-068** *[inspection]* — Given the shipped package's exported identifiers before and after, when they are compared, then no new exported error value exists, `S-ART-054`'s allowlist is byte-identical, and `ai.FailureCategories()` reports the same nine members.
- **S-ATL-069** *[inspection]* — Given the shipped production sources, when a reviewer searches them, then no code path validates arguments against a schema, invokes a tool, maps a reasoning field, or performs a usage merge, and every import is stdlib or an in-repo `src/ai` path.
- **S-ATL-070** *[test]* — Given `src/ai` and `src/agenttest` before and after this change, when their suites are run with `-race -count=1`, then no file in either package was modified and both suites pass unchanged.

### R-ATL-016 — Every wire-shape claim resolves to a citation or to a labelled, fixture-pinned dialect convention

Every statement of wire shape in this document, in the shipped source and in its tests MUST cite a claim from this change's `citations.md` (**C9.1 … C9.6**) or AI-28's **C1 … C8**, or be explicitly labelled **dialect-conventional** with a **named pinning fixture**. Given C9.2, C9.3, C9.4 and C9.6, **most** of this milestone's accumulation semantics are dialect-conventional, and the spec states that wholesale rather than case by case: the five labels are enumerated in "The five dialect-conventional labels this spec carries". A citation reference that does not resolve is a defect. Inference MUST NOT be recorded as fact.

#### Scenarios

- **S-ATL-071** *[inspection]* — Given every `C9.x` and `C1` … `C8` reference in this spec and in the shipped source comments, when each is resolved against the corresponding `citations.md`, then every reference names an existing claim and none names a claim number outside those ranges.
- **S-ATL-072** *[inspection]* — Given every wire-shape statement in the shipped source not backed by a `C`-claim, when a reviewer reads it, then it carries an explicit dialect-conventional label naming one of this spec's five labels, and that label's pinning fixture is committed under the package's fixture set.
- **S-ATL-073** *[inspection]* — Given this change's `citations.md` pinned commit, when it is compared against `doc.go`'s existing request-side provenance pin and the AI-28 citation table's, then all three name `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439` — one pin, not three.

---

## Coverage of doc 0002's test lists

Every test-list item in doc 0002 §§ AI-30.1 … AI-30.5 maps to at least one scenario.

| Doc 0002 item | Scenarios |
| --- | --- |
| AI-30.1 #1 — interleaved fragments accumulate per call, keyed by block position | `S-ATL-006` … `S-ATL-012`, `S-ATL-019` … `S-ATL-021` |
| AI-30.1 #2 — identity and name available from the start event, before any argument byte | `S-ATL-013` … `S-ATL-018` |
| AI-30.1 #3 — assembly exactly once, at the call's end; never partially parsed | `S-ATL-022` … `S-ATL-027` |
| AI-30.1 #4 — per-call accumulation bounded by a documented cap, typed failure | `S-ATL-028` … `S-ATL-032` |
| AI-30.2 #1 — an empty first fragment is a no-op | `S-ATL-033`, `S-ATL-035` |
| AI-30.2 #2 — zero accumulated bytes normalize to the canonical empty-arguments form | `S-ATL-034`, `S-ATL-037` |
| AI-30.2 #3 — a whole call normalizes identically to its fragmented equivalent | `S-ATL-036` |
| AI-30.3 #1 — byte-identical reassembly, including exotic-but-legal payloads | `S-ATL-038`, `S-ATL-039`, `S-ATL-041` |
| AI-30.3 #2 — the end event carries exact bytes; nothing re-marshals them | `S-ATL-040`, `S-ATL-042`, `S-ATL-043` |
| AI-30.4 #1 — mid-arguments cutoff → typed failure carrying the raw partial fragment, never a panic | `S-ATL-044`, `S-ATL-045`, `S-ATL-047` … `S-ATL-050` |
| AI-30.4 #2 — a malformed assembled payload yields the same typed, raw-carrying failure | `S-ATL-046` |
| AI-30.5 #1 — each call's ordinal observable regardless of interleaving | `S-ATL-055` … `S-ATL-058` |
| Element decode gate (proposal § "The citation gate") | `S-ATL-001` … `S-ATL-005`, `S-ATL-071` … `S-ATL-073` |
| Unrepresentable calls (coordinator ruling) | `S-ATL-052` … `S-ATL-054` |
| Acceptance proof (proposal § in-scope 6) | `S-ATL-059` … `S-ATL-061` |
| `function_call` recorded skip (coordinator ruling) | `S-ATL-062` … `S-ATL-064` |
| `R-ATS-026` discharge (proposal § "How `R-ATS-026` is superseded") | `S-ATL-065`, `S-ATL-066` |
| Charter boundary (proposal § out of scope) | `S-ATL-067` … `S-ATL-070` |

## Non-vacuity discipline

Scenarios above are written so their fixtures **can distinguish an implemented producer from an unimplemented one**, against the nine recorded vacuous-pass shapes. Concretely, by id:

- **Shape 1 (cannot distinguish implemented from not).** `S-ATL-054` asserts a silent drop and a typed failure differ **in the same test**; `S-ATL-060` replays through real transport rather than comparing scripted events to themselves.
- **Shape 2 (implementation compared against itself).** `S-ATL-042` runs two independent comparisons — against the fixture literal **and** against the concatenated fragments — neither derived from the other; `S-ATL-034` compares against `ai.NewToolCall`'s own canonicalization, a different code path.
- **Shape 4 (no meaningful content after the interesting position).** `S-ATL-019` and `S-ATL-020` place **distinguishable bytes in EVERY call after EVERY interleave point**, so a producer that dropped a tail or merged buffers cannot pass; `S-ATL-041` places seven further bytes after the split escape.
- **Shape 5 (cannot distinguish correct from misrouted-but-unobserved).** `S-ATL-020`'s three-call fixture gives each call a unique marker byte, so swapping any two buffers changes at least one hard-coded expected value; `S-ATL-009` asserts tool blocks never take the text block's index **and** that the text twin is unchanged.
- **Shape 7 (sort coincidence).** `S-ATL-055` and `S-ATL-056` use first-appearance order `2, 0, 1`, whose expected ordinal mapping is **not** reproducible by sorting on `index`; `S-ATL-057` gives both calls the same tool name so name-based grouping cannot substitute for ordinal derivation.
- **Shape 3 / 8 (unobservable magnitude, same fixture regardless of case).** `S-ATL-029` and `S-ATL-028` sit either side of the cap's strict boundary with different expected outcomes; `S-ATL-035` and `S-ATL-036` build genuinely different transcripts and assert equality of the **result**, not of the fixture.
- **Shape 9 (mutation that only disables a check).** `S-ATL-048` asserts a distinctive marker token is absent from **both** rendered strings, so a mutation that merely removed the redaction is caught positively rather than by an unrelated failure.

Byte-fidelity fixtures carry the exotic-but-legal payloads doc 0002 names — escape sequences (`S-ATL-038`, `S-ATL-041`) and extreme numeric forms (`S-ATL-039`). A scenario whose fixture would pass identically against a no-op producer is a spec defect and MUST be rewritten, not re-marked `[inspection]` to avoid the problem.
