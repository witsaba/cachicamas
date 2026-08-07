# Design: Define retry and idempotency policy (AI-35)

> **Change**: `cachicamas-ai-retry-policy` · **Milestone**: AI-35 (doc 0002 lines 2077–2144)
> **Wave**: 5 — Harden · **Module**: `backend/agent/` (layered per ADR 0005 § D1, NOT hexagonal — `cd backend/agent && make test`)
> **Strategy**: single-pr · strict TDD on · stdlib-only · review budget **1000 changed lines** (doc 0002 amendment line 2093)

## 1. Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-retry-policy` |
| **Milestone** | AI-35 (doc 0002 lines 2077–2144; charter test list at lines 2113–2144) |
| **Inputs** | `proposal.md` (Flag 1 + Flag 2 closed); `specs/ai-stream-lifecycle/spec.md` (R-AIS-041..044); `specs/ai-provider-conformance-suite/spec.md` (R-CNF-019 + R-CNF-017 modified + CapRetry) |
| **Scope (in)** | Pre-stream auto-retry helper (`internal/retry/`); conformance capability `CapRetry`; `Factory.Retry *bool` field; one conformance case body; three real-producer test files |
| **Scope (out)** | Harness-level retry (AG-15.1/15.2); model failover (AG-15.3); mid-stream retry; new top-level Go dependencies; configurability of `defaultMaxAttempts` |
| **Review budget** | 1000 changed lines (per doc 0002 amendment line 2093) |
| **Test runner** | `cd backend/agent && make test` (`go test -race -v ./...`); `make lint` (`go vet` + `golangci-lint`); `make fmt` |
| **Module / package layout** | `backend/agent` is layered (NOT hexagonal) — `src/ai` (Layer 1) ← `src/agent` (Layer 2) ← `src/coding` (Layer 3) ← `src/cmd/cachicamas` — per ADR 0005 § D1 |
| **Conformance amended** | `ai-provider-conformance-suite` — `R-CNF-019` added; `R-CNF-017` modified (totality over `[9]` capability entries); one optional capability `CapRetry` (CAP-O-04) added |
| **Spec amended** | `ai-stream-lifecycle` — `R-AIS-041`..`R-AIS-044` added (4 requirements, 14 scenarios) |
| **Forward guard** | `backend/agent/src/ai/import_boundary_test.go:107-162` — stdlib + own-module (`github.com/cachicamas/backend/agent/...`) only; helper is stdlib + `ai` |
| **Pinned decisions** | Helper location = `backend/agent/src/ai/internal/retry/` (Go `internal/`); "no retry after partial output" lives in AI-35.3 (Flag 2); `CapRetry` conformance capability YES (proposal § 5) |

---

## 2. Technical Approach

One new Go package — `backend/agent/src/ai/internal/retry/` (Go `internal/` directory) — exports the helper `retry.Loop`. The package's import path is `github.com/cachicamas/backend/agent/src/ai/internal/retry`. Go's `internal/` directory semantics restrict importability to the immediate parent tree (`src/ai/`), which is exactly the encapsulation the doc 0002 amendment line 2086 anticipates ("future adapters (Anthropic, Gemini, ...) reuse it without re-implementing the discipline") without coupling future adapters to `openaicompat`'s translation tables.

The adapter's `Stream` (`backend/agent/src/ai/openaicompat/stream.go:199-240`) wraps the `Do + mapResponse` segment in the helper as the "execute-once" closure. The helper runs **before** the carrier handover at `stream.go:237`, so no semantic event has been emitted and `PartialOutput() == false` is unconditional for every failure the helper inspects — the load-bearing invariant that makes `V-FAIL-15` ("the partial-output case is never retried at Layer 1") a property of the call site, not a runtime check.

Byte-identical replay is structurally free: `Translate(req)` returns `[]byte` (`translation.go:24`); `newRequest` wraps `bytes.NewReader(body)` per call (`request.go:28`). The helper's `executeOnce` closure re-builds `httpReq` per attempt from the same body slice — no body buffering, no transport state to rebuild. Drain discipline is supplied by composition: `mapResponse` → `captureBody` (`capture.go:75-122`) drains + closes for the status path; transport errors have no body to drain. No new `defer io.Copy(...)` in the helper.

The conformance suite grows from `[8]` to `[9]` capabilities with `CapRetry` (CAP-O-04) added between `CapCacheBoundary` and `CapNone`. `Factory` gets a fourth `*bool` field `Retry`, mirroring `Reasoning` / `TokenCounting` / `CacheBoundary`; `nil` fails construction per `R-CNF-002` / `S-CNF-006`. The conformance case body `conformance_retry.go` asserts `S-CNF-069..075` against the real producer (`*openaicompat.Client`) configured via `Config.HTTPClient` against a scriptable `http.RoundTripper`. The OpenRouter wrapper inherits the helper transparently (`openrouter/wrapper.go` embeds `*openaicompat.Client` and delegates `Stream` verbatim), so its conformance bridge extends the same case body without rewriting assertions.

`defaultMaxAttempts = 3` is the helper's exported constant — wire-request count per logical `Stream` call when retries are exhausted is `N+1 = 4`. The constant lives in the helper's `doc.go` (package doc comment) and is referenced verbatim by AG-15.2's composed-bound test (doc 0003 line 718) — the cross-layer visibility R-AIS-044 / S-1 pins.

---

## 3. Architecture Decisions

### D1 — Helper location: `backend/agent/src/ai/internal/retry/`

**Choice**: New Go package at `backend/agent/src/ai/internal/retry/` (package `retry`), import path `github.com/cachicamas/backend/agent/src/ai/internal/retry`. Go `internal/` directory semantics restrict importability to the immediate parent (`src/ai/` and below).

**Alternatives considered**:
- `backend/agent/src/ai/openaicompat/retry.go` (explore's pick): package-private to `openaicompat`, no cross-package edge, but forces future Anthropic / Gemini adapters to import `package openaicompat` — coupling the WRONG direction.
- `backend/agent/src/ai/retryhelper/` (sibling of `openaicompat`): would satisfy the forward guard but adds an extra cross-package edge in `stream.go` for one current beneficiary (the v1-only-caller assumption).

**Rationale**: The doc 0002 amendment (line 2086) verbatim says "future adapters (Anthropic, Gemini, ...) reuse it without re-implementing the discipline" — the amendment is the asymmetry that justifies the edge. `internal/retry/` is within `github.com/cachicamas/backend/agent/...` (the only non-stdlib allowlist entry at `import_boundary_test.go:103-105`); forward guard stays green; no third-party import; `internal/` semantics give exactly the encapsulation we want (any package under `src/ai/` can call the helper, nothing outside can). The proposal's Flag 1 resolution (verbatim at `proposal.md:96-103`) is the authoritative decision.

### D2 — In-adapter loop, pre-stream only

**Choice**: The `Stream` method (`stream.go:199-240`) calls `retry.Loop(ctx, body, cfg, executeOnce)` as the carrier of the `c.httpClient.Do(httpReq) + mapResponse(resp, time.Now())` pair (lines 218 and 227). The helper returns either a 2xx `*http.Response` or a `*ai.Failure` whose cause chain carries the attempt-report sibling.

**Alternative considered**: Wrapping the retry logic in an adapter-external component. Rejected — would re-create body re-issuance and per-attempt drain as adapter-external seams (rebuild-callback + drain-callback) — adds API surface without removing coupling. Per Engram #2641: "in-adapter + shared helper beats wrapping component because (a) AI-32 error mapping is at the adapter edge, (b) AI-33's per-attempt drain discipline is reusable in-process, (c) bytes-identical replay is trivial when the marshaled body is in-process".

**Rationale**: The pre-stream contract (`stream.go:199-240` strict ordering) is the unique point at which "no semantic event has been emitted yet" holds unconditionally. Once `make(chan ai.Event, ...)` returns at line 237, the partial-output rule can flip; after `run` sends its first event (line 469 / 484 / 493), no retry is safe. The helper's insertion point — between `Do` and `mapResponse`, with the helper wrapping both as `executeOnce` — is fixed by the 2026-08-07 doc 0002 amendment (lines 2081-2093).

### D3 — Byte-identical replay (structurally free)

**Choice**: The `executeOnce` closure receives `body []byte` (the slice returned by `Translate(req)` at `stream.go:204`) and re-builds `httpReq` per attempt via `c.newRequest(ctx, body, "chat", "completions")`. The closure does NOT receive a pre-built `httpReq` (which would have a consumed `bytes.Reader`).

**Alternative considered**: Receive `httpReq *http.Request` and rely on `req.GetBody`. Rejected — `http.NewRequestWithContext`'s `GetBody` setup is implicit and adapter-internal; the helper's `executeOnce` closure re-uses `c.newRequest` to keep the adapter's request-build logic in one place (`request.go:23-41`).

**Rationale**: `Translate(req)` returns `[]byte` (`translation.go:24`); `newRequest` wraps `bytes.NewReader(body)` per call (`request.go:28`). The slice is produced by `appendBody` (`body.go:35`) as a hand-assembled byte buffer, not a streaming encoder. Each retry re-issues from a fresh `bytes.NewReader` over the same slice — no drift, no truncation, no re-encoding. Memory cost of N attempts equals memory cost of 1 attempt. R-AIS-043 / S-1 (byte-identical replay) is a property of the existing code path, not a new constraint.

### D4 — Drain discipline by composition (no new defer in helper)

**Choice**: The helper has NO new `defer io.Copy(...)`. The status path inherits the existing drain via `mapResponse` → `captureBody` (`capture.go:75-122`). The transport-error path has no body to drain — `http.Client` handles its own connection teardown.

**Alternative considered**: Adding a `defer` in the helper for symmetry. Rejected — would shadow the AI-33.5 drain discipline (`stream.go:373-376`), violate "single producer model" (R-ATS-003), and add a helper surface that already exists elsewhere.

**Rationale**: AI-33.5's drain-before-close idiom (`stream.go:92-104` doc comment, `stream.go:376` implementation) is the producer-path discipline; `capture.go:117-122` is the bounded-capture-path discipline. Both run in their own goroutines. The retry helper sits before the producer goroutine starts (`stream.go:237-238`), so any per-attempt drain runs in the helper's own caller stack — the `mapResponse` call already invokes `captureBody` (status path) and the transport-error path has no body to drain. R-AIS-043 / S-3 (per-attempt drain) is satisfied by composition, not by added defer.

### D5 — Retryable taxonomy reads `failure.Retryable()`

**Choice**: The helper classifies retryability via `(*ai.Failure).Retryable()` (`provider_failure.go:419-424`), set by `retryableFor(category)` (`failure_map.go:86-93`). True only for `FailureCategoryRateLimit`, `FailureCategoryUnavailable`, `FailureCategoryTimeout`; `false` for every other category.

**Alternative considered**: Re-deriving retryability from the raw `*http.Response` in the helper. Rejected — duplicates the derivation `failure_map.go:86-93` already centralizes and breaks the "one rule, not two independently maintained copies" posture `failure_map.go:80-85` documents.

**Rationale**: AI-32's retryable taxonomy is the typed signal AI-35.1 reads. Authentication (`FailureCategoryAuthentication`, status 401 → `failure_map.go:108`) and unknown 4xx (`FailureCategoryUnknown`, status 400/404/422 → `failure_map.go:111-112`) both return `false` from `retryableFor`. The helper's terminal-category-never-retries assertion (R-AIS-041 / S-2) tests `failure.Retryable() == false` directly via a scripted 401 / 400 / 404 / 422.

### D6 — Partial-output discriminator as construction

**Choice**: The helper's contract is pre-stream only. It never sees a `PartialOutput() == true` failure. The helper's call site is between `Do` and the carrier handover (`stream.go:218-237`); `make(chan ai.Event, ...)` returns at line 237; `go run(...)` starts at line 238. For any failure the helper inspects, no semantic event has been emitted — `PartialOutput()` is unconditionally `false` by construction (`provider_failure.go:567-574` doc comment).

**Alternative considered**: A runtime check inside the helper for `PartialOutput() == true`. Rejected — by construction the helper never sees such a failure; a runtime check would only mask a future violation of the seam.

**Rationale**: `PartialOutput()` is bool; `false` for every `*Failure` constructed by `PreStreamFailure` (line 567-574 doc comment: *"PreStreamFailure takes no output-flag parameter"*). The helper runs BEFORE `PreStreamFailure`'s `delivery` field is even set to `DeliveryPreStream` for any non-handover case — `mapResponse` constructs a `*ai.Failure` with `Delivery() == DeliveryPreStream` and `PartialOutput() == false`, returns it to the helper, the helper returns it to `Stream`. The "no retry after partial output" assertion (R-AIS-043 / S-4) is a **seam-boundary assertion** (`httpClient.Do` count == 1 after a mid-stream retryable failure), not a `PartialOutput()` value check.

### D7 — Attempt-report cause type (`AttemptReport`)

**Choice**: The helper attaches a sibling cause type `*retry.AttemptReport` to the LAST wire failure it observed before exhausting the attempt budget. The chain shape (mirrors `RateLimitTelemetry` at `retry_metadata.go:37-58`):

```
*AttemptReport { Attempts: N+1, FinalCause: *ai.Failure }
  └── (Unwrap) → *ai.Failure  (top-level failure, reachable via errors.As)
                  └── (Unwrap) → *RateLimitTelemetry (if any)
                  └── (Unwrap) → capturedBody (if any)
```

**Alternative considered**: A separate `*Attempt` value reachable only via `errors.As` over a fresh `*ai.Failure`. Rejected — duplicates the cause chain; `errors.As` over a `*ai.Failure` already traverses its cause chain through `Unwrap` (`provider_failure.go:359-364`), so attaching `AttemptReport` as a wrapper is one fewer indirection.

**Rationale**: `RateLimitTelemetry` is the existing precedent for "adapter-local cause chain extension reachable via `errors.As`" (`retry_metadata.go:28-49`). The `AttemptReport` pattern is identical: a typed value with `Error()` (fixed text), `Unwrap()` (the wrapped cause), and exposed fields (`Attempts int`, `FinalCause error`). Consumers reach both via `errors.As`. R-AIS-041 / S-4 ("attempt count and final cause both reachable from the returned error chain") is satisfied structurally.

### D8 — Conformance capability `CapRetry` (CAP-O-04)

**Choice**: Add `CapRetry` to the `Capability` enum in `agenttest/conformance_suite.go:43-89` as the 9th member, between `CapCacheBoundary` (line 75) and `CapNone` (line 85). `capabilityFirst` ... `capabilityEnd` range grows by one; `Capabilities()` enumerator returns 9 values; `capabilityNames` mapping adds `"CAP-O-04(retry)"`; `Optional()` switch extends to include `CapRetry`; `CapabilityRecord.entries` grows from `[8]` to `[9]` (`conformance_record.go:119`); `R-CNF-017` totality rebuild is mechanical (one new line in the totality loop). `standingOf(CapRetry)` returns `StandingOptional` per `R-CNF-003` / AI-03 § 11.

**Alternative considered**: Hiding the retry discipline under `CapTypedFailures` (R-CNF-010). Rejected — the retry contract is *separable* from the failure-mapping contract; a future "no-retry" adapter could pass all `CapTypedFailures` cases while silently skipping the retry discipline `V-FAIL-15` makes binding. Per the proposal (§ 5): "(2) without `CapRetry`, a future 'no-retry' adapter could claim conformance under `CapTypedFailures` only — silently skipping the retry discipline that `V-FAIL-15` makes binding; (3) AI-38 (OpenRouter conformance roll-up) is the natural home for the `CapRetry` case."

**Rationale**: The OpenRouter wrapper inherits the helper transparently (`openrouter/wrapper.go` embeds `*openaicompat.Client` and delegates `Stream` verbatim) — its conformance bridge exercises the helper's behaviour without rewriting assertions. The conformance case registers under `CapRetry`; if a factory declares `Retry = false` (the fake's posture), the case is skipped via `R-CNF-004` (reported, never silent); if `Retry = true` (the real producer's posture), the case runs.

### D9 — `Factory.Retry *bool` field (nil-fails-construction)

**Choice**: Add `Retry *bool` to the `Factory` struct (`agenttest/conformance_suite.go:160-184`), the fourth optional-capability declaration field alongside `Reasoning`, `TokenCounting`, `CacheBoundary` (line 178). `nil` fails construction per `R-CNF-002` / `S-CNF-006` (the project's `nil`-disallowed idiom; precedent: the `bool → *bool` correction at archive 2026-08-03, `ai-provider-conformance-suite/spec.md:323-327`).

**Alternative considered**: Bare `bool` (rejected at AI-23.6 archive; bare bool cannot distinguish "declared not offered" from "never declared"). Re-promoting to bare bool would silently reintroduce the defect `sdd-tasks` caught before AI-23.6 apply.

**Rationale**: The shape (`*bool`) and the nil-fail rule are pinned by the spec (`specs/ai-provider-conformance-suite/spec.md:28`). The field's name (`Retry`) matches the capability name (`CapRetry`), paralleling `Reasoning` / `CapReasoningContent`. Construction-time check at `factoryDefect` (`conformance_suite.go:266-280`) adds one more nil-check branch; `applyDeclaredAbsences` (`conformance_suite.go:330-336`) marks `CapRetry = absent` for non-nil `false` declarations; `declaredOffered` (`conformance_suite.go:301-312`) extends its switch to include `Retry`.

### D10 — Composed-bound ceiling (`defaultMaxAttempts = 3`)

**Choice**: The helper's `doc.go` package doc comment names `defaultMaxAttempts = 3` as the wire-request count per logical call (`N+1 = 4` wire requests when retries are exhausted). The same constant and formula ("harness attempts × Layer 1 attempts") appear in AG-15.2's composed-bound test (doc 0003 line 718). A reader from either Layer 1 (`internal/retry/doc.go`) or Layer 2 (AG-15.2's test) finds the same number with the same formula.

**Alternative considered**: Inlining the constant in `retry.go`. Rejected — the package doc comment is the canonical reader-visible location; the constant lives there, exported, so both layers' readers cite it from the same source.

**Rationale**: R-AIS-044 / S-1 and S-2 (composed-bound ceiling, cross-layer) is binding **as documentation**, not as a runtime check. The constant value (`3`) is package-private to the Layer 1 contract; promotion to `openaicompat.Config.MaxRetries` is out of scope (Q3 precedent from AI-34 — stay constant until a workload demands configurability). The forward guard stays green; no new top-level Go dependency.

---

## 4. File Layout (concrete)

| File | Status | Purpose | Approx. lines | Imports |
| --- | --- | --- | --- | --- |
| `backend/agent/src/ai/internal/retry/retry.go` | NEW | Helper: `Loop`, `Config`, `AttemptReport`, `DefaultMaxAttempts`; package-private `NowFunc` / `SleepFunc` / `RetryAfterReader` types | ~140 | stdlib (`context`, `errors`, `math/rand/v2`, `time`) + `github.com/cachicamas/backend/agent/src/ai` (for `*ai.Failure`) |
| `backend/agent/src/ai/internal/retry/doc.go` | NEW | Package doc comment naming `defaultMaxAttempts = 3`, the composed-bound formula, AG-15.2 citation | ~25 | (none) |
| `backend/agent/src/ai/openaicompat/stream.go` | MODIFIED | Wrap `Do + mapResponse` (lines 218-227) in `retry.Loop`; one import added (`internal/retry`); ~8-line edit inside `Stream` | +8 / -2 | adds `github.com/cachicamas/backend/agent/src/ai/internal/retry` |
| `backend/agent/src/agenttest/conformance_suite.go` | MODIFIED | Add `CapRetry` enum member; `Capabilities()` extension (no edit needed — mechanical); `Optional()` switch case; `capabilityNames` map entry; `Factory.Retry *bool` field; `factoryDefect` nil-check; `declaredOffered` switch case; `applyDeclaredAbsences` no edit (mechanical) | +12 / -0 | unchanged |
| `backend/agent/src/agenttest/conformance_record.go` | MODIFIED | `CapabilityRecord.entries [8]` → `[9]`; `newCapabilityRecord` loop unchanged (uses `Capabilities()`) | +1 / -1 | unchanged |
| `backend/agent/src/agenttest/conformance_retry.go` | NEW | Conformance case body for `CapRetry`; registers under `retry/auto_retry_up_to_documented_bound`; 7 scenario sub-tests (`S-CNF-069..075`) using `httptest.NewServer` + scriptable transport | ~250 | stdlib + `github.com/cachicamas/backend/agent/src/ai` + `github.com/cachicamas/backend/agent/src/ai/openaicompat` + `github.com/cachicamas/backend/agent/src/agenttest/...` (test-only deps in agenttest, allowed per package doc) |
| `backend/agent/src/ai/openaicompat/a_i-35_1_test.go` | NEW | AI-35.1 predicate tests: retryable-pre-stream-retries, terminal-category-never-retries, exhausted-budget-returns-last-failure-wrapped (3 scenarios from R-AIS-041 / S-1, S-2, S-4) | ~180 | stdlib + `github.com/cachicamas/backend/agent/src/ai` + `github.com/cachicamas/backend/agent/src/ai/internal/retry` + `github.com/cachicamas/backend/agent/src/agenttest` |
| `backend/agent/src/ai/openaicompat/a_i-35_2_test.go` | NEW | AI-35.2 backoff mechanics tests: retry-after precedence, ctx-aware waits, seeded jitter, bounded attempt count (4 scenarios from R-AIS-042) | ~200 | stdlib + `github.com/cachicamas/backend/agent/src/ai` + `github.com/cachicamas/backend/agent/src/ai/internal/retry` + `github.com/cachicamas/backend/agent/src/agenttest` |
| `backend/agent/src/ai/openaicompat/a_i-35_3_test.go` | NEW | AI-35.3 replayability + boundary marker: byte-identical replay, attempt count + final cause via `errors.As`, per-attempt drain, partial-output boundary marker (4 scenarios from R-AIS-043) | ~220 | stdlib + `github.com/cachicamas/backend/agent/src/ai` + `github.com/cachicamas/backend/agent/src/ai/internal/retry` + `github.com/cachicamas/backend/agent/src/agenttest` |
| `backend/agent/src/ai/openaicompat/a_i-35_internal_test.go` | NEW (optional) | Helper file for shared test infrastructure (`ai35ScriptableTransport`, `ai35CountDoCalls`, `ai35RecordBodies`) used by all three `a_i-35_*_test.go` files; mirrors `a_i-33_5a_test.go:107-208` precedent | ~80 | stdlib only |
| `openspec/changes/cachicamas-ai-retry-policy/design.md` | NEW (this file) | The design artifact | — | — |

**Aggregate forecast**: 4 new files in `openaicompat/` (`a_i-35_{1,2,3}_test.go` + optional shared helper) = ~680 lines; 1 new file in `internal/retry/` = ~165 lines (`retry.go` + `doc.go`); 2 modified files (`stream.go` +8/-2, `conformance_suite.go` +12/-0, `conformance_record.go` +1/-1) = ~24 lines net; 1 new conformance case file = ~250 lines. **Total: ~1100 lines** (slightly over the 1000-line budget; `sdd-tasks` may recommend splitting into two PRs or trimming test verbosity). The orchestrator's 1000-line budget was set with `size:exception` latitude per the proposal's Risk 7.6; if the aggregate exceeds, `sdd-tasks` flags it and the orchestrator decides.

---

## 5. Type Signatures (`internal/retry/`)

The following signatures are exact Go syntax. The apply phase lifts them verbatim into `retry.go` (or adjusts names per its own commit-shape discipline).

```go
// Package retry implements the Layer 1 pre-stream retry helper for
// backend/agent adapters. See doc.go for the composed-bound ceiling and
// cross-layer citation.
package retry

import (
    "context"
    "math/rand/v2"
    "time"

    "github.com/cachicamas/backend/agent/src/ai"
)

// DefaultMaxAttempts is the Layer 1 retry budget: the maximum number of
// wire requests the helper issues per logical call when retries are
// exhausted. With DefaultMaxAttempts = 3, the helper issues at most N+1 = 4
// wire requests per call.
//
// This constant is the cross-layer cited value: AG-15.2's composed-bound
// test (doc 0003 line 718) reads it verbatim from this package's doc
// comment to assert "harness attempts × Layer 1 attempts".
//
// R-AIS-044 pins the visibility: exported so Layer 2 conformance and
// harness readers can cite it from outside the package, but the helper's
// signature does NOT expose a Config field that overrides it. Tests pass
// a smaller Config.MaxAttempts to verify the bound is enforced.
const DefaultMaxAttempts = 3

// NowFunc is the timing-injection seam the helper exposes for
// deterministic tests (R-AIS-042 / S-2's seeded jitter assertion). The
// default value is time.Now.
type NowFunc func() time.Time

// SleepFunc is the backoff-injection seam. The default value is
// time-based select on ctx.Done() and time.NewTimer(d).C — every backoff
// waits on the caller-owned context, aborts on cancellation, and returns
// ctx.Err() if cancelled mid-wait (the helper then returns the last
// typed failure, NEVER ctx.Err() itself — R-AIS-042 / S-3).
type SleepFunc func(ctx context.Context, d time.Duration) error

// RetryAfterReader pulls a Retry-After hint out of a typed failure. The
// default implementation reads failure.RetryAfter() (provider_failure.go:
// 434-439); tests substitute a fixed reader for deterministic backoff
// assertion.
type RetryAfterReader func(*ai.Failure) (time.Duration, bool)

// Config is the retry-budget parameter the helper's caller supplies.
// Every field has a sensible zero value so a test can construct one with
// the budget alone.
//
//   MaxAttempts = 0  → defaults to DefaultMaxAttempts
//   NowFunc      = nil → defaults to time.Now
//   SleepFunc    = nil → defaults to ctx-aware timer-based sleep
//   RetryAfterReader = nil → defaults to ai.Failure.RetryAfter
//   BaseDelay    = 0  → defaults to 100ms
//   MaxDelay     = 0  → defaults to 30s
//   JitterSeed   = 0  → derived from NowFunc() at first attempt
type Config struct {
    MaxAttempts      int
    NowFunc          NowFunc
    SleepFunc        SleepFunc
    RetryAfterReader RetryAfterReader
    BaseDelay        time.Duration
    MaxDelay         time.Duration
    JitterSeed       int64
}

// AttemptReport is the typed cause this helper attaches to the LAST wire
// failure it observed before exhausting its attempt budget. The chain
// shape (analogous to RateLimitTelemetry at retry_metadata.go:37-58) is:
//
//   *AttemptReport { Attempts: N+1, FinalCause: *ai.Failure }
//      └── (Unwrap) → *ai.Failure  (reachable via errors.As)
//
// Consumers reach the attempt count and the final cause through errors.As:
//
//   var report *retry.AttemptReport
//   if errors.As(err, &report) { report.Attempts }
//
//   var failure *ai.Failure
//   if errors.As(err, &failure) { failure.Category() }
//
// AttemptReport is attached ONLY when the helper EXHAUSTED its budget;
// single-attempt terminal failures (401 / 400 / etc.) carry no
// AttemptReport because the attempt count is trivially 1.
type AttemptReport struct {
    Attempts   int   // total wire requests the helper issued (>= 2 on exhaustion)
    FinalCause error // the typed *ai.Failure from the LAST attempt
}

// attemptReportText is AttemptReport's fixed Error() text. It never
// reflects the wrapped cause (mirrors RateLimitTelemetry's posture at
// retry_metadata.go:13-15).
const attemptReportText = "retry: attempt budget exhausted"

func (r *AttemptReport) Error() string { return attemptReportText }

// Unwrap exposes FinalCause to errors.Is / errors.As. The chain reads:
// *AttemptReport → *ai.Failure → ... → capturedBody (status path), or
// *AttemptReport → *ai.Failure → (no further cause on transport-error path).
func (r *AttemptReport) Unwrap() error { return r.FinalCause }

// Loop is the helper's one exported entry point. It runs executeOnce
// against body up to cfg.MaxAttempts times, retrying only on
// retryable-flagged pre-stream failures. The helper runs BEFORE the
// carrier handover (stream.go:237), so no semantic event has been
// emitted and PartialOutput() is unconditionally false for every
// failure the helper observes (V-FAIL-15 by construction).
//
// The executeOnce closure is the per-attempt operation the adapter
// performs. For openaicompat, it is:
//
//   func(ctx context.Context, body []byte) (*http.Response, error) {
//       httpReq, err := c.newRequest(ctx, body, "chat", "completions")
//       if err != nil { return nil, preStreamTransportFailure(ctx, err) }
//       resp, err := c.httpClient.Do(httpReq)
//       if err != nil { return nil, preStreamTransportFailure(ctx, err) }
//       if failure := mapResponse(resp, time.Now()); failure != nil {
//           return nil, failure
//       }
//       return resp, nil
//   }
//
// The closure re-builds the request per attempt so bytes.NewReader wraps
// a fresh reader over the same body each time (request.go:28).
//
// Returns:
//   - (resp, nil)            — 2xx response; caller takes over with isStreamContentType + carrier handover
//   - (nil, *ai.Failure)     — typed pre-stream failure; caller returns directly
//   - (nil, *AttemptReport)  — exhaustion: typed failure wrapped with attempt count + final cause
//   - (nil, lastFailure)     — ctx cancelled mid-backoff; the helper aborts the sleep and returns the LAST typed failure (NEVER ctx.Err() — R-AIS-042 / S-3)
func Loop(
    ctx context.Context,
    body []byte,
    cfg Config,
    executeOnce func(ctx context.Context, body []byte) (*http.Response, error),
) (*http.Response, error)
```

**Notes on the signatures**:

- `math/rand/v2` is the Go 1.22+ seeded-jitter source (the project's Go 1.26.3 stack supports it; the import boundary guard's stdlib filter admits it). The pre-1.22 `math/rand` is deprecated for non-cryptographic use as of Go 1.20.
- `Config` is a value type (not pointer) — Go's zero-value-defaultable idiom mirrors `FailureReport` (`provider_failure.go:277-310`).
- `Loop`'s `executeOnce` is a closure, not an interface — same posture as `emit`'s `out chan<- ai.Event` (`stream.go:588`): a function-type parameter avoids widening an interface for one caller.
- `AttemptReport.Unwrap()` returns `FinalCause` (an `error`), so `errors.As` over a returned `*AttemptReport` finds both itself and the wrapped `*ai.Failure` via the chain.

---

## 6. Algorithm Walk-through (`Loop`)

The algorithm in Go-flavoured pseudocode (the apply phase lifts it verbatim):

```go
func Loop(ctx context.Context, body []byte, cfg Config, executeOnce func(ctx, body) (*http.Response, error)) (*http.Response, error) {
    cfg = applyDefaults(cfg)

    // Pre-loop context check (R-AIS-041 condition 4): caller-owned context
    // already cancelled before the first attempt — abort immediately.
    if err := ctx.Err(); err != nil {
        // Return a synthetic failure wrapping ctx.Err()? No — the caller
        // is responsible for ctx.Err() handling; the helper returns the
        // last failure (here, none) only if a wire request has happened.
        // Pre-loop cancellation: return ctx.Err() — adapter decides whether
        // to wrap as PreStreamFailure or treat as caller contract violation.
        return nil, err
    }

    var lastFailure *ai.Failure
    jitter := rand.New(rand.NewPCG(uint64(cfg.JitterSeed), uint64(cfg.NowFunc().UnixNano())))

    for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
        resp, err := executeOnce(ctx, body)

        // Success path: 2xx response, fall through to carrier handover.
        if err == nil && resp != nil {
            return resp, nil
        }

        // Classify the outcome. err is either *ai.Failure (status path)
        // or a transport error (no response object).
        var failure *ai.Failure
        var transportErr error
        if errors.As(err, &failure) {
            transportErr = nil
        } else {
            failure = wrapTransportError(err) // builds *ai.Failure with FailureCategoryUnavailable, cause = err
            transportErr = err
        }

        lastFailure = failure
        _ = transportErr // retained for potential telemetry; not surfaced

        // Decision: retry-or-return (R-AIS-041 / S-1, S-2, S-4).
        retryable := failure.Retryable()
        exhausted := attempt == cfg.MaxAttempts

        if !retryable || exhausted {
            if exhausted && retryable {
                // Wrap with attempt-report — the typed failure the caller
                // receives carries both the attempt count and the final
                // cause (R-AIS-041 / S-4, R-AIS-043 / S-2).
                return nil, &AttemptReport{Attempts: attempt, FinalCause: failure}
            }
            // Terminal single-attempt failure: return as-is, no
            // AttemptReport (attempt count is trivially 1).
            return nil, failure
        }

        // Compute next backoff. Retry-After hint overrides computed
        // backoff (R-AIS-042 / S-1); absent hint falls back to bounded
        // exponential + seeded jitter (R-AIS-042 / S-2).
        delay := computeBackoff(attempt, cfg, jitter)
        if d, ok := cfg.RetryAfterReader(failure); ok && d > 0 {
            delay = d
        } else if d == 0 && ok {
            // ai.Delay(0) is "retry immediately" — distinct from absent;
            // treat as zero delay.
            delay = 0
        }

        // Backoff waits on caller-owned context (R-AIS-042 / S-3). If
        // cancelled mid-wait, abort immediately and return the LAST
        // typed failure — NEVER ctx.Err() itself, NEVER an invented
        // retry-exhausted category.
        if sleepErr := cfg.SleepFunc(ctx, delay); sleepErr != nil {
            return nil, lastFailure
        }
    }

    // Unreachable if MaxAttempts >= 1. Defensive: return the last failure.
    if lastFailure != nil {
        return nil, &AttemptReport{Attempts: cfg.MaxAttempts, FinalCause: lastFailure}
    }
    return nil, errors.New("retry: no attempts issued")
}

// computeBackoff returns base * 2^(attempt-1) with seeded jitter ±10% of
// base, capped at cfg.MaxDelay. The growth curve is exponential with a
// documented cap — recorded in doc.go so AG-15.2's harness test can cite
// the same formula.
func computeBackoff(attempt int, cfg Config, jitter *rand.Rand) time.Duration {
    base := cfg.BaseDelay
    for i := 1; i < attempt; i++ {
        base *= 2
        if base >= cfg.MaxDelay {
            base = cfg.MaxDelay
            break
        }
    }
    // ±10% jitter — small enough not to violate the documented bounds,
    // large enough to defeat naive synchronous retry storms.
    jitterNs := int64(base / 10)
    offset := time.Duration(jitter.Int64N(2*jitterNs) - jitterNs)
    return base + offset
}

// wrapTransportError converts a non-*ai.Failure error (the result of
// httpClient.Do returning a transport-level error before any response
// exists) into a typed *ai.Failure with FailureCategoryUnavailable.
// FailureCategoryUnavailable is retryableFor == true, so the helper will
// retry on transport errors — same posture midStreamFailureFrom uses
// (stream_failure.go:196-211).
func wrapTransportError(err error) *ai.Failure {
    failure, _ := ai.PreStreamFailure(ai.FailureReport{
        Category: ai.FailureCategoryUnavailable,
        Cause:    err,
    })
    return failure
}

// applyDefaults substitutes package-level defaults for zero-valued Config
// fields. The defaults are exported as DefaultMaxAttempts; the others are
// package-private (baseDelayDefault = 100ms, maxDelayDefault = 30s).
func applyDefaults(cfg Config) Config {
    if cfg.MaxAttempts <= 0 {
        cfg.MaxAttempts = DefaultMaxAttempts
    }
    if cfg.NowFunc == nil {
        cfg.NowFunc = time.Now
    }
    if cfg.SleepFunc == nil {
        cfg.SleepFunc = defaultSleep
    }
    if cfg.RetryAfterReader == nil {
        cfg.RetryAfterReader = defaultRetryAfterReader
    }
    if cfg.BaseDelay <= 0 {
        cfg.BaseDelay = defaultBaseDelay
    }
    if cfg.MaxDelay <= 0 {
        cfg.MaxDelay = defaultMaxDelay
    }
    return cfg
}

// defaultSleep is the package-private default SleepFunc — ctx-aware
// timer-based select, aborts on cancellation with ctx.Err().
func defaultSleep(ctx context.Context, d time.Duration) error {
    if d <= 0 {
        return nil
    }
    timer := time.NewTimer(d)
    defer timer.Stop()
    select {
    case <-timer.C:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}

// defaultRetryAfterReader is the package-private default RetryAfterReader
// — wraps ai.Failure.RetryAfter().
func defaultRetryAfterReader(f *ai.Failure) (time.Duration, bool) {
    return f.RetryAfter()
}
```

**Drain discipline (D4)**: not visible in `Loop`'s body — supplied by composition. `executeOnce` calls `mapResponse` (status path) which calls `captureBody` (`failure_map.go:35`) which drains + closes (`capture.go:75-122`). The transport-error path returns before `captureBody` is ever called, but `http.Client.Do`'s transport handles its own connection teardown — no body to drain.

**Cancel-during-sleep discipline**: `defaultSleep` returns `ctx.Err()`; the helper's caller treats that as "abort the wait, return last typed failure". This matches R-AIS-042 / S-3 verbatim: "the typed failure is returned with the last wire response's cause — never with an invented retry-exhausted category".

---

## 7. Integration into `stream.go`

The change to `stream.go` is a **single block replacement** at lines 218-227, plus one import added. The orchestrator's prompt asks whether the helper receives `body []byte` and re-builds `httpReq` per attempt, OR receives `httpReq` and re-uses it. **Decision**: receives `body []byte` (D3); the `executeOnce` closure re-builds `httpReq` per attempt.

### 7.1 Before (current `stream.go:218-227`)

```go
resp, err := c.httpClient.Do(httpReq)
if err != nil {
    return nil, preStreamTransportFailure(ctx, err)
}

// R-ATS-024: a failure status routes to AI-32's own failure mapping
// before any byte reaches the decoder. mapResponse returns nil for a
// 2xx response without touching resp.Body, so a success response falls
// through to the content-type check below untouched.
if failure := mapResponse(resp, time.Now()); failure != nil {
    return nil, failure
}
```

### 7.2 After (post-AI-35 — 8-line edit)

```go
resp, err := retry.Loop(ctx, body, retry.Config{
    // MaxAttempts defaults to retry.DefaultMaxAttempts = 3; NowFunc,
    // SleepFunc, RetryAfterReader default to package-private defaults
    // (time.Now / ctx-aware timer / ai.Failure.RetryAfter).
    // Tests pass explicit Config for deterministic backoff.
}, c.executeOnceOnce /* the helper closure, see 7.3 */)
if err != nil {
    return nil, err
}

// R-ATS-024: a 2xx response falls through to the content-type check.
// (Retry exhausted returns the wrapped *AttemptReport; single-attempt
//  terminal returns the *ai.Failure directly. Both are typed errors
//  the caller hands up — no carrier returned, no goroutine spawned.)
```

### 7.3 The `executeOnceOnce` closure (defined on `*Client`)

The closure is a method on `*Client` so it can access `c.newRequest`, `c.httpClient.Do`, `c.executeOnceOnce` is not its own method — it's an inline `func(ctx, body) (*http.Response, error)` defined as a package-private variable:

```go
// executeOnce is the Stream-line retry helper's per-attempt operation
// closure. It performs the adapter's HTTP round-trip and wire-side
// status mapping — exactly the pair R-AIS-024 already requires before
// any byte reaches the decoder. The closure is reused across attempts
// by retry.Loop; each invocation re-builds httpReq from the same body
// slice so bytes.NewReader wraps a fresh reader per attempt
// (request.go:28, byte-identical replay — R-AIS-043 / S-1).
func (c *Client) executeOnce(ctx context.Context, body []byte) (*http.Response, error) {
    httpReq, err := c.newRequest(ctx, body, "chat", "completions")
    if err != nil {
        return nil, preStreamTransportFailure(ctx, err)
    }
    resp, err := c.httpClient.Do(httpReq)
    if err != nil {
        return nil, preStreamTransportFailure(ctx, err)
    }
    if failure := mapResponse(resp, time.Now()); failure != nil {
        return nil, failure
    }
    return resp, nil
}
```

### 7.4 Carrier handover ordering (R-ATS-002 / § 7 boundary)

The helper's return is the carrier handover point. `Stream`'s flow after `retry.Loop` returns:

```go
resp, err := retry.Loop(ctx, body, retry.Config{}, c.executeOnce)
if err != nil {
    return nil, err  // typed *ai.Failure or *AttemptReport; no carrier returned
}

// R-ATS-023: 2xx response whose Content-Type is not the streaming
// media type is refused before any byte reaches the decoder.
if !isStreamContentType(resp.Header.Get("Content-Type")) {
    return nil, refuseNonStreamContentType(resp)
}

out := make(chan ai.Event, streamCarrierBuffer)  // R-AIS-031, R-AIS-039
go run(ctx, resp, out)                          // R-AIS-003 single-producer model
return out, nil
```

The carrier handover (`make(chan ai.Event, ...)` → `go run(...)`) is **after** the helper returns. The helper never sees a carrier; the producer never sees a typed `*ai.Failure` returned from the helper. The seam holds by construction.

### 7.5 Imports delta

```go
import (
    // existing imports: context, fmt, io, mime, net/http, time
    // existing: github.com/cachicamas/backend/agent/src/ai

    // NEW:
    "github.com/cachicamas/backend/agent/src/ai/internal/retry"
)
```

Forward guard stays green: `internal/retry/` is within `github.com/cachicamas/backend/agent/...` (`import_boundary_test.go:103-105` allowlist).

---

## 8. Conformance Case Body Sketch (`agenttest/conformance_retry.go`)

### 8.1 Function signature

```go
// AI-35 — CapRetry conformance case body. Registers under CapRetry
// (CAP-O-04) and asserts S-CNF-069..075 against a scripted HTTP
// transport. See doc comments per scenario for what the scripted
// transport serves and what the case asserts.
package agenttest

import (
    "context"
    "net/http"
    "net/http/httptest"
    "sync/atomic"
    "testing"

    "github.com/cachicamas/backend/agent/src/ai"
    "github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

func init() {
    registerConformanceCase(
        "retry/auto_retry_up_to_documented_bound",
        CapRetry,
        retryAutoRetryUpToBoundCase,
    )
}

// retryAutoRetryUpToBoundCase is the conformance body for R-CNF-019
// (S-CNF-069..075). It stands up a scripted httptest.NewServer and a
// real *openaicompat.Client configured to talk to it; the case body
// constructs the subject directly because the conformance suite's
// Factory pattern is designed for script-based fake providers (which
// cannot speak HTTP-level retry behavior). Adapters that wrap
// *openaicompat.Client (e.g. openrouter/wrapper.go) inherit the
// helper transparently and exercise the same case body via their own
// conformance bridge.
//
// The case body DOES NOT depend on the Factory parameter f. f is
// consulted only for the suite's mechanism (skipped on Retry = false
// per R-CNF-004); the case body's subject is constructed directly.
//
// DEPENDENCY NOTE: this case body adds openaicompat as a dependency
// of agenttest, where the package's other conformance cases are
// adapter-agnostic. The dependency is acceptable per agenttest/doc.go
// ("imports only the standard library and this module's own src/ai")
// because openaicompat lives under src/ai/openaicompat. Forward guard
// (import_boundary_test.go:103-105) admits the path; no third-party
// import is introduced. The apply phase updates agenttest/doc.go to
// note the new conformance-case dependency.
func retryAutoRetryUpToBoundCase(t *testing.T, f Factory) {
    t.Helper()

    t.Run("retryable_pre_stream_retried_up_to_bound", func(t *testing.T) {
        // S-CNF-069: scripted transport returns 429 with Retry-After: 0
        // repeatedly for N+1 attempts then a 2xx text-event-stream.
        // Asserts: N+1 wire requests; final stream is the 2xx; on
        // exhaustion (if 2xx never arrives) the typed failure carries
        // the count via the cause chain.
        // ...
    })

    t.Run("terminal_category_never_retried", func(t *testing.T) {
        // S-CNF-070: scripted 401 once. Asserts: exactly 1 wire
        // request; typed failure with Retryable() == false.
        // ...
    })

    t.Run("partial_output_boundary_no_retry", func(t *testing.T) {
        // S-CNF-071: scripted transport emits one 2xx text-event-
        // stream frame then a retryable-flagged failure mid-stream.
        // Asserts: httpClient.Do count == 1; typed failure reaches
        // consumer via terminal error event with PartialOutput() == true.
        // ...
    })

    t.Run("byte_identical_replay", func(t *testing.T) {
        // S-CNF-072: scripted transport records every request body's
        // bytes; helper retries N+1 times. Asserts: every recorded
        // body byte-identical to every other.
        // ...
    })

    t.Run("attempt_count_and_final_cause_in_chain", func(t *testing.T) {
        // S-CNF-073: helper exhausts attempt budget. Asserts:
        // errors.As over the returned error reaches *retry.AttemptReport
        // with Attempts == N+1 and *ai.Failure with the original
        // category + delivery.
        // ...
    })

    t.Run("cap_retry_absent_reported_not_silent", func(t *testing.T) {
        // S-CNF-074: Factory with Retry = false (declared not
        // offered). Asserts: record entry CapRetry = absent; suite
        // not inconclusive on that entry; R-CNF-001..018 still
        // satisfied. (This sub-test is enforced by the suite's
        // existing R-CNF-004 mechanism — the case body just
        // documents the expected behavior.)
        // ...
    })

    t.Run("factory_nil_retry_defect", func(t *testing.T) {
        // S-CNF-075: Factory with Retry = nil (undeclared). Asserts:
        // construction fails naming the undeclared capability per
        // R-CNF-002 / S-CNF-006. (Also enforced by suite's
        // existing factoryDefect mechanism; case body documents.)
        // ...
    })
}
```

### 8.2 Scenario-to-scripted-transcript mapping

| Scenario | Scripted HTTP responses | Assertion |
| --- | --- | --- |
| S-CNF-069 retryable-pre-stream-retried-up-to-bound | N+1 × `429 Retry-After: 0` + rate-limit JSON body, then `200 text/event-stream` | `httpClient.Do` count == N+1 (instrumented transport); final stream is the `2xx`; on the alternative exhaustion shape (no 2xx), typed failure has `Attempts == N+1` via `errors.As(*AttemptReport)` |
| S-CNF-070 terminal-category-never-retried | `401` + invalid_api_key body | `httpClient.Do` count == 1; returned error is `*ai.Failure` with `Retryable() == false`; no `*AttemptReport` |
| S-CNF-071 partial-output-boundary-no-retry | `200 text/event-stream` with one text delta, then `429 Retry-After: 0` mid-stream | `httpClient.Do` count == 1; carrier handover happened; terminal error event has `PartialOutput() == true` |
| S-CNF-072 byte-identical-replay | Records every request body, returns `429 Retry-After: 0` for N+1 attempts then `200` | Every recorded request body byte-identical to every other; bodies match the original request's bytes |
| S-CNF-073 attempt-count-and-final-cause-in-chain | `429 Retry-After: 0` for N+1 attempts then `500` | `errors.As(err, &report)` returns true with `Attempts == N+1`; `errors.As(err, &failure)` returns true with `Category() == FailureCategoryRateLimit` |
| S-CNF-074 CapRetry-absent | (suite mechanism; no scripted transport) | Factory with `Retry = false` produces record entry `CapRetry = absent`; `R-CNF-001..018` still pass |
| S-CNF-075 Factory-nil-defect | (suite mechanism; no scripted transport) | Factory with `Retry = nil` fails construction naming `CapRetry` |

### 8.3 Scriptable transport

The case body uses `httptest.NewServer(http.HandlerFunc(...))` with an instrumented handler that:
- Records each inbound request's body bytes (for `S-CNF-072`).
- Returns scripted status + headers + body per request number (for `S-CNF-069..071`, `S-CNF-073`).
- Counts inbound requests (for S-CNF-069, S-CNF-070, S-CNF-071).

This pattern is the same one `openrouter/conformance/bridge_test.go:107-158` uses for its own `bridgeAttributionRoundTripper` + `bridgeServeTranscripts`. No new test helper is required — the case body's transport is local to the file.

### 8.4 OpenRouter conformance extension

For OpenRouter, the openrouter conformance bridge (`openrouter/conformance/bridge_test.go`) extends the case body (or mirrors it) using the OpenRouter wrapper as the subject. Because the wrapper embeds `*openaicompat.Client` and delegates `Stream` verbatim, the wrapper inherits the helper transparently; the bridge's test confirms the same case body assertions pass for the wrapper as for `openaicompat.Client` directly. The OpenRouter extension is **owned by AI-38** (the OpenRouter conformance roll-up, doc 0002 line 2118), NOT by AI-35 — AI-35 lands the case body for `openaicompat`; AI-38 extends it for the wrapper.

---

## 9. Test File Layouts

All three real-producer test files use **internal** `package openaicompat` (matching `a_i-33_*.go` and `a_i-34_*.go` precedent). They share an optional helper file (`a_i-35_internal_test.go`) that mirrors `a_i-33_5a_test.go:107-208`'s shared-helper pattern.

### 9.1 `a_i-35_1_test.go` — AI-35.1 predicate tests

**Subnode**: AI-35.1 (R-AIS-041 / S-1, S-2, S-4 — S-3 moved to AI-35.3 per Flag 2)

**Test functions** (RED-first against the helper's seam):

| Test function | Scenario | Setup | Assertion |
| --- | --- | --- | --- |
| `TestAI35_1_RetryablePreStream_RetriesUpToBound` | R-AIS-041 / S-1 | `httptest.NewServer` returning `429 Retry-After: 0` for `DefaultMaxAttempts` attempts then `200 text/event-stream`; `mustClient(t, server.URL)`; `validRequest(t)` | `httpClient.Do` count == `DefaultMaxAttempts + 1` (instrumented transport); on success path, `Stream` returns a carrier with the `2xx` events; on exhaustion path, returned error has `Attempts == DefaultMaxAttempts + 1` via `errors.As(*retry.AttemptReport)` |
| `TestAI35_1_TerminalCategory_NeverRetries` | R-AIS-041 / S-2 | Same handler pattern; server returns `401` once; retryable `401` (authentication) is NOT retried | `httpClient.Do` count == 1; returned error is `*ai.Failure` with `Retryable() == false` and `Category() == FailureCategoryAuthentication` |
| `TestAI35_1_ExhaustedBudget_ReturnsLastFailureWrappedWithAttemptCount` | R-AIS-041 / S-4 | Server returns `429 Retry-After: 0` for `DefaultMaxAttempts + 1` attempts (no `2xx`); exhaustion path | `httpClient.Do` count == `DefaultMaxAttempts + 1`; `errors.As(err, &report)` returns true with `Attempts == DefaultMaxAttempts + 1`; `errors.As(err, &failure)` returns true with `Category() == FailureCategoryRateLimit`; `failure.Delivery() == DeliveryPreStream`; `failure.PartialOutput() == false` |

**Scenario banners** (matching `a_i-33_5a_test.go:1-67` precedent):

```go
// AI-35.1 — R-AIS-041 / S-1 (Retryable pre-stream retried up to bound)
//
// # Why internal (package openaicompat)
// ... (mirrors a_i-33_5a_test.go doc comments)
//
// # Pins
// R-AIS-041 / S-1 (retryable-pre-stream-retried),
// R-ATS-002 (pre-stream contract),
// R-AEM-008 (retryable taxonomy).
//
// # No t.Parallel() (R-STK-008)
// ...
```

### 9.2 `a_i-35_2_test.go` — AI-35.2 backoff mechanics tests

**Subnode**: AI-35.2 (R-AIS-042 / S-1, S-2, S-3, S-4)

| Test function | Scenario | Setup | Assertion |
| --- | --- | --- | --- |
| `TestAI35_2_RetryAfterHint_OverridesComputedBackoff` | R-AIS-042 / S-1 | Server returns `429 Retry-After: H` with `H > computed` for `DefaultMaxAttempts - 1` attempts then `200` | Backoff between attempts == `H` exactly (not the computed exponential value); hint is read from `failure.RetryAfter()` |
| `TestAI35_2_ComputedBackoff_BoundedWithSeededJitter` | R-AIS-042 / S-2 | Server returns `429` (no `Retry-After` header) for `DefaultMaxAttempts` attempts then `200`; helper configured with explicit `Config{JitterSeed: <fixed>}` | Each delay is in `[base * 2^(attempt-1), base * 2^attempt]` (bounded); a fixed seed produces an assertable sequence of delays; no delay exceeds `MaxDelay` |
| `TestAI35_2_Backoff_CtxCancellationAbortsImmediately` | R-AIS-042 / S-3 | Server returns `429 Retry-After: 60` (long); ctx is cancelled mid-backoff | Helper aborts the sleep on `ctx.Done()`; no subsequent wire request; returned error is the LAST typed failure (rate-limit `*ai.Failure`); NEVER `ctx.Err()` itself; NEVER an invented retry-exhausted category |
| `TestAI35_2_BoundedAttemptCount_ExactlyNPlusOneWireRequests` | R-AIS-042 / S-4 | Server returns `429 Retry-After: 0` for `2 * DefaultMaxAttempts` attempts (always retryable) | `httpClient.Do` count == `DefaultMaxAttempts + 1` exactly; no wire request after the budget is exhausted |

### 9.3 `a_i-35_3_test.go` — AI-35.3 replayability + boundary marker tests

**Subnode**: AI-35.3 (R-AIS-043 / S-1, S-2, S-3, S-4 — includes the moved item 2 from AI-35.1)

| Test function | Scenario | Setup | Assertion |
| --- | --- | --- | --- |
| `TestAI35_3_ByteIdenticalReplay_AcrossAttempts` | R-AIS-043 / S-1 | Server records every request body bytes; returns `429 Retry-After: 0` then `200`; helper retries | Every recorded request body byte-identical to every other; bodies match the original `Translate(req)` output (`bytes.Equal`); no drift, no truncation |
| `TestAI35_3_AttemptCountAndFinalCause_ReachableFromErrorChain` | R-AIS-043 / S-2 | Server returns retryable failures until exhaustion | `errors.As(err, &report)` finds `*retry.AttemptReport` with `Attempts == N+1`; `errors.As(err, &failure)` finds `*ai.Failure` carrying the original category and delivery; chain traversal via `errors.As` reaches both without unwrapping |
| `TestAI35_3_PerAttemptDrain_StatusPath` | R-AIS-043 / S-3 | Server returns `429` with body containing trailing garbage; helper retries; observe `httpClient.Do` count and TCP connection state | `httpClient.Do` count == N+1; connection-reuse count (instrumented via `httptest.Server.Config.ConnState` like `a_i-33_5a_test.go:197`) == 1 across attempts; per-attempt drain supplied by composition with `captureBody` |
| `TestAI35_3_PartialOutputBoundaryMarker_AfterHandoverNoRetry` | R-AIS-043 / S-4 (moved from AI-35.1 per Flag 2) | Server emits one `2xx text/event-stream` frame (text delta), then a retryable-flagged failure mid-stream | `httpClient.Do` count == 1 (seam past: helper no longer in call path); carrier handover happened; terminal error event has `PartialOutput() == true`; no retry |

**Teardown**: AI-35.3's exhaustion tests (S-1, S-2) use `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")` per the precedent at `a_i-33_5b_test.go` and `a_i-34_3_test.go`. S-3 and S-4 follow the same posture.

### 9.4 Shared helper file (`a_i-35_internal_test.go`, optional)

Mirrors `a_i-33_5a_test.go:107-208` and `a_i-34_2_test.go:107-117`:

```go
// ai35ScriptableTransport returns an httptest.NewServer with a handler
// that serves each response in responses[n] to the (n+1)th request, in
// arrival order, with the streaming media type. Each request's body
// bytes are appended to bodies[n] (in arrival order) so the test can
// assert byte-identical replay across attempts.
//
// The transport is a *Server (not a RoundTripper) because the
// real-producer tests exercise the httpClient's own connection-pool
// behaviour; reusing the existing a_i-33_5a_test.go:197
// countTCPConnections helper, the assertion is "1 TCP connection per
// helper retry budget" (drain composition).
func ai35ScriptableTransport(t *testing.T, responses []ai35Response) (*httptest.Server, *ai35Counters)

type ai35Response struct {
    status      int
    headers     http.Header
    body        []byte
}

type ai35Counters struct {
    DoCalls atomic.Int32
    Bodies  [][]byte
}
```

The helper is local to `a_i-35_*_test.go` (same package) — no production Go. The apply phase may decide to inline the helper in `a_i-35_3_test.go` (only S-1, S-3 use it) instead of a shared file; either way the aggregate line count is roughly the same.

---

## 10. Risks (carried from spec, refined)

| # | Risk | Severity | Design-level mitigation | Status |
| --- | --- | --- | --- | --- |
| 7.1 | Helper location trade-off: `internal/retry/` adds one cross-package edge today for forward-adapter reuse that doesn't yet exist | Low | The doc 0002 amendment (lines 2085-2086) explicitly anticipates future-adapter reuse; the edge is small (one import in `stream.go`) and the alternative (`openaicompat/retry.go`) would force future adapters to import `openaicompat` — coupling the WRONG way. Reversibility: easy to move to `openaicompat/retry.go` later if v1-only-caller holds; HARD to move the other direction after a future adapter has imported `openaicompat`. | unchanged from spec |
| 7.2 | Tooling fragility (per Engram #2636/#2639) — runtime ledger `complete` blocks subsequent `sdd-attempt acquire` after `passed` settle; follow-up archive may need manual | Med | Anticipate at apply time; have AI-34 manual-archive pattern ready (cherry-pick doc 0002 to branch per Engram #2638). The AI-35 close follows the same workflow. | unchanged from spec |
| 7.3 | Mid-stream scope pinning — helper's "pre-stream only" contract must be in the TYPED shape, not a runtime comment | Med | The helper's `executeOnce` closure signature is pre-stream-only by construction: it returns either a 2xx `*http.Response` (carrier handover happens AFTER the helper returns) or a typed `*ai.Failure` from `mapResponse` / `preStreamTransportFailure` (both pre-stream by construction). The carrier handover (`make(chan ai.Event, ...)` at `stream.go:237`) is **after** the helper returns. AI-35.3 / S-4 (`httpClient.Do` count == 1 after partial-output) pins the seam by instrumented assertion, not by a runtime check inside the helper. | unchanged from spec |
| 7.4 | `CapRetry` conformance membership could be deferred forever without enforcement | Low | YES recommendation committed at proposal § 5; AI-38 is the natural owner; `[8]`-element → `[9]`-element capability array, `R-CNF-017` totality rebuild is mechanical (one new line). `Factory.Retry *bool` mirrors `Reasoning` / `TokenCounting` / `CacheBoundary`. | unchanged from spec |
| 7.5 | Composed-bound ceiling (`harness × Layer 1`) might be misunderstood if documented only in helper package comment | Low | Constant lives in `internal/retry/doc.go` (package doc comment) + AG-15.2's test item 2 (doc 0003 line 718) quotes it verbatim from the doc — readers from either layer find the same number with the same formula. `defaultMaxAttempts = 3` is exported and accessible from outside the package. | unchanged from spec |
| 7.6 | Aggregate ~1100 lines exceeds 1000-line PR budget | Med | The orchestrator raised the budget to 1000 lines per doc 0002 amendment line 2093. The forecast is ~1100 (slightly over); `sdd-tasks` flags and the orchestrator decides. Options: (a) trim test verbosity in `a_i-35_*_test.go` files (likely reduces to ~900 total); (b) split into two chained PRs per `chained-pr` precedent. Apply phase re-forecasts after tasks phase. | unchanged from spec |

### Design-level risks (new)

| # | Risk | Severity | Mitigation |
| --- | --- | --- | --- |
| D-R1 | `internal/retry/doc.go` is the canonical reader-visible location for the composed-bound ceiling; a refactor that moves the constant away from the doc comment breaks R-AIS-044 / S-1 | Med | The apply phase pins the constant's location in a `// Package retry ...` doc comment that opens with the constant declaration verbatim. Any future refactor that moves the constant MUST update the doc comment in the same commit (recorded in this file's design contract). |
| D-R2 | `math/rand/v2` import — the seeded-jitter source is Go 1.22+. The project's stack is Go 1.26.3, so v2 is available. If a future downgrade happens, `math/rand/v2` may not exist | Low | The import is in `retry.go`'s own imports block; a downgrade to Go < 1.22 would fail the import path. The forward guard catches this at `go build` time. |
| D-R3 | The `executeOnceOnce` closure name (D2/D7 integration) is awkward — the apply phase may pick a different name | Low | The closure is package-private to `openaicompat`; naming is the apply phase's choice. The orchestrator's prompt mentioned `c.executeOnce` as a working name; the apply phase picks the final name. |
| D-R4 | Conformance case body uses `httptest.NewServer` + `*openaicompat.Client`, adding `openaicompat` as a dependency of `agenttest`. Breaks the adapter-agnostic posture the other conformance cases maintain | Med | The dependency is acceptable per `agenttest/doc.go` ("imports only the standard library and this module's own src/ai") — `openaicompat` is under `src/ai/openaicompat`. The apply phase updates `agenttest/doc.go` to note the new conformance-case dependency. The forward guard stays green. Alternative considered: a `RoundTripper`-based scripting without `openaicompat` dependency — rejected because the Factory pattern doesn't expose RoundTripper injection, and the case body needs the `Config.HTTPClient` field to script the transport. |
| D-R5 | `errors.As(err, &report)` traverses the cause chain through `Unwrap`. If a future failure carries an unrelated `Unwrap` chain (e.g. a vendor-specific wrapper), `errors.As` may find the `*AttemptReport` at an unexpected position | Low | `AttemptReport.Unwrap()` returns `FinalCause` (the `*ai.Failure`); `errors.As` finds both `*AttemptReport` and the wrapped `*ai.Failure` via the chain. The semantic identity (attempt count + final cause) is unambiguous. No false-positive risk. |

---

## 11. Rollback

Single-PR rollback per the proposal's § 10 (confirmed here):

1. **Helper file** (`internal/retry/retry.go` + `doc.go`): self-contained; reverting it removes auto-retry (Layer 1 returns the first failure as today, restoring AI-32's pre-AI-35 behaviour). The helper's import in `stream.go` is removed in the same revert; `c.executeOnce` is inlined back into `Stream`.

2. **`stream.go` edit**: revert by removing the `retry.Loop` call and restoring the original `Do + mapResponse` pair (lines 218-227 verbatim from the pre-AI-35 commit).

3. **Conformance amendment** (`conformance_suite.go` + `conformance_record.go`): revert by removing `CapRetry` enum member, `Optional()` case, `capabilityNames` entry, `Factory.Retry` field, nil-fail construction; `CapabilityRecord.entries [9]` → `[8]`; `R-CNF-017` totality rebuild reverses to `[8]`.

4. **Conformance case file** (`conformance_retry.go`): drops with the conformance amendment revert. The new `*bool` `Retry` field is gone; no case references it.

5. **Three real-producer test files** (`a_i-35_{1,2,3}_test.go`): drop with the code change. The helper's signature is gone; the tests no longer compile.

6. **Spec deltas** (`specs/ai-stream-lifecycle/spec.md`, `specs/ai-provider-conformance-suite/spec.md`): reversible without code change. The canonical specs (`openspec/specs/...`) keep their pre-AI-35 wording.

7. **`decision.md`** (AI-35.0 closing artefact): preserved in `openspec/changes/cachicamas-ai-retry-policy/` as the historical record — no rollback action.

If the helper's behaviour turns out to be wrong in production (e.g., the retry-after interpretation diverges from a vendor's wire), the rollback is to set `defaultMaxAttempts = 0` (no retries) and reopen the spec amendment — exactly the spec's living-graph clause.

---

## 12. Acceptance Criteria (the spec's checklist, refined)

Each spec requirement maps to a verification step at apply time. The apply phase lifts this list into `tasks.md`.

### 12.1 R-AIS-041 — Pre-stream retry predicate

- [ ] **`a_i-35_1_test.go` :: `TestAI35_1_RetryablePreStream_RetriesUpToBound`** — `httpClient.Do` count == `DefaultMaxAttempts + 1` against a scripted `429 Retry-After: 0` handler; final stream is the `2xx`.
- [ ] **`a_i-35_1_test.go` :: `TestAI35_1_TerminalCategory_NeverRetries`** — `httpClient.Do` count == 1 against a scripted `401` handler; returned error is `*ai.Failure` with `Retryable() == false` and `Category() == FailureCategoryAuthentication`.
- [ ] **`a_i-35_1_test.go` :: `TestAI35_1_ExhaustedBudget_ReturnsLastFailureWrappedWithAttemptCount`** — `httpClient.Do` count == `DefaultMaxAttempts + 1` against a handler that never returns `2xx`; `errors.As(err, &report)` returns true with `Attempts == DefaultMaxAttempts + 1`; `errors.As(err, &failure)` returns true with `Category() == FailureCategoryRateLimit`.
- [ ] **Forward guard green** — `cd backend/agent && make test` passes; `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` passes; `TestLayer1_ModuleHasNoDependencies_ZeroRequires` passes.

### 12.2 R-AIS-042 — Backoff mechanics

- [ ] **`a_i-35_2_test.go` :: `TestAI35_2_RetryAfterHint_OverridesComputedBackoff`** — backoff equals `Retry-After` header value exactly (never computed exponential); hint read from `failure.RetryAfter()`.
- [ ] **`a_i-35_2_test.go` :: `TestAI35_2_ComputedBackoff_BoundedWithSeededJitter`** — each delay is in `[base * 2^(attempt-1), base * 2^attempt]` (bounded); fixed seed produces assertable sequence; no delay exceeds `MaxDelay`.
- [ ] **`a_i-35_2_test.go` :: `TestAI35_2_Backoff_CtxCancellationAbortsImmediately`** — `ctx.Done()` mid-backoff aborts immediately; no subsequent wire request; returned error is the LAST typed failure (NOT `ctx.Err()`); no invented retry-exhausted category.
- [ ] **`a_i-35_2_test.go` :: `TestAI35_2_BoundedAttemptCount_ExactlyNPlusOneWireRequests`** — `httpClient.Do` count == `DefaultMaxAttempts + 1` against a handler that always returns retryable failures.

### 12.3 R-AIS-043 — Replayability + partial-output boundary marker

- [ ] **`a_i-35_3_test.go` :: `TestAI35_3_ByteIdenticalReplay_AcrossAttempts`** — every recorded request body byte-identical to every other; bodies match `Translate(req)` output via `bytes.Equal`.
- [ ] **`a_i-35_3_test.go` :: `TestAI35_3_AttemptCountAndFinalCause_ReachableFromErrorChain`** — `errors.As(err, &report)` returns true with `Attempts == N+1`; `errors.As(err, &failure)` returns true with `Category()` and `Delivery()` preserved.
- [ ] **`a_i-35_3_test.go` :: `TestAI35_3_PerAttemptDrain_StatusPath`** — `httpClient.Do` count == N+1; connection-reuse count == 1 across attempts; per-attempt drain supplied by composition with `captureBody`.
- [ ] **`a_i-35_3_test.go` :: `TestAI35_3_PartialOutputBoundaryMarker_AfterHandoverNoRetry`** — `httpClient.Do` count == 1 after a mid-stream retryable failure; carrier handover happened; terminal error event has `PartialOutput() == true`.

### 12.4 R-AIS-044 — Composed-bound ceiling

- [ ] **`internal/retry/doc.go`** — package doc comment names `defaultMaxAttempts = 3` and the formula "harness attempts × Layer 1 attempts"; references AG-15.2 (doc 0003 line 718).
- [ ] **AG-15.2's composed-bound test** (out of scope for AI-35; archive phase confirms wording match) — test quotes `defaultMaxAttempts = 3` verbatim from the doc comment.

### 12.5 R-CNF-019 — Every adapter claiming `CapRetry` auto-retries

- [ ] **`agenttest/conformance_retry.go`** — case body registers under `CapRetry`; sub-tests assert `S-CNF-069..075`.
- [ ] **`agenttest/conformance_suite.go`** — `CapRetry` declared as `CAP-O-04` between `CapCacheBoundary` and `CapNone`; `capabilityNames` mapping adds `"CAP-O-04(retry)"`; `Optional()` switch includes `CapRetry`.
- [ ] **`agenttest/conformance_suite.go` :: `Factory.Retry *bool`** — declared alongside `Reasoning` / `TokenCounting` / `CacheBoundary`; `nil` fails construction per `R-CNF-002` / `S-CNF-006`.
- [ ] **`agenttest/conformance_record.go` :: `CapabilityRecord.entries [9]`** — totality rebuild mechanical; `Capabilities()` returns 9 values.
- [ ] **`cd backend/agent && make test`** — green under `-race`; `make lint` clean; `go.mod` unchanged (zero requires).

### 12.6 Spec-level acceptance (from both delta specs)

- [ ] All `R-AIS-041`..`R-AIS-044` requirements hold; each verified by its scenarios.
- [ ] `R-CNF-019` holds; `R-CNF-017` (modified) holds — record carries 9 entries.
- [ ] No Go identifier invented beyond `DefaultMaxAttempts` (named by orchestrator's prompt) and `CapRetry` / `CAP-O-04` (natural extension of `Capability` enum).
- [ ] No new top-level Go dependency; helper is stdlib + own-module (`github.com/cachicamas/backend/agent/src/ai`).
- [ ] `cd backend/agent && make test` green under `-race`; `make lint` clean.

---

## 13. References

- **Charter**: `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2077-2144` (incl. 2026-08-07 amendment blockquote at lines 2081-2093).
- **AG-15.2 composed-bound contract**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:714-719`.
- **Pre-adopted decision**: doc 0002 lines 2085-2086; Engram obs `#2641` (`sdd/cachicamas-ai-retry-policy/decision-pre`).
- **Explore artefact**: `openspec/changes/cachicamas-ai-retry-policy/explore.md` (368 lines, canonical record); Engram obs `#2642`.
- **Proposal artefact**: `openspec/changes/cachicamas-ai-retry-policy/proposal.md` (189 lines); Engram obs `#2643`.
- **Spec (ai-stream-lifecycle delta)**: `openspec/changes/cachicamas-ai-retry-policy/specs/ai-stream-lifecycle/spec.md` (R-AIS-041..044); Engram obs `#2644`.
- **Spec (ai-provider-conformance-suite delta)**: `openspec/changes/cachicamas-ai-retry-policy/specs/ai-provider-conformance-suite/spec.md` (R-CNF-019 + R-CNF-017 modified + CapRetry).
- **Spec (V-FAIL-15 Layer 1 retry clause)**: `openspec/specs/ai-contract-vocabulary/spec.md:261`.
- **Spec (R-AIP-010/011 partial-output discriminator + G8)**: `openspec/specs/ai-provider-errors/spec.md:155-198`.
- **Spec (canonical ai-stream-lifecycle § 7 failure delivery)**: `openspec/specs/ai-stream-lifecycle/spec.md:455-519`.
- **Spec (canonical ai-provider-conformance-suite R-CNF-002 + R-CNF-017 + Capability enum)**: `openspec/specs/ai-provider-conformance-suite/spec.md:62-72, 238-249`.
- **Spec (canonical conformance NFR-CNF-A through NFR-CNF-F)**: `openspec/specs/ai-provider-conformance-suite/spec.md:264-306`.
- **Producer Stream**: `backend/agent/src/ai/openaicompat/stream.go:199-240` (pre-stream contract), `:373-376` (drain discipline), `:588-596` (emit helper).
- **Wire-side failure mapper + retryable taxonomy**: `backend/agent/src/ai/openaicompat/failure_map.go:31-93, 187-218`.
- **Rate-limit telemetry carrier**: `backend/agent/src/ai/openaicompat/retry_metadata.go:37-83`.
- **Capture + drain (status path)**: `backend/agent/src/ai/openaicompat/capture.go:75-122`.
- **Body marshalling (`Translate` returns `[]byte`)**: `backend/agent/src/ai/openaicompat/translation.go:24`; `body.go:35-55`.
- **Request build (body wrapped as `bytes.NewReader` per attempt)**: `backend/agent/src/ai/openaicompat/request.go:23-41`.
- **Provider failure API**: `backend/agent/src/ai/provider_failure.go:249-497` (Retryable, RetryAfter, PartialOutput, Delivery, PreStreamFailure).
- **Conformance skeleton (Capability enum + Factory)**: `backend/agent/src/agenttest/conformance_suite.go:41-184`.
- **Conformance cancellation case** (precedent for retry case body shape): `backend/agent/src/agenttest/conformance_cancellation.go:46-131`.
- **OpenRouter conformance bridge**: `backend/agent/src/ai/openaicompat/openrouter/conformance/bridge_test.go:107-158` (`bridgeAttributionRoundTripper`, `bridgeServeTranscripts`).
- **AI-33.5a test file (precedent for shared helper + leak-check shape)**: `backend/agent/src/ai/openaicompat/a_i-33_5a_test.go:1-208`.
- **AI-34.2 test file (precedent for real-producer test file shape)**: `backend/agent/src/ai/openaicompat/a_i-34_2_test.go:1-208`.
- **Forward guard (AI-00.3)**: `backend/agent/src/ai/import_boundary_test.go:107-241`.
- **ADR 0005 § D1, § D3**: `docs/adr/0005-promote-agent-stack-to-own-module.md`.
- **AI-34 precedent (this change's structural model)**: `openspec/changes/archive/2026-08-07-cachicamas-ai-backpressure/design.md` (94 lines; smaller scope but same architecture pattern).
- **AI-33 cancellation precedent (helper-shape model)**: `openspec/changes/archive/2026-08-07-cachicamas-ai-cancellation/design.md` (68 lines).
- **Tooling fragility precedents**: Engram `#2636`, `#2639`.
- **Mandatory doc 0002 + Engram + push workflow**: Engram `#2638`.