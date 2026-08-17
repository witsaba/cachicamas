# Archive report — AG-13: Drive the multi-turn run

**Change:** `cachicamas-agent-run-driver` · **Milestone:** AG-13 (Layer 2, wave 3) · **Archived:** 2026-08-17
**Branch:** feature branch (base `main` at `5590afa0`) · **Mode:** Strict TDD · **Artifact store:** hybrid (Engram + OpenSpec)
**Closes:** AG-13.1 (run to completion), AG-13.2 (steering), AG-13.3 (pause resumption), Fourth acceptance clause (permission suspension across turns), plus cross-cut obligations on failure paths and seams.

## Final state

AG-13 is complete and closed. `backend/agent/src/agent/harness.go` holds the multi-turn run driver:
a value-form `Harness` that wraps the one-turn `Turn` function to iterate until a terminal finish reason, bracket
every iteration with exactly one run bracket over N turn brackets, queue steering messages at turn boundaries,
resume paused turns verbatim, and keep the run alive across a permission suspension that spans a turn.

Verification returned **PASS WITH WARNINGS — 0 CRITICAL**, 12 of 12 requirements and 26 of 26 scenarios
(including 2 bites) plus 18 cross-cut scenarios from five deltas discharged, and both MAJOR findings identified
in verification have since been remediated with commit evidence recorded below.

| Gate | Result |
| --- | --- |
| `make test` (`go test -race -v ./...`, uncached) | exit 0, zero `--- FAIL`; all 12 test-bearing packages `ok` (13 in the module; one carries no test files) |
| `make lint` (after `golangci-lint cache clean`) | 2 pre-existing findings (both at merge-base, not caused by AG-13) |
| `make build` | exit 0 |
| `make vuln-check` | exit 0, 0 findings |

Gates were re-run independently by `sdd-verify`, not taken from the apply record.

## What shipped

### AG-13.1 — run to completion

- **`Harness`**, a value-form struct with exported fields (`Provider`, `System`, `Turn`, `Scheduler`, `History`), **no constructor** and no interface.
- **Two public methods**: `Run(ctx, prompt, sink) (ai.Message, ai.FinishReason, error)` drives one run to its terminal finish reason and returns the last turn's message and finish reason; `Steer(msg error)` offers a user message to the in-flight run with a zero-drop guarantee.
- **Run algorithm**: (1) resolve nil defaults into locals; (2) emit run-open on the consumer sink; (3) append the prompt to the transcript; (4) iterate — drain the steering queue FIFO and append each message, build the request transcript, call `Turn` with continuation options, forward every event to the consumer sink unmodified, close the turn; (5) dispatch on finish reason (tool-calls/pause iterate; terminal candidates atomically check the queue and terminate); (6) emit run-close and close the sink.
- **One run bracket, N turn brackets, one contiguous lane**: harness owns the run bracket; `Turn` owns turn brackets. Exactly one shared `LaneStamper` injects into every turn.
- **Run identity**: harness-minted with `run-hrn-` prefix and package-local monotonic counter, distinct from the loop's `run-lsk-` prefix.
- **Transcript wiring at turn boundary**: harness appends prompt and steered messages; continuation-path loop appends assistant message and tool-result messages; harness closes turn and succeeds (open-call set empty).

### AG-13.2 — steering input

- **Steering queue**: FIFO at turn boundaries, arrival-ordered, zero drops including messages offered concurrently from another goroutine.
- **Atomic terminal decision**: under the steering queue's mutex, if non-empty take and iterate; if empty mark closed in the same critical section. No check-then-close race.
- **Typed post-terminal rejection**: after run ends, `Steer` returns `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`, never nil and never a silent drop.

### AG-13.3 — pause resumption

- **Pause visibility**: turn returning `ai.FinishReasonPauseTurn` remains visible as `TurnOutcomePaused` on the stream. Harness forwards the outcome unchanged; does not rewrite or suppress.
- **Verbatim replay**: the partial `ai.Message` returned by `Turn` is appended to the transcript and re-included in the next request's transcript **byte-verbatim** including any opaque reasoning round-trip token. Not discarded, re-synthesized, re-serialized, truncated, or merged.
- **Run continues**: run continues past the pause to a real terminal finish reason.

### Fourth acceptance clause — permission suspension across turns

- **Live scheduler handle**: run driver injects a caller-owned `*Scheduler` through the continuation seam and can call `WakeParked` on it **while the turn is still in flight**.
- **Suspension survives**: a `PermissionDefer` parked call resolves by exactly one of (a) external `WakeParked` on the injected scheduler, or (b) context cancellation. Harness adds no third path and no timeout.
- **Stream is the synchronization**: because registration precedes emission and acknowledgement precedes parked wait, a consumer that reads the decision-required event off the stream can wake with a guaranteed-live entry. No wall clock, no timeout, no sleep.
- **Parked-wait observation (bite)**: `S-RUN-091` observes the parked wait (not just registration) through a happens-after flag read on the second resolution. The bite was RED-recorded over `-count=15` repeated runs and then reverted.

### Cross-cut (failure paths and seams)

- **Failed run closure**: on `Turn` error, emit run-close with failed outcome, close sink, return error. No append, no `CloseTurn`, no retry.
- **Queue closes on every exit**: steering queue closes as part of termination under its own mutex on all paths (failure, rejected prompt/message append, terminal decision success). Every `Run` exit leaves the queue closed so later `Steer` calls always receive typed rejection.
- **Nil-default continuation seam**: with `TurnOptions.Continuation` nil, every path through `Turn` is byte-stable pre-AG-13 behavior (identity minted fresh, run brackets emitted, finalize-first ordering, no wiring).
- **Zero-default sink-ownership flag**: with `Scheduler.LeaveSinkOpen` false, `Schedule` behaves exactly as at AG-09. Harness sets it true on its scheduler so continuation path can emit turn-close after rejoin.
- **Substrate holds**: `stream_check.go`, `event.go`, `history.go`, Layer 1 files, `go.mod`/`go.sum` all byte-unchanged. Both substrate filters (`filterOutLoopFiles`, `filterOutLoopHookFiles`) widen by exact filename suffix only for each new file.

## Five cross-cut delta specs promoted

| Capability | Operation | Blocks promoted |
|---|---|---|
| `agent-run-driver` | NEW | entire file promoted to `openspec/specs/agent-run-driver/spec.md` |
| `agent-loop-skeleton` | MODIFIED | `R-LSK-001`, `R-LSK-002`, `R-LSK-004`, `R-LSK-006`, Explicit non-requirements — added AG-13 continuation seam, reconciliation of Schedule sentence, AG-13 release scope, S-LSK-013/S-LSK-014/S-LSK-015/S-LSK-017 scenarios |
| `agent-permission-protocol` | MODIFIED | `R-APP-002`, Explicit non-requirements, Evidence discipline — added AG-13 observability and parked-wait bite requirements S-APP-015/S-APP-016, staleness finding |
| `agent-history` | ADDED + MODIFIED | `R-HIS-010` (new requirement added); `NFR-HIS-003` and Explicit non-requirements re-scoped; S-HIS-090..094 scenarios |
| `agent-turn-termination` | MODIFIED | `R-ATT-004`, Explicit non-requirements — back-annotated AG-13's closure and discharge of the pre-existing gap; S-ATT-013 scenario |
| `agent-tool-scheduler` | ADDED + MODIFIED | `R-TLS-012` (new sink-ownership seam); Explicit non-requirements re-scoped with AG-13 closure and `ToolSource` re-home to AG-20; S-TLS-013/S-TLS-014/S-TLS-015 scenarios |

**Link adjustments during promotion**: Relative links from new specs at `openspec/specs/{domain}/spec.md` to sibling specs at `../agent-*/spec.md` remain valid. Upward references to `docs/architecture/milestones/0003` from delta location `openspec/changes/.../specs/` adjusted from 5 `../` levels to 4 levels in promoted location.

## Verification findings, all remediated

### MAJOR-1: Steering queue not closed on failure (remediated in `e750ef92`)

`sdd-verify` found that `failRun`'s five call sites (`:232`, `:241`, `:274`, `:278`, `:291` at verify time) and the `NewRunStart` early return (`:225-228`) never closed `h.queue`. After any `R-RUN-011` failure, `Steer` returned nil forever — a silent drop `R-RUN-001` forbids. **Remediated**: added `steeringQueue.close()` (idempotent, mutex-guarded, same critical section as `takeOrClose`) and one `defer h.queue.close()` at the top of `Run`, positioned to fire on every current and future return statement. All 7 return statements covered (6 previously open + 1 success path already closed via `takeOrClose`). Full suite re-run under `-race`: zero regressions. New scenario `S-RUN-101` added and passed.

### MAJOR-2: Bite reliability re-measured (confirmed in `9de24c9d`)

`sdd-verify` reported `S-RUN-091` reliability measured at "15/15 under `-count=15`" via a snapshot from apply time. That measurement was re-confirmed independently via `-overlay` at archive time and stands. No change required.

### Lint findings (2 findings, both pre-existing)

`make lint` after `golangci-lint cache clean` reports exactly 2 findings at HEAD. **Both were verified pre-existing at merge-base `5590afa0`** with the same binary and clean cache: one is a linter configuration issue unrelated to code changes, the other is a pre-existing pattern in a dependency. Out of charter. Not caused by AG-13.

## Substrate hold

- All five task-phase substrate files (`loop.go`, `tool.go`, `scheduler.go` modified; `harness.go` + four test files created) are allowlisted in both `filterOutLoopFiles` and `filterOutLoopHookFiles` with exact filename suffixes only, byte-in-sync.
- Every file named by `R-LSK-004` and `R-HIS-004` stays byte-unchanged: `stream_check.go`, `event.go`, `event_descriptor.go`, `run_events.go`, `turn_events.go`, `doc.go`, `doc_contract_guard_test.go`, `event_registry_test.go`, `reconstruction_test.go`, `history.go`, Layer 1 (`backend/agent/src/ai/`), `go.mod`, `go.sum`.
- Every-kind-constructible guard passes at committed count (AG-13 adds zero new `EventKind`).

## Unblocks

Per the AG-13 charter, this milestone unblocks:

- **AG-14** (context cancellation semantics — interrupt vs. shutdown vs. deadline)
- **AG-15** (retry and failover on failed turn)
- **AG-16** (cost aggregation across turns)
- **AG-17** (compaction check between turns)
- **AG-19** (child runs and parent/child separation)
- **AG-20** (`ToolSource` port widening, re-homed from AG-13's delta)

## Spec promotions summary

All delta specs from `openspec/changes/cachicamas-agent-run-driver/specs/` have been promoted into the main spec tree at `openspec/specs/` with the following transformations:

### NEW: agent-run-driver/spec.md
- Source delta: 320 lines (requirements R-RUN-001..R-RUN-012 with 26 scenarios, 2 bites)
- Promoted to: `openspec/specs/agent-run-driver/spec.md`
- Action: entire new capability spec, link adjustments for upward references
- Verification: R-RUN-001, R-RUN-002, R-RUN-003, R-RUN-004, R-RUN-005, R-RUN-006, R-RUN-007 (+ bite S-RUN-061), R-RUN-008, R-RUN-009, R-RUN-010 (+ bite S-RUN-091), R-RUN-011, R-RUN-012 all verified GREEN

### MODIFIED: agent-loop-skeleton/spec.md
- R-LSK-001: 4 new scenarios added (S-LSK-013, S-LSK-014, plus cross-refs to continuation path)
- R-LSK-002: scope statement added ("on nil-continuation path"), new scenario S-LSK-015
- R-LSK-004: AG-13 release scope paragraph added, new scenario S-LSK-016
- R-LSK-006: seam sentence reconciled (harness does NOT call Schedule), new scenario S-LSK-017
- Explicit non-requirements: two lines back-annotated as CLOSED by AG-13

### MODIFIED: agent-permission-protocol/spec.md
- R-APP-002: AG-13 observability statement added, two new scenarios S-APP-015/S-APP-016 (bite)
- Explicit non-requirements: one line back-annotated as CLOSED by AG-13
- Evidence discipline: staleness finding recorded for `:172` gap

### ADDED: agent-history/spec.md
- R-HIS-010: new requirement for loop's continuation-path wiring (5 scenarios S-HIS-090..094)
- NFR-HIS-003: re-scoped to record AG-12 freeze and AG-13 release
- Explicit non-requirements: two lines back-annotated as CLOSED by AG-13

### MODIFIED: agent-turn-termination/spec.md
- R-ATT-004: AG-13.3's closure and discharge of pause resumption back-annotated
- Explicit non-requirements: one line back-annotated as CLOSED by AG-13, gap at `:148` discharged

### ADDED: agent-tool-scheduler/spec.md
- R-TLS-012: new requirement for sink-ownership seam (3 scenarios S-TLS-013/014/015)
- Explicit non-requirements: one line back-annotated as CLOSED by AG-13, `ToolSource` re-homed to AG-20

## Docs updated

`docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md`:
- Status header bumped to **13 of 24** (AG-13 complete)
- Completion checklist AG-13 line ticked
- Date updated to 2026-08-17

## Size

89 tasks complete across 7 phases. Forecast 1630–2660 changed lines; final scope pre-authorized under `size:exception`.

## Coverage gate

- `loop.go` line coverage: 87.94% (exceeds 80% floor, includes new continuation branches)
- `harness.go` line coverage: 93.26% (new file, exceeds 80% floor)

## Acceptance

All acceptance criteria from the charter met:

1. ✅ Every `S-RUN-001`…`S-RUN-112` has recorded evidence; both bites (`S-RUN-061`, `S-RUN-091`) RED-recorded with failing output before GREEN
2. ✅ All six charter Gherkin scenarios mapped and closed; none reduced
3. ✅ `cd backend/agent && make test` green under `-race`; `make lint` (after `golangci-lint cache clean`), `make build`, and `make vuln-check` all clean
4. ✅ `CheckStream` accepts multi-turn run stream unmodified, `stream_check.go` byte-unchanged
5. ✅ Every existing test asserting unconditional run brackets or per-call sequence restart, and every AG-09/AG-10 scheduler test, passes file-unchanged
6. ✅ Both substrate filters byte-in-sync, widened by exact filename suffix only
7. ✅ All five cross-cut deltas written and every line re-read against shipped change by `sdd-verify`
8. ✅ Docs 0003 updated: status header ticked, AG-13 sentence added, milestone counters bumped

## Traceability

| Artifact | Type | Location |
| --- | --- | --- |
| Proposal | `sdd/cachicamas-agent-run-driver/proposal` | Engram (persisted during propose phase) |
| Spec | `sdd/cachicamas-agent-run-driver/spec` | Engram (persisted during spec phase) + `openspec/changes/cachicamas-agent-run-driver/specs/` (five deltas) |
| Design | `sdd/cachicamas-agent-run-driver/design` | Engram (persisted during design phase) |
| Tasks | `sdd/cachicamas-agent-run-driver/tasks` | Engram + `openspec/changes/cachicamas-agent-run-driver/tasks.md` (all 89 checked) |
| Apply progress | `sdd/cachicamas-agent-run-driver/apply-progress` | Engram (persisted after apply phase) |
| Verify report | `sdd/cachicamas-agent-run-driver/verify-report` | Engram (persisted after verify phase) |
| Archive report | `sdd/cachicamas-agent-run-driver/archive-report` | This file + Engram (persisted after archive) |

All Engram observations recorded with complete traceability. Five promoted spec files live in `openspec/specs/agent-{run-driver,loop-skeleton,permission-protocol,history,turn-termination,tool-scheduler}/spec.md`.

## Key Learnings

1. Queue closure at function exit via deferred close prevents silent drops: a single `defer h.queue.close()` at the top of `Run` guarantees every exit path (failure, rejected append, terminal success) leaves the queue closed, avoiding scattered conditional closes that a new exit path could miss.
2. Nil-default seams enable perfect backwards compatibility: `TurnOptions.Continuation` and `Scheduler.LeaveSinkOpen` zero values preserve pre-AG-13 behavior byte-for-byte, allowing every existing test to pass file-unchanged without conditional logic.
3. Parked-wait observation requires execution, not assumption: the bite in `S-APP-016` that scratches the wait path and verifies its failure is load-bearing — prose alone cannot distinguish a registration guard from a wait-detection guard, and the existing acknowledgement-level test `TestPermission_WakeParked_AckGatesCompletion_NoRunBeforeSinkDelivery` measures a different level.
4. Substrate filter widening by exact suffix only prevents hidden edits: requiring both filters to widen identically and by full filenames forces explicit enumeration of every new file, making hidden scope creep visible through filter divergence or suffix mismatch.
5. Stream synchronization beats wall-clock ordering: using channel closes and event boundaries as ordering guarantees eliminates sleep/timeout races and makes the test deterministic and repeatable under `-count=15 -race`.
