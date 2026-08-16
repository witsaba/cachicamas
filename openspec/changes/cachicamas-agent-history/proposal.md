# Proposal: AG-12 — History and the pairing invariant

> **Change**: `cachicamas-agent-history` · **Milestone**: AG-12 (Layer 2 Wave 3, milestone 12 of 24; doc 0003 lines 1227–1292)
> **Branch**: `feat/agent-layer2-wave3-ag12` · **Artifact store**: hybrid (Engram + filesystem)
> **Pre-authorized**: `size:exception` against the 1000-line PR review budget
> **TDD**: strict, RED-first (`cd backend/agent && make test`)
> **Closes**: R-07's boundary enforcement · **Blocks**: AG-13, AG-17
> **Exploration**: `exploration.md` · Engram `sdd/cachicamas-agent-history/explore` (#3112)

## Intent

The harness has no transcript. `backend/agent/src/agent/` holds events, a loop skeleton, a scheduler and a permission protocol — but nothing that holds what was said across a run, and nothing that enforces the one invariant every later milestone leans on: **every tool call has a matching result**.

Layer 1 declines the job on purpose. `ai/tool_result.go:135-136` states it verbatim: "Pairing them is Layer 2's job: the invariant that every call in a transcript has a matching result is V-OUT-02." `ai.ToolCall.ID()` and `ai.ToolResult.CallID()` are the correlation key; nothing above them checks it. AG-12 is the first and correct owner.

Two consequences today:

1. **An orphaned call is unrepresentable-as-invalid.** Any caller can assemble `[]ai.Message` containing a tool call with no result and hand it to a provider. Layer 1 validates each part and each message; no contract validates the sequence.
2. **Interruption has no defined recovery.** A turn cut off after calls were issued but before results arrived leaves a transcript that cannot legally be sent again, and there is no mechanism that repairs it.

AG-12 builds the store, enforces the invariant **at the boundary and not at call sites**, and synthesizes results for orphaned calls after an interruption.

## Scope

### In

- **AG-12.1** — a `History` type in `backend/agent/src/agent/`: an ordered transcript of Layer 1 values across a run, appended in conversation order, read back in order.
- **AG-12.1** — **exactly one commit path**. Every public route that can extend or mutate the transcript funnels through a single unexported validating primitive; no privileged bypass for internal callers. The C1 lesson (doc 0001:376) applied to history.
- **AG-12.1** — read-only views: the transcript arrives as unmodified Layer 1 values with **stable entry identity**, and no read aliases internal storage.
- **AG-12.1** — **seeded construction** that re-runs the same validation over a pre-existing transcript; a valid seed is accepted, an unmatched-pair seed is rejected typed. This is the seam session resume and next-run model switching stand on, **frozen at this handoff**.
- **AG-12.1** — two typed rejections: appending a result whose call identity has no prior call; committing a state where a call has no result once the turn closes.
- **AG-12.2** — **orphan synthesis**: every orphaned call is closed with a result marked an interruption artifact, distinguishable from a real result. Idempotent and total — closes exactly N on the first pass, changes nothing on the second.
- The `L2C-07` doc-contract row plus its `expectedLayer2ContractRows` entry, landed in the same PR (`R-AGP-002` closed-amendment rule).

### Out — deferred but related, with the owner named

| Deferred | Owner |
|---|---|
| Wiring `History` into `Turn` / `Schedule` | **AG-13** (run driver). The charter places AG-12 beside wave 2; its only in-document edge is the scaffold. `loop.go` and `scheduler.go` stay **byte-unchanged**. |
| Persistence / session reload of a transcript | **Layer 3**. Charter "Out of scope" line, binding. |
| Compaction's interaction with the invariant | **AG-18.2** re-proves the invariant post-compaction. Nothing here anticipates it. |
| Cancellation semantics that *produce* an interruption | **AG-14**. AG-12 repairs the transcript an interruption left; it does not detect or cause one. |
| Context-window accounting over the transcript | **AG-17**. |
| Steering-message queueing at turn boundaries | **AG-13.2**. |
| Adding a rule class to `ai/validation.go` | **Not this milestone.** See decision 2. `backend/agent/src/ai/**` is consumed, never edited. |
| The `_ = sched.Schedule(...)` result-discard at `loop.go:265` | Pre-existing; **untouched**, as AG-11 also fenced it. |

## Decisions taken here (inherited, not re-opened)

### 1. `L2C-07` doc-contract row — **YES**

Evidence corrects the exploration. `doc.go:29-34` shows AG-04, AG-05 and AG-06 each added a row (`L2C-04`, `L2C-05`, `L2C-06`); AG-07 through AG-11 added none. The discriminator is not "per milestone" — it is **"does this milestone declare a new package-wide guarantee, or implement behavior inside one already declared?"** AG-07..AG-11 all ride `L2C-03`'s "the event stream is the only upward contract". History does not: it is a **second upward surface** — a transcript read by the loop and by a future session — that no existing row covers, and "one commit path, no privileged bypass" is exactly the machine-checkable, package-wide shape `L2C-01`..`L2C-04` are written in.

*Criterion to overturn in `sdd-design`*: if the guarantee turns out to be expressible only as behavior of one type rather than as a constraint on the package, drop the row and say so explicitly.

### 2. Typed pairing rejection lives in **`ai`'s existing violation machinery, with existing rule classes**

Reuse `ai.Invalid` / `ai.Violation` / `ai.At` / `ai.FirstFailure`. **Do not add a class to `ai/validation.go`. Do not introduce an `agent`-local violation type.**

| Rejection | Class | Justification |
|---|---|---|
| A result names a call the transcript does not carry | `ai.ErrUnresolvedReference` | Its own doc (`validation.go:93-96`): "a value naming something the request does not declare". Exact fit. |
| A call has no result once the turn closes | `ai.ErrEmpty`, positioned at the call's result slot | The required value — the matching result — is absent. The **position** carries the specificity; the class carries the kind. |

Three reasons: (a) `agent/run_events.go`'s `NewRunEnd` and `agent/event.go`'s `CheckEmit` already call `ai.Invalid`/`ai.FirstFailure` directly — the rule is Layer 2's, the error vocabulary is shared, and that relationship is established; (b) this milestone's own charter names the Layer 1 message contracts **frozen input**, so adding Layer 1 diff surface to a Layer 2 milestone needs a justification that reuse defeats; (c) a third typed-error vocabulary alongside `ai.Violation` and `agent.Failure` would make "which error do I match?" unanswerable for a consumer.

*`sdd-design` may substitute a different **existing** class for the second row if it finds a better fit. It may not extend `ai/validation.go` under this change.*

### 3. Seeded-construction shape — frozen at this handoff

Stated as obligations; `sdd-design` owns the Go rendering (doc 0003's authoring constraint).

- Accepts **an ordered sequence of Layer 1 messages and nothing else**. No caller-supplied entry identity — that would be the C1 back door.
- Returns a constructed history **and an error**. The zero value is not a usable history.
- Runs the **same** validation the append path runs, over the whole seed, and reports the **position** of the first offending entry so a resume can name what broke.
- **Entry identity is derived deterministically from ordinal position in the transcript.** The same seed therefore yields the same identities across processes — Layer 3 resume gets identity stability without any caller being able to mint one. Identity is exposed read-only.

### 4. Synthesized-vs-real distinguishability — the entry envelope, and the tension is apparent, not real

`ai.ToolResult` is `{callID, content, failed}` and frozen. Encoding a sentinel in `content` is rejected outright: content is stored byte-for-byte, a real tool can emit the same bytes, and string-sniffing is forgeable.

**Resolution**: history stores each transcript position in an opaque **entry envelope** — unexported fields, read-only accessors — carrying the unmodified Layer 1 value plus entry identity plus an **origin discriminator** (`appended` | `synthesized`). This is the `agent.Event` shape the package already uses four times.

The two scenarios never conflicted: AG-12.1 scenario 3 **already requires an envelope**, because a bare `ai.Message` cannot carry "stable entry identity" — its `MessageID` is Layer 1's and history does not mint it. "History exposes Layer 1 values" is satisfied because read-back yields the unmodified `ai.Message` / `ai.ToolResult`; the discriminator rides the envelope that scenario 3 forced into existence anyway.

Corollary: `ToolResult.Failed()` is **not** the discriminator. A synthesized result may set it, but a real failing tool sets it too.

## Capabilities

### New

- **`agent-history`** — the transcript store, the pairing invariant, one commit path, seeded construction, orphan synthesis. IDs `R-HIS-0NN` / `S-HIS-0NN`, bites `S-HIS-0NN`. Prefix `HIS` verified free: zero `[RS]-HIS-` matches repo-wide, and it collides with none of `AEV`, `AGE`, `AGP`, `AGM`, `AGV`, `ATT`, `TLS`, `APE`, `PRH`, `LSK`, `AMT`. Becomes `openspec/specs/agent-history/spec.md` at archive.

### Modified

- **`agent-event-envelope`** — `S-AEV-122` (`spec.md:217`) asserts `expectedLayer2ContractRows` "contains 6 rows (`L2C-01`..`L2C-06`)". Adding `L2C-07` makes that count false. A delta is **mandatory**, not optional: this is precisely the owning-spec-omission staleness this repo has been bitten by.
- **`agent-package-scaffold`** — `S-AGP-012` / `S-AGP-014` (`spec.md:53,55`) carry "as of AG-04, four rows" parentheticals. Their normative wording is unaffected; the delta is a back-annotation recording AG-12's row so the parenthetical stops drifting.

No other spec changes. `agent-loop-skeleton`, `agent-tool-scheduler` and `agent-turn-termination` are untouched because AG-12 does not wire into the loop.

## Approach — exploration Approach 1

An opaque `History` struct with unexported storage and one private validating commit primitive. Every public route — append, seeded construction, orphan synthesis — calls it. Orphan synthesis is a **method** that routes through the same primitive, so AG-12.1's "an orphaning sequence through **any** public route is rejected" scenario covers it for free instead of creating a second unaudited door.

This reapplies the idiom the codebase already trusts four times — `ai.Part`, `agent.Event`, `agent.Result`, `agent.Failure` — and matches `VL2-HAR-01`'s "exactly one commit path, with no privileged bypass for internal callers" almost verbatim.

**Rejected — `History` as an interface**: persistence is charter-excluded, so the second implementation the interface would serve does not exist. Every concrete boundary type in `agent/` from AG-04 to AG-11 is a struct until a second implementation actually arrived. Speculative generality.

**Rejected — orphan synthesis as a free function**: it needs its own route back through the validating path or becomes a bypass. Approach 1 with extra steps and a new hole.

Tests live in an **external** `agent_test` package. The "any public route" scenario is only meaningful against the public surface — an in-package test could reach the private primitive and prove nothing.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agent/history.go` | **New** | `History`, the entry envelope, the single commit primitive, seeded construction, `SynthesizeOrphans` |
| `backend/agent/src/agent/history_test.go` (`package agent_test`) | **New** | AG-12.1's four scenarios + AG-12.2's two, table-driven |
| `backend/agent/src/agent/doc.go` | **Modified (substrate)** | `L2C-07` row appended |
| `backend/agent/src/agent/doc_contract_guard_test.go` | **Modified (substrate)** | `expectedLayer2ContractRows` entry, same PR |
| `backend/agent/src/agent/loop_test.go`, `loop_hook_test.go` | **Modified** | `filterOutLoopFiles` / `filterOutLoopHookFiles` widened by **exact filename** for the new files, kept byte-in-sync (the AG-11 discipline) |
| `openspec/specs/agent-event-envelope`, `agent-package-scaffold` | **Delta** | Row-count and back-annotation deltas |
| `loop.go`, `scheduler.go`, `tool.go`, `event.go`, `event_descriptor.go`, `turn_events.go`, `run_events.go`, `failure.go`, `go.mod`, `go.sum`, `backend/agent/src/ai/**` | **NOT TOUCHED** | No loop wiring, no new dependency, no Layer 1 edit |

## Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 1 | Ordinal-derived entry identity breaks under compaction, which removes entries | Med | Compaction is charter-excluded and **AG-18.2 re-proves the invariant post-compaction**. Recorded here so AG-18 inherits a known, named constraint rather than a surprise. |
| 2 | The seeded-construction shape drifts before AG-13 consumes it | Med | Frozen in decision 3; `sdd-design` renders it, does not re-open it. AG-13's proposal cites this section. |
| 3 | `ai.ErrEmpty` is an imperfect class for "call unclosed at turn close" | Med | Position carries the specificity. `sdd-design` may substitute another **existing** class; extending `ai/validation.go` is forbidden under this change. |
| 4 | The entry envelope is read as contradicting "history exposes Layer 1 values" | Med | Decision 4 reconciles the two scenarios explicitly and in the spec, not implicitly in code. `sdd-spec` MUST carry the reconciliation as prose. |
| 5 | Adding `L2C-07` silently breaks `S-AEV-122`'s row count | High (certain) | Named a **mandatory** delta above; `sdd-verify` re-reads `agent-event-envelope/spec.md:217` against the shipped table. |
| 6 | Forgetting the new test files in both substrate filters | Med | AG-11's recorded lesson; exact-filename suffixes only, no wildcard, both filters byte-in-sync. |
| 7 | Review budget: forecast 1100–1600 lines against 1000 | High | `size:exception` pre-authorized. If `sdd-tasks` forecasts above ~1600, slice at the AG-12.1 / AG-12.2 leaf boundary — AG-12.1 is independently deliverable and AG-12.2 depends on it. |

## Rollback Plan

Single revert of the AG-12 merge commit. `history.go` and `history_test.go` are deleted; `doc.go` returns to six rows and `doc_contract_guard_test.go` to its six-entry table; both substrate filters return to their pre-AG-12 filename lists; the two spec deltas are dropped and `S-AEV-122`'s six-row claim is true again.

Because AG-12 wires into nothing — `loop.go`, `scheduler.go` and every Layer 1 file are byte-unchanged — the revert is **additively clean**: no consumer exists to break, no migration, no data, no `go.mod`/`go.sum` change, nothing outside `backend/agent`. Re-running `cd backend/agent && make test` at the parent commit confirms zero regression.

The one forward-looking cost: AG-13 is blocked by AG-12, so a revert re-blocks AG-13. That is a scheduling consequence, not a correctness one.

## Changed-line forecast

| Component | Estimate |
|---|---|
| `history.go` (this package's doc density) | 250–350 |
| `history_test.go` (6 scenarios + no-bypass audit + bites) | 350–500 |
| `doc.go` + guard table | ~8 |
| Substrate filter widening (2 files) | ~6 |
| SDD markdown (spec, design, tasks, apply-progress, verify-report) | 450–700 |
| **Total (authored, additions + deletions)** | **1100–1600** |

Against the 1000-line budget: **exceeded**. `size:exception` is pre-authorized for this PR, which carries only AG-12. The SDD markdown counts toward the attempt budget — `sdd-tasks` must forecast against the **full** diff, not the Go diff.

## Dependencies

- **AG-03** (archived) — the `agent` package, its import guard, its ambient-authority guard, `doc.go`'s guarded-row machinery.
- **AI-40 / Layer 1** — `ai.Message`, `ai.ToolCall`, `ai.ToolResult`, `ai.Violation` and the closed rule-class set. Consumed, never edited.
- **`VL2-HAR-01` / `VL2-COR-10` / `VL2-COR-11`** (archived, `2026-08-11-cachicamas-agent-contract-vocabulary`) — history, transcript, pairing invariant. Binding vocabulary.
- **doc 0003:1227-1292** — the AG-12 charter and its two Gherkin leaves.

## Success Criteria

- [ ] `cd backend/agent && make test` green with `-race`; all four AG-12.1 scenarios and both AG-12.2 scenarios closed
- [ ] Appending a result whose call identity has no prior call fails with an `ai.Violation` matching `ai.ErrUnresolvedReference`, at a position naming the offending entry
- [ ] Committing a state where a call has no result at turn close fails typed
- [ ] An orphaning sequence constructed through **every** public route is rejected — enumerated in the test, not asserted in prose
- [ ] A valid seed constructs; an unmatched-pair seed is rejected typed with the offending entry's position
- [ ] Read-back yields unmodified Layer 1 values, in order, with stable entry identity, and no read aliases internal storage
- [ ] Synthesis over N orphans closes exactly N on the first application and produces a byte-identical transcript on the second
- [ ] A synthesized result is distinguishable from a real one **by the envelope's origin, not by content inspection and not by `Failed()`**
- [ ] `L2C-07` lands in `doc.go` and `expectedLayer2ContractRows` in the same commit; the guard bite RED-records before GREEN
- [ ] `agent-event-envelope` delta written — `S-AEV-122`'s row count amended, not silently broken
- [ ] `loop.go`, `scheduler.go` and every file under `backend/agent/src/ai/` byte-unchanged; `go.mod`/`go.sum` unchanged
- [ ] `make lint` clean (after `golangci-lint cache clean`), `make build` clean, `make vuln-check` clean — `vuln-check` is NOT in `make all`
