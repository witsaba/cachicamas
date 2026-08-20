# Spec — Compaction: the call, the surgery, the record (`agent-compaction`)

> **Change**: `cachicamas-agent-compaction` · **AG-18** (Layer 2, Wave 4, milestone 18 of 24), charter `0003:1655-1760`
> **NEW capability**, minted by this change (AG-18). Promoted to `openspec/specs/agent-compaction/spec.md` at archive.
> **Nodes**: AG-18.1 `[leaf]` (the compaction call) · AG-18.2 `[leaf]` (invariant-safe surgery) · AG-18.3 `[leaf]` (recorded on stream) · AG-18.4 `[leaf]` (interruption recovery) · AG-18.5 `[leaf]` (on-demand entry point)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && go test -race -count=1 ./...`. **`-count=1` is mandatory**: the real uncached suite for this module is ~3 minutes, so a sub-second pass is a cache artifact and not evidence.
> **IDs**: `R-CMP-0NN` / `S-CMP-0NN`, **append-only**. Allocated `R-CMP-001`…`R-CMP-015` and `S-CMP-001`…`S-CMP-042` (41 defined: `S-CMP-009` was allocated but never written, and the range is left with that gap rather than renumbering stable ids), of which `S-CMP-030`, `S-CMP-031`, `S-CMP-032` and `S-CMP-033` are the four **bites**. The header states the allocated **range** and never a total, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`). Prefix `CMP` verified free under `openspec/` before minting. **No total is stated** — a count goes silently false the moment a later milestone appends (`S-LSK-020`). *(AG-20: `R-CMP-015` and `S-CMP-038`…`S-CMP-042` added; none of the four bites is among them.)*
> **Sources**: charter `0003:1655-1760`; the `cachicamas-agent-compaction` change's proposal (Deliverable 0 A1–A8, binding) and design (**AD-1…AD-12, binding and not re-opened here**); `sdd-design` ran before `sdd-spec` deliberately and in series, and where this spec differs from the proposal it follows the design. Archived at `openspec/changes/archive/2026-08-19-cachicamas-agent-compaction/`.
> **Ownership boundary**: this capability owns compaction's execution, its cut resolution, its protection and typing obligations, its stream record, its atomicity and its on-demand door. It does **not** own the transcript store and its commit primitive ([`../agent-history/spec.md`](../agent-history/spec.md)), the verdict carrier ([`../agent-context-strategy/spec.md`](../agent-context-strategy/spec.md)), the cost payload vocabulary ([`../agent-cost-events/spec.md`](../agent-cost-events/spec.md)), the run algorithm ([`../agent-run-driver/spec.md`](../agent-run-driver/spec.md)) or the substrate rule ([`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md)).
> **Every `file:line` cited below was opened in this worktree during this phase, against `origin/main@f6acc0d2`.** No citation is carried forward from `explore.md`, `proposal.md` or `design.md` unresolved.

## Purpose

The runtime can measure its context (`R-CTX-007`) and can ask whether to shrink it (`R-CTX-001`). It cannot shrink it: AG-17's verdict type carries **no field** (`R-CTX-003`) and the harness discards what `Resolve` returns (`harness.go:524-530`, re-read this phase). The transcript therefore only ever grows.

AG-18 is the mechanism, and it is the first Layer 2 milestone that **removes** something a run committed. That is why this spec's central obligation is not "add compaction" but **"for every MUST relaxed, name the property that replaces it"** — discharged in the relaxation ledger below and in this change's `agent-history` delta.

## Coverage — the charter's eight Gherkin scenarios, each mapped

Every charter leaf traces to at least one requirement **and** at least one scenario. No leaf is reduced and no scenario is orphaned.

| # | Leaf | Charter scenario | Lines | Owning requirement(s) | Scenario(s) |
|---|---|---|---|---|---|
| 1 | AG-18.1 | "compaction is its own model call with its own spend" | `0003:1683-1687` | `R-CMP-002`, `R-CMP-003` | `S-CMP-001`, `S-CMP-003`, `S-CMP-004` |
| 2 | AG-18.1 | "the instruction is injected, never authored" | `0003:1689-1692` | `R-CMP-002` | `S-CMP-002`, `S-CMP-005` |
| 3 | AG-18.2 | "the replaced span never splits a pair" | `0003:1702-1706` | `R-CMP-004`, `R-CMP-005` | `S-CMP-010`, `S-CMP-011`, `S-CMP-013`, bite `S-CMP-030` |
| 4 | AG-18.2 | "protected turns are untouched and the summary is typed" | `0003:1708-1712` | `R-CMP-006`, `R-CMP-007` | `S-CMP-014`, `S-CMP-015`, `S-CMP-016` |
| 5 | AG-18.3 | "the stream records what compaction did" | `0003:1722-1725` | `R-CMP-008`, `R-CMP-009` | `S-CMP-017`, `S-CMP-018`, `S-CMP-019` |
| 6 | AG-18.4 | "compaction is atomic-or-absent" | `0003:1735-1738` | `R-CMP-010` | `S-CMP-020`, `S-CMP-021`, bite `S-CMP-031` |
| 7 | AG-18.5 | "demanded and strategy-triggered compaction are one path" | `0003:1748-1751` | `R-CMP-001`, `R-CMP-011` | `S-CMP-022`, `S-CMP-023` |
| 8 | AG-18.5 | "mid-turn demands are refused typed" | `0003:1753-1756` | `R-CMP-011` | `S-CMP-024`, `S-CMP-025`, bite `S-CMP-033` |

Cross-cut requirements carrying no charter leaf of their own: `R-CMP-012` (turn attribution and its named v1 limitation), `R-CMP-013` (named consequences and the no-new-vocabulary fence), `R-CMP-014` (closed-sequence safety), `R-CMP-015` (AG-20: the pre-compact splice and the re-verification of `R-CMP-001`/`R-CMP-003`/`R-CMP-010`/`R-CMP-013`/`R-CMP-014` for a hook-carrying path).

## The relaxation ledger — every relaxed MUST names its replacement

Binding on every requirement below and reproduced in the `agent-history` and `agent-context-strategy` deltas so no reader meets one half without the other. **A relaxation with nothing put in its place is this change's characteristic failure mode.**

| Relaxed | Replacement invariant |
|---|---|
| `R-HIS-001` "MUST NOT support removal" | Exactly **one** removal shape exists — a **prefix** replacement reachable only through the commit primitive of `R-HIS-004`. Its boundary MUST end at a recorded turn-mark boundary, MUST be pairing-closed, and the post-replacement transcript MUST satisfy the same append-time pairing rules seeded construction enforces. Reordering and in-place mutation of a surviving entry remain MUST NOT on **every** route, compaction's included. |
| `R-HIS-005` "Identity MUST be stable" | Stability is scoped to a **transcript generation**; a compaction begins a new generation. Identity stays ordinal-derived and caller-unmintable, so the guarantee that matters — no caller mints an identity — is untouched. `TurnID` is the durable cross-generation handle. |
| `R-CTX-003` verdict unconstructible | The never-compact guarantee moves to the **zero value**: a verdict requesting no compaction requests nothing on any path, and the shipped default returns exactly that. |
| `R-CST-001` per-path table completeness | **Nothing is relaxed**: the iff sentence ships byte-unchanged and is satisfied on every arm by construction (`R-CMP-003`). The table gains one additive row so it does not silently under-enumerate. |
| `L2C-07`'s two clauses | The route enumeration re-closes at **four** with compaction named; "stable ordinal entry identity" becomes generation-stable. Both clauses amended together, byte-in-sync across `doc.go` and `doc_contract_guard_test.go`. |

## Requirements

### R-CMP-001 — Compaction is ONE operation reachable through exactly two doors, and both run with no turn open

The system MUST implement compaction as a **single** internal operation. Exactly two doors MUST reach it: the context strategy's verdict (`R-CTX-003` as amended by this change) consulted at the run driver's turn boundary, and an on-demand harness entry point (`R-CMP-011`). Neither door MAY carry its own copy of the mechanics.

Both doors MUST execute the operation in a state where **no turn bracket is open**. On the strategy door this is the gap after the previous turn's close (`harness.go:668`) and before the next `Turn` call; on the on-demand door it is guaranteed by the refusal gate of `R-CMP-011`.

At most **one** compaction MUST run per turn boundary. `R-CTX-001` consults the strategy exactly once per logical turn and the verdict is the only request carrier on that door, so a second compaction at the same boundary is unrequestable rather than merely unimplemented.

#### Scenarios

- **S-CMP-022** — **Charter AG-18.5 scenario 1, the shared-mechanics half.** Given one recorded run whose strategy requests a compaction at a turn boundary and one on-demand invocation over an equivalent transcript, when both recorded streams are reduced to their **compaction turn bracket sub-sequence** (`turn_start` through its matching `turn_end`), then the two sub-sequences are equal `Kind()`-for-`Kind()` in order, excluding the fresh `RunID` and `TurnID` values; and both histories read back with the same number of entries and the same origin sequence.
- **S-CMP-026** — Given a run driven by a strategy that returns a compaction request on **every** consultation, when the run completes, then the number of compaction brackets on the recorded stream equals the number of logical turns the run performed — never more — proving at most one compaction per turn boundary.

---

### R-CMP-002 — The compaction call is its own model call, and Layer 2 authors no instruction content on any path

The compaction call MUST use the **injected** model provider carried by the request, which MAY differ from the harness's own provider, and the injected options carried by the same request. It MUST issue exactly one non-tool completion request and MUST NOT enter the tool-continuation path.

The summarization instruction MUST arrive **injected**. Layer 2 production code MUST NOT author, prefix, suffix, template or default instruction text on any path, including error and fallback paths (charter `0003:1662`; `R-AGS-015`). A request with no instruction MUST fail typed rather than acquire a Layer 2-authored one.

The call MUST be issued on the run's own context so it participates in the existing cancellation tree. It MUST NOT introduce a new cancellation primitive, a compaction-only cancel signal, or a separate deadline.

#### Scenarios

- **S-CMP-001** — **Charter AG-18.1 scenario 1, the own-call half.** Given a harness whose own provider is instance A and a compaction request carrying a **distinct** provider instance B, when a compaction executes, then B's recorded request count is exactly one and A's recorded request count is unchanged from its pre-compaction value — the compaction call is proven to have gone to the injected provider, not to the run's.
- **S-CMP-002** — **Charter AG-18.1 scenario 2.** Given a scripted instruction string injected alongside provider and options, when the compaction call runs, then the request the injected provider **captured** carries that exact instruction, and no message part of that captured request carries any text the runtime authored; the assertion reads captured request bytes, never a source comment.
- **S-CMP-003** — **Charter AG-18.1 scenario 1, the cancellation half.** Given a compaction whose injected provider is held by a test gate, when the run's cancellation is fired while the call is outstanding, then the compaction bracket closes aborted carrying a typed failure, the harness value survives and remains usable for a later run, and no `compaction_finished` appears anywhere on the stream.
- **S-CMP-005** — Given a compaction request carrying an empty instruction, when compaction is attempted, then it fails typed, `compaction_failed` is emitted, no provider request is issued, and the transcript read-back is unchanged.

---

### R-CMP-003 — Compaction spend is reported inside the compaction bracket and folded into the run cumulative

Compaction spend MUST be reported as a `cost_turn` event carrying the compaction completion's `ai.Usage`, labelled `Final`, emitted **inside the compaction turn bracket and before that bracket's `turn_end`**. No new `CostLabel`, no new `EventKind` and no new payload type MUST be introduced; distinguishability comes from the bracket's own `TurnID`, correlatable through the finished event's span.

The spend MUST reach the run's cumulative figure, so the run's `cost_session` includes it on every close path.

**The fold is explicit at the emission site.** The exploration's claim that the existing forwarder folds compaction spend automatically is **FALSE and MUST NOT be repeated**: the accumulating fold at `harness.go:567-576` drains `turnSink`, a channel created per attempt at `harness.go:563` and passed **only** to `Turn(...)` at `harness.go:590`. Compaction events never traverse it. The emission site MUST therefore perform the accumulation itself, against the same run-scoped accumulator.

`R-CST-001`'s iff MUST hold **on every arm** and MUST NOT be relaxed to make room for this bracket:

| Arm | Bracket close | `cost_turn`? |
|---|---|---|
| Completion obtained, commit succeeds | non-aborted (`Finished`) | **Yes** |
| No completion — provider error or cancellation | **aborted** + typed failure | **No** |
| Completion obtained, then rejected (non-`Stop` finish, or commit validation failure) | non-aborted, outcome mapped from the completion's own finish reason | **Yes** — the spend is real and is reported |

Only a completion whose finish reason is `Stop` MUST admit its summary; any other finish reason takes the failed arm with the spend still counted, because a length-truncated summary is silently lossy.

#### Scenarios

- **S-CMP-004** — **Charter AG-18.1 scenario 1, the spend half.** Given a compaction whose injected provider is scripted with a distinct, non-zero `ai.Usage`, when the run completes, then a `cost_turn` labelled `Final` appears inside the compaction bracket before its `turn_end` carrying that usage figure-for-figure, and the run's `cost_session` figures equal the sum of every `cost_turn` on the stream including the compaction's.
- **S-CMP-006** — **The iff, arm by arm.** Given three runs — one whose compaction succeeds, one whose compaction provider errors before any completion, and one whose compaction completion returns a non-`Stop` finish reason — when each stream is recorded, then run 1's compaction bracket closes non-aborted and carries exactly one `cost_turn`; run 2's closes **aborted** with a typed failure and carries **no** `cost_turn`; run 3's closes **non-aborted** and carries a `cost_turn`, and emits `compaction_failed` rather than `compaction_finished`; and `CheckStream` accepts all three streams unmodified.
- **S-CMP-032** — **(bite)** RED-first. Given a scratch tree in which the explicit accumulation at the compaction emission site is removed on the assumption that the existing forwarder folds it, when `S-CMP-004` runs, then it FAILS reporting a `cost_session` short by exactly the compaction's usage — proving the fold is a written statement rather than an inherited one. RED-recorded BEFORE `S-CMP-004` is GREEN, then reverted.

---

### R-CMP-004 — The cut is resolved by RETRACTION ONLY, to a recorded turn-mark boundary, then verified pairing-closed — and the ban governs RESOLUTION, never the REQUEST

The request names a **naive cut**: an index into the transcript the strategy received, which is one-to-one with the history read-back (`transcriptFromHistory`, consumed at `harness.go:512`). Resolution MUST proceed as follows:

1. Retract the cut to the **greatest recorded turn-mark boundary ≤ the naive cut** (`R-CMP-012`).
2. Verify that the prefix `[0, cut)` is pairing-closed by read-only correlation over the public read-back, by the Layer 1 identity pair `ai.ToolCall.ID()` / `ai.ToolResult.CallID()` and by nothing else.
3. If a pair straddles the boundary, retract further — to the entry index of the earliest open call — and re-verify.

Resolution MUST NOT expand the cut **forward** under any condition. Forward expansion would compact entries at or after the boundary the strategy designated, which are exactly the entries the charter's protection clause designates protected; the protection clause is MUST-level and wins. This **overturns the proposal's two-directional fixed-point recommendation** (D1) deliberately, and the charter's "the boundary moved to include the whole pair" is satisfied because the pair ends up whole on the **protected** side.

**SCOPE OF THE FORWARD-EXPANSION BAN — normative, and stated because a reviewer will otherwise read a shipped violation.** The sentence above governs **RESOLUTION**: given a *requested* cut, the resolution procedure MUST NOT return a cut greater than the one it was given. It does **NOT** govern the **REQUEST**. Who may name a requested cut, and whether a later requester may name a **larger** index than an earlier one, is outside this requirement entirely.

Concretely, from AG-20 onward there are **two** requesters, and the ban applies identically to both — as a constraint on resolution, not on either requester:

| Requester | May name a cut | Resolution's obligation |
|---|---|---|
| the compaction strategy (or the on-demand caller) | anywhere in the transcript | retract only, from **that** naive cut |
| a **mutating pre-compact hook**, which MAY re-designate the cut **forward** of the already-resolved value (`R-HKS-003`) | anywhere in the transcript | retract only, from **the hook's new request** |

A forward-adjusting pre-compact hook therefore violates **nothing**: it issues a new request, and resolution retracts from that new request exactly as it always retracted from the strategy's. The invariant this requirement protects — *the committed prefix is pairing-closed and mark-aligned* — is preserved by re-running the identical procedure, which is precisely why AG-20 writes **no new validation code** for the charter's *"the adjusted span is revalidated by the invariant-safe surgery before use"*.

**Every requested cut MUST be resolved, and the re-resolution after a hook is UNCONDITIONAL.** An implementation MUST NOT skip the second resolution on the grounds that the first already ran or that the plan looks unchanged: "looks unchanged" is not a property this requirement can check, and skipping it is exactly what splits a pair.

**Idempotence on the unchanged path MUST be proven, not assumed.** For any cut `c` that is itself the output of this procedure, resolving `c` MUST return `c`. This follows from two properties of the procedure over an unmutated transcript — mark retraction is idempotent on its own image, and the open-pair scan is a pure function of the messages and the cut — but "follows from" is not an assertion, and until AG-20 nothing asserted it. `S-CMP-040` asserts it directly and `S-CMP-038` asserts its observable consequence.

Termination MUST be structural rather than empirical: the cut is a non-negative integer that strictly decreases on each iteration and is bounded below by zero. A cut resolving to zero means there is nothing to compact and MUST fail typed rather than commit an empty replacement. **When a re-resolution after a hook adjustment reaches zero, it MUST fail typed through the compaction bracket's existing failure arm carrying the aborted turn outcome**, at the splice's own call site — the upstream empty-cut refusal is not reachable from the post-splice position, so the arm's *shape* is reused while the call site is new.

Resolution MUST be a pure read-only computation over the public read-back. It MUST NOT widen `History`'s exported surface to reach private pairing state. **A pre-compact hook receives a value type that cannot reach the transcript**, so nothing between the first resolution and the commit mutates the history the second resolution reads.

(Previously: the requirement stated *"Resolution MUST NOT expand the cut **forward** under any condition"* with no scope qualifier, at a time when the strategy was the only requester and the distinction between a *requested* cut and a *resolved* cut had no consumer. AG-20 adds a second, deliberately mutating requester, and a reviewer reading a forward-adjusting hook against the unqualified sentence would conclude the change ships a violation of a MUST. The ban is unchanged in strength and is now scoped to the procedure it always described; the unconditional re-resolution obligation and the post-splice empty-cut arm are stated rather than left to inference.)

#### Scenarios

- **S-CMP-010** — **Charter AG-18.2 scenario 1.** Given a transcript in which a tool call at index *i* has its matching result at index *j > i*, and a compaction request whose naive cut falls strictly between *i* and *j*, when the cut is resolved, then the resolved cut is ≤ *i*, both halves of the pair are present in the protected tail after the replacement, and neither half was replaced.
- **S-CMP-011** — **Charter AG-18.2 scenario 1, the validation half.** Given the transcript produced by that same compaction, when a fresh history is constructed over its read-back messages through seeded construction, then construction **succeeds** — the post-compaction transcript passes the same boundary validation an ordinary seed does.
- **S-CMP-012** — **Retraction is monotone, and forward expansion is excluded by test.** Given a naive cut *c* and the resolved cut *r*, when any fixture in this capability's suite compacts, then `r ≤ c` holds for every one of them, and the protected tail after compaction is a **superset** of the entries at indices ≥ *c* before it. *(AG-20 update: "*c*" is the cut **as requested at the resolution being measured** — the strategy's naive cut on a hookless run, and the hook's re-designated cut on a hooked one. The scenario's falsifiable claim is exactly what it was: each resolution retracts from its own input. `S-CMP-041` asserts the hooked instance separately so the two are never conflated.)*
- **S-CMP-013** — **Termination on adversarial input.** Given a transcript with several nested and interleaved open calls whose naive cut splits more than one pair, when the cut is resolved, then resolution completes, the resolved cut is strictly less than the naive cut, and the resolved prefix is pairing-closed; and given a transcript whose every entry belongs to one straddling pair, when resolution reaches zero, then compaction fails typed with `compaction_failed` and the transcript is unchanged.
- **S-CMP-030** — **(bite)** RED-first, **mandated**. Given a scratch tree in which cut resolution skips mark alignment and pairing verification and uses the naive cut directly, when `S-CMP-010` runs, then it FAILS reporting a split pair across the boundary — proving the resolution step is load-bearing rather than incidentally satisfied by a fixture whose naive cut already landed cleanly. RED-recorded BEFORE `S-CMP-010` is GREEN, then reverted.
- **S-CMP-041** — **AG-20: a forward-adjusting hook is resolved, not refused, and the ban still bites on the new request.** Given a transcript whose strategy-requested cut resolves to *r*, and a pre-compact hook re-designating the cut **forward** to *f > r* where *f* lands strictly inside an open call/result pair, when compaction proceeds, then the compaction is **not** refused; the cut actually committed is ≤ *f*; the committed prefix is pairing-closed and mark-aligned; a fresh history seeded from the post-compaction read-back constructs successfully; and given a second hook re-designating the cut forward to an index that resolves to zero, when compaction proceeds, then it fails typed with `compaction_failed` carrying the aborted turn outcome, the transcript is byte-identical to its pre-attempt read-back, and the run continues. Cross-referenced to `R-HKS-003` / `S-HKS-008`.

---

### R-CMP-005 — The replacement is a PREFIX replacement through history's single commit primitive

Compaction MUST replace the resolved prefix `[0, cut)` with **one** summary entry, through the single validating commit primitive of `R-HIS-004`. It MUST NOT reach committed storage by any other route, and it MUST NOT expose a general mid-span replacement: every charter scenario compacts the oldest region and protects the recent tail, so a mid-span route has no consumer and would force a wider relaxation of `R-HIS-001` than the change needs.

The commit MUST be all-or-nothing. On any validation failure the transcript MUST be left byte-identical to its pre-attempt state — `R-HIS-002`'s "a failed commit is not a partial commit", carried forward verbatim to this route.

The summary message MUST be the compaction completion's own assistant message, **unchanged**. Layer 2 MUST NOT wrap, annotate, re-serialise or prefix it. The summary message MUST carry no tool parts: a prefix replacement MUST NOT reopen the open set.

The new exported route MUST be named in `S-HIS-030`'s enumeration in the **same commit** that introduces it, and the exported-surface guard MUST be updated deliberately rather than incidentally.

#### Scenarios

- **S-CMP-007** — Given a compaction whose resolved cut is *r*, when the commit succeeds, then the read-back holds exactly one summary entry at position 0 followed by the entries formerly at indices ≥ *r*, in their original relative order, and the total entry count equals `1 + (len(pre) - r)`.
- **S-CMP-027** — Given a summary message carrying a tool-call part, when the replacement is attempted, then the commit is rejected typed, the transcript is byte-identical to its pre-attempt read-back, and `compaction_failed` is emitted.
- **S-CMP-008** — Given the enumerated set of every public route that can extend or mutate the transcript after this change, when the closed enumeration of `S-HIS-030` runs, then the prefix-replacement route is named in it and drives the same orphaning rejection every other route drives.

---

### R-CMP-006 — Protected entries are preserved value-identical; entry identity is NOT preserved, and asserting otherwise is forbidden

Every entry at or after the resolved cut MUST survive the compaction with:

- its Layer 1 value equal to its pre-compaction value, compared through Layer 1's own value equality; and
- its origin discriminator equal to its pre-compaction origin;

compared **positionally**: pre-compaction entry at index `cut+i` against post-compaction entry at index `1+i`.

Entry identity MUST be **excluded** from that comparison and MUST be asserted to have **changed**. Identities are ordinal-derived (`R-HIS-005`), so a shortened transcript necessarily renumbers them; preserving them would require minting non-ordinal identities, the exact back door `R-HIS-005` exists to close. Renumbering is therefore **forced by the existing rule**, not chosen by this change.

**Whole-entry structural equality (`reflect.DeepEqual` over entry values including identity) MUST NOT be used in any protection assertion in this capability.** On a fixture that happens not to shift ordinals it would pass while encoding a claim `R-HIS-005` makes impossible; on any other fixture it fails spuriously. Either way it is the wrong assertion.

Every fixture asserting protection MUST have a replaced prefix of length **≥ 2**, so protected ordinals provably shift, and at least one protected entry MUST be asserted to carry a **different** identity after compaction than before.

#### Scenarios

- **S-CMP-014** — **Charter AG-18.2 scenario 2, the protection half.** Given a transcript whose replaced prefix holds at least two entries and whose protected tail holds at least two entries, when compaction completes, then for each protected index *i* the post-compaction entry's Layer 1 value equals the pre-compaction value and its origin equals the pre-compaction origin; and the assertion is written over values and origins, not over whole entry structs.
- **S-CMP-015** — **The identity half, asserted rather than assumed.** Given that same fixture, when the pre- and post-compaction identities of a protected entry are compared, then they are **different**, and the post-compaction identities are exactly the ordinal-derived identities of the new sequence — so no reader can take "byte-identical" from the charter to mean identity survived.

---

### R-CMP-007 — The summary entry is discriminated by envelope origin alone

The summary entry MUST be distinguishable from a model-authored message by a **third origin discriminator on the entry envelope**, beside the existing appended and synthesized members, mintable **only** by the prefix-replacement commit op.

Distinguishability MUST NOT depend on message content and MUST NOT depend on any failure flag. A content sentinel is **prohibited**, on `R-HIS-007`'s own already-shipped argument (`agent-history/spec.md:139`: "a real tool can emit any content bytes, so a content sentinel is forgeable and is prohibited"), which does not weaken for compaction: the summary's content is model-authored bytes, so any sentinel inside it is exactly as forgeable. No test in this capability MAY distinguish the summary by reading content.

#### Scenarios

- **S-CMP-016** — **Charter AG-18.2 scenario 2, the typing half.** Given a completed compaction and a control history in which a model assistant message with **byte-identical content** was appended ordinarily, when each entry's origin is read, then the summary entry reports the summarized origin, the control entry reports the appended origin, the two are distinguished correctly, and the assertion reads neither message content nor any failure flag.
- **S-CMP-028** — Given the public surface of the entry envelope after this change, when it is enumerated from an external test package, then no exported route exists by which a caller can set, supply or overwrite an origin discriminator, including the summarized one.

---

### R-CMP-008 — Compaction runs inside its own dedicated turn bracket, and `stream_check.go` stays byte-unchanged

One compaction operation MUST be recorded as **one dedicated turn bracket** carrying its own fresh `TurnID`, minted in a namespace that cannot collide with the run loop's turn identifiers. Within that bracket the event order MUST be:

`turn_start` → `compaction_started` → (`cost_turn` iff a completion exists, per `R-CMP-003`) → `compaction_finished` | `compaction_failed` → `turn_end`.

The bracket is required rather than stylistic: the compaction family is registered with turn placement, and `stream_check.go:161` reads `if d.Placement == PlacementTurn && !turnOpen { return violation(...) }` — verified verbatim in this worktree this phase. Emitting the family in the no-turn-open gap without a bracket would be rejected by the validator, and the validator is frozen. The bracket MUST be opened and closed through the **existing** exported turn-event constructors; **no** new `EventKind`, `TurnOutcome` or `CostLabel` MUST be registered, and `stream_check.go`, `event.go`, `event_descriptor.go`, `compaction_events.go` and `event_registry_test.go` MUST be byte-unchanged.

`CheckStream` MUST accept every recorded stream carrying a compaction bracket **unmodified**. Any fix MUST land on the emitter, never on the validator (`R-RUN-003`).

#### Scenarios

- **S-CMP-017** — **Charter AG-18.3 scenario, the placement half.** Given a run in which a strategy-requested compaction completes, when the recorded stream is validated, then `CheckStream` accepts it unmodified with `stream_check.go` byte-unchanged; the compaction bracket's events appear in the order this requirement states; the bracket's `TurnID` is distinct from every model turn's `TurnID` on the same stream; and the every-kind-constructible guard passes at its committed kind count with AG-18 registering none.

---

### R-CMP-009 — The finished event carries what a resumed session needs, and nothing has to be inferred

`compaction_finished` MUST carry a valid span naming the first and last turn identifiers of the replaced region, together with the identity of the summary message it produced. Those two facts MUST be sufficient, **without reading history**, for a consumer holding only the recorded stream to name exactly which turn brackets earlier on that stream were replaced and to locate the summary entry in the post-compaction transcript by message identity.

`compaction_failed` MUST carry a typed failure. `compaction_finished` and `compaction_failed` MUST NOT both appear for one compaction operation.

#### Scenarios

- **S-CMP-018** — **Charter AG-18.3 scenario, the reconstruction half.** Given a recorded run carrying at least two model turn brackets before a compaction bracket, when a reconstruction is driven **only** from the stream's `compaction_finished` payload, then the turn identifiers it names are exactly the identifiers of the turn brackets that appear earlier on that same stream and whose entries were replaced; and when the post-compaction history read-back is searched by the summary message identity that payload carries, then exactly one entry matches and it is the summary entry.
- **S-CMP-019** — Given every stream recorded by this capability's suite, when the compaction-family events are counted per operation, then no operation produced both a finished and a failed event, and every started event is followed by exactly one of the two inside the same bracket.

---

### R-CMP-010 — Atomic-or-absent by ordering, and a failed compaction never winds the run down

The commit MUST be **unreachable** until all of the following have succeeded, in this order: the cut is resolved and mark-aligned (`R-CMP-004`); the injected provider returned a completion whose finish reason is `Stop`; and the complete replacement message sequence passed, on a scratch value, the same pairing rules seeded construction enforces.

Because no statement writes the caller's transcript before that single commit, there MUST be **no partial state to roll back** and the implementation MUST NOT introduce a journal, a snapshot, an undo log or a rollback mechanism. Atomicity is a property of the ordering; the implementation MUST assert it rather than build a mechanism for it.

On any failure the system MUST emit `compaction_failed` and MUST NOT emit `compaction_finished`; the transcript MUST read back byte-identical to its pre-attempt state; and the run MUST continue at its next turn boundary against the **uncompacted** transcript.

A compaction failure MUST NOT route through the run's cancellation wind-down. `R-CAN-002`'s wind-down order is closed and validator-checked; entering it from inside compaction would break it, and a run that wound down cannot satisfy "the next turn proceeds".

A run-level cancellation arriving during compaction MUST close the compaction bracket aborted and MUST leave the run to wind down at its **existing** iteration-boundary cause check, exactly as it would have without compaction.

#### Scenarios

- **S-CMP-020** — **Charter AG-18.4 scenario.** Given a run whose compaction provider is gated and then made to fail, when the run continues, then the history read-back after the attempt is byte-identical to the read-back captured before it; `compaction_failed` appears on the stream and `compaction_finished` does not; the next model turn's request carries the **uncompacted** transcript, proved from the provider's captured requests; and the run reaches its ordinary close.
- **S-CMP-021** — **The wind-down is never entered.** Given that same failed compaction, when the recorded stream is inspected, then it carries no orphan-synthesis-driven wind-down signature and the run's close is the ordinary one, not the cancellation order of `R-CAN-002`; and the run performed at least one further turn after the failure.
- **S-CMP-031** — **(bite)** RED-first, **mandated**. Given a scratch tree in which the commit is moved to before the provider call returns, when `S-CMP-020` runs, then it FAILS reporting a mutated transcript after a failed compaction — proving atomicity comes from ordering and that the assertion detects its loss. RED-recorded BEFORE `S-CMP-020` is GREEN, then reverted.

---

### R-CMP-011 — The on-demand door emits its own minimal run bracket and refuses typed while a run is in flight

The system MUST expose an on-demand harness entry point invoking the **same** operation. It MUST emit its own minimal, independently valid stream: a run bracket opening, the compaction bracket of `R-CMP-008`, the run's cumulative cost event, and a run close. A bare compaction bracket with no run bracket MUST NOT be emitted, because the validator rejects any stream lacking a complete run bracket.

The on-demand door MUST refuse **whenever a run is in flight**, detected through the harness's existing signal mutex and in-flight run handle — the same state `Interrupt` and `Shutdown` already read (`harness.go:143-146`, `:157-161`, set at `:435`, cleared at `:450`; re-verified this phase). The refusal MUST be a typed sentinel error, distinguishable by errors-chain identity in the house pattern, returned synchronously, emitting **no** event on the caller's sink.

**This overturns the proposal's D4 recommendation of a new turn-in-flight flag, and the reason is evidence rather than preference.** `History` carries **no mutex** — verified in this worktree: `history.go`'s commit path is plain field mutation under no lock — and the run loop reads and writes the same history between turns (`harness.go:506-512`, `:668`). A concurrent on-demand mutation is therefore a data race **whether or not a turn is open**, so "between turns of a live run" is not a safe window and a turn-granular flag would gate the wrong predicate. Run-in-flight **subsumes** turn-in-flight: the charter's "Given a turn in flight" fixture is refused because its run is, and with no run in flight no turn is open anywhere — the strongest available form of "compaction happens only at turn boundaries".

The request MUST NOT be queued **in any form**, including a "queued, and the caller is told" variant. The charter's word *silently* MUST NOT be read to permit queueing that announces itself.

After a terminal shutdown the on-demand door MUST refuse typed under the existing terminal-shutdown semantics, on the same route as a post-shutdown prompt.

Equivalence between the two doors is scoped to the **compaction turn bracket sub-sequence**, `Kind()`-for-`Kind()`, excluding the fresh run and turn identifiers. Whole-stream equality MUST NOT be asserted: the strategy-triggered stream necessarily also carries its prompt turns, so a whole-stream comparison is either impossible or rigged.

#### Scenarios

- **S-CMP-023** — Given an on-demand compaction over a marked history with no run in flight, when it completes, then the stream it emitted is accepted by `CheckStream` unmodified, carries exactly one run bracket enclosing exactly one compaction bracket, and carries the run's cumulative cost event before the run close.
- **S-CMP-024** — **Charter AG-18.5 scenario 2.** Given a run held mid-turn by a test gate, when the on-demand door is invoked concurrently, then it returns synchronously with the typed refusal, the refusal is identifiable through the errors chain, **zero** events were delivered to the caller's sink, the in-flight turn completes unaffected, and the run's own recorded stream is byte-identical to the same run driven with no concurrent demand.
- **S-CMP-025** — Given a harness that has been shut down terminally, when the on-demand door is invoked, then it refuses typed on the existing terminal-shutdown route, emits no event, and mutates no transcript.
- **S-CMP-033** — **(bite)** RED-first, **mandated**. Given a scratch tree in which the on-demand door enqueues the demand for the next turn boundary instead of refusing, when `S-CMP-024` runs, then it FAILS — either on the missing typed refusal or on the later compaction the run then performs — proving the refusal is a refusal rather than a deferral. RED-recorded BEFORE `S-CMP-024` is GREEN, then reverted.

---

### R-CMP-012 — Turn attribution is recorded by history at turn close, and an unattributable span fails typed

The compaction span names **turn** identifiers, and the compactable region crosses **prior run invocations** — the harness uses the caller's history in place and never replaces it (pinned at `harness_test.go:1013`), and a caller may reuse one history across runs. A run-scoped structure therefore cannot attribute earlier runs' entries to their turns, and the transcript records no turn structure today.

The system MUST therefore record a **turn mark** — a turn identifier together with the entry count at that turn's close — in the transcript store itself, written through the same single commit primitive of `R-HIS-004`. The mark MUST be supplied by the harness from the successful attempt's own forwarded turn bracket, through a **package-private** marked-close door. The **exported** turn-close route MUST keep its exact signature and its exact semantics, so no existing caller changes behavior and no unmarked close becomes illegal.

Span derivation MUST read the marks fully contained in the resolved prefix: the first names the span's start turn and the last names its end turn. Because `R-CMP-004` always lands the cut on a mark boundary, the span is well defined in whole turns.

**Named v1 limitation, recorded here rather than discovered at verify**: a resolved prefix containing **unmarked** entries — for example a seeded prefix that no run ever drove — is unattributable. The system MUST fail such a compaction typed, emitting `compaction_failed`, and MUST NOT fabricate, synthesize or reuse a turn identifier to make the span constructible. Widening attribution to seeded prefixes is **not** in this milestone.

#### Scenarios

- **S-CMP-029** — Given a history driven through two complete runs, when a compaction resolves a cut at the boundary between them, then the span the finished event carries names the turn identifier of the first marked turn in the prefix and the turn identifier of the last, and both identifiers appear as turn brackets earlier on the recorded stream.
- **S-CMP-034** — **The limitation is asserted, not assumed.** Given a history constructed from a seed that no run drove, when a compaction over a prefix of that seed is requested, then it fails typed, `compaction_failed` is emitted, no turn identifier is invented, and the transcript is unchanged.
- **S-CMP-035** — Given the exported history surface after this change, when it is enumerated from an external test package, then the exported turn-close route carries its pre-AG-18 signature and semantics, an unmarked close still succeeds, and the marked-close door is **not** reachable from outside the package.

---

### R-CMP-013 — Named consequences and the no-new-vocabulary fence

Three consequences MUST be stated in this spec rather than discovered by a later consumer:

1. **Discarding is irreversible.** A compaction that has run has discarded transcript entries that exist nowhere else in the process. This change ships **no** journal, no undo and no pre-compaction snapshot. That is acceptable for v1 only because nothing persists across processes and no Layer 3 consumer exists; it is a named consequence, not a migration concern.
2. **`EntryID` MUST NOT be stored across a compaction by any consumer.** The durable handle is the turn identifier (`R-CMP-006`, `R-HIS-005` as amended).
3. **A compaction failure is visible on the stream and nowhere else.** The run continues and the caller's return value is unchanged (`L2C-03`: the stream is the only upward contract). A synchronous failure signal is a different requirement and is not this one.

AG-18 MUST register **no** new `EventKind`, add **no** new `TurnOutcome` and add **no** new `CostLabel`. `event.go`, `event_descriptor.go`, `event_registry_test.go`, `stream_check.go`, `compaction_events.go`, `cost_events.go`, `cost_usage.go`, `reconstruction_test.go`, `go.mod`, `go.sum` and every file under `backend/agent/src/ai/` MUST be byte-unchanged.

#### Scenarios

- **S-CMP-036** — Given the merge base of the AG-18 branch with `origin/main`, when `git diff` is taken over `backend/agent/`, then every file named byte-unchanged above is byte-unchanged, the diff under `backend/agent/src/ai/` is empty, the `go.mod`/`go.sum` diff is empty, and the every-kind-constructible guard passes at its committed kind count with AG-18 registering none.

---

### R-CMP-014 — Compaction is inert unless requested, and no run-level closed sequence moves

A run whose strategy requests no compaction MUST produce a stream and a transcript **identical** to the same run with no strategy installed. Compaction is reachable only from a non-zero verdict or an explicit on-demand call, so no existing enumerated run-level sequence can move.

**The reasoning is recorded here rather than left to be rediscovered**, because a new bracket inside an existing closed sequence is this repository's known spec-breakage shape. Verified against this tree: the enumerated expected-kind blocks in the run driver's own tests run with a **nil** strategy, and the run driver's contiguity assertion pins contiguity rather than a fixed length, so an insertion that only occurs on a requesting path cannot falsify either. `S-LSK-001`'s length-equality sequence is likewise not at risk: it asserts a direct one-turn call with no harness.

Any milestone that later makes compaction reachable on a previously non-requesting path MUST re-check the enumerated sequences before doing so.

#### Scenarios

- **S-CMP-037** — Given two runs of an identical multi-turn script — one with no strategy installed and one with a strategy that requests no compaction on every consultation — when both are recorded, then the two event streams are byte-identical, the two history read-backs are byte-identical, and neither carries any compaction-family kind.

---

### R-CMP-015 — AG-20 splices ONE pre-compact chain inside the ONE operation, and the three unamended requirements are re-verified for a hook-carrying path

**AG-20 adds exactly one splice to the compaction path, and its placement is chosen so that three shipped requirements need no amendment.** The chain is invoked inside the compaction operation, after the cut has been resolved and the span derived, and strictly before the compaction provider call. Four consequences MUST be stated rather than left to be inferred, because each is a separate thing a reader would otherwise have to re-derive:

1. **`R-CMP-001`'s one-operation, two-door shape is what makes one splice sufficient.** Both doors funnel through the one operation, so the single splice reaches both. A **second** splice, at either door or at the harness's verdict gap, is forbidden: it would be outside the compaction bracket, would have no typed arm to land a hook failure on, and could hand the hook only an index rather than a real span.
2. **`R-CMP-003`'s iff holds because the splice is PRE-PROVIDER.** A chain failure occurs before any completion is obtained, so it takes the "no completion" arm — aborted, no per-turn cost event — which is a case `S-CMP-006` already discriminates. No arm is added and no arm's condition is widened.
3. **`R-CMP-010` and `R-CMP-013` hold because the failure arm is REUSED, not replaced.** A chain failure and a post-splice empty-cut failure both emit the existing compaction-failed event and both leave the run running. The stream stays `CheckStream`-valid with the validator byte-unchanged.
4. **`R-CMP-014`'s fence holds: AG-20 registers no `EventKind`, adds no turn outcome and no cost label**, and `compaction_events.go`, `event.go`, `event_descriptor.go`, `stream_check.go` and `event_registry_test.go` are all byte-unchanged.

**The misplaced-options rejection gains the hook field, and this is enforcement rather than documentation.** The compaction request's options MUST reject a caller-supplied `Hooks` value alongside the fields it already rejects (`compaction.go:269-272`), so a pre-compact chain can reach the compaction operation **only** as an explicit unexported parameter from the harness and never smuggled through the request's own options. There is one registration surface (`R-HKS-001`), and this is where that claim is enforced on the compaction path.

**Inertness is by guard, not by convention.** When the chain is empty, no invocation occurs, no plan value is constructed, and the compaction path is byte-for-byte the path it is today on **both** doors and on **both** the success and failure arms.

#### Scenarios

- **S-CMP-038** — **AG-20: an identical-plan hook is byte-identical to no hook — the observational half of idempotence.** Given two runs of an identical compacting script — one with no hooks, one with a `PreCompact` chain of one element returning its input plan unchanged — when both are recorded, then the two event streams are byte-identical modulo the freshly minted run and turn identifiers, the two committed history read-backs are byte-identical, the two captured compaction provider requests are byte-identical, and both streams are `CheckStream`-valid; and given a third run with an **empty** chain, when it is recorded, then it too is byte-identical to the hookless run and no plan value was constructed.
- **S-CMP-039** — **AG-20: a chain failure lands on the existing arm, on both doors, and the run continues.** Given a `PreCompact` chain whose second element returns a non-nil error, when the compaction is reached through the strategy-verdict door, then the compaction provider recorded **zero** requests, the bracket closes with the aborted turn outcome carrying a typed failure, exactly one compaction-failed event is emitted and no compaction-finished event is, no per-turn cost event appears inside that bracket, the committed transcript is byte-identical to its pre-attempt read-back, the run continues to its own terminal, and `CheckStream` accepts the whole stream unmodified; and when the same fixture is driven through the **on-demand** door, then every one of those assertions holds identically. Cross-referenced to `R-CMP-003` / `S-CMP-006` and `R-CMP-010`.
- **S-CMP-040** — **AG-20: cut resolution is a fixed point, asserted directly.** Given the module's internal fixed-point table over the same class of naive cuts the surgery suite already uses — zero, on a mark, mid-mark, straddling an open pair, and beyond the transcript length — when resolution is applied to its own output, then it returns that output unchanged for **every** row; and the table is exercised under `NFR-CMP-001`'s pure-helper carve-out, whose own condition is satisfied because the observable consequence is asserted externally by `S-CMP-038`.
- **S-CMP-042** — **AG-20: the misplaced-options rejection is total for the hook field.** Given a compaction request whose options carry a non-zero hook value in **any one** of its members, asserted member by member, when the compaction operation is invoked, then it is rejected typed exactly as it already rejects the other misplaced option fields, no provider call occurs, and the transcript is unchanged; and given a compaction whose chain arrives as the operation's own parameter, when it runs, then the chain fires normally. Cross-referenced to `R-HKS-001` / `S-HKS-002`.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-CMP-001** | **External-package verifiability.** Every behavioral scenario MUST be verifiable from `package agent_test`. A behavior reachable only from inside the package is, for this spec, not reachable at all — the same rule `NFR-HIS-001` states. Pure helpers with no external surface MAY be tested internally, provided every claim about what a **caller** observes is also asserted externally. |
| **NFR-CMP-002** | **Determinism and race cleanliness.** Every test MUST be hermetic and MUST pass under `-race`. Synchronization MUST be by the package's test gate primitive, channel reads and channel closes; **no test may synchronize by sleep, timeout or wall-clock ordering**. Evidence MUST be recorded from a run with **`-count=1`**, and the wall-clock duration MUST be recorded with it; a sub-second suite for this module is a cache artifact, not evidence. |
| **NFR-CMP-003** | **Ambient authority and boundaries.** Production sources added by this change MUST NOT import process, filesystem, environment or network facilities; the ambient-authority and import-boundary guards MUST pass with zero change. |
| **NFR-CMP-004** | **Substrate.** Every file named by `R-LSK-004` MUST be byte-unchanged except the two released by this change's [`../agent-loop-skeleton/spec.md`](../agent-loop-skeleton/spec.md) delta — `doc.go` and `doc_contract_guard_test.go` — whose edits MUST be confined to the `L2C-07` row's two clauses and MUST be byte-in-sync with each other. Both substrate filters MUST widen by **exact filename suffix only**, with no wildcard, no prefix match and no directory-level relaxation, and MUST carry an identical entry set. |
| **NFR-CMP-005** | **Review budget.** This change ships as a **single** pull request under a pre-accepted `size:exception` against a 1000-line budget counted **excluding every path under `openspec/`**. The pull-request description MUST state why the change does not fit the default budget. If `sdd-tasks` forecasts beyond ~3000 counted lines, the reserve slicing boundary is the charter's own DAG: cut resolution plus the replacement route first (no provider call, no events), then the call and its atomicity, then the stream record and the on-demand door. |

## Explicit non-requirements — what this spec does NOT claim

| Not claimed | Owner and why the deferral is safe |
|---|---|
| **When** to compact — any threshold arithmetic over budget against accounting | **Layer 3** (`0003:1665`). AG-18 acts on a request; it never decides one |
| The summarization instruction's **content** | **Layer 3, injected** (`0003:1662`). `R-CMP-002` makes Layer 2 authorship an assertable prohibition, not a style note |
| Persistence of the compaction record across processes | **Layer 3 session** (`0003:1665`) |
| A real tokenizer or exact boundary sizing | **Out of scope and structurally impossible.** `L2C-01` allows the standard library and Layer 1's measured closure only, and `openspec/config.yaml:33` forbids a new top-level dependency without an ADR. AG-17's estimate stays a documented heuristic; AG-18 consumes it |
| Aggregating compaction spend into a parent or delegated run | was: *"**AG-19.** The run's own cumulative is in scope; nothing above it is"* — **ANSWERED by AG-19, and the answer is a permanent NO at Layer 2 rather than a hand-off.** No Layer-2 fold of any child figure into any parent figure occurs on any path: AG-19's publishing seam **refuses the `cost_turn` kind in production code**, so a child's compaction spend — like every other child `cost_turn` — cannot reach a parent's cumulative even by accident. A combined figure, if a frontend wants one, is a **consumer-side reconstruction** over two independently validated streams joined by walking parents. **This row therefore closes as "never at Layer 2", not as "AG-19 will do it"**, and no later milestone may reopen the fold without amending `R-CST-004` first. *(Unchanged and still true: this capability's own scope is the run's own cumulative, and nothing above it. What changed is that "above it" now has a settled answer instead of a pending owner. See `agent-cost-events`' `R-CST-007` as amended by this same change.)* **A second AG-19 consequence is recorded here rather than left to be rediscovered**: AG-19's admissibility rule **admits all three compaction kinds** onto a parent's stream — the first time compaction events can appear anywhere other than the run that performed the compaction. It is safe, and by mechanism: `R-CMP-008`'s bracket order holds because the seam refuses every bracket-role kind, so a child's compaction `turn_start`/`turn_end` cannot cross and only the three turn-placed, non-terminal, cardinality-*any* compaction kinds are admissible inside the parent's open turn, leaving `stream_check.go` byte-unchanged; `R-CMP-009`'s exactly-one-of-finished-or-failed is per **operation**, and a mirrored pair belongs to the child's operation, discriminated by `Event.Run()` — the same discriminator AG-19 mandates for cost; `R-CMP-003`, `R-CMP-001`, `R-CMP-011`, `R-CMP-014` and `R-CMP-013` are all untouched. The three kinds are admitted rather than special-cased because the rule is **derived from the registry**, and a child's compaction is legitimately visible to the human watching the parent's stream — it explains a gap in the child's transcript that would otherwise look like lost context |
| A compaction-only cancellation signal or deadline | **Not built.** The charter says compaction cancels via the run's existing tree — participating in it, not branching it |
| A general mid-span replacement route | **Not this milestone** (`R-CMP-005`). No charter scenario needs one, and it would force a wider relaxation than the change requires |
| Attribution of a seeded, never-run prefix | **Not this milestone** — the named v1 limitation of `R-CMP-012`, which fails typed rather than fabricating identifiers |
| A journal, undo or pre-compaction snapshot | **Not this milestone** — `R-CMP-013` records the irreversibility as a consequence rather than papering over it |
| A synchronous failure signal to the caller beyond the stream | **Not this milestone** (`L2C-03`) |
| Any new `EventKind`, `CostLabel` or `TurnOutcome` | **Never in AG-18** (`R-CMP-013`). AG-06 minted the compaction family; AG-18 is its first caller |
| Any edit under `backend/agent/src/ai/**` | **Never in Layer 2.** Layer 1 is consumed, never edited |
| Re-opening the cancellation wind-down order | **CLOSED by AG-14 and untouched** (`R-CMP-010`) |

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active. All five leaves are behavior, so all five are RED-first. The four bites — `S-CMP-030` (skip cut resolution), `S-CMP-031` (commit before the call returns), `S-CMP-032` (drop the explicit cost fold), `S-CMP-033` (queue instead of refuse) — MUST each be RED-recorded **before** its corresponding scenario is GREEN, with `-count=1`, and then reverted.
