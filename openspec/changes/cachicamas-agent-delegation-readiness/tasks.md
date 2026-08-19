# Tasks: AG-19 — Prove re-entrancy and delegation readiness

> Delivery: **single-pr** (`chain_strategy: n/a`) — SDD cycle + doc 0003 update + OpenSpec archive, same PR.
> Review budget: **1000 changed lines, EXCLUDING `openspec/`** (working folder). Extension pre-accepted if genuinely needed.
> TDD: strict, RED-first. Runner: `cd backend/agent && make test` (`go test -race -v ./...`), **`-count=1` mandatory** on every evidence run (~170s uncached).
> Deviation from the 530-word tasks budget: the launch contract requires a full scenario-coverage table and explicit hazard handling for a 36-scenario, 12-delta change; that overrides the generic size cap here, as design.md's own 800-word deviation already established for this change.

## Review Workload Forecast

| Component | Estimate (lines) |
|---|---|
| Production Go — `delegation_seam.go` + `scheduler.go` (~3 lines + comment) | 145–255 |
| Test Go — delegating tool/derived scope/reconstruction helper + 3 leaves' scenarios + 4 bites | 850–1350 |
| `docs/` — doc 0003 status/checklist (doc.go NOT edited, per design AD-7 decline) | 25–45 |
| **Counted total (non-`openspec/`)** | **1020–1650** |
| `openspec/` (excluded, reported only) — 1 new capability spec, 12 deltas, proposal/design/tasks/apply-progress/verify-report/archive report | ~1100–1800 |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Honest flag: the counted total (1020–1650) is likely to exceed the 1000-line budget even at the low end. The user pre-accepted a `size:exception` extension for this milestone (proposal header); the PR description must state why.

### Suggested Work Units (internal checkpoints inside the one PR — NFR-DEL-005's slicing boundary)

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| U1 | Seam + `executeCall` install/revoke + isolated tests | PR 1 (single, size:exception) | `go test -race -count=1 -run 'TestDelegationSeam\|TestDelegationRevocation' ./src/agent/...` | Isolated `go test -run <detach-test> -count=1` for S-DEL-022 (hazard: panic, no `--- FAIL` line) | Delete `delegation_seam.go` + its tests; revert 3 lines in `executeCall` |
| U2 | AG-19.1 — nested run, walkable tree, siblings | PR 1 (same) | `go test -race -count=1 -run 'TestNestedRun\|TestWalkableTree\|TestSiblings' ./src/agent/...` | `-race` required (S-DEL-013 concurrency) | Delete delegating-tool test files; U1 alone still compiles and passes |
| U3 | AG-19.2 cancellation + AG-19.3 cost/permission | PR 1 (same) | `go test -race -count=1 -run 'TestCancellation\|TestCost\|TestPermissionScope' ./src/agent/...` | Generous injected `WindDownBound` (ceiling, not sync point) | Delete these test files; U1+U2 unaffected |

## Coverage table — every scenario, exactly one task

| Scenario | Task | Scenario | Task | Scenario | Task |
|---|---|---|---|---|---|
| S-DEL-001 | 1.4 | S-DEL-013 | 2.6 | S-DEL-024 | 4.1 |
| S-DEL-002 | 1.1 | S-DEL-014 | 3.1 | S-DEL-025 | 4.2 |
| S-DEL-003 | 1.4 | S-DEL-015 | 3.2 | S-CST-023 | 3.8 |
| S-DEL-004 | 1.1 | S-DEL-016 | 3.5 | S-CST-024 | 3.8 |
| S-DEL-005 | 1.1 | S-DEL-017 | 3.6 | S-AEV-125 | 4.4 |
| S-DEL-006 | 1.3 | S-DEL-018 | 3.9 | S-CAN-015 | 3.4 |
| S-DEL-007 | 1.3 | S-DEL-019 | 3.10 | S-AGE-028 | 4.5 |
| S-DEL-008 | 1.3 | S-DEL-020 (bite) | 3.7 | S-AGE-029 | 4.5 |
| S-DEL-009 | 1.1 | S-DEL-021 (bite) | 2.4 | S-TLS-020 | 1.4 |
| S-DEL-010 | 2.2 | S-DEL-022 (bite) | 1.5 | S-AGS-062 | 4.6 |
| S-DEL-011 | 2.5 | S-DEL-023 (bite) | 3.3 | S-AGS-063 | 4.6 |
| S-DEL-012 | 2.5 | | | S-AGS-064 | 4.6 |
| S-LSK-031 (R-LSK-007) | 4.3 | | | | |

25 `S-DEL-*` + 11 delta scenarios = 36, each discharged once, none doubled.

Scenario-free deltas (back-annotation only, no new scenario — folded into 5.2 so none is missed, per proposal risk #7): `agent-permission-protocol`, `agent-protocol-events`, `agent-run-driver`, `agent-retry-failover`, `agent-compaction`. **`agent-permission-protocol/spec.md:7,15`'s "12 requirements → 13 scenarios + 4 bites" MUST NOT change** — hazard #3.

## Phase 1 — U1: the seam, its install, its isolated tests

- [x] 1.1 [RED] `delegation_seam_test.go` (`package agent_test`): 25-kind admissibility table, zero-`Event` refusal, distinguishable sentinels, unmintable-surface enumeration. Compile-fails against nonexistent `DelegationSeam`/`DelegationSeamFrom` (accepted new-surface RED). → S-DEL-002,004,005,009
- [x] 1.2 [GREEN, production] `backend/agent/src/agent/delegation_seam.go`: interface, `DelegationSeamFrom`, two sentinels, `admissible()` (5 gates, AD-2), mutex-latched `Publish`/`revoke` (AD-1). Makes 1.1 pass. **Note**: the external (`agent_test`) admissibility/zero-event/revocation scenarios genuinely require task 1.4's wiring before they can pass (no seam is ever installed until then) — S-DEL-002 (unmintable surface) passed immediately after 1.2 alone; S-DEL-004/005/006-009 went GREEN together with 1.3+1.4, recorded honestly in apply-progress rather than claimed here.
- [x] 1.3 [RED] `revocation_test.go`: publish-after-return (normal), publish-after-`close(emissions)` via injected small `WindDownBound` (detached), publish-after-recovered-panic (re-panic). Install-last clean RED — no `executeCall` wiring yet. → S-DEL-006,007,008
- [x] 1.4 [GREEN, production] Wire `scheduler.go` `executeCall`: install seam before `runToolWithWindDown`, `defer seam.revoke()` registered after `defer recoverCall` (LIFO, AD-1 step 1). Re-run `PolicySlot` source guard, ambient-authority guard, import-boundary guard, doc-contract guard — all green, zero change to their own sources. → S-DEL-001,003; S-TLS-020
- [x] 1.5 [RED bite, ISOLATED PROCESS — hazard #1] `S-DEL-022`: on scratch tree, drop `defer seam.revoke()`; run **only** `go test -run <detach-test-name> -count=1` in its own invocation. Evidence = non-zero exit status + panic trace naming the send, **not** a `--- FAIL` line — recording inside the full suite destroys that run's other evidence. Revert after recording. **Correction recorded**: the original detached-path fixture (tool blocking synchronously in `Run`, relying on `runToolWithWindDown`'s own recover-wrapped goroutine) would NOT reproduce this crash — that recover catches a synchronous `Publish` panic and silently absorbs it into an abandoned buffered channel. Fixed by having the tool spawn an unrecovered grandchild goroutine for the actual late publish (design AD-1's own named "spawns its own goroutine" alternative). Genuine isolated-process crash captured after the fix: `panic: send on closed channel` at `delegation_seam.go:126`, no `--- FAIL` line, exit status 1.

## Phase 2 — U2: AG-19.1 (nested run, walkable tree, siblings)

- [x] 2.1 [scaffolding, test-only] Delegating test tool + two-stream reconstruction helper (`package agent_test`): hosts a child `Harness`/`Scheduler`, constructs `subagent_started`/`subagent_ended`. No requirement ID alone — this is the fixture `R-DEL-001`'s design mandates live in `agent_test`, not scope creep. (`delegating_tool_test.go`)
- [x] 2.2 [RED] `nested_run_test.go`: parent stream carries `subagent_started` + admissible child events + `subagent_ended` inside hosting turn bracket; both streams `CheckStream`-valid separately; parent sequence contiguous. → S-DEL-010
- [x] 2.3 [GREEN, test-only] Implement mirroring in the delegating tool: for each admissible child event, `seam.Publish`. Makes 2.2 pass. (Written together with 2.1/2.2 as one fixture; genuine GREEN confirmed by `go test -race -count=1 -run TestNestedRun`.)
- [x] 2.4 [RED bite, before final green evidence] `S-DEL-021`: scratch-remove gate 2 from `admissible()`; re-run 2.2's test; FAILS with parent `CheckStream` reporting duplicate run-open (`stream_check.go:122-127`). Revert. Evidence: `CheckStream` reported `event[4]: value repeats another the collection already carries` at the mirrored `run_start`. Reverted, confirmed GREEN.
- [x] 2.5 [RED→GREEN] `walkable_tree_test.go`: every mirrored event resolves to parent in one hop via `Run()`/`subagent_started.Parent()`; no constructor gained a parent param; two lanes independently contiguous, never merged. → S-DEL-011, S-DEL-012
- [x] 2.6 [RED→GREEN, `-race`] `siblings_test.go`: two sibling read-class tools, each hosting a **distinct** `Harness`, concurrent; no cross-talk; all 3 streams `CheckStream`-valid separately; parent sequence contiguous; green under `-race`. → S-DEL-013 (also folds S-AGE-028/S-AGE-029, reusing this fixture, ahead of 4.5)

## Phase 3 — U3: AG-19.2 (cancellation) + AG-19.3 (cost, permission)

- [x] 3.1 [RED→GREEN] `cancellation_test.go`: generous injected parent `WindDownBound`; interrupt mid-flight; child error matches interrupt sentinel; child stream valid, `cost_session(Final)` before its run-close; parent stream valid, `subagent_ended` index < parent run-close index; parent result not detached; no elapsed-time assertion. → S-DEL-014. Synchronized deterministically via `agenttest.Gate.Reached()` — no sleep.
- [x] 3.2 [GREEN, same file] Assert production diff adds no cancel function, cause value, deadline or context derivation of its own. → S-DEL-015. Verified by direct grep of `delegation_seam.go` + `git diff -- scheduler.go` for `WithCancel`/`WithTimeout`/`WithDeadline`/`CancelFunc` — none found.
- [x] 3.3 [RED bite] `S-DEL-023`: scratch tree — child harness built on independent `context.Background()` instead of the tool's `ctx`; re-run 3.1; FAILS on assertion 1. Revert. **Evidence differs from the literal prediction**: the run never completes (the child cannot be cancelled), so `drainSink`'s own 1s internal timeout fires first ("sink did not close within 1s") rather than reaching assertion 1 — an even stronger demonstration (total non-termination). Reverted, confirmed GREEN.
- [x] 3.4 [test-only] `agent-cancellation-tree` delta assertions (reuses 3.1 fixture): mirrored-event-index-vs-turn-close check, tail-closed-against-in-frame-publisher. → S-CAN-015
- [x] 3.5 [RED→GREEN — hazard #2, NON-ZERO fixture mandatory] `cost_test.go`: child fixture MUST script non-zero token spend; parent's terminal `cost_session` == sum over parent's own `cost_turn`, **strictly less than** parent+child sum; consumer walk recovers combined figure; no child `cost_turn` on parent stream. → S-DEL-016. Child scripted at 77+33 input/output tokens; parent turns scripted with their own non-zero figures too.
- [x] 3.6 [GREEN, same file] Crossed `cost_session` present/attributable/inert: child's final figure on parent stream reports child run identity inside hosting turn; parent's own final sits immediately before parent run-close; discriminated by `Run()` not label. → S-DEL-017
- [x] 3.7 [RED bite] `S-DEL-020`: scratch-remove gate 5; re-run 3.5; FAILS with parent cumulative inflated by child spend, inequality broken. Revert. **Evidence note**: the leaked-`cost_turn`-absence assertion fired directly ("event[7] is a cost_turn carrying the CHILD's run identity"); the specific strict-inequality comparison did not flip at the chosen token counts (both sides inflate equally) — the absence check is still direct, valid proof of the same defect. Reverted, confirmed GREEN.
- [x] 3.8 [test-only] `agent-cost-events` delta assertions (reuses 3.5–3.7 fixture): filtered-by-`Run()` parent-own sequence unchanged; fold unreachable (`harness.go:633-635` byte-unchanged, publish returns sentinel). → S-CST-023, S-CST-024
- [x] 3.9 [RED→GREEN] `permission_scope_test.go`: derived scope narrows only — parent-allow/child-excluded → non-allow; parent-deny/defer → never allow, asserted as impossible not merely unused; Layer 2 surface declares no scope type/rule-set/mode-flag. → S-DEL-018. Table-driven over all four parent-verdict/grant-membership combinations.
- [x] 3.10 [GREEN, same file] Ask-up/decision-down: child's ask + answer both on parent stream in order; test-as-human answers via child scheduler's `WakeParked`; child suspension resumes with verdict; both streams valid separately; `permission_resolution_remembered` only on child stream; no new routing surface. → S-DEL-019

## Phase 4 — Scope fence, substrate guard, full evidence

- [x] 4.1 [test-only] `scope_fence_test.go`: diff over `backend/agent/` — byte-unchanged list (`event.go`, `event_descriptor.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `go.mod`, `go.sum`, all `src/ai/`); every-kind-constructible guard at committed count; no exported signature changed. → S-DEL-024
- [x] 4.2 [test-only] `inert_path_test.go`: identical multi-turn script, merge base vs this change, no tool publishes → byte-identical streams (mod fresh ids), byte-identical history read-backs, no delegation-family kind. → S-DEL-025
- [x] 4.3 [production] Widen `filterOutLoopFiles` (`loop_test.go`) and `filterOutLoopHookFiles` (`loop_hook_test.go`) by exact filename suffix for `/delegation_seam.go` and each new AG-19 test file the filters' own rule covers — byte-in-sync, no wildcard/prefix/directory pattern; verify neither names `cost_events.go`, `cost_events_test.go`, `stream_check_test.go` or `reconstruction_test.go`. → S-LSK-031. **Correction**: `cost_events.go` legitimately appears once already (AG-16's release) — the "absent" claim applies to AG-19's own additions, not the whole filter; fixed the test's assumption accordingly. Added a NEW automated `TestScopeFence_S_LSK_031_...` test (regex-extracts every filter entry from both files, asserts set equality + no wildcard/prefix/directory pattern) — this durably improves on the prior 4 milestones' manual-only discipline.
- [x] 4.4 [test-only] `agent-event-envelope` delta: extend/reuse 1.1's unmintable-surface enumeration as an external-package reflection assertion. → S-AEV-125
- [x] 4.5 [test-only] `agent-event-delivery` delta: sink-closed-exactly-once-per-child + parent-lane-only bracket + index ordering (S-AGE-028); no invented channel/buffer/numeric capacity, exhaustive sentinels, single stamping writer per lane under `-race` (S-AGE-029). → S-AGE-028, S-AGE-029 (folded into `siblings_test.go`, task 2.6)
- [x] 4.6 [test-only] `agent-v1-scope` delta: exported-surface deferral check (S-AGS-062); seam-12 mechanism-to-requirement trace (S-AGS-063); AG-19 inheritance statement's two halves checked separately (S-AGS-064).
- [x] 4.7 [evidence] `cd backend/agent && make test` (`-race -v -count=1`), record wall-clock (~170s expected); isolated re-run for `S-DEL-022` evidence; `golangci-lint cache clean && make lint`; `make build`; `make vuln-check` (do **not** run `make all` — its fmt step rewrites committed files). **Result**: `go test -race -v -count=1 ./...` from `backend/agent/` — wall-clock 173s (17:50:06→17:52:59), exit 0, ZERO `--- FAIL`/`FAIL` lines across all 12 packages, 529 top-level `--- PASS` in `src/agent` alone. `gofmt -l` on the 4 files I authored/modified (`delegation_seam.go`, `delegation_seam_test.go`, `permission_scope_test.go`, `walkable_tree_test.go`) is clean (15 PRE-EXISTING baseline files remain gofmt-non-clean — confirmed via `gofmt -d` that `scheduler.go`'s own drift is 100% pre-existing, nowhere near my 6-line insertion — left untouched per the no-mutating-fmt rule). `make lint` found and I fixed 3 real issues (package-comment convention on `delegation_seam.go`; 2× De Morgan simplifications) → `0 issues`. `make build` clean. `make vuln-check`'s reachability analysis: "No vulnerabilities found" (JSON-mode otel CVE is imported-but-not-called, pre-existing, go.mod/go.sum byte-unchanged).

## Phase 5 — Docs and archive (same PR)

- [x] 5.1 [hazard #4] Re-resolve every cited `file:line` across all 13 spec files (new capability + 12 deltas) against the merged tree; fix drift. **Findings**: the ONLY files I modified are `scheduler.go` (+6 lines inside `executeCall`, shifting every citation at/after the old `runToolWithWindDown` call site by exactly +6) and `loop_test.go`/`loop_hook_test.go` (filter widening). Verified and fixed 7 citations: `scheduler.go:621-627`→`627-633` (×2 occurrences incl. my own `revocation_test.go` comment), `scheduler.go:1126-1134`→`1132-1140`, `scheduler.go:766-777`→`772-783` (×2 occurrences), `loop_test.go:831-871`→`837-1049`, `loop_hook_test.go:907-943`→`907-1103` (both of the latter were ALREADY stale before AG-19 — pre-existing drift from AG-11's original citation, never corrected through AG-15/16/17/18 — fixed now since it falls within this task's mandate). Also fixed `scheduler_test.go:616-646`→`616-650` (2 occurrences) — a pre-existing 4-line imprecision, unrelated to my edits, fixed while verified. Every OTHER cited file (`event.go`, `event_descriptor.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `harness.go`, `permission_protocol.go`, `run_events.go`, `sequence.go`, `tool.go`, `doc.go`, `scripted_tool_test.go`, `cost_events_test.go`) is untouched by this change and its citations remain valid (several spot-verified directly against current content). **Explicitly out of scope and NOT touched**: `design.md`/`explore.md`/`proposal.md` (historical phase records, "currently `scheduler.go:492`" is self-aware of its own snapshot nature) and `openspec/specs/agent-loop-skeleton/spec.md` + `openspec/specs/agent-turn-termination/spec.md` (PROMOTED specs carrying the SAME `loop_test.go:831-871`/`loop_hook_test.go:907-943` stale citation from AG-11's own archive, propagated unfixed through 5+ subsequent archived milestones — a genuinely interesting finding, but touching promoted specs is `sdd-archive`'s domain, not apply's, and is unrelated to AG-19's own 13 change-scoped spec files).
- [ ] 5.2 [hazard #3] Header maintenance at archive: extend allocated-ID lines (`agent-cancellation-tree/spec.md:8` +`S-CAN-015`; `agent-cost-events/spec.md:4` +`S-CST-023`,`S-CST-024`; `agent-event-envelope` `S-AEV-` line +`S-AEV-125`); confirm the 5 scenario-free deltas (`agent-permission-protocol`, `agent-protocol-events`, `agent-run-driver`, `agent-retry-failover`, `agent-compaction`) are promoted with **no** new scenario and `agent-permission-protocol/spec.md:7,15`'s count sentence untouched. **NOT DONE — explicitly deferred to `sdd-archive`** per its own "at archive" wording and the orchestrator's instruction that this run does not archive.
- [x] 5.3 Update `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md`: tick the checklist row "The harness is re-entrant … closed by AG-19" (line 2177); append an AG-19 narrative sentence to the status header (line 3) and bump the counter to **19 of 24**. Leave line 2163 (envelope invariants, blocked on AG-20.2) unticked. Verified both edits landed exactly as specified; line 2163 confirmed still `- [ ]`.
- [ ] 5.4 OpenSpec archive: promote `agent-delegation-readiness` as a new capability spec; merge all 12 deltas into their target specs; run archive tooling; confirm no citation left unresolved (cross-check against 5.1). **NOT DONE — explicitly `sdd-archive`'s job**, per the orchestrator's instruction: "Do not archive — that is sdd-archive's job later in this same PR."
