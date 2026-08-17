# Delta for `agent-history` — AG-14 gives orphan synthesis its first production caller

> **Change**: `cachicamas-agent-cancellation-tree` · **AG-14** (Layer 2, Wave 3), `0003:1371-1442`
> **Modifies**: `agent-history` ([`../../../../specs/agent-history/spec.md`](../../../../specs/agent-history/spec.md)) — `R-HIS-007` (`spec.md:135-147`) and the Explicit non-requirements table (`spec.md:242-250`). **Back-annotation only.**
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES each block in the main spec with the block below; **full-block preservation is mandatory**.
> **Why this delta**: AG-12 shipped orphan synthesis and wired it into nothing — it "repairs the transcript an interruption left; it neither detects nor causes one" (`spec.md:247`), and AG-13 left that row open. AG-14 is the milestone that produces the interruption, so synthesis gains its **first production caller** inside the harness's wind-down (`R-CAN-002`). Nothing about synthesis's behavior changes; what changes is that a requirement previously exercised only from tests is now on a production path, and the spec should say so rather than let a reader assume it is still unreached.
> **Hard constraint carried, not relaxed**: AG-14 adds **no new exported `History` method** and does not modify `history.go`. `history_surface_guard_test.go` is a closed-route guard and MUST stay green with its source **byte-unchanged**. The wind-down uses the existing `SynthesizeOrphans`, `CloseTurn` and `Entries` surface only.
> **Ownership**: the wind-down algorithm and its ordering are owned by `R-CAN-002` in [`../agent-cancellation-tree/spec.md`](../agent-cancellation-tree/spec.md) and by the `R-RUN-011` carve-out in [`../agent-run-driver/spec.md`](../agent-run-driver/spec.md). This delta claims no new history behavior.

## MODIFIED Requirements

### R-HIS-007 — Orphan synthesis closes every orphaned pair with an interruption artifact

Synthesis MUST complete **every** orphaned tool call in the transcript with a result recorded as an interruption artifact. The artifact MUST be distinguishable from a real result **by the entry envelope's origin discriminator** (`appended` | `synthesized`), read through a read-only accessor.

Distinguishability MUST NOT depend on result content and MUST NOT depend on `ai.ToolResult.Failed()`. A synthesized result MAY set `Failed()`, but so does a real failing tool, so `Failed()` is not the discriminator; and a real tool can emit any content bytes, so a content sentinel is forgeable and is prohibited. No test in this capability may distinguish origin by inspecting content or `Failed()`.

Synthesis MUST route through the commit path of `R-HIS-004`; it is not a second door.

**AG-14 back-annotation: synthesis now has a production caller.** The harness's cancellation wind-down invokes it as its first step, before closing the turn and before emitting the run-close (`R-CAN-002`). Three consequences, recorded so no reader has to re-derive them:

1. **No behavior of this requirement changes.** The production caller uses the existing exported surface; `history.go` is byte-unchanged and no exported route is added.
2. **Idempotency is load-bearing, not merely nice.** `R-HIS-008` is what makes the wind-down safe to run on every cancellation path regardless of how far the turn got: a turn whose results were already committed synthesizes nothing, so the wind-down needs no special case for "were we mid-turn or between turns".
3. **The interruption this artifact was designed for is now real.** From AG-14 onward, a transcript carrying `synthesized`-origin entries is the ordinary observable outcome of an interrupted or shut-down run, not only a test fixture, and the run-driver carve-out of `R-RUN-011` is what routes to it.

(Previously: the requirement described synthesis with no statement of who calls it in production; the AG-12/AG-13 reading was that nothing did.)

#### Scenarios

- **S-HIS-060** — Given a transcript interrupted after tool calls were issued but before their results arrived, when synthesis runs, then every orphaned call has a matching result in the transcript, each such entry reports origin `synthesized`, and every pre-existing entry still reports origin `appended`.
- **S-HIS-061** — Given a synthesized result and a real appended result constructed with byte-identical content and with `Failed()` set identically, when each entry's origin is read, then the two are distinguished correctly, and the assertion reads neither content nor `Failed()`.
- **S-HIS-062** — Given a history over which synthesis has run, when the turn is closed, then the close succeeds — synthesis produced a transcript the pairing invariant of `R-HIS-003` accepts, through the same commit path.
- **S-HIS-097** — **AG-14: the production caller.** Given a harness-driven run interrupted while a tool call is in flight, when the run has returned, then the transcript read back through the existing surface holds a matching result for every call the interrupted turn had issued, each such entry reports origin `synthesized`, every entry committed before the signal still reports origin `appended`, and the turn is closed — no test invoked synthesis directly. Cross-referenced to `S-CAN-001`.
- **S-HIS-098** — Given the AG-14 branch, when the suite runs and `git diff` is taken against the merge base, then `history.go` is **byte-unchanged**, `history_surface_guard_test.go` is byte-unchanged and passes, and its enumerated exported route set is equal in both directions — AG-14 added no route and removed none.

## MODIFIED Explicit non-requirements

The table is reproduced in full; one row is back-annotated as closed and none is removed.

Stated so that no test, guard or acceptance line is written as if AG-12 closes more than it does.

| Not claimed | Owner |
|---|---|
| History is wired into `Turn` / `Schedule` | AG-13 (run driver). **CLOSED by AG-13**: `Turn` commits the turn's assistant message and its tool-result messages on the continuation path (`R-HIS-010`); the run driver commits the user-side messages and closes each turn (`R-RUN-005`). `Schedule` itself still commits nothing — it returns the rejoin and `Turn` commits from it |
| A transcript is persisted or reloaded across processes | Layer 3 |
| The invariant holds after compaction removes entries | AG-18.2, which re-proves it — ordinal-derived identity is a known, named constraint AG-18 inherits |
| Cancellation semantics that *produce* an interruption | AG-14. AG-12 repairs the transcript an interruption left; it neither detects nor causes one. **CLOSED by AG-14**: interrupt and shutdown ship in the `agent-cancellation-tree` capability, and the harness's wind-down is orphan synthesis's first production caller (`R-HIS-007` back-annotation, `R-CAN-002`). History itself needed **no change** to accept it — no new route, no `history.go` edit. The **deadline** signal remains unclaimed by any milestone |
| Context-window accounting over the transcript | AG-17 |
| Steering-message queueing at turn boundaries | AG-13.2. **CLOSED by AG-13**: the queue, its arrival ordering, its zero-drop guarantee and its typed post-terminal rejection are owned by `R-RUN-008`. History needed no change to accept it — `commitAppendOp` carries no role-alternation check, so consecutive same-role entries were already legal |
| A new rule class in `ai/validation.go` | Not this milestone, and forbidden under this change. *(Still true at AG-14: the cancellation sentinels are Go error values, not `ai` rule classes, and the typed aborts reuse existing ones.)* |

## Non-functional carry

`NFR-HIS-003` holds unchanged and needs no release: its byte-unchanged claim covers `history.go` and Layer 1 from AG-13 onward, and AG-14 touches neither. `NFR-HIS-004`'s exact-filename, byte-in-sync filter rule binds AG-14's new files; the enumerated widening is recorded in [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md) under `R-LSK-004` and MUST NOT be duplicated with a different list here.
