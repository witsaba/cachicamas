# Tasks: Enforce secret redaction (AI-36) — `cachicamas-ai-redaction`

> Design: Engram `sdd/cachicamas-ai-redaction/design` (id 2684) · Spec: Engram `sdd/cachicamas-ai-redaction/spec` (id 2682) · Strict TDD active. All paths under `backend/agent/src/`.

## Verification note (discrepancy recorded, not silently fixed)

The spec index and the launch brief both state "22 binding scenarios." Direct enumeration of the five binding scenario ranges (`S-CNF-077…081`=5, `S-APC-077…080`=4, `S-ART-090…094`=5, `S-STK-047…049`=3, `S-AIP-058…061`=4) sums to **21**, not 22. The traceability table below covers all 21 actual scenario IDs; the "22" total in upstream artifacts is a one-off arithmetic error, carried here as a flagged discrepancy rather than silently corrected in those artifacts.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1,310 (prod ~170–190 incl. contingent, test ~960–980, spec prose ~150) |
| 400-line budget risk | High (raised budget) |
| Chained PRs recommended | No — `size:exception` pre-granted for this milestone |
| Suggested split | Single PR; WU-1…WU-7 as internal chronological commits |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

Budget: **1600** lines (maintainer-raised exception for AI-36). Central estimate ~1,310, headroom ~290 (absorbs an AI-35-scale 168-line overrun). `size:exception` is pre-granted — no apply-time decision needed.

**Pre-approved trim levers (carried forward verbatim, apply may invoke without re-opening a decision):**
(a) collapse D-3's marker/allowlist to external-only scope (−150, least preferred — re-opens the disclosed blind spot);
(b) leave the smoke scanner duplicated instead of converging on `sweep` (−80, weakens S-CNF-080);
(c) drop the `Client` redaction fix if the WU-3 D-5 empirical run comes back green (−12, free).

### Per-WU line estimate

| WU | Prod | Test | Sub |
|---|---|---|---|
| WU-1 shared sweep helper | ~90 | ~180 | ~270 |
| WU-2 adversarial sweep + hostile server (D-4) | 0–10 | ~280 | ~290 |
| WU-3 config/client/header proofs (D-5) | 0–12 | ~190 | ~200 |
| WU-4 bounded-summary hygiene | 0 | ~90 | ~90 |
| WU-5 recursive scan + falsifiable allowlist | ~70 | ~160 | ~230 |
| WU-6 copied-value guard (D-1) | ~8 | ~70 | ~80 |
| WU-7 spec landing + gates | — | — | ~150 |
| **Total** | **~170–190** | **~960–980** | **~1,310** |

### Suggested Work Units (internal commits within the single PR)

| Unit | Goal | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|
| WU-1 | One shared `sweep` package; smoke + conformance converge | `go test ./agenttest/... -run Sweep` | N/A — pure unit tests, no live harness | `agenttest/sweep/`, revert 2 delegation diffs |
| WU-2 | Adversarial hostile-server sweep, D-4 resolved | `go test ./ai/openaicompat/... -run AI36_1` | N/A — httptest stub only | `a_i-36_1_test.go` + contingent `stream.go` hunk |
| WU-3 | Config/client/header redaction proofs, D-5 resolved | `go test ./ai/openaicompat/... -run AI36_2` | N/A | `a_i-36_2_test.go` + contingent `client.go` hunk |
| WU-4 | Failure-output hygiene proof | `go test ./agenttest/... -run DivergenceReport` | N/A | `stream_kit_diff_test.go` additions |
| WU-5 | Recursive scan + falsifiable allowlist | `go test ./ai/openaicompat/... -run CredentialScan` | N/A | `credential_scan_test.go` + 6 marker comments |
| WU-6 | Copied-value structural guard | `go test ./ai/... -run Failure_CopiedValue` | N/A | `provider_failure_test.go` additions + doc-comment |
| WU-7 | Spec landing + full gates | `cd backend/agent && make test` | `make lint && make build` | Spec delta status lines only |

---

## Scenario → Task Traceability (21/21 total)

| Scenario | Task(s) |
|---|---|
| S-CNF-077 | 2.2 |
| S-CNF-078 | 2.2 |
| S-CNF-079 | 2.1 (RED), 2.2 (GREEN) |
| S-CNF-080 | 1.3, 1.4, 1.5 |
| S-CNF-081 | 2.3, 2.4a/2.4b, 2.5 |
| S-APC-077 | 3.1 |
| S-APC-078 | 3.2, 3.3a/3.3b |
| S-APC-079 | 3.4, 3.5 |
| S-APC-080 | 3.6 |
| S-ART-090 | 5.1, 5.2 |
| S-ART-091 | 5.4, 5.5 |
| S-ART-092 | 5.6, 5.7, 5.8, 5.9 |
| S-ART-093 | 5.10 |
| S-ART-094 | 5.11 |
| S-STK-047 | 4.1 |
| S-STK-048 | 4.3 |
| S-STK-049 | 4.4 |
| S-AIP-058 | 6.1 |
| S-AIP-059 | 6.2 |
| S-AIP-060 | 6.3, 6.4 |
| S-AIP-061 | 6.6 |
| S-AEM-071 (contingent) | 2.4a/2.4b, 2.5 |
| S-AEM-072 (contingent) | 2.4a/2.4b, 2.5 |

Totality check: every one of the 21 binding scenario IDs listed in the spec deltas has ≥1 task above. No gaps.

## Positive-Control Pairing (absence assertion ↔ its control)

| Absence task | Control task | Mechanism |
|---|---|---|
| 2.2 (sweep absence, S-CNF-077/078) | 2.1 (`sweep.SelfTest` call before every scan, AD-2) | Self-assembled needle must bite before the real corpus is trusted |
| 2.3 (hostile-server absence, S-CNF-081) | 2.3 itself (RED failure = detection proof) | RED phase is the control: the sweep must first show it can catch the leak |
| 3.1 (config absence, S-APC-077) | inline: unredacted stand-in carrying same sentinel MUST fail | Per scenario text — control lives in the same test |
| 3.4 (header absence, S-APC-079) | inline: deliberately leaking rendering MUST fail | Per scenario text |
| 4.1 (divergence-report absence, S-STK-047) | 4.3 (deliberately leaking report MUST fail) | Separate task, same file |
| 5.1/5.2 (recursive-scan absence) | 5.8/5.9 (allowlist falsifiability) + 5.4 (marker-removed re-detection) | Two independent controls |
| 6.1 (mechanism pin — proves the leak is real) | 6.2 (pointer-shaped contrast — proves vacuous placement fails to surface) | Depth-hazard control (AD-3) |
| 6.3 (AST guard absence, S-AIP-060) | 6.4 (synthetic `func Leak() ai.Failure` MUST be flagged) | Structural positive control |

---

## WU-1: Shared sweep helper (`agenttest/sweep`)

Requirement basis: S-CNF-080 (one implementation, both consumers reach it, names unchanged).

- [x] 1.1 RED — `backend/agent/src/agenttest/sweep/sweep_test.go`: `TestScan_DetectsPlantedNeedle`, `TestScan_ReturnsFalseWhenAbsent`, `TestSelfTest_FailsNamingVectorOnBrokenDenyList` (+ `TestSelfTest_PassesForALiveDenyList` triangulation). Package does not exist yet — compile failure is the RED (confirmed: "no non-test Go files"). (test ~90)
- [x] 1.2 GREEN — `backend/agent/src/agenttest/sweep/sweep.go`: `type Entry struct{ Vector string; Needle []byte }`; `func Scan(corpus []byte, deny []Entry) (string, bool)`; `func SelfTest(deny []Entry) error` per AD-1/AD-2. Imports `bytes`+`fmt` only. (prod ~45)
- [x] 1.3 GREEN (behavior-preserving refactor) — `backend/agent/src/ai/openaicompat/openrouter/smoke/sentinel_sweep.go`: `type DenyEntry = sweep.Entry` (alias); `Scan`/`BuildDenyList` delegate internally to `sweep.Scan`, signatures unchanged. Reran the 8 existing R-OR-08 smoke tests — green (regression half of S-CNF-080). (prod ~20)
- [x] 1.4 GREEN (behavior-preserving refactor) — `backend/agent/src/agenttest/conformance_redaction.go`: `scanForSentinel`/`scanTextForSentinel` keep names + event-walking, delegate substring core to `sweep.Scan`. Reran the existing conformance-redaction case — green. (prod ~15)
- [x] 1.5 Regression pin, not RED-first — `backend/agent/src/agenttest/sweep_convergence_test.go` (package `agenttest`, internal test): same corpus+needle through `scanTextForSentinel` and `openrouter/smoke.Scan`, assert identical found outcome — discharges S-CNF-080's "both reach it" clause behaviorally. GREEN on first run as labeled. (test ~40)

## WU-2: Adversarial sweep + hostile server (D-4)

Requirement basis: R-CNF-027 (S-CNF-077…081), contingent R-AEM-019 (S-AEM-071/072).

- [x] 2.1 RED — `backend/agent/src/agenttest/conformance_redaction.go`: widened `redactionCase` to call two not-yet-defined helpers (`redactionSweepAllCategories`, `redactionSweepDivergenceReport`) — confirmed compile-failure RED (`undefined: redactionSweepAllCategories` / `undefined: redactionSweepDivergenceReport`). (test ~40)
- [x] 2.2 GREEN — implemented the widened sweep: `redactionSweepAllCategories` loops every `ai.FailureCategories()` member, calls `sweep.SelfTest` first for both sentinels, and `scanForSentinel` itself now also sweeps `%v`/`%+v`/`%#v` on the event and on the error payload (event metadata + verbose/Go-syntax renderings). Two distinct sentinels (credential stand-in + content-body stand-in) folded into Cause/RequestID. Discharges S-CNF-077/078/079. (prod ~10, test ~30)
- [x] 2.3 RED — `backend/agent/src/ai/openaicompat/a_i-36_1_test.go` (package `openaicompat_test`): hostile `httptest.Server` — 200 + `Content-Type: application/json`, body echoes Authorization header + request body — forces `refuseNonStreamContentType`. Confirmed genuine RED: sentinel credential surfaced in `unwrap-depth-1 .Error()`/`%v`/`%s`/`%+v`. D-4 empirical decider fired: **credential bites**. (test ~120)
- [x] 2.4a GREEN — **branch taken: credential vector bites.** `stream.go`: `refuseNonStreamContentType(resp, cred)` now redacts via new `redactCredential`/`credentialRedactedPlaceholder` (`bytes.ReplaceAll` of `cred.bearer()` then `cred.token`, both unexported). `R-AEM-019` delta promoted to LANDED in `specs/ai-provider-error-mapping/spec.md` with the empirical evidence recorded in the file. Test re-run GREEN. (prod ~24 incl. doc comments, ~10 excl.; test ~120 shared with 2.3)
- [x] 2.4b — **branch not taken** (2.4a fired instead; recorded here per the "must not drop silently" rule — not-triggered disposition never applied).
- [x] 2.5 GREEN — prompt-body echo recorded as a **named residual**: confirmed present on the same run (`t.Logf` in the test), documented in `stream.go`'s `nonStreamContentType` doc comment and in the spec delta's resolution note. (doc, no budget impact)

> D-4 resolved: exactly 2.4a landed; the contingent requirement was not dropped — its spec file's header/table were updated in place with the empirical evidence.

## WU-3: Config/client/header redaction proofs (D-5, D-6)

Requirement basis: R-APC-015 (S-APC-077…080).

- [x] 3.1 Pin test, expected GREEN on first run — `backend/agent/src/ai/openaicompat/a_i-36_2_test.go` (package `openaicompat_test`): `TestConfig_NeverRendersCredential` — verb-exhaustive (`%v %s %+v %#v` + `json.Marshal`) over a `Config` built with a sentinel credential, plus an unredacted stand-in inline control (S-APC-077). Confirmed GREEN on first run as labeled. (test ~50)
- [x] 3.2 RED — same file: `TestClient_BareUnwrapped_NeverRendersCredential` (S-APC-078). Confirmed genuine RED: bare `*Client`/`Client` leaked the raw token via `%v`/`%s`/`%+v`/`%#v` (both pointer and value forms) — `reflect.Value.CanInterface()` false on the unexported field, raw reflection fell through. D-5 empirical decider fired: **bites**. (test ~60)
- [x] 3.3a GREEN — **branch taken: 3.2 is RED.** `client.go`: added value-receiver `String()`/`GoString()` on `Client`, fixed label `"<openaicompat client>"`, mirroring `wrapper.go:90-101`. Pre-check done: `provider_boundary_test.go:28` + `stream_test.go:259` only assert `MethodByName("Stream")`, no method-count pin — CLEAR. Test re-run GREEN. (prod ~24 incl. doc comments, ~12 excl.)
- [x] 3.3b — **branch not taken** (3.3a fired instead).
- [x] 3.4 RED — `backend/agent/src/ai/openaicompat/retry_metadata_test.go`: `TestCaptureRateLimitTelemetry_NeverReproducesHeaderNameOrValue` (S-APC-079), via the compile-failure RED trick on the shared `headerCaptureLeak` helper — confirmed (`undefined: headerCaptureLeak` ×4). GREEN after implementing the helper. Inline positive control passes. (test ~40)
- [x] 3.5 GREEN — confirmed pin: the 3-name allowlist at `retry_metadata.go:69-84` already excludes the full header set; no allowlist-logic change needed. (0 prod, test 0)
- [x] 3.6 GREEN (via the same compile-failure RED as 3.4) — `TestHeaderDiagnostics_NoneCaptureWholeHeaderSet` (S-APC-080): allowlist-size structural pin (3 entries) + "newly present header stays unadmitted" case. (test ~40)

## WU-4: Bounded-summary hygiene

Requirement basis: R-STK-014 (S-STK-047…049).

- [x] 4.1 RED (label) → **empirically GREEN on first run** — `backend/agent/src/agenttest/stream_kit_diff_test.go`: `TestRequireSameEvents_DivergenceReport_SentinelFree` (S-STK-047), using `fakeTB` (the double actually accessible from package `agenttest_test`; `capturingTB` is unexported and package-`agenttest`-only — a necessary correction from the task's literal wording). Over-bound (>32-rune) sentinels via TextDelta + ToolCallDelta fragments. Passed immediately: `boundedFragment`'s existing rune cap already discharges "full sentinel never reconstructible" for free-form fragment fields — matches 4.2's own prediction. Honored as a legitimate empirical GREEN, not a manufactured RED (see apply-progress deviation D-1). (test ~70, both sub-cases)
- [x] 4.2 GREEN — **real gap found and fixed**, contradicting the "0 prod lines" prediction: task 4.4's table-driven run found `ResponseStart`'s id/model and `ToolCallStart`'s id/name render WHOLE (unbounded) in `summaryTable`, so an over-bound sentinel THERE leaked in full. Fixed both `summaryTable` entries in `stream_kit_diff.go` to use `boundedFragment`. (prod ~10)
- [x] 4.3 RED (label) → **empirically GREEN on first run** — `TestRequireSameEvents_DivergenceReport_PositiveControl` (S-STK-048): an UNDER-bound sentinel (≤32 runes) IS reproduced in full by the same mechanism, proving 4.1's absence claim is falsifiable. Passed immediately (boundedFragment only bounds length, never redacts content, by design). (test ~30)
- [x] 4.4 RED → GREEN — `TestRequireSameEvents_BoundHoldsForEveryEventKind` (S-STK-049): table-driven across 10 plantable fields spanning 8 kinds. Confirmed genuine RED: `ResponseStart/id`, `ResponseStart/model`, `ToolCallStart/id`, `ToolCallStart/name` all reconstructed the full over-bound sentinel. Fixed via 4.2; all 10 sub-cases GREEN after the fix, including regression check on `TestRequireSameEvents_EveryRegisteredKind_UsesASpecificSummaryNotTheGenericFallback`. (test ~90)

## WU-5: Recursive scan + falsifiable allowlist (D-3)

Requirement basis: R-ART-022 (S-ART-090…094).

- [x] 5.1 RED — `backend/agent/src/ai/openaicompat/credential_scan_test.go`: replaced `os.ReadDir`-based single-directory scan with recursive `filepath.WalkDir`; deleted `externalTestPackageClause`. `TestCredentialScan_RecursiveWalk_FindsPlantInANestedDirectory` proves S-ART-090. RED captured via the compile-failure trick (whole implementation stripped, `undefined: walkCredentialSurface` × several + others — confirmed). (test ~50)
- [x] 5.2 GREEN — implemented the recursive walk + marker-based exemption (`walkCredentialSurface`, marker `"credential-scan:" + "deliberate-plant"` assembled at runtime). (prod ~55)
- [x] 5.3 GREEN (same commit as 5.2) — added the marker comment to each of the 6 inventoried deliberate-plant files (file-level, one marker per file since exemption is file-scoped): `credential_test.go`, `capture_proof_test.go`, `viability_test.go`, `openrouter/credential_redaction_test.go`, `openrouter/smoke/smoke_test.go`, `openrouter/smoke/sentinel_sweep_test.go`. Also fixed 3 contiguous-literal sentinels accidentally introduced by WU-2/WU-3's own new tests (`a_i-36_2_test.go` ×2, `retry_metadata_test.go` ×1) to runtime-assembled form so they don't join the allowlist. (prod ~24 marker lines + 3 fix lines)
- [x] 5.4 RED — `TestCredentialScan_DeclarationRemoved_IsReported` (S-ART-091): synthetic temp-dir fixture, marker+allowlisted → not reported; identical content, marker removed → reported. Part of the same compile-failure RED capture as 5.1. (test ~35)
- [x] 5.5 GREEN — confirmed: 5.2's exemption logic (`marked && allowlistContains`) satisfies 5.4 with no additional prod code. (0 prod)
- [x] 5.6 RED — `TestCredentialScan_AllowlistMatchesEnumeratedList` (S-ART-092): `markedFiles` + `allowlistMismatch` compare the real tree's marked-file set against the committed allowlist in both directions, with an inline falsifiable control. Part of the same RED capture (`undefined: markedFiles`/`allowlistMismatch`). (test ~30)
- [x] 5.7 GREEN — implemented `deliberatePlantAllowlist` (6 relative paths, `filepath.Join` for multi-segment entries) + `allowlistMismatch` set-comparison. (prod ~10)
- [x] 5.8 RED — `TestCredentialScan_AllowlistFalsifiability`: verifies every real allowlist entry exists, carries the marker, AND still matches a credential pattern; synthetic "cleaned up" and "missing file" cases both fail as required. Part of the same RED capture (`undefined: verifyAllowlistFalsifiability`). (test ~30)
- [x] 5.9 GREEN — implemented `verifyAllowlistFalsifiability` (exists + marker + pattern-match, all three required). (prod ~18)
- [x] 5.10 RED then GREEN — `TestCredentialScan_NoReprintOfMatchedValue` (S-ART-093): `scanResult{relPath, class}` never carries matched bytes; self-scan of the real tree (scanner's own sources, declaration text, committed list) reports 0 violations. (test ~25)
- [x] 5.11 Verification-only — re-ran S-ART-013…017: `TestCredentialScan_ExpectationSurfaceIsClean`, `FailsOnASentinelInAnExpectationLiteral`, `FailsOnASentinelInTestSetup`, `HostsAndModelIdentifiersStayGreen` unchanged and green; `TestCredentialScan_IgnoresInternalTestPackageFiles` renamed+inverted to `TestCredentialScan_InternalPackageFileWithoutMarker_IsSwept` per design.md AD-4 item 4 — all 5 pass under the widened rule (S-ART-094). (0 lines, gate)

## WU-6: Copied-value structural guard (D-1)

Requirement basis: R-AIP-017 (S-AIP-058…061).

- [x] 6.1 Mechanism pin (expected GREEN) — `TestFailure_CopiedValue_ValueShapedCause_RendersCanary` (S-AIP-058): value-shaped `valueShapedCause`, `copied := *failure`, `%#v`/`%v`/`%+v` on `copied` all DO contain the canary. Confirmed GREEN as labeled — demonstrates the real, known, acceptable leak the discharge scopes around. (test ~45)
- [x] 6.2 Depth-hazard contrast pin (expected GREEN) — `TestFailure_CopiedValue_PointerShapedCause_DoesNotSurfaceCanary` (S-AIP-059): identical construction, pointer-shaped cause (`errors.New`), canary absent under all 3 verbs. Confirmed GREEN as labeled — proves 6.1's placement is non-vacuous. (test ~30)
- [x] 6.3 RED — `TestNoPublishedSurfaceReturnsFailureByValue` (S-AIP-060): AST escape guard referencing not-yet-defined `scanModuleForByValueFailure`/`findingsInFile`. RED captured via the compile-failure trick (whole implementation stripped, `undefined: scanModuleForByValueFailure` + `undefined: findingsInFile` confirmed). (test ~20)
- [x] 6.4 GREEN — implemented the guard (`go/parser` walk from `".."` — this module's `src/` tree, both `src/ai` and `src/agenttest` — flagging any exported func/method signature or exported struct field carrying `Failure`/`ai.Failure` by value); current tree clean (0 findings). Positive control (`TestNoPublishedSurfaceReturnsFailureByValue_PositiveControl`, in-memory synthetic `func Leak() ai.Failure`) correctly flagged. (prod ~65, test ~25)
- [x] 6.5 GREEN (doc only) — added the "Pinned by AI-36" scope-statement paragraph to the `GoString` doc comment at `provider_failure.go` (naming both new tests as the load-bearing proof). No method added — `S-AIP-008`'s guard untouched (re-verified: still only exempts `GoString`). (0 code lines)
- [x] 6.6 Regression pin — `TestFailure_PublishedRenderingSetUnchanged_AbsentPayloadTotality` (S-AIP-061): published rendering set (`Error`, `GoString`, no `String`) unchanged from R-AIP-016; absent-payload (nil `*Failure`) rendering under all 4 verbs matches `Error()`, no panic (`NFR-AIP-B`). (test ~25)

## WU-7: Spec landing + reflection re-check + verification gates

- [x] 7.1 Resolved: `specs/ai-provider-error-mapping/spec.md` promoted to **LANDED** (2.4a fired) with the empirical evidence recorded in the file's header and trigger table.
- [x] 7.2 Reflection blast-radius re-check against FINAL code (re-run, not trusted from design-time prediction): `sweep.Entry`/`Scan`/`SelfTest` (new package, nothing can reference it — CLEAR), `Client.String`/`GoString` (only `MethodByName("Stream")` checks exist on `&Client{}`, no method-count pin — CLEAR), `smoke.DenyEntry` alias (zero `%T`/`reflect.` under `smoke/` — CLEAR). Every new test-local helper is unexported. Full final-tree grep re-run: no new hits beyond the 4 already-catalogued `stream_test.go` signature-shape checks. All CLEAR.
- [x] 7.3 Verification gate — `cd backend/agent && make test` — see apply-progress for full output.
- [x] 7.4 Verification gate — `cd backend/agent && make lint` — see apply-progress for full output.
- [x] 7.5 Verification gate — `cd backend/agent && make build` — see apply-progress for full output.
- [x] 7.6 Verification gate — `git diff --stat origin/main -- backend/agent/go.mod backend/agent/go.sum` — see apply-progress for full output.
