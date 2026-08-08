# Apply Progress: Enforce secret redaction (AI-36) — `cachicamas-ai-redaction`

> Worktree: `cachicamas-worktrees/ai-36-redaction`. Branch `feat/ai-36-redaction` on `origin/main@1103769`.
> Strict TDD. Single PR, `size:exception` pre-granted, budget 1600 — **SEE THE SIZE RISK SECTION BELOW: this exceeded the ~1900 stop-and-report threshold.**

## Status: 43/43 tasks complete, all four verification gates green, but the size budget was exceeded past the stop-and-report line. See "Size / budget outcome" for the decision this hands back.

---

## D-4 and D-5 empirical outcomes (real command output)

### D-4 — non-streaming excerpt credential vector

Command: `go test ./src/ai/openaicompat/... -run TestHostileServer -v` (before the fix, `stream.go` at WU-1's state).

RED output (verbatim):
```
=== RUN   TestHostileServer_EchoesAuthorizationAndBody_CredentialNeverSurfacesInAnyRendering
    a_i-36_1_test.go:148: rendering "unwrap-depth-1 .Error()" leaked the caller's own credential (vector="sentinel credential") via the hostile server's echo (D-4/S-CNF-081)
    a_i-36_1_test.go:148: rendering "unwrap-depth-1 %s" leaked the caller's own credential (vector="sentinel credential") via the hostile server's echo (D-4/S-CNF-081)
    a_i-36_1_test.go:148: rendering "unwrap-depth-1 %v" leaked the caller's own credential (vector="sentinel credential") via the hostile server's echo (D-4/S-CNF-081)
    a_i-36_1_test.go:148: rendering "unwrap-depth-1 %+v" leaked the caller's own credential (vector="sentinel credential") via the hostile server's echo (D-4/S-CNF-081)
    a_i-36_1_test.go:164: named residual (S-CNF-081 item 2, D-4 item 4): prompt-body echo observed in a rendering = true — ...
--- FAIL: TestHostileServer_EchoesAuthorizationAndBody_CredentialNeverSurfacesInAnyRendering (0.00s)
```

**Outcome: credential BITES.** Branch **2.4a** taken: `stream.go` gained `redactCredential`/`credentialRedactedPlaceholder`; `refuseNonStreamContentType` now takes the client's credential and redacts it out of the captured excerpt before it reaches `nonStreamContentType`. `R-AEM-019` promoted to **LANDED** in `specs/ai-provider-error-mapping/spec.md` with this evidence recorded in the file. Re-run after the fix: PASS (`--- PASS: TestHostileServer_EchoesAuthorizationAndBody_CredentialNeverSurfacesInAnyRendering (0.00s)`).

The prompt-body echo **was also observed** (`prompt-body echo observed in a rendering = true`) — recorded as a named residual (S-CNF-081 item 2 / R-AEM-019 item 4), not fixed: the excerpt is the provider's own response, and suppressing its echo of the caller's own content would defeat R-ATS-023's diagnostic purpose.

### D-5 — bare, unwrapped configured client

Command: `go test ./src/ai/openaicompat/... -run TestClient_BareUnwrapped_NeverRendersCredential -v`

RED output (verbatim, both sub-tests):
```
=== NAME  TestClient_BareUnwrapped_NeverRendersCredential/Client_(value,_dereferenced)
    a_i-36_2_test.go:132: rendering via %#v leaked the sentinel: "openaicompat.Client{base:(*url.URL)(0x...), credential:openaicompat.Credential{token:\"sk-ai36-client-level-sentinel-...\"}, httpClient:(*http.Client)(0x...)}"
=== NAME  TestClient_BareUnwrapped_NeverRendersCredential/*Client_(pointer)
    a_i-36_2_test.go:127: rendering via %v leaked the sentinel: "&{0x... {sk-ai36-client-level-sentinel-...} 0x...}"
    a_i-36_2_test.go:127: rendering via %s leaked the sentinel: "&{%!s(*url.URL=&{...}) {sk-ai36-client-level-sentinel-...} %!s(*http.Client=&{...})}"
    a_i-36_2_test.go:127: rendering via %+v leaked the sentinel: "&{base:0x... credential:{token:sk-ai36-client-level-sentinel-...} httpClient:0x...}"
    a_i-36_2_test.go:127: rendering via %#v leaked the sentinel: "&openaicompat.Client{base:(*url.URL)(0x...), credential:openaicompat.Credential{token:\"sk-ai36-client-level-sentinel-...\"}, ...}"
--- FAIL: TestClient_BareUnwrapped_NeverRendersCredential (0.00s)
```

**Outcome: BITES** (all four verbs, both pointer and value forms — `reflect.Value.CanInterface()` is false for a value reached through an unexported field, so `fmt` falls back to raw reflection instead of dispatching to `Credential.String/GoString`). Branch **3.3a** taken: added value-receiver `String()`/`GoString()` on `Client`, fixed label `"<openaicompat client>"`, mirroring `wrapper.go:90-101`. Pre-check (mandatory, AI-41 W-2 process gap): `provider_boundary_test.go:28` and `stream_test.go:259` only assert `MethodByName("Stream")` presence/signature on `&Client{}` — no method-count pin exists — **CLEAR**. Re-run after the fix: PASS.

---

## TDD Cycle Evidence

| Task | Test file | Layer | RED | GREEN | Notes |
|---|---|---|---|---|---|
| 1.1/1.2 | `agenttest/sweep/sweep_test.go` | Unit | ✅ compile failure ("no non-test Go files") | ✅ | 4 tests incl. triangulation |
| 1.3 | `openrouter/smoke/sentinel_sweep.go` | Unit (regression) | N/A refactor | ✅ 8 R-OR-08 tests green | Behavior-preserving |
| 1.4 | `agenttest/conformance_redaction.go` | Unit (regression) | N/A refactor | ✅ existing case green | Behavior-preserving |
| 1.5 | `agenttest/sweep_convergence_test.go` | Unit | N/A pin | ✅ | Labeled non-RED-first |
| 2.1/2.2 | `agenttest/conformance_redaction.go` | Unit | ✅ `undefined: redactionSweepAllCategories`/`redactionSweepDivergenceReport` | ✅ | Compile-failure RED (captured via strip/restore) |
| 2.3/2.4a | `a_i-36_1_test.go` + `stream.go` | Integration (httptest) | ✅ real leak (see D-4 above) | ✅ | Genuine empirical RED |
| 3.1 | `a_i-36_2_test.go` | Unit | N/A — labeled expected-GREEN | ✅ first run | Honored the label |
| 3.2/3.3a | `a_i-36_2_test.go` + `client.go` | Unit | ✅ real leak (see D-5 above) | ✅ | Genuine empirical RED |
| 3.4/3.6 | `retry_metadata_test.go` | Unit | ✅ `undefined: headerCaptureLeak` ×4 | ✅ | Compile-failure RED |
| 3.5 | `retry_metadata.go` (unchanged) | N/A | N/A — confirm-only | ✅ | 0 lines, allowlist already sound |
| 4.1 | `stream_kit_diff_test.go` | Unit | **Labeled RED, empirically GREEN on first run** — see Deviation D-1 below | ✅ | Honored evidence over label |
| 4.2/4.4 | `stream_kit_diff.go` + `stream_kit_diff_test.go` | Unit | ✅ 4 of 10 kind sub-cases genuinely FAILed (`ResponseStart/id`, `ResponseStart/model`, `ToolCallStart/id`, `ToolCallStart/name`) | ✅ | Real gap found, contradicting the "0 prod lines" prediction — see Deviation D-2 |
| 4.3 | `stream_kit_diff_test.go` | Unit | **Labeled RED, empirically GREEN on first run** | ✅ | Mechanism-based control (under-bound sentinel bites by construction) |
| 5.1/5.2/5.4/5.6/5.8/5.10 | `credential_scan_test.go` | Unit | ✅ whole scanner implementation stripped: `undefined: walkCredentialSurface`, `markedFiles`, `allowlistMismatch`, `verifyAllowlistFalsifiability` | ✅ | One combined compile-failure RED for the cohesive rewrite |
| 5.3/5.7/5.9 | `credential_scan_test.go` + 6 marker files | N/A | N/A — GREEN mechanical/impl | ✅ | Same commit as 5.2 |
| 5.5/5.11 | (verification-only) | N/A | N/A | ✅ | 5 pre-AI-36 scenarios re-run green, 1 inverted per design AD-4 |
| 6.1/6.2 | `provider_failure_test.go` | Unit | N/A — labeled expected-GREEN | ✅ first run | Mechanism pin + contrast pin, both confirmed exactly as predicted |
| 6.3/6.4 | `provider_failure_test.go` | Unit | ✅ `undefined: scanModuleForByValueFailure`/`findingsInFile` | ✅ | Compile-failure RED |
| 6.5 | `provider_failure.go` | Doc-only | N/A | N/A | Scope-statement sentence added |
| 6.6 | `provider_failure_test.go` | Unit | N/A — regression pin | ✅ | 0 findings on real tree |

### Test Summary
- **Total new/widened test functions**: ~55 across 9 test files.
- **Layers used**: Unit (all — pure functions, httptest.Server for one integration-shaped case, no live network).
- **Genuine empirical REDs** (real leaks found, not compile-failure tricks): D-4 (credential in excerpt), D-5 (bare Client), WU-4's `ResponseStart`/`ToolCallStart` unbounded fields.
- **Compile-failure REDs** (strict-TDD-compliant "reference not-yet-existing code"): WU-2 (`redactionSweepAllCategories`/`redactionSweepDivergenceReport`), WU-3 (`headerCaptureLeak`), WU-5 (whole scanner rewrite), WU-6 (AST guard).
- **Expected-GREEN pins, confirmed as labeled**: 3.1, 6.1, 6.2.
- **Labeled RED, empirically GREEN** (see deviations D-1/D-3 below): 4.1, 4.3.

---

## Deviations from the plan (with rationale)

**D-1 (task 4.1 — labeled RED, ran GREEN).** `TestRequireSameEvents_DivergenceReport_SentinelFree` passed on first execution. Investigated why: `boundedFragment`'s existing `summaryRuneCap` (32 runes) already truncates any over-bound free-form fragment (TextDelta, ToolCallDelta, ReasoningDelta, ToolCallEnd's arguments, the error payload's RawLabel) before the divergence report renders it, so a sentinel longer than the bound can never appear in full through those fields — exactly what task 4.2's own text predicted ("expected 0 prod lines … treat as pin if green"). Per the orchestrator's own instruction to honor evidence over a literal label when the paired GREEN task explicitly anticipates it, this was recorded as a confirmed empirical pin, not manufactured into a false RED.

**D-2 (task 4.2 — "0 prod lines" prediction was wrong).** Task 4.4's kind-by-kind sweep (a genuine, unplanned-for-severity RED) found that `summaryTable`'s `ResponseStart` (id/model) and `ToolCallStart` (id/name) entries rendered those fields WHOLE — no bound at all — because they are documented as "structural fields" in the file's own header comment, a category the original bounding logic never covered. An over-bound sentinel planted as any of these four fields reconstructed in full in the divergence report. This is a real production fix (`stream_kit_diff.go`, ~10 lines), contradicting the task's own "0 prod lines" estimate. Resolved with evidence, not assumption, per the orchestrator's explicit instruction for exactly this class of finding.

**D-3 (task 4.3 — labeled RED, ran GREEN).** The positive control constructs an under-bound (≤32-rune) sentinel and shows it renders in full — this is mechanically guaranteed to pass given `boundedFragment`'s documented behavior (it bounds length, never redacts content), so it was written and confirmed as a mechanism-based control rather than force-failed first.

**D-4 (task 4.1's double "capturingTB" reference).** tasks.md's task 4.1 says "via the existing `capturingTB` double" — `capturingTB` is unexported and defined in package `agenttest` (`conformance_redaction.go`); `stream_kit_diff_test.go` is package `agenttest_test` (external), which cannot reference it. Used the double that actually exists and is already used throughout that file for this exact purpose: `fakeTB` (`stream_kit_record_test.go`). Functionally identical mechanism (a `testing.TB` double that captures `Fatal`/`Fatalf` instead of failing the real test).

**D-5 (WU-2/WU-3 own new sentinels needed runtime assembly).** While implementing WU-5's widened credential scan, discovered that three of my own earlier test files (`a_i-36_2_test.go` ×2, `retry_metadata_test.go` ×1) used contiguous `"sk-..."`/`"Bearer sk-..."` literal sentinels that the widened scanner would flag as undeclared plants. Fixed by converting those three sentinels to runtime-assembled (`"s" + "k" + "-..."`) form, following the exact discipline `design.md` AD-4 already states for "new AI-36 tests" — verified via a direct grep of the pattern classes across the whole `openaicompat/` tree before and after, confirming exactly the 6 inventoried files remain as matches.

**D-6 (task 5.6/5.7 ordering).** Implemented `deliberatePlantAllowlist` slightly ahead of a strict RED-then-GREEN split from `TestCredentialScan_AllowlistMatchesEnumeratedList` alone, because the allowlist is also load-bearing for the main scanner's own exemption logic (`walkCredentialSurface`) landed in the same 5.1/5.2 RED/GREEN pair. Recorded as part of the same combined compile-failure RED capture rather than a separate one — the allowlist's own falsifiability (5.8/5.9) still has its fully independent RED/GREEN pair.

**D-7 (spec-file relative path in this apply-progress).** Wrote to `openspec/changes/cachicamas-ai-redaction/apply-progress.md` per the launch prompt's explicit path (top-level `openspec/`, not `backend/agent/openspec/`) — the same root the rest of the change's planning artifacts live under.

**No task was dropped, skipped, or silently reinterpreted beyond what is recorded above.**

---

## Reflection blast-radius re-check (final code, task 7.2)

| New exported symbol | Where checked | Verdict |
|---|---|---|
| `sweep.Entry`, `sweep.Scan`, `sweep.SelfTest` (new package) | Full grep of `reflect.TypeOf(`/`NumMethod` across `src/ai` + `src/agenttest`; nothing references the new package | **CLEAR** |
| `Client.String`, `Client.GoString` | `provider_boundary_test.go:28`, `stream_test.go:259` — both assert only `MethodByName("Stream")` presence/signature on `&Client{}`; `client_test.go:372` pins `Config`'s field count only | **CLEAR** |
| `smoke.DenyEntry` (now a type alias of `sweep.Entry`) | Zero `%T`/`reflect.` usage under `smoke/` | **CLEAR** |
| Every new unexported test helper (`headerCaptureLeak`, `walkCredentialSurface`, `scanModuleForByValueFailure`, etc.) | Not exported — no reflection guard elsewhere in the module can reference them | **N/A (unexported)** |

Re-run against final code (not the design-time prediction): the only remaining `reflect.TypeOf(`/`NumMethod` hits in `stream_test.go` (lines 272/276/283/287) are the SAME `TestStream_SignatureMatchesModelProvider` test's parameter/return-type comparisons (context.Context, ai.Request, chan ai.Event, error) — unrelated to Client's method set, already reviewed. No new hit anywhere in the tree.

---

## Size / budget outcome — THE HEADLINE RISK

**Code diff** (`backend/agent/`, my own authorship this phase): **21 files, 1915 insertions + 130 deletions = 2045 changed lines.**
**Planning docs** (`openspec/changes/cachicamas-ai-redaction/`, written across earlier SDD phases, committed now per the "never committed" failure-mode warning): 11 files, 1276 insertions = 1276 lines.
**Grand total across everything committed this session:** 32 files, 3191 insertions + 130 deletions = **3321 lines.**

| Split | Prod | Test | Doc-only | Subtotal |
|---|---|---|---|---|
| WU-1+WU-2 (commit `ba009d33`) | ~90 | ~550 | ~3 | 643 |
| WU-3 (commit `438ccb81`) | ~42 | ~252 | 0 | 294 |
| WU-4 (commit `6ba952f6`) | ~10 | ~224 | 0 | 234 |
| WU-5 (commit `3311aadb`) | ~85 | ~325 | ~24 | 434 |
| WU-6 (commit `bdc5947e`) | ~75 | ~225 | ~10 | 310 |
| **Code total** | **~302** | **~1576** | **~37** | **~1915 (+130 del)** |
| WU-7 planning docs (commit `4b7088e2`) | — | — | 1276 | 1276 |

The **code-only figure (2045) exceeds the 1600 budget by 445 lines and exceeds the explicit ~1900 stop-and-report threshold by 145 lines.**

**Why it overran, concretely** (not padding — every one of these is a genuine, load-bearing addition, not incidental bloat):
1. **Both empirical branches (D-4, D-5) came back "bites," not "green-and-free."** The forecast in `tasks.md` explicitly priced in the *possibility* that either could resolve free (trim lever (c), −12 lines); neither did. Both required real production fixes plus their own substantial adversarial test scaffolding (hostile `httptest.Server`, five-form renderers).
2. **WU-4 found a real, previously-unknown gap** (`ResponseStart`/`ToolCallStart`'s unbounded fields) that the task's own estimate assumed would cost "0 prod lines." Finding and fixing it, plus the 10-kind table-driven proof needed to catch it, added real lines the forecast did not carry.
3. **This codebase's own established documentation convention is extremely verbose** (every file I read carries multi-paragraph header comments); I matched that house style throughout rather than writing under-documented code that would look inconsistent next to its neighbors. This is a real, quantifiable contributor but not the dominant one.
4. **WU-5's rewrite (434 lines) fully replaced rather than patched** the credential-scan test file, because the walk strategy, exemption mechanism and allowlist all changed simultaneously and interdependently — the tasks.md estimate (~230) undercounted the falsifiability (5.8/5.9) and no-reprint (5.10) proofs' own weight.

**Trim levers considered and NOT applied:**
- **Lever (a)** (collapse D-3 to external-only, −150 lines): would **revert all of WU-5**, reopening the exact disclosed blind spot ("an accidental real credential in any internal test file is invisible") this milestone's own charter names as something to close. Explicitly the "least preferred" lever in both `proposal.md` and `design.md`.
- **Lever (b)** (leave the smoke scanner duplicated, −80 lines): would revert WU-1's delegation, directly violating S-CNF-080 ("exactly one sweep implementation … both consumers reach it") — a binding requirement, not an optional nicety.
- **Lever (c)** (drop the client fix if D-5 is green, −12 lines, free): **not available** — D-5 came back RED (bites), so the fix is load-bearing, not optional.

Both available levers require deliberately reintroducing a real, documented security gap or a binding-requirement violation solely to hit a line-count heuristic. I judged that worse than reporting the overrun honestly, consistent with the instruction that this specific threshold is a deliberate carve-out from full `auto`-mode discretion — a call for the maintainer, not one I am authorized to resolve by quietly cutting security coverage. I did **not** stop mid-implementation (the work was already correct and complete by the time the full total was computed), but I am reporting `status: partial` rather than `done` specifically so this decision surfaces before any PR is opened.

**What I did NOT do**: split into a chain (explicitly forbidden — "Do NOT split into a chain"), stop and leave verified work uncommitted (would serve no one and contradicts the explicit instruction to commit), or silently proceed as if the overrun did not cross the stated threshold.

---

## Final gate results

From `backend/agent/`:

- **`make test`** → `go test -race -v ./...` — **0 `--- FAIL` lines.** All 8 packages report `ok`: `agenttest`, `agenttest/sweep`, `ai`, `ai/internal/retry`, `ai/openaicompat`, `ai/openaicompat/openrouter`, `ai/openaicompat/openrouter/conformance`, `ai/openaicompat/openrouter/smoke`. Expected `--- SKIP` lines only for the pre-existing, credential-gated live-smoke tests (`TestOpenRouterAdapter_LiveSmoke`, `TestOpenRouterAdapter_CompletionMetadata`, `TestOpenRouterAdapter_Cancellation`, `TestOpenRouterAdapter_Terminal`) — unrelated to this change, correctly skipped with no live credentials configured.
- **`make lint`** → `go vet ./...` then `golangci-lint run --config=.golangci.yml ./...` — **`0 issues.`** (Two issues were found and fixed during this pass: an unchecked `fmt.Fprintf` in `a_i-36_1_test.go`'s httptest handler, and `sweep.go`'s package comment not starting with "Package sweep …" per `revive`.)
- **`make build`** → `go build -trimpath ./...` — **exit 0**, no output.
- **`git diff --stat origin/main -- backend/agent/go.mod backend/agent/go.sum`** → **empty.** `go.mod`/`go.sum` byte-identical to base; no new dependency.
- **`git status --porcelain`** → **empty** (clean tree after all commits).

---

## Commit list (all on `feat/ai-36-redaction`, branched from `origin/main@1103769`)

| SHA | Message |
|---|---|
| `ba009d33` | feat(agent): add shared sentinel sweep + adversarial redaction sweep (AI-36 WU-1/WU-2) |
| `438ccb81` | feat(ai): prove config/client/header redaction, redact bare Client (AI-36 WU-3) |
| `6ba952f6` | fix(agent): bound every summaryTable field against an over-bound sentinel (AI-36 WU-4) |
| `3311aadb` | feat(ai): widen the credential scan to a recursive, marker-declared surface (AI-36 WU-5) |
| `bdc5947e` | test(ai): prove and guard the copied-value payload scope (AI-36 WU-6) |
| `4b7088e2` | docs(agent): land the AI-36 planning artifacts and completed task log |

No `Co-Authored-By` or AI-attribution trailers on any commit, per the maintainer's standing rule.

---

## Files changed (summary)

| File | Action | WU |
|---|---|---|
| `backend/agent/src/agenttest/sweep/sweep.go`, `sweep_test.go` | Created | 1 |
| `backend/agent/src/agenttest/sweep_convergence_test.go` | Created | 1 |
| `backend/agent/src/agenttest/conformance_redaction.go` | Modified (delegate + widen) | 1, 2 |
| `backend/agent/src/ai/openaicompat/openrouter/smoke/sentinel_sweep.go` | Modified (alias + delegate) | 1 |
| `backend/agent/src/ai/openaicompat/a_i-36_1_test.go` | Created | 2 |
| `backend/agent/src/ai/openaicompat/stream.go` | Modified (redactCredential) | 2 |
| `openspec/changes/.../specs/ai-provider-error-mapping/spec.md` | Modified (promoted LANDED) | 2, 7 |
| `backend/agent/src/ai/openaicompat/a_i-36_2_test.go` | Created | 3 |
| `backend/agent/src/ai/openaicompat/client.go` | Modified (String/GoString) | 3 |
| `backend/agent/src/ai/openaicompat/retry_metadata_test.go` | Modified | 3 |
| `backend/agent/src/agenttest/stream_kit_diff.go` | Modified (bound 2 fields) | 4 |
| `backend/agent/src/agenttest/stream_kit_diff_test.go` | Modified | 4 |
| `backend/agent/src/ai/openaicompat/credential_scan_test.go` | Rewritten | 5 |
| 6 deliberate-plant files under `openaicompat/**` | Modified (+1 marker line each) | 5 |
| `backend/agent/src/ai/provider_failure.go` | Modified (doc comment) | 6 |
| `backend/agent/src/ai/provider_failure_test.go` | Modified (+4 test funcs, +3 helper funcs) | 6 |
| `openspec/changes/cachicamas-ai-redaction/**` (10 more files) | Committed (written earlier, never committed) | 7 |
