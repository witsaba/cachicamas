# Archive Report: AG-16 — Emit cost and usage events

> **Change**: `cachicamas-agent-cost-events` · **Milestone**: AG-16 (Layer 2 Wave 3, **16 of 24**)
> **Branch**: `feat/agent-layer2-wave3-ag16` · **Base**: `main@09bb30e1`
> **Closes**: **G10**'s Layer 2 half (R-16) — Layer 1 counts; Layer 2 reports
> **Self-verification at apply close**: full suite green under `-race`, `make lint` 0 issues, `make build` clean, `make vuln-check` clean. **`sdd-verify` has not yet run independently** — this report records apply-phase evidence, not a separate verify pass, and does not claim one.
> **Delivery**: single PR, `size:exception` pre-authorized by the user, 1000-line review budget
> **Receipt-driven development**: not invoked in this session

## What shipped

Every non-aborted turn now reports its own token spend, and the harness reports the run's cumulative spend on every outcome.

- **`cost_turn` emission** — `finalize()` (`loop.go`) emits a turn's `cost_turn` as its first emission, inside the bracket it opened and before `turn_end`, on both `Turn` call paths and on the no-`Completion` fallback. An aborted close (mid-stream fatal, any of the three pre-stream failures, cancellation mid-turn) emits none — the iff's other half.
- **Per-figure presence** — `costPresence`, stored beside `CostFigures` (never inside it, keeping `S-APE-083`'s reflection pin untouched), read through ten new paired accessors on `CostTurn`/`CostSession` mirroring `ai.TokenCount.Count() (int64, bool)`. `(0, absent)` and `(0, reported)` are observably distinct; a whole-record flag was proven to ship the forbidden "invented zero" defect by a reverted bite (`S-CST-020`).
- **Run-scoped cumulative** — `costAccumulator` in the harness's forwarder intercepts every emitted `cost_turn` into a local (never a `Harness` field) running total: absence is the additive identity, presence is OR, a reported nought survives. Retry-inclusive by construction — no retry-awareness anywhere in the accumulator, proven by a reverted bite that skips a contribution (`S-CST-021`).
- **Estimate/final protocol** — `cost_session(Estimate)` before each continued logical turn (the ToolCalls/PauseTurn arm and the steered-message arm); `cost_session(Final)` immediately before every run close — success, failure, and both cancellation wind-downs (interrupt, shutdown) — correcting every earlier estimate. A reverted bite mislabelling the terminal event proved the label assertion is real, not vacuous (`S-CST-022`).
- **Non-happy closes report real spend** — `failRun` and `windDownRun` both widened with an unexported `total costAccumulator` parameter and their own best-effort `cost_session(Final)` emission; a failed or cancelled run reports the real spend of the turns that closed before the failure/signal, never five invented zeros for the one that did not.
- **`cost_events.go`'s AG-16-only bounded release** — exactly three things: the presence field, its ten accessors, and the two label-axis doc comments. `CostFigures` and `cost_events_test.go` stay byte-unchanged, verified by empty `git diff` against `main`, not by assertion.

### A genuine ID collision found and fixed during spec promotion

The change's own delta for `agent-cancellation-tree` minted `S-CAN-012` for R-CAN-002's new AG-16 scenario and `S-CAN-013` for R-CAN-005's — but `S-CAN-012` was already allocated, since AG-14, to `R-CAN-006`'s pre-existing bite in the shipped spec. Promoting literally would have shipped two different scenarios both claiming `S-CAN-012` in the same archived file. Renumbered to `S-CAN-013` and `S-CAN-014` (the next free sequential IDs) during promotion (task 8.1); the delta source is left as originally authored, and the correction is recorded in the promoted spec's own header and scenario text, in `tasks.md`, and in `apply-progress.md`.

### Two design-artifact citation defects found and worked around (not corrected in the spec prose)

1. `design.md` and the `agent-loop-skeleton` delta's own `S-LSK-009` scenario annotation cite `loop_test.go:1152-1165` as the "tool dispatch" wire-up test; that location is actually `TestTurn_ReasoningPassThroughByteExact` (S-LSK-005). The real tool-dispatch test lives in `loop_tool_dispatch_test.go` and uses kind-filtered counting, unaffected by AG-16 either way. The implementation fix (insert `EventKindCostTurn` before `EventKindTurnEnd` at the cited, verified-correct line range) is unaffected by the mislabel.
2. `design.md`/`tasks.md` cite `windDownRun` as having 2 call sites; it actually has 3 (the iteration-boundary cancellation check, missed by the citation). Widened at all three — Go's compiler made an undercount impossible to ship silently.

### Two promoted-spec text defects corrected after verification

`sdd-verify` established that defect 1 above was **not** confined to the planning artifacts: the same mislabelled citation had been copied into promoted normative text. Both defects below were corrected in the promoted specs before the pull request opened. The archived delta sources are left as originally authored, so this report is the record of the divergence.

1. **A false amendment claim in normative text.** `openspec/specs/agent-loop-skeleton/spec.md` — S-LSK-009's AG-16 note ended by claiming "the closed-order assertion at `loop_test.go:1152-1165` gains the kind at that position as an enumerated amendment". The line range is accurate but belongs to `S-LSK-005`'s reasoning test; S-LSK-009's own test, `loop_tool_dispatch_test.go`, has an empty diff against `origin/main` because it asserts kind-filtered counts rather than a closed sequence. The note's substantive ordering claim (the `cost_turn` lands after the rejoin-ordered tool events and before `turn_end`) is true and tested and was kept; the false attribution clause was replaced with the correct one.

2. **`R-CST-002` contradicted a file it pins.** Its closing sentence read "A count MUST NOT be readable without its discriminator", while `CostTurn.Figures()`/`CostSession.Figures()` remain public returning five bare `uint64`s and `cost_events_test.go:60-67` reads exactly that way — a file `NFR-CST-004` pins byte-unchanged. Both obligations could not hold, and no scenario asserted the clause, so nothing failed. The concrete risk was a later milestone reading it literally, deleting `Figures()`, and breaking a pinned file. Reworded in both `agent-cost-events` and `agent-protocol-events` to state what is actually true and enforced: the paired accessors are the **required** path for judging presence, `Figures()` must not be read to infer absence, and it must not be removed to satisfy the clause.

## Commits

| SHA | Subject |
|---|---|
| `4cf845b3` | `docs(openspec): AG-16 change proposal, spec, design, tasks` |
| `dc4f1a54` | `feat(agent): AG-16 phase 0 — usage capture, conversion, presence discriminator` |
| `2e35ba78` | `feat(agent): AG-16 phase 1+2 — cost_turn emission and blast-radius remediation` |
| `8b3b548d` | `feat(agent): AG-16 phase 3 — cumulative accumulator and estimate/final protocol` |
| `bb167957` | `test(agent): AG-16 phase 4 — non-happy run close coverage` |
| `449566a0` | `test(agent): AG-16 phase 5 — scope fence and substrate verification` |
| `14148e1e` | `docs(openspec): AG-16 apply-progress and tasks checklist through phase 6` |
| `6f11a61d` | `docs(0003): AG-16 reconciliation note, status line, and milestone counter` |
| `79f8cea6` | `chore(openspec): AG-16 spec promotion — six deltas merged, new agent-cost-events capability` |
| *(this commit)* | `chore(openspec): AG-16 archive — promote and close the cycle` |

## Capabilities promoted

| Capability | Kind | What promoted |
|---|---|---|
| `agent-cost-events` | **NEW** | `R-CST-001`…`007`, `S-CST-001`…`014` (incl. 3 bites `S-CST-020`/`021`/`022`). 233 lines. |
| `agent-protocol-events` | MODIFIED | `R-APE-004` per-figure presence discriminator; `R-APE-005` label axis re-scoped run-scoped; `S-APE-085`/`086` new |
| `agent-run-driver` | MODIFIED | `R-RUN-003` restated with cost events on the existing lane (`S-RUN-022`); `R-RUN-011` the failed run's real spend (`S-RUN-104`); deferred row closed |
| `agent-retry-failover` | back-annotation | "cost accounting for retried attempts" closed by dissolving the question — sum-over-emitted-events is retry-inclusive by construction, no retry-awareness needed |
| `agent-loop-skeleton` | MODIFIED | `R-LSK-001` nil-path `cost_turn` amendment (`S-LSK-001` amended, `S-LSK-024` new); `R-LSK-004`'s AG-16-only bounded `cost_events.go` release (`S-LSK-025`/`026`); **header allocated range extended to `S-LSK-026`** |
| `agent-turn-termination` | back-annotation | `NFR-ATT-004`/`R-ATT-007` confirmed held, not amended — the usage reaches the emission site through the turn's own accumulator, never a returned value |
| `agent-cancellation-tree` | MODIFIED | `R-CAN-002`/`R-CAN-005` wind-down order amended to insert `cost_session(Final)` before the run-close (`S-CAN-013`/`014`, renumbered — see above) |

## Verification at close

Re-run directly by this apply phase, not inherited from an unverified claim:

- `go test -race ./...` from `backend/agent/` — **12/12 packages `ok`**, zero FAIL (full module, not just `src/agent`)
- `golangci-lint cache clean && make lint` — **`0 issues`**; `go vet` clean; `make build` clean
- `gofmt -l` empty on every file this change touches (10 files); the 14 pre-existing gofmt-dirty files elsewhere in the package (`cost_events_test.go`, `event_registry_test.go`, `permission_events.go`, `scheduler.go`, `tool.go`, etc.) are confirmed byte-identical to `main@09bb30e1` and were never touched — pre-existing repo debt, several of them substrate files this change is forbidden to edit
- `govulncheck ./...` — **"No vulnerabilities found."**
- `src/agent` package coverage 79.0%; `finalize()` (the emission site) 100%; `Harness.Run` 94.2%
- Byte-unchanged vs `main@09bb30e1`: `cost_events_test.go`, `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `turn_events.go`, `failure.go`, `run_events.go`, `history.go`, `doc.go`, `doc_contract_guard_test.go`, and **all of `backend/agent/src/ai/`** and `go.mod`/`go.sum` — confirmed by empty `git diff`, not by assertion
- The non-test-file diff against `main` is exactly `{cost_events.go, cost_usage.go, harness.go, loop.go}` — matches `S-LSK-025` exactly
- Both substrate filters (`filterOutLoopFiles`, `filterOutLoopHookFiles`) carry an identical 46-entry set; `cost_events_test.go` and `stream_check_test.go` confirmed absent from both
- Every archived file's git blob hash confirmed identical before and after `git mv` — cryptographic proof against the known truncation-into-placeholder risk, not a visual diff
- All three bites (`S-CST-020`, `S-CST-021`, `S-CST-022`) RED-recorded with failing output, then reverted; `git diff --stat` confirmed each revert clean
- Base checkout at `/Users/braejan/workspace/witsaba/repositories/cachicamas` never touched; all work confined to the `ag16` worktree throughout

## Carried forward as follow-ups (non-blocking)

1. **The two citation defects above** (S-LSK-009's mislabelled location; `windDownRun`'s undercounted call sites) are worked around in the implementation but not corrected in the SDD planning prose (`design.md`, the `agent-loop-skeleton` delta's own scenario text) — out of scope for `sdd-apply`, worth a follow-up spec correction pass.
2. **`CostLabel`'s own type-level doc comment** (`cost_events.go:85-89`) still reads the pre-AG-16 wire-level framing ("distinguishes figures emitted before the stream's final usage update..."). Deliberate: the bounded `R-LSK-004` release for AG-16 names only the two constant doc comments at `:96-103` as an allowed edit, "and nothing else." A future milestone's own recorded release could restate it if that becomes load-bearing.
3. **`errorProvider` precedent citation drift**: `agent-retry-failover/spec.md` and `design.md` cite `loop_test.go:1408-1421` for the retry fixture; the actual fixture used and reused correctly (`preStreamFailingProvider` in `retry_policy_test.go`) lives elsewhere. Not load-bearing for any implementation decision — identified and used correctly by reading its own doc comment and call sites directly, not by trusting the citation.
4. AG-19's parent-scoped cost aggregation over delegated runs remains open, as this milestone's own charter and every promoted delta record explicitly.

## State at close

Layer 2 stands at **16 of 24**. AG-17 (compaction check at the turn boundary) is next.
