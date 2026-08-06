# Design: OpenRouter as the first concrete Layer 1 AI provider

> **Change**: `add-openrouter-first-provider`
> **Capability**: `ai-openrouter-first-provider` (new)
> **Status**: DRAFT
> **Date**: 2026-08-06
> **Artifact store**: both — file at this path AND Engram topic `sdd/add-openrouter-first-provider/design`
> **Composes (read-only, NOT modified)**: `ai-provider-client` (AI-25) · `ai-provider-conformance-suite` (AI-23/38) · `ai-stream-testkit` (AI-22) · `ai-model-provider` (AI-20)
> **Locked preflight (orchestrator-cached)**: `delivery_strategy=auto-chain` · `chain_strategy=feature-branch-chain` · `review_budget_lines=800` · `change_name=add-openrouter-first-provider`
> **Links**: [spec](./specs/ai-openrouter-first-provider/spec.md) · [proposal](./proposal.md) · [explore](./explore.md) · Engram `#2573` (spec) · `#2570` (proposal) · `#2571` (wrapper-placement) · `#2568` (explore)

---

## 1. Technical Approach

Three chained PRs under a no-merge tracker (`tracker/add-openrouter-first-provider`) concretize **OpenRouter** as the first vendor on the shipped, vendor-agnostic `openaicompat` adapter. The wrapper is a thin sub-100-line constructor + a wrapper-owned `http.RoundTripper`; the conformance bridge reuses the existing `openaicompat/bridge_test.go` factory pattern (test-only, sibling sub-package) with the OpenRouter-shaped envelope; the live smoke is `t.Skip`-gated on `OPENROUTER_API_KEY` and runs only via `workflow_dispatch`. Every spec SHALL is honored as a concrete implementation strategy in §5.

The change **composes** `openaicompat.Config`, `agenttest.RunConformance`, `agenttest.Factory`, and `openaicompat.Credential`. It does NOT widen any shipped package's surface. It adds zero `go.mod` requires (NFR-APC-A carried into this change).

---

## 2. Architecture Overview

```text
                                       Layer 3 (src/coding, future)
                                                  │
                                                  ▼  reads OPENROUTER_API_KEY
                                            Layer 3 wiring
                                                  │ (opaque bearer string)
                                                  ▼
                          ┌─────────────────────────────────────────────────┐
                          │  Layer 1 — package openrouter                   │
                          │  backend/agent/src/ai/openaicompat/openrouter/   │
                          │                                                 │
                          │  provider.go (Config, NewProvider, header       │
                          │             attach, no ambient authority)        │
                          │       │                                         │
                          │       ▼                                         │
                          │  transport.go (wrapper-owned http.RoundTripper   │
                          │                that injects HTTP-Referer,        │
                          │                X-OpenRouter-Title,               │
                          │                X-OpenRouter-Categories)          │
                          └────────┬────────────────────────────────────────┘
                                   │ constructs Config
                                   ▼
                          ┌─────────────────────────────────────────────────┐
                          │  Layer 1 — package openaicompat (UNCHANGED)     │
                          │  request.go sets ONLY Authorization +           │
                          │                Content-Type + Accept            │
                          │  body.go sets stream_options.include_usage=true │
                          │  decoder.go drops reasoning / reasoning_details │
                          └────────┬────────────────────────────────────────┘
                                   │ net/http (stdlib only)
                                   ▼
                          OpenRouter /api/v1/chat/completions
```

**Conformance bridge plug-in** (PR #2):

```text
  agenttest.RunConformance(t, openrouterBridgeFactory())
       │
       ▼
  openrouter/conformance/bridge_test.go: bridgeFactory.New(tb, scripts...)
       │ 1. renders scripts with OpenRouter envelope (3 attribution headers in stub transport)
       │ 2. stands up httptest.Server with the three headers
       │ 3. builds an openrouter.NewProvider(Config{...})
       │ 4. returns the *openaicompat.Client as ai.ModelProvider
       ▼
  agenttest.RunConformanceFor runs each registered case → CapabilityRecord
       ▼
  openrouter/conformance/capability_record_test.go
       │
       ▼
  Compares against AI-24 §8's expected "absent × 3" — fails loudly if CAP-O-01
  is anything other than absent under default model openai/gpt-4o (R-OR-05)
```

**Live smoke plug-in** (PR #3):

```text
  go test ./.../openrouter/smoke -run TestOpenRouterAdapter_LiveSmoke
       │
       ▼
  t.Skip if os.Getenv("OPENROUTER_API_KEY") == ""   (no outbound request, no log)
       │
       ▼
  ONE bounded (60s) request to openrouter.ai/api/v1/chat/completions
       │
       ▼
  drain → check start, at least one content event, exactly one terminal
       │
       ▼
  sentinel sweep over t.Logf output + captured error messages
  (openrouter/smoke/sentinel_sweep_test.go catches a deliberate t.Logf(key) mutation)
```

---

## 3. Architecture Decisions

| # | Choice | Alternatives considered | Rationale |
|---|---|---|---|
| D1 | **Wrapper placement** = sibling sub-package `backend/agent/src/ai/openaicompat/openrouter/` | (a) embed inside `openaicompat/`; (b) top-level `src/ai/openrouter/` | (a) couples wrapper to openaicompat test state and widens AI-25.2 scan scope to OpenRouter-specific header code; (b) splits the `openaicompat/` namespace across two locations, contradicting the layered-module convention. Engram #2571 records this; Q3 in the proposal. |
| D2 | **Attribution-header injection** = wrapper-owned `http.RoundTripper` that wraps the openaicompat-built transport | (a) widen `openaicompat/request.go`; (b) pass headers via `http.Client.Header` mutation | (a) widens openaicompat's documented "Authorization + Content-Type only" surface for one vendor's quirk; (b) http.Client has no Header field; RoundTripper is the only correct seam. C3 in the proposal. |
| D3 | **Default model = `openai/gpt-4o`** | (a) reasoning-capable model; (b) `:free` variant | (a) reopens AI-29 under trigger #1; (b) rate-limit flakiness (20 req/min). C1, C2, Q1 in the proposal. |
| D4 | **`stream_options.include_usage` stays set** | (a) drop the field on OpenRouter | (a) diverges the wire body from vLLM/Ollama/LocalAI — one wire body for all openaicompat-target vendors is the openaicompat posture. Field is deprecated + harmless on OpenRouter. C4 in the proposal. |
| D5 | **Conformance bridge lives at `openrouter/conformance/bridge_test.go`** (test-only), not in `package openrouter` itself | (a) put bridge inside `openrouter/`; (b) put bridge in `package openaicompat` | (a) makes the OpenRouter-shaped factory visible to the wrapper's own package, which keeps the AI-25.2 call-site scan scope clean; (b) reuses the `openaicompat/bridge_test.go` shape exactly — minimal new pattern. |
| D6 | **Three chained PRs under no-merge tracker** (auto-chain, feature-branch-chain) | (a) single PR; (b) stacked PRs to main | (a) over-runs the 800-line budget per corrected forecast; (b) requires the wrapper to integrate before main, which conflicts with the "revert PR #1 → no behavior change" rollback. Q6 in the proposal. |
| D7 | **Live smoke is `t.Skip`-gated, NOT build-tag-gated** | (a) `//go:build live_smoke` | (a) build-tag-gated smoke is invisible to `make test`; CI's normal `make test` MUST exercise the skip path (R-OR-07 scenario 1). The skip IS the run for `make test`; `workflow_dispatch` is the run for the secret. |
| D8 | **Sentinel-sweep helper API**: `openrouter/smoke.Scan(t, captured)` returns `error` (no sentinel re-print) | (a) inline `strings.Contains` in test | (a) a single helper that catches the three deny-list entries (literal env name, secret prefix 4 chars, planted prompt bytes) keeps R-OR-08 mechanical and the test mutation testable. |

---

## 4. Data Flow

**PR #1 outbound request** (one minimal `Stream` call):

```
Provider.Stream(ctx, req)
   │
   ▼
openaicompat.Client.stream(ctx, req)         (package openaicompat, unchanged)
   │
   ▼
openaicompat.Client.newRequest(ctx, body, "chat", "completions")
   │     ↑ sets Authorization, Content-Type, Accept (UNCHANGED surface)
   ▼
http.Request
   │
   ▼
http.Client.Do(req)
   │
   ▼
http.Client.Transport  =  wrapperOwnedRoundTripper(http.Transport)
                                  │  RoundTrip(req):
                                  │    1. clone req
                                  │    2. if HTTPReferer != ""  → req.Header.Set("HTTP-Referer", ...)
                                  │    3. if XTitle != ""       → req.Header.Set("X-OpenRouter-Title", ...)
                                  │                                              + req.Header.Set("X-Title", ...)
                                  │    4. if XCategories != ""  → req.Header.Set("X-OpenRouter-Categories", ...)
                                  │    5. return base.RoundTrip(req)
                                  ▼
                            net/http → openrouter.ai:443
```

Empty strings suppress the corresponding header (R-OR-02 scenario 2). Both `X-OpenRouter-Title` and `X-Title` are set when `XTitle != ""` (OpenRouter accepts both as aliases).

**PR #2 conformance run**:

```
TestOpenRouterAdapter_RunConformance
   │
   ▼
agenttest.RunConformance(t, openrouterBridgeFactory())
   │
   ▼
applyDeclaredAbsences → CAP-O-01, CAP-O-02, CAP-O-03 → OutcomeAbsent
   │
   ▼
runRegisteredCase for each case in conformanceRegistry
   │  - CapStreamingText → textOrderingCase, textEmptyCompletionCase (S-CNF-040…)
   │  - CapToolCalls     → 4 cases (S-ATL-059)
   │  - CapCompletionMetadata → finishReasonExhaustivenessCase, usageAbsentVsZeroCase
   │  - CapCancellation  → cancellation cases
   │  - CapTerminal      → exactly-one-terminal invariant
   ▼
returns CapabilityRecord (8 entries, all satisfied or absent)
   │
   ▼
TestOpenRouterAdapter_CapabilityRecordMatchesAI24
   │  compares got vs want[absent, absent, absent, satisfied, satisfied, satisfied, satisfied, satisfied]
   ▼
TestReasoningExtensionField_DroppedNotLeakedNotFailed
   │  drives a transcript carrying delta.reasoning_details through the bridge
   │  asserts: no text event carries reasoning bytes, no reasoning-typed event emitted, terminal is Completion
```

---

## 5. Spec MUST Implementation Strategy

Every R-OR-NN mapped to a concrete mechanism. Tests written BEFORE production code (RED-GREEN-REFACTOR, per `openspec/AGENTS.md`).

| Req | MUST | Concrete implementation strategy |
|---|---|---|
| **R-OR-01** | Wrapper construction is injection-only; no `os.Getenv`/`os/exec`/filesystem | `openrouter.Config` exposes `Credential openaicompat.Credential`, `HTTPClient *http.Client`, `HTTPReferer string`, `XTitle string`, `XCategories string`, `Model string`. Constructor: `NewProvider(Config) (ai.ModelProvider, error)`. **New test** `openrouter/ambient_authority_test.go` mirrors `openaicompat/ambient_authority_test.go`'s call-site scan over the new sub-package's non-test sources only; `forbiddenAmbientAuthorityPackages` table lifted verbatim. **Wrong endpoint** path: `NewProvider` rejects any `Endpoint != "https://openrouter.ai/api/v1"` with `ai.Invalid(ai.ErrMalformed, ai.At("endpoint"))` — no outbound request (asserted via stub transport probe). |
| **R-OR-02** | Attribution headers wrapper-injected, NOT openaicompat-injected | `openrouter/transport.go` defines `type attributionRoundTripper struct { base http.RoundTripper; referer, xTitle, xCategories string }` implementing `http.RoundTripper`. `RoundTrip` clones the request's `http.Header` (header is a map, must be copied), sets each non-empty header, then delegates to `base.RoundTrip`. Constructor builds `&http.Client{Transport: attributionRoundTripper{base: openaicompat-built-transport, ...}}`. **Negative proof** `openrouter/headers_unawareness_test.go` reads `openaicompat/request.go` and asserts the literal strings `"HTTP-Referer"`, `"X-OpenRouter-Title"`, `"X-OpenRouter-Categories"` do NOT appear anywhere in that file's bytes. |
| **R-OR-03** | Default model `openai/gpt-4o`; deliberate-model field | `Config.Model string` field defaults to `"openai/gpt-4o"` when zero value passed (constructor checks for empty and applies default). Bridge factory declares `factory.model = openrouterDefaultModel = "openai/gpt-4o"` as a const; test `TestOpenRouterBridge_DeclaresModelOpenAIGpt4o` reads it. |
| **R-OR-04** | `stream_options.include_usage` always set | This field is set by `openaicompat/body.go`, not by the wrapper — verified unchanged by `openrouter/wrapper_test.go`'s `TestOpenRouterProvider_DoesNotSuppressStreamOptionsIncludeUsage`, which reads the rendered body through a stub transport and asserts `stream_options.include_usage == true`. No new code in openaicompat. |
| **R-OR-05** | Capability record `CAP-O-01 = absent` under default model | `openrouter/conformance/capability_record_test.go` defines `TestOpenRouterAdapter_CapabilityRecordMatchesAI24`: runs `agenttest.RunConformance(t, openrouterBridgeFactory())`, reads `record.Entry(agenttest.CapReasoningContent)`, asserts `outcome == OutcomeAbsent`. **Silent-default-swap guard**: `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` parses `Config.Model`'s default and the `openrouterDefaultModel` constant, and asserts they are equal AND equal to the spec's literal `openai/gpt-4o`. If a future contributor changes the default, this fails until the spec amendment lands. |
| **R-OR-06** | Conformance bridge runs the suite against the OpenRouter wrapper; `TestReasoningExtensionField_DroppedNotLeakedNotFailed` cited | `openrouter/conformance/bridge_test.go`'s factory mirrors `openaicompat/bridge_test.go`'s `conformanceBridgeFactory()` verbatim except (a) it injects the three attribution headers into the stub server, (b) it declares the three optional capabilities non-nil `false`. **5 conformance scenarios** (PR #2): `CapStreamingText` (textOrderingCase, textEmptyCompletionCase), `CapToolCalls` (4 cases), `CapCompletionMetadata` (finishReasonExhaustivenessCase + usageAbsentVsZeroCase), `CapCancellation`, `CapTerminal`. Reasoning extension field: a new recorded fixture (PR #2) carries `delta.reasoning_details` and `delta.reasoning`; `TestReasoningExtensionField_DroppedNotLeakedNotFailed` is re-run against the OpenRouter bridge and cited in the test docstring as covering OpenRouter's renamed extension fields (the openaicompat test covers `reasoning_content`; the OpenRouter test covers `reasoning_details`/`reasoning`). |
| **R-OR-07** | Live smoke `t.Skip`-gated on `OPENROUTER_API_KEY`; workflow `workflow_dispatch` only | `openrouter/smoke/smoke_test.go` `TestOpenRouterAdapter_LiveSmoke` first line: `if os.Getenv("OPENROUTER_API_KEY") == "" { t.Skip("OPENROUTER_API_KEY not set; live smoke is opt-in") }`. **No build tag** — the skip IS the gate (see D7). `cd backend/agent && make test` runs the test, exercises the skip path, no outbound request. `.github/workflows/agent-openrouter-smoke.yml`: `on: workflow_dispatch` ONLY — no `schedule`, no `push`, no `pull_request`. Reads secret `${{ secrets.OPENROUTER_API_KEY }}`, sets env, runs `go test -race -v -run TestOpenRouterAdapter_LiveSmoke ./backend/agent/src/ai/openaicompat/openrouter/smoke/`. Concurrency group `openrouter-live-smoke` prevents overlapping dispatches. |
| **R-OR-08** | `openaicompat.Credential` redaction extends to live smoke logging | `openrouter/smoke/sentinel_sweep.go` exposes `Scan(t *testing.T, captured []byte) error`. Deny-list entries (built at runtime by string concatenation, never as contiguous literals, mirroring `openaicompat/credential_scan_test.go`'s discipline): literal `"OPENROUTER_API_KEY"`, the secret prefix (4 chars), the planted prompt bytes. The smoke test plants a known prompt into the request body; `Scan` runs at the end of the test on `t.Logf` output and any captured error. **Mutation test** `TestSentinelSweep_CatchesDeliberateLogfKeyMutation` in `openrouter/smoke/sentinel_sweep_test.go` calls `t.Logf(key)` deliberately; `Scan` must fail the test, naming the leak vector WITHOUT reprinting the credential. |
| **R-OR-09** | AI-00.3 forward guard stays green | `go.mod` UNCHANGED — wrapper imports only `net/http` (stdlib), `github.com/cachicamas/backend/agent/src/ai` (in-module), `github.com/cachicamas/backend/agent/src/ai/openaicompat` (in-module), `github.com/cachicamas/backend/agent/src/agenttest` (in-module, test-only). `TestLayer1_ModuleHasNoDependencies_ZeroRequires` passes unmodified (no new `require` lines). `allowedNonStdlibPrefixes` UNCHANGED — still one entry, the module itself. The wrapper does NOT introduce a `net/http` allowlist entry; `net/http` is stdlib. |
| **R-OR-10** | Out-of-scope fence (no Anthropic, no `error_type`, no `reasoning_content`, no `go.mod` require) | Mechanical barriers: (a) **file naming** — no file in the new sub-package may match `*anthropic*`; `openrouter/charter_test.go` walks the directory and fails on any such match. (b) **error_type absence** — no file in the new sub-package may import `openaicompat/failure_map.go`'s `error_type` symbol; the `error_type` string does not appear as a literal in any production source. (c) **reasoning_content emission** — no wrapper code renders `reasoning`/`reasoning_details`/`reasoning_content` as an `ai.EventKindReasoningDelta` payload; the existing `openaicompat/chunk.go`'s plain `json.Unmarshal` already drops the field, and the bridge test asserts drop-not-leak on OpenRouter-shaped transcripts. (d) **go.mod** — pinned to zero requires; if a future PR adds a `require`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires` fails immediately. |

---

## 6. File Changes (per PR)

### PR #1 — `feat/openrouter-wrapper` → `tracker/add-openrouter-first-provider`

| File | Action | Description |
|---|---|---|
| `backend/agent/src/ai/openaicompat/openrouter/doc.go` | Create | Package doc: first-concrete-vendor = OpenRouter; endpoint fixed; injection-only construction; the three attribution headers; the deliberate-model field; non-blocking reasoning field dropped silently; no ambient authority; AI-29 struck verdict preserved under default model. |
| `backend/agent/src/ai/openaicompat/openrouter/provider.go` | Create | `Config` (Credential, HTTPClient, HTTPReferer, XTitle, XCategories, Model), `NewProvider(Config) (ai.ModelProvider, error)`. Constructor: validates endpoint == `https://openrouter.ai/api/v1` (R-OR-01 scenario 2); sets default model when zero; builds `&http.Client{Transport: attributionRoundTripper{base: openaicompat-built, ...}}`; wraps via `openaicompat.New`. |
| `backend/agent/src/ai/openaicompat/openrouter/transport.go` | Create | `type attributionRoundTripper`, `RoundTrip` impl. Header-clone + set pattern. ~30 LOC. |
| `backend/agent/src/ai/openaicompat/openrouter/endpoint.go` | Create | Constant `openrouterBaseURL = "https://openrouter.ai/api/v1"`. |
| `backend/agent/src/ai/openaicompat/openrouter/headers.go` | Create | Helper `applyAttribution(req *http.Request, referer, xTitle, xCategories string)` — single place that owns the header names. ~15 LOC. |
| `backend/agent/src/ai/openaicompat/openrouter/wrapper_test.go` | Create | R-APC-001 stub-transport probe for the three headers (all empty / each non-empty); R-OR-01 wrong-endpoint rejection; R-OR-02 negative proof (`TestOpenRouter_DoesNotWidenOpenAICompatRequestSurface`); R-OR-03 default-model field; R-OR-04 `stream_options.include_usage` present. ~150 LOC. |
| `backend/agent/src/ai/openaicompat/openrouter/ambient_authority_test.go` | Create | Mirrors `openaicompat/ambient_authority_test.go` — call-site scan over the new sub-package's non-test sources. Verifies the AI-25.2 guard extends to the wrapper. ~80 LOC (mostly lifted). |
| `backend/agent/src/ai/openaicompat/openrouter/charter_test.go` | Create | Walks the new sub-package; fails on `*anthropic*` filename, on the literal `"error_type"` outside tests, on any new `require`-shaped string. Mechanical fence for R-OR-10. ~40 LOC. |
| `backend/agent/src/ai/openaicompat/openrouter/headers_unawareness_test.go` | Create | Negative proof: reads `openaicompat/request.go` as bytes; fails if `"HTTP-Referer"`, `"X-OpenRouter-Title"`, or `"X-OpenRouter-Categories"` appear. ~30 LOC. |
| `.golangci.yml` (project root) | UNCHANGED | (no project root .golangci.yml in repo; lint runs with default config) |

### PR #2 — `feat/openrouter-conformance-bridge` → PR #1's branch

| File | Action | Description |
|---|---|---|
| `backend/agent/src/ai/openaicompat/openrouter/conformance/bridge_test.go` | Create | `conformanceBridgeFactory()` mirrors `openaicompat/bridge_test.go`'s pattern. Adds the three attribution headers to the stub transport. `Reasoning/TokenCounting/CacheBoundary` declared non-nil `false`. **Recorded SSE fixtures** are var blocks (`var openrouterTextOnlyTranscript = "data: {...}\n\ndata: {...}\n\ndata: [DONE]\n\n"`) — these are goldens, **excluded** from authored risk count per `sdd-phase-common.md §E`. |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/capability_record_test.go` | Create | `TestOpenRouterAdapter_CapabilityRecordMatchesAI24` (R-OR-05). Asserts `record.Entry(agenttest.CapReasoningContent).Outcome == OutcomeAbsent`, etc. Plus `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment`. |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/reasoning_extension_test.go` | Create | Recorded fixture carrying `delta.reasoning_details` + `delta.reasoning`; `TestReasoningExtensionField_DroppedNotLeakedNotFailed` re-runs the openaicompat idiom against OpenRouter-shaped wire (R-OR-06). |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/run_for_test.go` | Create | `TestOpenRouterAdapter_RunConformanceFor_CapStreamingText` + four more (`CapToolCalls`, `CapCompletionMetadata`, `CapCancellation`, `CapTerminal`). |

### PR #3 — `feat/openrouter-live-smoke` → PR #2's branch

| File | Action | Description |
|---|---|---|
| `backend/agent/src/ai/openaicompat/openrouter/smoke/smoke_test.go` | Create | `TestOpenRouterAdapter_LiveSmoke` — `t.Skip`-gated, ONE bounded request (60s timeout), asserts `ResponseStart` + at least one `TextDelta` + exactly one `Completion`. |
| `backend/agent/src/ai/openaicompat/openrouter/smoke/sentinel_sweep.go` | Create | `Scan(t, captured []byte) error` — denies the literal `"OPENROUTER_API_KEY"`, the 4-char secret prefix, and the planted prompt bytes. Failures name the leak vector without reprinting the credential. |
| `backend/agent/src/ai/openaicompat/openrouter/smoke/sentinel_sweep_test.go` | Create | `TestSentinelSweep_CatchesDeliberateLogfKeyMutation` (R-OR-08 scenario 1) + `TestSentinelSweep_OpenaicompatCredentialStringHolds` (R-OR-08 scenario 2). |
| `.github/workflows/agent-openrouter-smoke.yml` | Create | `on: workflow_dispatch` ONLY. Concurrency group `openrouter-live-smoke`. Reads `${{ secrets.OPENROUTER_API_KEY }}`. Bounded 5-minute job timeout. |

**Recorded SSE fixture files** (`openrouter/conformance/fixtures/`) excluded from authored risk count per `sdd-phase-common.md §E`:
- `text_only_transcript.go` (`var openrouterTextOnlyTranscript = "..."`)
- `tool_call_transcript.go`
- `completion_metadata_transcript.go`
- `cancellation_transcript.go`
- `reasoning_extension_transcript.go` (carries `delta.reasoning_details` + `delta.reasoning`)

---

## 7. Type and API Design (pseudo-Go)

```go
package openrouter

import (
    "net/http"
    "github.com/cachicamas/backend/agent/src/ai"
    "github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

const (
    openrouterBaseURL = "https://openrouter.ai/api/v1"
    openrouterDefaultModel = "openai/gpt-4o"
)

// Config configures an OpenRouter Provider. Every field is set by
// injection; the constructor reads nothing from the environment.
type Config struct {
    Credential   openaicompat.Credential // opaque bearer; required
    HTTPClient   *http.Client            // optional; nil => wrapper builds its own bounded client
    HTTPReferer  string                  // "" => header suppressed
    XTitle       string                  // "" => X-OpenRouter-Title and X-Title both suppressed
    XCategories  string                  // "" => X-OpenRouter-Categories suppressed
    Model        string                  // "" => openrouterDefaultModel
}

// NewProvider returns an ai.ModelProvider wrapping an openaicompat.Client
// with the OpenRouter base URL and the injected attribution headers
// attached via a wrapper-owned http.RoundTripper.
func NewProvider(cfg Config) (ai.ModelProvider, error) { /* ... */ }

// attributionRoundTripper is the wrapper's own http.RoundTripper; it adds
// the three OpenRouter-only attribution headers and delegates.
type attributionRoundTripper struct {
    base       http.RoundTripper
    referer    string
    xTitle     string
    xCategories string
}

func (rt attributionRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
    // Clone header map (http.Header is a map[string][]string — copy the map AND the slices).
    cloned := req.Header.Clone()
    if rt.referer != "" { cloned.Set("HTTP-Referer", rt.referer) }
    if rt.xTitle != "" {
        cloned.Set("X-OpenRouter-Title", rt.xTitle)
        cloned.Set("X-Title", rt.xTitle)
    }
    if rt.xCategories != "" { cloned.Set("X-OpenRouter-Categories", rt.xCategories) }
    clonedReq := req.Clone(req.Context())
    clonedReq.Header = cloned
    return rt.base.RoundTrip(clonedReq)
}
```

**Provider interface satisfaction**: `ai.ModelProvider` requires `Stream(ctx, req) (<-chan Event, error)`. The wrapper returns the `*openaicompat.Client` value from `openaicompat.New(Config{...})`; `openaicompat.Client` already satisfies `ai.ModelProvider` (verified by the openaicompat test surface). The wrapper itself is therefore a **factory** (a function), not a new type satisfying `ai.ModelProvider`. The AI-20 signature guard passes unmodified because the guard parses `src/ai/provider.go`, not the wrapper.

---

## 8. PR Chain Topology (feature-branch-chain)

```
main
 └─ tracker/add-openrouter-first-provider    [draft PR, no-merge]
      ↑ PR #1 base
      └─ feat/openrouter-wrapper
           ↑ PR #2 base
           └─ feat/openrouter-conformance-bridge
                ↑ PR #3 base
                └─ feat/openrouter-live-smoke
```

Tracker PR opens first with a body that lists the three child PRs and cites this proposal. Each child rebases onto its immediate parent before review; polluted diffs are base bugs (chained-pr skill rule).

### Commits per PR (work-unit-commits skill)

**PR #1 — `feat/openrouter-wrapper`** (target branch: `tracker/add-openrouter-first-provider`)

| # | Commit (Conventional Commits, no AI attribution) | Work unit | Files | Verification |
|---|---|---|---|---|
| 1.1 | `chore(agent): add openrouter sub-package doc stub` | Place the directory + doc.go | `openrouter/doc.go` | `go build ./...` |
| 1.2 | `test(agent): add openrouter call-site scan fixture` | RED: ambient_authority_test.go's `forbiddenAmbientAuthorityPackages` lifted; guard runs against a `.` with NO production code yet → passes vacuously | `openrouter/ambient_authority_test.go` | `cd backend/agent && make test` |
| 1.3 | `feat(agent): openrouter wrapper Config + NewProvider` | RED→GREEN: TestOpenRouterProvider_NewProvider_InjectedValuesUsedAtStubTransport fails first (no impl); then provider.go implements Config + NewProvider; minimal `openaicompat.New(Config{Endpoint: openrouterBaseURL, Credential: cfg.Credential, HTTPClient: cfg.HTTPClient})` | `openrouter/provider.go`, `openrouter/wrapper_test.go` (stub-transport probe) | `cd backend/agent && make test` |
| 1.4 | `feat(agent): openrouter wrapper rejects wrong endpoint (R-OR-01)` | RED→GREEN: TestOpenRouterProvider_NewProvider_WrongEndpointFailsWithoutOutbound → NewProvider validates endpoint == openrouterBaseURL, returns `ai.Invalid(ai.ErrMalformed, ai.At("endpoint"))` | `openrouter/provider.go` (delta), `openrouter/wrapper_test.go` | `make test` |
| 1.5 | `feat(agent): openrouter attribution headers via wrapper-owned RoundTripper (R-OR-02)` | RED→GREEN: TestOpenRouterProvider_AttributionHeadersAttachedWhenNonEmpty → transport.go impl; TestOpenRouterProvider_AttributionHeadersSuppressedWhenEmpty → same impl with empty strings | `openrouter/transport.go`, `openrouter/headers.go`, `openrouter/wrapper_test.go` | `make test` |
| 1.6 | `test(agent): openrouter unawareness of attribution headers in openaicompat (R-OR-02 negative proof)` | Reads `openaicompat/request.go` as bytes; fails if any of the three header names appears | `openrouter/headers_unawareness_test.go` | `make test` |
| 1.7 | `feat(agent): openrouter default model = openai/gpt-4o (R-OR-03)` | RED→GREEN: TestOpenRouterProvider_ConfigCarriesModelFieldDefaultsToOpenAIGpt4o → provider.go applies default when Config.Model == "" | `openrouter/provider.go` (delta), `openrouter/wrapper_test.go` (delta) | `make test` |
| 1.8 | `test(agent): openrouter stream_options.include_usage stays set (R-OR-04)` | Stub-transport probe reads rendered body; asserts `stream_options.include_usage == true` (this is an openaicompat-side property the wrapper inherits) | `openrouter/wrapper_test.go` (delta) | `make test` |
| 1.9 | `test(agent): openrouter charter fence (R-OR-10)` | Walks new sub-package; fails on `*anthropic*` filenames, on `"error_type"` literal in non-test sources | `openrouter/charter_test.go` | `make test` |
| 1.10 | `test(agent): openrouter zero go.mod requires still holds (R-OR-09)` | Inline assertion: `make test` exits green; no new `require` lines | (verified by existing test) | `make test`, `cd backend/agent && head -3 go.mod` |

Build-green + test-green at every commit. ~10 commits, each ~50-300 LOC authored.

**PR #2 — `feat/openrouter-conformance-bridge`** (target branch: `feat/openrouter-wrapper`)

| # | Commit | Work unit | Verification |
|---|---|---|---|
| 2.1 | `test(agent): openrouter bridge factory declared with model openai/gpt-4o (R-OR-03 bridge half)` | bridge_test.go's `openrouterDefaultModel = "openai/gpt-4o"`; TestOpenRouterBridge_DeclaresModelOpenAIGpt4o | `make test` |
| 2.2 | `test(agent): openrouter bridge factory declares three optional caps false (R-OR-05)` | bridge_test.go's factory declares `Reasoning/TokenCounting/CacheBoundary` non-nil false | `make test` |
| 2.3 | `test(agent): openrouter conformance bridge text transcript` | Recorded text-only transcript var block; TestOpenRouterAdapter_RunConformanceFor_CapStreamingText | `make test` |
| 2.4 | `test(agent): openrouter conformance bridge tool-call transcript` | Recorded tool-call transcript var block; TestOpenRouterAdapter_RunConformanceFor_CapToolCalls | `make test` |
| 2.5 | `test(agent): openrouter conformance bridge completion-with-usage transcript` | Recorded metadata transcript; TestOpenRouterAdapter_RunConformanceFor_CapCompletionMetadata | `make test` |
| 2.6 | `test(agent): openrouter conformance bridge cancellation transcript` | Recorded cancellation transcript; TestOpenRouterAdapter_RunConformanceFor_CapCancellation | `make test` |
| 2.7 | `test(agent): openrouter conformance bridge terminal invariant` | TestOpenRouterAdapter_RunConformanceFor_CapTerminal | `make test` |
| 2.8 | `test(agent): openrouter reasoning extension dropped not leaked not failed (R-OR-06)` | Recorded transcript carrying `delta.reasoning_details` + `delta.reasoning`; TestReasoningExtensionField_DroppedNotLeakedNotFailed (mirrors openaicompat's, with renamed fields) | `make test` |
| 2.9 | `test(agent): openrouter capability record equals AI-24 §8 absent × 3 (R-OR-05)` | TestOpenRouterAdapter_CapabilityRecordMatchesAI24 runs RunConformance, compares entry by entry against expected [absent, absent, absent, satisfied, satisfied, satisfied, satisfied, satisfied] | `make test` |
| 2.10 | `test(agent): openrouter default-model swap fails until spec amendment (R-OR-05 scenario 2)` | TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment reads `openrouterDefaultModel` constant and `Config.Model`'s default; asserts they agree AND equal `openai/gpt-4o` | `make test` |

**PR #3 — `feat/openrouter-live-smoke`** (target branch: `feat/openrouter-conformance-bridge`)

| # | Commit | Work unit | Verification |
|---|---|---|---|
| 3.1 | `feat(agent): openrouter live smoke skip path (R-OR-07)` | TestOpenRouterAdapter_LiveSmoke t.Skips when env absent | `make test` (skip path) |
| 3.2 | `feat(agent): openrouter sentinel-sweep helper (R-OR-08)` | Scan(t, captured) helper; deny-list built at runtime | `make test` |
| 3.3 | `test(agent): openrouter sentinel sweep catches deliberate leak (R-OR-08 mutation test)` | TestSentinelSweep_CatchesDeliberateLogfKeyMutation | `make test` |
| 3.4 | `test(agent): openrouter Credential redaction carries through (R-OR-08 scenario 2)` | TestSentinelSweep_OpenaicompatCredentialStringHolds asserts credential renders as `<redacted>` through String/GoString | `make test` |
| 3.5 | `ci(agent): openrouter live smoke workflow dispatch only (R-OR-07)` | `.github/workflows/agent-openrouter-smoke.yml` | manual: `gh workflow view agent-openrouter-smoke.yml --yaml` |
| 3.6 | `docs(agent): openrouter live smoke setup block` | smoke_setup.md inside openrouter/doc.go (or doc.go delta) | `make test` |

---

## 9. Per-PR Size Forecast

Authored code (excluding generated goldens/recorded fixtures per `sdd-phase-common.md §E`):

| PR | Files (new) | Naive LOC | Corrected ×2–4 | Authored code (mid) | Within 800 budget? |
|---|---|---|---|---|---|
| **PR #1 wrapper** | 9 new files | ~360–580 | ~700–2300 (mid ~1100) | ~200 LOC | **Borderline; corrected mid exceeds.** **Mitigation: split commit 1.5 (attribution headers) into two commits** — first the header attachment (test + impl), then the unawareness negative proof. Authored LOC per commit stays ≤200; cumulative PR #1 still lands ~200 LOC authored; test bytes dominate the corrected count and are reviewable. **PR #1 stays within budget with the work-unit split.** |
| **PR #2 conformance bridge** | 4 new files + 5 fixture var-blocks | ~360–650 | ~700–2600 (mid ~1500) | ~400–600 LOC | **Yes for authored code.** Fixture bytes excluded per `sdd-phase-common.md §E`. |
| **PR #3 live smoke** | 4 new files + 1 workflow YAML | ~80–140 | ~150–550 | ~100 LOC | **Yes.** |
| **Total** | 17 new files | ~800–1370 | ~1550–5450 (mid ~3000) | ~700–900 | |

**PR #1 risk mitigation**: the `work-unit-commits` skill's split (commit 1.5 → 1.5a + 1.5b) keeps each commit ≤200 LOC authored; PR #1 stays reviewable in ≤60 min (chained-pr skill rule). Alternative if PR #1 still over-runs: defer `charter_test.go` (commit 1.9) to a follow-up commit on the same branch — it is review-only fence, not load-bearing behavior.

---

## 10. Risks

| Risk | Sev | Mitigation |
|---|---|---|
| **Tracker-chain diff pollution** (child PRs accumulate previous PR's diff) | Med | Rebase each child onto its immediate parent before review. Polluted diff = base bug (chained-pr skill rule). |
| **800-line budget on PR #1** | Med | Work-unit commit split in §8 PR #1.5a/b; fallback = defer charter_test.go. |
| **AI-29 reopen trigger #1** if default model changes | Med | R-OR-05 capability-record assertion + TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment gate. Any default swap fails the test until the spec amendment lands. |
| **Attribution-header leakage into openaicompat** (a future contributor widens `openaicompat/request.go`) | Low | `TestOpenRouter_DoesNotWidenOpenAICompatRequestSurface` (PR #1) reads `openaicompat/request.go` as bytes; fails on any of the three header names appearing. |
| **`:free` rate-limit flakiness in live smoke** | Low | Defaults are paid (`openai/gpt-4o`); spec C2/Q1. |
| **Live-smoke secret leakage via `t.Logf`** | Low | `openaicompat.Credential.String()` returns `<redacted>` (R-APC-014); sentinel-sweep helper catches deliberate `t.Logf(key)` mutations (R-OR-08 mutation test). |
| **AI-00.3 forward-guard regression** | VLow | Wrapper imports only stdlib + in-module; zero new `require` lines; existing tests stay unmodified. |
| **`error_type` vocabulary mismatch surfaces in production** (deferred) | Low | AI-32 widening is explicitly out of scope (R-OR-10); `openaicompat/failure_map.go` reads HTTP status, which works on OpenRouter wire. A future change widens AI-32. |
| **Header mutation race** (the cloned header is shared with a concurrent goroutine) | Low | `http.Header.Clone()` is documented thread-safe; `RoundTrip` runs serially per request in stdlib's `http.Client`. No new shared state. |
| **`TestLayer1_ModuleHasNoDependencies_ZeroRequires` flips if `net/http` were ever counted as a "non-stdlib" dep** | VLow | `net/http` is stdlib; the test filters via `.Standard` (per the guard's own comment, line 33). |
| **AI-38 capability record drift if a future contributor adds a CAP-O-01 reasoning case** | Low | `applyDeclaredAbsences` (conformance_suite.go:330) marks absent at the suite level; any new CAP-O-01 case would need `Factory.Reasoning = true` to run, which the OpenRouter factory does not set. |

---

## 11. Rollback Matrix

| PR | Rollback action | Observable behavior after revert | Sentinel test |
|---|---|---|---|
| **PR #1 wrapper** | `git revert -m1 <merge-sha>` on `tracker/add-openrouter-first-provider` | `make test` green; `go.mod` zero `require`; AI-00.3 + AI-25.2 guards green (no new sub-package to scan). **No behavior change** — no caller uses `openrouter.NewProvider` yet; the openaicompat package's own `bridge_test.go` continues to pass. | `TestOpenRouter_AmbientAuthorityGuardCoversNewSubPackage` (PR #1, fails if reverted only partially — guards must stay in lockstep with the sub-package). |
| **PR #2 conformance bridge** | Revert PR #2 merge commit | `make test` green; the wrapper alone (from PR #1) still works; `agenttest.RunConformance` still passes for the openaicompat-only factory. The AI-38 first-concrete-adapter claim is deferred. | `TestOpenRouterAdapter_CapabilityRecordMatchesAI24` (PR #2) — if reverted only partially, the test references a missing factory. |
| **PR #3 live smoke** | Revert PR #3 merge commit | `make test` green; `OPENROUTER_API_KEY` env no longer referenced; `.github/workflows/agent-openrouter-smoke.yml` removed — no real-money calls in CI. **No effect on PR #1/PR #2** (smoke is in its own subdirectory and workflow file). | `TestSentinelSweep_CatchesDeliberateLogfKeyMutation` (PR #3) — if reverted only partially, the sentinel helper is gone but the smoke test still calls it. |

**Sentinel integration test for the matrix** (PR #1's own work; gives reviewers confidence the rollback chain is sound):

```go
// In openrouter/charter_test.go (PR #1):
func TestOpenRouter_RevertProof_NonexistentAtRest(t *testing.T) {
    // This test is a placeholder for the reviewer-confidence story: it
    // asserts that the new sub-package's directory exists at HEAD; the
    // actual revert-proof is the merge-revert CI run, recorded in PR #1's
    // PR body, where `git revert -m1 <merge-sha>` leaves `make test` green.
    // A reviewer reads that record, not this test.
    if _, err := os.Stat("."); err != nil {
        t.Fatalf("openrouter sub-package not present at HEAD: %v", err)
    }
}
```

(The merge-revert CI evidence is the actual proof; the test is a directory-existence sentinel so a future contributor who deletes the directory notices via `make test` red before merging.)

---

## 12. Out of Scope (restated from R-OR-10)

- **No Anthropic native adapter.** No file under `backend/agent/src/ai/openaicompat/openrouter/` matches `*anthropic*`; the existing `ai.WithProviderExtension("anthropic", ...)` test fixtures in `openaicompat/extension_test.go` are generic, not Anthropic-specific, and stay unchanged.
- **No AI-32 widening for `error_type`.** No literal `"error_type"` in any non-test source under the new sub-package; `openaicompat/failure_map.go` is unchanged.
- **No `reasoning_content` (or `reasoning` / `reasoning_details`) emission.** Wrapper never renders these as an `ai.EventKindReasoningDelta` payload; `openaicompat/chunk.go`'s plain `json.Unmarshal` drops them silently at decode.
- **No new `go.mod` require.** `TestLayer1_ModuleHasNoDependencies_ZeroRequires` pins it.
- **No AI-29 reopen.** Default model `openai/gpt-4o` preserves the struck verdict; any reasoning-capable default would reopen AI-29 under trigger #1.
- **No Layer 3 (`src/coding`) composition root.** Layer 3 is not yet built; this change documents the wiring pattern only.
- **No AI-40 publication of the capability matrix.** A separate change; AI-40.2 publishes from AI-38.2's generated record.
- **No Wave-2 carryovers** (`CheckEmit` rule 4, `*Failure.GoString()`) — owned by AI-41.

---

## 13. References

- **Spec**: `openspec/changes/add-openrouter-first-provider/specs/ai-openrouter-first-provider/spec.md` · Engram `#2573`
- **Proposal**: `openspec/changes/add-openrouter-first-provider/proposal.md` · Engram `#2570`
- **Explore**: `openspec/changes/add-openrouter-first-provider/explore.md` · Engram `#2568`
- **Wrapper-placement decision**: Engram `#2571` (sibling sub-package `openaicompat/openrouter/`)
- **Composed shipped specs (read-only)**:
  - `openspec/specs/ai-provider-client/spec.md` (AI-25) — R-APC-001…014, NFR-APC-A…G
  - `openspec/specs/ai-provider-conformance-suite/spec.md` (AI-23/38) — R-CNF-001…018
  - `openspec/specs/ai-stream-testkit/spec.md` (AI-22) — R-STK-001…013
  - `openspec/specs/ai-model-provider/spec.md` (AI-20) — R-AMP-001…021, signature guard at R-AMP-014…016
- **Milestone charters** (doc 0002 — Layer 1 task graph): AI-25 lines 1447–1491 · AI-38 lines 2241–2277 · AI-39 lines 2279–2299
- **AI-24 pre-decision** (vendor/transport, OpenAI-compatible Chat Completions + raw `net/http` + zero `go.mod` requires): Engram `#2432`
- **AI-29 struck verdict** (2026-08-04, reasoning stream absent): `openspec/changes/archive/2026-08-04-cachicamas-ai-provider-reasoning-stream/decision.md` §5 row 4, §7, §9 (triggers)
- **ADR 0004** (3-layer agentic architecture): Engram `#1997` + `docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md`
- **ADR 0005** § D1 (layered-module dependency rule) + § D3 (observability boundary) + § D2 Guard C (signature guard's path resolution): `docs/adr/0005-promote-agent-stack-to-own-module.md`
- **Backend test runner**: Engram `#2055` (`cd backend/agent && make test` = `go test -race -v ./...`)
- **Existing shipped code** (anchor — read, NOT modified):
  - `backend/agent/src/ai/openaicompat/client.go` (Config, New) · `request.go` (header surface) · `credential.go` (redacting String/GoString) · `bridge_test.go` (factory pattern template) · `ambient_authority_test.go` (call-site scan template) · `reasoning_absence_test.go` (TestReasoningExtensionField_DroppedNotLeakedNotFailed template) · `credential_scan_test.go` (deny-list pattern)
  - `backend/agent/src/ai/import_boundary_test.go` (AI-00.3 forward guard — `allowedNonStdlibPrefixes` confirmed at one entry)
  - `backend/agent/src/agenttest/conformance_suite.go` (RunConformance, Factory) · `conformance_capabilities.go` (CAP-O-01…03 outcomes) · `conformance_record.go` (CapabilityRecord, Verdict) · `conformance_redaction.go` (sentinel-sweep template) · `provider_signature_guard_test.go` (AI-20 signature guard)
  - `backend/agent/src/ai/provider.go` (ModelProvider interface — unchanged)
  - `backend/agent/go.mod` (3 lines, zero `require` — confirmed)
  - `backend/agent/Makefile` (`make test = go test -race -v ./...`)
- **OpenSpec rules**: `openspec/config.yaml` — RFC 2119 + Given/When/Then; `tdd: true`; no new top-level deps without ADR.
- **Project rules**: `openspec/AGENTS.md` — Go 1.26.3 layered module; conventional commits (no AI attribution); strict RED-GREEN-REFACTOR; `backend/agent` is layered (NOT hexagonal); `allowedNonStdlibPrefixes` gate.
