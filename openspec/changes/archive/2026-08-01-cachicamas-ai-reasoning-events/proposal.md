# Proposal: cachicamas-ai-reasoning-events (AI-17)

## Intent

Layer 1 owns reasoning content (AI-07) but has no way to stream it. Without AI-17 a provider adapter must drop reasoning mid-stream or smuggle it through the text family — the exact defect doc 0001 § 3.1 records against the retired Layer 1 for this same content kind. AI-17 defines a structurally distinct reasoning block family so reasoning can never arrive as assistant text, and so AI-07's opaque round-trip token survives the event boundary byte-identically (a mangled token breaks turn two of every tool-using conversation).

## Scope

### In Scope

- Three new event kinds — reasoning block start / delta / end — registered in AI-14's envelope registry, each stamped with an explicit block index.
- Deltas carry reasoning-**text** byte fragments only: never accumulated text, never token bytes. Zero-delta blocks are legal.
- Round-trip token delivered whole on block end, reusing AI-07's `Token() ([]byte, bool)` presence/absence contract verbatim.
- Explicit redacted signal on the block-**start** event.
- Tests over AI-07's existing `opaqueTokens()` byte classes, plus redacted, token-only (empty-text) and rune-splitting-delta shapes.

### Out of Scope

- A public accumulation/reconstruction API (doc 0001 § 4.3 invariant 1: consumers accumulate; tests use a test-local helper).
- Any edit to `reasoning_content.go`, `ReasoningState`, or `MaxReasoningTokenLen`.
- Provider emission (AI-22+), and the text (AI-16) / tool-call (AI-18) families themselves.

## Capabilities

### New Capabilities

- `ai-reasoning-events`: streamed reasoning block lifecycle, block indexing, token delivery position, redacted and token-only wire shapes.

### Modified Capabilities

- None. `ai-reasoning-content` is consumed unchanged.

## Approach

Exploration approach 1 — an independent, structurally-parallel event family mirroring AI-16's text-block shape while reusing AI-07's contracts verbatim. Rejected: one generic block event with a family flag (violates AI-17.1 item 2). Three decisions are **locked here**, not deferred to design:

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | The redacted bit rides the **block-start** event, and only there. No event stores a `ReasoningState`; the block's state is derived from the reconstructed block exactly as `Reasoning.State()` derives it. | Redaction is not derivable from bytes (AI-07) and a consumer must know *before* rendering the first delta that the payload is withheld plaintext. Storing it once mirrors AI-07 setting it only at construction, and keeps the codebase's "no second copy two writers can disagree about" rule (`Part.Kind`, `ToolCalls()` ordinal). |
| D2 | The round-trip token arrives **whole on the block-end event**. It is never fragmented across deltas. | The token is one atomic opaque blob; fragmenting invents reassembly for non-text bytes with no stated need. Only the end event can express AI-17.2 item 2 (no token vs. empty token) via the two-result accessor. |
| D3 | Reasoning blocks use **shared block indexing with text blocks** — one contiguous per-stream block-index space, consistent with AI-16 and AI-18. | With per-family spaces, "text block 0" and "reasoning block 0" collide, so a consumer keying by index alone merges reasoning into assistant text — precisely what AI-17.1 item 2 forbids. AI-18's tool-call *ordinal* stays a distinct concept. |

**Coordination requirement**: D3 is shared ground with AI-16 and AI-18, which are running propose in parallel. Their decisions were not yet in Engram or on disk when this was written; the shared-space reading was chosen as the simplest one that invents no second index space. `sdd-spec` and `sdd-design` MUST cross-check D3 against the AI-16 and AI-18 artifacts and reconcile before apply.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/ai/reasoning_event*.go` (+ `_test.go`) | New | The reasoning block start/delta/end payloads, constructors and accessors. |
| AI-14's event-kind registry + exhaustiveness guard | Modified | Three registry entries appended, as AI-07/AI-09 appended to `PartKind`. |
| `backend/agent/src/ai/reasoning_content.go` | Untouched | Source of truth for token/state; read-only reference. |
| `backend/agent/src/ai/reasoning_content_test.go` | Reused | `opaqueTokens()` is the byte-class fixture, not reinvented. |
| `openspec/specs/ai-reasoning-events/spec.md` | New | Promoted at archive. |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| AI-16/AI-18 land a different block-index scheme (D3), producing three PRs that each pass their own tests and only break on mixed-content streams. | Med | D3 is stated as a coordination requirement; `sdd-spec`/`sdd-design` must cross-check before apply. |
| Apply is hard-blocked: AI-14/AI-15/AI-16 are not on disk in this worktree. | High | Propose/spec/design run against the charter only; re-verify signatures against landed AI-14/AI-15 code before apply. |
| The delta fragment reuses `Reasoning.validate`'s emptiness/`MaxTextLen` rules, breaking zero-length and whitespace-only fragments. | Med | Spec states the delta fragment carries no content-part rule; a fragment is not a complete part. |
| Redaction lost or duplicated on the wire, collapsing redacted vs. token-only. | Med | D1 makes it one explicit signal in one place, asserted by AI-17.3 tests. |

## Rollback Plan

Revert the feature branch. The change is purely additive: three registry entries plus new files. Reverting removes the reasoning event kinds and leaves AI-14/AI-15/AI-16 intact; no shipped consumer exists (first provider is AI-22), no persisted data, no migration.

## Dependencies

- **AI-07** (`ai-reasoning-content`, shipped) — `Reasoning`, `ReasoningState`, `Token()`, `MaxReasoningTokenLen`, `opaqueTokens()`.
- **AI-15** (`cachicamas-ai-response-events`) and transitively **AI-14** — event envelope, per-stream sequencing, kind registry. Not on disk yet.
- **AI-16** (`cachicamas-ai-text-events`) — the text-block shape this family mirrors and shares an index space with.
- doc 0001 § 4.3 invariants; doc 0002 §§ AI-17.1–AI-17.3.

## Success Criteria

- [ ] Reasoning block start/delta/end are three distinct registered event kinds; no boolean flag on a text event, and `go doc` exposes no way to build a text event carrying reasoning.
- [ ] Events for two interleaved blocks are attributable by block index alone, and reasoning/text indices never collide (D3).
- [ ] Concatenated deltas reconstruct the reasoning text byte-exactly, including a delta boundary that splits a multi-byte rune; a zero-delta block reconstructs to empty text.
- [ ] The token arrives whole on block end and is byte-identical for every class in `opaqueTokens()`; a block with no token is valid and distinguishable from one with an empty token.
- [ ] A redacted block signals redaction on its start event and reconstructs to `ReasoningStateRedacted`; a token-and-no-text block reconstructs to `ReasoningStateTokenOnly`.
- [ ] `make test` from `backend/agent/` is GREEN (`go test -race -v ./...`), with tests written RED first.
- [ ] `reasoning_content.go` and its test file are unchanged.
