# Spec — The ask–suspend–resume permission protocol (`agent-permission-protocol`)

> **Change**: `cachicamas-agent-permission-protocol` · **AG-10** (Layer 2, Wave 2, milestone 10 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-10--implement-the-permission-protocol), `0003:1005-1111`
> **Nodes**: AG-10.1 `[leaf]` ask and suspend (`:1026`) · AG-10.2 `[leaf]` four outcomes (`:1050`) · AG-10.3 `[leaf]` suspension does not block (`:1080`) · AG-10.4 `[leaf]` remembered resolutions (`:1098`)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario independently verifiable by a named test.
> **IDs**: `R-APP-NNN` / `S-APP-NNN`. Distinct from `R-TLS-`/`S-TLS-`, `R-LSK-`/`S-LSK-`, `R-APE-`/`S-APE-`, `R-AGE-`/`S-AGE-`, `R-AEV-`/`S-AEV-`, `R-AMT-`/`S-AMT-`, `R-PRH-`/`S-PRH-`.
> **Scenario count**: **12 requirements → 13 spec scenarios + 4 bites (`S-PPB-001`…`S-PPB-004`)**.
> **Traces to**: G1 protocol half (R-10 — ask–suspend–resume); v2 § 6 seam 2. `doc 0001 § 4.1` (the seam), `doc 0001 § 5.1` (the `PermissionPolicy` port).
> **Depends on**: AG-06.1 (permission event family), AG-09 (`Tool`/`Scheduler`/`Result`, merged at `6de08335`), AG-01 (event carrier). **Blocks**: AG-13 (`Harness` owns the upward-path wake wiring), AG-19 (subagent tool scope).

## Coverage

| Charter | Requirements | Spec scenarios | Bites |
|---|---|---|---|
| **4 of 4 leaves** | 12 (`R-APP-001`–`012`) | **13** | **4** (`S-PPB-001`…`S-PPB-004`) |

Charter → spec mapping: AG-10.1 gate → `R-APP-001` / `S-APP-001` + `S-APP-002` + bite `S-PPB-001`; AG-10.1 emit-before-park → `R-APP-002` / `S-APP-003` + bite `S-PPB-002`; AG-10.1 stray wake → `R-APP-003` / `S-APP-004` + bite `S-PPB-003`; AG-10.2 `AllowOnce` → `R-APP-004` / `S-APP-005`; AG-10.2 `Deny` → `R-APP-005` / `S-APP-006`; AG-10.2 `ModifyInput` → `R-APP-006` / `S-APP-007`; AG-10.2 + AG-10.4 `AllowAlways` → `R-APP-007` / `S-APP-008` + bite `S-PPB-004`; AG-10.3 sibling isolation → `R-APP-008` / `S-APP-009`; AG-10.3 cancellation → `R-APP-009` / `S-APP-010`; AG-10.4 remembered suppression → `R-APP-010` / `S-APP-011`; layer boundary → `R-APP-011` / `S-APP-012`; substrate carry → `R-APP-012` / `S-APP-013`. Cross-cut (`TurnOptions.PermissionPolicy`) → the `agent-loop-skeleton` delta in the change folder.

## Purpose

Define the protocol half of the ask–suspend–resume seam: a per-call decision gate that consults an injected `PermissionPolicy` before every scheduled tool call, a per-`Schedule` parked set that suspends one call on a `chan struct{}` keyed by `callID` while its siblings keep running, four typed outcomes (`AllowOnce | AllowAlways | Deny | ModifyInput`) surfaced on the event stream, and a cancellation discipline that resolves every parked call into a typed abort so the ordered rejoin is always fully populated.

Layer 2 owns the **protocol**; Layer 3 owns the **answer** (`doc 0001 § 5.1`, doc 0004 CO-03). AG-10 registers no new `EventKind`, adds no `PermissionOutcome` member, and adds no top-level Go dependency — it drives AG-06.1's existing event family through AG-09's existing scheduler.

## Requirements

### R-APP-001 — Per-call decision gate (D1, D2)

The system SHALL consult an injected `PermissionPolicy` exactly once per scheduled tool call, before that call's `ToolStart` is emitted and before its `Run` is invoked. `PermissionPolicy` MUST be an interface with `Resolve(ctx, ai.ToolCall) PermissionVerdict` and `Remember(ctx, toolName string, outcome PermissionOutcome) bool`. A synchronous verdict MUST let the gate dispatch immediately without emitting `permission_decision_required`; a `PermissionDefer` verdict MUST park the call. A **nil** policy MUST be a legitimate identity bypass: the gate returns "proceed" without consulting anything, emits no permission event, and reproduces AG-09's pre-AG-10 behaviour byte-for-byte.

#### Scenarios

- **S-APP-001** — AG-10.1 synchronous allow emits no `decision_required`. Given a scripted `PermissionPolicy` whose `Resolve` returns `PermissionVerdict{Outcome: PermissionOutcomeAllowOnce}` for a single read-class call, when `Schedule` runs and a consumer drains `sink`, then no `permission_decision_required` event appears on the stream, the tool executes exactly once, the ordinal `Result` slot carries `Success`, and the policy's own `Resolve` invocation counter reads exactly `1` — the gate genuinely ran rather than being short-circuited. Verified by `TestPermission_ImmediateAllow_NoEvent`.
- **S-APP-002** — AG-10.1 a deferred call parks while a synchronous sibling completes. Given two calls A and B where the policy returns `PermissionDefer` for A and `AllowOnce` for B, when `Schedule` runs, then A is registered in the parked set and does not execute, B executes to completion and its ordinal slot carries its typed `Result`, and B's completion does not depend on A being woken. Verified by `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot`.

### R-APP-002 — Registration precedes emission; the ack gates the park (D4)

On a `PermissionDefer` verdict the system MUST register the call in the parked set **before** emitting `permission_decision_required`, so a consumer that reads the event off `sink` and immediately wakes the call cannot race ahead of registration. Registration MUST NOT block — it is a mutex-guarded map insert, not the parked wait. The system MUST then wait for an acknowledgement the dispatcher closes only **after** `sink <- &stamped` has returned, so the parked **wait** begins strictly after the emission has genuinely reached `sink` rather than merely entering an internal buffer. That acknowledgement wait MUST be cancellation-aware: on `ctx.Done()` the system MUST deregister the parked entry, write a typed abort `Result` into the call's ordinal slot, emit the matching `tool_end_execution_failure`, and return without parking.

#### Scenarios

- **S-APP-003** — AG-10.1 the event reaches `sink` before the parked wait blocks. Given one call whose policy returns `PermissionDefer` and a consumer holding `sink`, when `Schedule` runs, then `permission_decision_required` for that `callID` is delivered on `sink`, the call is already registered in the parked set at the moment the event is readable (a wake issued immediately on receipt succeeds rather than returning `ErrStrayDecision`), and the tool's `Run` has not been invoked. Verified by `TestPermission_DeferEmitsBeforePark`, with the acknowledgement half exercised by `TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery` and the resume half by `TestPermission_WakeParked_ResumesAndCompletes`.

### R-APP-003 — A stray wake is a typed rejection, never a silent drop (D-A, D-B)

`Scheduler.WakeParked(callID string) error` MUST return the typed sentinel `ErrStrayDecision` when `callID` is not currently registered in an active parked set — whether because no `Schedule` call is in flight, because the ID was never parked, or because it has already been woken, cancelled, or deregistered. The rejection MUST NOT panic and MUST NOT be silently absorbed. A rejected wake MUST leave every other parked entry untouched: the lookup, the delete, and the `close` happen under one lock acquisition, and a miss returns before any mutation.

#### Scenarios

- **S-APP-004** — AG-10.1 wake to an unknown `callID` is rejected and touches nothing. Given one genuinely parked call `callID-A` and a wake issued for `callID-X`, which was never parked, when `WakeParked("callID-X")` is called, then it returns an error satisfying `errors.Is(err, ErrStrayDecision)`, `callID-A` remains parked and unexecuted, and a subsequent `WakeParked("callID-A")` still succeeds — proving the miss mutated no state. Verified by `TestPermission_StrayDecisionIsTypedError` and `TestPermission_WakeParked_UnknownCallID_TypedRejection_NoTouch`.

### R-APP-004 — `AllowOnce` executes and is recorded on the stream

On an `AllowOnce` verdict the system SHALL emit `permission_decision_made{outcome: AllowOnce}` for the call and then proceed with execution using the call's original arguments. If the `decision_made` constructor fails, the system MUST NOT proceed as though the decision had been recorded: it MUST write a typed execution failure into the ordinal slot and abort that call.

#### Scenarios

- **S-APP-005** — AG-10.2 `AllowOnce` executes and is recorded. Given a policy returning `AllowOnce` for a single call, when `Schedule` runs and a consumer drains `sink`, then exactly one `permission_decision_made` carrying `outcome == PermissionOutcomeAllowOnce` for that `callID` appears on the stream, the tool's `Run` is invoked exactly once with the original arguments, and the ordinal slot carries a `Success` `Result`. Verified by `TestPermission_FourOutcomes` (`AllowOnce` subtest) and `TestNoOpPermissionPolicy_AllowsEverySynchronously`.

### R-APP-005 — `Deny` surfaces as a typed result, not a Go error (D6)

On a `Deny` verdict the system SHALL skip execution entirely and SHALL populate the call's ordinal slot with `Result{Outcome: ExecutionFailure, Failure: <typed denial>}`, carrying the call's correlation ID, so the model observes the denial as a typed outcome. The denial MUST NOT be returned as a Go `error` from `Schedule`, which would hide it from the typed-failure surface. A `Deny` verdict that arrives without a populated `*Failure` is a policy defect; the system MUST still populate the slot with a typed execution failure derived from a documented sentinel rather than leaving the slot zero. The system SHALL emit `permission_decision_made{outcome: Deny}` and the matching `tool_end_execution_failure`.

#### Scenarios

- **S-APP-006** — AG-10.2 a denied call yields a typed failure in its own ordinal slot. Given a policy returning `Deny` with a typed `*Failure` for call index 1 of three calls, when `Schedule` runs, then `Schedule` returns normally with no Go error, `results[1].Outcome == ToolOutcomeExecutionFailure` with a non-nil typed `*Failure` and `results[1].CallID()` equal to the input call ID, the denied tool's `Run` was never invoked, the siblings' slots carry their own outcomes, and the stream carries `permission_decision_made{Deny}` followed by `tool_end_execution_failure`. Verified by `TestPermission_FourOutcomes` (`Deny` subtest), `TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin`, and `TestTurn_PermissionPolicy_E2E_DeferDenyModify` at the `Turn()` level.

### R-APP-006 — `ModifyInput` is transparent on the stream (D5)

On a `ModifyInput` verdict the system SHALL defer the call's `ToolStart` emission until the verdict is known and SHALL then execute the tool with the substituted arguments. `permission_decision_made` MUST carry the modified arguments, and the subsequently emitted `ToolStart.Arguments()` MUST byte-equal `decision_made.ModifiedArguments()`. A consumer reading only the stream MUST therefore always be able to tell exactly which bytes the tool actually ran with. If the `decision_made` constructor fails — for example because a policy returned `ModifyInput` without populating the arguments — the system MUST NOT fall back to executing with the original arguments while the stream stays silent; it MUST abort that call with a typed execution failure.

#### Scenarios

- **S-APP-007** — AG-10.2 modified arguments reach the tool and the stream in lockstep. Given a policy returning `ModifyInput` with `ModifiedArgs = {"cmd":"ls"}` for a call whose original arguments differ, when `Schedule` runs and a consumer drains `sink`, then `tool_start.Arguments()` is byte-equal to `decision_made.ModifiedArguments()`, the tool records having been invoked with exactly those same bytes, and no `ToolStart` carrying the original arguments appears anywhere on the stream. Verified by `TestPermission_FourOutcomes` (`ModifyInput` subtest), `TestPermission_RTLS008_SourceGuard_MixedOutcomesFullRejoin`, `TestPermission_ModifyInput_ConstructorFailure_DoesNotSilentlyProceed`, and `TestTurn_PermissionPolicy_E2E_DeferDenyModify`.

### R-APP-007 — `AllowAlways` invokes `Remember`, which gates one emission (AG-10.4)

On an `AllowAlways` verdict the system SHALL emit `permission_decision_made{outcome: AllowAlways}`, proceed with execution, and MUST invoke `policy.Remember(ctx, toolName, PermissionOutcomeAllowAlways)`. A `true` return records the tool name for the remainder of this `Schedule` call and emits `permission_resolution_remembered`; a `false` return suppresses the emission. The record MUST be written by a compare-and-set so that at most one `permission_resolution_remembered` reaches the stream per tool name per `Schedule` call even when two calls for the identical tool name resolve concurrently, preserving `CardinalityAtMostOne` **by construction** rather than by downstream stream validation. That compare-and-set MUST NOT serialize `Resolve`, defer either call, or otherwise narrow AG-09.2's read-class concurrency: both racing calls still run concurrently and both still execute; only the second emission is suppressed.

#### Scenarios

- **S-APP-008** — AG-10.4 `Remember` gates exactly one emission. Given a policy returning `AllowAlways` and a `Remember` that returns `true` in the first branch and `false` in the second, when `Schedule` runs each branch, then the tool executes in both branches, the `true` branch produces exactly one `permission_resolution_remembered` on the stream, the `false` branch produces zero, and under a forced concurrent race between two calls for the identical tool name the count is exactly one while both tools still record two invocations. Verified by `TestPermission_AllowAlways_Remember_Branches` (both subtests) and `TestPermission_RememberedCardinality_ConcurrentRace_AtMostOneEmission`.

### R-APP-008 — A parked call isolates no sibling (D2, AG-09.2 carry)

A parked call MUST NOT block any sibling call, the read-class fan-out semaphore, the serialized mutating/execute lane, or the downstream emission channel. Every non-parked call SHALL reach its own ordinal slot in the rejoin independently of whether, or when, a parked sibling is woken.

#### Scenarios

- **S-APP-009** — AG-10.3 a read-class sibling reaches its ordinal slot while a call stays parked. Given call A deferred and held parked, and call B a read-class call the policy allows synchronously, when `Schedule` runs and B completes, then `results[B]` carries B's typed `Result` with B's own correlation ID, B's `ToolStart`/`ToolEnd` pair is observable on `sink` while A is still parked, and no read-semaphore slot is held by the parked call. Verified by `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot`, extended to the `Turn()` level by `TestTurn_PermissionPolicy_E2E_DeferDenyModify`.

### R-APP-009 — Cancellation resolves every parked call into a typed abort (D7)

When the run's `context.Context` is cancelled while calls are parked, every parked call SHALL be released and SHALL write `Result{Outcome: ExecutionFailure, Failure: <typed abort>}` into its own ordinal slot, carrying its correlation ID, and SHALL emit the matching `tool_end_execution_failure`. Each releasing call MUST deregister its own parked entry so a late wake observes `ErrStrayDecision` rather than closing a channel nothing will read. The rejoin slice returned by `Schedule` MUST be fully populated, and no gate goroutine may wait forever: `runtime.NumGoroutine()` SHALL return to its pre-`Schedule` baseline. Cancellation MUST be honoured on both waits — the acknowledgement wait of `R-APP-002` and the parked wait itself.

#### Scenarios

- **S-APP-010** — AG-10.3 two parked calls, one cancel, a fully populated rejoin. Given two calls the policy defers and a run context cancelled while both are parked, when the cancellation fires, then both ordinal slots carry `ExecutionFailure` with a typed `*Failure` derived from `ctx.Err()` and their own call IDs, `Schedule` returns rather than hanging, a subsequent `WakeParked` for either ID returns `ErrStrayDecision`, and the goroutine count returns to baseline under `-race`. Verified by `TestPermission_CancellationMidPark_PopulatesRejoinAndNoLeak`, with the acknowledgement-wait arm covered by `TestPermission_AbandonedAckWithCancel_GateDeregistersPromptly`.

### R-APP-010 — A remembered resolution suppresses the ask, not merely the event (AG-10.4)

Once `policy.Remember` has returned `true` for a tool name within a `Schedule` call, every later call to that same tool name in that same `Schedule` call MUST proceed **without consulting `Resolve` at all** — the suppression happens before the ask, not after it. The stream for such a call MUST carry neither `permission_decision_required` nor `permission_decision_made`. The remembered set's lifetime is exactly one `Schedule` call; cross-run and cross-session persistence of a remembered rule is Layer 3's concern and is explicitly out of scope.

#### Scenarios

- **S-APP-011** — AG-10.4 the second identical call is never asked. Given a policy that returns `AllowAlways` with `Remember` returning `true` for `toolName = "read_file"`, and two sequential calls to `read_file` in one `Schedule`, when `Schedule` runs, then the policy's `Resolve` invocation counter reads exactly `1`, the second call still executes, and the stream carries no `permission_decision_required` and no second `permission_decision_made` for that tool. Verified by `TestPermission_RememberedSuppressesSubsequentAsk`.

### R-APP-011 — Layer 2 owns the protocol; Layer 3 owns the policy (D1)

The `PermissionPolicy` port MUST be declared in Layer 2 (`backend/agent/src/agent`) and Layer 2 MUST NOT define rule sets, mode flags, allow/deny lists, or any other policy **content**. Implementations are supplied from outside the package; the scheduler MUST consume them through the interface without naming any concrete type. Layer 1 MUST NOT be able to hold an implementation — the ADR 0005 § D1 import boundary forbids Layer 1 (including `src/agenttest`) from importing Layer 2, which is enforced as a hard test failure.

#### Scenarios

- **S-APP-012** — Layer boundary holds from both sides. Given the `PermissionPolicy` interface declared in `permission_protocol.go` and at least one implementation living in an external package, when the scheduler drives that implementation end to end, then it is consumed solely through the interface with no type assertion on a concrete policy type, Layer 2's sources contain zero rule sets or mode flags, and a probe that makes a Layer 1 package import Layer 2 fails the boundary guard. Verified by the external-package compile guards on `scriptedPermissionPolicy`, `wiringTestPolicy`, and `NoOpPermissionPolicy`, and from the other side by `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`.

### R-APP-012 — Substrate preservation, 7th consecutive milestone (D8, NFR-TLS-003 carry)

The system MUST NOT modify any of `event.go`, `event_descriptor.go`, `stream_check.go`, `sequence.go`, `import_boundary_test.go`, `backend/agent/go.mod`, `backend/agent/go.sum`, `Makefile`, or `.golangci.yml`. The substrate-untouched guards (`TestTurn_SubstrateUntouched` and `TestTurn_PreRequestHook_SubstrateUntouched`) SHALL widen their allowlists by exactly four exact-filename suffixes for the files AG-10 introduces — `permission_protocol.go`, `permission_protocol_test.go`, `loop_permission_e2e_test.go`, and `permission_policy_helpers_test.go` — with no wildcard, prefix, or directory-level widening.

`failure.go` was a member of this list at AG-10 and is **released for AG-11 only**, and only for the single addition `R-ATT-006` requires: a `PartialOutput() bool` accessor mirroring the nil-safe shape of `Category()` (`failure.go:44`), `Delivery()` (`:54`) and `Retryable()` (`:64`), delegating unchanged to `(*ai.Failure).PartialOutput()` (`provider_failure.go:515-520`). `NewFailure`'s nil rejection (`failure.go:33-38`) and the "AG-04 registers no separate error kind; failures ride the typed outcomes" rule (`failure.go:6-7`) MUST NOT change. Every other file above remains forbidden for AG-11, and this release does not extend to any later milestone without its own recorded delta.

AG-11 SHALL widen both guards' allowlists further by **exact filename suffixes only** — `failure.go`, `turn_events.go`, and one entry per test file AG-11 introduces — with no wildcard, no prefix match, and no directory-level widening, and the two guards' entry sets SHALL remain identical to each other. `turn_events.go` is listed here only because the guards enumerate changed files, not because this requirement ever forbade it; the normative release of `turn_events.go` lives in the `agent-loop-skeleton` delta (`R-LSK-004`).

(Previously: `failure.go` was listed as forbidden without exception, and the only widening rule described was AG-10's four newly-created files.)

#### Scenarios

- **S-APP-013** — Substrate byte-unchanged against the merge base. Given the merge base of the AG-10 branch with `origin/main`, when `git diff` is taken over `backend/agent/src/agent/` and over `go.mod`/`go.sum`, then only allowlisted files differ, every listed substrate file is byte-unchanged, the `go.mod`/`go.sum` diff is empty, the every-kind-constructible guard still passes at 25 kinds (AG-10 adds zero), and both substrate guards pass. Verified by `TestTurn_SubstrateUntouched` and `TestTurn_PreRequestHook_SubstrateUntouched`, corroborated by the merge-base diff (11 of 44 `.go` files changed, all allowlisted; 33 byte-unchanged).
- **S-APP-014** — AG-11's release of `failure.go` is bounded and exact. Given the merge base of the AG-11 branch with `origin/main`, when `git diff` is taken over `backend/agent/src/agent/failure.go`, then the only change is the addition of the nil-safe `PartialOutput()` accessor — `NewFailure`, `Category`, `Delivery`, `Retryable` and `Unwrap` are byte-unchanged; and when the two guards' allowlists are compared, then they carry an identical set of exact-filename entries with no wildcard, prefix, or directory pattern; and every file still named forbidden by this requirement is byte-unchanged; and the every-kind-constructible guard still passes at 25 kinds (AG-11 adds zero). Cross-referenced to `R-ATT-006` and `R-ATT-009`.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-APP-001** | External-package verifiability (NFR-TLS-001 carry): every scenario verifiable by `cd backend/agent && make test`; every behavioural test lives in `package agent_test` or another external package. |
| **NFR-APP-002** | Determinism and race cleanliness (NFR-TLS-002 carry): every test deterministic and hermetic, green under `go test -race`, including repeated runs (`-count=15` on `src/agent`). No ambient authority added in non-test sources. |
| **NFR-APP-003** | Substrate byte-unchanged, 7th consecutive milestone (R-APP-012 / NFR-TLS-003 carry). |
| **NFR-APP-004** | No new top-level Go dependency, no new `EventKind`, no new `PermissionOutcome` member — AG-06.1's event family stays byte-clean. |

## Explicit non-requirements

- **Policy content, mode flags, rule sets** — Layer 3 (doc 0004 CO-03). AG-10 ships the protocol only.
- **Cross-session or cross-run persistence of a remembered rule** (CO-16.1) — the remembered set lives exactly one `Schedule` call.
- **The upward-path wake wired into `Turn`** — AG-13. `Turn()` deliberately exposes no scheduler handle and no wake surface at AG-10.
- **Subagent tool scope** — AG-19.3.
- **The full cancellation tree** — AG-14. AG-10 owns per-call abort on the already-threaded `ctx` only.
- **Making `Schedule` return against a permanently abandoned `sink`** — a pre-existing AG-09 hazard. AG-10 releases the gate goroutine on cancellation; the dispatcher's `sink <- &stamped` send still requires a reader.

## Dependencies

- **Depends on**: AG-06.1 (permission event family: `permission_decision_required`, `permission_decision_made`, `permission_resolution_remembered`, `PermissionOutcome` vocabulary); AG-09 (`Tool`, `Registry`, `Scheduler.Schedule`, `Result`, ordered rejoin, bounded read fan-out); AG-01 (event carrier); AG-07 (`Turn` walking skeleton).
- **Closes**: G1 protocol half (R-10); v2 § 6 seam 2.
- **Blocks**: AG-13 (`Harness` wires the upward-path wake and iterates the cycle); AG-19 (subagent tool scope reuses the gate).

## Verification approach

- `cd backend/agent && make test` — full `-race -v ./...`; all 13 scenarios and all 4 bites green.
- `cd backend/agent && make lint` — `golangci-lint` clean after `cache clean`.
- `cd backend/agent && make build` — clean compile.
- `go test -race -count=15 ./src/agent/` — repeated-run stability for the park/wake and cancellation paths.
- Substrate-untouched check against `git merge-base HEAD origin/main`, with the `AG09_BASE_REF` env-var fallback.

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active.

- All four AG-10 leaves are behaviour, so all four are RED-first.
- **`S-PPB-001`** — the gate itself: deleting the whole gate makes `TestPermission_ImmediateAllow_NoEvent` fail on `invocation count = 0, want 1`.
- **`S-PPB-002`** — registration before emission: reverting the ordering makes `TestPermission_DeferEmitsBeforePark` fail deterministically (20/20 full-package runs).
- **`S-PPB-003`** — the stray-wake typed rejection: removing the typed sentinel makes `TestPermission_StrayDecisionIsTypedError` fail.
- **`S-PPB-004`** — `CardinalityAtMostOne`: a hand-built stream with two `permission_resolution_remembered` events is rejected by `CheckStream` with `ai.ErrDuplicate`, and ignoring the compare-and-set return value makes `TestPermission_RememberedCardinality_ConcurrentRace_AtMostOneEmission` fail with `count = 2, want exactly 1`.
- **Known gap (carried to AG-13)**: the `R-APP-002` acknowledgement itself currently has no non-vacuous guard — deleting the acknowledgement leaves the package green. The behaviour is present and correct in production; the missing bite must observe the parked **wait**, not the registration.

## Acceptance criteria

1. Every `S-APP-001`…`S-APP-013` has a named passing test.
2. `make test`, `make lint` (after `cache clean`), and `make build` are all green.
3. `backend/agent/go.mod` and `go.sum` are byte-unchanged.
4. The substrate files listed in `R-APP-012` are byte-unchanged (7th consecutive milestone).
5. The every-kind-constructible guard still passes at 25 kinds; AG-03's two boundary guards pass unchanged.
6. All four bites `S-PPB-001`…`S-PPB-004` are RED-recorded with failing output.
7. The scenario count `12 requirements → 13 scenarios + 4 bites` is stated identically in the proposal, tasks, apply-progress, and verify-report.
8. A nil `PermissionPolicy` reproduces AG-09 behaviour exactly (AG-07/AG-09 regression suites green with no policy injected).

## Traceability

| Requirement | Charter node | Decisions cited | Primary test |
|---|---|---|---|
| `R-APP-001` | AG-10.1 | D1, D2 | `TestPermission_ImmediateAllow_NoEvent` |
| `R-APP-002` | AG-10.1 | D4 | `TestPermission_DeferEmitsBeforePark` |
| `R-APP-003` | AG-10.1 | D-A, D-B | `TestPermission_WakeParked_UnknownCallID_TypedRejection_NoTouch` |
| `R-APP-004` | AG-10.2 | D1 | `TestPermission_FourOutcomes` (`AllowOnce`) |
| `R-APP-005` | AG-10.2 | D6 | `TestPermission_FourOutcomes` (`Deny`) |
| `R-APP-006` | AG-10.2 | D5 | `TestPermission_FourOutcomes` (`ModifyInput`) |
| `R-APP-007` | AG-10.2, AG-10.4 | D1 | `TestPermission_AllowAlways_Remember_Branches` |
| `R-APP-008` | AG-10.3 | D2 | `TestPermission_SuspensionDoesNotBlock_SiblingReachesOrdinalSlot` |
| `R-APP-009` | AG-10.3 | D7 | `TestPermission_CancellationMidPark_PopulatesRejoinAndNoLeak` |
| `R-APP-010` | AG-10.4 | D1 | `TestPermission_RememberedSuppressesSubsequentAsk` |
| `R-APP-011` | boundary | D1 | `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` |
| `R-APP-012` | substrate | D8 | `TestTurn_SubstrateUntouched` |

All four charter leaves are represented; none is reduced. Scenario count stated identically with the proposal: `12 requirements → 13 scenarios + 4 bites`.
