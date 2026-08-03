# Proposal: AI-23 — Provider conformance suite

> **Milestone**: AI-23 (doc 0002, Wave 3 final) · **8 leaves**: AI-23.1 … AI-23.8
> **Depends on**: AI-19, AI-21, AI-22, AI-03 (all shipped) · **Blocks**: AI-24, AI-38

## Intent

Today every Layer 1 behavior is proven only against the `ai` package's own tests. Nothing states what a *concrete adapter* must do, so AI-24's first vendor adapter would invent its own assertions and AI-38 would have no artifact to assert against. AI-23 turns those scattered rules into one suite any provider factory plugs into, so the second adapter is cheap and an omission is visible in the graph rather than in production.

## Scope

### In Scope

- **AI-23.1** pluggable skeleton: a provider `Factory` seam, case marking (required vs optional-capability) per `ai-minimum-capabilities` § 11, reported-never-silent skips.
- **AI-23.2** text + lifecycle: ordering, 1..N contiguity, exact reconstruction incl. a multi-byte rune split; empty completion legal.
- **AI-23.3** tool calls: fragmented interleave, zero-delta whole call, observable ordinal, mixed content + tool-call finish reason.
- **AI-23.4** terminal + error: exactly-one-terminal, partial-output discriminator, exhaustive iteration of `FailureCategories()` (9).
- **AI-23.5** cancellation + closure: bounded close and no leak via AI-22.4, abandoned-then-cancelled saturated drop per AI-20.3.
- **AI-23.7** redaction: a planted sentinel appears in no event, error string, or failure output.
- **AI-23.8** optional capabilities: `CAP-O-01` reasoning, `CAP-O-02` token counting, `CAP-O-03` cache-boundary honoring, plus finish-reason and usage cases that are **required** because they exercise `CAP-R-03`.
- **AI-23.6** capability record: total over `CAP-R-01…05` + `CAP-O-01…03`, four-value outcome set, standing from AI-03, verdict rule.
- AI-21's fake as first subject, passing.

### Out of Scope

- Live API credentials or network — the suite is deterministic by construction (AI-39 owns the one credentialed test).
- The first adapter's own reasoning/usage mapping (AI-29, AI-31) and its private hardening (AI-36).
- Transcript replay against a real adapter (AI-38) and report publication (AI-40.2).
- **Deferred but related**: no new optional capability is admitted here; a fourth arrives only by amendment to `ai-minimum-capabilities` § 13 rule 1.

## Capabilities

### New Capabilities

- `ai-provider-conformance-suite`: the factory seam, the case set and its required/optional marking, the capability record shape as the suite emits it, and the pass/fail/inconclusive verdict rule.

### Modified Capabilities

- None — unless risk **R3** resolves toward an exported finish-reason enumerator, which adds a delta to `ai-completion-metadata`. sdd-design decides and, if so, sdd-spec emits that delta.

## Approach

Script-driven factory (exploration's recommended option). The suite's scenario language is AI-21's existing `agenttest.Script` / `Step` / `Emit` / `Hold` / `Gate` vocabulary, so AI-21's factory is a one-line wrap and no second copy of the vocabulary exists. Assertions delegate to AI-22's `DrainAndRecord`, `RequireSameEvents`, `RequireValidStream` and `RequireNoGoroutineLeak`. Cases live in a table keyed by the capability they exercise, which is what makes the marking rule mechanical and the record total by construction. The redaction sentinel is carried on the factory's config value; the fake's factory adapter plants it into script payloads and failure-report fields, needing zero new AI-21 surface. Package placement (inside `agenttest` vs a sibling `agentconformance`) and the exact `Factory` signature are design decisions, not proposal decisions.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/agentconformance/` **or** `agenttest/conformance_*.go` | New | Suite skeleton, case tables, capability record, 8 leaves' cases |
| `backend/agent/src/agenttest/` | Consumed | Fake provider and stream kit used unchanged |
| `backend/agent/src/ai/finish_reason.go` | Modified (conditional) | Only if R3 resolves toward an exported enumerator |
| `openspec/specs/ai-provider-conformance-suite/spec.md` | New | Live contract home |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| **R1** `CAP-O-03` (cache-boundary honoring) has no askable seam — only `TokenCounter` is discoverable by type assertion — yet AI-03 § 12 requires AI-23.8 to cover all three optional subjects and § 10 requires a total record | High | Factory declares its expected optional capabilities alongside the provider; `absent` is then a conclusion, not a silent skip. sdd-design must fix this seam |
| **R2** doc 0002 AI-23.7 says the sentinel is planted "through the provider factory's configuration", but AI-21's `NewProvider` takes only scripts | Med | Sentinel lives on the suite-owned factory config; each subject's factory adapter decides how to wire it. Fake wires it into script payloads |
| **R3** `FinishReason` has no exported enumerator (unlike `FailureCategories()`/`EventKinds()`), so AI-23.8's exhaustive iteration risks silent drift | Med | Either hand-list the 7 values behind a drift-guard test, or add an exported enumerator to `ai` (same module, no new dependency, no ADR) |
| **R4** AI-21 and AI-22 openspec artifacts are absent from this worktree despite their code being landed and verified | Med | Backfill or confirm placement before archive; AI-23's spec must not cite a nonexistent spec file |
| **R5** Estimated **~2800–3200** changed lines; wave running total (AI-21 2810 + AI-22 2542) reaches **~8150–8550** against a 5000 budget | High | `size:exception` and `single-pr` already accepted for this wave; tracked, not gated. Correctness governs — no scope truncation to hit a number |

## Rollback Plan

The suite is additive and has no consumer yet (AI-24 and AI-38 are unstarted), so rollback is deletion, not migration.

1. `git revert` the AI-23 commit range on `feat/2026-08-02-cachicamas-ai-layer1-wave-3`, or delete the new conformance package directory.
2. `agenttest` and `ai` are untouched by default; if R3 added an exported enumerator, that addition is purely additive and backward compatible and may be kept or reverted independently.
3. Re-run `make test` from `backend/agent/` to confirm AI-21/AI-22 remain green.
4. Move `openspec/changes/cachicamas-ai-conformance-suite/` aside; no spec is promoted to `openspec/specs/` until archive, so no live contract is orphaned.

## Dependencies

- AI-19 (`FailureCategories()`, `Failure.Error()` redaction), AI-21 (fake provider), AI-22 (stream test kit), AI-03 (`ai-minimum-capabilities` §§ 9–12) — all shipped in-repo.
- Standard library plus this module's own `ai` and `agenttest` packages only. **No new top-level Go dependency**, so no ADR gate is triggered.

## Success Criteria

- [ ] A provider factory runs the whole suite with **zero copied assertions**; the AI-21 fake is the first subject and passes every required case.
- [ ] Every case is marked required or optional by `ai-minimum-capabilities` § 11 alone; every skipped optional case is reported.
- [ ] The emitted capability record carries one entry per `CAP-R-01…05` and `CAP-O-01…03`, uses the four-value outcome set with `absent` distinct from `not exercised`, and takes standing from AI-03 rather than from the run.
- [ ] A record with any `not exercised` entry is **inconclusive**; a failed required entry cannot pass.
- [ ] All 9 failure categories and all 7 finish reasons are iterated exhaustively, so a new vocabulary member cannot land without a suite case.
- [ ] A planted sentinel appears in no event, no error string, and no test-failure output.
- [ ] `make test` is green from `backend/agent/` with no goroutine leak reported.
