# Tasks: AG-08 — Add the pre-request hook seam

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | **350–600 added** (significantly smaller than AG-07's 1816) |
| 400-line budget risk | **Low** (forecast under 700 even at upper bound) |
| Chained PRs recommended | **No (single PR)** |
| Suggested split | N/A — single PR under pre-authorized `size:exception` |
| Delivery strategy | exception-ok (`size:exception` pre-authorized up to 1000-line budget; AG-08 forecast lands well under) |
| Chain strategy | size-exception (single PR accepted by user explicit instruction) |
| Files modified | 1 (`backend/agent/src/agent/loop.go`) |
| Files created | 1 (`backend/agent/src/agent/loop_hook_test.go`) + 1 fixture (`testdata/loop_skeleton_request.golden.json`) |
| Substrate files unchanged | 21 (5th consecutive "substrate untouched" milestone) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Hook seam + bite harness + identity default | PR #N (this PR) | `cd backend/agent && make test -run TestTurn_PreRequestHook` | `make test` in `backend/agent/` | revert `loop.go` + `loop_hook_test.go`; substrate byte-unchanged |
| 2 | Failure path + no-mutate + prefix stability | same PR | `cd backend/agent && make test -run 'TestTurn_(PreRequestHook\|PrefixStability)'` | `make test` | same |
| 3 | Unbuffered sink + substrate-untouched guard | same PR | `cd backend/agent && make test -run TestTurn_PreRequestHook_SubstrateUntouched` | `make test -cover` | same |

## Dependencies recap

- **Depends on**: AG-07 (`Turn`, `TurnOptions`, `buildLoopRequest`, `provider.Stream` call site — already merged at `93077c07` PR #167).
- **Parallel with**: AG-09 (tool execution contract + scheduler — no dependency).
- **Blocks**: AG-20 (the four-hook taxonomy registration surface widens the chain).
- **Layer 3 consumer**: doc 0004 CO-24 (cache-breakpoint placement — out of AG-08 scope).

## Phase 1: Spec + design committed (prerequisite)

- [x] 1.1 Commit `chore(agent): AG-08 spec + design committed` — `openspec/specs/agent-pre-request-hook/spec.md` + `openspec/changes/cachicamas-agent-pre-request-hook/{explore,proposal,design}.md` only. No code. **Type**: chore · **Closes**: prerequisite for tasks 2+ · **Verify**: `git log --oneline | head -5` shows spec + design commits.

## Phase 2: Hook seam + RED bites (AG-08.1 — R-PRH-002, R-PRH-005)

- [x] 2.1 RED bites — `TestTurn_PreRequestHook_NoSegmentBite` + `TestTurn_PreRequestHook_AddsSegmentBite` in `backend/agent/src/agent/loop_hook_test.go` (new file). Both bite hooks return `(req, nil)` unchanged; first asserts captured system region LACKS marker (RED: marker-absent), second asserts captured request IS `Equal` to skeleton's (RED but holds byte-equal — proves wiring distinguishes "no mutation" from "mutation applied"). ~80–120 lines. Commit `test(agent): AG-08.1 RED bites — S-PRH-001a + S-PRH-001b`. **Type**: test-red · **Closes**: S-PRH-001a, S-PRH-001b · **Verify**: `cd backend/agent && make test -run 'TestTurn_PreRequestHook_(NoSegment|AddsSegment)Bite'` — both RED with helpful messages.
- [x] 2.2 GREEN seam — add `PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)` field to `TurnOptions` (loop.go:46-51); add `applyPreRequestHook(ctx, req, hook) (ai.Request, error)` helper (~10 lines); insert 8-line branch in `Turn` between line 132 and line 140 (mirror pre-stream-failure on hook error: `closeSink(sink)`, return `(ai.Message{}, 0, typedErr)` reusing `ai.PreStreamFailure` with `FailureCategoryUnsupportedCapability` for hook-attribution, `StatusClass: 4`). ~20–40 lines on loop.go (457 → ~480). Commit `feat(agent): AG-08.1 hook seam — TurnOptions.PreRequestHook + applyPreRequestHook + Turn branch`. **Type**: feat · **Closes**: R-PRH-001 (surface), R-PRH-002 (hook invoked) · **Verify**: `cd backend/agent && make test -run 'TestTurn_PreRequestHook_(NoSegment|AddsSegment)Bite'` — both GREEN; `make lint` clean.
- [x] 2.3 GREEN happy path + identity default — `TestTurn_PreRequestHook_AddsSystemSegment` (hook returns `req.With(ai.WithSystemInstruction(<instr + segment>))`; assert captured request carries added segment; closes R-PRH-002 / S-PRH-001) + `TestTurn_PreRequestHook_NilIdentity` (zero-value `TurnOptions`; assert captured request `Equal` to fixture of AG-07 skeleton's known-good output; closes R-PRH-005 / S-PRH-002). ~80–120 lines + ~50-line fixture `testdata/loop_skeleton_request.golden.json` (committed). Commit `test(agent): AG-08.1 S-PRH-001 + S-PRH-002 (identity default)`. **Type**: test-green · **Closes**: S-PRH-001, S-PRH-002 · **Verify**: `cd backend/agent && make test` — both GREEN under `-race`.

## Phase 3: Failure path + no-mutate (AG-08.1 — R-PRH-003, R-PRH-004)

- [x] 3.1 GREEN failure + no-mutate — `TestTurn_PreRequestHook_FailureAbortsBeforeStream` (hook returns `(ai.Request{}, errors.New("hook boom"))`; assert `len(provider.Requests()) == 0`, sink drains unblocked, returned error wraps `*ai.PreStreamFailure`; closes R-PRH-003 / S-PRH-003) + `TestTurn_PreRequestHook_CannotMutateInput` (hook mutates slice from accessor: `msgs := req.Messages(); msgs[0] = mutated`; assert captured request `Equal` to skeleton's via `ai.Request.Equal` — substrate R-REX-001 holds; closes R-PRH-004 / S-PRH-004). ~60–90 lines. Commit `test(agent): AG-08.1 S-PRH-003 + S-PRH-004 (failure + no-mutate)`. **Type**: test-green · **Closes**: S-PRH-003, S-PRH-004, R-PRH-003, R-PRH-004 · **Verify**: `cd backend/agent && make test` — both GREEN.

## Phase 4: Prefix stability + determinism (AG-08.2 — R-PRH-006, R-PRH-007)

- [x] 4.1 GREEN prefix-stability + determinism — `TestTurn_PrefixStability_ByteStableToolsSystem` (two consecutive `Turn` calls, same system + same tools + same hook, second transcript = first + 1 appended message; assert `tools` and `system` regions `Equal` byte-equal across turns, `CacheBoundaries()` returns same cascade order, message region grew by 1, first N `Message.Equal`; closes R-PRH-006 / S-PRH-005) + `TestTurn_PrefixStability_DeterministicHook` (hook called N times with byte-equal inputs, assert byte-equal outputs; closes R-PRH-007 / S-PRH-006). ~80–120 lines. Commit `test(agent): AG-08.2 S-PRH-005 + S-PRH-006 (prefix stability + determinism)`. **Type**: test-green · **Closes**: S-PRH-005, S-PRH-006, R-PRH-006, R-PRH-007 · **Verify**: `cd backend/agent && make test` — both GREEN.

## Phase 5: Back-pressure carry + substrate guard + archive

- [x] 5.1 GREEN unbuffered sink + substrate-untouched — `TestTurn_PreRequestHook_UnbufferedSink` (unbuffered `sink` = `make(chan *Event)` + concurrent consumer goroutine + `runtime.NumGoroutine()` baseline before/after; AG-07 W1 carry; closes S-PRH-007) + `TestTurn_PreRequestHook_SubstrateUntouched` (uses `AG08_BASE_REF` env-var fallback shipped in AG-07 PR #167 W3 fix + dynamic merge-base; asserts 21 substrate files unchanged + AG-03 guards still green at 25 kinds; closes NFR-PRH-003). ~100–150 lines. Commit `test(agent): AG-08.2 S-PRH-007 (unbuffered sink) + substrate-untouched guard`. **Type**: test-green · **Closes**: S-PRH-007 + NFR-PRH-003 substrate guard · **Verify**: `cd backend/agent && make test` — both GREEN.
- [ ] 5.2 Archive + PR — after `sdd-verify` completes, commit verify-report and open PR. Commit `chore(agent): AG-08 archive — proposal/spec/design/tasks/verify-report + PR open`. **Type**: chore · **Closes**: milestone · **Verify**: PR opened with title `feat(agent): AG-08 — Add the pre-request hook seam`.

## Commit plan summary

| # | Commit | Type | Approx lines | Closes |
|---|---|---|---|---|
| 1 | `chore spec+design` | chore | 0 | prerequisite |
| 2 | `test RED bites` | test-red | 80–120 | S-PRH-001a/b |
| 3 | `feat seam` | feat | 20–40 | R-PRH-001/002 |
| 4 | `test identity+happy` | test-green | 80–120 + ~50 fixture | S-PRH-001/002 |
| 5 | `test failure+no-mutate` | test-green | 60–90 | S-PRH-003/004 |
| 6 | `test prefix-stability` | test-green | 80–120 | S-PRH-005/006 |
| 7 | `test unbuffered+substrate` | test-green | 100–150 | S-PRH-007 + NFR-PRH-003 |
| 8 | `chore archive + PR` | chore | 0 | milestone close |
| **Total** | | | **~420–690 added** | |

Each commit ≤ 400 lines; total well under the 1000-line `size:exception` pre-authorized budget.

## Verification approach

- `cd backend/agent && make test` — full `-race -v ./...` run; all 9 scenarios + bites green; AG-03 boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) stay green untouched.
- `cd backend/agent && make lint` — `golangci-lint` clean (`golangci-lint cache clean` first per AG-04 precedent).
- `cd backend/agent && make build` — compile `./bin/database_administrator` (or AG-08's binary) clean.
- `cd backend/agent && make vuln-check` — no vulnerabilities.
- `cd backend/agent && make test/cover` — `loop.go` ≥ 80% line coverage (AG-07 hit 85.89%; AG-08's +20–40 lines preserve margin; AG-04 W8 carry).
- Substrate-untouched check: NFR-PRH-003 verified by `TestTurn_PreRequestHook_SubstrateUntouched` (`AG08_BASE_REF` env-var fallback + dynamic merge-base).

## Acceptance criteria

- All 7 requirements R-PRH-001..007 closed (R-PRH-001 surface; R-PRH-002 invocation; R-PRH-003 failure abort; R-PRH-004 no-mutate; R-PRH-005 identity default; R-PRH-006 prefix stability; R-PRH-007 hook determinism).
- All 9 spec scenarios S-PRH-001..008 (incl. bites 001a/001b) GREEN under `-race`.
- All 4 NFRs satisfied (NFR-PRH-001 external-package posture; NFR-PRH-002 determinism + race + no-ambient-authority; NFR-PRH-003 substrate byte-unchanged; NFR-PRH-004 single PR under pre-authorized `size:exception`).
- `loop.go` coverage ≥ 80% (AG-04 W8 / AG-07 W2 carry).
- 21 substrate files unchanged (verified by `TestTurn_PreRequestHook_SubstrateUntouched`).
- Bite-first ordering enforced: task 2.1 RED before task 2.3 GREEN, with both RED bites recorded in test comments (AG-05 W1 carry).
- AG-07 W1 closed by S-PRH-007 (unbuffered sink).
- AG-07 W6 carried forward as NFR-PRH-001 (every test in `package agent_test`).
- AG-07 W3 carried forward — substrate-untouched test uses env-var fallback (shipped in AG-07 PR #167).
- `6 charter → 7 spec + 2 bites = 9 total` scenario count stated identically across proposal, spec, tasks, apply-progress, verify-report.

## Out of scope

- The other three hook points (AG-20 widens to chain composition).
- Concrete cache-breakpoint placement (Layer 3 wiring — doc 0004 CO-24).
- Translation interface changes (AG-07 SUGG 4 — parked; AG-13 may re-introduce).
- Tools / permission / retry / cost / context-check (AG-09, AG-10, AG-11, AG-15, AG-16).
- Append-only history discipline (AG-12.1).
- Value-form `Harness` (AG-13).
- `go.mod` / `go.sum` edits (no new deps).

## Carry-forwards summary

| Source | Finding | AG-08 mitigation |
|---|---|---|
| **AG-07 W1** | Every AG-07 test used buffered sink; back-pressure path unproven | `S-PRH-007` adds unbuffered-sink test with `runtime.NumGoroutine()` baseline |
| **AG-07 W6** | External-package test posture | NFR-PRH-001 — every AG-08 test in `package agent_test` |
| **AG-07 W3** | `TestTurn_SubstrateUntouched` hard-coded ref went stale on merge | AG-08 substrate-untouched test uses `AG08_BASE_REF` env-var fallback shipped in AG-07 PR #167 |
| **AG-07 W2** | `TestTurn_CoverageGate` is a skip marker; real gate is `make test/cover` | Re-stated as `make test/cover` gate; forecast ≥ 80% on `loop.go` |
| **AG-07 W4** | `mintLoopMessageID` discards two errors (latent) | AG-08 doesn't touch; carries to AG-23 |
| **AG-07 SUGG 4** | `translate()` could become method on `providerEventTranslator` | Hook wrap is not translation path; SUGG 4 stays parked |
| **AG-05 W1** | Vacuous reconstruction helper | Bite pattern on R-PRH-002 (`S-PRH-001a`/`S-PRH-001b`) — both RED before GREEN |
| **AG-04 W9** | Scenario count drift | State identically: `6 charter → 7 spec + 2 bites = 9 total` |
| **AG-04 W8** | Coverage ≥ 80% on `loop.go` | Re-stated as acceptance criterion |

## Notes for sdd-apply

- **Worktree discipline**: implementation runs in `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag08` on branch `feat/agent-layer2-wave2-ag08`, NOT the main checkout (AG-07 Engram lesson).
- **Strict TDD ratchet**: every AG-08 test goes RED-first; AG-05 W1 defense bites (S-PRH-001a/S-PRH-001b) MUST be RED-recorded BEFORE task 2.3 GREEN. Mirrors AG-07 S-LSK-003a/S-LSK-003b discipline.
- **Commit structure**: 8 commits per design.md commit plan. Each ≤ 400 lines; total ~420–690 added (well under 1000-line `size:exception` budget).
- **Open questions resolved by `sdd-tasks`**: (1) Test file → NEW `loop_hook_test.go` (mirrors AG-07's grep-discoverability principle; AG-08 tests are cohesive and easier to grep as a unit); (2) Hook failure `FailureCategory` → `FailureCategoryUnsupportedCapability` (hook-attributing; not provider-auth) per design.md D2 + R1 mitigation.
- **Substrate bet**: AG-08 modifies ONLY `loop.go` + creates `loop_hook_test.go`. No edits to envelope/descriptor/validator/substrate. **5th** consecutive "substrate untouched" milestone.
- **AGENTS.md discipline** (cachicamas): use Makefile targets (`make test | lint | build | vuln-check`), no `Co-Authored-By` trailer, conventional commits only, agent module is layered (not hexagonal), no edits to `go.mod`/`go.sum`/`Makefile`/`.golangci.yml`/AG-03 boundary guards.
- **Forecast**: 350–600 lines added; well under 1000-line budget. `size:exception` pre-authorized, single PR.

## Next step

Launch `sdd-apply` next.