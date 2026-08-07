# Explore — AI-35 retry and idempotency policy (Layer 1)

> **Change**: `cachicamas-ai-retry-policy`
> **Milestone**: AI-35 — Define retry and idempotency policy (doc 0002)
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-07
> **Target module**: `backend/agent/` (Layer 1, layered not hexagonal — see ADR 0005 § D1; AGENTS.md mandates `cd backend/agent && make test | lint | fmt | build`)
> **Authoring constraint**: doc 0002's authoring constraint binds every artifact of this change. **No Go identifier is invented here.** All names cited below are already present in the merged source at this HEAD (`238b9fa`, worktree `ai-35`).
> **Decision already on the record (2026-08-07 amendment)**: the AI-35 charter's `Deliverable` and `Acceptance` clauses carry the post-amendment text (doc 0002 lines 2098–2099). AI-35.0 closes with a `decision.md` artefact. The seam — **in-adapter loop, factored into a shared helper invoked by the adapter's execute-once function** — is fixed; this explore resolves the eight questions the propose phase must answer.
> **Review budget raised to 1000 changed lines** for this milestone (orchestrator decision 2026-08-07, recorded in the doc 0002 amendment blockquote).
> **Sibling memo**: Engram obs **#2641** (`sdd/cachicamas-ai-retry-policy/decision-pre`) — decision rationale. **#2640** — AI-34 close + workflow. **#2055** — backend runner is `make test` in `backend/agent/` (NOT the frontend runner).

---

## 1. Identity

AI-35 is the Layer 1 half of the retry contract. Its load-bearing sentence (doc 0002 line 2079, verbatim) is **"the partial-output case is never retried at Layer 1"** — the G8 sentence that the rest of the milestone exists to implement. The three subnodes are:

- **AI-35.0** — the policy and seam decision (`[decision]`). Closed ahead of explore by the 2026-08-07 amendment. This explore does not reopen it.
- **AI-35.1** — the retry predicate `[leaf]`. Three test items: retryable-before-output retries; non-retryable-and-anywhere-output-never-retries; terminal-category-never-retries.
- **AI-35.2** — backoff mechanics `[leaf]`. Four test items: retry-after precedence; ctx-aware waits; seeded jitter; bounded attempt count.
- **AI-35.3** — replayability `[leaf]`. Three test items: byte-identical replay; attempt count + final cause in the error chain; per-attempt body drain.

The milestone depends on **AI-32** (failure mapping + retryable taxonomy, done in PR #129) and **AI-33** (drain discipline + cancellation contract, done in PR #129). It blocks **AI-38** (first-adapter conformance roll-up; the OpenRouter conformance bridge already exists at `openaicompat/openrouter/conformance/bridge_test.go`). The Layer 2 half is **AG-15** (doc 0003 lines 687–726), and **AG-15.2's composed-bound test** (doc 0003 line 718) explicitly states "harness attempts × Layer 1 attempts" — that composed-bound assertion cannot be designed without a concrete Layer 1 mechanism, so AI-35 is the unblocking milestone for AG-15.2.

The amendment (doc 0002 lines 2085–2086) names the seam: **in-adapter loop, factored into a shared helper invoked by the adapter's "execute-once" function**. This explore maps that named seam onto existing code and answers the eight investigation questions the orchestrator asked.

---

## 2. Current state — what the merged source already does

Every named function, file and line below is already in the worktree at HEAD `238b9fa`.

### 2.1 The pre-stream boundary is the only place a retry is safe

`backend/agent/src/ai/openaicompat/stream.go:199–240` — the `Stream` entry point. The pre-stream contract (R-ATS-002) is enforced **strictly in order**: `req.IsZero()` (line 200) → `Translate(req)` (line 204) → `ctx.Err()` (line 209) → `c.newRequest(...)` (line 213) → `c.httpClient.Do(httpReq)` (line 218) → `mapResponse(resp, time.Now())` (line 227) → `isStreamContentType(...)` (line 233) → `out := make(chan ai.Event, streamCarrierBuffer)` (line 237) → `go run(ctx, resp, out)` (line 238) → `return out, nil` (line 239). **No carrier is allocated and no goroutine is spawned before line 237.** Every failure shape between line 218 and line 236 is a pre-stream failure (`*ai.Failure` with `Delivery() == DeliveryPreStream` and `PartialOutput() == false` by construction — `provider_failure.go:574`).

**The pre-stream boundary is therefore the unique point at which "no semantic event has been emitted yet" holds unconditionally.** Once `make(chan ai.Event, …)` returns (line 237), the partial-output rule can flip; after `run` sends its first event (line 469 / 484 / 493, the `emit` sites), no retry is safe. The retry helper's natural insertion point is the segment between `c.httpClient.Do(httpReq)` (line 218) and `mapResponse(resp, time.Now())` (line 227) — or, more accurately, it wraps the `Do + mapResponse` pair as the "execute-once" function.

### 2.2 The body is already an in-memory `[]byte` — replay is free

`backend/agent/src/ai/openaicompat/translation.go:24` — `func Translate(req ai.Request) ([]byte, error)` returns the marshaled wire body as a single byte slice. The slice is produced by `appendBody` (`body.go:35`) as a hand-assembled byte buffer (`make([]byte, 0, 256)` + `append`), not a streaming encoder — `body.go:20–29` is explicit about why (`encoding/json` would rewrite verbatim tool-schema bytes, defeating R-ART-010). `newRequest` (`request.go:23–41`) then wraps it as `bytes.NewReader(body)` per attempt (line 28).

**Implication: a retried attempt re-issues from a fresh `bytes.NewReader` over the same `body` slice. Byte-identical replay is structurally guaranteed — there is no live reader to corrupt, no transport state to rebuild.** No body buffering is required. The cost of a replay is exactly one additional HTTP round trip, not one additional memory copy. AI-35.3 item 1 ("byte-compared across attempts") is therefore a property of the existing code path, not a new constraint to engineer for.

### 2.3 The retryable-flag lives at `failure_map.go:86`, derived uniformly from category

`backend/agent/src/ai/openaicompat/failure_map.go:86–93` — `retryableFor(category)` is the single derivation point: `true` for `FailureCategoryRateLimit`, `FailureCategoryUnavailable`, `FailureCategoryTimeout`; `false` for every other category. Its doc comment (`failure_map.go:80–85`) explicitly says: *"Shared, uniform derivation (design.md D6) — stage 2's categorizeStreamError (stream_failure.go) reuses this same function for mid-stream failures, so retryability is one rule, not two independently maintained copies."*

This is the typed signal AI-35.1 item 3 ("Terminal-category failures — authentication, invalid request — never retry regardless of position") reads. Authentication (`FailureCategoryAuthentication`, status 401 → `failure_map.go:108`) and unknown 4xx (400/404/422 → `failure_map.go:111–112`) are both `false` from `retryableFor`. The flag is set on the failure itself (`FailureReport.Retryable`, `provider_failure.go:285`); readers consult `failure.Retryable()` (`provider_failure.go:419–424`).

### 2.4 The partial-output discriminator lives in the producer, not the adapter

`backend/agent/src/ai/openaicompat/stream.go:381–397` — `run` tracks `outputPreceded`, `blockOpen`, and `toolBlocksOpen` locally, updating them only when `emit` returns `true` (i.e. only when the carrier has actually received an event). Doc comment at lines 386–390 is explicit: *"run's own copy, updated only once emit actually confirms a send, is the one true account of what the CARRIER has observed."*

**This is critical for the retry helper's design.** The adapter as a whole does not know whether output has been emitted until the carrier has been handed over AND the producer has sent its first event. **The retry helper runs BEFORE the carrier is handed over** (line 237 in the Stream flow). Therefore, for any failure the helper sees, `outputPreceded` is unconditionally false — by construction. `Failure.PartialOutput()` is documented to always be false for a `*Failure` returned from `PreStreamFailure` (`provider_failure.go:567–574`, doc comment: *"PreStreamFailure takes no output-flag parameter: the fourth cell of R-AIP-010's table — a pre-stream failure that claims output preceded it — is unconstructible here"*).

**AI-35.1 item 2's "absence of a second wire request" test therefore tests the helper's behaviour directly, not `PartialOutput()`'s value** — the helper either retries (invoking `Do` again) or hands the failure up (returning). A test that counts `httpClient.Do` invocations against a scripted provider is the load-bearing assertion.

### 2.5 AI-33's drain discipline is the seam AI-35.3 reuses

`backend/agent/src/ai/openaicompat/stream.go:373–376` — `run`'s defer chain closes `resp.Body` without draining (stream.go:345 in slice 1) before AI-33.5 amended it. AI-33.5's drain-before-close idiom (lines 92–104 doc comment, line 376 implementation) is:

```go
defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
```

For the **pre-stream retry path**, the drain is already supplied by `mapResponse` → `captureBody` (`failure_map.go:35`, `capture.go:31`), which calls `captureBody(resp.Body)` which drains via `io.LimitReader` plus a closing `io.Copy(io.Discard, ...)` (`capture.go:18–22`, `S-AEM-056…059`). So a retryable pre-stream failure already returns the connection to the idle pool through the existing capture path — no second drain helper is required for the status path.

For the **transport-failure path** (an `httpClient.Do` that returns an error before any response exists), there is no body to drain; `http.Client`'s transport handles its own connection teardown. AI-35.3 item 3's "Each failed attempt's response body is closed and drained before the next begins" is therefore **already satisfied by the existing capture.go drain for the status path**, and the transport path needs no additional discipline. The "per-attempt connection leak" hazard AI-35.3 names is mitigated by composition, not by an added `defer io.Copy(...)` in the helper.

### 2.6 The retry-after hint is already parsed and typed

`backend/agent/src/ai/openaicompat/failure_map.go:187–218` — `parseRetryAfter(value, observedAt)` parses both RFC 9110 § 10.2.3 forms (delay-seconds integer, HTTP-date). The result is wrapped in `ai.RetryDelay` (`provider_failure.go:249–268`), presence-typed (`(time.Duration, bool)` two-result shape via `failure.RetryAfter()`, `provider_failure.go:434–439`). AI-35.2 item 1 ("Retry-after, when present, overrides computed backoff") consumes this exact two-result; the absence/zero/value-three-state discipline is already implemented upstream. **No new parsing code is required.**

### 2.7 The rate-limit telemetry carrier is already wired

`backend/agent/src/ai/openaicompat/retry_metadata.go` — `RateLimitTelemetry` carries the three `X-Ratelimit-*` headers (`retry_metadata.go:22–26`) wrapped onto the failure's cause chain (`retry_metadata.go:54–56`). Header access is allowlist-based (`retry_metadata.go:69–83`), credential-leak-safe (proven by `retry_metadata_test.go:84–114`, `TestRetryMetadata_CredentialHeaderNeverCaptured`). This is AI-32.4's deliverable, not AI-35's; the helper inherits the carrier unchanged. **The retry attempt's "final cause" reaches the caller through `errors.As` over the wrapped `*ai.Failure`**; this is exactly the shape AI-35.3 item 2 ("The attempt count and the final cause are both reachable from the returned error chain") reads.

### 2.8 The conformance suite's retry shape is not asserted today

`backend/agent/src/agenttest/conformance_suite.go:43–89` enumerates eight capabilities (`CapStreamingText`, `CapToolCalls`, `CapCompletionMetadata`, `CapCancellation`, `CapTypedFailures`, `CapReasoningContent`, `CapTokenCounting`, `CapCacheBoundary`) plus `CapNone`. **There is no `CapRetry` today.** The closest existing assertion is `terminal/partial_output_discriminator_both_states` (`conformance_terminal.go:105–161`), which proves the discriminator is *answerable* on both states — not that a retry decision follows from it.

`backend/agent/src/agenttest/conformance_terminal.go:184–228` — `terminalFailureCategoryExhaustivenessCase` already iterates every category in `ai.FailureCategories()` (line 188) against both delivery paths (lines 189, 209). The retry-shaped assertion ("category X retries at most N times before giving up") is **a new capability**, not a new case for an existing one. **The agenttest capability vocabulary may need a `CapRetry` member added in the spec phase, OR AI-35.1's predicate test belongs in the real-producer test file alongside `a_i-34_2_test.go` (not the conformance suite).** See § 5 Risks.

### 2.9 The AG-15.2 composed-bound test is the cross-layer contract

`docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:714–719` — AG-15.2's test list item 2 reads: *"The harness's attempt bound holds regardless of any Layer 1 retrying beneath it (asserted by provider-call count against a fake scripted to fail pre-stream forever), and the documented combined ceiling — harness attempts × Layer 1 attempts — is stated in the policy's documentation, where both layers' readers will find it."*

The doc 0002 amendment (line 2093) already records: *"the composed-bound test in AG-15.2 plus replayability tests justify the headroom"* against the 1000-line budget. **AG-15.2 does not need to be amended** — its conditional *"if doc 0002's AI-35 policy decision chooses Layer 1 auto-retry"* (doc 0003 line 698) now resolves to "yes", and its test list stands as written. The Layer 2 test will assert against the **fake** (Line 699: "provider-call count against a fake scripted to fail pre-stream forever"), not the real producer — the fake gives it a clean count, the real producer's script can do the same for AI-35.1's RED test.

### 2.10 The forward guard (AI-00.3) — stdlib-only, sibling-layer deny-by-default

`backend/agent/src/ai/import_boundary_test.go` — the Layer 1 import guard denies everything except the standard library and `github.com/cachicamas/backend/agent/...` (line 104). Forbidden prefixes include sibling layers (`/src/agent`, `/src/coding`, `/src/cmd`, lines 80–82), sibling backend modules (lines 73–74), and the OTel SDK/exporters/otelslog (lines 88–90). `TestLayer1_ModuleHasNoDependencies_ZeroRequires` (line 147) pins `go.mod`'s zero-requires invariant.

**A shared retry helper must satisfy this guard.** The forward guard admits stdlib (`context`, `errors`, `math/rand`, `time`, `sync/atomic`) and the agent module's own packages. A package-private helper inside `backend/agent/src/ai/openaicompat/` is admissible; a sub-package at `backend/agent/src/ai/retryhelper/` (sibling of `openaicompat/`) is admissible; a package outside `backend/agent/` is forbidden. See § 4 for the recommendation.

---

## 3. Test seams available

All helpers below are already shipped; AI-35 consumes them.

| Seam | Symbol | Location | Notes |
| --- | --- | --- | --- |
| Real-producer scriptable transport | `httptest.NewServer(http.HandlerFunc(...))` + `mustClient(t, server.URL)` | `stream_test.go:50–60` | The canonical real-HTTP seam. |
| Single-call transcript replay | `serveTranscripts(transcripts)` | `bridge_test.go:98–115` | Multi-call; each transcript served to the n+1-th request. |
| Per-attempt status scripted | `httptest.NewServer(...)` returning `http.StatusTooManyRequests` + `Retry-After` header | `failure_map_test.go:453–471` (already used for AI-32.4) | The shape AI-35.1's retryable-failure tests use. |
| Raw-HTTP parsing for fixture replay | `mustParseResponse(t, raw)` | `failure_map_test.go:84–87` | Parse raw HTTP into `*http.Response` for `mapResponse` driver. |
| Failure mapping (status → `*ai.Failure`) | `mapResponse(resp, observedAt)` | `failure_map.go:31–37` | The "execute-once" returns this failure on the status path. |
| Capture + drain for status path | `captureBody(resp.Body)` | `capture.go:75–122` | Drains + closes the body before the connection returns to the pool. |
| Retryable taxonomy | `retryableFor(category)` | `failure_map.go:86–93` | `true` only for rate-limit, unavailable, timeout. |
| Retry-after parsing | `parseRetryAfter(value, observedAt)` | `failure_map.go:187–218` | Both RFC 9110 forms; presence-typed result. |
| Failure carrier API | `failure.Retryable()`, `failure.PartialOutput()`, `failure.RetryAfter()`, `failure.Category()` | `provider_failure.go:395–483` | The predicates AI-35.1 reads. |
| Leak detection | `RequireNoGoroutineLeak(tb, scenario)` | `stream_kit_leak.go:107` | Serial-only; AI-35.3's exhaustion tests ride on this. |
| Bounded drain | `DrainAndRecord(tb, ch, timeout)` | `stream_kit_record.go:63` | Used by every AI-3x test. |
| Test-only RoundTripper for adapter-internal headers | (none in `openaicompat`; the bridge has its own) | `openrouter/conformance/bridge_test.go:107–134` | Pattern only; AI-35 doesn't need it. |

**No new helper is required for AI-35.** The "execute-once" pattern is `c.httpClient.Do(httpReq) + mapResponse(resp, time.Now())`; both functions already exist. The helper file is the only new production code beyond the Stream-line edits.

---

## 4. Investigation findings — the eight questions

Each answer below cites real code (file:line). No Go identifier is invented.

### 4.1 Q1 — Where does the shared helper live?

**Recommendation: `backend/agent/src/ai/openaicompat/retry.go` (package-private, alongside `stream.go`).**

Justification against the forward guard (AI-00.3, `import_boundary_test.go:107–162`):

- **Stdlib-only**: the helper uses `context`, `errors`, `time` — all stdlib. No new top-level dependency; `TestLayer1_ModuleHasNoDependencies_ZeroRequires` stays green.
- **No cross-adapter coupling**: the helper's inputs are the `*ai.Failure` shape (already shared via `package ai`) and a closure that produces one. It does not import `backend/agent/src/agenttest/` (forbidden for Layer 1 — `import_boundary_test.go:73–74`), does not import the OpenRouter wrapper, and does not import any sibling layer (`/src/agent`, `/src/coding`, `/src/cmd`). Future Anthropic / Gemini adapters reuse it because they share the same `*ai.Failure` shape, **not because they import a third package**.
- **Sibling-package option** (`backend/agent/src/ai/retryhelper/` or `internal/retry/`) was considered. It would satisfy the guard (own-module prefix `github.com/cachicamas/backend/agent` is allowlisted at line 104). It was rejected for two reasons: (1) it would force `package ai/openaicompat` to import a sibling — a new cross-package edge in a module whose entire `openaicompat/` tree is currently a flat package; (2) the helper's only caller in v1 is `openaicompat`, so a sibling package is one extra hop with no current beneficiary.

**The "forward-guard compliant" check is mechanical**: the helper file's `import` block is stdlib + `github.com/cachicamas/backend/agent/src/ai`; no third-party import. The deny-by-default guard (`import_boundary_test.go:107–137`) fails the file on the first non-stdlib, non-own-package import, so the constraint is structural, not a discipline ask.

### 4.2 Q2 — What is the execute-once function's signature shape?

In **behaviour terms** (no Go types invented here — the spec/design phase owns types):

The execute-once function is the operation that, on a fresh attempt, performs the work whose failure is or is not retryable. For AI-35's seam it is the **HTTP round trip + the wire-side status mapping**, because:

- **`c.httpClient.Do(httpReq)`** (`stream.go:218`) issues the request.
- **`mapResponse(resp, time.Now())`** (`stream.go:227`) translates a non-2xx response into a `*ai.Failure` with `DeliveryPreStream` and `PartialOutput() == false`. A 2xx response returns `nil` and the helper falls through to `isStreamContentType`.

So the execute-once closure has, in behavioural shape:

- **Inputs**: a request (the body slice, the credential, the URL — the helper re-uses the adapter's pre-built `httpReq` and the adapter's `httpClient`).
- **Outputs**: one of three observable outcomes — *(a) a usable 2xx response* (`*http.Response` with `Content-Type: text/event-stream`); *(b) a pre-handover failure* (`*ai.Failure` with `Delivery() == DeliveryPreStream`, `Retryable()` either true or false); *(c) a transport error* (no response object, error from `Do` itself).

The helper's contract:

1. **Call execute-once.** If it returns a 2xx response, return that response — the helper's job is done, the producer goroutine takes over.
2. **If it returns a pre-handover failure** with `Retryable() == true` AND an attempt budget remains AND the context is not cancelled AND the Retry-After (if present) fits within the remaining context budget, wait the appropriate backoff and try again. Otherwise, return the failure.
3. **If it returns a transport error**, treat it as `FailureCategoryUnavailable` with `Retryable() == true` (the same derivation `midStreamFailureFrom` uses, `stream_failure.go:196–211`). Same retry decision as (2).
4. **Per attempt**: drain + close the response body before the next attempt (already supplied by `captureBody` for status failures, `capture.go:75–122`; transport errors have no body to drain).
5. **On exhaustion**: return the last failure wrapped to carry the attempt count. The wrapping shape is **the adapter's existing `*ai.Failure` Cause chain**, accessible through `errors.As(err, &failure)`; the count is a sibling cause type the helper attaches (analogous to `RateLimitTelemetry`'s pattern at `retry_metadata.go:37–58`). The harness consumes it via `errors.As`, mirroring `retry_metadata_test.go:30–33`.

**Type names and function signatures are out of scope here** — the spec phase authors the typed contract, the design phase authors the Go signatures.

### 4.3 Q3 — Where does AI-19.4's partial-output discriminator live?

**It lives in `*ai.Failure.PartialOutput()`** (`provider_failure.go:469–483`). The accessor is bool; `false` for every `*Failure` constructed by `PreStreamFailure` (line 567–574 doc comment: *"PreStreamFailure takes no output-flag parameter"*); `true` for a `*Failure` constructed by `MidStreamFailure` with `outputPreceded == true` (line 585–587).

**For the retry helper, this is unconditional `false` on every failure it inspects.** The helper runs before `make(chan ai.Event, …)` (stream.go:237) and before `go run(ctx, resp, out)` (stream.go:238), so no semantic event has been emitted — `PartialOutput() == false` by construction for any failure the helper could observe. AI-35.1 item 2's "absence of a second wire request" test therefore targets the helper's *behaviour* (count `httpClient.Do` calls against a scripted provider), not the discriminator's value.

### 4.4 Q4 — Where does AI-32's retryable-flag live?

**It lives on `*ai.Failure.Retryable()`** (`provider_failure.go:419–424`), set in the `FailureReport.Retryable` field (`provider_failure.go:285`). For the status path, it is set in `mapErrorResponse` (`failure_map.go:43–78`) via `retryableFor(category)` (`failure_map.go:60`). For the mid-stream transport path, it is set in `midStreamFailureFrom` (`stream_failure.go:196–211`) via the same `retryableFor(category)` (line 200), and **in `failureFromErrorFrame` it is NOT set explicitly** (`stream_failure.go:72–91`) — `FailureCategoryUnknown` defaults to `false` from `retryableFor` (`failure_map.go:88–92`), so an in-band vendor error frame is correctly non-retryable.

**AI-35.1 item 3 reads `failure.Retryable()` directly.** A test that scripts a 401 (`FailureCategoryAuthentication` → `retryableFor == false`, `failure_map.go:108`) and a 400 (`FailureCategoryUnknown` → `retryableFor == false`, `failure_map.go:111–112`) and asserts the helper does NOT retry is the load-bearing assertion.

### 4.5 Q5 — Where does AI-33's drain discipline live?

**It lives in two places, by design:**

1. **Status path** — `mapResponse` calls `captureBody(resp.Body)` (`failure_map.go:35`), which drains via `io.LimitReader` + `io.Copy(io.Discard, ...)` (`capture.go:18–22`), then closes the body (`capture.go:75–122`). The drain-before-close idiom is AI-33.5's deliverable (R-AIS-033), proven by `capture_proof_test.go` (`capture.go:11–13` doc comment).
2. **Producer path** — `run`'s defer chain drains + closes `resp.Body` (`stream.go:376`):
   ```go
   defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()
   ```
   This catches every other exit path of the producer goroutine (clean completion, terminal error, each cancellation moment — proven by R-AIS-033 / S-1, S-2, S-3 at `spec.md:208–230`).

**For the retry helper**, the status path inherits the existing drain via composition (call `mapResponse` → it calls `captureBody` → drain fires). The transport-error path (no response body) needs no drain. **AI-35.3 item 3's "each failed attempt's response body is closed and drained before the next begins" is satisfied by composition, not by an added `defer io.Copy(...)` in the helper.** This is the answer to investigation question 7(c) about memory cost: **there is no body-buffering cost for replay**, because `Translate(req)` returns the body as `[]byte` (in-memory) and `newRequest` wraps a fresh `bytes.NewReader` per attempt (`request.go:28`). A retried attempt re-reads the same slice; the heap footprint per attempt is identical to a non-retried attempt.

### 4.6 Q6 — What conformance tests should AI-35 add?

**Recommended layout**:

- **A new real-producer test file** `backend/agent/src/ai/openaicompat/a_i-35_1_test.go` for AI-35.1's predicate (RED-first against the new helper). Pattern matches `a_i-33_*.go` (internal package, scripted HTTP via `httptest.NewServer` + `mustClient`).
- **A new real-producer test file** `backend/agent/src/ai/openaicompat/a_i-35_2_test.go` for AI-35.2's backoff mechanics. Pattern matches `a_i-33_5a_test.go` (timing injection, scripted 429s with `Retry-After`).
- **A new real-producer test file** `backend/agent/src/ai/openaicompat/a_i-35_3_test.go` for AI-35.3's replayability. Pattern matches `a_i-33_5a_test.go`'s `countTCPConnections` helper (`a_i-33_5a_test.go:197`) extended to count HTTP transactions per attempt.

**Whether to add `conformance_retry.go`** depends on a spec-phase decision: is the retry mechanism a new capability (`CapRetry`, `R-CNF-xxx`) or a property of `CapTypedFailures` (`R-CNF-010` is the closest existing home)?

- **Pro `CapRetry`**: the retry contract is *separable* from the failure-mapping contract; a future "no-retry" adapter would still pass all `CapTypedFailures` cases. The conformance suite already distinguishes separable concerns (cancellation is `CapCancellation`, distinct from typed failures; `conformance_cancellation.go` is its own file).
- **Pro keeping it under `CapTypedFailures`**: the retry decision is a *consequence* of `Retryable()` and `PartialOutput()`; it doesn't introduce new vocabulary. AG-15 (doc 0003) explicitly delegates retry to Layer 2, so the conformance surface for *Layer 1* retry could legitimately stay out of agenttest.

**My recommendation: add `CapRetry` to the conformance suite** (proposed as a spec-phase amendment to `conformance_suite.go:43–89`). Reasoning: the conformance suite is what an adapter author checks against, and AI-38 (the OpenRouter conformance roll-up, doc 0002 line 2118) is the milestone that owns the "real producer passes every conformance case" proof. If AI-35.1's predicate test lives only in `a_i-35_1_test.go`, the OpenRouter adapter's wrapper is not held to it. **The OpenRouter wrapper, however, embeds `*openaicompat.Client` (`openrouter/wrapper.go`) and delegates `Stream` verbatim — so it inherits the helper's behaviour transparently.** The conformance test would therefore exercise both adapters identically, and `CapRetry` would land in the same single registration pattern as `CapCancellation`. This decision is small and reversible; it belongs in the spec phase.

### 4.7 Q7 — Risks

**(a) Cross-adapter coupling that the forward guard would reject.** None, as designed. The helper takes a closure, returns a `*http.Response` or a `*ai.Failure` — both are Layer 1 vocabulary, neither is adapter-specific. The OpenRouter wrapper's `Stream` method (`openrouter/wrapper.go`, lines not yet read but inferred) embeds `*openaicompat.Client` and calls through; it inherits the helper transparently. A future Anthropic adapter would author its own `execute-once` closure over `*http.Client.Do + statusMapper` and reuse the helper. **Risk: low; mitigation: the import_boundary_test.go guard is mechanical, so any regression is caught at `go test` time.**

**(b) Does the in-adapter loop break the Layer 2 composed-bound count?** No, and this is the load-bearing observation behind the seam choice. Each retry consumes one wire request; the composed bound is `harness attempts × Layer 1 attempts`. The in-adapter loop is one Layer 1 retry per harness attempt; the composed count is `N × 1 = N` for v1 (Layer 1 retries independently, Layer 2 retries the harness). AG-15.2's test ("provider-call count against a fake scripted to fail pre-stream forever", doc 0003 line 718) is the assertion that this count holds. **The in-adapter loop does not change the shape of the composed count; it only changes the value of `Layer 1 attempts` from 1 (current) to N+1 (new).**

**(c) Does byte-identical replay need body buffering, and what's the memory cost?** No, and the cost is zero. `Translate(req)` returns `[]byte` (`translation.go:24`); `newRequest` wraps it as `bytes.NewReader(body)` per attempt (`request.go:28`). A retried attempt re-reads the same slice through a fresh reader — no body copy, no buffered pool, no extra heap. **Memory cost of N attempts = memory cost of 1 attempt.** This is what makes the in-adapter loop free; a wrapping component would have to expose a "rebuild request body" callback, which is exactly the API surface bloat the amendment rejects ("would re-create body re-issuance and per-attempt drain as adapter-external seams").

**(d) Helper introduces a hidden mid-stream retry.** The proposed seam is at the pre-stream boundary (§ 2.1). The helper never sees a mid-stream failure — those reach the carrier's terminal event, not the returned error. **The helper's contract MUST be: only pre-stream failures are eligible.** The spec phase must pin this in the typed shape (e.g. by constructing the helper only with the `*ai.Failure` from the pre-stream mapping, not with arbitrary errors).

**(e) Helper does not preserve `RetryAfter` across attempts.** `parseRetryAfter` already captures the hint per failure (`failure_map.go:51`). The helper reads `failure.RetryAfter()` for each attempt, so the hint is preserved per-call. **Risk: low; mitigation: AI-35.2's test item 1 reads the hint from the LAST attempt's failure (or from every attempt's failure), and the helper is mechanical enough that this is a single-line read.**

**(f) Tooling fragility** (per Engram #2636, #2639). The runtime ledger `complete` blocks subsequent `sdd-attempt acquire` after `passed` settle. If `sdd-apply` lands and verifies successfully, the follow-up archive may need to be manual (per the AI-34 precedent). **Risk: low for explore phase, but the orchestrator should anticipate it.**

### 4.8 Q8 — What does AI-35.1's first red test look like in concrete code?

**Sketch (behaviour-only; no Go types invented beyond what's already in source):**

```text
# TestAI-35_1_PartialOutputBoundary_RetryablePreStreamRetries (RED)
# Pattern: a_i-33_*.go (internal package, scripted HTTP, mustClient)

server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Retry-After", "0")
    w.WriteHeader(http.StatusTooManyRequests)
    _, _ = w.Write([]byte(`{"error":{"type":"rate_limit_error","message":"slow"}}`))
}))
defer server.Close()

c := mustClient(t, server.URL)
stream, err := c.Stream(t.Context(), validRequest(t))
# RED: err is non-nil (helper returns the first failure; the loop hasn't been authored yet)
# GREEN: err is still non-nil, but errors.As(err, &f) reports AttemptCount == 3
#         (the retry helper ran exactly N+1 attempts and surfaced the last)
```

The companion test for the "no second wire request" boundary:

```text
# TestAI-35_1_TerminalCategoryNeverRetries (RED)
# A 401 response from the scripted server is FailureCategoryAuthentication
# (failure_map.go:108) with retryableFor == false (failure_map.go:86-93).
# The helper must NOT retry; the first failure is returned as-is.

server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusUnauthorized)
    _, _ = w.Write([]byte(`{"error":{"type":"invalid_api_key","message":"bad"}}`))
}))
defer server.Close()

c := mustClient(t, server.URL)
stream, err := c.Stream(t.Context(), validRequest(t))
# RED: err is nil (the helper doesn't exist; Stream returns the 401 failure directly,
#       but a RED-first posture expects the assertion to drive helper authorship)
# GREEN: err is the 401 failure; request-count side-channel (instrumented RoundTripper)
#         shows exactly 1 Do() call, not N+1.
```

The third companion test for "no retry after partial output":

```text
# TestAI-35_1_PartialOutputNeverRetries (RED)
# By construction, the helper never sees a PartialOutput()==true failure (§ 2.4).
# The test exists to PROVE the construction — that a stream which emits one text
# delta and THEN sees a retryable status does NOT re-issue. This is the AG-15.1
# boundary as seen from Layer 1: the helper's seam ends at the carrier handover.
# The test belongs in AI-35.3 (replayability) as a boundary marker, not in AI-35.1.

server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.WriteHeader(http.StatusOK)
    # write one text delta, then a 429 mid-stream (the in-band error frame path
    # of stream_failure.go is the closer analogy; the 429 mid-stream scenario
    # may need a different scripted shape — see § 5 below for the open question)
}))
```

**The third test is the open seam.** Mid-stream retry is structurally forbidden by the proposed helper (§ 4.7(d)). The RED-first test for "no retry after partial output" therefore asserts a *property of the helper's seam*, not the helper's behaviour in the mid-stream path. The spec phase owns the wording; the design phase owns the implementation.

---

## 5. Decisions to confirm (carry to propose phase)

### D1 — Helper location confirmed: `backend/agent/src/ai/openaicompat/retry.go` (package-private)
See § 4.1. Forward-guard compliant; no cross-adapter coupling; helper file is the only new production code beyond the Stream-line edits.

### D2 — Helper contract scope confirmed: pre-stream only
The helper inspects `mapResponse` failures and transport errors only. Mid-stream failures (after the carrier handover) are never observed by the helper. AI-35.3's "per-attempt drain" is supplied by composition with `captureBody`, not by added defer.

### D3 — Test layout confirmed: three new real-producer test files, conformance amendment TBD
`a_i-35_1_test.go`, `a_i-35_2_test.go`, `a_i-35_3_test.go` — internal `package openaicompat`, matching `a_i-33_*.go` and `a_i-34_*.go`. Whether to amend `conformance_suite.go` with `CapRetry` is a spec-phase decision (§ 4.6).

### D4 — Composed-bound ceiling belongs in the helper's documentation, not its code
AG-15.2's "documented combined ceiling" (doc 0003 line 718) is `harness attempts × Layer 1 attempts`. The helper's package doc-comment carries the Layer 1 multiplier (e.g. `const defaultMaxAttempts = 3`, meaning "wire requests per logical Stream call"), so a reader of either layer finds the constant.

### D5 — Retry budget constant confirmed: helper default `defaultMaxAttempts = 3`
Doc 0002 amendment (line 2093) raised the review budget to 1000 lines; AI-35.2 item 4 ("A documented maximum attempt count terminates retrying, asserted as exactly N+1 wire requests followed by the last error") names N+1. The proposed default of `3` (meaning 4 wire requests total per logical call) is conservative; the value is package-private and overridable in tests via a parameter to the helper. **This is a default, not a config surface** — the Layer 1 contract is "Layer 1 retries up to N+1 times on a retryable pre-stream failure"; the harness reads the value from the package doc comment.

---

## 6. Open questions for the propose phase

### Q1 — Helper signature shape (carry to spec)
The spec phase authors the typed signature. The behaviour is fixed (§ 4.2): a closure that returns either a usable 2xx response, a pre-handover failure, or a transport error. The wrapper struct (attempt count, last failure, drain policy) is the spec phase's decision.

### Q2 — `CapRetry` membership in `conformance_suite.go`
§ 4.6 — recommend `CapRetry` so AI-38's conformance roll-up exercises the retry predicate against every adapter, not just `openaicompat`. Spec phase decides.

### Q3 — Mid-stream "no retry after partial output" assertion shape
§ 4.8's third test sketch is structurally redundant with the helper's seam (§ 4.7(d)). Two acceptable postures:

- **(a)** Test belongs in AI-35.3 as a boundary marker — proves the seam's construction, asserts that `httpClient.Do` count is 1 after a mid-stream retryable failure (a 429 mid-stream).
- **(b)** Test is redundant; AI-35.1's RED tests are sufficient, and the boundary is documented in the helper's package doc.

Recommend (a). The cost is one short test file; the benefit is the boundary is asserted mechanically, not by inspection.

### Q4 — Retry-after's "overrides computed backoff" math
AI-35.2 item 1 names both inputs (retry-after, computed backoff with seeded jitter). The helper's `time.NewTimer` selection + `ctx.Done()` race is the proposed shape (mirroring `run`'s `emit`, `stream.go:588–596`). Spec phase authors the jitter seed and the backoff growth curve (linear vs exponential).

### Q5 — Default attempt budget
§ 5 D5 proposes `defaultMaxAttempts = 3` (4 wire requests per logical call). The value is documented and testable; the spec phase confirms.

---

## 7. Recommended next step

**Proceed to `sdd-propose` for change `cachicamas-ai-retry-policy`.** The seam is fixed (in-adapter loop, factored into a shared helper); the eight investigation questions are answered with concrete code citations; the risks are bounded and mitigated by the forward guard (mechanical) and by composition with the existing AI-32 / AI-33 seams (no new drain discipline needed).

The propose phase must:
1. Author `proposal.md` with the deliverable wording (matches doc 0002 line 2098 verbatim) and the helper's location (`backend/agent/src/ai/openaicompat/retry.go`).
2. Author `decision.md` (AI-35.0 closing checklist item 1, post-amendment) carrying the rationale verbatim from doc 0002 lines 2085–2086 + the seam choice.
3. List the three subnodes' test lists verbatim from doc 0002 lines 2121–2144.
4. Confirm `CapRetry` membership (§ 6 Q2) or defer it to spec.
5. Reference AG-15.2's composed-bound test (doc 0003 line 718) as the cross-layer contract that becomes writable because of this change.

No new top-level Go dependency. No conformance amendment in propose (decide in spec). No production-shape change beyond the helper file and the Stream-line edits that invoke it.

---

## 8. References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2077–2102` (incl. 2026-08-07 amendment blockquote)
- **AI-35.0/35.1/35.2/35.3 test lists** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2113–2144`
- **AG-15.2 composed-bound contract** — `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:714–719`
- **Spec (R-AIS-031..038 Wave 5)** — `openspec/specs/ai-stream-lifecycle/spec.md:208–365`
- **Spec (V-FAIL-09 partial-output discriminator, V-FAIL-15 Layer 1 retry policy)** — `openspec/specs/ai-contract-vocabulary/spec.md:255–261`
- **Producer Stream (pre-stream contract + drain discipline)** — `backend/agent/src/ai/openaicompat/stream.go:92–104, 199–240, 371–376`
- **Wire-side failure mapper + retryable taxonomy** — `backend/agent/src/ai/openaicompat/failure_map.go:31–93`
- **Mid-stream failure mapper (categorizeStreamError reuses retryableFor)** — `backend/agent/src/ai/openaicompat/stream_failure.go:129–211`
- **Retry-after parser + presence-typed result** — `backend/agent/src/ai/openaicompat/failure_map.go:187–218`
- **Rate-limit telemetry carrier** — `backend/agent/src/ai/openaicompat/retry_metadata.go:37–83`
- **Rate-limit telemetry RED suite** — `backend/agent/src/ai/openaicompat/retry_metadata_test.go:1–136`
- **Capture + drain (status path)** — `backend/agent/src/ai/openaicompat/capture.go:31, 75–122`
- **Body marshalling (Translate returns `[]byte`)** — `backend/agent/src/ai/openaicompat/translation.go:24`, `body.go:35–55`
- **Request build (body wrapped as bytes.NewReader per attempt)** — `backend/agent/src/ai/openaicompat/request.go:23–41`
- **Provider failure API (Retryable, RetryAfter, PartialOutput, Delivery)** — `backend/agent/src/ai/provider_failure.go:249–496`
- **Conformance skeleton (Factory, Capability)** — `backend/agent/src/agenttest/conformance_suite.go:43–227`
- **Conformance terminal + partial-output discriminator case** — `backend/agent/src/agenttest/conformance_terminal.go:105–228`
- **Conformance cancellation case** — `backend/agent/src/agenttest/conformance_cancellation.go:46–131`
- **OpenRouter conformance bridge** — `backend/agent/src/ai/openaicompat/openrouter/conformance/bridge_test.go:1–470`
- **Forward guard (AI-00.3)** — `backend/agent/src/ai/import_boundary_test.go:107–241`
- **Layered module (NOT hexagonal) + `cd backend/agent && make test`** — `openspec/AGENTS.md` rule 2
- **Engram obs #2641** — decision rationale (already adopted)
- **Engram obs #2640** — AI-34 close + workflow
- **Engram obs #2055** — backend runner is `make test` in `backend/agent/`
