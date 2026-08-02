# Exploration: AI-17 — reasoning delta events (streamed reasoning content + round-trip token)

> Persisted from Engram observation #2201 (`sdd/cachicamas-ai-reasoning-events/explore`).
> The explore phase had no file-write tool, so this file was written during `sdd-propose`.

## Current State

`backend/agent/src/ai` (Wave 1, shipped) implements the content-part strategy AI-17 must stream from:

- `content_part.go` (AI-06): `Part` is a sealed sum type; `PartKind` is a closed, append-only registry (`PartKindText`, `PartKindReasoning`, `PartKindToolCall`, `PartKindToolResult`), kind derived from payload never stored. Adding a kind is a fixed 5-step recipe (declare const, payload type w/ `kind()`+`validate()`, constructor, accessor, registry entries).
- `reasoning_content.go` (AI-07, shipped): `Reasoning` payload holds `text string`, `token string`, `hasToken bool`, `redacted bool` (all unexported). `ReasoningState` is a **derived, closed** vocabulary (`ReasoningStateText`, `ReasoningStateRedacted`, `ReasoningStateTokenOnly`) computed from the bytes — except redaction, which is the one bit AI-07's own docs say is *not derivable from bytes* and is set only by `NewRedactedReasoning`. `Token() ([]byte, bool)` is the load-bearing accessor: presence is stored separately from length so "no token" and "empty token" never collapse — this is the exact distinction AI-17.2 item 2 requires on the wire. The token is never parsed/normalized/re-encoded (V-REQ-11); `MaxReasoningTokenLen` (1<<20) is the only rule applied to it. `reasoning_content_test.go`'s `opaqueTokens()` enumerates the byte classes (every byte value, malformed UTF-8, lone surrogate, embedded NUL, bidi control chars, base64-looking, violation-message-looking, etc.) that AI-17.2 item 1 says the streamed token must survive byte-exact for.
- **No event/stream code exists yet anywhere in `src/ai`** — confirmed by a repo-wide grep for stream/event/delta symbols in the package: only Wave 1 content/request/validation files are present. AI-14 (event envelope), AI-15 (response lifecycle events) and the AI-16/AI-18 siblings are all still charter-only in this worktree, consistent with the task note that AI-15's apply lands after this exploration.
- `doc.go` states the package will eventually own "the streaming event contracts" but currently owns none.
- doc 0001 §4.3 (event envelope) states four invariants that bind every future family, most relevant here: (1) deltas carry an index, never a snapshot of accumulated content; (4) errors are typed values. AI-14.4 (charter) restates invariant 1 for Layer 1 and adds: a block's start precedes its deltas precede its end; exactly one terminal per stream; nothing follows a terminal.
- Milestone doc 0002 lines 855–1034 give the authoritative Wave 2 charter. AI-16 (text delta events, parallel sibling) is the family AI-17.1 item 1 says to mirror: block start/delta/end events carrying a block index, deltas carry a fragment never accumulated text, byte fidelity across a delta that splits a multi-byte rune, and zero-delta blocks are legal. AI-18 (tool-call events) is the third parallel sibling, using an "ordinal" concept distinct from "block index."

## Affected Areas (once AI-15 lands — apply-phase only)

- `backend/agent/src/ai/reasoning_content.go` — read-only reference; AI-17 must not redefine `ReasoningState`, `Token()`'s presence/absence contract, or the byte-class coverage. It is the source of truth AI-17 streams.
- `backend/agent/src/ai/reasoning_content_test.go` — `opaqueTokens()` is the fixture list AI-17.2's test should reuse (not reinvent) to assert wire byte-exactness.
- New file, by the AI-06/AI-07 one-file-per-kind convention: something like `backend/agent/src/ai/reasoning_event.go` (+ `_test.go`) — does not exist yet; depends on AI-14's event envelope and AI-15's event registry existing first.
- Whatever AI-14 defines as the event-kind registry (analogous to `PartKind` in `content_part.go`) — AI-17's new event kinds append to it.
- Whatever AI-16 defines as the text block start/delta/end payload shape — AI-17.1 item 1 requires structural mirroring of it (same field shape and indexing scheme), while AI-17.1 item 2 requires the reasoning payload types to be independently typed/registered so no consumer can render reasoning as text by omitting a kind check.

## Approaches

1. **Independent, structurally-parallel event family (recommended)** — Three new event-kind members (block start / delta / end) with their own unexported payload types, field-for-field mirroring whatever AI-16 lands for text (index field on start/delta/end; delta payload carries a byte fragment, never accumulated text), plus reasoning-specific additions: the end-event payload carries `Token() ([]byte, bool)` reusing AI-07's exact presence/absence contract, and a `State() ReasoningState` derived the same way `Reasoning.State()` is. Registered as distinct `EventKind` constants, never a boolean flag on the text event.
   - Pros: satisfies AI-17.1 item 2's explicit anti-goal directly (structurally distinct, no flag-on-text-event); reuses AI-07's token/state contracts verbatim rather than redefining them, avoiding the token-redefinition risk called out for this milestone; follows the shipped AI-06/AI-07 precedent (new kind = new file + registry append) so no new pattern is invented, consistent with `reasoning_content.go`'s own "nothing new, on purpose" framing and with doc 0001 §3.1's warning about the retired Layer 1's two-strategies-for-one-contract defect (which was specifically reasoning's failure mode).
   - Cons: some structural duplication of the block-index/delta-fragment shape between the text and reasoning event families instead of one shared generic type.
   - Effort: Medium (bounded by AI-14/AI-15/AI-16 landing first; the reasoning-specific logic itself is small, mirroring AI-07's own small surface).

2. **Shared generic "block event" with a family discriminator field** — One block-event type parameterized by a kind/family tag, with reasoning's token layered on as an optional field.
   - Pros: less code duplication across text/reasoning/tool-call families.
   - Cons: directly violates AI-17.1 item 2 ("not a flag on a text event") and reproduces the exact defect doc 0001 §3.1 records the retired Layer 1 committing for reasoning specifically (implementing the content interface directly while other kinds kept a wrapper, "had to be reconciled"). A generic payload with an optional token field is precisely the shape that lets a consumer render reasoning as assistant text by skipping one check. Rejected.
   - Effort: Low upfront, but pays for it later exactly like the retired design did.

## Open Questions for sdd-propose / sdd-spec

> All three are resolved as locked decisions D1/D2/D3 in `proposal.md`.

1. **Where does the redaction bit surface on the wire?** AI-07's own documentation states redaction is "the one bit that is not derivable from the bytes" — it is set only by `NewRedactedReasoning`, never inferred. A streamed reasoning block therefore needs an explicit redacted signal somewhere in its start/delta/end events (most likely the start event, so a consumer can react before the block closes) — this is not stated in the AI-17 charter text and must be resolved before AI-17.3 can be specified precisely.
2. **Does the round-trip token arrive whole (on the end event only) or can it be fragmented across deltas like text?** AI-07 treats the token as one atomic opaque string with no meaningful sub-fragment; fragmenting an opaque, non-text blob would require reassembly logic that duplicates what already exists for text deltas without any stated need. Recommendation: deliver it whole, on the block-end event, exactly as `Reasoning.Token()` already models presence/absence — but AI-17.2 item 1 only says "at the documented position," which is not yet documented and needs the spec phase to pin down explicitly.
3. **Do reasoning blocks share one index/sequence space with text blocks, or does each family own its own contiguous index?** AI-16.1 item 1 says "events for two blocks are attributable to their own block by index alone" and AI-17.1 item 1 says reasoning "mirrors the text family's shape and indexing." Literally read this could mean either an identical *scheme* (own contiguous space) or a *shared* space across kinds. AI-18's tool calls use a separate "ordinal" vocabulary, suggesting index and ordinal are deliberately different concepts, but doesn't resolve text-vs-reasoning sharing.

## Recommendation

Approach 1 (independent, structurally-parallel event family), deferred at the code level until AI-14/AI-15/AI-16 land, but the *contract* (event kind names, field shapes, token/state accessor reuse) can and should be pinned in sdd-propose/sdd-spec now, against the charter, so AI-17's apply phase has no open design questions once its dependencies ship. The three open questions above should be resolved as explicit spec decisions, not left implicit, since each one is exactly the kind of ambiguity doc 0001 §3.1 and this project's own rework-pattern history flag as costly to resolve after code exists.

## Risks

- AI-15's response-lifecycle event types are not on disk in this worktree; AI-17's actual Go code cannot be written (apply phase) until AI-15's apply lands — propose/spec/design can proceed against the charter contract only, as instructed.
- The redaction-signal placement (open question 1) is unresolved by the charter text and risks becoming a naive design that loses the redacted/token-only distinction on the wire, mirroring the exact class of defect doc 0001 §3.1 already recorded once for this same content kind.
- Without an explicit decision on token fragmentation (open question 2), an implementation could accidentally split the opaque token across deltas, which nothing in the byte-class fixture set (`opaqueTokens()`) is designed to reassemble, risking a corrupted round-trip on the second turn of a tool-using conversation — precisely the failure mode AI-07's docs call out as costly to notice late.
- The explore session's tool set did not expose `mem_search`/`mem_get_observation` (only `mem_save`), so prior Engram context for AI-07's shipped token/byte-class details and the project's `sdd-rework-patterns`/`sdd-spec-inconsistency-patterns` learnings could not be retrieved as instructed; this exploration was built directly from the current source and the milestone/architecture docs instead.

## Ready for Proposal

Yes — with the three open questions above surfaced explicitly to the user/orchestrator before or during sdd-propose, since each is a real design fork, not a detail sdd-propose can silently pick a side on.
