# Spec — mapping HTTP and provider failures into the AI-19 taxonomy

> **Change**: `cachicamas-ai-provider-error-mapping`
> **Milestone**: AI-32 · **Nodes**: AI-32.1 … AI-32.5, all `[leaf]`
> **Phase**: spec (delta — new capability, ADDED only)
> **Canonical spec**: `openspec/specs/ai-provider-error-mapping/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Date**: 2026-08-04 · **Project**: cachicamas (witsaba)
> **Requirement IDs**: `R-AEM-0NN` · **Scenario IDs**: `S-AEM-0NN`
> **Binding predecessors, cited by identifier and never modified**: [`ai-provider-errors`](../../../../specs/ai-provider-errors/spec.md) (AI-19 — the closed nine-member vocabulary, `FailureReport`, `PreStreamFailure`, `MidStreamFailure`, `ErrorEvent`, R-AIP-009's redaction posture) · [`ai-stream-decoder`](../../cachicamas-ai-stream-decoder/specs/ai-stream-decoder/spec.md) (AI-27) · [`ai-stream-lifecycle`](../../../../specs/ai-stream-lifecycle/spec.md) (AI-20.2's pre-stream path) · [doc 0002](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) §§ AI-32 … AI-32.5 · [`proposal.md`](../../proposal.md) · [`citations.md`](../../citations.md)
> **Wire-fact source of record**: [`citations.md`](../../citations.md), evidence items **E1 … E6**, pinned `openai-openapi` commit `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439`.
> **Depends on**: AI-19, AI-27 (stage 1); AI-28.1 (stage 2). **Blocks**: AI-28.6, AI-35.

---

## ADDED Requirements

## Purpose

> **Revision note (design-validation corrective, 2026-08-04).** The premise below previously read "`openaicompat` today **names** two decoder categories (`Category()`) and constructs no `ai.Failure` of any kind". That was factually false: `policy.go`'s `refuse` already constructs `ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnsupportedCapability, …})` — AI-26.6's translation-time capability refusal. The corrected premise distinguishes **translation-time** construction (which exists) from **wire-side** construction (which does not). No requirement is weakened; requirement and scenario counts are unchanged.

`openaicompat` today constructs exactly **one** `ai.Failure`, and it is constructed at **translation time**: AI-26.6's capability refusal in `policy.go`, which returns `ai.PreStreamFailure` with `ai.FailureCategoryUnsupportedCapability` when the request asks for a feature this adapter refuses. That failure is decided entirely from the *request*, before any provider was contacted.

What the package has never done is construct a failure **from the wire**. `Category()` merely *names* a category for two decoder sentinels and constructs nothing; and no HTTP status, response header, vendor error body, in-band error frame, disconnect or expired deadline has ever been mapped into AI-19's taxonomy. The taxonomy is therefore unreachable from the wire, and **AI-32 is the first wire-side constructor**. This spec constrains what that mapping MUST guarantee.

The translation-time refusal and the wire-side mapping stay distinct throughout: this milestone neither replaces, reroutes nor amends AI-26.6's refusal.

Three further distinctions shape every requirement below and are stated once here.

**Status first; `type` and `code` are opaque.** The vendor error body is `{"error": {type, message, param, code}}` with all four fields required and `param`/`code` nullable (**cited, E1**), and `type`/`code` are **unconstrained strings with no enum anywhere** (**cited, E5**). Classification therefore matches on the HTTP status alone; `type` and `code` travel as opaque diagnostic strings and MUST NOT be matched against any value vocabulary.

**The status table is dialect-conventional, not spec-derived.** `POST /chat/completions` declares **no non-200 response at all**, and the whole pinned spec declares no `401`, `403`, `429` or `5xx` (**cited negative, E2**). Every row of the table in `R-AEM-002` is therefore labelled **dialect-conventional** and carries a fixture-pin obligation, in AI-24 § 3 rule 1's form: a claim labelled dialect-conventional is cited to the vendor's documented or observed behavior **with its fixture-pin obligation named**. An unlabelled wire claim is a defect; attributing a dialect convention to the pinned specification is a defect.

**A flag, never a retry.** Retryability is a classification (R-AIP-007). AI-35 owns acting on it. This milestone exports no backoff, attempt counter or failover identifier, and does not widen `ai.FailureCategory` — the vocabulary stays closed at nine members.

Requirement count: **19** (`R-AEM-001` … `R-AEM-019`). Scenario count: **72** (`S-AEM-001` … `S-AEM-072`) — **67 [test]**, **5 [inspection]**.

*Counted mechanically over the scenario bullets, not estimated. The five `[inspection]` scenarios are `S-AEM-045`, `S-AEM-067`, `S-AEM-068`, `S-AEM-069`, `S-AEM-070`. A reconciliation that finds a different split is reading a stale revision.*

*(Counts updated 2026-08-08 by AI-36, which appended `R-AEM-019` and its two `[test]` scenarios `S-AEM-071`/`S-AEM-072`. AI-32's own counts — 18 requirements, 70 scenarios, 65 `[test]` — are unchanged as a statement about AI-32; the totals above are for this file as it now stands.)*

## Requirement ownership by node and stage

| Node | Stage | Requirements |
| --- | --- | --- |
| AI-32.1 — status taxonomy | 1 | `R-AEM-001` … `R-AEM-007` |
| AI-32.4 — retry metadata | 1 | `R-AEM-008`, `R-AEM-009` |
| AI-32.2 — mid-stream error frames | 2 | `R-AEM-010`, `R-AEM-011` |
| AI-32.3 — disconnects and deadlines | 2 | `R-AEM-012` … `R-AEM-014` |
| AI-32.5 — bounded, sanitized capture | 2 | `R-AEM-015`, `R-AEM-016` |
| Charter boundary + citation integrity | 1 | `R-AEM-017`, `R-AEM-018` |

**Stage 2 requirements are written as observable behavior only.** AI-28.1's producer surface does not exist yet; no stage-2 requirement names a producer function, type or signature. Each states what an observer of the emitted event stream and the returned error MUST see.

## Definitions used by this spec

- **The mapper** — the component this milestone ships. It converts a wire failure into a constructed `*ai.Failure`; it owns no retry policy. It is the package's **wire-side** constructor, and is distinct from AI-26.6's pre-existing **translation-time** capability refusal in `policy.go`, which this milestone neither invokes nor alters.
- **The pre-stream path** — a failure returned directly, before any carrier was handed over (`ai.PreStreamFailure`, `Delivery() == DeliveryPreStream`, `PartialOutput() == false` by construction).
- **The mid-stream path** — a failure carried as the stream's terminal error event payload (`ai.MidStreamFailure` wrapped by `ai.ErrorEvent`).
- **A terminal error event** — an `ai.Event` whose `ErrorPayload()` reports `(failure, true)`, delivered as the last event of the stream.
- **The vendor error body** — either `{"error": {…}}` or a bare `{…}` carrying the same four fields; the pinned spec uses both wrappers where it declares errors at all (**cited, E1 + negative finding 6**).
- **The vendor label** — the body's `type`, or its `code` when `type` is absent, empty or null; an opaque string, never an enum member.
- **The retry-metadata carrier** — an adapter-local, exported value reaching the constructed failure through the error chain, retrievable with `errors.As`. It is adapter-local precisely because `ai.FailureReport` is closed and MUST NOT be widened.
- **The capture limit** — the fixed, documented maximum number of body bytes retained as a diagnostic. It MUST NOT exceed 64 KiB.
- **The truncation marker** — a fixed, documented ASCII suffix appended to a capture that hit the limit, present on no capture that did not.
- **A pinning fixture** — a recorded byte-level fixture in this change's fixture set that names the provider and the observed status, header or frame, and that a `[test]` scenario replays. A dialect-conventional claim with no pinning fixture is unsatisfied.
- **Scenario kind** — every scenario below is marked **[test]** (runnable and failable under `make test`) or **[inspection]** (reviewer-checkable over the artifact or shipped source). Under Strict TDD every **[test]** scenario MUST be demonstrated failing before the code satisfying it exists. A scenario that cannot be shown failing first is not a **[test]** scenario and MUST NOT be marked as one.
- **A distinguishing fixture** — a fixture whose payload makes an unimplemented behavior fail. Where a scenario below names one, that shape is normative, not advisory: it is the guard against a vacuous pass.

---

## AI-32.1 — Status taxonomy `[leaf]` · stage 1

### R-AEM-001 — A non-2xx response becomes a constructed pre-stream failure

Given a provider response whose status is not in the 2xx class, the mapper MUST return a non-nil `*ai.Failure` constructed through `ai.PreStreamFailure`, and MUST NOT return a bare error, a string, or an `ai.Failure` constructed on the mid-stream path. The failure MUST report `Delivery() == ai.DeliveryPreStream`, `PartialOutput() == false`, and a `StatusClass()` equal to the response status divided by 100, present.

A 2xx response MUST NOT produce a failure of any kind from this path.

#### Scenarios

- **S-AEM-001** *[test]* — Given a 401 response, when the mapper handles it, then a non-nil `*ai.Failure` is returned whose `Delivery()` is `ai.DeliveryPreStream` and whose `PartialOutput()` is false.
- **S-AEM-002** *[test]* — Given responses with statuses 400, 403, 404, 408, 422, 429, 500, 503 and 504, when each is mapped, then each failure's `StatusClass()` reports `(status/100, true)` and never `(0, false)`.

### R-AEM-002 — The status → category table `[dialect-conventional]`

Each row below MUST hold. Every row is **dialect-conventional** — the pinned specification declares none of these statuses (**cited negative, E2**) — and each MUST be pinned by a pinning fixture replayed by its scenario. No row MAY be inferred from the pinned specification.

| Status | `FailureCategory` | `Retryable()` | doc 0002 AI-32.1 item 1 class |
| --- | --- | --- | --- |
| 401 | `FailureCategoryAuthentication` | false | authentication |
| 403 | `FailureCategoryAuthorization` | false | permission |
| 400, 404, 422 | `FailureCategoryUnknown` | false | not-found or invalid |
| 408 | `FailureCategoryTimeout` | true | timeout |
| 429 | `FailureCategoryRateLimit` | true | rate limit |
| 500, 502, 503 | `FailureCategoryUnavailable` | true | overload or unavailable |
| 504 | `FailureCategoryTimeout` | true | timeout |

**Recorded compromise — the not-found/invalid row.** AI-19's nine-member vocabulary has no invalid-request or not-found member, and widening a shipped, promoted Layer 1 contract is out of this milestone's charter. `FailureCategoryUnknown` is its documented home: "the recorded outcome for a provider failure this vocabulary does not recognise", with the raw provider label surviving (R-AIP-006). This is the same recorded-compromise form `ErrFrameTooLarge` already uses in `errors.go`, and it reopens only when the taxonomy owner appends a member as its own change.

`FailureCategoryUnsupportedCapability` MUST NOT be produced from a status code alone: that member asserts the request used a capability the provider does not support, which a bare status does not establish. In this package that member remains AI-26.6's translation-time refusal's alone — decided from the request before any provider was contacted — and the wire-side mapper MUST NOT produce it.

#### Scenarios

- **S-AEM-003** *[test]* — Given a 401 pinning fixture, when it is mapped, then `Category()` is `ai.FailureCategoryAuthentication` and `errors.Is(failure, ai.ErrAuthentication)` holds.
- **S-AEM-004** *[test]* — Given a 403 pinning fixture, when it is mapped, then `Category()` is `ai.FailureCategoryAuthorization` and `errors.Is(failure, ai.ErrAuthorization)` holds.
- **S-AEM-005** *[test]* — Given 400, 404 and 422 pinning fixtures, when each is mapped, then `Category()` is `ai.FailureCategoryUnknown` for all three and `errors.Is(failure, ai.ErrUnknownFailure)` holds.
- **S-AEM-006** *[test]* — Given a 429 pinning fixture, when it is mapped, then `Category()` is `ai.FailureCategoryRateLimit` and `errors.Is(failure, ai.ErrRateLimited)` holds.
- **S-AEM-007** *[test]* — Given 500, 502 and 503 pinning fixtures, when each is mapped, then `Category()` is `ai.FailureCategoryUnavailable` for all three.
- **S-AEM-008** *[test]* — Given 408 and 504 pinning fixtures, when each is mapped, then `Category()` is `ai.FailureCategoryTimeout` for both.
- **S-AEM-009** *[test]* — Given the whole table driven as one table test, when every row is mapped, then no row produces `ai.FailureCategoryUnsupportedCapability` and no row produces the zero category.
- **S-AEM-010** *[test]* — Given a 200 response, when it is offered to the mapper, then no failure is produced and no error is returned.

### R-AEM-003 — Retryability follows the taxonomy, and is a flag only

`Retryable()` MUST be true for exactly `FailureCategoryRateLimit`, `FailureCategoryUnavailable` and `FailureCategoryTimeout`, and false for every other constructed category on this path — client-contract failures are terminal. `Retryable()` MUST be derivable from the category alone, independently of whether a retry-after hint was reported.

#### Scenarios

- **S-AEM-011** *[test]* — Given every row of `R-AEM-002`, when each failure's `Retryable()` is read, then it is true for the rate-limit, unavailable and timeout rows and false for all others.
- **S-AEM-012** *[test]* — Given a 429 response carrying no `Retry-After` header, when it is mapped, then `Retryable()` is true and `RetryAfter()` reports `(0, false)` — the flag does not depend on the hint.
- **S-AEM-013** *[test]* — Given a 401 response carrying a `Retry-After: 30` header, when it is mapped, then `Retryable()` is false while `RetryAfter()` still reports `(30s, true)` — the hint does not set the flag.

### R-AEM-004 — An unparseable or absent body still maps

Given a non-2xx response whose body is not valid JSON, is empty, is valid JSON of the wrong shape, or is absent entirely, the mapper MUST still construct a failure whose category and retryability come from the status per `R-AEM-002`. Body parse failure MUST NOT be reported as the failure's category, MUST NOT abort mapping, and MUST NOT panic. The bytes actually read MUST be retained as a bounded diagnostic reachable through the error chain and MUST NOT appear in `Failure.Error()`.

#### Scenarios

- **S-AEM-014** *[test]* — Given a 429 response whose body is the bytes `<html>rate limited</html>`, when it is mapped, then `Category()` is `ai.FailureCategoryRateLimit`, `Retryable()` is true, and no error other than the constructed failure is returned.
- **S-AEM-015** *[test]* — Given a 503 response with a zero-length body, when it is mapped, then `Category()` is `ai.FailureCategoryUnavailable` and `RawLabel()` is empty.
- **S-AEM-016** *[test]* — Given a 500 response whose body is `{"error": 12345}` (right key, wrong type), when it is mapped, then mapping succeeds with the status-derived category and does not panic.

### R-AEM-005 — Undocumented statuses map through the fallback

Given a non-2xx status absent from `R-AEM-002`'s table, the mapper MUST map it through its status class without crashing: class 5 → `FailureCategoryUnavailable`, retryable; classes 1, 3 and 4 → `FailureCategoryUnknown`, not retryable. `StatusClass()` MUST report the observed class. A status outside 100–599 MUST produce `FailureCategoryUnknown`, not retryable, with `StatusClass()` reporting `(0, false)` rather than an out-of-range value that would fail AI-19 construction.

#### Scenarios

- **S-AEM-017** *[test]* — Given a 418 response, when it is mapped, then `Category()` is `ai.FailureCategoryUnknown`, `Retryable()` is false and `StatusClass()` reports `(4, true)`.
- **S-AEM-018** *[test]* — Given a 599 response, when it is mapped, then `Category()` is `ai.FailureCategoryUnavailable`, `Retryable()` is true and `StatusClass()` reports `(5, true)`.
- **S-AEM-019** *[test]* — Given a 302 response, when it is mapped, then `Category()` is `ai.FailureCategoryUnknown` and `StatusClass()` reports `(3, true)`.
- **S-AEM-020** *[test]* — Given a synthetic status of 0 and one of 999, when each is mapped, then each yields `ai.FailureCategoryUnknown` with `StatusClass()` reporting `(0, false)`, and neither returns a construction error.

### R-AEM-006 — The vendor body is parsed tolerantly and its labels stay opaque

The mapper MUST accept both the wrapped body `{"error": {…}}` and the bare body `{…}` carrying the same fields (**cited, E1**; the pinned spec itself uses both wrappers, **negative finding 6**). It MUST tolerate `param` and `code` being JSON `null` (**cited, E1**).

`RawLabel` MUST be populated from the vendor label — `type`, falling back to `code` when `type` is absent, empty or null — subject to AI-19's drop-whole 64-byte sanitization. The mapper MUST NOT compare `type` or `code` against any value vocabulary, and no category MUST depend on their values (**cited, E5**: they are unconstrained strings with no enum). The body's `message` MUST NOT influence the category and MUST NOT reach `Failure.Error()`.

#### Scenarios

- **S-AEM-021** *[test]* — Given a 429 body `{"error":{"type":"provider_specific_throttle","message":"slow down","param":null,"code":null}}`, when it is mapped, then `RawLabel()` is `provider_specific_throttle` and `Category()` is still `ai.FailureCategoryRateLimit`.
- **S-AEM-022** *[test]* — Given the same four fields delivered as a bare object with no `error` wrapper, when it is mapped, then `RawLabel()` is identical to `S-AEM-021`'s.
- **S-AEM-023** *[test]* — Given a body whose `type` is `""` and whose `code` is `"insufficient_quota"`, when it is mapped, then `RawLabel()` is `insufficient_quota`.
- **S-AEM-024** *[test]* — Given two 400 bodies whose `type` values are `invalid_request_error` and `some_vendor_invented_type`, when both are mapped, then both yield the identical category `ai.FailureCategoryUnknown` — the label changes, the classification does not.
- **S-AEM-025** *[test]* — Given a body whose `message` is `the model gpt-x does not exist`, when the failure is rendered with `Error()` and with `fmt.Sprintf("%v", failure)`, then neither rendering contains any substring of that message.

### R-AEM-007 — Request-id capture is best-effort `[dialect-conventional]`

The pinned specification documents no request-identifier response header for Chat Completions (**cited negative, E6**). Capture is therefore **dialect-conventional** and MUST be pinned by a fixture: when a response carries an `x-request-id` header, its value MUST be offered as `FailureReport.RequestID`; its absence MUST be tolerated with `RequestID()` empty and no error. Header-name matching MUST be case-insensitive. AI-19's drop-whole bound MUST be observable: an over-long or control-character-bearing value yields an empty `RequestID()` while construction still succeeds — a drop, never a truncation.

#### Scenarios

- **S-AEM-026** *[test]* — Given a 500 response carrying `x-request-id: req_abc123`, when it is mapped, then `RequestID()` is `req_abc123`.
- **S-AEM-027** *[test]* — Given the same response with the header spelled `X-Request-Id`, when it is mapped, then `RequestID()` is `req_abc123`.
- **S-AEM-028** *[test]* — Given a 500 response with no request-id header, when it is mapped, then `RequestID()` is empty and mapping returns no error.
- **S-AEM-029** *[test]* — Given a request-id header value of 65 bytes whose first 64 bytes are `A`-repeated and whose 65th is `Z`, when it is mapped, then `RequestID()` is exactly empty — neither the 64-byte prefix nor any part of the value survives.

---

## AI-32.4 — Retry metadata `[leaf]` · stage 1

### R-AEM-008 — `Retry-After` arrives typed, in both RFC-defined forms

`Retry-After` is defined by **RFC 9110 § 10.2.3** as either `delay-seconds` (a non-negative integer count of seconds) or an `HTTP-date`. That grammar is **RFC-defined**; the header's **presence on this vendor's failures is dialect-conventional** (**cited negative, E3**: zero occurrences of `Retry-After` in the pinned spec) and MUST be pinned by fixture.

Both forms MUST parse into `ai.FailureReport.RetryAfter` via `ai.Delay`. The HTTP-date form MUST yield the non-negative duration from the response observation instant to the given date; a date at or before that instant MUST yield `ai.Delay(0)` — reported, present, and distinct from absent. A malformed, negative or unparseable value MUST yield an absent hint (`RetryAfter()` reports `(0, false)`) without failing construction and without changing the category.

#### Scenarios

- **S-AEM-030** *[test]* — Given a 429 response with `Retry-After: 30`, when it is mapped, then `RetryAfter()` reports `(30 * time.Second, true)`.
- **S-AEM-031** *[test]* — Given a 429 response with `Retry-After: 0`, when it is mapped, then `RetryAfter()` reports `(0, true)` — a reported immediate retry, not an absent hint.
- **S-AEM-032** *[test]* — Given a 503 response with `Retry-After: Wed, 21 Oct 2026 07:28:00 GMT` observed at `07:27:00 GMT` the same day, when it is mapped, then `RetryAfter()` reports `(60 * time.Second, true)`.
- **S-AEM-033** *[test]* — Given the same header observed at `07:29:00 GMT`, when it is mapped, then `RetryAfter()` reports `(0, true)` and never a negative duration.
- **S-AEM-034** *[test]* — Given `Retry-After: soon` and `Retry-After: -5`, when each is mapped, then `RetryAfter()` reports `(0, false)` for both, and both failures still carry their status-derived category.
- **S-AEM-035** *[test]* — Given a 429 response with no `Retry-After` header at all, when it is mapped, then `RetryAfter()` reports `(0, false)`, distinguishable from `S-AEM-031`'s reported zero.

### R-AEM-009 — Rate-limit telemetry lands in an adapter-local carrier

Rate-limit telemetry headers are undocumented by the pinned specification (**cited negative, E3**: `x-ratelimit` → 0 hits), so their names and presence are **dialect-conventional** and MUST be pinned by fixture. Captured telemetry MUST be exposed as a retry-metadata carrier reachable from the constructed failure with `errors.As`, and MUST NOT widen `ai.FailureReport` or `ai.Failure`.

Capture MUST be allowlist-driven over non-secret, support-relevant headers, machine-readable rather than a single opaque blob, and MUST tolerate any subset being absent. No captured header value MAY appear in `Failure.Error()`. A header outside the allowlist MUST NOT be captured, so a credential-bearing header cannot enter the carrier by accident.

#### Scenarios

- **S-AEM-036** *[test]* — Given a 429 response carrying `x-ratelimit-limit-requests`, `x-ratelimit-remaining-requests` and `x-ratelimit-reset-requests`, when it is mapped, then `errors.As` retrieves a carrier reporting all three values individually addressable, not as one concatenated string.
- **S-AEM-037** *[test]* — Given a 429 response carrying only `x-ratelimit-remaining-requests`, when it is mapped, then the carrier is retrievable and reports that one value with the others absent — absence is not an error.
- **S-AEM-038** *[test]* — Given a 429 response carrying `authorization: Bearer sk-planted-AEM038` alongside the telemetry headers, when it is mapped, then no rendering of the carrier or the failure contains `sk-planted-AEM038`.
- **S-AEM-039** *[test]* — Given a mapped 429 failure with full telemetry, when `Error()` is read, then it equals exactly `provider failure: rate_limit` and contains no header name or value.

---

## AI-32.2 — Mid-stream error frames `[leaf]` · stage 2

### R-AEM-010 — An in-band error frame terminates the stream with a typed terminal event `[dialect-conventional]`

Neither schema nor prose in the pinned specification defines or mentions an in-stream error frame for Chat Completions (**cited negative, E4**), so this behavior is **dialect-conventional** and MUST be pinned by a byte-level transcript fixture.

When the decoded stream delivers a frame carrying a vendor error payload, the stream MUST terminate with a terminal error event whose payload is an `ai.MidStreamFailure`. The event MUST carry the partial-output discriminator truthfully — true when a normalized output event preceded it, false otherwise. The vendor's error identity MUST survive as `RawLabel()` under the same opacity rule as `R-AEM-006`. No further events MUST follow the terminal error event.

An in-band error frame and a transport failure MUST remain distinguishable through the error chain by a stable identity match (`errors.Is`), not by inspecting message text, while both MAY report the same `FailureCategory`.

#### Scenarios

- **S-AEM-040** *[test]* — Given a transcript fixture of two content frames followed by an in-band error frame, when the stream is consumed to completion, then the last event's `ErrorPayload()` reports `(failure, true)` and `failure.PartialOutput()` is true.
- **S-AEM-041** *[test]* — Given the same fixture, when the events after the terminal error event are counted, then the count is zero.
- **S-AEM-042** *[test]* — Given an in-band error frame whose payload names vendor label `vendor_stream_fault`, when the terminal failure is read, then `RawLabel()` is `vendor_stream_fault`.
- **S-AEM-043** *[test]* — Given one stream terminated by an in-band error frame and one terminated by a transport read failure, when both terminal failures are matched with `errors.Is` against the in-band identity, then it holds for the first and not for the second, with no message-text inspection involved.

### R-AEM-011 — A new exported sentinel is admitted by name, never by weakening the guard

If `R-AEM-010`'s distinguishing identity is realised as an exported `Err`-prefixed package-level identifier, `TestPolicy_NoNewSentinelsExported` (S-ART-054) MUST be reconciled by **adding that entry to its enumerated allowlist in the scan's own order**, with a comment citing the requirement that sanctions it. The guard MUST NOT be re-frozen against the observed set, MUST NOT be weakened from exact set equality, and MUST NOT be skipped.

If the identity is realised without an `Err`-prefixed exported identifier, the allowlist MUST remain exactly `errors.go:ErrFrameTooLarge`, `errors.go:ErrTruncated`.

#### Scenarios

- **S-AEM-044** *[test]* — Given the shipped package after this milestone, when `TestPolicy_NoNewSentinelsExported` runs, then it passes against an enumerated allowlist that names every exported `Err`-prefixed identifier exactly once, in scan order.
- **S-AEM-045** *[inspection]* — Given the allowlist after this milestone, when a reviewer reads it, then either it is unchanged from the two AI-27 entries, or each added entry carries an adjacent comment naming the `R-AEM-` requirement that sanctions it; the guard's comparison remains exact set equality in both cases.

---

## AI-32.3 — Disconnects and deadlines `[leaf]` · stage 2

### R-AEM-012 — A disconnect after emitted output produces a terminal event with partial output preserved

When the transport disconnects after at least one normalized output event has been emitted, the stream MUST terminate with a terminal error event whose payload's `PartialOutput()` is true and whose `Delivery()` is `ai.DeliveryMidStream`. The already-emitted output events MUST remain delivered and unaltered. This fact MUST be assertable on the event; a failure delivered only as a returned value does not satisfy this requirement.

#### Scenarios

- **S-AEM-046** *[test]* — Given a transcript that emits two content events and then the transport closes abruptly, when the stream is consumed, then the two content events are received first and the final event's `ErrorPayload()` reports `(failure, true)` with `PartialOutput()` true.
- **S-AEM-047** *[test]* — Given the same stream, when the terminal failure's `Delivery()` is read, then it is `ai.DeliveryMidStream`.
- **S-AEM-048** *[test]* — Given the same stream, when the received content events are compared against the transcript's emitted content, then they are byte-identical — the failure discards no already-delivered output.

### R-AEM-013 — A disconnect before any output takes the pre-stream path

When the transport fails before any normalized output event has been emitted, the failure MUST take AI-20.2's pre-stream path: it MUST be returned as an `ai.PreStreamFailure` with `Delivery() == ai.DeliveryPreStream`, and `PartialOutput()` MUST be false. No terminal error event claiming partial output MUST be produced.

#### Scenarios

- **S-AEM-049** *[test]* — Given a transport that fails before the first frame is decoded, when the call is made, then a `*ai.Failure` is returned with `Delivery() == ai.DeliveryPreStream` and `PartialOutput()` false.
- **S-AEM-050** *[test]* — Given the same failure, when the event stream is drained, then it yields no event whose `ErrorPayload()` reports `PartialOutput()` true.

### R-AEM-014 — Deadline and cancellation are never conflated

A context deadline expiring mid-stream MUST map to `FailureCategoryTimeout`, retryable-flagged. An explicit caller cancellation MUST map to `FailureCategoryCancellation`, never retryable-flagged. The two MUST be distinguishable both through `errors.Is` against their AI-19 sentinels (`ai.ErrTimeout`, `ai.ErrCancelled`) and by reading the terminal event's payload category. Neither MUST be reported as the other, and neither MUST be reported as `FailureCategoryUnavailable`.

#### Scenarios

- **S-AEM-051** *[test]* — Given a stream whose context deadline expires between frames, when the terminal event's payload is read, then `Category()` is `ai.FailureCategoryTimeout` and `errors.Is(failure, ai.ErrTimeout)` holds.
- **S-AEM-052** *[test]* — Given a stream cancelled explicitly by the caller between frames, when the terminal event's payload is read, then `Category()` is `ai.FailureCategoryCancellation` and `errors.Is(failure, ai.ErrCancelled)` holds.
- **S-AEM-053** *[test]* — Given the cancelled stream's failure, when it is matched against `ai.ErrTimeout`, then `errors.Is` is false; and given the deadline stream's failure matched against `ai.ErrCancelled`, then `errors.Is` is false.
- **S-AEM-054** *[test]* — Given both streams, when each failure's `Retryable()` is read, then the deadline failure reports true and the cancellation failure reports false.
- **S-AEM-055** *[test]* — Given a deadline expiry after one content event was emitted, when the stream is consumed, then the terminal event's payload reports `Category()` timeout **and** `PartialOutput()` true — the two axes are independent.

---

## AI-32.5 — Bounded, sanitized capture `[leaf]` · stage 2

### R-AEM-015 — Body capture is size-limited, marked, and the remainder drained

Error-body capture MUST stop at the capture limit and MUST NOT retain more than that many body bytes regardless of the response's declared or actual length. A capture that hit the limit MUST carry the truncation marker; a capture that did not MUST NOT carry it. The unread remainder MUST be drained and the body closed exactly once, so the failure path never leaves the connection unusable.

The distinguishing fixture for truncation MUST place different, individually assertable content on each side of the cut point, so a mapper that silently captured everything and a mapper that captured nothing both fail.

#### Scenarios

- **S-AEM-056** *[test]* — Given a 500 response whose body is `HEAD-AEM056` followed by filler to exactly the capture limit and then `TAIL-AEM056`, when it is mapped, then the retained diagnostic contains `HEAD-AEM056`, does not contain `TAIL-AEM056`, and its length equals the capture limit plus the marker's length.
- **S-AEM-057** *[test]* — Given the same response, when the retained diagnostic's suffix is read, then it is exactly the truncation marker.
- **S-AEM-058** *[test]* — Given a 500 response whose body is 16 bytes long, when it is mapped, then the retained diagnostic is exactly those 16 bytes and contains no truncation marker.
- **S-AEM-059** *[test]* — Given a 500 response with a multi-megabyte body, when it is mapped, then the body reader reports it was read to completion and closed exactly once, and no more than the capture limit of body bytes was retained.

### R-AEM-016 — A credential echoed in a body never survives into typed error text

A sentinel credential appearing only inside an error body MUST NOT appear in `Failure.Error()`, in the rendered text of the cause the mapper constructs, in `fmt.Sprintf("%v", …)` or `%+v` of either, or in any exported accessor's value. `Failure.Error()`'s structural cleanliness (R-AIP-009) is inherited but is not sufficient: the guarantee lands on the **cause constructed at capture time**, because `Unwrap()` deliberately keeps the cause inspectable (S-AIP-029).

The distinguishing fixture MUST use a sentinel that could reach any output **only** by way of the error body — never a value also present in configuration, headers or the request — so a mapper that never captured the body cannot pass vacuously.

#### Scenarios

- **S-AEM-060** *[test]* — Given a 400 response whose body message embeds the sentinel `sk-AEM060-planted-in-body-only`, present nowhere in the client configuration, headers or request, when the failure is mapped, then `Failure.Error()` does not contain the sentinel.
- **S-AEM-061** *[test]* — Given the same failure, when `errors.Unwrap` is followed to every reachable cause and each cause's `Error()` is read, then no reachable text contains the sentinel.
- **S-AEM-062** *[test]* — Given the same failure, when it is rendered with `%v` and `%+v`, then neither rendering contains the sentinel.
- **S-AEM-063** *[test]* — Given the same fixture with the sentinel removed but the surrounding body text kept, when the mapped failure's retained diagnostic is read, then it contains the surrounding body text — proving the body was genuinely captured and `S-AEM-060` is not passing by never reading it.

---

## Charter boundary and citation integrity · stage 1

### R-AEM-017 — The charter boundary holds

This milestone MUST NOT export a backoff policy, an attempt counter, a failover identifier, or any scheduling surface: retryability is a flag and AI-35 owns acting on it. It MUST NOT widen `ai.FailureCategory` — the vocabulary stays at nine members. It MUST NOT amend any promoted spec. `backend/agent/go.mod` MUST continue to declare zero `require` lines.

#### Scenarios

- **S-AEM-064** *[test]* — Given the package's non-test sources parsed with `go/ast`, when exported top-level identifiers are scanned, then none names a backoff, attempt-count, or failover concept.
- **S-AEM-065** *[test]* — Given `ai.FailureCategories()`, when its length is read after this milestone, then it is exactly 9.
- **S-AEM-066** *[test]* — Given `backend/agent/go.mod`, when its lines beginning `require` are counted, then the count is 0.
> **Revision note (design-validation corrective, 2026-08-04).** `S-AEM-067` previously targeted "`doc.go`'s ruling that AI-27 constructs no `ai.Failure`". No such ruling exists in `doc.go`; the two passages that make that statement live in `errors.go` — the `ErrFrameTooLarge` doc comment ("AI-27 never constructs an `ai.Failure` … from this sentinel (S-ASD-064)") and the `Category` doc comment ("`Category` never constructs an `ai.Failure` value of any kind … AI-28 and AI-32 own construction"). The scenario is retargeted to those real passages, so the design's File Changes row amends `errors.go`. Scenario count and kind are unchanged.

- **S-AEM-067** *[inspection]* — Given `errors.go`'s two passages stating that AI-27 never constructs an `ai.Failure` from its sentinels and that `Category` constructs no `ai.Failure` of any kind, when a reviewer reads them after this milestone, then both are amended to record that AI-32 now owns **wire-side** construction while AI-28 owns the producer path; that AI-26.6's translation-time capability refusal in `policy.go` is a distinct, pre-existing construction site left untouched by this milestone; and that no promoted spec was edited to say so.

### R-AEM-018 — Every wire claim is labelled and every dialect claim is pinned

Every statement in this spec, and in the implementation's own documentation, about the vendor's wire behavior MUST be labelled either **cited** — resolving to `citations.md` evidence `E1` … `E6` at the pinned commit — or **dialect-conventional**, in which case it MUST name its fixture-pin obligation. An unlabelled wire claim is a defect. A **cited** label on a claim the pinned specification does not state is a defect. A reference that does not resolve is a defect.

#### Scenarios

- **S-AEM-068** *[inspection]* — Given every wire claim in this spec, when a reviewer reads its label, then each is labelled cited or dialect-conventional and none is unlabelled.
- **S-AEM-069** *[inspection]* — Given every `E1` … `E6` reference in this spec, when a reviewer resolves it against `citations.md`, then each resolves to an evidence item that states the claim.
- **S-AEM-070** *[inspection]* — Given every dialect-conventional claim, when a reviewer reads it, then it names a pinning fixture obligation and at least one `[test]` scenario replays that fixture; no dialect claim is attributed to the pinned specification.

---

## Credential disclosure through the bounded excerpt · added by AI-36

> **Amended 2026-08-08 (AI-36)** by [`cachicamas-ai-redaction`](../../changes/archive/2026-08-08-cachicamas-ai-redaction/) (AI-36 — Enforce secret redaction, Wave 5 — Harden). One requirement `R-AEM-019` added. No requirement is modified, removed or renamed — `R-AEM-001` … `R-AEM-018` stand unchanged, and in particular `R-AEM-015`'s capture limit and truncation marker are untouched: this requirement concerns **disclosure**, never size.
>
> **This requirement was contingent, and its trigger FIRED empirically. It is LANDED and binding, not conditional.** `R-AEM-019` was specified to become binding if and only if the hostile-server case required by `S-CNF-081` showed that the caller's **own credential** is reproduced inside the bounded response excerpt a non-streaming content-type refusal interpolates into its rendered text. It was. A provider driven to echo the caller's authorization value and request body back inside a non-streaming response reproduced that credential **verbatim, byte for byte**, in the refusal's rendered text and in the cause reachable one unwrap beneath it, under all four rendering verbs — observed as the failing phase of the test, before any production change existed. The removal step was then landed and the same case re-run green; it now stands as the regression pin for both scenarios below.

### R-AEM-019 (added 2026-08-08) — A bounded response excerpt never reproduces the caller's own credential

A refusal that reproduces a bounded excerpt of the provider's response into its rendered text MUST NOT reproduce the caller's own credential within that excerpt. Specifically:

1. Any occurrence of the caller's credential inside a captured response excerpt MUST be removed before that excerpt becomes reachable through a rendering.
2. The excerpt MUST remain present and readable to the caller reading the terminal failure's rendered text. Removal MUST be of the credential occurrences only — suppressing the excerpt would defeat its diagnostic purpose and would violate the landed obligation that the caller can see it.
3. The excerpt's existing size bound MUST be unchanged. This requirement concerns disclosure, never size.
4. A provider echoing the caller's own **content** back in its response is a **named residual**, not a defect, and MUST be recorded in writing rather than suppressed.
5. The absence claim MUST ship a positive control.

#### Scenarios

- **S-AEM-071** *[test]* — Given a provider that echoes the caller's authorization value into a non-streaming response carrying an unexpected content type, when the resulting refusal is rendered, then the bounded excerpt is present and readable but the credential appears nowhere in the rendered text; and given the same case with the removal step disabled, when the check runs, then it fails — proving the claim is falsifiable.
- **S-AEM-072** *[test]* — Given that same refusal, when its excerpt is measured against the previously landed size bound, then the bound is unchanged and the excerpt remains reachable to a caller reading the terminal failure's rendered text; and given a provider echoing the caller's own content rather than the credential, when the milestone closes, then that outcome is recorded in writing as a named residual.

> **Item 4's and `S-AEM-072`'s required recording, made at AI-36's close (2026-08-08).** The caller's own prompt content **was** observed echoed into the excerpt on the same run that exposed the credential, and it is **deliberately left unsuppressed**. Suppressing a provider's replay of the caller's own content would defeat item 2 — the excerpt exists to be read, and an excerpt scrubbed of everything the caller sent is not diagnostic. It is therefore recorded here, and in [`ai-provider-conformance-suite`](../ai-provider-conformance-suite/spec.md)'s `S-CNF-081`, as an accepted, maintainer-visible **named residual**. The distinction that makes this safe is exact: the **credential's** absence is proven, not assumed — that guard was empirically defeated and restored — while the **content's** presence is intended behaviour.
>
> **Recorded limitations of the removal step (informational, for AI-37).** Two are known and deliberately not closed here. First, removal is non-overlapping, so an interior credential prefix that is not itself a whole occurrence can survive a match at another position. Second, the credential can be bisected by the capture layer's fixed truncation offset, leaving an attacker-chosen prefix that whole-occurrence removal cannot match; this was found and narrowed during review, and the truncation-edge handling is deliberately **not** applied to the content-type header, because doing so could erase the very media type item 2 requires the refusal to name.

## doc 0002 test-list coverage map

Every item of every AI-32 node's test list maps to at least one scenario.

| doc 0002 item | Requirements | Scenarios |
| --- | --- | --- |
| AI-32.1 item 1 — each failing status class maps to its AI-19.2 category, table-tested | `R-AEM-001`, `R-AEM-002` | `S-AEM-001` … `S-AEM-010` |
| AI-32.1 item 2 — retryability follows the taxonomy; the flag, not the retry | `R-AEM-003`, `R-AEM-017` | `S-AEM-011` … `S-AEM-013`, `S-AEM-064` |
| AI-32.1 item 3 — unparseable body still maps; body kept as bounded diagnostic | `R-AEM-004`, `R-AEM-015` | `S-AEM-014` … `S-AEM-016`, `S-AEM-056` … `S-AEM-059` |
| AI-32.1 item 4 — undocumented status maps through the fallback without crashing | `R-AEM-005` | `S-AEM-017` … `S-AEM-020` |
| AI-32.2 item 1 — in-band frame → typed terminal event, vendor identity, partial-output discriminator, distinguishable from transport failure | `R-AEM-010`, `R-AEM-011` | `S-AEM-040` … `S-AEM-045` |
| AI-32.3 item 1 — disconnect after output → terminal event, partial output preserved | `R-AEM-012` | `S-AEM-046` … `S-AEM-048` |
| AI-32.3 item 2 — disconnect before output → AI-20.2 pre-stream path | `R-AEM-013` | `S-AEM-049`, `S-AEM-050` |
| AI-32.3 item 3 — deadline → timeout, cancel → cancellation, separable by `errors.Is` and in the event | `R-AEM-014` | `S-AEM-051` … `S-AEM-055` |
| AI-32.4 item 1 — retry-after typed, both delay-seconds and HTTP-date forms | `R-AEM-008` | `S-AEM-030` … `S-AEM-035` |
| AI-32.4 item 2 — rate-limit telemetry captured as safe machine-readable metadata | `R-AEM-009` | `S-AEM-036` … `S-AEM-039` |
| AI-32.5 item 1 — capture size-limited, remainder drained, limit and marker asserted | `R-AEM-015` | `S-AEM-056` … `S-AEM-059` |
| AI-32.5 item 2 — sentinel credential in a body does not survive into typed error text | `R-AEM-016` | `S-AEM-060` … `S-AEM-063` |
| Supporting: vendor body shape, label opacity, request id | `R-AEM-006`, `R-AEM-007` | `S-AEM-021` … `S-AEM-029` |
| Supporting: citation integrity, charter boundary | `R-AEM-017`, `R-AEM-018` | `S-AEM-064` … `S-AEM-070` |

## The spec holds when

1. `R-AEM-001` … `R-AEM-018` hold, each verified by its scenarios; and `R-AEM-019`, appended 2026-08-08 by AI-36, holds likewise.
2. Every row of `R-AEM-002` is replayed by a pinning fixture and labelled dialect-conventional.
3. Every `E1` … `E6` reference resolves at the pinned commit.
4. No promoted spec is amended and `ai.FailureCategory` still has nine members.
5. Stage 2's requirements are satisfied against AI-28.1's landed producer surface, never against an imagined one.
