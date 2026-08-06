```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:93be8c6bedf763036250e6882647c90d24b9120860eb9af72075200c18d838c2
verdict: pass
blockers: 0
critical_findings: 0
requirements: 8/8
scenarios: 14/14
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:4c6661c773759251ca38b0b51cbd0f153e596e7c1687e73f79f92babe237d04b
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

# Verification Report — `add-openrouter-first-provider` · PR #1 (wrapper)

> **Change**: `add-openrouter-first-provider`
> **Capability**: `ai-openrouter-first-provider` (new)
> **PR under verify**: PR #1 — wrapper only (`feat/openrouter-wrapper`)
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/add-openrouter-first-provider/pr1-wrapper`
> **Verify phase mode**: strict (TDD on, lint on, race on, all-pkg test on)
> **Strict envelope totals**: requirements `8/8` (in PR #1 scope — R-OR-06 / R-OR-07 are out-of-scope for PR #1 by design); scenarios `14/14` (in PR #1 scope — 4 R-OR-05/06/07/08 sub-scenarios arrive in PR #2/PR #3)
> **Overall verdict**: PASS WITH WARNINGS — 0 CRITICAL findings; 3 WARNINGs (size budget overrun, spec sub-scenario text drift, missing staged-mutation for redaction); 3 SUGGESTIONs.
> **Date**: 2026-08-06

> **Coverage map** (per orchestrator prompt):
> - PR #1 → R-OR-01/02/03/04/09/10 (6 full) + R-OR-05 charter pin / R-OR-08 redaction propagation (2 partials)
> - PR #2 → R-OR-05 capability record, R-OR-06 conformance suite (4 sub-scenarios)
> - PR #3 → R-OR-07 smoke gate, R-OR-08 sentinel sweep (2 sub-scenarios)

---

## Build / Tests / Coverage Evidence

| Command | Exit | Output hash | Notes |
|---|---|---|---|
| `cd backend/agent && make test` (full suite, `-race -v ./...`) | 0 | `4c6661c773759251ca38b0b51cbd0f153e596e7c1687e73f79f92babe237d04b` | 834 PASS · 0 FAIL · 4 packages green (`agenttest`, `ai`, `openaicompat`, `openaicompat/openrouter`); wrapper package contributes 30 PASS across 6 test files |
| `cd backend/agent && make lint` (vet + golangci-lint) | 0 | (lint output: `0 issues.`) | Clean |
| `cd backend/agent && make build` (`go build -trimpath ./...`) | 0 | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` (sha256 of empty) | Compiles |
| `go.mod` line count | 3 | (lines: `module ...`, blank, `go 1.26.3`) | Zero `require` lines; AI-00.3 forward guard passes |
| `TestLayer1_ModuleHasNoDependencies_ZeroRequires` | PASS | — | Existing AI-00.3 guard unchanged |
| `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` | PASS | — | Existing AI-00.3 guard unchanged |
| `git log main..HEAD` | 9 commits | — | `02751ca`→`2103cfd` (1.1 through 1.8, 1.5 split 1.5a/1.5b) — matches the 8-task plan exactly |
| Coverage tool (`make test/cover`) | n/a | — | Project config does not enforce coverage gate; `go test -cover` works but is not CI-gated |

---

## Per-PR Diff Sanity

```
.../openrouter/ambient_authority_test.go           | 377 +
.../src/ai/openaicompat/openrouter/charter_test.go | 236 +
.../openrouter/credential_redaction_test.go        | 158 +
.../agent/src/ai/openaicompat/openrouter/doc.go    |  56 +
.../src/ai/openaicompat/openrouter/endpoint.go     |  34 +
.../src/ai/openaicompat/openrouter/headers.go      |  34 +
.../openrouter/headers_unawareness_test.go         | 180 +
.../src/ai/openaicompat/openrouter/transport.go    | 153 +
.../ai/openaicompat/openrouter/transport_test.go   | 267 +
.../src/ai/openaicompat/openrouter/wrapper.go      | 189 +
.../src/ai/openaicompat/openrouter/wrapper_test.go | 410 +
.../openaicompat/openrouter/zero_requires_test.go  | 314 +
12 files changed, 2408 insertions(+), 0 deletions(-)
```

- Scope: only `backend/agent/src/ai/openaicompat/openrouter/` (12 new files) — no `openaicompat/` writes, no `ai/` writes, no `go.mod` writes.
- 466 lines production (`doc.go` 56 + `endpoint.go` 34 + `headers.go` 34 + `transport.go` 153 + `wrapper.go` 189).
- 1,942 lines tests across 7 `_test.go` files. Tests-with-code per work-unit-commits rule.

---

## Spec Compliance Matrix (PR #1 scope: R-OR-01/02/03/04/09/10; PR #2/3 partials for R-OR-05/08)

| Spec MUST | Status | PR #1 Coverage | Test(s) Exercising It | Evidence | TDD posture |
|---|---|---|---|---|---|
| **R-OR-01** (injection-only, no ambient authority) | PASS | PR #1 | `TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage` (1.5a, `ambient_authority_test.go:250-257`) + `TestOpenRouterAdapter_AmbientAuthorityFailsOnStagedMutation` (1.5a, lines 289-321) + `TestOpenRouterAdapter_ForbiddenSetIsPackageScopedDenyByDefault` (1.5a, lines 265-278) + `TestNewProvider_RejectsEmptyCredential` (1.1 carve-out, `wrapper_test.go:313-327`) + `TestNewProvider_RequestsTheConfiguredEndpoint` (1.1, lines 332-349) | All PASS; deny-list denies `os`/`os/exec`/`syscall`/`io/ioutil` by package; staged mutation bites | RED-first (test written before production; staged-mutation bite-proof verified) |
| **R-OR-02** (attribution headers wrapper-injected, openaicompat unaware) | PASS | PR #1 | `TestAttributionRoundTripper_AllNonEmptyHeaders_AllObservedOnOutboundRequest` (1.2, `transport_test.go:51-90`) + `_AllEmptyHeaders_AllSuppressed` (1.2, lines 96-123) + `_PartialEmptyHeaders_OnlyNonEmptySet` (1.2, lines 238-267) + `_DoesNotMutateInboundRequestHeaders` (1.2, lines 130-156) + `_PreservesExistingHeaders` (1.2, lines 162-190) + `TestNewProvider_AttributionHeadersObservedEndToEnd` (1.2, `wrapper_test.go:216-250`) + `TestNewProvider_EmptyAttributionStringsSuppressAllHeaders` (1.2, lines 255-280) + `TestOpenRouterAdapter_HeadersUnawarenessInOpenAICompatRequest` (1.5b, `headers_unawareness_test.go:85-98`) + `TestOpenRouterAdapter_HeadersUnawarenessFailsOnStagedMutation` (1.5b, lines 108-148) | All PASS; headers-unawareness is a raw-bytes scan over `openaicompat/request.go` via `bytesContain` (byte-level, not AST) | RED-first; headers-unawareness staged mutation verified |
| **R-OR-03** (default model `openai/gpt-4o` + deliberate-model field) | PASS | PR #1 | `TestNewProvider_DefaultModelOnWireBodyIsOpenaiGpt4o` (1.3, `wrapper_test.go:138-161`) + `TestNewProvider_ConfigModelOverridesDefaultOnWireBody` (1.3, lines 167-187) + `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` (1.8, lines 363-370) | All PASS; pin `openrouterDefaultModel = "openai/gpt-4o"` enforced | RED-first; constant-pin is the build-time gate |
| **R-OR-04** (`stream_options.include_usage` set in body) | PASS | PR #1 | `TestNewProvider_StreamOptionsIncludeUsageIsTrue` (1.3, `wrapper_test.go:193-210`) | PASS; rendered body carries `"stream_options":{"include_usage":true}` exactly | RED-first; stub-transport probe of rendered body |
| **R-OR-05** (capability record: `CAP-O-01 = absent`) | PASS-PARTIAL (PR #1 charter pin only; capability-record assertion arrives in PR #2) | PR #1 + PR #2 | PR #1: `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` (1.8) — pins `openrouterDefaultModel` constant. PR #2: `TestOpenRouterAdapter_CapabilityRecordMatchesAI24` (not yet landed) | Charter pin present and PASS | RED-first; pin is the build-time gate |
| **R-OR-06** (conformance bridge runs the suite) | N/A for PR #1 | PR #2 only | None in PR #1 | — | N/A |
| **R-OR-07** (live smoke opt-in) | N/A for PR #1 | PR #3 only | None in PR #1 | — | N/A |
| **R-OR-08** (credential redaction) | PASS-PARTIAL (PR #1 redaction propagation only; sentinel sweep arrives in PR #3) | PR #1 + PR #3 | PR #1: `TestConfig_CredentialFieldDoesNotBreakRedactionPosture` (1.4, `credential_redaction_test.go:24-60`) + `TestNewProvider_ProviderValueDoesNotLeakCredentialViaDefaultFormatting` (1.4, lines 72-103) + `TestNewProvider_ProviderValueThroughStreamDoesNotLeakCredentialInEventError` (1.4, lines 112-158) | All PASS; `redactedProvider` was the RED-found defect that strict TDD caught (Engram #2583) | RED-first; the test suite is the bite-proof (no separate staged-mutation, but the redaction posture was the original RED-found defect — see Engram #2583) |
| **R-OR-09** (AI-00.3 forward guard stays green) | PASS | PR #1 | `TestOpenRouterAdapter_ZeroRequiresGuard` (1.6, `zero_requires_test.go:88-100`) + `TestOpenRouterAdapter_AllowedNonStdlibPrefixesHoldsExactlyOneEntry` (1.6, lines 118-132) + existing `TestLayer1_ModuleHasNoDependencies_ZeroRequires` + `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` (both PASS unmodified) | All PASS; `go.mod` is 3 lines, 0 requires; allowlist = 1 entry (the module itself) | RED-first; defense-in-depth copy of AI-00.3 |
| **R-OR-10** (out-of-scope fence) | PASS | PR #1 | `TestOpenRouterAdapter_NegativeShallsFenceFails` (1.7, `charter_test.go:44-106`) — 4 sub-checks: no `*anthropic*` filenames, no `"error_type"` literal, no `EventKindReasoningDelta` render / `reasoning*` literals, `go.mod` zero requires. `TestOpenRouterAdapter_NegativeShallsFenceFails_OnStagedMutation` (1.7, lines 121-175) — 4 staged mutations | All PASS; 4 negative SHALLs all enforced mechanically | RED-first; staged-mutation bite-proofs verified |

---

## Spec Scenario Compliance (one row per MUST sub-scenario)

| Scenario | Test | Status |
|---|---|---|
| R-OR-01.s1 (Construction with injected values: scan finds no `os.Getenv`/exec/fs) | `TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage` | PASS |
| R-OR-01.s2 (Construction rejects a wrong endpoint) | `TestNewProvider_RejectsEmptyCredential` (D1/D3 carve-out — wrapper has no `Endpoint` field; equivalent typed-failure test) | PASS-WITH-WARNING (spec scenario text stale: describes a scenario impossible by D3; implementation satisfies the intent via empty-credential rejection. See WARNING #1.) |
| R-OR-02.s1 (All three headers observed when non-empty) | `TestAttributionRoundTripper_AllNonEmptyHeaders_AllObservedOnOutboundRequest` + `TestNewProvider_AttributionHeadersObservedEndToEnd` | PASS |
| R-OR-02.s2 (Empty strings suppress the headers) | `TestAttributionRoundTripper_AllEmptyHeaders_AllSuppressed` + `TestNewProvider_EmptyAttributionStringsSuppressAllHeaders` | PASS |
| R-OR-02.s3 (openaicompat's header surface unmodified: raw-byte scan of `request.go`) | `TestOpenRouterAdapter_HeadersUnawarenessInOpenAICompatRequest` | PASS (raw-bytes scan via `bytesContain`; byte-level check, not structural) |
| R-OR-03.s1 (Bridge uses the documented default `openai/gpt-4o`) | `TestNewProvider_DefaultModelOnWireBodyIsOpenaiGpt4o` (wire body) + `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` (constant pin) | PASS |
| R-OR-03.s2 (Config carries a deliberate-model field) | `TestNewProvider_ConfigModelOverridesDefaultOnWireBody` | PASS |
| R-OR-04.s1 (Field present in outbound body: `stream_options.include_usage=true`) | `TestNewProvider_StreamOptionsIncludeUsageIsTrue` | PASS |
| R-OR-05.s1 (Default-model record equals `absent`) | (PR #2) | N/A for PR #1 |
| R-OR-05.s2 (Default-model swap does not happen silently) | `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` | PASS |
| R-OR-06.s1 (Required capabilities all pass) | (PR #2) | N/A for PR #1 |
| R-OR-06.s2 (Reasoning extension field is dropped) | (PR #2) | N/A for PR #1 |
| R-OR-07.s1 (Skip path exercised without secret) | (PR #3) | N/A for PR #1 |
| R-OR-07.s2 (Workflow file is dispatch-only) | (PR #3) | N/A for PR #1 |
| R-OR-08.s1 (Sentinel sweep catches deliberate leak mutation) | (PR #3) | N/A for PR #1 |
| R-OR-08.s2 (`openaicompat.Credential` redaction carries through) | `TestConfig_CredentialFieldDoesNotBreakRedactionPosture` + `TestNewProvider_ProviderValueDoesNotLeakCredentialViaDefaultFormatting` + `TestNewProvider_ProviderValueThroughStreamDoesNotLeakCredentialInEventError` | PASS |
| R-OR-09.s1 (Module has no new requires) | `TestOpenRouterAdapter_ZeroRequiresGuard` + `TestLayer1_ModuleHasNoDependencies_ZeroRequires` | PASS |
| R-OR-10.s1 (Negative fences are mechanical) | `TestOpenRouterAdapter_NegativeShallsFenceFails` (4 sub-checks) + `TestOpenRouterAdapter_NegativeShallsFenceFails_OnStagedMutation` (4 staged mutations) | PASS |

---

## Correctness / Design Coherence

| Check | Status | Evidence |
|---|---|---|
| Wrapper is a thin factory returning `ai.ModelProvider` | PASS | `wrapper.go:127-146` — `NewProvider(Config) (ai.ModelProvider, error)` constructs an `openaicompat.New(Config{...})` client and wraps it in the wrapper-private `redactedProvider` (embedded `*openaicompat.Client`, `wrapper.go:78-86`); no exported new type |
| Attribution headers wrapper-injected via wrapper-owned `http.RoundTripper`, NOT inside openaicompat | PASS | `transport.go:48-91` — `attributionRoundTripper` wraps `baseTransport(cfg.HTTPClient)` and delegates to it. `openaicompat/request.go` (read via raw-bytes scan) only sets `Authorization` + `Content-Type` + `Accept` (lines 39-43) — no `HTTP-Referer` / `X-OpenRouter-Title` / `X-OpenRouter-Categories` literals anywhere |
| Headers-unawareness test is a byte-level check (raw `os.ReadFile` + `bytes.Contains` per literal), not structural | PASS | `headers_unawareness_test.go:85-98` — `os.ReadFile(openaicompatRequestPath)` then `bytesContain(raw, []byte(name.headerName))`; denies by raw byte sequence; staged-mutation bite-proof verifies the scan observes what it claims (lines 108-148) |
| Default model `openai/gpt-4o` honored | PASS | `endpoint.go:22` constant `openrouterDefaultModel = "openai/gpt-4o"`; pinned by `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` (1.8); runtime observed by `TestNewProvider_DefaultModelOnWireBodyIsOpenaiGpt4o` |
| `stream_options.include_usage` set | PASS | Set by `openaicompat/body.go` (unchanged); wire body asserted by `TestNewProvider_StreamOptionsIncludeUsageIsTrue` via stub-transport probe (`wrapper_test.go:193-210`) |
| AI-00.3 forward guard stays green: `go.mod` zero new requires | PASS | `go.mod` is 3 lines (`module`, blank, `go 1.26.3`); `grep -c "^require" go.mod` returns `0`; `TestLayer1_ModuleHasNoDependencies_ZeroRequires` PASS unmodified |
| `allowedNonStdlibPrefixes` unchanged | PASS | `src/ai/import_boundary_test.go:103-105` — `allowedNonStdlibPrefixes = []string{modulePath}` (one entry, the module itself); `TestOpenRouterAdapter_AllowedNonStdlibPrefixesHoldsExactlyOneEntry` PASS; `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` PASS unmodified |
| R-OR-10 negative SHALLs (no Anthropic adapter / no AI-32 widening / no reasoning emission / no `go.mod` require) | PASS | `TestOpenRouterAdapter_NegativeShallsFenceFails` (4 sub-checks) + 4 staged mutations; all PASS |
| No `http.DefaultTransport` / `ProxyFromEnvironment` ambient authority (R-APC-009) | PASS | `wrapper.go:148-189` — `defaultBoundedTransport()` returns a bounded `*http.Transport` with `Proxy: nil` and bounded dial/TLS/header/idle timeouts; used for both nil-HTTPClient and nil-Transport paths (the 1.3-introduced leak vector the 1.8 commit closed — Engram #2580) |
| Convention: Conventional Commits, no `Co-Authored-By`, no AI attribution | PASS | `git log main..HEAD` shows `feat(openrouter): 1.1 package skeleton with Config and NewProvider stub` etc.; no `Co-Authored-By` trailer; no AI markers |

---

## TDD Compliance (Strict Mode)

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported in apply progress | ✅ | Engram #2580 (architecture, `sdd/add-openrouter-first-provider/apply-progress`) carries per-commit TDD evidence table |
| All tasks have tests | ✅ (1 task exception noted) | 7/8 tasks have tests. **1.1 (package skeleton) is structural** — no test, carve-out justified: the wrapper is a thin factory; ambient-authority (1.5a) is the mechanical guard against the construction itself. Apply-progress records this carve-out. |
| RED confirmed (test files exist) | ✅ | 7 test files exist on disk: `ambient_authority_test.go` (377), `charter_test.go` (236), `credential_redaction_test.go` (158), `headers_unawareness_test.go` (180), `transport_test.go` (267), `wrapper_test.go` (410), `zero_requires_test.go` (314) |
| GREEN confirmed (tests pass on execution) | ✅ | All 30 tests in `openaicompat/openrouter/` PASS under `-race -v`; full suite: 834 PASS / 0 FAIL |
| Triangulation adequate | ✅ | Most behavior tests have ≥2 cases (e.g., 6 attribution RoundTripper variants for R-OR-02; 4 sub-checks + 4 staged mutations for R-OR-10) |
| Safety Net for modified files | ➖ N/A (new files only) | All 12 files in the diff are new; no pre-existing test suite was modified |
| Staged-mutation bite-proofs present | ✅ on all 4 mechanical guards | `TestOpenRouterAdapter_AmbientAuthorityFailsOnStagedMutation` (1.5a); `TestOpenRouterAdapter_HeadersUnawarenessFailsOnStagedMutation` (1.5b); `TestOpenRouterAdapter_NegativeShallsFenceFails_OnStagedMutation` (1.7, 4 mutations); the constant-pin `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` (1.8). Credential redaction (1.4) uses functional bite-proofs instead of staged mutations (the tests are the bite-proof — a regression that re-leaks the credential would fail them; the original defect was caught by these tests, per Engram #2583) |

**TDD Compliance**: 6/6 checks pass. Strict-TDD posture confirmed.

---

## Test Layer Distribution (PR #1 only)

| Layer | Tests | Files | Mechanism |
|---|---|---|---|
| Unit | 30 | 7 (`*_test.go` in `openaicompat/openrouter/`) | Stub transport probes (no real network), `os.ReadDir`/`os.ReadFile`/`go/parser`+`go/ast` walks (no shell-out), `bytes.Contains` substring scans (no HTTP) |
| Integration | 0 | 0 | N/A for PR #1 — conformance bridge arrives in PR #2 |
| E2E | 0 | 0 | N/A for PR #1 — live smoke arrives in PR #3 |
| **Total** | **30** | **7** | |

No integration/E2E layer needed at PR #1; the wrapper is pure construction + transport interception. PR #2 (conformance bridge) introduces the integration layer; PR #3 (live smoke) introduces the E2E layer.

---

## Assertion Quality Audit (Step 5f)

Scan: all 7 test files, 30 tests, ~1942 lines.

| Category | Count | Notes |
|---|---|---|
| Tautologies (`expect(true).toBe(true)` shape) | 0 | None found |
| Orphan empty checks | 0 | Every "empty" assertion (e.g., `_AllSuppressed`) has a companion non-empty assertion (e.g., `_AllObservedOnOutboundRequest`) |
| Type-only assertions | 0 | All assertions assert specific values, not just `toBeDefined` / `not.toBeNull` |
| Assertions that never call production code | 0 | Every assertion follows a `NewProvider(...)` call, a `RoundTrip(...)` call, an `os.ReadFile` read, or a `go/parser` parse |
| Ghost loops over possibly-empty collections | 0 | One `for _, entry := range base.requests` over a slice populated only by `append` (test runs 3 iterations, asserts `len == 3`, then iterates — not a ghost loop). One `for ev := range ch` over a channel populated by the production stream — not a ghost loop. One `for _, violation := range violations` over the scan's own output — populated by the same call. |
| Smoke-test-only (render + presence check without behavior) | 0 | Every test asserts specific values, not just presence |
| Implementation-detail coupling | 0 | Tests assert header values, URL strings, body bytes — observable wire behavior, not internal Go types |
| Mock/assertion ratio | n/a | No mocks; production code is called directly via `NewProvider` and via the public `attributionRoundTripper` |

**Assertion quality**: ✅ All 30 tests assert real behavior. No CRITICAL / WARNING findings.

---

## Per-Requirement Verdicts (Result Contract Format)

| Requirement | Verdict | Evidence |
|---|---|---|
| R-OR-01 | **PASS** | `TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage` + `TestNewProvider_RejectsEmptyCredential` + `TestNewProvider_RequestsTheConfiguredEndpoint`; staged-mutation bite-proof verified |
| R-OR-02 | **PASS** | 4 attribution tests + 6 RoundTripper tests + 2 headers-unawareness tests; raw-bytes byte-level check verified |
| R-OR-03 | **PASS** | `TestNewProvider_DefaultModelOnWireBodyIsOpenaiGpt4o` + `TestNewProvider_ConfigModelOverridesDefaultOnWireBody` + `TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment` |
| R-OR-04 | **PASS** | `TestNewProvider_StreamOptionsIncludeUsageIsTrue` (stub-transport probe of rendered body) |
| R-OR-05 | **PASS-PARTIAL** | PR #1 charter pin (`TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment`); capability-record assertion arrives in PR #2 |
| R-OR-06 | **N/A for PR #1** | Arrives in PR #2 |
| R-OR-07 | **N/A for PR #1** | Arrives in PR #3 |
| R-OR-08 | **PASS-PARTIAL** | PR #1 redaction propagation tests (3 tests in `credential_redaction_test.go`); sentinel sweep arrives in PR #3 |
| R-OR-09 | **PASS** | `TestOpenRouterAdapter_ZeroRequiresGuard` + `TestOpenRouterAdapter_AllowedNonStdlibPrefixesHoldsExactlyOneEntry`; existing AI-00.3 guards unmodified |
| R-OR-10 | **PASS** | `TestOpenRouterAdapter_NegativeShallsFenceFails` (4 sub-checks) + `TestOpenRouterAdapter_NegativeShallsFenceFails_OnStagedMutation` (4 staged mutations) |

---

## Findings

### CRITICAL

None.

### WARNING

1. **PR #1 size exceeds the 800-line authored-risk budget by ~3× (2,408 insertions vs. 800 budget).** Tasks.md forecast "Per-PR corrected mid: PR #1 ~700-800 (commit split at 1.5a/1.5b; fallback = defer `charter_test.go` to follow-up)". Actual corrected: 2,408 insertions across 12 files (466 production + 1,942 tests). The 1.5a/1.5b split was applied; `charter_test.go` (1.7, 236 lines) was NOT deferred. The fallback was not exercised. Per `sdd-phase-common.md §E`, this is a per-PR authored-risk overrun — reviewable in 60-minute slices per the work-unit-commits skill, but does exceed the orchestrator-locked 800-line budget. Recommend: orchestrator re-confirms the budget exception or splits `charter_test.go` (236 lines) + `zero_requires_test.go` (314 lines) + `headers_unawareness_test.go` (180 lines) into follow-up commits in a later PR.

2. **Spec R-OR-01 sub-scenario 2 ("Construction rejects a wrong endpoint") is unverifiable as written.** Design D1/D3 deliberately omit an `Endpoint` field on `Config` (the endpoint is hardcoded to `openrouterBaseURL`), so a "wrong endpoint" cannot be constructed. The closest implementation equivalent is `TestNewProvider_RejectsEmptyCredential` (1.1), which exercises an equivalent typed-failure at construction. The spec scenario's intent ("fail-fast at construction; no outbound request") is satisfied, but the scenario text describes an input shape that does not exist. Recommend: spec amendment to retitle the sub-scenario "Construction rejects invalid configuration" and describe the empty-credential case (this is a spec-quality issue, not an implementation defect).

3. **Credential redaction (R-OR-08 PR #1 portion) lacks a separate staged-mutation bite-proof.** The three redaction tests (`TestConfig_CredentialFieldDoesNotBreakRedactionPosture`, `TestNewProvider_ProviderValueDoesNotLeakCredentialViaDefaultFormatting`, `TestNewProvider_ProviderValueThroughStreamDoesNotLeakCredentialInEventError`) are functional bite-proofs — they would FAIL if production code reverted to a credential-leaking state — but they do not include a `Test...FailsOnStagedMutation` companion as the other mechanical guards do. This is consistent with the apply-progress note in Engram #2583: the original RED-found defect (default `%+v`/`%#v` on the embedded `*openaicompat.Client` exposing the unexported `token` field) was caught by these tests, not by a staged mutation. TDD posture is therefore justified by the RED-found-defect trail, but a follow-up staged mutation test would close the same loop the other guards do.

### SUGGESTION

1. **Consider tightening the spec scenario text for R-OR-01.s2** (see WARNING #2) — the scenario as written cannot be reproduced by a future test author.

2. **Consider adding an explicit assertion that `redactedProvider` satisfies `ai.ModelProvider` at compile time** (e.g., `var _ ai.ModelProvider = redactedProvider{}` in `wrapper.go`). The current design relies on the embedded `*openaicompat.Client` satisfying the interface; a static assertion would make the contract explicit at the call site. Not blocking — the existing tests prove the interface is satisfied at runtime.

3. **`TestNewProvider_ProviderValueThroughStreamDoesNotLeakCredentialInEventError`** iterates `for ev := range ch` and uses a `t.Logf` inside the loop that runs only if `ev.Kind() == ai.EventKindError`. The `t.Logf` is not a guard assertion (it does not fail the test); it is informational. Acceptable per the strict-TDD posture (real assertions are on the `fmt.Sprintf("%v", ev)` line above), but a SUGGESTION to convert it to a guarded no-op would close a future reviewer's concern about a no-op `t.Logf`.

---

## Result Contract (for orchestrator)

```
status: success
executive_summary:
  - 8/8 spec MUSTs in PR #1 scope are PASS (R-OR-01, R-OR-02, R-OR-03, R-OR-04, R-OR-09, R-OR-10) or PASS-PARTIAL (R-OR-05 charter pin; R-OR-08 redaction propagation). R-OR-06 and R-OR-07 are N/A (PR #2 / PR #3 respectively).
  - Full test suite: 834 PASS / 0 FAIL / 4 packages green; lint 0 issues; go.mod 3 lines / 0 requires; AI-00.3 forward guard PASS unmodified.
  - 9 commits land in the planned 1.1→1.8 order (1.5 split 1.5a/1.5b); each commit's stated scope matches its diff (work-unit-commits posture confirmed).
  - Strict TDD posture verified: 30 tests across 7 files; 4 staged-mutation bite-proofs (ambient_authority, headers_unawareness, charter × 4, default-model pin); RED-found defect trail recorded in Engram #2583.
  - 0 CRITICAL findings. 3 WARNINGs (size budget overrun, spec sub-scenario text drift, missing staged-mutation for redaction) + 3 SUGGESTIONs.
  - Architecture matches design D1/D2/D3: wrapper is a thin factory returning `ai.ModelProvider`; attribution headers wrapper-injected via wrapper-owned `http.RoundTripper`; `openaicompat/request.go` raw bytes contain zero attribution-header literals (verified by byte-level scan); default model `openai/gpt-4o` pinned by build-time test.
artifacts:
  - worktree: /Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/add-openrouter-first-provider/pr1-wrapper
  - branch: feat/openrouter-wrapper
  - 9 commits: 2103cfd (1.1) → bd3fd3f (1.2) → 3ee61b7 (1.3) → de4bf63 (1.4) → 9eaafd2 (1.5a) → 909f95f (1.5b) → 288f2ef (1.6) → 8bd4e3d (1.7) → 02751ca (1.8)
  - 12 new files: openrouter/{doc,endpoint,headers,transport,wrapper}.go + {ambient_authority,charter,credential_redaction,headers_unawareness,transport,wrapper_test,zero_requires_test}.go
  - go.mod: 3 lines, 0 require, 0 indirect
  - test_output_hash: 4c6661c773759251ca38b0b51cbd0f153e596e7c1687e73f79f92babe237d04b
  - build_output_hash: e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855 (empty build output)
  - Engram prior context: #2580 (apply-progress final tidy), #2583 (PR #1 complete + 2 RED-found defects), #2577 (tasks), #2574 (design), #2573 (spec), #2571 (placement decision), #2570 (proposal), #2568 (explore)
  - Captured command tails:
    - make test → 834 PASS / 0 FAIL / exit 0 / sha256(4c6661c773...)
    - make lint → "0 issues." / exit 0
    - make build → no output / exit 0
    - cat go.mod → "module github.com/cachicamas/backend/agent\n\ngo 1.26.3" / 54 bytes
    - go test -race -v -run 'TestLayer1_ModuleHasNoDependencies_ZeroRequires|...' → PASS / PASS
    - git log main..HEAD --oneline → 9 commits listed above
    - git diff main..HEAD --stat → 12 files / +2408 / -0
next_recommended: apply-pr2 (conformance bridge) — proceed to R-OR-05 capability-record assertion, R-OR-06 conformance suite run; PR #1 stands alone and is rollback-safe per its sentinel test (TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage)
risks:
  - PR #1 size (2408 lines) exceeds the locked 800-line authored-risk budget (WARNING #1) — orchestrator should re-confirm size exception or split test-heavy files into follow-ups
  - Spec R-OR-01.s2 scenario text drift (WARNING #2) — spec amendment recommended to retitle as "Construction rejects invalid configuration" before PR #2 lands
  - Missing staged-mutation bite-proof for R-OR-08 redaction (WARNING #3) — optional follow-up before archive; current functional tests are sufficient for the PR #1 scope
skill_resolution: paths-injected — orchestrator provided explicit skill paths: sdd-verify, _shared/strict-tdd (NOT FOUND on disk → strict-tdd-verify.md from this skill directory was loaded instead), chained-pr, work-unit-commits, go-testing, test-driven-development (NOT FOUND on disk), _shared/sdd-phase-common, _shared/persistence-contract, _shared/sdd-status-contract
tdd_posture: strict-tdd confirmed — RED-first tests written before production code (per apply-progress per-commit evidence table in Engram #2580); 4 staged-mutation bite-proofs verified at runtime; RED-found defect trail recorded (Engram #2583: redactedProvider for credential rendering + baseTransport for DefaultTransport avoidance)
verdicts:
  R-OR-01: PASS — TestOpenRouterAdapter_AmbientAuthorityGuardCoversNewSubPackage + TestNewProvider_RejectsEmptyCredential + TestNewProvider_RequestsTheConfiguredEndpoint; staged-mutation bite-proof verified
  R-OR-02: PASS — 4 attribution tests + 6 RoundTripper tests + 2 headers-unawareness tests; raw-bytes byte-level check verified over openaicompat/request.go
  R-OR-03: PASS — TestNewProvider_DefaultModelOnWireBodyIsOpenaiGpt4o + TestNewProvider_ConfigModelOverridesDefaultOnWireBody + TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment
  R-OR-04: PASS — TestNewProvider_StreamOptionsIncludeUsageIsTrue (stub-transport probe of rendered body asserts the literal byte sequence)
  R-OR-05: PASS-PARTIAL — TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment pins openrouterDefaultModel = "openai/gpt-4o"; capability-record assertion arrives in PR #2 (TestOpenRouterAdapter_CapabilityRecordMatchesAI24)
  R-OR-06: N/A for PR #1 — arrives in PR #2 (5 per-capability drivers + reasoning extension test)
  R-OR-07: N/A for PR #1 — arrives in PR #3 (TestOpenRouterAdapter_LiveSmoke + workflow YAML)
  R-OR-08: PASS-PARTIAL — 3 redaction tests in credential_redaction_test.go cover String/GoString/MarshalJSON/default-format verb coverage + event-error rendering; sentinel sweep arrives in PR #3
  R-OR-09: PASS — TestOpenRouterAdapter_ZeroRequiresGuard + TestOpenRouterAdapter_AllowedNonStdlibPrefixesHoldsExactlyOneEntry; existing AI-00.3 guards pass unmodified
  R-OR-10: PASS — TestOpenRouterAdapter_NegativeShallsFenceFails (4 sub-checks) + TestOpenRouterAdapter_NegativeShallsFenceFails_OnStagedMutation (4 staged mutations)
findings:
  CRITICAL: (none)
  WARNING:
    - PR #1 size (2,408 insertions) exceeds the orchestrator-locked 800-line authored-risk budget by ~3×; tasks.md's forecast (~700-800) was optimistic; the 1.5a/1.5b split was applied but charter_test.go was NOT deferred per the planned fallback
    - Spec R-OR-01.s2 scenario text describes "Construction rejects a wrong endpoint" — by design D1/D3 the wrapper has no Endpoint field, so this scenario is impossible to construct; closest implementation equivalent is TestNewProvider_RejectsEmptyCredential (covers fail-fast at construction but with empty credential, not wrong endpoint)
    - R-OR-08 PR #1 redaction tests are functional bite-proofs but lack a separate staged-mutation test (the other mechanical guards all have one); the RED-found-defect trail in Engram #2583 justifies the posture but a follow-up staged mutation test would close the loop
  SUGGESTION:
    - Tighten spec R-OR-01.s2 scenario text before PR #2 lands (retitle as "Construction rejects invalid configuration" or similar)
    - Add `var _ ai.ModelProvider = redactedProvider{}` static assertion in wrapper.go for explicit interface-saturation documentation
    - The informational `t.Logf` inside the `ev.Kind() == ai.EventKindError` branch in TestNewProvider_ProviderValueThroughStreamDoesNotLeakCredentialInEventError could be removed or guarded (no-op risk)
```

---

## Key Learnings

1. The wrapper-private `redactedProvider` is not a "new exported type" violation of design §6 — it is an unexported helper that satisfies `ai.ModelProvider` by embedding `*openaicompat.Client`, with the only purpose being custom `String`/`GoString` to prevent default `%+v`/`%#v` from descending into the unexported credential field. Callers see `ai.ModelProvider`, not the type.
2. `http.Client{Transport: nil}` falls back to `http.DefaultTransport`, which silently reads `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` from the environment — this is the R-APC-009 ambient-authority leak the 1.8 commit closed by extracting `defaultBoundedTransport()` and substituting it uniformly.
3. The headers-unawareness test reads `openaicompat/request.go` as raw bytes and fails on any of the three attribution-header name literals — byte-level check (via `bytesContain`), not structural. A future openaicompat change that adds any of these names to its header surface is caught immediately by `TestOpenRouterAdapter_HeadersUnawarenessInOpenAICompatRequest`.
4. Strict-TDD bite-proofs come in two shapes in this PR: staged-mutation tests that plant a violation in `t.TempDir()` and assert the scan catches it (ambient_authority, headers_unawareness, charter × 4); and functional bite-proofs where the test asserts the production-code-path output and would FAIL on a regression (credential redaction — the original RED-found defect per Engram #2583).
5. The 800-line authored-risk budget is a hard reviewer-load guard: PR #1's 2,408 insertions (466 production + 1,942 tests) land as 9 reviewable work-unit commits but the cumulative size does exceed the budget — orchestrator should re-confirm the budget exception or split test-heavy files (`zero_requires_test.go`, `charter_test.go`) into follow-up commits.
