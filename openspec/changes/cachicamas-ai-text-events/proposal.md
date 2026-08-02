# Proposal — indexed text block events

> **Change**: `cachicamas-ai-text-events`
> **Milestone**: AI-16 — Add text delta events (Wave 2 "Stream")
> **Nodes**: AI-16.1 `[leaf]` lifecycle · AI-16.2 `[leaf]` byte fidelity · AI-16.3 `[leaf]` zero-delta blocks
> **Status**: proposed · **Phase**: proposal · **Date**: 2026-08-01 · **Driver**: braejan
> **Charter**: `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` §§ AI-16.1 … AI-16.3 (lines 937–966)
> **Architecture**: `docs/architecture/milestones/0001-cachicamas-agent-stack-v2.md` § 4.3 invariant 1
> **Predecessor**: `exploration.md` (this folder) · Engram `sdd/cachicamas-ai-text-events/explore`
> **Depends on**: AI-15, transitively AI-14. **Parallel with**: AI-17, AI-18. **Blocks**: AI-28.1.1
> **Worktree**: `cachicamas-worktrees/ai-wave-2`, branch `feat/2026-08-01-cachicamas-ai-layer1-wave-2`

---

## Intent

Layer 1 can describe a *finished* text content part (AI-06) but has no way to describe text **arriving**. AI-16 defines the three events a streamed text block produces — start, delta, end — so a consumer can attribute every fragment to its block while blocks interleave, and reconstruct the exact bytes the model produced.

The failure this milestone forecloses is the one doc 0001 § 4.3 names: an event family that carries the accumulated message per token instead of a fragment, which is a data race when shared and quadratic allocation when copied. The second is subtler and is what the charter's rune-split test exists for: a contract that says "text fragment" and means "string fragment" is untrue the first time a provider splits a multi-byte rune across two frames.

## Locked decisions

Stated in one line each so a reviewer can reject the substance before reading `design.md`.

1. **Three separately registered event kinds** — text-block-start, text-delta, text-block-end — not one kind with an internal phase enum. Keeps AI-14.1's "kind is derived from the payload" literally true and keeps the family visible to AI-14's exhaustiveness guard. Exploration Approaches 2 and 3 are rejected there, with reasons.
2. **The block index is stamped by the producer on every one of the three events, never derived.** `tool_call.go`'s derived-ordinal precedent does not apply: at emission time there is no finished sequence to derive a position from. The precedent that applies is AI-14.2's per-stream sequence.
3. **The block index is 1-based, and 0 is the "never stamped" sentinel rejected at construction.** A 0-based index makes an unstamped event indistinguishable from block 0 — the absent-versus-zero collapse AI-13.3 already refused for usage counts.
4. **One block-index space per stream, shared across all content families.** AI-16.1's charter phrase is "attributable to their own block **by index alone**"; per-family numbering makes that false the moment AI-17's reasoning block and AI-16's text block both call themselves block 1. **AI-17 and AI-18 inherit this decision and must not re-decide it** (exploration Open Question 1, resolved here because AI-16 is the first of the three siblings to plan).
5. **A delta carries a raw byte fragment with no text-content validation reused from `textPayload`.** `NewText`'s `ErrEmpty` whitespace rule is a rule about a complete, model-visible value; applying it to a fragment would reject a legal `" "` token and fight AI-16.3. A zero-length delta is legal, and a block with zero deltas is legal and reconstructs to `""`.
6. **`[provisional]` — a delta fragment is capped at `MaxTextLen` bytes (`ErrOutOfRange`), reusing the existing constant, with no new bound invented.** The cap cannot reject anything a legal complete text could produce, since a fragment is never larger than the text it belongs to. `design.md` may drop or change it by recording the reason first.
7. **Layer 1 ships no public accumulation or reconstruction helper.** Doc 0001 § 4.3: consumers accumulate. Byte-exactness is proven by AI-16's own tests with a test-local concatenator.

## Capabilities

### New capabilities

- `ai-text-events`: the three text-block event kinds, their block-index contract, and the byte-fidelity guarantee. Requirement IDs `R-ATE-0NN`, scenario IDs `S-ATE-0NN` — prefix verified unused across `openspec/specs/` (Wave 1 lesson C1: one prefix per capability).

### Modified capabilities

- **None today.** `openspec/specs/` contains no event capability, because AI-14 and AI-15 have not been planned or landed. **Conditional**: if AI-14's `ai-event-envelope` spec lands before AI-16's spec phase, adding three kinds to its closed vocabulary is a spec-level change and requires a delta in this change folder. The spec phase MUST re-check this rather than assume.

## Approach

1. Mirror the AI-06 house style exactly: closed uint8-backed kind constants, an indexed name table (never a map), a sealed payload interface with unexported `kind()`/`validate()`, a `New…` constructor as the only door in, and a typed accessor per kind.
2. Register the three kinds into AI-14's event-kind table and extend AI-14's exhaustiveness guard — this change is **not purely additive**, exactly as AI-07/AI-09 had to touch `content_part.go`.
3. Store the fragment as an immutable `string` of raw bytes documented as possibly not well-formed UTF-8 on its own, matching `text_content.go`'s existing posture. A `[]byte` field would need copy-in and copy-out to preserve payload immutability; `design.md` owns the final spelling.
4. Report every rule violation through AI-04's taxonomy — no new sentinel, no second failure type.
5. Strict TDD per doc 0002's behavior-leaf grammar: every test-list item red → green → refactored, in order, with evidence recorded in `tasks.md`.

## Out of scope

| Excluded | Owner | Why not here |
| --- | --- | --- |
| The event envelope, kind registry, per-stream sequence | **AI-14** | Prerequisite, not deliverable |
| Response start and completion events | **AI-15** | Prerequisite |
| Reasoning block events; tool-call block events | **AI-17**, **AI-18** | Parallel siblings; they inherit decision 4 and re-decide nothing |
| A public accumulator, transcript rebuilder, or `Text()` over a recorded stream | **Layer 2** | Doc 0001 § 4.3 invariant 1 |
| Runtime enforcement of ordering, gap detection, conformance | **AI-22.3**, **AI-23** | AI-14.4 states the invariants; enforcement is packaged later |
| Terminal error events and partial-output posture | **AI-19** | Wave 2, after this |
| Mapping any vendor's `content_block_delta` frames | **AI-28.1.1** | This is the target that mapping aims at |
| Block-level metadata beyond the index (annotations, citations, signatures) | unclaimed | Charter asks for an index and a fragment; nothing more is invented |

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-text-events/` | New folder, five markdown files | None |
| `openspec/specs/ai-text-events/spec.md` | New capability (promoted at archive) | Low |
| `backend/agent/src/ai/` text-event source + tests | **New** — filenames are a `design.md` call once AI-14's layout is visible | Medium |
| AI-14's event-kind registry file and its guard test | **Modified** — three registrations | **Medium** — a file this change does not own |
| `backend/agent/go.mod`, `go.work` | **None** — zero requires preserved | — |
| doc 0002 | **None expected** | — |

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| Planning against AI-14/AI-15 surfaces that have not landed | **Certain** | High | Mandatory reconciliation gate before apply (see Dependencies). Every assumption recorded with file + symbol so the gate is execution, not re-analysis — Wave 1 rework lesson 1 |
| AI-17/AI-18 pick a different block-index scope | Medium | **High** — three green PRs, one broken consumer | Decision 4 is stated here as binding on all three, and the spec phase must cite it in `R-ATE` text so a sibling contradicting it fails review, not integration |
| `textPayload.validate` reused on a delta "because it is text" | Medium | High — breaks AI-16.3 and whitespace tokens | Decision 5 stated in the proposal, restated in the spec, and pinned by a whitespace-only-delta test |
| Contract says "text fragment" but behaves as a string fragment | Medium | High | AI-16.2 item 2's rune-split test is what makes the byte claim true; the word **byte** is required in the requirement text |
| Unstamped block index ships as a legal 0 | Low | High | Decision 3 — 0 is the rejected sentinel, tested |
| PR exceeds the review budget | Low | Medium | Three leaves, one narrow family; `tasks.md` carries the forecast against the accepted 5000-line budget |

## Rollback plan

Near-additive. Rollback is `git revert` of the change's commit range: the new source and test files disappear, and the only surviving edits to revert are the three registry entries and the guard-test rows in AI-14's files, which return that table to the AI-15 kind set. No data, no migration, no config.

Post-merge reversal has one asymmetric cost: **decision 4** (shared block-index space) is consumed by AI-17 and AI-18. Once either has landed against it, reversing it means editing three families at once and must be done before the first provider adapter (AI-28) reads real frames.

## Dependencies

- **AI-14** — envelope, kind registry, per-stream sequence, ordering invariants. **AI-15** — response-start and completion kinds, the first registrants to imitate.
- **Hard gate**: `sdd-apply` for AI-16 MUST NOT start until AI-14 and AI-15 are landed in this worktree, and MUST be preceded by a reconciliation pass that re-checks every assumption in this proposal and in `design.md` against the actual landed symbols.
- **AI-04** validation taxonomy, **AI-06** `text_content.go` (read-only precedent). No new Go dependency, no ADR.

## Success criteria

- [ ] Every test-list item of AI-16.1, AI-16.2, AI-16.3 taken red → green → refactored, in order, with both outputs recorded.
- [ ] Two interleaved text blocks are attributed by index alone, with no reliance on arrival order.
- [ ] Concatenated deltas reconstruct the text byte-exactly, including a delta boundary inside a multi-byte rune.
- [ ] A whitespace-only delta and a zero-length delta are both legal; a start/end pair with zero deltas reconstructs to `""`.
- [ ] A block index of 0 is rejected at construction with an AI-04 sentinel.
- [ ] `make test` in `backend/agent/` green under `-race`, `make lint` clean, both import-boundary guards passing, `go.mod` still zero requires.
- [ ] AI-14's exhaustiveness guard covers the three new kinds and is shown to bite against a scratch unregistered kind.

## Proposal question round (unanswered — auto mode)

Recorded for user review; the proposal proceeds on the stated assumptions until corrected.

1. Is decision 4 (one block-index space shared by text, reasoning and tool-call blocks) acceptable as binding on AI-17 and AI-18, or should each family number independently and consumers key on `(family, index)`?
2. Is the `[provisional]` `MaxTextLen` cap on a single delta fragment wanted at all, or should a fragment carry no bound at Layer 1?
3. Should AI-16 ship a test-only accumulator in `agenttest` for AI-17/AI-18/AI-23 to reuse, or stay strictly test-local as decision 7 states?
4. Confirm the sequencing expectation: is AI-16 planned now and applied only after AI-14 and AI-15 land, or should apply wait for a combined Wave 2 slice?

> **Deviation note**: this artifact exceeds the sdd-propose 450-word budget. The project's landed proposal precedent (`openspec/changes/archive/2026-08-01-cachicamas-ai-completion-metadata/proposal.md`) and `openspec/config.yaml` `rules.proposal` require an explicit rollback plan and an explicit deferred-but-related out-of-scope table; house convention wins.
