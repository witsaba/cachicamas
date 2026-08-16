# Proposal: AG-10 — Implement the permission protocol

> **Change**: `cachicamas-agent-permission-protocol` · **Milestone**: AG-10 (Layer 2 Wave 2, milestone 10 of 24; doc 0003 lines 1005–1111)
> **Branch**: `feat/agent-layer2-wave2-ag10` based at `main` `6de08335`
> **Artifact store**: hybrid (Engram + filesystem)
> **Pre-authorized**: `size:exception` against 400-line PR review budget (forecast 2500–3500 lines; AG-09 precedent 4258 lines, Engram #3027)
> **TDD**: strict, RED-first (`apply.tdd: true`, `apply.test_command: "go test ./..."`)
> **Closes**: G1's protocol half (R-10); v2 § 6 seam 2.

## Intent

Layer 2 exposes AG-06.1's permission event family and AG-09's hand-rolled scheduler, but lacks the seam that ties them together: doc 0001 § 4.1's "if approval is not a suspension **in the loop**, every frontend reimplements it out of band". AG-10 wires that seam — per-call decision-required suspension with sibling isolation, four typed outcomes (`AllowOnce | AllowAlways | Deny | ModifyInput`), modify-input transparency on the stream — so a frontend observes ONE surface (`R-AGE-013..017`) and one `PermissionPolicy` port to consult. Out of scope: policy content (doc 0004 CO-03) and cross-session remembered-rules persistence (CO-16.1).

## Scope

### In
- AG-10.1 decision-required emission, parked-set suspension, four outcomes, sibling isolation
- AG-10.2 modify-input transparency (`ToolStart.Arguments()` byte-equals `decision_made.ModifiedArguments()`)
- AG-10.3 per-call abort during park (via `context.Context`, NOT AG-14's tree)
- AG-10.4 `permission_resolution_remembered` emitted only when `Policy.Remember` returns true (preserves `CardinalityAtMostOne` at `event.go:322`)
- `PermissionPolicy` port: `Resolve(ctx, call) Verdict` + `Remember(ctx, toolName, outcome) bool` — Layer 2 contract, Layer 3 implementation (`doc 0001:603`)
- Per-call gate inside scheduler; parked-set keyed by `callID`; one `chan struct{}` per parked call
- Loop wire-up at `loop.go:240-244`

### Out
- Policy content / rule sets / mode flag (doc 0004 CO-03)
- Cross-session remembered-rules persistence (doc 0004 CO-16.1)
- Subagent tool scope (AG-19.3)
- AG-14's full cancellation tree
- New `PermissionOutcome` member or new permission event kind

## Capabilities

### New
- **`agent-permission-protocol`** — the ask-suspend-resume seam. Becomes `openspec/specs/agent-permission-protocol/spec.md` at archive.

### Modified
- **None.** AG-10 consumes AG-09's scheduler (`R-TLS-*` byte-clean) and AG-06.1's permission events (`R-APE-001..003` byte-clean). No existing capability requirement changes.

## Approach (Approach 1, recommended)

**Per-call gate inside scheduler; parked-set keyed by `callID`.** New file `backend/agent/src/agent/permission_protocol.go` (port + `PermissionVerdict` struct + per-call wake glue); new test file `permission_protocol_test.go` (`package agent_test`). `scheduler.go` modified to (a) accept an injected `PermissionPolicy` parameter on `Schedule`, (b) own the parked-set (the explore's `map[string]chan struct{}` keyed by `callID`), and (c) preserve the indexed `[]Result` rejoin + single-writer `LaneStamper` invariants byte-for-byte. `loop.go:240-244` modified to pass the policy into `Schedule` and route the upward-path wake through a per-call close. Loop stays stateless (R-LSK-002); parked-set lifetime is one `Schedule` call. AG-09's signature widens by exactly one parameter (`policy PermissionPolicy`); AG-09's body logic stays in one file.

**Not Approach 2 (sidecar registry):** collapses per-call into per-turn suspension unless the sidecar re-invents AG-09's scheduler invariants in a second file — violates NFR-TLS-003's single-substrate discipline.

**Substrate preservation (NFR-TLS-003, 7th carry):** zero edits to `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`. `TestTurn_SubstrateUntouched` filter widens to exclude `permission_protocol.go` + `permission_protocol_test.go`.

**Strict TDD (RED-first):** all four leaves are behavior → all four RED-first. `R-TLS-008` parked-set invariant source-guarded (bite). AG-10.3 cancellation scenario = bite. AG-10.4 `CardinalityAtMostOne` = bite.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/agent/scheduler.go` | Modified | Inject `policy PermissionPolicy` into `Schedule`; own parked-set (`map[string]chan struct{}` keyed by `callID`); per-call gate inside each goroutine's `executeCall`; preserve indexed `[]Result` rejoin + single-writer `LaneStamper` + AG-09's three sub-paths (read/serialize/orphan) byte-for-byte. Signature widens by exactly one parameter. |
| `backend/agent/src/agent/loop.go:240-244` | Modified | Pass `policy` into `Schedule`; route upward-path wake through per-call close |
| `backend/agent/src/agent/permission_protocol.go` | New | `PermissionPolicy` port + `PermissionVerdict` struct + per-call wake glue + per-call abort |
| `backend/agent/src/agent/permission_protocol_test.go` | New | AG-10.1..10.4 scenarios + bites (`package agent_test`) |
| `openspec/changes/cachicamas-agent-permission-protocol/specs/agent-permission-protocol/spec.md` | New (delta) | AG-10 capability delta |
| `openspec/specs/agent-permission-protocol/spec.md` | New (at archive) | Canonical spec, merged from delta by `sdd-archive` |
| `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go` | **NOT TOUCHED** | NFR-TLS-003 7th consecutive carry |

## Risks

| # | Risk | Mitigation |
|---|------|------------|
| 1 | Parked-call rejoin breaks `[]Result` ordering | Source-guard `R-TLS-008`; parked-set keyed by `callID`; wake in same goroutine; result slot pre-populated |
| 2 | Deny ≠ Go error — must surface as typed outcome | `Result{Outcome: ExecutionFailure, Failure: typedDenial}`; mirror orphan path; `loop.go:631-645` carries `Content` |
| 3 | AG-10.3 cancellation leaves parked call waiting | `select { case <-parkCh: case <-ctx.Done() }` → typed `ExecutionFailure`; order: park-goroutine → result → emissions close → `Schedule` returns |
| 4 | Modify-input transparency — `ToolStart` must show modified args | Defer `ToolStart` emission until verdict; `ModifyInput` rewrites `arguments`; byte-equality bite |
| 5 | Port signature too thin for Layer 3 needs | Minimal contract now; richer context = future `PermissionPolicy.WithContext(...)` decorator (AG-08 seam-1 `PreRequestHook` precedent) |
| 6 | Substrate byte-cleanliness (NFR-TLS-003 7th carry) | Source guard widens at apply; AG-07 → AG-08 → AG-09 precedent |
| 7 | `CardinalityAtMostOne` on `permission_resolution_remembered` | Single-emission discipline in `Remember` boolean hand-off; S-APE-082 validator bites |
| 8 | R-AGE-006 narrowing — committed facts must reach harness | `decision_required` emitted BEFORE park wait; flushed before `Schedule` returns |
| 9 | Per-call abort scope bleeds into AG-14 | Use only the `context.Context` already threaded (`_ context.Context` at `scheduler.go:89`); AG-14 owns its tree |
| 10 | Review budget (forecast 2500–3500 lines) | Pre-authorized `size:exception`; AG-09 precedent 4258 (#3027) |
| 11 | Wire-up edits `loop.go:240-244` + `scheduler.go` signature | `Schedule` widens by exactly one `policy` parameter; three sub-paths and rejoin stay byte-clean; loop stays stateless; AG-09's `finalize → Schedule → closeSink` order preserved |

## Rollback Plan

Single revert of the AG-10 merge commit. AG-09's scheduler invariants (indexed `[]Result`, single-writer `LaneStamper`, AG-09's three sub-paths) return to their prior shape; the `Schedule` signature loses the `policy` parameter. `loop.go:240-244` returns to its direct-`Schedule(...)` shape. `permission_protocol.go` + `permission_protocol_test.go` are deleted. AG-06.1's three permission events stay constructible and untouched. No migrations, no data. Re-running AG-09's `make test` confirms zero regression.

## Dependencies

- **AG-06.1** (shipped, merged in Layer 2 Wave 1) — permission events constructible (`permission_events.go:53-80`, `:138-154`, `:242-259`, `:326-335`)
- **AG-09** (shipped, PR #169, base `6de08335`) — hand-rolled scheduler + `Tool` contract + `PolicySlot` opaque discipline
- **doc 0004 CO-03** — Layer 3 implements `PermissionPolicy` (consumed only, not edited)
- **doc 0003 lines 1005-1111** — AG-10 charter + four Gherkin leaves
- **doc 0001 § 4.1, § 5.1** — G1's protocol/port distinction

## Success Criteria

- `cd backend/agent && make test` green with `-race`; all 4 Gherkin leaves + bites closed
- Source-guard test for parked-set invariant (R-TLS-008 carry) — bit must RED before green
- Bite test for AG-10.3 cancellation: parked call → typed abort failure under `-race`, no goroutine leak
- AG-10.4 `CardinalityAtMostOne` bite: second `permission_resolution_remembered` for same `toolName` fails validator (S-APE-082 carry)
- `TestTurn_SubstrateUntouched` filter widens to exclude `permission_protocol.go` + `permission_protocol_test.go`; NFR-TLS-003 7th carry holds
- `make lint` clean (after `cache clean`); `make build` clean; `make vuln-check` clean
- Zero edits to `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`, `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`
- AG-09 invariants preserved: 25 kinds constructible, `S-AEV-090` scope-fence untouched