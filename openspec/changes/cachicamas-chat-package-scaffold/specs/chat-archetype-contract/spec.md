# Delta for `chat-archetype-contract`

> **Change**: `cachicamas-chat-package-scaffold` · **Milestone**: CH-01 (Wave 0) of doc 0005 · same reasoning as proposal **D-2**
> **Target**: `openspec/specs/chat-archetype-contract/spec.md`
> **Authority to amend**: that spec's own append-only rule (`spec.md:9`) and its amendment convention, a `> **Amended …**` line in the header block (`spec.md:6`). This delta renumbers nothing and mints nothing.

## Why this delta exists

`R-CHT-007` (`spec.md:145-155`), promoted by CH-00 on 2026-08-23, requires the record to *"state plainly that **neither path exists yet**, and that there is no `backend/agent/src/cmd/` directory at all"* (`spec.md:149`), and `S-CHT-062` (`spec.md:155`) reads:

> Given a reader who checks the repository against the identity section, when the two paths are **not found**, then the record has already said so and no statement in it is falsified by their absence.

CH-01.1 created both paths. Verified in this worktree: `git ls-files -- backend/agent/src/chat backend/agent/src/cmd` lists `backend/agent/src/chat/doc.go` and `backend/agent/src/cmd/chat/main.go`. `S-CHT-062`'s `Given` is therefore **unreachable** — the two paths are found — and `R-CHT-007`'s normative "neither path exists yet" clause is false against the tree that carries it. `S-CHT-061` (`spec.md:154`) carries the same claim and is stale for the same reason. `S-CHT-060` (`spec.md:153`) asserts only that the record names the two paths and cites `0005:8`; it is **unaffected** and is reproduced byte-identical below.

**This is the identical defect shape the `agent-package-scaffold` delta in this very change exists to repair** — a promoted invariant stated without temporal scope, falsified by the merge of a later milestone that the same document already named as the owner of the falsifying act. `R-CHT-007` even names CH-01.1 as that owner in its own text (`spec.md:149`). The whole pipeline for this change — explore, propose, design, spec, tasks, apply — walked past it, including the phase that wrote the first delta while holding the second stale requirement open in the same reading. Recorded here rather than in a register, because the register entry would itself be the third instance of the shape.

## Sibling staleness — a checked result, not an unexamined absence

The whole target file was read and searched for any further claim that the archetype's paths do not exist, or that no Layer 3 archetype is on disk. Four sites carry the claim; two are requirements and two are non-normative prose. No fifth exists.

| Site | Text | Disposition |
|---|---|---|
| `spec.md:145`, `:149` (`R-CHT-007`) | "neither path exists yet"; "no `backend/agent/src/cmd/` directory at all" | **MODIFIED** below |
| `spec.md:154`, `:155` (`S-CHT-061`, `S-CHT-062`) | same claim, and a `Given` that is now unreachable | **MODIFIED** below |
| `spec.md:226` (`R-CHT-013`) | "the package such a guard would protect does not exist" | **MODIFIED** below — a rationale clause, but stated in the present tense and now false. Fixing `R-CHT-007` alone would leave the same false sentence one requirement away |
| `spec.md:19` (Purpose) and `spec.md:256` (Explicit non-requirements row) | "The chat archetype does not exist yet — there is no `backend/agent/src/chat/`, and no `backend/agent/src/cmd/` at all"; "the package a guard would protect does not exist" | **Non-normative prose**, carried below under *Prose corrections* with exact replacements, because leaving them would keep a false sentence in a promoted spec even after the requirements are repaired |

Checked and found **not** stale, stated so the absence is a result rather than an omission:

- `S-CHT-102` (`spec.md:206`) and `S-CHT-111` (`spec.md:219`) — both assert that no promoted spec is modified **by this change**, where "this change" is CH-00. CH-01 modifying this file does not falsify either; each is already scoped to its own merge.
- The *Explicit non-requirements* row at `spec.md:258` — "Creating `backend/agent/src/chat/` or `backend/agent/src/cmd/chat/` … owned by CH-01.1" — remains true. It states what CH-00 did not require and who owns the act; CH-01.1 performing it is the row being satisfied, not falsified.
- `R-CHT-009`'s `REQ-5` clause (`spec.md:174`) and `R-CHT-010`'s register rules are untouched by the existence of the two packages.

## Scope of this delta, pinned

**Changed** — and nothing else:

1. `R-CHT-007`'s title and second paragraph: the non-existence clause is scoped to **CH-00's own merge state** instead of standing as a live invariant.
2. `S-CHT-061` and `S-CHT-062`: scoped to the same merge state, each with its live successor obligation named.
3. `R-CHT-013`'s single rationale clause: the reason the milestone ships no guard is scoped to CH-00's moment.

**Deliberately untouched**, so a reviewer can diff the claim:

- `R-CHT-007`'s **first paragraph** — the archetype's name, its package path `backend/agent/src/chat/`, its composition root `backend/agent/src/cmd/chat/`, and the `0005:8` and ADR 0005 § D2 citations — survives **byte-identical**. Those statements were true when written and are true now; CH-01 confirms them rather than contradicting them.
- `S-CHT-060` survives byte-identical.
- `R-CHT-013`'s normative sentences — the charter-acceptance restatement, the `[decision]` typing, and the prohibition on discharging it by production code, a Go test or a guard — survive byte-identical. **Only the parenthetical reason changes.** The prohibition is not weakened: CH-00's acceptance is still discharged by reading the record, and this change does not retro-fit a guard to it.
- `S-CHT-120`, `S-CHT-121` and `S-CHT-122` survive byte-identical.
- No other requirement, scenario or non-functional requirement of this spec changes.
- **No identifier is minted and none is renumbered.** The file is append-only (`spec.md:9`); this delta appends nothing.

## MODIFIED Requirements

### R-CHT-007 — The archetype's name, package path and composition root are stated, together with the fact that neither path existed at CH-00

The record MUST state the archetype's name, its package path `backend/agent/src/chat/` and its composition root `backend/agent/src/cmd/chat/`, citing `0005:8` and ADR 0005 § D2.

It MUST state plainly that, **at CH-00's own merge state**, neither path existed and there was no `backend/agent/src/cmd/` directory at all. Stating the paths as though they existed would have made the record false on the day it landed, and CH-01.1 owns their creation (`0005:228`). That clause records the record's accuracy **at the moment CH-00 merged**; it is **not** a live invariant over the module's lifetime. From CH-01.1 onward both paths exist by design, and the live obligation moves: what remains true forever is that the record was accurate when written, and what now asserts the two packages' existence and shape is `chat-package-boundary` `R-CPB-001`, promoted by CH-01 in the same pull request as this amendment.

(Previously: the non-existence clause was stated with no temporal scope, so it read as a standing invariant; CH-01.1 created both paths and falsified it on the day CH-01 merged — the same defect shape, and the same repair, as this change's `agent-package-scaffold` delta on `R-AGP-001`.)

#### Scenarios

- **S-CHT-060** — Given the record, when a reader asks where this archetype and its composition root live, then the record names `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` and cites `0005:8`.
- **S-CHT-061** — Given the record **as it stood at CH-00's merge state**, when the identity section is read, then it states that neither path existed yet and that there was no `backend/agent/src/cmd/` at all, and it names CH-01.1 as the owner of their creation. The live successor obligation is that the record remains accurate **as a record of CH-00's moment** — its text is not rewritten by a later milestone — while `chat-package-boundary` `R-CPB-001` carries the current-state claim.

  (Previously: stated over the record with no temporal scope, so a reader checking it against any post-CH-01 tree would find a promoted scenario asserting a false present-tense fact.)
- **S-CHT-062** — Given a reader who checks the repository **at CH-00's merge state** against the identity section, when the two paths are not found, then the record has already said so and no statement in it is falsified by their absence. From CH-01.1 onward the two paths **are** found, and the record is not falsified by their presence either, because its claim is scoped to the moment it was written; `chat-package-boundary` `S-CPB-001` and `S-CPB-003` are what assert the paths at the current tree.

  (Previously: the `Given` required the two paths to be **not found**, which CH-01.1 made unreachable — a scenario nobody can run is not a satisfied scenario, and an unreachable `Given` hides that fact behind an assertion that never executes.)

### R-CHT-013 — The record answers any seam question directly, and that is this milestone's acceptance

The record MUST satisfy the charter's acceptance verbatim (`0005:210`): given the record, when a reader asks what this archetype's answer is to any seam Layer 2 names, then the record answers it directly and names that seam's injection point, without the reader consulting source.

This is a property of the **record**, discharged by reading it. This milestone is typed `[decision]` (`0005:215`) and MUST NOT be discharged by production code, by a new Go test, or by a guard: **at CH-00 the package such a guard would protect did not yet exist** (`R-CHT-007`, proposal D-1). The earliest honest mechanical binding is CH-01.2's import guard, not here — and CH-01.2 is where it landed.

(Previously: the rationale clause read "the package such a guard would protect does not exist", in the present tense; CH-01.1 created it. The requirement's own prohibition is unchanged — CH-00's acceptance is still discharged by reading the record, and no guard is retro-fitted to it — only the reason is scoped to the moment it was true.)

#### Scenarios

- **S-CHT-120** — Given the record and any seam of AG-23's frozen enumeration chosen at random by a reviewer, when the reviewer asks that seam's v1 answer and injection point, then the record supplies both directly and the reviewer opens no Go source file to obtain either.
- **S-CHT-121** — Given the record and the seven questions at `0005:219-225`, when each question is asked in turn, then the record carries a written answer to each, and the closing-verification section of the record names where each answer lives.
- **S-CHT-122** — Given this change as merged, when its diff is inspected, then it adds no Go file, no test file, no guard, and no change under `backend/` or `frontend/`; and `cd backend/agent && go test -race -count=1 ./...` run uncached is green with its wall-clock duration recorded.

## Prose corrections (non-normative, same defect)

These two sites carry the same false present-tense claim outside any requirement. The archive executor MUST apply them in the same promotion, because a spec whose requirements are repaired while its prose still asserts the opposite teaches the prose.

| Location | Current text | Replacement |
|---|---|---|
| `spec.md:19` (Purpose, first sentence) | "The chat archetype does not exist yet — there is no `backend/agent/src/chat/`, and no `backend/agent/src/cmd/` at all." | "When CH-00 was written the chat archetype did not exist — there was no `backend/agent/src/chat/`, and no `backend/agent/src/cmd/` at all. CH-01.1 created both; `chat-package-boundary` `R-CPB-001` is where their existence and shape are now asserted." |
| `spec.md:256` (Explicit non-requirements, first row's *Why* cell) | "Proposal D-1 — CH-00.1 is `[decision]`; the package a guard would protect does not exist. CH-01.2 owns the first mechanical binding" | "Proposal D-1 — CH-00.1 is `[decision]`; at CH-00 the package a guard would protect did not yet exist. CH-01.2 owns the first mechanical binding, and landed it" |

The remainder of the Purpose section and the whole *Explicit non-requirements* table are otherwise unchanged.

## Amendment note to append at promotion

The archive executor MUST append the following blockquote to the target spec's header amendment list (after the CH-00 archive line at `spec.md:6`), matching that block's existing convention:

> **Amended 2026-08-23 (CH-01, `cachicamas-chat-package-scaffold`)** by the archive executor: `R-CHT-007` and `R-CHT-013` are MODIFIED. `R-CHT-007`'s "neither path exists yet" clause, and `S-CHT-061` and `S-CHT-062` with it, are scoped to **CH-00's own merge state** rather than stated as live invariants: CH-01.1 created `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/`, which made `R-CHT-007`'s clause false and `S-CHT-062`'s `Given` unreachable on the day CH-01 merged. `R-CHT-013`'s rationale clause — "the package such a guard would protect does not exist" — is scoped the same way; its normative prohibition is unchanged and no guard is retro-fitted to CH-00's acceptance. Two non-normative sites carrying the same claim (the Purpose section's opening sentence and the first *Explicit non-requirements* row) are corrected in the same promotion. `R-CHT-007`'s first paragraph, `S-CHT-060`, `R-CHT-013`'s normative sentences and `S-CHT-120`…`S-CHT-122` are **byte-unchanged**; no other requirement, scenario or non-functional requirement is touched; **no identifier is minted and none is renumbered**, so the file's append-only rule (`spec.md:9`) holds. The current-state claim these clauses used to carry now lives in `chat-package-boundary` `R-CPB-001`, promoted by the same pull request. This is one of **three** repairs of a single defect shape in one change — the others being the `agent-package-scaffold` delta on `R-AGP-001`/`S-AGP-003` and the `ai-observability-boundary` delta on that spec's out-of-scope row — found by a deliberate sweep of `openspec/specs/` that also turned up a fourth site, `agent-contract-vocabulary` `NFR-AGV-C`, recorded and deliberately not repaired here. All four are registered in `chat-package-boundary`'s **Untemporal-invariant register**.

## Verification of this delta

- **V-1** — Given the promoted target after archive, when `R-CHT-007`'s first paragraph and `S-CHT-060` are diffed against their pre-promotion bytes, then each is byte-identical.
- **V-2** — Given the promoted target after archive, when `R-CHT-013`'s normative sentences and `S-CHT-120`, `S-CHT-121`, `S-CHT-122` are diffed against their pre-promotion bytes, then each is byte-identical.
- **V-3** — Given the promoted target after archive, when it is searched for any clause asserting in the present tense that `backend/agent/src/chat/` or `backend/agent/src/cmd/` does not exist, then none remains — in a requirement, in a scenario, or in prose.
- **V-4** — Given the promoted target after archive, when the set of requirement, scenario and non-functional identifiers is diffed against the pre-promotion set, then the two sets are **equal** — this delta mints nothing and renumbers nothing.
- **V-5** — Given the promoted target after archive, when `S-CHT-062` is read against the current tree, then its `Given` is reachable at CH-00's recorded merge state and its post-CH-01 clause is satisfied by the two paths being found — the scenario is runnable at both, rather than unreachable at either.
- **V-6** — Given the merged pull request, when each spec delta of this change is read, then each points at `chat-package-boundary`'s **Untemporal-invariant register**, which carries all four swept sites with their dispositions — so a reader who finds one repair finds the complete result rather than concluding the shape was a one-off.
