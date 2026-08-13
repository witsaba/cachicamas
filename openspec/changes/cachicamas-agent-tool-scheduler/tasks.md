# Tasks: AG-09 — Define the tool execution contract and scheduler

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | ~1,538 code (1,365 new + 173 modified) + ~10,000 planning docs |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR, 8 internal work-unit commits |
| Delivery strategy | `single-pr` with `size:exception` pre-authorized |
| Chain strategy | size-exception |
| Substrate files unchanged | 21 + go.mod + go.sum + Makefile + .golangci.yml (6th consecutive) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Commit | Focused test command | Runtime harness | Rollback boundary |
|------|------|--------|----------------------|-----------------|-------------------|
| 1 | Spec/design (no code) | `chore(agent): AG-09 spec+design` | `cd backend/agent && make test` | N/A | revert commit, no code yet |
| 2 | RED bites (no impl) | `test(agent): AG-09 RED bites` | `make test` (6 RED) | N/A | revert commit |
| 3 | Tool contract | `feat(agent): AG-09.1 contract` | `go test -run TestTool -race ./src/agent/` | N/A | revert file |
| 4 | Scheduler | `feat(agent): AG-09.2+09.3+09.4 scheduler` | `go test -run TestScheduler -race ./src/agent/` | N/A | revert file, wire-up absent |
| 5 | Turn wire-up + drainSink | `feat(agent): AG-09 wire-up + drainSink deadline` | `go test -run TestTurn -race ./src/agent/` | N/A | revert both files |
| 6 | End-to-end dispatch | `test(agent): AG-09 end-to-end tool dispatch` | `go test -run TestTurn_ToolDispatch -race ./src/agent/` | N/A | revert file |
| 7 | Substrate widening | `test(agent): AG-09 substrate-untouched widening` | `go test -run TestSubstrateUntouched -race ./src/agent/` | N/A | revert test file |
| 8 | Doc 0003 + spec reconcile | `docs(agent): AG-09 archive prep` | `make test` + `make lint` | N/A | revert commit |

## Phase 1: Spec + design (commit 1 — `chore`)

- [ ] 1.1 Commit `openspec/changes/cachicamas-agent-tool-scheduler/{proposal.md, exploration.md, specs/, design.md, tasks.md}` + `openspec/specs/agent-tool-scheduler/spec.md`. No code. `make test` green.

## Phase 2: Test RED bites (commit 2 — `test-red`)

> **Strict TDD ratchet** (AG-04/05/07/08 carries): all 6 bites MUST be RED BEFORE property GREEN.

- [ ] 2.1 Write 6 RED bites: S-TLS-002a (`tool_test.go` NEW, policy tag-strip); S-TLS-005a (unbounded fan-out); S-TLS-006a (start-at-rejoin); S-TLS-006b (start-before-end); S-TLS-010a (errgroup-shaped mock); S-TLS-011a (no-recover, panics under `-race`) — all in `scheduler_test.go` NEW except 002a. `make test` → exactly 6 RED. No production code.

## Phase 3: Core implementation (commits 3, 4, 5 — `feat`)

- [ ] 3.1 (commit 3) Create `backend/agent/src/agent/tool.go`: `Tool` interface (`Name()`, `EffectClass()`, `Run(ctx, args, policy) (Result, error)`); `EffectClass uint8` enum (3 members, `iota+1`, zero invalid, `String()`); `type PolicySlot any` (doc: "Layer 2 never reads"); `Result` struct (3 outcomes, `Content`, `Failure`, `CallID()`); ctor validations. ~180 LOC. `tool_test.go` for external-package posture. `go test -run TestTool -race` → S-TLS-002a bite GREEN.
- [ ] 3.2 (commit 4) Create `backend/agent/src/agent/scheduler.go`: `Registry interface{ Resolve(string) (Tool, bool) }`; `NewMapRegistry(map[string]Tool) Registry`; `Scheduler{MaxConcurrentReads int}` (default 8); `Schedule(ctx, calls, reg, runID, turnID, stamper, sink) []Result` with sub-methods (`executeCall`, `runDispatcher`, `scheduleReads`, `scheduleSerialized`, `recoverCall`). ~340 LOC. Source-guard tests scan `scheduler.go` for `policy.(*`/`.(PolicySlot)` + `errgroup` (D3a+D4a). `scheduler_test.go`: S-TLS-004/005/005a/006/006a/006b/008/009/010/010a/011/011a + unbuffered-sink test (AG-08 W1). `go test -run TestScheduler -race` → all GREEN.
- [ ] 3.3 (commit 5) Modify `backend/agent/src/agent/loop.go`: add `Tools Registry` field on `TurnOptions` (D9a; nil = typed `ExecutionFailure` per orphan); widen `translate()` switch on `ai.EventKindToolCallStart/Delta/End` (currently dropped `:468-472`); on `Completion{FinishReasonToolCalls}` call `Schedule` once between `provider.Stream` close and `finalize`; append `tool_results`. Modify `loop_test.go`'s `drainSink` (`loop_test.go:147`) with 1-line `select` deadline. Add `ordinalFromToolStart` to `agent_test_helpers_test.go`. `go test -run TestTurn -race` → GREEN.

## Phase 4: Test substrate + integration (commits 6, 7 — `test`)

- [ ] 4.1 (commit 6) Create `backend/agent/src/agenttest/scripted_tool.go` (in-memory `Tool`: configurable outcome, records start + policy byte-for-byte) + `scripted_tool_test.go`. Create `backend/agent/src/agent/loop_tool_dispatch_test.go` using AI-21 fake provider + scripted tools: streams `ToolCallStart/Delta/End` + `Completion{FinishReasonToolCalls}` → loop calls `Schedule` once → emits AG-05.2 events → finalizes. S-LSK-008 + S-LSK-008a bite. `go test -run TestTurn_ToolDispatch -race` → GREEN.
- [ ] 4.2 (commit 7) Modify `substrate_untouched_test.go` filter: widen to exclude `loop_tool_dispatch_test.go`. 21-path list unchanged. `AG09_BASE_REF` env-var + dynamic `git merge-base`. `go test -run TestSubstrateUntouched -race` → GREEN (NFR-TLS-003, 6th consecutive milestone).

## Phase 5: Verification + docs (commit 8 — `docs`)

- [ ] 5.1 (commit 8) Update `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` line 3: `"5 of 24"` → `"9 of 24"`. Add AG-09 to shipped list. Reconcile `openspec/specs/agent-loop-skeleton/spec.md`: MODIFIED R-LSK-001 (Tools field) + R-LSK-005 (coverage + S-LSK-008) + ADDED R-LSK-008 (one cycle, wording trap). `make test` + `make lint` (cache clean) both GREEN.

## Scenario count (AG-04 W9)

**11 charter → 12 spec + 6 bites = 18 total** + 1 cross-cut `S-LSK-008` + 1 bite `S-LSK-008a` = **20 total**. Restate identically in `apply-progress.md`.

## Verification + Hard rules

`cd backend/agent && make test` (all 12 scenarios + 6 bites + S-LSK-008/008a GREEN; AG-03 guards untouched). `make lint` (cache clean). `make build`. `make vuln-check`. `make test/cover` (loop.go ≥ 80%). `TestTurn_SubstrateUntouched` with `AG09_BASE_REF` fallback verifies 21-path byte-unchanged. Strict TDD · conventional commits · worktree `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag09` only · 21 substrate + go.mod + go.sum + Makefile + .golangci.yml byte-untouched · no new deps (no errgroup) · all 11 R-TLS covered · all 6 bites closed in named tests · AG-03 boundary guards untouched.

## Next step

`sdd-apply`. Auto mode → no user prompt. `size:exception` pre-authorized.