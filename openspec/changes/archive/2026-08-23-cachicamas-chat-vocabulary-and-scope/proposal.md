# Proposal: CH-00 — Record the archetype's vocabulary, seam answers and v1 scope

| | |
|---|---|
| **Change** | `cachicamas-chat-vocabulary-and-scope` |
| **Milestone** | CH-00 (`docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:202-240`) · Wave 0 · **1 of 12 — the first milestone; nothing depends on anything** |
| **Closes** | R-02, R-03, and register rows 3, 5, 6 |
| **Depends on** | nothing. **Blocks** CH-01 |
| **Worktree / base** | `cachicamas-worktrees/ch00-chat-vocabulary-scope`, branch `feat/chat-archetype-wave0-ch00` |
| **Artifact store** | hybrid (`openspec/changes/…` + Engram `sdd/cachicamas-chat-vocabulary-and-scope/*`) |
| **Delivery** | single-pr · review budget 1000 counted lines (extension pre-authorised) |
| **Evidence runner** | `cd backend/agent && go test -race -count=1 ./...` — wall-clock recorded. **A `(cached)` result is not evidence.** `make all` MUST NOT be run: its fmt step rewrites committed files |
| **Prefix** | **`CHT`**, capability `chat-archetype-contract` — verified free: 0 hits across `openspec/specs/`, `openspec/changes/`, `openspec/changes/archive/`. `ARC` is **not** free |
| **Exploration** | `openspec/changes/cachicamas-chat-vocabulary-and-scope/explore.md` |

> Every `file:line` cited below was re-resolved in this worktree during this phase, or is inherited from an exploration whose verification log records the check. Two citations the exploration corrected are used in their corrected form only.

## Intent

The chat archetype does not exist yet — there is no `backend/agent/src/chat/`, and no `backend/agent/src/cmd/` at all. Everything about it is still a sentence in a milestone doc, which means every later CH milestone is free to invent its own answer to the same question and no test will disagree.

Layer 2 froze its surface at AG-23 and published, for each seam, an injection point and a v1 default. It did that so the first consumer would have to *state* its answers rather than discover them. The trap CH-00 exists to prevent is named in its own charter (`0005:213`): **a seam left implicit**. "This archetype has no tools" is not a seam answer. "The tool source is empty in v1, injected at the registry seam on the run value, and CH-09 is where it stops being empty" is.

There is a second, quieter failure this record must avoid. Layer 2 has no term for a *session*, no term for a human *participant*, and no term for a *part*. If the record presents this archetype's nouns as a clean mapping onto Layer 2's register, it will read as though Layer 2 blessed vocabulary it never defined. The record must distinguish what **maps** onto Layer 2, what is **coined** here, and what is **inherited** from Layer 1 — three provenances, not two (the third was added during verification; see `design.md` D-B's 2026-08-22 amendment).

## Scope

### In scope

1. **A decision record** (`decision.md` in this change folder) closing the seven questions of CH-00.1 (`0005:219-225`), cited by every later CH milestone.
2. **One capability spec** — `chat-archetype-contract`, prefix `CHT` — promoting the record's falsifiable claims into requirements and scenarios.
3. **The vocabulary section**, marking each noun as exactly one of: a mapping onto a named `VL2-*` row, an explicit **coinage** with the gap that forced it, or a noun **inherited from Layer 1** with its `V-*` id (the third block was added during verification — see `design.md` D-B's 2026-08-22 amendment — because "message" is Layer 1's `V-REQ-02` and fits neither of the first two).
4. **The seam section** — exactly the **11** seams the frozen AG-23 enumeration names (8 frozen + 3 marked experimental), each with **both** injection point and v1 answer, none omitted.
5. **A separately labelled gap-findings section** recording Layer 2 seams that AG-23's enumeration does not carry (v2 § 6 seam 3 sandbox policy, seam 7 retry classification, `Harness.RetryTiming`, and the settable-but-unenumerated `RetryAttempts` / `ContextBudget` / `System`).
6. **An inconsistency register** naming the three defects below as found, cited, and **forwarded** — not repaired.
7. **Doc 0005 bookkeeping** in the same PR: tick the CH-00.1 completion-checklist row, move the status line to **1 of 12**.

### Out of scope

| Excluded | Reason |
|---|---|
| Implementing **any** seam answer | Charter, `0005:212` — owned by every later milestone |
| The package `backend/agent/src/chat/` itself | `0005:228` — owned by CH-01.1 |
| Renaming "application" → "archetype" in Layer 2's promoted spec | `0005:212` — owned by ADR 0009 § D7(a)'s own SDD change |
| Repairing the `VL2-*` citation defect, the ADR 0009 § D7(a) occurrences, or the `frontend-chat-layer1` guard-chain path | D-3 posture: **record, do not repair**. All three are forwarded as named follow-ups below |
| Any new Go file, guard test, or production code | D-1 |
| Re-litigating the deferred capabilities | `0005:997-1011` **states** them with their attaching seams; it does not decide them |
| Retiring the `"backend not wired — see PR for backend wire"` literal | Mandated by a promoted spec (`frontend-chat-layer1/spec.md:62`); needs a recorded spec delta at CH-05.2, never a silent deletion |

## Resolved decisions

### D-1 — The evidence gate is the record and its spec, not a test

**Decided.** This change ships a `decision.md`, one promoted capability spec, and the doc 0005 bookkeeping. It ships **no** new Go file, **no** guard test, and **no** production code.

**Why.** CH-00's charter puts implementing any answer out of scope, and CH-00.1 is typed `[decision]`. A guard test here would have to assert something about a package that does not exist. Closing evidence is therefore: the promoted spec, plus a green **uncached** `cd backend/agent && go test -race -count=1 ./...` proving this change regressed nothing. Strict TDD is satisfied vacuously — there is no production behaviour to drive.

**Consequence for verify.** The runner's wall-clock time is recorded. The agent module's real uncached race run is on the order of minutes; a sub-second result is a cache hit and is not evidence.

### D-2 — The record answers exactly 11 seams, then discloses the gap separately

**Decided.** The seam section carries the **8 frozen** seams (run's model provider; the permission decision port; the caller-owned wake handle; the tool registry; the transcript; the wider hook-family registration surface; the singular pre-request field; the tracing-API provider) and the **3 marked experimental** ones (the in-frame delegation door; the failover policy; the context-reduction policy). Verified this phase against `openspec/changes/archive/2026-08-21-cachicamas-agent-layer3-handoff/decision.md:31-53`; the experimental heading is at `:49`.

Each entry names **injection point AND v1 answer** — `S-L3H-028` (`openspec/specs/agent-layer3-handoff/spec.md:195`) forbids listing a seam without both. Each answer is stated directly and does not require reading source — `S-L3H-029` (`:196`).

A **separately labelled** section then records seams Layer 2's own v2 § 6 table names that AG-23's enumeration does not carry (`docs/architecture/0001-cachicamas-agent-stack-v2.md:658-683`). Separately labelled, because a gap finding is not a seam answer and must not be read as one.

**Two form constraints inherited from AG-23, carried verbatim into the record's rules.** (a) `agent-layer3-handoff/spec.md:183` requires the generic-client boundary be written "without reference to files, shells, skills or terminals" — CH-00's record is a *chat* archetype's record and may name chat concepts, but must name no coding-archetype concept. (b) `TurnOptions.PreRequestHook` is **frozen-and-superseded** (`archive/2026-08-21-…/decision.md:45`): kept, carrying no deprecation marker, and it MUST NOT be described as removed or deprecated anywhere in this change's artifacts.

**Rejected — answer only the seams v1 actually uses.** That is precisely the trap at `0005:213`. A seam with a deliberately empty answer is listed with that answer, never omitted.

### D-3 — Cite the archive where a row exists nowhere else; register the violation; forward the repair

**Decided.** Where a `VL2-*` row exists only in the archive, the record cites the archive, and the same record names that citation as a violation of `R-AGV-001` in an inconsistency register.

**Why the violation is unavoidable.** `openspec/specs/agent-contract-vocabulary/spec.md:339` is a placeholder — "[REGISTER CONTINUES WITH 86 ROWS IN 6 CATEGORIES — SEE ARCHIVED DECISION.MD FOR THE COMPLETE SNAPSHOT]". The 86 rows live only at `openspec/changes/archive/2026-08-11-cachicamas-agent-contract-vocabulary/decision.md`. `R-AGV-001` (`spec.md:36`) and `S-AGV-001` (`spec.md:40`) forbid the archive as a citation target for a term. So the register is **legally unreachable**: cite the spec and you cite a placeholder; cite the archive and you break the rule. Recording the defect honestly is the only move that leaves a true statement on the page.

**Rejected — repair the placeholder here.** Out of scope, and it would inline 86 rows into a change whose charter is a decision record.

**Rejected — paraphrase the rows without citation.** That launders the defect. The precedent is AG-23's own posture (`archive/2026-08-21-…/decision.md` § 4.3, heading `:110`): a defect "appears **nowhere** in the register above. Laundering it as a limitation would make this statement false on the day it lands."

### D-4 — One capability spec, slug `chat-archetype-contract`, prefix `CHT`

**Decided.** A single new capability at `openspec/specs/chat-archetype-contract/`, requirement prefix `CHT`. Verified free: 0 hits across `openspec/specs/`, `openspec/changes/` and `openspec/changes/archive/`. **`ARC` is not free** and was rejected on that ground before the slug was chosen.

**Why one spec and not several.** The record's claims are a single contract — what the words mean, what the seams answer, and what v1 excludes are three faces of one commitment. Splitting them would let a later change amend the vocabulary without touching the seam answers that use it.

## Follow-ups this change records and does **not** repair

| # | Defect | Evidence | Disposition |
|---|---|---|---|
| **F-1** | The `VL2-*` register is legally unreachable. The promoted spec carries a placeholder at `agent-contract-vocabulary/spec.md:339`; the 86 rows exist only in the archive; `R-AGV-001` (`:36`) and `S-AGV-001` (`:40`) forbid citing the archive for a term | verified this exploration | Recorded in the inconsistency register; repair is a future change against `agent-contract-vocabulary` |
| **F-2** | ADR 0009 § D7(a) is still open — no change has renamed "a Layer 3 application" to "archetype" in Layer 2's promoted specs. Occurrences (**eight, across three promoted specs**): `agent-contract-vocabulary/spec.md:146, :153, :335`; `agent-layer3-handoff/spec.md:17, :34, :177, :196`; `agent-v1-scope/spec.md:317`. This row originally listed six across two specs; it is corrected here, and `decision.md` § 9 records why the short list arose | grep-confirmed against the loose pattern `Layer 3 application` (8 hits / 3 specs), not the exact string `a Layer 3 application` (7 / 3) | Recorded. **F-1 and F-2 both land on `agent-contract-vocabulary/spec.md`, so one future change can close both** — the record says so explicitly |
| **F-3** | `openspec/specs/frontend-chat-layer1/spec.md` cites the wrong file for the auth guard chain at `:9`, `:40` and `:134`. It cites `frontend/src/routes/home/index.tsx:39-51`, which contains no guard chain (zero grep hits for all three function names) | verified this phase | Recorded. CH-00's record cites the **correct** location: `frontend/src/routes/home/layout.tsx:30-34` (`onRequest`: `setSsrCookieHeader` → `requireAuthRedirect` → `await requireOwnboarding`), doc comment `:21-29`, imports `:15-17` |

Until each follow-up lands, the record reads Layer 2's "a Layer 3 application" as "a Layer 3 archetype" under ADR 0009 § D7(a)'s substitution rule (`docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md:174-176`), and says so where it cites those artifacts.

## Capabilities

### New capabilities

- `chat-archetype-contract` — the archetype's vocabulary (mappings, coinages and Layer 1 inheritances, each structurally marked), its answer and injection point for all 11 AG-23 seams, its package and composition-root identity, its database-ownership answer, the frozen frontend wire it inherits, its v1 exclusions with attaching seams, the substitution rule, and the inconsistency register. Prefix **`CHT`** (verified free above).

### Modified capabilities

- **None.** No promoted spec's requirements change. F-1, F-2 and F-3 are recorded as defects against **four** promoted specs — `agent-contract-vocabulary` (F-1, F-2), `agent-layer3-handoff` (F-2), `agent-v1-scope` (F-2), `frontend-chat-layer1` (F-3) — but repaired by none of them here — that is D-3's posture and `0005:212`'s scope fence.

## Approach

1. Write `decision.md` question by question, in the order of `0005:219-225`, so a reader can verify closure against the checklist without reconstructing an argument.
2. For Q1, split the vocabulary table into blocks that cannot be confused — two at proposal time, three as landed, the third added for nouns inherited from Layer 1 (see `design.md` D-B's 2026-08-22 amendment): **mapped** nouns (`VL2-COR-04` run `:135`, `VL2-COR-05` turn `:136`, `VL2-COR-10` transcript, `VL2-COR-02` the loop, `VL2-COR-03` the harness, `VL2-COR-09` upward path, `VL2-COR-12` steering, `VL2-EVT-04` message lifecycle, `VL2-EVT-10` run outcome, `VL2-EVT-11` turn outcome) and **coined** nouns (conversation, participant), each coinage naming the gap that forced it — for session persistence, `VL2-OUT-07`, which is an *exclusion assigned to Layer 3*, not a term.
3. For Q2, walk `archive/2026-08-21-…/decision.md:31-53` in source order so no seam can be dropped by transcription; then the gap-findings section against v2 § 6.
4. For Q4, name the owner, not the intent: ADR 0009 § D6 (`0009-…md:152`, the quoted sentence at `:154-155`) — "Each business system owns its own tables; no archetype writes to another system's schema."
5. For Q6, reproduce doc 0005's deferral table (`0005:997-1011`) with each row's attaching seam, so a deferral is never a bare "not in v1".
6. Promote the falsifiable claims into `chat-archetype-contract` as `CHT` requirements with Given/When/Then scenarios; the acceptance at `0005:210` becomes the capability's headline scenario.

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `openspec/changes/cachicamas-chat-vocabulary-and-scope/` | New | proposal, design, spec delta, tasks, `decision.md`, verify-report |
| `openspec/specs/chat-archetype-contract/spec.md` | New | Promoted at archive |
| `docs/architecture/milestones/0005-…md` | Modified | CH-00.1 checklist row ticked; status line → **1 of 12** |
| `backend/agent/**`, `frontend/**` | **Unchanged** | Zero source edits — D-1 |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| A seam is transcribed with an injection point but no v1 answer, silently failing `S-L3H-028` | Med | The record walks `decision.md:31-53` in source order; the spec asserts the count is 11 and that no entry carries only one of the two halves |
| The vocabulary table reads as if Layer 2 defined "conversation" and "participant" | Med | Structurally distinct blocks, separated by their columns rather than by visual styling (design D-B chose structural over typographic, and the landed record carries three); every coinage names the gap that forced it, and `VL2-OUT-07` is quoted as an exclusion, not a term |
| A coding-archetype concept (file, shell, terminal, skill, repository) leaks into the record | Med | `agent-layer3-handoff/spec.md:183`'s constraint is restated as a rule in the record itself; the spec carries a scenario asserting the absence |
| The record describes `TurnOptions.PreRequestHook` as deprecated | Low | Explicit prohibition carried from `archive/2026-08-21-…/decision.md:45` into this proposal, the design and the spec |
| A recorded follow-up is read as a repair, or worse, silently repaired mid-change | Med | Follow-ups live in one labelled register with an explicit "recorded, not repaired" line; the "Modified capabilities" section says **None** |
| An inherited citation is stale and reproduced into a durable record | Med | The exploration's verification log found two real defects (a `VL2` off-by-one and the guard-chain path); both corrections are pinned in this proposal and must be used in their corrected form |
| The gap-findings section is read as extending Layer 2's frozen seam set | Low | Separately labelled and framed as findings against AG-23's enumeration, with no v1 answer attached |

## Rollback plan

Revert the single PR. This change adds two new directories under `openspec/` and edits two lines of `docs/architecture/milestones/0005-…md`. It touches **no** Go file, **no** frontend file, **no** `go.mod`/`go.sum`, no schema, no wire, and no configuration — so the revert is complete by construction and needs no migration or data step.

If the change lands and a seam answer is later found wrong, the rollback is **not** a revert: the record is the citation target of every later CH milestone, so a wrong answer is corrected by an amending SDD change against `chat-archetype-contract` that records what changed and why, exactly as AG-23 requires a breaking change to be an amendment rather than a silent drift.

If CH-01 is already merged when a defect is found, revert is unavailable and the amending path is mandatory.

## Dependencies

- None. CH-00 is the first milestone (`0005:211`, "**Depends on:** nothing").
- Read-only prerequisites, all already true: AG-23 merged and its surface frozen; ADR 0009 merged; `frontend-chat-layer1` promoted.
- **There is no `openspec` CLI, no `.github/`, and no CI in this repo.** Every gate is human-run; this change promises no `openspec validate`.

## Success criteria — traceability to the seven questions

| # | Question (`0005:219-225`) | Closed when | Evidence at verify |
|---|---|---|---|
| 1 | Conversation / turn / message / participant, and for each the Layer 2 term it maps onto — or, where Layer 2 supplies none, the Layer 1 term it inherits or the gap that forced a coinage | Every noun is in exactly one of **three** blocks — mapped (with its `VL2-*` id), coined (with the gap that forced it), or inherited (with its Layer 1 `V-*` id and its citation) | `decision.md` § vocabulary; `CHT` scenario asserting no noun is unmarked |
| 2 | Every AG-23 seam's v1 answer and injection point | All **11** appear (8 frozen + 3 experimental); none carries only one half | `decision.md` § seams walked against `archive/2026-08-21-…/decision.md:31-53` |
| 2b | *(D-2 disclosure)* | The gap-findings section is present and separately labelled | `decision.md` § gap findings |
| 3 | Archetype name, package path, composition root | `backend/agent/src/chat/` and `backend/agent/src/cmd/chat/` are named, with the fact that neither exists yet stated plainly | `decision.md` § identity, citing `0005:8` |
| 4 | Database, and whose tables | The **owner** is named, not the intent | `decision.md` § persistence, citing ADR 0009 § D6 (`:152`, quoted sentence `:154-155`) |
| 5 | Which frontend attaches, and what of its wire is frozen | `frontend-chat-layer1` REQ-1..REQ-7 enumerated as inherited-and-not-this-document's-to-design; the guard chain cited at its **correct** location | `decision.md` § frontend; F-3 in the register |
| 6 | What is out of v1, and the seam each deferral attaches to | Every deferral row carries an attaching seam; none is a bare exclusion | `decision.md` § deferrals against `0005:997-1011` |
| 7 | Which artifacts still say "a Layer 3 application", and the substitution rule | The eight occurrences, across three promoted specs, are listed with the matching rule they were derived under, and ADR 0009 § D7(a) is quoted | `decision.md` § substitution, citing `0009-…md:174-176`; F-2 in the register |
| 8 | *(D-1 gate)* Nothing regressed | Uncached green run, wall-clock recorded | `cd backend/agent && go test -race -count=1 ./...` |
| 9 | Bookkeeping | CH-00.1 row ticked; status line reads **1 of 12** | `docs/architecture/milestones/0005-…md` |

## Proposal question round (auto mode — recorded, not asked)

Execution mode is `auto`, so these product forks were decided from the milestone charter and the binding decisions rather than asked. Each is stated so a reviewer can overturn it deliberately:

1. **Is a record with no executable guard enough to bind twelve later milestones?** Decided *yes* (D-1) — CH-00.1 is typed `[decision]` and the package it would guard does not exist. If the maintainer prefers a mechanical binding, the earliest honest home is CH-01.2's import guard, not here.
2. **Should CH-00 repair the three citation defects it finds?** Decided *no* (D-3) — two are explicitly out of scope by charter, and the third belongs to a promoted spec this change does not own. If refused, F-1 and F-3 become in-scope and the change grows two spec deltas it currently has none of.
3. **Are "conversation" and "participant" coinages or mappings?** Decided *coinages* — Layer 2 defines no term for either, and `VL2-COR-14` "the delegated participant" is a different sense (subagent). If overturned, the record would have to claim a mapping the register does not support.
