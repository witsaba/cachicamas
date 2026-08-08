```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:6658c86e38d94d78732c49ebedd1b75f70ed1db5232ddeebd3c9bf08ddb3fc7d
verdict: fail
blockers: 0
critical_findings: 0
requirements: 12/18
scenarios: 72/80
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:f310184620a5f75ba7df78d541d2f83424e6c7020a4b4421b01b9f6041b70b70
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report (FINAL — supersedes `verify-report.md`)

**Change**: `cachicamas-ai-observability` (AI-37 — Add the observability boundary)
**Branch**: `feat/ai-37-observability` @ **`0b1fbd61`**, base `origin/main@ff28240a`, 23 commits
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-37-observability`
**Mode**: Strict TDD · hybrid store · `size:exception` pre-granted (size is **not** a finding)

This report **supersedes** the historical `verify-report.md` (verdict FAIL, committed at `0b1fbd61` as the record of the pre-remediation state). It reconciles that report's 1 CRITICAL / 6 WARNING / 6 SUGGESTION, both Judgment Day rounds, and every deferred item, against the code at HEAD.

**Scope note**: this pass is reconciliation, spec-truth and archivability. It does not re-open the two completed Judgment Day rounds. The two CRITICALs and the in-band-error-frame reorder were independently defeated by the orchestrator via `go test -overlay` at this HEAD; those defeats are cited, not repeated.

---

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 21 |
| Tasks complete | 21 |
| Tasks incomplete | **0** |
| Requirements assessed | 18 (16 `R-*` + 2 `NFR-AOB-*`) |
| Scenarios assessed | **80** (52 new + 28 restated) |
| Commits | 23 |
| Working tree | clean (`git status --porcelain` empty) |

`grep -c '^- \[x\]' tasks.md` → 21 · `grep -c '\[ \]' tasks.md` → **0**. Task state matches code state: every phase C1…C9 has a landed commit, and the two remediation rounds plus both Judgment Day rounds are recorded in `apply-progress.md` §§ "Remediation round", "Judgment Day round 1", "Judgment Day round 2".

**Scenario total moved 77 → 80** since the historical report: `S-AOB-040` (round 1), `S-AOB-041` and `S-APC-085` (round 2). Requirement total is unchanged at 18 — no correction round added a requirement.

---

### Build & Tests Execution (re-run independently at HEAD)

**Tests**: ✅ 9/9 packages `ok`, **0** `FAIL` lines, 2380 `--- PASS` lines, under `-race`.

```text
cd backend/agent && make test        # go test -race -v ./...
TEST_EXIT=0
grep -c '^--- FAIL\|^FAIL' → 0
grep -c '^ok '            → 9
```

**Build**: ✅ `cd backend/agent && make build` → exit 0 (output hash byte-identical to the historical report's, `617ff8b5…`).

**Lint**: ✅ `cd backend/agent && make lint` → exit 0, `0 issues.`

**Coverage**: ➖ Not available — this module's `Makefile` declares no coverage target.

All three exit codes were captured un-piped.

---

## Part 1 — Reconciliation of every open finding

### 1a. Historical verify CRITICAL

| ID | Finding | Status at HEAD | Evidence |
|---|---|---|---|
| **C-1** | `retry.count` recorded on every exit, asserted on none; `S-AOB-015` UNTESTED | ✅ **FIXED** | `ai37_retry_count_test.go` ships a 5-row table (`unretried_success`→0, `retried_twice_then_success`→2, `non_retryable_terminal`→0, `budget_exhausted`→3, `cancelled`→0). `TestAI37_RetryCount_PresentOnEveryExit` PASS. Orchestrator's `go test -overlay` hard-wiring `recordRetryCount` to `0` fails 2 rows; deleting the call from `endSpanPreHandover` fails 3. Falsifier proven; worktree never mutated. |

### 1b. Historical verify WARNINGs (6)

| ID | Finding | Status | Evidence / why it does not block |
|---|---|---|---|
| **W-1** | `S-AOB-029` failure output names the **vector only**, not the span and the attribute key | ⚠️ **STILL OPEN — ACCEPTABLE** | `ai37_denylist_test.go:270` is still `t.Errorf("denylist leak: vector %q found in the corpus …", entry.Vector)`. `R-AOB-007` requires "the vector, the span and the attribute key". **Non-blocking**: this is failure-message richness on a branch that only executes when a leak already exists; the *absolute* half of the requirement (never reprint matched bytes) is satisfied, and 8/8 corpus channels were defeat-proven. `S-AOB-029` graded **PARTIAL**. |
| **W-2** | Deviation 9's "strictly stronger" framing inflated (reasoning canary non-falsifiable by construction) | ✅ **FIXED (spec-level)** | Round 1 amended `R-AOB-008` item 3 and `S-AOB-032` to state exactly this: spec.md:170 now says the canary's absence "is guaranteed by the wire-decode drop, not demonstrated by this coverage guard, and it is retained for documentation value rather than as additional coverage evidence". The shipped guard enumerates exactly 4 kinds (`ai37_denylist_test.go:249`: `TextDelta, ToolCallStart, Completion, Error`). Spec text and code now agree. `S-AOB-032` **PARTIAL → COMPLIANT**. |
| **W-3** | `S-AOB-037` covered 14 of 15 no-tracer terminal shapes | ✅ **FIXED** | `grep -c 'PASS: TestAI37_NoopEquivalence_NoTracerConfigured_NoPanicAcrossTerminalPaths/'` → **15**. (Sibling `TestAI37_SpanLifecycle` also → 15.) `S-AOB-037` **PARTIAL → COMPLIANT**. |
| **W-4** | `S-AOB-011` shipped with no apply-recorded falsifier; verify supplied it | ✅ **CLOSED BY RECORD** | Verify defeat D is the recorded falsifier: `finalizeSpan` failure branch never ending → 10 post-handover rows FAIL; `endSpanPreHandover` never ending → 4 pre-handover rows FAIL; double-end → `success_done` FAIL. That evidence is carried into this archive record here, which is what W-4 asked for. No code change was owed. |
| **W-5** | `S-AOB-009`'s duration clause not assertable — `tracetest.Span` records no timestamps | ⚠️ **STILL OPEN — ACCEPTABLE** | Confirmed at HEAD: `tracetest.go` exposes `Started()`/`Ended()`/`AssertAllEndedOnce()` and no time field. **Non-blocking**: the load-bearing half of `S-AOB-009` (exactly one span across a retried-then-succeeded attempt) IS now proven — it is the `retried_twice_then_success` row of `TestAI37_RetryCount_PresentOnEveryExit`, added by the remediation round. The span-start *position* is structurally correct and directly readable: `stream.go:222` `startSpan(...)` precedes `stream.go:224` `retry.Loop(...)`. Only the metric assertion ("its recorded duration encloses every attempt") is unassertable. `S-AOB-009` graded **PARTIAL**. |
| **W-6** | `apply-progress.md` diff stat stale | ✅ **FIXED, then drifted again by design** | `apply-progress.md:386` records `34 files changed, 6000 insertions(+), 163 deletions(-)` — exact and correct as of `4bf8e90a`. At `0b1fbd61` the actual stat is `35 files changed, 6495 insertions(+), 163 deletions(-)`, the difference being the orchestrator-owned `verify-report.md` (+495) committed *after* apply's last correction. This number is inherently unstable across orchestrator-owned artifact commits (writing this report moves it again). **Not an apply defect; do not chase it.** See SUGGESTION S-5 below. |

### 1c. Historical verify SUGGESTIONs (6)

| ID | Finding | Status | Evidence |
|---|---|---|---|
| **S-1** | `S-AOB-003`: add the closure/`auto/sdk` reason at `import_boundary_test.go` | ⚠️ **PARTIALLY ADDRESSED — ACCEPTABLE** | `import_boundary_test.go:99-107` now carries a declination note in place: root `go.opentelemetry.io/otel` is named "Deliberately absent", "§ D3 permits it; D-1 declines it — Layer 1 takes an injected TracerProvider instead", and `auto/sdk` appears in the adjacent absent-path list (`:102`) and in the update procedure (`:176-178`). What is present satisfies "records the declination as a deliberate strict subset". What is **not** stated is the causal reason `S-AOB-003` names verbatim — *that path's package closure contains an SDK-named package*. The reader is told the declination is deliberate but is given the injection rationale, not the closure rationale. `S-AOB-003` graded **PARTIAL**. Non-blocking: prose-only. |
| **S-2** | Record that `attribute.Value.Emit()` is deprecated at v1.44.0 and `String()` is the correct accessor | ✅ **CLOSED BY RECORD** | Carried into this archive record. `S-AOB-027`'s substance is satisfied and falsifiable (verify defeat 5b: removing `String()` collapses the corpus). |
| **S-3** | Record `S-AOB-028` / `S-APC-083` as satisfied-by-construction | ✅ **CLOSED BY RECORD** | Both are structurally unconstructible once the root global getter is excluded from the import set. Independently re-confirmed at HEAD: `grep -rn 'otel\.GetTracerProvider\|otel\.SetTracerProvider\|"go.opentelemetry.io/otel"' backend/agent/src/` returns **exactly one** hit, `import_boundary_test.go:180`, which is a module `require` table literal — not a package import. There is no ambient path to install a global provider through, so no test can construct the "globally installed provider records nothing" observation as a discriminating one. Both graded **PARTIAL** and recorded here as satisfied-by-construction. |
| **S-4** | Canonical `S-AGM-020` asserts `go.work` declares `go 1.26.3`; shipped file declares `1.26.5` | ⚠️ **STILL OPEN — ESCALATED TO WARNING (see W-A)** | Re-confirmed: `go.work:1` → `go 1.26.5`. `backend/agent/go.mod` → `go 1.26.3` (so `S-AGM-001` is TRUE). No test asserts the `go.work` directive. See W-A below for why this is now a WARNING rather than a SUGGESTION. |
| **S-5** | `spec.md:18` arithmetic ("14 requirements touched" vs 16; "29 restated" vs 28) | ❌ **STILL OPEN, AND WORSE** — see Part 3 | Not fixed, and both correction rounds added new drift to the same index. Full arithmetic audit in Part 3. |
| **S-6** | `runNoPanic`'s `recover()` covers only the calling goroutine | ⚠️ **STILL OPEN — ACCEPTABLE, correctly skipped** | Apply evaluated and skipped it because closing it requires a production change to `run` plus a report-back mechanism. That is the right call under a skip-and-say-so instruction. Mitigating: a panic in the producer goroutine crashes the test binary, which fails the package anyway — it fails loudly, just not with the row's own name. Non-blocking. |

### 1d. Judgment Day deferred items

| Item | Status | Evidence |
|---|---|---|
| **Nil-check scan is a four-literal substring table**, evadable by a renamed local (round 2, Judge A SUGGESTION, deferred as last permitted round) | ⚠️ **STILL OPEN — ACCEPTABLE** | Confirmed at `ai37_noop_equivalence_test.go:286-291`: exactly `"span == nil"`, `"span != nil"`, `"tracer == nil"`, `"tracer != nil"`, matched with `strings.Contains` on the trimmed line (`:299-310`). `if s := span; s == nil` evades it. **Non-blocking**: the *substantive* claim is independently TRUE at HEAD — I ran `grep -rn "span == nil\|span != nil\|tracer == nil\|tracer != nil" backend/agent/src/ai/ \| grep -v _test.go` over **all** of Layer 1 and it returns nothing. The guard is weaker than the claim; the claim itself holds. Carries a non-vacuity floor (`wantMinScannedFiles = 10`, `:389`) and a falsifiability control (`TestAI37_NoopEquivalence_NilCheckScan_DetectsStagedMutation`, PASS). |
| **`S-AOB-038` says "Layer 1 sources"; the widened scan proves package scope only** | ⚠️ **STILL OPEN — ACCEPTABLE, but graded PARTIAL** — see W-B | See W-B below. |
| **`S-AOB-040` carries a `*(review)*` tag that may be a round-1 mislabel** | ⚠️ **CONFIRMED MISLABEL, and it has a second instance round 2 did not notice** — see W-C | See W-C below. |
| **Stamper sequence gap + bounded 5s (≈10s) delay accepted (round 2, Judge A WARNING)** | ✅ **DECISION VERIFIED CORRECT** — see Part 2b | |

---

## Part 2 — Were the round-2 decisions *right*, not merely implemented?

### 2a. `finalizeSpan` records `http.response.status_code` on post-handover failures; `R-AOB-006` widened rather than narrowed

**Verdict: the correct call, and the spec text is now true of the code on every path.**

The code (`trace.go:156-184`) sets `statusCodeKey` **unconditionally at `:169`**, before branching on `outcome.failure` at `:171`. So both the failure branch and the success branch record it.

The load-bearing question is whether `statusCode` is genuinely in hand on *every* path `finalizeSpan` renders. I traced the call graph exhaustively:

- `finalizeSpan` has exactly **one** call site: `stream.go:640`, `defer finalizeSpan(span, resp.StatusCode, &outcome)`, inside `run`.
- `run` has exactly **one** launch site: `stream.go:250`, `go run(ctx, resp, out, span)`.
- That launch is guarded by `stream.go:224-246`: `retry.Loop` must have returned `err == nil` (so `resp != nil`), **and** the content-type check at `:240` must have passed. Both pre-handover exits return before the launch.

Therefore `resp.StatusCode` is a real, single, unambiguous status code on all 11 of `run`'s named terminal paths. `R-AOB-006` bullet 2's clause — "every post-handover (mid-stream) terminal failure, which this capability only ever begins producing after that one response has already been obtained" (`spec.md:130`) — is **exactly true of the code**, with no path excepted.

**Why widening the code beats narrowing the spec**: bullet 1's omission is justified by *genuine multi-attempt ambiguity* — `retry.Loop` returns `nil` for the response on **every** error exit (`retry.go:92,95,98,112,116`), so at the recording site there is literally no response value to read a code from, and only a status *class* survived. Bullet 2's paths have no such ambiguity: exactly one response, its code held in a live variable. `R-AOB-006`'s own stated purpose is that narrowing a claim past what is actually available "would itself be the falsehood this requirement exists to prevent" (`spec.md:130`). Narrowing the spec back would have made the requirement contradict its own rationale. Recording `http.response.status_code=200` beside `error.type=<category>` is standard non-contradictory OTel telemetry: a transport-layer fact beside an application-layer fact.

Both new rows are test-backed and green: `TestAI37_Attributes_ContentTypeRefusalRecordsExactStatusCode` (pre-handover, `S-AOB-040`) and `TestAI37_Attributes_MidStreamFailureRecordsExactStatusCode` (post-handover, `S-AOB-041`) both PASS. The sibling omission test `TestAI37_Attributes_TerminalFailureOmitsStatusCode` still PASSes unchanged, so the split is proven in both directions.

### 2b. Keeping the best-effort recovery sends (Stamper gap + bounded delay)

**Verdict: correct, and both mitigating claims independently confirmed.**

*Claim 1 — `ai.CheckStream` does not check contiguity.* **CONFIRMED by source read**, `stream_check.go:52-54`:

```
// It reports the first violation in stream order (V-FAIL-04) and whether a
// terminal event was seen, independently (R-AEE-018). It does not check
// 1…N sequence contiguity — that is AI-22.3's charter (design.md D10).
```

*Claim 2 — no shipped assertion fails.* **CONFIRMED, and I checked the two openaicompat sites that could have.** There ARE shipped 1..N contiguity assertions inside this package — `ai37_span_lifecycle_test.go:493-498` (mid-stream cancellation) and `stream_test.go:443-448` (`S-ATS-010`, normal success). Both PASS. They do not fail because the gap is only reachable on the **abandoned-consumer** path: the precondition for the gap is that the consumer stopped draining, so by construction no consumer observes events on both sides of it. A consumer that keeps draining wins the send and never enters the recovery branch. Whole suite exit 0 confirms it empirically.

*The cost is real and visible.* `emitFailureSendBound = 5 * time.Second` (`stream.go:147`), and the `[DONE]` recovery can perform two bounded sends (`TextBlockEnd` at `:758`, `ErrorEvent` at `:767`), so the worst case is ~10s before `finalizeSpan` → drain → `close(out)`. This is directly observable in the suite: `TestAI37_TerminalCompletionSend_LosingTheRaceRecordsFailure` takes **5.01s**, the single slowest test in the module.

*Why keeping is still right*: dropping the recovery sends would regress the **pre-existing** `S-AEM-051/052` ("MUST still surface a typed terminal failure on cancel/deadline") for a caller using a deadline rather than an explicit cancel who resumes draining per this package's own drain-before-close convention (AI-33.5). Such a caller would then receive a typed `ErrorEvent` for every *other* mid-stream failure shape except this one. Trading a landed requirement for a bounded latency on an already-abandoned stream is the wrong trade. The posture also matches the pre-existing events-loop recovery (`stream.go:854,864,926,934`) — this is not a new shape. Documented in-source at `stream.go:714-748`. Recorded as SUGGESTION S-6 for a future revisit of the bound, not as a defect.

---

## Part 3 — Spec internal arithmetic (index vs delta files)

I counted every scenario and requirement definition directly out of the three delta files.

| Source | Requirements | Scenarios |
|---|---|---|
| `specs/ai-observability-boundary/spec.md` | 9 `R-AOB` + 2 `NFR-AOB` = **11** | `S-AOB-001`…`S-AOB-041` = **41** (all new) |
| `specs/agent-module-scaffold/spec.md` | **4** (`R-AGM-001,003,005,008`) | **23** = 6 new (`068`…`073`) + 17 restated (`001`–`004`, `020`–`023`, `040`–`048`) |
| `specs/ai-provider-client/spec.md` | **3** (`R-APC-001,003,016`) | **16** = 5 new (`081`…`085`) + 11 restated (`001`–`005`, `011`–`016`) |
| **Total** | **18** | **80** = **52 new + 28 restated** |

**The index and one delta Identity table are STALE.** Four distinct drifts:

| Location | Says | Should say | Cause |
|---|---|---|---|
| `spec.md:14` | `S-AOB-001` … **`S-AOB-039`** | `S-AOB-001` … `S-AOB-041` | rounds 1+2 added `S-AOB-040`, `S-AOB-041` |
| `spec.md:16` | `S-APC-081` … **`S-APC-084`** new | `S-APC-081` … `S-APC-085` | round 1 added `S-APC-085` |
| `spec.md:18` | "**49 new scenarios** (39 + 6 + 4)"; "**29** landed scenarios restated" | "**52 new scenarios** (41 + 6 + 5)"; "**28** landed scenarios restated" | rounds 1+2, plus the pre-existing 29-vs-28 miscount (verify S-5, never fixed) |
| `spec.md:18` | "**14 requirements touched** — 9 new, 2 added, 5 modified" | **16** (9 + 2 + 5 = 16) | pre-existing arithmetic error (verify S-5, never fixed) |
| `specs/ai-provider-client/spec.md:17` | "New scenarios: `S-APC-081` (under `R-APC-001`), `S-APC-082` … `S-APC-084` (under `R-APC-016`)" | must also name `S-APC-085` | round 1 added the scenario to the body but not to the Identity table |

**What round 2 *did* fix, correctly**: the `ai-observability-boundary` delta's own two range references — Identity `spec.md:17` (`S-AOB-001` … `S-AOB-041` ✅) and Acceptance criteria `spec.md:230` (`S-AOB-001` … `S-AOB-041` ✅). Those are right. The drift is confined to the **index** (`spec.md`) and the **`ai-provider-client` delta's Identity row**, neither of which round 2 was assigned to touch.

**The index does NOT agree with the delta files.** This is a documentation-accuracy defect in artifacts that the archive step promotes. It is **not** a code or test defect and blocks nothing behavioural — hence WARNING, not CRITICAL. See W-D.

---

## Part 4 — OpenRouter door sanity check

New public surface added late under widest-scope authorization. Three properties checked:

1. **Used verbatim** ✅ — `openrouter/wrapper.go:72` declares `TracerProvider trace.TracerProvider`; `NewProvider` (`:149-157`) passes `TracerProvider: cfg.TracerProvider` straight into `openaicompat.Config`. No wrapping, no conditional, no substitution, no default of its own. Proven end-to-end by `TestNewProvider_InjectedTracerProviderRecordsTheSpan` (`tracer_test.go:23`, PASS), which asserts `provider.Started() == 1` on the wrapper's own span.
2. **Defaults to no-op** ✅ — the wrapper adds no default; a nil field flows to `openaicompat.New`, which at `client.go:137-139` does `if tracerProvider == nil { tracerProvider = noop.NewTracerProvider() }` and derives the tracer once at `:146`. That is `R-APC-016`'s structural substitution, unchanged.
3. **Opens no ambient-authority or content path** ✅ — the wrapper's only new import is `go.opentelemetry.io/otel/trace` (the interface package). No import of the root `go.opentelemetry.io/otel` global-getter package exists anywhere in the module (independently re-grepped; the single textual hit is a `require`-table literal at `import_boundary_test.go:180`, a module require, not a package import — precisely the distinction `R-AGM-008` exists to make). `trace.TracerProvider` is an interface value, carries no content, and the wrapper reads no field of it. The module-wide dependency guard (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestLayer1_DependencySet_ExactRequiresAndClosure`) both PASS at HEAD, so the closure did not grow.

`R-APC-016`'s new clause requiring every composing wrapper to expose the same door (`specs/ai-provider-client/spec.md:84`) is satisfied by the only concrete adapter this module ships. **The door is safe.**

---

## Spec Compliance Matrix (summary)

**72/80 COMPLIANT · 8 PARTIAL · 0 UNTESTED · 0 FAILING.**

Movement since the historical report (68/77 COMPLIANT, 8 PARTIAL, 1 UNTESTED):

| Scenario | Was | Now | Why |
|---|---|---|---|
| `S-AOB-015` | ❌ UNTESTED | ✅ COMPLIANT | `TestAI37_RetryCount_PresentOnEveryExit`, 5 rows, overlay-falsified |
| `S-AOB-032` | ⚠️ PARTIAL | ✅ COMPLIANT | round 1 amended the spec to the 4 achievable kinds; guard matches |
| `S-AOB-037` | ⚠️ PARTIAL | ✅ COMPLIANT | 15/15 no-tracer rows |
| `S-AOB-040` | — | ✅ COMPLIANT (new) | `TestAI37_Attributes_ContentTypeRefusalRecordsExactStatusCode` |
| `S-AOB-041` | — | ✅ COMPLIANT (new) | `TestAI37_Attributes_MidStreamFailureRecordsExactStatusCode` |
| `S-APC-085` | — | ✅ COMPLIANT (new) | `TestNewProvider_InjectedTracerProviderRecordsTheSpan` |
| `S-AOB-038` | ✅ COMPLIANT | ⚠️ PARTIAL | **regrade** — guard scope is one package, scenario says "Layer 1 sources" (W-B) |
| `S-AGM-020` | ✅ COMPLIANT | ⚠️ PARTIAL | **regrade** — restated text says `go.work` declares `go 1.26.3`; it declares `1.26.5` (W-A) |

The 8 PARTIALs: `S-AOB-003` (declination reason differs), `S-AOB-009` (duration clause unassertable), `S-AOB-027` (pointer-shaped field not constructible in this API), `S-AOB-028` (satisfied-by-construction), `S-AOB-029` (failure names vector only), `S-AOB-038` (scan scope), `S-APC-083` (satisfied-by-construction), `S-AGM-020` (`go.work` version).

**Requirements 12/18 fully complete.** The 6 not fully complete, each by exactly one or more PARTIAL scenario and none by a failing one: `R-AOB-001`, `R-AOB-003`, `R-AOB-007`, `R-AOB-009`, `R-AGM-003`, `R-APC-016`.

### Milestone subnodes

| Subnode | Verdict | Evidence |
|---|---|---|
| **AI-37.1** — dependency + injection boundary | ✅ real | 4 defeat shapes (root otel, closure pin, `otel/sdk`, `otel/exporters`) all bite; set-equality pin proven by the `semconv/v1.37.0` case; no ambient getter anywhere |
| **AI-37.2** — one span, closed once, twelve keys | ✅ real, gap closed | 15/15 terminal rows; 12/12 character-exact keys; `retry.count` gap (the historical CRITICAL) now falsified on 5 rows |
| **AI-37.3** — content denylist by absence + 4 non-vacuity guards | ✅ real | 8/8 corpus channels defeat-proven; all 4 guards plus the 5th field-count guard bite |
| **AI-37.4** — no-op equivalence, no nil check | ✅ real, one scope caveat | 15/15 no-tracer rows (was 14/15); drained sequences equal; nil-check absence independently confirmed across all of Layer 1, guard scans one package (W-B) |

---

## TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | ✅ | `apply-progress.md` carries per-phase RED/GREEN records for C1…C9 plus both remediation and both Judgment Day rounds |
| All tasks have tests | ✅ | 21/21 |
| RED confirmed (test files exist) | ✅ | all AI-37 test files present and compiling |
| GREEN confirmed (tests pass) | ✅ | 66 `TestAI37_*` subtests PASS, 0 FAIL, under `-race` |
| Triangulation adequate | ✅ | 15-row terminal tables ×2, 5-row retry table, 8-vector denylist corpus, 5 guards |
| Safety net for modified files | ✅ | pre-existing suites re-run green at every commit |

**Recorded RED discipline**: the two Judgment Day CRITICALs and the in-band-error reorder were each proven with `go test -overlay` against the pre-fix source, worktree never mutated — the standing overlay discipline, correctly applied. One documented deviation remains from apply (C5's `git stash` RED); the historical report accepted it as non-vacuous and re-proved the same behaviours by overlay anyway (defeat D). Unchanged, still accepted.

## Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit / in-process integration (`httptest` + in-memory recording tracer) | 66 `TestAI37_*` subtests + the wrapper and boundary guards | 7 AI-37 files (`ai37_*.go` ×6, `openrouter/tracer_test.go`) | `go test -race` |
| E2E | 0 | 0 | not applicable — no exporter, by charter |
| **Total (module)** | **2380 PASS lines** | 9 packages | |

Source-scanning guards (`import_boundary_test.go`, the nil-check scan, the denylist sweep) are static-analysis tests executed by the runner, not substitutes for runtime evidence — every spec scenario graded COMPLIANT above has a covering test that PASSED at runtime.

## Changed File Coverage

➖ Coverage analysis skipped — no coverage target in `backend/agent/Makefile`. Not a failure.

## Assertion Quality

✅ **All assertions verify real behavior.** Audited the seven AI-37 test files. No tautologies, no orphan empty-collection checks, no type-only assertions standing alone, no ghost loops. Every absence assertion in `ai37_denylist_test.go` is fenced by a non-vacuity guard that `t.Fatalf`s (`:182` self-test, `:227` corpus-non-empty, `:238` attribute cardinality, `:251` event coverage), so no clean scan can run behind a broken control — the exact anti-vacuity posture `R-AOB-008` demands. The nil-check scan carries its own floor (`wantMinScannedFiles = 10`) plus a staged-mutation control. `TestAI37_NoopEquivalence_NilCheckScan_DetectsStagedMutation` writes its mutation to `t.TempDir()`, never the worktree.

**Assertion quality**: 0 CRITICAL, 0 WARNING.

## Quality Metrics

**Linter**: ✅ `0 issues.` (exit 0) · **Type checker**: ✅ `go vet ./...` clean, `make build` exit 0.

## Design Coherence

| Decision | Followed? | Notes |
|----------|-----------|-------|
| AD-1 otel v1.44.0 under the workspace | ✅ | `go.mod` requires `otel v1.44.0`, `otel/trace v1.44.0`, `xxhash/v2 v2.3.0` indirect — exactly the pinned table |
| AD-2 closure pin + update procedure | ✅ | `wantExternalClosure` set-equality pin; procedure at `import_boundary_test.go:173-178` |
| AD-3 recorder embeds `noop.*` | ✅ | `tracetest` embeds and overrides; method-count pins Span=11, Tracer=2, TracerProvider=2 |
| AD-4 reflect method-count pin | ✅ | `tracetest_test.go:264-270,303-305` |
| AD-5 uniform span close | ✅ | one `finalizeSpan` call site + one `endSpanPreHandover` per pre-handover exit; defer LIFO ordering verified at `stream.go:626-640` |
| AD-6 `retry.count` / `stream.event_count` carriers | ✅ | **upgraded from ⚠** — the historical report's only AD gap was `retry.count` unasserted; now falsified on 5 rows |
| AD-7 `gen_ai.system` = dialect | ✅ | `genAISystem = "openai_compat"`, `trace.go:62`; cross-vendor equality test PASS |
| AD-8 span kind + start attributes | ✅ | `trace.go:77-82` |

---

## Issues Found

### CRITICAL (0) — no defect blocks archive

Both Judgment Day CRITICALs are fixed and independently falsified by the orchestrator at this HEAD via `go test -overlay`, with the worktree never mutated:
- **`[DONE]` terminal race** — removing the `if !sendEvent(completion)` branch fails 3 assertions.
- **in-band error frame** — moving `outcome.failure = failure` back after the block-close send fails 2 assertions.
- **`retry.count`** (historical C-1) — hard-wiring `recordRetryCount` to `0` fails 2 rows.

The orchestrator additionally audited every `sendEvent`/`emit` call site in `stream.go` (`:648, 702, 805, 812, 836, 1051, 1071`) and confirmed each either honors its result or has `outcome.failure` already assigned before it — the defect *shape*, not just its two instances, is closed. Round 2's own 11-path enumeration reaches the same conclusion independently.

### WARNING (4)

**W-A — `S-AGM-020`'s restated text is FALSE at HEAD, and the archive step will promote it into canon.**
`go.work:1` declares `go 1.26.5`. AI-37's own delta restates the scenario at `specs/agent-module-scaffold/spec.md:57` as "it declares `go 1.26.3`". No test asserts the `go.work` directive (`grep` for it in `backend/agent/src/` returns nothing), so nothing catches the divergence. The historical report recorded this as SUGGESTION 4 ("pre-existing, not a blocker") and still counted `agent-module-scaffold` 23/23 COMPLIANT. **I am reversing that grade to PARTIAL**, because AI-37's own delta restates the false sentence and this change's spec discipline #5 exists precisely so that restated scenarios survive the archive step — which means archiving re-affirms the falsehood into the canonical spec. **Still not blocking**: the file is byte-unchanged by AI-37, the falsehood is pre-existing and identical in the canonical spec, and `go.mod`'s own `go 1.26.3` (which `S-AGM-001` asserts) IS true. Fix is a one-token edit in the delta before archive, or a follow-up owned by `agent-module-scaffold`.

**W-B — `S-AOB-038` says "Layer 1 sources"; the shipped guard scans one package.**
`ai37_noop_equivalence_test.go:362` scans `os.ReadDir(".")` — the `openaicompat` package directory only, 24 non-test files. Layer 1 (`backend/agent/src/ai/**`) holds ~71 non-test files across `src/ai` (30), `src/ai/openaicompat` (24), `src/ai/openaicompat/openrouter` (5), `src/ai/internal/retry` (2) and others. The gap that matters: `openrouter/wrapper.go` now holds a `trace.TracerProvider` field and is **not** scanned, so a nil check added there would evade `S-AOB-038` entirely. Round 1 widened the scan from 3 hardcoded files to the package, and the test's own doc comment (`:355`) admits it "narrowed [the header comment] to match" — i.e. the comment was aligned to the code, while the **scenario text was left unamended**. **Non-blocking**: I independently verified the substantive claim over all of Layer 1 and it holds — zero hits for any of the four patterns in any non-test file under `src/ai/`. The guard is narrower than the claim; the claim is true. Either widen the walk to `src/ai/...` or narrow `S-AOB-038` to "this package" before archive.

**W-C — the `*(review)*` mislabel has TWO instances, not one; round 2 disclosed only the first.**
This repo's documented convention (canonical `ai-provider-client/spec.md:8,51`) is explicit: `*(review)*` marks a scenario "discharged by reading landed prose, a diff or a document rather than by executing code… It carries no red phase and is therefore outside `NFR-APC-G`'s red→green evidence rule."
- `S-AOB-040` (`specs/ai-observability-boundary/spec.md:142`) carries `*(review)*` but is backed by a running test, `TestAI37_Attributes_ContentTypeRefusalRecordsExactStatusCode`, asserting at `ai37_attributes_test.go:380`. Round 2 disclosed this as a probable round-1 mislabel and deliberately left it (out of assigned scope).
- **`S-APC-085`** (`specs/ai-provider-client/spec.md:91`) carries the **same** `*(review)*` tag and is **also** test-backed — `TestNewProvider_InjectedTracerProviderRecordsTheSpan`, `openrouter/tracer_test.go:23`, PASS. **Round 2 did not notice this second instance.**
Both tags under-claim: they exempt from the red→green evidence rule two scenarios that actually have it, so a future maintainer could delete either test without tripping the convention. Contrast `S-AOB-024` — genuinely `*(review)*`, correctly tagged. **Non-blocking**: both scenarios ARE proven; only the tag is wrong. Drop `*(review)*` from `S-AOB-040` and `S-APC-085` before archive.

**W-D — the spec index no longer agrees with its own delta files (4 drifts + 1 in a delta Identity table).**
Full audit in Part 3. `spec.md:14` still says `S-AOB-001 … S-AOB-039`; `spec.md:16` still says `S-APC-081 … S-APC-084`; `spec.md:18` still says "49 new scenarios (39 + 6 + 4)" and "29 landed scenarios restated" and "14 requirements touched" against a delta reality of **52 new + 28 restated = 80 scenarios and 16 touched requirements (18 assessed, including 2 NFRs)**; `specs/ai-provider-client/spec.md:17`'s Identity row omits `S-APC-085`. Two of these predate the correction rounds (the historical report's SUGGESTION 5, never fixed); three were introduced by rounds 1 and 2. Round 2 correctly fixed the two range references inside the `ai-observability-boundary` delta itself. **Non-blocking**: purely documentary, no behavioural claim depends on it. But these are the artifacts the archive step promotes, so fix before archive.

### SUGGESTION (5)

**S-1** — `S-AOB-029`'s failure message (`ai37_denylist_test.go:270`) should name the span and the attribute key alongside the vector, as `R-AOB-007` requires. `sweep.Scan` currently returns no location and `Corpus()` is flattened, so this needs the sweep to carry a location tuple. Only observable when a leak already exists.

**S-2** — `S-AOB-003`: state the closure reason (*root otel's package closure contains `go.opentelemetry.io/auto/sdk`*) at the declination site `import_boundary_test.go:99-107`, not only the injection reason. The reader is sent there by the scenario and finds a different rationale.

**S-3** — The nil-check scan (`ai37_noop_equivalence_test.go:286-291`) is a four-literal substring table; `if s := span; s == nil` evades it. A lightweight AST walk tracking identifier bindings, or a broader token-pattern set, would close it. Deferred by the maintainer as the last permitted round; the substantive claim holds today.

**S-4** — `R-AOB-006` bullet 1 (`spec.md:129`) justifies the omission as "the retry mechanism itself fails **before any response is ever obtained**". On the budget-exhausted row, responses genuinely *were* obtained (three 503s) — they are just not returned: `retry.Loop` returns `nil` for the response on every error exit (`retry.go:92,95,98,112,116`). The operative rule ("MUST be omitted on that path") is unambiguous and correctly implemented, and the code-site comment at `trace.go:115-119` states it accurately ("has no `*http.Response` at all on this exit"). Only the spec's prose rationale is loose. Tighten to "no response value survives to the recording site" at archive.

**S-5** — `apply-progress.md:386` records `34 files / 6000 / 163`, exact as of `4bf8e90a`; at `0b1fbd61` the true stat is `35 files / 6495 / 163`, and it moves again with every orchestrator-owned artifact commit. Consider stating the stat as "as of `<sha>`, excluding orchestrator-owned verify artifacts" rather than chasing a self-referential number.

**S-6** — Revisit `emitFailureSendBound = 5s` (`stream.go:147`). The `[DONE]` recovery can spend ~10s before `close(out)`, and `TestAI37_TerminalCompletionSend_LosingTheRaceRecordsFailure` is now the module's slowest test at 5.01s. The decision to keep the sends is correct (Part 2b); the *bound* is a separate, revisitable knob.

*(`runNoPanic`'s single-goroutine `recover()` — the historical report's SUGGESTION 6 — remains open and remains correctly skipped; a producer-goroutine panic still fails the package loudly, just not by row name.)*

---

## Verdict

# **FAIL — on the completeness gate only. No CRITICAL. No defect.**

**0 CRITICAL · 0 blockers · 4 WARNING · 5 SUGGESTION** · 72/80 scenarios COMPLIANT, **8 PARTIAL**, 0 UNTESTED, 0 FAILING · 12/18 requirements fully complete · 21/21 tasks · `make test` / `make lint` / `make build` all exit 0 under `-race` with a clean tree.

**Read this verdict precisely — it is not the earlier FAIL.** The historical `verify-report.md` failed on a genuine defect: `retry.count` was recorded on every exit and asserted on none, and two whole-module overlay runs sabotaged it without a single test failing. That defect is gone, and so are both Judgment Day CRITICALs. **Nothing in the code is wrong.** This report fails admission for one reason: `gentle-ai sdd-verify-validate` requires a passing verdict to carry complete requirement *and* scenario counts, and eight scenarios are PARTIAL:

```text
gentle-ai sdd-verify-validate --input <this file> --requirements 18 --scenarios 80
# verdict: pass              → "admission denied: passing verdict contradicts failing or incomplete evidence"
# verdict: pass_with_warnings → same denial
# verdict: fail              → {"valid": true, "verdict": "fail"}
```

I declined to report 80/80 to obtain a passing envelope. Eight scenarios genuinely lack a fully covering passing test, and one of them — `S-AGM-020` — restates a sentence that is **factually false** against the shipped `go.work`. Inflating the counts would have contradicted this report's own compliance matrix, which is exactly the count-mismatched evidence the admission rule exists to reject.

**What was proven right.** The historical CRITICAL and both Judgment Day CRITICALs are fixed, and every fix is falsifiable — proven by `go test -overlay` runs that fail against the pre-fix source without ever mutating the worktree. Round 2's decisions are not merely implemented but **correct**: widening `R-AOB-006`'s code to match its general clause holds because `finalizeSpan`'s status code is genuinely in hand on all 11 post-handover paths (single call site at `stream.go:640`, single launch site at `:250`, both guarded by `:224-246`); keeping the recovery sends holds because `ai.CheckStream` verifiably does not check contiguity (`stream_check.go:52-54`) while dropping them would regress the landed `S-AEM-051/052`. The OpenRouter door is threaded verbatim, defaults to the API's own no-op, and opens no ambient-authority or content path.

# **ARCHIVABLE: NO — pending four documentation-only edits, zero code changes**

The change is behaviourally sound and delivery-ready. It is not yet *archivable*, because the archive step promotes these delta specs into the canonical spec set, and three of them would carry an untrue or over-broad statement into canon:

| # | Edit | Effect | File |
|---|---|---|---|
| 1 | `go 1.26.3` → `go 1.26.5` in the restated `S-AGM-020` (W-A) | `S-AGM-020` PARTIAL → COMPLIANT; `R-AGM-003` complete | `specs/agent-module-scaffold/spec.md:57` |
| 2 | Narrow `S-AOB-038`'s "Layer 1 sources" to "this package", **or** widen the scan's walk to `src/ai/...` (W-B) | `S-AOB-038` PARTIAL → COMPLIANT; `R-AOB-009` complete | `specs/ai-observability-boundary/spec.md:194` or `ai37_noop_equivalence_test.go:362` |
| 3 | Drop the `*(review)*` tag from `S-AOB-040` **and** `S-APC-085` — both are test-backed (W-C) | Convention accuracy; no count change | `specs/ai-observability-boundary/spec.md:142`, `specs/ai-provider-client/spec.md:91` |
| 4 | Fix the index arithmetic and the two stale scenario ranges (W-D) | Index agrees with the deltas; no count change | `spec.md:14,16,18`, `specs/ai-provider-client/spec.md:17` |

Edits 1 and 2 move the counts to **14/18 requirements, 74/80 scenarios**. The remaining six PARTIALs are structural, not fixable by editing: `S-AOB-009`'s duration clause needs timestamps `tracetest.Span` does not record; `S-AOB-027`, `S-AOB-028` and `S-APC-083` are satisfied-by-construction and unconstructible once the root global getter is excluded; `S-AOB-003` and `S-AOB-029` need prose and failure-message work (SUGGESTIONs S-2 and S-1). If the maintainer accepts those six as satisfied-by-construction and amends their scenario text accordingly, the counts reach 18/18 and 80/80 and this report re-runs as `pass_with_warnings`.

**Next phase: `sdd-apply`, doc-only scope — the four edits above, no Go source, no new tests.** This is emphatically **not** grounds for a third Judgment Day round: two rounds are complete, the last permitted correction round has been consumed, no adversarial finding remains open, and nothing here is a code defect.
