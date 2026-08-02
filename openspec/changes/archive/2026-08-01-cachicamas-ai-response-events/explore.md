# Exploration — AI-15: response lifecycle events (`cachicamas-ai-response-events`)

> Persisted from Engram observation `#2161` (`sdd/cachicamas-ai-response-events/explore`).
> The explore phase ran without filesystem write access, so this file is a faithful
> transcription of that artifact rather than a rewrite.

## Current state

`backend/agent/src/ai` (Wave 0+1 shipped, 14/41 milestones) already contains the two contracts AI-15 must reuse verbatim:

- **`finish_reason.go` (AI-13.1/.2)** — `FinishReason` is a closed `uint8` vocabulary (iota-based, zero value invalid) with 7 members including `FinishReasonRefusal` and `FinishReasonPauseTurn` (three-way distinguishable from `FinishReasonUnknown`). `String()` / `Validate(at ...Step)` / `NormalizeFinishReason(string) FinishReason` are the public surface. `Validate` returns `*Violation` wrapped by `FirstFailure`, class `ErrNotInVocabulary`.
- **`usage.go` (AI-13.3/.4)** — `Usage` struct of five `TokenCount` fields (Input, Output, CacheRead, CacheWrite, Reasoning). `TokenCount{count int64, present bool}` makes "absent" the zero value, distinct from `Tokens(0)`. `Usage.Validate` only checks non-negativity per field (`ErrOutOfRange`), never requires presence — "requiring a populated count is requiring a fabricated one".

AI-15's charter requires both types to be **consumed as-is**, not redefined — no new finish-reason or usage vocabulary.

**AI-14 (event envelope) is NOT yet on disk** in this worktree (sibling concurrent pipeline). No `event*.go` files exist under `backend/agent/src/ai/`. AI-15 must be explored against AI-14's *charter text* (doc 0002 lines 855-909), not its implementation:

- Envelope has a `kind` **derived from payload**, never stored (same pattern as `PartKind` / `content_part.go`'s `partPayload` interface: unexported `kind()` + `validate()` methods, registration table `[]string` indexed by the enum, `partKindFirst`/`partKindEnd` bounds, `AI-06.4`-style five-step "add a kind" guard).
- Per-stream 1-based contiguous sequence, explicitly **not** a process-global atomic counter (AI-14.3 is a `[guard]` milestone banning exactly the pattern `message.go`'s `lastMessageID atomic.Uint64` uses — the counter must live on/with the stream, not the package).
- AI-14.4 ordering invariants: block start precedes deltas precede block end; **exactly one terminal event per stream; nothing follows a terminal**; deltas carry an index, never an accumulated snapshot (this literally cites v2 doc § 4.3 invariant 1, `docs/architecture/0001-cachicamas-agent-stack-v2.md` lines 479-481).

Established package idioms AI-15 should reuse rather than invent:

- Closed vocabulary → `iota` block + `[]string` name table + `Validate` / `FirstFailure` / `Invalid` / `Step` / `At` (`validation.go`, AI-04).
- Provider-supplied opaque identity → **plain `string`**, byte-exact, no parser, no minting — `ToolCall.ID() string` (`tool_call.go` line 158) and `Request.Model() string` (`request.go` line 351) are the precedent. This is explicitly **not** the `MessageID` pattern (`message.go`), which is a Layer-1-minted, unforgeable, atomic-counter identity for values *this package* allocates — response identity and model come from the provider, so they do not qualify for that treatment.

## Affected areas

- `backend/agent/src/ai/` — two new payload files are expected (naming TBD by sdd-spec/design), analogous in shape to `tool_call.go`:
  - a **response-start** payload: provider response identity (plain string) + model actually used (plain string), plugged into AI-14's kind registry.
  - a **completion** payload: wraps `FinishReason` + `Usage` unchanged, marked as AI-14.4's "terminal" event.
- Whatever file/type AI-14 lands as its ordering-invariant checker — AI-15.1 test 2 ("a second response-start violates the AI-14.4 invariants") and AI-15.2 test 2 ("completion is terminal… nothing may follow it") both require AI-14's checker to expose either a generic "at most one of kind X" / "terminal" concept, or AI-15 has to special-case its own two kinds inside that checker.
- `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` — AI-15 charter (lines 911-935), AI-14 charter (855-909), AI-13 charter (798-847) already read and are the authoritative sources; also blocks AI-16…AI-19 (content-family events), so response-start/completion's shape determines what the delta-event milestones anchor to.

## Approaches

1. **Two payload files mirroring `tool_call.go`'s shape** — `NewResponseStart(id, model string) (Event, error)` and `NewCompletion(reason FinishReason, usage Usage) (Event, error)`, each an unexported struct satisfying AI-14's payload interface (`kind()`, `validate()`, presumably `terminal()` if AI-14 exposes that), constructed via `FirstFailure`.
   - Pros: matches every existing construction idiom in the package (AI-05/AI-06/AI-09); reviewers already know the shape; response identity/model validation (non-empty, `ErrEmpty`) is a one-line rule each.
   - Cons: cannot be locked down until AI-14's actual `Event` / kind-registration / terminal signatures exist — real risk of rework once AI-14 merges (see Risks).
   - Effort: Low, once AI-14 lands.
2. **Single "lifecycle" payload type with a start/end discriminant field** instead of two separate kinds.
   - Pros: fewer files.
   - Cons: violates "kind derived from payload, never stored" (the same principle `content_part.go` states for `Part.Kind()`) and reopens a discriminant-vs-payload disagreement the package has already closed twice (AI-06, and implicitly AI-14.1 test 1). Also breaks AI-14.1 test 3's "every registered kind has a constructible payload" table check, since one type would answer for two kinds.
   - Effort: Low to build, High risk of rejection at spec/review.

## Recommendation

Approach 1. It is the only option consistent with every closed-vocabulary/payload precedent already shipped in this package (AI-04 validation taxonomy, AI-06 content-part kind derivation, AI-09 tool call). Concretely: two new unexported payload types, `Usage` / `FinishReason` embedded unchanged, identity/model as plain non-empty strings validated via `ErrEmpty`, both registered into whatever kind-registry mechanism AI-14 ships. Do not attempt to lock exact function signatures in the proposal/spec until AI-14's code (not just charter) is available — spec AI-15's *behavior* (what a test must prove), not Go signatures, per the task graph's own authoring constraint (doc 0002 line 13).

## Risks

- **Hard sequencing dependency on AI-14's actual code.** AI-15 charter depends on AI-14, and AI-14 is not implemented in this worktree yet (confirmed: no `.codegraph/` index and no `event*.go` / `envelope*.go` files exist). `sdd-apply` for AI-15 cannot start before AI-14 merges; this exploration is deliberately typed against AI-14's charter prose only.
- **Open coordination gap**: it is unclear from AI-14's charter alone whether AI-14.4's ordering-invariant checker already generalizes to "exactly one event of kind X per stream" (needed for AI-15.1 test 2) and to a generic "terminal" flag per payload (needed for AI-15.2 test 2), or whether AI-14 only checks block start/delta/end triples for content families. If AI-14 ships without those generic hooks, AI-15 will need either a design amendment to AI-14 or a bolt-on special case that breaks the registry-driven extensibility the package has maintained since AI-06.4. **Recommend flagging this explicitly in AI-15's sdd-propose/sdd-spec as a coordination note with the AI-14 pipeline**, ideally before AI-14's own spec freezes.
- **Tooling gap in this exploration session**: `mem_search` / `mem_get_observation` were not available (only `mem_save` was exposed), so AI-13's Engram-recorded design rationale (if any exists beyond source comments) could not be retrieved directly; source code (`finish_reason.go`, `usage.go`) was read directly instead, which is authoritative for types/signatures but may not surface narrative decisions recorded only in Engram from AI-13's own explore/propose phases.
- No `.codegraph/` index exists for this worktree; investigation used direct Read/Grep/Glob rather than `codegraph_explore`.

## Ready for proposal

Yes, with a caveat: sdd-propose / sdd-spec for `cachicamas-ai-response-events` can proceed now to define **behavior** (test lists, acceptance criteria, the coordination note above), matching AI-15's own charter which is already fully specified in the task graph. It must NOT attempt to pin exact Go signatures for the envelope/event type until AI-14 is code-complete, and `sdd-apply` must be gated strictly behind AI-14's merge, exactly as the orchestrator's briefing already assumes.
