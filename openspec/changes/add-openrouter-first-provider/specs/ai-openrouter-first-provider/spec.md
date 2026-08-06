# Spec — OpenRouter as the first concrete AI provider for Layer 1

> **Change**: `add-openrouter-first-provider`
> **Capability**: `ai-openrouter-first-provider` (new)
> **Status**: DRAFT
> **Date**: 2026-08-06
> **Artifact store**: hybrid (file + engram)
> **Format**: RFC 2119 + Given/When/Then, per `openspec/config.yaml`
> **Requirement IDs**: `R-OR-0NN` · **Scenario IDs**: `S-OR-0NN`
> **Composes (read-only, NOT modified)**: `ai-provider-client` (AI-25) · `ai-provider-conformance-suite` (AI-23/38) · `ai-stream-testkit` (AI-22) · `ai-model-provider` (AI-20)
> **Links**: [proposal](../../proposal.md) · [explore](../../explore.md) · Engram `#2570` (proposal) · `#2568` (explore) · `#2571` (wrapper-placement decision)

---

## Purpose

ADR 0004 (Engram #1997) established the three-layer agentic architecture; AI-20 (Engram #2235) froze the `ModelProvider` interface every concrete adapter must satisfy; AI-24 (Engram #2432) pre-decided vendor = OpenAI-compatible Chat Completions streaming dialect with raw `net/http` transport and zero `go.mod` requires. The shipped `backend/agent/src/ai/openaicompat/` package is vendor-agnostic but has **no first concrete vendor**, so AI-38 (full deterministic adapter conformance) and AI-39 (opt-in live smoke) have no subject, and AI-40 (Layer 2 readiness handoff) cannot publish. This capability concretizes **OpenRouter** as that first vendor — composing (not re-implementing) the shipped `openaicompat` package, `agenttest.RunConformance`, and the `ai-stream-testkit` helpers. It ships as three chained PRs under a no-merge tracker: wrapper → conformance bridge → live smoke.

## ADDED Requirements

### R-OR-01 — Wrapper construction is injection-only

The wrapper SHALL accept the OpenRouter base URL, an opaque bearer credential, a model identifier, and an optional HTTP client — all by injection. The wrapper SHALL NOT read any environment variable, touch the filesystem, or spawn a process (AI-25.2 invariant). The wrapper SHALL construct an `openaicompat.Client` via the shipped `openaicompat.New(Config{...})` constructor.

#### Scenario: Construction with injected values

- GIVEN a wrapper built with the OpenRouter URL, opaque bearer, model `openai/gpt-4o`, no client
- WHEN the wrapper's non-test sources are scanned for ambient authority
- THEN no `os.Getenv`, `os/exec`, or filesystem call is present
- AND the underlying `openaicompat.Client` is the one the injected values configured.

#### Scenario: Construction rejects invalid configuration

- GIVEN a wrapper built with an empty (zero-value) bearer credential
- WHEN construction runs
- THEN it fails with an AI-04 typed failure naming the credential input
- AND no outbound request is made.

### R-OR-02 — Attribution headers are wrapper-injected, never openaicompat-injected

The wrapper SHALL inject `HTTP-Referer`, `X-OpenRouter-Title` (alias `X-Title`), and `X-OpenRouter-Categories` on every outbound request via a wrapper-owned `http.RoundTripper`. Empty strings SHALL suppress a header. The shipped `openaicompat` package SHALL remain unaware of these headers; its "`Authorization` and `Content-Type` are the only headers it sets" rule SHALL stay intact.

#### Scenario: All three headers observed when non-empty

- GIVEN a stub-transport probe with all three attribution strings non-empty
- WHEN the stub reads the outbound request
- THEN `HTTP-Referer`, `X-OpenRouter-Title`/`X-Title`, and `X-OpenRouter-Categories` are each present with the injected values.

#### Scenario: Empty strings suppress the headers

- GIVEN a stub-transport probe with all three attribution strings empty
- WHEN the stub reads the outbound request
- THEN none of the three headers is present.

#### Scenario: openaicompat's header surface is unmodified

- GIVEN the merged wrapper
- WHEN `openaicompat/request.go` is read
- THEN it does not name any of the three attribution headers and still sets only `Authorization` and `Content-Type`.

### R-OR-03 — Default model and deliberate-model field

The conformance bridge SHALL target `openai/gpt-4o` (non-reasoning, paid) by default. The wrapper SHALL expose a deliberate-model field on its `Config` so model swaps do not require a code change at construction sites.

#### Scenario: Bridge uses the documented default

- GIVEN a fresh conformance-bridge factory
- WHEN the factory declares the model
- THEN the declared model equals `openai/gpt-4o`.

#### Scenario: Config carries a deliberate-model field

- GIVEN the wrapper's `Config` shape
- WHEN a reader enumerates the fields
- THEN a model-identifier field is present and defaults to `openai/gpt-4o`.

### R-OR-04 — `stream_options.include_usage` stays set

The wrapper SHALL always emit `stream_options.include_usage = true` in the wire body. OpenRouter's deprecation of the field SHALL be documented but not acted on; dropping the field would diverge the wire body from other openaicompat-target vendors (vLLM, Ollama, LocalAI).

#### Scenario: Field is present in the outbound body

- GIVEN a stub-transport probe
- WHEN the stub reads the request body
- THEN `stream_options.include_usage` equals `true`.

### R-OR-05 — Capability-record assertion: `CAP-O-01` reasoning is absent under the default model

The capability record generated by AI-38.1 for the OpenRouter bridge SHALL record `CAP-O-01` (reasoning) as `absent` under the default model `openai/gpt-4o`. Switching the default to a reasoning-capable model SHALL require an explicit ADR AND a spec amendment that reopens AI-29's struck verdict under trigger #1 — never a silent default swap.

#### Scenario: Default-model record equals `absent`

- GIVEN the conformance bridge running `agenttest.RunConformance` with the default factory
- WHEN the generated record is read
- THEN `CAP-O-01` carries the outcome `absent`
- AND `TestOpenRouterAdapter_CapabilityRecordMatchesAI24` passes.

#### Scenario: Default-model swap does not happen silently

- GIVEN the wrapper's `Config`
- WHEN the default model is changed without reopening AI-29
- THEN the capability-record test fails until the spec amendment lands.

### R-OR-06 — Conformance bridge runs the suite against the OpenRouter wrapper

The conformance bridge SHALL run `agenttest.RunConformance` end-to-end against the OpenRouter wrapper using OpenRouter-shaped transcripts. At least the five required capabilities SHALL pass (`CapStreamingText`, `CapToolCalls`, `CapCompletionMetadata`, `CapCancellation`, `CapTerminal`). The `TestReasoningExtensionField_DroppedNotLeakedNotFailed` scenario SHALL be cited as covering OpenRouter's `reasoning` / `reasoning_details` extension fields.

#### Scenario: Required capabilities all pass

- GIVEN the OpenRouter bridge factory
- WHEN `agenttest.RunConformance(t, factory)` runs
- THEN every required capability case passes
- AND the capability record equals AI-24 §8's expected `absent × 3`.

#### Scenario: Reasoning extension field is dropped, not leaked, not failed

- GIVEN an OpenRouter transcript carrying a `delta.reasoning_details` extension field
- WHEN the conformance bridge decodes it
- THEN no reasoning byte appears in any text event
- AND the stream does not fail
- AND `TestReasoningExtensionField_DroppedNotLeakedNotFailed` passes.

### R-OR-07 — Live smoke is opt-in via env vars only (no CI workflow)

The live smoke (`TestOpenRouterAdapter_LiveSmoke`) SHALL be `t.Skip`-gated on the absence of BOTH `OPENROUTER_API_KEY` AND `RUN_LIVE_OPENROUTER_SMOKE=1` in the test process. `make test` in `backend/agent/` SHALL NOT depend on OpenRouter or any network credential. **No CI workflow file is required by this spec** — the repository's established posture is no `.github/workflows/` (see ADR 0005 § Enforcement, doc 0002 "No CI exists"), and the live smoke is opt-in for human/local runs only.

#### Scenario: Skip path exercised without the env vars

- GIVEN `OPENROUTER_API_KEY` OR `RUN_LIVE_OPENROUTER_SMOKE` absent from the test process
- WHEN `make test` runs
- THEN the live smoke is skipped with an attributable message
- AND no outbound request is made.

### R-OR-08 — Credential redaction extends to the live smoke's logging

The live smoke SHALL NOT log the credential, secret value, or full prompt. A sentinel-sweep helper SHALL scan captured output (test logs, error messages, `t.Logf` calls) against a deny-list that includes the literal `OPENROUTER_API_KEY` env-var name, the secret's prefix (4 chars), and planted prompt bytes. A sentinel match SHALL fail the test.

#### Scenario: Sentinel sweep catches a deliberate leak mutation

- GIVEN a mutated smoke test that calls `t.Logf(key)` with the credential
- WHEN the sentinel sweep runs
- THEN the test fails and names the leak vector without reprinting the credential.

#### Scenario: `openaicompat.Credential` redaction carries through

- GIVEN the wrapper's credential value
- WHEN it is rendered through `String()`, `GoString()`, `MarshalJSON`, or default formatting
- THEN no rendering contains the token.

### R-OR-09 — AI-00.3 forward guard stays green

The wrapper SHALL add no new top-level Go dependency. `backend/agent/go.mod` SHALL declare zero `require` lines after the change. `TestLayer1_ModuleHasNoDependencies_ZeroRequires` SHALL pass unmodified. `allowedNonStdlibPrefixes` SHALL remain at one entry (the module itself).

#### Scenario: Module has no new requires

- GIVEN the change merged
- WHEN `backend/agent/go.mod` is read
- THEN it declares zero `require` lines
- AND both AI-00 import guards pass.

### R-OR-10 — Out-of-scope fence (negative SHALLs)

The wrapper SHALL NOT support an Anthropic native adapter, SHALL NOT widen AI-32 to map OpenRouter's `error_type` discriminator, SHALL NOT emit a `reasoning_content` (or `reasoning` / `reasoning_details`) extension field, and SHALL NOT introduce any new `go.mod` require.

#### Scenario: Negative fences are mechanical

- GIVEN the merged wrapper sources and `go.mod`
- WHEN a reviewer looks for an Anthropic-only type, an `error_type` map, a `reasoning_content` render, or any new `require`
- THEN none is present.

## Decisions

| ID | Decision | Basis |
|---|---|---|
| **C1** | Default model = `openai/gpt-4o` (non-reasoning, paid) | Preserves AI-29's struck verdict; explore §2.7 |
| **C2** | Conformance + smoke use a paid model (no `:free` suffix) | `:free` rate-limit flakiness; explore §2.10 |
| **C3** | Attribution headers wrapper-injected, NOT openaicompat-injected | Keeps openaicompat's header surface narrow; explore §2.3, Engram #2571 |
| **C4** | `stream_options.include_usage` stays set | Cross-vendor wire-body uniformity; explore §2.5 |
| **Q1** | Conformance sweep = `openai/gpt-4o` only | Sweeping Anthropic passthrough reopens AI-29 under trigger #1 |
| **Q2** | Layer 3 reads env, passes opaque bearer into `Config.Credential` | AI-25.2 invariant; wrapper reads nothing (memory #2432 §3) |
| **Q3** | Wrapper placement = sibling sub-package `openaicompat/openrouter/` | AI-25.2 call-site scan scope stays clean; Engram #2571 |
| **Q4** | Conformance bridge ships in PR #2 of this change | AI-38 first-concrete-adapter charter; doc 0002 lines 2241–2277 |
| **Q5** | Live smoke opt-in via env vars `OPENROUTER_API_KEY` + `RUN_LIVE_OPENROUTER_SMOKE=1` (no CI workflow, consistent with ADR 0005 § Enforcement "no `.github/workflows/`") | AI-39.1 charter; doc 0002 lines 2279–2299 |
| **Q6** | Three chained PRs under no-merge tracker | 800-line PR budget per `sdd-phase-common.md §E`; natural work-unit split |

## Traceability

- **Proposal**: `openspec/changes/add-openrouter-first-provider/proposal.md` · Engram observation **#2570** (`sdd/add-openrouter-first-provider/proposal`)
- **Explore**: `openspec/changes/add-openrouter-first-provider/explore.md` · Engram topic `sdd/add-openrouter-first-provider/explore` (obs #2568)
- **Wrapper-placement decision**: Engram observation **#2571** (`decision/openrouter-wrapper-placement`)
- **Composed shipped specs** (read-only, NOT modified):
  - [`openspec/specs/ai-provider-client/spec.md`](../../../../../specs/ai-provider-client/spec.md) (AI-25) — provider client construction
  - [`openspec/specs/ai-provider-conformance-suite/spec.md`](../../../../../specs/ai-provider-conformance-suite/spec.md) (AI-23/38) — conformance runner + capability record
  - [`openspec/specs/ai-stream-testkit/spec.md`](../../../../../specs/ai-stream-testkit/spec.md) (AI-22) — drain, ordering, leak, redaction helpers
  - [`openspec/specs/ai-model-provider/spec.md`](../../../../../specs/ai-model-provider/spec.md) (AI-20) — `ModelProvider` interface + signature guard
- **Milestone charters** (doc 0002 — Layer 1 task graph): AI-25 lines 1447–1491 · AI-38 lines 2241–2277 · AI-39 lines 2279–2299
- **AI-24 pre-decision** (vendor/transport, OpenAI-compatible Chat Completions + raw `net/http` + zero `go.mod` requires): Engram observation **#2432**
- **AI-29 struck verdict** (2026-08-04, reasoning stream absent): `openspec/changes/archive/2026-08-04-cachicamas-ai-provider-reasoning-stream/decision.md` §5 row 4, §7, §9 (triggers)
- **ADR 0004** (3-layer agentic architecture): Engram observation **#1997** + `docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md`
- **Backend test runner**: Engram observation **#2055** (`cd backend/agent && make test` = `go test -race -v ./...`)
- **OpenSpec rules**: `openspec/config.yaml` `rules.specs` — Given/When/Then, RFC 2119 keywords, independently verifiable scenarios

## Per-PR coverage map

| PR | Branch chain | Milestone slice | Requirements covered |
|---|---|---|---|
| **PR #1 — wrapper** | `feat/openrouter-wrapper` → `tracker/add-openrouter-first-provider` | AI-25.1 + AI-25.3 | R-OR-01, R-OR-02, R-OR-03, R-OR-04, R-OR-09, R-OR-10 |
| **PR #2 — conformance bridge** | `feat/openrouter-conformance-bridge` → PR #1's branch | AI-38.1 + AI-38.2 + AI-38.3 | R-OR-05, R-OR-06 |
| **PR #3 — live smoke** | `feat/openrouter-live-smoke` → PR #2's branch | AI-39.1 | R-OR-07, R-OR-08 |