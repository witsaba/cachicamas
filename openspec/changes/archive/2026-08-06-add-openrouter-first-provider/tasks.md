# Tasks: OpenRouter as the first concrete AI provider (Layer 1)

> **Change ID**: `add-openrouter-first-provider`
> **Capability ID**: `ai-openrouter-first-provider` (new)
> **Status**: DRAFT
> **Date**: 2026-08-06
> **Artifact store**: hybrid (file + engram topic_key `sdd/add-openrouter-first-provider/tasks`)
> **Locked preflight**: `delivery_strategy=auto-chain` · `chain_strategy=feature-branch-chain` · `review_budget_lines=800`
> **Mode**: `auto` (orchestrator must proceed without re-asking)
> **Branch chain**: `main` → `tracker/add-openrouter-first-provider` (draft, no-merge) → `feat/openrouter-wrapper` (PR #1) → `feat/openrouter-conformance-bridge` (PR #2) → `feat/openrouter-live-smoke` (PR #3)
> **Convention**: Conventional Commits only, no `Co-Authored-By` trailer, no AI attribution; strict TDD (RED → GREEN → REFACTOR); runner = `cd backend/agent && make test` (per Engram #2055).
> **Wrapper path (per design D1, Engram #2571)**: `backend/agent/src/ai/openaicompat/openrouter/` — the prompt's `backend/agent/src/ai/openrouter/` shorthand is read as the package directory, not as a placement re-decision. **Surface this in the reply if the user intended otherwise.**

## Overview

Three chained PRs under a no-merge tracker concretize OpenRouter as the first concrete Layer 1 provider. **PR #1 (wrapper)** adds `backend/agent/src/ai/openaicompat/openrouter/` — a thin sub-100-line constructor + a wrapper-owned `http.RoundTripper` that injects three attribution headers, with stub-transport, ambient-authority, headers-unawareness, default-model, redaction, and charter tests. **PR #2 (conformance bridge)** adds `openrouter/conformance/` — a sibling sub-package whose `bridge_test.go` mirrors `openaicompat/bridge_test.go` and runs `agenttest.RunConformance` against `openrouter` wrapped in the three attribution headers, asserting `CAP-O-01=absent` under default model `openai/gpt-4o`. **PR #3 (live smoke)** adds `openrouter/smoke/` — a `t.Skip`-gated `TestOpenRouterAdapter_LiveSmoke`, a sentinel-sweep helper, and a `workflow_dispatch`-only CI workflow. Fixtures in PR #2 are excluded from the authored risk count per `sdd-phase-common.md §E`. Each PR is independently revertible; the chain is feature-branch-chain so children target their immediate parent branch (no polluted diffs).

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines (authored, naive) | ~700–900 across 3 PRs |
| 400-line budget risk | Low (per-PR) |
| Chained PRs recommended | Yes (auto-chain + feature-branch-chain locked at preflight) |
| Suggested split | PR #1 wrapper → PR #2 conformance bridge → PR #3 live smoke |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |
| Decision needed before apply | No |
| 400-line budget risk | Low |
| Per-PR authored naive | PR #1 ~200 · PR #2 ~400–600 (fixtures excluded per §E) · PR #3 ~100 |
| Per-PR corrected mid | PR #1 ~700–800 (commit split at 1.5a/1.5b; fallback = defer `charter_test.go` to follow-up) · PR #2 within · PR #3 within |
| Total naive authored | ~700–900 |
| Total corrected authored | ~2000–3600 (corrected at PR level, NOT consolidated) |
| Per-commit work-unit posture | every commit is build-green + test-green + narrow intent + rollback-friendly |

```
Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: Low
```

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Wrapper injects attribution headers, defaults model, no ambient authority | PR #1 | `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/...` | `make test` (per Engram #2055) — skip path for live smoke | remove `backend/agent/src/ai/openaicompat/openrouter/` — no callers yet |
| 2 | Conformance bridge runs `agenttest.RunConformance` against wrapped client | PR #2 | `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/conformance/...` | `make test` (no network) | remove `openrouter/conformance/` — wrapper alone still works |
| 3 | Live smoke + sentinel sweep + workflow YAML | PR #3 | `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/smoke/...` (skip path) | `git push` → manually trigger `.github/workflows/agent-openrouter-smoke.yml` with repo secret | remove smoke sub-package + workflow YAML — `make test` unaffected |

---

## PR #1 — OpenRouter wrapper

- **Branch**: `feat/openrouter-wrapper`
- **Targets**: `tracker/add-openrouter-first-provider` (draft, no-merge)
- **Rollback boundary**: `git revert -m1 <merge-sha>` → `make test` green; no behavior change; no callers yet. Sentinel test: `TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage` (R-OR-01).
- **Required path gain**: `backend/agent/src/ai/openaicompat/openrouter/` (new directory; AI-25.2's call-site scan re-runs against it).
- **Spec MUSTs covered**: R-OR-01, R-OR-02, R-OR-03, R-OR-04, R-OR-09, R-OR-10.

### Per-commit work-unit breakdown

#### 1.1 — Package skeleton (`doc.go`, `wrapper.go` constructor stub, package doc)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/doc.go` (package doc: composes `openaicompat`, no `Endpoint` field per D1/D3, no whole-request cap, dependency-free); `backend/agent/src/ai/openaicompat/openrouter/wrapper.go` (`Config{Credential, HTTPClient, HTTPReferer, XTitle, XCategories, Model string}`, `openrouterBaseURL = "https://openrouter.ai/api/v1"`, `openrouterDefaultModel = "openai/gpt-4o"`, `NewProvider(Config) (ai.ModelProvider, error)` — invalid endpoint rejected via `ai.Invalid(ai.ErrMalformed, ai.At("endpoint"))`); `backend/agent/src/ai/openaicompat/openrouter/endpoint.go` (the base URL const home).
- **Verification**: `cd backend/agent && make test`. Build only this commit; no tests yet.
- **Spec MUSTs**: R-OR-01 (injection-only — no env/fs/exec read by `NewProvider`).
- **Rollback**: revert constructor → wrapper absent; no callers; `openaicompat` unchanged.

#### 1.2 — Attribution header attach via wrapper-owned `http.RoundTripper`

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/transport.go` (`attributionRoundTripper` wraps `openaicompat.New`-built transport; `RoundTrip` calls `req.Header.Clone()`, sets `HTTP-Referer` / `X-OpenRouter-Title` + alias `X-Title` / `X-OpenRouter-Categories` when non-empty, `req.Clone(ctx)`, delegates); `backend/agent/src/ai/openaicompat/openrouter/headers.go` (`applyAttribution` helper).
- **Verification**: `cd backend/agent && make test`; new stub-transport probe in next commit.
- **Spec MUSTs**: R-OR-02 (wrapper-injected, NOT openaicompat-injected).
- **Rollback**: revert transport → openaicompat transport unchanged; no behavior change on the wire.

#### 1.3 — Default-model field + `stream_options.include_usage` set in the request envelope

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/wrapper_test.go` (RED: stub-transport probe asserts `Config.Model` defaults to `openai/gpt-4o` when zero; GREEN: wrapper's `Stream` path overrides `ai.Request.Model` from `Config.Model` when non-empty; also asserts `stream_options.include_usage == true` in the rendered body via the stub transport). Update `wrapper.go` to wire `Model` into the request before calling `openaicompat.Client.Stream`.
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/... ./src/ai/...`.
- **Spec MUSTs**: R-OR-03 (default model), R-OR-04 (stream_options.include_usage stays set).
- **Rollback**: revert → `Config.Model` removed; `stream_options` redaction test absent; `openaicompat` unchanged.

#### 1.4 — `openaicompat.Credential` redaction posture propagated through wrapper

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/wrapper_test.go` (add test: `openrouter.Config.Credential` rendered through `String()`, `GoString()`, `MarshalJSON`, default `%v`, `%+v`, `%#v` — none contains the token; also verifies `provider := openrouter.NewProvider(Config{...}); fmt.Sprintf("%v", provider)` does not contain the token).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/...`.
- **Spec MUSTs**: R-OR-08 (sentinel of `openaicompat.Credential` redaction carries through — R-OR-08 sub-scenario 2).
- **Rollback**: revert test → wrapper still produces no token leaks (verified by `openaicompat`'s own `credential.go`); no production change.

#### 1.5a — Ambient-authority test (call-site scan)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/ambient_authority_test.go` (lifted from `openaicompat/ambient_authority_test.go` — same `forbiddenAmbientAuthorityPackages` deny-list, `scanNonTestSourcesForAmbientAuthority` calls `os.ReadDir` on the new sub-package's directory; assert no `os` / `os/exec` / `syscall` / `io/ioutil` call sites in `*.go` non-test files).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/...`.
- **Spec MUSTs**: R-OR-01 (no ambient authority).
- **Rollback**: revert test → AI-25.2's existing scan still covers `openaicompat/`; the new sub-package is uncovered (regression risk).

#### 1.5b — Headers-unawareness test (openaicompat's header surface is unmodified)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/headers_unawareness_test.go` (RED: read `openaicompat/request.go` as raw bytes; assert no literal of `HTTP-Referer`, `X-OpenRouter-Title`, or `X-OpenRouter-Categories` appears anywhere in the file).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/...`.
- **Spec MUSTs**: R-OR-02 (openaicompat unaware of attribution headers).
- **Rollback**: revert test → openaicompat's narrow header surface is no longer asserted by OpenRouter tests; design's C3 (Authorization + Content-Type only) is asserted by openaicompat's own tests.

#### 1.6 — AI-00.3 forward-guard regression test

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/zero_requires_test.go` (asserts `backend/agent/go.mod` declares zero `require` lines and `allowedNonStdlibPrefixes` in `backend/agent/src/ai/import_boundary_test.go` holds exactly one entry — the module path itself).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/... ./src/ai/...`.
- **Spec MUSTs**: R-OR-09 (AI-00.3 forward guard stays green).
- **Rollback**: revert test → existing `TestLayer1_ModuleHasNoDependencies_ZeroRequires` still pins the property; this test is a defense-in-depth copy.

#### 1.7 — Charter test (R-OR-10 fence)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/charter_test.go` (walks the new sub-package's non-test sources; fails on any `*anthropic*` filename, any `"error_type"` literal in non-test sources, any `ai.EventKindReasoningDelta` render, and asserts `go.mod` has zero requires).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/...`.
- **Spec MUSTs**: R-OR-10 (out-of-scope fence).
- **Rollback**: revert test → no detector for negative SHALLs; reviewer vigilance only.

#### 1.8 — PR #1 review-ready commit (final tidy)

- **Files**: `go.mod` UNCHANGED (3 lines, zero requires); `backend/agent/src/ai/openaicompat/openrouter/` final tidy (no dead code, doc strings on every exported symbol, `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` present per R-OR-05 charter — RED: parses `openrouterDefaultModel` const and asserts equality with literal `"openai/gpt-4o"`; this is the gate that would fail if anyone silently swaps the default without reopening AI-29).
- **Verification**: `cd backend/agent && make test` (full suite green under `-race`); `make lint` clean.
- **Spec MUSTs**: R-OR-05 (charter; sub-scenario 2 — default-model swap does not happen silently).
- **Rollback**: revert merge → wrapper lands cleanly; no callers; AI-25.2's call-site scan still covers `openaicompat/`.

---

## PR #2 — Conformance bridge

- **Branch**: `feat/openrouter-conformance-bridge`
- **Targets**: `feat/openrouter-wrapper` (PR #1's branch)
- **Rollback boundary**: `git revert -m1 <merge-sha>` → wrapper alone still works; openaicompat's own `bridge_test.go` continues to pass; AI-38's first-concrete-adapter claim is just deferred. Sentinel test: `TestOpenRouterAdapter_CapabilityRecordMatchesAI24`.
- **Required path gain**: `backend/agent/src/ai/openaicompat/openrouter/conformance/` (new test-only sibling sub-package; `package openrouter_conformance`).
- **Spec MUSTs covered**: R-OR-05, R-OR-06.

### Per-commit work-unit breakdown

#### 2.1 — Bridge scaffold (`bridge_test.go` mirroring `openaicompat/bridge_test.go` factory)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/conformance/bridge_test.go` (`conformanceBridgeFactory()` returns `agenttest.Factory`; `New` builds a real `httptest.Server`, an `openaicompat.New` client, and wraps the client with `openrouter.attributionRoundTripper` adding the three headers at the server; declares `Reasoning: &false, TokenCounting: &false, CacheBoundary: &false` per `R-CNF-004`).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/conformance/...`.
- **Spec MUSTs**: R-OR-06 (conformance bridge runs the suite).
- **Rollback**: revert scaffold → `openaicompat/bridge_test.go` still passes; no OpenRouter factory.

#### 2.2 — Recorded SSE fixtures (≥3 conformance scenarios)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures/text_stream.go` (`openrouterTextStream` — a `[]byte` of one ResponseStart + N TextDelta + terminal chunk + `[DONE]` byte-rendered with `chunkObjectDiscriminator = "chat.completion.chunk"` and `bridgeChunkCreated = 1700000000`); `conformance/fixtures/tool_call.go` (`openrouterToolCallStream` — ToolCallStart + ToolCallDelta + ToolCallEnd shape); `conformance/fixtures/with_usage.go` (terminal chunk carrying `usage` with prompt/completion tokens before `[DONE]`); `conformance/fixtures/with_attribution_headers.go` (the three headers present and absent-variants). Fixture var-blocks are **excluded** from authored risk count per `sdd-phase-common.md §E`.
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/conformance/...`.
- **Spec MUSTs**: R-OR-06 (fixtures support the 5 required capabilities).
- **Rollback**: revert fixtures → only fixture bytes change; no behavior change.

#### 2.3 — `reasoning_details` + `reasoning` (renamed-field) drop test

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/conformance/reasoning_extension_test.go` (RED: fixture carries a `delta.reasoning_details` array of `{"type":"reasoning.text", "text":"..."}` PLUS a `delta.reasoning` string named after OpenRouter's renamed field; the test wraps the bridge factory and asserts (a) no reasoning byte appears in any text event, (b) no reasoning-typed event emitted, (c) the stream reaches its normal completion. This is the OpenRouter-specific twin of `TestReasoningExtensionField_DroppedNotLeakedNotFailed` in `openaicompat/reasoning_absence_test.go` — same mechanism, different fixture field names).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/conformance/...`.
- **Spec MUSTs**: R-OR-06 (reasoning extension sub-scenario).
- **Rollback**: revert test → `reasoning_absence_test.go` in `openaicompat` still covers `reasoning_content`; OpenRouter's renamed fields are uncovered.

#### 2.4 — Capability-record assertion test (`TestOpenRouterAdapter_CapabilityRecordMatchesAI24`)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/conformance/capability_record_test.go` (runs `agenttest.RunConformance(t, openrouterBridgeFactory())`; reads the generated `CapabilityRecord` from `agenttest/conformance_capabilities.go`; asserts `record.Entry(agenttest.CapReasoning).Outcome == OutcomeAbsent`, `TokenCounting == OutcomeAbsent`, `CacheBoundary == OutcomeAbsent` — matching AI-24 §8's expected `absent × 3` table).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/conformance/...`.
- **Spec MUSTs**: R-OR-05 (capability-record assertion).
- **Rollback**: revert test → no mechanical gate against AI-29 reopen trigger #1.

#### 2.5 — PR #2 review-ready commit (final tidy)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/conformance/run_for_test.go` (5 per-capability drivers: `TestOpenRouterAdapter_StreamingText`, `TestOpenRouterAdapter_ToolCalls`, `TestOpenRouterAdapter_CompletionMetadata`, `TestOpenRouterAdapter_Cancellation`, `TestOpenRouterAdapter_Terminal` — each calls `agenttest.RunConformanceFor(t, openrouterBridgeFactory(), <cap>)`); per-commit `make test` clean.
- **Verification**: `cd backend/agent && make test` (full suite green); `make lint` clean.
- **Spec MUSTs**: R-OR-06 (5 required capabilities).
- **Rollback**: revert merge → PR #1 stands alone; openaicompat tests still pass.

---

## PR #3 — Live smoke

- **Branch**: `feat/openrouter-live-smoke`
- **Targets**: `feat/openrouter-conformance-bridge` (PR #2's branch)
- **Rollback boundary**: `git revert -m1 <merge-sha>` → `make test` green; no real-money calls; PR #1/PR #2 unaffected. Sentinel test: `TestSentinelSweep_CatchesDeliberateLogfKeyMutation`.
- **Required path gain**: `backend/agent/src/ai/openaicompat/openrouter/smoke/` (new test-only sibling sub-package) + `.github/workflows/agent-openrouter-smoke.yml`.
- **Spec MUSTs covered**: R-OR-07, R-OR-08.

### Per-commit work-unit breakdown

#### 3.1 — Smoke test skeleton (`smoke_test.go` with `t.Skip` gate)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/smoke/smoke_test.go` (`TestOpenRouterAdapter_LiveSmoke` first line: `if os.Getenv("OPENROUTER_API_KEY") == "" { t.Skip("OPENROUTER_API_KEY not set; live smoke is opt-in") }`; then ONE bounded `provider.Stream(ctx, req)` for 60 s with `context.WithTimeout`; drain via `agenttest.DrainAndRecord`; assert events contain a `ResponseStart`, at least one `TextDelta`, and exactly one `Completion` or terminal error).
- **Verification**: `cd backend/agent && make test` (asserts the skip path runs without the secret; no outbound request).
- **Spec MUSTs**: R-OR-07 (skip path exercised without the secret).
- **Rollback**: revert test → no live smoke; CI unaffected.

#### 3.2 — Sentinel-sweep helper (`Scan(t, captured)`)

- **Files**: `backend/agent/src/ai/openaicompat/openrouter/smoke/sentinel_sweep.go` (`Scan(t, captured []byte) error` — deny-list built at runtime (never contiguous literals): `OPENROUTER_API_KEY` env-var name, the secret's prefix (4 chars), the planted prompt bytes; returns a typed error naming the offending deny-list entry WITHOUT reprinting the credential).
- **Verification**: `cd backend/agent && go test -race -v ./src/ai/openaicompat/openrouter/smoke/...`.
- **Spec MUSTs**: R-OR-08.
- **Rollback**: revert helper → live smoke has no leak guard (regression risk).

#### 3.3 — Workflow YAML (`agent-openrouter-smoke.yml`)

- **Files**: `.github/workflows/agent-openrouter-smoke.yml` (`on: workflow_dispatch` ONLY — no `schedule`, `push`, `pull_request`; reads `OPENROUTER_API_KEY` from `secrets.OPENROUTER_API_KEY`; concurrency group `openrouter-live-smoke` to de-duplicate overlapping runs; one bounded `go test -race -v -run TestOpenRouterAdapter_LiveSmoke ./src/ai/openaicompat/openrouter/smoke/...` step; bounded `timeout-minutes: 5`).
- **Verification**: `cat .github/workflows/agent-openrouter-smoke.yml | grep -E "^(on|workflow_dispatch|schedule|push|pull_request)"` — only `workflow_dispatch` appears.
- **Spec MUSTs**: R-OR-07 (workflow file is dispatch-only).
- **Rollback**: revert YAML → manual `workflow_dispatch` no longer possible; `make test` unaffected.

#### 3.4 — PR #3 review-ready commit (final tidy)

- **Files**: `TestSentinelSweep_CatchesDeliberateLogfKeyMutation` added to `smoke/sentinel_sweep_test.go` (RED: a mutated smoke test calls `t.Logf(key)` with the credential; `Scan` returns an error naming the leak vector; credential is NOT reprinted); `TestOpenRouterAdapter_CredentialRedactionCarriesThrough` asserts `openaicompat.Credential` rendering through `String()`/`GoString()`/`MarshalJSON`/default formatting never contains the token (R-OR-08 sub-scenario 2).
- **Verification**: `cd backend/agent && make test` (full suite green); `make lint` clean.
- **Spec MUSTs**: R-OR-08 (sentinel sweep + redaction).
- **Rollback**: revert merge → `make test` unaffected; smoke sub-package absent.

---

## Spec MUSTs coverage map

| Spec MUST | Coverage PR | Test name |
|-----------|-------------|-----------|
| R-OR-01 (injection-only, no ambient authority) | PR #1 | `TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage` (1.5a) + `wrong_endpoint_rejected` in `wrapper_test.go` (1.1) |
| R-OR-02 (attribution headers wrapper-injected, openaicompat unaware) | PR #1 | `TestOpenRouterAdapter_AttributionHeadersObservedWhenNonEmpty` + `TestOpenRouterAdapter_EmptyStringsSuppressAllHeaders` (1.2) + `TestOpenRouterAdapter_HeadersUnawarenessInOpenAICompatRequest` (1.5b) |
| R-OR-03 (default model `openai/gpt-4o` + deliberate-model field) | PR #1 | `TestOpenRouterAdapter_DefaultModelIsOpenaiGpt4o` + `TestOpenRouterAdapter_ConfigCarriesModelField` (1.3) |
| R-OR-04 (`stream_options.include_usage` set in body) | PR #1 | `TestOpenRouterAdapter_StreamOptionsIncludeUsageIsTrue` (1.3, stub-transport probe of rendered body) |
| R-OR-05 (`CAP-O-01 = absent` under default model) | PR #1 + PR #2 | `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` (1.8) + `TestOpenRouterAdapter_CapabilityRecordMatchesAI24` (2.4) |
| R-OR-06 (conformance bridge runs the suite + reasoning extension) | PR #2 | `TestOpenRouterAdapter_RunConformanceFor*` (5 per-capability, 2.5) + `TestOpenRouterAdapter_ReasoningExtensionField_DroppedNotLeakedNotFailed` (2.3) |
| R-OR-07 (live smoke `t.Skip`-gated + `workflow_dispatch` only) | PR #3 | `TestOpenRouterAdapter_LiveSmoke` (3.1) + YAML `grep` check (3.3) |
| R-OR-08 (credential redaction + sentinel sweep) | PR #1 + PR #3 | `TestOpenRouterAdapter_CredentialRedactionCarriesThrough` (1.4) + `TestSentinelSweep_CatchesDeliberateLogfKeyMutation` (3.4) + `TestOpenRouterAdapter_CredentialRedactionCarriesThrough` (3.4) |
| R-OR-09 (AI-00.3 forward guard stays green) | PR #1 | `TestOpenRouterAdapter_ZeroRequiresGuard` (1.6) |
| R-OR-10 (out-of-scope fence) | PR #1 | `TestOpenRouterAdapter_NegativeShallsFenceFails` (1.7) |

---

## Risks (refined at task level)

| Risk | Task that mitigates |
|------|---------------------|
| **Tracker-chain diff pollution** (Medium) | Each PR's PR body MUST state the parent branch; orchestrator rebases child before review; if a child shows the previous PR's bytes, retarget/rebase until clean (`chained-pr` skill). |
| **800-line budget on PR #1** (Medium) | 1.5a/1.5b split (commit granularity per the design's §8 mitigation); fallback if corrected mid still exceeds 800 = defer `charter_test.go` (task 1.7) to a follow-up commit in the same PR. |
| **AI-29 reopen trigger #1 if default model changes** (Medium) | `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` (1.8) + `TestOpenRouterAdapter_CapabilityRecordMatchesAI24` (2.4). |
| **Attribution-header leakage into openaicompat** (Low) | `TestOpenRouterAdapter_HeadersUnawarenessInOpenAICompatRequest` (1.5b) reads `openaicompat/request.go` as raw bytes and fails on any of the three header names. |
| **`:free` rate-limit flakiness in live smoke** (Low) | Default model is paid (`openai/gpt-4o`); guarded by the design's C2. |
| **Live-smoke secret leakage via `t.Logf`** (Low) | `TestSentinelSweep_CatchesDeliberateLogfKeyMutation` (3.4) + `Scan` helper (3.2). |
| **AI-00.3 forward-guard regression** (Very Low) | `TestOpenRouterAdapter_ZeroRequiresGuard` (1.6) — defense in depth. |
| **`error_type` vocabulary mismatch** (Low, deferred) | Out of scope per R-OR-10; `charter_test.go` (1.7) fails on any `"error_type"` literal in non-test sources. |
| **Header mutation race** (Low) | `http.Header.Clone()` is documented thread-safe; `RoundTrip` is serial per request. |
| **`net/http` miscounted as non-stdlib** (Very Low) | `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` filters via `.Standard` (per `import_boundary_test.go`'s load-bearing detail). |
| **AI-38 capability record drift if `CAP-O-01` case is added** (Low) | `applyDeclaredAbsences` (per `agenttest/conformance_suite.go:330`) handles additional cases. |
| **Wrapper path ambiguity in the user prompt** (Low) | Surface in reply: prompt's `backend/agent/src/ai/openrouter/` is read as the package directory, not a re-decision of D1. |
| **Workflow filename divergence in the user prompt** (Low) | Use design's canonical `.github/workflows/agent-openrouter-smoke.yml`; prompt's `openrouter-live-smoke.yml` is read as a shorthand. |

## Out of scope (restated)

- No Anthropic native adapter (no `*anthropic*` filenames in new sub-package; existing `ai.WithProviderExtension("anthropic", ...)` test fixtures are generic and unchanged).
- No AI-32 widening for OpenRouter's `error_type` discriminator.
- No `reasoning_content` / `reasoning` / `reasoning_details` emission as `ai.EventKindReasoningDelta` (R-OR-10 negative SHALL).
- No new `go.mod` require (R-OR-09; `go.mod` stays at 3 lines, zero requires).
- No AI-29 reopen (default `openai/gpt-4o` preserves struck verdict).
- No Layer 3 (`src/coding`) composition root.
- No AI-40 capability-matrix publication.
- No Wave-2 carryovers (AI-41).

## References

- **Design** (DRAFT): `openspec/changes/add-openrouter-first-provider/design.md` · Engram **#2574** (`sdd/add-openrouter-first-provider/design`).
- **Spec**: `openspec/changes/add-openrouter-first-provider/specs/ai-openrouter-first-provider/spec.md` · Engram **#2573**.
- **Proposal**: `openspec/changes/add-openrouter-first-provider/proposal.md` · Engram **#2570**.
- **Explore**: `openspec/changes/add-openrouter-first-provider/explore.md` · Engram **#2568** (topic `sdd/add-openrouter-first-provider/explore`).
- **Wrapper-placement decision**: Engram **#2571** (`decision/openrouter-wrapper-placement`).
- **Shipped specs (composed, NOT modified)**: `openspec/specs/ai-provider-client/spec.md` (AI-25), `openspec/specs/ai-provider-conformance-suite/spec.md` (AI-23/38), `openspec/specs/ai-stream-testkit/spec.md` (AI-22), `openspec/specs/ai-model-provider/spec.md` (AI-20).
- **Shipped code (read, NOT modified)**: `backend/agent/src/ai/openaicompat/{client,request,credential,bridge_test,ambient_authority_test,reasoning_absence_test,credential_scan_test}.go`, `backend/agent/src/ai/import_boundary_test.go`, `backend/agent/src/ai/provider.go`, `backend/agent/go.mod`.
- **AI-24 vendor/transport pre-decision**: Engram **#2432**.
- **AI-29 struck verdict**: `openspec/changes/archive/2026-08-04-cachicamas-ai-provider-reasoning-stream/decision.md`.
- **Backend test runner**: Engram **#2055** (`cd backend/agent && make test` = `go test -race -v ./...`).
- **ADR 0005** (agent module dependency): `docs/adr/0005-promote-agent-stack-to-own-module.md`.
- **doc 0002**: `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` — AI-25 lines 1447–1491, AI-38 lines 2241–2277, AI-39 lines 2279–2299.
- **Project rules**: `openspec/AGENTS.md` (TDD on, `make test`, conventional commits, no AI attribution).
- **OpenSpec rules**: `openspec/config.yaml` `rules.tasks` and `rules.apply` (`tdd: true`).
- **Skills applied**: `sdd-tasks` (sdd-phase-common §E review-workload guard), `chained-pr` (feature-branch-chain topology, ≤60 min PR rule, polluted-diff rule), `work-unit-commits` (every commit build-green + test-green + narrow intent + rollback-friendly).
