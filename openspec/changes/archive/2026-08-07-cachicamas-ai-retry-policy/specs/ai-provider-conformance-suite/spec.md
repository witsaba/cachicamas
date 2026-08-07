# Delta for `ai-provider-conformance-suite` — `CapRetry` and the retry-conformance case

> **Change**: `cachicamas-ai-retry-policy` · **Milestone**: AI-35 (doc 0002:2077–2144) · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-provider-conformance-suite/spec.md`](../../../../specs/ai-provider-conformance-suite/spec.md) § "AI-23.8 Optional-capability cases" and § "AI-23.6 The capability record" per the spec's amendment rule
> **Format**: RFC 2119 + Given/When/Then; per-scenario pins marked `(pin)` against the contract assertions the merged AI-35 work must not break
> **Conformance alignment**: `R-CNF-002` (capability expectation declared) cited as the new-capability registration rule; `R-CNF-003` (standing from AI-03) cited as the marking rule; `R-CNF-017` (totality) cited as the closed-list rebuild
> **Sources**: [proposal.md](../proposal.md) · [explore.md](../explore.md) · [doc 0002 § 2077–2144](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0003 § 687–726](../../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md)

---

## Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-retry-policy` |
| **Milestone** | AI-35 (doc 0002 lines 2077–2144) |
| **Capability amended** | `ai-provider-conformance-suite` (per proposal § Capabilities) |
| **Type** | Delta — **one `ADDED` requirement** (`R-CNF-019`); **one `MODIFIED` requirement** (`R-CNF-017`, totality over `[8]` → `[9]` capability array) — full block copied per the archive-replacement rule; **no `REMOVED`, no `RENAMED`** |
| **Capability enum change** | `CapRetry` added as the 9th member — `CAP-O-04` (optional). Array grows from `[8]` to `[9]`; `R-CNF-017` totality rebuild is mechanical (one new line in the totality loop). `R-CNF-002` capability-expectation declaration rule extended to declare `CapRetry` (alongside `CapReasoningContent`, `CapTokenCounting`, `CapCacheBoundary`) |
| **Conformance case** | `backend/agent/src/agenttest/conformance_retry.go` (NEW) — `registerConformanceCase("retry/auto_retry_up_to_documented_bound", CapRetry, <case body>)`. The case body's function name is the design/apply phase's; this delta pins the requirement it asserts |
| **Test runner** | `cd backend/agent && make test` (`go test -race -v ./...`) |
| **Module** | `backend/agent` — **layered** (not hexagonal), per ADR 0005 § D1 |

### What is being added

`R-CNF-019`: every adapter claiming `CapRetry` auto-retries retryable pre-stream failures up to a documented Layer 1 bound, never retries terminal-category failures regardless of position, and exposes the attempt count and final cause via the cause chain. Adapters not claiming `CapRetry` are not bound by `R-CNF-019` but MUST still satisfy `R-CNF-001` … `R-CNF-018` unchanged.

`CapRetry` registration: `Capability` enum gains `CapRetry` as `CAP-O-04`, declared after `CapCacheBoundary` and before `CapNone`. `capabilityFirst` … `capabilityEnd` range extends by one; `Capabilities()` enumerator grows by one; `capabilityNames` mapping adds `"CAP-O-04(retry)"`. `Optional()` switch extends to include `CapRetry`. A new `*bool` field on the existing `Factory` value (analogous to `Reasoning` / `TokenCounting` / `CacheBoundary`, the project's `nil`-disallowed idiom) carries the declaration — `nil` fails construction per `R-CNF-002` / `S-CNF-006`. The field's exact name is the design phase's; this spec pins only its shape and its nil-fail construction rule.

### Suite seams this delta rides on (all already merged at base `main` @ `238b9fa`)

| Seam | Location | Subnodes that ride on it |
| --- | --- | --- |
| `Capability` enum + `Capabilities()` enumerator + `Optional()` marking | `conformance_suite.go:43–145` | `R-CNF-019` registration; `R-CNF-017` totality rebuild |
| New `*bool` field on `Factory` (analogous to `Reasoning` / `TokenCounting` / `CacheBoundary`) + construction-time nil defect | `conformance_suite.go:271–277` (precedent); apply-time addition | `R-CNF-002` extension |
| `registerConformanceCase(name, capability, run)` registration pattern + `casesFor(capability)` discovery | `conformance_suite.go` + `conformance_scoped.go` | `retry/auto_retry_up_to_documented_bound` case registration |
| Capability-record totality (`[capabilityFirst]` … `[capabilityEnd]`) | `conformance_record.go` + `conformance_suite.go:119–125` | `R-CNF-017` rebuild |
| Conformance case body idiom (scripted transport, scripted status, drain + drain timeout) | `conformance_terminal.go` (precedent); `conformance_cancellation.go` (precedent) | New `retry/auto_retry_up_to_documented_bound` case body shape |
| Helper's documented Layer 1 bound | `internal/retry/doc.go` (apply-time); `defaultMaxAttempts = 3` per R-AIS-044 | `R-CNF-019` cross-layer citation |

---

## Retry capability (AI-35, dated 2026-08-07)

> **Amended 2026-08-07 (AI-35)** by `cachicamas-ai-retry-policy` (AI-35, Wave 5 — Harden). One optional capability `CapRetry` (CAP-O-04) added to the conformance suite's closed list; the `Capability` enum grows from `[8]` to `[9]`; `R-CNF-017` totality rebuild is mechanical. One requirement `R-CNF-019` added: every adapter claiming `CapRetry` MUST auto-retry retryable pre-stream failures up to the documented Layer 1 bound, MUST never retry terminal-category failures regardless of position, MUST re-issue with a byte-identical wire body across attempts, and MUST expose the attempt count and final cause via the cause chain. Adapters not claiming `CapRetry` MUST still satisfy `R-CNF-001` … `R-CNF-018` unchanged.

### ADDED Requirements

#### R-CNF-019 — Every adapter claiming `CapRetry` auto-retries retryable pre-stream failures up to a documented bound

An adapter that declares `CapRetry` (CAP-O-04) MUST satisfy all of the following:

1. **Every retryable-flagged pre-stream failure is retried up to the documented Layer 1 bound** (`defaultMaxAttempts = 3` per `R-AIS-044`, meaning up to `N+1 = 4` wire requests per logical call).
2. **Terminal-category failures — authentication, invalid request — are NEVER retried**, regardless of position.
3. **The per-attempt wire body is byte-identical to the original** across attempts (`R-AIS-043` / S-1).
4. **The attempt count and the final cause are reachable from the returned error chain** via the cause-chain accessor (`R-AIS-043` / S-2).

An adapter that does **not** declare `CapRetry` is not bound by `R-CNF-019` but MUST still satisfy `R-CNF-001` … `R-CNF-018`. The `CapRetry` declaration is mandatory input to the run (`R-CNF-002` extension: a `Factory` whose `CapRetry` declaration field is `nil` fails construction per `S-CNF-006`).

#### Scenarios

- **S-CNF-069** — Given a scripted transport that returns `429` with `Retry-After: 0` repeatedly for `N+1` attempts then a `2xx` text-event-stream, when the suite runs the `CapRetry` case against an adapter that declares `CapRetry`, then the case asserts every retryable-flagged pre-stream failure was retried up to the documented bound, AND the final stream is the `N+1`-th attempt's `2xx` stream, AND the typed failure (if returned) carries the attempt count via the cause chain.
- **S-CNF-070** — Given a scripted transport that returns `401` (authentication — `FailureCategoryAuthentication`) on a single attempt, when the suite runs the `CapRetry` case, then the case asserts no retry occurred (`httpClient.Do` count == 1, instrumented), AND the typed failure is returned with `Retryable() == false`, AND no second wire request follows.
- **S-CNF-071** — Given a scripted transport that returns one `2xx` text-event-stream frame, then a retryable-flagged failure mid-stream, when the suite runs the `CapRetry` case, then the case asserts `httpClient.Do` count == 1 (no automatic retry after a semantic event has been emitted), AND the typed failure reaches the consumer via the terminal error event with `PartialOutput() == true`.
- **S-CNF-072** — Given a scripted transport that records every request body's bytes, and a request the helper retries `N+1` times, when the suite runs the `CapRetry` case, then the case asserts every recorded body is byte-identical to every other (no drift, no re-encoding), AND the recorded request bodies match the original request's bytes verbatim.
- **S-CNF-073** — Given the helper has exhausted its attempt budget, when the suite reads the returned error via the cause-chain accessor, then the attempt count equals `N+1` and the final cause is reachable as a typed `*ai.Failure` carrying the original failure's category and delivery classification.
- **S-CNF-074** — Given an adapter whose `Factory` carries the new `CapRetry` declaration field as `false` (declared not offered), when the suite runs against it, then the record carries `CapRetry` = `absent` per `R-CNF-004` (skipped optional case reported, never silent), AND the run is not inconclusive on that entry, AND the adapter still satisfies `R-CNF-001` … `R-CNF-018`.
- **S-CNF-075** — Given a `Factory` whose `CapRetry` declaration field is `nil` (undeclared), when the suite runs, then it fails at construction naming the undeclared capability, per `R-CNF-002` / `S-CNF-006` — the same nil-disallowed discipline `Reasoning`, `TokenCounting`, `CacheBoundary` already enforce.

---

## MODIFIED Requirements

### Requirement: R-CNF-017 — The record is total over both closed lists, with standing taken from AI-03

The emitted record MUST identify its subject and its run, and MUST carry **exactly one entry per capability** in `CAP-R-01…05` and `CAP-O-01…04` — **nine entries, always**. Each entry's **standing** MUST be taken from AI-03 § 5 and § 6, never from the run. Each entry's **outcome** MUST come from the closed four-value set — `satisfied`, `absent`, `failed`, `not exercised` — with `absent` legal for **optional entries only**. A capability with no entry MUST be a defect in the run, not an absence. The record MUST carry no capability-specific detail, no model content, no credential and no raw provider text.

(Previously: `CAP-R-01…05` and `CAP-O-01…03` — eight entries, always. AI-35 adds `CapRetry` as `CAP-O-04`, the ninth optional capability.)

#### Scenarios

- **S-CNF-047** — Given any completed run, when the record is read, then it carries exactly **nine** entries, one per capability in the two closed lists (including `CAP-O-04(retry)`), each naming its capability, standing and outcome.
- **S-CNF-048** — Given a run whose subject offers no optional capability, when the record is read, then the **four** optional entries (`CAP-O-01…04`) are `absent` and no entry is `not exercised`.
- **S-CNF-049** — Given a run in which a required capability's cases failed, when the record is read, then that entry is `failed` — never `absent`, for which required standing has no legal value.
- **S-CNF-050** — Given a run that recorded a required capability as optional, when the record is validated, then it fails: standing is not the run's to supply.
- **S-CNF-051** — Given a published record, when it is inspected for capability-specific detail, model content or credentials, then it carries none.
- **S-CNF-076** *(new under AI-35)* — Given the `Capability` enum and `Capabilities()` enumerator, when a reviewer inspects them, then the closed list is `CAP-R-01…05` and `CAP-O-01…04` (nine entries, in declaration order), AND `CapNone` is deliberately excluded from the enumerator (unchanged from the eight-member rule), AND `CapRetry` is declared after `CapCacheBoundary` and before `CapNone` to keep `capabilityFirst` … `capabilityEnd` contiguous.

---

## REMOVED Requirements

None.

## RENAMED Requirements

None.

---

## `CapRetry` registration (apply-time mechanics — `[inspection]`-level evidence)

The capability enum grows from `[8]` to `[9]`. The following changes are mechanical and applied at apply time; this section records them so a reviewer can verify the canonical spec's structure after archive.

| Change | Before (8 members) | After (9 members) |
| --- | --- | --- |
| `Capability` enum declaration | `CapStreamingText`, `CapToolCalls`, `CapCompletionMetadata`, `CapCancellation`, `CapTypedFailures`, `CapReasoningContent`, `CapTokenCounting`, `CapCacheBoundary`, `CapNone` | Same, plus `CapRetry` between `CapCacheBoundary` and `CapNone` |
| `capabilityNames` mapping | Eight entries | Nine entries; `CapRetry` maps to `"CAP-O-04(retry)"` |
| `capabilityFirst` … `capabilityEnd` | `CapStreamingText` … `CapCacheBoundary + 1` | `CapStreamingText` … `CapRetry + 1` (range grows by one) |
| `Capabilities()` enumerator | `make(..., int(capabilityEnd-capabilityFirst))` returns eight entries | Same shape returns nine entries |
| `Optional()` switch | `CapReasoningContent`, `CapTokenCounting`, `CapCacheBoundary` | Same plus `CapRetry` (four optional values) |
| New `*bool` field on `Factory` (the `CapRetry` declaration field) | (none) | Added alongside `Reasoning`, `TokenCounting`, `CacheBoundary`; `nil` fails construction per `R-CNF-002` / `S-CNF-006`. The field's exact name is the design phase's; this spec pins only its shape and its nil-fail construction rule. |
| `RunConformance` construction-time nil defect | Names `CapReasoningContent`, `CapTokenCounting`, `CapCacheBoundary` | Same plus `CapRetry` (four nil-disallowed capabilities) |
| `standingOf(CapRetry)` | (none) | `StandingOptional` per `R-CNF-003` / AI-03 § 11 |
| Totality loop in `CapabilityRecord` initialisation | `[capabilityEnd - capabilityFirst]` entries initialised to `OutcomeNotExercised` | Same shape; one new line covers `CapRetry` |
| `casesFor(CapRetry)` | (none) | Returns the registered `retry/auto_retry_up_to_documented_bound` case; `[scope]`-level discoverability |

---

## Conformance case body shape (apply-time — `[inspection]`-level evidence)

The new case body registers under `CapRetry` and asserts each of `S-CNF-069` … `S-CNF-075` against a scripted transport. The case body's function name and closure shape are the design/apply phase's; this spec records the case body's scenario-to-scripted-transcript mapping:

| Scenario in the case body | What it scripts | What it asserts |
| --- | --- | --- |
| Retryable-pre-stream-retried (S-CNF-069) | A transport returning `429` with `Retry-After: 0` repeatedly for `N+1` attempts then a `2xx` text-event-stream | `N+1` wire requests; final stream is the `2xx`; on exhaustion the typed failure carries the count via the cause chain |
| Terminal-category-never-retried (S-CNF-070) | A transport returning `401` once | Exactly `1` wire request; typed failure with `Retryable() == false`; no second wire request |
| Partial-output-boundary-marker (S-CNF-071) | A transport emitting one `2xx` text-event-stream frame, then a retryable-flagged failure mid-stream | `httpClient.Do` count == 1; typed failure delivered via terminal error event with `PartialOutput() == true` |
| Byte-identical-replay (S-CNF-072) | A transport recording every request body's bytes | Every recorded body byte-identical to every other |
| Attempt-count-and-final-cause (S-CNF-073) | The exhausted-budget scenario above | Cause-chain accessor returns the count and the typed final cause |
| CapRetry-absent (S-CNF-074) | An adapter that does NOT declare `CapRetry` | Record entry `CapRetry` = `absent` per `R-CNF-004`; `R-CNF-001` … `R-CNF-018` still satisfied |
| Factory-nil-defect (S-CNF-075) | A `Factory` whose `CapRetry` declaration field is `nil` | Construction fails naming the undeclared capability per `R-CNF-002` / `S-CNF-006` |

The case body reuses existing failure-mapping and drain infrastructure; no new Layer 1 vocabulary is invented. The case is required when `CapRetry` is declared; otherwise it is skipped per `R-CNF-004`.

---

## Pins / regressions

Every behavior leaf in this delta is pinned to a contract assertion it must not break.

| AI-35 conformance subnode | Behavior leaf | Contract pin | Regression assertion |
| --- | --- | --- | --- |
| R-CNF-019 S-1 | Retryable pre-stream retried up to bound | `R-AIS-041` / S-1, `R-AIS-042` / S-4, `R-AIS-044` | Real producer test green; documented bound matches |
| R-CNF-019 S-2 | Terminal category never retried | `R-AIS-041` / S-2, `R-AEM-008`, `R-AEM-009` | Exactly 1 wire request for `401` / `400` / `404` / `422` |
| R-CNF-019 S-3 | No retry after partial output | `R-AIS-043` / S-4, `R-AIP-010`, `R-AIP-011`, `V-FAIL-15`, `AG-15.1` | Helper seam unchanged; `httpClient.Do` count == 1 after partial output |
| R-CNF-019 S-4 | Byte-identical replay | `R-AIS-043` / S-1, `R-ART-010` | Every recorded body byte-identical |
| R-CNF-019 S-5 | Attempt count + final cause in chain | `R-AIS-043` / S-2, `V-FAIL-15` | Cause-chain accessor returns both |
| R-CNF-019 S-6 | CapRetry absent is reported, not silent | `R-CNF-002`, `R-CNF-004` | Record entry `absent`; run not inconclusive |
| R-CNF-019 S-7 | Factory nil Retry fails construction | `R-CNF-002`, `S-CNF-006` | Same nil-disallowed discipline as `Reasoning` / `TokenCounting` / `CacheBoundary` |
| R-CNF-017 (modified) | Totality over nine capabilities | `R-CNF-003`, `R-CNF-018` | Record carries nine entries; `Capabilities()` enumerator returns nine |
| Conformance case | Real producer + scripted transport | `R-AIS-041`, `R-AIS-042`, `R-AIS-043`, `R-AIS-044` | Both `openaicompat` and OpenRouter wrapper pass the case (wrapper inherits helper transparently via `*openaicompat.Client` embed) |

---

## Out of scope

Aligned with proposal § 3 (Non-goals). Each item names the owner that does own it.

| Item | Owner |
| --- | --- |
| Harness-level retry predicate and bounded backoff | **Layer 2 / AG-15.1, AG-15.2** — doc 0003 lines 706–719 |
| Model failover seam | **Layer 2 / AG-15.3** — doc 0003 lines 721–726 |
| Helper's typed signature (closure type, attempt-report cause type, jitter seed default, sleep/now injection seams) | **Design phase** — this spec pins behavior; design owns Go identifiers |
| Conformance case body (the closure that builds the scripted transport) | **Apply phase** — this spec records the case shape; apply authors the Go |
| Mid-stream retry assertion shape (the "no retry after partial output" scenario's exact instrumented-RoundTripper count) | **Apply phase** — `R-AIS-043` / S-4 pins the behavior; apply authors the test |
| New top-level Go dependencies | **Blocked** — conformance case reuses existing failure-mapping and drain infrastructure; `backend/agent/go.mod` zero requires (`NFR-CNF-A`) |
| Layer 1 conformance test body (the real-producer tests at `a_i-35_{1,2,3}_test.go`) | **Apply phase** — `R-AIS-041` … `R-AIS-044` pin the behavior; the conformance case here asserts the same behavior against any adapter; the real-producer tests in `ai-stream-lifecycle/spec.md` prove the producer matches |
| The new `CapRetry` declaration field's name and `*bool` shape | **Design phase** — `*bool` precedent from `Reasoning` / `TokenCounting` / `CacheBoundary` is the project's idiom; design names the field and confirms the shape |

---

## Acceptance criteria

1. `R-CNF-019` holds, verified by its scenarios `S-CNF-069` … `S-CNF-075`.
2. `R-CNF-017` (modified) holds: the record carries nine entries (`CAP-R-01…05` + `CAP-O-01…04`); `Capabilities()` returns nine values; the totality rebuild is mechanical and observable by inspection.
3. The `Capability` enum grows from `[8]` to `[9]` with `CapRetry` declared as `CAP-O-04` between `CapCacheBoundary` and `CapNone`; `capabilityNames` mapping adds `"CAP-O-04(retry)"`; `Optional()` includes `CapRetry`.
4. A new `*bool` field on `Factory` (the `CapRetry` declaration field) is declared alongside `Reasoning` / `TokenCounting` / `CacheBoundary`; `nil` fails construction per `R-CNF-002` / `S-CNF-006`. The field's exact name is the design phase's; this spec pins only the shape and the nil-fail rule.
5. The conformance case body registers under `CapRetry` and runs against both `openaicompat` (real producer) and the OpenRouter wrapper (which inherits the helper transparently); both pass.
6. No Go identifier (type name, field name, method name, package path, signature) is invented in this spec — only existing identifiers from `conformance_suite.go`, `conformance_terminal.go`, `conformance_cancellation.go`, and the producer's `failure_map.go` / `provider_failure.go` are cited.
7. The `defaultMaxAttempts = 3` constant is referenced from `R-CNF-019`'s scenarios with the same formula `R-AIS-044` pins (the helper's package doc comment at apply time); a divergence between the conformance spec's wording and the helper's doc comment is observable as a `[inspection]` test failure.
8. No new top-level Go dependency is introduced — `NFR-CNF-A` holds unmodified; `backend/agent/go.mod` zero requires; the forward guard `import_boundary_test.go:107–162` stays green.
9. `cd backend/agent && make test` is green under `-race`; `make lint` is clean.
