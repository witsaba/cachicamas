# Exploration: add-openrouter-first-provider

> **Change**: `add-openrouter-first-provider`
> **Milestone scope**: concretizes the AI-24 vendor/dialect decision as OpenRouter, and starts AI-25 (client construction against the chosen vendor), AI-38 (full deterministic adapter conformance), and AI-39 (opt-in live smoke). AI-26 … AI-32 (translation, decoder, lifecycle, tool calls, usage, failure mapping, reasoning) and AI-29.0 (reasoning decision) are **already shipped** for the generic OpenAI-compatible dialect — see `backend/agent/src/ai/openaicompat/` and `decision.md` of the AI-29 change.
> **Date**: 2026-08-06
> **Sources cited below**: OpenRouter docs (URLs inline), Engram memories #2432, #1997, #2235, #2055; in-repo files at `backend/agent/` (paths inline).
> **Mode**: hybrid (OpenSpec file + Engram `mem_save` upsert with topic_key `sdd/add-openrouter-first-provider/explore`).
> **Skill Resolution**: paths-injected (the orchestrator named the only path explicitly, `/Users/braejan/.claude/skills/sdd-explore/SKILL.md`).

---

## 1. Senior-architect preamble — what this change is and is not

The user pre-decided vendor = OpenAI-compatible Chat Completions streaming dialect (memory #2432). Memory #2432 named OpenRouter, vLLM, Ollama, LocalAI and similar as in-scope for that dialect but did **not** pick the first concrete vendor. The `backend/agent/src/ai/openaicompat/` package that ships today is a **generic, vendor-agnostic OpenAI-compatible adapter** built against `net/http`, with zero `go.mod` requires — it works against any URL the caller injects. There is no OpenRouter-specific code anywhere in the repo (grep confirmed: only two mentions of "OpenRouter" in `openaicompat/doc.go`, both as in-scope-vendor prose, never as code).

This change **concretizes the first vendor as OpenRouter**. Concretely it is three things, in this order:

1. **An OpenRouter-specific Provider value** — a thin wrapper around `openaicompat.Client` that hardwires the OpenRouter endpoint (`https://openrouter.ai/api/v1`), the OpenRouter attribution headers (`HTTP-Referer`, `X-Title`/`X-OpenRouter-Title`), and a documented default model identifier. This is the *first* concrete provider value the project ships. Memory #2432's "zero requires" invariant stays intact: the wrapper adds no dependencies, only injection points and constants.
2. **An OpenRouter-specific conformance bridge** — analogous to the existing `openaicompat/bridge_test.go` that emits OpenAI-compatible wire bytes from a local `httptest.Server`. This bridge is what `agenttest.RunConformanceFor` runs against (AI-38.1) and what `agenttest.RunConformance` (AI-38.1's full run) will accept as a `Factory` for the first concrete adapter. The bridge is *the* proof the openaicompat package's wire behavior matches OpenRouter's — against recorded OpenRouter-format transcripts, not against the real network.
3. **An opt-in live smoke test** — gated on `OPENROUTER_API_KEY` env var presence, with a build-tag or skip discipline, that hits the real endpoint with a minimal request to prove the header attachment, endpoint reachability and the round-trip actually work against OpenRouter's own server. This is AI-39.1's *only* sanctioned live-network test in the entire stack; it must be opt-in, silent on secret material, and bypassed clean when the env var is absent.

What this change **does not** do, by deliberate inheritance from AI-24 and the shipped openaicompat work:

- It does **not** reopen the vendor/dialect choice. Vendor is OpenRouter.
- It does **not** introduce new dependencies. `go.mod` stays empty; `allowedNonStdlibPrefixes` stays at one entry.
- It does **not** introduce a new payload type, a new `EventKind`, or a new `ai.FailureCategory`. Every shape used is already in `package ai`.
- It does **not** reopen AI-29 (reasoning stream is **struck** as a documented capability absence, per `decision.md` of `cachicamas-ai-provider-reasoning-stream`, dated 2026-08-04). The OpenRouter-specific reasoning surface — `delta.reasoning_details` and `delta.reasoning` for models that support it — is **non-blocking, model-conditional, and explicitly out of v1 scope**. Picking a default model that does not emit reasoning (e.g. `openai/gpt-4o`) preserves the absence verdict; the conformance bridge's recorded transcript is what makes that verdict checkable on every run.
- It does **not** implement `TokenCounter` (CAP-O-02) — `openaicompat` does not currently advertise it, and AI-38.2 will record it absent on the generated capability report, against AI-24 §8's expected `absent`.

---

## 2. Part A — Deep web research on OpenRouter

Every claim below carries its source URL inline. Three labels distinguish evidence quality: **OpenRouter docs** (verbatim from the published documentation or OpenAPI specification), **observed example** (taken from one of OpenRouter's own code snippets on the docs page, not independently verified against a live wire trace), and **inference** (a claim neither pinned by docs nor observed; flagged so the proposal phase can resolve it against an actual transcript before spec lands).

### 2.1 Base URL and endpoint

**Base URL**: `https://openrouter.ai/api/v1` (https://openrouter.ai/docs/api-reference/overview — "POST your request to the `/api/v1/chat/completions` endpoint"). **Path**: `/chat/completions`. **Method**: `POST`. **Content-Type**: `application/json` (https://openrouter.ai/docs/api-reference/authentication — every cURL and fetch example sets `Content-Type: application/json`; same example given in https://openrouter.ai/docs/api-reference/streaming). **Accept header on streaming requests**: the openaicompat adapter already sets `Accept: text/event-stream` in `request.go:38` — OpenRouter honors that (https://openrouter.ai/docs/api-reference/streaming — "Server-Sent Events (SSE) are supported as well, to enable streaming *for all models*").

**No divergence** from memory #2432 §2 ("HTTPS, response `Content-Type: text/event-stream`, data-only frames"). The endpoint path is one segment (`/chat/completions`), joined to the base by the openaicompat package's `joinRequestPath` machinery at `endpoint.go:15` — a base ending in `/api/v1` with the relative path `chat/completions` yields exactly `/api/v1/chat/completions`, with no doubled separator and no dropped segment (R-APC-006).

### 2.2 Authentication header

**Header name**: `Authorization`. **Format**: `Bearer <OPENROUTER_API_KEY>` where the key is the user's OpenRouter API key (https://openrouter.ai/docs/api-reference/authentication — "set the `Authorization` header to a Bearer token with your API key"). **OpenRouter explicitly notes** that its keys are more powerful than first-party vendor keys (they can carry credit limits for apps and be used in OAuth flows) — a credential-handling subtlety the openaicompat package's `credential.go` does **not** need to know about, because that package treats the bearer value as opaque (R-APC-014, credential.go doc line 5–6: "an opaque bearer-token value").

**No divergence** from openaicompat `request.go:35` — `Authorization: Bearer <token>` is exactly the wire shape the adapter already emits.

### 2.3 Optional / recommended headers

OpenRouter supports three **optional** headers for app attribution and ranking (https://openrouter.ai/docs/api-reference/overview, "Headers" section; confirmed verbatim in the OpenAPI spec at `https://openrouter.ai/openapi.yaml` under `components/parameters`):

| Header | Required? | Purpose | Spec source |
|---|---|---|---|
| `HTTP-Referer` | **Optional** | "Identifies your app on openrouter.ai", used as the primary identifier for rankings and per-app usage tracking. Spec description: "The app identifier should be your app's URL and is used as the primary identifier for rankings. This is used to track API usage per application." | OpenRouter docs overview, OpenAPI `parameters/AppIdentifier` |
| `X-OpenRouter-Title` (also `X-Title` accepted) | **Optional** | "Sets/modifies your app's title" — for app appearance on the OpenRouter dashboard. Spec description: "The app display name allows you to customize how your app appears in OpenRouter's dashboard." | OpenRouter docs overview, OpenAPI `parameters/AppDisplayName` |
| `X-OpenRouter-Categories` | **Optional** | "Comma-separated list of app categories (e.g. 'cli-agent,cloud-agent'). Used for marketplace rankings." | OpenRouter docs overview, OpenAPI `parameters/AppCategories` |

These three are **not required for the API to function** — Quickstart confirms: "In the examples below, the OpenRouter-specific headers are optional. Setting them allows your app to appear on the OpenRouter leaderboards." (https://openrouter.ai/docs/quickstart).

**Divergence from the pre-decided dialect**: OpenAI's own Chat Completions API has no analogue. This is an **additive, OpenRouter-only** concern; it is the most concrete difference between "OpenAI-compatible dialect" and "OpenRouter specifically". The proposal should attach these headers through the wrapper's configuration (injected strings), not by mutating the openaicompat package's request header set — that package has a documented rule ("`Authorization` and `Content-Type` are the only headers it sets", per `request.go`) and a test guarding against header-set widening (R-APC-001, the "every injected value is used" rule, would otherwise re-argue what an injected HTTP client can do already).

**In a CI conformance test**: leaving the attribution headers empty produces a working request; setting them to non-empty values is the only way to *observe* them on the wire (R-APC-013's stub-transport proof). The conformance bridge for OpenRouter therefore needs a fixture that includes the three headers at non-empty values and a `Factory` constructor that plumbs them through.

### 2.4 Model routing via the `model` field

**Model identifier shape**: `<vendor>/<model-slug>` (e.g. `openai/gpt-4o`, `anthropic/claude-3.5-sonnet`, `meta-llama/llama-3.2-3b-instruct`). Verified across the OpenRouter docs and the OpenAPI spec's `model` field on `ChatRequest`: the example everywhere is `openai/gpt-4o` or `anthropic/claude-haiku-4.5` (https://openrouter.ai/docs/api-reference/overview; https://openrouter.ai/docs/api-reference/streaming; OpenAPI `ChatRequest` schema).

**Routing**: OpenRouter's docs state "OpenRouter will select the least expensive and best GPUs available to serve the request, and fall back to other providers or GPUs if it receives a 5xx response code or if you are rate-limited" (https://openrouter.ai/docs/api-reference/overview, "Model routing" Info box). The caller does **not** pick the provider — OpenRouter does. This is a meaningful semantic difference from first-party OpenAI: the same model ID can land on different physical providers across calls.

**Variants** (https://openrouter.ai/docs/guides/routing/model-variants — each is a suffix to the model ID):

| Suffix | Meaning | Notes |
|---|---|---|
| `:free` | Access a free variant of the same model (no per-token cost, but subject to account-wide rate limits — 20 req/min and 50 req/day for accounts with <10 credits purchased all-time, 20 req/min and 1000 req/day otherwise) | https://openrouter.ai/docs/guides/routing/model-variants/free |
| `:nitro` | High-speed model inference | https://openrouter.ai/docs/guides/routing/model-variants/nitro |
| `:thinking` | Enable extended reasoning on models that support it | https://openrouter.ai/docs/guides/routing/model-variants/thinking (NOTE: "The `:thinking` variant is no longer supported for Anthropic models. Use the `reasoning` parameter instead." — superseded by the unified `reasoning` parameter) |
| `:online` | Real-time web search | https://openrouter.ai/docs/guides/routing/model-variants/online |
| `:extended` | Extended context window | https://openrouter.ai/docs/guides/routing/model-variants/extended |
| `:exacto` | Quality-first provider sorting | https://openrouter.ai/docs/guides/routing/model-variants/exacto |

**Aliases**: `~openai/gpt-latest` (and similar `~<vendor>/<family>-latest` aliases) — "always resolves to the newest OpenAI flagship model — so your code keeps using the freshest version without redeploying" (https://openrouter.ai/docs/quickstart). Note the **tilde prefix** — a real syntactic marker, not just a mnemonic.

**Auto routers**: `openrouter/auto`, `openrouter/free`, and others route across the catalog at request time. Not used by this adapter's first concrete model.

**Implication for the conformance smoke**: a `:free` variant model is the cheapest option for a CI smoke test (zero credit cost) but the rate-limit table above (20 req/min) means a flaky CI that retries will exhaust the budget. The wrapper should pick a single small paid model by default for the smoke (e.g. a cheap non-reasoning model) and document that the CI smoke is expected to consume ~$0.001 per run.

**Divergence from OpenAI**: none — OpenAI also accepts opaque model identifiers, just without the `/` prefix. The openaicompat adapter never inspects the model string (it passes it through verbatim), so OpenRouter's `vendor/model` and OpenAI's `model` are wire-identical to it.

### 2.5 Streaming behavior

**Confirmed** (https://openrouter.ai/docs/api-reference/streaming, "Streaming" page; OpenAPI `ChatStreamingResponse` schema):

1. **Transport**: HTTPS, response `Content-Type: text/event-stream`. Data-only frames — no `event:` lines are sent by OpenRouter itself (the docs never name an event type and the schema models only `data:`-frame dispatch).
2. **Terminal sentinel**: `data: [DONE]` frame, identical to OpenAI's. All three example Python/TypeScript fetch snippets explicitly check `if (data === '[DONE]')` and break.
3. **Keep-alive comment lines**: OpenRouter **sends SSE comments** like `: OPENROUTER PROCESSING` periodically to prevent connection timeouts (https://openrouter.ai/docs/api-reference/streaming, "Additional information"). The docs warn: "Passing a comment line like `: OPENROUTER PROCESSING` to `JSON.parse` throws, and unhandled it will crash your stream loop." The openaicompat package's `decoder.go` does **not** need new code for this — the WHATWG HTML Living Standard §9.2 treats `:`-prefixed lines as **comments** that the parser ignores (memory #2432 §2 cites this verbatim). The existing `R-ATS-017` skip rule ("a frame whose SSE event type is not the default type is skipped unconditionally") does not apply to comments (they never reach the dispatch stage at all), so the decoder's existing line-grammar handling already covers them. **No change needed.**
4. **`stream_options.include_usage` is DEPRECATED** on OpenRouter — confirmed verbatim from the OpenAPI spec at `https://openrouter.ai/openapi.yaml` (`ChatStreamOptions` schema): "Deprecated: This field has no effect. Full usage details are always included." **This is a divergence from the pre-decided dialect.** The openaicompat package currently sets `stream_options: {"include_usage": true}` at every request (`body.go`, R-ART-017); the setting is redundant but **not harmful** on OpenRouter — OpenRouter ignores it and always returns usage. The proposal must record this as a deliberate, harmless redundancy, not silently drop the field (which would force AI-26.x to widen to keep emitting it). The alternative — dropping the field on OpenRouter — would diverge the wire body from what other openaicompat-target vendors expect; the openaicompat package's posture is **one wire body for all target vendors**, so the field stays.
5. **Usage arrives on the final chunk, before `[DONE]`**, with an empty `choices` array — verbatim from https://openrouter.ai/docs/api-reference/overview: "Usage data is always returned for non-streaming. When streaming, usage is returned exactly once in the final chunk before the [DONE] message, with an empty choices array." This matches OpenAI's documented behavior (memory #2432 §1, R-CAP-R-03 cleared).

**No change to decoder logic needed**. The bridge's recorded transcript must include a terminal chunk carrying `usage` followed by `data: [DONE]` — the openaicompat package's `usage.go` (`usageFromWire`) reads it; `stream.go`'s `[DONE]` recognition triggers `buildCompletion`. Confirmed end-to-end against the openaicompat `bridge_test.go` which already passes `TestConformanceBridge_StreamingText` (line 348) — that test does NOT currently inject a usage chunk, so the conformance bridge for OpenRouter must extend it to include usage in the terminal chunk for AI-38.1 to assert `CapCompletionMetadata` end-to-end.

### 2.6 Tool calls

**Confirmed**: OpenRouter streams `delta.tool_calls[]` carrying the same shape as OpenAI's dialect — `id`, `index`, `type: "function"`, `function: { name, arguments }`. From the OpenAPI spec (`ChatStreamToolCall` schema): id is a string, index is an integer, function is an object with `name` and `arguments` (a string). The example shows:

```json
{
  "function": { "arguments": "{\"location\": \"...\"}", "name": "get_weather" },
  "id": "call_abc123",
  "index": 0,
  "type": "function"
}
```

(https://openrouter.ai/docs/guides/features/tool-calling, "Streaming with Tool Calls" section; OpenAPI `ChatStreamToolCall`).

**Parallel tool calls** are supported (`parallel_tool_calls` parameter, default `true`) and OpenRouter streams each call on its own `index`. The example streaming code (https://openrouter.ai/docs/guides/features/tool-calling, "Streaming with Tool Calls") shows accumulation: `toolCalls.push(...chunk.choices[0].delta.tool_calls)` — confirming fragmented argument bytes accumulate across chunks keyed by `index`.

**Tool result placement** (replay side): OpenRouter expects `{"role": "tool", "tool_call_id": <id>, "content": <string>}` — verbatim from the worked example at https://openrouter.ai/docs/guides/features/tool-calling, "Step 3". This is exactly the dialect-conventional `role: "tool"` shape the openaicompat package's `tool_result.go` already renders (memory #2432 §2, AI-26.5 item 1).

**No divergence** from the pre-decided dialect. OpenRouter's tool-call wire is byte-identical to OpenAI's, and the openaicompat bridge already passes `TestConformanceBridge_ToolCalls` (line 367) — the bridge renders `tool_calls` deltas from a script via `writeToolStartChunk`/`writeToolDeltaChunk` (bridge_test.go:230–242) and the conformance cases all pass. **For OpenRouter the bridge needs no new tool-call code; it needs the OpenRouter-specific envelope** (the three attribution headers) wrapped around the same byte-rendering path.

### 2.7 Reasoning — the AI-29 reopen-trigger check

This is the **highest-leverage question** for AI-29.0's striking posture (memory #2432 §7, restated as a verdict in `decision.md` of `cachicamas-ai-provider-reasoning-stream`, 2026-08-04). OpenRouter exposes **two** reasoning surfaces on the streaming delta, **conditionally** on the underlying model:

**Surface A — `delta.reasoning` (string)**: From the OpenAPI spec (`ChatStreamDelta` schema): "Reasoning content delta" with `type: string | null`. From the docs example (https://openrouter.ai/docs/guides/best-practices/reasoning-tokens, "Example: Streaming with Anthropic Reasoning Tokens"):

```python
chunk.choices[0].delta.reasoning_details  # structured reasoning
chunk.choices[0].delta.content           # final answer text
```

The `reasoning` field is **not separately listed** in the streaming-delta example, but the same docs state: "Reasoning tokens will appear in the `reasoning` field of each message" and "You can also use `reasoning_content` as an alias - it functions identically to `reasoning`." (https://openrouter.ai/docs/guides/best-practices/reasoning-tokens, "Preserving Reasoning" section).

**Surface B — `delta.reasoning_details` (array of structured objects)**: From the OpenAPI spec (`ChatStreamReasoningDetails` schema, items typed `ReasoningDetailUnion`): an array of `{type, format, id, index, [text|summary|data]}`. The `type` values are `"reasoning.text"`, `"reasoning.summary"`, `"reasoning.encrypted"`. This is **the structured, model-portable reasoning surface** — the `format` field enumerates `"anthropic-claude-v1"`, `"openai-responses-v1"`, `"google-gemini-v1"`, `"xai-responses-v1"`, `"meta-responses-v1"`, `"bedrock-openai-responses-v1"`, `"azure-openai-responses-v1"`, `"unknown"`.

**What the dialect docs say vs what the pre-decided dialect said** (memory #2432 §7): memory #2432 read the **OpenAI Chat Completions API's `ChatCompletionStreamResponseDelta`** schema (C7, pinned) and stated "exactly five properties: `content`, `function_call`, `tool_calls`, `role`, `refusal`" — no reasoning field. Memory #2432's `decision.md` for `cachicamas-ai-provider-reasoning-stream` (2026-08-04) struck AI-29.1, AI-29.2 and AI-29.3 on that basis.

**The OpenRouter divergence**: OpenRouter **adds** two reasoning surfaces (`reasoning`, `reasoning_details`) to its delta schema beyond OpenAI's C7 five-property set. These are **OpenRouter-specific extensions**, layered on top of the underlying dialect. They are **not present on every model** — they appear only when the routed-to model supports reasoning (Anthropic Claude reasoning models, DeepSeek R1, Google Gemini thinking models, OpenAI o-series via OpenRouter — note the docs' caveat: "While most models and providers make reasoning tokens available in the response, some (like the OpenAI o-series) do not" — https://openrouter.ai/docs/guides/best-practices/reasoning-tokens, "Reasoning Tokens" note).

**Implications**:

- **If the first concrete adapter targets a non-reasoning model** (e.g. `openai/gpt-4o`), the AI-29 absence verdict stands. AI-38.2's expected-vs-generated comparison will record `CAP-O-01 = absent`, matching AI-24 §8's expected `absent`. **No AI-29 reopen.**
- **If the first concrete adapter targets a reasoning model** (e.g. `anthropic/claude-3.7-sonnet:thinking` or `~anthropic/claude-sonnet-latest` with `reasoning: {max_tokens: 2000}`), the wire will carry a `reasoning_details` array and possibly a `reasoning` string on each delta. The openaicompat package's `decodeChunk` (chunk.go:213) uses `json.Unmarshal` without `DisallowUnknownFields`, so unknown fields are silently dropped — including reasoning — and the conformance cases already cover this with the `TestReasoningExtensionField_DroppedNotLeakedNotFailed` test (reasoning_absence_test.go:149). **The reasoning content reaches the byte-buffer, is dropped at decode time, and never leaks into a text event.** This is exactly the "drop, not leak, not fail" posture the struck AI-29 verdict documents.

**Recommendation for this change**: pick a **non-reasoning default model** (e.g. `openai/gpt-4o` — the OpenRouter docs' canonical streaming example model, https://openrouter.ai/docs/api-reference/streaming). This preserves the AI-29 absence verdict and keeps the conformance suite's `CapReasoningContent` cases SKIP-record-absent (R-CNF-004, per `applyDeclaredAbsences` at agenttest/conformance_suite.go:330). The smoke test can exercise reasoning by adding a second recorded transcript **in a later change** — that change would reopen AI-29 under trigger #1 of the AI-29 decision (`decision.md` §9 trigger #1: "The backend selected for AI-38 / AI-39 is documented to carry a reasoning-bearing field on its streamed delta"), a deliberately deferred decision, **not** part of this change.

**The reasoning extension field is not a divergence the openaicompat package must handle specially** — the `decodeChunk` JSON decoder's "ignore unknown fields" default is the right posture and is already tested (reasoning_absence_test.go:149 covers a `reasoning_content` extension; OpenRouter uses the same shape, just with `reasoning` / `reasoning_details` instead of `reasoning_content` — the test would need one extra fixture for the renamed field, but the mechanism is the same).

### 2.8 Error envelope

**Pre-stream error** (HTTP non-2xx before any bytes are streamed) — from https://openrouter.ai/docs/api-reference/errors-and-debugging, "Errors" section, TypeScript example:

```typescript
type ErrorResponse = {
  error: {
    code: number;       // mirrors HTTP status code
    message: string;
    metadata?: Record<string, unknown>;
  };
};
```

The HTTP response status matches `error.code`. Common statuses: 400 (Bad Request, invalid parameters), 401 (Unauthorized, invalid API key), 402 (Payment Required, insufficient credits), 408 (Request Timeout), 429 (Too Many Requests, rate limited), 502 (Bad Gateway, provider error), 503 (Service Unavailable, no providers). **OpenRouter does not use `X-RateLimit-*` headers** in the docs — it uses the standard HTTP `Retry-After` header on 429 and 503 (https://openrouter.ai/docs/api-reference/errors-and-debugging, "Retry-After header" section). The retry-after value is seconds, per RFC 7231.

**Mid-stream error** (HTTP 200 already sent, then an error occurs after some bytes are streamed) — from https://openrouter.ai/docs/api-reference/streaming, "Mid-stream errors" section:

```typescript
type MidStreamError = {
  id: string;
  object: 'chat.completion.chunk';
  created: number;
  model: string;
  provider: string;
  error: {
    code: number;       // HTTP status code (e.g. 400, 429, 502)
    message: string;
    metadata?: {
      error_type: string;     // Typed error code — see table below
      provider_code?: string; // Original upstream error code (omitted on 500s)
    };
  };
  choices: [{
    index: 0;
    delta: { content: '' };
    finish_reason: 'error';
    native_finish_reason?: string;
  }];
};
```

Key feature: `error_type` is a stable string across all OpenRouter APIs ("Chat Completions", "Anthropic Messages", "Responses"), with a closed vocabulary documented in https://openrouter.ai/docs/api-reference/errors-and-debugging, "Typed Error Codes" section:

| Category | `error_type` values | HTTP status |
|---|---|---|
| Token & length | `context_length_exceeded`, `max_tokens_exceeded`, `token_limit_exceeded`, `string_too_long` | 400 |
| Auth | `authentication`, `permission_denied`, `payment_required` | 401/402/403 |
| Rate limit | `rate_limit_exceeded`, `provider_overloaded`, `provider_unavailable` | 429/502/503 |
| Validation | `invalid_request`, `invalid_prompt`, `not_found`, `precondition_failed`, `payload_too_large`, `unprocessable` | 400/404/412/413/422 |
| Content policy | `content_policy_violation`, `refusal` | 400 |
| Generic | `server`, `timeout`, `unmapped` | 500/504/500 |

**Divergence from OpenAI** (the openaicompat package's `failure_map.go` and `errors.go` are pinned to OpenAI's narrower vocabulary): OpenAI's documented error envelope is `{ "error": { "message": "...", "type": "...", "param": "...", "code": "..." } }` — `type` is the OpenAI discriminator (e.g. `"invalid_request_error"`). OpenRouter's discriminator is `error.metadata.error_type` and the vocabulary is broader.

**The openaicompat package's existing failure mapping** (failure_map.go, `mapResponse` for non-2xx + `failureFromErrorFrame` for in-band SSE errors) was built against OpenAI's narrower shape. The AI-32 milestone landed before OpenRouter-specific error envelopes were surveyed. **This is a known scope gap that this change does NOT have to close** — the conformance bridge's recorded transcript can emit OpenAI-shape errors for the failure cases (the bridge is recording *what the adapter should do*, not what OpenRouter will do in production) and AI-38.1 will assert the adapter maps them correctly. The OpenRouter-specific `error_type` surface is **out of v1 scope** and is recorded here as a known gap to revisit when AI-36 (secret redaction) and AI-32's failure taxonomy widen.

### 2.9 Cost / usage accounting

**OpenRouter adds fields beyond the OpenAI dialect's `usage`** — verbatim from the OpenAPI `ChatUsage` schema:

```typescript
type ResponseUsage = {
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  prompt_tokens_details?: {
    cached_tokens: number;
    cache_write_tokens?: number;
    audio_tokens?: number;
    video_tokens?: number;
  };
  completion_tokens_details?: {
    reasoning_tokens?: number;
    audio_tokens?: number;
    image_tokens?: number;
  };
  cost?: number;                                    // OpenRouter-specific
  is_byok?: boolean;                                // OpenRouter-specific
  cost_details?: {                                  // OpenRouter-specific
    upstream_inference_cost?: number;
    upstream_inference_prompt_cost: number;
    upstream_inference_completions_cost: number;
  };
  server_tool_use?: { web_search_requests?: number };
};
```

(Source: https://openrouter.ai/docs/api-reference/overview "Responses" section; OpenAPI `ChatUsage` schema.)

**Divergence from pre-decided dialect**: `cost`, `is_byok`, `cost_details` are OpenRouter-only. The openaicompat package's `usage.go` (`usageFromWire`) reads only the OpenAI-shape fields (prompt/completion/total tokens + the optional details blocks) — it does not parse `cost`. This is **correct**: Layer 1's neutral usage shape does not include cost (it's a vendor-specific accounting concern, not a Layer 1 contract), and AI-32.4 already records that "the adapter does not reproduce wire-side cost data in the neutral usage payload". The `cost` field is silently dropped at the JSON-unmarshal boundary, which is the same posture taken for the `reasoning_content` extension field — drop-not-leak.

**`reasoning_tokens` in `completion_tokens_details`** is **honored by openaicompat** — it is read and stored under the usage payload, per the existing tests at `openaicompat/usage_test.go:54` (`TestUsage_OnlyPromptAndCompletionTokens_UnmappedFieldsReadAbsent`). Wait: the test name says "UnmappedFieldsReadAbsent". Let me re-read carefully.

> From `decision.md` of `cachicamas-ai-provider-reasoning-stream` §5 row 4: "Unmapped reasoning token count — `usageFromWire` never reads the `reasoning_tokens` field; the `Reasoning` field on the neutral usage type stays absent whenever the wire reports a count".

**Corrected reading**: the openaicompat package **does NOT map `reasoning_tokens` from the wire into the neutral usage payload.** The reasoning-absence decision explicitly records this: `reasoning_tokens` is a count, not a content block (memory #2432 §7: "It is a **count**, not a block. It carries no text, no signature, and nothing a `%#v`-style round trip could replay"). AI-23.8 records the capability outcome as `absent` (decision.md §8). **No change to this posture for OpenRouter** — OpenRouter surfaces `reasoning_tokens` for reasoning models only; the openaicompat adapter ignores it on the read side and the absence verdict stands.

### 2.10 Free-tier / rate-limit gotchas

(https://openrouter.ai/docs/guides/routing/model-variants/free, "Rate limits" section)

- Free-variant models are **account-wide**, not per-model — pinning one `:free` model does not raise the limit.
- Accounts with **< 10 credits purchased all-time**: 20 requests/minute, 50 requests/day.
- Accounts with **≥ 10 credits purchased all-time**: 20 requests/minute, 1000 requests/day.
- The threshold is **lifetime purchased**, not current balance — a one-time $10 top-up keeps the higher daily limit forever.
- `:free` models "may have different availability than paid versions — free capacity is contributed by upstream providers and can change or be temporarily unavailable".

**CI conformance impact**: A CI smoke that retries on transient failures could exhaust the 20-req/min budget quickly. The smoke's `t.Skip` discipline (skip if `OPENROUTER_API_KEY` absent, run once if present) plus a hard retry cap (e.g. max 3 retries with exponential backoff) is the minimum hygiene. **A flaky CI that re-tries an OpenRouter `:free` smoke test is the most likely place this change will flake in CI** — the proposal phase should specify the retry policy.

**Real (non-`:free`) models are not subject to the free-tier cap** but are billed per-token. A $0.001 smoke cost per CI run is acceptable but should be explicit in the proposal.

---

## 3. Part B — Layer 1 implementation surface in-repo

### 3.1 `backend/agent/` package layout (current state)

The `backend/agent/` module (module path `github.com/cachicamas/backend/agent`, Go 1.26.3) is the **layered AI module** — Layer 1 (`src/ai`) ← Layer 2 (`src/agent`, not yet built) ← Layer 3 (`src/coding`, not yet built) — per `openspec/AGENTS.md` rule 2 ("The agent module is the exception and is NOT hexagonal ... governed by ADR 0005 § D1"). The current state:

```
backend/agent/
├── Makefile          (130 lines, derived from backend/database_administrator/Makefile; test runner = `make test` = `go test -race -v ./...`)
├── README.md
├── go.mod            (3 lines: module + `go 1.26.3`, **zero `require` lines** — confirmed verbatim, memory #2432 §1)
└── src/
    ├── ai/           (Layer 1 — all shipped)
    │   ├── provider.go              (interface `ModelProvider`, optional `TokenCounter`)
    │   ├── event.go, event_registry_test.go, event_test.go, ...
    │   ├── request.go, ...
    │   ├── tool_call.go, tool_result.go, ...
    │   ├── openaicompat/            (the GENERIC OpenAI-compatible adapter — already substantial)
    │   │   ├── client.go            (AI-25)
    │   │   ├── credential.go        (AI-25)
    │   │   ├── endpoint.go          (AI-25)
    │   │   ├── request.go           (AI-25, R-ATS-001)
    │   │   ├── stream.go            (AI-28.1.1, R-ATS-001…006)
    │   │   ├── stream_state.go      (AI-28.1.2)
    │   │   ├── decoder.go           (AI-27)
    │   │   ├── chunk.go             (AI-27)
    │   │   ├── frame.go             (AI-27)
    │   │   ├── translation.go       (AI-26)
    │   │   ├── body.go              (AI-26)
    │   │   ├── message.go           (AI-26.3/5)
    │   │   ├── tool.go              (AI-26.4)
    │   │   ├── system.go            (AI-26.2)
    │   │   ├── option.go            (AI-26.7, defines `Namespace = "openaicompat"`)
    │   │   ├── usage.go             (AI-31)
    │   │   ├── finish_reason.go     (AI-31)
    │   │   ├── failure_map.go       (AI-32.1)
    │   │   ├── errors.go            (AI-27 sentinels + AI-19 sentinels)
    │   │   ├── policy.go            (AI-26.6 reasoning refusal + AI-26.8.2)
    │   │   ├── bridge_test.go       (AI-23.2 + AI-30.1 — local httptest server + factory declaration)
    │   │   ├── doc.go               (692 lines — every decision and every citation, R-ART-001)
    │   │   └── …~50 other files
    └── agenttest/                    (Layer 1 testing infrastructure)
        ├── conformance_suite.go     (the AI-38.1 runner)
        ├── conformance_*.go         (the case tables)
        ├── fake_provider.go         (AI-21, scripted subject)
        ├── stream_kit_*.go          (AI-22 helpers)
        └── …
```

### 3.2 AI-00.3 forward guard state

`backend/agent/src/ai/import_boundary_test.go` (336 lines). The relevant state for this change:

- `modulePath = "github.com/cachicamas/backend/agent"` (line 45).
- `allowedNonStdlibPrefixes = []string{modulePath}` (line 103–105) — **exactly one entry today**.
- The package-level comment (line 96–102) states: "It holds exactly one entry today. One milestone may add a second, and it carries its own gate: AI-37 adds the OpenTelemetry API — and ONLY the API paths enumerated in ADR 0005 § D3, which pre-authorises them and nothing else. Anything else reaching this list should be challenged in review, not merged."
- `TestLayer1_ModuleHasNoDependencies_ZeroRequires` (line 147) — pins that `go.mod` carries zero `require` lines today.

**This change does NOT modify `allowedNonStdlibPrefixes`** — the OpenRouter-specific code path is built on stdlib (`net/http`, `net/url`, `encoding/json`, `bytes`, `context`, `io`, `mime`, `time`) plus `github.com/cachicamas/backend/agent/src/ai` and `github.com/cachicamas/backend/agent/src/agenttest`. No new require lines. The guard stays green.

**A stale comment fix is owed** (the same as the AI-25 milestone's `NFR-APC-C` work): the import-boundary test header text at line 11–13 says "the transport choice at AI-24 a visible, ADR-gated event rather than a quiet `go get`" — AI-24 made the transport choice and it added zero dependencies, so the text reads slightly outdated. The proposal should either confirm the existing text is still accurate enough or propose a clarifying edit.

### 3.3 AI-16's provider interface — the contract this change must satisfy

From `backend/agent/src/ai/provider.go` (134 lines), the public surface is:

```go
type ModelProvider interface {
    Stream(ctx context.Context, req Request) (<-chan Event, error)
}

type TokenCounter interface {
    CountTokens(ctx context.Context, req Request) (TokenCount, error)
}
```

(R-AMP-001 / V-PRV-01 / V-PRV-03; the `ModelProvider` interface is declared in `provider.go:96`. The optional `TokenCounter` is declared in `provider.go:130`.)

The provider value's stream is:
- `Request` — a normalized Layer 1 type, fully validated before any I/O (AI-10, AI-12).
- `<-chan Event` — a receive-only channel of `Event` values, sealed against any vendor type (AI-11; the `Event.Payload` is a sealed interface implemented only in `package ai`).
- The signature guard at `agenttest/provider_signature_guard_test.go` enforces the no-vendor-type boundary at AST level.

**Construction surface for the openaicompat package** (from `openaicompat/client.go:43`):

```go
type Config struct {
    Endpoint   string         // required, absolute http(s) URL
    Credential Credential     // required, opaque bearer token
    HTTPClient *http.Client   // optional; nil ⇒ adapter builds its own
}
```

This change's OpenRouter wrapper instantiates this `Config` with `Endpoint = "https://openrouter.ai/api/v1"`, an injected `Credential`, and an **optional injected `HTTPClient`** (so the live smoke test can inject a client with a sensible per-attempt timeout). It does NOT widen the openaicompat package's public surface — it composes it.

### 3.4 AI-25 milestone — nodes and what this change implements

From `openspec/changes/archive/2026-08-05-cachicamas-ai-provider-client/` (merged PR per the archive listing) and `openspec/specs/ai-provider-client/spec.md` (lines 56–248), AI-25 has three nodes:

| Node | Status | What this change does |
|---|---|---|
| AI-25.1 Injected construction `[leaf]` | **Shipped (2026-08-05)** | N/A — this change consumes `openaicompat.New(Config{...})` rather than re-implementing |
| AI-25.2 No ambient authority `[guard]` | **Shipped (2026-08-05)** | The new OpenRouter wrapper MUST be in `package openrouter` (a new sub-package under `backend/agent/src/ai/openaicompat/openrouter/`) so the AI-25.2 call-site scan can re-run against it OR MUST be in the existing `openaicompat` package itself if it adds no `os`/`os/exec`/etc. call sites. **Recommendation: put the wrapper in a new sub-package** to keep the guard's scope clean; the wrapper only needs `net/http` and the openaicompat package. |
| AI-25.3 Test-server viability `[leaf]` | **Shipped (2026-08-05)** | This change extends with the OpenRouter-specific envelope (the three attribution headers) on the same stub-transport mechanism. |

**Concretely, what AI-25 of THIS change delivers**: a new package `backend/agent/src/ai/openaicompat/openrouter/` (or analogous location) with:

- A `Provider` value or constructor that wires `openaicompat.Client` with `Endpoint = "https://openrouter.ai/api/v1"` and an injected `Credential`, plus an injected struct for the three attribution headers (`HTTPReferer`, `XTitle`, `XOpenRouterCategories` — empty strings mean "don't send").
- A small test suite that drives a local `httptest.Server` and asserts the three headers are observed on the wire when non-empty (R-APC-001's "every injected value is used" rule, applied to the new injection points).
- A `Factory` for `agenttest.RunConformanceFor` that scripts OpenRouter-format SSE transcripts (text, tool calls, completion-with-usage, errors) and declares the three optional capabilities correctly (Reasoning=false, TokenCounting=false, CacheBoundary=false — matching AI-29's absence verdict and AI-24 §8's expectations).

### 3.5 AI-26…AI-32 + AI-38/AI-39/AI-40 — milestones enabled by this change

The following table is **scoped to what this change ENABLES, what it implements, and what it leaves for later slices** — read each row's "Change implements?" column as the deliverable scope.

| Milestone | Already shipped (this is *context*, not new work) | This change | Later slices (out of scope) |
|---|---|---|---|
| AI-26 Request translation | Yes — `openaicompat/translation.go`, `body.go`, `message.go`, `system.go`, `tool.go`, `option.go`, `policy.go` (692-line doc.go records every citation, R-ART-001). | None — wrapper just composes. | — |
| AI-27 Streaming decoder | Yes — `decoder.go`, `chunk.go`, `frame.go`, plus 7 test files (`field_grammar_test.go`, `keepalive_test.go`, `tolerance_test.go`, etc.). | None — wrapper's stream uses the existing decoder. | — |
| AI-28 Response lifecycle + text stream | Yes — `stream.go`, `stream_state.go`, `text_events.go`, plus bridge_test.go's `TestConformanceBridge_StreamingText` already passes. | None — wrapper's stream composes. | — |
| AI-29 Reasoning stream | **Struck 2026-08-04** — `decision.md` of `cachicamas-ai-provider-reasoning-stream` records the absence verdict; AI-29.1, AI-29.2, AI-29.3 are struck legibly. `reasoning_absence_test.go` covers the drop-not-leak posture. | None — wrapper picks a non-reasoning default model to preserve the verdict. | A future change could reopen AI-29 under trigger #1 of the struck decision by adding a second `Factory` variant that targets a reasoning model — deliberately deferred. |
| AI-30 Tool-call stream | Yes — `tool_call_event.go`, `tool_stream_*.go` tests, `bridge_test.go`'s `TestConformanceBridge_ToolCalls` passes. | None — wrapper composes. | — |
| AI-31 Usage / finish reasons | Yes — `usage.go`, `finish_reason.go`, plus `usage_test.go`, `usage_completion_test.go`, `usage_position_test.go`. | None — wrapper composes. | — |
| AI-32 HTTP/provider failure mapping | Yes — `failure_map.go`, `errors.go`, plus `retry_metadata.go`. **Known gap**: `error_type` (OpenRouter's typed discriminator) is unmapped — see §2.8 above. | None — wrapper composes the existing failure mapping. | AI-32 widening to cover OpenRouter's `error_type` vocabulary is **out of scope for this change** and would be a follow-up change after AI-38.2's capability record settles. |
| AI-33 Cancellation / goroutine cleanup | Yes — `stream_state.go`, `stream_kit_leak.go`. | None — wrapper composes. | — |
| AI-34 Backpressure / buffer behavior | Yes — `openaicompat` uses the AI-01-locked `DefaultBufferSize = 16` (memory #2235 §3). | None. | — |
| AI-35 Retry / idempotency policy | Yes — `retry_metadata.go`. | None. | — |
| AI-36 Secret redaction | Yes — credential's `String()`/`GoString()` redacting; `credential_scan_test.go` enforces the sentinel pattern. | The OpenRouter wrapper's own `Credential` reuses the existing `openaicompat.Credential` (zero new credential material). | — |
| AI-37 Observability boundary | Out of scope — OTel API is gated on its own ADR (import_boundary_test.go:96–102). | None. | Not in this change. |
| AI-41 Wave-2 carryovers | Yes — `cachicamas-ai-conformance-lifecycle-amendment` and `cachicamas-ai-conformance-tool-amendment` (archived 2026-08-05). | None. | — |
| **AI-38 Full deterministic adapter conformance** | **THIS CHANGE'S PRIMARY DELIVERABLE** | The OpenRouter-specific `Factory` + recorded transcripts + `TestOpenRouterAdapter_ConformanceFor*` driver functions. The full `RunConformance` (AI-38.1) runs against the new factory and emits the capability record that AI-38.2 asserts against AI-24 §8's expected outcomes. | — |
| **AI-39 Opt-in live smoke test** | **THIS CHANGE'S SECOND DELIVERABLE** | A `TestOpenRouterAdapter_LiveSmoke` test that is `t.Skip`-gated on `OPENROUTER_API_KEY` env var absence, makes one minimal `Stream` call against the real endpoint, and asserts the response parses to a valid sequence including a `Completion` event. | Hardening (retry policy, rate-limit handling) lives in a later AI-39.x change. |
| **AI-40 Publish the Layer 2 readiness contract** | NOT in this change | None. | AI-40 publishes the v1 capability matrix and the AI-39-inherited publication duty — out of scope for this concrete-vendor change. |
| AI-23.8 Capability outcome recording | Yes (`applyDeclaredAbsences` at conformance_suite.go:330) | The wrapper's `Factory` declares `Reasoning: false`, `TokenCounting: false`, `CacheBoundary: false`, so AI-38.2's generated record carries `CAP-O-01 = absent`, `CAP-O-02 = absent`, `CAP-O-03 = absent` — matching AI-24 §8's expected `absent` for all three. | — |

### 3.6 Test scaffolding — what's available

The full testkit is at `backend/agent/src/agenttest/`:

- `conformance_suite.go` — `RunConformance(t, factory)` runs every registered case against the supplied `Factory`; `RunConformanceFor(t, factory, cap)` runs only one capability's cases. **This is what AI-38.1 consumes.**
- `conformance_lifecycle.go` — `CapCancellation` cases (capability-scoped).
- `conformance_text.go` — `CapStreamingText` cases.
- `conformance_tool_call.go` — `CapToolCalls` cases.
- `conformance_capabilities.go` — the capability-record generation (AI-38.2).
- `conformance_terminal.go` — terminal-event invariants.
- `conformance_scoped.go` — scoped-run helpers.
- `conformance_redaction.go` — the credential-sentinel sweep.
- `stream_kit_*.go` — drain helpers, recorders, leak detection, ordering helpers, differ.
- `fake_provider.go` + `fake_script.go` — AI-21's scripted subject (not what this change uses; the bridge factory is what runs against AI-38.1).

**What a "smoke test of a real provider" looks like**: a `t.Skip`-gated test that:

```go
func TestOpenRouterAdapter_LiveSmoke(t *testing.T) {
    key := os.Getenv("OPENROUTER_API_KEY")
    if key == "" {
        t.Skip("OPENROUTER_API_KEY not set; live smoke is opt-in")
    }
    provider := openrouter.NewProvider(openrouter.Config{
        Credential: openaicompat.NewCredential(key),
        // Endpoint default = https://openrouter.ai/api/v1
        HTTPReferer: "<from config or env>",
        XTitle:      "cachicamas-ai-agent",
    })
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    req := mustValidRequest(t) // one user message, one text part
    ch, err := provider.Stream(ctx, req)
    if err != nil { t.Fatalf("Stream: %v", err) }
    events := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout).Events()
    if !hasValidStream(events) { t.Fatalf("events %v", events) }
}
```

This is the AI-39.1 deliverable in full — one minimal smoke run, gated, silent on secret material (the credential's `String()` is `<redacted>` by R-APC-014, so `t.Logf("%v", provider)` does not leak).

---

## 4. Part C — OpenRouter vs the pre-decided dialect delta

This is the table the proposal phase must walk before locking the spec. Every row cites the source claim. **Divergence rows are flagged for risk** — they are not blockers, but each requires a deliberate decision (record, ignore, or handle) and that decision must be captured in the proposal.

| Pre-decided fact (memory #2432 §1, §2, §7) | OpenRouter match | Source for OpenRouter | Risk |
|---|---|---|---|
| Data-only SSE frames | **Match** | OpenAPI `ChatStreamingResponse` schema — no `event:` field modeled | None |
| `[DONE]` terminal sentinel | **Match** | OpenRouter streaming docs every example checks `if (data === '[DONE]')` | None |
| `stream_options.include_usage` honored (so usage arrives on final chunk) | **DIVERGE** (deprecation) | OpenAPI `ChatStreamOptions` schema: "Deprecated: This field has no effect. Full usage details are always included." | **Low** — openaicompat sets the field; OpenRouter ignores it; usage still arrives. **Decision: keep setting the field** (one wire body for all openaicompat-target vendors; not OpenRouter-specific). |
| Tool-call streaming shape (id, index, function.name, function.arguments fragments) | **Match** | OpenAPI `ChatStreamToolCall` schema, all three example streaming docs | None |
| Vendor assigns tool-call identifiers (no minting needed) | **Match** | OpenAPI `ChatStreamToolCall.id` is a string, present in examples | None |
| Tool result as `role: "tool"` message | **Match** | https://openrouter.ai/docs/guides/features/tool-calling "Step 3" example | None |
| Reasoning absent from stream (count only, in `usage.completion_tokens_details.reasoning_tokens`) | **CONDITIONAL** | OpenRouter adds `delta.reasoning_details` (array) and `delta.reasoning` (string) on reasoning-capable models. For non-reasoning models (e.g. `openai/gpt-4o`), absent. | **Medium** — see §2.7. **Decision: target a non-reasoning default model; preserve AI-29 absence verdict; record that a future change could reopen trigger #1.** |
| `usage.completion_tokens_details.reasoning_tokens` count present | **Match** (for reasoning models) / **Absent** (for non-reasoning models) | OpenAPI `ChatUsage.completion_tokens_details.reasoning_tokens` | None — openaicompat already does not read this field (decision.md §5 row 4). |
| Automatic caching (no per-request breakpoint marker) | **Match** (still automatic on most routed models) | OpenRouter inherits caching semantics from underlying providers | None |
| No mandatory output-token limit | **Match** | OpenRouter docs "Parameters" page: "max_tokens — Optional, integer, 1 or above" | None |
| Three attribution headers `HTTP-Referer`, `X-Title`/`X-OpenRouter-Title`, `X-OpenRouter-Categories` | **DIVERGE** (additive, OpenRouter-only) | https://openrouter.ai/docs/api-reference/overview, "Headers" section; OpenAPI `parameters/AppIdentifier`, `parameters/AppDisplayName`, `parameters/AppCategories` | **Low** — additive only; openaicompat does NOT set these (so it's a wrapper-level concern, not a package-level concern). **Decision: wrapper-level injection; tested via R-APC-001's stub-transport mechanism.** |
| Bearer auth via `Authorization` header | **Match** | https://openrouter.ai/docs/api-reference/authentication | None |
| `model` field carries opaque identifier | **Match** (with `vendor/model` shape, OpenAI-shape-agnostic to the adapter) | https://openrouter.ai/docs/api-reference/overview | None |
| Mid-stream error envelope: SSE data frame with top-level `error` field, HTTP status already 200 | **Match** (shape matches the openaicompat `failureFromErrorFrame` precondition) | https://openrouter.ai/docs/api-reference/streaming, "Mid-stream errors" section | None |
| Pre-stream error envelope: HTTP status + JSON `{ "error": { "code", "message", "metadata": {...} } }` | **PARTIAL** — OpenAI's `error.code` is a string; OpenRouter's `error.code` is the **HTTP status as an integer** (https://openrouter.ai/docs/api-reference/errors-and-debugging). The openaicompat `mapResponse` reads the HTTP status directly, not `error.code`, so this is harmless. | Same | None |
| Rate-limit headers (`Retry-After`, no `X-RateLimit-*`) | **DIVERGE** (additive, no X-RateLimit-*) | https://openrouter.ai/docs/api-reference/errors-and-debugging, "Retry-After header" | **None** — openaicompat does not read rate-limit headers (rate-limit handling is AI-35's concern, deferred). |
| `error_type` typed discriminator on the metadata envelope | **DIVERGE** (additive, OpenRouter-only) | https://openrouter.ai/docs/api-reference/errors-and-debugging, "Typed Error Codes" section | **Low for this change** — known scope gap, recorded as out-of-scope (see §2.8). AI-32.4 widening to map `error_type` is a future change. |
| `usage.cost` / `usage.cost_details` cost reporting | **DIVERGE** (additive, OpenRouter-only) | OpenAPI `ChatUsage.cost`, `ChatUsage.cost_details` | **None** — openaicompat silently drops unknown usage fields (the same posture as reasoning extensions). |

**Summary**: 3 divergences, all Low or Medium risk, none of them block the change. The Medium-risk one (reasoning presence on certain models) is deliberately defused by the model choice.

---

## 5. Part D — Risk and size forecast

### 5.1 Changed-lines forecast

**This change's actual deliverable** is two things: (a) a new OpenRouter wrapper package, and (b) the AI-38.1 conformance bridge for OpenRouter + an AI-39.1 live smoke test. Neither re-implements shipped work.

| Component | Naive forecast | Corrected (×2–4, repo multiplier per memory #2432 historical pattern: AI-21 1.95×, AI-16 3.7×) |
|---|---|---|
| New package `backend/agent/src/ai/openaicompat/openrouter/provider.go` (Config, Provider, NewProvider, constructor wiring, header-attachment logic) | ~150–200 lines | ~450–1200 |
| Test for header attachment (R-APC-001-style stub-transport probe, three headers at non-empty, three headers empty) | ~80–150 lines | ~200–600 |
| Conformance bridge (`bridge_test.go` analog for OpenRouter — transcripts for text, tool calls, completion-with-usage, error cases) | ~200–350 lines | ~600–1400 |
| `TestOpenRouterAdapter_RunConformanceFor*` drivers (one per capability, AI-38.1) | ~80–150 lines | ~200–600 |
| AI-39.1 live smoke (`TestOpenRouterAdapter_LiveSmoke` + skip-discipline helper) | ~60–100 lines | ~150–400 |
| Optional: a stale-comment fix in `import_boundary_test.go` (line 96–102 wording) | ~5 lines | ~20 |
| Optional: a doc.go amendment recording the "first concrete vendor = OpenRouter" | ~30 lines | ~80 |
| **Forecast total (realistic midpoint)** | **~600–1000 lines** | **~1700–4300 lines** |
| **Realistic midpoint with repo multiplier** | — | **~2200 lines** |

The AI-25 historical multiplier (3.7× for a similar-shaped "build the first concrete thing") applies because the wrapper is the first concrete thing on top of an already-shipped generic package — the multiplier tends to run higher when the work is "make the abstract concrete", because the abstract package's tests grow in lockstep.

### 5.2 Is the 800-line review budget enough?

**No.** With a realistic midpoint of ~2200 changed lines, **the 800-line budget is insufficient for `single-pr` delivery.** The 400-line default budget is also insufficient.

The 800-line budget **could** be sufficient if the change is *narrowly* scoped to:

- Just the OpenRouter wrapper package (~600–900 lines corrected) — fits a chained PR.
- Just the conformance bridge + the conformance drivers (~800–2000 lines corrected) — does NOT fit a single 800-line PR.

**Recommendation to the orchestrator**: revisit `delivery_strategy: single-pr` at the proposal phase. The natural PR split is:

1. **PR #1 — OpenRouter wrapper package** (~700–1000 lines, single PR, no chained review burden): the `provider.go`, the header-attachment test, and the stale-comment fix. This is the **AI-25.1 + AI-25.3** deliverable for OpenRouter.
2. **PR #2 — OpenRouter conformance bridge + AI-38.1 conformance driver** (~1000–1500 lines, single PR but reviewer should expect ~800 lines of test-only fixture bytes — goldens excluded from authored risk per `sdd-phase-common.md` §E): the bridge transcripts, the capability-scoped drivers. This is the **AI-38** deliverable for OpenRouter.
3. **PR #3 — OpenRouter live smoke test** (~200–400 lines, single PR, opt-in only — never runs unless `OPENROUTER_API_KEY` is set, so default CI ignores it): the smoke test, the env-gating helper. This is the **AI-39** deliverable.

**Three chained PRs is appropriate** because the three are independently deliverable, independently reviewable, and each has a clear rollback boundary (revert the wrapper PR ⇒ the conformance PR's tests fail because the wrapper doesn't exist; revert the conformance PR ⇒ the smoke PR still works against `openaicompat` directly). The "first adapter" milestone AI-24 envisioned has three natural work units; this change preserves them.

### 5.3 Blocking questions for the proposal phase

Six questions must be answered by the proposal before spec can be written:

1. **Which model does the conformance smoke target?** The recommendation is a **non-reasoning** model — `openai/gpt-4o` is OpenRouter's canonical streaming example (https://openrouter.ai/docs/api-reference/streaming), and it preserves the AI-29 absence verdict. Alternatives: a paid cheap model (e.g. `meta-llama/llama-3.1-8b-instruct`); a `:free` model (rate-limit risk, see §2.10). **Open question: which one and why.**

2. **Should the live smoke test default to `:free` or paid?** `:free` is cheaper but rate-limit-flaky in CI; paid (e.g. ~$0.001 per run) is more reliable. **Open question: cost vs reliability trade-off.** The recommendation is paid, with a documented "approximately $0.001 per CI run" in the proposal's cost section.

3. **Does the AI-38.1 conformance run emit a generated capability record that matches AI-24 §8's expected `absent` for all three optional capabilities?** This is what AI-38.2 will assert. The wrapper's `Factory` MUST declare `Reasoning: false, TokenCounting: false, CacheBoundary: false` (non-nil pointers), per R-CNF-004's "explicit non-nil pointer, distinguishable from omission" rule. **Open question: confirm the openaicompat bridge's existing factory declaration pattern (`conformanceBridgeFactory` at `openaicompat/bridge_test.go:46–73`) is reused as the template, with only the OpenRouter-specific wire envelope added.**

4. **Should the `stream_options.include_usage` field be dropped on OpenRouter?** **No** — keep it set (the openaicompat package's wire body is one body for all target vendors). OpenRouter's deprecation is harmless and the field is required by other shared-dialect servers (OpenAI, vLLM, llama.cpp). **This is a deliberate, recorded decision; the proposal must record it.**

5. **Does AI-25.2's call-site scan need to re-run against the new package?** Yes — the new `openaicompat/openrouter/` package is in scope for the scan. The package imports only stdlib + the existing `openaicompat` package + `package ai`, so the scan stays green. **The proposal must state this explicitly so the apply phase knows to add the new package to the scan's coverage list.**

6. **Is the import-boundary test's `allowedNonStdlibPrefixes` allowlist still one entry?** Yes — confirmed by reading the file verbatim. **No allowlist edit is needed; no ADR is needed.**

### 5.4 Risks summary

- **Medium**: AI-29 reopen-trigger #1 if a reasoning-capable default model is picked. Defused by choosing a non-reasoning default (§2.7). Re-record in the proposal.
- **Low**: The three OpenRouter-specific divergences (`stream_options.include_usage` deprecation, attribution headers, `error_type` discriminator) all have deliberate decisions recorded (§4 table).
- **Low**: Free-tier rate-limit flakiness in CI (§2.10). Defused by picking a paid default model and documenting the cost.
- **Low**: The AI-00.3 stale-comment text at `import_boundary_test.go:96–102` is slightly outdated post-AI-24 (it predates the "AI-24 added zero require" outcome). Optional fix; not a blocker.
- **Medium**: The 800-line `single-pr` budget is insufficient. **Reopen `delivery_strategy` at the proposal phase**; recommend three chained PRs (§5.2).
- **Low**: Known scope gap — `error_type` (OpenRouter's typed-error discriminator) is not mapped by AI-32's existing failure taxonomy. Recorded as out-of-scope for this change (§2.8); a future change widens AI-32.
- **Low**: Reasoning extension field rename (`reasoning_content` → `reasoning` / `reasoning_details`). Defused by the existing `decodeChunk` "drop unknown fields" posture (chunk.go:213 uses plain `json.Unmarshal`); the existing test fixture at `reasoning_absence_test.go:159` covers `reasoning_content` and would need a parallel fixture for `reasoning_details` if AI-29 is reopened (deferred).

---

## 6. Ready for proposal

**Yes** — pending the six open questions in §5.3 being resolved by the orchestrator before the proposal phase begins.

**What the orchestrator should tell the user**: the change is structurally sound and concretizes the pre-decided vendor cleanly. The work splits naturally into three chained PRs (wrapper → conformance bridge → live smoke), all gated on the same AI-38.1 capability record, all opt-in for CI, all preserving the AI-29 absence verdict. The user should confirm:

- (a) non-reasoning default model (recommendation: `openai/gpt-4o`),
- (b) paid smoke (~$0.001/run, no `:free` flakiness),
- (c) `delivery_strategy: chained-pr` instead of `single-pr`,
- (d) the three divergences are acceptable as recorded (no widening of openaicompat to handle `error_type`; keep `stream_options.include_usage` set; OpenRouter-specific headers are wrapper-injected).

Once those four confirmations land, the proposal phase can write a `proposal.md` that locks scope to the three chained PRs, and the spec phase writes the four spec files (one per PR plus the cross-cutting conformance record).

---

## 7. Artifact persistence

This artifact is persisted to both:
- **OpenSpec file**: `openspec/changes/add-openrouter-first-provider/explore.md` (this file).
- **Engram topic_key**: `sdd/add-openrouter-first-provider/explore` (mem_save upsert; type `architecture`; capture_prompt `false`; project `cachicamas`).

---

## 8. Limitations and follow-ups

- The OpenRouter OpenAPI spec (1.3 MB) was partially fetched; the relevant `ChatStreamDelta`, `ChatStreamOptions`, `ChatStreamToolCall`, `ChatStreamReasoningDetails`, and `ChatUsage` schemas were verified verbatim. The remaining schemas (`ChatRequest`, `ChatResult`, `ErrorResponse`) were cross-checked against the prose docs. No claim in this artifact is dependent on a schema I did not read.
- I could not verify whether OpenRouter's `error_type` is reliably present on every provider's wire bytes (some providers may not populate it). This is a known gap — see §2.8.
- I could not verify whether OpenRouter's `usage.cost` is always present on the streaming final chunk (the docs strongly imply yes but the example shows it on the non-streaming shape). The openaicompat posture "drop unknown usage fields" is correct regardless.
- I did not attempt to live-test OpenRouter from this exploration environment. The proposal should require the apply phase to add a CI smoke that hits the real endpoint and confirms the three attribution headers, the `[DONE]` sentinel, and the tool-call wire shape — all of which are pinned by OpenRouter's docs but only checkable against the real wire.

---

## Key Learnings

1. The `backend/agent/src/ai/openaicompat/` package is a fully-implemented generic OpenAI-compatible adapter; OpenRouter concretization needs only a thin wrapper, a conformance bridge, and a live smoke — three natural work units, not one.
2. AI-29's struck-node verdict (2026-08-04) is preserved by targeting a non-reasoning default model; the openaicompat package's `decodeChunk` already silently drops OpenRouter's `reasoning` / `reasoning_details` extension fields without changes.
3. `stream_options.include_usage` is deprecated on OpenRouter (no effect; usage is always included) — a divergence from the pre-decided dialect that is harmless because openaicompat sets the field and OpenRouter ignores it; the wire body stays uniform across all openaicompat-target vendors.
4. OpenRouter exposes three attribution headers (`HTTP-Referer`, `X-OpenRouter-Title`/`X-Title`, `X-OpenRouter-Categories`) that no other shared-dialect vendor exposes — they are wrapper-injected, not openaicompat-injected, to keep the openaicompat package's request-header surface narrow.
5. The 800-line `single-pr` budget is insufficient for this change; realistic midpoint is ~2200 lines; three chained PRs (wrapper → conformance bridge → live smoke) is the natural split and matches AI-24/AI-25/AI-38/AI-39's milestone boundaries.
