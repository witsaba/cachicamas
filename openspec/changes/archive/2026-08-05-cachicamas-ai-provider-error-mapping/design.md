# Design: Map HTTP and provider failures (AI-32)

## Technical Approach

Three new stage-1 files in `openaicompat` plus one stage-2 file. A pure classification core — `mapErrorResponse(status int, header http.Header, captured capturedBody, observedAt time.Time) *ai.Failure` — is wrapped by a thin `*http.Response` consumer `mapResponse(resp, observedAt)`, so AI-28.6 later calls either without redesign. Stage 2 adds stream-error seams whose only producer-facing input is `outputPreceded bool`, supplied by AI-28.1's landed closing site — no producer API is imagined here. All identity stays in `openaicompat`; `package ai` is consumed, never widened (R-AEM-017).

## Architecture Decisions

| # | Decision | Choice | Rejected alternative | Rationale |
|---|---|---|---|---|
| D1 | Capture limit | Unexported `captureLimit = 8 << 10` (8 KiB) + `truncationMarker = "...(truncated)"` (fixed ASCII), both in `capture.go`, GoDoc-documented | 64 KiB (needless retention); exported const (surface growth, no consumer) | Vendor error bodies are small JSON; 8 KiB ≤ spec's 64 KiB cap. S-AEM-056's `len == captureLimit + len(truncationMarker)` assertion is written from an **internal** test file (`package openaicompat`), the package's existing idiom (ambient_authority_test.go) |
| D2 | Retry-metadata carrier | Exported `type RateLimitTelemetry struct` (`retry_metadata.go`): three string fields `LimitRequests, RemainingRequests, ResetRequests` (empty = absent); implements `error` with **fixed** text `"openaicompat: rate-limit telemetry attached"`; `Unwrap()` to inner cause; built by `captureRateLimitTelemetry(http.Header)` from the fixed three-name allowlist `x-ratelimit-limit-requests`, `x-ratelimit-remaining-requests`, `x-ratelimit-reset-requests` — exactly the names S-AEM-036/037 pin | Widening `ai.FailureReport` (forbidden); map-typed blob (not individually addressable, S-AEM-036); a `*-tokens` header trio (dialect claim with no replaying scenario or fixture — R-AEM-018 forbids it) | `errors.As` needs an error-implementing type in the cause chain. Fixed `Error()` text means no header value can render (S-AEM-038/039). Allowlist-only reads make `authorization` structurally uncapturable |
| D3 | Injectable clock | `observedAt time.Time` parameter on the pure core; production callers pass `time.Now()` at the call site | Package-level `var now = time.Now` (mutable global, parallel-unsafe); clock field on `Client` (widens Config, S-APC-015) | S-AEM-032/033 pass fixed instants; pure function, zero global state |
| D4 | Mapping home + fixtures | `failure_map.go`: `classifyStatus(status) (ai.FailureCategory, bool)` (table R-AEM-002 + class fallback R-AEM-005 + out-of-range→Unknown/`StatusClass 0`), `parseVendorErrorBody([]byte)` (both wrappers, null-tolerant, label = `type` else `code`), `parseRetryAfter(string, observedAt) ai.RetryDelay`, `mapErrorResponse(...)`, `mapResponse(...)`. All unexported — AI-28.6 is in-package. Fixtures: `testdata/errormap/<provider>_<status>.http` raw byte-level HTTP responses replayed via `http.ReadResponse`; one file per **status** — 11 files (401, 403, 400, 404, 422, 408, 429, 500, 502, 503, 504), provider named in filename, matching S-AEM-005/007/008's plural fixture demands | Exported mapper (no external consumer; charter minimalism); inline string fixtures (not byte-level replay); one file per table row (S-AEM-005/007/008 each replay multiple statuses) | Pure core is testable pre-producer and composable for stage 2 |
| D5 | Stage-2 seams (observable shape only) | `stream_failure.go`: `categorizeStreamError(err) (ai.FailureCategory, bool)`; `midStreamFailureFrom(err error, outputPreceded bool) *ai.Failure`; `failureFromErrorFrame(payload []byte, outputPreceded bool) *ai.Failure` (category `Unknown` + `RawLabel`, `ErrInBandErrorFrame` in cause chain). **Producer integration deferred until AI-28.1 lands and chains merge**; `outputPreceded` arrives as a parameter from the producer's single closing site | Designing against imagined producer signatures | Spec rule: stage 2 states observable behavior only |
| D6 | Timeout/cancel tree | Order in `categorizeStreamError`: `errors.Is(err, context.Canceled)` → Cancellation; `errors.Is(err, context.DeadlineExceeded)` → Timeout; `errors.As(&net.Error)` + `Timeout()` → Timeout; `Category(err)` (decoder sentinels) → MalformedResponse; else (disconnect/reset/unexpected EOF) → Unavailable. One `retryableFor(cat)` derivation: true iff RateLimit/Unavailable/Timeout (R-AEM-003, applied uniformly) | Message-text inspection (forbidden, R-AEM-010/014) | Canceled checked before net errors because `*url.Error` wraps `context.Canceled` with `Timeout() == false` not guaranteed across transports; sentinel checks are transport-independent |
| D7 | Sanitization site | Unexported `capturedBody` (`capture.go`): bounded read via `io.LimitReader(rc, captureLimit+1)`; when the probe read returns `captureLimit+1` bytes, retain **exactly the first `captureLimit` bytes** (the probe byte is discarded) and append the marker, yielding `len == captureLimit + len(truncationMarker)`; then `io.Copy(io.Discard, rc)` drain and single `Close`. It implements `error` with **fixed** text `"openaicompat: provider error body captured"` **and** `Unwrap() error` returning the inner cause (`ErrInBandErrorFrame` on the frame path, else nil), so the chain below traverses for S-AEM-043/061; captured bytes reachable only via the unexported, non-rendered `bytes()` accessor. Cause chain: `Failure → [RateLimitTelemetry →] capturedBody [→ ErrInBandErrorFrame]` | Sanitize-at-render (fails S-AIP-029: `Unwrap` keeps causes inspectable); exported `Bytes()` accessor (would reopen R-AEM-016's structural hole — all consuming tests are internal) | A credential in the body can never reach any `Error()` because no error text is ever constructed from body bytes. Companion proof S-AEM-063 (vacuous-pass lesson 1) asserts `bytes()` contains surrounding body text |
| D8 | In-band identity | Exported `ErrInBandErrorFrame` declared in `errors.go` **after** `ErrTruncated`; S-ART-054 allowlist gains third entry `errors.go:ErrInBandErrorFrame` at its exact scan position (ReadDir filename order, then declaration order — here: appended last), with adjacent comment citing R-AEM-010/R-AEM-011 | Unexported identity (external callers could not run the `errors.Is` distinction R-AEM-010 grants); exported non-Err type with `Is` (evades rather than reconciles the guard) | Exported because the in-band-vs-transport distinction is part of the adapter's **public error contract** for external consumers — Layer 2 diagnostics matching `errors.Is(err, openaicompat.ErrInBandErrorFrame)` — consistent with `ErrFrameTooLarge`/`ErrTruncated`'s own exported precedent; internal tests alone would not justify the export. R-AEM-011's first branch sanctions the allowlist reconciliation |

**Exported-surface guards checked for D2/D8** (all clear): `TestPolicy_NoNewSentinelsExported` (Err-prefixed var/const only — the type `RateLimitTelemetry` does not match; the sentinel is allowlist-reconciled); `ambient_authority_test.go` (new files import no `os`/`os/exec`/`syscall`/`io/ioutil`); `credential_scan_test.go` — cleared only under a **load-bearing constraint**: that guard raw-byte-scans `package openaicompat_test` files for `sk-[A-Za-z0-9_-]{20,}` / `Bearer [A-Za-z0-9._-]{20,}`, and S-AEM-060's sentinel `sk-AEM060-planted-in-body-only` **matches** the first pattern, so the internal-package (`package openaicompat`) placement of every test file asserting that literal is load-bearing, not stylistic — **sdd-tasks MUST NOT place those tests in the external test package**; `provider_boundary_test.go` (no `Stream` method added); `feature_inventory_test.go`/`policy_walk_test.go` (scan `package ai` With* constructors and feature dispositions — unaffected); new S-AEM-064 scan (no exported identifier names backoff/attempt/failover; `RateLimitTelemetry` carries none).

> **Corrective (design-validation, 2026-08-04).** D2: the `*-tokens` header trio is dropped — the allowlist is exactly the three requests-series names S-AEM-036/037 pin; a dialect claim with no replaying scenario+fixture would violate R-AEM-018. D4: fixtures are per-status (11 files), not per-row. D7: `Unwrap() error` added so the cause chain traverses for S-AEM-043/061; accessor renamed to unexported `bytes()`; truncate-back mechanics stated explicitly. D8: export rationale replaced with the honest one (public error contract, `ErrFrameTooLarge`/`ErrTruncated` precedent). Credential-scan clearance restated as a load-bearing internal-package constraint, not a blanket pass.

## Data Flow

    Pre-stream (stage 1):
    *http.Response ──→ mapResponse(resp, observedAt)
        2xx → nil                     non-2xx ↓
    captureBody(rc) ──→ mapErrorResponse(status, header, captured, observedAt)
        classifyStatus + parseVendorErrorBody + parseRetryAfter + captureRateLimitTelemetry
        └─→ ai.PreStreamFailure(report{Cause: telemetry?→capturedBody})

    Mid-stream (stage 2, producer supplies outputPreceded):
    stream err / error frame ──→ categorizeStreamError | failureFromErrorFrame
        └─→ ai.MidStreamFailure(report, outputPreceded) ──→ ai.ErrorEvent

## File Changes

| File | Action | Stage | Description |
|---|---|---|---|
| `backend/agent/src/ai/openaicompat/capture.go` | Create | 1 | `capturedBody`, `captureBody`, limit + marker constants |
| `backend/agent/src/ai/openaicompat/failure_map.go` | Create | 1 | Status table, fallback, body parse, Retry-After, `mapResponse` core |
| `backend/agent/src/ai/openaicompat/retry_metadata.go` | Create | 1 | `RateLimitTelemetry`, header allowlist, capture |
| `backend/agent/src/ai/openaicompat/errors.go` | Modify | 1 | Amend the two doc comments stating AI-27/`Category` construct no `ai.Failure` — `ErrFrameTooLarge`'s ("AI-27 never constructs …", errors.go:49-53) and `Category`'s ("Category never constructs … AI-28 and AI-32 own construction", errors.go:74-77) — to record AI-32 as the **wire-side** constructor, AI-28 as the producer path, and AI-26.6's translation-time refusal in `policy.go` as a distinct, pre-existing construction site left untouched (S-AEM-067) |
| `backend/agent/src/ai/openaicompat/testdata/errormap/*` | Create | 1+2 | Byte-level pinning fixtures per dialect-conventional status/header/frame (11 status fixtures in stage 1, per D4) |
| `backend/agent/src/ai/openaicompat/stream_failure.go` | Create | 2 | `categorizeStreamError`, `midStreamFailureFrom`, `failureFromErrorFrame` |
| `backend/agent/src/ai/openaicompat/errors.go` | Modify | 2 | `ErrInBandErrorFrame` after `ErrTruncated` |
| `backend/agent/src/ai/openaicompat/reasoning_refusal_test.go` | Modify | 2 | Allowlist third entry + citing comment (S-AEM-044/045) |

> **Corrective (design-validation, 2026-08-04).** The S-AEM-067 amendment previously targeted `doc.go`; spec rev 2 corrected the scenario — no such ruling exists in `doc.go`. The two real passages live in `errors.go` (the `ErrFrameTooLarge` and `Category` doc comments), so the File Changes row is retargeted there and the `doc.go` edit is dropped: it has no justification on its own merits. `errors.go` is therefore modified in both stages — doc-comment amendment in stage 1, sentinel declaration in stage 2.

## Testing Strategy — slices and RED tests (Strict TDD)

| Slice | Gate | Scenarios | RED test files (internal `package openaicompat`) |
|---|---|---|---|
| 1 — stage 1 whole (AI-32.1 + AI-32.4 + charter) | None; target ≪ 5000-line budget | S-AEM-001…039, 064…066 (42 `[test]`); 067…070 `[inspection]` at review | `failure_map_test.go`, `retry_metadata_test.go`, `charter_boundary_test.go`; `capture_test.go` — its RED tests trace to R-AEM-004's retained-bounded-diagnostic sentence ("the bytes actually read MUST be retained as a bounded diagnostic reachable through the error chain and MUST NOT appear in `Failure.Error()`"), exercised through S-AEM-014…016 |
| 2a — AI-32.2 frames | AI-28.1 merged | S-AEM-040…044 (5 `[test]`); S-AEM-045 `[inspection]` at review | `stream_failure_test.go` (frame cases) |
| 2b — AI-32.3 disconnects/deadlines | AI-28.1 merged | S-AEM-046…055 (10 `[test]`) | `stream_failure_test.go` (transport cases) |
| 2c — AI-32.5 capture proofs | 2a + 2b | S-AEM-056…063 (8 `[test]`) | `capture_proof_test.go` |

Every `[test]` scenario is demonstrated failing before its implementation exists. `[test]` totals sum to the spec's 65 (42 + 5 + 10 + 8); the five `[inspection]` scenarios (S-AEM-045, 067…070) are reviewer-checked, never in a RED set. Note: S-AEM-056…063 mechanically exercise only stage-1 machinery, but stay in stage 2 per the spec's node ownership (AI-32.5 depends on AI-32.2/32.3).

> **Corrective (design-validation, 2026-08-04).** S-AEM-045 is `[inspection]` and is carved out of slice 2a's RED set exactly as slice 1 carves out 067…070; per-slice `[test]` counts are now explicit and sum to 65. `capture_test.go`'s slice-1 RED trace is named (R-AEM-004) so slice 1's RED set is not scenario-homeless.

## Requirement homes (18/18 — none homeless)

| Requirements | Home |
|---|---|
| R-AEM-001…005 | `failure_map.go` (`classifyStatus`, `mapResponse`) |
| R-AEM-006, 007 | `failure_map.go` (`parseVendorErrorBody`, request-id via `http.Header.Get` → AI-19 drop-whole) |
| R-AEM-008 | `failure_map.go` (`parseRetryAfter`, both RFC 9110 forms, `observedAt` math) |
| R-AEM-009 | `retry_metadata.go` (`RateLimitTelemetry`) |
| R-AEM-010, 011 | `stream_failure.go` + `errors.go` sentinel + allowlist reconciliation |
| R-AEM-012…014 | `stream_failure.go` (`categorizeStreamError`, `midStreamFailureFrom`) + producer closing site (deferred integration) |
| R-AEM-015, 016 | `capture.go` (`capturedBody`) + slice 2c proofs |
| R-AEM-017 | Charter scan `charter_boundary_test.go` (S-AEM-064…066); zero `require` lines preserved (stdlib only); S-AEM-067's amendment lands on `errors.go`'s two doc comments (per spec rev 2), not `doc.go` |
| R-AEM-018 | Spec/citations labels; implementation GoDoc labels every wire claim cited (E1…E6) or dialect-conventional with its fixture named |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Each slice reverts locally to `openaicompat` (proposal rollback plan); stage 1 ships independently of AI-28's chain.

## Open Questions

- [ ] None blocking. Category for a bare transport disconnect (D6 fallback → `Unavailable`, retryable) is a design ruling the spec deliberately leaves open (R-AEM-012 pins delivery/partial-output, not category); recorded here so sdd-tasks tests assert it intentionally.
