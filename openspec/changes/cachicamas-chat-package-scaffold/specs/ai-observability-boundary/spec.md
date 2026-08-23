# Delta for `ai-observability-boundary`

> **Change**: `cachicamas-chat-package-scaffold` · **Milestone**: CH-01 (Wave 0) of doc 0005
> **Target**: `openspec/specs/ai-observability-boundary/spec.md`
> **Authority to amend**: that spec's own rule at `spec.md:26` — *"A later milestone that needs to change one of these invariants amends **this file**, in the same pull request, under its own ADR gate."* Its status line at `spec.md:5` calls its invariants **live**.
> **ADR gate**: [ADR 0005 § D3](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary), the governing ADR this file already names at `spec.md:6`. No ADR is amended; § D3's existing table supplies the replacement reasoning.

## Why this delta exists

`spec.md:228`, a row of the *Out of scope, with the owner of each* table, reads:

> `| Tracing in Layer 2, Layer 3 or the composition root | Docs 0003/0004 — those directories do not exist |`

CH-01.1 creates `backend/agent/src/chat/` (the Layer 3 position) and `backend/agent/src/cmd/chat/` (the composition root). Verified in this worktree: `git ls-files -- backend/agent/src/chat backend/agent/src/cmd` lists `backend/agent/src/chat/doc.go` and `backend/agent/src/cmd/chat/main.go`. Two of the row's three "those directories do not exist" clauses therefore become false on the day CH-01 merges.

**This row is not the same case as this change's other two repairs, and the difference is recorded rather than smoothed over.** The row was **already partly false before CH-01 touched anything**: `backend/agent/src/agent/` — the Layer 2 directory — has existed since AG-03, and the row has been carrying a false clause about it ever since. That third of the claim is **AG-03's back-annotation debt, not this change's**.

| Clause of the row | State before CH-01 | State after CH-01 | Whose debt |
|---|---|---|---|
| Layer 2's directory does not exist | **already false** — `backend/agent/src/agent/` created by AG-03 | still false | **AG-03's**, inherited already broken |
| Layer 3's directory does not exist | true | **false** — `backend/agent/src/chat/` | **CH-01's**, forced by this merge |
| The composition root's directory does not exist | true | **false** — `backend/agent/src/cmd/chat/` | **CH-01's**, forced by this merge |

CH-01 repairs the whole row because it **worsens** it — a change that makes a false statement more false, and then leaves it, has chosen to ship the falsehood. But the delta does not present the Layer 2 clause as CH-01's to fix: it was broken before this branch existed, and a reader five milestones later should be able to see which milestone owed which correction.

This site was found by a deliberate sweep of `openspec/specs/` for the whole defect shape — any promoted claim that a directory does not exist, or that a layer has no code. The sweep found four sites; three are repaired by deltas in this change and one is recorded unrepaired. All four, with their dispositions, are registered in `chat-package-boundary`'s **Untemporal-invariant register**.

**The row's obligation is not weakened.** Tracing in Layer 2, Layer 3 and the composition root remains out of scope for `ai-observability-boundary`. Only the **reason** changes: from a directory-existence claim, which time falsifies, to the owner reasoning that was always the real ground — those positions' own layer documents, and ADR 0005 § D3's table, which assigns each position its own observability grant. A reason that expires is the defect; the ownership is unchanged.

## This file's in-place revision convention — checked, not assumed

The two targets this change already amends use different conventions, so this one was read in full rather than inferred.

- `agent-package-scaffold` records revisions with a `(Previously: …)` line under the requirement, and keeps a list of `> **Amended …**` blockquotes after its Status section.
- `chat-archetype-contract` keeps its `> **Amended …**` lines in the **header block** (`:6`).
- **`ai-observability-boundary` has neither.** It carries **no amendment log at all**, and its in-place revision convention is a **parenthetical appended to the revised item itself**, stating what was narrowed or substituted and why — `S-AOB-003` (`:62`), `S-AOB-009` (`:86`), `S-AOB-027` (`:167`), `S-AOB-029` (`:169`) and `S-AOB-038` (`:201`) all carry one, in the forms `(Narrowed: …)`, `(Narrowed to …)` and `(Accessor substitution recorded: …)`. Its only change-history prose is the dated verification paragraph closing *Acceptance criteria* (`:242`).

This delta therefore follows **that** file's own shape: the revised row carries its parenthetical in place, and the promotion note is a short dated paragraph appended to the *Status* section (`:22-26`) — the section whose own closing sentence (`:26`) establishes that amendment happens by amending this file. **No `> **Amended …**` blockquote is introduced**, because importing another file's convention into this one would be a silent format change nobody asked for.

## Scope of this delta, pinned

**Changed** — and nothing else:

1. The single table row at `spec.md:228`: its **Owner** cell's reasoning, plus an in-place parenthetical recording the correction and its split ownership.

**Deliberately untouched**, so a reviewer can diff the claim:

- The row's **Item** cell — *"Tracing in Layer 2, Layer 3 or the composition root"* — survives **byte-identical**. The scope fence itself is not moved.
- Every other row of the *Out of scope* table (`spec.md:220-234`) survives byte-identical, including the composition-root row at `:222` — which cites the charter and § D3 and was never a directory-existence claim.
- **No requirement, no scenario and no non-functional requirement is touched.** `R-AOB-001`…`R-AOB-009`, `S-AOB-001`…`S-AOB-041`, `NFR-AOB-001` and `NFR-AOB-002` are all unchanged, as are the twelve-key allowlist, the denylist, the Identity table and the Acceptance criteria.
- **No identifier is minted and none is renumbered.** The file is append-only (`spec.md:8`); this delta appends no identifier at all, because the amended text is a table row that carries none.
- This change adds **no** tracing anywhere. `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` ship documentation and an empty `main` (`chat-package-boundary` `R-CPB-001`); neither imports an observability path. The row's obligation is satisfied by this change, not merely restated by it.

## MODIFIED Requirements

**None.** This delta modifies no requirement, no scenario and no non-functional requirement — and that fact is stated under this heading **on purpose**, because a reader or executor that promotes by scanning for `## MODIFIED Requirements` would otherwise find no such section, conclude there was nothing to do, and silently drop this delta entirely. Its whole normative change is a **table row**, and it is mandatory. The two edits it requires are enumerated below; **both** MUST be applied.

## PROMOTION INSTRUCTION — the archive executor MUST apply both edits below

This delta is **not** discharged by promoting a requirement block. It is discharged by making exactly two textual edits to one file. Applying neither, or only one, leaves a promoted spec carrying a claim this change falsified.

**Target file**: `openspec/specs/ai-observability-boundary/spec.md`

### Edit 1 of 2 — replace one row of the *Out of scope, with the owner of each* table

**Location**: `spec.md:228`, the row whose **Item** cell reads `Tracing in Layer 2, Layer 3 or the composition root`. It is the seventh row of the table that runs `spec.md:220-234`.

**BEFORE** (the exact line to replace):

```markdown
| Tracing in Layer 2, Layer 3 or the composition root | Docs 0003/0004 — those directories do not exist |
```

**AFTER** (the exact replacement; the **Item** cell is byte-identical, only the **Owner** cell changes):

```markdown
| Tracing in Layer 2, Layer 3 or the composition root | Those positions' own layer documents (docs 0003 / 0004 / 0005) and [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary), whose table grants each position its own observability surface — Layer 1's grant is what this capability specifies, and no other position's is. *(Reason corrected at CH-01: the cell previously read "those directories do not exist", which time falsified. `backend/agent/src/agent/` has existed since AG-03 — that clause was already false and is AG-03's back-annotation debt, inherited here rather than introduced. `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` were created by CH-01.1 of doc 0005, which is what forced this correction. The scope fence is unchanged: tracing in those three positions is still out of scope for this capability.)* |
```

No other row of that table is touched. The parenthetical follows this file's own in-place revision convention (`spec.md:62`, `:86`, `:167`, `:169`, `:201`).

### Edit 2 of 2 — append the amendment note

**Location**: the *Status — this file is the canonical home of the contract* section, immediately after its closing sentence at `spec.md:26`.

The archive executor MUST append the dated paragraph below. It MUST NOT introduce a `> **Amended …**` blockquote list, which this file does not use.

> **Amended 2026-08-23 (CH-01, `cachicamas-chat-package-scaffold`).** One row of *Out of scope, with the owner of each* is corrected: the entry fencing tracing in Layer 2, Layer 3 and the composition root previously gave its reason as "those directories do not exist". Two of the three directories now exist — `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/`, created by CH-01.1 of doc 0005 — and the third, `backend/agent/src/agent/`, has existed since AG-03, so that clause was already false when CH-01 met it and is recorded as AG-03's back-annotation debt rather than as this change's. The reason is replaced by the owner reasoning that was always the real ground: those positions' own layer documents and ADR 0005 § D3's table. **The scope fence is unchanged** — tracing in those three positions remains out of scope for this capability — and no requirement, scenario or non-functional requirement is touched, no identifier is minted, and nothing is renumbered. CH-01 adds no tracing in any position; its two new packages ship documentation and an empty `main`.

## Verification of this delta

- **V-1** — Given the promoted target after archive, when the amended row's **Item** cell is diffed against its pre-promotion bytes, then it is byte-identical — the scope fence was not moved, only its reason replaced.
- **V-2** — Given the promoted target after archive, when every other row of the *Out of scope* table is diffed against its pre-promotion bytes, then each is byte-identical.
- **V-3** — Given the promoted target after archive, when the set of requirement, scenario and non-functional identifiers is diffed against the pre-promotion set, then the two sets are **equal** — this delta mints nothing and renumbers nothing.
- **V-4** — Given the promoted target after archive, when it is searched for any claim that `backend/agent/src/agent/`, `backend/agent/src/chat/` or `backend/agent/src/cmd/` does not exist, then none remains.
- **V-5** — Given the promoted target after archive, when the amended row is read, then it states which clause was **already** false before this change and whose debt that is, and which clauses **this** merge falsified — a reader can attribute each correction without opening this delta.
- **V-6** — Given the promoted target after archive, when its format is compared against its pre-promotion format, then no `> **Amended …**` blockquote list has been introduced and the amendment is a dated paragraph in the *Status* section, matching this file's own convention.
- **V-7** — Given this change's merged tree, when `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` are scanned for any OpenTelemetry import, then none exists — the fenced obligation is satisfied, not merely restated. (`chat-package-boundary` `S-CPB-002` and `S-CPB-003` carry the declares-nothing evidence.)
- **V-8** — Given the promoted target after archive, when it is searched for the exact string `those directories do not exist`, then no occurrence remains, **and** the *Status* section carries the dated amendment paragraph. Both edits of the promotion instruction were applied; finding the first applied without the second, or neither, means this delta was promoted partially or skipped — the failure mode a delta with no `## MODIFIED Requirements` block is exposed to.
