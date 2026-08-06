# Proposal: Conformance suite accepts the mandated response lifecycle

## Intent

The shipped conformance cases assert **exact drained event counts computed from a
pre-producer Script fixture that never emitted `ai.EventKindResponseStart`**
(`conformance_text.go:53` wants 4, `:99` wants 1). Doc 0002's AI-28.1.1 mandates a
real producer emit response-start → completion carrying the vendor's response
identity and served model. The neutral checker (`ai/stream_check.go`) is
payload-independent and permits the lifecycle to grow; the freeze lives **only** in
these hand-written counts and index-0 reads. Any conformant real adapter therefore
drains one extra event and fails the suite as shipped — AI-28's spec (R-ATS-011,
S-ATS-043) forbids editing, forking or skipping the case from inside AI-28, so this
amendment is the sanctioned route. It must land before AI-28 slice 2.

## Scope

### In Scope

- **Tier A — strengthen** the two AI-23.2 text cases: script gains a leading
  `ResponseStart`; counts 4→5 and 1→2; **plus new assertions** that the start is at
  position 0 and carries the scripted response identity and served model. Counts stay
  exact — exact counts caught this defect class.
- **Tier B — unfreeze** every other case whose drained window is a whole response and
  whose assertion is an exact count or an index-0/first-event read: fixture gains the
  same leading `ResponseStart`, count/index updated, **no new assertions** (identity
  belongs to Tier A only).
- Spec deltas for `ai-provider-conformance-suite` covering R-CNF-005/006 and the
  affected leaves.

### Out of Scope

- Any `openaicompat` change; any `src/ai` contract change (NFR-CNF-B holds).
- AI-28's bridge and text mapping (lands separately, depends on this).
- Rewriting exact counts into subsequence/presence assertions.

## Capabilities

### New Capabilities
None.

### Modified Capabilities
- `ai-provider-conformance-suite`: R-CNF-005 gains the response-start position and
  identity/model obligation (new scenario); R-CNF-005/006 scenarios restate the
  drained shape as `ResponseStart` + content + terminal.

## Scope table

| File / case | Change | Why |
|---|---|---|
| `conformance_text.go` `text/order_contiguity_byte_exact_reconstruction` | **A** | `len!=4`; kind list and delta indices are positional |
| `conformance_text.go` `text/empty_completion_is_legal` | **A** | `len!=1`; `events[0]` must be the completion |
| `conformance_tool_call.go` `tool_call/zero_delta_whole_call_accepted` | B | `len!=2` |
| `conformance_tool_call.go` `tool_call/mixed_text_and_tool_ends_on_tool_call_finish_reason` | B | `len!=6` |
| `conformance_terminal.go` `terminal/exactly_one_terminal_every_path` | B | `rec.Len()!=1` in both stream-bearing subtests |
| `conformance_cancellation.go` `cancellation/bounded_close_leak_free` | B | asserts the **first** received event is `TextBlockStart` |
| `conformance_capabilities.go` `reasoning/whole_blocks_never_leak_into_text` | B | redacted subtest: `len!=2` + `events[0]` |
| `conformance_capabilities.go` `cache_boundary/honoring_is_consumer_visible` | B | `len!=1` + `events[0]` |
| `conformance_capabilities.go` `finish_reason/all_seven_values_reachable_drift_guarded` | B | `len!=1` + `events[0]`, ×7 subtests |
| `conformance_capabilities.go` `usage/absent_vs_zero_distinguishable` | B | `len!=1` + `events[0]` |
| `tool_call/fragmented_interleaved_reconstructs_exactly` | none | reconstructs by block id, counts nothing |
| `tool_call/ordinal_distinguishes_same_tool_name` | none | same |
| `terminal/partial_output_discriminator_both_states` | none | indexes `events[len-1]` |
| `terminal/all_nine_failure_categories_exhaustive` | none | indexes `events[len-1]` |
| `cancellation/abandoned_then_cancelled_drops_bare` | none | scans all events for an invented terminal |
| `redaction/sentinel_absent_from_every_rendering` | none | scans all events; one more is harmless |
| `token_counting/asked_of_the_provider_value` | none | never opens a stream |

## Approach

Every case authors its own `Script`; for a real adapter that script becomes the wire
transcript the bridge replays, so the fixture — not the checker — decides whether a
response began. Add `ai.NewResponseStart(id, model)` as the first step of every script
representing a started response, then correct the arithmetic. Tier A additionally
proves the event survives with its identity intact (position 0, `ResponseID()` and
`ServedModel()` equal to the scripted values), converting a count fix into a lifecycle
assertion the suite previously had nowhere.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/agenttest/conformance_text.go` | Modified | Tier A |
| `backend/agent/src/agenttest/conformance_{tool_call,terminal,cancellation,capabilities}.go` | Modified | Tier B |
| `backend/agent/src/agenttest/conformance_suite_test.go` | Modified | direct per-case callers (lines ~570, ~576, ~682–760, ~878, ~965, ~1110, ~1146) and the two `RunConformance(t, FakeFactory())` end-to-end tests (~124, ~1355) must stay green |
| `openspec/specs/ai-provider-conformance-suite/spec.md` | Modified | via delta |

Blast radius is closed: `RunConformance` and the case functions have **no callers
outside `agenttest`** (verified by repo-wide grep). All existing callers drive
Script-based fakes through `FakeFactory()`, so updating the suite's own fixtures
updates every caller at once.

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Over-reach: Tier B is 8 cases, not the 2 AI-28 needs | Med | Tier B is fixture+arithmetic only, ~6 lines/case; the alternative is one amendment per later AI-28 slice, each blocked by R-ATS-011 |
| Making `ResponseStart` mandatory conflicts with R-CNF-006's "MUST NOT require any minimum event count" | Med | R-CNF-006 forbids a minimum on **text content**; spec delta must say so explicitly |
| Asserting identity equality is too strong for adapters that normalise ids | Low | Assert non-empty + equal to scripted; revisit if a real adapter proves otherwise |
| Not every started response has a vendor response-start frame (error-first transcripts) | Med | spec phase decides per case which scripts represent "a response began" |

## Rollback Plan

Single-commit revert of the `agenttest` fixture edits plus the spec delta. No
production code, no `src/ai` change, no API surface: reverting restores the shipped
counts exactly and leaves AI-28 blocked as before.

## Dependencies

- `ai.NewResponseStart` / `Event.ResponseStart()` already shipped (`src/ai/response_start.go`).
- Blocks AI-28 slice 2 (text mapping / real-adapter bridge).

## Success Criteria

- [ ] RED first: each updated case fails against a fixture missing `ResponseStart`,
      with a message naming the absent lifecycle event — the new assertion's bite proof.
- [ ] `go test ./... -count=1` green in `backend/agent`; zero `src/ai` edits; go.mod
      still zero-dependency.
- [ ] `RunConformance(t, FakeFactory())` still returns a pass verdict with all eight
      capability entries.
- [ ] Total authored diff under 400 changed lines.
