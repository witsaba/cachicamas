# Design: AG-11 — Complete turn termination and typed failure reporting

> **Change**: `cachicamas-agent-turn-termination` · **AG-11** (Layer 2 Wave 2, 11/24; doc 0003 lines 1113–1176) · **Depends**: AG-07, AG-09, AG-04.2/AG-04.3

## Technical Approach

Approach 1 of the proposal (inherited, not re-opened): extend `TurnOutcome` in place to a 1:1 mirror of `ai.FinishReason`'s seven members plus the existing `TurnOutcomeAborted`, add a pure exhaustive `outcomeForFinish(ai.FinishReason) TurnOutcome` in `loop.go`, and make `finalize()` consume it instead of hardcoding `TurnOutcomeFinished` (`loop.go:613`). The mid-stream-fatal branch (`loop.go:270-276`) is rewritten to emit `turn_end(Aborted, failure)` + `run_end(Failed, failure)` on `sink` before `closeSink` and to return the reconstructed partial message. `agent.Failure` gains nil-safe `PartialOutput()`. Confirmed: `TurnEnd.validate()` (`turn_events.go:119-130`) checks only the `>= turnOutcomeLimit` bound (which moves with the const block) and value-equality to `TurnOutcomeAborted`; the six new members are appended AFTER `Aborted` and all sit on the Failure-forbidden side, so the validator needs **zero change**. `RunEnd`'s validator (`run_events.go:152-163`) likewise holds: `finalize()` keeps `RunOutcomeCompleted`; only the fatal path emits `RunOutcomeFailed`.

## Architecture Decisions

| # | Decision | Choice | Rationale |
|---|---|---|---|
| **D1** | `t.fatal` type fork | Typed-`Aborted` emission ONLY when `turn.fatal` type-asserts to `*ai.Failure` (the `ev.ErrorPayload()` arm, `loop.go:591-593`). The internal-construction-error arm (`loop.go:454,467,477,494,507,524,573` — plain Go errors) keeps today's behavior byte-for-byte: drain, `closeSink`, `return ai.Message{}, 0, turn.fatal`. | AG-11.2's Gherkin is scoped to "a provider stream scripted to end in a terminal error". Wrapping a plain error would require fabricating an `ai.Failure` with an invented `Category`/`Delivery`, laundering a Layer-2 construction bug into Layer 1's provider taxonomy (violates `failure.go`'s S-AGV-020 thin-wrap posture; `NewFailure` rejects nil and takes only `*ai.Failure`, `failure.go:33-38`). Not a regression: that arm is unchanged, and the change is purely additive on the provider arm. `turn_events.go:14-18` already anticipates a later milestone modeling unmodelled aborts. Pinned by a RED test that the internal-error arm emits no `turn_end`. |
| **D2** | `finalize()` zero-finish ordering | Move the normalization `if turn.finish == 0 { turn.finish = ai.FinishReasonStop }` AHEAD of the `finalize()` call at `loop.go:286`, delete the post-hoc block at `:288-290`. `outcomeForFinish` keeps a defensive `default` returning `TurnOutcome(0)` (unreachable in production; `NewTurnEnd` would reject it). | Verified both call sites: the completion path (`loop.go:257`) can never see zero — `ai.NewCompletion` validates its `FinishReason` at construction (`completion.go:63-68`). Only the no-completion path can. The `:279-285` comment's "absence is recorded, not corrected" is AI-13.1's zero-vs-`Unknown` distinction (absence must not masquerade as the recorded `FinishReasonUnknown` observation); the Stop fallback itself is already the loop's documented deliberate semantics at `:288-290`. Relocating it applies the same fallback once, earlier, making the emitted outcome (`Finished`) and the returned finish (`Stop`) agree — which is exactly today's observable behavior on that path. Mapping zero to an outcome inside the dispatch was rejected: it would re-create the absence-as-observation conflation at a second site. |
| **D3** | Substrate-filter widening | Add exactly five exact-filename suffixes to BOTH `filterOutLoopFiles` (`loop_test.go:849-863`) and `filterOutLoopHookFiles` (`loop_hook_test.go:921-935`), byte-in-sync: `/turn_events.go`, `/failure.go`, `/invariant_pin_test.go`, `/turn_termination_test.go`, `/turn_failure_test.go`. No wildcard/prefix/directory widening. | First widening for EXISTING modified substrate files (`turn_events.go`, `failure.go`, `invariant_pin_test.go`) rather than newly added ones — recorded here and in `apply-progress.md` as the proposal's deliberate deviation (R-ATT-009). `invariant_pin_test.go` IS added: the `PartialOutput` scenario lands there so its line-8 claim "invariant 4 closes jointly with AG-11.2" stays auditable in one file (R-ATT-006, proposal risk 2). |
| **D4** | Dispatch mapping | `Stop→Finished`, `Length→LengthLimited`, `ToolCalls→ToolCalls`, `ContentFilter→ContentFiltered`, `Refusal→Refused`, `PauseTurn→Paused`, `Unknown→Unknown`. New members appended after `Aborted` (values 3..8), `turnOutcomeLimit` stays last. | Appending preserves `TurnOutcomeFinished`=1/`TurnOutcomeAborted`=2 and the validator untouched. S-LSK-001's walking-skeleton pin holds: Stop still maps to `Finished`. |
| **D5** | Run-level outcome under AG-11.1 | `finalize()` always emits `RunOutcomeCompleted`; refusal/pause/unknown are turn-level distinctions only. | `RunOutcome` (`run_events.go:100-118`) has no per-finish vocabulary; resumption is AG-13, retry is AG-15. Only the fatal path emits `Failed`. |
| **D6** | Partial-message reconstruction | Extract `finalize()`'s reconstruction body (`loop.go:627-676`) into `reconstructMessage()`; both `finalize()` and the fatal path call it. Same bracket rules as today (reasoning needs `started && ended`, `:628`; text needs `started && fragments`, `:639`). | One reconstruction, two callers — no semantic fork. AG-11.2's scripted pattern (`agenttest/conformance_terminal.go:105-161`) ends its text bracket before the error, so partial content reaches the caller. An un-ended reasoning bracket still drops, matching `finalize()`'s existing rule — noted boundary, not widened here. Fatal path returns finish `0` (no completion observed; absence recorded). |
| **D7** | Exactly-one provider call | The fatal path returns directly after emission — no second `provider.Stream`. Pinned by `len(provider.Requests()) == 1` (`agenttest/fake_provider.go:157-161`). | R-ATT-008; R-06/R-15: the loop reports retryability, never acts on it. |
| **D8** | Out-of-scope boundary | `loop.go:265`'s `_ = sched.Schedule(...)` comes out byte-unchanged. Zero edits to `event_descriptor.go`, `event.go`, `stream_check.go`, `sequence.go`, `run_events.go`, `tool_event.go`, `event_registry_test.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`. No new `EventKind` (`failure.go:7`: failures ride the typed outcomes). `Turn`'s signature is unchanged, so no `doc.go` row moves (`doc_contract_guard_test.go` passes unmodified). | Proposal scope fence; AG-15 owns retry, AG-13 owns resume. |

## Data Flow

```
provider event ──→ translate() ──→ t.finish (member, via NewCompletion validation)
                        │                │
                        │          normalize 0→Stop (no-completion path only, BEFORE finalize)
                        │                ▼
                        │          finalize() ──→ outcomeForFinish(t.finish) ──→ turn_end(outcome, nil)
                        │                                                    ──→ run_end(completed, nil)
                        └──→ t.fatal ──┬── *ai.Failure ──→ NewFailure ──→ turn_end(aborted, f) ──→ run_end(failed, f)
                                       │                        └──→ reconstructMessage() ──→ caller (msg, 0, fatal)
                                       └── plain error ──→ (unchanged) ai.Message{}, 0, fatal — no emission
```

## File Changes

| File | Action | Description | Est. lines |
|---|---|---|---|
| `backend/agent/src/agent/turn_events.go` | Modify (substrate) | Six members appended after `Aborted` in `:74-89`; six `String()` cases in `:93-104`; doc note. `validate()`/`NewTurnEnd` untouched. | +55 |
| `backend/agent/src/agent/failure.go` | Modify (substrate) | `PartialOutput() bool` after `Retryable()` (`:64-69`), nil-safe, delegates to `provider_failure.go:515`. | +12 |
| `backend/agent/src/agent/loop.go` | Modify | `outcomeForFinish`; fatal branch rewrite (D1); `finalize()` consumes dispatch; `reconstructMessage()` extraction (D6); normalization move (D2). `:265` byte-unchanged. | +110/−15 |
| `backend/agent/src/agent/loop_test.go` | Modify | Filter +5 exact suffixes (D3). | +7 |
| `backend/agent/src/agent/loop_hook_test.go` | Modify | Same 5 suffixes, byte-in-sync. | +7 |
| `backend/agent/src/agent/invariant_pin_test.go` | Modify | `TestFailure_PartialOutput_ReachableAsTypedValue` (+ nil-safe case) — invariant-4 joint closure. | +45 |
| `backend/agent/src/agent/turn_termination_test.go` | Create | `package agent_test`: agent-level exhaustiveness pin + 7 behavioral dispatch scenarios + `TurnOutcome` vocabulary pin. | +320 |
| `backend/agent/src/agent/turn_failure_test.go` | Create | `package agent_test`: AG-11.2 — aborted emission, partial msg, `run_end(failed)`, `Requests()==1`, internal-error-arm-unchanged pin. | +380 |

Total ≈ 940 changed lines — inside the pre-authorized `size:exception` forecast (900–1500).

## Interfaces / Contracts

```go
// loop.go — exhaustive dispatch. Total; the default is defensive and
// unreachable in production (both call sites hand it a member, D2).
func outcomeForFinish(finish ai.FinishReason) TurnOutcome {
	switch finish {
	case ai.FinishReasonStop:          return TurnOutcomeFinished
	case ai.FinishReasonLength:        return TurnOutcomeLengthLimited
	case ai.FinishReasonToolCalls:     return TurnOutcomeToolCalls
	case ai.FinishReasonContentFilter: return TurnOutcomeContentFiltered
	case ai.FinishReasonRefusal:       return TurnOutcomeRefused
	case ai.FinishReasonPauseTurn:     return TurnOutcomePaused
	case ai.FinishReasonUnknown:       return TurnOutcomeUnknown
	default:                           return 0 // no outcome; NewTurnEnd rejects
	}
}

// failure.go — mirrors Category()/Delivery()/Retryable() nil-safety.
func (f *Failure) PartialOutput() bool {
	if f == nil { return false }
	return f.wrapped.PartialOutput()
}
```

Fatal-branch shape (D1, D6, D7):

```go
if turn.fatal != nil {
	_ = drainProvider(pCh)
	if pf, ok := turn.fatal.(*ai.Failure); ok {
		if failure, ferr := NewFailure(pf); ferr == nil {
			if turnEnd, terr := NewTurnEnd(runID, turnID, TurnOutcomeAborted, failure); terr == nil {
				emitStamped(sink, stamper, turnEnd)
			}
			if runEnd, rerr := NewRunEnd(runID, RunOutcomeFailed, failure); rerr == nil {
				emitStamped(sink, stamper, runEnd)
			}
		}
		msg := turn.reconstructMessage()
		closeSink(sink)
		return msg, 0, turn.fatal
	}
	closeSink(sink) // internal-error arm: byte-equivalent to today
	return ai.Message{}, 0, turn.fatal
}
```

**Exhaustiveness pin mechanism** (`turn_termination_test.go`, external test package — `outcomeForFinish` is unexported, so the pin is membership-walk + behavioral, the `finish_reason_test.go:277-320` technique): (1) walk `ai.FinishReason(0..255)`, membership by `Validate()`; every member must appear in a hand-written `dispatchVocabulary map[ai.FinishReason]agent.TurnOutcome` and the counts must match — an eighth upstream member fails here without touching unexported bounds. (2) For each of the seven, script the fake provider to a Completion with that reason, drain `sink`, assert the final `turn_end.Outcome()` equals the table's entry — proving the loop's dispatch, not just the table. (3) Assert the seven mapped outcomes are pairwise distinct. (4) `TurnOutcome` vocabulary pin: walk `TurnOutcome(0..255)` using `NewTurnEnd` success as the membership oracle (`nil` failure for non-`Aborted`, a real `*Failure` for `Aborted`), assert exactly 8 members, none rendering `"unset"`/`"turnoutcome("`.

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | Exhaustiveness pin (4 assertions above); `TurnOutcome` `String()` per member; `PartialOutput()` true/false/nil (invariant_pin family); filter sync | Table-driven, RED-first |
| Integration | 7 scripted turns, one per finish reason → distinct `turn_end` outcome; refusal vs pause divergence (R-ATT-004) | `agenttest.NewProvider` + drain `sink` |
| E2E | `conformance_terminal.go:105-161` script against `agent.Turn`: partial content → `turn_end(aborted, f)` with `Category`/`Retryable`/`PartialOutput` inspectable → `run_end(failed, f)` → sink close; returned `msg` carries partial text; `len(provider.Requests())==1`; internal-error arm emits nothing (D1 pin); `-race` clean | `turn_failure_test.go` |

## Threat Matrix

`N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Pure in-package enum extension, dispatch, and event emission.`

## Migration / Rollout

No migration. Single revert restores the two-member `TurnOutcome`, the hardcoded `finalize()` pair, and the 13-filename filters (proposal rollback plan). Spec deltas required at archive: `R-LSK-004` (substrate list releases `turn_events.go`/`failure.go`), `R-AEV-008` (partial-output discriminator + emission obligation), `R-APP-012` (scoped to AG-10).

## Open Questions

None. D1/D2/D3 close the three forks the proposal delegated to design; cardinality was closed at proposal (1:1).
