# Design — the first provider and its transport

> **Change**: `cachicamas-ai-first-provider-decision`
> **Milestone**: AI-24 · **Nodes**: AI-24.1 + AI-24.2, both `[decision]`
> **Phase**: design (revised — one corrective pass against the parallel spec)
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-03
> **Inputs**: Engram `sdd/cachicamas-ai-first-provider-decision/explore`, `proposal.md`, `specs/ai-first-provider-decision/spec.md` (`R-APD-001 … 019`, `S-APD-001 … 064`)
> **Output**: the structure `decision.md` implements, the evidence rules it applies, and the doc 0002 amendment mechanics
> **Boundary**: requirements and scenarios are the spec phase's. This design owns artifact structure, amendment mechanics, and evidence organization only. No Go identifier appears here or in anything this design specifies.

---

## 1. What is being designed

Two deliverables with different failure modes:

| Deliverable | Design problem | What breaks if it is wrong |
| --- | --- | --- |
| `decision.md` | One artifact closing **two** checklists (AI-24.1's four items, AI-24.2's three) so a reviewer verifies each item without hunting | An item is answered "somewhere" and AI-25 … AI-32 inherit ambiguity |
| doc 0002 amendments | Nine dated entries (A0a–A8) under the revert-and-record clause, plus the AI-41 append | The AI-25.2 guard ships unable to bite; nodes lose subjects silently; the graph stops being the single true map |

The precedent is AI-03's `decision.md` (archive, `2026-07-31-cachicamas-ai-minimum-capabilities`): checklist-ordered spine, method sections before application sections, an inheritance table, and a closing-checklist verification table (§ 14). This design reuses that shape for two checklists instead of one.

## 2. Structure of `decision.md`

```
  §1  How to use this document        ← per-audience entry points
  §2  What was decided                ← both verdicts, one table, before any argument
  §3  Evidence rules                  ← the method, stated once (see § 3 below)
  ── AI-24.1 spine, checklist order ──
  §4  The seven-axis comparison       ← item 1: chosen vs Anthropic-native vs the SDK class
  §5  What the choice costs           ← item 1's fairness half: the two priced losses
  §6  Four divergences answered       ← item 2
  §7  Two further questions           ← item 3
  §8  The expected capability report  ← item 4: total, eight entries, floor clause first
  ── AI-24.2 spine, checklist order ──
  §9  Transport: the no-op ADR gate   ← item 1: command-verifiable zero requires, gate transferred
  §10 The streaming framing contract  ← item 2: spec-mandated vs dialect-only, separated
  §11 The credential boundary         ← item 3: enforceable subject for AI-25.2
  ── closing ──
  §12 What each blocked milestone inherits (AI-25 … AI-32, one row each)
  §13 Standing rules and the carryover assignment
       §13.1 the usage opt-in trap · §13.2 the AI-29.0 reservation
       §13.3 the Wave-2 carryovers → AI-41 (R-APD-017's home in the artifact)
  §14 Closing-checklist verification  ← seven rows, both nodes (see § 5 below)
```

Placement arguments, each earning its position:

- **§ 5 sits immediately after § 4**, not in an appendix. The losses (AI-07's round-trip token and AI-11's cap enforcement each lose their first real consumer) are the fairness evidence for the rejection of Anthropic-native; separating them from the comparison would make the comparison read as free.
- **§ 8 opens with the floor clause** — a candidate failing any required capability cannot be the first adapter, cited to the capability contract's floor statement (`S-APD-023`) — before the eight rows, so the report reads as a floor confirmation and not merely a table.
- **§ 9 states the dependency fact as a command-verifiable claim** — the module manifest's zero-requires state, checkable by command against the file, not only in prose (`S-APD-032`) — records the ADR gate as **evaluated and resolved to a no-op**, names the milestone the gate next binds (**AI-37**, the OTel API addition) so the gate is transferred rather than closed (`S-APD-036`), and records the routing of the forward guard's stale code comment to AI-25.
- **§ 10 is two labelled subsections** — § 10.1 *Spec-mandated* (every claim cited to WHATWG HTML Living Standard § 9.2: **the response content type**, field grammar, first-colon split, one-space strip, comment lines, multi-line `data:` LF joining, dispatch-time trailing-LF removal, **the disposition of the identifier and reconnection-time fields** — stated per spec even though the dialect emits neither, because the decoder is specified independently of one vendor's habits (`S-APD-039`) — BOM, and the three line terminators) and § 10.2 *Dialect-only conventions* (`data: [DONE]` terminal sentinel; data-only frames with no `event:` lines). The separation is structural, not editorial: a dialect convention **cannot appear** under § 10.1's heading, and § 10.2 opens with the sentence "No specification states anything in this subsection." Each § 10.2 entry is stated as an explicit AI-27 fixture-pin obligation. This is the mechanism that makes AI-27 inferring `[DONE]` from the SSE spec impossible without visibly contradicting the artifact.
- **§ 13.1 and § 13.2 hold the trap and the reservation** because both are standing rules, not answers to checklist items: the dialect emits in-stream usage only when the request opts in (`stream_options.include_usage` set true) — an unset option disguises an adapter bug as a capability absence, recorded as an obligation on AI-26.7 / AI-28.3 / AI-31.2 and **named again in § 8's `CAP-R-03` row itself** (`S-APD-024`); and AI-29.0's authority is preserved — § 7's no-signed-reasoning answer *indicates* absence but the emit-versus-absence decision remains AI-29.0's, made against the exact backend (some OpenAI-compatible servers emit a non-standard reasoning extension field).
- **§ 13.3 assigns the Wave-2 carryovers in the artifact itself**, because `R-APD-017` places the assignment obligation on `decision.md`, not on doc 0002 (`S-APD-056`). It names both carryovers (the emission-boundary checker's fourth-rule failure-path coverage gap; the missing redacting debug-rendering method on the provider-failure payload), names **AI-41** as owner with its two leaves and its `Blocks: AI-36` edge with the scope-overlap reason, and states the two Wave-5 scheduling reasons: **neither carryover blocks any Wave-4 node**, and **Wave 4 already forecasts 20,000–25,000 changed lines against a 5,000-line review budget**. It cross-references amendment A6, which lands the append in doc 0002; the artifact section and the amendment are two views of one assignment, and the artifact's is the normative one for the spec.

### 2.1 Entry shapes

Uniform shapes so a reviewer checks mechanically, per the AI-03 precedent:

| Section | Row shape |
| --- | --- |
| § 4 | axis · chosen dialect · Anthropic-native · vendor SDK — where "vendor SDK" is **one deliberate candidate class covering either vendor's Go SDK, stated as such in the artifact** so the axis-by-candidate product stays total and checkable (`S-APD-004`). Seven rows; **every cell rests on a cited vendor document, a cited spec clause, or a stated in-repo mechanical fact** (`S-APD-007`), never a bare assertion; verdicts grounded per row, never asserted globally |
| § 6, § 7 | question · answer · consequence for the named downstream node (AI-26.2, AI-26.5, AI-26.7, AI-30, AI-29.0) |
| § 8 | capability id · standing (copied from AI-03, never derived) · expected outcome (`satisfied` / `absent` only) · basis · **confirmation** — eight rows, one per entry across both closed lists. The basis cell names any adapter-side request option the clearance depends on: `CAP-R-03`'s basis names the usage opt-in option directly (`S-APD-024`). The confirmation cell marks an entry **pending** where its basis is not yet confirmed against the exact backend (`S-APD-027`): `CAP-O-01` is marked *pending AI-29.0's confirmation*, not settled |
| § 12 | milestone · what it inherits, in that milestone's own terms · the decision section it cites |

## 3. Evidence rules the artifact applies

1. **The source-label rule.** Every framing claim **anywhere in the artifact** — not only inside § 10; a framing claim restated in § 2, § 12 or § 14 carries the same label — is labelled spec-mandated (with the WHATWG § 9.2 citation) or dialect-conventional (with "fixture-pinned at AI-27"). A claim with no label is a defect (`S-APD-040`).
2. **The grounding rule.** Every § 4 axis entry rests on a cited vendor document, a cited spec clause, or a stated in-repo mechanical fact (`S-APD-007`). An empty cell is a defect, never an implied equivalence.
3. **The fairness rule.** Every rejected alternative's row records what is genuinely lost, not only why it lost. § 5 names the two orphaned contract paths plainly and states that both remain contract-mandatory (the neutral shape is contract regardless of who emits it).
4. **The totality rule.** § 8 is total over both closed lists; expectations are drawn only from `satisfied` and `absent` (AI-03 § 10/§ 12: `failed` and `not exercised` are run results, never predictions); the floor clause opens the section; unconfirmed bases are marked *pending*. This is what makes AI-38.2's generated report comparable entry by entry.
5. **The deletion test.** If deleting a sentence would give a later milestone more options and that milestone is not AI-24, the sentence is cut. Sharpest against AI-29.0: the artifact records the indication and explicitly does not decide.

## 4. The doc 0002 amendment plan — mechanics

One dated convention throughout (revert-and-record clause rule 4): `> **Amended 2026-08-03 (AI-24)** …` blockquote under the touched heading, strikethrough (`~~…~~`) **reserved for superseded claims only**, never a silent edit, all landing in this PR. A never-firing conditional is *not* a superseded claim — its text remains in force for a future adapter — so the three no-subject branches (A2, A3, A4) receive identical treatment: dated blockquote recording the not-applicable / deliberate-no-op standing, **no strikethrough** (`S-APD-052`, doc 0002 rule 4). Written at apply time; specified here exactly.

| # | Target heading | Struck text | Replacement / blockquote content |
| --- | --- | --- | --- |
| A0a | `## Rules for every future SDD milestone` (the `go.mod` bullet) | `~~until AI-24 selects a transport (its own ADR gate) and~~` | Bullet now reads "…zero requires from AI-00 until AI-37 adds the OTel API…". Blockquote: AI-24 selected raw `net/http` — standard library, zero requires; the ADR gate resolved to a **no-op**; the AI-00.3 allowlist gained no second entry |
| A0b | `### AI-24 — Select first provider and transport` (charter) | none (additive) | Blockquote records both verdicts, that the acceptance clause's ADR condition did not fire, and links `decision.md` |
| A1 | `#### AI-25.2 — No ambient authority` | `~~an AST import-and-call scan in the AI-00.3 style~~` | Item 1 now names **a call-site scan over the adapter package's own source files** — the spec's own wording (`S-APD-051`), naming no analysis package (`R-APD-019` permits only the transport's stdlib path, the module manifest, and vendor wire field names). Blockquote states the defect: AI-00.3 is an import-path scan (`go list -deps -test` against a deny-by-default allowlist); `net/http` transitively imports `os`, so the import mechanism would false-positive on legitimate use or miss a narrow environment read (`S-APD-050`). **Flagged load-bearing** — first in the amendment block |
| A2 | `#### AI-26.5 — Tool results and identifiers` | **none** — item 2's IF-guard simply never fires for this adapter; its text is not superseded | Blockquote: no subject for this adapter — the vendor assigns real identifiers on the call's opening delta; the branch is marked **not-applicable to this adapter**; **the requirement text remains in force for a future adapter** that needs minting; this adapter's conformance exercises only the vendor-assigns branch, never a silent skip |
| A3 | `#### AI-26.2 — System segments and cache markers` | **none** — item 3's WHEN-guard never fires; its text is not superseded | Blockquote: caching is automatic, so no vendor cap exists for this adapter; item 2's "dropped whole" branch is the one taken, exercising AI-11.3's advisory contract; **item 3 remains in force for a future explicit-caching adapter** |
| A4 | `#### AI-26.7 — Options, limits and the escape hatch` | none — item 2's IF-guard never fires | Blockquote: the mandatory-output-limit branch is a **deliberate no-op** for this vendor (`max_tokens` optional), recorded per the node's own instruction; the text remains in force for a future adapter |
| A5 | `#### AI-29.0 — Reasoning emission policy` | none (additive note) | Blockquote: AI-24 answers "no signed reasoning blocks; opaque token count only", which **strongly indicates absence** — but the decision remains AI-29.0's, made against the exact backend chosen for AI-38/AI-39, because some OpenAI-compatible servers emit a non-standard reasoning extension field. Explicitly labelled **a note, not a verdict** (`S-APD-030`). The node is not pre-empted and not deleted |
| A6 | New `### AI-41 — Discharge the Wave-2 carryovers`, appended at the end of the Wave 5 section (after AI-37, before `## Wave 6`) | none (append-only, next free milestone ordinal) | Charter: goal = discharge the two Wave-2 carryovers; `Depends on: AI-14, AI-19` · `Blocks: AI-36` (reason: the scope overlap that would otherwise leave one behavior owned by two nodes); scheduled Wave 5 with both reasons stated. Leaves: **AI-41.1** — the testkit emission-boundary check's fourth rule is exercised on its failure path (an event that rule itself rejects); **AI-41.2** — the provider-failure payload gains a redacting alternate-verb formatting method matching the pattern its sibling payloads already carry. Opening blockquote dates the append, states why now (a third silent carryover is the failure mode; assignment lands now, work lands in Wave 5), **and enumerates the four navigational surfaces the append forces**: the header denominator (A8), the Quick navigation Wave 5 line, the mermaid W5 label, and the Delivery sequence Wave 5 row |
| A7 | `### AI-36 — Enforce secret redaction` (charter) | none (additive) | Blockquote mirrors the new edge: AI-36 now additionally depends on AI-41 — its adversarial sweep must not run against a type still missing its redacting method. The authoritative edge lives on AI-41's `Blocks:` field per the append-only rule; this mirror exists so AI-36's reader sees it. **AI-36's own `Depends on:` line is not edited** |
| A8 | Document header block (the `> **Status:**` line) | `~~41~~` inside the blockquote | Dated blockquote **immediately below the header block**: `> **Amended 2026-08-03 (AI-24)** — milestone total ~~41~~ → 42; AI-41 appended per the Wave-2 carryover assignment. Shipped count unchanged at 24.` The status line itself then reads "**24 of 42** milestones shipped" |

### 4.1 Status line and navigation — the denominator change

Appending AI-41 makes the milestone total **42**. The repo's real precedent, read precisely, is "**progress reported in place, substantive graph changes amended**": every prior in-place status edit moved the *numerator* (a progress report), while the *denominator* is a statement about the milestone set itself, governed by the append-only rule and the living-graph clause — so it changes only under A8's dated blockquote (`S-APD-048`: no heading changed without one; `R-APD-016`: **every** forced change lands as an amendment). Four surfaces must agree, updated in **one task** at apply time and verified together:

| Surface | Edit | Amendment cover |
| --- | --- | --- |
| Header `> **Status:**` line | Reads "**24 of 42** milestones shipped" | **A8** — the dated blockquote immediately below the header block carries the struck `~~41~~` and the reason; future wave closes keep moving the numerator in place |
| Quick navigation, Wave 5 line | Append ` · [AI-41](#ai-41--discharge-the-wave-2-carryovers)` | Enumerated in **A6**'s blockquote |
| Global dependency graph (mermaid) | W5 node label becomes `AI-33 … AI-37 · AI-41` | Enumerated in **A6**'s blockquote |
| Delivery sequence table, Wave 5 row | Milestones cell becomes "AI-33 to AI-37, AI-41"; exit condition gains "…and the Wave-2 carryovers are discharged" | Enumerated in **A6**'s blockquote |

**Consistency check** (verification pass): grep for `of 42`, the A8 blockquote below the header, the AI-41 anchor in navigation, the mermaid label, and the delivery-sequence row — all five present or the amendment task is incomplete.

One further file: `openspec/specs/ai-stream-testkit/spec.md`'s carryover status line ("Still unassigned…") is updated to name AI-41 as owner — a prose status edit, no requirement touched (`S-APD-062`).

## 5. The `[decision]` closing mechanism

Doc 0002: a decision leaf closes when "the decision artifact answers every listed question and is merged." The artifact self-evidences this with § 14, following AI-03 § 14's precedent, extended to two nodes:

- **Seven rows** — AI-24.1 items 1–4, then AI-24.2 items 1–3. Columns: node · checklist item (quoted from doc 0002) · where answered (section reference) · status.
- Because §§ 4–8 and §§ 9–11 follow the two checklists in order, each row's "where answered" is a **single contiguous span** — §§ 4–5 for AI-24.1 item 1 (the comparison and its priced losses are one answer), a single section for every other item — so a reviewer walks doc 0002 and the artifact in parallel with no hunting.
- Below the table: the milestone acceptance clause restated and checked (one adapter named, rejected alternatives explained, ADR condition did not fire), node status ("AI-24.1 and AI-24.2 close on merge; no `make test` gate applies — the change touches nothing under `backend/`"), and the unblocked list (AI-25 … AI-32).

## 6. File changes

| File | Action | Owner phase |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-first-provider-decision/design.md` | Create (this file) | this phase |
| `…/decision.md` | Create per § 2 | apply |
| `…/tasks.md` | Create | tasks |
| `…/specs/ai-first-provider-decision/spec.md` | Created by the parallel spec phase | spec |
| `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` | Amend per § 4 (A0a–A8) | apply |
| `openspec/specs/ai-stream-testkit/spec.md` | One status line | apply |
| Anything under `backend/`, `go.mod`, `go.work` | **None** | — |

**Routing recorded, not done here:** the AI-00.3 guard's stale code comment (it names AI-24 as the milestone adding a second allowlist entry — an entry never added under this decision) is Go source and belongs to **AI-25**, the first milestone that opens that package. `decision.md` § 9 records this routing.

## 7. Verification approach

Everything is checkable by inspection; nothing runs. Ordered by cost of a missed defect:

| Rank | Check | Cost if missed |
| --- | --- | --- |
| 1 | A1's AI-25.2 correction present, struck text exact, replacement worded as "a call-site scan over the adapter package's own source files", the import-path/transitive-import reason stated | The guard ships as an import scan that cannot bite — worse than no guard, because it is believed |
| 2 | § 10's separation holds: `[DONE]` and data-only appear **only** under § 10.2, each with a fixture-pin obligation — **and every § 10.1 claim's cited § 9.2 subsection is verified to actually state it**, so a fabricated spec citation cannot pass (`S-APD-043`) | AI-27 builds a decoder conforming to something no spec states |
| 3 | § 8 total (eight rows), expectations only `satisfied`/`absent`, standing copied from AI-03, the floor clause opens the section, `CAP-R-03`'s row names the usage opt-in option, `CAP-O-01` marked *pending AI-29.0's confirmation* | AI-38.2 has nothing comparable to assert against, or an adapter bug ships disguised as a vendor limitation |
| 4 | A5 is a note, not a verdict; § 13.2's reservation names AI-29.0 as owner; no `[decision]` node struck anywhere | A downstream decision node is deleted |
| 5 | A6/A7/A8 present; § 13.3 present in the artifact with both Wave-5 scheduling reasons; all four § 4.1 surfaces agree on 42; **negative check: AI-36's `Depends on:` line confirmed unedited** | AI-36 sweeps early, the artifact fails `R-APD-017`, or the graph's own arithmetic contradicts itself |
| 6 | Amendment convention: dated blockquotes under every touched heading including the header block, strikethrough only on superseded claims (never on the three in-force no-subject branches), same PR, append-only ordinals | Silent edits — the exact failure the living-graph clause exists to prevent |
| 7 | No Go identifier of the Layer 1 contract anywhere in the artifacts (the transport's stdlib path, the module manifest, and vendor wire field names are the only permitted identifier categories); zero files under `backend/` | doc 0002's authoring constraint broken |

## 8. Threat matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Documentation-only change.

## 9. Open questions

None blocking. The one deliberately open item — whether the chosen backend emits a reasoning extension field — is AI-29.0's by design, and this artifact's job is to keep it open.

## 10. Next phase

`tasks.md` — seven checklist tasks (four AI-24.1, three AI-24.2), the amendment block task (§ 4, single task for all doc 0002 edits A0a–A8 plus the testkit status line), and the verification pass (§ 7). Zero red-green phases: `[decision]` nodes ship no code and close on a merged artifact.
