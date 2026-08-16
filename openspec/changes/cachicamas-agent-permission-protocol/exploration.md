# Exploration: `cachicamas-agent-permission-protocol` (AG-10)

> **Change**: `cachicamas-agent-permission-protocol` · **AG-10** (Layer 2 Wave 2, milestone 10 of 24; doc 0003 lines 1005–1111)
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag10`
> **Branch**: `feat/agent-layer2-wave2-ag10` based at `6de08335` (post-AG-09 PR #169 merge)
> **Closes**: G1's protocol half (R-10); v2 § 6 seam 2 (permission decision in the loop).
> **Format**: read-only investigation; no production code written.

---

## Exploration: AG-10 — Implement the permission protocol

### Current State

Layer 2 today, after AG-06.1 (event families shipped, `permission_events.go:1-359` + `event.go:166-177, 312-323`) and AG-09 (hand-rolled scheduler + tool contract, `tool.go:1-246` + `scheduler.go:1-503`), exposes everything AG-10 plugs into except the seam itself:

- **The four-outcome payload is fully constructed.** `PermissionOutcome` is a closed 4-member typed enum (`AllowOnce | AllowAlways | Deny | ModifyInput`, `permission_events.go:53-80`) with `String()` discipline; `NewPermissionDecisionMade` (`permission_events.go:242-259`) accepts `(callID, outcome, modifiedArguments, failure)` and the validator enforces the iff rule (modified args iff `ModifyInput`, typed failure iff `Deny`). `NewPermissionDecisionRequired` (`permission_events.go:138-154`) carries `callID, name, arguments` as distinct readable fields (R-APE-001). `NewPermissionResolutionRemembered` (`permission_events.go:326-335`) carries `toolName, outcome` and declares `CardinalityAtMostOne` on its descriptor row (`event.go:320-323`). S-APE-010..012 + S-APE-020 + S-APE-082 bites already exercise the full surface in `permission_events_test.go:1-318`.

- **The scheduler is hand-rolled and ready.** `Scheduler.Schedule(ctx, calls, reg, runID, turnID, stamper, sink) []Result` (`scheduler.go:88-156`) returns `[]Result` in call order, with three sub-paths (`scheduleRead` bounded by `chan struct{}` semaphore of `MaxConcurrentReads`=8 default, `scheduleSerialized` single-goroutine for mutating+execute, `scheduleOrphan` synchronous for registry miss) and a single dispatcher goroutine that owns the `LaneStamper` (D6b). `PolicySlot(call.ID())` is forwarded byte-exact to `tool.Run` (`scheduler.go:297`). The `Results` slice carries three typed outcomes (`Success | ResultFailure | ExecutionFailure`, `tool.go:137-142`), and a typed `*Failure` is required for the ExecutionFailure slot (`tool.go:164-169`).

- **The loop already calls `Schedule` exactly once per Turn** between `provider.Stream` close and `finalize` (`loop.go:240-244`). The wire-up order is `finalize → Schedule → closeSink` (AG-09 deviation #1, `apply-progress.md`), so the scheduler closes `sink` after the rejoin and the loop's `closeSink` is a `defer/recover` against a double-close (AG-09 deviation #2, `loop.go:279-290`).

- **The upward path is already designed but not yet built.** `agent-event-delivery/spec.md:187-240` (R-AGE-013..017) commits the one-surface-three-kinds design (permission decision + steering + interrupt), the harness-owned call-identity lookup whose shape is left to AG-10/AG-13, and the typed-rejection-at-two-granularities rule. AG-10's `loop.go` extension must consume that surface, not invent a channel.

- **Policy content lives above Layer 2.** `0001:603` names `PermissionPolicy` as one of the five Layer 3 ports ("may this call proceed, and should the answer be remembered"); `0001:494-495` explicitly says the *protocol* (ask, suspend, resume) "must live in the loop, or every frontend reimplements approval out of band", distinguishing from *policy* (which Layer 3 decides). doc 0004 CO-03 (`0004:253-...`) carries Layer 3's policy implementation.

- **AG-10.3 cancellation integrates with AG-14's future tree.** AG-10's charter explicitly names AG-14 as a downstream consumer and states that an aborting run must "wind down" suspended calls without waiting forever (Gherkin AG-10.3 cancellation scenario, `0003:1091-1093`). AG-14's cancellation tree is `task-graph-milestone-doc` Wave 3. The charter scopes AG-10's cancellation to **per-call abort via the loop's existing `context` discipline**, not the full subagent cancellation tree.

### Affected Areas

- `backend/agent/src/agent/loop.go:240-244` — the AG-09 wire-up site. AG-10 replaces the direct `Schedule(...)` call with a permission-gated dispatch path that may suspend one call while siblings continue.
- `backend/agent/src/agent/scheduler.go` — must be extended (or wrapped) so each call goroutine's `executeCall` (or a thin per-call pre-step) consults the policy before the `ToolStart` emission. The hand-rolled invariants (indexed `[]Result`, panic containment, single-writer `LaneStamper`, fan-out bound) MUST stay byte-clean.
- `backend/agent/src/agent/tool.go` — `Tool` interface stays as-is (seam 3 promise: `PolicySlot` is forwarded opaquely, `tool.go:115`). AG-10 does NOT add a second slot.
- `backend/agent/src/agent/permission_events.go` — the three event payloads and constructors stay byte-clean; AG-10 is the emitter, not the constructor author. **Gap check**: `NewPermissionDecisionMade` already carries `modifiedArguments` (S-APE-011) and the typed `*Failure` (S-APE-012); no AG-06.1 surface extension is required for AG-10's four-outcome flow. `NewPermissionResolutionRemembered` carries `toolName, outcome` — enough for AG-10.4's "remembered-eligible" emission after the policy confirms. **No surface gap; AG-06.1 anticipated AG-10 correctly.**
- `backend/agent/src/agent/event.go` — the registry row for `permission_decision_made` and `permission_decision_required` stay `PlacementTurn, CardinalityAny, Terminal:false`; the row for `permission_resolution_remembered` already declares `CardinalityAtMostOne`. AG-10 is a producer; it does not edit `event.go`.
- `backend/agent/src/agent/sequence.go` — substrate, byte-unchanged. AG-10 emits via the existing per-lane `LaneStamper` (no shared-state change).
- `backend/agent/src/agent/failure.go` — `Failure` (`failure.go:28`) carries the typed-failure surface; AG-10 reuses it for the `Deny` outcome's `*Failure` field on `NewPermissionDecisionMade`. No new typed failure needed.
- `openspec/specs/agent-tool-scheduler/spec.md` — the existing capability spec; AG-10 will modify `R-TLS-001` only if AG-10 introduces a new scheduler argument, or leave it untouched if AG-10 wraps the call rather than editing the scheduler signature.
- `openspec/specs/agent-protocol-events/spec.md` — AG-06.1's spec; AG-10 does NOT extend it. The protocol events stay constructible; AG-10 is the emission discipline.
- `openspec/specs/agent-event-delivery/spec.md` — R-AGE-013..017 are inherited constraints AG-10 must honor without inventing a channel.
- New file (proposed): `backend/agent/src/agent/permission_protocol.go` — the new `PermissionPolicy` port interface (`Resolve` + `Remember`), the per-call hold/wake coordination, and the upward-call dispatch glue. Lives beside `permission_events.go` (per AD-6 per-family file split).
- New file (proposed): `backend/agent/src/agent/permission_protocol_test.go` — the AG-10.1..AG-10.4 scenario + bite tests, in `package agent_test` (NFR-TLS-001 external posture).
- `openspec/changes/cachicamas-agent-permission-protocol/specs/agent-permission-protocol/spec.md` — the AG-10 capability spec delta (new file under the change folder's `specs/`).
- `openspec/specs/agent-permission-protocol/spec.md` (canonical at archive) — the source-of-truth spec created by `sdd-archive` from the delta.
- doc 0003 (milestones) — only `0003:1005-1111` is in scope; the closing-checklist row "Permission is a suspension on the stream with all four outcomes; suspension blocks nothing else — closed by AG-10" (`0003:2166`) flips at archive time.

### Approaches

#### 1. **Per-call gate inside the scheduler, hold via dedicated channel** *(recommended)*

Introduce a `PermissionPolicy` port with two methods (`Resolve` + `Remember`). Wrap each call goroutine in a pre-`ToolStart` step that asks the policy synchronously; on a deferring verdict, emit `decision_required` and park the call on a per-call `chan struct{}` while siblings proceed. The scheduler owns the parked set; the loop owns the upward path's "wake parked call" hand-off.

- **Pros**: scheduler invariants stay in one file; the parked set is a small new struct (`map[string]chan struct{}` keyed by `callID`, guard-tested for unknown ids); the wake path is one-line (`close(parkCh)`) consumed by the call goroutine's existing semaphore/serialize block; AG-10.3 cancellation resolves parked calls into a typed `ExecutionFailure` *without* waiting on AG-14's tree; `Result` rejoin is unchanged because the parked call's `Result` slot is pre-populated with `callID` (R-TLS-009) and overwritten on wake.
- **Cons**: introduces a synchronization point per call goroutine (a `chan`); must source-guard the `PolicySlot` discipline (no per-call type assertion); the per-call park channel must be cleared on `Schedule` exit (goroutine leak bite).
- **Effort**: Medium. ~7 work-unit commits mirroring AG-09's 8-commit shape: 1 spec/design chore, 1 RED bites, 3 feat (port + scheduler gate + loop wire-up), 1 end-to-end test, 1 docs/apply-progress.

#### 2. **Per-call gate outside the scheduler, hold via a sidecar registry**

Keep the scheduler byte-clean; introduce a new `PermissionGate` wrapper that the loop calls *before* `Schedule`, splitting the call list into "ask immediately" (immediate allow/deny) and "deferred" (parks the entire turn until all deferreds resolve).

- **Pros**: scheduler.go remains the exact post-AG-09 file; AG-10 lives in two new files (`permission_protocol.go` + `gate.go`).
- **Cons**: turns the per-call suspension into per-turn suspension (the sidecar must hold the entire turn's deferred calls); AG-10.1's "siblings continue" property becomes "the entire turn waits for the slowest policy ask" unless the sidecar holds individual calls inside its own goroutine pool — which re-introduces AG-09's scheduler invariants in a second file. This re-invents the scheduler by another name. **Violates the "single substrate" discipline NFR-TLS-003 proves across milestones.**

#### 3. **Replace scheduler with a third-party pool / `errgroup`** *(rejected)*

- **Pros**: well-known primitives.
- **Cons**: explicit ban in `scheduler.go:18-21` (no new top-level deps; `errgroup` first-error cancellation conflicts with R-TLS-010 "siblings complete"); source-guarded by `TestScheduler_SourceGuard_NoErrgroupImport` (`scheduler_test.go`). Also no extension experiment on the substrate (NFR-TLS-003 7th consecutive milestone).

### Recommendation

**Approach 1** — per-call gate inside the scheduler, hold via a dedicated channel. It is the only option that (a) keeps AG-09's invariants in one file and one byte-clean diff, (b) honors AG-10.3's "blocks neither siblings nor event delivery" without a second scheduling layer, and (c) lets AG-06.1's three permission events stay byte-clean producers — the new file is `permission_protocol.go`, the events ride through the existing `emission` channel via the dispatcher, and the `LaneStamper` single-writer invariant (D6b) holds by construction.

**Minimal `PermissionPolicy` port shape** (Layer 2 owns the contract; Layer 3 owns the implementation):

```go
// PermissionPolicy is the ask–suspend–resume seam AG-10 introduces
// (doc 0001 § 5.1 "PermissionPolicy" port, layer-3-implemented).
// Layer 2 owns the protocol; Layer 3 owns the answer.
type PermissionPolicy interface {
    // Resolve asks the policy for a verdict on one call. The policy
    // returns one of:
    //   - VerdictAllowOnce / VerdictAllowAlways / VerdictDeny / VerdictModifyInput
    //     synchronously — the call proceeds without a decision-required event.
    //   - VerdictDefer — the policy wants a human; the scheduler emits a
    //     decision-required event and parks the call.
    Resolve(ctx context.Context, call ai.ToolCall) PermissionVerdict

    // Remember is invoked after an AllowAlways decision reaches the
    // policy. The policy returns whether the resolution was committed
    // to its persistent store. A true return causes the scheduler to
    // emit permission_resolution_remembered; a false return suppresses it
    // (AG-10.4's "only when the policy says it was remembered").
    Remember(ctx context.Context, toolName string, outcome PermissionOutcome) bool
}

type PermissionVerdict struct {
    Outcome       PermissionOutcome // AllowOnce | AllowAlways | Deny | ModifyInput | Defer
    ModifiedArgs  []byte            // populated iff Outcome == ModifyInput
    Failure       *Failure          // populated iff Outcome == Deny
}
```

**Per-call hold mechanism** (the `defer` vs `park` choice): a parked call goroutine holds its scheduler slot (read semaphore OR serialized channel) so siblings are NOT pushed past the held call's ordinal slot in *call-order terms*. The call goroutine waits on a per-call `chan struct{}` keyed by `callID` in the parked set; the upward-path wake closes that channel; the goroutine then resumes its scheduled `executeCall` step (with `Decision.ModifiedArgs` substituted in for the `ModifyInput` case). The result slot is **pre-populated with `callID`** before parking (R-TLS-009), and re-populated on wake. AG-10.3 cancellation walks the parked set, types each entry as `ExecutionFailure`, and closes the channels — bounded by a context-aware `select`.

**Wire-up placement** (loop-side): `loop.go:240-244` becomes a call to the permission-gated scheduler. The loop's role is unchanged: it owns `provider.Stream` → `finalize` → schedule → `closeSink`. AG-10 keeps the loop stateless (R-LSK-002) — the parked-set lifetime is one `Schedule` call.

### Risks

1. **Scheduler integration invariant risk.** A per-call park inside the scheduler can break the indexed `[]Result` rejoin if the parked call's `executeCall` reorders relative to siblings. Mitigation: the parked set is keyed by `callID`; the wake path runs in the same goroutine that parked; the `results[ordinal]` slot is written exactly once on wake, just like the immediate path. **Source-guard** the byte-cleanliness of `Schedule`'s rejoin invariant (R-TLS-008) with a test that drives 3 parked + 2 immediate calls through 4 outcomes.

2. **Deny ≠ Go error.** A denied call's `Result` must be visible to the model as a typed outcome, NOT a Go error (AG-10.2 acceptance + AG-09.4 R-TLS-007). Mitigation: populate `results[ordinal] = Result{Outcome: ExecutionFailure, Failure: <typed denial failure>}` (mirroring the orphan path at `scheduler.go:239`) and emit `permission_decision_made{outcome=Deny}` + a `tool_end_execution_failure` for the ordinal. The loop's transcript reconstruction (`loop.go:631-645`) must surface the typed denial — the AG-09 `toolResults` slice carries typed `Result.Content`; the deny path needs `Content` populated with the denial rationale.

3. **Cancellation order risk (AG-10.3).** A `cancel` arriving while one or more calls are parked must NOT leave a parked call waiting forever, and MUST NOT skip the `run_end` closure (R-AGE-006 narrowing — "what history already knows must be delivered"). Mitigation: the parked set's wait is a `select { case <-parkCh: case <-ctx.Done(): }`; on `ctx.Done()` the goroutine writes `Result{Outcome: ExecutionFailure, Failure: <typed abort failure>}` and exits. The order is **park-goroutine → result slot write → emissions channel close → scheduler returns → loop finalizes → closeSink**. Add a bite test that cancels mid-park and asserts the rejoin slice is fully populated with typed abort failures.

4. **Modify-input transparency risk (AG-10.2 acceptance).** "modify-input executes with the modified arguments and the stream says so" requires that the `tool_start` event's arguments reflect the **modified** arguments, not the originals. AG-06.1's `ToolStart.Arguments()` is the field that a frontend reads; the scheduler must emit `ToolStart` AFTER the policy verdict, with the substituted `arguments`. The `permission_decision_made{outcome=ModifyInput}` event carries the modified args too (S-APE-011), so a session log can reconstruct what ran. Mitigation: pre-emit `permission_decision_required` before the policy ask; the scheduler defers the `ToolStart` emission until verdict; `ModifyInput` rewrites the `arguments` field; `decision_made` event follows. **Bite test**: assert `ToolStart.Arguments()` byte-equals `decision_made.ModifiedArguments()` for a `ModifyInput` flow.

5. **Policy port signature risk.** A `PermissionPolicy.Resolve(ctx, ai.ToolCall)` that returns synchronously is the simplest shape, but Layer 3's actual policy implementations (doc 0004 CO-03) may need a richer ask — e.g., the call's full transcript context, or a per-tool configuration object. Mitigation: keep the contract minimal in AG-10 (the four outcomes + `Defer`); layer richer context as a future `PermissionPolicy.WithContext(...)` *decorator* (the v2 seam-discipline pattern from `PreRequestHook` at AG-08, which is itself a function value, not an interface). **AG-10's bite**: a `PermissionPolicy` that returns `VerdictDefer` for a call with arguments `{"cmd":"rm -rf /"}` and the wake path reaches `decision_made{outcome=ModifyInput}` — proves the seam holds with no richer context required.

6. **Contract shifts from upstream milestones.** AG-08 introduced `TurnOptions.PreRequestHook` as a function value (not an interface), per the AG-08 design precedent (DRY with seam-1's "minimal contract"). AG-09's `Scheduler` field is value-typed (`Scheduler.MaxConcurrentReads int`) for the same reason. AG-10's `PermissionPolicy` port SHOULD follow the same discipline: an **interface** is the seam because (a) the policy has internal state (remembered rules, mode flag) that a function value cannot carry without closure capture, and (b) the AG-01 upward-path design (R-AGE-013) says "the harness is the stable addressable thing a frontend holds across a whole run" — the policy belongs to the harness's composition root. The **mismatch with AG-08's function-value pattern is intentional, not a drift**.

7. **`CardinalityAtMostOne` cardinality on `permission_resolution_remembered`.** AG-06.1's spec already declares this (`event.go:322`), and the validator's bite at `permission_events_test.go:263-318` (S-APE-082) records it. AG-10 must NOT emit two `permission_resolution_remembered` events for the same `toolName` on one stream. AG-10.4's "stream shows initial resolution_remembered then unasked executions" relies on a single emission at the policy-confirmation point; a second emission in the same run would fail the validator. Mitigation: source-guard the single-emission discipline inside the permission protocol's `Remember` hand-off (the boolean return controls whether to emit at all).

8. **Substrate preservation (NFR-TLS-003 carry, 7th milestone).** AG-10 must not edit `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, or `import_boundary_test.go`. AG-10's only production touch is `loop.go` (the wire-up call) + new files (`permission_protocol.go` + its tests). The substrate filter on `TestTurn_SubstrateUntouched` widens at apply time to include the new files — same pattern as AG-09's `scripted_tool_test.go` location shift (AG-09 deviation #7).

9. **Per-call abort vs. AG-14's cancellation tree.** AG-10.3's per-call abort is the AG-10 charter's scope; AG-14 owns the full cancellation tree (subagent cancellation, history-level interrupt). AG-10 must NOT reach into AG-14's surface (it does not exist yet). AG-10's per-call abort uses only the `context.Context` already threaded through `Schedule` (the parameter currently unused at `scheduler.go:89` — `_ context.Context` is the body; AG-09 noted it as a wire-up point for AG-10/AG-11). This avoids scope bleed.

10. **Event delivery's harness-facing boundary (R-AGE-006).** The AG-01 decision requires "the harness-facing boundary may never drop what history already knows". AG-10's parked-call cancellation must not lose a `permission_decision_required` already on the stream. Mitigation: `permission_decision_required` is emitted **before** the parked wait; even on abort, the `decision_required` event is already in the dispatcher's emissions channel and will be flushed before `Schedule` returns (per R-AGE-006's narrowing).

11. **Review budget risk.** AG-10's natural commit shape (port + scheduler gate + 4-outcome + cancellation + wake-up-path + end-to-end + docs + apply-progress) lands at ~2500–3500 lines including tests. AG-09 was pre-authorized `size:exception` at 4258 lines (per AG-09 archive-report, #3027); AG-10 should request the same exception in the proposal phase.

### Ready for Proposal

**Yes — with one preflight**: the orchestrator should brief the user that AG-10 is the natural size:exception milestone (forecast 2500–3500 lines per AG-09 precedent) and that AG-10's wire-up will edit `loop.go:240-244` (the AG-09 `Schedule` call site). The proposal phase should also flag Approach 2 (sidecar registry) as the only coherent alternative and explain why Approach 1 is recommended. AG-10 blocks AG-13 (multi-turn run driver, which needs the upward path) and AG-19 (delegation, which derives its scope from this protocol), so the cycle should advance promptly.

---

## Cross-references

- **AG-09 archive observation** (#3027, Engram) — "PR open. Ready for AG-10 (permission protocol wraps the scheduler) per the dependency graph in `openspec/specs/agent-tool-scheduler/spec.md`."
- **AG-09 verify-report** (#3026, Engram) — 0 blockers, 6 warnings. AG-09's substrate invariants held. AG-10 inherits those invariants (NFR-TLS-003 carry).
- **AG-09 apply-progress** (#3020, Engram) — 8 work-unit commits; wire-up order deviation (`finalize → Schedule → closeSink`) carries forward to AG-10.
- **AG-08 archive-report** (#3008, Engram) — `TurnOptions.PreRequestHook` is the seam-1 precedent; AG-10's `PermissionPolicy` interface is the seam-2 instantiation.
- **doc 0003 lines 1005-1111** — AG-10 charter and four Gherkin leaves.
- **doc 0003 lines 2207-2208** — R-09 (upward path) consumed by AG-10.1, AG-13.2, AG-14.1; R-10 (permission protocol) discharged by AG-06.1, AG-10.1, AG-10.2, AG-10.3, AG-10.4.
- **doc 0001 § 4.1 lines 483-495** — "Driving the permission protocol" is a loop-owned mechanism; "deciding permission" is a Layer 3 port. This is the G1 distinction.
- **doc 0001 § 5.1 line 603** — `PermissionPolicy` port named (Layer 3 implements).
- **`openspec/specs/agent-event-delivery/spec.md` lines 187-240** — R-AGE-013..017 inherited constraints.
- **`openspec/specs/agent-protocol-events/spec.md` lines 27-53** — R-APE-001..003 constructors (already shipped; AG-10 emits them).
- **`openspec/specs/agent-tool-scheduler/spec.md` lines 25-100** — R-TLS-001..011 invariants AG-10 inherits.