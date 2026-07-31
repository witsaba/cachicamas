# Archive report — Layer 1 stream lifecycle, ownership and carrier

> **Change**: `cachicamas-ai-stream-lifecycle`
> **Milestone**: AI-02 of [doc 0002](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-02--decide-stream-lifecycle-ownership-and-the-carrier) — Decide stream lifecycle, ownership, and the carrier
> **Node**: AI-02.1 — Lifecycle, ownership and carrier `[decision]`
> **Phase**: archive
> **Status**: **ARCHIVED**
> **Date**: 2026-07-31
> **Pull request**: #95 (`feat/2026-07-31-cachicamas-ai-layer1-wave-0` → `main`)
> **Merge commit**: `a831c06` · **Base**: `origin/main` @ `b6c59e6`
> **Change commit on `main`**: `461cc44`, corrected by `96af943` before merge. The verify report cites the pre-rebase hash `6da8593`.
> **Closes**: the concern doc 0001 and ADR 0005 track as **G13**
> **Verify verdict**: **PASS** — see [`verify-report.md`](verify-report.md)
> **Canonical spec**: [`openspec/specs/ai-stream-lifecycle/spec.md`](../../../specs/ai-stream-lifecycle/spec.md)

---

## 1. Charter acceptance

| # | Charter clause | Outcome |
| --- | --- | --- |
| 1 | A recorded decision covering **carrier, ownership, cancellation, buffering and the failure-delivery split** | **PASS** — five decisions, five sections, in checklist order |
| 2 | AI-14 … AI-20 can be written **without reopening any of these five questions** | **PASS** — node-level inheritance stated for all four blocked milestones, and demonstrated by AI-03 being written from it in the same wave |
| 3 | AI-34's buffer measurement has a stated starting point to confirm or change | **PASS** — capacity 64, three measurements, a direction per result, and a tie-break rule |
| 4 | doc 0002's note: the carrier choice is free here and only here, and **the SDD must record why it chose what it chose** | **PASS** — the clause most at risk, and the one verification concentrated on |

Clause 4 is the load-bearing one. The documented default was channels and the decision is channels, so an artifact could have satisfied the letter of "decide the carrier" while restating the default — which doc 0002 anticipated by deleting half of the argument that previously supported it (*"the retired plan's recommendation to keep channels rested partly on 'switching now would invalidate a shipped signature guard and behavioural scenarios merged days ago'. Nothing is shipped, so that argument is void."*). The test applied at verify was to strike every sentence that could have been written before reading doc 0001 and doc 0003 and see whether a decision survived. It does.

---

## 2. What was delivered

One `[decision]` leaf: `decision.md`, 515 lines, no production code, no `make test` gate. Five decisions in AI-02.1's own closing-checklist order.

| # | Question | Decision |
| --- | --- | --- |
| 1 | **Carrier** (`V-STR-02`) | A **receive-only channel** at the package boundary. Iterator ergonomics delegated to AI-22.5 as a `V-STR-22` carrier view. doc 0002 needs no amendment nodes |
| 2 | **Ownership** (`V-STR-05`) | **One sending goroutine, one closing site, every exit path, after the last send attempt.** Nothing else closes |
| 3 | **Cancellation** (`V-STR-06`, `V-STR-07`) | Caller-owned signal; every send waits on it; bounded close. The legal endings are **drain to close** and **cancel**. Anything else is **abandonment** — a documented contract violation, stated in the package contract because it cannot be tested to termination |
| 4 | **Buffering** (`V-STR-08`, `V-STR-09`, `V-STR-23`) | Bounded, **starting capacity 64**, falsifiable at AI-34.1. Backpressure is waiting, never dropping. Exactly one sanctioned loss path: cancellation with a saturated buffer drops late events and closes without a terminal |
| 5 | **Failure delivery** (`V-FAIL-11`, `V-FAIL-12`) | The boundary is **the handover of the carrier**, not the first event. Before it, the failure is returned directly and no stream and no producer exist. After it, every failure arrives as the terminal error event |

**What made each item closed rather than merely discussed**, per `verify-report.md` § 3:

- **The carrier is argued, not restated.** The iterator case is stated at full strength first, with three of its four advantages conceded as structural. Four defeating grounds, each citing a source that was re-resolved rather than taken on the artifact's word — the decisive one is doc 0003's AG-10.3 item 1, a Layer 2 test **already written** that an iterator boundary would make unsatisfiable, present verbatim including *"message deltas already in flight keep flowing"*. No ground is cost-of-change; the only mention of that argument is the explicit statement that it is void. "Both carriers at the boundary" is named and priced. The residual cost of the chosen option — a consumer who abandons and never cancels — is conceded in bold and paid in the only currency available, a statement in the package contract restated at the freeze by AI-40.3.
- **Ownership is structural.** Three misreadings of "exactly once" are named with the defect each produces, and each is a *shape* — a second closing site, several senders, a consumer-side close — which is what makes them visible in a diff rather than only in a race.
- **Cancellation enumerates its legal endings first**, so that "violation" has a complement and a consumer with a legitimate need to stop early gets a one-word answer rather than a prohibition. The untestable clause is kept explicitly *outside* the testable set, because a test that proves something weaker replaces the clause it stands in for — defect **C3** is what that substitution looks like when it ships.
- **The buffer has one number**, priced at both ends of the range against Layer 2's documented behavior (its consumer pauses **by design**, to drive the permission protocol), with concurrency named as the multiplier and memory explicitly *not* the ground the number stands on. It is labelled a hypothesis, with three measurements, a direction per result in both directions, and a tie-break rule so AI-34.1 is not left arbitrating.
- **The delivery split is separated at an observable moment.** The case that distinguishes the correct boundary from the intuitive one — a stream handed over that fails *before emitting any content* is mid-stream — is stated explicitly, and "the first event" is named and rejected because it fuses the delivery axis onto the partial-output axis and rebuilds the **G8** defect.

### 2.1 Register amendment landed with this change

Two nouns were appended to AI-01's register in this same pull request, under its § 9 rule 2:

| Appended | Owner | Why the register lacked it |
| --- | --- | --- |
| `V-STR-22` **carrier view** | AI-02 | AI-02.1 delegates iterator ergonomics to AI-22.5 and had no noun for the delegated thing. Without one, every downstream restatement says "iterator view" — a phrase welded to one carrier choice |
| `V-STR-23` **backpressure** | AI-02 | The word **already appeared inside `V-STR-08`'s definition without being defined** — exactly the drift the register exists to prevent |

Measured: four added lines and two replacements, both of the register's own arithmetic, in a 328-line file. **No existing row was renumbered, reworded, reordered or removed.** Register total 109 → 111. Neither term is defined in this change's artifacts; both are cited by identifier, and § 1 states the discipline in the artifact's own voice — *"They are used here; they are not defined here."*

---

## 3. Where the contract now lives

The delta spec `specs/ai-stream-lifecycle/spec.md` and the five decisions were promoted to [`openspec/specs/ai-stream-lifecycle/spec.md`](../../../specs/ai-stream-lifecycle/spec.md).

That canonical spec **carries the five decisions in full** — the carrier with its four sourced grounds and the iterator case stated first, ownership across completion, terminal-error, cancellation and unwinding exits, cancellation with the abandonment-is-a-contract-violation statement, buffering bounded at 64 with the single sanctioned loss path and AI-34.1's three measurements, and the pre-stream/mid-stream failure-delivery split with its two-axis grid — plus the complete lifecycle diagram, the eight package-contract statements AI-20.1 must publish, and the node-level inheritance for every blocked milestone. It does not merely point at this archive.

The reason is § 11 rule 4 and § 6's own status. **The starting capacity of 64 is a hypothesis, not a measurement**, and AI-34.1's charter is to confirm or change it *with measurements*. That is an amendment to the contract, in the pull request that resumes work — which requires a live home. AI-14 … AI-23 also cite this contract by section number, so the section numbering `§ 1` … `§ 12` is preserved in the canonical spec and those citations keep resolving.

| Artifact | Role |
| --- | --- |
| `openspec/specs/ai-stream-lifecycle/spec.md` | **The contract.** Live, amendable under § 11 rule 4, cited by AI-03, AI-14, AI-19, AI-20, AI-21, AI-22, AI-23, AI-33, AI-34, AI-35 and AI-40 |
| `openspec/changes/archive/2026-07-31-cachicamas-ai-stream-lifecycle/decision.md` | **The historical record of how the contract was decided** — the same five decisions as they stood at merge, with AI-02.1's closing-checklist verification. Immutable |

**Deltas promoted**

| Kind | Identifiers |
| --- | --- |
| Requirements | `R-AIS-001` … `R-AIS-014` |
| Scenarios | `S-AIS-001` … `S-AIS-041` |

Change voice was rewritten into standing voice without moving an identifier. `R-AIS-001` now states that exactly one statement of this contract is normative and it is the canonical spec, instead of naming this change's `decision.md` path; `R-AIS-013`'s scope fence and `R-AIS-014`'s vocabulary discipline now bind every amendment rather than only this change; `S-AIS-041`'s diff-hygiene scenario now reads against "any change that states or amends this contract". `S-AIS-011` carries into the canonical spec with the qualification verify recorded — § 4 names all three exit paths plus the unwinding one explicitly, while the per-path *emission* statements live in §§ 6 and 7 and the § 8 diagram, which draws all three converging on one closing site.

---

## 4. Findings recorded at verify, and their disposition

`verify-report.md` § 7 records three. None blocked the verdict.

| # | Finding | Disposition |
| --- | --- | --- |
| 7.1 | **MINOR** — § 5 attributed to defect **C3** the words *"a shipped test documented the resulting gaps as expected behavior"*, inside quotation marks. doc 0001's C3 row reads *"A shipped test documents the resulting gaps as expected"* — the tense changed and a word was added | **Fixed before merge**, in commit `96af943`. The quotation now reads verbatim. Corrected before merge rather than at archive, because a paraphrase inside quotation marks in a document whose purpose is anti-paraphrase discipline is the one defect class this change is least entitled to carry. The two quotations carrying real argumentative weight — doc 0003 AG-10.3 item 1 and doc 0001 § 7 **G8** — were both checked and were already verbatim |
| 7.2 | § 10's downstream table says of AI-03 that cancellation and typed failure delivery **"are required capabilities"** — a standing only AI-03.1 may assign, phrased as fact rather than expectation | **No change made, and none needed.** `R-AIS-013`'s scope list names AI-14, AI-19, AI-20, AI-22.4, AI-34 and AI-35 — not AI-03 — so it is not a spec violation, and doc 0002 lists AI-03 as depending on AI-02, which makes a forward-looking note legitimate. AI-03 independently reached the same classification through its own admission test 1 and assigned `CAP-R-04` and `CAP-R-05` as required, so no contradiction landed. The sentence is carried into the canonical spec unchanged, where it is now a statement of what AI-03 in fact decided |
| 7.3 | `decision.md`'s header carries the repository directory path `backend/agent/src/ai/` as metadata, and `S-AIS-040` forbids "a package path" | **Not a violation.** It is a directory path, not a Go package path, and not a spelling of any Layer 1 surface. Recorded because the literal reading of the scenario is tighter than the constraint behind it |

---

## 5. Deliberately not done

Verified absent in `verify-report.md` § 9, each deliberate.

- **Nothing under `backend/`.** The change's commit touches it zero times; the ten `backend/` paths in the wave's range all belong to AI-00.
- **doc 0002 not amended.** No node added, none renumbered, no stated claim corrected. § 3's consequence 2 records this as a **positive result** of the channel branch rather than an omission: the waves 2–5 amendment nodes the iterator branch would have triggered under the living-graph clause are not needed, and the absence is stated so nobody later reads it as a gap.
- **No event kind, payload, sequence rule or ordering invariant.** AI-14's. `V-STR-18` is cited as an assumption of §§ 4, 6 and 7 and explicitly left to AI-14 to define.
- **No failure category, retryability rule or terminal-payload shape.** AI-19's. § 7's exclusions name the abstention: *"This decision fixes where a failure appears, never what it says."*
- **No leak-detection mechanism** (AI-22.4 named), **no retry rule** (AI-35 named), **no constant-versus-configurable ruling** on the buffer (AI-34.1 named).
- **Layer 2's carrier not decided.** § 11 rule 6: doc 0003 AG-01.1 owns it; this artifact is an input, *"a recommendation with reasons, not an inheritance."*
- **No claim that 64 is measured.** Stated three times — in § 6's "Why 64", in § 6's exclusions, and as § 11 rule 5, which makes citing 64 as settled a misreading and citing it after AI-34.1 publishes a stale citation.

---

## 6. Lifecycle

`explore → proposal → spec → design → tasks → decide → verify → archive` — all phases delivered. `tasks.md` records six tasks plus a ten-check verification pass, all `[x]`.

| Phase | File |
| --- | --- |
| Explore | `explore.md` |
| Proposal | `proposal.md` |
| Spec (delta) | `specs/ai-stream-lifecycle/spec.md` |
| Design | `design.md` |
| Tasks | `tasks.md` |
| Decision | `decision.md` — **superseded as the live contract by the canonical spec; retained here as the historical record** |
| Verify | `verify-report.md` |
| Archive | `archive-report.md` (this file) |

**Unblocked by this decision:** AI-14 (`cachicamas-ai-event-envelope`), AI-20 (`cachicamas-ai-model-provider`), AI-21 (`cachicamas-ai-fake-provider`), AI-22 (`cachicamas-ai-stream-testkit`) — and, through them, AI-03, AI-19, AI-23, AI-33, AI-34, AI-35 and AI-40. AI-03 was written from this decision in the same wave and re-decided none of the five questions, citing §§ 5 and 7 for the observable shapes of cancellation and failure delivery.
