# Archive Report: AG-11 — Complete turn termination and typed failure reporting

**Change**: `cachicamas-agent-turn-termination` · **Milestone**: AG-11 (Layer 2 Wave 2, milestone 11 of 24)  
**Branch**: `feat/agent-layer2-wave2-ag11` (base `main` `b8eb7d75`, HEAD `b5cfe1c3`, 10 commits)  
**Archive date**: 2026-08-16  
**Mode**: Strict TDD, hybrid (Engram + OpenSpec filesystem)

## Final State Summary

AG-11 is complete and closed. The loop now terminates every turn with a decision over the finish-reason vocabulary and reports provider failures as typed values to the consumer, enabling mid-stream recovery and observability without deciding retry policy (that is AG-15's scope).

**Diff**: 24 files changed, **2774 insertions(+), 33 deletions(−)**  
Code-only across 8 implementation files: **1134 lines**  
**Verify verdict**: **PASS WITH WARNINGS** (0 blockers, 0 critical, 13/13 requirements, 22/22 scenarios)  
Both verify warnings were fixed in commit `2709a663` (post-verify, pre-archive) — see Verification section below.

---

## What Shipped

### The exhaustive finish-reason dispatch (AG-11.1)

- **`TurnOutcome` extended to 8 members**: `TurnOutcomeFinished` (← `FinishReasonStop`), `TurnOutcomeToolCalls`, `TurnOutcomeLengthLimited`, `TurnOutcomeContentFiltered`, `TurnOutcomeRefused`, `TurnOutcomePaused`, `TurnOutcomeUnknown`, and the existing `TurnOutcomeAborted` (the mid-stream-fatal path — not a finish reason).
- **Exhaustive `outcomeForFinish(ai.FinishReason) TurnOutcome` dispatch** in `loop.go`, consuming every `ai.FinishReason` member via a single mapped switch, with a defensive `default` case returning `TurnOutcome(0)` (unreachable in production, rejected by `NewTurnEnd`).
- **`finalize()` wiring** to consume the dispatch instead of hardcoding `TurnOutcomeFinished`.
- **D2 normalization move**: the zero-finish fallback (`if turn.finish == 0 { turn.finish = ai.FinishReasonStop }`) is now applied before `finalize()` on the no-completion path, ensuring the dispatch always receives a valid member.
- **Agent-level exhaustiveness pin**: `TestTurn_ExhaustivenessPin` walks `ai.FinishReason(0..255)`, asserts every validating member appears in the dispatch's own vocabulary table and in behavioral runs through the loop, proving "an eighth `ai.FinishReason` added upstream MUST fail this suite until the loop handles it".
- **Refusal and pause divergence**: both produce distinct `TurnOutcome` members (`TurnOutcomeRefused` vs `TurnOutcomePaused`), distinguishable by outcome value alone without reading the underlying `ai.FinishReason` — closing R-ATT-004 and the doc 0003:2167 acceptance criterion.

### The typed failure fatal path (AG-11.2)

- **Mid-stream-fatal branch rewrite**: the loop now type-asserts `turn.fatal` to `*ai.Failure` (the provider-error arm), wraps it via `agent.NewFailure`, and emits `turn_end(TurnOutcomeAborted, failure)` + `run_end(RunOutcomeFailed, failure)` on `sink` before `closeSink`, then returns the reconstructed partial message instead of `ai.Message{}`. The internal-construction-error arm (`loop.go:454,467,477,494,507,524,573` — plain Go errors) remains byte-identical: drain, close, `return ai.Message{}, 0, turn.fatal` with no emission (D1's type fork, pinned by `TestTurn_InternalErrorArm_EmitsNothing`).
- **Partial message reconstruction**: `finalize()`'s reconstruction body extracted into `(*turnAccumulator).reconstructMessage()`, called from both `finalize()` (successful path) and the fatal path, preserving the same bracket rules (reasoning needs `started && ended`; text needs `started && fragments`). Partial content accumulated before the failure reaches the caller, closing R-ATT-007 and S-LSK-011.
- **`agent.Failure.PartialOutput() bool` accessor**: nil-safe, delegating to `(*ai.Failure).PartialOutput()` (`provider_failure.go:515-520`), mirroring the shape of `Category()`, `Delivery()` and `Retryable()`. Enables consumers to ask "did output precede this failure?" through typed values alone, closing R-ATT-006, S-AEV-074, and the partial-output discriminator requirement of invariant 4.
- **Exactly-one provider call**: the fatal path returns directly after emission with no second `provider.Stream` invocation, pinned by `TestTurn_ExactlyOneProviderCall` (both failure-after-content and failure-before-any-content subtests). Closes R-ATT-008 and the "loop reports retryability, never acts on it" discipline (R-06, R-15).

### Spec promotion and invariant closures

- **New capability spec**: `openspec/specs/agent-turn-termination/spec.md` created and promoted from the ADDED delta, containing 9 requirements (`R-ATT-001..009`) + 12 scenarios + 5 bites covering AG-11.1's finish-reason dispatch and AG-11.2's typed failure upward.
- **Three MODIFIED deltas** integrated into pre-existing canonical specs:
  - `agent-loop-skeleton`: `R-LSK-001` updated with AG-11's fatal-path return contract; `R-LSK-004` amended to release `turn_events.go` and `failure.go` for AG-11 only, with exact-filename scoping and cross-reference to `R-ATT-009`.
  - `agent-permission-protocol`: `R-APP-012` amended to release `failure.go` for AG-11's `PartialOutput()` addition only, widening both substrate guards' allowlists by exact filename suffixes.
  - `agent-event-envelope`: `R-AEV-008` amended to add the partial-output discriminator to the typed-failure surface and record the loop-level emission obligation jointly with AG-11.2.
- **Invariant 4 joint closure** (typed failures — AG-04.3 + AG-11.2): AG-04.3 owns the typed surface and its pins; AG-11.2 owns the discriminator (`PartialOutput()`) and the loop-level emission path (`turn_end(Aborted, failure)` + `run_end(Failed, failure)`). Both are shipped and integrated into their respective specs.

---

## Design Decisions (D1–D8, shipped as specified)

| # | Decision | Shipped form |
|---|---|---|
| **D1** | `t.fatal` type fork | Typed-failure emission (turning `turn_end`/`run_end`) ONLY when `turn.fatal` type-asserts to `*ai.Failure` (provider-error arm). Internal-construction-error arm (plain Go errors) unchanged byte-for-byte: drain, close, `return ai.Message{}, 0, turn.fatal` with no emission. Pinned by `TestTurn_InternalErrorArm_EmitsNothing`. |
| **D2** | `finalize()` zero-finish ordering | The normalization `if turn.finish == 0 { turn.finish = ai.FinishReasonStop }` moved ahead of the `finalize()` call at `loop.go:286`; post-hoc correction block deleted. Applies the fallback once (before dispatch) rather than twice. Both call sites handled: completion path already non-zero by validation; no-completion path normalized before dispatch. |
| **D3** | Substrate-filter widening | Exact-filename suffixes ONLY to BOTH `filterOutLoopFiles` and `filterOutLoopHookFiles`: `/turn_events.go`, `/failure.go`, `/invariant_pin_test.go`, `/turn_termination_test.go`, `/turn_failure_test.go`. Byte-in-sync between filters. No wildcard/prefix/directory widening. |
| **D4** | Dispatch mapping | `Stop→Finished`, `Length→LengthLimited`, `ToolCalls→ToolCalls`, `ContentFilter→ContentFiltered`, `Refusal→Refused`, `PauseTurn→Paused`, `Unknown→Unknown`. New members appended after `Aborted` (values 3–8); `turnOutcomeLimit` stays last. |
| **D5** | Run-level outcome under AG-11.1 | `finalize()` always emits `RunOutcomeCompleted`; only the fatal path emits `RunOutcomeFailed`. No code change needed — already true. |
| **D6** | Partial-message reconstruction | `finalize()`'s reconstruction body extracted into `reconstructMessage()`; both callers use it. Same bracket rules. One reconstruction, two paths — no semantic fork. |
| **D7** | Exactly-one provider call | Fatal path returns directly after emission with no second `provider.Stream`. Pinned by `TestTurn_ExactlyOneProviderCall`. |
| **D8** | Out-of-scope boundary | `loop.go:265`'s `_ = sched.Schedule(...)` byte-unchanged. No edits to 11 other forbidden files. No new `EventKind` (guard stays at 25 kinds). `Turn`'s signature unchanged. |

---

## The Substrate Deviation — Recorded, not silent

**This is the first milestone to modify a pre-existing substrate file** (`turn_events.go`, `failure.go`) rather than only appending new ones. The constraint is structural: `TurnOutcome`'s const block, bound-check and `String()` switch are local to `turn_events.go`; a member declared elsewhere would be rejected by `validate()` and render as `turnoutcome(N)`. Similarly, `agent.Failure`'s accessor set is local to `failure.go`. The edits are confined to R-ATT-001 (outcome members + `String()` forms) and R-ATT-006 (`PartialOutput()`); `TurnEnd.validate()`'s failure-iff-aborted rule and `NewTurnEnd`'s signature are unchanged.

**The release does NOT extend to later milestones without their own recorded delta.** Both substrate guards' allowlists (`filterOutLoopFiles`, `filterOutLoopHookFiles`) widened by exact-filename suffixes only (no wildcard, prefix, or directory relaxation), with byte-in-sync verification. This widening is authorized and scoped by `R-LSK-004` and `R-APP-012` deltas.

**Mitigation and audit trail**: the deviation is explicitly named in the proposal's risk section, the design's D3 rationale, the tasks' substrate-filter-closure section, both MODIFIED spec deltas, and this archive report. No milestone after AG-11 may assume this release; each must record its own delta if edits to these files are needed.

---

## Verification Results

**Final verdict**: PASS WITH WARNINGS (0 blockers, 0 critical, 13/13 requirements, 22/22 scenarios)  
**Evidence hash**: `sha256:57b2d6ebd0a947b5d2c6da9ca1f7c3b05670a281ce3f7cbee15128886fa5dd90`

### Requirement and scenario coverage

- **13 requirements**: `R-ATT-001..009` (ADDED) + `R-LSK-001`, `R-LSK-004`, `R-APP-012`, `R-AEV-008` (MODIFIED)
- **22 scenarios**: 12 spec scenarios (`S-ATT-001..012`) + 5 bites (`S-TTB-001..005`) + 5 cross-cut (`S-LSK-011/012`, `S-APP-014`, `S-AEV-074/075`)
- **Every scenario mapping**: verified by defeat tests; 14 defeat tests all produced required failures (Engram #3092).

### Gates (all re-run by verify, exit 0)

| Gate | Command | Result |
|---|---|---|
| Tests | `cd backend/agent && make test` | **0** — 12 `ok` packages, 0 `FAIL`, `-race` |
| Lint | `cd backend/agent && make lint` | **0** — 0 issues (fresh install, no cache artifact) |
| Build | `cd backend/agent && make build` | **0** — `go build -trimpath ./...` |
| Vuln | `cd backend/agent && make vuln-check` | **0** — 0 findings; go1.26.6 resolved the 5 AG-10 stdlib advisories |

### Warnings (both FIXED after verify)

1. **openspec/specs/agent-turn-termination/spec.md:3** — the canonical promotion carried a 4-level relative link to the milestone doc where a 3-level link is correct. **Fixed in commit `2709a663`**; orchestrator confirmed the target now resolves correctly.

2. **Four MODIFIED requirement headings** — `R-LSK-001`, `R-LSK-004` in `agent-loop-skeleton`; `R-APP-012` in `agent-permission-protocol`; `R-AEV-008` in `agent-event-envelope` kept the delta form `### Requirement: Title — R-ID` instead of the canonical form `### R-ID — Title`. **Fixed in commit `2709a663`**; all four transformed and orchestrator confirmed zero delta-style headings remain.

Both fixes were post-verify, pre-archive and are covered by the subsequent re-run of `make test`, `make lint`, `make build` gates (all passing).

### Defeat-test proof

All 14 defeat tests proved non-vacuous (Engram #3092 full details):

- `TestTurn_ExhaustivenessPin` × 3 (dispatch mapping, constant fallback, 8th finish reason)
- `TestTurn_FinishReasonDispatch_...` × 1 (duplicate-outcome detector)
- `TestTurn_RefusalPauseFinished` × 1 (pairwise-distinct outcomes)
- `TestTurn_FatalPath_EmitsTypedBrackets` × 2 (fatal branch emission, pointer identity)
- `TestTurn_PartialContentSurvives` × 1 (reconstructed message)
- `TestTurn_NoCompletionPath` × 1 (D2 normalization)
- `TestTurn_ExactlyOneProviderCall` × 1 (no second call)
- `TestTurn_InternalErrorArm_EmitsNothing` × 1 (unchanged internal-error arm)
- `TestFailure_PartialOutput_...` + `TestTurn_TypedFailureFullyInspectable` × 1 (partial-output discriminator)
- `TestTurnOutcome_DistinctMemberPerFinishReason` × 1 (vocabulary)

---

## Tasks and Completion

**Status**: 19 of 20 tasks complete (`[x]`), with 1 deliberately deferred.

| Phase | Status | Details |
|---|---|---|
| 1 (substrate vocabulary) | ✅ | Vocab extended, `PartialOutput()` added, filters widened. 7 items. |
| 2–4 (dispatch + divergence) | ✅ | `outcomeForFinish`, D2 normalization, refusal/pause pinned. 16 items. |
| 5 (fatal path + typed failure) | ✅ | Fatal-branch rewrite, partial-content reconstruction, identity pin, exactly-one call. 10 items. |
| 6 (cross-cuts) | ✅ | Signature unchanged, loop.go:265 byte-intact, merge-base diff verified, failure.go scoped. 5 items. |
| 7 (spec promotion) | ✅ | New spec created, 3 MODIFIED deltas applied to canonical specs. 5 items. |
| 8 (docs) | ✅ | NFR-TLS-003 pointer added, milestone doc checkbox flipped (line 2167). 2 items. |
| 8.3 (archive move) | ⏸️ | **Deliberately deferred** per apply-progress.md reasoning: AG-10's precedent shows verify-report staged at non-archived path across remediation rounds, archived only at merge (commit `c4830cd7`). Moving the folder mid-apply would strand `sdd-verify`'s expected artifact path. Orchestrator performs this move post-archive. |
| 9 (final gates) | ✅ | `make test`, `make lint`, `make build`, `make vuln-check` all green. 6 items. |

**Known-carried items** (milestone doc acceptance checkboxes at line 2162/2167/2168):
- Line 2162 (`[ ] AG-10.1/AG-20.2/AG-19.1, ...`) — AG-11 does not flip (shared with other milestones, not complete).
- Line 2167 (`[x] Refusal, pause, and unknown finish reasons produce three distinct behaviors — closed by AG-11.1`) — flipped by AG-11, sole owner.
- Line 2168 (`[ ] AG-10.1/AG-03.3, ...`) — AG-11 does not flip (shared, not complete).

---

## Code Impact

| File | Action | Summary |
|---|---|---|
| `backend/agent/src/agent/turn_events.go` | Modified (substrate) | 6 new `TurnOutcome` members + 6 `String()` cases. `validate()` and `NewTurnEnd` untouched. |
| `backend/agent/src/agent/failure.go` | Modified (substrate) | `PartialOutput() bool` accessor added; `NewFailure`, `Category`, `Delivery`, `Retryable`, `Unwrap` unchanged. |
| `backend/agent/src/agent/loop.go` | Modified | `outcomeForFinish` dispatch, fatal-branch rewrite, `finalize()` wiring, zero-finish normalization move, `reconstructMessage()` extraction. `loop.go:265` byte-unchanged. |
| `backend/agent/src/agent/loop_test.go` | Modified | `filterOutLoopFiles` widened by 5 exact filenames. |
| `backend/agent/src/agent/loop_hook_test.go` | Modified | `filterOutLoopHookFiles` widened, byte-in-sync with `loop_test.go`. |
| `backend/agent/src/agent/invariant_pin_test.go` | Modified | `TestFailure_PartialOutput_ReachableAsTypedValue` added, invariant-4 joint-closure pinned. |
| `backend/agent/src/agent/turn_termination_test.go` | New | `package agent_test`: exhaustiveness pin, 7 dispatch scenarios, vocabulary pin. |
| `backend/agent/src/agent/turn_failure_test.go` | New | `package agent_test`: fatal-path emission, partial-content survival, identity pin, internal-error-arm pin, exactly-one-call pin. |
| Forbidden files (25 others) | Untouched | Event descriptors, registry, guards, boundaries, `go.mod`/`go.sum`, `Makefile`, lint config all byte-unchanged. Scope-fence still 25 kinds. |

**Total diff**: 24 files, **2774 insertions, 33 deletions**. Code-only: **1134 lines** across 8 files.

---

## Unblocks

- **AG-13** — Harness-level iteration and resumption. AG-11 makes pause visible as its own outcome; AG-13 wires the upward-path wake to support resumption cycles.
- **AG-15** — Retry loop and decision-making. AG-11 reports `Retryable()` on every failure; AG-15 builds the retry gate and implements the backoff policy.

---

## Engram Artifact References (for traceability)

| Artifact | Type | Engram ID | Observation |
|---|---|---|---|
| Proposal | architecture | #3064 | sdd/cachicamas-agent-turn-termination/proposal |
| Design | architecture | #3066 | sdd/cachicamas-agent-turn-termination/design |
| Spec | architecture | #3067 | sdd/cachicamas-agent-turn-termination/spec |
| Tasks | architecture | #3072 | sdd/cachicamas-agent-turn-termination/tasks |
| Apply-progress | architecture | #3083 | (no memory record; file-only) |
| Verify-report | architecture | #3092 | sdd/cachicamas-agent-turn-termination/verify-report |
| **Archive-report** | **architecture** | **(persisted now)** | sdd/cachicamas-agent-turn-termination/archive-report |

---

## Closure

AG-11 is archived. All 13 requirements implemented, all 22 scenarios verified, all gates passing, design decisions shipped, substrate deviation recorded and scoped, and both invariant closures (turn-termination dispatch + typed-failure surface) integrated into their owning specs. The loop now terminates every turn with a decision and reports provider failures to the consumer, unlocking AG-13 (resumption) and AG-15 (retry). The milestone's one deferred task (8.3 archive-folder move) is performed by the orchestrator post-archive.

---

**Archive date**: 2026-08-16  
**Archived by**: sdd-archive executor  
**Final state verified**: YES (pass_with_warnings, 0 blockers)
