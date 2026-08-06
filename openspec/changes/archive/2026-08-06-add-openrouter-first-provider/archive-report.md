# Archive Report — `add-openrouter-first-provider`

> **Change ID**: `add-openrouter-first-provider`
> **Capability ID**: `ai-openrouter-first-provider` (new)
> **Status**: **CLOSED**
> **Date**: 2026-08-06
> **Archived to**: `openspec/changes/archive/2026-08-06-add-openrouter-first-provider/`
> **Source-of-truth spec location**: `openspec/specs/ai-openrouter-first-provider/spec.md`
> **Artifact store**: hybrid (file + engram topic_key `sdd/add-openrouter-first-provider/archive-report`)

## Links to prior artifacts

- Explore: `openspec/changes/add-openrouter-first-provider/explore.md` · Engram obs **#2568**
- Proposal: `openspec/changes/add-openrouter-first-provider/proposal.md` · Engram obs **#2570**
- Wrapper-placement decision: Engram obs **#2571**
- Spec (post-amendment): `openspec/changes/add-openrouter-first-provider/specs/ai-openrouter-first-provider/spec.md` · Engram obs **#2573**
- Design: `openspec/changes/add-openrouter-first-provider/design.md` · Engram obs **#2574**
- Tasks: `openspec/changes/add-openrouter-first-provider/tasks.md` · Engram obs **#2577**
- Apply-progress (final, PR #3): `openspec/changes/add-openrouter-first-provider/apply-progress.md` · Engram obs **#2580**
- PR #1 apply result (RED-found defects + wrapper learnings): Engram obs **#2583**
- PR #1 verify report: `openspec/changes/add-openrouter-first-provider/verify-report-pr1.md` · Engram obs **#2586** · verify-result obs **#2587**
- PR #2 apply relaunch / result: Engram obs **#2588** / **#2589**
- PR #2 verify report: `openspec/changes/add-openrouter-first-provider/verify-report-pr2.md` · Engram obs **#2590**
- PR #3 apply result (R-OR-08 closure): Engram obs **#2591**
- PR #3 verify report: `openspec/changes/add-openrouter-first-provider/verify-report-pr3.md` · Engram obs **#2592** · verify-result obs **#2593**
- Archive decision (smoke warnings accepted → archive follows): Engram obs **#2594**
- Archive report (this file): `openspec/changes/add-openrouter-first-provider/archive-report.md` · Engram obs **#2595** (this observation)

---

## Executive summary

The `add-openrouter-first-provider` change is **closed**. OpenRouter concretizes the AI-24 vendor pre-decision (Engram **#2432**) as the first concrete Layer 1 provider, composing (not re-implementing) the shipped `openaicompat` package, the `agenttest` conformance runner, and the `ai-stream-testkit` helpers. Three chained PRs landed under no-merge tracker `tracker/add-openrouter-first-provider` using `feature-branch-chain`: PR #1 (wrapper) received a maintainer-approved `size:exception` at 2,408 insertions (~466 production + 1,942 tests); PR #2 (conformance bridge) carried the same disposition at ~1,451 authored lines (after §E exclusion of fixture bytes); PR #3 (live smoke) landed at 1,142 raw insertions, within reasonable authored count. Cumulative test posture across all 3 PRs is **893 PASS / 0 FAIL / 4 SKIP**. All 10 spec requirements (R-OR-01..R-OR-10) reach PASS or PASS-WITH-DOCUMENTED-SKIPS. AI-00.3 forward guard is green on all 3 PRs — `go.mod` unchanged at 3 lines / 0 require.

The change's only spec amendment landed in PR #2 precondition 2.0 (commit `2e566f2`) and retitled R-OR-01.s2 from "Construction rejects a wrong endpoint" to "Construction rejects invalid configuration" so the scenario text matches what the implementation (`TestNewProvider_RejectsEmptyCredential`) actually proves — addressing verify-report-pr1 WARNING #2. PR #3 surfaced two non-blocking design-coherence warnings that the user accepted as follow-up work, recorded under "Deferred items" below.

---

## Capability contract

| Capability | Action | Domain | Status |
|---|---|---|---|
| `ai-openrouter-first-provider` | **Added** (new) | new sibling sub-package `backend/agent/src/ai/openaicompat/openrouter/` | **Created** at `openspec/specs/ai-openrouter-first-provider/spec.md` |
| `ai-provider-client` (AI-25) | None | shipped | Composed (read-only) |
| `ai-provider-conformance-suite` (AI-23/38) | None | shipped | Composed (read-only) |
| `ai-stream-testkit` (AI-22) | None | shipped | Composed (read-only) |
| `ai-model-provider` (AI-20) | None | shipped | Composed (read-only) |

**Totals**: 1 new capability, 0 modified, 0 removed, 0 renamed.

**Per spec**: this change declares zero MODIFIED/REMOVED/RENAMED requirements (per proposal §5 PR #1 + spec §Traceability). The only spec change is the AMENDMENT to R-OR-01.s2 wording (PR #2 precondition 2.0, commit `2e566f2`), no behavior change.

---

## Branch state

### Final commits per branch

| Branch | Base | Final hash | Worktree path |
|---|---|---|---|
| `feat/openrouter-wrapper` | `main` (tracker not on disk locally) | `02751cac54b73307198fea57d4cf85decdbabb41` | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/add-openrouter-first-provider/pr1-wrapper` |
| `feat/openrouter-conformance-bridge` | `feat/openrouter-wrapper` (`02751ca`) | `e3a08961fd58cd373536826629293ced07846263` | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/add-openrouter-first-provider/pr2-conformance-bridge` |
| `feat/openrouter-live-smoke` | `feat/openrouter-conformance-bridge` (`e3a0896`) | `d021863a7b02f65fdd68f239d3c1dc3a5004a681` | `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/add-openrouter-first-provider/pr3-live-smoke` |

### Branch graph

```
main
 └─ tracker/add-openrouter-first-provider   [draft, no-merge — not present locally]
       ↑ PR #1 base
       └─ feat/openrouter-wrapper            (02751ca)  ← PR #1
            ↑ PR #2 base
            └─ feat/openrouter-conformance-bridge  (e3a0896)  ← PR #2
                 ↑ PR #3 base
                 └─ feat/openrouter-live-smoke      (d021863)  ← PR #3
```

All three branches are committed locally in worktrees. No push, no PR creation, no merge — PR mechanics are owned by the orchestrator/maintainer at GitHub PR time via the `branch-pr` skill.

---

## PR chain summary

### PR #1 — wrapper (`feat/openrouter-wrapper`, final hash `02751ca`)

- **Scope**: thin factory + wrapper-owned `http.RoundTripper` injecting three OpenRouter-only attribution headers (`HTTP-Referer`, `X-OpenRouter-Title`/`X-Title`, `X-OpenRouter-Categories`). `Config{Credential, HTTPClient, HTTPReferer, XTitle, XCategories, Model}`. Default model `openai/gpt-4o`. `stream_options.include_usage` always set. AI-00.3 forward guard regression test. Charter fence (R-OR-10).
- **Commits**: 8 — `1.1` package skeleton → `1.2` attribution headers via RoundTripper → `1.3` default model + `stream_options.include_usage` → `1.4` credential redaction → `1.5a` ambient-authority guard → `1.5b` headers-unawareness test → `1.6` AI-00.3 forward guard → `1.7` charter fence → `1.8` final tidy + charter pin (`TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment`).
- **Verification verdict**: **PASS WITH EXCEPTION** — 0 CRITICAL, 3 WARNING (size exception, R-OR-01.s2 wording drift, R-OR-08 deferred staged-mutation), 3 SUGGESTION.
- **Size**: **2,408 insertions** (466 production + 1,942 tests). ~3× the 800-line authored budget. **Maintainer-approved `size:exception`** on test-density justification (30 tests across 7 test files; mechanical-guard bite-proofs for ambient authority, headers unawareness, charter, default-model pin).
- **Tests contributed**: 30 PASS / 0 FAIL / 0 SKIP, across 7 test files in `backend/agent/src/ai/openaicompat/openrouter/`.
- **RED-found defects surfaced**: 2 (per Engram **#2583**) — `*openaicompat.Client` lacking `String()`/`GoString()` methods led to default `%+v`/`%#v` exposing the unexported `token` field; mitigated by wrapper-private `redactedProvider` with redacting `String`/`GoString`. `http.Client{Transport: nil}` falling back to `http.DefaultTransport` (silent `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` reads); mitigated by extracting `defaultBoundedTransport()`. Both caught by RED-first tests, confirming strict TDD's value over design review alone.
- **AI-00.3 guard**: PASS / PASS (existing `TestLayer1_ModuleHasNoDependencies_ZeroRequires` + `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`); `go.mod` unchanged at 3 lines / 0 require.

### PR #2 — conformance bridge (`feat/openrouter-conformance-bridge`, final hash `e3a0896`)

- **Scope**: `openrouter/conformance/` sibling sub-package. `bridge_test.go` mirrors `openaicompat/bridge_test.go`'s factory with declared-absent tri-state (`Reasoning`, `TokenCounting`, `CacheBoundary` all `*bool=false`). Recorded SSE fixtures (5: text, tool-call, with-usage, attribution-headers, reasoning-extensions). `reasoning_details` + `reasoning` renamed-field drop test. Capability-record assertion (`absent × 3`). 5 per-capability drivers via `agenttest.RunConformanceFor`.
- **Commits**: 6 — precondition `2.0` (commit `2e566f2`) spec amendment + `2.1` bridge scaffold → `2.2` fixtures → `2.3` reasoning-extension drop → `2.4` capability-record assertion → `2.5` final tidy (5 per-cap drivers + lint cleanup).
- **Verification verdict**: **PASS WITH EXCEPTION** — 0 CRITICAL, 2 WARNING (size extension + R-OR-08 carryover), 0 SUGGESTION.
- **Size**: **1,711 raw / ~1,451 authored** (after §E exclusion of ~260 fixture bytes per `sdd-phase-common.md`). ~1.8× the 800-line authored budget. Same disposition as PR #1's `size:exception` — implicit extension to PR #2.
- **Tests contributed**: 11 PASS / 3 SKIP (CompletionMetadata, Cancellation, Terminal — documented out-of-scope per design D3) / 0 FAIL. Wrapper regression: PR #1's 30 tests remain green.
- **Spec amendment (precondition 2.0)**: R-OR-01.s2 retitled from "Construction rejects a wrong endpoint" → "Construction rejects invalid configuration"; GIVEN/WHEN/THEN updated to describe the empty-credential shape (matches `TestNewProvider_RejectsEmptyCredential`). Implementation unchanged. Resolves verify-report-pr1 WARNING #2.
- **Bounded duplication**: `bridgeAttributionRoundTripper` (~30 LOC) intentionally mirrors `openrouter/transport.go`'s unexported `attributionRoundTripper` (no `modelOverride` field, no body mutation) per "do not modify PR #1" hard rule. Tracked at 3 points: `bridge_test.go` header, `with_attribution_headers.go` header, `TestFixtures_AttributionHeaderNamesConsistent` mechanical guard.
- **AI-00.3 guard**: PASS / PASS; `go.mod` unchanged.

### PR #3 — live smoke (`feat/openrouter-live-smoke`, final hash `d021863`)

- **Scope**: `openrouter/smoke/` sibling sub-package. `t.Skip`-gated `TestOpenRouterAdapter_LiveSmoke` (opt-in via `RUN_LIVE_OPENROUTER_SMOKE=1` and `OPENROUTER_API_KEY` env). Sentinel-sweep helper (`Scan(t, captured)`) with deny-list built at runtime (env-var name, secret prefix 4 chars, planted prompt bytes). Workflow `.github/workflows/agent-openrouter-smoke.yml` (`workflow_dispatch` only, secret reference, concurrency group, default false). Workflow structural guards as Go tests reading YAML raw bytes.
- **Commits**: 4 — `3.1` smoke skeleton with `t.Skip` gate → `3.2` sentinel-sweep helper + deferred R-OR-08 staged-mutation bite-proof (3 bite-proofs: env-var name, secret prefix, planted prompt) → `3.3` workflow YAML → `3.4` final tidy + workflow guards (`TestWorkflowFile_IsDispatchOnly` + `TestWorkflowFile_HasRunSmokeInputDefaultFalse`).
- **Verification verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 2 WARNING, 1 SUGGESTION (user-accepted; see Deferred items).
- **Size**: **1,142 raw insertions** (~147 production in `sentinel_sweep.go` + 361 tests + 634 workflow + workflow guards). Within budget by reasonable authored count.
- **Tests contributed**: 18 PR #3 PASS / 1 SKIP (live smoke under no env vars) / 0 FAIL.
- **Deferred R-OR-08 staged-mutation bite-proof CLOSED** (carried from verify-report-pr1 WARNING #3 and verify-report-pr2 WARNING #2). 3 staged-mutation bite-proofs cover all three deny-list entries — verified by staging no-op scan mutations and observing tests FAIL.
- **AI-00.3 guard**: PASS / PASS; `go.mod` unchanged.

---

## Test posture

### Cumulative counts across all 3 PRs

| Source | PASS | FAIL | SKIP |
|---|---:|---:|---:|
| PR #0 (prior baseline) | 834 | 0 | 0 |
| PR #1 (wrapper) | 30 | 0 | 0 |
| PR #2 (conformance bridge) | 11 | 0 | 3 |
| PR #3 (live smoke) | 18 | 0 | 1 |
| **Cumulative** | **893** | **0** | **4** |

### R-OR-06 documented SKIPs (CompletionMetadata, Cancellation, Terminal)

The 3 R-OR-06 SKIPs (`TestOpenRouterAdapter_CompletionMetadata`, `TestOpenRouterAdapter_Cancellation`, `TestOpenRouterAdapter_Terminal`) match shipped `openaicompat/bridge_test.go`'s text+tool-calls-only scope (design D3):

- **CompletionMetadata** — `finish_reason` / refusal / `pause_turn` / unknown are unreachable on the openaicompat wire (delivered via usage-only streaming model).
- **Cancellation** — `openaicompat` produces an error terminal on context cancel, so the per-cap cancellation case is a no-op against the openaicompat-shaped surface.
- **Terminal** — `ErrorEvent` rendering sits outside `openaicompat`'s declared contract.

Each SKIP carries an attributable `t.Skip` message citing the dialect-specific limitation. PR #2 verify report (Engram **#2590**) records the same accepted disposition.

### Live smoke SKIP (PR #3, 4th SKIP)

`TestOpenRouterAdapter_LiveSmoke` skips before provider construction at `smoke_test.go:305-309` when `OPENROUTER_API_KEY` is absent from the test process. This is intentional — the live path is opt-in via `workflow_dispatch`, not exercised by `make test` in normal CI.

### Strict TDD posture

Confirmed across all 3 PRs:

- **PR #1**: 6/6 strict-TDD checks pass. RED-first tests with staged-mutation bite-proofs verified for ambient authority, headers unawareness, charter fence, default-model pin. RED-found defect trail recorded in Engram **#2583**.
- **PR #2**: 6/6 strict-TDD checks pass. 5 RED-GREEN cycles + 1 spec-amendment REFACTOR + 1 final-tidy REFACTOR. 0 new RED-found defects this PR (PR #1's trail covers the redaction pattern).
- **PR #3**: 6/6 strict-TDD checks pass. 3 RED-GREEN cycles + 1 CI-configuration carve-out (YAML validated by Python parser + Go structural guards). 11 sentinel tests + 3 staged-mutation bite-proofs close the deferred R-OR-08 staged-mutation gap.

**Strict TDD posture**: **strict-tdd confirmed** across all 3 PRs. No PR relied on implementation-detail-only assertions, no ghost loops, no smoke-only tests, no orphan empty assertions.

---

## Decisions log

### Locked preflight (orchestrator-cached)

- `execution_mode = auto`
- `artifact_store.mode = both` (engram + filesystem)
- `delivery_strategy = auto-chain`
- `chain_strategy = feature-branch-chain`
- `review_budget_lines = 800`

### Proposal-level decisions (C1–C4 + Q1–Q6)

| ID | Decision | Basis |
|---|---|---|
| **C1** | Default model = `openai/gpt-4o` (non-reasoning, paid) | Preserves AI-29's struck verdict; explore §2.7 |
| **C2** | Conformance + smoke use paid model (no `:free` suffix) | `:free` rate-limit flakiness; explore §2.10 |
| **C3** | Attribution headers wrapper-injected, NOT openaicompat-injected | Keeps openaicompat's header surface narrow; Engram **#2571** |
| **C4** | `stream_options.include_usage` stays set | Cross-vendor wire-body uniformity; explore §2.5 |
| **Q1** | Conformance sweep = `openai/gpt-4o` only | Sweeping Anthropic passthrough reopens AI-29 under trigger #1 |
| **Q2** | Layer 3 reads env, passes opaque bearer into `Config.Credential` | AI-25.2 invariant; wrapper reads nothing (Engram **#2432** §3) |
| **Q3** | Wrapper placement = sibling sub-package `openaicompat/openrouter/` | AI-25.2 call-site scan scope stays clean; Engram **#2571** |
| **Q4** | Conformance bridge ships in PR #2 of this change | AI-38 first-concrete-adapter charter; doc 0002 lines 2241–2277 |
| **Q5** | Live smoke opt-in via `workflow_dispatch` + repo secret `OPENROUTER_API_KEY` | AI-39.1 charter; doc 0002 lines 2279–2299 |
| **Q6** | Three chained PRs under no-merge tracker | 800-line PR budget per `sdd-phase-common.md §E`; natural work-unit split |

### Per-PR decisions (locked during apply / verify)

- **PR #1**: Maintainer-approved `size:exception` at 2,408 insertions (test-density justification). R-OR-01.s2 spec amendment deferred to PR #2 precondition 2.0 (commit `2e566f2`) — resolved at PR #2.
- **PR #2**: Implicit extension of PR #1's `size:exception` at ~1,451 authored lines. Bounded duplication of `bridgeAttributionRoundTripper` (~30 LOC) accepted per "do not modify PR #1" hard rule. 5 `//nolint:revive` directives added to conformance package files with task-plan line citations as reviewer breadcrumbs.
- **PR #3**: User-accepted 2 design-coherence warnings as follow-up work (see Deferred items). `t.Skip` gate (NOT build tag) keeps skip path observable in `make test`. Sentinel-sweep needle bytes assembled at runtime to prevent the helper from matching its own source patterns.

### Conventions locked at preflight

- Conventional Commits only (no `Co-Authored-By`, no AI attribution) — confirmed across all 18 commits.
- Strict TDD (RED → GREEN → REFACTOR) — confirmed across all 3 PRs.
- Backend test runner = `cd backend/agent && make test` (Engram **#2055**).
- Wrapper path = `backend/agent/src/ai/openaicompat/openrouter/` (per design D1, Engram **#2571**) — prompt's `src/ai/openrouter/` shorthand read as package directory, not a placement re-decision.
- Workflow path = `.github/workflows/agent-openrouter-smoke.yml` (per design D5) — prompt's `openrouter-live-smoke.yml` shorthand read as such.

---

## Deferred items (follow-up work, NOT in this archive)

These items were raised by the verify phases but are **out of scope** for this archive. They are recorded here so a future `live-smoke-hardening` change can pick them up without re-deriving the context.

### Follow-up: `live-smoke-hardening` change (planned)

#### WARNING #1 — Live event-shape assertion is weaker than design §5 / task 3.1

- **Location**: `smoke_test.go:334-365`
- **Issue**: Currently accepts any one of `ResponseStart`, `TextDelta`, `ToolCallStart`, `ToolCallDelta`, or `Completion` and never asserts exactly one terminal. Design `design.md:88-95` and task 3.1 `tasks.md:187` require `ResponseStart` + at least one `TextDelta` + exactly one `Completion` or terminal error.
- **Disposition**: User accepted as-is at archive time (Engram **#2594**); follow-up change to tighten the live event-shape assertion.
- **Why non-blocking**: The formal R-OR-07 scenarios still pass because they cover skip/no-network and workflow dispatch; only the live-path design obligation is unmet (no credential was intentionally supplied in verify).
- **Source**: Engram **#2593** (verify-pr3 result), verify-report-pr3 WARNING #1.

#### WARNING #2 — Sentinel sweep is not wired into the live test's log/error capture path

- **Location**: `smoke_test.go:305-367`
- **Issue**: `smoke.Scan` is never called from the live test's actual `t.Logf`/error capture path. Design `design.md:194` describes scanning captured output at the end of the live test; current implementation only proves the helper via 11 standalone tests + 3 staged-mutation bite-proofs.
- **Disposition**: User accepted as-is at archive time (Engram **#2594**); follow-up change to wire `smoke.Scan` into the live test's log/error capture path.
- **Why non-blocking**: R-OR-08's helper and bite-proof acceptance gates pass; the integration gap is defense-in-depth, not a current credential leak.
- **Source**: Engram **#2593**, verify-report-pr3 WARNING #2.

#### SUGGESTION — Workflow guard missing-file branches skip instead of failing hard

- **Location**: `workflow_guards_test.go:91-94` and `122-124`
- **Issue**: `t.Skipf` when the workflow file is unreadable rather than a hard failure.
- **Disposition**: Cosmetic; current checkout has the file and both guards pass. Picked up in follow-up if the workflow ever goes missing.
- **Source**: verify-report-pr3 SUGGESTION.

### Follow-up scope note

These three items are the **only** deferred items. Everything else required by the spec is verified green. The follow-up change `live-smoke-hardening` should be small (estimated <200 LOC authored, within a single 400-line budget PR).

---

## AI-00.3 forward guard state

| PR | `go.mod` lines | `require` count | `TestLayer1_ModuleHasNoDependencies_ZeroRequires` | `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` |
|---|---:|---:|:---:|:---:|
| PR #1 | 3 | 0 | PASS | PASS |
| PR #2 | 3 | 0 | PASS | PASS |
| PR #3 | 3 | 0 | PASS | PASS |

**`go.mod` content unchanged** at 3 lines (module declaration, blank, `go 1.26.3`) with zero `require` directives across all 3 PRs. **No ADR gate fires** (no new top-level Go dependency was added — per `openspec/AGENTS.md` rule 5). The `allowedNonStdlibPrefixes` allowlist in `backend/agent/src/ai/import_boundary_test.go` remains at exactly one entry (the module path itself).

The wrapper composes only stdlib (`net/http`, `bytes`, `errors`, `fmt`, `os`, `strings`, `time` etc.), `package ai`, and `package openaicompat` (in-module). The conformance sub-package additionally imports `package agenttest` (in-module test-only). The smoke sub-package imports stdlib + `package openrouter` (sibling sub-package). No new transitive closure.

---

## No-PR-creation notice

**This archive phase does NOT push, does NOT open PRs, and does NOT merge.** All three feature branches (`feat/openrouter-wrapper`, `feat/openrouter-conformance-bridge`, `feat/openrouter-live-smoke`) are committed locally in their respective worktrees; nothing has been pushed to the remote.

PR mechanics (push, PR creation, merge to tracker, merge of tracker to `main`) are owned by the orchestrator (or maintainer) at GitHub PR time via the `branch-pr` skill. The maintainer-approved `size:exception` for PR #1 (and implicit extension to PR #2) is recorded here as part of the decisions log; the `branch-pr` invocation should cite this archive-report and the relevant verify-report when requesting re-review.

Tracker branch `tracker/add-openrouter-first-provider` does **not** exist locally (per Engram **#2583**: "Branch base is `main`, not the planned `tracker/add-openrouter-first-provider` (tracker does not exist locally). Orchestrator owns tracker creation."). This is a known project-level state — the orchestrator creates the tracker before PR #1's `branch-pr` invocation.

---

## Risks & known gaps (material)

### 1. Live smoke is weaker than design §5 / task 3.1

The PR #3 verify WARNINGs #1 and #2 together mean the live path is not yet fully design-conformant — the test skips without credentials during normal `make test` runs, so the live path was not exercised during this verify phase. The formal R-OR-07 scenarios still pass (skip path, workflow dispatch), and R-OR-08's helper and bite-proofs all pass standalone. The gap is **integration-shape**, not **implementation-correctness**.

**Mitigation**: Follow-up `live-smoke-hardening` change (see Deferred items).

### 2. R-OR-06 documented SKIPs

The 3 R-OR-06 SKIPs (`CompletionMetadata`, `Cancellation`, `Terminal`) are documented out-of-scope per design D3 — they match shipped `openaicompat/bridge_test.go`'s text+tool-calls-only scope. They are NOT regressions and NOT silent skips: each carries an attributable `t.Skip` message citing the dialect-specific limitation.

**Mitigation**: None needed. The openaicompat surface itself has the same disposition; this is not a wrapper-level gap.

### 3. Bounded duplication of `bridgeAttributionRoundTripper`

Per PR #2's "do not modify PR #1" hard rule, `bridgeAttributionRoundTripper` (~30 LOC) intentionally mirrors `openrouter/transport.go`'s unexported `attributionRoundTripper` without `modelOverride` field or body mutation. This is documented at 3 future-change tracking points: `bridge_test.go` header, `with_attribution_headers.go` header, and `TestFixtures_AttributionHeaderNamesConsistent` mechanical guard.

**Mitigation**: If the attribution-header set ever changes, the three tracking points will surface the duplication. Acceptable bounded duplication; not a code-smell that justifies touching PR #1's sealed surface.

### 4. Tracker branch not present locally

`tracker/add-openrouter-first-provider` does not exist locally — all three PR branches were based directly on `main`. The orchestrator (or maintainer) creates the tracker before invoking `branch-pr` on PR #1. This is recorded as a known project-level state and is not a defect of this change.

**Mitigation**: Tracker creation is part of the PR-mechanics workflow, not the SDD cycle. The `branch-pr` skill handles it.

### 5. PR size budgets exceeded (PR #1, PR #2)

PR #1 (2,408 insertions) and PR #2 (~1,451 authored) both exceed the 800-line authored budget. PR #1 received a maintainer-approved `size:exception`; PR #2 carries the same disposition implicitly. The budget was sized for reviewability, not as a hard limit, and the test-density justification (mechanical-guard bite-proofs across all four negative-SHALL areas) is on the record.

**Mitigation**: The follow-up `live-smoke-hardening` change should stay under 400 lines authored to prove the budget works for normal-sized follow-ups.

---

## References

### Prior SDD artifacts (this change)

| Phase | Artifact | Path | Engram obs |
|---|---|---|---|
| Explore | explore.md | `openspec/changes/add-openrouter-first-provider/explore.md` | **#2568** |
| Propose | proposal.md | `openspec/changes/add-openrouter-first-provider/proposal.md` | **#2570** |
| Decision | wrapper placement | (decision observation) | **#2571** |
| Spec | spec.md (post-amendment) | `openspec/changes/add-openrouter-first-provider/specs/ai-openrouter-first-provider/spec.md` · source-of-truth: `openspec/specs/ai-openrouter-first-provider/spec.md` | **#2573** |
| Design | design.md | `openspec/changes/add-openrouter-first-provider/design.md` | **#2574** |
| Tasks | tasks.md | `openspec/changes/add-openrouter-first-provider/tasks.md` | **#2577** |
| Apply | apply-progress.md (final, PR #3) | `openspec/changes/add-openrouter-first-provider/apply-progress.md` | **#2580** |
| Apply | PR #1 wrapper learnings + 2 RED-found defects | (decision observation) | **#2583** |
| Verify | verify-report-pr1.md | `openspec/changes/add-openrouter-first-provider/verify-report-pr1.md` | **#2586** |
| Verify | PR #1 verify result | (decision observation) | **#2587** |
| Apply | PR #2 apply relaunch | (decision observation) | **#2588** |
| Apply | PR #2 apply result | (decision observation) | **#2589** |
| Verify | verify-report-pr2.md | `openspec/changes/add-openrouter-first-provider/verify-report-pr2.md` | **#2590** |
| Apply | PR #3 apply result (R-OR-08 closure) | (decision observation) | **#2591** |
| Verify | verify-report-pr3.md | `openspec/changes/add-openrouter-first-provider/verify-report-pr3.md` | **#2592** |
| Verify | PR #3 verify result (2 warnings accepted) | (decision observation) | **#2593** |
| Decision | smoke warnings accepted → archive follows | (decision observation) | **#2594** |
| Archive | archive-report.md (this file) | `openspec/changes/archive/2026-08-06-add-openrouter-first-provider/archive-report.md` | **#2595** (this observation) |

### Milestone charters (doc 0002 — Layer 1 task graph)

- **AI-25** (Provider client construction): `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 1447–1491
- **AI-38** (Adapter conformance suite): doc 0002 lines 2241–2277
- **AI-39** (Live smoke): doc 0002 lines 2279–2299

### Composed shipped specs (read-only, NOT modified)

- [`openspec/specs/ai-provider-client/spec.md`](../../specs/ai-provider-client/spec.md) (AI-25)
- [`openspec/specs/ai-provider-conformance-suite/spec.md`](../../specs/ai-provider-conformance-suite/spec.md) (AI-23/AI-38)
- [`openspec/specs/ai-stream-testkit/spec.md`](../../specs/ai-stream-testkit/spec.md) (AI-22)
- [`openspec/specs/ai-model-provider/spec.md`](../../specs/ai-model-provider/spec.md) (AI-20)

### Shipped code (read, NOT modified)

- `backend/agent/src/ai/openaicompat/{client,request,credential,bridge_test,ambient_authority_test,reasoning_absence_test,credential_scan_test}.go`
- `backend/agent/src/ai/import_boundary_test.go`
- `backend/agent/src/ai/provider.go`
- `backend/agent/src/agenttest/{conformance_suite,conformance_capabilities,conformance_record,conformance_redaction,provider_signature_guard_test}.go`
- `backend/agent/go.mod` (3 lines, 0 require — unchanged)
- `backend/agent/Makefile`

### Decision lineage

- **AI-24 pre-decision** (vendor/transport, OpenAI-compatible Chat Completions + raw `net/http` + zero `go.mod` requires): Engram observation **#2432**
- **AI-29 struck verdict** (2026-08-04, reasoning stream absent): `openspec/changes/archive/2026-08-04-cachicamas-ai-provider-reasoning-stream/decision.md` §5 row 4, §7, §9 (triggers)
- **ADR 0004** (3-layer agentic architecture): Engram observation **#1997** + `docs/adr/0004-adopt-tau-3-layer-agentic-architecture.md`
- **ADR 0005** (agent module dependency rule v2): `docs/adr/0005-promote-agent-stack-to-own-module.md`
- **Backend test runner** (convention): Engram observation **#2055**

### Project rules

- `openspec/AGENTS.md` (TDD on, `make test`, conventional commits, no AI attribution, hexagonal layout exception for `backend/agent`)
- `openspec/config.yaml` `rules.specs` — Given/When/Then, RFC 2119 keywords, independently verifiable scenarios
- `openspec/config.yaml` `rules.apply.tdd = true`

### Skills applied during this change

- `sdd-archive` (this phase)
- `sdd-verify` (per-PR verify)
- `sdd-apply` (per-PR apply)
- `sdd-design`, `sdd-spec`, `sdd-tasks`, `sdd-propose`, `sdd-explore` (planning phases)
- `chained-pr` (feature-branch-chain topology, ≤60 min PR rule, polluted-diff rule)
- `work-unit-commits` (every commit build-green + test-green + narrow intent + rollback-friendly)
- `branch-pr` (planned, post-archive, at GitHub PR time — NOT invoked by archive phase)
- `test-driven-development`, `go-testing` (per `openspec/AGENTS.md` rules table)

---

## Result Contract

```yaml
status: success
change: add-openrouter-first-provider
capability: ai-openrouter-first-provider (new)
closed_at: 2026-08-06
archived_to: openspec/changes/archive/2026-08-06-add-openrouter-first-provider/
source_of_truth: openspec/specs/ai-openrouter-first-provider/spec.md
spec_sync:
  new_capabilities: 1 (ai-openrouter-first-provider)
  modified_capabilities: 0
  removed_capabilities: 0
  renamed_capabilities: 0
  spec_amendments_in_landed_prs: 1 (R-OR-01.s2 retitle, commit 2e566f2, no behavior change)
cumulative_tests:
  pass: 893
  fail: 0
  skip: 4  # 3 R-OR-06 documented out-of-scope + 1 live smoke under no env vars
critical_findings: 0
warnings_open_at_archive: 0  # 2 PR #3 verify warnings accepted as follow-up (not blocking archive)
ai_00_3_forward_guard: green
go_mod_state: unchanged (3 lines, 0 require)
pr_chain: 3 chained PRs (wrapper → conformance-bridge → live-smoke) under no-merge tracker
branches_committed_locally: 3  # feat/openrouter-wrapper, feat/openrouter-conformance-bridge, feat/openrouter-live-smoke
branches_pushed: 0  # archive does NOT push
prs_created: 0  # archive does NOT open PRs
next_recommended: none — change closed; follow-up change `live-smoke-hardening` is a separate cycle
risks:
  - live-smoke design-conformance gap (PR #3 verify WARNINGs #1 and #2) — deferred to follow-up
  - R-OR-06 documented SKIPs (CompletionMetadata, Cancellation, Terminal) — match shipped openaicompat scope
  - bounded duplication of bridgeAttributionRoundTripper (~30 LOC) per PR #2's "do not modify PR #1" rule
  - tracker branch not present locally — orchestrator/maintainer creates before branch-pr invocation
skill_resolution: paths-injected — orchestrator provided explicit skill paths
```

---

## Key Learnings

1. **Spec amendment during apply is acceptable when scoped to wording.** PR #2 precondition 2.0 (commit `2e566f2`) retitled R-OR-01.s2 to match what the implementation actually proves — zero behavior change, just scenario text. This is a clean pattern for resolving verify-report warnings without bloating a follow-up spec change.

2. **Bounded duplication with mechanical guards is preferable to back-porting across a feature-branch-chain.** `bridgeAttributionRoundTripper` (~30 LOC) is a clean intentional divergence from PR #1's `attributionRoundTripper` — the three tracking points (file headers + `_Consistent` mechanical guard) keep the duplication visible to future reviewers.

3. **`t.Skip` over build tags keeps the skip path observable.** PR #3's live smoke is `t.Skip`-gated rather than `//go:build live_smoke`-gated so `make test` exercises the skip path on every CI run — R-OR-07 scenario 1 is satisfied by the normal test run, not by an opt-in path.

4. **Runtime-built sentinel needles prevent the helper from matching its own source patterns.** PR #3's sentinel_sweep.go assembles env-var name bytes at runtime so the helper can't accidentally trip over the assembled literal in its own source — S-ART-014-style defense.

5. **Staged-mutation bite-proofs are the load-bearing strict-TDD pattern for mechanical guards.** PR #1 contributed 4 bite-proofs (ambient authority, headers unawareness, charter × 4 sub-checks, default-model pin); PR #3 closed the deferred R-OR-08 bite-proof with 3 more (env-var name, secret prefix, planted prompt). Each bite-proof is verified by staging a no-op regression mutation and observing the test FAIL with a clear, actionable error message — then restoring and observing PASS.

6. **Strict TDD catches design-level latent bugs.** PR #1's two RED-found defects (`*openaicompat.Client` lacking `String()`/`GoString()` methods, `http.DefaultTransport` silent proxy-env reads) were caught by RED-first tests written before the GREEN production code — confirming strict TDD's value over design review alone. Both defects are recorded in Engram **#2583** so future PRs can build on the pattern.

---

## Final integration (post-archive)

After the sdd-archive phase wrote this report, the orchestrator opened the PRs and the maintainer merged them in sequence:

| Step | PR | From → To | Merge commit | Merged at |
|---|---|---|---|---|
| 1 | #117 | `feat/openrouter-wrapper` → `tracker/add-openrouter-first-provider` | `a7c1e17` | 2026-08-06T21:24:17Z |
| 2 | #118 | `feat/openrouter-conformance-bridge` → `tracker/add-openrouter-first-provider` | `69b9a34` | 2026-08-06T21:24:52Z |
| 3 | #119 | `feat/openrouter-live-smoke` → `tracker/add-openrouter-first-provider` | `3a049ee` | 2026-08-06T21:30:53Z |
| 4 | #120 | `tracker/add-openrouter-first-provider` → `main` | `e44f59f` | 2026-08-06T21:32:57Z |

**Topology used in practice**: stacked into the tracker (not feature-branch-chain as the design proposed). The maintainer rebased all three PRs onto the tracker rather than stacking them onto intermediate branches. Topology doesn't affect correctness — the final commit graph on `main` is identical to what feature-branch-chain would have produced.

**Pre-PR #119 cleanup (commits 3.5a `1bcdd2f` + 3.5b `5f9f6ea`)**: workflow YAML + workflow guards test removed; spec R-OR-07 amended to drop the workflow-dispatch requirement. Both commits ship with PR #119 to `tracker`.

**Final state of `main`**: `e44f59f` includes all 25 commits (9 wrapper + 6 conformance + 6 smoke/cleanup + 4 merge commits). Local main fast-forwarded to match `origin/main`.

**Change status**: **CLOSED + INTEGRATED**. The `ai-openrouter-first-provider` capability is live on `main` and is the first concrete Layer 1 provider for the cachicamas AI agent. Follow-up `live-smoke-hardening` change is a separate cycle.
