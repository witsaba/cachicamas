# Delta for `agent-history` — AG-13 does the wiring AG-12 deferred

> **Change**: `cachicamas-agent-run-driver` · **AG-13** (Layer 2, Wave 3), `0003:1294-1370`
> **Modifies**: `agent-history` ([`../../../../specs/agent-history/spec.md`](../../../../specs/agent-history/spec.md)) — adds `R-HIS-010`; modifies `NFR-HIS-003` (`spec.md:185`) and the Explicit non-requirements table (`spec.md:201`, `spec.md:206`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES each block in the main spec with the block below; **full-block preservation is mandatory**.
> **Why this delta**: `NFR-HIS-003` states "`loop.go` and `scheduler.go` MUST be byte-unchanged: wiring history into `Turn`/`Schedule` is AG-13's". That statement was **scoped to AG-12**, not a standing prohibition — its own second clause names AG-13 as the wirer. AG-13 does the wiring, so the requirement must be re-scoped to say what it always meant and to record what the wiring is. Two non-requirement rows close: "History is wired into `Turn` / `Schedule`" (`:201`) and "Steering-message queueing at turn boundaries" (`:206`).
> **Ownership**: the **appending done inside `Turn`** is owned here by the new `R-HIS-010`. The **harness-side** appends (the prompt and each steered message), the turn-boundary `CloseTurn`, and steering-queue semantics are owned by `R-RUN-005` and `R-RUN-008` in [`../agent-run-driver/spec.md`](../agent-run-driver/spec.md). Exactly one requirement owns each message class; see the appender split below.
> **Hard constraint carried, not relaxed**: AG-13 adds **no new exported `History` method**. `history_surface_guard_test.go` is a closed-route guard and stays green with its source **byte-unchanged**; `history.go` itself is byte-unchanged. The wiring uses `Append`, `CloseTurn` and `Entries` only.

## ADDED Requirements

### Requirement: On the continuation path, `Turn` commits the turn's own messages — `R-HIS-010`

When `Turn` runs with a non-nil continuation carrying a transcript store (`R-LSK-001`), it MUST commit that turn's own messages to that store before returning, and it MUST commit **only** those. Specifically, after finalize and after the rejoin:

1. It MUST append the turn's assistant message — including the turn's tool-call parts, with provider-exact bytes — so that a pure-tool turn yields a constructible assistant message whose calls the pairing invariant of `R-HIS-003` can match. A turn that produced no content at all MUST append nothing rather than appending an empty message.
2. It MUST then append **one tool-result message per rejoin result, in call order**, mapping each result's outcome onto the Layer 1 tool-result value that already carries it — the success form for a successful outcome and the failure form for both failure outcomes. This is the one-message-per-result shape orphan synthesis already established.
3. An append failure MUST be returned as a typed error from `Turn`. It MUST NOT be swallowed, and a turn whose append failed MUST NOT be reported to the caller as successful.
4. `Turn` MUST NOT call `CloseTurn`. Closing the turn is the run driver's, at the turn boundary (`R-RUN-005`).

**The appender split is normative**, because a message written twice and a message written nowhere are the same defect wearing different clothes:

| Message class | Sole appender |
|---|---|
| The run's initial prompt | the run driver, at run entry (`R-RUN-005`) |
| Each steered user message | the run driver, at a turn boundary (`R-RUN-008`) |
| The turn's assistant message | `Turn`, on the continuation path (this requirement) |
| Each tool-result message | `Turn`, on the continuation path (this requirement) |

Because `Turn` appends calls first and results second, the open-call set is empty by the time the driver closes the turn, so the boundary `CloseTurn` succeeds under `R-HIS-003` without any special case.

With a **nil** continuation, `Turn` MUST commit nothing anywhere: there is no transcript store to commit to, and every pre-AG-13 path stays byte-stable.

Consecutive same-role entries MUST remain legal. `commitAppendOp` carries no role-alternation check, so this requires no History change and none is made.

#### Scenarios

- **S-HIS-090** — Given a continuation carrying a transcript store and a provider scripted with a turn that emits assistant text and one tool call whose tool succeeds, when `Turn` runs, then the store holds, in order, the turn's assistant message carrying the tool call with provider-exact argument bytes, followed by one tool-result message correlated to that call; and when the caller then closes the turn, then the close succeeds because no call is open.
- **S-HIS-091** — Given a continuation and a provider scripted with a turn producing **no** content and no tool call, when `Turn` runs, then nothing is appended to the store, a subsequent read returns the store unchanged, and `Turn` returns without error.
- **S-HIS-092** — Given a continuation and a turn whose scheduled calls resolve to a mix of success and both failure outcomes, when `Turn` runs, then exactly one tool-result message is appended per rejoin result, in call order, each correlated to its own call identity, and each failure outcome is carried by the Layer 1 failure form rather than by a content sentinel.
- **S-HIS-093** — Given a continuation whose transcript store rejects the append, when `Turn` runs, then it returns a non-nil typed error carrying that rejection, and the caller's run terminates through the failure path of `R-RUN-011` rather than reporting a successful turn.
- **S-HIS-094** — Given a `TurnOptions` with a **nil** continuation, when `Turn` runs any scripted shape, then no transcript store is touched, and the closed-route history surface guard passes with its source byte-unchanged.

## MODIFIED Non-functional requirements

### NFR-HIS-003 — No Layer 1 edit, no new dependency; the AG-12 wiring freeze, and what AG-13 released

`loop.go` and `scheduler.go` MUST be byte-unchanged **under AG-12**: wiring history into `Turn`/`Schedule` is AG-13's. That was a freeze scoped to the milestone that wrote it — its own second clause names its successor — and it is **not** a standing prohibition on every later milestone.

**AG-13 did the wiring**, as forecast. It modifies `loop.go` (the continuation seam and the commits of `R-HIS-010`), `scheduler.go` (sink ownership) and `tool.go` (the sink-ownership flag), and it does so without touching `history.go` and without adding any exported `History` route. From AG-13 onward, this requirement's byte-unchanged claim covers `history.go` and Layer 1 only; a later milestone that wants to change `loop.go` or `scheduler.go` needs no release from this requirement, but does need whatever release `R-LSK-004` requires.

Every file under `backend/agent/src/ai/` MUST be byte-unchanged, including `ai/validation.go`. `go.mod` and `go.sum` MUST be unchanged. Layer 1 is consumed, never edited. **`history.go` MUST be byte-unchanged and no exported `History` method may be added**: `history_surface_guard_test.go` is a closed-route guard, set-equal in both directions, and it MUST pass with its own source unchanged.

(Previously: "`loop.go` and `scheduler.go` MUST be byte-unchanged: wiring history into `Turn`/`Schedule` is AG-13's" — stated without a milestone qualifier, so a later reader could take it as a standing prohibition that AG-13 violates rather than as the AG-12 freeze it was, and it made no claim about `history.go` or the exported-route surface.)

#### Scenarios

- **S-HIS-095** — Given the merge base of the AG-13 branch with `origin/main`, when `git diff` is taken over `backend/agent/`, then `history.go` is byte-unchanged, every file under `backend/agent/src/ai/` is byte-unchanged, `go.mod` and `go.sum` are byte-unchanged, and the only Layer 2 non-test files that differ are `loop.go`, `scheduler.go`, `tool.go` and the run driver's new file.
- **S-HIS-096** — Given `history_surface_guard_test.go` and its committed expected-route set, when the suite runs against this change, then the guard passes with its source byte-unchanged and the enumerated exported route set is equal in both directions — AG-13 added no route and removed none.

## MODIFIED Explicit non-requirements

The table is reproduced in full; two rows are back-annotated as closed and none is removed.

Stated so that no test, guard or acceptance line is written as if AG-12 closes more than it does.

| Not claimed | Owner |
|---|---|
| History is wired into `Turn` / `Schedule` | AG-13 (run driver). **CLOSED by AG-13**: `Turn` commits the turn's assistant message and its tool-result messages on the continuation path (`R-HIS-010`); the run driver commits the user-side messages and closes each turn (`R-RUN-005`). `Schedule` itself still commits nothing — it returns the rejoin and `Turn` commits from it |
| A transcript is persisted or reloaded across processes | Layer 3 |
| The invariant holds after compaction removes entries | AG-18.2, which re-proves it — ordinal-derived identity is a known, named constraint AG-18 inherits |
| Cancellation semantics that *produce* an interruption | AG-14. AG-12 repairs the transcript an interruption left; it neither detects nor causes one. *(Still open at AG-13: the run driver propagates its context unmodified and defines no cancellation vocabulary.)* |
| Context-window accounting over the transcript | AG-17 |
| Steering-message queueing at turn boundaries | AG-13.2. **CLOSED by AG-13**: the queue, its arrival ordering, its zero-drop guarantee and its typed post-terminal rejection are owned by `R-RUN-008`. History needed no change to accept it — `commitAppendOp` carries no role-alternation check, so consecutive same-role entries were already legal |
| A new rule class in `ai/validation.go` | Not this milestone, and forbidden under this change. *(Still true at AG-13: the continuation's typed rejections and `Steer`'s typed rejection reuse existing `ai` rule classes.)* |
