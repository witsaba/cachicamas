# Citations — OpenAI-compatible streaming tool-call delta element (AI-30)

## Provenance

- **Repository:** `github.com/openai/openai-openapi`
- **Pinned commit:** `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439` (same commit cited by `backend/agent/src/ai/openaicompat/doc.go` and by the AI-28/AI-32 citation tables)
- **File:** `openapi.yaml` (84,002 lines; single spec file at this commit)
- **Retrieval date:** 2026-08-04
- **Retrieval method:** `curl -sSL https://raw.githubusercontent.com/openai/openai-openapi/d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439/openapi.yaml` (HTTP 200, 2,845,527 bytes), downloaded once into the session scratchpad and reused. All line numbers refer to `openapi.yaml` at this exact commit; every quote and count was extracted with `grep`/`sed`/`awk` over the downloaded copy.

Context anchor: the streaming delta's `tool_calls` field (`ChatCompletionStreamResponseDelta.tool_calls`, lines 31504-31507) is declared as a plain array whose items `$ref` `#/components/schemas/ChatCompletionMessageToolCallChunk` — no `oneOf`, no discriminator, and no description on the array itself.

---

## C9 — Streaming tool-call delta element

### C9.1 — `ChatCompletionMessageToolCallChunk`: complete property set and required list

The whole schema:

> `openapi.yaml:30700-30728`
> ```yaml
>     ChatCompletionMessageToolCallChunk:
>       type: object
>       properties:
>         index:
>           type: integer
>         id:
>           type: string
>           description: The ID of the tool call.
>         type:
>           type: string
>           enum:
>             - function
>           description: The type of the tool. Currently, only `function` is supported.
>           x-stainless-const: true
>         function:
>           type: object
>           properties:
>             name:
>               type: string
>               description: The name of the function to call.
>             arguments:
>               type: string
>               description: The arguments to call the function with, ...
>       required:
>         - index
> ```

(The `arguments` description is abridged above; it is quoted in full in C9.4. Lines 30722-30726 in the original.)

**Reading:** the element declares exactly four properties — `index` (integer), `id` (string), `type` (const-enum `function`), and `function` (object with optional `name` and `arguments` strings) — and the `required` list contains **only `index`**; `id`, `type`, and `function` (and both nested function fields) are all optional on every element.

### C9.2 — The per-element `index` field

> `openapi.yaml:30703-30704` and `30727-30728`
> ```yaml
>         index:
>           type: integer
>       required:
>         - index
> ```

**Reading:** `index` is present and is the **sole required** field, but it carries **no description whatsoever** at this commit — the spec never states what it indexes. (Contrast: the Assistants run-step delta tool-call objects do describe their index as "The index of the tool call in the tool calls array", lines 60675, 60757, 60782 — those schemas are `RunStepDeltaStepDetailsToolCalls*Object`, not reachable from chat completions.) That `index` correlates fragments of the same call across chunks is dialect convention, not spec text — it needs fixture pins.

### C9.3 — Are `id`/`function.name` first-element-only? Can they arrive fragmented? (NEGATIVE)

**Finding: NEGATIVE — the spec is silent in both directions.** The descriptions say only:

> `openapi.yaml:30705-30707` and `30717-30719`
> ```yaml
>         id:
>           type: string
>           description: The ID of the tool call.
>         ...
>             name:
>               type: string
>               description: The name of the function to call.
> ```

No prose anywhere in the file states that `id` or `name` appears only on the first element of a call, that they repeat on every element, or that either value may itself arrive fragmented across elements. Searches across all 84,002 lines (case-insensitive): `only the first` → 0; `first chunk` → 0; `first delta` → 0; `subsequent chunk` → 0; `subsequent delta` → 0; `fragment` → 2 (both Realtime transcript deltas, 55976/56091: "Transcript deltas are append-only text fragments"); `concatenat` → 1 (audio transcript, 36741); `accumulate` → 0.

**Reading (load-bearing negative):** the first-element-carries-`id`/`name` behavior every OpenAI-compatible provider exhibits is entirely undocumented at this commit — the adapter must treat "which element carries `id`/`name`, and whether they can fragment" as dialect behavior pinned by fixtures, and tolerate both presence and absence on any element (all fields being optional per C9.1 makes any distribution schema-valid).

### C9.4 — `function.arguments` fragment semantics and the `type` enum

**`function.arguments` — full description:**

> `openapi.yaml:30720-30726`
> ```yaml
>             arguments:
>               type: string
>               description: The arguments to call the function with, as generated by the model
>                 in JSON format. Note that the model does not always generate
>                 valid JSON, and may hallucinate parameters not defined by your
>                 function schema. Validate the arguments in your code before
>                 calling your function.
> ```

**`type` enum:**

> `openapi.yaml:30708-30713`
> ```yaml
>         type:
>           type: string
>           enum:
>             - function
>           description: The type of the tool. Currently, only `function` is supported.
>           x-stainless-const: true
> ```

**Reading:** `arguments` is a plain string whose description is copied verbatim from the non-streaming tool call — it says nothing about partial fragments or append semantics (the words "partial"/"appended" never occur in chat tool-call context; see Search log), so argument-concatenation across chunks is dialect convention; `type` has exactly one enum member, `function`, and no streaming counterpart exists for custom tool calls (`CustomToolCallChunk` → 0 hits; the non-streaming `ChatCompletionMessageToolCalls` union at 30729-30737 offers `oneOf` function/custom, but the streaming array items reference only `ChatCompletionMessageToolCallChunk`).

### C9.5 — The deprecated `function_call` delta object (streaming side)

Declared inline inside `ChatCompletionStreamResponseDelta` (schema at 31479):

> `openapi.yaml:31488-31503`
> ```yaml
>         function_call:
>           deprecated: true
>           type: object
>           description: Deprecated and replaced by `tool_calls`. The name and arguments of
>             a function that should be called, as generated by the model.
>           properties:
>             arguments:
>               type: string
>               description: The arguments to call the function with, as generated by the model
>                 in JSON format. Note that the model does not always generate
>                 valid JSON, and may hallucinate parameters not defined by your
>                 function schema. Validate the arguments in your code before
>                 calling your function.
>             name:
>               type: string
>               description: The name of the function to call.
> ```

**Reading:** the streaming `function_call` delta is an anonymous inline object (no named schema, unlike the tool-call chunk), deprecated, with optional `arguments` and `name` strings, no `required` list, no `index` (it models a single call), and — like C9.4 — zero fragmentation semantics in its descriptions; its `arguments`/`name` text is byte-identical to the tool-call chunk's.

### C9.6 — Call-completeness signaling (NEGATIVE for any per-call signal)

**The `finish_reason` description — the only completeness-adjacent text in chat scope:**

> `openapi.yaml:33344-33356` (choice-level `finish_reason` in `CreateChatCompletionStreamResponse`; enum at 33357-33363 with `nullable: true`)
> ```yaml
>                 description: >
>                   The reason the model stopped generating tokens. This will be
>                   `stop` if the model hit a natural stop point or a provided
>                   stop sequence,
>                   `length` if the maximum number of tokens specified in the
>                   request was reached,
>                   `content_filter` if content was omitted due to a flag from our
>                   content filters,
>                   `tool_calls` if the model called a tool, or `function_call`
>                   (deprecated) if the model called a function.
> ```

**Finding: NEGATIVE for any per-call end marker.** No schema or prose defines when an individual streamed tool call is complete: the chunk element (30700-30728) has no end/done/count field; the delta schema (31479-31521) and chunk schema (33295-33440) declare no per-call event; `tool call is complete` → 2 hits, both the Responses API `ResponseCustomToolCallInputDoneEvent` ("Event indicating that input for a custom tool call is complete.", 57674) and its beta twin (79018), neither reachable from chat completions; `appended` → 4 hits, all Realtime conversation items. The only completeness signals available on the chat stream are (a) the choice-level `finish_reason` transitioning from `null` to `tool_calls` (per the enum semantics above and the AI-28 C2 finding that `finish_reason` is null mid-stream) and (b) stream end (`data: [DONE]`, AI-28 C5).

**Reading (load-bearing negative):** the adapter must derive per-call completeness itself — a tool call is finishable only when the choice's `finish_reason` arrives (or the stream terminates); the spec provides no per-call end marker, count field, or done event for Chat Completions streaming.

---

## Negative findings

1. **`index` has no description (C9.2).** The sole required field of `ChatCompletionMessageToolCallChunk` carries no prose at this commit; its role as a fragment-correlation key is dialect convention needing fixture pins. The "index of the tool call in the tool calls array" descriptions exist only on Assistants run-step delta schemas (60675, 60757, 60782).
2. **No first-element or fragmentation prose for `id`/`name` (C9.3).** `only the first` → 0; `first chunk` → 0; `first delta` → 0; `subsequent chunk`/`subsequent delta` → 0; `fragment` → 2 (Realtime transcripts only); `concatenat` → 1 (audio transcript); `accumulate` → 0. All chunk-element fields except `index` are optional, so any distribution across elements is schema-valid.
3. **No fragment/append semantics for `arguments` (C9.4, C9.5).** The `arguments` descriptions (30720-30726, 31494-31500) are copied from the non-streaming schema and never mention partial delivery; `partial JSON` → 0; the 108 case-insensitive `partial` hits are batches, images, Realtime, and the legacy `stream` prose — none chat tool-call scoped.
4. **No streaming custom-tool-call element (C9.4).** `CustomToolCallChunk` → 0 hits; streaming `tool_calls` items reference only `ChatCompletionMessageToolCallChunk` (31504-31507, no `oneOf`), while the non-streaming union (30729-30737) includes `ChatCompletionMessageCustomToolCall`.
5. **No per-call completeness signal (C9.6).** `tool call is complete` → 2 hits, both Responses API custom-tool-call events (57674, 79018); `appended` → 4 (Realtime); no end marker, count, or done event exists in chat scope — completeness is choice-level `finish_reason: "tool_calls"` plus stream end only.

## Search log

All commands run over the downloaded `openapi.yaml` in the session scratchpad. Pattern → hit count (grep, case-insensitive where noted):

| Pattern / command | Hits |
|---|---|
| `ChatCompletionMessageToolCallChunk` | 2 (schema def 30700; sole ref 31507) |
| `only the first` (ci) | 0 |
| `first chunk` (ci) | 0 |
| `first delta` (ci) | 0 |
| `subsequent chunk` (ci) | 0 |
| `subsequent delta` (ci) | 0 |
| `fragment` (ci) | 2 (55976, 56091 — Realtime transcript deltas) |
| `partial JSON` (ci) | 0 |
| `partial` (ci) | 108 (batches, images, Realtime, legacy stream prose; none chat tool-call scoped — first 12 hits inspected: 1913, 8431, 8569, 28760, 33625, 35055, 35179, 37796, ...) |
| `appended` (ci) | 4 (46268, 46658, 49465, 49862 — all Realtime conversation items) |
| `concatenat` (ci) | 1 (36741 — audio transcript) |
| `accumulate` (ci) | 0 |
| `delta.tool_calls` (ci) | 0 |
| `streamed tool` (ci) | 0 |
| `tool call is complete` (ci) | 2 (57674 ResponseCustomToolCallInputDoneEvent; 79018 beta twin) |
| `CustomToolCallChunk` | 0 |
| `index of the tool` | 3 (60675, 60757, 60782 — Assistants `RunStepDeltaStepDetailsToolCalls*Object`) |
| awk lines 31504-31507 (delta `tool_calls` declaration) | plain array, items `$ref` only, no description |
| awk lines 30700-30728 (full chunk schema readback) | 4 properties, `required: [index]` |
