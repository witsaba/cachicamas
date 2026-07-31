# Tasks — Layer 1 stream lifecycle, ownership and carrier

> **Change**: `cachicamas-ai-stream-lifecycle`
> **Milestone**: AI-02 · **Node**: AI-02.1 — Lifecycle, ownership and carrier `[decision]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-stream-lifecycle/spec.md`, `design.md`
> **Forecast**: **1 PR**, documentation only, zero Go
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Depends on**: AI-01 (`cachicamas-ai-contract-vocabulary`) merged
> **Blocks**: AI-14, AI-20, AI-21, AI-22 — and through them AI-19, AI-23, AI-33, AI-34, AI-40

---

## Node type and what it means for this task list

AI-02.1 is a **`[decision]` leaf**. Per doc 0002's node grammar:

> **Decision leaf** — `[decision]` — A recorded choice with a closing checklist. **No production code.** Closes when: the decision artifact answers every listed question and is merged.

Consequences that shape this file:

- There is **no test list**. Behavior leaves carry test lists; a decision leaf carries a closing checklist.
- There is **no red-green-refactor cycle**, and this is not a TDD exemption taken quietly — `openspec/config.yaml` sets `apply.tdd: true` for **Go service code**, and this change writes none.
- There is **no `make test` evidence gate**. doc 0002's global evidence gate binds behavior and guard leaves; a decision leaf closes on its merged artifact.
- The whole milestone is **one phase with one node**, so the PR-chain forecast is degenerate: one PR.

---

## Phase AI-02.1 — Lifecycle, ownership and carrier `[decision]`

Five tasks, one per closing-checklist item, in the checklist's own order — then the register amendment the decision depends on, then the verification pass.

**Deliverable of the whole phase:** `openspec/changes/cachicamas-ai-stream-lifecycle/decision.md`.

---

### T-AIS-1 — Carrier decided, with rationale

- [x] Decide the carrier at the package boundary, state the strongest case for the rejected option before rejecting it, and rebut each of its grounds against a cited source.

**Required by the checklist:** receive-only channel versus range-over-func iterator, decided with rationale; if channels win, the iterator-ergonomics requirement is delegated to AI-22.5 and the decision says so; if iterators win, doc 0002's waves 2–5 gain amendment nodes under the living-graph clause.

**Additional obligation this change accepts:** doc 0002 declares the sunk-cost half of the retired argument void. The rationale must therefore rest on nothing resembling it (`R-AIS-002`, `S-AIS-004`).

**Decided:** the **receive-only channel**. Four grounds, each sourced: Layer 2 must wait on a stream and on other things simultaneously and an iterator cannot be waited on (doc 0001 § 4.1; doc 0003 AG-10.3 item 1); the consumer must not be the socket reader; the terminal event is an element and an iterator has nowhere clean to put it (`V-STR-18`, `V-FAIL-10`); the iteration shape connotes a repeatable collection walk (`V-PRV-10`'s wrong-physics hazard, one level wider). "Offer both" named and rejected.

**Evidence:** `decision.md` § 3. Delegation to AI-22.5 stated; doc 0002 amendment nodes explicitly **not** required (`S-AIS-007`, `S-AIS-008`).

---

### T-AIS-2 — Ownership stated, not implied

- [x] State what "exactly once" binds, structurally, and separately for the completion, terminal-error and cancellation paths.

**Required by the checklist:** the producer creates the stream and closes it exactly once; nothing else closes it; the consumer never closes it; what "exactly once" means across the three paths is **stated, not implied**.

**Decided:** one goroutine sends; one closing site exists in the producer; it runs on every exit path including an unwinding one; it runs after the last send attempt and never before. Internal fan-in in an adapter happens below the boundary. The consumer, the test kit and every consumer above Layer 1 are named as parties that do not close.

**Evidence:** `decision.md` § 4. Three paths each carry their own statement (`S-AIS-011`); "no send after close" is derived rather than added as a separate rule (`design.md` § 3.3).

---

### T-AIS-3 — Cancellation stated, and abandonment placed in the package contract

- [x] State the three cancellation obligations, mark each testable or statable, enumerate the legal consumer endings, and place the abandonment clause in the package contract with AI-40.3 named.

**Required by the checklist:** the caller owns a cancellable context; every send waits on it; cancellation closes the stream within bounded time. Abandoning a stream without cancelling is a documented contract violation rather than a supported mode — a statement that must appear in the package contract because it cannot be tested to termination (AI-40.3 restates it at the freeze).

**Decided:** the two legal consumer endings are *drain to close* and *cancel*; anything else is abandonment. "Bounded" is defined by what it excludes — after cancellation is observable the producer begins no new blocking wait on the network or on the consumer, and a backoff waits on the signal rather than sleeping. The first two obligations are testable and handed to AI-20.3; abandonment is not, and is handed to the package contract.

**Evidence:** `decision.md` § 5, § 9. AI-20.1's third test item is satisfiable from the artifact verbatim (`S-AIS-018`).

---

### T-AIS-4 — Buffering decided, with a number and its falsification criteria

- [x] Decide the bound, choose one starting capacity, justify it against both ends of the range, state the sanctioned loss path, and hand AI-34.1 an experiment rather than an opinion.

**Required by the checklist:** a bounded buffer with a decided starting capacity, revisited with measurements at AI-34; the sanctioned loss path — a saturated buffer during cancellation drops late events and closes without a terminal — stated here and proven at AI-20.3.

**Decided:** bounded, **starting capacity 64**. Backpressure is waiting, never dropping. Exactly one sanctioned loss path: cancellation with a saturated buffer drops late events and closes without a terminal event; a consumer that treats a missing terminal after its own cancellation as corruption is the party in error, and a stream that closes bare without having been cancelled is a producer defect rather than a second loss path. Constant-versus-configurable is explicitly **deferred to AI-34.1**.

**Evidence:** `decision.md` § 6. The number is presented as a hypothesis with three named measurements and the direction each result implies (`S-AIS-022`).

---

### T-AIS-5 — Failure delivery split at an observable moment

- [x] State what a caller observes on each path, identify the separating moment, name and reject the alternative boundary, and keep the delivery axis orthogonal to the partial-output discriminator.

**Required by the checklist:** what a caller observes when the request never becomes a stream, versus when a stream dies mid-flight — the split AI-19.5 implements as one vocabulary over two delivery paths.

**Decided:** the boundary is the **handover of the carrier**. Before it, `V-FAIL-11`: the failure is returned directly and no stream and no producer ever exist. After it, `V-FAIL-12`: the failure arrives as the terminal error event and the stream then closes, with no second route. A stream that fails after handover but before any content is **mid-stream**; classifying it as pre-stream is the conflation with `V-FAIL-09` that doc 0001 § 7 **G8** records. "The first event" is named as the rejected boundary. Ordinary cancellation is answered too, because AI-20.2 item 2 already assumes an answer.

**Evidence:** `decision.md` § 7, including the two-axis grid (`design.md` § 3.4). AI-19.5's third test item is satisfiable from the artifact (`S-AIS-031`).

---

### T-AIS-6 — Register amendment (AI-01), append-only

- [x] Append the two nouns AI-02 needs to AI-01's register, with the next free `V-STR` ordinals and a dated amendment blockquote; update the register's own counts.

**Appended:** `V-STR-22` **carrier view** and `V-STR-23` **backpressure**. Both owned by AI-02. Rationale for each is in `proposal.md`.

**Discipline applied** (AI-01 § 9 rules 2 and 3): appended in the same pull request that needs them; not defined locally in this change's artifacts, which cite them by identifier only; no existing row renumbered, reworded or removed; the register's term counts and its checklist range reference updated so the artifact does not contradict its own arithmetic.

**Evidence:** `openspec/changes/cachicamas-ai-contract-vocabulary/decision.md` § 4.1 (`S-AIS-037`, `S-AIS-038`).

---

## Verification pass (closes the milestone)

Run after T-AIS-1 … T-AIS-6, ordered by cost of a missed defect rather than by document order (`design.md` § 7). Every check is inspection; nothing executes.

- [x] **V-1** — The carrier rationale states the opposing case affirmatively before rebutting it, every rebuttal cites a source, and no ground is cost of change (`R-AIS-002`).
- [x] **V-2** — The delivery boundary is the handover of the carrier; "the first event" is named as rejected; the stream-that-fails-before-content case is classified mid-stream (`R-AIS-012`).
- [x] **V-3** — "Exactly once" is stated separately for completion, terminal error and cancellation, plus the unwinding exit (`R-AIS-005`).
- [x] **V-4** — The abandonment clause is present, assigned to the package contract, justified by untestability, and attributed to AI-40.3; the legal consumer endings are enumerated (`R-AIS-008`).
- [x] **V-5** — One capacity number is stated, justified at both ends of the range, and accompanied by the measurements that would move it and in which direction; constant-versus-configurable is deferred (`R-AIS-009`, `R-AIS-010`).
- [x] **V-6** — Deletion test over every normative sentence: no sentence removes an option from AI-14, AI-19, AI-20, AI-22.4, AI-34 or AI-35 (`R-AIS-013`).
- [x] **V-7** — Every Layer 1 noun in a normative sentence resolves to a register row by identifier; the two appended rows are append-only and the register's counts are consistent (`R-AIS-014`).
- [x] **V-8** — No Go type, field, method, interface or package identifier appears in any file of the change; language and standard-library shapes are named descriptively (`S-AIS-040`).
- [x] **V-9** — The inheritance section names AI-14, AI-20, AI-21, AI-22 and AI-34, each in that milestone's own terms, so the acceptance criterion is checkable from one table (`S-AIS-039`).
- [x] **V-10** — The diff contains only markdown under `openspec/changes/`; nothing under `backend/`, no build, module or infrastructure file (`S-AIS-041`).

---

## Review focus

For the reviewer, in priority order — the first three are where a defect is expensive, the rest where it is cheap to catch:

1. **A restated default.** Read § 3 of `decision.md` backwards: start at the conclusion and check that each supporting ground is a citation rather than an assertion. doc 0002 gave this milestone one job beyond deciding — recording *why*. An artifact that reaches the documented default without argument has not done it.
2. **The delivery boundary.** The one case that separates the correct boundary from the intuitive one is a stream handed over that fails before emitting content. Find that case in the artifact. If it is not there, the **G8** defect is being rebuilt.
3. **"Exactly once" on the cancellation path.** The completion path is easy and the error path is nearly as easy. Cancellation is where a second closing site appears, and it is the path doc 0002 singled out by requiring the meaning to be *stated, not implied*.
4. **Scope creep into AI-14 and AI-19.** The line is: this milestone decides the container, not the contents. A sentence about what an event carries, or about how a failure is classified, is over-reach even when correct.
5. **An unfalsifiable number.** 64 is defensible. "64, because it feels right" is not, and the difference is whether the measurement that would change it is written down.
6. **Register discipline.** Two rows were appended to a merged artifact. Check that nothing else in it moved, and that this change defines neither term locally.
7. **Leaked Go identifiers.** Term names are noun phrases with spaces. A single-token camel-case name, a package path, or a method-shaped name is the anomaly to look for — including in the carrier argument, where standard-library shapes are named descriptively on purpose.

---

## PR forecast and review budget

| PR | Content | Forecast | Depends on |
| --- | --- | --- | --- |
| 1 | six markdown artifacts under the change directory, plus an append-only amendment to AI-01's register | ~1,300 lines of prose, **0 Go** | AI-01 merged |

doc 0002's review budget — "prefer less than 250 changed lines; stop and reassess before 400" — is a **code** budget, expressed in the same document that requires each milestone's SDD to carry proposal, spec, design and tasks artifacts. A decision leaf's diff is entirely those artifacts.

No chaining applies, and the reason is the same one `explore.md` § 2 gives for the milestone's existence: the five questions are coupled, and splitting a coupled decision across pull requests produces exactly the partial-decision state that defect **C4** came from.

---

## Acceptance criteria for the milestone

1. All five closing-checklist items (T-AIS-1 … T-AIS-5) are answered in `decision.md`, each with rationale.
2. The register amendment (T-AIS-6) is merged in the same pull request.
3. The verification pass V-1 … V-10 is recorded as complete.
4. `spec.md`'s `R-AIS-001` … `R-AIS-014` hold.
5. The change adds six markdown files and amends exactly one existing file, append-only.
6. **doc 0002's own acceptance criterion:** AI-14 … AI-20 can be written without reopening any of the five questions, and AI-34's buffer measurement has a stated starting point to confirm or change.

## Next

- **AI-03** — `cachicamas-ai-minimum-capabilities`: the capability matrix and discovery mechanism. Depends on AI-01 and AI-02; consumes `V-PRV-06` … `V-PRV-09`, and inherits from this decision the fact that cancellation and typed failure delivery are *required* capabilities whose observable shape is already fixed.
- Then wave 1 (AI-04 … AI-13), which knows nothing about streams, and wave 2 (AI-14 … AI-20), which is where every line of this decision is spent.
