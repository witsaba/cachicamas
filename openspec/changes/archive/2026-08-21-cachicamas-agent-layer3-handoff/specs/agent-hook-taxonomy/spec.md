# Delta — `agent-hook-taxonomy` (AG-23)

> **Change**: `cachicamas-agent-layer3-handoff` · **AG-23** · Target: `openspec/specs/agent-hook-taxonomy/spec.md`
> **Ops**: MODIFIED `R-HKS-010` — the W-6 disposition AG-22 handed forward is **closed on the merits** here: one file is **restored** to the scope fence, one release is made **permanent** with its reasoning, and consequence 2's forward assignment of a removal to AG-23 is **resolved** (declined, with a post-v1 path) rather than left pointing at a milestone that has ended. Two scenarios (`S-HKS-027`, `S-HKS-028`) are appended.
> **Decision**: proposal `D-6` as bound by design `AD-7`, and `agent-layer3-handoff`'s `R-L3H-009`.

## The mechanism, read before deciding

The shipped fence resolves a **base ref** (an override if set, otherwise the merge base with the trunk) and
asserts the diff over each listed file is **empty**. It is therefore a **per-branch scope fence against a
moving merge base**, not a recorded hash. Since AG-22 merged, the baseline has moved, so *"restore an
entry"* means exactly: **this branch must not touch that file**. The two released files consequently decide
**independently**, and this delta decides them separately rather than as a pair.

## Two files, two different verdicts

| File | Verdict | Reason |
|---|---|---|
| `scheduler.go` | **RESTORED to the fence** | AG-23 ships no scheduler change: the kit and the proof are new sibling packages, the examples are a new test file, and the one production fix is confined to the harness's own file. Restoration costs nothing and re-arms a guard whose release expired with the milestone that took it |
| `import_boundary_test.go` | **NOT restored; the release is made PERMANENT and recorded** | That file's allowlists and pattern set are the layer's **designed extension point**. Every milestone that admits a dependency or adds a package sibling must edit it **by construction**. Two consecutive, independent milestones needing the same file proves the freeze **entry** was wrong, not that the milestones were. Freezing an extension point is a category error |

**This is a closure, not a silent carry-forward.** AG-23 is the last milestone: a carry-forward past it has
nowhere to go, so the reasoning is recorded **twice** — beside the list in the guard's own source, and in
the compatibility statement — precisely so a later reader cannot mistake it for a dropped follow-up.

## Not modified, and why

| Element | Verdict |
|---|---|
| The rest of `R-HKS-010`'s byte-unchanged list | **Byte-unchanged and honoured.** AG-23 touches none of the other named files; the harness's own file is on **no** fence list, which is what makes `R-RUN-014`'s fix admissible without a release |
| Consequences 1, 3, 4 and 5 | **Reproduced verbatim.** Only consequence 2 gains a resolution, and it gains it as an **addition**, not a rewrite of what AG-20 recorded |
| The stall-type placement paragraph | **Reproduced verbatim.** AG-23 moves no type |
| `R-HKS-011`'s closed-sequence table | **Not modified — checked, not assumed.** AG-23 registers no event kind, adds no producer and adds no goroutine class; the harness's forwarder gains a **cancellation path and a join**, which removes a goroutine's ability to outlive the unwind rather than adding one. `R-HKS-012`'s inertness and `S-HKS-025`'s row-by-row evaluation are unaffected |
| `NFR-HKS-005`'s release-before-baseline rule | **Not modified.** AG-23 gates no observing hook and samples no goroutine baseline |

## Header maintenance obligation at promotion

`sdd-archive` MUST add **`S-HKS-027`** and **`S-HKS-028`** wherever this spec's scenario identifiers are
enumerated, as a **range and never a total**. No existing `S-HKS-` identifier is renumbered.

## MODIFIED Requirements

### R-HKS-010 — The scope fence: what AG-20 ships, what it does not, and how its two releases are finally disposed of (AG-23 amendment)

AG-20 MUST register **no** new `EventKind` (the shipped guard stays at its committed kind count, `scope_fence_test.go:87`), add **no** new turn outcome member and **no** new cost label, and change **no** existing exported signature: `Turn` and `Run` MUST keep their signatures and `Harness` MUST gain no exported method.

`event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `delegation_events.go`, `permission_events.go`, `cost_events.go`, `cost_usage.go`, `turn_events.go`, `run_events.go`, `failure.go`, `sequence.go`, `compaction_events.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, `scheduler.go`, `delegation_seam.go`, `go.mod`, `go.sum` and **every file under `backend/agent/src/ai/`** MUST be **byte-unchanged**. `backend/agent/src/agenttest/` MUST be byte-unchanged: the test gate primitive is **used**, not widened.

**The stall types MUST live in the new hook file, not in the typed-failure file.** Editing the typed-failure file would pass the mechanical guard silently while violating `R-LSK-004`'s prose; the placement is declined explicitly rather than left to convenience.

Five consequences MUST be stated here rather than discovered by a later consumer:

1. **No concrete hook ships.** The charter's own out-of-scope line is verbatim: *"Any concrete hook implementation (Layer 3 wires them)"* (`0003:1874`). Layer 3's wiring is doc 0004 CO-24.1/CO-24.2, and `S-AGS-048` already names those nodes.
2. **`TurnOptions.PreRequestHook` is kept and superseded in prose only.** Its removal is AG-23's, and AG-20 does not take it. **(AG-23 resolution — the forward assignment is DISCHARGED, by DECLINING the removal.)** AG-23 is the milestone that **freezes** the v1 surface, and removing an exported member is a breaking change: the milestone that freezes a surface is the worst possible place to take one. The field is therefore **kept, unamended in behaviour, and frozen into v1**, recorded in the compatibility statement as frozen-and-superseded with a **post-v1** removal path. It MUST NOT be marked with Go's deprecation convention, for the reason `R-HKS-002` already records. No shipped exported-surface pin moves. **The assignment does not survive this milestone**: no requirement may continue to name AG-23 as the holder of this removal, because AG-23 has ended (`agent-layer3-handoff`'s `R-L3H-009`).
3. **AG-21 inherits the stalled-observer goroutine leak and the release-before-baseline test rule knowingly** (`NFR-HKS-005`), rather than discovering them.
4. **`scope_fence_test.go` is deliberately ABSENT from the byte-unchanged list above — a narrow, recorded release, discovered during `sdd-apply`, not planned at design time** (the [`agent-loop-skeleton` delta](../agent-loop-skeleton/spec.md)'s own `R-LSK-008` consequence 6 owns the full account). `TestScopeFence_S_TLS_020_SeamRidesBesidePolicySlotNotInsideIt`'s own `NumMethod()`-and-named-set assertions (`:102-105`, cited above) and the event-kind-count assertion (`:87`, cited above) both stay byte-unchanged and green; the ONE edit converts that same test's own anti-vacuity floor — an empty `git diff` over `scheduler.go` — from `t.Fatal` to `t.Skip`, because an empty diff means the absence it checks holds vacuously on a branch (any branch after AG-19's merge, AG-20 included) that does not itself touch `scheduler.go`. `cancellation_test.go` carries the identical fix for its own mirror (`S-DEL-015`) but was never named byte-unchanged by this requirement to begin with.
5. **`doc.go` and `doc_contract_guard_test.go` are ALSO deliberately ABSENT from the byte-unchanged list above — a second, unrelated narrow release, discovered during `sdd-verify` (CRITICAL-2), not planned at design time and not the same release consequence 4 records.** `L2C-08`'s package-wide contract row (`R-CAN-008`) claimed *"every goroutine the package itself owns has exited"* past the wind-down bound; AG-20's own observer lane goroutine may legitimately still be running, by design, while blocked inside a permanently stalled observer (`R-HKS-008` consequence 1), so the claim was false of the shipped code and no delta recorded it. Fixed by widening `L2C-08`'s existing third-party-tool carve-out to also name a permanently stalled third-party observing-hook invocation, reported typed by hook point and registration index — the identical mechanism the row already uses, not a weakening. The [`agent-cancellation-tree` delta](../agent-cancellation-tree/spec.md) (`R-CAN-006`, `R-CAN-008`, `S-CAN-016`, `S-CAN-017`) owns the full account. Both files are pre-existing entries in both substrate filters since AG-18 — the milestone that took the only PRIOR release of these same two files, for the unrelated reason of amending `L2C-07` — so this release needs no new filter entry either.

**AG-23 amendment — the fence's two AG-22 releases are disposed of, separately and finally.** The shipped
fence is a **per-branch scope fence against a moving merge base**, not a recorded hash: an entry asserts
that *the current branch does not touch that file*. A release is therefore scoped to the branch that took
it and expires with it unless it is renewed on the merits. AG-22 released two files; AG-23 is Layer 2's
**last** milestone, so both are disposed of here rather than carried:

- **`scheduler.go` is RESTORED to the enforced list.** AG-23 ships no scheduler change, so the entry costs
  nothing and re-arms a guard whose release expired with AG-22. AG-23 MUST NOT touch that file.
- **`import_boundary_test.go` is NOT restored. Its release is made PERMANENT, and the reasoning MUST be
  recorded in place — beside the list in the guard's own source — and again in the compatibility
  statement.** That file's allowlists and pattern set are the layer's **designed extension point**: every
  milestone that admits a dependency or adds a package sibling must edit it **by construction** (`R-AGP-003`
  requires the amendment and the import it authorises to be one commit, which makes editing it mandatory,
  not optional). Two consecutive, independent milestones needing the same file establishes that the freeze
  **entry** was wrong. **Freezing an extension point is a category error**, and restoring it would make the
  restoring branch fail its own restored guard. This is a closure on the merits, **not** a silent
  carry-forward, and the double recording exists so a later reader cannot read it as a dropped follow-up.

**No further release is granted.** A later phase electing to edit any other listed file MUST take its own
recorded release; permanence is granted here to exactly one named file, on exactly the stated reasoning.

(Previously: identical normative text without the AG-23 amendment and without consequence 2's resolution. Consequence 2 assigned the singular pre-request field's removal to AG-23 and stopped there; AG-23 is now the milestone in question and has **declined** the removal, so leaving the sentence as a forward assignment would point at a milestone that has ended. The AG-22 releases of `scheduler.go` and `import_boundary_test.go` were recorded on AG-22's branch and would otherwise expire unexamined against the moved merge base — one of them wrongly, since the milestone that must edit the extension point cannot also restore its freeze.)

#### Scenarios

- **S-HKS-024** — Given the merge base of this change's branch with `origin/main`, when `git diff` is taken over `backend/agent/`, then every file named byte-unchanged above is byte-unchanged; the diff under `backend/agent/src/ai/` and under `backend/agent/src/agenttest/` is **empty**; the `go.mod`/`go.sum` diff is empty; the event-kind guard passes at its committed kind count with AG-20 registering none; the every-kind-constructible guard passes; no turn outcome member and no cost label member was added; and `Turn`'s and `Run`'s signatures and `Harness`'s exported method set are unchanged.
- **S-HKS-027** — **(AG-23: the two releases are disposed of, and the disposition is checked rather than claimed.)** Given the merged AG-23 change, when the shipped fence's enforced list is read, then `scheduler.go` **appears** on it and `import_boundary_test.go` does **not**; when `git diff` is taken against the merge base over `scheduler.go`, then it is **empty**; when the source beside the list is read, then it records the permanent exclusion with its extension-point reasoning **in place**; and when the compatibility statement is read, then the same reasoning appears there too. Given the exclusion removed and `import_boundary_test.go` restored to the list, when the fence runs on this branch, then it FAILS — recording, as evidence rather than as argument, why restoration was refused. Recorded, then reverted.
- **S-HKS-028** — **(AG-23: consequence 2's forward assignment is discharged, and the discharge is checked against the shipped surface.)** Given the merged AG-23 change, when Layer 2's exported surface is enumerated from an external test package, then `TurnOptions.PreRequestHook` is **still present**, carries no deprecation marker, and every shipped exported-surface and signature-arity pin is unchanged; when the compatibility statement is read, then the field appears as frozen-and-superseded with a **post-v1** removal path; and when the shipped specs are searched for a requirement still naming AG-23 as the holder of that removal, then **none** remains — the assignment was resolved in place, not left pointing at a milestone that has ended.
