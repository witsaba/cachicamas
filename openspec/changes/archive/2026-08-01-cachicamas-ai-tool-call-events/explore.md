# Exploration: AI-18 — Add tool-call delta events (`cachicamas-ai-tool-call-events`)

> Materialized from Engram `sdd/cachicamas-ai-tool-call-events/explore` (obs #2216).
> The explore executor had no Write tool, so this file is written by `sdd-propose`.
> Milestone source: [`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` § AI-18](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-18--add-tool-call-delta-events).

## Current State

`backend/agent/src/ai` (Layer 1) is at Wave 1 completion only (AI-04…AI-13 shipped, PR #101 open, not merged). **Wave 2 has zero code on disk in this worktree**: no event/envelope/stream files exist anywhere under `src/ai`. AI-14 (event envelope), AI-15 (response lifecycle), AI-16 (text deltas), AI-17 (reasoning deltas) are all charter-only, same as AI-18. This exploration is therefore against the charter + the register + AI-09's shipped contract only.

**AI-09's shipped precedent** (`tool_call.go`, promoted spec [`openspec/specs/ai-tool-messages/spec.md`](../../../specs/ai-tool-messages/spec.md)) is the concrete type AI-18's events stream:

- `ToolCall{id, name, arguments string}` — opaque, comparable (`==`), fields stored as strings so the value stays comparable (a slice field would panic on `==`).
- `NewToolCall(id, name string, arguments []byte) (Part, error)` validates in order: id non-empty (`ErrEmpty`), name non-empty (`ErrEmpty`), arguments well-formed JSON via `isWellFormedJSON` (`ErrMalformed`, not `encoding/json.Valid` — I/O import ban).
- Nil/zero-length arguments canonicalize to `"{}"` (`emptyToolArguments`) — the one canonical empty form; supplied bytes are never touched (R-ATM-002 / R-ATM-003).
- `Arguments()` returns a byte-exact fresh copy every call; no re-marshalling anywhere on the path.
- **No ordinal field exists on `ToolCall`.** `ToolCalls(content []Part) []ToolCall` derives the ordinal purely from position among tool-call parts in a content sequence, skipping other kinds (R-ATM-006 / `V-STR-21`). The promoted spec is explicit: *"the ordinal MUST NOT be stored on the call at construction time, so that no stored value can disagree with the position a message actually holds"* and *"A later milestone that needs to store an ordinal amends this file rather than adding a parallel field."*
- `String()`/`GoString()` render only `"toolcall"` — id, name, arguments are all redacted (V-FAIL-13).

**Register constraints ([`openspec/specs/ai-contract-vocabulary/spec.md`](../../../specs/ai-contract-vocabulary/spec.md)) that bind AI-18 directly:**

- `V-STR-14` **block**: "a run of text, a reasoning passage, **a single tool call** — delimited by a start event and an end event, with zero or more deltas between them." Tool calls are explicitly one instance of the generic block family AI-14 defines.
- `V-STR-15` **block index**: the value separating interleaved blocks by index alone.
- `V-STR-16` **delta**: "an index and only the new fragment — never a snapshot." Deltas are "optional in the general case" — this is **G12(a)**, and AI-18.2 is its tool-call-specific instance/test.
- `V-STR-17` **ordering invariant** (AI-14.4): block start precedes its deltas precede its end; exactly one terminal event per stream.
- `V-STR-21` **call ordinal** — the single most load-bearing citation for this milestone: *"The observable position of a tool call among the calls of one response, preserved unchanged through normalization, streaming, and adapter translation… **Restated — never re-defined — at AI-18.3 (streaming) and AI-30.5 (adapter)**."* AI-18 does not get to invent a second ordinal concept; it must make the *same* R-ATM-006 concept observable from stream events.

**AI-02.1's stream-lifecycle spec** ([`openspec/specs/ai-stream-lifecycle/spec.md`](../../../specs/ai-stream-lifecycle/spec.md)) fixes five container questions AI-18 inherits for free and must not reopen: receive-only channel carrier, one producer goroutine / one closing site, bounded 64-capacity buffer, backpressure-never-drop except the one sanctioned loss path (cancel + saturated buffer drops late events, skips the terminal). AI-18 only defines what rides in the envelope for the call family — not container behavior.

**AI-06's registration pattern** (`content_part.go`) is the concrete precedent for how AI-14 will almost certainly shape the event envelope, and by extension how AI-18's payload types get registered: closed discriminator (`kind()`) derived from an unexported `partPayload`-shaped interface, five-step append-only registration (constant → payload type with `kind()`/`validate()` → constructor → `(T, bool)` accessor → name-table + doc entry), one mechanical AST-based guard (`content_part_registry_test.go`) proving the constant space, the enumeration and the table never disagree. AI-14.1's own test list ("kind derived from payload," "every registered kind has a constructible payload… asserted as a table over the registry") is this exact pattern restated for events.

## Affected Areas

- `backend/agent/src/ai/tool_call.go` — the type whose identity/name/argument-bytes shape AI-18's events transport; not modified by AI-18, but every design decision here is read against it.
- `backend/agent/src/ai/tool_call_test.go` — precedent for byte-exactness / no-aliasing test shape AI-18's own tests will mirror.
- `backend/agent/src/ai/content_part.go`, `content_part_registry_test.go` — precedent for the kind-derived-from-payload registration mechanics AI-14's envelope (and thus AI-18's payload types) will almost certainly reuse.
- [`openspec/specs/ai-tool-messages/spec.md`](../../../specs/ai-tool-messages/spec.md) — R-ATM-006's "ordinal is derived, never stored" rule is a hard constraint AI-18.3 must preserve verbatim, not reinterpret.
- [`openspec/specs/ai-contract-vocabulary/spec.md`](../../../specs/ai-contract-vocabulary/spec.md) — `V-STR-14`…`V-STR-21` are the exact vocabulary AI-18 must cite, not paraphrase; any new term needs a dated amendment blockquote per § 9 rule 2.
- [`openspec/specs/ai-stream-lifecycle/spec.md`](../../../specs/ai-stream-lifecycle/spec.md) — the five container decisions (§§ 3–7) AI-18 inherits and must not reopen.
- New (not yet created): `backend/agent/src/ai/tool_call_event.go` (or similar) — will not exist until AI-14's envelope skeleton lands; AI-18's own apply is blocked on AI-15's apply landing in this worktree.

## Approaches

1. **Three registered event-kinds (call-start / call-delta / call-end), sharing the generic block-index scheme AI-14 defines** — mirrors AI-16 (text) and AI-17 (reasoning) exactly; AI-17.1's charter text ("mirror the text family's shape and indexing") makes this the charter-implied default for AI-18 too.
   - Pros: uniform mental model across every streamed family; reuses `V-STR-15` block-index rather than inventing a second index scheme; keeps AI-18's contract minimal (payload shapes only, no new container mechanics); the ordinal (`V-STR-21`) stays derived — never stored — by filtering the reconstructed event/block sequence down to call-kind blocks and re-counting, the exact continuation of `ToolCalls()`'s "skip other kinds" derivation, now applied over a stream instead of a message.
   - Cons: none identified — this is not really a competing option, it's the charter's implied shape.
   - Effort: Medium — new payload types × 3, one registration pass, but the pattern is already proven three times over (text, reasoning, and AI-06's original two kinds).

2. **Two events (call-start-with-optional-deltas merged, call-end)** — collapsing delta into an accumulated field carried on start/end.
   - Pros: fewer types.
   - Cons: directly contradicts `V-STR-16` ("never a snapshot of accumulated content") and AI-14.4 invariant 2; deltas exist because they arrive as separate wire messages over time — merging them defeats streaming itself. Not a genuine contender.
   - Effort: N/A — rejected on contract grounds, not effort grounds.

3. **A single shared "block event" struct across text/reasoning/tool-call families (a "family" tag instead of a distinct registered kind per family).**
   - Pros: less code, one registration table entry.
   - Cons: violates `V-STR-11` ("kind derived from payload") and directly contradicts AI-17.1's explicit requirement that reasoning be "a structurally distinct family — not a flag on a text event." Tool-call events carry identity+name with no text/reasoning analog, so a shared struct needs optional/nullable fields per family — reintroducing exactly the "payload-less/mismatched envelope" hazard AI-14.1's acceptance criterion forbids ("a mismatched or payload-less envelope cannot be constructed"). Rejected.
   - Effort: N/A — rejected on contract grounds.

## Recommendation

Approach 1. It is the only one consistent with `V-STR-14`'s definition of "block" (which already lists a tool call as one instance of the generic family), with AI-17.1's explicit sibling-mirroring language, and with `V-STR-21`'s "restated — never re-defined" instruction: the ordinal must remain purely derived, so the call's per-stream correlation index should be the *same* shared block-index concept AI-14 defines for every family, not a second call-scoped counter — a second stored counter would itself be the "parallel field" R-ATM-006 forbids adding.

## Open Questions (raised for `sdd-propose` / `sdd-design`)

1. **Which index is "the call's index"?** AI-18.1 test 3 says argument deltas "carry fragments with the call's index." Is this the shared, stream-wide block index (`V-STR-15`, shared with text/reasoning blocks) or a call-scoped counter counting only tool-call blocks? Recommend the shared reading; the register does not spell this out to the byte, so it must be pinned explicitly. → **Resolved in `proposal.md` § Locked Decisions (D1).**
2. **Does the call-end event validate argument-byte well-formedness at construction**, mirroring `NewToolCall`'s `ErrMalformed` rule, or is that validation deferred entirely to AI-30's reassembly into an actual `ToolCall` part? AI-18.1 test 2 only asserts byte-equality, never well-formedness — suggesting AI-18's event layer stays a dumb transport carrier and AI-09's validation runs exactly once, at AI-30's reassembly (consistent with R-ATM-005's "one rule set, two entry points" — extending it to a third entry point at the event boundary looks out of scope for AI-18 specifically). → **Resolved in `proposal.md` § Locked Decisions (D2).**
3. **Does call-start validate non-empty identity/name at construction**, the same two rules `NewToolCall` enforces? AI-14.1 test 2 ("a payload-less event cannot be constructed; validation rejects it with an AI-04 sentinel") implies every event payload gets a `validate()` analogous to `partPayload.validate()` — so very likely yes, by direct structural analogy, but this should be made explicit rather than assumed. → **Resolved in `proposal.md` § Locked Decisions (D3).**
4. **W1 forward-note, not AI-18's problem directly**: the Wave-1 verify report records `isWellFormedJSON` has no nesting-depth cap (deferred to Wave 2, unreachable from untrusted input until AI-24). AI-18 itself never calls `isWellFormedJSON` — it doesn't construct a `ToolCall` — but AI-30's reassembly of streamed argument bytes will, so this remains live for a later milestone, not this one.
5. **Sequencing risk, not a design question**: AI-18 depends on AI-15 (response lifecycle) and AI-09 (shipped). Neither AI-14 nor AI-15 exist on disk in this worktree yet. `sdd-propose`/`sdd-design` can proceed against the charter now, but `sdd-apply` for AI-18 is hard-blocked until AI-15's apply lands in this branch.

## Risks

- Two Engram lookups requested by the explore task (`learning/sdd-rework-patterns`, `learning/sdd-spec-inconsistency-patterns`) could not be run — `mem_search`/`mem_get_observation` were not present in that executor's tool set. Any prior rework pattern specific to AI-09's own apply/verify cycle (which AI-18 extends conceptually) was not retrieved; a follow-up search before `sdd-design` freezes the shape is recommended.
- The openspec `explore.md` file could not be written by the explore executor (no Write tool available) — only the Engram half of hybrid persistence was completed. This file closes that gap.
- AI-18's apply is blocked on AI-15's apply landing in this worktree; if Wave 2's AI-16/AI-17/AI-18 parallel changes are proposed/designed before AI-14's envelope is actually pinned, all three risk re-deciding the same "which index" question independently and disagreeing.

## Ready for Proposal

Yes, with two caveats: (1) the openspec `exploration.md` file needed to be written to disk by an agent with file-write access — done here; (2) `sdd-design` should honor the locked decisions D1–D3 in `proposal.md` rather than let three parallel Wave-2 changes (AI-16/AI-17/AI-18) each answer them independently.
