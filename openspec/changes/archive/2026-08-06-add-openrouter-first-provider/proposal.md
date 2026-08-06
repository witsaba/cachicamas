# Proposal: add-openrouter-first-provider

> **Status**: DRAFT
> **Change ID**: `add-openrouter-first-provider`
> **Date**: 2026-08-06
> **Artifact store**: hybrid — file at `openspec/changes/add-openrouter-first-provider/proposal.md` AND Engram topic `sdd/add-openrouter-first-provider/proposal`
> **Depends on** (explore): `openspec/changes/add-openrouter-first-provider/explore.md` (Engram #2568)
> **Project rules**: [`openspec/AGENTS.md`](../../AGENTS.md) — Go 1.26.3 layered module; conventional commits; no Co-Authored-By; no new top-level deps without an ADR; TDD red→green→refactor enforced at apply time.
> **Locked preflight**: `delivery_strategy = auto-chain` · `chain_strategy = feature-branch-chain` · `review_budget_lines = 800` · `change_name = add-openrouter-first-provider`. These are orchestrator-cached and NOT reopened by this proposal.

---

## 1. Why

ADR 0004 (memory #1997, merged 2026-07-17) established the 3-layer agentic architecture for cachicamas, with `backend/agent/src/ai/` as Layer 1's home. AI-20 (memory #2235) froze the `ModelProvider` interface every adapter must satisfy. AI-24 explore (memory #2432) pre-decided vendor = **OpenAI-compatible Chat Completions streaming dialect** and transport = **raw `net/http`, zero `go.mod` requires**, and named OpenRouter, vLLM, Ollama, LocalAI as in-scope concrete vendors. The shipped `backend/agent/src/ai/openaicompat/` package (AI-25/26/27/28/30/31/32, all merged 2026-08-05) is the generic OpenAI-compatible adapter — vendor-agnostic, working against any injected base URL — but it has no first concrete vendor value, no first concrete conformance bridge, and no first live smoke. **Without a first concrete vendor, Layer 1 has no complete subject for AI-38 (full deterministic adapter conformance) or AI-39 (opt-in live smoke), and the Layer 2 readiness handoff (AI-40) cannot publish.**

This change concretizes **OpenRouter** as the first concrete vendor. It ships three natural work units (wrapper → conformance bridge → live smoke) as three chained PRs under a no-merge tracker, preserving the AI-29 struck-node verdict by targeting a non-reasoning model and adding zero `go.mod` dependencies.

**User-stated intent** (verbatim from the orchestrator prompt): *"Add OpenRouter as the FIRST concrete provider for Layer 1 of the cachicamas AI agent. The change concretizes the pre-decided OpenAI-compatible Chat Completions streaming dialect (memory #2432, prior AI-24 work) to OpenRouter specifically and implements AI-25 (client construction) + the OpenRouter-side parts of AI-38 (conformance) and AI-39 (live smoke) as the first three chained PRs."*

## 2. What changes (one-line per PR)

| PR | Milestone slice | One-line summary |
|---|---|---|
| **#1 — OpenRouter wrapper** | AI-25.1 + AI-25.3 (concretization on top of shipped openaicompat) | New `openrouter` sub-package under `openaicompat/` composing `openaicompat.Config` with hardwired endpoint, injected attribution headers, and a thin Provider value. |
| **#2 — Conformance bridge** | AI-38.1 + AI-38.2 + AI-38.3 | New `openrouter/bridge_test.go` factory + capability-record driver: `agenttest.RunConformance` runs end-to-end against the OpenRouter-shaped wire envelope. |
| **#3 — Live smoke** | AI-39.1 | `TestOpenRouterAdapter_LiveSmoke`: `t.Skip`-gated on `OPENROUTER_API_KEY` absence, opt-in workflow-dispatch in CI, silent on secrets, one bounded request, asserts stream-shape invariants only. |

Tracker PR is **no-merge / draft**; only the tracker merges to `main` after all three child PRs land (per the locked feature-branch-chain topology).

## 3. Impact / scope

**Lands parts of these milestones**:
- **AI-25 (provider configuration + client construction)** — concretized: the new OpenRouter wrapper exercises `openaicompat.New(Config{...})` with `Endpoint = "https://openrouter.ai/api/v1"` and an injected `Credential`. AI-25.1 (injected construction) and AI-25.3 (test-server viability) are proven against OpenRouter-shaped wire. AI-25.2 (call-site scan) re-runs against the new sub-package's own sources (per `R-APC-008`).
- **AI-38 (full deterministic adapter conformance)** — the OpenRouter-specific factory is the **first concrete adapter** AI-38.1 (`conformance_suite.go` `RunConformance`) consumes; AI-38.2's generated capability record is asserted against AI-24 §8's expected outcomes (`CAP-O-01 = absent`, `CAP-O-02 = absent`, `CAP-O-03 = absent`). AI-38.3's boundary-replay matrix runs against the recorded OpenRouter-format transcripts.
- **AI-39 (opt-in live smoke)** — the only sanctioned live-network test in the entire stack; skips clean without credentials.
- **AI-23.8 capability outcome recording** — the factory declares `Reasoning: false, TokenCounting: false, CacheBoundary: false` (non-nil pointers per R-CNF-004), so AI-38.2's generated record carries the three `absent` outcomes.

**Stays struck** (preserved by the proposed defaults):
- **AI-29 (reasoning stream)** — struck 2026-08-04 in `decision.md` of `cachicamas-ai-provider-reasoning-stream`. Picking `openai/gpt-4o` (non-reasoning) as the default model preserves the verdict; OpenRouter's `delta.reasoning` / `delta.reasoning_details` extension fields are silently dropped by `openaicompat/chunk.go:213`'s plain `json.Unmarshal` (the existing `reasoning_absence_test.go:149` covers the drop-not-leak posture; the field-name rename from `reasoning_content` to `reasoning` / `reasoning_details` is a deferred amendment).

**Subsystems touched**:

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/ai/openaicompat/openrouter/` | **New** | Wrapper package (`provider.go`, `doc.go`, tests). Imports only stdlib + `openaicompat` + `package ai`. |
| `backend/agent/src/ai/openaicompat/openrouter/bridge_test.go` | **New** | OpenRouter-shaped transcripts + factory (analog to `openaicompat/bridge_test.go`). |
| `backend/agent/src/ai/openaicompat/openrouter/smoke_test.go` | **New** | `TestOpenRouterAdapter_LiveSmoke` (opt-in only). |
| `backend/agent/src/ai/openaicompat/` | Unchanged surface | Wrapper composes existing `Config` + `New`; **no** openaicompat file edits (per the explore §2.3 / §2.5 / §2.8 decisions to keep the openaicompat package's header surface narrow). |
| `backend/agent/src/agenttest/` | Unchanged | Consumes existing `Factory`, `RunConformance`, `RunConformanceFor`. |
| `backend/agent/go.mod` | Unchanged | Zero new `require` lines (NFR-APC-A carried into this change). |
| `backend/agent/src/ai/import_boundary_test.go` | Optional comment refresh | The 2026-08-05 stale-comment fix from NFR-APC-C remains accurate; this change does not regress it. |

## 4. Out of scope

Explicit non-goals (must NOT land in any PR):

- **Anthropic native adapter** — single-vendor reach is not what AI-24's dialect choice buys; deferred to a separate change. Memory #2432 §1 records this rejection.
- **AI-32 widening for OpenRouter's `error_type` discriminator** — known scope gap (explore §2.8); `openaicompat/failure_map.go` is pinned to OpenAI's narrower vocabulary and works fine on OpenRouter wire (`HTTP-Referer` errors carry the same envelope shape except for `error.code` being an integer status mirror). A future change widens AI-32 against the OpenRouter-specific `error_type` vocabulary, **after** AI-38.2's capability record settles.
- **Reopening AI-29** — if a reasoning-capable default model is later chosen, that change must reopen AI-29 under trigger #1 of the struck decision and add a second `Factory` variant; **this change does not**.
- **Any new `go.mod` require** — NFR-APC-A preserved; no ADR gate fires.
- **Layer 3 (`src/coding`) composition root** — that work is a separate change. The proposal's Q2 documents the *expected wiring pattern* (Layer 3 reads the env / config file and passes an opaque bearer string into `openrouter.Config.Credential`) but does not implement Layer 3.
- **AI-40 publication of the capability matrix** — a separate change. AI-40.2 publishes the matrix from AI-38.2's generated record; that change closes Wave 6 of doc 0002.
- **Wave-2 carryovers (`CheckEmit` rule 4, `*Failure.GoString()`)** — parked at AI-21, owner = AI-41; out of scope for this change.
- **A user-facing CLI, scheduled billing-consuming runs, or production deployment** of the live smoke (per AI-39's charter line 2289).

## 5. PR chain (feature-branch-chain topology)

**Tracker branch** (draft, no-merge): `tracker/add-openrouter-first-provider`
**Base**: `main` at the commit the tracker PR is opened from.

```
main
  └─ tracker/add-openrouter-first-provider     [draft PR, no-merge]
       └─ feat/openrouter-wrapper              [PR #1, targets tracker]
            └─ feat/openrouter-conformance-bridge   [PR #2, targets PR #1's branch]
                 └─ feat/openrouter-live-smoke       [PR #3, targets PR #2's branch]
```

The tracker is opened **first** with an empty body that names the three child PRs and cites this proposal. The tracker merges to `main` only after all three children land (rebased if necessary so the diff stays clean per the chained-pr skill's "polluted diffs are base bugs" rule).

### Per-PR size forecast

| PR | Files (planned) | Naive (lines incl. tests) | Corrected ×2–4 (per AI-21 1.95×, AI-16 3.7× historical) | Within 800 budget? |
|---|---|---|---|---|
| **#1 wrapper** | `openrouter/provider.go` (Config, NewProvider, header attach) ~150–200 · `openrouter/provider_test.go` (header probe R-APC-001 style) ~80–150 · `openrouter/doc.go` ~30–50 · `openrouter/ambient_authority_test.go` (call-site scan re-run for the new sub-package) ~60–100 · `openrouter/charter_test.go` ~40–80 | **~360–580** | **~700–2300** | Borderline (mid ~1100); will be reassessed at sdd-tasks forecast. Generous test bytes + the AI-25.2 scan re-run are the dominant share; authored code ~200 lines. |
| **#2 conformance bridge** | `openrouter/bridge_test.go` (factory, transcript renderers for text / tool calls / completion-with-usage / errors) ~200–350 · `openrouter/conformance_test.go` (per-capability `RunConformanceFor` drivers) ~80–150 · supporting fixture files (recorded transcripts) ~80–150 | **~360–650** | **~700–2600** (mid ~1500) | **No for raw bytes; yes for authored code**. Per the chained-pr skill's `sdd-phase-common.md §E` rule, generated goldens and recorded fixtures are **excluded** from authored risk count. Authored code (factory, renderers, drivers) is ~400–600 lines corrected → within budget. |
| **#3 live smoke** | `openrouter/smoke_test.go` (env-gate helper, one bounded request, sentinel sweep over captured output) ~60–100 · `openrouter/smoke_setup.md` (credential-safe setup instructions, in the package's `doc.go`) ~20–40 | **~80–140** | **~150–550** | **Yes**, comfortably. |

**Total naive across all 3 PRs**: ~800–1370 lines; **total corrected**: ~1550–5450 lines. The 800-line budget is sufficient per PR when authored code is counted (per the chained-pr skill's rule); raw line counts that include recorded fixture bytes (PR #2) over-run it but the skill explicitly excludes those from the budget.

Each PR has **a clear start, clear finish, autonomous scope, verification, and reasonable rollback** (sdd-phase-common §E):

### PR #1 — OpenRouter wrapper
- **Files**: `backend/agent/src/ai/openaicompat/openrouter/{provider.go, provider_test.go, doc.go, ambient_authority_test.go, charter_test.go}`.
- **Contract**: `openrouter.NewProvider(openrouter.Config{...}) (ai.ModelProvider, error)`; Config carries `Credential openaicompat.Credential`, `HTTPClient *http.Client` (optional), `HTTPReferer`, `XTitle`, `XOpenRouterCategories` (three empty strings mean "don't send"); three attribution headers attached to the outbound `Authorization` request inside an injected `http.RoundTripper` wrapper that the wrapper owns (NOT in `openaicompat`).
- **Tests**: R-APC-001 stub-transport probe for the three attribution headers (all empty / each non-empty); R-APC-013-style one-request against a local `httptest.Server` (header shape + path); `R-APC-008`-style call-site scan re-run against the new sub-package; AI-25.2's four bite proofs (the new sub-package must be in scope).
- **Rollback**: revert merge → no behavior change anywhere; the conformance bridge and smoke PRs have not landed, so no callers exist. `go.mod` still zero requires. `R-AMP-020` "fully conformant on required surface" holds.

### PR #2 — Conformance bridge + AI-38.1 driver
- **Files**: `backend/agent/src/ai/openaicompat/openrouter/bridge_test.go` (transcript renderers + factory), `openrouter/conformance_test.go` (`TestOpenRouterAdapter_RunConformance*` per-capability drivers), plus recorded fixture bytes (transcripts) as `_test.go`-side `var` blocks.
- **Conformance runs exercised**: `RunConformanceFor(..., agenttest.CapStreamingText)`, `CapToolCalls`, `CapCompletionMetadata`, `CapCancellation`, `CapTerminal` — the full required-surface set. Optional capabilities (`Reasoning`, `TokenCounting`, `CacheBoundary`) are declared `false` per R-CNF-004; the conformance suite's `applyDeclaredAbsences` mechanism records the three `absent` outcomes on the generated capability record.
- **Capability record check**: PR #2 ships a `TestOpenRouterAdapter_CapabilityRecordMatchesAI24` that reads the generated record from `agenttest/conformance_capabilities.go` and asserts equality with AI-24 §8's expected `absent` × 3 table. This is the AI-38.2 deliverable for this concrete adapter.
- **Rollback**: revert merge → wrapper alone still works (the bridge is test-only); the openaicompat package's own `bridge_test.go` continues to pass its own conformance runs against the generic adapter. AI-38's first-concrete-adapter claim is just deferred (re-resolvable by adding another factory in a follow-up change).

### PR #3 — Live smoke
- **Files**: `backend/agent/src/ai/openaicompat/openrouter/smoke_test.go`, plus a `smoke_setup.md` doc block inside the package's `doc.go`.
- **CI wiring**: a new `.github/workflows/agent-openrouter-smoke.yml` (workflow_dispatch only); `make test` in `backend/agent/` does **NOT** touch the smoke. Repo secret `OPENROUTER_API_KEY` is read at the workflow step, never logged. The smoke is `t.Skip`-gated on env absence; CI's normal `make test` skip path is also exercised (a CI run without the secret exercises the skip path; a workflow_dispatch run with the secret exercises the live path).
- **Sentinel sweep**: per AI-39.1 item 3, captured output (test logs + error messages) is scanned over a deny-list of forbidden tokens: the literal `OPENROUTER_API_KEY` env-var name, the secret value's prefix (4 chars), the full prompt bytes. A sentinel match fails the test.
- **Retry policy**: per the explore §2.10 risk, **max 3 retries** with exponential backoff; bounded total runtime under 60 seconds; this is opt-in, so failure surfaces as a workflow_dispatch-visible red rather than a flaky `make test`.
- **Rollback**: revert merge → no real-money calls in CI; `make test` is unaffected (skip path unchanged). Smoke PR is the most easily reversible.

## 6. Decisions table

Every row is **a resolved orchestrator-cached default** that the user can redirect at proposal review (per the explicit "Proposed default (user can redirect)" rule). Rows are grouped by the user's `C1–C4` confirmations and `Q1–Q6` questions.

| ID | Decision | Basis | Trade-off rejected | Approved by | Date |
|---|---|---|---|---|---|
| **C1** | **Default model for the live smoke = `openai/gpt-4o`** (non-reasoning, paid, present on OpenRouter) | Explore §2.7 + §5.3; OpenRouter's canonical streaming example; preserves AI-29's struck-node verdict; `openaicompat/chunk.go:213` plain `json.Unmarshal` already drops any `reasoning` / `reasoning_details` extension fields without code changes. | A reasoning-capable default (e.g. `anthropic/claude-3.7-sonnet:thinking`) — would **reopen AI-29** under trigger #1 of the struck decision and add a second `Factory` variant. Rejected as out-of-scope for this change; would also force a `reasoning_absence_test.go` parallel fixture. | User (via orchestrator prompt) | 2026-08-06 |
| **C2** | **Conformance + smoke use a paid model** (no `:free` suffix) | Explore §2.10 — `:free` is rate-limited to 20 req/min account-wide; CI flakiness would silently burn through the daily quota. ~$0.001 per smoke run is acceptable and explicit in §5 PR #3. | `:free` model — zero per-token cost but rate-limit flakiness makes CI unreliable; the explore flagged this as the most likely flake source. Rejected. | User (via orchestrator prompt) | 2026-08-06 |
| **C3** | **Attribution headers `HTTP-Referer`, `X-OpenRouter-Title`, `X-OpenRouter-Categories` are wrapper-injected, NOT openaicompat-package-injected** | Explore §2.3 — these are OpenRouter-only (no other shared-dialect vendor exposes them); openaicompat's `request.go:35` has a documented rule "`Authorization` and `Content-Type` are the only headers it sets", plus R-APC-001's "every injected value is used" guard would re-argue what an injected HTTP client already does. | Putting the headers in openaicompat (visible to all openaicompat-target vendors) — would widen openaicompat's header surface for one vendor's quirk and risk leaking attribution metadata to non-OpenRouter targets. Rejected. | User (via orchestrator prompt) | 2026-08-06 |
| **C4** | **`stream_options.include_usage` stays set** (cross-vendor uniformity), despite OpenRouter deprecating the field | Explore §2.5 + §4 — OpenRouter's OpenAPI says "Deprecated: This field has no effect. Full usage details are always included." It is harmless (OpenRouter ignores it, usage still arrives). The openaicompat posture is one wire body for all target vendors; dropping the field on OpenRouter would silently diverge that wire body for vLLM/Ollama/LocalAI. | Dropping the field on OpenRouter — would force AI-26.x to widen conditional logic and diverge the wire body across vendors. Rejected as cross-vendor-incoherent. | User (via orchestrator prompt) | 2026-08-06 |
| **Q1** | **Conformance parameter sweep = `openai/gpt-4o` only** (single model, single transcript family) | Explore §5.3 — Anthropic passthrough (`anthropic/claude-3.5-sonnet`) would exercise the `reasoning_details` drop path but is **exactly** the AI-29 reopen trigger. | Sweeping multiple models (e.g. Anthropic passthrough) — would reopen AI-29 and add a second `Factory` variant. Rejected; deferred to a future change that explicitly reopens AI-29 under trigger #1. | User (via orchestrator prompt) | 2026-08-06 |
| **Q2** | **API-key wiring pattern at composition root (Layer 3)**: Layer 3 reads `OPENROUTER_API_KEY` (or its config-file equivalent — Layer 3's decision) and passes the **opaque bearer string** into `openrouter.Config.Credential`. The wrapper itself reads nothing (AI-25.2 invariant, R-APC-008). | Memory #2432 §3 (AI-24.2 credential-handling boundary) + AI-25.2 spec line 165–168 (injected credential; Layer 3 owns origin). **Layer 3 is not yet built** — this change documents the *expected* pattern; Layer 3 implementation is a separate change. | Reading env directly in the wrapper — would violate AI-25.2's mechanical guard (the call-site scan would catch `os.Getenv`); would also create the ambient-authority footgun the spec exists to prevent. Rejected. Reading in a separate "credentials package" — premature; Layer 3 will own this; not this change's concern. Rejected. | User (via orchestrator prompt) | 2026-08-06 |
| **Q3** | **Wrapper's structural relationship to openaicompat = thin constructor wrapper (composition), as a sibling sub-package `backend/agent/src/ai/openaicompat/openrouter/`** | Explore §3.4 + AI-25.2's call-site scan scope — a sibling sub-package keeps the scan's scope clean (the wrapper only needs `net/http`, the openaicompat package, and `package ai`); allows the wrapper to inject attribution headers via its own `http.RoundTripper` wrapper without widening openaicompat's header surface. | Embedding inside openaicompat (same package) — would couple the wrapper to openaicompat's test-time only state and force the AI-25.2 call-site scan to widen to OpenRouter-specific header code. A separate top-level `openrouter/` package outside `openaicompat/` — would split `openaicompat/` namespace across two locations, complicating the layered-module convention. Rejected. | User (via orchestrator prompt) | 2026-08-06 |
| **Q4** | **The conformance bridge ships in PR #2 (this change)**, gated behind AI-38's own milestone charter | doc 0002 lines 2241–2277 — AI-38 is "Run the AI-23 suite against the **first concrete adapter** through replayed transcripts"; this change IS that first concrete adapter. AI-38 has no separate charter; its SDD change (`cachicamas-ai-adapter-conformance`) is this PR #2. | Deferring to a separate `cachicamas-ai-adapter-conformance` change — would land wrapper without AI-38.2's capability-record check; leaves doc 0002's completion-checklist item 14 ("the conformance suite can be reused for every future adapter" — AI-23 + AI-38.1) un-closed for OpenRouter. Rejected as the natural work-unit boundary is at PR #2. | User (via orchestrator prompt) | 2026-08-06 |
| **Q5** | **Live smoke is opt-in via GitHub workflow_dispatch + repo secret `OPENROUTER_API_KEY`**; `make test` in `backend/agent/` NEVER touches OpenRouter | doc 0002 lines 2279–2299 (AI-39.1 charter) + explore §3.6 — AI-39.1 item 1 ("Without credentials the smoke test skips cleanly — `make test` never depends on it") is a charter obligation, not optional. The smoke `t.Skip`s on `OPENROUTER_API_KEY` env absence inside the test code itself (the in-test skip is the primary gate; the workflow file is the secondary gate that ensures no scheduled run bills the user). | Always-on against a paid key in a secrets store — would silently bill on every CI run; violates AI-39.1's "make test never depends on it" charter. Rejected. | User (via orchestrator prompt) | 2026-08-06 |
| **Q6** | **Rollback posture per PR** (as detailed in §5): PR #1 = revert merge = no behavior change (no callers yet); PR #2 = revert merge = wrapper alone still works (bridge is test-only); PR #3 = revert merge = no real-money calls in CI (skip path unchanged) | Explore §5.2 — the chained-pr skill's "rollback boundary names the exact files/behavior removable without unrelated work" rule (work-unit-commits SKILL.md). The three PRs are independently deliverable with non-overlapping files. | Single-PR delivery — exceeds 800-line budget per corrected forecast; the explore §5.2 forecast is ~2200 lines corrected midpoint, which the chained-pr skill's rule forces us to slice. Rejected. | User (via orchestrator prompt) | 2026-08-06 |

## 7. Capabilities (contract with sdd-spec)

> **Contract.** The `sdd-spec` agent reads this section to know exactly which spec files to create or update.

### New Capabilities
- **`ai-openrouter-first-provider`** (kebab-case) → becomes `openspec/specs/ai-openrouter-first-provider/spec.md`. Full spec covering: the OpenRouter sub-package's surface, the bridge factory, the capability-record assertions (against AI-24 §8), the live smoke's gating discipline + sentinel sweep + retry policy, and the doc-0002 traceability to AI-25 / AI-38 / AI-39.

### Modified Capabilities
- **None.** This change does NOT modify any shipped spec. `ai-provider-client` (AI-25), `ai-stream-testkit` (AI-22), `ai-provider-conformance-suite` (AI-23 / AI-38), `ai-stream-lifecycle`, `ai-event-envelope`, `ai-minimum-capabilities`, `ai-model-provider` (AI-20) — all are read-only. The wrapper composes the existing `Config`; the bridge uses the existing `Factory`; the smoke uses the existing `Credential` redaction. No existing spec's requirement text changes.

> Note for sdd-spec: if a future spec discovers that the AI-38.2 capability-record assertion against AI-24 §8's "absent × 3" expected outcomes reveals an actually-satisfied capability, that is **trigger #1** of AI-29's struck decision and warrants reopening AI-29 under a separate change — not a delta on `ai-openrouter-first-provider`.

## 8. Approach

**Compose, do not reimplement.** The shipped `backend/agent/src/ai/openaicompat/` package is the entire reusable adapter; the OpenRouter wrapper is a sub-100-line constructor + a 30-line `http.RoundTripper` that injects the three attribution headers when set. The bridge's transcript renderers are the existing `openaicompat/bridge_test.go` patterns lifted into a sibling `openrouter/bridge_test.go` with the OpenRouter-specific envelope (the three headers) added around the same byte-rendering path. The live smoke is one test function with a `t.Skip` guard and a sentinel sweep — analogous to AI-39.1's charter.

**Preserve AI-29's struck verdict** by choosing a non-reasoning model. **Preserve AI-00.3's forward guard** by adding zero `go.mod` requires. **Preserve AI-25.2's call-site guard** by re-running the call-site scan against the new sub-package's own sources (the scan's coverage list must be widened to include `backend/agent/src/ai/openaicompat/openrouter/` non-test files).

**Reuse conformance infrastructure.** `agenttest.RunConformance` and `agenttest.Factory` are the existing AI-23 surface; the new factory conforms to it. The capability record from `agenttest/conformance_capabilities.go` is the AI-38.2 deliverable; PR #2 asserts it equals AI-24 §8's expected `absent × 3` table.

## 9. Affected areas (file paths)

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/ai/openaicompat/openrouter/provider.go` | **New** | `Config`, `NewProvider`, attribution-header `RoundTripper`. ~150–200 lines. |
| `backend/agent/src/ai/openaicompat/openrouter/provider_test.go` | **New** | R-APC-001 stub-transport probe, three-header empty/non-empty matrix. ~80–150 lines. |
| `backend/agent/src/ai/openaicompat/openrouter/ambient_authority_test.go` | **New** | AI-25.2 call-site scan re-run against the new sub-package. ~60–100 lines. |
| `backend/agent/src/ai/openaicompat/openrouter/doc.go` | **New** | Package doc: three attribution headers, non-blocking model-conditional reasoning field dropped silently, "first concrete vendor = OpenRouter". ~30–50 lines. |
| `backend/agent/src/ai/openaicompat/openrouter/bridge_test.go` | **New** | OpenRouter-shaped transcript renderers + `conformanceBridgeFactory`. ~200–350 lines. |
| `backend/agent/src/ai/openaicompat/openrouter/conformance_test.go` | **New** | Per-capability `RunConformanceFor` drivers + capability-record assertion. ~80–150 lines. |
| `backend/agent/src/ai/openaicompat/openrouter/smoke_test.go` | **New** | `TestOpenRouterAdapter_LiveSmoke` + sentinel-sweep helper. ~60–100 lines. |
| `.github/workflows/agent-openrouter-smoke.yml` | **New** | `workflow_dispatch` only; reads repo secret; bounded timeout; never scheduled. ~20–40 lines. |
| `backend/agent/src/ai/openaicompat/` | Unchanged | All shipped files (`client.go`, `request.go`, `bridge_test.go`, etc.) — wrapper composes them, does NOT modify them. |
| `backend/agent/src/agenttest/` | Unchanged | Reads existing `Factory`, `RunConformance`, `RunConformanceFor`, `Script`, `conformance_capabilities.go`. |
| `backend/agent/go.mod` | Unchanged | Zero `require` lines (NFR-APC-A preserved). |
| `backend/agent/src/ai/import_boundary_test.go` | Unchanged | `allowedNonStdlibPrefixes` stays at one entry (the module itself); AI-00.3 guard stays green. |

## 10. Risks

| Risk | Likelihood | Severity | Mitigation |
|---|---|---|---|
| **AI-29 reopen trigger #1 fires** — default model `openai/gpt-4o` is documented to NOT carry reasoning, but OpenRouter routing can land on a reasoning-capable provider (e.g. Anthropic passthrough) and emit `delta.reasoning_details`. | Low | Medium | Per explore §2.7, the conformance bridge's recorded transcript uses `openai/gpt-4o` explicitly; the `decodeChunk` plain `json.Unmarshal` already drops the extension field silently. The capability record check in PR #2 (`TestOpenRouterAdapter_CapabilityRecordMatchesAI24`) asserts `CAP-O-01 = absent`; if a generated record ever shows `satisfied`, that is **trigger #1** of the struck decision and routes to a separate AI-29-reopen change. |
| **Credential boundary violation** — wrapper code accidentally reads `OPENROUTER_API_KEY` from env. | Low | High | AI-25.2's call-site scan (`ambient_authority_test.go`) re-runs against the new sub-package's non-test sources; R-APC-008's four bite proofs must pass on the new package. |
| **AI-00.3 forward guard regression** — wrapper introduces a new top-level dep. | Very Low | High | The wrapper imports only stdlib + `package ai` + `openaicompat` (an in-module package). NFR-APC-A is a non-negotiable gate at apply time. |
| **Attribution-header leakage to non-OpenRouter vendors** — if a future caller mistakenly uses the `openrouter.NewProvider` factory against a different vendor's endpoint. | Very Low | Low | The wrapper's `Endpoint` is fixed to `https://openrouter.ai/api/v1` (no `Endpoint` field on the public `openrouter.Config`); passing a different URL is impossible without modifying the wrapper. Documentation states this. |
| **`:free` rate-limit flakiness in CI** — accidentally picking a `:free` model variant. | Low (defaults are paid) | Medium | PR #1's Config struct has no `Model` field; the model identifier is passed at `Request` time (Layer 1's neutral shape). PR #3's smoke targets a single paid model explicitly. CI's retry policy caps at 3 attempts. |
| **`stream_options.include_usage` deprecation drift** — OpenRouter removes the field entirely in a future API version. | Very Low | Low | Per the explore §2.5 decision (C4), the field is harmless on OpenRouter today; if it ever returns a parse error, the openaicompat package's body-builder would need a `MarshalJSON` change — a future-change concern, not this change's. |
| **Tracker-PR chain drift** — child PR #2 or #3 accumulates the previous PR's diff in its own diff, violating the chained-pr skill's "polluted diffs are base bugs" rule. | Medium | Medium | The orchestrator rebases each child onto its immediate parent before review; if any child shows the previous PR's bytes, retarget or rebase until clean. |
| **Live smoke secrets leak via `t.Logf`** — accidentally printing the credential or the full prompt. | Low | High | The smoke uses `openaicompat.Credential` whose `String()` returns `<redacted>` (R-APC-014, R-APC-053). The sentinel sweep (`smoke_test.go`) scans captured output against a deny-list; a sentinel match fails the test. |
| **The 800-line budget is insufficient for PR #1 authored code** | Medium | Medium | Per the corrected forecast §5, PR #1 authored code is ~200 lines (~1100 corrected mid including tests). The sdd-tasks phase re-forecasts with exact line counts; if PR #1 still exceeds 800 corrected, slice further by deferring the ambient-authority scan re-run to a follow-up commit (the wrapper alone is the load-bearing piece for the conformance bridge). |
| **`error_type` vocabulary mismatch surfaces in production** | Low | Low | AI-32 widening is explicitly out of scope (explore §2.8); the openaicompat `failure_map.go` reads HTTP status + standard error envelope, which works on OpenRouter wire. A future change widens AI-32 against the OpenRouter-specific vocabulary. |

## 11. Acceptance criteria

**What `sdd-verify` will check**:

### Per-PR
- **PR #1**: `cd backend/agent && make test` green under `-race`; `make lint` clean; `go.mod` still zero `require`; AI-00.3 forward guard passes; AI-25.2 call-site scan re-runs against the new sub-package with all four bite proofs (plain, aliased, dot-import, process spawn) re-recorded as passing; AI-20 signature guard passes unmodified; the three attribution headers are observable on the wire when non-empty and absent when empty (R-APC-001's stub-transport probe); one-request against a local `httptest.Server` reaches it with `Authorization: Bearer <token>` and the joined path (R-APC-013).
- **PR #2**: `make test` green; `agenttest.RunConformance` runs end-to-end against the OpenRouter-shaped factory and passes every required capability (`CapStreamingText`, `CapToolCalls`, `CapCompletionMetadata`, `CapCancellation`, `CapTerminal`); the capability record equals AI-24 §8's expected `absent × 3` table (`TestOpenRouterAdapter_CapabilityRecordMatchesAI24`); AI-20 signature guard still passes unmodified; reverting PR #2 leaves the openaicompat package's own `bridge_test.go` runs passing identically.
- **PR #3**: `make test` green WITHOUT `OPENROUTER_API_KEY` set (skip path); `workflow_dispatch` run WITH the secret set: one bounded request reaches OpenRouter, asserts stream-shape invariants (start, at least one content event, exactly one terminal), bounded under 60 seconds, no sentinel leakage; the workflow file has no `schedule:` trigger and no `push:` / `pull_request:` trigger that would run on every commit; sentinel-sweep helper passes a mutation test (a deliberately-injected `t.Logf(key)` fails the sweep).

### Overall (after all three PRs merged)
- `make test` green; `make lint` clean; `go.mod` zero `require`; both AI-00 import guards pass; AI-20 signature guard passes unmodified; AI-25.2's call-site scan covers the new sub-package; the AI-38.2 capability record is committed and equals `absent × 3`; the AI-39.1 smoke skip path is exercised in `make test`; the AI-39.1 smoke live path is exercised in a separate `workflow_dispatch` run.
- doc 0002's completion-checklist item 14 ("reusable conformance" — AI-23 + AI-38.1) closes against this change; items 15 ("first adapter passes" — AI-38) and 17 ("live test optional and unreachable" — AI-39.1) close against this change; item 6's wire half remains open (restated by AI-40.2's amendment, deferred to AI-40).

## 12. References

- **Explore artifact** (this change's foundation): `openspec/changes/add-openrouter-first-provider/explore.md` and Engram observation **#2568** (`sdd/add-openrouter-first-provider/explore`).
- **AI-24 vendor/transport decision** (pre-decided): Engram observation **#2432** (`sdd/cachicamas-ai-first-provider-decision/explore`).
- **AI-16 provider interface** (the `ModelProvider` contract): Engram observation **#2235** (`sdd/cachicamas-ai-model-provider/proposal`).
- **AI-29 reasoning struck-node decision** (2026-08-04): `openspec/changes/archive/2026-08-04-cachicamas-ai-provider-reasoning-stream/decision.md` §5 row 4 + §7 + §9 (triggers).
- **AI-25 spec** (`ai-provider-client`): `openspec/specs/ai-provider-client/spec.md` — `R-APC-001…014`, `NFR-APC-A…G`, the four bite proofs at `R-APC-010`.
- **AI-20 spec** (provider interface): `openspec/specs/ai-model-provider/spec.md` — `R-AMP-001…021`, signature guard at `R-AMP-014…016`.
- **AI-22 spec** (stream testkit, the carrier view + leak helper): `openspec/specs/ai-stream-testkit/spec.md` — `R-STK-001…013`.
- **doc 0002** (Layer 1 task graph): `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` — AI-25 lines 1447–1491, AI-38 lines 2241–2277, AI-39 lines 2279–2299, AI-29 amendment 2026-08-04 at line 1759, AI-26.2 amendment at line 1530, AI-26.5 amendment at line 1556, AI-40.2 amendment at line 2334.
- **ADR 0004** (3-layer architecture): Engram observation **#1997** + `docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md`.
- **Backend test runner** (canonical): Engram observation **#2055** + `backend/agent/Makefile` (test runner = `make test` = `go test -race -v ./...`).
- **OpenRouter docs cited** (per explore §2):
  - Base URL + endpoint: https://openrouter.ai/docs/api-reference/overview
  - Authentication: https://openrouter.ai/docs/api-reference/authentication
  - Streaming: https://openrouter.ai/docs/api-reference/streaming
  - Attribution headers (`HTTP-Referer`, `X-Title`/`X-OpenRouter-Title`, `X-OpenRouter-Categories`): https://openrouter.ai/docs/api-reference/overview, "Headers" section; OpenAPI `components/parameters/AppIdentifier`, `AppDisplayName`, `AppCategories`
  - Model variants (`:free`, `:thinking`, `:nitro`, `:online`, `:extended`, `:exacto`): https://openrouter.ai/docs/guides/routing/model-variants
  - Free-tier rate limits: https://openrouter.ai/docs/guides/routing/model-variants/free
  - Errors + mid-stream error envelope: https://openrouter.ai/docs/api-reference/errors-and-debugging
  - Tool calling: https://openrouter.ai/docs/guides/features/tool-calling
  - Reasoning (ext): https://openrouter.ai/docs/guides/best-practices/reasoning-tokens
  - Quickstart: https://openrouter.ai/docs/quickstart
  - OpenAPI spec: https://openrouter.ai/openapi.yaml
- **`openspec/AGENTS.md`**: Go 1.26.3 layered module; conventional commits; no Co-Authored-By; no new top-level deps without an ADR; TDD red→green→refactor enforced at apply time.
- **`openspec/config.yaml`**: `rules.proposal` — include rollback plan for every change, state out-of-scope items explicitly. `rules.apply` — no new top-level deps without an ADR; `tdd: true`.

---

## 13. Key Learnings (for engram passive capture)

1. The `backend/agent/src/ai/openaicompat/` package is a fully-shipped generic OpenAI-compatible adapter (zero `go.mod` requires); OpenRouter concretization is three small work units (wrapper / bridge / smoke), not one monolithic change.
2. AI-29's struck-node verdict (2026-08-04) is preserved by targeting `openai/gpt-4o`; OpenRouter's `reasoning` / `reasoning_details` extension fields are dropped silently by `openaicompat/chunk.go:213`'s plain `json.Unmarshal` without code changes.
3. Three OpenRouter-only divergences (`stream_options.include_usage` deprecation, three attribution headers, `error_type` discriminator) all have deliberate decisions: keep the field set (cross-vendor wire body), wrapper-inject the headers (keep openaicompat surface narrow), defer `error_type` widening to a future AI-32 change.
4. The 800-line budget is insufficient for `single-pr`; the natural split is **three chained PRs under a no-merge tracker**, matching AI-25 / AI-38 / AI-39's milestone boundaries. Generated goldens and recorded fixture bytes are excluded from the per-PR authored risk count per `sdd-phase-common.md §E`.
5. The `openaicompat/bridge_test.go` factory pattern (`conformanceBridgeFactory`) is the template for the OpenRouter bridge — same byte-rendering path, OpenRouter-specific envelope (the three attribution headers) added around it. Reuse, do not reinvent.
