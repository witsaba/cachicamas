# Tasks: Map HTTP and provider failures into the AI-19 taxonomy (AI-32)

## Stage 2 Gate (read before Phase 2a)

Slices 2a/2b/2c are **BLOCKED** on AI-28.1's producer surface landing and its chain merging (proposal Delivery table, design D5). Do not branch or design against an imagined producer signature — `outputPreceded bool` and the frame/transport call sites must be read from AI-28's landed code. If the gate is not satisfied when a stage-2 slice would branch, STOP and report **blocked, not re-parented**; stage 1 still ships and unblocks AI-28.6 independently (design Migration/Rollout).

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (naive) | 900–1,300 |
| Estimated changed lines (corrected, repo's 2–4x undershoot) | 1,800–4,200 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 4 slices: 1 → 2a → 2b → 2c |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Stage 1 whole: AI-32.1 + AI-32.4 + charter (42 [test] + 3 [inspection] at review) | PR1 → tracker (`feat/ai-32-0-base`) | `go test -race -count=1 -run 'TestFailureMap|TestRetryMetadata|TestCharterBoundary|TestCapture' ./src/ai/openaicompat/...` | N/A — byte-level `http.ReadResponse` fixture replay, no live network | `capture.go`, `failure_map.go`, `retry_metadata.go`, `errors.go` (doc-comment amendment only), `testdata/errormap/*`, four new test files |
| 2a | AI-32.2 in-band error frames (5 [test] + 1 [inspection]) | PR2a → PR1 | `go test -race -count=1 -run TestStreamFailure_Frame ./src/ai/openaicompat/...` | N/A — transcript-fixture replay, no live provider process | `stream_failure.go` (frame branch), `errors.go` (`ErrInBandErrorFrame`), `reasoning_refusal_test.go` allowlist entry, `stream_failure_test.go` frame cases |
| 2b | AI-32.3 disconnects/deadlines (10 [test]) | PR2b → PR2a | `go test -race -count=1 -run TestStreamFailure_Transport ./src/ai/openaicompat/...` | N/A — simulated transport/context fixtures | `stream_failure.go` (transport branch), `stream_failure_test.go` transport cases |
| 2c | AI-32.5 bounded, sanitized capture proofs (8 [test]) | PR2c → PR2b | `go test -race -count=1 -run TestCaptureProof ./src/ai/openaicompat/...` | N/A — pure byte fixtures | `capture_proof_test.go`, `capture.go` (D7 finalization) |

Full-suite gate before each PR: `go test -race -count=1 ./...` (from `backend/agent/`) green. `sk-[A-Za-z0-9_-]{20,}` credential-scan guard clean.

## Coverage

18/18 requirements (`R-AEM-001…018`) and 70/70 scenarios (`S-AEM-001…070`, 65 `[test]` / 5 `[inspection]`: 045, 067–070) mapped below. No gap.

## Phase 0 — Preconditions

- [x] 0.1 Safety net: run `go test -race -count=1 ./...` from `backend/agent/` before any slice-1 file exists; capture baseline pass count and confirm `grep -c '^require' backend/agent/go.mod` is `0`. **Evidence**: baseline green — `ok src/agenttest 2.240s`, `ok src/ai 3.360s`, `ok src/ai/openaicompat 3.829s`; 137 passing subtests in `openaicompat`, 585 repo-wide (`go test -count=1 -v ./... | grep -c '^--- PASS'`); `grep -c '^require' backend/agent/go.mod` = `0`. Run on branch `feat/ai-32-1-status-taxonomy` before any slice-1 file existed.

## Phase 1 — Slice 1: AI-32.1 + AI-32.4 + charter (R-AEM-001…009, 017, 018 · S-AEM-001…039, 064…066; 067 inspection)

- [x] 1.1 RED `failure_map_test.go`: status→category table over every `R-AEM-002` row (401/403/400,404,422/429/500,502,503/408,504), 2xx produces no failure, `StatusClass()` always `(class, true)` (S-AEM-001–010).
- [x] 1.2 RED `failure_map_test.go`: `Retryable()` true only for rate-limit/unavailable/timeout, independent of the `Retry-After` hint (S-AEM-011–013).
- [x] 1.3 RED `failure_map_test.go`: unparseable/empty/wrong-shape/absent body still maps by status, no panic, no category leak (S-AEM-014–016).
- [x] 1.4 RED `failure_map_test.go`: undocumented status via class fallback incl. out-of-range 0/999 → `StatusClass() == (0, false)` (S-AEM-017–020).
- [x] 1.5 RED `failure_map_test.go`: vendor body tolerant parse (wrapped/bare/null fields), label opacity (no vocabulary match), `message` never reaches `Error()` (S-AEM-021–025).
- [x] 1.6 RED `failure_map_test.go`: request-id best-effort, case-insensitive header, drop-whole (not truncate) on over-long value (S-AEM-026–029).
- [x] 1.7 RED `failure_map_test.go`: `Retry-After` delay-seconds + HTTP-date forms incl. zero/negative/at-or-before-instant/malformed (S-AEM-030–035).
- [x] 1.8 RED `retry_metadata_test.go`: `RateLimitTelemetry` retrievable via `errors.As`, individually addressable fields, partial-subset tolerance, credential exclusion, fixed `Error()` text (S-AEM-036–039).
- [x] 1.9 RED `capture_test.go`: bytes actually read retained as bounded diagnostic reachable through the error chain, absent from `Failure.Error()` — traces R-AEM-004's retained-diagnostic sentence via S-AEM-014–016 at the capture layer.
- [x] 1.10 RED `charter_boundary_test.go`: no exported backoff/attempt-count/failover identifier; `ai.FailureCategories()` length exactly 9; `go.mod` zero `require` lines (S-AEM-064–066).
- [x] 1.11 Run all slice-1 RED suites together (`go test -race -count=1 ./src/ai/openaicompat/...`); capture the failing/undefined-symbol output as evidence before any production file below exists. **Evidence**: compile failure — `src/ai/openaicompat/capture_test.go:52:15: undefined: mapResponse`, `:57:11: undefined: capturedBody`, `src/ai/openaicompat/failure_map_test.go:100:15: undefined: mapResponse` (×6 more sites), `too many errors` (Go's 10-error cap) — `FAIL github.com/cachicamas/backend/agent/src/ai/openaicompat [build failed]`. Genuine RED: every new test file referenced not-yet-existing production symbols.
- [x] 1.12 Create 11 byte-level fixtures `testdata/errormap/<provider>_<status>.http` (401, 403, 400, 404, 422, 408, 429, 500, 502, 503, 504), replayed via `http.ReadResponse`; wire into 1.1–1.7's tests (design D4). **Note**: provider name `openai` (the pinned dialect); each fixture carries a `Content-Type: application/json` header and a minimal wrapped vendor body, no `Retry-After`/ratelimit headers (those variations are inline raw-HTTP-text fixtures in the test files, keeping each canonical fixture single-purpose).
- [x] 1.13 GREEN `capture.go`: `capturedBody` type, `captureBody()`, `captureLimit`/`truncationMarker` constants (D1) — minimal retain needed for 1.9; full bounded-read/drain/marker proof deferred to slice 2c. **Deviation note**: `captureBody` deliberately does NOT append `truncationMarker` on overflow or drain past `captureLimit` in this slice — doing so now would make slice 2c's own RED-first proofs (S-AEM-056…059) pass from birth, violating Strict TDD's "demonstrated failing before the code exists" rule for those scenarios. `truncationMarker`'s literal value is pinned by `TestCapture_TruncationMarkerIsPinned` so it is referenced (satisfies the `unused` linter) without exercising the mechanism.
- [x] 1.14 GREEN `failure_map.go`: `classifyStatus` (table + class fallback), `parseVendorErrorBody`, `parseRetryAfter`, request-id capture, `mapErrorResponse`/`mapResponse`. **Implementation note**: `classifyStatus`'s second `bool` return means "status is in the transport-legal 100–599 range" (feeds `StatusClass` presence), not retryable — `retryableFor(category)` is the single shared derivation design D6 calls for, reused later by stage 2's `categorizeStreamError`.
- [x] 1.15 GREEN `retry_metadata.go`: `RateLimitTelemetry` struct + `captureRateLimitTelemetry()` over the fixed 3-name allowlist (`x-ratelimit-limit-requests`, `-remaining-requests`, `-reset-requests`).
- [x] 1.16 GREEN `errors.go`: amend the `ErrFrameTooLarge` and `Category` doc comments to record AI-32 as wire-side constructor, AI-28 as producer path, AI-26.6's `policy.go` refusal as a distinct untouched site (S-AEM-067, inspection — reviewer-checked, no RED phase).
- [x] 1.17 Re-run slice-1 suites GREEN: `go test -race -count=1 -run 'TestFailureMap|TestRetryMetadata|TestCharterBoundary|TestCapture' ./src/ai/openaicompat/...`; capture pass evidence. **Deviation note**: test functions renamed from an initial `TestMapResponse_*`/`TestRateLimitTelemetry_*` scheme to `TestFailureMap_*`/`TestRetryMetadata_*` specifically so this documented filter regex matches them (the original names would have silently matched zero tests for those two groups). **Evidence**: `ok github.com/cachicamas/backend/agent/src/ai/openaicompat 1.357s`, 18 top-level `--- PASS` (9 `TestFailureMap_*` + 4 `TestRetryMetadata_*` + 3 `TestCharterBoundary_*` + 2 `TestCapture_*`), all 42 `[test]` scenarios S-AEM-001…039/064…066 covered plus the capture-layer angle on S-AEM-014…016.
- [x] 1.18 Zero-dependency check: `grep -c '^require' backend/agent/go.mod` reports `0`. **Evidence**: confirmed `0` after full slice-1 implementation.
- [x] 1.19 Full-suite gate: fresh `go test -race -count=1 ./...` from `backend/agent/`, zero regressions vs. 0.1's baseline. **Evidence**: two consecutive fresh `-race -count=1 ./...` runs both green (`ok src/agenttest`, `ok src/ai`, `ok src/ai/openaicompat`); 156 passing subtests in `openaicompat` (+19 vs. baseline — the 18 new top-level tests planned plus `TestCapture_TruncationMarkerIsPinned` added during lint cleanup, zero regressions), 604 repo-wide (+19). *(Counts corrected 2026-08-04 per verify finding W1 — the original 155/+18/603 note was written before the lint-cleanup test landed and never recounted.)* `make lint` clean (`0 issues.`) after fixing an `errcheck` (unchecked `rc.Close()`), 3× `revive` package-comments (blank line added before `package openaicompat` in the 3 new production files, matching the existing `policy.go` convention), and 1× `unused` (`truncationMarker`, resolved via the pinning test in 1.13's note) — golangci-lint v2.9.0 auto-installed into `bin/` (gitignored). Load-bearing guards re-confirmed passing unmodified: `TestCredentialScan_ExpectationSurfaceIsClean`, `TestPolicy_NoNewSentinelsExported`, `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires`. Committed as `88ef99f feat(openaicompat): map HTTP status failures into the AI-19 taxonomy` on branch `feat/ai-32-1-status-taxonomy`.

## Phase 2a — Slice 2a: AI-32.2 mid-stream error frames (R-AEM-010, 011 · S-AEM-040…044; 045 inspection) — BLOCKED-ON: AI-28.1 landed + chains merged

- [ ] 2a.0 Gate check: confirm AI-28.1's producer surface (single closing site, `outputPreceded bool`) is present on the tracker before branching; if absent, STOP and report blocked.
- [ ] 2a.1 RED `stream_failure_test.go`: in-band error frame → terminal event, `PartialOutput()` true iff output preceded, `RawLabel()` survives opaque, no events follow the terminal event (S-AEM-040–042).
- [ ] 2a.2 RED `stream_failure_test.go`: in-band frame vs. transport failure distinguishable by `errors.Is` stable identity, never by message-text inspection (S-AEM-043).
- [ ] 2a.3 RED `reasoning_refusal_test.go` — **S-ART-054 bite-proof**: run `TestPolicy_NoNewSentinelsExported` before the allowlist edit and capture it failing on the new `ErrInBandErrorFrame` identity (guard's own RED phase; a guard checked only pre-edit is green from birth otherwise).
- [ ] 2a.4 GREEN `errors.go`: declare exported `ErrInBandErrorFrame` immediately after `ErrTruncated` (D8).
- [ ] 2a.5 GREEN `stream_failure.go`: `categorizeStreamError` frame branch, `midStreamFailureFrom`, `failureFromErrorFrame` wired to the producer's `outputPreceded` parameter.
- [ ] 2a.6 GREEN `reasoning_refusal_test.go`: append `errors.go:ErrInBandErrorFrame` as the allowlist's **third entry, in the scan's own order**, with an adjacent comment citing R-AEM-010/R-AEM-011 — exact set equality preserved, never re-frozen, never weakened (S-AEM-044).
- [ ] 2a.7 Inspection (S-AEM-045): reviewer confirms the allowlist edit carries its citing comment and the guard's comparison remains exact set equality.
- [ ] 2a.8 Zero-dependency check: `grep -c '^require' backend/agent/go.mod` reports `0`.
- [ ] 2a.9 Full-suite gate: fresh `go test -race -count=1 ./...`, zero regressions.

## Phase 2b — Slice 2b: AI-32.3 disconnects and deadlines (R-AEM-012…014 · S-AEM-046…055) — BLOCKED-ON: same gate as 2a; branches from 2a

- [x] 2b.1 RED `stream_failure_test.go`: disconnect after output → terminal event, `PartialOutput()` true, `Delivery() == ai.DeliveryMidStream`, already-emitted content byte-identical and undisturbed (S-AEM-046–048).
- [x] 2b.2 RED `stream_failure_test.go`: disconnect before output → pre-stream path, `PartialOutput()` false, no terminal event claims partial output (S-AEM-049–050).
- [x] 2b.3 RED `stream_failure_test.go`: deadline → `Timeout` retryable, cancel → `Cancellation` never retryable, mutually exclusive via `errors.Is`, independent of the `PartialOutput` axis (S-AEM-051–055).
- [x] 2b.4 GREEN `stream_failure.go`: `categorizeStreamError` order — `context.Canceled`→Cancellation, `context.DeadlineExceeded`→Timeout, `net.Error.Timeout()`→Timeout, decoder `Category()` sentinels→MalformedResponse, else→Unavailable (D6); one uniform `retryableFor(cat)`.
- [x] 2b.5 Zero-dependency check: `grep -c '^require' backend/agent/go.mod` reports `0`.
- [x] 2b.6 Full-suite gate: fresh `go test -race -count=1 ./...`, zero regressions.

## Phase 2c — Slice 2c: AI-32.5 bounded, sanitized capture proofs (R-AEM-015, 016 · S-AEM-056…063) — BLOCKED-ON: 2a + 2b merged

- [ ] 2c.1 RED `capture_proof_test.go`: capture stops exactly at `captureLimit`, marker present iff truncated, `len == captureLimit + len(truncationMarker)` (S-AEM-056–058).
- [ ] 2c.2 RED `capture_proof_test.go`: multi-megabyte body fully drained and closed exactly once, never retains more than `captureLimit` (S-AEM-059).
- [ ] 2c.3 RED `capture_proof_test.go` — **load-bearing package warning**: `S-AEM-060`'s sentinel `sk-AEM060-planted-in-body-only` matches `credential_scan_test.go`'s raw regex `sk-[A-Za-z0-9_-]{20,}`. This file, and every file asserting the S-AEM-060/062 planted sentinel, **MUST declare `package openaicompat`** (internal) — `package openaicompat_test` WILL fail the credential-scan guard's build. Assert the sentinel absent from `Error()`, every `Unwrap()`-reachable cause's `Error()`, and `%v`/`%+v` renderings (S-AEM-060–062).
- [ ] 2c.4 RED `capture_proof_test.go`: sentinel-removed fixture, surrounding body text kept — retained diagnostic still contains it, proving genuine capture, not a vacuous pass (S-AEM-063).
- [ ] 2c.5 GREEN `capture.go`: finalize D7 — `io.LimitReader(rc, captureLimit+1)` probe read, retain exactly the first `captureLimit` bytes + marker on overflow, `io.Copy(io.Discard, rc)` drain + single `Close`, `Unwrap() error` to the inner cause (`RateLimitTelemetry` / `ErrInBandErrorFrame` / nil).
- [ ] 2c.6 Zero-dependency check: `grep -c '^require' backend/agent/go.mod` reports `0`.
- [ ] 2c.7 Full-suite gate: fresh `go test -race -count=1 ./...`, zero regressions; confirm all 65 `[test]` scenarios across the four slices are exercised (42 + 5 + 10 + 8).
