# Proposal — Define the agent event envelope and ordering invariants

> **Change**: `cachicamas-agent-event-envelope`
> **Milestone**: AG-04 — Define the agent event envelope and ordering invariants (Layer 2, Wave 1) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-04--define-the-agent-event-envelope-and-ordering-invariants), `0003:410-517`
> **Nodes**: AG-04.1 envelope + validation `[leaf]` · AG-04.2 run + turn lifecycle `[leaf]` · AG-04.3 invariant pins `[leaf]` · AG-04.4 every-kind-constructible `[guard]`
> **Status**: proposed · **Date**: 2026-08-12 · **Driver**: braejan · **Project**: cachicamas (witsaba)
> **Branch**: `feat/agent-layer2-wave1-ag04` · **Predecessor artifact**: `explore.md` (this change)
> **Traces to**: R-04, R-05 (`0003:2247`) · **Depends on**: AG-01, AG-03 · **Blocks**: AG-05, AG-06, AG-07 (`0003:419`)
> **Scope**: the first production Go code in `backend/agent/src/agent/`. Zero new dependencies. Zero `go.mod` change.
> **Delivery**: single PR, `size:exception` pre-authorized by the user against the 1000-line budget

---

## Intent

Layer 2 exists but declares nothing: `backend/agent/src/agent/` holds `doc.go` (19 lines, package clause plus a machine-checked contract comment) and three `_test.go` guards, nothing else. AG-04 is the AI-14 of this layer — it ships the envelope every agent event travels in, and it is the first milestone to put a type, a constant or a function into Layer 2 at all.

**Why now.** Both dependency edges are closed. AG-01 (`agent-event-delivery`) decided the carrier, the observer lanes and the three nested ownership scopes, and states in its own words what AG-04 inherits and what stays AG-04's: *"Not inherited, because it is `AG-04`'s: event kinds, payloads, ordering invariants, the envelope's contents"* (`openspec/changes/archive/2026-08-11-cachicamas-agent-event-delivery/decision.md:555`). AG-03 (`agent-package-scaffold`) shipped the package and its two boundary guards as PR #161, merge `8f2ce2a6`. Downstream, AG-04 blocks AG-05, AG-06 and AG-07 (`0003:419`) — and AG-07 is the wave-2 gate, so every event family and the entire loop sit behind this milestone.

**Why the envelope and its validator land together.** No producer exists until wave 2, so the invariants have no runtime that could exercise them. The charter's answer is the **stream-contract validator**: *"the reusable checker these tests (and later the AG-23 kit) run over any event sequence… the invariants must be assertable over hand-built sequences, and the validator is the named surface that makes them so"* (`0003:417`). An envelope shipped without it would be a set of unenforced conventions until wave 2 — exactly the shape the 2026-07-30 review's C3/C4 scar tissue was written against (`0003:412`).

---

## Settled inputs (not open questions)

Resolved upstream. Later phases implement them; they do not re-litigate them.

| # | Input | Evidence |
| --- | --- | --- |
| 1 | Layer 2 **wraps** events, ordering and failure — it does not reuse Layer 1's as-is. Layer 2's ordering is an independent agent-level counter, explicitly not `ai.Sequence`. Message identity, tool-call identity, finish reasons and usage are reused as-is | `openspec/specs/agent-contract-vocabulary/spec.md` `S-AGV-019`/`S-AGV-020`/`S-AGV-021` |
| 2 | "Per-consumer-stream" ordering **is AG-01's lane mechanism** — one canonical internal stream, one independently-ordered receive-only lane per attached consumer, each contiguous and 1-based | `.../2026-08-11-cachicamas-agent-event-delivery/decision.md` § 5 (`:224-313`) |
| 3 | The run and turn brackets have **named owners**: the harness owns and alone closes the run scope; the loop owns and alone closes the turn scope and emits turn-end on every exit path | same `decision.md:352-354`, `:550-555` |
| 4 | The validator is **production-exported, not a private test helper** — `VL2-EVT-16` requires it be *"reused wholesale by the Layer 3 readiness contract's kit (`VL2-SEAM-14`)"* | `.../2026-08-11-cachicamas-agent-contract-vocabulary/decision.md:177` |
| 5 | AG-04 registers **exactly two** of v2 § 4.3's eight families: run lifecycle (`VL2-EVT-02`) and turn lifecycle (`VL2-EVT-03`), with their typed outcomes (`VL2-EVT-10`, `VL2-EVT-11`) | same `decision.md:163-164`, `:171-172` |
| 6 | Delivery: single PR with `size:exception`, pre-authorized this session. **Not reopened by this proposal** | session preflight |

---

## Scope

### In scope

| # | Deliverable | Node |
| --- | --- | --- |
| 1 | **The envelope with validation** — kind derived from payload (never stored), a validation gate rejecting a nil or mismatched payload, and run/turn/parent identity readable from an external package. The parent identifier exists now because explicit nesting cannot be retrofitted; delegation fills it at AG-19 | AG-04.1 |
| 2 | **The ordering mechanism** — the envelope's public stamping surface, independent and 1-based per consumer lane (settled input 2), checked under `-race` | AG-04.1 |
| 3 | **Run lifecycle events** — one run-start preceding everything, one run-end following everything and carrying the run's typed outcome (completed, interrupted, failed) | AG-04.2 |
| 4 | **Turn lifecycle events** — turn-start/turn-end pairs nesting strictly inside the run bracket, never overlapping, turn-end distinguishing "model finished" from "turn aborted" by typed outcome | AG-04.2 |
| 5 | **The stream-contract validator** — production-exported (settled input 4), taking a finite ordered sequence offered after the fact, accepting it only if every envelope invariant *and* every lifecycle bracket holds, including "nothing follows the terminal" | AG-04.2 (the surface all four nodes assert through) |
| 6 | **Invariant pins** — a delta kind has no route to an accumulated-message payload; a failure payload exposes category and cause as inspectable values aligned with Layer 1's taxonomy, with no code path assigning meaning to a message string | AG-04.3 |
| 7 | **Ordering invariants stated in the package documentation** and pinned by test, plus the membership criterion the charter names — *"if it is not on the stream, no frontend can render it and no log can reconstruct it"* (`0003:418`) — stated in the package docs as the criterion later families are judged by | AG-04.1, AG-04.3 |
| 8 | **The every-kind-constructible guard** — iterates every kind AG-04 itself registers and constructs a valid instance of each through the public surface; closes on a recorded bite (a scratch kind with no constructible payload), never on green | AG-04.4 |
| 9 | Recorded evidence: `make test` and `make lint` in `backend/agent/`, pre- and post-change; the guard bite red; a RED-then-GREEN record per scenario | all |

**Total normative behavior**: 11 Gherkin scenarios (3 + 3 + 2 + 1 bite proof), `0003:438-514`. Spec restates them; it does not reduce them.

### Out of scope

**Deferred but related — each has a named owner, so none is a gap** (`openspec/config.yaml` `rules.proposal`):

- **Message lifecycle and tool execution families** (`VL2-EVT-04`, `VL2-EVT-05`) — **AG-05**. AG-04's charter says so directly (`0003:420`).
- **Permission, cost, delegation and compaction families** (`VL2-EVT-06`…`VL2-EVT-09`) — **AG-06**, same charter clause.
- **Delivery mechanics beyond what AG-01 decided** — the carrier, buffering posture, observer attachment and the upward path are AG-01's, already merged. *"A downstream milestone that finds itself deciding a delivery property is proposing an amendment to this artifact, not exercising a judgement call"* (`.../agent-event-delivery/decision.md:591`).
- **Any loop or harness behavior** — no producer, no driver, no turn execution. AG-07 onward.
- **Partial invariant closure — AG-04 closes no envelope invariant alone.** The traceability spine (`0003:2203`) reads: invariant 1 → AG-04.3 **+ AG-05.1**; invariant 2 → AG-04.1 **+ AG-19.1**; invariant 3 → **AG-01.1 + AG-20.2, AG-04 absent**; invariant 4 → AG-04.3 **+ AG-11.2**. AG-04's tests and its guard MUST NOT be written or accepted as if they close invariant 3 (observer asynchrony) or the full loop-level typed-error path. The guard proves construction-time exhaustiveness of the kind registry — nothing wider.
- **`agenttest` convenience wrappers over the validator** — Layer 1's split is `ai.CheckStream` (production) plus `agenttest.RequireValidStream` (test wrapper, AI-22). Whether Layer 2 gets the second half at AG-04 is open decision 4 below; the charter's dependency edges (AG-01, AG-03 only, `0003:419`) suggest not.
- **`src/coding/`, `src/cmd/`** — doc 0004. Not created, not stubbed; their absence keeps AG-03's forbidden-prefix rows testable.
- **Any new dependency.** No `go.mod` or `go.sum` edit, so `openspec/AGENTS.md` rule 5 is not triggered.
- **AG-03's W5 follow-up** (the ambient guard's alias and dot-import branches lack permanent covering tests, `.../2026-08-12-cachicamas-agent-package-scaffold/archive-report.md:91-93`, deferred "to AG-04 onward"). Named here so it is not silently dropped; it is guard-hardening work with no dependency on the envelope and does not belong in an already-oversized PR.
- **CI.** Still absent. Every gate runs when a human runs `make test` in `backend/agent/`.

---

## Capabilities

### New capabilities

- `agent-event-envelope`: the Layer 2 envelope (identity, derived kind, validation), its per-lane ordering mechanism, the run and turn lifecycle families with their typed outcomes, the typed-failure surface, the stream-contract validator, the documented ordering invariants and membership criterion, and the every-kind-constructible guard with its bite as closing condition. Requirement ids `R-AEV-0NN`, scenario ids `S-AEV-0NN` — **verified unused repo-wide** (`R-AGE-`/`S-AGE-` are taken by `agent-event-delivery`, `openspec/specs/agent-event-delivery/spec.md:22`).

### Modified capabilities

- `agent-package-scaffold` — **conditional on open decision 3 only.** If AG-04 adds a fourth guarded row (an envelope/L2C-04 clause) to `doc.go`, the committed expectation table pinned by `S-AGP-012..015` (`backend/agent/src/agent/doc.go:12`) changes in the same PR, and that spec takes a delta. If design decides no new row is needed, this list is empty.
- `agent-contract-vocabulary` — **not modified.** AG-04 consumes its register rows (`VL2-EVT-01`…`VL2-EVT-03`, `VL2-EVT-10`, `VL2-EVT-11`, `VL2-EVT-15`, `VL2-EVT-16`) as inputs; it changes none of them.

---

## Approach

```
backend/agent/src/agent/
├── doc.go                      # possibly MODIFIED — ordering invariants + membership criterion;
│                               #   a fourth guarded row only if open decision 3 says yes
├── doc_contract_guard_test.go  # MODIFIED only alongside a new guarded row
├── import_boundary_test.go     # UNCHANGED — AG-04 imports only src/ai + stdlib, already allowed
├── ambient_authority_test.go   # UNCHANGED — no I/O introduced
└── <new production files>      # NEW — envelope, kind derivation + registry, ordering,
                                #   run/turn payloads, typed-failure surface, validator
```

1. **AG-04.1** ships the envelope, its validation gate and the ordering mechanism. Structurally mirrors Layer 1's derive-don't-store kind (`backend/agent/src/ai/event.go:33-72`) and its `CheckEmit` gate (`event.go:321-368`) — the same mechanism shape, a different and independent vocabulary.
2. **AG-04.2** ships the run and turn families with their typed outcomes, and the validator that brackets them. Layer 1's `CheckStream`/`StreamReport` (`backend/agent/src/ai/stream_check.go:64-118`) is the closest precedent, but it enforced only single-level block discipline; Layer 2 adds the turn-inside-run nesting dimension (open decision 1).
3. **AG-04.3** pins the two invariants AG-04 co-owns. The typed-failure surface aligns with `ai.FailureCategory` and the pre-stream/mid-stream distinction already shipped at `backend/agent/src/ai/provider_failure.go` (open decision 2 fixes the Go shape).
4. **AG-04.4** mirrors the witness-table exhaustiveness guard proven in this repo — `event_registry_test.go:56-217`, two legs per kind (constructor and payload accessor) cross-checked bidirectionally — over AG-04's own kind set only, while staying structurally extensible by AG-05 and AG-06.

All four leaves are behavior, so **all four are RED-first**. `sdd-design` names the mechanisms; this proposal names their obligations and their sources, never their Go spelling — doc 0003's authoring constraint forbids naming Layer 2 types from a planning artifact.

---

## Affected areas

| Area | Impact | Path | Change |
| --- | --- | --- | --- |
| Layer 2 | **New** | `backend/agent/src/agent/*.go` | First production code: envelope, kind registry, ordering, run/turn payloads, typed-failure surface, validator |
| Layer 2 | New | `backend/agent/src/agent/*_test.go` | 11 scenarios + the witness-table guard, external `package agent_test` |
| Layer 2 | Conditional | `backend/agent/src/agent/doc.go`, `doc_contract_guard_test.go` | Invariants + membership criterion in the docs; a guarded row only per open decision 3 |
| Specs | New | `openspec/specs/agent-event-envelope/spec.md` | `R-AEV-0NN` / `S-AEV-0NN` |
| Specs | Conditional | `openspec/specs/agent-package-scaffold/spec.md` | Delta only if a `doc.go` row is added |
| Layer 1 | **Unchanged** | `backend/agent/src/ai/**` | Read and cited as precedent; frozen surface (AG-12 guards it later) |
| — | **Unchanged** | `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` | No dependency, no tooling change |
| — | **Unchanged** | `backend/agent/src/agenttest/**`, `src/handoff/**`, other modules, `docs/**` | Cited, never edited |

---

## Risks

| # | Risk | Likelihood | Mitigation |
| --- | --- | --- | --- |
| 1 | **Review budget.** Forecast 1400–2200 changed lines (~500–750 production + ~900–1500 test) against a 1000-line budget — 1.4–2.2x AG-03's already-over-ceiling 1005 lines (`.../agent-package-scaffold/archive-report.md:87-89`) | **High** | `size:exception` pre-authorized for a single PR; **splitting is not proposed**. Mitigated by four commits, one per node, each independently reviewable, and by a validator whose surface is small even where its rule table is not. `sdd-tasks` restates the forecast with the mandated guard lines |
| 2 | **Scope creep into AG-05/AG-06 families** — the guard registering a placeholder kind, or a payload shape, for a family AG-04 does not own | High | The guard iterates exactly AG-04's registered kinds. Spec states the two owned families by register row (`VL2-EVT-02`, `VL2-EVT-03`) and names the six it must not touch. Reviewer checklist item |
| 3 | **Validator built non-reusably** (inline in a test file, or unexported) — rework at AG-05, AG-06 and AG-23 | Medium | Settled input 4 makes production export a requirement, not a preference; open decision 1 forces design to argue the internal shape with evidence |
| 4 | **Invariant overreach** — a test or guard written as if AG-04 closes invariant 3, or the full typed-error path | Medium | The partial-closure map (`0003:2203`) is quoted in Out of scope and must be restated in the spec's acceptance criteria |
| 5 | **The ordering mechanism inherits Layer 1's unsynchronized-counter assumption without Layer 1's single-producer guarantee.** `ai.Stamper` (`sequence.go:50-60`) is safe only because AI-02 guarantees one producer per stream; Layer 2's lanes have a different topology | Medium | AG-04.1 scenario 2 mandates the `-race` detector over two independent streams. Design must state which actor stamps a lane and why that is single-producer, citing AG-01 § 5 |
| 6 | **Kind vocabulary painted into a corner** — a closed set AG-05/AG-06 cannot extend without touching the checker | Medium | Layer 1's registry has been extended by AI-15…AI-19 with zero checker changes, driven by the documented six-step procedure (`event_descriptor.go:12-32`). Layer 2 documents its own equivalent procedure at AG-04, not at AG-05 |
| 7 | **Doc-guard churn** — adding a `doc.go` row late breaks `S-AGP-012..015` inside an already-large PR | Low | Open decision 3 is resolved at design, before apply. The mechanism for adding a row is already shipped and understood |

---

## Rollback plan

Required by `openspec/config.yaml`. Three levels, all cheap: nothing consumes Layer 2 yet, and this change adds no dependency, no schema, no migration and no running process.

1. **Revert the `doc.go` row and its table entry alone** (if open decision 3 added one). Two files, both already guarded; `S-AGP-012..015` returns to its merged state. Only meaningful together with level 2.
2. **Delete AG-04's production and test files from `backend/agent/src/agent/`.** The package returns to its post-AG-03 state: `doc.go` plus three guards. Nothing imports Layer 2 — Layer 1's own forbidden-prefix row proves Layer 1 never will (`backend/agent/src/ai/import_boundary_test.go`), and `src/coding/` and `src/cmd/` do not exist. `make test` in all three modules returns to the recorded pre-change baseline.
3. **Revert the merge commit.** Levels 1 and 2 together. No persisted state, no container, no published artifact, no dependency to unwind.

**Forward-fix preference.** If the envelope proves wrong at AG-05 — a family that will not fit the kind registry, a validator rule too strict for a real sequence — the correct move is to amend the mechanism in *that* milestone's change with its own justification, never to delete a rule or an invariant to unblock a dependency. Deleting a guard to pass is the exact failure ADR 0005 § Guard A was written against, and AG-03's rollback plan records the same preference.

---

## Dependencies

- **AG-01** — `agent-event-delivery`, archived 2026-08-11. Supplies the carrier (§ 3), the consumer lanes (§ 5) and the three ownership scopes (§ 6). Its § 10 states per-milestone exactly what AG-04 inherits (`decision.md:550-555`).
- **AG-03** — `agent-package-scaffold`, PR #161, merge `8f2ce2a6`, archived 2026-08-12. Supplies the package and both boundary guards, which AG-04 must keep green without editing their logic.
- **AG-00** — `agent-contract-vocabulary`, live at `openspec/specs/agent-contract-vocabulary/spec.md`. Supplies every term and register row AG-04 implements.
- **Layer 1** — `backend/agent/src/ai`, closed 42/42 at AI-40. Frozen surface; AG-04 imports it and edits nothing in it.
- **Toolchain**: Go 1.26.3, `golangci-lint v2.9.0`. **No new dependency in any module.**

---

## Success criteria

- [ ] `cd backend/agent && make test` green and `make lint` clean, recorded pre- and post-change.
- [ ] `backend/agent/go.mod` and `go.sum` byte-unchanged (recorded diff).
- [ ] All 11 charter scenarios (`0003:438-514`) exist as independently verifiable spec scenarios and as passing tests, each with a recorded RED.
- [ ] An **external-package** test constructs, validates and inspects every event kind this milestone registers (the charter's own acceptance criterion, `0003:418`).
- [ ] The stream-contract validator is exported from the production package and callable from another package without a test-only build tag.
- [ ] Two independently stamped hand-built streams are checked under `-race` and each is contiguous and 1-based.
- [ ] Every violating lifecycle permutation named by AG-04.2 is rejected: two run-starts, no run-end, overlapping turns, a turn outside the run bracket, any event after run-end.
- [ ] No construction route exists from a delta kind to an accumulated-message payload.
- [ ] A failure payload's category and cause are reachable as values; pre-stream and mid-stream failures are distinguishable; no test asserts on a message string's content as meaning.
- [ ] The every-kind-constructible guard bites: a scratch kind registered without a constructible payload fails the guard **by name**, red recorded, scratch file absent from the merged diff.
- [ ] The guard iterates exactly AG-04's own kinds; no message, tool, permission, cost, delegation or compaction kind is registered.
- [ ] The ordering invariants and the membership criterion are stated in the package documentation and pinned by test.
- [ ] AG-03's `import_boundary_test.go` and `ambient_authority_test.go` pass with **zero changes to their own logic**.
- [ ] No spec, test or acceptance line claims AG-04 closes envelope invariant 3, or invariant 1, 2 or 4 on its own (`0003:2203`).
- [ ] `git status` clean; PR labelled `size:exception` with the forecast acknowledged.

---

## Open decisions for design

Carried forward from `explore.md` § 8, **unresolved by design**. `sdd-design` MUST address each with evidence and a stated rationale; it may not inherit an answer from this proposal, because this proposal does not give one.

| # | Decision | Why it is genuinely open |
| --- | --- | --- |
| 1 | **The stream-contract validator's Go design** — mirror `ai.CheckStream`'s descriptor-driven engine (kind → descriptor → generic rule engine, `stream_check.go:64-118`), or a materially new design because Layer 2 adds a turn-inside-run nesting dimension Layer 1's single-level block model never had | A real design fork, not a naming question. Load-bearing for AG-05, AG-06 and AG-23's kit; a wrong early choice is rework at AG-23 |
| 2 | **The typed-failure payload's Go shape** — a thin wrap/embed of `*ai.Failure`, or an independent category/cause-compatible value type | "Wrap, not reuse-as-is" is **already settled** by `S-AGV-020` / `VL2-EVT-15`. Only the Go shape is open, and doc 0003's authoring constraint deliberately leaves it to design/spec |
| 3 | **Whether AG-04 adds a fourth guarded row to `doc.go`'s machine-checked layer-contract table.** The three existing rows (L2C-01/02/03, `doc.go:16-18`) cover imports, no-I/O and "the event stream is the only upward contract" — none is envelope-specific | *How* to add a row is already resolved by the shipped guard. *Whether* AG-04 needs one is undecided, and it changes whether `agent-package-scaffold` takes a delta |
| 4 | **Whether AG-04's tests use `backend/agent/src/agenttest` at all** | The charter's dependency edges (AG-01 and AG-03 only, `0003:419`) and its "hand-built sequences" wording (`0003:417`) both suggest **no** — but that is inferred from an absence, never stated anywhere. Confirm rather than assume |

---

## Contradictions and divergences recorded

Recorded, not silently resolved.

1. **The invariant-closure map is sharper than the summary usually given.** `0003:2203` shows AG-04 co-closes invariant **1** with AG-05.1 as well as co-closing 2 and 4 — so AG-04 closes *no* envelope invariant on its own, and is absent from invariant 3 entirely. Any statement that AG-04 "fully closes invariant 1" contradicts the spine. The spine is authoritative here.
2. **The two exploration copies diverge in one sentence.** `explore.md` § 7 (OpenSpec) ends the sizing forecast at the risk flag; the Engram copy (`#2920`) additionally names chaining into two PRs as a live option. The session's cached delivery strategy (`exception-ok`, single PR) settles it in favour of the OpenSpec wording. Flagged so a later reader does not treat the Engram sentence as an open question.
3. **The Engram copy's header note is stale.** It says `explore.md` could not be written to disk; the file exists at `openspec/changes/cachicamas-agent-event-envelope/explore.md` and was read for this proposal.

---

## Notes for the following phases

- **`sdd-spec`**: one new spec `specs/agent-event-envelope/spec.md` (`R-AEV-0NN` / `S-AEV-0NN`), plus a conditional `agent-package-scaffold` delta if open decision 3 says yes. The 11 charter scenarios are the normative starting set — restate them as independently verifiable Given/When/Then, do not reduce them. The guard requirement closes on a recorded bite, never on green. State the partial-closure map as an explicit non-claim.
- **`sdd-design`**: must resolve all four open decisions above with evidence; must state which actor stamps a lane and why that is single-producer (risk 5); must document Layer 2's own "adding a kind" procedure so AG-05 and AG-06 extend the registry without touching the checker; must not name a Go type this proposal has not authorized it to invent — it authorizes design to choose them, which is different.
- **`sdd-tasks`**: four phases matching the node graph — AG-04.1, then AG-04.2 and AG-04.3 in parallel, then AG-04.4 (`0003:422-432`). Evidence gate is `make test` in `backend/agent/` for all four. Single PR; `size:exception` pre-authorized; the tasks file must carry the guard lines and restate why chaining was not chosen.
