# Tasks — the v1 capability set and optional-capability discovery

> **Change**: `cachicamas-ai-minimum-capabilities`
> **Milestone**: AI-03 · **Node**: AI-03.1 — The capability matrix `[decision]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-minimum-capabilities/spec.md`, `design.md`
> **Forecast**: **1 PR**, documentation only, zero Go
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Depends on**: AI-01 (`cachicamas-ai-contract-vocabulary`) and AI-02 (`cachicamas-ai-stream-lifecycle`) merged
> **Blocks**: AI-20.5, AI-23, AI-24 — and through them AI-29.0, AI-38, AI-40

---

## Node type and what it means for this task list

AI-03.1 is a **`[decision]` leaf**. Per doc 0002's node grammar:

> **Decision leaf** — `[decision]` — A recorded choice with a closing checklist. **No production code.** Closes when: the decision artifact answers every listed question and is merged.

Consequences that shape this file:

- There is **no test list**. Behavior leaves carry test lists; a decision leaf carries a closing checklist.
- There is **no red-green-refactor cycle**, and this is not a TDD exemption taken quietly — `openspec/config.yaml` sets `apply.tdd: true` for **Go service code**, and this change writes none.
- There is **no `make test` evidence gate**. doc 0002's global evidence gate binds behavior and guard leaves; a decision leaf closes on its merged artifact.
- The whole milestone is **one phase with one node**, so the PR-chain forecast is degenerate: one PR.

This is the **last node of wave 0**. On its merge, doc 0002's wave-0 exit condition holds: "The module exists, both import directions bite, and vocabulary, stream lifecycle, carrier and capability scope are recorded decisions."

---

## Phase AI-03.1 — The capability matrix `[decision]`

Five tasks, one per closing-checklist item, in the checklist's own order — then the register amendment the decision depends on, then the verification pass.

**Deliverable of the whole phase:** `openspec/changes/cachicamas-ai-minimum-capabilities/decision.md`.

---

### T-AIC-1 — Required capabilities enumerated

- [x] Enumerate the required capabilities, each with what it obliges, what it does **not** oblige, and the admission-test clauses that give it its standing.

**Required by the checklist:** streaming text, tool calls, completion metadata (finish reason and usage), cancellation, typed failures with the partial-output distinction.

**Additional obligation this change accepts:** the expensive defect in a required list is not an omission but an entry that forces an honest adapter to fabricate data (`design.md` § 2.1). Every entry therefore carries an explicit negative clause.

**Decided:** five entries, closed — `CAP-R-01` streaming text, `CAP-R-02` tool calls, `CAP-R-03` completion metadata, `CAP-R-04` cancellation, `CAP-R-05` typed failures with the partial-output distinction. The two negative clauses that matter most: requiring the usage record does **not** require any token count to be populated (`V-MET-10`, `V-MET-11`, AI-13.3), and requiring the finish-reason vocabulary to be reachable does **not** require every value to be emitted (`V-MET-08` is a conformant mapping). A block delivered whole with zero deltas is conformant; an adapter minting identifiers for a vendor that assigns none is satisfying `CAP-R-02`, not lacking it. `CAP-R-04` and `CAP-R-05` cite AI-02 §§ 5 and 7 for their observable shapes rather than re-deciding them.

**Evidence:** `decision.md` § 5 (`R-AIC-004`, `S-AIC-008` … `S-AIC-012`).

---

### T-AIC-2 — Optional capabilities enumerated, each with its reason

- [x] Enumerate the optional capabilities, each with the reason it is optional **rather than required**, and state whether the list is closed and how a new entry is admitted.

**Required by the checklist:** reasoning content, token counting, honoring cache-boundary markers, and anything else v1 admits — each with the reason it is optional rather than required.

**Decided:** three entries, closed — `CAP-O-01` reasoning content, `CAP-O-02` token counting, `CAP-O-03` honoring cache-boundary markers. "Anything else v1 admits" resolves to nothing: five further candidates were run through the admission tests and each failed one, and the reasons are recorded so the rejections do not have to be re-derived. A fourth entry arrives by amendment in the pull request that needs it, because each entry costs AI-23 a suite case, AI-38 a record entry and AI-24.1 a recorded expectation.

**The load-bearing entry, in full:** `CAP-O-02` token counting. The opposing reading is stated at its strongest first — a consumer genuinely needs a real count, and doc 0001 § 6 seam 6 says estimating by character count "is wrong by enough to matter" — and is defeated on universality: a required count leaves an adapter whose vendor offers none exactly two options, failing conformance for a reason that is not a defect, or fabricating. A fabricated count corrupts a compaction decision silently, where an absent one degrades to a visible estimate. **Corollary, stated as a standing rule:** Layer 1 never supplies a fallback estimate for an absent capability, because a default implementation that estimates is a fabrication with better provenance.

**The arguable entry:** `CAP-O-03`. The case for calling it an adapter-local mapping obligation is stated first — markers are advisory by contract (`V-REQ-23`) and AI-11.3 proves ignoring them wholesale is conformant — and is answered by `V-REQ-24`, the breakpoint cap: a provider that honors markers enforces a small hard cap whose breach is a caller-contract failure, and a provider that caches automatically has none. That difference changes what a consumer may legally construct, which is what makes it consumer-visible and therefore a capability.

**Evidence:** `decision.md` § 6 (`R-AIC-005`, `R-AIC-006`, `R-AIC-008`).

---

### T-AIC-3 — Exclusions enumerated, with reasons and a rule

- [x] Enumerate what v1 excludes, each with its reason, and state the rule that separates an exclusion from an optional capability.

**Required by the checklist:** multimodal content beyond text (it needs per-provider capability detection v1 does not model), embeddings, batch APIs, server-side tool execution.

**Decided:** four entries — `CAP-X-01` multimodal content beyond text, `CAP-X-02` embeddings, `CAP-X-03` batch APIs, `CAP-X-04` server-side tool execution. The separating rule: **an optional capability has a defined absence; an excluded one has no defined presence.** There is nothing to advertise, discover or record, because the neutral vocabulary does not model what a positive answer would mean.

Two reasons are principled rather than merely scope-based, and the decision says which: `CAP-X-02` would require a provider to return something other than a normalized event stream, which widens the very contract item 4 forbids widening; `CAP-X-04` would place tool execution below Layer 1, routing around the permission protocol (`V-OUT-05`, **G1**) and the sandbox (`V-OUT-16`, **G2**) — a trap-1 violation, not a scope choice.

**Evidence:** `decision.md` § 7 (`R-AIC-014`).

---

### T-AIC-4 — The discovery mechanism decided, both halves

- [x] Decide the mechanism inside the inherited family; state how an adapter advertises, how a consumer asks, what absence looks like, and which alternatives are excluded.

**Required by the checklist:** an optional capability is an additional, separately-asserted contract on the provider value, so the core provider interface never widens. How an adapter advertises it, and how a consumer asks, are both stated.

**Inherited, not chosen:** the mechanism *family* is stated by the checklist itself. Unlike AI-02's carrier, this is not a free choice, and the decision says so rather than pretending to have argued it. Five sub-questions inside the family were open and are decided.

**Decided:** one separately-asserted contract **per capability** (never one aggregate — an aggregate is all-or-none and cannot grow without breaking adapters that already satisfy it), declared in Layer 1 beside the provider contract so the set is enumerable, and asked **of the provider value** — never of a model identity, a configuration entry or a catalog (`V-OUT-14`). An adapter advertises by satisfying the contract and by no other means; an adapter without the capability declares nothing at all. A consumer observes a clean absence — not an error, not a zero (AI-20.5 item 1's words). **Advertising binds**: a provider that satisfies an optional contract and then declines is non-conformant, not absent. **Wrappers forward or document**: a wrapper that forwards nothing removes every optional capability invisibly, and AI-37 is the first wrapper this binds.

**Evidence:** `decision.md` § 9 (`R-AIC-010`, `R-AIC-011`). Four alternatives named and rejected: a widened core contract, a provider-returned capability list, a configuration- or catalog-driven table, and a single aggregate contract.

---

### T-AIC-5 — The capability record sketched, with "absent" as a recorded outcome

- [x] Sketch the record AI-23.6 emits and AI-38.2 asserts: what it carries, what makes it total, its closed outcome set, and its verdict rule.

**Required by the checklist:** "Absent" is a **recorded outcome**, not an unrun test: the shape of the capability record AI-23.6 emits and AI-38.2 asserts is sketched here.

**Decided:** a record carries its subject (which provider, which run) and **one entry per capability in the closed lists, required and optional alike** — totality, which is only possible because both lists are closed, and which makes a missing entry a defect in the run rather than an absence. Each entry carries the capability, its **standing** (taken from this decision, never from the run, so a run cannot demote a required capability) and one **outcome** from a closed four-value set: *satisfied*, *absent*, *failed*, *not exercised*.

**The item's whole content, made mechanical:** *absent* and *not exercised* are different values. Absent is a conclusion — asked, does not advertise, deliberately not exercised, fully conformant. Not exercised is the **absence** of a conclusion — a missing transcript, a harness error, an interrupted run. The verdict rule follows: a record passes when every required entry is *satisfied*, every optional entry is *satisfied* or *absent*, and no entry is *not exercised*; a record containing any *not exercised* entry is **inconclusive**, not failing and not passing.

**Also decided:** records are comparable entry by entry, because AI-24.1 records an expectation before an adapter exists and AI-38.2 asserts the generated record against it; a difference in either direction is a finding. The record carries no capability-specific detail, no model content, no credentials and no raw provider text — AI-40.2 publishes it, so `V-FAIL-13` and `V-FAIL-14`'s posture binds.

**Evidence:** `decision.md` § 10 (`R-AIC-012`, `R-AIC-013`).

---

### T-AIC-6 — Register amendment (AI-01), append-only

- [x] Append the three nouns AI-03 needs to AI-01's register, with the next free `V-PRV` ordinals and a dated amendment blockquote; update the register's own counts.

**Appended:** `V-PRV-16` **capability**, `V-PRV-17` **token counting**, `V-PRV-18` **capability outcome**. All three owned by AI-03. Rationale for each is in `proposal.md`.

**Discipline applied** (AI-01 § 9 rules 2, 3 and 5): appended in the same pull request that needs them; not defined locally in this change's artifacts, which cite them by identifier only; no existing row renumbered, reworded or removed; the register's term counts updated so the artifact does not contradict its own arithmetic; and **each definition defers its substance to AI-03 by name**, the way `V-PRV-08` defers the discovery mechanism — a register row stating which outcome values exist would be AI-01 deciding AI-03's matrix retroactively.

One row is different in kind: `V-PRV-16` **capability** closes a gap AI-01 identified in its own text. Its § 7 preamble names five terms "AI-03's charter is not writable without" and its table delivers four.

**Evidence:** `openspec/changes/cachicamas-ai-contract-vocabulary/decision.md` § 7 (`S-AIC-053`, `S-AIC-054`, `S-AIC-055`).

---

## Verification pass (closes the milestone)

Run after T-AIC-1 … T-AIC-6, ordered by cost of a missed defect rather than by document order (`design.md` § 7). Every check is inspection; nothing executes.

- [x] **V-1** — Token counting is optional; the opposing reading is stated at its strongest and defeated on universality rather than on effort; the no-fallback-estimate corollary is present as a standing rule (`R-AIC-005`).
- [x] **V-2** — *Absent* and *not exercised* are distinct values of a closed outcome set, and the verdict rule makes a record with any *not exercised* entry inconclusive rather than passing (`R-AIC-013`).
- [x] **V-3** — The marking rule is a biconditional over the optional list with a **required** default, and the artifact says why the required list cannot be the marking source (`R-AIC-009`).
- [x] **V-4** — Every required capability states what it does **not** oblige; the usage and finish-reason clauses are present (`R-AIC-004`, `S-AIC-009`, `S-AIC-010`).
- [x] **V-5** — Every optional capability carries the reason it is optional rather than required, plus what a consumer does on a recorded absence; the list is stated closed with an amendment route (`R-AIC-006`).
- [x] **V-6** — Advertising binds, and the wrapper forwarding rule is present with AI-37 named (`R-AIC-010`, `R-AIC-011`).
- [x] **V-7** — The nine-row leakage cross-check is present with a verdict per row, and the divergence rule is stated (`R-AIC-007`).
- [x] **V-8** — Deletion test over every normative sentence, run hardest against AI-24.1 and AI-29.0: no sentence removes an option from AI-11, AI-13, AI-19, AI-20, AI-23, AI-24 or AI-29 (`R-AIC-015`).
- [x] **V-9** — Every Layer 1 noun in a normative sentence resolves to a register row by identifier; the three appended rows are append-only, their definitions defer to AI-03, and the register's counts are consistent (`S-AIC-052` … `S-AIC-055`).
- [x] **V-10** — No Go type, field, method, interface or package identifier appears in any file of the change; language mechanisms are named descriptively (`S-AIC-058`).
- [x] **V-11** — The inheritance section names AI-20.5, AI-23 and AI-24, each in that milestone's own terms, so the acceptance criterion is checkable from one table (`S-AIC-056`).
- [x] **V-12** — The diff contains only markdown under `openspec/changes/`; nothing under `backend/`, no build, module or infrastructure file (`S-AIC-059`).

---

## Review focus

For the reviewer, in priority order — the first three are where a defect is expensive, the rest where it is cheap to catch:

1. **The token-counting argument.** Read § 6's entry for `CAP-O-02` backwards: start at "optional" and check that the ground is *universality without a lie*, not adapter convenience. doc 0002 predicts this being reopened, and the reopening always arrives dressed as "but compaction needs a real count" — which the artifact concedes, because it is true and it is not the point. If the artifact does not concede it, the argument is weaker than it looks.
2. **The two outcome values that look alike.** Find *not exercised* in § 10 and check the verdict rule treats it as inconclusive rather than as a pass or a failure. A four-value set that collapses to three under the verdict rule has defeated `V-PRV-09` while appearing to honor it.
3. **The negative clauses on the required list.** For each of the five, ask: what is the most demanding honest reading of this obligation, and would an adapter have to invent data to satisfy it? The two known cases are inside `CAP-R-03`. If a third exists, it is in `CAP-R-01` or `CAP-R-02` and it will look like a reasonable minimum.
4. **Scope creep into two decision nodes.** AI-24.1 chooses a vendor's capability expectations; AI-29.0 chooses whether the first adapter emits reasoning. This artifact's only legitimate contribution to the second is *making both answers legal*. A sentence that recommends either answer removes AI-29.0's reason to exist.
5. **The optional list's length.** Three. If a fourth appeared, check it against the nine-row cross-check in § 8 — the most likely source of a spurious entry is a leakage row read as a provider capability.
6. **Register discipline.** Three rows were appended to a merged artifact. Check that nothing else moved, that this change defines none of them locally, and — the clause that is easy to miss — that each appended definition **defers** rather than decides.
7. **Leaked Go identifiers.** Term names are noun phrases with spaces. The tempting place to slip is § 9, where a language mechanism is being described; it is named descriptively on purpose.

---

## PR forecast and review budget

| PR | Content | Forecast | Depends on |
| --- | --- | --- | --- |
| 1 | six markdown artifacts under the change directory, plus an append-only amendment to AI-01's register | ~1,400 lines of prose, **0 Go** | AI-01 and AI-02 merged |

doc 0002's review budget — "prefer less than 250 changed lines; stop and reassess before 400" — is a **code** budget, expressed in the same document that requires each milestone's SDD to carry proposal, spec, design and tasks artifacts. A decision leaf's diff is entirely those artifacts.

No chaining applies. The five checklist items are one classification process applied to one set of candidates: splitting the required list from the optional list across pull requests would produce a state in which a behavior is on neither list, and "on neither list" is what the marking rule's default exists to make impossible.

---

## Acceptance criteria for the milestone

1. All five closing-checklist items (T-AIC-1 … T-AIC-5) are answered in `decision.md`.
2. The register amendment (T-AIC-6) is merged in the same pull request.
3. The verification pass V-1 … V-12 is recorded as complete.
4. `spec.md`'s `R-AIC-001` … `R-AIC-015` hold.
5. The change adds six markdown files and amends exactly one existing file, append-only.
6. **doc 0002's own acceptance criterion:** "AI-23's suite can mark each case required or optional from this list alone; a provider lacking an optional capability is fully conformant and records 'absent' rather than skipping silently." The first half is the marking rule of § 11; the second is § 6's standing plus § 10's outcome set.

## Next

- **Wave 0 closes on this merge.** doc 0002's exit condition — module, both guards biting, vocabulary, stream lifecycle, carrier and capability scope recorded — holds.
- **Wave 1 (AI-04 … AI-13)** begins, and knows nothing about capabilities: it builds the neutral request from the failure vocabulary outward. Two of its milestones are cited by this decision and constrained by nothing in it — AI-11 owns cache-boundary markers, AI-13 owns finish reasons and usage.
- **The first consumers of this artifact are in waves 2, 3 and 4**: AI-20.5 implements the mechanism, AI-23 marks every case against these lists, and AI-24.1 records the first expected capability record.
