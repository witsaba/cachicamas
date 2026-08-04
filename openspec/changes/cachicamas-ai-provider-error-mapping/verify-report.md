```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:bfa8f11768d27fab62841a8008cf7eea22549f8970a883a0c244269cc95941f2
verdict: fail
blockers: 0
critical_findings: 0
requirements: 11/18
scenarios: 45/70
test_command: go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:0661a73215170a5131cc6d150918c91ca27d8a39025fb9397d6856da1c4d0a02
build_command: make lint
build_exit_code: 0
build_output_hash: sha256:66d9a3373b26e70b4206ef2aab426698da81f1f718bd3e1c6bb58b06ca3eb38a
```

> **Envelope semantics — read this before routing on `verdict: fail`.**
> The YAML verdict describes the **whole change**, not stage 1. `gentle-ai sdd-verify-validate` admits only `pass` or `fail`, and it rejects `pass` whenever the requirement/scenario counts are incomplete. Seven requirements (`R-AEM-010`…`016`) and 25 scenarios remain unimplemented **by design**, because slices 2a/2b/2c are gated on AI-28.1 landing. The change as a whole is therefore correctly **not archive-ready**, and `fail` is the only truthful admissible verdict at that scope.
>
> **The stage-1 verdict is PASS WITH WARNINGS.** Zero CRITICAL findings. Nothing in stage 1 is broken. Do **not** read `verdict: fail` as a stage-1 regression, and do not route this to `sdd-apply` for repair — route it to slice 2a once the AI-28.1 gate opens.

# Stage-1 verify report

**Change**: `cachicamas-ai-provider-error-mapping` (AI-32)
**Version**: spec rev 2 · design rev 2 · proposal rev 2
**Mode**: Strict TDD
**Scope**: **STAGE 1 ONLY** — Phase 0 + Phase 1. Requirements `R-AEM-001…009`, `017`, `018`; scenarios `S-AEM-001…039` + `064…066` (42 `[test]`) and `S-AEM-067…070` (4 `[inspection]`).
**Verdict basis**: this is a **stage-1 verdict**. Phases 2a/2b/2c are gated on AI-28.1 and are unimplemented **by design**; their absence is confirmed correct below and is **not** counted as a gap.
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-4-errors`
**Branch**: `feat/ai-32-1-status-taxonomy` @ `daddf1a` (`9dfd6d4` → `88ef99f` → `daddf1a`)

---

## Verdict

**PASS WITH WARNINGS** — every stage-1 `[test]` scenario is covered by a test that was observed passing at runtime, and four independent mutation probes prove the load-bearing assertions genuinely bite. Five warnings concern evidence-count accuracy, one conditionally-skipped assertion block, guard-test provenance, fixture-location convention, and reviewer workload. None blocks archive of stage 1.

---

## Completeness

| Metric | Value |
|--------|-------|
| Tasks total (whole change) | 43 |
| Tasks complete | 20 (Phase 0: 1/1, Phase 1: 19/19) |
| Tasks incomplete | 23 — **all** are gated stage-2 tasks (`2a.*`, `2b.*`, `2c.*`) |
| Stage-1 tasks incomplete | **0** |

Verified mechanically: `grep -oE '^- \[ \] [0-9a-c.]+' tasks.md | grep -vE '^2[abc]\.'` returns nothing — no unchecked task lies outside the gated stage-2 phases.

---

## Build & Tests Execution

**Build / lint**: PASS

```text
$ make lint            # from backend/agent
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
exit 0
```

**Tests**: PASS — two consecutive fresh runs, both exit 0.

```text
$ go test -race -count=1 ./...     # run 1
ok  github.com/cachicamas/backend/agent/src/agenttest        2.212s
ok  github.com/cachicamas/backend/agent/src/ai              3.658s
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat 4.104s

$ go test -race -count=1 ./...     # run 2
ok  github.com/cachicamas/backend/agent/src/agenttest        2.038s
ok  github.com/cachicamas/backend/agent/src/ai              3.530s
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat 3.873s
exit 0
```

**Auxiliary gates**

| Check | Command | Result |
|---|---|---|
| Formatting | `gofmt -l ./src/ai/openaicompat/` | empty — clean |
| Zero dependencies | `grep -c '^require' go.mod` | `0` |
| Vet | `go vet ./...` | exit 0, empty output |
| Cross-package drift | `git diff --stat feat/ai-32-0-base...HEAD` | 20 files, **only** `openaicompat/*` + `openspec/.../tasks.md` |

**Counted tallies (published only because counted here, not copied from the apply report)**

| Tally | Counted value |
|---|---|
| Top-level tests matching the documented slice-1 filter | **19** |
| Subtests under that filter | **37** |
| Total PASS lines under that filter | **56**, FAIL lines **0** |
| `openaicompat` top-level passing tests (whole package) | **156** |
| Repo-wide top-level passing tests | **604** |
| New top-level tests by prefix | `TestFailureMap_` 9, `TestRetryMetadata_` 4, `TestCharterBoundary_` 3, `TestCapture_` 3 |

**Coverage**: not measured — no coverage threshold is configured for this module. Not a failure.

---

## Non-vacuity proof: mutation spot-probes

Every probe was staged in the worktree, run, then reverted with `git checkout --`. `git status --porcelain` was confirmed **empty** after each probe and after all five.

| # | Mutation | Expected biter | Observed | Verdict |
|---|---|---|---|---|
| P1 | `classifyStatus`: 403 → `FailureCategoryAuthentication` | S-AEM-004 | `--- FAIL: TestFailureMap_StatusCategoryTable/403`; all 10 other rows PASS; exit 1 | Bites, and names the exact status |
| P2 | `parseRetryAfter`: HTTP-date branch made unparseable | S-AEM-032, S-AEM-033 | `--- FAIL: .../HTTP-date_in_the_future_(S-AEM-032)` **and** `.../never_negative_(S-AEM-033)`; the four delay-seconds subtests PASS; exit 1 | Both fixed-instant assertions bite |
| P3 | `parseRetryAfter`: negative-duration clamp deleted | S-AEM-033 only | `--- FAIL: .../(S-AEM-033)` alone; S-AEM-032 PASS; exit 1 | Clamp is independently load-bearing |
| P4 | Export `func AttemptCounter() int` in `failure_map.go` | S-AEM-064 | `charter_boundary_test.go:91: exported identifiers naming a forbidden charter concept (R-AEM-017): [failure_map.go:AttemptCounter (contains "attempt")]`; exit 1 | Charter scan bites and names file+identifier |
| P5 | `captureRateLimitTelemetry` disabled entirely (always returns `nil, false`) | S-AEM-036, 037, 038 | S-AEM-036 FAIL, S-AEM-037 FAIL, **S-AEM-038 PASS**, S-AEM-039 PASS | Partial — see **WARNING-2** |

---

## Vacuous-pass sweep (Engram obs #2471, nine shapes)

| # | Shape | Result |
|---|---|---|
| 1 | Fixture cannot distinguish implemented from not-implemented | **1 hit** — S-AEM-038 (WARNING-2). All other probed scenarios distinguish (P1–P4). |
| 2 | Implementation compared against itself | Clear — every expectation is a hard-coded literal (`30*time.Second`, `"provider failure: rate_limit"`, explicit category constants). |
| 3 | Corruption magnitude unobservable | Clear — S-AEM-029 uses 64×`A` + `Z`, so a truncating implementation yields 64 `A`s, distinguishable from the required empty string. |
| 4 | No meaningful content after the interesting position | Clear — same S-AEM-029 construction. |
| 5 | Misrouted-but-unobserved handling | N/A — no stream handling in stage 1. |
| 6 | Stale state coincidence | Clear — `mapErrorResponse` is pure; `observedAt` is a parameter, no package clock (design D3). |
| 7 | Sort coincidence | N/A — no ordering assertions. |
| 8 | Helper builds the same fixture regardless of case | Clear — `mustFixtureResponse` derives its path from `tc.status` via `fmt.Sprintf("openai_%d.http", status)`; all 11 rows load distinct files. |
| 9 | Staged mutation only disables a check | Avoided — every probe changed **production behavior**, never a test assertion. |

---

## TDD audit trail

The RED→GREEN cycle landed as a single commit (`88ef99f`), so git history is not bisectable to a RED state. The audit trail is therefore `tasks.md`'s recorded evidence, audited here against the shipped files.

**Task 1.11's recorded RED output is the consolidated compile failure.** Its `line:column` coordinates were re-derived against the current test files:

| Recorded RED error | Expected symbol at that coordinate | Observed | Result |
|---|---|---|---|
| `capture_test.go:52:15: undefined: mapResponse` | `mapResponse` | `mapResponse` | **MATCH** |
| `capture_test.go:57:11: undefined: capturedBody` | `capturedBody` | `capturedBody` | **MATCH** |
| `failure_map_test.go:100:15: undefined: mapResponse` | `mapResponse` | `mapResponse` | **MATCH** |

All three coordinates land byte-exactly on the named symbol in the shipped test files — strong corroboration that the recorded RED run executed against these exact files. Both named symbols are now defined in production: `failure_map.go:31 func mapResponse`, `capture.go:38 type capturedBody`.

**Per-task evidence audit (≥5 tasks):**

| Task | Recorded evidence | Audit result |
|---|---|---|
| 0.1 | Baseline 137 `openaicompat` / 585 repo-wide; `go.mod` 0 requires | Deltas re-counted: now 156 / 604 = **+19**, not the recorded +18 (WARNING-1). Zero-require confirmed `0`. |
| 1.11 | `undefined: mapResponse`, `undefined: capturedBody`, `[build failed]` | Coordinates MATCH exactly (table above). Genuine RED via missing symbols. |
| 1.12 | 11 fixtures, `http.ReadResponse` replay | Confirmed: 11 files present, statuses 400/401/403/404/408/422/429/500/502/503/504 — exactly R-AEM-002's table rows, no more, no less. |
| 1.13 | `captureBody` deliberately omits marker/drain; `truncationMarker` pinned by a test | Confirmed in `capture.go:66-70` and `capture_test.go:102-109`. Deviation is documented and scoped (see Coherence D7). |
| 1.14 | `classifyStatus` second bool = "in 100–599 range", not retryable; `retryableFor` shared | Confirmed `failure_map.go:105-134` and `:86-93`. Probe P1 proves the table bites. |
| 1.17 | "18 top-level `--- PASS` (9+4+3+**2**)" | **Off by one** — actual is 19 (9+4+3+**3**). `TestCapture_TruncationMarkerIsPinned` is the uncounted third (WARNING-1). |
| 1.19 | 155 `openaicompat` / 603 repo-wide; `make lint` 0 issues | Lint re-confirmed `0 issues.` Counts re-counted as 156 / 604 (WARNING-1). |

**Fixture byte-level audit (5 of 11 read in full):** `openai_400/401/403/429/503.http` each carry a correct status line, `Content-Type: application/json`, a blank separator line, and a wrapped `{"error":{type,message,param,code}}` body with `param`/`code` exercising JSON `null` — matching E1's cited shape and their scenarios' demands. All 11 status lines verified. All 11 are LF-terminated (see SUGGESTION-1).

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD evidence reported | Yes | `tasks.md` evidence notes + apply-progress obs #2487 |
| All tasks have tests | Yes | 4 test files cover all 20 stage-1 tasks |
| RED confirmed | Yes (with caveat) | Consolidated compile-failure RED; coordinates match exactly. `charter_boundary_test.go` is RED only by package-build association (WARNING-3) |
| GREEN confirmed | Yes | 19/19 new top-level tests pass on fresh execution |
| Triangulation adequate | Yes | Status table 11 rows; Retry-After 6 subtests; request-id 4 subtests; label opacity 5 subtests |
| Safety net | Yes | Task 0.1 captured a pre-change baseline; zero regressions across two fresh runs |
| Mutation-proved non-vacuity | 4 of 5 probes clean | P5 exposed WARNING-2 |

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit (pure/table, in-package) | 19 top-level / 37 subtests | 4 | `go test -race` |
| Integration | 0 | 0 | not applicable — no live network by design |
| E2E | 0 | 0 | not installed |

All four test files declare `package openaicompat` (internal), matching design.md's Testing Strategy table and the load-bearing `credential_scan_test.go` constraint.

### Assertion Quality

| File | Line | Assertion | Issue | Severity |
|------|------|-----------|-------|----------|
| `retry_metadata_test.go` | 101–108 | `if errors.As(failure, &telemetry) { … }` | Carrier assertions live inside a conditional; they silently skip when no carrier exists | WARNING |

No tautologies, no orphan empty-collection checks, no type-only assertions, no ghost loops, no smoke-only tests, no mock-heavy files (zero mocks in the change). `TestCapture_EmptyBodyRetainsExactlyZeroBytes` asserts an empty result but has an explicit companion non-empty test in the same file, satisfying the Empty Collection Rule.

---

## Spec Compliance Matrix — stage 1

| Requirement | Scenarios | Covering test | Result |
|---|---|---|---|
| R-AEM-001 | S-AEM-001, 002 | `failure_map_test.go > TestFailureMap_StatusCategoryTable` (11 subtests) | COMPLIANT |
| R-AEM-002 | S-AEM-003…010 | `TestFailureMap_StatusCategoryTable`, `TestFailureMap_NoRowProducesUnsupportedCapabilityOrZeroCategory`, `TestFailureMap_2xxProducesNoFailure` | COMPLIANT (P1-proved) |
| R-AEM-003 | S-AEM-011…013 | `TestFailureMap_StatusCategoryTable`, `TestFailureMap_RetryableIndependentOfRetryAfterHint` | COMPLIANT |
| R-AEM-004 | S-AEM-014…016 | `TestFailureMap_UnparseableOrAbsentBodyStillMaps`, `TestCapture_RetainedBytesReachableThroughErrorChain`, `TestCapture_EmptyBodyRetainsExactlyZeroBytes` | COMPLIANT |
| R-AEM-005 | S-AEM-017…020 | `TestFailureMap_UndocumentedStatusFallback` (4 subtests) | COMPLIANT |
| R-AEM-006 | S-AEM-021…025 | `TestFailureMap_VendorBodyToleranceAndLabelOpacity` (5 subtests) | COMPLIANT |
| R-AEM-007 | S-AEM-026…029 | `TestFailureMap_RequestIDBestEffort` (4 subtests) | COMPLIANT |
| R-AEM-008 | S-AEM-030…035 | `TestFailureMap_RetryAfter` (6 subtests) | COMPLIANT (P2/P3-proved) |
| R-AEM-009 | S-AEM-036, 037, 039 | `TestRetryMetadata_AllThreeHeadersIndividuallyAddressable`, `_PartialSubsetTolerated`, `_FailureErrorTextIsFixedAndClean` | COMPLIANT |
| R-AEM-009 | S-AEM-038 | `TestRetryMetadata_CredentialHeaderNeverCaptured` | **PARTIAL** — see WARNING-2 |
| R-AEM-017 | S-AEM-064 | `TestCharterBoundary_NoBackoffAttemptOrFailoverExported` | COMPLIANT (P4-proved) |
| R-AEM-017 | S-AEM-065 | `TestCharterBoundary_FailureCategoriesHasNineMembers` | COMPLIANT |
| R-AEM-017 | S-AEM-066 | `TestCharterBoundary_GoModHasZeroRequireLines` | COMPLIANT |
| R-AEM-017 | S-AEM-067 *[inspection]* | performed below | COMPLIANT |
| R-AEM-018 | S-AEM-068 *[inspection]* | performed below | COMPLIANT |
| R-AEM-018 | S-AEM-069 *[inspection]* | performed below | COMPLIANT |
| R-AEM-018 | S-AEM-070 *[inspection]* | performed below | **PARTIAL** — see WARNING-4 |

**Stage-1 compliance summary**: 42/42 `[test]` scenarios have a covering test observed passing at runtime; S-AEM-038 passes but is conditionally assertive. 3/4 in-scope `[inspection]` scenarios fully satisfied, 1 partial.

Authoritative spec counts, recounted mechanically: **18** requirements, **70** scenarios (**65** `[test]`, **5** `[inspection]` = 045, 067, 068, 069, 070). A naive grep reports 66 `[test]` because S-AEM-070's *prose* contains the token `` `[test]` ``; counting by kind-marker position gives the spec's declared 65. The spec is self-consistent.

### S-AEM-067 *[inspection]* — performed

Read `errors.go` directly. Both required passages are amended:

- `ErrFrameTooLarge` doc comment (`errors.go:49-61`): retains "AI-27 never constructs an `ai.Failure` … (S-ASD-064)" and adds "AI-32 owns turning a named category into a constructed, **wire-side** provider failure … AI-28 owns the producer path that calls it. AI-26.6's `policy.go` refuse remains a distinct, pre-existing construction site … and AI-32 neither invokes nor alters it (S-AEM-067)."
- `Category` doc comment (`errors.go:83-90`): retains "Category never constructs an `ai.Failure` value of any kind" and adds the same three-way distinction, citing S-AEM-067.

Both record AI-32 as the wire-side constructor, AI-28 as the producer path, and AI-26.6's translation-time refusal as distinct and untouched. **No promoted spec was edited** — confirmed by `git diff --name-only feat/ai-32-0-base...HEAD`, which touches no file under `openspec/specs/`. **SATISFIED.**

### S-AEM-068 / S-AEM-069 *[inspection]* — performed

- **S-AEM-068**: every wire claim in the spec carries an explicit `[dialect-conventional]` marker (R-AEM-002, R-AEM-007, R-AEM-008's presence clause, R-AEM-009, R-AEM-010) or a **cited** `E1…E6` reference. No unlabelled wire claim found. **SATISFIED.**
- **S-AEM-069**: all six references resolve to real headed evidence sections in `citations.md` — `## E1 — Vendor error-body shape`, `## E2 — Non-200 HTTP statuses … (load-bearing NEGATIVE)`, `## E3 — Rate-limit signaling … (NEGATIVE)`, `## E4 — Errors delivered inside a Chat Completions stream`, `## E5 — type/code value vocabularies`, `## E6 — Request-id surfacing (NEGATIVE …)`. **SATISFIED.**

### S-AEM-070 *[inspection]* — performed, PARTIAL

R-AEM-002's eleven table rows are each pinned by a provider-named byte-level fixture in `testdata/errormap/` and replayed by `TestFailureMap_StatusCategoryTable`. **Satisfied.** The dialect-conventional claims in R-AEM-007 (`x-request-id`), R-AEM-008 (`Retry-After` presence) and R-AEM-009 (`x-ratelimit-*`) are pinned by **inline raw-HTTP text** inside the test files rather than by a file in the change's fixture set. See WARNING-4.

---

## Correctness (Static Evidence)

| Requirement | Status | Notes |
|---|---|---|
| R-AEM-001 | Implemented | `mapResponse` returns nil for 2xx; `ai.PreStreamFailure` otherwise (`failure_map.go:31-37`) |
| R-AEM-002 | Implemented | `classifyStatus` switch matches the spec table row-for-row; `FailureCategoryUnsupportedCapability` unreachable from status |
| R-AEM-003 | Implemented | `retryableFor` derives from category alone — single shared derivation (design D6) |
| R-AEM-004 | Implemented | `parseVendorErrorBody` returns `("", false)` on any parse failure; never errors, never panics |
| R-AEM-005 | Implemented | Class fallback + explicit `status < 100 \|\| status > 599` → `(Unknown, false)` |
| R-AEM-006 | Implemented | Wrapped-then-bare unmarshal; pointer fields distinguish null from absent; label = `type` else `code` |
| R-AEM-007 | Implemented | `header.Get("X-Request-Id")` — net/http canonicalization gives case-insensitivity for free; AI-19's `sanitizeOpaqueField` performs the drop-whole |
| R-AEM-008 | Implemented | `strconv.Atoi` then `http.ParseTime`; negative clamp to 0; malformed → absent |
| R-AEM-009 | Implemented | Fixed 3-name allowlist; `Error()` is a fixed constant; pointer-receiver carrier in the cause chain |
| R-AEM-017 | Implemented | No backoff/attempt/failover export; 9 categories; 0 requires |
| R-AEM-018 | Implemented (partial) | GoDoc labels every wire claim; fixture-location caveat in WARNING-4 |

---

## Coherence (Design rev 2)

| Decision | Followed? | Notes |
|---|---|---|
| D1 — capture limit | Yes | `captureLimit = 8 << 10`, `truncationMarker = "...(truncated)"`, both unexported in `capture.go` |
| D2 — retry-metadata carrier | Yes | Exported `RateLimitTelemetry`, three individually addressable string fields, fixed `Error()` text, `Unwrap()`, exactly the three pinned header names |
| D3 — injectable clock | Yes | `observedAt time.Time` parameter; no package-level clock anywhere in the change |
| D4 — mapping home + fixtures | Yes | All five functions unexported in `failure_map.go`; 11 per-status fixtures replayed via `http.ReadResponse` |
| D5 — stage-2 seams | Correctly deferred | `stream_failure.go` absent; no producer signature imagined |
| D6 — timeout/cancel tree | Partially landed, by design | `retryableFor` (the stage-1 half) is implemented and shared; `categorizeStreamError` is stage 2 |
| D7 — sanitization site | **Partial, by design** | `capturedBody` + fixed `Error()` + `Unwrap()` + unexported `bytes()` all landed. The `captureLimit+1` probe read, marker append, and `io.Copy(io.Discard, rc)` drain are deliberately deferred to slice 2c, because implementing them now would make S-AEM-056…059 pass from birth and violate the spec's own Strict TDD clause. `tasks.md` work-unit 2c explicitly lists "capture.go (D7 finalization)", so this is planned scope, not drift. |
| D8 — in-band identity | Correctly deferred | `ErrInBandErrorFrame` is **not declared**; it appears only inside a `capture.go` doc comment as a forward reference |

---

## Stage-2 gate integrity (confirmed untouched — correct, not a gap)

| Check | Command / evidence | Result |
|---|---|---|
| `stream_failure.go` absent | filesystem test | ABSENT — correct |
| `capture_proof_test.go` absent | filesystem test | ABSENT — correct |
| `ErrInBandErrorFrame` not declared | `grep -rn` over `backend/agent/src/` | Single hit, and it is a **doc-comment forward reference** in `capture.go:48`, not a declaration — correct |
| S-ART-054 allowlist unchanged | `reasoning_refusal_test.go:258-261` | Exactly two entries: `errors.go:ErrFrameTooLarge`, `errors.go:ErrTruncated` — correct per R-AEM-011's second branch |
| Stage-2 tasks unchecked | `tasks.md` | All 23 of `2a.*`/`2b.*`/`2c.*` remain `[ ]` — correct |
| `TestPolicy_NoNewSentinelsExported` still green | full suite run | PASS, unmodified |

---

## Issues Found

### CRITICAL: None

### WARNING

**WARNING-1 — Recorded test counts are each off by exactly one.**
`tasks.md` 1.17 records "18 top-level `--- PASS` (9 `TestFailureMap_*` + 4 `TestRetryMetadata_*` + 3 `TestCharterBoundary_*` + **2** `TestCapture_*`)"; 1.19 and apply-progress obs #2487 record 155 `openaicompat` / 603 repo-wide.

```text
$ go test -race -count=1 -v -run 'TestFailureMap|TestRetryMetadata|TestCharterBoundary|TestCapture' ./src/ai/openaicompat/...
TOP-LEVEL PASS: 19        (TestCapture_ : 3, not 2)
$ go test -count=1 -v ./src/ai/openaicompat/  → 156 top-level PASS   (recorded: 155)
$ go test -count=1 -v ./...                   → 604 top-level PASS   (recorded: 603)
```

Root cause: `TestCapture_TruncationMarkerIsPinned` was added later, during lint cleanup, to satisfy the `unused` linter for `truncationMarker`; the counts were not re-taken afterwards. **Impact**: none on correctness — the true delta is +19, all green, zero regressions. It is an evidence-accuracy defect only. **Action**: correct the three numbers in `tasks.md` and the apply-progress observation before archive.

**WARNING-2 — S-AEM-038's carrier assertions are conditionally skipped.**
`retry_metadata_test.go:100-108` wraps its carrier checks in `if errors.As(failure, &telemetry) { … }` with no `else`/`Fatal`. Probe P5 disabled telemetry capture entirely and this test still **passed**:

```text
--- FAIL: TestRetryMetadata_AllThreeHeadersIndividuallyAddressable
--- FAIL: TestRetryMetadata_PartialSubsetTolerated
--- PASS: TestRetryMetadata_CredentialHeaderNeverCaptured     <-- vacuous under this mutation
--- PASS: TestRetryMetadata_FailureErrorTextIsFixedAndClean
```

This is obs #2471 shape 1. **Mitigation already present**: the test's *unconditional* half — `strings.Contains(failure.Error(), planted)` — still guards S-AEM-038's core claim, and sibling tests S-AEM-036/037 do fail, so the regression would not escape the suite. **Action**: change the `if` to a `Fatalf`-guarded `errors.As`, matching the sibling tests' own idiom.

**WARNING-3 — `charter_boundary_test.go` was green from birth and has no permanent negative-case companion.**
The file references **no** AI-32 production symbol (verified by grep for all nine new identifiers). It therefore compiled and passed before any production file existed; its "RED" was purely the package-wide build failure recorded in task 1.11. S-AEM-064/065/066 assert invariants that already held pre-change, so they cannot be shown genuinely failing first.

This sits in tension with the spec's own Strict TDD clause ("A scenario that cannot be shown failing first is not a `[test]` scenario") and with obs #2471's recorded rule that "a guard's negative case must be permanently provable (factor the check into a pure function + synthetic violating input, the `TestSequenceGuard_*_Fails` idiom)". The repo has **30+** established `_Fails`/`_Detects` examples of that idiom; this change follows none of them.

Probe P4 proves the S-AEM-064 guard *does* bite today. **Action**: add a `TestCharterBoundary_SyntheticForbiddenIdentifier_Fails` companion that runs `checkCharterIdent` against a synthetic violating input — `checkCharterIdent` is already factored as a pure function, so this is a small addition. Not a stage-1 blocker.

**WARNING-4 — Three dialect-conventional claims are pinned by inline text, not by the change's fixture set (S-AEM-070 PARTIAL).**
The spec defines a *pinning fixture* as "a recorded byte-level fixture **in this change's fixture set** that **names the provider** and the observed status, header or frame". R-AEM-002's eleven rows meet this fully. But the `x-request-id` (R-AEM-007), `Retry-After` (R-AEM-008) and `x-ratelimit-*` (R-AEM-009) dialect claims are pinned by raw-HTTP string literals inside `failure_map_test.go` / `retry_metadata_test.go`.

The **anti-vacuity purpose is met** — these are genuine byte-level responses parsed through `http.ReadResponse`, explicitly never hand-built `http.Header` maps — so classification is proved against real wire bytes. Only the fixture *location* and *provider-naming* convention is unmet. This was a deliberate apply-phase choice (keep each canonical fixture single-purpose). **Action**: either add provider-named header fixtures under `testdata/errormap/`, or record a spec-side note that inline byte-level replay satisfies the pin. Reviewer decision, not a stage-1 blocker.

**WARNING-5 — PR1 exceeds the 400-line review budget by ~3.3×.**
`git diff --stat feat/ai-32-0-base...HEAD` = **1330 insertions, 25 deletions** across 20 files (production 395 lines, tests 876, fixtures 44, tasks.md 40). `tasks.md`'s own forecast predicted this (`400-line budget risk: High`, `Chained PRs recommended: Yes`, `Delivery strategy: auto-chain`) and the four-slice chain is in place, so this is an accepted, forecast condition rather than an unplanned overrun. Flagged so the reviewer sizes the session accordingly; the 876 test lines are highly repetitive table/subtest structure.

### SUGGESTION

**SUGGESTION-1 — Fixtures use LF, not CRLF.** All 11 `.http` files contain zero `\r` bytes. Real HTTP wire bytes use CRLF line endings; `http.ReadResponse` tolerates bare LF, so every test passes. Converting to CRLF would make the "byte-level fixture" claim literally true and would exercise the parser the way a real socket does.

**SUGGESTION-2 — S-AEM-066 duplicates an existing guard.** `TestCharterBoundary_GoModHasZeroRequireLines` asserts the same property as the pre-existing `TestLayer1_ModuleHasNoDependencies_ZeroRequires` (`src/ai/import_boundary_test.go:147`). Harmless and spec-mandated; worth a cross-reference comment so a future reader knows which is canonical.

**SUGGESTION-3 — `truncationMarker` is currently pinned but unexercised.** It exists only to be value-asserted by `TestCapture_TruncationMarkerIsPinned`, which satisfies the `unused` linter without exercising truncation. This is deliberate and well-documented, but it means slice 2c must not assume any truncation behavior exists yet.

---

## Verdict

**PASS WITH WARNINGS** (stage 1) — 0 CRITICAL, 5 WARNING, 3 SUGGESTION.

All 20 stage-1 tasks are complete and match the code state; all 42 stage-1 `[test]` scenarios are covered by tests observed passing in two consecutive fresh `-race -count=1` runs; lint, gofmt, vet and the zero-dependency charter all pass; five mutation probes confirm the load-bearing assertions bite and the tree reverted clean; and the stage-2 surface is confirmed correctly untouched. Stage 1 is complete and ready to unblock AI-28.6 and slice 2a; correct WARNING-1's three recorded numbers first. The **change** is not archive-ready and must not be archived now — `R-AEM-010`…`016` are unimplemented by design, which is exactly what the envelope's `verdict: fail` records.
