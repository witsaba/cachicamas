# Delta for `agent-turn-termination` — AG-13 acts on the pause AG-11 made visible

> **Change**: `cachicamas-agent-run-driver` · **AG-13** (Layer 2, Wave 3), `0003:1294-1370`
> **Modifies**: `agent-turn-termination` ([`../../../../specs/agent-turn-termination/spec.md`](../../../../specs/agent-turn-termination/spec.md)) — `R-ATT-004` (`spec.md:65`) and the Explicit non-requirements list (`spec.md:146`, `spec.md:148`).
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES each block in the main spec with the block below; **full-block preservation is mandatory**.
> **Why this delta**: `R-ATT-004` closes with "**Acting on** a pause — performing the resumption — is AG-13's, not this capability's (`0003:1123`)", and the non-requirements list repeats it at `:146`. AG-13.3 acts on it, so both must be back-annotated. Separately, `:148` reserves "the `_ = sched.Schedule(...)` tool-result discard at `loop.go:265`" as a pre-existing gap AG-11 must leave byte-unchanged; AG-13 is the milestone that changes that line, on the continuation path, so the reservation is discharged rather than left contradicting merged code.
> **Ownership**: the **resumption** — appending the partial verbatim, replaying it in the next request, continuing to a real terminal — is owned by `R-RUN-009` in [`../agent-run-driver/spec.md`](../agent-run-driver/spec.md). The **visibility** of the pause as its own turn outcome stays owned here and is byte-unchanged in behavior. AG-13 adds no `TurnOutcome` member and no `EventKind`.

## MODIFIED Requirements

### Requirement: Refusal and pause diverge, and both diverge from finished — `R-ATT-004`

A turn ending in `FinishReasonRefusal` and a turn ending in `FinishReasonPauseTurn` MUST be distinguishable **from each other and from a normally finished turn by the `TurnOutcome` value alone**. A consumer MUST NOT need to inspect any `ai.FinishReason` to tell them apart.

Refusal MUST mean the turn is over: it is a complete answer of the form "no" (`finish_reason.go:71-80`), so the loop MUST close the turn's brackets and MUST NOT mark it resumable. Pause MUST be visible as its own **suspended-resumable** outcome carrying the obligation recorded at `finish_reason.go:82-91` — the turn expects resumption with the content already received replayed **verbatim**. The content the loop returns and emits for a paused turn MUST therefore be byte-identical to the content received, with no re-serialisation, re-ordering, or stripping of an opaque round-trip token.

**Acting on** a pause — performing the resumption — was AG-13's, not this capability's (`0003:1123`), and **AG-13.3 has now done it**. The division of labour is recorded here so no later reader re-derives it: this capability owns the pause's **visibility**, AG-13 owns the **action**. Concretely, AG-13 imposes two obligations on this capability that are already satisfied and MUST stay satisfied:

- the paused turn's `turn_end` MUST continue to carry the paused outcome, and a caller driving a run MUST forward it unchanged rather than rewriting or absorbing it — a resumption that hid the pause would defeat this requirement's whole point;
- the partial `ai.Message` this capability requires the loop to return MUST remain the exact value a resumption replays. AG-13 appends **that value** and re-includes it verbatim in the next request's transcript; it never re-synthesizes an equivalent one. If this capability ever stopped returning byte-identical content, `R-RUN-009` would break, so the byte-identity clause above is now load-bearing for two capabilities rather than one.

A refusal MUST still terminate. Refusal is a terminal candidate for the run driver's dispatch (`R-RUN-002`), never an iteration trigger — the divergence this requirement pins is what makes that dispatch correct.

(Previously: the requirement deferred the pause's resumption to AG-13 with no statement of what AG-13 would depend on, so the byte-identity clause read as a property of one capability rather than as a contract two capabilities share.)

#### Scenarios

- **S-ATT-005** — Refusal, pause and finished are three values. Given three turns scripted to end in `FinishReasonRefusal`, `FinishReasonPauseTurn` and `FinishReasonStop` respectively, when each runs and its `turn_end` outcome is read, then the three outcome values are pairwise different, and no assertion in the scenario reads an `ai.FinishReason` to distinguish them.
- **S-ATT-006** — Pause replays received content verbatim. Given a turn scripted with interleaved reasoning and text deltas, a non-empty `[]byte` reasoning round-trip token, and a completion carrying `FinishReasonPauseTurn`, when `Turn` runs, then the returned `msg` and the emitted delta sequence carry the received fragments byte-for-byte, the reasoning round-trip token is byte-equal to the script's, and the `turn_end` outcome is the paused member — not the refused member and not `TurnOutcomeFinished`.
- **S-TTB-003** — **(bite)** Given the pause outcome aliased onto the refusal outcome (or onto `TurnOutcomeFinished`), when `S-ATT-005` runs, then it reports the collapse — the loop-termination defect AI-13 exists to prevent (`0003:1121`) is detected, not assumed absent. RED-recorded BEFORE `S-ATT-005` is GREEN.
- **S-ATT-013** — **AG-13 forwards the pause, and the partial survives the run.** Given a harness-driven run whose first turn ends in `FinishReasonPauseTurn` and whose second ends in a terminal reason, when the run's consumer stream is read, then turn one's `turn_end` carries the paused outcome unrewritten and unabsorbed; and when the transcript entry for that turn is compared to the message `Turn` returned, then the two are byte-identical including the reasoning round-trip token. Owned jointly with `R-RUN-009` / `S-RUN-080` / `S-RUN-081`.

## MODIFIED Explicit non-requirements

The list is reproduced in full; two lines are back-annotated as closed and none is removed.

The following are **out of scope** and MUST NOT be implemented by this change:

- **Acting on retryability** — AG-15. The loop reports `Retryable()` through the typed surface and MUST NOT retry, back off, or route to a fallback provider. *(Still open at AG-13: the run driver ends a failed run typed and retries nothing, `R-RUN-011`.)*
- **Acting on pause** — AG-13. AG-11 makes pause visible as its own outcome and stops there; the loop MUST NOT attempt a resumption. **CLOSED by AG-13**: the resumption ships in the run driver, owned by `R-RUN-009`. The constraint on `Turn` is unchanged and still binding — **the loop still MUST NOT resume**. Resumption is the driver calling `Turn` again with a transcript that already contains the partial; `Turn` itself remains a one-turn function with no knowledge that it is resuming anything.
- **Pairing enforcement for turns that fail mid-tool-calls** — owned jointly by AG-11.2's typed path and AG-12's history boundary and orphan synthesis (`0003:1151`). This capability owns only the dispatch and the typed emission.
- **The `_ = sched.Schedule(...)` tool-result discard at `loop.go:265`** — a pre-existing gap immediately adjacent to this change's edit site. It MUST come out of AG-11 **byte-unchanged**: neither fixed nor further entrenched. **DISCHARGED by AG-13**, and the scope of the fix is stated exactly: on the **continuation path** the rejoin results are captured and committed to the run's transcript (`R-HIS-010`), so the discard no longer exists there. On the **nil-continuation path** the discard remains — a bare `Turn` invocation still has no transcript to commit to, and changing it would alter behavior no caller asked to change. The reservation is therefore discharged for the path AG-13 drives and left standing for the path it does not.
- **A new `EventKind` for failure** — `failure.go:6-7` already settled it: failures ride the typed outcomes. *(Still true at AG-13: the run driver's failed run-close reuses the existing run outcome and the existing typed failure surface.)*
- **Multi-turn state, cancellation tree, cost events** — AG-12…AG-18. *(AG-13 ships the multi-turn **run**; state that outlives a run is AG-21's, the cancellation tree is AG-14's, cost aggregation is AG-16's.)*
