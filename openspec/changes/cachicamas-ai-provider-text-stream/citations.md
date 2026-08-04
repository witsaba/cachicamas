# Citations — OpenAI-compatible streaming response wire shape (AI-28)

## Provenance

- **Repository:** `github.com/openai/openai-openapi`
- **Pinned commit:** `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439` (same commit cited by the request-side claims in `backend/agent/src/ai/openaicompat/doc.go`)
- **File:** `openapi.yaml` (84,002 lines; single spec file at this commit)
- **Retrieval date:** 2026-08-04
- **Retrieval method:** `curl -sSL https://raw.githubusercontent.com/openai/openai-openapi/d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439/openapi.yaml` (HTTP 200, 2,845,527 bytes). All line numbers below refer to `openapi.yaml` at this exact commit; every quote was extracted from the downloaded copy with `grep`/`sed`/`awk`.

Scope note: this table cites the **Chat Completions** streaming surface (`POST /chat/completions`, `text/event-stream`). The same spec file also defines the Responses API and Assistants API streaming surfaces; where their schemas are cited below, it is only to establish negative findings about the Chat Completions stream.

---

## C1 — Streaming chunk object: schema name, `object` discriminator, required vs optional top-level fields

**Schema name:** `CreateChatCompletionStreamResponse`. It is the declared `text/event-stream` response schema of `POST /chat/completions`.

> `openapi.yaml:2151-2156`
> ```yaml
>             text/event-stream:
>               schema:
>                 $ref: "#/components/schemas/CreateChatCompletionStreamResponse"
> ```

> `openapi.yaml:33295-33297`
> ```yaml
>     CreateChatCompletionStreamResponse:
>       type: object
>       description: |
> ```

**`object` discriminator:**

> `openapi.yaml:33387-33392`
> ```yaml
>         object:
>           type: string
>           description: The object type, which is always `chat.completion.chunk`.
>           enum:
>             - chat.completion.chunk
>           x-stainless-const: true
> ```

**Required top-level fields:**

> `openapi.yaml:33424-33429`
> ```yaml
>       required:
>         - choices
>         - created
>         - id
>         - model
>         - object
> ```

**Reading:** the chunk schema is `CreateChatCompletionStreamResponse` with `object` fixed to `"chat.completion.chunk"`; `id`, `model`, `created`, `choices`, and `object` are required, while `usage` (nullable, lines 33393-33395), `system_fingerprint` (deprecated `type: string`, lines 33377-33378), `service_tier` (line 33375), and `moderation` (line 33414) are optional.

---

## C2 — Per-choice shape: `index`, `delta`, `finish_reason`, delta schema, `finish_reason` enum and null behavior

**Choice item required fields and delta reference:**

> `openapi.yaml:33313-33321`
> ```yaml
>           items:
>             type: object
>             required:
>               - delta
>               - finish_reason
>               - index
>             properties:
>               delta:
>                 $ref: "#/components/schemas/ChatCompletionStreamResponseDelta"
> ```

**`finish_reason` enum with nullability:**

> `openapi.yaml:33357-33366`
> ```yaml
>                 enum:
>                   - stop
>                   - length
>                   - tool_calls
>                   - content_filter
>                   - function_call
>                 nullable: true
>               index:
>                 type: integer
>                 description: The index of the choice in the list of choices.
> ```

The `finish_reason` prose (lines 33344-33356) reads: "The reason the model stopped generating tokens. This will be `stop` if the model hit a natural stop point or a provided stop sequence, `length` if the maximum number of tokens specified in the request was reached, `content_filter` if content was omitted due to a flag from our content filters, `tool_calls` if the model called a tool, or `function_call` (deprecated) if the model called a function."

**Null behavior mid-stream:** `nullable: true` (line 33363) plus the canonical example at `openapi.yaml:33434-33440` shows `"finish_reason":null` on every chunk until the terminal chunk carries `"finish_reason":"stop"` with an empty `"delta":{}`.

**Reading:** each choice requires `delta`, `finish_reason`, and `index`; `finish_reason` is one of {`stop`, `length`, `tool_calls`, `content_filter`, `function_call`} and stays JSON `null` mid-stream; the optional fourth choice field is `logprobs` (nullable object, lines 33322-33341). The delta schema is `ChatCompletionStreamResponseDelta` — see C7 for its full field set (no `required` list; every delta field is optional).

---

## C3 — Block/content-part START or STOP signals in the Chat Completions stream (NEGATIVE — load-bearing)

**Finding: NEGATIVE.** The Chat Completions stream schema defines no block, content-part, start, or stop framing events of any kind. Text arrives solely as flat `delta.content` string fragments (C7), boundaries are signaled only implicitly (`role` on the first delta, empty `delta` + non-null `finish_reason` on the last chunk, per the example at lines 33434-33440), and the only chat-scoped stream schemas in the entire file are `ChatCompletionStreamOptions` (line 31425) and `ChatCompletionStreamResponseDelta` (line 31479):

> `openapi.yaml` — `grep -n "^    ChatCompletionStream" openapi.yaml`
> ```
> 31425:    ChatCompletionStreamOptions:
> 31479:    ChatCompletionStreamResponseDelta:
> ```

`content_part.added` / `content_part.done` events **do** exist at this commit, but exclusively in the Responses API (`response.content_part.added`, e.g. lines 18015-18016, and schemas `ResponseContentPartDoneEvent` at 57512 / `BetaResponseContentPartDoneEvent` at 81628); none is referenced by `CreateChatCompletionStreamResponse` or reachable from the `POST /chat/completions` `text/event-stream` response (lines 2151-2156, which reference only `CreateChatCompletionStreamResponse`).

Searches that came up empty across all 84,002 lines: `content_block` (0), `block_start` (0), `block_stop` (0), `message_start` (0), `message_stop` (0), `part_added` (0); plus zero hits for `event` or `error` inside the chunk schema body (lines 33295-33440) and the delta schema body (lines 31479-31521).

**Reading:** the adapter must mint block boundaries itself — the wire provides no START/STOP content-part signal in Chat Completions streaming, only flat `delta.content` strings.

---

## C4 — Usage in streaming with `stream_options.include_usage`

**`ChatCompletionStreamOptions.include_usage`:**

> `openapi.yaml:31433-31449` (reflowed from YAML folded scalar)
> ```yaml
>             include_usage:
>               type: boolean
>               description: >
>                 If set, an additional chunk will be streamed before the `data:
>                 [DONE]` message. The `usage` field on this chunk shows the token
>                 usage statistics for the entire request, and the `choices` field
>                 will always be an empty array.
>                 All other chunks will also include a `usage` field, but with a
>                 null value. **NOTE:** If the stream is interrupted, you may not
>                 receive the final usage chunk which contains the total token
>                 usage for the request.
> ```

**Chunk-side `usage` field:**

> `openapi.yaml:33393-33405` (reflowed)
> ```yaml
>         usage:
>           $ref: "#/components/schemas/CompletionUsage"
>           nullable: true
>           description: >
>             An optional field that will only be present when you set
>             `stream_options: {"include_usage": true}` in your request. When
>             present, it contains a null value **except for the last chunk**
>             which contains the token usage statistics for the entire request.
> ```

Corroborating: the `choices` description (lines 33306-33312) says choices "Can also be empty for the last chunk if you set `stream_options: {"include_usage": true}`", and `choices` items require `delta`/`finish_reason`/`index` yet an empty array trivially satisfies that.

**Reading:** with `include_usage: true`, one extra chunk arrives before `data: [DONE]` carrying `usage` (`CompletionUsage`) and an **empty** `choices` array, while every preceding chunk carries `usage: null`; the final usage chunk is not guaranteed if the stream is interrupted or cancelled (lines 33408-33413).

---

## C5 — Terminal sentinel `[DONE]`

**Exact literal form:** `data: [DONE]` (the SSE data payload is the 6-character literal `[DONE]`).

Chat-scoped documentation (inside `ChatCompletionStreamOptions.include_usage`):

> `openapi.yaml:31436-31437`
> ```yaml
>                 If set, an additional chunk will be streamed before the `data:
>                 [DONE]` message.
> ```

Explicit termination statement (legacy `/v1/completions` `stream` parameter — the same SSE protocol, quoted as the spec's clearest statement of the terminator):

> `openapi.yaml:33623-33625`
> ```yaml
>         stream:
>           description: |
>             Whether to stream back partial progress. If set, tokens will be sent as data-only [server-sent events](...) as they become available, with the stream terminated by a `data: [DONE]` message.
> ```

Literal pinned as a schema constant (Assistants API `DoneEvent`):

> `openapi.yaml:37580-37598` (abridged)
> ```yaml
>     DoneEvent:
>       type: object
>       properties:
>         event:
>           type: string
>           enum:
>             - done
>         data:
>           type: string
>           enum:
>             - "[DONE]"
>       description: Occurs when a stream ends.
> ```

**Reading:** the stream terminator is the SSE frame `data: [DONE]`; for Chat Completions it is documented in prose (the include_usage description at 31436-31437 presupposes it), not as a schema constant — the only schema-level `"[DONE]"` constant at this commit belongs to the Assistants `DoneEvent`. The Chat Completions `stream` request parameter description itself (lines 32952-32959) does not mention `[DONE]`; it defers to external streaming docs.

---

## C6 — Mid-stream error frames in Chat Completions streaming (NEGATIVE)

**Finding: NEGATIVE.** The Chat Completions streaming response defines no in-stream error payload shape. The `POST /chat/completions` operation declares exactly one response, `"200"`, whose `text/event-stream` schema is `CreateChatCompletionStreamResponse` alone (lines 2147-2156) — no error event union. The chunk schema body (33295-33440) and delta schema body (31479-31521) contain zero occurrences of `error`.

An in-stream `ErrorEvent` **does** exist at this commit, but only as a member of the Assistants API stream union:

> `openapi.yaml:37862-37876` (abridged)
> ```yaml
>     ErrorEvent:
>       type: object
>       properties:
>         event:
>           type: string
>           enum:
>             - error
>         data:
>           $ref: "#/components/schemas/Error"
>       description: Occurs when an [error](/docs/guides/error-codes#api-errors) occurs.
>         This can happen due to an internal server error or a timeout.
> ```

Its only stream-union reference is `AssistantStreamEvent` (`oneOf` includes `ErrorEvent` at line 28622; union declared at 28574). The Responses API separately has `ResponseErrorEvent` (57751) / `BetaResponseErrorEvent` (81507). Neither is reachable from the Chat Completions endpoint.

Searches used: `ErrorEvent` (referenced only at 28622 within `AssistantStreamEvent`), `error` scoped to lines 33295-33440 (0 hits) and 31479-31521 (0 hits), plus inspection of the full `POST /chat/completions` responses block (2147-2156, single `200` response).

**Reading:** the adapter cannot rely on a schema-defined mid-stream error frame from OpenAI-compatible Chat Completions streams; the spec models no such payload for this endpoint at the pinned commit.

---

## C7 — Full field set of the delta schema (`content` vs `refusal` and every sibling)

`ChatCompletionStreamResponseDelta` (lines 31479-31521) declares exactly five properties and **no `required` list** — every field is optional:

> `openapi.yaml:31479-31521` (structure; prose descriptions abridged)
> ```yaml
>     ChatCompletionStreamResponseDelta:
>       type: object
>       description: A chat completion delta generated by streamed model responses.
>       properties:
>         content:            # 31483 — anyOf: string ("The contents of the chunk message.") | "null"
>         function_call:      # 31488 — deprecated: true, object {arguments: string, name: string}
>         tool_calls:         # 31504 — array of ChatCompletionMessageToolCallChunk
>         role:               # 31508 — string enum: developer, system, user, assistant, tool
>         refusal:            # 31517 — anyOf: string ("The refusal message generated by the model.") | "null"
> ```

Verbatim for the two content-bearing fields:

> `openapi.yaml:31483-31488` and `31517-31521`
> ```yaml
>         content:
>           anyOf:
>             - type: string
>               description: The contents of the chunk message.
>             - type: "null"
>         refusal:
>           anyOf:
>             - type: string
>               description: The refusal message generated by the model.
>             - type: "null"
> ```

**Reading:** the real delta field set the spec must tolerate is exactly {`content`, `function_call` (deprecated), `tool_calls`, `role`, `refusal`} — all optional, with `content` and `refusal` additionally nullable, `role` restricted to the five-value enum at 31508-31515, and no other siblings (no `reasoning`, no `audio`, no `annotations` in this schema at this commit). Note: `include_obfuscation` (lines 31450-31473) documents that providers may add an `obfuscation` field to "streaming delta events" by default — an undeclared-property tolerance case even though the delta schema does not declare it.

---

## C8 — Streaming usage object: `CompletionUsage` field names, required vs optional, and schema identity

The chunk's `usage` field (`CreateChatCompletionStreamResponse.usage`, openapi.yaml:33393-33394) `$ref`s `#/components/schemas/CompletionUsage`.

**`CompletionUsage` — top-level fields and required list:**

> `openapi.yaml:31871-31886` and `31935-31938`
> ```yaml
>     CompletionUsage:
>       type: object
>       description: Usage statistics for the completion request.
>       properties:
>         completion_tokens:
>           type: integer
>           default: 0
>           description: Number of tokens in the generated completion.
>         prompt_tokens:
>           type: integer
>           default: 0
>           description: Number of tokens in the prompt.
>         total_tokens:
>           type: integer
>           default: 0
>           description: Total number of tokens used in the request (prompt + completion).
>       required:
>         - prompt_tokens
>         - completion_tokens
>         - total_tokens
> ```

**Nested detail objects (both optional — absent from the `required` list):**

> `openapi.yaml:31887-31934` (structure; prose abridged)
> ```yaml
>         completion_tokens_details:
>           type: object
>           description: Breakdown of tokens used in a completion.
>           properties:
>             accepted_prediction_tokens:   # 31891 — integer, default 0
>             audio_tokens:                 # 31897 — integer, default 0
>             reasoning_tokens:             # 31901 — integer, default 0
>             rejected_prediction_tokens:   # 31905 — integer, default 0
>         prompt_tokens_details:
>           type: object
>           description: Breakdown of tokens used in the prompt.
>           properties:
>             audio_tokens:                 # 31923 — integer, default 0
>             cached_tokens:                # 31927 — integer, default 0
>             cache_write_tokens:           # 31931 — integer, default 0
> ```

Neither nested object declares a `required` list, so every detail field (`accepted_prediction_tokens`, `audio_tokens`, `reasoning_tokens`, `rejected_prediction_tokens`; `audio_tokens`, `cached_tokens`, `cache_write_tokens`) is individually optional; all are integers with `default: 0`.

**Schema identity (one citation covers streaming and non-streaming):** the non-streaming `CreateChatCompletionResponse.usage` (33238-33239) `$ref`s the **same** `#/components/schemas/CompletionUsage`:

> `openapi.yaml:33238-33239` (inside `CreateChatCompletionResponse`, schema declared at 33139)
> ```yaml
>         usage:
>           $ref: "#/components/schemas/CompletionUsage"
> ```

The legacy `CreateCompletionResponse.usage` (33767-33768, schema at 33679) also references the same schema. `CompletionUsage` has exactly three references in the file (33239, 33394, 33768); the Assistants API uses distinct schemas (`RunCompletionUsage` at 60173, `RunStepCompletionUsage` at 60577) that are not reachable from chat completions.

**Reading:** the streaming usage object is `CompletionUsage` with required `prompt_tokens`/`completion_tokens`/`total_tokens` (integers, default 0) and two optional detail objects — `completion_tokens_details` {accepted_prediction_tokens, audio_tokens, reasoning_tokens, rejected_prediction_tokens} and `prompt_tokens_details` {audio_tokens, cached_tokens, cache_write_tokens}, all fields optional — and it is byte-for-byte the same schema the non-streaming chat response uses, so this single citation covers both wire shapes.

---

## Negative findings

1. **No block/content-part START or STOP signal in Chat Completions streaming (C3).** `content_block` → 0 hits; `block_start` → 0; `block_stop` → 0; `message_start` → 0; `message_stop` → 0; `part_added` → 0. `content_part` → 44 hits, all in Responses API event schemas/examples (first hit: line 3332; representative: `response.content_part.added` at 18015), none reachable from `CreateChatCompletionStreamResponse`. Text streams solely as flat `delta.content` strings; the adapter must mint block boundaries.
2. **No mid-stream error payload for Chat Completions (C6).** `error` inside chunk schema lines 33295-33440 → 0 hits; inside delta lines 31479-31521 → 0 hits; `ErrorEvent` is referenced only by `AssistantStreamEvent` (line 28622); the chat endpoint declares only a `200` response with a single stream schema.
3. **`[DONE]` is not a schema constant for Chat Completions (C5).** It appears in chat scope only inside the `include_usage` prose (31436-31437); the only `enum: - "[DONE]"` at this commit is the Assistants `DoneEvent` (37591). The Chat Completions `stream` parameter description (32952-32959) does not itself state the terminator.
4. **No `usage` in the per-chunk required list (C1/C4).** `usage` is absent from `required` (33424-33429) and is `nullable: true` — consumers must accept chunks with the field missing entirely as well as present-but-null.
5. **`system_fingerprint` is optional and deprecated (C1).** Declared at 33377-33378 with `deprecated: true`; not in the required list.

## Search log

All commands run over the downloaded `openapi.yaml` in the session scratchpad. Pattern → hit count (grep unless noted):

| Pattern / command | Hits |
|---|---|
| `CreateChatCompletionStreamResponse\|ChatCompletionStreamResponseDelta\|chat.completion.chunk` | 13 |
| `\[DONE\]` | 11 (lines 19129, 19299, 20384, 20546, 21386, 31437, 33625, 36107, 36387, 37591/37598, 61756) |
| `ChatCompletionStreamOptions\|stream_options` | 10+ (shown 20; schema at 31425) |
| `content_part` | 44 (all Responses API) |
| `content_block` | 0 |
| `block_start` | 0 |
| `block_stop` | 0 |
| `message_start` | 0 |
| `message_stop` | 0 |
| `content_part.added` | 21 (all Responses API) |
| `part_added` | 0 |
| `ChatCompletionStream` | 8 (only 2 schema definitions: 31425, 31479) |
| `^    ChatCompletionStream` (anchored schema names) | 2 |
| `output_text.delta\|response.completed\|response.failed\|response.error` | 10+ (all Responses API) |
| `ErrorEvent\|error.event\|ErrorResponse` | 15+ (ErrorEvent def 37862; sole stream-union ref 28622) |
| `DoneEvent` | 60+ (chat-relevant: none; Assistants DoneEvent 37580) |
| `AssistantStreamEvent` | 2 (def 28574) |
| `CreateChatCompletionRequest:` | 1 (line 32739) |
| awk `error`/`event` in lines 33295-33440 (chunk schema body) | 0 |
| awk `error` in lines 31479-31521 (delta schema body) | 0 |
| awk `^            stream:` in lines 32739-33295 (chat request) | 1 (line 32952) |
| `CompletionUsage` (C8) | 8 (schema def 31871; refs 33239, 33394, 33768; distinct `RunCompletionUsage` 60173 / `RunStepCompletionUsage` 60577) |
| `^    CompletionUsage:` (anchored schema name, C8) | 1 (line 31871) |
| awk schema-parent resolution for refs 33239 / 33768 (C8) | `CreateChatCompletionResponse` (33139) / `CreateCompletionResponse` (33679) |
