# Delta for `ai-stream-lifecycle` — AI-35 retry discipline and the partial-output boundary

> **Change**: `cachicamas-ai-retry-policy` · **Milestone**: AI-35 (doc 0002:2077–2144) · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-stream-lifecycle/spec.md`](../../../../specs/ai-stream-lifecycle/spec.md) § 7 (V-FAIL-11/12 failure delivery) and § 10 (inheritance) per the contract's amendment rule (lines 30–41). The canonical spec gains a "Retry discipline (AI-35, dated 2026-08-07)" blockquote at § 7 at archive time.
> **Format**: RFC 2119 + Given/When/Then; per-scenario pins marked `(pin)` against the conformance assertions and contract clauses the merged AI-35 work must not break
> **Conformance alignment**: `R-CNF-019` cited as the cross-layer assertion; `R-AIS-031`/`R-AIS-039` cited for the buffer-and-leak posture that the per-attempt drain rides on
> **Sources**: [proposal.md](../proposal.md) · [explore.md](../explore.md) · [doc 0002 § 2077–2144](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0003 § 687–726](../../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md)

---

## Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-retry-policy` |
| **Milestone** | AI-35 (doc 0002 lines 2077–2144) |
| **Capability amended** | `ai-stream-lifecycle` (per proposal § Capabilities) |
| **Type** | Delta — `ADDED` requirements; **no `MODIFIED`, no `REMOVED`, no `RENAMED`** (the amended clauses add new obligations; § 7's "the boundary is handover" rule is unpolluted by this delta — pre-stream retries still happen before handover) |
| **Conformance amended** | `R-CNF-019` added; `R-CNF-017` totality rebuild (one new capability in the closed list) |
| **Test runner** | `cd backend/agent && make test` (`go test -race -v ./...`) |
| **Module** | `backend/agent` — **layered** (not hexagonal), per ADR 0005 § D1 |

### What is being added

Four behavior-only requirements under a single "Retry discipline (AI-35, dated 2026-08-07)" section. They pin AI-35.1's retry predicate, AI-35.2's backoff mechanics, AI-35.3's replayability and partial-output boundary marker, and the cross-layer composed-bound ceiling that AG-15.2's harness-attempt test reads against. The seam — **in-adapter loop, factored into a shared helper invoked by the adapter's execute-once function** — is recorded in the canonical spec at archive time as a § 7 amendment blockquote. The 2026-08-07 doc 0002 amendment (lines 2081–2093) already committed this seam; this delta binds the behavior.

### Producer seams this delta rides on (all already merged at base `main` @ `238b9fa`)

| Seam | Location | Subnodes that ride on it |
| --- | --- | --- |
| Pre-stream ordering: `req.IsZero()` → `Translate(req)` → `ctx.Err()` → `c.newRequest(...)` → `c.httpClient.Do(...)` → `mapResponse(...)` → `isStreamContentType(...)` → `make(chan ai.Event, ...)` → `go run(...)` | `stream.go:199–240` | All four; the helper inserts between `Do` and `mapResponse` |
| Wire-side failure mapper with `retryableFor(category)` derivation | `failure_map.go:31–93` | AI-35.1 item 3 (terminal categories never retry) |
| `RetryAfter()` two-result hint (RFC 9110 § 10.2.3 both forms) | `failure_map.go:187–218`; `provider_failure.go:434–439` | AI-35.2 item 1 (retry-after precedence) |
| Capture-and-drain on the status path (AI-33.5 deliverable) | `capture.go:75–122` | AI-35.3 item 3 (per-attempt drain — supplied by composition) |
| Body marshalling: `Translate(req)` returns `[]byte`; `newRequest` wraps a fresh `bytes.NewReader` per attempt | `translation.go:24`; `body.go:35–55`; `request.go:23–41` | AI-35.3 item 1 (byte-identical replay) |
| `*ai.Failure.PartialOutput()`, `Retryable()`, `RetryAfter()`, `Category()`, `Delivery()` | `provider_failure.go:249–496` | AI-35.1, AI-35.3 |
| Forward guard (AI-00.3): stdlib + own-module imports only | `import_boundary_test.go:107–162` | Helper package's mechanical constraint |

---

## Retry discipline (AI-35, dated 2026-08-07)

> **Amended 2026-08-07 (AI-35)** by `cachicamas-ai-retry-policy` (AI-35, Wave 5 — Harden). Four behavior-only requirements added: **R-AIS-041** (pre-stream retry predicate), **R-AIS-042** (backoff mechanics), **R-AIS-043** (replayability + partial-output boundary marker), **R-AIS-044** (composed-bound ceiling, cross-layer). The seam — in-adapter loop, factored into a shared helper invoked by the adapter's execute-once function — is fixed by the 2026-08-07 doc 0002 amendment (lines 2081–2093) and recorded verbatim in § 7's amendment blockquote at archive time. The canonical `V-FAIL-15` row ("the partial-output case is never retried at Layer 1") is the load-bearing sentence these requirements turn into structure. No `MODIFIED` requirements; the existing § 7 failure-delivery contract (handover as the boundary) is unpolluted by retry, because pre-stream retry is by construction *before* handover.

### R-AIS-041 — Pre-stream retry predicate

The retry helper MAY retry exactly when **all** four conditions hold simultaneously:

1. The failure originates **before** the carrier handover (no semantic event has been emitted).
2. The failure is `Retryable()` per the typed taxonomy (`FailureCategoryRateLimit`, `FailureCategoryUnavailable`, `FailureCategoryTimeout`).
3. The attempt budget has not been exhausted.
4. The caller-owned context has not been cancelled, and the remaining context budget accommodates the next delay.

If any condition fails, the helper MUST return the typed failure as-is — the typed error with its partial-output discriminator is handed up and the harness (Layer 2) decides. The boundary is **"nothing emitted"**, not **"nothing completed"** — a stream handed over that fails before emitting any content is mid-stream delivery (`V-FAIL-12`) and is never retried by Layer 1.

#### Scenario: R-AIS-041 / S-1 — Retryable pre-stream failure is retried *(pin: `R-CNF-019`, `R-ATS-002`, `R-AEM-008`)*

- **GIVEN** a real transport returning `429` with `Retry-After: 0`, repeated across the script's attempt budget
- **WHEN** the caller invokes `Stream(ctx, req)` and no semantic event has been emitted
- **THEN** the helper issues `N+1` wire requests (`N = defaultMaxAttempts`, per R-AIS-044), each one retryable-flagged, AND the typed failure carries the attempt count via the cause chain, AND the carrier handover never occurs

#### Scenario: R-AIS-041 / S-2 — Terminal-category failure is never retried *(pin: `R-CNF-019`, `R-AEM-008`, `R-AEM-009`)*

- **GIVEN** a real transport returning `401` (authentication) or `400` / `404` / `422` (invalid request — `FailureCategoryUnknown`)
- **WHEN** the caller invokes `Stream(ctx, req)`
- **THEN** the helper issues exactly one wire request, AND the typed failure is returned with `Retryable() == false` regardless of position, AND no retry occurs even if the attempt budget has not been exhausted

#### Scenario: R-AIS-041 / S-3 — After any semantic event has been emitted, no retry occurs *(pin: `R-CNF-019`, `R-AIP-010`, `R-AIP-011`, `R-FAIL-09`)*

- **GIVEN** the helper's seam: the helper runs **before** `make(chan ai.Event, ...)` (the carrier handover at `stream.go:237`) and therefore never observes a partial-output failure
- **WHEN** a reviewer inspects the helper's call site and the carrier handover ordering
- **THEN** no failure with `PartialOutput() == true` can reach the helper by construction — the typed error with its partial-output discriminator reaches the consumer through the terminal error event, AND the helper's absence of a retry branch on the partial-output path is observable in the helper's source

#### Scenario: R-AIS-041 / S-4 — Exhausted attempt budget returns the last failure wrapped to carry the count *(pin: `R-CNF-019`, `V-FAIL-15`)*

- **GIVEN** a real transport returning a retryable-flagged failure on every attempt and an attempt budget of `N` (`defaultMaxAttempts`)
- **WHEN** the helper exhausts its budget
- **THEN** exactly `N+1` wire requests have been issued, AND the typed failure returned carries the attempt count reachable via the cause chain (`errors.As`-style accessor), AND the final cause is the last wire response's typed failure

---

### R-AIS-042 — Backoff mechanics

Backoff is **bounded, injectable, and context-aware**. Three observable behaviors:

1. When the typed failure carries a `Retry-After` hint, the next attempt MUST wait that hint; the hint overrides any computed backoff.
2. When no `Retry-After` hint is present, backoff MUST grow within documented bounds — exponential with a documented cap, with jitter that is seeded and therefore assertable. The seed and the growth curve are test seams, not hidden implementation choices.
3. Backoff MUST wait on the caller-owned context. Cancellation during backoff aborts immediately; a remaining context budget smaller than the next delay short-circuits to the last error.

A documented maximum attempt count terminates retrying — **exactly `N+1` wire requests** are issued, then the last error is returned. Unbounded retry against a hard-down endpoint is the incident class this requirement prevents.

#### Scenario: R-AIS-042 / S-1 — Retry-After hint overrides computed backoff *(pin: `R-CNF-019`, `R-AEM-008`)*

- **GIVEN** a real transport returning `429` with a `Retry-After` hint of `H` seconds
- **WHEN** the helper observes the hint and computes the next delay
- **THEN** the next delay equals `H` exactly — never any computed exponential value — AND the hint is read from `failure.RetryAfter()` (the presence-typed accessor), AND absent hint falls back to computed backoff

#### Scenario: R-AIS-042 / S-2 — Computed backoff grows within documented bounds and jitter is assertable *(pin: `R-CNF-019`, `R-STK-009`)*

- **GIVEN** a real transport returning a retryable-flagged failure repeatedly, with no `Retry-After` hint, and an injectable jitter seed
- **WHEN** the helper computes the next delay across attempts
- **THEN** each delay is within the documented bounded range (exponential growth, capped), AND a fixed jitter seed produces an assertable sequence of delays across attempts, AND no delay exceeds the documented maximum

#### Scenario: R-AIS-042 / S-3 — Backoff waits on context; cancellation aborts immediately *(pin: `R-CNF-019`, `R-CNF-011`)*

- **GIVEN** a real transport returning a retryable-flagged failure with a long `Retry-After`, and a context with bounded remaining budget
- **WHEN** the caller cancels the context during a backoff wait
- **THEN** the helper aborts the wait immediately on the cancellation signal, AND no subsequent wire request is issued, AND the typed failure is returned with the last wire response's cause — never with an invented retry-exhausted category

#### Scenario: R-AIS-042 / S-4 — Bounded attempt count: exactly N+1 wire requests, then last error *(pin: `R-CNF-019`, `R-AIS-041` / S-4)*

- **GIVEN** a real transport returning a retryable-flagged failure on every attempt and `defaultMaxAttempts = N`
- **WHEN** the helper runs to exhaustion
- **THEN** exactly `N+1` wire requests are issued (counted via an instrumented transport), AND the returned typed failure is the `N+1`-th attempt's failure, AND no additional wire request follows

---

### R-AIS-043 — Replayability and the partial-output boundary marker

A retried request MUST re-issue from scratch with an identical body — byte-compared across attempts; nothing consumed on attempt one may corrupt attempt two. The attempt count and the final cause MUST both be reachable from the returned error chain via the cause-chain accessor. Each failed attempt's response body MUST be closed and drained before the next begins (the per-attempt connection leak hazard; exhausts the connection pool exactly during the rate-limit storm that triggered the retries).

**The partial-output boundary marker**: when the carrier has been handed over **and** the producer has emitted a semantic event **and** a retryable failure subsequently occurs, **no automatic retry occurs**. The retry helper's seam — running before the carrier handover — guarantees this by construction; this scenario pins the construction against future drift.

#### Scenario: R-AIS-043 / S-1 — Byte-identical replay across attempts *(pin: `R-CNF-019`, `R-ART-010`)*

- **GIVEN** a real transport that records each request body's bytes, and a scripted request that the helper retries
- **WHEN** the helper issues `N+1` attempts
- **THEN** every recorded request body is byte-identical to every other (no drift, no truncation, no re-encoding), AND the in-memory body slice is re-read via a fresh reader per attempt without mutation

#### Scenario: R-AIS-043 / S-2 — Attempt count and final cause are reachable from the error chain *(pin: `R-CNF-019`, `V-FAIL-15`)*

- **GIVEN** the helper has exhausted its attempt budget
- **WHEN** a consumer reads the returned error via the cause-chain accessor
- **THEN** the attempt count equals `N+1`, AND the final cause is reachable as a typed `*ai.Failure` carrying the original failure's category and delivery classification, AND no cause is dropped from the chain on exhaustion

#### Scenario: R-AIS-043 / S-3 — Per-attempt body drain (status path) *(pin: `R-CNF-019`, `R-AIS-033` / S-1, S-2, `R-ATS-003`)*

- **GIVEN** the status path (a non-2xx response with a body) for one attempt
- **WHEN** that attempt completes (success or typed failure) before the next attempt begins
- **THEN** the response body for that attempt is drained to a discarding sink and closed — supplied by composition with the existing `captureBody` drain (`capture.go:75–122`), not by a new defer in the helper — AND no attempt leaks a connection slot

#### Scenario: R-AIS-043 / S-4 — Partial-output boundary marker: after handover + emitted event, no retry *(pin: `R-CNF-019`, `R-AIP-010`, `R-AIP-011`, `V-FAIL-15`, `AG-15.1` item 2)*

- **GIVEN** a real transport returning one `2xx` text-event-stream frame, then a retryable-flagged failure mid-stream
- **WHEN** the caller invokes `Stream(ctx, req)` and the producer emits the first semantic event before seeing the failure
- **THEN** the helper's seam is past: the helper is no longer in the call path, AND no second wire request is issued (`httpClient.Do` count == 1, instrumented), AND the typed failure reaches the consumer through the terminal error event with `PartialOutput() == true`

---

### R-AIS-044 — Composed-bound ceiling (cross-layer contract)

The composed bound **"harness attempts × Layer 1 attempts"** is documented where both layers' readers find it. The Layer 1 multiplier `defaultMaxAttempts = 3` is documented in the helper's package doc comment (the file the helper ships in — `backend/agent/src/ai/internal/retry/doc.go` at apply time) and referenced verbatim by AG-15.2's composed-bound test (doc 0003 line 718). A reader from either Layer 1 (`backend/agent/src/ai/internal/retry/doc.go`) or Layer 2 (AG-15.2's test) finds the same number with the same formula.

This requirement carries **no production-code obligation** beyond the package doc comment's wording. Its purpose is the cross-layer visibility — the contract is binding *as documentation*, not as a runtime check.

#### Scenario: R-AIS-044 / S-1 — Layer 1 multiplier documented in helper's package doc comment *(pin: `R-CNF-019`, `AG-15.2` item 2)*

- **GIVEN** the helper's package doc comment file at apply time (`backend/agent/src/ai/internal/retry/doc.go`)
- **WHEN** a Layer 1 reader opens the file
- **THEN** the doc comment names `defaultMaxAttempts = 3` as the wire-request count per logical call (i.e. `N+1 = 4` wire requests when retries are exhausted), AND the composed-bound formula "harness attempts × Layer 1 attempts" appears in the same doc comment, AND the doc comment identifies AG-15.2 (doc 0003) as the cross-layer consumer

#### Scenario: R-AIS-044 / S-2 — Layer 2 reader sees the same number with the same formula *(pin: `R-CNF-019`, `AG-15.2` item 2)*

- **GIVEN** AG-15.2's harness-attempt test at apply time (doc 0003 line 718)
- **WHEN** a Layer 2 reader reads AG-15.2's test
- **THEN** the test quotes the Layer 1 multiplier verbatim from the helper's package doc comment, AND the composed-bound formula matches the helper's wording, AND a divergence between the two layers' wording is observable as a `[inspection]` test failure

---

## Pins / regressions

Every behavior leaf in this delta is pinned to a conformance or contract assertion it must not break.

| AI-35 subnode | Behavior leaf | Conformance / contract pin | Regression assertion |
| --- | --- | --- | --- |
| AI-35.1 S-1 | Retryable pre-stream retried | `R-CNF-019`, `R-ATS-002`, `R-AEM-008` | Real-producer retry test passes; no carrier is created |
| AI-35.1 S-2 | Terminal category never retried | `R-CNF-019`, `R-AEM-008`, `R-AEM-009` | Real-producer terminal test passes; exactly 1 wire request |
| AI-35.1 S-3 | No retry after partial output | `R-CNF-019`, `R-AIP-010`, `R-AIP-011`, `V-FAIL-09` | Helper seam unchanged; partial-output branch absent by construction |
| AI-35.1 S-4 | Exhausted budget returns last failure | `R-CNF-019`, `V-FAIL-15` | `N+1` wire requests; cause chain carries count + final cause |
| AI-35.2 S-1 | Retry-After overrides computed backoff | `R-CNF-019`, `R-AEM-008` | Hint read from `failure.RetryAfter()`; computed backoff skipped when hint present |
| AI-35.2 S-2 | Computed backoff bounded + seeded jitter | `R-CNF-019`, `R-STK-009` | Jitter seed produces assertable sequence; bounded growth |
| AI-35.2 S-3 | Backoff waits on context | `R-CNF-019`, `R-CNF-011` | Cancellation aborts immediately; remaining-budget short-circuit |
| AI-35.2 S-4 | Bounded attempt count: exactly N+1 | `R-CNF-019`, `R-AIS-041` / S-4 | Instrumented transport counts `N+1` requests exactly |
| AI-35.3 S-1 | Byte-identical replay | `R-CNF-019`, `R-ART-010` | Every recorded body byte-identical across attempts |
| AI-35.3 S-2 | Attempt count + final cause in chain | `R-CNF-019`, `V-FAIL-15` | Cause-chain accessor returns both |
| AI-35.3 S-3 | Per-attempt drain (status path) | `R-CNF-019`, `R-AIS-033`, `R-ATS-003` | Composition with `captureBody`; no new defer |
| AI-35.3 S-4 | Partial-output boundary marker | `R-CNF-019`, `R-AIP-010`, `R-AIP-011`, `V-FAIL-15`, `AG-15.1` | `httpClient.Do` count == 1 after partial-output failure |
| AI-35 (all) | Layer 1 multiplier cross-layer visibility | `R-CNF-019`, `AG-15.2` | Helper doc comment + AG-15.2 test wording agree |
| AI-35 (all) | Stdlib-only posture preserved | `R-STK-009`, `NFR-CNF-A`, `import_boundary_test.go:107–162` | `backend/agent/go.mod` zero requires; forward guard green |

---

## Out of scope

Aligned with proposal § 3 (Non-goals). Each item names the owner that does own it.

| Item | Owner |
| --- | --- |
| Harness-level (turn-level) retry and the harness predicate | **Layer 2 / AG-15.1** — doc 0003 lines 706–712 |
| Bounded backoff at the harness (timing injection, composed-bound test body) | **Layer 2 / AG-15.2** — doc 0003 lines 714–719 |
| Model failover seam (re-budgeting tokens, prices, cache prefix) | **Layer 2 / AG-15.3** — doc 0003 lines 721–726 |
| Mid-stream retry (after a semantic event has been emitted) | **Forever** — the helper's seam is pre-stream only; the typed partial-output failure reaches the harness via the terminal error event |
| Configuration surface for max attempts (`defaultMaxAttempts` promoted to `Config`) | **None** — `defaultMaxAttempts` is package-private; overridable in tests via helper parameter; stays constant until a workload demands configurability (AI-34 precedent — Q3) |
| New top-level Go dependencies | **Blocked** — the forward guard at `import_boundary_test.go:107–162` is mechanical; helper is stdlib-only |
| `CapRetry` registration in the conformance suite (the capability enum grows from `[8]` to `[9]`) | **Conformance delta** — `openspec/changes/cachicamas-ai-retry-policy/specs/ai-provider-conformance-suite/spec.md` (R-CNF-019 + `CapRetry`) |
| Helper signature shape (closure type, error-chain sibling cause type, attempt-budget parameter, sleep/now injection seams) | **Design phase** — spec states behavior; design owns Go identifiers |
| The named attempt-report cause type's exact identifier and accessor surface | **Design phase** — behavior is fixed; spec does not invent identifiers |
| Backoff curve (linear vs exponential) and jitter seed default | **Design phase** — spec pins "bounded + seeded"; design names the curve and seed |

---

## Acceptance criteria

1. `R-AIS-041` through `R-AIS-044` hold, each verified by its scenarios.
2. Every scenario marked `(pin)` cites the conformance or contract identifier it must not break; the merged AI-35 work leaves the conformance suite green and the contract assertions intact.
3. The canonical spec gains a `> **Amended 2026-08-07 (AI-35)**` blockquote at § 7 (failure delivery) at archive time, naming the in-adapter-loop seam and the four `R-AIS-04NN` requirements.
4. No Go identifier (type name, field name, method name, package path, signature) is invented in this spec — only existing identifiers from `stream.go`, `failure_map.go`, `capture.go`, `provider_failure.go`, `agenttest/`, and the conformance suite are cited.
5. The `defaultMaxAttempts = 3` constant appears in the helper's package doc comment at apply time, with the composed-bound formula referenced from AG-15.2's test (doc 0003 line 718).
6. No new top-level Go dependency is introduced — the helper is stdlib-only; `backend/agent/go.mod` zero requires; the forward guard `import_boundary_test.go:107–162` stays green.
7. `cd backend/agent && make test` is green under `-race`; `make lint` is clean.
