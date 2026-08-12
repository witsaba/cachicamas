```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:e94bc9e8a586db68f749058a596920c0cb10d4b027b56425b10c4cd8f7346d16
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 19/19
scenarios: 27/27
test_command: scratchpad/inspect.sh
test_exit_code: 0
test_output_hash: sha256:b59886c179afaf0919929ff25edf36ce4fef45a65f5c3bafc43477f25031ebd2
build_command: scratchpad/structure.sh
build_exit_code: 0
build_output_hash: sha256:4696eded5de65616ab403ffd94479a2a9c79641bf848280740617a2589525714
```

# Verify report — Layer 2 agent event delivery and the observer model

> **Change**: `cachicamas-agent-event-delivery`
> **Milestone**: AG-01 of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md)
> **Node**: AG-01.1 — The delivery decision `[decision]`
> **Phase**: verify
> **Status**: **PASS WITH WARNINGS** — 0 CRITICAL · 3 WARNING · 4 SUGGESTION
> **Date**: 2026-08-11
> **Worktree**: `cachicamas-worktrees/agent-layer2-wave0` · **Branch**: `feat/agent-layer2-wave0` · **Base**: `b9b4e662`
> **Mode**: `[decision]` leaf — no production code, no `make test` gate. Every check below is inspection, and every citation was re-resolved against its source rather than taken on the artifact's word.
> **Evidence revision**: `sha256:e94bc9e8…7346d16` over the six artifacts of this change, in the order listed in § 2.

---

## 0. What this verification did not take on trust

The apply phase reported eight claims. Each was treated as a hypothesis and re-derived by command or by opening the cited source:

| Apply claim | Re-derived | Result |
| --- | --- | --- |
| 32/32 tasks complete | `grep -c '^- \[x\]'` / `'^- \[ \]'` | **Confirmed** — 32 / 0 |
| 29 distinct `VL2-*` citations, all resolving in AG-00's register | set difference of cited vs. defined | **Confirmed** — 29 cited, 83 rows, empty difference |
| Zero Go identifiers | three-pattern scan, positive control first | **Confirmed** — control fires, artifacts clean |
| AG-23.2 and AG-21.2 exist and bear their claims | doc 0003 lines 2130–2146, 2011–2027, 1962–1972 | **Confirmed for AG-23.2; qualified for AG-21.2** (WARNING 3) |
| Five closing-checklist items with real argumentative content | doc 0003 lines 236–241 walked against `decision.md` § 12 | **Confirmed** |
| Both source forks recorded with losing readings | doc 0001 § 2.2 mermaid opened; § 2.2/§ 2.3 quotes matched | **Confirmed** |
| 622 lines reflects "cross-reference rather than repeat", not incompleteness | requirement-by-requirement walk, § 4 | **Substantially confirmed, one exception** (WARNING 2) |
| Loss postures nameable distinctly | AG-00 register rows `VL2-EVT-18`/`-19` opened and compared to design § 4.2–§ 4.3 | **Confirmed** |

**On the line count.** 622 lines against a tasks forecast of ~1,600–2,200 is a 3× shortfall, and it was tested against the spec rather than against the estimate. All 19 requirements and all 27 scenarios are satisfied (§ 4, § 5). The brevity is real compression, not omission — but the stated mechanism is only partly the true one. Two places deliberately repeat rather than cross-reference: the rendezvous objection (§ 4 and § 5, both carrying the verbatim Layer 1 quote and the receive-step answer — WARNING 2) and the inheritance table (§ 7 and § 10, with an explicit non-divergence claim that was checked row by row and holds, § 6). The compression comes mostly from tighter prose per argument.

---

## 1. Evidence commands and their output

Two commands were executed. Neither is `make test`: this change adds no Go file, no build target, and no test target, so the node's evidence gate is inspection by construction (`design.md` § 13, `tasks.md` § "Node type"). Both scripts are reproduced in § 9.

### 1.1 Authoring-constraint and citation-resolution gate

```
$ /…/scratchpad/inspect.sh   →  exit 0
== 1. Go-identifier scan (positive control first) ==
positive-control matches: 2
OK   cachicamas-agent-event-delivery/decision.md -> no Go identifier
OK   cachicamas-agent-event-delivery/design.md -> no Go identifier
OK   cachicamas-agent-event-delivery/tasks.md -> no Go identifier
OK   cachicamas-agent-event-delivery/proposal.md -> no Go identifier
OK   cachicamas-agent-event-delivery/explore.md -> no Go identifier
OK   cachicamas-agent-event-delivery/specs/agent-event-delivery/spec.md -> no Go identifier

== 2. VL2-* citation resolution against AG-00 register ==
distinct cited:       29 | register rows:       83
OK   every cited term resolves to a register row

== 3. Task completion ==
complete=32 incomplete=0

== 4. No numeric Layer 2 capacity ==
OK   no numeric capacity attributed to a Layer 2 boundary

== 5. Diff scope ==
OK   no backend, build or dependency file touched

exit=0
```

`test_output_hash: sha256:b59886c179afaf0919929ff25edf36ce4fef45a65f5c3bafc43477f25031ebd2`

**The positive control is load-bearing.** Before the artifacts were scanned, the same three patterns were run against a synthetic file containing `func (h *Harness) Run(ctx context.Context) (<-chan StreamEvent, error)` and `type EventEnvelope struct{ RunID string }`. It matched. A scan that reports "clean" without a firing control reports only that the regex is broken. Three tokens are whitelisted and were inspected individually: `PascalCase` and `camelCase` in `tasks.md` § "Review focus" (they name the *prohibition*), and `OpenTelemetry` in `proposal.md` (a product name in observability prose). None is a Go type, field, method or package identifier.

### 1.2 Structural gate

```
$ /…/scratchpad/structure.sh   →  exit 0
== A. Outbound markdown links resolve ==
OK   ../../../docs/architecture/0001-cachicamas-agent-stack-v2.md
OK   ../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md
OK   ../../specs/ai-stream-lifecycle/spec.md
OK   ../archive/2026-07-31-cachicamas-ai-stream-lifecycle/decision.md
OK   ../archive/2026-08-07-cachicamas-ai-backpressure/decision.md
OK   ../cachicamas-agent-contract-vocabulary/specs/agent-contract-vocabulary/spec.md

== B. Twelve-section skeleton (design.md §8) ==
top-level numbered sections: 12

== C. Five-part shape in each decision section §3..§7 ==
§3: Decision>Why>What this excludes>Consequences>Who inherits it>
§4: Decision>Why>What this excludes>Consequences>Who inherits it>
§5: Decision>Why>What this excludes>Consequences>Who inherits it>
§6: Decision>Why>What this excludes>Consequences>Who inherits it>
§7: Decision>Why>What this excludes>Consequences>Who inherits it>

== D. Internal subsection cross-references resolve inside decision.md ==
DANGLING line 29 -> § 4.2 § 5.3
DANGLING line 188 -> § 5.4
DANGLING line 607 -> § 5.4
dangling internal subsection refs: 3
WARN (non-blocking): decision.md declares no numbered subsections, so these references resolve to nothing

exit=0
```

`build_output_hash: sha256:4696eded5de65616ab403ffd94479a2a9c79641bf848280740617a2589525714`

Check D is reported as `WARN` and does not set the exit code, because a dangling label is not a spec failure and encoding it as one would overstate its severity (WARNING 1 in § 7).

---

## 2. Deliverable inventory

| Path | Present | Lines | Note |
| --- | :---: | ---: | --- |
| `decision.md` | ✅ | 622 | twelve sections, §§ 3–7 in AG-01.1's closing-checklist order |
| `specs/agent-event-delivery/spec.md` | ✅ | 281 | `R-AGE-001` … `R-AGE-019`, `S-AGE-001` … `S-AGE-027` |
| `design.md` | ✅ | 369 | fifteen sections; § 8 is the skeleton `decision.md` instantiates |
| `tasks.md` | ✅ | 166 | 32 tasks, all `[x]` |
| `proposal.md` | ✅ | 242 | |
| `explore.md` | ✅ | 169 | |
| anything under `backend/`, `go.work`, `go.mod`, `go.sum`, `Makefile` | ✅ absent | — | `git status --porcelain` over those paths is empty |

`git status --porcelain` shows exactly three untracked directories, all under `openspec/changes/`: this change, AG-00 (`cachicamas-agent-contract-vocabulary`) and AG-02 (`cachicamas-agent-v1-scope`). This change added six files, all inside its own directory. It edited no existing file — including AG-00's register, which task 8.1 explicitly delegates to AG-00's own apply.

---

## 3. Requirement-by-requirement verdict

| Requirement | Verdict | Deciding evidence |
| --- | :---: | --- |
| `R-AGE-001` carrier decided and argued at Layer 2 | **PASS** | `decision.md` § 3: four Layer 1 grounds reproduced; Ground 1 cites `AG-10.3`, Ground 3 cites `AG-04.2`, both re-opened in doc 0003 (lines 1080–1094, 459–480) and both bear the claims; iterator case stated at full strength before rebuttal (lines 65–74); push-callback and send-capable-handle both in the exclusion table; abandonment conceded at line 94 |
| `R-AGE-002` liveness rule adopted unweakened | **PASS** | `decision.md` lines 100–105, mapped 1:1 against Layer 1 spec § 5 (lines 239–254). Four clauses, same force, no third ending anywhere |
| `R-AGE-003` iterator-view ergonomics outside the runtime package | **PASS** | `decision.md` line 109; owner `AG-23.2` confirmed as doc 0003's test-kit-and-examples node (lines 2130–2146), and AG-01's own charter note (line 232) already routes iterator-view ergonomics to the test kit |
| `R-AGE-004` lossless and ordered | **PASS** | `decision.md` line 141; full sweep of all 64 loss/drop/discard mentions found no third sanctioned circumstance (§ 5, S-AGE-005) |
| `R-AGE-005` loop-internal boundary inherits unchanged | **PASS** | `decision.md` lines 143, 173, 179; hazard recorded at 173 with AG-00's register named as mitigation at 216; `VL2-EVT-18` row exists at register line 412 |
| `R-AGE-006` harness-facing boundary never drops committed history | **PASS** | `decision.md` lines 145, 155–169, 180; measured capacity `0` matches Layer 1 spec lines 67 and 391; zero-capacity objection answered at 188 |
| `R-AGE-007` droppable set stated | **PASS** | `decision.md` lines 190–194: droppable set positive at 192, why-not-fully-lossless at 194, cancelled-mid-tool-result trace worked at 184 |
| `R-AGE-008` stalled observer structurally unable to stall | **PASS** | `decision.md` lines 230, 252, 256–267; four-row impossibility table at 273–278 |
| `R-AGE-009` more than one attached consumer, none privileged | **PASS** | `decision.md` lines 282–284; the losing reading is a real reading of doc 0001 § 2.2's mermaid, verified (§ 6.2) |
| `R-AGE-010` three scopes, one closer each | **PASS** | `decision.md` lines 320–356, 362–366 |
| `R-AGE-011` exactly-one-terminal at agent level | **PASS** | `decision.md` lines 352–358; the "no bare close at run scope" claim is anchored on `VL2-EVT-19`, which is `R-AGE-006` |
| `R-AGE-012` post-terminal consumer assumption | **PASS** | `decision.md` line 358, differentiated by outcome, citing § 4's narrowing |
| `R-AGE-013` one inbound surface, three payload kinds | **PASS** | `decision.md` lines 397, 426; both doc 0001 quotes verified verbatim (doc 0001 lines 274, 313) |
| `R-AGE-014` lookup exists, harness-owned, shape open | **PASS** | `decision.md` line 397; deferral to `AG-10`/`AG-13` at 451; no sentence constrains shape |
| `R-AGE-015` typed rejection at two granularities, two carve-outs | **PASS** | `decision.md` lines 397, 428; `AG-14.1`'s idempotence scenario and `AG-13.3`'s pause node both confirmed in doc 0003 (1407–1410, 1357–1367) |
| `R-AGE-016` upward path recurses under delegation | **PASS** | `decision.md` line 443 |
| `R-AGE-017` inheritance table for the six blocked milestones | **PASS** | `decision.md` lines 467–476 (§ 7) and 550–585 (§ 10); no-invented-channel rule at 476, restated at 596 |
| `R-AGE-018` deferred numbers name owner and closing evidence | **PASS** (with WARNING 3) | `decision.md` lines 196–200; no numeric Layer 2 capacity anywhere (command-verified); `AG-21.2` named; closing evidence and inherited tie-break stated |
| `R-AGE-019` acceptance criterion and authoring constraints | **PASS** (merge-order half unverifiable pre-merge) | `decision.md` § 12; command-verified authoring constraint; all five doc 0003 checklist items walked |

---

## 4. Scenario-by-scenario verdict

Several scenarios are written to be failed by a plausible artifact. Those are marked **[adversarial]** and were applied as written.

| Scenario | Verdict | Deciding evidence |
| --- | :---: | --- |
| `S-AGE-001` **[adversarial]** each ground cites a Layer 2 source | **PASS** | Ground 1 header names `AG-10.3` and quotes its scenario; Ground 3 header names `AG-04.2` and quotes its bracket discipline. Both node charters opened in doc 0003 and both bear the claim. Grounds 2 and 4 also argue from Layer 2 facts (the loop's own scheduling/permission work; Layer 2 delivery being single-use and side-effecting) rather than from "Layer 1 decided the same" — the disqualifying pattern is absent from all four |
| `S-AGE-002` alternatives priced | **PASS** (weak on one half) | push-callback cost stated directly (line 120: "relocates Ground 2's hidden buffer to every observer, uncounted"); send-capable-handle cost stated inversely (line 119) — see SUGGESTION 1; abandonment residual at 94 and § 9 item 4 |
| `S-AGE-003` four clauses present, unweakened | **PASS** | Table at 100–105 against Layer 1 § 5. No unconditional send, no advisory cancellation, no third ending admitted anywhere in the artifact |
| `S-AGE-004` view placement stated with its reason | **PASS** | Line 109 places it in a test-support sibling, "never inside the package Layer 2 itself ships" (the production/test closure split), carries "never a second contract", and defers the existence question to `AG-23.2` by name — see SUGGESTION 4 |
| `S-AGE-005` lossless with named exceptions only | **PASS** | Full sweep of every loss-word occurrence in `decision.md`. Exactly two sanctioned circumstances (`VL2-EVT-18`, `VL2-EVT-19`). Line 276's drop-on-overflow is a *rejected* mechanism, explicitly labelled "a second, unsanctioned loss circumstance"; line 208 forbids a third by name |
| `S-AGE-006` boundary named, hazard recorded | **PASS** | Two boundaries stated separately at 143/145 and tabulated separately at 179/180; hazard at 173; mitigation named as AG-00's register at 216 |
| `S-AGE-007` **[adversarial]** committed side effect cannot be silently absent | **PASS** | Rule at 145 and 180 requires delivery before or with run-end; the worked trace at 155–169 is the exact run the scenario describes and reaches the opposite of the permitted outcome. Nothing in the artifact permits dropping such an event on saturation |
| `S-AGE-008` narrowing rests on the measured capacity | **PASS** | The standing figure cited is `0` (line 151), matching Layer 1 spec lines 67 and 391. Both mentions of 64 are marked historical (151, 210). The "consumer pauses by design" objection is answered by the receive-step / after-hand-off distinction at 188 and 290 |
| `S-AGE-009` **[adversarial]** both sides of the line stated | **PASS** | Line 192 names three classes of still-droppable event; line 194 explains why full losslessness is unavailable at any price. The artifact does not state only what may not be dropped |
| `S-AGE-010` **[adversarial]** stalled-observer trace has no path back | **PASS** | See § 6.1 below. The defence is a mechanism, not an obligation: the offer into lane B does not wait *because* trilemma property (c) was deliberately bent (line 252), and the trace at 256–263 terminates at forwarding activity B. A convention-only defence would fail; none is offered |
| `S-AGE-011` rejected mechanisms judged by impossibility | **PASS** | All four rows of the table at 273–278 state an impossibility. Blocking synchronous multicast row reads "**Nothing.**"; drop-on-overflow row reads "**Rejected against `R-AGE-004`.**" |
| `S-AGE-012` fork decided with rebuttal on the page | **PASS** | Lines 282–284. The losing reading is a genuine reading of doc 0001 § 2.2 — verified against the mermaid itself (§ 6.2) — and two rejection grounds are given. A later reader can tell a decision from an oversight without opening `explore.md` |
| `S-AGE-013` every scope has exactly one closer | **PASS** | Lines 322–356; the turn/run split is argued from `VL2-COR-02`/`VL2-COR-17` loop statelessness at 354 and 364 |
| `S-AGE-014` delegation does not break the ownership rule | **PASS** | Line 356: child's run-end emitted on the child's own stream before the parent emits subagent-ended; neither closes the other's. Matches `AG-19.2`'s scenario verbatim in force (doc 0003 lines 1836–1839) |
| `S-AGE-015` cancelled run still ends with a terminal | **PASS** | Turn-end on every exit path with the aborted outcome (352); run-end with the interrupted outcome (354); nothing after the terminal (354); the no-bare-close claim anchored on `VL2-EVT-19` (358) |
| `S-AGE-016` **[adversarial]** assumption stated per outcome | **PASS** | Line 358 differentiates completed from interrupted/failed and cites § 4's narrowing as what makes the prefix trustworthy. No single undifferentiated claim appears. Interrupted and failed share one statement, but both are named explicitly in it |
| `S-AGE-017` **[adversarial]** two source levels reconciled in writing | **PASS** | Line 426 carries both doc 0001 quotes — verified verbatim at doc 0001 lines 274 and 313 — and states the reason (the harness is the stable addressable thing across a run; the loop is re-instantiated per turn). The mismatch is not left implicit |
| `S-AGE-018` existence decided, contents left open | **PASS** | Line 397 states existence, ownership and lifetime and explicitly declines shape, structure and storage; line 451 names `AG-10` and `AG-13` as the owners |
| `S-AGE-019` message to an ended run rejected typed | **PASS** | Line 397 and the § 7 diagram's run-granularity branch; "never a silent drop" stated; line 450 forbids the silent drop by name |
| `S-AGE-020` redundant interrupt during wind-down is not a rejection | **PASS** | Line 428 requires silent idempotence during wind-down and states that the rejection machinery begins only once run-end has been emitted. `AG-14.1`'s "nothing changes and nothing panics" verified in doc 0003 |
| `S-AGE-021` pause-resumption carved out | **PASS** | Line 428: model-initiated, harness-internal, "not an instance of the upward path at all", never routed through identity or rejection machinery; `AG-13.3` named and confirmed as doc 0003's pause-resumption node |
| `S-AGE-022` decision for a nested call reaches the child | **PASS** | Line 443: sent to the parent's surface, arrives at the child's own suspension lookup; the routing obligation is stated here and its mechanism assigned to `AG-19` |
| `S-AGE-023` every blocked milestone has a row | **PASS** | § 7's table at 469–474 has a row for each of AG-04, AG-10, AG-13, AG-14, AG-19, AG-20; the no-invented-channel rule is stated once at 476. The criterion is checkable from that one table |
| `S-AGE-024` **[adversarial]** capacity deferral is a decision, not an omission | **PASS** | Command-verified: no numeric capacity is attributed to any Layer 2 boundary; the only figures present (`0`, `64`) are Layer 1's, one measured and one marked historical. The deferral is recorded at 196–200 with `AG-21.2` named and its closing evidence described. This is not silent omission — see WARNING 3 on the owner's charter |
| `S-AGE-025` checklist walked and closed | **PARTIAL** | § 12's table answers all five items with an evidence location and a *what this excludes* pair, and each item was re-walked against doc 0003 lines 236–241. The second half — "closed before AG-04 starts" — is verifiable only in part: no AG-04 change exists in this worktree at verification time, but merge ordering cannot be observed pre-merge |
| `S-AGE-026` authoring constraint holds across the change | **PASS** | Command-verified with a firing positive control across all six files; all six added files are inside this change's directory; `git status --porcelain` over `backend`, `go.work`, `go.mod`, `go.sum` and `Makefile` is empty |
| `S-AGE-027` vocabulary cited, never minted | **PASS** | 29 distinct `VL2-*` citations, empty set difference against the register's 83 rows. `VL2-EVT-18`/`-19` were appended by AG-00 in the same PR under a dated category note; the register's per-category counts (23·19·9·9·14·9) sum to the stated 83, so the count update task 8.1 requires was performed |

**Totals**: 26 PASS, 1 PARTIAL, 0 FAIL.

---

## 5. Task completion

32 declared, 32 checked, 0 unchecked. Spot-checked against the artifact rather than accepted:

| Task | Claim | Checked against |
| --- | --- | --- |
| 4.2 (load-bearing) | run-extent-bounded lane absorption written | `decision.md` 230, 252, 256–267, 584 — all four components present (§ 6.1) |
| 3.3 | measured capacity 0, saturation ordinary | `decision.md` 151 vs. Layer 1 spec 67, 391 |
| 3.5 | `AG-21.2` named with closing evidence | `decision.md` 196–200 vs. doc 0003 2011–2027 (WARNING 3) |
| 2.3 | `AG-23.2` named | `decision.md` 109 vs. doc 0003 2130–2146 and AG-01's own charter note |
| 8.1 | rows exist in AG-00's register, not appended here | register lines 391, 412, 413; `git status` shows no edit from this change |
| 7.3 | § 10 does not diverge from the per-decision inheritance | six milestones compared row by row; § 10 is a strict superset (§ 6.3) |
| 9.1 / 9.2 | design § 14's eight criteria and § 13's eight-item pass | seven of eight hold; criterion 5 does not (WARNING 2) |

Task 8.1's completion note is worded "**Verified complete by citation, not performed here**", which is the correct disposition — the rows belong to AG-00's file and this change's diff must not touch it. Both halves were confirmed independently.

---

## 6. The three deep checks the orchestrator singled out

### 6.1 The load-bearing addition — run-extent-bounded lane absorption

Present, and the argument terminates rather than asserting. The chain is:

1. `decision.md` 252 names the trilemma and states which property bends: **(c)**, the fixed buffer bound. "each lane absorbs the gap between the canonical producer's progress and its own consumer's progress, **without a fixed bound chosen in advance**."
2. Because the lane has no fixed bound, the distribution step's offer into lane B cannot block. That is the deriving step, not a restatement.
3. Because the offer does not block, the distribution step's receive from the canonical stream stays free (256–263).
4. Because that receive stays free, the canonical producer never waits on consumer B.

The absorption is bounded intrinsically by the run's extent — "a run is finite, a lane drains as its consumer catches up, and everything a lane holds is freed at run end (`R-02`)" — and `R-02` was confirmed as doc 0003's no-I/O requirement (doc 0003 line 56). The bound is restated in § 10's AG-20 row (584) so `AG-20.2` inherits it.

Had the artifact asserted only "the forwarding activity must not block", `S-AGE-010`'s second clause would have failed it. It does not: the non-blocking property is derived from a deliberately bent invariant, and the cost of bending it is named and assigned. **PASS.**

The residual is real and is recorded as SUGGESTION 2 rather than as a defect: a permanently stalled consumer grows its lane for the run's whole duration. The artifact assigns whether a tighter bound is wanted to `AG-21.2` (252) but never prices the memory residual the way § 3 prices the abandonment residual.

### 6.2 The backpressure narrowing is not vacuous

Both halves are present and the worst case is worked, not named.

- **What may never be lost** — 145, 180: any event describing a fact already committed to `VL2-HAR-01` history, and run-end itself.
- **What may still be lost** — 192, stated positively: "in-flight deltas of a turn cut short, progress of tools that never completed, and everything of turns that never began", with the observation that the set is non-empty at every interruption.
- **Why full losslessness is unavailable** — 194: a consumer that stops reading and never cancels would convert `VL2-HAR-05` bounded wind-down into an unbounded one.
- **The cancelled-mid-tool-result trace is worked** — 184, not merely named. The event drops at the loop-internal boundary under `VL2-EVT-18`; history never saw it; `VL2-HAR-02` orphan synthesis then fires; **the synthesised result is itself a committed history fact**, so `VL2-EVT-19` requires *its* delivery before run-end. The argument's conclusion — "the guarantee is anchored on history, not on any one event instance" — is what closes the gap between the two postures, and 182's watershed argument supplies the reason no committed fact can leak out of both rules.

`VL2-HAR-02`'s register row and `AG-14.1`'s charter were both opened: synthesis runs during interrupt wind-down, before run-end, so the argument's premise holds (SUGGESTION 3 notes a wording difference between the two sources). **PASS.**

### 6.3 The three challengeable choices

| Choice | Owner's charter opened | Can it bear the claim? |
| --- | --- | --- |
| Carrier-view convenience → `AG-23.2` | doc 0003 2130–2146 | **Yes, strongly.** The node ships "a scripted-harness kit… importable… sibling to the Layer 1 kit conventions" and "runnable package examples covering… consuming events" — both quoted accurately in `decision.md` 109. AG-01's own charter note (doc 0003 line 232) independently states "The iterator-view ergonomics live in the test kit", so the assignment matches the milestone document's own routing rather than inventing one |
| Capacity measurement → `AG-21.2` | doc 0003 1962–1972, 2011–2027 | **Partly.** The `AI-33/AI-34 of this layer` quote is verbatim (doc 0003 line 1964). AG-21.2's two scenarios are quoted accurately. But AG-21's charter says **"Out of scope: Performance targets — correctness under pressure only"**, its deliverable is a hardening suite and leak checks, and neither it nor AG-21.2 commits to producing a "lane-absorption high-water mark". `R-AGE-018` is satisfied on its face; the owner's charter does not yet carry the deliverable — WARNING 3 |
| Two loss-posture rows → AG-00's register | register lines 391, 412, 413 | **Yes.** `VL2-EVT-18` reproduces design § 4.3's loop-internal row (rule, what may be lost, what more is promised) and `VL2-EVT-19` reproduces design § 4.2–§ 4.3's harness-facing row (the history guard, the wind-down obligation, both loss columns) with owner `AG-01` and provenance pointing back at this change's `design.md`. The category note at line 391 records that the rows were authored on AG-01's behalf at AG-01's request. Category counts and the stated sum of 83 are internally consistent |

### 6.4 The inheritance table

Both § 7 (467–476) and § 10 (550–585) carry all six milestones: AG-04, AG-10, AG-13, AG-14, AG-19, AG-20. The artifact claims at 548 that "this table does not diverge from any per-decision 'Who inherits it' subsection above". Compared row by row: **§ 10 is a strict superset of § 7 in every case** (it adds the liveness rule and the per-outcome assumption for AG-04, the run-scope closer for AG-13, the loop-internal loss posture for AG-14, the re-emission point for AG-19, and a "Not inherited" line for AG-04 and AG-10). No contradiction. The claim holds.

### 6.5 The Layer 1 citations

This change's exploration previously carried a false claim about doc 0003 that had to be corrected mid-flight, so every Layer 1 claim was re-derived from `openspec/specs/ai-stream-lifecycle/spec.md`:

| Claim in `decision.md` | Layer 1 source | Verdict |
| --- | --- | --- |
| capacity is `0`, measured and fixed by AI-34.1 | lines 28, 67, 391 | **Accurate** |
| the 64 hypothesis is struck through in the live contract | line 67: `~~starting capacity 64~~` | **Accurate** |
| the "why 64" rationale is annotated historical | line 413: `> **Historical (annotated 2026-08-10).**` | **Accurate** |
| exactly one sanctioned loss path at Layer 1 | lines 67, 708, 764 | **Accurate** |
| "a missing terminal after your own cancellation makes you the party in error" | lines 401, 708 | **Accurate in force** (paraphrase, not quoted) |
| the four carrier grounds | lines 100–133 | **Accurate** — all four reproduced, in order |
| the four liveness clauses | lines 239–254 | **Accurate** — one-for-one, none weakened |
| the zero-capacity objection, quoted verbatim | line 421 | **Verbatim** — character-for-character |
| tie-break "prefer the smaller" | lines 391, 438 | **Accurate** |
| AI-40.3 restates the abandonment clause at the freeze | line 264 | **Accurate** |

No Layer 1 claim was found to be false or overstated.

---

## 7. Issues

### CRITICAL

None.

### WARNING

**W1 — three lines carry four internal subsection references that resolve to nothing.**
`decision.md` uses only unnumbered `###` headings inside its twelve sections, so it declares no `§ N.M` label anywhere. Line 29 directs a reviewer to "§ 4.2's worked case and § 5.3's terminating trace… the two places a reader should look first for a hole in the argument"; lines 188 and 607 point at "§ 5.4". Those numbers are `design.md`'s, not `decision.md`'s. The content is present — the worked case is at 155–169, the trace at 256–263, the rendezvous proof at 286–290 — but the artifact's own reviewer entry point sends a reader to labels that do not exist. Command evidence in § 1.2 check D.
*Not CRITICAL*: no requirement or scenario turns on it, and § 12's evidence column locates the same material correctly.

**W2 — the rendezvous objection is stated twice, not once-and-cross-referenced.**
`design.md` § 14 criterion 5 requires it "answered… in decision § 4 or § 5, **once**, and cross-referenced rather than repeated"; `tasks.md` 4.5 makes the same commitment and 9.1 records it verified. In fact the verbatim Layer 1 quote *and* the receive-step/after-hand-off answer both appear in full at 188 (§ 4) and again at 288–290 (§ 5). The duplication is deliberate and argued in-text ("this section states the conclusion it depends on, once, so a reader of the backpressure rule alone is not left with an unanswered objection on the page"), and no spec requirement is violated — but the acceptance criterion as written is not met, and task 9.1's recorded verification of it is therefore inaccurate.

**W3 — `AG-21.2`'s charter does not yet carry the deliverable the capacity deferral assumes.**
`decision.md` 196–200 names `AG-21.2` as the capacity-measurement owner with closing evidence of "the lane-absorption high-water mark and producer-wait observations". Doc 0003's AG-21 charter states **"Out of scope: Performance targets — correctness under pressure only"**; its deliverable is "a hardening suite over combined scenarios… leak checks"; and AG-21.2's two scenarios assert *correctness* ("a stalled consumer loses nothing", "cancellation unblocks… within the documented bound"), not a measurement. `R-AGE-018` is satisfied — an owner is named and closing evidence is described — but the owner's charter would need amending, or the closing evidence re-derived from the scenarios it actually has, before the deferral can close. The decision partially acknowledges this at 200 ("`AG-21.2`'s charter demands correctness under pressure, not a number to test") without resolving it.

### SUGGESTION

**S1** — § 3's exclusion row for the send-capable handle (line 119) states the cost inversely, as the benefit of receive-only, where the push-callback row (120) states its cost directly. `S-AGE-002` passes either way; symmetry would make it unambiguous.

**S2** — the memory residual of bending trilemma property (c) — a permanently stalled consumer grows its lane for the run's whole duration — is assigned to `AG-21.2` (252) but never priced as a residual cost, unlike the abandonment residual which § 3 and § 9 item 4 price explicitly. § 3's own discipline applied to § 5 would name it.

**S3** — `decision.md` 184 says orphan synthesis fires "before history closes"; `VL2-HAR-02`'s register row says "before the next turn runs". `AG-14.1`'s charter resolves the ambiguity in the decision's favour (synthesis runs during interrupt wind-down), so the argument holds, but the two phrasings differ.

**S4** — `S-AGE-004` asks the placement reason to cite "the no-I/O rule or the production/test closure split". Line 109 *describes* the split rather than citing `R-02` by identifier, where § 5 cites `R-02` explicitly. The scenario passes; an identifier would make it uniform with the artifact's own citation discipline.

---

## 8. What could not be verified

| Item | Why |
| --- | --- |
| "closed before AG-04 starts" (`R-AGE-019`, `S-AGE-025`) | Merge ordering is not observable pre-merge. What *was* confirmed: no AG-04 change exists in this worktree, and `openspec/changes/` holds only AG-00, AG-01 and AG-02 |
| "merged in the same pull request as AG-00" (`tasks.md` acceptance 2) | No PR exists yet. What *was* confirmed: AG-00's register carries both rows this change depends on, and this change's diff does not touch that file |
| Downstream milestones honouring the no-invented-channel rule | Those milestones do not exist. The rule is stated; its enforcement is a future verification's |

---

## 9. Evidence scripts

Both scripts were run from `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/agent-layer2-wave0`, read-only.

`inspect.sh` — five checks: (1) Go-identifier scan across all six artifacts, preceded by a positive control that must fire; (2) set difference of `VL2-*` identifiers cited in `decision.md` against rows defined in AG-00's register; (3) `[x]` / `[ ]` task counts; (4) grep for a numeric capacity attributed to a Layer 2 boundary; (5) `git status --porcelain` over `backend`, `go.work`, `go.mod`, `go.sum`, `Makefile`. Exit 0.

`structure.sh` — four checks: (A) every outbound markdown link target exists on disk; (B) `decision.md` has exactly twelve top-level numbered sections, matching `design.md` § 8's skeleton; (C) each of §§ 3–7 carries the five-part shape in order; (D) every `§ N.M` reference is classified external (doc 0001) or dangling. Exit 0 — check D reports `WARN` without setting the exit code.

---

## 10. Verdict

**PASS WITH WARNINGS.**

All 19 requirements hold. 26 of 27 scenarios pass outright; `S-AGE-025` is PARTIAL for a reason no pre-merge verification can close. Zero CRITICAL findings, so nothing blocks archive. The three WARNINGs are corrections a reviewer should see before merge: W1 is a one-line fix (add subsection numbers, or repoint the three references), W2 is either a `design.md` criterion to relax or a duplication to collapse plus a task 9.1 note to correct, and W3 is the one that outlives this change — `AG-21.2` inherits a closing-evidence obligation its charter does not currently state, and that gap should be recorded where AG-21 will find it.

The apply phase's line-count deviation is not a completeness defect. The artifact is short because its arguments are tight, not because a requirement went unanswered.

**Next**: `sdd-archive`.
