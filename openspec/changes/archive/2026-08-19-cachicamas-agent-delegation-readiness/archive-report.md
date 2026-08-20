# Archive Report — `cachicamas-agent-delegation-readiness` (AG-19)

> **SDD ARCHIVE STATUS: CLOSED.**
> **Milestone**: AG-19 — Prove re-entrancy and delegation readiness. Layer 2, **Wave 5**, milestone **19 of 24**.
> **Charter**: `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:1793-1862`.
> **Closes**: **G7**'s structural half (R-14); v2 § 8 **seam 12** — re-entrancy cannot be added later.
> **Branch**: `feat/agent-layer2-wave5-ag19`, 10 commits off `558641f3`. Worktree `cachicamas-worktrees/ag19`; the base checkout was never written to.
> **GitHub PR**: opened by the orchestrator after this archive commit. Merge status at time of writing: **not yet merged**.

## What shipped

A **publishing seam** — the smallest door that lets a tool reach the parent's event stream — plus proof of four structural properties under a nested run. The seam is the only production surface:

| File | Change |
|---|---|
| `backend/agent/src/agent/delegation_seam.go` | **new**, ~178 lines. Four exported identifiers: `DelegationSeam` (interface), `DelegationSeamFrom` (accessor), `ErrDelegationRevoked`, `ErrDelegationInadmissible` |
| `backend/agent/src/agent/scheduler.go` | +7/−1 in `executeCall`: install the seam on the call's `ctx`, `defer` its revocation |

Everything else is test-only, in `package agent_test`, which production code **cannot import** — so "no subagent tool ships in v1" (AG-02's verdict, `0003:1794`) is enforced by the compiler, not by convention.

## The four proven properties

1. **Nested run, parent-identified.** A child `Harness` runs to completion inside a tool call; its `subagent_started`/`subagent_ended` bracket crosses onto the parent's stream. Attribution is a **one-hop walk**, not a per-event stamp — only 3 of 25 event constructors accept a parent, so the literal reading would have required rewriting the whole construction surface.
2. **Sibling isolation.** Two sibling tools each hosting a child run interleave with no cross-talk, proven under `-race`, using **two distinct `Harness` values** (never two concurrent `Run` calls on one value, which `agent-cancellation-tree:188` excludes).
3. **Leaf-first cancellation.** Inherited through the **existing** context tree at zero plumbing cost: a tool that threads its received `ctx` into `child.Run` gets propagation free. Ordering is asserted as `index(subagent_ended) < index(parent run_end)`, plus `errors.Is(childErr, ErrInterrupted)` and a not-`DetachedCallError` check.
4. **Cost and permission crossing.** See the two decisions below.

## The two decisions that shaped the milestone

### Child cost NEVER folds into the parent — and the seam enforces it in production

`R-CST-004` defines a run's cumulative as the sum over `cost_turn` events emitted **within that run bracket**, and forbids a second writer to the run's event path. The parent's per-attempt forwarder does `if ct, ok := ev.CostTurn(); ok { total.add(ct) }` with **no run-identity filter** — so any child `cost_turn` routed through the parent's `turnSink` would fold **silently**, shipping a requirement violation that a green suite would never reveal.

The seam therefore **refuses the `cost_turn` kind in production code** (gate 5). "Child cost aggregates into the parent's cumulative" is satisfied as a **consumer-side reconstruction** over two independently validated streams, matching the charter's own framing — "a frontend can show both *this subagent cost X* and *the run cost Y*" — two displayed numbers, never one merged Layer-2 event.

Measured, both directions: shipped tree reports parent-own **15** against child spend **77**; with gate 5 removed the parent's terminal `cost_session` inflates to **92** and a child `cost_turn` leaks. The refusal is load-bearing, not decorative.

### Admissibility is registry-gated, because a descriptor-only rule is not total

The proposal's predicate read the descriptor alone. A **zero `Event`** (kind 0, no registry row) has zero-value descriptor fields, so it passed every gate — and `CheckStream` then *skips* unregistered kinds with a bare `continue` (`stream_check.go:112-118`), leaving junk on a stream the validator calls valid. `sdd-design` added a registry-membership gate ahead of the descriptor gates. Verdict table: **refuse 6, admit 19**, total over all 25 kinds.

## Verification — three rounds, and what each caught

The production seam was correct from the first apply. **Every blocker across all three rounds was a test that could not fail.**

| Round | Verdict | Findings |
|---|---|---|
| 1 | **FAIL** | 6 blockers: a test body ending at a comment reading "grep-verified at apply time"; three `errors.Is(x, x)` tautologies; an empty subtest; two spec scenario sentences **disproved by overlay**; two scenarios with no covering test. Plus the decisive WARNING: `cost_test.go` computed the parent's sum **unfiltered by run identity**, so with the gate removed *both sides* of the equality moved 15→92 together and it still passed — a test that certified the defect |
| 2 | **FAIL** | 1 blocker: the round-1 correction **re-encoded the same defect class in the scenario round 1 named**. `S-DEL-001`'s publish clause searched for any event matching kind-plus-run — a shape the parent's own turn-2 text response already emits. Overlaying `Publish` to discard dropped matches 2→1 and the test still passed |
| 3 | **PASS WITH WARNINGS** | 0 blockers, 0 CRITICAL. 18/22 requirements, **31/36 scenarios COMPLIANT**, 5 PARTIAL, **0 UNTESTED** |

### The defect found after verify closed

**Commit `c4e4455f`, found by the orchestrator, not by any verify round.** Round 3 recorded it as an accepted note; that classification was wrong and is superseded here.

The `S-DEL-015` and `S-TLS-020` diff guards resolve their base ref from `git merge-base HEAD origin/main`. Once AG-19 lands on `main`, that resolves to `HEAD` itself, the diff empties, and the anti-vacuity floor added for round 2's WARNING-9 becomes a hard `t.Fatal` — **reddening main's suite on merge**.

Both guards assert an **absence**: that this change's own added lines introduce no forbidden symbol. An empty diff satisfies that trivially. The floor was **presence-guard logic applied to an absence guard**, and it diverged from the AG-07 substrate guard (`loop_test.go`), which has passed on an empty diff for five milestones.

`baseRefIsHEAD` now detects the post-merge condition and degrades to a logged skip; the floor stays in force on-branch, where an empty diff does signal a real problem. WARNING-9 is **scoped, not reverted**. Proven in three states: post-merge PASS+SKIP (was FAIL+FAIL), on-branch clean `ok`, on-branch with a scratch `context.WithCancel` in `delegation_seam.go` still FAILs with the intended diagnostic.

## Evidence

- `cd backend/agent && go test -race -count=1 ./...` → **exit 0**, all 12 packages `ok`, zero `(cached)` markers, ~175s uncached. Re-run independently by the orchestrator after `c4e4455f`.
- `-count=1` is mandatory: the real uncached suite is ~170s, so a sub-second pass is a cache artifact and not evidence.
- `gofmt -l` clean on every file this change touched. `go vet ./...` clean.

## Size

| Category | Lines |
|---|---|
| Production Go | +185 / −1 |
| Test Go | +3417 / −2 |
| `docs/` | +2 / −2 |
| **Counted total (excl. `openspec/`)** | **3609** |
| `openspec/` | +2788 (excluded — a working folder, per the user's explicit instruction) |

Shipped under an accepted **`size:exception`**, pre-authorised by the user against a 1000-line budget. The forecast was 1020–1650; the overrun is almost entirely test Go (3417 of 3609), which is the expected shape for a milestone whose deliverable is *proof* — the charter forbids shipping the product a subagent tool would be.

## Spec promotion

**New capability** `agent-delegation-readiness` promoted to `openspec/specs/agent-delegation-readiness/spec.md` — `R-DEL-001`…`R-DEL-010`, `S-DEL-001`…`S-DEL-025` (bites `S-DEL-020`…`S-DEL-023`), `NFR-DEL-001`…`NFR-DEL-005`. The header states the allocated **range**, never a total, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`).

**12 delta specs merged**: `agent-cost-events`, `agent-cancellation-tree`, `agent-permission-protocol`, `agent-event-envelope`, `agent-protocol-events`, `agent-event-delivery`, `agent-tool-scheduler`, `agent-v1-scope`, `agent-retry-failover`, `agent-loop-skeleton`, `agent-run-driver`, `agent-compaction`.

Two shipped requirements were **amended because AG-19 genuinely falsifies them**:

- **`R-CST-005`** read "zero or more estimates, followed by **exactly one** carrying `CostLabelFinal` emitted immediately before the run-close event." Mirroring the child's final `cost_session` **mid-bracket** falsifies it — not through any defect, but because the sentence was written before a producer of this shape existed. Re-scoped by `Event.Run()`. `S-CST-009` was amended in the same block, since it restates the phrase twice.
- **`R-AEV-003`** read "an event belonging to a delegated harness MUST carry its parent identifier." Only 3 of 25 constructors accept a parent, so every mirrored child event violates the literal reading. Scoped to **construction**, with the one-hop walk stated as its replacement.

Three non-requirement rows were **corrected rather than annotated**, because each named AG-19 as owner of something AG-19 does not deliver:

- `agent-retry-failover` — subagent-scoped retry **re-owned to post-v1**. AG-19 ships no retry surface at all.
- `agent-run-driver` — the row bundling "a production subagent tool" with "nested or child runs" **split**: the second half closes, the first does not.
- `agent-compaction` — the parent-aggregation row **answered** as a permanent NO at Layer 2, correcting the proposal's "no delta" assessment.

`agent-loop-skeleton` gained `R-LSK-007`/`S-LSK-031` as an **ADDED** requirement rather than a `MODIFIED` reproduction of `R-LSK-004` — a partial `MODIFIED` block silently drops whatever it omits, a failure this repository has recorded.

### Count-assertion check (task 5.2)

`openspec/specs/agent-permission-protocol/spec.md` lines 7 and 15 both assert **"12 requirements → 13 spec scenarios + 4 bites"**. Its delta is deliberately **scenario-free** precisely so those stay true. **Verified after the merge: both still hold.** No other promoted spec's count assertion or allocated-range line changed.

## Carried forward — accepted, not hidden

**Five PARTIAL scenarios** prove a weaker proposition than they state. Zero scenarios lack a covering test; these five have passing tests asserting something narrower: `S-DEL-014`, `S-AGE-029`, `S-DEL-025`, `S-AGS-063`, `S-AGS-064`. Accepted as non-blocking by explicit orchestrator acknowledgement before archive.

**On the persisted verify envelope**: it reads `verdict: fail` with `blockers: 0`. This is **not** a real failure. `gentle-ai sdd-verify-validate` gates on *evidence completeness* — it requires 22/22 requirements and 36/36 scenarios — not on blocker count. A round carrying accepted PARTIALs cannot emit a passing token without falsifying a count, and the verify agent correctly **refused to inflate a count to buy one**. The honest counts stand and the token gives way.

Round-1 WARNING-2/3/4/5/6, its SUGGESTION-1..4, and round-2's three SUGGESTIONs remain open by agreement.

## Pre-existing issues AG-19 did NOT fix

Recorded, deliberately untouched — out of this milestone's scope:

1. **Two `errors.Is(x, x)` tautologies** at `cancellation_events_test.go:119,122`, traced to commit `045f8095` (**AG-14**). The same defect class round 1 blocked on, shipped five milestones ago and surviving every review since. The shape is evidently not confined to the instances verify found here.
2. **Stale citations** in the promoted `openspec/specs/agent-loop-skeleton/spec.md` and `agent-turn-termination/spec.md` — the same `loop_test.go`/`loop_hook_test.go` range drift, unfixed since AG-11 through five subsequent archived milestones.
3. **15 files fail `gofmt -l`** under the current toolchain — baseline drift, unrelated to this change.

## Post-merge note

Both AG-19 diff guards are **branch-scoped by construction**. After merge they detect that the base ref equals `HEAD` and degrade to a logged skip, which is correct: the absence they assert is about what *this change* introduced, and once merged there is nothing left to defend. This mirrors the AG-07 substrate guard's long-standing behaviour.

## Layer 2 status after AG-19

**19 of 24 milestones shipped.** Wave 5 (`AG-19` · `AG-20`) is **open** — AG-19 has shipped, **AG-20 (the hook taxonomy) has not**. AG-20 is next.
