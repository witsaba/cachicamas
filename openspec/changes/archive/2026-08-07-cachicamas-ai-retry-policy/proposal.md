# Proposal: Define retry and idempotency policy (AI-35)

> **Change**: `cachicamas-ai-retry-policy` · **Milestone**: AI-35 (doc 0002 lines 2077–2144)
> **Wave**: 5 — Harden · **Module**: `backend/agent/` (layered per ADR 0005 § D1, NOT hexagonal)
> **Strategy**: single-pr · strict TDD on · stdlib-only · review budget 1000 changed lines (per doc 0002 amendment line 2093)

## 1. Intent

AI-35 makes one sentence into structure: **the partial-output case is never retried at Layer 1** (doc 0002 line 2079, codified as `V-FAIL-15` at `ai-contract-vocabulary/spec.md:261`). The change introduces pre-stream auto-retry with bounded attempts, retry-after handling, byte-identical replay, and per-attempt drain — executed as an in-adapter loop factored into a shared helper invoked by the adapter's "execute-once" function (per the 2026-08-07 amendment, doc 0002 lines 2081–2093). The helper runs BEFORE the carrier handover (`make(chan ai.Event, ...)` at `stream.go:237`), so the partial-output discriminator (`V-FAIL-09`) is **unconditionally `false`** by construction for every failure the helper inspects — the load-bearing invariant that makes the G8 sentence a structural property, not a runtime check.

## 2. Scope (in)

- **Pre-stream retry mechanism** — bounded attempts (default `defaultMaxAttempts = 3` per explore § 4.8 / D5), retry-after override, context-aware backoff with seeded jitter, byte-identical replay via `[]byte` body + fresh `bytes.NewReader` per attempt.
- **Shared helper location** — `backend/agent/src/ai/internal/retry/` (Go `internal/` package). See **Flag 1** resolution below.
- **Four subnodes AI-35.0 / AI-35.1 / AI-35.2 / AI-35.3** per doc 0002 lines 2113–2144, with the "no retry after partial output" boundary assertion placed in AI-35.3 (replayability). See **Flag 2** resolution below.
- **Composed-bound documentation constant** — `defaultMaxAttempts = 3` in the helper's package doc comment; referenced by AG-15.2's composed-bound test (doc 0003 line 718).
- **Three new real-producer test files** — `backend/agent/src/ai/openaicompat/a_i-35_{1,2,3}_test.go` (internal `package openaicompat`, matching AI-33 / AI-34 precedent).
- **Conformance capability `CapRetry`** added to `backend/agent/src/agenttest/conformance_suite.go:43–89`, with case `conformance_retry.go`. See Test Strategy § 5.
- **Spec delta** — amend `openspec/specs/ai-stream-lifecycle/spec.md` with R-AIS-041/042/043/044 (pre-stream retry predicate; backoff mechanics; replayability; composed-bound ceiling). Amend `openspec/specs/ai-provider-conformance-suite/spec.md` with R-CNF-019 + `CapRetry`.
- **`decision.md`** — AI-35.0 closing artefact carrying the rationale verbatim from doc 0002 lines 2085–2086 + the seam choice (sibling of `proposal.md`).

## 3. Non-goals (out)

- **Agent-turn retries** — Layer 2's AG-15 (doc 0003 lines 687–726). The G8 sentence has its Layer 2 half there.
- **Model failover** — Layer 2 seam 8 (doc 0001 § 6). Re-opens the token budget, the price table and the cache prefix.
- **Mid-stream retry** — helper contract is pre-stream only (`stream.go:199–240` strict ordering; the `run` goroutine starts AFTER the helper exits).
- **Any new top-level Go dependency** — helper is stdlib-only (`context`, `errors`, `time`; possibly `math/rand/v2`). Forward guard `import_boundary_test.go:107–162` stays green; `TestLayer1_ModuleHasNoDependencies_ZeroRequires` stays green.
- **Configuration surface for max attempts** — `defaultMaxAttempts` is a package-private constant, overridable in tests via helper parameter; NOT promoted to `openaicompat.Config` (Q3 precedent from AI-34 — stay constant until a workload demands configurability).

## 4. Approach

- **In-adapter loop pattern** — the `execute-once` closure wraps `c.httpClient.Do(httpReq)` (stream.go:218) + `mapResponse(resp, time.Now())` (stream.go:227). Three observable outcomes: (a) usable 2xx response — helper falls through; (b) pre-handover `*ai.Failure` with `Retryable()` true or false; (c) transport error — treated as `FailureCategoryUnavailable`, retryable.
- **Byte-identical replay** — structurally free: `Translate(req)` returns `[]byte` (translation.go:24); `newRequest` wraps a fresh `bytes.NewReader` per attempt (request.go:28). Memory cost of N attempts == memory cost of 1 attempt.
- **Drain discipline** — reused by composition. `mapResponse` → `captureBody` (capture.go:75–122) drains + closes for the status path; transport errors have no body. NO new drain helper required.
- **Retryable taxonomy** — reads `*ai.Failure.Retryable()` (provider_failure.go:419–424), set via `retryableFor(category)` (failure_map.go:86–93). True only for `FailureCategoryRateLimit`, `FailureCategoryUnavailable`, `FailureCategoryTimeout`; Authentication and 4xx-Unknown return false.
- **Partial-output discriminator as construction** — the helper runs BEFORE `make(chan ai.Event, ...)` (stream.go:237). No semantic event has been emitted. `PartialOutput() == false` is unconditional. The helper's contract MUST be pre-stream only — a typed invariant pinned by the spec (not a runtime check).
- **Attempt-count + final-cause surface** — the helper attaches a sibling cause type to the returned `*ai.Failure`, mirroring `RateLimitTelemetry`'s pattern (retry_metadata.go:37–58). Consumers reach it via `errors.As` over the wrapped `*ai.Failure`. The named cause type is `[proposed]` — spec/design phases own final naming.

## 5. Test strategy

- **Test file naming** — `a_i-35_1_test.go` (predicate), `a_i-35_2_test.go` (backoff mechanics), `a_i-35_3_test.go` (replayability + boundary marker). Internal `package openaicompat`, matching `a_i-33_*.go` and `a_i-34_*.go`. Serial-only enforcement for leak-checked tests via `tb.Setenv("AGENTTEST_STREAM_KIT_LEAK_CHECK", "1")`.
- **Conformance suite decision: YES — add `CapRetry`.** Rationale: (1) OpenRouter wrapper inherits the helper transparently (`openrouter/wrapper.go` embeds `*openaicompat.Client`), so the conformance case exercises both adapters with zero new code path; (2) without `CapRetry`, a future "no-retry" adapter could claim conformance under `CapTypedFailures` only — silently skipping the retry discipline that `V-FAIL-15` makes binding; (3) AI-38 (OpenRouter conformance roll-up, doc 0002 line 2118) is the natural home for the `CapRetry` case, paralleling how `CapCancellation` was added at AI-33.3. The `[8]`-element capability array grows to `[9]`; `R-CNF-017` totality rebuild is mechanical.
- **"No retry after partial output" placement: AI-35.3 (replayability) — CONFIRMED.** See **Flag 2** resolution below for justification.
- **Strict TDD ordering** per `openspec/AGENTS.md` rule 3: RED first — each test compiles and runs, assertion fails for the right reason. GREEN — minimum production code that makes it pass. REFACTOR — clean up while green, re-run `make test`.
- **No wall-clock sleeps** — all timing injected via the helper's `nowFunc` / `sleepFunc` parameters (test seam), paralleling AI-33.5a's timing-injection pattern (`a_i-33_5a_test.go:197`).
- **RED-first ordering across subnodes** — AI-35.1's predicate tests land first (the helper's seam is their precondition); AI-35.2's backoff tests second; AI-35.3's replayability tests third.

## 6. Dependencies

### Done (input to this change)

| Dep | What it provides | Status |
| --- | --- | --- |
| **AI-32** (failure mapping + retryable taxonomy) | `retryableFor(category)` at failure_map.go:86–93; `mapResponse` at failure_map.go:31–37 | PR #129, merged 2026-08-07 |
| **AI-33** (drain discipline + cancellation contract) | `captureBody` drain at capture.go:75–122; R-AIS-033..038 at ai-stream-lifecycle/spec.md | PR #129, merged 2026-08-07 |
| **AI-19.4** (partial-output discriminator) | `*ai.Failure.PartialOutput()` at provider_failure.go:469–483; R-AIP-010/011 at ai-provider-errors/spec.md | Archived `2026-08-01-cachicamas-ai-provider-errors` |

### Unblocks

- **AI-38** (OpenRouter conformance roll-up) — first-adapter conformance bridge at `openrouter/conformance/bridge_test.go` exists; `CapRetry` conformance case attaches here.
- **AG-15.2** (composed-bound test in Layer 2, doc 0003 line 718) — "harness attempts × Layer 1 attempts" ceiling is now writable; helper's `defaultMaxAttempts = 3` provides the Layer 1 multiplier.

## 7. Risks

| # | Risk | Likelihood | Mitigation |
|---|------|------------|------------|
| 7.1 | **Helper location trade-off**: `internal/retry/` adds one cross-package edge today (openaicompat → internal/retry) for forward-adapter reuse that doesn't yet exist. | Low | The doc 0002 amendment (lines 2085–2086) explicitly anticipates future-adapter reuse; the edge is small (one import in stream.go) and the alternative (`openaicompat/retry.go`) would force future adapters to import `openaicompat` — coupling the WRONG way. See Flag 1 resolution. |
| 7.2 | **Tooling fragility** (per Engram #2636, #2639) — runtime ledger `complete` blocks subsequent `sdd-attempt acquire` after `passed` settle; follow-up archive may need manual. | Med | Anticipate at apply time; have AI-34 manual-archive pattern (cherry-pick doc 0002 to branch) ready. |
| 7.3 | **Mid-stream scope pinning** — helper's "pre-stream only" contract must be in the TYPED shape, not a runtime comment. | Med | Spec phase pins it: helper's input is constrained to `mapResponse` failures + transport errors, both pre-handover. AI-35.3's boundary test asserts the seam holds. |
| 7.4 | **`CapRetry` conformance membership** could be deferred forever without enforcement. | Low | YES recommendation commits it to the spec phase; AI-38 is the natural owner; `[8]`-element → `[9]`-element capability array, `R-CNF-017` totality rebuild is mechanical. |
| 7.5 | **Composed-bound ceiling** (`harness × Layer 1`) might be misunderstood if documented only in helper package comment. | Low | Constant lives in `internal/retry/doc.go` (package doc comment) + AG-15.2's test item 2 quotes it verbatim from the doc — readers from either layer find the same number. |
| 7.6 | **Aggregate ~600 lines exceeds 400-line PR budget** (per `work-unit-commits` guard). | Med | Single-PR strategy approved at session preflight; review budget raised to 1000 lines per doc 0002 amendment line 2093. `sdd-tasks` may recommend split if any test file exceeds 400 lines alone. |

## 8. Open questions for spec phase

- **Type signatures** (proposal does NOT invent Go types) — helper's typed contract: input closure shape, output union (response | failure | error), retry-budget parameter, sleep/now injection seams, attempt-report cause type. Spec and design phases own these.
- **`CapRetry` conformance case body** — spec phase authors `conformance_retry.go` + the requirement in `ai-provider-conformance-suite/spec.md`. Proposal commits to "added" but not to the case body.
- **Backoff curve and jitter seed** — AI-35.2 item 1 names "jitter that is seeded and therefore assertable". Spec phase authors the seed and growth curve (linear vs exponential); explore § 6 Q4 leaves both open.
- **Attempt-report shape** — the exact cause type for "attempt count + final cause in the error chain". Explore suggests a sibling of `RateLimitTelemetry` (retry_metadata.go:37–58); spec phase authors the named type and its accessor surface.
- **Default attempt budget constant** — explore § 5 D5 proposes `defaultMaxAttempts = 3`. Spec phase confirms or adjusts.

## 9. Recommended next step

`sdd-spec` for change `cachicamas-ai-retry-policy`. Spec phase authors:
1. Delta spec for `ai-stream-lifecycle/spec.md` — R-AIS-041..044 covering the four subnodes' behavioural requirements.
2. Delta spec for `ai-provider-conformance-suite/spec.md` — `CapRetry` registration + R-CNF-019 covering "every adapter auto-retries retryable pre-stream failures up to a documented bound".
3. Spec body for the helper's typed contract (signature, error chain, attempt-report shape).
4. Spec body for `conformance_retry.go` case shape.

No new top-level Go dependency. No conformance amendment in propose (decision committed; case body authored in spec). No production-shape change beyond the helper file (`internal/retry/{retry.go, doc.go}`), the Stream-line edits that invoke it, the four test files, and the conformance case.

---

## Flag 1 resolution — Helper location

**Pick: `backend/agent/src/ai/internal/retry/`** (Go `internal/` package, encapsulated to the `src/ai/` tree).

Justification against the doc 0002 amendment's reuse intent (line 2086 verbatim: "future adapters (Anthropic, Gemini, ...) reuse it without re-implementing the discipline"):

1. **Honors the amendment directly.** The explore's `openaicompat/retry.go` option would force Anthropic and Gemini adapters to import `package openaicompat` — coupling them to a specific vendor's translation tables, exactly the wrong coupling. `internal/retry/` lets every adapter under `src/ai/` (`openaicompat`, `openrouter`, future `anthropic`, future `gemini`) call the helper without depending on any specific vendor's adapter.
2. **Honors the forward guard (AI-00.3).** `internal/retry/` is within `github.com/cachicamas/backend/agent/...` — the only non-stdlib prefix the allowlist admits at `import_boundary_test.go:103–105`. No third-party import. Go's `internal/` directory semantics — "only the immediate parent can import" — is precisely the encapsulation we want: any package under `src/ai/` can call the helper; nothing outside `src/ai/` can.
3. **Trade-off acknowledged.** `openaicompat/retry.go` (explore's pick) saves one cross-package edge today for no current beneficiary beyond `openaicompat`. The amendment's "future-adapter reuse" is the asymmetry that justifies the edge: the cost (one extra import in stream.go) is paid once; the benefit (no future adapter ever imports another adapter) compounds.
4. **Reversibility.** If the v1-only-caller assumption later proves wrong (no Anthropic / Gemini adapter ever lands), the helper can be moved to `openaicompat/retry.go` with a one-line import-edge removal in `stream.go`. The reverse direction (moving from `openaicompat/` to `internal/` after a future adapter has imported `openaicompat`) is much harder — that adapter would have to be reworked to break the vendor coupling the amendment explicitly rejects.

## Flag 2 resolution — "No retry after partial output" placement

**Confirm explore recommendation: MOVE item 2 from AI-35.1 to AI-35.3.**

Rationale against keeping it in AI-35.1 (the doc 0002 charter's literal placement at lines 2125–2126):

- **Item 2 is a seam-boundary assertion, not a predicate-logic assertion.** AI-35.1's three items become a clean predicate test set when item 2 moves out: (1) retryable pre-stream → retry; (3) terminal category → don't retry; (2)'s absence-of-second-wire-request belongs with the SEAM tests, not the predicate tests. The helper's seam (running BEFORE `make(chan ai.Event, ...)` at stream.go:237) is what makes "no retry after partial output" true by construction; the predicate logic never sees a partial-output failure to evaluate.
- **AI-35.3 ("replayability") is precisely the right home.** AI-35.3 is about "can we safely re-issue?" — and "no retry after partial output" is the load-bearing safety assertion. AI-35.3's three tests then become: (1) byte-identical replay; (2) attempt count + final cause in error chain; (3) per-attempt drain + the "no retry after partial output" boundary marker.
- **The doc 0002 charter test list is descriptive.** The amendment (line 2093) raised the review budget to 1000 lines; a doc 0002 amendment at archive time, refining item 2's subnode grouping, is in scope if the spec phase chooses to keep the charter's literal placement. The proposal treats this as a subnode regrouping, not a substantive design change.

---

## Capabilities (contract with sdd-spec)

### New Capabilities

None at the `package ai` vocabulary level. `V-FAIL-15` already exists at `ai-contract-vocabulary/spec.md:261` and is the binding vocabulary row AI-35 implements.

### Modified Capabilities

- `ai-stream-lifecycle` — new requirements R-AIS-041 (pre-stream retry predicate: retryable AND pre-stream → retry; everything else → don't retry), R-AIS-042 (backoff mechanics: retry-after precedence, ctx-aware waits, seeded jitter, bounded attempt count = exactly N+1 wire requests), R-AIS-043 (replayability: byte-identical replay + attempt count + final cause in error chain + per-attempt drain), R-AIS-044 (composed-bound ceiling: `harness attempts × Layer 1 attempts` documented in the helper's package doc comment). Amends § 6 (carrier + retry discipline) and § 7 (lifecycle picture).
- `ai-provider-conformance-suite` — adds `CapRetry` to the `Capability` enum (grows from 8 to 9 members) + new requirement R-CNF-019 covering "every adapter auto-retries retryable pre-stream failures up to a documented bound, and the per-attempt body is byte-identical to the original". `[8]`-element → `[9]`-element capability array; `R-CNF-017` totality rebuild is mechanical.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/ai/internal/retry/retry.go` | New | Shared helper: bounded attempts, retry-after handling, context-aware backoff, byte-identical replay, attempt-report cause type. Stdlib-only. |
| `backend/agent/src/ai/internal/retry/doc.go` | New | Package doc comment naming `defaultMaxAttempts = 3` + the composed-bound ceiling formula. Read by both Layer 1 and Layer 2 readers. |
| `backend/agent/src/ai/openaicompat/stream.go:218–227` | Modified | Wrap the `Do + mapResponse` segment in the helper. One import added; ~5 lines of code change inside `Stream`. |
| `backend/agent/src/ai/openaicompat/a_i-35_1_test.go` | New | Predicate tests: retryable-pre-stream-retries; terminal-category-never-retries. Internal `package openaicompat`. RED-first. |
| `backend/agent/src/ai/openaicompat/a_i-35_2_test.go` | New | Backoff mechanics tests: retry-after precedence; ctx-aware waits; seeded jitter; bounded attempt count. Timing injection only — no wall-clock sleeps. |
| `backend/agent/src/ai/openaicompat/a_i-35_3_test.go` | New | Replayability + boundary marker: byte-identical replay; attempt count + final cause in error chain; no retry after partial output (seam boundary assertion). |
| `backend/agent/src/agenttest/conformance_suite.go:43–89` | Modified | Add `CapRetry` to the `Capability` enum. Array grows from `[8]` to `[9]`; `R-CNF-017` totality rebuild. |
| `backend/agent/src/agenttest/conformance_retry.go` | New | `CapRetry` conformance case. Reuses existing failure-mapping infrastructure; no new vocabulary. |
| `openspec/changes/cachicamas-ai-retry-policy/specs/ai-stream-lifecycle/spec.md` | New (delta) | R-AIS-041..044. |
| `openspec/changes/cachicamas-ai-retry-policy/specs/ai-provider-conformance-suite/spec.md` | New (delta) | R-CNF-019 + `CapRetry` registration. |
| `openspec/changes/cachicamas-ai-retry-policy/decision.md` | New | AI-35.0 closing artefact carrying the rationale verbatim from doc 0002 lines 2085–2086 + the seam choice (sibling of `proposal.md`). |
| `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` | Modified (archive) | Top-status updated; amendment blockquote referencing this change at archive time per Engram #2638's mandatory workflow. |
| `openspec/specs/ai-stream-lifecycle/spec.md` | Modified (archive) | R-AIS-041..044 + composed-bound ceiling reference lands in canonical spec at archive phase. |

## Rollback Plan

Revert the single PR. The helper (`internal/retry/retry.go`) is self-contained; reverting it removes auto-retry (Layer 1 returns the first failure as today, restoring AI-32's pre-AI-35 behaviour). The four test files drop with the code change. Spec deltas (`specs/ai-stream-lifecycle/spec.md`, `specs/ai-provider-conformance-suite/spec.md`) are reversible without code change. `decision.md` is preserved in `openspec/changes/cachicamas-ai-retry-policy/` as the historical record — no rollback action.

If the helper's behaviour turns out to be wrong in production (e.g., the retry-after interpretation diverges from a vendor's wire), the rollback is to set `defaultMaxAttempts = 0` (no retries) and reopen the spec amendment — exactly the spec's living-graph clause.

## Success Criteria

- [ ] `backend/agent/src/ai/internal/retry/retry.go` exists; helper compiles with stdlib-only imports; `TestLayer1_ModuleHasNoDependencies_ZeroRequires` stays green.
- [ ] `stream.go:218–227` invokes the helper; `Stream` returns either a usable 2xx response or a `*ai.Failure` whose error chain carries the attempt-report sibling cause.
- [ ] `a_i-35_1_test.go` proves: retryable-pre-stream-retries; terminal-category-never-retries (401 / 400 / 404 / 422 → no retry, instrumented RoundTripper counts exactly 1 `Do` call).
- [ ] `a_i-35_2_test.go` proves: retry-after overrides computed backoff; ctx-aware waits (cancellation aborts immediately); seeded jitter is assertable; bounded attempt count is exactly N+1 wire requests.
- [ ] `a_i-35_3_test.go` proves: byte-identical replay (body byte-compared across attempts); attempt count + final cause reachable from error chain via `errors.As`; no retry after partial output (seam boundary assertion — instrumented RoundTripper counts exactly 1 `Do` call after a mid-stream retryable failure).
- [ ] `conformance_retry.go` registers `CapRetry` and the case passes for the real producer (`openaicompat`) AND the OpenRouter wrapper (inherits helper transparently).
- [ ] `openspec/changes/cachicamas-ai-retry-policy/decision.md` records the seam rationale verbatim from doc 0002 lines 2085–2086 + the helper-location pick.
- [ ] `cd backend/agent && make test` green; no new top-level Go dependency introduced.
- [ ] Spec amendment to `openspec/specs/ai-stream-lifecycle/spec.md` flagged as follow-up owned by archive phase (R-AIS-041..044 + composed-bound ceiling reference).

## References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2077–2144` (incl. 2026-08-07 amendment blockquote at lines 2081–2093).
- **AI-35.0/35.1/35.2/35.3 test lists** — doc 0002 lines 2113–2144.
- **AG-15.2 composed-bound contract** — `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:714–719`.
- **Explore artefact** — `openspec/changes/cachicamas-ai-retry-policy/explore.md` (368 lines, canonical record).
- **Pre-adopted decision** — doc 0002 lines 2085–2086 + Engram obs `#2641` (`sdd/cachicamas-ai-retry-policy/decision-pre`).
- **Engram observations** — `#2641` (decision rationale), `#2642` (explore findings), `#2640` (AI-34 close + workflow + tooling risks).
- **Spec (`V-FAIL-15` Layer 1 retry clause)** — `openspec/specs/ai-contract-vocabulary/spec.md:261`.
- **Spec (`R-AIP-010/011` partial-output discriminator + G8)** — `openspec/specs/ai-provider-errors/spec.md:155–198`.
- **Spec (`R-AEM-008/009` retry-after + telemetry carrier)** — `openspec/specs/ai-provider-error-mapping/spec.md:115–189`.
- **Producer Stream (pre-stream contract + drain discipline)** — `backend/agent/src/ai/openaicompat/stream.go:92–104, 199–240, 371–376`.
- **Wire-side failure mapper + retryable taxonomy** — `backend/agent/src/ai/openaicompat/failure_map.go:31–93`.
- **Retry-after parser + presence-typed result** — `backend/agent/src/ai/openaicompat/failure_map.go:187–218`.
- **Rate-limit telemetry carrier** — `backend/agent/src/ai/openaicompat/retry_metadata.go:37–83`.
- **Capture + drain (status path)** — `backend/agent/src/ai/openaicompat/capture.go:31, 75–122`.
- **Body marshalling (Translate returns `[]byte`)** — `backend/agent/src/ai/openaicompat/translation.go:24`, `body.go:35–55`.
- **Request build (body wrapped as bytes.NewReader per attempt)** — `backend/agent/src/ai/openaicompat/request.go:23–41`.
- **Provider failure API (Retryable, RetryAfter, PartialOutput, Delivery)** — `backend/agent/src/ai/provider_failure.go:249–496`.
- **Conformance skeleton (Factory, Capability)** — `backend/agent/src/agenttest/conformance_suite.go:43–227`.
- **OpenRouter conformance bridge** — `backend/agent/src/ai/openaicompat/openrouter/conformance/bridge_test.go:1–470`.
- **Forward guard (AI-00.3)** — `backend/agent/src/ai/import_boundary_test.go:107–241`.
- **ADR 0005 § D1, § D3** — `docs/adr/0005-promote-agent-stack-to-own-module.md`.
- **AI-34 precedent (this change's structural model)** — `openspec/changes/archive/2026-08-07-cachicamas-ai-backpressure/proposal.md`.
- **Tooling fragility precedents** — Engram `#2636`, `#2639`.
- **Mandatory doc 0002 + Engram + push workflow** — Engram `#2638`.
