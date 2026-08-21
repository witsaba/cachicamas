# Delta — `agent-run-driver` (AG-21)

> **Change**: `cachicamas-agent-concurrency-hardening` · **AG-21** · Target: `openspec/specs/agent-run-driver/spec.md`
> **Op**: MODIFIED `R-RUN-001` (back-annotation). Two sites are discharged: *"Cross-run transcript state remains AG-21's"* (`agent-run-driver/spec.md:87`) and, by the same inventory, `R-RUN-013` consequence 2's *"does not pre-empt AG-21's cross-run state"* (`:319`).
> **Decision**: proposal D2, binding, option (b) — the deferral is **OWED and discharged by a scenario**, not by declaring the wording stale.

## Header maintenance obligation at promotion

**No new `S-RUN-` or `R-RUN-` identifier is minted by this delta**, so the target's Allocated-IDs line (`agent-run-driver/spec.md:24`) is **unchanged**: it continues to read `R-RUN-001` through `R-RUN-013` and `S-RUN-001` through `S-RUN-113`, as a **range and never a total**. `sdd-archive` MUST NOT add, renumber or total anything there.

The discharge is carried by scenarios in the new capability — `S-CNH-014` (the absence half, with its defeat test), `S-CNH-015` (the legitimate-carry half and the inventory row by row) and `S-CNH-016` (the mandatory defeat bite) — cross-referenced below rather than duplicated here. Minting a parallel `S-RUN-` scenario would create a second implementing test for one claim, which is how two tests come to disagree.

## Not modified, and why

| Element | Verdict |
|---|---|
| `R-RUN-013` (`agent-run-driver/spec.md:314-332`) | **Byte-unchanged.** Its consequence 2 sentence *"does not pre-empt AG-21's cross-run state"* (`:319`) was a **forward reference**, and it stays literally true: the session-start latch still holds no transcript and never resumes a run. What AG-21 adds is the inventory that now enumerates it — recorded in `R-RUN-001` below, where the cross-run clause actually lives, rather than by editing a requirement whose own text needs no change |
| `R-RUN-011`, `S-RUN-101`, `S-RUN-102` | **Byte-unchanged.** The queue-closes-on-every-exit property they own is *asserted* by `S-CNH-015`, not amended |
| `S-RUN-003` (`:95`) | **Byte-unchanged, and its limit is stated rather than silently repaired.** It proves the queue reopened — a **presence** claim. It asserts nothing about anything from run 1 being **absent** from run 2, which is why the deferral survived it for two milestones. `S-CNH-014` is the absence claim it never made |
| `R-RUN-010` (no third path, no timeout) | **Byte-unchanged.** AG-21 adds no wall clock anywhere (`NFR-CNH-002`) |
| `R-RUN-012`'s substrate clause | **Byte-unchanged.** AG-21 requests no release from `R-LSK-004`; its own filter-widening obligation is recorded in this change's `agent-loop-skeleton` delta (`R-LSK-009`) |

## MODIFIED Requirements

### R-RUN-001 — The `Harness` is a value-form driver with a named four-method surface

The run driver MUST be a value-form struct with exported configuration fields, **no constructor** and **no interface**, mirroring the `Scheduler` precedent and the AG-04..AG-12 rule that a concrete boundary type stays a struct until a second implementation actually arrives.

Its public surface MUST be exactly the following methods, **enumerated by name rather than by count** so the pin stays meaningful as the type evolves:

| Method | Contract |
|---|---|
| `Run` | drives one run to its terminal and returns the last turn's `(ai.Message, ai.FinishReason, error)` |
| `Steer` | offers a user message to the in-flight run |
| `Interrupt` | fires the interrupt signal (`R-CAN-002`) |
| `Shutdown` | fires the shutdown signal and latches the terminal refusal (`R-CAN-005`) |

A method not named in that table MUST be observable as a guard failure, never as an unaudited addition. `Interrupt` and `Shutdown` sit beside `Steer` on the **same non-privileged upward path** (`R-RUN-006`): they reach the loop through no privileged channel and hold no handle into it — they flip a context the loop already observes.

**A consequence stated so `sdd-apply` is not read as a silent test rewrite:** the reflection subtest `"exactly two exported methods"` (`harness_test.go:1018-1024`, whose `want` is `[]string{"Run", "Steer"}` at `:1027`) changes as a direct consequence of this requirement. That edit is **conscious and delta-backed**, not incidental; its new expectation MUST be the four names above, sorted, and its subtest name MUST stop asserting a count.

Nil-valued optional fields MUST be resolved to defaults at `Run` entry into locals. The harness MUST NOT mutate the caller's fields, with exactly one recorded exception: it MAY set the sink-ownership flag of `R-RUN-012` once, on the scheduler it drives, before the first turn. AG-14 adds **no second caller-field mutation**: the wind-down bound is a caller-owned zero-default field on the `Scheduler` the caller already injects, which the harness reads and never writes.

`Steer` MUST guarantee **zero drops**: a `Steer` returning nil means the message enters the transcript before a subsequent provider call. After the run's terminal decision has been taken, `Steer` MUST return a typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))` — never a silent drop and never a nil return.

**One run at a time per harness value** — re-scoped from "one `Run` per harness value", because AG-14.1's charter requires the reuse the old wording forbade (`0003:1400`: "a new prompt on the same harness works afterward"). A run that has ended, whether completed, failed or interrupted, MAY be followed by another `Run` on the **same value**, and the steering queue MUST reopen at `Run` entry so the second run's `Steer` calls are accepted with the zero-drop guarantee above rather than meeting the closed queue the first run left behind. **Concurrent runs on one value stay out of scope** and are not made safe by this change.

**Cross-run transcript state: CLOSED by AG-21, and closed by an enumerated inventory with an absence assertion rather than by a citation.** The clause this requirement carried — *"Cross-run transcript state remains AG-21's"* — is discharged here. The claim AG-21 proves is **not** "no state carries over": that is false, and it would forbid a continuing conversation. It is that **the only state outliving a run is state the caller explicitly owns or a shipped requirement already enumerates, and the harness itself retains nothing.** The inventory is closed, and every row is asserted:

| State | Outlives a run? | Owner |
|---|---|---|
| the terminal shutdown flag | **yes** — terminal, one-way, holds no transcript, never resumes a run | `R-CAN-005` |
| the session-start latch | **yes** — one-way, per-value bookkeeping only, holds no transcript | `R-RUN-013` consequence 2 |
| the caller-supplied transcript | **only if the caller set it**; unset ⇒ a fresh transcript per run | the **caller** |
| the run's cancel function | **no** — cleared at exit | `R-CAN-001` |
| the steering queue | **no** — reopened at entry, closed on every exit | this requirement, and `R-RUN-011` |

Three things about the discharge are stated so no later reader re-derives them:

- **Both branches are correct, and neither was asserted anywhere before AG-21.** With a caller-supplied transcript, an adversarial run's wind-down writes orphan synthesis and the turn close into it, so the next run's first provider request legitimately carries them — the caller's continuing conversation. With none supplied, each run resolves a fresh transcript and nothing carries. AG-21 asserts both.
- **The transcript row is proven as an ABSENCE with a defeat test, because a presence assertion proves nothing here.** `S-RUN-003` above and `S-CAN-002` both assert presence — that a second run is accepted, that a steer reaches its transcript, that it reaches its own terminal — and **neither asserts that anything from run 1 is absent from run 2**. Either would stay green with the fresh-per-run mechanism fully defeated. `S-CNH-014` names a **uniquely minted** run-1 artifact, asserts it appears nowhere in run 2's captured request, first proves the needle findable in run 1's own read-back, and carries its defeat — the same assertion re-run against a deliberately shared transcript, which MUST go RED.
- **"Concurrent runs on one value stay out of scope" is restated as STILL TRUE, unweakened.** AG-21 runs the whole combined matrix under the race detector and every cell drives exactly **one** `Run` at a time; signals fire from the test goroutine, which is `R-CAN-004`'s already-decided shape, not concurrency (`R-CNH-002`). A `-race`-clean concurrent-`Run` cell would publish a guarantee this clause and `agent-cancellation-tree/spec.md:204` both deny, so AG-21 fences it by construction rather than leaving it to the fixture.

(Previously: the surface was pinned as "exactly two methods", `Run` and `Steer`, by a bare count; and the requirement closed with "One `Run` per harness value. Cross-run state is **AG-21**'s.", which forbade the same-value reuse AG-14.1 requires.)

(Previously, at AG-21: the requirement ended *"**Cross-run transcript state remains AG-21's**: the only state AG-14 lets outlive a run is the terminal, one-way shutdown flag of `R-CAN-005`, which holds no transcript and never resumes a run."* — an open forward deferral, and an inventory of exactly one row written before the session-start latch of `R-RUN-013` existed. A reader could not tell from this requirement what the complete set of run-outliving state was, nor whether anything from an adversarially ended run reached the next run's provider request, and no scenario in this capability asserted either way.)

#### Scenarios

- **S-RUN-001** — Given a harness value constructed as a struct literal with only its required provider field set, when `Run` drives a scripted single-turn conversation from an external test package, then the run completes without any constructor call, the caller-visible field values are unchanged after `Run` returns except for the recorded sink-ownership flag, and the type's exported method set read by reflection is equal, in both directions, to `{Run, Steer, Interrupt, Shutdown}` — an extra exported method fails the assertion and so does a missing one.
- **S-RUN-002** — Given a run that has taken its terminal decision and returned, when `Steer` is called with a well-formed user message, then it returns an error that satisfies the typed rejection `ai.Invalid(ai.ErrMisplaced, ai.At("steering"))`, the transcript is unchanged, and no further event reaches the consumer sink.
- **S-RUN-003** — **AG-14 serial reuse.** Given a harness value whose first `Run` has returned through the interrupt wind-down of `R-CAN-002`, when a second `Run` is invoked on that same value, then it is accepted, a `Steer` issued during that second run returns nil and its message reaches the second run's transcript before the next provider call, and the second run reaches its own terminal — proving the queue reopened rather than staying closed from the first run. Cross-referenced to `S-CAN-002`. *(AG-21 update: unchanged in claim, and its limit is now recorded rather than assumed away — this scenario asserts **presence** only. The absence half of the cross-run claim is `S-CNH-014`'s, with its defeat test; the two are separate and must not be conflated.)*

*(AG-21: the cross-run inventory above is discharged by `S-CNH-014`, `S-CNH-015` and the mandatory defeat bite `S-CNH-016` in `agent-concurrency-hardening`. No `S-RUN-` scenario is minted for it, so this capability keeps exactly one implementing test per claim.)*
