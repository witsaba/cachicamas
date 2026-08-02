# Exploration: AI-20 — the model provider interface (Wave 2 join point)

> **Change**: `cachicamas-ai-model-provider` · **Milestone**: AI-20 · **Nodes**: AI-20.1 … AI-20.5
> **Phase**: explore · **Project**: cachicamas (witsaba)
> **Source of record**: Engram `sdd/cachicamas-ai-model-provider/explore` (obs #2234)
> **Provenance**: the explore phase ran without filesystem write access; this file is that
> artifact persisted verbatim into the change folder by `sdd-propose` on 2026-08-01.

> [!NOTE]
> **Drift note (2026-08-01, added by `sdd-propose`, body left unmodified).** The
> "Current State" section below records `src/agenttest/` as holding only `doc.go` and
> `import_compile_test.go`. Wave 1 has since landed `cache_boundary_test.go`,
> `content_part_test.go` and `request_test.go` there. This strengthens rather than weakens
> the exploration's finding: the external-consumer pattern AI-20.1 item 2 needs is now
> established practice in that package, not a first use.
>
> A second, unrelated lineage note: the change name `cachicamas-ai-model-provider` was also
> used by a retired pre-2026-07-30 SDD cycle under the old AI-16 numbering. Those artifacts
> live under Engram topic keys prefixed `retired/old-plan/`. Everything below is the live
> AI-20 lineage.

---

## Current State

`backend/agent/src/ai` (module `github.com/cachicamas/backend/agent`) is Layer 1, still empty of anything AI-14/AI-19/AI-20 will need to declare a stream. What is actually on disk today, all Wave 0/1, shipped:

- **AI-04** — `validation.go`: the caller-contract error taxonomy. Two axes: `errors.Is(err, ErrEmpty)` (rule class, closed sentinel set) and `errors.As(err, &v)` (position). `FirstFailure(rules ...Rule) error` runs a rule list in documented order and returns the first `*Violation`.
- **AI-05/06/07/08/09** (implied shipped, not read in depth) — `message.go`, `content_part.go`, `tool.go`, `tool_call.go`, `tool_choice.go`, `tool_result.go`, `tool_set.go`, `reasoning_content.go`, `system_instruction.go`, `role.go`, `finish_reason.go`, `usage.go`.
- **AI-10/AI-12** — `request.go` + `request_extension.go`: `Request` is a sealed, immutable value. `NewRequest(model, messages, opts...)` and `Request.With(opts...)` are the *only* two constructors, both running `FirstFailure(draft.rules()...)` then `freeze()`. On failure both return the **zero `Request`**. `requestDraft.rules()` and `freeze()` are private to package `ai`. `ProviderExtension` (AI-12.3) is the typed, namespaced, opaque provider escape hatch (`WithProviderExtension`), attached to `Request` without widening its constructor surface.
- **AI-11** — `cache_boundary.go`: `MaxCacheBoundaries = 4`, `CacheRegion`, `Request.CacheBoundaries()`. Markers are advisory (`CAP-O-03`), enforced only by the cap rule.
- `import_boundary_test.go` — Guard A (Layer 1 import purity, `go list -deps -test` deny-by-default) is live and already documents the exact pattern AI-20.4 will need to imitate for its own guard.
- `openspec/specs/ai-stream-lifecycle/spec.md` (AI-02.1, **decision-only, zero Go**) and `openspec/specs/ai-minimum-capabilities/spec.md` (AI-03.1, **decision-only, zero Go**) are both live specs in `openspec/specs/`, not archived pointers — they are the actual contract AI-20 must translate into a Go declaration.

`backend/agent/src/agenttest` today holds only `doc.go` and `import_compile_test.go` — a blank-import compile proof, nothing else. Its `doc.go` **already forward-declares AI-20.4's guard**: "A later milestone adds a signature guard over the provider interface that resolves `../ai` from its own source file via `runtime.Caller`... See ADR 0005 § D2 and Guard C." This is the exact mechanism the charter's AI-20.4 asks for, already named as a load-bearing structural constraint before AI-20 exists.

**Not yet on disk, and out of scope for this exploration by the wave's own ordering constraint:** AI-14 (event envelope — the payload the AI-02 carrier will carry) and AI-19 (provider error taxonomy — the terminal error event's shape). AI-20's apply runs strictly after both land. This exploration is written against their **charter contracts** (doc 0002 lines 855-909 for AI-14, 1035-1116 region for AI-19), not against implementations.

**Engram retrieval tools (`mem_search`/`mem_get_observation`) were not available in this execution's tool grant** — only `mem_save` was. I substituted by reading the four live openspec spec files directly (`ai-stream-lifecycle/spec.md`, `ai-minimum-capabilities/spec.md`, `ai-cache-breakpoints` implied via `cache_boundary.go`, ADR 0005 directly), which is the same authoritative source Engram would have surfaced, since `openspec/specs/` is the project's own stated source of truth. No content gap results from this substitution; it is a tooling-path difference, not a coverage gap. This has no bearing on the artifact's correctness, only on how it was retrieved.

## Affected Areas (files this milestone will create — none exist yet)

- `backend/agent/src/ai/provider.go` (name illustrative) — AI-20.1: the `ModelProvider`-shaped interface, its GoDoc restating AI-02.1 § 9's eight ownership/cancellation/loss statements verbatim, and AI-20.5's optional-capability contract types (e.g. a `TokenCounter`-shaped interface).
- `backend/agent/src/agenttest/provider_stub_test.go` (or similar) — AI-20.1 item 2: an external-package stub implementing the interface, compiled and exercised, on the sibling-layout precedent `agenttest/doc.go` already names.
- `backend/agent/src/agenttest/provider_signature_guard_test.go` — AI-20.4: the AST/reflection guard, resolving `../ai` via `runtime.Caller(0)` exactly as `agenttest/doc.go` promises, extending the *pattern* `import_boundary_test.go` already established (deny-by-default, forbidden-first, allowlist-second) to a signature rather than an import set.
- `backend/agent/src/ai/request.go` — a possible small addition: AI-20.2's pre-stream contract needs a way to detect "never validated" (the zero `Request`) from outside `requestDraft`. See Open Question 1.
- No test-double/fake yet — AI-21 (the scripted fake provider) is blocked *by* AI-20, so AI-20.3's mid-stream proof cannot borrow AI-21's fake and must build its own minimal in-file test producer.

## How the shipped pieces compose into the AI-20 signature

The interface's one streaming method has exactly one plausible shape given what already exists:

```go
type ModelProvider interface {
    Stream(ctx context.Context, req Request) (<-chan Event, error)
}
```

- **Input**: `context.Context` (stdlib) + `ai.Request` (AI-10/AI-12, already sealed and self-validating). No vendor type, no wire type — satisfies AI-20.1 item 1 directly, because `Request` was designed from the start as the provider-neutral carrier (`V-REQ-20`).
- **Output carrier**: AI-02.1 § 3 already decided this *is* a receive-only channel, not an iterator, argued at length and cited by identifier (`V-STR-02`). AI-20.1/AI-20.4 only choose the Go spelling; the shape is not open. It will be `<-chan Event` once AI-14 lands `Event`.
- **Output error**: a bare `error`, carrying an AI-04-shaped caller-contract failure pre-stream, or later an AI-19-shaped provider/transport failure pre-stream — both pre-handover, per AI-02.1 § 7's `V-FAIL-11` path. AI-20.1 item 1 does not require the error to be a distinguished type at the signature level (the taxonomy is behind `error`, resolved by `errors.As`), matching AI-04's own two-axis design.

The optional-capability mechanism (AI-20.5) is **inherited, not invented**: AI-03.1 § 9 already fixed the mechanism *family* as an inherited constraint from doc 0001/doc 0002/ADR 0005 § D4 row G3 — "an additional contract, declared beside the provider contract, one per capability, asserted on the provider value... The provider interface never widens." AI-20 only spells it. The idiomatic Go form matching AI-03's own wording ("an additional contract asserted on the provider value") is the standard optional-interface pattern already used elsewhere in Go's stdlib (`http.Flusher`, `io.ReaderFrom`):

```go
type TokenCounter interface {
    CountTokens(ctx context.Context, req Request) (int, error)
}
```

A consumer does `if tc, ok := provider.(TokenCounter); ok { ... }` — the type assertion *is* the discovery, matching AI-03.1's "advertising is not a flag... the provider value either satisfies the extra contract or it does not, and that fact is the advertisement." `CAP-O-01` (reasoning — no separate discovery needed, reasoning content is always in the neutral `Event`/content vocabulary, never gated) and `CAP-O-03` (cache-boundary honoring) do **not** need discovery interfaces of their own by this same reasoning — only `CAP-O-02` (token counting) is genuinely "askable" in AI-03's sense (§ 3's test), because it is the only optional capability whose presence changes what a consumer can legally *do* (ask for a count) rather than what it can *receive* (reasoning events, cache-marker honoring — both are receive-side and need no discovery, only observation of the stream/record).

## Pre-stream and mid-stream contract enforcement — recommended shape

**AI-20.2 (pre-stream).** Two failure classes must be reported *before* any goroutine or channel exists:
1. An invalid request — a typed AI-04 error.
2. A context already cancelled at call time — reported *after* validation, per AI-02.1 § 7's stated order ("validation runs first... the cancellation category is reported only if the request is valid").

Recommended enforcement: `Stream` begins with a synchronous, allocation-free check (no I/O, no goroutine, no channel) that runs exactly these two gates in order, then and only then spawns the producer goroutine and returns the channel. This is directly testable by asserting nothing observable happens (no goroutine count change, no channel returned) before both gates pass — AI-20.2 item 3.

**AI-20.3 (mid-stream).** AI-02.1 §§ 4-6 already state the ownership/cancellation/loss rules structurally (one sender, one closing site, every exit path, after the last send, every send selects on `ctx.Done()`). AI-20.3's four test items are direct restatements of AI-02.1 §§ 4, 5, 6 as tests, and AI-02.1 § 10 explicitly maps: item 1 → § 4, item 2 → § 5, item 3 → § 6's sanctioned loss path, item 4 → § 5's bounded close. Since AI-21's scripted fake is blocked by AI-20 (cannot exist yet), AI-20.3 needs its **own** minimal test producer inside `provider_test.go` (or similar) that can be told to stall, to saturate its buffer, and to be cancelled mid-send — a smaller, single-purpose double, not a reusable fake. This is worth flagging explicitly in the design so it isn't mistaken for a missing dependency.

## Open Questions

1. **How does `Stream` distinguish a validly-constructed `Request` from the zero-value one, from outside `package ai`?** `Request`'s only constructors (`NewRequest`, `With`) already validate and return the zero `Request` on failure — so the *only* "invalid request" reachable by a caller who follows the type's contract is the zero value (or a value from an ignored error). `requestDraft.rules()` is private. If adapters end up living outside `package ai` (plausible per AI-24+, since ADR 0005 D2 doesn't pin adapter location precisely), they need a public way to ask "is this the constructed zero value." Recommend AI-20.2 (or a small AI-10 follow-up) exposes a minimal `Request.IsZero() bool`-shaped check, mirroring `MessageID.IsZero()`'s precedent, rather than re-running `FirstFailure` from outside the package.
2. **Where does the interface (`provider.go`) physically live relative to the guard's `runtime.Caller` resolution?** `agenttest/doc.go` already commits to `../ai` as the resolved relative path from `agenttest`'s own source file — AI-20.4 must place the interface's file at a location that resolves consistently (i.e., directly in `src/ai/`, not a subpackage), or the promised guard breaks silently exactly as ADR 0005 Guard C warns.
3. **Does AI-20.4's guard need to special-case which file it targets, or scan the whole `src/ai` package for the interface declaration?** The charter says "resolves the Layer 1 source relative to its own source file" (singular), suggesting a named file (`provider.go`) is the deliberate target, not a package-wide AST walk — cheaper and less brittle to unrelated changes elsewhere in `src/ai`.
4. **Bite-proof mechanics for AI-20.4 item 2**: the charter demands "a scratch change to the signature... fails the guard; recorded, dropped" for two separate mutations (adding a vendor type, changing the carrier). This needs a documented manual-mutation-and-revert procedure (matching `import_boundary_test.go`'s own bite-proof precedent in AI-00.3), not a new mechanism — worth pointing the design phase at that existing precedent directly.

## Approaches

1. **Single streaming method + Go-idiomatic optional interfaces (recommended)** — `Stream(ctx, Request) (<-chan Event, error)` plus one small `TokenCounter`-shaped interface for `CAP-O-02` only.
   - Pros: Matches AI-02.1's already-decided carrier exactly (no reopening); matches AI-03.1's already-decided discovery mechanism exactly (no reopening); idiomatic Go (`io.ReaderFrom`-style optional interfaces are a well-understood pattern); minimal new surface — one interface, one method, one optional interface.
   - Cons: Requires the `Request.IsZero()`-shaped addition (Open Question 1) to make AI-20.2 testable from outside `package ai` cleanly; requires AI-20.3 to build its own small test producer rather than reusing AI-21's fake.
   - Effort: Low-Medium — the hard architectural decisions (carrier, discovery mechanism) are already made upstream; AI-20's job is spelling and guarding them, not deciding them.

2. **Widen the core interface with capability query methods (e.g. `SupportsTokenCounting() bool` on `ModelProvider` itself)** — rejected by inheritance, not re-argued here.
   - Pros: none that survive AI-03.1 § 9's "What this excludes" table.
   - Cons: Directly contradicts AI-03.1's already-decided, already-shipped mechanism ("The provider interface never widens" — the exact prohibition AI-20.5 item 3's pin exists to catch). Would require reopening a shipped decision (AI-02.1 § 11 rule 1 / AI-03.1 § 13 rule 1 treat this as an amendment, not a local judgement call).
   - Effort: N/A — not a real option; included only to record why it was rejected, since it is the most tempting wrong turn for a reader new to this milestone.

## Recommendation

Approach 1. There is effectively no design freedom left at AI-20: AI-02.1 fixed the carrier, AI-03.1 fixed the discovery mechanism, AI-10/AI-12 fixed the request shape, and AI-04 fixed the error-reporting idiom. AI-20's actual work is (a) spelling all four decisions into one Go interface, (b) writing the eight-statement package contract into GoDoc verbatim from AI-02.1 § 9, (c) building the AST/reflection signature guard the sibling packages already promise exists, and (d) resolving the two small open questions above (a `Request` zero-check accessor, and AI-20.3's own minimal test producer) — neither of which reopens an upstream decision, both of which are implementation details local to AI-20.

## Risks

- **Sequencing risk (real, already flagged by the orchestrator):** AI-20's apply must land strictly after AI-14 and AI-19, and effectively after AI-15/16/17/18 (the event kinds AI-14's `Event` will carry). Writing `provider.go` against a charter contract rather than a shipped `Event` type means the exact spelling of the channel element type (`<-chan Event` vs. some AI-14-chosen name) cannot be pinned until AI-14 merges. This exploration deliberately did not guess AI-14's type name.
- **Guard fragility (named by ADR 0005 Guard C and `agenttest/doc.go` themselves):** the signature guard's `runtime.Caller`-based path resolution is a silent-failure hazard if `src/ai` or `src/agenttest` is ever reorganized. This is a known, already-documented risk, not a new one this milestone introduces.
- **`Request.IsZero()` gap:** if AI-20.2's pre-stream contract is implemented without exposing some public "was this ever validated" signal on `Request`, either the guard has to live awkwardly inside `package ai` only (constraining where adapters can be written) or AI-20.2's test cannot be honestly proven from `agenttest`. This should be resolved explicitly at spec/design time, not discovered during apply.
- **AI-20.3's test double is not AI-21's fake** — if the tasks/design phase assumes AI-21's fake is available for AI-20.3's mid-stream proof (as the AI-02.1 spec's inheritance table casually suggests), that assumption is wrong given the dependency direction (AI-21 depends on AI-20, not the reverse) and will block apply if not caught now.

## Ready for Proposal

Yes. The upstream decisions (AI-02.1, AI-03.1) are shipped and read as live, current specs; the request shape (AI-10/AI-12) is shipped and read directly from source; ADR 0005 Guard C's structural constraint is already anticipated in `agenttest/doc.go`. The four open questions above are small enough to resolve inside `sdd-propose`/`sdd-design` rather than requiring further exploration. The orchestrator should tell the user: AI-20 has no remaining architectural ambiguity, only implementation-detail decisions to make explicit during design (the `Request.IsZero()` question, and AI-20.3's standalone test producer) — but its apply is still correctly blocked on AI-14 and AI-19 landing first.
