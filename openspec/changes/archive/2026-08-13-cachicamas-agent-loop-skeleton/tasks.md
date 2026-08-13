# Tasks: AG-07 — Build the one-turn walking skeleton

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 400–700 added |
| 400-line budget risk | Low |
| Chained PRs recommended | No (single PR) |
| Suggested split | Single PR — walking-skeleton scope is small |
| Delivery strategy | exception-ok (size:exception pre-authorized) |
| Chain strategy | size-exception (single PR accepted) |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Walking skeleton emit (R-LSK-001 / S-LSK-001..003 + bites) | PR #N (this PR) | `go test -race -run TestTurn_WalkingSkeleton ./backend/agent/src/agent/` | `make test` in `backend/agent/` | revert `loop.go` + `loop_test.go`; substrate byte-unchanged |
| 2 | Statelessness + reasoning (R-LSK-002..003 / S-LSK-004..005) | same PR | `go test -race -run TestTurn_TwoSequential ./backend/agent/src/agent/` | `make test` | same |
| 3 | Substrate + coverage gates (R-LSK-004..005 / S-LSK-006..007) | same PR | `git diff main -- backend/agent/src/agent/ \| grep -v '^loop' \| head` (must be empty) | `make test -cover` | same |

## Phase 1: Foundation — package skeleton + types

- [x] 1.1 Create `backend/agent/src/agent/loop.go` with `TurnOptions` struct (trivial fields per design), `Turn` function declared but no body yet.
- [x] 1.2 Create `backend/agent/src/agent/loop_test.go` with imports, package decl, placeholder helper `scriptTextResponse` around `agenttest.Script`.
- [x] 1.3 Verify `make test` + `make build` stay green — substrate untouched; this is a no-op compile check.

## Phase 2: Walking skeleton (R-LSK-001 — S-LSK-001, S-LSK-002, S-LSK-003 + bites)

- [x] 2.1 RED S-LSK-001 — `TestTurn_WalkingSkeleton_EmitsContractEventOrder`: scripted text response; consumer observes `run_start`→`turn_start`→`message_start_text`→deltas→`message_end_text`→`turn_end`→`run_end` in order; sink closed; returns `(msg, finish, nil)` where `finish ==` script's finish. RUN, expect RED.
- [x] 2.2 GREEN S-LSK-001 — implement `Turn` minimally: emit `run_start`/`turn_start`, translate provider text events to substrate brackets, capture finish from `ai.Completion`, emit `turn_end`/`run_end`, close sink, return. RUN, expect GREEN.
- [x] 2.3 RED S-LSK-002 — `TestTurn_ProviderStreamDrainedAndCtxRespected`: scripted text + non-cancelled `ctx`; consumer drain unblocks without stranded producer. RUN, expect RED.
- [x] 2.4 GREEN S-LSK-002 — drain provider channel fully before `turn_end`. RUN, expect GREEN.
- [x] 2.5 RED S-LSK-003a (drop-a-delta bite) — `TestTurn_OneSourceOfTruth_BiteDropDelta`: 3-delta sequence reconstructed WITH MIDDLE DELTA DROPPED must differ from loop's `msg`. RUN, expect RED.
- [x] 2.6 RED S-LSK-003b (double-a-delta bite) — same shape but DOUBLE the middle delta. RUN, expect RED.
- [x] 2.7 GREEN S-LSK-003 — `TestTurn_OneSourceOfTruth`: 3-delta sequence reconstructed from FULL emitted sequence equals loop's returned `msg`. RUN, expect GREEN (bites RED-recorded first).
- [x] 2.8 REFACTOR — extract `translateProviderEvent` switch helper + `turnAccumulator` struct (per-turn `LaneStamper` mint + sequence). RUN, expect GREEN.

## Phase 3: Statelessness + reasoning pass-through (R-LSK-002 + R-LSK-003)

- [x] 3.1 RED+GREEN S-LSK-004 — `TestTurn_TwoSequentialTurnsShareNothing`: invoke `Turn(...)` twice with fresh slices/opts/sink/script; second turn's events carry fresh per-stream ordering starting at 1, no shared `LaneStamper`. RUN, expect RED then GREEN.
- [x] 3.2 RED+GREEN S-LSK-005 — `TestTurn_ReasoningPassThroughByteExact`: interleaved reasoning+text with non-empty reasoning token; assert (a) reasoning vs text emitted as separate kinds, (b) assistant message's reasoning token byte-equals script's token. RUN, expect RED then GREEN (extend translation switch to reasoning kind).

## Phase 4: Substrate + coverage gates (R-LSK-004 + R-LSK-005)

- [x] 4.1 S-LSK-006 — `TestTurn_SubstrateUntouched`: `git diff main -- backend/agent/src/agent/ | grep -v -E '(loop\.go|loop_test\.go)'` empty; `git diff main -- backend/agent/go.mod backend/agent/go.sum` empty; every-kind-constructible guard still passes at 25 kinds. RUN, expect GREEN.
- [x] 4.2 S-LSK-007 — `TestTurn_CoverageGate`: `go test -cover -run TestTurn ./backend/agent/src/agent/` asserts `loop.go` coverage ≥ 80%. RUN, expect GREEN.
- [x] 4.3 Run `make test` (race), `make lint` (after `golangci-lint cache clean` per AG-04 precedent), `make build`, `make vuln-check` — all green.

## Phase 5: Cleanup + final gates

- [x] 5.1 Re-run all four Makefile gates (`make test`, `make lint`, `make build`, `make vuln-check`) — all green.
- [x] 5.2 Verify `go.mod`/`go.sum` byte-identical to main: `git diff main -- backend/agent/go.mod backend/agent/go.sum` empty.
- [x] 5.3 Verify AG-03 boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) still pass — AG-07 uses only stdlib + `src/ai` + `src/agenttest`.
- [x] 5.4 Verify scenario count stated identically across spec, tasks, apply-progress, verify-report: `5 charter → 7 spec + 2 bites = 9 total`.

## Notes for sdd-apply

- **Worktree discipline**: implementation runs in `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave2-ag07` on branch `feat/agent-layer2-wave2-ag07`, NOT the main checkout (Engram #2963 lesson).
- **Strict TDD ratchet**: every test goes RED-first; AG-05 W1 defense bites (S-LSK-003a/S-LSK-003b) MUST be RED-recorded BEFORE S-LSK-003 GREEN.
- **Commit structure**: one commit per phase. P1 = `feat(agent): loop package skeleton + TurnOptions (AG-07.1)`. P2 = `feat(agent): walking skeleton — emit + drain + one source of truth (AG-07.1)`. P3 = `feat(agent): statelessness + reasoning pass-through (AG-07.2)`. P4 = `feat(agent): substrate + coverage gates (AG-07 R-LSK-004 + R-LSK-005)`. P5 = `chore(agent): AG-07 final gate verification`. Total: 5 commits.
- **Forecast**: 400–700 lines added; under 1000-line budget. `size:exception` pre-authorized, single PR.
- **Substrate bet**: AG-07 modifies ONLY `loop.go` + `loop_test.go`. No edits to envelope/descriptor/validator/substrate. 4th consecutive "substrate untouched" milestone.