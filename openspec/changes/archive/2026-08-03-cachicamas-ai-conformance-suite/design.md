# Design: AI-23 — Provider conformance suite

## Technical Approach

Extend `backend/agent/src/agenttest` with `conformance_*.go` files (no sibling package). The suite is a library: exported entry point `RunConformance(t *testing.T, f Factory) CapabilityRecord` runs a table of cases, each building a fresh subject via the factory, scripting behavior in AI-21's `Script`/`Step`/`Emit`/`Hold`/`Gate` vocabulary, and asserting through AI-22's `DrainAndRecord`/`RequireSameEvents`/`RequireValidStream`/`RequireNoGoroutineLeak`. The record and verdict implement AI-03 §10–11 literally. `src/ai` is not modified (R3 resolved: hand-list + drift guard), so `ai-completion-metadata` gains no delta and Modified Capabilities stays None.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| D1 Placement | `agenttest`, `conformance_` prefix | Sibling `agentconformance` | Sibling-of-`src/ai` guard constraint stays untouched (doc.go); in-package access to `summarize`/diff internals for redaction inspection; already decided at boundary |
| D2 Factory shape | Struct with `New func(testing.TB, ...Script) ai.ModelProvider` + declared optional capabilities + sentinel | Bare function type | Decision 3 (R1) needs declaration fields; a bare func cannot carry them |
| D3 Finish reasons | Hand-list 7 values in `conformance_capabilities.go` behind a behavioral drift guard | Export enumerator from `ai` | Zero `src/ai` surface (decided); AI-22.2's outside-the-package exhaustiveness technique already proven |
| D4 Sentinel channel | Plant in `FailureReport.Cause` and `RequestID` (and a scripted-payload variant); **not** `RawLabel` | RawLabel as sentinel | `summaryTable` renders `RawLabel` head by design (bounded safe metadata, AI-19); planting there would fail a sanctioned rendering, not detect a leak |
| D5 Outcome zero value | Zero is not a member; `OutcomeNotExercised` is explicit member 4 | Zero = not exercised | Package-wide "zero names nothing" idiom (`FinishReason`, `FailureCategory`) |
| D6 CAP-O-02 mismatch | Declaration vs `ai.TokenCounter` type assertion cross-checked; divergence in either direction records `failed` | Declaration wins | §9 advertising-binds + AI-24.1 comparability: an unexpected `satisfied` is a finding, not a pass |

## Interfaces / Contracts

```go
type Capability uint8 // 8 members: CapStreamingText…CapTypedFailures (CAP-R-01..05),
                      // CapReasoningContent, CapTokenCounting, CapCacheBoundary (CAP-O-01..03)
func Capabilities() []Capability          // enumerator; record totality by construction
func (c Capability) Optional() bool       // §11 marking rule: optional iff CAP-O member

type Factory struct {
    New      func(tb testing.TB, scripts ...Script) ai.ModelProvider // fresh subject per case
    Reasoning, TokenCounting, CacheBoundary *bool // expected optional capabilities (R1 seam);
        // nil = undeclared (S-CNF-006: suite fails construction naming the undeclared
        // capability); non-nil false = declared not offered (records absent); non-nil
        // true = declared offered (cross-checked against discovery per D6 where askable)
    Sentinel string // redaction canary; "" selects the suite default
}
func FakeFactory() Factory // reference subject: New wraps NewProvider; Reasoning=ptr(true),
                            // TokenCounting=ptr(false), CacheBoundary=ptr(false) — all three
                            // declared, exercising both the satisfied and absent outcomes

type Outcome uint8  // OutcomeSatisfied, OutcomeAbsent, OutcomeFailed, OutcomeNotExercised
type Standing uint8 // StandingRequired, StandingOptional — from AI-03, never from the run
type CapabilityRecord struct{ /* subject id + [8]entry{Capability, Standing, Outcome} */ }
func (r CapabilityRecord) Verdict() Verdict // pass | fail | inconclusive
```

Runner: entries initialize to `OutcomeNotExercised`; each case names the capability it exercises (marking input). At construction the runner checks all three `*bool` fields are non-nil (S-CNF-006) — a nil field fails construction immediately, naming the undeclared capability, before any case runs. A non-nil `false` field records `absent` and reports via `t.Skipf` naming the capability — never silent (this is the *declared-not-offered* path, distinct from the construction-time undeclared failure above). A non-nil `true` field is cross-checked against askable discovery where one exists (D6); a match records `satisfied`, a mismatch records `failed`. Any surviving `not exercised` → inconclusive; any failed required → fail (AI-03 §10 verdict rule verbatim: outcome set `satisfied | absent | failed | not exercised`).

Redaction (AI-23.7): suite scripts a mid-stream failure carrying the sentinel in `Cause`/`RequestID`; assertion walks the recording rendering every event through in-package `summarize` plus `Error()` on error payloads, and replays a forced divergence against an unexported message-capturing `testing.TB` (promoting `fakeTB`, stream_kit_record_test.go:30) asserting the sentinel absent from all captured failure text — direct precedent: provider_failure_test.go:1172.

## File Changes

| File (backend/agent/src/agenttest/) | Action | Description |
|---|---|---|
| `conformance_suite.go` | Create | `Factory`, `Capability`, case table, runner, marking (AI-23.1) |
| `conformance_text.go` | Create | AI-23.2: ordering, contiguity, multi-byte-rune reconstruction, empty completion |
| `conformance_tool_call.go` | Create | AI-23.3: fragmented/zero-delta calls, ordinals, mixed finish reason |
| `conformance_terminal.go` | Create | AI-23.4: one-terminal, partial-output discriminator, 9-category loop over `FailureCategories()` |
| `conformance_cancellation.go` | Create | AI-23.5: Gate-held cancel, saturated drop, `RequireNoGoroutineLeak` (serial-only) |
| `conformance_redaction.go` | Create | AI-23.7: sentinel plumbing + capture TB |
| `conformance_capabilities.go` | Create | AI-23.8: CAP-O cases, required finish-reason/usage cases, 7-value hand list |
| `conformance_record.go` | Create | AI-23.6: record, outcomes, verdict |
| `conformance_suite_test.go` | Create | Fake as first passing subject; finish-reason drift guard (probe `FinishReason(n).String() != "invalid"` upward, count must equal hand list); record/verdict unit tests |

All files carry milestone-ID-headed doc comments; stdlib + `ai` only (dependency-free pin holds).

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit | Record/verdict rules, marking, drift guard, sentinel sanitizer interplay | Table tests in `conformance_suite_test.go`, RED-first (strict TDD, `make test` from `backend/agent/`) |
| Integration | Whole suite vs `FakeFactory()` | Expect pass with CAP-O-01 `satisfied`, CAP-O-02/03 `absent` — exercises both optional outcomes |
| E2E | N/A | Deterministic by construction; live subject is AI-38/AI-39 |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Additive package files; rollback is deletion (proposal's plan unchanged). R4: verify AI-21/AI-22 openspec artifact placement before archive.

## Open Questions

- None blocking. AI-24 committing to `Script` as its fixture language is the accepted reversible risk already recorded in the proposal.
