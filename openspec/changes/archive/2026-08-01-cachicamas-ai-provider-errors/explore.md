# Exploration — the provider error taxonomy and the terminal error event

> **Change**: `cachicamas-ai-provider-errors`
> **Milestone**: AI-19 — Define the provider error taxonomy and the terminal error event
> **Nodes**: AI-19.1 … AI-19.5 (all `[leaf]`)
> **Status**: explored
> **Phase**: exploration
> **Project**: cachicamas (witsaba) · **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Date**: 2026-08-01
> **Wave**: 2 — Stream (this milestone is the wave's **keystone**)
> **Depends on**: AI-04, AI-14, AI-15 · **Blocks**: AI-20, AI-21, AI-23, AI-27, AI-32
> **Engram mirror**: `sdd/cachicamas-ai-provider-errors/explore`
> **Provenance note**: the exploring executor had no filesystem write tool; this file is that
> exploration's content persisted verbatim afterwards to satisfy the project's `hybrid`
> artifact-store requirement. The Engram observation is the original.

---

## Current State

`backend/agent/src/ai/` (Go 1.26.3, module `github.com/cachicamas/backend/agent`) is Layer 1 of the agent stack. As of this exploration, AI-01 through AI-13 are shipped (archived). Neither AI-14 (event envelope) nor AI-15 (response lifecycle events) is on disk yet in this worktree — AI-19's apply phase is explicitly sequenced after both land, so this exploration reasons from their charters/decision artifacts, not their code.

Two established architectural idioms recur across every shipped file and should be reused verbatim rather than reinvented:

1. **Sealed sum type via unexported payload interface** (`content_part.go`): a value type (`Part`) holds one unexported `partPayload` field; the interface declares `kind() PartKind` and `validate(at Path) *Violation`; both are unexported so only types inside package `ai` can implement it. The kind is *derived* from the payload on every call, never stored — "a kind and a payload can never disagree." AI-14's event envelope is near-certain to repeat this exact shape (its charter literally says "kind derived from payload... a caller cannot set a kind that disagrees with what the event carries" — same wording).
2. **Closed, append-only, zero-value-invalid vocabulary** (`finish_reason.go`, `content_part.go`'s `PartKind`): `uint8` type, `iota` starting at a blank `_` (zero value is not a member), a `nameLimit` bound constant that stays last, a `[limit]string` array (not a map — "nothing in this package may let an unordered iteration decide anything"), a `Validate` that reports `ErrNotInVocabulary`, and — critically — a **total normalizer function** (`NormalizeFinishReason`) that never errors and maps anything unrecognized to the vocabulary's "unknown" member while never panicking on a vendor's next surprise string.
3. **Presence-vs-zero as a first-class distinct fact** (`usage.go`'s `TokenCount`): zero value means "absent" (not "zero"), built via a constructor (`Tokens(n)`), read via a two-result accessor `(count int64, present bool)` — the exact shape needed for AI-19.3's typed retry-after duration ("never re-parsed from message text").
4. **Caller-contract failure vocabulary** (`validation.go`, AI-04, shipped): sentinel errors (`ErrEmpty`, `ErrNotInVocabulary`, …) as rule *classes*, one concrete `*Violation` type carrying `rule error` + `at Path`, reached via `errors.Is` (class) and `errors.As` (position) through any number of wrappers. `Violation.Error()` renders only from the registered class's own text plus structural (filtered) position data — **never** `rule.Error()` — which is explicitly called out as "the first thing in this package that formats caller data" and the origin of the redaction posture (V-FAIL-13).

## Binding vocabulary already decided (openspec/changes/archive)

`2026-07-31-cachicamas-ai-contract-vocabulary/decision.md` (AI-01, shipped) already defines and closes ownership of every noun AI-19 needs — AI-19 must cite these rows, not redefine them:

- `V-FAIL-05` provider/transport failure (AI-19) — anything after a valid request leaves the process; not the caller's bug, not knowable without I/O.
- `V-FAIL-06` failure category — the closed classification a consumer switches on.
- `V-FAIL-07` retryability — knowable only where the wire failure is (Layer 1); whether to *act* on it is Layer 2's (out of scope here, confirmed by charter).
- `V-FAIL-08` partial output — content already delivered before failure; preserved, never discarded.
- `V-FAIL-09` partial-output discriminator — the fact a correct retry decision turns on.
- `V-FAIL-10` **terminal error event** — "carrying the failure's category, its retryability, its partial-output discriminator, and its wrapped cause — constructible by an adapter in another package." This is C4 verbatim, and it is owned by AI-19, not AI-14. AI-14 owns the generic `V-STR-18` terminal-event container/invariants (exactly one per stream, nothing follows it, its two instances are `V-STR-20` completion and `V-FAIL-10` error); AI-19 owns the error instance's payload.
- `V-FAIL-11` / `V-FAIL-12` pre-stream / mid-stream delivery — AI-02.1 (shipped decision) fixed the *boundary* as carrier handover, not first event, and fixed that both paths expose the same category/retryability/partial-output — this is exactly AI-19.5's job to implement, not to decide.
- `V-FAIL-13` safe metadata — status classes, categories, retry hints, bounded/sanitized excerpts; never a credential, header, raw body, or model content.
- `V-FAIL-15` retry policy (Layer 1 half, AI-35, out of scope) — only the one clause AI-19 must not violate: "the partial-output case is never retried at Layer 1," which depends on AI-19's discriminator being unpolluted by delivery.

`2026-07-31-cachicamas-ai-stream-lifecycle/decision.md` (AI-02.1, shipped) already answered the delivery-boundary question AI-19.5 is graded against: **the boundary is carrier handover**, not the first event — a stream handed over that fails before emitting content is *mid-stream*, not pre-stream (this is the intuition trap the spec calls out explicitly, S-AIS-029). It also states the exact wording AI-19 must reuse for cancellation: "the producer offers a terminal error event carrying the category AI-19 assigns to cancellation, without waiting." Cancellation is therefore a *required* category member, not optional — confirmed independently by the charter's own minimum list.

## AI-04 composition — sibling, not shared vocabulary

AI-04's `ruleClasses`/`Violation` machinery is the **caller-contract** failure vocabulary (V-FAIL-01…04). AI-19's category vocabulary (V-FAIL-06) is a **different, orthogonal** closed vocabulary for provider/transport failures — the register's own diagram (§6) puts them on perpendicular axes and states the boundary test is "decidable without I/O," not severity or blame. AI-19 must **not** add its categories to AI-04's `ruleClasses` slice and must **not** reuse `Violation`/`Invalid` for provider failures (those are typed to `error` rule classes, not to a closed category enum with retry metadata). AI-19 should instead build a **sibling** type following the *same shape* (sentinel-per-category + one concrete carrier type + `errors.Is`/`errors.As` reach-through), for consistency and reviewer familiarity, not for code reuse.

## Affected Areas

- `backend/agent/src/ai/errors.go` (new, name illustrative) — the category vocabulary, the failure/terminal-error type, retry-hint carrier, partial-output discriminator. This is the file AI-19 delivers.
- `backend/agent/src/ai/validation.go` — read-only reference for the sentinel + single-concrete-type + `errors.Is`/`errors.As` idiom AI-19 should mirror; not modified by AI-19 (different vocabulary).
- `backend/agent/src/ai/finish_reason.go` — read-only reference for the closed-vocabulary uint8/iota/array/`Validate`/total-normalizer idiom.
- `backend/agent/src/ai/usage.go` — read-only reference for the presence-vs-zero two-result accessor idiom (`TokenCount`), directly reusable shape for the retry-after duration.
- `backend/agent/src/ai/content_part.go` — read-only reference for the sealed-payload/kind-derived-from-payload idiom that AI-14's envelope (and therefore AI-19's terminal-error payload) will almost certainly repeat.
- AI-14 (`cachicamas-ai-event-envelope`, not yet applied) and AI-15 (`cachicamas-ai-response-events`, not yet applied) — AI-19's terminal error event is a payload that must satisfy whatever unexported envelope-payload interface AI-14 ships. Since that interface will almost certainly be **sealed** (unexported methods, per the established idiom), AI-19's payload type must live inside package `ai` itself — which it does, closing C4 by construction: the payload type and the interface it satisfies are compiled together, so "no error payload type existed and the interface was sealed" cannot recur.
- `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 851-1057 — the binding charter text for this wave and this milestone.

## Approaches

1. **Mirror AI-04's sentinel + single-concrete-type shape exactly (recommended)** — one closed `uint8` category enum (iota, zero-value invalid, array-indexed names, `Validate`), one exported concrete failure type (e.g. `Failure`) implementing `error` + `Unwrap() error`, with `Category() Category`, `Retryable() bool`, `RetryAfter() (time.Duration, bool)` (presence-vs-zero, mirroring `TokenCount`), `PartialOutput() PartialOutputDiscriminator`, and category-named sentinel errors (e.g. `ErrRateLimited`) so `errors.Is(err, ai.ErrRateLimited)` works directly without unwrapping to the concrete type first — while `errors.As(err, &f)` reaches the full `Failure` for retry hints and safe metadata. This one type is what both delivery paths carry: returned directly pre-stream, wrapped as the terminal event's payload mid-stream (AI-19.5's "one vocabulary, two delivery paths" is then a structural fact, not a maintained parallel implementation).
   - Pros: matches an idiom reviewers and AI-23's later exhaustiveness checks already know from AI-04/AI-13; `errors.Is`/`errors.As` reach-through is free; the shared type physically prevents the two delivery paths from drifting apart.
   - Cons: none identified — this is the path of least resistance the codebase itself argues for.
   - Effort: Low (the pattern is copy-adapt, not invention).

2. **Two separate types — a lightweight pre-stream error and a richer terminal-event payload wrapping it** — pre-stream returns a plain typed error; the terminal event's payload embeds/wraps it plus the partial-output discriminator (which is meaningless pre-stream).
   - Pros: keeps the pre-stream error "light" (no stream-specific fields on it).
   - Cons: directly risks failing AI-19.5's own acceptance test ("the terminal error event payload and the pre-stream returned error expose the same taxonomy") unless the wrapping is done with extreme discipline; two types is exactly the "second, parallel failure vocabulary" AI-04's design commentary (V-FAIL-03) warns against by analogy; partial-output discriminator has a legitimate "pre-stream" state anyway (AI-19.4 test 1 lists it as one of three shapes), so there's no real field mismatch to justify the split.
   - Effort: Medium, and the risk is a spec-conformance defect, not just extra code.

3. **Category as an open string with a small set of well-known constants (no closed enum)** — categories are strings; a handful of exported string constants are provided for convenience.
   - Pros: trivially extensible without a package release.
   - Cons: directly violates the charter's explicit requirement ("category membership is closed and enumerable, so a later exhaustiveness assertion (AI-23) can iterate it exhaustively") and V-FAIL-06's "closed classification... closed, because an open one pushes classification back into message parsing" — this is the specific anti-pattern the register calls out as the failure mode of "every normalizer that crashed on a novel value."
   - Effort: Low, but rejected on spec grounds, not effort.

## Recommendation

Approach 1. It is not really a choice among competing designs so much as the only approach consistent with (a) the shipped AI-04/AI-13 idioms this same package already committed to, (b) the register's explicit closed-vocabulary requirement, and (c) AI-19.5's dual-path exhaustiveness test, which all but requires one shared concrete type. The open design decisions for `sdd-propose`/`sdd-design` to settle are naming (Go identifiers — out of scope for the vocabulary register per V-AIV-009, in scope for this milestone's own SDD) and the exact shape of the "raw provider label preserved for diagnostics" clause (AI-19.2 test 2) — see Open Questions below.

## Open Questions

> All four are resolved as explicit decisions in `proposal.md` § "The four decisions this proposal commits to".

1. **How does "raw provider label preserved for diagnostics" (AI-19.2.2) get carried?** Unlike `FinishReason`'s `NormalizeFinishReason`, which has a real, bounded table of known vendor spellings to match against, provider/transport failures don't have an equivalent stable string vocabulary across vendors (HTTP status text, vendor error codes, etc. are unbounded and adapter-specific). Recommend: no total-normalizer-by-string-table like `finish_reason.go`'s; instead the category is *assigned by the adapter* (a later, Wave 4 concern — AI-32), and AI-19 only needs to guarantee the `Failure` type has a place to carry an opaque, safe-metadata-bounded raw label string when category is Unknown. This should be confirmed in `sdd-design`.
2. **Does `Failure.Error()` ever call the wrapped cause's `Error()`?** `validation.go`'s `Violation.Error()` deliberately never renders `rule.Error()` to keep redaction a type property. AI-19's acceptance criterion ("error strings never require secrets or response bodies to be useful; wrapped causes remain inspectable") reads the same way: `Unwrap()` should expose the cause for callers who want it, but the default `Error()` string should be built only from category text + safe metadata, never the cause's own text. Recommend following `Violation.Error()`'s precedent exactly. Full adversarial redaction sweeps are explicitly AI-36's (later wave) — AI-19 only needs the type-level default to be safe, not a proven-adversarial guarantee.
3. **Exact shape of the partial-output discriminator's three states.** AI-19.4 names them descriptively (pre-stream failure; mid-stream, zero output; mid-stream, output-then-failure) but doesn't fix a Go shape. Two candidates: (a) one 3-member closed enum living on `Failure` itself, populated identically on both delivery paths (pre-stream constructors set it to the "pre-stream" member by construction); (b) a 2-state bool (`hadOutput bool`) plus the delivery axis being a separate, orthogonal fact the caller already knows structurally (did I get a stream at all?). The register's own diagram (§7) draws delivery and partial-output as **orthogonal axes**, which argues for (b) — a pre-stream `Failure` simply never needs to report "did output precede me," because by definition none did and the caller already knows it never got a stream. Needs `sdd-design` to pick, referencing AI-19.5's "one vocabulary, two paths" test carefully so the chosen shape doesn't reintroduce the delivery/partial-output conflation the register calls out as the historical defect (G8).
4. **Retryability: bool vs richer signal.** Charter says "every category carries a machine-readable retryability signal" (singular signal) but also separately requires a typed retry-after duration when supplied. Likely a simple `Retryable() bool` (or a fixed per-category default) plus an independent `RetryAfter() (time.Duration, bool)` is sufficient and matches "the classification lives here... the decision to retry lives one layer up" — Layer 1 doesn't need a richer backoff-shape, just the wire fact.

## Risks

- **Engram retrieval unavailable during exploration**: the exploring executor was exposed only `mem_save` — `mem_search`/`mem_get_observation` returned "No such tool available." All prior-context needs were satisfied instead directly from the authoritative filesystem source (`openspec/changes/archive/2026-07-31-cachicamas-ai-contract-vocabulary/decision.md` and `.../cachicamas-ai-stream-lifecycle/decision.md`, plus the shipped `validation.go`/`finish_reason.go`/`usage.go`), which under this project's `hybrid` artifact-store mode are equally authoritative. If a genuinely Engram-only prior decision exists that isn't mirrored in the openspec archive, it was not retrieved.
- **No filesystem write tool was available to the exploring executor.** Only the Engram save (`sdd/cachicamas-ai-provider-errors/explore`) was completed at exploration time; this file was written afterwards by the `sdd-propose` executor from that observation.
- AI-14/AI-15 are not yet on disk. This exploration reasoned from their charters and from the shipped `content_part.go` sealing idiom as the best available proxy for what their envelope/payload interface will look like. If AI-14 ships a materially different (e.g. exported, non-sealed) payload interface, the "payload type must live in-package" argument for why AI-19 resolves C4 by construction would need re-checking at `sdd-design` time for this change.
- The category-count-vs-metadata split trigger ("Split if: category-specific metadata... grows the list past seven items") should be watched during `sdd-spec`/`sdd-design`: rate-limit reset and quota identity are named as the first candidates likely to trigger a spin-off (AI-19.6).

## Ready for Proposal

Yes. The binding vocabulary (AI-01), the delivery-boundary decision (AI-02.1), and the established in-package idioms (AI-04, AI-13, content-part sealing) give `sdd-propose`/`sdd-spec` everything needed to commit to Approach 1 without further exploration. The three open questions above are `sdd-design`-level decisions, not blockers to writing the proposal.
