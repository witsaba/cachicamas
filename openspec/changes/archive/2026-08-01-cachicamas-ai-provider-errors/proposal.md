# Proposal — the provider error taxonomy and the terminal error event

> **Change**: `cachicamas-ai-provider-errors`
> **Milestone**: AI-19 — Define the provider error taxonomy and the terminal error event
> **Nodes**: AI-19.1 `[leaf]` · AI-19.2 `[leaf]` · AI-19.3 `[leaf]` · AI-19.4 `[leaf]` · AI-19.5 `[leaf]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Date**: 2026-08-01
> **Driver**: braejan
> **Wave**: 2 — Stream · this milestone is the wave's **keystone**, deliberately placed **before** AI-20
> **Scope**: `openspec/changes/cachicamas-ai-provider-errors/`, one new production file plus its test file(s) under `backend/agent/src/ai/`, and a new capability spec. No `go.mod` change, no new dependency
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-04 (merged) · AI-14, AI-15 (**wave 2, must land first — see Dependencies**)
> **Blocks**: AI-20, AI-21, AI-23, AI-27, AI-32
> **Closes**: **C4** by construction · the Layer 1 half of **G8**

---

## Intent

Give Layer 1 **one** inspectable vocabulary for everything that goes wrong after a valid request leaves the process, and make it reachable by **both** delivery paths — returned directly when a request never becomes a stream, and carried as the stream's terminal error event when a stream dies mid-flight.

The ordering is the point. The retired plan defined the provider interface first and this taxonomy second, and so declared a mandatory terminal error event whose payload **no adapter could construct** — defect **C4**. Defining the taxonomy first makes C4 unreachable rather than fixed: the payload type ships before the interface that requires it, in the same package as the sealed envelope interface it must satisfy.

The second recorded defect this closes is **G8**: a consumer that cannot tell *"did any output precede this failure?"* from the failure itself falls back to the naive predicate "retry if nothing completed", which is wrong for the single most common real-world failure — a stream that dies **after** emitting output.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can tell the inherited from the decided.

1. **The vocabulary is AI-01's.** `V-FAIL-05` … `V-FAIL-13` are used with their register definitions and cited by identifier, never paraphrased or redefined. This change *implements* them.
2. **The delivery boundary is AI-02.1's, already decided**: the boundary is **carrier handover**, not the first event. A stream handed over that fails before emitting content is **mid-stream** (`V-FAIL-12`), not pre-stream (`V-FAIL-11`). AI-19 implements this split; it does not reopen it.
3. **Cancellation is a required category member**, not optional — AI-02.1's producer wording ("a terminal error event carrying the category AI-19 assigns to cancellation, without waiting") depends on it existing, and the charter's minimum list names it independently.
4. **Terminal-event container invariants are AI-14's** (`V-STR-18`: exactly one per stream, nothing follows it, its two instances are completion and error). AI-19 owns only the **error instance's payload** (`V-FAIL-10`).
5. **This taxonomy is the complement of AI-04's, not an extension of it.** `R-AIE-010` reserves categories, retryability, partial output and terminal events for AI-19. AI-19 must not add categories to AI-04's `ruleClasses` registry and must not reuse `Violation` for provider failures.
6. **Redaction posture** — `V-FAIL-13`: status classes, categories, retry hints, bounded sanitized excerpts. Never a credential, header, raw provider body, or model content. Adversarial sweeps are AI-36's; the **type-level default** is AI-19's.
7. **`V-FAIL-15`'s one binding clause**: the partial-output case is never retried at Layer 1 — which is only enforceable if AI-19's discriminator stays **unpolluted by delivery path**.
8. **The module carries zero requires.** `time` and `errors` are standard library. Both AI-00 import guards must still pass.
9. **Strict TDD.** `openspec/config.yaml` sets `apply.tdd: true`; every test-list item is taken red → green → refactored, recorded in `tasks.md`. Evidence gate is recorded green `make test` (= `go test -race -v ./...`) in `backend/agent/`.

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `explore.md` | Shipped-idiom survey, the binding register rows, three approaches, four open questions *(persisted)* |
| `proposal.md` | This file |
| `specs/ai-provider-errors/spec.md` | `R-AIP-0NN` requirements over runtime behavior, with Given/When/Then scenarios `S-AIP-0NN` |
| `design.md` | The Go spellings — type, constructor, accessor and sentinel names — and the reasoning behind each |
| `tasks.md` | AI-19.1 … AI-19.5 as phases, one task per test-list item, red→green record, review workload forecast |
| `backend/agent/src/ai/<provider-failure>.go` | **The deliverable.** Category vocabulary, the one failure type, retry hints, safe metadata, partial-output discriminator, terminal-error payload. Filename settled in `design.md` |
| `backend/agent/src/ai/<provider-failure>_test.go` (+ internal test file if the registry needs one) | The five nodes' test-list items, in external test package `ai_test` |
| `backend/agent/src/agenttest/` (read/extend only if AI-19.1.1 requires it) | AI-19.1's "an adapter in **another package** constructs the terminal error event" proof — the property whose absence was C4 |

### The four decisions this proposal commits to

The exploration left four questions open. All four are resolved here, one paragraph each, so a reviewer can accept or reject the substance before `design.md` fixes spellings.

**1. The raw provider label is a bounded field on the failure, not a string table.**
`finish_reason.go`'s total normalizer works because vendor finish-reason spellings are a small, bounded, knowable set. Provider and transport failures have no such set — HTTP status text, vendor error codes and gateway messages are unbounded and adapter-specific — so AI-19 ships **no cross-vendor normalizer**. Category assignment is the **adapter's** job (Wave 4, AI-32). AI-19's obligation under AI-19.2.2 is narrower and fully satisfiable now: the failure type **has a place to carry an opaque raw provider label**, bounded and sanitized per `V-FAIL-13`, preserved whenever the category is unknown. AI-19 proves the field exists, is bounded, survives wrapping and never widens the redaction surface; it does not populate it from any real vendor.

**2. `Error()` never renders the wrapped cause's text.**
Following `Violation.Error()`'s precedent exactly (`R-AIE-006`): the rendered string is built **only** from the category's own registered text, bounded safe metadata and fixed punctuation. `Unwrap()` exposes the cause for any caller who wants it, so "wrapped causes remain inspectable" holds — but a cause carrying a response body or a credential cannot reach a log line through the default string. This makes redaction a **property of the type** rather than of everybody's discipline, which is the only version of the claim that survives a refactor.

**3. The partial-output discriminator is a single bool; delivery stays a separate, orthogonal fact.**
The register draws delivery (`V-FAIL-11`/`V-FAIL-12`) and partial output (`V-FAIL-08`/`V-FAIL-09`) as **perpendicular axes**, and `V-FAIL-15`'s never-retry clause turns on partial output *alone*. A 3-member enum (`pre-stream` / `mid-stream-no-output` / `mid-stream-after-output`) flattens that product into one field and re-introduces exactly the conflation G8 records as the historical defect: it makes the naive-retry predicate read two facts out of one field. So: the discriminator is **one unpolluted bool** — *did normalized output events precede this failure?* — and the delivery path is its **own** accessor, set by two distinct constructors (pre-stream, mid-stream). AI-19.4.1's three distinguishable shapes are then the two axes read together (`pre-stream` × `false`, `mid-stream` × `false`, `mid-stream` × `true`), still answerable from the failure value alone; AI-19.4.2's "is a naive retry safe?" is answerable from the bool alone, which is the whole point. The impossible fourth cell (`pre-stream` × `true`) is unconstructible because the pre-stream constructor takes no output flag.

**4. Retryability is `Retryable() bool` plus an independent, typed retry-after — locked.**
The classification lives here, where the wire evidence is; the **decision** to retry lives one layer up (v2 § 6 seam 7; `V-OUT-11`), and Layer 2 is out of scope for AI-19. Layer 1 therefore needs the wire fact, not a backoff shape: one machine-readable retryability signal per failure, plus a separately-read retry-after **duration carried typed** when the provider supplied one — using the presence-vs-zero two-result accessor idiom `usage.go`'s `TokenCount` already establishes, so "absent" and "zero seconds" stay distinguishable and no caller ever re-parses message text. A richer backoff descriptor is `AI-35`'s and would be speculative here.

### Out of scope (explicit, and deferred-but-related is called out)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| The provider interface itself | **AI-20** | The whole ordering decision of this wave. AI-20 may only declare a mandatory terminal error that already exists |
| Any concrete vendor adapter, and any real category assignment from a real wire failure | **AI-32** (Wave 4) | AI-19 defines the vocabulary and the carrier; mapping a vendor's error onto it is adapter work |
| Backoff execution, retry scheduling, model failover | **AI-35**, Layer 2 | Retryability is classified here and *acted on* one layer up. Collapsing the two loses both |
| The repo-wide adversarial redaction sweep | **AI-36** | `V-FAIL-14` is AI-36's. AI-19 owns the type-level default, not a proven-adversarial guarantee |
| Cross-package stream conformance / exhaustiveness enforcement | **AI-23** | AI-19 must only make category membership **closed and enumerable** so AI-23.4 can iterate it |
| Gap-detection and recorded-stream helpers | **AI-22.3** | Consumer-side tooling over the invariants, not the taxonomy |
| Category-specific metadata (rate-limit reset, quota identity) | **AI-19.6, deferred** | doc 0002's split trigger: appended as its own node **if** it grows the category list past seven items. Watched during `sdd-spec`/`sdd-design` |
| Terminal-event container invariants (one per stream, nothing follows) | **AI-14.4** | AI-19 supplies the error payload; AI-14 owns the container it rides in |
| Structured logging, error codes, wire rendering | doc 0003 / doc 0004 | Layer 1 returns values; it does not render them |

## Capabilities

### New Capabilities

- `ai-provider-errors`: the provider/transport failure taxonomy — the closed failure-category vocabulary, the single failure type carried by both delivery paths, retryability and typed retry-after, bounded safe metadata including the raw provider label, the partial-output discriminator, and the constructible terminal error event payload.

### Modified Capabilities

- None expected. `ai-validation-errors` (`R-AIE-010`) already reserves this surface for AI-19 rather than describing it, so no requirement of it changes. `ai-stream-lifecycle` and AI-14's envelope capability are **consumed**, not amended. `ai-contract-vocabulary` needs **no** append: `V-FAIL-05` … `V-FAIL-13` already exist and already close the ownership question — if `sdd-spec` or `sdd-design` meets a Layer 1 noun no register row describes, it appends a row in this same PR under the register's own § 9 rule 2, and this section is amended to say so.

## Approach

1. **Mirror AI-04's shape as a sibling, not a shared vocabulary** (exploration Approach 1). One closed `uint8` category enum built on the shipped idiom — `iota` from a blank `_` so the zero value is not a member, a trailing bound constant, a `[limit]string` array rather than a map (nothing in this package lets unordered iteration decide anything), and a `Validate` reporting through AI-04's `ErrNotInVocabulary`. Category membership is closed and enumerable, which is what AI-23.4 later iterates.
2. **One concrete failure type, carried by both delivery paths.** It implements `error` and `Unwrap() error`, and exposes category, retryability, typed retry-after, the partial-output bool, the delivery path, and the bounded raw label. It is **returned directly** pre-stream and **wrapped as the terminal event's payload** mid-stream. AI-19.5's "one vocabulary, two delivery paths" then becomes a *structural fact* — the same type, physically unable to drift — rather than two implementations somebody must keep in sync.
3. **Per-category sentinels for `errors.Is`.** A caller writes `errors.Is(err, ai.Err<Category>)` without first unwrapping to the concrete type; `errors.As` reaches the full failure for retry hints and metadata. Both must survive at least one layer of wrapping (AI-19.5.1), exactly as `Violation` does today.
4. **Close C4 by construction, then prove it from outside.** The payload type lives in package `ai`, compiled together with the sealed envelope-payload interface it satisfies, so "a registered kind with no constructible payload" is not expressible. AI-19.1.1 then proves the public half: an adapter in **another package** constructs the terminal error event through the exported surface and succeeds.
5. **Keep the two axes perpendicular in the type, not only in the prose.** Decision 3's separation is what makes `V-FAIL-15`'s never-retry-after-partial-output clause mechanically checkable instead of a comment.
6. **Ship only what the charter promises.** No normalizer, no vendor table, no backoff policy, no category-specific metadata. Every one of those has a named later owner in the out-of-scope table.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-provider-errors/` | New change directory, five markdown artifacts | None — new directory |
| `openspec/specs/ai-provider-errors/spec.md` | New capability spec | None — new file |
| `backend/agent/src/ai/<provider-failure>.go` | New file — the wave-2 keystone surface | **High** — AI-20, AI-21, AI-23, AI-27 and AI-32 all consume it. Mitigated by this proposal's four locked decisions and by `spec.md` |
| `backend/agent/src/ai/<provider-failure>_test.go` | New file(s) | None |
| `backend/agent/src/agenttest/` | Possibly extended for AI-19.1.1's external-package construction proof | Low — additive test surface |
| `backend/agent/src/ai/validation.go`, `finish_reason.go`, `usage.go`, `content_part.go` | **None** — read-only idiom references | — |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **None expected** (see Capabilities) | Low — any append is append-only, in this same PR |
| `backend/agent/go.mod`, `go.work` | **None** — zero requires preserved | — |
| doc 0002 | **None expected.** Any amendment required by the revert-and-record clause lands in this same PR and is reported prominently | — |

## Rollback plan

The change is additive: one new package file, its test file(s), one new change directory, one new capability spec. Nothing existing is modified. Rollback is `git revert` of the commit range; no build depends on the markdown, and `src/ai` returns to its pre-AI-19 exported surface. Because AI-20 has not landed, **nothing in the package depends on this type at revert time** — this is precisely why the milestone is scheduled before the provider interface rather than after it.

Partial rollback has a shape worth stating. Reverting the production file while keeping the spec would leave `openspec/specs/ai-provider-errors/spec.md` describing a surface that does not exist; the correct move is to revert both together, or to reject the change and re-propose. Reverting the taxonomy **after** AI-20 lands is a different and much more expensive operation — it would invalidate the provider interface's mandatory terminal error and reproduce C4 exactly — which is the standing argument for getting the four decisions above right in review rather than in a later wave.

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| **AI-14 / AI-15 are not on disk in this worktree.** AI-19's payload must satisfy AI-14's envelope-payload interface | **High** (current state) | **High** — apply cannot start | Proposal, spec and design proceed now (they reason from charters + the shipped sealing idiom). `sdd-apply` is **blocked** until AI-14 and AI-15 land. Re-check the sealed-interface assumption at `sdd-design` sign-off |
| AI-14 ships an **exported / non-sealed** payload interface, invalidating the "payload must live in-package" argument | Low | Medium | The C4-by-construction claim is re-verified against AI-14's landed code before apply; the in-package location remains correct either way |
| The category list is wrong — too coarse or too fine | High | Low | Wrong is the expected case. The vocabulary is closed but **append-only**, following AI-04's `R-AIE-003` discipline: a missing category is appended in the PR that meets it. *Unappendable* would be the failure |
| Category-specific metadata pushes the list past seven items mid-flight | Medium | Medium | doc 0002's own split trigger: it becomes **AI-19.6**, appended, rather than growing this change. Checked in `sdd-spec` and again in `sdd-tasks` |
| Decision 3 is read by a reviewer as failing AI-19.4.1's "three distinguishable shapes" | Medium | Medium | `sdd-spec` MUST write the scenario so it names both discriminating inputs explicitly and asserts the three shapes are distinguishable **from the failure value alone**. If spec phase concludes the bool alone must carry all three, escalate rather than silently switching to the enum |
| The delivery/partial-output axes get re-conflated by a later convenience accessor | Medium | High — it is G8 verbatim | The spec states the perpendicularity as a requirement, not a note, so a convenience accessor that collapses them fails a scenario |
| Redaction rests on discipline rather than construction | Medium | High — the leak surface is every log line above Layer 2 | Decision 2 makes it structural, proven with a distinctive sentinel body planted in the wrapped cause and asserted absent from the rendered string |
| The keystone grows past the review budget | Medium | Medium | Session delivery strategy is `exception-ok` with a ~5000-line budget accepted up front. `tasks.md` still carries the forecast and the split triggers |

## Dependencies

- **AI-04** — merged. Provides `ErrNotInVocabulary`, the sentinel/`errors.Is` idiom this change mirrors, and `R-AIE-010`, which reserves this surface.
- **AI-14** (`cachicamas-ai-event-envelope`) — **required before apply.** Owns the envelope, the sealed payload interface AI-19's payload must satisfy, and the terminal-event invariants of `V-STR-18`.
- **AI-15** (`cachicamas-ai-response-events`) — **required before apply.** AI-19.1.3's terminal exclusivity ("a stream ends in completion **or** error, never both") is asserted against AI-15's completion event.
- **AI-01 register** and **AI-02.1 stream-lifecycle decision** — merged; both are cited, neither is reopened.
- **No new Go dependency, and no ADR required.** `errors` and `time` are standard library; the module stays dependency-free.

## Success criteria

1. An adapter in **another package** constructs the terminal error event through the public surface, and it succeeds — the property whose absence was **C4**.
2. The constructed event satisfies AI-14's envelope invariants: kind derived from payload, nil or mismatched payload rejected, the event is terminal.
3. Terminal exclusivity holds: a stream ends in completion **or** error, never both, and no accessor can confuse the two payloads.
4. The category vocabulary distinguishes at minimum authentication, authorization, rate limit, unavailable/overloaded, timeout, cancellation, malformed response, unsupported capability, and unknown — each constructible, each distinguishable.
5. An unmodelled provider failure maps to **unknown with its raw provider label preserved**, bounded and sanitized.
6. Category membership is **closed and enumerable** — AI-23.4 can iterate it rather than listing cases by hand.
7. Every failure carries a machine-readable retryability signal, and a provider-supplied retry-after is carried **typed**, with "absent" distinguishable from "zero".
8. Machine-readable fields are separate from human text, and the human text is useful **without** containing a credential or a response body — proven with a planted sentinel body in the wrapped cause.
9. The three partial-output shapes (pre-stream; mid-stream, zero output; mid-stream, after output) are distinguishable from the failure value alone, and "is a naive retry safe?" is answerable from the discriminator alone without replaying the stream.
10. `errors.Is` and `errors.As` reach category, retryability and the wrapped cause through at least one layer of wrapping.
11. The terminal error event payload and the pre-stream returned error expose the **same** taxonomy — one vocabulary, two delivery paths — and a caller that only ever inspects one of them can still classify every failure the taxonomy defines.
12. Every test-list item of AI-19.1 … AI-19.5 is taken red → green → refactored, in order, with both outputs recorded in `tasks.md`.
13. `make test` in `backend/agent/` is green under `-race`, its output recorded; both import guards pass and `go.mod` still carries zero requires.

## Notes for the following phases

- **`spec.md`** — new capability `ai-provider-errors`. Requirement IDs `R-AIP-0NN`, scenarios `S-AIP-0NN`, RFC 2119 + Given/When/Then, every scenario independently verifiable by a test. Two requirements deserve unusual care: the delivery/partial-output perpendicularity (Decision 3), and the "human text is useful without a body or a credential" claim (Decision 2).
- **`design.md`** — owns every Go spelling: the category type and its members, the sentinel set, the failure type, its two constructors, its accessors, the retry-after presence idiom, the bounded raw-label filter, and the terminal-error payload's satisfaction of AI-14's sealed interface. It must also re-verify the AI-14 sealed-interface assumption against landed code.
- **`tasks.md`** — five phases, one per node; one task per test-list item; red and green evidence recorded per item; the AI-19.6 split trigger checked against the landed category count; the review workload forecast under the session's `exception-ok` strategy.
- **Sequencing** — `sdd-spec` and `sdd-design` may run now. `sdd-apply` MUST NOT start until AI-14 and AI-15 have landed in this worktree.
