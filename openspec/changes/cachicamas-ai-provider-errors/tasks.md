# Tasks: The provider error taxonomy and the terminal error event (AI-19)

> Nodes: AI-19.1 terminal error event · AI-19.2 category vocabulary · AI-19.3 retry hints + safe metadata · AI-19.4 partial-output discriminator · AI-19.5 one vocabulary, two paths.
> Package `backend/agent/src/ai`. Strict TDD: every item RED → GREEN → REFACTOR, in spec order. Runner: `make test` (`go test -race -v ./...`) from `backend/agent/`.
> **Reconciliation done at task time**: AI-14's `design.md` (landed) re-verified — `eventPayload{ kind() EventKind; validate(at Path) *Violation }` is sealed, matching `R-AEE-003`; `event.go`'s `eventRegistry []eventRegistration{name, descriptor}` and `event_descriptor.go`'s 6-step kind-adding recipe are the pinned integration surface. `design.md`'s "provisional" caveat is resolved and removed. AI-19.6 split trigger re-checked: category count stays 9, no category-specific metadata field added — not appended (spec open item 1).

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1600–2200 (1 production file + 2 append-only touches + 2 test files, vs. `finish_reason.go`/`usage.go` precedent for a smaller surface) |
| 400-line budget risk | High |
| Chained PRs recommended | No — resolves directly to size:exception per accepted wave-wide 5000-line budget |
| Suggested split | Single PR (work units below for review/rollback granularity only) |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units (internal granularity, not separate PRs)

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Foundation + AI-19.1: skeleton, `EventKindError` wiring | PR 1 (exception) | `go test -run TestErrorEvent -race ./backend/agent/src/ai/...` | N/A — pure library, no provider/runtime integration at this milestone | Revert `provider_failure.go` + `event.go`/`doc.go` append lines; nothing else imports them yet |
| 2 | AI-19.2/3: category vocabulary, retry hints, redaction | PR 1 (exception) | `go test -run TestFailureCategory\|TestRetry -race ./backend/agent/src/ai/...` | N/A | Revert additive blocks in `provider_failure.go` |
| 3 | AI-19.4/5: discriminator, sentinel/wrap parity | PR 1 (exception) | `go test -run TestPartialOutput\|TestFailureParity -race ./backend/agent/src/ai/...` | N/A | Revert additive blocks in `provider_failure.go`; no other package depends on this milestone yet |

## Phase 1: Foundation — category and failure skeleton (blocks all nodes)

- [x] 1.1 RED `provider_failure_internal_test.go`: `FailureCategory` zero value and `DeliveryPath` zero value are not members (groundwork for R-AIP-005/R-AIP-010).
- [x] 1.2 GREEN `provider_failure.go`: `FailureCategory uint8` full iota block (9 charter members + `failureCategoryLimit`), `DeliveryPath uint8` (`DeliveryPreStream`/`DeliveryMidStream`/`deliveryPathLimit`), `RetryDelay`/`Delay`, `FailureReport`, `Failure` struct (unexported fields), stub `Error()`/`Unwrap()`.
- [x] 1.3 REFACTOR: `gofmt`/`go vet` clean; no test changes needed.

## Phase 2: AI-19.1 — Terminal error event (R-AIP-001..003)

- [x] 2.1 RED `provider_failure_test.go` (`ai_test`): S-AIP-001..003 — construct a `*Failure` and wrap it via `ErrorEvent` using only exported identifiers; no in-package assistance; no provider interface exists yet.
- [x] 2.2 GREEN `event.go`: `EventKindError` constant (moves `eventKindEnd`) + `eventRegistry` append line (steps 1/5 of AI-14's recipe). GREEN `provider_failure.go`: `PreStreamFailure`/`MidStreamFailure` constructors return `*Failure`; `*Failure` implements `eventPayload{kind, validate}` directly (D7); `ErrorEvent`/`ErrorPayload`.
- [x] 2.3 RED S-AIP-004..006: kind derived from payload, not supplied separately; nil and mismatched-type payload rejected via AI-04 sentinel at `At("payload")`; `Terminated()` true and readable without inspecting category.
- [x] 2.4 GREEN: `ErrorEvent` validation (`ErrEmpty` at `At("payload")` for nil); payload `kind()` returns `EventKindError`.
- [x] 2.5 RED S-AIP-007..009: terminal exclusivity — no shared accessor name/return type with AI-15's completion payload; kind-only discrimination, no string comparison.
- [x] 2.6 GREEN: confirm accessor disjointness — structural, no new code.
- [x] 2.7 REFACTOR: kind-list entry (landed in `event.go`'s own `EventKind` doc comment, not `doc.go` — see apply-progress deviation 1); shared `event_registry_test.go` exhaustiveness guard extended; `go test -race ./src/ai/...`; gofmt/vet clean; record red/green output.

## Phase 3: AI-19.2 — Category vocabulary (R-AIP-004..006)

- [x] 3.1 RED: S-AIP-010..012 nine categories construct and are mutually distinct; S-AIP-015..016 enumeration stable order from external package; internal S-AIP-013..014 zero-value/out-of-range → `ErrNotInVocabulary`, no panic.
- [x] 3.2 GREEN `provider_failure.go`: `String()` (`[failureCategoryLimit]string` array), `Validate(at ...Step) error`, `FailureCategories() []FailureCategory`; also wired the category rule into `Failure.validate` (`newFailure`/`PreStreamFailure`/`MidStreamFailure` now reject an invalid category with `ErrNotInVocabulary` at `category` — proven with an added test, not silently added).
- [x] 3.3 RED S-AIP-017..021: unknown-category raw label preserved across one wrap; over-long/control-char label dropped whole (D6); no cross-vendor mapping function exists (proven by an AST scan for any top-level `func(string) FailureCategory`, mirroring `NormalizeFinishReason`'s shape); empty label on a modelled category still constructs.
- [x] 3.4 GREEN: `RawLabel()` field + accessor with 64-byte drop-whole bound (`sanitizeOpaqueField`, shared with `RequestID` in Phase 4).
- [x] 3.5 REFACTOR `provider_failure_internal_test.go`: exhaustiveness pin for the name array (length matches `failureCategoryLimit`, every member non-empty) — done. The "every category has a sentinel" half is deferred to Phase 6 task 6.4 (sentinels don't exist until then; same Go-compile-order reasoning as the Phase 2/3 `validate()` staging, see apply-progress.md deviation 3). `go test -race ./src/ai/...` green; gofmt/vet clean.

## Phase 4: AI-19.3 — Retry hints and safe metadata (R-AIP-007..009)

- [x] 4.1 RED S-AIP-022..023: retryability boolean readable for every category (all 9); no retry-scheduling/backoff/attempt-counter/failover identifier exported (AST scan reusing `text_events_test.go`'s shared `exportedTopLevelNames` helper rather than duplicating it). S-AIP-024..027: `RetryDelay` presence-vs-value distinguishability, incl. explicit zero.
- [x] 4.2 GREEN: `Retryable()`, `RetryAfter() (time.Duration, bool)`.
- [x] 4.3 RED S-AIP-028..031: planted-sentinel test (embedded credential string in the wrapped cause) — `Error()` excludes cause text, `Unwrap()` still exposes it (and is errors.Is-reachable); `StatusClass()`/`RequestID()` are dedicated accessors, not substrings.
- [x] 4.4 GREEN: `Error()` fixed prefix + category text only (D5); `StatusClass()`/`RequestID()` accessors; status-class 0..5 bound wired into `validate()`. Process note: the status-class bound was written before its own dedicated test (a real strict-TDD ordering slip on that one sub-rule, recorded honestly — see apply-progress.md deviation 6) — a test was added immediately after and the bound was verified genuinely load-bearing by temporarily removing it, confirming the test fails, then restoring it.
- [x] 4.5 REFACTOR: `Error()`'s doc comment records the redaction guarantee as a type property; confirmed structurally that `Error()` never references `f.cause`. `go test -race ./src/ai/...` green; gofmt/vet clean (pre-existing `completion_test.go` nit excluded, not mine).

## Phase 5: AI-19.4 — Partial-output discriminator (R-AIP-010..012)

- [ ] 5.1 RED S-AIP-032..035: three shapes (`pre-stream/false`, `mid-stream/false`, `mid-stream/true`) distinguishable from the failure value alone; `PreStreamFailure` takes no output-flag parameter; delivery path alone cannot distinguish the two mid-stream shapes.
- [ ] 5.2 GREEN: `PartialOutput()`, `Delivery() DeliveryPath` wired from `PreStreamFailure`/`MidStreamFailure(_, outputPreceded bool)` (D8).
- [ ] 5.3 RED S-AIP-036..040: naive-retry-safe answerable from the discriminator alone regardless of category/retryability/delivery path; no accessor/predicate combines the two axes.
- [ ] 5.4 GREEN: confirm no combining accessor exists — structural, no new code.
- [ ] 5.5 REFACTOR: consolidate the three-shape table test; record red/green.

## Phase 6: AI-19.5 — One vocabulary, two delivery paths (R-AIP-013..015)

- [ ] 6.1 RED S-AIP-041..044: dynamic type identical on both delivery paths; no second failure type, second vocabulary, converter, or AI-04 rule-class registry edit.
- [ ] 6.2 GREEN: confirm structurally (delivered by Phases 1–5); add `Is(target error) bool` matching the failure's own category sentinel (D4), **no umbrella sentinel**.
- [ ] 6.3 RED S-AIP-045..047: `errors.Is`/`errors.As` reach through at least one wrap, including the cause's own sentinel chain via `Unwrap()`.
- [ ] 6.4 GREEN: per-category sentinels (`ErrAuthentication` … `ErrUnknownFailure`); finalize `Is`/`Unwrap` wiring.
- [ ] 6.5 RED S-AIP-048..050: pre-stream-only and terminal-event-only consumers each classify every category; identical accessor sets on both paths.
- [ ] 6.6 GREEN/REFACTOR: both-paths parity table test; `make test -run TestFailure`; record red/green.

## Phase 7: Cross-cutting NFRs and closeout

- [ ] 7.1 RED totality table (S-AIP-052, NFR-AIP-B): zero category, zero `Failure`, nil `*Failure`, out-of-range category, over-long raw label, nil cause, nil payload — none panics.
- [ ] 7.2 GREEN: totality guards across `provider_failure.go` accessors and constructors.
- [ ] 7.3 Verify NFR-AIP-A (S-AIP-051): `go.mod` still zero requires; both AI-00 import guards pass.
- [ ] 7.4 Verify NFR-AIP-C (S-AIP-053): every rejecting scenario in this spec resolves through AI-04's failure value and a landed sentinel, position names the offending field.
- [ ] 7.5 Verify NFR-AIP-E (S-AIP-055): re-confirm AI-14/AI-15 landed surfaces match this file's Phase-1 reconciliation note.
- [ ] 7.6 Record NFR-AIP-D evidence (S-AIP-054): red/green output + refactor note per test-list item, per node, in this file.
- [ ] 7.7 Run `make test` (`go test -race -v ./...`) and `make lint` from `backend/agent/`; confirm green/clean before archive.

> **Deviation note**: exceeds the sdd-tasks 530-word budget. `NFR-AIP-D` requires every one of 15 requirements' test-list items tracked red→green→refactor with recorded output across 5 nodes and 55 scenarios; the spec and design artifacts for this same change, and AI-14's `tasks.md` (this same wave), recorded the identical deviation for the same reason; house convention wins.
