# Proposal: Tool-call conformance cases assert reconstruction, not delta shape

## Intent

Two conformance cases assert exact drained event-kind lists that **no conformant
OpenAI-compatible adapter can satisfy**. The wire's only argument channel is
fragment strings inside `tool_calls` delta elements (AI-30 citation C9.4 — the
`arguments` description carries zero fragmentation semantics and there is no end
signal), so any non-empty-argument call forces ≥1 `ToolCallDelta` that
`conformance_tool_call.go:164` and `:256` do not list. Deeper: an exact per-window
delta count contradicts the neutral contract itself — **R-ATC-010** forbids Layer 1
from exposing delta count or fragment boundaries, so a suite that counts deltas
asserts a property the contract declares unobservable. AI-30 slice 2 cannot turn on
`CapToolCalls` until this lands.

## Scope

### In Scope

- Convert `toolCallZeroDeltaCase` and `mixedTextAndToolCallCase` from exact
  `requireDrainedKinds` lists to reconstruction-based assertions.
- Keep the lifecycle bite: replace the count's implicit position-0 proof with
  `checkLifecyclePrefix` (position 0 + `ResponseID`/`ServedModel` equality) — a
  net **strengthening**, since neither case carries an identity assertion today.
- Keep terminal assertions (last event is the completion; mixed keeps its
  `FinishReasonToolCalls` equality; zero-delta adds none — D6 holds).
- Spec delta for `ai-provider-conformance-suite`.

### Out of Scope

- Any `src/ai`, `openaicompat`, `fake_*.go` or `stream_kit_*.go` change (NFR-CNF-D).
- The other 15 registered cases; AI-30's adapter itself.
- Relaxing counts wherever the script still makes them observable.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `ai-provider-conformance-suite`: R-CNF-007 (S-CLA-006) and R-CNF-008 (S-CLA-007)
  restate their windows as order-plus-reconstruction; R-CNF-019 gains its boundary
  (an exact count is derivable only where the neutral contract makes it observable —
  never across a fragmentable argument channel); R-CNF-020's non-vacuity obligation
  re-anchors on the identity check for these two; R-CNF-021's tolerance register
  gains both.

## Scope table — all 17 registered cases

| Case(s) | Change | Why |
|---|---|---|
| `tool_call/zero_delta_whole_call_accepted` | **Yes** | `:164` pins 4 kinds; a real adapter emits ≥1 delta for `{"q":"weather"}` |
| `tool_call/mixed_text_and_tool_ends_on_tool_call_finish_reason` | **Yes** | `:256` pins 7 kinds; same forced delta |
| `tool_call/fragmented_interleaved_reconstructs_exactly`, `tool_call/ordinal_distinguishes_same_tool_name` | No | already reconstruct by block id, count nothing (R-CNF-021) |
| `text/*` (2), `terminal/*` (3), `cancellation/*` (2), `reasoning`, `cache_boundary`, `finish_reason`, `usage`, `token_counting` (5), `redaction` (1) | No | 13 cases: no tool-call block in any script, so no fragmentable channel — their counts stay script-derivable |

## Approach

Each amended case keeps its script and asserts three independent things instead of
one list:

1. **Lifecycle** — `checkLifecyclePrefix(events, scriptedID, scriptedModel)`.
2. **Order, not census** — kinds appear in the scripted *relative* order
   (subsequence), with delta occurrences unconstrained. One new shared helper,
   mirroring `requireDrainedKinds`' one-derivation-shape discipline.
3. **Content by reconstruction** — `reconstructToolCalls` keyed by block id proves
   the call's id, name and argument **bytes**. Because `NewToolCallEnd` explicitly
   permits zero-length arguments, the assertion must accept either channel: end
   bytes when non-empty, else the concatenated deltas. Design pins the precedence.

This makes the zero-delta case prove what it always meant — a zero-delta *script*
reconstructs identically however the subject chose to deliver it — which is
R-ATC-010 stated as a test rather than a shape frozen from a fixture.

**Pattern named**: this is the *second* dialect-reality amendment (first: exact
counts excluding `ResponseStart`). The recurring defect class is **conformance
fixtures asserting shapes no real producer was ever run against** — an assertion is
suspect whenever it pins a quantity the neutral contract declares unobservable. Each
real-adapter milestone flushes more; a third instance should be recognised on sight.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/agenttest/conformance_tool_call.go` | Modified | two case bodies |
| `backend/agent/src/agenttest/conformance_lifecycle.go` | Modified | one added order helper |
| `openspec/changes/.../specs/ai-provider-conformance-suite/spec.md` | New | delta |

Blast radius is the same closed world as the first amendment: `RunConformance` and
the case functions have **no callers outside `agenttest`**; `conformance_suite_test.go`
invokes both cases directly but holds no copy of their assertions, and no guard test
references S-CLA-006/007.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Order-tolerant assertions lose bite — a subject emitting nothing could pass | Med | Reconstruction asserts exact bytes; `checkLifecyclePrefix` asserts identity; R-CNF-020's start-less-fixture proof must still fail both cases |
| Both argument channels populated → double-counted bytes | Med | Design pins channel precedence (end-if-non-empty), not naive concatenation |
| Spec delta targets requirements that live only in the *unarchived* first amendment | High | This change stacks on that branch; spec phase writes against its post-state, and archive order is first→second |
| Over-relaxation spreads to cases whose counts are still honest | Low | Scope table freezes the boundary at 2 cases |

## Rollback Plan

Single-commit revert of `conformance_tool_call.go` + `conformance_lifecycle.go` plus
the spec delta. No production code, no `src/ai`: reverting restores the exact lists
and leaves AI-30 slice 2 blocked as before.

## Dependencies

- Stacks on `cachicamas-ai-conformance-lifecycle-amendment` (branch tip).
- Blocks AI-30 slice 2's `CapToolCalls` conformance turn-on.

## Success Criteria

- [ ] RED first: each amended case fails against a start-less fixture (R-CNF-020
      non-vacuity) and against a fixture whose reconstructed bytes differ.
- [ ] A synthetic fragmented delivery of the *same* arguments passes both amended
      cases unchanged — the wire-reality proof the shipped lists fail.
- [ ] `go test ./... -count=1` green in `backend/agent`; zero `src/ai`,
      `openaicompat`, `fake_*.go`, `stream_kit_*.go` edits (NFR-CNF-D).
- [ ] `RunConformance(t, FakeFactory())` still passes with all eight capabilities.
- [ ] Authored diff well under 400 changed lines.
