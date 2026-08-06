# Citations — OpenAI-compatible error wire shape (AI-32)

## Provenance

- **Repository:** `github.com/openai/openai-openapi`
- **Pinned commit:** `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439` (same commit cited by the request-side claims in `backend/agent/src/ai/openaicompat/doc.go` and by the AI-28 streaming citations)
- **File:** `openapi.yaml` (84,002 lines; single spec file at this commit)
- **Retrieval date:** 2026-08-04
- **Retrieval method:** `curl -sSL https://raw.githubusercontent.com/openai/openai-openapi/d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439/openapi.yaml` (HTTP 200, 2,845,527 bytes), downloaded once into the session scratchpad and reused. All line numbers refer to `openapi.yaml` at this exact commit; every quote and count was extracted with `grep`/`sed`/`awk` over the downloaded copy.

Scope note: the milestone maps failures of the **Chat Completions** surface (`POST /chat/completions`). Where Assistants, Realtime, or Responses API schemas are cited below, it is to establish what the spec pins elsewhere versus what it leaves undocumented for Chat Completions.

---

## E1 — Vendor error-body shape: `Error` and `ErrorResponse`

**`Error` schema — every declared field:**

> `openapi.yaml:37842-37861`
> ```yaml
>     Error:
>       type: object
>       properties:
>         code:
>           anyOf:
>             - type: string
>             - type: "null"
>         message:
>           type: string
>         param:
>           anyOf:
>             - type: string
>             - type: "null"
>         type:
>           type: string
>       required:
>         - type
>         - message
>         - param
>         - code
> ```

**`ErrorResponse` wrapper:**

> `openapi.yaml:37879-37885`
> ```yaml
>     ErrorResponse:
>       type: object
>       properties:
>         error:
>           $ref: "#/components/schemas/Error"
>       required:
>         - error
> ```

**Reading:** the vendor error body declares exactly four fields — `type` (string), `message` (string), `param` (string|null), `code` (string|null) — **all four required**, with `param` and `code` required-but-nullable; the wrapper shape is `{"error": {...}}` via `ErrorResponse`, whose single `error` field is required. No enum constrains `type` or `code` in these schemas (see E5). Note the spec is inconsistent about the wrapper: some declared error responses return bare `Error`, others `ErrorResponse` (see E2).

---

## E2 — Non-200 HTTP statuses declared by the spec (chat endpoint: load-bearing NEGATIVE)

**Chat Completions declares only `200`.** The `POST /chat/completions` (create) operation declares exactly one response:

> `openapi.yaml:2134-2143` (path `/chat/completions:` at line 1985)
> ```yaml
>       responses:
>         "200":
>           description: OK
>           content:
>             application/json:
>               schema:
>                 $ref: "#/components/schemas/CreateChatCompletionResponse"
>             text/event-stream:
>               schema:
>                 $ref: "#/components/schemas/CreateChatCompletionStreamResponse"
> ```

`GET /chat/completions` (list) likewise declares only `"200"` (line 2037). An awk sweep of every `"NNN":` response key inside the `/chat/completions` path block (lines 1985-3010) finds exactly those two `"200"` entries and nothing else.

**Spec-wide declared status inventory** (`grep -E '^        "[0-9]{3}":'` over all 84,002 lines):

| Status | Count | Where |
|---|---|---|
| `200` | 301 | everywhere |
| `201` | 3 | `/evals` (4691), `/evals/{eval_id}/runs` (5327), `/realtime/calls` (16519) |
| `400` | 10 | `/evals/{eval_id}/runs` (5333, schema `Error`); 9 organization-admin endpoints (11257, 11460, 12404, 12477, 12537, 13062, 13123, 13231, 13292 — schema `ErrorResponse`) |
| `404` | 6 | `/evals/{eval_id}` (5102) and `/evals/{eval_id}/runs/{run_id}` (5950) — schema `Error`; `/responses/{response_id}` (18557) and `/cancel` (18614) plus beta twins (27318, 27388) — schema `Error` |

Representative declared error responses:

> `openapi.yaml:5102-5107`
> ```yaml
>         "404":
>           description: Evaluation not found.
>           content:
>             application/json:
>               schema:
>                 $ref: "#/components/schemas/Error"
> ```

> `openapi.yaml:11257-11262`
> ```yaml
>         "400":
>           description: Error response when updating the default project.
>           content:
>             application/json:
>               schema:
>                 $ref: "#/components/schemas/ErrorResponse"
> ```

**Reading (load-bearing negative):** `/chat/completions` declares no non-200 response at all, and the spec as a whole declares **no** `401`, `403`, `429`, or `5xx` on any endpoint at this commit — HTTP status-code mapping for chat failures is therefore dialect-conventional, not spec-derived, and requires fixture pins. A secondary finding: even where errors are declared, the spec is split between bare `Error` (evals, responses) and wrapped `ErrorResponse` (organization admin), so the adapter should tolerate both `{...}` and `{"error": {...}}` bodies.

---

## E3 — Rate-limit signaling: `Retry-After`, `x-ratelimit-*`, 429 semantics (NEGATIVE)

**Finding: NEGATIVE.** Across all 84,002 lines: `Retry-After` → 0 hits; `retry-after` → 0; `x-ratelimit` → 0; `X-RateLimit` → 0; `ratelimit` → 0; `429` → 0. The spec declares no response headers for rate limiting and no 429 response anywhere.

What the spec **does** contain under rate-limit terms is unrelated to HTTP 429 signaling:

- Organization admin CRUD for per-project model rate limits, e.g. "Returns the rate limits per model for a project." (12291) and "Updates a project rate limit." (12373) — configuration objects, not error signaling.
- Assistants `RunObject.last_error.code` and `RunStepObject.last_error.code` include `rate_limit_exceeded` as a **stored state value** on the run object, not an HTTP response (see E5; 60404-60407, 61149-61151).
- Prose on vector-store ingestion limits: "Vector store file attach requests are rate limited per vector store (300 requests per minute ...)" (22032-22034) — a documented limit with no header or status semantics attached.

**Reading:** the pinned spec gives the adapter zero normative basis for `Retry-After`, `x-ratelimit-*` headers, or 429 behavior; any rate-limit mapping for OpenAI-compatible dialects must come from fixtures/provider docs, not this spec.

---

## E4 — Errors delivered inside a Chat Completions stream (combined posture: schema-negative + prose-negative)

**Schema side (established in AI-28 C6, re-verified):** the `POST /chat/completions` `text/event-stream` response references only `CreateChatCompletionStreamResponse` (2141-2143); the chunk schema body (33295-33440) and delta schema body (31479-31521) contain zero occurrences of `error`; the in-stream `ErrorEvent` (37862-37878, `event: error` carrying `Error`) is referenced solely by the Assistants `AssistantStreamEvent` union (28622), and `ResponseErrorEvent` (57751) belongs to the Responses API.

**Prose side (new):** a case-insensitive sweep for `error` over the entire `/chat/completions` path block (lines 1985-3010, including all x-oaiMeta prose and streaming examples), the full `CreateChatCompletionRequest` schema (32739-33295), the chunk schema (33295-33440), and `ChatCompletionStreamOptions` (31425-31478) returned **zero hits**. Not one description, example, or note in the chat surface mentions errors — mid-stream or otherwise. The Assistants `ErrorEvent` description, by contrast, is explicit for its own surface:

> `openapi.yaml:37875-37876`
> ```yaml
>       description: Occurs when an [error](/docs/guides/error-codes#api-errors) occurs.
>         This can happen due to an internal server error or a timeout.
> ```

**Reading:** the combined posture is fully negative — neither schema nor prose defines or even mentions an in-stream `{"error": ...}` data frame for Chat Completions at this commit; real-world providers that emit one are exercising undocumented dialect behavior the adapter must tolerate defensively and pin with fixtures.

---

## E5 — `type`/`code` value vocabularies: what is pinned vs absent

**Absent where it matters:** the vendor `Error` schema (37842-37861) leaves `type` and `code` as unconstrained strings — no enum, no examples. Searches for canonical values in the chat error context: `invalid_request_error` → 4 hits (none in `Error`); `insufficient_quota` → 0; `invalid_api_key` → 0; `authentication_error` → 0; `context_length_exceeded` → 0.

**Pinned elsewhere in the spec (other surfaces):**

Responses API `ResponseError`/`ResponseErrorCode` — a required, enumerated `code`:

> `openapi.yaml:57726-57735` (enum continues to line 57750)
> ```yaml
>     ResponseErrorCode:
>       type: string
>       description: |
>         The error code for the response.
>       enum:
>         - server_error
>         - rate_limit_exceeded
>         - invalid_prompt
>         - data_residency_mismatch
>         - bio_policy
> ```

(The full enum, 57731-57750, is: `server_error`, `rate_limit_exceeded`, `invalid_prompt`, `data_residency_mismatch`, `bio_policy`, `vector_store_timeout`, plus 14 image-specific codes `invalid_image` ... `failed_to_download_image`. A Beta twin `BetaResponseErrorCode` exists at 79561.)

Assistants `RunObject.last_error`:

> `openapi.yaml:60401-60407`
> ```yaml
>             code:
>               type: string
>               description: One of `server_error`, `rate_limit_exceeded`, or `invalid_prompt`.
>               enum:
>                 - server_error
>                 - rate_limit_exceeded
>                 - invalid_prompt
> ```

(`RunStepObject.last_error.code` is the two-value subset `server_error` | `rate_limit_exceeded`, 61146-61151.)

Realtime error events mention `invalid_request_error` only as a prose example, with no enum:

> `openapi.yaml:47762-47766` (`RealtimeBetaServerEventError`, schema at 47736; twin `RealtimeServerEventError` at 51972/52001)
> ```yaml
>             type:
>               type: string
>               description: >
>                 The type of error (e.g., "invalid_request_error",
>                 "server_error").
> ```

**Reading:** for the Chat Completions error body, `type` and `code` vocabularies are entirely unpinned at this commit — `invalid_request_error` appears only as a Realtime prose example and `insufficient_quota` not at all — so the taxonomy mapping must treat `error.type`/`error.code` as open string sets pinned by fixtures; the only closed enums (`ResponseErrorCode`, run `last_error.code`) belong to the Responses and Assistants surfaces.

---

## E6 — Request-id surfacing (`x-request-id` or similar) (NEGATIVE as schema; one prose trace)

**Finding: NEGATIVE at the schema level.** The spec declares no response headers anywhere (`headers:` sections under responses: none relevant; `Retry-After`/ratelimit sweep in E3 also covers this), and `x-request-id` is never documented as part of any response contract. The string occurs exactly **once** in the whole file, inside a curl example for the image-edit endpoint that dumps response headers to stderr:

> `openapi.yaml:8254` (x-oaiMeta curl example of `POST /v1/images/edits`, "Create image edit")
> ```
>                 curl -s -D >(grep -i x-request-id >&2) \
> ```

Other searches: `X-Request-Id` → 0; `request-id` → 1 (the same line 8254); `request_id` → 21 hits, all of which are body fields of other surfaces (eval run results, Realtime/usage objects), none a documented HTTP response header of `/chat/completions`.

**Reading:** the pinned spec never documents a request-identifier response header for Chat Completions (the lone `x-request-id` occurrence is an incidental curl idiom in an images example), so surfacing a request id in error reports is dialect-conventional and must be fixture-pinned, with absence tolerated.

---

## Negative findings

1. **`/chat/completions` declares no non-200 response (E2).** Both operations declare only `"200"` (lines 2037, 2135); the whole spec declares only `200`/`201`/`400`/`404`, never `401`, `403`, `429`, or any `5xx`. Status-code mapping is dialect-conventional and needs fixture pins.
2. **No rate-limit signaling exists (E3).** `Retry-After` → 0, `retry-after` → 0, `x-ratelimit` → 0, `X-RateLimit` → 0, `ratelimit` → 0, `429` → 0 across 84,002 lines.
3. **No in-stream chat error frame, in schema or prose (E4).** Case-insensitive `error` over lines 1985-3010 (chat path block), 32739-33295 (request schema), 33295-33440 (chunk schema), 31425-31478 (stream options) → 0 hits in each range; `ErrorEvent` is Assistants-only (union ref at 28622).
4. **No `type`/`code` vocabulary for the vendor `Error` (E5).** `Error.type` and `Error.code` are unconstrained strings; `insufficient_quota` → 0, `invalid_api_key` → 0, `authentication_error` → 0, `context_length_exceeded` → 0; `invalid_request_error` appears only in Realtime prose (47765, 52001) and examples (47799, 52035).
5. **No documented request-id response header (E6).** `x-request-id` → 1 hit total, an images-endpoint curl example (8254), not a contract; `X-Request-Id` → 0.
6. **Error-body wrapper is inconsistent where declared (E1/E2).** Evals and Responses 404/400s return bare `Error` (5102-5107, 5333-5338, 18557-18562); organization-admin 400s return wrapped `ErrorResponse` (11257-11262 et al.). The adapter must tolerate both shapes.

## Search log

All commands run over the downloaded `openapi.yaml` in the session scratchpad. Pattern → hit count (grep unless noted):

| Pattern / command | Hits |
|---|---|
| `^    Error:` (anchored schema name) | 1 (line 37842) |
| `ErrorEvent\|error.event\|ErrorResponse` (from AI-28, reused) | 15+ (ErrorResponse def 37879; refs 11262, 11465, 12409, 12482, 12542, 13067, 13128, 13236, 13297) |
| `grep -E '^        "[0-9]{3}":'` (all declared statuses) | 320 total: 200 ×301, 201 ×3, 400 ×10, 404 ×6 |
| awk map of non-200 statuses to owning paths | 19 rows (evals, org admin, realtime, responses) |
| awk `^        "[0-9]{3}":` in lines 1985-3010 (chat path block) | 2 (both `"200"`: 2037, 2135) |
| `Retry-After` | 0 |
| `retry-after` | 0 |
| `x-ratelimit` | 0 |
| `X-RateLimit` | 0 |
| `ratelimit` | 0 |
| `429` | 0 |
| `rate limit` | 28 (admin CRUD, vector-store prose; no 429/header semantics) |
| `rate_limit` | 38 (admin objects, `rate_limit_exceeded` enums) |
| `rate_limit_exceeded` | 6 (57732, 60406, 61151, 79567, plus descriptions 60403, 61148) |
| `invalid_request_error` | 4 (47765, 47799, 52001, 52035 — all Realtime) |
| `insufficient_quota` | 0 |
| `invalid_api_key` | 0 |
| `authentication_error` | 0 |
| `context_length_exceeded` | 0 |
| `server_error` | 24 (ResponseErrorCode 57731, run/step last_error, Realtime prose, eval grader fields) |
| `invalid_prompt` | 4 (57733, 60407, plus descriptions) |
| `x-request-id` | 1 (line 8254, images curl example) |
| `X-Request-Id` | 0 |
| `request-id` | 1 (same line 8254) |
| `request_id` | 21 (body fields of evals/realtime/usage objects; no response header) |
| awk ci `error` in lines 1985-2740 and 1985-3010 (chat path) | 0 |
| awk ci `error` in lines 33295-33440 (chunk schema) | 0 |
| awk ci `error` in lines 31425-31478 (stream options) | 0 |
| awk ci `error` in lines 32739-33295 (chat request schema) | 0 |
