# Proposal: AI-18 — Add tool-call delta events

> Change `cachicamas-ai-tool-call-events` · Wave 2 "Stream" · Depends on AI-09 (shipped), AI-15 · Parallel with AI-16, AI-17.
> Charter: [`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` § AI-18](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-18--add-tool-call-delta-events) · Exploration: [`exploration.md`](./exploration.md).
> Supersedes the retired pre-2026-07-30 proposal that used the old AI-15 numbering and doc `0001`.

## Intent

Layer 2 must correlate interleaved parallel tool calls, show a tool name before any argument byte arrives, and rejoin results positionally (v2 § 7 **G5**) — several providers reject positionally mismatched results. Today only the finished `ToolCall` part exists (AI-09); nothing streams it. AI-18 defines the streamed form so a call delivered whole is indistinguishable, after reconstruction, from a fragmented one (**G12(a)**).

## Scope

### In Scope

- Three registered event kinds — call-start, call-delta, call-end — as payload types on AI-14's envelope.
- Call-start carries identity + tool name; call-delta carries an index and only the new fragment; call-end carries the exact argument bytes byte-equal, never re-marshalled.
- Zero-delta calls are legal and complete; the contract text states no consumer may require a delta.
- Interleaved calls reconstruct independently; each call's ordinal is observable from its events.

### Out of Scope

- Envelope, sequencing, ordering invariants (AI-14); response lifecycle events (AI-15); text/reasoning families (AI-16/AI-17).
- Stream container behavior — carrier, ownership, buffering, cancellation (AI-02.1, not reopened).
- Reassembling events back into a `ToolCall` part and validating argument JSON — **AI-30**.
- Tool execution, name resolution, schema validation — never Layer 1's.

## Locked Decisions

| Id | Question | Decision |
| --- | --- | --- |
| **D1** | Is "the call's index" (AI-18.1 test 3) the shared block index or a call-scoped counter? | The **shared, stream-wide block index** (`V-STR-15`). A second call-scoped counter would itself be the parallel field R-ATM-006 forbids. Consistent with AI-16/AI-17 indexing. |
| **D2** | Does call-end validate argument well-formedness? | **No — deferred to AI-30.** The event layer is a transport carrier; AI-09's rules run once, at reassembly (R-ATM-005, "one rule set, two entry points" — not a third). AI-18.1 test 2 asserts byte-equality only. |
| **D3** | Does call-start validate non-empty identity/name? | **Yes**, at construction, by direct analogy to `partPayload.validate()` and AI-14.1 test 2, using AI-04 sentinels (`ErrEmpty`). |

The ordinal stays **derived, never stored** — obtained by filtering the reconstructed block sequence to call-kind blocks and re-counting, the stream-side continuation of `ToolCalls()`.

## Capabilities

### New Capabilities

- `ai-tool-call-events`: the streamed tool-call family — three kinds, their payload rules, delta optionality, interleaving and the derived stream-side ordinal.

### Modified Capabilities

- None. `ai-tool-messages` R-ATM-006 and `ai-contract-vocabulary` V-STR-21 are **restated, never re-defined** — neither file is amended, because no ordinal is stored.

## Approach

Follow AI-06's five-step append-only registration (constant → payload with `kind()`/`validate()` → constructor → `(T, bool)` accessor → name table + doc), once per kind, on AI-14's envelope. Mirror AI-09's byte-exactness discipline: fragments and final bytes stored as strings for comparability, returned as fresh copies, never re-marshalled, never aliased. Redact identity, name and bytes from `String()`/`GoString()` (V-FAIL-13). Strict TDD: RED tests from the AI-18.1/.2/.3 test lists first.

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/ai/tool_call_event.go` | New | Three payload types, constructors, accessors. |
| `backend/agent/src/ai/tool_call_event_test.go` (+ external `ai_test`) | New | Lifecycle, byte-equality, zero-delta, interleaving, ordinal. |
| AI-14 envelope registry + name table | Modified | Three append-only entries. |
| `backend/agent/src/ai/tool_call.go` | Unchanged | Read as the contract precedent only. |
| `openspec/specs/ai-tool-call-events/spec.md` | New | Promoted after verify. |

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| AI-14/AI-15 absent on disk; apply hard-blocked | High | Plan now against the charter; sequence apply after AI-15 lands in this branch. |
| AI-16/AI-17/AI-18 answer the index question differently | Medium | D1 pins the shared block index; orchestrator cross-checks the three sibling proposals before design. |
| A stored ordinal creeps in, reintroducing the R-ATM-006 hazard | Low | Spec forbids it; a test asserts no ordinal field on any call payload. |
| Deferring D2 hides malformed bytes until AI-30 | Low | Explicit in spec text; AI-30 owns the single validation entry point. |

## Rollback Plan

Self-contained additive change: delete `tool_call_event.go` + its tests, revert the three registry/name-table entries, and drop `openspec/specs/ai-tool-call-events/` (or the change folder if unpromoted). No existing type, spec or behavior is modified, so no consumer breaks and no migration is needed. `git revert` of the single PR is sufficient.

## Dependencies

- AI-09 — shipped (`ToolCall`, R-ATM-006, `ErrEmpty`/`ErrMalformed` sentinels).
- AI-15 — response lifecycle events; blocks apply, not planning.
- AI-14 — envelope, block index, ordering invariants; transitively via AI-15.

## Success Criteria

- [ ] Start event exposes identity and tool name before any argument byte.
- [ ] End event's argument bytes are byte-equal to what was streamed; no re-marshalling on the path.
- [ ] A zero-delta call and its fragmented equivalent reconstruct identically.
- [ ] Two interleaved calls reconstruct independently with no cross-contamination, under `-race`.
- [ ] Each call's ordinal is observable from its events and is stored nowhere.
- [ ] `make test` (`go test -race -v ./...`) green from `backend/agent/`.
