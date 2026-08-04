# Design: Tool-call conformance cases assert reconstruction, not delta shape

## Technical Approach

Replace the exact `requireDrainedKinds` census in `toolCallZeroDeltaCase` and `mixedTextAndToolCallCase` with three independent assertions — `checkLifecyclePrefix` (identity equality), one new shared relative-order helper, and byte reconstruction through a single R-CNF-025 observable. Each amended case's post-drain assertion set is factored into a **pure `check*Window(events) error` function** (the `checkLifecyclePrefix` idiom), so the same assertions run over fake-shaped, adapter-shaped, start-less and byte-mutated fixtures — giving S-CTA-011's four-outcome guard without TB-capture machinery.

**Risk 1 verdict — script route WORKS, no fallback needed.** Verified against landed code: `Emit` panics only on zero-Kind events (`fake_script.go:31`); `NewToolCallEnd` stores zero-length arguments as supplied and validates only block index (`tool_call_event.go:291`); `Provider.Stream` validates scripts via `ai.CheckStream` (kind/block ordering — deltas between start and end are legal; the interleaved case already ships delta-carrying scripts) and `produce` replays verbatim after stamping. An adapter-shaped fixture is therefore authorable entirely in test code:

```go
Script{Steps: []Step{Emit(responseStart), Emit(callStart),
    Emit(delta1 /* {"q": */), Emit(delta2 /* "weather"} */),
    Emit(endZeroLen /* NewToolCallEnd(1, nil) */), Emit(completion)}}
```

Zero `fake_*.go` / `stream_kit_*.go` / `src/ai` bytes (NFR-CNF-D).

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Matcher semantics | Anchored walk with windowed delta tolerance (rule below) | Naive subsequence scan | Subsequence skips *any* unexpected kind, so S-CTA-006's text-delta-in-window negative would vacuously pass; anchoring restores bite |
| Matcher home/shape | Pure `checkRelativeKindOrder(events, want) error` + `requireRelativeKindOrder(tb, ...)` wrapper in `conformance_lifecycle.go` | Per-case loops | R-CNF-019 mandates one shared helper; pure core enables permanent synthetic negatives (S-CTA-009) |
| R-CNF-025 observable | Method `(*reconstructedCall).reconstructedArguments() []byte`: end bytes when `len(arguments) > 0`, else `fromDeltas` (arrival-order concat); never summed | Changing `reconstructToolCalls`' return type | Method keeps the out-of-scope interleaved case (which asserts both channels deliberately, for attribution) byte-untouched; one stated rule, one place |
| Case assertions | Pure `checkZeroDeltaCaseWindow` / `checkMixedCaseWindow` in `conformance_tool_call.go`; cases call them after `RequireValidStream` | Inline assertions | Reused verbatim by adapter-shaped, start-less and mutated fixtures — "same case assertions" is literal, not paraphrased |
| S-CTA-017 surface | The observable is the method; `arguments`/`fromDeltas` fields remain internal accumulation state | Field hiding | Go has no sub-package privacy — an in-package author can always count deltas from raw events; R-CNF-025's own text scopes unobservability "through this helper" |

**The matcher rule (pinned):** cursor over `want`; each drained event must either (a) equal `want[cursor]` → consume, or (b) be `EventKindToolCallDelta` while inside an open expected tool-call window (a consumed `ToolCallStart` whose expected `ToolCallEnd` is unconsumed) → skip, count unconstrained; any other kind FAILS naming index, got-kind and expected kind. Events exhausted with `cursor < len(want)` FAILS naming the first missing kind. Delta block-attribution is not the matcher's job: `RequireValidStream` rejects deltas for never-started blocks, and reconstruction proves bytes.

## Data Flow

    Script (case- or test-authored) → Provider.Stream (stamp + ai.CheckStream) → drained window
        → checkLifecyclePrefix (position 0 + ResponseID/ServedModel equality)
        → checkRelativeKindOrder (anchored walk)
        → reconstructToolCalls → reconstructedArguments() → bytes.Equal

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agenttest/conformance_tool_call.go` | Modify | Two case bodies → pure window checks; add `reconstructedArguments()`; mixed keeps text-survival + last-event `FinishReasonToolCalls` equality; zero-delta adds no finish-reason equality (D6) |
| `backend/agent/src/agenttest/conformance_lifecycle.go` | Modify | Add `checkRelativeKindOrder` + `requireRelativeKindOrder` |
| `backend/agent/src/agenttest/conformance_lifecycle_test.go` | Modify | Matcher permanent negatives: text-delta-in-window (S-CTA-006), dropped end (S-CTA-009), unexpected-kind, missing-kind |
| `backend/agent/src/agenttest/conformance_tool_call_test.go` | Create | Precedence units (S-CTA-014/015/016/017); adapter-shaped fixtures (S-CTA-001/002/007); byte-mutation (S-CTA-004); start-less double-run four-outcome guard (S-CTA-003/008/011) |

## Interfaces / Contracts

```go
func checkRelativeKindOrder(events []ai.Event, want []ai.EventKind) error   // pure, R-CNF-019
func requireRelativeKindOrder(tb testing.TB, events []ai.Event, want []ai.EventKind)
func (c *reconstructedCall) reconstructedArguments() []byte                 // R-CNF-025 precedence
func checkZeroDeltaCaseWindow(events []ai.Event) error                      // R-CNF-007
func checkMixedCaseWindow(events []ai.Event) error                          // R-CNF-008
```

Completeness stays `sawStart && sawEnd` — a zero-length end is never a defect (S-CTA-015).

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (pure) | Matcher rule + precedence | Synthetic `[]ai.Event` slices against pure checks; permanent guards, never staged mutations (R-CNF-020) |
| Case-level | Both fixture shapes pass identically | Adapter-shaped script through fake replay, asserted by the same `check*Window` |
| Suite | `RunConformance(t, FakeFactory())` all-green; `go test ./... -count=1` | Regression |

**TDD slicing (one PR):** compile-RED for new helpers/tests; **inverted-TDD RED** for precedence — write the adapter-shaped S-CTA-002/007 tests first; they fail against the landed end-bytes-only read (`call.arguments` empty) until `reconstructedArguments` lands. Permanent negatives are REDs by construction. Forecast ~300–350 authored lines — under the 400 budget, no chaining.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. **Archive order (R-CNF-026, carries into tasks):** `cachicamas-ai-conformance-lifecycle-amendment` archives FIRST, this change second; archiving against shipped main or out of order is a defect.

## Open Questions

- [ ] None blocking. Verify note: S-CTA-017 read as "through the helper's observable" (per R-CNF-025's own wording); struct fields remain in-package accumulation state used by the out-of-scope interleaved case.
