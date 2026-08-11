# PRD 0001 — The cachicamas delivery loop

> **Status:** Draft — awaiting review. Consumed by no milestone document yet.
> **Decisions** live in [ADR 0008](../adr/0008-adopt-the-cachicamas-delivery-loop.md); this document states *what must be true*, never *which option was picked*.
> **Downstream:** this PRD is the Phase-0 intake source for `.claude/skills/task-graph-milestone-doc`. It produces `docs/architecture/milestones/0005-cachicamas-delivery-loop-task-graph.md`, prefix **`DF-`**.
> **Plan structure is not in scope here.** ADR 0007 owns the DAG contract; the skill owns the PRD→milestones method. This PRD must not restate either.
> **Date:** 2026-08-10
> **Numbering:** `docs/prd/` opens its own append-only sequence. The milestone document that consumes this PRD takes the next number in the `docs/architecture/` stack sequence (0005).

> [!IMPORTANT]
> **Authoring constraint.** Every claim about the repository in §2 and §3 was verified by command at the revision cited. No type name, field name, signature, or framework call is invented: the DDL in §7 is a *shape under review*, not a migration. Migrations are authored under Strict TDD inside an SDD change targeting `backend/database_administrator/`, per `openspec/AGENTS.md`.

---

## Outcome first

Walking every requirement in this document to green produces a delivery loop in which **a typed node from a milestone document is taken to a merged PR by agents, with the human present at exactly six decision points and absent everywhere else**, and in which every state transition carries falsifiable evidence bound to an immutable commit — so that "this milestone is done" is a query, not a claim.

---

## Contents

| § | What it settles | One-line takeaway |
|---|---|---|
| [1](#1-problem) | Why the current loop does not hold | Two heavy pipelines, one dead schema, zero enforcement |
| [2](#2-sources-and-research) | What the evidence says | Ticket-shape and review-shape transfer; ceremony does not |
| [3](#3-inconsistency-register) | What the repo contradicts about itself | Four verified contradictions, all blocking |
| [4](#4-scope-boundary) | Owns / must not own | The loop owns execution; ADR 0007 keeps the plan |
| [5](#5-the-delivery-loop-normative) | The loop itself | Lanes by node type, six human gates, bounded correction |
| [6](#6-requirements) | The contract | R-DF-001 … R-DF-092 |
| [7](#7-data-requirements) | What the database must hold | Projection (P) and execution state (S), never both |
| [8](#8-skill-inventory) | What agents load | Small model-invoked primitives, thin user-invoked orchestrators |
| [9](#9-non-goals) | What this is not | No auto-merge, no plan authoring in the database |
| [10](#10-risks) | What will hurt | Two parsers, orphaned state, silent skill drop |
| [11](#11-success-criteria) | How we know it worked | Six falsifiable outcomes |

---

## 1. Problem

cachicamas has **three plans and no loop**.

The plan is in good shape. `docs/architecture/milestones/0002–0004` hold 91 milestones across 22 waves, governed by a real machine-readable contract (ADR 0007) with a working validator (`scripts/validate_taskgraph.py`, 8 checks). Layer 1 closed 42 of 42. That machinery is not the problem and this PRD does not touch it.

Everything downstream of the plan is the problem.

**The execution pipeline is too heavy.** The current SDD runs eight phases per change — explore → propose → spec → design → tasks → apply → verify → archive — producing five durable files per milestone on top of a milestone document section that already carries a Goal, a Deliverable, an Acceptance line, an Out-of-scope list, and 1–7 Gherkin scenarios. The Charter *is* a proposal. The Gherkin *is* a spec. Writing both is writing the same decision twice, in two places, with two chances to drift. The independently measured cost of this shape is roughly an order of magnitude: 33m30 of agent time and 2,577 lines of markdown against 8 minutes for the same feature built iteratively, with 3.5 hours of review against 24 minutes — and the ceremonious run still shipped a bug.[^eberhardt]

**Nothing closes the loop.** This is the single most-documented failure of the lightweight alternative too. Matt Pocock's `implement` skill ends at a commit: it never closes the ticket, never ticks an acceptance box, and never acts on the findings its own review produced. His documentation states the consequence plainly — *"the frontier never advances by itself."* Any loop cachicamas adopts must answer *who advances the state, on what evidence* before it answers anything else.

**The state has nowhere to live.** `backend/database_administrator` carries a `project → requirement → milestone → task → spec → spec_phase` chain that models this pipeline almost exactly — `spec_phase` already enumerates `tdd_red, implementation, tdd_green, verify, pr, technical_ai_review, ai_approved, human_approved`. It has **zero Go code**: no domain type, no repository, no service, no HTTP route. Nothing can have written a row. Worse, it cannot hold the plan even if it did: `milestone` is declared

```sql
CREATE TABLE milestone (
    requirement_id  BIGINT  PRIMARY KEY REFERENCES requirement(id),
```

— the primary key **is** the foreign key, so one requirement admits exactly one milestone, permanently. Doc 0002 has 42 under one PRD. There is also no wave, no node type, and no dependency-edge table anywhere in the schema, so `Depends on:` is unrepresentable.

**There is no enforcement.** `.github/workflows/` does not exist. Every gate in both pipelines is a promise an agent makes about itself.

The loop this PRD specifies is not a third pipeline. It **deletes** the duplicated planning artifacts, keeps the milestone document as the single authored plan, gives execution state a real home, and puts the human at the decisions that are actually theirs.

---

## 2. Sources and research

Research digest. Each finding names what it changed in this document; a finding that changed nothing is recorded as such.

### 2.1 What transfers from Matt Pocock's skills

Read at `mattpocock/skills@main`, tree `84fdeff`, plugin 1.2.3.

| Finding | Takeaway | Changed |
|---|---|---|
| **Tracer-bullet vertical slices**[^tickets] — *"cuts a narrow but COMPLETE path through every layer… demoable or verifiable on its own… sized to fit in a single fresh context window"* | The sizing rule is a property of the agent, not the language | Nothing. cachicamas already sizes by the 400-line review budget, which binds harder and is measurable. Recorded as a rejected alternative in §5.2 |
| **Expand–migrate–contract is the explicit exception** to vertical slicing, for changes whose blast radius means *"no vertical slice can land green"* | A pipeline that assumes every unit is a vertical slice cannot land a wide refactor | **R-DF-012.** Added a third ingress type |
| **`implement` is nine lines** and has no completion step | Thin orchestrators work; the completion problem is not solved by making them thinner | **R-DF-047, R-DF-045.** Transitions are written by the orchestrator against typed evidence, not by the implementing agent |
| **`tdd`: "No test is written at an unconfirmed seam."** | The seam gate is the one rule that structurally requires a human, and it is cheap | **R-DF-005.** Promoted to a named gate |
| **The refactor step was deleted in June 2026** *"because agents essentially never performed it"* | It was deleted for being unobservable, not for being wrong | **R-DF-006.** Kept, with a falsifiable definition it never had |
| **Two-axis review** — Standards and Spec as parallel sub-agents, findings presented side by side and explicitly *not merged or reranked* | The value is the refusal to rank across axes; merging destroys the axis information | **R-DF-020…022.** Adopted, with the axes redefined as inputs |
| **`code-review` documented defects**: recursive fan-out (*one report reached 50-plus agents*), self-review bias (*"Same context reviewing itself isn't review, it's confirmation bias with a slash command"*), no convergence (*"do not run it in a loop until it comes back clean, because it will not"*), findings never re-verified | Every one of these is a design requirement in disguise | **R-DF-023, R-DF-025, R-DF-027** |
| **Prose delegation does not reliably load the target skill** — documented, unfixed, *fails silently and partially* | A mandatory gate invoked by a sentence is not a gate | **R-DF-029.** Mandatory gates are tool calls with receipts |
| **`grilling`**: frontier rounds; terminates on empty frontier **and** explicit human confirmation | Two termination gates because the first is model-judged | Informs **R-DF-025**'s terminal states |
| **CONTEXT.md is a glossary and nothing else**; three-gate ADRs (hard to reverse · surprising · a real trade-off) | Kills most ADR ceremony | Nothing here; recorded for the skills work |

### 2.2 What the ecosystem says

| Source | Finding | Changed |
|---|---|---|
| Anthropic Agent Skills spec[^skillspec] | `name` 1–64 chars, must match directory, no `anthropic`/`claude`; `description` 1–1024; progressive disclosure at three levels; keep SKILL.md under 500 lines; scripts' code never enters context, only output | **R-DF-081** |
| Claude Code skills reference[^ccskills] | `disable-model-invocation` removes the description from context entirely; a project skill named `code-review` collides with the built-in `/code-review`, which is diff-only and posts inline comments | **R-DF-028, R-DF-082** |
| Jesse Vincent[^fsck] | Skill/command descriptions are silently dropped past a ~15,000-character budget; *"there's no warning when you go over"*; community trigger rate ≈50% | **R-DF-082** |
| Colin Eberhardt[^eberhardt] | Spec Kit: 33m30 + 2,577 md lines + 3.5h review vs 8min + 24min iterative; ~10× slower and still shipped a bug | Frames §1 and the honest accounting in §5.6 |
| Eretz Kdosha[^ranthe] | Spec-Kit *"produced code didn't faithfully map to spec intent"*; OpenSpec *"sometimes assumes context and adds rationale to decisions you didn't make"*; BMAD twelve agents, shared output directory | Reinforces **R-DF-021** (spec conformance must be enumerated, not asserted) |
| GitHub Spec Kit / Amazon Kiro | Both keep the plan in files and drive it from files; Spec Kit makes tests **optional** | Rejected. cachicamas' Strict TDD and ADR 0007 are stronger and already shipped |

### 2.3 What research changed nothing

Ticket sizing by context window (§2.1 row 1); the five-field ticket schema — it has no slot for **Out of scope**, which doc 0002 calls the field that keeps siblings mutually exclusive; `CONTEXT.md` — the milestone documents' *Layer boundary* sections already carry the vocabulary and the wording traps.

---

## 3. Inconsistency register

Four contradictions between the repository and its own documents. Each was verified by command. All four block the first milestone that touches the affected area.

| # | Conflict | Both sides | Disposition |
|---|---|---|---|
| **I-1** | Five migrations carry the comment `Forward-only migration per openspec/AGENTS.md`. `grep -in "migration\|forward-only\|goose" openspec/AGENTS.md` returns **nothing**. Meanwhile the archived `witsaba-core-tables` spec mandates the opposite: *"Down blocks reverse the corresponding Up blocks"*, *"Down then Up cycle is byte-identical"* | `src/migration/sql/2026070812*.sql`, `…0715*`, `…0717*` vs `openspec/AGENTS.md` vs `openspec/changes/archive/2026-06-22-witsaba-core-tables/specs/witsaba-core-tables/spec.md` D1/D2 | **Flagged to user — blocking.** A reviewer applying the written spec rejects an empty Down block; a reviewer applying the folklore rejects a populated one. Both are citable. Settled by **R-DF-091** before the first new migration |
| **I-2** | `openspec/specs/witsaba-core-tables/` does not exist. The spec governing `milestone`, `task`, `spec`, `spec_phase` (49 requirement sections) was never promoted out of its archive folder, unlike `db-migrations`, `identity-schema`, `prompts`, `workspaces` | `ls openspec/specs/` (53 capabilities, absent) vs `openspec/changes/archive/2026-06-22-witsaba-core-tables/` | **Flagged to user — blocking.** A delta spec would be written against a capability the source of truth does not contain: the requirements this PRD rescinds would never appear as rescinded. Settled by **R-DF-090** |
| **I-3** | `judgment-day` presents "two blind judges" as its corroboration mechanism. `diff ~/.claude/agents/jd-judge-a.md ~/.claude/agents/jd-judge-b.md` shows **four changed lines**: the name, one letter in the description, one letter in the body, and `effort: medium` → `effort: high` | `~/.claude/agents/jd-judge-{a,b}.md` vs `~/.claude/skills/judgment-day/SKILL.md` Hard Rule *"Fix only severe findings confirmed by both judges"* | **Reconciled by design.** This is one reviewer sampled twice at two budgets. Agreement measures prompt determinism, not correctness — and under the both-must-confirm rule, correlated judges make dual confirmation nearly free while feeling like corroboration. **R-DF-020** differentiates by *input*, not by effort |
| **I-4** | `openspec/config.yaml` declares `test_command`, `build_command`, and a 400-line review budget as gates. `.github/workflows/` does not exist. Doc 0002 concedes *"recorded green output means output pasted into the PR, not a check mark from a runner"* | `openspec/config.yaml` vs `ls .github/workflows` | **Reconciled.** An eleven-gate loop with zero automated enforcement points is a loop of promises. **R-DF-092** makes three gates falsifiable for less cost than any single stage |

---

## 4. Scope boundary

### The delivery loop owns

Execution: the lane a node runs in; the order of stages within a lane; the evidence that closes a stage; the claim and lease protocol that lets sessions work concurrently; the review protocol and its adjudication; the placement of every human gate; the projection of a milestone document into queryable state; the persistence of everything that happened.

### The delivery loop must not own

| Concern | Owner |
|---|---|
| The DAG contract — node grammar, edge declaration, acyclicity, palette | ADR 0007 |
| The PRD→milestones method — intake, research, waves, Gherkin authoring, validation | `.claude/skills/task-graph-milestone-doc` |
| The *content* of any plan — which waves exist, which milestones, which scenarios | The milestone document, authored by a human with the skill |
| Layer 1/2/3 architecture and its seams | `docs/architecture/0001` and ADRs 0004–0006 |
| Merge authority | The human. Always |

### Wording traps

1. **"Feature" is not a level.** The containment tree is document → wave → milestone → node, with nodes nesting up to three deep (ADR 0007 D2.3 as amended 2026-08-10). A `[compound]` node *is* what "feature" was reaching for, and it is a type tag, not a table. Doc 0002 already states the mapping in its own words: *"the leaves of that milestone's graph become its `tasks.md` phases."* Node ≡ task. A third name for it would be the eighth level of a seven-level tree. **The word "feature" does not appear in the loop.**

2. **"Projection" is not "storage".** When this document says the database holds waves and milestones, it means *derived rows rebuildable from the markdown and discardable without information loss*. It never means a place where a plan is authored. §7.1 states the invariant that makes the difference checkable rather than aspirational.

---

## 5. The delivery loop (normative)

### 5.1 Shape

```
PRD  ──(task-graph-milestone-doc skill, unchanged)──▶  milestone document
                                                       waves → milestones → typed nodes → Gherkin scenarios
                                                              │
                                          project (derived)   ▼
                                                       ┌──────────────┐
                                                       │   frontier   │  nodes whose Depends-on are all closed
                                                       └──────┬───────┘
                                                              │  lane selected by node type
              ┌───────────────────────────────┬───────────────┴──────────────┬─────────────────────────┐
              ▼                               ▼                              ▼                         ▼
        [leaf] / [guard]                 [decision]                    [mechanical]              [compound]
          TDD lane                      decision lane                 evidence lane          never worked directly
              │                               │                              │
   per scenario:                    research → recorded artifact     perform → record
   ① seam confirmed ⟨human⟩              → human sign-off ⟨human⟩      objective evidence
   ② RED  (scenario verbatim, one test)       │                              │
   ③ implementation (minimum)                 │                              │
   ④ GREEN (test + evidence gate)             │                              │
   ⑤ REFACTOR (separate commit,               │                              │
      empty test diff, green both ends)       │                              │
              └───────────────────────────────┴──────────────┬───────────────┘
                                                             ▼
                                                     ⑥ PR opened
                                              candidate frozen: base_sha · head_sha · diff_sha256
                                                             ▼
                                    ⑦ two judges, differentiated by INPUT, in parallel
                                 ┌───────────────────────┴───────────────────────┐
                          Judge-SPEC                                       Judge-DIFF
                  scenarios + charter + neighbours                   base…HEAD + repo standards
                  forbidden the diff on pass 1                       forbidden the spec
                  output: scenario → proving test,                   output: violations and smells,
                          + every UNPROVEN scenario                          hard vs judgement call
                                 └───────────────────────┬───────────────────────┘
                                                         ▼
                                              ⑧ adjudication (bounded)
                                  both approve ──────────────────▶ ai_approved
                                  both request the same change ──▶ verify → 1 correction round (max 2)
                                  they disagree ─────────────────▶ escalate ⟨human⟩
                                  rounds exhausted ──────────────▶ escalate ⟨human⟩
                                                         ▼
                                              ⑨ human merges ⟨human⟩
```

### 5.2 The unit of work

**One node. One PR. Sized by the 400-line review budget** (`openspec/config.yaml`: prefer under 250, reassess before 400, chain when risk is High).

The rejected alternative is Pocock's *"sized to fit in a single fresh context window."* A 400-line Go diff is nowhere near a context window, and a `[decision]` node can exhaust one while producing zero lines. The line budget is the *review* budget, which is the thing that actually binds, and it is measurable at the moment it matters.

### 5.3 Lanes

The node type is not decoration — **it selects the lane**, and doc 0002's *How it closes* column already specifies each one. This matters immediately: Layer 2 opens with AG-00, AG-01 and AG-02, and AG-01/AG-02 are `[decision]` nodes whose deliverable is *"a recorded decision, no production code."* A pipeline that runs `Specs → TDD Red → Implementation` cannot execute the first two things Layer 2 needs.

| Type | Lane | Closes on |
|---|---|---|
| `[leaf]` | TDD | every scenario red → green → refactored |
| `[guard]` | TDD | the bite shown red against a scratch violation, then green |
| `[decision]` | Decision | recorded artifact answering every closing-checklist item, **human sign-off** |
| `[mechanical]` | Evidence | recorded objective evidence; no red/green |
| `[compound]` | — | children closed + one-line exit check. **Never worked directly** |

### 5.4 Ingresses

A loop with one entrance is a loop that cannot receive a bug.

| Ingress | Trigger | Path |
|---|---|---|
| **planned** | A node on the frontier | §5.1 as drawn |
| **defect** | A failure discovered after merge | Failing test **first**, no spec, no new milestone. The regression assertion is appended to the owning leaf's scenario list carrying doc 0002's existing `*(pin)*` marker — green-from-birth, exempt from red-first. The PR cites the node that owned the behaviour |
| **wide refactor** | One mechanical change whose blast radius means no vertical slice can land green | A `[mechanical]` node with a Check list and **three PRs**: expand (add the new form beside the old), migrate (batches sized by blast radius, each blocked by expand), contract (delete the old form, blocked by every batch). Not a tracer bullet, and not forced into one |

### 5.5 Re-entry: the plan was wrong

The straight line has exactly one backward arrow that is not a correction round: **graph amendment**. Doc 0002's revert-and-record clause is normative and Layer 1 used it 35 times.

Revert to green → append the prerequisite node at the next free ordinal → draw the edge → move remaining scenarios into children if the leaf became compound → land the amendment **in the same PR that resumes work**. `validate_taskgraph.py` must re-pass with zero errors and no cycle *in that PR*, and the PR body must cite the disproved node id.

This gate is **human-approved without exception**. An agent free to append nodes can plan its way out of any hard problem.

### 5.6 The six human gates, and the honest accounting

| # | Gate | Why it cannot be delegated | Cost if skipped |
|---|---|---|---|
| 1 | **Seam confirmation** before the first RED | *"No test is written at an unconfirmed seam."* Writing the test first silently decides the seam | The revert-and-record clause fires later, at maximum cost |
| 2 | **`[decision]` sign-off** | AG-01.1 alone carries five open checklist questions, each with a documented default an agent will rubber-stamp | A whole layer's vocabulary decided by acquiescence |
| 3 | **Before round-one correction** | Authorising a fix is authorising the diagnosis | Agents fix the wrong thing confidently |
| 4 | **Graph amendment** | §5.5 | Plan drift with no reviewer |
| 5 | **Judge contradiction** | There is no tiebreaker that is not a third correlated opinion | An unresolved disagreement resolved by whoever spoke last |
| 6 | **Merge** | Repo policy: no auto-merge, ever | — |

Gates 1–5 are cheap before code and expensive after. Gate 6 is the only one that exists today.

**Is this lighter than SDD? In artifacts yes, in gates no, and the claim is only true if the deletion in §5.7 actually happens.** Stated honestly: the loop has eleven stages (two of them parallel) against SDD's eight phases. It adds a review stage SDD never had. What it removes is duplication, and only that.

### 5.7 What is deleted

The saving comes from one named place, not from being vague about ceremony.

| Deleted | Because |
|---|---|
| `proposal.md` | The milestone Charter **is** the proposal: Goal, Deliverable, Acceptance, Depends on, Blocks, Out of scope, Notes |
| `specs/<capability>/spec.md` per change | The node's Gherkin **is** the spec. Scenarios are already Given/When/Then and already 1:1 with tests |
| `tasks.md` | The node's scenarios **are** the task list, in order, with the walking skeleton first |
| `archive` as a phase | Replaced by ticking the node's completion-checklist box and recording the merge |
| `verify-report.md` as a separate artifact | Replaced by typed evidence on the phase event: command, exit code, output hash |

**Kept:** `design.md`, and only where it *is* the deliverable — a `[decision]` node's recorded artifact. Kept: `openspec/specs/` as the promoted, living capability register. That register is the durable memory of what the system guarantees; the per-change delta was the duplication.

Net per milestone: **one durable artifact (the milestone-document section, already written) plus the PR.**

### 5.8 The two judges

They are not two opinions. They are **two inputs**.

| | Judge-SPEC | Judge-DIFF |
|---|---|---|
| Sees | the node's Gherkin scenarios, its Charter Acceptance line, its `Depends on:`/`Blocks:` neighbours | `git diff <base_sha>...<head_sha>` and the repo's documented standards |
| **Does not** see | the diff, on the first pass | the spec |
| Deliverable | an enumeration: *scenario → the test that proves it*, and **every scenario with no proving test, listed explicitly** | violations, each labelled *hard* (documented standard breached) or *judgement call* (baseline smell). A documented repo standard overrides the baseline |
| Catches | omission | commission |

This split exists because **"only the lines changed in the PR" is blind to omission by construction.** A requirement never implemented produces zero changed lines. Concretely: AG-02.1 demands one verdict per register row across eight G-concerns; implement seven and the diff contains nothing about the eighth — a diff-only reviewer passes it. The same blindness covers a deleted caller outside the diff, a downstream `Blocks:` whose assumption you broke, an absent test file, and — worst here — the `[guard]` bite, which doc 0002 closes on *"the recorded red run against the scratch violation"*, a run that happens in a scratch tree that never enters any diff.

Diff-only is one axis of two. It must be the second one.

**Constraints on both judges.** Read-only. No `Agent` tool, no delegation — this is what closes the 50-agent fan-out, and it must not be "improved" by granting it back. Verdicts bind to the frozen candidate `(base_sha, head_sha, diff_sha256)`; a force-push mints a new candidate and structurally strands the old verdicts with no invalidation logic anywhere. Findings are hypotheses until reproduced: **one verification actor with execution rights reproduces every severe finding before any fix is authorised.**

**Adjudication is bounded.** At most two correction rounds; re-judgment sees only the frozen ledger plus the fix delta — that clause, not the round count, is what forces convergence. Terminal states are `approved` and `escalated`, nothing else; an exhausted lineage is never reset or extended. Disagreement escalates to the human. **A third judge is never added** — a third clone contributes majority-of-correlated-errors, not information.

**Fixed point under chained PRs.** For PR 3 of a chain, `main...HEAD` contains PRs 1 and 2, so judges re-review approved code and manufacture new findings. The fixed point is the **chain parent branch**, recorded as `base_sha` on the candidate. If a correction diff pushes past 400 lines, that is a `chained-pr` trigger, not another round.

### 5.9 Refactor, made falsifiable

Pocock deleted this step because agents never performed it. It was unobservable. Here it is observable:

> A refactor is **a separate commit whose test-file diff is empty**, with the full suite green at both ends.

`git diff <impl-commit>..<refactor-commit> -- '*_test.go'` must be empty. Scenarios never change during refactor. The definition also stops the refactor hiding inside the feature diff, where neither judge can separate restructuring from behaviour change.

Refactor is a **per-scenario** stage, not a per-milestone one: doc 0002 already says *every test-list item taken red → green → refactored*.

---

## 6. Requirements

RFC 2119 keywords. IDs are append-only and never reused.

### 6.1 The loop

- **R-DF-001** — The markdown milestone document MUST remain the only authored source of plan structure. No other artefact may be edited to change a wave, a milestone, a node, a node type, an edge, or a scenario.
- **R-DF-002** — The unit of work MUST be one ADR-0007 typed node, delivered by one PR, sized against the 400-line review budget. The loop MUST NOT introduce a level between milestone and node.
- **R-DF-003** — The node's `[type]` tag MUST select its lane per §5.3. A node whose type has no lane MUST be rejected, not defaulted.
- **R-DF-004** — In the TDD lane, each scenario MUST be driven RED → implementation → GREEN → refactor as exactly one test, in the order the scenarios appear, the walking skeleton first.
- **R-DF-005** — Before the first RED of a node, the agent MUST state the exact surface it will test against and obtain human confirmation. No test may be written at an unconfirmed seam.
- **R-DF-006** — A refactor MUST be a separate commit whose test-file diff is empty, with the evidence-gate command green at both ends. A refactor MUST NOT modify any scenario.
- **R-DF-007** — A `[decision]` node MUST close on a recorded artifact answering every closing-checklist item, plus human sign-off. It MUST NOT produce production code and MUST NOT be given Gherkin.
- **R-DF-008** — A `[mechanical]` node MUST close on recorded objective evidence and MUST NOT be driven red/green.
- **R-DF-009** — A `[compound]` node MUST NOT be claimed or worked directly.
- **R-DF-010** — The loop MUST accept three ingress types — *planned*, *defect*, *wide refactor* — per §5.4, and MUST record which one produced each PR.
- **R-DF-011** — A *defect* ingress MUST begin with a failing test, MUST NOT create a milestone or a spec, and MUST append its regression assertion to the owning leaf's scenario list marked `*(pin)*`.
- **R-DF-012** — A *wide refactor* MUST be sequenced expand → migrate → contract as separate PRs with declared blocking edges, and MUST NOT be forced into a vertical slice.
- **R-DF-013** — When execution disproves the plan, the loop MUST revert to green and land the graph amendment in the same PR that resumes work; `validate_taskgraph.py` MUST re-pass with zero errors in that PR; the amendment MUST be human-approved.
- **R-DF-014** — The frontier MUST be computed from `Depends on:` and `Blocks:` only. `Parallel with:` MUST NOT be read as an edge.

### 6.2 Review

- **R-DF-020** — The two judges MUST be differentiated by input per §5.8. Reasoning effort MUST NOT be used as the differentiator.
- **R-DF-021** — Judge-SPEC MUST be denied the diff on its first pass and MUST deliver a scenario → proving-test enumeration that lists every scenario with no proving test.
- **R-DF-022** — Judge-DIFF MUST review `base_sha...head_sha` against the repo's documented standards and MUST label each finding *hard* or *judgement call*. A documented repo standard MUST override any baseline heuristic.
- **R-DF-023** — Judges MUST be read-only and MUST NOT have delegation capability.
- **R-DF-024** — Every verdict MUST bind to an immutable candidate identified by `(base_sha, head_sha, diff_sha256)`. A verdict MUST NOT be reusable across candidates.
- **R-DF-025** — Adjudication MUST permit at most two correction rounds; re-judgment MUST see only the frozen ledger and the fix delta; terminal states MUST be exactly `approved` and `escalated`; an exhausted lineage MUST NOT be reset or extended.
- **R-DF-026** — Judge disagreement MUST escalate to the human. The system MUST NOT add a third judge and MUST NOT auto-resolve.
- **R-DF-027** — A severe finding MUST be reproduced by an actor with execution rights before any fix is authorised.
- **R-DF-028** — No skill in this system may be named `code-review`.
- **R-DF-029** — Judges MUST be launched by tool call, and the transition to `ai_approved` MUST carry both verdict identifiers as evidence. A gate invoked only by prose MUST NOT be treated as run.
- **R-DF-030** — For a chained PR, the review fixed point MUST be the chain parent, recorded as `base_sha`.

### 6.3 Projection and state

- **R-DF-040** — **The projection invariant.** Every column in the planning schema MUST be exactly one of: **(P)** a pure projection of a byte range of a source document identified by `(path, content_sha256)`, rebuildable by re-running the parser and discardable without information loss; or **(S)** execution state with no representation in any document. No column may be both.
- **R-DF-041** — Exactly one implementation of the ADR-0007 grammar MUST exist. `validate_taskgraph.py` MUST emit the canonical machine-readable projection, and the service MUST ingest that output rather than re-parsing markdown.
- **R-DF-042** — Ingest MUST be an idempotent full replacement of (P) rows for the document, and MUST refuse to run when the document's `content_sha256` does not match the revision under review.
- **R-DF-043** — There MUST be no write path from the database back to a source document, and no human- or agent-facing update on any (P) row.
- **R-DF-044** — Scenario identity MUST be `(node_ref, ordinal)`. `gherkin_sha256` MUST be stored as a change detector, never as the key. When a scenario's text changes while its state is at or past GREEN, the system MUST record the change and force re-verification; it MUST NOT silently rebind and MUST NOT silently discard.
- **R-DF-045** — Phase state MUST be an append-only event log. Every event MUST carry the actor, the claim epoch it was written under, and typed evidence.
- **R-DF-046** — Evidence MUST be typed per phase, at minimum: GREEN and refactor ⇒ `{command, exit_code, output_sha256}`; PR ⇒ `{pr_url, base_sha}`; review ⇒ `{verdict_ids[], round}`; human approval ⇒ `{reviewer, merge_sha}`. A transition missing its required evidence keys MUST be rejected.
- **R-DF-047** — Transitions MUST be written by the orchestrator. An implementing agent MUST NOT write a transition about its own work, and MUST NOT be able to write a human-approval event.
- **R-DF-048** — A claim MUST combine a lease with a monotonically increasing fencing epoch. Every subsequent write MUST be guarded on the epoch it claimed under; a write under a stale epoch MUST affect zero rows and the agent MUST abort rather than retry.
- **R-DF-049** — Agents MUST reach this state over HTTP. The agent module cannot import the database service (`src/domain/imports_test.go`, ADR 0005 § D1), so claim, lease, fencing and idempotency MUST be expressed in the HTTP contract.
- **R-DF-050** — Every agent write MUST be idempotent under retry via a caller-supplied key scoped to the entity, never a global key.
- **R-DF-051** — Acyclicity MUST be enforced primarily by the validator, where the error names the offending ids to a human. The database MAY carry a deferred backstop; if it does, concurrent edge writes MUST be serialised by a document-scoped advisory lock, because two individually-acyclic edges committed concurrently can be jointly cyclic.

### 6.4 Schema

- **R-DF-060** — The unimplemented `milestone`/`task`/`spec`/`spec_phase` chain MUST be retired in one migration guarded by a row-count assertion that aborts if any row exists. Expand–migrate–contract MUST NOT be used: it protects live readers, and there are none.
- **R-DF-061** — Waves, milestones and nodes at every depth MUST live in one containment table with a self-referencing parent, because ADR 0007 nests nodes three levels and a fixed table-per-level cannot express recursion.
- **R-DF-062** — Dependency edges MUST live in exactly one table with two real foreign keys, stored in one canonical direction (prerequisite → dependent) with `Blocks:` normalised at ingest. A polymorphic `(kind, id)` edge table MUST NOT be used. A single `depends_on` column MUST NOT be used — ADR 0007 measured 68 of 90 milestones with more than one dependency parent.
- **R-DF-063** — Scenarios MUST be rows, not text inside a content column, because the scenario is the unit of claim, of phase state, and of agent work.
- **R-DF-064** — Every (P) row MUST be traceable to its source document, that document's `content_sha256`, and the parser version that produced it.
- **R-DF-065** — The phase vocabulary MUST be data, not a CHECK constraint, so adding a phase is an insert rather than a lock-taking migration.
- **R-DF-066** — A pull request MUST be identified by `(provider, repo, number)` and never by its state; state and head SHA are mutable attributes.
- **R-DF-067** — Under squash- or rebase-merge the landed commit is a different object from the approved head. The system MUST record `merge_commit_sha` and `merge_method` as *provenance* and MUST NOT assert post-merge SHA equality.
- **R-DF-068** — "Which judge" MUST be data — model, provider, prompt revision — never a hard-coded enum, so a verdict is reproducible and a judge can be retired without a migration. The required number of concurring verdicts MUST remain configuration, not schema.

### 6.5 Skills

- **R-DF-080** — The skill set MUST follow the shape in §8: small model-invoked primitives, thin user-invoked orchestrators.
- **R-DF-081** — Every skill MUST conform to the Agent Skills spec: `name` 1–64 chars matching its directory, no reserved words, `description` ≤1024 chars, SKILL.md under 500 lines, detail behind `references/`.
- **R-DF-082** — The combined skill-description budget MUST be monitored. Descriptions are dropped silently past ~15,000 characters, and a dropped skill is unavailable with no warning. Any skill whose failure is invisible MUST be user-invoked or tool-launched, never left to autonomous triggering.

### 6.6 Governance

- **R-DF-090** — `openspec/changes/archive/2026-06-22-witsaba-core-tables/specs/witsaba-core-tables/spec.md` MUST be promoted verbatim to `openspec/specs/witsaba-core-tables/spec.md` **before** any delta touches these tables, so the rescinded requirements appear as rescinded. *(Resolves I-2.)*
- **R-DF-091** — The migration reversibility rule MUST be settled in `openspec/AGENTS.md` and the promoted spec's D1/D2 amended or rescinded to match, before the next migration is written. No further migration may cite a rule that does not exist. *(Resolves I-1.)*
- **R-DF-092** — CI MUST run, on every pull request, at minimum: the per-module test command, the per-module lint command, and `validate_taskgraph.py` over every milestone document. *(Resolves I-4.)*

---

## 7. Data requirements

Target: `backend/database_administrator`. This section states the **shape and the reasoning**. The migrations themselves are authored under Strict TDD inside an SDD change, after R-DF-090 and R-DF-091 are settled.

### 7.1 The invariant that makes this not a second source of truth

ADR 0007 D3 rejected a structured sidecar as *"a second source of truth that drifts from the prose within one amendment."* A database table is a *worse* sidecar than a YAML file — a YAML file at least appears in the PR diff next to the prose it drifted from. This tension is real and must be resolved on the record, not papered over.

The resolution is ADR 0007's own words about renderings: *"Neither rendering is authoritative… the field wins and the rendering is regenerated."* **A projection is a rendering.** What makes it one, rather than a sidecar, is R-DF-040's column partition plus three mechanical consequences:

1. (P) rows are deleted and reinserted wholesale on reparse. They hold no information that would be lost.
2. Ingest refuses to run when the document hash does not match the revision under review, so drift is a hard failure rather than a silent divergence.
3. There is no write path back to the document, and no update path on a (P) row.

**Honest cost:** a parser, a reprojection job, a staleness gate, and inherited state-orphaning (§10). The alternative — agents committing execution state back into markdown — makes git merge conflicts the concurrency primitive, with no lease mechanism at all. That is worse.

This resolution lands as a dated amendment to ADR 0007 D3, authored alongside ADR 0008, so the contradiction is on the record.

### 7.2 Shape

Three migrations, each a reviewable PR inside the 400-line budget.

**Migration 1 — retire and project.**

| Table | Holds | Why this shape |
|---|---|---|
| *(drop)* `spec_phase`, `spec`, `task`, `milestone` | — | Guarded by a `DO $$ … RAISE EXCEPTION` block asserting zero rows. The belief "no rows" is an inference from "no Go code"; the migration must prove it, not assume it, and abort loudly if wrong |
| `source_document` | `requirement_id`, `path`, `content_sha256`, `parser_version`, `projected_at` | (P) anchor. `content_sha256` is what makes drift detectable rather than silent. One PRD yields many documents — 0002/0003/0004 are three from one product intent — so a table, not columns |
| `plan_node` | `document_id`, `parent_id`, `node_kind ∈ (wave, milestone, node)`, `node_ref` (`'AI-40.2.1'`), `title`, `node_type ∈ (leaf, guard, decision, mechanical, compound)`, `ordinal`, `body`, `sdd_change_slug` | (P). **One table for all three levels.** Nodes nest three deep, so table-per-level cannot express the grammar — and unifying is what collapses the DAG onto one edge table with real foreign keys. A composite FK on `(parent_id, parent_kind)` plus a CHECK enforces the containment grammar in SQL instead of trusting the importer |
| `plan_edge` | `from_node_id`, `to_node_id`, PK on both | (P). Canonical direction only. `Depends on:` and `Blocks:` describe the same edge from two ends; a direction discriminator would store both and manufacture a spurious 2-cycle — the exact false positive ADR 0007 already warns about for `Parallel with:`. The PK does the deduplication |
| `scenario` | `plan_node_id`, `ordinal (1–7)`, `title`, `gherkin`, `gherkin_sha256`, `text_changed_at` **(P)** · `claimed_by_session_id`, `lease_expires_at`, `claim_epoch` **(S)** | The only table carrying both kinds of column, split explicitly. Lease and epoch live on the claimed row: a partial unique index cannot express liveness, because `now()` is not `IMMUTABLE` and a predicate referencing it fails at `CREATE INDEX` |

**Migration 2 — execution state.**

| Table | Holds | Why |
|---|---|---|
| `agent_session` | `kind ∈ (agent, human)`, `user_id` \| `model_id` + `provider` + `prompt_revision_id`, timestamps | (S). Answers *which agent* without an enum, and makes a green or a verdict reproducible. A CHECK makes the identity exclusive: a human session has a user, an agent session has a model |
| `phase` | `code` PK, `ordinal`, `is_terminal` | (S). Vocabulary as rows. The v1 CHECK already lacks `refactor` and `changes_requested`; every future phase would otherwise be a `DROP CONSTRAINT`/`ADD CONSTRAINT` migration taking `ACCESS EXCLUSIVE` |
| `phase_event` | `scenario_id`, `phase_code`, `session_id`, `claim_epoch`, `evidence`, `pull_request_id`, `notes`, `idempotency_key`, `occurred_at` | (S). **Events, not intervals.** Interval rows need an UPDATE to close, which breaks append-only and is exactly why v1 needed a "at most one open phase" index it then deferred. With events there is nothing to keep open — the invariant disappears instead of being enforced. Current phase is a `DISTINCT ON`. `UNIQUE (scenario_id, idempotency_key)` is the retry guard |

**Migration 3 — PR and review.**

| Table | Holds | Why |
|---|---|---|
| `pull_request` | `plan_node_id`, `provider`, `repo_full_name`, `number`, `url`, `state`, `head_sha`, `base_sha`, `merge_commit_sha`, `merge_method`, timestamps | Mutable by design. Identity is `(provider, repo, number)` and never involves state — GitHub reuses the number on reopen, so keying on state mints a duplicate |
| `review_candidate` | `pull_request_id`, `head_sha`, `base_sha`, `diff_sha256`, `changed_lines` | **Immutable — no update path, no `updated_at`.** This is the whole anti-reuse mechanism. `diff_sha256` is required *in addition* to both SHAs: when the base advances under an untouched branch the diff changes while `head_sha` does not, and SHAs alone miss it. The identity tuple doubles as the idempotency key |
| `judge` | `slug`, `model_id`, `provider`, `prompt_revision_id`, `retired_at` | Judges as data. Adding, re-prompting or retiring a judge is an insert, not DDL, and a verdict stays reproducible after retirement |
| `judge_verdict` | `candidate_id`, `judge_id`, `outcome ∈ (approve, request_changes, abstain)`, `findings`, `UNIQUE (candidate_id, judge_id)` | The unique constraint is simultaneously the disagreement model and the idempotency guard: a retried judge collides instead of double-voting, and a tie is two rows with different outcomes — **a query, never a stored consensus column**, which would be derived state that drifts |

Human tie-resolution needs no table: it is a `phase_event` written by a session whose `kind` is `human`. Findings stay a payload on the verdict until a finding needs its own lifecycle.

### 7.3 Deliberately not modelled

| Cut | Why |
|---|---|
| A `feature` table or level | Not in the ADR-0007 grammar. `[compound]` is the feature, and it is a type tag |
| `task`, `spec` as tables | Absorbed. `SDD change:` sits at the milestone heading per ADR 0007 D2.2, so one `sdd_change_slug` column replaces a table, an FK and an index |
| A separate `wave` table | Absorbed into `plan_node`. Its only distinct attributes — gate, exit condition — are prose |
| A `finding` table | JSONB until a finding needs its own resolution state |
| A consensus / `required_verdict_count` column | Policy, not schema. Derived state that drifts |
| A separate claim table | Two columns and an epoch on the claimed row. A claim table needs a liveness predicate a partial index cannot express |
| Materialised transitive closure | Premature at 91 milestones. A recursive CTE over an indexed `BIGINT → BIGINT` edge table is microseconds. Add when measured |
| `updated_at` on append-only tables | It would be a lying duplicate of `created_at` — the existing tables already carry it with no trigger maintaining it |

---

## 8. Skill inventory

Shape, from the most transferable architectural idea in the source material: **primitives are model-invoked and tiny; orchestrators are user-invoked and thin.** `grilling` is thirteen lines and is reached by five other skills; `implement` is nine.

| Skill | Invocation | Owns |
|---|---|---|
| `task-graph-milestone-doc` | user | **Exists. Unchanged.** PRD → milestone document |
| `df-frontier` | user | Compute and present the frontier; claim one node with a lease and epoch |
| `df-implement` | user | Run the node in its lane; drive the per-scenario cycle; open the PR |
| `tdd-seams` | model | The seam vocabulary, the confirmation gate, the three anti-patterns |
| `df-refactor` | model | The falsifiable refactor: separate commit, empty test diff, green both ends |
| `df-review` | user | Launch both judges by tool call, adjudicate, record verdicts. **Not named `code-review`** (R-DF-028) |
| `df-judge-spec` | agent | Scenario → proving-test enumeration; denied the diff on pass 1 |
| `df-judge-diff` | agent | `base...head` against documented standards; denied the spec |
| `df-amend-graph` | user | The revert-and-record path, human-approved |

Nine entries, four of them user-invoked and therefore costing zero description budget. Mandatory gates are tool calls, never sentences (R-DF-029): a gate invoked by prose that silently does not fire produces **no artifact to be missing**, which is the worst failure shape available.

---

## 9. Non-goals

| Not doing | Why |
|---|---|
| Auto-merge, or any agent merge authority | Repo policy, and the one gate that is never worth optimising |
| Authoring plans in the database | Directly contradicts ADR 0007 D2. The database projects; it never authors |
| A third judge, or a majority vote | A third correlated clone adds majority-of-correlated-errors, not information |
| Replacing `openspec/specs/` | The promoted capability register is the durable memory of what the system guarantees. The per-change delta was the duplication, not the register |
| Restructuring docs 0002–0004 | They are v1-profile and explicitly exempt. Validate with `--profile v1` |
| A UI for any of this | Out of scope. The loop must work headless first |
| Cross-repository orchestration | Single repository, single workspace |
| Replacing `judgment-day` | Its bounded-correction machinery is imported wholesale. What is replaced is its judge *differentiation*, which is broken (I-3) |

---

## 10. Risks

| Risk | Severity | Mitigation |
|---|---|---|
| **Two parsers diverge.** A Go importer re-implementing the ADR-0007 grammar will drift from `validate_taskgraph.py`; the module boundary forbids sharing code. CI goes green while the projection is structurally wrong | High | R-DF-041: the validator emits the canonical projection and the service ingests *that*. One grammar, one implementation, and the projection is reviewable in CI output |
| **Orphaned execution state.** Editing a document reshuffles (P) rows. Keying scenarios on ordinal means an inserted scenario shifts the rest and GREEN lands on the wrong one — silent corruption. Keying on the text hash discards a legitimate green over a typo fix — loud but wasteful | High | R-DF-044: identity is `(node_ref, ordinal)`, the hash is a detector. On change past GREEN, surface and force re-verification. **This is detection, not prevention.** Anyone claiming a clean answer here is wrong |
| **Skill descriptions dropped silently** past the budget; ~50% autonomous trigger rate | Medium | R-DF-029, R-DF-082: mandatory gates are tool calls; anything whose failure is invisible is user-invoked |
| **Correlated judges** re-emerge because differentiating by prompt is easy and differentiating by input is work | Medium | R-DF-020 states the differentiator; R-DF-021/022 state the denials that make it structural rather than stylistic |
| **The deletion in §5.7 does not happen** and the loop becomes the third pipeline, strictly heavier than both | Medium | It is a requirement, not an aspiration. §5.6 records the honest accounting so the claim stays falsifiable |
| **`RAISE EXCEPTION` guard fires** because rows exist against expectation | Low | That is the guard working. Goose runs in a transaction; the migration aborts and the container crashes loudly |

---

## 11. Success criteria

- [ ] One `[leaf]` node from doc 0003 is taken frontier → merged PR entirely through the loop, and every state transition in the database carries typed evidence.
- [ ] One `[decision]` node (AG-01 or AG-02) closes through the decision lane with human sign-off, producing no production code and no Gherkin.
- [ ] Judge-SPEC catches a deliberately omitted scenario that Judge-DIFF passes — the omission-blindness test, run on a seeded PR.
- [ ] A force-push during review strands the prior verdicts with no invalidation code executed anywhere.
- [ ] Re-running ingest over an unchanged document produces byte-identical (P) rows and touches no (S) row.
- [ ] `validate_taskgraph.py`, the module test command and the module lint command run on every PR in CI, and a seeded violation of each fails the build.

---

## Method sources

Matt Pocock, *Skills for Real Engineers* — `to-tickets`, `implement`, `tdd`, `code-review`, `grilling`, `codebase-design`, `writing-for-agents`, and the failure reports in `docs/engineering/*.md`.[^pocock] · Anthropic Agent Skills specification.[^skillspec] · Claude Code skills reference.[^ccskills] · Kent Beck, Canon TDD test lists. · Michael Feathers, seams. · John Ousterhout, deep modules (depth as leverage, not as a line ratio). · Alistair Cockburn, walking skeleton. · This repository: ADR 0004–0007, `openspec/AGENTS.md`, `openspec/config.yaml`, `.claude/skills/task-graph-milestone-doc/references/method.md`.

[^pocock]: https://github.com/mattpocock/skills — read at tree `84fdeff`, plugin 1.2.3, MIT.
[^skillspec]: https://agentskills.io/specification and https://platform.claude.com/docs/en/agents-and-tools/agent-skills/overview
[^ccskills]: https://code.claude.com/docs/en/skills
[^fsck]: Jesse Vincent, *Claude Code skills not triggering? It might not see them.*, 17 Dec 2025 — https://blog.fsck.com/2025/12/17/claude-code-skills-not-triggering/
[^eberhardt]: Colin Eberhardt, *Putting Spec Kit Through Its Paces: Radical Idea or Reinvented Waterfall?*, Scott Logic, 26 Nov 2025 — https://blog.scottlogic.com/2025/11/26/putting-spec-kit-through-its-paces-radical-idea-or-reinvented-waterfall.html
[^ranthe]: Itzhak Eretz Kdosha, *I Tested Three Spec-Driven AI Tools*, 13 Apr 2026 — https://ranthebuilder.cloud/blog/i-tested-three-spec-driven-ai-tools-here-s-my-honest-take/
[^tickets]: `skills/engineering/to-tickets/SKILL.md`, `<vertical-slice-rules>` block.
