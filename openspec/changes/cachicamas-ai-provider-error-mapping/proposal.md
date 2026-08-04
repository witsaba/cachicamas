# Proposal: Map HTTP and provider failures (AI-32)

## Intent

> **Revision note (design-validation corrective, 2026-08-04).** The original first sentence — "`openaicompat` **names** failure categories and constructs none" — was factually false. `policy.go`'s `refuse` already constructs `ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnsupportedCapability, …})` for AI-26.6's capability refusal. The correct statement is that the package constructs exactly one failure, at **translation** time, and has never constructed one **from the wire**; AI-32 is the first wire-side constructor. Scope, capabilities and success criteria are unchanged. The Affected Areas table is corrected with it: the "amend the no-construction ruling" edit belongs to `errors.go`, not `doc.go` — `doc.go` contains no such ruling, and its `doc.go` row is dropped rather than left pointing at text that does not exist.

`openaicompat` constructs exactly one `ai.Failure` today, and it is decided at **translation** time from the request alone: AI-26.6's capability refusal (`policy.go`), an `ai.PreStreamFailure` carrying `ai.FailureCategoryUnsupportedCapability`. From the **wire** it constructs nothing. `Category()` (errors.go) only returns a bare `ai.FailureCategory` for two decoder sentinels; nothing turns an HTTP status, a vendor error body, an in-band error frame or a disconnect into `ai.PreStreamFailure` / `ai.MidStreamFailure`. AI-19's taxonomy is unreachable from the wire, and AI-32 is the first wire-side constructor — leaving AI-26.6's translation-time refusal distinct and untouched.

Load-bearing acceptance (doc 0002, AI-32 charter): a mid-stream disconnect must produce a terminal error **event** carrying the partial-output discriminator, not merely a returned error. An error that arrives only as a return value has thrown away the fact Layer 2's retry decision depends on.

## Scope

### In Scope

- **AI-32.1** status → category table (authentication, authorization, not-found/invalid, rate limit, unavailable, timeout); retryability flags; unparseable-body and undocumented-status fallbacks.
- **AI-32.4** typed retry-after from both delay-seconds and HTTP-date forms; rate-limit telemetry captured as safe machine-readable metadata.
- **AI-32.2** in-band error frame → terminal `ai.ErrorEvent`, distinguishable from a transport failure.
- **AI-32.3** disconnect after output (terminal event, partial preserved) vs before output (pre-stream path); deadline → `Timeout`, cancel → `Cancellation`, separable via `errors.Is`.
- **AI-32.5** size-limited body capture with remainder drained and truncation marker; a sentinel credential echoed in a body never survives into typed error text.

### Out of Scope

- Acting on retryability — AI-35 owns the decision; this is the flag only. No backoff, attempt counter or failover.
- Widening `ai.FailureCategory` (closed, append-only). The recorded tenth resource-exhaustion member stays recorded, not acted on.
- Amending `ai-provider-errors` — cited and satisfied, never edited.
- Adversarial redaction sweep (AI-36), usage mapping (AI-31), the producer itself (AI-28), any `go.mod` require.

## Capabilities

### New Capabilities

- `ai-provider-error-mapping`: wire failure (status, body, frame, disconnect, deadline) → AI-19 taxonomy, for the OpenAI-compatible adapter.

### Modified Capabilities

- None. `ai-provider-errors`, `ai-stream-lifecycle`, `ai-response-events`, `ai-provider-conformance-suite`, and the in-flight `ai-stream-decoder` / `ai-provider-text-stream` are cited by identifier and satisfied, never amended.

## Approach

Two mapping seams, one per delivery path (AI-19's `DeliveryPath`, never conflated):

1. **Pre-stream** — a response check reads status + a bounded, drained, sanitized body and returns `ai.PreStreamFailure`. This is AI-28.6's blocking dependency.
2. **Mid-stream** — a mapper returns `ai.MidStreamFailure(report, outputPreceded)`, wrapped by `ai.ErrorEvent`, from the producer's single closing site.

**Sanitize at capture, not at render.** `Failure.Error()` already excludes body, label, status and request id (R-AIP-009), but `Unwrap()` deliberately keeps the cause inspectable (S-AIP-029), so AI-32.5's credential guarantee must hold on the cause this milestone constructs — not be inherited from AI-19's rendering.

## Delivery — two stages

| Stage | Nodes | Gate |
|---|---|---|
| 1 | AI-32.1, AI-32.4 | None. AI-19 + AI-27 landed. Stage 1 unblocks AI-28.6, which AI-28 sequences last. |
| 2 | AI-32.2, AI-32.3, AI-32.5 | AI-28.1 landed and the chains merged. The producer surface is read from AI-28's landed code, never designed against an imagined API. |

The bounded-capture helper AI-32.1 item 3 needs ("body kept as a bounded diagnostic") lands in stage 1; AI-32.5 is its proof node in stage 2.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/ai/openaicompat/errors.go` | Modified | Status → category mapping; any new failure identity; **the two doc comments** (`ErrFrameTooLarge`, `Category`) stating AI-27 constructs no `ai.Failure` amended to record AI-32's wire-side ownership |
| `backend/agent/src/ai/openaicompat/` (new files) | New | Bounded sanitized capture, pre-stream response check, retry metadata |
| `backend/agent/src/ai/openaicompat/reasoning_refusal_test.go` | Modified | S-ART-054 allowlist gains a cited entry **only if** a sentinel is exported |
| `backend/agent/src/ai/` (package `ai`) | Unchanged | Taxonomy consumed, never widened |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| **No cited vendor status table or error-body shape exists.** AI-24 § 12 hands AI-32 "raw HTTP status codes and this vendor's own structured error-body shape"; § 10's citations are framing-only and § 8's `CAP-R-05` asserts the body unlabelled | High | Spec phase must cite every status/body claim against pinned commit `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439` **or** label it dialect-conventional with a fixture-pin obligation, in § 10.2's exact form. No wire fact asserted from memory |
| **Rate-limit telemetry has no home.** `ai.FailureReport` carries only Category, Retryable, RetryAfter, RawLabel, StatusClass, RequestID, Cause — nowhere for AI-32.4 item 2's telemetry headers | Med-High | Decide in spec/design: adapter-local carrier vs. mapping only what fits. Widening AI-19 is out of scope |
| A new exported sentinel (e.g. distinguishing an in-band error frame from a transport failure, AI-32.2) bites `TestPolicy_NoNewSentinelsExported` | Med | Enumerated allowlist gains a cited entry — never a re-freeze, never a weakened guard |
| Stage-2 producer API does not exist yet | High | Stage 2 is gated on AI-28.1 landing; stage 1 ships independently |
| A vendor failure with no clean AI-19 member (resource exhaustion) | Med | Record the compromise in `ErrFrameTooLarge`'s documented form; never widen the vocabulary here |
| `sanitizeOpaqueField`'s 64-byte drop-whole bound silently drops a real vendor label or request id | Low-Med | Assert the drop is observable and total, never a truncation |

## Rollback Plan

Each node is its own chain slice. Revert the slice commit: no promoted spec is amended, no `package ai` surface changes, and no `go.mod` line moves, so a revert is local to `openaicompat`. The only cross-file edits are the `errors.go` doc-comment amendment and the S-ART-054 allowlist entry, both of which revert with their slice. Stage 1 is revertible without touching AI-28's chain; stage 2 reverts to a stage-1-only adapter that still unblocks AI-28.6.

## Dependencies

- AI-19 (`provider_failure.go`) — landed.
- AI-27 (decoder + `Category()`) — landed.
- AI-28.1 (producer) — **stage 2 only**, in flight on `feat/ai-28-0-integration-base`.
- AI-24 decision §§ 9, 12 — inherited, with the citation gap above unresolved.

## Success Criteria

- [ ] Every failing status class maps to its AI-19.2 category, table-tested, with a documented fallback for undocumented statuses.
- [ ] Retryability is a flag on every constructed failure; no backoff, attempt counter or failover identifier is exported.
- [ ] A mid-stream disconnect after emitted output produces a terminal **event** whose `PartialOutput()` is true — asserted on the event, not on a returned error.
- [ ] Deadline expiry and explicit cancellation stay distinguishable through `errors.Is` and in the terminal event.
- [ ] A multi-megabyte error body is bounded and drained, with both the limit and the truncation marker asserted.
- [ ] A planted credential sentinel in an error body appears in no typed error text.
- [ ] Every vendor wire claim in the spec carries a citation or a labelled fixture-pin obligation.
- [ ] `grep -c '^require' backend/agent/go.mod` still reports 0.
