# Design: AG-10 — Implement the permission protocol

> **Change**: `cachicamas-agent-permission-protocol` · **AG-10** (Layer 2 Wave 2, 10/24; doc 0003 lines 1005–1111) · **Depends**: AG-06.1, AG-09

## Technical Approach

Per-call gate inside the scheduler (Approach 1 of exploration). Each call goroutine consults the injected `PermissionPolicy` before `ToolStart`; sync verdicts proceed, `Defer` parks the call on a per-call `chan struct{}` keyed by `callID` while siblings continue. The loop owns the upward-path wake (close by callID); the scheduler owns the parked set. `PermissionPolicy` is an interface (intentional mismatch with AG-08's `PreRequestHook` function value) because the policy carries internal state a function value cannot hold. Layer 2 owns the protocol; Layer 3 (doc 0004 CO-03) implements it. `Deny` surfaces as `Result{Outcome: ExecutionFailure, Failure: <typed denial>}` — not a Go error. Substrate preserved (NFR-TLS-003 7th).

## Architecture Decisions

| # | Decision | Choice | Rationale |
|---|---|---|---|
| **D1** | Port shape | `PermissionPolicy` interface (`Resolve`+`Remember`) | Policy has state a function value cannot carry. Mismatch with AG-08 intentional. |
| **D2** | Park locus | Inside scheduler | Keeps AG-09 invariants in one file; sidecar collapses per-call into per-turn. |
| **D3** | Result slot pre-fill | `SetCallID(call.ID())` before park (R-TLS-009) | Guarantees rejoin fully populated even on mid-park cancel. |
| **D4** | `decision_required` ordering | Emit BEFORE park wait | R-AGE-006: history already knows; emission reaches `sink` before parked goroutine blocks. |
| **D5** | ModifyInput transparency | Defer `ToolStart` until verdict; rewrite `arguments` | `ToolStart.Arguments()` byte-equals `decision_made.ModifiedArguments()` (AG-10.2; bite). |
| **D6** | Deny surface | `Result{Outcome: ExecutionFailure, Failure: typedDenial}` | Mirror orphan path; Go error would hide denial. |
| **D7** | Per-call abort | `context.Context` already threaded through `Schedule` | `_ context.Context` at `scheduler.go:89` is AG-09's wire-up point. |
| **D8** | Substrate preservation (NFR-TLS-003 7th) | ZERO edits to 10 files: `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go` | AG-07→08→09 precedent; substrate filter widens. |

## Data Flow

```
loop.go:240-244 ──→ Schedule(ctx, calls, reg, runID, turnID, policy, stamper, sink)
                          │
                          ▼
                      executeCall ──→ policy.Resolve ──→ Verdict
                          │
                          ├── AllowOnce/AllowAlways ──→ emit decision_made ──→ executeCall
                          │       └── AllowAlways → policy.Remember → true? emit resolution_remembered
                          ├── ModifyInput ──→ emit decision_made → substitute args → executeCall
                          ├── Deny        ──→ emit decision_made → results = ExecutionFailure
                          └── Defer       ──→ emit decision_required → park on parkedCh[callID]
                                                  └── select { parkedCh | ctx.Done() }
                                                          wake: re-evaluate verdict
                                                          cancel: results = ExecutionFailure{aborted}
                                                  ──→ results[0..n-1] populated
```

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/agent/permission_protocol.go` | Create | `PermissionPolicy` + `PermissionVerdict` + parked-set map + per-call gate + wake hand-off |
| `backend/agent/src/agent/permission_protocol_test.go` | Create | `package agent_test` — AG-10.1..AG-10.4 scenarios + bites |
| `backend/agent/src/agent/scheduler.go` | Modify | Add `policy` param to `Schedule`; thread parked set + per-call `chan`; rejoin unchanged |
| `backend/agent/src/agent/loop.go` | Modify | Lines 240-244: pass `policy`; upward-path wake (close parked channel by `callID`) |
| **10 substrate files** (D8 list) | **NOT TOUCHED** | NFR-TLS-003 7th carry |

## Interfaces / Contracts

```go
// PermissionPolicy — Layer 2 contract (Layer 3 implements, doc 0004 CO-03).
type PermissionPolicy interface {
    Resolve(ctx context.Context, call ai.ToolCall) PermissionVerdict
    Remember(ctx context.Context, toolName string, outcome PermissionOutcome) bool
}

// PermissionVerdict — NEW. Wraps the four outcomes + Defer.
type PermissionVerdict struct {
    Outcome      PermissionOutcome // AllowOnce|AllowAlways|Deny|ModifyInput|Defer
    ModifiedArgs []byte           // populated iff Outcome == ModifyInput
    Failure      *Failure         // populated iff Outcome == Deny
}
```

`PermissionOutcome` enum exists (`permission_events.go:53-80`); `PermissionVerdict` is new; `PermissionPolicy` is the only new interface. `Scheduler.Schedule` gains one argument: `policy PermissionPolicy` between `turnID` and `stamper`.

## Testing Strategy

| Layer | What to Test | Approach |
|---|---|---|
| Unit | AG-10.1 immediate allow; AG-10.1 `Defer`+`AllowOnce` (sibling isolation); AG-10.1 wake to unknown `callID` (typed rejection); AG-10.2 four-outcome matrix; AG-10.3 cancellation mid-park; AG-10.4 `Remember` branches; substrate filter update | `permission_protocol_test.go` (`package agent_test`); table-driven `t.Run`; bites RED-before-GREEN |
| Integration | 3 parked + 2 immediate calls through 4 outcomes (R-TLS-008 source-guard) | Single `Schedule`, drain `sink`, assert rejoin + stream order + `CardinalityAtMostOne` |
| E2E | Scripted fake provider drives a turn with one each: deferred, denied, modified | Assert full stream: `decision_required` → `decision_made` → `tool_start` (modified args) → `tool_end_*`; `-race` clean |

## Threat Matrix

`N/A — AG-10 adds no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. New surface is a Go interface + per-call coordination primitive inside the existing scheduler.`

## Migration / Rollout

No migration required. AG-10 is purely additive within `backend/agent/src/agent/`. `PermissionPolicy` injected at `loop.go:240-244`; nil-policy tests retained for AG-07 regression (suspend-the-protocol was the AG-09 default). A no-op pass-through decorator goes in `agenttest` for tests bypassing the gate.

## Open Questions

None. The proposal pre-authorized `size:exception`; AG-10 inherits the AG-09 wire-up order without further user input. AG-14's full cancellation tree is out of scope.
